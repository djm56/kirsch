package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// Fakes and helpers.
//
// The fakes copy with deepClone, a reflection-based copier that shares no code
// and no field list with the production copy helpers in agent.go. A bug in the
// production copy therefore cannot hide in a fake that reproduces it.
// ---------------------------------------------------------------------------

// deepClone returns a copy of v that shares no slice or pointer with it. It
// preserves nil slices and nil pointers, and it panics on kinds the agent types
// do not use, so a new field of such a kind fails loudly.
func deepClone[T any](v T) T {
	return cloneValue(reflect.ValueOf(&v).Elem()).Interface().(T)
}

// cloneValue is the recursive step of deepClone.
func cloneValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		p := reflect.New(v.Type().Elem())
		p.Elem().Set(cloneValue(v.Elem()))
		return p
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		s := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			s.Index(i).Set(cloneValue(v.Index(i)))
		}
		return s
	case reflect.Struct:
		s := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			s.Field(i).Set(cloneValue(v.Field(i)))
		}
		return s
	case reflect.Interface, reflect.Map, reflect.Func, reflect.Chan, reflect.Array:
		panic(fmt.Sprintf("deepClone: unsupported kind %s", v.Kind()))
	default:
		return v
	}
}

// dump renders v as indented JSON for failure messages.
func dump(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%#v", v)
	}
	return string(b)
}

// expectEqual fails the test when got and want differ structurally. nil and empty
// slices count as different.
func expectEqual(t *testing.T, label string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s mismatch:\n got: %s\nwant: %s", label, dump(got), dump(want))
	}
}

// fakeModel implements Model by replaying scripted turns.
type fakeModel struct {
	turns     [][]Event
	index     int
	requests  []Request      // a copy of each request, taken before mutateReq runs
	mutateReq func(*Request) // optional hook run on the request the agent passed
	streamErr error          // optional error returned from Stream after the events
}

// newFakeModel creates a fake model with scripted turns.
func newFakeModel(turns ...[]Event) *fakeModel {
	return &fakeModel{turns: turns}
}

// Stream implements Model.Stream.
func (f *fakeModel) Stream(ctx context.Context, req Request, onEvent func(Event)) error {
	if f.index >= len(f.turns) {
		return fmt.Errorf("fake model exhausted: scripted with %d turns, received request %d", len(f.turns), f.index+1)
	}
	f.requests = append(f.requests, deepClone(req))
	if f.mutateReq != nil {
		f.mutateReq(&req)
	}

	turn := f.turns[f.index]
	f.index++
	for _, ev := range turn {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		onEvent(ev)
	}
	return f.streamErr
}

// Info implements Model.Info.
func (f *fakeModel) Info() ModelInfo {
	return ModelInfo{ContextWindow: 200000, MaxOutput: 4096}
}

// fakeTools implements Tools with scripted results.
type fakeTools struct {
	specs    []ToolSpec
	results  map[string]ToolResult
	invoked  []ToolCall      // a copy of each call as Invoke received it
	onInvoke func(*ToolCall) // optional hook run on the call after it is recorded
}

// Describe implements Tools.Describe.
func (f *fakeTools) Describe() []ToolSpec {
	return deepClone(f.specs)
}

// Invoke implements Tools.Invoke.
func (f *fakeTools) Invoke(ctx context.Context, c ToolCall) ToolResult {
	f.invoked = append(f.invoked, deepClone(c))
	if f.onInvoke != nil {
		f.onInvoke(&c)
	}
	if result, ok := f.results[c.Name]; ok {
		return result
	}
	return ToolResult{OK: false, Content: fmt.Sprintf("unknown tool: %s", c.Name), ErrorKind: "tool_input_invalid"}
}

// recEntry is one call to Recorder.Record.
type recEntry struct {
	Kind RecordKind
	Msg  Message
}

// fakeRecorder implements Recorder by keeping an independent copy of each call.
type fakeRecorder struct {
	entries []recEntry
}

// Record implements Recorder.Record.
func (f *fakeRecorder) Record(kind RecordKind, m Message) {
	f.entries = append(f.entries, recEntry{Kind: kind, Msg: deepClone(m)})
}

// mutatingRecorder implements Recorder without copying anything for itself: it
// changes the message it was handed, as a careless session store might.
type mutatingRecorder struct {
	mutate  func(*Message) int
	touched int // total fields the mutations changed
}

// Record implements Recorder.Record.
func (r *mutatingRecorder) Record(kind RecordKind, m Message) {
	r.touched += r.mutate(&m)
}

// rebuild replays recorded entries the way a session store would, following the
// documented Recorder contract, and returns the conversation they describe. It
// returns an error for a record the contract does not allow.
func rebuild(entries []recEntry) ([]Message, error) {
	var msgs []Message
	for i, e := range entries {
		switch e.Kind {
		case RecordAppend:
			msgs = append(msgs, deepClone(e.Msg))
		case RecordMerge:
			if len(msgs) == 0 {
				return nil, fmt.Errorf("record %d is a merge with no message to merge into", i)
			}
			last := &msgs[len(msgs)-1]
			if last.Role != e.Msg.Role {
				return nil, fmt.Errorf("record %d merges a %s message into a %s message", i, e.Msg.Role, last.Role)
			}
			last.Content = append(last.Content, deepClone(e.Msg.Content)...)
		default:
			return nil, fmt.Errorf("record %d has unknown kind %d", i, e.Kind)
		}
	}
	return msgs, nil
}

// expectRebuilds fails the test unless replaying the recorder's entries rebuilds
// exactly the harness conversation.
func expectRebuilds(t *testing.T, h *harness) {
	t.Helper()
	got, err := rebuild(h.rec.entries)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	expectEqual(t, "rebuilt conversation", got, h.conv.Messages)
}

// harness bundles an Agent with its fakes and a conversation.
type harness struct {
	model   *fakeModel
	tools   *fakeTools
	rec     *fakeRecorder
	agent   *Agent
	conv    *Conversation
	emitted []Event
}

