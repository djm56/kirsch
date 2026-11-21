package tui

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// These tests cover syntax highlighting, ui-spec §3.2 and §4.1.
//
// The mapping tests call hlLexer, hlRole and hlRuns directly. The fence tests
// call mdLayout and drawMarkdown, and mdCache.rows for the cache. The modal
// tests go through a Model: a card is appended, the Model's own openDetail (or
// applyDiffResult for /diff) opens the modal, and modalRows draws each line
// with the Model's own Styles, so the colours asserted are the ones drawn.

// The §10.1 foreground codes the tests name. They are written out here, apart
// from styles.go, because the tests assert what the palette table says, not
// what the code under test says.
const (
	cDim    = "245"
	cMuted  = "248"
	cText   = "253"
	cAccent = "117"
	cOK     = "120"
	cErr    = "203"
	cHunk   = "123"
	cKw     = "141"
	cStr    = "180"
	cNum    = "216"
	cType   = "152"
)

// hlSeg is a stretch of drawn text and the foreground it is drawn in.
type hlSeg struct{ fg, text string }

var hlSegRe = regexp.MustCompile("\x1b\\[38;5;(\\d+)m([^\x1b]*)\x1b\\[0m")

// hlSegs splits a drawn line into its coloured stretches, merging neighbours of
// one colour. It fails the test if anything in s is not such a stretch, so an
// unstyled byte or a stray escape cannot slip past.
func hlSegs(t *testing.T, s string) []hlSeg {
	t.Helper()
	if rest := hlSegRe.ReplaceAllString(s, ""); rest != "" {
		t.Fatalf("drawn line holds bytes outside a foreground stretch: %q in %q", rest, s)
	}
	var out []hlSeg
	for _, m := range hlSegRe.FindAllStringSubmatch(s, -1) {
		if n := len(out); n > 0 && out[n-1].fg == m[1] {
			out[n-1].text += m[2]
			continue
		}
		out = append(out, hlSeg{m[1], m[2]})
	}
	return out
}

func segsEqual(a, b []hlSeg) bool {
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

// hlTok is a token's text and the role it is drawn in.
type hlTok struct {
	text string
	role mdRole
}

// hlToks merges neighbouring runs of one role.
func hlToks(runs []mdRun) []hlTok {
	var out []hlTok
	for _, r := range runs {
		if n := len(out); n > 0 && out[n-1].role == r.Role {
			out[n-1].text += r.Text
			continue
		}
		out = append(out, hlTok{r.Text, r.Role})
	}
	return out
}

func toksEqual(a, b []hlTok) bool {
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

func lexerName(lx chroma.Lexer) string {
	if lx == nil {
		return ""
	}
	return lx.Config().Name
}

// TestHighlightLanguageMapping pins the one language mapping. Entry point:
// hlLexer, with a path and with a fence info string. Unmapped input must give
// no lexer at all, never chroma's fallback. Positive controls: each mapped case
// names its lexer; "main.go" maps as a path but not as an info string, so the
// info path really refuses a filename that chroma's Get would map by its
// filename fallback; and "Makefile" as an info string maps, because it is the
// Makefile lexer's own name, so the name-or-alias check admits what it should.
func TestHighlightLanguageMapping(t *testing.T) {
	paths := []struct{ in, want string }{
		{"foo.go", "Go"},
		{"calc/divide.go", "Go"},
		{"Makefile", "Makefile"},
		{"Dockerfile", "Docker"},
		{"x.py", "Python"},
		{"x.unknownext", ""},
		{"", ""},
	}
	for _, c := range paths {
		if got := lexerName(hlLexer(hlPath, c.in)); got != c.want {
			t.Errorf("path %q: lexer %q, want %q", c.in, got, c.want)
		}
	}
	infos := []struct{ in, want string }{
		{"go", "Go"},
		{"Go", "Go"},
		{"GO", "Go"},
		{"python", "Python"},
		{`rust title="x"`, "Rust"},
		{"  go  ", "Go"},
		{"nosuchlang", ""},
		{"", ""},
		{"main.go", ""},
		{"x.unknownext", ""},
		{"Makefile", "Makefile"},
		{"makefile", "Makefile"},
	}
	for _, c := range infos {
		if got := lexerName(hlLexer(hlInfo, c.in)); got != c.want {
			t.Errorf("info %q: lexer %q, want %q", c.in, got, c.want)
		}
	}
	if hlLexer(hlPath, "main.go") == nil {
		t.Error("positive control: the path main.go maps to no lexer")
	}
}

// TestHighlightTokenRoles pins the token-to-role mapping of §3.2, one case per
// row of the mapping. Entry point: hlRuns on one real line each, then hlRole on
// token types the lines do not reach. Every row is asserted token by token, so a
// token drawn in the wrong role fails. For each token type in the sample below,
// success and error are asserted absent from its role, with the role it does
// map to as the control; the sample is not every token type chroma has.
// Keyword constants that a lexer tags as a name constant (Nix true, MySQL NULL)
// are keywords; a name exception is text.
func TestHighlightTokenRoles(t *testing.T) {
	goLex := hlLexer(hlPath, "a.go")
	pyLex := hlLexer(hlPath, "a.py")
	cases := []struct {
		name string
		lx   chroma.Lexer
		line string
		want []hlTok
	}{
		{
			"keyword, function name, string, number, comment", goLex,
			`func main() { x := "s" + 42 // c }`,
			[]hlTok{
				{"func", roleSynKeyword},
				{" ", roleText},
				{"main", roleAccent},
				{`() { x := `, roleText},
				{`"s"`, roleSynString},
				{" + ", roleText},
				{"42", roleSynNumber},
				{" ", roleText},
				{"// c }", roleMuted},
			},
		},
		{
			"keyword constants, keyword type, builtin", goLex,
			`if x == nil || true { var s string; append(a, b) }`,
			[]hlTok{
				{"if", roleSynKeyword},
				{" x == ", roleText},
				{"nil", roleSynKeyword},
				{" || ", roleText},
				{"true", roleSynKeyword},
				{" { ", roleText},
				{"var", roleSynKeyword},
				{" s ", roleText},
				{"string", roleSynType},
				{"; ", roleText},
				{"append", roleAccent},
				{"(a, b) }", roleText},
			},
		},
		{
			"class name", pyLex,
			`class A(B): return None`,
			[]hlTok{
				{"class", roleSynKeyword},
				{" ", roleText},
				{"A", roleSynType},
				{"(B): ", roleText},
				{"return", roleSynKeyword},
				{" ", roleText},
				{"None", roleSynKeyword},
			},
		},
		{
			"name constant as keyword (Nix)", hlLexer(hlInfo, "nix"),
			`x = true;`,
			[]hlTok{
				{"x = ", roleText},
				{"true", roleSynKeyword},
				{";", roleText},
			},
		},
		{
			"name constant as keyword (MySQL)", hlLexer(hlInfo, "mysql"),
			`SELECT NULL;`,
			[]hlTok{
				{"SELECT", roleSynKeyword},
				{" ", roleText},
				{"NULL", roleSynKeyword},
				{";", roleText},
			},
		},
	}
	for _, c := range cases {
		if c.lx == nil {
			t.Fatalf("%s: no lexer", c.name)
		}
		got := hlToks(hlRuns(c.lx, c.line))
		if !toksEqual(got, c.want) {
			t.Errorf("%s:\n got  %v\n want %v", c.name, got, c.want)
		}
	}

	types := []struct {
		t    chroma.TokenType
		want mdRole
	}{
		{chroma.Keyword, roleSynKeyword},
		{chroma.KeywordConstant, roleSynKeyword},
		{chroma.KeywordDeclaration, roleSynKeyword},
		{chroma.KeywordType, roleSynType},
		{chroma.NameClass, roleSynType},
		{chroma.NameConstant, roleSynKeyword},
		{chroma.NameException, roleText},
		{chroma.NameBuiltin, roleAccent},
		{chroma.NameBuiltinPseudo, roleAccent},
		{chroma.NameFunction, roleAccent},
		{chroma.NameFunctionMagic, roleAccent},
		{chroma.LiteralString, roleSynString},
		{chroma.LiteralStringDouble, roleSynString},
		{chroma.LiteralStringEscape, roleSynString},
		{chroma.LiteralNumber, roleSynNumber},
		{chroma.LiteralNumberFloat, roleSynNumber},
		{chroma.LiteralNumberHex, roleSynNumber},
		{chroma.LiteralDate, roleSynNumber},
		{chroma.Literal, roleSynNumber},
		{chroma.CommentSingle, roleMuted},
		{chroma.CommentMultiline, roleMuted},
		{chroma.CommentHashbang, roleMuted},
		{chroma.Comment, roleMuted},
		{chroma.CommentPreproc, roleText},
		{chroma.CommentPreprocFile, roleText},
		{chroma.Operator, roleText},
		{chroma.OperatorWord, roleText},
		{chroma.Punctuation, roleText},
		{chroma.Text, roleText},
		{chroma.TextWhitespace, roleText},
		{chroma.NameVariable, roleText},
		{chroma.NameOther, roleText},
		{chroma.Error, roleText},
		{chroma.GenericDeleted, roleText},
	}
	for _, c := range types {
		if got := hlRole(c.t); got != c.want {
			t.Errorf("hlRole(%v) = %d, want %d", c.t, got, c.want)
		}
		if r := hlRole(c.t); r == roleSuccess || r == roleError {
			t.Errorf("hlRole(%v) = a diff-marker role", c.t)
		}
	}
}

// fenceLine is a Go line with a keyword, a function name, a string, a number
// and a comment.
const fenceLine = `func main() { x := "s" + 42 // c }`

func fence(info, body string) string { return "```" + info + "\n" + body + "\n```" }

// TestReplyFenceHighlights is the fence rows of §3.2. Entry point: mdLayout
// then drawMarkdown with identity and live styles. A Go fence carries the
// keyword, accent, string, number and muted roles; the plain rows are the exact
// rows a fence had before highlighting (two columns of indent and the line);
// a fence with an unmapped language, one with no language, and an indented block
// are all text. Positive control: the Go fence's live rows carry the syntax
// colours that the unmapped fence's rows do not.
func TestReplyFenceHighlights(t *testing.T) {
	g := NewGlyphs(true)
	rows, ok := mdLayout(fence("go", fenceLine), 80, g)
	if !ok || len(rows) != 1 {
		t.Fatalf("go fence: ok=%v rows=%d, want 1 row", ok, len(rows))
	}
	got := hlToks(rows[0])
	want := []hlTok{
		{"  ", roleText},
		{"func", roleSynKeyword},
		{" ", roleText},
		{"main", roleAccent},
		{`() { x := `, roleText},
		{`"s"`, roleSynString},
		{" + ", roleText},
		{"42", roleSynNumber},
		{" ", roleText},
		{"// c }", roleMuted},
	}
	if !toksEqual(got, want) {
		t.Errorf("go fence roles:\n got  %v\n want %v", got, want)
	}

	plainWant := []string{"  " + fenceLine}
	if got := drawMarkdown(rows, mdIdentity); !equalRows(got, plainWant) {
		t.Errorf("go fence plain rows:\n got  %q\n want %q", got, plainWant)
	}
	live := drawMarkdown(rows, mdLiveSty)
	for _, code := range []string{cKw, cAccent, cStr, cNum, cMuted} {
		if !strings.Contains(live[0], "\x1b[38;5;"+code+"m") {
			t.Errorf("go fence live row lacks colour %s: %q", code, live[0])
		}
	}

	for _, info := range []string{"nosuchlang", ""} {
		rows, ok := mdLayout(fence(info, fenceLine), 80, g)
		if !ok {
			t.Fatalf("fence %q: layout failed", info)
		}
		for _, run := range rows[0] {
			if run.Role != roleText {
				t.Errorf("fence %q: run %q has role %d, want text", info, run.Text, run.Role)
			}
		}
		if got := drawMarkdown(rows, mdIdentity); !equalRows(got, plainWant) {
			t.Errorf("fence %q plain rows:\n got  %q\n want %q", info, got, plainWant)
		}
		for _, code := range []string{cKw, cStr, cNum, cMuted, cAccent} {
			if strings.Contains(drawMarkdown(rows, mdLiveSty)[0], "\x1b[38;5;"+code+"m") {
				t.Errorf("fence %q: unmapped fence drawn in colour %s", info, code)
			}
		}
	}

	// An indented block has no info string, so it is plain.
	rows, _ = mdLayout("para\n\n    "+fenceLine, 80, g)
	for _, row := range rows {
		for _, run := range row {
			if run.Role != roleText {
				t.Errorf("indented block: run %q has role %d, want text", run.Text, run.Role)
			}
		}
	}
}

// TestReplyFenceKeepsTheWidthAndTheText is the width post-condition and the
// identical-layout property for a highlighted fence. Entry point: mdLayout then
// drawMarkdown at every width from 20 to 120, in both glyph sets. No row may be
// wider than the width; every wrapped row keeps the two-column indent; the rows'
// text joined back is the source line, so the hard wrap lost nothing; and the live
// rows stripped equal the identity rows. Positive control: at the widest width the
// live rows carry the keyword colour, and at the narrowest the line wraps, so the
// wrap path is the one under test.
func TestReplyFenceKeepsTheWidthAndTheText(t *testing.T) {
	line := `func compute(a int, b string) { s := "a long string literal for wrapping" + 123456 // and a trailing comment }`
	src := fence("go", line)
	for _, uni := range []bool{true, false} {
		g := NewGlyphs(uni)
		for w := 20; w <= 120; w++ {
			rows, ok := mdLayout(src, w, g)
			if !ok {
				t.Fatalf("width %d: layout failed", w)
			}
			plainRows := drawMarkdown(rows, mdIdentity)
			liveRows := drawMarkdown(rows, mdLiveSty)
			var joined strings.Builder
			for i, r := range plainRows {
				if cw := cellWidth(r); cw > w {
					t.Errorf("unicode=%v width %d: row %d is %d cells: %q", uni, w, i, cw, r)
				}
				if !strings.HasPrefix(r, "  ") {
					t.Errorf("unicode=%v width %d: row %d lost the block indent: %q", uni, w, i, r)
				}
				joined.WriteString(strings.TrimPrefix(r, "  "))
				if got := stripSGR(liveRows[i]); got != r {
					t.Errorf("unicode=%v width %d: strip(styled) != plain at row %d\n %q\n %q", uni, w, i, got, r)
				}
			}
			if joined.String() != line {
				t.Errorf("unicode=%v width %d: rows join to %q, want the source line", uni, w, joined.String())
			}
			if w == 20 && len(plainRows) < 2 {
				t.Errorf("positive control: the line did not wrap at width 20")
			}
			if w == 120 && !strings.Contains(strings.Join(liveRows, ""), "\x1b[38;5;"+cKw+"m") {
				t.Errorf("positive control: no keyword colour at width 120")
			}
		}
	}
}

// TestReplyFenceHighlightIsCached: the cache holds the highlighted runs, and each
// drawing pass draws them through its own Styles. Entry point: mdCache.rows twice
// for one reply, then drawMarkdown with live, flattened and identity styles. The
// second lookup must not lay out again; the live draw carries syntax colours; the
// flattened draw carries dim alone; the identity draw carries none. Positive
// control: the cached rows hold more than one role, so the draws can differ.
func TestReplyFenceHighlightIsCached(t *testing.T) {
	c := newMdCache()
	g := NewGlyphs(true)
	lines := strings.Split(fence("go", fenceLine), "\n")
	_, ok := c.rows(7, lines, 80, g)
	if !ok {
		t.Fatal("layout failed")
	}
	rows2, _ := c.rows(7, lines, 80, g)
	if c.layouts != 1 {
		t.Errorf("layouts = %d after two lookups, want 1", c.layouts)
	}
	roles := map[mdRole]bool{}
	for _, run := range rows2[0] {
		roles[run.Role] = true
	}
	if len(roles) < 4 {
		t.Fatalf("positive control: cached rows hold roles %v, want several", roles)
	}

	live := drawMarkdown(rows2, mdLiveSty)[0]
	if !strings.Contains(live, "\x1b[38;5;"+cKw+"m") {
		t.Errorf("live draw lacks the keyword colour: %q", live)
	}
	flat := drawMarkdown(rows2, mdLiveSty.Flat())[0]
	for _, p := range sgrRe.FindAllStringSubmatch(flat, -1) {
		if p[1] != "0" && p[1] != "38;5;"+cDim {
			t.Errorf("flattened draw carries SGR %q, want dim only", p[1])
		}
	}
	if !strings.Contains(flat, "\x1b[38;5;"+cDim+"m") {
		t.Errorf("positive control: flattened draw has no dim: %q", flat)
	}
	if id := drawMarkdown(rows2, mdIdentity)[0]; strings.Contains(id, "\x1b") {
		t.Errorf("identity draw holds an escape: %q", id)
	}
}

// hlColourModel is a coloured Unicode Model at 100x30.
func hlColourModel(t *testing.T) Model {
	t.Helper()
	return newMDModel(t, true, true, true, 100, 30)
}

// openCard appends a tool card and opens its detail modal through openDetail.
func openCard(t *testing.T, m Model, name, target string, out []string) Model {
	t.Helper()
	id := m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
		Name: name, Target: target, State: StateOK, Out: out,
	}})
	m.sel = id
	m = m.openDetail(m.layout())
	if m.modal == nil {
		t.Fatalf("openDetail opened no modal for %s", name)
	}
	return m
}

