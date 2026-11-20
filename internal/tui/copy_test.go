package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/clipboard"
)

// copyRecorder stands in for Model.Copy. It records every payload it is handed
// and reports a successful local copy, so a test can tell "Copy was called with
// this text" from "no copy was started".
type copyRecorder struct{ got []string }

func (r *copyRecorder) copy(text string) clipboard.Result {
	r.got = append(r.got, text)
	return clipboard.Result{Chars: utf8.RuneCountInString(text), Local: true}
}

// newCopyModel is an 80x24 empty-transcript model with a recording Copy wired.
func newCopyModel(t *testing.T) (Model, *copyRecorder) {
	t.Helper()
	rec := &copyRecorder{}
	m := newTestModel()
	m.Copy = rec.copy
	return m, rec
}

// withReply streams one assistant reply through Update and ends the turn, the
// live path that sets Source.
func withReply(m Model, text string) Model {
	return drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: text}, TurnCompleteMsg{})
}

// slashCopy submits "/copy"+args through the composer's Enter key, the real
// entry point. It returns the model and the Cmd Update returned, without running it.
func slashCopy(t *testing.T, m Model, args string) (Model, tea.Cmd) {
	t.Helper()
	line := "/copy"
	if args != "" {
		line += " " + args
	}
	return submitSlash(t, m, line)
}

// runCopyCmd runs the Cmd a copy returned and feeds its message back through
// Update, as the Bubble Tea runtime would.
func runCopyCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("no tea.Cmd returned; the copy result has no way back into Update")
	}
	msg := cmd()
	if _, ok := msg.(CopyResultMsg); !ok {
		t.Fatalf("copy command produced %T, want CopyResultMsg", msg)
	}
	return drive(m, msg)
}

func copiedNotice(text string) string {
	return "copied " + itoa(len([]rune(text))) + " characters"
}

// TestCopyArgumentsAndPrecedence drives every accepted and refused argument
// form through submit, on a reply with two code blocks. Notice-only rows must
// start no copy (nil Cmd, recorder untouched); copying rows must start one and
// deliver the right text. The two kinds sit in one table so "nothing was
// copied" is always judged beside a row where something was.
func TestCopyArgumentsAndPrecedence(t *testing.T) {
	reply := "intro\n```go\nA1\n```\nmid\n~~~\nB2\n~~~\n"
	const usage = "usage: /copy [code [N]]"
	const rng = "the last reply has 2 code blocks"
	cases := []struct {
		name       string
		args       string
		wantCopied string // empty: notice-only
		wantHint   string // notice-only rows
	}{
		{"whole reply", "", reply, ""},
		{"code is the last block", "code", "B2", ""},
		{"code 1", "code 1", "A1", ""},
		{"code 2", "code 2", "B2", ""},
		{"code 02, leading zero", "code 02", "B2", ""},
		{"digits without code", "1", "", usage},
		{"code with 25 digits, value 1", "code 0000000000000000000000001", "A1", ""},
		{"code 0", "code 0", "", rng},
		{"code 3 above the count", "code 3", "", rng},
		{"code 25-digit N too large to parse", "code 9999999999999999999999999", "", rng},
		{"code +1 is not digits only", "code +1", "", usage},
		{"CODE is lower case only", "CODE", "", usage},
		{"Code", "Code 1", "", usage},
		{"code 1 2 has too many tokens", "code 1 2", "", usage},
		{"foo", "foo", "", usage},
		{"code x", "code x", "", usage},
		{"code -1", "code -1", "", usage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, rec := newCopyModel(t)
			m = withReply(m, reply)
			m, cmd := slashCopy(t, m, c.args)
			if c.wantCopied == "" {
				if cmd != nil {
					t.Fatalf("/copy %s returned a Cmd; a notice-only outcome must start no copy", c.args)
				}
				if m.comp.Hint != c.wantHint {
					t.Errorf("hint = %q, want %q", m.comp.Hint, c.wantHint)
				}
				if len(rec.got) != 0 {
					t.Errorf("Copy was called with %q on a notice-only outcome", rec.got)
				}
				return
			}
			if m.comp.Hint != "" {
				t.Errorf("hint before the result = %q, want none: the notice belongs to the result", m.comp.Hint)
			}
			m = runCopyCmd(t, m, cmd)
			if len(rec.got) != 1 || rec.got[0] != c.wantCopied {
				t.Errorf("Copy was called with %q, want exactly [%q]", rec.got, c.wantCopied)
			}
			if want := copiedNotice(c.wantCopied); m.comp.Hint != want {
				t.Errorf("hint = %q, want %q", m.comp.Hint, want)
			}
		})
	}
}

