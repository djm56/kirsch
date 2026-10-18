package tui

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// submitSlash types a slash command into the composer and presses Enter,
// returning the model and the command Update returned. Unlike drive, it keeps
// the tea.Cmd, which is where /diff's answer travels.
func submitSlash(t *testing.T, m Model, line string) (Model, tea.Cmd) {
	t.Helper()
	for _, r := range line {
		m = drive(m, key(r))
	}
	next, cmd := m.Update(keyType(tea.KeyEnter))
	return next.(Model), cmd
}

// readyModel is a driven fixture with its opening approval resolved, so keys
// reach the composer.
func readyModel(t *testing.T) Model {
	t.Helper()
	return drive(newDriven(t), key('y'))
}

// finishDiff runs the command /diff returned and feeds its message back through
// Update, the way the Bubble Tea runtime would.
func finishDiff(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("/diff returned no tea.Cmd; its result has no way back into Update")
	}
	msg := cmd()
	if _, ok := msg.(DiffResultMsg); !ok {
		t.Fatalf("/diff command produced %T, want DiffResultMsg", msg)
	}
	return drive(m, msg)
}

// lastNotice returns the text of the newest transcript item, failing if it is
// not a notice.
func lastNotice(t *testing.T, m Model) string {
	t.Helper()
	items := m.tr.Items()
	if len(items) == 0 {
		t.Fatal("no items in transcript")
	}
	last := items[len(items)-1]
	if last.Kind != KindNotice {
		t.Fatalf("last item kind = %v, want KindNotice", last.Kind)
	}
	return last.Notice.Text
}

// TestDiffCommandDoesNotBlockUpdate drives /diff through Update with a GetDiff
// that blocks until released. Update must return first, carrying a tea.Cmd, and
// no modal may exist until the result message is delivered.
func TestDiffCommandDoesNotBlockUpdate(t *testing.T) {
	m := readyModel(t)
	release := make(chan struct{})
	m.GetDiff = func() DiffResult {
		<-release
		return DiffResult{Lines: []string{"@@ -1 +1 @@", "+x"}, Added: 1}
	}

	type out struct {
		m   Model
		cmd tea.Cmd
	}
	done := make(chan out, 1)
	go func() {
		mm, cmd := submitSlash(t, m, "/diff")
		done <- out{mm, cmd}
	}()

	var got out
	select {
	case got = <-done:
	case <-time.After(2 * time.Second):
		close(release) // let the stuck goroutine finish before failing
		t.Fatal("Update blocked on GetDiff: /diff must run git in a tea.Cmd, not inline")
	}
	if got.m.modal != nil {
		t.Fatal("a modal was open before the diff result arrived")
	}
	if got.cmd == nil {
		t.Fatal("Update returned no tea.Cmd for /diff")
	}

	close(release)
	got.m = finishDiff(t, got.m, got.cmd)
	if got.m.modal == nil || got.m.modal.Kind != ModalDiff {
		t.Fatalf("modal after result = %+v, want ModalDiff", got.m.modal)
	}
}

// TestDiffCommandOpensModalWithCallbackLines checks, through the submit path,
// that the modal carries exactly the callback's lines and counts. The stub
// lines are unlike anything the old fake diff held, so a fake cannot match.
func TestDiffCommandOpensModalWithCallbackLines(t *testing.T) {
	m := readyModel(t)
	want := []string{"@@ -1,2 +1,3 @@", " keep", "-gone", "+one", "+two"}
	m.GetDiff = func() DiffResult {
		return DiffResult{Lines: want, Added: 2, Removed: 1}
	}

	m, cmd := submitSlash(t, m, "/diff")
	m = finishDiff(t, m, cmd)

	if m.modal == nil {
		t.Fatal("/diff did not open a modal")
	}
	if m.modal.Kind != ModalDiff {
		t.Errorf("modal kind = %v, want ModalDiff", m.modal.Kind)
	}
	if m.modal.Title != "working tree" {
		t.Errorf("modal title = %q, want %q", m.modal.Title, "working tree")
	}
	if m.modal.Added != 2 || m.modal.Removed != 1 {
		t.Errorf("modal counts = +%d -%d, want +2 -1", m.modal.Added, m.modal.Removed)
	}
	if strings.Join(m.modal.Lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("modal lines = %q, want %q", m.modal.Lines, want)
	}
}

