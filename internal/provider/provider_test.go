package provider

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// T1: text-only turn
func TestFakeTextOnly(t *testing.T) {
	fake := NewFake(Turn{
		Text: "Hello world",
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
		StopReason: StopEndTurn,
	})

	events := collectEvents(t, fake, Request{Model: "test"})

	// Verify event order and types.
	textDeltas := 0
	var lastStopReason StopReason

	for _, e := range events {
		switch e.Type {
		case EventTextDelta:
			textDeltas++
			// Verify no empty deltas.
			if e.Text == "" {
				t.Error("text delta is empty")
			}
		case EventMessageDone:
			lastStopReason = e.StopReason
			if e.Usage == nil {
				t.Fatal("MessageDone missing Usage")
			}
			if e.Usage.InputTokens != 100 || e.Usage.OutputTokens != 50 {
				t.Fatalf("Usage mismatch: got %+v", e.Usage)
			}
		case EventError:
			t.Fatal("unexpected error event")
		case EventThinkingDelta, EventThinkingDone, EventToolCallStart, EventToolCallDelta, EventToolCallEnd:
			t.Fatalf("unexpected event type in text-only turn: %v", e.Type)
		}
	}

	if textDeltas < 2 {
		t.Errorf("expected at least 2 text deltas, got %d", textDeltas)
	}
	if lastStopReason != StopEndTurn {
		t.Errorf("expected StopEndTurn, got %v", lastStopReason)
	}

	// Verify text concatenation.
	var text strings.Builder
	for _, e := range events {
		if e.Type == EventTextDelta {
			text.WriteString(e.Text)
		}
	}
	if text.String() != "Hello world" {
		t.Errorf("concatenated text mismatch: got %q", text.String())
	}
}

// T2: single tool call
func TestFakeSingleToolCall(t *testing.T) {
	toolCall := ToolCall{
		ID:    "call_001",
		Name:  "read_file",
		Input: json.RawMessage(`{"path":"/etc/hosts"}`),
	}

	fake := NewFake(Turn{
		ToolCalls: []ToolCall{toolCall},
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
		StopReason: StopToolUse,
	})

	events := collectEvents(t, fake, Request{Model: "test"})

	// Verify event sequence: start, deltas, end.
	var starts, ends int
	var deltaCount int

	for _, e := range events {
		switch e.Type {
		case EventToolCallStart:
			starts++
			if e.ToolCall == nil || e.ToolCall.ID != "call_001" || e.ToolCall.Name != "read_file" {
				t.Fatalf("ToolCallStart mismatch: %+v", e.ToolCall)
			}
		case EventToolCallDelta:
			deltaCount++
		case EventToolCallEnd:
			ends++
			if e.ToolCall == nil {
				t.Fatal("ToolCallEnd missing ToolCall")
			}
		case EventTextDelta, EventThinkingDelta, EventThinkingDone, EventMessageDone, EventError:
			// Expected events, no action needed
		}
	}

	if starts != 1 || ends != 1 {
		t.Errorf("expected 1 start and 1 end, got %d/%d", starts, ends)
	}

	// Verify at least 2 deltas.
	if deltaCount < 2 {
		t.Errorf("expected at least 2 tool call deltas, got %d", deltaCount)
	}

	// Verify input concatenation and byte equality with ToolCallEnd.
	var input strings.Builder
	for _, e := range events {
		if e.Type == EventToolCallDelta {
			input.WriteString(e.Text)
		}
	}
	concatenatedInput := input.String()
	expectedInput := `{"path":"/etc/hosts"}`
	if concatenatedInput != expectedInput {
		t.Errorf("concatenated input mismatch: expected %q, got %q", expectedInput, concatenatedInput)
	}

	// Find ToolCallEnd and verify Input matches concatenated deltas.
	for _, e := range events {
		if e.Type == EventToolCallEnd {
			if string(e.ToolCall.Input) != concatenatedInput {
				t.Errorf("ToolCallEnd Input mismatch: expected %q, got %q", concatenatedInput, string(e.ToolCall.Input))
			}
			break
		}
	}

	// Verify stop reason is asserted unconditionally.
	if len(events) == 0 || events[len(events)-1].Type != EventMessageDone {
		t.Error("expected EventMessageDone as last event")
	} else if events[len(events)-1].StopReason != StopToolUse {
		t.Errorf("expected StopToolUse, got %v", events[len(events)-1].StopReason)
	}
}

