package tui

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// This file turns a finished assistant reply into rows. ui-spec §3.2.
//
// It is in two halves that meet only at mdRow. Layout (mdLayout and everything
// it calls) parses the reply, walks the tree and wraps it to a width, producing
// rows of runs: text, a §10.1 role, and bold, italic and strike flags. It never
// sees a Styles. Drawing (drawMarkdown) turns rows into strings through a
// Styles and is the only function here that does. The layout is cached per
// reply (mdCache), and the live, flattened and identity passes in relayout all
// draw from the same cached rows, which is why strip(styled) == plain holds
// without a separate check.

// mdRole names the §10.1 role a run is drawn in. It is a role and not a Style so
// that rows stay free of styling until drawMarkdown picks the Styles to use.
type mdRole uint8

const (
	roleText       mdRole = iota // body text, code the lexer leaves plain
	roleMuted                    // quote prefix, link URL, comments
	roleAccent                   // headings, function names and builtins
	roleBorder                   // rules, table separators
	roleSynString                // inline code, strings
	roleSynKeyword               // keywords
	roleSynNumber                // numbers and other literals
	roleSynType                  // type and class names
	roleSuccess                  // the "+" of a highlighted diff line
	roleError                    // the "-" of a highlighted diff line
)

// mdStyle is what a run is drawn with: a role and three attributes.
type mdStyle struct {
	Role                 mdRole
	Bold, Italic, Strike bool
}

// mdRun is a fragment of text drawn in one mdStyle. Text holds no newline.
type mdRun struct {
	Text string
	mdStyle
}

// mdRow is one rendered row. A nil row is a blank row.
type mdRow []mdRun

// mdMinContent is the narrowest a nested block's content is laid out. Without
// it a deeply nested list or quote at a small width would leave no cells to put
// text in; the clamp in mdLayout still holds the post-condition if a row ends
// up wider than that.
const mdMinContent = 4

// mdParser parses with CommonMark plus the two extensions the spec's element
// table names: strikethrough and tables. Nothing else is enabled, so a bare URL
// stays plain text and is shown as written. Strikethrough is mdStrikethrough,
// not the library's, so that only a double tilde is strikethrough.
var mdParser = goldmark.New(goldmark.WithExtensions(mdStrikethrough{}, extension.Table))

// mdStrikethrough is the library's strikethrough extension with one change: a
// delimiter run counts only when it is exactly two tildes, and any other run
// stays literal text. The library also strikes a single-tilde pair such as
// "a ~x~ b", dropping the tildes; here that line is drawn as written.
//
// mdStrikeDelims, mdStrikeParser and Extend are derived from goldmark's
// extension/strikethrough.go (MIT, Copyright (c) 2019 Yusuke Inuzuka); see NOTICE.
type mdStrikethrough struct{}

// Extend registers the inline parser. Only the parser is needed: the layout
// reads the tree and nothing renders it to HTML.
func (mdStrikethrough) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(mdStrikeParser{}, 500)))
}

// mdStrikeDelims is the delimiter processor for strikethrough: a tilde run opens
// and closes against another, and a match makes a strikethrough node.
type mdStrikeDelims struct{}

// IsDelimiter reports whether b can make a delimiter run.
func (mdStrikeDelims) IsDelimiter(b byte) bool { return b == '~' }

// CanOpenCloser reports whether opener and closer are the same delimiter.
func (mdStrikeDelims) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

// OnMatch makes the node for a matched pair.
func (mdStrikeDelims) OnMatch(consumes int) ast.Node { return east.NewStrikethrough() }

// mdStrikeParser is the inline parser triggered by a tilde.
type mdStrikeParser struct{}

// Trigger names the byte that starts this parser.
func (mdStrikeParser) Trigger() []byte { return []byte{'~'} }

// Parse pushes a delimiter for a run of exactly two tildes and returns nil for
// any other run, which the parser then reads as text. A run directly after
// another tilde is left alone, as the library's parser leaves it.
func (mdStrikeParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 1, mdStrikeDelims{})
	if node == nil || node.OriginalLength != 2 || before == '~' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

