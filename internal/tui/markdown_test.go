package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// These tests cover the markdown rendering of assistant replies, ui-spec §3.2.
//
// Each test names its entry point. The element, styling and width tests call
// mdLayout, mdLayoutRaw and drawMarkdown directly: they pin what the layout and
// the drawing function produce for one piece of markdown, with no Model in the
// way. TestMarkdownCacheIsPerGlyphSetAndText calls mdCache.rows directly. The
// tests of the Model-level properties, cache reuse, the fallbacks, scrolling and
// copy go through a Model, by Update and relayout, because those behaviours are
// the Model's: renderAssistant picks the path, relayout draws it twice and keeps
// the scroll position.

var (
	mdIdentity = NewStyles(nil, false)
	// mdLiveSty emits real escape sequences. NewRenderer(true) is required: a
	// renderer built any other way resolves every style to ASCII and a "coloured"
	// result equals the plain one, which would let every strip(styled) == plain
	// check pass vacuously. See NewRenderer.
	mdLiveSty = NewStyles(NewRenderer(true), true)
)

// mdPlainRows lays src out at width and draws it with identity styles.
func mdPlainRows(t *testing.T, src string, width int, unicode bool) []string {
	t.Helper()
	rows, ok := mdLayout(src, width, NewGlyphs(unicode))
	if !ok {
		t.Fatalf("mdLayout failed on %q", src)
	}
	return drawMarkdown(rows, mdIdentity)
}

// mdLiveRows is mdPlainRows with the live palette.
func mdLiveRows(t *testing.T, src string, width int, unicode bool) []string {
	t.Helper()
	rows, ok := mdLayout(src, width, NewGlyphs(unicode))
	if !ok {
		t.Fatalf("mdLayout failed on %q", src)
	}
	return drawMarkdown(rows, mdLiveSty)
}

