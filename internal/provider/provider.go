// Package provider defines the provider-neutral interface for model streaming.
//
// Everything in this package is expressible without naming any vendor. The
// provider-specific wire format and HTTP details live in a separate adapter
// package.
package provider

import (
	"context"
	"encoding/json"
)

// Provider is the interface that model providers must implement.
type Provider interface {
	// Stream sends a request to the model and streams events through the callback.
	// It returns an error if the request fails. It is cancelled when ctx is cancelled.
	Stream(ctx context.Context, req Request, onEvent func(StreamEvent)) error
}

// Role is the role of a message sender.
type Role string

const (
	// RoleUser is a user message.
	RoleUser Role = "user"
	// RoleAssistant is an assistant message.
	RoleAssistant Role = "assistant"
)

// Request is a request to stream a model response.
type Request struct {
	Model     string
	System    []SystemBlock
	Messages  []Message
	Tools     []ToolDef
	MaxTokens int
	Thinking  ThinkingLevel
}

// SystemBlock is a block in the system prompt.
// Cache is a neutral abstraction over provider-specific prompt caching features.
type SystemBlock struct {
	// Text is the system prompt text.
	Text string
	// Cache is true to place a prompt-cache breakpoint after this block.
	Cache bool
}

// ToolDef is a tool definition sent to the model.
type ToolDef struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// Message is a message in the conversation.
type Message struct {
	Role    Role
	Content []Block
}

// BlockKind identifies the kind of block.
type BlockKind int

const (
	// BlockText is a text block.
	BlockText BlockKind = iota
	// BlockThinking is a thinking block.
	BlockThinking
	// BlockToolCall is a tool call block.
	BlockToolCall
	// BlockToolResult is a tool result block.
	BlockToolResult
)

// String returns the string representation of a BlockKind.
func (b BlockKind) String() string {
	switch b {
	case BlockText:
		return "BlockText"
	case BlockThinking:
		return "BlockThinking"
	case BlockToolCall:
		return "BlockToolCall"
	case BlockToolResult:
		return "BlockToolResult"
	default:
		return "BlockUnknown"
	}
}

// Block is a block of content in a message.
type Block struct {
	Kind       BlockKind
	Text       string      // BlockText
	Thinking   *Thinking   // BlockThinking
	ToolCall   *ToolCall   // BlockToolCall
	ToolResult *ToolResult // BlockToolResult
}

// Thinking is a reasoning block. Signature and Data are opaque, provider-issued
// continuation tokens. They must be echoed back byte-identical within a turn,
// at every ThinkingLevel. Redacted and Data are neutral abstractions over
// provider-specific reasoning features.
type Thinking struct {
	// Text is the thinking text.
	Text string
	// Signature is an opaque provider-issued token for continuation.
	Signature string
	// Redacted is true if Text is empty and Data carries the opaque content.
	Redacted bool
	// Data is the opaque content of a redacted block, echoed back unchanged.
	Data string
}

// ToolCall is a tool call in the assistant message.
type ToolCall struct {
	ID    string // the provider's own id; requests, approvals and results correlate on it
	Name  string
	Input json.RawMessage // complete, valid JSON
}

// ToolResult is the result of a tool call.
type ToolResult struct {
	CallID  string
	Content string
	IsError bool
}

// ThinkingLevel controls the amount of thinking in responses.
type ThinkingLevel int

const (
	// ThinkingOff disables thinking blocks.
	ThinkingOff ThinkingLevel = iota
	// ThinkingLow enables low-level thinking.
	ThinkingLow
	// ThinkingMedium enables medium-level thinking.
	ThinkingMedium
	// ThinkingHigh enables high-level thinking.
	ThinkingHigh
)

// String returns the string representation of a ThinkingLevel.
func (t ThinkingLevel) String() string {
	switch t {
	case ThinkingOff:
		return "off"
	case ThinkingLow:
		return "low"
	case ThinkingMedium:
		return "medium"
	case ThinkingHigh:
		return "high"
	default:
		return "unknown"
	}
}

// EventType identifies the type of stream event.
type EventType int

const (
	// EventTextDelta is a text delta event.
	EventTextDelta EventType = iota
	// EventThinkingDelta is a thinking delta event.
	EventThinkingDelta
	// EventThinkingDone is a thinking done event.
	EventThinkingDone
	// EventToolCallStart is a tool call start event.
	EventToolCallStart
	// EventToolCallDelta is a tool call delta event.
	EventToolCallDelta
	// EventToolCallEnd is a tool call end event.
	EventToolCallEnd
	// EventMessageDone is a message done event.
	EventMessageDone
	// EventError is an error event.
	EventError
)

// String returns the string representation of an EventType.
func (e EventType) String() string {
	switch e {
	case EventTextDelta:
		return "EventTextDelta"
	case EventThinkingDelta:
		return "EventThinkingDelta"
	case EventThinkingDone:
		return "EventThinkingDone"
	case EventToolCallStart:
		return "EventToolCallStart"
	case EventToolCallDelta:
		return "EventToolCallDelta"
	case EventToolCallEnd:
		return "EventToolCallEnd"
	case EventMessageDone:
		return "EventMessageDone"
	case EventError:
		return "EventError"
	default:
		return "EventUnknown"
	}
}

// StopReason is the reason the model stopped generating.
type StopReason string

const (
	// StopEndTurn means the model ended the turn.
	StopEndTurn StopReason = "end_turn"
	// StopToolUse means the model is requesting tool use.
	StopToolUse StopReason = "tool_use"
	// StopMaxTokens means the max token limit was reached.
	StopMaxTokens StopReason = "max_tokens"
	// StopOther means the model stopped for another reason.
	StopOther StopReason = "other"
)

// String returns the string representation of a StopReason.
func (s StopReason) String() string {
	return string(s)
}

// Usage tracks token usage.
type Usage struct {
	// InputTokens is the number of input tokens used.
	InputTokens int
	// OutputTokens is the number of output tokens used.
	OutputTokens int
	// CacheReadTokens is the number of tokens read from cache.
	CacheReadTokens int
	// CacheWriteTokens is the number of tokens written to cache.
	CacheWriteTokens int
}

// StreamEvent is an event emitted during streaming.
type StreamEvent struct {
	// Type is the event type.
	Type EventType
	// Text is the text fragment for TextDelta, ThinkingDelta, or ToolCallDelta events.
	Text string
	// Thinking is the complete thinking block for ThinkingDone events.
	Thinking *Thinking
	// ToolCall is the tool call (ID and Name only for ToolCallStart; complete with Input for ToolCallEnd).
	ToolCall *ToolCall
	// Usage is the token usage for MessageDone events.
	Usage *Usage
	// StopReason is the stop reason for MessageDone events.
	StopReason StopReason
	// Err is the error for Error events.
	Err error
}
