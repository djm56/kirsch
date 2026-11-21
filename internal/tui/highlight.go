package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/mattn/go-runewidth"
)

// This file is the syntax highlighting of ui-spec §3.2 and §4.1: the language
// mapping, the token-to-role mapping, and the per-modal state that says which
// modal line takes which lexer. Everything here produces mdRuns tagged with a
// role; none of it sees a Styles, so the reply cache and every drawing pass
// (live, flattened, identity) work on it unchanged.

// hlSource says what kind of string hlLexer is given.
type hlSource uint8

const (
	hlPath hlSource = iota // a file path or file name
	hlInfo                 // a fence's info string
)

// hlLexer maps a path or a fence info string to a chroma lexer, or nil when
// nothing maps. It is the one language mapping that reply fences, the content
// modal and the diff modal share.
//
// A path goes through lexers.Match, chroma's filename match, which covers
// extensions and names such as Makefile and Dockerfile. An info string goes
// through lexers.Get with its first word, case-insensitive, after a word in
// hlInfoAlias has been replaced by the word it stands for. Get falls back to a
// filename match when the word is no name or alias, which would map "main.go"
// written as an info string to Go, so a result counts only when the word, as
// replaced, really is one of the lexer's names or aliases. "Makefile" as an info
// string is the Makefile lexer's own name, so it maps. Chroma's content analysis
// and its fallback lexer are never used: unmapped input stays plain. Whatever
// either lookup finds goes through hlUsable, so a lexer in hlUnsafeLexers, or one
// with no config, maps to nil too.
func hlLexer(src hlSource, key string) chroma.Lexer {
	switch src {
	case hlPath:
		if key == "" {
			return nil
		}
		return hlUsable(lexers.Match(key))
	case hlInfo:
		f := strings.Fields(key)
		if len(f) == 0 {
			return nil
		}
		word := strings.ToLower(f[0])
		if to, ok := hlInfoAlias[word]; ok {
			word = to
		}
		lx := hlUsable(lexers.Get(word))
		if lx == nil {
			return nil
		}
		cfg := lx.Config()
		if strings.ToLower(cfg.Name) == word {
			return lx
		}
		for _, a := range cfg.Aliases {
			if strings.ToLower(a) == word {
				return lx
			}
		}
	}
	return nil
}

// hlInfoAlias maps a fence info word to the word hlLexer looks up in its place.
// Chroma maps "postgresql" and "postgres" to its PostgreSQL SQL dialect lexer,
// which is in hlUnsafeLexers, so hlLexer looks them up as "sql", chroma's generic
// SQL lexer, which is not. It applies to info strings only: a path is matched by
// lexers.Match, and the dialect lexer has no file pattern. The other names of an
// excluded lexer are not listed, so they stay plain.
var hlInfoAlias = map[string]string{
	"postgresql": "sql",
	"postgres":   "sql",
}

// hlUnsafeLexers names, by Config().Name, the chroma lexers that hlLexer never
// returns, because lexing with them writes state that every lexing of the same
// lexer shares. hlLexLine can leave a lexing running past hlLexDeadline while the
// UI lexes again, so two lexings of one lexer can overlap, and with these lexers
// that is a data race on the lexer's rule table.
//
// Raku is the writer: in chroma v2.27.0, its replaceRule mutator
// (lexers/raku.go) assigns new rules into state.Rules, which is the lexer's own
// rule table, not a copy, while it lexes. The other entries are the lexers that
// hand text on one line to a sub-lexer named by the text itself, through
// chroma's UsingByGroup, and so can run Raku inside their own lexing: a
// PostgreSQL or Materialize "DO LANGUAGE raku $$ … $$" block, and a Svelte
// <script lang="raku"> element. A UsingByGroup lexer that cannot reach Raku
// through hlLexLine, which lexes one line at a time, is not listed: one whose
// pattern needs more than one line, such as Markdown's fences, or one whose
// pattern admits only other language names.
//
// The set is tied to chroma v2.27.0: a chroma upgrade must re-check it. The
// re-check covers every emitter that looks up a lexer by a name taken from the
// text, not only UsingByGroup. rstCodeBlock in chroma's lexers/rst.go is one: it
// calls Get on the language word in a code-block directive.
// TestHighlightUnsafeLexersReviewedChromaVersion fails when the chroma module
// version differs from the one this list was last reviewed against.
var hlUnsafeLexers = map[string]bool{
	"Raku":                    true,
	"PostgreSQL SQL dialect":  true,
	"Materialize SQL dialect": true,
	"Svelte":                  true,
}