// T3: two tool calls
func TestFakeTwoToolCalls(t *testing.T) {
	calls := []ToolCall{
		{ID: "call_001", Name: "search_code", Input: json.RawMessage(`{"query":"TODO"}`)},
		{ID: "call_002", Name: "read_file", Input: json.RawMessage(`{"path":"main.go"}`)},
	}

	fake := NewFake(Turn{
		ToolCalls: calls,
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	})

	events := collectEvents(t, fake, Request{Model: "test"})

	// Verify every event for call 1 comes before call 2's start.
	var call1EndIndex, call2StartIndex int
	foundCall1End := false
	foundCall2Start := false

	for i, e := range events {
		if e.Type == EventToolCallEnd && e.ToolCall != nil && e.ToolCall.ID == "call_001" {
			call1EndIndex = i
			foundCall1End = true
		}
		if e.Type == EventToolCallStart && e.ToolCall != nil && e.ToolCall.ID == "call_002" {
			call2StartIndex = i
			foundCall2Start = true
		}
	}

	if foundCall1End && foundCall2Start && call1EndIndex >= call2StartIndex {
		t.Error("call 1's EventToolCallEnd should come before call 2's EventToolCallStart")
	}

	// Verify calls appear in order with their IDs preserved.
	callEnds := []*ToolCall{}
	for _, e := range events {
		if e.Type == EventToolCallEnd {
			callEnds = append(callEnds, e.ToolCall)
		}
	}

	if len(callEnds) != 2 {
		t.Errorf("expected 2 tool call ends, got %d", len(callEnds))
	}
	if len(callEnds) >= 2 {
		if callEnds[0].ID != "call_001" || callEnds[1].ID != "call_002" {
			t.Errorf("tool call IDs out of order: %v, %v", callEnds[0].ID, callEnds[1].ID)
		}
	}

	// Verify stop reason is asserted unconditionally.
	if len(events) == 0 || events[len(events)-1].Type != EventMessageDone {
		t.Fatalf("expected EventMessageDone as last event")
	}
	stopReason := events[len(events)-1].StopReason
	if stopReason != StopToolUse {
		t.Errorf("expected StopToolUse, got %v", stopReason)
	}

	// New subtest: with no StopReason set, verify default.
	t.Run("default_stop_reason_with_calls", func(t *testing.T) {
		fake := NewFake(Turn{
			ToolCalls: []ToolCall{
				{ID: "call_001", Name: "test", Input: json.RawMessage(`{}`)},
			},
		})
		events := collectEvents(t, fake, Request{Model: "test"})
		if len(events) == 0 || events[len(events)-1].Type != EventMessageDone {
			t.Fatalf("expected EventMessageDone as last event")
		}
		stopReason := events[len(events)-1].StopReason
		if stopReason != StopToolUse {
			t.Errorf("expected default StopToolUse with calls, got %v", stopReason)
		}
	})

	t.Run("default_stop_reason_no_calls", func(t *testing.T) {
		fake := NewFake(Turn{
			Text: "hello",
		})
		events := collectEvents(t, fake, Request{Model: "test"})
		if len(events) == 0 || events[len(events)-1].Type != EventMessageDone {
			t.Fatalf("expected EventMessageDone as last event")
		}
		stopReason := events[len(events)-1].StopReason
		if stopReason != StopEndTurn {
			t.Errorf("expected default StopEndTurn without calls, got %v", stopReason)
		}
	})
}

