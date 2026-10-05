package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/djm56/kirsch/internal/provider"
)

// StreamError is an error event received inside the stream.
type StreamError struct {
	// Type is the type field of the error event (for example "overloaded_error").
	Type string
	// Message is the server-supplied message of the error event, passed through
	// unchanged. For an error event with a missing, null or non-object error
	// field it is the fixed text "malformed error event" and Type is "error".
	Message string
}

// Error returns "provider stream error: <Type>: <Message>".
func (e *StreamError) Error() string {
	return fmt.Sprintf("provider stream error: %s: %s", e.Type, e.Message)
}

// maxBlockIndex is the highest content block index accepted.
const maxBlockIndex = 1023

// maxLineBytes is the longest single SSE line the scanner accepts.
const maxLineBytes = 8 << 20

// maxBlockBytes caps the content retained for one block: thinking text,
// signature, redacted data and tool input together.
var maxBlockBytes = 16 << 20

// maxEventBytes caps the pending (not yet dispatched) SSE event.
var maxEventBytes = 16 << 20

// maxStreamBytes caps all content retained across every block of the stream,
// including tool call ids and names.
var maxStreamBytes = 64 << 20

// clip returns s if len(s) <= n, otherwise backs up to preserve UTF-8 validity
// and returns s[:k] + "…" where k <= n.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	k := n
	for k > 0 && !utf8.RuneStart(s[k]) {
		k--
	}
	return s[:k] + "…"
}

// wireEvent is a decoded SSE event; fields are populated per event type.
type wireEvent struct {
	Type         string          `json:"type"`
	Index        json.RawMessage `json:"index"`
	Message      *wireMessage    `json:"message"`
	ContentBlock *wireBlock      `json:"content_block"`
	Delta        *wireDelta      `json:"delta"`
	Usage        *wireUsage      `json:"usage"`
	Error        json.RawMessage `json:"error"`
}

// wireMessage carries the message fields of message_start.
type wireMessage struct {
	Usage *wireUsage `json:"usage"`
}

// wireBlock is the content_block of a content_block_start event.
type wireBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
	Data      string `json:"data"`
	ID        string `json:"id"`
	Name      string `json:"name"`
}

// wireDelta is the delta of a content_block_delta or message_delta event.
type wireDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	Thinking    string `json:"thinking"`
	Signature   string `json:"signature"`
	PartialJSON string `json:"partial_json"`
	StopReason  string `json:"stop_reason"`
}

// wireUsage is token usage; a nil field means the event did not carry it.
type wireUsage struct {
	InputTokens              *int64 `json:"input_tokens"`
	OutputTokens             *int64 `json:"output_tokens"`
	CacheReadInputTokens     *int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens *int64 `json:"cache_creation_input_tokens"`
}

// blockKind is the kind of a content block. Any wire type the decoder does not
// know maps to kindUnknown, so hostile type strings are never retained.
type blockKind int

const (
	kindUnknown blockKind = iota
	kindText
	kindThinking
	kindRedacted
	kindToolUse
)

// kindOf maps a wire block type to its blockKind.
func kindOf(wireType string) blockKind {
	switch wireType {
	case "text":
		return kindText
	case "thinking":
		return kindThinking
	case "redacted_thinking":
		return kindRedacted
	case "tool_use":
		return kindToolUse
	}
	return kindUnknown
}

// deltaKind maps a known wire delta type to the block kind that accepts it.
func deltaKind(deltaType string) (blockKind, bool) {
	switch deltaType {
	case "text_delta":
		return kindText, true
	case "thinking_delta", "signature_delta":
		return kindThinking, true
	case "input_json_delta":
		return kindToolUse, true
	}
	return kindUnknown, false
}

// blockState is the retained state of one started content block.
type blockState struct {
	kind      blockKind
	thinking  strings.Builder
	signature strings.Builder
	data      string // redacted_thinking payload
	toolID    string
	toolName  string
	toolInput strings.Builder
	closed    bool // set by content_block_stop
	bytes     int  // retained content bytes, checked against maxBlockBytes
}

