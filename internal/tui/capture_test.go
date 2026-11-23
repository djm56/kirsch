package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/clipboard"
)

// Screen captures for the manual walkthroughs (DIR-021).
//
// A walkthrough that tells the operator what a screen looks like has to quote a
// screen the program drew. Each row of captureTable builds one model state with
// canned transcript items and real key messages through Update, the way the
// golden scenarios do: no provider, no network, no model key. Its frame is
// written to plan/testing/manual/captures/<release>/<name>.txt, and the test
// compares every file against the current render so a capture cannot go stale
// silently.
//
// Later releases add rows to captureTable with their own Release string. The file
// locations, the write and compare modes and the inventory check all follow from
// the rows. A new key name for capKeys is the one other edit: it goes in
// captureKeyNames, and capKeys fails on a multi-character name that is not there.
//
// Inputs are the producers' own formats. The read_file card is fed the summary and
// the numbered content that internal/tool's read_file returns, /diff is fed the text
// git prints for the changed files, and the patch approval is fed the messages a
// model-driven apply_patch call sends. internal/tui may not import internal/tool or
// internal/app, so the literals and patterns that describe those formats live in one
// file, testdata/capture-inputs.json, which capIn reads on first use.
// TestCaptureInputFormats holds the canned inputs to it, and the tests in
// internal/app/capture_drift_test.go hold the real read_file tool, a real git diff,
// the real apply_patch call, the real resume and the committed capture files to the
// same file.
//
// Besides the table rows, captureExtras lists files that a test builds itself, such
// as the chain replay (chain_replay_test.go). They are written, compared and
// inventory-checked exactly like the rows' files. A release folder may also hold a
// NOT-CAPTURED.md, written by hand, that names the walkthrough cues no frame can
// show; the inventory test does not count it as a stray file.
//
// A row sets what its walkthrough section assumes: Markdown (render_markdown), Env (the
// environment colour and glyphs are detected from, as NO_COLOR=1 sets it) and Canned
// (the reply text a (model) section made up). Each is recorded in the file's header.
//
// Modes. KIRSCH_CAPTURE_WRITE=1 (re)writes the files. Without it the test fails,
// with the differing rows, when a file is missing or differs. Write mode never
// deletes: a stray file is reported by TestCaptureInventory and removed by hand.
//
// Which frame is captured. The .txt file is View() from a colour-off model, so
// it is the exact text a terminal shows and reads without escapes. A state that
// sets Annotate is built a second time with colour on and also gets a
// <name>.annot.txt listing the colour roles; colour-off and colour-on frames are
// asserted to carry the same text (TestCaptureStyledStripsToPlain), so the two
// files describe one frame. The .annot.txt header ends with the line
// "=== annotations ===" and holds no frame; the frame is in the .txt file.
//
// Annotation form. After a short header and a legend, one line per cell run:
//
//	row 07 cols 12-15 fg=Success(38;5;120) "+foo"
//
// Rows count from 0 at the top of the frame and columns from 0 at its left edge,
// margin included. A run is consecutive cells on one row with the same attributes.
// An attribute is fg=<role>(<SGR>), bg=<role>(<SGR>) or bold, italic, strike, with
// the SGR parameters as the program emitted them. The role name is looked up from
// the SGR number alone, in capturePalette, which holds one name per number. Only the
// roles listed there are ever named: a role that shares a number with a listed one is
// reported under the listed name (the modal border, BorderFocus, uses 117 and is
// reported as Accent). An SGR number outside capturePalette fails the test.
// Annotate "bar" lists, for every row whose last cell is in the last column, that
// cell alone, marking one with no attributes "plain"; a row that stops short of the
// last column has no line. Annotate "emph" lists every run except plain runs and
// runs whose only attribute is a foreground of Text, Muted, Dim or Border: the
// diff colours, the syntax roles, the backgrounds, bold and the rest. Annotate "roles"
// lists every run that has any attribute, those quiet runs included.

const capturesRoot = "../../plan/testing/manual/captures"

// capInputsPath is the file of pinned inputs and patterns, shared with the tests in
// internal/app. It sits in testdata so no build includes it.
const capInputsPath = "testdata/capture-inputs.json"

// capInputsFile is the shape of capInputsPath. Options.Version is the version the
// captured frames show in the header. It is fixed in the file, not taken from
// fixtureVersion or the release, so a version bump leaves the capture files alone.
type capInputsFile struct {
	About   string `json:"_about"`
	Options struct {
		Version, Project, Branch, Model, Family string
		Dirty                                   bool
	}
	FrameSep string
	// ToolElapsedMs is the Elapsed, in milliseconds, of every tool completion the
	// captures send (capStampElapsed).
	ToolElapsedMs int64
	// ApprovalElapsedMs is how far the capture clock moves on each read
	// (capClock), which is the duration a resolved approval line shows.
	ApprovalElapsedMs int64
	ReadFile          struct {
		Text, Summary, Content, OneLineText, OneLineSummary string
	}
	ModalReadFile struct{ Path, Text string }
	LiveReadFile  struct {
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
	Diff struct {
		Path, Old, New string
		Lines          []string
		Added, Removed int
		FuncContextMax int
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
	PatchApproval struct {
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
	ResumeMessages []string
	Committed      []struct {
		File         string
		MustMatch    []string
		MustNotMatch []string
	}
}

// readCapInputs parses the file at path. A key that capInputsFile has no field for
// is an error (DisallowUnknownFields), so a pinned input that no test reads cannot
// sit in the file unnoticed.
func readCapInputs(path string) (capInputsFile, error) {
	var in capInputsFile
	raw, err := os.ReadFile(path)
	if err != nil {
		return in, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return in, err
	}
	return in, nil
}

// capInputsLoad holds capInputsPath as capIn first read it. The file is read on the
// first call, not when the test binary starts, so a file that is missing or does not
// parse fails the test that needs it with t.Fatalf instead of stopping the binary.
var capInputsLoad struct {
	once sync.Once
	in   capInputsFile
	err  error
}

// capIn is the shared inputs file, read once. It stops the calling test when the
// file cannot be read or parsed.
func capIn(t testing.TB) capInputsFile {
	t.Helper()
	capInputsLoad.once.Do(func() { capInputsLoad.in, capInputsLoad.err = readCapInputs(capInputsPath) })
	if capInputsLoad.err != nil {
		t.Fatalf("capture inputs %s: %v", capInputsPath, capInputsLoad.err)
	}
	return capInputsLoad.in
}

// capOptions is the Options every captured model is built from, with colour and
// glyphs chosen by the caller. It reads the shared inputs file, so the tests in
// internal/app that build the same model read the same values.
func capOptions(t testing.TB, colour, unicode bool) Options {
	t.Helper()
	o := capIn(t).Options
	return Options{
		Now:     capClock(time.Duration(capIn(t).ApprovalElapsedMs) * time.Millisecond),
		Version: o.Version,
		Caps:    Caps{Colour: colour, Unicode: unicode},
		Session: SessionInfo{Project: o.Project, Branch: o.Branch, Dirty: o.Dirty},
		Status:  Status{Model: o.Model, Family: o.Family},
	}
}

// capClock returns a clock for Options.Now that moves forward by step on every
// read, from a fixed start, so no capture reads the wall clock. The model reads
// the clock when an approval is requested and again when it is resolved, with no
// read between them in any capture, so the resolved approval line shows exactly
// step. No capture presses Ctrl+C (captureKeyNames has no such key), the one other
// reader of the clock; two such reads would be step apart.
func capClock(step time.Duration) func() time.Time {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(step)
		return now
	}
}

// captureAnnotSep ends an .annot.txt file's header: the run listing follows its line.
// The matching line of a .txt file, which the frame follows, is the shared file's
// frameSep, read through capIn, because the tests in internal/app read it from the
// same file.
const captureAnnotSep = "=== annotations ===\n"

// captureStep is one input to the model. Msgs builds the messages from the
// current model, because some inputs depend on it (the /diff result carries the
// sequence number the model issued). It takes the test so an input read from the
// shared inputs file can stop the test that needs it.
type captureStep struct {
	Name string // what the walkthrough quotes: a key, or a canned arrival
	Msgs func(t testing.TB, m Model) []tea.Msg
}

// captureState is one row of the table.
type captureState struct {
	Release  string // the folder under captures/, e.g. "v0.1.5"
	Name     string // the file name stem, and the name the walkthrough cites
	W, H     int
	Unicode  bool
	Annotate string // "", "bar" or "emph"; see the annotation form above

	// Markdown is "on" or "off" to set Options.RenderMarkdown, which the header then
	// records; "" leaves the option unset (replies drawn plain) and the header silent.
	Markdown string

	// Env, when not nil, is the environment the state's colour and glyph set are
	// detected from: the model's Caps come from DetectCaps under it, and the header
	// records it. The colour argument of captureModel and the Unicode field are then
	// not used.
	Env map[string]string

	// Canned, when not empty, is a header line naming the reply text the file made up
	// for a walkthrough section that is marked (model).
	Canned string

	Setup func(t testing.TB, m *Model)
	Steps []captureStep
}

var captureKeyNames = map[string]tea.KeyType{
	"Shift+↑": tea.KeyShiftUp, "PgUp": tea.KeyPgUp, "PgDn": tea.KeyPgDown,
	"Home": tea.KeyHome, "End": tea.KeyEnd, "Esc": tea.KeyEsc,
	"Enter": tea.KeyEnter, "Tab": tea.KeyTab, "↑": tea.KeyUp, "↓": tea.KeyDown,
}

// capKeyMsg is the message for one key name: a name from captureKeyNames, or a
// single character. Any other name is an error, so a misspelt key cannot type its
// first letter.
func capKeyMsg(n string) (tea.Msg, error) {
	if kt, ok := captureKeyNames[n]; ok {
		return keyType(kt), nil
	}
	if r := []rune(n); len(r) == 1 {
		return key(r[0]), nil
	}
	return nil, fmt.Errorf("unknown key name %q: not in captureKeyNames and not a single character", n)
}

// capKeys builds a step that presses the named keys in order, by capKeyMsg. A name
// capKeyMsg rejects panics when the step runs, which fails the test that built the
// model.
func capKeys(names ...string) captureStep {
	return captureStep{Name: strings.Join(names, " "), Msgs: func(testing.TB, Model) []tea.Msg {
		var out []tea.Msg
		for _, n := range names {
			msg, err := capKeyMsg(n)
			if err != nil {
				panic("capKeys: " + err.Error())
			}
			out = append(out, msg)
		}
		return out
	}}
}

// capKeysCopying is capKeys for a key whose Update returns the copy command (Enter
// after /copy, and c on a reply). After each key it runs that command, which is a
// closure over the model's Copy and returns a CopyResultMsg without blocking, and
// delivers the result after the key, the order the program's runtime delivers them.
// A key whose Update returns no command adds nothing. A command that yields any
// other message fails the test: this step does not run arbitrary commands, because
// a tea.Tick command sleeps.
func capKeysCopying(names ...string) captureStep {
	return captureStep{Name: strings.Join(names, " ") + " (the copy result arrives)", Msgs: func(t testing.TB, m Model) []tea.Msg {
		t.Helper()
		var out []tea.Msg
		cur := m
		for _, n := range names {
			msg, err := capKeyMsg(n)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, msg)
			next, cmd := cur.Update(msg)
			cur = next.(Model)
			if cmd == nil {
				continue
			}
			res := cmd()
			if _, ok := res.(CopyResultMsg); !ok {
				t.Fatalf("key %q returned a command that yields a %T, want a CopyResultMsg", n, res)
			}
			out = append(out, res)
			next, _ = cur.Update(res)
			cur = next.(Model)
		}
		return out
	}}
}

// capStampElapsed returns msgs with the Elapsed of every ToolCompletedMsg set to
// the one canned duration in the inputs file (toolElapsedMs); every other message
// is returned as it is. The live path sets Elapsed on every model-driven tool
// completion, so a card head draws a duration wherever the layout sets
// ShowDuration, and the captures draw that canned one, not a measured one. The
// state builder and the chain replay stamp every message they deliver through
// this function.
func capStampElapsed(t testing.TB, msgs []tea.Msg) []tea.Msg {
	t.Helper()
	d := time.Duration(capIn(t).ToolElapsedMs) * time.Millisecond
	out := make([]tea.Msg, len(msgs))
	for i, m := range msgs {
		if done, ok := m.(ToolCompletedMsg); ok {
			done.Elapsed = d
			m = done
		}
		out[i] = m
	}
	return out
}

// capMsgs builds a step that delivers canned messages, as the app would.
func capMsgs(name string, msgs ...tea.Msg) captureStep {
	return captureStep{Name: name, Msgs: func(testing.TB, Model) []tea.Msg { return msgs }}
}

// capText is a step that types text into the composer, one key per character.
func capText(text string) captureStep {
	return captureStep{Name: text, Msgs: func(testing.TB, Model) []tea.Msg {
		var out []tea.Msg
		for _, r := range text {
			out = append(out, key(r))
		}
		return out
	}}
}

// capChat is a user prompt and its canned reply: the typed text, Enter, then the
// reply's text delta and TurnCompleteMsg, which is the order a live turn sends them
// in. A live turn sends more: a BudgetMsg before the reply and a UsageMsg and a
// SessionUsageMsg after it (TestLiveTurnSendsTheMessagesInThisOrder in internal/app
// lists the order for a turn with a tool call), and the capture leaves those out.
// block is the reply's BlockID, which must differ between the replies of one model.
func capChat(block int64, prompt, reply string) []captureStep {
	return []captureStep{
		capText(prompt), capKeys("Enter"),
		capMsgs("(the reply streams, the turn completes)",
			AssistantTextDeltaMsg{BlockID: block, Delta: reply}, TurnCompleteMsg{}),
	}
}

// capTurn is a canned user message and a reply, loaded the way a resumed session
// loads them: the assistant block carries Lines and the unsanitised Source, as the
// resume loader in internal/app builds it.
func capTurn(user string, reply ...string) tea.Msg {
	text := strings.Join(reply, "\n")
	return TranscriptLoadedMsg{Items: []Item{
		{Kind: KindUser, Text: &TextBlock{Lines: []string{user}}},
		{Kind: KindAssistant, Text: &TextBlock{Lines: SanitizeLines(text), Source: text}},
	}}
}

// capResumeTurn is a resumed session that holds one user message and one reply: the
// notice and then the transcript, which App.ResumeLastSession sends first. The
// SessionUsageMsg it sends after them is not sent here.
func capResumeTurn(user, reply string) captureStep {
	return capMsgs("(the session resumes: a notice, then the transcript)",
		NoticeMsg{Text: "session resumed"}, capTurn(user, reply))
}

// capReadFile is a finished read_file call as a live session sends it: the
// messages internal/app sends for a model-driven call whose provider call ID is
// known. adaptTurnEvent sends the ToolStartedMsg when the call starts, and the
// Anthropic start event carries the call's ID and name and no input, so the
// message has no target; it sends the ToolTargetMsg when the call ends, which is
// when the complete input is known; toolExecutor.Invoke, finding the call's ID,
// sends no second start and sends the ToolCompletedMsg with the target, the tool's
// DisplaySummary and its Content. The content is the file's text numbered
// "%d\t%s\n" per line and the summary is "lines 1-N", each as
// internal/tool/readfile.go writes them for a whole file under its line and byte
// caps. TestCaptureInputFormats pins this output to the shared inputs file
// (liveReadFile), and TestCaptureLiveReadFileMatchesProducers in
// internal/app/capture_drift_test.go holds the real call to the same file.
func capReadFile(target, text string) []tea.Msg { return capReadFileID(1, target, text) }

// capReadFileID is capReadFile for a call with the given tool-call ID, so a state
// that finishes several read_file calls keeps their IDs apart.
func capReadFileID(id int64, target, text string) []tea.Msg {
	var numbered strings.Builder
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, line := range lines {
		fmt.Fprintf(&numbered, "%d\t%s\n", i+1, line)
	}
	return []tea.Msg{
		ToolStartedMsg{ID: id, Name: "read_file"},
		ToolTargetMsg{ID: id, Target: target},
		ToolCompletedMsg{
			ID: id, Name: "read_file", Target: target, OK: true,
			Summary: fmt.Sprintf("lines 1-%d", len(lines)), Content: numbered.String(),
		},
	}
}

// capReplyLines is n short reply rows, ASCII only so the ASCII state shows glyphs
// from the bar and not from the text.
func capReplyLines(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("Step %d: the zero check in Divide runs after the division.", i+1)
	}
	return out
}

