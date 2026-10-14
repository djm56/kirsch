// Package session writes Kirsch's durable JSONL session log.
//
// It implements agent.Recorder with a single writer goroutine so every line
// written to the session file is owned by one goroutine. The store serialises
// each Record call into a JSONL event and queues it for the writer; the writer
// batches output with bufio and fsyncs on a ticker and on explicit Flush.
package session

import (
	"encoding/json"
	"fmt"

	"github.com/djm56/kirsch/internal/agent"
)

const (
	// eventVersion is the JSONL schema version required by ADR 0002.
	eventVersion = 1

	// eventTypeUserMessage is the JSONL type for a user text message.
	eventTypeUserMessage = "user.message"
	// eventTypeAssistantMessage is the JSONL type for an assistant message.
	eventTypeAssistantMessage = "assistant.message"
	// eventTypeAssistantThinking is the JSONL type for an assistant thinking block.
	eventTypeAssistantThinking = "assistant.thinking"
	// eventTypeToolCompleted is the JSONL type for a completed tool call.
	eventTypeToolCompleted = "tool.completed"
)

// baseEvent carries fields present on every JSONL line.
type baseEvent struct {
	// Version is the JSONL schema version, always eventVersion.
	Version int `json:"v"`
	// Type names the event type per plan §7.
	Type string `json:"type"`
}

// userMessageEvent records a block of user text.
//
// Consecutive user.message events emitted for the same user turn are merged
// by the loader so that a RecordMerge delta does not require a separate event
type userMessageEvent struct {
	baseEvent
	Role    string `json:"role"`
	Content string `json:"content"`
}