func equalRows(a, b []string) bool {
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

// TestMarkdownElementRows is the §3.2 element table, one row of the table per
// case, in Unicode and in ASCII. Entry point: mdLayout then drawMarkdown with
// identity styles, so each want is the exact plain row list. Markers kept and
// markers dropped are both pinned by the strings: a heading must keep "# ", bold
// must lose "**", strikethrough must keep "~~".
func TestMarkdownElementRows(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		width int
		uni   []string
		ascii []string // nil when it equals uni
	}{
		{
			name: "heading keeps its marker", src: "# Title", width: 40,
			uni: []string{"# Title"},
		},
		{
			name: "deeper heading keeps all its markers", src: "### Deep **b**", width: 40,
			uni: []string{"### Deep b"},
		},
		{
			name: "bold drops its markers", src: "a **b** c", width: 40,
			uni: []string{"a b c"},
		},
		{
			name: "italic drops its marker", src: "a *b* c", width: 40,
			uni: []string{"a b c"},
		},
		{
			name: "strikethrough keeps its markers", src: "a ~~b~~ c", width: 40,
			uni: []string{"a ~~b~~ c"},
		},
		{
			name: "a single-tilde pair is literal text", src: "a ~x~ b", width: 40,
			uni: []string{"a ~x~ b"},
		},
		{
			name: "a single-tilde pair after a word is literal text", src: "about ~5s~", width: 40,
			uni: []string{"about ~5s~"},
		},
		{
			name: "strikethrough nests an emphasis run", src: "~~**x**~~", width: 40,
			uni: []string{"~~x~~"},
		},
		{
			name: "emphasis nests a strikethrough run", src: "*~~x~~*", width: 40,
			uni: []string{"~~x~~"},
		},
		{
			name: "a double tilde closed by a single one is literal", src: "~~x~", width: 40,
			uni: []string{"~~x~"},
		},
		{
			name: "inline code keeps its backticks", src: "a `b` c", width: 40,
			uni: []string{"a `b` c"},
		},
		{
			name: "bulleted list with nesting by two", src: "- one\n- two\n  - nested\n- three", width: 40,
			uni:   []string{"• one", "• two", "  • nested", "• three"},
			ascii: []string{"- one", "- two", "  - nested", "- three"},
		},
		{
			name: "numbered list keeps its numbers", src: "3. a\n4. b", width: 40,
			uni: []string{"3. a", "4. b"},
		},
		{
			name: "numbers as written: all ones", src: "1. a\n1. b\n1. c", width: 40,
			uni: []string{"1. a", "1. b", "1. c"},
		},
		{
			name: "numbers as written: a gap", src: "1. a\n3. b", width: 40,
			uni: []string{"1. a", "3. b"},
		},
		{
			name: "numbers as written: leading zeros are kept", src: "01. a\n02. b", width: 40,
			uni: []string{"01. a", "02. b"},
		},
		{
			// Rule: an item whose first content is not directly after the marker (a
			// heading) has no readable number, so it takes the number after the item
			// before it. Here that is also the number written.
			name: "numbers: a heading first child takes the previous number plus one", src: "1. # h\n2. # i", width: 40,
			uni: []string{"1. # h", "2. # i"},
		},
		{
			// Same rule, with a written number that differs from previous+1: the
			// heading item shows 2, not the 5 written, and the number after it is 3.
			name: "numbers: a heading item ignores the number written", src: "1. # h\n5. # i", width: 40,
			uni: []string{"1. # h", "2. # i"},
		},
		{
			// Rule: an empty item has no readable number and takes previous+1, so the
			// 5 written on the second item is not shown; the third item has a first
			// line, reads its own 6 and keeps it.
			name: "numbers: empty items take previous plus one, a later written number is kept", src: "1.\n5.\n6. c", width: 40,
			uni: []string{"1.", "2.", "6. c"},
		},
		{
			// Rule: an item with a first line reads its own number, so a gap
			// between written numbers is kept.
			name: "numbers: a gap between items with text is kept", src: "1. a\n5. b\n6. c", width: 40,
			uni: []string{"1. a", "5. b", "6. c"},
		},
		{
			// Rule: an item whose first child is a nested list has no readable
			// number, so it takes the number after the item before it. This is the
			// first item, so that number is the list's start, 1. The nested list's
			// own first item keeps the number written on it.
			name: "numbers: a nested list as the first child", src: "1. 1. a", width: 40,
			uni: []string{"1. 1. a"},
		},
		{
			name: "numbers as written: paren marker", src: "2) a\n7) b", width: 40,
			uni: []string{"2) a", "7) b"},
		},
		{
			name: "numbers as written in a loose list", src: "4. a\n\n9. b", width: 40,
			uni: []string{"4. a", "", "9. b"},
		},
		{
			name: "an empty bullet is its marker alone", src: "- a\n-\n- b", width: 40,
			uni:   []string{"• a", "•", "• b"},
			ascii: []string{"- a", "-", "- b"},
		},
		{
			name: "an empty numbered item is its marker alone", src: "1. a\n2.\n3. c", width: 40,
			uni: []string{"1. a", "2.", "3. c"},
		},
		{
			name: "block quote", src: "> q1\n> q2", width: 40,
			uni:   []string{"│ q1", "│ q2"},
			ascii: []string{"| q1", "| q2"},
		},
		{
			name: "link shows text then url", src: "[t](https://e.x/p)", width: 40,
			uni: []string{"t (https://e.x/p)"},
		},
		{
			name: "bare url shown as written", src: "see https://e.x/p", width: 40,
			uni: []string{"see https://e.x/p"},
		},
		{
			name: "horizontal rule spans the width", src: "---", width: 10,
			uni:   []string{"──────────"},
			ascii: []string{"----------"},
		},
		{
			name: "fence indented two, fence lines and info string dropped, indentation kept",
			src:  "```go\nfunc f() {\n    x := 1\n}\n```", width: 40,
			uni: []string{"  func f() {", "      x := 1", "  }"},
		},
		{
			name: "fence wrapped rows keep the block indent",
			src:  "```\nabcdefghijklmnopqrstuvwxyz\n```", width: 12,
			uni: []string{"  abcdefghij", "  klmnopqrst", "  uvwxyz"},
		},
		{
			name: "table aligned with separators",
			src:  "| a | b |\n|---|---|\n| 1 | 22 |", width: 40,
			uni:   []string{"a │ b", "──────", "1 │ 22"},
			ascii: []string{"a | b", "------", "1 | 22"},
		},
		{
			name: "raw html block is literal", src: "<div>x</div>", width: 40,
			uni: []string{"<div>x</div>"},
		},
		{
			name: "raw inline html is literal", src: "a <b>x</b> c", width: 40,
			uni: []string{"a <b>x</b> c"},
		},
		{
			name: "entities inside a raw inline html tag are not resolved", src: `x <a title="&amp;\*">y</a> z`, width: 40,
			uni: []string{`x <a title="&amp;\*">y</a> z`},
		},
		{
			name: "entities in the text between raw inline html tags are resolved", src: "<b>&amp;</b>", width: 40,
			uni: []string{"<b>&</b>"},
		},
		{
			name: "image shows its alt text", src: "![alt text](u.png)", width: 40,
			uni: []string{"alt text"},
		},
		{
			name: "blocks are separated by one blank row", src: "# H\n\npara", width: 40,
			uni: []string{"# H", "", "para"},
		},
		{
			name: "runs of blank lines collapse", src: "a\n\n\n\nb", width: 40,
			uni: []string{"a", "", "b"},
		},
		{
			name: "a newline in a paragraph ends the row", src: "one\ntwo", width: 40,
			uni: []string{"one", "two"},
		},
		{
			name: "words wrap at the width", src: "aaa bbb ccc", width: 7,
			uni: []string{"aaa bbb", "ccc"},
		},
		{
			name: "a long word is hard-wrapped", src: strings.Repeat("x", 26), width: 10,
			uni: []string{"xxxxxxxxxx", "xxxxxxxxxx", "xxxxxx"},
		},
		{
			name: "double-width runes wrap by cells", src: "你好世界你好", width: 7,
			uni: []string{"你好世", "界你好"},
		},
		{
			name: "backslash escape and entity are resolved", src: `a \* b &amp; c`, width: 40,
			uni: []string{"a * b & c"},
		},
		{
			name: "a typographic character is folded in ASCII", src: "a — b", width: 40,
			uni:   []string{"a — b"},
			ascii: []string{"a - b"},
		},
		{
			name: "a character an entity produces is folded in ASCII", src: "a &mdash; b", width: 40,
			uni:   []string{"a — b"},
			ascii: []string{"a - b"},
		},
		{
			name: "an entity-produced character is folded in a heading", src: "# a &mdash; b", width: 40,
			uni:   []string{"# a — b"},
			ascii: []string{"# a - b"},
		},
		{
			// An entity for a control character becomes "·" in replaceControls, and
			// the ASCII glyph set folds that to ".".
			name: "a control entity becomes a dot, a full stop in ASCII", src: "a &#1; b", width: 40,
			uni:   []string{"a · b"},
			ascii: []string{"a . b"},
		},
		{
			name: "folded in inline code", src: "`a — b`", width: 40,
			uni:   []string{"`a — b`"},
			ascii: []string{"`a - b`"},
		},
		{
			name: "folded in a fenced block", src: "```\na — b\n```", width: 40,
			uni:   []string{"  a — b"},
			ascii: []string{"  a - b"},
		},
		{
			name: "folded in a raw html block", src: "<div>a — b</div>", width: 40,
			uni:   []string{"<div>a — b</div>"},
			ascii: []string{"<div>a - b</div>"},
		},
		{
			name: "folded in a table cell", src: "| — | b |\n|---|---|\n| 1 | 2 |", width: 40,
			uni:   []string{"— │ b", "─────", "1 │ 2"},
			ascii: []string{"- | b", "-----", "1 | 2"},
		},
	}
	for _, c := range cases {
		for _, uni := range []bool{true, false} {
			want := c.uni
			if !uni && c.ascii != nil {
				want = c.ascii
			}
			name := fmt.Sprintf("%s/unicode=%v", c.name, uni)
			t.Run(name, func(t *testing.T) {
				got := mdPlainRows(t, c.src, c.width, uni)
				if !equalRows(got, want) {
					t.Errorf("rows for %q at %d:\n got  %q\n want %q", c.src, c.width, got, want)
				}
			})
		}
	}
}

// mdTestNode is a block kind the layout has no case for. It stands in for "any
// other element" in the §3.2 catch-all, which no goldmark node reachable with
// the two enabled extensions produces.
type mdTestNode struct{ ast.BaseBlock }

var kindMDTest = ast.NewNodeKind("MDTest")

func (*mdTestNode) Kind() ast.NodeKind { return kindMDTest }