// T4: thinking blocks
func TestFakeThinkingBlocks(t *testing.T) {
	thinking := Thinking{
		Text:      "Let me think about this",
		Signature: "sig_12345",
		Redacted:  false,
		Data:      "",
	}

	redactedThinking := Thinking{
		Text:      "",
		Signature: "sig_67890",
		Redacted:  true,
		Data:      "opaque_data",
	}

	fake := NewFake(Turn{
		Thinking: []Thinking{thinking, redactedThinking},
		Text:     "Result",
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	})

	events := collectEvents(t, fake, Request{Model: "test"})

	// Verify thinking events come before text.
	textDeltaIndex := -1
	thinkingDoneIndices := []int{}

	for i, e := range events {
		if e.Type == EventTextDelta && textDeltaIndex == -1 {
			textDeltaIndex = i
		}
		if e.Type == EventThinkingDone {
			thinkingDoneIndices = append(thinkingDoneIndices, i)
		}
	}

	if textDeltaIndex != -1 && len(thinkingDoneIndices) > 0 {
		for _, idx := range thinkingDoneIndices {
			if idx > textDeltaIndex {
				t.Error("thinking events should come before text events")
				break
			}
		}
	}

	// Verify normal thinking: concatenate deltas and verify against script.
	var normalDeltas []string
	for i, e := range events {
		if e.Type == EventThinkingDone && e.Thinking != nil && e.Thinking.Signature == "sig_12345" {
			// Look backward for all deltas up to this done.
			for j := i - 1; j >= 0; j-- {
				if events[j].Type == EventThinkingDelta {
					normalDeltas = append([]string{events[j].Text}, normalDeltas...)
				} else if events[j].Type == EventThinkingDone {
					break
				}
			}
			break
		}
	}

	// Verify concatenated deltas equal scripted Text.
	var concatenated strings.Builder
	for _, d := range normalDeltas {
		if d == "" {
			t.Error("thinking delta is empty")
		}
		concatenated.WriteString(d)
	}
	if concatenated.String() != "Let me think about this" {
		t.Errorf("normal thinking deltas mismatch: expected %q, got %q", "Let me think about this", concatenated.String())
	}

	// Verify ThinkingDone Text and Signature.
	for _, e := range events {
		if e.Type == EventThinkingDone && e.Thinking != nil && e.Thinking.Signature == "sig_12345" {
			if e.Thinking.Text != "Let me think about this" {
				t.Errorf("ThinkingDone Text mismatch: expected %q, got %q", "Let me think about this", e.Thinking.Text)
			}
			if e.Thinking.Signature != "sig_12345" {
				t.Errorf("ThinkingDone Signature mismatch: expected %q, got %q", "sig_12345", e.Thinking.Signature)
			}
		}
	}

	// Verify redacted thinking block produces zero ThinkingDelta events.
	redactedDeltaCount := 0
	firstThinkingDone := false
	for _, e := range events {
		if e.Type == EventThinkingDone && e.Thinking != nil && e.Thinking.Signature == "sig_12345" {
			firstThinkingDone = true
		}
		if firstThinkingDone && e.Type == EventThinkingDelta {
			redactedDeltaCount++
		}
		if e.Type == EventThinkingDone && e.Thinking != nil && e.Thinking.Signature == "sig_67890" {
			// Verify redacted done block has Redacted==true and intact Data.
			if e.Thinking.Redacted != true {
				t.Error("redacted thinking Redacted should be true")
			}
			if e.Thinking.Data != "opaque_data" {
				t.Errorf("redacted thinking Data mismatch: expected %q, got %q", "opaque_data", e.Thinking.Data)
			}
			break
		}
	}

	if redactedDeltaCount > 0 {
		t.Errorf("redacted thinking block should emit zero deltas, got %d", redactedDeltaCount)
	}
}

