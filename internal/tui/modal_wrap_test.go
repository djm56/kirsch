package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for long lines in the modal (ui-spec §4.1, amendment 103): a body line
// wider than the text width wraps onto continuation rows, and the modal's scroll
// offset counts display rows.
//
// Two entry points are used and each test says which. modalRows and
// modalRowCount take the text width as a parameter, so a test that is about
// where a line breaks names the width itself; modalBox, modalExtent and the key
// path (Update, which dispatches to keyModal) take a Layout, so a test about the
// box or the scroll range goes through those. The plain Models here have colour
// off, so a drawn row is its text; the coloured ones are marked.

// drawRows draws every display row of line idx of the open modal at inner cells.
func drawRows(m Model, idx, inner int) []string {
	return m.modalRows(m.modal, m.modal.Lines[idx], idx, inner)
}

// plainRows is drawRows with SGR removed.
func plainRows(m Model, idx, inner int) []string {
	rows := drawRows(m, idx, inner)
	for i := range rows {
		rows[i] = stripSGR(rows[i])
	}
	return rows
}

// wrapModel is a plain (no colour) Unicode Model at w x h with no modal open.
func wrapModel(t *testing.T, w, h int) Model {
	t.Helper()
	return newMDModel(t, false, true, false, w, h)
}

// bodyText returns the plain text of a box's body rows: the box rows between the
// top border and the rule or footer, without the border cells and the pad. The
// rows are those that begin with the vertical border, less the footer row.
func bodyText(t *testing.T, box []string) []string {
	t.Helper()
	var rows []string
	for _, r := range box[1 : len(box)-1] {
		p := stripSGR(r)
		if !strings.HasPrefix(p, "│") {
			continue // the rule
		}
		rows = append(rows, strings.TrimRight(strings.TrimSuffix(strings.TrimPrefix(p, "│ "), "│"), " "))
	}
	if len(rows) == 0 {
		return nil
	}
	return rows[:len(rows)-1] // the footer
}

// allRows is every display row of the open modal at the layout's text width,
// worked out from modalRows alone, line by line: the oracle the scroll tests
// compare the box against. It does not go through modalExtent.
func allRows(m Model, lay Layout) []string {
	inner := modalFrameFor(lay).inner
	var out []string
	for i := range m.modal.Lines {
		for _, r := range drawRows(m, i, inner) {
			out = append(out, strings.TrimRight(stripSGR(r), " "))
		}
	}
	return out
}

// scrollModal is a content modal with a mix of one-row and three-row lines, on a
// plain Model at w x h. long is the width of a line that takes exactly three
// rows at the box's text width. n is the number of source lines; every third is long.
func scrollModal(t *testing.T, w, h, n int) (Model, string) {
	t.Helper()
	m := wrapModel(t, w, h)
	textW := modalFrameFor(m.layout()).inner - 5
	long := strings.Repeat("w", 2*textW+3)
	var lines []string
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			lines = append(lines, fmt.Sprintf("L%02d%s", i, long[3:]))
		} else {
			lines = append(lines, fmt.Sprintf("short %d", i))
		}
	}
	m.openModal(ModalState{Kind: ModalContent, Title: "scroll", Lines: lines})
	return m, long
}

// TestContentWrapAtTextWidthBoundary: entry point modalRows at inner 40, so the
// text width is 35. A line of exactly 35 cells is one row; one cell more is two,
// the second holding the one cell over. Both go through the same call, so the
// one-row result is the control for the two-row one.
func TestContentWrapAtTextWidthBoundary(t *testing.T) {
	const inner = 40
	const textW = inner - 5
	exact := strings.Repeat("x", textW)
	m := wrapModel(t, 100, 30)
	m.openModal(ModalState{Kind: ModalContent, Title: "t", Lines: []string{exact, exact + "y"}})

	if rows := plainRows(m, 0, inner); len(rows) != 1 || rows[0] != "   1 "+exact {
		t.Errorf("a line exactly at the text width: %q, want one row", rows)
	}
	rows := plainRows(m, 1, inner)
	if len(rows) != 2 || rows[0] != "   2 "+exact || rows[1] != "     y" {
		t.Errorf("a line one cell over the text width: %q, want its first %d cells, then %q", rows, textW, "     y")
	}
}