// TestDiffCommandCleanTreeShowsNotice feeds the shape the app returns for a
// clean tree (no lines, no error) and expects the notice and no modal. The
// positive control is the sibling test above, where lines open a modal through
// the same path.
func TestDiffCommandCleanTreeShowsNotice(t *testing.T) {
	m := readyModel(t)
	m.GetDiff = func() DiffResult { return DiffResult{} }

	m, cmd := submitSlash(t, m, "/diff")
	m = finishDiff(t, m, cmd)

	if m.modal != nil {
		t.Fatalf("clean tree opened a modal: %+v", m.modal)
	}
	if got := lastNotice(t, m); got != "working tree clean" {
		t.Errorf("notice = %q, want %q", got, "working tree clean")
	}
}

// TestDiffCommandErrorShowsNotice checks an Err result becomes a notice with
// the error text and opens no modal.
func TestDiffCommandErrorShowsNotice(t *testing.T) {
	m := readyModel(t)
	m.GetDiff = func() DiffResult {
		return DiffResult{Err: "not a Git repository", Lines: []string{"+ignored"}}
	}

	m, cmd := submitSlash(t, m, "/diff")
	m = finishDiff(t, m, cmd)

	if m.modal != nil {
		t.Fatalf("an error result opened a modal: %+v", m.modal)
	}
	if got := lastNotice(t, m); got != "not a Git repository" {
		t.Errorf("notice = %q, want %q", got, "not a Git repository")
	}
}

// TestDiffCommandWithoutCallbackShowsNotice is the nil-callback arm: no
// command is returned and the notice says the diff is unavailable.
func TestDiffCommandWithoutCallbackShowsNotice(t *testing.T) {
	m := readyModel(t)
	m.GetDiff = nil

	m, cmd := submitSlash(t, m, "/diff")
	if cmd != nil {
		t.Error("/diff with no callback returned a tea.Cmd")
	}
	if got := lastNotice(t, m); got != "working tree diff not available" {
		t.Errorf("notice = %q, want %q", got, "working tree diff not available")
	}
}

// TestDiffResultTruncationMarkerDerivesFromCap checks the marker the modal adds
// under a truncated diff. The cap in the text comes from DiffResult.CapBytes: a
// 64 KiB cap must read 64, which a hard-coded figure cannot. The non-truncated
// case is the mirror, with no marker row.
func TestDiffResultTruncationMarkerDerivesFromCap(t *testing.T) {
	base := []string{"@@ -1 +1 @@", "+x"}

	t.Run("truncated carries the cap", func(t *testing.T) {
		m := readyModel(t)
		m.GetDiff = func() DiffResult {
			return DiffResult{Lines: base, Added: 1, Truncated: true, CapBytes: 64 << 10}
		}
		m, cmd := submitSlash(t, m, "/diff")
		m = finishDiff(t, m, cmd)
		if m.modal == nil {
			t.Fatal("no modal opened")
		}
		if len(m.modal.Lines) != len(base)+1 {
			t.Fatalf("modal has %d lines, want %d (marker appended)", len(m.modal.Lines), len(base)+1)
		}
		want := m.gly.Trunc + " truncated — 64 KiB cap"
		if got := m.modal.Lines[len(m.modal.Lines)-1]; got != want {
			t.Errorf("marker = %q, want %q", got, want)
		}
	})

	t.Run("not truncated has no marker", func(t *testing.T) {
		m := readyModel(t)
		m.GetDiff = func() DiffResult {
			return DiffResult{Lines: base, Added: 1, CapBytes: 64 << 10}
		}
		m, cmd := submitSlash(t, m, "/diff")
		m = finishDiff(t, m, cmd)
		if m.modal == nil {
			t.Fatal("no modal opened")
		}
		if len(m.modal.Lines) != len(base) {
			t.Errorf("modal lines = %q, want exactly %q", m.modal.Lines, base)
		}
	})
}

