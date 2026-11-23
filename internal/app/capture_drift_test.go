package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// The screen captures under plan/testing/manual/captures are drawn from canned
// inputs in internal/tui/capture_test.go. internal/tui must not import
// internal/tool (internal/arch), so these tests, in internal/app, which imports both
// internal/tui and internal/tool, hold the real producers and the committed capture
// files to the literals and patterns in internal/tui/testdata/capture-inputs.json.
// The tui tests read the same file, so there is one copy of each literal: change the
// file and the tests in each package move with it.

// captureInputsFile is the part of capture-inputs.json these tests read. Field names
// match the file's keys, case-insensitively.
type captureInputsFile struct {
	Options struct {
		Version, Project, Branch, Model, Family string
		Dirty                                   bool
	}
	FrameSep string
	ReadFile struct {
		Text, Summary, Content, OneLineText, OneLineSummary string
	}
	ModalReadFile struct{ Path, Text string }
	LiveReadFile  liveReadFilePin
	Diff          struct {
		Path, Old, New string
		Lines          []string
		Added, Removed int
		HeaderShapes   []string
	}
	DiffTwoFiles struct {
		Files []struct {
			Path, Old, New string
			IntentToAdd    bool
		}
		Lines          []string
		Added, Removed int
	}
	PatchApproval  patchApprovalPin
	ResumeMessages []string
	Committed      []struct {
		File         string
		MustMatch    []string
		MustNotMatch []string
	}
}

// liveReadFilePin is the part of capture-inputs.json that pins what a model-driven
// read_file call for modalReadFile.path sends the tui, as the live path sends it:
// Started, Targeted and Completed are its messages, in order.
type liveReadFilePin struct {
	Started struct {
		ID           int64
		Name, Target string
	}
	Targeted struct {
		ID     int64
		Target string
	}
	Completed struct {
		ID               int64
		Name, Target     string
		OK               bool
		Summary, Content string
	}
}

// patchApprovalPin is the part of capture-inputs.json that pins what a model-driven
// apply_patch call sends the tui, as the real producer sends it. Input is what the call
// is given; Started, Requested and Completed are its messages, in order.
type patchApprovalPin struct {
	Input   struct{ Diff, Description string }
	Started struct {
		ID           int64
		Name, Target string
	}
	Requested struct {
		ID                         int64
		Kind, Subject, Description string
		Detail                     []string
		CanApproveForSession       bool
		ToolID                     int64
		DiffFilename               string
		DiffLines                  []string
		DiffAdded, DiffRemoved     int
	}
	Completed struct {
		OK                           bool
		Summary, ErrorKind, ErrorMsg string
	}
}

// isolateGit keeps the user's git configuration out of the git commands a test
// runs: the global and system files are not read. Without it a setting such as
// core.abbrev changes the blob hashes in the index row of a diff, and the output the
// tests compare byte for byte depends on the machine.
func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// loadCaptureInputs reads the shared inputs file.
func loadCaptureInputs(t *testing.T) captureInputsFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "tui", "testdata", "capture-inputs.json"))
	if err != nil {
		t.Fatalf("read capture inputs: %v", err)
	}
	var in captureInputsFile
	if err := json.Unmarshal(raw, &in); err != nil {
		t.Fatalf("parse capture inputs: %v", err)
	}
	return in
}

// committedCapture reads one committed v0.1.5 capture file.
func committedCapture(t *testing.T, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "plan", "testing", "manual", "captures", "v0.1.5", file))
	if err != nil {
		t.Fatalf("read committed capture: %v", err)
	}
	return string(b)
}

// committedFrame is the frame of a committed capture: the text after the header's
// separator line.
func committedFrame(t *testing.T, in captureInputsFile, file string) string {
	t.Helper()
	text := committedCapture(t, file)
	_, frame, ok := strings.Cut(text, in.FrameSep)
	if !ok {
		t.Fatalf("%s has no %q line", file, in.FrameSep)
	}
	return frame
}

// committedProblems names what is wrong with content under one spec: each MustMatch
// pattern with no match and each MustNotMatch pattern with one.
func committedProblems(content string, mustMatch, mustNotMatch []string) []string {
	var out []string
	for _, p := range mustMatch {
		if !regexp.MustCompile(p).MatchString(content) {
			out = append(out, "no match for "+p)
		}
	}
	for _, p := range mustNotMatch {
		if regexp.MustCompile(p).MatchString(content) {
			out = append(out, "matches "+p)
		}
	}
	return out
}