// decoder holds the state of one stream decode.
type decoder struct {
	ctx          context.Context
	onEvent      func(provider.StreamEvent)
	messageUsage *provider.Usage
	blocks       map[int]*blockState
	stopReason   string // raw stop reason string; mapped to StopReason at message_stop
	streamBytes  int64  // retained content bytes, checked against maxStreamBytes
}

// Decode reads a Messages API SSE stream from r and reports it through onEvent.
//
// Contract:
//   - Events arrive in stream order.
//   - MessageDone is the last event on success and always carries a non-nil Usage.
//   - Every non-cancellation failure emits exactly one EventError carrying the
//     returned error; nothing follows it.
//   - Cancellation returns ctx.Err() with no event.
//   - Decode cannot interrupt a blocked Read, so the caller closes the body on
//     cancellation.
//   - Content block indexes must be integers in [0, 1023].
//
// Memory is bounded by three caps. maxEventBytes (16 MiB) limits one pending SSE
// event, maxBlockBytes (16 MiB) limits the content retained for one block, and
// maxStreamBytes (64 MiB) limits all content retained across the stream. Retained
// content is thinking text, signatures, redacted data, tool input, and tool call
// ids and names, whether it arrives on a start event or a delta. Single lines
// are capped at 8 MiB by the scanner.
func Decode(ctx context.Context, r io.Reader, onEvent func(provider.StreamEvent)) error {
	d := &decoder{ctx: ctx, onEvent: onEvent, blocks: make(map[int]*blockState)}
	if err := d.run(r); err != nil {
		return d.fail(err)
	}
	return nil
}

// fail is the only place an EventError is emitted. A cancelled context wins
// over any error: it returns ctx.Err() and emits nothing.
func (d *decoder) fail(err error) error {
	if cerr := d.ctx.Err(); cerr != nil {
		return cerr
	}
	d.onEvent(provider.StreamEvent{Type: provider.EventError, Err: err})
	return err
}

// run reads SSE lines from r and dispatches each complete event. It returns nil
// only after message_stop; any other return is a failure for fail to report.
func (d *decoder) run(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLineBytes)
	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if buf.Len() == 0 {
				continue
			}
			stopped, err := d.processEvent(buf.String())
			if err != nil {
				return err
			}
			if stopped {
				return nil
			}
			buf.Reset()
			continue
		}
		// Comments (":"), event:, id:, retry: and unknown fields are ignored.
		if rest, ok := strings.CutPrefix(line, "data:"); ok {
			if err := appendData(&buf, rest); err != nil {
				return err
			}
		}
	}
	if err := d.ctx.Err(); err != nil {
		return err
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("provider stream read: %w", err)
	}
	return errors.New("stream ended before message_stop")
}

// appendData adds one data line to the pending event, stripping one leading
// space and joining lines with "\n". It fails once the event exceeds
// maxEventBytes, so a stream without blank lines cannot grow it unbounded.
func appendData(buf *strings.Builder, data string) error {
	data = strings.TrimPrefix(data, " ")
	if buf.Len() > 0 {
		buf.WriteByte('\n')
	}
	buf.WriteString(data)
	if buf.Len() > maxEventBytes {
		return fmt.Errorf("sse event exceeds %d bytes", maxEventBytes)
	}
	return nil
}

// processEvent decodes and dispatches one event, then checks for cancellation.
// It reports whether the event was message_stop, which succeeds even when the
// callback cancelled while receiving MessageDone.
func (d *decoder) processEvent(data string) (bool, error) {
	if err := d.ctx.Err(); err != nil {
		return false, err
	}
	var we wireEvent
	if err := json.Unmarshal([]byte(data), &we); err != nil {
		return false, fmt.Errorf("invalid sse data: %w", err)
	}
	stopped, err := d.dispatch(&we)
	if err != nil || stopped {
		return stopped, err
	}
	return false, d.ctx.Err()
}

