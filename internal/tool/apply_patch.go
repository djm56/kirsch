package tool

import (
	"context"
	"encoding/json"

	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/workspace"
)

// ApplyPatch applies a unified diff to files in the workspace.
// The order of operations is critical: parse → resolve → dry-run → approval → commit.
// No approval prompt is shown for patches that would fail or escape the workspace.
type ApplyPatch struct {
	WS       *workspace.Workspace
	Approver Approver
}

// Approver is the narrow interface for requesting patch approval.
// It is not the full app.Approver to avoid a compile cycle (app imports tool).
type Approver interface {
	// Request blocks until the user responds to the approval request.
	// It never returns before the user presses y/n/a.
	Request(ctx context.Context, req ApprovalRequest) policy.Decision

	// Resolve is called from the TUI's Update and must never block.
	// It records the decision and unblocks a pending Request.
	Resolve(id int64, decision policy.Decision)
}

// ApprovalRequest carries the information the approver needs to present to the user.
type ApprovalRequest struct {
	ID          int64
	Operation   policy.Operation
	Description string
	Changes     []patch.FileChange
	Argv        []string // Command argv for session grant; nil for patches
}

type applyPatchInput struct {
	Diff        string `json:"diff"`
	Description string `json:"description"`
}

// Name implements Tool.
func (t *ApplyPatch) Name() string { return "apply_patch" }

// Description implements Tool.
func (t *ApplyPatch) Description() string {
	return "Apply a unified diff to files in the workspace. " +
		"The diff is validated before requesting approval, and patches always require user approval."
}