func (n *mdTestNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

// TestMarkdownOtherElementIsItsText covers the catch-all row of §3.2: a block the
// table does not name is drawn as its source text in the text role, markers and
// all. Entry point: mdLayouter.block called directly on a hand-built node, since
// the parser never produces one.
func TestMarkdownOtherElementIsItsText(t *testing.T) {
	const src = "raw **markers** [kept]"
	n := &mdTestNode{}
	n.SetLines(text.NewSegments())
	n.Lines().Append(text.NewSegment(0, len(src)))
	l := &mdLayouter{src: []byte(src), g: NewGlyphs(true)}
	rows := l.block(n, 40)

	if got := drawMarkdown(rows, mdIdentity); !equalRows(got, []string{src}) {
		t.Errorf("plain rows = %q, want [%q]", got, src)
	}
	want := mdLiveSty.Text(src)
	if got := drawMarkdown(rows, mdLiveSty); !equalRows(got, []string{want}) {
		t.Errorf("live rows = %q, want text role %q", got, want)
	}
}

// TestMarkdownElementStyling pins the live form of each styled element: which
// role and attributes each fragment is drawn with. Entry point: mdLayout then
// drawMarkdown with the live palette; each want is composed from the same Styles
// so a role swapped for another fails. The plain fragments in each row are
// asserted too, so a style leaking onto neighbouring text fails.
func TestMarkdownElementStyling(t *testing.T) {
	s := mdLiveSty
	cases := []struct {
		name  string
		src   string
		width int
		want  string
	}{
		{
			"heading is bold accent", "# Title", 40,
			s.Bold(s.Accent("# Title")),
		},
		{
			"bold", "a **b** c", 40,
			s.Text("a ") + s.Bold(s.Text("b")) + s.Text(" c"),
		},
		{
			"italic", "a *b* c", 40,
			s.Text("a ") + s.Italic(s.Text("b")) + s.Text(" c"),
		},
		{
			"bold italic nest", "***b***", 40,
			s.Bold(s.Italic(s.Text("b"))),
		},
		{
			"strikethrough: markers plain, text struck", "a ~~b~~ c", 40,
			s.Text("a ~~") + s.Text(s.Strike("b")) + s.Text("~~ c"),
		},
		{
			"inline code is synString, backticks included", "a `b` c", 40,
			s.Text("a ") + s.SynString("`b`") + s.Text(" c"),
		},
		{
			"quote prefix is muted", "> q", 40,
			s.Muted("│ ") + s.Text("q"),
		},
		{
			"link url is muted, parentheses included", "[t](u)", 40,
			s.Text("t") + s.Muted(" (u)"),
		},
		{
			"rule is border", "---", 6,
			s.Border("──────"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mdLiveRows(t, c.src, c.width, true)
			if len(got) != 1 || got[0] != c.want {
				t.Errorf("live row for %q:\n got  %q\n want %q", c.src, got, c.want)
			}
		})
	}
}

// TestMarkdownTableStyling checks the table's separators and header rule are
// drawn in border and the cells in text.
func TestMarkdownTableStyling(t *testing.T) {
	s := mdLiveSty
	got := mdLiveRows(t, "| a | b |\n|---|---|\n| 1 | 22 |", 40, true)
	want := []string{
		s.Text("a") + s.Text(" ") + s.Border("│") + s.Text(" ") + s.Text("b"),
		s.Border("──────"),
		s.Text("1") + s.Text(" ") + s.Border("│") + s.Text(" ") + s.Text("22"),
	}
	if !equalRows(got, want) {
		t.Errorf("live table rows:\n got  %q\n want %q", got, want)
	}
}

// mdSGRParams returns the SGR parameter strings of the escapes that form the
// whole of s's start (lead true) or end (lead false).
func mdSGRParams(s string, lead bool) []string {
	re := regexp.MustCompile(`^(?:\x1b\[[0-9;]*m)+`)
	if !lead {
		re = regexp.MustCompile(`(?:\x1b\[[0-9;]*m)+$`)
	}
	var out []string
	for _, m := range sgrRe.FindAllStringSubmatch(re.FindString(s), -1) {
		out = append(out, m[1])
	}
	return out
}

// TestMarkdownStrikeKeepsTheWholeRunStyled: a struck word of several runes is
// drawn as one run inside the colour and bold of its context, with no reset
// between its runes. Entry point: mdLayout then drawMarkdown with the live
// palette, for a paragraph, a heading and bold text. For each, the word must
// appear unbroken in the row (an escape between its runes would split it), the
// escapes directly before it must open its role colour, its attributes and
// strike, and those directly after must close strike and then reset: that
// is the colour and bold covering the whole word. Positive controls: lipgloss's
// own strikethrough, the form the live Strike replaced, does not leave the word
// unbroken, so the check can fail; and the live Strike carries SGR 9 and 29.
func TestMarkdownStrikeKeepsTheWholeRunStyled(t *testing.T) {
	const word = "struck"
	own := NewRenderer(true).NewStyle().Strikethrough(true).Render(word)
	if strings.Contains(own, word) {
		t.Fatalf("checker control failed: lipgloss's strikethrough %q leaves the word unbroken", own)
	}
	if got := mdLiveSty.Strike("x"); got != "\x1b[9mx\x1b[29m" {
		t.Fatalf("live Strike(x) = %q, want SGR 9 and 29 around the text", got)
	}

	cases := []struct {
		name, src string
		want      []string // parameters that must open the word's run
	}{
		{"paragraph", "a ~~" + word + "~~ b", []string{"38;5;253", "9"}},
		{"heading", "# ~~" + word + "~~ head", []string{"1", "38;5;117", "9"}},
		{"bold", "**~~" + word + "~~**", []string{"1", "38;5;253", "9"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rows := mdLiveRows(t, c.src, 40, true)
			if len(rows) != 1 {
				t.Fatalf("rows = %q, want one", rows)
			}
			row := rows[0]
			i := strings.Index(row, word)
			if i < 0 {
				t.Fatalf("the struck word is not unbroken in %q", row)
			}
			open := mdSGRParams(row[:i], false)
			for _, w := range c.want {
				found := false
				for _, p := range open {
					found = found || p == w
				}
				if !found {
					t.Errorf("escapes before the word are %q, want %q among them: %q", open, w, row)
				}
			}
			// After the word comes 29 and then the resets that end its enclosing
			// colour and attributes; whatever follows opens the next run.
			closed := mdSGRParams(row[i+len(word):], true)
			if len(closed) < 2 || closed[0] != "29" || closed[1] != "0" {
				t.Errorf("escapes after the word are %q, want 29 then a reset", closed)
			}
		})
	}
}

// mdHasStrike reports whether the tree md parses src into holds a
// strikethrough node.
func mdHasStrike(md goldmark.Markdown, src string) bool {
	found := false
	doc := md.Parser().Parse(text.NewReader([]byte(src)))
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if _, ok := n.(*east.Strikethrough); ok {
			found = true
		}
		return ast.WalkContinue, nil
	})
	return found
}

// TestMarkdownSingleTildeIsNotStrikethrough: only a double tilde strikes.
// Entry point: mdParser's tree for each input, and mdLayout then drawMarkdown
// with the live palette for the rows. Control: every single-tilde input is
// parsed twice, once by goldmark's own extension.Strikethrough, whose tree must
// hold a strikethrough node, and once by mdParser, whose tree must not. An input
// the library does not strike would prove nothing about mdStrikethrough, so it
// fails the control. The drawn rows are the input as written, with no strike
// escape; the double-tilde form is the second control, struck.
func TestMarkdownSingleTildeIsNotStrikethrough(t *testing.T) {
	library := goldmark.New(goldmark.WithExtensions(extension.Strikethrough))
	for _, src := range []string{"a ~x~ b", "about ~5s~", "~x~", "~~x~", "~x~~"} {
		t.Run(src, func(t *testing.T) {
			if !mdHasStrike(library, src) {
				t.Fatalf("control failed: goldmark's own strikethrough does not strike %q", src)
			}
			if mdHasStrike(mdParser, src) {
				t.Errorf("mdParser strikes %q, which is not a double-tilde pair", src)
			}
			rows := mdLiveRows(t, src, 40, true)
			if want := []string{mdLiveSty.Text(src)}; !equalRows(rows, want) {
				t.Errorf("%q drawn as %q, want the plain text %q", src, rows, want)
			}
			if strings.Contains(strings.Join(rows, ""), "\x1b[9m") {
				t.Errorf("%q produced a strike escape", src)
			}
		})
	}
	double := mdLiveRows(t, "about ~~5s~~", 40, true)
	if !strings.Contains(strings.Join(double, ""), "\x1b[9m5s\x1b[29m") {
		t.Errorf("control failed: ~~5s~~ is not struck: %q", double)
	}
	if !mdHasStrike(mdParser, "about ~~5s~~") {
		t.Error("control failed: mdParser does not strike a double-tilde pair")
	}
}

