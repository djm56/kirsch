package session

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/agent"
)

// stoppedMessage returns an assistant text message carrying the stop-reason
// metadata the agent sets from a stream.
func stoppedMessage(text, stopReason string, maxTokens int) agent.Message {
	m := assistantText(text)
	m.StopReason = stopReason
	m.MaxTokens = maxTokens
	return m
}

// assistantLine returns the parsed assistant.message line of the session at
// path, failing unless there is exactly one. It reads the file Store.Record
// wrote, so it exercises the real writer path (Record -> encodeAssistantAppend).
func assistantLine(t *testing.T, path string) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, l := range readLines(t, path) {
		if l["type"] == "assistant.message" {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		t.Fatalf("assistant.message lines = %d, want 1", len(found))
	}
	return found[0]
}

// TestRecordWritesStopReasonAndMaxTokensOnAssistantMessage records an assistant
// message that carries both fields and checks both reach the assistant.message
// line, with the event version still 1.
func TestRecordWritesStopReasonAndMaxTokensOnAssistantMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path, record{Kind: agent.RecordAppend, Message: stoppedMessage("cut", "max_tokens", 16384)})

	line := assistantLine(t, path)
	if line["stop_reason"] != "max_tokens" {
		t.Errorf("stop_reason = %v, want max_tokens", line["stop_reason"])
	}
	if line["max_tokens"] != float64(16384) {
		t.Errorf("max_tokens = %v, want 16384", line["max_tokens"])
	}
	if line["v"] != float64(1) {
		t.Errorf("v = %v, want 1", line["v"])
	}
}

// TestRecordOmitsStopReasonFieldsWhenEmpty records an assistant message with
// neither field set and checks neither key is written. The message in the
// previous test is the positive control: the same path writes both keys.
func TestRecordOmitsStopReasonFieldsWhenEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeSession(t, path, record{Kind: agent.RecordAppend, Message: assistantText("plain")})

	line := assistantLine(t, path)
	for _, k := range []string{"stop_reason", "max_tokens"} {
		if _, ok := line[k]; ok {
			t.Errorf("key %q written for a message with no stop reason or limit", k)
		}
	}
	if line["content"] != "plain" {
		t.Errorf("control: content = %v, want plain", line["content"])
	}
}

// TestResumeRoundTripsStopReasonAndMaxTokens writes a max_tokens message through
// Store.Record and reads it back through Resume; the conversation message and the
// transcript entry both carry the two fields.
func TestResumeRoundTripsStopReasonAndMaxTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	msg := stoppedMessage("cut", "max_tokens", 16384)
	writeSession(t, path, record{Kind: agent.RecordAppend, Message: msg})

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !reflect.DeepEqual(conv.Messages, []agent.Message{msg}) {
		t.Errorf("conversation mismatch:\n got %+v\nwant %+v", conv.Messages, []agent.Message{msg})
	}
	if len(tr.Messages) != 1 || tr.Messages[0].StopReason != "max_tokens" || tr.Messages[0].MaxTokens != 16384 {
		t.Errorf("transcript = %+v, want one entry with stop reason max_tokens and limit 16384", tr.Messages)
	}
}

// TestResumeOldSessionLineWithoutStopFields resumes a hand-written
// assistant.message line of the shape written before v0.1.2 (no stop_reason, no
// max_tokens) and checks it loads with both fields zero and no error.
func TestResumeOldSessionLineWithoutStopFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.jsonl")
	old := `{"v":1,"type":"user.message","content":"hi"}` + "\n" +
		`{"v":1,"type":"assistant.message","content":"hello"}` + "\n"
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	conv, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume of an old session: %v", err)
	}
	if len(conv.Messages) != 2 || len(tr.Messages) != 2 {
		t.Fatalf("messages: conv %d, transcript %d, want 2 and 2", len(conv.Messages), len(tr.Messages))
	}
	if conv.Messages[1].StopReason != "" || conv.Messages[1].MaxTokens != 0 {
		t.Errorf("old line: conv stop fields = %q/%d, want empty/0", conv.Messages[1].StopReason, conv.Messages[1].MaxTokens)
	}
	if tr.Messages[1].StopReason != "" || tr.Messages[1].MaxTokens != 0 {
		t.Errorf("old line: transcript stop fields = %q/%d, want empty/0", tr.Messages[1].StopReason, tr.Messages[1].MaxTokens)
	}
}

