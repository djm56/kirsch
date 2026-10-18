package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// sizedModel returns an empty model at the requested terminal size.
func sizedModel(t *testing.T, w, h int, colour bool) Model {
	t.Helper()
	m := New(Options{Version: fixtureVersion, Caps: Caps{Colour: colour, Unicode: true}})
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next.(Model)
}

// TestAssistantTextDeltaCreatesStreamingTextBlock checks that the first delta
// opens a streaming assistant block and that later deltas append to it.
func TestAssistantTextDeltaCreatesStreamingTextBlock(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "Hello"})

	id, ok := m.textBlocks[1]
	if !ok {
		t.Fatal("textBlocks[1] not created")
	}
	it, _, found := m.tr.Find(id)
	if !found {
		t.Fatal("transcript item not found")
	}
	if it.Kind != KindAssistant || it.Text == nil {
		t.Fatalf("want KindAssistant text block, got %+v", it)
	}
	if !it.Text.Streaming {
		t.Error("new text block should be streaming")
	}
	if got := strings.Join(it.Text.Lines, "\\n"); got != "Hello" {
		t.Errorf("lines = %q, want Hello", got)
	}

	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: " world"})
	it, _, _ = m.tr.Find(id)
	if got := strings.Join(it.Text.Lines, "\\n"); got != "Hello world" {
		t.Errorf("lines after second delta = %q, want Hello world", got)
	}
}

// TestTurnCompleteFinalisesStreamingText checks that TurnCompleteMsg clears the
// streaming flag and the adapter's block map.
func TestTurnCompleteFinalisesStreamingText(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "Hello"})
	id := m.textBlocks[1]
	m = drive(m, TurnCompleteMsg{})

	if m.textBlocks[1] != 0 {
		t.Error("textBlocks not cleared after TurnCompleteMsg")
	}
	it, _, found := m.tr.Find(id)
	if !found {
		t.Fatal("original transcript item not found")
	}
	if it.Text.Streaming {
		t.Error("text block still streaming after TurnCompleteMsg")
	}
	if m.busy.Active {
		t.Error("busy still active after TurnCompleteMsg")
	}
}

// TestThinkingDeltaRendersAsCollapsedDimmedCard checks that a thinking delta
// creates a KindThinking item and that its rendered card is dimmed, not an
// error-coloured card.
func TestThinkingDeltaRendersAsCollapsedDimmedCard(t *testing.T) {
	m := sizedModel(t, 80, 24, true)
	m = drive(m, ThinkingDeltaMsg{BlockID: 1, Delta: "planning"})

	id, ok := m.thinkingBlocks[1]
	if !ok {
		t.Fatal("thinkingBlocks[1] not created")
	}
	it, _, found := m.tr.Find(id)
	if !found || it.Kind != KindThinking || it.Thinking == nil {
		t.Fatalf("want KindThinking card, got %+v", it)
	}

	view := m.View()
	if !strings.Contains(view, "thinking") {
		t.Errorf("view missing 'thinking':\n%s", view)
	}
	if strings.Contains(view, "\x1b[38;5;203m") {
		t.Error("thinking card rendered with error colour")
	}
}

// TestTurnErrorRendersErrorCardPositiveControl proves the SGR colour check in
// TestThinkingDeltaRendersAsCollapsedDimmedCard is meaningful: an actual error
// card does emit the error colour.
func TestTurnErrorRendersErrorCardPositiveControl(t *testing.T) {
	m := sizedModel(t, 80, 24, true)
	m = drive(m, TurnErrorMsg{Kind: "max_turns_exceeded", Message: "too many rounds"})

	view := m.View()
	if !strings.Contains(view, "max_turns_exceeded") {
		t.Errorf("view missing error kind:\n%s", view)
	}
	if !strings.Contains(view, "\x1b[38;5;203m") {
		t.Error("error card did not render with error colour; positive control failed")
	}
}

// TestUsageMsgUpdatesStatusBarTokens checks that a UsageMsg adds the token
// counts into m.status.Tokens.
func TestUsageMsgUpdatesStatusBarTokens(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, UsageMsg{InputTokens: 10, OutputTokens: 5, CacheReadTokens: 1, CacheWriteTokens: 2})

	if m.status.Tokens != 18 {
		t.Errorf("status.Tokens = %d, want 18", m.status.Tokens)
	}
}