// drawLine draws the first display row of line idx of the open modal at inner
// cells. A line that fits is one row, so for it this is the whole line.
func drawLine(m Model, idx, inner int) string {
	return m.modalRows(m.modal, m.modal.Lines[idx], idx, inner)[0]
}

// TestContentModalHighlightsReadFile is the read_file rule of §4.1. Entry point:
// openDetail on a read_file card, then modalRows for each line. Numbered lines
// draw the modal's own number dim, the tool's number prefix muted, and the rest
// lexed. The prefix is the digits plus exactly N spaces, so "5" and seven spaces
// gives "5" and three. The marker line and a line with too few spaces are text.
// Controls: the numbered lines carry syntax colours, so "all text" cannot pass.
func TestContentModalHighlightsReadFile(t *testing.T) {
	m := openCard(t, hlColourModel(t), "read_file", "calc/divide.go", []string{
		`10  y := 42`,
		"5       foo",
		"12 y := 1",
		"99",
		`100 func f() {}`,
		"1234    x",
		"‹… output truncated …›",
		"no number here",
	})
	if !m.modal.ReadFile || m.modal.Path != "calc/divide.go" {
		t.Fatalf("modal source = ReadFile %v Path %q", m.modal.ReadFile, m.modal.Path)
	}
	cases := []struct {
		idx  int
		want []hlSeg
	}{
		{0, []hlSeg{{cDim, "   1 "}, {cMuted, "10  "}, {cText, "y := "}, {cNum, "42"}}},
		{1, []hlSeg{{cDim, "   2 "}, {cMuted, "5   "}, {cText, "    foo"}}},
		{2, []hlSeg{{cDim, "   3 "}, {cText, "12 y := 1"}}},
		{3, []hlSeg{{cDim, "   4 "}, {cText, "99"}}},
		{4, []hlSeg{{cDim, "   5 "}, {cMuted, "100 "}, {cKw, "func"}, {cText, " "}, {cAccent, "f"}, {cText, "() {}"}}},
		{5, []hlSeg{{cDim, "   6 "}, {cMuted, "1234    "}, {cText, "x"}}},
		{6, []hlSeg{{cDim, "   7 "}, {cText, "‹… output truncated …›"}}},
		{7, []hlSeg{{cDim, "   8 "}, {cText, "no number here"}}},
	}
	for _, c := range cases {
		got := hlSegs(t, drawLine(m, c.idx, 60))
		if !segsEqual(got, c.want) {
			t.Errorf("line %d %q:\n got  %v\n want %v", c.idx, m.modal.Lines[c.idx], got, c.want)
		}
	}
}

