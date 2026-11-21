package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for the transcript scrollbar (ui-spec §2.1, §2.4, amendment 103).
//
// Every frame here comes from View(), the entry point the terminal runs. The
// thumb maths is also called directly (scrollbarThumb) so its edge cases can be
// hit without building a frame for each, and a test says which of the two it
// uses.

// barUnicode, barASCII and barColour are the capability sets the bar tests build
// their models with: Unicode glyphs, ASCII glyphs, and Unicode glyphs with colour.
var (
	barUnicode = Caps{Unicode: true}
	barASCII   = Caps{Unicode: false}
	barColour  = Caps{Unicode: true, Colour: true}
)

// heightFor finds a terminal height at the given width whose transcript region
// is exactly th rows, and fails when there is none. Heights are searched rather
// than computed so the test does not repeat computeLayout's chrome budget.
func heightFor(t *testing.T, w, th int) int {
	t.Helper()
	for h := 1; h <= 80; h++ {
		if headerModel(t, defaultSession(), w, h).layout().TranscriptH == th {
			return h
		}
	}
	t.Fatalf("no terminal height at width %d gives a %d-row transcript region", w, th)
	return 0
}

// barModel builds a model at w×h whose transcript holds exactly n lines: one
// assistant reply of n-2 short rows, between the two blank rows relayout adds. It
// sets the size with Update, appends the reply with tr.Append and flattens it
// with relayout, and fails if the line count is not n.
func barModel(t *testing.T, caps Caps, w, h, n int) Model {
	t.Helper()
	if n < 3 {
		t.Fatalf("barModel needs n >= 3, got %d", n)
	}
	m := New(Options{
		Version: fixtureVersion,
		Caps:    caps,
		Session: defaultSession(),
		Status:  Status{Model: "claude-sonnet-5-5", Family: "sonnet-5-5"},
	})
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = mm.(Model)
	body := make([]string, n-2)
	for i := range body {
		body[i] = fmt.Sprintf("row %d", i)
	}
	m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{Lines: body}})
	m.relayout(m.layout())
	if len(m.lines) != n {
		t.Fatalf("barModel: transcript has %d lines, want %d", len(m.lines), n)
	}
	return m
}

// barCells returns, for each transcript row of m's frame, the last terminal
// column with its escapes stripped: the right margin cell. A row that stops short
// of the terminal width has "" there, which is how "no bar" reads.
func barCells(t *testing.T, m Model) []string {
	t.Helper()
	lay := m.layout()
	rows := strings.Split(m.View(), "\n")
	start := lay.HeaderH()
	if start+lay.TranscriptH > len(rows) {
		t.Fatalf("frame has %d rows, transcript band ends at %d", len(rows), start+lay.TranscriptH)
	}
	cells := make([]string, 0, lay.TranscriptH)
	for _, row := range rows[start : start+lay.TranscriptH] {
		plain := stripSGR(row)
		if cellWidth(plain) < lay.TermW {
			cells = append(cells, "")
			continue
		}
		r := []rune(plain)
		cells = append(cells, string(r[len(r)-1]))
	}
	return cells
}

