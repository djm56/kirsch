package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// TestPlanCompactionNotEnoughTurns tests that fewer than 6 turns returns not ok.
// Each turn count runs as its own subtest, with and without tool-result user
// messages. The tool-result variants are the mirror of
// TestPlanCompactionToolResultOnlyDoesNotStartTurn: tool-result-only messages
// must not push 5 real turns over the 6-turn threshold.
func TestPlanCompactionNotEnoughTurns(t *testing.T) {
	for _, interleave := range []bool{false, true} {
		for numTurns := 0; numTurns <= 5; numTurns++ {
			t.Run(fmt.Sprintf("turns=%d/toolresults=%v", numTurns, interleave), func(t *testing.T) {
				msgs := buildMessages(numTurns, interleave)
				from, to, ok := PlanCompaction(msgs)
				if ok {
					t.Errorf("%d real turns (tool-result messages=%v): got ok=true (from=%d, to=%d), want ok=false", numTurns, interleave, from, to)
				}
				if from != 0 || to != 0 {
					t.Errorf("%d real turns (tool-result messages=%v): got from=%d, to=%d, want 0, 0", numTurns, interleave, from, to)
				}
			})
		}
	}
}

// TestPlanCompactionEmptySlice tests that an empty slice returns not ok.
func TestPlanCompactionEmptySlice(t *testing.T) {
	msgs := []agent.Message{}
	from, to, ok := PlanCompaction(msgs)
	if ok {
		t.Errorf("PlanCompaction with empty slice: got ok=true, want false")
	}
	if from != 0 || to != 0 {
		t.Errorf("PlanCompaction with empty slice: got from=%d, to=%d, want 0, 0", from, to)
	}
}

// TestPlanCompactionSevenTurns tests that 7 turns returns the correct range.
func TestPlanCompactionSevenTurns(t *testing.T) {
	msgs := buildMessages(7, true)
	from, to, ok := PlanCompaction(msgs)
	if !ok {
		t.Fatal("PlanCompaction with 7 turns: got ok=false, want true")
	}
	// With 7 turns (each turn is 3 messages: user text, assistant tool call, user result):
	// Turn 0 starts at msg 0
	// Turn 1 starts at msg 3
	// Turn 2 starts at msg 6
	// Turn 3 starts at msg 9 (4th-from-last turn)
	// Turn 4 starts at msg 12
	// Turn 5 starts at msg 15
	// Turn 6 starts at msg 18
	// We keep msg 0 + last 4 turns (3-6), compact msgs 1-8
	// from = 1 (keep only msg 0)
	// to = msg 9-1 = 8 (just before turn 3, the 4th-from-last)
	if from != 1 || to != 8 {
		t.Errorf("PlanCompaction with 7 turns: got from=%d, to=%d, want 1, 8", from, to)
	}
}

// TestPlanCompactionFirstMessageKept tests that the first message is kept.
func TestPlanCompactionFirstMessageKept(t *testing.T) {
	msgs := []agent.Message{
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "initial task"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 1"}}},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "follow-up 1"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 2"}}},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "follow-up 2"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 3"}}},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "follow-up 3"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 4"}}},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "follow-up 4"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 5"}}},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "follow-up 5"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 6"}}},
		{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: "follow-up 6"}}},
		{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: "response 7"}}},
	}
	from, _, ok := PlanCompaction(msgs)
	if !ok {
		t.Fatal("PlanCompaction: got ok=false, want true")
	}
	// from should be exactly 1 to keep only the first message (index 0).
	if from != 1 {
		t.Errorf("from=%d should be 1 to keep only the first message", from)
	}
}

// textMsg, callMsg, resultMsg and replyMsg build the single-block messages used
// by TestPlanCompactionToolResultOnlyDoesNotStartTurn.
func textMsg(text string) agent.Message {
	return agent.Message{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: text}}}
}

func callMsg(id string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockToolCall, ToolCall: &agent.ToolCall{ID: id, Name: "run", Input: json.RawMessage("null")}}}}
}

func resultMsg(id string) agent.Message {
	return agent.Message{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockToolResult, Result: &agent.ResultBlock{CallID: id, Content: "result"}}}}
}

func replyMsg(text string) agent.Message {
	return agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: text}}}
}

