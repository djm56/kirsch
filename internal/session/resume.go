package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/djm56/kirsch/internal/agent"
)

// ErrMalformedSession indicates that a session log line is structurally valid
// JSON but violates the semantic rules required for reconstruction. It is
// distinct from a torn final line, which is treated as a repair boundary and
// silently truncates the log.
var ErrMalformedSession = errors.New("session: malformed session log")

// Transcript is the user-visible reconstruction of a session.
//
// It is intentionally independent of the TUI package: it uses agent.Block so
// the application layer can map it to rendered state without internal/session
// importing presentation code. See architecture.md §6 for the distinction
// between transcript (what the user saw), conversation (what the model sees),
// and the session log (every event).
type Transcript struct {
	Messages []TranscriptMessage
}

// TranscriptMessage is one entry in the transcript.
type TranscriptMessage struct {
	Role    agent.Role
	Content []agent.Block
}

// Resume reconstructs the provider conversation and a user-visible transcript
// from the JSONL session log at path.
//
// Reconstruction follows the locked rule in plan §7: only user.message,
// assistant.message, assistant.thinking and tool.completed events contribute.
// assistant.delta events are ignored. compaction.applied events replace the
// indicated message range in the conversation only; the transcript is left
// untouched so it continues to show everything the user saw.
//
// A torn or invalid JSON line is treated as the end of the log, matching the
// repair-on-load behaviour of Store.New. Any other semantic problem in a
// recognised event type returns ErrMalformedSession.
func Resume(path string) (*agent.Conversation, Transcript, error) {
	// #nosec G304 -- path is caller-supplied; containment is the caller's
	// responsibility, matching Store.New and other file-boundary helpers.
	f, err := os.Open(path)
	if err != nil {
		return nil, Transcript{}, err
	}
	defer func() { _ = f.Close() }()

	var (
		conv    []agent.Message
		trans   []TranscriptMessage
		pending map[string]struct{} // expected tool call ids from last assistant.message
		scanner = bufio.NewScanner(f)
	)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		// A line that is not valid JSON is a torn line; stop here. This mirrors
		// repairSession's behaviour of truncating to the last good line.
		if !json.Valid(line) {
			break
		}

		var base baseEvent
		if err := json.Unmarshal(line, &base); err != nil {
			break
		}

		switch base.Type {
		case eventTypeUserMessage:
			var e userMessageEvent
			if err := json.Unmarshal(line, &e); err != nil {
				return nil, Transcript{}, fmt.Errorf("%w: invalid user.message: %w", ErrMalformedSession, err)
			}
			conv = mergeUserText(conv, e.Content)
			trans = appendTranscriptText(trans, e.Content)

		case eventTypeAssistantThinking:
			var e assistantThinkingEvent
			if err := json.Unmarshal(line, &e); err != nil {
				return nil, Transcript{}, fmt.Errorf("%w: invalid assistant.thinking: %w", ErrMalformedSession, err)
			}
			m := agent.Message{
				Role: agent.RoleAssistant,
				Content: []agent.Block{{
					Kind: agent.BlockThinking,
					Thinking: &agent.Thinking{
						Text:          e.Content,
						Signature:     e.Signature,
						DropOnSummary: true,
					},
				}},
			}
			conv = append(conv, m)
			trans = appendTranscriptMessage(trans, m)

		case eventTypeAssistantMessage:
			var e assistantMessageEvent
			if err := json.Unmarshal(line, &e); err != nil {
				return nil, Transcript{}, fmt.Errorf("%w: invalid assistant.message: %w", ErrMalformedSession, err)
			}
			if e.Content == "" && len(e.ToolCalls) == 0 {
				return nil, Transcript{}, fmt.Errorf("%w: assistant.message has no content or tool_calls", ErrMalformedSession)
			}
			m := buildAssistantMessage(e)
			pending = make(map[string]struct{}, len(e.ToolCalls))
			for _, tc := range e.ToolCalls {
				pending[tc.ID] = struct{}{}
			}
			conv = append(conv, m)
			trans = appendTranscriptMessage(trans, m)

		case eventTypeToolCompleted:
			var e toolCompletedEvent
			if err := json.Unmarshal(line, &e); err != nil {
				return nil, Transcript{}, fmt.Errorf("%w: invalid tool.completed: %w", ErrMalformedSession, err)
			}
			if e.ID == "" {
				return nil, Transcript{}, fmt.Errorf("%w: tool.completed has empty id", ErrMalformedSession)
			}
			if _, ok := pending[e.ID]; !ok {
				return nil, Transcript{}, fmt.Errorf("%w: tool.completed id %q does not match a pending tool call", ErrMalformedSession, e.ID)
			}
			delete(pending, e.ID)
			result := agent.ResultBlock{
				CallID:    e.ID,
				Content:   e.Content,
				IsError:   !e.OK,
				Truncated: e.Truncated,
			}
			conv = mergeUserResult(conv, result)
			trans = appendTranscriptResult(trans, result)

		case eventTypeCompactionApplied:
			var e compactionAppliedEvent
			if err := json.Unmarshal(line, &e); err != nil {
				return nil, Transcript{}, fmt.Errorf("%w: invalid compaction.applied: %w", ErrMalformedSession, err)
			}
			conv, err = applyCompaction(conv, e.ReplacedFrom, e.ReplacedTo, e.Summary)
			if err != nil {
				return nil, Transcript{}, err
			}

		case eventTypeAssistantDelta:
			// UI-replay data; never a source of truth for reconstruction.
			continue

		default:
			// Ignore every other event type.
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, Transcript{}, err
	}

	return &agent.Conversation{Messages: conv}, Transcript{Messages: trans}, nil
}