const capDivideFile = "package calc\n\nimport \"errors\"\n\n" +
	"// Divide returns a/b, or an error when b is zero.\n" +
	"func Divide(a, b float64) (float64, error) {\n" +
	"\tif b == 0 {\n\t\treturn 0, errors.New(\"division by zero\")\n\t}\n" +
	"\treturn a / b, nil\n}\n"

// capOverflow loads a transcript longer than any viewport in the table and ends
// with a read_file card, the card the modal states open with `d`.
func capOverflow() []captureStep {
	return []captureStep{
		capMsgs("(canned transcript loads)",
			capTurn("Why does Divide panic on zero?", capReplyLines(14)...)),
		capMsgs("(canned second turn loads)",
			capTurn("And where is it called from?", capReplyLines(14)...)),
		capMsgs("(a read_file card finishes)", capReadFile("calc/divide.go", capDivideFile)...),
	}
}

func appendSteps(base []captureStep, more ...captureStep) []captureStep {
	return append(append([]captureStep(nil), base...), more...)
}

// capDiff is the canned /diff result: the lines in the shared file's diff.lines, which
// are git's own output for diff.old to diff.new (a hunk header that wraps, a long
// removed line and a long added line, with git's default context either side). The
// result passes through SanitizeLines as GetDiff's does in internal/app.
// TestCaptureInputFormats and the tests in internal/app hold the lines to git.
func capDiff(t testing.TB) DiffResult {
	t.Helper()
	d := capIn(t).Diff
	return DiffResult{
		Lines: SanitizeLines(strings.Join(d.Lines, "\n")),
		Added: d.Added, Removed: d.Removed,
	}
}

// capDiffTwoFiles is the canned /diff result for a modified file and a new one
// registered with git add -N: the shared file's diffTwoFiles.lines, which are git's
// own output for that change, passed through SanitizeLines as GetDiff's are.
func capDiffTwoFiles(t testing.TB) DiffResult {
	t.Helper()
	d := capIn(t).DiffTwoFiles
	return DiffResult{
		Lines: SanitizeLines(strings.Join(d.Lines, "\n")),
		Added: d.Added, Removed: d.Removed,
	}
}

// capDiffSteps types /diff, presses Enter and delivers the result, as the app does
// for a /diff command: the DiffResultMsg carries the sequence number the model
// issued.
func capDiffSteps(result func(t testing.TB) DiffResult) []captureStep {
	return []captureStep{
		capText("/diff"), capKeys("Enter"),
		{Name: "(the git diff result arrives)", Msgs: func(t testing.TB, m Model) []tea.Msg {
			return []tea.Msg{DiffResultMsg{Result: result(t), Seq: m.pendingDiffSeq}}
		}},
	}
}

// capCopyRoute is the copy route the copy captures wire into Model.Copy: it reports
// that the local tool took the text, so the hint reads "copied N characters" with N
// the code points handed over. It writes nowhere; no clipboard is touched.
func capCopyRoute(text string) clipboard.Result {
	return clipboard.Result{Chars: utf8.RuneCountInString(text), Local: true}
}

// capStatusInfo is the canned input of the /status command: the values the notice
// card shows. internal/app computes the real ones from config and environment. The
// dirty count follows the header's dirty marker (Options.Dirty in the shared inputs
// file): one uncommitted change when the marker is drawn, none when it is not, so the
// header and the notice never disagree in one frame.
func capStatusInfo(t testing.TB) StatusInfoMsg {
	t.Helper()
	dirty := 0
	if capIn(t).Options.Dirty {
		dirty = 1
	}
	return StatusInfoMsg{
		Endpoint: "anthropic", BaseURLHost: "api.anthropic.com", KeySource: "ANTHROPIC_API_KEY",
		SessionID: "none", CompactionPercent: -1, DirtyCount: dirty,
	}
}

// capCopySetup is the Setup of the copy captures: the canned copy route and the
// canned /status input.
func capCopySetup(t testing.TB, m *Model) {
	m.Copy = capCopyRoute
	m.StatusInfo = func() StatusInfoMsg { return capStatusInfo(t) }
}

// capCopyReplies is the setup of the v0.1.3 walkthrough, which "uses the model":
// prompts, each with a canned one-sentence reply.
func capCopyReplies() []captureStep {
	return appendSteps(
		capChat(1, "say hello in one sentence", capHelloReply),
		capChat(2, "say goodbye in one sentence", capGoodbyeReply)...)
}

// The canned one-sentence replies of the v0.1.3 setup.
const (
	capHelloReply   = "Hello, it is good to meet you."
	capGoodbyeReply = "Goodbye, and take care."
)

// capTwoBlockReply is a canned reply of fenced Go code blocks with a sentence of plain
// text between them, for the /copy code sections.
const capTwoBlockReply = "```go\nfunc One() int { return 1 }\n```\n\nThe second function returns two.\n\n```go\nfunc Two() int { return 2 }\n```"

// capMarkdownReply is a canned reply that uses each markdown feature the v0.1.4
// walkthrough names: a heading, bold, inline code, strike-through, bullets and a go
// fence.
const capMarkdownReply = "# Heading check\n\nA paragraph with **bold words**, `inline code` and ~~struck words~~.\n\n- first item\n- second item\n\n```go\nfunc f() int { return 1 }\n```"

// capMarkdownPrompt is the prompt the v0.1.4 walkthrough types in section 1, which
// capMarkdownReply answers.
const capMarkdownPrompt = "Reply with only this markdown: a heading `# Heading check`; a blank line; a paragraph with **bold words**, `inline code` and ~~struck words~~; a blank line; two bullets, `- first item` and `- second item`; a blank line; a go code fence containing `func f() int { return 1 }`. Do not wrap the whole reply in a code fence."

// capPatchApprovalMsgs is what a model-driven apply_patch call sends the tui up to
// the approval card: the tool starting and the approval request, each as the shared
// file pins them from the live path (a decoded Anthropic stream, the agent loop,
// adaptTurnEvent, toolExecutor.Invoke and App.Request). The start message has no
// target, the approval request carries the card's tool ID, and no ToolTargetMsg is
// sent because the call's input names no target.
func capPatchApprovalMsgs(t testing.TB) []tea.Msg {
	t.Helper()
	p := capIn(t).PatchApproval
	return []tea.Msg{
		ToolStartedMsg{ID: p.Started.ID, Name: p.Started.Name, Target: p.Started.Target},
		ApprovalRequestedMsg{
			ID: p.Requested.ID, ToolID: p.Requested.ToolID, Description: p.Requested.Description,
			Kind: p.Requested.Kind, CanApproveForSession: p.Requested.CanApproveForSession,
			Subject: p.Requested.Subject, Detail: p.Requested.Detail,
			DiffFilename: p.Requested.DiffFilename, DiffLines: p.Requested.DiffLines,
			DiffAdded: p.Requested.DiffAdded, DiffRemoved: p.Requested.DiffRemoved,
		},
	}
}

// capPatchRejectedMsg is the ToolCompletedMsg the rejected apply_patch call ends in,
// as the shared file pins it from the real producer.
func capPatchRejectedMsg(t testing.TB) tea.Msg {
	t.Helper()
	p := capIn(t).PatchApproval
	return ToolCompletedMsg{
		ID: p.Started.ID, Name: p.Started.Name, Target: p.Started.Target, OK: p.Completed.OK,
		Summary: p.Completed.Summary, ErrorKind: p.Completed.ErrorKind, ErrorMsg: p.Completed.ErrorMsg,
	}
}