// wantBar renders an expected bar for a track of h rows with the thumb at
// [top, top+size), using the Unicode glyphs.
func wantBar(h, top, size int) []string {
	out := make([]string, h)
	for r := range out {
		if r >= top && r < top+size {
			out[r] = "┃"
		} else {
			out[r] = "│"
		}
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// barDrawnIn reports whether any transcript row of the frame ends in a bar cell.
func barDrawnIn(t *testing.T, m Model) bool {
	t.Helper()
	for _, c := range barCells(t, m) {
		if c != "" {
			return true
		}
	}
	return false
}

// TestScrollbarThumbMaths calls scrollbarThumb directly and pins its output to
// numbers worked by hand from the formulas in ui-spec §2.4, not from the code. The
// first block is the spec's own worked examples; the rest are the edges: the
// smallest overflow, an offset outside [0, maxOff] on both sides, and the
// smallest region the bar is drawn in.
func TestScrollbarThumbMaths(t *testing.T) {
	cases := []struct {
		name          string
		h, n, off     int
		wantTop, size int
	}{
		// §2.4: h 10, n 40, maxOff 30, thumbH 2.
		{"spec h10 n40 off0", 10, 40, 0, 0, 2},
		{"spec h10 n40 off1 rounds to the top", 10, 40, 1, 0, 2},
		{"spec h10 n40 off15", 10, 40, 15, 4, 2},
		{"spec h10 n40 off29 is not pinned but sits at the bottom", 10, 40, 29, 8, 2},
		{"spec h10 n40 off30 pinned", 10, 40, 30, 8, 2},
		// §2.4: h 12, n 100, maxOff 88, thumbH 1.
		{"spec h12 n100 off44", 12, 100, 44, 6, 1},
		{"spec h12 n100 off88 pinned", 12, 100, 88, 11, 1},
		// Screen 14: h 8, n 24, offset 8 of 16.
		{"screen14 h8 n24 off8", 8, 24, 8, 3, 2},
		// n = h+1: maxOff 1, so maxOff/2 is 0 and the thumb has two positions.
		{"n=h+1 off0", 10, 11, 0, 0, 9},
		{"n=h+1 off1", 10, 11, 1, 1, 9},
		// Offsets outside [0, maxOff] clamp to the nearest end.
		{"offset above maxOff clamps", 10, 40, 99, 8, 2},
		{"offset exactly maxOff", 10, 40, 30, 8, 2},
		{"negative offset clamps", 10, 40, -4, 0, 2},
		// h 3 is the least height the bar is drawn at; n 10 gives thumbH
		// max(1, 9/10) = 1 and maxOff 7.
		{"h3 n10 off0", 3, 10, 0, 0, 1},
		{"h3 n10 off3", 3, 10, 3, 1, 1},
		{"h3 n10 off7", 3, 10, 7, 2, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			top, size := scrollbarThumb(c.h, c.n, c.off)
			if top != c.wantTop || size != c.size {
				t.Errorf("scrollbarThumb(h=%d, n=%d, off=%d) = top %d, size %d; want top %d, size %d",
					c.h, c.n, c.off, top, size, c.wantTop, c.size)
			}
		})
	}
}

// TestScrollbarThumbOnTheFrame drives the same worked examples through View() and
// reads the bar back out of the right margin column, so the glyph rows, the
// transcript height and the stored offset are all the real ones.
func TestScrollbarThumbOnTheFrame(t *testing.T) {
	cases := []struct {
		name          string
		th, n, off    int
		wantTop, size int
	}{
		{"h10 n40 off0", 10, 40, 0, 0, 2},
		{"h10 n40 off15", 10, 40, 15, 4, 2},
		{"h10 n40 off29", 10, 40, 29, 8, 2},
		{"h10 n40 off30", 10, 40, 30, 8, 2},
		{"h12 n100 off44", 12, 100, 44, 6, 1},
		{"h12 n100 off88", 12, 100, 88, 11, 1},
		{"h8 n24 off8 (screen 14)", 8, 24, 8, 3, 2},
		{"h10 n11 off1 (n=h+1)", 10, 11, 1, 1, 9},
		{"h3 n10 off7 (smallest region)", 3, 10, 7, 2, 1},
		// A stored offset above maxOff: the thumb stays inside the track.
		{"h10 n40 stored offset 99", 10, 40, 99, 8, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := barModel(t, barUnicode, 80, heightFor(t, 80, c.th), c.n)
			if got := m.layout().TranscriptH; got != c.th {
				t.Fatalf("transcript region is %d rows, want %d", got, c.th)
			}
			m.scroll = Scroll{Offset: c.off}
			got := barCells(t, m)
			want := wantBar(c.th, c.wantTop, c.size)
			if !sameStrings(got, want) {
				t.Errorf("bar column\n  got  %q\n  want %q", got, want)
			}
		})
	}
}

