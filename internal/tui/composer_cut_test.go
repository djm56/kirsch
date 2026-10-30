package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// cutFrame returns the composer rows of a full View() frame at 40 columns, with
// SGR and the left margin removed, plus the layout they were drawn under. It
// drives the real render path (View), so the marker the user sees is what is
// measured.
func cutFrame(t *testing.T, unicode bool, value, hint string) ([]string, Layout) {
	t.Helper()
	m := NewWithFixture(Options{Version: fixtureVersion, Caps: Caps{Colour: false, Unicode: unicode}})
	m = drive(m, tea.WindowSizeMsg{Width: 40, Height: 24}, key('y'))
	if m.mode() != ModeComposing {
		t.Fatalf("setup: mode = %v, want Composing", m.mode())
	}
	m.comp.SetValue(value)
	m.comp.Hint = hint
	lay := m.layout()
	lines := strings.Split(stripSGR(m.View()), "\n")
	first := len(lines) - lay.HintH - lay.ComposerH
	if first < 0 {
		t.Fatalf("frame of %d rows has no composer of %d rows", len(lines), lay.ComposerH)
	}
	out := make([]string, 0, lay.ComposerH)
	for _, ln := range lines[first : first+lay.ComposerH] {
		out = append(out, strings.TrimRight(strings.TrimPrefix(ln, strings.Repeat(" ", lay.Pad)), " "))
	}
	return out, lay
}

// TestComposerCutRowsEndInMarkerAt40 drives View() at 40 columns with composer
// lines wider than the frame and checks the marker (ui-spec §7.3): every row
// whose text is wider than the content width ends in the truncation glyph
// (`...` without Unicode, §10.2), a row that fits is not marked, and the prompt
// row and the hint row follow the same rule.
//
// The rows are written out by hand by construction: line 0 (prompt row) and
// line 2 are long, line 1 fits. The hint is the long one in the second case.
func TestComposerCutRowsEndInMarkerAt40(t *testing.T) {
	long0 := strings.Repeat("a", 60)
	long2 := strings.Repeat("c", 80)
	value := long0 + "\nshort\n" + long2

	for _, tc := range []struct {
		name    string
		unicode bool
		marker  string
	}{
		{"unicode", true, "⋯"},
		{"ascii", false, "..."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, lay := cutFrame(t, tc.unicode, value, "")
			if len(rows) != 3 {
				t.Fatalf("composer has %d rows, want 3: %q", len(rows), rows)
			}
			if lay.ContentW >= 60 {
				t.Fatalf("ContentW = %d: the fixture lines must be wider than it", lay.ContentW)
			}
			// Prompt row: line 0, cut.
			if !strings.HasPrefix(rows[0], "> a") {
				t.Errorf("prompt row lost its prompt: %q", rows[0])
			}
			if !strings.HasSuffix(rows[0], tc.marker) {
				t.Errorf("cut prompt row has no %q marker: %q", tc.marker, rows[0])
			}
			if got := cellWidth(rows[0]); got != lay.ContentW {
				t.Errorf("cut prompt row is %d cells, want %d: %q", got, lay.ContentW, rows[0])
			}
			// Positive control: a row that fits is untouched and unmarked.
			if rows[1] != "  short" {
				t.Errorf("fitting row = %q, want %q", rows[1], "  short")
			}
			if strings.Contains(rows[1], tc.marker) {
				t.Errorf("row that fits carries the marker: %q", rows[1])
			}
			// Continuation row, cut.
			if !strings.HasPrefix(rows[2], "  c") || !strings.HasSuffix(rows[2], tc.marker) {
				t.Errorf("cut continuation row = %q, want a `  c…` row ending in %q", rows[2], tc.marker)
			}
			if got := cellWidth(rows[2]); got != lay.ContentW {
				t.Errorf("cut continuation row is %d cells, want %d: %q", got, lay.ContentW, rows[2])
			}
		})
	}

	// Hint row: one short line leaves a row for a hint, which is cut too.
	rows, lay := cutFrame(t, true, "hi", strings.Repeat("h", 70))
	last := rows[len(rows)-1]
	if !strings.HasPrefix(last, "  h") || !strings.HasSuffix(last, "⋯") || cellWidth(last) != lay.ContentW {
		t.Errorf("long hint row = %q (%d cells), want a `  h…` row of %d cells ending in ⋯",
			last, cellWidth(last), lay.ContentW)
	}
	// ...and a hint that fits is not marked.
	rows, _ = cutFrame(t, true, "hi", "fits")
	if last := rows[len(rows)-1]; last != "  fits" {
		t.Errorf("short hint row = %q, want %q", last, "  fits")
	}

	// Control for the check itself: exactly() is the frame's hard edge and adds
	// no marker (§7.3), so a long row passed straight to it ends mid-text. If
	// the assertions above were satisfied by exactly() they would be vacuous.
	hard := exactly([]string{long0}, 1, 40)[0]
	if strings.HasSuffix(hard, "⋯") || cellWidth(hard) != 40 {
		t.Errorf("exactly() = %q, want a 40-cell cut with no marker", hard)
	}
}

