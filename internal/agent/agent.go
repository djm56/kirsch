// Package agent implements the turn state machine for multi-turn conversations with tools.
//
// The agent coordinates between a Model (provider interface), Tools (execution interface),
// and a Recorder (session log interface). It is testable in isolation with fakes.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrStreamProtocol indicates a violation of the model stream protocol.
var ErrStreamProtocol = errors.New("agent: model stream protocol violation")

// ErrStreamFailed indicates the model stream emitted an EventError whose Err was
// nil. Turn returns it as is, not wrapped.
var ErrStreamFailed = errors.New("agent: model stream error event without an error")

// ErrMaxTurnsExceeded indicates the turn reached the maximum number of tool
// rounds without the model producing a final answer.
var ErrMaxTurnsExceeded = errors.New("agent: max turns exceeded")

// ErrToolNameHallucination indicates the model exhausted the allowed retries for
// hallucinated tool names.
var ErrToolNameHallucination = errors.New("agent: tool name hallucination retries exhausted")

// Tool error kinds returned to the model in result blocks. They mirror the
// plan §3.3 vocabulary without importing the implementation tool package.
const (
	toolErrPolicyDenied = "policy_denied"
	toolErrCancelled    = "cancelled"
	toolErrInputInvalid = "tool_input_invalid"
)

// defaultMaxToolRounds is the default number of tool rounds a Turn is allowed
// before the max-turn guard trips.
const defaultMaxToolRounds = 25

// maxHallucinationRetries is the number of retries the model gets for
// hallucinated tool names before Turn returns ErrToolNameHallucination.
const maxHallucinationRetries = 2

// Role is the role of a message sender.
type Role string

const (
	// RoleUser is a user message.
	RoleUser Role = "user"
	// RoleAssistant is an assistant message.
	RoleAssistant Role = "assistant"
)

// BlockKind identifies the kind of block in a message.
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

// Thinking is a reasoning block. Signature and Data are opaque, provider-issued
// continuation tokens. They must be echoed back byte-identical within a turn,
// at every thinking level.
type Thinking struct {
	// Text is the thinking text.
	Text string
	// Signature is an opaque provider-issued token for continuation.
	Signature string
	// Redacted is true if Text is empty and Data carries the opaque content.
	Redacted bool
	// Data is the opaque content of a redacted block, echoed back unchanged.
	Data string
	// DropOnSummary is the M4-compaction marker. When true, compactors must
	// drop this block rather than summarise it. It is set by the agent on every
	// assembled thinking block and preserved by the deep-copy helpers.
	DropOnSummary bool
}

// ToolCall is a tool call in an assistant message.
type ToolCall struct {
	// ID is the provider's own id; requests, approvals and results correlate on it.
	ID string
	// Name is the name of the tool being called.
	Name string
	// Input is the tool input as complete, valid JSON.
	Input json.RawMessage
}

// ToolResult is the result of a tool invocation, returned by Tools.Invoke.
type ToolResult struct {
	// OK is true if the tool execution succeeded.
	OK bool
	// Content is the tool output or error message.
	Content string
	// Truncated is true if the output was truncated.
	Truncated bool
	// ErrorKind is the error kind, for structured error handling.
	ErrorKind string
}

// ResultBlock is a tool result in a message sent back to the model.
type ResultBlock struct {
	// CallID is the ID of the tool call this result answers.
	CallID string
	// Content is the result content.
	Content string
	// IsError is true if this is an error result.
	IsError bool
	// Truncated is true if the tool truncated its output. It is carried from
	// ToolResult so a recorded session can tell a truncated result from a
	// complete one; whether the provider adapter shows it to the model is the
	// adapter's decision, not the agent's.
	Truncated bool
	// ErrorKind is the error kind for structured error handling.
	ErrorKind string
}

// Block is a block of content in a message.
type Block struct {
	// Kind identifies the kind of block.
	Kind BlockKind
	// Text is the content for BlockText.
	Text string
	// Thinking is the reasoning block for BlockThinking.
	Thinking *Thinking
	// ToolCall is the tool call for BlockToolCall.
	ToolCall *ToolCall
	// Result is the tool result for BlockToolResult.
	Result *ResultBlock
}

// Message is a message in the conversation.
type Message struct {
	// Role is the sender's role.
	Role Role
	// Content is the message content blocks.
	Content []Block
}

