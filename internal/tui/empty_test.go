package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The onboarding screen — wordmark, tagline, three example prompts — is the
// first thing a fresh session shows, and ui-spec §1 puts the smallest supported
// terminal at 40 columns. These tests pin the two decisions that meet there: the
// width the tagline is allowed to spend, and the width the wordmark's band is
// read from.

// emptyRows renders the onboarding band on its own, without the frame around it.
//
// Deliberately below View: View ends every frame with a truncEnd to the content
// width carrying an EMPTY marker, because its job is the frame's hard edge
// rather than an elision. An unbounded row therefore reaches the user cut off
// mid-word and reaches a frame-level assertion already the right length. The
// overrun these tests exist for is only visible before that last cut, which is
// the layer this helper reads.
func emptyRows(ver string, unicode bool, w, h int) (Model, Layout, []string) {
	m := New(Options{Version: ver, Caps: Caps{Unicode: unicode}})
	lay := computeLayout(w, h, 1, false)
	return m, lay, m.emptyStateRows(lay)
}

// taglineOf returns the plain text of the tagline row, and whether there was one.
//
// Found by its leading "v" rather than by index. Nothing else in the band can
// begin with one: the wordmark rows open with a half-block or "K", the lead-in
// with "A", and every suggestion with the bullet glyph. An index would have to
// know how many rows the wordmark occupies, which differs between the Unicode
// and ASCII forms and is not what these tests are about.
func taglineOf(rows []string) (string, bool) {
	for _, r := range rows {
		if p := stripSGR(r); strings.HasPrefix(p, "v") {
			return p, true
		}
	}
	return "", false
}

// TestEmptyStateTaglineFitsTheContentWidth pins the bound on the tagline.
//
// The row is assembled from a build-time version string, so its width is not a
// constant: the fixtures rendered "0.1.0" and measured 37 cells, while the
// shipped binary renders "0.1.0-dev" (cmd/kirsch/main.go) and measures 41
// against the 38 content columns a 40-column terminal leaves. It carried no
// width bound at all, so between 40 and 42 columns it overran and View's final
// cut took it — markerless, mid-word, reading "…terminal-native coding ag" while
// every other truncation in the package ends in `⋯`.
//
// Two version strings are walked and neither is the fixture's business to be
// right about. The second is release-candidate shaped and long enough to force a
// cut well past the minimum width, which is what stops this passing for the
// accidental reason that today's default happens to fit at 43 columns.
//
// The expected full string is taken from a 200-column render rather than
// restated here. A literal would pin the marketing copy, which screen 01 already
// does; what this test is for is the relationship between that copy and the
// space it is given.
func TestEmptyStateTaglineFitsTheContentWidth(t *testing.T) {
	versions := []string{fixtureVersion, "0.1.0-rc.1+build.20260916"}
	widths := []int{40, 41, 42, 43, 44, 46, 50, 60, 80, 120}

	for _, ver := range versions {
		for _, unicode := range []bool{true, false} {
			// The oracle: the same row where nothing can truncate it.
			_, _, wide := emptyRows(ver, unicode, 200, 24)
			full, ok := taglineOf(wide)
			if !ok {
				t.Fatalf("v=%q unicode=%v: no tagline row at 200 columns", ver, unicode)
			}

			for _, w := range widths {
				m, lay, rows := emptyRows(ver, unicode, w, 24)
				got, ok := taglineOf(rows)
				if !ok {
					t.Errorf("v=%q unicode=%v %d cols: no tagline row", ver, unicode, w)
					continue
				}

				// The binding itself. Anything over the content width is a row
				// the frame will have to cut, and the frame cuts without a marker.
				if cw := cellWidth(got); cw > lay.ContentW {
					t.Errorf("v=%q unicode=%v %d cols: tagline is %d cells of %d content columns: %q",
						ver, unicode, w, cw, lay.ContentW, got)
				}

				switch {
				case cellWidth(full) <= lay.ContentW:
					// Room for all of it: it must not have been cut anyway.
					if got != full {
						t.Errorf("v=%q unicode=%v %d cols: tagline fits in %d columns but rendered %q, want %q",
							ver, unicode, w, lay.ContentW, got, full)
					}
				default:
					// No room: it must say so, in the package's own marker and
					// not with a silent cut.
					if !strings.HasSuffix(got, m.gly.Trunc) {
						t.Errorf("v=%q unicode=%v %d cols: tagline was cut without the %q marker: %q",
							ver, unicode, w, m.gly.Trunc, got)
					}
					// The version leads, so it is what survives a cut. A ladder
					// that dropped it would leave the row saying nothing the
					// wordmark above has not already said.
					if !strings.HasPrefix(got, "v"+ver[:1]) {
						t.Errorf("v=%q unicode=%v %d cols: cut took the version: %q", ver, unicode, w, got)
					}
				}
			}
		}
	}
}