// newHarness builds a harness whose model replays turns and whose tools offer
// read_file (see fullTools).
func newHarness(turns ...[]Event) *harness {
	h := &harness{
		model: newFakeModel(turns...),
		tools: fullTools(),
		rec:   &fakeRecorder{},
		conv:  &Conversation{},
	}
	h.agent = &Agent{Model: h.model, Tools: h.tools, Recorder: h.rec, MaxTokens: 1024}
	return h
}

// turn runs one Turn on the harness conversation, capturing emitted events.
func (h *harness) turn(text string) error {
	return h.agent.Turn(context.Background(), h.conv, text, func(ev Event) {
		h.emitted = append(h.emitted, ev)
	})
}

// mustTurn runs a Turn and fails the test on error.
func (h *harness) mustTurn(t *testing.T, text string) {
	t.Helper()
	if err := h.turn(text); err != nil {
		t.Fatalf("Turn(%q) failed: %v", text, err)
	}
}

// message returns conversation message i, failing the test if there is none.
func (h *harness) message(t *testing.T, i int) Message {
	t.Helper()
	if i >= len(h.conv.Messages) {
		t.Fatalf("conversation has %d messages, want at least %d", len(h.conv.Messages), i+1)
	}
	return h.conv.Messages[i]
}

// request returns the model's request i, failing the test if there is none.
func (h *harness) request(t *testing.T, i int) Request {
	t.Helper()
	if i >= len(h.model.requests) {
		t.Fatalf("model got %d requests, want at least %d", len(h.model.requests), i+1)
	}
	return h.model.requests[i]
}

// invokedCall returns the tool call Invoke received as call i, failing the test
// if there is none.
func (h *harness) invokedCall(t *testing.T, i int) ToolCall {
	t.Helper()
	if i >= len(h.tools.invoked) {
		t.Fatalf("Invoke ran %d times, want at least %d", len(h.tools.invoked), i+1)
	}
	return h.tools.invoked[i]
}

// firstN returns the first n of msgs, failing the test if there are fewer.
func firstN(t *testing.T, msgs []Message, n int) []Message {
	t.Helper()
	if len(msgs) < n {
		t.Fatalf("got %d messages, want at least %d: %s", len(msgs), n, dump(msgs))
	}
	return msgs[:n]
}

// blockAt returns blocks[i], failing the test if there is none.
func blockAt(t *testing.T, blocks []Block, i int) Block {
	t.Helper()
	if i >= len(blocks) {
		t.Fatalf("got %d blocks, want at least %d: %s", len(blocks), i+1, dump(blocks))
	}
	return blocks[i]
}

// appendEntries returns the recEntries for msgs, all RecordAppend.
func appendEntries(msgs []Message) []recEntry {
	out := make([]recEntry, len(msgs))
	for i, m := range msgs {
		out[i] = recEntry{Kind: RecordAppend, Msg: m}
	}
	return out
}

// Event constructors.

func textEv(s string) Event     { return Event{Type: EventTextDelta, Text: s} }
func thinkDelta(s string) Event { return Event{Type: EventThinkingDelta, Text: s} }
func msgDone() Event            { return Event{Type: EventMessageDone} }
func errEv(err error) Event     { return Event{Type: EventError, Err: err} }
func thinkDone(t Thinking) Event {
	return Event{Type: EventThinkingDone, Thinking: &t}
}

// callEnd returns an EventToolCallEnd; an empty input leaves Input nil.
func callEnd(id, name, input string) Event {
	var in json.RawMessage
	if input != "" {
		in = json.RawMessage(input)
	}
	return Event{Type: EventToolCallEnd, ToolCall: &ToolCall{ID: id, Name: name, Input: in}}
}

// fullTools offers read_file, which succeeds with a truncated "file body".
func fullTools() *fakeTools {
	return &fakeTools{
		specs: []ToolSpec{{Name: "read_file", Description: "Read a file", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		results: map[string]ToolResult{
			"read_file": {OK: true, Content: "file body", Truncated: true},
		},
	}
}

// fullRound1 is a model message with every accumulated block kind: a thinking
// block built from deltas and closed by Done, a redacted block that is only a
// Done, text from two deltas, and one tool call (with Start and Delta events).
func fullRound1() []Event {
	return []Event{
		thinkDelta("thi"),
		thinkDelta("nk"),
		thinkDone(Thinking{Text: "think", Signature: "sig-1"}),
		thinkDone(Thinking{Redacted: true, Data: "opaque", Signature: "sig-2"}),
		textEv("let me "),
		textEv("look"),
		{Type: EventToolCallStart, ToolCall: &ToolCall{ID: "call-1", Name: "read_file"}},
		{Type: EventToolCallDelta, Text: `{"path":`},
		{Type: EventToolCallDelta, Text: `"a.txt"}`},
		callEnd("call-1", "read_file", `{"path":"a.txt"}`),
		msgDone(),
	}
}

// fullRound2 is the model's final answer.
func fullRound2() []Event {
	return []Event{textEv("do"), textEv("ne"), {Type: EventMessageDone, Usage: &Usage{InputTokens: 1}}}
}

// wantConv is the conversation fullRound1 and fullRound2 must produce for the
// user text "Read a.txt", written out by hand.
func wantConv() []Message {
	return []Message{
		{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "Read a.txt"}}},
		{Role: RoleAssistant, Content: []Block{
			{Kind: BlockThinking, Thinking: &Thinking{Text: "think", Signature: "sig-1"}},
			{Kind: BlockThinking, Thinking: &Thinking{Redacted: true, Data: "opaque", Signature: "sig-2"}},
			{Kind: BlockText, Text: "let me look"},
			{Kind: BlockToolCall, ToolCall: &ToolCall{ID: "call-1", Name: "read_file", Input: json.RawMessage(`{"path":"a.txt"}`)}},
		}},
		{Role: RoleUser, Content: []Block{
			{Kind: BlockToolResult, Result: &ResultBlock{CallID: "call-1", Content: "file body", Truncated: true}},
		}},
		{Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "done"}}},
	}
}

