package tui

import (
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
)

// Binding documents one key. The help overlay is built from this table, so the
// two cannot drift apart.
type Binding struct{ Key, Desc string }

// BindingGroup is one mode's bindings.
type BindingGroup struct {
	Mode     string
	Bindings []Binding
}

// bindingGroups is what the help overlay renders, mode by mode. ui-spec §5.2.
//
// It is the authority on what help *says*, not on what the program accepts.
// Keys are handled and deliberately unlisted — Home and End and their g/G
// equivalents in the transcript among them, and Esc in more modes than the grid
// shows — because the grid itself is pinned by screen 06 in
// plan/kirsch-ui-screens.md, so adding a row here is a spec amendment rather
// than a code change. Read a missing row as "not documented yet", never as
// "not bound": the handlers in this file are what decide that.
//
// Quitting has no key row because it has no key. A bare `q` used to quit from
// an empty composer and from the transcript; both are gone, because a key that
// ends the session on the first character of "query" is a key that ends it by
// accident. /quit and /exit are the whole surface, and they are listed in the
// commands column rather than here.
var bindingGroups = []BindingGroup{
	{"composing", []Binding{
		{"Enter", "send"},
		{"Shift+Enter", "newline"},
		{"Ctrl+J", "newline (alt)"},
		{"Tab", "complete /cmd"},
		{"↑ at line 1", "browse"},
		{"Esc", "cancel turn"},
	}},
	{"browsing", []Binding{
		{"↑/↓", "select card"},
		{"PgUp/PgDn", "scroll"},
		{"Enter", "expand"},
		{"d", "diff / content"},
		{"End", "bottom, re-pin"},
		{"?", "help"},
	}},
	{"approval", []Binding{
		{"y", "approve"},
		{"a", "+ session"},
		{"n", "reject"},
		{"d", "detail"},
	}},
	{"modal", []Binding{
		{"j/k ↑/↓", "scroll"},
		{"g/G", "top/bottom"},
		{"Esc", "close"},
	}},
}

// NormalizeInput builds the input reader for the Bubble Tea program, so that
// SS3-encoded Home and End reach it as the CSI forms it can decode.
//
// in is the process's standard input; ownership stays with the caller, because
// nothing here reads, writes or closes it. The return is the value to hand to
// tea.WithInput — or nil when in is nil or is not a terminal, and nil here means
// "install no input option at all", never "call tea.WithInput(nil)", which is
// Bubble Tea's idiom for disabling input entirely. Leaving the option off is
// what preserves Bubble Tea's own default path, which opens /dev/tty when stdin
// has been piped or redirected; tea.WithInput takes that fallback away, so a
// piped invocation wired through it draws a frame and then never sees a
// keystroke.
//
// Bubble Tea v1.3.10's key table carries `\x1b[1~`, `\x1b[H` and `\x1b[7~` for
// Home and `\x1b[4~`, `\x1b[F` and `\x1b[8~` for End, and it carries the SS3
// arrows `\x1bOA`–`\x1bOD`. It carries no `\x1bOH` and no `\x1bOF`. macOS
// Terminal's xterm-256color terminfo sets khome=\EOH and kend=\EOF, so on that
// terminal Home and End are exactly the two SS3 forms the table is missing —
// which is why the arrows work there and Home and End do not.
//
// An unmatched `\x1bOH` does not simply get dropped. Bubble Tea's fallback
// treats the ESC as an Alt prefix and the `O` as the key, so one Home keypress
// is delivered as two messages: alt+O, then a literal `H`. In the transcript
// pane both are no-ops; in the composer the stray rune is typed into the text.
// Rewriting the byte is what makes tea.KeyHome and tea.KeyEnd arrive at all, so
// the handlers keyed on them — and the textarea's own line-start and line-end
// bindings — start working without any of them changing.
//
// The rewrite is done on the bytes rather than by pairing the two messages back
// up in handleKey, because that pairing needs state on the Model, cannot tell a
// real alt+O followed by an H from a Home keypress, and would hold the latch
// open indefinitely between the two. Teaching Bubble Tea the two sequences
// instead is not on offer: its table is an unexported package variable built at
// init with no hook to extend it.
//
// The parameter is *os.File rather than io.Reader because Bubble Tea decides
// what a program's input can do by type-asserting it, so a wrapper that narrows
// the input to io.Reader silently takes two capabilities away:
//
//   - tty_unix.go asserts to term.File before calling term.MakeRaw. A
//     non-File input leaves the terminal canonical and echoing: keystrokes
//     arrive only on Enter, characters echo over the alt screen, and Ctrl+C
//     comes back as SIGINT rather than as a key.
//   - cancelreader.NewReader asserts to cancelreader.File, and the fallback it
//     returns otherwise has a Cancel that always reports failure, so every quit
//     and suspend burns Bubble Tea's full read-loop timeout with the read
//     goroutine still blocked.
//
// ss3File therefore embeds the file and overrides Read alone, and the two
// assertions beside it fail the build if that ever stops being true.
//
// Known gaps. Each degrades to the behaviour that was there before this
// function existed rather than to something worse, and the list is what has
// been found rather than a proof that nothing else is missing:
//   - A sequence split across two Read calls is not rewritten. Nothing is
//     buffered on purpose: holding a trailing ESC back to see what follows
//     would delay every bare Esc, and Esc cancels a turn, closes a modal and
//     leaves the transcript pane. A single keypress arrives in one read in
//     practice.
//   - The rewrite is not suppressed inside a bracketed paste, so a paste
//     containing the literal bytes ESC O H would have them altered. Pasting raw
//     escape bytes is already not meaningful input.
//   - A piped or redirected invocation is not covered at all, which is the
//     widest gap listed here. The nil above is what preserves Bubble Tea's own
//     fallback, and that fallback opens /dev/tty inside initInput, where nothing
//     can wrap it. So `kirsch < file` and `echo … | kirsch` do reach a real
//     terminal for input, and still deliver Home as alt+O plus a stray H —
//     exactly as every invocation did before this function existed. A gap in the
//     fix, not a regression from it, and easy to read as covered because the two
//     sentences above are about the same fallback working correctly.
//
// That last one is closable rather than blocked. NormalizeInput could open
// /dev/tty itself when stdin is not a terminal and return that wrapped, which is
// the same file Bubble Tea would otherwise have opened. The wrapper survives the
// trip: cancelreader's BSD path special-cases file.Name() == "/dev/tty" to avoid
// kqueue, and Name is promoted from the embedded *os.File, so that check still
// resolves through ss3File. What it costs is an owned descriptor to close and a
// second failure path on a startup that has none today, so it is left for a
// change that can carry both.
func NormalizeInput(in *os.File) io.Reader {
	if in == nil || !term.IsTerminal(in.Fd()) {
		return nil
	}
	return &ss3File{File: in}
}

