package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Shared helpers.
// ---------------------------------------------------------------------------

const (
	fixtureToolID    = "toolu_01T1x1fJ34qAmk2tNTrN7Up6"
	fixtureToolName  = "get_weather"
	fixtureToolPlace = "San Francisco, CA"
)

// testOpts returns options for a run against baseURL with no pause between
// requests.
func testOpts(baseURL, outDir string, models ...string) options {
	return options{
		run:     true,
		baseURL: baseURL,
		models:  models,
		outDir:  outDir,
		key:     "test-key",
		pause:   0,
		getenv:  os.Getenv,
	}
}

// mustReadFile reads a test fixture or fails the test.
func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return data
}

// requestMessages returns the "messages" array of a request body, or nil.
func requestMessages(body []byte) []interface{} {
	var req struct {
		Messages []interface{} `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}
	return req.Messages
}

// isS2Request reports whether body is the S2 tool-call request: a single user
// message that asks for read_file.
func isS2Request(body []byte) bool {
	messages := requestMessages(body)
	if len(messages) != 1 {
		return false
	}
	msg, ok := messages[0].(map[string]interface{})
	if !ok {
		return false
	}
	content, ok := msg["content"].(string)
	return ok && strings.Contains(content, "read_file")
}

// readJSONMap reads a JSON object from a file or fails the test.
func readJSONMap(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(mustReadFile(t, path), &m); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return m
}

// scenarioSummary returns summary.json's entry for one model and scenario.
func scenarioSummary(t *testing.T, outDir, model, scenario string) map[string]interface{} {
	t.Helper()
	summary := readJSONMap(t, filepath.Join(outDir, "summary.json"))
	results, ok := summary["results"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary.json has no results object: %v", summary)
	}
	modelResults, ok := results[model].(map[string]interface{})
	if !ok {
		t.Fatalf("summary.json has no results for model %q: %v", model, results)
	}
	entry, ok := modelResults[scenario].(map[string]interface{})
	if !ok {
		t.Fatalf("summary.json has no entry for %s/%s: %v", model, scenario, modelResults)
	}
	return entry
}

// assertNoFileContains fails the test if any file under dir contains needle.
func assertNoFileContains(t *testing.T, dir, needle string) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Contains(data, []byte(needle)) {
			t.Errorf("file %s contains %q", path, needle)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The expected thinking text and signature are computed with regexps over the
// fixture text, independently of ParseSSEStream/parseSSEStream, so a parser
// bug cannot make both sides of an assertion wrong in the same way.
var (
	thinkingDeltaRe  = regexp.MustCompile(`"type":\s*"thinking_delta",\s*"thinking":\s*("(?:[^"\\]|\\.)*")`)
	signatureDeltaRe = regexp.MustCompile(`"type":\s*"signature_delta",\s*"signature":\s*("(?:[^"\\]|\\.)*")`)
)

// expectedThinking returns the concatenated thinking_delta text and the
// signature_delta value found in a fixture, and fails the test if either is
// empty or the thinking text is not the product of several deltas.
func expectedThinking(t *testing.T, fixture []byte) (thinking, signature string) {
	t.Helper()
	deltas := thinkingDeltaRe.FindAllSubmatch(fixture, -1)
	if len(deltas) < 2 {
		t.Fatalf("fixture has %d thinking_delta events, want at least 2", len(deltas))
	}
	for _, m := range deltas {
		var s string
		if err := json.Unmarshal(m[1], &s); err != nil {
			t.Fatalf("decoding thinking delta %s: %v", m[1], err)
		}
		thinking += s
	}
	sigs := signatureDeltaRe.FindAllSubmatch(fixture, -1)
	if len(sigs) != 1 {
		t.Fatalf("fixture has %d signature_delta events, want 1", len(sigs))
	}
	if err := json.Unmarshal(sigs[0][1], &signature); err != nil {
		t.Fatalf("decoding signature %s: %v", sigs[0][1], err)
	}
	if thinking == "" {
		t.Fatal("expected thinking text is empty")
	}
	if signature == "" {
		t.Fatal("expected signature is empty")
	}
	return thinking, signature
}

// ---------------------------------------------------------------------------
// T1: Key lookup from environment.
// ---------------------------------------------------------------------------

func TestKeyLookup(t *testing.T) {
	tests := []struct {
		name     string
		prefixed string
		unpref   string
		want     string
	}{
		{name: "both set, prefixed wins", prefixed: "pref-key", unpref: "unpref-key", want: "pref-key"},
		{name: "only unprefixed", prefixed: "", unpref: "unpref-key", want: "unpref-key"},
		{name: "only prefixed", prefixed: "pref-key", unpref: "", want: "pref-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				switch key {
				case "KIRSCH_OPENCODE_API_KEY":
					return tt.prefixed
				case "OPENCODE_API_KEY":
					return tt.unpref
				}
				return ""
			}
			if got := lookupKey(getenv); got != tt.want {
				t.Errorf("lookupKey got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("no key set", func(t *testing.T) {
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(200)
		}))
		defer server.Close()

		opts := testOpts(server.URL, t.TempDir(), "test-model")
		opts.key = ""

		err := run(context.Background(), opts, io.Discard)
		if err == nil {
			t.Fatal("expected error for missing key")
		}
		if !strings.Contains(err.Error(), "KIRSCH_OPENCODE_API_KEY") || !strings.Contains(err.Error(), "OPENCODE_API_KEY") {
			t.Errorf("error should name both variables: %v", err)
		}
		if n := hits.Load(); n != 0 {
			t.Errorf("expected 0 hits, got %d", n)
		}
	})
}

// ---------------------------------------------------------------------------
// T2: The key never reaches files, stdout or the returned error.
// ---------------------------------------------------------------------------

func TestKeyNoLeak(t *testing.T) {
	const key = "SENTINEL-KEY-7f3a"
	const canned = `event: message_start
data: {"type":"message_start","message":{"id":"test","type":"message"}}

event: message_stop
data: {"type":"message_stop"}

`
	runWith := func(t *testing.T, handler http.HandlerFunc) (dir string, stdout *bytes.Buffer, err error) {
		t.Helper()
		server := httptest.NewServer(handler)
		defer server.Close()
		dir = t.TempDir()
		stdout = &bytes.Buffer{}
		opts := testOpts(server.URL, dir, "test-model")
		opts.key = key
		return dir, stdout, run(context.Background(), opts, stdout)
	}

	t.Run("server never echoes the key", func(t *testing.T) {
		dir, stdout, err := runWith(t, func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("x-api-key"); got != key {
				t.Errorf("key not sent correctly")
			}
			fmt.Fprint(w, canned)
		})
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}
		if bytes.Contains(stdout.Bytes(), []byte(key)) {
			t.Error("stdout contains API key")
		}
		assertNoFileContains(t, dir, key)
	})

	// When the server reflects the key back, the write guard is the only thing
	// between the key and the disk. The run must fail, name the guard, and
	// leave no file holding the key.
	echoes := map[string]http.HandlerFunc{
		"server echoes the key in the body": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "event: message_start\ndata: {\"echo\":%q}\n\n", r.Header.Get("x-api-key"))
		},
		"server echoes the key in a header": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Echo", r.Header.Get("x-api-key"))
			fmt.Fprint(w, canned)
		},
	}
	for name, handler := range echoes {
		t.Run(name, func(t *testing.T) {
			dir, stdout, err := runWith(t, handler)
			if err == nil {
				t.Fatal("expected the write guard to fail the run")
			}
			if !strings.Contains(err.Error(), "contains API key") {
				t.Errorf("error should name the key guard, got: %v", err)
			}
			if strings.Contains(err.Error(), key) {
				t.Error("returned error contains the API key")
			}
			if bytes.Contains(stdout.Bytes(), []byte(key)) {
				t.Error("stdout contains API key")
			}
			assertNoFileContains(t, dir, key)
		})
	}
}

