package tui

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// h1: Submit three distinct entries, ↑ three times recalls them newest first.
// A fourth ↑ stays on the oldest.
func TestHistoryRecall(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit three distinct entries using slash commands that don't open modals
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))
	m = drive(m, key('/'), key('c'), key('o'), key('m'), key('p'), key('a'), key('c'), key('t'), keyType(tea.KeyEnter))
	m = drive(m, key('/'), key('f'), key('i'), key('l'), key('e'), key('s'), keyType(tea.KeyEnter))

	// ↑ should recall "/files" (most recent)
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/files" {
		t.Errorf("after first ↑: composer = %q, want %q", m.comp.Value(), "/files")
	}

	// ↑ again should recall "/compact"
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/compact" {
		t.Errorf("after second ↑: composer = %q, want %q", m.comp.Value(), "/compact")
	}

	// ↑ again should recall "/status" (oldest)
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/status" {
		t.Errorf("after third ↑: composer = %q, want %q", m.comp.Value(), "/status")
	}

	// ↑ again should stay on "/status" (already at oldest)
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/status" {
		t.Errorf("after fourth ↑: composer = %q, want %q", m.comp.Value(), "/status")
	}
}

// h2: Type a draft without submitting, ↑, then ↓. The draft is restored.
func TestHistoryDraftRestored(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit one entry so there's history
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// Type a draft
	m = drive(m, key('d'), key('r'), key('a'), key('f'), key('t'))
	draft := m.comp.Value()
	if draft != "draft" {
		t.Fatalf("setup: draft = %q, want %q", draft, "draft")
	}

	// ↑ should recall the history entry
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/status" {
		t.Errorf("after ↑: composer = %q, want %q", m.comp.Value(), "/status")
	}

	// ↓ should restore the draft
	m = drive(m, keyType(tea.KeyDown))
	if m.comp.Value() != draft {
		t.Errorf("after ↓: composer = %q, want %q", m.comp.Value(), draft)
	}
}

// h3: No history. ↑ stays in Composing and leaves the composer text unchanged.
func TestHistoryUpWithNoHistory(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type some text but don't submit
	m = drive(m, key('x'), key('y'), key('z'))
	text := m.comp.Value()

	// ↑ should do nothing (no history)
	m = drive(m, keyType(tea.KeyUp))

	// Still in Composing mode
	if m.mode() != ModeComposing {
		t.Errorf("mode = %v, want Composing", m.mode())
	}
	// Text is unchanged
	if m.comp.Value() != text {
		t.Errorf("composer = %q, want %q", m.comp.Value(), text)
	}
}

// h4: Multi-line composer with the cursor on line 1. ↑ moves the cursor up
// and does not recall.
func TestHistoryUpInMiddleOfComposer(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit one entry so there's history
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// Type a multi-line entry: "line1\nline2"
	m = drive(m, key('l'), key('i'), key('n'), key('e'), key('1'), keyType(tea.KeyCtrlJ), key('l'), key('i'), key('n'), key('e'), key('2'))

	// Cursor should be on line 1 (second line, 0-indexed)
	if m.comp.ta.Line() != 1 {
		t.Fatalf("setup: cursor line = %d, want 1", m.comp.ta.Line())
	}

	// ↑ should move cursor up to line 0, not recall history
	before := m.comp.Value()
	m = drive(m, keyType(tea.KeyUp))

	if m.comp.ta.Line() != 0 {
		t.Errorf("cursor line = %d, want 0", m.comp.ta.Line())
	}
	if m.comp.Value() != before {
		t.Errorf("composer value changed; before = %q, after = %q", before, m.comp.Value())
	}
}

// h5: Consecutive duplicates are stored once. Empty and whitespace-only entries are never stored.
func TestHistoryDeduplication(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit "/files", then "/status", then "/status" again
	m = drive(m, key('/'), key('f'), key('i'), key('l'), key('e'), key('s'), keyType(tea.KeyEnter))
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// With dedupe: history is ['/status', '/files'], position at newest
	// First ↑ should recall '/status'
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/status" {
		t.Errorf("first ↑: composer = %q, want %q", m.comp.Value(), "/status")
	}

	// Second ↑ should recall '/files'
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/files" {
		t.Errorf("second ↑: composer = %q, want %q", m.comp.Value(), "/files")
	}
}

