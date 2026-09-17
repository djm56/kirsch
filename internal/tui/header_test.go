package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

// The header is two rows: the wordmark with a rule running to the right edge,
// then the session line — project, branch, dirty marker, compaction note. These
// tests pin the geometry that follows from that, because the row count feeds
// the chrome budget, the modal origin and every §2.2 breakpoint, and a header
// that quietly renders one row or three corrupts all three at once.

// headerModel builds a sized model with an explicit session, so nothing here
// depends on the fixture, the clock or the environment.
func headerModel(t *testing.T, sess SessionInfo, w, h int) Model {
	t.Helper()
	m := New(Options{
		Version: fixtureVersion,
		Caps:    Caps{Unicode: true},
		Session: sess,
		Status:  Status{Model: "claude-sonnet-5", Family: "sonnet-5"},
	})
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model)
}

func defaultSession() SessionInfo {
	return SessionInfo{Project: "my-project", Branch: "main", Dirty: true}
}

// frameRows splits a rendered frame into rows and takes the frame's left margin
// off each one, so what comes back is content the way every renderer wrote it.
//
// The margin is CHECKED here, not merely stripped. Every assertion below is
// about content — which row the project name is on, where the rule reaches, what
// the session line says — and each would read the same whether padFrame put a
// margin there or not. A helper that stripped an optional prefix would make the
// whole file blind to the margin disappearing; one that requires it makes every
// test in the file a witness to it at no cost to what each is actually about.
func frameRows(t *testing.T, m Model) []string {
	t.Helper()
	lay := m.layout()
	margin := strings.Repeat(" ", lay.Pad)
	rows := strings.Split(m.View(), "\n")
	for i, r := range rows {
		if !strings.HasPrefix(r, margin) {
			t.Errorf("frame row %d is %q, which does not open with the %d-column margin",
				i+1, r, lay.Pad)
			continue
		}
		rows[i] = strings.TrimPrefix(r, margin)
	}
	return rows
}

// TestHeaderOccupiesTwoRows is the load-bearing one: when the header is shown
// it consumes exactly headerHeight rows of the frame, at every size where it is
// shown at all. Everything downstream — chrome, modal origin, transcript band —
// is derived from that number rather than re-deriving it, so this is the only
// place it is measured against a real render.
func TestHeaderOccupiesTwoRows(t *testing.T) {
	for _, w := range []int{200, 80, 79, 60, 59, 40, 39, 20} {
		for _, h := range []int{60, 24, 12, 11, 10} {
			m := headerModel(t, defaultSession(), w, h)
			lay := m.layout()
			if !lay.ShowHeader {
				t.Fatalf("%dx%d: header hidden, so this case proves nothing — "+
					"the size table has drifted from the height band", w, h)
			}
			if lay.HeaderH() != 2 {
				t.Errorf("%dx%d: HeaderH()=%d, want 2", w, h, lay.HeaderH())
			}
			rows := frameRows(t, m)
			if len(rows) < 2 {
				t.Fatalf("%dx%d: %d rows rendered, want at least 2", w, h, len(rows))
			}
			// Row 1 is the wordmark, row 2 the session line. Proving both are
			// header rows is what makes "two rows" a measurement rather than a
			// restatement of the constant.
			if !strings.HasPrefix(rows[0], "Kirsch") {
				t.Errorf("%dx%d row 1: %q does not start with the wordmark", w, h, rows[0])
			}
			if !strings.HasPrefix(rows[1], "my-project") {
				t.Errorf("%dx%d row 2: %q does not start with the project name", w, h, rows[1])
			}
		}
	}
}

// TestHeaderRowsMatchHeaderH checks the header's declared height against the
// rows it actually produces, rather than against the constant it is derived
// from. A headerRows that returned one row or three would keep HeaderH() at 2
// and silently shift every row below it.
func TestHeaderRowsMatchHeaderH(t *testing.T) {
	for _, w := range []int{80, 40} {
		m := headerModel(t, defaultSession(), w, 24)
		lay := m.layout()
		if got := len(m.headerRows(lay)); got != lay.HeaderH() {
			t.Errorf("width %d: headerRows produced %d rows, HeaderH()=%d", w, got, lay.HeaderH())
		}
	}
}

// TestChromeBudgetAtHeightBands walks the §2.2 height ladder and pins the
// transcript height each band leaves. The header band still starts at h=10 —
// ui-spec §1 names 40×10 as the smallest supported terminal, and the smallest
// supported terminal is where the full layout has to still be the full layout —
// so the two-row header is paid for out of the transcript, which drops from
// five rows to four in that band.
func TestChromeBudgetAtHeightBands(t *testing.T) {
	cases := []struct {
		h          int
		composer   int
		header     bool
		rules      bool
		transcript int
		why        string
	}{
		{
			h: 24, composer: 1, header: true, rules: true, transcript: 18,
			why: "chrome = header 2 + rules 2 + status 1 + composer 1",
		},
		{h: 12, composer: 1, header: true, rules: true, transcript: 6, why: "12 - 6"},
		{
			h: 11, composer: 1, header: true, rules: true, transcript: 5,
			why: "11 - 6; the old five-row floor now needs one more row",
		},
		{
			h: 10, composer: 1, header: true, rules: true, transcript: 4,
			why: "the band boundary stays at 10; the floor drops from 5 to 4",
		},
		{
			h: 9, composer: 1, header: false, rules: true, transcript: 5,
			why: "header gone frees 2 rows, so the transcript grows across the boundary",
		},
		{
			h: 6, composer: 1, header: false, rules: true, transcript: 2,
			why: "unchanged: no header in this band",
		},
		{
			h: 5, composer: 1, header: false, rules: true, transcript: 1,
			why: "unchanged: no header in this band",
		},
		{
			h: 4, composer: 1, header: false, rules: false, transcript: 2,
			why: "unchanged: below 5 the rules go as a pair",
		},
		{
			h: 10, composer: 5, header: true, rules: true, transcript: 0,
			why: "a full composer spends the whole budget; nothing degrades because nothing is negative",
		},
	}
	for _, c := range cases {
		lay := computeLayout(80, c.h, c.composer, false)
		if lay.ShowHeader != c.header || lay.ShowRules != c.rules {
			t.Errorf("80x%d composer=%d: ShowHeader=%v ShowRules=%v, want %v/%v (%s)",
				c.h, c.composer, lay.ShowHeader, lay.ShowRules, c.header, c.rules, c.why)
		}
		if lay.TranscriptH != c.transcript {
			t.Errorf("80x%d composer=%d: TranscriptH=%d, want %d (%s)",
				c.h, c.composer, lay.TranscriptH, c.transcript, c.why)
		}
		// The budget has to agree with the frame it produces, or the arithmetic
		// above is only checking itself.
		wantChrome := 1 + lay.HeaderH() + lay.ComposerH
		if lay.ShowRules {
			wantChrome += 2
		}
		if got := c.h - lay.TranscriptH; got != wantChrome {
			t.Errorf("80x%d composer=%d: frame leaves %d chrome rows, budget says %d",
				c.h, c.composer, got, wantChrome)
		}
	}
}

// TestHeaderBandStartsAtTen states the boundary directly, so moving it fails a
// test that names the reason rather than only shifting a number in a table.
func TestHeaderBandStartsAtTen(t *testing.T) {
	if l := computeLayout(40, 10, 1, false); !l.ShowHeader {
		t.Error("40x10 is the smallest supported terminal per ui-spec §1 and must " +
			"still show the header; raising the band to 11 would leave the " +
			"advertised minimum without one")
	}
	if l := computeLayout(40, 9, 1, false); l.ShowHeader {
		t.Error("40x9: header shown below the band")
	}
}

// TestSessionLineNeverRendersEmpty is why the header's height is a constant.
// Row 2 has to carry something at every width and for every session a model can
// actually hold, or the header is paying a row for nothing.
//
// The empty session is the load-bearing case, and it passes today for a reason
// that is about to expire. New substitutes the whole Milestone 0 demo session
// when Options.Session has no project, so no model in this milestone reaches
// headerRows anonymous. headerRows itself has no such guard —
// TestSessionLineOmitsAbsentFields assigns m.sess directly, bypassing New, and
// pins a blank row 2 for the empty case. The two are reconciled only by that
// substitution.
//
// So this fails the moment M2 wires a real session and the fixture goes, which
// is the point: it is the one place that says out loud that the constant has
// stopped being free. Adding a fallback to headerRows instead would invent a
// header field ui-spec §2 does not have and contradict the test above it, so
// the choice — guarantee the session a name, give row 2 a fallback, or make the
// height depend on content — is left where the information to make it will be.
func TestSessionLineNeverRendersEmpty(t *testing.T) {
	cases := []struct {
		name string
		sess SessionInfo
	}{
		{"explicit session", defaultSession()},
		{"empty session (New substitutes the Milestone 0 fixture)", SessionInfo{}},
	}
	for _, c := range cases {
		for _, w := range []int{200, 80, 79, 60, 59, 40, 39, 20, 12} {
			m := headerModel(t, c.sess, w, 24)
			rows := strings.Split(m.View(), "\n")
			if strings.TrimSpace(rows[1]) == "" {
				t.Errorf("%s, width %d: session line is blank, so the header is paying "+
					"a row for nothing", c.name, w)
			}
		}
	}
}

