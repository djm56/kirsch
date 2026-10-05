package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/provider"
)

const fixtures = "../../../testdata/probe/opencode/2026-10-02T1421Z"

func decodeString(t *testing.T, sse string) ([]provider.StreamEvent, error) {
	t.Helper()
	var evs []provider.StreamEvent
	err := Decode(context.Background(), strings.NewReader(sse), func(e provider.StreamEvent) { evs = append(evs, e) })
	return evs, err
}

func decodeFixture(t *testing.T, rel string) []provider.StreamEvent {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtures, rel))
	if err != nil {
		t.Fatal(err)
	}
	evs, err := decodeString(t, string(b))
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return evs
}

func types(evs []provider.StreamEvent) string {
	var s []string
	for _, e := range evs {
		s = append(s, e.Type.String())
	}
	return strings.Join(s, ",")
}

const sseHead = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":0}}}\n\n"

const sseTail = "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":9}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

func block(data string) string { return "event: x\ndata: " + data + "\n\n" }

// TestDecodeAllFixtures: every recorded 200 stream decodes cleanly and ends
// with exactly one MessageDone, as its last event.
func TestDecodeAllFixtures(t *testing.T) {
	statuses, err := filepath.Glob(filepath.Join(fixtures, "minimax-*", "*.status"))
	if err != nil || len(statuses) < 18 {
		t.Fatalf("found %d status files: %v", len(statuses), err)
	}
	n := 0
	for _, st := range statuses {
		b, err := os.ReadFile(st)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(b)) != "200" {
			continue
		}
		rel, err := filepath.Rel(fixtures, strings.TrimSuffix(st, ".status")+".sse")
		if err != nil {
			t.Fatal(err)
		}
		evs := decodeFixture(t, rel)
		done := 0
		for _, e := range evs {
			if e.Type == provider.EventMessageDone {
				done++
			}
			if e.Type == provider.EventError {
				t.Errorf("%s: error event %v", rel, e.Err)
			}
		}
		if done != 1 || evs[len(evs)-1].Type != provider.EventMessageDone {
			t.Errorf("%s: %d MessageDone, last %v", rel, done, evs[len(evs)-1].Type)
		}
		n++
	}
	if n < 18 {
		t.Fatalf("decoded %d fixtures, want at least 18", n)
	}
}

// TestDecodeText: S1 for minimax-m3.
func TestDecodeText(t *testing.T) {
	evs := decodeFixture(t, "minimax-m3/S1-text.sse")
	if got := types(evs); got != "EventTextDelta,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	if evs[0].Text != "pong" {
		t.Fatalf("text = %q", evs[0].Text)
	}
	d := evs[1]
	if d.StopReason != provider.StopEndTurn || d.Usage == nil ||
		*d.Usage != (provider.Usage{InputTokens: 42, OutputTokens: 2, CacheReadTokens: 128}) {
		t.Fatalf("done = %+v usage %+v", d, d.Usage)
	}
}

// TestDecodeToolCall: S2 for minimax-m3.
func TestDecodeToolCall(t *testing.T) {
	evs := decodeFixture(t, "minimax-m3/S2-tool-call.sse")
	if got := types(evs); got != "EventToolCallStart,EventToolCallDelta,EventToolCallEnd,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	start, end := evs[0].ToolCall, evs[2].ToolCall
	if start.ID != "call_01a0fcfdd40d71f2b8525abb" || start.Name != "read_file" || start.Input != nil {
		t.Fatalf("start = %+v", start)
	}
	if end.ID != start.ID || end.Name != "read_file" || string(end.Input) != `{"path":"README.md"}` {
		t.Fatalf("end = %+v input %s", end, end.Input)
	}
	if evs[3].StopReason != provider.StopToolUse {
		t.Fatalf("stop = %v", evs[3].StopReason)
	}
}

// TestDecodeThinkingSignature: S5d for minimax-m3. The signature is kept
// whole, and message_delta usage overrides message_start's field by field.
func TestDecodeThinkingSignature(t *testing.T) {
	evs := decodeFixture(t, "minimax-m3/S5d-enabled.sse")
	if got := types(evs); got != "EventThinkingDelta,EventThinkingDelta,EventThinkingDone,EventTextDelta,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	th := evs[2].Thinking
	want := "The user is asking a simple multiplication: 17 × 23.\n\nLet me calculate: 17 × 23 = 17 × 20 + 17 × 3 = 340 + 51 = 391."
	if th == nil || th.Text != want || th.Redacted ||
		th.Signature != "874b3fff4f39eb4a3027d281cb9a1464432e3385db5ed6655874b43052282249" {
		t.Fatalf("thinking = %+v", th)
	}
	if u := evs[4].Usage; u == nil || *u != (provider.Usage{InputTokens: 31, OutputTokens: 55, CacheReadTokens: 156}) {
		t.Fatalf("usage = %+v", u)
	}
}