// T5: mid-stream error
func TestFakeMidStreamError(t *testing.T) {
	expectedErr := errors.New("stream error")
	fake := NewFake(Turn{
		Text:      "partial response",
		FailAfter: 2,
		Err:       expectedErr,
	})

	eventCount := 0
	var lastErr error

	err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
		eventCount++
		if e.Type == EventError {
			lastErr = e.Err
		}
	})

	if eventCount != 3 { // 2 events + error event
		t.Errorf("expected 3 events (2 + error), got %d", eventCount)
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if lastErr != expectedErr {
		t.Errorf("error event mismatch: expected %v, got %v", expectedErr, lastErr)
	}
}

// W-FA: Error and FailAfter semantics
func TestFailAfterAndErrSemantics(t *testing.T) {
	t.Run("Err_with_FailAfter_zero", func(t *testing.T) {
		// Err != nil && FailAfter == 0: emit EventError immediately, then return Err.
		expectedErr := errors.New("immediate error")
		fake := NewFake(Turn{
			Text:      "should not emit",
			FailAfter: 0,
			Err:       expectedErr,
		})

		var events []StreamEvent
		err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
			events = append(events, e)
		})

		// Should emit only one event: EventError
		if len(events) != 1 {
			t.Errorf("expected 1 event, got %d", len(events))
		}
		if len(events) > 0 && events[0].Type != EventError {
			t.Errorf("expected EventError, got %v", events[0].Type)
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("Err_with_FailAfter_positive", func(t *testing.T) {
		// Err != nil && FailAfter > 0: emit FailAfter events, then EventError, then return Err.
		expectedErr := errors.New("delayed error")
		fake := NewFake(Turn{
			Text:      "abcdefgh",
			FailAfter: 2,
			Err:       expectedErr,
		})

		var events []StreamEvent
		var eventTypes []EventType
		err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
			events = append(events, e)
			eventTypes = append(eventTypes, e.Type)
		})

		// Should emit 2 text deltas + 1 error event = 3 total.
		// Verify no MessageDone when Err is set.
		errorFound := false
		for _, e := range events {
			if e.Type == EventError {
				errorFound = true
			}
			if e.Type == EventMessageDone {
				t.Error("should not emit MessageDone when Err is set")
			}
		}
		if !errorFound {
			t.Error("expected EventError")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		want := []EventType{EventTextDelta, EventTextDelta, EventError}
		if !slices.Equal(eventTypes, want) {
			t.Errorf("event types = %v, want %v", eventTypes, want)
		}
	})

	t.Run("Err_nil_with_FailAfter_positive", func(t *testing.T) {
		// Err == nil && FailAfter > 0: this is a script error.
		fake := NewFake(Turn{
			Text:      "hello",
			FailAfter: 2,
			Err:       nil,
		})

		eventCount := 0
		err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
			eventCount++
		})

		if eventCount != 0 {
			t.Errorf("expected 0 events, got %d", eventCount)
		}
		if err == nil {
			t.Error("expected script error when Err is nil but FailAfter > 0")
		}
		if err != nil && !strings.Contains(err.Error(), "script error") {
			t.Errorf("expected script error message, got %v", err)
		}
	})
}

func TestFailAfterBeyondTurnLength(t *testing.T) {
	scripted := errors.New("boom")
	fake := NewFake(Turn{Text: "hi", FailAfter: 10, Err: scripted})
	var types []EventType
	err := fake.Stream(context.Background(), Request{}, func(e StreamEvent) { types = append(types, e.Type) })
	if !errors.Is(err, scripted) {
		t.Fatalf("err = %v, want %v", err, scripted)
	}
	want := []EventType{EventTextDelta, EventTextDelta, EventError}
	if !slices.Equal(types, want) {
		t.Fatalf("events = %v, want %v", types, want)
	}
}