// TestWordmarkRowCarriesOnlyTheWordmark is the operator's actual request: the
// first row is `Kirsch` and a rule, and nothing else migrates back onto it.
func TestWordmarkRowCarriesOnlyTheWordmark(t *testing.T) {
	sess := defaultSession()
	sess.Compacted = true
	m := headerModel(t, sess, 80, 24)
	row1 := frameRows(t, m)[0]
	for _, forbidden := range []string{"my-project", "main", "●", "(compacted)"} {
		if strings.Contains(row1, forbidden) {
			t.Errorf("row 1 %q still carries %q, which belongs on the session line",
				row1, forbidden)
		}
	}
	rule := strings.TrimPrefix(row1, "Kirsch ")
	if strings.Trim(rule, "─") != "" {
		t.Errorf("row 1 after the wordmark is %q, want nothing but rule", rule)
	}
}

// TestOnlyTheWordmarkRowIsRuled keeps the header reading as a header. A second
// full-width rule immediately under the first reads as the top edge of a box.
func TestOnlyTheWordmarkRowIsRuled(t *testing.T) {
	m := headerModel(t, defaultSession(), 80, 24)
	rows := frameRows(t, m)
	// The content edge, not the terminal edge: at 80 columns the frame keeps a
	// margin at each side, so the rule runs to 78 and the two it gives up are
	// the margin rather than a rule that stops short.
	edge := m.layout().ContentW
	if cellWidth(rows[0]) != edge {
		t.Errorf("row 1 is %d cols, want the rule to reach the content edge at %d",
			cellWidth(rows[0]), edge)
	}
	if strings.Contains(rows[1], "──") {
		t.Errorf("row 2 %q carries a rule; only the wordmark row is ruled", rows[1])
	}
	if w := cellWidth(strings.TrimRight(rows[1], " ")); w >= edge {
		t.Errorf("row 2 fills %d cols, so it is being ruled to the edge", w)
	}
}

// TestBranchAndDirtyMarkerLiveOnTheSessionLine covers the §2.2 width band where
// the branch drops. Below 60 columns both the branch and the marker that
// qualifies it go; the row stays, carrying the project name alone.
func TestBranchAndDirtyMarkerLiveOnTheSessionLine(t *testing.T) {
	// Verified rather than assumed: computeLayout is the authority on where the
	// branch drops, and these cases are only meaningful either side of it.
	if computeLayout(60, 24, 1, false).ShowBranch == computeLayout(59, 24, 1, false).ShowBranch {
		t.Fatal("the branch does not change state between 59 and 60 columns; " +
			"the width band has moved and these cases are testing nothing")
	}
	wide := frameRows(t, headerModel(t, defaultSession(), 60, 24))
	if !strings.Contains(wide[1], "main") || !strings.Contains(wide[1], "●") {
		t.Errorf("at 60 cols the session line is %q, want branch and dirty marker", wide[1])
	}
	if strings.Contains(wide[0], "main") || strings.Contains(wide[0], "●") {
		t.Errorf("at 60 cols row 1 is %q, want neither on the wordmark row", wide[0])
	}

	narrow := frameRows(t, headerModel(t, defaultSession(), 59, 24))
	if strings.Contains(narrow[1], "main") {
		t.Errorf("at 59 cols the session line is %q, want the branch dropped", narrow[1])
	}
	if strings.Contains(narrow[1], "●") {
		t.Errorf("at 59 cols the session line is %q: a dirty marker with no branch "+
			"to qualify says only that something, somewhere, is uncommitted", narrow[1])
	}
	if !strings.HasPrefix(narrow[1], "my-project") {
		t.Errorf("at 59 cols the session line is %q, want the project name", narrow[1])
	}
}

// TestCompactedSitsOnTheSessionLine places the one remaining header field. It
// is session state, the same class as the branch and the dirty marker, so it
// keeps their company rather than crowding the wordmark.
func TestCompactedSitsOnTheSessionLine(t *testing.T) {
	sess := defaultSession()
	sess.Compacted = true
	rows := frameRows(t, headerModel(t, sess, 80, 24))
	if !strings.Contains(rows[1], "(compacted)") {
		t.Errorf("session line %q is missing the compaction note", rows[1])
	}
	if got, want := rows[1], "my-project ─ main ● (compacted)"; got != want {
		t.Errorf("session line: got %q, want %q", got, want)
	}
}

// TestSessionLineOmitsAbsentFields guards the join. A workspace with no name or
// a repo with no branch must not leave a dangling separator behind.
func TestSessionLineOmitsAbsentFields(t *testing.T) {
	cases := []struct {
		name string
		sess SessionInfo
		want string
	}{
		{"no branch", SessionInfo{Project: "my-project"}, "my-project"},
		{"no branch, dirty", SessionInfo{Project: "my-project", Dirty: true}, "my-project ●"},
		{"no project", SessionInfo{Branch: "main"}, "main"},
		{"neither", SessionInfo{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := headerModel(t, defaultSession(), 80, 24)
			m.sess = c.sess // New substitutes a default for an empty project
			row2 := m.headerRows(m.layout())[1]
			if row2 != c.want {
				t.Errorf("session line: got %q, want %q", row2, c.want)
			}
		})
	}
}

// modalBodies are the three body sizes the geometry has to hold for: the real
// help overlay, a single line, and none at all. The empty case is what the
// body-row floor in modalBox exists for.
func modalBodies() map[string][]string {
	return map[string][]string{
		"help":  helpLines(),
		"one":   {"one line"},
		"empty": nil,
	}
}

