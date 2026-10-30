package tui

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// assertFits fails if any line is wider than w cells or carries a raw tab.
func assertFits(t *testing.T, lines []string, w int) {
	t.Helper()
	for i, ln := range lines {
		if got := cellWidth(ln); got > w {
			t.Errorf("line %d is %d cells, want <= %d: %q", i, got, w, ln)
		}
		if strings.ContainsRune(ln, '\t') {
			t.Errorf("line %d carries a raw tab: %q", i, ln)
		}
	}
}

// TestWrapIndentedTrace200 feeds wrap a 200-column indented stack-trace line.
// The first line keeps its indentation and nothing exceeds the frame.
func TestWrapIndentedTrace200(t *testing.T) {
	indent := strings.Repeat(" ", 8)
	var words []string
	for i := 0; i < 30; i++ {
		words = append(words, "frame.func")
	}
	src := indent + strings.Join(words, " ") // 8 + 30*10 + 29 = 337 cells
	if got := cellWidth(src); got <= 200 {
		t.Fatalf("fixture is %d cells, want > 200", got)
	}
	for _, w := range []int{200, 80, 33} {
		got := wrap(src, w)
		assertFits(t, got, w)
		if len(got) < 2 {
			t.Fatalf("w=%d: want a wrapped result, got %d line(s)", w, len(got))
		}
		if !strings.HasPrefix(got[0], indent+"frame.func") {
			t.Errorf("w=%d: first line lost its indentation: %q", w, got[0])
		}
		// Nothing is lost: re-joining the words gives the original words.
		if a, b := strings.Fields(strings.Join(got, " ")), words; !slices.Equal(a, b) {
			t.Errorf("w=%d: words changed", w)
		}
	}
}