// dispatch routes an event by its type. It is the one place a block index is
// parsed. Unknown event types are ignored.
func (d *decoder) dispatch(we *wireEvent) (bool, error) {
	switch we.Type {
	case "content_block_start", "content_block_delta", "content_block_stop":
		idx, err := parseIndex(we.Index)
		if err != nil {
			return false, err
		}
		return false, d.dispatchBlock(idx, we)
	case "message_start":
		if we.Message != nil {
			d.messageUsage = overlayUsage(nil, we.Message.Usage)
		}
	case "message_delta":
		if we.Delta != nil && we.Delta.StopReason != "" {
			d.stopReason = we.Delta.StopReason
		}
		d.messageUsage = overlayUsage(d.messageUsage, we.Usage)
	case "message_stop":
		d.messageDone()
		return true, nil
	case "error":
		return false, streamError(we.Error)
	}
	return false, nil
}

// dispatchBlock handles the three block event types for an already parsed index.
func (d *decoder) dispatchBlock(idx int, we *wireEvent) error {
	if we.Type == "content_block_start" {
		return d.startBlock(idx, we.ContentBlock)
	}
	bs := d.blocks[idx]
	if bs == nil {
		return fmt.Errorf("content block index %d not started", idx)
	}
	if bs.closed {
		return fmt.Errorf("content block index %d already stopped", idx)
	}
	if we.Type == "content_block_stop" {
		bs.closed = true
		return d.finishBlock(bs)
	}
	if we.Delta == nil {
		return nil
	}
	return d.applyDelta(idx, bs, we.Delta)
}

// parseIndex validates a raw JSON index token against ^(0|[1-9][0-9]*)$ and the
// range [0, maxBlockIndex]. Negatives, floats, exponents, quoted numbers and
// leading zeros are all out of range; absent or null is missing.
func parseIndex(raw json.RawMessage) (int, error) {
	text := string(raw)
	if text == "" || text == "null" {
		return 0, errors.New("content block index missing")
	}
	if len(text) > len(strconv.Itoa(maxBlockIndex)) || (text[0] == '0' && len(text) > 1) {
		return 0, fmt.Errorf("content block index %s out of range", clip(text, 32))
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, fmt.Errorf("content block index %s out of range", clip(text, 32))
		}
	}
	idx, err := strconv.Atoi(text)
	if err != nil || idx > maxBlockIndex {
		return 0, fmt.Errorf("content block index %s out of range", clip(text, 32))
	}
	return idx, nil
}

// retain is the single accounting point for retained content. It charges n
// bytes against the stream cap and, when bs is non-nil, against the block cap
// too. Every path that stores content calls it before storing.
func (d *decoder) retain(idx int, bs *blockState, n int) error {
	if bs != nil {
		if bs.bytes+n > maxBlockBytes {
			return fmt.Errorf("content block index %d exceeds %d bytes", idx, maxBlockBytes)
		}
		bs.bytes += n
	}
	d.streamBytes += int64(n)
	if d.streamBytes > int64(maxStreamBytes) {
		return fmt.Errorf("stream exceeds %d bytes", maxStreamBytes)
	}
	return nil
}

// emit delivers one event unless the context is cancelled, in which case it
// returns ctx.Err() and delivers nothing.
func (d *decoder) emit(ev provider.StreamEvent) error {
	if err := d.ctx.Err(); err != nil {
		return err
	}
	d.onEvent(ev)
	return nil
}

