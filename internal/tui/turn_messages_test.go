package tui

import (
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