// TestModalStaysInsideTranscriptRegion is Layout invariant 3 stated as
// arithmetic: the box starts at the top of the transcript region and ends no
// lower than its last row, at every size the frame renders.
//
// It is walked rather than sampled because the region's height is the one thing
// the two-row header changed. At the smallest supported terminal — 40×10, ui-
// spec §1 — the region is four rows, and the box's own chrome is four, so this
// is the size where the two numbers collide; h=9 and h=11 both leave five rows
// and hide the collision. Nothing in the screen reference reaches it either:
// neither golden modal is capped, so the cap this pins is never exercised
// there.
func TestModalStaysInsideTranscriptRegion(t *testing.T) {
	for name, lines := range modalBodies() {
		for _, w := range []int{40, 60, 80, 120} {
			for h := 1; h <= 40; h++ {
				for _, composer := range []int{1, 3, 5} {
					m := headerModel(t, defaultSession(), w, h)
					m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: lines})
					lay := computeLayout(w, h, composer, true)
					box, top, left := m.modalBox(lay)
					if len(box) == 0 {
						t.Errorf("%s %dx%d composer=%d: the modal drew nothing",
							name, w, h, composer)
						continue
					}
					if lay.TranscriptH < 1 {
						// The floor: no region to be inside of, so the notice
						// takes the frame's last row. Stated, not skipped.
						if want := h - 1; top != want {
							t.Errorf("%s %dx%d composer=%d: regionless frame put the notice "+
								"on row %d, want the last row %d", name, w, h, composer, top, want)
						}
						// And it spans the frame rather than sitting in a
						// centred box. The row it lands on is the status bar's,
						// which it is replacing outright: indented by even one
						// cell, the bar's own text shows to the left of the
						// notice and the row reads as one line saying two
						// things.
						if left != 0 {
							t.Errorf("%s %dx%d composer=%d: regionless notice starts at column "+
								"%d, want 0 — the row beneath shows around it",
								name, w, h, composer, left)
						}
						// Column zero is half of "spans the frame" and this is
						// the other half. The check above was the whole of it
						// for one review round, and a notice starting at zero
						// and ending at cell 22 of 80 leaves fifty-eight cells
						// of status bar finishing the row — the same defect the
						// left-edge check is worded against, on the other side.
						// spliceAt replaces only the cells the string occupies,
						// so the row has to BE the width, not merely start at
						// its left edge.
						if got := cellWidth(box[0]); got != lay.ContentW {
							t.Errorf("%s %dx%d composer=%d: regionless notice is %d cells of a "+
								"%d-cell frame row; the row beneath keeps the other %d:\n%q",
								name, w, h, composer, got, lay.ContentW, lay.ContentW-got, stripSGR(box[0]))
						}
						continue
					}
					if top != lay.HeaderH() {
						t.Errorf("%s %dx%d composer=%d: box top %d, want %d",
							name, w, h, composer, top, lay.HeaderH())
					}
					end, regionEnd := top+len(box), lay.HeaderH()+lay.TranscriptH
					if end > regionEnd {
						t.Errorf("%s %dx%d composer=%d: box occupies rows [%d,%d) but the "+
							"transcript region ends at %d — %d row(s) land on the status bar "+
							"or below (transcriptH=%d, box=%d rows)",
							name, w, h, composer, top, end, regionEnd,
							end-regionEnd, lay.TranscriptH, len(box))
					}
					// The box has two legal forms and the region height picks
					// between them, so a single row floor cannot state this.
					// Anything bordered must be tall enough to hold a footer
					// between its borders; the bare notice row is legal only
					// where no bordered box fits.
					//
					// This replaced a blanket `len(box) < 3`, and the exchange
					// was not a strengthening — it cannot have been. A notice is
					// one row at any region height, so the old floor fired on
					// every notice: applied against a mutant that returns one
					// where a bordered box fits, it fires 168 times. The
					// replacement is strictly WEAKER across the one-and-two-row
					// band, necessarily, because that band is where a one-row
					// box became legal, and equal everywhere else. It is the
					// right assertion for a different reason — the old one
					// forbids the rung that stops an open modal being invisible,
					// so it cannot be restored — and that reason is what a later
					// reader needs when deciding whether this can be relaxed
					// again. It can be relaxed no further: each of the three
					// branches below is pinned by its own mutant.
					//
					// The three are separate checks rather than switch cases. As
					// a switch the row-count branch shadowed the region-height
					// branch, so a two-row notice drawn into a large region was
					// reported as a count error alone and the more serious fault
					// — drawing a notice where a real box fits — never printed.
					bordered := strings.Contains(box[0], m.gly.BoxTL)
					if bordered && len(box) < 3 {
						t.Errorf("%s %dx%d composer=%d: bordered box is %d rows; a box with "+
							"no room for a footer between its borders is an empty shell",
							name, w, h, composer, len(box))
					}
					if !bordered && len(box) != 1 {
						t.Errorf("%s %dx%d composer=%d: unbordered notice is %d rows, want 1",
							name, w, h, composer, len(box))
					}
					if !bordered && lay.TranscriptH >= 3 {
						t.Errorf("%s %dx%d composer=%d: notice drawn with %d rows of region; "+
							"three rows hold a bordered box and the box is what should be drawn",
							name, w, h, composer, lay.TranscriptH)
					}
					// In-region, the coverage obligation is against the BOX's
					// span rather than the frame's: the notice stands in for a
					// bordered box, so it owns exactly the cells that box would
					// have owned and the transcript cannot show up beside it
					// there either.
					//
					// The span is read off the bordered form at this same width
					// rather than recomputed here. modalBox's width is a
					// function of lay.ContentW and ModalPct alone — not of the region
					// — so a taller layout at this width draws the very box the
					// notice is standing in for. Recomputing it would be a
					// second copy of that arithmetic, and a mutant that moved
					// the real one would pass against the copy.
					if !bordered {
						ref, _, refLeft := m.modalBox(computeLayout(w, 40, 1, true))
						if got, want := cellWidth(box[0]), cellWidth(ref[0]); got != want {
							t.Errorf("%s %dx%d composer=%d: in-region notice is %d cells where the "+
								"bordered box at this width is %d; the transcript keeps the other "+
								"%d:\n%q", name, w, h, composer, got, want, want-got, stripSGR(box[0]))
						}
						if left != refLeft {
							t.Errorf("%s %dx%d composer=%d: in-region notice starts at column %d, "+
								"the bordered box at this width starts at %d",
								name, w, h, composer, left, refLeft)
						}
					}
				}
			}
		}
	}
}

// TestModalLeavesTheRowsOutsideTheRegionUntouched checks the same invariant
// against a real frame rather than against the box's declared height, so a
// modal that overran the region would be caught even if its arithmetic agreed
// with itself.
//
// The composer rows are excluded: opening a modal moves input capture off the
// composer, so its row legitimately differs between the two renders. Everything
// else below the region — the rules and the status bar — must be byte-identical
// with the modal open and closed, and so must the header above it.
func TestModalLeavesTheRowsOutsideTheRegionUntouched(t *testing.T) {
	for _, size := range [][2]int{{80, 10}, {40, 10}, {80, 11}, {80, 24}, {80, 32}} {
		w, h := size[0], size[1]
		plain := headerModel(t, defaultSession(), w, h)
		withModal := plain
		withModal.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})

		lay := plain.layout()
		before := strings.Split(plain.View(), "\n")
		after := strings.Split(withModal.View(), "\n")
		if len(before) != len(after) {
			t.Fatalf("%dx%d: %d rows without the modal, %d with it", w, h, len(before), len(after))
		}
		regionEnd := lay.HeaderH() + lay.TranscriptH
		for i := range before {
			switch {
			case i >= lay.HeaderH() && i < regionEnd:
				continue // the region is the modal's to overwrite
			case i >= len(before)-lay.ComposerH:
				continue // input capture moved; the composer row is expected to change
			case before[i] != after[i]:
				t.Errorf("%dx%d row %d is outside the transcript region [%d,%d) but the "+
					"modal changed it:\n without: %q\n    with: %q",
					w, h, i, lay.HeaderH(), regionEnd, before[i], after[i])
			}
		}
	}
}

// TestModalFitsItsContentWhenTheRegionAllows pins the other half of the height
// rule. The box is as tall as what it holds, not as tall as the space it is
// offered: a region with room to spare must not leave blank rows inside the
// borders.
func TestModalFitsItsContentWhenTheRegionAllows(t *testing.T) {
	const chrome = 4 // top border, footer rule, footer, bottom border
	for name, lines := range modalBodies() {
		body := maxInt(len(lines), 1) // an empty modal still shows one row
		for _, size := range [][2]int{{80, 40}, {80, 32}, {80, 24}, {120, 40}} {
			w, h := size[0], size[1]
			m := headerModel(t, defaultSession(), w, h)
			m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: lines})
			lay := m.layout()
			if lay.TranscriptH < body+chrome {
				continue // capped by the region; the cap is the other test's subject
			}
			box, _, _ := m.modalBox(lay)
			if got, want := len(box), body+chrome; got != want {
				t.Errorf("%s %dx%d: box is %d rows for %d body line(s) in a %d-row region, "+
					"want %d — the box is padding itself out to the region",
					name, w, h, got, body, lay.TranscriptH, want)
			}
		}
	}
}

// TestModalDropsItsRuleBeforeItsFooter pins the order the box degrades in when
// the region cannot hold all of it. The rule separates the footer from the body
// and says nothing the footer does not; the footer carries the only way out of
// a modal that traps every key. So the rule goes first, and it goes at exactly
// one region height — four rows, where the full box is five. Width has no say in
// it; 80 columns is the case below only because a case needs some width, and the
// two cases differ in terminal height because that is what moves the region.
func TestModalDropsItsRuleBeforeItsFooter(t *testing.T) {
	cases := []struct {
		h        int
		region   int
		rows     int
		rule     bool
		why      string
		expected string
	}{
		{h: 11, region: 5, rows: 5, rule: true, why: "five rows of region hold the whole box"},
		{h: 10, region: 4, rows: 4, rule: false, why: "one row short: the rule goes, the footer stays"},
	}
	for _, c := range cases {
		m := headerModel(t, defaultSession(), 80, c.h)
		m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
		lay := m.layout()
		if lay.TranscriptH != c.region {
			t.Fatalf("80x%d: transcript region is %d rows, not %d — the height ladder has "+
				"moved and this case is testing nothing", c.h, lay.TranscriptH, c.region)
		}
		box, _, _ := m.modalBox(lay)
		if len(box) != c.rows {
			t.Errorf("80x%d: box is %d rows, want %d (%s)", c.h, len(box), c.rows, c.why)
		}
		var ruled bool
		for _, line := range box {
			if strings.Contains(line, m.gly.BoxLT) {
				ruled = true
			}
		}
		if ruled != c.rule {
			t.Errorf("80x%d: footer rule present=%v, want %v (%s)", c.h, ruled, c.rule, c.why)
		}
		// Whatever else goes, the way out stays on screen. The guard is not
		// reachable at these two fixed heights, but "the box degraded further
		// than expected" is a result this test should report rather than panic
		// on — index arithmetic over a box whose height is the subject cannot
		// assume the height.
		if len(box) < 2 {
			t.Errorf("80x%d: box is %d row(s); there is no footer above a bottom border",
				c.h, len(box))
			continue
		}
		if footer := box[len(box)-2]; !strings.Contains(footer, "Esc") {
			t.Errorf("80x%d: row above the bottom border is %q, want the footer carrying Esc",
				c.h, footer)
		}
	}
}

// TestModalOpensBelowTheWholeHeader pins the one geometry outside view.go that
// has to know how tall the header is. A modal anchored to a one-row header
// paints over the session line.
func TestModalOpensBelowTheWholeHeader(t *testing.T) {
	m := headerModel(t, defaultSession(), 80, 24)
	m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
	lay := m.layout()
	_, top, _ := m.modalBox(lay)
	if top != lay.HeaderH() {
		t.Errorf("modal top row %d, want %d (below the whole header)", top, lay.HeaderH())
	}
	rows := frameRows(t, m)
	if !strings.HasPrefix(rows[1], "my-project") {
		t.Errorf("row 2 is %q: the modal has painted over the session line", rows[1])
	}
}

