package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/workspace"
)

const (
	// maxOutputBytes is the reported output cap: the size Content is cut to,
	// and the threshold against which Truncated and the elision marker are
	// decided. Truncate (tool.go) is the single truncation mechanism in this
	// path — it owns the marker, the rune/line-boundary safety, and the
	// head/tail split that the model-facing output actually shows.
	maxOutputBytes = 200 << 10

	// writerSafetyCapBytes bounds capWriter's own buffering so a pathological
	// command (e.g. `yes`) can't make Invoke hold unbounded memory. It is
	// deliberately larger than maxOutputBytes: whenever real output exceeds
	// maxOutputBytes, it still exceeds maxOutputBytes after passing through
	// capWriter (since writerSafetyCapBytes > maxOutputBytes), so Truncate
	// always sees over-cap input and always sets the marker and Truncated
	// correctly. capWriter's own head/tail cut in Write is therefore a
	// last-resort memory bound, not a second truncation decision — collapsing
	// what used to be two competing cap points (capWriter's exact-200KB cut
	// with no marker, and Truncate's 200KB cut with one) into a single
	// reporting authority.
	writerSafetyCapBytes = maxOutputBytes * 4
)

// RunCommand runs a shell command with policy approval.
// Commands must be provided as explicit argv (no shell), environment is filtered
// to remove secrets, and output is capped at 200KB.
type RunCommand struct {
	WS           *workspace.Workspace
	Config       *config.Config
	Policy       *policy.Policy
	Approver     Approver
	ProgressSink func(chunk string) // Optional: receives output chunks as they arrive
}

type runCommandInput struct {
	Argv           []string `json:"argv"`
	Cwd            string   `json:"cwd"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

// Name implements Tool.
func (t *RunCommand) Name() string { return "run_command" }

// Description implements Tool.
func (t *RunCommand) Description() string {
	return "Run a command in the workspace. Commands must be provided as explicit argv, " +
		"not a single string. The environment is filtered to remove secrets."
}

// Schema implements Tool.
func (t *RunCommand) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "argv": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Command to run as an array, e.g. [\"go\", \"test\", \"./...\"]"
    },
    "cwd": {
      "type": "string",
      "description": "Workspace-relative working directory. Defaults to \".\"."
    },
    "timeout_seconds": {
      "type": "integer",
      "description": "Command timeout in seconds. Maximum 3600."
    }
  },
  "required": ["argv"],
  "additionalProperties": false
}`)
}