// CloseBlock has nothing to finish.
func (mdStrikeParser) CloseBlock(parent ast.Node, pc parser.Context) {}

// mdLayout parses source and lays it out in rows no wider than width cells,
// with the glyphs g chooses. source is the reply's sanitised lines joined with
// newlines. When g is the ASCII table the text drawn is folded to ASCII after
// the parser has resolved entities, so a character that an entity such as
// &mdash; produces is folded too.
//
// ok is false when the parser or the layout panicked. The caller then draws the
// reply on the plain path: a malformed reply must not take the interface down.
//
// The layout wraps every block to the room it has, so a row is wider than width
// only where nesting leaves a block less than mdMinContent cells; mdClamp, applied
// after it, cuts those rows. mdLayoutRaw is the layout alone, so a test can tell
// the two apart. mdLayout and mdLayoutRaw are test entry points: the reply path
// reaches the layout through mdCache.rows and mdLayoutWith.
func mdLayout(source string, width int, g Glyphs) (rows []mdRow, ok bool) {
	return mdLayoutWith(source, width, g, nil)
}

// mdLayoutWith is mdLayout that calls hook, when it is not nil, at the start of
// the layout, inside the recover. mdCache.rows passes its layoutHook.
func mdLayoutWith(source string, width int, g Glyphs, hook func()) (rows []mdRow, ok bool) {
	rows, ok = mdLayoutHooked(source, width, g, hook)
	if width < 1 {
		width = 1
	}
	for i := range rows {
		rows[i] = mdClamp(rows[i], width)
	}
	return rows, ok
}

// mdLayoutRaw is mdLayout without the final clamp. It is a test entry point; see
// mdLayout.
func mdLayoutRaw(source string, width int, g Glyphs) (rows []mdRow, ok bool) {
	return mdLayoutHooked(source, width, g, nil)
}

// mdLayoutHooked is mdLayoutRaw that calls hook, when it is not nil, inside the
// recover before parsing, so a panic in the hook is a failed layout.
func mdLayoutHooked(source string, width int, g Glyphs, hook func()) (rows []mdRow, ok bool) {
	defer func() {
		if recover() != nil {
			rows, ok = nil, false
		}
	}()
	if hook != nil {
		hook()
	}
	if width < 1 {
		width = 1
	}
	src := []byte(source)
	doc := mdParser.Parser().Parse(text.NewReader(src))
	l := &mdLayouter{src: src, g: g}
	return l.blocks(doc, width), true
}

// mdLayouter carries what the tree walk needs: the source the tree's segments
// index into, and the glyph table.
type mdLayouter struct {
	src []byte
	g   Glyphs
}

// blocks lays out the block children of parent in order, separated by one blank
// row. A child that lays out to no rows adds neither rows nor a separator, so
// runs of blank source lines collapse to one blank row.
func (l *mdLayouter) blocks(parent ast.Node, w int) []mdRow {
	var rows []mdRow
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		cr := l.block(c, w)
		if len(cr) == 0 {
			continue
		}
		if len(rows) > 0 {
			rows = append(rows, nil)
		}
		rows = append(rows, cr...)
	}
	return rows
}

// block lays out one block node at width w. A node kind the table does not name
// falls to l.other, which draws its text in the text role.
func (l *mdLayouter) block(n ast.Node, w int) []mdRow {
	switch n := n.(type) {
	case *ast.Heading:
		head := mdStyle{Role: roleAccent, Bold: true}
		runs := []mdRun{{Text: strings.Repeat("#", n.Level) + " ", mdStyle: head}}
		runs = append(runs, l.inline(n, head)...)
		return l.wrapRuns(runs, w)
	case *ast.Paragraph, *ast.TextBlock:
		return l.wrapRuns(l.inline(n, mdStyle{}), w)
	case *ast.ThematicBreak:
		return []mdRow{{{Text: strings.Repeat(l.g.Sep, w/maxInt(1, cellWidth(l.g.Sep))), mdStyle: mdStyle{Role: roleBorder}}}}
	case *ast.Blockquote:
		cw := maxInt(w-cellWidth(l.g.BoxV+" "), mdMinContent)
		prefix := mdRow{{Text: l.g.BoxV + " ", mdStyle: mdStyle{Role: roleMuted}}}
		return mdPrefix(l.blocks(n, cw), prefix, prefix, false)
	case *ast.List:
		return l.list(n, w)
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		return l.code(n, w)
	case *ast.HTMLBlock:
		return l.html(n, w)
	case *east.Table:
		return l.table(n, w)
	}
	return l.other(n, w)
}