// ---------------------------------------------------------------------------
// T3: Redirects are not followed.
// ---------------------------------------------------------------------------

func TestNoRedirectFollow(t *testing.T) {
	var targetHits, serverAHits atomic.Int32
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits.Add(1)
		w.WriteHeader(200)
	}))
	defer targetServer.Close()

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverAHits.Add(1)
		http.Redirect(w, r, targetServer.URL, http.StatusFound)
	}))
	defer serverA.Close()

	tmpDir := t.TempDir()
	err := run(context.Background(), testOpts(serverA.URL, tmpDir, "test-model"), io.Discard)
	if err == nil {
		t.Fatal("expected error for redirect")
	}

	if n := serverAHits.Load(); n != 1 {
		t.Errorf("server A hit count: got %d, want 1", n)
	}
	if n := targetHits.Load(); n != 0 {
		t.Errorf("server B hit count: got %d, want 0", n)
	}

	targetURL, err2 := url.Parse(targetServer.URL)
	if err2 != nil {
		t.Fatal(err2)
	}
	if !strings.Contains(err.Error(), targetURL.Host) {
		t.Errorf("error should contain target host %q, got: %v", targetURL.Host, err)
	}

	statusData := mustReadFile(t, filepath.Join(tmpDir, "_global", "S0-models.status"))
	if strings.TrimSpace(string(statusData)) != "302" {
		t.Errorf("S0-models.status = %q, want 302", statusData)
	}
}

// TestRedirectLocationNotEchoed checks what a redirect's Location may reach.
// The terminal and the summary get a host (or a fixed stand-in) and never a
// path or query. headers.json deliberately keeps the full Location, as the
// evidence record; the README documents that decision.
func TestRedirectLocationNotEchoed(t *testing.T) {
	tests := []struct {
		name      string
		location  string
		wantHost  string
		forbidden []string
		inHeaders bool
	}{
		{"absolute", "https://other.example/private/path?token=abc#frag", "other.example", []string{"private", "token", "frag"}, true},
		{"relative", "/secret/path?token=abc", "<no-host>", []string{"secret", "token"}, true},
		{"unparseable", "http://[::1", "<unparseable>", []string{"[::1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", tt.location)
				w.WriteHeader(http.StatusFound)
			}))
			defer server.Close()

			tmpDir := t.TempDir()
			var stdout bytes.Buffer
			err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), &stdout)
			if err == nil {
				t.Fatal("expected the run to stop")
			}
			combined := err.Error() + "\n" + stdout.String()
			if !strings.Contains(err.Error(), tt.wantHost) {
				t.Errorf("error should name %q, got: %v", tt.wantHost, err)
			}
			for _, bad := range tt.forbidden {
				if strings.Contains(combined, bad) {
					t.Errorf("output leaks %q: %s", bad, combined)
				}
			}
			if tt.inHeaders {
				headers := string(mustReadFile(t, filepath.Join(tmpDir, "_global", "S0-models.headers.json")))
				if !strings.Contains(headers, tt.location) {
					t.Errorf("headers.json should keep the full Location %q, got: %s", tt.location, headers)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T4: 429 stops the probe and writes results.
// ---------------------------------------------------------------------------

func TestStop429WithResults(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 3 {
			w.Header().Set("retry-after", "60")
			w.WriteHeader(429)
			fmt.Fprint(w, `{"error":"rate limit exceeded"}`)
			return
		}
		w.WriteHeader(200)
		fmt.Fprint(w, "event: message_stop\ndata: {}\n\n")
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), &stdout)
	if err == nil {
		t.Fatal("expected error when 429 received")
	}
	if n := requests.Load(); n != 3 {
		t.Errorf("expected 3 hits, got %d", n)
	}

	// The third request (S2-tool-call) carries the retry-after header.
	headers := strings.ToLower(string(mustReadFile(t, filepath.Join(tmpDir, "test-model", "S2-tool-call.headers.json"))))
	if !strings.Contains(headers, "retry-after") {
		t.Errorf("headers should contain retry-after, got: %s", headers)
	}

	// Its body is non-empty and at most 4096 bytes.
	sse := mustReadFile(t, filepath.Join(tmpDir, "test-model", "S2-tool-call.sse"))
	if len(sse) == 0 || len(sse) > 4096 {
		t.Errorf("S2-tool-call.sse length = %d, want 1..4096", len(sse))
	}

	// What was collected before the stop is on disk.
	for _, path := range []string{
		filepath.Join(tmpDir, "_global", "S0-models.sse"),
		filepath.Join(tmpDir, "test-model", "S1-text.sse"),
		filepath.Join(tmpDir, "SUMMARY.md"),
		filepath.Join(tmpDir, "summary.json"),
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("%s should exist after a stop: %v", path, statErr)
		}
	}

	// The summary table was printed.
	if !strings.Contains(stdout.String(), "OpenCode Go Probe Results") {
		t.Errorf("stdout should contain the summary table, got: %s", stdout.String())
	}
}

// TestStopBodyTruncated checks that a stopping body is cut to 4KB on disk for
// every stop status, and that nothing else is cut.
func TestStopBodyTruncated(t *testing.T) {
	for _, status := range []int{http.StatusFound, 401, 403, 429} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			big := strings.Repeat("x", 10000)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if status == http.StatusFound {
					w.Header().Set("Location", "https://other.example/")
				}
				w.WriteHeader(status)
				fmt.Fprint(w, big)
			}))
			defer server.Close()

			tmpDir := t.TempDir()
			if err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), io.Discard); err == nil {
				t.Fatal("expected the run to stop")
			}
			sse := mustReadFile(t, filepath.Join(tmpDir, "_global", "S0-models.sse"))
			if len(sse) != 4096 {
				t.Errorf("stop body length = %d, want 4096", len(sse))
			}
		})
	}
}

// TestRunAndWriteErrorsJoined checks that a run error and a write error are
// both reported, rather than one hiding the other.
func TestRunAndWriteErrorsJoined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer server.Close()

	// An output directory beneath a regular file cannot be created.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run(context.Background(), testOpts(server.URL, filepath.Join(blocker, "out"), "test-model"), io.Discard)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"probe stopped: received 429", "writing results"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should contain %q, got: %v", want, err)
		}
	}
}

// ---------------------------------------------------------------------------
// T5: The write guard refuses data containing the key.
// ---------------------------------------------------------------------------

func TestWriteGuard(t *testing.T) {
	tmpDir := t.TempDir()
	key := "secret-key-xyz"

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := writeFile(filePath, []byte("This contains "+key+" in it"), key); err == nil {
		t.Error("writeFile should have refused to write data containing key")
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should not have been created")
	}

	// Clean data is written.
	if err := writeFile(filePath, []byte("clean"), key); err != nil {
		t.Errorf("writeFile refused clean data: %v", err)
	}
}

// ---------------------------------------------------------------------------
// T6: The SSE parser parses real fixtures correctly.
// ---------------------------------------------------------------------------