// Invoke implements Tool.
func (t *RunCommand) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in runCommandInput
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}

	// Validate argv is not a single string (common mistake).
	if len(in.Argv) == 0 {
		return Fail(KindToolInputInvalid,
			"argv is required and cannot be empty. Pass the command as an array, e.g. [\"go\", \"test\", \"./...\"]")
	}

	// Reject single-string commands that look like someone passed "go test ..." as one string.
	if len(in.Argv) == 1 && strings.ContainsAny(in.Argv[0], " \t") {
		return Fail(KindToolInputInvalid,
			"command appears to be a single string: %q. Pass each argument separately as an array: [\"%s\"]",
			in.Argv[0], strings.Join(strings.Fields(in.Argv[0]), "\", \""))
	}

	// Resolve the working directory.
	if in.Cwd == "" {
		in.Cwd = "."
	}
	absDir, res, bad := resolve(t.WS, in.Cwd)
	if bad {
		return res
	}

	// Consult policy.
	decision, reason := t.Policy.ForCommand(in.Argv)
	if decision == policy.DecisionDeny {
		return Fail(KindPolicyDenied, "command denied by policy: %s", reason)
	}

	// Request approval if needed.
	if decision == policy.DecisionAskUser {
		appDecision := t.Approver.Request(ctx, ApprovalRequest{
			ID:          1, // TODO: provided by approval flow in M3
			Operation:   policy.OperationCommand,
			Description: fmt.Sprintf("Run: %s", strings.Join(in.Argv, " ")),
			Argv:        in.Argv,
		})
		if appDecision != policy.DecisionAllow && appDecision != policy.DecisionSession {
			return Fail(KindPolicyDenied, "command rejected")
		}
	}

	// Build the environment: start with PATH, HOME, LANG.
	env := t.buildEnvironment()

	// Parse timeout; default to 3600s (1 hour) if not specified.
	timeout := time.Duration(in.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 3600 * time.Second
	}
	timeoutSeconds := int(timeout.Seconds()) // Use resolved value for error messages

	// Create a context with timeout for the command.
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create the command with explicit argv (no shell).
	cmd := exec.CommandContext(cmdCtx, in.Argv[0], in.Argv[1:]...) // #nosec G204 -- argv[0] gated by policy.ForCommand (internal/policy/policy.go): allowlist match (exact, or prefix for extendable entries), session grant, or explicit user approval; shells always ask; no shell is used; cmd.Dir resolved within the workspace
	cmd.Dir = absDir
	cmd.Env = env

	// stdin is /dev/null: a command that prompts must fail immediately, not hang.
	devNull, devNullErr := os.Open(os.DevNull)
	if devNullErr != nil {
		return Fail(KindInternal, "failed to open /dev/null: %v", devNullErr)
	}
	// The file is /dev/null opened read-only, so a close error carries no information.
	defer func() { _ = devNull.Close() }()
	cmd.Stdin = devNull

	// Set up process group for cancellation: Setpgid makes the process a leader,
	// so we can kill the group (including children) later.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Install custom cancellation handler. When the context is done, cancel
	// will be called to kill the process group (SIGTERM, grace period, SIGKILL).
	// This ensures the group kill runs against a live process, not after Wait returns.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		t.killProcessGroup(cmd)
		return nil
	}

	// Create a writer that collects output and forwards chunks to ProgressSink.
	// This avoids pipe-based I/O, which can hang if detached descendant processes
	// survive the group kill and hold the write end open. Its cap is a memory
	// safety bound, not the reported truncation point — see writerSafetyCapBytes.
	outputWriter := &capWriter{cap: writerSafetyCapBytes, progressSink: t.ProgressSink}
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter

	// Set WaitDelay to bound the time spent waiting on unexpected delays:
	// both a child that refuses to exit after context cancellation, and a child
	// that exits but leaves I/O pipes unclosed (e.g., a detached descendant).
	// 500ms is long enough for healthy commands' final output flushes, short
	// enough to prevent indefinite hangs from detached descendants.
	cmd.WaitDelay = 500 * time.Millisecond

	// Start the command.
	startErr := cmd.Start()
	if startErr != nil {
		return Fail(KindCommandFailed, "failed to start command: %v", startErr)
	}

	// Wait for the command to finish, handling cancellation and timeout.
	// cmd.WaitDelay ensures this returns within a bounded time even if detached
	// descendants still hold output pipes open. os/exec copies Stdout/Stderr
	// to our outputWriter during Wait(), so we must call Wait() before reading output.
	waitErr := cmd.Wait()

	// Now safe to read the accumulated output from our writer.
	output := outputWriter.String()

	// Check for cancellation or timeout.
	if cmdCtx.Err() != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			// Timeout occurred. cmd.Cancel has already been called to kill the group.
			return Fail(KindCommandTimeout, "command timeout after %d seconds", timeoutSeconds)
		}
		if cmdCtx.Err() == context.Canceled {
			// Timeout context was cancelled. cmd.Cancel has already been called to kill the group.
			return Fail(KindCancelled, "cancelled during command execution")
		}
	}

	// Check if the outer context was cancelled while we were running.
	if ctx.Err() != nil {
		// Outer context (not the timeout context) was cancelled.
		// cmd.Cancel has already been called, but make an extra call here for safety
		// in case there was an issue with the cancellation handler.
		t.killProcessGroup(cmd)
		return Fail(KindCancelled, "cancelled during command execution")
	}

	// Handle execution errors.
	if waitErr != nil {
		// Command failed with non-zero exit.
		if _, ok := waitErr.(*exec.ExitError); ok {
			// The command ran but exited with an error; include its output.
			content, truncated := Truncate(output, maxOutputBytes)
			return Result{
				OK:             false,
				Content:        content,
				DisplaySummary: "command failed",
				Truncated:      truncated,
				Error:          &Error{Kind: KindCommandFailed, Message: waitErr.Error()},
			}
		}
		// Some other execution error (e.g., command not found).
		return Fail(KindCommandFailed, "failed to run command: %v", waitErr)
	}

	// Success: return the output, capped at 200KB.
	content, truncated := Truncate(output, maxOutputBytes)
	summary := "completed"
	if truncated {
		summary += " (output capped)"
	}
	return OKResult(content, summary, truncated)
}