// TestUsageMsgRendersTokensInStatusBar checks that real usage tokens are
// reflected in the rendered status bar, not just stored in m.status.
func TestUsageMsgRendersTokensInStatusBar(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, UsageMsg{InputTokens: 10, OutputTokens: 5, CacheReadTokens: 1, CacheWriteTokens: 2})

	row := stripEscapes(m.statusRow(m.layout()))
	if !strings.Contains(row, "18 tok") {
		t.Errorf("status row does not render token count:\n%s", row)
	}
}

// TestModelNameFromWiredValue checks that the status bar renders the model name
// supplied in Options.Status, not the removed hard-coded default. The absence
// check is paired with a grep that the literal no longer appears in this
// package (DIR-026 positive control).
func TestModelNameFromWiredValue(t *testing.T) {
	m := New(Options{Version: fixtureVersion, Caps: Caps{Colour: false, Unicode: true}, Status: Status{Model: "minimax-m2.7"}})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)

	row := stripEscapes(m.statusRow(m.layout()))
	if !strings.Contains(row, "minimax-m2.7") {
		t.Errorf("status row does not render wired model name:\n%s", row)
	}
}

// TestBudgetMsgRendersBudgetPercent checks that a BudgetMsg drives the right-
// side budget indicator and that a turn-end message clears the estimate.
func TestBudgetMsgRendersBudgetPercent(t *testing.T) {
	m := New(Options{Version: fixtureVersion, Caps: Caps{Colour: false, Unicode: true}, Status: Status{Model: "claude-sonnet-5-5", ContextWindow: 128000}})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)

	m = drive(m, BudgetMsg{EstimatedTokens: 100000, ContextWindow: 128000})
	row := stripEscapes(m.statusRow(m.layout()))
	if !strings.Contains(row, "context 80% full") {
		t.Errorf("status row does not render budget percent:\n%s", row)
	}

	m = drive(m, TurnCompleteMsg{})
	if m.status.EstimatedTokens != 0 {
		t.Errorf("EstimatedTokens not cleared after turn complete: got %d", m.status.EstimatedTokens)
	}
}

// stripEscapes removes SGR escape sequences so tests can assert on visible text.
func stripEscapes(s string) string {
	return sgrRe.ReplaceAllString(s, "")
}

// TestTurnCancelledMarksStreamingTextCancelled checks that a cancellation
// message marks the streaming assistant text as cancelled and clears busy.
func TestTurnCancelledMarksStreamingTextCancelled(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "Hello"})
	id := m.textBlocks[1]
	m = drive(m, TurnCancelledMsg{})

	if m.textBlocks[1] != 0 {
		t.Error("textBlocks not cleared after TurnCancelledMsg")
	}
	it, _, found := m.tr.Find(id)
	if !found {
		t.Fatal("original transcript item not found")
	}
	if !it.Text.Cancelled {
		t.Error("text block not marked cancelled")
	}
	if m.busy.Active {
		t.Error("busy still active after TurnCancelledMsg")
	}
}

// TestCommandOutputChunkAppendsToRunningToolCard checks that a command output
// chunk appends sanitised lines to the Out field of the matching tool card.
func TestCommandOutputChunkAppendsToRunningToolCard(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m,
		ToolStartedMsg{ID: 7, Name: "run_command", Target: "go test"},
		CommandOutputChunkMsg{ID: 7, Chunk: "ok\r\nPASS\n"},
	)

	id, ok := m.toolCards[7]
	if !ok {
		t.Fatal("toolCards[7] not created")
	}
	it, _, found := m.tr.Find(id)
	if !found || it.Kind != KindTool || it.Tool == nil {
		t.Fatalf("want KindTool card, got %+v", it)
	}
	want := []string{"ok", "PASS"}
	if len(it.Tool.Out) != len(want) {
		t.Fatalf("tool Out = %v, want %v", it.Tool.Out, want)
	}
	for i := range want {
		if it.Tool.Out[i] != want[i] {
			t.Errorf("Out[%d] = %q, want %q", i, it.Tool.Out[i], want[i])
		}
	}
}

