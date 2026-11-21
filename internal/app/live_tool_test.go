package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/provider/anthropic"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// The tests in this file run a model-driven tool call the way a live session
// does: Submit, the agent loop, and a stream that the real anthropic.Decode turns
// into provider events. The Anthropic start event of a tool call carries the
// call's ID and name and no input (startTool in
// internal/provider/anthropic/decode.go); the complete input arrives with the end
// event (the kindToolUse case in the same file). Nothing here hand-builds a
// ToolStartedMsg or ToolCompletedMsg: they are what adaptTurnEvent and
// toolExecutor.Invoke send for those events.

// sseMessageStart opens every scripted stream.
const sseMessageStart = `event: message_start
data: {"type":"message_start","message":{"id":"msg_live","type":"message","role":"assistant","model":"synthetic","content":[],"usage":{"input_tokens":1,"output_tokens":0}}}

`

// sseMessageEnd closes a stream with the given stop reason.
func sseMessageEnd(stop string) string {
	return fmt.Sprintf(`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":%q},"usage":{"input_tokens":1,"output_tokens":1}}

event: message_stop
data: {"type":"message_stop"}

`, stop)
}

// jsonString is s as a JSON string literal. The name and the ID of a block go through
// it too, because Go's %q is not JSON: it writes \x1b where JSON needs \u001b.
func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// splitAtRune cuts s in two at the rune boundary nearest its middle byte, at or
// before it, so neither half ends in part of a multi-byte character.
func splitAtRune(s string) (string, string) {
	mid := len(s) / 2
	for mid > 0 && !utf8.RuneStart(s[mid]) {
		mid--
	}
	return s[:mid], s[mid:]
}

// sseTextBlock is the events of a text block at index idx holding text.
func sseTextBlock(idx int, text string) string {
	return fmt.Sprintf(`event: content_block_start
data: {"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}

event: content_block_stop
data: {"type":"content_block_stop","index":%d}

`, idx, idx, jsonString(text), idx)
}

// sseToolUseBlock is the events of a tool_use block at index idx: the block starts with
// the call's ID and name and an empty input object, the input arrives as one
// input_json_delta per fragment, in order, and the block stops.
func sseToolUseBlock(idx int, callID, name string, frags ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `event: content_block_start
data: {"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":%s,"name":%s,"input":{}}}

`, idx, jsonString(callID), jsonString(name))
	for _, f := range frags {
		fmt.Fprintf(&b, `event: content_block_delta
data: {"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%s}}

`, idx, jsonString(f))
	}
	fmt.Fprintf(&b, `event: content_block_stop
data: {"type":"content_block_stop","index":%d}

`, idx)
	return b.String()
}

// sseToolUse is a stream with one tool_use block whose input arrives as the
// input_json_delta fragments splitAtRune cuts it into. It is the shape the Anthropic
// API sends.
func sseToolUse(callID, name, inputJSON string) string {
	first, second := splitAtRune(inputJSON)
	return sseMessageStart + sseToolUseBlock(0, callID, name, first, second) + sseMessageEnd("tool_use")
}

// sseTextThenToolUse is a stream with a text block at index 0 and then a tool_use block
// at index 1 whose first input_json_delta fragment is empty and whose second holds the
// whole input. The API can send text before a call, and an empty first fragment.
func sseTextThenToolUse(text, callID, name, inputJSON string) string {
	return sseMessageStart + sseTextBlock(0, text) + sseToolUseBlock(1, callID, name, "", inputJSON) + sseMessageEnd("tool_use")
}

// sseReply is a stream with one text block.
func sseReply(text string) string {
	return sseMessageStart + sseTextBlock(0, text) + sseMessageEnd("end_turn")
}

// scriptedSSE is a provider.Provider that answers its Nth Stream call by running the
// Nth body through the real anthropic.Decode; the last body answers every call after.
type scriptedSSE struct {
	mu     sync.Mutex
	bodies []string
	calls  int
}

// Stream decodes the body for this call into onEvent.
func (p *scriptedSSE) Stream(ctx context.Context, _ provider.Request, onEvent func(provider.StreamEvent)) error {
	p.mu.Lock()
	i := p.calls
	if i >= len(p.bodies) {
		i = len(p.bodies) - 1
	}
	p.calls++
	body := p.bodies[i]
	p.mu.Unlock()
	return anthropic.Decode(ctx, bytes.NewReader([]byte(body)), onEvent)
}

// liveEnv is an App on the anthropic endpoint, over a repo the test names, whose
// provider answers from scripted SSE bodies.
type liveEnv struct {
	t *testing.T
	a *App
	c *collector
}

