package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// testAPIKey is the key the test environment supplies for TEST_API_KEY. It must
// never appear in a tool_result the model receives.
const testAPIKey = "secret-key"

// failingProbe is a test-only tool that fails and also returns output. It lets
// the test see that a failed result keeps its Content beside the error line.
type failingProbe struct{}

func (failingProbe) Name() string        { return "probe_fail" }
func (failingProbe) Description() string { return "test-only tool that always fails" }
func (failingProbe) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (failingProbe) Invoke(ctx context.Context, raw json.RawMessage) tool.Result {
	return tool.Result{
		OK:             false,
		Content:        "partial output",
		DisplaySummary: "failed",
		Error:          &tool.Error{Kind: tool.KindCommandFailed, Message: "exit status 3"},
	}
}

// bareFailProbe is a test-only tool that returns OK false with Content and no
// Error. No production tool does this: tool.Fail always sets Error, and the
// run_command exit-status path sets it too. The branch of modelToolContent that
// handles a failure with no Error is therefore reached only through this probe.
type bareFailProbe struct{}

func (bareFailProbe) Name() string { return "probe_bare_fail" }
func (bareFailProbe) Description() string {
	return "test-only tool that fails with Content and no Error"
}

func (bareFailProbe) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (bareFailProbe) Invoke(ctx context.Context, raw json.RawMessage) tool.Result {
	return tool.Result{OK: false, Content: "x"}
}

// turnResult is what one runToolTurn call observed. reqs are the two requests
// the provider received. approvals counts the approval requests the turn raised.
type turnResult struct {
	reqs      []provider.Request
	approvals int
}

// runToolTurn drives one turn through App.Submit, the entry point the TUI uses,
// against the git repo at repo. The fake provider asks for one call to toolName
// with input on the first round and answers "done" on the second. The turn runs
// agent.RunTurn, which calls toolExecutor.Invoke, which calls the real Registry.
// No network is used.
//
// Every approval request is refused at once through the sendFn hook, so a turn
// that reaches approval ends at once and is counted, rather than waiting out the
// timeout. label names the calling case in the timeout message, so a turn that
// hangs is reported against the path that hung.
func runToolTurn(t *testing.T, label, repo, toolName string, input json.RawMessage) turnResult {
	t.Helper()
	ws, err := workspace.Detect(repo)
	if err != nil {
		t.Fatalf("%s: workspace.Detect: %v", label, err)
	}
	cfg := config.Config{
		Provider: config.ProviderConfig{
			Default: "test",
			Endpoints: map[string]config.EndpointConfig{
				"test": {
					BaseURL:   "https://example.com",
					Auth:      "x-api-key",
					APIKeyEnv: "TEST_API_KEY",
					Model:     "claude-sonnet-5-5",
				},
			},
		},
	}
	a := New(ws, cfg, telemetry.Disabled())
	defer a.Close()

	// Signal the test when the turn ends, whichever message ends it. Count and
	// refuse approval requests on the way through.
	finished := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	approvals := 0
	a.sendFn = func(msg any) {
		switch m := msg.(type) {
		case tui.ApprovalRequestedMsg:
			mu.Lock()
			approvals++
			mu.Unlock()
			a.Resolve(m.ID, ApprovalOutcomeDeny)
		case tui.TurnCompleteMsg, tui.TurnErrorMsg:
			once.Do(func() { close(finished) })
		}
	}
	a.reg.Register(failingProbe{})
	a.reg.Register(bareFailProbe{})

	fake := provider.NewFake(
		provider.Turn{ToolCalls: []provider.ToolCall{{
			ID:    "call-1",
			Name:  toolName,
			Input: input,
		}}},
		provider.Turn{Text: "done"},
	)
	a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
		return fake, nil
	}

	getenv := func(k string) string {
		if k == "TEST_API_KEY" {
			return testAPIKey
		}
		return ""
	}

	a.StartSession(getenv)
	a.Submit("apply the change", getenv)
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatalf("%s: turn did not finish within 10s", label)
	}

	reqs := fake.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%s: provider received %d requests, want 2", label, len(reqs))
	}
	mu.Lock()
	n := approvals
	mu.Unlock()
	return turnResult{reqs: reqs, approvals: n}
}