// TestWrapKeepsInteriorSpaceRuns checks runs of spaces inside a line survive.
func TestWrapKeepsInteriorSpaceRuns(t *testing.T) {
	got := wrap("a  b   c    d", 80)
	if want := []string{"a  b   c    d"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestWrapSpaceAtBreakIsConsumed pins decision (a): at a break the whole run of
// spaces goes, and the next line does not start with spaces.
func TestWrapSpaceAtBreakIsConsumed(t *testing.T) {
	got := wrap("aaaa     bbbb", 6)
	if want := []string{"aaaa", "bbbb"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestWrapHardBreaksOnlyOverlongWords checks that a word wider than w is cut at
// w cells and that a neighbouring word that fits is left whole.
func TestWrapHardBreaksOnlyOverlongWords(t *testing.T) {
	got := wrap("ab abcdefghij cd", 4)
	want := []string{"ab", "abcd", "efgh", "ij", "cd"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	assertFits(t, got, 4)
}

// TestWrapFittingWordIsNotHardBroken checks a word that fits on a fresh line
// moves whole to it, and that a word exactly w wide stays whole.
func TestWrapFittingWordIsNotHardBroken(t *testing.T) {
	got := wrap("abc defgh", 6)
	if want := []string{"abc", "defgh"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	got = wrap("abc defghi", 6)
	if want := []string{"abc", "defghi"}; !slices.Equal(got, want) {
		t.Errorf("exact-width word: got %q, want %q", got, want)
	}
}

// TestWrapSentenceBreaksAtSpaces checks ordinary prose still breaks at spaces.
func TestWrapSentenceBreaksAtSpaces(t *testing.T) {
	got := wrap("the quick brown fox jumps over the lazy dog", 15)
	want := []string{"the quick brown", "fox jumps over", "the lazy dog"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	assertFits(t, got, 15)
}

// TestWrapLineWidthBoundary checks a line exactly w cells wide is kept whole
// and that one more cell wraps it.
func TestWrapLineWidthBoundary(t *testing.T) {
	if got, want := wrap("aaa bbb", 7), []string{"aaa bbb"}; !slices.Equal(got, want) {
		t.Errorf("exact fit: got %q, want %q", got, want)
	}
	if got, want := wrap("aaa bbb", 6), []string{"aaa", "bbb"}; !slices.Equal(got, want) {
		t.Errorf("one over: got %q, want %q", got, want)
	}
}

// TestWrapTabsExpand checks a raw tab never survives wrap and expands to 4-col
// stops, leading and interior alike. §7.1.
func TestWrapTabsExpand(t *testing.T) {
	got := wrap("\tif x\t{", 80)
	// The interior tab sits at column 8, so it pads to the stop at 12.
	if want := []string{"    if x    {"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	assertFits(t, wrap("\t\tfoo bar\tbaz qux", 9), 9)
}

// TestWrapOtherSpacesBecomeSpaces checks that a newline, a carriage return, a
// no-break space (U+00A0) and an ideographic space (U+3000, 2 cells wide) each
// become one plain space, so no unmeasured or wider gap reaches the screen.
func TestWrapOtherSpacesBecomeSpaces(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"newline", "a\nb c", "a b c"},
		{"carriage return", "a\rb c", "a b c"},
		{"NBSP", "a\u00a0b c", "a b c"},
		{"ideographic space", "a\u3000b c", "a b c"},
	}
	for _, c := range cases {
		got := wrap(c.in, 80)
		if want := []string{c.want}; !slices.Equal(got, want) {
			t.Errorf("%s: got %q, want %q", c.name, got, want)
		}
	}
}

// TestWrapIndentWiderThanFrame pins decision (d): indentation of w cells or
// more is clamped to w-1 cells. A word that fits the one cell left stays on
// the indented line; a wider word moves to a fresh line at column 0.
func TestWrapIndentWiderThanFrame(t *testing.T) {
	over := strings.Repeat(" ", 20)
	if got, want := wrap(over+"x", 5), []string{"    x"}; !slices.Equal(got, want) {
		t.Errorf("one-cell word: got %q, want %q", got, want)
	}
	if got, want := wrap(over+"hi", 5), []string{"hi"}; !slices.Equal(got, want) {
		t.Errorf("wider word: got %q, want %q", got, want)
	}
	if got, want := wrap(over+"hello world", 5), []string{"hello", "world"}; !slices.Equal(got, want) {
		t.Errorf("two words: got %q, want %q", got, want)
	}
}

// TestWrapIndentThenFittingWord checks the common case: a word that fits after
// the indent stays on the indented line.
func TestWrapIndentThenFittingWord(t *testing.T) {
	if got, want := wrap("    hi there", 12), []string{"    hi there"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	// A word that fits the frame but not the space after the indent moves to a
	// fresh line; no indent-only row is emitted.
	got := wrap("      abcdef", 8)
	if want := []string{"abcdef"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestWrapNoHangingIndent pins decision (b): continuation lines start at
// column 0.
func TestWrapNoHangingIndent(t *testing.T) {
	got := wrap("  aa bb cc dd", 8)
	want := []string{"  aa bb", "cc dd"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestWrapWhitespaceOnlyLine pins decision (c): a whitespace-only line is one
// empty row.
func TestWrapWhitespaceOnlyLine(t *testing.T) {
	for _, in := range []string{" ", "      ", "\t", " \t "} {
		if got, want := wrap(in, 10), []string{""}; !slices.Equal(got, want) {
			t.Errorf("wrap(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestWrapTrailingSpacesDropped checks trailing spaces are not carried.
func TestWrapTrailingSpacesDropped(t *testing.T) {
	if got, want := wrap("abc    ", 10), []string{"abc"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestWrapWideRunes checks widths are cells, not runes.
func TestWrapWideRunes(t *testing.T) {
	got := wrap("日本語 日本語", 6)
	if want := []string{"日本語", "日本語"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	got = wrap("日本語日本語", 4)
	if want := []string{"日本", "語日", "本語"}; !slices.Equal(got, want) {
		t.Errorf("w=4: got %q, want %q", got, want)
	}
	assertFits(t, got, 4)
	// A rune wider than the frame is emitted alone rather than looping.
	got = wrap("日本", 1)
	if want := []string{"日", "本"}; !slices.Equal(got, want) {
		t.Errorf("w=1: got %q, want %q", got, want)
	}
}

// TestWrapNonPositiveWidthAndEmpty pins the unchanged edge cases.
func TestWrapNonPositiveWidthAndEmpty(t *testing.T) {
	for _, w := range []int{0, -1, -80} {
		if got, want := wrap("some text", w), []string{""}; !slices.Equal(got, want) {
			t.Errorf("w=%d: got %q, want %q", w, got, want)
		}
	}
	for _, w := range []int{80, 1, 0} {
		if got, want := wrap("", w), []string{""}; !slices.Equal(got, want) {
			t.Errorf("empty, w=%d: got %q, want %q", w, got, want)
		}
	}
}

// TestTranscriptKeepsIndentAndSpaceRuns drives the real render path: it adds a
// user card and an assistant block to the transcript through Model.appendBlock
// (a direct call, not an Update message), then reads View(). Entry point:
// Model.appendBlock + View, not wrap directly. The
// lines here are narrower than the frame; wrapping is TestTranscriptWrapsIndentedLines.
func TestTranscriptKeepsIndentAndSpaceRuns(t *testing.T) {
	m := sizedModel(t, 100, 30, false)
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{
		"        at pkg.Func(file.go:12)  [vendored]",
	}}})
	m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{
		"    indented  reply",
	}}})
	m.relayout(m.layout())
	view := m.View()
	if !strings.Contains(view, "        at pkg.Func(file.go:12)  [vendored]") {
		t.Errorf("user card lost indentation or the double space:\n%s", view)
	}
	if !strings.Contains(view, "    indented  reply") {
		t.Errorf("assistant block lost indentation or the double space:\n%s", view)
	}
}

// rowsContaining returns the rendered rows of view from the first one that
// contains first, through the last one that contains last.
func rowsContaining(t *testing.T, view, first, last string) []string {
	t.Helper()
	rows := strings.Split(view, "\n")
	lo, hi := -1, -1
	for i, r := range rows {
		if lo < 0 && strings.Contains(r, first) {
			lo = i
		}
		if strings.Contains(r, last) {
			hi = i
		}
	}
	if lo < 0 || hi < lo {
		t.Fatalf("rows with %q..%q not found in view:\n%s", first, last, view)
	}
	return rows[lo : hi+1]
}

// TestTranscriptWrapsIndentedLines drives an indented line wider than the
// frame through the real renderUser (cards.go) and renderAssistant paths via
// appendBlock + View, at width 40. The first row keeps its indent and every
// continuation row starts at column 0 with text.
func TestTranscriptWrapsIndentedLines(t *testing.T) {
	const indent = "      " // 6 spaces
	line := indent + "w1aa w2bb w3cc w4dd w5ee w6ff w7gg w8hh w9ii"
	for _, kind := range []ItemKind{KindUser, KindAssistant} {
		m := sizedModel(t, 40, 30, false)
		m.appendBlock(Item{Kind: kind, Text: &TextBlock{Lines: []string{line}}})
		m.relayout(m.layout())
		rows := rowsContaining(t, m.View(), "w1aa", "w9ii")
		// padFrame (view.go) insets every row by a one-cell margin at this
		// width; drop it so the indent is measured from the content edge.
		for i, r := range rows {
			if !strings.HasPrefix(r, " ") {
				t.Fatalf("kind %v: row %d has no frame margin: %q", kind, i, r)
			}
			rows[i] = r[1:]
		}
		if len(rows) < 2 {
			t.Fatalf("kind %v: line was not wrapped, rows=%q", kind, rows)
		}
		if !strings.HasPrefix(rows[0], indent+"w1aa") {
			t.Errorf("kind %v: first row lost its indent: %q", kind, rows[0])
		}
		for i, r := range rows[1:] {
			if !strings.HasPrefix(r, "w") {
				t.Errorf("kind %v: continuation row %d does not start at column 0: %q", kind, i+1, r)
			}
		}
	}
}

// TestTranscriptStreamedTabNeverReachesScreen drives an assistant delta that
// carries a tab on a continuation chunk. Entry point: Update with
// AssistantTextDeltaMsg, then View.
func TestTranscriptStreamedTabNeverReachesScreen(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "start"})
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "\tindented"})
	view := m.View()
	if strings.ContainsRune(view, '\t') {
		t.Errorf("raw tab reached the screen:\n%q", view)
	}
	// "start" ends at column 5, so the tab pads to the stop at 8: three spaces.
	if !strings.Contains(view, "start   indented") {
		t.Errorf("tab not expanded to the 4-column stop:\n%q", view)
	}
}

// streamView drives the deltas as AssistantTextDeltaMsg for one block through
// Update and returns View().
func streamView(t *testing.T, deltas ...string) string {
	t.Helper()
	m := sizedModel(t, 80, 24, false)
	for _, d := range deltas {
		m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: d})
	}
	return m.View()
}

// TestStreamedEscapeSplitAcrossDeltasNeverReachesScreen: a CSI and an OSC
// sequence each split across two deltas render exactly as the whole text does.
func TestStreamedEscapeSplitAcrossDeltasNeverReachesScreen(t *testing.T) {
	cases := [][]string{
		{"before \x1b[3", "1mred\x1b[0m after"},
		{"x \x1b]0;ti", "tle\x07 done"},
	}
	for _, c := range cases {
		view := streamView(t, c...)
		if strings.ContainsRune(view, 0x1b) || strings.ContainsRune(view, 0x07) {
			t.Errorf("escape reached the screen for %q:\n%q", c, view)
		}
		if want := streamView(t, strings.Join(c, "")); view != want {
			t.Errorf("split %q differs from whole:\n got %q\nwant %q", c, view, want)
		}
	}
	// Positive control: the text around the sequence is shown.
	if view := streamView(t, "before \x1b[3", "1mred\x1b[0m after"); !strings.Contains(view, "before red after") {
		t.Errorf("surrounding text missing:\n%q", view)
	}
}

// TestStreamedNewlineInLaterDeltaStartsNewRow: a \n in a later delta produces a
// new rendered row.
func TestStreamedNewlineInLaterDeltaStartsNewRow(t *testing.T) {
	rows := rowsContaining(t, streamView(t, "line one", "\nline two"), "line one", "line two")
	if len(rows) != 2 {
		t.Fatalf("want line one and line two on separate rows, got %q", rows)
	}
	if strings.Contains(rows[0], "line two") || strings.Contains(rows[1], "line one") {
		t.Errorf("lines share a row: %q", rows)
	}
}

// TestStreamedControlInLaterDeltaRendersAsDot: a C0 control in a later delta
// renders as the spec's middle dot.
func TestStreamedControlInLaterDeltaRendersAsDot(t *testing.T) {
	view := streamView(t, "ab", "c\x01d")
	if !strings.Contains(view, "abc·d") {
		t.Errorf("control not rendered as ·:\n%q", view)
	}
	if strings.ContainsRune(view, 0x01) {
		t.Errorf("raw control reached the screen:\n%q", view)
	}
}

// TestStreamedCRLFSplitAcrossDeltas: "\r" ends one delta and "\n" starts the
// next; the result is two rows, as CRLF in one delta gives.
func TestStreamedCRLFSplitAcrossDeltas(t *testing.T) {
	view := streamView(t, "alpha\r", "\nbeta")
	rows := rowsContaining(t, view, "alpha", "beta")
	if len(rows) != 2 {
		t.Fatalf("want alpha and beta on separate rows, got %q", rows)
	}
	if want := streamView(t, "alpha\r\nbeta"); view != want {
		t.Errorf("split CRLF differs from whole:\n got %q\nwant %q", view, want)
	}
}

// streamMixed is an input mixing every case Sanitize handles: CSI, OSC, CRLF,
// a bare \r overwrite, a tab, a control, wide runes, and a newline.
const streamMixed = "head \x1b[1;31mred\x1b[0m\r\nprogress 10%\rprogress 100%\n" +
	"tab\there \x1b]0;t\x07ctl\x02end 日本語\nlast"

// TestStreamedEqualsWholeAtEverySplit drives AssistantTextDeltaMsg through
// Update. For every byte offset strictly inside a mixed input — the loop
// counts bytes, so offsets inside each 3-byte rune of 日本語 are included — two
// deltas split there render identically to one delta carrying the whole text.
// So do a stream of one byte per delta and a stream of one rune per delta.
// Control: the input really has offsets that are not rune starts, so the byte
// loop covers splits a range over the string would skip.
func TestStreamedEqualsWholeAtEverySplit(t *testing.T) {
	want := streamView(t, streamMixed)
	if strings.ContainsAny(want, "\x1b\x07\x02\t") {
		t.Fatalf("whole-text view carries a raw control:\n%q", want)
	}
	midRune := 0
	for i := 1; i < len(streamMixed); i++ {
		if !utf8.RuneStart(streamMixed[i]) {
			midRune++
		}
		if got := streamView(t, streamMixed[:i], streamMixed[i:]); got != want {
			t.Errorf("split at byte %d (%q|%q) differs:\n got %q\nwant %q", i, streamMixed[:i], streamMixed[i:], got, want)
		}
	}
	if midRune == 0 {
		t.Error("control: no mid-rune offset in streamMixed, so no split inside a rune was tested")
	}
	perByte := make([]string, len(streamMixed))
	for i := range len(streamMixed) {
		perByte[i] = streamMixed[i : i+1]
	}
	if got := streamView(t, perByte...); got != want {
		t.Errorf("one-byte deltas differ:\n got %q\nwant %q", got, want)
	}
	var runes []string
	for _, r := range streamMixed {
		runes = append(runes, string(r))
	}
	if got := streamView(t, runes...); got != want {
		t.Errorf("one-rune deltas differ:\n got %q\nwant %q", got, want)
	}
}

// thinkingCard returns the model's only thinking card.
func thinkingCard(t *testing.T, m Model) *ThinkingCard {
	t.Helper()
	for i := range m.tr.Items() {
		if it := &m.tr.Items()[i]; it.Kind == KindThinking {
			return it.Thinking
		}
	}
	t.Fatal("no thinking card in the transcript")
	return nil
}

// TestStreamedThinkingIsSanitisedAcrossDeltas drives ThinkingDeltaMsg through
// Update. renderThinking draws only the token count, so the thinking text is
// never on screen; this checks BodyLines instead. A View-absence check was
// dropped: View never draws thinking text, so it could not fail. A sequence
// split across deltas is removed, and BodyLines equals the sanitised whole for
// a two-delta split at every byte offset inside the input (the loop counts
// bytes, not runes).
func TestStreamedThinkingIsSanitisedAcrossDeltas(t *testing.T) {
	whole := "plan \x1b[3" + "1mhid\x1b[0m\r\nden\x02 step"
	wantBody := SanitizeLines(whole)
	for i := 1; i < len(whole); i++ {
		m := sizedModel(t, 80, 24, false)
		m = drive(m,
			ThinkingDeltaMsg{BlockID: 1, Delta: whole[:i]},
			ThinkingDeltaMsg{BlockID: 1, Delta: whole[i:]})
		card := thinkingCard(t, m)
		if got := card.BodyLines(); !slices.Equal(got, wantBody) {
			t.Errorf("split at %d: BodyLines %q, want %q", i, got, wantBody)
		}
	}
	// Positive control: the token count still follows the deltas' words.
	m := sizedModel(t, 80, 24, false)
	m = drive(m,
		ThinkingDeltaMsg{BlockID: 1, Delta: "one two"},
		ThinkingDeltaMsg{BlockID: 1, Delta: " three"})
	if got := thinkingCard(t, m).Tokens; got != 3 {
		t.Errorf("Tokens = %d, want 3", got)
	}
}

// streamingBlock appends a streaming assistant block built directly from lines
// (Raw empty) to a bare Transcript and returns the transcript and its ID.
func streamingBlock(lines []string) (*Transcript, ItemID) {
	tr := &Transcript{}
	id := tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{Lines: lines, Streaming: true}})
	return tr, id
}

// TestAppendTextSeedsFromLinesWhenRawEmpty calls Transcript.AppendText on a
// streaming block built with Lines and no Raw: the existing lines survive and
// the delta lands after them, including a delta that opens a new line.
func TestAppendTextSeedsFromLinesWhenRawEmpty(t *testing.T) {
	tr, id := streamingBlock([]string{"one", "two"})
	if !tr.AppendText(id, "!") {
		t.Fatal("AppendText refused a streaming block")
	}
	if got, want := tr.Items()[0].Text.Lines, []string{"one", "two!"}; !slices.Equal(got, want) {
		t.Errorf("after first delta: Lines %q, want %q", got, want)
	}
	tr.AppendText(id, "\nthree")
	if got, want := tr.Items()[0].Text.Lines, []string{"one", "two!", "three"}; !slices.Equal(got, want) {
		t.Errorf("after second delta: Lines %q, want %q", got, want)
	}
}

// TestAppendThinkingSeedsFromBodyWhenRawEmpty calls Transcript.AppendThinking
// on a card built with Body and no Raw (the resume shape): the existing text
// survives in BodyLines with the delta after it, and Body is cleared so Raw is
// the single source.
func TestAppendThinkingSeedsFromBodyWhenRawEmpty(t *testing.T) {
	tr := &Transcript{}
	id := tr.Append(Item{Kind: KindThinking, Thinking: &ThinkingCard{Body: []string{"one", "two"}}})
	if !tr.AppendThinking(id, " x") {
		t.Fatal("AppendThinking refused the card")
	}
	card := tr.Items()[0].Thinking
	if got, want := card.BodyLines(), []string{"one", "two x"}; !slices.Equal(got, want) {
		t.Errorf("BodyLines %q, want %q", got, want)
	}
	if card.Body != nil {
		t.Errorf("Body %q not cleared after seeding Raw", card.Body)
	}
}

// TestThinkingBodyLines calls ThinkingCard.BodyLines directly: a streamed card
// (Raw set) returns sanitised lines, a card built from Body returns Body.
func TestThinkingBodyLines(t *testing.T) {
	streamed := &ThinkingCard{Raw: "a\x1b[31mb\x02c\r\nd"}
	if got, want := streamed.BodyLines(), []string{"ab·c", "d"}; !slices.Equal(got, want) {
		t.Errorf("streamed: BodyLines %q, want %q", got, want)
	}
	built := &ThinkingCard{Body: []string{"x", "y"}}
	if got, want := built.BodyLines(), []string{"x", "y"}; !slices.Equal(got, want) {
		t.Errorf("built: BodyLines %q, want %q", got, want)
	}
}

// TestStreamEndReleasesRaw sends each stream-ending message through Update
// (TurnCompleteMsg, TurnErrorMsg, TurnCancelledMsg) after streamed deltas: the
// assistant block's Raw is empty afterwards while its Lines are unchanged, and
// the text is still on screen. Control: Raw is non-empty before the message.
func TestStreamEndReleasesRaw(t *testing.T) {
	msgs := map[string]any{
		"complete":  TurnCompleteMsg{},
		"error":     TurnErrorMsg{Kind: "x", Message: "m"},
		"cancelled": TurnCancelledMsg{},
	}
	for name, msg := range msgs {
		t.Run(name, func(t *testing.T) {
			m := sizedModel(t, 80, 24, false)
			m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "alpha \x1b[3"},
				AssistantTextDeltaMsg{BlockID: 1, Delta: "1mbeta"})
			tb := m.tr.Items()[0].Text
			if tb.Raw == "" {
				t.Fatal("control: Raw empty while streaming")
			}
			before := slices.Clone(tb.Lines)
			m = drive(m, msg.(tea.Msg))
			tb = m.tr.Items()[0].Text
			if tb.Streaming {
				t.Error("block still streaming")
			}
			if tb.Raw != "" {
				t.Errorf("Raw %q not released", tb.Raw)
			}
			if !slices.Equal(tb.Lines, before) {
				t.Errorf("Lines changed: %q, was %q", tb.Lines, before)
			}
			if v := m.View(); !strings.Contains(v, "alpha beta") {
				t.Errorf("text missing from view:\n%s", v)
			}
		})
	}
}

// TestAppendTextRefusesFinishedBlock streams two deltas through Update, ends
// the turn with TurnCompleteMsg through Update, then calls
// Transcript.AppendText directly on the finished block (Raw released,
// Streaming false). It must return false and leave Lines and Raw unchanged.
// Control: the same direct call on the block before TurnCompleteMsg returns
// true and changes Lines, so the refusal is the Streaming check and not a
// missing block.
func TestAppendTextRefusesFinishedBlock(t *testing.T) {
	m := sizedModel(t, 80, 24, false)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "alpha "},
		AssistantTextDeltaMsg{BlockID: 1, Delta: "beta"})
	id := m.tr.Items()[0].ID
	if !m.tr.AppendText(id, " gamma") {
		t.Fatal("control: AppendText refused the block while streaming")
	}
	if got, want := m.tr.Items()[0].Text.Lines, []string{"alpha beta gamma"}; !slices.Equal(got, want) {
		t.Fatalf("control: Lines %q, want %q", got, want)
	}

	m = drive(m, TurnCompleteMsg{})
	tb := m.tr.Items()[0].Text
	if tb.Streaming || tb.Raw != "" {
		t.Fatalf("precondition: Streaming %v, Raw %q after TurnCompleteMsg", tb.Streaming, tb.Raw)
	}
	before := slices.Clone(tb.Lines)
	if m.tr.AppendText(id, " late") {
		t.Error("AppendText accepted a delta on a finished block")
	}
	tb = m.tr.Items()[0].Text
	if !slices.Equal(tb.Lines, before) {
		t.Errorf("Lines changed on a finished block: %q, was %q", tb.Lines, before)
	}
	if tb.Raw != "" {
		t.Errorf("Raw re-seeded on a finished block: %q", tb.Raw)
	}
}