// TestTurnMessageGoldenProperties runs the ui-spec §13 checks over the new
// rendered surfaces introduced in this step.
func TestTurnMessageGoldenProperties(t *testing.T) {
	cases := []struct {
		name string
		msgs []tea.Msg
	}{
		{
			name: "assistant_text_delta",
			msgs: []tea.Msg{AssistantTextDeltaMsg{BlockID: 1, Delta: "Hello, world."}},
		},
		{
			name: "thinking_delta",
			msgs: []tea.Msg{ThinkingDeltaMsg{BlockID: 1, Delta: "planning the answer"}},
		},
		{
			name: "turn_error",
			msgs: []tea.Msg{TurnErrorMsg{Kind: "context_overflow", Message: "too large"}},
		},
		{
			name: "command_output_chunk",
			msgs: []tea.Msg{
				ToolStartedMsg{ID: 1, Name: "run_command", Target: "go test"},
				CommandOutputChunkMsg{ID: 1, Chunk: "PASS"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			styled := sizedModel(t, 80, 24, true)
			styled = drive(styled, tc.msgs...)
			styledView := styled.View()

			plain := sizedModel(t, 80, 24, false)
			plain = drive(plain, tc.msgs...)
			plainView := plain.View()

			// §13 property 1: stripping escapes yields the plain frame.
			if got := sgrRe.ReplaceAllString(styledView, ""); got != plainView {
				t.Errorf("stripped styled != plain")
			}

			// §13 property 2 & 3: only palette SGR escapes appear in styled output.
			for i := 0; i < len(styledView); i++ {
				if styledView[i] != 0x1b {
					continue
				}
				loc := sgrRe.FindStringIndex(styledView[i:])
				if loc == nil || loc[0] != 0 {
					t.Fatalf("non-SGR escape at byte %d", i)
				}
				params := sgrRe.FindStringSubmatch(styledView[i:])[1]
				if !PaletteSGR[params] {
					t.Errorf("SGR %q is not in the palette", params)
				}
				i += loc[1] - 1
			}

			// §13 property 4: frame dimensions are exact and rows fit the width.
			lines := strings.Split(styledView, "\n")
			if len(lines) != 24 {
				t.Errorf("%d rows, want 24", len(lines))
			}
			for i, l := range lines {
				if cw := cellWidth(l); cw > 80 {
					t.Errorf("row %d: %d cols, want <= 80", i+1, cw)
				}
			}
		})
	}
}

// TestNoColourTurnMessagesEmitZeroEscapes is the NO_COLOR half of the §13
// escape property for the new message surfaces.
func TestNoColourTurnMessagesEmitZeroEscapes(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m,
		AssistantTextDeltaMsg{BlockID: 1, Delta: "Hello"},
		ThinkingDeltaMsg{BlockID: 2, Delta: "think"},
		TurnErrorMsg{Kind: "max_turns_exceeded", Message: "too many"},
	)
	if i := strings.IndexByte(m.View(), 0x1b); i >= 0 {
		t.Errorf("escape byte at %d", i)
	}
}

// TestCompactingMsgSetsBusyVerbCompacting checks that CompactingMsg marks the
// model busy with the verb "compacting", which the status bar shows.
func TestCompactingMsgSetsBusyVerbCompacting(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	if m.busy.Active {
		t.Fatal("precondition: a fresh model is busy")
	}
	m = drive(m, CompactingMsg{})
	if !m.busy.Active || m.busy.Verb != "compacting" {
		t.Errorf("busy = %+v, want Active with Verb \"compacting\"", m.busy)
	}
}

// TestSessionCompactedMsgMarksHeaderCompactedAndKeepsBusy checks that
// SessionCompactedMsg sets sess.Compacted and shows "(compacted)" in the
// rendered header, and that it does not itself clear the busy state: the turn
// that follows compaction is still running.
func TestSessionCompactedMsgMarksHeaderCompactedAndKeepsBusy(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	if strings.Contains(m.View(), "(compacted)") {
		t.Fatal("control: header shows (compacted) before SessionCompactedMsg")
	}
	m = drive(m, CompactingMsg{}, SessionCompactedMsg{})
	if !m.sess.Compacted {
		t.Error("sess.Compacted not set by SessionCompactedMsg")
	}
	if !strings.Contains(m.View(), "(compacted)") {
		t.Error("header does not show (compacted) after SessionCompactedMsg")
	}
	if !m.busy.Active || m.busy.Verb != "compacting" {
		t.Errorf("SessionCompactedMsg changed busy to %+v", m.busy)
	}
}

