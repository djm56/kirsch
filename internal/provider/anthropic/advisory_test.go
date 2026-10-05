package anthropic

import (
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/provider"
)

// TestDecodeErrorTextBounded: hostile tokens echoed into errors are clipped.
func TestDecodeErrorTextBounded(t *testing.T) {
	long := strings.Repeat("9", 5000)
	_, err := decodeString(t, sseHead+block(`{"type":"content_block_stop","index":`+long+`}`)+sseTail)
	if err == nil || len(err.Error()) > 200 || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("index: err (%d bytes) = %.300v", len(err.Error()), err)
	}
	id := strings.Repeat("x", 5000)
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"`+id+`","name":"n","input":{}}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"[1]"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	_, err = decodeString(t, sse)
	if err == nil || len(err.Error()) > 200 || !strings.Contains(err.Error(), "not a JSON object") {
		t.Fatalf("tool id: err (%d bytes) = %.300v", len(err.Error()), err)
	}
	msg := strings.Repeat("m", 5000)
	_, err = decodeString(t, sseHead+block(`{"type":"error","error":{"type":"overloaded_error","message":"`+msg+`"}}`))
	if err == nil || len(err.Error()) > 700 || !strings.Contains(err.Error(), "overloaded_error") {
		t.Fatalf("stream error: err (%d bytes) = %.300v", len(err.Error()), err)
	}
}

// TestDecodeUsageOnlyDeltaKeepsStopReason: a later message_delta with no
// stop_reason does not erase an earlier one.
func TestDecodeUsageOnlyDeltaKeepsStopReason(t *testing.T) {
	sse := sseHead +
		block(`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":3}}`) +
		block(`{"type":"message_delta","delta":{},"usage":{"output_tokens":4}}`) +
		block(`{"type":"message_stop"}`)
	evs, err := decodeString(t, sse)
	if err != nil || len(evs) == 0 {
		t.Fatalf("err=%v", err)
	}
	last := evs[len(evs)-1]
	if last.Type != provider.EventMessageDone || last.StopReason != provider.StopToolUse || last.Usage.OutputTokens != 4 {
		t.Fatalf("last = %+v usage %+v", last, last.Usage)
	}
}
