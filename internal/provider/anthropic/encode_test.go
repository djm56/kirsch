package anthropic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/djm56/kirsch/internal/provider"
)

func asJSON(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, b)
	}
	return v
}

func fixtureRequest(t *testing.T, rel string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtures, rel))
	if err != nil {
		t.Fatal(err)
	}
	return asJSON(t, b)
}

func readFileTool() provider.ToolDef {
	return provider.ToolDef{
		Name: "read_file", Description: "Read a file from the repository",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
	}
}

func encode(t *testing.T, req provider.Request, opts EncodeOptions) map[string]any {
	t.Helper()
	b, err := EncodeRequest(req, opts)
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	return asJSON(t, b)
}

func userText(s string) provider.Message {
	return provider.Message{Role: provider.RoleUser, Content: []provider.Block{{Kind: provider.BlockText, Text: s}}}
}

const readPrompt = "Use the read_file tool to read README.md. You must call the tool before answering."

// TestEncodeMatchesProbeToolResult: the minimax-m3 S3 request, which the live
// endpoint accepted.
func TestEncodeMatchesProbeToolResult(t *testing.T) {
	req := provider.Request{
		Model: "minimax-m3", MaxTokens: 512, Tools: []provider.ToolDef{readFileTool()},
		Messages: []provider.Message{
			userText(readPrompt),
			{Role: provider.RoleAssistant, Content: []provider.Block{{
				Kind:     provider.BlockToolCall,
				ToolCall: &provider.ToolCall{ID: "call_01a0fcfdd40d71f2b8525abb", Name: "read_file", Input: json.RawMessage(`{"path":"README.md"}`)},
			}}},
			{Role: provider.RoleUser, Content: []provider.Block{{
				Kind:       provider.BlockToolResult,
				ToolResult: &provider.ToolResult{CallID: "call_01a0fcfdd40d71f2b8525abb", Content: "# Kirsch\nA terminal coding agent."},
			}}},
		},
	}
	got := encode(t, req, EncodeOptions{})
	want := fixtureRequest(t, "minimax-m3/S3-tool-result.request.json")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("encoded\n%v\nwant\n%v", got, want)
	}
}

// TestEncodeMatchesProbeThinkingEcho: the minimax-m2.7 S3 request, which
// echoed a signed thinking block and was accepted.
func TestEncodeMatchesProbeThinkingEcho(t *testing.T) {
	want := fixtureRequest(t, "minimax-m2.7/S3-tool-result.request.json")
	req := provider.Request{
		Model: "minimax-m2.7", MaxTokens: 512, Tools: []provider.ToolDef{readFileTool()},
		Messages: []provider.Message{
			userText(readPrompt),
			{Role: provider.RoleAssistant, Content: []provider.Block{
				{Kind: provider.BlockThinking, Thinking: &provider.Thinking{
					Text:      "The user wants me to read the README.md file using the read_file tool. Let me do that first.",
					Signature: "efdda6f44a7ebf04ccaa7b8fec455a0ab4bab8841e5d52a62c1eab82a4aafe1b",
				}},
				{Kind: provider.BlockToolCall, ToolCall: &provider.ToolCall{ID: "call_function_ui3e9pj8nr99_1", Name: "read_file", Input: json.RawMessage(`{"path":"README.md"}`)}},
			}},
			{Role: provider.RoleUser, Content: []provider.Block{{
				Kind:       provider.BlockToolResult,
				ToolResult: &provider.ToolResult{CallID: "call_function_ui3e9pj8nr99_1", Content: "# Kirsch\nA terminal coding agent."},
			}}},
		},
	}
	got := encode(t, req, EncodeOptions{})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("encoded\n%v\nwant\n%v", got, want)
	}
}

// TestEncodeMatchesProbeCache: the minimax-m3 S4 request, with a cache
// breakpoint on the system prompt; none is sent with prompt caching off.
func TestEncodeMatchesProbeCache(t *testing.T) {
	want := fixtureRequest(t, "minimax-m3/S4-cache-1.request.json")
	sys := want["system"].([]any)[0].(map[string]any)["text"].(string)
	req := provider.Request{
		Model: "minimax-m3", MaxTokens: 256,
		System:   []provider.SystemBlock{{Text: sys, Cache: true}},
		Messages: []provider.Message{userText("Hello, world.")},
	}
	got := encode(t, req, EncodeOptions{PromptCaching: true})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("encoded\n%v\nwant\n%v", got, want)
	}
	off := encode(t, req, EncodeOptions{PromptCaching: false})
	if _, ok := off["system"].([]any)[0].(map[string]any)["cache_control"]; ok {
		t.Fatal("cache_control sent with prompt_caching off")
	}
}

