package app

// Tests for m5-d18: the session totals after resume (C5) and the /status
// compaction percentage after a manual /compact (C50).
//
// Entry points, so each name can be read against what it really calls:
//   - The resume tests call App.ResumeLastSession, the method cmd/kirsch/main.go
//     calls at startup, over a session log written with session.Store. The
//     messages it sends are fed into a real tui.Model through Update.
//   - The compaction test calls App.Submit (to put the fixture at 60%),
//     App.ManualCompact (the /compact callback) and App.StatusInfo (what the
//     /status command reads). It does not call runCompaction directly.
//
// TUI messages reach the collector through one goroutine per
// sendAsyncSequence call, so these tests assert which messages arrive, not the
// order between calls.

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/tui"
)

// resumeUsageEvents are the usage events written to the fixture log. Their sum
// is in 3500, out 550, cache read 40, cache write 12.
var resumeUsageEvents = []agent.Usage{
	{InputTokens: 1000, OutputTokens: 200, CacheReadTokens: 30, CacheWriteTokens: 7},
	{InputTokens: 2000, OutputTokens: 300},
	{InputTokens: 500, OutputTokens: 50, CacheReadTokens: 10, CacheWriteTokens: 5},
}

// resumeSumUsage is the sum of resumeUsageEvents, as provider.Usage so the
// existing usageCost helper prices it.
var resumeSumUsage = provider.Usage{InputTokens: 3500, OutputTokens: 550, CacheReadTokens: 40, CacheWriteTokens: 12}

// resumeApp builds an App on the anthropic endpoint (its model has per-token
// pricing) with a collector that never closes. The App has not started a
// session, so ResumeLastSession is its first session operation.
func resumeApp(t *testing.T, stateDir string) (*App, *collector) {
	t.Helper()
	cfg := config.Defaults()
	cfg.Provider.Default = "anthropic"
	cfg.Session.StorageDir = filepath.Join(stateDir, "sessions")
	c := newCollector(-1)
	a := testAppWithConfig(t, cfg, c)
	t.Cleanup(a.Close)
	return a, c
}

// writeUsageLog creates a session for a's workspace, records one transcript
// message and one usage event per element of usages, and returns the log path.
// extra, when non-nil, may append raw bytes to the log after the events.
func writeUsageLog(t *testing.T, a *App, usages []agent.Usage, extra func(path string)) string {
	t.Helper()
	root, err := a.stateRoot()
	if err != nil {
		t.Fatalf("stateRoot: %v", err)
	}
	layout, err := session.NewLayout(root, a.ws.CanonicalRoot)
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}
	path, _, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	s, err := session.New(path, session.WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	s.Record(agent.RecordAppend, userTextMessage("hello"))
	for _, u := range usages {
		s.RecordUsage(u)
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if extra != nil {
		extra(path)
	}
	return path
}

// TestResumeLastSessionSendsLogTotalsToStatusBar resumes a session whose log
// holds three usage events and checks, before any turn runs, that the
// SessionUsageMsg ResumeLastSession sent carries their sum, that a real
// tui.Model fed the sent messages renders it in the status bar, and that the
// App's own totals (what /status reads) agree. usage.json at the state root is
// seeded with totals from every session that uses the same data root, which differ from the log's sum; if the
// resume read it, the totals would be those instead. The log's sum arriving is
// the positive control for the usage.json figures being absent.
func TestResumeLastSessionSendsLogTotalsToStatusBar(t *testing.T) {
	a, c := resumeApp(t, t.TempDir())
	writeUsageLog(t, a, resumeUsageEvents, nil)

	root, err := a.stateRoot()
	if err != nil {
		t.Fatal(err)
	}
	// Totals from other sessions that use the same data root: not this session's.
	if _, err := session.AddUsage(root, 777000, 888000, 0, 0, 0, false, time.Now().UTC()); err != nil {
		t.Fatalf("AddUsage: %v", err)
	}

	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}

	got := sessionUsage(t, c, 1)[0]
	wantTotals(t, "resumed session", got, resumeSumUsage, usageCost(resumeSumUsage), "per_token", true)

	m, _ := frameWith(newFrameModel(), c.snapshot(), 0)
	view := m.View()
	if !strings.Contains(view, "in 3.5k out 550 cache 40/12") {
		t.Errorf("status bar lacks the log's totals before any new usage:\n%s", view)
	}
	if strings.Contains(view, "777") || strings.Contains(view, "888") {
		t.Errorf("status bar shows usage.json's figures for every session that uses the same data root:\n%s", view)
	}

	a.storeMu.Lock()
	in, out, read, write := a.sessionTokensIn, a.sessionTokensOut, a.sessionTokensRead, a.sessionTokensWrite
	cost := a.sessionCostUSD
	a.storeMu.Unlock()
	if in != 3500 || out != 550 || read != 40 || write != 12 {
		t.Errorf("App totals = %d/%d/%d/%d, want 3500/550/40/12", in, out, read, write)
	}
	if math.Abs(cost-usageCost(resumeSumUsage)) > 1e-12 {
		t.Errorf("App cost = %v, want %v", cost, usageCost(resumeSumUsage))
	}
	info := a.StatusInfo(getenvKey)
	if info.TokensIn != 3500 || info.TokensOut != 550 || info.TokensCache != 52 {
		t.Errorf("StatusInfo tokens = %d/%d/%d, want 3500/550/52", info.TokensIn, info.TokensOut, info.TokensCache)
	}
}