// other draws a node the element table does not name as its text, in the text
// role, with whatever markers the source had. A leaf block shows its source
// lines; a container shows its children.
func (l *mdLayouter) other(n ast.Node, w int) []mdRow {
	if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
		return l.sourceLines(n, w)
	}
	if n.HasChildren() {
		if n.FirstChild().Type() == ast.TypeBlock {
			return l.blocks(n, w)
		}
		return l.wrapRuns(l.inline(n, mdStyle{}), w)
	}
	return nil
}

// sourceLines draws a block's source lines literally, one source line per row
// group, wrapped to w. Consecutive blank lines collapse to one blank row.
func (l *mdLayouter) sourceLines(n ast.Node, w int) []mdRow {
	var rows []mdRow
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		rows = l.appendLiteral(rows, string(seg.Value(l.src)), w)
	}
	if hb, ok := n.(*ast.HTMLBlock); ok && hb.HasClosure() {
		rows = l.appendLiteral(rows, string(hb.ClosureLine.Value(l.src)), w)
	}
	return rows
}

// appendLiteral wraps one literal line and appends its rows, dropping a blank
// row that would follow another.
func (l *mdLayouter) appendLiteral(rows []mdRow, line string, w int) []mdRow {
	line = strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(line) == "" {
		if len(rows) > 0 && len(rows[len(rows)-1]) > 0 {
			rows = append(rows, nil)
		}
		return rows
	}
	return append(rows, l.wrapRuns([]mdRun{{Text: l.g.Fold(line)}}, w)...)
}

// html draws a raw HTML block as literal text. Trailing blank rows are dropped:
// the block separator supplies the gap.
func (l *mdLayouter) html(n *ast.HTMLBlock, w int) []mdRow {
	rows := l.sourceLines(n, w)
	for len(rows) > 0 && len(rows[len(rows)-1]) == 0 {
		rows = rows[:len(rows)-1]
	}
	return rows
}

// code draws a fenced or indented code block: content indented two columns,
// fence lines and info string not drawn, the content's own indentation kept, and
// long lines hard-wrapped at the width less the block indent so every wrapped
// row keeps that indent. A fence whose info string maps to a language is lexed
// one line at a time and each token takes its role; every other block, and a
// fence with an unmapped or missing language, is drawn in the text role.
func (l *mdLayouter) code(n ast.Node, w int) []mdRow {
	const indent = 2
	cw := maxInt(w-indent, mdMinContent)
	pad := mdRow{{Text: strings.Repeat(" ", indent)}}
	var lx chroma.Lexer
	if fc, ok := n.(*ast.FencedCodeBlock); ok && fc.Info != nil {
		lx = hlLexer(hlInfo, string(fc.Info.Value(l.src)))
	}
	var rows []mdRow
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		line := strings.TrimRight(string(seg.Value(l.src)), "\r\n")
		if line == "" {
			rows = append(rows, nil)
			continue
		}
		runs := hlRuns(lx, line)
		for i := range runs {
			runs[i].Text = l.g.Fold(runs[i].Text)
		}
		for _, chunk := range mdHardWrap(mdCells(runs), cw) {
			rows = append(rows, append(append(mdRow{}, pad...), mdRowOf(chunk)...))
		}
	}
	return rows
}

