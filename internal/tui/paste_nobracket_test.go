package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// unbracketedMsgs turns text into the messages Bubble Tea v1.3.10 delivers
// when the terminal does NOT bracket a paste, so the bytes arrive as ordinary
// typing. Read from detectOneMsg and readAnsiInputs in key.go (v1.3.10) and
// from key_sequences.go:
//
//   - a run of printable, non-space runes is one KeyRunes message with Paste
//     false (detectOneMsg, the loop that collects runes);
//   - a run ends at the end of a read that returned fewer bytes than the
//     256-byte buffer, because readAnsiInputs treats a short read as an event
//     boundary (canHaveMoreData is false). The OS decides where short reads
//     fall, so a paste may be cut into runs of any length; chunk models that;
//   - a run that reaches the end of a FULL 256-byte read is not cut there:
//     detectOneMsg returns 0, nil and readAnsiInputs carries the bytes into
//     the next read, so the run continues unbroken. A full buffer therefore
//     adds no boundary of its own, except where a multi-byte rune straddles
//     it (below), and no chunk size here is chosen to match it;
//   - a space is never part of a run: it is its own KeySpace message carrying
//     Runes [' '] (key_sequences.go, the space entry; detectOneMsg stops a run
//     at ' ');
//   - a newline is a control byte, so it is its own message: \r is KeyEnter
//     and \n is KeyCtrlJ (key.go key constants).
//
// Not modelled: a multi-byte rune that straddles a read boundary. There
// detectOneMsg's utf8.DecodeRune sees a truncated tail, gets RuneError and
// ends the run; the orphaned lead byte then comes out as unknownInputByteMsg
// and its continuation byte in the next read is dropped the same way, so the
// rune is lost in real delivery. This helper emits only whole runes.
//
// chunk caps the runes per KeyRunes message; 1 is one rune per message. It
// counts runes, whereas real cuts fall at byte positions in a read.
func unbracketedMsgs(text string, chunk int) []tea.Msg {
	var msgs []tea.Msg
	var run []rune
	flush := func() {
		for len(run) > 0 {
			n := min(chunk, len(run))
			msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: append([]rune(nil), run[:n]...)})
			run = run[n:]
		}
	}
	for _, r := range text {
		switch r {
		case ' ':
			flush()
			msgs = append(msgs, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
		case '\r':
			flush()
			msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter})
		case '\n':
			flush()
			msgs = append(msgs, tea.KeyMsg{Type: tea.KeyCtrlJ})
		default:
			run = append(run, r)
		}
	}
	flush()
	return msgs
}