func TestSSEParserWithFixtures(t *testing.T) {
	toolUseData := mustReadFile(t, "testdata/tool_use_stream.sse")
	parsed, err := parseSSEStream(toolUseData, "")
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if len(parsed.Errors) != 0 {
		t.Errorf("tool_use fixture produced parse errors: %v", parsed.Errors)
	}
	if len(parsed.ContentBlocks) < 2 {
		t.Fatalf("expected at least 2 content blocks, got %d", len(parsed.ContentBlocks))
	}
	toolBlock := parsed.ContentBlocks[1]
	if toolBlock.ToolUse == nil {
		t.Fatal("expected tool use block at index 1")
	}
	if toolBlock.ToolUse.ID != fixtureToolID {
		t.Errorf("tool use ID: got %q, want %q", toolBlock.ToolUse.ID, fixtureToolID)
	}
	if toolBlock.ToolUse.Name != fixtureToolName {
		t.Errorf("tool use name: got %q, want %q", toolBlock.ToolUse.Name, fixtureToolName)
	}
	if location, ok := toolBlock.ToolUse.Input["location"].(string); !ok || location != fixtureToolPlace {
		t.Errorf("tool input location: got %v, want %q", toolBlock.ToolUse.Input["location"], fixtureToolPlace)
	}
	if inputTokens, ok := parsed.Usage["input_tokens"].(float64); !ok || int(inputTokens) != 472 {
		t.Errorf("input_tokens: got %v, want 472", parsed.Usage["input_tokens"])
	}
	if outputTokens, ok := parsed.Usage["output_tokens"].(float64); !ok || int(outputTokens) != 89 {
		t.Errorf("output_tokens: got %v, want 89", parsed.Usage["output_tokens"])
	}

	thinkingData := mustReadFile(t, "testdata/thinking_stream.sse")
	thinkingParsed, err := parseSSEStream(thinkingData, "")
	if err != nil {
		t.Fatalf("parseSSEStream for thinking failed: %v", err)
	}
	wantThinking, wantSignature := expectedThinking(t, thinkingData)
	if len(thinkingParsed.ContentBlocks) < 1 {
		t.Fatal("expected at least 1 content block for thinking")
	}
	thinkingBlock := thinkingParsed.ContentBlocks[0]
	if thinkingBlock.Thinking == nil {
		t.Fatal("expected a thinking block at index 0")
	}
	if thinkingBlock.Thinking.Thinking != wantThinking {
		t.Errorf("thinking text:\ngot:  %q\nwant: %q", thinkingBlock.Thinking.Thinking, wantThinking)
	}
	if thinkingBlock.Thinking.Signature != wantSignature {
		t.Errorf("signature:\ngot:  %q\nwant: %q", thinkingBlock.Thinking.Signature, wantSignature)
	}

	redactedParsed, err := parseSSEStream(mustReadFile(t, "testdata/constructed_redacted_cache.sse"), "")
	if err != nil {
		t.Fatalf("parseSSEStream for redacted failed: %v", err)
	}
	if len(redactedParsed.ContentBlocks) < 1 || redactedParsed.ContentBlocks[0].RedactedThinking == nil {
		t.Fatal("expected redacted_thinking block at index 0")
	}
	if got := redactedParsed.ContentBlocks[0].RedactedThinking.Data; got != "reasoning_content_hash_abc123" {
		t.Errorf("redacted data: got %q, want %q", got, "reasoning_content_hash_abc123")
	}
	if v, ok := redactedParsed.Usage["cache_creation_input_tokens"].(float64); !ok || int(v) != 80 {
		t.Errorf("cache_creation_input_tokens: got %v, want 80", redactedParsed.Usage["cache_creation_input_tokens"])
	}
	if v, ok := redactedParsed.Usage["cache_read_input_tokens"].(float64); !ok || int(v) != 20 {
		t.Errorf("cache_read_input_tokens: got %v, want 20", redactedParsed.Usage["cache_read_input_tokens"])
	}
}

// TestParseSSEErrorEvent checks that an error event is recorded, not dropped.
func TestParseSSEErrorEvent(t *testing.T) {
	parsed, err := parseSSEStream(mustReadFile(t, "testdata/error_stream.sse"), "")
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	want := []string{"stream error event (overloaded_error): Overloaded"}
	if !reflect.DeepEqual(parsed.Errors, want) {
		t.Errorf("Errors = %q, want %q", parsed.Errors, want)
	}
}

// TestParseSSEMalformedPartialJSON checks that a tool_use whose streamed input
// does not parse is recorded as an error and leaves the input empty.
func TestParseSSEMalformedPartialJSON(t *testing.T) {
	stream := `event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_x","name":"read_file","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\": \"READ"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}
`
	parsed, err := parseSSEStream([]byte(stream), "")
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if len(parsed.Errors) != 1 || !strings.Contains(parsed.Errors[0], "tool_use block 0: invalid partial_json") {
		t.Fatalf("Errors = %q, want one invalid partial_json entry for block 0", parsed.Errors)
	}
	if got := parsed.ContentBlocks[0].ToolUse.Input; len(got) != 0 {
		t.Errorf("tool input = %v, want empty after a parse failure", got)
	}
}

// TestParseSSEMalformedEventData checks that an event whose data is not JSON
// is recorded rather than silently skipped.
func TestParseSSEMalformedEventData(t *testing.T) {
	parsed, err := parseSSEStream([]byte("event: content_block_start\ndata: {not json\n"), "")
	if err != nil {
		t.Fatalf("parseSSEStream failed: %v", err)
	}
	if len(parsed.Errors) != 1 || !strings.Contains(parsed.Errors[0], "content_block_start event: invalid JSON") {
		t.Errorf("Errors = %q, want one invalid JSON entry", parsed.Errors)
	}
}

// TestStreamErrorsSurfaced runs end to end against a stream that carries an
// error event, and checks the error reaches summary.json, that S3 is skipped
// for that reason, and that the recorded stream is not truncated.
func TestStreamErrorsSurfaced(t *testing.T) {
	body := append(mustReadFile(t, "testdata/error_stream.sse"), []byte(strings.Repeat(": filler\n", 1000))...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write(body)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	if err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), &stdout); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	s1 := scenarioSummary(t, tmpDir, "test-model", "S1-text")
	if msg, _ := s1["error"].(string); !strings.Contains(msg, "overloaded_error") || !strings.Contains(msg, "Overloaded") {
		t.Errorf("S1 summary error = %q, want the stream error", msg)
	}
	if status, _ := s1["status"].(float64); status != 200 {
		t.Errorf("S1 status = %v, want 200", s1["status"])
	}

	s3 := scenarioSummary(t, tmpDir, "test-model", "S3-tool-result")
	if msg, _ := s3["error"].(string); msg != "S2 recorded an error (see its summary entry)" {
		t.Errorf("S3 skip reason = %q", msg)
	}

	saved := mustReadFile(t, filepath.Join(tmpDir, "test-model", "S1-text.sse"))
	if !bytes.Equal(saved, body) {
		t.Errorf("recorded stream was altered: %d bytes saved, %d received", len(saved), len(body))
	}
}

// ---------------------------------------------------------------------------
// T7: Dry run makes no requests.
// ---------------------------------------------------------------------------

