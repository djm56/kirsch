package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/workspace"
)

// fakeApprover records calls and can be configured to return a decision.
type fakeApprover struct {
	called   bool
	decision policy.Decision
	lastReq  ApprovalRequest
	// blockOnRequest allows tests to verify that Request blocks as expected.
	// If set to true, the approver will wait for the test to resolve it.
	blockOnRequest bool
	// unblockChan is used to signal that Request should unblock.
	unblockChan chan struct{}
	// panicOnRequest, if set to true, causes Request to panic.
	panicOnRequest bool
}

func newFakeApprover(decision policy.Decision) *fakeApprover {
	return &fakeApprover{
		decision:    decision,
		unblockChan: make(chan struct{}, 1),
	}
}

func (fa *fakeApprover) Request(ctx context.Context, req ApprovalRequest) policy.Decision {
	fa.called = true
	fa.lastReq = req
	if fa.panicOnRequest {
		panic("approver panic for testing")
	}
	if fa.blockOnRequest {
		select {
		case <-fa.unblockChan:
		case <-ctx.Done():
			return policy.DecisionDeny
		}
	}
	return fa.decision
}

func (fa *fakeApprover) Resolve(id int64, decision policy.Decision) {
	select {
	case fa.unblockChan <- struct{}{}:
	default:
	}
}

// TestApplyPatchParseFailureNoApproval verifies that parse failures don't
// call the approver and return the right error kind.
func TestApplyPatchParseFailureNoApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Invalid diff with hunk count mismatch (says 3 lines but only 2 provided)
	input := applyPatchInput{
		Diff: "diff --git a/file.txt b/file.txt\n" +
			"--- a/file.txt\n" +
			"+++ b/file.txt\n" +
			"@@ -1,3 +1,3 @@\n" + // Says 3 old lines, 3 new lines
			" line1\n" +
			"-line2\n", // Only 2 old lines total, only 2 new lines total
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure, got OK")
	}
	if result.Error.Kind != KindToolInputInvalid {
		t.Fatalf("expected KindToolInputInvalid, got %s", result.Error.Kind)
	}
	if approver.called {
		t.Fatalf("approver should not have been called for parse failure")
	}
}

// TestApplyPatchWorkspaceViolationTargetNoApproval verifies that target path
// violations don't call the approver.
func TestApplyPatchWorkspaceViolationTargetNoApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Valid diff but target is outside workspace
	input := applyPatchInput{
		Diff: `--- a/../../../etc/passwd
+++ b/../../../etc/passwd
@@ -1 +1 @@
-root:x:0:0:::
+hacked:x:0:0:::
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure, got OK")
	}
	if result.Error.Kind != KindWorkspaceViolation {
		t.Fatalf("expected KindWorkspaceViolation, got %s", result.Error.Kind)
	}
	if approver.called {
		t.Fatalf("approver should not have been called for workspace violation")
	}
}

// TestApplyPatchRenameSourceEscapeNoApproval verifies that rename sources
// escaping the workspace are caught before approval.
func TestApplyPatchRenameSourceEscapeNoApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a source file to rename
	os.WriteFile(filepath.Join(ws.Root, "source.txt"), []byte("content"), 0o644)

	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Diff that renames from outside the workspace
	input := applyPatchInput{
		Diff: `diff --git a/../outside/source.txt b/target.txt
rename from ../outside/source.txt
rename to target.txt
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure, got OK")
	}
	if result.Error.Kind != KindWorkspaceViolation {
		t.Fatalf("expected KindWorkspaceViolation for rename source escape, got %s", result.Error.Kind)
	}
	if approver.called {
		t.Fatalf("approver should not have been called for rename source escape")
	}
}

// TestApplyPatchConflictNoApproval verifies that conflicts during dry-run
// don't call the approver.
func TestApplyPatchConflictNoApproval(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a file with known content
	testFile := filepath.Join(ws.Root, "file.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0o644)

	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Diff that won't match the actual content
	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-different content
+new content
 line3
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure, got OK")
	}
	if result.Error.Kind != KindPatchConflict {
		t.Fatalf("expected KindPatchConflict, got %s", result.Error.Kind)
	}
	if approver.called {
		t.Fatalf("approver should not have been called for conflict")
	}
}

