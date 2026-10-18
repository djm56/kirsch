package app

// Tests for the session usage totals and the unknown-model notice of m4-d9.
//
// Entry points, so each name can be read against what it really calls:
//   - Tests named ...Submit... or ...Turn... drive App.Submit with a
//     provider.Fake behind a.providerBuilder, so the usage travels through the
//     real providerModel, the turn adapter and recordUsage.
//   - The compaction tests drive App.Submit over a history above the threshold
//     (automatic) or App.ManualCompact (manual).
//   - TestSessionCostPricesEachCallAtItsOwnModel calls recordUsage and
//     buildSessionUsageMsg directly: a model switch mid-session has no
//     production entry point short of editing config, and the property is about
//     those two functions.
//   - The notice tests feed the messages the app really sent into a real
//     tui.Model through Update, and count the notices in its rendered frame.
//
// TUI messages reach the collector through one goroutine per sendAsyncSequence
// call, so the tests never assert an order between them. SessionUsageMsg
// totals only grow, so they are ordered by TokensIn.

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
)

const unknownNoticeText = "cost display unavailable, conservative budget in use"

// usageEnv is one App on a custom endpoint whose model the test names.
type usageEnv struct {
	t    *testing.T
	a    *App
	c    *collector
	fake *provider.Fake
}

// newUsageEnv builds an App whose default endpoint uses model. The real
// provider.LookupModel is in force, so an id outside the table is really unknown.
func newUsageEnv(t *testing.T, model string, script ...provider.Turn) *usageEnv {
	t.Helper()
	cfg := config.Defaults()
	cfg.Provider.Default = "usage-test"
	cfg.Provider.Endpoints["usage-test"] = config.EndpointConfig{
		BaseURL:   "https://example.com",
		Auth:      "x-api-key",
		APIKeyEnv: "USAGE_TEST_KEY",
		Model:     model,
	}
	cfg.Session.StorageDir = filepath.Join(t.TempDir(), "sessions")

	c := newCollector(-1) // never closes done; tests poll snapshot()
	a := testAppWithConfig(t, cfg, c)
	t.Cleanup(a.Close)
	fake := provider.NewFake(script...)
	a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return fake, nil
	}
	a.StartSession(getenvKey)
	return &usageEnv{t: t, a: a, c: c, fake: fake}
}

// countMsgs returns how many messages of type T the slice holds.
func countMsgs[T any](msgs []any) int {
	n := 0
	for _, m := range msgs {
		if _, ok := m.(T); ok {
			n++
		}
	}
	return n
}

// turn submits text and returns once its TurnCompleteMsg has arrived and the
// turn goroutine has exited.
func (e *usageEnv) turn(text string) {
	e.t.Helper()
	before := countMsgs[tui.TurnCompleteMsg](e.c.snapshot())
	e.a.Submit(text, getenvKey)
	deadline := time.Now().Add(5 * time.Second)
	for countMsgs[tui.TurnCompleteMsg](e.c.snapshot()) <= before {
		if time.Now().After(deadline) {
			e.t.Fatalf("no TurnCompleteMsg after 5s; got %#v", e.c.snapshot())
		}
		time.Sleep(5 * time.Millisecond)
	}
	e.a.inFlight.Wait()
}

