package app

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/provider/anthropic"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// cutOffView is the notice as the frame draws it for a request sent with the
// unknown-model output limit (provider.UnknownModelMaxOutput, 16384): the card
// draws the leading bullet, the text comes from cutOffNoticeText.
const cutOffView = "· reply cut off — output limit reached (16384 tokens)"

// unknownModel is a model id outside the table, so a turn on it sends 16384.
const unknownModel = "mystery-model-1"

// sessionLogLines returns the parsed lines of the single session log under the
// env's storage directory.
func sessionLogLines(t *testing.T, a *App) []map[string]any {
	t.Helper()
	var found []string
	_ = filepath.WalkDir(a.cfg.Session.StorageDir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".jsonl") {
			found = append(found, p)
		}
		return nil
	})
	if len(found) != 1 {
		t.Fatalf("session logs = %v, want exactly one", found)
	}
	raw, err := os.ReadFile(found[0])
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("log line %q: %v", l, err)
		}
		out = append(out, m)
	}
	return out
}

// liveView feeds everything the app sent during the env's turns into a real TUI
// model and returns its frame.
//
// The outbox delivers a turn's messages in the order RunTurn produced them, and
// delivers the final TurnCompleteMsg after every message produced before it, so
// a snapshot read after TurnCompleteMsg (usageEnv.turn waits for it) holds the
// whole turn, in order, and a frame built from it is the frame the user sees.
// Absence checks below read only after that wait. Entry points that skip
// RunTurn's queue (turnEventMsgs called directly) say so in their own comments.
func liveView(e *usageEnv) string {
	m, _ := frameWith(newFrameModel(), e.c.snapshot(), 0)
	return m.View()
}

// toolCall returns the scripted call to the registered fake_tool.
func toolCall() provider.ToolCall {
	return provider.ToolCall{ID: "call-1", Name: "fake_tool", Input: json.RawMessage(`{}`)}
}

// TestSubmitCutOffTextOnlyShowsNoticeAndRecordsFields drives a turn through
// Submit -> providerModel -> provider.Fake -> RunTurn whose only content is text
// and whose stop reason is max_tokens. The frame holds the notice once, with the
// limit the request carried, and the session log holds both fields on the
// assistant.message line. The frame is built from the snapshot usageEnv.turn
// leaves once TurnCompleteMsg has been delivered; the outbox delivers that
// message after every earlier one, so the notice is already in it.
func TestSubmitCutOffTextOnlyShowsNoticeAndRecordsFields(t *testing.T) {
	e := newUsageEnv(t, unknownModel, provider.Turn{Text: "partial answer", StopReason: provider.StopMaxTokens})
	e.turn("go")

	view := liveView(e)
	if n := strings.Count(view, cutOffView); n != 1 {
		t.Errorf("frame holds the cut-off notice %d times, want 1:\n%s", n, view)
	}
	if !findMsgType[tui.AssistantTextDeltaMsg](e.c.snapshot()) {
		t.Error("control: the turn sent no reply text")
	}
	if got := e.fake.Requests()[0].MaxTokens; got != 16384 {
		t.Errorf("control: request MaxTokens = %d, want 16384", got)
	}

	var assistant map[string]any
	for _, l := range sessionLogLines(t, e.a) {
		if l["type"] == "assistant.message" {
			assistant = l
		}
	}
	if assistant == nil {
		t.Fatal("no assistant.message line in the session log")
	}
	if assistant["stop_reason"] != "max_tokens" || assistant["max_tokens"] != float64(16384) {
		t.Errorf("assistant.message stop_reason/max_tokens = %v/%v, want max_tokens/16384",
			assistant["stop_reason"], assistant["max_tokens"])
	}
}