// runFull runs the full two-round conversation on a fresh harness.
func runFull(t *testing.T) *harness {
	t.Helper()
	h := newHarness(fullRound1(), fullRound2())
	h.mustTurn(t, "Read a.txt")
	return h
}

// ---------------------------------------------------------------------------
// Conversation, recorder, requests, tools.
// ---------------------------------------------------------------------------

// TestConversationAfterToolRound checks the whole conversation, block by block,
// for a turn with one tool round.
func TestConversationAfterToolRound(t *testing.T) {
	h := runFull(t)
	expectEqual(t, "conversation", h.conv.Messages, wantConv())
}

// TestRecorderGetsEveryAppendedMessage checks the Recorder saw exactly the four
// messages of the full turn, in order, all as RecordAppend, and that they are
// the conversation.
func TestRecorderGetsEveryAppendedMessage(t *testing.T) {
	h := runFull(t)
	expectEqual(t, "recorded entries", h.rec.entries, appendEntries(wantConv()))
}

// TestRecorderTextOnly checks the text-only case: user message, then assistant
// text accumulated from two deltas.
func TestRecorderTextOnly(t *testing.T) {
	h := newHarness([]Event{textEv("Hello "), textEv("world"), msgDone()})
	h.mustTurn(t, "Hello")
	want := []Message{
		{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "Hello"}}},
		{Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "Hello world"}}},
	}
	expectEqual(t, "recorded entries", h.rec.entries, appendEntries(want))
	expectEqual(t, "conversation", h.conv.Messages, want)
}

// TestRequestsCarryHistory checks each round's request: the history so far, the
// tool specs from Describe, and MaxTokens. The second request must hold the first
// round's tool results, so a request built once outside the loop fails here.
func TestRequestsCarryHistory(t *testing.T) {
	h := runFull(t)
	if len(h.model.requests) != 2 {
		t.Fatalf("want 2 requests, got %d", len(h.model.requests))
	}
	want := wantConv()
	expectEqual(t, "request 0 messages", h.request(t, 0).Messages, want[:1])
	expectEqual(t, "request 1 messages", h.request(t, 1).Messages, want[:3])
	for i, req := range h.model.requests {
		expectEqual(t, fmt.Sprintf("request %d tools", i), req.Tools, h.tools.specs)
		if req.MaxTokens != 1024 {
			t.Fatalf("request %d MaxTokens: want 1024, got %d", i, req.MaxTokens)
		}
	}
}

// TestInvokeReceivesCall checks Invoke is called once with the call's ID, Name
// and Input exactly as the model's ToolCallEnd carried them.
func TestInvokeReceivesCall(t *testing.T) {
	h := runFull(t)
	want := []ToolCall{{ID: "call-1", Name: "read_file", Input: json.RawMessage(`{"path":"a.txt"}`)}}
	expectEqual(t, "invoked calls", h.tools.invoked, want)
}

// TestEmitForwarding checks that on a successful two-round turn emit receives
// every event of both rounds, in arrival order, unchanged, and that a nil emit
// does not panic. Error-path forwarding is tested in TestErrorPaths.
func TestEmitForwarding(t *testing.T) {
	h := runFull(t)
	want := append(fullRound1(), fullRound2()...)
	expectEqual(t, "emitted events", h.emitted, want)

	t.Run("nil_emit_does_not_panic", func(t *testing.T) {
		quiet := newHarness(fullRound1(), fullRound2())
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Turn with a nil emit panicked: %v", r)
			}
		}()
		if err := quiet.agent.Turn(context.Background(), quiet.conv, "Read a.txt", nil); err != nil {
			t.Fatalf("Turn with nil emit failed: %v", err)
		}
		expectEqual(t, "conversation with nil emit", quiet.conv.Messages, wantConv())
	})
}

// TestNilRecorderIsAllowed checks a nil Recorder is tolerated, on every path that
// records: a new user message, a merge, an assistant message and tool results.
func TestNilRecorderIsAllowed(t *testing.T) {
	h := newHarness(fullRound1(), []Event{errEv(errors.New("boom"))}, fullRound2())
	h.agent.Recorder = nil
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Turn with a nil Recorder panicked: %v", r)
		}
	}()
	if err := h.turn("Read a.txt"); err == nil {
		t.Fatal("first Turn returned nil")
	}
	h.mustTurn(t, "again") // merges into the kept tool-results message
	if got := len(h.conv.Messages); got != 4 {
		t.Fatalf("want 4 messages, got %d: %s", got, dump(h.conv.Messages))
	}
}

// TestBlockOrder checks that text, a tool call, and more text keep their order
// and that text after the tool call is a new block.
func TestBlockOrder(t *testing.T) {
	h := newHarness(
		[]Event{textEv("before"), callEnd("c1", "read_file", `{}`), textEv("after"), msgDone()},
		[]Event{textEv("final"), msgDone()},
	)
	h.mustTurn(t, "test")
	want := []Block{
		{Kind: BlockText, Text: "before"},
		{Kind: BlockToolCall, ToolCall: &ToolCall{ID: "c1", Name: "read_file", Input: json.RawMessage(`{}`)}},
		{Kind: BlockText, Text: "after"},
	}
	expectEqual(t, "assistant blocks", h.message(t, 1).Content, want)
}

// TestTwoToolCallsInOrder checks two calls in one message are invoked in order,
// and that the one results message holds a distinct result for each, in order.
func TestTwoToolCallsInOrder(t *testing.T) {
	h := newHarness(
		[]Event{
			callEnd("call-1", "search_code", `{"query":"test"}`),
			callEnd("call-2", "list_files", `{"path":"*.go"}`),
			msgDone(),
		},
		[]Event{textEv("Done"), msgDone()},
	)
	h.tools.results = map[string]ToolResult{
		"search_code": {OK: true, Content: "found test"},
		"list_files":  {OK: true, Content: "file1.go file2.go"},
	}
	h.mustTurn(t, "Search and list")

	wantCalls := []ToolCall{
		{ID: "call-1", Name: "search_code", Input: json.RawMessage(`{"query":"test"}`)},
		{ID: "call-2", Name: "list_files", Input: json.RawMessage(`{"path":"*.go"}`)},
	}
	expectEqual(t, "invoked calls", h.tools.invoked, wantCalls)

	wantResults := []Block{
		{Kind: BlockToolResult, Result: &ResultBlock{CallID: "call-1", Content: "found test"}},
		{Kind: BlockToolResult, Result: &ResultBlock{CallID: "call-2", Content: "file1.go file2.go"}},
	}
	expectEqual(t, "results message", firstN(t, h.request(t, 1).Messages, 3)[2].Content, wantResults)
}