// TestMarkdownStrikeNestsWithEmphasis: strikethrough and bold or italic nest in
// either order. Entry point: mdLayout, read as runs. For "~~**x**~~" and
// "*~~x~~*" the run that is exactly "x" carries Strike and the emphasis; any
// tilde marker run carries no Strike, so the strike covers the word only.
// Control: "~~x~~" alone gives x Strike and no emphasis.
func TestMarkdownStrikeNestsWithEmphasis(t *testing.T) {
	word := func(t *testing.T, src string) mdRun {
		t.Helper()
		rows, ok := mdLayout(src, 40, NewGlyphs(true))
		if !ok || len(rows) != 1 {
			t.Fatalf("mdLayout(%q) = %d rows, ok %v", src, len(rows), ok)
		}
		var got *mdRun
		for i, r := range rows[0] {
			if r.Text == "~~" && r.Strike {
				t.Errorf("%q: the marker run %d is struck", src, i)
			}
			if r.Text == "x" {
				got = &rows[0][i]
			}
		}
		if got == nil {
			t.Fatalf("%q: no run is exactly x: %+v", src, rows[0])
		}
		return *got
	}
	if w := word(t, "~~x~~"); !w.Strike || w.Bold || w.Italic {
		t.Fatalf("control failed: x in ~~x~~ is %+v", w)
	}
	if w := word(t, "~~**x**~~"); !w.Strike || !w.Bold {
		t.Errorf("~~**x**~~: x is %+v, want Strike and Bold", w)
	}
	if w := word(t, "*~~x~~*"); !w.Strike || !w.Italic {
		t.Errorf("*~~x~~*: x is %+v, want Strike and Italic", w)
	}
}

// mdSamples are replies that between them use every element the renderer draws.
var mdSamples = map[string]string{
	"everything": "# Head\n\nSome **bold** and *it* and `code` and ~~x~~ with a [link](https://e.x/p).\n\n" +
		"- item one\n- item two\n  - nested\n\n1. first\n2. second\n\n> quoted\n> lines\n\n---\n\n" +
		"```go\nfunc f() {\n    return 1\n}\n```\n\n| a | b |\n|---|---|\n| 1 | 22 |\n\n<div>html</div>\n\n![alt](i.png)",
	"plain":    "just a sentence with nothing to render",
	"entities": "a &#27;[31mred&#27;[0m and &amp; and \\*stars\\*",
}

// TestMarkdownStyledStripsToPlain is §13 property 1 for a rendered reply,
// through the drawing function: the same rows drawn with the live palette and
// with identity styles must agree once the escapes are stripped. Positive
// control: the live rows carry 0x1b, so the comparison is between a styled and
// an unstyled string and not between two plain ones.
func TestMarkdownStyledStripsToPlain(t *testing.T) {
	for name, src := range mdSamples {
		for _, uni := range []bool{true, false} {
			for _, w := range []int{20, 40, 80} {
				t.Run(fmt.Sprintf("%s/unicode=%v/w=%d", name, uni, w), func(t *testing.T) {
					live := mdLiveRows(t, src, w, uni)
					plain := mdPlainRows(t, src, w, uni)
					if name != "plain" && !strings.Contains(strings.Join(live, "\n"), "\x1b") {
						t.Fatal("positive control failed: live rows contain no escape byte")
					}
					stripped := make([]string, len(live))
					for i, r := range live {
						stripped[i] = sgrRe.ReplaceAllString(r, "")
					}
					if !equalRows(stripped, plain) {
						t.Errorf("strip(live) != plain\n live-stripped %q\n plain         %q", stripped, plain)
					}
				})
			}
		}
	}
}

// TestMarkdownPlainPathHasNoEscapes is §13 property 2 for a rendered reply: no
// 0x1b in the identity pass. Entry point: mdLayout and drawMarkdown with identity
// styles. The "entities" sample asks for an escape byte through &#27;, which the
// text path must turn into a visible mark; the live rows of the same reply are the
// positive control that 0x1b does appear when Kirsch styles it.
func TestMarkdownPlainPathHasNoEscapes(t *testing.T) {
	for name, src := range mdSamples {
		plain := strings.Join(mdPlainRows(t, src, 60, true), "\n")
		if strings.Contains(plain, "\x1b") {
			t.Errorf("%s: plain rows contain an escape byte: %q", name, plain)
		}
	}
	if !strings.Contains(strings.Join(mdLiveRows(t, mdSamples["everything"], 60, true), "\n"), "\x1b") {
		t.Fatal("positive control failed: live rows of a styled reply contain no escape")
	}
	live := strings.Join(mdLiveRows(t, mdSamples["entities"], 60, true), "\n")
	// The only escapes in the live rows are Kirsch's own, each a complete SGR.
	if got := sgrRe.ReplaceAllString(live, ""); strings.Contains(got, "\x1b") {
		t.Errorf("an entity produced a non-SGR escape: %q", got)
	}
}

// newMDModel builds a Model at w x h with markdown rendering on or off.
func newMDModel(t *testing.T, colour, unicode, renderMD bool, w, h int) Model {
	t.Helper()
	m := New(Options{
		Version:        fixtureVersion,
		Caps:           Caps{Colour: colour, Unicode: unicode},
		RenderMarkdown: renderMD,
	})
	return drive(m, tea.WindowSizeMsg{Width: w, Height: h})
}

// addReply appends a finished reply, as a resumed one is built: lines already
// sanitised, Source set, not streaming. It relayouts and returns the item ID.
func addReply(m *Model, src string) ItemID {
	id := m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: SanitizeLines(src), Source: src}})
	m.relayout(m.layout())
	return id
}

// itemRows returns the styled and plain rows an item occupies in the model's
// flattened transcript.
func itemRows(t *testing.T, m Model, id ItemID) (live, plain []string) {
	t.Helper()
	r, ok := m.rowFor(id)
	if !ok {
		t.Fatalf("item %d has no rows", id)
	}
	return m.lines[r.Start : r.Start+r.N], m.plainLines[r.Start : r.Start+r.N]
}