// TestContentWrapNeverSplitsWideRune: entry point modalRows at inner 40 (text
// width 35). A wide rune that would straddle the boundary starts the next row, so
// the first row is a cell short; the same rune one cell earlier fits. The rows
// laid end to end are the line.
func TestContentWrapNeverSplitsWideRune(t *testing.T) {
	const inner = 40
	m := wrapModel(t, 100, 30)
	straddle := strings.Repeat("x", 34) + "世" + "z"
	fits := strings.Repeat("x", 33) + "世"
	allWide := strings.Repeat("世", 30)
	m.openModal(ModalState{Kind: ModalContent, Title: "t", Lines: []string{straddle, fits, allWide}})

	if rows := plainRows(m, 1, inner); len(rows) != 1 || rows[0] != "   2 "+fits {
		t.Fatalf("control: the wide rune ending exactly at the boundary: %q, want one row", rows)
	}
	rows := plainRows(m, 0, inner)
	if len(rows) != 2 || rows[0] != "   1 "+strings.Repeat("x", 34) || rows[1] != "     世z" {
		t.Errorf("a wide rune at the boundary: %q, want it moved whole to the second row", rows)
	}
	rows = plainRows(m, 2, inner)
	var joined strings.Builder
	for i, r := range rows {
		if w := cellWidth(r); w > inner {
			t.Errorf("row %d is %d cells, over the %d-cell text area", i, w, inner)
		}
		joined.WriteString(strings.TrimPrefix(strings.TrimPrefix(r, "   3 "), "     "))
	}
	if joined.String() != allWide || len(rows) != 2 {
		t.Errorf("a line of wide runes: %d rows joining to %q, want 2 rows joining to the line", len(rows), joined.String())
	}
}

// TestContentWrapNumberBlankOnContinuation: entry point modalBox. The modal's own
// number is on the first row of a source line and the continuation row has five
// blank columns in its place; the next line's number follows. A one-row line
// numbered either side is the control that numbers do print.
func TestContentWrapNumberBlankOnContinuation(t *testing.T) {
	m := wrapModel(t, 100, 30)
	inner := modalFrameFor(m.layout()).inner
	textW := inner - 5
	m.openModal(ModalState{Kind: ModalContent, Title: "t", Lines: []string{
		"aaa", strings.Repeat("x", textW+10), "ccc",
	}})
	rows := bodyText(t, mustBox(m))
	want := []string{
		"   1 aaa",
		"   2 " + strings.Repeat("x", textW),
		"     " + strings.Repeat("x", 10),
		"   3 ccc",
	}
	if len(rows) < len(want) {
		t.Fatalf("body has %d rows, want at least %d: %q", len(rows), len(want), rows)
	}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("body row %d = %q, want %q", i, rows[i], w)
		}
	}
}

// TestDiffWrapMarkerBlankOnCodeContinuation: entry point modalRows on a patch
// approval diff at inner 40. For "+", "-" and context lines alike, the marker is
// the first cell of the first row, and each continuation row has one blank column
// where it was and then the rest of the line, which laid end to end is the line.
// The first row starting with the marker is the control for the blank.
func TestDiffWrapMarkerBlankOnCodeContinuation(t *testing.T) {
	const inner = 40
	m := wrapModel(t, 100, 30)
	body := strings.Repeat("abcdefghij", 10)
	var lines []string
	for _, mk := range []string{"+", "-", " "} {
		lines = append(lines, mk+body)
	}
	m.openModal(ModalState{Kind: ModalDiff, Title: "t.unknownext", Lines: lines})
	for idx, mk := range []string{"+", "-", " "} {
		rows := plainRows(m, idx, inner)
		if len(rows) < 3 {
			t.Fatalf("%q line: %d rows, want at least 3", mk, len(rows))
		}
		if !strings.HasPrefix(rows[0], mk+"abc") || cellWidth(rows[0]) != inner {
			t.Errorf("%q line: first row %q, want the marker then the code, %d cells", mk, rows[0], inner)
		}
		joined := rows[0]
		for i, r := range rows[1:] {
			if len(r) < 2 || r[0] != ' ' || r[1] == ' ' {
				t.Errorf("%q line: row %d = %q, want one blank column then code", mk, i+1, r)
			}
			joined += r[1:]
		}
		if joined != mk+body {
			t.Errorf("%q line: rows join to %q, want the line", mk, joined)
		}
	}
}

