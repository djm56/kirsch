package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// composerRow returns one rendered composer row from a full frame, stripped of
// styling and of the frame's left margin.
//
// It reads the frame rather than calling Composer.rows directly: the defect
// these tests pin is that the caret the *user sees* does not follow the cursor,
// and a test that renders the component in isolation cannot tell the difference
// between a component that never painted it and a frame that never showed it.
func composerRow(t *testing.T, m Model, n int) string {
	t.Helper()
	lines := strings.Split(stripSGR(m.View()), "\n")
	lay := m.layout()
	first := len(lines) - lay.ComposerH
	if first < 0 || n >= lay.ComposerH {
		t.Fatalf("composer row %d does not exist: frame has %d rows, composer is %d",
			n, len(lines), lay.ComposerH)
	}
	return strings.TrimRight(strings.TrimPrefix(lines[first+n], strings.Repeat(" ", lay.Pad)), " ")
}

// composing returns a model in ModeComposing with the fixture's opening
// approval resolved, so keys reach the textarea.
func composing(t *testing.T) Model {
	t.Helper()
	m := drive(newDriven(t), key('y'))
	if m.mode() != ModeComposing {
		t.Fatalf("setup: mode after approve = %v, want Composing", m.mode())
	}
	return m
}

// typed returns a composing model with s entered a rune at a time and the
// cursor then walked back lefts positions from the end.
//
// A rune at a time and not SetValue: SetValue drops the cursor at the end, and
// the cursor's position under arrow-key movement is the whole subject here.
func typed(t *testing.T, m Model, s string, lefts int) Model {
	t.Helper()
	for _, r := range s {
		m = drive(m, key(r))
	}
	for i := 0; i < lefts; i++ {
		m = drive(m, keyType(tea.KeyLeft))
	}
	return m
}

