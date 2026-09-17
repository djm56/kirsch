package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Every layout calculation in the package goes through these primitives. They
// measure in display cells, never in bytes or runes: a rune is not a cell, CJK
// and emoji are two, combining marks are zero, and several ASCII glyph
// fallbacks are multi-cell strings. Assuming otherwise corrupts every border on
// the screen. ui-spec §2.1.

// sgrPrefixLen reports the byte length of an SGR sequence starting at s[0], or
// 0 if there is not one there.
//
// Styled and unstyled text flow through the same primitives, so every one of
// them has to skip escapes rather than count them. Measuring a styled string
// naively inflates its width by the escape bytes, which pads it too little and
// walks every border on the row to the left — and because the plain pass has no
// escapes, the two renders silently disagree.
func sgrPrefixLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b || s[1] != '[' {
		return 0
	}
	for i := 2; i < len(s); i++ {
		c := s[i]
		if c == 'm' {
			return i + 1
		}
		if (c < '0' || c > '9') && c != ';' {
			return 0
		}
	}
	return 0
}

// stripSGR removes every SGR sequence from s.
func stripSGR(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if n := sgrPrefixLen(s[i:]); n > 0 {
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// cellWidth returns the visible display width of s in terminal cells, ignoring
// any SGR sequences it carries.
func cellWidth(s string) int { return runewidth.StringWidth(stripSGR(s)) }

// pad right-pads s to w visible cells. Longer strings are returned untouched.
func pad(s string, w int) string {
	n := w - cellWidth(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

// truncEnd cuts s to w visible cells, passing SGR sequences through untouched
// and closing any style it leaves open.
func truncEnd(s string, w int, marker string) string {
	if cellWidth(s) <= w {
		return s
	}
	limit := w - cellWidth(marker)
	if limit < 0 {
		limit = 0
	}
	var b strings.Builder
	col := 0
	styled := false
	for i := 0; i < len(s); {
		if n := sgrPrefixLen(s[i:]); n > 0 {
			b.WriteString(s[i : i+n])
			styled = s[i:i+n] != "\x1b[0m"
			i += n
			continue
		}
		r, size := decodeRune(s[i:])
		rw := runewidth.RuneWidth(r)
		if col+rw > limit {
			break
		}
		b.WriteString(s[i : i+size])
		col += rw
		i += size
	}
	b.WriteString(marker)
	if styled {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// fitWidest picks the first variant that fits w display cells, falling back to
// the last one truncated when none does.
//
// The variants run widest first and the last is the floor — the one span the
// ladder is not allowed to drop. Passing a single variant is just truncEnd,
// which is the honest reading of "there is nothing left to drop".
//
// It fits the modal's ladders and not the status bar's, though both are the
// same shape. This one measures the string it returns, so the caller styles the
// winner afterwards; statusRow interleaves styling span by span and has to
// measure a plain twin of what it emits, which is why it carries its own
// variant type. Widening this to a plain/styled pair would let the two share an
// implementation, but that is a change to the status bar and belongs with one.
func fitWidest(variants []string, w int, marker string) string {
	for _, v := range variants {
		if cellWidth(v) <= w {
			return v
		}
	}
	if len(variants) == 0 {
		return ""
	}
	return truncEnd(variants[len(variants)-1], w, marker)
}

// truncMid elides the middle of s, keeping both ends legible. Long paths and
// commands ellipsize this way rather than wrapping inside a card title.
func truncMid(s string, w int, ell string) string {
	if cellWidth(s) <= w {
		return s
	}
	if w <= cellWidth(ell) {
		return truncEnd(s, w, "")
	}
	keep := w - cellWidth(ell)
	left, right := keep/2, keep-keep/2
	rs := []rune(stripSGR(s))
	var head, tail string
	for i := 0; i < len(rs) && cellWidth(head+string(rs[i])) <= left; i++ {
		head += string(rs[i])
	}
	for i := len(rs) - 1; i >= 0 && cellWidth(string(rs[i])+tail) <= right; i-- {
		tail = string(rs[i]) + tail
	}
	return head + ell + tail
}

// fill repeats r to exactly w cells.
func fill(r string, w int) string {
	if w <= 0 {
		return ""
	}
	rw := cellWidth(r)
	if rw == 0 {
		return ""
	}
	return strings.Repeat(r, w/rw)
}

// wrap soft-wraps s to w cells, breaking on spaces where possible and hard-
// breaking words longer than the line. There is no horizontal scrolling in
// v0.1, so everything that does not fit wraps. ui-spec §7.3.
func wrap(s string, w int) []string {
	if w <= 0 {
		return []string{""}
	}
	if s == "" {
		return []string{""}
	}
	var out []string
	for _, word := range strings.Fields(s) {
		switch {
		case len(out) == 0:
			out = append(out, word)
		case cellWidth(out[len(out)-1])+1+cellWidth(word) <= w:
			out[len(out)-1] += " " + word
		default:
			out = append(out, word)
		}
		// Hard-break anything still over width (a URL, a minified line).
		for cellWidth(out[len(out)-1]) > w {
			line := out[len(out)-1]
			head := runewidth.Truncate(line, w, "")
			out[len(out)-1] = head
			out = append(out, strings.TrimPrefix(line, head))
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// overlay blits box over base starting at row top, left-aligned at col left.
// Cells the box covers are replaced; everything else is untouched.
func overlay(base, box []string, top, left, w int) []string {
	out := make([]string, len(base))
	copy(out, base)
	for i, boxLine := range box {
		row := top + i
		if row < 0 || row >= len(out) {
			continue
		}
		out[row] = spliceAt(out[row], boxLine, left, w)
	}
	return out
}

// spliceAt replaces cells [at, at+width(ins)) of base with ins.
//
// It works in display cells, not bytes: if the splice lands midway through a
// wide rune, that rune is replaced with a space rather than being half-
// overwritten, which would shift the whole row by one cell and misalign every
// border below it.
func spliceAt(base, ins string, at, w int) string {
	insW := cellWidth(ins)
	left := cellsPrefix(base, at)
	rightStart := at + insW
	right := cellsSuffix(base, rightStart)
	line := left + ins + right
	return truncEnd(line, w, "")
}

// cellsPrefix returns the prefix of s occupying exactly n visible cells,
// space-padded if s is shorter, and space-substituted if the cut falls inside a
// wide rune. SGR sequences are carried through and closed.
func cellsPrefix(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	col := 0
	styled := false
	for i := 0; i < len(s); {
		if k := sgrPrefixLen(s[i:]); k > 0 {
			b.WriteString(s[i : i+k])
			styled = s[i:i+k] != "\x1b[0m"
			i += k
			continue
		}
		r, size := decodeRune(s[i:])
		rw := runewidth.RuneWidth(r)
		if col+rw > n {
			break
		}
		b.WriteString(s[i : i+size])
		col += rw
		i += size
	}
	if styled {
		b.WriteString("\x1b[0m")
	}
	if col < n {
		b.WriteString(strings.Repeat(" ", n-col))
	}
	return b.String()
}

// cellsSuffix returns the part of s from visible cell n onward, substituting a
// space if the cut falls inside a wide rune.
func cellsSuffix(s string, n int) string {
	var b strings.Builder
	col := 0
	for i := 0; i < len(s); {
		if k := sgrPrefixLen(s[i:]); k > 0 {
			if col >= n {
				b.WriteString(s[i : i+k])
			}
			i += k
			continue
		}
		r, size := decodeRune(s[i:])
		rw := runewidth.RuneWidth(r)
		switch {
		case col >= n:
			b.WriteString(s[i : i+size])
		case col+rw > n:
			b.WriteString(strings.Repeat(" ", col+rw-n))
		}
		col += rw
		i += size
	}
	return b.String()
}

// decodeRune is utf8.DecodeRuneInString, kept local so the width primitives
// have one obvious place to look.
func decodeRune(s string) (rune, int) {
	return utf8.DecodeRuneInString(s)
}

// ── Layout ──────────────────────────────────────────────────────────────────

// headerHeight is the number of rows the header occupies when it is shown: the
// wordmark and its rule on the first, the session line — project, branch, dirty
// marker, compaction note — on the second.
//
// It is a constant rather than a width-dependent value on purpose, and what
// makes that safe is the session rather than the width. Width decides only
// whether the branch is shown; the project name is never dropped for width.
// Row 2 is non-empty because every model carries a named session — New
// substitutes the Milestone 0 demo session when Options.Session has no project
// — so the row never renders empty and never needs to fold back into the first.
//
// That is a fixture, not an invariant. headerRows has no fallback of its own:
// when M2 wires a real session and the substitution goes, an anonymous session
// gives a blank row 2 and this constant stops being free.
// TestSessionLineNeverRendersEmpty is the test that will say so.
//
// Keeping the height out of the width bands is what lets ShowHeader stay a pure
// function of terminal height, which every §2.2 breakpoint test and the
// degradation ladder below both assume.
const headerHeight = 2

// framePad is the blank columns the frame keeps at its left and right edges.
//
// Horizontal only. There is no vertical equivalent and there is not meant to be
// one: the chrome budget in computeLayout spends every row the terminal has, so
// a blank row at the top or the bottom would come straight out of the
// transcript, and at the advertised minimum of 40×10 (ui-spec §1) the
// transcript is already down to four rows. A column is cheap at every width
// this app supports; a row is not.
const framePad = 1

// framePadding resolves a terminal width into the pad it keeps at each edge and
// the cells that leaves for content.
//
// The pad is unconditional wherever it leaves a column to frame. It is not
// gated on the §2.2 width bands, and deliberately so: the margin is a property
// of the frame rather than of a band, so the 38-column "terminal too narrow"
// notice sits inside the same margin as an 80-column session and the app does
// not change shape at the moment it is most degraded.
//
// At two columns and below there is nothing left to frame — the margin would be
// the whole frame, and the result would be a blank terminal rather than a
// narrow one — so the pad is dropped and the content takes every column. That
// makes content width dip by one between two columns and three, which is the
// seam any threshold has; it is placed as low as it can go, far below the
// supported minimum, where nothing legible renders either way.
func framePadding(w int) (pad, content int) {
	if c := w - 2*framePad; c >= 1 {
		return framePad, c
	}
	return 0, w
}

// Layout is the resolved geometry for one frame. It is a pure function of the
// terminal size and the composer's content height, which makes every §2.2
// breakpoint a table test with no rendering involved.
//
// Two widths, and which one a caller wants is never ambiguous:
//
//   - TermW is the terminal. It is what the §2.2 breakpoint bands are read
//     from, and nothing draws into it.
//   - ContentW is what every renderer in the package measures against, and the
//     only width any of them sees. Padding is applied once, in View, after the
//     last overlay has been composited — so no component knows a margin exists.
//
// TermW == ContentW + 2*Pad holds at every size.
type Layout struct {
	TermW, ContentW, Pad, H int

	ShowHeader     bool
	ShowTranscript bool
	ShowBranch     bool
	ShowTokens     bool
	ShowDuration   bool
	ShowRules      bool

	ModalPct    int
	TranscriptH int
	ComposerH   int
}

// HeaderH is how many rows the header contributes to the frame: headerHeight
// when it is shown, zero when it is not.
//
// Everything that needs to skip past the header — the chrome budget, the modal
// origin, the transcript band a test slices out — goes through this rather than
// through a literal 1. That is what stopped the header's row count being spelt
// out in four places that could disagree.
func (l Layout) HeaderH() int {
	if !l.ShowHeader {
		return 0
	}
	return headerHeight
}

// computeLayout resolves the frame geometry. ui-spec §2.1, §2.2.
//
// Chrome is header(2) + rule(1) + status(1) + rule(1) + composer(1..5) — six
// rows with a header, four without. The bands in §2.2 follow from that budget,
// and the degradation ladder below is the order they collapse in: composer and
// status are never dropped, the two rules go as a pair so the bar is never
// half-framed, then the header, and the transcript takes the remainder.
//
// overlayOpen is the one input that is not terminal geometry, and it is here
// rather than in the overlay because the region is what an overlay needs and
// the region is decided here. An overlay traps every key, so a frame that
// leaves it no row to draw into is not a smaller frame: it is an idle-looking
// frame that answers nothing and cannot be left. The composer is what pays for
// the row — under an overlay it accepts no input, which makes its rows the
// cheapest in the frame — and the ladder already spends the composer first, so
// the reservation is a floor on the existing order rather than a second one.
//
// Passing it through the same function instead of adjusting the region at the
// overlay is deliberate. The defect this closes was exactly that seam: the
// overlay decided it had no room while the layout had never been asked to make
// any, and the two could not disagree if there were only one of them. m.layout()
// is the single caller that supplies it, so View and every handler see the same
// geometry by construction.
//
// The header band still starts at h=10 even though the header now costs two
// rows rather than one. ui-spec §1 names 40×10 as the smallest supported
// terminal, and the smallest supported terminal is exactly where the full
// layout has to still be the full layout; raising the band to 11 would mean the
// advertised minimum no longer renders a header at all. The cost is that the
// full-layout band's transcript floor falls from five rows to four, and that
// shrinking from 10 rows to 9 now grows the transcript by one as the two-row
// header goes. That discontinuity is unavoidable once the header is a two-row
// unit — moving the band would relocate it, not remove it.
func computeLayout(w, h, composerLines int, overlayOpen bool) Layout {
	l := Layout{TermW: w, ContentW: w, H: h}
	if w <= 0 || h <= 0 {
		return l
	}
	l.Pad, l.ContentW = framePadding(w)

	// Width bands, read from the TERMINAL width and not from what the padding
	// leaves. ui-spec §1 puts the smallest supported terminal at 40 columns and
	// §2.2 starts the full-layout band there, so the two have to be the same
	// number: banding on content width would make a 40-column terminal render
	// the 38-column "terminal too narrow" notice and retire the advertised
	// minimum by a side effect of adding a margin. The same reasoning holds at
	// 60 and 80 — a band is a statement about the terminal the user has.
	switch {
	case w >= 80:
		l.ShowBranch, l.ShowTokens, l.ShowDuration, l.ShowTranscript = true, true, true, true
		l.ModalPct = 80
	case w >= 60:
		l.ShowBranch, l.ShowDuration, l.ShowTranscript = true, true, true
		l.ModalPct = 90
	case w >= 40:
		l.ShowTranscript = true
		l.ModalPct = 100
	default:
		l.ShowTranscript = false
		l.ModalPct = 100
	}

	l.ComposerH = clamp(composerLines, 1, 5)

	// Height ladder.
	switch {
	case h >= 10:
		l.ShowHeader, l.ShowRules = true, true
	case h >= 5:
		l.ShowHeader, l.ShowRules = false, true
	default:
		l.ShowHeader, l.ShowRules = false, false
	}

	// The region an open overlay must have. Zero otherwise: an empty transcript
	// region is an ordinary frame, and only an overlay makes it a trap.
	need := 0
	if overlayOpen {
		need = 1
	}

	for {
		chrome := 1 // status bar, never dropped
		chrome += l.HeaderH()
		if l.ShowRules {
			chrome += 2
		}
		chrome += l.ComposerH

		l.TranscriptH = h - chrome
		if l.TranscriptH >= need {
			break
		}
		// Not enough rows: degrade one step and re-derive.
		switch {
		case l.ComposerH > 1:
			l.ComposerH--
		case l.ShowRules:
			l.ShowRules = false
		case l.ShowHeader:
			l.ShowHeader = false
		default:
			// Everything the ladder can spend is spent. With need==0 this is a
			// frame too short for its own chrome; with need==1 it is a frame of
			// two rows or fewer, which has a status bar and a single composer
			// row and nothing left to buy a region with. The overlay does not
			// vanish there either — it leaves the region for the frame's last
			// row instead. See overlayFloorTop.
			l.TranscriptH = maxInt(0, l.TranscriptH)
			return l
		}
	}
	if !l.ShowTranscript {
		// Below 40 cols the transcript is replaced by a one-line notice, which
		// still occupies the region so the row count is unchanged.
		return l
	}
	return l
}

// clamp confines v to [lo, hi].
//
// It requires lo <= hi and does not check. Called with lo > hi the range is
// empty, there is no answer, and what comes back is hi — the upper bound wins
// and the lower one is silently discarded. Every caller here passes constants
// in the right order except one that did not: overlayConfirm sized its box with
// clamp(want, 20, lay.ContentW-2), and at twenty content columns the cap was
// eighteen and the floor twenty, so the floor lost at the width it existed for.
// Nothing rendered wrong; a comment elsewhere simply stopped being true.
//
// The lesson is about the shape rather than that one site: a bound derived from
// the terminal and a bound that is a constant can cross, and a clamp cannot say
// which of them matters. Where they can cross, write the two steps out in the
// order that states which wins. See confirmBoxWidth.
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// View renders the whole frame.
//
// Every layer returns []string whose length is a contract; only this function
// joins. That is what keeps the row count checkable at each stage instead of
// being an emergent property of string concatenation.
func (m Model) View() string {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return "" // §2.2: degenerate sizes render nothing rather than panicking
	}
	// m.layout() and not a fresh computeLayout: the overlay flag has to be the
	// same one every handler resolved geometry against, and a second call site
	// is a second chance to leave it out.
	lay := m.layout()

	rows := make([]string, 0, h)
	if lay.ShowHeader {
		rows = append(rows, m.headerRows(lay)...)
	}
	rows = append(rows, m.transcriptRows(lay)...)
	if lay.ShowRules {
		rows = append(rows, m.ruleRow(lay))
	}
	rows = append(rows, m.statusRow(lay))
	if lay.ShowRules {
		rows = append(rows, m.ruleRow(lay))
	}
	comp := m.comp
	comp.Focused = m.mode() == ModeComposing && !m.busy.Active
	comp.Onboarding = m.tr.Len() == 0
	rows = append(rows, comp.rows(lay, m.sty, m.gly, m.busy.Active)...)

	// Modals overwrite the transcript region only; the status bar and composer
	// stay visible. Layout invariant 3.
	if m.modal != nil {
		rows = m.overlayModal(rows, lay)
	} else if m.confirm != nil {
		rows = m.overlayConfirm(rows, lay)
	}

	return strings.Join(padFrame(exactly(rows, h, lay.ContentW), lay), "\n")
}

// exactly forces the frame to h rows of at most contentW cells. Any drift here
// is a bug upstream, but a truncated frame beats a corrupted terminal.
//
// contentW and not the terminal width: this runs before padFrame, so the rows
// it bounds are content rows and the margin has not been added yet.
func exactly(rows []string, h, contentW int) []string {
	for len(rows) < h {
		rows = append(rows, "")
	}
	if len(rows) > h {
		rows = rows[:h]
	}
	for i := range rows {
		rows[i] = truncEnd(rows[i], contentW, "")
	}
	return rows
}

// padFrame insets the finished frame by the layout's margin.
//
// The single place in the package that knows the margin exists. Every layer
// beneath it renders at lay.ContentW and is unaware it is being inset, which is
// what keeps the knowledge out of the components: a renderer that padded itself
// would have to be right about the margin, and there are a dozen of them.
//
// It runs last, after the overlays. An overlay covers the span of the row it
// replaces, and that span is the content span — the margin is not the overlay's
// to fill. Padding before compositing would make it exactly that.
//
// Only the left column is written. The right one is RESERVED rather than
// emitted: content is bounded at lay.ContentW and shifted right by the pad, so
// nothing can reach the last column, and writing a space into it would put
// trailing whitespace on every row of every frame without changing a single
// rendered cell. Backgrounds do not change that — CodeBg and SelectionBg are
// applied inside a card, well within the content span, so the margin columns
// carry the terminal's own background at both edges either way.
func padFrame(rows []string, lay Layout) []string {
	if lay.Pad <= 0 {
		return rows
	}
	left := strings.Repeat(" ", lay.Pad)
	for i := range rows {
		rows[i] = left + rows[i]
	}
	return rows
}

// headerRows renders the two-row header. ui-spec §2.
//
//	Kirsch ────────────────────────────────────────
//	project ─ branch ● (compacted)
//
// The wordmark stands alone on the first row with the rule running to the last
// content column — the frame's margin sits outside it, added by padFrame —
// and everything that identifies the session sits on the second. Splitting
// them is what stops a long project name, a long branch name and the compaction
// note competing for the same row — at 40 columns the single-row form was
// already spending most of its width on text rather than rule.
//
// Only the first row carries a rule. A second full-width rule directly beneath
// the first would read as the top edge of a box rather than as a header, and
// the transcript's own separator already closes the region below.
//
// The return is always headerHeight rows, so the caller's row arithmetic does
// not have to inspect the content.
func (m Model) headerRows(lay Layout) []string {
	// Row 1: the wordmark, then a rule to the content edge.
	wordmark := m.sty.Accent("Kirsch")
	if n := lay.ContentW - cellWidth("Kirsch") - 1; n > 0 {
		wordmark += " " + m.sty.Border(fill(m.gly.Sep, n))
	}

	// Row 2: the session line. Spans are collected and joined rather than
	// concatenated with leading spaces, so an absent project name or branch
	// leaves no stray separator behind.
	var names []string
	if m.sess.Project != "" {
		names = append(names, m.sess.Project)
	}
	// The branch and its dirty marker share one gate, and that gate is width
	// alone: below 60 columns §2.2 drops the branch, and a marker left behind
	// would be qualifying a branch the reader cannot see.
	//
	// It is deliberately not a gate on the branch being set. At 60 columns or
	// wider a session with no branch still shows the marker, because there it
	// qualifies nothing hidden — it says the worktree is dirty, which is true
	// and is the whole of what it claims. TestSessionLineOmitsAbsentFields pins
	// that case at "my-project ●".
	if lay.ShowBranch && m.sess.Branch != "" {
		names = append(names, m.sess.Branch)
	}
	var spans []string
	if joined := strings.Join(names, " "+m.gly.Sep+" "); joined != "" {
		spans = append(spans, m.sty.Accent(joined))
	}
	if lay.ShowBranch && m.sess.Dirty {
		spans = append(spans, m.sty.Warning(m.gly.Dirty))
	}
	if m.sess.Compacted {
		spans = append(spans, m.sty.Muted("(compacted)"))
	}

	return []string{wordmark, strings.Join(spans, " ")}
}

// ruleRow is a separator across the full content width — which is the frame
// less its margin, not the terminal. padFrame is what puts the margin either
// side of it.
func (m Model) ruleRow(lay Layout) string {
	return m.sty.Border(fill(m.gly.Sep, lay.ContentW))
}