// TestCompactingBusyIsClearedByTurnErrorMsg checks that a context_overflow (or
// any) TurnErrorMsg after CompactingMsg clears the busy state, so a failed
// compaction cannot leave the spinner running. The Compacting-only model is
// the control: it is busy until the error arrives.
func TestCompactingBusyIsClearedByTurnErrorMsg(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, CompactingMsg{})
	if !m.busy.Active {
		t.Fatal("control: not busy after CompactingMsg")
	}
	m = drive(m, TurnErrorMsg{Kind: "context_overflow", Message: "too big", Detail: "/new"})
	if m.busy.Active || m.busy.Verb != "" {
		t.Errorf("busy = %+v after TurnErrorMsg, want cleared", m.busy)
	}
}

// TestCompactingBusyIsClearedByTurnCancelledMsg checks that TurnCancelledMsg
// after CompactingMsg clears the busy state, the way a cancelled compaction
// ends.
func TestCompactingBusyIsClearedByTurnCancelledMsg(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, CompactingMsg{})
	if !m.busy.Active {
		t.Fatal("control: not busy after CompactingMsg")
	}
	m = drive(m, TurnCancelledMsg{})
	if m.busy.Active || m.busy.Verb != "" {
		t.Errorf("busy = %+v after TurnCancelledMsg, want cleared", m.busy)
	}
}

// TestSessionUsageWideRender checks that a SessionUsageMsg on a wide layout
// renders all four token counts (in, out, cache read/write) plus cost.
func TestSessionUsageWideRender(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m.status.Model = "claude-sonnet-5-5"
	m = drive(m, SessionUsageMsg{
		TokensIn:   1000,
		TokensOut:  300,
		CacheRead:  8000,
		CacheWrite: 2000,
		CostUSD:    0.04,
		CostState:  "per_token",
	})

	if m.status.SessionTokensIn != 1000 {
		t.Errorf("status.SessionTokensIn = %d, want 1000", m.status.SessionTokensIn)
	}
	if m.status.SessionTokensOut != 300 {
		t.Errorf("status.SessionTokensOut = %d, want 300", m.status.SessionTokensOut)
	}
	if m.status.SessionTokensCacheR != 8000 {
		t.Errorf("status.SessionTokensCacheR = %d, want 8000", m.status.SessionTokensCacheR)
	}
	if m.status.SessionTokensCacheW != 2000 {
		t.Errorf("status.SessionTokensCacheW = %d, want 2000", m.status.SessionTokensCacheW)
	}
	if m.status.SessionCostUSD != 0.04 {
		t.Errorf("status.SessionCostUSD = %v, want 0.04", m.status.SessionCostUSD)
	}
	if m.status.SessionCostState != "per_token" {
		t.Errorf("status.SessionCostState = %q, want per_token", m.status.SessionCostState)
	}

	row := stripEscapes(m.statusRow(m.layout()))
	if !strings.Contains(row, "in 1.0k out 300 cache 8.0k/2.0k") {
		t.Errorf("status row does not render session tokens:\n%s", row)
	}
	if !strings.Contains(row, "$0.04") {
		t.Errorf("status row does not render cost:\n%s", row)
	}
}

// TestSessionUsageFlatNoCost checks that a Flat pricing model shows no cost.
func TestSessionUsageFlatNoCost(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m.status.Model = "minimax-m2.7"
	m = drive(m, SessionUsageMsg{
		TokensIn:   1000,
		TokensOut:  300,
		CacheRead:  100,
		CacheWrite: 50,
		CostState:  "flat",
	})

	row := stripEscapes(m.statusRow(m.layout()))
	if strings.Contains(row, "$") {
		t.Errorf("status row should not show cost for flat model:\n%s", row)
	}
	if !strings.Contains(row, "in 1.0k out 300") {
		t.Errorf("status row does not render session tokens:\n%s", row)
	}
}

