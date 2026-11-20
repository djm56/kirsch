package tui

import (
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/clipboard"
)

// Copy notices that never go through the clipboard routes. ui-spec §6.
const (
	copyUsageHint     = "usage: /copy [code [N]]"
	copyNothingNotice = "nothing to copy"
	copyNoCodeNotice  = "no code block in the last reply"
	copyCardsNotice   = "c copies replies only"
)

// copyRequest is a parsed /copy argument list.
type copyRequest struct {
	code     bool // the code forms: `code` and `code N`
	hasN     bool // `code N`: n holds the block number, unless tooLarge
	n        int
	tooLarge bool // N is all digits but does not fit an int
}

// parseCopyArgs parses the text after /copy. ui-spec §6, "Arguments".
//
// Accepted: nothing; `code`; `code` and one token of ASCII digits (leading
// zeros allowed, no sign). `code` is lower case only. Anything else returns
// ok == false and the caller shows the usage hint. A digit string too long for
// an int is accepted with tooLarge set, so it reaches the range notice rather
// than the usage hint.
func parseCopyArgs(args string) (req copyRequest, ok bool) {
	f := strings.Fields(args)
	switch {
	case len(f) == 0:
		return req, true
	case f[0] != "code" || len(f) > 2:
		return req, false
	}
	req.code = true
	if len(f) == 1 {
		return req, true
	}
	for i := 0; i < len(f[1]); i++ {
		if f[1][i] < '0' || f[1][i] > '9' {
			return copyRequest{}, false
		}
	}
	req.hasN = true
	n, err := strconv.Atoi(f[1])
	if err != nil {
		req.tooLarge = true
		return req, true
	}
	req.n = n
	return req, true
}

// setHint sets the composer's dim hint. isCopy records that the hint is a copy
// notice, which is what lets a later CopyResultMsg replace it; every hint that
// is not one passes false, and an empty hint is never a copy notice.
func (m *Model) setHint(text string, isCopy bool) {
	m.comp.Hint = text
	m.comp.hintIsCopy = isCopy && text != ""
}

// copyHintOnly shows a copy notice that needed no clipboard route. It increments
// copySeq, so a copy still in flight cannot overwrite this notice when its
// result arrives. It never re-pins or scrolls: the notice is the whole answer,
// as for an unknown command. The caller is on the key path, where dispatchKey
// rebuilds the layout the hint changes. ui-spec §2.4, §6.
func (m *Model) copyHintOnly(text string) {
	m.copySeq++
	m.setHint(text, true)
}

// lastReplySource returns the source text of the last assistant reply: the last
// assistant item whose text block is not empty. ok is false when there is none.
func (m *Model) lastReplySource() (string, bool) {
	items := m.tr.Items()
	for i := len(items) - 1; i >= 0; i-- {
		it := items[i]
		if it.Kind != KindAssistant || it.Text == nil {
			continue
		}
		if src := it.Text.SourceText(); src != "" {
			return src, true
		}
	}
	return "", false
}

// startCopy begins a copy of text, which is a reply's source text or one code
// block of it. /copy and `c` call it. It cleans the text, so a caller cannot forget
// to. Cleaning a block that was found in already-cleaned text
// changes nothing: Clean is idempotent (TestCleanIsIdempotent).
//
// A payload that cleans to nothing shows "nothing to copy" and starts no copy.
// Otherwise the returned Cmd runs Model.Copy off the update loop and delivers a
// CopyResultMsg carrying the number copySeq was incremented to. With no Copy
// wired the Cmd reports a result in which no route worked, which shows "copy
// failed: no clipboard route".
func (m *Model) startCopy(text string) tea.Cmd {
	payload := clipboard.Clean(text)
	if payload == "" {
		m.copyHintOnly(copyNothingNotice)
		return nil
	}
	m.copySeq++
	seq := m.copySeq
	send := m.Copy
	return func() tea.Msg {
		if send == nil {
			return CopyResultMsg{Seq: seq, Result: clipboard.Result{Chars: utf8.RuneCountInString(payload)}}
		}
		return CopyResultMsg{Seq: seq, Result: send(payload)}
	}
}

// copySelected runs `c` in Browsing. With nothing selected the notice is
// "nothing to copy"; on a selected card it is "c copies replies only"; on a
// selected reply it copies the reply's source text, not its wrapped lines, and
// startCopy cleans it. Every outcome that is only a notice goes through
// copyHintOnly, so it supersedes a copy in flight as /copy does. It never
// re-pins or scrolls. ui-spec §6, "Precedence".
func (m Model) copySelected() (tea.Model, tea.Cmd) {
	it, _, ok := m.tr.Find(m.sel)
	switch {
	case !ok:
		m.copyHintOnly(copyNothingNotice)
		return m, nil
	case it.Kind == KindAssistant && it.Text != nil:
		cmd := m.startCopy(it.Text.SourceText())
		return m, cmd
	default:
		m.copyHintOnly(copyCardsNotice)
		return m, nil
	}
}

// copyCommand runs /copy. It applies the §6 precedence in order: usage, no
// reply, no code block, range, empty block; then startCopy refuses a payload
// that cleans to nothing. Every outcome is a hint or a background copy, so it
// never re-pins or scrolls the transcript. ui-spec §6, "Precedence".
func (m Model) copyCommand(args string) (tea.Model, tea.Cmd) {
	req, ok := parseCopyArgs(args)
	if !ok {
		m.setHint(copyUsageHint, false)
		return m, nil
	}
	src, ok := m.lastReplySource()
	if !ok {
		m.copyHintOnly(copyNothingNotice)
		return m, nil
	}
	if !req.code {
		cmd := m.startCopy(src)
		return m, cmd
	}

	blocks := clipboard.CodeBlocks(clipboard.Clean(src))
	if len(blocks) == 0 {
		m.copyHintOnly(copyNoCodeNotice)
		return m, nil
	}
	k := len(blocks) // bare `code` is the last block
	if req.hasN {
		if req.tooLarge || req.n == 0 || req.n > len(blocks) {
			m.copyHintOnly(copyRangeNotice(len(blocks)))
			return m, nil
		}
		k = req.n
	}
	block := blocks[k-1]
	if block == "" {
		m.copyHintOnly("code block " + strconv.Itoa(k) + " is empty")
		return m, nil
	}
	cmd := m.startCopy(block)
	return m, cmd
}

// copyRangeNotice is the notice for a block number that does not exist. count is
// the block count, empty blocks included.
func copyRangeNotice(count int) string {
	if count == 1 {
		return "the last reply has 1 code block"
	}
	return "the last reply has " + strconv.Itoa(count) + " code blocks"
}

// applyCopyResult shows a finished copy's notice. A result older than the latest
// request is ignored, so the latest copy wins. The notice replaces the hint only
// when the hint is empty or is itself a copy notice: an unknown-command hint,
// the approval-release hint or a usage hint is left showing. ui-spec §6.
func (m *Model) applyCopyResult(msg CopyResultMsg) {
	if msg.Seq != m.copySeq {
		return
	}
	if m.comp.Hint != "" && !m.comp.hintIsCopy {
		return
	}
	m.setHint(clipboard.Notice(msg.Result), true)
	m.relayout(m.layout())
}