func TestDryRun(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	models := splitModels(defaultModels)
	if len(models) != 5 {
		t.Fatalf("default model list has %d models, want 5: %v", len(models), models)
	}
	opts := testOpts(server.URL, "", models...)
	opts.run = false
	opts.key = ""

	var output bytes.Buffer
	if err := run(context.Background(), opts, &output); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("dry run made %d requests; should make 0", n)
	}

	out := output.String()
	// S0 once, then per model: S1, S2, S3, S4 x2, S5 x4 = 9, and S6 for the first.
	if !strings.Contains(out, "Planned 47 requests (auth: x-api-key):") {
		t.Errorf("dry run output should open with the plan size, got: %s", out)
	}
	if !strings.Contains(out, "Total: 47 requests") {
		t.Errorf("dry run output should end with 'Total: 47 requests', got: %s", out)
	}

	planLine := regexp.MustCompile(`^\s+\d+\.\s+(\S+)\s+(S\d\S*)\s+(GET|POST)\s+/\S+$`)
	known := map[string]bool{"_global": true}
	for _, m := range models {
		known[m] = true
	}
	var planLines int
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, " ") {
			continue
		}
		planLines++
		m := planLine.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("plan line does not name model, scenario, method and path: %q", line)
			continue
		}
		if !known[m[1]] {
			t.Errorf("plan line names an unknown model %q: %q", m[1], line)
		}
	}
	if planLines != 47 {
		t.Errorf("plan has %d lines, want 47", planLines)
	}
	if !strings.Contains(out, "_global") {
		t.Error("dry run output should show _global for S0")
	}
}

// TestDryRunShowsAuthMode checks the plan header names the chosen mode.
func TestDryRunShowsAuthMode(t *testing.T) {
	for _, mode := range []string{"x-api-key", "bearer"} {
		opts := testOpts("https://example.com", "", "test-model")
		opts.run = false
		opts.key = ""
		opts.auth = mode
		var out bytes.Buffer
		if err := run(context.Background(), opts, &out); err != nil {
			t.Fatalf("%s: run failed: %v", mode, err)
		}
		if want := "(auth: " + mode + "):"; !strings.Contains(out.String(), want) {
			t.Errorf("%s: dry run should show %q, got: %s", mode, want, out.String())
		}
	}
}

// ---------------------------------------------------------------------------
// Auth header selection and the per-run output directory.
// ---------------------------------------------------------------------------

// TestAuthHeader checks that each mode sends exactly one auth header, carrying
// the key, on every request including S0 /models.
func TestAuthHeader(t *testing.T) {
	const key = "SENTINEL-AUTH-91c2"
	cases := []struct {
		name, mode, present, absent string
		wantValue                   string
	}{
		{"default", "", "x-api-key", "Authorization", key},
		{"x-api-key", "x-api-key", "x-api-key", "Authorization", key},
		{"bearer", "bearer", "Authorization", "x-api-key", "Bearer " + key},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var seen []http.Header
			var paths []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				seen = append(seen, r.Header.Clone())
				paths = append(paths, r.URL.Path)
				mu.Unlock()
				w.WriteHeader(200)
			}))
			defer server.Close()

			dir := t.TempDir()
			opts := testOpts(server.URL, dir, "test-model")
			opts.key = key
			opts.auth = tc.mode
			// Not under test: the S3 skip and stream errors do not stop the run.
			_ = run(context.Background(), opts, io.Discard)

			mu.Lock()
			defer mu.Unlock()
			if len(seen) < 2 {
				t.Fatalf("expected several requests, got %d", len(seen))
			}
			var sawModels bool
			for i, h := range seen {
				if strings.HasSuffix(paths[i], "/models") {
					sawModels = true
				}
				if got := h.Get(tc.present); got != tc.wantValue {
					t.Errorf("request %d (%s): %s = %q, want %q", i, paths[i], tc.present, got, tc.wantValue)
				}
				if _, ok := h[http.CanonicalHeaderKey(tc.absent)]; ok {
					t.Errorf("request %d (%s): %s header must be absent", i, paths[i], tc.absent)
				}
				for name, vals := range h {
					for _, v := range vals {
						if strings.Contains(v, key) && !strings.EqualFold(name, tc.present) {
							t.Errorf("request %d: key carried by unexpected header %s", i, name)
						}
					}
				}
			}
			if !sawModels {
				t.Error("no /models request was observed")
			}

			wantMode := tc.mode
			if wantMode == "" {
				wantMode = "x-api-key"
			}
			summary := readJSONMap(t, filepath.Join(dir, "summary.json"))
			if summary["auth"] != wantMode {
				t.Errorf("summary.json auth = %v, want %q", summary["auth"], wantMode)
			}
			md := string(mustReadFile(t, filepath.Join(dir, "SUMMARY.md")))
			if !strings.Contains(strings.SplitN(md, "\n", 2)[0], "auth: "+wantMode) {
				t.Errorf("SUMMARY.md heading should name the auth mode, got: %q", strings.SplitN(md, "\n", 2)[0])
			}
			assertNoFileContains(t, dir, key)
		})
	}
}

// TestInvalidAuthRefused checks a bad -auth value makes no request and names
// the allowed values, in dry-run and live mode alike.
func TestInvalidAuthRefused(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	for _, live := range []bool{true, false} {
		opts := testOpts(server.URL, t.TempDir(), "test-model")
		opts.run = live
		opts.auth = "basic"
		err := run(context.Background(), opts, io.Discard)
		if err == nil {
			t.Fatalf("run=%v: expected an error for -auth basic", live)
		}
		for _, want := range []string{"x-api-key", "bearer"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("run=%v: error should name %q, got: %v", live, want, err)
			}
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("expected 0 requests, got %d", n)
	}
}

// TestOutDirNonEmptyRefused checks a live run will not write into a directory
// that already holds files, and makes no request when it refuses.
func TestOutDirNonEmptyRefused(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()

	dir := t.TempDir()
	existing := filepath.Join(dir, "earlier.txt")
	if err := os.WriteFile(existing, []byte("earlier run"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), testOpts(server.URL, dir, "test-model"), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected a non-empty refusal, got: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("expected 0 requests, got %d", n)
	}
	if got := string(mustReadFile(t, existing)); got != "earlier run" {
		t.Errorf("existing file was altered: %q", got)
	}

	// An existing, empty directory is fine (t.TempDir creates one).
	if err := run(context.Background(), testOpts(server.URL, t.TempDir(), "test-model"), io.Discard); err != nil && strings.Contains(err.Error(), "not empty") {
		t.Errorf("empty directory should be accepted, got: %v", err)
	}
	if n := hits.Load(); n == 0 {
		t.Error("the empty directory run should have made requests")
	}
}

// TestDefaultOutDir checks the default directory is timestamped to the minute
// in UTC, so a re-run lands in a different directory.
func TestDefaultOutDir(t *testing.T) {
	loc := time.FixedZone("plus2", 2*3600)
	got := defaultOutDir(time.Date(2026, 10, 2, 17, 5, 59, 0, loc))
	want := filepath.Join("testdata", "probe", "opencode", "2026-10-02T1505Z")
	if got != want {
		t.Errorf("defaultOutDir = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// T8: Base URL validation.
// ---------------------------------------------------------------------------

func TestBaseURLValidation(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		wantError bool
	}{
		{"valid https", "https://opencode.ai/zen/go/v1", false},
		{"localhost", "http://localhost:8080/api", false},
		{"127.0.0.1", "http://127.0.0.1:8080/api", false},
		{"::1", "http://[::1]:8080/api", false},
		{"http non-loopback", "http://example.com/api", true},
		{"userinfo evil", "http://localhost@evil.example/api", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseURL(tt.baseURL)
			if (err != nil) != tt.wantError {
				t.Errorf("validateBaseURL(%q) error = %v, wantError = %v", tt.baseURL, err, tt.wantError)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T9: S3 is rebuilt from S2's parsed stream.
// ---------------------------------------------------------------------------

const messageStopOnly = "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

// s3Server serves s2 (with the given content type and status) to the S2
// request and a bare message_stop to everything else. It records every
// request that has three messages, which is how S3 is recognised.
type s3Server struct {
	*httptest.Server
	mu       sync.Mutex
	s3Bodies [][]byte
}

func newS3Server(status int, contentType string, s2 []byte) *s3Server {
	s := &s3Server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case isS2Request(body):
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
			w.WriteHeader(status)
			w.Write(s2)
		case len(requestMessages(body)) == 3:
			s.mu.Lock()
			s.s3Bodies = append(s.s3Bodies, body)
			s.mu.Unlock()
			fmt.Fprint(w, messageStopOnly)
		default:
			fmt.Fprint(w, messageStopOnly)
		}
	}))
	return s
}

func (s *s3Server) bodies() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]byte(nil), s.s3Bodies...)
}

