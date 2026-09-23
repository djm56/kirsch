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

// TestApprovalRequestBlocksUntilResolved: Request must block in the tool
// goroutine until Resolve is called. The approval uses a capacity-1 buffered
// channel so Resolve never blocks, and the tool waits on the receive end.
func TestApprovalRequestBlocksUntilResolved(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description:          "test approval",
		Kind:                 "test",
		CanApproveForSession: false,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome := a.Request(ctx, req)
		resultCh <- outcome
	}()

	// Give the goroutine time to reach the blocking select.
	time.Sleep(100 * time.Millisecond)

	// Should not have a result yet.
	select {
	case <-resultCh:
		t.Fatal("Request returned before Resolve was called")
	default:
	}

	// Now resolve and check the result. ID 1 is correct because this is the
	// first approval generated by the app.
	a.Resolve(1, ApprovalOutcomeOnce)
	select {
	case outcome := <-resultCh:
		if outcome != ApprovalOutcomeOnce {
			t.Errorf("outcome = %v, want ApprovalOutcomeOnce", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("Request did not return after Resolve")
	}
}

// TestApprovalResolveNeverBlocks: Resolve must return immediately without
// blocking, even if nothing is waiting.
func TestApprovalResolveNeverBlocks(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	done := make(chan struct{})
	go func() {
		a.Resolve(999, ApprovalOutcomeOnce)
		close(done)
	}()

	select {
	case <-done:
		// Expected: Resolve returned immediately
	case <-time.After(time.Second):
		t.Fatal("Resolve blocked for over 1 second")
	}
}

// TestApprovalDuplicateSendIsNeverReached: this is the old test, which doesn't
// actually exercise the duplicate-send branch. It resolves twice AFTER the
// approval is deleted from the map, so the second call returns early and
// never reaches the default clause. Kept for regression, but see
// TestApprovalDuplicateResolveBeforeRequest for the real test.
func TestApprovalResolvingTwiceIsHarmless(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description:          "test approval",
		Kind:                 "test",
		CanApproveForSession: false,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome := a.Request(ctx, req)
		resultCh <- outcome
	}()

	time.Sleep(100 * time.Millisecond)

	// First resolve.
	a.Resolve(1, ApprovalOutcomeOnce)
	select {
	case <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("Request did not return after first Resolve")
	}

	// Second resolve must not block or panic.
	done := make(chan struct{})
	go func() {
		a.Resolve(1, ApprovalOutcomeDeny)
		close(done)
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		t.Fatal("second Resolve blocked")
	}
}

// TestApprovalDuplicateResolveBeforeRequest: the real test for the duplicate-send
// branch. An approval with a buffered channel has a value already sent to it.
// A second resolve must reach the default clause and drop silently without blocking.
//
// This test verifies the second rule of the approval contract: Resolve sends on
// a capacity-1 buffered channel and the default clause catches duplicate sends,
// ensuring Resolve never blocks.
//
// To verify this test actually reaches the default clause, remove the default:
// clause from Resolve and confirm this test fails (would block on the second send).
func TestApprovalDuplicateResolveBeforeRequest(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	// Manually create an approval in the map with a value already in the channel.
	// This simulates the state where Resolve has been called once and the next
	// Resolve must hit the default clause to avoid blocking.
	manualApprovalID := int64(999)
	manualApproval := &approval{
		id:       manualApprovalID,
		decision: make(chan ApprovalOutcome, 1),
		argv:     nil,
	}
	// Pre-fill the channel with a value
	manualApproval.decision <- ApprovalOutcomeOnce

	a.approveMu.Lock()
	a.approvals[manualApprovalID] = manualApproval
	a.approveMu.Unlock()

	// Now Resolve again on the same approval. The channel is full, so without
	// the default clause this would block trying to send.
	done := make(chan struct{})
	go func() {
		a.Resolve(manualApprovalID, ApprovalOutcomeDeny)
		close(done)
	}()

	select {
	case <-done:
		// Expected: second resolve hit the default clause and returned
	case <-time.After(time.Second):
		t.Fatal("second Resolve blocked (default clause not reached)")
	}

	// Verify the channel still has the first value
	select {
	case outcome := <-manualApproval.decision:
		if outcome != ApprovalOutcomeOnce {
			t.Errorf("channel value = %v, want ApprovalOutcomeOnce", outcome)
		}
	default:
		t.Fatal("channel is empty, first value was lost")
	}
}

