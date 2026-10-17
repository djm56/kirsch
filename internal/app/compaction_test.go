package app

// Compaction tests for m4-d7.
//
// Every test here drives App.Submit, the production entry point, with a
// provider.Fake behind a.providerBuilder and a real session.Store on a
// temporary directory. Nothing calls RunTurn or runCompaction directly.
//
// The model's context window is made small through the a.modelInfo seam
// (production uses provider.LookupModel), so the 75% threshold is reached with
// a few thousand characters instead of hundreds of kilobytes. The figures are:
//
//	window 30000, reserve 4096 -> budget 25904, threshold 19428 tokens
//
// TUI messages reach the collector through one goroutine per
// sendAsyncSequence call, so their relative order is not defined. These tests
// assert which messages arrive, never the order between them. Order is
// asserted only on the JSONL log, which one writer goroutine produces.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
)

const (
	compactionWindow    = 30000
	compactionBudget    = compactionWindow - outputReserve
	compactionThreshold = compactionBudget * 75 / 100
	compactionUserText  = "please continue"
	compactionSummary   = "Objective: finish the refactor.\nFiles inspected: a.go\nOpen questions: none"
)

// compactionEnv is one App wired to a fake provider and a real store.
type compactionEnv struct {
	t     *testing.T
	a     *App
	c     *collector
	fake  *provider.Fake
	model string
	dir   string
	// logAtExit is the log's event types read the moment the turn goroutine
	// had exited, before settle's quiet period. The store's own 500ms flush
	// tick cannot have fired by then, so a write the goroutine did not flush
	// itself is absent.
	logAtExit []string
}

// newCompactionEnv builds the App. maxOutput is the model's MaxOutput as the
// modelInfo seam reports it. script is the fake provider's reply sequence.
func newCompactionEnv(t *testing.T, maxOutput int, script ...provider.Turn) *compactionEnv {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(dir, "sessions")

	c := newCollector(-1) // never closes done; tests poll snapshot()
	a := testAppWithConfig(t, cfg, c)
	t.Cleanup(a.Close)

	a.modelInfo = func(id string) provider.ModelInfo {
		return provider.ModelInfo{ID: id, Known: true, ContextWindow: compactionWindow, MaxOutput: maxOutput}
	}
	fake := provider.NewFake(script...)
	a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return fake, nil
	}

	e := &compactionEnv{
		t: t, a: a, c: c, fake: fake, dir: dir,
		model: cfg.Provider.Endpoints[cfg.Provider.Default].Model,
	}
	a.StartSession(getenvKey)
	return e
}

func getenvKey(string) string { return "fake-key" }

// seed fills a.conv and the store with turns user/assistant pairs sized so the
// estimate of the next request (seeded history plus compactionUserText) lands
// on targetTokens. It fails the test if the estimate misses by more than 40.
func (e *compactionEnv) seed(turns, targetTokens int) {
	e.t.Helper()
	store, _, err := e.a.ensureStore()
	if err != nil {
		e.t.Fatalf("ensureStore: %v", err)
	}
	tools := &toolExecutor{app: e.a}
	overhead := estimateTokens(e.a.systemPrompt, "", &agent.Conversation{}, tools)
	chars := ((targetTokens-overhead)*4 - len(compactionUserText)) / (2 * turns)
	if chars < 16 {
		e.t.Fatalf("target %d tokens leaves %d chars per message (overhead %d)", targetTokens, chars, overhead)
	}
	pad := func(label string) string {
		return label + strings.Repeat("x", chars-len(label))
	}
	for i := 0; i < turns; i++ {
		for _, m := range []agent.Message{
			{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: pad(fmt.Sprintf("u%02d-", i))}}},
			{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: pad(fmt.Sprintf("a%02d-", i))}}},
		} {
			e.a.conv.Messages = append(e.a.conv.Messages, cloneMsgs([]agent.Message{m})...)
			store.Record(agent.RecordAppend, m)
		}
	}
	if err := store.Flush(); err != nil {
		e.t.Fatalf("Flush seed: %v", err)
	}
	got := e.a.estimateRequest(compactionUserText, e.a.conv, tools)
	if got < targetTokens-40 || got > targetTokens+40 {
		e.t.Fatalf("seeded estimate = %d tokens, want %d +-40", got, targetTokens)
	}
}