// capResolveApproval is the canned ResolveApproval the approval captures wire in: the
// key handler calls it on y, a and n, and the real one answers the blocked tool.
func capResolveApproval(int64, ApprovalOutcome) {}

// captureTable is every state a manual walkthrough cites. Steps name the keys the
// operator presses; the walkthrough quotes the Name column.
var captureTable = []captureState{
	{
		Release: "v0.1.5", Name: "scrollbar-bottom", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: capOverflow(),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-top", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: appendSteps(capOverflow(), capKeys("Shift+↑"), capKeys("g")),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-mid-new", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: appendSteps(capOverflow(), capKeys("Shift+↑"), capKeys("g"), capKeys("PgDn"),
			capMsgs("(notices arrive)", NoticeMsg{Text: "one"}, NoticeMsg{Text: "two"}, NoticeMsg{Text: "three"})),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-hidden-short", W: 80, H: 24, Unicode: true,
		Steps: []captureStep{
			capMsgs("(canned short transcript loads)", capTurn("Hello", "Hi there.")),
			capMsgs("(a read_file card finishes)", capReadFile("calc/divide.go", "package calc\n")...),
		},
	},
	{
		Release: "v0.1.5", Name: "scrollbar-hidden-modal", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: appendSteps(capOverflow(), capKeys("Shift+↑"), capKeys("d")),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-ascii", W: 80, H: 24, Unicode: false, Annotate: "bar",
		Steps: capOverflow(),
	},
	{
		Release: "v0.1.5", Name: "modal-readfile-wrapped", W: 80, H: 24, Unicode: true, Annotate: "emph",
		Steps: []captureStep{
			{Name: "(a read_file card finishes)", Msgs: func(t testing.TB, _ Model) []tea.Msg {
				in := capIn(t).ModalReadFile
				return capReadFile(in.Path, in.Text)
			}},
			capKeys("Shift+↑"), capKeys("d"),
		},
	},
	{
		Release: "v0.1.5", Name: "modal-diff-wrapped", W: 80, H: 32, Unicode: true, Annotate: "emph",
		Setup: func(t testing.TB, m *Model) { m.GetDiff = func() DiffResult { return capDiff(t) } },
		Steps: capDiffSteps(capDiff),
	},
	{
		Release: "v0.1.5", Name: "modal-after-G", W: 80, H: 24, Unicode: true,
		Steps: []captureStep{
			capMsgs("(a read_file card finishes)",
				capReadFile("calc/long.txt", strings.Repeat("line of the file\n", 60))...),
			capKeys("Shift+↑"), capKeys("d"), capKeys("G"),
		},
	},
	{
		Release: "v0.1.5", Name: "help-cut", W: 60, H: 24, Unicode: true,
		Steps: []captureStep{capKeys("Shift+↑"), capKeys("?")},
	},

	// v0.1.3, section 1: arrows stop on replies and skip the user's messages.
	copyRow("s01-arrows-last-reply", 80, 24, "emph", keySteps("Shift+↑")...),
	copyRow("s01-arrows-first-reply", 80, 24, "emph", keySteps("Shift+↑", "↑")...),
	copyRow("s01-arrows-top-stays", 80, 24, "emph", keySteps("Shift+↑", "↑", "↑")...),
	copyRow("s01-arrows-composer", 80, 24, "emph", keySteps("Shift+↑", "↑", "↑", "↓", "↓")...),
	// Section 2: Enter and d do nothing on a reply.
	copyRow("s02-enter-d-noop", 80, 24, "", keySteps("Shift+↑", "Enter", "d")...),
	// Section 3: c copies a reply; the paste into another app is not captured.
	copyRow("s03-copy-reply", 80, 24, "", capKeys("Shift+↑"), capKeysCopying("c")),
	// Section 4: c on a card gives the hint; the notice card is the /status command's.
	copyRow("s04-copy-card-hint", 80, 24, "emph", appendSteps(replySelected(), capText("/status"), capKeys("Enter"),
		capKeys("Shift+↑"), capKeys("↓"), capKeys("c"))...),
	// Section 5: /copy.
	copyRow("s05-copy-command", 80, 24, "", capText("/copy"), capKeysCopying("Enter")),
	// Sections 6 and 7: /copy code on capTwoBlockReply.
	copyRow("s06-copy-code", 80, 24, "", appendSteps(capChat(3, capTwoBlockPrompt, capTwoBlockReply),
		capText("/copy code"), capKeysCopying("Enter"))...),
	copyRow("s07-copy-code-range", 80, 24, "", appendSteps(capChat(3, capTwoBlockPrompt, capTwoBlockReply),
		capText("/copy code 9"), capKeysCopying("Enter"))...),
	// Section 8: /copy foo.
	copyRow("s08-copy-usage", 80, 24, "", capText("/copy foo"), capKeys("Enter")),
	// Section 9: the copy notice clears when a key is typed.
	copyRow("s09-copy-notice-clears", 80, 24, "", capText("/copy"), capKeysCopying("Enter"), capText("x")),
	// Section 10: /c and /co do not complete, /cop does.
	copyRow("s10-c-tab-x", 80, 24, "", capText("/c"), capKeys("Tab"), capText("x")),
	copyRow("s10-co-tab", 80, 24, "", capText("/co"), capKeys("Tab")),
	copyRow("s10-cop-tab", 80, 24, "", capText("/cop"), capKeys("Tab")),
	// Section 11: the help overlay at 80 columns by 28 rows, with the status notice
	// card selected as the walkthrough leaves it.
	copyRow("s11-help-open", 80, 28, "", appendSteps(noticeSelected(), capKeys("Shift+↑"), capKeys("?"))...),
	copyRow("s11-help-closed-hint", 80, 28, "", appendSteps(noticeSelected(),
		capKeys("Shift+↑"), capKeys("?"), capKeys("?"), capKeys("c"))...),

	// v0.1.4, section 1: a markdown reply.
	withCanned(capCannedReply, markdownRow("s01-markdown-reply", 80, 24, "on", nil, capChat(1, capMarkdownPrompt, capMarkdownReply)...)),
	// Section 2: render_markdown = false, on a resumed session.
	withCanned(capCannedResumed, markdownRow("s02-markdown-off", 80, 24, "off", nil, capResumeTurn(capMarkdownPrompt, capMarkdownReply))),
	// Section 3: the read_file modal highlights a Go file.
	annotated("roles", withCanned(capCannedFile, markdownRow("s03-readfile-modal-go", 80, 24, "on", nil,
		capText("Use read_file on hello.go, then say done."), capKeys("Enter"),
		captureStep{Name: "(the read_file call for hello.go finishes)", Msgs: func(t testing.TB, _ Model) []tea.Msg {
			return capReadFile("hello.go", capIn(t).DiffTwoFiles.Files[0].Old)
		}},
		capMsgs("(the reply streams, the turn completes)",
			AssistantTextDeltaMsg{BlockID: 1, Delta: "Done."}, TurnCompleteMsg{}),
		capKeys("Shift+↑"), capKeys("d")))),
	// Section 4: a patch approval, its diff and the rejection.
	withCanned(capCannedPatch, markdownRow("s04-approval-card", 80, 24, "on", nil, patchApprovalSteps()...)),
	withCanned(capCannedPatch, markdownRow("s04-approval-diff", 80, 32, "on", nil, appendSteps(patchApprovalSteps(), capKeys("d"))...)),
	withCanned(capCannedPatch, markdownRow("s04-approval-rejected", 80, 24, "on", nil, appendSteps(patchApprovalSteps(),
		capKeys("n"),
		captureStep{Name: "(the apply_patch call ends: patch rejected)", Msgs: func(t testing.TB, _ Model) []tea.Msg {
			return []tea.Msg{capPatchRejectedMsg(t)}
		}})...)),
	// Section 5: /diff over a modified Go file and a new Python file.
	withCanned(capCannedDiff, markdownRow("s05-diff-two-files", 80, 32, "on", nil, capDiffSteps(capDiffTwoFiles)...)),
	// Section 6: NO_COLOR=1.
	withCanned(capCannedDiff, markdownRow("s06-no-color-diff", 80, 32, "on", noColourEnv, capDiffSteps(capDiffTwoFiles)...)),
	withCanned(capCannedResumed, markdownRow("s06-no-color-reply", 80, 24, "on", noColourEnv,
		capResumeTurn(capMarkdownPrompt, capMarkdownReply), capKeys("Shift+↑"))),
	// Section 7: a postgresql fence and a sql fence, each highlighted: the
	// postgresql info string is drawn with the SQL lexer (hlInfoAlias).
	withCanned(capCannedReply, markdownRow("s07-fence-languages", 80, 24, "on", nil, capChat(1, capFencePrompt, capFenceReply)...)),
}

// Environments the walkthroughs set, for the states whose colour and glyphs are
// detected from them. TERM and LC_ALL are set so DetectCaps does not read the
// environment the test runs in: a colour terminal and a UTF-8 locale, with NO_COLOR
// as the walkthrough's section 6 sets it.
var noColourEnv = map[string]string{"NO_COLOR": "1", "TERM": "xterm-256color", "LC_ALL": "en_US.UTF-8"}

// colourEnv is noColourEnv with NO_COLOR empty, which DetectCaps reads as unset. It
// is the control for the NO_COLOR states, not a captured state.
var colourEnv = map[string]string{"NO_COLOR": "", "TERM": "xterm-256color", "LC_ALL": "en_US.UTF-8"}

// Canned header lines. capCannedCopy is the v0.1.3 rows'. Each v0.1.4 row names only
// what its own state cans (withCanned): a state whose frame holds no patch does not
// say it cans one.
const (
	capCannedCopy = "the replies' text (the walkthrough sections that use the model write their own); " +
		"where a state uses them, the clipboard route, which reports success and writes nowhere, and the /status notice's field values"
	capCannedReply   = "the reply text (a walkthrough section that uses the model writes its own)"
	capCannedResumed = "the reply text of the resumed session (the walkthrough's own session holds the reply its model gave)"
	capCannedFile    = "the reply text (a walkthrough section that uses the model writes its own) and the text of hello.go, which testdata/capture-inputs.json holds"
	capCannedPatch   = "the patch the model proposes and its description (a walkthrough section that uses the model writes its own), as testdata/capture-inputs.json holds them"
	capCannedDiff    = "the changed files' text, which testdata/capture-inputs.json holds, and git's diff of them"
)

// withCanned is cs with its Canned header line set.
func withCanned(canned string, cs captureState) captureState {
	cs.Canned = canned
	return cs
}

// capTwoBlockPrompt is the prompt the v0.1.3 walkthrough types in section 6.
const capTwoBlockPrompt = "Write two short Go functions. Put each in its own fenced code block with the language tag go. Put one sentence of plain text between the two blocks."

// capFencePrompt and capFenceReply are the v0.1.4 walkthrough's section 7 prompt and
// a canned answer to it.
const (
	capFencePrompt = "Reply with only two code fences. The first is tagged postgresql and the second is tagged sql. Each holds the line SELECT 1;"
	capFenceReply  = "```postgresql\nSELECT 1;\n```\n\n```sql\nSELECT 1;\n```"
)

// annotated is cs with its Annotate mode set.
func annotated(mode string, cs captureState) captureState {
	cs.Annotate = mode
	return cs
}

// keySteps is one capKeys step per name, so a header lists the keys one by one.
func keySteps(names ...string) []captureStep {
	out := make([]captureStep, len(names))
	for i, n := range names {
		out[i] = capKeys(n)
	}
	return out
}

// copyRow is a row of the v0.1.3 walkthrough: the canned replies of its setup, the
// copy and /status wiring, then steps. Every v0.1.3 row has the same options.
func copyRow(name string, w, h int, annotate string, steps ...captureStep) captureState {
	return captureState{
		Release: "v0.1.3", Name: name, W: w, H: h, Unicode: true, Annotate: annotate,
		Markdown: "on", Canned: capCannedCopy, Setup: capCopySetup,
		Steps: appendSteps(capCopyReplies(), steps...),
	}
}