// composerAt40 returns a Composing model at 40 columns (ContentW 38) with the
// given capabilities, the fixture's opening approval resolved.
func composerAt40(t *testing.T, caps Caps) Model {
	t.Helper()
	m := NewWithFixture(Options{Version: fixtureVersion, Caps: caps})
	m = drive(m, tea.WindowSizeMsg{Width: 40, Height: 24}, key('y'))
	if m.mode() != ModeComposing {
		t.Fatalf("setup: mode = %v, want Composing", m.mode())
	}
	return m
}

// composerRowsOf renders m through View() and returns its composer rows with the
// frame's left margin and trailing spaces removed. SGR is stripped when plain is
// true and kept when it is false.
func composerRowsOf(t *testing.T, m Model, plain bool) ([]string, Layout) {
	t.Helper()
	lay := m.layout()
	frame := m.View()
	if plain {
		frame = stripSGR(frame)
	}
	lines := strings.Split(frame, "\n")
	first := len(lines) - lay.HintH - lay.ComposerH
	if first < 0 {
		t.Fatalf("frame of %d rows has no composer of %d rows", len(lines), lay.ComposerH)
	}
	out := make([]string, 0, lay.ComposerH)
	for _, ln := range lines[first : first+lay.ComposerH] {
		out = append(out, strings.TrimRight(strings.TrimPrefix(ln, strings.Repeat(" ", lay.Pad)), " "))
	}
	return out, lay
}

// lastSGR returns the last SGR escape in s, or "" when there is none.
func lastSGR(s string) string {
	i := strings.LastIndex(s, "\x1b[")
	if i < 0 {
		return ""
	}
	return s[i : i+sgrPrefixLen(s[i:])]
}

// letters returns n distinct-looking ASCII characters (a..z repeating), so a
// hidden or shifted character changes the row.
func letters(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(byte('a' + i%26))
	}
	return b.String()
}