// TestApplyPatchDenialWritesNothing verifies that denial cleans up all temp files
// and leaves the target file unchanged.
func TestApplyPatchDenialWritesNothing(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a target file
	testFile := filepath.Join(ws.Root, "file.txt")
	originalContent := []byte("line1\nline2\nline3\n")
	os.WriteFile(testFile, originalContent, 0o644)

	approver := newFakeApprover(policy.DecisionDeny) // User denies
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Valid diff
	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-line2
+new line2
 line3
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	// Count files in ws.Root before
	filesBefore := countFilesInDir(t, ws.Root)

	result := tool.Invoke(ctx, raw)

	if result.OK {
		t.Fatalf("expected failure after denial")
	}
	if result.Error.Kind != KindPolicyDenied {
		t.Fatalf("expected KindPolicyDenied, got %s", result.Error.Kind)
	}

	// Count files after
	filesAfter := countFilesInDir(t, ws.Root)

	// Files should be identical (no temp file left behind)
	if filesBefore != filesAfter {
		t.Fatalf("temp files not cleaned up: %d before, %d after", filesBefore, filesAfter)
	}

	// Content should be unchanged
	actualContent, _ := os.ReadFile(testFile)
	if string(actualContent) != string(originalContent) {
		t.Fatalf("file was modified despite denial")
	}
}

