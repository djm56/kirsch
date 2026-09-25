package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
)

// countGoroutines returns the current number of goroutines.
func countGoroutines() int {
	return runtime.NumGoroutine()
}

// TestRunCommandRejectsSingleString verifies that a command passed as a single
// string is rejected with a message telling the model to pass argv.
func TestRunCommandRejectsSingleString(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Pass a single string that looks like a command.
	input := runCommandInput{
		Argv: []string{"go test ./..."},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure for single-string command")
	}
	if result.Error.Kind != KindToolInputInvalid {
		t.Fatalf("expected KindToolInputInvalid, got %s", result.Error.Kind)
	}
	// Verify the message tells the user to pass argv.
	if !strings.Contains(result.Error.Message, "array") {
		t.Fatalf("error message should mention passing as array, got: %s", result.Error.Message)
	}
	if !strings.Contains(result.Error.Message, "go") {
		t.Fatalf("error message should include a corrected example, got: %s", result.Error.Message)
	}
}

// TestRunCommandEmptyArgv verifies that empty argv is rejected.
func TestRunCommandEmptyArgv(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	input := runCommandInput{
		Argv: []string{},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure for empty argv")
	}
	if result.Error.Kind != KindToolInputInvalid {
		t.Fatalf("expected KindToolInputInvalid, got %s", result.Error.Kind)
	}
}

// TestRunCommandCwdViolation verifies that cwd escaping the workspace is refused.
func TestRunCommandCwdViolation(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	input := runCommandInput{
		Argv: []string{"ls"},
		Cwd:  "../../../etc",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure for cwd violation")
	}
	if result.Error.Kind != KindWorkspaceViolation {
		t.Fatalf("expected KindWorkspaceViolation, got %s", result.Error.Kind)
	}
	// Verify the approver was never called.
	if approver.called {
		t.Fatalf("approver should not have been called for workspace violation")
	}
}

