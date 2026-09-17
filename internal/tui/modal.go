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

// overlayFloorTop is the frame row an overlay takes when the region has no rows
// at all: the last one.
//
// It is the floor under both overlay paths and it is not reachable at any
// supported size. computeLayout reserves a region row while an overlay is open
// and pays for it out of the composer, so only a frame of two rows or fewer —
// a status bar and a single composer row, with nothing left to spend — arrives
// here. ui-spec §1 puts the smallest supported terminal at 40×10.
//
// Leaving the region is a deliberate exception to Layout invariant 3, and the
// invariant names it. The rule protects the status bar and the composer from
// being painted over by an overlay that had somewhere else to go. At two rows
// there is nowhere else: the choice is not between a correct frame and a
// corrupted one but between a frame that states the question and a frame that
// says `idle` while every key is trapped. A status bar that is wrong about the
// state of the session has not earned the row it is protected for.
//
// The last row and not the first, and what that costs differs by one row of
// height, so both are worth stating:
//
//   - At h==2 the frame is the status bar then the composer, and the last row is
//     the composer's. The status bar survives intact; what is given up is a row
//     of a draft that cannot be typed into while the overlay holds every key.
//     This is the cheaper trade, and it is the one the common case takes.
//   - At h==1 the frame is built as status plus composer and then cut to one row
//     by exactly, so the row written here is the only row that survives. The
//     status bar is what is given up, and the paragraph above is why.
//
// It is also the row nearest the composer at both, which is where ui-spec §4.3
// puts the confirm prompt when there is a region to put it in.
func overlayFloorTop(lay Layout) int {
	if lay.H < 1 {
		return 0
	}
	return lay.H - 1
}

// overlayRow frames a one-row overlay form as a box that covers every cell of
// the span it is spliced into.
//
// The pad is the whole of the function, and it is load-bearing rather than
// tidying. modalNotice and confirmRow are width LADDERS: each returns the widest
// variant that FITS the width it is handed, which is almost never a variant of
// that width. overlay splices through spliceAt, and spliceAt replaces only the
// cells the string occupies — so a variant shorter than its span leaves the row
// underneath showing beside it, which is one row reading as two things. That is
// the exact frame the floor comment in modalBox says it is avoiding, and without
// this pad the code produced it at all four one-row sites:
//
//   - 80×10 with a three-line draft: the help notice is 22 cells of a 64-cell
//     box span, and the transcript supplies the other 42.
//   - h≤2, 80 columns: "Discard?  [y] yes   [n] no" over a status bar, which
//     finishes the row with "2.3k tok".
//   - h=1, 20 columns: "Discard?  y/n" over the composer, which finishes it with
//     "le" — the answer key itself reads as "y/nle".
//
// Bordered forms never needed it. They are built at exactly the box width with a
// border on each end, and every row inside them is padded where it is PLACED —
// modalBox's body and footer rows, overlayConfirm's question row. That is this
// package's rule, and these four were the placements missing it.
//
// Padding inside the two ladders instead would have been one edit rather than
// four, and it is the wrong place for two reasons. confirmRow is also called
// from inside the bordered box, where the caller already pads: the same string
// would then be assembled differently depending on which form drew it, with the
// trailing cells inside the SGR span on one path and outside it on the other.
// And a ladder that pads is answering two questions — what fits, and what covers
// — where only the first is a ladder's. Keeping them apart is what lets the
// span be a box on one call and the whole frame on the next.
func overlayRow(s string, w int) []string { return []string{pad(s, w)} }