// TestDiffResultDoesNotReplaceAnOpenModal delivers a result while help is open
// and expects help to stay, with a notice saying how to retry.
func TestDiffResultDoesNotReplaceAnOpenModal(t *testing.T) {
	m := readyModel(t)
	m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})

	m = drive(m, DiffResultMsg{Result: DiffResult{Lines: []string{"@@ -1 +1 @@", "+x"}, Added: 1}})

	if m.modal == nil || m.modal.Kind != ModalHelp {
		t.Fatalf("modal = %+v, want the help modal untouched", m.modal)
	}
	if got := lastNotice(t, m); !strings.Contains(got, "run /diff again") {
		t.Errorf("notice = %q, want it to say \"run /diff again\"", got)
	}
}

// TestDiffModalLinesAreSanitised feeds the modal ESC, NUL and an embedded
// newline and expects none of them to survive. The positive control is that the
// text around each is still present and the row count is unchanged.
func TestDiffModalLinesAreSanitised(t *testing.T) {
	m := readyModel(t)
	in := []string{"+\x1b[31mred\x1b[0m", "+nul\x00byte", "+multi\nline", " plain"}
	m.GetDiff = func() DiffResult { return DiffResult{Lines: in, Added: 3} }

	m, cmd := submitSlash(t, m, "/diff")
	m = finishDiff(t, m, cmd)

	if m.modal == nil {
		t.Fatal("no modal opened")
	}
	if len(m.modal.Lines) != len(in) {
		t.Fatalf("modal has %d lines, want %d", len(m.modal.Lines), len(in))
	}
	joined := strings.Join(m.modal.Lines, "|")
	for name, bad := range map[string]string{"ESC": "\x1b", "NUL": "\x00", "newline": "\n"} {
		if strings.Contains(joined, bad) {
			t.Errorf("modal lines still contain %s: %q", name, m.modal.Lines)
		}
	}
	for _, keep := range []string{"red", "nul", "byte", "multi", "line", "plain"} {
		if !strings.Contains(joined, keep) {
			t.Errorf("modal lines lost %q: %q", keep, m.modal.Lines)
		}
	}
}

// TestFilesCommandListsPaths checks /files prints exactly the callback's paths.
func TestFilesCommandListsPaths(t *testing.T) {
	m := readyModel(t)
	m.GetTouchedFiles = func() []string { return []string{"a.go", "b.go", "c.go"} }

	m, _ = submitSlash(t, m, "/files")

	want := "files touched this session: a.go, b.go, c.go"
	if got := lastNotice(t, m); got != want {
		t.Errorf("/files notice = %q, want %q", got, want)
	}
}

// TestFilesCommandEmptyNotice checks the empty list reads as the empty notice,
// not as a list prefix with nothing after it.
func TestFilesCommandEmptyNotice(t *testing.T) {
	m := readyModel(t)
	m.GetTouchedFiles = func() []string { return []string{} }

	m, _ = submitSlash(t, m, "/files")

	if got := lastNotice(t, m); got != "no files touched this session" {
		t.Errorf("/files empty notice = %q, want %q", got, "no files touched this session")
	}
}

// TestFilesCommandWithoutCallbackShowsNotice is the nil-callback arm.
func TestFilesCommandWithoutCallbackShowsNotice(t *testing.T) {
	m := readyModel(t)
	m.GetTouchedFiles = nil

	m, _ = submitSlash(t, m, "/files")

	if got := lastNotice(t, m); got != "file tracking not available" {
		t.Errorf("notice = %q, want %q", got, "file tracking not available")
	}
}

