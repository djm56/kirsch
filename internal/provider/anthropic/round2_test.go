package anthropic

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/provider"
)

// TestDecodeEventSizeCap: data: lines with no blank line cannot grow the
// pending event without limit.
func TestDecodeEventSizeCap(t *testing.T) {
	old := maxEventBytes
	maxEventBytes = 64
	defer func() { maxEventBytes = old }()
	sse := sseHead + strings.Repeat("data: 0123456789\n", 10) + "\n" + sseTail
	evs, err := decodeString(t, sse)
	if err == nil || !strings.Contains(err.Error(), "sse event exceeds 64 bytes") {
		t.Fatalf("err = %v", err)
	}
	if got := types(evs); got != "EventError" {
		t.Fatalf("events = %s", got)
	}
}

// TestDecodeStreamSizeCap: block content summed across blocks is bounded.
func TestDecodeStreamSizeCap(t *testing.T) {
	old := maxStreamBytes
	maxStreamBytes = 10
	defer func() { maxStreamBytes = old }()
	var sb strings.Builder
	sb.WriteString(sseHead)
	for i := 0; i < 3; i++ {
		n := strconv.Itoa(i)
		sb.WriteString(block(`{"type":"content_block_start","index":` + n + `,"content_block":{"type":"thinking","thinking":""}}`))
		sb.WriteString(block(`{"type":"content_block_delta","index":` + n + `,"delta":{"type":"thinking_delta","thinking":"abcd"}}`))
		sb.WriteString(block(`{"type":"content_block_stop","index":` + n + `}`))
	}
	sb.WriteString(sseTail)
	_, err := decodeString(t, sb.String())
	if err == nil || !strings.Contains(err.Error(), "stream exceeds 10 bytes") {
		t.Fatalf("err = %v", err)
	}
}

// TestDecodeSignatureCap: the signature buffer is bounded like the others.
func TestDecodeSignatureCap(t *testing.T) {
	old := maxBlockBytes
	maxBlockBytes = 8
	defer func() { maxBlockBytes = old }()
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"123456789"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	_, err := decodeString(t, sse)
	if err == nil || !strings.Contains(err.Error(), "content block index 0 exceeds 8 bytes") {
		t.Fatalf("err = %v", err)
	}
}

// TestDecodeIndexOutOfRangeEverywhere: every block event names a bad index
// as out of range, not as "not started"; integer-looking floats, quoted
// numbers and leading zeros are refused.
func TestDecodeIndexOutOfRangeEverywhere(t *testing.T) {
	for _, idx := range []string{"-1", "-0", "1e9", "1e300", "1.5", "1.0", "1e2", "0.0", "4096", "9007199254740993", `"0"`} {
		for _, typ := range []string{"content_block_start", "content_block_delta", "content_block_stop"} {
			data := `{"type":"` + typ + `","index":` + idx +
				`,"content_block":{"type":"text","text":""},"delta":{"type":"text_delta","text":"x"}}`
			evs, err := decodeString(t, sseHead+block(data)+sseTail)
			want := "content block index " + idx + " out of range"
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("%s: err = %v, want %q", typ, err, want)
				continue
			}
			if last := evs[len(evs)-1]; last.Type != provider.EventError || last.Err != err {
				t.Errorf("%s index %s: last event %+v", typ, idx, last)
			}
		}
	}
}

// TestDecodeUndispatchedTailEmitsNoDone: a cut-off message_stop is never
// dispatched, so no MessageDone precedes the truncation error.
func TestDecodeUndispatchedTailEmitsNoDone(t *testing.T) {
	evs, err := decodeString(t, sseHead+"data: {\"type\":\"message_stop\"}\n")
	if err == nil || !strings.Contains(err.Error(), "stream ended before message_stop") {
		t.Fatalf("err = %v", err)
	}
	if got := types(evs); got != "EventError" {
		t.Fatalf("events = %s, want only EventError", got)
	}
}

// TestDecodeThinkingStartSeeded: text and signature carried on the start
// block are kept.
func TestDecodeThinkingStartSeeded(t *testing.T) {
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"ab","signature":"s1"}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"cd"}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"s2"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	evs, err := decodeString(t, sse)
	if err != nil {
		t.Fatal(err)
	}
	if got := types(evs); got != "EventThinkingDelta,EventThinkingDelta,EventThinkingDone,EventMessageDone" {
		t.Fatalf("events = %s", got)
	}
	if th := evs[2].Thinking; th == nil || th.Text != "abcd" || th.Signature != "s1s2" {
		t.Fatalf("thinking = %+v", th)
	}
}

