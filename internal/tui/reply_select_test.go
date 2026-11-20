package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mixedTranscript is a model whose transcript holds, in order: a user message,
// a reply, a tool card, an approval card folded into that tool card, a reply
// with no lines, a second reply and a second user message. Of those, only the
// two replies and the tool card are selectable.
type mixedTranscript struct {
	m                         Model
	user1, replyA, tool, appr ItemID
	empty, replyB, user2      ItemID
	rec                       *copyRecorder
}

// newMixedTranscript builds the transcript above on an 80x24 model with a
// recording Copy wired, in Composing with nothing selected. Items go in through
// appendBlock; the WindowSizeMsg afterwards makes Update lay them out.
func newMixedTranscript(t *testing.T) mixedTranscript {
	t.Helper()
	x := mixedTranscript{rec: &copyRecorder{}}
	m := newTestModel()
	m.Copy = x.rec.copy
	x.user1 = m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"question one"}}})
	x.replyA = m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"reply A"}, Source: "reply A"}})
	x.tool = m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "read_file", Target: "a.go", State: StateOK, Out: []string{"package a"},
	}})
	x.appr = m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{
		Kind: ApprovalCommand, Outcome: Approved, ToolItem: x.tool,
	}})
	x.empty = m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{}})
	x.replyB = m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"reply B"}, Source: "reply B"}})
	x.user2 = m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"question two"}}})
	x.m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !x.folded() {
		t.Fatal("setup: the approval card is not folded, so it would not exercise the skip")
	}
	if x.m.mode() != ModeComposing || x.m.sel != 0 {
		t.Fatalf("setup: mode = %v, sel = %d, want Composing with nothing selected", x.m.mode(), x.m.sel)
	}
	return x
}

// folded reports whether the transcript's approval card is folded.
func (x mixedTranscript) folded() bool {
	it, _, ok := x.m.tr.Find(x.appr)
	return ok && it.Approval.Folded()
}

// browse enters Browsing with Shift+Up, the real entry point from Composing.
func browse(m Model) Model { return drive(m, keyType(tea.KeyShiftUp)) }

// pressC sends `c` through Update and returns the model and the Cmd.
func pressC(m Model) (Model, tea.Cmd) {
	next, cmd := m.Update(key('c'))
	return next.(Model), cmd
}

