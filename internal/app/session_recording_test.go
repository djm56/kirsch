package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// recordingProvider is a fake provider.Provider that emits a fixed stream of
// events for each request. It is the M3-style fake used to drive Submit end to
// end without network access.
type recordingProvider struct {
	stream func(ctx context.Context, onEvent func(provider.StreamEvent))
}

func (r *recordingProvider) Stream(ctx context.Context, _ provider.Request, onEvent func(provider.StreamEvent)) error {
	if r.stream != nil {
		r.stream(ctx, onEvent)
	}
	return nil
}

// sessionRecordingApprovalTool is a tool that requires approval and succeeds
// when the approver allows it. It is registered for the integration test so
// that the approval-resolution path can be exercised.
type sessionRecordingApprovalTool struct {
	approver tool.Approver
}

func (sessionRecordingApprovalTool) Name() string { return "session_recording_approval_tool" }
func (sessionRecordingApprovalTool) Description() string {
	return "approval tool for session recording test"
}

func (sessionRecordingApprovalTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t *sessionRecordingApprovalTool) Invoke(ctx context.Context, _ json.RawMessage) tool.Result {
	dec := t.approver.Request(ctx, tool.ApprovalRequest{
		Operation:   policy.OperationCommand,
		Description: "record session approval",
		Argv:        []string{"go", "test"},
	})
	if dec == policy.DecisionAllow || dec == policy.DecisionSession {
		return tool.OKResult("approved", "approved", false)
	}
	return tool.Fail(tool.KindPolicyDenied, "denied")
}

