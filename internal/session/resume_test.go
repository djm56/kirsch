package session

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// record is a single Store.Record call captured for test setup.
type record struct {
	Kind    agent.RecordKind
	Message agent.Message
}

// agentThinking returns an assistant thinking block with DropOnSummary set,
// matching the value the agent produces and the loader restores.
func agentThinking(text, signature string) agent.Message {
	return agent.Message{
		Role: agent.RoleAssistant,
		Content: []agent.Block{{
			Kind: agent.BlockThinking,
			Thinking: &agent.Thinking{
				Text:          text,
				Signature:     signature,
				DropOnSummary: true,
			},
		}},
	}
}

// writeSession creates a session file by recording the supplied messages and
// closing the store.
func writeSession(t *testing.T, path string, records ...record) {
	t.Helper()
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, r := range records {
		s.Record(r.Kind, r.Message)
	}
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// mustMarshalLine returns a JSONL line for v, failing the test on error.
func mustMarshalLine(t *testing.T, v any) []byte {
	t.Helper()
	b, err := marshalLine(v)
	if err != nil {
		t.Fatalf("marshalLine: %v", err)
	}
	return b
}

// mustMarshalCompactionApplied returns a JSONL line for a compaction.applied
// event, failing the test on error.
func mustMarshalCompactionApplied(t *testing.T, from, to int, summary string) []byte {
	t.Helper()
	b, err := marshalCompactionApplied(from, to, summary)
	if err != nil {
		t.Fatalf("marshalCompactionApplied: %v", err)
	}
	return b
}

// appendLines appends raw bytes to path, creating it if necessary.
func appendLines(t *testing.T, path string, lines ...[]byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	defer func() { _ = f.Close() }()
	for _, line := range lines {
		if _, err := f.Write(line); err != nil {
			t.Fatalf("append line: %v", err)
		}
	}
}

// transcriptFromMessages builds a transcript slice from conversation messages
// using the same deep-copy semantics Resume applies.
func transcriptFromMessages(msgs []agent.Message) []TranscriptMessage {
	out := make([]TranscriptMessage, len(msgs))
	for i, m := range msgs {
		out[i] = TranscriptMessage{Role: m.Role, Content: cloneBlocks(m.Content)}
	}
	return out
}

// TestResumeEmptyFile verifies that resuming an empty session yields an empty
// conversation and transcript.
func TestResumeEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if len(conv.Messages) != 0 {
		t.Errorf("conversation messages = %d, want 0", len(conv.Messages))
	}
	if len(tr.Messages) != 0 {
		t.Errorf("transcript messages = %d, want 0", len(tr.Messages))
	}
}

// TestResumeMergesConsecutiveUserMessages verifies that consecutive
// user.message events become a single user message with multiple text blocks.
func TestResumeMergesConsecutiveUserMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: userText("hello")},
		record{Kind: agent.RecordMerge, Message: userTextMerge(" world")},
	)

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	want := &agent.Conversation{Messages: []agent.Message{{
		Role: agent.RoleUser,
		Content: []agent.Block{
			{Kind: agent.BlockText, Text: "hello"},
			{Kind: agent.BlockText, Text: " world"},
		},
	}}}
	if !reflect.DeepEqual(conv, want) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv, want)
	}
	if !reflect.DeepEqual(tr.Messages, transcriptFromMessages(want.Messages)) {
		t.Errorf("transcript mismatch:\n got %+v\nwant %+v", tr.Messages, want.Messages)
	}
}

// TestResumeReconstructsAssistantText verifies that an assistant.message event
// becomes an assistant message with a text block.
func TestResumeReconstructsAssistantText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: assistantText("hi there")},
	)

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	want := &agent.Conversation{Messages: []agent.Message{{
		Role:    agent.RoleAssistant,
		Content: []agent.Block{{Kind: agent.BlockText, Text: "hi there"}},
	}}}
	if !reflect.DeepEqual(conv, want) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv, want)
	}
	if !reflect.DeepEqual(tr.Messages, transcriptFromMessages(want.Messages)) {
		t.Errorf("transcript mismatch:\n got %+v\nwant %+v", tr.Messages, want.Messages)
	}
}

// TestResumeReconstructsAssistantThinking verifies that an assistant.thinking
// event becomes an assistant message containing a signed thinking block.
func TestResumeReconstructsAssistantThinking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: agentThinking("<thinking>", "sig-1")},
	)

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	want := &agent.Conversation{Messages: []agent.Message{agentThinking("<thinking>", "sig-1")}}
	if !reflect.DeepEqual(conv, want) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv, want)
	}
	if !reflect.DeepEqual(tr.Messages, transcriptFromMessages(want.Messages)) {
		t.Errorf("transcript mismatch:\n got %+v\nwant %+v", tr.Messages, want.Messages)
	}
}