// sessionUsage waits for want SessionUsageMsg, then confirms no further one
// arrives in the next 100ms, and returns them in ascending TokensIn order. The
// arrival of want is the positive control for the absence of more.
func sessionUsage(t *testing.T, c *collector, want int) []tui.SessionUsageMsg {
	t.Helper()
	collect := func() []tui.SessionUsageMsg {
		var out []tui.SessionUsageMsg
		for _, m := range c.snapshot() {
			if u, ok := m.(tui.SessionUsageMsg); ok {
				out = append(out, u)
			}
		}
		return out
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(collect()) < want {
		if time.Now().After(deadline) {
			t.Fatalf("got %d SessionUsageMsg after 5s, want %d: %#v", len(collect()), want, collect())
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	got := collect()
	if len(got) != want {
		t.Fatalf("got %d SessionUsageMsg, want exactly %d: %#v", len(got), want, got)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].TokensIn < got[j].TokensIn })
	return got
}

// wantTotals fails the test unless got carries exactly the totals of u and the
// given cost, state and model-known flag.
func wantTotals(t *testing.T, what string, got tui.SessionUsageMsg, u provider.Usage, cost float64, state string, known bool) {
	t.Helper()
	if got.TokensIn != u.InputTokens || got.TokensOut != u.OutputTokens ||
		got.CacheRead != u.CacheReadTokens || got.CacheWrite != u.CacheWriteTokens {
		t.Errorf("%s: in/out/read/write = %d/%d/%d/%d, want %d/%d/%d/%d", what,
			got.TokensIn, got.TokensOut, got.CacheRead, got.CacheWrite,
			u.InputTokens, u.OutputTokens, u.CacheReadTokens, u.CacheWriteTokens)
	}
	if math.Abs(got.CostUSD-cost) > 1e-12 {
		t.Errorf("%s: CostUSD = %v, want %v", what, got.CostUSD, cost)
	}
	if got.CostState != state {
		t.Errorf("%s: CostState = %q, want %q", what, got.CostState, state)
	}
	if got.ModelKnown != known {
		t.Errorf("%s: ModelKnown = %v, want %v", what, got.ModelKnown, known)
	}
}

var (
	usageTurn1 = provider.Usage{InputTokens: 4000, OutputTokens: 900, CacheReadTokens: 60, CacheWriteTokens: 11}
	usageTurn2 = provider.Usage{InputTokens: 700, OutputTokens: 55, CacheReadTokens: 9, CacheWriteTokens: 3}
)

// TestSubmitSendsExactSessionTotalsPerTurn runs two turns on claude-sonnet-5-5
// through Submit. Each turn must produce one SessionUsageMsg whose totals are
// the running sums, with cost summed from the per-call prices.
func TestSubmitSendsExactSessionTotalsPerTurn(t *testing.T) {
	e := newUsageEnv(t, "claude-sonnet-5-5",
		provider.Turn{Text: "one", Usage: usageTurn1},
		provider.Turn{Text: "two", Usage: usageTurn2})

	e.turn("first")
	e.turn("second")

	got := sessionUsage(t, e.c, 2)
	wantTotals(t, "after turn 1", got[0], usageTurn1, usageCost(usageTurn1), "per_token", true)
	wantTotals(t, "after turn 2", got[1], sumUsage(usageTurn1, usageTurn2),
		usageCost(usageTurn1)+usageCost(usageTurn2), "per_token", true)
}

// TestSessionCostPricesEachCallAtItsOwnModel calls recordUsage twice with
// different models, then buildSessionUsageMsg for the first. The cost must be
// the sum of the two per-call prices (the figure usage.json accumulates), not
// the summed tokens priced at the current model's rates.
func TestSessionCostPricesEachCallAtItsOwnModel(t *testing.T) {
	e := newUsageEnv(t, "claude-sonnet-5-5")
	u1 := agent.Usage{InputTokens: 1000, OutputTokens: 200, CacheReadTokens: 30, CacheWriteTokens: 7}
	u2 := agent.Usage{InputTokens: 2000, OutputTokens: 100, CacheReadTokens: 10, CacheWriteTokens: 5}

	e.a.recordUsage(u1, "claude-sonnet-5-5")
	e.a.recordUsage(u2, "claude-opus-5-5")
	got := e.a.buildSessionUsageMsg("claude-sonnet-5-5")

	// Written out from the table's prices: sonnet 2.0/10.0/0.2/2.5, opus 4.0/20.0/0.2/5.0.
	sonnet1 := (1000*2.0 + 200*10.0 + 30*0.2 + 7*2.5) / 1e6
	opus2 := (2000*4.0 + 100*20.0 + 10*0.2 + 5*5.0) / 1e6
	repriced := (3000*2.0 + 300*10.0 + 40*0.2 + 12*2.5) / 1e6
	if math.Abs(got.CostUSD-(sonnet1+opus2)) > 1e-12 {
		t.Errorf("CostUSD = %v, want per-call sum %v", got.CostUSD, sonnet1+opus2)
	}
	if math.Abs(sonnet1+opus2-repriced) < 1e-9 {
		t.Fatal("control: the per-call sum equals the re-priced figure, so this test cannot tell them apart")
	}
}

// TestSubmitCompactionSendsSessionTotals drives Submit over a history above the
// threshold. The summary call and the turn call each report usage, so two
// SessionUsageMsg arrive: the first holds exactly the summary call's totals
// (sent after the compaction), the second the sum.
func TestSubmitCompactionSendsSessionTotals(t *testing.T) {
	e := newCompactionEnvOn(t, "anthropic", 8192,
		provider.Turn{Text: compactionSummary, Usage: summaryUsage},
		provider.Turn{Text: "done", Usage: turnUsage})
	e.seed(8, compactionThreshold+2500)

	msgs := e.submit()
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Fatalf("turn did not complete: %#v", msgs)
	}

	got := sessionUsage(t, e.c, 2)
	wantTotals(t, "after the compaction", got[0], summaryUsage, usageCost(summaryUsage), "per_token", true)
	wantTotals(t, "after the turn", got[1], sumUsage(summaryUsage, turnUsage),
		usageCost(summaryUsage)+usageCost(turnUsage), "per_token", true)
}