// overlayFill is the cell every sentinel frame in this file is built from: a
// cell no overlay has written.
//
// It used to be "·", which is Glyphs.Bullet — the character the modal notice
// puts between its title and its exit key. A sentinel the overlay itself can
// emit cannot answer the question these frames exist to ask, "which cells did
// the row beneath keep", and the one assertion that tried worked around the
// collision by reading a PREFIX. That workaround is a large part of why four
// unpadded splice sites survived thirty-nine mutants: the cells that survived
// were on the tail.
//
// One cell wide, and in no glyph table, footer, prompt or help row.
// TestSentinelIsAbsentFromEveryOverlay holds the second half of that, because a
// sentinel the overlay can emit weakens every assertion here silently rather
// than failing.
const overlayFill = "▚"

// overlaidRows composites an overlay onto a sentinel frame and reports which
// rows it actually changed: the first, how many, and their plain text.
//
// It measures placement the way the terminal sees it rather than the way the
// overlay declares it. modalBox returns its own top, so a test that reads that
// number is asking the code under test where it put itself; compositing asks
// the frame. The confirm prompt has no declared top at all, which is how it
// spent the whole of milestone 1 anchored to the status bar without a test
// noticing.
//
// The same frame answers coverage as well as placement: every cell of a
// returned row still holding overlayFill is a cell of the row beneath that the
// overlay left showing. See survivingCells.
//
// The sentinel rows are lay.ContentW wide and not the terminal's width, because
// that is what the rows an overlay composites over are: View builds every layer
// at the content width and padFrame adds the margin afterwards, once the last
// overlay is already down. A frame built at the terminal width would be asking
// the overlays to fill two columns that are not theirs, and the answer it got
// would come from spliceAt's truncation rather than from the overlay.
func overlaidRows(fn func([]string, Layout) []string, lay Layout, h int) (top, n int, body []string) {
	base := make([]string, h)
	for i := range base {
		base[i] = strings.Repeat(overlayFill, maxInt(lay.ContentW, 1))
	}
	got := fn(append([]string(nil), base...), lay)
	top = -1
	for i := range got {
		if i < len(base) && got[i] == base[i] {
			continue
		}
		if top < 0 {
			top = i
		}
		n = i - top + 1
		body = append(body, stripSGR(got[i]))
	}
	if top < 0 {
		top = 0
	}
	return top, n, body
}

// survivingCells reports the display-cell indices of row still holding
// overlayFill: the cells of the row underneath that the overlay did not write.
//
// It is indices and not a count because where they are is the whole question. A
// count says the row is wrong; the indices say whether the overlay was placed
// too far left, cut too short on the right, or both, and the failure messages
// below print them.
//
// It walks cells rather than runes. A wide rune occupies two cells and is
// neither of them, so a rune-indexed answer would drift by one for every CJK
// character in the row beneath.
func survivingCells(row string) []int {
	var out []int
	col := 0
	for _, r := range stripSGR(row) {
		rw := runewidth.RuneWidth(r)
		if string(r) == overlayFill {
			out = append(out, col)
		}
		col += rw
	}
	return out
}

// sameCells reports whether two cell-index lists are equal.
func sameCells(a, b []int) bool {
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

// TestModalIsNeverInvisibleWhileTheRegionHasARow is the regression test for the
// invisible modal: a region of one or two rows used to draw nothing at all,
// while mode() still reported Modal and every key was still trapped. The frame
// was indistinguishable from an idle session, so the only way out of it was to
// guess that Esc did something.
//
// It is reachable at the advertised minimum terminal — 40×10, ui-spec §1 — with
// a three-line draft in the composer, which is why this is walked rather than
// sampled: the band moves with the composer's height, so no fixed size pins it.
//
// The rule it states is the ladder's floor: the modal draws something, always,
// and whatever it draws names the key that closes it.
//
// It used to carry a `lay.TranscriptH == 0` arm asserting that NOTHING was
// drawn there, which is the defect written down as a requirement — a test that
// checks for the absence of output reads as coverage while pinning the hole it
// is named after. The region is reserved in computeLayout now, so a region of
// no rows is only reachable on a frame of two rows or fewer, and there the
// notice takes the frame's last row. Either way a row exists, so the arm is
// gone rather than inverted: there is no size left that it describes.
func TestModalIsNeverInvisibleWhileTheRegionHasARow(t *testing.T) {
	for name, lines := range modalBodies() {
		for _, w := range []int{38, 40, 60, 80, 120} {
			for h := 1; h <= 40; h++ {
				for composer := 1; composer <= 5; composer++ {
					m := headerModel(t, defaultSession(), w, h)
					m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: lines})
					lay := computeLayout(w, h, composer, true)
					box, _, _ := m.modalBox(lay)

					if len(box) == 0 {
						t.Errorf("%s %dx%d composer=%d: the region has %d row(s) and the "+
							"modal drew nothing; the frame reads as an idle session with "+
							"every key trapped", name, w, h, composer, lay.TranscriptH)
						continue
					}
					var wayOut bool
					for _, line := range box {
						if strings.Contains(stripSGR(line), "Esc") {
							wayOut = true
						}
					}
					if !wayOut {
						t.Errorf("%s %dx%d composer=%d: %d-row modal names no way out:\n%s",
							name, w, h, composer, len(box), strings.Join(plain(box), "\n"))
					}
				}
			}
		}
	}
}

// plain strips styling from a block of rows, for failure messages.
func plain(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = stripSGR(r)
	}
	return out
}

// TestModalFooterNamesTheWayOutAtEveryWidth pins the footer ladder.
//
// A single truncEnd was not enough: at 40 columns — the advertised minimum —
// the help footer's full form cut to "kirsch v0.1.0 · docs: doc/usage.md…" and
// the three words naming the exit key were the ones that went. The footer was
// present and structurally intact, which is why the row-count assertions above
// all passed while the modal was, in the only sense that matters, unleaveable.
//
// Both footer families are walked, because the shortfall was the help footer's
// length rather than the mechanism, and a fix that only shortened that one
// string would leave the next long footer to rediscover it.
func TestModalFooterNamesTheWayOutAtEveryWidth(t *testing.T) {
	kinds := map[ModalKind]string{
		ModalHelp:    "Esc or ?",
		ModalContent: "Esc",
		ModalDiff:    "Esc",
	}
	for kind, want := range kinds {
		for _, w := range []int{38, 40, 45, 50, 60, 80, 120, 200} {
			m := headerModel(t, defaultSession(), w, 24)
			m.openModal(ModalState{Kind: kind, Title: "a modal title", Lines: helpLines()})
			lay := m.layout()
			box, _, _ := m.modalBox(lay)
			// Same guard as TestModalDropsItsRuleBeforeItsFooter: unreachable at
			// h=24, and a report rather than a panic if the geometry moves.
			if len(box) < 2 {
				t.Errorf("kind=%d width=%d: box is %d row(s); no footer to read",
					kind, w, len(box))
				continue
			}
			footer := stripSGR(box[len(box)-2])
			if !strings.Contains(footer, want) {
				t.Errorf("kind=%d width=%d: footer is %q, want it to contain %q",
					kind, w, footer, want)
			}
		}
	}
}

