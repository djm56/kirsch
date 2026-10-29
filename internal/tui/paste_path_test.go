package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// pasteMsg builds a bracketed paste the way Bubble Tea v1 delivers one: a
// KeyMsg with Paste set. There is no tea.PasteMsg in v1.
func pasteMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Paste: true, Runes: []rune(s)}
}

// composerRows returns every rendered composer row of the frame, in order, with
// styling and the left margin stripped. It reads the frame through
// composerRow, so what it sees is what the user sees.
func composerRows(t *testing.T, m Model) []string {
	t.Helper()
	n := m.layout().ComposerH
	rows := make([]string, n)
	for i := range rows {
		rows[i] = composerRow(t, m, i)
	}
	return rows
}

// numberedLines returns n lines "line00".."line<n-1>" joined by newlines.
func numberedLines(n int) string {
	ls := make([]string, n)
	for i := range ls {
		ls[i] = fmt.Sprintf("line%02d", i)
	}
	return strings.Join(ls, "\n")
}

// TestPasteLandsVerbatim drives an indented multi-line paste through
// Model.Update (handleKey -> handlePaste -> InsertString) and asserts the
// composer holds exactly the pasted bytes. Indentation is spaces: the
// textarea's sanitiser rewrites a tab, so a tab is not carried verbatim.
func TestPasteLandsVerbatim(t *testing.T) {
	text := "func main() {\n    if ok {\n        run()\n    }\n\n  trailing  \n}"
	m := drive(composing(t), pasteMsg(text))
	if m.mode() != ModeComposing {
		t.Fatalf("mode after an ordinary paste = %v, want Composing", m.mode())
	}
	if got := m.comp.Value(); got != text {
		t.Errorf("composer = %q, want the pasted text %q", got, text)
	}
	// The paste is literal text, not keys: it must not have been submitted.
	if m.busy.Active {
		t.Error("an ordinary paste started a turn")
	}
}

// TestLargePasteConfirmAcceptAndDecline covers the §7.2 prompt for a paste over
// PasteWarnBytes: raised through handlePaste, resolved through keyConfirm, and
// on accept inserted by applyConfirm's ConfirmLargePaste arm.
func TestLargePasteConfirmAcceptAndDecline(t *testing.T) {
	// 9 bytes a line, so the text is multi-line and over the threshold.
	payload := strings.Repeat("  abcdef\n", PasteWarnBytes/9+10)
	if len(payload) <= PasteWarnBytes {
		t.Fatalf("setup: payload is %d bytes, not over %d", len(payload), PasteWarnBytes)
	}

	raise := func(t *testing.T) Model {
		t.Helper()
		m := drive(composing(t), pasteMsg(payload))
		if m.mode() != ModeConfirm {
			t.Fatalf("an oversized paste raised no prompt: mode = %v", m.mode())
		}
		if got := m.comp.Value(); got != "" {
			t.Fatalf("composer = %d bytes before the prompt was answered, want none", len(got))
		}
		return m
	}

	t.Run("accept", func(t *testing.T) {
		m := drive(raise(t), key('y'))
		if m.mode() != ModeComposing {
			t.Errorf("mode after accepting = %v, want Composing", m.mode())
		}
		if got := m.comp.Value(); got != payload {
			t.Errorf("composer = %d bytes after accept, want the %d-byte payload", len(got), len(payload))
		}
	})

	t.Run("decline", func(t *testing.T) {
		// Both keys that decline: n and Esc.
		for name, k := range map[string]tea.KeyMsg{"n": key('n'), "esc": keyType(tea.KeyEsc)} {
			m := drive(raise(t), k)
			if m.mode() != ModeComposing {
				t.Errorf("%s: mode after declining = %v, want Composing", name, m.mode())
			}
			if got := m.comp.Value(); got != "" {
				t.Errorf("%s: composer holds %d bytes after declining, want none", name, len(got))
			}
		}

		// Control: absence above only means something if the same text, admitted
		// through the same entry point under the threshold, does show up. The
		// first PasteWarnBytes bytes of the payload are exactly at the limit,
		// which handlePaste inserts without a prompt.
		small := payload[:PasteWarnBytes]
		c := drive(composing(t), pasteMsg(small))
		if c.mode() != ModeComposing {
			t.Fatalf("control: a paste at the threshold raised a prompt: mode = %v", c.mode())
		}
		if got := c.comp.Value(); got != small {
			t.Errorf("control: composer = %d bytes, want the %d-byte paste", len(got), len(small))
		}
	})
}