// TestScrollbarDrawingConditions walks each §2.1 drawing condition with the bar
// present on one side and absent on the other, through View(). Every absent case
// is paired with a control frame that does draw the bar (DIR-026), so an absence
// cannot come from a frame that was never going to have one. For all but the
// narrow band the two differ in that one condition. The narrow band is reached
// two ways, and the subtest says how each differs from its control.
func TestScrollbarDrawingConditions(t *testing.T) {
	const w = 80
	h8 := heightFor(t, w, 8)

	t.Run("overflow draws, a transcript that fits does not", func(t *testing.T) {
		over := barModel(t, barUnicode, w, h8, 9)
		fits := barModel(t, barUnicode, w, h8, 8)
		if !barDrawnIn(t, over) {
			t.Error("control: n = h+1 should draw the bar")
		}
		if barDrawnIn(t, fits) {
			t.Error("n = h drew a bar; there is nothing to scroll")
		}
		if n := barModel(t, barUnicode, w, h8, 3); barDrawnIn(t, n) {
			t.Error("a short transcript drew a bar")
		}
	})

	t.Run("an open modal hides the bar", func(t *testing.T) {
		tall := heightFor(t, w, 12)
		for name, st := range map[string]ModalState{
			"help": {Kind: ModalHelp, Title: "help", Lines: helpLines()},
			"diff": {Kind: ModalDiff, Title: "calc/divide.go", Lines: fakeDiff(), Added: 12, Removed: 4},
		} {
			closed := barModel(t, barUnicode, w, tall, 40)
			if !barDrawnIn(t, closed) {
				t.Fatalf("%s: control: the frame with the modal closed should draw the bar", name)
			}
			open := closed
			open.openModal(st)
			open.relayout(open.layout())
			if len(open.lines) <= open.layout().TranscriptH {
				t.Fatalf("%s: control: the transcript no longer overflows with the modal open", name)
			}
			if barDrawnIn(t, open) {
				t.Errorf("%s modal open: the bar is still drawn", name)
			}
		}
	})

	t.Run("an open confirm prompt hides the bar", func(t *testing.T) {
		tall := heightFor(t, w, 12)
		closed := barModel(t, barUnicode, w, tall, 40)
		if !barDrawnIn(t, closed) {
			t.Fatal("control: the frame with the prompt closed should draw the bar")
		}
		open := closed
		open.askConfirm(ConfirmState{Prompt: "Discard the running turn?", Action: ConfirmNewSession})
		open.relayout(open.layout())
		if barDrawnIn(t, open) {
			t.Error("confirm prompt open: the bar is still drawn")
		}
	})

	t.Run("a region of 3 rows draws and a region of 2 does not", func(t *testing.T) {
		h3, h2 := heightFor(t, w, 3), heightFor(t, w, 2)
		if !barDrawnIn(t, barModel(t, barUnicode, w, h3, 10)) {
			t.Error("a 3-row transcript region should draw the bar")
		}
		if barDrawnIn(t, barModel(t, barUnicode, w, h2, 10)) {
			t.Error("a 2-row transcript region drew the bar")
		}
	})

	t.Run("the narrow band draws no transcript and no bar", func(t *testing.T) {
		const h = 24
		wide := barModel(t, barUnicode, 40, h, 60)
		if !barDrawnIn(t, wide) {
			t.Fatal("control: a 40-column terminal with an overflowing transcript should draw the bar")
		}
		// The reachable frame, built with Update. Not barModel: below 40 columns
		// relayout drops the transcript lines, so this frame has n = 0 and the
		// transcript-has-lines condition keeps the bar out on its own. It says
		// nothing about the ShowTranscript clause; the next case does.
		m := New(Options{
			Version: fixtureVersion, Caps: barUnicode, Session: defaultSession(),
			Status: Status{Model: "claude-sonnet-5-5", Family: "sonnet-5-5"},
		})
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 39, Height: h})
		m = mm.(Model)
		if m.layout().ShowTranscript {
			t.Fatal("control: a 39-column terminal should be in the narrow band")
		}
		if barDrawnIn(t, m) {
			t.Error("a 39-column terminal drew a bar")
		}

		// The ShowTranscript clause alone. Update never leaves a model with lines
		// below 40 columns, so this state is built directly: the 40-column
		// control with its width set to 39 and no relayout, which keeps its 60
		// lines. Everything but the width is the control's, so the layout's
		// ShowTranscript is the only condition that differs from it.
		narrow := wide
		narrow.width = 39
		lay := narrow.layout()
		if lay.ShowTranscript {
			t.Fatal("control: width 39 should be in the narrow band")
		}
		if lay.TranscriptH != wide.layout().TranscriptH || lay.TranscriptH < 3 || len(narrow.lines) <= lay.TranscriptH {
			t.Fatalf("control: the narrow frame must still overflow a region of the same height; "+
				"TranscriptH %d (control %d), %d lines", lay.TranscriptH, wide.layout().TranscriptH, len(narrow.lines))
		}
		if barDrawnIn(t, narrow) {
			t.Error("a narrow-band frame whose model still holds overflowing lines drew a bar")
		}
	})
}

