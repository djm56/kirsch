package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Chain replay. The v0.1.5 walkthrough in plan/testing/manual/ux-round-v0.1.md runs
// its sections 1 to 10 one after another in one session, so what a section starts
// from is what the section before it left behind. The single-state captures cannot
// show that: each starts from an empty model. chainReplay drives sections 1 to 10 in
// order through one Model and writes down, before and after each section, what the
// walkthrough's cues depend on: which item is selected and whether any of its rows
// lie in the visible transcript window (the gutter is drawn only then), the key-hint
// row, the mode, the scroll offset and pin, the open modal, the size and the glyph
// set. The text goes to captures/v0.1.5/chain-replay.txt, a captureExtras file with
// the same write and compare modes as the captures.
//
// Every input is a message the app would send or a key the walkthrough names:
// Update takes the user's Enter, the messages a live read_file call sends
// (ToolStartedMsg with no target, ToolTargetMsg and ToolCompletedMsg: capReadFileID),
// AssistantTextDeltaMsg chunks and TurnCompleteMsg for each reply,
// and the /diff command's own DiffResultMsg (the command Update returns is run).
// Where a step says "press ↑ or ↓ until the gutter shows on X", arrowUntil presses
// the key in a loop with a bound and records the direction and the count; its goal is
// a selected item with a row in view, not selection alone.
//
// Section 4 is run in each ordering of the same inputs that chainOrder names, and the
// replay records each. In the main path, which the chain goes on from, the keys are
// pressed after the README.md tool card finishes and before the reply's first text,
// as the walkthrough says: "as soon as the first tool card shows". Each retry branch
// runs on a separate model that repeats sections 1 to 3, and its record follows
// section 4's. In retry branch 1 the reply has started to stream when the keys are
// pressed; in retry branch 2 the reply has finished, which is the walkthrough's
// `reply finished first` case. The header of the file says which is which.
//
// What is canned: every file name and text (README.md, long.txt, the replies), the
// diff (the shared inputs file's diff, one file, so the long.txt hunk is not below a
// hello.go hunk), and the resumed transcript in section 10, which is the live model's own
// items handed to a new model in a TranscriptLoadedMsg. The messages the replay does
// not send are named in the file's header.

// chainReadme is the README.md the replay's first read_file call returns.
const chainReadme = "# Kirsch\n\nA terminal coding assistant.\n\n## Install\n\nRun npm run build.\n"

// chainRun is a replay in progress: the model, the text written so far and the
// section being recorded.
type chainRun struct {
	t       *testing.T
	m       Model
	unicode bool
	b       strings.Builder
}