// captureModel is the tui model every capture is built from, at w by h, colour off,
// Unicode glyphs, with the options the shared file names.
func captureModel(in captureInputsFile, w, h int) tui.Model {
	o := in.Options
	m := tui.New(tui.Options{
		Version: o.Version,
		Caps:    tui.Caps{Colour: false, Unicode: true},
		Session: tui.SessionInfo{Project: o.Project, Branch: o.Branch, Dirty: o.Dirty},
		Status:  tui.Status{Model: o.Model, Family: o.Family},
	})
	return feed(m, tea.WindowSizeMsg{Width: w, Height: h})
}

// feed applies messages to m through Update, discarding commands.
func feed(m tui.Model, msgs ...tea.Msg) tui.Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}
	return m
}

// keysOf is the key messages for text, one rune each.
func keysOf(text string) []tea.Msg {
	var out []tea.Msg
	for _, r := range text {
		out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return out
}

// liveCallID is the provider call ID of the tool calls the live tests script. The
// tui never sees it; it is how adaptTurnEvent and toolExecutor.Invoke find the
// card of one call.
const liveCallID = "toolu_01CaptureCall"

// readFileMessages runs a model-driven read_file call for path through the live
// path (live_tool_test.go: Submit, the agent loop, a stream the real
// anthropic.Decode reads, adaptTurnEvent and toolExecutor.Invoke) and returns the
// tool messages the app sends the tui, in order: ToolStartedMsg with no target,
// ToolTargetMsg and ToolCompletedMsg. The file is in a temporary repo that holds
// text; git runs there with the user's git configuration isolated (isolateGit).
func readFileMessages(t *testing.T, path, text string) []tea.Msg {
	t.Helper()
	repo := newGitRepo(t, map[string]string{"seed.txt": "seed\n"})
	full := filepath.Join(repo, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	input, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	e := newLiveEnv(t, repo, nil, sseToolUse(liveCallID, "read_file", string(input)), sseReply("done"))
	var out []tea.Msg
	for _, m := range toolMsgs(e.turn("read " + path)) {
		out = append(out, m)
	}
	return out
}

// readFileProblems names each field of the messages a model-driven read_file call sent
// that differs from the pin, and a completion whose Elapsed is not above zero. The
// pin holds no Elapsed value: the live path times the call, so only that it is set
// is checked. got must be these messages in order: ToolStartedMsg, ToolTargetMsg,
// ToolCompletedMsg.
func readFileProblems(p liveReadFilePin, got []tea.Msg) []string {
	var out []string
	if len(got) != 3 {
		return []string{fmt.Sprintf("the call sent %d tool messages, want ToolStartedMsg, ToolTargetMsg and ToolCompletedMsg: %#v", len(got), got)}
	}
	eq := func(what string, got, want any) {
		if !reflect.DeepEqual(got, want) {
			out = append(out, fmt.Sprintf("%s is %#v, the pin says %#v", what, got, want))
		}
	}
	start, ok := got[0].(tui.ToolStartedMsg)
	if !ok {
		return []string{fmt.Sprintf("the first message is a %T, want ToolStartedMsg", got[0])}
	}
	eq("started ID", start.ID, p.Started.ID)
	eq("started name", start.Name, p.Started.Name)
	eq("started target", start.Target, p.Started.Target)
	tgt, ok := got[1].(tui.ToolTargetMsg)
	if !ok {
		return append(out, fmt.Sprintf("the second message is a %T, want ToolTargetMsg", got[1]))
	}
	eq("targeted ID", tgt.ID, p.Targeted.ID)
	eq("targeted target", tgt.Target, p.Targeted.Target)
	done, ok := got[2].(tui.ToolCompletedMsg)
	if !ok {
		return append(out, fmt.Sprintf("the third message is a %T, want ToolCompletedMsg", got[2]))
	}
	c := p.Completed
	eq("completed ID", done.ID, c.ID)
	eq("completed name", done.Name, c.Name)
	eq("completed target", done.Target, c.Target)
	eq("completed OK", done.OK, c.OK)
	eq("completed summary", done.Summary, c.Summary)
	eq("completed content", done.Content, c.Content)
	if done.Elapsed <= 0 {
		out = append(out, fmt.Sprintf("completed Elapsed is %v, want above zero: the live path times every tool call", done.Elapsed))
	}
	return out
}

// TestCaptureLiveReadFileMatchesProducers: entry point Submit, through the agent loop
// and a stream that the real anthropic.Decode reads (readFileMessages), for a
// read_file call on modalReadFile.path. The tool messages must match
// liveReadFile field by field: ToolStartedMsg with no target, because the Anthropic
// start event carries no input; ToolTargetMsg with the path; ToolCompletedMsg with the
// path, the summary and the numbered content, and a completion with Elapsed above zero.
// Controls: a copy of the pin with another target in the targeted message, one with a
// target in the started message, and one with another completed summary each report a
// problem, as does a copy of the messages whose completion has Elapsed zero, beside the
// real pin and messages, which report none.
func TestCaptureLiveReadFileMatchesProducers(t *testing.T) {
	in := loadCaptureInputs(t)
	msgs := readFileMessages(t, in.ModalReadFile.Path, in.ModalReadFile.Text)
	pin := in.LiveReadFile
	if p := readFileProblems(pin, msgs); len(p) != 0 {
		t.Errorf("the messages the live read_file call sent differ from capture-inputs.json: %v", p)
	}
	wrongTarget := pin
	wrongTarget.Targeted.Target = "other.go"
	if len(readFileProblems(wrongTarget, msgs)) == 0 {
		t.Error("control: a pin with another targeted file reported no problem")
	}
	startTarget := pin
	startTarget.Started.Target = pin.Targeted.Target
	if len(readFileProblems(startTarget, msgs)) == 0 {
		t.Error("control: a pin whose started message has a target reported no problem")
	}
	wrongSummary := pin
	wrongSummary.Completed.Summary = "lines 1-9"
	if len(readFileProblems(wrongSummary, msgs)) == 0 {
		t.Error("control: a pin with another completed summary reported no problem")
	}
	zeroed := append([]tea.Msg(nil), msgs...)
	if done, ok := zeroed[2].(tui.ToolCompletedMsg); ok {
		done.Elapsed = 0
		zeroed[2] = done
	} else {
		t.Fatalf("setup: message at index 2 is a %T, want ToolCompletedMsg", zeroed[2])
	}
	if len(readFileProblems(pin, zeroed)) == 0 {
		t.Error("control: a completion with Elapsed zero reported no problem")
	}
}

// TestCaptureInputsMatchProducers: entry points are the real read_file tool's
// Invoke on a temporary file, tool.RunGit with tool.DiffArgs and GetDiff on a
// temporary git repo that holds the canned diff's old and new file, and the
// committed scrollbar-bottom and modal-readfile-wrapped capture files. The tool's
// DisplaySummary and Content must equal the file's literals. The raw git output must
// equal the file's diff rows, row for row, and GetDiff's rows and counts must equal
// them after SanitizeLines. The committed files must satisfy the file's patterns.
// Positive controls: a one-line file gives the range "lines 1-1", not a bare count;
// the real diff has more rows than the header patterns, so the patterns were read off
// a real diff with a hunk; and the pattern check fails on a copy of each committed
// file with its summary or numbering edited, beside the real file, which passes. Git runs
// with the user's git configuration isolated (isolateGit).
func TestCaptureInputsMatchProducers(t *testing.T) {
	isolateGit(t)
	in := loadCaptureInputs(t)

	dir := t.TempDir()
	for name, body := range map[string]string{"two.txt": in.ReadFile.Text, "one.txt": in.ReadFile.OneLineText} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := workspace.Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	rf := &tool.ReadFile{WS: ws}
	read := func(path string) tool.Result {
		input, err := json.Marshal(map[string]string{"path": path})
		if err != nil {
			t.Fatal(err)
		}
		res := rf.Invoke(context.Background(), input)
		if !res.OK {
			t.Fatalf("read_file %s failed: %+v", path, res.Error)
		}
		return res
	}
	two := read("two.txt")
	if two.DisplaySummary != in.ReadFile.Summary || two.Content != in.ReadFile.Content {
		t.Errorf("read_file gave summary %q content %q, want %q and %q (testdata/capture-inputs.json)",
			two.DisplaySummary, two.Content, in.ReadFile.Summary, in.ReadFile.Content)
	}
	if one := read("one.txt"); one.DisplaySummary != in.ReadFile.OneLineSummary {
		t.Errorf("control: one-line file summary %q, want %q", one.DisplaySummary, in.ReadFile.OneLineSummary)
	}

	repo := diffRepo(t, in)
	raw, err := tool.RunGit(context.Background(), repo, tool.DiffArgs(false, "")...)
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	rawLines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	if len(rawLines) <= len(in.Diff.HeaderShapes) {
		t.Fatalf("control: the real diff has %d rows, want more than the %d header rows", len(rawLines), len(in.Diff.HeaderShapes))
	}
	if strings.Join(rawLines, "\n") != strings.Join(in.Diff.Lines, "\n") {
		t.Errorf("the canned diff is not git's output for its old and new file:\n got  %q\n want %q", rawLines, in.Diff.Lines)
	}
	for i, shape := range in.Diff.HeaderShapes {
		if !regexp.MustCompile(shape).MatchString(rawLines[i]) {
			t.Errorf("real git diff row %d is %q, want it to match %s", i, rawLines[i], shape)
		}
	}
	res := newDiffApp(repo).GetDiff()
	if res.Err != "" {
		t.Fatalf("GetDiff: %s", res.Err)
	}
	if want := tui.SanitizeLines(strings.Join(in.Diff.Lines, "\n")); strings.Join(res.Lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("GetDiff rows differ from the canned diff after SanitizeLines:\n got  %q\n want %q", res.Lines, want)
	}
	if res.Added != in.Diff.Added || res.Removed != in.Diff.Removed {
		t.Errorf("GetDiff counts +%d -%d, the canned diff says +%d -%d", res.Added, res.Removed, in.Diff.Added, in.Diff.Removed)
	}

	if len(in.Committed) == 0 {
		t.Fatal("capture-inputs.json lists no committed file to check")
	}
	for _, spec := range in.Committed {
		content := committedCapture(t, spec.File)
		if p := committedProblems(content, spec.MustMatch, spec.MustNotMatch); len(p) != 0 {
			t.Errorf("%s: %v", spec.File, p)
		}
		// Control: the same check on a copy with the first pattern's matches removed.
		broken := regexp.MustCompile(spec.MustMatch[0]).ReplaceAllString(content, "")
		if p := committedProblems(broken, spec.MustMatch, spec.MustNotMatch); len(p) == 0 {
			t.Errorf("control: %s with its first pattern's matches removed still passes the check", spec.File)
		}
	}
	// Control for the absence patterns: scrollbar-bottom with its summary edited back to
	// "read" must trip the missing-summary pattern and the bare-summary pattern.
	bottom := committedCapture(t, "scrollbar-bottom.txt")
	edited := regexp.MustCompile(`lines \d+-\d+`).ReplaceAllString(bottom, "read")
	if edited == bottom {
		t.Fatal("setup: the edit changed nothing")
	}
	for _, spec := range in.Committed {
		if spec.File != "scrollbar-bottom.txt" {
			continue
		}
		if p := committedProblems(edited, spec.MustMatch, spec.MustNotMatch); len(p) != 2 {
			t.Errorf("scrollbar-bottom.txt with the summary edited back to \"read\": problems %v, want a missing summary and a bare summary", p)
		}
	}
}

// diffRepo makes a repo whose first commit holds the canned diff's old file and
// whose working tree holds its new file, so git diff shows the canned change.
func diffRepo(t *testing.T, in captureInputsFile) string {
	t.Helper()
	repo := t.TempDir()
	write := func(text string) {
		path := filepath.Join(repo, filepath.FromSlash(in.Diff.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, repo, "init")
	write(in.Diff.Old)
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-m", "initial")
	write(in.Diff.New)
	return repo
}

// TestCaptureReadFileChainMatchesCommitted: entry point Submit, with a model-driven
// read_file call on the real tool (readFileMessages), whose tool messages
// (the ToolCompletedMsg carries res.DisplaySummary and res.Content, app.go's Summary
// and Content mapping) go through Update on a tui model; the model's View is then
// compared with the committed frames. The modal frame must equal modal-readfile-wrapped.txt, which
// carries the tool's own line numbers; the card's head line, read from the model with
// no modal open, must match the summary pattern the committed scrollbar-bottom
// capture is held to. Control: the same pattern check fails on that head line with
// its summary replaced by the bare word "read".
func TestCaptureReadFileChainMatchesCommitted(t *testing.T) {
	in := loadCaptureInputs(t)

	msgs := readFileMessages(t, in.ModalReadFile.Path, in.ModalReadFile.Text)
	m := feed(captureModel(in, 80, 24), msgs...)
	card := m.View()

	var bottom []string
	var bottomNot []string
	for _, spec := range in.Committed {
		if spec.File == "scrollbar-bottom.txt" {
			bottom, bottomNot = spec.MustMatch, spec.MustNotMatch
		}
	}
	if len(bottom) == 0 {
		t.Fatal("capture-inputs.json has no scrollbar-bottom.txt spec")
	}
	if p := committedProblems(card, bottom, bottomNot); len(p) != 0 {
		t.Errorf("the card the real read_file result draws: %v\n%s", p, card)
	}
	if p := committedProblems(strings.Replace(card, " · lines ", " · read ", 1), bottom, bottomNot); len(p) == 0 {
		t.Error("control: the pattern check passes on a card whose summary was edited")
	}

	m = feed(m, tea.KeyMsg{Type: tea.KeyShiftUp}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := m.View() + "\n"
	want := committedFrame(t, in, "modal-readfile-wrapped.txt")
	if got != want {
		t.Errorf("the modal the real read_file result draws differs from modal-readfile-wrapped.txt:\n got  %q\n want %q", got, want)
	}
}

// TestCaptureDiffChainMatchesCommitted: entry point Update on a tui model whose
// GetDiff is the real App.GetDiff, in a temporary repo that holds the canned
// diff's old and new file. The test types /diff and presses Enter, runs the command
// Update returns, which calls GetDiff and yields the DiffResultMsg, and feeds that
// message back through Update. The View must equal the frame of the committed
// modal-diff-wrapped.txt, and the committed file's header block, which includes the
// index row with the files' blob hashes, must hold git's own first rows. Git runs
// with the user's git configuration isolated (isolateGit).
func TestCaptureDiffChainMatchesCommitted(t *testing.T) {
	isolateGit(t)
	in := loadCaptureInputs(t)
	repo := diffRepo(t, in)
	app := newDiffApp(repo)

	m := captureModel(in, 80, 32)
	m.GetDiff = app.GetDiff
	m = feed(m, keysOf("/diff")...)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(tui.Model)
	if cmd == nil {
		t.Fatal("/diff returned no command")
	}
	msg := cmd()
	if _, ok := msg.(tui.DiffResultMsg); !ok {
		t.Fatalf("the /diff command yields a %T, want DiffResultMsg", msg)
	}
	m = feed(m, msg)

	got := m.View() + "\n"
	want := committedFrame(t, in, "modal-diff-wrapped.txt")
	if got != want {
		t.Errorf("the modal the real GetDiff result draws differs from modal-diff-wrapped.txt:\n got  %q\n want %q", got, want)
	}

	committed := committedCapture(t, "modal-diff-wrapped.txt")
	res := app.GetDiff()
	if len(res.Lines) < 5 {
		t.Fatalf("GetDiff gave %d rows, want the header block and a hunk", len(res.Lines))
	}
	for _, row := range res.Lines[:4] {
		if !strings.Contains(committed, row) {
			t.Errorf("modal-diff-wrapped.txt does not hold git's header row %q", row)
		}
	}
	if !strings.Contains(committed, res.Lines[4][:30]) {
		t.Errorf("modal-diff-wrapped.txt does not hold the start of git's hunk header %q", res.Lines[4])
	}
}

// twoFileRepo makes a temporary repo and returns its path. It runs git init, writes
// the old text of each pinned file whose Old is non-empty (creating parent
// directories), and commits with git add . and git commit. It then writes the New
// text of every pinned file over or beside that, and runs git add -N on the files
// whose pin sets IntentToAdd, which is how the walkthrough makes /diff list a new
// file. A file with an empty Old is therefore not in the commit. It does not isolate
// git's configuration; its one caller, TestCaptureTwoFileDiffMatchesGit, calls
// isolateGit first.
func twoFileRepo(t *testing.T, in captureInputsFile) string {
	t.Helper()
	repo := t.TempDir()
	write := func(path, text string) {
		full := filepath.Join(repo, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, repo, "init")
	for _, f := range in.DiffTwoFiles.Files {
		if f.Old != "" {
			write(f.Path, f.Old)
		}
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-m", "initial")
	for _, f := range in.DiffTwoFiles.Files {
		write(f.Path, f.New)
		if f.IntentToAdd {
			gitIn(t, repo, "add", "-N", f.Path)
		}
	}
	return repo
}

// TestCaptureTwoFileDiffMatchesGit: entry points are tool.RunGit with tool.DiffArgs and
// GetDiff on a temporary repo that twoFileRepo builds from the pinned files: each
// pinned file with an old text is committed and then changed to its new text, a file
// with no old text is written after the commit, and a file whose pin sets IntentToAdd
// is registered with git add -N. The raw output must equal diffTwoFiles.lines row for row,
// and GetDiff's rows and counts must equal them after SanitizeLines. Git runs with the
// user's git configuration isolated. Control: the same comparison on a copy of the
// pinned rows with one cell of the new file's index row changed reports a difference,
// beside the real rows, which it accepts.
func TestCaptureTwoFileDiffMatchesGit(t *testing.T) {
	isolateGit(t)
	in := loadCaptureInputs(t)
	repo := twoFileRepo(t, in)
	raw, err := tool.RunGit(context.Background(), repo, tool.DiffArgs(false, "")...)
	if err != nil {
		t.Fatalf("git diff: %v", err)
	}
	rawLines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	if len(in.DiffTwoFiles.Lines) < 10 {
		t.Fatalf("control: the pinned diff has %d rows, want at least 10", len(in.DiffTwoFiles.Lines))
	}
	if strings.Join(rawLines, "\n") != strings.Join(in.DiffTwoFiles.Lines, "\n") {
		t.Errorf("the pinned diff is not git's output for its files:\n got  %q\n want %q", rawLines, in.DiffTwoFiles.Lines)
	}
	bad := append([]string(nil), in.DiffTwoFiles.Lines...)
	for i, l := range bad {
		if strings.HasPrefix(l, "index 0000000..") {
			bad[i] = strings.Replace(l, "index 0000000..", "index 0000001..", 1)
		}
	}
	if strings.Join(rawLines, "\n") == strings.Join(bad, "\n") {
		t.Error("control: git's output equals a copy of the pinned rows with the index row edited")
	}

	res := newDiffApp(repo).GetDiff()
	if res.Err != "" {
		t.Fatalf("GetDiff: %s", res.Err)
	}
	if want := tui.SanitizeLines(strings.Join(in.DiffTwoFiles.Lines, "\n")); strings.Join(res.Lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("GetDiff rows differ from the pinned rows after SanitizeLines:\n got  %q\n want %q", res.Lines, want)
	}
	if res.Added != in.DiffTwoFiles.Added || res.Removed != in.DiffTwoFiles.Removed {
		t.Errorf("GetDiff counts +%d -%d, the pin says +%d -%d", res.Added, res.Removed, in.DiffTwoFiles.Added, in.DiffTwoFiles.Removed)
	}
}

// TestCaptureGitIsolationIgnoresUserConfig: the control for isolateGit. A user git
// configuration that sets core.abbrev to 12 is written to a temporary HOME. Before
// isolateGit, git's diff of the pinned file shows 12-character blob hashes in its index
// row, so the setting does reach the commands; after isolateGit, the same repo's diff
// shows the 7-character hashes the pin holds.
func TestCaptureGitIsolationIgnoresUserConfig(t *testing.T) {
	in := loadCaptureInputs(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[core]\n\tabbrev = 12\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := diffRepo(t, in)
	index := func() string {
		raw, err := tool.RunGit(context.Background(), repo, tool.DiffArgs(false, "")...)
		if err != nil {
			t.Fatalf("git diff: %v", err)
		}
		return regexp.MustCompile(`(?m)^index \S+`).FindString(raw)
	}
	if got := index(); !regexp.MustCompile(`^index [0-9a-f]{12}\.\.[0-9a-f]{12}$`).MatchString(got) {
		t.Fatalf("control: with the user's core.abbrev = 12 the index row is %q, want 12-character hashes", got)
	}
	isolateGit(t)
	if got := index(); !regexp.MustCompile(`^index [0-9a-f]{7}\.\.[0-9a-f]{7}$`).MatchString(got) {
		t.Errorf("after isolateGit the index row is %q, want 7-character hashes", got)
	}
}

// approvalProblems names each field of the messages a model-driven apply_patch call sent
// that differs from the pin, and a completion whose Elapsed is not above zero. The pin
// holds no Elapsed value: the live path times the call, so only that it is set is
// checked. got must be these messages in order: ToolStartedMsg, ApprovalRequestedMsg,
// ToolCompletedMsg.
func approvalProblems(p patchApprovalPin, got []any) []string {
	var out []string
	if len(got) != 3 {
		return []string{fmt.Sprintf("the call sent %d messages, want ToolStartedMsg, ApprovalRequestedMsg and ToolCompletedMsg: %#v", len(got), got)}
	}
	eq := func(what string, got, want any) {
		if !reflect.DeepEqual(got, want) {
			out = append(out, fmt.Sprintf("%s is %#v, the pin says %#v", what, got, want))
		}
	}
	start, ok := got[0].(tui.ToolStartedMsg)
	if !ok {
		return []string{fmt.Sprintf("the first message is a %T, want ToolStartedMsg", got[0])}
	}
	eq("started ID", start.ID, p.Started.ID)
	eq("started name", start.Name, p.Started.Name)
	eq("started target", start.Target, p.Started.Target)
	ask, ok := got[1].(tui.ApprovalRequestedMsg)
	if !ok {
		return append(out, fmt.Sprintf("the second message is a %T, want ApprovalRequestedMsg", got[1]))
	}
	r := p.Requested
	eq("requested ID", ask.ID, r.ID)
	eq("requested kind", ask.Kind, r.Kind)
	eq("requested subject", ask.Subject, r.Subject)
	eq("requested description", ask.Description, r.Description)
	eq("requested detail", ask.Detail, r.Detail)
	eq("requested CanApproveForSession", ask.CanApproveForSession, r.CanApproveForSession)
	eq("requested ToolID", ask.ToolID, r.ToolID)
	eq("requested DiffFilename", ask.DiffFilename, r.DiffFilename)
	eq("requested DiffLines", ask.DiffLines, r.DiffLines)
	eq("requested DiffAdded", ask.DiffAdded, r.DiffAdded)
	eq("requested DiffRemoved", ask.DiffRemoved, r.DiffRemoved)
	done, ok := got[2].(tui.ToolCompletedMsg)
	if !ok {
		return append(out, fmt.Sprintf("the third message is a %T, want ToolCompletedMsg", got[2]))
	}
	eq("completed ID", done.ID, p.Started.ID)
	eq("completed OK", done.OK, p.Completed.OK)
	eq("completed summary", done.Summary, p.Completed.Summary)
	eq("completed error kind", done.ErrorKind, p.Completed.ErrorKind)
	eq("completed error message", done.ErrorMsg, p.Completed.ErrorMsg)
	if done.Elapsed <= 0 {
		out = append(out, fmt.Sprintf("completed Elapsed is %v, want above zero: the live path times every tool call", done.Elapsed))
	}
	return out
}

// TestCapturePatchApprovalMatchesProducers: entry point Submit, through the agent loop
// and a stream that the real anthropic.Decode reads, with an apply_patch call whose
// input is the pin's. The app's outbound messages are recorded through sendFn; the
// approval is answered with ApprovalOutcomeDeny through App.Resolve from inside that
// hook, which is the user pressing n. The tool messages must match patchApprovalPin
// field by field: the ToolStartedMsg with no target, the ApprovalRequestedMsg whose
// ToolID is the card's, and the ToolCompletedMsg; no ToolTargetMsg is sent, because the
// call's input names no target; the ToolCompletedMsg has Elapsed above zero. Controls:
// a copy of the pin with the subject changed, one with the diff rows changed, one with
// another ToolID and one with the detail's closing parenthesis dropped each report a
// problem, as does a copy of the messages whose completion has Elapsed zero, beside the
// real pin and messages, which report none.
func TestCapturePatchApprovalMatchesProducers(t *testing.T) {
	in := loadCaptureInputs(t)
	pin := in.PatchApproval
	repo := newGitRepo(t, map[string]string{"hello.go": "package main\n"})
	input, err := json.Marshal(map[string]string{"diff": pin.Input.Diff, "description": pin.Input.Description})
	if err != nil {
		t.Fatal(err)
	}
	e := newLiveEnv(t, repo, func(a *App, msg any) {
		if ask, ok := msg.(tui.ApprovalRequestedMsg); ok {
			go a.Resolve(ask.ID, ApprovalOutcomeDeny)
		}
	}, sseToolUse(liveCallID, pin.Started.Name, string(input)), sseReply("done"))
	msgs := toolMsgs(e.turn("patch"))
	if p := approvalProblems(pin, msgs); len(p) != 0 {
		t.Errorf("the messages the live apply_patch call sent differ from capture-inputs.json: %v", p)
	}
	if _, err := os.Stat(filepath.Join(repo, pin.Requested.DiffFilename)); err == nil {
		t.Errorf("%s exists: the rejected patch was applied", pin.Requested.DiffFilename)
	}

	wrongSubject := pin
	wrongSubject.Requested.Subject = "2 files changed"
	if len(approvalProblems(wrongSubject, msgs)) == 0 {
		t.Error("control: a pin with another subject reported no problem")
	}
	wrongRows := pin
	wrongRows.Requested.DiffLines = append([]string(nil), pin.Requested.DiffLines...)
	wrongRows.Requested.DiffLines[1] = "+package other"
	if len(approvalProblems(wrongRows, msgs)) == 0 {
		t.Error("control: a pin with another diff row reported no problem")
	}
	wrongLink := pin
	wrongLink.Requested.ToolID = 0
	if len(approvalProblems(wrongLink, msgs)) == 0 {
		t.Error("control: a pin whose approval request has no tool ID reported no problem")
	}
	openParen := pin
	openParen.Requested.Detail = []string{strings.TrimSuffix(pin.Requested.Detail[0], ")")}
	if len(approvalProblems(openParen, msgs)) == 0 {
		t.Error("control: a pin whose detail has no closing parenthesis reported no problem")
	}
	zeroed := append([]any(nil), msgs...)
	if done, ok := zeroed[2].(tui.ToolCompletedMsg); ok {
		done.Elapsed = 0
		zeroed[2] = done
	} else {
		t.Fatalf("setup: message at index 2 is a %T, want ToolCompletedMsg", zeroed[2])
	}
	if len(approvalProblems(pin, zeroed)) == 0 {
		t.Error("control: a completion with Elapsed zero reported no problem")
	}
}

// TestCaptureResumeMessagesMatchProducers: entry point App.ResumeLastSession, which
// cmd/kirsch/main.go calls at startup, over a session log written with session.Store.
// The types of the messages it sends to the tui, in order, must equal the pin's
// resumeMessages, which the chain replay's header names. Control: the pin with one
// message dropped, and the pin in another order, are not equal to the sent types.
func TestCaptureResumeMessagesMatchProducers(t *testing.T) {
	in := loadCaptureInputs(t)
	a, c := resumeApp(t, t.TempDir())
	writeUsageLog(t, a, resumeUsageEvents, nil)
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}
	sessionUsage(t, c, 1) // waits for the last message and for any that trail it
	var sent []string
	for _, m := range c.snapshot() {
		sent = append(sent, fmt.Sprintf("%T", m))
	}
	if !reflect.DeepEqual(sent, in.ResumeMessages) {
		t.Errorf("ResumeLastSession sent %v, capture-inputs.json says %v", sent, in.ResumeMessages)
	}
	if len(in.ResumeMessages) < 2 {
		t.Fatalf("control: the pin lists %d messages", len(in.ResumeMessages))
	}
	if reflect.DeepEqual(sent, in.ResumeMessages[1:]) {
		t.Error("control: the sent types equal the pin with its first message dropped")
	}
	swapped := append([]string(nil), in.ResumeMessages...)
	swapped[0], swapped[1] = swapped[1], swapped[0]
	if reflect.DeepEqual(sent, swapped) {
		t.Error("control: the sent types equal the pin in another order")
	}
}
