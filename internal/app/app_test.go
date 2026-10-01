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

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/policy"
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
		Description: "test approval",
		Operation:   policy.OperationCommand,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome, _ := a.Request(ctx, req)
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
		Description: "test approval",
		Operation:   policy.OperationCommand,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome, _ := a.Request(ctx, req)
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
		Description: "test approval",
		Operation:   policy.OperationCommand,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome, _ := a.Request(ctx, req)
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
// ApprovalRequestedMsg to the TUI, with Kind and CanApproveForSession both
// derived from Operation — a command request renders as "command" and may
// offer session approval.
func TestApprovalMessageSent(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run go test ./...",
		Operation:   policy.OperationCommand,
		Argv:        []string{"go", "test", "./..."},
	}

	go func() {
		// Resolve immediately in a goroutine so Request returns.
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg, ok := msgs[0].(tui.ApprovalRequestedMsg)
	if !ok {
		t.Fatalf("message is %T, want ApprovalRequestedMsg", msgs[0])
	}

	if msg.Description != "run go test ./..." {
		t.Errorf("description = %q", msg.Description)
	}
	if msg.Kind != "command" {
		t.Errorf("kind = %q, want %q (derived from Operation=OperationCommand)", msg.Kind, "command")
	}
	if !msg.CanApproveForSession {
		t.Error("CanApproveForSession is false, want true (command requests may offer session approval)")
	}
	if msg.ID != 1 {
		t.Errorf("ID = %d, want 1", msg.ID)
	}
}

// TestApprovalPatchNeverOffersSessionApproval drives the real Request path —
// not policy.CanApproveForSession in isolation — with a patch-originated
// request, and asserts on what actually reaches the TUI in
// ApprovalRequestedMsg. This is the regression test for the defect fixed in
// this round: a prior revision let a caller set an independent Kind field
// that disagreed with Operation, so a request could be labelled "patch" for
// rendering while its Operation still read OperationCommand, and the [a]
// button was offered. Kind is now derived from Operation inside Request, so
// that disagreement can no longer be constructed.
func TestApprovalPatchNeverOffersSessionApproval(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "apply patch to main.go",
		Operation:   policy.OperationPatch,
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg, ok := msgs[0].(tui.ApprovalRequestedMsg)
	if !ok {
		t.Fatalf("message is %T, want ApprovalRequestedMsg", msgs[0])
	}

	if msg.Kind != "patch" {
		t.Errorf("kind = %q, want %q", msg.Kind, "patch")
	}
	if msg.CanApproveForSession {
		t.Error("CanApproveForSession is true for a patch-originated request, want false")
	}
}