// shorten cuts s to at most n runes.
func shorten(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// identity names the item with ID id by kind and a short text, and its place in
// the transcript, or "none" for ID 0 and a note when no item has that ID.
func (c *chainRun) identity(id ItemID) string {
	if id == 0 {
		return "none"
	}
	it, i, ok := c.m.tr.Find(id)
	if !ok {
		return fmt.Sprintf("no item with ID %d", id)
	}
	at := fmt.Sprintf(" (item %d of %d)", i+1, c.m.tr.Len())
	switch it.Kind {
	case KindTool:
		return fmt.Sprintf("tool %s %s%s", it.Tool.Name, it.Tool.Target, at)
	case KindAssistant:
		return fmt.Sprintf("reply %q, %d lines%s", shorten(it.Text.Lines[0], 24), len(it.Text.Lines), at)
	case KindUser:
		return fmt.Sprintf("user %q%s", shorten(it.Text.Lines[0], 24), at)
	case KindNotice:
		return fmt.Sprintf("notice %q%s", shorten(it.Notice.Text, 24), at)
	default:
		return fmt.Sprintf("item kind %d%s", it.Kind, at)
	}
}

// selectedInView reports whether the selected item has any row inside the visible
// transcript window, which is the rows from the scroll offset for TranscriptH rows. The
// gutter is drawn on every row of the selected item, so a selection with no row in the
// window shows no gutter. It returns the selected item's first and last row, the
// window's first and last row, and ok false when there is no selection or the model
// holds no rows for it.
func selectedInView(m Model) (inView bool, first, last, top, bot int, ok bool) {
	if m.sel == 0 {
		return false, 0, 0, 0, 0, false
	}
	r, found := m.rowFor(m.sel)
	if !found {
		return false, 0, 0, 0, 0, false
	}
	top = m.scroll.Offset
	bot = top + m.layout().TranscriptH - 1
	first, last = r.Start, r.Start+r.N-1
	return first <= bot && last >= top, first, last, top, bot, true
}

// gutterLine is the record of what the gutter depends on.
func (c *chainRun) gutterLine() string {
	inView, first, last, top, bot, ok := selectedInView(c.m)
	if !ok {
		return "no item selected, so no gutter"
	}
	line := fmt.Sprintf("selected item's rows %d-%d, transcript window rows %d-%d, a row in view: %v", first, last, top, bot, inView)
	if c.m.modal != nil {
		line += " (a modal is open: its box covers part of the window, and these rows are the transcript's)"
	}
	return line
}

// hintLine is the text of the key-hint row, the last row of the frame, or a note when
// the height shows none.
func (c *chainRun) hintLine() string {
	if c.m.layout().HintH != 1 {
		return "(none at this height)"
	}
	rows := strings.Split(c.m.View(), "\n")
	return fmt.Sprintf("%q", strings.TrimSpace(rows[len(rows)-1]))
}

// state writes the model's recorded fields under label.
func (c *chainRun) state(label string) {
	modal := "none"
	if md := c.m.modal; md != nil {
		kind := map[ModalKind]string{ModalContent: "content", ModalDiff: "diff", ModalHelp: "help"}[md.Kind]
		modal = fmt.Sprintf("%s %q, offset %d", kind, md.Title, md.Off)
	}
	glyphs := "ascii"
	if c.unicode {
		glyphs = "unicode"
	}
	fmt.Fprintf(&c.b, "%s:\n  selection: %s\n  gutter: %s\n  hint row: %s\n  mode: %s\n  scroll: offset %d, pinned %v, new %d\n  modal: %s\n  size: %dx%d, glyphs %s\n",
		label, c.identity(c.m.sel), c.gutterLine(), c.hintLine(), c.m.mode(),
		c.m.scroll.Offset, c.m.scroll.Pinned, c.m.scroll.NewSince,
		modal, c.m.width, c.m.height, glyphs)
}

// begin starts a section and records the state it starts from.
func (c *chainRun) begin(title string) {
	fmt.Fprintf(&c.b, "\n=== section %s ===\n", title)
	c.state("before")
}

// end records the state a section ends in.
func (c *chainRun) end() { c.state("after") }

// step writes one input line.
func (c *chainRun) step(format string, args ...any) {
	fmt.Fprintf(&c.b, "step: "+format+"\n", args...)
}

// keys presses the named keys in order, by capKeyMsg, through Update.
func (c *chainRun) keys(names ...string) {
	c.t.Helper()
	for _, n := range names {
		msg, err := capKeyMsg(n)
		if err != nil {
			c.t.Fatal(err)
		}
		c.step("key %s", n)
		c.m = drive(c.m, msg)
	}
}

// text types text into the composer, one key per character, through Update.
func (c *chainRun) text(s string) {
	c.step("type %q", s)
	for _, r := range s {
		c.m = drive(c.m, key(r))
	}
}

// deliver sends canned messages, as the app would, under one described step.
// The Elapsed stamp has no observable effect here, because no chain-replay check
// reads a tool card head's duration. It is there so the replay delivers the
// messages the state builder delivers, and nothing pins it.
func (c *chainRun) deliver(what string, msgs ...tea.Msg) {
	c.step("%s", what)
	c.m = drive(c.m, capStampElapsed(c.t, msgs)...)
}

// expectMode fails the replay when the model is not in want. The walkthrough's
// steps name the mode a key leads to, so a key that does not lead there stops the
// replay instead of recording a state the walkthrough never reaches.
func (c *chainRun) expectMode(want Mode, after string) {
	c.t.Helper()
	if got := c.m.mode(); got != want {
		c.t.Fatalf("after %s the mode is %s, the walkthrough expects %s", after, got, want)
	}
}

// onTool reports whether the selected item is a tool card for target and has a row in
// the visible transcript window. That is when the gutter shows on the card: the
// selection alone does not draw it once the card is scrolled out of the window.
func onTool(target string) func(Model) bool {
	return func(m Model) bool {
		it, _, ok := m.tr.Find(m.sel)
		if !ok || it.Kind != KindTool || it.Tool.Target != target {
			return false
		}
		inView, _, _, _, _, _ := selectedInView(m)
		return inView
	}
}

// pressArrows presses k up to bound times on m in Browsing until pred holds. A press
// that leaves Browsing is followed by Shift+↑, which the walkthrough names as the
// way back; recovers counts those. It returns the model, the arrow presses made and
// whether pred held.
func pressArrows(m Model, k tea.KeyType, bound int, pred func(Model) bool) (out Model, presses, recovers int, ok bool) {
	for presses < bound {
		if pred(m) {
			return m, presses, recovers, true
		}
		m = drive(m, keyType(k))
		presses++
		if m.mode() != ModeBrowsing {
			m = drive(m, keyType(tea.KeyShiftUp))
			recovers++
		}
	}
	return m, presses, recovers, pred(m)
}

// arrowUntil does what "press ↑ or ↓ until the gutter shows on <what>" says: with the
// gutter already there (pred: the card selected and a row of it in view) it presses
// nothing, otherwise it tries ↑ and then ↓, each at most chainArrowBound presses, and
// keeps the first that gets there. The ↓ try starts again from the state the model was
// in before any ↑ press, and the record says so. It fails the replay when neither gets
// there, so a cue the walkthrough names that the model cannot reach is never recorded
// as reached.
func (c *chainRun) arrowUntil(what string, pred func(Model) bool) {
	c.t.Helper()
	if pred(c.m) {
		c.step("arrows until the gutter is on %s: the card is already selected with a row in view, 0 presses", what)
		return
	}
	for i, d := range []struct {
		name string
		k    tea.KeyType
	}{{"↑", tea.KeyUp}, {"↓", tea.KeyDown}} {
		from := ""
		if i > 0 {
			from = fmt.Sprintf(", starting again from the state before the %s presses, not from where they ended", "↑")
		}
		if m, n, rec, ok := pressArrows(c.m, d.k, chainArrowBound, pred); ok {
			c.step("arrows until the gutter is on %s: %s pressed %d times, %d Shift+↑ to return to Browsing%s", what, d.name, n, rec, from)
			c.m = m
			return
		}
		c.step("arrows until the gutter is on %s: %s for %d presses did not get there%s", what, d.name, chainArrowBound, from)
	}
	c.t.Fatalf("neither ↑ nor ↓ reaches %s within %d presses", what, chainArrowBound)
}

// chainArrowBound is the most arrow presses arrowUntil makes in one direction.
const chainArrowBound = 40

// chainArrowTimeLimit is how long arrowUntilWithin waits for arrowUntil to return. The
// longest honest call makes 2*chainArrowBound presses of a key, which takes
// milliseconds; the limit is far above that.
const chainArrowTimeLimit = 5 * time.Second

// arrowUntilWithin is c.arrowUntil on its own goroutine with a time limit. A press
// loop with no bound never returns when the card cannot be reached, and a test that
// called arrowUntil directly would then hang until the test binary's own timeout.
// Here it fails after chainArrowTimeLimit, with a message that names the cause. The
// goroutine of a call that timed out is left running and ends with the test binary.
// An arrowUntil that fails the replay (t.Fatalf) ends its goroutine, and done is
// closed either way.
func arrowUntilWithin(t *testing.T, c *chainRun, what string, pred func(Model) bool) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.arrowUntil(what, pred)
	}()
	select {
	case <-done:
	case <-time.After(chainArrowTimeLimit):
		t.Fatalf("arrowUntil(%q) did not return within %v: the press loop has no bound", what, chainArrowTimeLimit)
	}
	if t.Failed() {
		t.FailNow()
	}
}