// asMap asserts v is a JSON object.
func asMap(t *testing.T, what string, v interface{}) map[string]interface{} {
	t.Helper()
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("%s is %T, want an object", what, v)
	}
	return m
}

func TestS3Rebuild(t *testing.T) {
	t.Run("S2 with thinking and tool_use", func(t *testing.T) {
		fixture := mustReadFile(t, "testdata/thinking_tool_use_stream.sse")
		wantThinking, wantSignature := expectedThinking(t, fixture)

		server := newS3Server(200, "text/event-stream", fixture)
		defer server.Close()

		var stdout bytes.Buffer
		if err := run(context.Background(), testOpts(server.URL, t.TempDir(), "test-model"), &stdout); err != nil {
			t.Fatalf("run failed: %v", err)
		}

		bodies := server.bodies()
		if len(bodies) != 1 {
			t.Fatalf("S3 requests sent = %d, want exactly 1", len(bodies))
		}
		var req map[string]interface{}
		if err := json.Unmarshal(bodies[0], &req); err != nil {
			t.Fatalf("unmarshal S3 request: %v", err)
		}

		messages, _ := req["messages"].([]interface{})
		if len(messages) != 3 {
			t.Fatalf("S3 has %d messages, want 3", len(messages))
		}

		// messages[1]: the assistant turn, thinking first and unchanged.
		assistant := asMap(t, "messages[1]", messages[1])
		if assistant["role"] != "assistant" {
			t.Errorf("messages[1].role = %v, want assistant", assistant["role"])
		}
		content, _ := assistant["content"].([]interface{})
		if len(content) != 2 {
			t.Fatalf("messages[1].content has %d blocks, want 2 (thinking, tool_use)", len(content))
		}

		thinking := asMap(t, "content[0]", content[0])
		if thinking["type"] != "thinking" {
			t.Errorf("content[0].type = %v, want thinking", thinking["type"])
		}
		if thinking["thinking"] != wantThinking {
			t.Errorf("content[0].thinking:\ngot:  %q\nwant: %q", thinking["thinking"], wantThinking)
		}
		if thinking["signature"] != wantSignature {
			t.Errorf("content[0].signature:\ngot:  %q\nwant: %q", thinking["signature"], wantSignature)
		}

		toolUse := asMap(t, "content[1]", content[1])
		if toolUse["type"] != "tool_use" {
			t.Errorf("content[1].type = %v, want tool_use", toolUse["type"])
		}
		if toolUse["id"] != fixtureToolID {
			t.Errorf("content[1].id = %v, want %s", toolUse["id"], fixtureToolID)
		}
		if toolUse["name"] != fixtureToolName {
			t.Errorf("content[1].name = %v, want %s", toolUse["name"], fixtureToolName)
		}
		wantInput := map[string]interface{}{"location": fixtureToolPlace}
		if !reflect.DeepEqual(toolUse["input"], wantInput) {
			t.Errorf("content[1].input = %v, want %v", toolUse["input"], wantInput)
		}

		// messages[2]: the tool_result answering that tool_use.
		user := asMap(t, "messages[2]", messages[2])
		resultContent, _ := user["content"].([]interface{})
		if len(resultContent) == 0 {
			t.Fatal("messages[2].content is empty")
		}
		result := asMap(t, "messages[2].content[0]", resultContent[0])
		if result["type"] != "tool_result" {
			t.Errorf("tool result type = %v, want tool_result", result["type"])
		}
		if result["tool_use_id"] != fixtureToolID {
			t.Errorf("tool_use_id = %v, want %s", result["tool_use_id"], fixtureToolID)
		}

		if maxTokens, _ := req["max_tokens"].(float64); maxTokens != 512 {
			t.Errorf("max_tokens = %v, want 512", req["max_tokens"])
		}
		if tools, _ := req["tools"].([]interface{}); len(tools) == 0 {
			t.Error("tools is empty or not an array")
		}
	})

	t.Run("S2 with two tool_use blocks", func(t *testing.T) {
		const twoTools = `event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_A","name":"read_file","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"a.md\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_B","name":"read_file","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"path\":\"b.md\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_stop
data: {"type":"message_stop"}
`
		server := newS3Server(200, "text/event-stream", []byte(twoTools))
		defer server.Close()

		if err := run(context.Background(), testOpts(server.URL, t.TempDir(), "test-model"), io.Discard); err != nil {
			t.Fatalf("run failed: %v", err)
		}
		bodies := server.bodies()
		if len(bodies) != 1 {
			t.Fatalf("S3 requests sent = %d, want exactly 1", len(bodies))
		}
		var req struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(bodies[0], &req); err != nil {
			t.Fatalf("unmarshal S3 request: %v", err)
		}
		if len(req.Messages) != 3 {
			t.Fatalf("S3 has %d messages, want 3", len(req.Messages))
		}
		blocksOf := func(i int) []map[string]interface{} {
			var blocks []map[string]interface{}
			if err := json.Unmarshal(req.Messages[i].Content, &blocks); err != nil {
				t.Fatalf("messages[%d].content is not a block array: %v", i, err)
			}
			return blocks
		}
		var useIDs []interface{}
		for _, block := range blocksOf(1) {
			if block["type"] == "tool_use" {
				useIDs = append(useIDs, block["id"])
			}
		}
		var resultIDs []interface{}
		for _, block := range blocksOf(2) {
			if block["type"] != "tool_result" {
				t.Errorf("messages[2] holds a %v block, want only tool_result", block["type"])
				continue
			}
			resultIDs = append(resultIDs, block["tool_use_id"])
			if block["content"] != "# Kirsch\nA terminal coding agent." {
				t.Errorf("tool_result content = %v, want the fixed content", block["content"])
			}
		}
		want := []interface{}{"toolu_A", "toolu_B"}
		if !reflect.DeepEqual(useIDs, want) {
			t.Errorf("tool_use ids = %v, want %v", useIDs, want)
		}
		if !reflect.DeepEqual(resultIDs, want) {
			t.Errorf("tool_result ids = %v, want %v, in order", resultIDs, want)
		}
	})

	t.Run("S2 text-only", func(t *testing.T) {
		const textOnly = `event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Sorry, I cannot read files."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_stop
data: {"type":"message_stop"}
`
		server := newS3Server(200, "", []byte(textOnly))
		defer server.Close()

		tmpDir := t.TempDir()
		var stdout bytes.Buffer
		if err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), &stdout); err != nil {
			t.Fatalf("run failed: %v", err)
		}

		if n := len(server.bodies()); n != 0 {
			t.Errorf("S3 request must not be sent when S2 has no tool_use, but %d were", n)
		}
		const reason = "S2 returned no tool_use"
		if !strings.Contains(stdout.String(), "⊘ test-model S3-tool-result (skipped: "+reason+")") {
			t.Errorf("stdout should report the S3 skip, got: %s", stdout.String())
		}
		s3 := scenarioSummary(t, tmpDir, "test-model", "S3-tool-result")
		if got, _ := s3["error"].(string); got != reason {
			t.Errorf("S3 recorded reason = %q, want %q", got, reason)
		}
		if status, _ := s3["status"].(float64); status != 0 {
			t.Errorf("S3 status = %v, want 0 (never sent)", s3["status"])
		}
		if _, err := os.Stat(filepath.Join(tmpDir, "test-model", "S3-tool-result.request.json")); !os.IsNotExist(err) {
			t.Errorf("a skipped S3 must leave no request file (stat err: %v)", err)
		}
	})
}