// TestRenderedReplyStyledStripsToPlainInModel is property 1 through relayout:
// the coloured Model's styled rows, stripped, equal its plain rows, and equal
// the rows of a Model with colour off. Entry point: Update (WindowSizeMsg) and
// relayout on three Models of the same reply. Positive control: the styled rows
// contain 0x1b and the colour-off Model's rows do not.
func TestRenderedReplyStyledStripsToPlainInModel(t *testing.T) {
	for _, uni := range []bool{true, false} {
		on := newMDModel(t, true, uni, true, 80, 40)
		off := newMDModel(t, false, uni, true, 80, 40)
		idOn := addReply(&on, mdSamples["everything"])
		idOff := addReply(&off, mdSamples["everything"])
		live, plain := itemRows(t, on, idOn)
		offLive, offPlain := itemRows(t, off, idOff)

		if !strings.Contains(strings.Join(live, "\n"), "\x1b") {
			t.Fatalf("unicode=%v: positive control failed: styled rows contain no escape", uni)
		}
		stripped := make([]string, len(live))
		for i, r := range live {
			stripped[i] = sgrRe.ReplaceAllString(r, "")
		}
		if !equalRows(stripped, plain) {
			t.Errorf("unicode=%v: strip(styled) != plain\n%q\n%q", uni, stripped, plain)
		}
		if strings.Contains(strings.Join(offLive, "\n"), "\x1b") {
			t.Errorf("unicode=%v: colour-off rows contain an escape", uni)
		}
		if !equalRows(offLive, offPlain) || !equalRows(offPlain, plain) {
			t.Errorf("unicode=%v: colour off is not the same structure, unstyled\n off %q\n on  %q", uni, offPlain, plain)
		}
	}
}

// TestRenderedReplySGRIsInThePalette is property 3 for a rendered reply. Entry
// point: Update and relayout on a coloured Model; the rows are read from
// m.lines. Every escape must be a whole SGR whose parameters are in PaletteSGR,
// no OSC 8 may appear, and no background parameter may appear. Positive control:
// the rows must contain bold, italic, strikethrough and each foreground colour
// the sample uses, so an allow-list check cannot pass on rows that carry none.
func TestRenderedReplySGRIsInThePalette(t *testing.T) {
	m := newMDModel(t, true, true, true, 80, 60)
	id := addReply(&m, mdSamples["everything"])
	live, _ := itemRows(t, m, id)
	out := strings.Join(live, "\n")

	seen := map[string]bool{}
	for i := 0; i < len(out); i++ {
		if out[i] != 0x1b {
			continue
		}
		loc := sgrRe.FindStringIndex(out[i:])
		if loc == nil || loc[0] != 0 {
			t.Fatalf("non-SGR escape at byte %d: %q", i, out[i:minInt(i+24, len(out))])
		}
		params := sgrRe.FindStringSubmatch(out[i:])[1]
		seen[params] = true
		if !PaletteSGR[params] {
			t.Errorf("SGR %q is not in PaletteSGR", params)
		}
		if strings.HasPrefix(params, "48;") || strings.Contains(params, ";48;") {
			t.Errorf("background SGR %q in a rendered reply", params)
		}
		i += loc[1] - 1
	}
	if strings.Contains(out, "\x1b]8") {
		t.Error("OSC 8 hyperlink in a rendered reply")
	}
	for _, want := range []string{"1", "3", "9", "0", "38;5;117", "38;5;180", "38;5;248", "38;5;244", "38;5;253"} {
		if !seen[want] {
			t.Errorf("positive control: SGR %q never appeared, so the allow-list was not exercised on it (saw %v)", want, seen)
		}
	}
}

// mdWidthSamples are replies built to push a row past the width: a long word, a
// long URL, a wide table, a deep list and others.
func mdWidthSamples() map[string]string {
	deep := ""
	for i := 0; i < 6; i++ {
		deep += strings.Repeat("  ", i) + "- level " + fmt.Sprint(i) + " with some words to wrap around\n"
	}
	cols := make([]string, 8)
	rule := make([]string, 8)
	cells := make([]string, 8)
	for i := range cols {
		cols[i] = fmt.Sprintf("column%d", i)
		rule[i] = "---"
		cells[i] = strings.Repeat("c", 12+i)
	}
	return map[string]string{
		"long word": "start " + strings.Repeat("x", 300) + " end",
		"long url": "see [link](https://example.com/" + strings.Repeat("a/b", 80) +
			") and https://example.com/" + strings.Repeat("q", 150),
		"wide table": "| " + strings.Join(cols, " | ") + " |\n|" + strings.Join(rule, "|") + "|\n| " +
			strings.Join(cells, " | ") + " |\n",
		"deep list":  deep,
		"nested":     "> > > quote text " + strings.Repeat("word ", 40),
		"cjk":        strings.Repeat("你好世界 ", 40) + strings.Repeat("🙂", 30),
		"fence":      "```\n" + strings.Repeat("0123456789", 30) + "\n    " + strings.Repeat("y", 100) + "\n```",
		"heading":    "## " + strings.Repeat("title ", 30),
		"everything": mdSamples["everything"],
	}
}

// TestMarkdownRowsNeverExceedTheWidth is the §3.2 post-condition at every width
// from 20 to 120, for each sample. Entry point: mdLayoutRaw, the layout before
// the backstop clamp, so the layout itself is held to the width and the clamp
// cannot hide a wrapping fault. The "long word" and "long url" samples carry a
// positive control: some row reaches the width exactly, so the check is not
// satisfied by rows that are all short. The checker itself is shown to report a
// too-wide row before the sweep runs.
func TestMarkdownRowsNeverExceedTheWidth(t *testing.T) {
	exceeds := func(rows []string, w int) bool {
		for _, r := range rows {
			if cellWidth(r) > w {
				return true
			}
		}
		return false
	}
	if !exceeds([]string{"abcdef"}, 5) {
		t.Fatal("checker control failed: a 6-cell row was not reported at width 5")
	}
	for name, src := range mdWidthSamples() {
		for _, uni := range []bool{true, false} {
			for w := 20; w <= 120; w++ {
				rows, ok := mdLayoutRaw(src, w, NewGlyphs(uni))
				if !ok {
					t.Fatalf("%s w=%d: layout failed", name, w)
				}
				plain := drawMarkdown(rows, mdIdentity)
				if exceeds(plain, w) {
					t.Errorf("%s unicode=%v w=%d: a row is wider than the width: %q", name, uni, w, plain)
					break
				}
				if name == "long word" || name == "long url" {
					reached := false
					for _, r := range plain {
						if cellWidth(r) == w {
							reached = true
						}
					}
					if !reached {
						t.Errorf("%s w=%d: no row reaches the width, so the check measured nothing: %q", name, w, plain)
					}
				}
			}
		}
	}
}