// Schema implements Tool.
func (t *ApplyPatch) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "diff": {
      "type": "string",
      "description": "A unified diff (git diff format). Parsed and applied to workspace-relative paths."
    },
    "description": {
      "type": "string",
      "description": "Human-readable description of what the patch does."
    }
  },
  "required": ["diff"],
  "additionalProperties": false
}`)
}

// Invoke implements Tool.
func (t *ApplyPatch) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in applyPatchInput
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}

	// Step 1: Parse the diff.
	changes, err := patch.Parse([]byte(in.Diff))
	if err != nil {
		return t.mapParseError(err)
	}

	// Step 2: Resolve every path — both targets and rename sources.
	resolvedChanges := make([]patch.FileChange, len(changes))
	for i, change := range changes {
		resolvedChanges[i] = change

		// Resolve the target path
		abs, res, bad := resolve(t.WS, change.Path)
		if bad {
			return res
		}
		resolvedChanges[i].Path = abs

		// Resolve rename source if present
		if change.OldPath != "" {
			absOld, res, bad := resolve(t.WS, change.OldPath)
			if bad {
				return res
			}
			resolvedChanges[i].OldPath = absOld
		}
	}

	// Step 3: Dry-run the application against temp copies.
	handle, err := patch.Stage(resolvedChanges)
	if err != nil {
		return t.mapApplyError(err)
	}
	// Handle is now staged. If anything goes wrong from here, we must discard it.

	// Ensure the handle is discarded if the approval flow does not complete normally.
	// The patch.StageHandle.Discard method checks state internally: it only succeeds
	// if the handle is in StateStaged, and returns ErrInvalidState otherwise. This
	// prevents double-discard and ensures discard does not fire after a successful commit.
	defer func() {
		_ = handle.Discard()
	}()

	// Step 4: Request approval (never for a patch that would fail).
	decision := t.Approver.Request(ctx, ApprovalRequest{
		ID:          1, // TODO: This will be provided by the approval flow in M3.
		Operation:   policy.OperationPatch,
		Description: in.Description,
		Changes:     changes, // Send original changes (with relative paths) for display
	})

	// Step 5: On approval, commit; on denial, discard.
	// DecisionAllow means allow once; DecisionSession means allow and record a grant.
	// Both mean we proceed. Any other decision means denial.
	if decision != policy.DecisionAllow && decision != policy.DecisionSession {
		// User or policy denied the patch.
		// The defer will attempt to discard, but we also do it explicitly here.
		// The defer's discard will fail (handle no longer staged) but that's safe.
		_ = handle.Discard()
		return Fail(KindPolicyDenied, "patch rejected")
	}

	commitErr := handle.Commit()
	if commitErr != nil {
		return t.mapApplyError(commitErr)
	}

	return OKResult("patch applied", "applied", false)
}

// mapParseError maps parse errors to tool error kinds.
func (t *ApplyPatch) mapParseError(err error) Result {
	parseErr, ok := err.(*patch.ParseError)
	if !ok {
		return Fail(KindInternal, "unexpected error type in patch parsing: %v", err)
	}

	switch parseErr.Kind {
	case patch.ErrMissingHeader:
		return Fail(KindToolInputInvalid, "malformed diff: %s", parseErr.Message)
	case patch.ErrBadMode:
		return Fail(KindToolInputInvalid, "invalid file mode in diff: %s", parseErr.Message)
	case patch.ErrHunkCountMismatch:
		return Fail(KindToolInputInvalid, "hunk arithmetic error in diff: %s", parseErr.Message)
	default:
		return Fail(KindToolInputInvalid, "diff parse error: %s", parseErr.Message)
	}
}

// mapApplyError maps patch.ApplyError to tool.Error kinds.
// This is the key mapping from patch layer error kinds to tool layer kinds.
func (t *ApplyPatch) mapApplyError(err error) Result {
	applyErr, ok := err.(*patch.ApplyError)
	if !ok {
		// Not an ApplyError, treat as internal
		return Fail(KindInternal, "unexpected error during patch application: %v", err)
	}

	switch applyErr.Kind {
	case patch.ErrPatchConflict:
		return Fail(KindPatchConflict, "patch cannot be applied: %s", applyErr.Message)
	case patch.ErrLineEnding:
		return Fail(KindToolInputInvalid, "patch would change line endings: %s", applyErr.Message)
	case patch.ErrBinaryNotSupported:
		return Fail(KindToolInputInvalid, "binary files cannot be patched: %s", applyErr.Message)
	case patch.ErrInvalidState:
		return Fail(KindInternal, "internal error: %s", applyErr.Message)
	default:
		// Catch any remaining error kinds from patch package
		return t.mapApplyErrorKind(applyErr.Kind, applyErr.Message)
	}
}

// mapApplyErrorKind handles all distinct error kinds from internal/patch.
// The patch package emits 8 distinct inline kinds: backup_failed, cleanup_failed,
// file_exists, file_not_found, invalid_op, io_error, partial_rollback, rename_failed.
// Each is mapped deliberately to a tool error kind. Unknown kinds fall through to default.
func (t *ApplyPatch) mapApplyErrorKind(kind patch.ApplyErrorKind, message string) Result {
	// Map the error kind to a tool.Kind. Every kind from patch is listed explicitly.
	switch kind {
	// Already handled above
	case patch.ErrPatchConflict, patch.ErrLineEnding, patch.ErrBinaryNotSupported, patch.ErrInvalidState:
		return Fail(KindInternal, "error kind already handled: %s", kind)

	// IO and file errors
	case "io_error":
		return Fail(KindInternal, "I/O error during patch: %s", message)
	case "file_not_found":
		return Fail(KindFileNotFound, "file not found during patch: %s", message)
	case "file_exists":
		return Fail(KindToolInputInvalid, "target file already exists: %s", message)

	// Backup and rename errors
	case "backup_failed":
		return Fail(KindInternal, "backup failed: %s", message)
	case "rename_failed":
		return Fail(KindInternal, "file operation failed: %s", message)
	case "partial_rollback":
		return Fail(KindInternal, "partial failure during rollback: %s", message)
	case "cleanup_failed":
		return Fail(KindInternal, "cleanup failed: %s", message)

	// Operation errors
	case "invalid_op":
		return Fail(KindInternal, "unsupported operation in patch: %s", message)

	// Catch-all for unknown kinds
	default:
		return Fail(KindInternal, "unknown patch error: %s: %s", kind, message)
	}
}