// Bubble Tea reaches for both of these by type assertion, and an assertion that
// fails is silent — it selects a degraded path rather than returning an error.
// term.File gates raw mode; cancelreader.File gates cancellable reads. Asserting
// against the library's own interfaces rather than a local copy of their shape
// is deliberate: if a Bubble Tea upgrade widens either one, this stops compiling
// instead of quietly losing the capability again.
var (
	_ term.File         = (*ss3File)(nil)
	_ cancelreader.File = (*ss3File)(nil)
)

// ss3File is NormalizeInput's reader: the input file with Read overridden.
//
// The file is embedded rather than held in a named io.Reader field so that Fd,
// Name, Write and Close all reach the real descriptor. That is what keeps the
// value a term.File and a cancelreader.File — see NormalizeInput for what each
// of those gates — and it is also what keeps the rewrite on the read path under
// the kqueue cancel reader, which polls the descriptor it was handed but takes
// its bytes through this Read.
type ss3File struct{ *os.File }

// Read fills p from the input file and rewrites SS3 Home and End in place.
//
// It returns the file's own count and error unchanged: the rewrite never alters
// the length, so it cannot short-read or stall the caller. The n > 0 guard is
// there because p[:n] panics on a negative count, and a reader misbehaving
// upstream should not become a panic inside Bubble Tea's input goroutine.
func (s *ss3File) Read(p []byte) (int, error) {
	n, err := s.File.Read(p)
	if n > 0 {
		normalizeSS3(p[:n])
	}
	return n, err
}

// normalizeSS3 rewrites ESC O H into ESC [ H and ESC O F into ESC [ F in place.
//
// Only Home and End are touched. The SS3 arrows and SS3 F1–F4 are left alone
// because Bubble Tea already decodes those, and rewriting a sequence that
// already works is how a second bug gets introduced to fix the first.
func normalizeSS3(b []byte) {
	for i := 0; i+2 < len(b); i++ {
		if b[i] == 0x1b && b[i+1] == 'O' && (b[i+2] == 'H' || b[i+2] == 'F') {
			b[i+1] = '['
			i += 2 // step over the sequence just rewritten
		}
	}
}

// Messages driving time-based behaviour. Every one of them is an explicit
// message rather than a clock read, which is what keeps View() deterministic.
type (
	spinnerTickMsg  struct{}
	spinnerStartMsg struct{}
	streamTickMsg   struct{}
)

