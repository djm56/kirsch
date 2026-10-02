package tui

import "time"

// Typed messages the TUI receives from internal/app.
//
// They live here, in the TUI, rather than in app, because of architecture.md
// §3 rule 2: internal/tui imports no provider, tool, workspace or agent
// package. Everything it renders arrives as one of these, carrying plain
// strings and numbers rather than a tool.Result — so the boundary is enforced
// by what the types can hold, not by remembering not to import something.

// WorkspaceInfoMsg carries repository metadata for the header.
type WorkspaceInfoMsg struct {
	Project string
	Branch  string
	Dirty   bool
	Types   []string
}

// ToolStartedMsg announces a tool invocation. The ID correlates it with the
// ToolCompletedMsg that follows.
type ToolStartedMsg struct {
	ID     int64
	Name   string
	Target string
}

// ToolCompletedMsg carries a finished tool result, already flattened to
// primitives.
type ToolCompletedMsg struct {
	ID        int64
	Name      string
	Target    string
	OK        bool
	Summary   string
	Content   string
	Truncated bool
	ErrorKind string
	ErrorMsg  string
	Elapsed   time.Duration
}

// ErrorMsg is an infrastructure failure — not a tool error the model could
// handle. architecture.md §8.
type ErrorMsg struct {
	Kind    string
	Message string
	Detail  string
}

// NoticeMsg is a one-line system notice.
type NoticeMsg struct{ Text string }

// Intent is something the user asked for, emitted by the TUI and consumed by
// internal/app. The TUI never performs the action itself.
type Intent interface{ isIntent() }

// RunToolIntent asks app to invoke a tool.
type RunToolIntent struct {
	Name  string
	Input map[string]any
}

func (RunToolIntent) isIntent() {}

// CancelIntent asks app to cancel the in-flight turn.
type CancelIntent struct{}

func (CancelIntent) isIntent() {}

// ApprovalRequestedMsg asks the user for approval on an action. The ID
// correlates it with the Resolve call that answers it.
type ApprovalRequestedMsg struct {
	ID                   int64    // Unique approval ID
	ToolID               int64    // App-side tool ID (non-zero if linked to a tool)
	Description          string   // What is being asked for approval
	Kind                 string   // "command", "patch", etc. for the renderer
	CanApproveForSession bool     // Whether [a] button should be shown
	Subject              string   // Collapsed form summary (e.g., "2 files changed" or "go test ...")
	Detail               []string // Detailed lines for expansion
	GrantScope           string   // argv prefix for session grant; empty for patches
	Argv                 []string // Full command argv for session grant; nil for patches

	// Diff data for patch approvals (empty for command approvals).
	// For a patch touching multiple files, the modal shows the first file only.
	DiffFilename string   // The path of the first file in the patch
	DiffLines    []string // Diff lines starting from "@@ " hunks, no "diff --git", "---", "+++" headers
	DiffAdded    int      // Number of added lines (lines with '+' prefix) in the first file
	DiffRemoved  int      // Number of removed lines (lines with '-' prefix) in the first file
}

// ApprovalResolvedMsg reports the actual outcome of an approval after validation.
// It carries the approval ID and the confirmed outcome (which may differ from
// what the user chose if validation failed, e.g., grant refused).
type ApprovalResolvedMsg struct {
	ID      int64           // Approval ID this resolves
	Outcome ApprovalOutcome // Actual confirmed outcome
}
