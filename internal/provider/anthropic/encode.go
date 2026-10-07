package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/djm56/kirsch/internal/provider"
)

// EncodeOptions carries per-endpoint settings the neutral Request does not.
type EncodeOptions struct {
	// PromptCaching controls whether cache_control is placed on SystemBlocks with Cache set.
	PromptCaching bool
}

// EncodeRequest renders req as a streaming Messages API request body.
// It validates the request first and encodes nothing when it is refused.
//
// Refusals (each returns an error prefixed "EncodeRequest:"):
//   - model is empty
//   - max_tokens is not greater than 0
//   - there are no messages
//   - a message role is neither user nor assistant
//   - a message has no content blocks
//   - a text block is empty
//   - a thinking block has no Thinking payload
//   - a tool call block has no ToolCall payload, or its non-empty input is not
//     a JSON object (invalid JSON, null, array, string and number all refuse)
//   - a tool result block has no ToolResult payload
//   - a block has an unknown kind
//   - a tool definition has no name, no input_schema, or an input_schema that
//     is not a JSON object
//
// PromptCaching in opts adds cache_control only to system blocks that set Cache.
func EncodeRequest(req provider.Request, opts EncodeOptions) ([]byte, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	return encodeMessages(req, opts)
}

// validateRequest checks required fields and structure.
func validateRequest(req provider.Request) error {
	if req.Model == "" {
		return fmt.Errorf("EncodeRequest: model is required")
	}
	if req.MaxTokens <= 0 {
		return fmt.Errorf("EncodeRequest: max_tokens must be greater than 0")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("EncodeRequest: messages is required")
	}

	for _, msg := range req.Messages {
		if msg.Role != provider.RoleUser && msg.Role != provider.RoleAssistant {
			return fmt.Errorf("EncodeRequest: invalid role %q", msg.Role)
		}
		if len(msg.Content) == 0 {
			return fmt.Errorf("EncodeRequest: message content cannot be empty")
		}
		for _, block := range msg.Content {
			if err := validateBlock(block); err != nil {
				return err
			}
		}
	}

	for _, tool := range req.Tools {
		if tool.Name == "" {
			return fmt.Errorf("EncodeRequest: tool name is required")
		}
		if len(tool.InputSchema) == 0 {
			return fmt.Errorf("EncodeRequest: tool input_schema is required")
		}
		if !json.Valid(tool.InputSchema) {
			return fmt.Errorf("EncodeRequest: tool input_schema is not valid JSON")
		}
		// Check for null or non-object types
		trimmed := strings.TrimSpace(string(tool.InputSchema))
		if trimmed == "null" {
			return fmt.Errorf("EncodeRequest: tool %q input_schema must be a JSON object", tool.Name)
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(tool.InputSchema, &obj); err != nil {
			return fmt.Errorf("EncodeRequest: tool %q input_schema must be a JSON object", tool.Name)
		}
	}
	return nil
}

// encodeMessages builds the request map and encodes it as JSON.
func encodeMessages(req provider.Request, opts EncodeOptions) ([]byte, error) {
	reqMap := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"stream":     true,
	}

	if len(req.System) > 0 {
		system := make([]map[string]interface{}, len(req.System))
		for i, block := range req.System {
			system[i] = map[string]interface{}{
				"type": "text",
				"text": block.Text,
			}
			if opts.PromptCaching && block.Cache {
				system[i]["cache_control"] = map[string]string{
					"type": "ephemeral",
				}
			}
		}
		reqMap["system"] = system
	}

	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{"role": string(msg.Role)}
		if len(msg.Content) == 1 && msg.Content[0].Kind == provider.BlockText {
			messages[i]["content"] = msg.Content[0].Text
		} else {
			content := make([]interface{}, len(msg.Content))
			for j, block := range msg.Content {
				content[j] = encodeBlock(block)
			}
			messages[i]["content"] = content
		}
	}
	reqMap["messages"] = messages

	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]interface{}{
				"name":         tool.Name,
				"input_schema": tool.InputSchema,
			}
			if tool.Description != "" {
				tools[i]["description"] = tool.Description
			}
		}
		reqMap["tools"] = tools
	}

	encodeThinkingFields(reqMap, req.Thinking)

	return json.Marshal(reqMap)
}