// TestTheCaretOverstrikesAndNeverWidensTheRow pins both halves of where the
// caret goes: which character it marks, and that marking it costs no column.
//
// Two defects have shipped here, and this test is written to fail against each.
//
//  1. The caret was appended to the last line, so the only position it could
//     report was end-of-input while the cursor moved freely beneath it. Every
//     row below whose caret is not at the end fails that rendering.
//  2. The caret was then inserted at the cursor, which fixed the position and
//     broke the width: given a cell of its own it shifted every following
//     character one column right, so `test` with the cursor in the middle read
//     as `te▌st` — a gap the user could see. Invisible at end-of-input, where
//     nothing follows, and wrong at every other position. The Width column
//     below is what catches it; the row strings alone would not, which is
//     precisely how that defect got past a test asserting only the caret's
//     index.
//
// So the width is asserted explicitly and separately rather than left implicit
// in the expected string. A reader can see that it was considered, and a future
// expectation edited to match new output has to be edited twice, in two
// different notations, before it will agree with a wrong renderer.
//
// The expected rows are written out by hand rather than computed from the same
// head/caret/tail rule the renderer uses. A test that rebuilds the production
// expression agrees with it by construction and proves nothing.
func TestTheCaretOverstrikesAndNeverWidensTheRow(t *testing.T) {
	base := composing(t)
	caret := base.gly.Caret

	// 59 runes, comfortably past the 40 columns the textarea soft-wraps at
	// internally. The composer never calls SetWidth, so that wrap is invisible
	// in the output but very much present in the LineInfo the caret's column is
	// read from — column 45 exercises the second wrapped row, column 0 the
	// first, and neither may change what is drawn.
	const long = "the quick brown fox jumps over the lazy dog and keeps going"

	// atEnd is stated per row rather than inferred from lefts. Inferring it was
	// a real defect: `lefts == 0` is a property of how the case is *driven*, and
	// a row added later that reaches end-of-input some other way — or one with
	// `lefts: 0` whose text ends in a newline, so the cursor is at the end of a
	// line that is not the end of the input — would be silently handed the wrong
	// expectation by the very test meant to catch it. The subtest cross-checks
	// the claim against the cursor, so a mis-stated row fails loudly instead of
	// weakening the assertion below it.
	cases := []struct {
		name  string
		text  string
		lefts int
		want  string // the whole composer row, prompt included
		width int    // cells, prompt excluded
		atEnd bool   // the cursor is past the last character of the input
	}{
		// `test` is 4 cells. Every row but the last is 4 cells too: the caret
		// stands in a character's cell instead of taking one of its own.
		{"end of input", "test", 0, "> test" + caret, 5, true},
		{"on the last rune", "test", 1, "> tes" + caret, 4, false},
		{"mid line", "test", 2, "> te" + caret + "t", 4, false},
		{"second rune", "test", 3, "> t" + caret + "st", 4, false},
		{"column 0", "test", 4, "> " + caret + "est", 4, false},
		// Left at column 0 is a no-op, not a wrap to the previous line.
		{"column 0, held", "test", 6, "> " + caret + "est", 4, false},

		{
			"long line, column 45", long, 59 - 45,
			"> the quick brown fox jumps over the lazy dog a" + caret + "d keeps going", 59, false,
		},
		{
			"long line, column 40", long, 59 - 40,
			"> the quick brown fox jumps over the lazy " + caret + "og and keeps going", 59, false,
		},
		{
			"long line, column 39", long, 59 - 39,
			"> the quick brown fox jumps over the lazy" + caret + "dog and keeps going", 59, false,
		},
		{
			"long line, column 0", long, 59,
			"> " + caret + "he quick brown fox jumps over the lazy dog and keeps going", 59, false,
		},
		{"long line, end", long, 0, "> " + long + caret, 60, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := typed(t, composing(t), tc.text, tc.lefts)

			if got := composerRow(t, m, 0); got != tc.want {
				t.Errorf("composer row = %q, want %q", got, tc.want)
			}
			// Width of the row's body, measured independently of the string
			// above. `> ` is the prompt and is not part of the text.
			body := strings.TrimPrefix(composerRow(t, m, 0), "> ")
			if got := cellWidth(body); got != tc.width {
				t.Errorf("composer row is %d cells wide, want %d: the caret must "+
					"stand in a character's cell, not take one of its own "+
					"(text is %d cells)", got, tc.width, cellWidth(tc.text))
			}
			// The table's atEnd claim, checked against the model before anything
			// is derived from it.
			if _, col := m.comp.cursor(); (col >= len([]rune(tc.text))) != tc.atEnd {
				t.Fatalf("the table says atEnd=%v, but the cursor is at column %d of %d runes",
					tc.atEnd, col, len([]rune(tc.text)))
			}
			// The text is exactly as wide as the row at every position except
			// end-of-input, where there is no character to stand in for. Stated
			// as its own relation so the table cannot quietly drift into
			// agreeing with a renderer that widens every row by one.
			slack := tc.width - cellWidth(tc.text)
			if want := map[bool]int{true: 1, false: 0}[tc.atEnd]; slack != want {
				t.Errorf("row is %d cells wider than its text, want %d "+
					"(end of input: %v)", slack, want, tc.atEnd)
			}
			// A caret in the right column with the wrong value would be a
			// textarea that had stopped moving its cursor and started eating
			// characters instead.
			if got := m.comp.Value(); got != tc.text {
				t.Errorf("moving the cursor changed the value to %q, want %q", got, tc.text)
			}
		})
	}
}

// TestTheCaretStandsInAWideRunesCells covers the other direction the width can
// go wrong.
//
// Overstriking a two-cell rune with a one-cell caret pulls every following
// character one column left, which is the gap defect mirrored. The trailing `x`
// is load-bearing: composerRow right-trims, so a caret padded at the very end
// of a line would have its padding trimmed away and the row would measure short
// for a reason that is not a rendering fault.
func TestTheCaretStandsInAWideRunesCells(t *testing.T) {
	caret := composing(t).gly.Caret

	for _, tc := range []struct {
		name  string
		lefts int
		want  string
	}{
		{"on the second ideograph", 2, "> 漢" + caret + " x"},
		{"on the first ideograph", 3, "> " + caret + " 字x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := typed(t, composing(t), "漢字x", tc.lefts)
			got := composerRow(t, m, 0)
			if got != tc.want {
				t.Errorf("composer row = %q, want %q", got, tc.want)
			}
			body := strings.TrimPrefix(got, "> ")
			if w, want := cellWidth(body), cellWidth("漢字x"); w != want {
				t.Errorf("composer row is %d cells wide, want %d", w, want)
			}
		})
	}
}