// TestEmptyStateTaglineReachesTheFrameIntact is the frame-level half of the
// test above: the bound is worth nothing if View then re-cuts the row.
//
// 40 columns because that is where the defect was, and the marker because its
// absence is the whole visible symptom.
func TestEmptyStateTaglineReachesTheFrameIntact(t *testing.T) {
	for _, unicode := range []bool{true, false} {
		m := New(Options{Version: fixtureVersion, Caps: Caps{Unicode: unicode}})
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 18})
		frame := mm.(Model)

		var row string
		for _, ln := range strings.Split(frame.View(), "\n") {
			if p := strings.TrimLeft(stripSGR(ln), " "); strings.HasPrefix(p, "v"+fixtureVersion) {
				row = p
			}
		}
		if row == "" {
			t.Fatalf("unicode=%v: no tagline in the 40x18 frame", unicode)
		}
		if !strings.HasSuffix(row, frame.gly.Trunc) {
			t.Errorf("unicode=%v: 40-column tagline %q does not end in the %q marker",
				unicode, row, frame.gly.Trunc)
		}
		t.Logf("unicode=%v 40 cols: %q", unicode, row)
	}
}

// TestEmptyStateWordmarkBandReadsTerminalWidth pins the one §2.2-shaped band in
// the package that no golden grid covers.
//
// computeLayout's own switch is pinned by the grids — every boundary in it
// changes a screen, so moving one fails TestMatchesScreenReference. This band is
// not in that switch. It is a single comparison inside a renderer, and the
// goldens sit at 80x18 (screen 01, where 78 >= 40 either way) and 40x11 (screen
// 10, which has a transcript and so never reaches the empty state at all). So
// the comparison could be reverted from TermW to ContentW and nothing would say
// so — which is exactly what it was before, and what the margin work had to fix.
//
// 40 and 41 are the only two widths where the two readings disagree: the frame
// spends two columns on its margin, so ContentW is TermW-2, and only at those
// two is one of them at or above 40 while the other is not. Both sides of the
// boundary are walked, so a band moved in either direction fails.
func TestEmptyStateWordmarkBandReadsTerminalWidth(t *testing.T) {
	for _, unicode := range []bool{true, false} {
		mark := wordmarkUTF8[0]
		if !unicode {
			mark = wordmarkASCII[0]
		}
		for w := 36; w <= 44; w++ {
			_, lay, rows := emptyRows(fixtureVersion, unicode, w, 24)

			seen := false
			for _, r := range rows {
				if stripSGR(r) == mark {
					seen = true
				}
			}
			want := w >= 40
			if seen != want {
				t.Errorf("unicode=%v %d cols (content %d): wordmark shown=%v, want %v",
					unicode, w, lay.ContentW, seen, want)
			}
		}
	}
}

// TestEmptyStateWordmarkSurvivesTheAdvertisedMinimum is the band test's
// frame-level counterpart, and the claim a reader actually cares about: at
// ui-spec §1's smallest supported terminal, the first screen of a fresh session
// still carries the wordmark.
func TestEmptyStateWordmarkSurvivesTheAdvertisedMinimum(t *testing.T) {
	m := New(Options{Version: fixtureVersion, Caps: Caps{Unicode: true}})
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 18})
	out := stripSGR(mm.(Model).View())
	for i, row := range wordmarkUTF8 {
		if !strings.Contains(out, row) {
			t.Errorf("40x18 empty session is missing wordmark row %d (%q)", i+1, row)
		}
	}
}

// TestEmptyStateWordmarkHeightBandIsUnchanged is the height half of the same
// comparison, kept beside the width half so a fix to one cannot quietly drop the
// other. 10 rows is where ui-spec §2.2 starts the full layout.
func TestEmptyStateWordmarkHeightBandIsUnchanged(t *testing.T) {
	for h := 8; h <= 12; h++ {
		_, _, rows := emptyRows(fixtureVersion, true, 80, h)
		seen := false
		for _, r := range rows {
			if stripSGR(r) == wordmarkUTF8[0] {
				seen = true
			}
		}
		if want := h >= 10; seen != want {
			t.Errorf("80x%d: wordmark shown=%v, want %v", h, seen, want)
		}
	}
}