// TestSessionUsageSmallCostFormat checks that costs below $0.01 use 4 decimals.
func TestSessionUsageSmallCostFormat(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m.status.Model = "claude-sonnet-5-5"
	m = drive(m, SessionUsageMsg{
		TokensIn:   100,
		TokensOut:  50,
		CacheRead:  10,
		CacheWrite: 5,
		CostUSD:    0.001234,
		CostState:  "per_token",
	})

	row := stripEscapes(m.statusRow(m.layout()))
	if !strings.Contains(row, "$0.0012") {
		t.Errorf("status row does not use 4 decimals for small cost:\n%s", row)
	}
}

// countNotices returns how many transcript items are a notice reading text.
func countNotices(m Model, text string) int {
	n := 0
	for _, it := range m.tr.Items() {
		if it.Kind == KindNotice && it.Notice != nil && it.Notice.Text == text {
			n++
		}
	}
	return n
}

// statusRowAt renders the status row of a model driven by msgs at width w.
func statusRowAt(t *testing.T, w int, msgs ...any) string {
	t.Helper()
	m := sizedModel(t, w, 24, false)
	m.status.Model = "claude-sonnet-5-5"
	for _, msg := range msgs {
		m = drive(m, msg)
	}
	return stripEscapes(m.statusRow(m.layout()))
}

// TestSessionUsageUnknownPricingShowsQuestionMark: a model the table does not
// know has "unknown" cost state and the row shows "$?". The model name carries
// neither "unknown" nor "$", so the row cannot satisfy the check by echoing it.
// The mirrors show "$?" is absent for a per-token and a flat model, each of
// which still renders its tokens (the positive control for the absence).
func TestSessionUsageUnknownPricingShowsQuestionMark(t *testing.T) {
	use := func(state string, known bool) SessionUsageMsg {
		return SessionUsageMsg{
			TokensIn: 1000, TokensOut: 300, CacheRead: 100, CacheWrite: 50,
			CostUSD: 0.04, CostState: state, ModelKnown: known,
		}
	}
	for _, tc := range []struct {
		name      string
		state     string
		known     bool
		wantQuery bool
	}{
		{"unlisted model shows $?", "unknown", false, true},
		{"known model with unknown pricing shows $?", "unknown", true, true},
		{"per-token model does not", "per_token", true, false},
		{"flat model does not", "flat", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := sizedModel(t, 100, 24, false)
			m.status.Model = "acme-model-7"
			m = drive(m, use(tc.state, tc.known))
			row := stripEscapes(m.statusRow(m.layout()))
			if !strings.Contains(row, "in 1.0k out 300 cache 100/50") {
				t.Fatalf("control: tokens not rendered:\n%s", row)
			}
			if got := strings.Contains(row, "$?"); got != tc.wantQuery {
				t.Errorf("row contains $? = %v, want %v:\n%s", got, tc.wantQuery, row)
			}
		})
	}
}

// TestSessionUsageCostBoundary pins the 4-decimal switch at $0.01: below it the
// figure has 4 decimals, from it 2. Each side is the mirror of the other.
func TestSessionUsageCostBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost float64
		want string
		not  string
	}{
		{"just below 0.01 uses 4 decimals", 0.0099, "$0.0099", ""},
		{"exactly 0.01 uses 2 decimals", 0.01, "$0.01", "$0.0100"},
		{"above 0.01 uses 2 decimals", 1.234, "$1.23", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := statusRowAt(t, 100, SessionUsageMsg{TokensIn: 10, CostUSD: tc.cost, CostState: "per_token", ModelKnown: true})
			if !strings.Contains(row, tc.want) {
				t.Errorf("row lacks %q:\n%s", tc.want, row)
			}
			if tc.not != "" && strings.Contains(row, tc.not) {
				t.Errorf("row contains %q:\n%s", tc.not, row)
			}
		})
	}
}