// findToolResult returns the tool_result for callID in req, and fails the test
// if there is none. Without this control a content check could pass on absence.
func findToolResult(t *testing.T, req provider.Request, callID string) *provider.ToolResult {
	t.Helper()
	var got *provider.ToolResult
	for _, m := range req.Messages {
		for _, b := range m.Content {
			if b.Kind == provider.BlockToolResult && b.ToolResult != nil && b.ToolResult.CallID == callID {
				got = b.ToolResult
			}
		}
	}
	if got == nil {
		t.Fatalf("second request has no tool_result for %s", callID)
	}
	return got
}

// assertNoAPIKey fails if testAPIKey appears in any tool_result the provider
// received. label names the calling case so a failure identifies its path. It
// also fails if no tool_result was scanned, so the absence check cannot pass on
// an empty set.
func assertNoAPIKey(t *testing.T, reqs []provider.Request, label string) {
	t.Helper()
	scanned := 0
	for _, r := range reqs {
		for _, m := range r.Messages {
			for _, b := range m.Content {
				if b.Kind != provider.BlockToolResult || b.ToolResult == nil {
					continue
				}
				scanned++
				if strings.Contains(b.ToolResult.Content, testAPIKey) {
					t.Errorf("%s: API key reached the model in tool_result %q: %q",
						label, b.ToolResult.CallID, b.ToolResult.Content)
				}
			}
		}
	}
	if scanned == 0 {
		t.Fatalf("%s: API key check scanned no tool_result, so the absence check is vacuous", label)
	}
}

// TestSubmitFailedToolResultReachesModel drives a full turn through App.Submit
// and checks the exact model-facing content and IsError flag of a tool_result.
// It covers each branch of modelToolContent, with a subtest per branch:
//
//	B1 OK result: Content unchanged.
//	B2 failure with no Error: Content unchanged.
//	B3 failure with an Error and empty Content: the error line alone.
//	B4 failure with an Error and Content: the error line, a blank line, Content.
func TestSubmitFailedToolResultReachesModel(t *testing.T) {
	// The hunk has a header and no body. Header counts that disagree with a
	// body are now recounted, so a bad-count diff would apply. A zero-body hunk
	// is still rejected at parse time, before any approval is requested. This is
	// the same shape as TestApplyPatchParseFailureNoApproval.
	badPatch := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1,3 +1,3 @@\n"
	patchInput, err := json.Marshal(map[string]string{"diff": badPatch, "description": "test"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		toolName  string
		input     json.RawMessage
		wantError bool
		want      string // the exact model-facing content
		failText  string // distinct per case, so a failure names its path
	}{
		{
			// read_file numbers each line, so the model receives the numbered form.
			name:      "B1 read_file success",
			toolName:  "read_file",
			input:     json.RawMessage(`{"path":"file.txt"}`),
			wantError: false,
			want:      "1\tline1\n2\tline2\n3\tline3\n",
			failText:  "a successful result must reach the model as its Content, unchanged",
		},
		{
			name:      "B2 failure with content and no error",
			toolName:  "probe_bare_fail",
			input:     json.RawMessage(`{}`),
			wantError: true,
			want:      "x",
			failText:  "a failed result with no Error must reach the model as its Content, unchanged",
		},
		{
			name:      "B3 apply_patch with empty tool content",
			toolName:  "apply_patch",
			input:     patchInput,
			wantError: true,
			want:      "tool_input_invalid: hunk arithmetic error in diff: in a/file.txt @@ -1,3 +1,3 @@: hunk has no body lines",
			failText:  "empty tool content must be the error line alone, with no separator",
		},
		{
			name:      "B4 failing tool keeps its content",
			toolName:  "probe_fail",
			input:     json.RawMessage(`{}`),
			wantError: true,
			want:      "command_failed: exit status 3\n\npartial output",
			failText:  "failed tool content must be the error line, a blank line, then Content",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newGitRepo(t, map[string]string{"file.txt": "line1\nline2\nline3\n"})
			res := runToolTurn(t, tc.name, repo, tc.toolName, tc.input)

			got := findToolResult(t, res.reqs[1], "call-1")
			if got.IsError != tc.wantError {
				t.Errorf("%s: tool_result IsError = %v, want %v", tc.failText, got.IsError, tc.wantError)
			}
			if got.Content != tc.want {
				t.Errorf("%s: tool_result content = %q, want exactly %q", tc.failText, got.Content, tc.want)
			}
			assertNoAPIKey(t, res.reqs, tc.failText)
		})
	}
}