// TestCopyWithNoReply: with no assistant reply a bad argument is still a usage
// error (precedence 1 before 2), and a good one is "nothing to copy".
func TestCopyWithNoReply(t *testing.T) {
	cases := []struct{ args, want string }{
		{"foo", "usage: /copy [code [N]]"},
		{"", "nothing to copy"},
		{"code", "nothing to copy"},
		{"code 2", "nothing to copy"},
	}
	for _, c := range cases {
		m, rec := newCopyModel(t)
		m, cmd := slashCopy(t, m, c.args)
		if cmd != nil || len(rec.got) != 0 {
			t.Errorf("/copy %s with no reply started a copy", c.args)
		}
		if m.comp.Hint != c.want {
			t.Errorf("/copy %s with no reply: hint = %q, want %q", c.args, m.comp.Hint, c.want)
		}
	}
}

// TestCopyCodeOnReplyWithoutBlocks: "no code block" wins over the range notice
// (precedence 3 before 4), whatever N is. The control is the same reply copied
// whole, which starts a copy.
func TestCopyCodeOnReplyWithoutBlocks(t *testing.T) {
	for _, args := range []string{"code", "code 1", "code 3", "code 0"} {
		m, _ := newCopyModel(t)
		m = withReply(m, "just prose\n")
		m, cmd := slashCopy(t, m, args)
		if cmd != nil || m.comp.Hint != "no code block in the last reply" {
			t.Errorf("/copy %s: hint = %q, cmd nil = %v; want the no-code-block notice and no copy",
				args, m.comp.Hint, cmd == nil)
		}
	}
	m, rec := newCopyModel(t)
	m = withReply(m, "just prose\n")
	m, cmd := slashCopy(t, m, "")
	_ = runCopyCmd(t, m, cmd)
	if len(rec.got) != 1 || rec.got[0] != "just prose\n" {
		t.Errorf("control: whole-reply copy handed %q to Copy", rec.got)
	}
}

// TestCopyRangeNoticeSingularAndPlural: M counts every block, and 1 is singular.
func TestCopyRangeNoticeSingularAndPlural(t *testing.T) {
	cases := []struct {
		reply, args, want string
	}{
		{"```\na\n```\n", "code 2", "the last reply has 1 code block"},
		{"```\na\n```\n", "code 0", "the last reply has 1 code block"},
		{"```\na\n```\n```\n```\n", "code 3", "the last reply has 2 code blocks"}, // the second is empty and still counts
		{"```\na\n```\n```\nb\n```\n```\nc\n```\n", "code 4", "the last reply has 3 code blocks"},
	}
	for _, c := range cases {
		m, _ := newCopyModel(t)
		m = withReply(m, c.reply)
		m, _ = slashCopy(t, m, c.args)
		if m.comp.Hint != c.want {
			t.Errorf("reply %q /copy %s: hint = %q, want %q", c.reply, c.args, m.comp.Hint, c.want)
		}
	}
}