// chainReplyLines is the numbered reply to "count from 1 to 300".
func chainReplyLines(from, to int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		fmt.Fprintf(&b, "%d\n", i)
	}
	return b.String()
}

// chainLongTxt is long.txt as the walkthrough's setup writes it: a 150-character
// line, the lines seq 1 80 prints, a second 150-character line.
func chainLongTxt() string {
	var b strings.Builder
	b.WriteString(strings.Repeat("x", 150) + "\n")
	for i := 1; i <= 80; i++ {
		fmt.Fprintf(&b, "%d\n", i)
	}
	b.WriteString(strings.Repeat("y", 150) + "\n")
	return b.String()
}

// chainResumeMsgs is what the replay sends for a resume: the notice and then the
// transcript, which open the messages App.ResumeLastSession sends.
func chainResumeMsgs(items []Item) []tea.Msg {
	return []tea.Msg{NoticeMsg{Text: "session resumed"}, TranscriptLoadedMsg{Items: items}}
}

// msgTypes is the %T of each message, in order.
func msgTypes(msgs []tea.Msg) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = fmt.Sprintf("%T", m)
	}
	return out
}

// minus is the elements of all that are not in some.
func minus(all, some []string) []string {
	have := map[string]bool{}
	for _, s := range some {
		have[s] = true
	}
	var out []string
	for _, s := range all {
		if !have[s] {
			out = append(out, s)
		}
	}
	return out
}

// newChain is a replay with a model at 80x24, Unicode, colour off and the canned
// /diff wired in.
func newChain(t *testing.T) *chainRun {
	t.Helper()
	c := &chainRun{t: t, unicode: true}
	c.m = New(capOptions(t, false, true))
	c.m.GetDiff = func() DiffResult { return capDiff(t) }
	c.m = drive(c.m, tea.WindowSizeMsg{Width: 80, Height: 24})
	return c
}

// chainSections1to3 runs sections 1 to 3, which every ordering of section 4 starts from.
func chainSections1to3(c *chainRun) {
	// Section 1: a short transcript.
	c.begin("1 · a short transcript has no bar")
	c.text("read README.md and reply in one short sentence")
	c.keys("Enter")
	c.deliver("(the read_file call for README.md finishes)", capReadFileID(1, "README.md", chainReadme)...)
	c.deliver("(the reply streams, the turn completes)",
		AssistantTextDeltaMsg{BlockID: 1, Delta: "Kirsch is a terminal coding assistant."}, TurnCompleteMsg{})
	c.end()

	// Section 2: the overflowing transcript.
	c.begin("2 · the bar at the bottom of an overflowing transcript")
	c.text("count from 1 to 300, one number per line")
	c.keys("Enter")
	c.deliver("(the reply streams, the turn completes)",
		AssistantTextDeltaMsg{BlockID: 2, Delta: chainReplyLines(1, 150)},
		AssistantTextDeltaMsg{BlockID: 2, Delta: chainReplyLines(151, 300)}, TurnCompleteMsg{})
	c.end()

	// Section 3: Browsing at the top.
	c.begin("3 · the bar at the top")
	c.keys("Shift+↑")
	c.expectMode(ModeBrowsing, "Shift+↑")
	c.keys("g")
	c.end()
}

// chainOrder is when section 4's keys (Shift+↑, g, PgDn) are pressed relative to the
// reply.
type chainOrder int

const (
	// orderKeysFirst presses the keys after the README.md tool card finishes and
	// before the reply's first text delta.
	orderKeysFirst chainOrder = iota
	// orderReplyStartedFirst delivers the reply's first 100 lines, then presses the
	// keys, then delivers the rest of the reply and the end of the turn.
	orderReplyStartedFirst
	// orderReplyFinishedFirst delivers the whole reply and the end of the turn, then
	// presses the keys: the walkthrough's `reply finished first`.
	orderReplyFinishedFirst
)

