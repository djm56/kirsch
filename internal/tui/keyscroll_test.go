package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// pageRows is the transcript page size the spec fixes for PgUp/PgDn in both
// modes: the transcript height minus one row, and at least one (ui-spec §2.4).
func pageRows(m Model) int { return maxInt(1, m.layout().TranscriptH-1) }

// walkToBottomMinusOne scrolls a Browsing model with j until its offset is one
// row short of the bottom, and fails if the model pins on the way. Every step is
// a tea.KeyMsg through Update, so it exercises the same arm the tests below
// assert on.
func walkToBottomMinusOne(t *testing.T, m Model) Model {
	t.Helper()
	target := pinnedBottom(m) - 1
	for i := 0; m.scroll.Offset < target; i++ {
		if i > len(m.lines) {
			t.Fatalf("setup: j never reached offset %d (stuck at %d)", target, m.scroll.Offset)
		}
		m = drive(m, key('j'))
		if m.scroll.Pinned {
			t.Fatalf("setup: pinned at offset %d, before the last row short of the bottom (%d)",
				m.scroll.Offset, target)
		}
	}
	return m
}

// TestComposingPgUpPgDnScrollOnePage asserts that PgUp and PgDn, delivered to
// Update as tea.KeyMsg while the mode is Composing, move the viewport by exactly
// one page (TranscriptH-1) and leave the mode, the selection and the composer's
// text untouched. The page size is asserted as a number rather than "some
// scroll" so a page of TranscriptH rows fails.
func TestComposingPgUpPgDnScrollOnePage(t *testing.T) {
	m := drive(composingUnpinned(t), key('d'), key('r'))
	if m.mode() != ModeComposing || m.comp.Value() != "dr" {
		t.Fatalf("precondition: mode %v, composer %q; want Composing and %q", m.mode(), m.comp.Value(), "dr")
	}
	// One page up first, so both directions have a full page of room and neither
	// lands on an end (the bottom would re-pin, and that is the next test's job).
	m = drive(m, keyType(tea.KeyPgUp))
	p := pageRows(m)
	start, sel := m.scroll.Offset, m.sel
	if start-p < 0 || start+p >= pinnedBottom(m) {
		t.Fatalf("precondition: offset %d with page %d runs into an end (bottom %d); the page size would be clamped",
			start, p, pinnedBottom(m))
	}

	up := drive(m, keyType(tea.KeyPgUp))
	if up.scroll.Offset != start-p {
		t.Errorf("PgUp in Composing: offset %d -> %d, want %d (one page of %d rows)", start, up.scroll.Offset, start-p, p)
	}
	down := drive(m, keyType(tea.KeyPgDown))
	if down.scroll.Offset != start+p {
		t.Errorf("PgDn in Composing: offset %d -> %d, want %d (one page of %d rows)", start, down.scroll.Offset, start+p, p)
	}
	for name, got := range map[string]Model{"PgUp": up, "PgDn": down} {
		if got.mode() != ModeComposing {
			t.Errorf("%s changed the mode to %v", name, got.mode())
		}
		if got.sel != sel {
			t.Errorf("%s moved the selection %d -> %d", name, sel, got.sel)
		}
		if got.comp.Value() != "dr" {
			t.Errorf("%s changed the composer text to %q", name, got.comp.Value())
		}
		if got.scroll.Pinned {
			t.Errorf("%s left the transcript pinned mid-scroll at offset %d", name, got.scroll.Offset)
		}
	}
}