// TestManualCompactSendsSessionTotals calls ManualCompact, which makes the
// summary call and starts no turn. The status bar must get that call's totals.
func TestManualCompactSendsSessionTotals(t *testing.T) {
	e := newCompactionEnvOn(t, "anthropic", 8192, provider.Turn{Text: compactionSummary, Usage: summaryUsage})
	e.seed(6, 15000)

	msg := e.a.ManualCompact(getenvKey)
	if msg.Outcome != tui.ManualCompactApplied {
		t.Fatalf("ManualCompact outcome = %#v, want applied", msg)
	}

	got := sessionUsage(t, e.c, 1)
	wantTotals(t, "after manual compaction", got[0], summaryUsage, usageCost(summaryUsage), "per_token", true)
}

// TestSessionTotalsRestartAfterNewSession runs a turn, calls NewSession, and
// runs a second turn. The second SessionUsageMsg holds only the second turn's
// usage and cost.
func TestSessionTotalsRestartAfterNewSession(t *testing.T) {
	e := newUsageEnv(t, "claude-sonnet-5-5",
		provider.Turn{Text: "one", Usage: usageTurn1},
		provider.Turn{Text: "two", Usage: usageTurn2})

	e.turn("first")
	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	e.turn("second")

	// usageTurn1 has the larger TokensIn, so it sorts last.
	got := sessionUsage(t, e.c, 2)
	wantTotals(t, "first session", got[1], usageTurn1, usageCost(usageTurn1), "per_token", true)
	wantTotals(t, "second session", got[0], usageTurn2, usageCost(usageTurn2), "per_token", true)
}

// frameWith feeds msgs[from:] into m the way the program would and returns the
// model and the index fed up to.
func frameWith(m tui.Model, msgs []any, from int) (tui.Model, int) {
	for _, msg := range msgs[from:] {
		next, _ := m.Update(msg.(tea.Msg))
		m = next.(tui.Model)
	}
	return m, len(msgs)
}

// newFrameModel builds a TUI model wide and tall enough to hold a short
// transcript without wrapping the notice.
func newFrameModel() tui.Model {
	m := tui.New(tui.Options{Version: "0.1.0-test", Caps: tui.Caps{Colour: false, Unicode: true}})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 60})
	return next.(tui.Model)
}