// TestDiffWrapHeadersFlush: entry point modalRows. A "@@" line and a descriptive
// line wrap flush in a patch approval diff. In /diff (WorkTree) a "---" or "+++"
// header line, between "diff --git" and the first "@@", wraps flush although it
// begins with "-" or "+"; header status decides, not the first character. The
// "+++" code line after the "@@" is the control through the same modal: it is a
// code line, so its continuation row starts with the blank column.
func TestDiffWrapHeadersFlush(t *testing.T) {
	const inner = 40
	body := strings.Repeat("0123456789", 10)

	m := wrapModel(t, 100, 30)
	m.openModal(ModalState{Kind: ModalDiff, Title: "t.unknownext", Lines: []string{
		"@@ " + body, "renamed " + body,
	}})
	for idx, pre := range []string{"@@ ", "renamed "} {
		rows := plainRows(m, idx, inner)
		if len(rows) < 3 || rows[0] != (pre + body)[:inner] {
			t.Fatalf("%q line: rows %q", pre, rows)
		}
		if rows[1] != (pre + body)[inner:2*inner] {
			t.Errorf("%q line: second row %q, want the next %d cells flush", pre, rows[1], inner)
		}
	}

	m = wrapModel(t, 100, 30)
	m.openModal(ModalState{Kind: ModalDiff, Title: "working tree", WorkTree: true, Lines: []string{
		"diff --git a/a.go b/a.go", // 0
		"--- a/" + body,            // 1: header, begins "-"
		"+++ b/" + body,            // 2: header, begins "+"
		"@@ -1 +1 @@",              // 3
		"+++" + body,               // 4: after the first @@, a code line
	}})
	for _, idx := range []int{1, 2} {
		raw := m.modal.Lines[idx]
		rows := plainRows(m, idx, inner)
		if len(rows) < 3 || rows[0] != raw[:inner] || rows[1] != raw[inner:2*inner] {
			t.Errorf("header line %d: rows %q, want flush rows of %d cells", idx, rows, inner)
		}
	}
	rows := plainRows(m, 4, inner)
	raw := m.modal.Lines[4]
	if len(rows) < 3 || rows[0] != raw[:inner] || rows[1] != " "+raw[inner:inner+inner-1] {
		t.Errorf("control: the +++ code line: rows %q, want a blank column on the continuation", rows)
	}
}

// TestDiffWrapLineAfterTruncationMarkerIsCode: entry point modalRows on a /diff
// (WorkTree) modal at inner 40. A truncation marker ends the file header run, so a
// "---" line after it is a code line (blank column on its continuation), while the
// "---" line before it, inside the header run, is a header line (flush). The
// marker line itself is neither and takes one row.
func TestDiffWrapLineAfterTruncationMarkerIsCode(t *testing.T) {
	const inner = 40
	body := strings.Repeat("0123456789", 10)
	m := wrapModel(t, 100, 30)
	m.openModal(ModalState{Kind: ModalDiff, Title: "working tree", WorkTree: true, Lines: []string{
		"diff --git a/a.go b/a.go",
		"--- a/" + body, // 1: inside the header run
		ToolTruncMarker, // 2
		"---" + body,    // 3: after the marker
	}})
	before := plainRows(m, 1, inner)
	if len(before) < 3 || before[1] != m.modal.Lines[1][inner:2*inner] {
		t.Errorf("control: the header line before the marker: %q, want it flush", before)
	}
	if rows := plainRows(m, 2, inner); len(rows) != 1 {
		t.Errorf("the marker line: %q, want one row", rows)
	}
	after := plainRows(m, 3, inner)
	raw := m.modal.Lines[3]
	if len(after) < 3 || after[1] != " "+raw[inner:inner+inner-1] {
		t.Errorf("the line after the marker: %q, want a code line with a blank column", after)
	}
}