// TestModalNamesNoInertBindingAtTheBodylessRung pins the other half of what the
// footer is for. The rung above the notice keeps the footer and drops the body,
// so the box is two borders and a footer over nothing — and the footer was still
// offering "j/k scroll · g/G top/bottom", three keys with no rows to move.
//
// A footer that names a binding doing nothing is worse than a short one. It is
// the row a user reads when the frame has already stopped looking right, so it
// is the last place that should be describing a modal other than the one on
// screen. The rung starts one variant down instead.
//
// The band is walked rather than sampled because it moves with the composer, the
// same way the notice band does.
func TestModalNamesNoInertBindingAtTheBodylessRung(t *testing.T) {
	var seen int
	for name, lines := range modalBodies() {
		for _, w := range []int{38, 40, 60, 80, 120} {
			for h := 1; h <= 40; h++ {
				for composer := 1; composer <= 5; composer++ {
					m := headerModel(t, defaultSession(), w, h)
					m.openModal(ModalState{Kind: ModalContent, Title: "a file", Lines: lines})
					lay := computeLayout(w, h, composer, true)
					box, _, _ := m.modalBox(lay)
					// The bodyless rung is the three-row bordered box: two
					// borders and a footer, with nothing between the footer and
					// the top border.
					if len(box) != 3 || !strings.Contains(box[0], m.gly.BoxTL) {
						continue
					}
					seen++
					footer := stripSGR(box[1])
					for _, inert := range []string{"j/k", "g/G", "scroll", "top/bottom"} {
						if strings.Contains(footer, inert) {
							t.Errorf("%s %dx%d composer=%d: bodyless box offers %q in its "+
								"footer %q, over zero body rows",
								name, w, h, composer, inert, footer)
						}
					}
					if !strings.Contains(footer, "Esc") {
						t.Errorf("%s %dx%d composer=%d: bodyless box footer %q names no way out",
							name, w, h, composer, footer)
					}
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("the bodyless rung was never reached; this test asserted nothing")
	}
}

// TestAsciiFallbackNeverEmitsAnEllipsis is the §10.2 fallback stated as a
// property of the frame rather than of the glyph table.
//
// Every truncation marker in this package has to come from Glyphs.Trunc, which
// is "⋯" on a UTF-8 terminal and "..." elsewhere. Four of them were spelled as a
// literal "…" instead, and a literal survives NewGlyphs(false) untouched: the
// terminal that cannot render "⋯" gets a raw U+2026 in the same frame whose
// other markers folded to ASCII. It is also the wrong character in Unicode mode,
// where the rest of the frame uses "⋯".
//
// The assertion is over the whole rendered frame at widths that force every
// truncation path to fire, so a fifth literal added later fails here without
// anyone having to find it.
//
// It checks the two ellipsis characters and no other typography, which is
// narrower than the name §10.2 would suggest and deliberately so. Glyphs.Fold
// covers the rest — em dashes and smart quotes arriving inside content — and it
// is applied at seven sites, all of them in cards.go, and nowhere else.
//
// The gap that leaves is the overlay BODY, not the title. ModalState.Lines
// carries it.Tool.Out and it.Err.Detail — arbitrary command output and error
// text — straight to truncEnd unfolded, and command output is far likelier to
// hold an em dash or a smart quote than a title is. Titles are the smaller half
// of the same hole and the one that reads first; bodies are where it will
// actually be hit.
//
// It is NOT fixed here. Closing it means deciding where Fold belongs on the
// overlay paths — at the modal's line renderer, or earlier at the point content
// enters ModalState.Lines, which is also where SanitizeLines already runs — and
// that is a content-pipeline change rather than a marker swap. Widening this
// test to "no non-ASCII typography", with a body carrying an em dash in the
// fixture, is the right move once it is.
func TestAsciiFallbackNeverEmitsAnEllipsis(t *testing.T) {
	// Long enough that the title, the card target, the suggestion lines and the
	// modal body all have to cut.
	const long = "a considerably longer name than any terminal this narrow can show"
	frames := map[string]func(m *Model){
		"empty": func(m *Model) {},
		"modal": func(m *Model) { m.openModal(ModalState{Kind: ModalContent, Title: long, Lines: []string{long}}) },
		"help":  func(m *Model) { m.openModal(ModalState{Kind: ModalHelp, Title: long, Lines: helpLines()}) },
		"confirm": func(m *Model) {
			m.askConfirm(ConfirmState{Prompt: long + "?", Action: ConfirmNewSession})
		},
	}
	for name, build := range frames {
		for _, w := range []int{40, 60, 80} {
			m := New(Options{
				Version: fixtureVersion,
				Caps:    Caps{Unicode: false},
				Session: SessionInfo{Project: long, Branch: long, Dirty: true},
				Status:  Status{Model: "claude-sonnet-5", Family: "sonnet-5"},
			})
			mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 24})
			m = mm.(Model)
			build(&m)
			for _, bad := range []string{"…", "⋯"} {
				if strings.Contains(m.View(), bad) {
					t.Errorf("%s %dx24 ASCII caps: frame contains %q; every truncation "+
						"marker must come from Glyphs.Trunc", name, w, bad)
				}
			}
		}
	}
}

// TestConfirmStaysInsideTranscriptRegion is Layout invariant 3 applied to the
// confirm prompt.
//
// ui-spec §4.3 puts the confirm prompt inside §4, so it is a modal and the
// invariant binds it: the transcript region is what an overlay may overwrite,
// and the status bar is what it may not. The prompt was anchored to
// `lay.H - lay.ComposerH - 3` — the frame's rule/status/rule band — which put
// it over the status bar at every size, and no test caught that because every
// other modal assertion reads modalBox's declared top and the confirm prompt
// does not go through modalBox.
//
// The invariant has one exception and this test holds it rather than looking
// away from it: a frame of two rows or fewer has no region, and there the
// prompt takes the frame's last row. See overlayFloorTop.
func TestConfirmStaysInsideTranscriptRegion(t *testing.T) {
	for _, w := range []int{38, 40, 60, 80, 120} {
		for h := 1; h <= 40; h++ {
			for composer := 1; composer <= 5; composer++ {
				m := headerModel(t, defaultSession(), w, h)
				m.askConfirm(ConfirmState{
					Prompt: "Start a new session? This clears the transcript.",
					Action: ConfirmNewSession,
				})
				lay := computeLayout(w, h, composer, true)
				top, n, body := overlaidRows(m.overlayConfirm, lay, h)
				if n == 0 {
					t.Errorf("%dx%d composer=%d: the confirm prompt drew nothing (region %d)",
						w, h, composer, lay.TranscriptH)
					continue
				}
				if lay.TranscriptH < 1 {
					// The floor, and the one exception the invariant names. A
					// frame of two rows or fewer has no region to contain
					// anything, so containment is not the question there; that
					// the prompt is on the frame's LAST row is. Asserted rather
					// than skipped, because `continue` on the size where the
					// rules change is how the last hole stayed open.
					if want := h - 1; top != want {
						t.Errorf("%dx%d composer=%d: regionless frame put the prompt on row "+
							"%d, want the last row %d:\n%s",
							w, h, composer, top, want, strings.Join(body, "\n"))
					}
					// And it spans the frame rather than sitting in a centred
					// box: the sentinel row it is drawn over is the status
					// bar's, and any cell of it left showing finishes the row
					// with something the prompt did not say.
					//
					// This read a PREFIX until step 15 — overlaidRows fills its
					// base with a sentinel, and the assertion asked only whether
					// the first cell of it survived. It could not have asked
					// more: the sentinel was "·", which is the bullet the modal
					// notice prints, so "no sentinel anywhere" was unstateable
					// on the other overlay family and the two families' floor
					// assertions were written to match. The prompt's surviving
					// cells are on the TAIL, so the prefix form passed on every
					// frame while "Discard?  y/n" at twenty columns was reading
					// "Discard?  y/nle" over the composer.
					if surv := survivingCells(body[0]); len(surv) > 0 {
						t.Errorf("%dx%d composer=%d: the row beneath survives at %d of the %d "+
							"cells (%v); a regionless prompt replaces the whole row:\n%q",
							w, h, composer, len(surv), lay.ContentW, surv, body[0])
					}
					continue
				}
				regionTop, regionEnd := lay.HeaderH(), lay.HeaderH()+lay.TranscriptH
				if top < regionTop || top+n > regionEnd {
					t.Errorf("%dx%d composer=%d: prompt occupies rows [%d,%d) but the "+
						"transcript region is [%d,%d) — it is painting over the header, the "+
						"status bar or the composer:\n%s",
						w, h, composer, top, top+n, regionTop, regionEnd,
						strings.Join(body, "\n"))
				}
			}
		}
	}
}

// TestConfirmSitsAtTheFootOfTheRegion pins where in the region it lands.
//
// Bottom-aligned rather than top-aligned: the prompt is asking about the
// composer, and reading best means sitting immediately above it. Containment
// alone would be satisfied by the top of the region too, so without this the
// fix for the status-bar overlap could drift back to the wrong end and still
// pass every other assertion here.
func TestConfirmSitsAtTheFootOfTheRegion(t *testing.T) {
	for _, w := range []int{40, 80, 120} {
		for h := 6; h <= 40; h++ {
			for composer := 1; composer <= 5; composer++ {
				m := headerModel(t, defaultSession(), w, h)
				m.askConfirm(ConfirmState{Prompt: "Clear all session grants?", Action: ConfirmClearGrants})
				lay := computeLayout(w, h, composer, true)
				top, n, _ := overlaidRows(m.overlayConfirm, lay, h)
				if n == 0 {
					continue
				}
				if want := lay.HeaderH() + lay.TranscriptH - n; top != want {
					t.Errorf("%dx%d composer=%d: prompt top row %d, want %d (the foot of a "+
						"%d-row region, for a %d-row prompt)",
						w, h, composer, top, want, lay.TranscriptH, n)
				}
			}
		}
	}
}

// confirmPrompts returns every prompt string the app actually raises, taken
// from the call sites rather than retyped.
//
// The fixture is derived because a retyped one is what hid the confirm prompt's
// worst frame for two milestones. TestConfirmDegradesRatherThanVanishing below
// already carried the right assertion, over the full size space, and passed only
// because its hand-written prompt was seventeen cells. The longest prompt the app
// raises is forty-nine, and at 40 columns — the advertised minimum, ui-spec §1 —
// it named neither answer key; substituting it and changing nothing else failed
// 465 of the 860 frames where the prompt is drawn. Both halves were in this file
// the whole time and never met, because the test holding the long prompt
// (TestConfirmStaysInsideTranscriptRegion) measures placement, not legibility.
//
// Driving the handlers is what stops that recurring. A prompt lengthened in
// update.go arrives here without anyone remembering to widen a literal, and a
// call site that stops raising a prompt fails loudly rather than silently
// shrinking the fixture.
func confirmPrompts(t *testing.T) []string {
	t.Helper()
	var out []string
	raise := func(name string, fn func(Model) (tea.Model, tea.Cmd)) {
		m := headerModel(t, defaultSession(), 80, 24)
		next, _ := fn(m)
		got, ok := next.(Model)
		if !ok {
			t.Fatalf("%s: handler returned %T, not a Model", name, next)
		}
		if got.confirm == nil {
			t.Fatalf("%s: raised no confirm prompt; this fixture no longer covers it", name)
		}
		out = append(out, got.confirm.Prompt)
	}
	// /new while a turn is in flight. update.go, runSlash.
	raise("/new", func(m Model) (tea.Model, tea.Cmd) {
		m.busy.Active = true
		return m.runSlash("new", "", m.layout())
	})
	// /approvals with grants to clear. update.go, runSlash.
	raise("/approvals", func(m Model) (tea.Model, tea.Cmd) {
		m.status.Grants = 3
		return m.runSlash("approvals", "", m.layout())
	})
	// A paste over the §7.2 warning threshold. update.go, handlePaste. The size
	// is part of the prompt, so this one has no single length — twelve kilobytes
	// is representative, and a four-digit paste adds two cells to it.
	raise("paste", func(m Model) (tea.Model, tea.Cmd) {
		return m.handlePaste(strings.Repeat("x", 12<<10))
	})
	return out
}

// TestConfirmDegradesRatherThanVanishing is the confirm prompt's half of the
// invisible-modal rule. It traps every key but y, n and Esc, so a region too
// small for the bordered box gets the prompt as a bare row — the alternative is
// a frame that looks idle and answers nothing.
//
// Three things are asserted of every drawn frame, and the second and third are
// what the bordered form failed until the width ladder existed:
//
//  1. the row count matches the rung the region height selects;
//  2. the row names BOTH answer keys, not either — "[y] yes⋯" is a frame that
//     offers one of the two answers to a destructive question;
//  3. the row still says what is being answered, so "y/n" alone is not a pass.
//
// Every frame is a drawn frame now. This carried a `lay.TranscriptH == 0` arm
// that asserted the prompt drew NOTHING there and skipped the other three
// checks — the invisible trap stated as a requirement, and by some distance the
// most expensive line in this file: it read as coverage of exactly the case it
// exempted. The region is reserved in computeLayout now, and below the size
// where that is affordable the prompt takes the frame's last row, so no size
// draws nothing and the arm has no subject left.
//
// It walks every prompt the app raises rather than one chosen by hand. See
// confirmPrompts.
func TestConfirmDegradesRatherThanVanishing(t *testing.T) {
	for _, prompt := range confirmPrompts(t) {
		for _, w := range []int{38, 40, 60, 80, 120} {
			for h := 1; h <= 40; h++ {
				for composer := 1; composer <= 5; composer++ {
					m := headerModel(t, defaultSession(), w, h)
					m.askConfirm(ConfirmState{Prompt: prompt, Action: ConfirmLargePaste})
					lay := computeLayout(w, h, composer, true)
					_, n, body := overlaidRows(m.overlayConfirm, lay, h)
					switch {
					case lay.TranscriptH < 3:
						if n != 1 {
							t.Errorf("%q %dx%d composer=%d: %d row(s) in a %d-row region, want 1",
								prompt, w, h, composer, n, lay.TranscriptH)
						}
					default:
						if n != 3 {
							t.Errorf("%q %dx%d composer=%d: %d row(s) in a %d-row region, want "+
								"the 3-row bordered box", prompt, w, h, composer, n, lay.TranscriptH)
						}
					}
					joined := strings.Join(body, "\n")
					both := strings.Contains(joined, confirmKeysShort) ||
						strings.Contains(joined, "[y] yes") && strings.Contains(joined, "[n] no")
					if !both {
						t.Errorf("%q %dx%d composer=%d: prompt does not name both keys that "+
							"answer it:\n%s", prompt, w, h, composer, joined)
					}
					// The keys outrank the question, but not to the point of
					// replacing it: a row reading "y/n" over a question the user
					// cannot see asks them to agree to something unnamed.
					if !strings.Contains(joined, firstWord(prompt)) {
						t.Errorf("%q %dx%d composer=%d: the row names the keys but not what "+
							"they answer:\n%s", prompt, w, h, composer, joined)
					}
				}
			}
		}
	}
}

// firstWord is the shortest span of a prompt that identifies it, for the
// assertion that the question survives alongside the answer keys.
//
// Deliberately minimal, and stated so nobody reads more into it than it claims.
// It is a stand-in for "the question survives", not a test of it: a prompt
// beginning "Confirm…" or "Are…" would satisfy it on a row that had dropped
// everything distinguishing. What earns its place is that it kills M30 — the
// mutant that drops the question entirely and returns the answer keys alone —
// which is the failure the assertion exists for. Strengthening it to a span
// that identifies WHICH prompt is drawn means picking that span per prompt, and
// the fixture derives its prompts from the handlers precisely so that nothing
// here has to be written per prompt.
func firstWord(s string) string {
	if i := strings.IndexByte(s, ' '); i > 0 {
		return s[:i]
	}
	return s
}

// TestOverlayReservesARegionRow pins the mechanism the invisible-overlay fix
// turns on, which nothing else in this file can see.
//
// Every other overlay assertion reads what the overlay DREW. That is what let
// the hole survive two reviews: the overlay and the layout each behaved
// reasonably on their own, and the defect lived in the fact that the layout was
// never asked for a region and the overlay concluded it had none. With the floor
// underneath now drawing a row either way, a fix that removed the reservation
// entirely would leave every drawing assertion in this file passing. So the
// reservation is asserted here, against the geometry, and not through a render.
//
// Two halves, and the second matters as much as the first:
//
//  1. While an overlay is open, a frame of three rows or more has a region row.
//     Three, because two rows are a status bar and a composer row with nothing
//     left to spend, and that is the floor's subject rather than this one's.
//  2. Where the region already had a row, opening an overlay changes NOTHING.
//     The reservation is a floor on the existing ladder, not a second one, and
//     an overlay that reflowed a frame it fitted inside would move the thirteen
//     reference grids and every transcript offset with them.
func TestOverlayReservesARegionRow(t *testing.T) {
	for _, w := range []int{38, 40, 60, 80, 120} {
		for h := 1; h <= 40; h++ {
			for composer := 1; composer <= 5; composer++ {
				plainLay := computeLayout(w, h, composer, false)
				overLay := computeLayout(w, h, composer, true)

				if h >= 3 && overLay.TranscriptH < 1 {
					t.Errorf("%dx%d composer=%d: an open overlay was given a region of %d "+
						"row(s); it has nowhere to draw and every key is trapped",
						w, h, composer, overLay.TranscriptH)
				}
				// The status bar is never what pays. ui-spec §2.2 and the
				// ladder's own ordering; a reservation taken from the bar would
				// buy visibility for the overlay with the state of the session.
				if h >= 1 && overLay.ComposerH < 1 {
					t.Errorf("%dx%d composer=%d: composer degraded to %d rows",
						w, h, composer, overLay.ComposerH)
				}
				if plainLay.TranscriptH >= 1 && overLay != plainLay {
					t.Errorf("%dx%d composer=%d: opening an overlay reflowed a frame that "+
						"already had a region:\n without: %+v\n with:    %+v",
						w, h, composer, plainLay, overLay)
				}
			}
		}
	}
}

// TestFrameNamesTheWayOutOfEveryOverlay is the acceptance assertion for the
// whole of it, and the one that would have caught the defect on its own.
//
// Everything else here tests a layer: computeLayout's geometry, modalBox's rows,
// overlayConfirm's placement onto a sentinel frame. The invisible confirm prompt
// passed all three, because each layer was right about its own part and the
// frame the user saw was the composition of them. So this drives View() — the
// real frame, at a real size, with a real draft typed into the composer — and
// asks the only question that matters: can the overlay be answered from what is
// on the screen.
//
// A composer draft is how the defect is reached, and typing is all it takes:
// five lines at 40×10 left no region, and the frame was a header, a status bar
// reading `idle`, and the draft. No question, no answer keys, and `y` still
// applied the action.
//
// Heights run from 1, below anything ui-spec §1 supports, because "unsupported"
// describes what the app promises to lay out well and not what it promises to
// trap a user inside.
func TestFrameNamesTheWayOutOfEveryOverlay(t *testing.T) {
	type overlay struct {
		open func(*Model)
		// names is satisfied if the frame contains any one of these.
		names []string
	}
	overlays := map[string]overlay{
		"help": {
			open:  func(m *Model) { m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()}) },
			names: []string{"Esc"},
		},
		"content": {
			open: func(m *Model) {
				m.openModal(ModalState{Kind: ModalContent, Title: "a file", Lines: []string{"one", "two"}})
			},
			names: []string{"Esc"},
		},
		"diff": {
			open: func(m *Model) {
				m.openModal(ModalState{Kind: ModalDiff, Title: "working tree", Lines: fakeDiff(), Added: 12, Removed: 4})
			},
			names: []string{"Esc"},
		},
	}
	for _, prompt := range confirmPrompts(t) {
		p := prompt
		overlays["confirm: "+firstWord(p)] = overlay{
			open:  func(m *Model) { m.askConfirm(ConfirmState{Prompt: p, Action: ConfirmLargePaste}) },
			names: []string{confirmKeysShort, confirmKeysLong},
		}
	}

	for name, o := range overlays {
		for _, w := range []int{40, 80} {
			for h := 1; h <= 20; h++ {
				for composer := 1; composer <= 5; composer++ {
					m := headerModel(t, defaultSession(), w, h)
					m.comp.SetValue(draft(composer))
					o.open(&m)
					m.relayout(m.layout())
					frame := stripSGR(m.View())

					var named bool
					for _, want := range o.names {
						if strings.Contains(frame, want) {
							named = true
						}
					}
					if !named {
						t.Errorf("%s %dx%d composer=%d: the frame names none of %q — every "+
							"key is trapped and nothing on screen says so:\n%s",
							name, w, h, composer, o.names, boxed(frame))
					}
				}
			}
		}
	}
}