// replySelected leaves the v0.1.3 transcript with the second reply selected and the
// composer focused, which is where the walkthrough's section 1 leaves it. Shift+↑ with
// nothing selected selects the last item, so a state that reaches Browsing without this
// first selects a different item from the walkthrough's.
func replySelected() []captureStep {
	return []captureStep{capKeys("Shift+↑"), capKeys("Tab")}
}

// noticeSelected leaves the v0.1.3 transcript with the /status notice card selected and
// the composer focused, as the walkthrough's section 11 starts.
func noticeSelected() []captureStep {
	return appendSteps(replySelected(),
		capText("/status"), capKeys("Enter"), capKeys("Shift+↑"), capKeys("↓"), capKeys("Tab"))
}

// markdownRow is a row of the v0.1.4 walkthrough, with render_markdown as markdown names
// it and, when env is not nil, the environment the colour is detected from. It sets no
// Canned line: the caller wraps it in withCanned with what the state cans.
func markdownRow(name string, w, h int, markdown string, env map[string]string, steps ...captureStep) captureState {
	return captureState{
		Release: "v0.1.4", Name: name, W: w, H: h, Unicode: true, Annotate: "emph",
		Markdown: markdown, Env: env,
		Setup: func(t testing.TB, m *Model) {
			m.GetDiff = func() DiffResult { return capDiffTwoFiles(t) }
			m.ResolveApproval = capResolveApproval
		},
		Steps: steps,
	}
}

// patchApprovalSteps is v0.1.4 section 4 up to the approval card: the prompt, the
// tool call starting and the approval request, which the shared file pins from the
// real producer.
func patchApprovalSteps() []captureStep {
	return []captureStep{
		capText("Add a new file notes.go with package main and a function func Hello() string that returns \"hi\". Propose it with apply_patch and wait for my approval."),
		capKeys("Enter"),
		{Name: "(the apply_patch call starts and asks for approval)", Msgs: func(t testing.TB, _ Model) []tea.Msg {
			return capPatchApprovalMsgs(t)
		}},
	}
}

// captureModel builds a state's model with colour on or off and returns it after
// every step. A state with an Env takes its Caps from DetectCaps under that
// environment instead; the colour argument is then not used.
func captureModel(t *testing.T, cs captureState, colour bool) Model {
	t.Helper()
	o := capOptions(t, colour, cs.Unicode)
	if cs.Env != nil {
		for k, v := range cs.Env {
			t.Setenv(k, v)
		}
		o.Caps = DetectCaps(nil)
	}
	switch cs.Markdown {
	case "on":
		o.RenderMarkdown = true
	case "off":
		o.RenderMarkdown = false
	case "":
	default:
		t.Fatalf("state %s: Markdown %q, want on, off or empty", cs.Name, cs.Markdown)
	}
	m := New(o)
	if cs.Setup != nil {
		cs.Setup(t, &m)
	}
	m = drive(m, tea.WindowSizeMsg{Width: cs.W, Height: cs.H})
	for _, st := range cs.Steps {
		m = drive(m, capStampElapsed(t, st.Msgs(t, m))...)
	}
	return m
}

// captureHeader is the file header: the state, the size, the glyph set and the
// inputs, then sep, the separator line that ends the header. A state that sets
// Env, Markdown or Canned has a line for each after the steps.
func captureHeader(t testing.TB, cs captureState, kind, sep string) string {
	t.Helper()
	glyphs := "unicode"
	if !cs.Unicode {
		glyphs = "ascii"
	}
	var steps []string
	for _, st := range cs.Steps {
		steps = append(steps, st.Name)
	}
	var extra strings.Builder
	if cs.Env != nil {
		keys := make([]string, 0, len(cs.Env))
		for k := range cs.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var pairs []string
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%q", k, cs.Env[k]))
		}
		fmt.Fprintf(&extra, "# env (colour and glyphs are detected from it): %s\n", strings.Join(pairs, " "))
	}
	if cs.Markdown != "" {
		fmt.Fprintf(&extra, "# render_markdown: %s\n", cs.Markdown)
	}
	if cs.Canned != "" {
		fmt.Fprintf(&extra, "# canned: %s\n", cs.Canned)
	}
	return fmt.Sprintf("# state: %s\n# release: %s\n# size: %dx%d\n# glyphs: %s\n# steps: %s\n%s# %s\n# no model, no network, no key: canned messages through the real Model\n%s",
		cs.Name, cs.Release, cs.W, cs.H, glyphs, strings.Join(steps, " | "), extra.String(), kind, sep)
}

// capturePlain is the text file's content for a state.
func capturePlain(t *testing.T, cs captureState) string {
	t.Helper()
	m := captureModel(t, cs, false)
	return captureHeader(t, cs, "frame text: View() with colour off", capIn(t).FrameSep) + m.View() + "\n"
}

// capturePalette maps the foreground and background SGR parameters to the role
// names of the palette comments in styles.go.
var capturePalette = map[string]string{
	"38;5;253": "Text", "38;5;248": "Muted", "38;5;245": "Dim", "38;5;117": "Accent",
	"38;5;120": "Success", "38;5;203": "Error", "38;5;215": "Warning", "38;5;123": "Hunk",
	"38;5;244": "Border", "38;5;141": "SynKeyword", "38;5;180": "SynString",
	"38;5;216": "SynNumber", "38;5;152": "SynType",
	"48;5;235": "CodeBg", "48;5;236": "SelectionBg",
}

// capRun is a run of cells on one row sharing one attribute set.
type capRun struct {
	Row, From, To int
	Attr          string // "" for no attributes
	Quiet         bool   // the only attribute is a foreground of Text, Muted, Dim or Border
	Text          string
}

// captureRuns is captureRunsErr that fails the test on an error.
func captureRuns(t *testing.T, row int, s string) []capRun {
	t.Helper()
	runs, err := captureRunsErr(row, s)
	if err != nil {
		t.Fatal(err)
	}
	return runs
}

// captureRunsErr parses one styled row into runs. It returns an error for an
// escape it does not handle and for a foreground or background colour that is not
// in capturePalette, since a colour without a role would print an empty role name.
// The attribute groups it handles are the ones the program emits: reset, bold,
// italic, strike, their off codes, default foreground, and 256-colour fg and bg.
func captureRunsErr(row int, s string) ([]capRun, error) {
	var fg, bg string
	var bold, italic, strike bool
	var runs []capRun
	col := 0
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "\x1b[") {
			end := strings.IndexByte(s[i:], 'm')
			if end < 0 {
				return nil, fmt.Errorf("row %d: unterminated escape", row)
			}
			switch p := s[i+2 : i+end]; {
			case p == "0":
				fg, bg, bold, italic, strike = "", "", false, false, false
			case p == "1":
				bold = true
			case p == "3":
				italic = true
			case p == "9":
				strike = true
			case p == "22":
				bold = false
			case p == "23":
				italic = false
			case p == "29":
				strike = false
			case p == "39":
				fg = ""
			case strings.HasPrefix(p, "38;5;"):
				fg = p
			case strings.HasPrefix(p, "48;5;"):
				bg = p
			default:
				return nil, fmt.Errorf("row %d: unhandled SGR %q", row, p)
			}
			i += end + 1
			continue
		}
		r := []rune(s[i:])[0]
		i += len(string(r))
		w := cellWidth(string(r))
		var attrs []string
		if fg != "" {
			if _, ok := capturePalette[fg]; !ok {
				return nil, fmt.Errorf("row %d col %d: foreground SGR %q is not in capturePalette", row, col, fg)
			}
			attrs = append(attrs, fmt.Sprintf("fg=%s(%s)", capturePalette[fg], fg))
		}
		if bg != "" {
			if _, ok := capturePalette[bg]; !ok {
				return nil, fmt.Errorf("row %d col %d: background SGR %q is not in capturePalette", row, col, bg)
			}
			attrs = append(attrs, fmt.Sprintf("bg=%s(%s)", capturePalette[bg], bg))
		}
		for _, f := range []struct {
			on   bool
			name string
		}{{bold, "bold"}, {italic, "italic"}, {strike, "strike"}} {
			if f.on {
				attrs = append(attrs, f.name)
			}
		}
		attr := strings.Join(attrs, " ")
		quiet := bg == "" && !bold && !italic && !strike && fg != "" &&
			map[string]bool{"Text": true, "Muted": true, "Dim": true, "Border": true}[capturePalette[fg]]
		if n := len(runs); n > 0 && runs[n-1].Attr == attr && runs[n-1].To == col-1 {
			runs[n-1].To = col + w - 1
			runs[n-1].Text += string(r)
		} else {
			runs = append(runs, capRun{Row: row, From: col, To: col + w - 1, Attr: attr, Quiet: quiet, Text: string(r)})
		}
		col += w
	}
	return runs, nil
}

// captureAnnot is the annotated file's content for a state: the colour-on frame
// described by its runs, or for a state with an Env the frame that env gives.
func captureAnnot(t *testing.T, cs captureState) string {
	t.Helper()
	m := captureModel(t, cs, true)
	var b strings.Builder
	which := "the colour-on frame"
	if cs.Env != nil {
		which = "the frame under the env above"
	}
	b.WriteString(captureHeader(t, cs, "annotations: colour roles of "+which+" ("+cs.Annotate+")", captureAnnotSep))
	b.WriteString("legend: row/cols count from 0 at the top-left of the frame; fg/bg=<role>(<SGR>)\n")
	listed := 0
	for r, row := range strings.Split(m.View(), "\n") {
		for _, run := range captureRuns(t, r, row) {
			switch cs.Annotate {
			case "bar":
				if run.To != cs.W-1 {
					// a run reaching the last column is the bar cell; a run ending
					// earlier is a row with no cell there.
					continue
				}
				attr := run.Attr
				if attr == "" {
					attr = "plain"
				}
				listed++
				fmt.Fprintf(&b, "row %02d col %d %s %q\n", r, cs.W-1, attr, run.Text[len(run.Text)-lastRuneLen(run.Text):])
			case "emph", "roles":
				// roles lists the quiet runs too, so a cue about a Dim or Muted foreground
				// has a line to point at; emph leaves them out.
				if run.Attr == "" || (cs.Annotate == "emph" && run.Quiet) {
					continue
				}
				listed++
				fmt.Fprintf(&b, "row %02d cols %d-%d %s %q\n", r, run.From, run.To, run.Attr, run.Text)
			}
		}
	}
	if cs.Annotate == "bar" && listed == 0 {
		fmt.Fprintf(&b, "no row of the frame has a cell in column %d\n", cs.W-1)
	}
	if cs.Annotate == "emph" && listed == 0 {
		b.WriteString("no run of the frame has an attribute other than a Text, Muted, Dim or Border foreground\n")
	}
	if cs.Annotate == "roles" && listed == 0 {
		b.WriteString("no run of the frame has any attribute\n")
	}
	return b.String()
}

// lastRuneLen is the byte length of the final rune of s.
func lastRuneLen(s string) int {
	r := []rune(s)
	return len(string(r[len(r)-1]))
}

// captureDir is the folder a release's files live in.
func captureDir(release string) string {
	return filepath.Join(filepath.FromSlash(capturesRoot), release)
}

// captureFiles is the file name to content for one state.
func captureFiles(t *testing.T, cs captureState) map[string]string {
	t.Helper()
	out := map[string]string{cs.Name + ".txt": capturePlain(t, cs)}
	if cs.Annotate != "" {
		out[cs.Name+".annot.txt"] = captureAnnot(t, cs)
	}
	return out
}