// TestWrapHighlightedTokenKeepsRoleAcrossRows: entry point modalRows on a coloured
// read_file modal (openCard, which sets the path) at inner 40. A string token cut
// across two rows is the string colour at the end of the first row and at the start
// of the second. The same line at a width it fits is the control: its string token
// is one stretch in the same colour.
func TestWrapHighlightedTokenKeepsRoleAcrossRows(t *testing.T) {
	line := `1   s := "` + strings.Repeat("a", 60) + `"`
	m := openCard(t, hlColourModel(t), "read_file", "a.go", []string{line})
	whole := hlSegs(t, drawRows(m, 0, 200)[0])
	if last := whole[len(whole)-1]; last.fg != cStr || !strings.HasSuffix(last.text, `a"`) {
		t.Fatalf("control: the unwrapped line ends in %v, want its string token in %s", last, cStr)
	}
	rows := drawRows(m, 0, 40)
	if len(rows) != 3 {
		t.Fatalf("%d rows, want 3", len(rows))
	}
	first := hlSegs(t, rows[0])
	second := hlSegs(t, strings.TrimPrefix(rows[1], "     "))
	if first[len(first)-1].fg != cStr {
		t.Errorf("first row ends in %v, want the string colour", first[len(first)-1])
	}
	if second[0].fg != cStr || !strings.HasPrefix(second[0].text, "a") {
		t.Errorf("second row starts with %v, want the string colour", second[0])
	}
}

// TestWrapPlainLineKeepsWholeLineColour: entry point modalRows on coloured patch
// approval diffs at inner 40. A line drawn the plain way keeps its whole-line
// colour on every row: "@@" hunk, "+" success, "-" error, context muted, where the
// language is unmapped; and a "+" line over the 512-column highlighting cutoff,
// where the language is mapped. A short mapped "+" line has more than one colour,
// the control that the cutoff line really did lose its syntax colours.
func TestWrapPlainLineKeepsWholeLineColour(t *testing.T) {
	const inner = 40
	body := strings.Repeat("ab", 40)

	check := func(t *testing.T, m Model, idx int, want string, blank bool) {
		t.Helper()
		rows := drawRows(m, idx, inner)
		if len(rows) < 2 {
			t.Fatalf("line %d: %d rows, want a wrapped line", idx, len(rows))
		}
		for i, r := range rows {
			if blank && i > 0 {
				r = strings.TrimPrefix(r, " ")
			}
			if segs := hlSegs(t, r); len(segs) != 1 || segs[0].fg != want {
				t.Errorf("line %d row %d: %v, want one stretch in %s", idx, i, segs, want)
			}
		}
	}

	t.Run("unmapped language", func(t *testing.T) {
		m := hlColourModel(t)
		m.openModal(ModalState{Kind: ModalDiff, Title: "x.unknownext", Lines: []string{
			"@@ " + body, "+" + body, "-" + body, " " + body,
		}})
		check(t, m, 0, cHunk, false)
		check(t, m, 1, cOK, true)
		check(t, m, 2, cErr, true)
		check(t, m, 3, cMuted, true)
	})

	t.Run("over the cutoff", func(t *testing.T) {
		m := hlColourModel(t)
		over := "+" + strings.Repeat("a", hlMaxLineCols+88)
		m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: []string{over, "+x := 1"}})
		check(t, m, 0, cOK, true)
		if segs := hlSegs(t, drawRows(m, 1, inner)[0]); len(segs) < 2 {
			t.Errorf("control: the short mapped line is %v, want several colours", segs)
		}
	})
}

// TestHelpLinesAreCutNotWrapped: entry point modalRowCount and modalRows at inner
// 30, narrower than the widest help line. Every help line is one row, and a line
// wider than inner ends in the truncation glyph. The same lines in a content modal,
// through the same two calls, do wrap, which is the control that the narrow width
// bites. modalExtent counts a help line as one row.
func TestHelpLinesAreCutNotWrapped(t *testing.T) {
	const inner = 30
	lines := helpLines()
	m := wrapModel(t, 100, 40)
	m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: lines})
	help := m.modal

	wide := -1
	for i, raw := range lines {
		if cellWidth(raw) > inner {
			wide = i
		}
		if n := help.modalRowCount(m.gly.Trunc, raw, i, inner); n != 1 {
			t.Errorf("help line %d: %d rows, want 1", i, n)
		}
		rows := drawRows(m, i, inner)
		if len(rows) != 1 {
			t.Errorf("help line %d: %d rows drawn, want 1", i, len(rows))
		}
		if cellWidth(raw) > inner && !strings.HasSuffix(stripSGR(rows[0]), m.gly.Trunc) {
			t.Errorf("help line %d: %q, want it cut with %q", i, stripSGR(rows[0]), m.gly.Trunc)
		}
	}
	if wide < 0 {
		t.Fatalf("no help line is wider than %d cells; the test would pass vacuously", inner)
	}
	if total, _ := m.modalExtent(m.layout()); total != len(lines) {
		t.Errorf("modalExtent total = %d, want %d, one row per help line", total, len(lines))
	}

	m.openModal(ModalState{Kind: ModalContent, Title: "same lines", Lines: lines})
	if n := m.modal.modalRowCount(m.gly.Trunc, lines[wide], wide, inner); n < 2 {
		t.Errorf("control: the widest help line is %d row in a content modal at inner %d, want it to wrap", n, inner)
	}
}