// draft is n lines of composer content, the way a user reaches them: by typing.
func draft(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return strings.Join(lines, "\n")
}

// boxed renders a frame with visible row edges, so a failure shows where the
// frame ends rather than leaving trailing space to be counted by eye.
func boxed(frame string) string {
	rows := strings.Split(frame, "\n")
	for i, r := range rows {
		rows[i] = "  |" + r + "|"
	}
	return strings.Join(rows, "\n")
}

// TestConfirmBoxFloorIsAFloor pins the confirm box's minimum width at the one
// place it stopped being one.
//
// The box was sized with clamp(want, 20, lay.ContentW-2), and clamp returns hi when
// lo > hi — so at twenty columns, where the cap is eighteen and the floor is
// twenty, the floor lost and the box came out at eighteen. Benign in what it
// rendered, and precisely the shape of thing that is benign until it is not:
// confirmRow documents its last arm as unreachable "through overlayConfirm,
// whose box floor is twenty cells", and that sentence was false at exactly one
// width. A floor that yields to the cap is not a floor, it is a preference.
func TestConfirmBoxFloorIsAFloor(t *testing.T) {
	// Long enough that the sizing always wants more than the cap, which is what
	// puts the two bounds in the wrong order at the narrow end.
	const prompt = "Discard the running turn and start a new session?"
	for w := 1; w <= 40; w++ {
		m := headerModel(t, defaultSession(), w, 24)
		m.askConfirm(ConfirmState{Prompt: prompt, Action: ConfirmNewSession})
		got := m.confirmBoxWidth(computeLayout(w, 24, 1, true))
		// Cap then floor: the CONTENT width bounds it from above — the frame's
		// margin is not the box's to spend — confirmMinW from below, and at the
		// narrow end the floor is what stands.
		_, contentW := framePadding(w)
		want := maxInt(confirmMinW, contentW-2)
		if got != want {
			t.Errorf("width=%d: box is %d cells, want %d (cap %d, floor %d)",
				w, got, want, contentW-2, confirmMinW)
		}
	}
	// The floor's whole purpose, stated where confirmRow states it: the box is
	// never narrow enough to reach the arm that drops the question entirely.
	m := headerModel(t, defaultSession(), confirmMinW, 24)
	m.askConfirm(ConfirmState{Prompt: prompt, Action: ConfirmNewSession})
	inner := m.confirmBoxWidth(computeLayout(confirmMinW, 24, 1, true)) - 4
	if room := inner + 1 - cellWidth(confirmGap) - cellWidth(confirmKeysShort); room < 1 {
		t.Errorf("at %d columns the box leaves %d cells for the question; confirmRow's "+
			"last arm is reachable and its comment says it is not", confirmMinW, room)
	}
}