// submit runs Submit and returns the messages once the turn goroutine has
// exited, the terminal message has been delivered and the collector has been
// quiet for 100ms. The terminal message is any of TurnCompleteMsg,
// TurnErrorMsg and TurnCancelledMsg.
func (e *compactionEnv) submit() []any {
	e.t.Helper()
	e.a.Submit(compactionUserText, getenvKey)
	return e.settle()
}

// settle waits for the terminal message, the turn goroutine and a quiet
// collector. Late arrivals could only turn an absence assertion into a false
// pass, never a presence assertion into a false failure.
func (e *compactionEnv) settle() []any {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		msgs := e.c.snapshot()
		if findMsgType[tui.TurnCompleteMsg](msgs) || findMsgType[tui.TurnErrorMsg](msgs) || findMsgType[tui.TurnCancelledMsg](msgs) {
			break
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("no terminal message after 5s; got %#v", msgs)
		}
		time.Sleep(5 * time.Millisecond)
	}
	e.a.inFlight.Wait()
	e.logAtExit = e.logTypes()
	last, quietSince := e.c.count(), time.Now()
	for time.Since(quietSince) < 100*time.Millisecond {
		time.Sleep(10 * time.Millisecond)
		if n := e.c.count(); n != last {
			last, quietSince = n, time.Now()
		}
	}
	return e.c.snapshot()
}

// logPath returns the single session JSONL file under the storage dir.
func (e *compactionEnv) logPath() string {
	e.t.Helper()
	var found []string
	_ = filepath.WalkDir(filepath.Join(e.dir, "sessions"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".jsonl") {
			found = append(found, p)
		}
		return nil
	})
	if len(found) != 1 {
		e.t.Fatalf("session logs = %v, want exactly one", found)
	}
	return found[0]
}

// logTypes returns the "type" of every line in the session log, in file order.
func (e *compactionEnv) logTypes() []string {
	e.t.Helper()
	data, err := os.ReadFile(e.logPath())
	if err != nil {
		e.t.Fatalf("read log: %v", err)
	}
	var types []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &base); err != nil {
			e.t.Fatalf("log line %q: %v", line, err)
		}
		types = append(types, base.Type)
	}
	return types
}

func indexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}

func countOf(xs []string, want string) int {
	n := 0
	for _, x := range xs {
		if x == want {
			n++
		}
	}
	return n
}

// cloneMsgs deep-copies text-only messages.
func cloneMsgs(in []agent.Message) []agent.Message {
	out := make([]agent.Message, len(in))
	for i, m := range in {
		out[i] = agent.Message{Role: m.Role, Content: append([]agent.Block(nil), m.Content...)}
	}
	return out
}

