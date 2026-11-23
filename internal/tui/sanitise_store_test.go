package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The tests in this file hold the store-time sanitising of bidi controls, of \r in
// single-row fields, and of error and notice text. Each store site is driven through
// Update with the message the app sends, and the stored card is read back, so a site
// that stores its text raw fails the test written for it.

// dirty is text a hostile source could send: a bidi override and a newline in the
// middle. Stored sanitised it reads "x·y z".
var dirty = "x" + string(rune(0x202e)) + "y\nz"

// dirtyWant is dirty as it must be stored.
const dirtyWant = "x·y z"

// storeModel is a model at 100x30 with no content.
func storeModel(t *testing.T) Model {
	t.Helper()
	return drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 100, Height: 30})
}

// lastOfKind is the last transcript item of the kind, or nil.
func lastOfKind(m Model, k ItemKind) *Item {
	items := m.tr.Items()
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Kind == k {
			return &items[i]
		}
	}
	return nil
}

// TestSanitizeReplacesBidiControls: entry point Sanitize, called directly. Every bidi
// format control is shown as a bullet and not deleted, and the text beside it is kept.
// Controls: the runes in the second loop's slice are not bidi controls, and each
// lies next to or between the replaced ranges; they must be left alone, so the
// test can tell a range that is right from one that is too wide.
func TestSanitizeReplacesBidiControls(t *testing.T) {
	bidi := []rune{0x061c, 0x200e, 0x200f, 0x202a, 0x202b, 0x202c, 0x202d, 0x202e, 0x2066, 0x2067, 0x2068, 0x2069}
	for _, r := range bidi {
		if got, want := Sanitize("a"+string(r)+"b"), "a·b"; got != want {
			t.Errorf("Sanitize with U+%04X is %q, want %q", r, got, want)
		}
	}
	for _, r := range []rune{0x061b, 0x061d, 0x200d, 0x2010, 0x2029, 0x2060, 0x202f, 0x2065, 0x206a} {
		in := "a" + string(r) + "b"
		if got := Sanitize(in); got != in {
			t.Errorf("control: Sanitize changed U+%04X, got %q", r, got)
		}
	}
}

// TestSanitizeShowsReplacementRuneAndInvalidBytesAsBullets: entry point Sanitize,
// called directly. A literal U+FFFD rune and an invalid UTF-8 byte ("\xff", which
// strings.Map hands to the mapping function as U+FFFD) are each shown as a bullet,
// with the text beside them kept. Control, through the same call: a non-control
// rune (é, U+00E9) beside the same text is left unchanged, so the bullets are the
// two inputs being replaced and not the whole line.
func TestSanitizeShowsReplacementRuneAndInvalidBytesAsBullets(t *testing.T) {
	for name, in := range map[string]string{
		"literal U+FFFD":     "a" + string(rune(0xfffd)) + "b",
		"invalid byte \\xff": "a\xffb",
	} {
		if got, want := Sanitize(in), "a·b"; got != want {
			t.Errorf("%s: Sanitize(%q) = %q, want %q", name, in, got, want)
		}
	}
	if got, want := Sanitize("aéb"), "aéb"; got != want {
		t.Errorf("control: Sanitize changed a non-control rune, got %q, want %q", got, want)
	}
}