// hlUsable returns lx, or nil when lx is nil, has no config, or is named in
// hlUnsafeLexers. hlLexer passes every lexer it finds through it.
func hlUsable(lx chroma.Lexer) chroma.Lexer {
	if lx == nil {
		return nil
	}
	cfg := lx.Config()
	if cfg == nil || hlUnsafeLexers[cfg.Name] {
		return nil
	}
	return lx
}

// hlRole maps a chroma token type to the role it is drawn in, by the token
// hierarchy. Keyword types and class names are checked before the broader
// keyword and name groups. A name constant is drawn as a keyword. In the lexers
// that tag words such as true or NULL as names rather than keywords, those are
// keyword constants. In some lexers, for example Ruby's capitalised names and
// all-caps names in Rust, Perl and Crystal, NameConstant also covers names that
// are not constants. Those also draw as keywords. Builtin and function names draw
// as accent. Names not covered above, exceptions among them, are text. A
// preprocessor directive is a comment in chroma's hierarchy but is not drawn
// as one: it is code, and a muted directive would read as commented out, so it
// falls to text with the operators.
func hlRole(t chroma.TokenType) mdRole {
	switch {
	case t == chroma.KeywordType, t == chroma.NameClass:
		return roleSynType
	case t.InCategory(chroma.Keyword), t == chroma.NameConstant:
		return roleSynKeyword
	case t.InSubCategory(chroma.NameBuiltin), t.InSubCategory(chroma.NameFunction):
		return roleAccent
	case t.InSubCategory(chroma.LiteralString):
		return roleSynString
	case t.InCategory(chroma.Literal):
		return roleSynNumber
	case t.InSubCategory(chroma.CommentPreproc):
		return roleText
	case t.InCategory(chroma.Comment):
		return roleMuted
	}
	return roleText
}

// hlMaxLineCols is the longest line, in terminal columns, that is lexed. A
// longer line is drawn without syntax colours wherever highlighting applies:
// reply fences and both modals. Lexing runs chroma's regular expressions over
// the whole line, so its cost grows with the line; the cutoff caps that growth.
// It does not cap a rule that backtracks catastrophically, which can be slow on
// a line far shorter than the cutoff: hlLexDeadline and the breaker behind it
// cap that. Whichever of these fires, the line is drawn without syntax colours:
// the same text, and a diff's +/- marker keeps its colour.
const hlMaxLineCols = 512

// hlLexDeadline is how long hlLexLine waits for one line to be lexed before it
// gives the line up as plain. The value rests on these measurements. A lexer
// compiles its rules on first use, once per process, and the first lex of a
// short line took a few milliseconds at most for the lexers of the common fence
// languages. The steady-state lex of the worst ordinary line the highlight
// benchmark builds, a minified script at the cutoff, took under a millisecond.
// The value is well above each of them, so ordinary content, a lexer's first
// use included, does not miss it on a loaded machine or under the race
// detector. It is also well below chroma's per-rule timeout, which a
// catastrophically backtracking rule can run to on every line it is given. A
// lexer that misses the deadline costs this wait once: the breaker then keeps
// hlLexLine from lexing with it again, so no later line waits for it at all.
const hlLexDeadline = 50 * time.Millisecond

// hlSlow is the circuit breaker's state: the names of the lexers that have
// missed hlLexDeadline in this process. A name is added when a lexer first
// misses and is never removed while the program runs.
var hlSlow = struct {
	mu    sync.Mutex
	names map[string]bool
}{names: map[string]bool{}}

// hlIsSlow reports whether the lexer called name has missed hlLexDeadline.
func hlIsSlow(name string) bool {
	hlSlow.mu.Lock()
	defer hlSlow.mu.Unlock()
	return hlSlow.names[name]
}

// hlMarkSlow records that the lexer called name missed hlLexDeadline.
func hlMarkSlow(name string) {
	hlSlow.mu.Lock()
	defer hlSlow.mu.Unlock()
	hlSlow.names[name] = true
}