// TestContentModalOnlyHighlightsReadFileCards: highlighting applies to no other
// card. Entry point: openDetail on a read_file card and on cards of other tools
// with the same target and lines. Positive control: the read_file card draws the
// keyword colour for the same line.
func TestContentModalOnlyHighlightsReadFileCards(t *testing.T) {
	out := []string{`1   func f() {}`}
	hl := openCard(t, hlColourModel(t), "read_file", "a.go", out)
	if !strings.Contains(drawLine(hl, 0, 60), "\x1b[38;5;"+cKw+"m") {
		t.Fatal("positive control: the read_file card line has no keyword colour")
	}
	for _, name := range []string{"git_diff", "search_code", "run_command"} {
		m := openCard(t, hlColourModel(t), name, "a.go", out)
		if m.modal.ReadFile {
			t.Errorf("%s: modal marked as a read_file view", name)
		}
		want := []hlSeg{{cDim, "   1 "}, {cText, "1   func f() {}"}}
		if got := hlSegs(t, drawLine(m, 0, 60)); !segsEqual(got, want) {
			t.Errorf("%s card:\n got  %v\n want %v", name, got, want)
		}
	}
	// An unmapped extension is plain too.
	m := openCard(t, hlColourModel(t), "read_file", "notes.unknownext", out)
	want := []hlSeg{{cDim, "   1 "}, {cText, "1   func f() {}"}}
	if got := hlSegs(t, drawLine(m, 0, 60)); !segsEqual(got, want) {
		t.Errorf("unknown extension:\n got  %v\n want %v", got, want)
	}
}

// TestContentModalWrapsAfterStyling: a long line is lexed whole and then wrapped.
// Entry point: modalRows on a read_file line whose string literal runs past the
// width, through openCard (openDetail). The line takes several rows, each a full
// inner wide but the last; no row carries the truncation marker; the text of the
// rows laid end to end is the line; and the string colour reaches the first cut
// (rows[0]) and holds on rows[1], the row below it: lexing the fragments instead
// would see an unterminated string and would not colour them. Positive control:
// the same line at a width it fits is one row and carries the string colour.
func TestContentModalWrapsAfterStyling(t *testing.T) {
	long := `1   s := "` + strings.Repeat("a", 300) + `"`
	m := openCard(t, hlColourModel(t), "read_file", "a.go", []string{long})
	fits := drawRows(m, 0, 400)
	if len(fits) != 1 || !strings.Contains(fits[0], "\x1b[38;5;"+cStr+"m") {
		t.Fatalf("positive control: the line at 400 cells is %d rows and has no string colour: %q", len(fits), fits)
	}
	for _, uni := range []bool{true, false} {
		mm := newMDModel(t, true, uni, true, 100, 30)
		mm = openCard(t, mm, "read_file", "a.go", []string{long})
		const inner = 40
		rows := drawRows(mm, 0, inner)
		const textW = inner - 5
		if want := (cellWidth(long) + textW - 1) / textW; len(rows) != want {
			t.Fatalf("unicode=%v: %d rows, want %d", uni, len(rows), want)
		}
		var text strings.Builder
		for i, row := range rows {
			if strings.Contains(stripSGR(row), mm.gly.Trunc) {
				t.Errorf("unicode=%v row %d: carries the truncation marker: %q", uni, i, stripSGR(row))
			}
			if w := cellWidth(row); i < len(rows)-1 && w != inner {
				t.Errorf("unicode=%v row %d: visible width %d, want %d", uni, i, w, inner)
			}
			text.WriteString(stripSGR(row)[5:])
		}
		if text.String() != long {
			t.Errorf("unicode=%v: rows laid end to end are %q, want the line %q", uni, text.String(), long)
		}
		first := hlSegs(t, rows[0])
		if last := first[len(first)-1]; last.fg != cStr {
			t.Errorf("unicode=%v: the first row ends in %v, want the string colour", uni, last)
		}
		mid := hlSegs(t, strings.TrimPrefix(rows[1], "     "))
		if len(mid) != 1 || mid[0].fg != cStr || mid[0].text != strings.Repeat("a", textW) {
			t.Errorf("unicode=%v: a continuation row is %v, want %d a's in the string colour", uni, mid, textW)
		}
	}
}

// TestApprovalDiffHighlights is the patch approval diff rule. Entry point:
// openDetail on a patch approval card, then modalRows. The +/- marker keeps its
// success or error colour and the rest of the line is lexed; @@ is hunk; context
// is muted whole and not lexed; a "+++" code line is a code line. Controls: each
// code line has syntax colour, so "plain" cannot pass.
func TestApprovalDiffHighlights(t *testing.T) {
	m := hlColourModel(t)
	id := m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{
		Kind: ApprovalPatch, DiffFilename: "calc/x.go", Added: 3, Removed: 1,
		Diff: []string{
			"@@ -1,3 +1,3 @@",
			` return nil`,
			`-old := 1`,
			`+new := "s" // c`,
			`+++42`,
			`---x`,
		},
	}})
	m.sel = id
	m = m.openDetail(m.layout())
	if m.modal == nil || m.modal.Kind != ModalDiff {
		t.Fatalf("modal = %+v, want the diff modal", m.modal)
	}
	cases := []struct {
		idx  int
		want []hlSeg
	}{
		{0, []hlSeg{{cHunk, "@@ -1,3 +1,3 @@"}}},
		{1, []hlSeg{{cMuted, " return nil"}}},
		{2, []hlSeg{{cErr, "-"}, {cText, "old := "}, {cNum, "1"}}},
		{3, []hlSeg{{cOK, "+"}, {cText, "new := "}, {cStr, `"s"`}, {cText, " "}, {cMuted, "// c"}}},
		{4, []hlSeg{{cOK, "+"}, {cText, "++"}, {cNum, "42"}}},
		{5, []hlSeg{{cErr, "-"}, {cText, "--x"}}},
	}
	for _, c := range cases {
		got := hlSegs(t, drawLine(m, c.idx, 60))
		if !segsEqual(got, c.want) {
			t.Errorf("line %d %q:\n got  %v\n want %v", c.idx, m.modal.Lines[c.idx], got, c.want)
		}
	}
}

// TestApprovalDiffLanguageComesFromTheTitle: the language is the title's path,
// and for a rename the new path after " → ". Entry point: modalRows on a diff
// modal opened with each title. The same line "+func f() {}" is a Go keyword
// under a Go path and plain under a Python path, so the rename cases decide which
// path was used in both directions.
func TestApprovalDiffLanguageComesFromTheTitle(t *testing.T) {
	line := "+func f() {}"
	cases := []struct {
		title  string
		wantKw bool
	}{
		{"a.go", true},
		{"a.py", false},
		{"old.py → new.go", true},
		{"old.go → new.py", false},
		{"unmapped.unknownext", false},
	}
	for _, c := range cases {
		m := hlColourModel(t)
		m.openModal(ModalState{Kind: ModalDiff, Title: c.title, Lines: []string{line}})
		got := hlSegs(t, drawLine(m, 0, 60))
		hasKw := false
		for _, s := range got {
			hasKw = hasKw || (s.fg == cKw && s.text == "func")
		}
		if hasKw != c.wantKw {
			t.Errorf("title %q: keyword colour = %v, want %v (%v)", c.title, hasKw, c.wantKw, got)
		}
		if got[0].fg != cOK || !strings.HasPrefix(got[0].text, "+") {
			t.Errorf("title %q: first stretch %v, want the success +", c.title, got[0])
		}
	}
}

// workTreeLines is a /diff of three files: a Go file, a Python file, and a file
// whose diff --git line is quoted so it cannot be parsed.
func workTreeLines() []string {
	return []string{
		/* 0 */ "diff --git a/a.go b/a.go",
		/* 1 */ "index 111..222 100644",
		/* 2 */ "--- a/a.go",
		/* 3 */ "+++ b/a.go",
		/* 4 */ "@@ -1,3 +1,3 @@",
		/* 5 */ " return nil",
		/* 6 */ "-x := 1",
		/* 7 */ "+func f() {}",
		/* 8 */ "+++42",
		/* 9 */ "diff --git a/b.py b/b.py",
		/* 10 */ "--- a/b.py",
		/* 11 */ "+++ b/b.py",
		/* 12 */ "@@ -1 +1 @@",
		/* 13 */ "+def f(): pass",
		/* 14 */ "+func f() {}",
		/* 15 */ `diff --git "a/x y.go" "b/x y.go"`,
		/* 16 */ "--- a/x y.go",
		/* 17 */ "+++ b/x y.go",
		/* 18 */ "@@ -1 +1 @@",
		/* 19 */ "+func f() {}",
	}
}

func workTreeModal(t *testing.T, lines []string) Model {
	t.Helper()
	m := hlColourModel(t)
	m.applyDiffResult(DiffResult{Lines: lines, Added: 1, Removed: 1})
	if m.modal == nil || m.modal.Kind != ModalDiff || !m.modal.WorkTree {
		t.Fatalf("applyDiffResult opened %+v, want the working tree diff", m.modal)
	}
	return m
}