// fiveKB returns exactly 5120 bytes of space-separated words, including a
// two-byte rune, with no newline.
func fiveKB() string {
	var b strings.Builder
	for i := 0; b.Len() < 5<<10; i++ {
		w := "word"
		if i%11 == 0 {
			w = "café"
		}
		if b.Len()+len(w)+1 > 5<<10 {
			w = strings.Repeat("x", 5<<10-b.Len())
		}
		b.WriteString(w)
		if b.Len() < 5<<10 {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// TestUnbracketed5KBPasteLandsWholeThroughKeyMessages proves that with bracketed
// paste off, once a 5 KB single-line paste has been turned into whole-rune key
// messages it lands whole in the composer: nothing is lost or reordered at any
// chunk size, nothing is submitted, and no large-paste confirm is raised (that
// prompt belongs to the Paste path only, and 5 KB is under PasteWarnBytes
// anyway). This says nothing about real unbracketed delivery of non-ASCII
// text: the fixture contains a multi-byte rune, and a rune split across two
// reads is lost before it reaches Update (see unbracketedMsgs), which no
// debounce in the model could recover. A positive control proves the same
// chunks, fed with a message dropped, would be caught by the comparison.
//
// Entry point: this drives Model.Update (via drive) with hand-built tea.KeyMsg
// values from unbracketedMsgs, which model Bubble Tea's delivery. Bubble Tea's
// input reader (readAnsiInputs / detectOneMsg) is NOT exercised, and a
// multi-byte rune straddling a read boundary is outside the model (see
// unbracketedMsgs). Chunk sizes, in runes per message (real cuts fall at byte
// positions): 1 (every rune its own message, the most messages), 3 (cuts each
// four-rune word unevenly, so runs of mixed length reach the composer), and
// 1<<20 (no cutting: one message per word, the fewest). No run in this fixture
// exceeds 5 runes (the words are 4, the final filler at most 5), so any finite
// size of 5 or more would add nothing over 1<<20. For the same reason a long
// unspaced run, i.e. one large KeyRunes message, is not exercised here.
func TestUnbracketed5KBPasteLandsWholeThroughKeyMessages(t *testing.T) {
	text := fiveKB()
	if len(text) != 5<<10 || !utf8.ValidString(text) || strings.ContainsAny(text, "\r\n") {
		t.Fatalf("setup: text is %d bytes, valid=%v", len(text), utf8.ValidString(text))
	}
	start := composing(t)
	base := start.tr.Len() // the fixture opens with items of its own
	for _, chunk := range []int{1, 3, 1 << 20} {
		m := drive(composing(t), unbracketedMsgs(text, chunk)...)
		if got := m.comp.Value(); got != text {
			t.Errorf("chunk %d: composer holds %d bytes, want the %d pasted; first difference at byte %d",
				chunk, len(got), len(text), firstDiff(got, text))
		}
		if m.busy.Active || m.tr.Len() != base {
			t.Errorf("chunk %d: the paste submitted (busy=%v, transcript items=%d)", chunk, m.busy.Active, m.tr.Len())
		}
		if m.mode() != ModeComposing {
			t.Errorf("chunk %d: mode = %v, want Composing (no large-paste confirm)", chunk, m.mode())
		}
	}

	// Positive control: the same messages minus the final one must NOT match,
	// so the equality above can fail.
	msgs := unbracketedMsgs(text, 3)
	short := drive(composing(t), msgs[:len(msgs)-1]...)
	if short.comp.Value() == text {
		t.Error("control: a composer fed one message fewer still equals the full text")
	}
}

// firstDiff returns the byte index where a and b first differ, or the shorter
// length when one is a prefix of the other.
func firstDiff(a, b string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// TestUnbracketedMultiLinePasteCurrentBehaviour DESCRIBES today's behaviour; it
// is not a statement of intent. With bracketed paste off, a pasted newline
// arrives as KeyEnter (terminals send \r), which keyComposing treats as submit
// (update.go:748-761 -> submit, :1404). So the first line is sent as a message
// and the turn goes busy; every later key of the paste is then DROPPED, because
// the composer is input-disabled while a turn is live (update.go:853-855) and a
// busy Enter returns early (:758).
// A \n byte would instead arrive as Ctrl+J, which inserts a newline (:763).
func TestUnbracketedMultiLinePasteCurrentBehaviour(t *testing.T) {
	start := composing(t)
	base := start.tr.Len() // the fixture opens with items of its own
	m := drive(composing(t), unbracketedMsgs("first line\rsecond line\rthird", 255)...)
	if m.tr.Len() != base+1 {
		t.Fatalf("transcript items = %d, want %d (only the first line submitted)", m.tr.Len(), base+1)
	}
	if it := m.tr.Items()[base]; it.Kind != KindUser || strings.Join(it.Text.Lines, "\n") != "first line" {
		t.Errorf("submitted item = %+v, want the user message %q", it, "first line")
	}
	if !m.busy.Active {
		t.Error("busy = false, want true: the submit started a turn")
	}
	// The rest of the paste is lost: the composer is input-disabled while busy.
	if got := m.comp.Value(); got != "" {
		t.Errorf("composer = %q, want empty (later lines dropped while busy)", got)
	}

	// Control: with \n (Ctrl+J) as the separator nothing is submitted.
	n := drive(composing(t), unbracketedMsgs("first line\nsecond line", 255)...)
	if n.tr.Len() != base || n.comp.Value() != "first line\nsecond line" {
		t.Errorf("control: Ctrl+J separators gave transcript=%d composer=%q", n.tr.Len(), n.comp.Value())
	}
}
