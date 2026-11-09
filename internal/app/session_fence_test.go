package app

// Tests for C49: messages from a turn that /new cancelled must not reach the
// TUI after NewSession has returned, because by then the TUI is about to show
// (or already shows) the new session.
//
// Entry points, so each test name can be read against what it really calls:
//   - TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns calls
//     App.Submit to start the turn (or, for the refusal subtest, to refuse it)
//     and App.NewSession, the method the /new callback returns, to cancel it.
//     Delivery goes through the real sendAsyncSequenceFenced path only (no
//     subtest's turn calls a tool, so none reaches send) into a sendFn that
//     stalls one message type per subtest, which stands in for a Bubble Tea
//     program that is slow to accept a message. The stall is released by an
//     event (see runNewSessionFence). The subtests with no in-flight control
//     first run the same Submit and cancel it with App.CancelTurn instead (see
//     requireSentOnCancel).
//   - TestTurnToolContextCarriesTheSubmitGeneration calls App.Submit with a
//     scripted provider whose first turn calls a probe tool, so RunTurn and the
//     real tool executor run; the probe records what its context carries.
//   - TestCurrentGenDoesNotWaitForAHeldGenMuWriteLock takes genMu.Lock directly
//     and calls currentGen and deliverIfCurrent; no bump and no NewSession.
//   - TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne calls
//     App.NewSession to move the generation on, then calls each sender
//     directly (sendAsyncSequenceFenced, send, Request, toolExecutor.Invoke,
//     flushStore, recordCompactionUsage) with a generation read before and
//     one read after. It does not go through RunTurn: a turn cannot be held
//     between creating a message and delivering it from outside, so the
//     generation is supplied by the test.
//   - TestTurnPathsUseOnlyFencedSenders parses the non-test files of package
//     app with go/ast and runs none of them. It is the pin for the fenced call sites no behavioural test
//     here reaches.

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
)

// stallingSink is a sendFn that records every message. Until open is called it
// blocks the delivery of each message for which stall returns true, before
// recording it; every other message is recorded at once. started lists the
// messages whose delivery has begun, so a test can tell what was in flight
// while the sink was stalled.
type stallingSink struct {
	gate     chan struct{}
	openOnce sync.Once
	stall    func(msg any) bool
	mu       sync.Mutex
	got      []any
	started  []any // every message whose delivery has begun, delivered or not yet
}

func newStallingSink(stall func(msg any) bool) *stallingSink {
	return &stallingSink{gate: make(chan struct{}), stall: stall}
}

func (s *stallingSink) send(msg any) {
	s.mu.Lock()
	s.started = append(s.started, msg)
	s.mu.Unlock()
	if s.stall(msg) {
		<-s.gate
	}
	s.mu.Lock()
	s.got = append(s.got, msg)
	s.mu.Unlock()
}

// stalled returns a copy of the messages whose delivery has begun, whether or
// not it has finished.
func (s *stallingSink) stalled() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]any(nil), s.started...)
}

// open lets the stalled deliveries through. It may be called more than once.
func (s *stallingSink) open() { s.openOnce.Do(func() { close(s.gate) }) }

// delivered returns a copy of what has been recorded so far.
func (s *stallingSink) delivered() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]any(nil), s.got...)
}

