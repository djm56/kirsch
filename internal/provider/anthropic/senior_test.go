package anthropic

import (
	"errors"
	"strings"
	"testing"
)

// TestDecodeStartEventsCounted: content carried on start events counts
// toward both the per-block and the per-stream cap.
func TestDecodeStartEventsCounted(t *testing.T) {
	stop := block(`{"type":"content_block_stop","index":0}`)
	cases := map[string]string{
		"thinking seed":  `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"123456789"}}`,
		"signature seed": `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":"123456789"}}`,
		"redacted data":  `{"type":"content_block_start","index":0,"content_block":{"type":"redacted_thinking","data":"123456789"}}`,
	}
	for name, start := range cases {
		func() {
			old := maxBlockBytes
			maxBlockBytes = 8
			defer func() { maxBlockBytes = old }()
			_, err := decodeString(t, sseHead+block(start)+stop+sseTail)
			if err == nil || !strings.Contains(err.Error(), "content block index 0 exceeds 8 bytes") {
				t.Errorf("%s, block cap: err = %v", name, err)
			}
		}()
		func() {
			old := maxStreamBytes
			maxStreamBytes = 8
			defer func() { maxStreamBytes = old }()
			_, err := decodeString(t, sseHead+block(start)+stop+sseTail)
			if err == nil || !strings.Contains(err.Error(), "stream exceeds 8 bytes") {
				t.Errorf("%s, stream cap: err = %v", name, err)
			}
		}()
	}
}

// TestDecodeToolStartCounted: a tool call's id and name count toward the
// per-stream cap.
func TestDecodeToolStartCounted(t *testing.T) {
	old := maxStreamBytes
	maxStreamBytes = 8
	defer func() { maxStreamBytes = old }()
	start := block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"12345","name":"6789","input":{}}}`)
	_, err := decodeString(t, sseHead+start+block(`{"type":"content_block_stop","index":0}`)+sseTail)
	if err == nil || !strings.Contains(err.Error(), "stream exceeds 8 bytes") {
		t.Fatalf("err = %v", err)
	}
}

// TestDecodeToolInputNullRefused: null is valid JSON but not an object.
func TestDecodeToolInputNullRefused(t *testing.T) {
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t8","name":"x","input":{}}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"null"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	_, err := decodeString(t, sse)
	if err == nil || !strings.Contains(err.Error(), `tool call "t8" input is not a JSON object`) {
		t.Fatalf("err = %v", err)
	}
}

// TestDecodeErrorNullIsMalformed: an error event whose error is null is
// malformed, not an empty StreamError.
func TestDecodeErrorNullIsMalformed(t *testing.T) {
	_, err := decodeString(t, sseHead+block(`{"type":"error","error":null}`)+block(`{"type":"message_stop"}`))
	var se *StreamError
	if !errors.As(err, &se) || se.Type != "error" || se.Message != "malformed error event" {
		t.Fatalf("err = %#v", err)
	}
}

// TestDecodeDeltaKindMismatch: a known delta on a block of another kind is
// a protocol error, not silently accumulated.
func TestDecodeDeltaKindMismatch(t *testing.T) {
	text := block(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
	tool := block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"t","name":"x","input":{}}}`)
	for name, c := range map[string]struct{ start, delta, want string }{
		"json on text": {
			text, `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}`,
			"content block index 0 does not accept input_json_delta",
		},
		"thinking on tool": {
			tool, `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"x"}}`,
			"content block index 0 does not accept thinking_delta",
		},
		"signature on text": {
			text, `{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"x"}}`,
			"content block index 0 does not accept signature_delta",
		},
	} {
		_, err := decodeString(t, sseHead+c.start+block(c.delta)+sseTail)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	unknown := block(`{"type":"content_block_start","index":0,"content_block":{"type":"server_tool_use"}}`)
	evs, err := decodeString(t, sseHead+unknown+
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}`)+
		block(`{"type":"content_block_stop","index":0}`)+sseTail)
	if err != nil || types(evs) != "EventMessageDone" {
		t.Fatalf("unknown block kind: err=%v events=%s", err, types(evs))
	}
}