// TestSubmitCutOffNoticeIsAnchoredToTheFirstToolCard runs, through Submit, a
// reply that has text, then a tool call, and stops on max_tokens. The app sends
// ToolStartedMsg (at the call's start event) before it learns the stop reason
// (at the message-done event), so the NoticeMsg must name that tool's id for the
// TUI to insert it before the card. It asserts the NoticeMsg's BeforeTool is the
// started tool's id and its text and count, and that the ToolStartedMsg is
// delivered before the NoticeMsg (the outbox delivers in the order RunTurn
// produced them; see liveView). The snapshot is read after TurnCompleteMsg.
func TestSubmitCutOffNoticeIsAnchoredToTheFirstToolCard(t *testing.T) {
	e := newUsageEnv(t, unknownModel,
		provider.Turn{Text: "let me look", ToolCalls: []provider.ToolCall{toolCall()}, StopReason: provider.StopMaxTokens},
		provider.Turn{Text: "all done"})
	e.a.reg.Register(fakeTool{})
	e.turn("go")

	msgs := e.c.snapshot()
	started := findMsg[tui.ToolStartedMsg](msgs)
	notice := findMsg[tui.NoticeMsg](msgs)
	if started == nil || notice == nil {
		t.Fatalf("want ToolStartedMsg and NoticeMsg, got %#v", msgs)
	}
	if notice.BeforeTool != started.ID {
		t.Errorf("NoticeMsg.BeforeTool = %d, want the started tool's id %d", notice.BeforeTool, started.ID)
	}
	if indexOfMsg[tui.ToolStartedMsg](msgs) > indexOfMsg[tui.NoticeMsg](msgs) {
		t.Errorf("NoticeMsg delivered before the ToolStartedMsg it names: %#v", msgs)
	}
	if notice.Text != "reply cut off — output limit reached (16384 tokens)" {
		t.Errorf("NoticeMsg.Text = %q", notice.Text)
	}
	if n := countMsgs[tui.NoticeMsg](msgs); n != 1 {
		t.Errorf("NoticeMsg count = %d, want 1 (the second round ended on end_turn)", n)
	}
}