// TestPlanCompactionToolResultOnlyDoesNotStartTurn tests that a user message
// holding only tool results does not start a turn.
//
// The fixture has exactly 6 real turns, and tool-result-only messages sit
// inside the last four of them (indices 8, 12, 18) as well as in turn 0
// (index 2). A planner that counted those messages as turn starts would see 10
// starts, take the 4th-from-last at index 12, and return to=11 instead of 5.
// This test fails on that planner without help from any other test.
func TestPlanCompactionToolResultOnlyDoesNotStartTurn(t *testing.T) {
	msgs := []agent.Message{
		textMsg("task 1"), // 0  turn 0
		callMsg("a"),      // 1
		resultMsg("a"),    // 2  tool-result-only
		replyMsg("done"),  // 3
		textMsg("task 2"), // 4  turn 1
		replyMsg("r2"),    // 5
		textMsg("task 3"), // 6  turn 2 (4th-from-last)
		callMsg("b"),      // 7
		resultMsg("b"),    // 8  tool-result-only
		replyMsg("r3"),    // 9
		textMsg("task 4"), // 10 turn 3
		callMsg("c"),      // 11
		resultMsg("c"),    // 12 tool-result-only
		replyMsg("r4"),    // 13
		textMsg("task 5"), // 14 turn 4
		replyMsg("r5"),    // 15
		textMsg("task 6"), // 16 turn 5
		callMsg("d"),      // 17
		resultMsg("d"),    // 18 tool-result-only
		replyMsg("r6"),    // 19
	}
	from, to, ok := PlanCompaction(msgs)
	if !ok {
		t.Fatal("6 real turns with tool-result-only messages: got ok=false, want true")
	}
	if from != 1 {
		t.Errorf("6 real turns with tool-result-only messages: got from=%d, want 1", from)
	}
	if to != 5 {
		t.Errorf("got to=%d, want 5 (the message before the 4th-from-last real turn, at index 6); to=11 means tool-result-only messages were counted as turn starts", to)
	}
}

// FakeModel implements agent.Model for testing.
type FakeModel struct {
	onStream func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error
}

func (m *FakeModel) Stream(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
	if m.onStream != nil {
		return m.onStream(ctx, req, onEvent)
	}
	onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
	return nil
}

func (m *FakeModel) Info() agent.ModelInfo {
	return agent.ModelInfo{ContextWindow: 4096}
}

// TestThinkingBlocksNotInSummaryRequest tests that thinking blocks don't appear in the summary request.
// It verifies absence of thinking text with positive controls for other content.
func TestThinkingBlocksNotInSummaryRequest(t *testing.T) {
	msgs := buildMessages(7, true)

	// With 7 turns the compacted range is [1, 8] (SevenTurns pins it). Assistant
	// messages sit at indices 1, 4 and 7 inside it; each keeps its tool call and
	// gains a thinking block plus a unique text block. Only assistant positions
	// are rewritten, so no user message changes and the turn starts do not move.
	from, to, ok := PlanCompaction(msgs)
	if !ok || from > 1 || to < 7 {
		t.Fatalf("fixture assumption broken: PlanCompaction = (%d, %d, %v), want a range covering 1..7", from, to, ok)
	}
	thinkingAt := map[int]agent.Block{
		1: {Kind: agent.BlockThinking, Thinking: &agent.Thinking{Text: "thinking 1", DropOnSummary: true}},
		4: {Kind: agent.BlockThinking, Thinking: &agent.Thinking{Text: "thinking 2", DropOnSummary: true}},
		// A redacted block with a non-empty sentinel and DropOnSummary false:
		// a renderer that drops only DropOnSummary blocks would leak it.
		7: {Kind: agent.BlockThinking, Thinking: &agent.Thinking{Text: "REDACTED_SHOULD_BE_DROPPED_sentinel", Redacted: true, Data: "redacted_data_12345"}},
	}
	uniqueText := map[int]string{1: "unique_response_abc123", 4: "unique_task_xyz789", 7: "unique_middle_text_def456"}
	for idx, th := range thinkingAt {
		if msgs[idx].Role != agent.RoleAssistant || len(msgs[idx].Content) != 2 || msgs[idx].Content[1].Kind != agent.BlockToolCall {
			t.Fatalf("fixture assumption broken: msgs[%d] is not an assistant text+tool-call message", idx)
		}
		msgs[idx].Content = []agent.Block{
			th,
			{Kind: agent.BlockText, Text: uniqueText[idx]},
			msgs[idx].Content[1],
		}
	}

	capturedReq := &agent.Request{}
	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			*capturedReq = req
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			return nil
		},
	}

	_, err := Compact(context.Background(), model, msgs, 1000)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Get the rendered request text from the user message content.
	if len(capturedReq.Messages) == 0 || len(capturedReq.Messages[0].Content) == 0 {
		t.Fatal("Request has no user message or content")
	}
	renderedText := capturedReq.Messages[0].Content[0].Text

	// Check that thinking text doesn't appear.
	if strings.Contains(renderedText, "thinking 1") || strings.Contains(renderedText, "thinking 2") {
		t.Errorf("Summary request contains thinking text, should be dropped")
	}
	// Check that the redacted thinking sentinel text doesn't appear.
	if strings.Contains(renderedText, "REDACTED_SHOULD_BE_DROPPED_sentinel") {
		t.Errorf("Summary request contains redacted thinking text, should be dropped")
	}
	// Check that redacted data doesn't appear.
	if strings.Contains(renderedText, "redacted_data_12345") {
		t.Errorf("Summary request contains redacted thinking data, should be dropped")
	}

	// Positive controls: verify that other text IS present (using unique strings).
	if !strings.Contains(renderedText, "unique_response_abc123") {
		t.Errorf("Summary request missing non-thinking text 'unique_response_abc123'")
	}
	if !strings.Contains(renderedText, "unique_task_xyz789") {
		t.Errorf("Summary request missing non-thinking text 'unique_task_xyz789'")
	}
	if !strings.Contains(renderedText, "unique_middle_text_def456") {
		t.Errorf("Summary request missing text 'unique_middle_text_def456'")
	}
	// Tool call name and input should appear.
	if !strings.Contains(renderedText, "test_tool") {
		t.Errorf("Summary request missing tool call name 'test_tool'")
	}
	if !strings.Contains(renderedText, "value") {
		t.Errorf("Summary request missing tool input content")
	}
}