// TestItemSelectableByKind: a reply with at least one line is selectable, a
// reply with no lines is not, a user message is not, and a card is. The two
// replies differ only in their lines, so the zero-line row is judged beside one
// that is selected.
func TestItemSelectableByKind(t *testing.T) {
	cases := []struct {
		name string
		it   Item
		want bool
	}{
		{"reply with one line", Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"x"}}}, true},
		{"reply with no lines", Item{Kind: KindAssistant, Text: &TextBlock{}}, false},
		{"reply with a nil text block", Item{Kind: KindAssistant}, false},
		{"user message with a line", Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"x"}}}, false},
		{"tool card", Item{Kind: KindTool, Tool: &ToolCard{}}, true},
		{"notice card", Item{Kind: KindNotice, Notice: &NoticeCard{}}, true},
	}
	for _, c := range cases {
		if got := c.it.Selectable(); got != c.want {
			t.Errorf("%s: Selectable() = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestBrowsingArrowsStopOnRepliesAndSkipTheRest drives the Up and Down keys
// through Update on a mixed transcript. Up from the last reply must stop on the
// tool card and then the first reply, passing over the zero-line reply and the
// folded approval card; user messages are never stopped on. Down walks back and
// leaves Browsing past the last selectable item. The stops are compared as a
// whole list, so a reply skipped or a user message stopped on both fail it.
func TestBrowsingArrowsStopOnRepliesAndSkipTheRest(t *testing.T) {
	x := newMixedTranscript(t)
	m := browse(x.m)
	if m.mode() != ModeBrowsing || m.sel != x.replyB {
		t.Fatalf("entering Browsing with nothing selected: mode %v, sel %d, want Browsing on the last reply %d", m.mode(), m.sel, x.replyB)
	}

	var up []ItemID
	for i := 0; i < 5; i++ { // more presses than stops: the extra ones must change nothing
		m = drive(m, keyType(tea.KeyUp))
		up = append(up, m.sel)
	}
	wantUp := []ItemID{x.tool, x.replyA, x.replyA, x.replyA, x.replyA}
	if !reflect.DeepEqual(up, wantUp) {
		t.Errorf("selection after each Up = %v, want %v (tool %d, reply A %d)", up, wantUp, x.tool, x.replyA)
	}

	var down []ItemID
	for i := 0; i < 2; i++ {
		m = drive(m, keyType(tea.KeyDown))
		down = append(down, m.sel)
	}
	if want := []ItemID{x.tool, x.replyB}; !reflect.DeepEqual(down, want) {
		t.Errorf("selection after each Down = %v, want %v", down, want)
	}
	if m.mode() != ModeBrowsing {
		t.Fatalf("mode on the last selectable item = %v, want Browsing", m.mode())
	}
	m = drive(m, keyType(tea.KeyDown))
	if m.mode() != ModeComposing {
		t.Errorf("Down on the last selectable item left mode %v, want Composing", m.mode())
	}
}

// TestEnteringBrowsingKeepsTheSelection: Shift+Up from Composing keeps a
// selection that already exists, here the first reply. The control is the same
// transcript with nothing selected, which lands on the last selectable item.
func TestEnteringBrowsingKeepsTheSelection(t *testing.T) {
	x := newMixedTranscript(t)
	if got := browse(x.m).sel; got != x.replyB {
		t.Fatalf("control: nothing selected, entering Browsing selected %d, want the last reply %d", got, x.replyB)
	}
	x.m.sel = x.replyA
	if got := browse(x.m).sel; got != x.replyA {
		t.Errorf("entering Browsing with reply A selected moved the selection to %d, want %d", got, x.replyA)
	}
}

// selectedRows returns the plain transcript rows of one item.
func selectedRows(t *testing.T, m Model, id ItemID) []string {
	t.Helper()
	r, ok := m.rowFor(id)
	if !ok {
		t.Fatalf("item %d has no rows", id)
	}
	return m.plainLines[r.Start : r.Start+r.N]
}

// TestSelectedReplyDrawsTheGutterAndWrapsNarrower: a selected reply carries the
// gutter on every row and wraps 2 columns narrower, in Composing as well as in
// Browsing, with the Unicode glyph and with the ASCII fallback. Entry point:
// the model's own relayout after `sel` is set (Composing) and Shift+Up
// (Browsing). The control is the same reply unselected: no gutter, and a row
// wider than the narrowed width, which shows the narrowing is observable.
func TestSelectedReplyDrawsTheGutterAndWrapsNarrower(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("ab ", 100))
	for _, caps := range []Caps{{Unicode: true}, {Unicode: false}} {
		name := "Unicode"
		if !caps.Unicode {
			name = "ASCII"
		}
		t.Run(name, func(t *testing.T) {
			m := New(Options{Caps: caps})
			id := m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{long}, Source: long}})
			m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})
			full := m.layout().ContentW
			gutter := m.gly.Gutter
			narrow := full - gutterWidth(m.gly)

			// Control: unselected.
			plain := selectedRows(t, m, id)
			widest := 0
			for _, row := range plain {
				if strings.HasPrefix(row, gutter) {
					t.Errorf("control: unselected row %q starts with the gutter", row)
				}
				widest = maxInt(widest, cellWidth(row))
			}
			if widest <= narrow {
				t.Fatalf("control: widest unselected row is %d cells, not wider than %d, so narrowing could not be seen", widest, narrow)
			}

			for _, mode := range []string{"Composing", "Browsing"} {
				mm := m
				if mode == "Composing" {
					mm.sel = id
					mm.relayout(mm.layout())
				} else {
					mm = browse(mm)
				}
				if mm.sel != id {
					t.Fatalf("%s: setup: selection = %d, want %d", mode, mm.sel, id)
				}
				rows := selectedRows(t, mm, id)
				for _, row := range rows {
					if !strings.HasPrefix(row, gutter+" ") {
						t.Errorf("%s: row %q does not start with the gutter %q", mode, row, gutter)
					}
					if w := cellWidth(row); w > full {
						t.Errorf("%s: row %q is %d cells, over the content width %d", mode, row, w, full)
					}
					if w := cellWidth(strings.TrimPrefix(row, gutter+" ")); w > narrow {
						t.Errorf("%s: text %q is %d cells, over the narrowed width %d", mode, row, w, narrow)
					}
				}
				if len(rows) < len(plain) {
					t.Errorf("%s: selected reply has %d rows, fewer than the %d unselected", mode, len(rows), len(plain))
				}
			}
		})
	}
}