// composingCaps is composing() with the terminal's capabilities stated rather
// than defaulted, for the tests whose subject is what the ASCII fallback draws.
func composingCaps(t *testing.T, caps Caps) Model {
	t.Helper()
	m := NewWithFixture(Options{Version: fixtureVersion, Caps: caps})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24}, key('y'))
	if m.mode() != ModeComposing {
		t.Fatalf("setup: mode after approve = %v, want Composing", m.mode())
	}
	return m
}

// TestTheCaretStandsInAWholeGraphemeCluster covers the third way the width has
// gone wrong, and the one the operator reported: a visible gap after the cursor
// on ordinary pasted text.
//
// A rune is not a character. `👍🏽` is an emoji plus a skin-tone modifier, `👨‍👩`
// is two emoji joined by U+200D, `❤️` is a heart plus a variation selector, and
// `é` arrives from most keyboards and most of the web as `e` + U+0301. Every one
// of them is one character to the reader and several runes to the textarea,
// whose cursor column counts runes — so a caret placed by rune index lands
// *inside* a character, and two separate faults follow:
//
//  1. The pad was measured out of context. The renderer measured the single rune
//     under the cursor, and a lone rune has no cluster to be measured in:
//     U+1F3FD measures 2 on its own and contributes 0 inside `👍🏽`, so the caret
//     was padded by two cells for a character already exactly as wide as it was.
//  2. Replacing one rune of a cluster destroys the cluster. The survivors no
//     longer join and re-render at their standalone widths — `👨`(2) beside
//     `👩`(2) where the joined pair was 2 — and no padding brings that back.
//
// Each row below is a position the old renderer got wrong by between one and
// three cells. The assertion that catches all of it is the same one the plain
// test makes: the row is exactly as wide as its text everywhere but
// end-of-input.
func TestTheCaretStandsInAWholeGraphemeCluster(t *testing.T) {
	caret := composing(t).gly.Caret

	// The invisible runes are written as escapes. A variation selector, a ZWJ
	// and a combining acute are the whole subject here and none of them shows up
	// in a diff, an editor's gutter, or a reviewer's terminal.
	for _, tc := range []struct {
		name string
		text string
		// rows[n] is the whole composer row — prompt included — with the cursor
		// n positions left of the end.
		rows []string
	}{
		{
			"variation selector", "❤\ufe0fx", // ❤️x
			[]string{
				"> ❤\ufe0fx" + caret,
				"> ❤\ufe0f" + caret,
				"> " + caret + "x",
				"> " + caret + "x",
			},
		},
		{
			"variation selector, narrow base", "✔\ufe0fx", // ✔️x
			[]string{
				"> ✔\ufe0fx" + caret,
				"> ✔\ufe0f" + caret,
				"> " + caret + "x",
				"> " + caret + "x",
			},
		},
		{
			// The caret is padded to the cluster's two cells, so the row keeps
			// its width instead of losing one.
			"emoji modifier", "\U0001f44d\U0001f3fdx", // 👍🏽x
			[]string{
				"> \U0001f44d\U0001f3fdx" + caret,
				"> \U0001f44d\U0001f3fd" + caret,
				"> " + caret + " x",
				"> " + caret + " x",
			},
		},
		{
			// Three runes, one character. Overstriking any one of them used to
			// split the pair and widen the row by two cells or three.
			"zero-width joiner", "\U0001f468\u200d\U0001f469x", // 👨‍👩x
			[]string{
				"> \U0001f468\u200d\U0001f469x" + caret,
				"> \U0001f468\u200d\U0001f469" + caret,
				"> " + caret + " x",
				"> " + caret + " x",
				"> " + caret + " x",
			},
		},
		{
			"combining mark", "a\u0301bc", // ábc, decomposed
			[]string{
				"> a\u0301bc" + caret,
				"> a\u0301b" + caret,
				"> a\u0301" + caret + "c",
				"> " + caret + "bc",
				"> " + caret + "bc",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := typed(t, composing(t), tc.text, 0)
			if got := m.comp.Value(); got != tc.text {
				t.Fatalf("typing produced %q, want %q: the textarea is not holding the "+
					"text this case is about", got, tc.text)
			}
			if n := len([]rune(tc.text)) + 1; len(tc.rows) != n {
				t.Fatalf("the case lists %d rows for %d cursor positions", len(tc.rows), n)
			}

			for lefts, want := range tc.rows {
				got := composerRow(t, m, 0)
				if got != want {
					t.Errorf("%d left of the end: composer row = %q, want %q", lefts, got, want)
				}
				// The relation, asserted independently of the string above: the
				// row is its text's width everywhere but end-of-input, where the
				// caret has nothing to stand in for. This is the half that fails
				// against a renderer whose caret is in the right place and the
				// wrong cell.
				body := strings.TrimPrefix(got, "> ")
				slack := cellWidth(body) - cellWidth(tc.text)
				want := map[bool]int{true: 1, false: 0}[lefts == 0]
				if slack != want {
					t.Errorf("%d left of the end: row is %d cells wider than its text, want %d "+
						"(row %q is %d cells, text %q is %d)",
						lefts, slack, want, body, cellWidth(body), tc.text, cellWidth(tc.text))
				}
				m = drive(m, keyType(tea.KeyLeft))
			}
		})
	}
}

