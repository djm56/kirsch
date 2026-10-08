package app

import (
	"github.com/djm56/kirsch/internal/agent"
)

// outputReserve is the headroom reserved for the model's response.
// plan §6.3.
const outputReserve = 4096

// estimateTokens approximates the token count of the request that will be built
// from the system prompt, the user's text, the current conversation, and the
// tool definitions. It uses the plan's chars/4 rule with no tokenizer.
func estimateTokens(system, userText string, conv *agent.Conversation, tools agent.Tools) int {
	if tools == nil {
		return 0
	}
	total := len(system) + len(userText)
	for _, msg := range conv.Messages {
		total += messageChars(msg)
	}
	for _, spec := range tools.Describe() {
		total += len(spec.Name) + len(spec.Description) + len(spec.InputSchema)
	}
	return total / 4
}

// messageChars returns the raw character count of every textual field in a
// message. It deliberately does not recurse deeply into JSON; the estimate is
// intentionally rough.
func messageChars(msg agent.Message) int {
	n := 0
	for _, b := range msg.Content {
		switch b.Kind {
		case agent.BlockText:
			n += len(b.Text)
		case agent.BlockThinking:
			if b.Thinking != nil {
				n += len(b.Thinking.Text) + len(b.Thinking.Signature) + len(b.Thinking.Data)
			}
		case agent.BlockToolCall:
			if b.ToolCall != nil {
				n += len(b.ToolCall.ID) + len(b.ToolCall.Name) + len(b.ToolCall.Input)
			}
		case agent.BlockToolResult:
			if b.Result != nil {
				n += len(b.Result.CallID) + len(b.Result.Content) + len(b.Result.ErrorKind)
			}
		}
	}
	return n
}

// compactPercent returns the estimated budget consumption as a percentage of
// (contextWindow - outputReserve). Values above 100 mean the request exceeds
// the budget; the caller surfaces context_overflow instead of sending.
func compactPercent(estimated, contextWindow int) int {
	budget := contextWindow - outputReserve
	if budget <= 0 || estimated < 0 {
		return 0
	}
	return estimated * 100 / budget
}