// TestSessionUsageASCIIGlyphs renders the cost with Unicode off: the separators
// are the glyph set's ASCII bullet and the row holds no U+00B7. The Unicode
// model is the mirror: it has the middle dot and the same cost text.
func TestSessionUsageASCIIGlyphs(t *testing.T) {
	render := func(unicode bool) string {
		m := New(Options{Version: fixtureVersion, Caps: Caps{Colour: false, Unicode: unicode}})
		next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
		m = next.(Model)
		m.status.Model = "claude-sonnet-5-5"
		m = drive(m, SessionUsageMsg{
			TokensIn: 1000, TokensOut: 300, CacheRead: 8000, CacheWrite: 2000,
			CostUSD: 0.04, CostState: "per_token", ModelKnown: true,
		})
		return stripEscapes(m.statusRow(m.layout()))
	}

	ascii := render(false)
	if !strings.Contains(ascii, "cache 8.0k/2.0k . $0.04") {
		t.Errorf("ASCII row does not join the cost with the ASCII bullet:\n%s", ascii)
	}
	if strings.Contains(ascii, "\u00b7") {
		t.Errorf("ASCII row contains U+00B7:\n%s", ascii)
	}
	uni := render(true)
	if !strings.Contains(uni, "cache 8.0k/2.0k \u00b7 $0.04") {
		t.Errorf("Unicode row does not join the cost with the middle dot:\n%s", uni)
	}
}

// TestSessionUsageNarrowLayoutFallsBackToSessionTotal drives a per-call UsageMsg
// (111 tokens) and session totals (11.3k tokens). Where the detail does not fit
// the row shows the session total and never the per-call figure; below the
// spec width it shows no token figure. The wide row is the control that the
// totals are what the bar normally shows.
func TestSessionUsageNarrowLayoutFallsBackToSessionTotal(t *testing.T) {
	msgs := []any{
		UsageMsg{InputTokens: 100, OutputTokens: 11},
		SessionUsageMsg{
			TokensIn: 1000, TokensOut: 300, CacheRead: 8000, CacheWrite: 2000,
			CostUSD: 0.04, CostState: "per_token", ModelKnown: true,
		},
	}
	build := func(w int, model string) string {
		m := sizedModel(t, w, 24, false)
		m.status.Model = model
		m.status.Family = "fam"
		for _, msg := range msgs {
			m = drive(m, msg)
		}
		return stripEscapes(m.statusRow(m.layout()))
	}

	t.Run("wide shows the detail", func(t *testing.T) {
		row := build(100, "claude-sonnet-5-5")
		if !strings.Contains(row, "in 1.0k out 300 cache 8.0k/2.0k") {
			t.Errorf("control: no detail in wide row:\n%s", row)
		}
	})

	t.Run("detail does not fit: session total, not the per-call figure", func(t *testing.T) {
		// 80 columns, a model name long enough to push the detail out.
		row := build(80, strings.Repeat("m", 40))
		if strings.Contains(row, "cache") {
			t.Fatalf("control: the detail still fits, so this is not the fallback:\n%s", row)
		}
		if !strings.Contains(row, "11.3k tok") {
			t.Errorf("row has no session total:\n%s", row)
		}
		if strings.Contains(row, "111 tok") {
			t.Errorf("row shows the per-call figure:\n%s", row)
		}
	})

	t.Run("below the spec width: no tokens at all", func(t *testing.T) {
		row := build(79, "claude-sonnet-5-5")
		if strings.Contains(row, "tok") || strings.Contains(row, "cache") {
			t.Errorf("row shows tokens below 80 columns:\n%s", row)
		}
		if !strings.Contains(row, "claude-sonnet-5-5") {
			t.Errorf("control: the model is missing from the row:\n%s", row)
		}
	})
}