// list lays out a list: a bullet (or the item's own number) in front of each
// item, items one under another, and a blank row between items only when the
// list is loose. A numbered item shows the number written in its own source
// line, so "1." three times stays 1, 1, 1; an item whose line cannot be found
// (see itemNumber) takes the number after the item before it.
func (l *mdLayouter) list(n *ast.List, w int) []mdRow {
	var rows []mdRow
	next := strconv.Itoa(n.Start)
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		marker := l.g.MdBullet + " "
		if n.IsOrdered() {
			if num, ok := l.itemNumber(item); ok {
				next = num
			}
			marker = next + string(n.Marker) + " "
			next = strconv.Itoa(mdAtoi(next) + 1)
		}
		if len(rows) > 0 && !n.IsTight {
			rows = append(rows, nil)
		}
		rows = append(rows, l.listItem(item, marker, w, !n.IsTight)...)
	}
	return rows
}

// mdOrderedMarker matches the end of the text before an item's first content on
// its source line: the digits of an ordered marker, the "." or ")", and the
// spaces after it.
var mdOrderedMarker = regexp.MustCompile(`(\d{1,9})[.)] *$`)

// itemNumber returns the digits written for an ordered item, read from the
// source line its first content starts on. ok is false when that line cannot be
// read: an item with no content, one whose first child is a list or whose first
// line is not directly after the marker (a heading, a quote, a fence), where
// the text before the content does not end in an ordered marker.
func (l *mdLayouter) itemNumber(item ast.Node) (string, bool) {
	c := item.FirstChild()
	for c != nil && c.Lines().Len() == 0 {
		if _, nested := c.(*ast.List); nested {
			return "", false
		}
		c = c.FirstChild()
	}
	if c == nil {
		return "", false
	}
	start := c.Lines().At(0).Start
	if start > len(l.src) {
		return "", false
	}
	before := l.src[:start]
	if i := bytes.LastIndexByte(before, '\n'); i >= 0 {
		before = before[i+1:]
	}
	m := mdOrderedMarker.FindSubmatch(before)
	if m == nil {
		return "", false
	}
	return string(m[1]), true
}

// mdAtoi reads the digits itemNumber returns; they are at most nine, so it
// cannot overflow, and it returns 0 for anything else.
func mdAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// listItem lays out one item. Its first row starts with marker; the rest of the
// item's rows are indented to the text after the marker, except a nested list,
// which is indented two columns. Blank rows are not indented, so an indent adds
// no trailing spaces. An item with no content is its marker alone, without the
// space after it.
func (l *mdLayouter) listItem(item ast.Node, marker string, w int, loose bool) []mdRow {
	mw := cellWidth(marker)
	var rows []mdRow
	first := true
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		ind := mw
		if _, nested := c.(*ast.List); nested && !first {
			ind = 2
		}
		cr := l.block(c, maxInt(w-ind, mdMinContent))
		if len(cr) == 0 {
			continue
		}
		pad := mdRow{{Text: strings.Repeat(" ", ind)}}
		head := pad
		if first {
			head = mdRow{{Text: marker}}
		} else if loose {
			rows = append(rows, nil)
		}
		rows = append(rows, mdPrefix(cr, head, pad, true)...)
		first = false
	}
	if len(rows) == 0 {
		rows = []mdRow{{{Text: strings.TrimRight(marker, " ")}}}
	}
	return rows
}

