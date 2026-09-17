package tui

import (
	"fmt"
	"strings"
)

// renderCtx is everything a card needs to draw itself. Notably it carries the
// spinner Frame rather than letting cards read a clock, which is what keeps
// View() a pure function of the model and the goldens byte-stable.
type renderCtx struct {
	// W is the cells this one card may draw into — Layout.ContentW less the
	// gutter where the item draws one. It is never the terminal's width and
	// never the frame's: by the time a card is rendered the frame's margin has
	// already been taken out of Layout.ContentW, one level up.
	W        int
	Gutter   bool
	Expanded bool
	Selected bool
	Sty      *Styles
	G        Glyphs
	Frame    int
	ShowDur  bool
}

// renderItem is the one place a transcript item becomes lines.
//
// One switch, one arm per kind. A separate Lines() implementation per kind would
// make every one of them an opportunity to break the rule that line counts must
// not change with colour; here there is a single place to audit.
func renderItem(it Item, ctx renderCtx) []string {
	var body []string
	// Content is folded to ASCII at the render boundary, not in the stored
	// item: the transcript holds what actually happened, and the terminal's
	// capabilities are a property of the frame, not of the data.
	switch it.Kind {
	case KindUser:
		body = renderUser(it.Text, ctx)
	case KindAssistant:
		body = renderAssistant(it.Text, ctx)
	case KindTool:
		body = renderTool(it.Tool, ctx)
	case KindApproval:
		body = renderApproval(it.Approval, ctx)
	case KindError:
		body = renderError(it.Err, ctx)
	case KindNotice:
		body = renderNotice(it.Notice, ctx)
	case KindThinking:
		body = renderThinking(it.Thinking, ctx)
	default:
		body = []string{ctx.Sty.Dim("· unknown item")}
	}
	if ctx.Gutter {
		g := ctx.Sty.Accent(ctx.G.Gutter)
		for i, l := range body {
			body[i] = g + " " + l
		}
	}
	return body
}

// gutterWidth is the space renderItem's gutter takes from the content width.
func gutterWidth(g Glyphs) int { return cellWidth(g.Gutter) + 1 }

// box draws a bordered panel whose every row is exactly the same width.
//
// The reference grids were committed with ragged borders — right edges
// wandering by a cell between the top rule and the interior — because three
// boxes each did their own arithmetic. There is one box builder for that
// reason: outer width is the single input, and top, interior and bottom rows
// are derived from it, so they cannot disagree.
func box(indent int, title string, body []string, outer int, edge Style, ctx renderCtx) []string {
	if outer < 6 {
		outer = 6
	}
	content := outer - 2 // cells between the two vertical borders
	pre := strings.Repeat(" ", indent)

	top := ctx.G.BoxTL
	if title != "" {
		t := " " + truncEnd(title, content-2, ctx.G.Trunc) + " "
		top += ctx.G.BoxH + t + fill(ctx.G.BoxH, content-cellWidth(t)-1)
	} else {
		top += fill(ctx.G.BoxH, content)
	}
	top += ctx.G.BoxTR

	out := []string{pre + edge(top)}
	for _, ln := range body {
		out = append(out, pre+edge(ctx.G.BoxV)+pad(ln, content)+edge(ctx.G.BoxV))
	}
	out = append(out, pre+edge(ctx.G.BoxBL+fill(ctx.G.BoxH, content)+ctx.G.BoxBR))
	return out
}

func renderUser(t *TextBlock, ctx renderCtx) []string {
	label := " you "
	rule := ctx.G.Sep + ctx.G.Sep
	n := ctx.W - cellWidth(rule) - cellWidth(label)
	head := ctx.Sty.Border(rule) + ctx.Sty.Muted(label)
	if n > 0 {
		head += ctx.Sty.Border(fill(ctx.G.Sep, n))
	}
	out := []string{head}
	for _, ln := range t.Lines {
		for _, w := range wrap(ctx.G.Fold(ln), ctx.W) {
			out = append(out, ctx.Sty.Text(w))
		}
	}

	return out
}