// TestMultiLinePasteStartingWithSlashIsNotACommand pastes text whose first line
// is a command name and presses Enter. parseSlash must refuse it, so the text
// is an ordinary message: it stays in the composer after the paste, and Enter
// submits it whole instead of opening /help.
func TestMultiLinePasteStartingWithSlashIsNotACommand(t *testing.T) {
	text := "/help\n  second line"

	// Control: the same command name on one line is a command. Without it, "no
	// modal opened" below could mean Enter did nothing at all.
	ctl := drive(composing(t), pasteMsg("/help"), keyType(tea.KeyEnter))
	if ctl.mode() != ModeModal {
		t.Fatalf("control: single-line /help left mode = %v, want Modal", ctl.mode())
	}

	var sent []string
	m := composing(t)
	m.Submit = func(s string) { sent = append(sent, s) }
	m = drive(m, pasteMsg(text))
	if got := m.comp.Value(); got != text {
		t.Fatalf("composer after the paste = %q, want %q", got, text)
	}
	if m.mode() != ModeComposing {
		t.Fatalf("mode after the paste = %v, want Composing: a paste must not run a command", m.mode())
	}

	m = drive(m, keyType(tea.KeyEnter))
	if m.mode() == ModeModal {
		t.Error("Enter on a multi-line /help paste opened the help overlay")
	}
	if len(sent) != 1 || sent[0] != text {
		t.Errorf("Submit calls = %q, want exactly one with %q", sent, text)
	}
}

// TestComposerScrollsToTheCaret pastes 12 lines — more than the five rows the
// composer can show — and checks that the rendered rows follow the caret. It
// reads the rows of the frame (Model.View), so it checks what the user sees.
func TestComposerScrollsToTheCaret(t *testing.T) {
	m := drive(composing(t), pasteMsg(numberedLines(12)))
	caret := m.gly.Caret
	h := m.layout().ComposerH
	if h != 5 {
		t.Fatalf("setup: composer is %d rows tall, want 5", h)
	}

	// Caret at the end of line 11 (the paste leaves it there): the window ends
	// on line 11, and the first row carries no prompt.
	rows := composerRows(t, m)
	want := []string{"  line07", "  line08", "  line09", "  line10", "  line11" + caret}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("caret on line 11: row %d = %q, want %q", i, rows[i], want[i])
		}
	}

	// Walk the caret up and check the window at each edge. caretRow 5 is the
	// first row past the top window (start 1); 4 is the last row still inside it.
	ups := func(m Model, n int) Model {
		for i := 0; i < n; i++ {
			m = drive(m, keyType(tea.KeyUp))
		}
		return m
	}
	cases := []struct {
		name  string
		row   int // logical line the caret is on
		first string
		last  string
	}{
		{"line 5, one past the top window", 5, "  line01", "  line05" + caret},
		{"line 4, last row of the top window", 4, "> line00", "  line04" + caret},
		{"line 2, inside the top window", 2, "> line00", "  line04"},
	}
	for _, c := range cases {
		got := composerRows(t, ups(m, 11-c.row))
		if got[0] != c.first {
			t.Errorf("%s: first row = %q, want %q", c.name, got[0], c.first)
		}
		if got[4] != c.last {
			t.Errorf("%s: last row = %q, want %q", c.name, got[4], c.last)
		}
		n := 0
		for _, r := range got {
			n += strings.Count(r, caret)
		}
		if n != 1 {
			t.Errorf("%s: %d caret glyphs in rows %q, want exactly 1", c.name, n, got)
		}
	}
}