// modalBox builds the modal's lines and its placement within the transcript
// region. Modals clamp to the §2.2 breakpoint sizes and trap all input.
func (m Model) modalBox(lay Layout) (box []string, top, left int) {
	w := lay.ContentW * lay.ModalPct / 100
	if w > lay.ContentW-2 {
		w = lay.ContentW - 2
	}
	if w < 20 {
		w = 20
	}
	md := m.modal
	left = (lay.ContentW - w) / 2
	top = lay.HeaderH()
	inner := w - 4

	// Height fits the content, capped by the region, and the box never leaves
	// the region — Layout invariant 3 is what stops a modal painting over the
	// status bar and the composer.
	//
	// What the box spends on itself, in the order it gives it up: the footer
	// rule first, then the body. The two borders and the footer are never
	// dropped. That ordering is the whole of the ladder's reasoning — a modal
	// traps every key, so the row naming the way out outranks the rows showing
	// the content, exactly as it outranks the rule that separates them. A box
	// holding one line of a help overlay and no exit key is worth less than one
	// holding no content and an exit key, because only the second can be left.
	//
	// The ladder exists because the smallest supported terminal (40×10, ui-spec
	// §1) leaves a four-row region once the two-row header is paid for, one row
	// short of the full box.
	//
	// Below three rows no bordered box holds a footer, so the modal degrades
	// further — out of the box form entirely, into the single notice row
	// modalNotice draws. It is never absent: a modal that overruns the region
	// corrupts the rows underneath it, but a modal drawing nothing at all leaves
	// a frame that reads as an ordinary idle session with every key silently
	// trapped, and that is the worse of the two. A region of no rows is not the
	// exception it used to be — the notice leaves the region for the frame's
	// last row rather than going out, and overlayFloorTop says why.
	showRule, showBody := true, true
	var chrome int
	for {
		chrome = 3 // the two borders and the footer
		if showRule {
			chrome++
		}
		need := chrome
		if showBody {
			need++
		}
		if lay.TranscriptH >= need {
			break
		}
		switch {
		case showRule:
			showRule = false
		case showBody:
			showBody = false
		default:
			if lay.TranscriptH < 1 {
				// The floor row replaces a frame row outright instead of sitting
				// inside a region, so it spans the frame and starts at column
				// zero: a notice centred in a twenty-cell box over the status bar
				// would leave the bar's own text showing on both sides of it,
				// reading as one row that says two things.
				//
				// Column zero is half of that claim and overlayRow is the other
				// half. Starting at zero only settles the left side; the notice is
				// as wide as its text and the bar finishes the row to the right
				// unless the row is padded to the span it is replacing. Both
				// halves are stated because for one review round only the first
				// was true.
				return overlayRow(m.modalNotice(lay.ContentW), lay.ContentW), overlayFloorTop(lay), 0
			}
			// In-region, the notice takes exactly the cells a bordered box would
			// have owned — hence the box width and not the notice's own.
			return overlayRow(m.modalNotice(w), w), top, left
		}
	}
	h := len(md.Lines) + chrome
	if h > lay.TranscriptH {
		h = lay.TranscriptH
	}

	title := " " + truncEnd(md.Title, inner-10, m.gly.Trunc) + " "
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

	// Body: whatever the chrome leaves. The floor covers a modal opened with no
	// lines at all — an empty box still shows one row rather than folding its
	// borders together — and the loop above has already reserved the row it
	// needs, so the floor cannot push the box past the region. At the bottom
	// rung the loop has withdrawn the body altogether and the floor does not
	// apply: that box is borders and a footer.
	bodyH := 0
	if showBody {
		if bodyH = h - chrome; bodyH < 1 {
			bodyH = 1
		}
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

	if showRule {
		box = append(box, m.sty.BorderFocus(m.gly.BoxLT+fill(m.gly.BoxH, w-2)+m.gly.BoxRT))
	}

	footer := fitWidest(m.modalFooters(md, showBody), inner+1, m.gly.Trunc)
	box = append(box, m.sty.BorderFocus(m.gly.BoxV)+" "+
		pad(m.sty.Muted(footer), inner+1)+
		m.sty.BorderFocus(m.gly.BoxV))
	box = append(box, m.sty.BorderFocus(m.gly.BoxBL+fill(m.gly.BoxH, w-2)+m.gly.BoxBR))
	return box, top, left
}

// modalFooters is the footer at descending detail, widest first.
//
// A single truncEnd is not enough here and the smallest supported terminal is
// where that shows: at 40 columns the help footer's full form cuts to
// "kirsch v0.1.0 · docs: doc/usage.md…" and the words naming the way out are
// the ones that go. The ladder drops the version and the docs pointer instead,
// because the exit key is the span this modal cannot be left without — the same
// reasoning that keeps the footer itself when the body goes.
//
// The last variant is the floor: it is the exit key alone, so a width that fits
// nothing else still fits the way out.
//
// scrollable is false at the rung where the height ladder has withdrawn the body
// altogether. A footer offering "j/k scroll · g/G top/bottom" over zero body rows
// names three bindings that do nothing, and an inert binding is worse than a
// short footer: it is the row the user trusts when the frame has stopped making
// sense. So that rung starts one variant down, at the exit key alone. The help
// family needs no such branch — none of its variants names a scroll binding.
//
// Every literal naming the exit key comes from modalExit, including the floors.
// Spelling it twice is how the footer and the notice row would come to disagree
// about which key leaves this modal.
func (m Model) modalFooters(md *ModalState, scrollable bool) []string {
	dot := " " + m.gly.Bullet + " "
	key := m.modalExit(md)
	if md.Kind == ModalHelp {
		return []string{
			"kirsch v" + m.sess.Version + dot + "docs: doc/usage.md" + dot + key + " closes",
			"kirsch v" + m.sess.Version + dot + key + " closes",
			key + " closes",
			key,
		}
	}
	exit := key + " close"
	if m.pendingApproval != 0 {
		exit = key + " back to approval"
	}
	full := []string{
		"j/k scroll" + dot + "g/G top/bottom" + dot + exit,
		"j/k scroll" + dot + exit,
		exit,
		key,
	}
	if !scrollable {
		return full[2:] // no body rows; the scroll bindings would be inert
	}
	return full
}

// modalExit names the key that leaves this modal, for the surfaces that have
// room for that and nothing else.
func (m Model) modalExit(md *ModalState) string {
	if md.Kind == ModalHelp {
		return "Esc or ?"
	}
	return "Esc"
}

// modalNotice is the bottom of the degradation ladder: a region of one or two
// rows cannot hold a bordered box with a footer inside it, so the modal folds
// to a single row naming itself and the key that closes it.
//
// It follows transcriptRows' "terminal too narrow" precedent — where a region
// cannot hold the real thing, it holds a line saying so rather than nothing —
// and it is the row that stops an open modal being indistinguishable from an
// idle session. The composer's caret is the only other difference between those
// two frames, and Layout invariant 4 keeps line counts identical with and
// without colour, so on a monochrome terminal a missing caret is not a cue.
//
// It returns one row, and says so in its type. It used to return a []string of
// one element and promise the count in prose, and it used to return none at all
// when the region had none, on the stated grounds that "statusRow carries it
// instead" — which statusRow does not and never did: it renders model, state,
// tokens, grants and warnings, and reads neither m.modal nor m.confirm. The
// comment was the only thing standing between an open modal and a frame that
// showed nothing, said `idle`, and swallowed every key. Where the row goes is
// the caller's to decide; that it exists is not.
//
// What comes back is the widest variant that FITS w, not a row of w cells. The
// difference is the caller's to close, through overlayRow, and overlayRow is
// where the reasoning for that division sits.
//
// It takes a width and no Layout: the row's content is a function of the cells
// it has, and the region height it used to consult was only ever used to decide
// whether to draw at all.
func (m Model) modalNotice(w int) string {
	md := m.modal
	exit := m.modalExit(md)
	dot := " " + m.gly.Bullet + " "
	return m.sty.BorderFocus(fitWidest([]string{
		md.Title + dot + exit + " closes",
		md.Title + dot + exit,
		exit,
	}, w, m.gly.Trunc))
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
	case ModalDiff:
		// Handled below, where the leading character selects the colour.
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
	return overlay(rows, box, top, left, lay.ContentW)
}

// overlayConfirm composites the one-line confirm prompt. ui-spec §4.3.
//
// The box sits at the foot of the transcript region, not below it. ui-spec §4.3
// calls the confirm prompt a modal and puts it inside §4, so Layout invariant 3
// binds it exactly as it binds the content, diff and help modals: the region is
// what an overlay may overwrite, and the status bar and composer are what it may
// not. Anchoring it to the frame's rule/status/rule band instead — which is what
// `lay.H - lay.ComposerH - 3` did — put the prompt over the status bar at every
// size, which is the one thing invariant 3 exists to forbid. Bottom-aligning it
// keeps it where it reads best, immediately above the composer it is asking
// about, without leaving the region to get there.
//
// It degrades the same way a modal does, for the same reason: a confirm prompt
// traps every key but y, n and Esc, so a region too small for the box gets the
// prompt as a bare row rather than nothing — and a region with no rows at all
// gets it on the frame's last row rather than nothing. This arm used to return
// the frame untouched, on the stated grounds that "statusRow carries it
// instead". statusRow does not: it renders model, state, tokens, grants and
// warnings, and reads m.confirm nowhere. At 40×10 — the advertised minimum,
// ui-spec §1 — a five-line draft in the composer left a region of no rows, and
// the frame that produced was a header, a status bar reading `idle`, and the
// draft: no question, no answer keys, and `y` still applying the action. The
// region is reserved in computeLayout now, so that frame is unreachable; the
// floor below it is what makes the claim unconditional rather than probable.
//
// Both forms compose their row through confirmRow. The height ladder picks
// between them; the width ladder lives in one place for both, because the first
// version of this gave the one-row form a ladder and left the bordered form on a
// bare truncEnd, and at 40 columns — the advertised minimum, ui-spec §1 — the
// rung the prompt degraded FROM named fewer answer keys than the rung it
// degraded TO. Two forms of one overlay cannot diverge on width if only one of
// them knows how width is spent.
func (m Model) overlayConfirm(rows []string, lay Layout) []string {
	c := m.confirm
	w := m.confirmBoxWidth(lay)
	left := (lay.ContentW - w) / 2
	inner := w - 4

	var box []string
	top := lay.HeaderH()
	switch {
	case lay.TranscriptH >= 3:
		box = []string{
			m.sty.BorderFocus(m.gly.BoxTL + fill(m.gly.BoxH, w-2) + m.gly.BoxTR),
			m.sty.BorderFocus(m.gly.BoxV) + " " +
				pad(m.sty.Text(m.confirmRow(c.Prompt, inner+1)), inner+1) +
				m.sty.BorderFocus(m.gly.BoxV),
			m.sty.BorderFocus(m.gly.BoxBL + fill(m.gly.BoxH, w-2) + m.gly.BoxBR),
		}
		top += lay.TranscriptH - len(box)
	case lay.TranscriptH >= 1:
		// The bare row takes exactly the cells the bordered form above it would
		// have owned, which is why the span is the box width and not the row's.
		box = overlayRow(m.sty.Text(m.confirmRow(c.Prompt, w)), w)
		top += lay.TranscriptH - len(box)
	default:
		// No region at all. The row leaves the region rather than going out;
		// overlayFloorTop carries the reasoning and the reachability. It spans
		// the frame and starts at column zero for the same reason the modal's
		// notice does: it is replacing a frame row, not sitting inside one — and
		// spanning it means padding to it, not merely starting at zero. See
		// overlayRow.
		box = overlayRow(m.sty.Text(m.confirmRow(c.Prompt, lay.ContentW)), lay.ContentW)
		top, left = overlayFloorTop(lay), 0
	}
	return overlay(rows, box, top, left, lay.ContentW)
}

// confirmMinW is the confirm box's width floor, and the reason confirmRow's
// "nothing left for the question" arm is unreachable through overlayConfirm.
const confirmMinW = 20

// confirmBoxWidth sizes the confirm box: wide enough for the widest row
// confirmRow can return, capped by the terminal, floored at confirmMinW. Past
// the cap the ladder inside confirmRow is what gives way.
//
// The cap is applied before the floor, in that order and deliberately not as
// one clamp call. clamp(v, lo, hi) returns hi when lo > hi, and lo > hi is
// reachable right here: at lay.ContentW == 20 the cap is eighteen and the floor is
// twenty, so the clamp returned eighteen and the floor lost at exactly the
// width it exists for. Nothing visibly broke — eighteen cells still fit "y/n"
// and a dozen of the question — which is the point. confirmRow documents its
// last arm as unreachable "through overlayConfirm, whose box floor is twenty
// cells", and a floor that yields to the cap is not a floor, it is a
// preference, and the sentence resting on it was false at one width.
//
// Written out, the floor wins and the box overhangs a terminal narrower than
// itself. That is what modalBox already does at the same width, and spliceAt
// and exactly are what absorb it.
//
// It is a named function and not four lines inline because the arithmetic is
// the whole of the rule: the order of the two bounds is the invariant, and an
// invariant that cannot be called cannot be tested except through a render.
func (m Model) confirmBoxWidth(lay Layout) int {
	w := cellWidth(m.confirm.Prompt+confirmGap+confirmKeysLong) + 4
	if w > lay.ContentW-2 {
		w = lay.ContentW - 2
	}
	if w < confirmMinW {
		w = confirmMinW
	}
	return w
}

// The confirm prompt's two key spellings and the gap that separates them from
// the question. Both name y and n; the long form only says so at more length.
const (
	confirmGap       = "  "
	confirmKeysLong  = "[y] yes   [n] no"
	confirmKeysShort = "y/n"
)

// confirmRow composes a confirm prompt and its answer keys into w cells.
//
// The keys are never dropped and the question is never dropped either — what
// gives way is first the keys' spelling, then the tail of the question. That
// order is the whole of the ladder's reasoning:
//
//   - The keys outrank the question, because a row that has lost "y/n" leaves an
//     overlay that traps every key and names none that answers it. This is the
//     reverse of where truncation naturally falls, since the keys sit at the END
//     of the row and truncEnd takes the end.
//   - The question outranks the keys' spelling, because "[y] yes   [n] no" and
//     "y/n" name the same two keys and the difference between them is only
//     friendliness, while the difference between a truncated question and a
//     dropped one is whether the user can see what they are agreeing to.
//
// So the long spelling is spent first, and only then does the question start to
// truncate. At the smallest supported terminal — 40 columns, of which the frame
// keeps two for its margin — the longest prompt the app raises reads
// "Discard the running turn an⋯  y/n": cut, but both answerable and legible.
//
// Esc is not named. ui-spec §4.3 gives the prompt three keys, but n and Esc do
// the same thing — both clear the prompt with no effect — so Esc is an alias for
// a key the row already names, and the cells it would cost at 40 columns come
// straight out of the question. Naming the way out is an obligation here as much
// as on a modal footer; "n" is what discharges it.
func (m Model) confirmRow(prompt string, w int) string {
	for _, keys := range []string{confirmKeysLong, confirmKeysShort} {
		if cellWidth(prompt)+cellWidth(confirmGap)+cellWidth(keys) <= w {
			return prompt + confirmGap + keys
		}
	}
	room := w - cellWidth(confirmGap) - cellWidth(confirmKeysShort)
	if room < 1 {
		// Narrower than the keys and their gap. Nothing is left to spend on the
		// question, so the row is the two answers alone — the one span it is not
		// allowed to lose. Unreachable through overlayConfirm, whose box floor is
		// twenty cells, and stated here rather than assumed.
		return truncEnd(confirmKeysShort, w, m.gly.Trunc)
	}
	return truncEnd(prompt, room, m.gly.Trunc) + confirmGap + confirmKeysShort
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
	// Milestone 1 scaffolding, removed in M3 when the model drives tools.
	// Labelled so nobody mistakes them for product surface.
	right = append(right, "", "debug (M1 only)")
	for i := 0; i < len(DebugCommands); i += 2 {
		right = append(right, strings.Join(DebugCommands[i:minInt(i+2, len(DebugCommands))], "  "))
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