// newLiveEnv builds the App over repo and starts its session. onMsg, when not nil,
// sees every message the app sends the tui, on the sending goroutine, after the
// collector has it.
func newLiveEnv(t *testing.T, repo string, onMsg func(*App, any), bodies ...string) *liveEnv {
	t.Helper()
	isolateGit(t)
	ws, err := workspace.Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Provider.Default = "anthropic"
	cfg.Session.StorageDir = filepath.Join(t.TempDir(), "sessions")
	c := newCollector(-1)
	a := New(ws, cfg, telemetry.Disabled())
	a.sendFn = func(msg any) {
		c.send(msg)
		if onMsg != nil {
			onMsg(a, msg)
		}
	}
	t.Cleanup(a.Close)
	p := &scriptedSSE{bodies: bodies}
	a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return p, nil
	}
	a.StartSession(getenvKey)
	return &liveEnv{t: t, a: a, c: c}
}

// turn submits text and returns every message the app sent once the turn's
// TurnCompleteMsg has arrived and the turn goroutine has exited.
func (e *liveEnv) turn(text string) []any {
	e.t.Helper()
	e.a.Submit(text, getenvKey)
	deadline := time.Now().Add(10 * time.Second)
	for countMsgs[tui.TurnCompleteMsg](e.c.snapshot()) == 0 {
		if time.Now().After(deadline) {
			e.t.Fatalf("no TurnCompleteMsg after 10s; got %#v", e.c.snapshot())
		}
		time.Sleep(5 * time.Millisecond)
	}
	e.a.inFlight.Wait()
	return e.c.snapshot()
}

// toolMsgs keeps the messages that concern tool cards and their approvals, in order.
func toolMsgs(msgs []any) []any {
	var out []any
	for _, m := range msgs {
		switch m.(type) {
		case tui.ToolStartedMsg, tui.ToolTargetMsg, tui.ApprovalRequestedMsg, tui.ToolCompletedMsg:
			out = append(out, m)
		}
	}
	return out
}

// liveModel is the tui model the live tests feed: colour on, Unicode glyphs, 100 by 40.
func liveModel(colour bool) tui.Model {
	m := tui.New(tui.Options{Version: "0.0.0-test", Caps: tui.Caps{Colour: colour, Unicode: true}})
	return feed(m, tea.WindowSizeMsg{Width: 100, Height: 40})
}

// toMsgs converts what the collector holds to tea messages.
func toMsgs(in []any) []tea.Msg {
	out := make([]tea.Msg, len(in))
	for i, m := range in {
		out[i] = m
	}
	return out
}

// msgTypeNames is the %T of each message, in order.
func msgTypeNames(msgs []any) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = fmt.Sprintf("%T", m)
	}
	return out
}

// keywordSGR is what the colour-on frame wraps a Go keyword in: the SynKeyword role,
// 38;5;141 in the capture annotations (plan/testing/manual/captures/v0.1.4/
// s03-readfile-modal-go.annot.txt). A modal row whose "package" is wrapped in it is
// highlighted.
const keywordSGR = "\x1b[38;5;141m"