// TestCompactionSuccessPathImmutability tests that the input messages are not mutated after successful compaction.
// This verifies that deep-copying prevents aliasing issues.
// It mutates tail blocks through their pointers to ensure deep copy protects them.
func TestCompactionSuccessPathImmutability(t *testing.T) {
	msgs := buildMessages(7, true)

	// Take a snapshot of the input before compaction.
	inputSnapshot := make([]agent.Message, len(msgs))
	for i := range msgs {
		inputSnapshot[i] = agent.Message{
			Role:    msgs[i].Role,
			Content: cloneBlocks(msgs[i].Content),
		}
	}

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			return nil
		},
	}

	comp, err := Compact(context.Background(), model, msgs, 1000)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Mutate the kept messages in the result through their slices and pointers.
	// Each mutation must land on something, or the isolation check proves nothing.
	mutatedText, mutatedCall, mutatedResult := 0, 0, 0
	comp.Messages[0].Content[0].Text = "MUTATED_TEXT"
	mutatedText++
	for _, m := range comp.Messages {
		for _, b := range m.Content {
			if b.Kind == agent.BlockToolCall && b.ToolCall != nil {
				b.ToolCall.Name = "MUTATED_TOOL"
				mutatedCall++
			}
			if b.Kind == agent.BlockToolResult && b.Result != nil {
				b.Result.Content = "MUTATED_RESULT"
				mutatedResult++
			}
		}
	}
	if mutatedText == 0 || mutatedCall == 0 || mutatedResult == 0 {
		t.Fatalf("fixture assumption broken: mutated %d texts, %d tool calls, %d tool results; want each > 0", mutatedText, mutatedCall, mutatedResult)
	}

	// Verify that the input messages are unchanged.
	if !reflect.DeepEqual(msgs, inputSnapshot) {
		t.Fatal("Input messages were mutated after successful compaction through pointer aliasing")
	}
}

// TestCompactionRequestShape tests the shape of the summary request.
func TestCompactionRequestShape(t *testing.T) {
	msgs := buildMessages(7, true)

	capturedReq := &agent.Request{}
	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			*capturedReq = req
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			return nil
		},
	}

	maxTokens := 1000
	_, err := Compact(context.Background(), model, msgs, maxTokens)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Verify exactly one user message.
	if len(capturedReq.Messages) != 1 {
		t.Errorf("Expected exactly 1 message in request, got %d", len(capturedReq.Messages))
	}
	if len(capturedReq.Messages) > 0 && capturedReq.Messages[0].Role != agent.RoleUser {
		t.Errorf("Expected user message, got role %v", capturedReq.Messages[0].Role)
	}

	// Verify the message has content.
	if len(capturedReq.Messages) > 0 && len(capturedReq.Messages[0].Content) == 0 {
		t.Errorf("User message has no content blocks")
	}

	// Verify the first block is text and non-empty.
	if len(capturedReq.Messages) > 0 && len(capturedReq.Messages[0].Content) > 0 {
		if capturedReq.Messages[0].Content[0].Kind != agent.BlockText {
			t.Errorf("Expected first block to be text, got %v", capturedReq.Messages[0].Content[0].Kind)
		}
		if capturedReq.Messages[0].Content[0].Text == "" {
			t.Errorf("User message text is empty")
		}
	}

	// Verify MaxTokens.
	if capturedReq.MaxTokens != maxTokens {
		t.Errorf("Expected MaxTokens=%d, got %d", maxTokens, capturedReq.MaxTokens)
	}

	// Verify System contains all section names from compactionSystemPrompt.
	sectionNames := []string{
		"Objective",
		"Files inspected",
		"Key findings",
		"Files changed",
		"Commands run",
		"Limitations",
		"Open questions",
	}
	for _, section := range sectionNames {
		if !strings.Contains(capturedReq.System, section) {
			t.Errorf("System prompt missing section: %s", section)
		}
	}

	// Verify Tools is empty.
	if len(capturedReq.Tools) > 0 {
		t.Errorf("Expected empty Tools, got %d tools", len(capturedReq.Tools))
	}
}