// chainSection4 runs section 4 under title, with the keys pressed at the point order names.
func chainSection4(c *chainRun, title string, order chainOrder) {
	c.begin(title)
	c.keys("Tab")
	c.expectMode(ModeComposing, "Tab")
	c.text("read README.md, then count from 1 to 300, one number per line")
	c.keys("Enter")
	c.deliver("(the read_file call for README.md finishes)", capReadFileID(2, "README.md", chainReadme)...)
	pressKeys := func() {
		c.keys("Shift+↑")
		c.expectMode(ModeBrowsing, "Shift+↑")
		c.keys("g", "PgDn")
	}
	switch order {
	case orderKeysFirst:
		pressKeys()
		c.deliver("(the reply streams, the turn completes)",
			AssistantTextDeltaMsg{BlockID: 3, Delta: chainReplyLines(1, 300)}, TurnCompleteMsg{})
	case orderReplyStartedFirst:
		c.deliver("(the reply starts to stream)", AssistantTextDeltaMsg{BlockID: 3, Delta: chainReplyLines(1, 100)})
		pressKeys()
		c.deliver("(the rest of the reply streams, the turn completes)",
			AssistantTextDeltaMsg{BlockID: 3, Delta: chainReplyLines(101, 300)}, TurnCompleteMsg{})
	case orderReplyFinishedFirst:
		c.deliver("(the reply streams, the turn completes)",
			AssistantTextDeltaMsg{BlockID: 3, Delta: chainReplyLines(1, 300)}, TurnCompleteMsg{})
		pressKeys()
	}
	c.end()
}

