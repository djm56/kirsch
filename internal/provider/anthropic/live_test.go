//go:build live

package anthropic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
)

// TestLiveOpencodeSmoke sends one streamed, text-only request to the opencode
// endpoint through Kirsch's own adapter and checks that it completes and
// reports usage. It is compiled only with -tags live and is run by hand
// (npm run test:live); go test ./... and CI never build it. The key is read
// from the environment by config's own lookup and is never printed. The model
// name defaults to the opencode endpoint's configured default model and can be
// overridden by the KIRSCH_LIVE_MODEL environment variable, which is useful
// when the default model is disabled on the upstream endpoint. If
// KIRSCH_LIVE_MODEL is set but invalid (empty or containing whitespace or
// control characters), the test fails with a validation error before any
// network request is made.
func TestLiveOpencodeSmoke(t *testing.T) {
	ep := config.Defaults().Provider.Endpoints["opencode"]
	key, source := ep.ResolveKey(os.Getenv)
	if key == "" {
		t.Skipf("set KIRSCH_%s or %s to run the live smoke test", ep.APIKeyEnv, ep.APIKeyEnv)
	}
	model, modelSource, err := liveModel(os.LookupEnv, ep.Model)
	if err != nil {
		t.Fatalf("invalid model override: %v", err)
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	c, err := New(Options{
		Endpoint: "opencode", BaseURL: ep.BaseURL, Auth: ep.Auth, APIKey: key, KeyEnv: ep.APIKeyEnv,
		PromptCaching: ep.PromptCaching, Version: "smoke", SessionID: hex.EncodeToString(b[:]),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req := provider.Request{
		Model: model, MaxTokens: 256,
		Messages: []provider.Message{userText("Reply with the single word: pong")},
	}
	var text strings.Builder
	var done *provider.StreamEvent
	err = c.Stream(ctx, req, func(e provider.StreamEvent) {
		switch e.Type {
		case provider.EventTextDelta:
			text.WriteString(e.Text)
		case provider.EventMessageDone:
			ev := e
			done = &ev
		default:
			// Other events are not needed for the smoke check.
		}
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if done == nil || done.Usage == nil || done.Usage.InputTokens <= 0 || done.Usage.OutputTokens <= 0 {
		t.Fatalf("no usage reported: %+v", done)
	}
	if strings.TrimSpace(text.String()) == "" {
		t.Fatal("no text streamed")
	}
	t.Logf("endpoint=opencode model=%s model_source=%s key_source=%s text_bytes=%d stop=%s usage=%+v",
		model, modelSource, source, text.Len(), done.StopReason, *done.Usage)
}

// TestLiveThinkingRoundTrip sends a two-turn conversation to the live opencode
// endpoint and checks that (1) a signed thinking block returned in the first
// response can be echoed back in the second request without an API-shape error,
// at every ThinkingLevel including off, and (2) a replayed assistant message
// made only of thinking (no text content) is also accepted. This test targets
// the default model minimax-m2.7, which accepts thinking blocks at every level.
func TestLiveThinkingRoundTrip(t *testing.T) {
	ep := config.Defaults().Provider.Endpoints["opencode"]
	key, source := ep.ResolveKey(os.Getenv)
	if key == "" {
		t.Skipf("set KIRSCH_%s or %s to run the live thinking round-trip test", ep.APIKeyEnv, ep.APIKeyEnv)
	}
	model, modelSource, err := liveModel(os.LookupEnv, ep.Model)
	if err != nil {
		t.Fatalf("invalid model override: %v", err)
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	c, err := New(Options{
		Endpoint: "opencode", BaseURL: ep.BaseURL, Auth: ep.Auth, APIKey: key, KeyEnv: ep.APIKeyEnv,
		PromptCaching: ep.PromptCaching, Version: "thinking-round-trip", SessionID: hex.EncodeToString(b[:]),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	levels := []provider.ThinkingLevel{provider.ThinkingOff, provider.ThinkingLow, provider.ThinkingMedium, provider.ThinkingHigh}
	for _, lvl := range levels {
		t.Run(lvl.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			first := provider.Request{
				Model: model, MaxTokens: 512, Thinking: lvl,
				Tools:    []provider.ToolDef{readFileTool()},
				Messages: []provider.Message{userText(readPrompt)},
			}
			blocks, err := liveAssistantBlocks(ctx, c, first)
			if err != nil {
				t.Fatalf("first request failed: %v", err)
			}

			var thinking *provider.Thinking
			var call *provider.ToolCall
			for _, blk := range blocks {
				if blk.Kind == provider.BlockThinking && blk.Thinking != nil && blk.Thinking.Signature != "" {
					thinking = blk.Thinking
				}
				if blk.Kind == provider.BlockToolCall && blk.ToolCall != nil {
					call = blk.ToolCall
				}
			}
			if thinking == nil {
				t.Fatalf("no signed thinking block in first response: %s", dumpBlocks(blocks))
			}
			if call == nil {
				t.Fatalf("no tool call in first response: %s", dumpBlocks(blocks))
			}

			second := provider.Request{
				Model: model, MaxTokens: 512, Thinking: lvl,
				Tools: []provider.ToolDef{readFileTool()},
				Messages: []provider.Message{
					userText(readPrompt),
					{Role: provider.RoleAssistant, Content: []provider.Block{
						{Kind: provider.BlockThinking, Thinking: thinking},
						{Kind: provider.BlockToolCall, ToolCall: call},
					}},
					{Role: provider.RoleUser, Content: []provider.Block{{
						Kind:       provider.BlockToolResult,
						ToolResult: &provider.ToolResult{CallID: call.ID, Content: "# Kirsch\nA terminal coding agent."},
					}}},
				},
			}
			if _, err := liveAssistantBlocks(ctx, c, second); err != nil {
				t.Fatalf("second request (echoed thinking + tool call) failed: %v", err)
			}

			onlyThinking := provider.Request{
				Model: model, MaxTokens: 512, Thinking: lvl,
				Tools: []provider.ToolDef{readFileTool()},
				Messages: []provider.Message{
					userText(readPrompt),
					{Role: provider.RoleAssistant, Content: []provider.Block{
						{Kind: provider.BlockThinking, Thinking: thinking},
					}},
					userText("Summarise your reasoning."),
				},
			}
			if _, err := liveAssistantBlocks(ctx, c, onlyThinking); err != nil {
				t.Fatalf("only-thinking replay failed: %v", err)
			}
		})
	}

	t.Logf("endpoint=opencode model=%s model_source=%s key_source=%s thinking round-trip passed for all levels",
		model, modelSource, source)
}

// liveAssistantBlocks streams req and returns the assistant message blocks.
func liveAssistantBlocks(ctx context.Context, c *Client, req provider.Request) ([]provider.Block, error) {
	var blocks []provider.Block
	err := c.Stream(ctx, req, func(e provider.StreamEvent) {
		switch e.Type {
		case provider.EventThinkingDone:
			blocks = append(blocks, provider.Block{Kind: provider.BlockThinking, Thinking: e.Thinking})
		case provider.EventToolCallEnd:
			blocks = append(blocks, provider.Block{Kind: provider.BlockToolCall, ToolCall: e.ToolCall})
		case provider.EventTextDelta:
			if n := len(blocks); n > 0 && blocks[n-1].Kind == provider.BlockText {
				blocks[n-1].Text += e.Text
			} else {
				blocks = append(blocks, provider.Block{Kind: provider.BlockText, Text: e.Text})
			}
		}
	})
	return blocks, err
}

// dumpBlocks returns a compact JSON representation of blocks for failure messages.
func dumpBlocks(blocks []provider.Block) string {
	b, err := json.Marshal(blocks)
	if err != nil {
		return fmt.Sprintf("%#v", blocks)
	}
	return string(b)
}