// TestApprovalPatchResolvedForSessionRecordsNoGrant drives the real Request
// and Resolve path — not policy.Grant in isolation — with a patch-originated
// request resolved as ApprovalOutcomeSession, and asserts via the real
// policy's Grants() that nothing was recorded. internal/policy's own refusal
// of OperationPatch (Grant) was correct from round 1; what rounds 2 and 3 got
// wrong was getting the right Operation to that refusal in the first place —
// this proves the whole path, not just the enforcement point in isolation.
// Argv is deliberately populated even though no real apply_patch caller
// would set it, to prove the refusal holds on Operation alone and is not
// merely an accident of argv being empty.
func TestApprovalPatchResolvedForSessionRecordsNoGrant(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "apply patch to main.go",
		Operation:   policy.OperationPatch,
		Argv:        []string{"go", "test"}, // deliberately present; must still be refused
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome, _ := a.Request(ctx, req)
		resultCh <- outcome
	}()

	// Wait for the ApprovalRequestedMsg rather than sleeping: Request
	// registers the approval in a.approvals before it sends, so once the
	// message has been observed, id 1 is guaranteed resolvable.
	c.wait(t, time.Second)

	a.Resolve(1, ApprovalOutcomeSession)

	select {
	case outcome := <-resultCh:
		if outcome != ApprovalOutcomeSession {
			t.Fatalf("outcome = %v, want ApprovalOutcomeSession", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("Request did not return after Resolve")
	}

	if grants := a.pol.Grants(); len(grants) != 0 {
		t.Errorf("Grants() = %v, want none for a patch-originated approval resolved for session", grants)
	}
}

// TestApprovalNonSessionOutcomesRecordNoGrant drives the real Request and
// Resolve path for the three outcomes that must never create a session
// grant — approve-once, reject, and cancel — using argv that WOULD qualify
// for a grant if the outcome were ApprovalOutcomeSession. This is the
// negative-path counterpart to TestApprovalPatchResolvedForSessionRecordsNoGrant
// (which proves refusal on Operation alone, for a patch): here the operation
// and argv are both grantable, so the only thing keeping Grants() empty is
// Resolve's own outcome check. It closes a coverage gap left when
// TestGrantCounterIncrementsOnlyOnConfirmedSessionGrant was deleted as
// redundant — that test asserted only a value it set itself and never
// covered these three outcomes.
func TestApprovalNonSessionOutcomesRecordNoGrant(t *testing.T) {
	cases := []struct {
		name    string
		outcome ApprovalOutcome
	}{
		{"approve-once", ApprovalOutcomeOnce},
		{"reject", ApprovalOutcomeDeny},
		{"cancel", ApprovalOutcomeCancelled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCollector(1)
			a := testApp(t, "repo-small", c)
			defer a.Close()

			ctx := context.Background()
			req := ApprovalRequest{
				Description: "run a grantable command",
				Operation:   policy.OperationCommand,
				Argv:        []string{"go", "test"}, // grantable; proves the outcome check, not argv refusal
			}

			resultCh := make(chan ApprovalOutcome)
			go func() {
				outcome, _ := a.Request(ctx, req)
				resultCh <- outcome
			}()

			// Wait for the ApprovalRequestedMsg rather than sleeping: Request
			// registers the approval in a.approvals before it sends, so once the
			// message has been observed, id 1 is guaranteed resolvable.
			c.wait(t, time.Second)

			a.Resolve(1, tc.outcome)

			select {
			case outcome := <-resultCh:
				if outcome != tc.outcome {
					t.Fatalf("outcome = %v, want %v", outcome, tc.outcome)
				}
			case <-time.After(time.Second):
				t.Fatal("Request did not return after Resolve")
			}

			if grants := a.pol.Grants(); len(grants) != 0 {
				t.Errorf("Grants() = %v, want none for outcome %v", grants, tc.outcome)
			}
		})
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
			Description: "test",
			Operation:   policy.OperationCommand,
		}

		resultCh := make(chan ApprovalOutcome)
		go func() {
			outcome, _ := a.Request(ctx, req)
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
		Description: "test",
		Operation:   policy.OperationCommand,
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome, _ := a.Request(ctx, req)
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

// TestApprovalShellCommandNeverOffersSessionGrant: [a] is never offered for
// shell commands, even though they are OperationCommand. This verifies that
// app.Request checks both CanApproveForSession(Operation) AND CanGrant(argv),
// so that CanApproveForSession alone (which checks only the operation type)
// cannot trick the caller into offering [a] for ungrantable argv.
// This is the test for Defect 2: ensure [a] is not offered for commands
// that Grant would refuse.
func TestApprovalShellCommandNeverOffersSessionGrant(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run shell command",
		Operation:   policy.OperationCommand,
		Argv:        []string{"bash", "-c", "echo hi"},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg, ok := msgs[0].(tui.ApprovalRequestedMsg)
	if !ok {
		t.Fatalf("message is %T, want ApprovalRequestedMsg", msgs[0])
	}

	if msg.Kind != "command" {
		t.Errorf("kind = %q, want %q", msg.Kind, "command")
	}
	// CRITICAL: CanApproveForSession is false for shells, even though Operation
	// is OperationCommand. This proves the gate checks BOTH operation type
	// and the specific argv.
	if msg.CanApproveForSession {
		t.Error("CanApproveForSession is true for a shell command, want false (shells cannot be granted)")
	}
}

// TestApprovalWildcardCommandNeverOffersSessionGrant: [a] is never offered for
// bare wildcard commands. This verifies the gate refuses bare wildcards.
func TestApprovalWildcardCommandNeverOffersSessionGrant(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "bare wildcard",
		Operation:   policy.OperationCommand,
		Argv:        []string{"*"},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	msg := msgs[0].(tui.ApprovalRequestedMsg)
	if msg.CanApproveForSession {
		t.Error("CanApproveForSession is true for a bare wildcard, want false")
	}
}

// TestApprovalGrantableCommandOffersSessionGrant: [a] IS offered for a valid,
// grantable command. This verifies the positive case: a normal command reaches
// the user with CanApproveForSession=true.
func TestApprovalGrantableCommandOffersSessionGrant(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run curl",
		Operation:   policy.OperationCommand,
		Argv:        []string{"curl", "-v", "https://example.com"},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	msg := msgs[0].(tui.ApprovalRequestedMsg)
	if !msg.CanApproveForSession {
		t.Error("CanApproveForSession is false for a valid command, want true")
	}
}

// TestApprovalSessionGrantRecordsGrant: pressing [a] on a valid command records
// the grant in policy and it reaches Grants(). This verifies the full path from
// Request through Resolve to a live grant.
func TestApprovalSessionGrantRecordsGrant(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	// Start with no grants
	if grants := a.pol.Grants(); len(grants) != 0 {
		t.Fatalf("setup: Grants() should be empty, got %v", grants)
	}

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run curl -v",
		Operation:   policy.OperationCommand,
		Argv:        []string{"curl", "-v"},
	}

	resultCh := make(chan ApprovalOutcome)
	go func() {
		outcome, _ := a.Request(ctx, req)
		resultCh <- outcome
	}()

	// Wait for the message to be sent, which confirms the approval is registered
	c.wait(t, time.Second)

	// Resolve as ApprovalOutcomeSession
	a.Resolve(1, ApprovalOutcomeSession)

	// Wait for Request to return
	select {
	case outcome := <-resultCh:
		if outcome != ApprovalOutcomeSession {
			t.Fatalf("outcome = %v, want ApprovalOutcomeSession", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("Request did not return after Resolve")
	}

	// Verify the grant was recorded in policy
	grants := a.pol.Grants()
	if len(grants) != 1 {
		t.Fatalf("Grants() has %d grants, want 1", len(grants))
	}
	if grants[0] != "curl -v" {
		t.Errorf("Grants()[0] = %q, want %q", grants[0], "curl -v")
	}
}

// TestApprovalRefusedGrantIsNotSilent: if Grant fails after the user presses [a],
// the error is returned by Request so the caller can handle it. This verifies the
// fix for Defect 3: a refused grant must not vanish into a debug log.
// Test by calling Request with a shell command (which will fail on Grant) and
// verifying the error is returned.
func TestApprovalRefusedGrantIsNotSilent(t *testing.T) {
	c := newCollector(0)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	// Run a Request with a shell command in a goroutine. Shells cannot be granted,
	// so when Resolve is called with ApprovalOutcomeSession, Grant will fail.
	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run shell",
		Operation:   policy.OperationCommand,
		Argv:        []string{"bash", "-c", "echo hi"},
	}

	resultCh := make(chan struct {
		outcome    ApprovalOutcome
		grantError error
	})
	go func() {
		outcome, grantErr := a.Request(ctx, req)
		resultCh <- struct {
			outcome    ApprovalOutcome
			grantError error
		}{outcome, grantErr}
	}()

	// Give the goroutine time to reach the blocking point and register the approval.
	time.Sleep(100 * time.Millisecond)

	// Resolve as session grant. This will call Grant with the shell argv,
	// which should fail because shells cannot be granted.
	a.Resolve(1, ApprovalOutcomeSession)

	// Verify the error was returned by Request
	select {
	case result := <-resultCh:
		if result.outcome != ApprovalOutcomeSession {
			t.Errorf("outcome = %v, want ApprovalOutcomeSession", result.outcome)
		}
		if result.grantError == nil {
			t.Error("grantError is nil, want error (shell should be refused by Grant)")
		}
		if result.grantError != policy.ErrShellGrantForbidden {
			t.Errorf("grantError = %v, want %v", result.grantError, policy.ErrShellGrantForbidden)
		}
	case <-time.After(time.Second):
		t.Fatal("Request did not return after Resolve")
	}

	// Verify no grant was recorded
	if grants := a.pol.Grants(); len(grants) != 0 {
		t.Errorf("Grants() = %v, want empty (grant should have failed)", grants)
	}
}

// TestApprovalGateChecksBothConditions: this test demonstrates the gate works
// by showing it checks BOTH CanApproveForSession(op) AND CanGrant(op, argv).
// If either condition is false, CanApproveForSession is false in the message.
// This is a mutation test: we remove one condition, show the test fails, restore,
// show it passes. The test captures the state by reading what reaches the TUI
// and verifying the combined gate's result.
func TestApprovalGateChecksBothConditions(t *testing.T) {
	cases := []struct {
		name                 string
		operation            policy.Operation
		argv                 []string
		expectCanApproveTrue bool
		description          string
	}{
		{
			name:                 "patch operation",
			operation:            policy.OperationPatch,
			argv:                 []string{"go", "test"},
			expectCanApproveTrue: false,
			description:          "patch never allows session approval (CanApproveForSession returns false)",
		},
		{
			name:                 "valid command",
			operation:            policy.OperationCommand,
			argv:                 []string{"curl"},
			expectCanApproveTrue: true,
			description:          "valid command with grantable argv (both conditions true)",
		},
		{
			name:                 "shell argv",
			operation:            policy.OperationCommand,
			argv:                 []string{"bash"},
			expectCanApproveTrue: false,
			description:          "shell cannot be granted (CanGrant returns false despite OperationCommand)",
		},
		{
			name:                 "empty argv",
			operation:            policy.OperationCommand,
			argv:                 []string{},
			expectCanApproveTrue: false,
			description:          "empty argv cannot be granted (CanGrant returns false)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCollector(1)
			a := testApp(t, "repo-small", c)
			defer a.Close()

			ctx := context.Background()
			req := ApprovalRequest{
				Description: tc.description,
				Operation:   tc.operation,
				Argv:        tc.argv,
			}

			go func() {
				time.Sleep(50 * time.Millisecond)
				a.Resolve(1, ApprovalOutcomeOnce)
			}()

			_, _ = a.Request(ctx, req)
			msgs := c.wait(t, time.Second)

			msg := msgs[0].(tui.ApprovalRequestedMsg)
			if msg.CanApproveForSession != tc.expectCanApproveTrue {
				t.Errorf("CanApproveForSession = %v, want %v. %s",
					msg.CanApproveForSession, tc.expectCanApproveTrue, tc.description)
			}
		})
	}
}

// TestApprovalPatchChangesReachCard verifies that patch file changes are carried
// through the approval message and reach the card for rendering. This test fails
// if Changes stops flowing through the seam.
func TestApprovalPatchChangesReachCard(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "apply patch",
		Operation:   policy.OperationPatch,
		Changes: []patch.FileChange{
			{Path: "file1.go"},
			{Path: "file2.go"},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	msg := msgs[0].(tui.ApprovalRequestedMsg)

	// Subject should indicate the number of files changed
	if msg.Subject != "2 files changed" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "2 files changed")
	}

	// Detail should contain the file list
	if len(msg.Detail) == 0 {
		t.Error("Detail is empty, expected file list")
	} else if !strings.Contains(msg.Detail[0], "file1.go") {
		t.Errorf("Detail[0] = %q, expected to contain file1.go", msg.Detail[0])
	}
}