func allMsgs[T any](msgs []any) []T {
	var out []T
	for _, m := range msgs {
		if v, ok := m.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

func msgText(m provider.Message) string {
	var sb strings.Builder
	for _, b := range m.Content {
		sb.WriteString(b.Text)
	}
	return sb.String()
}

// turnReply is the fake's reply to the real turn.
var turnReply = provider.Turn{Text: "done"}

// TestSubmitBelowThresholdSendsOneRequestAndNoCompaction drives Submit with an
// estimate just under the 75% threshold. It is the mirror of the compaction
// tests below, which use the same eight-turn history at a larger size.
func TestSubmitBelowThresholdSendsOneRequestAndNoCompaction(t *testing.T) {
	e := newCompactionEnv(t, 8192, turnReply)
	e.seed(8, compactionThreshold-300)

	msgs := e.submit()

	if n := len(e.fake.Requests()); n != 1 {
		t.Errorf("below threshold: provider requests = %d, want exactly 1 (the turn)", n)
	}
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Errorf("below threshold: no TurnCompleteMsg in %#v", msgs)
	}
	if findMsgType[tui.CompactingMsg](msgs) {
		t.Error("below threshold: CompactingMsg was sent")
	}
	if findMsgType[tui.SessionCompactedMsg](msgs) {
		t.Error("below threshold: SessionCompactedMsg was sent")
	}
	if got := countOf(e.logTypes(), "compaction.applied"); got != 0 {
		t.Errorf("below threshold: %d compaction.applied events in the log, want 0", got)
	}
}

// TestSubmitAboveThresholdCompactsThenRunsTheTurn drives Submit with the
// eight-turn history just over the threshold and under budget. The table also
// covers the summary request's MaxTokens at both sides of the output reserve.
func TestSubmitAboveThresholdCompactsThenRunsTheTurn(t *testing.T) {
	for _, tc := range []struct {
		name          string
		maxOutput     int
		wantMaxTokens int
	}{
		{"MaxOutput above reserve is capped at the reserve", 8192, outputReserve},
		{"MaxOutput below reserve is kept", 2048, 2048},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newCompactionEnv(t, tc.maxOutput, provider.Turn{Text: compactionSummary}, turnReply)
			e.seed(8, compactionThreshold+2500)
			snapshot := cloneMsgs(e.a.conv.Messages)
			tools := &toolExecutor{app: e.a}
			preEstimate := e.a.estimateRequest(compactionUserText, e.a.conv, tools)

			msgs := e.submit()

			reqs := e.fake.Requests()
			if len(reqs) != 2 {
				t.Fatalf("provider requests = %d, want 2 (summary, turn)", len(reqs))
			}
			sum, turn := reqs[0], reqs[1]

			// Request 1 is the summary request.
			if len(sum.System) == 0 || !strings.Contains(sum.System[0].Text, "Objective") {
				t.Errorf("request 1 is not the summary request: system = %+v", sum.System)
			}
			if sum.Model != e.model {
				t.Errorf("summary request model = %q, want the session model %q", sum.Model, e.model)
			}
			if sum.MaxTokens != tc.wantMaxTokens {
				t.Errorf("summary request MaxTokens = %d, want %d", sum.MaxTokens, tc.wantMaxTokens)
			}

			// Request 2 is the turn: the compacted conv plus the user text.
			if turn.Model != e.model {
				t.Errorf("turn request model = %q, want %q", turn.Model, e.model)
			}
			for _, sb := range turn.System {
				if strings.Contains(sb.Text, "summarizing a conversation") {
					t.Error("request 2 carries the summary system prompt, so it is not the turn")
				}
			}
			post := e.a.conv.Messages
			if len(post) < 3 {
				t.Fatalf("conv has %d messages after the turn", len(post))
			}
			compacted := post[:len(post)-2] // everything before the user text and the reply
			wantMsgs := make([]provider.Message, 0, len(compacted)+1)
			for _, m := range append(cloneMsgs(compacted), agent.Message{
				Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: compactionUserText}},
			}) {
				wantMsgs = append(wantMsgs, provider.Message{Role: provider.Role(m.Role), Content: agentBlocksToProvider(m.Content)})
			}
			if !reflect.DeepEqual(turn.Messages, wantMsgs) {
				t.Errorf("turn request messages differ from compacted conv + user text:\n got %d messages\nwant %d messages", len(turn.Messages), len(wantMsgs))
			}
			// The compacted shape, derived from the snapshot and not from conv:
			// message 0, the summary, the last four turns, then the user text.
			if got, want := len(turn.Messages), 1+1+8+1; got != want {
				t.Fatalf("turn request has %d messages, want %d", got, want)
			}
			if got, want := msgText(turn.Messages[0]), snapshot[0].Content[0].Text; got != want {
				t.Error("turn request message 0 is not the original first message")
			}
			if !strings.Contains(msgText(turn.Messages[1]), "Objective: finish the refactor.") || turn.Messages[1].Role != provider.RoleAssistant {
				t.Errorf("turn request message 1 is not the summary: %q", msgText(turn.Messages[1]))
			}
			if got, want := msgText(turn.Messages[2]), snapshot[len(snapshot)-8].Content[0].Text; got != want {
				t.Error("turn request message 2 is not the first message of the 4th-from-last turn")
			}
			if got := msgText(turn.Messages[len(turn.Messages)-1]); got != compactionUserText {
				t.Errorf("turn request last message = %q, want the user text", got)
			}

			// Store order: compaction.applied before turn.started.
			types := e.logTypes()
			ci, ti := indexOf(types, "compaction.applied"), indexOf(types, "turn.started")
			if ci < 0 {
				t.Fatalf("no compaction.applied in the log: %v", types)
			}
			if ti < 0 || ci > ti {
				t.Errorf("compaction.applied at %d, turn.started at %d; want compaction first (%v)", ci, ti, types)
			}

			// TUI messages.
			if !findMsgType[tui.CompactingMsg](msgs) {
				t.Error("no CompactingMsg")
			}
			if !findMsgType[tui.SessionCompactedMsg](msgs) {
				t.Error("no SessionCompactedMsg")
			}
			if !findMsgType[tui.TurnCompleteMsg](msgs) {
				t.Error("no TurnCompleteMsg")
			}
			wantPost := e.a.estimateRequest(compactionUserText, &agent.Conversation{Messages: compacted}, tools)
			budgets := allMsgs[tui.BudgetMsg](msgs)
			if len(budgets) != 2 {
				t.Fatalf("BudgetMsg count = %d, want 2 (before and after compaction): %+v", len(budgets), budgets)
			}
			var sawPre, sawPost bool
			for _, b := range budgets {
				sawPre = sawPre || b.EstimatedTokens == preEstimate
				sawPost = sawPost || b.EstimatedTokens == wantPost
			}
			if !sawPre {
				t.Errorf("no BudgetMsg with the pre-compaction estimate %d: %+v", preEstimate, budgets)
			}
			if !sawPost || wantPost >= preEstimate {
				t.Errorf("no BudgetMsg with the post-compaction estimate %d (pre %d): %+v", wantPost, preEstimate, budgets)
			}
		})
	}
}