// TestRenderedReplyKeepsItsTextInModel: no wrapping, narrowing or clamp loses
// text. Entry point: Update (WindowSizeMsg) and relayout; the rows are the plain
// rows the Model holds for the item. For every sample of mdWidthSamples, each
// listed character must occur in the rows exactly as often as the author put it
// in the text shown. For all but "everything" that is the count in the source,
// those samples having no markup that holds the character and is not drawn;
// "everything" drops its info string, image destination and fence lines, so its
// counts are written out: q only in "quoted", u in "quoted", "func" and "return".
// Positive control: the same count over rows with the last row removed differs
// for "long word", so the check can see a lost row.
func TestRenderedReplyKeepsItsTextInModel(t *testing.T) {
	type probe struct {
		char string
		want int // -1: the count in the source
	}
	probes := map[string][]probe{
		"long word":  {{"x", -1}},
		"long url":   {{"q", -1}, {"b", -1}},
		"wide table": {{"c", -1}},
		"deep list":  {{"w", -1}},
		"nested":     {{"w", -1}},
		"cjk":        {{"世", -1}, {"🙂", -1}},
		"fence":      {{"0", -1}, {"y", -1}},
		"heading":    {{"l", -1}},
		"everything": {{"q", 1}, {"u", 3}},
	}
	count := func(rows []string, ch string) int { return strings.Count(strings.Join(rows, "\n"), ch) }

	for name, src := range mdWidthSamples() {
		ps, ok := probes[name]
		if !ok {
			t.Errorf("sample %q has no probe, so its text is not checked", name)
			continue
		}
		for _, w := range []int{40, 80} {
			m := newMDModel(t, false, true, true, w, 30)
			id := addReply(&m, src)
			_, plain := itemRows(t, m, id)
			for _, p := range ps {
				want := p.want
				if want < 0 {
					want = strings.Count(src, p.char)
				}
				if want == 0 {
					t.Fatalf("%s: probe %q does not occur in the source, so it checks nothing", name, p.char)
				}
				if got := count(plain, p.char); got != want {
					t.Errorf("%s w=%d: %q occurs %d times in the rows, want %d: %q", name, w, p.char, got, want, plain)
				}
				if name == "long word" && count(plain[:len(plain)-1], p.char) == want {
					t.Errorf("control failed: dropping the last row of %q did not change the count", name)
				}
			}
		}
	}
}

// TestMarkdownExtremeNestingStaysWithinWidth: nesting deeper than the width can
// hold leaves blocks too little room, and the backstop clamp must still hold the
// post-condition. Entry point: mdLayout (with the clamp). The raw layout of the
// same reply is wider at width 20, which shows the clamp is what held it.
func TestMarkdownExtremeNestingStaysWithinWidth(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 14; i++ {
		b.WriteString(strings.Repeat("  ", i) + "- deep level text\n")
	}
	g := NewGlyphs(true)
	raw, _ := mdLayoutRaw(b.String(), 20, g)
	wide := false
	for _, r := range drawMarkdown(raw, mdIdentity) {
		if cellWidth(r) > 20 {
			wide = true
		}
	}
	if !wide {
		t.Fatal("positive control failed: the raw layout fits 20 cells, so the clamp was not needed")
	}
	rows, _ := mdLayout(b.String(), 20, g)
	for _, r := range drawMarkdown(rows, mdIdentity) {
		if cellWidth(r) > 20 {
			t.Errorf("row exceeds 20 cells after the clamp: %q", r)
		}
	}
}

// TestRenderedReplyFitsTheItemWidthInModel runs the width post-condition through
// the Model at terminal widths from 40 to 120, for an unselected reply and for a selected
// one, which is two columns narrower. Entry point: Update (WindowSizeMsg), then
// relayout. For the selected reply the gutter's two cells are removed and the
// rest must fit ContentW-2; the cache entry's width is checked to be the
// selected width, so the test cannot pass with the reply laid out at the wide
// one.
func TestRenderedReplyFitsTheItemWidthInModel(t *testing.T) {
	samples := mdWidthSamples()
	for name, src := range samples {
		// Below 40 terminal columns no transcript is drawn (computeLayout), so the
		// Model-level sweep starts there; mdLayoutRaw's own sweep covers 20 up.
		for w := 40; w <= 120; w += 7 {
			m := newMDModel(t, false, true, true, w, 30)
			id := addReply(&m, src)
			cw := m.layout().ContentW
			_, plain := itemRows(t, m, id)
			for _, r := range plain {
				if cellWidth(r) > cw {
					t.Fatalf("%s w=%d: row wider than ContentW %d: %q", name, w, cw, r)
				}
			}
			if got := m.md.entries[id].width; got != cw {
				t.Fatalf("%s w=%d: laid out at %d, want ContentW %d", name, w, got, cw)
			}

			m.sel = id
			m.relayout(m.layout())
			_, plain = itemRows(t, m, id)
			for _, r := range plain {
				if !strings.HasPrefix(r, "┃ ") {
					t.Fatalf("%s w=%d: selected row has no gutter: %q", name, w, r)
				}
				if cellWidth(strings.TrimPrefix(r, "┃ ")) > cw-2 {
					t.Fatalf("%s w=%d: selected row wider than ContentW-2 = %d: %q", name, w, cw-2, r)
				}
			}
			if got := m.md.entries[id].width; got != cw-2 {
				t.Fatalf("%s w=%d: selected reply laid out at %d, want %d", name, w, got, cw-2)
			}
		}
	}
}

// TestMarkdownCacheLaysOutOncePerWidth: a reply is parsed and laid out once for
// its width and glyph set however many times relayout runs, again after a width
// change, and only the latest width is kept. Entry point: Update and relayout;
// the count is mdCache.layouts, which counts calls to mdLayout made through the
// cache. The first assertion pins that a layout happened at all, so the later
// "unchanged" counts are about reuse and not about nothing running.
func TestMarkdownCacheLaysOutOncePerWidth(t *testing.T) {
	m := newMDModel(t, true, true, true, 80, 30)
	id := addReply(&m, mdSamples["everything"])
	other := addReply(&m, "second reply")
	if got := m.md.layouts; got != 2 {
		t.Fatalf("after two replies: %d layouts, want 2 (one per reply, the plain pass reusing the live pass's)", got)
	}

	for i := 0; i < 5; i++ {
		m.relayout(m.layout())
	}
	m = drive(m, UsageMsg{InputTokens: 1}, WorkspaceInfoMsg{Project: "p", Branch: "b"}, key('x'))
	if got := m.md.layouts; got != 2 {
		t.Errorf("repeated relayouts at one width: %d layouts, want 2", got)
	}

	// An overlay flattens the transcript: another Styles over the same rows.
	m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
	m.relayout(m.layout())
	if got := m.md.layouts; got != 2 {
		t.Errorf("flattened pass laid out again: %d layouts, want 2", got)
	}
	m.closeModal()

	m = drive(m, tea.WindowSizeMsg{Width: 60, Height: 30})
	if got := m.md.layouts; got != 4 {
		t.Errorf("after a width change: %d layouts, want 4 (both replies once more)", got)
	}
	if n := len(m.md.entries); n != 2 {
		t.Errorf("cache holds %d entries for 2 replies, want one each", n)
	}
	for _, rid := range []ItemID{id, other} {
		if got, want := m.md.entries[rid].width, m.layout().ContentW; got != want {
			t.Errorf("item %d cached at width %d, want the latest, %d", rid, got, want)
		}
	}

	// Selecting a reply narrows it by the gutter: one more layout for that reply.
	m.sel = id
	m.relayout(m.layout())
	if got := m.md.layouts; got != 5 {
		t.Errorf("after selecting a reply: %d layouts, want 5", got)
	}
}