// TestErrorToolResultFields checks that a failed tool result carries its Content,
// ErrorKind and Truncated flag, with IsError set.
func TestErrorToolResultFields(t *testing.T) {
	h := newHarness(
		[]Event{callEnd("call-1", "read_file", `{"path":"/nonexistent"}`), msgDone()},
		[]Event{textEv("File not found"), msgDone()},
	)
	h.tools.results = map[string]ToolResult{
		"read_file": {OK: false, Content: "file not found", ErrorKind: "file_not_found", Truncated: true},
	}
	h.mustTurn(t, "Read /nonexistent")

	want := []Block{{Kind: BlockToolResult, Result: &ResultBlock{
		CallID: "call-1", Content: "file not found", IsError: true, ErrorKind: "file_not_found", Truncated: true,
	}}}
	expectEqual(t, "results message", h.message(t, 2).Content, want)
}

// TestToolInputNilAndEmptyKeepShape checks a nil Input stays nil and an empty
// non-nil Input stays empty and non-nil, in what Invoke receives, in the
// conversation and in the next request.
func TestToolInputNilAndEmptyKeepShape(t *testing.T) {
	cases := []struct {
		name  string
		input json.RawMessage
	}{
		{"nil", nil},
		{"empty", json.RawMessage{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			end := Event{Type: EventToolCallEnd, ToolCall: &ToolCall{ID: "c1", Name: "read_file", Input: c.input}}
			h := newHarness([]Event{end, msgDone()}, []Event{textEv("ok"), msgDone()})
			h.mustTurn(t, "go")

			want := ToolCall{ID: "c1", Name: "read_file", Input: c.input}
			expectEqual(t, "invoked call", h.tools.invoked, []ToolCall{want})
			expectEqual(t, "stored tool call", blockAt(t, h.message(t, 1).Content, 0).ToolCall, &want)
			expectEqual(t, "requested tool call", blockAt(t, firstN(t, h.request(t, 1).Messages, 2)[1].Content, 0).ToolCall, &want)
		})
	}
}

// ---------------------------------------------------------------------------
// Thinking blocks.
// ---------------------------------------------------------------------------

// TestThinkingBlocks checks how thinking blocks accumulate: a block of deltas
// closed by Done takes Done's data; two consecutive blocks stay two blocks; a
// Done with no deltas makes a block; a Done on an open block takes Done's
// Redacted and Data, and Done's Text replaces the deltas' text; a tool call that
// ends right after a closed thinking block is accepted.
func TestThinkingBlocks(t *testing.T) {
	cases := []struct {
		name   string
		events []Event
		want   []Block
	}{
		{
			name: "deltas_then_done",
			events: []Event{
				thinkDelta("thinking "), thinkDelta("more"),
				thinkDone(Thinking{Text: "thinking more", Signature: "sig1"}),
				textEv("answer"), msgDone(),
			},
			want: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Text: "thinking more", Signature: "sig1"}},
				{Kind: BlockText, Text: "answer"},
			},
		},
		{
			name: "consecutive_blocks",
			events: []Event{
				thinkDelta("first"), thinkDone(Thinking{Text: "first", Signature: "sig1"}),
				thinkDelta("second"), thinkDone(Thinking{Text: "second", Signature: "sig2"}),
				textEv("answer"), msgDone(),
			},
			want: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Text: "first", Signature: "sig1"}},
				{Kind: BlockThinking, Thinking: &Thinking{Text: "second", Signature: "sig2"}},
				{Kind: BlockText, Text: "answer"},
			},
		},
		{
			name: "redacted_done_only_then_normal",
			events: []Event{
				thinkDone(Thinking{Signature: "sig1", Redacted: true, Data: "opaque-content"}),
				thinkDelta("visible"), thinkDone(Thinking{Text: "visible", Signature: "sig2"}),
				textEv("answer"), msgDone(),
			},
			want: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Signature: "sig1", Redacted: true, Data: "opaque-content"}},
				{Kind: BlockThinking, Thinking: &Thinking{Text: "visible", Signature: "sig2"}},
				{Kind: BlockText, Text: "answer"},
			},
		},
		{
			name: "done_on_open_block_takes_redacted_and_data",
			events: []Event{
				thinkDelta("partial"),
				thinkDone(Thinking{Signature: "sig", Redacted: true, Data: "opaque"}),
				textEv("answer"), msgDone(),
			},
			want: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Signature: "sig", Redacted: true, Data: "opaque"}},
				{Kind: BlockText, Text: "answer"},
			},
		},
		{
			name: "tool_call_end_right_after_closed_thinking",
			events: []Event{
				thinkDelta("a"), thinkDone(Thinking{Text: "a", Signature: "sig"}),
				callEnd("c1", "read_file", `{}`), msgDone(),
			},
			want: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Text: "a", Signature: "sig"}},
				{Kind: BlockToolCall, ToolCall: &ToolCall{ID: "c1", Name: "read_file", Input: json.RawMessage(`{}`)}},
			},
		},
		{
			name: "done_text_replaces_delta_text",
			events: []Event{
				thinkDelta("partial"),
				thinkDone(Thinking{Text: "full", Signature: "sig"}),
				textEv("answer"), msgDone(),
			},
			want: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Text: "full", Signature: "sig"}},
				{Kind: BlockText, Text: "answer"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(c.events, []Event{textEv("ok"), msgDone()}) // the second round answers a tool call
			h.mustTurn(t, "test")
			expectEqual(t, "assistant blocks", h.message(t, 1).Content, c.want)
			wantRequests := 1
			for _, b := range c.want {
				if b.Kind == BlockToolCall {
					wantRequests = 2 // a tool call is answered by a second request
				}
			}
			if len(h.model.requests) != wantRequests {
				t.Fatalf("model was called %d times, want %d", len(h.model.requests), wantRequests)
			}
		})
	}
}