// TestComposingPgUpLeavesHintAlone asserts, through Update with a PgUp and a
// PgDn KeyMsg in Composing, that an "unknown command" hint under the composer
// survives. A key that fell through to the textarea would clear the hint on the
// way out, which is the only observable difference since neither key carries
// runes. This also covers "the textarea never sees the key": bubbles v1.0.0's
// textarea has no PgUp/PgDn binding, so the fall-through would be silent
// otherwise.
func TestComposingPgUpLeavesHintAlone(t *testing.T) {
	m := typeCmd(composingUnpinned(t), "/nope")
	if m.comp.Hint == "" {
		t.Fatal("precondition: /nope set no hint")
	}
	want := m.comp.Hint
	for name, k := range map[string]tea.KeyMsg{"PgUp": keyType(tea.KeyPgUp), "PgDn": keyType(tea.KeyPgDown)} {
		if got := drive(m, k).comp.Hint; got != want {
			t.Errorf("%s changed the hint from %q to %q", name, want, got)
		}
	}
}

// TestComposingPgUpPgDnWorkWhileBusy asserts that PgUp and PgDn scroll in
// Composing while a turn is live (busy.Active), via Update with a KeyMsg. They
// sit ahead of keyComposing's busy guard, which swallows everything else. The
// positive control is the same fixture's scroll when idle (the other composing
// test), so a no-op here is the guard, not a clamp.
func TestComposingPgUpPgDnWorkWhileBusy(t *testing.T) {
	m := drive(composingUnpinned(t), ToolStartedMsg{ID: 42, Name: "run_command", Target: "go test ./..."})
	if !m.busy.Active || m.scroll.Pinned {
		t.Fatalf("precondition: busy=%v pinned=%v; want a live turn and an unpinned transcript", m.busy.Active, m.scroll.Pinned)
	}
	p, start := pageRows(m), m.scroll.Offset
	if up := drive(m, keyType(tea.KeyPgUp)); up.scroll.Offset != start-p {
		t.Errorf("PgUp while busy: offset %d -> %d, want %d", start, up.scroll.Offset, start-p)
	}
	if down := drive(m, keyType(tea.KeyPgDown)); down.scroll.Offset != start+p {
		t.Errorf("PgDn while busy: offset %d -> %d, want %d", start, down.scroll.Offset, start+p)
	}
}

// TestComposingPgUpFromThePinnedBottom asserts, through Update with PgUp and
// PgDn KeyMsgs in Composing, the first PgUp from a transcript that is still
// pinned — every other Composing test here starts unpinned. From the pinned
// bottom, PgUp lands exactly one page above the bottom and unpins, and PgDn from
// there lands back on the bottom and re-pins. PgDn on the pinned bottom stays
// there, pinned. On its own that assertion would also pass for a PgDn that does
// nothing, so its control is the one before it (`back`): PgDn from one page up
// must move onto the bottom and re-pin, which a PgDn that does nothing fails.
//
// While pinned, Offset is whatever relayout last derived (model.go relayout),
// and the precondition asserts it is the bottom. The PgUp assertion wants
// exactly one page above that, so this test fails if the arm starts PgUp from a
// pinned position that differs from the one relayout derived.
func TestComposingPgUpFromThePinnedBottom(t *testing.T) {
	m := drive(newDrivenSize(t, 80, 12), key('y'))
	p, bottom := pageRows(m), pinnedBottom(m)
	if m.mode() != ModeComposing || !m.scroll.Pinned || m.scroll.Offset != bottom {
		t.Fatalf("precondition: mode %v pinned %v offset %d; want Composing, pinned, at the bottom %d",
			m.mode(), m.scroll.Pinned, m.scroll.Offset, bottom)
	}
	if bottom-p <= 0 {
		t.Fatalf("precondition: bottom %d is not more than one page (%d) from the top; PgUp would clamp to 0", bottom, p)
	}
	sel := m.sel

	up := drive(m, keyType(tea.KeyPgUp))
	if up.scroll.Offset != bottom-p || up.scroll.Pinned {
		t.Errorf("PgUp from the pinned bottom: offset %d pinned %v, want %d (one page of %d above the bottom %d) and unpinned",
			up.scroll.Offset, up.scroll.Pinned, bottom-p, p, bottom)
	}
	if back := drive(up, keyType(tea.KeyPgDown)); back.scroll.Offset != bottom || !back.scroll.Pinned {
		t.Errorf("PgDn one page back: offset %d pinned %v, want the bottom %d and pinned",
			back.scroll.Offset, back.scroll.Pinned, bottom)
	}
	if down := drive(m, keyType(tea.KeyPgDown)); down.scroll.Offset != bottom || !down.scroll.Pinned {
		t.Errorf("PgDn on the pinned bottom: offset %d pinned %v, want %d and still pinned",
			down.scroll.Offset, down.scroll.Pinned, bottom)
	}
	if up.mode() != ModeComposing || up.sel != sel {
		t.Errorf("PgUp from the pinned bottom: mode %v sel %d; want Composing and sel %d unchanged", up.mode(), up.sel, sel)
	}
}