func tickSpinner() tea.Cmd {
	return tea.Tick(SpinnerFrame, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

// tickSpinnerOnce starts a tick chain only if one is not already running.
//
// Without the guard every ToolStartedMsg started a second chain alongside the
// one Init began, so the spinner ran at twice the rate after the first tool
// call and faster still after the next — and each chain kept a pending Cmd
// alive for the life of the program.
func (m *Model) tickSpinnerOnce() tea.Cmd {
	if m.spinnerAlive {
		return nil
	}
	m.spinnerAlive = true
	return tickSpinner()
}

func tickStream() tea.Cmd {
	return tea.Tick(StreamCoalesce, func(time.Time) tea.Msg { return streamTickMsg{} })
}

// Update is the single entry point for all state change. architecture.md §3
// rule 4: TUI state is mutated only here.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.relayout(m.layout())
		return m, nil

	case spinnerTickMsg:
		m.frame++
		return m, tickSpinner()

	case spinnerStartMsg:
		return m, m.tickSpinnerOnce()

	case streamTickMsg:
		return m.advanceFake()

	case WorkspaceInfoMsg:
		m.sess.Project, m.sess.Branch, m.sess.Dirty = msg.Project, msg.Branch, msg.Dirty
		m.projectTypes = msg.Types
		m.relayout(m.layout())
		return m, nil

	case ToolStartedMsg:
		id := m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
			Name: msg.Name, Target: msg.Target, State: StateRunning,
		}})
		m.toolCards[msg.ID] = id
		m.busy = Busy{Active: true, Verb: msg.Name}
		m.relayout(m.layout())
		return m, m.tickSpinnerOnce()

	case ToolCompletedMsg:
		return m.applyToolResult(msg)

	case ErrorMsg:
		m.appendBlock(Item{Kind: KindError, Err: &ErrorCard{
			Kind: msg.Kind, Message: msg.Message,
			Hint:   "Enter to expand",
			Detail: SanitizeLines(msg.Detail),
		}})
		m.busy = Busy{}
		m.relayout(m.layout())
		return m, nil

	case NoticeMsg:
		m.notice(msg.Text)
		m.relayout(m.layout())
		return m, nil

	case tea.KeyMsg:
		lay := m.layout()
		m.relayout(lay)
		return m.dispatchKey(msg, lay)
	}
	return m, nil
}

// dispatchKey runs one keypress and rebuilds the transcript cache when that
// keypress invalidated it.
//
// Update relayouts once, before the key is dispatched, against the layout the
// model had *before* the key. Three things a handler does can make that stale,
// and each leaves the frame View is about to draw disagreeing with the state the
// model reports:
//
//   - The overlay state changed. relayout flattens the transcript with a dimmed
//     palette while a modal or confirm is open, so a handler that opens or closes
//     one invalidates the very cache it is about to be drawn from. Closing an
//     overlay left the whole window dimmed with nothing drawn over it until the
//     next keypress rebuilt the cache, which reads as "Esc twice to get the
//     colours back"; opening one left the strip of transcript beside the box at
//     full brightness for exactly as long.
//   - The layout changed. The composer's height is part of the geometry — a line
//     added or removed, a hint row appearing or going away — and it sets where
//     the transcript's bottom edge falls. View computes the layout fresh, so a
//     handler that changes the composer's height and does not rebuild leaves the
//     pinned offset measured against a viewport that no longer exists: the model
//     still reports Pinned while the bottom line of the transcript sits below the
//     fold.
//   - The gutter moved. relayout draws the selection gutter beside whichever item
//     is m.sel or m.pendingApproval, and takes its two columns out of that item's
//     content width — so moving either one rewraps two cards and cannot be a
//     cache hit. The two halves of this check are not the same kind of thing. The
//     m.sel half fixes a live defect: keyComposing's Up arm moves the selection
//     through setBase, which does not relayout, and neither does the
//     revealSelection call beside it — so entering Browsing drew a frame with no
//     gutter until some later keypress happened to rebuild.
//     TestEnteringBrowsingDrawsTheGutterInTheSameFrame pins exactly that, and
//     removing this half of the check fails it. The m.pendingApproval half is
//     genuine belt-and-braces: every path that moves it today relayouts behind
//     the call, so removing that half alone breaks nothing. It is here because it
//     is the identical shape, and the next handler to move it is the one that
//     will forget.
//
// All three checks live here rather than at the handlers that trip them, and all
// three key off exactly what relayout and View consume, so a path added later
// cannot reintroduce any of the defects by forgetting. A handler is still free to
// relayout for its own reasons; doing so simply makes the comparison below find
// nothing to do.
func (m Model) dispatchKey(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	before := m.overlayOpen()
	next, cmd := m.handleKey(k, lay)
	// Every handler returns a Model by construction; the comma-ok form degrades
	// to the un-rebuilt frame rather than panicking inside the render loop if
	// that ever stops being true.
	nm, ok := next.(Model)
	if !ok {
		return next, cmd
	}
	// Layout is a plain value of ints and bools, so == compares the whole
	// geometry rather than the composer height alone.
	after := nm.layout()
	if nm.overlayOpen() == before && after == lay &&
		nm.sel == m.sel && nm.pendingApproval == m.pendingApproval {
		return next, cmd
	}
	nm.relayout(after)
	return nm, cmd
}