// table lays out a table: columns padded to align, the separator glyph between
// columns, and a rule under the header row. When the columns do not fit, the
// widest are narrowed and their cells wrap inside the column; when even
// one-cell columns do not fit, the rows are drawn as running text and
// hard-wrapped, so no row is ever wider than w.
func (l *mdLayouter) table(n *east.Table, w int) []mdRow {
	var rows [][][]mdRun // row, column, runs
	var header bool
	for r := n.FirstChild(); r != nil; r = r.NextSibling() {
		var cells [][]mdRun
		for c := r.FirstChild(); c != nil; c = c.NextSibling() {
			cells = append(cells, l.inline(c, mdStyle{}))
		}
		if _, ok := r.(*east.TableHeader); ok && len(rows) == 0 {
			header = true
		}
		rows = append(rows, cells)
	}
	cols := 0
	for _, r := range rows {
		cols = maxInt(cols, len(r))
	}
	if cols == 0 {
		return nil
	}

	sep := mdRow{{Text: " "}, {Text: l.g.BoxV, mdStyle: mdStyle{Role: roleBorder}}, {Text: " "}}
	sepW := cellWidth(" " + l.g.BoxV + " ")
	widths := make([]int, cols)
	for _, r := range rows {
		for c, runs := range r {
			widths[c] = maxInt(widths[c], mdRunsWidth(runs))
		}
	}
	total := sepW * (cols - 1)
	for c := range widths {
		widths[c] = maxInt(widths[c], 1)
		total += widths[c]
	}
	for total > w {
		widest := 0
		for c := range widths {
			if widths[c] > widths[widest] {
				widest = c
			}
		}
		if widths[widest] <= 1 {
			return l.flatTable(rows, header, sep, w)
		}
		widths[widest]--
		total--
	}

	var out []mdRow
	for ri, r := range rows {
		cellLines := make([][]mdRow, cols)
		height := 1
		for c := 0; c < cols; c++ {
			var runs []mdRun
			if c < len(r) {
				runs = r[c]
			}
			cellLines[c] = l.wrapRuns(runs, widths[c])
			if len(cellLines[c]) == 0 {
				cellLines[c] = []mdRow{nil}
			}
			height = maxInt(height, len(cellLines[c]))
		}
		for li := 0; li < height; li++ {
			var row mdRow
			for c := 0; c < cols; c++ {
				var cell mdRow
				if li < len(cellLines[c]) {
					cell = cellLines[c][li]
				}
				row = append(row, cell...)
				if c < cols-1 {
					if gap := widths[c] - mdRunsWidth(cell); gap > 0 {
						row = append(row, mdRun{Text: strings.Repeat(" ", gap)})
					}
					row = append(row, sep...)
				}
			}
			out = append(out, row)
		}
		if header && ri == 0 {
			out = append(out, mdRow{{Text: strings.Repeat(l.g.Sep, total/maxInt(1, cellWidth(l.g.Sep))), mdStyle: mdStyle{Role: roleBorder}}})
		}
	}
	return out
}

// flatTable is table's last resort for a width too small for any column: each
// row becomes its cells joined by the separator, wrapped like a paragraph.
func (l *mdLayouter) flatTable(rows [][][]mdRun, header bool, sep mdRow, w int) []mdRow {
	var out []mdRow
	for ri, r := range rows {
		var runs []mdRun
		for c, cell := range r {
			if c > 0 {
				runs = append(runs, sep...)
			}
			runs = append(runs, cell...)
		}
		out = append(out, l.wrapRuns(runs, w)...)
		if header && ri == 0 {
			out = append(out, mdRow{{Text: strings.Repeat(l.g.Sep, w/maxInt(1, cellWidth(l.g.Sep))), mdStyle: mdStyle{Role: roleBorder}}})
		}
	}
	return out
}

// inline flattens the inline children of n into runs. st is the style the text
// inherits; emphasis, strikethrough, code and links add to it.
func (l *mdLayouter) inline(n ast.Node, st mdStyle) []mdRun {
	var runs []mdRun
	add := func(t string, s mdStyle) {
		if t != "" {
			runs = append(runs, mdRun{Text: l.g.Fold(t), mdStyle: s})
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			add(mdText(c.Segment.Value(l.src)), st)
			// A newline inside a paragraph, hard or soft, ends the row. CommonMark
			// would join a soft break into a space; the reply is drawn as the model
			// broke it, as v0.1.3 drew it, so a list written without markers or a
			// "Name: x" line under another keeps its lines.
			if c.HardLineBreak() || c.SoftLineBreak() {
				add("\n", st)
			}
		case *ast.String:
			add(mdText(c.Value), st)
		case *ast.CodeSpan:
			code := st
			code.Role = roleSynString
			add("`"+l.codeSpanText(c)+"`", code)
		case *ast.Emphasis:
			s := st
			if c.Level >= 2 {
				s.Bold = true
			} else {
				s.Italic = true
			}
			runs = append(runs, l.inline(c, s)...)
		case *east.Strikethrough:
			struck := st
			struck.Strike = true
			add("~~", st)
			runs = append(runs, l.inline(c, struck)...)
			add("~~", st)
		case *ast.Link:
			runs = append(runs, l.inline(c, st)...)
			if dest := mdText(c.Destination); dest != "" {
				add(" ("+dest+")", mdStyle{Role: roleMuted})
			}
		case *ast.AutoLink:
			add(mdText(c.URL(l.src)), mdStyle{Role: roleMuted})
		case *ast.Image:
			runs = append(runs, l.inline(c, st)...)
		case *ast.RawHTML:
			// Raw HTML is shown as written: no entity or backslash resolution.
			for i := 0; i < c.Segments.Len(); i++ {
				seg := c.Segments.At(i)
				add(replaceControls(string(seg.Value(l.src))), st)
			}
		default:
			runs = append(runs, l.inline(c, st)...)
		}
	}
	return runs
}