// TestDecodePartialJSONAcrossDeltas: tool input arrives in fragments that are
// not JSON until joined. CRLF line endings and comment lines are tolerated.
func TestDecodePartialJSONAcrossDeltas(t *testing.T) {
	sse := sseHead +
		": keep-alive comment\n\n" +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t1","name":"edit","input":{}}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"pa"}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"th\": \"a b\", "}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"n\": 2}"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	sse = strings.ReplaceAll(sse, "\n", "\r\n")
	evs, err := decodeString(t, sse)
	if err != nil {
		t.Fatal(err)
	}
	if got := types(evs); got != "EventToolCallStart,EventToolCallDelta,EventToolCallDelta,EventToolCallDelta,EventToolCallEnd,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	if evs[1].Text != `{"pa` {
		t.Fatalf("first fragment = %q", evs[1].Text)
	}
	in := evs[4].ToolCall.Input
	var v map[string]any
	if err := json.Unmarshal(in, &v); err != nil || v["path"] != "a b" || v["n"] != float64(2) {
		t.Fatalf("input %s: %v", in, err)
	}
	if u := evs[5].Usage; u == nil || *u != (provider.Usage{InputTokens: 5, OutputTokens: 9}) {
		t.Fatalf("usage = %+v", u)
	}
}

// TestDecodeRedactedThinkingAndEmptyToolInput: a redacted block keeps its
// data; a tool call with no input deltas ends with {}.
func TestDecodeRedactedThinkingAndEmptyToolInput(t *testing.T) {
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"redacted_thinking","data":"OPAQUE=="}}`) +
		block(`{"type":"content_block_stop","index":0}`) +
		block(`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"t2","name":"list","input":{}}}`) +
		block(`{"type":"content_block_stop","index":1}`) + sseTail
	evs, err := decodeString(t, sse)
	if err != nil {
		t.Fatal(err)
	}
	if got := types(evs); got != "EventThinkingDone,EventToolCallStart,EventToolCallEnd,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	if th := evs[0].Thinking; th == nil || !th.Redacted || th.Data != "OPAQUE==" || th.Text != "" {
		t.Fatalf("redacted = %+v", th)
	}
	if in := string(evs[2].ToolCall.Input); in != "{}" {
		t.Fatalf("empty input = %q", in)
	}
}

// TestDecodeFailures: every failure emits one EventError carrying the error
// Decode returns, and no MessageDone.
func TestDecodeFailures(t *testing.T) {
	cases := map[string]struct{ sse, want string }{
		"error event": {sseHead + block(`{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`), "overloaded_error: busy"},
		"truncated":   {sseHead + block(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`), "stream ended before message_stop"},
		"bad json":    {sseHead + "data: {not json\n\n", "invalid sse data"},
		"bad tool input": {sseHead +
			block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t3","name":"x","input":{}}}`) +
			block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"a\":"}}`) +
			block(`{"type":"content_block_stop","index":0}`) + sseTail, `tool call "t3" input is not valid JSON`},
		"unknown index": {sseHead + block(`{"type":"content_block_delta","index":7,"delta":{"type":"text_delta","text":"x"}}`) + sseTail, "content block index 7 not started"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			evs, err := decodeString(t, c.sse)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
			if len(evs) == 0 {
				t.Fatal("no events")
			}
			last := evs[len(evs)-1]
			if last.Type != provider.EventError || last.Err != err {
				t.Fatalf("last event = %+v", last)
			}
			for _, e := range evs[:len(evs)-1] {
				if e.Type == provider.EventMessageDone || e.Type == provider.EventError {
					t.Fatalf("unexpected %v before the error", e.Type)
				}
			}
		})
	}
	_, err := decodeString(t, sseHead+block(`{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`))
	var se *StreamError
	if !errors.As(err, &se) || se.Type != "overloaded_error" || se.Message != "busy" {
		t.Fatalf("StreamError = %#v", err)
	}
}

// TestDecodeCancelled: a cancelled context stops decoding with ctx.Err() and
// no further events.
func TestDecodeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b, err := os.ReadFile(filepath.Join(fixtures, "minimax-m3/S5d-enabled.sse"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	err = Decode(ctx, strings.NewReader(string(b)), func(provider.StreamEvent) {
		n++
		cancel()
	})
	if !errors.Is(err, context.Canceled) || n != 1 {
		t.Fatalf("err=%v events=%d", err, n)
	}
}

// TestStopReasonMapping covers every mapped value and the fallback.
func TestStopReasonMapping(t *testing.T) {
	for in, want := range map[string]provider.StopReason{
		"end_turn": provider.StopEndTurn, "tool_use": provider.StopToolUse,
		"max_tokens": provider.StopMaxTokens, "stop_sequence": provider.StopOther,
		"refusal": provider.StopOther, "": provider.StopOther,
	} {
		sse := sseHead + block(`{"type":"message_delta","delta":{"stop_reason":"`+in+`"},"usage":{}}`) +
			block(`{"type":"message_stop"}`)
		evs, err := decodeString(t, sse)
		if err != nil || len(evs) == 0 || evs[len(evs)-1].StopReason != want {
			t.Errorf("%q: err=%v events=%+v", in, err, evs)
		}
	}
}