// hlLexable reports whether line is short enough to be lexed: no wider than
// hlMaxLineCols columns.
func hlLexable(line string) bool {
	if len(line) <= hlMaxLineCols {
		return true // a column takes at least one byte
	}
	return runewidth.StringWidth(line) <= hlMaxLineCols
}

// hlRuns returns one line's runs, each tagged with its role. A line with no
// lexer, or one wider than hlMaxLineCols, is the whole line as one text run;
// any other line is lexed by hlLexLine, which gives the same plain result when
// the lexer misses hlLexDeadline or has missed it before. In every plain case
// the drawn text is the line, without syntax colours.
func hlRuns(lx chroma.Lexer, line string) []mdRun {
	if lx == nil || !hlLexable(line) {
		return []mdRun{{Text: line}}
	}
	return hlLexLine(lx, line)
}

// hlLexLine lexes one line on its own, whatever its length, and returns its
// runs, each tagged with its role. With no lexer it is the whole line as one
// text run. It also returns the whole line as one text run when the lexer has
// no config, fails, panics, or returns tokens whose text is not the line, so the
// drawn text is the line.
//
// The lexing runs in a goroutine of its own and hlLexLine waits at most
// hlLexDeadline for it. When the timer fires, a result the goroutine has
// already sent is still taken: a select with more than one case ready picks
// among them at random, so the timer case looks at the channel once more,
// without waiting, before it gives up. Otherwise the line is returned as one
// text run, the lexer's name is recorded in hlSlow, and the goroutine is left to
// finish on its own with its result discarded; it sends on a buffered channel,
// so it never blocks. Once a lexer's name is in hlSlow, hlLexLine returns the
// plain line for that lexer without starting a goroutine, so a lexer costs at
// most one deadline's wait and one abandoned goroutine per process. A line
// lexed in time is returned as hlLexNow returns it.
func hlLexLine(lx chroma.Lexer, line string) []mdRun {
	if lx == nil || line == "" {
		return []mdRun{{Text: line}}
	}
	cfg := lx.Config()
	if cfg == nil {
		return []mdRun{{Text: line}}
	}
	name := cfg.Name
	if hlIsSlow(name) {
		return []mdRun{{Text: line}}
	}
	done := make(chan []mdRun, 1)
	go func() { done <- hlLexNow(lx, line) }()
	timer := time.NewTimer(hlLexDeadline)
	defer timer.Stop()
	select {
	case runs := <-done:
		return runs
	case <-timer.C:
		select {
		case runs := <-done:
			return runs
		default:
		}
		hlMarkSlow(name)
		return []mdRun{{Text: line}}
	}
}

// hlLexNow lexes one line on the calling goroutine, with no deadline, and
// returns its runs, each tagged with its role. It returns the whole line as one
// text run when the lexer fails, panics, or returns tokens whose text is not the
// line. The recover sits here so that it holds when hlLexLine runs this in a
// goroutine of its own, where an unrecovered panic would end the program.
func hlLexNow(lx chroma.Lexer, line string) (runs []mdRun) {
	plain := []mdRun{{Text: line}}
	if lx == nil || line == "" {
		return plain
	}
	defer func() {
		if recover() != nil {
			runs = plain
		}
	}()
	it, err := lx.Tokenise(nil, line)
	if err != nil {
		return plain
	}
	var b strings.Builder
	for t := it(); t != chroma.EOF; t = it() {
		if t.Value == "" {
			continue
		}
		b.WriteString(t.Value)
		runs = append(runs, mdRun{Text: t.Value, mdStyle: mdStyle{Role: hlRole(t.Type)}})
	}
	// Some lexers end the input with a newline of their own.
	if got := b.String(); got != line {
		if got != line+"\n" || len(runs) == 0 {
			return plain
		}
		last := &runs[len(runs)-1]
		last.Text = strings.TrimSuffix(last.Text, "\n")
		if last.Text == "" {
			runs = runs[:len(runs)-1]
		}
	}
	return runs
}