func TestRequestsReturnsDeepCopies(t *testing.T) {
	fake := NewFake(Turn{Text: "ok"})
	req := Request{
		Model: "m",
		Messages: []Message{
			{Role: RoleUser, Content: []Block{{Kind: BlockText, Text: "hello"}}},
			{Role: RoleAssistant, Content: []Block{
				{Kind: BlockThinking, Thinking: &Thinking{Text: "t", Signature: "sig"}},
				{Kind: BlockToolCall, ToolCall: &ToolCall{ID: "c1", Name: "read_file", Input: json.RawMessage(`{"path":"a"}`)}},
			}},
		},
		Tools: []ToolDef{{Name: "read_file", InputSchema: json.RawMessage(`{"type":"object"}`)}},
	}
	if err := fake.Stream(context.Background(), req, func(StreamEvent) {}); err != nil {
		t.Fatal(err)
	}
	got := fake.Requests()
	got[0].Messages[0].Content[0].Text = "MUTATED"
	got[0].Messages[1].Content[0].Thinking.Signature = "MUTATED"
	got[0].Messages[1].Content[1].ToolCall.Input[0] = 'X'
	got[0].Tools[0].InputSchema[0] = 'X'

	again := fake.Requests()[0]
	if s := again.Messages[0].Content[0].Text; s != "hello" {
		t.Errorf("Text = %q, want hello", s)
	}
	if s := again.Messages[1].Content[0].Thinking.Signature; s != "sig" {
		t.Errorf("Signature = %q, want sig", s)
	}
	if s := string(again.Messages[1].Content[1].ToolCall.Input); s != `{"path":"a"}` {
		t.Errorf("Input = %s", s)
	}
	if s := string(again.Tools[0].InputSchema); s != `{"type":"object"}` {
		t.Errorf("InputSchema = %s", s)
	}
}

func TestToolCallEndDoesNotAliasScript(t *testing.T) {
	ctx := context.Background()
	turn := Turn{ToolCalls: []ToolCall{{ID: "c1", Name: "n", Input: json.RawMessage(`{"a":1}`)}}}
	f1 := NewFake(turn)
	_ = f1.Stream(ctx, Request{}, func(e StreamEvent) {
		if e.Type == EventToolCallEnd {
			e.ToolCall.Input[0] = 'X'
		}
	})
	if s := string(turn.ToolCalls[0].Input); s != `{"a":1}` {
		t.Fatalf("script mutated: %s", s)
	}
	f2 := NewFake(turn)
	var got string
	_ = f2.Stream(ctx, Request{}, func(e StreamEvent) {
		if e.Type == EventToolCallEnd {
			got = string(e.ToolCall.Input)
		}
	})
	if got != `{"a":1}` {
		t.Fatalf("second fake saw %s", got)
	}
}

func TestThinkingEchoRoundTrip(t *testing.T) {
	ctx := context.Background()
	normal := Thinking{Text: "reasoning", Signature: "sig-abc"}
	redacted := Thinking{Redacted: true, Data: "opaque-xyz"}
	fake := NewFake(Turn{Thinking: []Thinking{normal, redacted}, Text: "answer"}, Turn{Text: "done"})
	var blocks []Block
	if err := fake.Stream(ctx, Request{}, func(e StreamEvent) {
		if e.Type == EventThinkingDone {
			blocks = append(blocks, Block{Kind: BlockThinking, Thinking: e.Thinking})
		}
	}); err != nil {
		t.Fatal(err)
	}
	req2 := Request{Messages: []Message{{Role: RoleAssistant, Content: blocks}}}
	if err := fake.Stream(ctx, req2, func(StreamEvent) {}); err != nil {
		t.Fatal(err)
	}
	rec := fake.Requests()[1].Messages[0].Content
	if len(rec) != 2 {
		t.Fatalf("got %d echoed blocks, want 2", len(rec))
	}
	if *rec[0].Thinking != normal {
		t.Errorf("normal block = %+v, want %+v", *rec[0].Thinking, normal)
	}
	if *rec[1].Thinking != redacted {
		t.Errorf("redacted block = %+v, want %+v", *rec[1].Thinking, redacted)
	}
}