// TestSubmitRecordsFullEventSequence drives a complete turn through Submit with
// a fake provider and a real Store on a temporary directory, then reads the
// JSONL back and asserts the expected event sequence and schema fields.
func TestSubmitRecordsFullEventSequence(t *testing.T) {
	tmp := t.TempDir()

	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")

	a := New(ws, cfg, telemetry.Disabled())
	defer a.Close()

	a.reg.Register(&sessionRecordingApprovalTool{approver: &approvalAdapter{app: a}})

	var streamMu sync.Mutex
	var streamStartOnce sync.Once
	streamStarted := make(chan struct{})
	var callCount int
	a.providerBuilder = func(_ string, _ config.EndpointConfig, _, _, _ string, _ *telemetry.Logger) (provider.Provider, error) {
		return &recordingProvider{
			stream: func(ctx context.Context, onEvent func(provider.StreamEvent)) {
				streamStartOnce.Do(func() { close(streamStarted) })
				streamMu.Lock()
				defer streamMu.Unlock()
				callCount++

				if callCount == 1 {
					// First model response: assistant text/thinking plus a tool call.
					onEvent(provider.StreamEvent{Type: provider.EventTextDelta, Text: "Hello"})
					onEvent(provider.StreamEvent{Type: provider.EventThinkingDelta, Thinking: &provider.Thinking{Text: "planning"}})
					onEvent(provider.StreamEvent{Type: provider.EventThinkingDone, Thinking: &provider.Thinking{Text: "planning", Signature: "sig"}})
					onEvent(provider.StreamEvent{Type: provider.EventToolCallStart, ToolCall: &provider.ToolCall{ID: "call-1", Name: "session_recording_approval_tool"}})
					onEvent(provider.StreamEvent{Type: provider.EventToolCallEnd, ToolCall: &provider.ToolCall{ID: "call-1", Name: "session_recording_approval_tool"}})
					onEvent(provider.StreamEvent{
						Type:       provider.EventMessageDone,
						StopReason: provider.StopToolUse,
						Usage: &provider.Usage{
							InputTokens:      10,
							OutputTokens:     5,
							CacheReadTokens:  1,
							CacheWriteTokens: 2,
						},
					})
					return
				}

				// Subsequent response: a plain assistant message with no tool calls.
				onEvent(provider.StreamEvent{Type: provider.EventTextDelta, Text: "Done"})
				onEvent(provider.StreamEvent{
					Type:       provider.EventMessageDone,
					StopReason: provider.StopEndTurn,
					Usage: &provider.Usage{
						InputTokens:      1,
						OutputTokens:     1,
						CacheReadTokens:  0,
						CacheWriteTokens: 0,
					},
				})
			},
		}, nil
	}

	var msgsMu sync.Mutex
	var msgs []any
	done := make(chan struct{})
	var doneOnce sync.Once

	a.sendFn = func(msg any) {
		if req, ok := msg.(tui.ApprovalRequestedMsg); ok {
			go a.Resolve(req.ID, ApprovalOutcomeSession)
		}

		msgsMu.Lock()
		msgs = append(msgs, msg)
		switch msg.(type) {
		case tui.TurnCompleteMsg, tui.TurnErrorMsg, tui.TurnCancelledMsg:
			msgsMu.Unlock()
			doneOnce.Do(func() { close(done) })
			return
		}
		msgsMu.Unlock()
	}

	a.Submit("run the approval tool", func(key string) string {
		if key == "OPENCODE_API_KEY" {
			return "fake-key"
		}
		return ""
	})

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for turn to complete")
	}

	// Let the turn goroutine finish its flush before we read the file.
	a.Close()

	// Find the JSONL session file.
	matches, err := filepath.Glob(filepath.Join(tmp, "sessions", "*", "*.jsonl"))
	if err != nil {
		t.Fatalf("glob session file: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("want exactly one session file, got %d: %v", len(matches), matches)
	}

	lines, err := readSessionLines(matches[0])
	if err != nil {
		t.Fatalf("read session file: %v", err)
	}

	var types []string
	for _, line := range lines {
		types = append(types, line["type"].(string))
	}

	// The agent emits EventUsage immediately after the assistant stream finishes,
	// before appending the assistant message, so usage appears before
	// assistant.thinking/assistant.message. The tool round is followed by a plain
	// assistant message and its usage. This matches the actual ordering produced
	// by the real turn flow rather than the stylised example in plan §7.
	wantTypes := []string{
		"turn.started",
		"user.message",
		"usage",
		"assistant.thinking",
		"assistant.message",
		"approval.scope_granted",
		"tool.completed",
		"usage",
		"assistant.message",
		"turn.completed",
	}
	if len(types) != len(wantTypes) {
		t.Fatalf("event type sequence length mismatch:\nwant %v\ngot  %v", wantTypes, types)
	}
	for i, want := range wantTypes {
		if types[i] != want {
			t.Fatalf("event %d: want %q, got %q; full sequence: %v", i, want, types[i], types)
		}
	}

	assertTurnStarted(t, lines[0], 1)
	assertUserMessage(t, lines[1], "run the approval tool")
	assertUsage(t, lines[2], 10, 5, 1, 2)
	assertAssistantThinking(t, lines[3], "planning", "sig")
	assertAssistantMessage(t, lines[4], "Hello", "call-1", "session_recording_approval_tool")
	assertApprovalScopeGranted(t, lines[5], "run_command", "go test")
	assertToolCompleted(t, lines[6], "call-1", true, "approved", false)
	assertUsage(t, lines[7], 1, 1, 0, 0)
	assertAssistantMessage(t, lines[8], "Done", "", "")
	assertTurnCompleted(t, lines[9], "success")

	// The aggregate usage file should have been updated too. The state root is
	// tmp because cfg.Session.StorageDir is tmp/sessions and stateRoot strips
	// the trailing "sessions" directory.
	usage, err := session.ReadUsage(tmp)
	if err != nil {
		t.Fatalf("read usage.json: %v", err)
	}
	if usage.PromptTokens != 11 {
		t.Errorf("usage.PromptTokens = %d, want 11", usage.PromptTokens)
	}
	if usage.CompletionTokens != 6 {
		t.Errorf("usage.CompletionTokens = %d, want 6", usage.CompletionTokens)
	}
	if usage.CacheReadTokens != 1 {
		t.Errorf("usage.CacheReadTokens = %d, want 1", usage.CacheReadTokens)
	}
	if usage.CacheWriteTokens != 2 {
		t.Errorf("usage.CacheWriteTokens = %d, want 2", usage.CacheWriteTokens)
	}
	// TotalTokens = in + out + cacheRead + cacheWrite
	if usage.TotalTokens != 20 {
		t.Errorf("usage.TotalTokens = %d, want 20", usage.TotalTokens)
	}
}

// TestSessionStoreOnErrorLogsToTelemetry verifies that the store's OnError hook
// is wired to the telemetry debug log in ensureStore. An encoding error is
// injected by recording a message with an unsupported role; the resulting debug
// log entry must contain the "session store error" message.
func TestSessionStoreOnErrorLogsToTelemetry(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "debug.log")
	log, err := telemetry.New(telemetry.Options{Path: logPath, Enabled: true})
	if err != nil {
		t.Fatalf("create telemetry logger: %v", err)
	}

	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")

	a := New(ws, cfg, log)
	defer a.Close()

	// Force store creation so the OnError hook is wired.
	if _, _, err := a.ensureStore(); err != nil {
		t.Fatalf("ensureStore: %v", err)
	}

	// Inject an encoding error without touching internal/agent.
	a.store.Record(agent.RecordAppend, agent.Message{Role: "unknown-role"})

	if err := log.Close(); err != nil {
		t.Fatalf("close log: %v", err)
	}

	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read debug log: %v", err)
	}
	if !strings.Contains(string(contents), "session store error") {
		t.Fatalf("debug log does not contain 'session store error'; contents:\n%s", contents)
	}
}