// TestThinkingOnlyMessageIsAccepted checks that an assistant message made only of
// one closed thinking block is valid: it is appended and recorded, and with no
// tool calls it ends the Turn after one request. (Nothing in plan/spec or
// milestone 3 rules such a message out; the agent requires only one block.)
func TestThinkingOnlyMessageIsAccepted(t *testing.T) {
	h := newHarness([]Event{
		thinkDelta("t"), thinkDone(Thinking{Text: "t", Signature: "sig"}), msgDone(),
	})
	h.mustTurn(t, "go")
	want := []Message{
		{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "go"}}},
		{Role: RoleAssistant, Content: []Block{{Kind: BlockThinking, Thinking: &Thinking{Text: "t", Signature: "sig"}}}},
	}
	expectEqual(t, "conversation", h.conv.Messages, want)
	expectEqual(t, "recorded entries", h.rec.entries, appendEntries(want))
	if len(h.model.requests) != 1 {
		t.Fatalf("model was called %d times, want 1", len(h.model.requests))
	}
}

// ---------------------------------------------------------------------------
// Errors, forwarding and first-error-wins.
// ---------------------------------------------------------------------------

// TestErrorPaths drives every error path of a round and checks, for each, the
// exact error text, errors.Is, the exact events that reached emit, and that the
// conversation and the Recorder hold only the user message: nothing is rolled
// back, the partial assistant message is not appended, and no tool ran.
// The cases from "violation_then_later_violation" on feed more events, or a Stream
// error, after the first error, to check the first error wins and that nothing
// after it is forwarded.
func TestErrorPaths(t *testing.T) {
	errModel := errors.New("model error")
	errOther := errors.New("other model error")
	errConn := errors.New("connection lost")
	proto := func(reason string) string { return ErrStreamProtocol.Error() + ": " + reason }
	nilToolCall := Event{Type: EventToolCallEnd}
	nilThinking := Event{Type: EventThinkingDone}
	const textWhileOpen = "text delta while thinking block is open"

	cases := []struct {
		name      string
		events    []Event
		streamErr error
		want      string // exact err.Error()
		is        error  // errors.Is must hold, if set
		isNot     error  // errors.Is must not hold, if set
		emitted   []Event
	}{
		{
			name: "stream_returns_error", events: []Event{textEv("partial"), msgDone()}, streamErr: errConn,
			want: "agent: model stream: connection lost", is: errConn,
			emitted: []Event{textEv("partial"), msgDone()},
		},
		{
			name: "stream_returns_error_no_events", events: []Event{}, streamErr: errConn,
			want: "agent: model stream: connection lost", is: errConn,
		},
		{
			name: "event_error_with_err", events: []Event{textEv("text"), errEv(errModel)},
			want: "agent: model stream: model error", is: errModel,
			emitted: []Event{textEv("text"), errEv(errModel)},
		},
		{
			name: "event_error_without_err", events: []Event{textEv("text"), errEv(nil)},
			want: ErrStreamFailed.Error(), is: ErrStreamFailed,
			emitted: []Event{textEv("text"), errEv(nil)},
		},
		{
			name: "no_message_done", events: []Event{textEv("text")},
			want: proto("no MessageDone event received"), is: ErrStreamProtocol,
			emitted: []Event{textEv("text")},
		},
		{
			name: "event_after_message_done", events: []Event{textEv("text"), msgDone(), textEv("extra")},
			want: proto("event received after MessageDone"), is: ErrStreamProtocol,
			emitted: []Event{textEv("text"), msgDone()},
		},
		{
			name: "event_error_after_message_done", events: []Event{textEv("text"), msgDone(), errEv(errModel)},
			want: proto("event received after MessageDone"), is: ErrStreamProtocol, isNot: errModel,
			emitted: []Event{textEv("text"), msgDone()},
		},
		{
			name: "second_message_done", events: []Event{textEv("text"), msgDone(), msgDone()},
			want: proto("event received after MessageDone"), is: ErrStreamProtocol,
			emitted: []Event{textEv("text"), msgDone()},
		},
		{
			name: "empty_assistant_message", events: []Event{msgDone()},
			want: proto("assistant message has no content blocks"), is: ErrStreamProtocol,
			emitted: []Event{msgDone()},
		},
		{
			name: "tool_call_end_nil_tool_call", events: []Event{nilToolCall, msgDone()},
			want: proto("EventToolCallEnd with nil ToolCall"), is: ErrStreamProtocol,
		},
		{
			name: "thinking_done_nil_thinking", events: []Event{nilThinking, msgDone()},
			want: proto("EventThinkingDone with nil Thinking"), is: ErrStreamProtocol,
		},
		{
			name: "thinking_not_closed_at_message_done", events: []Event{thinkDelta("thinking"), msgDone()},
			want: proto("thinking block not closed at MessageDone"), is: ErrStreamProtocol,
			emitted: []Event{thinkDelta("thinking")},
		},
		{
			name: "text_delta_while_thinking_open", events: []Event{thinkDelta("thinking"), textEv("text"), msgDone()},
			want: proto(textWhileOpen), is: ErrStreamProtocol,
			emitted: []Event{thinkDelta("thinking")},
		},
		{
			name: "tool_call_end_while_thinking_open", events: []Event{thinkDelta("thinking"), callEnd("c1", "read_file", `{}`), msgDone()},
			want: proto("tool call end while thinking block is open"), is: ErrStreamProtocol,
			emitted: []Event{thinkDelta("thinking")},
		},
		{
			name:   "violation_then_later_violation",
			events: []Event{thinkDelta("a"), textEv("x"), callEnd("c1", "read_file", `{}`), nilThinking, msgDone()},
			want:   proto(textWhileOpen), is: ErrStreamProtocol,
			emitted: []Event{thinkDelta("a")},
		},
		{
			name:   "violation_then_event_error",
			events: []Event{thinkDelta("a"), textEv("x"), errEv(errModel)},
			want:   proto(textWhileOpen), isNot: errModel,
			emitted: []Event{thinkDelta("a")},
		},
		{
			name:   "event_error_then_event_error",
			events: []Event{errEv(errModel), errEv(errOther)},
			want:   "agent: model stream: model error", is: errModel, isNot: errOther,
			emitted: []Event{errEv(errModel)},
		},
		{
			name:   "event_error_then_violation",
			events: []Event{textEv("x"), errEv(errModel), nilThinking},
			want:   "agent: model stream: model error", is: errModel, isNot: ErrStreamProtocol,
			emitted: []Event{textEv("x"), errEv(errModel)},
		},
		{
			name:   "event_error_then_valid_events",
			events: []Event{errEv(errModel), textEv("late"), msgDone()},
			want:   "agent: model stream: model error", is: errModel,
			emitted: []Event{errEv(errModel)},
		},
		{
			name:   "event_error_then_stream_error",
			events: []Event{textEv("x"), errEv(errModel)}, streamErr: errConn,
			want: "agent: model stream: model error", is: errModel, isNot: errConn,
			emitted: []Event{textEv("x"), errEv(errModel)},
		},
		{
			name:   "violation_then_stream_error",
			events: []Event{thinkDelta("a"), textEv("x")}, streamErr: errConn,
			want: proto(textWhileOpen), is: ErrStreamProtocol, isNot: errConn,
			emitted: []Event{thinkDelta("a")},
		},
		{
			name:   "event_after_message_done_then_stream_error",
			events: []Event{textEv("x"), msgDone(), textEv("extra")}, streamErr: errConn,
			want: proto("event received after MessageDone"), is: ErrStreamProtocol, isNot: errConn,
			emitted: []Event{textEv("x"), msgDone()},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness(c.events)
			h.model.streamErr = c.streamErr
			err := h.turn("test")

			if err == nil {
				t.Fatalf("%s: Turn returned nil", c.name)
			}
			if err.Error() != c.want {
				t.Fatalf("%s: error text:\n got: %q\nwant: %q", c.name, err.Error(), c.want)
			}
			if c.is != nil && !errors.Is(err, c.is) {
				t.Fatalf("%s: error %q does not wrap %q", c.name, err, c.is)
			}
			if c.isNot != nil && errors.Is(err, c.isNot) {
				t.Fatalf("%s: error %q wraps %q, which must lose", c.name, err, c.isNot)
			}
			if len(h.emitted) != len(c.emitted) || (len(c.emitted) > 0 && !reflect.DeepEqual(h.emitted, c.emitted)) {
				t.Fatalf("%s: emitted events mismatch:\n got: %s\nwant: %s", c.name, dump(h.emitted), dump(c.emitted))
			}

			user := []Message{{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "test"}}}}
			expectEqual(t, c.name+": conversation", h.conv.Messages, user)
			expectEqual(t, c.name+": recorded entries", h.rec.entries, appendEntries(user))
			if len(h.tools.invoked) != 0 {
				t.Fatalf("%s: a tool ran on a failed round: %v", c.name, h.tools.invoked)
			}
			if len(h.model.requests) != 1 {
				t.Fatalf("%s: model was called %d times, want 1", c.name, len(h.model.requests))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ownership: who may change what.
// ---------------------------------------------------------------------------

// depthMutation changes one depth of a message in place and reports how many
// fields it changed.
type depthMutation struct {
	name  string
	apply func(m *Message) int
}

// depthMutations covers each level a shallow copy could share: a Content
// element, a Thinking, a ToolCall's fields and Input bytes, and a ResultBlock.
var depthMutations = []depthMutation{
	{"content_element", func(m *Message) int {
		if len(m.Content) == 0 {
			return 0
		}
		m.Content[0] = Block{Kind: BlockText, Text: "MUTATED"}
		return 1
	}},
	{"thinking", func(m *Message) int {
		n := 0
		for _, b := range m.Content {
			if b.Thinking != nil {
				b.Thinking.Text, b.Thinking.Signature, b.Thinking.Data = "MUTATED", "MUTATED", "MUTATED"
				n++
			}
		}
		return n
	}},
	{"tool_call_name", func(m *Message) int {
		n := 0
		for _, b := range m.Content {
			if b.ToolCall != nil {
				b.ToolCall.Name = "MUTATED"
				n++
			}
		}
		return n
	}},
	{"tool_call_input", func(m *Message) int {
		n := 0
		for _, b := range m.Content {
			if b.ToolCall != nil && len(b.ToolCall.Input) > 0 {
				b.ToolCall.Input[0] = 'X'
				n++
			}
		}
		return n
	}},
	{"result", func(m *Message) int {
		n := 0
		for _, b := range m.Content {
			if b.Result != nil {
				b.Result.Content, b.Result.IsError = "MUTATED", true
				n++
			}
		}
		return n
	}},
}

// requireEffective is the positive control for depthMutations: the mutation must
// change a copy of wantConv, otherwise "the conversation survived it" proves
// nothing.
func requireEffective(t *testing.T, d depthMutation) {
	t.Helper()
	touched := 0
	for _, m := range wantConv() {
		c := deepClone(m)
		if d.apply(&c) > 0 {
			touched++
			if reflect.DeepEqual(c, m) {
				t.Fatalf("mutation %s reports a change but the message is equal", d.name)
			}
		}
	}
	if touched == 0 {
		t.Fatalf("mutation %s touches no message of the conversation", d.name)
	}
}

// TestRequestIsDeepCopyOfConversation mutates the request the model receives, at
// each depth, and checks the conversation does not change. The conversation is
// seeded with every block kind, so each mutation has something to hit.
func TestRequestIsDeepCopyOfConversation(t *testing.T) {
	run := func(t *testing.T, name string, hook func(*Request) int) {
		h := newHarness([]Event{textEv("reply"), msgDone()})
		h.conv.Messages = wantConv()
		mutated := 0
		h.model.mutateReq = func(r *Request) { mutated += hook(r) }
		h.mustTurn(t, "next")
		if mutated == 0 {
			t.Fatalf("%s: the hook changed nothing", name)
		}
		expectEqual(t, name+": conversation after mutating the request", firstN(t, h.conv.Messages, 4), wantConv())
	}

	for _, d := range depthMutations {
		t.Run(d.name, func(t *testing.T) {
			requireEffective(t, d)
			run(t, d.name, func(r *Request) int {
				n := 0
				for i := range r.Messages {
					n += d.apply(&r.Messages[i])
				}
				return n
			})
		})
	}
	t.Run("messages_slice_element", func(t *testing.T) {
		run(t, "messages_slice_element", func(r *Request) int {
			r.Messages[0] = Message{Role: RoleAssistant}
			return 1
		})
	})
}

// TestRecorderReceivesCopies gives the agent a Recorder that keeps no copy of its
// own and mutates what it is handed, at each depth, and checks neither the
// conversation nor the second request changes.
func TestRecorderReceivesCopies(t *testing.T) {
	for _, d := range depthMutations {
		t.Run(d.name, func(t *testing.T) {
			requireEffective(t, d)
			h := newHarness(fullRound1(), fullRound2())
			rec := &mutatingRecorder{mutate: d.apply}
			h.agent.Recorder = rec
			h.mustTurn(t, "Read a.txt")
			if rec.touched == 0 {
				t.Fatalf("%s: the recorder changed nothing", d.name)
			}
			expectEqual(t, d.name+": conversation after the recorder mutated its messages", h.conv.Messages, wantConv())
			expectEqual(t, d.name+": second request after the recorder mutated its messages", h.request(t, 1).Messages, wantConv()[:3])
		})
	}
}

// TestInvokeReceivesCopy has the tool overwrite its Input in place, and checks the
// conversation, the next request and what Invoke was first given are unchanged.
func TestInvokeReceivesCopy(t *testing.T) {
	h := newHarness(fullRound1(), fullRound2())
	calls := 0
	h.tools.onInvoke = func(c *ToolCall) {
		calls++
		if len(c.Input) > 0 {
			c.Input[0] = 'X'
		}
	}
	h.mustTurn(t, "Read a.txt")
	if calls != 1 {
		t.Fatalf("Invoke ran %d times, want 1", calls)
	}
	expectEqual(t, "conversation after the tool overwrote its Input", h.conv.Messages, wantConv())
	expectEqual(t, "second request after the tool overwrote its Input", h.request(t, 1).Messages, wantConv()[:3])
	expectEqual(t, "input Invoke was given", h.invokedCall(t, 0).Input, json.RawMessage(`{"path":"a.txt"}`))
}

// emitFixtures returns a round whose emitted Thinking (on an open block and on
// none) and ToolCall structs the test keeps pointers to, and the blocks the round
// must produce.
func emitFixtures() (round []Event, open, bare *Thinking, call *ToolCall, want []Block) {
	open = &Thinking{Text: "partial done", Signature: "sig-a"}
	bare = &Thinking{Redacted: true, Data: "opaque", Signature: "sig-b"}
	call = &ToolCall{ID: "c1", Name: "read_file", Input: json.RawMessage(`{"path":"a.txt"}`)}
	round = []Event{
		thinkDelta("par"),
		{Type: EventThinkingDone, Thinking: open},
		{Type: EventThinkingDone, Thinking: bare},
		{Type: EventToolCallEnd, ToolCall: call},
		msgDone(),
	}
	want = []Block{
		{Kind: BlockThinking, Thinking: &Thinking{Text: "partial done", Signature: "sig-a"}},
		{Kind: BlockThinking, Thinking: &Thinking{Redacted: true, Data: "opaque", Signature: "sig-b"}},
		{Kind: BlockToolCall, ToolCall: &ToolCall{ID: "c1", Name: "read_file", Input: json.RawMessage(`{"path":"a.txt"}`)}},
	}
	return round, open, bare, call, want
}

// TestModelEmitsLeftUnmodified checks the structs the model emitted are exactly as
// they were after the Turn: the agent reads them and never writes to them. It
// compares each emitted struct with a copy taken before the Turn.
func TestModelEmitsLeftUnmodified(t *testing.T) {
	round, open, bare, call, _ := emitFixtures()
	before := struct {
		open, bare Thinking
		call       ToolCall
	}{*open, *bare, deepClone(*call)}

	h := newHarness(round, []Event{textEv("done"), msgDone()})
	h.mustTurn(t, "go")

	expectEqual(t, "emitted Thinking on an open block", *open, before.open)
	expectEqual(t, "emitted Thinking on no open block", *bare, before.bare)
	expectEqual(t, "emitted ToolCall", *call, before.call)
}

// TestStoredBlocksDoNotAliasModelEmits modifies the structs the model emitted
// after the Turn, and checks the blocks stored in the conversation do not change.
func TestStoredBlocksDoNotAliasModelEmits(t *testing.T) {
	round, open, bare, call, want := emitFixtures()
	input := call.Input // the slice the model emitted, kept apart from call
	h := newHarness(round, []Event{textEv("done"), msgDone()})
	h.mustTurn(t, "go")

	open.Text, open.Signature = "MUTATED", "MUTATED"
	bare.Data, bare.Redacted = "MUTATED", false
	call.Name = "MUTATED"
	input[0] = 'X'

	expectEqual(t, "stored assistant blocks", h.message(t, 1).Content, want)
}

// ---------------------------------------------------------------------------
// Failed turns, merging, and the recorded sequence.
// ---------------------------------------------------------------------------

// TestMergeAfterErrorWithoutAssistantReply checks that a Turn after an error with
// no assistant reply adds its text to the existing user message instead of making
// a second user message, in the conversation, in the next request and in the
// record.
func TestMergeAfterErrorWithoutAssistantReply(t *testing.T) {
	h := newHarness([]Event{errEv(errors.New("boom"))}, []Event{textEv("answer"), msgDone()})
	if err := h.turn("first"); err == nil {
		t.Fatal("first Turn returned nil")
	}
	h.mustTurn(t, "second")

	want := []Message{
		{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "first"}, {Kind: BlockText, Text: "second"}}},
		{Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "answer"}}},
	}
	expectEqual(t, "conversation", h.conv.Messages, want)
	expectEqual(t, "second request", h.request(t, 1).Messages, want[:1])
	expectEqual(t, "recorded entries", h.rec.entries, []recEntry{
		{RecordAppend, Message{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "first"}}}},
		{RecordMerge, Message{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "second"}}}},
		{RecordAppend, want[1]},
	})
	expectRebuilds(t, h)
}