// chainReplay is the text of chain-replay.txt. Entry point: Model.Update, through
// drive for keys and canned messages and called directly for the commands whose
// result is part of the chain: /diff's DiffResultMsg and /quit's tea.Quit.
func chainReplay(t *testing.T) string {
	t.Helper()
	c := newChain(t)
	realResume := capIn(t).ResumeMessages
	sentResume := msgTypes(chainResumeMsgs(nil))

	fmt.Fprintf(&c.b, "# chain replay: sections 1 to 10 of the v0.1.5 walkthrough, in order, in one Model\n")
	fmt.Fprintf(&c.b, "# release: v0.1.5\n")
	fmt.Fprintf(&c.b, "# no model, no network, no key: canned messages and real key messages through the real Model\n")
	fmt.Fprintf(&c.b, "# canned: file names and texts, the replies, a one-file /diff, and the resumed transcript of section 10 (the live model's own items)\n")
	fmt.Fprintf(&c.b, "# section 4 is run in each of these orderings of the same inputs. The main path, which the chain goes on from, presses Shift+↑, g and PgDn after the README.md tool card finishes and before the reply's first text: the walkthrough says to press them \"as soon as the first tool card shows\". In retry branch 1 the reply has started to stream (its first 100 lines have arrived) when the keys are pressed. In retry branch 2 the reply has finished, with the end of the turn, before the keys are pressed: the walkthrough's `reply finished first` case, which it says to try again with a longer prompt\n")
	fmt.Fprintf(&c.b, "# state per section: selection, gutter, key-hint row, mode, scroll, modal, size and glyphs, read from the Model\n")
	fmt.Fprintf(&c.b, "# the \"selection\" line is the item the gutter is on; \"none\" means no item is selected\n")
	fmt.Fprintf(&c.b, "# the \"gutter\" line gives the selected item's rows and the transcript window's rows: the gutter is drawn only when a row of the item is in the window\n")
	fmt.Fprintf(&c.b, "# the \"hint row\" line is the last row of the frame, or says there is none at that height\n")
	fmt.Fprintf(&c.b, "# section 10 resume: App.ResumeLastSession sends %v; this replay sends %v; not sent: %v\n",
		realResume, sentResume, minus(realResume, sentResume))
	fmt.Fprintf(&c.b, "# the resumed items in section 10 are the live model's own, not what the app builds from a session log, so the offset recorded there is not evidence of what a real resume shows\n")
	fmt.Fprintf(&c.b, "# live turns send further messages the replay leaves out, among them a BudgetMsg before each turn and a UsageMsg and a SessionUsageMsg for each usage report\n")

	chainSections1to3(c)

	// Section 4: the main path, then each retry branch on a separate model.
	chainSection4(c, "4 · the bar in the middle, with ↓ N new (main path: keys pressed before the reply's first text)", orderKeysFirst)
	for _, br := range []struct {
		title string
		order chainOrder
	}{
		{"4 · the bar in the middle, with ↓ N new (retry branch 1: the reply has started to stream before the keys)", orderReplyStartedFirst},
		{"4 · the bar in the middle, with ↓ N new (retry branch 2: the reply has finished before the keys: the walkthrough's `reply finished first`)", orderReplyFinishedFirst},
	} {
		branch := newChain(t)
		chainSections1to3(branch)
		branch.b.Reset()
		chainSection4(branch, br.title, br.order)
		c.b.WriteString(branch.b.String())
	}
	c.b.WriteString("\n(the sections below continue from the main path)\n")

	// Section 5: no bar under an open modal.
	c.begin("5 · no bar under an open modal")
	c.arrowUntil("a read_file card for README.md", onTool("README.md"))
	c.expectMode(ModeBrowsing, "the arrows")
	c.state("gutter on the card")
	c.keys("d")
	c.expectMode(ModeModal, "d")
	c.state("modal open")
	c.keys("Esc")
	c.expectMode(ModeBrowsing, "Esc")
	c.end()

	// Section 6: the long.txt modal.
	c.begin("6 · a wrapped line in a read_file modal")
	c.keys("Tab")
	c.expectMode(ModeComposing, "Tab")
	c.text("read long.txt and reply with one word")
	c.keys("Enter")
	c.deliver("(the read_file call for long.txt finishes)", capReadFileID(3, "long.txt", chainLongTxt())...)
	c.deliver("(the reply streams, the turn completes)",
		AssistantTextDeltaMsg{BlockID: 4, Delta: "Done."}, TurnCompleteMsg{})
	c.keys("Shift+↑")
	c.expectMode(ModeBrowsing, "Shift+↑")
	c.arrowUntil("the read_file card for long.txt", onTool("long.txt"))
	c.expectMode(ModeBrowsing, "the arrows")
	c.state("gutter on the card")
	c.keys("d")
	c.expectMode(ModeModal, "d")
	c.end()

	// Section 7: G, then close.
	c.begin("7 · G lands on the last full page")
	c.keys("G")
	c.state("after G")
	c.keys("Esc")
	c.expectMode(ModeBrowsing, "Esc")
	c.end()

	// Section 8: /diff.
	c.begin("8 · a wrapped + line and a wrapped @@ line in /diff")
	c.step("resize to 80x32")
	c.m = drive(c.m, tea.WindowSizeMsg{Width: 80, Height: 32})
	c.keys("Tab")
	c.expectMode(ModeComposing, "Tab")
	c.text("/diff")
	next, cmd := c.m.Update(keyType(tea.KeyEnter))
	c.m = next.(Model)
	c.step("key Enter")
	if cmd == nil {
		t.Fatal("/diff returned no command: the diff would never arrive")
	}
	got := cmd()
	res, ok := got.(DiffResultMsg)
	if !ok {
		t.Fatalf("the command /diff returns yields a %T, want DiffResultMsg", got)
	}
	c.deliver("(the /diff command's DiffResultMsg arrives)", res)
	c.expectMode(ModeModal, "the diff result")
	chainDiffInView(c)
	c.state("diff modal open")
	c.keys("Esc")
	c.expectMode(ModeComposing, "Esc")
	c.end()

	// Section 9: help at 60x24.
	c.begin("9 · help lines are cut, not wrapped, at 60x24")
	c.step("resize to 60x24")
	c.m = drive(c.m, tea.WindowSizeMsg{Width: 60, Height: 24})
	c.keys("Shift+↑")
	c.expectMode(ModeBrowsing, "Shift+↑")
	c.keys("?")
	c.expectMode(ModeModal, "?")
	c.state("help open")
	c.keys("Esc")
	c.expectMode(ModeBrowsing, "Esc")
	c.end()

	// Section 10: quit and resume with the C locale.
	c.begin("10 · the bar with ASCII glyphs")
	c.step("resize to 80x24")
	c.m = drive(c.m, tea.WindowSizeMsg{Width: 80, Height: 24})
	c.keys("Tab")
	c.expectMode(ModeComposing, "Tab")
	c.text("/quit")
	next, cmd = c.m.Update(keyType(tea.KeyEnter))
	c.m = next.(Model)
	c.step("key Enter")
	if cmd == nil {
		t.Fatal("/quit returned no command")
	}
	if quit := cmd(); quit == nil {
		t.Fatal("/quit's command yields no message, want tea.QuitMsg")
	} else if _, ok := quit.(tea.QuitMsg); !ok {
		t.Fatalf("/quit's command yields a %T, want tea.QuitMsg", quit)
	}
	items := append([]Item(nil), c.m.tr.Items()...)
	c.step("(a new process starts with LC_ALL=C: ascii glyphs, 80x24, no --new)")
	c.m = New(capOptions(t, false, false))
	c.unicode = false
	c.m = drive(c.m, tea.WindowSizeMsg{Width: 80, Height: 24})
	c.state("new process, before the resume messages")
	c.deliver("(the session resumes: a notice, then the transcript)", chainResumeMsgs(items)...)
	c.end()

	return c.b.String()
}

// diffInViewErr names why the canned diff's hunk is not in view without a j or PgDn
// press, or returns nil when it is: every row of the diff fits the modal's body
// (total rows, bodyH body rows), the modal is at offset off 0, and the frame holds
// the hunk header's first three fields.
func diffInViewErr(total, bodyH, off int, frame, hunk string) error {
	switch {
	case total > bodyH:
		return fmt.Errorf("the diff's %d rows do not fit the modal's %d-row body", total, bodyH)
	case off != 0:
		return fmt.Errorf("the modal is at offset %d, not 0", off)
	case !strings.Contains(frame, hunk):
		return fmt.Errorf("the frame does not hold the hunk header %q", hunk)
	}
	return nil
}

