package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/rivo/uniseg"
)

// Placeholder is the composer's empty-state text. ui-spec §2.
const Placeholder = "Ask anything (Enter to send, /help for help)"

// Composer wraps the textarea with Kirsch's own prompt and hint rows.
type Composer struct {
	ta         textarea.Model
	Disabled   bool   // mirrors Busy; the only thing Busy does here
	Focused    bool   // the composer holds input capture
	Onboarding bool   // nothing has been said yet: show the placeholder
	Hint       string // dim inline hint for an unknown /command, §6
}

func newComposer() Composer {
	ta := textarea.New()
	ta.Placeholder = Placeholder
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.Focus()
	return Composer{ta: ta}
}

// Value returns the composer's contents.
func (c *Composer) Value() string { return c.ta.Value() }

// SetValue replaces the composer's contents.
func (c *Composer) SetValue(s string) { c.ta.SetValue(s) }

// Reset clears the composer and any pending hint.
func (c *Composer) Reset() {
	c.ta.Reset()
	c.Hint = ""
}

// InsertString inserts literal text at the cursor. Used for bracketed paste,
// where the content must never be interpreted as keys. ui-spec §7.2.
func (c *Composer) InsertString(s string) { c.ta.InsertString(s) }

// cursor reports the textarea's insertion point as a logical (row, column)
// pair, where row indexes the lines of Value() and column counts runes into
// that line.
//
// bubbles v1.0.0 exposes the row through Line() but publishes no getter for the
// column: the field is unexported and the only window onto it is LineInfo,
// which describes the *soft-wrapped* row the textarea computes internally.
// StartColumn is where that wrapped row begins within the logical line and
// ColumnOffset is the cursor's distance into it, so the sum re-adds the wrap and
// lands back on the logical line.
//
// The sum holds at any wrap width, but not because the wrapped rows partition
// the line: textarea's wrap() appends trailing space runes of its own
// (textarea.go:1452 and its three siblings), so the grid it returns holds more
// runes than the line went in with. The reason is narrower and stronger.
// LineInfo walks that grid accumulating `counter`, then returns StartColumn =
// counter and ColumnOffset = m.col - counter (textarea.go:840-847). counter
// appears once with each sign, so it cancels and the sum is m.col by
// construction — whatever wrap() did to get there. The early-return branch at
// textarea.go:827 sets StartColumn = m.col and ColumnOffset = 0, which sums to
// m.col as well.
//
// The composer never calls SetWidth, so the textarea wraps at its own default
// of 40 columns while the rendering below wraps not at all; going through the
// logical index is what stops those two widths having to agree.
func (c *Composer) cursor() (row, col int) {
	li := c.ta.LineInfo()
	return c.ta.Line(), li.StartColumn + li.ColumnOffset
}

// contentLines is the composer's height in rows: its content clamped to 1..5,
// plus a row for the hint when one is showing.
//
// The hint occupies a row *of* the composer region rather than becoming a fifth
// region, because row order is always header, transcript, status, composer.
func (c *Composer) contentLines() int {
	n := clamp(strings.Count(c.ta.Value(), "\n")+1, 1, 5)
	if c.Hint != "" {
		n++
	}
	return n
}