// TestFilesCommandSanitisesPaths feeds ESC, NUL and newline in paths and
// expects none in the notice. The positive control is the surviving path text.
func TestFilesCommandSanitisesPaths(t *testing.T) {
	m := readyModel(t)
	m.GetTouchedFiles = func() []string {
		return []string{"path\nwith\nnewline.go", "path\x00with\x00nul.go", "esc\x1b[31m.go"}
	}

	m, _ = submitSlash(t, m, "/files")

	text := lastNotice(t, m)
	for name, bad := range map[string]string{"ESC": "\x1b", "NUL": "\x00", "newline": "\n"} {
		if strings.Contains(text, bad) {
			t.Errorf("/files notice still contains %s: %q", name, text)
		}
	}
	for _, keep := range []string{"newline.go", "nul.go", "esc"} {
		if !strings.Contains(text, keep) {
			t.Errorf("/files notice lost %q: %q", keep, text)
		}
	}
}

// runCmd executes cmd the way the Bubble Tea runtime does and returns every
// message it produced, flattening a tea.Batch and dropping the spinner's own
// ticks. It blocks until every command in the batch has returned.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, runCmd(c)...)
		}
		return out
	}
	if _, ok := msg.(spinnerTickMsg); ok {
		return nil
	}
	return []tea.Msg{msg}
}

// newSessionStub is a NewSession callback that records its calls and returns
// err once release is closed (or at once when release is nil).
type newSessionStub struct {
	calls   atomic.Int32
	release chan struct{}
	err     error
}

func (s *newSessionStub) fn(func(string) string) error {
	s.calls.Add(1)
	if s.release != nil {
		<-s.release
	}
	return s.err
}

// slashInGoroutine runs submitSlash on its own goroutine and reports whether
// Update returned within the timeout, which is how "Update did not block" is
// observed.
func slashInGoroutine(t *testing.T, m Model, line string, unblock func()) (Model, tea.Cmd) {
	t.Helper()
	type out struct {
		m   Model
		cmd tea.Cmd
	}
	done := make(chan out, 1)
	go func() {
		mm, cmd := submitSlash(t, m, line)
		done <- out{mm, cmd}
	}()
	select {
	case got := <-done:
		return got.m, got.cmd
	case <-time.After(2 * time.Second):
		unblock() // let the stuck goroutine finish before failing
		t.Fatalf("Update blocked on %s", line)
		return m, nil
	}
}

// TestSlashNewRunsCallbackInCmdAndResetsOnDone types /new and presses Enter.
// Update must return a tea.Cmd that runs the NewSession callback; nothing may
// reset, and the callback must not have run, until that Cmd's
// NewSessionDoneMsg is delivered to Update. The entry point is the keypress,
// through keyComposing, submit and runSlash into applyConfirm; the test never
// calls applyConfirm itself.
func TestSlashNewRunsCallbackInCmdAndResetsOnDone(t *testing.T) {
	m := readyModel(t)
	stub := &newSessionStub{}
	m.NewSession = stub.fn

	items := len(m.tr.Items())
	if items == 0 {
		t.Fatal("fixture transcript is empty: a reset would be unobservable")
	}
	m.pendingDiffSeq = 5
	m.sess.Compacted = true
	m.status.Tokens = 1000

	m, cmd := submitSlash(t, m, "/new")

	if got := stub.calls.Load(); got != 0 {
		t.Fatalf("NewSession ran %d time(s) inside Update, want 0 until the Cmd runs", got)
	}
	if len(m.tr.Items()) < items {
		t.Errorf("transcript shrank from %d to %d before the new session existed", items, len(m.tr.Items()))
	}
	if !m.busy.Active || m.busy.Verb != "new session" {
		t.Errorf("busy = {%v %q} while NewSession pending, want {true \"new session\"}", m.busy.Active, m.busy.Verb)
	}

	msgs := runCmd(cmd)
	if got := stub.calls.Load(); got != 1 {
		t.Fatalf("NewSession ran %d time(s) after the Cmd, want 1", got)
	}
	if len(msgs) != 1 {
		t.Fatalf("Cmd produced %d messages, want 1: %#v", len(msgs), msgs)
	}
	done, ok := msgs[0].(NewSessionDoneMsg)
	if !ok || done.Err != nil {
		t.Fatalf("Cmd produced %#v, want NewSessionDoneMsg{Err: nil}", msgs[0])
	}
	m = drive(m, done)

	if n := len(m.tr.Items()); n != 0 {
		t.Errorf("transcript holds %d items after the session started, want 0", n)
	}
	if m.pendingDiffSeq <= 5 {
		t.Errorf("pendingDiffSeq = %d after /new, want it advanced past 5", m.pendingDiffSeq)
	}
	if m.sess.Compacted {
		t.Error("Compacted flag not reset by /new")
	}
	if m.status.Tokens != 0 {
		t.Errorf("Tokens = %d after /new, want 0", m.status.Tokens)
	}
	if m.busy.Active || m.newSessionPending {
		t.Errorf("busy = %v, newSessionPending = %v after the done message, want both false", m.busy.Active, m.newSessionPending)
	}
}