// TestS3SkipReasons checks that the S3 skip reason names the actual cause.
func TestS3SkipReasons(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		s2          []byte
		want        string
	}{
		{"non-200", 400, "application/json", []byte(`{"error":"bad request"}`), "S2 returned status 400"},
		{"stream error", 200, "text/event-stream", []byte("event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"message\":\"boom\"}}\n\n"), "S2 recorded an error (see its summary entry)"},
		{"not a stream", 200, "application/json", []byte(`{"content":[]}`), "S2 response was not an event stream"},
		{"no tool_use", 200, "text/event-stream", []byte("event: message_stop\ndata: {}\n\n"), "S2 returned no tool_use"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newS3Server(tt.status, tt.contentType, tt.s2)
			defer server.Close()
			tmpDir := t.TempDir()
			if err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), io.Discard); err != nil {
				t.Fatalf("run failed: %v", err)
			}
			if n := len(server.bodies()); n != 0 {
				t.Errorf("S3 was sent %d times, want 0", n)
			}
			s3 := scenarioSummary(t, tmpDir, "test-model", "S3-tool-result")
			if got, _ := s3["error"].(string); got != tt.want {
				t.Errorf("skip reason = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T10: Summary generation.
// ---------------------------------------------------------------------------

func TestSummaryGeneration(t *testing.T) {
	const textStream = `event: message_start
data: {"type":"message_start","message":{"id":"test","type":"message","usage":{"input_tokens":100,"output_tokens":50}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"response"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":75}}

event: message_stop
data: {"type":"message_stop"}
`
	toolStream := mustReadFile(t, "testdata/thinking_tool_use_stream.sse")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		if isS2Request(body) {
			w.Write(toolStream)
			return
		}
		fmt.Fprint(w, textStream)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	var stdout bytes.Buffer
	if err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), &stdout); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	summary := readJSONMap(t, filepath.Join(tmpDir, "summary.json"))
	if ts, _ := summary["timestamp"].(string); ts == "" {
		t.Error("summary missing timestamp")
	}

	// S1: a text stream. Usage and stop reason come from the stream; no tool.
	s1 := scenarioSummary(t, tmpDir, "test-model", "S1-text")
	usage := asMap(t, "S1 usage", s1["usage"])
	if out, _ := usage["output_tokens"].(float64); out != 75 {
		t.Errorf("S1 usage.output_tokens = %v, want 75", usage["output_tokens"])
	}
	if s1["stop_reason"] != "end_turn" {
		t.Errorf("S1 stop_reason = %v, want end_turn", s1["stop_reason"])
	}
	if _, present := s1["tool_use_ok"]; present {
		t.Error("S1 must not report tool_use_ok: its stream has no tool_use")
	}

	// S2: a thinking block and a tool_use, both counted.
	s2 := scenarioSummary(t, tmpDir, "test-model", "S2-tool-call")
	usage = asMap(t, "S2 usage", s2["usage"])
	if in, _ := usage["input_tokens"].(float64); in != 472 {
		t.Errorf("S2 usage.input_tokens = %v, want 472", usage["input_tokens"])
	}
	if s2["stop_reason"] != "tool_use" {
		t.Errorf("S2 stop_reason = %v, want tool_use", s2["stop_reason"])
	}
	if s2["tool_use_ok"] != true {
		t.Errorf("S2 tool_use_ok = %v, want true", s2["tool_use_ok"])
	}
	if n, _ := s2["thinking_blocks"].(float64); n != 1 {
		t.Errorf("S2 thinking_blocks = %v, want 1", s2["thinking_blocks"])
	}
	for _, field := range []string{"ttfb_ms", "total_ms"} {
		if _, ok := s2[field]; !ok {
			t.Errorf("S2 summary missing %s", field)
		}
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "SUMMARY.md")); err != nil {
		t.Errorf("SUMMARY.md should exist: %v", err)
	}
	if !strings.Contains(stdout.String(), "OpenCode Go Probe Results") {
		t.Errorf("stdout should contain the SUMMARY.md table, got: %s", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// Content type, model names, duplicates.
// ---------------------------------------------------------------------------

func TestIsEventStream(t *testing.T) {
	tests := map[string]bool{
		"text/event-stream":                true,
		"text/event-stream; charset=utf-8": true,
		"TEXT/EVENT-STREAM;charset=UTF-8":  true,
		"application/json":                 false,
		"text/plain; x=text/event-stream":  false,
		"":                                 false,
	}
	for contentType, want := range tests {
		if got := isEventStream(contentType); got != want {
			t.Errorf("isEventStream(%q) = %v, want %v", contentType, got, want)
		}
	}
}

// TestEventStreamWithCharsetIsParsed checks that a Content-Type carrying a
// charset still selects the SSE parser. The body opens with a comment line, so
// the Content-Type is the only thing that can route it.
func TestEventStreamWithCharsetIsParsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		fmt.Fprint(w, ": keep-alive\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n")
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	if err := run(context.Background(), testOpts(server.URL, tmpDir, "test-model"), io.Discard); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if got := scenarioSummary(t, tmpDir, "test-model", "S1-text")["stop_reason"]; got != "end_turn" {
		t.Errorf("S1 stop_reason = %v, want end_turn", got)
	}
}

func TestDuplicateModelsRefused(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer server.Close()

	err := run(context.Background(), testOpts(server.URL, t.TempDir(), "alpha", "beta", "alpha"), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "duplicate model") || !strings.Contains(err.Error(), "alpha") {
		t.Errorf("expected a duplicate model error naming alpha, got: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("a refused run made %d requests, want 0", n)
	}
}

func TestModelNameValidation(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantError bool
	}{
		{"valid name", "claude-3-opus", false},
		{"with dots", "model.v1.2", false},
		{"with underscores", "model_name", false},
		{"with numbers", "model123", false},
		{"with slash", "model/v1", true},
		{"with space", "model name", true},
		{"with special char", "model@x", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModelName(tt.model)
			if (err != nil) != tt.wantError {
				t.Errorf("validateModelName(%q) error = %v, wantError = %v", tt.model, err, tt.wantError)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Key redaction: stdout and returned errors, not only files.
// ---------------------------------------------------------------------------

func TestRedactKey(t *testing.T) {
	const key = "SENTINEL-KEY-7f3a"
	tests := []struct{ name, in, key, want string }{
		{"empty key leaves the input alone", "abc", "", "abc"},
		{"no occurrence", "abc", key, "abc"},
		{"every occurrence", key + " and " + key, key, "<redacted> and <redacted>"},
		{"inside a host", key + ".example", key, "<redacted>.example"},
	}
	for _, tt := range tests {
		if got := redactKey(tt.in, tt.key); got != tt.want {
			t.Errorf("%s: redactKey(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}

	// Redaction runs before truncation: a key straddling the cut must not
	// leave a prefix behind.
	p := NewProber(options{key: key})
	straddling := strings.Repeat("x", 294) + key
	if got := p.safe(straddling, 300); strings.Contains(got, "SENTIN") {
		t.Errorf("a truncated key prefix survived: %q", got[250:])
	}
}

// TestKeyReflectedNotPrinted drives run() against servers that reflect the key
// into text the probe prints, returns as an error or records in a summary. The
// key, a prefix of it and its length must reach none of stdout, the returned
// error, or any file.
func TestKeyReflectedNotPrinted(t *testing.T) {
	const key = "SENTINEL-KEY-7f3a"
	lengthToken := regexp.MustCompile(`\b` + fmt.Sprint(len(key)) + `\b`)

	cases := map[string]http.HandlerFunc{
		"302 Location host": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://"+key+".example/")
			w.WriteHeader(http.StatusFound)
		},
		"transport failure": func(w http.ResponseWriter, r *http.Request) {
			conn, buf, err := w.(http.Hijacker).Hijack()
			if err != nil {
				return
			}
			defer conn.Close()
			// A status line whose code is not a number makes net/http quote
			// the code in the transport error.
			buf.WriteString("HTTP/1.1 " + key + "\r\n\r\n")
			buf.Flush()
		},
		"stream error straddling the truncation": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			// The first character is JSON-escaped so the raw body does not hold
			// the key: the write guard would otherwise refuse the .sse file and
			// summary.json would never be written. The decoded message holds the
			// whole key, starting 10 runes before the 500-rune truncation point.
			fmt.Fprintf(w, "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"message\":\"%s\\u0053%s\"}}\n\n",
				strings.Repeat("x", 490), key[1:])
		},
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()

			dir := t.TempDir()
			var stdout bytes.Buffer
			opts := testOpts(server.URL, dir, "test-model")
			opts.key = key
			err := run(context.Background(), opts, &stdout)

			var errText string
			if err != nil {
				errText = err.Error()
			}
			for what, text := range map[string]string{"stdout": stdout.String(), "returned error": errText} {
				if strings.Contains(text, key) {
					t.Errorf("%s contains the key: %s", what, text)
				}
				if strings.Contains(text, key[:8]) {
					t.Errorf("%s contains a prefix of the key: %s", what, text)
				}
				if lengthToken.MatchString(text) {
					t.Errorf("%s contains the key's length (%d): %s", what, len(key), text)
				}
			}
			assertNoFileContains(t, dir, key)
			assertNoFileContains(t, dir, key[:8])

			// Each case must actually reach the path it is named for.
			switch name {
			case "302 Location host":
				if err == nil || !strings.Contains(errText, "redirect to <redacted>.example") {
					t.Errorf("expected the redirect stop with a redacted host, got: %v", err)
				}
			case "transport failure":
				if err == nil || !strings.Contains(errText, "malformed HTTP status code") || !strings.Contains(stdout.String(), "(error: ") {
					t.Errorf("expected a transport error on stdout and as the error, got: %v / %s", err, stdout.String())
				}
				if !strings.Contains(errText, "<redacted>") {
					t.Errorf("the reflected key should appear redacted, got: %s", errText)
				}
			case "stream error straddling the truncation":
				if err != nil {
					t.Errorf("a stream error is recorded, not fatal; got: %v", err)
				}
				if !strings.Contains(stdout.String(), "(stream error)") {
					t.Errorf("stdout should mark the stream error, got: %s", stdout.String())
				}
				if got, _ := scenarioSummary(t, dir, "test-model", "S1-text")["error"].(string); !strings.Contains(got, "<redacted>") {
					t.Errorf("summary.json should hold the redacted message, got: %q", got)
				}
			}
		})
	}
}

// TestStreamErrorMarked checks that a 200 whose stream carried an error is not
// shown as a clean pass, on stdout or in SUMMARY.md.
func TestStreamErrorMarked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"message\":\"boom\"}}\n\n")
	}))
	defer server.Close()

	dir := t.TempDir()
	var stdout bytes.Buffer
	if err := run(context.Background(), testOpts(server.URL, dir, "test-model"), &stdout); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "S1-text (") || !strings.Contains(stdout.String(), ") (stream error)\n") {
		t.Errorf("stdout should mark the stream error, got: %s", stdout.String())
	}
	md := string(mustReadFile(t, filepath.Join(dir, "SUMMARY.md")))
	if !strings.Contains(md, " | 200*") {
		t.Errorf("SUMMARY.md should mark 200 with an asterisk, got: %s", md)
	}
	if !strings.Contains(md, "\\* 200 response whose stream carried an error") {
		t.Errorf("SUMMARY.md should carry the footnote, got: %s", md)
	}
}