// TestTheCaretIsVisibleOnEveryClusterItCanSitOn is the property a fixed glyph
// cannot have: wherever the cursor is, something is on screen that would not be
// there without it.
//
// Overstriking loses the cursor completely when the glyph drawn equals the text
// it covers — the row comes out byte-identical to the plain text. In ASCII the
// caret is `_`, and the composer of a coding agent is typed full of `_`:
// `parse_slash`, `snake_case_name`, `/read my_file.go`. The ASCII table is not a
// corner either; unicodeLocale reports false whenever LC_ALL, LC_CTYPE and LANG
// are all unset, which is the default in `docker run`, in a systemd unit, under
// cron and on many CI runners. The earlier caret test missed all of it by only
// ever typing `test`, which contains no glyph from either table.
//
// The check is deliberately a sweep and not a table of expected strings. A
// table is written against the glyph in force when it was written, so swapping
// one glyph for another moves the defect onto text the table stopped covering
// and every case still passes. Sweeping every cursor position of every text —
// each text built from the mode's own caret, or drawn from the inputs measured
// on the defect — asserts the thing that has to be true rather than the strings
// that happen to be true today.
func TestTheCaretIsVisibleOnEveryClusterItCanSitOn(t *testing.T) {
	for _, mode := range []struct {
		name string
		caps Caps
	}{
		{"colour and unicode", Caps{Colour: true, Unicode: true}},
		{"NO_COLOR", Caps{Colour: false, Unicode: true}},
		{"ASCII fallback", Caps{Colour: false, Unicode: false}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			g := composingCaps(t, mode.caps).gly

			// The whole argument rests on the two glyphs differing. If they ever
			// stop differing the sweep below still passes on most inputs and
			// fails obscurely on the rest, so it is checked first and fatally.
			if g.CaretAlt == g.Caret {
				t.Fatalf("the caret's stand-in is the caret itself (%q), so it cannot be "+
					"visible over it", g.Caret)
			}

			// The mechanism, pinned exactly once: on its own glyph the caret
			// draws the stand-in, in the same cell, at the same width.
			m := typed(t, composingCaps(t, mode.caps), "a"+g.Caret+"b", 2)
			if got, want := composerRow(t, m, 0), "> a"+g.CaretAlt+"b"; got != want {
				t.Errorf("composer row = %q, want %q", got, want)
			}

			for _, text := range []string{
				"a" + g.Caret + "b",
				g.Caret + g.Caret + g.Caret,
				"parse_slash",
				"snake_case_name",
				"/read my_file.go",
				"internal/tui/view_smoke_test.go",
			} {
				t.Run(text, func(t *testing.T) {
					m := typed(t, composingCaps(t, mode.caps), text, 0)
					n := len([]rune(text))
					for lefts := 0; lefts <= n; lefts++ {
						body := strings.TrimPrefix(composerRow(t, m, 0), "> ")
						if body == text {
							t.Errorf("%d left of the end: the row renders as its own plain "+
								"text %q — the caret %q is standing on a copy of itself and "+
								"there is no cursor on screen at all", lefts, text, g.Caret)
						}
						// Visible is not enough on its own: a caret that is
						// visible because it widened the row is the gap defect
						// coming back the other way.
						if lefts > 0 {
							if got, want := cellWidth(body), cellWidth(text); got != want {
								t.Errorf("%d left of the end: row %q is %d cells, want %d",
									lefts, body, got, want)
							}
						}
						m = drive(m, keyType(tea.KeyLeft))
					}
				})
			}
		})
	}
}