// TestComposingPgDnToBottomRePinsAndClearsNewSince asserts, through Update with
// PgDn KeyMsgs in Composing, that scrolling down onto the bottom re-pins and
// clears the "↓ N new" count and its indicator.
func TestComposingPgDnToBottomRePinsAndClearsNewSince(t *testing.T) {
	m := drive(composingUnpinned(t), NoticeMsg{Text: "first"}, NoticeMsg{Text: "second"})
	if m.scroll.NewSince != 2 || m.scroll.Pinned {
		t.Fatalf("precondition: NewSince %d pinned %v; want 2 and unpinned", m.scroll.NewSince, m.scroll.Pinned)
	}
	for i := 0; !m.scroll.Pinned; i++ {
		if i > len(m.lines) {
			t.Fatal("PgDn never reached the bottom")
		}
		m = drive(m, keyType(tea.KeyPgDown))
	}
	if m.scroll.NewSince != 0 {
		t.Errorf("PgDn to the bottom left NewSince = %d, want 0", m.scroll.NewSince)
	}
	if m.scroll.Offset != pinnedBottom(m) {
		t.Errorf("pinned at offset %d, want the bottom %d", m.scroll.Offset, pinnedBottom(m))
	}
	if hasIndicator(m, transcriptBand(t, m)) {
		t.Error("the new-content indicator is still drawn after PgDn re-pinned")
	}
	if m.mode() != ModeComposing {
		t.Errorf("mode = %v, want Composing", m.mode())
	}
}

// TestBrowsingPgDnToBottomRePinsAndClearsNewSince is the Browsing half of the
// test above, through Update with PgDn KeyMsgs.
func TestBrowsingPgDnToBottomRePinsAndClearsNewSince(t *testing.T) {
	m := drive(scrolledUp(t), NoticeMsg{Text: "first"}, NoticeMsg{Text: "second"})
	if m.scroll.NewSince != 2 || m.scroll.Pinned || m.mode() != ModeBrowsing {
		t.Fatalf("precondition: NewSince %d pinned %v mode %v", m.scroll.NewSince, m.scroll.Pinned, m.mode())
	}
	for i := 0; !m.scroll.Pinned; i++ {
		if i > len(m.lines) {
			t.Fatal("PgDn never reached the bottom")
		}
		m = drive(m, keyType(tea.KeyPgDown))
	}
	if m.scroll.NewSince != 0 {
		t.Errorf("PgDn to the bottom left NewSince = %d, want 0", m.scroll.NewSince)
	}
	if hasIndicator(m, transcriptBand(t, m)) {
		t.Error("the new-content indicator is still drawn after PgDn re-pinned")
	}
}

