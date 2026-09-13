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

// Layout is the resolved geometry for one frame. It is a pure function of the
// terminal size and the composer's content height, which makes every §2.2
// breakpoint a table test with no rendering involved.
type Layout struct {
	W, H int

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

// computeLayout resolves the frame geometry. ui-spec §2.1, §2.2.
//
// Chrome is header(1) + rule(1) + status(1) + rule(1) + composer(1..5) — five
// rows with a header, four without. The bands in §2.2 follow from that budget,
// and the degradation ladder below is the order they collapse in: composer and
// status are never dropped, the two rules go as a pair so the bar is never
// half-framed, then the header, and the transcript takes the remainder.
func computeLayout(w, h, composerLines int) Layout {
	l := Layout{W: w, H: h}
	if w <= 0 || h <= 0 {
		return l
	}

	// Width bands.
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

	for {
		chrome := 1 // status bar, never dropped
		if l.ShowHeader {
			chrome++
		}
		if l.ShowRules {
			chrome += 2
		}
		chrome += l.ComposerH

		l.TranscriptH = h - chrome
		if l.TranscriptH >= 0 {
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
			l.TranscriptH = 0
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
	lay := computeLayout(w, h, m.comp.contentLines())

	rows := make([]string, 0, h)
	if lay.ShowHeader {
		rows = append(rows, m.headerRow(lay))
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

	return strings.Join(exactly(rows, h, w), "\n")
}

// exactly forces the frame to h rows of at most w cells. Any drift here is a
// bug upstream, but a truncated frame beats a corrupted terminal.
func exactly(rows []string, h, w int) []string {
	for len(rows) < h {
		rows = append(rows, "")
	}
	if len(rows) > h {
		rows = rows[:h]
	}
	for i := range rows {
		rows[i] = truncEnd(rows[i], w, "")
	}
	return rows
}

// headerRow renders `Kirsch ─ project ─ branch ●` followed by a rule to the
// right edge. ui-spec §2.
func (m Model) headerRow(lay Layout) string {
	title := "Kirsch " + m.gly.Sep + " " + m.sess.Project
	if lay.ShowBranch && m.sess.Branch != "" {
		title += " " + m.gly.Sep + " " + m.sess.Branch
	}
	out := m.sty.Accent(title)
	plain := title
	if lay.ShowBranch && m.sess.Dirty {
		out += " " + m.sty.Warning(m.gly.Dirty)
		plain += " " + m.gly.Dirty
	}
	if m.sess.Compacted {
		out += m.sty.Muted(" (compacted)")
		plain += " (compacted)"
	}
	if n := lay.W - cellWidth(plain) - 1; n > 0 {
		out += " " + m.sty.Border(fill(m.gly.Sep, n))
	}
	return out
}

// ruleRow is a full-width separator.
func (m Model) ruleRow(lay Layout) string {
	return m.sty.Border(fill(m.gly.Sep, lay.W))
}