// readSessionLines reads a JSONL session file and returns each line as a
// generic map so tests can inspect the exact fields written.
func readSessionLines(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var line map[string]any
		if err := json.Unmarshal([]byte(raw), &line); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func assertTurnStarted(t *testing.T, line map[string]any, wantTurn int) {
	t.Helper()
	if line["type"] != "turn.started" {
		t.Fatalf("want turn.started, got %q", line["type"])
	}
	if got := int(line["turn"].(float64)); got != wantTurn {
		t.Errorf("turn.started turn = %d, want %d", got, wantTurn)
	}
	if line["v"] == nil {
		t.Error("turn.started missing v")
	}
}

func assertUserMessage(t *testing.T, line map[string]any, wantContent string) {
	t.Helper()
	if line["type"] != "user.message" {
		t.Fatalf("want user.message, got %q", line["type"])
	}
	if line["content"] != wantContent {
		t.Errorf("user.message content = %q, want %q", line["content"], wantContent)
	}
}

func assertUsage(t *testing.T, line map[string]any, input, output, cacheRead, cacheWrite int) {
	t.Helper()
	if line["type"] != "usage" {
		t.Fatalf("want usage, got %q", line["type"])
	}
	if got := int(line["input_tokens"].(float64)); got != input {
		t.Errorf("usage input_tokens = %d, want %d", got, input)
	}
	if got := int(line["output_tokens"].(float64)); got != output {
		t.Errorf("usage output_tokens = %d, want %d", got, output)
	}
	if got := int(line["cache_read"].(float64)); got != cacheRead {
		t.Errorf("usage cache_read = %d, want %d", got, cacheRead)
	}
	if got := int(line["cache_write"].(float64)); got != cacheWrite {
		t.Errorf("usage cache_write = %d, want %d", got, cacheWrite)
	}
}

func assertAssistantThinking(t *testing.T, line map[string]any, wantContent, wantSig string) {
	t.Helper()
	if line["type"] != "assistant.thinking" {
		t.Fatalf("want assistant.thinking, got %q", line["type"])
	}
	if line["content"] != wantContent {
		t.Errorf("assistant.thinking content = %q, want %q", line["content"], wantContent)
	}
	if line["signature"] != wantSig {
		t.Errorf("assistant.thinking signature = %q, want %q", line["signature"], wantSig)
	}
}

func assertAssistantMessage(t *testing.T, line map[string]any, wantContent, wantCallID, wantTool string) {
	t.Helper()
	if line["type"] != "assistant.message" {
		t.Fatalf("want assistant.message, got %q", line["type"])
	}
	if line["content"] != wantContent {
		t.Errorf("assistant.message content = %q, want %q", line["content"], wantContent)
	}
	rawCalls, hasCalls := line["tool_calls"]
	if wantCallID == "" {
		if hasCalls && rawCalls != nil {
			t.Errorf("assistant.message tool_calls = %v, want absent/nil", rawCalls)
		}
		return
	}
	calls, ok := rawCalls.([]any)
	if !ok || len(calls) != 1 {
		t.Fatalf("assistant.message tool_calls = %v, want one call", rawCalls)
	}
	call, ok := calls[0].(map[string]any)
	if !ok {
		t.Fatalf("tool call is %T, want map", calls[0])
	}
	if call["id"] != wantCallID {
		t.Errorf("tool call id = %q, want %q", call["id"], wantCallID)
	}
	if call["tool"] != wantTool {
		t.Errorf("tool call tool = %q, want %q", call["tool"], wantTool)
	}
}

func assertApprovalScopeGranted(t *testing.T, line map[string]any, wantAction, wantPattern string) {
	t.Helper()
	if line["type"] != "approval.scope_granted" {
		t.Fatalf("want approval.scope_granted, got %q", line["type"])
	}
	if line["action"] != wantAction {
		t.Errorf("approval.scope_granted action = %q, want %q", line["action"], wantAction)
	}
	if line["pattern"] != wantPattern {
		t.Errorf("approval.scope_granted pattern = %q, want %q", line["pattern"], wantPattern)
	}
}

func assertToolCompleted(t *testing.T, line map[string]any, wantID string, wantOK bool, wantContent string, wantTruncated bool) {
	t.Helper()
	if line["type"] != "tool.completed" {
		t.Fatalf("want tool.completed, got %q", line["type"])
	}
	if line["id"] != wantID {
		t.Errorf("tool.completed id = %q, want %q", line["id"], wantID)
	}
	if line["ok"] != wantOK {
		t.Errorf("tool.completed ok = %v, want %v", line["ok"], wantOK)
	}
	if line["content"] != wantContent {
		t.Errorf("tool.completed content = %q, want %q", line["content"], wantContent)
	}
	if line["truncated"] != wantTruncated {
		t.Errorf("tool.completed truncated = %v, want %v", line["truncated"], wantTruncated)
	}
}

func assertTurnCompleted(t *testing.T, line map[string]any, wantStatus string) {
	t.Helper()
	if line["type"] != "turn.completed" {
		t.Fatalf("want turn.completed, got %q", line["type"])
	}
	if line["status"] != wantStatus {
		t.Errorf("turn.completed status = %q, want %q", line["status"], wantStatus)
	}
}