// rows renders the composer to exactly lay.ComposerH lines.
//
// Three empty-composer states, which is what the screens show and what §2's
// bare "placeholder when empty" does not distinguish:
//
//   - onboarding (nothing said yet): the placeholder      — screen 01
//   - focused, with a transcript:    the cursor           — screens 07, 08, 10
//   - not focused (an approval or modal holds capture): neither — screens 03-06
//
// A placeholder that persists behind a pending approval reads as an invitation
// to type into a composer that is deliberately not accepting input.
func (c *Composer) rows(lay Layout, sty *Styles, g Glyphs, busy bool) []string {
	h := lay.ComposerH
	out := make([]string, 0, h)

	prompt := sty.Accent(">")
	if busy {
		prompt = sty.Dim(">")
	}

	value := c.ta.Value()
	var lines []string
	switch {
	case value == "" && c.Onboarding:
		lines = []string{sty.Dim(Placeholder)}
	case value == "" && c.Focused:
		lines = []string{sty.Dim(g.Caret)}
	case value == "":
		lines = []string{""}
	default:
		raw := strings.Split(value, "\n")
		row, col := -1, 0
		if c.Focused {
			row, col = c.cursor()
			row = clamp(row, 0, len(raw)-1)
		}
		lines = make([]string, len(raw))
		for i, l := range raw {
			if i != row {
				lines[i] = sty.Text(l)
				continue
			}
			// The caret is drawn *into* the line at the cursor rather than
			// appended to the end of it, and it takes the cells of the
			// character it is on instead of a cell of its own. Both halves are
			// load-bearing and each fixes a defect the other left behind:
			//
			//   - Appending was the original defect. The textarea moved its
			//     cursor on every arrow key and went on inserting at the new
			//     position, but the only place the glyph could appear was after
			//     the last character, so the caret sat still while the text it
			//     claimed to point at moved out from under it.
			//   - Inserting fixed the position and broke the width. A caret
			//     given its own cell pushes every following character one
			//     column right, so the row read as `te▌st` with a visible gap —
			//     invisible at end-of-input, where nothing follows it, and wrong
			//     everywhere else.
			//
			// Overstriking is the only form that is right in both: the row
			// measures exactly as wide as its text at every position except
			// end-of-input, where there is nothing to stand in for and the caret
			// is appended. That one exception is why the end-of-input case still
			// renders byte-for-byte as it always has.
			//
			// What is overstruck is a grapheme cluster and never a rune — see
			// splitAtCursor — and which glyph is drawn depends on what it
			// covers — see caretFor. Both are corrections to a first version of
			// this that took the rune at the cursor and drew a fixed glyph over
			// it: that one broke the width of every cluster spanning more than
			// one rune, and disappeared entirely wherever the text already held
			// the glyph.
			at := clamp(col, 0, utf8.RuneCountInString(l))
			head, cluster, tail := splitAtCursor(l, at)
			lines[i] = styled(sty.Text, head) +
				sty.Dim(overstrike(caretFor(g, cluster), cluster)) +
				styled(sty.Text, tail)
		}
	}

	// First row carries the prompt; when a turn is live the right edge carries
	// the disabled hint instead of the caret.
	first := prompt + " " + lines[0]
	if busy {
		hint := "(input disabled, Esc cancels)"
		plain := "> " + stripFirst(value)
		first = prompt + " " + sty.Dim(stripFirst(value))
		if cellWidth(plain)+cellWidth(hint)+1 <= lay.ContentW {
			gap := lay.ContentW - cellWidth(plain) - cellWidth(hint)
			first += strings.Repeat(" ", gap) + sty.Dim(hint)
		}
	}
	out = append(out, first)
	for _, l := range lines[1:] {
		if len(out) >= h {
			break
		}
		out = append(out, "  "+l)
	}
	if c.Hint != "" && len(out) < h {
		out = append(out, sty.Dim("  "+c.Hint))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out[:h]
}

// splitAtCursor cuts line into the text before the cursor's grapheme cluster,
// that cluster, and the text after it. An empty cluster means at is past the
// last one: end of input.
//
// at is a rune index because a rune index is what the textarea reports, but a
// rune is not what the terminal draws, and the gap between the two is the whole
// reason this function exists rather than a slice expression.
//
// A cluster is what the reader sees as one character, and it is routinely
// several runes: `👍🏽` is an emoji plus a skin-tone modifier, `👨‍👩` is two emoji
// joined by U+200D, `é` may arrive as `e` + U+0301, and `❤️` is a heart plus the
// U+FE0F variation selector. Replacing the single rune at index at therefore
// replaces a *piece* of a character, and that breaks the cluster: the runes
// left behind no longer join, so they re-render at their standalone widths —
// `👨`(2) beside
// `👩`(2) where the joined pair was 2 — and the row grows by cells no amount of
// padding can take back. Cutting on the boundary is what keeps the caret's cell
// the same cell the reader sees.
//
// It also fixes the measurement, which was wrong even where the cluster
// survived. The caller used to measure the pad from a one-rune string, and a
// lone rune has no cluster to be measured in: U+1F3FD, the skin-tone modifier,
// measures 2 on its own and contributes 0 inside `👍🏽`, so the cursor there
// earned two cells of padding for a character that was already as wide as the
// caret. Measuring the cluster is measuring the thing that occupies the cells.
func splitAtCursor(line string, at int) (head, cluster, tail string) {
	state, b, n := -1, 0, 0
	for b < len(line) {
		c, _, _, next := uniseg.FirstGraphemeClusterInString(line[b:], state)
		state = next
		w := utf8.RuneCountInString(c)
		if at < n+w {
			return line[:b], c, line[b+len(c):]
		}
		b, n = b+len(c), n+w
	}
	return line, "", ""
}

// caretFor returns the glyph to draw over cluster: the caret, or its stand-in
// where the caret would be invisible.
//
// Overstriking loses the cursor entirely whenever the glyph drawn and the text
// it covers are the same string — the row comes out byte-identical to the plain
// text, and there is no cursor on screen at all. In ASCII the caret is `_`, and
// `parse_slash`, `snake_case_name` and `my_file.go` are exactly what a coding
// agent's composer gets typed full of. ASCII is not an exotic mode either:
// unicodeLocale reports false whenever LC_ALL, LC_CTYPE and LANG are all unset,
// which is the ordinary condition in `docker run`, in a systemd unit, under
// cron and on many CI runners.
//
// No fixed glyph fixes it, and picking a rarer one only hides it. This is a
// free-text field, so every printable ASCII character is something the user can
// type — `#` opens a comment and a markdown heading, `[` opens a link and a
// slice index — and whichever character is chosen, some ordinary input is that
// character. Swapping one glyph for another moves the defect onto text the
// tests no longer happen to cover, which looks like a fix and is not one.
//
// So the glyph is chosen against what it covers instead of being fixed. The
// stand-in is drawn precisely when the primary would vanish, and differs from
// the primary by construction, so the glyph drawn is never equal to the cluster
// it replaces — for every input, which is the property no fixed glyph has.
//
// What it does not promise is that the caret is the only one of its kind on the
// row: `a_b` with the cursor on the `_` renders `a#b`, and a line already
// carrying a `#` elsewhere shows two. Uniqueness is unreachable for any glyph
// in a field that accepts arbitrary text, it is no worse here than under the
// scheme this replaces, and an ambiguous cursor is a different thing from an
// absent one.
func caretFor(g Glyphs, cluster string) string {
	if cluster == g.Caret {
		return g.CaretAlt
	}
	return g.Caret
}

// overstrike returns caret padded to occupy exactly the cells of cluster, the
// grapheme cluster the cursor is sitting on. An empty cluster is end of input:
// there is nothing to stand in for, so the caret is returned unpadded and the
// row is one cell wider than its text.
//
// A caret is one cell and a cluster may be two, so replacing one with the other
// is only width-preserving for the narrow case. A CJK ideograph under the
// cursor would otherwise pull every following character one column left, which
// is the same class of defect as the gap this replaced, just in the other
// direction. The padding is a plain space and never a second glyph: two carets
// would read as two cursors.
//
// The width comes from cellWidth — the same runewidth call the layout, the
// wrapper and the frame's own width bound are measured with — and deliberately
// not from uniseg's width, which disagrees with it on some clusters (`❤️` is one
// cell to runewidth and two to uniseg). The caret has to agree with whatever
// the rest of the frame is measured by rather than with whatever is nearest to
// correct; disagreeing is how the composer row stops matching the columns the
// layout budgeted for it.
//
// A cluster can contribute no columns at all — a combining mark, a variation
// selector or a ZWJ stranded at the start of a line is a zero-width cluster of
// its own — so the subtraction can go negative; the guard leaves the caret
// alone there rather than truncating it. That row is one cell wide where its
// text is none, which is the one place the width rule bends: a zero-width
// cursor is not a cursor.
func overstrike(caret, cluster string) string {
	n := cellWidth(cluster) - cellWidth(caret)
	if n <= 0 {
		return caret
	}
	return caret + strings.Repeat(" ", n)
}

// styled applies st to s, leaving the empty string alone.
//
// Splitting a line at the cursor produces an empty half whenever the cursor is
// at either end, and lipgloss renders an empty string as a bare open/close SGR
// pair: two escapes that change no cell but do change the bytes of the frame.
// Skipping them is what keeps a caret at end-of-input rendering exactly as it
// did when it was appended, which is the case every screen grid records.
func styled(st Style, s string) string {
	if s == "" {
		return ""
	}
	return st(s)
}

func stripFirst(v string) string {
	if v == "" {
		return ""
	}
	return strings.SplitN(v, "\n", 2)[0]
}

// parseSlash recognises a slash command.
//
// The grammar is deliberately strict: a single line beginning with `/`.
// Multi-line content starting with `/` is an ordinary message, so a pasted
// diff or stack trace is never mistaken for a command. ui-spec §6.
func parseSlash(v string) (cmd, args string, ok bool) {
	if !strings.HasPrefix(v, "/") || strings.Contains(v, "\n") {
		return "", "", false
	}
	body := strings.TrimSpace(v[1:])
	if body == "" {
		return "", "", false
	}
	parts := strings.SplitN(body, " ", 2)
	if len(parts) == 2 {
		return parts[0], strings.TrimSpace(parts[1]), true
	}
	return parts[0], "", true
}

// SlashCommands is the v0.1 command set. ui-spec §6.
//
// /exit is an alias of /quit, not a second behaviour: the two share one arm in
// runSlash. Both are listed because quitting has no key binding any more, and a
// session-ending command is the one command a reader should not have to guess
// the spelling of.
var SlashCommands = []string{
	"help", "status", "diff", "files", "approvals", "new", "compact", "quit", "exit",
}

// DebugCommands are Milestone 1 scaffolding: they exist so a real repository
// can be read from inside the TUI before the model drives tools, and they are
// removed in M3. Tab-completable alongside the real set, but labelled (debug)
// in the help overlay.
var DebugCommands = []string{
	"/read", "/ls", "/search", "/gitstatus", "/gitdiff",
}

// completeSlash completes a unique prefix, returning the completion and whether
// exactly one candidate matched.
func completeSlash(prefix string) (string, bool) {
	var match string
	n := 0
	candidates := append(append([]string{}, SlashCommands...), trimSlashes(DebugCommands)...)
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			match = c
			n++
		}
	}
	return match, n == 1
}

func trimSlashes(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.TrimPrefix(s, "/")
	}
	return out
}