// chainDiffInView records the walkthrough step "press j or PgDn until the long.txt hunk
// is in view" for the canned diff, which holds one file, calc/divide.go, with one
// hunk. The count is worked out, not written down: it is zero presses only when
// diffInViewErr finds the diff in view at offset 0. A diff that diffInViewErr refuses
// stops the replay, because the replay does not press j or PgDn. The replay only tests
// diffInViewErr on the one diff it is given, whose rows the record shows filling the
// modal's body with none to spare, so a diffInViewErr whose row comparison was
// inverted would accept it and the replay would not notice; TestChainDiffInViewIsWorkedOut
// is what tests diffInViewErr itself. The numbers in the record are the model's own
// modalExtent and offset.
func chainDiffInView(c *chainRun) {
	c.t.Helper()
	total, bodyH := c.m.modalExtent(c.m.layout())
	hunk := strings.Join(strings.Fields(capIn(c.t).Diff.Lines[4])[:3], " ")
	if err := diffInViewErr(total, bodyH, c.m.modal.Off, c.m.View(), hunk); err != nil {
		c.t.Fatalf("the canned diff's hunk %q is not in view: %v; the replay would have to press j or PgDn", hunk, err)
	}
	c.step("j or PgDn until the calc/divide.go hunk (the walkthrough's long.txt hunk) is in view: 0 presses, the modal's offset is %d, its %d rows fit its %d-row body and the frame holds the hunk header %q", c.m.modal.Off, total, bodyH, hunk)
}

// TestChainDiffInViewIsWorkedOut: entry points diffInViewErr, called with a diff that is
// in view and with one input broken at a time, and chainDiffInView, called on a model
// that holds the canned diff at 80x32 as section 8 does. The real call must record the
// modal's own offset, its modalExtent rows and the hunk header, so a record that prints
// a number the model does not hold fails. Controls: the in-view input gives nil beside
// the broken ones, each of which names its own reason (rows that do not fit, an offset
// past 0, a hunk the frame lacks); the real call's frame is also the one the error
// function accepts.
func TestChainDiffInViewIsWorkedOut(t *testing.T) {
	const hunk = "@@ -7,7 +7,7 @@"
	frame := "x\n" + hunk + "\ny"
	if err := diffInViewErr(10, 12, 0, frame, hunk); err != nil {
		t.Fatalf("control: an in-view diff gave %v", err)
	}
	for _, tc := range []struct {
		name                string
		total, bodyH, off   int
		frame, wantContains string
	}{
		{"rows that do not fit", 13, 12, 0, frame, "do not fit"},
		{"an offset past 0", 10, 12, 3, frame, "offset 3"},
		{"a frame without the hunk", 10, 12, 0, "x\ny", "does not hold"},
	} {
		err := diffInViewErr(tc.total, tc.bodyH, tc.off, tc.frame, hunk)
		if err == nil || !strings.Contains(err.Error(), tc.wantContains) {
			t.Errorf("%s: diffInViewErr gave %v, want an error holding %q", tc.name, err, tc.wantContains)
		}
	}

	c := newChain(t)
	c.m = drive(c.m, tea.WindowSizeMsg{Width: 80, Height: 32})
	c.m = drive(c.m, DiffResultMsg{Result: capDiff(t), Seq: c.m.pendingDiffSeq})
	if c.m.modal == nil || c.m.modal.Kind != ModalDiff {
		t.Fatalf("setup: the DiffResultMsg did not open the diff modal (mode %s)", c.m.mode())
	}
	total, bodyH := c.m.modalExtent(c.m.layout())
	chainDiffInView(c)
	want := fmt.Sprintf("the modal's offset is %d, its %d rows fit its %d-row body", c.m.modal.Off, total, bodyH)
	if !strings.Contains(c.b.String(), want) {
		t.Errorf("the record lacks the model's own %q:\n%s", want, c.b.String())
	}
	if err := diffInViewErr(total, bodyH, c.m.modal.Off, c.m.View(), strings.Join(strings.Fields(capIn(t).Diff.Lines[4])[:3], " ")); err != nil {
		t.Errorf("control: the real frame was refused: %v", err)
	}
}

// TestChainArrowUntilFindsTheTargetInEitherDirection: entry point arrowUntil, which
// presses KeyUp and KeyDown through Update. The transcript is a reply, a README.md
// card, a reply and a long.txt card, with the gutter on the long.txt card. Looking
// for README.md, ↑ gets there in two presses (the reply between them is selectable)
// and the record says so. Looking for long.txt from README.md, ↑ runs out of items
// without getting there, which the record names, and ↓ gets there in two presses and
// the record says it started again from the state before the ↑ presses. Each
// direction runs through the same function, so a loop that only ever pressed ↑ would
// fail the second half, and one that stopped at once would fail the first.
func TestChainArrowUntilFindsTheTargetInEitherDirection(t *testing.T) {
	c := &chainRun{t: t, unicode: true}
	c.m = drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 80, Height: 24})
	c.m = drive(c.m, capTurn("first question", "first answer"))
	c.m = drive(c.m, capReadFileID(1, "README.md", "a\n")...)
	c.m = drive(c.m, capTurn("second question", "second answer"))
	c.m = drive(c.m, capReadFileID(2, "long.txt", "b\n")...)
	c.m = drive(c.m, keyType(tea.KeyShiftUp))
	if !onTool("long.txt")(c.m) || c.m.mode() != ModeBrowsing {
		t.Fatalf("setup: selection %s, mode %s, want the long.txt card in Browsing", c.identity(c.m.sel), c.m.mode())
	}

	arrowUntilWithin(t, c, "the README.md card", onTool("README.md"))
	if !onTool("README.md")(c.m) {
		t.Fatalf("after arrowUntil README.md the selection is %s", c.identity(c.m.sel))
	}
	arrowUntilWithin(t, c, "the long.txt card", onTool("long.txt"))
	if !onTool("long.txt")(c.m) {
		t.Fatalf("after arrowUntil long.txt the selection is %s", c.identity(c.m.sel))
	}
	rec := c.b.String()
	for _, want := range []string{
		"README.md card: ↑ pressed 2 times, 0 Shift+↑ to return to Browsing\n",
		fmt.Sprintf("long.txt card: ↑ for %d presses did not get there\n", chainArrowBound),
		"long.txt card: ↓ pressed 2 times, 0 Shift+↑ to return to Browsing, starting again from the state before the ↑ presses",
	} {
		if !strings.Contains(rec, want) {
			t.Errorf("the record lacks %q:\n%s", want, rec)
		}
	}
}