// applyToolResult folds a finished tool into its running card.
//
// The card is mutated in place through the transcript's terminal-state guard
// rather than appended fresh, so the running card the user is already looking
// at becomes the completed one — ui-spec §3.8's rule that a card is never
// rewritten once *terminal* is about final states, not about this transition.
func (m Model) applyToolResult(msg ToolCompletedMsg) (tea.Model, tea.Cmd) {
	cardID, known := m.toolCards[msg.ID]
	if !known {
		cardID = m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
			Name: msg.Name, Target: msg.Target, State: StateRunning,
		}})
	}
	delete(m.toolCards, msg.ID)

	m.tr.MutateTool(cardID, func(c *ToolCard) {
		c.Elapsed = msg.Elapsed
		c.Out = SanitizeLines(msg.Content)
		switch {
		case msg.OK:
			c.State = StateOK
			c.Summary = msg.Summary
		case msg.ErrorKind == "cancelled":
			c.State = StateCancelled
			c.Summary = msg.ErrorMsg
		default:
			c.State = StateError
			c.Summary = msg.ErrorMsg
		}
		if msg.Truncated {
			c.Trunc = "truncated"
		}
	})

	// A workspace violation is a tool error the model could handle, so it stays
	// a tool card with ✗ rather than being promoted to a red error card.
	// architecture.md §8; ui-spec §3.6.
	m.busy = Busy{}
	m.sel = cardID
	m.relayout(m.layout())
	return m, nil
}

// layout resolves this model's frame geometry. It is the only place the overlay
// flag is supplied, so View and every handler read the same region height.
func (m Model) layout() Layout {
	return computeLayout(m.width, m.height, m.comp.contentLines(), m.overlayOpen())
}

// handleKey dispatches to exactly one mode handler.
//
// There is no fallback arm and no shared pre-pass beyond the force-quit check,
// so a key cannot be handled twice. In particular a tea.KeyMsg reaches the
// textarea through exactly one call — ta.Update at the foot of keyComposing,
// the only ta.Update in the package — which is what stops every other mode
// from leaking keystrokes into the composer. ui-spec §5.2.
//
// That is narrower than "the textarea is only written from keyComposing", and
// deliberately so: handlePaste and applyConfirm's ConfirmLargePaste arm both
// write it through InsertString from outside. They carry literal text, never
// keys, so they leave the rule above intact.
func (m Model) handleKey(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	// Bracketed paste is literal text, never keys. ui-spec §7.2. On Bubble Tea
	// v1 it arrives as a KeyMsg with Paste set; there is no tea.PasteMsg.
	if k.Paste {
		return m.handlePaste(string(k.Runes))
	}

	// The only key allowed to bypass mode dispatch: the double-Ctrl+C force
	// quit, which fires from any mode. A *single* Ctrl+C does not bypass —
	// its meaning is per-mode, so it falls through. ui-spec §5.3.
	if k.Type == tea.KeyCtrlC {
		now := m.now()
		if !m.lastCtrlC.IsZero() && now.Sub(m.lastCtrlC) < DoubleInterrupt {
			return m, tea.Quit
		}
		m.lastCtrlC = now
	}

	switch m.mode() {
	case ModeModal:
		return m.keyModal(k, lay)
	case ModeApprovalPending:
		return m.keyApproval(k, lay)
	case ModeConfirm:
		return m.keyConfirm(k, lay)
	case ModeBrowsing:
		return m.keyBrowsing(k, lay)
	default:
		return m.keyComposing(k, lay)
	}
}