// TestTurnEventMsgsPutNoticeAfterTextAndToolStart calls turnEventMsgs, the
// function RunTurn's emit sends from, with the agent events of a reply that has
// text, then a tool call, then stops on max_tokens, and checks the messages it
// returns in order: text delta, ToolStartedMsg, then the NoticeMsg naming that
// tool. This is the emission order; delivery to the TUI is a separate matter
// (see liveView). Entry point is turnEventMsgs, not RunTurn.
func TestTurnEventMsgsPutNoticeAfterTextAndToolStart(t *testing.T) {
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()

	var textBlock, thinkingBlock int64
	callIDs := map[string]int64{}
	var cutOff cutOffTracker
	var got []tea.Msg
	for _, ev := range []agent.Event{
		{Type: agent.EventTextDelta, Text: "let me look"},
		{Type: agent.EventToolCallStart, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_tool"}},
		{Type: agent.EventToolCallEnd, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_tool"}},
		{Type: agent.EventMessageDone, StopReason: "max_tokens"},
	} {
		got = append(got, a.turnEventMsgs(ev, &textBlock, &thinkingBlock, callIDs, "", &cutOff, 16384)...)
	}

	if len(got) != 3 {
		t.Fatalf("messages = %#v, want text delta, tool started, notice", got)
	}
	if _, ok := got[0].(tui.AssistantTextDeltaMsg); !ok {
		t.Errorf("message 0 = %#v, want AssistantTextDeltaMsg", got[0])
	}
	started, ok := got[1].(tui.ToolStartedMsg)
	if !ok {
		t.Fatalf("message 1 = %#v, want ToolStartedMsg", got[1])
	}
	notice, ok := got[2].(tui.NoticeMsg)
	if !ok {
		t.Fatalf("message 2 = %#v, want NoticeMsg", got[2])
	}
	if notice.BeforeTool != started.ID || notice.Text != "reply cut off — output limit reached (16384 tokens)" {
		t.Errorf("notice = %+v, want text with 16384 and BeforeTool %d", notice, started.ID)
	}
}

// TestTurnEventMsgsAnchorOnlyToTheMessagesOwnFirstTool runs two messages through
// turnEventMsgs: a first that ends in a tool call on tool_use, then a second that
// has text only and stops on max_tokens. The second message's notice must not
// name the first message's tool (BeforeTool 0, so it is appended).
func TestTurnEventMsgsAnchorOnlyToTheMessagesOwnFirstTool(t *testing.T) {
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()

	var textBlock, thinkingBlock int64
	callIDs := map[string]int64{}
	var cutOff cutOffTracker
	var got []tea.Msg
	for _, ev := range []agent.Event{
		{Type: agent.EventToolCallStart, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_tool"}},
		{Type: agent.EventMessageDone, StopReason: "tool_use"},
		{Type: agent.EventTextDelta, Text: "second"},
		{Type: agent.EventMessageDone, StopReason: "max_tokens"},
	} {
		got = append(got, a.turnEventMsgs(ev, &textBlock, &thinkingBlock, callIDs, "", &cutOff, 16384)...)
	}
	notice := findMsg[tui.NoticeMsg](anySlice(got))
	if notice == nil {
		t.Fatalf("no NoticeMsg in %#v", got)
	}
	if notice.BeforeTool != 0 {
		t.Errorf("NoticeMsg.BeforeTool = %d, want 0 (the second message started no tool)", notice.BeforeTool)
	}
	if n := countMsgs[tui.NoticeMsg](anySlice(got)); n != 1 {
		t.Errorf("NoticeMsg count = %d, want 1 (only the max_tokens message)", n)
	}
}

// indexOfMsg returns the index of the first message of type T, or -1.
func indexOfMsg[T any](msgs []any) int {
	for i, m := range msgs {
		if _, ok := m.(T); ok {
			return i
		}
	}
	return -1
}

// anySlice converts messages to the []any the test helpers take.
func anySlice(msgs []tea.Msg) []any {
	out := make([]any, len(msgs))
	for i, m := range msgs {
		out[i] = m
	}
	return out
}

// TestSubmitCutOffWithNoTextAnchorsNoticeToTheToolCard is the empty-reply case
// through Submit: the message has no text, only a tool call, and stops on
// max_tokens. The notice is still sent, and names the tool. Read after
// TurnCompleteMsg, which the outbox delivers after every earlier message.
func TestSubmitCutOffWithNoTextAnchorsNoticeToTheToolCard(t *testing.T) {
	e := newUsageEnv(t, unknownModel,
		provider.Turn{ToolCalls: []provider.ToolCall{toolCall()}, StopReason: provider.StopMaxTokens},
		provider.Turn{Text: "all done"})
	e.a.reg.Register(fakeTool{})
	e.turn("go")

	msgs := e.c.snapshot()
	started, notice := findMsg[tui.ToolStartedMsg](msgs), findMsg[tui.NoticeMsg](msgs)
	if started == nil || notice == nil {
		t.Fatalf("want ToolStartedMsg and NoticeMsg, got %#v", msgs)
	}
	if notice.BeforeTool != started.ID {
		t.Errorf("NoticeMsg.BeforeTool = %d, want %d", notice.BeforeTool, started.ID)
	}
}

// TestSubmitStopReasonDecidesNotice runs the same two-round script through Submit
// with each stop reason on the first round. Only max_tokens sends a NoticeMsg;
// the max_tokens row is the positive control for the three absences, sent
// through the same path. The tool card's ToolStartedMsg is the second control:
// the turn ran. The absence rows read the snapshot after usageEnv.turn has seen
// TurnCompleteMsg, which the outbox delivers after every earlier message of the
// turn, so a NoticeMsg that was going to arrive has arrived.
func TestSubmitStopReasonDecidesNotice(t *testing.T) {
	for _, tc := range []struct {
		stop       provider.StopReason
		wantNotice bool
	}{
		{provider.StopMaxTokens, true},
		{provider.StopEndTurn, false},
		{provider.StopToolUse, false},
		{provider.StopOther, false},
	} {
		t.Run(string(tc.stop), func(t *testing.T) {
			e := newUsageEnv(t, unknownModel,
				provider.Turn{Text: "first round", ToolCalls: []provider.ToolCall{toolCall()}, StopReason: tc.stop},
				provider.Turn{Text: "second round"})
			e.a.reg.Register(fakeTool{})
			e.turn("go")

			msgs := e.c.snapshot()
			if !findMsgType[tui.ToolStartedMsg](msgs) {
				t.Fatalf("control: the turn started no tool: %#v", msgs)
			}
			if got := findMsgType[tui.NoticeMsg](msgs); got != tc.wantNotice {
				t.Errorf("stop reason %q: NoticeMsg sent = %v, want %v", tc.stop, got, tc.wantNotice)
			}
			if got := strings.Contains(liveView(e), "reply cut off"); got != tc.wantNotice {
				t.Errorf("stop reason %q: notice in frame = %v, want %v", tc.stop, got, tc.wantNotice)
			}
		})
	}
}

// TestCutOffNoticeText pins the two texts of spec §3.7: with the count, and the
// count-less form for a stored message that has a stop reason but no limit.
func TestCutOffNoticeText(t *testing.T) {
	if got, want := cutOffNoticeText(16384), "reply cut off — output limit reached (16384 tokens)"; got != want {
		t.Errorf("cutOffNoticeText(16384) = %q, want %q", got, want)
	}
	if got, want := cutOffNoticeText(0), "reply cut off — output limit reached"; got != want {
		t.Errorf("cutOffNoticeText(0) = %q, want %q", got, want)
	}
}

// TestRequestEncodingCarriesNeitherStopReasonNorStoredLimit sends a history whose
// assistant message holds a stop reason and a stored limit (777) through
// providerModel.Stream and the Anthropic encoder, and checks neither appears in
// the request body. The request's own limit (16384) is the positive control: it
// is the one max_tokens in the body.
func TestRequestEncodingCarriesNeitherStopReasonNorStoredLimit(t *testing.T) {
	var captured provider.Request
	pm := &providerModel{
		p:     providerFunc(func(req provider.Request) { captured = req }),
		model: unknownModel,
	}
	history := []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "q"}}},
		{
			Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "a"}},
			StopReason: "max_tokens", MaxTokens: 777,
		},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "go on"}}},
	}
	if err := pm.Stream(t.Context(), agent.Request{Messages: history, MaxTokens: 16384}, func(agent.Event) {}); err != nil {
		t.Fatalf("Stream: %v", err)
	}
	body, err := anthropic.EncodeRequest(captured, anthropic.EncodeOptions{})
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["max_tokens"] != float64(16384) {
		t.Fatalf("control: body max_tokens = %v, want 16384: %s", decoded["max_tokens"], body)
	}
	if strings.Contains(string(body), "stop_reason") {
		t.Errorf("request body carries stop_reason: %s", body)
	}
	if strings.Contains(string(body), "777") {
		t.Errorf("request body carries the stored limit 777: %s", body)
	}
	if n := strings.Count(string(body), "max_tokens"); n != 1 {
		t.Errorf("request body holds max_tokens %d times, want 1: %s", n, body)
	}
}