// TestScrollbarNotDrawnOverIntroRows sweeps notice counts, terminal sizes and both
// intro kinds, through View(), and checks the §2.1 intro condition: where the
// onboarding or empty-state rows are showing (showIntro) the bar is not drawn,
// even when the transcript has more lines than rows.
//
// Such frames exist only in a short region, where the intro's own rows are cut
// away to nothing (introRows trims them) and the notices alone fill it. The sweep
// has two positive controls. It must reach frames where the intro shows over an
// overflowing transcript, or the absence would hold on an empty set. And it must
// reach overflowing frames where the intro does not show, in the same sweep, and
// find the bar drawn on every one, so a bar that is never drawn for notices fails.
func TestScrollbarNotDrawnOverIntroRows(t *testing.T) {
	introOverflow, otherOverflow := 0, 0
	for _, w := range []int{40, 60, 80} {
		for h := 6; h <= 40; h++ {
			for notices := 0; notices <= 30; notices++ {
				for _, onboarding := range []bool{false, true} {
					m := headerModel(t, defaultSession(), w, h)
					if onboarding {
						m.onboarding = &OnboardingState{NoAPIKey: true, KeyVars: [2]string{"A", "B"}}
					}
					for i := 0; i < notices; i++ {
						m.tr.Append(Item{Kind: KindNotice, Notice: &NoticeCard{Text: fmt.Sprintf("notice %d", i)}})
					}
					m.relayout(m.layout())
					lay := m.layout()
					if lay.TranscriptH < 3 || len(m.lines) <= lay.TranscriptH {
						continue
					}
					drawn := barDrawnIn(t, m)
					if m.showIntro(lay) {
						introOverflow++
						if drawn {
							t.Errorf("%dx%d, %d notices, onboarding %v: the bar is drawn over intro rows",
								w, h, notices, onboarding)
						}
					} else {
						otherOverflow++
						if !drawn {
							t.Errorf("%dx%d, %d notices, onboarding %v: overflowing notices, no intro, and no bar",
								w, h, notices, onboarding)
						}
					}
				}
			}
		}
	}
	if introOverflow == 0 || otherOverflow == 0 {
		t.Fatalf("the sweep saw %d overflowing intro frames and %d overflowing non-intro frames; "+
			"a side it never reaches is a side it does not cover", introOverflow, otherOverflow)
	}
	t.Logf("%d overflowing intro frames, %d overflowing non-intro frames", introOverflow, otherOverflow)
}

// TestScrollbarASCIIGlyphs draws the bar on a non-UTF-8 terminal: `|` for the
// track and `#` for the thumb, with the thumb where the maths puts it.
func TestScrollbarASCIIGlyphs(t *testing.T) {
	m := barModel(t, barASCII, 80, heightFor(t, 80, 10), 40)
	m.scroll = Scroll{Offset: 15}
	got := barCells(t, m)
	want := wantBar(10, 4, 2)
	for i := range want {
		want[i] = map[string]string{"│": "|", "┃": "#"}[want[i]]
	}
	if !sameStrings(got, want) {
		t.Errorf("ASCII bar column\n  got  %q\n  want %q", got, want)
	}
	// Control: the same frame on a UTF-8 terminal uses the Unicode pair, so the
	// ASCII result comes from the glyph set and not from a bar that is always `|#`.
	u := barModel(t, barUnicode, 80, heightFor(t, 80, 10), 40)
	u.scroll = Scroll{Offset: 15}
	if got := barCells(t, u); !sameStrings(got, wantBar(10, 4, 2)) {
		t.Errorf("Unicode bar column\n  got  %q\n  want %q", got, wantBar(10, 4, 2))
	}
}

// TestScrollbarColourOffEmitsNoEscapes renders a frame with the bar and no
// colour: the glyphs are there and the frame carries no SGR sequence at all. The
// control is the same state with colour on, which must carry escapes on the same
// cells, so the absence is not an artefact of a frame that never styles anything.
func TestScrollbarColourOffEmitsNoEscapes(t *testing.T) {
	h := heightFor(t, 80, 10)
	off := barModel(t, barUnicode, 80, h, 40)
	off.scroll = Scroll{Offset: 15}
	view := off.View()
	if strings.ContainsRune(view, 0x1b) {
		t.Error("a colour-off frame with the bar carries an escape sequence")
	}
	if got := barCells(t, off); !sameStrings(got, wantBar(10, 4, 2)) {
		t.Errorf("colour off: bar glyphs\n  got  %q\n  want %q", got, wantBar(10, 4, 2))
	}

	on := barModel(t, barColour, 80, h, 40)
	on.scroll = Scroll{Offset: 15}
	rows := strings.Split(on.View(), "\n")
	band := rows[on.layout().HeaderH() : on.layout().HeaderH()+10]
	track, thumb := on.sty.Dim("│"), on.sty.Muted("┃")
	if track == "│" || thumb == "┃" {
		t.Fatal("control: the colour palette is inert, so this test cannot tell styled from plain")
	}
	if strings.ReplaceAll(thumb, "┃", "│") == track {
		t.Fatal("control: track and thumb share a style, so the roles are not distinct")
	}
	for r, row := range band {
		want := track
		if r == 4 || r == 5 {
			want = thumb
		}
		if !strings.HasSuffix(row, want) {
			t.Errorf("colour on, transcript row %d: does not end in %q: %q", r, want, row)
		}
	}
}