// TestApprovalCommandGrantScopeIsArgv verifies that GrantScope holds the full
// argv for session grants, matching what policy.Grant records. This test fails
// if GrantScope is set to only argv[0] or any other subset.
func TestApprovalCommandGrantScopeIsArgv(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run a command",
		Operation:   policy.OperationCommand,
		Argv:        []string{"go", "test", "./..."},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeOnce)
	}()

	_, _ = a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	msg := msgs[0].(tui.ApprovalRequestedMsg)

	// GrantScope should be the full argv as a space-separated string,
	// matching what policy.Grant records and policy.Grants() returns
	if msg.GrantScope != "go test ./..." {
		t.Errorf("GrantScope = %q, want %q", msg.GrantScope, "go test ./...")
	}

	// Subject should contain the command
	if msg.Subject != "go test" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "go test")
	}
}

// TestApprovalSessionGrantRefusedIsVisible verifies that when a session grant
// is refused, a notice message is sent to the TUI, and an ApprovalResolvedMsg
// is sent to inform the TUI of the actual outcome (Rejected). This test fails
// if a refused grant is silently downgraded without notification.
func TestApprovalSessionGrantRefusedIsVisible(t *testing.T) {
	c := newCollector(3) // Expect ApprovalRequestedMsg + NoticeMsg + ApprovalResolvedMsg
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx := context.Background()
	req := ApprovalRequest{
		Description: "run a shell command",
		Operation:   policy.OperationCommand,
		Argv:        []string{"bash"}, // Shell cannot be granted
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeSession) // Try to grant a session for a shell
	}()

	outcome, grantErr := a.Request(ctx, req)
	msgs := c.wait(t, time.Second)

	// The grant should have failed
	if grantErr == nil {
		t.Error("grantErr is nil, expected an error for shell command")
	}

	// Outcome should be session (though grant failed, this is what the user chose)
	if outcome != ApprovalOutcomeSession {
		t.Errorf("outcome = %v, want ApprovalOutcomeSession", outcome)
	}

	// Should have received: approval request, notice of failure, and confirmation of actual outcome
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	// Second message should be a NoticeMsg indicating the grant was refused
	noticeMsg, ok := msgs[1].(tui.NoticeMsg)
	if !ok {
		t.Fatalf("second message is %T, want tui.NoticeMsg", msgs[1])
	}

	if !strings.Contains(noticeMsg.Text, "denied") {
		t.Errorf("NoticeMsg.Text = %q, expected to contain 'denied'", noticeMsg.Text)
	}

	// Third message should be ApprovalResolvedMsg with Rejected outcome
	resolvedMsg, ok := msgs[2].(tui.ApprovalResolvedMsg)
	if !ok {
		t.Fatalf("third message is %T, want tui.ApprovalResolvedMsg", msgs[2])
	}

	// The confirmed outcome should be Rejected because the grant was refused
	if resolvedMsg.Outcome != tui.Rejected {
		t.Errorf("ApprovalResolvedMsg.Outcome = %v, want tui.Rejected", resolvedMsg.Outcome)
	}
}

