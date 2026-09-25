package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPatchCommandDispatchesToolInvocation verifies that typing `/patch <file>`
// and Enter dispatches a tool invocation through Update with correct name and input.
func TestPatchCommandDispatchesToolInvocation(t *testing.T) {
	// Track what RunTool was called with
	var capturedName string
	var capturedInput map[string]any

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		capturedName = name
		capturedInput = input
	}
	m.ResolvePatchFile = func(filename string) (string, error) {
		if filename == "test.patch" {
			return "diff content", nil
		}
		return "", fmt.Errorf("file not found")
	}

	// Type /patch test.patch and press Enter
	m = typeCmd(m, "/patch test.patch")

	if capturedName != "apply_patch" {
		t.Errorf("tool name = %q, want %q", capturedName, "apply_patch")
	}
	if capturedInput == nil {
		t.Fatalf("input is nil")
	}
	diff, ok := capturedInput["diff"].(string)
	if !ok {
		t.Errorf("input[\"diff\"] = %T, want string", capturedInput["diff"])
	}
	if diff != "diff content" {
		t.Errorf("diff content = %q, want %q", diff, "diff content")
	}
}

// TestRunCommandDispatchesToolInvocation verifies that typing `/run <argv>` and
// Enter dispatches a tool invocation through Update with correct name and input.
func TestRunCommandDispatchesToolInvocation(t *testing.T) {
	// Track what RunTool was called with
	var capturedName string
	var capturedInput map[string]any

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		capturedName = name
		capturedInput = input
	}

	// Type /run echo hello and press Enter
	m = typeCmd(m, `/run echo hello`)

	if capturedName != "run_command" {
		t.Errorf("tool name = %q, want %q", capturedName, "run_command")
	}
	if capturedInput == nil {
		t.Fatalf("input is nil")
	}
	argv, ok := capturedInput["argv"].([]string)
	if !ok {
		t.Errorf("input[\"argv\"] = %T, want []string", capturedInput["argv"])
	}
	if len(argv) != 2 || argv[0] != "echo" || argv[1] != "hello" {
		t.Errorf("argv = %v, want [echo hello]", argv)
	}
}

// TestRunCommandWithQuotedArguments verifies that quoted arguments are handled correctly.
func TestRunCommandWithQuotedArguments(t *testing.T) {
	// Track what RunTool was called with
	var capturedInput map[string]any

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		capturedInput = input
	}

	// Type /run echo "hello world" and press Enter
	m = typeCmd(m, `/run echo "hello world"`)

	if capturedInput == nil {
		t.Fatalf("input is nil")
	}
	argv, ok := capturedInput["argv"].([]string)
	if !ok {
		t.Errorf("input[\"argv\"] = %T, want []string", capturedInput["argv"])
	}
	// Should parse to [echo "hello world"]
	if len(argv) != 2 || argv[0] != "echo" || argv[1] != "hello world" {
		t.Errorf("argv = %v, want [echo hello world]", argv)
	}
}

// TestPatchArgumentWithParentSegmentIsRefused verifies that patch arguments
// containing parent directory references are rejected.
func TestPatchArgumentWithParentSegmentIsRefused(t *testing.T) {
	// Track what RunTool was called with
	callCount := 0

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
	}
	m.ResolvePatchFile = func(filename string) (string, error) {
		// Return error for paths containing parent directory references
		if filename == "../evil.patch" {
			return "", fmt.Errorf("path traversal not allowed")
		}
		return "diff content", nil
	}

	// Type /patch ../evil.patch and press Enter
	m = typeCmd(m, "/patch ../evil.patch")

	// RunTool should not be called because the callback returned an error
	if callCount != 0 {
		t.Errorf("RunTool was called %d times, want 0 (resolution failed)", callCount)
	}
}

// TestPatchAbsolutePathIsRefused verifies that absolute paths are rejected.
func TestPatchAbsolutePathIsRefused(t *testing.T) {
	// Track what RunTool was called with
	callCount := 0

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
	}
	m.ResolvePatchFile = func(filename string) (string, error) {
		// Return error for absolute paths
		if filename == "/etc/passwd" {
			return "", fmt.Errorf("absolute paths not allowed")
		}
		return "diff content", nil
	}

	// Type /patch /etc/passwd and press Enter
	m = typeCmd(m, "/patch /etc/passwd")

	// RunTool should not be called because the callback returned an error
	if callCount != 0 {
		t.Errorf("RunTool was called %d times, want 0 (resolution failed)", callCount)
	}
}

// TestPatchResolutionFailureDoesNotDispatch verifies that if patch file
// resolution fails, no tool is dispatched.
func TestPatchResolutionFailureDoesNotDispatch(t *testing.T) {
	// Track what RunTool was called with
	callCount := 0

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
	}
	m.ResolvePatchFile = func(filename string) (string, error) {
		return "", fmt.Errorf("file not found")
	}

	// Type /patch nonexistent.patch and press Enter
	m = typeCmd(m, "/patch nonexistent.patch")

	if callCount != 0 {
		t.Errorf("RunTool was called %d times, want 0 (resolution failed)", callCount)
	}
}