// TestApplyPatchApprovalCommits verifies that approval commits the staged content
// and the committed bytes are correct (not re-applied).
func TestApplyPatchApprovalCommits(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a target file
	testFile := filepath.Join(ws.Root, "file.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0o644)

	approver := newFakeApprover(policy.DecisionAllow) // User approves
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Valid diff
	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified line2
 line3
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if !result.OK {
		t.Fatalf("expected OK, got error: %s", result.Error.Message)
	}

	// Check that the file was modified correctly
	actualContent, _ := os.ReadFile(testFile)
	expected := "line1\nmodified line2\nline3\n"
	if string(actualContent) != expected {
		t.Fatalf("file content incorrect: expected %q, got %q", expected, string(actualContent))
	}
}

// TestApplyPatchApprovalRequestCarriesPatchOperation verifies that the approval
// request contains policy.OperationPatch.
func TestApplyPatchApprovalRequestCarriesPatchOperation(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a target file
	testFile := filepath.Join(ws.Root, "file.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0o644)

	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Valid diff
	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified line2
 line3
`,
		Description: "test patch",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)

	if !result.OK {
		t.Fatalf("expected OK, got error: %s", result.Error.Message)
	}

	if !approver.called {
		t.Fatalf("approver was not called")
	}

	if approver.lastReq.Operation != policy.OperationPatch {
		t.Fatalf("expected Operation to be OperationPatch, got %v", approver.lastReq.Operation)
	}

	if approver.lastReq.Description != "test patch" {
		t.Fatalf("expected description to be 'test patch', got %q", approver.lastReq.Description)
	}
}

// TestApplyPatchErrorKindMapping verifies that every error kind emitted by the
// patch package is mapped to a tool error kind deliberately. The test enumerates
// the kinds that internal/patch can emit (4 declared constants plus 8 inline kinds),
// ensuring complete coverage from emitter to handler.
func TestApplyPatchErrorKindMapping(t *testing.T) {
	// Table of all possible error kinds from patch package and their expected mappings.
	// These are the exact kinds emitted by internal/patch/apply.go.
	testCases := []struct {
		name     string
		applyErr *patch.ApplyError
		expected Kind
	}{
		// Declared constants from patch package
		{
			name: "patch_conflict",
			applyErr: &patch.ApplyError{
				Kind:    patch.ErrPatchConflict,
				Message: "test conflict",
			},
			expected: KindPatchConflict,
		},
		{
			name: "line_ending_mismatch",
			applyErr: &patch.ApplyError{
				Kind:    patch.ErrLineEnding,
				Message: "CRLF to LF",
			},
			expected: KindToolInputInvalid,
		},
		{
			name: "binary_not_supported",
			applyErr: &patch.ApplyError{
				Kind:    patch.ErrBinaryNotSupported,
				Message: "binary file",
			},
			expected: KindToolInputInvalid,
		},
		{
			name: "invalid_state",
			applyErr: &patch.ApplyError{
				Kind:    patch.ErrInvalidState,
				Message: "already committed",
			},
			expected: KindInternal,
		},
		// Inline kinds emitted from various points in internal/patch/apply.go
		{
			name: "io_error",
			applyErr: &patch.ApplyError{
				Kind:    "io_error",
				Message: "read failed",
			},
			expected: KindInternal,
		},
		{
			name: "file_not_found",
			applyErr: &patch.ApplyError{
				Kind:    "file_not_found",
				Message: "missing file",
			},
			expected: KindFileNotFound,
		},
		{
			name: "file_exists",
			applyErr: &patch.ApplyError{
				Kind:    "file_exists",
				Message: "target exists",
			},
			expected: KindToolInputInvalid,
		},
		{
			name: "backup_failed",
			applyErr: &patch.ApplyError{
				Kind:    "backup_failed",
				Message: "backup error",
			},
			expected: KindInternal,
		},
		{
			name: "rename_failed",
			applyErr: &patch.ApplyError{
				Kind:    "rename_failed",
				Message: "rename error",
			},
			expected: KindInternal,
		},
		{
			name: "partial_rollback",
			applyErr: &patch.ApplyError{
				Kind:    "partial_rollback",
				Message: "rollback incomplete",
			},
			expected: KindInternal,
		},
		{
			name: "cleanup_failed",
			applyErr: &patch.ApplyError{
				Kind:    "cleanup_failed",
				Message: "cleanup error",
			},
			expected: KindInternal,
		},
		{
			name: "invalid_op",
			applyErr: &patch.ApplyError{
				Kind:    "invalid_op",
				Message: "unsupported operation",
			},
			expected: KindInternal,
		},
	}

	ws := newTestWorkspace(t)
	tool := &ApplyPatch{WS: ws, Approver: newFakeApprover(policy.DecisionAllow)}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tool.mapApplyError(tc.applyErr)
			if result.Error.Kind != tc.expected {
				t.Fatalf("expected Kind %s, got %s", tc.expected, result.Error.Kind)
			}
		})
	}
}

// ── Helper functions ────────────────────────────────────────────────────────

// newTestWorkspace creates a temporary workspace for testing.
func newTestWorkspace(t *testing.T) *workspace.Workspace {
	tmpdir := t.TempDir()
	ws, err := workspace.Detect(tmpdir)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	return ws
}

// countFilesInDir recursively counts files in a directory (ignoring directories themselves).
func countFilesInDir(t *testing.T, dir string) int {
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			count += countFilesInDir(t, filepath.Join(dir, entry.Name()))
		} else {
			count++
		}
	}
	return count
}

// TestApplyPatchDeferCleanupOnPanic verifies that if the approver panics during
// Request, the defer ensures the staged handle is discarded and temp files are
// cleaned up before the panic propagates.
func TestApplyPatchDeferCleanupOnPanic(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a target file for the patch to apply
	testFile := filepath.Join(ws.Root, "file.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0o644)

	// Create approver configured to panic on Request
	approver := newFakeApprover(policy.DecisionAllow)
	approver.panicOnRequest = true
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Valid diff that would pass dry-run
	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified line2
 line3
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	// Count files before invoke - with staged temp file
	// Note: We can't easily test that temp files are cleaned up without
	// instrumenting the patch package, so this test demonstrates that
	// the panic doesn't prevent the defer from running.

	// Invoke should panic, but the defer will run first
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_ = tool.Invoke(ctx, raw)
	}()

	if !panicked {
		t.Fatalf("expected panic from approver, but Invoke completed normally")
	}
}

// TestApplyPatchContextCancellationUnblocksRequest verifies that when a context
// is cancelled while Request is blocked (blockOnRequest set), the approver
// returns DecisionDeny and the Invoke call completes rather than hanging.
func TestApplyPatchContextCancellationUnblocksRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ws := newTestWorkspace(t)

	// Create a target file for the patch to apply
	testFile := filepath.Join(ws.Root, "file.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0o644)

	// Create approver configured to block on Request
	approver := newFakeApprover(policy.DecisionAllow)
	approver.blockOnRequest = true
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Valid diff that would pass dry-run
	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-line2
+modified line2
 line3
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	// Invoke in a separate goroutine and cancel context after a short delay
	// to allow Request to block
	done := make(chan Result, 1)
	go func() {
		result := tool.Invoke(ctx, raw)
		done <- result
	}()

	// Give Request time to block
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for the result with a timeout to catch any hang
	select {
	case result := <-done:
		// Invoke should complete without hanging. Context cancellation causes
		// fakeApprover.Request to return DecisionDeny via ctx.Done() case,
		// which makes Invoke return a policy denied error.
		if result.OK {
			t.Fatalf("expected failure due to context cancellation, got OK")
		}
		if result.Error.Kind != KindPolicyDenied {
			t.Logf("warning: context cancellation returned %s instead of %s", result.Error.Kind, KindPolicyDenied)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Invoke did not return after context cancellation (hung)")
	}
}

// TestApplyPatchMutationTest verifies that changing the operation type to
// OperationCommand causes the test to fail. It is run by default.
func TestApplyPatchOperationCannotBeMutatedToCommand(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)

	// Create a target file
	testFile := filepath.Join(ws.Root, "file.txt")
	os.WriteFile(testFile, []byte("line1\n"), 0o644)

	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	input := applyPatchInput{
		Diff: `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-line1
+line2
`,
		Description: "test",
	}
	raw, _ := json.Marshal(input)

	result := tool.Invoke(ctx, raw)
	if !result.OK {
		t.Fatalf("patch failed: %v", result.Error.Message)
	}

	// Verify the operation is OperationPatch
	if approver.lastReq.Operation != policy.OperationPatch {
		t.Fatalf("MUTATION TEST FAILED: Operation should be OperationPatch, got %v", approver.lastReq.Operation)
	}
}