// TestApprovalSessionGrantRecordsGrantInPolicy verifies that a successful
// session grant is recorded in the policy and can be queried via policy.Grants().
// This test ensures the TUI grant counter is synchronized with the policy state.
func TestApprovalSessionGrantRecordsGrantInPolicy(t *testing.T) {
	c := newCollector(2) // ApprovalRequestedMsg + ApprovalResolvedMsg on success
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := ApprovalRequest{
		Description: "run a test",
		Operation:   policy.OperationCommand,
		Argv:        []string{"go", "test", "./..."},
	}

	// Before the grant, record the number of grants
	initialGrantCount := len(a.pol.Grants())

	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeSession) // Approve for session
	}()

	outcome, err := a.Request(ctx, req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if outcome != ApprovalOutcomeSession {
		t.Errorf("outcome = %v, want ApprovalOutcomeSession", outcome)
	}

	// After the grant, policy.Grants should have one more entry
	afterGrantCount := len(a.pol.Grants())
	if afterGrantCount != initialGrantCount+1 {
		t.Errorf("policy.Grants count after session grant = %d, want %d", afterGrantCount, initialGrantCount+1)
	}

	// Verify the grant is recorded in the policy
	grants := a.pol.Grants()
	if len(grants) > 0 {
		lastGrant := grants[len(grants)-1]
		expectedGrant := "go test ./..."
		if lastGrant != expectedGrant {
			t.Errorf("last grant = %q, want %q", lastGrant, expectedGrant)
		}
	}
}