// TestChainArrowUntilNeedsTheCardInView: entry point arrowUntil. The gutter is drawn
// on the selected item only when a row of it is in the transcript window, so a card
// that is selected but scrolled out of the window is not "already there". The
// transcript is a long reply and then a README.md card, which a finished tool call
// selects; Shift+↑ and g leave the card selected with the window at the top. The
// control is the same model before g, where the card is selected and in view and
// arrowUntil presses nothing. With the card out of view, ↑ does not bring it back,
// and ↓ steps past the last item, which the walkthrough answers with Shift+↑: the
// record names the failed ↑, the one ↓ press and the one Shift+↑.
func TestChainArrowUntilNeedsTheCardInView(t *testing.T) {
	newModel := func() *chainRun {
		c := &chainRun{t: t, unicode: true}
		c.m = drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 80, Height: 24})
		c.m = drive(c.m, capTurn("question", capReplyLines(60)...))
		c.m = drive(c.m, capReadFileID(1, "README.md", "a\n")...)
		c.m = drive(c.m, keyType(tea.KeyShiftUp))
		return c
	}

	inView := newModel()
	if !onTool("README.md")(inView.m) {
		t.Fatalf("control setup: selection %s, want the README.md card selected and in view", inView.identity(inView.m.sel))
	}
	arrowUntilWithin(t, inView, "the README.md card", onTool("README.md"))
	if !strings.Contains(inView.b.String(), "already selected with a row in view, 0 presses") {
		t.Errorf("control: with the card selected and in view the record is:\n%s", inView.b.String())
	}

	c := newModel()
	c.m = drive(c.m, key('g'))
	if sel, _, ok := c.m.tr.Find(c.m.sel); !ok || sel.Kind != KindTool {
		t.Fatalf("setup: g moved the selection to %s", c.identity(c.m.sel))
	}
	if in, _, _, _, _, _ := selectedInView(c.m); in {
		t.Fatal("setup: the README.md card is still in view after g")
	}
	arrowUntilWithin(t, c, "the README.md card", onTool("README.md"))
	rec := c.b.String()
	for _, want := range []string{
		fmt.Sprintf("README.md card: ↑ for %d presses did not get there\n", chainArrowBound),
		"README.md card: ↓ pressed 1 times, 1 Shift+↑ to return to Browsing, starting again from the state before the ↑ presses",
	} {
		if !strings.Contains(rec, want) {
			t.Errorf("the record lacks %q:\n%s", want, rec)
		}
	}
	if strings.Contains(rec, "already") {
		t.Errorf("a card selected but out of view was recorded as already there:\n%s", rec)
	}
	if !onTool("README.md")(c.m) {
		t.Errorf("after arrowUntil the selection is %s, want the README.md card in view", c.identity(c.m.sel))
	}
}

// gutterRowsInFrame returns the rows of frame that carry the selection gutter: the
// rows whose cell in column 1, the first after the one-cell margin, is the glyph table's
// gutter glyph. The scrollbar's thumb is the same glyph but sits in the last column,
// so it is not counted.
func gutterRowsInFrame(frame string, gutter string) []int {
	var rows []int
	for i, row := range strings.Split(frame, "\n") {
		if cells := []rune(row); len(cells) > 1 && string(cells[1]) == gutter {
			rows = append(rows, i)
		}
	}
	return rows
}