// startBlock registers a content block and processes the content its start
// event carries. A start without a content_block registers nothing.
func (d *decoder) startBlock(idx int, cb *wireBlock) error {
	if _, used := d.blocks[idx]; used {
		return fmt.Errorf("content block index %d already started", idx)
	}
	if cb == nil {
		return nil
	}
	bs := &blockState{kind: kindOf(cb.Type)}
	var err error
	switch bs.kind {
	case kindUnknown:
		// Unknown block kinds retain nothing; only the index is recorded.
	case kindText:
		err = d.emitText(cb.Text)
	case kindThinking:
		err = d.seedThinking(idx, bs, cb)
	case kindRedacted:
		if err = d.retain(idx, bs, len(cb.Data)); err == nil {
			bs.data = cb.Data
		}
	case kindToolUse:
		err = d.startTool(bs, cb)
	}
	if err != nil {
		return err
	}
	d.blocks[idx] = bs
	return nil
}

// seedThinking keeps the text and signature a thinking start carries,
// emitting the text as a ThinkingDelta.
func (d *decoder) seedThinking(idx int, bs *blockState, cb *wireBlock) error {
	if err := d.appendThinking(idx, bs, cb.Thinking); err != nil {
		return err
	}
	return d.appendSignature(idx, bs, cb.Signature)
}

// startTool keeps a tool call's id and name, charged to the stream cap only,
// and emits ToolCallStart.
func (d *decoder) startTool(bs *blockState, cb *wireBlock) error {
	if err := d.retain(0, nil, len(cb.ID)+len(cb.Name)); err != nil {
		return err
	}
	bs.toolID, bs.toolName = cb.ID, cb.Name
	return d.emit(provider.StreamEvent{
		Type:     provider.EventToolCallStart,
		ToolCall: &provider.ToolCall{ID: cb.ID, Name: cb.Name},
	})
}

// applyDelta applies a delta to an open block. A known delta type must match
// the block's kind. Deltas of unknown type, or on a block of unknown kind, are
// ignored.
func (d *decoder) applyDelta(idx int, bs *blockState, dl *wireDelta) error {
	want, known := deltaKind(dl.Type)
	if !known || bs.kind == kindUnknown {
		return nil
	}
	if bs.kind != want {
		return fmt.Errorf("content block index %d does not accept %s", idx, dl.Type)
	}
	switch dl.Type {
	case "text_delta":
		return d.emitText(dl.Text)
	case "thinking_delta":
		return d.appendThinking(idx, bs, dl.Thinking)
	case "signature_delta":
		return d.appendSignature(idx, bs, dl.Signature)
	}
	return d.appendToolInput(idx, bs, dl.PartialJSON)
}

// emitText emits a TextDelta for non-empty text. Text is streamed, not retained.
func (d *decoder) emitText(text string) error {
	if text == "" {
		return nil
	}
	return d.emit(provider.StreamEvent{Type: provider.EventTextDelta, Text: text})
}

// appendThinking retains thinking text and emits it as a ThinkingDelta.
func (d *decoder) appendThinking(idx int, bs *blockState, text string) error {
	if text == "" {
		return nil
	}
	if err := d.retain(idx, bs, len(text)); err != nil {
		return err
	}
	bs.thinking.WriteString(text)
	return d.emit(provider.StreamEvent{Type: provider.EventThinkingDelta, Text: text})
}

// appendSignature retains signature bytes silently; they surface in ThinkingDone.
func (d *decoder) appendSignature(idx int, bs *blockState, sig string) error {
	if err := d.retain(idx, bs, len(sig)); err != nil {
		return err
	}
	bs.signature.WriteString(sig)
	return nil
}

// appendToolInput retains a tool input fragment and emits it as a ToolCallDelta.
func (d *decoder) appendToolInput(idx int, bs *blockState, partial string) error {
	if partial == "" {
		return nil
	}
	if err := d.retain(idx, bs, len(partial)); err != nil {
		return err
	}
	bs.toolInput.WriteString(partial)
	return d.emit(provider.StreamEvent{Type: provider.EventToolCallDelta, Text: partial})
}