// TestFitWidestPicksTheFirstThatFits pins the shared ladder helper directly.
//
// Every caller in this package passes a floor short enough to fit at the
// narrowest width the callers can reach — "Esc or ?" is eight cells and the
// narrowest modal is twenty — so the truncating fallback is unreachable through
// them today. That makes it exactly the branch a mutation run cannot reach
// either, and the branch that will be load-bearing the first time a footer
// ladder is written with a longer floor. Pinning it here is cheaper than
// discovering it then.
func TestFitWidestPicksTheFirstThatFits(t *testing.T) {
	variants := []string{"the widest form", "middle", "min"}
	cases := []struct {
		name     string
		variants []string
		w        int
		want     string
	}{
		{"widest fits", variants, 20, "the widest form"},
		{"exactly the widest", variants, 15, "the widest form"},
		{"one short of the widest", variants, 14, "middle"},
		{"only the floor fits", variants, 5, "min"},
		{"nothing fits: the floor truncates", variants, 2, "m…"},
		{"no variants", nil, 10, ""},
		{"a single variant is just truncEnd", []string{"abcdef"}, 4, "abc…"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := fitWidest(c.variants, c.w, "…"); got != c.want {
				t.Errorf("fitWidest(%q, %d) = %q, want %q", c.variants, c.w, got, c.want)
			}
		})
	}
}

// TestSentinelIsAbsentFromEveryOverlay keeps the sweep below honest.
//
// Every coverage assertion in this file reads "a cell still holding overlayFill
// is a cell the overlay did not write". That sentence is only true while no
// overlay can print overlayFill itself, and the sentinel this file used before
// step 15 could: it was "·", the bullet the modal notice puts between its title
// and its exit key. A colliding sentinel does not fail — it quietly reports
// covered cells as covered when they are not, which is the failure mode that
// makes a green suite worth nothing.
//
// The frames are rendered over a space-filled terminal rather than a sentinel
// one, so anything the assertion finds came out of the renderer.
func TestSentinelIsAbsentFromEveryOverlay(t *testing.T) {
	overlays := map[string]func(m *Model){
		"help":    func(m *Model) { m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()}) },
		"content": func(m *Model) { m.openModal(ModalState{Kind: ModalContent, Title: "a file", Lines: helpLines()}) },
		"diff": func(m *Model) {
			m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: []string{"+a", "-b", "@@ x"}, Added: 1, Removed: 1})
		},
		"confirm": func(m *Model) { m.askConfirm(ConfirmState{Prompt: "Discard?", Action: ConfirmNewSession}) },
	}
	if got := cellWidth(overlayFill); got != 1 {
		t.Fatalf("overlayFill %q is %d cells wide; the sentinel frames are no longer "+
			"one cell per column and every index reported here is off", overlayFill, got)
	}
	for name, open := range overlays {
		for _, w := range []int{20, 40, 80, 120} {
			for _, h := range []int{1, 2, 4, 10, 24} {
				for _, unicode := range []bool{true, false} {
					m := New(Options{
						Version: fixtureVersion,
						Caps:    Caps{Unicode: unicode, Colour: true},
						Session: defaultSession(),
						Status:  Status{Model: "claude-sonnet-5", Family: "sonnet-5"},
					})
					mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
					m = mm.(Model)
					open(&m)
					if frame := m.View(); strings.Contains(frame, overlayFill) {
						t.Errorf("%s %dx%d unicode=%v: the frame prints %q, which is the "+
							"sentinel the coverage assertions treat as an unwritten cell:\n%s",
							name, w, h, unicode, overlayFill, boxed(stripSGR(frame)))
					}
				}
			}
		}
	}
}