// TestWorkTreeDiffHighlightsPerFile is the /diff rule. Entry point:
// applyDiffResult, then modalRows on every line. Each file section takes its
// language from its own diff --git line, so "+func f() {}" is a keyword in the Go
// section and plain in the Python one, and "+def f(): pass" the other way round.
// Header lines keep the red and green their leading character gives and are not
// lexed; after the first @@ a "+++42" line is a code line. A section whose diff
// --git line does not parse is plain: its code line is green whole, with the
// header lines of that section still by leading character. Context stays muted.
func TestWorkTreeDiffHighlightsPerFile(t *testing.T) {
	m := workTreeModal(t, workTreeLines())
	cases := []struct {
		idx  int
		want []hlSeg
	}{
		{0, []hlSeg{{cMuted, "diff --git a/a.go b/a.go"}}},
		{1, []hlSeg{{cMuted, "index 111..222 100644"}}},
		{2, []hlSeg{{cErr, "--- a/a.go"}}},
		{3, []hlSeg{{cOK, "+++ b/a.go"}}},
		{4, []hlSeg{{cHunk, "@@ -1,3 +1,3 @@"}}},
		{5, []hlSeg{{cMuted, " return nil"}}},
		{6, []hlSeg{{cErr, "-"}, {cText, "x := "}, {cNum, "1"}}},
		{7, []hlSeg{{cOK, "+"}, {cKw, "func"}, {cText, " "}, {cAccent, "f"}, {cText, "() {}"}}},
		{8, []hlSeg{{cOK, "+"}, {cText, "++"}, {cNum, "42"}}},
		{9, []hlSeg{{cMuted, "diff --git a/b.py b/b.py"}}},
		{10, []hlSeg{{cErr, "--- a/b.py"}}},
		{11, []hlSeg{{cOK, "+++ b/b.py"}}},
		{12, []hlSeg{{cHunk, "@@ -1 +1 @@"}}},
		{13, []hlSeg{{cOK, "+"}, {cKw, "def"}, {cText, " "}, {cAccent, "f"}, {cText, "(): "}, {cKw, "pass"}}},
		{14, []hlSeg{{cOK, "+"}, {cText, "func f() {}"}}},
		{15, []hlSeg{{cMuted, `diff --git "a/x y.go" "b/x y.go"`}}},
		{16, []hlSeg{{cErr, "--- a/x y.go"}}},
		{17, []hlSeg{{cOK, "+++ b/x y.go"}}},
		{18, []hlSeg{{cHunk, "@@ -1 +1 @@"}}},
		{19, []hlSeg{{cOK, "+func f() {}"}}},
	}
	for _, c := range cases {
		got := hlSegs(t, drawLine(m, c.idx, 80))
		if !segsEqual(got, c.want) {
			t.Errorf("line %d %q:\n got  %v\n want %v", c.idx, m.modal.Lines[c.idx], got, c.want)
		}
	}
}

// TestWorkTreeDiffBPathParsing: only an unambiguous "a/<p> b/<p>" line gives a
// language. Entry point: hlDiffBPath. The no-prefix, mnemonic, quoted and
// ambiguous forms give "" (plain); a path with a space and a rename give the b/
// path. Positive controls are the parsed cases.
func TestWorkTreeDiffBPathParsing(t *testing.T) {
	cases := []struct{ line, want string }{
		{"diff --git a/a.go b/a.go", "a.go"},
		{"diff --git a/dir/my file.go b/dir/my file.go", "dir/my file.go"},
		{"diff --git a/old.py b/new.go", "new.go"},
		{"diff --git a/x b/", ""},
		{"diff --git a/a b/b b/c", ""},
		{`diff --git "a/x y.go" "b/x y.go"`, ""},
		{"diff --git i/a.go w/a.go", ""},
		{"diff --git a.go a.go", ""},
		{"diff --git", ""},
	}
	for _, c := range cases {
		if got := hlDiffBPath(c.line); got != c.want {
			t.Errorf("hlDiffBPath(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

// TestWorkTreeDiffTruncationMarkerEndsHighlighting: a marker line ends
// highlighting until the next diff --git. Entry point: modalRows on a /diff with
// each marker form, the tools' own and the trailing one in both glyph sets. The
// code line after the marker is drawn whole in green; the code line before it and
// the one after the next diff --git are lexed. The marker line itself is muted.
func TestWorkTreeDiffTruncationMarkerEndsHighlighting(t *testing.T) {
	for _, uni := range []bool{true, false} {
		for _, name := range []string{"tool", "trailing"} {
			m := newMDModel(t, true, uni, true, 100, 30)
			marker := ToolTruncMarker
			if name == "trailing" {
				marker = m.gly.Trunc + " truncated — 200 KiB cap"
			}
			m.applyDiffResult(DiffResult{Lines: []string{
				/* 0 */ "diff --git a/c.go b/c.go",
				/* 1 */ "@@ -1 +1 @@",
				/* 2 */ "+func f() {}",
				/* 3 */ marker,
				/* 4 */ "+func g() {}",
				/* 5 */ "diff --git a/d.go b/d.go",
				/* 6 */ "@@ -1 +1 @@",
				/* 7 */ "+func h() {}",
			}})
			if m.modal == nil {
				t.Fatalf("%s unicode=%v: no modal", name, uni)
			}
			hasKw := func(idx int) bool {
				for _, s := range hlSegs(t, drawLine(m, idx, 80)) {
					if s.fg == cKw {
						return true
					}
				}
				return false
			}
			if !hasKw(2) || !hasKw(7) {
				t.Errorf("%s unicode=%v: lines before the marker and after the next diff --git are not lexed", name, uni)
			}
			if hasKw(4) {
				t.Errorf("%s unicode=%v: the line after the marker was lexed", name, uni)
			}
			if got, want := hlSegs(t, drawLine(m, 4, 80)), []hlSeg{{cOK, "+func g() {}"}}; !segsEqual(got, want) {
				t.Errorf("%s unicode=%v: line after marker %v, want %v", name, uni, got, want)
			}
			if got, want := hlSegs(t, drawLine(m, 3, 80)), []hlSeg{{cMuted, marker}}; !segsEqual(got, want) {
				t.Errorf("%s unicode=%v: marker line %v, want %v", name, uni, got, want)
			}
		}
	}
}

// TestWorkTreeDiffTrailingMarkerFromApplyDiffResult: the marker the Model itself
// appends is the one the section pass recognises. Entry point: applyDiffResult
// with Truncated set. The appended last line is muted whole, and the line before
// it is still lexed.
func TestWorkTreeDiffTrailingMarkerFromApplyDiffResult(t *testing.T) {
	m := hlColourModel(t)
	m.applyDiffResult(DiffResult{
		Lines:     []string{"diff --git a/a.go b/a.go", "@@ -1 +1 @@", "+func f() {}"},
		Truncated: true, CapBytes: 200 << 10,
	})
	last := len(m.modal.Lines) - 1
	if !strings.HasPrefix(m.modal.Lines[last], m.gly.Trunc+" truncated") {
		t.Fatalf("setup: last line %q is not the trailing marker", m.modal.Lines[last])
	}
	if got := hlSegs(t, drawLine(m, last, 80)); len(got) != 1 || got[0].fg != cMuted {
		t.Errorf("trailing marker drawn %v, want muted whole", got)
	}
	hasKw := false
	for _, s := range hlSegs(t, drawLine(m, last-1, 80)) {
		hasKw = hasKw || s.fg == cKw
	}
	if !hasKw {
		t.Error("the line before the trailing marker is not lexed")
	}
}

// TestModalSectionStateIsComputedOncePerOpen: the highlighting state is worked out
// the first time something needs it, and not again. Entry point: modalBox, which
// draws the body lines, called repeatedly with the modal scrolled by Update's own
// scroll keys. The count is read from the ModalState's own hook field. Positive
// controls: nothing is computed before the first draw, and a second open of the
// modal computes again, so the counter does count.
func TestModalSectionStateIsComputedOncePerOpen(t *testing.T) {
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, "diff --git a/a.go b/a.go", "@@ -1 +1 @@", "+func f() {}")
	}
	for _, kind := range []string{"worktree", "approval", "readfile"} {
		m := hlColourModel(t)
		switch kind {
		case "worktree":
			m.applyDiffResult(DiffResult{Lines: lines})
		case "approval":
			m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: lines})
		case "readfile":
			m = openCard(t, m, "read_file", "a.go", lines)
		}
		if m.modal.hlComputes != 0 {
			t.Fatalf("%s: computed %d times before any draw", kind, m.modal.hlComputes)
		}
		for i := 0; i < 30; i++ {
			m.modalBox(m.layout())
			m = drive(m, key('j'))
		}
		if m.modal.hlComputes != 1 {
			t.Errorf("%s: computed %d times over 30 draws, want 1", kind, m.modal.hlComputes)
		}
		if m.modal.Off == 0 {
			t.Errorf("%s: control: the modal never scrolled", kind)
		}
		first := m.modal
		m.closeModal()
		switch kind {
		case "worktree":
			m.applyDiffResult(DiffResult{Lines: lines})
		case "approval":
			m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: lines})
		case "readfile":
			m = openCard(t, m, "read_file", "a.go", lines)
		}
		m.modalBox(m.layout())
		if m.modal == first || m.modal.hlComputes != 1 {
			t.Errorf("%s: a reopened modal did not compute afresh (count %d)", kind, m.modal.hlComputes)
		}
	}
}

// hlOutputs builds, for one Model, every highlighted surface: the modal body rows
// of a read_file modal, the rows of a working tree diff, and a reply with a Go
// fence. It returns them joined, so a test can look at their escapes together.
func hlOutputs(t *testing.T, colour bool) (modalRows []string, reply []string) {
	t.Helper()
	m := newMDModel(t, colour, true, true, 100, 30)
	id := addReply(&m, "text\n\n"+fence("go", fenceLine))
	live, _ := itemRows(t, m, id)
	reply = append(reply, live...)

	m = openCard(t, m, "read_file", "a.go", []string{`1   func f(s string) { return "s" + 4 }`})
	box, _, _ := m.modalBox(m.layout())
	modalRows = append(modalRows, box...)
	m.closeModal()

	m.applyDiffResult(DiffResult{Lines: workTreeLines()})
	box, _, _ = m.modalBox(m.layout())
	modalRows = append(modalRows, box...)
	return modalRows, reply
}

