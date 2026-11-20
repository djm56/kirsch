package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// cutOffText is the notice text the app sends for a request that carried 16384
// output tokens; the card draws the leading bullet.
const cutOffText = "reply cut off — output limit reached (16384 tokens)"

// itemKinds returns the kinds of the model's transcript items, in order.
func itemKinds(m Model) []ItemKind {
	var out []ItemKind
	for _, it := range m.tr.Items() {
		out = append(out, it.Kind)
	}
	return out
}

// kindsEqual reports whether two kind lists are equal.
func kindsEqual(a, b []ItemKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// cutOffModel returns a 100x40 model.
func cutOffModel() Model {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	return drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
}

// TestNoticeBeforeToolInsertsBeforeThatCard drives Model.Update with a reply's
// text, two tool starts and then a NoticeMsg naming the first tool: the notice
// lands between the text and the first card, and the second card stays after.
func TestNoticeBeforeToolInsertsBeforeThatCard(t *testing.T) {
	m := drive(cutOffModel(),
		AssistantTextDeltaMsg{BlockID: 1, Delta: "the reply"},
		ToolStartedMsg{ID: 7, Name: "read_file", Target: "a.go"},
		ToolStartedMsg{ID: 8, Name: "run_command", Target: "go test"},
		NoticeMsg{Text: cutOffText, BeforeTool: 7},
	)
	want := []ItemKind{KindAssistant, KindNotice, KindTool, KindTool}
	if got := itemKinds(m); !kindsEqual(got, want) {
		t.Fatalf("item kinds = %v, want %v", got, want)
	}
	frame := stripSGR(m.View())
	text, note := strings.Index(frame, "the reply"), strings.Index(frame, "· "+cutOffText)
	first, second := strings.Index(frame, "read_file"), strings.Index(frame, "run_command")
	if text < 0 || note < 0 || first < 0 || second < 0 {
		t.Fatalf("frame lacks text (%d), notice (%d), first card (%d) or second card (%d):\n%s", text, note, first, second, frame)
	}
	if text >= note || note >= first || first >= second {
		t.Errorf("frame order: text %d, notice %d, first card %d, second card %d:\n%s", text, note, first, second, frame)
	}
}

// TestNoticeAppendsWhenItNamesNoCard covers two ways BeforeTool does not place a
// notice: zero, and an id no card was started for. Each appends, so the notice is
// never lost. TestNoticeBeforeToolInsertsBeforeThatCard is the positive control
// for insertion through the same message; the next test covers an id whose card
// has not arrived yet.
func TestNoticeAppendsWhenItNamesNoCard(t *testing.T) {
	for _, tc := range []struct {
		name       string
		beforeTool int64
	}{
		{"zero", 0},
		{"unknown id", 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := drive(cutOffModel(),
				AssistantTextDeltaMsg{BlockID: 1, Delta: "the reply"},
				ToolStartedMsg{ID: 7, Name: "read_file"},
				NoticeMsg{Text: cutOffText, BeforeTool: tc.beforeTool},
			)
			want := []ItemKind{KindAssistant, KindTool, KindNotice}
			if got := itemKinds(m); !kindsEqual(got, want) {
				t.Errorf("item kinds = %v, want %v (appended)", got, want)
			}
		})
	}
}

// TestNoticeBeforeToolArrivingFirstStillPrecedesTheCard covers a NoticeMsg that
// reaches the model before the ToolStartedMsg it names. The app's outbox keeps
// them in production order, so this is a guard on the model, not an order the
// app produces. The notice is appended, the card starts after it, and the
// result is the same order as when they arrive the other way.
func TestNoticeBeforeToolArrivingFirstStillPrecedesTheCard(t *testing.T) {
	m := drive(cutOffModel(),
		AssistantTextDeltaMsg{BlockID: 1, Delta: "the reply"},
		NoticeMsg{Text: cutOffText, BeforeTool: 7},
		ToolStartedMsg{ID: 7, Name: "read_file"},
	)
	want := []ItemKind{KindAssistant, KindNotice, KindTool}
	if got := itemKinds(m); !kindsEqual(got, want) {
		t.Errorf("item kinds = %v, want %v", got, want)
	}
}

// TestTranscriptInsertBefore checks Transcript.InsertBefore directly: the new
// item takes the position, gets a fresh unique id, and a missing id inserts
// nothing and reports false.
func TestTranscriptInsertBefore(t *testing.T) {
	var tr Transcript
	a := tr.Append(Item{Kind: KindUser})
	b := tr.Append(Item{Kind: KindAssistant})

	id, ok := tr.InsertBefore(b, Item{Kind: KindNotice})
	if !ok {
		t.Fatal("InsertBefore of an existing id reported false")
	}
	if id == a || id == b || id == 0 {
		t.Errorf("inserted id %d collides with %d or %d or is zero", id, a, b)
	}
	items := tr.Items()
	if len(items) != 3 || items[0].ID != a || items[1].ID != id || items[2].ID != b {
		t.Errorf("order after insert = %+v, want [a, new, b]", items)
	}

	rev := tr.rev
	if _, ok := tr.InsertBefore(9999, Item{Kind: KindNotice}); ok {
		t.Error("InsertBefore of a missing id reported true")
	}
	if tr.Len() != 3 || tr.rev != rev {
		t.Errorf("a refused insert changed the transcript: len %d, rev %d -> %d", tr.Len(), rev, tr.rev)
	}
}