// TestDecodeBlockRestartRefused: a started index cannot be started again,
// whether it is open or stopped.
func TestDecodeBlockRestartRefused(t *testing.T) {
	start := block(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
	stop := block(`{"type":"content_block_stop","index":0}`)
	for name, sse := range map[string]string{
		"while open": sseHead + start + start + sseTail,
		"after stop": sseHead + start + stop + start + sseTail,
	} {
		_, err := decodeString(t, sse)
		if err == nil || !strings.Contains(err.Error(), "content block index 0 already started") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// TestDecodeStoppedBlockMessage: a delta or stop on a stopped block says so.
func TestDecodeStoppedBlockMessage(t *testing.T) {
	start := block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t","name":"x","input":{}}}`)
	stop := block(`{"type":"content_block_stop","index":0}`)
	delta := block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}`)
	for name, extra := range map[string]string{"second stop": stop, "delta after stop": delta} {
		_, err := decodeString(t, sseHead+start+stop+extra+sseTail)
		if err == nil || !strings.Contains(err.Error(), "content block index 0 already stopped") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// TestDecodeCancelInMessageDoneCallback: once MessageDone is delivered the
// stream succeeded, even if the callback then cancels.
func TestDecodeCancelInMessageDoneCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b, err := os.ReadFile(filepath.Join(fixtures, "minimax-m3/S1-text.sse"))
	if err != nil {
		t.Fatal(err)
	}
	var last provider.StreamEvent
	err = Decode(ctx, strings.NewReader(string(b)), func(e provider.StreamEvent) {
		last = e
		if e.Type == provider.EventMessageDone {
			cancel()
		}
	})
	if err != nil || last.Type != provider.EventMessageDone {
		t.Fatalf("err=%v last=%v", err, last.Type)
	}
}

// TestDecodeUsageClampedHigh: counts above MaxInt32 clamp rather than fail.
func TestDecodeUsageClampedHigh(t *testing.T) {
	evs, err := decodeString(t, block(`{"type":"message_start","message":{"usage":{"input_tokens":3000000000}}}`)+
		block(`{"type":"message_stop"}`))
	if err != nil || len(evs) != 1 || evs[0].Usage.InputTokens != math.MaxInt32 {
		t.Fatalf("err=%v events=%+v", err, evs)
	}
}

// TestDecodeToolInputMustBeObject: valid JSON that is not an object is refused.
func TestDecodeToolInputMustBeObject(t *testing.T) {
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t9","name":"x","input":{}}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"[1]"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	_, err := decodeString(t, sse)
	if err == nil || !strings.Contains(err.Error(), `tool call "t9" input is not a JSON object`) {
		t.Fatalf("err = %v", err)
	}
}

// TestEncodeRequiresJSONObjects: schemas and tool inputs must be objects.
func TestEncodeRequiresJSONObjects(t *testing.T) {
	for _, raw := range []string{" null", "null\n", "[1]", `"x"`, "5"} {
		req := provider.Request{
			Model: "m", MaxTokens: 1, Messages: []provider.Message{userText("x")},
			Tools: []provider.ToolDef{{Name: "x", InputSchema: json.RawMessage(raw)}},
		}
		_, err := EncodeRequest(req, EncodeOptions{})
		if err == nil || !strings.Contains(err.Error(), "input_schema must be a JSON object") {
			t.Errorf("schema %q: err = %v", raw, err)
		}
		call := provider.Request{Model: "m", MaxTokens: 1, Messages: []provider.Message{{
			Role: provider.RoleAssistant,
			Content: []provider.Block{{
				Kind: provider.BlockToolCall, ToolCall: &provider.ToolCall{ID: "t", Name: "x", Input: json.RawMessage(raw)},
			}},
		}}}
		_, err = EncodeRequest(call, EncodeOptions{})
		if err == nil || !strings.Contains(err.Error(), "input must be a JSON object") {
			t.Errorf("input %q: err = %v", raw, err)
		}
	}
}