// TestComposerFitIsMeasuredOnTheTextNotTheCaret pins the fit boundary at 40
// columns (ContentW 38), in both glyph sets. The prompt row and a continuation
// row carry a two-cell lead, so their text fits while it is at most
// ContentW-2 = 36 cells wide.
//
// The caret at end-of-input is a cell of its own past the last character, so a
// 36-cell text with the caret is 39 cells wide. That row must not be cut or
// marked: the frame's hard edge drops the caret cell and all 36 typed
// characters stay visible. One cell wider (37) the text itself no longer fits
// and the row is cut: the first ContentW-w cells of the row, where w is the
// marker's width in cells (1 for the Unicode glyph, 3 for `...`), then the
// marker. Each width is run with the caret at end of input and in the middle of
// the text (overstriking a character, so it never adds a cell), on both the
// prompt row and a continuation row.
//
// The expectation is built from the text per rule, not from a stored row: text
// at most 36 wide keeps every cell up to ContentW; wider text keeps the first
// ContentW-w cells and the marker. The text is ASCII, so a rune is a cell and
// the cut lands exactly on the limit (a wide rune at the cut is covered in
// TestComposerCaretOverWideAndZeroWidthAtTheBoundary).
func TestComposerFitIsMeasuredOnTheTextNotTheCaret(t *testing.T) {
	const contentW = 38
	for _, glyphs := range []struct {
		name    string
		unicode bool
	}{
		{"unicode", true},
		{"ascii", false},
	} {
		for _, rowKind := range []struct {
			name, prefix, before string
			row                  int
		}{
			{"prompt row", "> ", "", 0},
			{"continuation row", "  ", "x\n", 1},
		} {
			for _, caretKind := range []struct {
				name  string
				lefts int
			}{
				{"caret at end", 0},
				{"caret in middle", 3},
			} {
				for _, n := range []int{35, 36, 37} {
					name := glyphs.name + "/" + rowKind.name + "/" + caretKind.name + "/" + string(rune('0'+n/10)) + string(rune('0'+n%10)) + " cells"
					t.Run(name, func(t *testing.T) {
						m := composerAt40(t, Caps{Colour: false, Unicode: glyphs.unicode})
						text := letters(n)
						m.comp.SetValue(rowKind.before + text)
						for i := 0; i < caretKind.lefts; i++ {
							m = drive(m, keyType(tea.KeyLeft))
						}
						rows, lay := composerRowsOf(t, m, true)
						if lay.ContentW != contentW {
							t.Fatalf("ContentW = %d, the boundary is written for %d", lay.ContentW, contentW)
						}
						got := rows[rowKind.row]

						// The row as drawn, caret included, before any cut.
						caret := m.gly.Caret
						full := rowKind.prefix + text + caret
						if caretKind.lefts > 0 {
							p := n - caretKind.lefts
							full = rowKind.prefix + text[:p] + caret + text[p+1:]
						}
						rs := []rune(full)
						var want string
						if n <= contentW-2 {
							want = string(rs[:min(len(rs), contentW)])
						} else {
							want = string(rs[:contentW-cellWidth(m.gly.Trunc)]) + m.gly.Trunc
						}
						if got != want {
							t.Errorf("row = %q (%d cells), want %q", got, cellWidth(got), want)
						}
						if cellWidth(got) > contentW {
							t.Errorf("row is %d cells, over ContentW %d: %q", cellWidth(got), contentW, got)
						}
						// The point of the boundary: while the text fits, no
						// marker and no typed character hidden.
						if n <= contentW-2 {
							if strings.Contains(got, m.gly.Trunc) {
								t.Errorf("text of %d cells fits but the row is marked: %q", n, got)
							}
							if caretKind.lefts == 0 && !strings.Contains(got, text) {
								t.Errorf("a typed character is hidden: row %q does not hold %q", got, text)
							}
						} else if !strings.HasSuffix(got, m.gly.Trunc) {
							t.Errorf("text of %d cells does not fit but the row is not marked: %q", n, got)
						}
					})
				}
			}
		}
	}
}