// noticeRows returns the trimmed frame rows from the one holding "reply cut off"
// up to the first blank row, which is the notice as drawn.
func noticeRows(t *testing.T, frame string) []string {
	t.Helper()
	lines := strings.Split(frame, "\n")
	for i, l := range lines {
		if !strings.Contains(l, "reply cut off") {
			continue
		}
		var rows []string
		for _, r := range lines[i:] {
			if strings.TrimSpace(r) == "" {
				break
			}
			rows = append(rows, strings.TrimSpace(r))
		}
		return rows
	}
	t.Fatalf("no notice in frame:\n%s", frame)
	return nil
}

// TestCutOffNoticeWrapsAtNarrowWidths renders the 16384-token notice through
// Model.Update and View at widths 80 and 40. The existing notice rendering wraps
// at word boundaries (renderNotice calls wrap) and never clips: at 80 the notice
// is one row, at 40 it is two rows and every word is still on screen.
func TestCutOffNoticeWrapsAtNarrowWidths(t *testing.T) {
	for _, tc := range []struct {
		width int
		want  []string
	}{
		{80, []string{"· reply cut off — output limit reached (16384 tokens)"}},
		{40, []string{"· reply cut off — output limit reached", "(16384 tokens)"}},
	} {
		m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
		m = drive(m, tea.WindowSizeMsg{Width: tc.width, Height: 30},
			AssistantTextDeltaMsg{BlockID: 1, Delta: "hi"},
			NoticeMsg{Text: cutOffText})
		got := noticeRows(t, stripSGR(m.View()))
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Errorf("width %d: notice rows = %q, want %q", tc.width, got, tc.want)
		}
	}
}

// TestCutOffNoticeFoldsToAsciiOnANonUnicodeTerminal renders the notice with
// Caps.Unicode false: the bullet is drawn as "." and the dash as "-" (ui-spec
// §10.2), by the existing notice rendering, with no change to the text sent.
func TestCutOffNoticeFoldsToAsciiOnANonUnicodeTerminal(t *testing.T) {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: false}})
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30}, NoticeMsg{Text: cutOffText})
	got := noticeRows(t, stripSGR(m.View()))
	want := ". reply cut off - output limit reached (16384 tokens)"
	if len(got) != 1 || got[0] != want {
		t.Errorf("notice rows = %q, want [%q]", got, want)
	}
}

// scrolledModel returns a 100x40 model holding 60 tool cards (ids 1..60), with
// the viewport scrolled up so its top row is at row top, unpinned. It checks the
// setup took: unpinned, offset as asked, and the viewport shorter than the
// transcript.
func scrolledModel(t *testing.T, top int) Model {
	t.Helper()
	m := cutOffModel()
	for id := int64(1); id <= 60; id++ {
		m = drive(m, ToolStartedMsg{ID: id, Name: "read_file", Target: "a.go"})
	}
	h := m.layout().TranscriptH
	m.scroll.setOffset(top, len(m.lines), h)
	if m.scroll.Pinned || m.scroll.Offset != top || len(m.lines) <= h {
		t.Fatalf("setup: pinned=%v offset=%d (want %d), lines=%d, viewport=%d",
			m.scroll.Pinned, m.scroll.Offset, top, len(m.lines), h)
	}
	return m
}

// visibleRows returns the transcript rows the viewport shows.
func visibleRows(m Model) []string {
	h := m.layout().TranscriptH
	return append([]string(nil), m.lines[m.scroll.Offset:m.scroll.Offset+h]...)
}

// rowsEqual reports whether two row lists are equal.
func rowsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestNoticeInsertedAboveAnUnpinnedViewportDoesNotMoveWhatTheReaderSees drives
// Model.Update with a NoticeMsg naming a card above the top of a scrolled-up
// viewport (ui-spec §2.4: the scroll position is the user's). The rows on screen
// are the same before and after, the offset moved by exactly the rows the notice
// added, and NewSince is unchanged. The notice's own text is in the transcript
// (the control: the insert happened).
func TestNoticeInsertedAboveAnUnpinnedViewportDoesNotMoveWhatTheReaderSees(t *testing.T) {
	m := scrolledModel(t, 20)
	before, linesBefore, newBefore := visibleRows(m), len(m.lines), m.scroll.NewSince

	m = drive(m, NoticeMsg{Text: cutOffText, BeforeTool: 3})

	if !strings.Contains(strings.Join(m.plainLines, "\n"), cutOffText) {
		t.Fatal("control: the notice is not in the transcript")
	}
	if m.scroll.Pinned {
		t.Error("the insert re-pinned the viewport")
	}
	if got := visibleRows(m); !rowsEqual(got, before) {
		t.Errorf("visible rows changed:\nbefore %q\nafter  %q", before, got)
	}
	if want := 20 + len(m.lines) - linesBefore; m.scroll.Offset != want {
		t.Errorf("offset = %d, want %d (old offset plus the rows added)", m.scroll.Offset, want)
	}
	if m.scroll.NewSince != newBefore {
		t.Errorf("NewSince = %d, want %d: an insert above the viewport is not a block below the fold", m.scroll.NewSince, newBefore)
	}
}

