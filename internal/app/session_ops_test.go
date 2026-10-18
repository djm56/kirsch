package app

// Tests for the session operations of m4-d8: manual /compact and /new.
//
// Entry points, so each test name can be read against what it really calls:
//   - ManualCompact tests call App.ManualCompact(getenvKey), the method the
//     /compact callback in cmd/kirsch/main.go returns. The provider is a
//     provider.Fake behind a.providerBuilder, as in compaction_test.go.
//   - NewSession tests call App.NewSession, the method the /new callback
//     returns, on a turn started through App.Submit.
//
// Nothing here calls runCompaction, RunTurn or installSession directly.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent/prompt"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
)

// manualSeedTokens sizes the seeded history. It is well under the 75%
// threshold, so a turn started over it would not compact on its own: any
// compaction a test observes was asked for.
const manualSeedTokens = 5000

// waitRequests blocks until the fake has received n requests.
func (e *compactionEnv) waitRequests(n int) {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(e.fake.Requests()) < n {
		if time.Now().After(deadline) {
			e.t.Fatalf("provider received %d requests after 5s, want %d", len(e.fake.Requests()), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// manualCompactAsync runs ManualCompact on a goroutine and returns the
// channel its message arrives on.
func (e *compactionEnv) manualCompactAsync() <-chan tui.ManualCompactMsg {
	ch := make(chan tui.ManualCompactMsg, 1)
	go func() { ch <- e.a.ManualCompact(getenvKey) }()
	return ch
}

// await returns the message or fails the test after 5 seconds.
func await(t *testing.T, ch <-chan tui.ManualCompactMsg) tui.ManualCompactMsg {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("ManualCompact did not return within 5s")
		return tui.ManualCompactMsg{}
	}
}

// TestManualCompactAppliesWithoutStartingATurn calls ManualCompact over an
// eight-turn history. The summary request must be the only provider call, the
// conversation must be replaced, compaction.applied must already be in the
// file when ManualCompact returns (the flush on exit), and no turn may start.
// Resume over the file must rebuild the same conversation.
func TestManualCompactAppliesWithoutStartingATurn(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary})
	e.seed(8, manualSeedTokens)
	snapshot := cloneMsgs(e.a.conv.Messages)

	got := e.a.ManualCompact(getenvKey)

	if got.Outcome != tui.ManualCompactApplied {
		t.Fatalf("outcome = %v (detail %q), want Applied", got.Outcome, got.Detail)
	}
	reqs := e.fake.Requests()
	if len(reqs) != 1 {
		t.Fatalf("provider requests = %d, want 1 (the summary)", len(reqs))
	}
	if len(reqs[0].System) == 0 || !strings.Contains(reqs[0].System[0].Text, "Objective") {
		t.Errorf("request 1 is not the summary request: system = %+v", reqs[0].System)
	}
	if reqs[0].MaxTokens != outputReserve {
		t.Errorf("summary MaxTokens = %d, want the output reserve %d", reqs[0].MaxTokens, outputReserve)
	}
	conv := e.a.conv.Messages
	if len(conv) >= len(snapshot) {
		t.Fatalf("conv has %d messages, not fewer than the %d seeded", len(conv), len(snapshot))
	}
	if msgText(provider.Message{Content: agentBlocksToProvider(conv[0].Content)}) != snapshot[0].Content[0].Text {
		t.Error("conv message 0 is not the original first message")
	}
	if !strings.Contains(conv[1].Content[0].Text, "Objective: finish the refactor.") {
		t.Errorf("conv message 1 is not the summary: %q", conv[1].Content[0].Text)
	}

	// Read straight away: only ManualCompact's own flush can have put the
	// event in the file.
	types := e.logTypes()
	if got := countOf(types, "compaction.applied"); got != 1 {
		t.Errorf("compaction.applied events = %d, want 1 (%v)", got, types)
	}
	if indexOf(types, "turn.started") >= 0 {
		t.Errorf("manual compaction started a turn: %v", types)
	}
	if n := e.a.turnCount.Load(); n != 0 {
		t.Errorf("turnCount = %d, want 0", n)
	}
	if n := e.c.count(); n != 0 {
		t.Errorf("ManualCompact sent %d TUI messages itself, want 0: %#v", n, e.c.snapshot())
	}

	want := cloneMsgs(conv)
	e.a.Close()
	resumed, _, err := session.Resume(e.logPath())
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !reflect.DeepEqual(resumed.Messages, want) {
		t.Errorf("resumed conversation differs from the live one: %d vs %d messages", len(resumed.Messages), len(want))
	}
}

// TestManualCompactWithTooFewTurnsReportsNothing calls ManualCompact over a
// four-turn history. It mirrors the applied test: same call, fewer turns.
func TestManualCompactWithTooFewTurnsReportsNothing(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary})
	e.seed(4, manualSeedTokens)
	snapshot := cloneMsgs(e.a.conv.Messages)
	before := e.logTypes()

	got := e.a.ManualCompact(getenvKey)

	if got.Outcome != tui.ManualCompactNothing {
		t.Fatalf("outcome = %v (detail %q), want Nothing", got.Outcome, got.Detail)
	}
	if n := len(e.fake.Requests()); n != 0 {
		t.Errorf("provider requests = %d, want 0: nothing to compact must not call the model", n)
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("conv changed")
	}
	if after := e.logTypes(); !reflect.DeepEqual(after, before) {
		t.Errorf("log changed from %v to %v", before, after)
	}
}