// TestSelectedInViewAgreesWithTheRenderedGutter: entry points selectedInView, the
// predicate the chain replay records and arrowUntil stops on, and View(), the frame a
// terminal shows. The transcript is a long reply and a README.md card the finished tool
// call selects. Card in view: View() carries the gutter glyph in column 1 on exactly the
// rows of the card that lie in the window, and selectedInView says true. After g with the
// card out of the window: View() carries the glyph on no row, and selectedInView says
// false. With the diff modal open over the card in view: the frame still shows the
// gutter and also a modal box, so the record's "covers part of the window" holds. The
// control is that the in-view frame and the frame after g differ in the count of
// gutter rows, so the check can see presence and absence.
func TestSelectedInViewAgreesWithTheRenderedGutter(t *testing.T) {
	gutter := NewGlyphs(true).Gutter
	newModel := func() Model {
		m := drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 80, Height: 24})
		m = drive(m, capTurn("question", capReplyLines(60)...))
		m = drive(m, capReadFileID(1, "README.md", "a\n")...)
		return drive(m, keyType(tea.KeyShiftUp))
	}
	m := newModel()

	in, first, last, top, bot, ok := selectedInView(m)
	if !ok || !in {
		t.Fatalf("setup: the README.md card is selected and should be in view: %v %v", in, ok)
	}
	rows := gutterRowsInFrame(m.View(), gutter)
	lo, hi := max(first, top), min(last, bot)
	if len(rows) != hi-lo+1 {
		t.Errorf("card in view: the frame has the gutter on %d rows %v, selectedInView says the card has %d rows in the window (%d-%d, window %d-%d)", len(rows), rows, hi-lo+1, first, last, top, bot)
	}
	inViewCount := len(rows)

	m = drive(m, key('g'))
	in, _, _, _, _, _ = selectedInView(m)
	rows = gutterRowsInFrame(m.View(), gutter)
	if in || len(rows) != 0 {
		t.Errorf("card out of view: selectedInView %v, the frame has the gutter on rows %v, want false and none", in, rows)
	}
	if inViewCount == 0 {
		t.Error("control: the in-view frame has no gutter row, so an empty result above proves nothing")
	}

	m = drive(newModel(), key('d'))
	if m.modal == nil {
		t.Fatal("setup: d opened no modal on the card")
	}
	frame := m.View()
	if len(gutterRowsInFrame(frame, gutter)) == 0 {
		t.Errorf("a modal is open over the card: the frame has no gutter row, but the record says only part of the window is covered:\n%s", frame)
	}
	if !strings.Contains(frame, NewGlyphs(true).BoxTL) {
		t.Errorf("a modal is open: the frame has no box:\n%s", frame)
	}
}

// TestChainSection4OrderDecidesTheNewIndicator: entry point chainSection4 on a model
// after sections 1 to 3, run once in each ordering the table lists. Keys first (Shift+↑,
// g, PgDn before the reply's first text) must leave a ↓ N new count above 0, because the
// reply's block arrives below a view that is not pinned. Each ordering that puts the
// reply first must leave 0: when the reply has finished before the keys nothing arrives
// below the view afterwards, and when its first 100 lines are in, the rest joins the
// block already there and counts nothing. Each record's "after" scroll line must also
// carry the model's own offset, pin and NewSince, so the recorded `new` is the model's
// field. Control: the expectations include a count above 0 and a count of 0, so a run
// that gave every ordering the same count fails one of them.
func TestChainSection4OrderDecidesTheNewIndicator(t *testing.T) {
	for _, tc := range []struct {
		name    string
		order   chainOrder
		wantNew func(int) bool
		wantMsg string
	}{
		{"keys first", orderKeysFirst, func(n int) bool { return n > 0 }, "above 0"},
		{"reply started first", orderReplyStartedFirst, func(n int) bool { return n == 0 }, "0"},
		{"reply finished first", orderReplyFinishedFirst, func(n int) bool { return n == 0 }, "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newChain(t)
			chainSections1to3(c)
			c.b.Reset()
			chainSection4(c, "4", tc.order)
			if n := c.m.scroll.NewSince; !tc.wantNew(n) {
				t.Errorf("%s: NewSince is %d, want %s", tc.name, n, tc.wantMsg)
			}
			want := fmt.Sprintf("scroll: offset %d, pinned %v, new %d\n", c.m.scroll.Offset, c.m.scroll.Pinned, c.m.scroll.NewSince)
			if !strings.Contains(c.b.String(), "  "+want) {
				t.Errorf("%s: the record lacks the model's own %q:\n%s", tc.name, want, c.b.String())
			}
		})
	}
}

// TestToolCompletionSelectsItsCard: entry point Update with the messages that
// capReadFileID builds (ToolStartedMsg, ToolTargetMsg and ToolCompletedMsg), on a model
// that holds one reply. A finished tool call selects its own card, whether no item was
// selected before or the reply was. The chain replay's record depends on it: after
// section 1 it shows the README.md card selected in a turn where no arrow was pressed.
// Control: after the ToolStartedMsg alone, and after the ToolTargetMsg as well, the card
// exists and the selection is what it was, so the selection after the completion comes
// from the completion and not from either earlier message.
func TestToolCompletionSelectsItsCard(t *testing.T) {
	for _, c := range []struct {
		name   string
		before func(Model) Model
	}{
		{"nothing selected", func(m Model) Model { return m }},
		{"the reply selected", func(m Model) Model { return drive(m, keyType(tea.KeyShiftUp)) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 80, Height: 24})
			m = c.before(drive(m, capTurn("question", "answer")))
			was := m.sel
			msgs := capReadFileID(1, "README.md", "a\n")

			m = drive(m, msgs[0])
			card, ok := m.toolCards[1]
			if !ok {
				t.Fatal("control: the ToolStartedMsg made no card")
			}
			if m.sel != was || m.sel == card {
				t.Fatalf("control: after the ToolStartedMsg the selection is %d, want it unchanged at %d and not the card %d", m.sel, was, card)
			}

			m = drive(m, msgs[1])
			if m.sel != was {
				t.Fatalf("control: after the ToolTargetMsg the selection is %d, want it unchanged at %d", m.sel, was)
			}

			m = drive(m, msgs[2])
			if m.sel != card {
				t.Errorf("after the ToolCompletedMsg the selection is %d, want the card %d", m.sel, card)
			}
		})
	}
}