// encodeThinkingFields adds the top-level thinking and output_config fields for
// the Anthropic Messages API. The mapping was decided against the live docs:
//   - off  -> thinking.type="between_tools", output_config.effort="high"
//   - low  -> thinking.type="adaptive",      output_config.effort="low"
//   - medium -> thinking.type="adaptive",    output_config.effort="medium"
//   - high -> thinking.type="adaptive",      output_config.effort="high"
//
// "between_tools" is used for off because Claude Sonnet 5.5 rejects
// thinking.type="disabled" and thinking.type="enabled"+budget_tokens; see
// https://platform.claude.com/docs/en/build-with-claude/thinking. Effort is set
// via output_config.effort; see
// https://platform.claude.com/docs/en/build-with-claude/effort.
func encodeThinkingFields(reqMap map[string]interface{}, lvl provider.ThinkingLevel) {
	switch lvl {
	case provider.ThinkingOff:
		reqMap["thinking"] = map[string]string{"type": "between_tools"}
		reqMap["output_config"] = map[string]string{"effort": "high"}
	case provider.ThinkingLow:
		reqMap["thinking"] = map[string]string{"type": "adaptive"}
		reqMap["output_config"] = map[string]string{"effort": "low"}
	case provider.ThinkingMedium:
		reqMap["thinking"] = map[string]string{"type": "adaptive"}
		reqMap["output_config"] = map[string]string{"effort": "medium"}
	case provider.ThinkingHigh:
		reqMap["thinking"] = map[string]string{"type": "adaptive"}
		reqMap["output_config"] = map[string]string{"effort": "high"}
	}
}

// validateBlock checks that a block has the required payload pointers and valid structure.
func validateBlock(block provider.Block) error {
	switch block.Kind {
	case provider.BlockText:
		if block.Text == "" {
			return fmt.Errorf("EncodeRequest: BlockText cannot be empty")
		}
		return nil
	case provider.BlockThinking:
		if block.Thinking == nil {
			return fmt.Errorf("EncodeRequest: BlockThinking requires Thinking payload")
		}
		return nil
	case provider.BlockToolCall:
		if block.ToolCall == nil {
			return fmt.Errorf("EncodeRequest: BlockToolCall requires ToolCall payload")
		}
		if len(block.ToolCall.Input) > 0 {
			if !json.Valid(block.ToolCall.Input) {
				return fmt.Errorf("EncodeRequest: tool call %q input must be a JSON object", block.ToolCall.ID)
			}
			trimmed := strings.TrimSpace(string(block.ToolCall.Input))
			if trimmed == "null" {
				return fmt.Errorf("EncodeRequest: tool call %q input must be a JSON object", block.ToolCall.ID)
			}
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(block.ToolCall.Input, &obj); err != nil {
				return fmt.Errorf("EncodeRequest: tool call %q input must be a JSON object", block.ToolCall.ID)
			}
		}
		return nil
	case provider.BlockToolResult:
		if block.ToolResult == nil {
			return fmt.Errorf("EncodeRequest: BlockToolResult requires ToolResult payload")
		}
		return nil
	default:
		return fmt.Errorf("EncodeRequest: unknown block kind %d", block.Kind)
	}
}

// encodeBlock encodes a provider Block to its wire format for the Messages API.
func encodeBlock(block provider.Block) interface{} {
	switch block.Kind {
	case provider.BlockText:
		return map[string]interface{}{
			"type": "text",
			"text": block.Text,
		}
	case provider.BlockThinking:
		if block.Thinking.Redacted {
			return map[string]interface{}{
				"type": "redacted_thinking",
				"data": block.Thinking.Data,
			}
		}
		return map[string]interface{}{
			"type":      "thinking",
			"thinking":  block.Thinking.Text,
			"signature": block.Thinking.Signature,
		}
	case provider.BlockToolCall:
		tc := block.ToolCall
		input := json.RawMessage("{}")
		if len(tc.Input) > 0 {
			input = tc.Input
		}
		return map[string]interface{}{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": input,
		}
	case provider.BlockToolResult:
		tr := block.ToolResult
		result := map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": tr.CallID,
			"content":     tr.Content,
		}
		if tr.IsError {
			result["is_error"] = true
		}
		return result
	default:
		return nil
	}
}