func renderAssistant(t *TextBlock, ctx renderCtx) []string {
	var out []string
	for i, ln := range t.Lines {
		wrapped := wrap(ctx.G.Fold(ln), ctx.W)
		for j, w := range wrapped {
			last := i == len(t.Lines)-1 && j == len(wrapped)-1
			s := ctx.Sty.Text(w)
			if last && t.Streaming {
				s += " " + ctx.Sty.Dim(ctx.G.Caret)
			}
			out = append(out, s)
		}
	}
	if t.Cancelled {
		out = append(out, ctx.Sty.Dim(ctx.G.Cancelled+" cancelled"))
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

// badge returns the lifecycle glyph and its styled form. Every state is
// identifiable from the glyph alone; colour only reinforces it. ui-spec §12.
func badge(st CardState, ctx renderCtx) (plain, styled string) {
	switch st {
	case StatePending:
		return ctx.G.Pending, ctx.Sty.Dim(ctx.G.Pending)
	case StateRunning:
		return ctx.G.Running, ctx.Sty.Accent(ctx.G.Running)
	case StateOK:
		l := ctx.G.okLabel()
		return l, ctx.Sty.Success(l)
	case StateError:
		return ctx.G.Err, ctx.Sty.Error(ctx.G.Err)
	case StateCancelled:
		return ctx.G.Cancelled, ctx.Sty.Dim(ctx.G.Cancelled)
	}
	return "", ""
}

func renderTool(c *ToolCard, ctx renderCtx) []string {
	glyph := ctx.G.Collapsed
	if ctx.Expanded {
		glyph = ctx.G.Expanded
	}
	if c.State == StateRunning {
		glyph = ctx.G.Running
	}

	dot := " " + ctx.G.Bullet + " "
	head := ctx.Sty.Accent(glyph) + " " + ctx.Sty.Bold(ctx.Sty.Text(c.Name))
	if c.Target != "" {
		head += " " + ctx.Sty.Muted(truncMid(c.Target, ctx.W/2, ctx.G.Trunc))
	}
	// Field order is glyph, name, target, summary, duration, status — §3.3 and
	// every grid. An errored command carries its summary with the ✗ instead, so
	// the line reads "· 2.4s · ✗ exit 1" rather than repeating it.
	if c.Summary != "" && c.State != StateError {
		head += ctx.Sty.Muted(dot + c.Summary)
	}
	if ctx.ShowDur && c.Elapsed > 0 {
		head += ctx.Sty.Muted(dot + formatDuration(c.Elapsed))
	}
	switch c.State {
	case StateRunning:
		head += ctx.Sty.Muted(dot + "running")
	case StateError:
		// An errored command carries its summary with the ✗ rather than in the
		// slot before the duration, so the line reads "· 2.4s · ✗ exit 1".
		txt := ctx.G.Err
		if c.Summary != "" {
			txt += " " + c.Summary
		}
		head += ctx.Sty.Muted(dot) + ctx.Sty.Error(txt)
	case StatePending, StateOK, StateCancelled:
		if _, styled := badge(c.State, ctx); styled != "" {
			head += ctx.Sty.Muted(dot) + styled
		}
	}
	if c.Trunc != "" {
		head += " " + ctx.Sty.Warning(ctx.G.Trunc)
	}
	if c.Note != "" {
		head += ctx.Sty.Muted(dot + c.Note)
	}
	out := []string{head}

	if !ctx.Expanded || len(c.Out) == 0 {
		return out
	}

	// Expanded: a fenced panel, capped at 200 rendered lines. The card keeps
	// everything; the cap is a render decision so `d` can still show it all.
	const indent = 2
	outer := ctx.W - indent
	inner := outer - 3 // borders plus the leading space
	if inner < 8 {
		inner = 8
	}
	shown := c.Out
	if len(shown) > InlineExpandCap {
		shown = shown[:InlineExpandCap]
	}
	body := make([]string, 0, len(shown))
	for _, ln := range shown {
		body = append(body, " "+ctx.Sty.CodeBg(pad(truncEnd(ln, inner, ctx.G.Trunc), inner)))
	}
	out = append(out, box(indent, "", body, outer, ctx.Sty.Border, ctx)...)
	if len(c.Out) > InlineExpandCap {
		out = append(out, strings.Repeat(" ", indent)+ctx.Sty.Warning(
			fmt.Sprintf("‹%d of %s lines — press d for full output›", InlineExpandCap, comma(len(c.Out)))))
	}
	return out
}

func renderApproval(a *ApprovalCard, ctx renderCtx) []string {
	// Once resolved the card collapses to a single tool-card-shaped line. Same
	// item, same position — it is not rewritten, it renders its later state.
	if a.Outcome != Unresolved {
		dot := " " + ctx.G.Bullet + " "
		line := ctx.Sty.Accent(ctx.G.Collapsed) + " " + ctx.Sty.Bold(ctx.Sty.Text(toolNameFor(a)))
		line += " " + ctx.Sty.Muted(a.Subject)
		if ctx.ShowDur && a.Elapsed > 0 {
			line += ctx.Sty.Muted(dot + formatDuration(a.Elapsed))
		}
		switch a.Outcome {
		case Approved:
			line += ctx.Sty.Muted(dot) + ctx.Sty.Success(ctx.G.OK+" approved")
		case ApprovedSession:
			line += ctx.Sty.Muted(dot) + ctx.Sty.Success(ctx.G.OK+" approved")
			line += ctx.Sty.Muted(dot + "session grant: " + a.GrantScope)
		case Rejected:
			line += ctx.Sty.Muted(dot) + ctx.Sty.Error(ctx.G.Err+" rejected")
		case Unresolved:
			// Unreachable: the enclosing branch tests for it. Listed so the
			// compiler and the linter both notice if a new outcome is added.
		}
		return []string{line}
	}

	body := append([]string{ctx.G.Fold(a.Title), ""}, a.Detail...)
	body = append(body, "", a.actionRow(ctx))
	for i, ln := range body {
		body[i] = " " + styleActionRow(ctx.G.Fold(ln), ctx)
	}
	return box(0, "approval required", body, ctx.W, ctx.Sty.Warning, ctx)
}

// actionRow builds the key row. [a] appears only where OffersSessionGrant says
// it may, which for a patch is never. ADR 0006.
func (a *ApprovalCard) actionRow(ctx renderCtx) string {
	parts := []string{"[y] approve"}
	if a.OffersSessionGrant() {
		parts = append(parts, "[a] approve for session")
	}
	parts = append(parts, "[n] reject")
	if a.Kind == ApprovalPatch {
		parts = append(parts, "[d] view diff")
	} else {
		parts = append(parts, "[d] detail")
	}
	return strings.Join(parts, "   ")
}

func toolNameFor(a *ApprovalCard) string {
	if a.Kind == ApprovalPatch {
		return "apply_patch"
	}
	return "run_command"
}

// styleActionRow colours the bracketed keys without disturbing the row's width:
// the substitutions are same-length, so padding computed on the plain string
// still lands correctly.
func styleActionRow(s string, ctx renderCtx) string {
	if !strings.Contains(s, "[y]") {
		return ctx.Sty.Text(s)
	}
	out := s
	out = strings.Replace(out, "[y]", ctx.Sty.Success("[y]"), 1)
	out = strings.Replace(out, "[a]", ctx.Sty.Success("[a]"), 1)
	out = strings.Replace(out, "[n]", ctx.Sty.Error("[n]"), 1)
	out = strings.Replace(out, "[d]", ctx.Sty.Accent("[d]"), 1)
	return out
}

func renderError(e *ErrorCard, ctx renderCtx) []string {
	inner := ctx.W - 3
	if inner < 12 {
		inner = 12
	}
	lines := wrap(ctx.G.Fold(e.Message), inner)
	if e.Hint != "" {
		lines = append(lines, e.Hint)
	}
	body := make([]string, 0, len(lines))
	for _, ln := range lines {
		body = append(body, " "+ctx.Sty.Text(ctx.G.Fold(ln)))
	}
	return box(0, e.Kind, body, ctx.W, ctx.Sty.Error, ctx)
}

func renderNotice(n *NoticeCard, ctx renderCtx) []string {
	var out []string
	for _, w := range wrap(ctx.G.Bullet+" "+ctx.G.Fold(n.Text), ctx.W) {
		out = append(out, ctx.Sty.Dim(w))
	}
	return out
}

func renderThinking(t *ThinkingCard, ctx renderCtx) []string {
	return []string{ctx.Sty.Dim(fmt.Sprintf("%s thinking %s %d tok",
		ctx.G.Collapsed, ctx.G.Bullet, t.Tokens))}
}

// comma groups an integer with thousands separators, matching the "4,176
// lines" in screen 07's grid. The colour map beside that grid still reads
// 4,181 — ui-spec §7's figure, for a differently-composed card — and the grid
// is the half the goldens compare against.
func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}