// TestComposerCaretOverWideAndZeroWidthAtTheBoundary runs the fit boundary with
// the caret mid-text over clusters whose width is not one, each through View()
// at 40 columns (ContentW 38) on the prompt row (a two-cell lead, so 36 cells of
// text fit).
//
//   - wide: the caret overstrikes a two-cell rune and takes both its cells, so
//     the row is exactly as wide as its text. 18 wide runes (36 cells) fit and
//     are unmarked; 19 (38 cells) do not. The cut lands on a wide rune, so the
//     marker row is 37 cells: at most ContentW, one short of it. That is the
//     width the post-condition allows.
//   - zero-width (U+200B, and a stranded U+0301 at the start of the line): the
//     cluster has no cells, so the caret draws a cell of its own in the middle
//     of the row. 36 letters plus the cluster fit as text (38 cells with the
//     lead) but the caret makes the row 39, and the hard edge would drop a typed
//     letter off the end. The row must be cut and marked instead. 35 letters
//     plus the cluster is 38 cells with the caret and stays whole.
func TestComposerCaretOverWideAndZeroWidthAtTheBoundary(t *testing.T) {
	const contentW = 38
	cases := []struct {
		name    string
		value   string
		lefts   int // Left presses from end of input to put the caret on the cluster
		cut     bool
		wantLen int // expected cell width of the row; 0 skips the check
	}{
		{"wide/18 runes fit", strings.Repeat("世", 18), 9, false, 38},
		{"wide/19 runes cut", strings.Repeat("世", 19), 9, true, 37},
		{"zwsp/35 letters fit", letters(3) + "\u200b" + letters(32), 33, false, 38},
		{"zwsp/36 letters cut", letters(3) + "\u200b" + letters(33), 34, true, 38},
		{"stranded mark/35 letters fit", "́" + letters(35), 36, false, 38},
		{"stranded mark/36 letters cut", "́" + letters(36), 37, true, 38},
	}
	for _, unicode := range []bool{true, false} {
		for _, tc := range cases {
			name := tc.name
			if !unicode {
				name = "ascii/" + name
			}
			t.Run(name, func(t *testing.T) {
				m := composerAt40(t, Caps{Colour: false, Unicode: unicode})
				m.comp.SetValue(tc.value)
				for i := 0; i < tc.lefts; i++ {
					m = drive(m, keyType(tea.KeyLeft))
				}
				rows, lay := composerRowsOf(t, m, true)
				if lay.ContentW != contentW {
					t.Fatalf("ContentW = %d, the boundary is written for %d", lay.ContentW, contentW)
				}
				got := rows[0]
				marked := strings.HasSuffix(got, m.gly.Trunc)
				if marked != tc.cut {
					t.Errorf("marked = %v, want %v: %q (%d cells)", marked, tc.cut, got, cellWidth(got))
				}
				if cellWidth(got) > contentW {
					t.Errorf("row is %d cells, over ContentW %d: %q", cellWidth(got), contentW, got)
				}
				// Per-case widths are written for the Unicode caret (one cell);
				// the ASCII marker is wider, so only the marked/unmarked and
				// the ceiling are checked there.
				if unicode && tc.wantLen != 0 && cellWidth(got) != tc.wantLen {
					t.Errorf("row is %d cells, want %d: %q", cellWidth(got), tc.wantLen, got)
				}
				// A row that is not cut keeps every typed letter.
				if !tc.cut {
					for _, r := range letters(26) {
						if strings.ContainsRune(tc.value, r) && !strings.ContainsRune(got, r) {
							t.Errorf("typed %q is missing from %q", r, got)
							break
						}
					}
				}
			})
		}
	}
}

// TestComposerBusyFirstRowTakesNoCaretAllowance calls rows directly with the
// composer Focused and busy, a state View never builds (it unfocuses the
// composer while a turn is live), so the (!busy || i != 0) guard on caretOut is
// pinned here and nowhere else. While busy, line 0 is drawn from the plain value
// with no caret cell, so a row of ContentW+1 cells is plain text that does not
// fit and must be cut and marked rather than given the end-of-input allowance.
func TestComposerBusyFirstRowTakesNoCaretAllowance(t *testing.T) {
	m := composerAt40(t, Caps{Colour: false, Unicode: true})
	// "> " plus 37 cells is 39 = ContentW+1; the hint does not fit beside it.
	m.comp.SetValue(letters(37))
	m.comp.Focused = true
	lay := m.layout()

	// Control: not busy, the same text puts the caret at end-of-input on the
	// row, and the row (text 39 wide with the caret, 40) is cut and marked.
	rows := m.comp.rows(lay, m.sty, m.gly, false)
	if got := stripSGR(rows[0]); !strings.HasSuffix(got, m.gly.Trunc) {
		t.Fatalf("control: not-busy row = %q, want it marked", got)
	}

	rows = m.comp.rows(lay, m.sty, m.gly, true)
	got := stripSGR(rows[0])
	if !strings.HasSuffix(got, m.gly.Trunc) || cellWidth(got) != lay.ContentW {
		t.Errorf("busy first row = %q (%d cells), want a %d-cell row ending in %q",
			got, cellWidth(got), lay.ContentW, m.gly.Trunc)
	}
}