func (m Model) keyComposing(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		if m.busy.Active {
			return m.cancelTurn(), nil
		}
		m.comp.Reset()
		return m, nil

	case tea.KeyEnter:
		// ESC+CR arrives here as Alt=true. Terminals send it for Shift+Enter,
		// for Alt+Enter, or for neither — which one a given terminal produces
		// is not something Kirsch gets to decide, so it accepts the sequence
		// rather than the key name. Ctrl+J below is the always-representable
		// route. ui-spec §5.2, plan amendments 29 and 40.
		if k.Alt {
			m.comp.ta.InsertString("\n")
			return m, nil
		}
		if m.busy.Active {
			return m, nil
		}
		return m.submit(lay)

	case tea.KeyCtrlJ:
		m.comp.ta.InsertString("\n")
		return m, nil

	case tea.KeyTab:
		if cmd, _, ok := parseSlash(m.comp.Value()); ok {
			if full, unique := completeSlash(cmd); unique {
				m.comp.SetValue("/" + full)
			}
		}
		return m, nil

	case tea.KeyCtrlG:
		// The way back to the bottom without leaving the composer and without
		// sending anything. Before this, every re-pin reachable from Composing
		// put something in the transcript first — sending a message, or
		// submitting a slash command — so a reader who scrolled up and then
		// decided not to send after all had no single key that returned the
		// view. ↑ then End did it in two, by way of a mode they did not want.
		//
		// Ctrl+G rather than End or Ctrl+End. All three spell "go to the
		// bottom"; the other two are already spoken for here, because bubbles
		// v1.0.0 binds End to LineEnd and Ctrl+End to InputEnd (textarea.go:88
		// and :91). Taking either would buy a scroll at the price of a cursor
		// movement in a composer that can be several lines tall. Ctrl+G is also
		// the sturdiest of the three on the wire: it is BEL, a single C0 byte,
		// where Ctrl+End is a modified-key escape sequence. This package already
		// carries evidence that the plain forms of those keys are not decoded
		// everywhere — normalizeSS3 above exists because macOS Terminal sends
		// bare Home and End as SS3, which Bubble Tea v1.3.10 does not read. That
		// is about the unmodified keys, so it does not by itself prove anything
		// about Ctrl+End; what it establishes is that this family of keys is
		// where terminal disagreement actually lands, and a single control byte
		// sidesteps the question. Bare `G` was never a candidate: it has to stay
		// an ordinary character while typing, and
		// TestGAndCapitalGAreLiteralInTheComposer holds that line.
		//
		// repin() sets the pin and no offset — relayout derives the offset from
		// the pin — so the two are called together, in this order, exactly as
		// both Browsing arms do it. Split them and the model reports itself
		// pinned while the frame still shows the old position.
		//
		// Deliberately ahead of the busy guard below: this moves the viewport
		// rather than entering text, and a live turn streaming into the
		// transcript is when getting back to the bottom is worth the most.
		//
		// ui-spec §2.4 lists no re-pin key reachable from Composing and §5.2's
		// composing table has no row for this one. Amending both is tracked
		// separately from this change.
		m.scroll.repin() // re-pin: Ctrl+G is "go to the bottom" from the composer
		m.relayout(lay)
		return m, nil

	case tea.KeyUp:
		if m.comp.ta.Line() == 0 {
			m.setBase(BaseBrowsing)
			m.revealSelection(lay.TranscriptH)
			return m, nil
		}
	}

	if m.busy.Active {
		return m, nil // composer is input-disabled while a turn is live, §2
	}

	// Everything else reaches the textarea — including `?`, which is a literal
	// character here and NOT the help key. A help overlay that fires mid-
	// sentence makes the composer unusable. ui-spec §5.1.
	var cmd tea.Cmd
	m.comp.ta, cmd = m.comp.ta.Update(k)
	m.comp.Hint = ""
	return m, cmd
}

func (m Model) keyBrowsing(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		// Esc hands focus back to the composer and leaves the viewport exactly
		// where it is. It used to re-pin as well, and that made §2.4's "typing
		// does not re-pin" unreachable in practice: Esc is the only way back to
		// the composer that does not first walk the selection past the last
		// card, so scrolling up to read something was undone by the very act of
		// going to type about it. The remaining triggers are End, sending, and
		// `/new`. ui-spec §2.4 and §5.2 both still list Esc; amending them is
		// tracked separately from this change.
		m.setBase(BaseComposing)
		return m, nil

	case tea.KeyUp:
		if id, ok := m.tr.nextSelectable(m.sel, -1); ok {
			m.sel = id
			m.relayout(lay)
			m.revealSelection(lay.TranscriptH)
		}
		return m, nil

	case tea.KeyDown:
		if id, ok := m.tr.nextSelectable(m.sel, +1); ok {
			m.sel = id
			m.relayout(lay)
			m.revealSelection(lay.TranscriptH)
			return m, nil
		}
		m.setBase(BaseComposing) // past the last card: focus returns to the composer
		return m, nil

	case tea.KeyPgUp:
		// Scrolls without moving the selection. ui-spec §2.4.
		m.scroll.setOffset(m.scroll.Offset-maxInt(1, lay.TranscriptH-1), len(m.lines), lay.TranscriptH)
		return m, nil

	case tea.KeyPgDown:
		m.scroll.setOffset(m.scroll.Offset+maxInt(1, lay.TranscriptH-1), len(m.lines), lay.TranscriptH)
		return m, nil

	case tea.KeyHome:
		m.scroll.setOffset(0, len(m.lines), lay.TranscriptH)
		return m, nil

	case tea.KeyEnd:
		m.scroll.repin() // re-pin: End returns to the bottom
		m.relayout(lay)
		return m, nil

	case tea.KeyEnter:
		if m.sel != 0 {
			m.expanded[m.sel] = !m.expanded[m.sel]
			m.relayout(lay)
			m.revealSelection(lay.TranscriptH)
		}
		return m, nil

	case tea.KeyCtrlC:
		return m, nil
	}

	switch string(k.Runes) {
	case "d":
		return m.openDetail(lay), nil

	// g/G mirror the modal's own g/G rather than inventing a second vocabulary
	// for the same gesture, and they are the keyboard-reachable form of Home and
	// End: on macOS Terminal those two arrive as SS3, which Bubble Tea v1.3.10
	// does not decode — see normalizeSS3. They are deliberately absent from
	// bindingGroups: that table renders the help overlay, whose grid is pinned by
	// screen 06 in plan/kirsch-ui-screens.md, so listing them there is a spec
	// amendment rather than a code change.
	case "g":
		m.scroll.setOffset(0, len(m.lines), lay.TranscriptH)
		return m, nil
	case "G":
		m.scroll.repin() // re-pin: G is "go to the bottom", the same intent as End
		m.relayout(lay)
		return m, nil

	case "?":
		m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
		return m, nil
	}
	return m, nil
}

