package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
)

// fakeModel implements agent.Model for adapter tests. stream is invoked for each
// agent round; it emits events through onEvent and returns the assistant message
// for that round.
type fakeModel struct {
	stream func(ctx context.Context, onEvent func(agent.Event)) (agent.Message, error)
	info   agent.ModelInfo
}

func (f *fakeModel) Stream(ctx context.Context, _ agent.Request, onEvent func(agent.Event)) error {
	_, err := f.stream(ctx, onEvent)
	return err
}

func (f *fakeModel) Info() agent.ModelInfo { return f.info }

// fakeTool is a tool registered with the app's registry so agent-driven tool
// calls can be exercised without a real provider.
type fakeTool struct{}

func (fakeTool) Name() string            { return "fake_tool" }
func (fakeTool) Description() string     { return "a fake tool" }
func (fakeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (fakeTool) Invoke(ctx context.Context, _ json.RawMessage) tool.Result {
	return tool.OKResult("done", "done", false)
}

// fakeApprovalTool is a tool that requires approval. Its Invoke calls the
// configured Approver and succeeds only when the approver returns
// policy.DecisionAllow. It is used to verify that agent-driven tool calls
// actually travel through the approval path instead of being executed directly.
type fakeApprovalTool struct {
	approver tool.Approver
	once     sync.Once
	called   chan struct{} // closed the first time Invoke reaches the approver
}

func (*fakeApprovalTool) Name() string            { return "fake_approval_tool" }
func (*fakeApprovalTool) Description() string     { return "a fake approval tool" }
func (*fakeApprovalTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *fakeApprovalTool) Invoke(ctx context.Context, _ json.RawMessage) tool.Result {
	if t.called != nil {
		t.once.Do(func() { close(t.called) })
	}
	dec := t.approver.Request(ctx, tool.ApprovalRequest{
		Operation:   policy.OperationCommand,
		Description: "fake approval request",
		Argv:        []string{"fake"},
	})
	if dec == policy.DecisionAllow {
		return tool.OKResult("approved", "approved", false)
	}
	return tool.Fail(tool.KindPolicyDenied, "denied")
}

// TestRunTurnMapsTextDeltaToAssistantTextDeltaMsg checks that an EventTextDelta
// becomes an AssistantTextDeltaMsg with the same delta text.
func TestRunTurnMapsTextDeltaToAssistantTextDeltaMsg(t *testing.T) {
	c := newCollector(2)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "Hello"})
				onEvent(agent.Event{Type: agent.EventMessageDone})
				return agent.Message{Role: agent.RoleAssistant}, nil
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 2*time.Second)

	got := findMsg[tui.AssistantTextDeltaMsg](msgs)
	if got == nil {
		t.Fatalf("want AssistantTextDeltaMsg, got %#v", msgs)
	}
	if got.Delta != "Hello" {
		t.Errorf("delta = %q, want %q", got.Delta, "Hello")
	}
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Error("expected TurnCompleteMsg")
	}
}

// TestRunTurnMapsThinkingDeltaToThinkingDeltaMsg checks that an
// EventThinkingDelta opens a thinking card in the adapter's message stream.
func TestRunTurnMapsThinkingDeltaToThinkingDeltaMsg(t *testing.T) {
	c := newCollector(2)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				onEvent(agent.Event{Type: agent.EventThinkingDelta, Text: "planning"})
				onEvent(agent.Event{Type: agent.EventThinkingDone, Thinking: &agent.Thinking{Text: "planning"}})
				onEvent(agent.Event{Type: agent.EventMessageDone})
				return agent.Message{Role: agent.RoleAssistant}, nil
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 2*time.Second)

	got := findMsg[tui.ThinkingDeltaMsg](msgs)
	if got == nil {
		t.Fatalf("want ThinkingDeltaMsg, got %#v", msgs)
	}
	if got.Delta != "planning" {
		t.Errorf("delta = %q, want %q", got.Delta, "planning")
	}
	if findMsgType[tui.ErrorMsg](msgs) {
		t.Error("thinking path leaked an infrastructure ErrorMsg")
	}
}

// TestRunTurnErrorMsgPresenceControl is the positive control for absence
// assertions like the one in TestRunTurnMapsThinkingDeltaToThinkingDeltaMsg: it
// proves the collector really can observe an ErrorMsg when one is produced.
func TestRunTurnErrorMsgPresenceControl(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, _ func(agent.Event)) (agent.Message, error) {
				return agent.Message{}, errors.New("boom")
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 2*time.Second)

	if !findMsgType[tui.TurnErrorMsg](msgs) {
		t.Fatalf("want TurnErrorMsg, got %#v", msgs)
	}
}