// TestBrowsingJKScrollOneRowWithoutMovingSelection asserts, through Update with
// rune KeyMsgs in Browsing, that j scrolls down exactly one row and k up exactly
// one, and that neither changes the selection or the mode.
func TestBrowsingJKScrollOneRowWithoutMovingSelection(t *testing.T) {
	m := scrolledUp(t)
	start, sel := m.scroll.Offset, m.sel
	if start <= 0 || start >= pinnedBottom(m)-1 {
		t.Fatalf("precondition: offset %d is at an end (bottom %d)", start, pinnedBottom(m))
	}
	j := drive(m, key('j'))
	k := drive(m, key('k'))
	if j.scroll.Offset != start+1 {
		t.Errorf("j: offset %d -> %d, want %d", start, j.scroll.Offset, start+1)
	}
	if k.scroll.Offset != start-1 {
		t.Errorf("k: offset %d -> %d, want %d", start, k.scroll.Offset, start-1)
	}
	for name, got := range map[string]Model{"j": j, "k": k} {
		if got.sel != sel {
			t.Errorf("%s moved the selection %d -> %d", name, sel, got.sel)
		}
		if got.mode() != ModeBrowsing {
			t.Errorf("%s changed the mode to %v", name, got.mode())
		}
	}
}

// TestBrowsingKStopsAtTheTop asserts, through Update with rune KeyMsgs in
// Browsing, that k at offset 0 stays at 0 and unpinned (a clamp, not a wrap or
// an underflow). Its control comes first and is asserted step by step: from
// offset 0, j alone moves to offset 1, and k from there moves back to 0. That
// shows k is live on this model, so k leaving offset 0 unchanged is the clamp
// rather than a k that does nothing.
func TestBrowsingKStopsAtTheTop(t *testing.T) {
	m := drive(scrolledUp(t), key('g'))
	if m.scroll.Offset != 0 || m.scroll.Pinned {
		t.Fatalf("precondition: g left offset %d pinned %v, want 0 and unpinned", m.scroll.Offset, m.scroll.Pinned)
	}
	if pinnedBottom(m) <= 1 {
		t.Fatalf("precondition: bottom is offset %d; offset 1 would be the bottom and j would re-pin", pinnedBottom(m))
	}

	j := drive(m, key('j'))
	if j.scroll.Offset != 1 || j.scroll.Pinned {
		t.Fatalf("control: j from offset 0 gave offset %d pinned %v, want 1 and unpinned", j.scroll.Offset, j.scroll.Pinned)
	}
	if back := drive(j, key('k')); back.scroll.Offset != 0 || back.scroll.Pinned {
		t.Errorf("control: k from offset 1 gave offset %d pinned %v, want 0 and unpinned", back.scroll.Offset, back.scroll.Pinned)
	}

	if got := drive(m, key('k')); got.scroll.Offset != 0 || got.scroll.Pinned {
		t.Errorf("k at offset 0 gave offset %d pinned %v, want 0 and unpinned", got.scroll.Offset, got.scroll.Pinned)
	}
}

// TestBrowsingJRePinsAtTheBottomAndStops asserts, through Update with rune
// KeyMsgs in Browsing, that j taking the viewport onto the last row re-pins and
// clears NewSince, that the row before it does not, and that a further j stays
// at the bottom.
func TestBrowsingJRePinsAtTheBottomAndStops(t *testing.T) {
	m := drive(scrolledUp(t), NoticeMsg{Text: "first"}, NoticeMsg{Text: "second"})
	if m.scroll.NewSince != 2 {
		t.Fatalf("precondition: NewSince = %d, want 2", m.scroll.NewSince)
	}
	m = walkToBottomMinusOne(t, m)
	if m.scroll.Pinned || m.scroll.NewSince != 2 {
		t.Fatalf("precondition: one row short of the bottom, pinned %v NewSince %d; want unpinned and 2",
			m.scroll.Pinned, m.scroll.NewSince)
	}
	m = drive(m, key('j'))
	if !m.scroll.Pinned {
		t.Error("j onto the bottom row did not re-pin")
	}
	if m.scroll.NewSince != 0 {
		t.Errorf("j onto the bottom row left NewSince = %d, want 0", m.scroll.NewSince)
	}
	if m.scroll.Offset != pinnedBottom(m) {
		t.Errorf("offset %d, want the bottom %d", m.scroll.Offset, pinnedBottom(m))
	}
	if again := drive(m, key('j')); again.scroll.Offset != m.scroll.Offset || !again.scroll.Pinned {
		t.Errorf("j past the bottom: offset %d pinned %v, want %d and pinned", again.scroll.Offset, again.scroll.Pinned, m.scroll.Offset)
	}
}