// TestScrollbarMeetsTheNewIndicator checks that the `↓ N new` row, which ends at
// the last content column, sits flush against the bar: the indicator's last
// character is directly followed by the glyph, and the row is as wide as the
// terminal. The other transcript rows in the same frame are the control that the
// bar column is read from the right place.
func TestScrollbarMeetsTheNewIndicator(t *testing.T) {
	const th = 8
	m := barModel(t, barUnicode, 80, heightFor(t, 80, th), 24)
	m.scroll = Scroll{Offset: 8, NewSince: 4}
	lay := m.layout()
	rows := strings.Split(m.View(), "\n")
	last := stripSGR(rows[lay.HeaderH()+th-1])
	if cellWidth(last) != lay.TermW {
		t.Errorf("last transcript row is %d cells, want %d: %q", cellWidth(last), lay.TermW, last)
	}
	// Offset 8 of 16 puts the thumb on transcript rows 3 and 4, so the last row
	// (7) is a track cell.
	if !strings.HasSuffix(last, "↓ 4 new│") {
		t.Errorf("the indicator is not flush against the track glyph: %q", last)
	}
	first := stripSGR(rows[lay.HeaderH()])
	if strings.Contains(first, "new") || cellWidth(first) != lay.TermW {
		t.Errorf("control: the first transcript row should be a plain bar row: %q", first)
	}
}

// TestScrollbarFramePadding checks the padding half of ui-spec §2.1: on a frame
// that draws the bar every transcript row is as wide as the terminal, and on a
// frame that does not, no transcript row is padded, so none ends in whitespace.
func TestScrollbarFramePadding(t *testing.T) {
	h := heightFor(t, 80, 8)
	with := barModel(t, barUnicode, 80, h, 24)
	without := barModel(t, barUnicode, 80, h, 8)
	lay := with.layout()
	for i, row := range strings.Split(with.View(), "\n")[lay.HeaderH() : lay.HeaderH()+8] {
		if cellWidth(stripSGR(row)) != lay.TermW {
			t.Errorf("bar frame, transcript row %d is %d cells, want %d", i, cellWidth(stripSGR(row)), lay.TermW)
		}
	}
	padded := 0
	for i, row := range strings.Split(without.View(), "\n")[lay.HeaderH() : lay.HeaderH()+8] {
		// A blank row is the left margin alone, which is not padding.
		if row != strings.Repeat(" ", lay.Pad) && row != strings.TrimRight(row, " ") {
			t.Errorf("no-bar frame, transcript row %d ends in whitespace: %q", i, row)
		}
		if row != "" && cellWidth(row) > lay.Pad {
			padded++
		}
	}
	if padded == 0 {
		t.Error("control: the no-bar frame has no row with content, so the trailing-space check saw nothing")
	}
}

// TestScrollbarStyledStripsToPlain checks the identity the golden files rest on:
// a frame drawn with colour, with its escapes removed, is the frame drawn without
// colour. It runs on frames that carry the bar, pinned and unpinned and with the
// indicator, and each frame must really have the bar and really be styled.
func TestScrollbarStyledStripsToPlain(t *testing.T) {
	h := heightFor(t, 80, 10)
	for name, scroll := range map[string]Scroll{
		"pinned":              {Pinned: true},
		"unpinned at top":     {Offset: 0},
		"unpinned in middle":  {Offset: 15},
		"unpinned, new below": {Offset: 15, NewSince: 3},
	} {
		t.Run(name, func(t *testing.T) {
			styled := barModel(t, barColour, 80, h, 40)
			plain := barModel(t, barUnicode, 80, h, 40)
			if scroll.Pinned {
				styled.relayout(styled.layout())
				plain.relayout(plain.layout())
			} else {
				styled.scroll, plain.scroll = scroll, scroll
			}
			sv, pv := styled.View(), plain.View()
			if !strings.ContainsRune(sv, 0x1b) {
				t.Fatal("control: the styled frame carries no escapes")
			}
			if !barDrawnIn(t, plain) {
				t.Fatal("control: the frame has no bar")
			}
			if got := stripSGR(sv); got != pv {
				t.Errorf("strip(styled) != plain\n  styled stripped: %q\n  plain:           %q", got, pv)
			}
		})
	}
}