// TestComposerCutAcrossModesAt40 covers the other states the composer's cut
// touches, each through View() at 40 columns (ContentW 38). ui-spec §7.3 says
// every row that is cut ends in the marker; none of these states is exempt, so
// each cut row below is required to end in the marker, to be at most ContentW
// cells, and to keep its lead.
func TestComposerCutAcrossModesAt40(t *testing.T) {
	plainCaps := Caps{Colour: false, Unicode: true}

	t.Run("placeholder", func(t *testing.T) {
		// New, not NewWithFixture: onboarding is an empty transcript. The
		// placeholder is 44 cells, so "> " + it is 46 and is cut.
		m := New(Options{Version: fixtureVersion, Caps: plainCaps})
		m = drive(m, tea.WindowSizeMsg{Width: 40, Height: 24})
		rows, lay := composerRowsOf(t, m, true)
		if cellWidth("> "+Placeholder) <= lay.ContentW {
			t.Fatalf("setup: placeholder row fits %d cells; the case needs it wider", lay.ContentW)
		}
		want := "> Ask anything (Enter to send, /help " + "⋯"
		if rows[0] != want || cellWidth(rows[0]) != lay.ContentW {
			t.Errorf("placeholder row = %q (%d cells), want %q (%d cells)",
				rows[0], cellWidth(rows[0]), want, lay.ContentW)
		}
	})

	t.Run("busy first row", func(t *testing.T) {
		m := composerAt40(t, plainCaps)
		m.comp.SetValue(strings.Repeat("a", 60))
		m.busy = Busy{Active: true, Verb: "thinking"}
		rows, lay := composerRowsOf(t, m, true)
		want := "> " + strings.Repeat("a", 35) + "⋯"
		if rows[0] != want || cellWidth(rows[0]) != lay.ContentW {
			t.Errorf("busy cut row = %q (%d cells), want %q", rows[0], cellWidth(rows[0]), want)
		}
		// Control through the same path: a short busy row is unmarked and
		// still carries the disabled hint.
		m.comp.SetValue("hi")
		rows, _ = composerRowsOf(t, m, true)
		if strings.Contains(rows[0], "⋯") || !strings.HasSuffix(rows[0], "(input disabled, Esc cancels)") {
			t.Errorf("short busy row = %q, want unmarked with the disabled hint", rows[0])
		}
	})

	t.Run("window starts mid-text", func(t *testing.T) {
		// Seven lines, caret on the last: the five-row window is lines 2-6
		// (m5-d6), so no row carries the prompt. Line 2 and 4 are long; line 6,
		// where the caret is, is exactly 36 cells and must not be marked.
		lines := []string{"l0", "l1", strings.Repeat("c", 60), "l3", strings.Repeat("e", 50), "l5", letters(36)}
		m := composerAt40(t, plainCaps)
		m.comp.SetValue(strings.Join(lines, "\n"))
		rows, lay := composerRowsOf(t, m, true)
		if len(rows) != 5 {
			t.Fatalf("composer has %d rows, want 5: %q", len(rows), rows)
		}
		want := []string{
			"  " + strings.Repeat("c", 35) + "⋯", // line 2, cut
			"  l3",                               // control: fits
			"  " + strings.Repeat("e", 35) + "⋯", // line 4, cut
			"  l5",
			"  " + letters(36), // caret line, text fits: caret cell dropped, no marker
		}
		for i := range want {
			if rows[i] != want[i] {
				t.Errorf("row %d = %q, want %q", i, rows[i], want[i])
			}
			if cellWidth(rows[i]) > lay.ContentW {
				t.Errorf("row %d is %d cells, over ContentW", i, cellWidth(rows[i]))
			}
		}
	})

	t.Run("wide rune at the cut", func(t *testing.T) {
		// "> " + 30 two-cell runes is 62 cells. The cut limit is 37, which an
		// odd cell cannot reach with whole wide runes: 17 runes is 36 cells, the
		// 18th would end at 38 > 37. The row is 37 cells (within ContentW), ends
		// in the marker, and never splits a rune.
		m := composerAt40(t, plainCaps)
		m.comp.SetValue(strings.Repeat("漢", 30))
		rows, lay := composerRowsOf(t, m, true)
		want := "> " + strings.Repeat("漢", 17) + "⋯"
		if rows[0] != want {
			t.Errorf("row = %q (%d cells), want %q", rows[0], cellWidth(rows[0]), want)
		}
		if cellWidth(rows[0]) > lay.ContentW {
			t.Errorf("row is %d cells, over ContentW %d", cellWidth(rows[0]), lay.ContentW)
		}
		// Boundary with the caret: 18 wide runes is 36 text cells, which fits
		// (the caret's own cell is dropped, not marked); 19 is cut.
		m.comp.SetValue(strings.Repeat("漢", 18))
		rows, _ = composerRowsOf(t, m, true)
		if rows[0] != "> "+strings.Repeat("漢", 18) {
			t.Errorf("fitting wide row = %q, want all 18 runes and no marker", rows[0])
		}
		m.comp.SetValue(strings.Repeat("漢", 19))
		rows, _ = composerRowsOf(t, m, true)
		if !strings.HasSuffix(rows[0], "⋯") || cellWidth(rows[0]) > lay.ContentW {
			t.Errorf("19 wide runes = %q, want a marked row within %d cells", rows[0], lay.ContentW)
		}
	})
}

