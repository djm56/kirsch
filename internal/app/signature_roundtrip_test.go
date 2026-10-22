package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/provider/anthropic"
	"github.com/djm56/kirsch/internal/session"
)

// sigFixtures is the probe fixture tree the anthropic package tests also read.
const sigFixtures = "../../testdata/probe/opencode"

// seededSig is the signature carried by seededSSE.
const seededSig = "5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed5eed"

// seededSSE is a synthetic stream written for this test, not a captured one.
// Its thinking block carries both text and signature on content_block_start, so
// they enter through the decoder's seedThinking path; one thinking_delta adds
// more text, and no signature_delta follows.
const seededSSE = `event: message_start
data: {"type":"message_start","message":{"id":"msg_synthetic","type":"message","role":"assistant","model":"synthetic","content":[],"usage":{"input_tokens":1,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"Seeded at start.","signature":"` + seededSig + `"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":" Then one delta."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"ok"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":1,"output_tokens":1}}

event: message_stop
data: {"type":"message_stop"}

`

// replayProvider is a provider.Provider that answers every Stream call by
// running one SSE body through the real anthropic.Decode. It records each
// provider.Request it receives, so a test can encode exactly what
// providerModel.Stream built.
type replayProvider struct {
	body []byte
	reqs []provider.Request
}

// Stream records req and decodes the stored body into onEvent.
func (p *replayProvider) Stream(ctx context.Context, req provider.Request, onEvent func(provider.StreamEvent)) error {
	p.reqs = append(p.reqs, req)
	return anthropic.Decode(ctx, bytes.NewReader(p.body), onEvent)
}

// noTools is an agent.Tools with no tools; the streams in this test call none.
type noTools struct{}

// Describe returns no tool specs.
func (noTools) Describe() []agent.ToolSpec { return nil }

// Invoke panics: no stream in this test carries a tool call.
func (noTools) Invoke(ctx context.Context, c agent.ToolCall) agent.ToolResult {
	panic("noTools.Invoke: the streams in this test carry no tool calls")
}

// firstThinking returns the first thinking block found in msgs, or nil.
func firstThinking(msgs []agent.Message) *agent.Thinking {
	for _, m := range msgs {
		for _, b := range m.Content {
			if b.Kind == agent.BlockThinking && b.Thinking != nil {
				return b.Thinking
			}
		}
	}
	return nil
}

// wireShape describes one encoded message as `role "text"` when its content
// is a JSON string, or `role [type type ...]` when it is a block array.
func wireShape(role string, content json.RawMessage) string {
	var blocks []struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(content, &blocks) != nil {
		return role + " " + string(content)
	}
	types := make([]string, len(blocks))
	for i, b := range blocks {
		types[i] = b.Type
	}
	return role + " [" + strings.Join(types, " ") + "]"
}

