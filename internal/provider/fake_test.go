package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestFakeEmitsEveryEventType proves the Fake can emit every StreamEvent type
// defined by the Provider interface, and that a blocking turn holds open until
// its context is cancelled.
func TestFakeEmitsEveryEventType(t *testing.T) {
	t.Run("all event types", func(t *testing.T) {
		fake := NewFake(
			Turn{
				Thinking:  []Thinking{{Text: "thinking"}},
				Text:      "hello",
				ToolCalls: []ToolCall{{ID: "call_1", Name: "read_file", Input: []byte(`{"path":"/etc/hosts"}`)}},
				Usage:     Usage{InputTokens: 10, OutputTokens: 5},
			},
			Turn{Err: errors.New("scripted failure"), FailAfter: 0},
		)

		var events []StreamEvent
		if err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
			events = append(events, e)
		}); err != nil {
			t.Fatalf("first stream failed: %v", err)
		}
		if err := fake.Stream(context.Background(), Request{Model: "test"}, func(e StreamEvent) {
			events = append(events, e)
		}); err == nil {
			t.Fatal("second stream expected an error")
		}

		got := make(map[EventType]struct{}, len(events))
		for _, e := range events {
			got[e.Type] = struct{}{}
		}
		want := []EventType{
			EventThinkingDelta,
			EventThinkingDone,
			EventTextDelta,
			EventToolCallStart,
			EventToolCallDelta,
			EventToolCallEnd,
			EventMessageDone,
			EventError,
		}
		for _, typ := range want {
			if _, ok := got[typ]; !ok {
				t.Errorf("missing event type %v", typ)
			}
		}
	})

	t.Run("blocks until cancelled", func(t *testing.T) {
		fake := NewFake(Turn{BlockUntilCancel: true})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)

		go func() {
			err := fake.Stream(ctx, Request{Model: "test"}, func(e StreamEvent) {
				t.Errorf("unexpected event %v", e)
			})
			done <- err
		}()

		// Give the goroutine time to reach the blocking select.
		time.Sleep(10 * time.Millisecond)
		select {
		case <-done:
			t.Fatal("stream returned before cancellation")
		default:
		}

		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		case <-time.After(time.Second):
			t.Fatal("stream did not return after cancellation")
		}
	})
}
