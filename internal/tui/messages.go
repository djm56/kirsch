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