// TestMergeAfterFailedRoundKeepsToolResults checks that when round 2 fails after
// round 1's tool results were appended, the conversation keeps the user message,
// the tool call and its results, and the next Turn adds its text to the results
// message.
func TestMergeAfterFailedRoundKeepsToolResults(t *testing.T) {
	h := newHarness(fullRound1(), []Event{errEv(errors.New("boom"))}, []Event{textEv("recovered"), msgDone()})
	if err := h.turn("Read a.txt"); err == nil {
		t.Fatal("first Turn returned nil")
	}
	kept := wantConv()[:3]
	expectEqual(t, "conversation after the failed turn", h.conv.Messages, kept)

	h.mustTurn(t, "try again")
	resultsAndText := Message{Role: RoleUser, Content: []Block{
		kept[2].Content[0],
		{Kind: BlockText, Text: "try again"},
	}}
	want := []Message{kept[0], kept[1], resultsAndText, {Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "recovered"}}}}
	expectEqual(t, "conversation", h.conv.Messages, want)
	expectEqual(t, "third request", h.request(t, 2).Messages, want[:3])

	kinds := []RecordKind{}
	for _, e := range h.rec.entries {
		kinds = append(kinds, e.Kind)
	}
	expectEqual(t, "record kinds", kinds, []RecordKind{RecordAppend, RecordAppend, RecordAppend, RecordMerge, RecordAppend})
	expectRebuilds(t, h)
}