// TestApprovalCancellationReleasesRequest: cancelling the context while a
// Request is pending must unblock it and return ApprovalOutcomeCancelled.
// This exercises the context cancellation path.
func TestApprovalCancellationReleasesRequest(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := ApprovalRequest{
		Description:          "test approval",
		Kind:                 "test",
		CanApproveForSession: false,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome := a.Request(ctx, req)
		resultCh <- outcome
	}()

	time.Sleep(100 * time.Millisecond)

	// Cancel the context.
	cancel()

	select {
	case outcome := <-resultCh:
		if outcome != ApprovalOutcomeCancelled {
			t.Errorf("outcome = %v, want ApprovalOutcomeCancelled", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not release Request")
	}
}

// TestApprovalMessageSent: when Request is called, it sends an
// ApprovalRequestedMsg to the TUI.
func TestApprovalMessageSent(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description:          "apply patch to main.go",
		Kind:                 "patch",
		CanApproveForSession: true,
	}

	go func() {
		// Resolve immediately in a goroutine so Request returns.
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg, ok := msgs[0].(tui.ApprovalRequestedMsg)
	if !ok {
		t.Fatalf("message is %T, want ApprovalRequestedMsg", msgs[0])
	}

	if msg.Description != "apply patch to main.go" {
		t.Errorf("description = %q", msg.Description)
	}
	if msg.Kind != "patch" {
		t.Errorf("kind = %q", msg.Kind)
	}
	if !msg.CanApproveForSession {
		t.Error("CanApproveForSession is false, want true")
	}
	if msg.ID != 1 {
		t.Errorf("ID = %d, want 1", msg.ID)
	}
}

// TestApprovalNoGoroutineLeakOnFullCycle: requesting and resolving an approval
// must not leak goroutines. Creates multiple cycles and verifies cleanup.
func TestApprovalNoGoroutineLeakOnFullCycle(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	before := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		ctx := context.Background()
		req := ApprovalRequest{
			Description:          "test",
			Kind:                 "test",
			CanApproveForSession: false,
		}

		resultCh := make(chan ApprovalOutcome)
		go func() {
			outcome := a.Request(ctx, req)
			resultCh <- outcome
		}()

		time.Sleep(50 * time.Millisecond)
		// The approval ID increments each time, so it's i+1, i+2, etc.
		a.Resolve(int64(i+1), ApprovalOutcomeOnce)

		select {
		case <-resultCh:
		case <-time.After(time.Second):
			t.Fatal("Request did not complete")
		}
	}

	// Allow time for goroutines to settle.
	deadline := time.Now().Add(1 * time.Second)
	var after int
	for time.Now().Before(deadline) {
		runtime.GC()
		after = runtime.NumGoroutine()
		if after <= before+1 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("goroutines grew from %d to %d across 5 approval cycles", before, after)
}

// TestApprovalConcurrentRequests: two concurrent approvals with distinct IDs
// must both complete independently. This verifies the approval map tracks them
// separately.
func TestApprovalConcurrentRequests(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	// Manually create two approvals to test concurrent resolution
	id1 := int64(1000)
	id2 := int64(1001)

	a.approveMu.Lock()
	a.approvals[id1] = &approval{
		id:       id1,
		decision: make(chan ApprovalOutcome, 1),
		argv:     nil,
	}
	a.approvals[id2] = &approval{
		id:       id2,
		decision: make(chan ApprovalOutcome, 1),
		argv:     nil,
	}
	a.approveMu.Unlock()

	// Resolve both approvals with different outcomes
	a.Resolve(id1, ApprovalOutcomeOnce)
	a.Resolve(id2, ApprovalOutcomeDeny)

	// Read both outcomes
	select {
	case outcome := <-a.approvals[id1].decision:
		if outcome != ApprovalOutcomeOnce {
			t.Errorf("outcome1 = %v, want ApprovalOutcomeOnce", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("approval 1 did not receive outcome")
	}

	select {
	case outcome := <-a.approvals[id2].decision:
		if outcome != ApprovalOutcomeDeny {
			t.Errorf("outcome2 = %v, want ApprovalOutcomeDeny", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("approval 2 did not receive outcome")
	}
}

// TestApprovalRootContextCancellation: cancelling the root context while an
// approval is pending must unblock it and return ApprovalOutcomeCancelled.
// This exercises the third branch of the select in Request.
func TestApprovalRootContextCancellation(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)

	// Don't defer a.Close() — we'll call it manually to test root cancellation.

	ctx := context.Background()
	req := ApprovalRequest{
		Description:          "test",
		Kind:                 "test",
		CanApproveForSession: false,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome := a.Request(ctx, req)
		resultCh <- outcome
	}()

	time.Sleep(100 * time.Millisecond)

	// Cancel the root context (simulating shutdown).
	a.Close()

	select {
	case outcome := <-resultCh:
		if outcome != ApprovalOutcomeCancelled {
			t.Errorf("outcome = %v, want ApprovalOutcomeCancelled", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("root cancellation did not release Request")
	}
}