// TestCopyEmptyBlockNotice: bare `code` names the last block's number, `code K`
// names K, and a block of spaces is content, not empty.
func TestCopyEmptyBlockNotice(t *testing.T) {
	cases := []struct {
		name, reply, args string
		wantHint          string // empty: a copy of wantCopied starts
		wantCopied        string
	}{
		{"bare code, last of one block is empty", "```\n```\n", "code", "code block 1 is empty", ""},
		{"bare code, last of three blocks is empty", "```\nx\n```\n```\ny\n```\n```\n```\n", "code", "code block 3 is empty", ""},
		{"code 2 of three", "```\nx\n```\n```\n\n```\n```\ny\n```\n", "code 2", "code block 2 is empty", ""},
		{"code 1 and 3 of the same reply copy", "```\nx\n```\n```\n\n```\n```\ny\n```\n", "code 3", "", "y"},
		{"a block of spaces is not empty", "```\n  \n```\n", "code", "", "  "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, rec := newCopyModel(t)
			m = withReply(m, c.reply)
			m, cmd := slashCopy(t, m, c.args)
			if c.wantHint != "" {
				if cmd != nil || len(rec.got) != 0 {
					t.Errorf("an empty block started a copy")
				}
				if m.comp.Hint != c.wantHint {
					t.Errorf("hint = %q, want %q", m.comp.Hint, c.wantHint)
				}
				return
			}
			_ = runCopyCmd(t, m, cmd)
			if len(rec.got) != 1 || rec.got[0] != c.wantCopied {
				t.Errorf("Copy was called with %q, want [%q]", rec.got, c.wantCopied)
			}
		})
	}
}

// TestCopyReplyThatCleansToNothing: a reply whose source cleans to the empty
// string gives "nothing to copy" and starts no copy. The control is a reply with
// one visible character beside the same escape sequence, which copies.
func TestCopyReplyThatCleansToNothing(t *testing.T) {
	m, rec := newCopyModel(t)
	m = withReply(m, "\x1b[31m\x00\u202e")
	m, cmd := slashCopy(t, m, "")
	if cmd != nil || len(rec.got) != 0 || m.comp.Hint != "nothing to copy" {
		t.Errorf("reply that cleans to nothing: hint = %q, cmd nil = %v, copies = %q",
			m.comp.Hint, cmd == nil, rec.got)
	}

	m, rec = newCopyModel(t)
	m = withReply(m, "\x1b[31mx\x00\u202e")
	m, cmd = slashCopy(t, m, "")
	m = runCopyCmd(t, m, cmd)
	if len(rec.got) != 1 || rec.got[0] != "x" {
		t.Errorf("control: Copy was called with %q, want [\"x\"]: the payload is the cleaned source", rec.got)
	}
}

// TestCopyWithoutRouteWired: with Model.Copy nil the notice is the failure
// notice, through the same message path.
func TestCopyWithoutRouteWired(t *testing.T) {
	m := newTestModel()
	m = withReply(m, "héllo")
	m, cmd := slashCopy(t, m, "")
	m = runCopyCmd(t, m, cmd)
	if m.comp.Hint != "copy failed: no clipboard route" {
		t.Errorf("hint = %q, want the failure notice", m.comp.Hint)
	}
}

// TestCopyLastReplyIsTheLastAssistantItem: an older reply and a user message
// after it do not change which reply is copied.
func TestCopyLastReplyIsTheLastAssistantItem(t *testing.T) {
	m, rec := newCopyModel(t)
	m = withReply(m, "first reply")
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"a question"}}})
	m = drive(m, AssistantTextDeltaMsg{BlockID: 2, Delta: "second reply"}, TurnCompleteMsg{})
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"another question"}}})
	m, cmd := slashCopy(t, m, "")
	_ = runCopyCmd(t, m, cmd)
	if len(rec.got) != 1 || rec.got[0] != "second reply" {
		t.Errorf("Copy was called with %q, want [\"second reply\"]", rec.got)
	}
}

// lastAssistantText returns the text block of the newest assistant item (the
// newest item overall can be an error card).
func lastAssistantText(t *testing.T, m Model) *TextBlock {
	t.Helper()
	items := m.tr.Items()
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].Kind == KindAssistant {
			return items[i].Text
		}
	}
	t.Fatal("no assistant item in the transcript")
	return nil
}

