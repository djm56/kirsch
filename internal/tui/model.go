// Package tui renders Kirsch's terminal interface.
//
// Milestone 0 drives it entirely from fake data (see fake.go). The package
// imports no other internal package and never will: it receives typed messages
// and emits typed intents, so it cannot reach a tool, a provider or the
// filesystem. architecture.md §3 rule 2.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BaseMode is the resting focus: the composer, or the transcript.
type BaseMode uint8

const (
	BaseComposing BaseMode = iota
	BaseBrowsing
)

// Mode is the effective input mode. ui-spec §5.1.
type Mode uint8

const (
	ModeComposing Mode = iota
	ModeBrowsing
	ModeConfirm
	ModeApprovalPending
	ModeModal
)

// String names the mode, for test failure messages.
func (m Mode) String() string {
	switch m {
	case ModeBrowsing:
		return "Browsing"
	case ModeConfirm:
		return "Confirm"
	case ModeApprovalPending:
		return "ApprovalPending"
	case ModeModal:
		return "Modal"
	default:
		return "Composing"
	}
}

// Busy is an orthogonal flag, not a mode: it dims the composer and drives the
// status-bar spinner, but does not change which mode is active. ui-spec §5.1.
type Busy struct {
	Active bool
	Verb   string
}

// SessionInfo is the header's data. Injected, never read from git — otherwise
// every golden would break the moment tests ran on a feature branch.
type SessionInfo struct {
	Project   string
	Branch    string
	Dirty     bool
	Compacted bool
	Version   string
}

// Options configures a Model. Everything environment-dependent arrives here so
// that no fallback state requires touching process globals to reach.
type Options struct {
	Version string
	Caps    Caps
	Session SessionInfo
	Status  Status
	Now     func() time.Time
}

// Model is the root Bubble Tea model.
//
// There is no `mode` field. The mode is derived from four independent facts —
// see mode() — which is what makes both of ui-spec §5.1's easy-to-get-wrong
// subtleties disappear rather than needing to be handled.
type Model struct {
	width, height int

	caps Caps
	sty  *Styles
	gly  Glyphs
	rend *lipgloss.Renderer

	tr         Transcript
	sel        ItemID
	expanded   map[ItemID]bool
	scroll     Scroll
	lines      []string  // flattened transcript, styled
	plainLines []string  // the same lines with identity styles
	rows       []itemRow // per-item extents into lines

	// Mode inputs. Each is independent; precedence lives in mode().
	base            BaseMode
	pendingApproval ItemID
	modal           *ModalState
	confirm         *ConfirmState

	busy   Busy
	sess   SessionInfo
	status Status

	comp Composer

	// Callbacks into internal/app. Function fields rather than an interface
	// so the TUI depends on behaviour it names itself and cannot be handed a
	// package it is forbidden to import.
	RunTool func(name string, input map[string]any)
	Cancel  func()

	projectTypes []string
	toolCards    map[int64]ItemID // app-side tool id → transcript card

	frame        int  // spinner frame; advanced only by a tick message
	spinnerAlive bool // a tick chain is in flight; see tickSpinnerOnce
	now          func() time.Time
	lastCtrlC    time.Time

	fake fakeDriver
}

// itemRow records where an item landed in the flattened transcript, which is
// what makes "scroll only as far as needed" implementable.
type itemRow struct {
	ID    ItemID
	Start int
	N     int
}

// New builds a Model with the Milestone 0 fixture loaded.
func New(o Options) Model {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Session.Project == "" {
		o.Session = SessionInfo{Project: "my-project", Branch: "main", Dirty: true}
	}
	o.Session.Version = o.Version
	if o.Status.Model == "" {
		o.Status = Status{Model: "claude-sonnet-5", Family: "sonnet-5"}
	}

	rend := NewRenderer(o.Caps.Colour)
	m := Model{
		caps:      o.Caps,
		rend:      rend,
		sty:       NewStyles(rend, o.Caps.Colour),
		gly:       NewGlyphs(o.Caps.Unicode),
		expanded:  map[ItemID]bool{},
		toolCards: map[int64]ItemID{},
		scroll:    Scroll{Pinned: true},
		sess:      o.Session,
		status:    o.Status,
		comp:      newComposer(),
		now:       o.Now,
	}
	return m
}

// NewWithFixture is New with loadFixture's transcript preloaded — a reduced
// subset of ui-spec §8, not the whole of it; see loadFixture. Used by the
// golden tests and by `--demo`; the plain constructor starts empty so the
// first thing anyone sees is the onboarding screen (§7.5, screen 01).
func NewWithFixture(o Options) Model {
	m := New(o)
	m.loadFixture()
	m.relayout(m.layout())
	return m
}

// Init starts the single spinner tick chain.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return spinnerStartMsg{} }
}