// TestSlashNewDoesNotBlockUpdate runs /new against a NewSession that blocks
// until released. Update must return first; while it is pending Enter must be
// swallowed (no Submit) and Esc must not clear busy or reach Cancel.
func TestSlashNewDoesNotBlockUpdate(t *testing.T) {
	m := readyModel(t)
	stub := &newSessionStub{release: make(chan struct{})}
	m.NewSession = stub.fn
	var submitted, cancelled atomic.Int32
	m.Submit = func(string) { submitted.Add(1) }
	m.Cancel = func() { cancelled.Add(1) }

	m, cmd := slashInGoroutine(t, m, "/new", func() { close(stub.release) })

	if got := stub.calls.Load(); got != 0 {
		t.Fatalf("NewSession ran inside Update (%d calls)", got)
	}
	if !m.busy.Active {
		t.Fatal("busy not active while NewSession is pending")
	}

	// Input while pending: Esc keeps busy; typed text plus Enter does not submit.
	m = drive(m, keyType(tea.KeyEsc))
	if !m.busy.Active {
		t.Error("Esc cleared busy while NewSession was pending")
	}
	if got := cancelled.Load(); got != 0 {
		t.Errorf("Esc reached Cancel %d time(s) while NewSession was pending", got)
	}
	m = drive(m, key('h'), key('i'), keyType(tea.KeyEnter))
	if got := submitted.Load(); got != 0 {
		t.Errorf("Enter while NewSession was pending submitted %d time(s)", got)
	}

	close(stub.release)
	msgs := runCmd(cmd)
	if len(msgs) != 1 {
		t.Fatalf("Cmd produced %d messages, want 1: %#v", len(msgs), msgs)
	}
	m = drive(m, msgs[0])
	if m.busy.Active {
		t.Error("busy still active after the done message")
	}
}

// TestSlashNewFailureKeepsTranscript runs /new against a NewSession that
// fails. The transcript, the Compacted flag and the token display belong to
// the session that is still live, so none of them may be reset; the failure
// shows as a notice and busy clears.
func TestSlashNewFailureKeepsTranscript(t *testing.T) {
	m := readyModel(t)
	stub := &newSessionStub{err: errors.New("disk full")}
	m.NewSession = stub.fn
	m.sess.Compacted = true
	m.status.Tokens = 1000
	m.pendingDiffSeq = 5
	before := len(m.tr.Items())
	if before == 0 {
		t.Fatal("fixture transcript is empty: a wipe would be unobservable")
	}

	m, cmd := submitSlash(t, m, "/new")
	for _, msg := range runCmd(cmd) {
		m = drive(m, msg)
	}

	if stub.calls.Load() != 1 {
		t.Fatalf("NewSession calls = %d, want 1", stub.calls.Load())
	}
	if got := len(m.tr.Items()); got != before+1 {
		t.Errorf("transcript holds %d items, want the original %d plus one failure notice", got, before)
	}
	if got := lastNotice(t, m); !strings.Contains(got, "new session failed") || !strings.Contains(got, "disk full") {
		t.Errorf("notice = %q, want it to report the failure and its cause", got)
	}
	if !m.sess.Compacted || m.status.Tokens != 1000 || m.pendingDiffSeq != 5 {
		t.Errorf("state reset by a failed /new: Compacted=%v Tokens=%d pendingDiffSeq=%d", m.sess.Compacted, m.status.Tokens, m.pendingDiffSeq)
	}
	if m.busy.Active {
		t.Error("busy still active after a failed /new")
	}
}