// codeSpanText is a code span's text, with line endings turned to spaces, as
// the library's own HTML renderer does. Backslashes and entities are literal in
// a code span.
func (l *mdLayouter) codeSpanText(n *ast.CodeSpan) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		t, ok := c.(*ast.Text)
		if !ok {
			continue
		}
		v := string(t.Segment.Value(l.src))
		if strings.HasSuffix(v, "\n") {
			v = v[:len(v)-1] + " "
		}
		b.WriteString(v)
	}
	return replaceControls(b.String())
}

// mdText turns a text segment into the text to show: a backslash before
// punctuation is dropped, and numeric and named entities are resolved, as the
// library's HTML writer does. The result goes through replaceControls because
// an entity such as &#27; resolves to an escape byte after the reply was
// sanitised.
func mdText(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var sb strings.Builder
	resolve := func(chunk []byte) {
		chunk = util.ResolveNumericReferences(chunk)
		chunk = util.ResolveEntityNames(chunk)
		sb.Write(chunk)
	}
	n := 0
	for i := 0; i < len(b); {
		if b[i] == '\\' && i+1 < len(b) && util.IsPunct(b[i+1]) {
			resolve(b[n:i])
			sb.WriteByte(b[i+1])
			i += 2
			n = i
			continue
		}
		i++
	}
	resolve(b[n:])
	return replaceControls(sb.String())
}

// wrapRuns word-wraps runs to w cells and returns the rows.
func (l *mdLayouter) wrapRuns(runs []mdRun, w int) []mdRow {
	lines := mdWrap(mdCells(runs), w)
	rows := make([]mdRow, len(lines))
	for i, ln := range lines {
		rows[i] = mdRowOf(ln)
	}
	return rows
}

// mdPrefix puts first before rows[0] and rest before every other row. With
// skipBlank, a blank row after the first is left blank, so an indent adds no
// trailing spaces; a quote's bar is drawn on blank rows too.
func mdPrefix(rows []mdRow, first, rest mdRow, skipBlank bool) []mdRow {
	out := make([]mdRow, len(rows))
	for i, r := range rows {
		p := rest
		if i == 0 {
			p = first
		}
		if i > 0 && skipBlank && len(r) == 0 {
			continue
		}
		out[i] = append(append(mdRow{}, p...), r...)
	}
	return out
}

// mdCell is one grapheme cluster of a run, the unit the wrapper works in. s is
// "\n" for a hard line break.
type mdCell struct {
	s  string
	w  int
	st mdStyle
}

// mdCells splits runs into grapheme clusters with their cell widths, measured
// the way cellWidth measures a string.
func mdCells(runs []mdRun) []mdCell {
	var cells []mdCell
	for _, r := range runs {
		g := uniseg.NewGraphemes(r.Text)
		for g.Next() {
			s := g.Str()
			w := runewidth.StringWidth(s)
			if s == "\n" {
				w = 0
			}
			cells = append(cells, mdCell{s: s, w: w, st: r.mdStyle})
		}
	}
	return cells
}