func (m Model) keyApproval(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyEsc || k.Type == tea.KeyCtrlC {
		return m.resolveApproval(Rejected, lay), nil
	}
	switch string(k.Runes) {
	case "y":
		return m.resolveApproval(Approved, lay), nil
	case "a":
		// Inert unless the card offers a grant. Enforced here and not only in
		// the renderer — otherwise a keyboard user could grant a patch. ADR 0006.
		if it, _, ok := m.tr.Find(m.pendingApproval); ok && it.Approval.OffersSessionGrant() {
			return m.resolveApproval(ApprovedSession, lay), nil
		}
		return m, nil
	case "n":
		return m.resolveApproval(Rejected, lay), nil
	case "d":
		m.sel = m.pendingApproval
		return m.openDetail(lay), nil
	case "?":
		m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
		return m, nil
	}
	// Every other key is swallowed, never forwarded to the composer. §5.2.
	return m, nil
}

func (m Model) keyConfirm(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	if k.Type == tea.KeyEsc {
		m.confirm = nil
		return m, nil
	}
	switch string(k.Runes) {
	case "y":
		c := m.confirm
		m.confirm = nil
		return m.applyConfirm(c, lay), nil
	case "n":
		m.confirm = nil
		return m, nil
	}
	return m, nil
}

// keyModal scrolls the open modal. ui-spec §4.
//
// Home and End sit alongside g/G rather than being left to them. The pair used
// to be unreachable on macOS Terminal, where both arrive as SS3 — which is why
// g/G exist at all — and NormalizeInput now delivers them as tea.KeyHome and
// tea.KeyEnd everywhere. A key that works in the transcript and does nothing in
// a modal reads as a broken modal, not as a deliberate omission.
func (m Model) keyModal(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	page := maxInt(1, lay.TranscriptH-4)
	switch k.Type {
	case tea.KeyHome:
		m.modal.Off = 0
		return m, nil
	case tea.KeyEnd:
		m.modal.Off = maxInt(0, len(m.modal.Lines)-1)
		return m, nil
	case tea.KeyEsc, tea.KeyCtrlC:
		// Returns to the *previous* mode, which the derivation supplies for
		// free: close the modal and mode() re-evaluates. From an approval that
		// means back to the pending approval, not the composer. ui-spec §4.
		m.closeModal()
		return m, nil
	case tea.KeyUp:
		m.modal.Off = maxInt(0, m.modal.Off-1)
		return m, nil
	case tea.KeyDown:
		m.modal.Off++
		return m, nil
	case tea.KeyPgUp:
		m.modal.Off = maxInt(0, m.modal.Off-page)
		return m, nil
	case tea.KeyPgDown:
		m.modal.Off += page
		return m, nil
	}
	switch string(k.Runes) {
	case "k":
		m.modal.Off = maxInt(0, m.modal.Off-1)
	case "j":
		m.modal.Off++
	case "g":
		m.modal.Off = 0
	case "G":
		m.modal.Off = maxInt(0, len(m.modal.Lines)-1)
	case "?":
		if m.modal.Kind == ModalHelp {
			m.closeModal() // ? also closes help, §4.2
		}
	}
	return m, nil
}