// TestUnknownModelNoticeOnceAcrossTwoTurns runs two turns on a model outside
// the table through Submit and feeds everything the app sent into a real TUI
// model. The frame holds the notice exactly once after both turns, and the
// status bar's right segment (Warnings) holds nothing. Both turns' totals
// having been fed is the positive control for the single count.
func TestUnknownModelNoticeOnceAcrossTwoTurns(t *testing.T) {
	e := newUsageEnv(t, "mystery-model-1",
		provider.Turn{Text: "one", Usage: usageTurn1},
		provider.Turn{Text: "two", Usage: usageTurn2})

	e.turn("first")
	e.turn("second")
	got := sessionUsage(t, e.c, 2)
	wantTotals(t, "unknown model, turn 2", got[1], sumUsage(usageTurn1, usageTurn2), 0, "unknown", false)

	m, _ := frameWith(newFrameModel(), e.c.snapshot(), 0)
	view := m.View()
	if n := strings.Count(view, unknownNoticeText); n != 1 {
		t.Errorf("frame holds the notice %d times, want 1:\n%s", n, view)
	}
	if !strings.Contains(view, "in 4.7k out 955") {
		t.Errorf("control: the frame lacks both turns' totals:\n%s", view)
	}
	if !strings.Contains(view, "$?") {
		t.Errorf("frame lacks $? for the unknown model:\n%s", view)
	}

	// The budget the turn ran on is the conservative one, not the known models'.
	budget := findMsg[tui.BudgetMsg](e.c.snapshot())
	if budget == nil {
		t.Fatal("no BudgetMsg")
	}
	if want := provider.LookupModel("mystery-model-1").ContextWindow; budget.ContextWindow != want {
		t.Errorf("BudgetMsg.ContextWindow = %d, want the conservative %d", budget.ContextWindow, want)
	}
	if budget.ContextWindow == provider.LookupModel("claude-sonnet-5-5").ContextWindow {
		t.Error("control: the unknown model got a known model's window")
	}
}

// TestKnownModelShowsNoUnknownNotice is the mirror: the same two turns on a
// per-token model put no notice in the frame, and the totals still render.
func TestKnownModelShowsNoUnknownNotice(t *testing.T) {
	e := newUsageEnv(t, "claude-sonnet-5-5",
		provider.Turn{Text: "one", Usage: usageTurn1},
		provider.Turn{Text: "two", Usage: usageTurn2})

	e.turn("first")
	e.turn("second")
	sessionUsage(t, e.c, 2)

	m, _ := frameWith(newFrameModel(), e.c.snapshot(), 0)
	view := m.View()
	if !strings.Contains(view, "in 4.7k out 955") {
		t.Fatalf("control: the frame lacks the totals:\n%s", view)
	}
	if strings.Contains(view, unknownNoticeText) {
		t.Errorf("frame holds the unknown-model notice for a known model:\n%s", view)
	}
	if strings.Contains(view, "$?") {
		t.Errorf("frame shows $? for a per-token model:\n%s", view)
	}
}

// TestUnknownModelNoticeAgainAfterNewSession runs a turn, calls NewSession and
// delivers the NewSessionDoneMsg the TUI's /new command would, then runs a
// second turn. The second session's frame holds the notice once, and the first
// session's totals are gone from it.
func TestUnknownModelNoticeAgainAfterNewSession(t *testing.T) {
	e := newUsageEnv(t, "mystery-model-1",
		provider.Turn{Text: "one", Usage: usageTurn1},
		provider.Turn{Text: "two", Usage: usageTurn2})

	m := newFrameModel()
	e.turn("first")
	sessionUsage(t, e.c, 1)
	m, fed := frameWith(m, e.c.snapshot(), 0)
	if n := strings.Count(m.View(), unknownNoticeText); n != 1 {
		t.Fatalf("control: %d notices in the first session, want 1", n)
	}

	if err := e.a.NewSession(getenvKey); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	next, _ := m.Update(tui.NewSessionDoneMsg{})
	m = next.(tui.Model)
	if n := strings.Count(m.View(), unknownNoticeText); n != 0 {
		t.Fatalf("%d notices right after /new, want the transcript cleared", n)
	}

	e.turn("second")
	sessionUsage(t, e.c, 2)
	m, _ = frameWith(m, e.c.snapshot(), fed)
	view := m.View()
	if n := strings.Count(view, unknownNoticeText); n != 1 {
		t.Errorf("%d notices in the second session, want 1:\n%s", n, view)
	}
	if !strings.Contains(view, "in 700 out 55") {
		t.Errorf("control: the second session's totals are missing:\n%s", view)
	}
}