// providerFunc is a provider.Provider that hands the request to a function and
// emits nothing.
type providerFunc func(provider.Request)

func (f providerFunc) Stream(_ context.Context, req provider.Request, _ func(provider.StreamEvent)) error {
	f(req)
	return nil
}

// assistantPlainMessage is an assistant message of one text block.
func assistantPlainMessage(text string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: text}}}
}

// resumedView writes the given messages to a session log, resumes it through
// App.ResumeLastSession and returns the frame the TUI draws from the messages the
// app sent.
func resumedView(t *testing.T, msgs ...agent.Message) string {
	t.Helper()
	stateRoot := filepath.Join(t.TempDir(), "state")
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")
	wsRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(wsRoot)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := session.NewLayout(stateRoot, ws.CanonicalRoot)
	if err != nil {
		t.Fatal(err)
	}
	path, _, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(path, session.WithCadence(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range msgs {
		s.Record(agent.RecordAppend, m)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	c := newCollector(3) // the notice, the transcript, the session totals
	a := New(ws, cfg, telemetry.Disabled())
	a.sendFn = c.send
	defer a.Close()
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}
	got := c.wait(t, 2*time.Second)
	m, _ := frameWith(newFrameModel(), got, 0)
	return m.View()
}

// TestResumeShowsCutOffNoticeBetweenTextAndToolCard resumes a session whose
// assistant.message has stop_reason max_tokens, text and a tool call, through
// ResumeLastSession. The notice has its count and sits after the text and before
// the tool card, as it did live.
func TestResumeShowsCutOffNoticeBetweenTextAndToolCard(t *testing.T) {
	stopped := assistantTextToolMessage("let me look", "call-1", "read_file", `{"path":"main.go"}`)
	stopped.StopReason, stopped.MaxTokens = "max_tokens", 16384

	view := resumedView(t,
		userTextMessage("plan it"), stopped, userToolResultMessage("call-1", "package main\n", false))

	text, note, card := strings.Index(view, "let me look"), strings.Index(view, cutOffView), strings.Index(view, "read_file")
	if text < 0 || note < 0 || card < 0 {
		t.Fatalf("frame lacks the text (%d), the notice (%d) or the tool card (%d):\n%s", text, note, card, view)
	}
	if text >= note || note >= card {
		t.Errorf("order in frame: text at %d, notice at %d, tool card at %d; want text < notice < card:\n%s",
			text, note, card, view)
	}
}

// TestResumeShowsCountlessNoticeWhenLimitMissing resumes a message with stop
// reason max_tokens and no max_tokens: the notice has no count.
func TestResumeShowsCountlessNoticeWhenLimitMissing(t *testing.T) {
	stopped := assistantPlainMessage("cut here")
	stopped.StopReason = "max_tokens"

	view := resumedView(t, userTextMessage("hi"), stopped)
	if !strings.Contains(view, "· reply cut off — output limit reached") {
		t.Fatalf("frame lacks the notice:\n%s", view)
	}
	if strings.Contains(view, "tokens)") {
		t.Errorf("count-less notice shows a count:\n%s", view)
	}
}

// TestResumeOfOldSessionShowsNoNotice resumes a session whose assistant.message
// carries neither field, the shape written before v0.1.2. The reply text shows
// and no notice does; the max_tokens-stopped session above is the positive
// control for the same path.
func TestResumeOfOldSessionShowsNoNotice(t *testing.T) {
	view := resumedView(t, userTextMessage("hi"), assistantPlainMessage("old reply"))
	if !strings.Contains(view, "old reply") {
		t.Fatalf("control: the frame lacks the reply text:\n%s", view)
	}
	if strings.Contains(view, "reply cut off") {
		t.Errorf("old session shows the cut-off notice:\n%s", view)
	}
}

// TestThinkingOnlyReplyCutOffShowsNoticeLiveButNotOnResume pins a known
// exception to ui-spec §3.7's "the notice is also shown on resume". A reply made
// only of thinking blocks that stops at max_tokens is run through Submit (live:
// the frame holds the notice), then its session log is read back through
// session.Resume and a.transcriptToItems (resume: no notice item). The cause is
// the storage: encodeAssistantAppend writes the stop reason only on an
// assistant.message line, and a thinking-only message writes none. The text-only
// turn in the same session is the positive control for the resume half: the
// same path does give a notice item for a message that has an assistant.message
// line. Entry points: Submit for the live half, session.Resume plus
// transcriptToItems for the resume half (not ResumeLastSession).
func TestThinkingOnlyReplyCutOffShowsNoticeLiveButNotOnResume(t *testing.T) {
	e := newUsageEnv(t, unknownModel,
		provider.Turn{Thinking: []provider.Thinking{{Text: "pondering", Signature: "sig"}}, StopReason: provider.StopMaxTokens},
		provider.Turn{Text: "partial answer", StopReason: provider.StopMaxTokens})

	e.turn("first")
	if !strings.Contains(liveView(e), "reply cut off") {
		t.Fatalf("live: no cut-off notice for the thinking-only reply:\n%s", liveView(e))
	}
	if n := countMsgs[tui.NoticeMsg](e.c.snapshot()); n != 1 {
		t.Fatalf("live: NoticeMsg count = %d, want 1", n)
	}

	var logPath string
	_ = filepath.WalkDir(e.a.cfg.Session.StorageDir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".jsonl") {
			logPath = p
		}
		return nil
	})
	resumedNotices := func() (notices, thinking int) {
		_, tr, err := session.Resume(logPath)
		if err != nil {
			t.Fatalf("Resume: %v", err)
		}
		for _, it := range e.a.transcriptToItems(tr) {
			if it.Kind == tui.KindNotice {
				notices++
			}
			if it.Kind == tui.KindThinking {
				thinking++
			}
		}
		return notices, thinking
	}
	notices, thinking := resumedNotices()
	if thinking != 1 {
		t.Fatalf("control: resumed transcript has %d thinking items, want 1 (the reply was stored)", thinking)
	}
	if notices != 0 {
		t.Errorf("resume: %d notice items for a thinking-only reply, want 0 (no assistant.message line carries its stop reason)", notices)
	}

	e.turn("second")
	if notices, _ = resumedNotices(); notices != 1 {
		t.Errorf("control: resume shows %d notices after a cut-off text reply, want 1", notices)
	}
}