// TestStreamEndKeepsSource ends a stream by each of its three routes and checks
// that Source holds the unsanitised text and Raw is empty. Entry points: Update
// with TurnCompleteMsg, TurnErrorMsg and TurnCancelledMsg, and the Esc key,
// which reaches cancelTurn by a different road. The control is the model before
// the stream ends: Source is empty and Raw holds the text.
func TestStreamEndKeepsSource(t *testing.T) {
	const raw = "he\x1b[31mllo\x00 wor"
	const rest = "ld"
	ends := []struct {
		name string
		end  func(Model) Model
	}{
		{"TurnCompleteMsg", func(m Model) Model { return drive(m, TurnCompleteMsg{}) }},
		{"TurnErrorMsg", func(m Model) Model { return drive(m, TurnErrorMsg{Kind: "k", Message: "msg"}) }},
		{"TurnCancelledMsg", func(m Model) Model { return drive(m, TurnCancelledMsg{}) }},
		{"Esc key while busy", func(m Model) Model { return drive(m, keyType(tea.KeyEsc)) }},
	}
	for _, e := range ends {
		t.Run(e.name, func(t *testing.T) {
			m := newTestModel()
			m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: raw}, AssistantTextDeltaMsg{BlockID: 1, Delta: rest})
			tb := lastAssistantText(t, m)
			if tb.Raw != raw+rest || tb.Source != "" || !tb.Streaming {
				t.Fatalf("control: while streaming Raw = %q, Source = %q, Streaming = %v", tb.Raw, tb.Source, tb.Streaming)
			}
			m = e.end(m)
			tb = lastAssistantText(t, m)
			if tb.Streaming {
				t.Fatalf("setup: %s did not end the stream", e.name)
			}
			if tb.Source != raw+rest {
				t.Errorf("Source = %q, want the unsanitised text %q", tb.Source, raw+rest)
			}
			if tb.Raw != "" {
				t.Errorf("Raw = %q after the stream ended, want it emptied", tb.Raw)
			}
			if tb.SourceText() != raw+rest {
				t.Errorf("SourceText = %q, want %q", tb.SourceText(), raw+rest)
			}
			for _, l := range tb.Lines {
				if strings.ContainsAny(l, "\x1b\x00") {
					t.Errorf("Lines %q still hold control characters; Source must differ from Lines", tb.Lines)
				}
			}
		})
	}
}