// TestEnterAndDOnAReplyDoNothing: Enter and d on a selected reply change no
// state and open no modal. Entry point: the Enter key and the `d` rune through
// Update in Browsing. State is the expanded and collapsed maps, the mode, the
// modal and the rendered lines. Controls, through the same keys and the same
// path: Enter on a selected thinking card still toggles expanded, and d on a
// selected tool card still opens a modal.
func TestEnterAndDOnAReplyDoNothing(t *testing.T) {
	x := newMixedTranscript(t)
	think := x.m.appendBlock(Item{Kind: KindThinking, Thinking: &ThinkingCard{Body: []string{"a thought"}, Tokens: 2}})
	m := drive(x.m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Controls.
	c := m
	c.sel = think
	c.base = BaseBrowsing
	c = drive(c, keyType(tea.KeyEnter))
	if !c.expanded[think] {
		t.Error("control: Enter on a thinking card did not toggle expanded")
	}
	d := m
	d.sel = x.tool
	d.base = BaseBrowsing
	d = drive(d, key('d'))
	if d.modal == nil || d.mode() != ModeModal {
		t.Error("control: d on a tool card did not open a modal")
	}

	for _, id := range []ItemID{x.replyA, x.replyB} {
		r := m
		r.sel = id
		r.base = BaseBrowsing
		r.relayout(r.layout())
		before := struct {
			expanded, collapsed map[ItemID]bool
			lines               []string
		}{copyMap(r.expanded), copyMap(r.collapsed), append([]string(nil), r.lines...)}
		for _, k := range []tea.KeyMsg{keyType(tea.KeyEnter), key('d')} {
			got := drive(r, k)
			if !reflect.DeepEqual(copyMap(got.expanded), before.expanded) || !reflect.DeepEqual(copyMap(got.collapsed), before.collapsed) {
				t.Errorf("reply %d, key %v: expanded/collapsed changed to %v / %v", id, k, got.expanded, got.collapsed)
			}
			if got.modal != nil || got.mode() != ModeBrowsing || got.sel != id {
				t.Errorf("reply %d, key %v: modal %v, mode %v, sel %d; want no modal, Browsing, same selection", id, k, got.modal, got.mode(), got.sel)
			}
			if !reflect.DeepEqual(got.lines, before.lines) {
				t.Errorf("reply %d, key %v: the rendered transcript changed", id, k)
			}
		}
	}
}

// TestEnterOnAReplyDoesNotScroll: Enter on a selected reply that is out of
// view leaves the scroll offset, the pin and the new-content count alone.
// Entry point: Shift+Up into Browsing, which selects the reply at the bottom,
// then PgUp to scroll it out of view, then the Enter key through Update. The
// control runs the same keys on the same model with a thinking card selected
// above the view instead, and Enter scrolls the card into view.
func TestEnterOnAReplyDoesNotScroll(t *testing.T) {
	m := newTestModel()
	filler := func() {
		m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"filler"}}})
	}
	for i := 0; i < 5; i++ {
		filler()
	}
	think := m.appendBlock(Item{Kind: KindThinking, Thinking: &ThinkingCard{Body: []string{"a thought"}, Tokens: 2}})
	for i := 0; i < 30; i++ { // enough rows that the card and the reply never share a page
		filler()
	}
	reply := m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"the reply"}, Source: "the reply"}})
	m = browse(drive(m, tea.WindowSizeMsg{Width: 80, Height: 24}))
	if m.mode() != ModeBrowsing || m.sel != reply {
		t.Fatalf("setup: mode %v, sel %d, want Browsing on the reply %d", m.mode(), m.sel, reply)
	}
	m = drive(m, keyType(tea.KeyPgUp))
	h := m.layout().TranscriptH
	visible := func(mm Model, id ItemID) bool {
		r, ok := mm.rowFor(id)
		return ok && r.Start >= mm.scroll.Offset && r.Start+r.N-1 <= mm.scroll.Offset+h-1
	}
	if m.scroll.Pinned || visible(m, reply) || visible(m, think) {
		t.Fatalf("setup: Pinned %v, reply visible %v, card visible %v; want unpinned with both out of view",
			m.scroll.Pinned, visible(m, reply), visible(m, think))
	}

	// Control: Enter on a thinking card out of view brings it into view.
	c := m
	c.sel = think
	c = drive(c, keyType(tea.KeyEnter))
	if !visible(c, think) || c.scroll.Offset == m.scroll.Offset {
		t.Fatalf("control: Enter on the thinking card left it out of view (Offset %d, was %d)", c.scroll.Offset, m.scroll.Offset)
	}

	got := drive(m, keyType(tea.KeyEnter))
	if got.scroll.Offset != m.scroll.Offset || got.scroll.Pinned != m.scroll.Pinned || got.scroll.NewSince != m.scroll.NewSince {
		t.Errorf("Enter on the reply moved the viewport: Offset %d (was %d), Pinned %v (was %v), NewSince %d (was %d)",
			got.scroll.Offset, m.scroll.Offset, got.scroll.Pinned, m.scroll.Pinned, got.scroll.NewSince, m.scroll.NewSince)
	}
	if got.sel != reply || got.mode() != ModeBrowsing {
		t.Errorf("Enter on the reply: sel %d, mode %v; want the reply, Browsing", got.sel, got.mode())
	}
}