// TestResumeReconstructsToolCallAndResult verifies that an assistant.message
// with tool calls and a matching tool.completed are reconstructed as the
// corresponding assistant and user messages.
func TestResumeReconstructsToolCallAndResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: assistantToolCall("call-1", "read_file", `{"path":"/x"}`)},
		record{Kind: agent.RecordAppend, Message: userToolResult("call-1", "read_file", true, "content")},
	)

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	want := &agent.Conversation{Messages: []agent.Message{
		{
			Role: agent.RoleAssistant,
			Content: []agent.Block{{
				Kind: agent.BlockToolCall,
				ToolCall: &agent.ToolCall{
					ID:    "call-1",
					Name:  "read_file",
					Input: []byte(`{"path":"/x"}`),
				},
			}},
		},
		{
			Role: agent.RoleUser,
			Content: []agent.Block{{
				Kind: agent.BlockToolResult,
				Result: &agent.ResultBlock{
					CallID:  "call-1",
					Content: "content",
					IsError: false,
				},
			}},
		},
	}}
	if !reflect.DeepEqual(conv, want) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv, want)
	}
	if !reflect.DeepEqual(tr.Messages, transcriptFromMessages(want.Messages)) {
		t.Errorf("transcript mismatch:\n got %+v\nwant %+v", tr.Messages, want.Messages)
	}
}

// TestResumeMergesConsecutiveToolResults verifies that multiple tool.completed
// events answering the same assistant message merge into a single user message.
func TestResumeMergesConsecutiveToolResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: agent.Message{
			Role: agent.RoleAssistant,
			Content: []agent.Block{
				{Kind: agent.BlockToolCall, ToolCall: &agent.ToolCall{ID: "call-1", Name: "read_file", Input: []byte(`{}`)}},
				{Kind: agent.BlockToolCall, ToolCall: &agent.ToolCall{ID: "call-2", Name: "read_file", Input: []byte(`{}`)}},
			},
		}},
		record{Kind: agent.RecordAppend, Message: userToolResult("call-1", "read_file", true, "a")},
		record{Kind: agent.RecordAppend, Message: userToolResult("call-2", "read_file", true, "b")},
	)

	conv, _, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if len(conv.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(conv.Messages))
	}
	last := conv.Messages[1]
	if last.Role != agent.RoleUser {
		t.Errorf("last role = %q, want user", last.Role)
	}
	if len(last.Content) != 2 {
		t.Errorf("last content blocks = %d, want 2", len(last.Content))
	}
}

// TestResumeIgnoresAssistantDelta verifies that assistant.delta events do not
// affect the reconstructed conversation or transcript.
func TestResumeIgnoresAssistantDelta(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: userText("prompt")},
	)
	appendLines(t, path, []byte(`{"v":1,"type":"assistant.delta","text":"delta fragment"}`+"\n"))

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if len(conv.Messages) != 1 {
		t.Errorf("conversation messages = %d, want 1", len(conv.Messages))
	}
	if len(tr.Messages) != 1 {
		t.Errorf("transcript messages = %d, want 1", len(tr.Messages))
	}
}

// TestResumeAppliesCompactionToConversationNotTranscript verifies that a
// compaction.applied event replaces the indicated conversation range and
// leaves the transcript untouched.
func TestResumeAppliesCompactionToConversationNotTranscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: userText("one")},
		record{Kind: agent.RecordAppend, Message: assistantText("two")},
		record{Kind: agent.RecordAppend, Message: userText("three")},
	)
	appendLines(t, path, mustMarshalCompactionApplied(t, 0, 1, "summary"))

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	if len(conv.Messages) != 2 {
		t.Fatalf("conversation messages = %d, want 2", len(conv.Messages))
	}
	if conv.Messages[0].Role != agent.RoleAssistant || conv.Messages[0].Content[0].Text != "summary" {
		t.Errorf("compacted message = %+v, want assistant summary", conv.Messages[0])
	}
	if conv.Messages[1].Role != agent.RoleUser {
		t.Errorf("post-compaction message role = %q, want user", conv.Messages[1].Role)
	}

	if len(tr.Messages) != 3 {
		t.Fatalf("transcript messages = %d, want 3", len(tr.Messages))
	}
}

// TestResumeReturnsMalformedSessionOnToolMismatch verifies that a
// tool.completed event whose id does not match the most recent assistant
// message's tool calls returns ErrMalformedSession.
func TestResumeReturnsMalformedSessionOnToolMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: assistantToolCall("call-1", "read_file", `{}`)},
	)
	appendLines(t, path, mustMarshalLine(t, toolCompletedEvent{
		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
		ID:        "call-2",
		OK:        true,
	}))

	_, _, err := Resume(path)
	if !errors.Is(err, ErrMalformedSession) {
		t.Fatalf("Resume error = %v, want ErrMalformedSession", err)
	}
}