// TestModalScrollsByDisplayRow: entry point Update with j and k, on a content
// modal whose first source line takes three rows. One j moves the view by one
// display row, to the line's second row (blank number column), and not to the
// next source line; k moves it back.
func TestModalScrollsByDisplayRow(t *testing.T) {
	m, _ := scrollModal(t, 100, 30, 12)
	rows0 := bodyText(t, mustBox(m))
	if !strings.HasPrefix(rows0[0], "   1 L00") {
		t.Fatalf("first body row %q, want line 1", rows0[0])
	}
	m = drive(m, key('j'))
	if m.modal.Off != 1 {
		t.Errorf("Off after j = %d, want 1", m.modal.Off)
	}
	if got := bodyText(t, mustBox(m))[0]; !strings.HasPrefix(got, "     w") {
		t.Errorf("first body row after j = %q, want line 1's continuation row", got)
	}
	m = drive(m, key('k'))
	if m.modal.Off != 0 || bodyText(t, mustBox(m))[0] != rows0[0] {
		t.Errorf("Off after k = %d, first row %q, want 0 and %q", m.modal.Off, bodyText(t, mustBox(m))[0], rows0[0])
	}
}

// TestModalGoesToLastFullPage: entry point Update with G, End, g, Home and the
// Down keys, on a modal with wrapped lines. G and End land on the top of the last
// full page, total-bodyH, so the body is full and its last row is the modal's last
// row; g and Home go back to the top, which is the control that the body then
// shows the first rows. j and PgDn stop at the last page; after G one k moves up
// one row, with no press lost.
func TestModalGoesToLastFullPage(t *testing.T) {
	m, _ := scrollModal(t, 100, 20, 30)
	lay := m.layout()
	total, bodyH := m.modalExtent(lay)
	last := total - bodyH
	if bodyH < 3 || last < 10 {
		t.Fatalf("total %d, bodyH %d: too little to scroll through", total, bodyH)
	}
	if total == len(m.modal.Lines) {
		t.Fatalf("total %d equals the source line count; nothing wrapped", total)
	}
	all := allRows(m, lay)
	if len(all) != total {
		t.Fatalf("modalRows gives %d rows, modalExtent says %d", len(all), total)
	}
	wantBody := func(off int) []string { return all[off : off+bodyH] }
	same := func(got, want []string) bool { return strings.Join(got, "\n") == strings.Join(want, "\n") }

	for _, k := range []tea.KeyMsg{key('G'), keyType(tea.KeyEnd)} {
		mm := drive(m, k)
		if mm.modal.Off != last {
			t.Errorf("Off after %v = %d, want the last page %d (total %d, body %d)", k, mm.modal.Off, last, total, bodyH)
		}
		if got := bodyText(t, mustBox(mm)); !same(got, wantBody(last)) {
			t.Errorf("body after %v:\n%q\nwant\n%q", k, got, wantBody(last))
		}
	}

	atEnd := drive(m, key('G'))
	for i := 0; i < total+5; i++ {
		atEnd = drive(atEnd, key('j'))
	}
	if atEnd.modal.Off != last {
		t.Errorf("Off after many j = %d, want it held at %d", atEnd.modal.Off, last)
	}
	atEnd = drive(atEnd, keyType(tea.KeyPgDown), keyType(tea.KeyDown))
	if atEnd.modal.Off != last {
		t.Errorf("Off after PgDn and Down at the end = %d, want %d", atEnd.modal.Off, last)
	}
	atEnd = drive(atEnd, key('k'))
	if atEnd.modal.Off != last-1 {
		t.Errorf("Off after k at the end = %d, want %d, no lost press", atEnd.modal.Off, last-1)
	}
	if got := bodyText(t, mustBox(atEnd)); !same(got, wantBody(last-1)) {
		t.Errorf("body after k from the end:\n%q\nwant\n%q", got, wantBody(last-1))
	}

	for _, k := range []tea.KeyMsg{key('g'), keyType(tea.KeyHome)} {
		mm := drive(drive(m, key('G')), k)
		if mm.modal.Off != 0 || !same(bodyText(t, mustBox(mm)), wantBody(0)) {
			t.Errorf("after %v: Off %d, body %q, want the first page", k, mm.modal.Off, bodyText(t, mustBox(mm)))
		}
	}
}