// TestEncodeNeverSendsThinkingField: m3-d4 owns the thinking request field.
func TestEncodeNeverSendsThinkingField(t *testing.T) {
	for _, lvl := range []provider.ThinkingLevel{provider.ThinkingOff, provider.ThinkingLow, provider.ThinkingMedium, provider.ThinkingHigh} {
		got := encode(t, provider.Request{Model: "m", MaxTokens: 1, Thinking: lvl, Messages: []provider.Message{userText("x")}}, EncodeOptions{})
		if _, ok := got["thinking"]; ok {
			t.Fatalf("thinking field sent at %v", lvl)
		}
		if got["stream"] != true {
			t.Fatalf("stream = %v", got["stream"])
		}
		if len(got) != 4 {
			t.Fatalf("top-level fields = %v, want model, max_tokens, stream, messages", got)
		}
	}
}

// TestEncodeBlocks: multi-block text, redacted thinking, an empty tool input
// and error results.
func TestEncodeBlocks(t *testing.T) {
	got := encode(t, provider.Request{Model: "m", MaxTokens: 1, Messages: []provider.Message{
		{Role: provider.RoleUser, Content: []provider.Block{{Kind: provider.BlockText, Text: "a"}, {Kind: provider.BlockText, Text: "b"}}},
		{Role: provider.RoleAssistant, Content: []provider.Block{
			{Kind: provider.BlockThinking, Thinking: &provider.Thinking{Redacted: true, Data: "OPAQUE=="}},
			{Kind: provider.BlockToolCall, ToolCall: &provider.ToolCall{ID: "t1", Name: "x"}},
		}},
		{Role: provider.RoleUser, Content: []provider.Block{
			{Kind: provider.BlockToolResult, ToolResult: &provider.ToolResult{CallID: "t1", Content: "boom", IsError: true}},
			{Kind: provider.BlockToolResult, ToolResult: &provider.ToolResult{CallID: "t1", Content: "ok"}},
		}},
	}}, EncodeOptions{})
	b, err := json.Marshal(got["messages"])
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"content":[{"text":"a","type":"text"},{"text":"b","type":"text"}],"role":"user"},` +
		`{"content":[{"data":"OPAQUE==","type":"redacted_thinking"},{"id":"t1","input":{},"name":"x","type":"tool_use"}],"role":"assistant"},` +
		`{"content":[{"content":"boom","is_error":true,"tool_use_id":"t1","type":"tool_result"},{"content":"ok","tool_use_id":"t1","type":"tool_result"}],"role":"user"}]`
	if string(b) != want {
		t.Fatalf("messages\n%s\nwant\n%s", b, want)
	}
}

// TestEncodeRejects: malformed neutral requests are refused, not sent.
func TestEncodeRejects(t *testing.T) {
	one := func(b provider.Block) []provider.Message {
		return []provider.Message{{Role: provider.RoleUser, Content: []provider.Block{b}}}
	}
	text := provider.Block{Kind: provider.BlockText, Text: "x"}
	cases := map[string]provider.Request{
		"no model":        {MaxTokens: 1, Messages: one(text)},
		"no max_tokens":   {Model: "m", Messages: one(text)},
		"bad tool input":  {Model: "m", MaxTokens: 1, Messages: one(provider.Block{Kind: provider.BlockToolCall, ToolCall: &provider.ToolCall{ID: "t", Name: "x", Input: json.RawMessage(`{"a":`)}})},
		"nil tool call":   {Model: "m", MaxTokens: 1, Messages: one(provider.Block{Kind: provider.BlockToolCall})},
		"nil thinking":    {Model: "m", MaxTokens: 1, Messages: one(provider.Block{Kind: provider.BlockThinking})},
		"nil tool result": {Model: "m", MaxTokens: 1, Messages: one(provider.Block{Kind: provider.BlockToolResult})},
		"unknown kind":    {Model: "m", MaxTokens: 1, Messages: one(provider.Block{Kind: provider.BlockKind(99)})},
		"bad schema":      {Model: "m", MaxTokens: 1, Messages: one(text), Tools: []provider.ToolDef{{Name: "x", InputSchema: json.RawMessage(`{`)}}},
		"no schema":       {Model: "m", MaxTokens: 1, Messages: one(text), Tools: []provider.ToolDef{{Name: "x"}}},
	}
	for name, req := range cases {
		if _, err := EncodeRequest(req, EncodeOptions{}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