// mdRowOf joins cells back into runs, merging neighbours of the same style.
func mdRowOf(cells []mdCell) mdRow {
	var row mdRow
	for _, c := range cells {
		if n := len(row); n > 0 && row[n-1].mdStyle == c.st {
			row[n-1].Text += c.s
			continue
		}
		row = append(row, mdRun{Text: c.s, mdStyle: c.st})
	}
	return row
}

// mdRunsWidth is the cell width of runs laid end to end.
func mdRunsWidth(runs []mdRun) int {
	w := 0
	for _, r := range runs {
		w += runewidth.StringWidth(r.Text)
	}
	return w
}

// mdClamp cuts a row to width cells. It is the post-condition's backstop:
// layout keeps rows within the width by construction, and this only acts when a
// nested block was given less room than mdMinContent.
func mdClamp(r mdRow, width int) mdRow {
	if mdRunsWidth(r) <= width {
		return r
	}
	cells := mdCells(r)
	w, n := 0, 0
	for n < len(cells) && w+cells[n].w <= width {
		w += cells[n].w
		n++
	}
	return mdRowOf(cells[:n])
}

// mdWrap soft-wraps cells to width. A line breaks at a space where one fits and
// the spaces at the break are dropped; a word wider than the width is
// hard-broken; interior runs of spaces are kept; a source line's leading spaces
// are kept, clamped to leave one cell. A "\n" cell ends the line. A cell wider
// than the width, which needs width 1 and a double-width rune, is placed alone
// rather than looping.
func mdWrap(cells []mdCell, width int) [][]mdCell {
	if width < 1 {
		width = 1
	}
	var lines [][]mdCell
	var cur []mdCell
	curW := 0
	hasText := false
	lineStart := true
	flush := func() {
		if !hasText {
			cur = nil
		}
		lines = append(lines, cur)
		cur, curW, hasText = nil, 0, false
	}
	i := 0
	for i < len(cells) {
		if cells[i].s == "\n" {
			flush()
			lineStart = true
			i++
			continue
		}
		j := i
		for j < len(cells) && cells[j].s == " " {
			j++
		}
		k := j
		for k < len(cells) && cells[k].s != " " && cells[k].s != "\n" {
			k++
		}
		gap, word := cells[i:j], cells[j:k]
		i = k
		if len(word) == 0 {
			continue // trailing spaces are dropped
		}
		ww := 0
		for _, c := range word {
			ww += c.w
		}
		if lineStart {
			lineStart = false
			if n := minInt(len(gap), width-1); n > 0 {
				cur = append(cur, gap[:n]...)
				curW = n
			}
			gap = nil
		}
		if curW+len(gap)+ww <= width {
			cur = append(append(cur, gap...), word...)
			curW += len(gap) + ww
			hasText = true
			continue
		}
		if hasText {
			flush()
		} else {
			cur, curW = nil, 0 // an indent-only line is dropped
		}
		for ww > width {
			n, nw := 0, 0
			for n < len(word) && nw+word[n].w <= width {
				nw += word[n].w
				n++
			}
			if n == 0 {
				n, nw = 1, word[0].w
			}
			lines = append(lines, word[:n])
			word = word[n:]
			ww -= nw
		}
		if len(word) > 0 {
			cur = append([]mdCell(nil), word...)
			curW = ww
			hasText = true
		}
	}
	if hasText {
		flush()
	}
	return lines
}

// mdHardWrap cuts cells into chunks of at most width cells with no regard for
// spaces. An empty input gives one empty chunk. Used for code, where a line
// must keep its spacing.
func mdHardWrap(cells []mdCell, width int) [][]mdCell {
	if width < 1 {
		width = 1
	}
	var out [][]mdCell
	for len(cells) > 0 {
		n, nw := 0, 0
		for n < len(cells) && nw+cells[n].w <= width {
			nw += cells[n].w
			n++
		}
		if n == 0 {
			n = 1
		}
		out = append(out, cells[:n])
		cells = cells[n:]
	}
	if len(out) == 0 {
		out = [][]mdCell{nil}
	}
	return out
}