// TestModalFirstKeyAfterResizeMovesView: entry point Update with a
// tea.WindowSizeMsg and then k. A resize leaves the stored offset alone, so after
// the terminal grows taller the stored offset is past the new last page. The render
// shows a clamped copy; the first k clamps first and then steps, so the view moves
// up one row from the last page. Without the first clamp the stored offset would
// step from past the end and the press would show nothing. j from there stays on
// the last page.
func TestModalFirstKeyAfterResizeMovesView(t *testing.T) {
	m, _ := scrollModal(t, 100, 16, 30)
	m = drive(m, key('G'))
	stale := m.modal.Off

	grown := drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	total, bodyH := grown.modalExtent(grown.layout())
	last := total - bodyH
	if grown.modal.Off != stale || stale <= last {
		t.Fatalf("setup: stored offset %d after the resize (was %d), new last page %d; it must be past the range", grown.modal.Off, stale, last)
	}
	all := allRows(grown, grown.layout())
	if got := bodyText(t, mustBox(grown)); strings.Join(got, "\n") != strings.Join(all[last:last+bodyH], "\n") {
		t.Fatalf("the render after the resize is not the clamped last page")
	}

	up := drive(grown, key('k'))
	if up.modal.Off != last-1 {
		t.Errorf("Off after the first k = %d, want %d", up.modal.Off, last-1)
	}
	if got := bodyText(t, mustBox(up)); strings.Join(got, "\n") != strings.Join(all[last-1:last-1+bodyH], "\n") {
		t.Errorf("the first k after the resize did not move the view up one row:\n%q", got)
	}
	down := drive(grown, key('j'))
	if down.modal.Off != last {
		t.Errorf("Off after the first j = %d, want it clamped to %d", down.modal.Off, last)
	}
}

// TestModalScrollRangeAtNoBodyAndNoticeRungs: entry point keyModal with a Layout
// whose region is cut to 5, 4, 3, 2 and 0 rows: the full box, the box without its
// rule, the box without a body, the one-row notice, and no region. Where a body is
// drawn bodyH is positive and G stops at total-bodyH. Where none is, bodyH is 0,
// total is the display-row count, which is more than the line count here, and the
// formulas give [0, total]: G goes to total, j holds there, k steps down one.
func TestModalScrollRangeAtNoBodyAndNoticeRungs(t *testing.T) {
	m, _ := scrollModal(t, 100, 30, 12)
	for _, rows := range []int{5, 4, 3, 2, 0} {
		lay := m.layout()
		lay.TranscriptH = rows
		total, bodyH := m.modalExtent(lay)
		if total <= len(m.modal.Lines) {
			t.Fatalf("rows %d: total %d is not the display-row count (%d source lines)", rows, total, len(m.modal.Lines))
		}
		wantBodyH := map[int]int{5: 1, 4: 1, 3: 0, 2: 0, 0: 0}[rows]
		if bodyH != wantBodyH {
			t.Errorf("rows %d: bodyH = %d, want %d", rows, bodyH, wantBodyH)
		}
		last := total - bodyH

		m.modal.Off = 0
		press := func(k tea.KeyMsg) {
			nm, _ := m.keyModal(k, lay)
			m = nm.(Model)
		}
		press(key('G'))
		if m.modal.Off != last {
			t.Errorf("rows %d: Off after G = %d, want %d", rows, m.modal.Off, last)
		}
		press(key('j'))
		if m.modal.Off != last {
			t.Errorf("rows %d: Off after j at the end = %d, want %d", rows, m.modal.Off, last)
		}
		press(key('k'))
		if m.modal.Off != last-1 {
			t.Errorf("rows %d: Off after k = %d, want %d", rows, m.modal.Off, last-1)
		}
		press(keyType(tea.KeyHome))
		if m.modal.Off != 0 {
			t.Errorf("rows %d: Off after Home = %d, want 0", rows, m.modal.Off)
		}
	}
}