// TestThinkingBlocksPreservedInResult tests that thinking blocks remain in the result Messages
// with all fields (Text, Signature, Redacted, Data, DropOnSummary) preserved.
func TestThinkingBlocksPreservedInResult(t *testing.T) {
	msgs := buildMessages(7, true)

	// With 7 turns, msg 0 and msgs 9-20 are kept (SevenTurns pins from=1, to=8).
	// Add thinking blocks to assistant messages at indices 10, 13, and 16 (all inside kept tail).
	// Blocks at 10 and 13 have Signature and DropOnSummary=true.
	// Block at 16 has Redacted=true with non-empty Data.
	originalThinking := []*agent.Thinking{
		{
			Text:          "thinking 3",
			Signature:     "sig_thinking_3",
			DropOnSummary: true,
		},
		{
			Text:          "thinking 4",
			Signature:     "sig_thinking_4",
			DropOnSummary: true,
		},
		{
			Text:      "thinking redacted",
			Signature: "sig_redacted",
			Redacted:  true,
			Data:      "redacted_data_payload",
		},
	}

	// Inject thinking blocks into kept messages at indices 10, 13, and 16.
	insertIndices := []int{10, 13, 16}
	for i, idx := range insertIndices {
		if msgs[idx].Role != agent.RoleAssistant {
			t.Fatalf("fixture assumption broken: msgs[%d] is not an assistant message", idx)
		}
		msgs[idx].Content = append([]agent.Block{
			{Kind: agent.BlockThinking, Thinking: originalThinking[i]},
		}, msgs[idx].Content...)
	}

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			return nil
		},
	}

	result, err := Compact(context.Background(), model, msgs, 1000)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	// Extract all thinking blocks from result Messages.
	var foundThinking []*agent.Thinking
	for _, m := range result.Messages {
		for _, b := range m.Content {
			if b.Kind == agent.BlockThinking && b.Thinking != nil {
				foundThinking = append(foundThinking, b.Thinking)
			}
		}
	}

	// Verify all original thinking blocks were preserved with exact field equality.
	if len(foundThinking) != len(originalThinking) {
		t.Errorf("expected %d thinking blocks, got %d", len(originalThinking), len(foundThinking))
	}

	// Check that each original thinking block is in the result with exact field match.
	for _, want := range originalThinking {
		found := false
		for _, got := range foundThinking {
			if thinkingEqual(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("thinking block not fully preserved:\n  want: %+v\n  not found in result", want)
		}
	}
}

// TestCompactionCancellation tests that a cancelled compaction leaves the session uncompacted.
func TestCompactionCancellation(t *testing.T) {
	msgs := buildMessages(7, true)
	msgsCopy := make([]agent.Message, len(msgs))
	copy(msgsCopy, msgs)

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Compact(ctx, model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed with cancelled context")
	}
	if !errors.Is(err, ErrCompactionFailed) {
		t.Errorf("Expected ErrCompactionFailed, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled wrapped in error chain")
	}
	if result.Messages != nil || result.Summary != "" {
		t.Errorf("Compact should return empty Compaction on error, got %+v", result)
	}
	if !reflect.DeepEqual(msgs, msgsCopy) {
		t.Fatal("Input messages were mutated during failed compaction")
	}
}

// TestCompactionModelError tests that a model error is wrapped correctly.
func TestCompactionModelError(t *testing.T) {
	msgs := buildMessages(7, true)
	msgsCopy := make([]agent.Message, len(msgs))
	copy(msgsCopy, msgs)

	modelErr := errors.New("model unavailable")
	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			return modelErr
		},
	}

	result, err := Compact(context.Background(), model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed")
	}
	if !errors.Is(err, ErrCompactionFailed) {
		t.Errorf("Expected ErrCompactionFailed, got %v", err)
	}
	if !errors.Is(err, modelErr) {
		t.Errorf("Expected wrapped modelErr in error chain")
	}
	if result.Messages != nil || result.Summary != "" {
		t.Errorf("Compact should return empty Compaction on error")
	}
	if !reflect.DeepEqual(msgs, msgsCopy) {
		t.Fatal("Input messages were mutated during failed compaction")
	}
}

// TestCompactionEmptySummary tests that an empty summary is rejected.
func TestCompactionEmptySummary(t *testing.T) {
	msgs := buildMessages(7, true)
	msgsCopy := make([]agent.Message, len(msgs))
	copy(msgsCopy, msgs)

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			return nil // No events = empty summary
		},
	}

	result, err := Compact(context.Background(), model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed with empty summary")
	}
	if !errors.Is(err, ErrCompactionFailed) {
		t.Errorf("Expected ErrCompactionFailed, got %v", err)
	}
	if result.Messages != nil || result.Summary != "" {
		t.Errorf("Compact should return empty Compaction on error")
	}
	if !reflect.DeepEqual(msgs, msgsCopy) {
		t.Fatal("Input messages were mutated during failed compaction")
	}
}