// TestManualCompactProviderErrorIsAFailureNotAnOverflow calls ManualCompact
// with a summary call that fails. The outcome must be Failed with the cause,
// and unlike a failed compaction inside a turn it must not send a
// context_overflow TurnErrorMsg: the manual command reports through its one
// ManualCompactMsg.
func TestManualCompactProviderErrorIsAFailureNotAnOverflow(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Err: errors.New("provider exploded")})
	e.seed(8, manualSeedTokens)
	snapshot := cloneMsgs(e.a.conv.Messages)
	before := e.logTypes()

	got := e.a.ManualCompact(getenvKey)

	if got.Outcome != tui.ManualCompactFailed {
		t.Fatalf("outcome = %v, want Failed", got.Outcome)
	}
	if !strings.Contains(got.Detail, "provider exploded") {
		t.Errorf("detail %q does not carry the cause", got.Detail)
	}
	time.Sleep(100 * time.Millisecond) // let any stray async send land
	if findMsgType[tui.TurnErrorMsg](e.c.snapshot()) {
		t.Errorf("a failed manual compaction sent a TurnErrorMsg: %#v", e.c.snapshot())
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("conv changed")
	}
	if after := e.logTypes(); !reflect.DeepEqual(after, before) {
		t.Errorf("log changed from %v to %v", before, after)
	}
}

// TestManualCompactCancelDuringModelCallReportsCancelled calls ManualCompact
// with a summary call that blocks, then CancelTurn. session.Compact wraps the
// context error inside ErrCompactionFailed, so this is the path a text match
// on "cancelled" misses; the outcome must be Cancelled, not Failed.
func TestManualCompactCancelDuringModelCallReportsCancelled(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{BlockUntilCancel: true})
	e.seed(8, manualSeedTokens)
	snapshot := cloneMsgs(e.a.conv.Messages)
	before := e.logTypes()

	ch := e.manualCompactAsync()
	e.waitRequests(1)
	select {
	case msg := <-ch:
		t.Fatalf("ManualCompact returned %#v before it was cancelled", msg)
	case <-time.After(100 * time.Millisecond):
	}
	e.a.CancelTurn()
	got := await(t, ch)

	if got.Outcome != tui.ManualCompactCancelled {
		t.Fatalf("outcome = %v (detail %q), want Cancelled", got.Outcome, got.Detail)
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("conv changed by a cancelled compaction")
	}
	if after := e.logTypes(); !reflect.DeepEqual(after, before) {
		t.Errorf("log changed from %v to %v", before, after)
	}
	if e.a.isCompacting() {
		t.Error("compacting flag still set after ManualCompact returned")
	}
}

// TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies calls
// ManualCompact with a summary call that succeeds and cancels from
// afterCompactHook, which runs after the summary and before the swap. The
// hook is runCompaction's seam, so this fails if ManualCompact does not go
// through runCompaction or drops its cancellation re-check.
func TestManualCompactCancelAfterSummaryBeforeSwapNeverHalfApplies(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary})
	e.seed(8, manualSeedTokens)
	snapshot := cloneMsgs(e.a.conv.Messages)
	before := e.logTypes()
	e.a.afterCompactHook = e.a.CancelTurn

	got := e.a.ManualCompact(getenvKey)

	if n := len(e.fake.Requests()); n != 1 {
		t.Fatalf("provider requests = %d, want 1 (the summary completed)", n)
	}
	if got.Outcome != tui.ManualCompactCancelled {
		t.Fatalf("outcome = %v (detail %q), want Cancelled", got.Outcome, got.Detail)
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("conv was replaced although the compaction was cancelled")
	}
	if after := e.logTypes(); !reflect.DeepEqual(after, before) {
		t.Errorf("log changed from %v to %v", before, after)
	}
}

// TestManualCompactRefusesWhileATurnIsLive starts a turn through Submit whose
// provider call blocks, then calls ManualCompact. It must report Busy without
// a second provider request. After the turn has ended, the same call must
// compact, which is the control showing the refusal was about the live turn.
func TestManualCompactRefusesWhileATurnIsLive(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{BlockUntilCancel: true}, provider.Turn{Text: compactionSummary})
	e.seed(8, manualSeedTokens)

	e.a.Submit(compactionUserText, getenvKey)
	e.waitRequests(1)

	got := e.a.ManualCompact(getenvKey)

	if got.Outcome != tui.ManualCompactBusy {
		t.Fatalf("outcome during a live turn = %v, want Busy", got.Outcome)
	}
	if n := len(e.fake.Requests()); n != 1 {
		t.Errorf("provider requests = %d, want 1 (the turn only)", n)
	}

	e.a.CancelTurn()
	e.settle()
	if got := e.a.ManualCompact(getenvKey); got.Outcome != tui.ManualCompactApplied {
		t.Errorf("outcome after the turn ended = %v (detail %q), want Applied", got.Outcome, got.Detail)
	}
}