// styleFor returns the Style a role is drawn with.
func (s *Styles) styleFor(r mdRole) Style {
	switch r {
	case roleText:
		return s.Text
	case roleMuted:
		return s.Muted
	case roleAccent:
		return s.Accent
	case roleBorder:
		return s.Border
	case roleSynString:
		return s.SynString
	case roleSynKeyword:
		return s.SynKeyword
	case roleSynNumber:
		return s.SynNumber
	case roleSynType:
		return s.SynType
	case roleSuccess:
		return s.Success
	case roleError:
		return s.Error
	}
	return s.Text
}

// drawMarkdown turns laid-out rows into strings through sty. It is the only
// function in the markdown path that touches styling: it never looks at what
// sty is, so the live, flattened and identity passes differ only in the Styles
// they hand it.
//
// Within a run, the attributes are laid on in a fixed order: Strike on the bare
// text, then the role colour, then italic, then bold. Strike is a bare pair of
// SGR codes with no reset inside, so the colour, italic and bold that wrap it
// cover the whole run, and every group is one attribute or one colour.
func drawMarkdown(rows []mdRow, sty *Styles) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		var b strings.Builder
		for _, r := range row {
			if r.Text == "" {
				continue
			}
			s := r.Text
			if r.Strike {
				s = sty.Strike(s)
			}
			s = sty.styleFor(r.Role)(s)
			if r.Italic {
				s = sty.Italic(s)
			}
			if r.Bold {
				s = sty.Bold(s)
			}
			b.WriteString(s)
		}
		out[i] = b.String()
	}
	return out
}

// mdCache holds the laid-out rows of replies, so a reply is parsed and laid out
// once per consecutive run at the same width and glyph set, however many times
// relayout runs. The key includes the reply's text, so a changed reply is laid
// out again.
//
// Model is a value type and Update returns a modified copy, so the cache is
// held by pointer: every copy of the Model reaches the same entries, and a
// layout done while handling one message is still there for the next. Only the
// latest width is kept for each reply, in one entry per item ID.
type mdCache struct {
	entries map[ItemID]*mdEntry

	// layouts counts calls to mdLayoutWith made through rows. Tests read it to see
	// whether a relayout reused an entry.
	layouts int

	// layoutHook, when not nil, is called at the start of every layout made
	// through rows, inside the recover. It is nil in production; a test sets it
	// to count layouts or to make one panic.
	layoutHook func()
}

// mdEntry is one cached layout and what it was laid out from.
type mdEntry struct {
	lines   []string // the reply's lines, compared with the item's on every lookup
	width   int
	unicode bool
	rows    []mdRow
	failed  bool // the layout panicked; rows is nil
}

// newMdCache returns an empty cache.
func newMdCache() *mdCache { return &mdCache{entries: map[ItemID]*mdEntry{}} }

// rows returns the reply's laid-out rows at width w for glyphs g, laying them
// out and storing them when the entry for id is missing, was made for another
// width or glyph set, or was made from other text. The text check compares the
// lines one by one: strings that share their bytes compare in constant time, so
// an unchanged reply costs a pointer comparison per line, and a reply whose text
// changed, or a new session reusing an item ID, cannot be served a stale layout.
// A layout that failed is stored as a negative entry for the same id, width,
// glyph set and text, so the next lookup returns ok false without parsing the
// reply again.
func (c *mdCache) rows(id ItemID, lines []string, w int, g Glyphs) (rows []mdRow, ok bool) {
	if e := c.entries[id]; e != nil && e.width == w && e.unicode == g.Unicode && sameLines(e.lines, lines) {
		return e.rows, !e.failed
	}
	c.layouts++
	rows, ok = mdLayoutWith(strings.Join(lines, "\n"), w, g, c.layoutHook)
	c.entries[id] = &mdEntry{
		lines: append([]string(nil), lines...), width: w, unicode: g.Unicode, rows: rows, failed: !ok,
	}
	return rows, ok
}

// reset drops every entry. A new session reuses item IDs, and its replies would
// otherwise sit beside the old session's entries until each was replaced.
func (c *mdCache) reset() { c.entries = map[ItemID]*mdEntry{} }

// sameLines reports whether a and b hold the same strings in the same order.
func sameLines(a, b []string) bool {
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