// TestCompactionWhitespaceOnlySummary tests that a whitespace-only summary is rejected.
func TestCompactionWhitespaceOnlySummary(t *testing.T) {
	msgs := buildMessages(7, true)
	msgsCopy := make([]agent.Message, len(msgs))
	copy(msgsCopy, msgs)

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "  \n\t"})
			return nil
		},
	}

	result, err := Compact(context.Background(), model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed with whitespace-only summary")
	}
	if !errors.Is(err, ErrCompactionFailed) {
		t.Errorf("Expected ErrCompactionFailed, got %v", err)
	}
	if result.Messages != nil || result.Summary != "" {
		t.Errorf("Compact should return empty Compaction on error")
	}
	if !reflect.DeepEqual(msgs, msgsCopy) {
		t.Fatal("Input messages were mutated during failed compaction")
	}
}

// TestCompactionCancelAndEmit tests that the model can emit text before being cancelled,
// and the error chain captures both ErrCompactionFailed and context.Canceled.
func TestCompactionCancelAndEmit(t *testing.T) {
	msgs := buildMessages(7, true)
	msgsCopy := make([]agent.Message, len(msgs))
	copy(msgsCopy, msgs)

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			<-ctx.Done()
			return nil // Return nil even though context was cancelled
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(10 * time.Millisecond)
		cancel()
	}()

	result, err := Compact(ctx, model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed when context is cancelled")
	}
	if !errors.Is(err, ErrCompactionFailed) {
		t.Errorf("Expected ErrCompactionFailed in error chain")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled in error chain")
	}
	if result.Messages != nil || result.Summary != "" {
		t.Errorf("Compact should return empty Compaction on error")
	}
	if !reflect.DeepEqual(msgs, msgsCopy) {
		t.Fatal("Input messages were mutated during failed compaction")
	}
}

// TestCompactionEventError tests that an EventError from the model is captured in the error chain.
func TestCompactionEventError(t *testing.T) {
	msgs := buildMessages(7, true)
	msgsCopy := make([]agent.Message, len(msgs))
	copy(msgsCopy, msgs)

	modelErr := errors.New("model streaming error")
	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventError, Err: modelErr})
			return nil
		},
	}

	result, err := Compact(context.Background(), model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed when EventError is emitted")
	}
	if !errors.Is(err, ErrCompactionFailed) {
		t.Errorf("Expected ErrCompactionFailed in error chain")
	}
	if !errors.Is(err, modelErr) {
		t.Errorf("Expected model error in error chain")
	}
	if result.Messages != nil || result.Summary != "" {
		t.Errorf("Compact should return empty Compaction on error")
	}
	if !reflect.DeepEqual(msgs, msgsCopy) {
		t.Fatal("Input messages were mutated during failed compaction")
	}
}

// TestCompactionWithFiveTurns tests that Compact returns ErrNothingToCompact for 5 turns,
// without calling the model's Stream method.
func TestCompactionWithFiveTurns(t *testing.T) {
	msgs := buildMessages(5, true)

	streamCalled := false
	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			streamCalled = true
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			return nil
		},
	}

	result, err := Compact(context.Background(), model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed with ErrNothingToCompact for 5 turns")
	}
	if !errors.Is(err, ErrNothingToCompact) {
		t.Errorf("Expected ErrNothingToCompact, got %v", err)
	}
	if streamCalled {
		t.Errorf("Model's Stream should not have been called for 5 turns")
	}
	if result != nil {
		t.Errorf("Compact should return nil Compaction on ErrNothingToCompact")
	}
}

// TestCompactionUsageCapture tests that usage from EventMessageDone is captured
// in the Compaction result.
func TestCompactionUsageCapture(t *testing.T) {
	msgs := buildMessages(7, true)

	expectedUsage := agent.Usage{
		InputTokens:      100,
		OutputTokens:     50,
		CacheReadTokens:  10,
		CacheWriteTokens: 5,
	}

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary text"})
			// Emit EventMessageDone with usage, as the provider adapter does.
			onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &expectedUsage})
			return nil
		},
	}

	comp, err := Compact(context.Background(), model, msgs, 1000)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	if comp == nil {
		t.Fatal("Compact should return non-nil Compaction on success")
	}

	if comp.Usage.InputTokens != expectedUsage.InputTokens ||
		comp.Usage.OutputTokens != expectedUsage.OutputTokens ||
		comp.Usage.CacheReadTokens != expectedUsage.CacheReadTokens ||
		comp.Usage.CacheWriteTokens != expectedUsage.CacheWriteTokens {
		t.Errorf("Usage mismatch: got %+v, want %+v", comp.Usage, expectedUsage)
	}
}