// TestModalClippingMarkerCountsDisplayRows: entry point modalBox. The offset is
// set by five j presses, each through Update, and the box is then built by
// modalBox. The marker's counts are display rows hidden above and below. On a
// modal whose lines wrap, that differs from the number of source lines, which is
// the control that the two are told apart.
func TestModalClippingMarkerCountsDisplayRows(t *testing.T) {
	m, _ := scrollModal(t, 100, 20, 30)
	lay := m.layout()
	total, bodyH := m.modalExtent(lay)
	m = drive(m, key('j'), key('j'), key('j'), key('j'), key('j'))
	off := m.modal.Off
	if off != 5 {
		t.Fatalf("Off = %d, want 5", off)
	}
	raw, _, gotBody, _ := ruleOf(t, mustBox(m), markerUnicode)
	if gotBody != bodyH {
		t.Fatalf("box body is %d rows, modalExtent says %d", gotBody, bodyH)
	}
	want := wantMarker(markerUnicode, off, bodyH, total)
	bySource := wantMarker(markerUnicode, off, bodyH, len(m.modal.Lines))
	if want == bySource {
		t.Fatalf("control: counting source lines gives the same marker %q", want)
	}
	assertMarkerRule(t, markerUnicode, raw, want)
}

// TestModalExtentAgreesWithModalBox: entry point modalExtent against modalBox,
// over content, diff and help modals at several widths and every terminal height
// from 3 to 40. The body height modalExtent gives is the number of body rows the
// box draws (0 for the notice row and the box with no body). At the top the body is
// the first display rows, and after G it is the last bodyH rows, computed from
// modalRows line by line and not from modalExtent.
func TestModalExtentAgreesWithModalBox(t *testing.T) {
	long := strings.Repeat("0123456789", 12)
	content := []string{"one", long, "three", long[:70], "five", "six"}
	diff := []string{"@@ -1,2 +1,2 @@ " + long, " ctx", "-" + long, "+" + long[:90], "+short", " end"}
	for _, kind := range []string{"content", "diff", "help"} {
		for _, w := range []int{40, 60, 80, 120} {
			for h := 3; h <= 40; h++ {
				m := wrapModel(t, w, h)
				switch kind {
				case "content":
					m.openModal(ModalState{Kind: ModalContent, Title: "c", Lines: content})
				case "diff":
					m.openModal(ModalState{Kind: ModalDiff, Title: "t.unknownext", Lines: diff, Added: 2, Removed: 1})
				case "help":
					m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
				}
				lay := m.layout()
				name := fmt.Sprintf("%s %dx%d", kind, w, h)
				total, bodyH := m.modalExtent(lay)
				box := mustBox(m)

				drawn := 0
				if len(box) > 1 {
					drawn = len(bodyText(t, box))
				}
				if drawn != bodyH {
					t.Errorf("%s: box draws %d body rows, modalExtent says %d", name, drawn, bodyH)
					continue
				}
				if bodyH == 0 {
					continue
				}
				all := allRows(m, lay)
				if len(all) != total {
					t.Errorf("%s: modalRows gives %d rows, modalExtent says %d", name, len(all), total)
					continue
				}
				if got, want := bodyText(t, box), all[:minInt(bodyH, total)]; strings.Join(got, "\n") != strings.Join(want, "\n") {
					t.Errorf("%s: top of the body is\n%q\nwant\n%q", name, got, want)
				}
				if total > bodyH {
					at := drive(m, key('G'))
					lastRows := all[total-bodyH:]
					if got := bodyText(t, mustBox(at)); strings.Join(got, "\n") != strings.Join(lastRows, "\n") {
						t.Errorf("%s: after G the body is\n%q\nwant the last page\n%q", name, got, lastRows)
					}
				}
			}
		}
	}
}