// toolCallEvent is the shape of a tool call inside assistant.message.
type toolCallEvent struct {
	ID    string          `json:"id"`
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

// assistantMessageEvent records an assistant turn.
type assistantMessageEvent struct {
	baseEvent
	Content   string          `json:"content"`
	ToolCalls []toolCallEvent `json:"tool_calls,omitempty"`
	// StopReason is not populated in m4-d1; the agent layer does not yet
	// surface it. The field is reserved for m4-d3 end-to-end verification.
	StopReason string `json:"stop_reason,omitempty"`
}

// assistantThinkingEvent records an assistant thinking block.
type assistantThinkingEvent struct {
	baseEvent
	Content   string `json:"content"`
	Signature string `json:"signature,omitempty"`
}

// toolCompletedEvent records the result of a tool call.
type toolCompletedEvent struct {
	baseEvent
	ID        string `json:"id"`
	Tool      string `json:"tool,omitempty"`
	OK        bool   `json:"ok"`
	Content   string `json:"content,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	// DurationMS is not populated in m4-d1; the agent layer does not yet
	// surface timing. The field is reserved for m4-d3 end-to-end verification.
	DurationMS int `json:"duration_ms,omitempty"`
}

// encode turns a RecordKind and Message into one or more JSONL lines.
//
// It owns the RecordKind to plan-§7 event-type mapping. An error means the
// message cannot be represented in the current schema and is reported through
// the store's error hook; no line is queued.
func encode(kind agent.RecordKind, m agent.Message) ([][]byte, error) {
	switch kind {
	case agent.RecordAppend:
		return encodeAppend(m)
	case agent.RecordMerge:
		return encodeMerge(m)
	default:
		return nil, fmt.Errorf("unknown RecordKind %d", kind)
	}
}

// encodeAppend handles RecordAppend events.
func encodeAppend(m agent.Message) ([][]byte, error) {
	switch m.Role {
	case agent.RoleUser:
		return encodeUserAppend(m)
	case agent.RoleAssistant:
		return encodeAssistantAppend(m)
	default:
		return nil, fmt.Errorf("unknown message role %q for RecordAppend", m.Role)
	}
}

// encodeMerge handles RecordMerge events. In the current agent this is only
// used to append text to the last user message, so it emits a user.message.
func encodeMerge(m agent.Message) ([][]byte, error) {
	if m.Role != agent.RoleUser {
		return nil, fmt.Errorf("RecordMerge for role %q is not supported", m.Role)
	}
	if len(m.Content) != 1 || m.Content[0].Kind != agent.BlockText {
		return nil, fmt.Errorf("RecordMerge user message must contain exactly one text block, got %d block(s)", len(m.Content))
	}
	line, err := marshalLine(userMessageEvent{
		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
		Role:      "user",
		Content:   m.Content[0].Text,
	})
	if err != nil {
		return nil, err
	}
	return [][]byte{line}, nil
}

// encodeUserAppend encodes a user RecordAppend.
//
// A user append is either a text message or one or more tool results, never a
// mixture in the current agent.
func encodeUserAppend(m agent.Message) ([][]byte, error) {
	if len(m.Content) == 0 {
		return nil, nil
	}
	kind := m.Content[0].Kind
	for _, b := range m.Content[1:] {
		if b.Kind != kind {
			return nil, fmt.Errorf("mixed block kinds in user RecordAppend: %v", blockKinds(m.Content))
		}
	}
	switch kind {
	case agent.BlockText:
		var text string
		for _, b := range m.Content {
			text += b.Text
		}
		line, err := marshalLine(userMessageEvent{
			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
			Role:      "user",
			Content:   text,
		})
		if err != nil {
			return nil, err
		}
		return [][]byte{line}, nil
	case agent.BlockToolResult:
		var lines [][]byte
		for _, b := range m.Content {
			if b.Result == nil {
				return nil, fmt.Errorf("BlockToolResult with nil Result")
			}
			line, err := marshalLine(toolCompletedEvent{
				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
				ID:        b.Result.CallID,
				OK:        !b.Result.IsError,
				Content:   b.Result.Content,
				Truncated: b.Result.Truncated,
			})
			if err != nil {
				return nil, err
			}
			lines = append(lines, line)
		}
		return lines, nil
	default:
		return nil, fmt.Errorf("unsupported block kind in user RecordAppend: %v", kind)
	}
}

// encodeAssistantAppend encodes an assistant RecordAppend.
//
// Thinking blocks are emitted as separate assistant.thinking events. Text and
// tool-call blocks are emitted as a single assistant.message event; the plan §7
// representation places text in content and tool calls in tool_calls, so the
// original interleaving of text and tool calls within one message is flattened.
func encodeAssistantAppend(m agent.Message) ([][]byte, error) {
	var lines [][]byte
	var text string
	var calls []toolCallEvent
	for _, b := range m.Content {
		switch b.Kind {
		case agent.BlockText:
			text += b.Text
		case agent.BlockThinking:
			if b.Thinking == nil {
				return nil, fmt.Errorf("BlockThinking with nil Thinking")
			}
			line, err := marshalLine(assistantThinkingEvent{
				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantThinking},
				Content:   b.Thinking.Text,
				Signature: b.Thinking.Signature,
			})
			if err != nil {
				return nil, err
			}
			lines = append(lines, line)
		case agent.BlockToolCall:
			if b.ToolCall == nil {
				return nil, fmt.Errorf("BlockToolCall with nil ToolCall")
			}
			calls = append(calls, toolCallEvent{
				ID:    b.ToolCall.ID,
				Tool:  b.ToolCall.Name,
				Input: b.ToolCall.Input,
			})
		default:
			return nil, fmt.Errorf("unsupported block kind in assistant RecordAppend: %v", b.Kind)
		}
	}
	if text != "" || len(calls) > 0 {
		line, err := marshalLine(assistantMessageEvent{
			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantMessage},
			Content:   text,
			ToolCalls: calls,
		})
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// marshalLine marshals v to a single JSONL line terminated by a newline.
func marshalLine(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// blockKinds returns the kinds of blocks for error messages.
func blockKinds(blocks []agent.Block) []agent.BlockKind {
	kinds := make([]agent.BlockKind, len(blocks))
	for i, b := range blocks {
		kinds[i] = b.Kind
	}
	return kinds
}