// TestTheCaretIsVisibleInEveryCapabilityMode checks the three renderings a
// terminal can ask for.
//
// This is the constraint that decided the mechanism. Reverse video would keep
// the character under the cursor readable, but NewStyles returns the identity
// function for every Style when colour is off (styles.go), so a reverse-video
// marker routed through the palette renders nothing at all and the cursor
// disappears. Emitting the escape directly instead is worse than it looks:
// DetectCaps turns colour off when stdout is not a character device, so colour
// being off also means the frame is going into a pipe or a file. A glyph is the
// only marker that survives all three modes, and it survives them identically.
func TestTheCaretIsVisibleInEveryCapabilityMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		caps Caps
	}{
		{"colour and unicode", Caps{Colour: true, Unicode: true}},
		{"NO_COLOR", Caps{Colour: false, Unicode: true}},
		{"ASCII fallback", Caps{Colour: false, Unicode: false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewWithFixture(Options{Version: fixtureVersion, Caps: tc.caps})
			m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24}, key('y'))
			if m.mode() != ModeComposing {
				t.Fatalf("setup: mode after approve = %v, want Composing", m.mode())
			}
			m = typed(t, m, "test", 2)

			caret := m.gly.Caret
			if got, want := composerRow(t, m, 0), "> te"+caret+"t"; got != want {
				t.Errorf("composer row = %q, want %q", got, want)
			}
			// The caret has to survive stripSGR, which is what proves it is a
			// glyph in the frame's text and not a colour that a no-colour
			// terminal would drop.
			body := strings.TrimPrefix(composerRow(t, m, 0), "> ")
			if !strings.Contains(body, caret) {
				t.Errorf("composer row %q carries no caret %q", body, caret)
			}
			if got := cellWidth(body); got != 4 {
				t.Errorf("composer row is %d cells wide, want 4 (text is `test`)", got)
			}
		})
	}
}

// TestCaretFollowsTheCursorOntoASecondLine covers the row half of the position.
//
// The column alone is not enough: the caret has to be drawn on the line the
// cursor is on, and a fix that placed it by column while still choosing the
// last line would paint it into the wrong row of a multi-line composer.
func TestCaretFollowsTheCursorOntoASecondLine(t *testing.T) {
	m := composing(t)
	for _, r := range "ab" {
		m = drive(m, key(r))
	}
	m = drive(m, keyType(tea.KeyCtrlJ))
	for _, r := range "cd" {
		m = drive(m, key(r))
	}
	caret := m.gly.Caret

	m = drive(m, keyType(tea.KeyLeft))
	if got, want := composerRow(t, m, 0), "> ab"; got != want {
		t.Errorf("first composer row = %q, want %q: the caret belongs on the cursor's line", got, want)
	}
	// `c▌` and not `c▌d`: the caret stands in `d`'s cell rather than beside it,
	// so the row stays exactly as wide as the line it draws.
	if got, want := composerRow(t, m, 1), "  c"+caret; got != want {
		t.Errorf("second composer row = %q, want %q", got, want)
	}
	if got, want := cellWidth(strings.TrimPrefix(composerRow(t, m, 1), "  ")), 2; got != want {
		t.Errorf("second composer row is %d cells wide, want %d (the line is `cd`)", got, want)
	}

	// Up puts the cursor on the first line; the caret has to move with it.
	m = drive(m, keyType(tea.KeyUp))
	if m.mode() != ModeComposing {
		t.Fatalf("Up from the second line left Composing (mode = %v)", m.mode())
	}
	if got := composerRow(t, m, 0); !strings.Contains(got, caret) {
		t.Errorf("first composer row = %q, want the caret on it after Up", got)
	}
	if got := composerRow(t, m, 1); strings.Contains(got, caret) {
		t.Errorf("second composer row = %q, want no caret on it after Up", got)
	}
}