// T1: submit rejects whitespace-only input; such entries are never stored in history.
func TestSubmitRejectsWhitespaceOnly(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit "/files"
	m = drive(m, key('/'), key('f'), key('i'), key('l'), key('e'), key('s'), keyType(tea.KeyEnter))

	// Type whitespace only
	m = drive(m, key(' '), key(' '), key(' '))

	// Try to submit whitespace (will be rejected by submit(), leaving composer with whitespace)
	m = drive(m, keyType(tea.KeyEnter))

	// Composer still has whitespace (submit rejected it without clearing)
	if m.comp.Value() != "   " {
		t.Fatalf("setup: after rejected whitespace submit, composer = %q, want %q", m.comp.Value(), "   ")
	}

	// Clear the composer manually
	m = drive(m, keyType(tea.KeyEsc))

	// Submit "/status"
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// ↑ should show "/status" (the newest entry)
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/status" {
		t.Errorf("first ↑: composer = %q, want %q", m.comp.Value(), "/status")
	}

	// ↑ again should show "/files" (not whitespace), confirming the whitespace entry was never stored
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/files" {
		t.Errorf("second ↑: composer = %q, want %q (whitespace entry should not exist)", m.comp.Value(), "/files")
	}
}

// h6: Shift+↑ from Composing enters Browsing with the gutter drawn.
func TestShiftUpEntersBrowsing(t *testing.T) {
	m := NewWithFixture(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Need to close the approval first
	m = drive(m, keyType(tea.KeyEsc))

	// Should be in Composing
	if m.mode() != ModeComposing {
		t.Fatalf("setup: mode = %v, want Composing", m.mode())
	}

	// Shift+↑ should enter Browsing
	m = drive(m, keyType(tea.KeyShiftUp))

	if m.mode() != ModeBrowsing {
		t.Errorf("mode = %v, want Browsing", m.mode())
	}
}

// T3a: Tab with "/qu" completes to "/quit" and stays in Composing.
func TestTabCompletesSlashCommand(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = drive(m, key('/'), key('q'), key('u'), keyType(tea.KeyTab))
	if m.comp.Value() != "/quit" {
		t.Errorf("after Tab on '/qu': composer = %q, want %q", m.comp.Value(), "/quit")
	}
	if m.mode() != ModeComposing {
		t.Errorf("mode = %v, want Composing", m.mode())
	}
}

// T3b: Tab with "/" alone stays in Composing and leaves text unchanged.
func TestTabAmbiguousCommand(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = drive(m, key('/'), keyType(tea.KeyTab))
	if m.comp.Value() != "/" {
		t.Errorf("after Tab on '/': composer = %q, want %q", m.comp.Value(), "/")
	}
	if m.mode() != ModeComposing {
		t.Errorf("mode = %v, want Composing", m.mode())
	}
}

// T3c: Tab with "/quit now" moves to Browsing (has space, not a name-only command).
func TestTabWithArgumentEntersBrowsing(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = drive(m, key('/'), key('q'), key('u'), key('i'), key('t'), key(' '), key('n'), keyType(tea.KeyTab))
	if m.mode() != ModeBrowsing {
		t.Errorf("mode = %v, want Browsing", m.mode())
	}
}

// T3d: Tab with "hello" moves to Browsing (not a slash command).
func TestTabNonCommandEntersBrowsing(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = drive(m, key('h'), key('e'), key('l'), key('l'), key('o'), keyType(tea.KeyTab))
	if m.mode() != ModeBrowsing {
		t.Errorf("mode = %v, want Browsing", m.mode())
	}
}

// T3e: Tab with "/quit\\nfoo" (multiline) moves to Browsing.
func TestTabMultilineEntersBrowsing(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	m = drive(m, key('/'), key('q'), key('u'), key('i'), key('t'), keyType(tea.KeyCtrlJ), key('f'), key('o'), key('o'), keyType(tea.KeyTab))
	if m.mode() != ModeBrowsing {
		t.Errorf("mode = %v, want Browsing", m.mode())
	}
}

// h8: Tab in Browsing returns to Composing.
func TestTabInBrowsingReturnsToComposing(t *testing.T) {
	m := NewWithFixture(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Close the approval first
	m = drive(m, keyType(tea.KeyEsc))

	// Enter Browsing
	m = drive(m, keyType(tea.KeyShiftUp))
	if m.mode() != ModeBrowsing {
		t.Fatalf("setup: mode = %v, want Browsing", m.mode())
	}

	// Tab should return to Composing
	m = drive(m, keyType(tea.KeyTab))
	if m.mode() != ModeComposing {
		t.Errorf("mode = %v, want Composing", m.mode())
	}
}

// h9: While busy, ↑ does not recall.
func TestHistoryUpWhileBusy(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit an entry so there's history
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// Submit a plain message to start a busy turn
	m = drive(m, key('t'), key('e'), key('s'), key('t'), keyType(tea.KeyEnter))

	// Now we're busy
	if !m.busy.Active {
		t.Fatalf("setup: not busy")
	}

	// ↑ should not recall anything
	before := m.comp.Value()
	m = drive(m, keyType(tea.KeyUp))

	if m.comp.Value() != before {
		t.Errorf("while busy, ↑ changed composer: before = %q, after = %q", before, m.comp.Value())
	}
}

// T2: 101 submissions keep exactly the newest 100. (Alias for h10, different driver)
func TestHistoryCap101Entries(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit 101 distinct entries using a recognised slash command (/status i).
	// Ordinary text would set busy.Active and block further input in this
	// harness, so a non-busy command keeps the test focused on history capacity.
	for i := 0; i <= 100; i++ {
		text := "/status " + strconv.Itoa(i)
		for _, r := range text {
			m = drive(m, key(r))
		}
		m = drive(m, keyType(tea.KeyEnter))
	}

	// Press ↑ 100 times: should show "/status 1" (the oldest kept, since "/status 0" was dropped)
	for i := 0; i < 100; i++ {
		m = drive(m, keyType(tea.KeyUp))
	}
	want := "/status 1"
	if m.comp.Value() != want {
		t.Errorf("after 100 ↑: composer = %q, want %q", m.comp.Value(), want)
	}

	// Press ↑ once more: should still show the oldest entry
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != want {
		t.Errorf("after 101st ↑: composer = %q, want %q", m.comp.Value(), want)
	}
}

// T4: Esc clears the history position, so the next ↑ shows the newest entry (not skipping one).
func TestEscResetsHistoryPosition(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit two entries using recognised slash commands that do not open a
	// modal or start a turn, so the harness stays in Composing.
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))
	m = drive(m, key('/'), key('f'), key('i'), key('l'), key('e'), key('s'), keyType(tea.KeyEnter))

	// Type some text
	m = drive(m, key('a'), key('b'), key('c'))

	// ↑ shows the newest history entry
	want := "/files"
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != want {
		t.Errorf("after first ↑: composer = %q, want %q", m.comp.Value(), want)
	}

	// Esc clears the composer and resets history
	m = drive(m, keyType(tea.KeyEsc))
	if m.comp.Value() != "" {
		t.Errorf("after Esc: composer = %q, want empty", m.comp.Value())
	}

	// ↑ should show "/help" again, not "/status"
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != want {
		t.Errorf("after Esc and ↑: composer = %q, want %q", m.comp.Value(), want)
	}

	// ↓ should restore the empty draft (not "abc")
	m = drive(m, keyType(tea.KeyDown))
	if m.comp.Value() != "" {
		t.Errorf("after ↓: composer = %q, want empty", m.comp.Value())
	}
}

