package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// userText returns a user message containing a single text block.
func userText(text string) agent.Message {
	return agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	}
}

// userTextMerge returns the delta used by RecordMerge.
func userTextMerge(text string) agent.Message {
	return agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	}
}

// assistantText returns an assistant message containing text blocks.
func assistantText(text string) agent.Message {
	return agent.Message{
		Role:    agent.RoleAssistant,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	}
}

// assistantThinking returns an assistant thinking block.
func assistantThinking(content string) agent.Message {
	return agent.Message{
		Role:    agent.RoleAssistant,
		Content: []agent.Block{{Kind: agent.BlockThinking, Thinking: &agent.Thinking{Text: content}}},
	}
}

// assistantToolCall returns an assistant message with a tool call.
func assistantToolCall(id, tool, input string) agent.Message {
	return agent.Message{
		Role: agent.RoleAssistant,
		Content: []agent.Block{{
			Kind: agent.BlockToolCall,
			ToolCall: &agent.ToolCall{
				ID:    id,
				Name:  tool,
				Input: []byte(input),
			},
		}},
	}
}

// userToolResult returns a user message with a tool result.
func userToolResult(id, tool string, ok bool, content string) agent.Message {
	return agent.Message{
		Role: agent.RoleUser,
		Content: []agent.Block{{
			Kind: agent.BlockToolResult,
			Result: &agent.ResultBlock{
				CallID:  id,
				IsError: !ok,
				Content: content,
			},
		}},
	}
}

// readLines reads and parses every JSONL line in path.
func readLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open session file: %v", err)
	}
	defer func() { _ = f.Close() }()

	var lines []map[string]any
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		var line map[string]any
		if err := json.Unmarshal(scan.Bytes(), &line); err != nil {
			t.Fatalf("unmarshal line %q: %v", scan.Text(), err)
		}
		lines = append(lines, line)
	}
	if err := scan.Err(); err != nil {
		t.Fatalf("scan session file: %v", err)
	}
	return lines
}

// TestAppendMapping verifies that every RecordKind value maps to the expected
// plan §7 event type and that every line carries "v":1.
func TestAppendMapping(t *testing.T) {
	tests := []struct {
		name     string
		kind     agent.RecordKind
		msg      agent.Message
		wantType string
		wantRole string
		wantText string
	}{
		{
			name:     "user append text",
			kind:     agent.RecordAppend,
			msg:      userText("hello"),
			wantType: eventTypeUserMessage,
			wantRole: "user",
			wantText: "hello",
		},
		{
			name:     "user merge text",
			kind:     agent.RecordMerge,
			msg:      userTextMerge(" world"),
			wantType: eventTypeUserMessage,
			wantRole: "user",
			wantText: " world",
		},
		{
			name:     "assistant text",
			kind:     agent.RecordAppend,
			msg:      assistantText("hi there"),
			wantType: eventTypeAssistantMessage,
			wantText: "hi there",
		},
		{
			name:     "assistant thinking",
			kind:     agent.RecordAppend,
			msg:      assistantThinking("<thinking>"),
			wantType: eventTypeAssistantThinking,
			wantText: "<thinking>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.jsonl")
			s, err := New(path, WithCadence(time.Hour))
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			s.Record(tt.kind, tt.msg)
			if err := s.Flush(); err != nil {
				t.Fatalf("Flush: %v", err)
			}
			if err := s.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			lines := readLines(t, path)
			if len(lines) != 1 {
				t.Fatalf("expected 1 line, got %d", len(lines))
			}
			line := lines[0]
			if got := int(line["v"].(float64)); got != eventVersion {
				t.Errorf("v = %d, want %d", got, eventVersion)
			}
			if got := line["type"].(string); got != tt.wantType {
				t.Errorf("type = %q, want %q", got, tt.wantType)
			}
			if tt.wantRole != "" {
				if got := line["role"].(string); got != tt.wantRole {
					t.Errorf("role = %q, want %q", got, tt.wantRole)
				}
			}
			if tt.wantText != "" {
				var got string
				switch tt.wantType {
				case eventTypeAssistantThinking:
					got = line["content"].(string)
				case eventTypeAssistantMessage:
					got = line["content"].(string)
				case eventTypeUserMessage:
					got = line["content"].(string)
				}
				if got != tt.wantText {
					t.Errorf("text = %q, want %q", got, tt.wantText)
				}
			}
		})
	}
}