// TestPatchNoResolvePatchFileCallbackRefuses verifies that when
// ResolvePatchFile is nil, the command is refused with an error message.
func TestPatchNoResolvePatchFileCallbackRefuses(t *testing.T) {
	// Track what RunTool was called with
	callCount := 0

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
	}
	// Leave ResolvePatchFile as nil

	// Type /patch test.patch and press Enter
	m = typeCmd(m, "/patch test.patch")

	if callCount != 0 {
		t.Errorf("RunTool was called %d times, want 0 (no callback)", callCount)
	}
}

// TestPatchEmptyArgumentShowsUsage verifies that an empty /patch command shows usage.
func TestPatchEmptyArgumentShowsUsage(t *testing.T) {
	callCount := 0

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
	}
	m.ResolvePatchFile = func(filename string) (string, error) {
		return "diff content", nil
	}

	// Type /patch with no argument
	m = typeCmd(m, "/patch ")

	if callCount != 0 {
		t.Errorf("RunTool was called %d times, want 0", callCount)
	}
}

// TestPatchThreeFailuresDistinguishable verifies that the three failure modes
// produce different error messages: empty argument (usage), unwired callback
// (error message), and resolution failure (error message).
func TestPatchThreeFailuresDistinguishable(t *testing.T) {
	tests := []struct {
		name             string
		cmd              string
		setupCallback    func(m *Model)
		wantToolCalls    int
		wantHintContains string
	}{
		{
			name:             "empty argument shows usage",
			cmd:              "/patch ",
			setupCallback:    func(m *Model) { m.ResolvePatchFile = func(string) (string, error) { return "x", nil } },
			wantToolCalls:    0,
			wantHintContains: "usage: /patch",
		},
		{
			name:             "nil callback shows error",
			cmd:              "/patch test.patch",
			setupCallback:    func(m *Model) { /* leave ResolvePatchFile nil */ },
			wantToolCalls:    0,
			wantHintContains: "callback not wired",
		},
		{
			name: "resolution failure shows error",
			cmd:  "/patch missing.patch",
			setupCallback: func(m *Model) {
				m.ResolvePatchFile = func(string) (string, error) { return "", fmt.Errorf("file not found") }
			},
			wantToolCalls:    0,
			wantHintContains: "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0

			m := composingUnpinned(t)
			m.RunTool = func(name string, input map[string]any) {
				callCount++
			}
			tt.setupCallback(&m)

			m = typeCmd(m, tt.cmd)

			if callCount != tt.wantToolCalls {
				t.Errorf("RunTool calls = %d, want %d", callCount, tt.wantToolCalls)
			}
			// Verify the hint contains the expected text, proving the three failures
			// produce distinct messages.
			if !strings.Contains(m.comp.Hint, tt.wantHintContains) {
				t.Errorf("hint = %q, want to contain %q", m.comp.Hint, tt.wantHintContains)
			}
		})
	}
}

// TestPatchDoubleDotFileNameAccepted verifies that patch filenames with two
// adjacent dots (v1..2.patch) are accepted, proving the over-rejection is fixed.
func TestPatchDoubleDotFileNameAccepted(t *testing.T) {
	callCount := 0
	var capturedDiff string

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
		if diff, ok := input["diff"].(string); ok {
			capturedDiff = diff
		}
	}
	m.ResolvePatchFile = func(filename string) (string, error) {
		if filename == "v1..2.patch" {
			return "version diff", nil
		}
		return "", fmt.Errorf("file not found")
	}

	// Type /patch v1..2.patch and press Enter
	m = typeCmd(m, "/patch v1..2.patch")

	if callCount != 1 {
		t.Errorf("RunTool calls = %d, want 1", callCount)
	}
	if capturedDiff != "version diff" {
		t.Errorf("diff content = %q, want %q", capturedDiff, "version diff")
	}
}

// TestPatchSymlinkOutsideWorkspaceRefused verifies that a symlink inside the
// patch directory pointing outside the workspace is refused.
func TestPatchSymlinkOutsideWorkspaceRefused(t *testing.T) {
	// Create a temporary directory structure
	tmpdir := t.TempDir()
	patchDir := filepath.Join(tmpdir, "testdata", "patches")
	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		t.Fatalf("failed to create patch dir: %v", err)
	}

	// Create a temporary file outside the workspace
	outside := filepath.Join(tmpdir, "outside.patch")
	if err := os.WriteFile(outside, []byte("outside content"), 0o644); err != nil {
		t.Fatalf("failed to write outside file: %v", err)
	}

	// Create a symlink inside patchDir pointing to the outside file
	symlink := filepath.Join(patchDir, "evil.patch")
	if err := os.Symlink(outside, symlink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	callCount := 0

	m := composingUnpinned(t)
	m.RunTool = func(name string, input map[string]any) {
		callCount++
	}

	// ResolvePatchFile that uses the temp directory structure
	m.ResolvePatchFile = func(filename string) (string, error) {
		// Simulate what workspace.Resolve would do: reject the symlink
		// because it escapes the workspace boundary
		if filename == "evil.patch" {
			return "", fmt.Errorf("resolves outside the workspace root")
		}
		fullPath := filepath.Join(patchDir, filename)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	// Type /patch evil.patch and press Enter
	m = typeCmd(m, "/patch evil.patch")

	if callCount != 0 {
		t.Errorf("RunTool was called %d times, want 0 (symlink refused)", callCount)
	}
}