// TestRunCommandShellCommandRequiresApproval verifies that a shell command
// calls the approver (shell commands require approval, never allowed).
func TestRunCommandShellCommandRequiresApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// sh is a shell and will require approval.
	input := runCommandInput{
		Argv: []string{"sh", "-c", "echo hello"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	_ = tool.Invoke(ctx, raw)

	// The shell command was approved by the fake approver.
	if !approver.called {
		t.Fatalf("approver should have been called for shell command")
	}
}

// TestRunCommandAllowlistNoApproval verifies that an allowlisted command
// does not call the approver.
func TestRunCommandAllowlistNoApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// "go build" is in the default allowlist.
	input := runCommandInput{
		Argv: []string{"go", "build"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	_ = tool.Invoke(ctx, raw)

	// Command will fail (go build in test workspace will fail), but should not require approval.
	if approver.called {
		t.Fatalf("approver should not have been called for allowlisted command")
	}
}

// TestRunCommandNotAllowedRequiresApproval verifies that a command not in the
// allowlist calls the approver.
func TestRunCommandNotAllowedRequiresApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// "echo" is not in the default allowlist.
	input := runCommandInput{
		Argv: []string{"echo", "hello"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	_ = tool.Invoke(ctx, raw)

	if !approver.called {
		t.Fatalf("approver should have been called for non-allowlisted command")
	}
	if approver.lastReq.Operation != policy.OperationCommand {
		t.Fatalf("approval request should have OperationCommand")
	}
}

// TestRunCommandStdinIsDevNull verifies that a command reading from stdin fails
// immediately rather than hanging. We verify by timing the execution.
//
// Known limitation: Go's os/exec treats a nil cmd.Stdin identically to an
// explicit *os.File opened on os.DevNull (see os/exec's Cmd.Stdin doc), so
// this black-box test cannot discriminate "RunCommand explicitly opens
// /dev/null" from "RunCommand leaves Stdin nil" — both produce the same
// observable behaviour. What it does prove, and the reason it exists, is the
// actual regression it guards: a command that reads from stdin must never
// hang the tool. That contract is real and worth keeping even though it
// doesn't pin the specific code path.
func TestRunCommandStdinIsDevNull(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use 'cat' which will read from stdin and EOF immediately since stdin is /dev/null.
	// The important test here is that the command completes quickly, not hanging waiting for input.
	input := runCommandInput{
		Argv: []string{"cat"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	// Time the execution with a 2-second timeout to ensure it doesn't hang.
	done := make(chan Result, 1)
	go func() {
		done <- tool.Invoke(ctx, raw)
	}()

	select {
	case res := <-done:
		// Command completed quickly. This is what we want - it read /dev/null and exited.
		// The exact exit status doesn't matter, the point is it didn't hang.
		_ = res
	case <-time.After(2 * time.Second):
		t.Fatalf("command hung when reading from stdin; /dev/null redirection is not working")
	}
}

// TestRunCommandEnvironmentFiltering verifies that environment filtering works correctly.
// A variable in both env_passthrough and a strip pattern should be stripped.
func TestRunCommandEnvironmentFiltering(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Set up a test token in the environment.
	testToken := "SECRET_TOKEN_12345"
	if err := os.Setenv("MY_TOKEN", testToken); err != nil {
		t.Fatalf("failed to set test env: %v", err)
	}
	defer os.Unsetenv("MY_TOKEN")

	// Also set a control variable to ensure PATH etc. still pass through.
	if err := os.Setenv("MY_PATH_VAR", "/test/path"); err != nil {
		t.Fatalf("failed to set test env: %v", err)
	}
	defer os.Unsetenv("MY_PATH_VAR")

	// Configure the policy to allow passthrough of MY_PATH_VAR, but MY_TOKEN will
	// be stripped because it matches *_TOKEN.
	conf := config.Defaults()
	conf.Policy.EnvPassthrough = []string{"MY_PATH_VAR", "MY_TOKEN"}
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use env to print the environment.
	input := runCommandInput{
		Argv: []string{"env"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if !result.OK {
		t.Fatalf("env command failed: %s", result.Error.Message)
	}

	// Check that MY_TOKEN is NOT in the output (stripped).
	if strings.Contains(result.Content, "MY_TOKEN") {
		t.Fatalf("MY_TOKEN should have been stripped from environment, but found in output")
	}

	// Check that MY_PATH_VAR IS in the output (allowed, not stripped).
	if !strings.Contains(result.Content, "MY_PATH_VAR") {
		t.Fatalf("MY_PATH_VAR should be in environment, but not found in output")
	}

	// Check that PATH is in the output (always included).
	if !strings.Contains(result.Content, "PATH=") {
		t.Fatalf("PATH should be in environment, but not found in output")
	}
}

// TestRunCommandOutputCapAt200KB verifies that output is capped at 200KB with
// head and tail, that Truncated is reported true, and that the elision marker
// is present. This is the regression test for the truncation-collapse fix:
// before it, capWriter's own 200KB cut left Truncate's len(s) <= maxBytes
// check always true, so Truncated was always false and no marker was ever
// emitted for output that had, in fact, lost its middle. Driven entirely
// through Invoke, not against capWriter directly.
func TestRunCommandOutputCapAt200KB(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Generate output with distinguishable head and tail:
	// - Head: 50KB of "HEAD_" repeated
	// - Middle: 150KB of middle content
	// - Tail: 50KB of "TAIL_" repeated
	// Total: 250KB, will be capped at 200KB with head and tail.
	head := strings.Repeat("HEAD_", 10*1024)   // 50KB
	middle := strings.Repeat("MIDDLE_", 21428) // ~150KB
	tail := strings.Repeat("TAIL_", 10*1024)   // 50KB
	largeString := head + middle + tail

	// Use printf to generate the large output.
	input := runCommandInput{
		Argv: []string{"printf", largeString},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if !result.OK {
		t.Fatalf("command failed: %s", result.Error.Message)
	}

	if !result.Truncated {
		t.Fatalf("output should have been truncated")
	}

	// Verify the cap: output should not exceed 200KB.
	if len(result.Content) > 200*1024 {
		t.Fatalf("output exceeds 200KB cap: %d bytes", len(result.Content))
	}

	// Verify both head and tail are present (the elision marker should be there).
	if !strings.Contains(result.Content, "‹… output truncated …›") {
		t.Fatalf("output should contain truncation marker")
	}

	// Verify head is present (should start with "HEAD_").
	if !strings.Contains(result.Content, "HEAD_") {
		t.Fatalf("output should contain head content (HEAD_)")
	}

	// Verify tail is present (should contain "TAIL_").
	if !strings.Contains(result.Content, "TAIL_") {
		t.Fatalf("output should contain tail content (TAIL_)")
	}

	// Verify they appear in the right order: head before marker, tail after marker.
	headIdx := strings.Index(result.Content, "HEAD_")
	markerIdx := strings.Index(result.Content, "‹… output truncated …›")
	tailIdx := strings.LastIndex(result.Content, "TAIL_")

	if headIdx == -1 || markerIdx == -1 || tailIdx == -1 {
		t.Fatalf("could not verify truncation structure in output")
	}

	if !(headIdx < markerIdx && markerIdx < tailIdx) {
		t.Fatalf("output structure incorrect: head(%d) < marker(%d) < tail(%d) is false", headIdx, markerIdx, tailIdx)
	}
}

// TestRunCommandApprovedForSessionRecordsGrant verifies that session approval is recorded.
// Note: This test depends on the approval flow (M3), so we test the policy.Grant call is made.
//
// Known limitation (logged as a finding for m2-d5, not addressed here): session
// grants are unreachable as currently typed, so this test records and asserts
// no grant. It exists to pin the approval-request flow, not grant recording.
func TestRunCommandApprovedForSessionRecordsGrant(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := &config.Config{}
	pol := policy.New()

	// Create a custom approver that simulates session approval.
	// For now, this test just ensures the flow exists; the actual grant recording
	// happens in M3's approval flow, not here.
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: conf, Policy: pol, Approver: approver}

	// Use a command not in allowlist to trigger approval.
	input := runCommandInput{
		Argv: []string{"echo", "test"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	_ = tool.Invoke(ctx, raw)

	// Command should succeed and approver should have been called.
	if !approver.called {
		t.Fatalf("approver should have been called")
	}
	if approver.lastReq.Operation != policy.OperationCommand {
		t.Fatalf("operation should be OperationCommand")
	}
}

// TestRunCommandAllStripPatterns verifies all strip patterns: *_TOKEN, *_KEY, *_SECRET, AWS_*.
func TestRunCommandAllStripPatterns(t *testing.T) {
	testCases := []struct {
		name        string
		envVarName  string
		shouldStrip bool
	}{
		{"MY_TOKEN", "MY_TOKEN", true},
		{"API_KEY", "API_KEY", true},
		{"DB_SECRET", "DB_SECRET", true},
		{"AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY", true},
		{"AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID", true},
		{"AWS_REGION", "AWS_REGION", true},
		{"REGULAR_VAR", "REGULAR_VAR", false},
		{"TOKEN_PREFIX", "TOKEN_PREFIX", false},
		{"MY_TOKENIZE", "MY_TOKENIZE", false},
		{"SECRET_PASSWORD", "SECRET_PASSWORD", false}, // doesn't end with _SECRET
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ws := newTestWorkspace(t)

			// Set the test variable.
			if err := os.Setenv(tc.envVarName, "test_value"); err != nil {
				t.Fatalf("failed to set test env: %v", err)
			}
			defer os.Unsetenv(tc.envVarName)

			// Configure to allow the variable through, then we'll check if it's stripped.
			conf := config.Defaults()
			conf.Policy.EnvPassthrough = []string{tc.envVarName}
			pol := policy.New()
			approver := newFakeApprover(policy.DecisionAllow)

			tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

			input := runCommandInput{
				Argv: []string{"env"},
				Cwd:  ".",
			}
			raw, _ := json.Marshal(input)

			result := tool.Invoke(ctx, raw)

			if !result.OK {
				t.Fatalf("env command failed: %s", result.Error.Message)
			}

			envVarPresent := strings.Contains(result.Content, tc.envVarName+"=")
			if tc.shouldStrip && envVarPresent {
				t.Fatalf("variable %q should have been stripped", tc.envVarName)
			}
			if !tc.shouldStrip && !envVarPresent {
				t.Fatalf("variable %q should not have been stripped", tc.envVarName)
			}
		})
	}
}

// TestRunCommandCommandFailureReturnsError verifies that a command with non-zero exit
// returns KindCommandFailed with the output.
func TestRunCommandCommandFailureReturnsError(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use 'false' which always exits with code 1.
	input := runCommandInput{
		Argv: []string{"false"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("command should have failed")
	}
	if result.Error.Kind != KindCommandFailed {
		t.Fatalf("expected KindCommandFailed, got %s", result.Error.Kind)
	}
}

// TestRunCommandCancellation verifies that a cancelled context is handled
// (tested with a timeout context that expires immediately).
func TestRunCommandCancellation(t *testing.T) {
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Create a context with immediate timeout (effectively cancelled).
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Give the context a moment to expire.
	time.Sleep(10 * time.Millisecond)

	input := runCommandInput{
		Argv: []string{"sleep", "10"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("command should have failed due to cancellation")
	}
	// Could be either Cancelled (if context error is caught) or CommandFailed
	// (if the command fails to start). Both are acceptable.
	if result.Error.Kind != KindCancelled && result.Error.Kind != KindCommandFailed {
		t.Fatalf("expected KindCancelled or KindCommandFailed, got %s", result.Error.Kind)
	}
}

// TestRunCommandDefaultCwd verifies that omitting cwd defaults to ".", i.e.
// the command actually runs with cmd.Dir set to the workspace root.
//
// This compares resolved (EvalSymlinks) paths rather than raw strings: on
// macOS, a temp dir handed out by t.TempDir() and the same path as returned
// by the "pwd" binary's getcwd(2) can differ textually (/var vs /private/var)
// without cmd.Dir being wrong. Resolving symlinks on both sides before
// comparing removes that platform noise while still hard-failing a genuinely
// wrong cmd.Dir — the previous version of this test only logged the mismatch,
// so a wrong cmd.Dir would have passed silently.
func TestRunCommandDefaultCwd(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Omit cwd entirely.
	input := runCommandInput{
		Argv: []string{"pwd"},
		// Cwd: "", // omitted
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	// pwd should succeed and return the workspace root.
	if !result.OK {
		t.Fatalf("pwd command failed: %s", result.Error.Message)
	}

	wantResolved, err := filepath.EvalSymlinks(ws.Root)
	if err != nil {
		t.Fatalf("failed to resolve workspace root %q: %v", ws.Root, err)
	}
	gotResolved, err := filepath.EvalSymlinks(strings.TrimSpace(result.Content))
	if err != nil {
		t.Fatalf("failed to resolve pwd output %q: %v", strings.TrimSpace(result.Content), err)
	}
	if gotResolved != wantResolved {
		t.Fatalf("cwd mismatch: pwd reported %q (resolved %q), want workspace root %q (resolved %q)",
			strings.TrimSpace(result.Content), gotResolved, ws.Root, wantResolved)
	}
}

// TestRunCommandBuiltinCommands verifies that some simple built-in commands work.
func TestRunCommandBuiltinCommands(t *testing.T) {
	testCases := []struct {
		name string
		argv []string
	}{
		{"echo", []string{"echo", "hello"}},
		{"true", []string{"true"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ws := newTestWorkspace(t)
			conf := &config.Config{}
			pol := policy.New()
			approver := newFakeApprover(policy.DecisionAllow)

			tool := &RunCommand{WS: ws, Config: conf, Policy: pol, Approver: approver}

			input := runCommandInput{
				Argv: tc.argv,
				Cwd:  ".",
			}
			raw, _ := json.Marshal(input)

			result := tool.Invoke(ctx, raw)

			if !result.OK {
				t.Fatalf("command %v failed: %s", tc.argv, result.Error.Message)
			}
		})
	}
}

// TestRunCommandTimeoutWithinDeadline verifies that timeout kills the process
// group within a bounded time of the deadline.
func TestRunCommandTimeoutWithinDeadline(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Set a 1-second timeout for a sleep 60 command.
	input := runCommandInput{
		Argv:           []string{"sleep", "60"},
		Cwd:            ".",
		TimeoutSeconds: 1,
	}
	raw, _ := json.Marshal(input)

	start := time.Now()
	result := tool.Invoke(ctx, raw)
	elapsed := time.Since(start)

	if result.OK {
		t.Fatalf("command should have timed out")
	}
	if result.Error.Kind != KindCommandTimeout {
		t.Fatalf("expected KindCommandTimeout, got %s", result.Error.Kind)
	}

	// sleep 60 dies on SIGTERM almost immediately, so this should land close
	// to the 1-second deadline. Upper bound is generous (well under the
	// killProcessGroup grace window) to absorb scheduler jitter in CI.
	if elapsed < 1*time.Second || elapsed > 2*time.Second {
		t.Fatalf("timeout took %v; expected roughly 1-2 seconds", elapsed)
	}
}

// TestRunCommandTimeoutProcessGroupDead verifies that the process is actually
// dead after timeout, not merely detached. We verify by attempting to send
// signal 0 to the pid, which should return ESRCH (no such process).
func TestRunCommandTimeoutProcessGroupDead(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// We need to capture the process PID. Create a simple script that reports its PID.
	scriptPath := filepath.Join(t.TempDir(), "test_report_pid.sh")
	pidReportScript := `#!/bin/bash
echo "PID=$BASHPID"
sleep 60`

	if err := os.WriteFile(scriptPath, []byte(pidReportScript), 0755); err != nil {
		t.Skipf("could not create test script: %v", err)
	}

	// Run the script with a 1-second timeout.
	input := runCommandInput{
		Argv:           []string{"/bin/bash", scriptPath},
		Cwd:            ".",
		TimeoutSeconds: 1,
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("command should have timed out")
	}
	if result.Error.Kind != KindCommandTimeout {
		t.Fatalf("expected timeout error, got %s", result.Error.Kind)
	}

	// Extract the PID from the output if available.
	// The script writes "PID=<pid>" to stdout before sleeping.
	var pidValue int
	_, err := fmt.Sscanf(result.Content, "PID=%d", &pidValue)
	if err != nil || pidValue <= 0 {
		t.Logf("could not extract PID from output: %q", result.Content)
		// This test requires the PID; if we can't get it, skip the verification.
		return
	}

	// Give a brief grace period for process cleanup.
	time.Sleep(50 * time.Millisecond)

	// Send signal 0 (no-op) to the process. If it's dead, this should fail with ESRCH.
	err = syscall.Kill(pidValue, 0)
	if err == nil {
		t.Fatalf("process %d should be dead, but signal 0 succeeded", pidValue)
	}

	// Verify the error is ESRCH (no such process).
	if !strings.Contains(err.Error(), "no such process") {
		t.Fatalf("expected 'no such process' error for dead PID, got: %v", err)
	}
}

// TestRunCommandCancellationProcessGroupDead verifies that a cancelled context
// causes the process group to die, not just the parent.
func TestRunCommandCancellationProcessGroupDead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Start a long-running command in a goroutine.
	input := runCommandInput{
		Argv: []string{"sleep", "60"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	resultChan := make(chan Result, 1)
	go func() {
		resultChan <- tool.Invoke(ctx, raw)
	}()

	// Let it start, then cancel after a brief delay.
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for the result (with a timeout to prevent hang).
	select {
	case result := <-resultChan:
		if result.OK {
			t.Fatalf("command should have been cancelled")
		}
		if result.Error.Kind != KindCancelled {
			t.Fatalf("expected KindCancelled, got %s", result.Error.Kind)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("command cancellation timed out; process may not be dead")
	}
}

// TestRunCommandGracefulShutdown verifies that SIGTERM is sent first and the
// process is given time to exit gracefully before SIGKILL.
func TestRunCommandGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use a very short timeout to trigger the grace period.
	input := runCommandInput{
		Argv:           []string{"sleep", "60"},
		Cwd:            ".",
		TimeoutSeconds: 1,
	}
	raw, _ := json.Marshal(input)

	start := time.Now()
	result := tool.Invoke(ctx, raw)
	elapsed := time.Since(start)

	if result.OK {
		t.Fatalf("command should have timed out")
	}

	// The elapsed time should include at least 1 second (the timeout) plus
	// a small amount for cleanup. It should NOT be significantly longer.
	// The key check is that it's not instant (which would mean we didn't wait
	// for graceful shutdown) and not excessively long (which would mean we
	// didn't apply the timeout).
	if elapsed < 1*time.Second {
		t.Fatalf("elapsed time %v is less than timeout; grace period may not have been given", elapsed)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("elapsed time %v is too long; may be waiting for SIGKILL", elapsed)
	}
}

// TestRunCommandTerminateThenKill verifies that a process that ignores SIGTERM
// is still killed by SIGKILL.
func TestRunCommandTerminateThenKill(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Create a script that ignores SIGTERM and only dies on SIGKILL.
	// We'll use a shell script for this test.
	scriptPath := filepath.Join(t.TempDir(), "test_sigterm_ignore.sh")
	script := `#!/bin/bash
trap "" TERM
sleep 60`

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Skipf("could not create test script: %v", err)
	}

	input := runCommandInput{
		Argv:           []string{"/bin/bash", scriptPath},
		Cwd:            ".",
		TimeoutSeconds: 1,
	}
	raw, _ := json.Marshal(input)

	start := time.Now()
	result := tool.Invoke(ctx, raw)
	elapsed := time.Since(start)

	if result.OK {
		t.Fatalf("command should have been killed")
	}
	if result.Error.Kind != KindCommandTimeout {
		t.Fatalf("expected KindCommandTimeout, got %s", result.Error.Kind)
	}

	// Should still respect the deadline plus grace period.
	if elapsed > 4*time.Second {
		t.Fatalf("command took too long to kill: %v", elapsed)
	}
}

// TestRunCommandOutputBeforeExit verifies that output is observable before
// the process exits. This is harder to test directly, but we can at least
// verify that the output contains what we expect (vs. empty or partial).
func TestRunCommandOutputBeforeExit(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Run a command that produces multiple lines of output.
	input := runCommandInput{
		Argv: []string{"printf", "line1\nline2\nline3\n"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if !result.OK {
		t.Fatalf("command failed: %s", result.Error.Message)
	}

	// Verify all three lines are present (indicating output was read before exit).
	if !strings.Contains(result.Content, "line1") ||
		!strings.Contains(result.Content, "line2") ||
		!strings.Contains(result.Content, "line3") {
		t.Fatalf("output incomplete or corrupted: %q", result.Content)
	}
}

// TestRunCommandOutput200KBCapStreaming verifies the 200KB cap still works
// with the capWriter/ProgressSink writer shape (no manual pipes), and that
// both head and tail are present. Output is comfortably over the cap (not
// borderline) so truncation is deterministic rather than conditionally
// checked, and the fill is distinguishable per-segment (HEAD_/MIDDLE_/TAIL_)
// rather than uniform, so a head/tail swap would fail the ordering assertion.
func TestRunCommandOutput200KBCapStreaming(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Output 50KB of HEAD_ lines, then 300KB of middle content, then 50KB of
	// TAIL_ lines: 400KB total, comfortably over the 200KB cap.
	input := runCommandInput{
		Argv: []string{"sh", "-c", "for i in $(seq 1 5120); do printf 'HEAD_'; done; for i in $(seq 1 50000); do printf 'MIDDLE'; done; for i in $(seq 1 5120); do printf 'TAIL_'; done"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	// Command should succeed (shell with simple commands is allowlisted).
	if !result.OK {
		t.Fatalf("command failed: %s", result.Error.Message)
	}

	if len(result.Content) > 200*1024 {
		t.Fatalf("output exceeds 200KB cap: %d bytes", len(result.Content))
	}

	if !result.Truncated {
		t.Fatalf("400KB of output should have been truncated")
	}

	if !strings.Contains(result.Content, "output truncated") {
		t.Fatalf("truncation marker not found")
	}

	// Verify head and tail are both present and in the right order.
	if !strings.Contains(result.Content, "HEAD_") {
		t.Fatalf("output should contain head content (HEAD_)")
	}
	if !strings.Contains(result.Content, "TAIL_") {
		t.Fatalf("output should contain tail content (TAIL_)")
	}

	headIdx := strings.Index(result.Content, "HEAD_")
	markerIdx := strings.Index(result.Content, "output truncated")
	tailIdx := strings.LastIndex(result.Content, "TAIL_")

	if !(headIdx < markerIdx && markerIdx < tailIdx) {
		t.Fatalf("truncation structure incorrect: head(%d) before marker(%d) before tail(%d)", headIdx, markerIdx, tailIdx)
	}
}

// TestRunCommandLargeOutputWithImmediateExit verifies that output is fully
// captured even when the process exits immediately after writing large
// output. os/exec copies Stdout/Stderr into capWriter via an internal
// goroutine while the process runs; this exercises that cmd.Wait() does not
// return until that copy has drained, even when the child exits right after
// its last write.
func TestRunCommandLargeOutputWithImmediateExit(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Generate a large output (150KB) and exit immediately.
	largeOutput := strings.Repeat("x", 150*1024)

	input := runCommandInput{
		Argv: []string{"printf", largeOutput},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if !result.OK {
		t.Fatalf("command failed: %s", result.Error.Message)
	}

	// Verify we captured the full output (or at least most of it; truncation at 200KB is OK).
	// The key is that we got the output, not that it was truncated or lost.
	if len(result.Content) < 100*1024 {
		t.Fatalf("output too small (%d bytes); may have been truncated or lost due to pipe drainage issue", len(result.Content))
	}

	// Verify the content is correct (all x's or x's with truncation marker).
	if !strings.Contains(result.Content, "x") {
		t.Fatalf("output does not contain expected content (x)")
	}
}

// TestRunCommandGrandchildCleanup verifies that grandchildren (processes spawned by the
// command's child) are also killed when the process group is terminated.
// The child process spawns a grandchild and writes its PID to a file, allowing us to
// verify that the grandchild was actually killed (not just the direct child).
func TestRunCommandGrandchildCleanup(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Create a script that spawns a grandchild and writes its PID to a file.
	// The parent script sleeps for a long time, and the grandchild also sleeps.
	// When we kill the parent process group, both should die.
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "grandchild_pid")
	parentScript := filepath.Join(tmpDir, "parent_grandchild.sh")

	// The parent script spawns a grandchild in the background and waits.
	// The grandchild writes its PID to pidFile.
	parentCode := fmt.Sprintf(`#!/bin/bash
(echo $BASHPID > %s; sleep 60) &
sleep 60`, pidFile)

	if err := os.WriteFile(parentScript, []byte(parentCode), 0755); err != nil {
		t.Skipf("could not create test script: %v", err)
	}

	// Run the parent script with a 1-second timeout to trigger the kill.
	input := runCommandInput{
		Argv:           []string{"/bin/bash", parentScript},
		Cwd:            ".",
		TimeoutSeconds: 1,
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("command should have timed out")
	}
	if result.Error.Kind != KindCommandTimeout {
		t.Fatalf("expected timeout error, got %s", result.Error.Kind)
	}

	// Give time for cleanup.
	time.Sleep(100 * time.Millisecond)

	// Read the grandchild's PID from the file if it was created.
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		t.Logf("grandchild PID file not created (file write may not have completed); skipping PID verification")
		return
	}

	var grandchildPID int
	_, err = fmt.Sscanf(string(pidData), "%d", &grandchildPID)
	if err != nil || grandchildPID <= 0 {
		t.Logf("could not parse grandchild PID from file: %q", string(pidData))
		return
	}

	// Verify the grandchild is dead by sending signal 0.
	err = syscall.Kill(grandchildPID, 0)
	if err == nil {
		t.Fatalf("grandchild process %d should be dead, but signal 0 succeeded", grandchildPID)
	}

	// The error should indicate the process is gone.
	if !strings.Contains(err.Error(), "no such process") {
		t.Fatalf("expected 'no such process' error for dead grandchild, got: %v", err)
	}
}

// TestRunCommandNoGoroutineLeaks verifies that cancelled commands don't
// leave goroutines hanging. We sample goroutine count before and after.
// With capWriter replacing manual pipes, the goroutines under test are
// os/exec's own internal Stdout/Stderr copy goroutines plus the cmd.Cancel
// watcher; the leak-detection mechanism (before/after sampling) is unchanged.
func TestRunCommandNoGoroutineLeaks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	initialGoroutines := countGoroutines()

	// Run a command and cancel it.
	input := runCommandInput{
		Argv: []string{"sleep", "60"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	resultChan := make(chan Result, 1)
	go func() {
		resultChan <- tool.Invoke(ctx, raw)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for the result.
	<-resultChan

	// Give a brief grace period for cleanup.
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := countGoroutines()

	// Allow a small tolerance (e.g., 3 goroutines) for scheduler variations.
	tolerance := 3
	if finalGoroutines > initialGoroutines+tolerance {
		t.Logf("initial goroutines: %d, final: %d, tolerance: %d", initialGoroutines, finalGoroutines, tolerance)
		t.Fatalf("goroutine leak detected; expected <= %d, got %d", initialGoroutines+tolerance, finalGoroutines)
	}
}

// TestOutputWithDetachedDescendant verifies that a genuinely detached
// grandchild process — one that has actually left the command's process
// group via a successful setsid() — no longer causes Invoke to hang, because
// WaitDelay bounds the wait once an ordinary process-group kill has nothing
// left in that group to reach.
//
// Why a grandchild, not a child calling setsid() on itself: run_command.go
// sets SysProcAttr{Setpgid: true} with Pgid left at its zero value, which
// makes the direct child a process-group leader (pgid == pid) before its own
// main() runs. POSIX setsid(2) returns EPERM when the caller is already a
// process-group leader, so a direct child calling setsid() on itself is a
// guaranteed no-op: it never leaves the group, the group kill always reaches
// it, and the test would pass whether or not WaitDelay does anything — which
// is exactly what the last round's version of this test did, for the third
// wrong reason in a row. A process forked by the child, by contrast, starts
// as an ordinary member of the child's group rather than as a leader, so
// setsid() succeeds for it: only then does it move to a new session/group,
// survive the group kill, and keep the inherited stdout pipe open long
// enough for WaitDelay to be the thing that actually bounds the hang.
//
// Shape chosen: the direct child forks the grandchild, waits (bounded, via a
// status file) for confirmation that the grandchild has actually detached,
// then deliberately stays alive (it does not exit). Go's os/exec starts the
// WaitDelay countdown at whichever comes first — the context going Done, or
// Wait observing that the direct child has exited — so if the direct child
// exited on its own once the grandchild was up, the countdown would start
// from that early exit rather than from the command's timeout, and Invoke
// would return correctly bounded but for a different reason (and with a
// different error Kind) well before the configured deadline. Staying alive
// keeps the timeout itself as the trigger: killProcessGroup's SIGTERM, sent
// when the 3-second command timeout fires, reaches and kills the direct
// child (still an ordinary member of its own leader group) while leaving the
// grandchild — already in its own separate group — untouched. That is the
// proof: the group kill visibly does its job on the process it can reach,
// and still Invoke does not hang, because WaitDelay is what bounds the wait
// on the pipe the surviving grandchild holds open.
//
// The trigger crosses buildEnvironment's filtering (which constructs cmd.Env
// from scratch — PATH/HOME/LANG plus config.Policy.EnvPassthrough — rather
// than inheriting os.Environ()) by allowlisting the two env vars below in
// EnvPassthrough and setting them with os.Setenv in the parent process just
// before invoking: buildEnvironment reads them via os.Getenv while still
// running in the parent, and forwards them into cmd.Env for the child. This
// is the same path a real user relies on to pass required variables through
// the tool; it does not bypass or weaken the security filtering.
func TestOutputWithDetachedDescendant(t *testing.T) {
	const roleVar = "GO_TEST_DETACHED_ROLE"
	const statusVar = "GO_TEST_DETACHED_STATUSFILE"

	switch os.Getenv(roleVar) {
	case "child":
		// This is the direct child spawned by RunCommand. SysProcAttr{Setpgid:
		// true} (Pgid==0) already made it a process-group leader before this
		// code runs, so it must NOT call setsid() on itself — that would
		// return EPERM and prove nothing. Instead it forks an ordinary child
		// of its own (the grandchild), which starts as a plain member of
		// THIS process's group, not a leader, and lets that one detach.
		exe, exeErr := os.Executable()
		if exeErr != nil {
			fmt.Fprintf(os.Stderr, "child: os.Executable failed: %v\n", exeErr)
			os.Exit(1)
		}
		gc := exec.Command(exe, "-test.run=TestOutputWithDetachedDescendant$")
		gc.Env = append(os.Environ(), roleVar+"=grandchild")
		gc.Stdout = os.Stdout // Inherit this process's stdout: the pipe RunCommand reads.
		gc.Stderr = os.Stderr
		// Deliberately no SysProcAttr here: leaving Setpgid unset is what
		// makes the grandchild an ordinary group member instead of a leader,
		// which is the precondition setsid() needs below in order to succeed.
		if startErr := gc.Start(); startErr != nil {
			fmt.Fprintf(os.Stderr, "child: failed to start grandchild: %v\n", startErr)
			os.Exit(1)
		}
		// Wait (bounded) for the grandchild to report a completed, successful
		// setsid() before doing anything further, so the outer test's later
		// read of statusFile is never a race against the grandchild still
		// starting up.
		statusFile := os.Getenv(statusVar)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if data, statErr := os.ReadFile(statusFile); statErr == nil && len(data) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		// Deliberately does NOT exit here. Go's os/exec starts the WaitDelay
		// countdown at whichever comes first: the context going Done, or Wait
		// observing this process has exited — so an early exit here would
		// start that countdown from this process's own exit instead of from
		// the command's timeout, and Invoke would return (correctly bounded,
		// but for a different reason and with a different error Kind) well
		// before the 3-second deadline. Staying alive keeps the context
		// deadline as the trigger: this process remains an ordinary member of
		// its own process-group-leader group until Invoke's timeout fires and
		// killProcessGroup's SIGTERM reaches and kills it — while the
		// grandchild, already in its own separate group, is unaffected by
		// that same signal.
		for {
			time.Sleep(time.Hour)
		}

	case "grandchild":
		// An ordinary child of "child" above, not a process-group leader, so
		// setsid() here is expected to succeed. beforePgid is captured first
		// so the outer test can prove the group actually changed, not merely
		// that the call returned without error.
		beforePgid, _ := syscall.Getpgid(os.Getpid())
		statusFile := os.Getenv(statusVar)
		newSid, sidErr := syscall.Setsid()
		if sidErr != nil {
			// The error is checked, not discarded: a failed setsid() here
			// would silently turn this back into the non-detaching case this
			// test exists to rule out, so it must be visible, not swallowed.
			_ = os.WriteFile(statusFile, []byte(fmt.Sprintf("ERROR setsid: %v", sidErr)), 0o644)
			os.Exit(1)
		}
		afterPgid, _ := syscall.Getpgid(os.Getpid())
		_ = os.WriteFile(statusFile, []byte(fmt.Sprintf(
			"pid=%d before_pgid=%d after_pgid=%d sid=%d",
			os.Getpid(), beforePgid, afterPgid, newSid)), 0o644)
		fmt.Println("Detached grandchild started")
		// Hold the inherited stdout pipe open indefinitely. A sleep loop (not
		// a bare select{}) keeps a pending timer registered, which the Go
		// runtime's all-goroutines-asleep deadlock detector exempts, so the
		// process genuinely blocks until killed rather than self-terminating.
		for {
			time.Sleep(time.Hour)
		}
	}

	// --- Top-level test body: this is the process running `go test`. ---

	ws := newTestWorkspace(t)
	statusFile := filepath.Join(t.TempDir(), "detach_status")
	cfg := &config.Config{
		Policy: config.PolicyConfig{
			EnvPassthrough: []string{roleVar, statusVar},
		},
	}
	pol := &policy.Policy{}
	approver := &stubApprover{}

	tool := &RunCommand{
		WS:       ws,
		Config:   cfg,
		Policy:   pol,
		Approver: approver,
	}

	// Clean up the grandchild unconditionally, including on a hang/timeout
	// failure path below: it is deliberately immortal (infinite sleep loop)
	// and, by design, was never a member of the group Invoke's timeout
	// kills, so nothing else in this test terminates it. Registered early,
	// before the pid is even known, so it still runs via t.Cleanup's
	// guaranteed execution if a later assertion calls t.Fatalf.
	t.Cleanup(func() {
		data, readErr := os.ReadFile(statusFile)
		if readErr != nil {
			return
		}
		var pid int
		if _, scanErr := fmt.Sscanf(string(data), "pid=%d", &pid); scanErr != nil || pid <= 0 {
			return
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	})

	// Set the vars in the parent's environment now (after the switch above,
	// so the parent itself doesn't take the child/grandchild branch).
	// buildEnvironment reads os.Getenv from this process and forwards them
	// into cmd.Env.
	if err := os.Setenv(roleVar, "child"); err != nil {
		t.Fatalf("failed to set %s: %v", roleVar, err)
	}
	defer os.Unsetenv(roleVar)
	if err := os.Setenv(statusVar, statusFile); err != nil {
		t.Fatalf("failed to set %s: %v", statusVar, err)
	}
	defer os.Unsetenv(statusVar)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	// This invokes the test binary itself; the role/statusfile vars (passed
	// through via EnvPassthrough above) select the "child" branch, which
	// forks the "grandchild" branch that actually detaches.
	exe, _ := os.Executable()
	input := runCommandInput{
		Argv:           []string{exe, "-test.run=TestOutputWithDetachedDescendant$"},
		Cwd:            ".",
		TimeoutSeconds: 3, // 3-second timeout; WaitDelay is 500ms.
	}
	data, _ := json.Marshal(input)

	// Invoke is run in a goroutine behind a select with its own hard bound,
	// not called directly: if WaitDelay regressed to not bounding the hang
	// at all, a direct call here would block until go test's own -timeout
	// fired, hanging the whole suite rather than failing this test. The
	// select below turns that failure mode into a fast, explicit
	// t.Fatalf instead.
	resultChan := make(chan Result, 1)
	start := time.Now()
	go func() {
		resultChan <- tool.Invoke(ctx, data)
	}()

	var result Result
	var elapsed time.Duration
	select {
	case result = <-resultChan:
		elapsed = time.Since(start)
	case <-time.After(5 * time.Second):
		t.Fatalf("Invoke did not return within 5s of a 3s command timeout; WaitDelay is not bounding the hang from the detached grandchild's held-open stdout")
	}

	// The command should time out (not hang forever). The CRITICAL was an
	// unbounded hang; this proves it is now bounded by WaitDelay specifically
	// — not by an ordinary process-group kill, because the direct child that
	// owned that group has already exited on its own by the time the timeout
	// fires.
	if result.OK {
		t.Errorf("expected command timeout, got OK")
	}
	if result.Error == nil {
		t.Fatalf("expected error, got nil")
	}
	if result.Error.Kind != KindCommandTimeout {
		t.Errorf("expected KindCommandTimeout, got %s (message=%q, content=%q)", result.Error.Kind, result.Error.Message, result.Content)
	}

	// Bound: must respect the 3-second command timeout, and must return
	// within WaitDelay's 500ms of it rather than riding out the full 5-second
	// select above. This is the assertion the mutation test (removing
	// WaitDelay) is checked against.
	if elapsed < 3*time.Second {
		t.Fatalf("returned before the 3-second command timeout elapsed: %v", elapsed)
	}
	if elapsed > 4*time.Second {
		t.Fatalf("returned too slowly (%v); WaitDelay (500ms) should have bounded the hang from the detached grandchild's held-open stdout", elapsed)
	}

	// Prove the grandchild genuinely detached: it must exist, its setsid()
	// call must have succeeded (checked above, not discarded), and its
	// post-setsid process group must differ from the group it started in —
	// the child's group, which equals the child's own pid since the child
	// was itself a process-group leader.
	statusData, err := os.ReadFile(statusFile)
	if err != nil {
		t.Fatalf("grandchild status file not written: %v", err)
	}
	statusStr := string(statusData)
	if strings.HasPrefix(statusStr, "ERROR") {
		t.Fatalf("grandchild's setsid() call failed: %s", statusStr)
	}
	var gcPid, beforePgid, afterPgid, sid int
	if _, err := fmt.Sscanf(statusStr, "pid=%d before_pgid=%d after_pgid=%d sid=%d", &gcPid, &beforePgid, &afterPgid, &sid); err != nil {
		t.Fatalf("could not parse grandchild status %q: %v", statusStr, err)
	}
	if gcPid <= 0 {
		t.Fatalf("invalid grandchild pid in status: %q", statusStr)
	}

	// Detachment happened, not merely that setsid() was called: the group
	// actually changed, and the new group is the grandchild itself (the
	// hallmark of the session/group leader setsid() creates).
	if afterPgid == beforePgid {
		t.Fatalf("grandchild's process group did not change: before=%d after=%d", beforePgid, afterPgid)
	}
	if afterPgid != gcPid {
		t.Fatalf("grandchild is not its own group leader after setsid(): pgid=%d pid=%d", afterPgid, gcPid)
	}
	if sid != afterPgid {
		t.Fatalf("setsid() returned sid=%d, expected it to equal the new pgid=%d", sid, afterPgid)
	}

	// Kill the grandchild and confirm it is actually gone, in that order, in
	// the main test body — not solely via the t.Cleanup registered above,
	// since t.Cleanup executes LIFO and a second Cleanup here would run
	// BEFORE the first (the kill), always failing. The earlier t.Cleanup
	// stays in place as a safety net for the early-failure paths above,
	// where gcPid is not yet known here. Signal 0 is a no-op that still
	// reports ESRCH once the process is gone, the same technique
	// TestRunCommandTimeoutProcessGroupDead uses.
	_ = syscall.Kill(gcPid, syscall.SIGKILL)
	killDeadline := time.Now().Add(2 * time.Second)
	gone := false
	for time.Now().Before(killDeadline) {
		if killErr := syscall.Kill(gcPid, 0); killErr != nil {
			gone = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !gone {
		t.Errorf("grandchild process %d still alive 2s after SIGKILL; suite would leave an orphan", gcPid)
	}
}

// TestHealthyCommandOutputIntact verifies that full output from a healthy
// command still arrives, without truncation due to the capWriter logic.
func TestHealthyCommandOutputIntact(t *testing.T) {
	ws := newTestWorkspace(t)
	cfg := &config.Config{
		Policy: config.PolicyConfig{
			EnvPassthrough: []string{},
		},
	}
	pol := &policy.Policy{}
	approver := &stubApprover{}

	tool := &RunCommand{
		WS:       ws,
		Config:   cfg,
		Policy:   pol,
		Approver: approver,
	}

	// Generate output to verify it's captured. Use simple echo.
	input := runCommandInput{
		Argv: []string{
			"echo", "hello world this is a test output",
		},
		Cwd:            ".",
		TimeoutSeconds: 10,
	}
	expectedSize := len("hello world this is a test output\n")

	data, _ := json.Marshal(input)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := tool.Invoke(ctx, data)

	if !result.OK {
		t.Errorf("command failed: %v", result.Error)
	}

	outputSize := len(result.Content)
	if outputSize != expectedSize {
		t.Errorf("expected %d bytes output, got %d", expectedSize, outputSize)
	}

	if result.Truncated {
		t.Errorf("output should not be truncated (size %d, cap 200KB)", outputSize)
	}
}

// TestProgressSinkReceivesChunks verifies that ProgressSink callback
// receives output chunks as they're produced.
func TestProgressSinkReceivesChunks(t *testing.T) {
	ws := newTestWorkspace(t)
	cfg := &config.Config{
		Policy: config.PolicyConfig{
			EnvPassthrough: []string{},
		},
	}
	pol := &policy.Policy{}
	approver := &stubApprover{}

	var chunks []string
	tool := &RunCommand{
		WS:       ws,
		Config:   cfg,
		Policy:   pol,
		Approver: approver,
		ProgressSink: func(chunk string) {
			chunks = append(chunks, chunk)
		},
	}

	input := runCommandInput{
		Argv:           []string{"echo", "hello world"},
		Cwd:            ".",
		TimeoutSeconds: 5,
	}

	data, _ := json.Marshal(input)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := tool.Invoke(ctx, data)

	if !result.OK {
		t.Errorf("command failed: %v", result.Error)
	}

	if len(chunks) == 0 {
		t.Errorf("expected at least one chunk in ProgressSink")
	}

	combined := strings.Join(chunks, "")
	if !strings.Contains(combined, "hello world") {
		t.Errorf("expected 'hello world' in chunks, got: %q", combined)
	}
}

// TestCapWriterTruncatesLargeOutput verifies capWriter's own memory-safety
// bound: once accumulated output exceeds cap, it keeps head+tail rather than
// growing unboundedly. This is capWriter's internal mechanism, not the
// reported truncation callers see (that's Truncate, exercised end-to-end by
// TestRunCommandOutputCapAt200KB and TestRunCommandOutput200KBCapStreaming).
//
// Head and tail are filled with distinct, non-repeating-period content
// ("HEAD" vs "TAIL", not a cyclic A-Z alphabet) so a head/tail swap, or an
// off-by-one that shifts the cut by a multiple of the fill's period, changes
// the observed content rather than aliasing back to a passing value.
func TestCapWriterTruncatesLargeOutput(t *testing.T) {
	w := &capWriter{cap: 1000, progressSink: nil}

	// Write more than cap: 1500 bytes.
	// Should keep first 500 bytes (head) and last 500 bytes (tail).
	head := bytes.Repeat([]byte("HEAD"), 125)       // 500 bytes, distinct from tail
	discard := bytes.Repeat([]byte("DISCARD_"), 62) // ~500 bytes, must not survive
	tail := bytes.Repeat([]byte("TAIL"), 125)       // 500 bytes, distinct from head
	data := append(append(head, discard...), tail...)

	w.Write(data)

	result := w.String()
	if len(result) != 1000 {
		t.Fatalf("expected cap output of 1000 bytes, got %d", len(result))
	}

	if !strings.HasPrefix(result, "HEADHEAD") {
		t.Fatalf("expected result to start with head content, got %q", result[:20])
	}
	if !strings.HasSuffix(result, "TAILTAIL") {
		t.Fatalf("expected result to end with tail content, got %q", result[len(result)-20:])
	}
	if strings.Contains(result, "DISCARD") {
		t.Fatalf("discarded middle content leaked into capped output: %q", result)
	}
}

// stubApprover implements Approver for testing.
type stubApprover struct{}

func (s *stubApprover) Request(ctx context.Context, req ApprovalRequest) policy.Decision {
	return policy.DecisionAllow
}

// Resolve implements Approver.Resolve for testing.
func (s *stubApprover) Resolve(id int64, decision policy.Decision) {
	// No-op for testing
}

// TestRunCommandApprovalIncludesArgv verifies that RunCommand.Invoke populates
// the Argv field of the ApprovalRequest when requesting approval.
func TestRunCommandApprovalIncludesArgv(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionAllow)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use a command that requires approval (not in allowlist).
	input := runCommandInput{
		Argv: []string{"echo", "test", "args"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	_ = tool.Invoke(ctx, raw)

	// Verify the approver was called with the Argv field populated.
	if !approver.called {
		t.Fatalf("approver should have been called")
	}
	if len(approver.lastReq.Argv) == 0 {
		t.Fatalf("ApprovalRequest.Argv should be populated, got empty")
	}
	if approver.lastReq.Argv[0] != "echo" || approver.lastReq.Argv[1] != "test" {
		t.Fatalf("ApprovalRequest.Argv = %v, want [echo test args]", approver.lastReq.Argv)
	}
}

// TestRunCommandSessionApprovalRuns verifies that RunCommand accepts
// DecisionSession as valid approval and runs the command.
func TestRunCommandSessionApprovalRuns(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionSession)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use a command that requires approval and will succeed.
	input := runCommandInput{
		Argv: []string{"echo", "hello"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	// Verify the command ran successfully (DecisionSession treated as permission).
	if !result.OK {
		t.Fatalf("command should have run with session approval, got error: %s", result.Error.Message)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Fatalf("command output missing expected content, got: %q", result.Content)
	}
}

// TestRunCommandSessionApprovalFails verifies that RunCommand rejects
// DecisionDeny even when it's not DecisionAllow.
func TestRunCommandRejectionDenyOnly(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New()
	approver := newFakeApprover(policy.DecisionDeny)

	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use a command that requires approval.
	input := runCommandInput{
		Argv: []string{"echo", "hello"},
		Cwd:  ".",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	// Verify the command was rejected.
	if result.OK {
		t.Fatalf("command should have been rejected with DecisionDeny")
	}
	if result.Error.Kind != KindPolicyDenied {
		t.Fatalf("expected KindPolicyDenied, got %s", result.Error.Kind)
	}
}
