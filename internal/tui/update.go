package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Binding documents one key. The help overlay is built from this table, so the
// two cannot drift apart.
type Binding struct{ Key, Desc string }

// BindingGroup is one mode's bindings.
type BindingGroup struct {
	Mode     string
	Bindings []Binding
}

// bindingGroups is the authoritative binding table. ui-spec §5.2.
var bindingGroups = []BindingGroup{
	{"composing", []Binding{
		{"Enter", "send"},
		{"Shift+Enter", "newline"},
		{"Ctrl+J", "newline (alt)"},
		{"Tab", "complete /cmd"},
		{"↑ at line 1", "browse"},
		{"Esc", "cancel turn"},
		{"q", "quit (idle)"},
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
		id := m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
			Name: msg.Name, Target: msg.Target, State: StateRunning,
		}})
		m.toolCards[msg.ID] = id
		m.busy = Busy{Active: true, Verb: msg.Name}
		m.relayout(m.layout())
		return m, m.tickSpinnerOnce()

	case ToolCompletedMsg:
		return m.applyToolResult(msg)

	case ErrorMsg:
		m.tr.Append(Item{Kind: KindError, Err: &ErrorCard{
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
		return m.handleKey(msg, lay)
	}
	return m, nil
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
		cardID = m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
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

func (m Model) layout() Layout {
	return computeLayout(m.width, m.height, m.comp.contentLines())
}

// handleKey dispatches to exactly one mode handler.
//
// There is no fallback arm and no shared pre-pass beyond the force-quit check,
// so a key cannot be handled twice. In particular the textarea is updated from
// exactly one place — inside keyComposing — which is what stops every other
// mode from leaking keystrokes into the composer. ui-spec §5.2.
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

	// `q` quits only when idle with an empty composer; otherwise it is a
	// literal character. ui-spec §5.2.
	if k.Type == tea.KeyRunes && string(k.Runes) == "q" && m.comp.Value() == "" {
		return m, tea.Quit
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
		m.setBase(BaseComposing)
		m.scroll.repin() // re-pin trigger 2 of 3
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
		m.scroll.repin() // re-pin trigger 3 of 3
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
	case "?":
		m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
		return m, nil
	case "q":
		return m, tea.Quit
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

func (m Model) keyModal(k tea.KeyMsg, lay Layout) (tea.Model, tea.Cmd) {
	page := maxInt(1, lay.TranscriptH-4)
	switch k.Type {
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
		m.scroll.repin()
		m.relayout(lay)
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
func (m Model) submit(lay Layout) (tea.Model, tea.Cmd) {
	v := strings.TrimRight(m.comp.Value(), "\n")
	if strings.TrimSpace(v) == "" {
		return m, nil
	}
	if cmd, args, ok := parseSlash(v); ok {
		m.comp.Reset()
		return m.runSlash(cmd, args, lay)
	}
	m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{Lines: SanitizeLines(v)}})
	m.comp.Reset()
	m.scroll.repin() // re-pin trigger 1 of 3: sending
	m.busy = Busy{Active: true, Verb: "thinking"}
	m.fake.begin(&m)
	m.relayout(lay)
	return m, tickStream()
}

func (m Model) runSlash(cmd, args string, lay Layout) (tea.Model, tea.Cmd) {
	switch cmd {
	case "help":
		m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
	case "quit":
		return m, tea.Quit
	case "new":
		if m.busy.Active {
			m.askConfirm(ConfirmState{Prompt: "Discard the running turn and start a new session?",
				Action: ConfirmNewSession})
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
		return m, nil

	default:
		// Unknown commands get a dim inline hint, never an error card, and are
		// never sent to the model. ui-spec §6.
		m.comp.Hint = "unknown command /" + cmd
		return m, nil
	}
	m.relayout(lay)
	return m, nil
}

func (m *Model) notice(text string) {
	m.tr.Append(Item{Kind: KindNotice, Notice: &NoticeCard{Text: text}})
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