// handlePaste inserts literal text, warning first when it is large. §7.2.
func (m Model) handlePaste(s string) (tea.Model, tea.Cmd) {
	if m.mode() != ModeComposing {
		// A bracketed paste arriving under an open overlay is DISCARDED, not
		// queued: an overlay traps every key, and text that landed in a textarea
		// the user cannot see would appear in the composer on close with nothing
		// naming where it came from. Silent is the right behaviour and the wrong
		// word for it to go unwritten — a reader looking for where the paste went
		// should find the answer here rather than infer it from mode().
		return m, nil
	}
	if len(s) > PasteWarnBytes {
		m.askConfirm(ConfirmState{
			Prompt:  "Paste " + itoa(len(s)/1024) + "KB into the composer?",
			Action:  ConfirmLargePaste,
			Payload: s,
		})
		return m, nil
	}
	m.comp.InsertString(s)
	return m, nil
}

func (m Model) applyConfirm(c *ConfirmState, lay Layout) Model {
	switch c.Action {
	case ConfirmLargePaste:
		m.comp.InsertString(c.Payload)
	case ConfirmNewSession:
		m.tr = Transcript{}
		m.sel, m.pendingApproval = 0, 0
		m.expanded = map[ItemID]bool{}
		m.busy = Busy{}
		m.scroll.repin() // re-pin: a new session starts at the bottom
		// No relayout here: every path that reaches this arm already rebuilds
		// behind the call. Today that is runSlash's idle branch, which resets in
		// place and falls through to its own relayout with the same lay, so a
		// rebuild here would do the same work twice for every reset.
		//
		// The other route in — askConfirm's "/new while a turn is in flight"
		// prompt, whose accept would rebuild via dispatchKey's repaint — cannot
		// currently execute: keyComposing swallows Enter while m.busy.Active, so
		// submit never reaches runSlash during a live turn and that prompt is
		// never raised. It is named here because the branch is still in the code
		// above; do not read it as a live caller.
	case ConfirmClearGrants:
		m.status.Grants = 0
	}
	return m
}

// openDetail opens the content or diff modal for the selected card.
func (m Model) openDetail(lay Layout) Model {
	it, _, ok := m.tr.Find(m.sel)
	if !ok {
		return m
	}
	switch it.Kind {
	case KindTool:
		m.openModal(ModalState{
			Kind: ModalContent, Title: it.Tool.Name + " " + it.Tool.Target,
			Source: it.ID, Lines: it.Tool.Out,
		})
	case KindApproval:
		m.openModal(ModalState{
			Kind: ModalDiff, Title: "calc/divide.go", Source: it.ID,
			Lines: it.Approval.Diff, Added: it.Approval.Added, Removed: it.Approval.Removed,
		})
	case KindError:
		m.openModal(ModalState{
			Kind: ModalContent, Title: it.Err.Kind, Source: it.ID, Lines: it.Err.Detail,
		})
	default:
		// User and assistant text, notices and thinking cards have no detail
		// view: `d` on them is a no-op rather than an empty modal.
	}
	return m
}

func (m Model) resolveApproval(o ApprovalOutcome, lay Layout) Model {
	it, _, ok := m.tr.Find(m.pendingApproval)
	if !ok {
		m.pendingApproval = 0
		return m
	}
	it.Approval.Outcome = o
	it.Approval.Elapsed = 2400 * time.Millisecond
	if o == ApprovedSession {
		m.status.Grants++
	}
	m.pendingApproval = 0
	m.tr.rev++
	m.relayout(lay)
	return m
}

func (m Model) cancelTurn() Model {
	if m.Cancel != nil {
		m.Cancel()
	}
	m.busy = Busy{}
	for _, it := range m.tr.Items() {
		if it.Kind == KindTool && it.Tool.State == StateRunning {
			it.Tool.State = StateCancelled
		}
		if it.Kind == KindAssistant && it.Text.Streaming {
			it.Text.Streaming = false
			it.Text.Cancelled = true
		}
	}
	return m
}

// submit sends the composer's contents.
//
// The user's own message goes through appendBlock like anything else, so from a
// scrolled-up composer it briefly takes the "n new" count — and the re-pin two
// lines below clears it again within the same keystroke. That ordering is the
// §2.4 rule rather than an accident of it: sending is a re-pin trigger, so by
// the time the frame is drawn the viewport is at the bottom and there is
// nothing unread below the fold to point at.
func (m Model) submit(lay Layout) (tea.Model, tea.Cmd) {
	v := strings.TrimRight(m.comp.Value(), "\n")
	if strings.TrimSpace(v) == "" {
		return m, nil
	}
	if cmd, args, ok := parseSlash(v); ok {
		m.comp.Reset()
		return m.runSlash(cmd, args, lay)
	}
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: SanitizeLines(v)}})
	m.comp.Reset()
	m.scroll.repin() // re-pin: sending returns to the bottom
	m.busy = Busy{Active: true, Verb: "thinking"}
	m.fake.begin(&m)
	m.relayout(lay)
	return m, tickStream()
}