// copyMap returns a copy of m that is never nil, so nil and empty compare equal.
func copyMap(m map[ItemID]bool) map[ItemID]bool {
	out := map[ItemID]bool{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// TestCCopiesTheReplysSourceText: `c` on a selected reply hands the cleaned
// source text to Copy, not its Lines. Entry point: the `c` rune through Update
// in Browsing, then the Cmd run and its result fed back. The reply's source has
// a tab and an ANSI sequence and its Lines say something else, so a payload
// built from Lines fails, a payload that keeps the escape fails, and one that
// loses the tab fails. The control is the second reply on the same path, which
// copies its own source.
func TestCCopiesTheReplysSourceText(t *testing.T) {
	x := newMixedTranscript(t)
	const src = "col1\tcol2 \x1b[31mred\x1b[0m end"
	const want = "col1\tcol2 red end"
	x.m.tr.items[1].Text.Source = src
	x.m.tr.items[1].Text.Lines = []string{"LINES-NOT-SOURCE"}
	m := browse(x.m)

	for _, c := range []struct {
		id   ItemID
		want string
	}{{x.replyA, want}, {x.replyB, "reply B"}} {
		x.rec.got = nil
		m.sel = c.id
		next, cmd := pressC(m)
		if len(x.rec.got) != 0 {
			t.Fatalf("Copy ran on the update loop; it must run in the Cmd")
		}
		next = runCopyCmd(t, next, cmd)
		if len(x.rec.got) != 1 || x.rec.got[0] != c.want {
			t.Errorf("c on item %d handed %q to Copy, want [%q]", c.id, x.rec.got, c.want)
		}
		if got, wantHint := next.comp.Hint, copiedNotice(c.want); got != wantHint {
			t.Errorf("c on item %d: hint = %q, want %q", c.id, got, wantHint)
		}
	}
}

// TestCOnACardGivesTheHint: `c` on a selected card shows "c copies replies
// only", starts no copy and calls no Copy. The control is `c` on a reply in the
// same transcript, which does start one.
func TestCOnACardGivesTheHint(t *testing.T) {
	x := newMixedTranscript(t)
	m := browse(x.m)

	m.sel = x.replyB
	if _, cmd := pressC(m); cmd == nil {
		t.Fatal("control: c on a reply returned no Cmd")
	}

	m.sel = x.tool
	seq := m.copySeq
	got, cmd := pressC(m)
	if cmd != nil {
		t.Error("c on a card returned a Cmd; it must start no copy")
	}
	if got.comp.Hint != "c copies replies only" || !got.comp.hintIsCopy {
		t.Errorf("hint = %q (copy notice %v), want %q as a copy notice", got.comp.Hint, got.comp.hintIsCopy, "c copies replies only")
	}
	if got.copySeq != seq+1 {
		t.Errorf("copySeq = %d, want %d: a notice-only c takes a sequence number", got.copySeq, seq+1)
	}
	if len(x.rec.got) != 0 {
		t.Errorf("Copy was called with %q on a card", x.rec.got)
	}
}

// TestCWithNothingSelectedGivesNothingToCopy: in Browsing with nothing selected
// `c` shows "nothing to copy" and starts no copy. The transcript holds only a
// user message, so Browsing has nothing to land on. The control is the same
// model after a reply arrives, where `c` copies it.
func TestCWithNothingSelectedGivesNothingToCopy(t *testing.T) {
	rec := &copyRecorder{}
	m := newTestModel()
	m.Copy = rec.copy
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"hello"}}})
	m = browse(drive(m, tea.WindowSizeMsg{Width: 80, Height: 24}))
	if m.mode() != ModeBrowsing || m.sel != 0 {
		t.Fatalf("setup: mode %v, sel %d, want Browsing with nothing selected", m.mode(), m.sel)
	}
	got, cmd := pressC(m)
	if cmd != nil || got.comp.Hint != "nothing to copy" || !got.comp.hintIsCopy {
		t.Errorf("c with nothing selected: Cmd %v, hint %q (copy notice %v), want no Cmd and %q", cmd != nil, got.comp.Hint, got.comp.hintIsCopy, "nothing to copy")
	}

	m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"hi"}, Source: "hi"}})
	m.sel = m.tr.lastSelectable()
	if _, cmd := pressC(m); cmd == nil {
		t.Error("control: c on a reply returned no Cmd")
	}
}

