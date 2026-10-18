package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/djm56/kirsch/internal/agent"
)

// Sentinel errors for compaction.
var (
	ErrNothingToCompact = errors.New("nothing to compact")
	ErrCompactionFailed = errors.New("compaction failed")
)

// compactionSystemPrompt is the system prompt for the compaction model call.
// It asks for a structured summary with specific sections.
const compactionSystemPrompt = `You are summarizing a conversation for context compression. Produce a structured summary of the messages provided, covering:

**Objective**: The primary task or question being worked on.

**Files inspected**: Any files that were read or examined during the work.

**Key findings**: Important discoveries, patterns, or results from analysis or tool execution.

**Files changed**: Any files that were modified or created.

**Commands run**: Commands executed with their purposes and outcomes, including PASS or FAIL result for each.

**Limitations**: Any constraints, deferred work, or unresolved issues.

**Open questions**: Any unresolved issues or next steps needed.

Be concise but complete. Format each section as a heading followed by bullet points or short paragraphs.

If a final report already appears in the summarized messages, preserve the most recent final report verbatim. Never omit or paraphrase an existing final report.`

// Compaction represents the result of a compaction operation.
type Compaction struct {
	From     int
	To       int
	Summary  string
	Messages []agent.Message
	Usage    agent.Usage // Token usage for the summary model call
}

// PlanCompaction returns the inclusive message-index range to replace for compaction.
//
// A "turn" starts at a user message containing a BlockText block (a user prompt).
// A user message containing only tool results continues the current turn.
//
// The function returns:
//   - from: index 1 (keeps index 0 verbatim, compaction starts from index 1)
//   - to: the index just before the first message of the 4th-from-last turn
//   - ok: false when there is nothing to compact (<6 turns or empty range)
func PlanCompaction(msgs []agent.Message) (from, to int, ok bool) {
	if len(msgs) == 0 {
		return 0, 0, false
	}

	// Identify turn boundaries: a turn starts with a user message containing text.
	// Count turns and record their starting indices.
	var turnStarts []int
	for i, m := range msgs {
		if m.Role == agent.RoleUser {
			// Check if this user message has a text block (starts a new turn).
			hasText := false
			for _, b := range m.Content {
				if b.Kind == agent.BlockText {
					hasText = true
					break
				}
			}
			if hasText {
				turnStarts = append(turnStarts, i)
			}
		}
	}

	// Fewer than 6 turns: report not ok. From 6 turns up, message 0 and the
	// last 4 turns are kept and everything between them is compacted.
	if len(turnStarts) < 6 {
		return 0, 0, false
	}

	// from = 1: message 0 is outside the compacted range; compaction starts at index 1.
	from = 1

	// to: index just before the first message of the 4th-from-last turn.
	// We want to keep the last 4 turns verbatim.
	// 4th-from-last turn starts at turnStarts[len(turnStarts)-4].
	toTurnIdx := len(turnStarts) - 4
	to = turnStarts[toTurnIdx] - 1

	// Ensure the range is valid (from <= to).
	if from > to {
		return 0, 0, false
	}

	return from, to, true
}