// TestCompactionUsageReturnedOnError tests that usage is returned even when
// compaction fails after the model emits EventMessageDone with usage.
func TestCompactionUsageReturnedOnError(t *testing.T) {
	msgs := buildMessages(7, true)

	expectedUsage := agent.Usage{
		InputTokens:      100,
		OutputTokens:     50,
		CacheReadTokens:  10,
		CacheWriteTokens: 5,
	}

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			// Emit usage on EventMessageDone
			onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &expectedUsage})
			// Then emit an error event
			onEvent(agent.Event{Type: agent.EventError, Err: errors.New("model error after usage")})
			return nil
		},
	}

	comp, err := Compact(context.Background(), model, msgs, 1000)
	if err == nil {
		t.Fatal("Compact should have failed with model error")
	}

	// Even on error, comp should be non-nil and carry the usage that was emitted.
	if comp == nil {
		t.Fatal("Compact should return non-nil Compaction even on error, to carry usage")
	}

	if comp.Usage.InputTokens != expectedUsage.InputTokens ||
		comp.Usage.OutputTokens != expectedUsage.OutputTokens ||
		comp.Usage.CacheReadTokens != expectedUsage.CacheReadTokens ||
		comp.Usage.CacheWriteTokens != expectedUsage.CacheWriteTokens {
		t.Errorf("Usage mismatch on error: got %+v, want %+v", comp.Usage, expectedUsage)
	}
}

// TestCompactionUsageReturnedOnEveryErrorPath drives Compact down each of its
// four error returns that follow a model call, with the model reporting usage
// on EventMessageDone first. Every return must carry that usage in a non-nil
// Compaction that was not applied (no Messages). Each subtest is one return,
// with its own failure text, so deleting the usage from one return cannot be
// masked by the others.
func TestCompactionUsageReturnedOnEveryErrorPath(t *testing.T) {
	want := agent.Usage{InputTokens: 111, OutputTokens: 22, CacheReadTokens: 33, CacheWriteTokens: 4}

	for _, tc := range []struct {
		name   string
		stream func(ctx context.Context, cancel context.CancelFunc, onEvent func(agent.Event)) error
	}{
		{
			name: "Stream returns an error after MessageDone",
			stream: func(_ context.Context, _ context.CancelFunc, onEvent func(agent.Event)) error {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
				onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &want})
				return errors.New("transport broke after MessageDone")
			},
		},
		{
			name: "context cancelled after Stream returned nil",
			stream: func(_ context.Context, cancel context.CancelFunc, onEvent func(agent.Event)) error {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
				onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &want})
				cancel()
				return nil
			},
		},
		{
			name: "EventError after MessageDone with Stream nil",
			stream: func(_ context.Context, _ context.CancelFunc, onEvent func(agent.Event)) error {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
				onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &want})
				onEvent(agent.Event{Type: agent.EventError, Err: errors.New("error event after MessageDone")})
				return nil
			},
		},
		{
			name: "whitespace-only summary",
			stream: func(_ context.Context, _ context.CancelFunc, onEvent func(agent.Event)) error {
				onEvent(agent.Event{Type: agent.EventTextDelta, Text: " \n\t "})
				onEvent(agent.Event{Type: agent.EventMessageDone, Usage: &want})
				return nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			model := &FakeModel{onStream: func(ctx context.Context, _ agent.Request, onEvent func(agent.Event)) error {
				return tc.stream(ctx, cancel, onEvent)
			}}

			comp, err := Compact(ctx, model, buildMessages(7, true), 1000)

			if !errors.Is(err, ErrCompactionFailed) {
				t.Fatalf("%s: error = %v, want ErrCompactionFailed", tc.name, err)
			}
			if comp == nil {
				t.Fatalf("%s: Compaction is nil, so the usage the model reported is lost", tc.name)
			}
			if comp.Usage != want {
				t.Errorf("%s: Usage = %+v, want %+v", tc.name, comp.Usage, want)
			}
			if comp.Messages != nil || comp.Summary != "" {
				t.Errorf("%s: failed Compaction carries a result (summary %q, %d messages)", tc.name, comp.Summary, len(comp.Messages))
			}
		})
	}
}

// TestCompactionRoundTrip tests that a compaction can be stored and resumed:
// Compact, record the original messages and the compaction event in a Store,
// then Resume the file and compare with the compacted messages.
func TestCompactionRoundTrip(t *testing.T) {
	roundTrip(t, buildMessages(7, false))
}

// TestCompactionRoundTripWithToolCalls runs the same round trip over a
// conversation with parallel tool calls.
//
// Resume rebuilds user messages through two merges: mergeUserResult
// (resume.go) folds consecutive tool.completed events into one user message,
// and mergeUserText folds a user text into a directly preceding user message.
// Each turn here has an assistant message with two tool calls, so its two
// results are recorded as two events and must come back as the single user
// message in msgs; that exercises mergeUserResult. No user text directly
// follows a user result, so mergeUserText never fires and message indices in
// msgs and in the resumed conversation agree. The exact comparison against
// comp.Messages therefore holds. (buildMessages(n, true) does put a user text
// directly after a result; Resume would merge them and shift the indices that
// RecordCompaction was given, so that shape is not used for round trips.)
func TestCompactionRoundTripWithToolCalls(t *testing.T) {
	msgs := buildParallelToolMessages(7)
	comp := roundTrip(t, msgs)

	// The fixture really contains a two-result user message in the kept tail,
	// so the merge was exercised rather than skipped.
	merged := 0
	for _, m := range comp.Messages {
		if m.Role == agent.RoleUser && len(m.Content) == 2 && m.Content[0].Kind == agent.BlockToolResult {
			merged++
		}
	}
	if merged == 0 {
		t.Fatal("fixture assumption broken: no two-result user message in the compacted messages")
	}
}