// runNewSessionFence runs one scenario of the NewSession fence. start is
// called once and starts the work whose messages are under test; sink must
// stall only those messages. The scenario then waits for control to hold of
// the messages whose delivery has begun (control may be nil when no message is
// guaranteed to be in flight), calls NewSession, and checks two things.
//
//   - recorded, when non-nil, must hold of what the sink had recorded when
//     NewSession returned. For a fenced message whose delivery had passed its
//     check, it does: the bump waits for that delivery, and the sink is
//     released while the bump waits (below).
//   - nothing may be recorded after NewSession returned.
//
// The stall is released by an event, not by a delay. A goroutine polls
// genMu.TryRLock, which fails exactly while bumpGeneration is waiting for or
// holding the write lock, and releases the sink then; a fenced delivery in
// flight makes the bump wait at that point, so the sink must be released there
// for the bump to proceed. If the poll never sees the write lock, the
// goroutine releases the sink once NewSession has returned and the test has
// taken its snapshot. The poll can also catch the brief write-lock hold of a
// bump that waited for nothing and release the sink early, before NewSession
// returns. An unfenced message stalled in the sink can then be recorded before
// the snapshot, and neither check sees it. So these subtests do not prove a
// sender is fenced; TestTurnPathsUseOnlyFencedSenders pins that. The 30s cap
// only keeps a broken fence from hanging the run; it reports an error when it
// fires. A cleanup opens the sink however the scenario ends, including a
// failed control or a failing start, so no delivery goroutine is left
// holding a genMu read lock.
func runNewSessionFence(t *testing.T, a *App, sink *stallingSink, start func(), control, recorded func([]any) bool) {
	t.Helper()
	t.Cleanup(sink.open)
	start()

	if control != nil {
		deadline := time.Now().Add(5 * time.Second)
		for !control(sink.stalled()) && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if !control(sink.stalled()) {
			t.Fatalf("control: the message under test never reached the sink; the test proves nothing (saw %#v)", sink.stalled())
		}
	}

	returned := make(chan struct{})
	go func() {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-returned:
				sink.open()
				return
			default:
			}
			if a.genMu.TryRLock() {
				a.genMu.RUnlock()
				time.Sleep(time.Millisecond)
				continue
			}
			// A writer is waiting: the bump is pending behind a stalled delivery.
			sink.open()
			return
		}
		t.Errorf("the sink was never released: NewSession neither returned nor waited on the generation fence")
		sink.open()
	}()

	err := a.NewSession(getenvKey)
	atReturnMsgs := sink.delivered()
	close(returned)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if recorded != nil && !recorded(atReturnMsgs) {
		t.Errorf("NewSession returned while the old turn's message was still undelivered: %#v", atReturnMsgs)
	}

	// Wait until every delivery that has begun has finished, then give a
	// delivery that has not begun yet the same again to show up.
	deadline := time.Now().Add(5 * time.Second)
	for len(sink.stalled()) != len(sink.delivered()) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	if all := sink.delivered(); len(all) != len(atReturnMsgs) {
		t.Errorf("%d old-turn message(s) delivered after NewSession returned: %#v", len(all)-len(atReturnMsgs), all[len(atReturnMsgs):])
	}
}

// TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns starts a turn
// through Submit against a provider that is slow to stop, then calls NewSession
// (the method the /new callback returns) to cancel it. Each subtest stalls one
// message type only, so a fenced message cannot mask an unfenced one: Submit's
// BudgetMsg, the turn's AssistantTextDeltaMsg, the turn's final message, and
// Submit's refusal for an unknown endpoint. Delivery goes through the real
// sendAsyncSequenceFenced path into a sendFn that stalls (see
// runNewSessionFence for how the stall is released and what is checked).
// Most other messages a turn can send have no subtest here, for example
// CompactingMsg, SessionCompactedMsg, the flush ErrorMsg, SessionUsageMsg and
// Submit's other refusals; TestTurnPathsUseOnlyFencedSenders pins their call
// sites, and TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne drives
// flushStore and recordCompactionUsage directly.
//
// BudgetMsg is sent by Submit at once, before the turn starts, so it is
// in flight and past the fence's check when the bump comes; that subtest has a
// control (the message is stalled in the sink before NewSession is called) and
// requires the message in the record by the time NewSession returns. The delta
// and the final message are sent by the turn goroutine as the provider unwinds
// after the cancel, and the bump may land before or after each: each may be
// dropped by the fence or delivered ahead of the bump, so those subtests can
// require only that nothing arrives after NewSession returns. That is an
// absence check, so each first runs requireSentOnCancel, the presence case:
// the same Submit cancelled without a NewSession must deliver the message.
// The refusal subtest has the same control as BudgetMsg, for the TurnErrorMsg
// Submit sends when the configured endpoint does not exist.
func TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns(t *testing.T) {
	isBudget := func(msg any) bool { _, ok := msg.(tui.BudgetMsg); return ok }
	isDelta := func(msg any) bool { _, ok := msg.(tui.AssistantTextDeltaMsg); return ok }
	// A turn ends with exactly one of these, so stalling them together stalls
	// one message.
	isFinal := func(msg any) bool {
		switch msg.(type) {
		case tui.TurnCompleteMsg, tui.TurnCancelledMsg, tui.TurnErrorMsg:
			return true
		}
		return false
	}
	isConfigRefusal := func(msg any) bool {
		e, ok := msg.(tui.TurnErrorMsg)
		return ok && e.Kind == "config"
	}
	has := func(is func(any) bool) func([]any) bool {
		return func(msgs []any) bool {
			for _, m := range msgs {
				if is(m) {
					return true
				}
			}
			return false
		}
	}

	turnCases := []struct {
		name     string
		stall    func(any) bool
		inFlight bool // guaranteed to be stalled in the sink before /new begins
	}{
		{"BudgetMsg", isBudget, true},
		{"AssistantTextDeltaMsg", isDelta, false},
		{"final turn message", isFinal, false},
	}
	for _, tc := range turnCases {
		t.Run(tc.name, func(t *testing.T) {
			e := newCompactionEnv(t, 8192)
			sink := newStallingSink(tc.stall)
			e.a.sendFn = sink.send
			lp := &lateProvider{started: make(chan struct{}), lag: 50 * time.Millisecond}
			e.a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
				return lp, nil
			}
			e.seed(8, manualSeedTokens)

			var control, recorded func([]any) bool
			if tc.inFlight {
				control, recorded = has(tc.stall), has(tc.stall)
			} else {
				requireSentOnCancel(t, tc.stall)
			}
			runNewSessionFence(t, e.a, sink, func() {
				e.a.Submit(compactionUserText, getenvKey)
				<-lp.started
			}, control, recorded)
		})
	}

	t.Run("Submit refusal", func(t *testing.T) {
		e := newCompactionEnv(t, 8192)
		sink := newStallingSink(isConfigRefusal)
		e.a.sendFn = sink.send
		e.a.cfg.Provider.Default = "no-such-endpoint"
		runNewSessionFence(t, e.a, sink, func() {
			e.a.Submit(compactionUserText, getenvKey)
		}, has(isConfigRefusal), has(isConfigRefusal))
	})
}

