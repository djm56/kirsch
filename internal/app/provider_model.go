// Package app wires the pieces together and routes between them.
//
// provider_model.go adapts a provider.Provider to the agent.Model interface so
// that the agent turn machinery can drive a real HTTP endpoint. It is wiring,
// not business logic: it only translates between the two type systems.
package app

import (
	"context"
	"strings"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/provider"
)

// providerModel adapts provider.Provider to agent.Model.
type providerModel struct {
	p             provider.Provider
	model         string
	promptCaching bool
	thinking      provider.ThinkingLevel
}

// Info returns capability information for the configured model.
func (pm *providerModel) Info() agent.ModelInfo {
	info := provider.LookupModel(pm.model)
	return agent.ModelInfo{
		ContextWindow: info.ContextWindow,
		MaxOutput:     info.MaxOutput,
	}
}

// Stream translates an agent.Request into a provider.Request, forwards events
// to the provider, and maps each provider.StreamEvent back into an agent.Event.
func (pm *providerModel) Stream(ctx context.Context, req agent.Request, onEvent func(agent.Event)) error {
	preq := provider.Request{
		Model:     pm.model,
		MaxTokens: req.MaxTokens,
		Thinking:  pm.thinking,
	}

	if req.System != "" {
		preq.System = []provider.SystemBlock{
			{Text: req.System, Cache: pm.promptCaching},
		}
	}

	if len(req.Tools) > 0 {
		preq.Tools = make([]provider.ToolDef, len(req.Tools))
		for i, t := range req.Tools {
			preq.Tools[i] = provider.ToolDef{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			}
		}
	}

	if len(req.Messages) > 0 {
		preq.Messages = make([]provider.Message, len(req.Messages))
		for i, m := range req.Messages {
			preq.Messages[i] = provider.Message{
				Role:    provider.Role(m.Role),
				Content: agentBlocksToProvider(m.Content),
			}
		}
	}

	return pm.p.Stream(ctx, preq, func(ev provider.StreamEvent) {
		onEvent(providerEventToAgent(ev))
	})
}

// agentBlocksToProvider maps agent blocks to provider-neutral blocks.
func agentBlocksToProvider(in []agent.Block) []provider.Block {
	out := make([]provider.Block, len(in))
	for i, b := range in {
		switch b.Kind {
		case agent.BlockText:
			out[i] = provider.Block{Kind: provider.BlockText, Text: b.Text}
		case agent.BlockThinking:
			if b.Thinking != nil {
				out[i] = provider.Block{
					Kind: provider.BlockThinking,
					Thinking: &provider.Thinking{
						Text:      b.Thinking.Text,
						Signature: b.Thinking.Signature,
						Redacted:  b.Thinking.Redacted,
						Data:      b.Thinking.Data,
					},
				}
			}
		case agent.BlockToolCall:
			if b.ToolCall != nil {
				out[i] = provider.Block{
					Kind: provider.BlockToolCall,
					ToolCall: &provider.ToolCall{
						ID:    b.ToolCall.ID,
						Name:  b.ToolCall.Name,
						Input: b.ToolCall.Input,
					},
				}
			}
		case agent.BlockToolResult:
			if b.Result != nil {
				out[i] = provider.Block{
					Kind: provider.BlockToolResult,
					ToolResult: &provider.ToolResult{
						CallID:  b.Result.CallID,
						Content: b.Result.Content,
						IsError: b.Result.IsError,
					},
				}
			}
		}
	}
	return out
}

// providerEventToAgent maps provider stream events back to agent events.
func providerEventToAgent(ev provider.StreamEvent) agent.Event {
	ae := agent.Event{
		Type:       agent.EventType(ev.Type),
		Text:       ev.Text,
		StopReason: string(ev.StopReason),
	}

	switch ev.Type {
	case provider.EventTextDelta, provider.EventThinkingDelta:
		// Text is already copied above; nothing else to map.
	case provider.EventThinkingDone:
		if ev.Thinking != nil {
			ae.Thinking = &agent.Thinking{
				Text:      ev.Thinking.Text,
				Signature: ev.Thinking.Signature,
				Redacted:  ev.Thinking.Redacted,
				Data:      ev.Thinking.Data,
			}
		}
	case provider.EventToolCallStart, provider.EventToolCallEnd, provider.EventToolCallDelta:
		if ev.ToolCall != nil {
			ae.ToolCall = &agent.ToolCall{
				ID:    ev.ToolCall.ID,
				Name:  ev.ToolCall.Name,
				Input: ev.ToolCall.Input,
			}
		}
	case provider.EventMessageDone:
		ae.StopReason = string(ev.StopReason)
		if ev.Usage != nil {
			ae.Usage = &agent.Usage{
				InputTokens:      ev.Usage.InputTokens,
				OutputTokens:     ev.Usage.OutputTokens,
				CacheReadTokens:  ev.Usage.CacheReadTokens,
				CacheWriteTokens: ev.Usage.CacheWriteTokens,
			}
		}
	case provider.EventError:
		ae.Err = ev.Err
	}

	return ae
}

// parseThinkingLevel converts a configuration thinking value to a provider level.
func parseThinkingLevel(s string) provider.ThinkingLevel {
	switch strings.ToLower(s) {
	case "off":
		return provider.ThinkingOff
	case "low":
		return provider.ThinkingLow
	case "medium":
		return provider.ThinkingMedium
	case "high":
		return provider.ThinkingHigh
	default:
		return provider.ThinkingOff
	}
}