// mode derives the effective input mode.
//
// The switch order *is* ui-spec §5.1's precedence: Modal > ApprovalPending >
// Confirm > Browsing/Composing. Deriving rather than storing is what makes an
// approval arriving during an open modal render without stealing capture — the
// derivation simply keeps returning ModeModal until the modal closes — and what
// makes Esc return to the previous mode without a returnTo field that could go
// stale. A push-down stack would get both wrong, because it grants capture by
// recency rather than by rank.
func (m *Model) mode() Mode {
	switch {
	case m.modal != nil:
		return ModeModal
	case m.pendingApproval != 0:
		return ModeApprovalPending
	case m.confirm != nil:
		return ModeConfirm
	case m.base == BaseBrowsing:
		return ModeBrowsing
	default:
		return ModeComposing
	}
}

// setBase moves focus between the composer and the transcript. Apart from the
// initial Focus() in newComposer, focus and blur belong here and nowhere else.
func (m *Model) setBase(b BaseMode) {
	m.base = b
	if b == BaseComposing {
		m.comp.ta.Focus()
		if m.sel == 0 {
			m.sel = m.tr.lastSelectable()
		}
	} else {
		m.comp.ta.Blur()
		if m.sel == 0 {
			m.sel = m.tr.lastSelectable()
		}
	}
}

func (m *Model) openModal(s ModalState) { m.modal = &s }
func (m *Model) closeModal()            { m.modal = nil }
func (m *Model) askConfirm(c ConfirmState) {
	m.confirm = &c
}

// overlayOpen reports whether a modal or a confirm prompt is on screen.
//
// It is relayout's flatten condition and dispatchKey's repaint trigger, named
// once so the two cannot drift: an overlay kind added to the flatten condition
// but not to the repaint trigger would dim the transcript and never undim it.
//
// The opposite seam is open, but not reachable. dispatchKey repaints only when
// this predicate *changes*, so a keypress that swapped one overlay for another —
// closing a modal and raising a confirm inside a single handler — would read true
// on both sides and skip the repaint. Nothing in Update does that today, and it
// would be harmless anyway while relayout gives modals and confirms the identical
// Flat() treatment: what the skipped repaint would have rebuilt is the cached
// flattened transcript, and that cache is the same either way. The two *frames*
// are not the same bytes — View draws a modal box or a confirm box over the band,
// and those differ — but the band underneath is already correct, which is the part
// a repaint would have changed. Give the two kinds different treatment in relayout
// and this stops being theoretical.
func (m *Model) overlayOpen() bool { return m.modal != nil || m.confirm != nil }

// appendBlock adds a block to the transcript, counting it against the "↓ n new"
// indicator when the viewport is scrolled away from the bottom.
//
// The count lives here rather than at each caller because ui-spec §2.4 defines
// it as "blocks arrived since unpinning" — one fact about arrival, not a rule
// each message handler has to remember. A handler that forgets it fails
// silently: the indicator simply never appears, which is how NewSince spent
// Milestone 0 as a field nothing ever incremented.
//
// TestAppendBlockIsTheOnlyAppendPath is the mechanical half of that rule: it
// parses this directory's non-test files and fails on any `.Append(` call
// outside this function. It matches on the method name alone, so it catches
// the spellings someone reaches for by reflex and not an append that avoids
// the word — see that test's docblock for what it knowingly does not cover.
// Treat it as a check on the obvious routes, not a proof that this one is the
// only one.
//
// Arrival is the trigger, not change. A tool card reaching a terminal state
// (MutateTool) and words streaming into an open assistant block (AppendText)
// are both blocks the user has already been counted for; counting them again
// would have "↓ n new" promise more unread blocks below the fold than exist.
func (m *Model) appendBlock(it Item) ItemID {
	id := m.tr.Append(it)
	if !m.scroll.Pinned {
		m.scroll.NewSince++
	}
	return id
}