// TestSanitizeSingleLineCarriageReturnKeepsText: entry point SanitizeSingleLine,
// called directly. A \r becomes a space and the text before it is kept, and a CRLF
// pair becomes one space. A CR followed by a CRLF is two spaces (the CRLF pair is
// read first), and a trailing CR is one space. Control: a plain string passes
// through unchanged.
func TestSanitizeSingleLineCarriageReturnKeepsText(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"lone CR", "before\rafter", "before after"},
		{"CR run", "one\rtwo\rthree", "one two three"},
		{"CRLF pair", "before\r\nafter", "before after"},
		{"LF", "before\nafter", "before after"},
		{"CR then CRLF", "a\r\r\nb", "a  b"},
		{"trailing CR", "a\r", "a "},
		{"no CR", "plain", "plain"},
	}
	for _, c := range cases {
		if got := SanitizeSingleLine(c.in); got != c.want {
			t.Errorf("%s: SanitizeSingleLine(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// TestSanitizeMultiLineCarriageReturnKeepsLastSegment: entry point Sanitize, called
// directly. This is the control for the single-line rule: in multi-line output a \r
// run still shows its last segment, as a progress bar finishes, and a CRLF file keeps
// every line.
func TestSanitizeMultiLineCarriageReturnKeepsLastSegment(t *testing.T) {
	if got, want := Sanitize("10%\r50%\r100%"), "100%"; got != want {
		t.Errorf("Sanitize of a \\r run is %q, want %q", got, want)
	}
	if got, want := Sanitize("a\r\nb\r\nc"), "a\nb\nc"; got != want {
		t.Errorf("Sanitize of a CRLF file is %q, want %q", got, want)
	}
}

// TestErrorMsgStoresKindSanitised: entry point Update with ErrorMsg. The stored
// ErrorCard.Kind, the box title, is the single-line sanitised text. Only Kind carries
// the dirty text, so a Message stored raw cannot satisfy or fail this test.
func TestErrorMsgStoresKindSanitised(t *testing.T) {
	m := drive(storeModel(t), ErrorMsg{Kind: dirty, Message: "clean"})
	it := lastOfKind(m, KindError)
	if it == nil {
		t.Fatal("no error card stored")
	}
	if it.Err.Kind != dirtyWant {
		t.Errorf("stored Kind is %q, want %q", it.Err.Kind, dirtyWant)
	}
}

// TestErrorMsgStoresMessageSanitised: entry point Update with ErrorMsg. The stored
// ErrorCard.Message is the single-line sanitised text. It is cleaned at the store
// because renderError draws it through wrap, and wrap strips no escapes or control
// characters (view.go). Only Message carries the dirty text.
func TestErrorMsgStoresMessageSanitised(t *testing.T) {
	m := drive(storeModel(t), ErrorMsg{Kind: "clean", Message: dirty})
	it := lastOfKind(m, KindError)
	if it == nil {
		t.Fatal("no error card stored")
	}
	if it.Err.Message != dirtyWant {
		t.Errorf("stored Message is %q, want %q", it.Err.Message, dirtyWant)
	}
}

// TestTurnErrorMsgStoresKindSanitised: entry point Update with TurnErrorMsg. The
// stored ErrorCard.Kind is the single-line sanitised text. Only Kind carries the
// dirty text.
func TestTurnErrorMsgStoresKindSanitised(t *testing.T) {
	m := drive(storeModel(t), TurnErrorMsg{Kind: dirty, Message: "clean"})
	it := lastOfKind(m, KindError)
	if it == nil {
		t.Fatal("no error card stored")
	}
	if it.Err.Kind != dirtyWant {
		t.Errorf("stored Kind is %q, want %q", it.Err.Kind, dirtyWant)
	}
}

// TestTurnErrorMsgStoresMessageSanitised: entry point Update with TurnErrorMsg. The
// stored ErrorCard.Message is the single-line sanitised text. Only Message carries
// the dirty text.
func TestTurnErrorMsgStoresMessageSanitised(t *testing.T) {
	m := drive(storeModel(t), TurnErrorMsg{Kind: "clean", Message: dirty})
	it := lastOfKind(m, KindError)
	if it == nil {
		t.Fatal("no error card stored")
	}
	if it.Err.Message != dirtyWant {
		t.Errorf("stored Message is %q, want %q", it.Err.Message, dirtyWant)
	}
}

// TestNoticeMsgInsertBeforeStoresSanitised: entry point Update with ToolStartedMsg,
// then NoticeMsg whose BeforeTool names that card, which takes the insertBlockBefore
// path and not the m.notice helper. Control: the notice is found directly before the
// tool card, so the test is known to have taken the insert path.
func TestNoticeMsgInsertBeforeStoresSanitised(t *testing.T) {
	m := drive(storeModel(t), ToolStartedMsg{ID: 7, Name: "read_file"}, NoticeMsg{Text: dirty, BeforeTool: 7})
	items := m.tr.Items()
	n := len(items)
	if n < 2 || items[n-1].Kind != KindTool || items[n-2].Kind != KindNotice {
		t.Fatalf("control: the notice was not inserted before the tool card; kinds end %v", kindsOf(items))
	}
	if got := items[n-2].Notice.Text; got != dirtyWant {
		t.Errorf("stored notice text is %q, want %q", got, dirtyWant)
	}
}

// TestNoticeMsgAppendedStoresSanitised: entry point Update with NoticeMsg that has no
// BeforeTool, which goes through the m.notice helper. That helper is the one every
// other notice goes through.
func TestNoticeMsgAppendedStoresSanitised(t *testing.T) {
	m := drive(storeModel(t), NoticeMsg{Text: dirty})
	it := lastOfKind(m, KindNotice)
	if it == nil {
		t.Fatal("no notice stored")
	}
	if it.Notice.Text != dirtyWant {
		t.Errorf("stored notice text is %q, want %q", it.Notice.Text, dirtyWant)
	}
}

// kindsOf lists the kinds of items, for a failure message.
func kindsOf(items []Item) []ItemKind {
	out := make([]ItemKind, 0, len(items))
	for _, it := range items {
		out = append(out, it.Kind)
	}
	return out
}

// TestSanitiseToolHeadSanitisesNote: entry point Update with TranscriptLoadedMsg, the
// path that calls sanitiseToolHead. The card's Note is the single-line sanitised
// text. Control: Name is stored clean too, and a Note with nothing to clean is kept
// as it is.
func TestSanitiseToolHeadSanitisesNote(t *testing.T) {
	m := drive(storeModel(t), TranscriptLoadedMsg{Items: []Item{
		{Kind: KindTool, Tool: &ToolCard{Name: "read_file", Note: dirty, State: StateOK}},
		{Kind: KindTool, Tool: &ToolCard{Name: "list_files", Note: "session grant: go test", State: StateOK}},
	}})
	var cards []*ToolCard
	for _, it := range m.tr.Items() {
		if it.Kind == KindTool {
			cards = append(cards, it.Tool)
		}
	}
	if len(cards) != 2 {
		t.Fatalf("stored %d tool cards, want 2", len(cards))
	}
	if cards[0].Note != dirtyWant {
		t.Errorf("stored Note is %q, want %q", cards[0].Note, dirtyWant)
	}
	if cards[1].Note != "session grant: go test" {
		t.Errorf("control: a clean Note was stored as %q", cards[1].Note)
	}
}

// TestApprovalSubjectKeepsTextBeforeCarriageReturn: entry point Update with
// ApprovalRequestedMsg. The stored subject keeps the text before a \r, and a CRLF
// pair is one space. Control: a subject with no \r is stored as sent.
func TestApprovalSubjectKeepsTextBeforeCarriageReturn(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"lone CR", "go test\rok", "go test ok"},
		{"CRLF", "go test\r\nok", "go test ok"},
		{"clean", "go test ./...", "go test ./..."},
	}
	for _, c := range cases {
		m := drive(storeModel(t), ApprovalRequestedMsg{ID: 1, Kind: "command", Description: "run", Subject: c.in})
		it := lastOfKind(m, KindApproval)
		if it == nil {
			t.Fatalf("%s: no approval card stored", c.name)
		}
		if got := it.Approval.Subject; got != c.want || strings.Contains(got, "\r") {
			t.Errorf("%s: stored subject is %q, want %q", c.name, got, c.want)
		}
	}
}

// overLength is a line longer than MaxRenderedLineWidth columns.
var overLength = strings.Repeat("x", MaxRenderedLineWidth+200)

// assertCut fails unless got is cut at MaxRenderedLineWidth columns and ends in the
// "⋯" marker. Control: want is the same text cut by hand, so a cut at the wrong
// column fails as well as no cut.
func assertCut(t *testing.T, what, got string) {
	t.Helper()
	want := strings.Repeat("x", MaxRenderedLineWidth-1) + "⋯"
	if got != want {
		t.Errorf("%s: stored text is %d columns ending %q, want %d columns ending %q",
			what, cellWidth(got), got[max(0, len(got)-4):], MaxRenderedLineWidth, "⋯")
	}
}

// TestStoreSitesCutOverLengthText: entry point Update with NoticeMsg (no BeforeTool,
// so the m.notice helper), NoticeMsg with BeforeTool (the insert-before path),
// ErrorMsg and TurnErrorMsg. ui-spec §7.1 caps every rendered line at MaxRenderedLineWidth columns
// with a "⋯", and the store sites follow it. Controls: the insert-before case is
// checked to sit before its tool card, and a short value is stored whole.
func TestStoreSitesCutOverLengthText(t *testing.T) {
	m := drive(storeModel(t), NoticeMsg{Text: overLength})
	it := lastOfKind(m, KindNotice)
	if it == nil {
		t.Fatal("no notice stored")
	}
	assertCut(t, "notice through m.notice", it.Notice.Text)

	m = drive(storeModel(t), ToolStartedMsg{ID: 7, Name: "read_file"}, NoticeMsg{Text: overLength, BeforeTool: 7})
	items := m.tr.Items()
	n := len(items)
	if n < 2 || items[n-1].Kind != KindTool || items[n-2].Kind != KindNotice {
		t.Fatalf("control: the notice was not inserted before the tool card; kinds end %v", kindsOf(items))
	}
	assertCut(t, "notice inserted before a tool card", items[n-2].Notice.Text)

	m = drive(storeModel(t), ErrorMsg{Kind: overLength, Message: overLength})
	er := lastOfKind(m, KindError)
	if er == nil {
		t.Fatal("no error card stored")
	}
	assertCut(t, "ErrorMsg Kind", er.Err.Kind)
	assertCut(t, "ErrorMsg Message", er.Err.Message)

	m = drive(storeModel(t), TurnErrorMsg{Kind: overLength, Message: overLength})
	te := lastOfKind(m, KindError)
	if te == nil {
		t.Fatal("no turn error card stored")
	}
	assertCut(t, "TurnErrorMsg Kind", te.Err.Kind)
	assertCut(t, "TurnErrorMsg Message", te.Err.Message)

	m = drive(storeModel(t), NoticeMsg{Text: "short"})
	if got := lastOfKind(m, KindNotice).Notice.Text; got != "short" {
		t.Errorf("control: a short notice was stored as %q", got)
	}
}

// TestApprovalsNoticeCutsLongGrantList: entry point Update with key presses typing
// "/approvals" then Enter (via approvalsNotice), so the real submit path reaches
// runSlash. Each grant is short, but the joined list is longer than
// MaxRenderedLineWidth columns, so the notice is cut there and ends in "⋯", and
// the grants past the cut are not shown.
// Control: the first grant is in the notice, so the notice did take the grant list.
func TestApprovalsNoticeCutsLongGrantList(t *testing.T) {
	grants := make([]string, 0, 300)
	for i := 0; i < 300; i++ {
		grants = append(grants, "grant-"+itoa(i))
	}
	text := approvalsNotice(t, grants)
	if !strings.HasPrefix(text, "Active session grants: grant-0, grant-1") {
		t.Fatalf("control: the notice does not start with the grant list: %q", text[:min(len(text), 60)])
	}
	if cellWidth(text) != MaxRenderedLineWidth || !strings.HasSuffix(text, "⋯") {
		t.Errorf("notice is %d columns, want %d, ending in the cut marker; ends %q",
			cellWidth(text), MaxRenderedLineWidth, text[max(0, len(text)-12):])
	}
	if strings.Contains(text, "grant-299") {
		t.Errorf("notice shows a grant past the cut: %q", text[max(0, len(text)-30):])
	}
}

// approvalsNotice returns the text of the /approvals notice for grants. Entry point:
// Update with key presses typing "/approvals" then Enter, so the real submit path
// reaches runSlash; the result is read from the last transcript notice.
func approvalsNotice(t *testing.T, grants []string) string {
	t.Helper()
	m := New(Options{})
	m.GetGrants = func() []string { return grants }
	for _, r := range "/approvals" {
		updated, _ := m.Update(key(r))
		m = updated.(Model)
	}
	updated, _ := m.Update(keyType(tea.KeyEnter))
	m = updated.(Model)
	it := lastOfKind(m, KindNotice)
	if it == nil {
		t.Fatal("no notice stored")
	}
	return it.Notice.Text
}

// TestApprovalsNoticeKeepsGrantsAroundEscapes: entry point Update with key presses
// typing "/approvals" then Enter (via approvalsNotice), so the real submit path
// reaches runSlash. Grant 1 holds an unterminated OSC (ESC ] ... with no BEL or ST)
// and grant 2 holds a BEL. Joined raw, the OSC would run from grant 1 to grant 2's
// BEL and swallow the end of grant 1, the ", " separator and the start of grant 2.
// Each grant is cleaned before the join, so both grants' text and the separator
// stay visible. This pins the per-grant SanitizeSingleLine call in runSlash.
// Control: two plain grants show both and the separator through the same path.
func TestApprovalsNoticeKeepsGrantsAroundEscapes(t *testing.T) {
	const prefix = "Active session grants: "
	if got, want := approvalsNotice(t, []string{"a", "b"}), prefix+"a, b"; got != want {
		t.Fatalf("control: notice = %q, want %q", got, want)
	}
	got := approvalsNotice(t, []string{"a\x1b]0;evil", "b\x07c"})
	// Alone, grant 1's unterminated "ESC ]" is removed as a two-byte escape, leaving
	// "a0;evil"; grant 2's BEL becomes "·".
	want := prefix + "a0;evil, b·c"
	if got != want {
		t.Errorf("notice = %q, want %q", got, want)
	}
}