// TestStatusLadderDropsTokensBeforeGrants sweeps the model-name length and, for
// every width of the left segment, checks ui-spec §2.3's order: a row that still
// shows a token figure also shows the grants. The sweep must reach rows with
// tokens and grants, with grants only, and with neither, or it proved nothing.
func TestStatusLadderDropsTokensBeforeGrants(t *testing.T) {
	var both, grantsOnly, neither int
	for n := 1; n <= 70; n++ {
		m := sizedModel(t, 80, 24, false)
		m.GetGrantCount = func() int { return 2 }
		m.status.Model = strings.Repeat("m", n)
		m.status.Family = "fam"
		m = drive(m, SessionUsageMsg{
			TokensIn: 1000, TokensOut: 300, CacheRead: 8000, CacheWrite: 2000,
			CostUSD: 0.04, CostState: "per_token", ModelKnown: true,
		})
		row := stripEscapes(m.statusRow(m.layout()))
		hasTok := strings.Contains(row, "tok") || strings.Contains(row, "cache")
		hasGrants := strings.Contains(row, "2 grants")
		switch {
		case hasTok && hasGrants:
			both++
		case hasTok && !hasGrants:
			t.Errorf("n=%d: tokens shown while grants dropped:\n%s", n, row)
		case hasGrants:
			grantsOnly++
		default:
			neither++
		}
	}
	if both == 0 || grantsOnly == 0 || neither == 0 {
		t.Errorf("sweep did not cover every rung: both=%d grantsOnly=%d neither=%d", both, grantsOnly, neither)
	}
}

// TestUnknownModelNotice covers the dim transcript notice for a model missing
// from the table (ui-spec §7.5): once per session, never in the status bar's
// Warnings, not for a known model, and again after /new.
func TestUnknownModelNotice(t *testing.T) {
	unknown := SessionUsageMsg{TokensIn: 1000, CacheRead: 100, CostState: "unknown", ModelKnown: false}
	known := SessionUsageMsg{TokensIn: 1000, CacheRead: 100, CostState: "unknown", ModelKnown: true}

	t.Run("shown once across two usage messages", func(t *testing.T) {
		m := sizedModel(t, 100, 24, false)
		m = drive(m, unknown)
		if got := countNotices(m, unknownModelNotice); got != 1 {
			t.Fatalf("after the first message: %d notices, want 1", got)
		}
		m = drive(m, unknown)
		if got := countNotices(m, unknownModelNotice); got != 1 {
			t.Errorf("after the second message: %d notices, want 1", got)
		}
		if len(m.status.Warnings) != 0 {
			t.Errorf("status.Warnings = %v, want none: the notice is a transcript item", m.status.Warnings)
		}
	})

	t.Run("not shown for a known model whose pricing is unknown", func(t *testing.T) {
		m := sizedModel(t, 100, 24, false)
		m = drive(m, known, known)
		if got := countNotices(m, unknownModelNotice); got != 0 {
			t.Errorf("%d notices for a known model, want 0", got)
		}
		// Control: the same messages did reach the model.
		if m.status.SessionTokensIn != 1000 {
			t.Errorf("control: SessionTokensIn = %d, want 1000", m.status.SessionTokensIn)
		}
	})

	t.Run("shown again, once, after a new session", func(t *testing.T) {
		m := sizedModel(t, 100, 24, false)
		m = drive(m, unknown)
		if got := countNotices(m, unknownModelNotice); got != 1 {
			t.Fatalf("control: %d notices before /new, want 1", got)
		}
		m = drive(m, NewSessionDoneMsg{})
		if got := countNotices(m, unknownModelNotice); got != 0 {
			t.Errorf("%d notices right after /new, want the transcript cleared", got)
		}
		if m.status.SessionTokensIn != 0 || m.status.SessionCostState != "" {
			t.Errorf("totals not reset by /new: in=%d state=%q", m.status.SessionTokensIn, m.status.SessionCostState)
		}
		m = drive(m, unknown, unknown)
		if got := countNotices(m, unknownModelNotice); got != 1 {
			t.Errorf("%d notices in the second session, want 1", got)
		}
		if len(m.status.Warnings) != 0 {
			t.Errorf("status.Warnings = %v after /new, want none", m.status.Warnings)
		}
	})

	t.Run("a new session does not reset the totals when it fails", func(t *testing.T) {
		m := sizedModel(t, 100, 24, false)
		m = drive(m, unknown)
		m = drive(m, NewSessionDoneMsg{Err: errors.New("boom")})
		if m.status.SessionTokensIn != 1000 {
			t.Errorf("SessionTokensIn = %d after a failed /new, want 1000", m.status.SessionTokensIn)
		}
	})
}