// TestThinkingSignatureRoundTrip pins what Kirsch does with a thinking
// block's signature across the whole round trip: SSE decode, provider
// event, agent message, session JSONL, resume, and the next request's encode.
//
// Entry points: the "decode" subtest calls anthropic.Decode directly. The
// "agent message" subtest drives agent.Agent.Turn through providerModel.Stream,
// with a replayProvider standing in for the HTTP client, into a real
// session.Store. "session JSONL" reads the file that store wrote, and "resume"
// calls session.Resume on it. "encode" runs a second Turn on the resumed
// conversation through providerModel.Stream and passes the provider.Request
// that Turn produced to anthropic.EncodeRequest. App.Submit is not used, and
// no HTTP call is made.
//
// Each subtest checks its own stage and fails with text naming that stage. A
// subtest skips, rather than fails, when an earlier stage produced nothing for
// it to read.
//
// Cases. "qwen empty" is the qwen3.8-flash fixture, whose thinking block
// carries signature "" on content_block_start and in its signature_delta.
// "minimax signed" is the minimax-m2.7 fixture, whose 64-character signature
// arrives in a signature_delta; it is the positive control and must survive
// byte for byte. "seeded at start" is seededSSE, a synthetic stream whose
// signature arrives on content_block_start.
//
// This test pins current behaviour; it does not assert the endpoint accepts
// an empty signature.
func TestThinkingSignatureRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		// file is a fixture under sigFixtures; sse is used when file is empty.
		file string
		sse  string
		// model is the model name providerModel puts on the request.
		model string
		// wantSig is the signature the stream carries.
		wantSig string
		// wantText is the exact thinking text the stream carries.
		wantText string
		// wantReply is the text block the stream carries.
		wantReply string
	}{
		{
			name:      "qwen empty",
			file:      "2026-10-22/qwen3.8-flash/say-hello.sse",
			model:     "qwen3.8-flash",
			wantSig:   "",
			wantText:  `We need to respond to user: "say hello". Simple. Need final say hello. Ensure no analysis leaked.`,
			wantReply: "Hello!",
		},
		{
			name:      "minimax signed",
			file:      "2026-10-02T1421Z/minimax-m2.7/S1-text.sse",
			model:     "minimax-m2.7",
			wantSig:   "02101bd8777fa55b41d890249e759e991b474a2433f0213820057b49b01a4da9",
			wantText:  `The user is asking me to respond with exactly the word "pong". This is a simple request that doesn't involve any harmful content. I should just respond with exactly what they asked for.`,
			wantReply: "pong",
		},
		{
			name:      "seeded at start",
			sse:       seededSSE,
			model:     "qwen3.8-flash",
			wantSig:   seededSig,
			wantText:  "Seeded at start. Then one delta.",
			wantReply: "ok",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(tc.sse)
			if tc.file != "" {
				var err error
				if body, err = os.ReadFile(filepath.Join(sigFixtures, tc.file)); err != nil {
					t.Fatalf("test setup: %v", err)
				}
			}
			if len(tc.wantSig) != 0 && len(tc.wantSig) != 64 {
				t.Fatalf("test setup: control signature is %d bytes, want 64", len(tc.wantSig))
			}
			ctx := context.Background()
			// The session file lives in the case's temp dir, not the agent
			// message subtest's, so it outlives that subtest for the stages after it.
			dir := t.TempDir()

			// Values one stage hands to the next. Each is set only once the stage
			// that produces it has produced it.
			var (
				path    string // session JSONL, set once the store is closed
				resumed *agent.Conversation
			)

			t.Run("decode", func(t *testing.T) {
				var done []*provider.Thinking
				var reply strings.Builder
				err := anthropic.Decode(ctx, bytes.NewReader(body), func(ev provider.StreamEvent) {
					switch ev.Type {
					case provider.EventThinkingDone:
						c := *ev.Thinking
						done = append(done, &c)
					case provider.EventTextDelta:
						reply.WriteString(ev.Text)
					default:
						// The other events are irrelevant to this test.
					}
				})
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(done) != 1 {
					t.Fatalf("decode: got %d ThinkingDone events, want 1", len(done))
				}
				th := done[0]
				if th.Text != tc.wantText {
					t.Errorf("decode: thinking text = %q, want %q", th.Text, tc.wantText)
				}
				if th.Signature != tc.wantSig {
					t.Errorf("decode: Signature = %q, want %q", th.Signature, tc.wantSig)
				}
				if th.Redacted {
					t.Errorf("decode: block marked Redacted")
				}
				if reply.String() != tc.wantReply {
					t.Errorf("decode: text block = %q, want %q", reply.String(), tc.wantReply)
				}
			})

			t.Run("agent message", func(t *testing.T) {
				p := filepath.Join(dir, "session.jsonl")
				store, err := session.New(p, session.WithCadence(time.Hour))
				if err != nil {
					t.Fatalf("agent message: session.New: %v", err)
				}
				a := &agent.Agent{
					Model:     &providerModel{p: &replayProvider{body: body}, model: tc.model},
					Tools:     noTools{},
					Recorder:  store,
					MaxTokens: 64,
				}
				conv := &agent.Conversation{}
				if err := a.Turn(ctx, conv, "say hello", nil); err != nil {
					t.Fatalf("agent message: Turn: %v", err)
				}
				if err := store.Flush(); err != nil {
					t.Fatalf("agent message: Flush: %v", err)
				}
				if err := store.Close(); err != nil {
					t.Fatalf("agent message: Close: %v", err)
				}
				path = p
				th := firstThinking(conv.Messages)
				if th == nil {
					t.Fatalf("agent message: no thinking block in conversation")
				}
				if th.Text != tc.wantText {
					t.Errorf("agent message: thinking text = %q, want %q", th.Text, tc.wantText)
				}
				if th.Signature != tc.wantSig {
					t.Errorf("agent message: Signature = %q, want %q", th.Signature, tc.wantSig)
				}
			})

			t.Run("session JSONL", func(t *testing.T) {
				if path == "" {
					t.Skip("session JSONL: skipped, the agent message subtest wrote no session file")
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("session JSONL: %v", err)
				}
				var found int
				for _, line := range bytes.Split(raw, []byte("\n")) {
					var m map[string]any
					if len(line) == 0 || json.Unmarshal(line, &m) != nil || m["type"] != "assistant.thinking" {
						continue
					}
					found++
					if m["content"] != tc.wantText {
						t.Errorf("session JSONL: content = %v, want %q", m["content"], tc.wantText)
					}
					sig, present := m["signature"]
					if tc.wantSig == "" {
						// Current behaviour, pinned: assistantThinkingEvent tags
						// Signature `json:"signature,omitempty"`, so an empty
						// signature is omitted from the line, not written as "".
						// Resume decodes a missing key and an empty one to the same "".
						if present {
							t.Errorf("session JSONL: empty signature was written as %v; expected the key omitted (omitempty)", sig)
						}
					} else if sig != tc.wantSig {
						t.Errorf("session JSONL: signature = %v, want %q", sig, tc.wantSig)
					}
				}
				if found != 1 {
					t.Errorf("session JSONL: %d assistant.thinking lines, want 1\n%s", found, raw)
				}
			})

			t.Run("resume", func(t *testing.T) {
				if path == "" {
					t.Skip("resume: skipped, the agent message subtest wrote no session file")
				}
				conv, _, err := session.Resume(path)
				if err != nil {
					t.Fatalf("resume: %v", err)
				}
				resumed = conv
				th := firstThinking(conv.Messages)
				if th == nil {
					t.Fatalf("resume: no thinking block after resume")
				}
				if th.Text != tc.wantText {
					t.Errorf("resume: thinking text = %q, want %q", th.Text, tc.wantText)
				}
				if th.Signature != tc.wantSig {
					t.Errorf("resume: Signature = %q, want %q", th.Signature, tc.wantSig)
				}
			})

			t.Run("encode", func(t *testing.T) {
				if resumed == nil {
					t.Skip("encode: skipped, the resume subtest produced no conversation")
				}
				rp := &replayProvider{body: body}
				a := &agent.Agent{
					Model:     &providerModel{p: rp, model: tc.model},
					Tools:     noTools{},
					MaxTokens: 64,
				}
				if err := a.Turn(ctx, resumed, "and again", nil); err != nil {
					t.Fatalf("encode: second Turn: %v", err)
				}
				if len(rp.reqs) != 1 {
					t.Fatalf("encode: provider received %d requests, want 1", len(rp.reqs))
				}
				enc, err := anthropic.EncodeRequest(rp.reqs[0], anthropic.EncodeOptions{})
				if err != nil {
					t.Fatalf("encode: EncodeRequest: %v", err)
				}
				var wire struct {
					Messages []struct {
						Role    string          `json:"role"`
						Content json.RawMessage `json:"content"`
					} `json:"messages"`
				}
				if err := json.Unmarshal(enc, &wire); err != nil {
					t.Fatalf("encode: not JSON: %v", err)
				}

				// Observed current behaviour, pinned: the session file holds the
				// assistant.thinking line before the assistant.message line,
				// session.Resume rebuilds them as two consecutive assistant
				// messages in that order, and EncodeRequest sends them as two
				// messages without merging them. A message whose only block is
				// text is encoded with string content.
				replyJSON, _ := json.Marshal(tc.wantReply)
				wantShape := []string{
					`user "say hello"`,
					`assistant [thinking]`,
					`assistant ` + string(replyJSON),
					`user "and again"`,
				}
				gotShape := make([]string, len(wire.Messages))
				for i, m := range wire.Messages {
					gotShape[i] = wireShape(m.Role, m.Content)
				}
				if strings.Join(gotShape, "\n") != strings.Join(wantShape, "\n") {
					t.Fatalf("encode: message shape\n got: %q\nwant: %q\n%s", gotShape, wantShape, enc)
				}

				var blocks []map[string]json.RawMessage
				if err := json.Unmarshal(wire.Messages[1].Content, &blocks); err != nil {
					t.Fatalf("encode: thinking message content: %v", err)
				}
				b := blocks[0]
				sig, present := b["signature"]
				if !present {
					t.Fatalf("encode: thinking block has no \"signature\" key\n%s", enc)
				}
				wantSigJSON, _ := json.Marshal(tc.wantSig)
				if string(sig) != string(wantSigJSON) {
					t.Errorf("encode: signature = %s, want %s", sig, wantSigJSON)
				}
				var text string
				if err := json.Unmarshal(b["thinking"], &text); err != nil || text != tc.wantText {
					t.Errorf("encode: thinking = %s, want %q", b["thinking"], tc.wantText)
				}
			})
		})
	}
}