// TestSubmitCancelDuringCompactionLeavesSessionUntouched drives Submit with a
// summary call that blocks until cancelled, then calls CancelTurn.
func TestSubmitCancelDuringCompactionLeavesSessionUntouched(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{BlockUntilCancel: true}, turnReply)
	e.seed(8, compactionThreshold+2500)
	snapshot := cloneMsgs(e.a.conv.Messages)
	before := e.logTypes()
	turnsBefore := e.a.turnCount.Load()

	e.a.Submit(compactionUserText, getenvKey)
	deadline := time.Now().Add(5 * time.Second)
	for len(e.fake.Requests()) < 1 {
		if time.Now().After(deadline) {
			t.Fatal("summary request never reached the provider")
		}
		time.Sleep(5 * time.Millisecond)
	}
	e.a.CancelTurn()
	msgs := e.settle()

	if !findMsgType[tui.CompactingMsg](msgs) {
		t.Error("cancel: no CompactingMsg, so the cancel did not land during compaction")
	}
	if !findMsgType[tui.TurnCancelledMsg](msgs) {
		t.Errorf("cancel: no TurnCancelledMsg in %#v", msgs)
	}
	if findMsgType[tui.TurnErrorMsg](msgs) || findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Errorf("cancel: turn ended another way: %#v", msgs)
	}
	if n := len(e.fake.Requests()); n != 1 {
		t.Errorf("cancel: provider requests = %d, want 1 (summary only, no turn)", n)
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("cancel: conv changed")
	}
	if after := e.logTypes(); !reflect.DeepEqual(after, before) {
		t.Errorf("cancel: log changed from %v to %v", before, after)
	}
	if got := e.a.turnCount.Load(); got != turnsBefore {
		t.Errorf("cancel: turnCount %d -> %d, want unchanged", turnsBefore, got)
	}
}

// TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies drives Submit with a
// summary call that succeeds, and cancels from afterCompactHook, which runs
// after the summary call returned and before the swap. That is the window
// session.Compact's own cancellation checks cannot cover.
func TestSubmitCancelAfterSummaryBeforeSwapNeverHalfApplies(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary}, turnReply)
	e.seed(8, compactionThreshold+2500)
	snapshot := cloneMsgs(e.a.conv.Messages)
	before := e.logTypes()
	turnsBefore := e.a.turnCount.Load()
	e.a.afterCompactHook = e.a.CancelTurn

	msgs := e.submit()

	if n := len(e.fake.Requests()); n != 1 {
		t.Fatalf("late cancel: provider requests = %d, want 1 (the summary completed)", n)
	}
	if !findMsgType[tui.TurnCancelledMsg](msgs) {
		t.Errorf("late cancel: no TurnCancelledMsg in %#v", msgs)
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("late cancel: conv was replaced although the turn was cancelled")
	}
	if after := e.logTypes(); !reflect.DeepEqual(after, before) {
		t.Errorf("late cancel: log changed from %v to %v", before, after)
	}
	if findMsgType[tui.SessionCompactedMsg](msgs) {
		t.Error("late cancel: SessionCompactedMsg sent for a compaction that was not applied")
	}
	if got := e.a.turnCount.Load(); got != turnsBefore {
		t.Errorf("late cancel: turnCount %d -> %d, want unchanged", turnsBefore, got)
	}
}