// TestSubmitRefusesWhileManualCompactIsRunning starts ManualCompact with a
// blocking summary call, then calls Submit. Submit must send a busy
// TurnErrorMsg and make no provider request; after the compaction is
// cancelled, Submit must run its turn, which is the control.
func TestSubmitRefusesWhileManualCompactIsRunning(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{BlockUntilCancel: true}, turnReply)
	e.seed(8, manualSeedTokens)

	ch := e.manualCompactAsync()
	e.waitRequests(1)

	e.a.Submit(compactionUserText, getenvKey)

	deadline := time.Now().Add(5 * time.Second)
	var refusal *tui.TurnErrorMsg
	for refusal == nil {
		if refusal = findMsg[tui.TurnErrorMsg](e.c.snapshot()); refusal == nil {
			if time.Now().After(deadline) {
				t.Fatal("Submit during a compaction sent no TurnErrorMsg")
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	if refusal.Kind != "busy" {
		t.Errorf("refusal kind = %q, want busy", refusal.Kind)
	}
	if n := len(e.fake.Requests()); n != 1 {
		t.Errorf("provider requests = %d, want 1 (the summary only)", n)
	}

	e.a.CancelTurn()
	if got := await(t, ch); got.Outcome != tui.ManualCompactCancelled {
		t.Fatalf("compaction outcome = %v, want Cancelled", got.Outcome)
	}

	e.a.Submit(compactionUserText, getenvKey)
	e.waitRequests(2)
	e.a.inFlight.Wait()
	if !findMsgType[tui.TurnCompleteMsg](e.c.snapshot()) {
		// The terminal message is delivered asynchronously.
		waitForMsg[tui.TurnCompleteMsg](t, e.c, 5*time.Second)
	}
}

// lateProvider models a provider that is slow to stop. Stream blocks until
// its context is cancelled, takes lag longer to wind down, and then completes
// the reply anyway, usage included.
type lateProvider struct {
	started chan struct{}
	once    sync.Once
	lag     time.Duration
}

func (p *lateProvider) Stream(ctx context.Context, _ provider.Request, onEvent func(provider.StreamEvent)) error {
	p.once.Do(func() { close(p.started) })
	<-ctx.Done()
	time.Sleep(p.lag)
	onEvent(provider.StreamEvent{Type: provider.EventTextDelta, Text: "late reply"})
	onEvent(provider.StreamEvent{
		Type:       provider.EventMessageDone,
		Usage:      &provider.Usage{InputTokens: 7, OutputTokens: 3},
		StopReason: provider.StopEndTurn,
	})
	return nil
}

// jsonlFiles lists the session logs under the env's storage dir.
func (e *compactionEnv) jsonlFiles() []string {
	e.t.Helper()
	var found []string
	_ = filepath.WalkDir(filepath.Join(e.dir, "sessions"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".jsonl") {
			found = append(found, p)
		}
		return nil
	})
	return found
}

// typesOf returns the event types in the JSONL file at path.
func typesOf(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		out = append(out, jsonType(t, line))
	}
	return out
}

// jsonType returns the "type" field of one JSONL line.
func jsonType(t *testing.T, line string) string {
	t.Helper()
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(line), &base); err != nil {
		t.Fatalf("log line %q: %v", line, err)
	}
	return base.Type
}