// T5: Soft-wrapped text: cursor movement with ↑/↓ on a long line without newlines.
func TestHistoryWithSoftWrappedText(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 40, Height: 24})

	// Submit one entry so there's history
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// Type a long line (200 characters) that wraps at 40 columns
	longLine := strings.Repeat("a", 200)
	for _, r := range longLine {
		m = drive(m, key(r))
	}

	// Move to the end
	m = drive(m, keyType(tea.KeyEnd))

	// Verify cursor is on the last logical line
	if m.comp.ta.Line() != 0 {
		t.Fatalf("setup: cursor should be on line 0, got %d", m.comp.ta.Line())
	}

	// Verify the text wraps (visual rows > 1)
	li := m.comp.ta.LineInfo()
	if li.Height <= 1 {
		t.Fatalf("setup: text should wrap (Height > 1), got %d", li.Height)
	}

	// ↑ at the end of the last visual row should move cursor up a visual row,
	// not enter history
	before := m.comp.Value()
	beforeRow := m.comp.ta.LineInfo().RowOffset
	m = drive(m, keyType(tea.KeyUp))

	// The value should be unchanged (history not entered)
	if m.comp.Value() != before {
		t.Errorf("↑ replaced history: before = %q, after = %q", before, m.comp.Value())
	}

	// The cursor should have moved up a visual row
	afterRow := m.comp.ta.LineInfo().RowOffset
	if afterRow >= beforeRow {
		t.Errorf("↑ did not move cursor up: before RowOffset = %d, after = %d", beforeRow, afterRow)
	}
}

