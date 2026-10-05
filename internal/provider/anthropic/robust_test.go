package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/provider"
)

// cancelThen serves head, then cancels and fails with err, the way an HTTP
// body behaves when its request context is cancelled.
type cancelThen struct {
	cancel context.CancelFunc
	head   *strings.Reader
	err    error
}

func (r *cancelThen) Read(p []byte) (int, error) {
	if r.head.Len() > 0 {
		return r.head.Read(p)
	}
	r.cancel()
	return 0, r.err
}

// TestDecodeMultiLineData: one event's data split across data: lines is
// joined with \n before parsing; id: and retry: lines are ignored.
func TestDecodeMultiLineData(t *testing.T) {
	sse := "id: 1\nretry: 3000\nevent: message_start\n" +
		"data: {\"type\":\"message_start\",\n" +
		"data:  \"message\":{\"usage\":{\"input_tokens\":7}}}\n\n" +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\ndata: \"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n" +
		block(`{"type":"message_stop"}`)
	evs, err := decodeString(t, sse)
	if err != nil {
		t.Fatal(err)
	}
	if got := types(evs); got != "EventTextDelta,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	if evs[0].Text != "hi" || evs[1].Usage == nil || evs[1].Usage.InputTokens != 7 {
		t.Fatalf("events = %+v usage %+v", evs, evs[1].Usage)
	}
}

// TestDecodeUndispatchedTail: data with no terminating blank line is never
// dispatched, so a stream cut mid-event is truncated, not completed.
func TestDecodeUndispatchedTail(t *testing.T) {
	_, err := decodeString(t, sseHead+"data: {\"type\":\"message_stop\"}\n")
	if err == nil || !strings.Contains(err.Error(), "stream ended before message_stop") {
		t.Fatalf("err = %v", err)
	}
}

// TestDecodeHostileIndex: an out-of-range block index is an error, never a
// panic or an unbounded allocation.
func TestDecodeHostileIndex(t *testing.T) {
	for _, idx := range []string{"-1", "1e9", "1e300", "1.5", "4096", `"0"`} {
		for _, typ := range []string{"content_block_start", "content_block_delta", "content_block_stop"} {
			data := `{"type":"` + typ + `","index":` + idx +
				`,"content_block":{"type":"text","text":""},"delta":{"type":"text_delta","text":"x"}}`
			evs, err := decodeString(t, sseHead+block(data)+sseTail)
			if err == nil || !strings.Contains(err.Error(), "content block index") {
				t.Errorf("%s index %s: err = %v", typ, idx, err)
				continue
			}
			if last := evs[len(evs)-1]; last.Type != provider.EventError || last.Err != err {
				t.Errorf("%s index %s: last event %+v", typ, idx, last)
			}
		}
	}
	_, err := decodeString(t, sseHead+block(`{"type":"content_block_stop"}`)+sseTail)
	if err == nil || !strings.Contains(err.Error(), "content block index missing") {
		t.Errorf("missing index: err = %v", err)
	}
}