// TestMarkdownCacheIsPerGlyphSetAndText: the same item ID is laid out again for
// the other glyph set and for different text, so a stale layout is never served.
// Entry point: mdCache.rows called directly, with the Glyphs and lines varied.
func TestMarkdownCacheIsPerGlyphSetAndText(t *testing.T) {
	c := newMdCache()
	lines := []string{"- a"}
	uni, asc := NewGlyphs(true), NewGlyphs(false)
	r1, _ := c.rows(1, lines, 40, uni)
	c.rows(1, lines, 40, uni)
	if c.layouts != 1 {
		t.Fatalf("same key: %d layouts, want 1", c.layouts)
	}
	r2, _ := c.rows(1, lines, 40, asc)
	if c.layouts != 2 {
		t.Errorf("other glyph set: %d layouts, want 2", c.layouts)
	}
	if drawMarkdown(r1, mdIdentity)[0] == drawMarkdown(r2, mdIdentity)[0] {
		t.Error("Unicode and ASCII layouts drew the same bullet")
	}
	c.rows(1, []string{"- b"}, 40, asc)
	if c.layouts != 3 {
		t.Errorf("other text under the same ID: %d layouts, want 3", c.layouts)
	}
}

// TestNewSessionClearsTheMarkdownCache: a new session starts with an empty
// cache. Entry point: NewSessionDoneMsg through Update, which reaches
// resetForNewSession. The positive control is the entry present before the
// message; afterwards a reply in the new session is laid out afresh and is the
// only entry.
func TestNewSessionClearsTheMarkdownCache(t *testing.T) {
	m := newMDModel(t, false, true, true, 80, 24)
	id := addReply(&m, "# old\ntext")
	if m.md.entries[id] == nil {
		t.Fatal("control failed: the reply has no cache entry before the new session")
	}
	m = drive(m, NewSessionDoneMsg{})
	if n := len(m.md.entries); n != 0 {
		t.Fatalf("%d cache entries survive the new session, want none", n)
	}
	before := m.md.layouts
	id2 := addReply(&m, "# new\ntext")
	if m.md.layouts != before+1 || len(m.md.entries) != 1 || m.md.entries[id2] == nil {
		t.Errorf("new session reply: layouts %d -> %d, %d entries", before, m.md.layouts, len(m.md.entries))
	}
}

// TestMarkdownLayoutPanicFallsBackToPlainRowsAndIsNotRetried: a reply whose
// layout panics is drawn as plain rows, the interface keeps working, and a
// later relayout at the same width does not parse it again. Entry point: Update
// and relayout; mdCache.layoutHook is the test hook that counts every layout and
// makes it panic. The plain rows are those of a Model with markdown off. Controls:
// the hook without a panic counts one layout and gives rendered rows, and a
// changed width parses the reply again.
func TestMarkdownLayoutPanicFallsBackToPlainRowsAndIsNotRetried(t *testing.T) {
	const src = "# T\n**b** and `c`\n- item"

	off := newMDModel(t, false, true, false, 80, 24)
	_, raw := itemRows(t, off, addReply(&off, src))

	parses, fail := 0, false
	hook := func() {
		parses++
		if fail {
			panic("forced layout failure")
		}
	}

	ok := newMDModel(t, false, true, true, 80, 24)
	ok.md.layoutHook = hook
	_, rendered := itemRows(t, ok, addReply(&ok, src))
	if parses != 1 || equalRows(rendered, raw) {
		t.Fatalf("control failed: parses = %d (want 1), rendered %q vs raw %q", parses, rendered, raw)
	}

	fail, parses = true, 0
	m := newMDModel(t, false, true, true, 80, 24)
	m.md.layoutHook = hook
	id := addReply(&m, src)
	_, got := itemRows(t, m, id)
	if !equalRows(got, raw) {
		t.Errorf("failed layout drew %q, want the plain rows %q", got, raw)
	}
	if parses != 1 {
		t.Errorf("the first relayout parsed the failing reply %d times, want 1", parses)
	}
	for i := 0; i < 3; i++ {
		m.relayout(m.layout())
	}
	m = drive(m, UsageMsg{InputTokens: 1}, key('x'))
	if parses != 1 {
		t.Errorf("later relayouts parsed the failing reply again: %d parses, want 1", parses)
	}
	if !strings.Contains(m.View(), "# T") {
		t.Error("the interface does not draw the plain fallback after the failure")
	}

	m = drive(m, tea.WindowSizeMsg{Width: 60, Height: 24})
	if parses != 2 {
		t.Errorf("after a width change: %d parses, want 2", parses)
	}
}

// TestRenderMarkdownOffGivesTheRawRows: with render_markdown false a finished
// reply shows exactly the raw text as v0.1.3 drew it. Entry point: Update and
// relayout. The want rows are literal, so the test does not depend on the code
// that produces them. The control is the same reply on a Model with the setting
// on, whose rows differ.
func TestRenderMarkdownOffGivesTheRawRows(t *testing.T) {
	const src = "# T\n**b** and `c`\n- item"
	off := newMDModel(t, false, true, false, 80, 24)
	idOff := addReply(&off, src)
	_, got := itemRows(t, off, idOff)
	want := []string{"# T", "**b** and `c`", "- item"}
	if !equalRows(got, want) {
		t.Errorf("render_markdown=false rows = %q, want %q", got, want)
	}
	if off.md.layouts != 0 {
		t.Errorf("render_markdown=false still laid out %d times", off.md.layouts)
	}

	on := newMDModel(t, false, true, true, 80, 24)
	idOn := addReply(&on, src)
	_, rendered := itemRows(t, on, idOn)
	if equalRows(rendered, want) {
		t.Errorf("control failed: render_markdown=true drew the raw rows %q", rendered)
	}
}