// runSlash executes one slash command. ui-spec §6.
//
// The re-pin sits on the shared tail rather than in submit or in each arm.
// submit would be wrong because it cannot see which arm ran: the paths that only
// set m.comp.Hint — an unknown command, or a debug command missing its argument
// — produce nothing in the transcript, and their answer is already on screen in
// the composer. Re-pinning those would throw away a scroll position to report a
// typo. Per-arm would be wrong the other way: it is one fact about slash
// commands, not a rule every arm has to remember, and the next arm added is the
// one that would forget it.
//
// So the tail covers every arm that produces something — a notice, a modal, a
// confirm, or a tool invocation whose card arrives later — and the hint-only
// paths return before reaching it. ui-spec §2.4 lists sending a message as a
// re-pin trigger and §3.1 renders a slash invocation as a message, so the two
// agree.
//
// Returning early is not by itself the distinction: /quit returns early too, and
// re-pinning a program that is on its way out would be pointless rather than
// wrong. What the hint-only paths have in common is that their whole answer is
// already on screen in the composer, so moving the viewport would cost the
// reader their place to tell them about a typo. Any arm added here that puts
// something in the transcript belongs on the tail.
//
// Neither hint-only path relayouts, and neither needs to: setting a hint adds a
// composer row and so changes the geometry, and dispatchKey rebuilds whenever
// the layout a handler leaves behind differs from the one it was given.
//
// /help and /diff need nothing extra: they open a modal over a transcript that
// is now pinned underneath, so closing it lands at the bottom. /new needs
// nothing extra either — applyConfirm re-pins for its own reason (a new session
// starts at the bottom) and this re-pin also covers the branch where /new raises
// a confirm instead of resetting.
func (m Model) runSlash(cmd, args string, lay Layout) (tea.Model, tea.Cmd) {
	switch cmd {
	case "help":
		m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
	case "quit", "exit":
		return m, tea.Quit
	case "new":
		if m.busy.Active {
			m.askConfirm(ConfirmState{
				Prompt: "Discard the running turn and start a new session?",
				Action: ConfirmNewSession,
			})
		} else {
			m = m.applyConfirm(&ConfirmState{Action: ConfirmNewSession}, lay)
		}
	case "approvals":
		if m.status.Grants > 0 {
			m.askConfirm(ConfirmState{Prompt: "Clear all session grants?", Action: ConfirmClearGrants})
		} else {
			m.notice("no active session grants")
		}
	case "status":
		m.notice("model " + m.status.Model + " " + m.gly.Bullet + " branch " + m.sess.Branch +
			" " + m.gly.Bullet + " " + formatTokens(m.status.Tokens))
	case "diff":
		m.openModal(ModalState{Kind: ModalDiff, Title: "working tree", Lines: fakeDiff(), Added: 12, Removed: 4})
	case "files":
		m.notice("files touched this session: calc/divide.go, calc/divide_test.go")
	case "compact":
		m.notice("nothing to compact yet")
	case "read", "ls", "search", "gitstatus", "gitdiff":
		// Temporary M1 scaffolding, removed in M3 when the model drives tools.
		// Labelled (debug) in /help so nobody mistakes them for product surface.
		if m.RunTool == nil {
			m.notice("tools are not wired up in this build")
			break
		}
		name, input := debugToolCall(cmd, args)
		if name == "" {
			m.comp.Hint = "usage: /" + cmd + " " + debugUsage(cmd)
			return m, nil
		}
		m.RunTool(name, input)

	default:
		// Unknown commands get a dim inline hint, never an error card, and are
		// never sent to the model. ui-spec §6.
		m.comp.Hint = "unknown command /" + cmd
		return m, nil
	}
	m.scroll.repin() // re-pin: a slash command is a request, so its answer is shown
	m.relayout(lay)
	return m, nil
}

func (m *Model) notice(text string) {
	m.appendBlock(Item{Kind: KindNotice, Notice: &NoticeCard{Text: text}})
}

// debugToolCall maps an M1 debug command to a tool invocation.
func debugToolCall(cmd, args string) (string, map[string]any) {
	args = strings.TrimSpace(args)
	switch cmd {
	case "read":
		if args == "" {
			return "", nil
		}
		return "read_file", map[string]any{"path": args}
	case "ls":
		if args == "" {
			args = "."
		}
		return "list_files", map[string]any{"path": args}
	case "search":
		if args == "" {
			return "", nil
		}
		return "search_code", map[string]any{"query": args}
	case "gitstatus":
		return "git_status", map[string]any{}
	case "gitdiff":
		return "git_diff", map[string]any{}
	}
	return "", nil
}

func debugUsage(cmd string) string {
	switch cmd {
	case "read":
		return "<path>"
	case "search":
		return "<query>"
	case "ls":
		return "[path]"
	}
	return ""
}