// TestSourceTextFallbacks pins the order SourceText reads: Source, then Raw,
// then Lines joined. The Lines row is the fallback for blocks that tests build
// from Lines alone.
func TestSourceTextFallbacks(t *testing.T) {
	cases := []struct {
		name string
		tb   TextBlock
		want string
	}{
		{"Source wins", TextBlock{Source: "src", Raw: "raw", Lines: []string{"l"}}, "src"},
		{"Raw while streaming", TextBlock{Raw: "raw", Lines: []string{"l"}}, "raw"},
		{"Lines joined", TextBlock{Lines: []string{"a", "b"}}, "a\nb"},
		{"nothing", TextBlock{}, ""},
	}
	for _, c := range cases {
		if got := c.tb.SourceText(); got != c.want {
			t.Errorf("%s: SourceText = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestCopyRefusedWhileBusy shows the keyboard path: while a turn runs, Enter is
// ignored, so /copy cannot be submitted. (Typing is ignored too; the composer
// is input-disabled.) The control is the same keys after the turn ends.
func TestCopyRefusedWhileBusy(t *testing.T) {
	m, rec := newCopyModel(t)
	m = withReply(m, "done")
	m = drive(m, AssistantTextDeltaMsg{BlockID: 2, Delta: "streaming"})
	if !m.busy.Active {
		t.Fatal("setup: the delta did not make the model busy")
	}
	m, cmd := slashCopy(t, m, "")
	if cmd != nil || len(rec.got) != 0 || m.comp.Hint != "" || m.copySeq != 0 {
		t.Errorf("/copy while busy: cmd nil = %v, copies = %q, hint = %q, copySeq = %d; want it refused",
			cmd == nil, rec.got, m.comp.Hint, m.copySeq)
	}

	m = drive(m, TurnCompleteMsg{})
	m, cmd = slashCopy(t, m, "")
	_ = runCopyCmd(t, m, cmd)
	if len(rec.got) != 1 {
		t.Errorf("control: /copy after the turn ended made %d copies, want 1", len(rec.got))
	}
}

// TestCopyOfAStreamingReplyReadsRaw calls copyCommand and startCopy directly,
// not through the keyboard, because the keyboard path is refused while busy (see
// TestCopyRefusedWhileBusy). A block that is still streaming has Raw and no
// Source, and the copy gets Raw, cleaned. The control is the same block after
// the stream ends, which reads Source and gets the same text.
func TestCopyOfAStreamingReplyReadsRaw(t *testing.T) {
	m, rec := newCopyModel(t)
	m = drive(m, AssistantTextDeltaMsg{BlockID: 1, Delta: "so far\x1b[0m"})
	tb := m.tr.Items()[m.tr.Len()-1].Text
	if !tb.Streaming || tb.Source != "" || tb.Raw == "" {
		t.Fatalf("setup: Streaming = %v, Source = %q, Raw = %q", tb.Streaming, tb.Source, tb.Raw)
	}

	next, cmd := m.copyCommand("")
	m = next.(Model)
	m = runCopyCmd(t, m, cmd)
	if len(rec.got) != 1 || rec.got[0] != "so far" {
		t.Errorf("copy of a streaming reply handed %q to Copy, want [\"so far\"]", rec.got)
	}

	m = drive(m, TurnCompleteMsg{})
	if m.tr.Items()[m.tr.Len()-1].Text.Raw != "" {
		t.Fatal("setup: Raw survived the end of the stream")
	}
	cmd = m.startCopy(m.tr.Items()[m.tr.Len()-1].Text.SourceText())
	_ = runCopyCmd(t, m, cmd)
	if len(rec.got) != 2 || rec.got[1] != "so far" {
		t.Errorf("copy after the stream ended handed %q to Copy, want the same text from Source", rec.got)
	}
}

// TestOlderCopyResultIsIgnored: the latest copy wins. Two copies are started
// and their results delivered out of order; the earlier request's result must
// not show. The control is the later request's result, which does.
func TestOlderCopyResultIsIgnored(t *testing.T) {
	m, _ := newCopyModel(t)
	m = withReply(m, "aaaa") // 4 characters
	m, cmd1 := slashCopy(t, m, "")
	m = drive(m, AssistantTextDeltaMsg{BlockID: 2, Delta: "bbbbbbbb"}, TurnCompleteMsg{}) // 8 characters
	m, cmd2 := slashCopy(t, m, "")
	msg1, msg2 := cmd1(), cmd2()
	if m.copySeq != 2 {
		t.Fatalf("setup: copySeq = %d, want 2", m.copySeq)
	}

	// The older result arrives while the newer is still in flight.
	stale := drive(m, msg1)
	if stale.comp.Hint != "" {
		t.Errorf("older result set the hint to %q while a newer copy was in flight", stale.comp.Hint)
	}
	// The newer result arrives, then the older one.
	m = drive(m, msg2)
	if m.comp.Hint != "copied 8 characters" {
		t.Fatalf("control: newer result hint = %q, want %q", m.comp.Hint, "copied 8 characters")
	}
	m = drive(m, msg1)
	if m.comp.Hint != "copied 8 characters" {
		t.Errorf("older result replaced the newer notice with %q", m.comp.Hint)
	}
}

// TestCopyResultDoesNotOverwriteOtherHints: a copy result replaces only an empty
// hint or a copy notice. The unknown-command hint, a usage hint and the
// approval-release hint stay. Each is paired with the same result arriving on an
// empty hint, which shows its notice.
func TestCopyResultDoesNotOverwriteOtherHints(t *testing.T) {
	t.Run("unknown-command hint", func(t *testing.T) {
		m, _ := newCopyModel(t)
		m = withReply(m, "hello")
		m, cmd := slashCopy(t, m, "")
		msg := cmd()
		m = typeCmd(m, "/nosuchthing")
		if m.comp.Hint != "unknown command /nosuchthing" {
			t.Fatalf("setup: hint = %q", m.comp.Hint)
		}
		m = drive(m, msg)
		if m.comp.Hint != "unknown command /nosuchthing" {
			t.Errorf("copy result overwrote the unknown-command hint with %q", m.comp.Hint)
		}
	})
	t.Run("usage hint", func(t *testing.T) {
		m, _ := newCopyModel(t)
		m = withReply(m, "hello")
		m, cmd := slashCopy(t, m, "")
		msg := cmd()
		m, _ = slashCopy(t, m, "foo")
		if m.comp.Hint != "usage: /copy [code [N]]" || m.copySeq != 1 {
			t.Fatalf("setup: hint = %q, copySeq = %d; the usage hint must not bump copySeq, "+
				"or the Seq guard would explain the result below", m.comp.Hint, m.copySeq)
		}
		m = drive(m, msg)
		if m.comp.Hint != "usage: /copy [code [N]]" {
			t.Errorf("copy result overwrote the usage hint with %q", m.comp.Hint)
		}
	})
	t.Run("approval-release hint", func(t *testing.T) {
		m := newDriven(t) // a fixture approval is pending; keys other than navigation release it
		rec := &copyRecorder{}
		m.Copy = rec.copy
		cmd := m.startCopy("hello")
		msg := cmd()
		m = drive(m, key('x'))
		if m.comp.Hint != approvalReleaseHint {
			t.Fatalf("setup: hint = %q, want the approval-release hint", m.comp.Hint)
		}
		m = drive(m, msg)
		if m.comp.Hint != approvalReleaseHint {
			t.Errorf("copy result overwrote the approval-release hint with %q", m.comp.Hint)
		}
	})
	t.Run("control: the same result on an empty hint", func(t *testing.T) {
		m, _ := newCopyModel(t)
		m = withReply(m, "hello")
		m, cmd := slashCopy(t, m, "")
		m = runCopyCmd(t, m, cmd)
		if m.comp.Hint != "copied 5 characters" {
			t.Errorf("hint = %q, want the copy notice", m.comp.Hint)
		}
	})
}

// TestStaleCopyFlagDoesNotLetAResultOverwriteAHint: the copy-notice flag must go
// with the hint it described. A copy notice is shown, then a new /copy resets
// the composer (clearing the notice), then /nosuchthing sets a non-copy hint
// while that copy is still in flight. The flag left over from the first notice
// would let the result overwrite the unknown-command hint.
func TestStaleCopyFlagDoesNotLetAResultOverwriteAHint(t *testing.T) {
	m, _ := newCopyModel(t)
	m = withReply(m, "hello")
	m, cmd := slashCopy(t, m, "")
	m = runCopyCmd(t, m, cmd)
	if !m.comp.hintIsCopy {
		t.Fatal("setup: a shown copy notice is not flagged as one")
	}
	m, cmd = slashCopy(t, m, "")
	msg := cmd()
	m = typeCmd(m, "/nosuchthing")
	m = drive(m, msg)
	if m.comp.Hint != "unknown command /nosuchthing" {
		t.Errorf("hint = %q: a flag left from an earlier copy notice let the result overwrite it", m.comp.Hint)
	}
}

// TestCopyNoticeClearing: the notice clears on keys that reach the text area
// (typing and cursor movement), on Esc, and when an approval card arrives;
// Ctrl+G, PgUp and PgDn leave it. The loop's last step is the positive control
// for the absence assertions before it: the notice was present until a typed key.
func TestCopyNoticeClearing(t *testing.T) {
	show := func(t *testing.T) Model {
		t.Helper()
		m, _ := newCopyModel(t)
		m = withReply(m, "hello")
		m, cmd := slashCopy(t, m, "")
		m = runCopyCmd(t, m, cmd)
		if m.comp.Hint != "copied 5 characters" || !m.comp.hintIsCopy {
			t.Fatalf("setup: hint = %q, hintIsCopy = %v", m.comp.Hint, m.comp.hintIsCopy)
		}
		return m
	}
	leaves := []struct {
		name string
		k    tea.KeyMsg
	}{
		{"Ctrl+G", keyType(tea.KeyCtrlG)},
		{"PgUp", keyType(tea.KeyPgUp)},
		{"PgDn", keyType(tea.KeyPgDown)},
	}
	for _, c := range leaves {
		t.Run(c.name+" leaves it", func(t *testing.T) {
			m := drive(show(t), c.k)
			if m.comp.Hint != "copied 5 characters" || !m.comp.hintIsCopy {
				t.Errorf("%s changed the hint to %q (copy flag %v)", c.name, m.comp.Hint, m.comp.hintIsCopy)
			}
		})
	}
	clears := []struct {
		name string
		step func(Model) Model
	}{
		{"a typed key", func(m Model) Model { return drive(m, key('a')) }},
		{"cursor movement (Left)", func(m Model) Model { return drive(m, keyType(tea.KeyLeft)) }},
		{"Esc", func(m Model) Model { return drive(m, keyType(tea.KeyEsc)) }},
		{"an approval card arriving", func(m Model) Model {
			return drive(m, ApprovalRequestedMsg{ID: 1, Kind: "command", Description: "run x", Subject: "x"})
		}},
	}
	for _, c := range clears {
		t.Run(c.name+" clears it", func(t *testing.T) {
			m := c.step(show(t))
			if m.comp.Hint != "" || m.comp.hintIsCopy {
				t.Errorf("after %s: hint = %q, hintIsCopy = %v; want both cleared", c.name, m.comp.Hint, m.comp.hintIsCopy)
			}
		})
	}
}

// TestCopyDoesNotMoveTheViewport: /copy, whether it copies or only shows a
// notice, and the result arriving afterwards, leave the offset, the pin and the
// new-content count alone. The control is /status, which re-pins in the same
// setup, so an unmoved viewport is not a viewport that cannot move.
func TestCopyDoesNotMoveTheViewport(t *testing.T) {
	setup := func(t *testing.T) Model {
		t.Helper()
		rec := &copyRecorder{}
		m := composingUnpinned(t)
		m.Copy = rec.copy
		m = withReply(m, "prose\n```\ncode\n```\n")
		if m.scroll.Pinned {
			t.Fatal("setup: the reply re-pinned the transcript")
		}
		return m
	}
	for _, args := range []string{"", "code", "code 9", "foo"} {
		t.Run("/copy "+args, func(t *testing.T) {
			m := setup(t)
			offset, newSince := m.scroll.Offset, m.scroll.NewSince
			m, cmd := slashCopy(t, m, args)
			if cmd != nil {
				m = runCopyCmd(t, m, cmd)
			}
			if m.scroll.Pinned || m.scroll.Offset != offset || m.scroll.NewSince != newSince {
				t.Errorf("/copy %s moved the viewport: Pinned %v, Offset %d (was %d), NewSince %d (was %d)",
					args, m.scroll.Pinned, m.scroll.Offset, offset, m.scroll.NewSince, newSince)
			}
		})
	}
	m := setup(t)
	m = typeCmd(m, "/status")
	if !m.scroll.Pinned {
		t.Error("control: /status did not re-pin in this setup")
	}
}

// TestSlashCompletionWithCopy: /c and /co are ambiguous between /compact and
// /copy and complete nothing; /cop completes to /copy. Entry point: the Tab key
// in the composer. The completeSlash rows pin the helper itself.
func TestSlashCompletionWithCopy(t *testing.T) {
	for _, c := range []struct{ typed, want string }{
		{"/c", "/c"},
		{"/co", "/co"},
		{"/cop", "/copy"},
		{"/comp", "/compact"},
	} {
		m := newTestModel()
		for _, r := range c.typed {
			m = drive(m, key(r))
		}
		m = drive(m, keyType(tea.KeyTab))
		if got := m.comp.Value(); got != c.want {
			t.Errorf("Tab after %q: composer = %q, want %q", c.typed, got, c.want)
		}
	}
	if _, unique := completeSlash("c"); unique {
		t.Error("completeSlash(\"c\") was unique")
	}
	if got, unique := completeSlash("cop"); !unique || got != "copy" {
		t.Errorf("completeSlash(\"cop\") = %q, %v; want copy, true", got, unique)
	}
	if last := SlashCommands[len(SlashCommands)-1]; last != "copy" {
		t.Errorf("last slash command = %q, want copy", last)
	}
}

// TestCopyNoticeOnlyInvocationSupersedesACopyInFlight: every /copy invocation
// takes a sequence number, notice-only outcomes included, so a copy still in
// flight cannot overwrite a newer notice. Entry point: /copy through the
// composer's Enter key, twice. The control delivers the same first result with
// no second invocation, and it lands.
func TestCopyNoticeOnlyInvocationSupersedesACopyInFlight(t *testing.T) {
	m, _ := newCopyModel(t)
	m = withReply(m, "aaaa") // 4 characters, no code block
	m, cmd1 := slashCopy(t, m, "")
	msg1 := cmd1()

	// Control: nothing newer was typed, so the first result shows.
	if got := drive(m, msg1); got.comp.Hint != "copied 4 characters" {
		t.Fatalf("control: hint = %q, want %q", got.comp.Hint, "copied 4 characters")
	}

	// A notice-only /copy while the first copy is in flight.
	m, cmd2 := slashCopy(t, m, "code 9")
	if cmd2 != nil {
		t.Fatal("setup: /copy code 9 started a copy")
	}
	const notice = "no code block in the last reply"
	if m.comp.Hint != notice {
		t.Fatalf("setup: hint = %q, want %q", m.comp.Hint, notice)
	}
	m = drive(m, msg1)
	if m.comp.Hint != notice {
		t.Errorf("the older copy's result replaced the newer notice with %q", m.comp.Hint)
	}
}

// TestCopyHintAddsComposerRow: a notice-only /copy makes the computed layout's
// ComposerH one row taller. Entry point: /copy through the composer's
// Enter key, so copyCommand sets the hint with no relayout of its own. It checks
// the layout computed from the model after the key, not the rows relayout
// stores, which depend on the width and not on the composer. Each notice-only
// outcome must leave a layout one row taller than before; the control is /copy
// on a reply, which shows no hint until its result arrives and so leaves the
// layout unchanged.
func TestCopyHintAddsComposerRow(t *testing.T) {
	for _, args := range []string{"foo", "code", "code 9"} {
		t.Run("/copy "+args, func(t *testing.T) {
			m, _ := newCopyModel(t)
			m = withReply(m, "plain reply, no code block")
			before := m.layout()
			m, _ = slashCopy(t, m, args)
			after := m.layout()
			if m.comp.Hint == "" {
				t.Fatal("setup: no hint was set")
			}
			if after.ComposerH != before.ComposerH+1 {
				t.Errorf("ComposerH = %d, want %d (before %d plus the hint row)", after.ComposerH, before.ComposerH+1, before.ComposerH)
			}
		})
	}
	m, _ := newCopyModel(t)
	m = withReply(m, "plain reply, no code block")
	before := m.layout()
	m, _ = slashCopy(t, m, "")
	if after := m.layout(); after != before {
		t.Errorf("control: /copy on a reply changed the layout: %+v -> %+v", before, after)
	}
}