// TestCompactCommandHoldsBusyUntilRealOutcome runs /compact against a
// ManualCompact stub that blocks until released and then returns a Nothing
// outcome. Until the Cmd returns, busy must stay active with the spinner
// scheduled, no notice may appear and the stub must have run exactly once;
// after release the message the Cmd carries is the stub's own, so the notice
// must be the nothing-to-compact one and not a synthesised success.
func TestCompactCommandHoldsBusyUntilRealOutcome(t *testing.T) {
	m := readyModel(t)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	m.ManualCompact = func() ManualCompactMsg {
		calls.Add(1)
		close(started)
		<-release
		return ManualCompactMsg{Outcome: ManualCompactNothing}
	}
	before := len(m.tr.Items())

	m, cmd := submitSlash(t, m, "/compact")

	if !m.busy.Active || m.busy.Verb != "compacting" || !m.compacting {
		t.Errorf("after /compact: busy = {%v %q}, compacting = %v; want {true \"compacting\"}, true", m.busy.Active, m.busy.Verb, m.compacting)
	}
	if !m.spinnerAlive {
		t.Error("/compact scheduled no spinner tick")
	}
	if len(m.tr.Items()) != before {
		t.Errorf("a notice appeared before the compaction outcome: %d items, had %d", len(m.tr.Items()), before)
	}

	result := make(chan []tea.Msg, 1)
	go func() { result <- runCmd(cmd) }()
	<-started
	select {
	case msgs := <-result:
		t.Fatalf("the /compact Cmd returned %#v while the callback was still running", msgs)
	case <-time.After(150 * time.Millisecond):
	}
	if !m.busy.Active {
		t.Error("busy cleared before the outcome arrived")
	}

	close(release)
	msgs := <-result
	if len(msgs) != 1 {
		t.Fatalf("Cmd produced %d messages, want exactly 1: %#v", len(msgs), msgs)
	}
	got, ok := msgs[0].(ManualCompactMsg)
	if !ok || got.Outcome != ManualCompactNothing {
		t.Fatalf("Cmd produced %#v, want the callback's ManualCompactMsg{Nothing}", msgs[0])
	}
	if calls.Load() != 1 {
		t.Errorf("callback ran %d times, want 1", calls.Load())
	}
	m = drive(m, got)
	if m.busy.Active || m.compacting {
		t.Error("busy not cleared by the outcome message")
	}
	if text := lastNotice(t, m); text != "nothing to compact yet" {
		t.Errorf("notice = %q, want the stub's own outcome \"nothing to compact yet\"", text)
	}
}

// TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc runs /compact with a
// blocking callback. While it runs, Esc must reach Cancel but leave busy
// active (the outcome message clears it), and Enter with typed text must not
// call Submit.
func TestCompactCommandRefusesSubmitAndKeepsBusyOnEsc(t *testing.T) {
	m := readyModel(t)
	release := make(chan struct{})
	m.ManualCompact = func() ManualCompactMsg {
		<-release
		return ManualCompactMsg{Outcome: ManualCompactCancelled}
	}
	var submitted, cancelled atomic.Int32
	m.Submit = func(string) { submitted.Add(1) }
	m.Cancel = func() { cancelled.Add(1) }

	m, cmd := submitSlash(t, m, "/compact")
	result := make(chan []tea.Msg, 1)
	go func() { result <- runCmd(cmd) }()

	m = drive(m, keyType(tea.KeyEsc))
	if got := cancelled.Load(); got != 1 {
		t.Errorf("Esc reached Cancel %d time(s), want 1", got)
	}
	if !m.busy.Active || !m.compacting {
		t.Errorf("Esc cleared busy (%v) or compacting (%v) before the outcome arrived", m.busy.Active, m.compacting)
	}
	m = drive(m, key('h'), key('i'), keyType(tea.KeyEnter))
	if got := submitted.Load(); got != 0 {
		t.Errorf("Enter during compaction called Submit %d time(s)", got)
	}

	close(release)
	msgs := <-result
	m = drive(m, msgs...)
	if m.busy.Active {
		t.Error("busy still active after the outcome")
	}
	if text := lastNotice(t, m); text != "compaction cancelled" {
		t.Errorf("notice = %q, want \"compaction cancelled\"", text)
	}
}