// TestHighlightEmitsNoEscapeWithoutColour is property 2 for the highlighted
// surfaces. Entry point: a Model with colour off and a Model with colour on,
// each drawing a fence, a read_file modal and a /diff modal. The colour-off rows
// must hold no 0x1b; the colour-on rows must hold syntax colours (the positive
// control), and stripped must equal the colour-off rows byte for byte.
func TestHighlightEmitsNoEscapeWithoutColour(t *testing.T) {
	offModal, offReply := hlOutputs(t, false)
	onModal, onReply := hlOutputs(t, true)
	for name, rows := range map[string][]string{"modal": offModal, "reply": offReply} {
		if strings.Contains(strings.Join(rows, "\n"), "\x1b") {
			t.Errorf("colour off: %s rows contain 0x1b", name)
		}
	}
	for name, rows := range map[string][]string{"modal": onModal, "reply": onReply} {
		out := strings.Join(rows, "\n")
		if !strings.Contains(out, "\x1b[38;5;"+cKw+"m") {
			t.Errorf("positive control: colour on, %s rows hold no keyword colour", name)
		}
	}
	if !equalRows(plain(onModal), offModal) {
		t.Errorf("modal: strip(styled) != unstyled\n%q\n%q", plain(onModal), offModal)
	}
	if !equalRows(plain(onReply), offReply) {
		t.Errorf("reply: strip(styled) != unstyled\n%q\n%q", plain(onReply), offReply)
	}
}

// TestHighlightSGRIsInThePalette is property 3 for the modal and the fence. Entry
// point: the same surfaces as above, drawn live. Every escape is a whole SGR from
// PaletteSGR, none sets a background, and the four syntax colours and the diff
// colours all appear (the positive control), so the allow-list was exercised on
// them. The box border draws a background in no row.
func TestHighlightSGRIsInThePalette(t *testing.T) {
	modalRows, reply := hlOutputs(t, true)
	out := strings.Join(append(modalRows, reply...), "\n")
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
		if strings.HasPrefix(params, "48;") {
			t.Errorf("background SGR %q", params)
		}
		i += loc[1] - 1
	}
	for _, c := range []string{cKw, cStr, cNum, cType, cAccent, cMuted, cOK, cErr, cHunk, cText} {
		if !seen["38;5;"+c] {
			t.Errorf("positive control: colour %s never appeared (saw %v)", c, seen)
		}
	}
}

// TestHighlightSurvivesTheFullFrame: the highlighted modal reaches View(). Entry
// point: View on a Model with a read_file modal open. The syntax colour is in the
// frame, which can only come from the modal because the transcript behind an open
// modal is flattened to dim, and the frame's rows are as tall and as wide as the
// terminal. Positive control: the frame is not all dim.
func TestHighlightSurvivesTheFullFrame(t *testing.T) {
	m := openCard(t, hlColourModel(t), "read_file", "a.go", []string{`1   func f() {}`})
	view := m.View()
	if !strings.Contains(view, "\x1b[38;5;"+cKw+"m") {
		t.Error("the frame holds no keyword colour")
	}
	rows := strings.Split(view, "\n")
	if len(rows) != 30 {
		t.Errorf("frame is %d rows, want 30", len(rows))
	}
	for i, r := range rows {
		if w := cellWidth(r); w > 100 {
			t.Errorf("row %d is %d cells wide", i, w)
		}
	}
}

// hlFakeLexer is a chroma.Lexer whose Tokenise is the function it holds, so a
// test can make lexing fail in one chosen way and leave the rest of hlRuns as
// it is.
type hlFakeLexer struct {
	tokenise func(text string) (chroma.Iterator, error)
}

// Config names the fake; hlRuns does not read it.
func (f hlFakeLexer) Config() *chroma.Config { return &chroma.Config{Name: "fake"} }

// Tokenise calls the function the fake holds.
func (f hlFakeLexer) Tokenise(_ *chroma.TokeniseOptions, text string) (chroma.Iterator, error) {
	return f.tokenise(text)
}

// SetRegistry is a no-op; the fake looks up no other lexer.
func (f hlFakeLexer) SetRegistry(*chroma.LexerRegistry) chroma.Lexer { return f }

// SetAnalyser is a no-op; the fake does no content analysis.
func (f hlFakeLexer) SetAnalyser(func(text string) float32) chroma.Lexer { return f }

// AnalyseText scores every text zero.
func (f hlFakeLexer) AnalyseText(string) float32 { return 0 }

// hlFakeTokens returns a Tokenise function that yields toks, then EOF.
func hlFakeTokens(toks ...chroma.Token) func(string) (chroma.Iterator, error) {
	return func(string) (chroma.Iterator, error) {
		return chroma.Literator(toks...), nil
	}
}

// TestHighlightRunsFallsBackToPlain is the fallback rule of hlRuns: whatever the
// lexer does wrong, the drawn text is the line. Entry point: hlRuns with a fake
// lexer. A lexer that returns an error, one that panics in Tokenise, one whose
// iterator panics after its first token, and ones whose tokens drop or add text
// must each give the line as one text run. Positive controls through the same
// fake: tokens that reassemble the line give their own roles, and tokens that
// add only a trailing newline give the line with the newline dropped, so the
// fake does reach the token path and the plain results are the fallback's.
func TestHighlightRunsFallsBackToPlain(t *testing.T) {
	const line = "if x"
	plainWant := []hlTok{{line, roleText}}
	kw := chroma.Token{Type: chroma.Keyword, Value: "if"}
	rest := chroma.Token{Type: chroma.Text, Value: " x"}
	cases := []struct {
		name string
		lx   hlFakeLexer
		want []hlTok
	}{
		{"error", hlFakeLexer{func(string) (chroma.Iterator, error) {
			return nil, errors.New("lexer failed")
		}}, plainWant},
		{"panic in Tokenise", hlFakeLexer{func(string) (chroma.Iterator, error) {
			panic("lexer panicked")
		}}, plainWant},
		{"panic in the iterator", hlFakeLexer{func(string) (chroma.Iterator, error) {
			n := 0
			return func() chroma.Token {
				n++
				if n > 1 {
					panic("iterator panicked")
				}
				return kw
			}, nil
		}}, plainWant},
		{"tokens drop text", hlFakeLexer{hlFakeTokens(kw, chroma.Token{Type: chroma.Text, Value: " "})}, plainWant},
		{"tokens add text", hlFakeLexer{hlFakeTokens(kw, chroma.Token{Type: chroma.Text, Value: " xy"})}, plainWant},
		{
			"control: tokens reassemble the line",
			hlFakeLexer{hlFakeTokens(kw, rest)},
			[]hlTok{{"if", roleSynKeyword}, {" x", roleText}},
		},
		{
			"control: a trailing newline only",
			hlFakeLexer{hlFakeTokens(kw, chroma.Token{Type: chroma.Text, Value: " x\n"})},
			[]hlTok{{"if", roleSynKeyword}, {" x", roleText}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hlToks(hlRuns(c.lx, line)); !toksEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// hlLongGo is a Go line exactly n columns wide: the keyword "func" and a name,
// so a lexed line has a keyword run and a plain one has none. With wide set,
// the name is two-column characters, led by one "a" when n - 5 is odd, so the
// line has more bytes than columns.
func hlLongGo(n int, wide bool) string {
	if wide {
		return "func " + strings.Repeat("a", (n-5)%2) + strings.Repeat("界", (n-5)/2)
	}
	return "func " + strings.Repeat("a", n-5)
}

// hlHasKeyword reports whether any run is in the keyword role.
func hlHasKeyword(runs []mdRun) bool {
	for _, r := range runs {
		if r.Role == roleSynKeyword {
			return true
		}
	}
	return false
}

// TestHighlightLineLengthCutoff: a line wider than hlMaxLineCols is not lexed
// and is drawn plain, in a reply fence and in the modals that highlight. Entry
// points: hlRuns; mdLayout for a Go fence; modalRows on a read_file content modal and on a
// patch approval diff modal. Each over-the-cutoff line is one column wider than
// its control, which is exactly hlMaxLineCols wide and is lexed (the positive
// control: it has the keyword role, or the keyword colour); hlRuns also lexes a
// line one column under the cutoff. The cutoff is in columns, not bytes: lines
// of two-column characters one under and at the cutoff have more bytes than the
// cutoff and are still lexed, and one over it is not.
func TestHighlightLineLengthCutoff(t *testing.T) {
	goLex := hlLexer(hlPath, "a.go")
	at, over := hlLongGo(hlMaxLineCols, false), hlLongGo(hlMaxLineCols+1, false)
	if cellWidth(at) != hlMaxLineCols || cellWidth(over) != hlMaxLineCols+1 {
		t.Fatalf("setup: widths %d and %d", cellWidth(at), cellWidth(over))
	}

	for _, line := range []string{hlLongGo(hlMaxLineCols-1, false), at} {
		if !hlHasKeyword(hlRuns(goLex, line)) {
			t.Errorf("hlRuns: the %d-column line, within the cutoff, was not lexed", cellWidth(line))
		}
	}
	if got := hlRuns(goLex, over); len(got) != 1 || got[0].Role != roleText || got[0].Text != over {
		t.Errorf("hlRuns: the line over the cutoff is %d runs, want the line as one text run", len(got))
	}
	for _, n := range []int{hlMaxLineCols - 1, hlMaxLineCols} {
		wide := hlLongGo(n, true)
		if cellWidth(wide) != n || len(wide) <= hlMaxLineCols {
			t.Fatalf("setup: wide line is %d columns, %d bytes", cellWidth(wide), len(wide))
		}
		if !hlHasKeyword(hlRuns(goLex, wide)) {
			t.Errorf("hlRuns: a wide line of %d columns, more bytes than the cutoff, was not lexed", n)
		}
	}
	wideOver := hlLongGo(hlMaxLineCols+1, true)
	if cellWidth(wideOver) != hlMaxLineCols+1 {
		t.Fatalf("setup: wide line over the cutoff is %d columns", cellWidth(wideOver))
	}
	if got := hlRuns(goLex, wideOver); len(got) != 1 {
		t.Errorf("hlRuns: a wide line over the cutoff in columns is %d runs, want 1", len(got))
	}

	g := NewGlyphs(true)
	fenceKw := func(line string) bool {
		rows, ok := mdLayout(fence("go", line), 80, g)
		if !ok {
			t.Fatalf("fence: layout failed")
		}
		for _, row := range rows {
			if hlHasKeyword(row) {
				return true
			}
		}
		return false
	}
	if !fenceKw(at) {
		t.Error("fence: the line at the cutoff was not lexed")
	}
	if fenceKw(over) {
		t.Error("fence: the line over the cutoff was lexed")
	}

	readFile := func(line string) string {
		m := openCard(t, hlColourModel(t), "read_file", "a.go", []string{"1   " + line})
		return drawLine(m, 0, 60)
	}
	if !strings.Contains(readFile(at), "\x1b[38;5;"+cKw+"m") {
		t.Error("read_file modal: the line at the cutoff has no keyword colour")
	}
	if got := hlSegs(t, readFile(over)); len(got) != 2 || got[0].fg != cDim || got[1].fg != cText {
		t.Errorf("read_file modal: the line over the cutoff is %v, want the number dim and the rest text", got)
	}

	diff := func(line string) string {
		m := hlColourModel(t)
		m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: []string{"+" + line}})
		return drawLine(m, 0, 60)
	}
	if !strings.Contains(diff(at), "\x1b[38;5;"+cKw+"m") {
		t.Error("diff modal: the line at the cutoff has no keyword colour")
	}
	if got := hlSegs(t, diff(over)); len(got) != 1 || got[0].fg != cOK {
		t.Errorf("diff modal: the line over the cutoff is %v, want green whole", got)
	}
}

// TestModalLexesEachVisibleLineOnce: a modal lexes each line at most once per
// open, however often it is redrawn or scrolled. Entry point: modalBox, which
// draws the body lines, called after each of Update's own scroll keys, down and
// back up. Each code line carries a name of its own, vNNN, so the drawn rows
// say which code lines were visible; the lex count is read from the
// ModalState's own hook field and must equal the number of distinct code lines
// drawn. Positive
// controls: the modal was drawn many more times than lines were lexed, and the
// lines drawn outnumber one screen, so scrolling reached new lines.
func TestModalLexesEachVisibleLineOnce(t *testing.T) {
	var code []string
	for i := 0; i < 80; i++ {
		code = append(code, fmt.Sprintf("+v%03d := %d", i, i))
	}
	idRe := regexp.MustCompile(`v(\d{3})`)
	for _, kind := range []string{"approval", "worktree", "readfile"} {
		t.Run(kind, func(t *testing.T) {
			m := hlColourModel(t)
			switch kind {
			case "approval":
				m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: code})
			case "worktree":
				m.applyDiffResult(DiffResult{Lines: append([]string{"diff --git a/a.go b/a.go", "@@ -1 +1 @@"}, code...)})
			case "readfile":
				var lines []string
				for i, c := range code {
					lines = append(lines, fmt.Sprintf("%d   %s", i+1, c[1:]))
				}
				m = openCard(t, m, "read_file", "a.go", lines)
			}
			seen := map[string]bool{}
			draws, perScreen := 0, 0
			draw := func() {
				box, _, _ := m.modalBox(m.layout())
				draws++
				n := 0
				for _, row := range box {
					if id := idRe.FindStringSubmatch(stripSGR(row)); id != nil {
						seen[id[1]] = true
						n++
					}
				}
				perScreen = maxInt(perScreen, n)
			}
			for i := 0; i < 3; i++ {
				draw()
			}
			for i := 0; i < 40; i++ {
				m = drive(m, key('j'))
				draw()
				draw()
			}
			for i := 0; i < 40; i++ {
				m = drive(m, key('k'))
				draw()
			}
			if m.modal.hlLexes != len(seen) {
				t.Errorf("lexed %d lines, want %d, the distinct code lines drawn", m.modal.hlLexes, len(seen))
			}
			if draws < 2*len(seen) || len(seen) <= perScreen || perScreen == 0 {
				t.Errorf("positive control: %d draws, %d distinct lines, %d per screen", draws, len(seen), perScreen)
			}
		})
	}
}