// roundTrip compacts msgs, writes msgs plus the compaction event to a Store,
// resumes the file, and requires the resumed conversation to equal
// comp.Messages exactly. It returns the compaction for further checks.
func roundTrip(t *testing.T, msgs []agent.Message) *Compaction {
	t.Helper()

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary of middle messages"})
			return nil
		},
	}

	comp, err := Compact(context.Background(), model, msgs, 1000)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	storePath := filepath.Join(t.TempDir(), "session.jsonl")
	store, err := New(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	for _, m := range msgs {
		store.Record(agent.RecordAppend, m)
	}
	store.RecordCompaction(comp.From, comp.To, comp.Summary)
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	conv, _, err := Resume(storePath)
	if err != nil {
		t.Fatalf("Failed to resume session: %v", err)
	}

	if len(conv.Messages) != len(comp.Messages) {
		t.Fatalf("Resumed conversation has %d messages, want %d", len(conv.Messages), len(comp.Messages))
	}
	for i := range comp.Messages {
		if !messagesEqual(conv.Messages[i], comp.Messages[i]) {
			t.Errorf("Resumed message %d differs from compacted message %d: got %+v, want %+v", i, i, conv.Messages[i], comp.Messages[i])
		}
	}
	if !reflect.DeepEqual(conv.Messages, comp.Messages) {
		t.Errorf("Resumed conversation does not DeepEqual the compacted messages")
	}
	return comp
}

// TestCompactionOriginalTaskAndLastFourPreserved tests that the original task and last 4 turns are byte-identical.
func TestCompactionOriginalTaskAndLastFourPreserved(t *testing.T) {
	msgs := buildMessages(7, true)

	model := &FakeModel{
		onStream: func(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
			onEvent(agent.Event{Type: agent.EventTextDelta, Text: "summary"})
			return nil
		},
	}

	comp, err := Compact(context.Background(), model, msgs, 1000)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	from, to, _ := PlanCompaction(msgs)

	// Messages before the compacted range should be deep-equal to the original.
	for i := 0; i < from; i++ {
		if !messagesEqual(comp.Messages[i], msgs[i]) {
			t.Errorf("Message at original index %d not equal after compaction", i)
		}
	}

	// The summary message should come next (at index from).
	if comp.Messages[from].Role != agent.RoleAssistant {
		t.Errorf("Expected summary at index %d, got role %v", from, comp.Messages[from].Role)
	}
	// Assert the summary content text matches.
	if comp.Summary != "summary" {
		t.Errorf("Expected summary text 'summary', got %q", comp.Summary)
	}
	if len(comp.Messages[from].Content) == 0 || comp.Messages[from].Content[0].Kind != agent.BlockText {
		t.Errorf("Summary message at index %d should have text content", from)
	}
	if len(comp.Messages[from].Content) > 0 && comp.Messages[from].Content[0].Text != "summary" {
		t.Errorf("Summary message text should be 'summary', got %q", comp.Messages[from].Content[0].Text)
	}

	// Messages after the summary should match the original messages after the compacted range.
	// comp.Messages[0:from] = msgs[0:from] (before compaction, kept verbatim)
	// comp.Messages[from] = summary
	// comp.Messages[from+1:] = msgs[to+1:] (after compaction, kept verbatim)

	for i := to + 1; i < len(msgs); i++ {
		compIdx := from + 1 + (i - (to + 1))
		if compIdx >= len(comp.Messages) {
			t.Errorf("Compacted result is shorter than expected: want at least %d messages, got %d", compIdx+1, len(comp.Messages))
			break
		}
		if !messagesEqual(comp.Messages[compIdx], msgs[i]) {
			t.Errorf("Message at original index %d not equal after compaction (compacted index %d)", i, compIdx)
		}
	}
}

// buildParallelToolMessages creates numTurns turns of: user text, assistant
// text plus two tool calls, one user message holding both results, assistant
// text. It matches the shape the live loop records, so Resume reproduces it.
func buildParallelToolMessages(numTurns int) []agent.Message {
	var msgs []agent.Message
	for i := 0; i < numTurns; i++ {
		idA, idB := fmt.Sprintf("tool_%d_a", i), fmt.Sprintf("tool_%d_b", i)
		msgs = append(msgs,
			agent.Message{Role: agent.RoleUser, Content: []agent.Block{{Kind: agent.BlockText, Text: fmt.Sprintf("task %d", i)}}},
			agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{
				{Kind: agent.BlockText, Text: fmt.Sprintf("working %d", i)},
				{Kind: agent.BlockToolCall, ToolCall: &agent.ToolCall{ID: idA, Name: "test_tool", Input: json.RawMessage(`{"param":"a"}`)}},
				{Kind: agent.BlockToolCall, ToolCall: &agent.ToolCall{ID: idB, Name: "test_tool", Input: json.RawMessage(`{"param":"b"}`)}},
			}},
			agent.Message{Role: agent.RoleUser, Content: []agent.Block{
				{Kind: agent.BlockToolResult, Result: &agent.ResultBlock{CallID: idA, Content: fmt.Sprintf("result %d a", i)}},
				{Kind: agent.BlockToolResult, Result: &agent.ResultBlock{CallID: idB, Content: fmt.Sprintf("result %d b", i), IsError: true}},
			}},
			agent.Message{Role: agent.RoleAssistant, Content: []agent.Block{{Kind: agent.BlockText, Text: fmt.Sprintf("done %d", i)}}},
		)
	}
	return msgs
}

