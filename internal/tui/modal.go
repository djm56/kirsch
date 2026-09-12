package tui

import (
	"fmt"
	"strings"
)

// ModalKind selects how a modal's body is rendered.
type ModalKind uint8

const (
	ModalContent ModalKind = iota + 1
	ModalDiff
	ModalHelp
)

// ModalState is an open modal. One modal serves both the content and diff jobs.
// ui-spec §4.1.
type ModalState struct {
	Kind    ModalKind
	Title   string
	Source  ItemID // the card it was opened from; zero for /diff and /help
	Lines   []string
	Added   int
	Removed int
	Off     int // the modal's own scroll offset, never the transcript's
}

// ConfirmAction is what a confirm prompt will do on `y`.
//
// An enum rather than a func(): a closure in the model makes it unprintable and
// undiffable, and hides the effect from update.go, which is meant to own every
// effect.
type ConfirmAction uint8

const (
	ConfirmNewSession ConfirmAction = iota + 1
	ConfirmClearGrants
	ConfirmLargePaste
)

// ConfirmState is the one-line destructive-action prompt. ui-spec §4.3.
type ConfirmState struct {
	Prompt  string
	Action  ConfirmAction
	Payload string // pending paste content, for ConfirmLargePaste
}

// modalBox builds the modal's lines and its placement within the transcript
// region. Modals clamp to the §2.2 breakpoint sizes and trap all input.
func (m Model) modalBox(lay Layout) (box []string, top, left int) {
	w := lay.W * lay.ModalPct / 100
	if w > lay.W-2 {
		w = lay.W - 2
	}
	if w < 20 {
		w = 20
	}
	md := m.modal

	// Height fits the content, capped by the region: chrome is the top border,
	// the footer rule, the footer and the bottom border.
	const chrome = 4
	h := len(md.Lines) + chrome
	if h > lay.TranscriptH {
		h = lay.TranscriptH
	}
	if h < 5 {
		h = 5
	}
	left = (lay.W - w) / 2
	top = 0
	if lay.ShowHeader {
		top = 1
	}

	inner := w - 4

	title := " " + truncEnd(md.Title, inner-10, "…") + " "
	head := m.gly.BoxTL + m.gly.BoxH + title
	if md.Kind == ModalDiff && (md.Added > 0 || md.Removed > 0) {
		stat := fmt.Sprintf(" +%d −%d ", md.Added, md.Removed)
		n := w - cellWidth(head) - cellWidth(stat) - 1
		if n > 0 {
			head += fill(m.gly.BoxH, n)
		}
		head += stat + m.gly.BoxTR
	} else {
		n := w - cellWidth(head) - 1
		if n > 0 {
			head += fill(m.gly.BoxH, n)
		}
		head += m.gly.BoxTR
	}
	box = append(box, m.sty.BorderFocus(head))

	// Body: height minus the top border, the footer rule and the footer line.
	bodyH := h - 4
	if bodyH < 1 {
		bodyH = 1
	}
	off := clamp(md.Off, 0, maxInt(0, len(md.Lines)-bodyH))
	for i := 0; i < bodyH; i++ {
		var content string
		if idx := off + i; idx < len(md.Lines) {
			content = m.modalLine(md, md.Lines[idx], idx, inner)
		}
		box = append(box, m.sty.BorderFocus(m.gly.BoxV)+" "+pad(content, inner+1)+
			m.sty.BorderFocus(m.gly.BoxV))
	}

	box = append(box, m.sty.BorderFocus(m.gly.BoxLT+fill(m.gly.BoxH, w-2)+m.gly.BoxRT))

	footer := "j/k scroll " + m.gly.Bullet + " g/G top/bottom " + m.gly.Bullet + " Esc "
	if m.pendingApproval != 0 {
		footer += "back to approval"
	} else {
		footer += "close"
	}
	if md.Kind == ModalHelp {
		footer = "kirsch v" + m.sess.Version + " " + m.gly.Bullet +
			" docs: doc/usage.md " + m.gly.Bullet + " Esc or ? closes"
	}
	box = append(box, m.sty.BorderFocus(m.gly.BoxV)+" "+
		pad(m.sty.Muted(truncEnd(footer, inner+1, "…")), inner+1)+
		m.sty.BorderFocus(m.gly.BoxV))
	box = append(box, m.sty.BorderFocus(m.gly.BoxBL+fill(m.gly.BoxH, w-2)+m.gly.BoxBR))
	return box, top, left
}