// TestDiffModalWrapsLongMarkedLines: a long "+" or "-" line in the diff modal is
// lexed whole and wrapped, and the colours of its marker stay. Entry point:
// modalRows on a patch approval diff, in both glyph sets. The marker is the first
// cell of the first row, in success or error; each continuation row starts with one
// blank column and no marker; the string colour holds on every row after the first
// cut; and no row carries the truncation glyph. Positive control: the stretch
// before the first cut is the string colour, so the line went through the
// highlighted path and not the plain one.
func TestDiffModalWrapsLongMarkedLines(t *testing.T) {
	body := `s := "` + strings.Repeat("a", 300) + `"`
	for _, uni := range []bool{true, false} {
		for _, c := range []struct{ marker, fg string }{{"+", cOK}, {"-", cErr}} {
			m := newMDModel(t, true, uni, true, 100, 30)
			m.openModal(ModalState{Kind: ModalDiff, Title: "a.go", Lines: []string{c.marker + body}})
			const inner = 40
			rows := drawRows(m, 0, inner)
			// The first row holds inner cells, each later one inner-1 and a blank.
			line := c.marker + body
			if want := 1 + (cellWidth(line)-inner+inner-2)/(inner-1); len(rows) != want {
				t.Fatalf("%s unicode=%v: %d rows, want %d", c.marker, uni, len(rows), want)
			}
			first := hlSegs(t, rows[0])
			if len(first) < 3 {
				t.Fatalf("%s unicode=%v: %v", c.marker, uni, first)
			}
			if f := first[0]; f.fg != c.fg || f.text != c.marker {
				t.Errorf("%s unicode=%v: first stretch %v, want %q in %s", c.marker, uni, f, c.marker, c.fg)
			}
			if first[len(first)-1].fg != cStr {
				t.Errorf("%s unicode=%v: the stretch before the cut is not the string colour: %v", c.marker, uni, first)
			}
			var text strings.Builder
			text.WriteString(stripSGR(rows[0]))
			for i, row := range rows[1:] {
				if !strings.HasPrefix(row, " \x1b[38;5;"+cStr+"m") {
					t.Errorf("%s unicode=%v row %d: want one blank column then the string colour: %q", c.marker, uni, i+1, row)
				}
				text.WriteString(stripSGR(row)[1:])
			}
			if text.String() != c.marker+body {
				t.Errorf("%s unicode=%v: rows laid end to end are %q", c.marker, uni, text.String())
			}
			if strings.Contains(text.String(), m.gly.Trunc) {
				t.Errorf("%s unicode=%v: a row carries the truncation glyph", c.marker, uni)
			}
		}
	}
}

// hlResetBreaker empties the set of lexers that have missed hlLexDeadline. A
// test that trips the breaker registers it with t.Cleanup, so that no later test
// in the package finds a real lexer already tripped. Nothing outside tests
// clears the set. The breaker is package state, so a test that trips it or
// resets it must not run in parallel with another.
func hlResetBreaker() {
	hlSlow.mu.Lock()
	defer hlSlow.mu.Unlock()
	hlSlow.names = map[string]bool{}
}

// hlGateLexer is a chroma.Lexer that counts its Tokenise calls and, when gate is
// not nil, blocks in Tokenise until gate is closed. Once through, it returns the
// whole text as one keyword token, so a lexed line has the keyword role and a
// plain one does not. Its name is the key the breaker records.
type hlGateLexer struct {
	name  string
	gate  chan struct{}
	calls atomic.Int32
}

// Config names the lexer; hlLexLine reads the name for the breaker.
func (g *hlGateLexer) Config() *chroma.Config { return &chroma.Config{Name: g.name} }

// Tokenise counts the call, waits for the gate if there is one, then returns the
// text as one keyword token.
func (g *hlGateLexer) Tokenise(_ *chroma.TokeniseOptions, text string) (chroma.Iterator, error) {
	g.calls.Add(1)
	if g.gate != nil {
		<-g.gate
	}
	return chroma.Literator(chroma.Token{Type: chroma.Keyword, Value: text}), nil
}

// SetRegistry is a no-op; the fake looks up no other lexer.
func (g *hlGateLexer) SetRegistry(*chroma.LexerRegistry) chroma.Lexer { return g }

// SetAnalyser is a no-op; the fake does no content analysis.
func (g *hlGateLexer) SetAnalyser(func(text string) float32) chroma.Lexer { return g }

// AnalyseText scores every text zero.
func (g *hlGateLexer) AnalyseText(string) float32 { return 0 }

// hlOnePlainRun reports whether runs is the line as one text run.
func hlOnePlainRun(runs []mdRun, line string) bool {
	return len(runs) == 1 && runs[0].Role == roleText && runs[0].Text == line
}

// hlAdvGo is a Go line about 80 columns wide whose unterminated string makes
// chroma's Go string rule backtrack until its own timeout: `x := "` and then
// repeated backslash pairs and a letter.
func hlAdvGo() string { return `x := "` + strings.Repeat(`\\a`, 25) }