// TestRunTurnMapsUsageToUsageMsg checks that a usage payload on
// EventMessageDone is forwarded as a plain-field UsageMsg.
func TestRunTurnMapsUsageToUsageMsg(t *testing.T) {
	c := newCollector(3)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "x"})
				onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &agent.Usage{
					InputTokens: 10, OutputTokens: 5,
					CacheReadTokens: 1, CacheWriteTokens: 2,
				}})
				return agent.Message{Role: agent.RoleAssistant}, nil
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 2*time.Second)

	got := findMsg[tui.UsageMsg](msgs)
	if got == nil {
		t.Fatalf("want UsageMsg, got %#v", msgs)
	}
	if got.InputTokens != 10 || got.OutputTokens != 5 ||
		got.CacheReadTokens != 1 || got.CacheWriteTokens != 2 {
		t.Errorf("usage fields = %+v, want {10 5 1 2}", got)
	}
}

// TestRunTurnMapsToolCallToToolStartedAndCompleted checks that EventToolCallStart
// maps to ToolStartedMsg and that the agent-driven tool invocation produces a
// matching ToolCompletedMsg with the same app-side ID.
func TestRunTurnMapsToolCallToToolStartedAndCompleted(t *testing.T) {
	c := newCollector(4)
	a := testApp(t, "repo-small", c)
	defer a.Close()
	a.reg.Register(fakeTool{})

	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				onEvent(agent.Event{Type: agent.EventToolCallStart, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_tool"}})
				onEvent(agent.Event{Type: agent.EventToolCallEnd, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_tool"}})
				onEvent(agent.Event{Type: agent.EventMessageDone})
				return agent.Message{
					Role: agent.RoleAssistant,
					Content: []agent.Block{{
						Kind:     agent.BlockToolCall,
						ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_tool"},
					}},
				}, nil
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 2*time.Second)

	started := findMsg[tui.ToolStartedMsg](msgs)
	completed := findMsg[tui.ToolCompletedMsg](msgs)
	if started == nil || completed == nil {
		t.Fatalf("want ToolStartedMsg and ToolCompletedMsg, got %#v", msgs)
	}
	if started.ID != completed.ID {
		t.Errorf("started id %d != completed id %d", started.ID, completed.ID)
	}
	if started.Name != "fake_tool" {
		t.Errorf("started name = %q, want fake_tool", started.Name)
	}
	if !completed.OK {
		t.Errorf("tool failed: %s %s", completed.ErrorKind, completed.ErrorMsg)
	}
}

// TestRunTurnMapsContextOverflowToTurnErrorMsg checks the sentinel mapping for
// the context_overflow error kind required by the milestone.
func TestRunTurnMapsContextOverflowToTurnErrorMsg(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, _ func(agent.Event)) (agent.Message, error) {
				return agent.Message{}, agent.ErrContextOverflow
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 2*time.Second)

	errMsg := findMsg[tui.TurnErrorMsg](msgs)
	if errMsg == nil {
		t.Fatalf("want TurnErrorMsg, got %#v", msgs)
	}
	if errMsg.Kind != "context_overflow" {
		t.Errorf("kind = %q, want context_overflow", errMsg.Kind)
	}
	if !strings.Contains(errMsg.Message, "context overflow") {
		t.Errorf("message = %q, want it to mention context overflow", errMsg.Message)
	}
}

// TestRunTurnMapsMaxTurnsExceededToTurnErrorMsg checks the sentinel mapping for
// the max_turns_exceeded error kind.
func TestRunTurnMapsMaxTurnsExceededToTurnErrorMsg(t *testing.T) {
	c := newCollector(51)
	a := testApp(t, "repo-small", c)
	defer a.Close()
	a.reg.Register(fakeTool{})

	round := 0
	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				onEvent(agent.Event{Type: agent.EventToolCallStart, ToolCall: &agent.ToolCall{ID: "call", Name: "fake_tool"}})
				onEvent(agent.Event{Type: agent.EventToolCallEnd, ToolCall: &agent.ToolCall{ID: "call", Name: "fake_tool"}})
				onEvent(agent.Event{Type: agent.EventMessageDone})
				round++
				return agent.Message{
					Role: agent.RoleAssistant,
					Content: []agent.Block{{
						Kind:     agent.BlockToolCall,
						ToolCall: &agent.ToolCall{ID: "call", Name: "fake_tool"},
					}},
				}, nil
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 5*time.Second)

	if !findMsgType[tui.TurnErrorMsg](msgs) {
		t.Fatalf("want TurnErrorMsg, got %#v", msgs)
	}
	errMsg := findMsg[tui.TurnErrorMsg](msgs)
	if errMsg.Kind != "max_turns_exceeded" {
		t.Errorf("kind = %q, want max_turns_exceeded", errMsg.Kind)
	}
}

// TestRunTurnCancellationMapsToTurnCancelledMsg checks that cancelling the turn
// context produces a TurnCancelledMsg, not a TurnErrorMsg.
func TestRunTurnCancellationMapsToTurnCancelledMsg(t *testing.T) {
	c := newCollector(1)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	block := make(chan struct{})
	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(ctx context.Context, _ func(agent.Event)) (agent.Message, error) {
				select {
				case <-block:
				case <-ctx.Done():
				}
				return agent.Message{}, ctx.Err()
			},
		},
	}

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	// Give RunTurn time to enter Stream before we cancel.
	time.Sleep(50 * time.Millisecond)
	a.CancelTurn()
	close(block)

	msgs := c.wait(t, 2*time.Second)
	if !findMsgType[tui.TurnCancelledMsg](msgs) {
		t.Fatalf("want TurnCancelledMsg, got %#v", msgs)
	}
	if findMsgType[tui.TurnErrorMsg](msgs) {
		t.Error("cancellation produced a TurnErrorMsg")
	}
}