// buildAssistantMessage converts an assistant.message event into an agent
// message, preserving text and tool-call blocks.
func buildAssistantMessage(e assistantMessageEvent) agent.Message {
	var blocks []agent.Block
	if e.Content != "" {
		blocks = append(blocks, agent.Block{Kind: agent.BlockText, Text: e.Content})
	}
	for _, tc := range e.ToolCalls {
		blocks = append(blocks, agent.Block{
			Kind: agent.BlockToolCall,
			ToolCall: &agent.ToolCall{
				ID:    tc.ID,
				Name:  tc.Tool,
				Input: cloneRaw(tc.Input),
			},
		})
	}
	return agent.Message{Role: agent.RoleAssistant, Content: blocks}
}

// mergeUserText appends a user text block, merging with the previous user
// message when one exists. This reproduces the RecordMerge semantics the
// loader is documented to provide.
func mergeUserText(msgs []agent.Message, text string) []agent.Message {
	if n := len(msgs); n > 0 && msgs[n-1].Role == agent.RoleUser {
		msgs[n-1].Content = append(msgs[n-1].Content, agent.Block{Kind: agent.BlockText, Text: text})
		return msgs
	}
	return append(msgs, agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	})
}

// mergeUserResult appends a tool result block, merging with the previous user
// message when one exists so that consecutive tool.completed events for the
// same turn become a single user message.
func mergeUserResult(msgs []agent.Message, result agent.ResultBlock) []agent.Message {
	if n := len(msgs); n > 0 && msgs[n-1].Role == agent.RoleUser {
		msgs[n-1].Content = append(msgs[n-1].Content, agent.Block{Kind: agent.BlockToolResult, Result: &result})
		return msgs
	}
	return append(msgs, agent.Message{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockToolResult, Result: &result}},
	})
}

// appendTranscriptText appends a user text entry to the transcript, merging
// with the previous transcript entry when it is also a user message.
func appendTranscriptText(msgs []TranscriptMessage, text string) []TranscriptMessage {
	if n := len(msgs); n > 0 && msgs[n-1].Role == agent.RoleUser {
		msgs[n-1].Content = append(msgs[n-1].Content, agent.Block{Kind: agent.BlockText, Text: text})
		return msgs
	}
	return append(msgs, TranscriptMessage{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockText, Text: text}},
	})
}

// appendTranscriptMessage appends a whole assistant message to the transcript,
// using a deep copy so the transcript and conversation remain independent.
func appendTranscriptMessage(msgs []TranscriptMessage, m agent.Message) []TranscriptMessage {
	return append(msgs, TranscriptMessage{Role: m.Role, Content: cloneBlocks(m.Content)})
}

// appendTranscriptResult appends a tool result block to the transcript,
// merging with the previous transcript entry when it is also a user message.
func appendTranscriptResult(msgs []TranscriptMessage, result agent.ResultBlock) []TranscriptMessage {
	if n := len(msgs); n > 0 && msgs[n-1].Role == agent.RoleUser {
		msgs[n-1].Content = append(msgs[n-1].Content, agent.Block{Kind: agent.BlockToolResult, Result: &result})
		return msgs
	}
	return append(msgs, TranscriptMessage{
		Role:    agent.RoleUser,
		Content: []agent.Block{{Kind: agent.BlockToolResult, Result: &result}},
	})
}

// applyCompaction replaces conversation messages in [from, to] (inclusive)
// with a single assistant text message carrying the summary.
func applyCompaction(msgs []agent.Message, from, to int, summary string) ([]agent.Message, error) {
	if from < 0 || to < from || from >= len(msgs) || to >= len(msgs) {
		return nil, fmt.Errorf("%w: compaction.applied range [%d,%d] invalid for %d messages", ErrMalformedSession, from, to, len(msgs))
	}
	replacement := agent.Message{
		Role:    agent.RoleAssistant,
		Content: []agent.Block{{Kind: agent.BlockText, Text: summary}},
	}
	out := append(msgs[:from:from], replacement)
	if to+1 < len(msgs) {
		out = append(out, msgs[to+1:]...)
	}
	return out, nil
}

// cloneBlocks returns a deep copy of blocks.
func cloneBlocks(blocks []agent.Block) []agent.Block {
	if blocks == nil {
		return nil
	}
	out := make([]agent.Block, len(blocks))
	for i, b := range blocks {
		out[i] = cloneBlock(b)
	}
	return out
}

// cloneBlock returns a deep copy of a single block.
func cloneBlock(b agent.Block) agent.Block {
	if b.Thinking != nil {
		t := *b.Thinking
		b.Thinking = &t
	}
	if b.ToolCall != nil {
		tc := *b.ToolCall
		tc.Input = cloneRaw(tc.Input)
		b.ToolCall = &tc
	}
	if b.Result != nil {
		r := *b.Result
		b.Result = &r
	}
	return b
}

// cloneRaw returns a copy of in. A nil input stays nil; an empty one stays empty.
func cloneRaw(in json.RawMessage) json.RawMessage {
	return json.RawMessage(bytes.Clone(in))
}