// TestResumeLastSessionSendsZeroTotalsWithoutUsageEvents resumes a session
// whose log has no usage event. The totals sent are zero, which is the control
// that the sum in the test above comes from the log's events and not from a
// constant, and a SessionUsageMsg is still sent.
func TestResumeLastSessionSendsZeroTotalsWithoutUsageEvents(t *testing.T) {
	a, c := resumeApp(t, t.TempDir())
	writeUsageLog(t, a, nil, nil)

	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}

	got := sessionUsage(t, c, 1)[0]
	wantTotals(t, "resumed session with no usage events", got, provider.Usage{}, 0, "per_token", true)
}

// TestResumeLastSessionTornUsageLine resumes a session whose log ends in a torn
// line after two intact usage events and a third event that follows the tear.
// The totals are the two events before the tear: the loader, like Resume and
// LoadGrants, treats the first torn line as the end of the log.
func TestResumeLastSessionTornUsageLine(t *testing.T) {
	a, c := resumeApp(t, t.TempDir())
	writeUsageLog(t, a, resumeUsageEvents[:2], func(path string) {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		// A torn line, then a whole usage event that must not be counted.
		if _, err := f.WriteString("{\"v\":1,\"type\":\"usage\",\"input_to\n" +
			`{"v":1,"type":"usage","input_tokens":99999,"output_tokens":99999,"cache_read":0,"cache_write":0}` + "\n"); err != nil {
			t.Fatal(err)
		}
	})

	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}

	got := sessionUsage(t, c, 1)[0]
	want := sumUsage(provider.Usage{
		InputTokens: resumeUsageEvents[0].InputTokens, OutputTokens: resumeUsageEvents[0].OutputTokens,
		CacheReadTokens: resumeUsageEvents[0].CacheReadTokens, CacheWriteTokens: resumeUsageEvents[0].CacheWriteTokens,
	}, provider.Usage{
		InputTokens: resumeUsageEvents[1].InputTokens, OutputTokens: resumeUsageEvents[1].OutputTokens,
	})
	wantTotals(t, "resumed session with a torn line", got, want, usageCost(want), "per_token", true)
}

// TestManualCompactUpdatesStatusCompactionPercent puts a fixture session at 60%
// on /status, then calls ManualCompact and reads /status again.
//
// The fixture: six seeded turns sized so the estimate of the next request is
// about 15556 tokens, then one App.Submit turn, which stores that estimate for
// /status. 60% of the 25904-token budget (window 30000, reserve 4096) is
// 15542.4 and 61% is 15801.44, and compactPercent floors, so an estimate in
// 15543..15801 reads 60%; the fixture's 15556 is inside it. The before value is
// asserted to be 60.
//
// The after value is compactPercent of the estimate of the compacted
// conversation, which is the Messages the compaction returned (runCompaction
// stores them in a.conv.Messages). It is asserted to differ from 60 and to be
// below it. ManualCompact sends no message for this; the figure is the one
// StatusInfo reads.
func TestManualCompactUpdatesStatusCompactionPercent(t *testing.T) {
	e := newCompactionEnv(t, 8192,
		provider.Turn{Text: "ok"},
		provider.Turn{Text: compactionSummary})
	e.seed(6, 15556)

	e.a.Submit(compactionUserText, getenvKey)
	waitForMsg[tui.TurnCompleteMsg](t, e.c, 5*time.Second)
	e.a.inFlight.Wait()

	before := e.a.StatusInfo(getenvKey).CompactionPercent
	t.Logf("compaction %% before /compact: %d", before)
	if before != 60 {
		t.Fatalf("compaction %% before /compact = %d, want 60 (fixture estimate is off)", before)
	}

	msg := e.a.ManualCompact(getenvKey)
	if msg.Outcome != tui.ManualCompactApplied {
		t.Fatalf("ManualCompact outcome = %v (detail %q), want Applied", msg.Outcome, msg.Detail)
	}

	// What the compaction returned is now the conversation: the estimate of
	// the compacted request, no user text pending, is the figure /status must
	// show.
	wantEstimate := e.a.estimateRequest("", e.a.conv, &toolExecutor{app: e.a})
	wantAfter := compactPercent(wantEstimate, compactionWindow)
	after := e.a.StatusInfo(getenvKey).CompactionPercent
	t.Logf("compaction %% after /compact: %d (estimate %d tokens)", after, wantEstimate)
	if after == 60 {
		t.Fatalf("compaction %% after /compact is still 60: /status is stale")
	}
	if after != wantAfter {
		t.Errorf("compaction %% after /compact = %d, want %d", after, wantAfter)
	}
	if after >= before {
		t.Errorf("compaction %% after /compact = %d, not below the %d before", after, before)
	}
}