// Compact makes one model call that summarizes the messages in the range
// PlanCompaction selects. It returns a Compaction carrying that range, the
// summary text, and the conversation with the range replaced by one assistant
// message holding the summary.
//
// Usage is read from the EventMessageDone the model emits. On every error
// return after the model call began, the returned Compaction is non-nil and
// carries only that Usage, so a caller can record billable tokens for a
// compaction it does not apply; its From, To, Summary and Messages are zero.
// ErrNothingToCompact returns a nil Compaction because no call was made.
//
// It returns ErrNothingToCompact, without calling the model, when
// PlanCompaction finds fewer than 6 turns. Every other failure (model error,
// EventError, cancelled context, empty or whitespace-only summary) wraps
// ErrCompactionFailed. msgs is never modified, and the returned Messages share
// no slices or pointers with it.
//
// maxTokens is the output cap on the summary request only. Compact does not
// measure the compacted result against any budget. All thinking blocks are
// dropped from the summary request.
func Compact(ctx context.Context, m agent.Model, msgs []agent.Message, maxTokens int) (*Compaction, error) {
	from, to, ok := PlanCompaction(msgs)
	if !ok {
		return nil, ErrNothingToCompact
	}

	// Extract the messages to compact (the middle range).
	middleMessages := msgs[from : to+1]

	// Render the middle range to text. All thinking blocks are dropped.
	summaryText := renderMessagesForSummary(middleMessages)

	// Build the summary request.
	req := agent.Request{
		System: compactionSystemPrompt,
		Messages: []agent.Message{
			{
				Role: agent.RoleUser,
				Content: []agent.Block{
					{
						Kind: agent.BlockText,
						Text: fmt.Sprintf("Summarize this work session:\n\n%s", summaryText),
					},
				},
			},
		},
		MaxTokens: maxTokens,
	}

	// Collect the summary text from the model's response.
	// Usage is captured even if later errors occur (stream, cancellation, or validation).
	var summaryBuf strings.Builder
	var streamErr error
	var usage agent.Usage

	onEvent := func(ev agent.Event) {
		switch ev.Type {
		case agent.EventTextDelta:
			summaryBuf.WriteString(ev.Text)
		case agent.EventError:
			streamErr = ev.Err
		case agent.EventMessageDone:
			// The model adapter attaches token counts to EventMessageDone;
			// it never emits agent.EventUsage (the agent does, after the
			// stream), so this is the only place usage can arrive from.
			if ev.Usage != nil {
				usage = *ev.Usage
			}
		default:
			// Other event types (thinking, tool calls, etc.) are not relevant
			// for the summary request and are ignored.
		}
	}

	err := m.Stream(ctx, req, onEvent)
	if err != nil {
		return &Compaction{Usage: usage}, fmt.Errorf("%w: %w", ErrCompactionFailed, err)
	}

	// Check for cancellation after Stream returns, even if it returned nil error.
	if ctx.Err() != nil {
		return &Compaction{Usage: usage}, fmt.Errorf("%w: %w", ErrCompactionFailed, ctx.Err())
	}

	if streamErr != nil {
		return &Compaction{Usage: usage}, fmt.Errorf("%w: %w", ErrCompactionFailed, streamErr)
	}

	// Reject whitespace-only summaries.
	summary := summaryBuf.String()
	if strings.TrimSpace(summary) == "" {
		return &Compaction{Usage: usage}, fmt.Errorf("%w: empty or whitespace-only summary", ErrCompactionFailed)
	}

	// Apply the compaction to create the new message slice.
	// Deep-copy only the messages outside [from, to]: applyCompaction replaces
	// that range, so the entries there stay zero-valued and are never read.
	msgsCopy := make([]agent.Message, len(msgs))
	for i := range msgs {
		if i >= from && i <= to {
			continue
		}
		msgsCopy[i] = agent.Message{
			Role:    msgs[i].Role,
			Content: cloneBlocks(msgs[i].Content),
		}
	}

	newMsgs, err := applyCompaction(msgsCopy, from, to, summary)
	if err != nil {
		return &Compaction{Usage: usage}, fmt.Errorf("%w: %w", ErrCompactionFailed, err)
	}

	return &Compaction{
		From:     from,
		To:       to,
		Summary:  summary,
		Messages: newMsgs,
		Usage:    usage,
	}, nil
}

// renderMessagesForSummary renders the middle messages to text for the summary request.
// All thinking blocks are dropped from the summary input (spec 6.5).
func renderMessagesForSummary(msgs []agent.Message) string {
	var sb strings.Builder

	for i, m := range msgs {
		fmt.Fprintf(&sb, "\n--- Message %d (%s) ---\n", i, m.Role)

		for _, b := range m.Content {
			switch b.Kind {
			case agent.BlockText:
				sb.WriteString(b.Text)
				sb.WriteString("\n")

			case agent.BlockThinking:
				// Drop all thinking blocks from the summary input.
				continue

			case agent.BlockToolCall:
				if b.ToolCall != nil {
					fmt.Fprintf(&sb, "[Tool Call: %s]\nInput: %s\n", b.ToolCall.Name, string(b.ToolCall.Input))
				}

			case agent.BlockToolResult:
				if b.Result != nil {
					fmt.Fprintf(&sb, "[Tool Result for %s]\n", b.Result.CallID)
					if b.Result.IsError {
						sb.WriteString("Error: ")
					}
					sb.WriteString(b.Result.Content)
					sb.WriteString("\n")
				}
			}
		}
	}

	return sb.String()
}
