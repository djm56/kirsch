package tui

import "time"

// Typed messages the TUI receives from internal/app.
//
// They live here, in the TUI, rather than in app, because of architecture.md
// §3 rule 2: internal/tui imports no provider, tool, workspace or agent
// package. Everything it renders arrives as one of these, carrying plain
// strings and numbers rather than a tool.Result — so the boundary is enforced
// by what the types can hold, not by remembering not to import something.

// DataFlowNotice is the ADR 0008 wording shared by the onboarding screen and
// the /status command. Keeping the text in the TUI package means both renderers
// read from one source, while internal/app only supplies a boolean flag.
const DataFlowNotice = "Using opencode sends prompts and file contents to OpenCode's gateway and to whoever hosts the chosen model, not to Anthropic."

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

// OnboardingStateMsg carries the plain, provider-free facts the onboarding
// screen needs. internal/app computes these from config and provider state;
// internal/tui only stores and renders them. ui-spec §7.5, screen 12.
type OnboardingStateMsg struct {
	NoAPIKey     bool
	KeyVars      [2]string // prefixed variable, then bare variable
	NotGitRepo   bool
	UnknownModel bool
	Endpoint     string // active endpoint kind, e.g. "opencode" or "anthropic"
}

// AssistantTextDeltaMsg appends a fragment to a streaming assistant text block.
// BlockID is an opaque int64 owned by the adapter; the TUI correlates it to a
// transcript ItemID. ui-spec §3.2.
type AssistantTextDeltaMsg struct {
	BlockID int64
	Delta   string
}

// ThinkingDeltaMsg appends a fragment to a thinking card. BlockID is opaque to
// the TUI and owned by the adapter. ui-spec §3.5.
type ThinkingDeltaMsg struct {
	BlockID int64
	Delta   string
}

// UsageMsg reports token consumption for the turn. It carries only plain ints
// so the TUI can update the status bar without importing the agent package.
type UsageMsg struct {
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

// SessionUsageMsg reports cumulative token and cost totals for the session.
// It carries only plain ints, a float and strings so the status bar can show
// real session totals without importing the provider package.
//
// CostState is one of: "unknown" (no pricing figures), "flat" (subscription),
// or "per_token" (charged by the token). CostUSD is meaningful only for
// per_token, and is the sum of each call priced at its own model's rates.
//
// ModelKnown reports whether the current model is in the model table. It is a
// separate signal from CostState: a known model can have unknown pricing, and
// only an unknown model runs on the conservative budget. The zero value reads
// as unknown, so a sender that omits it asks for the notice.
type SessionUsageMsg struct {
	TokensIn   int
	TokensOut  int
	CacheRead  int
	CacheWrite int
	CostUSD    float64
	CostState  string // "unknown", "flat", "per_token"
	ModelKnown bool
}

// BudgetMsg carries the estimated token count and model context window so the
// status bar can show the budget indicator. It crosses the TUI-implementation
// boundary as plain ints, so internal/tui imports no provider package.
type BudgetMsg struct {
	EstimatedTokens int
	ContextWindow   int
}

// TurnCompleteMsg closes a turn that finished normally. It carries no fields:
// the final assistant text and tool cards are already in the transcript.
type TurnCompleteMsg struct{}

// TurnErrorMsg reports a turn-level failure. Kind is a plain string such as
// "max_turns_exceeded" or "context_overflow". ui-spec §3.6.
type TurnErrorMsg struct {
	Kind    string
	Message string
	Detail  string
}

// TurnCancelledMsg reports that the user cancelled the in-flight turn.
type TurnCancelledMsg struct{}

// CompactingMsg indicates that compaction is in progress. It sets the status
// bar verb to "compacting". Sent before the model call begins.
type CompactingMsg struct{}

// SessionCompactedMsg indicates successful compaction. It sets SessionInfo.Compacted
// to true so the header shows "(compacted)".
type SessionCompactedMsg struct{}

// CommandOutputChunkMsg appends one chunk of output to a running command tool
// card. ID is the app-side tool ID that was sent with ToolStartedMsg.
type CommandOutputChunkMsg struct {
	ID    int64
	Chunk string
}

// StatusInfoMsg carries the plain facts the /status command displays.
// internal/app computes these from config and environment; internal/tui only
// renders them. ui-spec §6, /status.
type StatusInfoMsg struct {
	Endpoint           string // active endpoint kind, e.g. "opencode" or "anthropic"
	BaseURLHost        string // hostname of the configured base_url
	KeySource          string // name of the env variable that supplied the key
	ProxyHost          string // proxy hostname, or empty if none is configured
	ContextFile        string // chosen project-context file, or empty if none
	ContextSize        int    // on-disk size of ContextFile in bytes
	ShowDataFlowNotice bool   // true when the ADR 0008 notice should be shown
	DirtyCount         int    // number of uncommitted changes in the workspace
	SessionID          string // current session identifier, or "none" if no session has started
	TokensIn           int    // cumulative input tokens for this session
	TokensOut          int    // cumulative output tokens for this session
	TokensCache        int    // cumulative cache tokens (read + write) for this session
	CompactionPercent  int    // estimated budget consumption as a percentage, or -1 if not yet computed
	ActiveGrants       int    // number of active session grants, or 0 if none
}

// DiffResultMsg delivers the outcome of the /diff command's GetDiff call, which
// runs in a tea.Cmd so that git never blocks Update.
type DiffResultMsg struct {
	Result DiffResult
	Seq    int // sequence number for /diff requests; ignored if stale (W-B)
}

// TranscriptLoadedMsg bulk-loads a transcript snapshot, for example when a
// session is resumed from disk. The TUI appends every item through the normal
// append path and, when Recovered is true, adds the recovered-session warning
// required by ui-spec §2.3.
type TranscriptLoadedMsg struct {
	Items     []Item // pre-built transcript items to append
	Recovered bool   // true when the session log was repaired on load
}

// StatusWarningMsg adds a persistent warning to the status bar. Warnings are
// used for conditions that must be visible to the user at all times (e.g.,
// "another Kirsch is running here").
type StatusWarningMsg struct {
	Text string
}

// ManualCompactOutcome is how a manual /compact ended. The zero value is not
// an outcome, so a message built without one cannot read as success.
type ManualCompactOutcome uint8

// The manual compaction outcomes.
const (
	_ ManualCompactOutcome = iota
	// ManualCompactApplied: the conversation was replaced and recorded.
	ManualCompactApplied
	// ManualCompactNothing: too few turns to compact; no model call was made.
	ManualCompactNothing
	// ManualCompactCancelled: cancelled; the conversation and log are unchanged.
	ManualCompactCancelled
	// ManualCompactBusy: refused because a turn or another compaction is live.
	ManualCompactBusy
	// ManualCompactFailed: the summary call or the swap failed; Detail says why.
	ManualCompactFailed
)

// ManualCompactMsg reports how a manual /compact ended. The app produces it
// once, when compaction has really finished; the TUI never builds one.
type ManualCompactMsg struct {
	Outcome ManualCompactOutcome
	Detail  string // failure cause, for ManualCompactFailed
}

// NewSessionDoneMsg reports the end of the NewSession callback that /new runs
// in a tea.Cmd. Err is non-nil when no new session was started; the old one
// is then still in place.
type NewSessionDoneMsg struct {
	Err error
}