// buildMessages creates a test message slice with the specified number of turns.
// Each turn is: user message with text, assistant tool call + text, user tool result.
// If interleave is false, omits the tool result messages (simple user/assistant pairs).
func buildMessages(numTurns int, interleave bool) []agent.Message {
	var msgs []agent.Message
	for i := 0; i < numTurns; i++ {
		msgs = append(msgs, agent.Message{
			Role: agent.RoleUser,
			Content: []agent.Block{
				{Kind: agent.BlockText, Text: fmt.Sprintf("task %d", i)},
			},
		})
		assistantContent := []agent.Block{
			{Kind: agent.BlockText, Text: fmt.Sprintf("response %d", i)},
		}
		if interleave {
			assistantContent = append(assistantContent, agent.Block{
				Kind: agent.BlockToolCall,
				ToolCall: &agent.ToolCall{
					ID:    fmt.Sprintf("tool_%d", i),
					Name:  "test_tool",
					Input: json.RawMessage(`{"param":"value"}`),
				},
			})
		}
		msgs = append(msgs, agent.Message{
			Role:    agent.RoleAssistant,
			Content: assistantContent,
		})
		if interleave {
			msgs = append(msgs, agent.Message{
				Role: agent.RoleUser,
				Content: []agent.Block{
					{Kind: agent.BlockToolResult, Result: &agent.ResultBlock{
						CallID:  fmt.Sprintf("tool_%d", i),
						Content: fmt.Sprintf("result %d", i),
						IsError: false,
					}},
				},
			})
		}
	}
	return msgs
}

// messagesEqual checks if two messages are equal.
func messagesEqual(a, b agent.Message) bool {
	if a.Role != b.Role {
		return false
	}
	if len(a.Content) != len(b.Content) {
		return false
	}
	for i, block := range a.Content {
		if !blocksEqual(block, b.Content[i]) {
			return false
		}
	}
	return true
}

// blocksEqual checks if two blocks are equal.
func blocksEqual(a, b agent.Block) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case agent.BlockText:
		return a.Text == b.Text
	case agent.BlockThinking:
		return thinkingEqual(a.Thinking, b.Thinking)
	case agent.BlockToolCall:
		return toolCallEqual(a.ToolCall, b.ToolCall)
	case agent.BlockToolResult:
		return resultEqual(a.Result, b.Result)
	default:
		return false
	}
}

// thinkingEqual checks if two thinking blocks are equal, guarding against nil.
func thinkingEqual(a, b *agent.Thinking) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Text == b.Text && a.Signature == b.Signature && a.Redacted == b.Redacted && a.Data == b.Data && a.DropOnSummary == b.DropOnSummary
}

// toolCallEqual checks if two tool calls are equal.
func toolCallEqual(a, b *agent.ToolCall) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID && a.Name == b.Name && string(a.Input) == string(b.Input)
}

// resultEqual checks if two tool results are equal.
func resultEqual(a, b *agent.ResultBlock) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.CallID == b.CallID && a.Content == b.Content && a.IsError == b.IsError && a.Truncated == b.Truncated && a.ErrorKind == b.ErrorKind
}

// TestCompactionSystemPromptCommandOutcomes verifies that compactionSystemPrompt
// contains the required phrases for command outcome tracking and limitations.
func TestCompactionSystemPromptCommandOutcomes(t *testing.T) {
	requiredPhrases := []string{
		"PASS or FAIL",
		"Limitations",
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(compactionSystemPrompt, phrase) {
			t.Errorf("compactionSystemPrompt missing required phrase: %q", phrase)
		}
	}
}

// TestCompactionSystemPromptPreservesFinalReport verifies that compactionSystemPrompt
// contains the instruction to preserve final reports verbatim.
// Each pinned phrase covers a distinct fragment of the instruction (the
// trigger condition, the "most recent" selector with "verbatim", and the
// prohibition), so deleting one fragment fails only its own pin.
func TestCompactionSystemPromptPreservesFinalReport(t *testing.T) {
	requiredPhrases := []string{
		"If a final report already appears in the summarized messages",
		"the most recent final report verbatim",
		"Never omit or paraphrase an existing final report",
	}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(compactionSystemPrompt, phrase) {
			t.Errorf("compactionSystemPrompt missing required phrase: %q", phrase)
		}
	}
}