// diffText names the rows where got differs from want, one - and + pair each, or
// the row counts when they differ in length. It returns "" when the two are equal.
func diffText(want, got string) string {
	if want == got {
		return ""
	}
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		wok, gok := i < len(w), i < len(g)
		if wok {
			wl = w[i]
		}
		if gok {
			gl = g[i]
		}
		if wok && gok && wl == gl {
			continue
		}
		if wok {
			fmt.Fprintf(&b, "line %d - %q\n", i+1, wl)
		}
		if gok {
			fmt.Fprintf(&b, "line %d + %q\n", i+1, gl)
		}
	}
	return b.String()
}

// checkCaptureFile compares the file at path with got and returns the
// differences, or the read error, or "" when they match. It is the one compare
// the test uses, and the control test calls it on an edited copy.
func checkCaptureFile(path, got string) string {
	want, err := os.ReadFile(path)
	if err != nil {
		return "cannot read: " + err.Error()
	}
	return diffText(string(want), got)
}

// captureExtra is a file under a release folder that a test builds itself, as the
// chain replay does, instead of a captureTable row building it from a model.
type captureExtra struct {
	Release string
	File    string
	Build   func(t *testing.T) string
}

// captureExtras is every file under a release folder that no captureTable row
// produces.
var captureExtras = []captureExtra{
	{Release: "v0.1.5", File: "chain-replay.txt", Build: chainReplay},
}