// requireSentOnCancel is the presence control for a message the fence test can
// only check for absence. It builds a fresh env with the same slow provider
// and seed, Submits the same text, waits for the provider to start, and
// cancels the turn with App.CancelTurn: no NewSession, so no generation bump.
// The message must then be delivered through the same sendAsyncSequenceFenced
// path into a sink that stalls nothing. It fails if is matches nothing the
// turn delivered within 5s.
func requireSentOnCancel(t *testing.T, is func(any) bool) {
	t.Helper()
	e := newCompactionEnv(t, 8192)
	sink := newStallingSink(func(any) bool { return false })
	e.a.sendFn = sink.send
	lp := &lateProvider{started: make(chan struct{}), lag: 50 * time.Millisecond}
	e.a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return lp, nil
	}
	e.seed(8, manualSeedTokens)
	e.a.Submit(compactionUserText, getenvKey)
	<-lp.started
	e.a.CancelTurn()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, m := range sink.delivered() {
			if is(m) {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("presence control: with no NewSession, the cancelled turn never delivered the message under test (got %#v)", sink.delivered())
}

// probeTool is a tool that records what its context carries when RunTurn's tool
// executor invokes it.
type probeTool struct {
	gen    atomic.Uint64
	hasGen atomic.Bool
}

func (*probeTool) Name() string            { return "gen_probe" }
func (*probeTool) Description() string     { return "records the turn generation in its context" }
func (*probeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (p *probeTool) Invoke(ctx context.Context, _ json.RawMessage) tool.Result {
	g, ok := ctx.Value(turnGenKey{}).(uint64)
	p.gen.Store(g)
	p.hasGen.Store(ok)
	return tool.OKResult("done", "done", false)
}

// TestTurnToolContextCarriesTheSubmitGeneration calls App.Submit after one
// NewSession (so the generation is not the zero value) with a provider that
// calls the probe tool, then checks the tool's context carried the generation
// Submit started under. This is what lets a message sent on the turn's behalf
// by the tool executor stay fenced to that generation; without the context
// value the executor would fall back to whatever generation is current when it
// sends. The control is the same Submit's ToolCompletedMsg, delivered for the
// same call, which shows the tool really ran under RunTurn.
func TestTurnToolContextCarriesTheSubmitGeneration(t *testing.T) {
	probe := &probeTool{}
	e := newCompactionEnv(t, 8192,
		provider.Turn{ToolCalls: []provider.ToolCall{{ID: "p1", Name: "gen_probe", Input: json.RawMessage(`{}`)}}},
		provider.Turn{Text: "ok"},
	)
	e.a.reg.Register(probe)
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	want := e.a.currentGen()
	if want == 0 {
		t.Fatal("setup: generation is still 0 after NewSession")
	}

	e.a.Submit(compactionUserText, getenvKey)
	deadline := time.Now().Add(5 * time.Second)
	for !hasToolCompleted(e.c.snapshot(), "gen_probe") {
		if time.Now().After(deadline) {
			t.Fatalf("control: the probe tool never completed under RunTurn; got %#v", e.c.snapshot())
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !probe.hasGen.Load() {
		t.Fatal("the tool's context carried no turn generation")
	}
	if got := probe.gen.Load(); got != want {
		t.Errorf("the tool's context carried generation %d, want %d", got, want)
	}
}

// TestCurrentGenDoesNotWaitForAHeldGenMuWriteLock takes genMu for writing
// directly (genMu.Lock in the test; bumpGeneration is not called) and calls
// currentGen, which runs on the Update goroutine in TUI callbacks such as
// ClearGrants and Submit, and in RunTurn (called by Submit). It must return at
// once. The test never makes a bump pending: it holds the write lock itself,
// which is the state genMu is in while bumpGeneration waits behind a stalled
// delivery as far as readers are concerned (a waiting writer holds back new
// readers, as a held one does).
// bumpGeneration itself holds the write lock only for its Add(1). The control
// is deliverIfCurrent called while the lock is held: it must still be waiting,
// which shows the lock really is held against readers that take it. No
// NewSession is involved.
func TestCurrentGenDoesNotWaitForAHeldGenMuWriteLock(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	gen := e.a.currentGen()
	e.a.genMu.Lock()

	delivered := make(chan struct{})
	go func() {
		e.a.deliverIfCurrent(gen, func() {})
		close(delivered)
	}()
	select {
	case <-delivered:
		e.a.genMu.Unlock()
		t.Fatal("control: deliverIfCurrent returned while genMu was held for writing")
	case <-time.After(100 * time.Millisecond):
	}

	got := make(chan uint64, 1)
	go func() { got <- e.a.currentGen() }()
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		e.a.genMu.Unlock()
		t.Fatal("currentGen blocked while genMu was held for writing")
	}
	e.a.genMu.Unlock()
	<-delivered
}

// TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne moves the
// session generation on with NewSession, then sends one message through each
// fenced sender under the old generation and then one under the new. The
// current-generation message is the positive control for the same sender: it
// shows the sender delivers at all, so the stale one's absence means it was
// dropped.
//
// Most pairs are told apart by a marker in their text. flushStore's two
// reports are identical (the store was closed by NewSession, so both flushes
// fail the same way), so that pair is told apart by count: exactly one
// session_store ErrorMsg. recordCompactionUsage's pair is told apart by its
// figures: the stale call records 1000 input tokens and the fresh call 1 more,
// so a delivered stale SessionUsageMsg shows 1000 and the fresh one 1001.
//
// The Request and Invoke cases carry the generation in the context, as RunTurn
// and RunTool put it there; RunTurn itself is not called (see the file
// comment).
func TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne(t *testing.T) {
	e := newCompactionEnv(t, 8192)
	oldStore, _, err := e.a.ensureStore()
	if err != nil {
		t.Fatalf("ensureStore: %v", err)
	}
	oldGen := e.a.currentGen()
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	newGen := e.a.currentGen()
	if newGen == oldGen {
		t.Fatalf("NewSession left the generation at %d", oldGen)
	}

	// Already cancelled, so Request returns after sending its message instead
	// of waiting for a decision.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	withGen := func(g uint64) context.Context { return context.WithValue(cancelled, turnGenKey{}, g) }

	const staleTokens, freshTokens = 1000, 1
	for _, g := range []struct {
		gen    uint64
		marker string
		tokens int
	}{{oldGen, "stale", staleTokens}, {newGen, "fresh", freshTokens}} {
		e.a.sendAsyncSequenceFenced(g.gen, []tea.Msg{tui.NoticeMsg{Text: g.marker + "-sequence"}})
		e.a.send(g.gen, tui.NoticeMsg{Text: g.marker + "-send"})
		if _, err := e.a.Request(withGen(g.gen), ApprovalRequest{
			Description: g.marker + "-approval", Operation: policy.OperationCommand, Argv: []string{"go", "test"},
		}); err != nil {
			t.Fatalf("Request: %v", err)
		}
		input, _ := json.Marshal(map[string]string{"path": "main.go"})
		(&toolExecutor{app: e.a}).Invoke(withGen(g.gen), agent.ToolCall{ID: "c-" + g.marker, Name: g.marker + "_tool", Input: input})
		e.a.flushStore(g.gen, oldStore)
		e.a.recordCompactionUsage(g.gen, &session.Compaction{Usage: agent.Usage{InputTokens: g.tokens}}, e.model)
	}

	// The asynchronous senders' controls: the fresh sequence notice, a flush
	// report and the fresh usage figures. Wait for all three, then give a
	// wrongly delivered stale message the same time to show up.
	asyncControls := func(msgs []any) bool {
		return hasNotice(msgs, "fresh-sequence") && flushReports(msgs) > 0 && hasSessionTokensIn(msgs, staleTokens+freshTokens)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !asyncControls(e.c.snapshot()) {
		if time.Now().After(deadline) {
			t.Fatalf("control: an asynchronous sender's current-generation message never arrived; got %#v", e.c.snapshot())
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)

	got := e.c.snapshot()
	if n := flushReports(got); n != 1 {
		t.Errorf("flushStore: %d session_store ErrorMsg(s) delivered, want 1 (the current generation's only)", n)
	}
	if hasSessionTokensIn(got, staleTokens) {
		t.Errorf("recordCompactionUsage: the previous generation's SessionUsageMsg (TokensIn %d) was delivered", staleTokens)
	}
	for _, want := range []struct {
		what string
		has  func(string) bool
	}{
		{"sendAsyncSequenceFenced", func(m string) bool { return hasNotice(got, m+"-sequence") }},
		{"send", func(m string) bool { return hasNotice(got, m+"-send") }},
		{"Request", func(m string) bool { return hasApproval(got, m+"-approval") }},
		{"Invoke (ToolStartedMsg)", func(m string) bool { return hasToolStarted(got, m+"_tool") }},
		{"Invoke (ToolCompletedMsg)", func(m string) bool { return hasToolCompleted(got, m+"_tool") }},
	} {
		if !want.has("fresh") {
			t.Errorf("%s: control failed, the current-generation message was not delivered", want.what)
		}
		if want.has("stale") {
			t.Errorf("%s: a message from the previous generation was delivered", want.what)
		}
	}
}

// flushReports counts the ErrorMsgs flushStore sends for a failed flush.
func flushReports(msgs []any) int {
	n := 0
	for _, m := range msgs {
		if e, ok := m.(tui.ErrorMsg); ok && e.Kind == "session_store" {
			n++
		}
	}
	return n
}

// hasSessionTokensIn reports whether a SessionUsageMsg with TokensIn in is
// among msgs.
func hasSessionTokensIn(msgs []any, in int) bool {
	for _, m := range msgs {
		if u, ok := m.(tui.SessionUsageMsg); ok && u.TokensIn == in {
			return true
		}
	}
	return false
}

func hasNotice(msgs []any, text string) bool {
	for _, m := range msgs {
		if n, ok := m.(tui.NoticeMsg); ok && n.Text == text {
			return true
		}
	}
	return false
}

func hasApproval(msgs []any, desc string) bool {
	for _, m := range msgs {
		if a, ok := m.(tui.ApprovalRequestedMsg); ok && a.Description == desc {
			return true
		}
	}
	return false
}

func hasToolStarted(msgs []any, name string) bool {
	for _, m := range msgs {
		if a, ok := m.(tui.ToolStartedMsg); ok && a.Name == name {
			return true
		}
	}
	return false
}

func hasToolCompleted(msgs []any, name string) bool {
	for _, m := range msgs {
		if a, ok := m.(tui.ToolCompletedMsg); ok && a.Name == name {
			return true
		}
	}
	return false
}

// unfencedSenders are the names that deliver to the TUI without a generation
// check: the unfenced sequence sender and its parts, the App helpers built on
// it, the sendFn override, and program.Send (matched as any method Send).
var unfencedSenders = map[string]bool{
	"sendAsyncSequence": true,
	"sendSequence":      true,
	"deliverOne":        true,
	"SendNotice":        true,
	"SendStatusWarning": true,
	"sendFn":            true,
	"Send":              true,
}

// fencedSenders take the generation as their first argument: the fenced
// senders and deliverIfCurrent, the check they are built on.
var fencedSenders = map[string]bool{
	"send":                    true,
	"sendAsyncSequenceFenced": true,
	"flushStore":              true,
	"recordCompactionUsage":   true,
	"deliverIfCurrent":        true,
}

// knownUnfenced lists the functions in package app (non-test files) that may
// call one of unfencedSenders, each with the reason it is there. Listing a
// function records it as known; it is not a claim that each of its messages is
// safe to deliver after /new.
var knownUnfenced = map[string]string{
	"App.sendAsyncSequence":       "the unfenced sequence sender itself",
	"App.deliverOne":              "the delivery step of both sequence senders",
	"App.SendNotice":              "unfenced helper; its callers in package app are checked here",
	"App.SendStatusWarning":       "unfenced helper; its callers in package app are checked here",
	"App.send":                    "fenced sender: its sendFn and program.Send calls run inside deliverIfCurrent",
	"App.sendAsyncSequenceFenced": "fenced sender: its sendSequence call delivers through deliverIfCurrent",
	"App.AcquireLock":             "startup lock warnings (cmd/kirsch/main.go), before any turn",
	"App.ResumeLastSession":       "startup session load (cmd/kirsch/main.go), before any turn",
	"App.CheckOnboarding":         "startup onboarding check (cmd/kirsch/main.go), before any turn",
	"App.Resolve":                 "known unfenced: the ApprovalResolvedMsg sent on the TUI's approval callback",
}

// fencedCallers lists the functions in package app (non-test files) that may
// call one of fencedSenders, with the forms of generation each may pass as the
// first argument: the identifier gen (the local read when the work started, or
// the parameter that carries it), turnGen() for callers that hold only a
// context, or currentGen() for callers on the Update goroutine.
var fencedCallers = map[string][]string{
	"App.Submit":                  {"gen", "currentGen()"},
	"App.ClearGrants":             {"currentGen()"},
	"App.RunTurn":                 {"gen"},
	"App.flushStore":              {"gen"},
	"App.recordCompactionUsage":   {"gen"},
	"App.ManualCompact":           {"gen"},
	"App.RunTool":                 {"gen"},
	"App.Request":                 {"turnGen()"},
	"toolExecutor.Invoke":         {"turnGen()"},
	"App.send":                    {"gen"},
	"App.sendAsyncSequenceFenced": {"gen"},
}

// calleeName returns the name a call is made through: the selector for
// x.f(...), the identifier for f(...), and "" otherwise.
func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}
	return ""
}

// genArgForm renders a fenced call's first argument as the identifier's name
// for an identifier, "name()" for a call through name, and "other" otherwise.
func genArgForm(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.CallExpr:
		if n := calleeName(x); n != "" {
			return n + "()"
		}
	}
	return "other"
}

// senderCall is one call to a sender name: where it is, the name, and (for a
// fencedSenders name) the form of its generation argument, "" when it has none.
type senderCall struct{ where, name, gen string }

// senderCalls is what scanSenders finds in one declaration.
type senderCalls struct {
	unfenced []senderCall
	fenced   []senderCall
}

// scanSenders walks n, function literals included, and appends to sc every
// call it makes to an unfencedSenders or fencedSenders name.
func scanSenders(fset *token.FileSet, n ast.Node, sc *senderCalls) {
	ast.Inspect(n, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := calleeName(call)
		pos := fset.Position(call.Pos())
		where := fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
		switch {
		case unfencedSenders[name]:
			sc.unfenced = append(sc.unfenced, senderCall{where: where, name: name})
		case fencedSenders[name]:
			fc := senderCall{where: where, name: name}
			if len(call.Args) > 0 {
				fc.gen = genArgForm(call.Args[0])
			}
			sc.fenced = append(sc.fenced, fc)
		}
		return true
	})
}

// declKey names a top-level declaration: "Recv.Name" for a method (pointer
// receivers unwrapped), the bare name for a function, and "(package level)"
// for anything else (a var holding a function literal, for example).
func declKey(d ast.Decl) string {
	fd, ok := d.(*ast.FuncDecl)
	if !ok {
		return "(package level)"
	}
	if fd.Recv == nil || len(fd.Recv.List) != 1 {
		return fd.Name.Name
	}
	recv := fd.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	if id, ok := recv.(*ast.Ident); ok {
		return id.Name + "." + fd.Name.Name
	}
	return "(receiver?)." + fd.Name.Name
}

// collectSenders scans every top-level declaration in files and returns the
// sender calls of each, keyed by declKey. Every function appears, including
// those that make no sender call.
func collectSenders(fset *token.FileSet, files []*ast.File) map[string]*senderCalls {
	found := map[string]*senderCalls{}
	for _, f := range files {
		for _, d := range f.Decls {
			k := declKey(d)
			if found[k] == nil {
				found[k] = &senderCalls{}
			}
			scanSenders(fset, d, found[k])
		}
	}
	return found
}

// senderPolicyProblems checks found against knownUnfenced and fencedCallers
// and returns one line per breach, sorted by declaration:
//
//   - a call to an unfencedSenders name from a function not in knownUnfenced;
//   - a call to a fencedSenders name from a function not in fencedCallers, or
//     with a generation form its entry does not allow;
//   - an entry in either list whose function is missing, or makes no call of
//     its kind, so neither list can keep a stale entry.
func senderPolicyProblems(found map[string]*senderCalls) []string {
	var problems []string
	keys := make([]string, 0, len(found))
	for k := range found {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sc := found[k]
		if knownUnfenced[k] == "" {
			for _, u := range sc.unfenced {
				problems = append(problems, fmt.Sprintf("%s: %s: calls unfenced %s and is not in knownUnfenced", k, u.where, u.name))
			}
		}
		allowed, listed := fencedCallers[k]
		for _, fc := range sc.fenced {
			if !listed {
				problems = append(problems, fmt.Sprintf("%s: %s: calls %s and is not in fencedCallers", k, fc.where, fc.name))
				continue
			}
			if !slices.Contains(allowed, fc.gen) {
				problems = append(problems, fmt.Sprintf("%s: %s: %s is given generation %q, want one of %v", k, fc.where, fc.name, fc.gen, allowed))
			}
		}
	}
	var stale []string
	for k := range knownUnfenced {
		if sc := found[k]; sc == nil || len(sc.unfenced) == 0 {
			stale = append(stale, fmt.Sprintf("control: knownUnfenced entry %s is missing or calls no unfenced sender", k))
		}
	}
	for k := range fencedCallers {
		if sc := found[k]; sc == nil || len(sc.fenced) == 0 {
			stale = append(stale, fmt.Sprintf("control: fencedCallers entry %s is missing or calls no fenced sender", k))
		}
	}
	sort.Strings(stale)
	return append(problems, stale...)
}

// parsePackageApp parses every non-test .go file in the current directory
// (package app, when go test runs here).
func parsePackageApp(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, f)
	}
	return fset, files
}

