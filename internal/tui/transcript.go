package tui

import (
	"regexp"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// ItemID identifies a transcript item. Zero is the none sentinel.
//
// Selection is held as an ID rather than an index because §3.9 requires it to
// survive new content arriving, and because from M4 the transcript is rebuilt
// by replaying a log — where indices mean nothing.
type ItemID uint64

// ItemKind tags the union below.
type ItemKind uint8

const (
	KindUser ItemKind = iota + 1
	KindAssistant
	KindTool
	KindApproval
	KindError
	KindNotice
	KindThinking
)

// Item is a closed tagged union: exactly one payload pointer is non-nil,
// selected by Kind.
//
// A tag rather than an interface, for three reasons. Fixtures read like the
// reference grids instead of like constructor calls; rendering has exactly one
// switch, so the rule that line counts must not change with colour has one
// place to be audited rather than one per kind; and the "never rewritten once
// terminal" guard has one place to live rather than one per payload type —
// see MutateTool, which is also where its limits are written down.
type Item struct {
	ID   ItemID
	Kind ItemKind

	Text     *TextBlock
	Tool     *ToolCard
	Approval *ApprovalCard
	Err      *ErrorCard
	Notice   *NoticeCard
	Thinking *ThinkingCard
}

// Selectable reports whether ↑/↓ stop on this item. Only cards are selectable;
// text blocks are skipped, because Enter does nothing on them and stopping
// there is pure friction. ui-spec §3.9.
func (i Item) Selectable() bool {
	switch i.Kind {
	case KindTool, KindApproval, KindError, KindNotice, KindThinking:
		return true
	default:
		return false
	}
}

// CardState is a card's lifecycle position. ui-spec §3.8.
type CardState uint8

const (
	StatePending   CardState = iota // ◌ awaiting approval or queued
	StateRunning                    // ◐ executing
	StateOK                         // ✓
	StateError                      // ✗
	StateCancelled                  // ⊘ turn cancelled mid-tool
)

// Terminal reports whether the card has reached a final state. A terminal card
// is frozen: it is never rewritten in place. ui-spec §3.8.
func (s CardState) Terminal() bool { return s >= StateOK }

// TextBlock is a user or assistant message. Lines are stored unwrapped —
// wrapping is a function of the current width, and the width changes on resize.
type TextBlock struct {
	Lines     []string
	Streaming bool // assistant only: render a trailing caret
	Cancelled bool // partial text from a cancelled turn stays, marked
}

// ToolCard is one tool invocation. Out holds the full output: the 200-line cap
// is applied at render, never to the stored data, because the transcript is
// "what the user saw" and the user can always open the modal.
type ToolCard struct {
	Name    string // rendered bold, as is an approval card's tool name
	Target  string
	Summary string
	State   CardState
	Elapsed time.Duration // zero when not applicable (a running card has none)
	Trunc   string        // upstream cap note, e.g. "truncated — 200KB cap"
	Out     []string
	Note    string // e.g. "session grant: go test"
}

// ApprovalKind distinguishes the two approval variants, which differ in whether
// they may offer a session grant.
type ApprovalKind uint8

const (
	ApprovalPatch ApprovalKind = iota + 1
	ApprovalCommand
)

// ApprovalOutcome is how the user resolved an approval.
type ApprovalOutcome uint8

const (
	Unresolved ApprovalOutcome = iota
	Approved
	ApprovedSession
	Rejected
)

// ApprovalCard is a pending or resolved approval. ui-spec §3.4.
type ApprovalCard struct {
	Kind       ApprovalKind
	Title      string
	Subject    string // the collapsed form's summary, e.g. "2 files changed"
	Detail     []string
	GrantScope string // argv prefix; must be empty for ApprovalPatch
	Outcome    ApprovalOutcome
	Elapsed    time.Duration
	Diff       []string
	Added      int
	Removed    int
}

// OffersSessionGrant reports whether [a] appears on this card.
//
// Derived, never stored. ADR 0006 is emphatic that `a` is offered for
// run_command only and never for apply_patch; a stored bool invites a fixture
// that sets it on a patch card, which is precisely the bug. A patch card has no
// scope to name, so the wrong state is unrepresentable.
func (a ApprovalCard) OffersSessionGrant() bool {
	return a.Kind == ApprovalCommand && a.GrantScope != ""
}

// ErrorCard is an infrastructure failure — not a tool error the model could
// handle, which stays a tool card with an ✗ badge. architecture.md §8.
type ErrorCard struct {
	Kind    string
	Message string
	Hint    string
	Detail  []string
}

// NoticeCard is a single dim line, no border. ui-spec §3.7.
type NoticeCard struct{ Text string }

// ThinkingCard is collapsed and dimmed throughout. Default off in v0.1.
type ThinkingCard struct {
	Tokens int
	Body   []string
}

// Transcript is the ordered list of items, plus an ID allocator and a revision
// counter the layout cache uses for invalidation.
type Transcript struct {
	items  []Item
	nextID ItemID
	rev    uint64
}

// Append adds an item and returns its assigned ID.
//
// Model code reaches this through Model.appendBlock, never directly: the
// transcript does not know where the viewport is, so the "↓ n new" count has to
// be taken one level up.
//
// TestAppendBlockIsTheOnlyAppendPath checks that mechanically by rejecting any
// `.Append(` call in a non-test file in this package outside appendBlock. It
// keys on the method name only, so it does not catch an append spelled some
// other way or one that writes t.items directly; that test's docblock lists
// what falls outside it.
func (t *Transcript) Append(it Item) ItemID {
	t.nextID++
	it.ID = t.nextID
	t.items = append(t.items, it)
	t.rev++
	return it.ID
}

// Items returns the underlying slice. Callers must not mutate it; use the
// Mutate* methods, which enforce the terminal-state guard.
func (t *Transcript) Items() []Item { return t.items }

// Len returns the item count.
func (t *Transcript) Len() int { return len(t.items) }

// Find returns the item with the given ID.
func (t *Transcript) Find(id ItemID) (Item, int, bool) {
	for i, it := range t.items {
		if it.ID == id {
			return it, i, true
		}
	}
	return Item{}, -1, false
}

// MutateTool applies fn to a tool card, refusing if the card has already
// reached a terminal state, so a late-arriving message cannot rewrite a card
// the user has already read. ui-spec §3.8.
//
// Not a seam the type system closes. Item.Tool and Item.Text are pointers, so
// any holder of an Item — from Find, or from ranging Items() — shares the
// payload and can write it directly, past whatever the method would have
// checked. cancelTurn sets it.Tool.State without coming through here, and is
// correct only because it tests for StateRunning itself. Route a new tool
// mutation through this method: writing the field is a decision to re-derive
// the guard by hand, and it skips t.rev — which costs nothing today only
// because nothing reads t.rev.
//
// Text has the same shape and one less method. cancelTurn and advanceFake both
// clear it.Text.Streaming in the open because AppendText only appends; there
// is no guarded way to end a stream, so that flag has no seam at all.
func (t *Transcript) MutateTool(id ItemID, fn func(*ToolCard)) bool {
	it, _, ok := t.Find(id)
	if !ok || it.Kind != KindTool || it.Tool.State.Terminal() {
		return false
	}
	fn(it.Tool)
	t.rev++
	return true
}

// AppendText appends to a streaming text block. Used by the fake stream driver.
func (t *Transcript) AppendText(id ItemID, s string) bool {
	it, _, ok := t.Find(id)
	if !ok || it.Text == nil || !it.Text.Streaming {
		return false
	}
	lines := it.Text.Lines
	if len(lines) == 0 {
		it.Text.Lines = []string{s}
	} else {
		lines[len(lines)-1] += s
	}
	t.rev++
	return true
}

// nextSelectable walks to the next selectable item in the given direction,
// reporting false at the ends so the caller can transfer focus.
func (t *Transcript) nextSelectable(from ItemID, dir int) (ItemID, bool) {
	start := -1
	for i, it := range t.items {
		if it.ID == from {
			start = i
			break
		}
	}
	if start < 0 {
		// No current selection: enter from the appropriate end.
		if dir < 0 {
			for i := len(t.items) - 1; i >= 0; i-- {
				if t.items[i].Selectable() {
					return t.items[i].ID, true
				}
			}
		}
		for _, it := range t.items {
			if it.Selectable() {
				return it.ID, true
			}
		}
		return 0, false
	}
	for i := start + dir; i >= 0 && i < len(t.items); i += dir {
		if t.items[i].Selectable() {
			return t.items[i].ID, true
		}
	}
	return 0, false
}

// lastSelectable returns the final selectable item, or zero.
func (t *Transcript) lastSelectable() ItemID {
	for i := len(t.items) - 1; i >= 0; i-- {
		if t.items[i].Selectable() {
			return t.items[i].ID
		}
	}
	return 0
}

// ── Scroll ──────────────────────────────────────────────────────────────────

// Scroll is the transcript's viewport position.
//
// Pinned is authoritative; while pinned, Offset is derived on every relayout
// and never trusted. That makes "new content while pinned keeps it pinned" and
// "resize while pinned stays at the bottom" free rather than code.
type Scroll struct {
	Offset   int
	Pinned   bool
	NewSince int // blocks arrived since unpinning — the "↓ n new" count
}

// setOffset moves the viewport and maintains every pin invariant.
//
// The "scrolling back to the bottom" re-pin lives here, so it cannot be
// forgotten at a call site.
func (s *Scroll) setOffset(off, total, viewH int) {
	max := total - viewH
	if max < 0 {
		max = 0
	}
	if off < 0 {
		off = 0
	}
	if off > max {
		off = max
	}
	s.Offset = off
	if off >= max {
		s.Pinned, s.NewSince = true, 0
	} else {
		s.Pinned = false
	}
}

// repin returns the transcript to the bottom.
//
// What matters is where it is *not* called: nowhere in the typing path,
// nowhere in the append path, nowhere in View. That absence is the entire
// content of ui-spec §2.4's "typing does not re-pin". Every call site states
// its own reason in a trailing comment, so grep for it rather than keeping a
// list here that goes stale on the next one.
func (s *Scroll) repin() {
	s.Pinned, s.NewSince = true, 0
}

// ── Sanitisation ────────────────────────────────────────────────────────────

// ansiRe matches CSI, OSC and two-byte escape sequences, in both their 7-bit
// (ESC-prefixed) and 8-bit (C1) forms.
//
// The 8-bit forms are written as runes (\x{009b}) rather than raw bytes: a bare
// 0x9b is not valid UTF-8, and Go's regexp rejects the pattern outright. Stray
// C1 bytes that are not part of a sequence are caught by replaceControls.
var ansiRe = regexp.MustCompile(
	"\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)" + // OSC ... BEL or ST
		"|\x1b\\[[0-?]*[ -/]*[@-~]" + // CSI
		"|\\x{009b}[0-?]*[ -/]*[@-~]" + // 8-bit CSI
		"|\x1b[@-Z\\\\-_]" + // two-byte escapes
		"|\x1b\\([A-Za-z0-9]", // charset selection
)

// Sanitize makes captured output safe to render. ui-spec §7.1, applied in this
// order.
//
// This runs once, when an item is created — never at render time. View() runs
// on every keystroke and every stream tick, so re-stripping a 4,000-line output
// each time would be pure waste; more importantly, output stored clean means
// the renderer *cannot* leak an escape, which makes the property provable
// rather than merely tested.
func Sanitize(s string) string {
	s = ansiRe.ReplaceAllString(s, "")

	// Normalise CRLF to LF before collapsing \r runs. Order matters: collapsing
	// first would reduce a whole CRLF file to its last line.
	s = strings.ReplaceAll(s, "\r\n", "\n")

	out := make([]string, 0, 8)
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "\r") {
			line = lastNonEmptySegment(line)
		}
		line = expandTabs(line)
		line = replaceControls(line)
		line = capWidth(line, MaxRenderedLineWidth)
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// SanitizeLines is Sanitize, split ready for storage.
func SanitizeLines(s string) []string {
	return strings.Split(Sanitize(s), "\n")
}

// lastNonEmptySegment keeps only the final segment of a \r-separated run, which
// is how a progress bar renders as its finished state. §7.1.
func lastNonEmptySegment(line string) string {
	parts := strings.Split(line, "\r")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}

// expandTabs expands to a 4-column tab stop, consistently everywhere. §7.1.
func expandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := 4 - col%4
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col += runewidth.RuneWidth(r)
	}
	return b.String()
}

// replaceControls swaps remaining control characters for a visible bullet.
//
// Covers C0 (0x00–0x1f), DEL, the C1 range (0x80–0x9f) left over after ansiRe,
// and the replacement rune that invalid UTF-8 decodes to — none of which have a
// sensible width and any of which can move a terminal's cursor. §7.1.
func replaceControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == 0xfffd {
			return '·'
		}
		return r
	}, s)
}

// capWidth truncates to n display columns, marker included, so the cap is a
// hard guarantee. A minified bundle on one line must not hang the wrapper.
func capWidth(s string, n int) string {
	if cellWidth(s) <= n {
		return s
	}
	return runewidth.Truncate(s, n, "⋯")
}