// T6: With no history at the oldest entry, ↑ falls through to textarea and clears Hint.
func TestHistoryUpWithNoHistoryReachesTextarea(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Submit a recognised slash command (/status) to put something in history
	// without entering the busy model-turn state.
	m = drive(m, key('/'), key('s'), key('t'), key('a'), key('t'), key('u'), key('s'), keyType(tea.KeyEnter))

	// Recall the entry and install a hint manually: the unknown-command hint
	// path no longer exists, but the "↑ at oldest clears Hint" behaviour does.
	m = drive(m, keyType(tea.KeyUp))
	if m.comp.Value() != "/status" {
		t.Fatalf("setup: after ↑, should recall /status, got %q", m.comp.Value())
	}
	m.comp.Hint = "hint to clear"

	// Press ↑ again at the oldest entry.
	// History returns false, key falls through to textarea, which clears Hint.
	m = drive(m, keyType(tea.KeyUp))

	// Hint should now be cleared
	if m.comp.Hint != "" {
		t.Errorf("after ↑ at oldest: Hint should be cleared, got %q", m.comp.Hint)
	}

	// Composer value should still be /status
	if m.comp.Value() != "/status" {
		t.Errorf("after ↑ at oldest: composer = %q, want %q", m.comp.Value(), "/status")
	}
}

// B2: Pressing ↓ on a soft-wrapped history entry moves cursor down within the entry.
func TestHistoryDownOnWrappedEntryMovesCursor(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 40, Height: 24})

	// Create a 200-character single-line entry that wraps to at least 3 visual rows
	longLine := strings.Repeat("a", 200)

	// Manually add entries to history to avoid issues with drive+type
	m.comp.addHistory(longLine)
	m.comp.addHistory("/b")

	// Recall the longLine entry (second in history, after /b)
	m = drive(m, keyType(tea.KeyUp)) // recall /b (newest)
	m = drive(m, keyType(tea.KeyUp)) // recall longLine (older)

	// Verify we got the long line
	if m.comp.Value() != longLine {
		t.Fatalf("setup: after recalling, value length = %d", len(m.comp.Value()))
	}

	// Move to the end
	m = drive(m, keyType(tea.KeyEnd))

	// Get line info at end
	liAtEnd := m.comp.ta.LineInfo()
	if liAtEnd.Height <= 1 {
		t.Fatalf("setup: long entry should wrap (Height = %d)", liAtEnd.Height)
	}

	// Press ↑ to move cursor up one visual row (not history navigation)
	m = drive(m, keyType(tea.KeyUp))
	liAfterUp := m.comp.ta.LineInfo()

	// Press ↓ to move cursor back down
	m = drive(m, keyType(tea.KeyDown))

	// Value should still be the long line
	if m.comp.Value() != longLine {
		t.Errorf("after ↓: value should be unchanged (length %d, want %d)", len(m.comp.Value()), len(longLine))
	}

	// Cursor should be further down than after ↑
	liAfterDown := m.comp.ta.LineInfo()
	if liAfterDown.RowOffset <= liAfterUp.RowOffset {
		t.Errorf("↓ should move cursor down: RowOffset %d → %d", liAfterUp.RowOffset, liAfterDown.RowOffset)
	}
}

// T7: Shift+↑ while busy still enters Browsing.
func TestShiftUpWhileBusyEntersBrowsing(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Pre-populate with a transcript item so Browsing has something to select
	m = drive(m, key('h'), key('e'), key('l'), key('l'), key('o'), keyType(tea.KeyEnter))

	// Now we're busy
	if !m.busy.Active {
		t.Fatalf("setup: not busy after submitting a message")
	}

	// Shift+↑ should enter Browsing even while busy
	m = drive(m, keyType(tea.KeyShiftUp))

	if m.mode() != ModeBrowsing {
		t.Errorf("mode = %v, want Browsing", m.mode())
	}
}