// TestDecodeBlockSizeCap: accumulated block content is bounded.
func TestDecodeBlockSizeCap(t *testing.T) {
	old := maxBlockBytes
	maxBlockBytes = 8
	defer func() { maxBlockBytes = old }()
	for name, sse := range map[string]string{
		"tool input": sseHead +
			block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t","name":"x","input":{}}}`) +
			block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"a\":\"12345"}}`) +
			block(`{"type":"content_block_stop","index":0}`) + sseTail,
		"thinking": sseHead +
			block(`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`) +
			block(`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"123456789"}}`) +
			block(`{"type":"content_block_stop","index":0}`) + sseTail,
	} {
		_, err := decodeString(t, sse)
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// TestDecodeBlockClosedAfterStop: a stopped block takes no more deltas and
// cannot stop twice.
func TestDecodeBlockClosedAfterStop(t *testing.T) {
	start := block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t","name":"x","input":{}}}`)
	stop := block(`{"type":"content_block_stop","index":0}`)
	for name, extra := range map[string]string{
		"second stop":      stop,
		"delta after stop": block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}`),
	} {
		_, err := decodeString(t, sseHead+start+stop+extra+sseTail)
		if err == nil || !strings.Contains(err.Error(), "content block index 0") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// TestDecodeCancelledDuringRead: cancellation that surfaces as a read error
// or as an early EOF returns ctx.Err() and emits nothing.
func TestDecodeCancelledDuringRead(t *testing.T) {
	for _, readErr := range []error{io.EOF, errors.New("read on closed body")} {
		ctx, cancel := context.WithCancel(context.Background())
		r := &cancelThen{cancel: cancel, head: strings.NewReader(sseHead), err: readErr}
		var evs []provider.StreamEvent
		err := Decode(ctx, r, func(e provider.StreamEvent) { evs = append(evs, e) })
		cancel()
		if !errors.Is(err, context.Canceled) || len(evs) != 0 {
			t.Errorf("read error %v: err=%v events=%+v", readErr, err, evs)
		}
	}
}

// TestDecodeReadErrorWrapped: a read failure that is not cancellation is
// emitted once and returned wrapped.
func TestDecodeReadErrorWrapped(t *testing.T) {
	boom := errors.New("connection reset")
	r := &cancelThen{cancel: func() {}, head: strings.NewReader(sseHead), err: boom}
	var evs []provider.StreamEvent
	err := Decode(context.Background(), r, func(e provider.StreamEvent) { evs = append(evs, e) })
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "provider stream read: connection reset") {
		t.Fatalf("err = %v", err)
	}
	if len(evs) != 1 || evs[0].Type != provider.EventError || evs[0].Err != err {
		t.Fatalf("events = %+v", evs)
	}
}

// TestDecodeMalformedErrorEvent: an error event without an error object is
// still a failure, never swallowed.
func TestDecodeMalformedErrorEvent(t *testing.T) {
	for _, data := range []string{`{"type":"error"}`, `{"type":"error","error":"busy"}`} {
		_, err := decodeString(t, sseHead+block(data)+block(`{"type":"message_stop"}`))
		var se *StreamError
		if !errors.As(err, &se) || se.Type != "error" || se.Message != "malformed error event" {
			t.Errorf("%s: err = %#v", data, err)
		}
	}
}

// TestDecodeUsageValues: MessageDone always carries a Usage; negative counts
// become zero; a non-integer count is invalid data.
func TestDecodeUsageValues(t *testing.T) {
	stop := block(`{"type":"message_stop"}`)
	evs, err := decodeString(t, block(`{"type":"message_start","message":{}}`)+stop)
	if err != nil || len(evs) != 1 || evs[0].Usage == nil || *evs[0].Usage != (provider.Usage{}) {
		t.Fatalf("no usage: err=%v events=%+v", err, evs)
	}
	evs, err = decodeString(t, block(`{"type":"message_start","message":{"usage":{"input_tokens":-5,"output_tokens":3}}}`)+stop)
	if err != nil || *evs[0].Usage != (provider.Usage{OutputTokens: 3}) {
		t.Fatalf("negative: err=%v events=%+v", err, evs)
	}
	_, err = decodeString(t, block(`{"type":"message_start","message":{"usage":{"input_tokens":1e300}}}`)+stop)
	if err == nil || !strings.Contains(err.Error(), "invalid sse data") {
		t.Fatalf("1e300: err = %v", err)
	}
}

// TestEncodeCacheNeedsBlockFlag: prompt caching places a breakpoint only on
// blocks that ask for one.
func TestEncodeCacheNeedsBlockFlag(t *testing.T) {
	got := encode(t, provider.Request{
		Model: "m", MaxTokens: 1,
		System:   []provider.SystemBlock{{Text: "a"}, {Text: "b", Cache: true}},
		Messages: []provider.Message{userText("x")},
	}, EncodeOptions{PromptCaching: true})
	b, err := json.Marshal(got["system"])
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"text":"a","type":"text"},{"cache_control":{"type":"ephemeral"},"text":"b","type":"text"}]`
	if string(b) != want {
		t.Fatalf("system = %s, want %s", b, want)
	}
}

// TestEncodeRejectsEmpty: requests the API would refuse are refused here.
func TestEncodeRejectsEmpty(t *testing.T) {
	cases := map[string]provider.Request{
		"no messages":   {Model: "m", MaxTokens: 1},
		"empty content": {Model: "m", MaxTokens: 1, Messages: []provider.Message{{Role: provider.RoleUser}}},
		"empty text":    {Model: "m", MaxTokens: 1, Messages: []provider.Message{userText("")}},
		"bad role": {Model: "m", MaxTokens: 1, Messages: []provider.Message{
			{Role: provider.Role("system"), Content: []provider.Block{{Kind: provider.BlockText, Text: "x"}}},
		}},
		"null schema": {
			Model: "m", MaxTokens: 1, Messages: []provider.Message{userText("x")},
			Tools: []provider.ToolDef{{Name: "x", InputSchema: json.RawMessage(`null`)}},
		},
	}
	for name, req := range cases {
		if _, err := EncodeRequest(req, EncodeOptions{}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