// TestComposerWindowFollowsTheCaretLineByLine pins how the window moves between
// adjacent caret lines, which TestComposerScrollsToTheCaret's spot checks do not:
// the window is derived from the caret's line alone, so past the top window it
// moves one line for every line the caret moves and the caret stays on the
// bottom row. A window that held still while the caret moved inside it would
// pass the spot checks at lines 11 and 4 and fail here at line 10.
//
// It pastes 12 lines through Model.Update, then presses Up one line at a time
// from line 11 to line 0 through Model.Update, and after each key compares all
// five rows of the frame (Model.View) against the expected window.
func TestComposerWindowFollowsTheCaretLineByLine(t *testing.T) {
	const n = 12
	m := drive(composing(t), pasteMsg(numberedLines(n)))
	caret := m.gly.Caret
	h := m.layout().ComposerH
	if h != 5 {
		t.Fatalf("setup: composer is %d rows tall, want 5", h)
	}

	// topLine[r] is the first line shown with the caret on line r, written out
	// rather than computed so the test does not restate the code's formula.
	// Lines 4 and 5 are where the window first leaves the top.
	topLine := [n]int{0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7}

	// want builds the rows for a caret on line r: lines topLine[r] onwards, the
	// prompt on line 0 only, the caret appended (every line is the same width
	// and Up keeps the column, so the caret is at end of line).
	want := func(r int) []string {
		start := topLine[r]
		rows := make([]string, h)
		for i := range rows {
			l := start + i
			prefix := "  "
			if l == 0 {
				prefix = "> "
			}
			rows[i] = prefix + fmt.Sprintf("line%02d", l)
			if l == r {
				rows[i] += caret
			}
		}
		return rows
	}

	for r := n - 1; r >= 0; r-- {
		if r < n-1 {
			m = drive(m, keyType(tea.KeyUp))
		}
		if got := m.comp.ta.Line(); got != r {
			t.Fatalf("setup: caret on line %d after %d Up keys, want %d", got, n-1-r, r)
		}
		got := composerRows(t, m)
		exp := want(r)
		if got[0] != exp[0] || got[h-1] != exp[h-1] {
			t.Errorf("caret on line %d: first/last row = %q/%q, want %q/%q",
				r, got[0], got[h-1], exp[0], exp[h-1])
		}
		for i := 1; i < h-1; i++ {
			if got[i] != exp[i] {
				t.Errorf("caret on line %d: row %d = %q, want %q", r, i, got[i], exp[i])
			}
		}
		glyphs := 0
		for _, row := range got {
			glyphs += strings.Count(row, caret)
		}
		if glyphs != 1 {
			t.Errorf("caret on line %d: %d caret glyphs in rows %q, want exactly 1", r, glyphs, got)
		}
	}
}

// TestComposerWindowStaysAtTheTopWithoutACaret covers the cases that have no
// caret to follow: an unfocused composer shows the top of a tall input, and so
// does a busy one (View clears Focused while busy).
//
// The unfocused arm calls Composer.rows directly and not the frame: Model.View
// sets Focused from the mode, and no frame state reaches an unfocused composer
// holding a tall paste without opening an overlay that hides it.
func TestComposerWindowStaysAtTheTopWithoutACaret(t *testing.T) {
	m := drive(composing(t), pasteMsg(numberedLines(12)))
	// Control: focused, the same input has scrolled away from the top.
	if got := composerRows(t, m)[0]; got != "  line07" {
		t.Fatalf("control: focused first row = %q, want %q", got, "  line07")
	}

	// Control for the direct call: focused, rows scrolls exactly as the frame did.
	m.comp.Focused = true
	rows := m.comp.rows(m.layout(), m.sty, m.gly, false)
	if got := stripSGR(rows[0]); got != "  line07" {
		t.Fatalf("control: focused rows[0] = %q, want %q", got, "  line07")
	}
	m.comp.Focused = false
	rows = m.comp.rows(m.layout(), m.sty, m.gly, false)
	if got := stripSGR(rows[0]); got != "> line00" {
		t.Errorf("unfocused: first row = %q, want the top of the input", got)
	}
	if got := stripSGR(rows[4]); got != "  line04" {
		t.Errorf("unfocused: last row = %q, want line 4", got)
	}

	busy := m
	busy.busy = Busy{Active: true, Verb: "thinking"}
	got := composerRows(t, busy)
	if !strings.HasPrefix(got[0], "> line00") || got[4] != "  line04" {
		t.Errorf("busy: rows = %q, want the top five lines", got)
	}
}

// TestComposerHintYieldsToAFullWindow pins the Hint rule: the hint takes the
// row after the text while one is left over, and is dropped, not the caret row,
// once the text fills every row.
func TestComposerHintYieldsToAFullWindow(t *testing.T) {
	const hint = "unknown command"
	// Three lines plus a hint fit in the composer's four rows: the hint shows.
	m := drive(composing(t), pasteMsg(numberedLines(3)))
	m.comp.Hint = hint
	rows := composerRows(t, m)
	if got := rows[len(rows)-1]; got != "  "+hint {
		t.Fatalf("control: last row = %q, want the hint with room to spare", got)
	}

	// Twelve lines fill all five rows: the hint is dropped, the caret row stays.
	m = drive(composing(t), pasteMsg(numberedLines(12)))
	m.comp.Hint = hint
	rows = composerRows(t, m)
	for i, r := range rows {
		if strings.Contains(r, hint) {
			t.Errorf("row %d = %q: the hint displaced a text row", i, r)
		}
	}
	if got, want := rows[len(rows)-1], "  line11"+m.gly.Caret; got != want {
		t.Errorf("last row = %q, want the caret row %q", got, want)
	}
}
