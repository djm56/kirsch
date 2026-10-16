package app

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestResumeCompletesPartialTurn is the m4-d5 integration test: a session with
// a partial turn (user message, assistant tool call, tool result, and a granted
// scope) is resumed, the TUI receives the reconstructed transcript, grants are
// restored, and the next Submit continues the conversation with a valid message
// array for the provider.
func TestResumeCompletesPartialTurn(t *testing.T) {
	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")

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

	// Create a partial session that ends mid-task: the assistant asked for a
	// tool and the user/tool layer answered it, then the process quit before the
	// assistant could continue.
	layout, err := session.NewLayout(stateRoot, ws.CanonicalRoot)
	if err != nil {
		t.Fatalf("NewLayout: %v", err)
	}
	path, _, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	s, err := session.New(path, session.WithCadence(time.Hour))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Record(agent.RecordAppend, userTextMessage("plan the refactor"))
	s.Record(agent.RecordAppend, assistantTextToolMessage("I'll read main.go", "call-1", "read_file", `{"path":"main.go"}`))
	s.Record(agent.RecordAppend, userToolResultMessage("call-1", "package main\n", false))
	s.RecordScopeGranted("run_command", "go test ./...")
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Resume the session through the app.
	c := newCollector(2)
	a := New(ws, cfg, telemetry.Disabled())
	a.sendFn = c.send
	defer a.Close()

	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("ResumeLastSession: %v", err)
	}

	msgs := c.wait(t, 2*time.Second)
	var loaded tui.TranscriptLoadedMsg
	for _, m := range msgs {
		if lm, ok := m.(tui.TranscriptLoadedMsg); ok {
			loaded = lm
		}
	}
	if loaded.Items == nil {
		t.Fatalf("TranscriptLoadedMsg not received: got %+v", msgs)
	}
	if len(loaded.Items) != 3 {
		t.Fatalf("transcript items = %d, want 3", len(loaded.Items))
	}
	if loaded.Items[0].Kind != tui.KindUser {
		t.Errorf("item[0].Kind = %v, want KindUser", loaded.Items[0].Kind)
	}
	if loaded.Items[1].Kind != tui.KindAssistant {
		t.Errorf("item[1].Kind = %v, want KindAssistant", loaded.Items[1].Kind)
	}
	if loaded.Items[2].Kind != tui.KindTool {
		t.Errorf("item[2].Kind = %v, want KindTool", loaded.Items[2].Kind)
	}

	// The session-scoped grant must be restored into the policy.
	grants := a.Grants()
	if len(grants) != 1 || grants[0] != "go test ./..." {
		t.Fatalf("grants = %v, want [go test ./...]", grants)
	}

	// Now complete the turn. The fake provider records the request it receives.
	fake := provider.NewFake(provider.Turn{Text: "done"})
	a.providerBuilder = func(_ string, _ config.EndpointConfig, _, _, _ string, _ *telemetry.Logger) (provider.Provider, error) {
		return fake, nil
	}

	c2 := newCollector(6)
	a.sendFn = c2.send

	a.Submit("continue", func(string) string { return "fake-key" })

	// Wait for the BudgetMsg, assistant text deltas, usage, and turn completion.
	c2.wait(t, 2*time.Second)

	requests := fake.Requests()
	if len(requests) != 1 {
		t.Fatalf("provider requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if len(req.Messages) != 3 {
		t.Fatalf("request messages = %d, want 3", len(req.Messages))
	}

	// The resumed conversation must be present and every tool call answered.
	if req.Messages[0].Role != provider.RoleUser {
		t.Errorf("message[0].Role = %v, want user", req.Messages[0].Role)
	}
	if req.Messages[1].Role != provider.RoleAssistant {
		t.Errorf("message[1].Role = %v, want assistant", req.Messages[1].Role)
	}
	if req.Messages[2].Role != provider.RoleUser {
		t.Errorf("message[2].Role = %v, want user", req.Messages[2].Role)
	}

	var callIDs []string
	var resultIDs []string
	for _, msg := range req.Messages {
		for _, b := range msg.Content {
			switch b.Kind {
			case provider.BlockToolCall:
				if b.ToolCall == nil {
					t.Fatal("tool call block with nil ToolCall")
				}
				callIDs = append(callIDs, b.ToolCall.ID)
			case provider.BlockToolResult:
				if b.ToolResult == nil {
					t.Fatal("tool result block with nil ToolResult")
				}
				resultIDs = append(resultIDs, b.ToolResult.CallID)
			case provider.BlockText, provider.BlockThinking:
				// Text and thinking blocks carry no call IDs to match.
			}
		}
	}
	if len(callIDs) != 1 || callIDs[0] != "call-1" {
		t.Fatalf("tool calls = %v, want [call-1]", callIDs)
	}
	if len(resultIDs) != 1 || resultIDs[0] != "call-1" {
		t.Fatalf("tool results = %v, want [call-1]", resultIDs)
	}

	// The in-memory conversation grew by the merged follow-up user message and
	// the assistant response, proving the resumed conversation was the pending
	// state.
	if len(a.conv.Messages) != 4 {
		t.Fatalf("conv.Messages = %d, want 4", len(a.conv.Messages))
	}
}

// TestResumeLastSessionNoSession verifies that ResumeLastSession surfaces the
// sentinel error when the workspace has no recorded session.
func TestResumeLastSessionNoSession(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")

	c := newCollector(0)
	a := testAppWithConfig(t, cfg, c)
	defer a.Close()

	err := a.ResumeLastSession()
	if !errors.Is(err, session.ErrNoSession) {
		t.Fatalf("ResumeLastSession error = %v, want ErrNoSession", err)
	}
}

// TestStartNewSessionCreatesStore verifies that StartNewSession opens a new
// session store and makes it available for subsequent turns.
func TestStartNewSessionCreatesStore(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(tmp, "sessions")

	c := newCollector(0)
	a := testAppWithConfig(t, cfg, c)
	defer a.Close()

	if err := a.StartNewSession(); err != nil {
		t.Fatalf("StartNewSession: %v", err)
	}
	if a.store == nil {
		t.Fatal("store is nil after StartNewSession")
	}
}

func userTextMessage(text string) agent.Message {
	return agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	}
}

func assistantTextToolMessage(text, callID, toolName, input string) agent.Message {
	content := []agent.Block{}
	if text != "" {
		content = append(content, agent.Block{Kind: agent.BlockText, Text: text})
	}
	content = append(content, agent.Block{
		Kind: agent.BlockToolCall,
		ToolCall: &agent.ToolCall{
			ID:    callID,
			Name:  toolName,
			Input: json.RawMessage(input),
		},
	})
	return agent.Message{Role: agent.RoleAssistant, Content: content}
}

func userToolResultMessage(callID, content string, isErr bool) agent.Message {
	return agent.Message{
		Role: agent.RoleUser,
		Content: []agent.Block{{
			Kind: agent.BlockToolResult,
			Result: &agent.ResultBlock{
				CallID:  callID,
				Content: content,
				IsError: isErr,
			},
		}},
	}
}