// TestCDoesNotMoveTheViewport: `c` leaves the scroll offset, the pin and the
// new-content count alone, whether it copies a reply or shows a notice for a
// card. Entry point: PgDn in Browsing to leave the view mid-transcript, then
// `c`. The control is `G` in the same setup, which does move the view.
func TestCDoesNotMoveTheViewport(t *testing.T) {
	x := newMixedTranscript(t)
	for i := 0; i < 20; i++ { // enough rows that the transcript overflows the viewport
		x.m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"filler"}}})
	}
	m := browse(drive(x.m, tea.WindowSizeMsg{Width: 80, Height: 24}))
	m = drive(m, keyType(tea.KeyPgDown)) // entering Browsing revealed the last reply, near the top; a page down leaves the view mid-transcript
	if m.scroll.Pinned || m.scroll.Offset == 0 {
		t.Fatalf("setup: Pinned = %v, Offset = %d, want unpinned and below the top", m.scroll.Pinned, m.scroll.Offset)
	}
	if g := drive(m, key('G')); !g.scroll.Pinned {
		t.Fatal("control: G did not re-pin in this setup")
	}
	for _, id := range []ItemID{x.replyA, x.tool} {
		mm := m
		mm.sel = id
		got, cmd := pressC(mm)
		if cmd != nil {
			got = runCopyCmd(t, got, cmd)
		}
		if got.scroll.Pinned || got.scroll.Offset != m.scroll.Offset || got.scroll.NewSince != m.scroll.NewSince {
			t.Errorf("c on item %d moved the viewport: Pinned %v, Offset %d (was %d), NewSince %d (was %d)",
				id, got.scroll.Pinned, got.scroll.Offset, m.scroll.Offset, got.scroll.NewSince, m.scroll.NewSince)
		}
	}
}