// TestApprovalJKReleaseTheCardAndPgKeysDoNot asserts, through Update with
// KeyMsgs in ApprovalPending, that j and k are ordinary keys there (each
// releases the card and shows the release hint, and neither scrolls), while
// PgUp and PgDn are navigation keys that neither release the card nor scroll.
//
// The model is parked unpinned at offset 10. The precondition asserts that the
// bottom offset is greater than 10, so a scroll down of a row or a page would
// change the offset rather than be clamped away; that 10 is greater than the top
// offset (0) holds by construction, not by assertion, so a scroll up would
// change it too. j/k releasing and setting the release hint is the positive
// control for the PgUp/PgDn release and hint checks: the same path does release
// on an ordinary key. The no-scroll checks have no same-path control, because no
// key scrolls in ApprovalPending; they rest on the parked offset having room to
// move in both directions.
func TestApprovalJKReleaseTheCardAndPgKeysDoNot(t *testing.T) {
	base := newDrivenSize(t, 80, 12)
	if base.mode() != ModeApprovalPending || base.approvalReleased {
		t.Fatalf("precondition: mode %v released %v; want an armed approval", base.mode(), base.approvalReleased)
	}
	const start = 10
	if pinnedBottom(base) <= start {
		t.Fatalf("precondition: bottom is offset %d; offset %d is not strictly above it, so a scroll down would be clamped away",
			pinnedBottom(base), start)
	}
	base.scroll = Scroll{Offset: start, Pinned: false}

	for _, r := range []rune{'j', 'k'} {
		got := drive(base, key(r))
		if !got.approvalReleased {
			t.Errorf("%q did not release the approval card", r)
		}
		if got.comp.Hint != approvalReleaseHint {
			t.Errorf("%q left hint %q, want the release hint", r, got.comp.Hint)
		}
		if got.scroll.Offset != start {
			t.Errorf("%q scrolled in ApprovalPending: offset %d -> %d", r, start, got.scroll.Offset)
		}
	}
	for name, kt := range map[string]tea.KeyType{"PgUp": tea.KeyPgUp, "PgDn": tea.KeyPgDown} {
		got := drive(base, keyType(kt))
		if got.approvalReleased {
			t.Errorf("%s released the approval card", name)
		}
		if got.comp.Hint != base.comp.Hint {
			t.Errorf("%s changed the hint to %q", name, got.comp.Hint)
		}
		if got.scroll.Offset != start || got.scroll.Pinned {
			t.Errorf("%s scrolled in ApprovalPending: offset %d -> %d", name, start, got.scroll.Offset)
		}
		if got.mode() != ModeApprovalPending {
			t.Errorf("%s left ApprovalPending for %v", name, got.mode())
		}
	}
}

// TestHelpBindingRowsMatchAmendment101 asserts the rendered left column of the
// help overlay, read from helpColumn (what helpLines composes), carries the
// rows amendment 101 gives. Screen 06's golden pins the whole frame; this names
// the rows so a restored old row fails here with a readable message.
func TestHelpBindingRowsMatchAmendment101(t *testing.T) {
	want := []string{
		"composing",
		"Enter         send",
		"Shift+Enter   newline (or Ctrl+J)",
		"↑/↓           history",
		"Shift+↑       cards",
		"Tab           complete /cmd · cards",
		"Esc           cancel turn",
		"Ctrl+G        bottom, re-pin",
		"PgUp/PgDn     scroll",
		"",
		"browsing",
		"↑/↓           select card/reply",
		"PgUp/PgDn j/k scroll",
		"Enter         toggle preview",
		"d             diff / content",
		"g/G End       top / bottom, re-pin",
		"c             copy reply",
	}
	got := helpColumn(13, "composing", "browsing")
	if len(got) != len(want) {
		t.Fatalf("left column has %d rows, want %d:\n%q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}