// TestSubmitCompactionProviderErrorSurfacesContextOverflow drives Submit with
// a summary call that fails.
func TestSubmitCompactionProviderErrorSurfacesContextOverflow(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Err: errors.New("provider exploded")}, turnReply)
	e.seed(8, compactionThreshold+2500)
	snapshot := cloneMsgs(e.a.conv.Messages)

	msgs := e.submit()

	errMsg := findMsg[tui.TurnErrorMsg](msgs)
	if errMsg == nil {
		t.Fatalf("provider error: no TurnErrorMsg in %#v", msgs)
	}
	if errMsg.Kind != "context_overflow" {
		t.Errorf("provider error: kind = %q, want context_overflow", errMsg.Kind)
	}
	if !strings.Contains(errMsg.Detail, "/new") || !strings.Contains(errMsg.Detail, "provider exploded") {
		t.Errorf("provider error: detail %q does not name /new and the cause", errMsg.Detail)
	}
	if n := len(e.fake.Requests()); n != 1 {
		t.Errorf("provider error: provider requests = %d, want 1 (summary only, no turn)", n)
	}
	if findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Error("provider error: the turn ran anyway")
	}
	if !reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("provider error: conv changed")
	}
	types := e.logTypes()
	if indexOf(types, "compaction.applied") >= 0 || indexOf(types, "turn.started") >= 0 {
		t.Errorf("provider error: log has compaction or turn events: %v", types)
	}
}

// TestSubmitFewerThanSixTurnsOverBudgetSurfacesContextOverflow drives Submit
// with a four-turn history that is over budget, so there is nothing to
// compact and nothing that fits. The mirror case, over the threshold but under
// budget, must run the turn.
func TestSubmitFewerThanSixTurnsOverBudgetSurfacesContextOverflow(t *testing.T) {
	t.Run("over budget refuses", func(t *testing.T) {
		e := newCompactionEnv(t, 8192, turnReply)
		e.seed(4, compactionBudget+2000)

		msgs := e.submit()

		errMsg := findMsg[tui.TurnErrorMsg](msgs)
		if errMsg == nil || errMsg.Kind != "context_overflow" {
			t.Fatalf("over budget, 4 turns: want context_overflow, got %#v", msgs)
		}
		if !strings.Contains(errMsg.Detail, "/new") {
			t.Errorf("over budget, 4 turns: detail %q does not name /new", errMsg.Detail)
		}
		if n := len(e.fake.Requests()); n != 0 {
			t.Errorf("over budget, 4 turns: provider requests = %d, want 0", n)
		}
		if findMsgType[tui.CompactingMsg](msgs) {
			t.Error("over budget, 4 turns: CompactingMsg flashed with nothing to compact")
		}
	})
	t.Run("over threshold under budget proceeds", func(t *testing.T) {
		e := newCompactionEnv(t, 8192, turnReply)
		e.seed(4, compactionThreshold+2500)

		msgs := e.submit()

		if !findMsgType[tui.TurnCompleteMsg](msgs) {
			t.Fatalf("under budget, 4 turns: turn did not complete: %#v", msgs)
		}
		if findMsgType[tui.TurnErrorMsg](msgs) {
			t.Errorf("under budget, 4 turns: unexpected TurnErrorMsg: %#v", msgs)
		}
		if n := len(e.fake.Requests()); n != 1 {
			t.Errorf("under budget, 4 turns: provider requests = %d, want 1 (the turn)", n)
		}
		if findMsgType[tui.CompactingMsg](msgs) {
			t.Error("under budget, 4 turns: CompactingMsg flashed with nothing to compact")
		}
		if indexOf(e.logTypes(), "compaction.applied") >= 0 {
			t.Error("under budget, 4 turns: compaction.applied recorded")
		}
	})
}