// TestNoticeOnlyCSupersedesACopyInFlight: a `c` that is only a notice takes a
// sequence number, so the result of an earlier copy still in flight cannot
// replace it. Entry point: `c` on a reply, then `c` on a card, then the first
// copy's result into Update. The control delivers the same first result with no
// second `c`, and it lands.
func TestNoticeOnlyCSupersedesACopyInFlight(t *testing.T) {
	x := newMixedTranscript(t)
	m := browse(x.m)
	m.sel = x.replyB
	m, cmd1 := pressC(m)
	msg1 := cmd1()

	if got := drive(m, msg1); got.comp.Hint != copiedNotice("reply B") {
		t.Fatalf("control: hint = %q, want %q", got.comp.Hint, copiedNotice("reply B"))
	}

	m.sel = x.tool
	m, cmd2 := pressC(m)
	if cmd2 != nil || m.comp.Hint != "c copies replies only" {
		t.Fatalf("setup: Cmd %v, hint %q, want no Cmd and the card hint", cmd2 != nil, m.comp.Hint)
	}
	m = drive(m, msg1)
	if m.comp.Hint != "c copies replies only" {
		t.Errorf("the older copy's result replaced the newer notice with %q", m.comp.Hint)
	}
}

// TestCOnAStreamingReplyCopiesTheTextSoFar: `c` on a reply that is still
// streaming copies what has arrived. Entry point: a text delta through Update,
// Shift+Up into Browsing, then `c`. The control is the finished reply, whose
// Source holds the same text, copied the same way.
func TestCOnAStreamingReplyCopiesTheTextSoFar(t *testing.T) {
	m, rec := newCopyModel(t)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "so far\x1b[0m"})
	m = browse(m)
	tb := m.tr.Items()[m.tr.Len()-1].Text
	if !tb.Streaming || tb.Source != "" || m.sel == 0 {
		t.Fatalf("setup: Streaming %v, Source %q, sel %d; want a streaming reply, no Source, selected", tb.Streaming, tb.Source, m.sel)
	}
	next, cmd := pressC(m)
	runCopyCmd(t, next, cmd)
	if len(rec.got) != 1 || rec.got[0] != "so far" {
		t.Fatalf("c on a streaming reply handed %q to Copy, want [\"so far\"]", rec.got)
	}

	m = drive(m, TurnCompleteMsg{})
	if m.tr.Items()[m.tr.Len()-1].Text.Source == "" {
		t.Fatal("setup: the finished reply has no Source")
	}
	next, cmd = pressC(m)
	runCopyCmd(t, next, cmd)
	if len(rec.got) != 2 || rec.got[1] != "so far" {
		t.Errorf("c on the finished reply handed %q to Copy, want the same text", rec.got)
	}
}