// TestTurnPathsUseOnlyFencedSenders parses every non-test file of package app
// and runs none of the code. It is the pin for fenced call sites that no
// behavioural test reaches, because a turn cannot be held between building a
// message and delivering it from outside. It is closed-world over the package:
// every top-level declaration is scanned, a call to an unfencedSenders name is
// allowed only from a function in knownUnfenced, and a call to a fencedSenders
// name only from a function in fencedCallers, passing a generation form that
// function's entry allows. Calls inside function literals (RunTurn's goroutine
// and emit, RunTool's goroutine and progress sink) count as calls of the
// enclosing declaration.
//
// Limits. Calls are matched by name (calleeName): a sender reached under
// another name, through a method value or a func variable, is not seen, and
// any method called Send counts as program.Send. Only package app is scanned;
// callers in other packages of exported unfenced helpers (cmd/kirsch/main.go
// calls SendNotice) are not checked. A function in knownUnfenced may send any
// message unfenced; the list records known senders, it does not review them.
// Submit's currentGen() form is allowed for all its fenced calls, though only
// its opMu.TryLock refusal uses it.
//
// Controls: an entry in either list whose function is missing or makes no call
// of its kind is reported, so a rename or a moved call cannot leave a stale
// entry. The "control" subtest runs the same check on a small source text in
// which each kind of breach appears once: an unlisted function calling
// sendAsyncSequence, an unlisted function calling send, and a RunTool passing
// currentGen() to send where its entry allows only gen. Each must be reported.
// In the same text an allowlisted AcquireLock calls SendStatusWarning and must
// not be, which shows the check admits as well as fires.
func TestTurnPathsUseOnlyFencedSenders(t *testing.T) {
	t.Run("package app", func(t *testing.T) {
		fset, files := parsePackageApp(t)
		found := collectSenders(fset, files)
		if found["App.RunTurn"] == nil || len(found["App.RunTurn"].fenced) == 0 {
			t.Fatalf("control: App.RunTurn not found or makes no fenced call; the scan read nothing")
		}
		for _, p := range senderPolicyProblems(found) {
			t.Error(p)
		}
	})

	t.Run("control", func(t *testing.T) {
		const src = `package app

func (a *App) AcquireLock() error { a.SendStatusWarning("w"); return nil }
func (a *App) newHelper() { a.sendAsyncSequence(nil) }
func (a *App) unlisted(gen uint64) { a.send(gen, nil) }
func (a *App) RunTool() { a.send(a.currentGen(), nil) }
`
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "control.go", src, 0)
		if err != nil {
			t.Fatalf("parse control source: %v", err)
		}
		got := strings.Join(senderPolicyProblems(collectSenders(fset, []*ast.File{f})), "\n")
		for _, want := range []string{
			"App.newHelper: control.go:4: calls unfenced sendAsyncSequence and is not in knownUnfenced",
			"App.unlisted: control.go:5: calls send and is not in fencedCallers",
			`App.RunTool: control.go:6: send is given generation "currentGen()", want one of [gen]`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("control: problems lack %q; got:\n%s", want, got)
			}
		}
		if strings.Contains(got, "App.AcquireLock: ") {
			t.Errorf("control: the allowlisted AcquireLock was reported; got:\n%s", got)
		}
	})
}