// TestManualCompactMsgOutcomesShowDistinctNotices delivers each outcome
// straight to Update and checks its notice. Each outcome has its own text, so
// one shared branch cannot satisfy two rows; the zero value and an unknown
// value are the mirror rows and must never read as success.
func TestManualCompactMsgOutcomesShowDistinctNotices(t *testing.T) {
	for _, tc := range []struct {
		name          string
		msg           ManualCompactMsg
		want          string
		wantCompacted bool
	}{
		{"applied", ManualCompactMsg{Outcome: ManualCompactApplied}, "compaction complete", true},
		{"nothing", ManualCompactMsg{Outcome: ManualCompactNothing}, "nothing to compact yet", false},
		{"cancelled", ManualCompactMsg{Outcome: ManualCompactCancelled}, "compaction cancelled", false},
		{"busy", ManualCompactMsg{Outcome: ManualCompactBusy}, "cannot compact while a turn is running", false},
		{"failed", ManualCompactMsg{Outcome: ManualCompactFailed, Detail: "boom\x1b[31m"}, "compaction failed: boom", false},
		{"zero value", ManualCompactMsg{}, "compaction failed: unrecognised outcome", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := readyModel(t)
			m.busy = Busy{Active: true, Verb: "compacting"}
			m.compacting = true

			m = drive(m, tc.msg)

			text := lastNotice(t, m)
			if !strings.HasPrefix(text, tc.want) {
				t.Errorf("notice = %q, want prefix %q", text, tc.want)
			}
			if strings.Contains(text, "\x1b") {
				t.Errorf("notice %q carries an escape byte", text)
			}
			if m.sess.Compacted != tc.wantCompacted {
				t.Errorf("Compacted = %v, want %v", m.sess.Compacted, tc.wantCompacted)
			}
			if m.busy.Active || m.compacting {
				t.Errorf("busy = %v, compacting = %v after the outcome, want both false", m.busy.Active, m.compacting)
			}
		})
	}
}

// TestCompactCommandWithoutCallback shows the fallback notice.
func TestCompactCommandWithoutCallback(t *testing.T) {
	m := readyModel(t)
	m.ManualCompact = nil

	m, _ = submitSlash(t, m, "/compact")

	// With no ManualCompact callback, it should show "nothing to compact yet"
	if got := lastNotice(t, m); got != "nothing to compact yet" {
		t.Errorf("notice = %q, want \"nothing to compact yet\"", got)
	}
}

// TestDiffResultIgnoresStaleSequence verifies that W-B is implemented:
// a diff result with an old sequence number is ignored.
func TestDiffResultIgnoresStaleSequence(t *testing.T) {
	m := readyModel(t)
	m.pendingDiffSeq = 2

	// First /diff with seq 1 (stale)
	oldResult := DiffResultMsg{
		Result: DiffResult{Lines: []string{"@@ -1 +1 @@", "+old"}, Added: 1},
		Seq:    1,
	}

	oldModalCount := 0
	if m.modal != nil {
		oldModalCount = 1
	}

	m = drive(m, oldResult)

	// Modal should not have opened
	if m.modal != nil && oldModalCount == 0 {
		t.Error("stale diff result opened a modal")
	}

	// Now send current sequence (should work)
	currentResult := DiffResultMsg{
		Result: DiffResult{Lines: []string{"@@ -1 +1 @@", "+new"}, Added: 1},
		Seq:    2,
	}

	m = drive(m, currentResult)

	// Modal should now be open
	if m.modal == nil || m.modal.Kind != ModalDiff {
		t.Fatal("current diff result did not open modal")
	}

	if !strings.Contains(strings.Join(m.modal.Lines, " "), "new") {
		t.Errorf("modal has wrong content: %v", m.modal.Lines)
	}
}