// TestCInApprovalPendingReleasesTheCard: `c` is a Browsing key only. With an
// approval pending it is an ordinary key that releases the card and shows the
// release hint, and it starts no copy. Entry point: the `c` rune through Update
// on the fixture, which opens with an approval pending. One control is Down,
// a navigation key, which leaves the card armed on the same path. The other
// resolves the card with `y`, appends a reply, selects it and enters Browsing
// with Shift+Up, where `c` on the same model does start a copy, so the copy
// path is live and the refusal above comes from ApprovalPending.
func TestCInApprovalPendingReleasesTheCard(t *testing.T) {
	m := newDriven(t)
	m.Copy = (&copyRecorder{}).copy
	if m.mode() != ModeApprovalPending || m.approvalReleased {
		t.Fatalf("setup: mode %v, released %v, want an armed approval", m.mode(), m.approvalReleased)
	}
	if got := drive(m, keyType(tea.KeyDown)); got.approvalReleased {
		t.Fatal("control: Down released the card")
	}
	seq := m.copySeq
	got, cmd := pressC(m)
	if !got.approvalReleased || got.comp.Hint != approvalReleaseHint {
		t.Errorf("c: released %v, hint %q; want released with %q", got.approvalReleased, got.comp.Hint, approvalReleaseHint)
	}
	if cmd != nil || got.copySeq != seq {
		t.Errorf("c in ApprovalPending started a copy: Cmd %v, copySeq %d (was %d)", cmd != nil, got.copySeq, seq)
	}

	r := drive(m, key('y'))
	reply := r.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{"after"}, Source: "after"}})
	r.sel = reply
	r = browse(r)
	if r.mode() != ModeBrowsing || r.sel != reply {
		t.Fatalf("control setup: mode %v, sel %d; want Browsing on the reply %d", r.mode(), r.sel, reply)
	}
	if got, cmd := pressC(r); cmd == nil || got.copySeq == r.copySeq {
		t.Errorf("control: c on a reply after the card resolved started no copy (Cmd %v, copySeq %d)", cmd != nil, got.copySeq)
	}
}

// browsingHint is the key-hint row with its padding and margin removed.
func browsingHint(m Model) string { return strings.TrimSpace(m.hintRow(m.layout())) }

// TestBrowsingHintRowNamesCOnAReply: the Browsing key-hint row on a selected
// reply reads "↑↓ select · c copy · Tab/Esc prompt"; on a selected card it
// keeps the old row. Entry point: hintRow after Shift+Up and a selection change,
// at 80 columns (both glyph sets) and at 40. At 40 columns the content width is
// 38: the reply row (35 cells) fits whole, and the card row is cut from the end,
// so "↑↓ select", "Enter preview" and "d detail" survive and "Tab/Esc prompt" is
// dropped.
func TestBrowsingHintRowNamesCOnAReply(t *testing.T) {
	cases := []struct {
		name        string
		caps        Caps
		width       int
		reply, card string
	}{
		{
			"80 columns",
			Caps{Unicode: true},
			80,
			"↑↓ select · c copy · Tab/Esc prompt",
			"↑↓ select · Enter preview · d detail · Tab/Esc prompt",
		},
		{
			"80 columns ASCII",
			Caps{Unicode: false},
			80,
			"Up/Down select . c copy . Tab/Esc prompt",
			"Up/Down select . Enter preview . d detail . Tab/Esc prompt",
		},
		{
			"40 columns",
			Caps{Unicode: true},
			40,
			"↑↓ select · c copy · Tab/Esc prompt",
			"↑↓ select · Enter preview · d detail",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := newMixedTranscript(t)
			m := New(Options{Caps: c.caps})
			m.Copy = x.rec.copy
			m.tr = x.m.tr
			m = browse(drive(m, tea.WindowSizeMsg{Width: c.width, Height: 24}))

			m.sel = x.tool
			if got := browsingHint(m); got != c.card {
				t.Errorf("card selected: hint = %q, want %q", got, c.card)
			}
			m.sel = x.replyB
			if got := browsingHint(m); got != c.reply {
				t.Errorf("reply selected: hint = %q, want %q", got, c.reply)
			}
		})
	}
}