// ToolSpec is a tool specification sent to the model.
type ToolSpec struct {
	// Name is the tool name.
	Name string
	// Description is the tool description.
	Description string
	// InputSchema is the tool input schema as JSON.
	InputSchema json.RawMessage
}

// Request is a request to the model.
type Request struct {
	// Messages is the conversation history.
	Messages []Message
	// Tools is the available tool specifications.
	Tools []ToolSpec
	// MaxTokens is the maximum output tokens.
	MaxTokens int
	// System is the system prompt text. The agent package does not import
	// provider, so this is a plain string; the app adapter maps it to
	// provider.SystemBlock slices.
	System string
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

// Event is an event emitted during model streaming.
type Event struct {
	// Type is the event type.
	Type EventType
	// Text is the text fragment for TextDelta, ThinkingDelta, or ToolCallDelta.
	Text string
	// Thinking is the complete thinking block for ThinkingDone.
	Thinking *Thinking
	// ToolCall is the complete tool call for ToolCallEnd, or partial for ToolCallStart.
	ToolCall *ToolCall
	// Usage is the token usage for MessageDone.
	Usage *Usage
	// StopReason is the stop reason for MessageDone.
	StopReason string
	// Err is the error for EventError.
	Err error
}

// ModelInfo holds model capability information.
type ModelInfo struct {
	// ContextWindow is the model's context window size in tokens.
	ContextWindow int
	// MaxOutput is the maximum output tokens.
	MaxOutput int
}

// Model is the interface for a model provider.
type Model interface {
	// Stream sends a request to the model and streams events through the callback.
	// It is cancelled when ctx is cancelled.
	Stream(ctx context.Context, req Request, onEvent func(Event)) error
	// Info returns the model's capability information.
	Info() ModelInfo
}

// Tools is the interface for tool execution.
type Tools interface {
	// Describe returns the list of available tools.
	Describe() []ToolSpec
	// Invoke executes a tool call and returns the result.
	Invoke(ctx context.Context, c ToolCall) ToolResult
}

// RecordKind says how a recorded message relates to the conversation.
type RecordKind int

const (
	// RecordAppend means the message is a new message appended to the end of the
	// conversation. It is the zero value.
	RecordAppend RecordKind = iota
	// RecordMerge means the message is not a new message: its Content blocks are
	// appended to the Content of the conversation's last message, which has the
	// same Role as the recorded message.
	RecordMerge
)

// Recorder receives every change Turn makes to a Conversation, in the order Turn
// makes it.
//
// Contract. Replaying the records in order onto the conversation as Turn found it
// rebuilds the conversation exactly: RecordAppend appends the message as a new
// message, RecordMerge appends the message's Content blocks to the Content of the
// last message. Nothing else changes the conversation, and nothing is recorded
// that is not in it: the partial assistant message of a failed stream is never
// appended to the conversation and so is never recorded. Turn hands Record a deep
// copy that the Recorder owns; it may keep or modify the copy, and nothing it does
// can reach the conversation or a later request. Record is called synchronously
// from Turn and must not call back into the Agent.
type Recorder interface {
	// Record records one change to the conversation.
	Record(kind RecordKind, m Message)
}

// Conversation holds the message history for a multi-turn conversation.
type Conversation struct {
	// Messages is the conversation history.
	Messages []Message
}

// Agent implements the turn state machine for multi-turn conversations.
type Agent struct {
	// Model is the provider interface.
	Model Model
	// Tools is the tool execution interface.
	Tools Tools
	// Recorder records changes to the conversation. May be nil.
	Recorder Recorder
	// MaxTokens is the maximum output tokens for requests.
	MaxTokens int
	// System is the system prompt sent with every model request. It is set at
	// construction and treated as stable for the life of the session, which
	// keeps the prompt cache friendly. The plan requires the prompt on every
	// request; storing it on the Agent is a spec-silent design choice.
	System string
}

// Turn runs one user turn: it adds userText to conv, calls the model, invokes the
// tool calls the model returns, and calls the model again with the results, until
// a model message carries no tool calls.
//
// User text. If the last message of conv is a user message, userText is added to
// it as a further BlockText block (a merge, recorded as RecordMerge) and no second
// user message is created. That is the state a failed Turn leaves behind: the
// conversation keeps everything that happened, so it can end on the user message
// (an error before any assistant reply) or on a tool-results message (an error in
// a later round). Otherwise userText becomes a new user message (RecordAppend).
// A failed Turn never removes anything it added.
//
// Each round, in order:
//   - Model.Stream gets a deep copy of the conversation; the tool specs are
//     passed as Tools.Describe returns them, not copied.
//   - The streamed events are accumulated into an assistant message. The agent
//     enforces the stream protocol: EventMessageDone must arrive, and be the last
//     event; text, EventToolCallEnd and EventMessageDone are rejected while a
//     thinking block is open; the message must have at least one block; the
//     payloads of EventThinkingDone and EventToolCallEnd must be non-nil. Events
//     the agent does not accumulate (EventToolCallStart, EventToolCallDelta) are
//     only forwarded, even while a thinking block is open: the call they describe
//     is rejected at its EventToolCallEnd if the block is still open then. An
//     EventError is accepted before EventMessageDone, when it is the first error;
//     after EventMessageDone it is an event after MessageDone like any other.
//     A message made only of thinking blocks is valid: it has at least one block,
//     so it is appended, and with no tool calls it ends the Turn. Payloads are
//     copied; the model's own structs are never modified or stored.
//   - The assistant message is appended and recorded.
//   - Without tool calls, Turn returns nil. With tool calls, they are invoked
//     sequentially, in order, each with a copy of its ToolCall, and every result
//     goes into one user message that is appended and recorded.
//
// Forwarding. emit gets each event the agent accepts, in arrival order, after the
// agent has taken its own copy of the payload. An event that violates the protocol
// is not forwarded, and neither is any event after the first error; an EventError
// is forwarded when it is the first error. emit may be nil. On each error path the
// events that reach emit are:
//   - EventError (with or without Err): the events before it, and the EventError.
//   - Protocol violation by an event, including an event after EventMessageDone:
//     the events before the violating event; not the violating event, not the rest.
//   - Model.Stream returning an error: every event the agent accepted before it.
//   - Stream ending without EventMessageDone, or with an empty message: every
//     event of the stream.
//
// Errors. The first error of a round wins. Errors found in events are kept in
// preference to the error Model.Stream returns afterwards: an earlier
// EventError or violation is the cause, a later Stream error a consequence. An
// EventError carrying Err returns that Err wrapped as "agent: model stream: ...";
// one without Err returns ErrStreamFailed as is; a violation wraps
// ErrStreamProtocol with its reason; a Stream error is wrapped as "agent: model
// stream: ...". Turn never returns nil on a failed round, and the partial
// assistant message of a failed round is not appended.
func (a *Agent) Turn(ctx context.Context, conv *Conversation, userText string, emit func(Event)) error {
	a.addUserText(conv, userText)

	toolRounds := 0
	hallucinationRetries := 0

	for {
		if toolRounds >= defaultMaxToolRounds {
			return fmt.Errorf("%w: %d tool rounds exceeded limit of %d", ErrMaxTurnsExceeded, toolRounds, defaultMaxToolRounds)
		}

		req := Request{
			Messages:  copyMessages(conv.Messages),
			Tools:     a.Tools.Describe(),
			MaxTokens: a.MaxTokens,
			System:    a.System,
		}

		assistant, err := a.streamAssistant(ctx, req, emit)
		if err != nil {
			return err
		}

		calls := toolCalls(assistant)

		// validateCallIDs runs before appendMessage because a malformed
		// assistant message is not something that happened; the Recorder
		// contract keeps history only for turns that occurred.
		if err := validateCallIDs(calls); err != nil {
			return err
		}

		// The assistant message is committed only once its tool calls have
		// identifiers the agent can answer unambiguously.
		a.appendMessage(conv, assistant)

		if len(calls) == 0 {
			return nil
		}

		results, hasHallucination, err := a.invokeTools(ctx, calls)
		if err != nil {
			// Cancellation after the assistant message was appended must leave a
			// result block for every tool_use id, with cancelled for the calls
			// that did not finish.
			if errors.Is(err, context.Canceled) {
				a.appendMessage(conv, resultMessage(results))
			}
			return err
		}

		a.appendMessage(conv, resultMessage(results))
		toolRounds++
		if hasHallucination {
			hallucinationRetries++
			if hallucinationRetries > maxHallucinationRetries {
				return fmt.Errorf("%w: model produced unknown tool names %d times", ErrToolNameHallucination, hallucinationRetries)
			}
		}
	}
}

// addUserText adds userText to conv: as a further text block of the last message
// when that is a user message, otherwise as a new user message. Both are recorded.
func (a *Agent) addUserText(conv *Conversation, userText string) {
	block := Block{Kind: BlockText, Text: userText}
	if n := len(conv.Messages); n > 0 && conv.Messages[n-1].Role == RoleUser {
		last := &conv.Messages[n-1]
		last.Content = append(last.Content, block)
		a.record(RecordMerge, Message{Role: RoleUser, Content: []Block{block}})
		return
	}
	a.appendMessage(conv, Message{Role: RoleUser, Content: []Block{block}})
}

// appendMessage appends m to conv and records it as RecordAppend.
func (a *Agent) appendMessage(conv *Conversation, m Message) {
	conv.Messages = append(conv.Messages, m)
	a.record(RecordAppend, m)
}

// record passes a deep copy of m to the Recorder, if there is one.
func (a *Agent) record(kind RecordKind, m Message) {
	if a.Recorder != nil {
		a.Recorder.Record(kind, copyMessage(m))
	}
}

// streamAssistant sends req to the model and returns the assistant message the
// stream built. It returns the first error of the round; see Turn.
func (a *Agent) streamAssistant(ctx context.Context, req Request, emit func(Event)) (Message, error) {
	asm := &assembly{msg: Message{Role: RoleAssistant}}
	streamErr := a.Model.Stream(ctx, req, func(ev Event) {
		if asm.accept(ev) && emit != nil {
			emit(ev)
		}
	})
	if asm.err != nil {
		return Message{}, asm.err
	}
	if streamErr != nil {
		return Message{}, fmt.Errorf("agent: model stream: %w", streamErr)
	}
	if err := asm.finish(); err != nil {
		return Message{}, err
	}
	return asm.msg, nil
}

// invokeTools invokes calls sequentially, in order, and returns one result per
// call. It short-circuits on the first failing result: remaining calls are
// reported as cancelled, or as policy_denied when the failing result itself was
// policy_denied. A call to an unknown tool name is reported as
// tool_input_invalid and is counted as a hallucination, but it is not invoked.
//
// Cancellation is respected between invocations and while an invocation is in
// flight: the current call and all remaining calls are reported as cancelled.
// The returned error is the context error.
func (a *Agent) invokeTools(ctx context.Context, calls []ToolCall) ([]ResultBlock, bool, error) {
	results := make([]ResultBlock, len(calls))
	hasHallucination := false
	known := a.knownToolNames()

	for i, tc := range calls {
		select {
		case <-ctx.Done():
			results[i] = cancelledResult(tc.ID)
			for j := i + 1; j < len(calls); j++ {
				results[j] = cancelledResult(calls[j].ID)
			}
			return results, hasHallucination, ctx.Err()
		default:
		}

		if !known[tc.Name] {
			// Only unknown tool names consume the hallucination retry budget;
			// a known tool with invalid input does not. The spec addresses only
			// hallucinated names, so this is a chosen reading.
			hasHallucination = true
			results[i] = unknownToolResult(tc.ID, tc.Name, a.toolNamesList())
			for j := i + 1; j < len(calls); j++ {
				results[j] = cancelledResult(calls[j].ID)
			}
			return results, hasHallucination, nil
		}

		out := make(chan ToolResult, 1)
		go func(call ToolCall) {
			out <- a.Tools.Invoke(ctx, copyToolCall(call))
		}(tc)

		select {
		case <-ctx.Done():
			results[i] = cancelledResult(tc.ID)
			for j := i + 1; j < len(calls); j++ {
				results[j] = cancelledResult(calls[j].ID)
			}
			return results, hasHallucination, ctx.Err()
		case res := <-out:
			results[i] = ResultBlock{
				CallID:    tc.ID,
				Content:   res.Content,
				IsError:   !res.OK,
				ErrorKind: res.ErrorKind,
				Truncated: res.Truncated,
			}
			if !res.OK {
				for j := i + 1; j < len(calls); j++ {
					results[j] = shortCircuitResult(calls[j].ID, res.ErrorKind)
				}
				return results, hasHallucination, nil
			}
		}
	}

	return results, hasHallucination, nil
}

// validateCallIDs returns an error if any tool call has an empty id or if ids
// repeat within the message. Rejecting these cases makes "a result for every
// tool_use id" unambiguous.
func validateCallIDs(calls []ToolCall) error {
	seen := make(map[string]struct{}, len(calls))
	for _, tc := range calls {
		if tc.ID == "" {
			return fmt.Errorf("%w: tool call with empty id", ErrStreamProtocol)
		}
		if _, ok := seen[tc.ID]; ok {
			return fmt.Errorf("%w: duplicate tool call id %q", ErrStreamProtocol, tc.ID)
		}
		seen[tc.ID] = struct{}{}
	}
	return nil
}

// knownToolNames returns the set of tool names the agent considers valid, built
// from Tools.Describe.
func (a *Agent) knownToolNames() map[string]bool {
	specs := a.Tools.Describe()
	names := make(map[string]bool, len(specs))
	for _, s := range specs {
		names[s.Name] = true
	}
	return names
}

// toolNamesList returns the sorted list of tool names from Tools.Describe.
func (a *Agent) toolNamesList() []string {
	specs := a.Tools.Describe()
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	return names
}

// cancelledResult returns a result block for a call that did not run.
func cancelledResult(callID string) ResultBlock {
	return ResultBlock{
		CallID:    callID,
		Content:   "cancelled",
		IsError:   true,
		ErrorKind: toolErrCancelled,
	}
}

// shortCircuitResult returns the result block given to calls skipped after a
// sibling call failed. When the failing call was policy_denied, the remaining
// calls are also marked policy_denied; otherwise they are cancelled. This is a
// chosen reading: the spec requires a result for every skipped call, but does
// not prescribe which kind.
func shortCircuitResult(callID string, failedKind string) ResultBlock {
	if failedKind == toolErrPolicyDenied {
		return ResultBlock{
			CallID:    callID,
			Content:   "policy_denied",
			IsError:   true,
			ErrorKind: toolErrPolicyDenied,
		}
	}
	return cancelledResult(callID)
}

// unknownToolResult returns the result block for a call to a tool name that is
// not in the registry. The content lists the real tool names so the model can
// self-correct.
func unknownToolResult(callID, name string, available []string) ResultBlock {
	content := fmt.Sprintf("unknown tool %q; available tools are: %s", name, joinToolNames(available))
	return ResultBlock{
		CallID:    callID,
		Content:   content,
		IsError:   true,
		ErrorKind: toolErrInputInvalid,
	}
}

// joinToolNames formats a list of tool names the same way the M1 registry does.
func joinToolNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// toolCalls returns the tool calls of an assistant message, in block order.
func toolCalls(m Message) []ToolCall {
	var calls []ToolCall
	for _, b := range m.Content {
		if b.Kind == BlockToolCall && b.ToolCall != nil {
			calls = append(calls, *b.ToolCall)
		}
	}
	return calls
}

// resultMessage returns the user message that carries results, one block each.
func resultMessage(results []ResultBlock) Message {
	m := Message{Role: RoleUser, Content: make([]Block, len(results))}
	for i := range results {
		m.Content[i] = Block{Kind: BlockToolResult, Result: &results[i]}
	}
	return m
}

// assembly accumulates the events of one model stream into an assistant message.
type assembly struct {
	// msg is the assistant message built so far.
	msg Message
	// open is the thinking block still waiting for its EventThinkingDone, or nil.
	open *Thinking
	// done is true once EventMessageDone has been accepted.
	done bool
	// err is the first error of the stream. Once set, accept ignores every event.
	err error
}

// fail records err as the stream's error and reports the event as not accepted.
// The caller has already checked that err is the first.
func (s *assembly) fail(err error) bool {
	s.err = err
	return false
}

// accept takes one event and reports whether it was accepted, that is whether the
// caller should forward it. An event that violates the protocol sets s.err and is
// not accepted; so is every event once s.err is set, and every event once
// EventMessageDone has been accepted, an EventError included. An EventError that
// arrives before EventMessageDone and is the first error is accepted and sets
// s.err.
func (s *assembly) accept(ev Event) bool {
	if s.err != nil {
		return false
	}
	if s.done {
		return s.fail(fmt.Errorf("%w: event received after MessageDone", ErrStreamProtocol))
	}

	switch ev.Type {
	case EventTextDelta:
		if s.open != nil {
			return s.fail(fmt.Errorf("%w: text delta while thinking block is open", ErrStreamProtocol))
		}
		if n := len(s.msg.Content); n > 0 && s.msg.Content[n-1].Kind == BlockText {
			s.msg.Content[n-1].Text += ev.Text
		} else {
			s.msg.Content = append(s.msg.Content, Block{Kind: BlockText, Text: ev.Text})
		}

	case EventThinkingDelta:
		if s.open == nil {
			s.open = &Thinking{}
			s.msg.Content = append(s.msg.Content, Block{Kind: BlockThinking, Thinking: s.open})
		}
		s.open.Text += ev.Text

	case EventThinkingDone:
		if ev.Thinking == nil {
			return s.fail(fmt.Errorf("%w: EventThinkingDone with nil Thinking", ErrStreamProtocol))
		}
		if s.open == nil {
			s.open = &Thinking{}
			s.msg.Content = append(s.msg.Content, Block{Kind: BlockThinking, Thinking: s.open})
		}
		// The Done event's payload replaces everything the deltas built, so Text,
		// Signature, Redacted and Data all come from it.
		*s.open = *ev.Thinking
		// Mark the block so M4 compaction drops it instead of summarising it.
		s.open.DropOnSummary = true
		s.open = nil

	case EventToolCallEnd:
		if s.open != nil {
			return s.fail(fmt.Errorf("%w: tool call end while thinking block is open", ErrStreamProtocol))
		}
		if ev.ToolCall == nil {
			return s.fail(fmt.Errorf("%w: EventToolCallEnd with nil ToolCall", ErrStreamProtocol))
		}
		tc := copyToolCall(*ev.ToolCall)
		s.msg.Content = append(s.msg.Content, Block{Kind: BlockToolCall, ToolCall: &tc})

	case EventToolCallStart, EventToolCallDelta:
		// Not accumulated: the complete call arrives with EventToolCallEnd. The
		// event is only forwarded.

	case EventMessageDone:
		if s.open != nil {
			return s.fail(fmt.Errorf("%w: thinking block not closed at MessageDone", ErrStreamProtocol))
		}
		s.done = true

	case EventError:
		if ev.Err != nil {
			s.err = fmt.Errorf("agent: model stream: %w", ev.Err)
		} else {
			s.err = ErrStreamFailed
		}
	}
	return true
}

// finish checks a stream that ended without an error: it must have delivered
// EventMessageDone and at least one block.
func (s *assembly) finish() error {
	if !s.done {
		return fmt.Errorf("%w: no MessageDone event received", ErrStreamProtocol)
	}
	if len(s.msg.Content) == 0 {
		return fmt.Errorf("%w: assistant message has no content blocks", ErrStreamProtocol)
	}
	return nil
}

// cloneRaw returns a copy of in. A nil input stays nil; an empty one stays empty.
func cloneRaw(in json.RawMessage) json.RawMessage {
	return json.RawMessage(bytes.Clone(in))
}

// copyToolCall returns a copy of tc that shares no memory with it.
func copyToolCall(tc ToolCall) ToolCall {
	tc.Input = cloneRaw(tc.Input)
	return tc
}

// copyBlock returns a copy of b that shares no memory with it.
func copyBlock(b Block) Block {
	if b.Thinking != nil {
		t := *b.Thinking
		b.Thinking = &t
	}
	if b.ToolCall != nil {
		tc := copyToolCall(*b.ToolCall)
		b.ToolCall = &tc
	}
	if b.Result != nil {
		r := *b.Result
		b.Result = &r
	}
	return b
}

// copyMessage returns a copy of m that shares no memory with it.
func copyMessage(m Message) Message {
	if m.Content != nil {
		content := make([]Block, len(m.Content))
		for i, b := range m.Content {
			content[i] = copyBlock(b)
		}
		m.Content = content
	}
	return m
}

// copyMessages returns a copy of msgs that shares no memory with it. A nil slice
// stays nil.
func copyMessages(msgs []Message) []Message {
	if msgs == nil {
		return nil
	}
	out := make([]Message, len(msgs))
	for i, m := range msgs {
		out[i] = copyMessage(m)
	}
	return out
}