// relayout rebuilds the flattened transcript and its per-item extents.
//
// This runs in Update, before key handling, not lazily inside View: minimal
// scrolling needs to know where the selected card sits and how tall the
// viewport is, and if that only existed at render time it could not be
// implemented at all.
func (m *Model) relayout(lay Layout) {
	if !lay.ShowTranscript {
		m.lines, m.plainLines, m.rows = nil, nil, nil
		return
	}
	// Content width is per item: the gutter's two columns are only reserved
	// for items that actually draw one. Reserving them everywhere shortens
	// every speaker rule and separator by two cells.
	fullW := lay.ContentW
	if fullW < 8 {
		fullW = 8
	}
	sty := m.sty
	if m.overlayOpen() {
		sty = m.sty.Flat() // the transcript recedes behind an overlay
	}

	// Rendered twice: once with the live palette, once with identity styles.
	// The plain pass is what the "↓ n new" overlay measures against and what
	// the golden .txt files record — and because both passes run the same code
	// with only the Styles swapped, strip(styled) == plain holds by
	// construction rather than by inspection. ui-spec §13.
	plainSty := NewStyles(nil, false)

	lines := make([]string, 0, 64)
	plain := make([]string, 0, 64)
	rows := make([]itemRow, 0, m.tr.Len())
	lines = append(lines, "")
	plain = append(plain, "")

	// Inter-item spacing: a blank line between items, except between adjacent
	// one-line cards of the same kind, which group visually. This is the rule
	// every grid follows — screen 02's two tool cards sit together, screen 08's
	// two notices sit together, but a collapsed card followed by a boxed one is
	// always separated (screens 03, 04).
	items := m.tr.Items()
	prevKind := ItemKind(0)
	prevLines := 0
	for i, it := range items {
		gutter := it.ID == m.sel || it.ID == m.pendingApproval
		w := fullW
		if gutter {
			w = fullW - gutterWidth(m.gly)
		}
		ctx := renderCtx{
			W:        w,
			Gutter:   gutter,
			Expanded: m.expanded[it.ID],
			Selected: it.ID == m.sel,
			G:        m.gly,
			Frame:    m.frame,
			ShowDur:  lay.ShowDuration,
		}
		ctx.Sty = sty
		body := renderItem(it, ctx)
		ctx.Sty = plainSty
		plainBody := renderItem(it, ctx)

		grouped := prevLines == 1 && len(body) == 1 && prevKind == it.Kind
		if i > 0 && !grouped {
			lines = append(lines, "")
			plain = append(plain, "")
		}
		rows = append(rows, itemRow{ID: it.ID, Start: len(lines), N: len(body)})
		lines = append(lines, body...)
		plain = append(plain, plainBody...)
		prevKind, prevLines = it.Kind, len(body)
	}
	lines = append(lines, "")
	plain = append(plain, "")
	m.lines, m.plainLines, m.rows = lines, plain, rows

	if m.scroll.Pinned {
		m.scroll.Offset = maxInt(0, len(lines)-lay.TranscriptH)
	}
}

// rowFor returns an item's extent in the flattened transcript.
func (m *Model) rowFor(id ItemID) (itemRow, bool) {
	for _, r := range m.rows {
		if r.ID == id {
			return r, true
		}
	}
	return itemRow{}, false
}

// revealSelection scrolls the minimum distance needed to bring the selected
// card into view — and does nothing at all when it is already visible, which is
// the literal implementation of "scrolling and selection must not be collapsed
// into one". ui-spec §2.4.
func (m *Model) revealSelection(viewH int) {
	r, ok := m.rowFor(m.sel)
	if !ok || viewH <= 0 {
		return
	}
	top, bot := r.Start, r.Start+r.N-1
	total := len(m.lines)
	switch {
	case r.N > viewH:
		m.scroll.setOffset(top, total, viewH) // over-tall card: align its summary line
	case top < m.scroll.Offset:
		m.scroll.setOffset(top, total, viewH)
	case bot > m.scroll.Offset+viewH-1:
		m.scroll.setOffset(bot-viewH+1, total, viewH)
	}
}

// transcriptRows renders the visible window, plus the unpinned indicator.
func (m Model) transcriptRows(lay Layout) []string {
	h := lay.TranscriptH
	if h <= 0 {
		return nil
	}
	if !lay.ShowTranscript {
		out := make([]string, h)
		if h > 0 {
			out[0] = m.sty.Dim("terminal too narrow")
		}
		return out
	}

	// Onboarding, while nothing has been said yet. §7.5, screen 01.
	if m.tr.Len() == 0 {
		return m.emptyStateRows(lay)
	}

	out := make([]string, 0, h)
	for i := 0; i < h; i++ {
		idx := m.scroll.Offset + i
		if idx >= 0 && idx < len(m.lines) {
			out = append(out, m.lines[idx])
		} else {
			out = append(out, "")
		}
	}

	// The indicator overlays the last row rather than occupying one of its
	// own: a row would change the line count between pinned and unpinned,
	// which the "identical structure" rule forbids.
	if !m.scroll.Pinned && m.scroll.NewSince > 0 && h > 0 {
		tail := m.gly.New + " " + itoa(m.scroll.NewSince) + " new"
		idx := m.scroll.Offset + h - 1
		plain := ""
		if idx >= 0 && idx < len(m.plainLines) {
			plain = m.plainLines[idx]
		}
		keep := lay.ContentW - cellWidth(tail) - 1
		out[h-1] = pad(truncEnd(plain, maxInt(0, keep), m.gly.Trunc), maxInt(0, keep)) +
			" " + m.sty.Accent(tail)
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