// TestNoticeInsertedBeforeACardWhoseFirstRowIsTheViewportTop is the boundary
// between the two tests around it: the card the notice names starts on exactly
// the row the viewport shows first. insertBlockBefore reads "at or above the
// viewport top" as start <= offset, so this counts as above: the rows on screen
// are the same before and after, the offset moved by the rows the notice added
// (the card is still the first row shown), and NewSince is unchanged. The
// notice's own text in the transcript is the control that the insert happened.
func TestNoticeInsertedBeforeACardWhoseFirstRowIsTheViewportTop(t *testing.T) {
	m := scrolledModel(t, 0)
	card, ok := m.toolCards[3]
	if !ok {
		t.Fatal("setup: no card for tool 3")
	}
	row, ok := m.rowFor(card)
	if !ok || row.Start == 0 {
		t.Fatalf("setup: card 3 row = %+v, found = %v; want a start below row 0", row, ok)
	}
	top := row.Start
	m.scroll.setOffset(top, len(m.lines), m.layout().TranscriptH)
	if m.scroll.Pinned || m.scroll.Offset != top {
		t.Fatalf("setup: pinned=%v offset=%d, want unpinned at %d", m.scroll.Pinned, m.scroll.Offset, top)
	}
	before, linesBefore, newBefore := visibleRows(m), len(m.lines), m.scroll.NewSince

	m = drive(m, NoticeMsg{Text: cutOffText, BeforeTool: 3})

	if !strings.Contains(strings.Join(m.plainLines, "\n"), cutOffText) {
		t.Fatal("control: the notice is not in the transcript")
	}
	if got := visibleRows(m); !rowsEqual(got, before) {
		t.Errorf("visible rows changed:\nbefore %q\nafter  %q", before, got)
	}
	if want := top + len(m.lines) - linesBefore; m.scroll.Offset != want {
		t.Errorf("offset = %d, want %d (the card's first row plus the rows added)", m.scroll.Offset, want)
	}
	if m.scroll.NewSince != newBefore {
		t.Errorf("NewSince = %d, want %d: an insert at the viewport top is not a block below the fold", m.scroll.NewSince, newBefore)
	}
}

// TestNoticeInsertedBelowAnUnpinnedViewportTopIsCounted is the other side: the
// card the notice names is below the viewport top, so the visible rows above are
// untouched, the offset stays, and NewSince goes up by one.
func TestNoticeInsertedBelowAnUnpinnedViewportTopIsCounted(t *testing.T) {
	m := scrolledModel(t, 0)
	before, newBefore := visibleRows(m), m.scroll.NewSince

	m = drive(m, NoticeMsg{Text: cutOffText, BeforeTool: 55})

	if !strings.Contains(strings.Join(m.plainLines, "\n"), cutOffText) {
		t.Fatal("control: the notice is not in the transcript")
	}
	if m.scroll.Offset != 0 || m.scroll.Pinned {
		t.Errorf("offset = %d pinned = %v, want 0 and unpinned", m.scroll.Offset, m.scroll.Pinned)
	}
	if got := visibleRows(m); !rowsEqual(got, before) {
		t.Errorf("visible rows changed:\nbefore %q\nafter  %q", before, got)
	}
	if m.scroll.NewSince != newBefore+1 {
		t.Errorf("NewSince = %d, want %d", m.scroll.NewSince, newBefore+1)
	}
}

// TestNoticeInsertedWhilePinnedStaysPinnedAtTheBottom drives the same insert
// with the viewport at the bottom: still pinned, still showing the last rows,
// NewSince still zero.
func TestNoticeInsertedWhilePinnedStaysPinnedAtTheBottom(t *testing.T) {
	m := cutOffModel()
	for id := int64(1); id <= 60; id++ {
		m = drive(m, ToolStartedMsg{ID: id, Name: "read_file", Target: "a.go"})
	}
	h := m.layout().TranscriptH
	if !m.scroll.Pinned || len(m.lines) <= h {
		t.Fatalf("setup: pinned=%v lines=%d viewport=%d", m.scroll.Pinned, len(m.lines), h)
	}

	m = drive(m, NoticeMsg{Text: cutOffText, BeforeTool: 3})

	if !m.scroll.Pinned || m.scroll.NewSince != 0 {
		t.Errorf("pinned = %v NewSince = %d, want true and 0", m.scroll.Pinned, m.scroll.NewSince)
	}
	if want := len(m.lines) - h; m.scroll.Offset != want {
		t.Errorf("offset = %d, want %d (the bottom)", m.scroll.Offset, want)
	}
}