// TestOneRowOverlayFormsCoverTheirWholeSpan is the acceptance sweep for the
// four places an overlay is spliced as a single row.
//
// modalNotice and confirmRow are width LADDERS: each returns the widest variant
// that FITS the width it is handed, never a variant padded to it. spliceAt
// replaces only the cells the string occupies, so for four sites — the modal
// notice in-region and on the floor, the confirm prompt's bare row in-region
// and on the floor — the row underneath survived beside the overlay wherever
// the chosen variant was short of its span. The worst frame was not an extreme
// size but 80×10 with a three-line draft: twenty-two cells of help notice and
// forty-two cells of transcript, reading as one row saying two things. At
// twenty columns the confirm prompt's own answer key ran into the composer's
// placeholder and read "y/nle".
//
// It is a sweep of every width from 1 to 200 because the defect is a function
// of the gap between a variant's length and its span, and that gap opens and
// closes as the ladder steps: a sampled width can sit on a rung where the
// widest variant happens to fill the row. It is not a walk of heights: the
// height selects WHICH of the four sites is drawn, and the heights below reach
// all four at every width — which the counters at the end prove rather than
// assume.
//
// Two spans are asserted, because the four sites do not share one:
//
//   - On the floor the overlay is replacing a FRAME row — the status bar's or
//     the composer's — so it owns every cell from column zero.
//   - In-region it is standing in for a bordered box, so it owns exactly the
//     cells that box would have owned and no more. That span is read off the
//     bordered form rendered at the same width rather than recomputed here.
//     Recomputing would be a second copy of modalBox's and confirmBoxWidth's
//     arithmetic, and a mutant that changed the real one would pass against the
//     copy.
//
// The second is the half a "no survivors" assertion cannot state on its own: an
// overlay that covered the whole frame row would leave no survivors either, and
// would be painting over the transcript either side of a box.
func TestOneRowOverlayFormsCoverTheirWholeSpan(t *testing.T) {
	type family struct {
		name string
		open func(m *Model)
		fn   func(m Model) func([]string, Layout) []string
	}
	families := []family{
		{
			name: "modal/help",
			open: func(m *Model) { m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()}) },
			fn:   func(m Model) func([]string, Layout) []string { return m.overlayModal },
		},
		{
			name: "modal/content",
			open: func(m *Model) { m.openModal(ModalState{Kind: ModalContent, Title: "a file", Lines: nil}) },
			fn:   func(m Model) func([]string, Layout) []string { return m.overlayModal },
		},
	}
	for _, prompt := range confirmPrompts(t) {
		p := prompt
		families = append(families, family{
			name: "confirm/" + firstWord(p),
			open: func(m *Model) { m.askConfirm(ConfirmState{Prompt: p, Action: ConfirmLargePaste}) },
			fn:   func(m Model) func([]string, Layout) []string { return m.overlayConfirm },
		})
	}

	var floorSeen, regionSeen int
	for _, f := range families {
		for w := 1; w <= 200; w++ {
			// One model per width. The overlay functions read the Layout they
			// are handed and the model's own content, never m.width or
			// m.height, so the height is varied through computeLayout alone.
			m := headerModel(t, defaultSession(), w, 40)
			f.open(&m)
			draw := f.fn(m)

			// The span a bordered form owns at this width. Both box widths are
			// functions of the terminal width and the content alone — not of
			// the region — so a tall layout at this width draws the box the
			// one-row form is standing in for.
			tall := computeLayout(w, 40, 1, true)
			if tall.TranscriptH < 3 {
				t.Fatalf("%s width=%d: the reference layout has a %d-row region and draws no "+
					"bordered box; there is nothing to compare the one-row form against",
					f.name, w, tall.TranscriptH)
			}
			_, refN, refBody := overlaidRows(draw, tall, 40)
			if refN < 3 {
				t.Fatalf("%s width=%d: the reference overlay is %d row(s), not a bordered box",
					f.name, w, refN)
			}
			refSurv := survivingCells(refBody[0])

			// Heights 1..12 cover every region of two rows or fewer at every
			// composer height: with the header the chrome is 5+composer, so a
			// region of 2 needs h <= 7+composer <= 12, and without it the
			// bands are lower still.
			for h := 1; h <= 12; h++ {
				for composer := 1; composer <= 5; composer++ {
					lay := computeLayout(w, h, composer, true)
					top, n, body := overlaidRows(draw, lay, h)
					if n == 0 {
						t.Errorf("%s %dx%d composer=%d: the overlay drew nothing (region %d)",
							f.name, w, h, composer, lay.TranscriptH)
						continue
					}
					if n != 1 {
						continue // a bordered box; this sweep is the one-row forms
					}
					surv := survivingCells(body[0])
					if lay.TranscriptH < 1 {
						floorSeen++
						if len(surv) > 0 {
							t.Errorf("%s %dx%d composer=%d: floor row on frame row %d keeps %d of "+
								"%d cells of the row beneath, at %v:\n  got  %q\n  want the whole "+
								"row written",
								f.name, w, h, composer, top, len(surv), lay.ContentW, surv, body[0])
						}
						continue
					}
					regionSeen++
					if !sameCells(surv, refSurv) {
						t.Errorf("%s %dx%d composer=%d: the in-region one-row form covers "+
							"different cells from the bordered box at this width.\n"+
							"  one-row  %q\n    survivors %v\n"+
							"  bordered %q\n    survivors %v",
							f.name, w, h, composer, body[0], surv, refBody[0], refSurv)
					}
				}
			}
		}
	}
	// A sweep that reached neither rung would pass in silence, and both rungs
	// moved during this mission.
	if floorSeen == 0 || regionSeen == 0 {
		t.Fatalf("the sweep reached %d floor frame(s) and %d in-region one-row frame(s); "+
			"a rung it never reaches is a rung it does not cover", floorSeen, regionSeen)
	}
	t.Logf("covered %d floor frames and %d in-region one-row frames", floorSeen, regionSeen)
}

// TestFrameKeepsAMarginAtBothEdges is the acceptance sweep for the frame's
// horizontal padding: one blank column at the left edge, one at the right, at
// every width and in every state that draws.
//
// Four claims, because a frame can hold any three of them and still be wrong:
//
//   - Every row opens with the margin. Not "the rows with content on them" —
//     padFrame insets the frame, and a frame whose blank rows are a column
//     narrower than its full ones is a frame that pads its content instead.
//   - No row reaches the right margin. This is the half a left-edge check
//     cannot state, and the half that separates a frame that was inset from a
//     renderer that indented itself: the two are identical on the left and
//     differ by a column on the right.
//   - TermW == ContentW + 2*Pad. The margin is bought once. A layout that took
//     its padding out of the content AND inset the result would satisfy both
//     edge checks while giving the content two columns less than it says.
//   - Where the layout draws a rule, some row reaches the last content column.
//     Without it every claim above is satisfied by a frame that renders
//     nothing at all, and the margin would be pinned on an empty screen.
//
// The widths run from 1 so the band where the margin is dropped is walked
// rather than reasoned about: below three columns there is nothing to frame,
// framePadding says so, and this is where that claim is checked against a
// render.
func TestFrameKeepsAMarginAtBothEdges(t *testing.T) {
	states := map[string]func(m *Model){
		"empty": func(m *Model) {},
		"turn": func(m *Model) {
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"Fix the Divide validation"}}})
		},
		"modal": func(m *Model) { m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()}) },
		"confirm": func(m *Model) {
			m.askConfirm(ConfirmState{Prompt: "Discard the running turn?", Action: ConfirmNewSession})
		},
	}
	var ruledSeen, unpaddedSeen int
	for name, build := range states {
		for w := 1; w <= 120; w++ {
			for _, h := range []int{1, 2, 5, 6, 10, 11, 24} {
				m := headerModel(t, defaultSession(), w, h)
				build(&m)
				lay := m.layout()

				if lay.TermW != lay.ContentW+2*lay.Pad {
					t.Errorf("%s %dx%d: TermW=%d but ContentW=%d + 2*Pad=%d; the margin is "+
						"being spent twice or not at all",
						name, w, h, lay.TermW, lay.ContentW, lay.Pad)
				}
				if lay.Pad == 0 {
					unpaddedSeen++
				}

				rows := strings.Split(m.View(), "\n")
				margin := strings.Repeat(" ", lay.Pad)
				widest := 0
				for i, row := range rows {
					if !strings.HasPrefix(row, margin) {
						t.Errorf("%s %dx%d row %d: %q does not open with the %d-column margin",
							name, w, h, i+1, row, lay.Pad)
					}
					if cw := cellWidth(row); cw > lay.TermW-lay.Pad {
						t.Errorf("%s %dx%d row %d: %d cells of a %d-column terminal, so it has "+
							"written into the right margin: %q",
							name, w, h, i+1, cw, lay.TermW, row)
					} else if cw > widest {
						widest = cw
					}
				}
				// A rule fills the content span, so where one is drawn the
				// widest row is exactly the margin plus the content.
				if lay.ShowRules && lay.ContentW > 0 {
					ruledSeen++
					if want := lay.Pad + lay.ContentW; widest != want {
						t.Errorf("%s %dx%d: widest row is %d cells, want %d (margin %d + content %d); "+
							"the rule is not reaching the last content column",
							name, w, h, widest, want, lay.Pad, lay.ContentW)
					}
				}
			}
		}
	}
	// Both bands are walked, and a band this sweep never entered is a band it
	// does not cover.
	if ruledSeen == 0 || unpaddedSeen == 0 {
		t.Fatalf("the sweep saw %d ruled frame(s) and %d unpadded frame(s); "+
			"a band it never reaches is a band it does not cover", ruledSeen, unpaddedSeen)
	}
	t.Logf("covered %d ruled frames and %d frames too narrow for a margin", ruledSeen, unpaddedSeen)
}