func TestReservedAndCaseInsensitiveModels(t *testing.T) {
	for _, name := range []string{"_global", "_GLOBAL", "_Global"} {
		if err := validateModelName(name); err == nil {
			t.Errorf("validateModelName(%q) should refuse the reserved name", name)
		}
	}
	err := run(context.Background(), testOpts("http://127.0.0.1:1", t.TempDir(), "Alpha", "alpha"), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "duplicate model") {
		t.Errorf("a case-only duplicate should be refused, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Identity headers and gateway policy errors.
// ---------------------------------------------------------------------------

// recordedRequest is what a test server saw of one request.
type recordedRequest struct {
	path   string
	header http.Header
	body   []byte
}

// recordingServer records every request, serves s2 to the S2 request and a
// bare message_stop to the rest, and lets respond override any response.
type recordingServer struct {
	*httptest.Server
	mu   sync.Mutex
	reqs []recordedRequest
}

// newRecordingServer starts a server. respond may be nil; when it returns true
// it has written the response itself.
func newRecordingServer(s2 []byte, respond func(n int, r recordedRequest, w http.ResponseWriter) bool) *recordingServer {
	s := &recordingServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec := recordedRequest{path: r.URL.Path, header: r.Header.Clone(), body: body}
		s.mu.Lock()
		s.reqs = append(s.reqs, rec)
		n := len(s.reqs)
		s.mu.Unlock()
		if respond != nil && respond(n, rec, w) {
			return
		}
		if isS2Request(body) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write(s2)
			return
		}
		fmt.Fprint(w, messageStopOnly)
	}))
	return s
}

func (s *recordingServer) requests() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedRequest(nil), s.reqs...)
}

var sessionIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// runIdentityProbe runs one full single-model probe against a recording server
// whose S2 yields a tool call, so S3 runs. Request order is then fixed:
// S0, S1, S2, S3, S4-1, S4-2, S5a-d, S6.
func runIdentityProbe(t *testing.T) []recordedRequest {
	t.Helper()
	server := newRecordingServer(mustReadFile(t, "testdata/thinking_tool_use_stream.sse"), nil)
	defer server.Close()
	if err := run(context.Background(), testOpts(server.URL, t.TempDir(), "test-model"), io.Discard); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	reqs := server.requests()
	if len(reqs) != 11 {
		t.Fatalf("requests = %d, want 11 (S0, S1, S2, S3, S4-1, S4-2, S5a-d, S6)", len(reqs))
	}
	if n := len(requestMessages(reqs[3].body)); n != 3 {
		t.Fatalf("request 4 has %d messages, want the 3 of S3", n)
	}
	return reqs
}

