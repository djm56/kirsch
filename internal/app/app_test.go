package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// collector stands in for the Bubble Tea program: App only ever reaches the TUI
// through Send, so intercepting that is enough to observe everything it emits.
type collector struct {
	mu   sync.Mutex
	msgs []any
	done chan struct{}
	want int
}

func newCollector(want int) *collector {
	return &collector{done: make(chan struct{}), want: want}
}

func (c *collector) send(msg any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
	if len(c.msgs) == c.want {
		close(c.done)
	}
}

func (c *collector) wait(t *testing.T, d time.Duration) []any {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(d):
		t.Fatalf("timed out waiting for %d messages; got %d", c.want, len(c.msgs))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]any(nil), c.msgs...)
}

func testApp(t *testing.T, fixture string, c *collector) *App {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	a.sendFn = c.send
	return a
}

func TestRunToolReportsStartAndCompletion(t *testing.T) {
	c := newCollector(2)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	a.RunTool("read_file", map[string]any{"path": "main.go"})
	msgs := c.wait(t, 5*time.Second)

	started, ok := msgs[0].(tui.ToolStartedMsg)
	if !ok {
		t.Fatalf("first message is %T, want ToolStartedMsg", msgs[0])
	}
	if started.Name != "read_file" || started.Target != "main.go" {
		t.Errorf("started = %+v", started)
	}

	done, ok := msgs[1].(tui.ToolCompletedMsg)
	if !ok {
		t.Fatalf("second message is %T, want ToolCompletedMsg", msgs[1])
	}
	if done.ID != started.ID {
		t.Errorf("completion id %d does not match start id %d", done.ID, started.ID)
	}
	if !done.OK {
		t.Fatalf("read failed: %s %s", done.ErrorKind, done.ErrorMsg)
	}
	if !strings.Contains(done.Content, "package main") {
		t.Errorf("content missing:\n%s", done.Content)
	}
}

// TestViolationReachesTheTUIAsAToolError: a denied path is information the
// model can act on, so it arrives as a failed tool result rather than as an
// ErrorMsg. architecture.md §8.
func TestViolationReachesTheTUIAsAToolError(t *testing.T) {
	c := newCollector(2)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	a.RunTool("read_file", map[string]any{"path": ".env"})
	msgs := c.wait(t, 5*time.Second)

	done := msgs[1].(tui.ToolCompletedMsg)
	if done.OK {
		t.Fatal(".env was read")
	}
	if done.ErrorKind != string(tool.KindWorkspaceViolation) {
		t.Errorf("kind = %q, want workspace_violation", done.ErrorKind)
	}
	for _, m := range msgs {
		if _, isErr := m.(tui.ErrorMsg); isErr {
			t.Error("a tool error was escalated to an infrastructure ErrorMsg")
		}
	}
}

// slowTool blocks until its context is cancelled, so cancellation can be tested
// without depending on a real tool being slow.
type slowTool struct{ started chan struct{} }

func (s *slowTool) Name() string            { return "slow" }
func (s *slowTool) Description() string     { return "blocks until cancelled" }
func (s *slowTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (s *slowTool) Invoke(ctx context.Context, _ json.RawMessage) tool.Result {
	close(s.started)
	<-ctx.Done()
	return tool.Fail(tool.KindCancelled, "cancelled mid-tool")
}

// TestCancelReturnsWithinOneSecond is ui-spec §11's cancellation target,
// asserted rather than eyeballed. Wired now with no model call to cancel,
// because cancellation retrofitted later is cancellation that does not work.
func TestCancelReturnsWithinOneSecond(t *testing.T) {
	c := newCollector(2)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	slow := &slowTool{started: make(chan struct{})}
	a.reg.Register(slow)

	a.RunTool("slow", map[string]any{})
	select {
	case <-slow.started:
	case <-time.After(2 * time.Second):
		t.Fatal("tool never started")
	}

	begin := time.Now()
	a.CancelTurn()
	msgs := c.wait(t, 2*time.Second)
	elapsed := time.Since(begin)

	if elapsed > time.Second {
		t.Errorf("cancellation took %v, target is under 1s", elapsed)
	}
	done := msgs[1].(tui.ToolCompletedMsg)
	if done.OK {
		t.Fatal("cancelled tool reported success")
	}
	if done.ErrorKind != string(tool.KindCancelled) {
		t.Errorf("kind = %q, want cancelled", done.ErrorKind)
	}
}

// TestBeginTurnCancelsThePrevious — a new turn must not leave the old one
// running behind it.
func TestBeginTurnCancelsThePrevious(t *testing.T) {
	c := newCollector(4)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	first := &slowTool{started: make(chan struct{})}
	a.reg.Register(first)
	a.RunTool("slow", map[string]any{})
	<-first.started

	a.RunTool("read_file", map[string]any{"path": "main.go"})
	msgs := c.wait(t, 3*time.Second)

	var cancelled bool
	for _, m := range msgs {
		if d, ok := m.(tui.ToolCompletedMsg); ok && d.ErrorKind == string(tool.KindCancelled) {
			cancelled = true
		}
	}
	if !cancelled {
		t.Error("starting a second turn did not cancel the first")
	}
}

// TestNoGoroutineLeak: the app is the first real concurrency in the codebase,
// so a leak here is the kind that survives to production.
func TestNoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	for i := 0; i < 20; i++ {
		c := newCollector(2)
		a := testApp(t, "repo-small", c)
		a.RunTool("list_files", map[string]any{"path": "."})
		c.wait(t, 5*time.Second)
		a.Close()
	}

	// Goroutines from the runtime and the test framework settle asynchronously.
	deadline := time.Now().Add(2 * time.Second)
	var after int
	for time.Now().Before(deadline) {
		runtime.GC()
		after = runtime.NumGoroutine()
		if after <= before+2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("goroutines grew from %d to %d across 20 app lifecycles", before, after)
}

func TestWorkspaceInfo(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-go-module", c)
	defer a.Close()

	info := a.WorkspaceInfo()
	if info.Project != "repo-go-module" {
		t.Errorf("project = %q", info.Project)
	}
	found := false
	for _, ty := range info.Types {
		if ty == "go" {
			found = true
		}
	}
	if !found {
		t.Errorf("types = %v, want go", info.Types)
	}
}

func TestUnknownToolIsReportedNotFatal(t *testing.T) {
	c := newCollector(2)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	a.RunTool("no_such_tool", map[string]any{})
	msgs := c.wait(t, 5*time.Second)
	done := msgs[1].(tui.ToolCompletedMsg)
	if done.OK {
		t.Fatal("unknown tool reported success")
	}
	if done.ErrorKind != string(tool.KindToolInputInvalid) {
		t.Errorf("kind = %q", done.ErrorKind)
	}
}