// TestResumeStopReasonWithoutMaxTokens resumes a hand-written line that has
// stop_reason but no max_tokens, which the writer never produces but spec §3.7
// names; the stop reason survives and the limit stays zero.
func TestResumeStopReasonWithoutMaxTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.jsonl")
	line := `{"v":1,"type":"assistant.message","content":"cut","stop_reason":"max_tokens"}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if len(tr.Messages) != 1 || tr.Messages[0].StopReason != "max_tokens" || tr.Messages[0].MaxTokens != 0 {
		t.Errorf("transcript = %+v, want one entry, stop reason max_tokens, limit 0", tr.Messages)
	}
}

// TestRecordedStopReasonIsNotInUserMessages checks a user message never gets the
// fields: encodeUser* does not read them, so a stray value on a user message
// must not be written.
func TestRecordedStopReasonIsNotInUserMessages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	user := agent.Message{
		Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "q"}},
		StopReason: "max_tokens", MaxTokens: 5,
	}
	writeSession(t, path, record{Kind: agent.RecordAppend, Message: user})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"type":"user.message"`) {
		t.Fatalf("control: no user.message line in %q", raw)
	}
	if strings.Contains(string(raw), "stop_reason") || strings.Contains(string(raw), "max_tokens") {
		t.Errorf("user.message line carries stop fields: %q", raw)
	}
}

// TestCompactKeepsStopReasonAndMaxTokensOnRetainedAssistantMessages compacts a
// seven-turn conversation of parallel tool calls whose last assistant message (retained verbatim, after
// the compacted range) carries a stop reason and a limit. Both must survive in
// the result, so the live conversation matches what Resume rebuilds from the log.
// The comparison against the other retained assistant messages, which carry
// neither, is the control that only the marked one has them. roundTrip then
// writes the original messages and the compaction to a log and requires Resume
// to DeepEqual comp.Messages, which fails when the two disagree about the fields.
func TestCompactKeepsStopReasonAndMaxTokensOnRetainedAssistantMessages(t *testing.T) {
	msgs := buildParallelToolMessages(7) // a shape Resume reproduces exactly (see TestCompactionRoundTripWithToolCalls)
	last := len(msgs) - 1                // the final assistant text, retained after the compacted range
	if msgs[last].Role != agent.RoleAssistant {
		t.Fatalf("setup: message %d is %v, want an assistant message", last, msgs[last].Role)
	}
	msgs[last].StopReason, msgs[last].MaxTokens = "max_tokens", 16384

	comp := roundTrip(t, msgs)

	var withStop []int
	for i, m := range comp.Messages {
		if m.StopReason != "" || m.MaxTokens != 0 {
			withStop = append(withStop, i)
		}
	}
	wantIdx := len(comp.Messages) - 1
	if len(withStop) != 1 || withStop[0] != wantIdx {
		t.Fatalf("messages carrying stop fields = %v, want only [%d]", withStop, wantIdx)
	}
	if got := comp.Messages[wantIdx]; got.StopReason != "max_tokens" || got.MaxTokens != 16384 {
		t.Errorf("retained assistant message stop fields = %q/%d, want max_tokens/16384", got.StopReason, got.MaxTokens)
	}
}

// TestResumeMaxTokensWithoutStopReason resumes a hand-written assistant.message
// line that has max_tokens but no stop_reason (what the writer produces when the
// provider gave no stop reason). Both transcript fields are read as written: the
// limit is kept and the stop reason stays empty, which is what the app turns
// into "no notice".
func TestResumeMaxTokensWithoutStopReason(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nostop.jsonl")
	line := `{"v":1,"type":"assistant.message","content":"cut","max_tokens":16384}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, tr, err := Resume(path)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if len(tr.Messages) != 1 || tr.Messages[0].StopReason != "" || tr.Messages[0].MaxTokens != 16384 {
		t.Errorf("transcript = %+v, want one entry, empty stop reason, limit 16384", tr.Messages)
	}
}