// capWriter is an io.Writer that accumulates output up to a cap, keeping the
// first and last portions (head+tail) once it overflows, and forwards chunks
// to a ProgressSink. It is thread-safe and used to replace pipe-based I/O,
// which could hang if detached descendant processes survive cancellation.
//
// capWriter's cap is a memory safety bound, not the truncation callers see:
// Invoke constructs it with writerSafetyCapBytes (larger than maxOutputBytes)
// and applies Truncate to its output afterwards. Truncate is what decides
// Content, Truncated, and the elision marker. See writerSafetyCapBytes for
// why that ordering guarantees Truncate always fires correctly.
type capWriter struct {
	mu           sync.Mutex
	cap          int          // Maximum accumulated bytes
	buf          []byte       // Buffered output
	progressSink func(string) // Optional callback for output chunks
}

// Write implements io.Writer for capWriter.
func (w *capWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(p)

	// Forward chunk to ProgressSink immediately if set.
	if w.progressSink != nil {
		w.progressSink(string(p))
	}

	// Append to buffer.
	w.buf = append(w.buf, p...)

	// If buffer exceeds cap, truncate to head+tail.
	if len(w.buf) > w.cap {
		half := w.cap / 2
		// Keep first half and last half.
		head := w.buf[:half]
		tail := w.buf[len(w.buf)-(w.cap-half):]
		w.buf = append(head, tail...)
	}

	return n, nil
}

// String returns the accumulated output (head + tail, truncated if necessary).
func (w *capWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.buf)
}

// killProcessGroup kills the process group of the command.
// It sends SIGTERM first, waits 2 seconds for graceful shutdown,
// then sends SIGKILL to ensure the process is dead.
// Note: the grace-period poll uses signal 0 (no-op), which succeeds for
// both live processes and zombies. Thus even a promptly-dying child will
// not cause the poll to exit early; every cancellation waits the full
// 2-second grace window. This is acceptable because cancellation is rare
// and 2 seconds is a bounded cost. Proper detection would require checking
// /proc on Linux or equivalent, which is fragile across OSes.
func (t *RunCommand) killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid
	if pid <= 0 {
		return
	}

	// Send SIGTERM to the process group (-pid signals the group).
	_ = syscall.Kill(-pid, syscall.SIGTERM)

	// Wait up to 2 seconds, polling for process death.
	// Note: signal 0 (no-op) succeeds for both live and zombie processes,
	// so this poll does not exit early; it always waits the full 2 seconds.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		// Attempt signal 0 (no-op); succeeds for live processes and zombies.
		if err := syscall.Kill(pid, 0); err != nil {
			// Process is completely gone (unlikely to reach here).
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Still alive after 2 seconds; send SIGKILL to the group.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// buildEnvironment constructs the environment for the command.
// It includes PATH, HOME, LANG, plus any vars in config.Policy.EnvPassthrough,
// then strips anything matching *_TOKEN, *_KEY, *_SECRET, or AWS_*.
func (t *RunCommand) buildEnvironment() []string {
	// Start with the pass-through set.
	passthrough := map[string]string{}

	// Always include these.
	for _, key := range []string{"PATH", "HOME", "LANG"} {
		if val := os.Getenv(key); val != "" {
			passthrough[key] = val
		}
	}

	// Add config-specified pass-through variables.
	if t.Config != nil {
		for _, key := range t.Config.Policy.EnvPassthrough {
			if val := os.Getenv(key); val != "" {
				passthrough[key] = val
			}
		}
	}

	// Strip secret patterns: *_TOKEN, *_KEY, *_SECRET, AWS_*.
	// These are stripped even if allowlisted, which is the test case.
	stripPatterns := func(key string) bool {
		return strings.HasSuffix(key, "_TOKEN") ||
			strings.HasSuffix(key, "_KEY") ||
			strings.HasSuffix(key, "_SECRET") ||
			strings.HasPrefix(key, "AWS_")
	}

	for key := range passthrough {
		if stripPatterns(key) {
			delete(passthrough, key)
		}
	}

	// Convert to []string.
	var env []string
	for key, val := range passthrough {
		env = append(env, key+"="+val)
	}
	return env
}