// TestNewUserMessageAfterAssistantReply is the opposite edge of the merge rule: a
// Turn on a conversation whose last message is an assistant message appends a
// new user message and records it as RecordAppend.
func TestNewUserMessageAfterAssistantReply(t *testing.T) {
	h := newHarness(
		[]Event{textEv("one"), msgDone()},
		[]Event{textEv("two"), msgDone()},
	)
	h.mustTurn(t, "first")
	h.mustTurn(t, "second")
	want := []Message{
		{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "first"}}},
		{Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "one"}}},
		{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "second"}}},
		{Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "two"}}},
	}
	expectEqual(t, "conversation", h.conv.Messages, want)
	expectEqual(t, "recorded entries", h.rec.entries, appendEntries(want))
}

// TestRecordedSequenceRebuildsConversation runs four Turns, two of them failing,
// and checks that replaying the record rebuilds the conversation exactly, and
// that the record's kinds are the ones the merge rule predicts.
func TestRecordedSequenceRebuildsConversation(t *testing.T) {
	h := newHarness(
		fullRound1(), fullRound2(), // turn 1: succeeds
		[]Event{errEv(errors.New("boom"))},                                     // turn 2: fails before any reply
		[]Event{callEnd("call-2", "read_file", `{"path":"b.txt"}`), msgDone()}, // turn 3, round 1: a tool call
		[]Event{errEv(errors.New("boom"))},                                     // turn 3, round 2: fails
		[]Event{textEv("last"), msgDone()},                                     // turn 4: succeeds
	)
	h.mustTurn(t, "Read a.txt")
	if err := h.turn("q2"); err == nil {
		t.Fatal("turn 2 returned nil")
	}
	if err := h.turn("q3"); err == nil {
		t.Fatal("turn 3 returned nil")
	}
	h.mustTurn(t, "q4")

	kinds := []RecordKind{}
	for _, e := range h.rec.entries {
		kinds = append(kinds, e.Kind)
	}
	expectEqual(t, "record kinds", kinds, []RecordKind{
		RecordAppend, RecordAppend, RecordAppend, RecordAppend, // turn 1: user, assistant, results, assistant
		RecordAppend,                            // turn 2: user q2
		RecordMerge, RecordAppend, RecordAppend, // turn 3: q3 merges into q2; assistant call; results
		RecordMerge, RecordAppend, // turn 4: q4 merges into the results; assistant
	})
	if len(h.conv.Messages) != 8 {
		t.Fatalf("want 8 messages, got %d: %s", len(h.conv.Messages), dump(h.conv.Messages))
	}
	expectRebuilds(t, h)
}

// TestRebuildRejectsBadRecords is the control for rebuild: a merge into a message
// of another role, a merge with nothing to merge into, and an unknown kind must
// each be rejected, so a passing rebuild in the tests above means the records
// followed the contract.
func TestRebuildRejectsBadRecords(t *testing.T) {
	user := Message{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "u"}}}
	asst := Message{Role: RoleAssistant, Content: []Block{{Kind: BlockText, Text: "a"}}}
	cases := map[string][]recEntry{
		"merge_into_other_role": {{RecordAppend, asst}, {RecordMerge, user}},
		"merge_into_nothing":    {{RecordMerge, user}},
		"unknown_kind":          {{RecordKind(99), user}},
	}
	for name, entries := range cases {
		if _, err := rebuild(entries); err == nil {
			t.Fatalf("%s: rebuild accepted the records", name)
		}
	}
	if got, err := rebuild([]recEntry{{RecordAppend, asst}, {RecordMerge, asst}}); err != nil || len(got) != 1 || len(got[0].Content) != 2 {
		t.Fatalf("a valid merge was not applied: %v %s", err, dump(got))
	}
}