// TestStreamingReplyStaysPlain: a streaming reply is drawn plain with its caret
// even with markdown on, and the same text finished is drawn rendered. Entry
// point: AssistantTextDeltaMsg and TurnCompleteMsg through Update. The finished
// form is the positive control for the streaming one.
func TestStreamingReplyStaysPlain(t *testing.T) {
	m := newMDModel(t, false, true, true, 80, 24)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "# Head\n**b** text"})
	id := m.tr.Items()[0].ID
	_, streaming := itemRows(t, m, id)
	wantStreaming := []string{"# Head", "**b** text ▌"}
	if !equalRows(streaming, wantStreaming) {
		t.Errorf("streaming rows = %q, want %q", streaming, wantStreaming)
	}
	if m.md.layouts != 0 {
		t.Errorf("a streaming reply was laid out %d times", m.md.layouts)
	}

	m = drive(m, TurnCompleteMsg{})
	_, done := itemRows(t, m, id)
	wantDone := []string{"# Head", "", "b text"}
	if !equalRows(done, wantDone) {
		t.Errorf("finished rows = %q, want %q", done, wantDone)
	}
}

// replyWithFiller returns a Model whose transcript is tall enough to scroll,
// ending in a streaming reply that has had text. Entry point: Update.
func replyWithFiller(t *testing.T, delta string) Model {
	t.Helper()
	m := newMDModel(t, false, true, true, 80, 12)
	for i := 0; i < 10; i++ {
		m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{fmt.Sprintf("question %d", i)}}})
	}
	return drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: delta})
}

// TestReflowOnCompletionKeepsAPinnedViewPinned and its twin pin the scroll rules
// for the one reflow, plain to rendered. The reply is built so that rendering
// changes its row count; each test asserts that change happened, so a reflow
// that did nothing cannot satisfy them.
func TestReflowOnCompletionKeepsAPinnedViewPinned(t *testing.T) {
	m := replyWithFiller(t, "# Head\nBody text")
	if !m.scroll.Pinned {
		t.Fatal("setup: view is not pinned")
	}
	before := len(m.lines)
	m = drive(m, TurnCompleteMsg{})
	if len(m.lines) == before {
		t.Fatalf("positive control failed: completion did not change the row count (%d)", before)
	}
	lay := m.layout()
	if !m.scroll.Pinned {
		t.Error("pinned view came unpinned on the reflow")
	}
	if want := maxInt(0, len(m.lines)-lay.TranscriptH); m.scroll.Offset != want {
		t.Errorf("pinned offset = %d, want the bottom, %d", m.scroll.Offset, want)
	}
}

func TestReflowOnCompletionKeepsAnUnpinnedOffset(t *testing.T) {
	m := replyWithFiller(t, "# Head\nBody text")
	m.scroll = Scroll{Offset: 2, Pinned: false}
	before := len(m.lines)
	m = drive(m, TurnCompleteMsg{})
	if len(m.lines) == before {
		t.Fatalf("positive control failed: completion did not change the row count (%d)", before)
	}
	if m.scroll.Pinned {
		t.Error("an unpinned view was pinned by the reflow")
	}
	if m.scroll.Offset != 2 {
		t.Errorf("unpinned offset = %d, want 2", m.scroll.Offset)
	}
}

// TestCancelledAndErroredRepliesRender: a reply ended by a cancel or by a turn
// error is drawn as markdown, and the cancelled marker sits on its own row after
// it. Entry point: AssistantTextDeltaMsg then TurnCancelledMsg or TurnErrorMsg
// through Update. A resumed reply is covered by addReply, which builds the item
// as the resume path does.
func TestCancelledAndErroredRepliesRender(t *testing.T) {
	const delta = "# Head\nBody **bold**"
	want := []string{"# Head", "", "Body bold"}

	m := newMDModel(t, false, true, true, 80, 24)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: delta})
	id := m.tr.Items()[0].ID
	m = drive(m, TurnCancelledMsg{})
	_, got := itemRows(t, m, id)
	if !equalRows(got, append(append([]string{}, want...), "⊘ cancelled")) {
		t.Errorf("cancelled reply rows = %q", got)
	}

	m = newMDModel(t, false, true, true, 80, 24)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: delta})
	id = m.tr.Items()[0].ID
	m = drive(m, TurnErrorMsg{Kind: "provider_error", Message: "boom"})
	_, got = itemRows(t, m, id)
	if !equalRows(got, want) {
		t.Errorf("errored reply rows = %q, want %q", got, want)
	}

	m = newMDModel(t, false, true, true, 80, 24)
	id = addReply(&m, delta)
	_, got = itemRows(t, m, id)
	if !equalRows(got, want) {
		t.Errorf("resumed reply rows = %q, want %q", got, want)
	}
}

// TestFlatRenderedReplyIsDimOnly: with an overlay open the transcript is drawn
// through Flat(), and a rendered reply must then use no colour but dim. Entry
// point: openModal then relayout on a coloured Model. The control is the same
// reply before the overlay opened, whose rows carry other colours and bold.
func TestFlatRenderedReplyIsDimOnly(t *testing.T) {
	m := newMDModel(t, true, true, true, 80, 60)
	id := addReply(&m, mdSamples["everything"])
	paramsOf := func(rows []string) map[string]bool {
		got := map[string]bool{}
		for _, m := range sgrRe.FindAllStringSubmatch(strings.Join(rows, "\n"), -1) {
			got[m[1]] = true
		}
		return got
	}
	live, _ := itemRows(t, m, id)
	before := paramsOf(live)
	if !before["38;5;117"] || !before["1"] {
		t.Fatalf("positive control failed: unflattened rows lack accent or bold (%v)", before)
	}

	m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
	m.relayout(m.layout())
	flat, plain := itemRows(t, m, id)
	for p := range paramsOf(flat) {
		if p != "38;5;245" && p != "0" {
			t.Errorf("flattened reply emitted SGR %q, want dim (38;5;245) only", p)
		}
	}
	if !paramsOf(flat)["38;5;245"] {
		t.Error("flattened reply carries no dim at all")
	}
	stripped := make([]string, len(flat))
	for i, r := range flat {
		stripped[i] = sgrRe.ReplaceAllString(r, "")
	}
	if !equalRows(stripped, plain) {
		t.Error("flattened rows do not strip to the plain rows")
	}
}

// TestCopyOfARenderedReplyIsItsSource: `c` on a rendered reply copies the source
// text, not the rendered rows. Entry point: the `c` key through Update in
// Browsing, then the Cmd run. The reply's rendered rows are shown to differ from
// its source, so a payload built from the rows fails the assertion.
func TestCopyOfARenderedReplyIsItsSource(t *testing.T) {
	const src = "# Head\n**bold** and `code`\n- item"
	rec := &copyRecorder{}
	m := newMDModel(t, false, true, true, 80, 24)
	m.Copy = rec.copy
	id := addReply(&m, src)
	_, rows := itemRows(t, m, id)
	if strings.Join(rows, "\n") == src {
		t.Fatalf("setup: the rendered rows equal the source, so the test cannot tell them apart: %q", rows)
	}

	m = browse(m)
	m.sel = id
	next, cmd := pressC(m)
	runCopyCmd(t, next, cmd)
	if len(rec.got) != 1 || rec.got[0] != src {
		t.Errorf("c handed %q to Copy, want the source [%q]", rec.got, src)
	}
}