// TestComposerCutColour runs the cut with colour on, where the warning style and
// the SGR close actually emit bytes (with Colour false every Style is the
// identity, so none of this is reachable). A cut row must carry the marker in
// the warning colour (palette 215, ui-spec §10.1) and must leave no style open:
// the last escape on the row is the reset. The second case is a row whose text
// fits but whose caret cell is dropped by the hard edge; the escape that opened
// the caret's style has to be closed there too.
func TestComposerCutColour(t *testing.T) {
	m := composerAt40(t, Caps{Colour: true, Unicode: true})

	m.comp.SetValue(strings.Repeat("a", 60))
	rows, _ := composerRowsOf(t, m, false)
	cut := rows[0]
	if !strings.Contains(cut, "\x1b[38;5;215m⋯") {
		t.Errorf("cut row's marker is not in the warning style (palette 215): %q", cut)
	}
	if !strings.HasSuffix(cut, "⋯\x1b[0m") && !strings.HasSuffix(cut, "⋯\x1b[0m\x1b[0m") {
		t.Errorf("cut row does not end marker then reset: %q", cut)
	}
	if got := lastSGR(cut); got != "\x1b[0m" {
		t.Errorf("cut row leaves a style open: last escape %q in %q", got, cut)
	}

	// Fits-but-for-the-caret: 36 cells of text, caret dropped by exactly().
	m.comp.SetValue(letters(36))
	rows, _ = composerRowsOf(t, m, false)
	fit := rows[0]
	if strings.Contains(stripSGR(fit), "⋯") || stripSGR(fit) != "> "+letters(36) {
		t.Errorf("fitting row = %q, want %q unmarked", stripSGR(fit), "> "+letters(36))
	}
	if got := lastSGR(fit); got != "\x1b[0m" {
		t.Errorf("hard-edge row leaves a style open: last escape %q in %q", got, fit)
	}
	if strings.Contains(fit, "215m") {
		t.Errorf("row that fits carries the warning style: %q", fit)
	}
}