// T6: block until cancel
func TestFakeBlockUntilCancel(t *testing.T) {
	fake := NewFake(Turn{
		BlockUntilCancel: true,
	})

	ctx, cancel := context.WithCancel(context.Background())

	eventReceived := make(chan bool, 1)
	done := make(chan error, 1)

	go func() {
		err := fake.Stream(ctx, Request{Model: "test"}, func(e StreamEvent) {
			eventReceived <- true
		})
		done <- err
	}()

	// Wait a bit to verify no immediate return or events.
	time.Sleep(50 * time.Millisecond)
	select {
	case <-eventReceived:
		t.Fatal("events should not be emitted in BlockUntilCancel mode")
	case <-done:
		t.Fatal("should not return immediately")
	default:
	}

	// Cancel the context.
	cancel()

	// Should return within 1s with context.Canceled.
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Stream did not return within 1s after cancellation")
	}
}

// T7: out of turns
func TestFakeOutOfTurns(t *testing.T) {
	fake := NewFake(Turn{Text: "response 1"})

	// First call should succeed.
	err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {})
	if err != nil {
		t.Fatalf("first Stream call failed: %v", err)
	}

	// Second call should fail with exhaustion error.
	eventCount := 0
	err = fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
		eventCount++
	})

	if err == nil {
		t.Fatal("expected error for exhausted turns")
	}
	if !strings.Contains(err.Error(), "exhausted") {
		t.Errorf("expected 'exhausted' in error message, got: %v", err)
	}
	if eventCount > 0 {
		t.Error("should emit no events when out of turns")
	}
}