// TestSubmitOverBudgetCompactsInsteadOfRefusing drives Submit with an
// eight-turn history that is over budget before compaction. Compaction must be
// tried before overflow is declared, and when it brings the request under
// budget the turn runs.
func TestSubmitOverBudgetCompactsInsteadOfRefusing(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary}, turnReply)
	e.seed(8, compactionBudget+2000)

	msgs := e.submit()

	if findMsgType[tui.TurnErrorMsg](msgs) {
		t.Fatalf("over budget then compacted: TurnErrorMsg in %#v", msgs)
	}
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Errorf("over budget then compacted: turn did not complete: %#v", msgs)
	}
	if n := len(e.fake.Requests()); n != 2 {
		t.Errorf("over budget then compacted: provider requests = %d, want 2", n)
	}
	if !findMsgType[tui.SessionCompactedMsg](msgs) {
		t.Error("over budget then compacted: no SessionCompactedMsg")
	}
	if got := countOf(e.logTypes(), "compaction.applied"); got != 1 {
		t.Errorf("over budget then compacted: %d compaction.applied events, want 1", got)
	}
}

// TestSubmitStillOverBudgetAfterCompactionSurfacesContextOverflow drives
// Submit with an eight-turn history so large that the part compaction keeps
// (message 0 and the last four turns) alone exceeds the budget. The header
// must still be told about the compaction that was persisted, the turn must
// not start, and the compaction event must be flushed by the early return.
// The summary is short on purpose: a long line would overflow the store's
// write buffer and reach the file without any flush.
func TestSubmitStillOverBudgetAfterCompactionSurfacesContextOverflow(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary}, turnReply)
	e.seed(8, compactionBudget*2+3000)
	snapshot := cloneMsgs(e.a.conv.Messages)

	msgs := e.submit()

	errMsg := findMsg[tui.TurnErrorMsg](msgs)
	if errMsg == nil || errMsg.Kind != "context_overflow" {
		t.Fatalf("still oversized: want context_overflow, got %#v", msgs)
	}
	if !strings.Contains(errMsg.Message, "compacted request still exceeds") || !strings.Contains(errMsg.Detail, "/new") {
		t.Errorf("still oversized: message %q, detail %q", errMsg.Message, errMsg.Detail)
	}
	if n := len(e.fake.Requests()); n != 1 {
		t.Errorf("still oversized: provider requests = %d, want 1 (summary only, no turn)", n)
	}
	if !findMsgType[tui.SessionCompactedMsg](msgs) {
		t.Error("still oversized: header was not told about the persisted compaction")
	}
	if got := len(allMsgs[tui.BudgetMsg](msgs)); got != 1 {
		t.Errorf("still oversized: BudgetMsg count = %d, want 1 (the pre-compaction one only)", got)
	}
	types := e.logAtExit // read without Close: relies on the early-return Flush
	if indexOf(types, "compaction.applied") < 0 {
		t.Errorf("still oversized: compaction.applied not in the flushed log: %v", types)
	}
	if indexOf(types, "turn.started") >= 0 {
		t.Errorf("still oversized: turn.started recorded: %v", types)
	}
	if reflect.DeepEqual(e.a.conv.Messages, snapshot) {
		t.Error("still oversized: conv was not replaced although the compaction was recorded")
	}
}

// TestSubmitCompactionSurvivesResume drives a compaction and a turn through
// Submit, closes the app, and checks that session.Resume on the log rebuilds
// exactly the conversation the app holds.
func TestSubmitCompactionSurvivesResume(t *testing.T) {
	e := newCompactionEnv(t, 8192, provider.Turn{Text: compactionSummary}, turnReply)
	e.seed(8, compactionThreshold+2500)
	snapshotLen := len(e.a.conv.Messages)

	msgs := e.submit()
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Fatalf("resume: turn did not complete: %#v", msgs)
	}
	if len(e.a.conv.Messages) >= snapshotLen {
		t.Fatalf("resume: conv has %d messages, not fewer than the %d seeded; nothing was compacted", len(e.a.conv.Messages), snapshotLen)
	}
	want := cloneMsgs(e.a.conv.Messages)
	e.a.Close()

	got, _, err := session.Resume(e.logPath())
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !reflect.DeepEqual(got.Messages, want) {
		t.Errorf("resumed conversation differs from the live one: %d messages vs %d", len(got.Messages), len(want))
	}
}