// hlSplitNumbered splits a read_file line into its number prefix and its text.
// A line is numbered when it begins with one or more ASCII digits followed by
// at least N spaces, where N is 4 minus the digit count mod 4. The prefix is
// the digits and exactly the first N spaces; the text is everything after,
// whatever it starts with. ok is false for any other line.
func hlSplitNumbered(line string) (prefix, text string, ok bool) {
	d := 0
	for d < len(line) && line[d] >= '0' && line[d] <= '9' {
		d++
	}
	if d == 0 {
		return "", "", false
	}
	n := 4 - d%4
	if len(line) < d+n || strings.Trim(line[d:d+n], " ") != "" {
		return "", "", false
	}
	return line[:d+n], line[d+n:], true
}

// ToolTruncMarker is the line the tools put between the kept head and tail of
// output they cut (internal/tool/tool.go, in Truncate). It is exported so that a
// test in internal/app, which may import both packages, pins it to Truncate's
// output; internal/tui must not import internal/tool.
const ToolTruncMarker = "‹… output truncated …›"

// hlLine is how one line of a /diff modal is drawn: header lines are never
// lexed, and a code line is lexed with lex, or not at all when lex is nil.
type hlLine struct {
	header bool
	lex    chroma.Lexer
}

// modalHL is a modal's highlighting state, worked out once per open.
type modalHL struct {
	lex   chroma.Lexer // one lexer for the whole modal: a read_file card or a patch approval diff
	lines []hlLine     // per line, for /diff

	// runs holds, by index in the modal's Lines, the lexed runs of each line
	// lexed so far: the line's text without its number prefix or diff marker,
	// before the line is cut into display rows. A nil entry is a line wider than hlMaxLineCols,
	// which is drawn plain. Redraws and scrolling read it rather than lex again.
	runs map[int][]mdRun
}

// highlight returns the modal's highlighting state, working it out on first use
// and keeping it for the rest of the modal's life: the modal redraws on every
// key, and the sections of a /diff are found by a pass over every line, so the
// pass must not run per drawn line. trunc is the glyph table's truncation glyph,
// which starts the trailing marker /diff adds when its output was cut.
func (md *ModalState) highlight(trunc string) *modalHL {
	if md.hl != nil {
		return md.hl
	}
	md.hlComputes++
	h := &modalHL{runs: map[int][]mdRun{}}
	switch {
	case md.Kind == ModalContent && md.ReadFile:
		h.lex = hlLexer(hlPath, md.Path)
	case md.Kind == ModalDiff && md.WorkTree:
		h.lines = hlSections(md.Lines, trunc)
	case md.Kind == ModalDiff:
		// A patch approval diff names its file in the title; for a rename
		// ("old → new") the language is the new path's.
		h.lex = hlLexer(hlPath, DiffTitlePath(md.Title))
	}
	md.hl = h
	return h
}

// DiffTitlePath returns the path a patch approval diff modal takes its language
// from: the title itself, or for a rename titled "old → new" the new path after
// the last " → ". It is exported so that a test in internal/app pins it against
// the title internal/app builds.
func DiffTitlePath(title string) string {
	if i := strings.LastIndex(title, " → "); i >= 0 {
		return title[i+len(" → "):]
	}
	return title
}

// hlSections assigns every line of a /diff its lexer and whether it is a header
// line. A file's header runs from its "diff --git" line to its first "@@" line,
// so after the first "@@" a line beginning "+++" or "---" is a code line. The
// lexer comes from the b/ path of the section's "diff --git" line. A truncation
// marker ends highlighting until the next "diff --git".
func hlSections(lines []string, trunc string) []hlLine {
	out := make([]hlLine, len(lines))
	var lex chroma.Lexer
	inHeader := false
	for i, ln := range lines {
		switch {
		case strings.HasPrefix(ln, "diff --git "):
			inHeader = true
			lex = hlLexer(hlPath, hlDiffBPath(ln))
			out[i] = hlLine{header: true}
		case hlIsTruncMarker(ln, trunc):
			inHeader, lex = false, nil
		case inHeader && strings.HasPrefix(ln, "@@"):
			inHeader = false
			out[i] = hlLine{lex: lex}
		case inHeader:
			out[i] = hlLine{header: true}
		default:
			out[i] = hlLine{lex: lex}
		}
	}
	return out
}

// hlIsTruncMarker reports whether ln is a truncation marker: the tools' own
// marker line, or the "<trunc> truncated" line that /diff appends when its
// output hit the cap, which may be followed by the cap's size.
func hlIsTruncMarker(ln, trunc string) bool {
	return ln == ToolTruncMarker || strings.HasPrefix(ln, trunc+" truncated")
}