// modalFrame feeds msgs to a colour-on model, presses Shift+↑ and d, and returns the
// frame with the modal the card opens.
func modalFrame(msgs []tea.Msg) string {
	m := feed(liveModel(true), msgs...)
	m = feed(m, tea.KeyMsg{Type: tea.KeyShiftUp}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	return m.View()
}

// keywordPackage reports whether the frame draws "package" in the keyword colour.
func keywordPackage(frame string) bool {
	return regexp.MustCompile(regexp.QuoteMeta(keywordSGR) + `package\x1b\[0m`).MatchString(frame)
}

// liveReadFile runs a model-driven read_file call for path, in a temporary repo that holds
// text there (its directory made if need be), through the live path, and returns every
// message the app sent.
func liveReadFile(t *testing.T, path, text string) []any {
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
	return e.turn("read " + path)
}

// withoutTarget is msgs with every ToolTargetMsg dropped and, when blank, the target of
// every ToolCompletedMsg cleared too. Dropping is the sequence a call sends when no end
// event was seen; dropping and clearing is the sequence before the target was
// carried anywhere, which is the control for the assertions that a target is there.
func withoutTarget(msgs []any, blank bool) []any {
	var out []any
	for _, m := range msgs {
		switch v := m.(type) {
		case tui.ToolTargetMsg:
			continue
		case tui.ToolCompletedMsg:
			if blank {
				v.Target = ""
			}
			m = v
		}
		out = append(out, m)
	}
	return out
}

// TestLiveTurnSendsTheMessagesInThisOrder: entry point Submit, through the agent loop and
// a stream the real anthropic.Decode reads, for a turn whose first reply is a read_file
// call and whose second is text. The messages the app sends the tui, as their types, must
// equal the list below exactly. internal/tui's capture comments cite this order: a
// BudgetMsg before the turn, a ToolStartedMsg with no target and a ToolTargetMsg as the
// call starts and ends, the first reply's UsageMsg and SessionUsageMsg, the
// ToolCompletedMsg when the tool has run, then the text reply, its usage messages and
// TurnCompleteMsg. The captures and the chain replay send the tool messages and the text
// and leave the budget and usage messages out.
func TestLiveTurnSendsTheMessagesInThisOrder(t *testing.T) {
	msgs := liveReadFile(t, "main.go", "package main\n")
	want := []string{
		"tui.BudgetMsg", "tui.ToolStartedMsg", "tui.ToolTargetMsg", "tui.UsageMsg", "tui.SessionUsageMsg",
		"tui.ToolCompletedMsg", "tui.AssistantTextDeltaMsg", "tui.UsageMsg", "tui.SessionUsageMsg", "tui.TurnCompleteMsg",
	}
	if got := msgTypeNames(msgs); !reflect.DeepEqual(got, want) {
		t.Errorf("the live turn sent\n got  %v\n want %v", got, want)
	}
}

// TestLiveReadFileCardNamesItsFileBeforeTheToolFinishes: entry point Submit with a
// read_file call whose stream the real anthropic.Decode reads. The Anthropic start event
// carries no input, so the ToolStartedMsg has no target (the card could not name its file
// from it); the ToolTargetMsg the call's end event produces comes before the
// ToolCompletedMsg and names the path. A tui model fed the messages up to and including
// the ToolTargetMsg draws the card's head with the path while the card is still running.
// Control: the model fed only up to the ToolStartedMsg draws a head without the path, so
// the assertion can see its absence.
func TestLiveReadFileCardNamesItsFileBeforeTheToolFinishes(t *testing.T) {
	msgs := liveReadFile(t, "main.go", "package main\n")
	idx := func(f func(any) bool) int {
		for i, m := range msgs {
			if f(m) {
				return i
			}
		}
		return -1
	}
	started := idx(func(m any) bool { _, ok := m.(tui.ToolStartedMsg); return ok })
	targeted := idx(func(m any) bool { _, ok := m.(tui.ToolTargetMsg); return ok })
	completed := idx(func(m any) bool { _, ok := m.(tui.ToolCompletedMsg); return ok })
	if started < 0 || targeted < 0 || completed < 0 {
		t.Fatalf("want a ToolStartedMsg, a ToolTargetMsg and a ToolCompletedMsg, got %v", msgTypeNames(msgs))
	}
	if started >= targeted || targeted >= completed {
		t.Fatalf("the messages are in order %v, want start, target, completion", msgTypeNames(msgs))
	}
	if got := msgs[started].(tui.ToolStartedMsg).Target; got != "" {
		t.Errorf("the ToolStartedMsg has target %q, want none: the Anthropic start event carries no input", got)
	}
	if got := msgs[targeted].(tui.ToolTargetMsg).Target; got != "main.go" {
		t.Errorf("the ToolTargetMsg names %q, want main.go", got)
	}

	running := feed(liveModel(false), toMsgs(msgs[:targeted+1])...).View()
	if !strings.Contains(running, "read_file main.go") {
		t.Errorf("the running card's head does not name the file:\n%s", running)
	}
	before := feed(liveModel(false), toMsgs(msgs[:started+1])...).View()
	if strings.Contains(before, "main.go") {
		t.Errorf("control: the card's head names the file before any message carried it:\n%s", before)
	}
}

// TestLiveReadFileModalHighlightsItsFile: entry point Submit with a read_file call, the
// tui model fed everything the app sent, then Shift+↑ and d. The modal's title names the
// file and the Go keyword "package" is drawn in the keyword colour. Controls, through the
// same assertions: the same call for a file with no known language (the path is in the
// title, "package" is not coloured), and the sequence a card got before the target was
// carried anywhere (no ToolTargetMsg and no target on the completion): the title has no
// path and "package" is not coloured.
func TestLiveReadFileModalHighlightsItsFile(t *testing.T) {
	msgs := liveReadFile(t, "main.go", "package main\n")
	frame := modalFrame(toMsgs(msgs))
	if !strings.Contains(frame, "read_file main.go") || !keywordPackage(frame) {
		t.Errorf("the modal for the live card: title names the file %v, \"package\" is coloured %v:\n%s",
			strings.Contains(frame, "read_file main.go"), keywordPackage(frame), frame)
	}

	other := modalFrame(toMsgs(liveReadFile(t, "notes.nosuchlanguage", "package main\n")))
	if !strings.Contains(other, "read_file notes.nosuchlanguage") || keywordPackage(other) {
		t.Errorf("control: a file with no known language: title names the file %v, \"package\" is coloured %v",
			strings.Contains(other, "read_file notes.nosuchlanguage"), keywordPackage(other))
	}

	bare := modalFrame(toMsgs(withoutTarget(msgs, true)))
	if strings.Contains(bare, "main.go") || keywordPackage(bare) {
		t.Errorf("control: with no target carried anywhere the modal still names the file or highlights:\n%s", bare)
	}
}

// TestLiveReadFileCardTakesItsTargetFromTheResult: entry point Submit with a read_file
// call, with the live messages minus the ToolTargetMsg, which is what the tui gets when the
// call's end event was not seen. The ToolCompletedMsg that toolExecutor.Invoke sent still
// carries the target, and the card takes it: the finished card's modal names the file and
// is highlighted. Control: with the completion's target cleared as well, neither source
// is left and the modal has no path and no colour.
func TestLiveReadFileCardTakesItsTargetFromTheResult(t *testing.T) {
	msgs := liveReadFile(t, "main.go", "package main\n")
	noEnd := withoutTarget(msgs, false)
	if countMsgs[tui.ToolTargetMsg](noEnd) != 0 {
		t.Fatal("setup: the ToolTargetMsg is still in the sequence")
	}
	done := findMsg[tui.ToolCompletedMsg](noEnd)
	if done == nil || done.Target != "main.go" {
		t.Fatalf("setup: the result carries target %v, want main.go", done)
	}
	frame := modalFrame(toMsgs(noEnd))
	if !strings.Contains(frame, "read_file main.go") || !keywordPackage(frame) {
		t.Errorf("the modal of a card that got its target from the result: title names the file %v, \"package\" is coloured %v:\n%s",
			strings.Contains(frame, "read_file main.go"), keywordPackage(frame), frame)
	}
	bare := modalFrame(toMsgs(withoutTarget(msgs, true)))
	if strings.Contains(bare, "main.go") || keywordPackage(bare) {
		t.Errorf("control: with neither source the modal still names the file or highlights:\n%s", bare)
	}
}

// TestLiveApplyPatchApprovalLinksToItsToolCard: entry point Submit with a model-driven
// apply_patch call that the test approves through App.Resolve (the user pressing y), in a
// temporary repo. ui-spec §3.4 and §3.8: once approved, the approval card folds into its
// tool card, the result card's head shows "approved", and d on the card opens the diff.
// The tool messages must be, as their types, exactly the start, the approval request and
// the completion: apply_patch's input names no target, so no ToolTargetMsg is sent. The
// ApprovalRequestedMsg must carry the tool card's ID, and a tui model fed the live
// messages with y pressed at the approval must show the fold, the badge and the diff.
// Control: the same messages with the request's ToolID zeroed, which is what the app
// sent before the call was tagged, leave the approval card in the transcript and open
// the result modal, not the diff, so each assertion can see its absence.
func TestLiveApplyPatchApprovalLinksToItsToolCard(t *testing.T) {
	in := loadCaptureInputs(t).PatchApproval
	repo := newGitRepo(t, map[string]string{"hello.go": "package main\n"})
	input, err := json.Marshal(map[string]string{"diff": in.Input.Diff, "description": in.Input.Description})
	if err != nil {
		t.Fatal(err)
	}
	e := newLiveEnv(t, repo, func(a *App, msg any) {
		if ask, ok := msg.(tui.ApprovalRequestedMsg); ok {
			go a.Resolve(ask.ID, ApprovalOutcomeOnce)
		}
	}, sseToolUse(liveCallID, "apply_patch", string(input)), sseReply("done"))
	msgs := toolMsgs(e.turn("patch"))
	if _, err := os.Stat(filepath.Join(repo, in.Requested.DiffFilename)); err != nil {
		t.Fatalf("the approved patch was not applied: %v", err)
	}
	if got, want := msgTypeNames(msgs), []string{"tui.ToolStartedMsg", "tui.ApprovalRequestedMsg", "tui.ToolCompletedMsg"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the apply_patch call sent the tool messages\n got  %v\n want %v", got, want)
	}
	start, ask := findMsg[tui.ToolStartedMsg](msgs), findMsg[tui.ApprovalRequestedMsg](msgs)
	if start == nil || ask == nil {
		t.Fatalf("want a ToolStartedMsg and an ApprovalRequestedMsg, got %v", msgTypeNames(msgs))
	}
	if ask.ToolID == 0 || ask.ToolID != start.ID {
		t.Errorf("the approval request has ToolID %d, want the started card's ID %d", ask.ToolID, start.ID)
	}

	// show feeds msgs with y pressed when the approval arrives, finishes the call, and
	// returns the frame, then the frame after Shift+↑ and d.
	show := func(msgs []any) (frame, afterD string) {
		m := liveModel(false)
		m.ResolveApproval = func(int64, tui.ApprovalOutcome) {}
		for _, msg := range msgs {
			m = feed(m, msg)
			if _, ok := msg.(tui.ApprovalRequestedMsg); ok {
				m = feed(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			}
		}
		frame = m.View()
		m = feed(m, tea.KeyMsg{Type: tea.KeyShiftUp}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		return frame, m.View()
	}
	const unfolded = "apply_patch 1 file changed"
	frame, afterD := show(msgs)
	if strings.Contains(frame, unfolded) {
		t.Errorf("the approval card did not fold into its tool card:\n%s", frame)
	}
	if !strings.Contains(frame, "· approved ·") {
		t.Errorf("the result card's head does not show approved:\n%s", frame)
	}
	if !strings.Contains(afterD, "┌─ "+in.Requested.DiffFilename) || !strings.Contains(afterD, "+package main") {
		t.Errorf("d on the card did not open the diff:\n%s", afterD)
	}

	var unlinked []any
	for _, m := range msgs {
		if a, ok := m.(tui.ApprovalRequestedMsg); ok {
			a.ToolID = 0
			m = a
		}
		unlinked = append(unlinked, m)
	}
	frame, afterD = show(unlinked)
	if !strings.Contains(frame, unfolded) {
		t.Errorf("control: with no ToolID the approval card is gone from the frame:\n%s", frame)
	}
	if strings.Contains(frame, "· approved ·") {
		t.Errorf("control: with no ToolID the result card's head shows approved:\n%s", frame)
	}
	if strings.Contains(afterD, "┌─ "+in.Requested.DiffFilename) {
		t.Errorf("control: with no ToolID d opened the diff:\n%s", afterD)
	}
}

// TestFormatPatchDisplayClosesItsParenthesis: entry point formatPatchDisplay with each
// count of modified files in the table below. The detail lists the files as "files: N
// changed (a," then a line per further file indented to the first name, and the parenthesis
// closes after the last name: on the first line for one file. Each detail's parentheses
// are balanced. Control: the same check run on the one-file detail with its closing
// parenthesis removed reports it unbalanced, beside the real detail, which is balanced.
func TestFormatPatchDisplayClosesItsParenthesis(t *testing.T) {
	files := func(names ...string) []patch.FileChange {
		var out []patch.FileChange
		for _, n := range names {
			out = append(out, patch.FileChange{Op: patch.OpModify, Path: n})
		}
		return out
	}
	balanced := func(lines []string) bool {
		all := strings.Join(lines, "\n")
		return strings.Count(all, "(") == strings.Count(all, ")") && !strings.HasSuffix(strings.TrimSpace(all), "(")
	}
	for _, tc := range []struct {
		name    string
		changes []patch.FileChange
		subject string
		detail  []string
	}{
		{"one file", files("a.go"), "1 file changed", []string{"files: 1 changed (a.go)"}},
		{"two files", files("a.go", "b.go"), "2 files changed", []string{"files: 2 changed (a.go,", "       b.go)"}},
		{
			"three files", files("a.go", "b.go", "c.go"), "3 files changed",
			[]string{"files: 3 changed (a.go,", "       b.go,", "       c.go)"},
		},
		{"no files", nil, "0 files changed", nil},
	} {
		subject, detail := formatPatchDisplay(tc.changes)
		if subject != tc.subject || !reflect.DeepEqual(detail, tc.detail) {
			t.Errorf("%s: subject %q detail %q, want %q and %q", tc.name, subject, detail, tc.subject, tc.detail)
		}
		if !balanced(detail) {
			t.Errorf("%s: the detail's parentheses are not balanced: %q", tc.name, detail)
		}
	}
	if balanced([]string{"files: 1 changed (a.go"}) {
		t.Error("control: a detail with no closing parenthesis was reported balanced")
	}
}

// hostilePath is a path a model could write: a CSI colour sequence, an OSC 52 clipboard
// write ending in BEL, a newline and a carriage return. hostileToolName is a tool name
// with a screen-clearing CSI, an OSC 52 write and a newline.
const (
	hostilePath     = "evil\x1b[31mred\x1b[0m\x1b]52;c;ZXZpbA==\x07\nnext\rcr.go"
	hostileToolName = "rogue\x1b[2J\x1b]52;c;ZXZpbA==\x07\ntool"
)

// leak names the first thing in s that sanitised one-row text must not hold: an escape,
// a BEL, a newline, a carriage return or the OSC 52 payload. It is "" when s holds none.
func leak(s string) string {
	for _, bad := range []string{"\x1b", "\x07", "\n", "\r", "52;c;"} {
		if strings.Contains(s, bad) {
			return fmt.Sprintf("%q", bad)
		}
	}
	return ""
}

// frameLeak names an escape or a BEL in a colour-off frame, which draws neither, or "".
func frameLeak(frame string) string {
	for _, bad := range []string{"\x1b", "\x07", "52;c;"} {
		if strings.Contains(frame, bad) {
			return fmt.Sprintf("%q", bad)
		}
	}
	return ""
}

// TestDescribeInputSanitisesEveryField: entry points describeInput and
// describeToolCallInput. A path, a query and an argv each carrying a CSI sequence, an
// OSC 52 write, a newline and a carriage return come back as SanitizeSingleLine of the
// text, holding none of them. A path that is nothing but an escape sanitises to "" and is
// passed over for the query. Positive controls: an ordinary path with a non-ASCII rune
// and a space, an ordinary query and an ordinary argv come back byte for byte.
func TestDescribeInputSanitisesEveryField(t *testing.T) {
	const ordinary = "src/ïïve file.go"
	for _, tc := range []struct {
		name  string
		input map[string]any
		want  string
	}{
		{"hostile path", map[string]any{"path": hostilePath}, tui.SanitizeSingleLine(hostilePath)},
		{"hostile query", map[string]any{"query": hostilePath}, tui.SanitizeSingleLine(hostilePath)},
		{"hostile argv as []any", map[string]any{"argv": []any{"echo", hostilePath}}, tui.SanitizeSingleLine("echo " + hostilePath)},
		{"hostile argv as []string", map[string]any{"argv": []string{"echo", hostilePath}}, tui.SanitizeSingleLine("echo " + hostilePath)},
		{"a path that is only an escape", map[string]any{"path": "\x1b[0m", "query": "needle"}, "needle"},
		{"control: ordinary path", map[string]any{"path": ordinary}, ordinary},
		{"control: ordinary query", map[string]any{"query": "func main"}, "func main"},
		{"control: ordinary argv", map[string]any{"argv": []any{"go", "test", "./..."}}, "go test ./..."},
	} {
		got := describeInput(tc.input)
		if got != tc.want || leak(got) != "" {
			t.Errorf("%s: describeInput gave %q (holds %s), want %q", tc.name, got, leak(got), tc.want)
		}
		raw, err := json.Marshal(tc.input)
		if err != nil {
			t.Fatal(err)
		}
		if viaJSON := describeToolCallInput(raw); viaJSON != tc.want {
			t.Errorf("%s: describeToolCallInput gave %q, want %q", tc.name, viaJSON, tc.want)
		}
	}
}

// TestLiveHostileTargetReachesTheScreenClean: entry point Submit with a read_file call
// whose path carries a CSI sequence, an OSC 52 write, a newline and a carriage return,
// through the agent loop and a stream the real anthropic.Decode reads. No such file
// exists, so the tool fails with a message that names the path. The ToolTargetMsg and
// the ToolCompletedMsg the app sends carry the target sanitised (the app-side defence,
// describeInput). A colour-off tui model fed the messages draws no escape and no BEL:
// on the running card's head, once the call has finished (whose summary is the tool's
// message, which only the tui sanitises), and in the modal d opens, whose title names
// the target. Each frame draws the target's visible part "evilred cr.go", so the
// absence is of the raw bytes and not of the card.
func TestLiveHostileTargetReachesTheScreenClean(t *testing.T) {
	repo := newGitRepo(t, map[string]string{"seed.txt": "seed\n"})
	input, err := json.Marshal(map[string]string{"path": hostilePath})
	if err != nil {
		t.Fatal(err)
	}
	e := newLiveEnv(t, repo, nil, sseToolUse(liveCallID, "read_file", string(input)), sseReply("done"))
	msgs := e.turn("read it")
	clean := tui.SanitizeSingleLine(hostilePath)

	target, done := findMsg[tui.ToolTargetMsg](msgs), findMsg[tui.ToolCompletedMsg](msgs)
	if target == nil || done == nil {
		t.Fatalf("want a ToolTargetMsg and a ToolCompletedMsg, got %v", msgTypeNames(msgs))
	}
	if target.Target != clean || leak(target.Target) != "" {
		t.Errorf("the ToolTargetMsg carries %q (holds %s), want %q", target.Target, leak(target.Target), clean)
	}
	if done.Target != clean || leak(done.Target) != "" {
		t.Errorf("the ToolCompletedMsg carries target %q (holds %s), want %q", done.Target, leak(done.Target), clean)
	}
	if done.OK || leak(done.ErrorMsg) == "" {
		t.Fatalf("setup: the read of a missing file should fail with a message holding the raw path, got OK %v message %q", done.OK, done.ErrorMsg)
	}

	upTo := -1
	for i, m := range msgs {
		if _, ok := m.(tui.ToolTargetMsg); ok {
			upTo = i
		}
	}
	running := feed(liveModel(false), toMsgs(msgs[:upTo+1])...).View()
	finished := feed(liveModel(false), toMsgs(msgs)...)
	modal := feed(finished, tea.KeyMsg{Type: tea.KeyShiftUp}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}).View()
	for _, f := range []struct{ name, frame, shows string }{
		{"the running card", running, "read_file " + clean},
		{"the finished card", finished.View(), "read_file " + clean},
		{"the modal", modal, "read_file " + clean},
	} {
		if l := frameLeak(f.frame); l != "" {
			t.Errorf("%s: the colour-off frame holds %s:\n%q", f.name, l, f.frame)
		}
		if !strings.Contains(f.frame, f.shows) {
			t.Errorf("control: %s: the frame does not draw %q, so the absence proves nothing:\n%s", f.name, f.shows, f.frame)
		}
	}
}

// TestLiveHostileToolNameReachesTheScreenClean: entry point Submit with a tool_use block
// whose name carries a screen-clearing CSI, an OSC 52 write and a newline, through the
// real anthropic.Decode. The name is the model's, sent in the start event before any tool
// checked it, and the app passes it on as it came: the ToolStartedMsg holds it raw, so
// the tui's store-time sanitising is the only defence. A colour-off tui model fed the
// messages up to the ToolStartedMsg draws no escape or BEL, and its status bar's busy
// verb is the sanitised name "rogue tool"; once the call has failed (no tool has that
// name) the frame is still clean and the card's head names "rogue tool".
func TestLiveHostileToolNameReachesTheScreenClean(t *testing.T) {
	repo := newGitRepo(t, map[string]string{"main.go": "package main\n"})
	e := newLiveEnv(t, repo, nil, sseToolUse(liveCallID, hostileToolName, `{"path":"main.go"}`), sseReply("done"))
	msgs := e.turn("go")
	start := -1
	for i, m := range msgs {
		if _, ok := m.(tui.ToolStartedMsg); ok {
			start = i
		}
	}
	if start < 0 {
		t.Fatalf("want a ToolStartedMsg, got %v", msgTypeNames(msgs))
	}
	if got := msgs[start].(tui.ToolStartedMsg).Name; got != hostileToolName {
		t.Fatalf("setup: the ToolStartedMsg names %q, want the stream's raw name: the test relies on the tui being the defence", got)
	}
	running := feed(liveModel(false), toMsgs(msgs[:start+1])...).View()
	finished := feed(liveModel(false), toMsgs(msgs)...).View()
	for _, f := range []struct{ name, frame string }{{"running", running}, {"finished", finished}} {
		if l := frameLeak(f.frame); l != "" {
			t.Errorf("%s: the colour-off frame holds %s:\n%q", f.name, l, f.frame)
		}
		if !strings.Contains(f.frame, "rogue tool") {
			t.Errorf("control: %s: the frame does not draw the sanitised name, so the absence proves nothing:\n%s", f.name, f.frame)
		}
	}
}

// TestLiveTextThenToolCallWithAnEmptyFirstFragment: entry point Submit with a stream that
// holds a text block and then a read_file tool_use block whose first input_json_delta is
// empty and whose second holds the whole input. The call still gets its target: the
// ToolTargetMsg names the path, and so does the ToolCompletedMsg, and each comes after
// the reply's text and the ToolStartedMsg. Control: the same call in the plain stream
// (sseToolUse) gives the same target, so the assertion is not met by any non-empty value.
func TestLiveTextThenToolCallWithAnEmptyFirstFragment(t *testing.T) {
	repo := newGitRepo(t, map[string]string{"main.go": "package main\n"})
	const in = `{"path":"main.go"}`
	e := newLiveEnv(t, repo, nil, sseTextThenToolUse("Reading it.", liveCallID, "read_file", in), sseReply("done"))
	msgs := e.turn("read main.go")
	order := msgTypeNames(toolMsgsAndText(msgs))
	want := []string{"tui.AssistantTextDeltaMsg", "tui.ToolStartedMsg", "tui.ToolTargetMsg", "tui.ToolCompletedMsg", "tui.AssistantTextDeltaMsg"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("the turn sent\n got  %v\n want %v", order, want)
	}
	target, done := findMsg[tui.ToolTargetMsg](msgs), findMsg[tui.ToolCompletedMsg](msgs)
	if target == nil || target.Target != "main.go" || done == nil || done.Target != "main.go" {
		t.Errorf("the ToolTargetMsg is %+v and the ToolCompletedMsg %+v, want each to name main.go", target, done)
	}
	plain := liveReadFile(t, "main.go", "package main\n")
	if pt := findMsg[tui.ToolTargetMsg](plain); pt == nil || pt.Target != "main.go" {
		t.Errorf("control: the plain stream's ToolTargetMsg is %+v, want main.go", pt)
	}
}

// toolMsgsAndText keeps the tool messages (toolMsgs) and the text deltas, in order.
func toolMsgsAndText(msgs []any) []any {
	var out []any
	for _, m := range msgs {
		switch m.(type) {
		case tui.ToolStartedMsg, tui.ToolTargetMsg, tui.ApprovalRequestedMsg, tui.ToolCompletedMsg, tui.AssistantTextDeltaMsg:
			out = append(out, m)
		}
	}
	return out
}

// TestLiveNonASCIIPathSurvivesAndIsHighlighted: entry point Submit with a read_file call
// for "src/ïïve file.go", a path with non-ASCII runes and a space, through the real
// anthropic.Decode. sseToolUse splits the input at a rune boundary; the setup checks that
// a split at the byte midpoint would have cut a rune, so the input tests that. The
// ToolTargetMsg and the ToolCompletedMsg carry the path byte for byte, and the colour-on
// modal names it in its title and draws "package" in the keyword colour, so sanitising
// leaves an ordinary path able to drive highlighting by extension.
func TestLiveNonASCIIPathSurvivesAndIsHighlighted(t *testing.T) {
	const path = "src/ïïve file.go"
	in, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	if utf8.ValidString(string(in[:len(in)/2])) {
		t.Fatalf("setup: the byte midpoint of %s does not cut a rune, so this input does not test the split", in)
	}
	if a, b := splitAtRune(string(in)); !utf8.ValidString(a) || !utf8.ValidString(b) || a+b != string(in) {
		t.Fatalf("splitAtRune cut %s into %q and %q", in, a, b)
	}
	msgs := liveReadFile(t, path, "package main\n")
	target, done := findMsg[tui.ToolTargetMsg](msgs), findMsg[tui.ToolCompletedMsg](msgs)
	if target == nil || target.Target != path || done == nil || done.Target != path {
		t.Errorf("the ToolTargetMsg is %+v and the ToolCompletedMsg %+v, want each to carry %q byte for byte", target, done, path)
	}
	frame := modalFrame(toMsgs(msgs))
	if !strings.Contains(frame, "read_file "+path) || !keywordPackage(frame) {
		t.Errorf("the modal: title names the path %v, \"package\" is coloured %v:\n%s",
			strings.Contains(frame, "read_file "+path), keywordPackage(frame), frame)
	}
}

// TestResumedToolCardHeadReachesTheScreenClean: entry points session.Resume and
// a.transcriptToItems, the resume path's builder, then a tui model fed the items in a
// TranscriptLoadedMsg, as ResumeLastSession sends them. The stored session holds a tool
// call whose name and path the model wrote with a CSI sequence, an OSC 52 write and a
// newline. The rebuilt card's target is the sanitised path (describeInput); its name is
// the stored one, unchanged by the app, and the tui sanitises it when it stores the
// item. The colour-off frame and the modal d opens on the card hold no escape or BEL and
// draw "rogue tool" and the target's visible part, so the absence is of the raw bytes.
func TestResumedToolCardHeadReachesTheScreenClean(t *testing.T) {
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()
	call, err := json.Marshal(map[string]any{
		"id": "c1", "tool": hostileToolName, "input": map[string]string{"path": hostilePath},
	})
	if err != nil {
		t.Fatal(err)
	}
	log := `{"v":1,"type":"user.message","content":"hi"}` + "\n" +
		`{"v":1,"type":"assistant.message","content":"","tool_calls":[` + string(call) + `]}` + "\n" +
		`{"v":1,"type":"tool.completed","id":"c1","ok":true,"content":"1\tx\n"}` + "\n"
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	_, tr, err := session.Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	items := a.transcriptToItems(tr)
	var card *tui.ToolCard
	for _, it := range items {
		if it.Kind == tui.KindTool {
			card = it.Tool
		}
	}
	if card == nil {
		t.Fatalf("setup: the resumed items hold no tool card: %+v", items)
	}
	clean := tui.SanitizeSingleLine(hostilePath)
	if card.Target != clean || leak(card.Target) != "" {
		t.Errorf("the resumed card's target is %q (holds %s), want %q", card.Target, leak(card.Target), clean)
	}
	if card.Name != hostileToolName {
		t.Fatalf("setup: the resumed card's name is %q, want the stored raw name: the tui is its defence", card.Name)
	}
	m := feed(liveModel(false), tui.TranscriptLoadedMsg{Items: items})
	frame := m.View()
	modal := feed(m, tea.KeyMsg{Type: tea.KeyShiftUp}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}).View()
	for _, f := range []struct{ name, frame string }{{"transcript", frame}, {"modal", modal}} {
		if l := frameLeak(f.frame); l != "" {
			t.Errorf("%s: the colour-off frame holds %s:\n%q", f.name, l, f.frame)
		}
		if !strings.Contains(f.frame, "rogue tool "+clean) {
			t.Errorf("control: %s: the frame does not draw %q, so the absence proves nothing:\n%s", f.name, "rogue tool "+clean, f.frame)
		}
	}
}
