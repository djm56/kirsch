package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Scenario represents a single probe scenario.
type Scenario struct {
	Model    string
	Scenario string
	Method   string
	Path     string
	Request  interface{}
	// SessionGroup names the conversation this scenario belongs to, within its
	// model. Scenarios of one model that share a SessionGroup share a session
	// ID; an empty group means the scenario is a conversation of its own.
	SessionGroup string
	// Build generates the request body dynamically and may skip the scenario.
	// Returns (body, skipReason). If skipReason is non-empty, the scenario is skipped.
	Build func(results map[string]map[string]*Response) (body interface{}, skip string)
}

// Response holds a probe response.
type Response struct {
	Status      int
	Headers     http.Header
	Body        []byte
	Time        time.Duration
	Error       string
	RequestBody []byte     // The actual request sent (F5)
	Parsed      *ParsedSSE // Parsed SSE data for building S3 requests (F3)
	TTFBMS      int        // Time to first byte in milliseconds (N10)
	TotalMS     int        // Total time in milliseconds (N10)
	Session     string     // x-opencode-session sent with the request; not a secret
}

// Identity headers the gateway requires of a client (see the README). They are
// sent on every request, /models included.
const (
	// userAgent identifies the probe, rather than Go's default Go-http-client.
	userAgent = "kirsch-probe/0.1"
	// sessionHeader carries a stable per-conversation ID for routing and caching.
	sessionHeader = "x-opencode-session"
)

// newSessionID returns 128 random bits from crypto/rand, hex-encoded.
func newSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating session ID: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// sessionFor returns the session ID for sc, generating it on first use. IDs
// live in the Prober, so a new run never reuses an earlier run's IDs.
func (p *Prober) sessionFor(sc *Scenario) (string, error) {
	group := sc.SessionGroup
	if group == "" {
		group = sc.Scenario
	}
	k := sc.Model + "\x00" + group
	if id, ok := p.sessions[k]; ok {
		return id, nil
	}
	id, err := newSessionID()
	if err != nil {
		return "", err
	}
	p.sessions[k] = id
	return id, nil
}

// isGatewayPolicyError reports whether a 400 body is a gateway policy error
// (an error type such as MissingSessionID) rather than a complaint about the
// request body. Retrying such a request cannot succeed, so the run stops.
func isGatewayPolicyError(status int, body []byte) (errType string, ok bool) {
	if status != http.StatusBadRequest {
		return "", false
	}
	var e struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) != nil {
		return "", false
	}
	t := e.Error.Type
	return t, t == "MissingSessionID" || strings.Contains(t, "Session")
}

// ParsedSSE holds parsed streaming data from SSE responses.
type ParsedSSE struct {
	Usage         map[string]interface{}
	StopReason    string
	ContentBlocks []SSEContentBlock
	// Errors collects every problem found in the stream: error events and
	// event payloads that failed to decode (including malformed partial_json).
	// All entries are sanitised; none is a raw server string.
	Errors []string
}

// SSEContentBlock represents a parsed content block.
type SSEContentBlock struct {
	Type             string
	Index            int
	ToolUse          *SSEToolUse
	Text             string
	Thinking         *SSEThinking
	RedactedThinking *SSERedactedThinking
}

// SSEToolUse holds tool use block data.
type SSEToolUse struct {
	ID    string
	Name  string
	Input map[string]interface{}
}

// SSEThinking holds thinking block data.
type SSEThinking struct {
	Thinking  string
	Signature string
}

// SSERedactedThinking holds redacted thinking block data.
type SSERedactedThinking struct {
	Data string
}

// Prober orchestrates the probe execution.
type Prober struct {
	opts      options
	client    *http.Client
	results   map[string]map[string]*Response // model -> scenario -> response
	scenarios []Scenario
	sessions  map[string]string // model+group -> session ID, fresh per Prober
}

// NewProber creates a new probe runner.
func NewProber(opts options) *Prober {
	return &Prober{
		opts: opts,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		results:  make(map[string]map[string]*Response),
		sessions: make(map[string]string),
	}
}

// Plan builds the probe scenarios.
func (p *Prober) Plan() []Scenario {
	p.scenarios = buildScenarios(p.opts.models)
	return p.scenarios
}