// TestRunTurnApprovalToolUsesApprovalPath checks that an agent-driven tool call
// whose tool requires approval blocks on the approval path through reg.Invoke
// -> the tool's Invoke -> approvalAdapter.Request -> App.RequestToolApproval.
// If the adapter routed execution around the approval path, the tool would run
// without producing ApprovalRequestedMsg and this test would fail.
func TestRunTurnApprovalToolUsesApprovalPath(t *testing.T) {
	c := newCollector(5)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	called := make(chan struct{})
	a.reg.Register(&fakeApprovalTool{
		approver: &approvalAdapter{app: a},
		called:   called,
	})

	round := 0
	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				if round == 0 {
					round++
					onEvent(agent.Event{Type: agent.EventToolCallStart, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_approval_tool"}})
					onEvent(agent.Event{Type: agent.EventToolCallEnd, ToolCall: &agent.ToolCall{ID: "call-1", Name: "fake_approval_tool"}})
					onEvent(agent.Event{Type: agent.EventMessageDone})
					return agent.Message{Role: agent.RoleAssistant}, nil
				}
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "done"})
				onEvent(agent.Event{Type: agent.EventMessageDone})
				return agent.Message{Role: agent.RoleAssistant}, nil
			},
		},
	}

	go func() {
		select {
		case <-called:
			// The first approval ID allocated by App.Request is 1 for a fresh app.
			a.ResolveToolApproval(1, policy.DecisionAllow)
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for approval tool to request approval")
		}
	}()

	a.RunTurn(ag, &agent.Conversation{}, "hi")
	msgs := c.wait(t, 5*time.Second)

	if !findMsgType[tui.ApprovalRequestedMsg](msgs) {
		t.Fatalf("approval path was not reached: want ApprovalRequestedMsg, got %#v", msgs)
	}
	completed := findMsg[tui.ToolCompletedMsg](msgs)
	if completed == nil {
		t.Fatalf("want ToolCompletedMsg, got %#v", msgs)
	}
	if !completed.OK {
		t.Errorf("approval tool did not succeed: kind=%s msg=%s", completed.ErrorKind, completed.ErrorMsg)
	}
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Error("want TurnCompleteMsg")
	}
}

// TestRunTurnUsageNotStoredInConversation checks that EventUsage is forwarded as
// a UsageMsg for the TUI but is never persisted inside the conversation's
// message transcript. Usage lives in the turn metadata, not in conv.Messages.
func TestRunTurnUsageNotStoredInConversation(t *testing.T) {
	c := newCollector(3)
	a := testApp(t, "repo-small", c)
	defer a.Close()

	conv := &agent.Conversation{}
	ag := &agent.Agent{
		Model: &fakeModel{
			stream: func(_ context.Context, onEvent func(agent.Event)) (agent.Message, error) {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "hello"})
				onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &agent.Usage{
					InputTokens: 10, OutputTokens: 5,
				}})
				return agent.Message{Role: agent.RoleAssistant}, nil
			},
		},
	}

	a.RunTurn(ag, conv, "hi")
	msgs := c.wait(t, 2*time.Second)

	if !findMsgType[tui.UsageMsg](msgs) {
		t.Fatalf("want UsageMsg in TUI stream, got %#v", msgs)
	}
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Error("want TurnCompleteMsg")
	}

	if len(conv.Messages) != 2 {
		t.Fatalf("want 2 conversation messages, got %d: %+v", len(conv.Messages), conv.Messages)
	}
	asst := conv.Messages[1]
	if asst.Role != agent.RoleAssistant {
		t.Fatalf("second message role is %v, want assistant", asst.Role)
	}
	if len(asst.Content) != 1 || asst.Content[0].Kind != agent.BlockText || asst.Content[0].Text != "hello" {
		t.Fatalf("usage leaked into assistant message content: %+v", asst.Content)
	}
}

// findMsg returns the first message of type T in msgs, or nil if none.
func findMsg[T any](msgs []any) *T {
	for _, m := range msgs {
		if v, ok := m.(T); ok {
			return &v
		}
	}
	return nil
}

// findMsgType reports whether msgs contains at least one message of type T.
func findMsgType[T any](msgs []any) bool {
	return findMsg[T](msgs) != nil
}