// TestNewSessionIsolatesTheOldTurnFromTheNewSession starts a turn through
// Submit whose provider is slow to stop, then calls NewSession. NewSession
// must return only after the old turn has finished into the OLD log; the old
// log must not change afterwards; the NEW log must hold nothing from the old
// turn; and every piece of session state must be fresh. The provider's late
// usage event is what would land in the new store if NewSession did not wait.
func TestNewSessionIsolatesTheOldTurnFromTheNewSession(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	lp := &lateProvider{started: make(chan struct{}), lag: 150 * time.Millisecond}
	e.a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return lp, nil
	}
	e.seed(8, manualSeedTokens)
	oldID, oldPath := e.a.currentSessionID, e.logPath()
	oldConv := e.a.conv

	// Session state the new session must not inherit.
	e.a.storeMu.Lock()
	e.a.sessionTokensIn, e.a.sessionTokensOut = 100, 200
	e.a.sessionTokensRead, e.a.sessionTokensWrite = 30, 40
	e.a.lastEstimatedTokens, e.a.lastContextWindow = 500, compactionWindow
	e.a.storeMu.Unlock()
	e.a.mu.Lock()
	e.a.addTouchedFile("file.go")
	e.a.mu.Unlock()
	if err := e.a.pol.Grant(policy.OperationCommand, []string{"go", "test"}); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if got := e.a.Grants(); len(got) != 1 {
		t.Fatalf("setup: grants = %v, want 1", got)
	}

	// The project context changes between sessions: /new must re-read it.
	e.a.loadProjectCtxFn = func([]string, int, prompt.Engine) (string, string, int, []string, error) {
		return "NEW-SESSION-CONTEXT-MARKER", "ctx.md", 77, nil, nil
	}
	if strings.Contains(e.a.promptState(), "NEW-SESSION-CONTEXT-MARKER") {
		t.Fatal("setup: marker already in the prompt")
	}

	e.a.Submit(compactionUserText, getenvKey)
	<-lp.started
	if n := e.a.turnCount.Load(); n != 1 {
		t.Fatalf("setup: turnCount = %d with the turn live, want 1", n)
	}

	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	// Old and new files.
	files := e.jsonlFiles()
	if len(files) != 2 {
		t.Fatalf("session logs = %v, want old and new", files)
	}
	if e.a.currentSessionID == oldID {
		t.Errorf("session id unchanged after NewSession: %s", oldID)
	}
	var newPath string
	for _, f := range files {
		if f != oldPath {
			newPath = f
		}
	}
	if newPath == "" {
		t.Fatalf("no new session file among %v (old %s)", files, oldPath)
	}

	// The old turn finished into the old log before NewSession returned.
	oldTypes := typesOf(t, oldPath)
	if indexOf(oldTypes, "usage") < 0 || indexOf(oldTypes, "turn.completed") < 0 {
		t.Errorf("old log lacks the old turn's usage/turn.completed: %v", oldTypes)
	}
	oldBytes, _ := os.ReadFile(oldPath)

	// The new log holds none of it. Flush first so a queued late write
	// would be in the file.
	e.a.storeMu.Lock()
	newStore := e.a.store
	e.a.storeMu.Unlock()
	if err := newStore.Flush(); err != nil {
		t.Fatalf("Flush new store: %v", err)
	}
	if newTypes := typesOf(t, newPath); len(newTypes) != 0 {
		t.Errorf("new log holds events from the old session: %v", newTypes)
	}

	// The old log stays unchanged, past the store's 500ms flush cadence.
	time.Sleep(700 * time.Millisecond)
	if after, _ := os.ReadFile(oldPath); string(after) != string(oldBytes) {
		t.Error("old log changed after NewSession returned")
	}

	// State.
	if len(e.a.conv.Messages) != 0 || e.a.conv == oldConv {
		t.Errorf("conv not replaced by an empty one: %d messages, same pointer %v", len(e.a.conv.Messages), e.a.conv == oldConv)
	}
	e.a.storeMu.Lock()
	tokens := []int64{e.a.sessionTokensIn, e.a.sessionTokensOut, e.a.sessionTokensRead, e.a.sessionTokensWrite}
	est, win := e.a.lastEstimatedTokens, e.a.lastContextWindow
	e.a.storeMu.Unlock()
	if !reflect.DeepEqual(tokens, []int64{0, 0, 0, 0}) {
		t.Errorf("token totals = %v, want all zero (the late usage event is the old turn's)", tokens)
	}
	if est != 0 || win != 0 {
		t.Errorf("lastEstimatedTokens/lastContextWindow = %d/%d, want 0/0", est, win)
	}
	if n := e.a.turnCount.Load(); n != 0 {
		t.Errorf("turnCount = %d, want 0", n)
	}
	if got := e.a.GetTouchedFiles(); len(got) != 0 {
		t.Errorf("touched files not cleared: %v", got)
	}
	if got := e.a.Grants(); len(got) != 0 {
		t.Errorf("grants not cleared: %v", got)
	}
	if !strings.Contains(e.a.promptState(), "NEW-SESSION-CONTEXT-MARKER") {
		t.Error("system prompt was not re-read for the new session")
	}
	if file, size := e.a.contextState(); file != "ctx.md" || size != 77 {
		t.Errorf("context file/size = %q/%d, want ctx.md/77", file, size)
	}
}