// T8: request recording and deep copy
func TestFakeRequestRecording(t *testing.T) {
	// Build a request with thinking, text, tool call, and tools.
	toolInput := json.RawMessage(`{"key":"value"}`)
	toolSchema := json.RawMessage(`{"type":"object"}`)

	fake := NewFake(
		Turn{Text: "response 1"},
		Turn{
			Thinking: []Thinking{{Text: "think", Signature: "sig_1"}},
			Text:     "response 2",
			ToolCalls: []ToolCall{
				{ID: "call_001", Name: "tool1", Input: toolInput},
			},
		},
	)

	// First request
	req1 := Request{
		Model: "model-1",
		Messages: []Message{
			{
				Role: RoleUser,
				Content: []Block{
					{Kind: BlockText, Text: "hello"},
				},
			},
		},
	}

	err := fake.Stream(context.Background(), req1, func(e StreamEvent) {})
	if err != nil {
		t.Fatalf("first Stream call failed: %v", err)
	}

	// Mutate the original request.
	req1.Messages[0].Content[0].Text = "MUTATED"

	// Second request with thinking block echo, tool call, and tools.
	req2 := Request{
		Model: "model-1",
		Messages: []Message{
			{
				Role: RoleUser,
				Content: []Block{
					{Kind: BlockText, Text: "hello"},
				},
			},
			{
				Role: RoleAssistant,
				Content: []Block{
					{Kind: BlockThinking, Thinking: &Thinking{Text: "think", Signature: "sig_1"}},
					{Kind: BlockToolCall, ToolCall: &ToolCall{ID: "call_001", Name: "tool1", Input: toolInput}},
				},
			},
		},
		Tools: []ToolDef{
			{Name: "tool1", Description: "test tool", InputSchema: toolSchema},
		},
	}

	err = fake.Stream(context.Background(), req2, func(e StreamEvent) {})
	if err != nil {
		t.Fatalf("second Stream call failed: %v", err)
	}

	// Mutate nested fields in the original request.
	if req2.Messages[1].Content[0].Thinking != nil {
		req2.Messages[1].Content[0].Thinking.Signature = "MUTATED_SIG"
	}
	if req2.Messages[1].Content[1].ToolCall != nil && len(req2.Messages[1].Content[1].ToolCall.Input) > 0 {
		req2.Messages[1].Content[1].ToolCall.Input[0] = 'X'
	}
	if len(req2.Tools) > 0 && len(req2.Tools[0].InputSchema) > 0 {
		req2.Tools[0].InputSchema[0] = 'Y'
	}

	// Verify recorded copies are unchanged via Requests().
	recorded := fake.Requests()

	if len(recorded) != 2 {
		t.Fatalf("expected 2 recorded requests, got %d", len(recorded))
	}

	// First recorded request should still have original text.
	if recorded[0].Messages[0].Content[0].Text != "hello" {
		t.Errorf("recorded request 1 was mutated: %q", recorded[0].Messages[0].Content[0].Text)
	}

	// Second recorded request: verify all nested fields are unchanged.
	if recorded[1].Messages[1].Content[0].Thinking.Signature != "sig_1" {
		t.Errorf("recorded request 2's thinking signature was mutated: %q", recorded[1].Messages[1].Content[0].Thinking.Signature)
	}

	if string(recorded[1].Messages[1].Content[1].ToolCall.Input) != `{"key":"value"}` {
		t.Errorf("recorded request 2's tool input was mutated: %s", recorded[1].Messages[1].Content[1].ToolCall.Input)
	}

	if string(recorded[1].Tools[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("recorded request 2's tool schema was mutated: %s", recorded[1].Tools[0].InputSchema)
	}

	// Call Requests() again and verify same results (testing slice/snapshot isolation).
	recorded2 := fake.Requests()
	if len(recorded2) != 2 {
		t.Errorf("second Requests() call returned wrong count")
	}
	if recorded2[1].Messages[1].Content[0].Thinking.Signature != "sig_1" {
		t.Errorf("second Requests() call: thinking signature was mutated")
	}
}

// T9: multibyte text (UTF-8 rune boundaries)
func TestFakeMultibyteText(t *testing.T) {
	text := "héllo wörld ✓"
	fake := NewFake(Turn{
		Text: text,
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	})

	events := collectEvents(t, fake, Request{Model: "test"})

	// Verify all fragments are valid UTF-8.
	var concatenated strings.Builder
	for _, e := range events {
		if e.Type == EventTextDelta {
			if !utf8.ValidString(e.Text) {
				t.Errorf("invalid UTF-8 in delta: %q", e.Text)
			}
			concatenated.WriteString(e.Text)
		}
	}

	if concatenated.String() != text {
		t.Errorf("concatenated text mismatch: expected %q, got %q", text, concatenated.String())
	}
}

// T10: cancellation mid-turn
func TestFakeCancellationMidTurn(t *testing.T) {
	// Script a turn with at least 10 events: a long text plus a tool call.
	fake := NewFake(Turn{
		Text: "abcdefghijklmnopqr",
		ToolCalls: []ToolCall{
			{
				ID:    "call_001",
				Name:  "test_tool",
				Input: json.RawMessage(`{"key":"value"}`),
			},
		},
		Usage: Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	})

	ctx, cancel := context.WithCancel(context.Background())

	var eventCount int

	err := fake.Stream(ctx, Request{Model: "test"}, func(e StreamEvent) {
		eventCount++
		// Cancel on the first event.
		if eventCount == 1 {
			cancel()
		}
	})

	// After Stream returns, assert exactly 1 event was emitted.
	if eventCount != 1 {
		t.Errorf("expected 1 event after cancel, got %d", eventCount)
	}

	// Assert the error is context.Canceled.
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// T11: model table
func TestModelTable(t *testing.T) {
	cases := []struct {
		id             string
		expectKnown    bool
		expectCtx      int
		expectMaxOut   int
		expectPricing  Pricing
		expectInputPer float64
	}{
		{"claude-sonnet-5-5", true, 1000000, 128000, PricingPerToken, 2.0},
		{"claude-opus-5-5", true, 1000000, 128000, PricingPerToken, 4.0},
		{"claude-fable-5-1", true, 1000000, 128000, PricingPerToken, 10.0},
		{"claude-haiku-4-5-20251001", true, 200000, 64000, PricingPerToken, 1.0},
		{"minimax-m3", true, 128000, 4096, PricingFlat, 0.0},
		{"unknown-model", false, 128000, 4096, PricingUnknown, 0.0},
	}

	for _, c := range cases {
		m := LookupModel(c.id)

		if m.Known != c.expectKnown {
			t.Errorf("%s: Known mismatch, expected %v, got %v", c.id, c.expectKnown, m.Known)
		}
		if m.ContextWindow != c.expectCtx {
			t.Errorf("%s: ContextWindow mismatch, expected %d, got %d", c.id, c.expectCtx, m.ContextWindow)
		}
		if m.MaxOutput != c.expectMaxOut {
			t.Errorf("%s: MaxOutput mismatch, expected %d, got %d", c.id, c.expectMaxOut, m.MaxOutput)
		}
		if m.Pricing != c.expectPricing {
			t.Errorf("%s: Pricing mismatch, expected %v, got %v", c.id, c.expectPricing, m.Pricing)
		}
		if m.Pricing == PricingPerToken && m.InputPerMTok != c.expectInputPer {
			t.Errorf("%s: InputPerMTok mismatch, expected %f, got %f", c.id, c.expectInputPer, m.InputPerMTok)
		}
	}
}

// T12: concurrency
func TestFakeConcurrency(t *testing.T) {
	fake := NewFake(
		Turn{Text: "t1"},
		Turn{Text: "t2"},
		Turn{Text: "t3"},
		Turn{Text: "t4"},
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	collectedTexts := make([]string, 0)

	// Launch 2 goroutines, each consuming 2 turns.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 2; j++ {
				var turnText strings.Builder
				err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
					if e.Type == EventTextDelta {
						turnText.WriteString(e.Text)
					}
					// Call Requests() and Remaining() in callback.
					// This would deadlock if the lock were held during the callback.
					_ = fake.Requests()
					_ = fake.Remaining()
				})
				mu.Lock()
				errs = append(errs, err)
				collectedTexts = append(collectedTexts, turnText.String())
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify all calls succeeded.
	for _, err := range errs {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	// Verify we got 4 requests recorded.
	recorded := fake.Requests()
	if len(recorded) != 4 {
		t.Errorf("expected 4 recorded requests, got %d", len(recorded))
	}

	// Verify each request has the right model.
	for i, req := range recorded {
		if req.Model != "test" {
			t.Errorf("request %d: expected model %q, got %q", i, "test", req.Model)
		}
	}

	// Verify Remaining() == 0.
	if fake.Remaining() != 0 {
		t.Errorf("expected 0 remaining turns, got %d", fake.Remaining())
	}

	// Verify the multiset of collected texts is exactly {t1, t2, t3, t4}, each once.
	expectedMultiset := map[string]int{"t1": 1, "t2": 1, "t3": 1, "t4": 1}
	actualMultiset := make(map[string]int)
	for _, text := range collectedTexts {
		actualMultiset[text]++
	}
	if len(actualMultiset) != len(expectedMultiset) {
		t.Errorf("multiset mismatch: expected %v, got %v", expectedMultiset, actualMultiset)
	}
	for k := range expectedMultiset {
		if actualMultiset[k] != expectedMultiset[k] {
			t.Errorf("text %q: expected count %d, got %d", k, expectedMultiset[k], actualMultiset[k])
		}
	}
}

// Helper: collectEvents runs Stream and collects all events.
func collectEvents(t *testing.T, fake *Fake, req Request) []StreamEvent {
	var events []StreamEvent
	err := fake.Stream(context.Background(), req, func(e StreamEvent) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	return events
}