// TestResumeReturnsMalformedSessionOnDuplicateToolResult verifies that two
// tool.completed events for the same id return ErrMalformedSession.
func TestResumeReturnsMalformedSessionOnDuplicateToolResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: assistantToolCall("call-1", "read_file", `{}`)},
	)
	appendLines(t, path,
		mustMarshalLine(t, toolCompletedEvent{
			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
			ID:        "call-1",
			OK:        true,
		}),
		mustMarshalLine(t, toolCompletedEvent{
			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
			ID:        "call-1",
			OK:        true,
		}),
	)

	_, _, err := Resume(path)
	if !errors.Is(err, ErrMalformedSession) {
		t.Fatalf("Resume error = %v, want ErrMalformedSession", err)
	}
}

// TestResumeReturnsMalformedSessionOnOrphanToolResult verifies that a
// tool.completed event with no preceding assistant.message tool call returns
// ErrMalformedSession.
func TestResumeReturnsMalformedSessionOnOrphanToolResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	appendLines(t, path, mustMarshalLine(t, toolCompletedEvent{
		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
		ID:        "call-1",
		OK:        true,
	}))

	_, _, err := Resume(path)
	if !errors.Is(err, ErrMalformedSession) {
		t.Fatalf("Resume error = %v, want ErrMalformedSession", err)
	}
}

// TestResumeReturnsMalformedSessionOnEmptyAssistantMessage verifies that an
// assistant.message event with neither content nor tool calls is rejected.
func TestResumeReturnsMalformedSessionOnEmptyAssistantMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	appendLines(t, path, mustMarshalLine(t, assistantMessageEvent{
		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantMessage},
	}))

	_, _, err := Resume(path)
	if !errors.Is(err, ErrMalformedSession) {
		t.Fatalf("Resume error = %v, want ErrMalformedSession", err)
	}
}

// TestResumeReturnsMalformedSessionOnCompactionRangeOutOfBounds verifies that
// a compaction.applied event with an invalid range returns ErrMalformedSession.
func TestResumeReturnsMalformedSessionOnCompactionRangeOutOfBounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: userText("one")},
	)
	appendLines(t, path, mustMarshalCompactionApplied(t, 0, 5, "summary"))

	_, _, err := Resume(path)
	if !errors.Is(err, ErrMalformedSession) {
		t.Fatalf("Resume error = %v, want ErrMalformedSession", err)
	}
}

// TestResumeTornLineTruncates verifies that an invalid final JSON line is
// treated as a repair boundary and ignored, returning the valid prefix.
func TestResumeTornLineTruncates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: userText("good")},
	)
	appendLines(t, path, []byte(`{not valid json`))

	conv, _, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if len(conv.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(conv.Messages))
	}
	if conv.Messages[0].Content[0].Text != "good" {
		t.Errorf("text = %q, want %q", conv.Messages[0].Content[0].Text, "good")
	}
}

// TestResumeRoundTripAllEventTypes records a representative conversation
// through the Store and verifies that Resume reconstructs it exactly.
func TestResumeRoundTripAllEventTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	original := &agent.Conversation{Messages: []agent.Message{
		userText("hello"),
		agentThinking("think", "sig"),
		{
			Role: agent.RoleAssistant,
			Content: []agent.Block{
				{Kind: agent.BlockText, Text: "answer"},
				{
					Kind: agent.BlockToolCall,
					ToolCall: &agent.ToolCall{
						ID:    "call-1",
						Name:  "read_file",
						Input: []byte(`{"path":"/x"}`),
					},
				},
			},
		},
		{
			Role: agent.RoleUser,
			Content: []agent.Block{{
				Kind: agent.BlockToolResult,
				Result: &agent.ResultBlock{
					CallID:    "call-1",
					Content:   "result",
					IsError:   false,
					Truncated: false,
				},
			}},
		},
	}}

	records := []record{
		{Kind: agent.RecordAppend, Message: original.Messages[0]},
		{Kind: agent.RecordAppend, Message: original.Messages[1]},
		{Kind: agent.RecordAppend, Message: original.Messages[2]},
		{Kind: agent.RecordAppend, Message: original.Messages[3]},
	}
	writeSession(t, path, records...)

	conv, _, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !reflect.DeepEqual(conv, original) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv, original)
	}
}

// TestResumeRoundTripMerge verifies that RecordMerge semantics survive a full
// store-write and resume cycle.
func TestResumeRoundTripMerge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	original := &agent.Conversation{Messages: []agent.Message{{
		Role: agent.RoleUser,
		Content: []agent.Block{
			{Kind: agent.BlockText, Text: "hello"},
			{Kind: agent.BlockText, Text: " world"},
		},
	}}}
	writeSession(t, path,
		record{Kind: agent.RecordAppend, Message: userText("hello")},
		record{Kind: agent.RecordMerge, Message: userTextMerge(" world")},
	)

	conv, _, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !reflect.DeepEqual(conv, original) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv, original)
	}
}

// FuzzResume verifies that Resume never panics on arbitrary file content.
func FuzzResume(f *testing.F) {
	f.Add([]byte(`{"v":1,"type":"user.message","content":"x"}` + "\n"))
	f.Add([]byte("not json"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "session.jsonl")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		_, _, _ = Resume(path)
	})
}