// Run executes all scenarios.
func (p *Prober) Run(ctx context.Context, stdout io.Writer) error {
	if len(p.scenarios) == 0 {
		p.scenarios = buildScenarios(p.opts.models)
	}

	// Initialize results map.
	for _, model := range p.opts.models {
		p.results[model] = make(map[string]*Response)
	}
	p.results["_global"] = make(map[string]*Response)

	// Execute scenarios sequentially.
	for i, sc := range p.scenarios {
		// Pause between requests (F9 - skip before first request).
		if i > 0 {
			select {
			case <-time.After(p.opts.pause):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		session, err := p.sessionFor(&sc)
		if err != nil {
			return err
		}

		// Dynamic build (which may skip) or a fixed request.
		var requestBody []byte
		var requestData interface{}

		if sc.Build != nil {
			body, skip := sc.Build(p.results)
			if skip != "" {
				respObj := &Response{
					Status:  0,
					Error:   skip,
					Session: session,
				}
				p.saveResponse(&sc, respObj)
				if _, err := fmt.Fprintf(stdout, "⊘ %s %s (skipped: %s)\n", sc.Model, sc.Scenario, skip); err != nil {
					return err
				}
				continue
			}
			requestData = body
		} else {
			requestData = sc.Request
		}

		// Marshal request body.
		if requestData != nil {
			var err error
			requestBody, err = json.Marshal(requestData)
			if err != nil {
				return fmt.Errorf("marshaling request for %s: %v", sc.Scenario, err)
			}
			if p.opts.key != "" && bytes.Contains(requestBody, []byte(p.opts.key)) {
				return fmt.Errorf("refusing to send request for %s: contains API key", sc.Scenario)
			}
		}

		// Create request with context timeout (F7).
		reqCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, sc.Method, p.opts.baseURL+sc.Path, bytes.NewReader(requestBody))
		if err != nil {
			cancel()
			return err
		}

		// Set headers.
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set(sessionHeader, session)
		setAuth(req, p.opts.auth, p.opts.key)
		if sc.Scenario != "S6-no-version-header" {
			req.Header.Set("anthropic-version", "2023-06-01")
		}

		// Execute request.
		start := time.Now()
		resp, err := p.client.Do(req)
		ttfb := time.Since(start) // Time to first byte (response headers received)

		respObj := &Response{
			RequestBody: requestBody,
			Session:     session,
		}

		// Handle request errors (F7).
		if err != nil {
			cancel() // Cancel context once we're done with the request (N9).
			// Transport errors can embed server-derived text (net/http quotes a
			// bad Location header), so they are sanitised before they go anywhere.
			msg := p.safe(err.Error(), 300)
			if strings.Contains(err.Error(), "failed to parse Location header") {
				// net/http quotes the raw Location here; withhold it entirely.
				msg = "redirect not followed, Location host: " + locationUnparseable
			}
			respObj.Error = msg
			respObj.Status = 0
			p.saveResponse(&sc, respObj)
			if _, err := fmt.Fprintf(stdout, "✗ %s %s (error: %s)\n", sc.Model, sc.Scenario, msg); err != nil {
				return err
			}
			return fmt.Errorf("request failed for %s: %s", sc.Scenario, msg)
		}

		respObj.Status = resp.StatusCode
		respObj.Headers = filterHeaders(resp.Header)

		// Read response with limit (F7 - 8MB).
		limitedBody := io.LimitReader(resp.Body, 8*1024*1024)
		body, err := io.ReadAll(limitedBody)
		totalElapsed := time.Since(start) // Total time including body read (N10)
		closeErr := resp.Body.Close()
		cancel() // Cancel context once body has been read (N9).

		// Record timing metrics (N10).
		respObj.Time = totalElapsed
		respObj.TTFBMS = int(ttfb.Milliseconds())
		respObj.TotalMS = int(totalElapsed.Milliseconds())

		// Check read error before close error (F7).
		if err != nil {
			msg := p.safe(fmt.Sprintf("reading response body: %v", err), 300)
			respObj.Error = msg
			respObj.Body = body // Save partial body
			p.saveResponse(&sc, respObj)
			if _, err := fmt.Fprintf(stdout, "✗ %s %s (read error)\n", sc.Model, sc.Scenario); err != nil {
				return err
			}
			return fmt.Errorf("reading response body for %s failed: %s", sc.Scenario, msg)
		}

		if closeErr != nil {
			msg := p.safe(fmt.Sprintf("closing response body: %v", closeErr), 300)
			respObj.Error = msg
			respObj.Body = body
			p.saveResponse(&sc, respObj)
			if _, err := fmt.Fprintf(stdout, "✗ %s %s (close error)\n", sc.Model, sc.Scenario); err != nil {
				return err
			}
			return fmt.Errorf("closing response body for %s failed: %s", sc.Scenario, msg)
		}

		respObj.Body = body

		// Parse SSE response.
		if isEventStream(resp.Header.Get("Content-Type")) || bytes.HasPrefix(body, []byte("event:")) {
			parsed, parseErr := parseSSEStream(body, p.opts.key)
			respObj.Parsed = parsed
			var problems []string
			if parseErr != nil {
				problems = append(problems, fmt.Sprintf("parsing SSE: %s", p.safe(parseErr.Error(), 200)))
			}
			if parsed != nil {
				problems = append(problems, parsed.Errors...)
			}
			if len(problems) > 0 {
				respObj.Error = strings.Join(problems, "; ")
			}
		}

		p.saveResponse(&sc, respObj)

		// A 200 whose stream carried a problem is not a clean pass.
		note := ""
		if respObj.Error != "" {
			note = " (stream error)"
		}
		if _, err := fmt.Fprintf(stdout, "✓ %d %s %s (%.1fs)%s\n", resp.StatusCode, sc.Model, sc.Scenario, totalElapsed.Seconds(), note); err != nil {
			return err
		}

		// Check for stop conditions (N5, F2). The stopping body is truncated
		// to 4KB here, so WriteResults never has to guess which bodies are
		// stop bodies.
		isRedirect := resp.StatusCode >= 300 && resp.StatusCode < 400
		policyType, isPolicy := isGatewayPolicyError(resp.StatusCode, respObj.Body)
		if isRedirect || isPolicy || resp.StatusCode == 429 || resp.StatusCode == 401 || resp.StatusCode == 403 {
			if len(respObj.Body) > 4096 {
				respObj.Body = respObj.Body[:4096]
			}
			if isRedirect {
				host := redirectHost(resp.Header.Get("Location"), p.opts.key)
				respObj.Error = fmt.Sprintf("received %d response, Location host: %s", resp.StatusCode, host)
				p.saveResponse(&sc, respObj)
				return fmt.Errorf("probe stopped: received %d response for %s, redirect to %s", resp.StatusCode, sc.Scenario, host)
			}
			if isPolicy {
				t := p.safe(policyType, 100)
				respObj.Error = fmt.Sprintf("received %d response, gateway error type: %s", resp.StatusCode, t)
				p.saveResponse(&sc, respObj)
				return fmt.Errorf("probe stopped: received %d response for %s, gateway error type %s", resp.StatusCode, sc.Scenario, t)
			}
			respObj.Error = fmt.Sprintf("received %d response", resp.StatusCode)
			p.saveResponse(&sc, respObj)
			return fmt.Errorf("probe stopped: received %d response for %s", resp.StatusCode, sc.Scenario)
		}
	}

	return nil
}

// safe makes a server-derived string fit to print or store: the API key is
// redacted first and the result is then sanitised and truncated, in that order,
// so truncation can never leave a prefix of the key behind. Every string the
// probe prints, returns as an error or records in a summary goes through it.
func (p *Prober) safe(s string, n int) string {
	return sanitize(redactKey(s, p.opts.key), n)
}

// saveResponse saves a response to the results map.
func (p *Prober) saveResponse(sc *Scenario, respObj *Response) {
	if sc.Model != "" {
		p.results[sc.Model][sc.Scenario] = respObj
	} else {
		p.results["_global"][sc.Scenario] = respObj
	}
}

// Results returns the collected results.
func (p *Prober) Results() map[string]map[string]*Response {
	return p.results
}

// isEventStream reports whether a Content-Type header names text/event-stream,
// ignoring parameters such as charset.
func isEventStream(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "text/event-stream"
}

// Placeholders shown instead of a Location host that cannot be shown.
const (
	locationUnparseable = "<unparseable>"
	locationNoHost      = "<no-host>"
)

// redirectHost reduces a Location header to its host for display. The full
// value is never echoed to the terminal: it is server-derived, and may carry
// a path or query. A value that does not parse yields locationUnparseable; one
// that parses without a host (a relative redirect) yields locationNoHost.
func redirectHost(location, key string) string {
	u, err := url.Parse(location)
	if err != nil {
		return locationUnparseable
	}
	if u.Host == "" {
		return locationNoHost
	}
	return sanitize(redactKey(u.Host, key), 100)
}

// redactKey replaces every occurrence of key in s with "<redacted>". An empty
// key leaves s unchanged. It must run before sanitize truncates, never after.
func redactKey(s, key string) string {
	if key == "" {
		return s
	}
	return strings.ReplaceAll(s, key, "<redacted>")
}

// sanitize makes a server-derived string safe to print or store in a summary:
// anything outside printable ASCII becomes '?', and the result is cut to limit
// runes. It does not redact the key; callers redact first (see Prober.safe).
func sanitize(s string, limit int) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= limit {
			b.WriteString("...")
			break
		}
		if r < 0x20 || r > 0x7e {
			r = '?'
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

// parseSSEStream parses an SSE response body. Server text kept in Errors has
// key redacted from it before it is truncated.
func parseSSEStream(body []byte, key string) (*ParsedSSE, error) {
	events, err := ParseSSEStream(body)
	if err != nil {
		return nil, err
	}

	result := &ParsedSSE{
		Usage: make(map[string]interface{}),
	}

	// Track partial JSON per block index.
	toolInputPartial := make(map[int]string)

	// safe redacts the key before sanitising, as Prober.safe does.
	safe := func(s string, n int) string { return sanitize(redactKey(s, key), n) }

	// decodeFailed records an event payload that did not decode.
	decodeFailed := func(event string, err error) {
		result.Errors = append(result.Errors, fmt.Sprintf("%s event: invalid JSON: %s", event, safe(err.Error(), 200)))
	}

	for _, event := range events {
		switch event.Event {
		case "error":
			// Handle error events (N8).
			var errEvent struct {
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(event.Data, &errEvent); err != nil {
				decodeFailed("error", err)
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("stream error event (%s): %s",
					safe(errEvent.Error.Type, 100), safe(errEvent.Error.Message, 500)))
			}

		case "message_start":
			var msg struct {
				Message struct {
					Usage map[string]interface{} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal(event.Data, &msg); err != nil {
				decodeFailed(event.Event, err)
			} else {
				for k, v := range msg.Message.Usage {
					result.Usage[k] = v
				}
			}

		case "message_delta":
			var delta struct {
				Delta struct {
					StopReason string `json:"stop_reason"`
				} `json:"delta"`
				Usage map[string]interface{} `json:"usage"`
			}
			if err := json.Unmarshal(event.Data, &delta); err != nil {
				decodeFailed(event.Event, err)
			} else {
				if delta.Delta.StopReason != "" {
					result.StopReason = delta.Delta.StopReason
				}
				for k, v := range delta.Usage {
					result.Usage[k] = v
				}
			}

		case "content_block_start":
			var start struct {
				Index        int `json:"index"`
				ContentBlock struct {
					Type      string `json:"type"`
					ID        string `json:"id"`
					Name      string `json:"name"`
					Thinking  string `json:"thinking"`
					Signature string `json:"signature"`
					Data      string `json:"data"`
				} `json:"content_block"`
			}
			if err := json.Unmarshal(event.Data, &start); err != nil {
				decodeFailed(event.Event, err)
			} else {
				block := SSEContentBlock{
					Type:  start.ContentBlock.Type,
					Index: start.Index,
				}
				switch start.ContentBlock.Type {
				case "tool_use":
					block.ToolUse = &SSEToolUse{
						ID:    start.ContentBlock.ID,
						Name:  start.ContentBlock.Name,
						Input: make(map[string]interface{}),
					}
				case "thinking":
					block.Thinking = &SSEThinking{
						Thinking:  start.ContentBlock.Thinking,
						Signature: start.ContentBlock.Signature,
					}
				case "redacted_thinking":
					block.RedactedThinking = &SSERedactedThinking{
						Data: start.ContentBlock.Data,
					}
				}
				result.ContentBlocks = append(result.ContentBlocks, block)
			}

		case "content_block_delta":
			var delta struct {
				Index int `json:"index"`
				Delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					PartialJSON string `json:"partial_json"`
					Thinking    string `json:"thinking"`
					Signature   string `json:"signature"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(event.Data, &delta); err != nil {
				decodeFailed(event.Event, err)
			} else {
				// Find or update the content block at this index
				for i := range result.ContentBlocks {
					if result.ContentBlocks[i].Index == delta.Index {
						switch delta.Delta.Type {
						case "text_delta":
							result.ContentBlocks[i].Text += delta.Delta.Text
						case "input_json_delta":
							if result.ContentBlocks[i].ToolUse != nil {
								// Accumulate partial JSON.
								toolInputPartial[delta.Index] += delta.Delta.PartialJSON
							}
						case "thinking_delta":
							if result.ContentBlocks[i].Thinking != nil {
								result.ContentBlocks[i].Thinking.Thinking += delta.Delta.Thinking
							}
						case "signature_delta":
							if result.ContentBlocks[i].Thinking != nil {
								result.ContentBlocks[i].Thinking.Signature += delta.Delta.Signature
							}
						}
						break
					}
				}
			}

		case "content_block_stop":
			var stop struct {
				Index int `json:"index"`
			}
			if err := json.Unmarshal(event.Data, &stop); err != nil {
				decodeFailed(event.Event, err)
			} else {
				// Parse accumulated partial JSON for tool use.
				for i := range result.ContentBlocks {
					if result.ContentBlocks[i].Index == stop.Index && result.ContentBlocks[i].ToolUse != nil {
						if partial := toolInputPartial[stop.Index]; partial != "" {
							var parsed map[string]interface{}
							if err := json.Unmarshal([]byte(partial), &parsed); err != nil {
								result.Errors = append(result.Errors, fmt.Sprintf("tool_use block %d: invalid partial_json: %s",
									stop.Index, safe(err.Error(), 200)))
							} else {
								result.ContentBlocks[i].ToolUse.Input = parsed
							}
						}
						break
					}
				}
			}
		}
	}

	return result, nil
}

// getS2ToolsDefinition returns the shared tools definition for S2 (and S3).
func getS2ToolsDefinition() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "read_file",
			"description": "Read a file from the repository",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

// getS2UserMessage returns the shared user message for S2 (and S3).
func getS2UserMessage() string {
	return "Use the read_file tool to read README.md. You must call the tool before answering."
}

// buildScenarios constructs the list of scenarios to run.
func buildScenarios(models []string) []Scenario {
	var scenarios []Scenario

	// S0: GET /models (once).
	scenarios = append(scenarios, Scenario{
		Model:    "",
		Scenario: "S0-models",
		Method:   "GET",
		Path:     "/models",
		Request:  nil,
	})

	for idx, model := range models {
		// S1: Text completion.
		scenarios = append(scenarios, Scenario{
			Model:    model,
			Scenario: "S1-text",
			Method:   "POST",
			Path:     "/messages",
			Request: map[string]interface{}{
				"model":      model,
				"max_tokens": 64,
				"system":     "You are a helpful assistant.",
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Reply with exactly the word: pong"},
				},
				"stream": true,
			},
		})

		// S2: Tool call.
		scenarios = append(scenarios, Scenario{
			Model:        model,
			Scenario:     "S2-tool-call",
			SessionGroup: "tool-conversation",
			Method:       "POST",
			Path:         "/messages",
			Request: map[string]interface{}{
				"model":      model,
				"max_tokens": 512,
				"system":     "You are a helpful assistant with access to file tools.",
				"messages": []map[string]interface{}{
					{"role": "user", "content": getS2UserMessage()},
				},
				"tools":  getS2ToolsDefinition(),
				"stream": true,
			},
		})

		// S3: Tool result round trip (skip if S2 had no tool call).
		// Capture model in closure for the Build function.
		currentModel := model
		scenarios = append(scenarios, Scenario{
			Model:        model,
			Scenario:     "S3-tool-result",
			SessionGroup: "tool-conversation",
			Method:       "POST",
			Path:         "/messages",
			Request:      nil, // Will be generated by Build.
			Build: func(results map[string]map[string]*Response) (interface{}, string) {
				s2 := results[currentModel]["S2-tool-call"]
				switch {
				case s2 == nil:
					return nil, "S2 has no recorded response"
				case s2.Status != 200:
					return nil, fmt.Sprintf("S2 returned status %d", s2.Status)
				case s2.Error != "":
					// The message may be server-derived, so it stays out of the reason.
					return nil, "S2 recorded an error (see its summary entry)"
				case s2.Parsed == nil:
					return nil, "S2 response was not an event stream"
				}

				// Rebuild assistant content blocks in index order, and answer every
				// tool_use with its own tool_result, in the same order: the API rejects
				// a turn that leaves any tool_use unanswered.
				var assistantContent []interface{}
				var toolResults []map[string]interface{}
				for i := range s2.Parsed.ContentBlocks {
					block := &s2.Parsed.ContentBlocks[i]
					switch {
					case block.Thinking != nil:
						assistantContent = append(assistantContent, map[string]interface{}{
							"type":      "thinking",
							"thinking":  block.Thinking.Thinking,
							"signature": block.Thinking.Signature,
						})
					case block.RedactedThinking != nil:
						assistantContent = append(assistantContent, map[string]interface{}{
							"type": "redacted_thinking",
							"data": block.RedactedThinking.Data,
						})
					case block.Text != "":
						assistantContent = append(assistantContent, map[string]interface{}{
							"type": "text",
							"text": block.Text,
						})
					case block.ToolUse != nil:
						assistantContent = append(assistantContent, map[string]interface{}{
							"type":  "tool_use",
							"id":    block.ToolUse.ID,
							"name":  block.ToolUse.Name,
							"input": block.ToolUse.Input,
						})
						toolResults = append(toolResults, map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": block.ToolUse.ID,
							"content":     "# Kirsch\nA terminal coding agent.",
						})
					}
				}

				if len(toolResults) == 0 {
					return nil, "S2 returned no tool_use"
				}

				// Build S3 request.
				body := map[string]interface{}{
					"model":      currentModel,
					"max_tokens": 512,
					"stream":     true,
					"tools":      getS2ToolsDefinition(),
					"messages": []map[string]interface{}{
						{"role": "user", "content": getS2UserMessage()},
						{"role": "assistant", "content": assistantContent},
						{"role": "user", "content": toolResults},
					},
				}
				return body, ""
			},
		})

		// S4: Cache (two requests with same ~20KB filler system prompt).
		cacheSystemPrompt := generateCachePayload(20000)
		scenarios = append(scenarios, Scenario{
			Model:        model,
			Scenario:     "S4-cache-1",
			SessionGroup: "cache-pair",
			Method:       "POST",
			Path:         "/messages",
			Request: map[string]interface{}{
				"model":      model,
				"max_tokens": 256,
				"system": []map[string]interface{}{
					{
						"type": "text",
						"text": cacheSystemPrompt,
						"cache_control": map[string]interface{}{
							"type": "ephemeral",
						},
					},
				},
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Hello, world."},
				},
				"stream": true,
			},
		})

		scenarios = append(scenarios, Scenario{
			Model:        model,
			Scenario:     "S4-cache-2",
			SessionGroup: "cache-pair",
			Method:       "POST",
			Path:         "/messages",
			Request: map[string]interface{}{
				"model":      model,
				"max_tokens": 256,
				"system": []map[string]interface{}{
					{
						"type": "text",
						"text": cacheSystemPrompt,
						"cache_control": map[string]interface{}{
							"type": "ephemeral",
						},
					},
				},
				"messages": []map[string]interface{}{
					{"role": "user", "content": "Hello, world."},
				},
				"stream": true,
			},
		})

		// S5: Thinking forms (four variants).
		thinkingVariants := []map[string]interface{}{
			nil, // (a) no thinking field
			{"type": "between_tools"},
			{"type": "disabled"},
			{"type": "enabled", "budget_tokens": 1024},
		}
		thinkingNames := []string{"S5a-no-thinking", "S5b-between-tools", "S5c-disabled", "S5d-enabled"}

		for j, thinking := range thinkingVariants {
			req := map[string]interface{}{
				"model":      model,
				"max_tokens": 2048,
				"messages": []map[string]interface{}{
					{"role": "user", "content": "What is 17 × 23? Answer briefly."},
				},
				"stream": true,
			}
			if thinking != nil {
				req["thinking"] = thinking
			}

			scenarios = append(scenarios, Scenario{
				Model:    model,
				Scenario: thinkingNames[j],
				Method:   "POST",
				Path:     "/messages",
				Request:  req,
			})
		}

		// S6: Version header test (first model only).
		if idx == 0 {
			scenarios = append(scenarios, Scenario{
				Model:    model,
				Scenario: "S6-no-version-header",
				Method:   "POST",
				Path:     "/messages",
				Request: map[string]interface{}{
					"model":      model,
					"max_tokens": 64,
					"system":     "You are a helpful assistant.",
					"messages": []map[string]interface{}{
						{"role": "user", "content": "Reply with exactly the word: pong"},
					},
					"stream": true,
				},
			})
		}
	}

	return scenarios
}

// generateCachePayload generates a deterministic filler payload of approximately size bytes.
func generateCachePayload(size int) string {
	var buf bytes.Buffer
	lineNum := 1
	for buf.Len() < size {
		fmt.Fprintf(&buf, "Line %d: Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.\n", lineNum)
		lineNum++
	}
	s := buf.String()
	if len(s) > size {
		s = s[:size]
	}
	return s
}

// validateBaseURL checks that baseURL is either https:// or a literal loopback host.
func validateBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base_url: %v", err)
	}

	// Extract host (removing userinfo).
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("base_url has no host")
	}

	// Check for https or loopback.
	isHTTPS := u.Scheme == "https"
	isLoopback := isLoopbackHost(host)

	if !isHTTPS && !isLoopback {
		return fmt.Errorf("base_url must use https:// or be a literal loopback host (localhost, 127.0.0.0/8, ::1); got %s", u.Scheme)
	}

	return nil
}

// isLoopbackHost checks if host is localhost, in 127.0.0.0/8, or ::1.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}

// filterHeaders removes Set-Cookie and returns a copy.
func filterHeaders(h http.Header) http.Header {
	filtered := make(http.Header)
	for k, v := range h {
		if k != "Set-Cookie" && k != "set-cookie" {
			filtered[k] = v
		}
	}
	return filtered
}

// WriteResults saves probe output to disk (F6 - comprehensive).
func WriteResults(outDir string, models []string, results map[string]map[string]*Response, scenarios []Scenario, key, auth string) error {
	// Create output directory.
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("creating output directory: %v", err)
	}

	// Build scenario index for lookups.
	scenariosByID := make(map[string]Scenario)
	for _, sc := range scenarios {
		scenariosByID[sc.Scenario] = sc
	}

	// Collect summary data.
	summaryData := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"auth":      auth,
		"models":    len(models),
		"scenarios": len(scenarios),
		"results":   make(map[string]interface{}),
	}

	// Write S0 results under _global/ (F6).
	if globalResponses, ok := results["_global"]; ok && len(globalResponses) > 0 {
		globalDir := filepath.Join(outDir, "_global")
		if err := os.MkdirAll(globalDir, 0o750); err != nil {
			return fmt.Errorf("creating global directory: %v", err)
		}

		for _, scenarioName := range []string{"S0-models"} {
			if resp, ok := globalResponses[scenarioName]; ok {

				// Write S0 status file.
				statusFile := filepath.Join(globalDir, scenarioName+".status")
				if err := writeFile(statusFile, []byte(fmt.Sprintf("%d\n", resp.Status)), key); err != nil {
					return err
				}

				// Write headers.
				headersFile := filepath.Join(globalDir, scenarioName+".headers.json")
				headersJSON, err := json.MarshalIndent(resp.Headers, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling headers: %v", err)
				}
				if err := writeFile(headersFile, headersJSON, key); err != nil {
					return err
				}

				// Write body.
				sseFile := filepath.Join(globalDir, scenarioName+".sse")
				if err := writeFile(sseFile, resp.Body, key); err != nil {
					return err
				}

				// Write the session ID.
				if err := writeSession(globalDir, scenarioName, resp.Session, key); err != nil {
					return err
				}
			}
		}
	}

	// Write per-model results.
	modelSummaries := make(map[string]map[string]interface{})
	for _, model := range models {
		modelResponses := results[model]
		if len(modelResponses) == 0 {
			continue
		}

		modelDir := filepath.Join(outDir, model)
		if err := os.MkdirAll(modelDir, 0o750); err != nil {
			return fmt.Errorf("creating model directory: %v", err)
		}

		modelSummaries[model] = make(map[string]interface{})

		// Sort scenarios for consistent output.
		var scenarioNames []string
		for name := range modelResponses {
			scenarioNames = append(scenarioNames, name)
		}
		sort.Strings(scenarioNames)

		for _, scenarioName := range scenarioNames {
			resp := modelResponses[scenarioName]

			// Write request body (F5).
			if resp.RequestBody != nil {
				requestFile := filepath.Join(modelDir, scenarioName+".request.json")
				if err := writeFile(requestFile, resp.RequestBody, key); err != nil {
					return err
				}
			}

			// Write status.
			statusFile := filepath.Join(modelDir, scenarioName+".status")
			if err := writeFile(statusFile, []byte(fmt.Sprintf("%d\n", resp.Status)), key); err != nil {
				return err
			}

			// Write headers.
			headersFile := filepath.Join(modelDir, scenarioName+".headers.json")
			headersJSON, err := json.MarshalIndent(resp.Headers, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling headers: %v", err)
			}
			if err := writeFile(headersFile, headersJSON, key); err != nil {
				return err
			}

			// Write the raw response. Stop bodies were already truncated by Run.
			sseFile := filepath.Join(modelDir, scenarioName+".sse")
			if err := writeFile(sseFile, resp.Body, key); err != nil {
				return err
			}

			if err := writeSession(modelDir, scenarioName, resp.Session, key); err != nil {
				return err
			}

			// Collect summary for this scenario.
			scenarioSummary := map[string]interface{}{
				"status": resp.Status,
			}
			if resp.Session != "" {
				scenarioSummary["session"] = resp.Session
			}
			if resp.Parsed != nil {
				scenarioSummary["ttfb_ms"] = resp.TTFBMS   // N10: Use recorded TTFB
				scenarioSummary["total_ms"] = resp.TotalMS // N10: Use recorded total
				scenarioSummary["usage"] = resp.Parsed.Usage
				scenarioSummary["stop_reason"] = resp.Parsed.StopReason
				// Count blocks
				thinkingCount := 0
				redactedCount := 0
				toolUseOk := false
				for _, block := range resp.Parsed.ContentBlocks {
					if block.Thinking != nil {
						thinkingCount++
					}
					if block.RedactedThinking != nil {
						redactedCount++
					}
					if block.ToolUse != nil && block.ToolUse.ID != "" && block.ToolUse.Name != "" && len(block.ToolUse.Input) > 0 {
						toolUseOk = true
					}
				}
				scenarioSummary["thinking_blocks"] = thinkingCount
				scenarioSummary["redacted_thinking"] = redactedCount
				if toolUseOk {
					scenarioSummary["tool_use_ok"] = true
				}
			}
			if resp.Error != "" {
				scenarioSummary["error"] = resp.Error
			}
			modelSummaries[model][scenarioName] = scenarioSummary
		}
	}
	summaryData["results"] = modelSummaries

	// Write summary.json (F6).
	summaryFile := filepath.Join(outDir, "summary.json")
	summaryJSON, err := json.MarshalIndent(summaryData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling summary: %v", err)
	}
	if err := writeFile(summaryFile, summaryJSON, key); err != nil {
		return err
	}

	// Generate and write SUMMARY.md (F6).
	summaryMD := generateSummaryMarkdown(models, results, auth)
	summaryMDFile := filepath.Join(outDir, "SUMMARY.md")
	if err := writeFile(summaryMDFile, []byte(summaryMD), key); err != nil {
		return err
	}

	return nil
}

// generateSummaryMarkdown creates a summary markdown table (F6).
func generateSummaryMarkdown(models []string, results map[string]map[string]*Response, auth string) string {
	var buf bytes.Buffer
	hasStreamError := false
	fmt.Fprintf(&buf, "# OpenCode Go Probe Results (auth: %s)\n\n", auth)
	buf.WriteString("| Model | S1 | S2 | S3 | S4-1 | S4-2 | S5a | S5b | S5c | S5d | S6 |\n")
	buf.WriteString("|-------|----|----|----|----|----|----|----|----|----|----|----|\n")

	for _, model := range models {
		modelResults := results[model]
		fmt.Fprintf(&buf, "| %s", model)

		scenarios := []string{"S1-text", "S2-tool-call", "S3-tool-result", "S4-cache-1", "S4-cache-2", "S5a-no-thinking", "S5b-between-tools", "S5c-disabled", "S5d-enabled", "S6-no-version-header"}
		for _, sc := range scenarios {
			if resp, ok := modelResults[sc]; ok {
				if resp.Status == 200 && resp.Error != "" {
					fmt.Fprintf(&buf, " | %d*", resp.Status)
					hasStreamError = true
				} else {
					fmt.Fprintf(&buf, " | %d", resp.Status)
				}
			} else {
				buf.WriteString(" | -")
			}
		}
		buf.WriteString(" |\n")
	}

	if hasStreamError {
		buf.WriteString("\n\\* 200 response whose stream carried an error or an undecodable event; see `error` in summary.json.\n")
	}

	return buf.String()
}

// writeSession records the session ID a scenario was sent with, as
// <scenario>.session. A scenario with no ID writes nothing.
func writeSession(dir, scenario, session, key string) error {
	if session == "" {
		return nil
	}
	return writeFile(filepath.Join(dir, scenario+".session"), []byte(session+"\n"), key)
}

// writeFile writes data to a file after checking for key leakage.
func writeFile(path string, data []byte, key string) error {
	if key != "" && bytes.Contains(data, []byte(key)) {
		return fmt.Errorf("refusing to write %s: contains API key", path)
	}
	return os.WriteFile(path, data, 0o644) // #nosec G306 -- probe output is not sensitive
}