// writeOrCheckCapture writes got to path in write mode, creating the folder, and
// otherwise compares the file with got and reports any difference on t.
func writeOrCheckCapture(t *testing.T, write bool, path, got string) {
	t.Helper()
	if write {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if d := checkCaptureFile(path, got); d != "" {
		t.Errorf("%s differs from the render (regenerate with KIRSCH_CAPTURE_WRITE=1 go test -count=1 -run TestCapture ./internal/tui):\n%s", path, d)
	}
}

// TestCaptureFiles is the capture itself. With KIRSCH_CAPTURE_WRITE=1 it writes
// every state's files and every captureExtras file; without it, it fails for each
// file that is missing or differs from the current render. Entry point: View() on
// the model each row builds (captureFiles), and each extra's Build.
func TestCaptureFiles(t *testing.T) {
	write := os.Getenv("KIRSCH_CAPTURE_WRITE") == "1"
	for _, cs := range captureTable {
		cs := cs
		t.Run(cs.Release+"/"+cs.Name, func(t *testing.T) {
			for name, got := range captureFiles(t, cs) {
				writeOrCheckCapture(t, write, filepath.Join(captureDir(cs.Release), name), got)
			}
		})
	}
	for _, ex := range captureExtras {
		ex := ex
		t.Run(ex.Release+"/"+ex.File, func(t *testing.T) {
			writeOrCheckCapture(t, write, filepath.Join(captureDir(ex.Release), ex.File), ex.Build(t))
		})
	}
}

// duplicateRows names the "release/name" pairs that more than one row of table
// carries.
func duplicateRows(table []captureState) []string {
	seen := map[string]bool{}
	var dup []string
	for _, cs := range table {
		id := cs.Release + "/" + cs.Name
		if seen[id] {
			dup = append(dup, id)
		}
		seen[id] = true
	}
	return dup
}

// foldersWithoutRows names the folders directly under root that are not in
// releases, the set of release names the table has rows for.
func foldersWithoutRows(root string, releases map[string]bool) ([]string, error) {
	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, d := range dirs {
		if d.IsDir() && !releases[d.Name()] {
			out = append(out, d.Name())
		}
	}
	return out, nil
}

// TestCaptureInventory checks the table and captureExtras against the folders: every
// file a row or an extra produces exists, every file under a release folder
// (dotfiles and NOT-CAPTURED.md aside) belongs to a row or an extra, no two rows
// share a name, and every release folder has rows. A stray, a duplicate or a missing
// file fails.
func TestCaptureInventory(t *testing.T) {
	want := map[string][]string{}
	for _, id := range duplicateRows(captureTable) {
		t.Errorf("table has two rows named %s", id)
	}
	for _, cs := range captureTable {
		for name := range captureFiles(t, cs) {
			want[cs.Release] = append(want[cs.Release], name)
		}
	}
	for _, ex := range captureExtras {
		for _, name := range want[ex.Release] {
			if name == ex.File {
				t.Errorf("%s/%s is produced by a table row and by captureExtras", ex.Release, ex.File)
			}
		}
		want[ex.Release] = append(want[ex.Release], ex.File)
	}
	for release, names := range want {
		missing, stray := inventoryGap(captureDir(release), names)
		for _, n := range missing {
			t.Errorf("table row has no file: %s/%s", release, n)
		}
		for _, n := range stray {
			t.Errorf("file belongs to no table row: %s/%s", release, n)
		}
	}
	releases := map[string]bool{}
	for r := range want {
		releases[r] = true
	}
	empty, err := foldersWithoutRows(filepath.FromSlash(capturesRoot), releases)
	if err != nil {
		t.Fatalf("read captures folder: %v", err)
	}
	for _, d := range empty {
		t.Errorf("folder %s has no rows in captureTable", d)
	}
}

// TestCaptureInventoryDetectsStrayAndMissing is the control for the inventory
// test: the same comparison, on a temporary folder, fails for a stray file and for
// a missing one, passes when the sets match, ignores a dotfile (.DS_Store) and the
// NOT-CAPTURED.md file, flags
// a duplicate row and flags a folder with no rows, each beside a case that passes.
func TestCaptureInventoryDetectsStrayAndMissing(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.txt", "b.txt", ".DS_Store", notCapturedFile} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if m, s := inventoryGap(dir, []string{"a.txt", "b.txt"}); len(m)+len(s) != 0 {
		t.Fatalf("control: matching sets reported missing %v stray %v", m, s)
	}
	if m, s := inventoryGap(dir, []string{"a.txt", "b.txt", "c.txt"}); len(m) != 1 || len(s) != 0 {
		t.Errorf("a missing file: missing %v stray %v, want one missing", m, s)
	}
	if m, s := inventoryGap(dir, []string{"a.txt"}); len(m) != 0 || len(s) != 1 {
		t.Errorf("a stray file: missing %v stray %v, want one stray", m, s)
	}
	// .DS_Store and NOT-CAPTURED.md are in dir in every call above and in no want
	// list. The matching-sets call reports nothing and the stray call reports b.txt
	// alone, so the dotfile and NOT-CAPTURED.md are ignored while another stray is
	// still found.

	rows := []captureState{{Release: "v1", Name: "a"}, {Release: "v1", Name: "b"}, {Release: "v2", Name: "a"}}
	if d := duplicateRows(rows); len(d) != 0 {
		t.Errorf("control: rows with distinct release/name reported duplicates %v", d)
	}
	if d := duplicateRows(append(rows, captureState{Release: "v1", Name: "a"})); len(d) != 1 || d[0] != "v1/a" {
		t.Errorf("a repeated release/name: duplicates %v, want [v1/a]", d)
	}

	root := t.TempDir()
	for _, n := range []string{"v1", "v9"} {
		if err := os.Mkdir(filepath.Join(root, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if f, err := foldersWithoutRows(root, map[string]bool{"v1": true, "v9": true}); err != nil || len(f) != 0 {
		t.Errorf("control: folders %v err %v with rows for every folder, want none", f, err)
	}
	if f, err := foldersWithoutRows(root, map[string]bool{"v1": true}); err != nil || len(f) != 1 || f[0] != "v9" {
		t.Errorf("a folder with no rows: %v err %v, want [v9]", f, err)
	}
}

// notCapturedFile is the hand-written file in a release folder that names the
// walkthrough cues no frame can show. It is the one file there that no row or extra
// produces.
const notCapturedFile = "NOT-CAPTURED.md"

// inventoryGap compares the file names in dir with want and returns the missing
// names and the stray ones. A name starting with "." is never stray (.DS_Store and
// the like), nor is notCapturedFile, and every other file in dir that is not in want is.
func inventoryGap(dir string, want []string) (missing, stray []string) {
	files, _ := os.ReadDir(dir)
	have := map[string]bool{}
	for _, f := range files {
		have[f.Name()] = true
	}
	w := map[string]bool{}
	for _, n := range want {
		w[n] = true
		if !have[n] {
			missing = append(missing, n)
		}
	}
	for n := range have {
		if !w[n] && !strings.HasPrefix(n, ".") && n != notCapturedFile {
			stray = append(stray, n)
		}
	}
	return missing, stray
}

// compareControlStates names one state per release that has files: the control below
// runs on each one's .txt and .annot.txt file, so a new release's captures are
// covered by the compare check as the v0.1.5 ones are.
var compareControlStates = []struct{ release, name string }{
	{"v0.1.5", "scrollbar-bottom"},
	{"v0.1.3", "s01-arrows-first-reply"},
	{"v0.1.4", "s01-markdown-reply"},
}

// TestCaptureCompareFailsOnOneCellEdit is the control for the compare mode, run on
// the .txt file and the .annot.txt file of one state from each release in
// compareControlStates. On a temporary copy of a committed capture, an unedited copy
// passes; a copy with one cell added after the header separator fails and the report
// names that line as one - and one + row; a copy with one trailing space appended
// fails. The committed files are never touched.
func TestCaptureCompareFailsOnOneCellEdit(t *testing.T) {
	for _, st := range compareControlStates {
		cs := capState(t, st.name)
		if cs.Release != st.release {
			t.Fatalf("state %s is in release %s, the control lists it under %s", st.name, cs.Release, st.release)
		}
		for _, c := range []struct{ file, sep string }{
			{cs.Name + ".txt", capIn(t).FrameSep},
			{cs.Name + ".annot.txt", captureAnnotSep},
		} {
			t.Run(cs.Release+"/"+c.file, func(t *testing.T) {
				got := captureFiles(t, cs)[c.file]
				committed, err := os.ReadFile(filepath.Join(captureDir(cs.Release), c.file))
				if err != nil {
					t.Fatalf("read committed capture: %v", err)
				}
				tmp := filepath.Join(t.TempDir(), c.file)
				check := func(content string) string {
					if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
					return checkCaptureFile(tmp, got)
				}
				if d := check(string(committed)); d != "" {
					t.Fatalf("control: an unedited copy differs from the render:\n%s", d)
				}

				edited := strings.Replace(string(committed), c.sep, c.sep+"X", 1)
				if edited == string(committed) {
					t.Fatal("setup: the edit changed nothing")
				}
				d := check(edited)
				if d == "" {
					t.Fatal("a copy with one cell added after the header separator passed the compare")
				}
				if strings.Count(d, "\n") != 2 {
					t.Errorf("the report is not one - and one + line:\n%s", d)
				}

				// A trailing space on the last row: the frames carry trailing spaces on
				// purpose, so the compare must not trim.
				if d := check(strings.TrimSuffix(string(committed), "\n") + " \n"); d == "" {
					t.Error("a copy with one trailing space added to the last row passed the compare")
				}
			})
		}
	}
}

// TestCaptureKeyNames is the control for capKeys: a name in captureKeyNames and a
// single character give a message, an unknown name longer than one character gives
// an error and does not fall back to its first letter, and running a step built
// with such a name panics.
func TestCaptureKeyNames(t *testing.T) {
	for _, n := range []string{"PgDn", "Esc", "g", "d"} {
		if msg, err := capKeyMsg(n); err != nil || msg == nil {
			t.Errorf("control: capKeyMsg(%q) = %v, %v, want a message", n, msg, err)
		}
	}
	for _, n := range []string{"PageDown", "Shift+Up", "Entr", ""} {
		if msg, err := capKeyMsg(n); err == nil {
			t.Errorf("capKeyMsg(%q) = %v, want an error", n, msg)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("running capKeys(\"PageDown\") did not panic")
			}
		}()
		capKeys("PageDown").Msgs(t, Model{})
	}()
	if n := len(capKeys("PgDn", "g").Msgs(t, Model{})); n != 2 {
		t.Errorf("control: capKeys(PgDn, g) built %d messages, want 2", n)
	}
}

// TestCaptureRunsRejectUnknownColour is the control for captureRunsErr: a
// foreground and a background that are in capturePalette parse, one that is not
// fails for each of fg and bg, and an unknown parameter group still fails.
func TestCaptureRunsRejectUnknownColour(t *testing.T) {
	if runs, err := captureRunsErr(0, "\x1b[38;5;117m\x1b[48;5;235mx\x1b[0m"); err != nil || len(runs) != 1 ||
		runs[0].Attr != "fg=Accent(38;5;117) bg=CodeBg(48;5;235)" {
		t.Fatalf("control: palette colours gave %+v, %v", runs, err)
	}
	for _, row := range []string{"\x1b[38;5;99mx\x1b[0m", "\x1b[48;5;99mx\x1b[0m", "\x1b[7mx\x1b[0m"} {
		if runs, err := captureRunsErr(0, row); err == nil {
			t.Errorf("captureRunsErr(%q) = %+v, want an error", row, runs)
		}
	}
}

// capHunkHeader matches a unified-diff hunk header and captures the old and new
// line counts and the function context after the closing @@.
var capHunkHeader = regexp.MustCompile(`^@@ -\d+,(\d+) \+\d+,(\d+) @@(?: (.*))?$`)

// hunkCountsErr checks every hunk header in lines against the rows under it: the
// old count is the context rows plus the removed rows, the new count is the context
// rows plus the added rows. It returns nil for a diff with no hunk header, so the
// caller must show a hunk is present.
func hunkCountsErr(lines []string) error {
	for i := 0; i < len(lines); i++ {
		mm := capHunkHeader.FindStringSubmatch(lines[i])
		if mm == nil {
			continue
		}
		var oldN, newN int
		for _, l := range lines[i+1:] {
			if strings.HasPrefix(l, "@@") || strings.HasPrefix(l, "diff --git ") {
				break
			}
			switch {
			case strings.HasPrefix(l, "-"):
				oldN++
			case strings.HasPrefix(l, "+"):
				newN++
			default:
				oldN++
				newN++
			}
		}
		if mm[1] != fmt.Sprint(oldN) || mm[2] != fmt.Sprint(newN) {
			return fmt.Errorf("hunk header %q counts -%s +%s, its rows count -%d +%d", lines[i], mm[1], mm[2], oldN, newN)
		}
	}
	return nil
}

// funcContextErr checks the function context of every hunk header in lines, the
// text after the closing @@, against git's cut at max bytes.
func funcContextErr(lines []string, max int) error {
	for _, l := range lines {
		if mm := capHunkHeader.FindStringSubmatch(l); mm != nil && len(mm[3]) > max {
			return fmt.Errorf("hunk header function context is %d bytes, git cuts it at %d: %q", len(mm[3]), max, l)
		}
	}
	return nil
}

// TestCaptureStampsToolCompletionsWithTheCannedElapsed: entry point capStampElapsed,
// called directly on the messages capReadFile builds, and then the same messages
// drawn by a model at 80 columns, where a card head shows its duration. The
// completion must come back with Elapsed equal to toolElapsedMs of the inputs
// file, which must be above zero because a zero Elapsed draws no duration; the
// start and target messages must come back unchanged; and the messages passed in
// must keep their own Elapsed of zero, so the function stamps a copy. The frame
// of the stamped messages must draw the duration on the read_file head. Control,
// through the same render: the frame of the unstamped messages draws no duration
// on that head.
func TestCaptureStampsToolCompletionsWithTheCannedElapsed(t *testing.T) {
	want := time.Duration(capIn(t).ToolElapsedMs) * time.Millisecond
	if want <= 0 {
		t.Fatalf("toolElapsedMs is %d: a zero Elapsed draws no duration on a card head", capIn(t).ToolElapsedMs)
	}
	in := capReadFile("calc/divide.go", capDivideFile)
	out := capStampElapsed(t, in)
	if len(out) != len(in) {
		t.Fatalf("capStampElapsed returned %d messages for %d", len(out), len(in))
	}
	if done, ok := out[2].(ToolCompletedMsg); !ok || done.Elapsed != want {
		t.Errorf("stamped completion is %#v, want a ToolCompletedMsg with Elapsed %v", out[2], want)
	}
	if !reflect.DeepEqual(out[:2], in[:2]) {
		t.Errorf("capStampElapsed changed a message that is not a completion: %#v, was %#v", out[:2], in[:2])
	}
	if done := in[2].(ToolCompletedMsg); done.Elapsed != 0 {
		t.Errorf("the messages passed in were changed: Elapsed is %v, want 0", done.Elapsed)
	}

	head := func(msgs []tea.Msg) string {
		m := drive(New(capOptions(t, false, true)), tea.WindowSizeMsg{Width: 80, Height: 24})
		for _, line := range strings.Split(drive(m, msgs...).View(), "\n") {
			if strings.Contains(line, "read_file calc/divide.go") {
				return line
			}
		}
		t.Fatalf("no read_file head in the frame of %#v", msgs)
		return ""
	}
	if h := head(out); !strings.Contains(h, " · "+formatDuration(want)+" · ") {
		t.Errorf("the stamped card head %q does not draw the canned duration %s", h, formatDuration(want))
	}
	if h := head(in); strings.Contains(h, formatDuration(want)) {
		t.Errorf("control: the unstamped card head %q draws the canned duration", h)
	}
}

// TestCaptureInputFormats pins the canned inputs to the producers' formats, read
// from the shared inputs file: the read_file summary and numbered content, the
// messages of a live read_file call, and the /diff text.
// Entry points: capReadFile and capDiff, called directly; the real producers are
// held to the same file by internal/app/capture_drift_test.go. The live read_file
// messages capReadFile builds for the pinned path and text must equal liveReadFile
// field by field, and the ToolStartedMsg must carry no target, as the live start
// event carries no input. Controls: a one-line file, whose summary is the range
// "lines 1-1", not a bare count; a copy of the pinned messages with the targeted
// message's target changed is reported as a difference, beside the real messages,
// which are not; the old canned hunk header, @@ -12,7 +12,15 @@, put over the pinned
// rows is refused by hunkCountsErr and a function context one byte over the cut is
// refused by funcContextErr, each beside the real diff, which they accept.
func TestCaptureInputFormats(t *testing.T) {
	rf := capIn(t).ReadFile
	msgs := capReadFile("x.go", rf.Text)
	if len(msgs) != 3 {
		t.Fatalf("capReadFile sent %d messages, want ToolStartedMsg, ToolTargetMsg and ToolCompletedMsg: %#v", len(msgs), msgs)
	}
	done, ok := msgs[2].(ToolCompletedMsg)
	if !ok {
		t.Fatalf("capReadFile's third message is %T, want ToolCompletedMsg", msgs[2])
	}
	if done.Summary != rf.Summary || done.Content != rf.Content {
		t.Errorf("capReadFile summary %q content %q, want %q and %q (testdata/capture-inputs.json)", done.Summary, done.Content, rf.Summary, rf.Content)
	}
	one := capReadFile("x.go", rf.OneLineText)[2].(ToolCompletedMsg)
	if one.Summary != rf.OneLineSummary {
		t.Errorf("control: one-line file summary %q, want %q", one.Summary, rf.OneLineSummary)
	}

	live := capIn(t).LiveReadFile
	mr := capIn(t).ModalReadFile
	want := []tea.Msg{
		ToolStartedMsg{ID: live.Started.ID, Name: live.Started.Name, Target: live.Started.Target},
		ToolTargetMsg{ID: live.Targeted.ID, Target: live.Targeted.Target},
		ToolCompletedMsg{
			ID: live.Completed.ID, Name: live.Completed.Name, Target: live.Completed.Target,
			OK: live.Completed.OK, Summary: live.Completed.Summary, Content: live.Completed.Content,
		},
	}
	if live.Started.Target != "" {
		t.Errorf("liveReadFile.started.target is %q: the Anthropic start event carries no input, so the start message has no target", live.Started.Target)
	}
	if !reflect.DeepEqual(capReadFile(mr.Path, mr.Text), want) {
		t.Errorf("capReadFile(%q, modalReadFile.text) differs from liveReadFile:\n got  %#v\n want %#v", mr.Path, capReadFile(mr.Path, mr.Text), want)
	}
	wrong := append([]tea.Msg(nil), want...)
	wrong[1] = ToolTargetMsg{ID: live.Targeted.ID, Target: "other/" + live.Targeted.Target}
	if reflect.DeepEqual(capReadFile(mr.Path, mr.Text), wrong) {
		t.Error("control: capReadFile equals a copy of the pin whose targeted message names another file")
	}

	d := capIn(t).Diff
	shown := capDiff(t)
	// SanitizeLines, which GetDiff applies, expands tabs, so only the header rows
	// are byte-equal to git's; every row keeps its place.
	if len(shown.Lines) != len(d.Lines) {
		t.Fatalf("capDiff has %d rows, the pinned git output %d", len(shown.Lines), len(d.Lines))
	}
	if got, want := strings.Join(shown.Lines[:len(d.HeaderShapes)], "\n"), strings.Join(d.Lines[:len(d.HeaderShapes)], "\n"); got != want {
		t.Errorf("capDiff header rows differ from the pinned git output:\n%s", diffText(want, got))
	}
	if len(d.Lines) <= len(d.HeaderShapes) {
		t.Fatalf("control: the pinned diff has %d rows, want more than the %d header rows", len(d.Lines), len(d.HeaderShapes))
	}
	for i, shape := range d.HeaderShapes {
		if !regexp.MustCompile(shape).MatchString(d.Lines[i]) {
			t.Errorf("pinned diff row %d is %q, want it to match %s", i, d.Lines[i], shape)
		}
	}
	if err := hunkCountsErr(d.Lines); err != nil {
		t.Errorf("pinned diff: %v", err)
	}
	if err := funcContextErr(d.Lines, d.FuncContextMax); err != nil {
		t.Errorf("pinned diff: %v", err)
	}
	if mm := capHunkHeader.FindStringSubmatch(d.Lines[4]); mm == nil || len(mm[3]) != d.FuncContextMax {
		t.Errorf("pinned hunk header %q: want a function context of exactly %d bytes, the cut git applies to a longer line", d.Lines[4], d.FuncContextMax)
	}

	bad := append([]string(nil), d.Lines...)
	bad[4] = "@@ -12,7 +12,15 @@ func Divide"
	if hunkCountsErr(bad) == nil {
		t.Error("hunkCountsErr accepted a header whose counts disagree with its rows")
	}
	bad[4] = capHunkHeader.FindStringSubmatch(d.Lines[4])[0] + "x"
	if funcContextErr(bad, d.FuncContextMax) == nil {
		t.Error("funcContextErr accepted a function context one byte over the cut")
	}
}

// TestCaptureStyledStripsToPlain: the colour-on frame of every annotated state,
// with its escapes stripped, equals the colour-off frame, so the .annot.txt file
// describes the frame in the .txt file.
func TestCaptureStyledStripsToPlain(t *testing.T) {
	n := 0
	for _, cs := range captureTable {
		if cs.Annotate == "" {
			continue
		}
		n++
		plain := captureModel(t, cs, false).View()
		styled := stripSGR(captureModel(t, cs, true).View())
		if plain != styled {
			t.Errorf("%s: stripped colour frame differs from plain:\n%s", cs.Name, diffText(plain, styled))
		}
	}
	if n == 0 {
		t.Fatal("no annotated state in the table")
	}
}

// capState finds a row by name.
func capState(t *testing.T, name string) captureState {
	t.Helper()
	for _, cs := range captureTable {
		if cs.Name == name {
			return cs
		}
	}
	t.Fatalf("no capture state named %q", name)
	return captureState{}
}

// capHasBar reports whether any of the cells is a bar glyph.
func capHasBar(cells []string) bool {
	for _, c := range cells {
		if c == "│" || c == "┃" || c == "|" || c == "#" {
			return true
		}
	}
	return false
}

// TestCaptureScrollbarStates: entry point View() through captureModel. Each
// scrollbar capture shows what its name says, read from the last column of the
// transcript rows. The bottom capture is the positive control for the modal
// capture, which has the same transcript and the same size and draws a bar when
// no modal is open (scrollbar-hidden-modal adds only the Shift+Up and d keys). The
// short capture has a different, shorter transcript; its control is the same
// short transcript at a height that does not fit it, which does draw a bar.
func TestCaptureScrollbarStates(t *testing.T) {
	bottom := barCells(t, captureModel(t, capState(t, "scrollbar-bottom"), false))
	if !capHasBar(bottom) {
		t.Fatalf("control: the overflowing transcript draws no bar: %q", bottom)
	}
	last := len(bottom) - 1
	if bottom[last] != "┃" || bottom[0] != "│" {
		t.Errorf("pinned at the bottom: first %q last %q, want the track on top and the thumb at the bottom", bottom[0], bottom[last])
	}

	top := barCells(t, captureModel(t, capState(t, "scrollbar-top"), false))
	if top[0] != "┃" || top[len(top)-1] != "│" {
		t.Errorf("scrolled to the top: first %q last %q, want the thumb at the top", top[0], top[len(top)-1])
	}

	midM := captureModel(t, capState(t, "scrollbar-mid-new"), false)
	mid := barCells(t, midM)
	if mid[0] != "│" || mid[len(mid)-1] != "│" && mid[len(mid)-1] != "┃" {
		t.Errorf("scrolled mid-way: first %q last %q", mid[0], mid[len(mid)-1])
	}
	thumb := -1
	for i, c := range mid {
		if c == "┃" {
			thumb = i
		}
	}
	if thumb <= 0 || thumb >= len(mid)-1 {
		t.Errorf("scrolled mid-way: the thumb's last row is %d of %d, want it strictly inside the track: %q", thumb, len(mid), mid)
	}
	lay := midM.layout()
	rows := strings.Split(midM.View(), "\n")
	band := stripSGR(rows[lay.HeaderH()+lay.TranscriptH-1])
	if !strings.HasSuffix(band, "↓ 3 new"+mid[len(mid)-1]) {
		t.Errorf("mid-way: the last transcript row is %q, want it to end \"↓ 3 new\" and then the bar cell", band)
	}

	short := capState(t, "scrollbar-hidden-short")
	if capHasBar(barCells(t, captureModel(t, short, false))) {
		t.Error("the short transcript capture draws a bar")
	}
	tight := short
	tight.H = 9
	if !capHasBar(barCells(t, captureModel(t, tight, false))) {
		t.Error("control: the short transcript at a 9-row height draws no bar, so the hidden-short capture does not show the bar being hidden for lack of overflow")
	}
	hm := captureModel(t, capState(t, "scrollbar-hidden-modal"), false)
	if hm.modal == nil {
		t.Fatal("the hidden-under-modal capture has no modal open")
	}
	if capHasBar(barCells(t, hm)) {
		t.Error("a bar glyph is drawn in the last column under an open modal")
	}

	asc := barCells(t, captureModel(t, capState(t, "scrollbar-ascii"), false))
	var sawPipe, sawHash bool
	for _, c := range asc {
		switch c {
		case "|":
			sawPipe = true
		case "#":
			sawHash = true
		case "│", "┃":
			t.Errorf("the ASCII capture draws the Unicode glyph %q", c)
		}
	}
	if !sawPipe || !sawHash {
		t.Errorf("the ASCII capture: track | seen %v, thumb # seen %v: %q", sawPipe, sawHash, asc)
	}
}

// TestCaptureModalStates: entry point modalBox on the captured models. The read_file
// capture numbers a wrapped line on its first row only; the /diff capture leaves
// the marker column blank on code continuations and wraps the @@ line flush; the
// G capture is the last full page; the help capture cuts lines, never wraps them.
func TestCaptureModalStates(t *testing.T) {
	// read_file: the long line is numbered 3; its continuation rows start with
	// five blanks and then text. The line before it is the control that a one-row
	// line does carry its number.
	rm := captureModel(t, capState(t, "modal-readfile-wrapped"), false)
	rows := bodyText(t, mustBox(rm))
	at := -1
	for i, r := range rows {
		if regexp.MustCompile(`^ {3}3 3 +var usage`).MatchString(r) {
			at = i
		}
	}
	if at < 0 || at+1 >= len(rows) {
		t.Fatalf("read_file capture: no row starts with the number 3 twice and then \"var usage\": %q", rows)
	}
	if !strings.HasPrefix(rows[at-1], "   2 2") {
		t.Errorf("control: the one-row line before the long one is %q, want it numbered 2", rows[at-1])
	}
	if !regexp.MustCompile(`^ {5}\S`).MatchString(rows[at+1]) {
		t.Errorf("the continuation row is %q, want five blank columns then text", rows[at+1])
	}
	if regexp.MustCompile(`^ *\d+ `).MatchString(rows[at+1]) {
		t.Errorf("the continuation row carries a line number: %q", rows[at+1])
	}

	// /diff.
	dm := captureModel(t, capState(t, "modal-diff-wrapped"), false)
	if dm.modal == nil || dm.modal.Kind != ModalDiff {
		t.Fatalf("diff capture: modal %+v, want the diff modal", dm.modal)
	}
	drows := bodyText(t, mustBox(dm))
	// find is the first row that starts with marker and holds word. The code
	// lines hold a tab in the canned diff, which the modal shows as spaces.
	find := func(marker, word string) int {
		for i, r := range drows {
			if strings.HasPrefix(r, marker) && strings.Contains(r, word) {
				return i
			}
		}
		t.Fatalf("diff capture: no row starts %q and holds %q: %q", marker, word, drows)
		return -1
	}
	if i := find("@@ -", "Quotient"); !regexp.MustCompile(`^\S`).MatchString(drows[i+1]) {
		t.Errorf("the @@ continuation is %q, want it flush", drows[i+1])
	}
	for _, c := range []struct{ marker, word string }{{"+", "divideChecked"}, {"-", "legacyDivide"}} {
		i := find(c.marker, c.word)
		if i+1 >= len(drows) || !regexp.MustCompile(`^ \S`).MatchString(drows[i+1]) {
			t.Errorf("the %q line's next row is %q, want a blank marker column then code", c.marker, drows[i+1])
		}
	}

	// G: the last page, full.
	gm := captureModel(t, capState(t, "modal-after-G"), false)
	total, bodyH := gm.modalExtent(gm.layout())
	if total <= bodyH || gm.modal.Off != total-bodyH {
		t.Errorf("after G: offset %d with %d rows in a %d-row body, want the offset at total-body", gm.modal.Off, total, bodyH)
	}
	grows := bodyText(t, mustBox(gm))
	if len(grows) != bodyH || !strings.HasPrefix(grows[len(grows)-1], fmt.Sprintf("%4d ", total)) {
		t.Errorf("after G: %d body rows ending %q, want %d ending numbered %d", len(grows), grows[len(grows)-1], bodyH, total)
	}

	// help: every visible row is a prefix of the help line it shows, and at least
	// one is shorter than its line (cut). A wrapped line would show its tail on the
	// next row, so the rows would not be prefixes of consecutive help lines.
	hm := captureModel(t, capState(t, "help-cut"), false)
	help := helpLines()
	hrows := bodyText(t, mustBox(hm))
	cut := 0
	for i, r := range hrows {
		src := strings.TrimRight(help[hm.modal.Off+i], " ")
		if !strings.HasPrefix(src, strings.TrimSuffix(r, "⋯")) {
			t.Errorf("help row %d is %q, not a prefix of line %q", i, r, src)
		}
		if cellWidth(r) < cellWidth(src) {
			cut++
		}
	}
	if cut == 0 {
		t.Errorf("no help row is shorter than its source line at this size: %q", hrows)
	}
}

// TestCaptureInputsFileLoads: entry point readCapInputs, the reader behind capIn. The
// committed file parses. Controls: a file with a key capInputsFile has no field for is
// refused, so a pinned input no test reads cannot sit in it, and a missing file is
// refused. Each is beside the committed file, which is accepted by the same call.
func TestCaptureInputsFileLoads(t *testing.T) {
	if _, err := readCapInputs(capInputsPath); err != nil {
		t.Fatalf("the committed inputs file: %v", err)
	}
	raw, err := os.ReadFile(capInputsPath)
	if err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(t.TempDir(), "extra.json")
	edited := strings.Replace(string(raw), `"frameSep"`, `"noSuchKey": 1, "frameSep"`, 1)
	if edited == string(raw) {
		t.Fatal("setup: the edit changed nothing")
	}
	if err := os.WriteFile(extra, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readCapInputs(extra); err == nil {
		t.Error("a file with a key no field reads was accepted")
	}
	if _, err := readCapInputs(filepath.Join(t.TempDir(), "absent.json")); err == nil {
		t.Error("a missing file was accepted")
	}
}

// capCue is what a captured state must show: patterns the .txt frame matches or does
// not, and patterns the .annot.txt file matches or does not.
type capCue struct {
	state           string
	frame, notFrame []string
	annot, notAnnot []string
}

// cueProblems names each pattern of cue that the frame and annotation text fail.
func cueProblems(frame, annot string, cue capCue) []string {
	var out []string
	check := func(text string, pats []string, want bool, what string) {
		for _, p := range pats {
			if regexp.MustCompile(p).MatchString(text) != want {
				out = append(out, fmt.Sprintf("%s: %s %q", cue.state, what, p))
			}
		}
	}
	check(frame, cue.frame, true, "frame lacks")
	check(frame, cue.notFrame, false, "frame holds")
	check(annot, cue.annot, true, "annotation lacks")
	check(annot, cue.notAnnot, false, "annotation holds")
	return out
}

// capCues is the cue each v0.1.3 and v0.1.4 capture is there to show, written from
// the walkthrough's "You should see" text and not from the frames. A walkthrough cue
// that is a hint, a marker or a colour role is a pattern here. Rows 07 and 09 of
// s07-fence-languages are the code lines: the postgresql block and the sql block,
// each highlighted, the postgresql one with the SQL lexer that hlInfoAlias gives
// its info string.
var capCues = []capCue{
	{state: "s01-arrows-last-reply", frame: []string{`(?m)^ ┃ ` + capGoodbyeReply + `\s*$`}, notFrame: []string{`┃ Hello`, `┃ say`}},
	{state: "s01-arrows-first-reply", frame: []string{`(?m)^ ┃ ` + capHelloReply}, notFrame: []string{`┃ Goodbye`, `┃ say goodbye`}},
	{state: "s01-arrows-top-stays", frame: []string{`(?m)^ ┃ ` + capHelloReply}, notFrame: []string{`┃ Goodbye`, `┃ say`}},
	{state: "s01-arrows-composer", frame: []string{`(?m)^ ┃ ` + capGoodbyeReply, `(?m)^ > ▌`, `↑↓ history`}},
	{state: "s02-enter-d-noop", frame: []string{`(?m)^ ┃ ` + capGoodbyeReply, `↑↓ select · c copy · Tab/Esc prompt`}, notFrame: []string{`┌─`}},
	{state: "s03-copy-reply", frame: []string{`copied 23 characters`, `(?m)^ ┃ ` + capGoodbyeReply}},
	{state: "s04-copy-card-hint", frame: []string{`c copies replies only`, `(?m)^ ┃ · model`}, notFrame: []string{`copied`}},
	{state: "s05-copy-command", frame: []string{`copied 23 characters`}},
	{state: "s06-copy-code", frame: []string{`copied 27 characters`}},
	{state: "s07-copy-code-range", frame: []string{`the last reply has 2 code blocks`}, notFrame: []string{`copied`}},
	{state: "s08-copy-usage", frame: []string{`usage: /copy \[code \[N\]\]`}, notFrame: []string{`copied`}},
	{state: "s09-copy-notice-clears", frame: []string{`(?m)^ > x▌`}, notFrame: []string{`copied`}},
	{state: "s10-c-tab-x", frame: []string{`(?m)^ > /cx▌`}},
	{state: "s10-co-tab", frame: []string{`(?m)^ > /co▌`}},
	{state: "s10-cop-tab", frame: []string{`(?m)^ > /copy▌`}},
	{state: "s11-help-open", frame: []string{`┌─ help`, `Esc or \? closes`, `c +copy reply`, `/exit  /copy`}, notFrame: []string{`↓ \d+ more`}},
	{state: "s11-help-closed-hint", frame: []string{`c copies replies only`}, notFrame: []string{`┌─ help`}},

	{
		state:    "s01-markdown-reply",
		frame:    []string{`(?m)^ # Heading check`, `(?m)^ A paragraph with bold words, `, `• first item`, `• second item`, "`inline code`", `~~struck words~~`},
		notFrame: []string{`(?m)^ A paragraph with \*\*`},
		annot: []string{
			`bold "bold words"`, `strike "struck words"`,
			`fg=SynKeyword\(38;5;141\) "func"`, `fg=SynKeyword\(38;5;141\) "return"`,
		},
	},
	{
		state:    "s02-markdown-off",
		frame:    []string{`# Heading check`, `\*\*bold words\*\*`, `~~struck words~~`, "(?m)^ ```go", "(?m)^ ``` +[┃│]"},
		notFrame: []string{`•`},
		notAnnot: []string{`SynKeyword`, `bold`, `strike`},
	},
	{
		state: "s03-readfile-modal-go",
		frame: []string{`┌─ read_file hello.go`},
		annot: []string{
			`fg=SynKeyword\(38;5;141\) "package"`, `fg=SynKeyword\(38;5;141\) "func"`,
			`fg=SynString\(38;5;180\) "\\"hi\\""`, `cols 11-15 fg=Dim\(38;5;245\) "   1 "`,
		},
	},
	{state: "s04-approval-card", frame: []string{`approval required`, `\[d\] view diff`}},
	{
		state: "s04-approval-diff",
		frame: []string{`┌─ notes.go`},
		annot: []string{
			`fg=Success\(38;5;120\) "\+"`, `fg=SynKeyword\(38;5;141\) "package"`,
			`fg=SynKeyword\(38;5;141\) "func"`, `fg=SynString\(38;5;180\) "\\"hi\\""`,
		},
	},
	{state: "s04-approval-rejected", frame: []string{`✗ patch rejected`}},
	{
		state:    "s05-diff-two-files",
		frame:    []string{`\+func extra\(\) \{\}`, `\+def extra\(\): pass`},
		annot:    []string{`fg=Success\(38;5;120\) "\+"`, `fg=SynKeyword\(38;5;141\) "func"`, `fg=SynKeyword\(38;5;141\) "def"`},
		notAnnot: []string{`SynKeyword[^\n]*diff`},
	},
	{
		state: "s06-no-color-diff",
		frame: []string{`\+func extra\(\) \{\}`, `\+def extra\(\): pass`},
	},
	{
		state:    "s06-no-color-reply",
		frame:    []string{`# Heading check`, `A paragraph with bold words, `, "`inline code`", `~~struck words~~`},
		notFrame: []string{`A paragraph with \*\*`},
	},
	{
		state: "s07-fence-languages",
		annot: []string{
			`row 07 cols 3-8 fg=SynKeyword\(38;5;141\) "SELECT"`, `row 07 cols 10-10 fg=SynNumber\(38;5;216\) "1"`,
			`row 09 cols 3-8 fg=SynKeyword\(38;5;141\) "SELECT"`, `row 09 cols 10-10 fg=SynNumber\(38;5;216\) "1"`,
		},
	},
}

// TestCaptureStatesShowTheirCues: entry point captureFiles on each state in capCues,
// the same text the compare test holds the committed files to. Each state's frame and
// annotation match the cue patterns written from the walkthrough. Control: a state's
// cue checked against another state's text fails, so the patterns tell the states
// apart.
func TestCaptureStatesShowTheirCues(t *testing.T) {
	files := func(name string) (frame, annot string) {
		got := captureFiles(t, capState(t, name))
		return got[name+".txt"], got[name+".annot.txt"]
	}
	for _, cue := range capCues {
		frame, annot := files(cue.state)
		for _, p := range cueProblems(frame, annot, cue) {
			t.Error(p)
		}
	}

	copyFrame, _ := files("s03-copy-reply")
	usageFrame, _ := files("s08-copy-usage")
	if len(cueProblems(usageFrame, "", capCue{state: "s03 cue on s08", frame: []string{`copied 23 characters`}})) == 0 {
		t.Error("control: the s03 cue matched the s08 frame")
	}
	if len(cueProblems(copyFrame, "", capCue{state: "s08 cue on s03", frame: []string{`usage: /copy`}})) == 0 {
		t.Error("control: the s08 cue matched the s03 frame")
	}
	_, fenceAnnot := files("s07-fence-languages")
	_, mdAnnot := files("s01-markdown-reply")
	if len(cueProblems("", mdAnnot, capCue{state: "s07 cue on s01", annot: []string{`row 09 cols 3-8 fg=SynKeyword\(38;5;141\) "SELECT"`}})) == 0 {
		t.Error("control: the s07 annotation cue matched the s01 annotation")
	}
	if len(cueProblems("", fenceAnnot, capCue{state: "s01 cue on s07", annot: []string{`bold "bold words"`}})) == 0 {
		t.Error("control: the s01 annotation cue matched the s07 annotation")
	}
}

// TestCaptureAbsenceCuesHavePositiveControls: entry point captureFiles on the states
// whose cue is something absent, each beside a state built from the same steps in which
// the thing is present, so the pattern the cue forbids can be seen to match when it should.
// s02-enter-d-noop forbids a modal box (`┌─`): the same keys, Shift+↑, Enter and d, on a
// read_file card open one, and the frame matches the pattern. s11-help-open forbids the
// help overlay's clipping marker (`↓ N more`): the same steps in a 24-row window, where
// the overlay does not fit, show it. Each real state is checked against the pattern here
// as well, so the forbidden pattern and its control come from one regexp.
func TestCaptureAbsenceCuesHavePositiveControls(t *testing.T) {
	frameOf := func(cs captureState) string { return captureFiles(t, cs)[cs.Name+".txt"] }
	for _, c := range []struct {
		name, pattern string
		control       captureState
	}{
		{
			"s02-enter-d-noop", `┌─`,
			copyRow("ctl-d-on-a-card", 80, 24, "", appendSteps(
				[]captureStep{capMsgs("(a read_file card finishes)", capReadFile("calc/divide.go", capDivideFile)...)},
				keySteps("Shift+↑", "Enter", "d")...)...),
		},
		{
			"s11-help-open", `↓ \d+ more`,
			copyRow("ctl-help-at-24-rows", 80, 24, "", appendSteps(noticeSelected(), capKeys("Shift+↑"), capKeys("?"))...),
		},
	} {
		re := regexp.MustCompile(c.pattern)
		if re.MatchString(frameOf(capState(t, c.name))) {
			t.Errorf("%s: the frame matches the forbidden pattern %q", c.name, c.pattern)
		}
		if !re.MatchString(frameOf(c.control)) {
			t.Errorf("%s: control: the frame of %s does not match %q, so its absence above proves nothing", c.name, c.control.Name, c.pattern)
		}
	}
}

// TestCaptureNoColourStatesCarryNoEscapes: entry point captureModel on the NO_COLOR
// states, whose Caps come from DetectCaps under noColourEnv. Their frames carry no
// escape sequence and read the same as the frames of the same steps under colourEnv
// with the escapes stripped, which is the walkthrough's "the same layout". Controls: the
// same steps under colourEnv do carry escapes, so the absence is not a model that
// never styles anything.
func TestCaptureNoColourStatesCarryNoEscapes(t *testing.T) {
	for _, name := range []string{"s06-no-color-diff", "s06-no-color-reply"} {
		cs := capState(t, name)
		if cs.Env["NO_COLOR"] != "1" {
			t.Fatalf("%s: Env %v does not set NO_COLOR=1", name, cs.Env)
		}
		plainFrame := captureModel(t, cs, true).View()
		if strings.Contains(plainFrame, "\x1b") {
			t.Errorf("%s: the NO_COLOR frame carries an escape sequence:\n%q", name, plainFrame)
		}
		withColour := cs
		withColour.Env = colourEnv
		coloured := captureModel(t, withColour, true).View()
		if !strings.Contains(coloured, "\x1b[") {
			t.Errorf("%s: control: the frame under colourEnv carries no escape sequence", name)
		}
		if got := stripSGR(coloured); got != plainFrame {
			t.Errorf("%s: the NO_COLOR frame differs from the colour frame with escapes stripped:\n%s", name, diffText(got, plainFrame))
		}
	}
}

// walkthroughSections is the section numbers of release in the walkthrough text: the
// "### N. " headings between the "## <release> ·" heading and the next "## " one.
func walkthroughSections(text, release string) []int {
	var out []int
	in := false
	heading := regexp.MustCompile(`^### (\d+)\. `)
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "## "+release+" ·"):
			in = true
		case strings.HasPrefix(line, "## "):
			in = false
		case in:
			if m := heading.FindStringSubmatch(line); m != nil {
				var n int
				fmt.Sscan(m[1], &n)
				out = append(out, n)
			}
		}
	}
	return out
}