// TestResumeOfMaxTokensWithoutStopReasonShowsNoNotice resumes a session whose
// stored assistant.message has max_tokens but no stop_reason (what is written
// when the provider gave no stop reason) through session.Resume and
// a.transcriptToItems. The notice is decided by the stop reason alone, so none
// is shown. The same line with stop_reason max_tokens added is the positive
// control: one notice item through the same path.
func TestResumeOfMaxTokensWithoutStopReasonShowsNoNotice(t *testing.T) {
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()

	noticeItems := func(assistantLine string) int {
		path := filepath.Join(t.TempDir(), "session.jsonl")
		log := `{"v":1,"type":"user.message","content":"hi"}` + "\n" + assistantLine + "\n"
		if err := os.WriteFile(path, []byte(log), 0o600); err != nil {
			t.Fatal(err)
		}
		_, tr, err := session.Resume(path)
		if err != nil {
			t.Fatalf("Resume: %v", err)
		}
		n := 0
		for _, it := range a.transcriptToItems(tr) {
			if it.Kind == tui.KindNotice {
				n++
			}
		}
		return n
	}
	if n := noticeItems(`{"v":1,"type":"assistant.message","content":"cut","stop_reason":"max_tokens","max_tokens":16384}`); n != 1 {
		t.Fatalf("control: notice items with a stop reason = %d, want 1", n)
	}
	if n := noticeItems(`{"v":1,"type":"assistant.message","content":"cut","max_tokens":16384}`); n != 0 {
		t.Errorf("notice items with max_tokens but no stop reason = %d, want 0", n)
	}
}