// modalLine styles one body line: diffs from parsed structure, plain content
// with line numbers.
//
// Diff colour comes from the leading character, never from escapes in the
// content — Kirsch colours diffs itself. ui-spec §4.1, §7.1.
func (m Model) modalLine(md *ModalState, raw string, idx, inner int) string {
	switch md.Kind {
	case ModalHelp:
		return m.styleHelpLine(raw, inner)
	case ModalContent:
		num := m.sty.Dim(fmt.Sprintf("%4d ", idx+1))
		return num + m.sty.Text(truncEnd(raw, inner-5, m.gly.Trunc))
	}
	body := truncEnd(raw, inner, m.gly.Trunc)
	switch {
	case strings.HasPrefix(raw, "@@"):
		return m.sty.Hunk(body)
	case strings.HasPrefix(raw, "+"):
		return m.sty.Success(body)
	case strings.HasPrefix(raw, "-"):
		return m.sty.Error(body)
	default:
		return m.sty.Muted(body)
	}
}

// overlayModal composites the modal over the transcript region, dimming what is
// behind it. The status bar and composer are left untouched.
func (m Model) overlayModal(rows []string, lay Layout) []string {
	box, top, left := m.modalBox(lay)
	return overlay(rows, box, top, left, lay.W)
}

// overlayConfirm composites the one-line confirm prompt. ui-spec §4.3.
func (m Model) overlayConfirm(rows []string, lay Layout) []string {
	c := m.confirm
	text := c.Prompt + "  [y] yes   [n] no"
	w := clamp(cellWidth(text)+4, 20, lay.W-2)
	left := (lay.W - w) / 2
	top := lay.H - lay.ComposerH - 3
	if top < 0 {
		top = 0
	}
	inner := w - 4
	box := []string{
		m.sty.BorderFocus(m.gly.BoxTL + fill(m.gly.BoxH, w-2) + m.gly.BoxTR),
		m.sty.BorderFocus(m.gly.BoxV) + " " +
			pad(m.sty.Text(truncEnd(text, inner+1, "…")), inner+1) +
			m.sty.BorderFocus(m.gly.BoxV),
		m.sty.BorderFocus(m.gly.BoxBL + fill(m.gly.BoxH, w-2) + m.gly.BoxBR),
	}
	return overlay(rows, box, top, left, lay.W)
}

// helpLines builds the help overlay body as two columns, matching screen 06.
//
// Both columns are generated from the same binding table update.go dispatches
// on, so a binding that changes behaviour cannot quietly keep its old
// description here.
func helpLines() []string {
	left := helpColumn(13, "composing", "browsing")
	right := helpColumn(9, "approval", "modal")
	right = append(right, "", "commands")
	var cmds []string
	for _, c := range SlashCommands {
		cmds = append(cmds, "/"+c)
	}
	for i := 0; i < len(cmds); i += 2 {
		right = append(right, strings.Join(cmds[i:minInt(i+2, len(cmds))], "  "))
	}

	n := maxInt(len(left), len(right))
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		var l, r string
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if r == "" {
			out = append(out, strings.TrimRight(l, " "))
			continue
		}
		out = append(out, pad(l, helpColWidth)+r)
	}
	return out
}

// helpColWidth is where the right-hand column starts.
const helpColWidth = 37

// helpColumn renders the named binding groups as "key  description" rows, with
// the key field padded to keyW. The right-hand column uses a narrower key field
// because its keys are single characters — a shared width would push its
// descriptions past the modal's edge at the §2.2 80% width.
func helpColumn(keyW int, groups ...string) []string {
	var out []string
	for gi, name := range groups {
		if gi > 0 {
			out = append(out, "")
		}
		for _, g := range bindingGroups {
			if g.Mode != name {
				continue
			}
			out = append(out, g.Mode)
			for _, b := range g.Bindings {
				out = append(out, fmt.Sprintf("%-*s %s", keyW, b.Key, b.Desc))
			}
		}
	}
	return out
}

// styleHelpLine colours a help row: section headings warning, bindings muted.
func (m Model) styleHelpLine(raw string, inner int) string {
	body := truncEnd(raw, inner, m.gly.Trunc)
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return body
	}
	// A heading is a bare group name with no key column beside it.
	for _, g := range bindingGroups {
		if trimmed == g.Mode || strings.HasPrefix(body, g.Mode) && !strings.Contains(body, "  ") {
			return m.sty.Warning(body)
		}
	}
	if trimmed == "commands" {
		return m.sty.Warning(body)
	}
	return m.sty.Muted(body)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