// sectionsGap is the sections that have no capture and no entry in notCaptured. A
// capture belongs to section N when its name starts "sNN-"; an entry is a line of
// notCaptured that starts "- Section N" and then ends the number.
func sectionsGap(sections []int, names []string, notCaptured string) []int {
	var out []int
	for _, n := range sections {
		prefix := fmt.Sprintf("s%02d-", n)
		found := false
		for _, name := range names {
			if strings.HasPrefix(name, prefix) {
				found = true
			}
		}
		if !found && !regexp.MustCompile(fmt.Sprintf(`(?m)^- Section %d\b`, n)).MatchString(notCaptured) {
			out = append(out, n)
		}
	}
	return out
}

// TestCaptureSectionsAreAccountedFor: entry point sectionsGap on the walkthrough's own
// headings. Every section of the v0.1.3 and v0.1.4 parts is either the start of a
// state's name in captureTable or named in its release's NOT-CAPTURED.md. Controls: the
// same check on a copy of the headings with an extra section 99 reports that section
// alone, and on a NOT-CAPTURED.md with the entry for a named section removed reports
// that section.
func TestCaptureSectionsAreAccountedFor(t *testing.T) {
	raw, err := os.ReadFile("../../plan/testing/manual/ux-round-v0.1.md")
	if err != nil {
		t.Fatalf("read the walkthrough: %v", err)
	}
	for _, release := range []string{"v0.1.3", "v0.1.4"} {
		sections := walkthroughSections(string(raw), release)
		if len(sections) == 0 {
			t.Fatalf("%s: no section headings found in the walkthrough", release)
		}
		var names []string
		for _, cs := range captureTable {
			if cs.Release == release {
				names = append(names, cs.Name)
			}
		}
		nc, err := os.ReadFile(filepath.Join(captureDir(release), notCapturedFile))
		if err != nil {
			t.Fatalf("%s: %v", release, err)
		}
		if gap := sectionsGap(sections, names, string(nc)); len(gap) != 0 {
			t.Errorf("%s: sections with neither a capture nor a %s entry: %v", release, notCapturedFile, gap)
		}
		if gap := sectionsGap(append(append([]int(nil), sections...), 99), names, string(nc)); len(gap) != 1 || gap[0] != 99 {
			t.Errorf("%s: control: an extra section 99 gave the gap %v, want [99]", release, gap)
		}
		// A section with a capture, removed from names, is a gap when notCaptured does
		// not name it either.
		if len(names) > 0 {
			var section int
			fmt.Sscanf(names[0], "s%d-", &section)
			var rest []string
			for _, n := range names {
				if !strings.HasPrefix(n, fmt.Sprintf("s%02d-", section)) {
					rest = append(rest, n)
				}
			}
			if gap := sectionsGap(sections, rest, ""); len(gap) == 0 {
				t.Errorf("%s: control: removing section %d's captures left no gap", release, section)
			}
		}
	}
}