// hlDiffBPath returns the b/ path of a "diff --git a/<p> b/<p>" line, or "" when
// the line does not parse with exactly one " b/" split. Quoted paths, git's
// mnemonic prefixes and the no-prefix setting all fail the "a/" test, and a path
// that itself contains " b/" fails the count, so those sections are plain.
func hlDiffBPath(line string) string {
	rest, ok := strings.CutPrefix(line, "diff --git a/")
	if !ok || strings.Count(rest, " b/") != 1 {
		return ""
	}
	return rest[strings.Index(rest, " b/")+len(" b/"):]
}

// lineRuns returns the lexed runs of text, the code part of line idx, lexing it
// with lx only the first time line idx is asked for in this open and reading the
// cache after that. ok is false when text is wider than hlMaxLineCols, so the
// line is drawn the plain way. hlLexes counts the lexing calls; a test reads it.
func (md *ModalState) lineRuns(h *modalHL, idx int, lx chroma.Lexer, text string) (runs []mdRun, ok bool) {
	if runs, hit := h.runs[idx]; hit {
		return runs, runs != nil
	}
	if hlLexable(text) {
		md.hlLexes++
		runs = hlLexLine(lx, text)
	}
	h.runs[idx] = runs
	return runs, runs != nil
}

// contentRuns is the highlighted form of one body line of a read_file content
// modal, uncut: the number prefix muted, then the lexed runs of the text after it.
// idx is the line's index in md.Lines. The runs laid end to end are raw, so the
// caller cuts them to rows by byte offset into raw (see runsSlice). ok is false
// when the line is drawn the plain way: the modal is not a read_file view, its
// language is unmapped, the line is not a numbered one, or its text is wider than
// hlMaxLineCols.
func (md *ModalState) contentRuns(trunc, raw string, idx int) (runs []mdRun, ok bool) {
	h := md.highlight(trunc)
	if h.lex == nil {
		return nil, false
	}
	prefix, text, numbered := hlSplitNumbered(raw)
	if !numbered {
		return nil, false
	}
	code, ok := md.lineRuns(h, idx, h.lex, text)
	if !ok {
		return nil, false
	}
	return append([]mdRun{{Text: prefix, mdStyle: mdStyle{Role: roleMuted}}}, code...), true
}

// diffIsCode reports whether line idx of a diff modal is a code line: not a header
// line, and beginning with "+", "-" or a space. Header status decides before the
// first character does, so in /diff a "---" or "+++" line between a "diff --git"
// line and the first "@@" is not a code line. A patch approval diff has no header
// lines, so there the first character alone decides. A line after a truncation
// marker is not a header line either (hlSections), so it is a code line by its
// first character.
func (md *ModalState) diffIsCode(trunc, raw string, idx int) bool {
	if md.WorkTree {
		if h := md.highlight(trunc); idx < len(h.lines) && h.lines[idx].header {
			return false
		}
	}
	return raw != "" && (raw[0] == '+' || raw[0] == '-' || raw[0] == ' ')
}

// diffRuns is the highlighted form of one line of a diff modal, uncut: the marker
// in its colour, then the lexed runs of the text after it. idx is the line's index
// in md.Lines. The runs laid end to end are raw. ok is false when the line is
// drawn the plain way: a header, @@ or context line, a line with no lexer, or a
// line whose text after the marker is wider than hlMaxLineCols.
func (md *ModalState) diffRuns(trunc, raw string, idx int) (runs []mdRun, ok bool) {
	h := md.highlight(trunc)
	lx := h.lex
	if md.WorkTree {
		if idx >= len(h.lines) || h.lines[idx].header {
			return nil, false
		}
		lx = h.lines[idx].lex
	}
	if lx == nil {
		return nil, false
	}
	var marker mdRole
	switch {
	case strings.HasPrefix(raw, "+"):
		marker = roleSuccess
	case strings.HasPrefix(raw, "-"):
		marker = roleError
	default:
		return nil, false
	}
	code, ok := md.lineRuns(h, idx, lx, raw[1:])
	if !ok {
		return nil, false
	}
	return append([]mdRun{{Text: raw[:1], mdStyle: mdStyle{Role: marker}}}, code...), true
}