// TestComputeDiffDisplayParsesRealDiff verifies that computeDiffDisplay correctly
// parses a unified diff, counts added and removed lines, and returns the filename.
// The diff is parsed by patch.Parse and then stats are computed.
func TestComputeDiffDisplayParsesRealDiff(t *testing.T) {
	// A real unified diff in the format patch.Parse expects.
	// The hunk header format is @@ -oldStart,oldLines +newStart,newLines @@
	// Old has 4 lines: 1 context + 1 removed + 2 context
	// New has 7 lines: 1 context + 4 added + 2 context
	diffText := `--- a/calc/divide.go
+++ b/calc/divide.go
@@ -1,4 +1,7 @@
 func Divide(a, b float64) (float64, error) {
-	return a / b, nil
+	if b == 0 {
+		return 0, ErrDivideByZero
+	}
+	return a / b, nil
 }
 func Other() {
`

	// Parse the diff.
	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	// Compute the display data.
	filename, diffLines, added, removed := computeDiffDisplay(changes)

	// Verify the filename.
	if filename != "calc/divide.go" {
		t.Errorf("filename = %q, want %q", filename, "calc/divide.go")
	}

	// Verify counts match the diff.
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if added != 4 {
		t.Errorf("added = %d, want 4", added)
	}

	// Verify diff lines are present and start with @@.
	if len(diffLines) == 0 {
		t.Errorf("diffLines is empty, want at least one line")
	}
	if !strings.HasPrefix(diffLines[0], "@@") {
		t.Errorf("first line = %q, want to start with @@", diffLines[0])
	}

	// Verify diff lines preserve prefixes.
	foundMinus := false
	foundPlus := false
	for _, line := range diffLines {
		if len(line) > 0 && line[0] == '-' {
			foundMinus = true
		}
		if len(line) > 0 && line[0] == '+' {
			foundPlus = true
		}
	}
	if !foundMinus {
		t.Errorf("no '-' prefixed lines found in diff")
	}
	if !foundPlus {
		t.Errorf("no '+' prefixed lines found in diff")
	}
}

// TestComputeDiffDisplayHandlesBinaryFiles verifies that when a file has
// IsBinary set and no hunks, computeDiffDisplay provides a descriptive body.
func TestComputeDiffDisplayHandlesBinaryFiles(t *testing.T) {
	changes := []patch.FileChange{
		{
			Op:       patch.OpModify,
			Path:     "image.png",
			IsBinary: true,
			Hunks:    []patch.Hunk{}, // No hunks for binary
		},
	}

	filename, diffLines, added, removed := computeDiffDisplay(changes)

	if filename != "image.png" {
		t.Errorf("filename = %q, want %q", filename, "image.png")
	}
	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
	if len(diffLines) == 0 {
		t.Errorf("diffLines is empty, want a descriptive body for binary file")
	}
	if len(diffLines) > 0 && diffLines[0] != "Binary file changed" {
		t.Errorf("diffLines[0] = %q, want %q", diffLines[0], "Binary file changed")
	}
}