// TestIdentityHeaders checks the User-Agent on every request and the session
// ID rules: one per conversation, none shared across conversations or runs.
func TestIdentityHeaders(t *testing.T) {
	reqs := runIdentityProbe(t)
	const (
		s0, s1, s2, s3, s41, s42, s5a, s5d, s6 = 0, 1, 2, 3, 4, 5, 6, 9, 10
	)

	for i, r := range reqs {
		if got := r.header.Get("User-Agent"); got != "kirsch-probe/0.1" {
			t.Errorf("request %d (%s): User-Agent = %q, want kirsch-probe/0.1", i+1, r.path, got)
		}
		id := r.header.Get("x-opencode-session")
		if !sessionIDPattern.MatchString(id) {
			t.Errorf("request %d (%s): x-opencode-session = %q, want 32 hex characters", i+1, r.path, id)
		}
	}
	id := func(i int) string { return reqs[i].header.Get("x-opencode-session") }

	if id(s2) != id(s3) {
		t.Errorf("S2 and S3 are one conversation: sessions %q and %q differ", id(s2), id(s3))
	}
	if id(s41) != id(s42) {
		t.Errorf("S4-1 and S4-2 are a cache pair: sessions %q and %q differ", id(s41), id(s42))
	}

	// One representative per conversation, including S0 and every S5 request.
	conversations := []int{s0, s1, s2, s41, s5a, s5a + 1, s5a + 2, s5d, s6}
	seen := make(map[string]int)
	for _, i := range conversations {
		if prev, dup := seen[id(i)]; dup {
			t.Errorf("requests %d and %d are different conversations but share session %q", prev+1, i+1, id(i))
		}
		seen[id(i)] = i
	}

	// A second run shares no ID with the first.
	second := runIdentityProbe(t)
	firstIDs := make(map[string]bool)
	for _, r := range reqs {
		firstIDs[r.header.Get("x-opencode-session")] = true
	}
	for i, r := range second {
		if got := r.header.Get("x-opencode-session"); firstIDs[got] {
			t.Errorf("run 2 request %d reuses session %q from run 1", i+1, got)
		}
	}
}

// TestSessionRecorded checks each scenario's session ID is saved, matches what
// was sent, and that the key is nowhere in it.
func TestSessionRecorded(t *testing.T) {
	server := newRecordingServer(mustReadFile(t, "testdata/thinking_tool_use_stream.sse"), nil)
	defer server.Close()
	dir := t.TempDir()
	if err := run(context.Background(), testOpts(server.URL, dir, "test-model"), io.Discard); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	reqs := server.requests()

	global := strings.TrimSpace(string(mustReadFile(t, filepath.Join(dir, "_global", "S0-models.session"))))
	if global != reqs[0].header.Get("x-opencode-session") {
		t.Errorf("_global session file = %q, sent %q", global, reqs[0].header.Get("x-opencode-session"))
	}
	for i, name := range map[int]string{1: "S1-text", 2: "S2-tool-call", 3: "S3-tool-result", 5: "S4-cache-2"} {
		want := reqs[i].header.Get("x-opencode-session")
		file := strings.TrimSpace(string(mustReadFile(t, filepath.Join(dir, "test-model", name+".session"))))
		if file != want {
			t.Errorf("%s session file = %q, sent %q", name, file, want)
		}
		if got := scenarioSummary(t, dir, "test-model", name)["session"]; got != want {
			t.Errorf("%s summary session = %v, sent %q", name, got, want)
		}
	}
	assertNoFileContains(t, dir, "test-key")
}

// missingSessionBody is the gateway's exact reply to a request with no session.
const missingSessionBody = `{"type":"error","error":{"type":"MissingSessionID","message":"Request is missing x-opencode-session and cannot be routed efficiently. Please see https://opencode.ai/docs/go/#where-can-i-use-it"}}`

// TestMissingSessionStops checks a MissingSessionID 400 stops the run after the
// first Messages request, with results still written.
func TestMissingSessionStops(t *testing.T) {
	server := newRecordingServer(nil, func(n int, r recordedRequest, w http.ResponseWriter) bool {
		if !strings.HasSuffix(r.path, "/messages") {
			return false
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, missingSessionBody)
		return true
	})
	defer server.Close()

	dir := t.TempDir()
	var stdout bytes.Buffer
	err := run(context.Background(), testOpts(server.URL, dir, "test-model"), &stdout)
	if err == nil {
		t.Fatal("expected the run to stop on MissingSessionID")
	}
	if !strings.Contains(err.Error(), "MissingSessionID") {
		t.Errorf("error should name the gateway error type, got: %v", err)
	}
	if n := len(server.requests()); n != 2 {
		t.Errorf("requests = %d, want exactly 2 (S0 and S1)", n)
	}
	for _, path := range []string{
		filepath.Join(dir, "_global", "S0-models.sse"),
		filepath.Join(dir, "test-model", "S1-text.sse"),
		filepath.Join(dir, "test-model", "S1-text.session"),
		filepath.Join(dir, "SUMMARY.md"),
		filepath.Join(dir, "summary.json"),
	} {
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("%s should exist after a stop: %v", path, statErr)
		}
	}
	if got := string(mustReadFile(t, filepath.Join(dir, "test-model", "S1-text.sse"))); got != missingSessionBody {
		t.Errorf("stopping body not saved, got: %q", got)
	}
	if got := scenarioSummary(t, dir, "test-model", "S1-text")["status"]; got != float64(400) {
		t.Errorf("S1 summary status = %v, want 400", got)
	}
}

// TestSessionErrorTypeStops checks any error type containing "Session" stops
// the run, and that the stopping body is cut to 4KB.
func TestSessionErrorTypeStops(t *testing.T) {
	big := `{"error":{"type":"InvalidSessionID","message":"` + strings.Repeat("x", 10000) + `"}}`
	server := newRecordingServer(nil, func(n int, r recordedRequest, w http.ResponseWriter) bool {
		if !strings.HasSuffix(r.path, "/messages") {
			return false
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, big)
		return true
	})
	defer server.Close()

	dir := t.TempDir()
	if err := run(context.Background(), testOpts(server.URL, dir, "test-model"), io.Discard); err == nil {
		t.Fatal("expected the run to stop on a Session error type")
	}
	if n := len(server.requests()); n != 2 {
		t.Errorf("requests = %d, want 2", n)
	}
	if got := len(mustReadFile(t, filepath.Join(dir, "test-model", "S1-text.sse"))); got != 4096 {
		t.Errorf("stopping body length = %d, want 4096", got)
	}
}

// TestPlain400DoesNotStop checks a 400 for a bad request body, as S5's thinking
// forms may draw, does not stop the run.
func TestPlain400DoesNotStop(t *testing.T) {
	server := newRecordingServer(nil, func(n int, r recordedRequest, w http.ResponseWriter) bool {
		if !bytes.Contains(r.body, []byte("between_tools")) {
			return false
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"type":"error","error":{"type":"invalid_request_error","message":"thinking.type: bad value"}}`)
		return true
	})
	defer server.Close()

	dir := t.TempDir()
	if err := run(context.Background(), testOpts(server.URL, dir, "test-model"), io.Discard); err != nil {
		t.Fatalf("a plain 400 must not stop the run: %v", err)
	}
	reqs := server.requests()
	// S3 is skipped (S2 had no tool call), so the plan is 10 requests.
	if len(reqs) != 10 {
		t.Fatalf("requests = %d, want all 10", len(reqs))
	}
	for _, name := range []string{"S5b-between-tools", "S5c-disabled", "S5d-enabled", "S6-no-version-header"} {
		if _, err := os.Stat(filepath.Join(dir, "test-model", name+".status")); err != nil {
			t.Errorf("%s should have run: %v", name, err)
		}
	}
	if got := scenarioSummary(t, dir, "test-model", "S5b-between-tools")["status"]; got != float64(400) {
		t.Errorf("S5b status = %v, want 400", got)
	}
}