// finishBlock emits the closing event for a stopped block: ThinkingDone for
// normal or redacted thinking, ToolCallEnd for a tool call. Text and unknown
// blocks emit nothing.
func (d *decoder) finishBlock(bs *blockState) error {
	switch bs.kind {
	case kindThinking:
		return d.emit(provider.StreamEvent{
			Type:     provider.EventThinkingDone,
			Thinking: &provider.Thinking{Text: bs.thinking.String(), Signature: bs.signature.String()},
		})
	case kindRedacted:
		return d.emit(provider.StreamEvent{
			Type:     provider.EventThinkingDone,
			Thinking: &provider.Thinking{Redacted: true, Data: bs.data},
		})
	case kindToolUse:
		input, err := toolInput(bs)
		if err != nil {
			return err
		}
		return d.emit(provider.StreamEvent{
			Type:     provider.EventToolCallEnd,
			ToolCall: &provider.ToolCall{ID: bs.toolID, Name: bs.toolName, Input: input},
		})
	case kindText, kindUnknown:
		// Nothing closes a text or unknown block.
	}
	return nil
}

// toolInput returns the accumulated tool input: "{}" when empty, otherwise the
// input, which must be valid JSON and a JSON object (null and arrays are not).
func toolInput(bs *blockState) (json.RawMessage, error) {
	if bs.toolInput.Len() == 0 {
		return json.RawMessage("{}"), nil
	}
	raw := []byte(bs.toolInput.String())
	if !json.Valid(raw) {
		return nil, fmt.Errorf("tool call %q input is not valid JSON", clip(bs.toolID, 64))
	}
	if bytes.TrimLeft(raw, " \t\r\n")[0] != '{' {
		return nil, fmt.Errorf("tool call %q input is not a JSON object", clip(bs.toolID, 64))
	}
	return raw, nil
}

// messageDone emits MessageDone with a non-nil Usage. It bypasses the
// cancellation check: once the terminal event is dispatched the stream has
// succeeded, even if the callback cancels while receiving it.
func (d *decoder) messageDone() {
	if d.messageUsage == nil {
		d.messageUsage = &provider.Usage{}
	}
	d.onEvent(provider.StreamEvent{
		Type:       provider.EventMessageDone,
		Usage:      d.messageUsage,
		StopReason: mapStopReason(d.stopReason),
	})
}

// streamError converts the error field of an error event to a *StreamError. A
// missing, null or malformed error object becomes "malformed error event".
func streamError(raw json.RawMessage) error {
	var ed *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &ed) != nil || ed == nil {
		return &StreamError{Type: "error", Message: "malformed error event"}
	}
	return &StreamError{Type: clip(ed.Type, 512), Message: clip(ed.Message, 512)}
}

// clamp32 converts a wire count to int, limiting it to [0, MaxInt32].
func clamp32(v int64) int {
	if v < 0 {
		return 0
	}
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int(v)
}

// overlayUsage overlays the usage fields wu carries onto current (allocating it
// if nil), clamping each to [0, MaxInt32]. Fields wu omits are left unchanged.
func overlayUsage(current *provider.Usage, wu *wireUsage) *provider.Usage {
	if current == nil {
		current = &provider.Usage{}
	}
	if wu == nil {
		return current
	}
	for _, f := range []struct {
		src *int64
		dst *int
	}{
		{wu.InputTokens, &current.InputTokens},
		{wu.OutputTokens, &current.OutputTokens},
		{wu.CacheReadInputTokens, &current.CacheReadTokens},
		{wu.CacheCreationInputTokens, &current.CacheWriteTokens},
	} {
		if f.src != nil {
			*f.dst = clamp32(*f.src)
		}
	}
	return current
}

// mapStopReason maps a wire stop reason to the neutral type; unknown values
// become StopOther.
func mapStopReason(reason string) provider.StopReason {
	switch reason {
	case "end_turn":
		return provider.StopEndTurn
	case "tool_use":
		return provider.StopToolUse
	case "max_tokens":
		return provider.StopMaxTokens
	}
	return provider.StopOther
}