// benchText returns n bytes of assistant-style text: several lines, a few
// ANSI colour runs, tabs.
func benchText(n int) string {
	const unit = "func main() {\n\tfmt.Println(\"\x1b[32mok\x1b[0m\")\t// tab\n    return\n}\n"
	return strings.Repeat(unit, n/len(unit)+1)[:n]
}

// benchDeltas splits benchText(64 KB) into 4-byte deltas, about 16,000 of
// them.
func benchDeltas() []string {
	text := benchText(64 * 1024)
	var out []string
	for i := 0; i < len(text); i += 4 {
		out = append(out, text[i:min(i+4, len(text))])
	}
	return out
}

// benchDeltasSmall splits benchText(8 KB) into deltas whose sizes cycle
// deterministically through 16 to 32 bytes, nearer a real streamed reply than
// benchDeltas' 4-byte slivers.
func benchDeltasSmall() []string {
	text := benchText(8 * 1024)
	var out []string
	for i, k := 0, 0; i < len(text); k++ {
		size := 16 + (k*7)%17 // 16..32
		out = append(out, text[i:min(i+size, len(text))])
		i += size
	}
	return out
}

// BenchmarkUpdateAssistantTextDeltas8KB streams benchDeltasSmall through
// Model.Update(AssistantTextDeltaMsg), the whole per-delta cost for a
// typically sized reply.
func BenchmarkUpdateAssistantTextDeltas8KB(b *testing.B) {
	deltas := benchDeltasSmall()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		m := New(Options{Version: fixtureVersion, Caps: Caps{Colour: false, Unicode: true}})
		next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m = next.(Model)
		for _, d := range deltas {
			next, _ := m.Update(AssistantTextDeltaMsg{BlockID: 1, Delta: d})
			m = next.(Model)
		}
	}
}

// BenchmarkUpdateAssistantTextDeltas streams the deltas through
// Model.Update(AssistantTextDeltaMsg), the whole per-delta cost.
func BenchmarkUpdateAssistantTextDeltas(b *testing.B) {
	deltas := benchDeltas()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		m := New(Options{Version: fixtureVersion, Caps: Caps{Colour: false, Unicode: true}})
		next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m = next.(Model)
		for _, d := range deltas {
			next, _ := m.Update(AssistantTextDeltaMsg{BlockID: 1, Delta: d})
			m = next.(Model)
		}
	}
}

// BenchmarkTranscriptAppendText streams the same deltas through
// Transcript.AppendText only: the sanitising cost without Update or layout.
func BenchmarkTranscriptAppendText(b *testing.B) {
	deltas := benchDeltas()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tr, id := streamingBlock([]string{""})
		for _, d := range deltas {
			tr.AppendText(id, d)
		}
	}
}