// TestSubmitDenylistedPatchToolResultHidesContent drives an apply_patch at .env,
// which internal/workspace denies. The denied path is refused when apply_patch
// resolves it, before the patch is staged and before any approval is requested.
// The tool_result must name the workspace_violation kind and must not carry the
// file's contents. The file holds a sentinel so that any leak of its bytes would
// be visible.
func TestSubmitDenylistedPatchToolResultHidesContent(t *testing.T) {
	const sentinel = "DENYLIST-SENTINEL-4e7b1c"
	repo := newGitRepo(t, map[string]string{
		"file.txt": "line1\n",
		".env":     sentinel + "\n",
	})

	// Setup control: the sentinel is on disk before the turn runs.
	onDisk, err := os.ReadFile(filepath.Join(repo, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), sentinel) {
		t.Fatalf("setup: .env does not contain the sentinel: %q", onDisk)
	}

	// A valid single-line replacement of the sentinel line. The diff is well
	// formed, so the refusal can only come from the path check.
	diff := "diff --git a/.env b/.env\n" +
		"--- a/.env\n" +
		"+++ b/.env\n" +
		"@@ -1 +1 @@\n" +
		"-" + sentinel + "\n" +
		"+replaced\n"
	input, err := json.Marshal(map[string]string{"diff": diff, "description": "test"})
	if err != nil {
		t.Fatal(err)
	}

	res := runToolTurn(t, "denylist refusal", repo, "apply_patch", input)
	got := findToolResult(t, res.reqs[1], "call-1")

	// The refusal precedes approval, so a denied path must raise no approval
	// request. A refusal that arrives later would show up here, not as a timeout.
	if res.approvals != 0 {
		t.Errorf("denylist refusal: %d approval request(s) raised for a denied path, want 0", res.approvals)
	}
	if !got.IsError {
		t.Errorf("denylist refusal: tool_result IsError = false, want true")
	}
	// Positive control: the refusal kind is present, so the absence check below
	// is shown to be able to see a tool_result at all.
	if !strings.HasPrefix(got.Content, "workspace_violation: ") {
		t.Fatalf("denylist refusal: tool_result = %q, want it to start with \"workspace_violation: \"", got.Content)
	}
	if strings.Contains(got.Content, sentinel) {
		t.Errorf("denylist refusal: tool_result leaked the denied file's contents: %q", got.Content)
	}
	assertNoAPIKey(t, res.reqs, "denylist refusal")

	// Positive control for the approval count. The same path, with an allowed
	// file, must raise exactly one approval request. Without this, zero approvals
	// above could mean the hook never fired.
	ctrlDiff := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1 +1 @@\n" +
		"-line1\n" +
		"+changed\n"
	ctrlInput, err := json.Marshal(map[string]string{"diff": ctrlDiff, "description": "test"})
	if err != nil {
		t.Fatal(err)
	}
	ctrl := runToolTurn(t, "denylist control", repo, "apply_patch", ctrlInput)
	if ctrl.approvals != 1 {
		t.Errorf("denylist control: %d approval request(s) raised for an allowed path, want 1", ctrl.approvals)
	}
}