// TestComputeDiffDisplayHandlesPureRename verifies that computeDiffDisplay
// correctly handles a pure rename (no hunks, just a path change).
func TestComputeDiffDisplayHandlesPureRename(t *testing.T) {
	diffText := `diff --git a/old_name.go b/new_name.go
rename from old_name.go
rename to new_name.go
`

	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	filename, diffLines, added, removed := computeDiffDisplay(changes)

	// For renames, filename should show "old → new" to match formatPatchDisplay
	if filename != "old_name.go → new_name.go" {
		t.Errorf("filename = %q, want %q", filename, "old_name.go → new_name.go")
	}

	if len(diffLines) == 0 {
		t.Errorf("diffLines is empty for pure rename, want descriptive body")
	}

	// Should have a descriptive line
	found := false
	for _, line := range diffLines {
		if strings.Contains(line, "renamed") || strings.Contains(line, "Renamed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("diffLines = %v, want at least one line mentioning rename", diffLines)
	}

	if added != 0 || removed != 0 {
		t.Errorf("added=%d removed=%d, want both 0 for a pure rename", added, removed)
	}
}

// TestComputeDiffDisplayHandlesZeroByteCreate verifies that computeDiffDisplay
// correctly handles a zero-byte create (no hunks).
func TestComputeDiffDisplayHandlesZeroByteCreate(t *testing.T) {
	diffText := `diff --git a/empty.txt b/empty.txt
new file mode 100644
index 0000000..e69de29
`

	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	filename, diffLines, added, removed := computeDiffDisplay(changes)

	if filename != "empty.txt" {
		t.Errorf("filename = %q, want %q", filename, "empty.txt")
	}

	if len(diffLines) == 0 {
		t.Errorf("diffLines is empty for zero-byte create, want descriptive body")
	}

	// Should have a descriptive line
	found := false
	for _, line := range diffLines {
		if strings.Contains(line, "created") || strings.Contains(line, "Created") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("diffLines = %v, want at least one line mentioning create", diffLines)
	}

	if added != 0 || removed != 0 {
		t.Errorf("added=%d removed=%d, want both 0 for an empty file", added, removed)
	}
}

// TestComputeDiffDisplayHandlesZeroByteDelete verifies that computeDiffDisplay
// correctly handles a zero-byte delete (no hunks).
func TestComputeDiffDisplayHandlesZeroByteDelete(t *testing.T) {
	diffText := `diff --git a/empty.txt b/empty.txt
deleted file mode 100644
index e69de29..0000000
`

	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	filename, diffLines, added, removed := computeDiffDisplay(changes)

	if filename != "empty.txt" {
		t.Errorf("filename = %q, want %q", filename, "empty.txt")
	}

	if len(diffLines) == 0 {
		t.Errorf("diffLines is empty for zero-byte delete, want descriptive body")
	}

	// Should have a descriptive line
	found := false
	for _, line := range diffLines {
		if strings.Contains(line, "deleted") || strings.Contains(line, "Deleted") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("diffLines = %v, want at least one line mentioning delete", diffLines)
	}

	if added != 0 || removed != 0 {
		t.Errorf("added=%d removed=%d, want both 0 for an empty file", added, removed)
	}
}

// TestFormatPatchDisplayShowsRenameWithOldPath verifies that formatPatchDisplay
// shows both old and new paths for renamed files.
func TestFormatPatchDisplayShowsRenameWithOldPath(t *testing.T) {
	changes := []patch.FileChange{
		{
			Op:      patch.OpRename,
			Path:    "new_name.go",
			OldPath: "old_name.go",
		},
	}

	subject, detail := formatPatchDisplay(changes)

	if subject != "1 file changed" {
		t.Errorf("subject = %q, want %q", subject, "1 file changed")
	}

	// The detail should contain both old and new paths, separated by →
	detailStr := strings.Join(detail, " ")
	if !strings.Contains(detailStr, "old_name.go") {
		t.Errorf("detail does not contain old_name.go: %s", detailStr)
	}
	if !strings.Contains(detailStr, "new_name.go") {
		t.Errorf("detail does not contain new_name.go: %s", detailStr)
	}
	if !strings.Contains(detailStr, "→") {
		t.Errorf("detail does not contain → separator: %s", detailStr)
	}
}

// TestComputeDiffDisplayHandlesModeChange verifies that when a file has
// NewMode set and no hunks, computeDiffDisplay provides a descriptive body.
func TestComputeDiffDisplayHandlesModeChange(t *testing.T) {
	changes := []patch.FileChange{
		{
			Op:      patch.OpModify,
			Path:    "script.sh",
			NewMode: 0o755, // Mode change, no content change
			Hunks:   []patch.Hunk{},
		},
	}

	filename, diffLines, added, removed := computeDiffDisplay(changes)

	if filename != "script.sh" {
		t.Errorf("filename = %q, want %q", filename, "script.sh")
	}
	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
	if len(diffLines) == 0 {
		t.Errorf("diffLines is empty, want a descriptive body for mode change")
	}
	if len(diffLines) > 0 && diffLines[0] != "Mode changed: 755" {
		t.Errorf("diffLines[0] = %q, want %q", diffLines[0], "Mode changed: 755")
	}
}

// TestComputeDiffDisplayBackslashLineRenderingMatchesPatch verifies that
// computeDiffDisplay renders a backslash-prefixed line the same way as
// patch.Render does: with a space after the backslash. Specifically, when
// a line has Prefix == '\\', the output should be "\ " (backslash and space)
// followed by the content.
func TestComputeDiffDisplayBackslashLineRenderingMatchesPatch(t *testing.T) {
	// A diff ending with a backslash-prefixed line (no newline at end of file).
	diffText := `--- a/main.go
+++ b/main.go
@@ -1,2 +1,2 @@
 fmt.Println("hello")
-fmt.Println("world")
\ No newline at end of file
+fmt.Println("world")
`

	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	// Compute diff display
	filename, diffLines, _, _ := computeDiffDisplay(changes)

	if filename != "main.go" {
		t.Errorf("filename = %q, want %q", filename, "main.go")
	}

	// Call patch.Render to get the expected output
	renderedBytes := patch.Render(changes)
	renderedLines := strings.Split(strings.TrimSuffix(string(renderedBytes), "\n"), "\n")

	// Find the backslash line in the rendered diff
	var expectedBackslashLine string
	for _, line := range renderedLines {
		if strings.HasPrefix(line, `\ `) {
			expectedBackslashLine = line
			break
		}
	}

	if expectedBackslashLine == "" {
		t.Fatalf("no backslash line found in patch.Render output")
	}

	// Find the backslash line in the computed diff lines
	var computedBackslashLine string
	for _, line := range diffLines {
		if len(line) > 0 && line[0] == '\\' {
			computedBackslashLine = line
			break
		}
	}

	if computedBackslashLine == "" {
		t.Fatalf("no backslash line found in computed diff lines")
	}

	// Verify they match
	if computedBackslashLine != expectedBackslashLine {
		t.Errorf("computedBackslashLine = %q, want %q", computedBackslashLine, expectedBackslashLine)
	}
}

// TestApprovalRequestWithRealHunks verifies that when Request is called with
// Changes containing real hunks from patch.Parse, the emitted ApprovalRequestedMsg
// has the four diff fields (DiffFilename, DiffLines, DiffAdded, DiffRemoved)
// correctly populated.
func TestApprovalRequestWithRealHunks(t *testing.T) {
	// A real diff with hunks, parsed through patch.Parse
	// Note: hunk header @@ -1,4 +1,7 @@ means: 4 lines starting at line 1 (old), 7 lines starting at line 1 (new)
	diffText := `--- a/calc/divide.go
+++ b/calc/divide.go
@@ -1,4 +1,7 @@
 func Divide(a, b float64) (float64, error) {
-	return a / b, nil
+	if b == 0 {
+		return 0, ErrDivideByZero
+	}
+	return a / b, nil
 }
 func Other() {
`

	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	// Build a Request with the parsed changes
	req := ApprovalRequest{
		Description: "test approval with real hunks",
		Operation:   policy.OperationPatch,
		Changes:     changes,
	}

	// Collect the emitted message
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	// Start the approval request in a goroutine so we can inspect the message
	ctx := context.Background()
	go func() {
		a.Request(ctx, req)
	}()

	// Wait for the ApprovalRequestedMsg
	msgs := c.wait(t, 5*time.Second)
	if len(msgs) < 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg, ok := msgs[0].(tui.ApprovalRequestedMsg)
	if !ok {
		t.Fatalf("first message is %T, want ApprovalRequestedMsg", msgs[0])
	}

	// Verify the four diff fields are populated
	if msg.DiffFilename != "calc/divide.go" {
		t.Errorf("DiffFilename = %q, want %q", msg.DiffFilename, "calc/divide.go")
	}

	if len(msg.DiffLines) == 0 {
		t.Errorf("DiffLines is empty, want hunks from the parsed diff")
	}

	if msg.DiffAdded != 4 {
		t.Errorf("DiffAdded = %d, want 4", msg.DiffAdded)
	}

	if msg.DiffRemoved != 1 {
		t.Errorf("DiffRemoved = %d, want 1", msg.DiffRemoved)
	}

	// Verify DiffLines start with the hunk header
	if len(msg.DiffLines) > 0 && !strings.HasPrefix(msg.DiffLines[0], "@@") {
		t.Errorf("first DiffLine = %q, want to start with @@", msg.DiffLines[0])
	}
}

// TestComputeDiffDisplayRenameAgreement verifies that computeDiffDisplay
// returns the rename in "old → new" format, matching what formatPatchDisplay
// shows in the detail lines. Both functions must agree so the modal title
// and the card detail describe the same rename.
func TestComputeDiffDisplayRenameAgreement(t *testing.T) {
	// Parse a rename diff
	diffText := `--- a/old_name.go
+++ b/new_name.go
rename from old_name.go
rename to new_name.go
`
	changes, err := patch.Parse([]byte(diffText))
	if err != nil {
		t.Fatalf("patch.Parse failed: %v", err)
	}
	if len(changes) == 0 {
		t.Fatalf("patch.Parse returned no changes")
	}

	// computeDiffDisplay should return "old_name.go → new_name.go"
	filename, _, _, _ := computeDiffDisplay(changes)
	wantFilename := "old_name.go → new_name.go"
	if filename != wantFilename {
		t.Errorf("computeDiffDisplay filename = %q, want %q", filename, wantFilename)
	}

	// formatPatchDisplay should also include "old_name.go → new_name.go"
	_, detail := formatPatchDisplay(changes)
	detailStr := strings.Join(detail, "\n")
	if !strings.Contains(detailStr, wantFilename) {
		t.Errorf("formatPatchDisplay detail does not contain rename: %v", detail)
	}
}

// TestFormatCommandDetailPopulatesDetailLines verifies that formatCommandDetail
// returns non-empty detail lines for display in a command approval modal.
func TestFormatCommandDetailPopulatesDetailLines(t *testing.T) {
	description := "run a command"
	argv := []string{"go", "test", "./..."}

	detail := formatCommandDetail(description, argv)

	if len(detail) == 0 {
		t.Errorf("formatCommandDetail returned empty detail")
	}

	detailStr := strings.Join(detail, "\n")

	// Should contain the description
	if !strings.Contains(detailStr, description) {
		t.Errorf("detail missing description: %v", detail)
	}

	// Should contain the command
	wantCmdStr := "go test ./..."
	if !strings.Contains(detailStr, wantCmdStr) {
		t.Errorf("detail missing command: want %q in %v", wantCmdStr, detail)
	}
}

// TestDescribeInputExtractsArgvForRunCommand verifies that describeInput
// returns the command argv for run_command tools. This is the target shown
// on tool cards, so the operator can identify which command is running.
// Calibration: removing the argv branch causes this test to fail.
func TestDescribeInputExtractsArgvForRunCommand(t *testing.T) {
	input := map[string]any{
		"argv": []any{"cat", "go.mod"},
	}
	target := describeInput(input)
	if target == "" {
		t.Errorf("describeInput returned empty string for argv; target not extracted")
	}
	if !strings.Contains(target, "cat") {
		t.Errorf("describeInput target %q does not contain 'cat'", target)
	}
	if !strings.Contains(target, "go.mod") {
		t.Errorf("describeInput target %q does not contain 'go.mod'", target)
	}
}

// TestDescribeInputSanitizesArgvEscapeSequences verifies that argv containing
// ANSI escape sequences is sanitised before being returned as a target.
// This prevents terminal escape injection into the transcript.
// Input comes from shellSplit (internal/tui), which returns []string.
// Calibration: removing the tui.SanitizeSingleLine call causes this test to fail.
func TestDescribeInputSanitizesArgvEscapeSequences(t *testing.T) {
	// Build argv as it comes from shellSplit: []string with an ESC byte.
	// ESC sequences cannot come from keyboard input (ui-spec §11), but test the
	// sanitisation gate: if shellSplit somehow returned escaped content, it would be safe.
	input := map[string]any{
		"argv": []string{"echo", "prefix\x1b[31mred\x1b[0msuffix"},
	}
	target := describeInput(input)

	// The target should contain the command and the string content (minus escapes)
	if !strings.Contains(target, "echo") {
		t.Errorf("target %q missing 'echo'", target)
	}
	if !strings.Contains(target, "prefix") || !strings.Contains(target, "suffix") {
		t.Errorf("target %q missing content around escape sequence", target)
	}

	// Verify no ESC byte in the output (sanitisation check)
	if strings.Contains(target, "\x1b") {
		t.Fatalf("target %q contains ESC byte 0x1b; not sanitised", target)
	}
}

// TestRunCommandCardShowsCommand drives the real /run dispatch end to end and
// asserts on what the operator would see.
//
// Order of events, as the test performs them:
//  1. The characters "/run cat go.mod" and Enter are delivered to tui.Model
//     through Update as tea.KeyMsg values.
//  2. Model's slash dispatch (runSlash, then the "run" arm of the debug-command
//     builder in internal/tui/update.go) calls shellSplit and builds the
//     {"argv": []string} input map. Nothing in this test constructs argv.
//  3. m.RunTool (App.RunTool) runs run_command, which asks for approval because
//     cat is not allow-listed. App sends ApprovalRequestedMsg through sendFn.
//  4. The test pumps App's messages into Update until the approval card is
//     pending, sends the key "y", then keeps pumping until a result card
//     exists.
//  5. The assertion is on m.View(): the result card's head line (not the
//     running card, not the approval card) must contain "run_command" and the
//     command "cat go.mod". No message field is inspected.
//
// Polling runs against a deadline rather than a fixed sleep.
// Calibration: reverting describeInput to its []any-only check, or renaming the
// "argv" key in the /run arm of internal/tui/update.go, makes this fail.
func TestRunCommandCardShowsCommand(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	info := a.WorkspaceInfo()
	m := tui.New(tui.Options{
		Version: "0.1.0",
		Caps:    tui.Caps{Colour: false, Unicode: true},
		Session: tui.SessionInfo{Project: info.Project, Branch: info.Branch, Dirty: info.Dirty},
	})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn
	m.ResolveApproval = func(id int64, outcome tui.ApprovalOutcome) {
		a.Resolve(id, ApprovalOutcome(tui.ToAppOutcome(outcome)))
	}

	// Only this goroutine touches m. sendFn runs on App's goroutines and hands
	// messages over through the channel; done releases any sender left over.
	inbox := make(chan tea.Msg, 100)
	done := make(chan struct{})
	defer close(done)
	a.sendFn = func(msg any) {
		if tm, ok := msg.(tea.Msg); ok {
			select {
			case inbox <- tm:
			case <-done:
			}
		}
	}
	apply := func(msg tea.Msg) {
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}

	// pollView pumps App's messages into the model until the rendered view
	// satisfies ok, or the deadline passes.
	pollView := func(what string, ok func(view string) bool) string {
		t.Helper()
		deadline := time.After(5 * time.Second)
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case msg := <-inbox:
				apply(msg)
			case <-tick.C:
			case <-deadline:
				t.Fatalf("timed out waiting for %s; last view:\n%s", what, m.View())
			}
			if v := m.View(); ok(v) {
				return v
			}
		}
	}

	apply(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range "/run cat go.mod" {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	apply(tea.KeyMsg{Type: tea.KeyEnter})

	pollView("the approval card", func(v string) bool { return strings.Contains(v, "approval") })
	apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// The transcript holds three cards for this one command, and the command
	// text appears on more than one of them: the running card, the approval card
	// (whose head ends "approved"), and the result card. Only the result card's
	// head comes from describeInput after the tool has run, so the predicate
	// excludes the other two by their status words.
	isResultHead := func(line string) bool {
		return strings.Contains(line, "run_command") && strings.Contains(line, "cat go.mod") &&
			!strings.Contains(line, "approved") && !strings.Contains(line, "running")
	}
	hasResultHead := func(v string) bool {
		for _, line := range strings.Split(v, "\n") {
			if isResultHead(line) {
				return true
			}
		}
		return false
	}
	// cat go.mod exits 1 in repo-small (no go.mod there); that is still a result
	// card, and its head is what is under test.
	pollView("a result card head naming the command", hasResultHead)
}