// TestAppendToolEvents verifies that user tool results and assistant tool calls
// serialise to tool.completed and assistant.message respectively.
func TestAppendToolEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	s.Record(agent.RecordAppend, assistantToolCall("call-1", "read_file", `{"path":"/x"}`))
	s.Record(agent.RecordAppend, userToolResult("call-1", "read_file", true, "content"))
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, path)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if got := lines[0]["type"].(string); got != eventTypeAssistantMessage {
		t.Errorf("first line type = %q, want %q", got, eventTypeAssistantMessage)
	}
	calls, ok := lines[0]["tool_calls"].([]any)
	if !ok || len(calls) != 1 {
		t.Fatalf("expected one tool_call, got %v", lines[0]["tool_calls"])
	}
	if got := lines[1]["type"].(string); got != eventTypeToolCompleted {
		t.Errorf("second line type = %q, want %q", got, eventTypeToolCompleted)
	}
	if got := lines[1]["id"].(string); got != "call-1" {
		t.Errorf("tool id = %q, want %q", got, "call-1")
	}
	if got := lines[1]["ok"].(bool); got != true {
		t.Errorf("tool ok = %v, want true", got)
	}
}

// TestFlushDurability verifies that buffered lines are durable after Flush and
// that a Flush error surfaces when the underlying file is closed.
func TestFlushDurability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	s.Record(agent.RecordAppend, userText("before flush"))
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 durable line before close, got %d", len(lines))
	}

	// Close the underlying file descriptor to force subsequent writes to fail.
	// The cadence is set to an hour so the ticker cannot race with this close.
	if err := s.file.Close(); err != nil {
		t.Fatalf("close underlying file: %v", err)
	}

	s.Record(agent.RecordAppend, userText("after close"))
	if err := s.Flush(); err == nil {
		t.Fatal("expected Flush error after underlying file closed")
	}

	// Close must still complete without panicking.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestConcurrentAppend verifies that many goroutines can record simultaneously
// without interleaving bytes or losing lines.
func TestConcurrentAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const n = 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Record(agent.RecordAppend, userText("hello from goroutine"))
		}(i)
	}
	wg.Wait()

	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Ensure every line is exactly one JSON object and nothing is interleaved.
	scan := bufio.NewScanner(f)
	var count int
	for scan.Scan() {
		line := scan.Text()
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
			t.Fatalf("line %d is not a single JSON object: %q", count, line)
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("line %d invalid JSON: %v", count, err)
		}
		if obj["type"] != eventTypeUserMessage {
			t.Fatalf("line %d type = %v, want %v", count, obj["type"], eventTypeUserMessage)
		}
		count++
	}
	if err := scan.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != n {
		t.Fatalf("expected %d lines, got %d", n, count)
	}
}

// TestCadenceFlush verifies that the background ticker flushes buffered lines
// without an explicit Flush call.
func TestCadenceFlush(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	cadence := 25 * time.Millisecond
	s, err := New(path, WithCadence(cadence))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	s.Record(agent.RecordAppend, userText("tick"))
	// Wait long enough for the ticker to fire without an explicit Flush.
	time.Sleep(5 * cadence)

	// The file must contain the line even though Flush was never called.
	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line after ticker flush, got %d", len(lines))
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestCloseSemantics verifies that Close drains pending events and that
// subsequent Record/Flush operations are handled cleanly.
func TestCloseSemantics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	var reported []error
	s, err := New(path, WithCadence(time.Hour), WithOnError(func(err error) {
		reported = append(reported, err)
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	s.Record(agent.RecordAppend, userText("queued"))
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, path)
	if len(lines) != 1 {
		t.Fatalf("expected 1 drained line, got %d", len(lines))
	}

	s.Record(agent.RecordAppend, userText("after close"))
	if len(reported) == 0 || !errors.Is(reported[len(reported)-1], ErrClosed) {
		t.Fatalf("expected last recorded error to be ErrClosed, got %v", reported)
	}

	if err := s.Flush(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Flush after Close: %v, want ErrClosed", err)
	}

	// Close is idempotent.
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestOnErrorHook verifies that mid-turn fsync failures reach the consumer
// through the injected OnError hook.
func TestOnErrorHook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	var (
		mu       sync.Mutex
		reported []error
	)
	cadence := 25 * time.Millisecond
	s, err := New(path, WithCadence(cadence), WithOnError(func(err error) {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, err)
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.Flush(); err != nil {
		t.Fatalf("initial Flush: %v", err)
	}
	if err := s.file.Close(); err != nil {
		t.Fatalf("close underlying file: %v", err)
	}

	s.Record(agent.RecordAppend, userText("doomed"))
	// Wait for the background ticker to attempt a flush/fsync and fail.
	time.Sleep(5 * cadence)

	mu.Lock()
	haveError := len(reported) > 0
	mu.Unlock()
	if !haveError {
		t.Fatal("expected OnError to be called for fsync failure")
	}

	_ = s.Close()
}

// TestStoreImplementsRecorder is a compile-time check that *Store implements
// agent.Recorder.
var _ agent.Recorder = (*Store)(nil)

// BenchmarkRecord measures the cost of queuing an event.
func BenchmarkRecord(b *testing.B) {
	path := filepath.Join(b.TempDir(), "session.jsonl")
	s, err := New(path, WithCadence(time.Hour))
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	defer func() { _ = s.Close() }()

	msg := userText("benchmark payload")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Record(agent.RecordAppend, msg)
	}
}