// TestHighlightDeadlineGivesUpOnABlockedLexer is the deadline and the breaker of
// hlLexLine. Entry point: hlRuns with a fake lexer, which reaches hlLexLine. A
// lexer that blocks in Tokenise must give the line as one text run, before it is
// released and within a generous multiple of hlLexDeadline; the breaker must
// then hold its name; and a second call must not call the lexer at all. The
// control is a fast fake through the same path: it is lexed (keyword role), is
// not recorded, and is called again on a second call, so the blocked lexer's
// skipped second call is the breaker and not a lexer that never runs.
func TestHighlightDeadlineGivesUpOnABlockedLexer(t *testing.T) {
	const line = "if x"
	t.Cleanup(hlResetBreaker)

	blocked := &hlGateLexer{name: "test-blocked", gate: make(chan struct{})}
	t.Cleanup(func() { close(blocked.gate) }) // lets the abandoned goroutine end
	start := time.Now()
	got := hlRuns(blocked, line)
	if elapsed := time.Since(start); elapsed > 4*hlLexDeadline {
		t.Errorf("blocked lexer: hlRuns took %v, want within 4 x %v", elapsed, hlLexDeadline)
	}
	if !hlOnePlainRun(got, line) {
		t.Errorf("blocked lexer: got %v, want the line as one text run", got)
	}
	if !hlIsSlow("test-blocked") {
		t.Error("blocked lexer: the breaker did not record it")
	}
	if n := blocked.calls.Load(); n != 1 {
		t.Errorf("blocked lexer: %d Tokenise calls after the first line, want 1", n)
	}
	if got := hlRuns(blocked, line); !hlOnePlainRun(got, line) {
		t.Errorf("blocked lexer, second line: got %v, want the line as one text run", got)
	}
	if n := blocked.calls.Load(); n != 1 {
		t.Errorf("blocked lexer: %d Tokenise calls after the second line, want 1 (the breaker skips it)", n)
	}

	quick := &hlGateLexer{name: "test-quick"}
	for i := 1; i <= 2; i++ {
		got := hlRuns(quick, line)
		if len(got) != 1 || got[0].Role != roleSynKeyword || got[0].Text != line {
			t.Errorf("control, line %d: got %v, want the line lexed as one keyword run", i, got)
		}
		if n := quick.calls.Load(); int(n) != i {
			t.Errorf("control: %d Tokenise calls after line %d, want %d", n, i, i)
		}
	}
	if hlIsSlow("test-quick") {
		t.Error("control: the breaker recorded a fast lexer")
	}
}

// TestHighlightBreakerOnTheRealGoLexer is the same rule on chroma's own Go lexer
// and the line that costs it a chroma timeout per line. Entry point: hlRuns with
// the lexer from hlLexer. The adversarial line must come back plain within a
// generous multiple of hlLexDeadline and trip the Go lexer; afterwards a normal
// Go line is plain too, which is the breaker. Control, through the same path:
// before the trip and after hlResetBreaker, that same normal line is
// highlighted. The test is pinned to chroma v2.27.0's Go string rule, which
// backtracks on hlAdvGo's line: a chroma upgrade that fixes that rule makes the
// adversarial line lex in time and fails this test, which must then be revisited
// with another slow line or retired.
func TestHighlightBreakerOnTheRealGoLexer(t *testing.T) {
	t.Cleanup(hlResetBreaker)
	hlResetBreaker()
	goLex := hlLexer(hlPath, "a.go")
	const normal = "return nil"
	if !hlHasKeyword(hlRuns(goLex, normal)) {
		t.Fatal("control before the trip: the normal Go line was not highlighted")
	}

	adv := hlAdvGo()
	start := time.Now()
	got := hlRuns(goLex, adv)
	if elapsed := time.Since(start); elapsed > 4*hlLexDeadline {
		t.Errorf("adversarial line: hlRuns took %v, want within 4 x %v", elapsed, hlLexDeadline)
	}
	if !hlOnePlainRun(got, adv) {
		t.Errorf("adversarial line: got %d runs, want the line as one text run", len(got))
	}
	if !hlIsSlow(goLex.Config().Name) {
		t.Error("adversarial line: the Go lexer was not recorded")
	}
	if got := hlRuns(goLex, normal); !hlOnePlainRun(got, normal) {
		t.Errorf("normal line after the trip: got %v, want the line as one text run", got)
	}

	hlResetBreaker()
	if !hlHasKeyword(hlRuns(goLex, normal)) {
		t.Error("control after the reset: the normal Go line was not highlighted")
	}
}

// TestHighlightFenceOfSlowLinesStaysResponsive is the deadline seen through the
// reply layout. Entry point: mdLayout on a Go fence. A fence of several
// adversarial lines must lay out within a generous multiple of hlLexDeadline,
// not the several chroma timeouts those lines cost one by one, and every line
// must be drawn plain, as its text with two columns of indent. Control, through
// the same path: a fence of normal Go lines laid out before the trip has the
// keyword role.
func TestHighlightFenceOfSlowLinesStaysResponsive(t *testing.T) {
	t.Cleanup(hlResetBreaker)
	hlResetBreaker()
	g := NewGlyphs(true)

	normal, ok := mdLayout(fence("go", "return nil\nreturn err"), 100, g)
	if !ok || len(normal) != 2 {
		t.Fatalf("control fence: ok=%v rows=%d, want 2 rows", ok, len(normal))
	}
	for i, r := range normal {
		if !hlHasKeyword(r) {
			t.Errorf("control fence row %d has no keyword run", i)
		}
	}

	adv := hlAdvGo()
	body := strings.Join([]string{adv, adv + "b", adv + "c", adv + "d", adv + "e"}, "\n")
	start := time.Now()
	rows, ok := mdLayout(fence("go", body), 100, g)
	if elapsed := time.Since(start); elapsed > 4*hlLexDeadline {
		t.Errorf("fence layout took %v, want within 4 x %v", elapsed, hlLexDeadline)
	}
	lines := strings.Split(body, "\n")
	if !ok || len(rows) != len(lines) {
		t.Fatalf("fence: ok=%v rows=%d, want %d rows", ok, len(rows), len(lines))
	}
	for i, r := range rows {
		if got := hlToks(r); !toksEqual(got, []hlTok{{"  " + lines[i], roleText}}) {
			t.Errorf("fence row %d is not one plain text run: %v", i, got)
		}
	}
}

// TestHighlightBreakerIsPerLexer: a tripped lexer does not plain-out another.
// Entry points: hlRuns trips the Go lexer; mdLayout on a Python fence is the
// main assertion, and the Python fence must still be highlighted. The control
// goes through hlRuns, not mdLayout: the tripped Go lexer's own normal line is
// plain, so the trip is in force while Python is lexed. Both entry points reach
// hlLexLine, which holds the breaker.
func TestHighlightBreakerIsPerLexer(t *testing.T) {
	t.Cleanup(hlResetBreaker)
	hlResetBreaker()
	goLex := hlLexer(hlPath, "a.go")
	hlRuns(goLex, hlAdvGo())
	if !hlIsSlow(goLex.Config().Name) {
		t.Fatal("setup: the Go lexer was not tripped")
	}
	if got := hlRuns(goLex, "return nil"); hlHasKeyword(got) {
		t.Error("control: the tripped Go lexer still highlighted")
	}
	rows, ok := mdLayout(fence("python", "def f(a): return a"), 100, NewGlyphs(true))
	if !ok || len(rows) != 1 || !hlHasKeyword(rows[0]) {
		t.Errorf("python fence after the Go trip: ok=%v rows=%v, want a highlighted row", ok, rows)
	}
}

// hlLexJob is one text for hlLexConcurrently to lex, and the lexer to lex it
// with.
type hlLexJob struct {
	lx   chroma.Lexer
	text string
}