// TestNewSessionFailureLeavesTheSessionIntact calls NewSession with the
// storage directory unwritable. It must return the error and change nothing:
// same store and id, conversation, totals, grants and touched files, and the
// running turn's context must not have been cancelled.
func TestNewSessionFailureLeavesTheSessionIntact(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	e.seed(8, manualSeedTokens)
	oldID, oldConv := e.a.currentSessionID, e.a.conv
	snapshot := cloneMsgs(e.a.conv.Messages)
	e.a.storeMu.Lock()
	oldStore := e.a.store
	e.a.sessionTokensIn = 100
	e.a.lastEstimatedTokens = 500
	e.a.storeMu.Unlock()
	e.a.turnCount.Store(3)
	e.a.mu.Lock()
	e.a.addTouchedFile("file.go")
	e.a.mu.Unlock()
	if err := e.a.pol.Grant(policy.OperationCommand, []string{"go", "test"}); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	turnCtx := e.a.BeginTurn()

	// A regular file where the sessions directory must go.
	blocker := filepath.Join(e.dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	e.a.cfg.Session.StorageDir = filepath.Join(blocker, "sessions")

	err := e.a.NewSession(getenvKey)

	if err == nil {
		t.Fatal("NewSession succeeded against an unwritable storage dir")
	}
	e.a.storeMu.Lock()
	sameStore := e.a.store == oldStore
	tokens, est := e.a.sessionTokensIn, e.a.lastEstimatedTokens
	e.a.storeMu.Unlock()
	if !sameStore || e.a.currentSessionID != oldID {
		t.Errorf("store replaced (same=%v) or id changed (%q -> %q) by a failed NewSession", sameStore, oldID, e.a.currentSessionID)
	}
	if e.a.conv != oldConv || !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("conversation reset by a failed NewSession")
	}
	if tokens != 100 || est != 500 || e.a.turnCount.Load() != 3 {
		t.Errorf("counters reset by a failed NewSession: tokensIn=%d estimate=%d turnCount=%d", tokens, est, e.a.turnCount.Load())
	}
	if got := e.a.GetTouchedFiles(); len(got) != 1 {
		t.Errorf("touched files = %v, want the one from before", got)
	}
	if got := e.a.Grants(); len(got) != 1 {
		t.Errorf("grants = %v, want the one from before", got)
	}
	select {
	case <-turnCtx.Done():
		t.Error("a failed NewSession cancelled the running turn")
	default:
	}
}

// TestSubmitRefusesWhileNewSessionWaitsForATurn calls NewSession on a slow
// turn and, while it is waiting for that turn to stop, calls Submit. Submit
// must be refused with a busy TurnErrorMsg and must not reach the provider; it
// is the case where a Submit passes StartSession and would otherwise slip
// past inFlight.Wait. After NewSession returns, Submit runs, which is the
// control.
func TestSubmitRefusesWhileNewSessionWaitsForATurn(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	lp := &lateProvider{started: make(chan struct{}), lag: 300 * time.Millisecond}
	e.a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return lp, nil
	}
	e.seed(8, manualSeedTokens)
	e.a.Submit(compactionUserText, getenvKey)
	<-lp.started

	done := make(chan error, 1)
	go func() { done <- e.a.NewSession(getenvKey) }()

	// Wait until NewSession holds opMu.
	deadline := time.Now().Add(5 * time.Second)
	for e.a.opMu.TryLock() {
		e.a.opMu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("NewSession never took opMu")
		}
		time.Sleep(time.Millisecond)
	}

	e.a.Submit(compactionUserText, getenvKey)
	refusal := waitForMsg[tui.TurnErrorMsg](t, e.c, 5*time.Second)
	if refusal.Kind != "busy" {
		t.Errorf("refusal kind = %q, want busy", refusal.Kind)
	}
	if err := <-done; err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if got := e.a.conv.Messages; len(got) != 0 {
		t.Errorf("conv has %d messages after NewSession; the refused Submit leaked into it", len(got))
	}

	// Control: with NewSession finished, a Submit starts a turn again.
	e.a.Submit(compactionUserText, getenvKey)
	e.a.CancelTurn()
	e.a.inFlight.Wait()
	if got := e.a.turnCount.Load(); got != 1 {
		t.Errorf("turnCount after the post-NewSession Submit = %d, want 1", got)
	}
}