// TestDiffResultDroppedAfterNewSession types /diff, /new and /diff again, then
// delivers the second result followed by the first. The sequence number must
// never repeat across /new, so the first result is dropped even though the
// second /diff reuses a count the first one once held. The second result,
// delivered through the same path, is the positive control: it opens the modal.
func TestDiffResultDroppedAfterNewSession(t *testing.T) {
	m := readyModel(t)
	calls := 0
	m.GetDiff = func() DiffResult {
		calls++
		return DiffResult{Lines: []string{"@@ -1 +1 @@", "+call" + itoa(calls)}, Added: 1}
	}
	m.NewSession = func(func(string) string) error { return nil }

	m, oldCmd := submitSlash(t, m, "/diff")
	oldMsg := runCmd(oldCmd) // GetDiff call 1: "+call1", Seq 1
	if len(oldMsg) != 1 {
		t.Fatalf("first /diff produced %d messages, want 1", len(oldMsg))
	}

	m, newCmd := submitSlash(t, m, "/new")
	for _, msg := range runCmd(newCmd) {
		m = drive(m, msg)
	}
	m, freshCmd := submitSlash(t, m, "/diff")
	freshMsg := runCmd(freshCmd) // GetDiff call 2: "+call2"
	if len(freshMsg) != 1 {
		t.Fatalf("second /diff produced %d messages, want 1", len(freshMsg))
	}

	m = drive(m, freshMsg[0])
	if m.modal == nil || !strings.Contains(strings.Join(m.modal.Lines, " "), "+call2") {
		t.Fatalf("the post-/new result did not open the modal: %+v", m.modal)
	}
	m = drive(m, keyType(tea.KeyEsc)) // close it
	if m.modal != nil {
		t.Fatal("could not close the modal")
	}

	m = drive(m, oldMsg[0])
	if m.modal != nil {
		t.Errorf("the pre-/new diff result opened a modal after /new: %+v", m.modal)
	}
}

// TestDiffResultBlockedByConfirmPrompt verifies W-A: a confirm prompt is not
// covered by a diff modal, and the diff shows a notice instead (W-A test).
func TestDiffResultBlockedByConfirmPrompt(t *testing.T) {
	m := readyModel(t)

	// Raise a confirm prompt (ConfirmNewSession would work, but any confirm is fine)
	m.askConfirm(ConfirmState{
		Prompt: "Test confirm",
		Action: ConfirmNewSession,
	})

	// Verify confirm is open
	if m.confirm == nil {
		t.Fatal("failed to set up confirm prompt")
	}

	initialItems := len(m.tr.Items())

	// Set pendingDiffSeq to 1 (as if /diff was called)
	m.pendingDiffSeq = 1

	// Now send a diff result with matching seq
	diffResult := DiffResultMsg{
		Result: DiffResult{Lines: []string{"@@ -1 +1 @@", "+x"}, Added: 1},
		Seq:    1,
	}

	m = drive(m, diffResult)

	// Modal should not have opened (confirm takes precedence via overlayOpen)
	if m.modal != nil {
		t.Errorf("diff opened modal while confirm was visible: %+v", m.modal)
	}

	// Confirm should still be open
	if m.confirm == nil {
		t.Error("confirm prompt was closed by diff result")
	}

	// A notice should have been added saying to retry
	items := m.tr.Items()
	if len(items) > initialItems {
		got := items[len(items)-1].Notice.Text
		if !strings.Contains(got, "run /diff again") {
			t.Errorf("notice = %q, want it to say \"run /diff again\"", got)
		}
	} else {
		t.Error("no notice added when diff result blocked by confirm")
	}
}