// hlLexConcurrently lexes every job with hlLexNow from workers goroutines at
// once, rounds times over, and returns each worker's last runs per job, indexed
// [worker][job]. The goroutines are released together, and each starts at a
// different job, so different jobs, and with them the lexers each one nests,
// are lexed at the same moment.
func hlLexConcurrently(jobs []hlLexJob, workers, rounds int) [][][]mdRun {
	results := make([][][]mdRun, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for w := 0; w < workers; w++ {
		results[w] = make([][]mdRun, len(jobs))
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < rounds; i++ {
				for k := range jobs {
					j := (k + w) % len(jobs)
					results[w][j] = hlLexNow(jobs[j].lx, jobs[j].text)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	return results
}

// TestHighlightConcurrentLexingOnOneLexer lexes with real lexers from several
// goroutines at once. Entry point: hlLexNow, through hlLexConcurrently, so the
// deadline and the breaker are out of the way and chroma's own concurrency is
// what runs. It is meant for -race.
//
// Through hlLexLine one lexer is not lexed twice at once by name: the UI waits
// for each line, and once a lexer misses the deadline the breaker keeps the UI
// from using it again. Two lexings of one lexer overlap through nesting
// instead: a lexer runs other lexers inside its own lexing, so a goroutine
// abandoned inside, say, the HTML lexer can be in the JavaScript lexer that
// HTML runs for a script element while the UI lexes a JavaScript line. The jobs
// cover that case on one line each, and also a Markdown fence holding Go code,
// which runs the Go lexer inside the Markdown lexer through chroma's
// UsingByGroup while other goroutines lex Go lines directly. hlLexLine never
// gets the fence, which spans lines; it is here because it drives the same
// nesting through the by-name route. The Ruby lines are lexed directly by
// every goroutine, and since no other test in the package uses Ruby, the
// goroutines also race its first compile when this test runs first on it.
//
// Each goroutine's last runs for every job must equal a serial lex of that job
// made afterwards. Positive controls on the serial lexes: the direct lines and
// the nested texts each have the keyword role, and the nested ones have it on
// a word only the nested lexer tags, so the nesting is shown to run and not to
// fail into plain runs.
func TestHighlightConcurrentLexingOnOneLexer(t *testing.T) {
	get := func(name string) chroma.Lexer {
		lx := lexers.Get(name)
		if lx == nil {
			t.Fatalf("no %s lexer", name)
		}
		return lx
	}
	ruby, goLex, md, html, js := get("ruby"), get("go"), get("markdown"), get("html"), get("javascript")
	jobs := []hlLexJob{
		{ruby, `def f(a) a + 1 end`},
		{ruby, `puts "hi" # c`},
		{ruby, `class A; end`},
		{ruby, `x = [1, 2, 3].map { |v| v * 2 }`},
		{goLex, `return nil`},
		{goLex, `x := func() {}`},
		{md, "```go\nreturn nil\n```"},
		{js, `var x = 1;`},
		{html, `<p><script>var x = 1;</script></p>`},
	}
	// keyword is, per job index, a word that a serial lex must draw in the
	// keyword role: the positive controls.
	keyword := map[int]string{0: "def", 4: "return", 5: "func", 6: "return", 7: "var", 8: "var"}

	const workers = 8
	results := hlLexConcurrently(jobs, workers, 5)
	for j, job := range jobs {
		want := hlLexNow(job.lx, job.text)
		if kw, ok := keyword[j]; ok && !hlHasKeywordText(want, kw) {
			t.Errorf("control: serial lex of %q has no keyword run %q: %v", job.text, kw, want)
		}
		for w := 0; w < workers; w++ {
			if !reflect.DeepEqual(results[w][j], want) {
				t.Errorf("worker %d, %q: got %v, want the serial %v", w, job.text, results[w][j], want)
			}
		}
	}
}

// hlHasKeywordText reports whether runs has a run in the keyword role whose
// text is word.
func hlHasKeywordText(runs []mdRun, word string) bool {
	for _, r := range runs {
		if r.Role == roleSynKeyword && r.Text == word {
			return true
		}
	}
	return false
}

// hlNilConfigLexer is a chroma.Lexer whose Config is nil. Once lexed, it gives
// the whole text as one keyword token, so a line that reached the lexer has the
// keyword role.
type hlNilConfigLexer struct{}

// Config returns nil, the case under test.
func (hlNilConfigLexer) Config() *chroma.Config { return nil }

// Tokenise returns the text as one keyword token.
func (hlNilConfigLexer) Tokenise(_ *chroma.TokeniseOptions, text string) (chroma.Iterator, error) {
	return chroma.Literator(chroma.Token{Type: chroma.Keyword, Value: text}), nil
}

// SetRegistry is a no-op; the fake looks up no other lexer.
func (l hlNilConfigLexer) SetRegistry(*chroma.LexerRegistry) chroma.Lexer { return l }

// SetAnalyser is a no-op; the fake does no content analysis.
func (l hlNilConfigLexer) SetAnalyser(func(text string) float32) chroma.Lexer { return l }

// AnalyseText scores every text zero.
func (hlNilConfigLexer) AnalyseText(string) float32 { return 0 }

// hlNoPanic calls f and reports whether it returned without panicking.
func hlNoPanic(f func()) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	f()
	return true
}

// TestHighlightNilConfigIsNoLexer: a lexer with no config is treated as no
// lexer. Entry points: hlUsable, which hlLexer passes every lexer it finds
// through (a fake with no config cannot be put in chroma's registry, so hlLexer
// itself cannot be handed one), and hlRuns, which reaches hlLexLine. hlUsable
// must return nil for the fake, and hlRuns must give the line as one text run;
// neither may panic. Controls through the same calls: hlUsable returns a real Go
// lexer unchanged, and hlRuns lexes a fake that has a config (keyword role).
func TestHighlightNilConfigIsNoLexer(t *testing.T) {
	t.Cleanup(hlResetBreaker)
	const line = "if x"
	var nilCfg hlNilConfigLexer

	var got chroma.Lexer
	if !hlNoPanic(func() { got = hlUsable(nilCfg) }) {
		t.Error("hlUsable panicked on a lexer with no config")
	} else if got != nil {
		t.Errorf("hlUsable: got %v for a lexer with no config, want nil", got)
	}
	goLex := lexers.Get("go")
	if hlUsable(goLex) != goLex {
		t.Error("control: hlUsable did not return the Go lexer")
	}

	var runs []mdRun
	if !hlNoPanic(func() { runs = hlRuns(nilCfg, line) }) {
		t.Error("hlRuns panicked on a lexer with no config")
	} else if !hlOnePlainRun(runs, line) {
		t.Errorf("hlRuns: got %v for a lexer with no config, want the line as one text run", runs)
	}
	ctl := hlRuns(&hlGateLexer{name: "test-has-config"}, line)
	if len(ctl) != 1 || ctl[0].Role != roleSynKeyword {
		t.Errorf("control: a fake with a config gave %v, want one keyword run", ctl)
	}
}

// TestHighlightUnsafeLexersMapToNoLexer: every lexer in hlUnsafeLexers maps to
// no lexer. Entry point: hlLexer, from a path and from an info string. Each
// path and word below is one that chroma itself maps to an excluded lexer (the
// test checks that through lexers.Match and lexers.Get first, so a chroma
// change that moves the name fails the test rather than passing it), and
// hlLexer must return nil for each. Every hlUnsafeLexers entry must be reached
// by at least one case, so a new entry needs a case. Controls through the same
// calls: an ordinary lexer, Go, maps from a path and from an info string, and
// Perl, which shares file extensions with Raku, still maps from its path.
func TestHighlightUnsafeLexersMapToNoLexer(t *testing.T) {
	cases := []struct {
		src  hlSource
		key  string
		name string
	}{
		{hlPath, "x.raku", "Raku"},
		{hlInfo, "raku", "Raku"},
		{hlInfo, "perl6", "Raku"},
		{hlInfo, "postgresql", "PostgreSQL SQL dialect"},
		{hlInfo, "postgres", "PostgreSQL SQL dialect"},
		{hlInfo, "materialize", "Materialize SQL dialect"},
		{hlPath, "x.svelte", "Svelte"},
		{hlInfo, "svelte", "Svelte"},
	}
	covered := map[string]bool{}
	for _, c := range cases {
		var raw chroma.Lexer
		if c.src == hlPath {
			raw = lexers.Match(c.key)
		} else {
			raw = lexers.Get(c.key)
		}
		if raw == nil || raw.Config().Name != c.name {
			t.Errorf("setup: chroma does not map %q to %s", c.key, c.name)
			continue
		}
		covered[c.name] = true
		if got := hlLexer(c.src, c.key); got != nil {
			t.Errorf("hlLexer(%q) = %s, want no lexer", c.key, got.Config().Name)
		}
	}
	for name := range hlUnsafeLexers {
		if !covered[name] {
			t.Errorf("no case reaches the excluded lexer %s", name)
		}
	}
	for _, c := range []struct {
		src       hlSource
		key, want string
	}{{hlPath, "x.go", "Go"}, {hlInfo, "go", "Go"}, {hlPath, "x.pl", "Perl"}} {
		if got := hlLexer(c.src, c.key); got == nil || got.Config().Name != c.want {
			t.Errorf("control: hlLexer(%q) = %v, want %s", c.key, got, c.want)
		}
	}
}

// hlLexGoroutines returns the IDs of the goroutines whose stack holds the
// goroutine hlLexLine starts and, inside it, hlGateLexer's Tokenise, from a dump
// of every goroutine's stack.
func hlLexGoroutines(t *testing.T) map[string]bool {
	t.Helper()
	ids := map[string]bool{}
	for _, g := range strings.Split(hlAllStacks(), "\n\n") {
		if strings.Contains(g, "hlLexLine.func1") && strings.Contains(g, "hlGateLexer).Tokenise") {
			ids[strings.SplitN(g, " ", 3)[1]] = true
		}
	}
	return ids
}

// hlAllStacks returns runtime.Stack for every goroutine, growing the buffer
// until the dump fits.
func hlAllStacks() string {
	buf := make([]byte, 1<<16)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// hlHasGoroutine reports whether a goroutine with the given ID is in a dump of
// every goroutine's stack.
func hlHasGoroutine(id string) bool {
	return strings.Contains("\n\n"+hlAllStacks(), "\n\ngoroutine "+id+" [")
}

// TestHighlightAbandonedLexGoroutineEnds: the goroutine hlLexLine abandons at
// the deadline ends once the lexer returns, rather than blocking on its send.
// Entry point: hlRuns with a fake lexer that blocks in Tokenise until its gate
// is closed, which reaches hlLexLine. After hlRuns has given the line up, the
// gate is closed and the stacks of every goroutine are polled until the
// abandoned one is gone, or the test fails after a few seconds. The goroutine
// is picked out by ID, as the one new goroutine with hlLexLine's goroutine and
// the fake's Tokenise on its stack, because other tests leave goroutines in
// hlLexLine that are still inside chroma's own lexers. Control, through the
// same poll: while the gate is still closed, that goroutine is found.
func TestHighlightAbandonedLexGoroutineEnds(t *testing.T) {
	t.Cleanup(hlResetBreaker)
	const line = "if x"
	before := hlLexGoroutines(t)
	g := &hlGateLexer{name: "test-exit", gate: make(chan struct{})}
	closed := false
	t.Cleanup(func() {
		if !closed {
			close(g.gate)
		}
	})
	if got := hlRuns(g, line); !hlOnePlainRun(got, line) {
		t.Fatalf("setup: got %v, want the line given up as one text run", got)
	}
	var id string
	for gid := range hlLexGoroutines(t) {
		if !before[gid] {
			if id != "" {
				t.Fatalf("setup: more than one new lexing goroutine (%s, %s)", id, gid)
			}
			id = gid
		}
	}
	if id == "" || !hlHasGoroutine(id) {
		t.Fatal("control: the abandoned goroutine was not found while the lexer is blocked")
	}

	close(g.gate)
	closed = true
	deadline := time.Now().Add(5 * time.Second)
	for hlHasGoroutine(id) {
		if time.Now().After(deadline) {
			t.Fatalf("goroutine %s is still running 5s after the lexer returned", id)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
