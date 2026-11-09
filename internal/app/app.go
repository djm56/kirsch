// Package app wires the pieces together and routes between them.
//
// It exists because it is the only place that knows about both sides, which
// makes it the only place adapters can live. It holds no business logic: if a
// rule about what Kirsch may do ends up here, it is in the wrong package.
// architecture.md §3.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/agent/prompt"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/provider/anthropic"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// toolCallsKey is the context key for carrying the provider-call-id → app-side
// tool ID map from RunTurn into the toolExecutor that completes agent-driven
// tool calls. architecture.md §4.
type toolCallsKey struct{}

// turnGenKey is the context key for the session generation a turn (or a manual
// tool run) started under, so the tool executor and the approval request, which
// receive only a context, send their messages fenced to that generation.
type turnGenKey struct{}

// compactionParams holds the information needed to decide whether compaction
// should run before the turn. It is computed in Submit and passed to RunTurn.
type compactionParams struct {
	estimatedTokens int    // estimated token count for the pending request
	contextWindow   int    // model's context window in tokens
	modelID         string // the model ID for cost computation
	maxOutput       int    // model's max output in tokens (for compaction's maxTokens)
}

// ApprovalRequest describes what is being asked for approval.
//
// There is deliberately no Kind field here. An earlier revision carried an
// independent Kind string next to Operation: policy enforcement read
// Operation, the TUI's rendering read Kind, and nothing reconciled the two,
// so a caller could describe a patch as a command (or the reverse) and the
// disagreement was never caught. The string the TUI needs is derived from
// Operation inside Request (see kindString) instead of accepted from the
// caller, so the two can no longer disagree — the same principle
// internal/tui applies to ApprovalCard.OffersSessionGrant: a value that
// exists only to be derived is not stored, so the invalid combination has
// nowhere to live.
type ApprovalRequest struct {
	ID          int64              // Approval ID; non-zero when mapped from a tool invocation, zero to let App allocate
	Description string             // Human-readable text for the approval card
	Operation   policy.Operation   // The operation type; drives both policy enforcement and the TUI's rendering kind
	Argv        []string           // Command argv for session grant; nil if not applicable
	Changes     []patch.FileChange // File changes for patch operations; nil for commands
}

// kindString derives the "command"/"patch" string that
// tui.ApprovalRequestedMsg.Kind expects, from the policy Operation that
// governs the same request. It exists so that string is computed, never
// supplied — see the ApprovalRequest doc comment. Anything other than
// OperationCommand renders as "patch", the more restrictive kind, matching
// Policy.CanApproveForSession, which also refuses everything but
// OperationCommand.
func kindString(op policy.Operation) string {
	if op == policy.OperationCommand {
		return "command"
	}
	return "patch"
}

// formatPatchDisplay computes the Subject and Detail lines for a patch approval.
func formatPatchDisplay(changes []patch.FileChange) (subject string, detail []string) {
	if len(changes) == 0 {
		return "0 files changed", nil
	}

	if len(changes) == 1 {
		subject = "1 file changed"
	} else {
		subject = fmt.Sprintf("%d files changed", len(changes))
	}

	// Build the file list, wrapped to multiple lines if needed.
	// For renames, show both the source (OldPath) and destination (Path).
	var files []string
	for _, c := range changes {
		if c.Op == patch.OpRename {
			files = append(files, c.OldPath+" → "+c.Path)
		} else {
			files = append(files, c.Path)
		}
	}

	// Format as "files: N changed (file1,\n       file2, ...)"
	if len(files) > 0 {
		detail = []string{
			fmt.Sprintf("files: %d changed (%s", len(files), files[0]),
		}
		for i := 1; i < len(files); i++ {
			if i == len(files)-1 {
				detail = append(detail, fmt.Sprintf("       %s)", files[i]))
			} else {
				detail = append(detail, fmt.Sprintf("       %s,", files[i]))
			}
		}
	}

	return subject, detail
}

// formatCommandDisplay computes the Subject and GrantScope for a command approval.
func formatCommandDisplay(argv []string) (subject, grantScope string) {
	if len(argv) == 0 {
		return "", ""
	}

	// Subject is the full command as a string
	subject = argv[0]
	if len(argv) > 1 {
		subject += " " + argv[1]
	}

	// GrantScope is the full argv formatted as a space-separated string,
	// matching what policy.Grant records and policy.Grants() returns.
	grantScope = strings.Join(argv, " ")

	return subject, grantScope
}

// formatCommandDetail computes detail lines for a command approval.
// It provides the full command with arguments when expanded.
func formatCommandDetail(description string, argv []string) []string {
	if len(argv) == 0 {
		return []string{}
	}

	detail := []string{description}
	if len(argv) > 0 {
		detail = append(detail, "command: "+strings.Join(argv, " "))
	}
	return detail
}

// computeDiffDisplay extracts diff data from the first file in a patch.
// For a patch touching multiple files, it returns data for the first file only.
// It returns the filename, the diff lines (starting from "@@ " hunks), and counts
// of added and removed lines.
// For renames, filename is formatted as "old → new" to match formatPatchDisplay.
func computeDiffDisplay(changes []patch.FileChange) (filename string, diffLines []string, added int, removed int) {
	if len(changes) == 0 {
		return "", nil, 0, 0
	}

	file := changes[0]
	if file.Op == patch.OpRename {
		filename = file.OldPath + " → " + file.Path
	} else {
		filename = file.Path
	}

	// Build diff lines from hunks.
	// Each hunk is formatted as:
	//   @@ -oldStart,oldLines +newStart,newLines @@ optional context
	//   <lines with prefix: space, '+', '-', or '\'>
	for _, hunk := range file.Hunks {
		// Write the hunk header line.
		hunkLine := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
		diffLines = append(diffLines, hunkLine)

		// Write each line in the hunk with its prefix.
		// Special case: backslash prefix (\ ) is followed by a space before content.
		for _, line := range hunk.Lines {
			var prefixedLine string
			switch line.Prefix {
			case '\\':
				prefixedLine = `\ ` + line.Content
			default:
				prefixedLine = string(line.Prefix) + line.Content
			}
			diffLines = append(diffLines, prefixedLine)

			// Count added and removed lines.
			switch line.Prefix {
			case '+':
				added++
			case '-':
				removed++
			}
		}
	}

	// If there are no hunks but the file was changed (binary, mode change, pure rename,
	// or zero-byte create/delete), provide a descriptive body.
	if len(diffLines) == 0 {
		switch file.Op {
		case patch.OpRename:
			// Pure rename: no hunks, just a path change
			diffLines = append(diffLines, "File renamed")
		case patch.OpCreate:
			// Zero-byte create: uses CreateMode, not NewMode
			diffLines = append(diffLines, "Empty file created")
		case patch.OpDelete:
			// Zero-byte delete: uses CreateMode, not NewMode
			diffLines = append(diffLines, "Empty file deleted")
		default:
			// Binary or mode change
			if file.IsBinary {
				diffLines = append(diffLines, "Binary file changed")
			} else if file.NewMode != 0 {
				diffLines = append(diffLines, fmt.Sprintf("Mode changed: %o", file.NewMode))
			}
		}
	}

	return filename, diffLines, added, removed
}

// ApprovalOutcome represents a user's decision on an approval.
type ApprovalOutcome uint8

const (
	ApprovalOutcomeOnce ApprovalOutcome = iota
	ApprovalOutcomeSession
	ApprovalOutcomeDeny
	ApprovalOutcomeCancelled
)

// approvalAdapter adapts App's approval methods to the tool.Approver interface.
// The tool layer expects methods named Request and Resolve with specific signatures,
// while App provides RequestToolApproval and ResolveToolApproval. The adapter bridges
// these by forwarding to the app's methods unchanged.
//
// Compile-time assertion: if this ever stops satisfying tool.Approver, the build fails.
var _ tool.Approver = (*approvalAdapter)(nil)

type approvalAdapter struct {
	app *App
}

// Request implements tool.Approver.
func (aa *approvalAdapter) Request(ctx context.Context, req tool.ApprovalRequest) policy.Decision {
	return aa.app.RequestToolApproval(ctx, req)
}

// Resolve implements tool.Approver.
func (aa *approvalAdapter) Resolve(id int64, decision policy.Decision) {
	aa.app.ResolveToolApproval(id, decision)
}

// toolExecutor adapts App's tool registry to the agent.Tools interface. It is
// responsible for sending ToolStartedMsg and ToolCompletedMsg for tool calls
// driven by the agent, keeping the TUI-implementation boundary intact.
// architecture.md §4.
type toolExecutor struct {
	app *App
}

// Describe implements agent.Tools.
func (te *toolExecutor) Describe() []agent.ToolSpec {
	tools := te.app.reg.List()
	out := make([]agent.ToolSpec, 0, len(tools))
	for _, t := range tools {
		out = append(out, agent.ToolSpec{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}
	return out
}

// Invoke implements agent.Tools. It sends ToolStartedMsg and ToolCompletedMsg
// for the call, reusing the app-side tool ID allocated by adaptTurnEvent when
// EventToolCallStart was seen, or allocating a new one when it was not.
func (te *toolExecutor) Invoke(ctx context.Context, c agent.ToolCall) agent.ToolResult {
	target := describeToolCallInput(c.Input)

	var id int64
	if m, ok := ctx.Value(toolCallsKey{}).(map[string]int64); ok {
		id = m[c.ID]
	}
	if id == 0 {
		id = te.app.nextID.Add(1)
		te.app.send(te.app.turnGen(ctx), tui.ToolStartedMsg{ID: id, Name: c.Name, Target: target})
	}

	res := te.app.reg.Invoke(ctx, c.Name, c.Input)

	// Track touched files when apply_patch succeeds.
	if res.OK && c.Name == "apply_patch" {
		te.app.recordTouchedFilesFromDiff(c.Input)
	}

	msg := tui.ToolCompletedMsg{
		ID: id, Name: c.Name, Target: target,
		OK:        res.OK,
		Summary:   res.DisplaySummary,
		Content:   res.Content,
		Truncated: res.Truncated,
	}
	if res.Error != nil {
		msg.ErrorKind = string(res.Error.Kind)
		msg.ErrorMsg = res.Error.Message
	}
	te.app.send(te.app.turnGen(ctx), msg)

	return agent.ToolResult{
		OK:        res.OK,
		Content:   modelToolContent(res),
		Truncated: res.Truncated,
		ErrorKind: msg.ErrorKind,
	}
}

// modelToolContent returns the text the model receives for a tool result. A
// failed result leads with its "kind: message" error line, followed by any
// Content the tool produced. The envelope keeps the error apart from Content,
// so without this line a failed call with no output reaches the model as an
// empty tool_result and the model cannot see why it failed.
func modelToolContent(res tool.Result) string {
	if res.OK || res.Error == nil {
		return res.Content
	}
	if res.Content == "" {
		return res.Error.Error()
	}
	return res.Error.Error() + "\n\n" + res.Content
}

// approval tracks a pending approval request.
type approval struct {
	id         int64
	decision   chan ApprovalOutcome // capacity 1; Resolve drops writes after first
	argv       []string             // Command argv for session grant; nil if not applicable
	operation  policy.Operation     // The operation type for this approval
	grantError error                // Set by Resolve if Grant fails; nil if grant not attempted or succeeded
}

// App owns the root context and the wiring between the TUI and the tools.
type App struct {
	ws  *workspace.Workspace
	cfg config.Config
	log *telemetry.Logger
	reg *tool.Registry
	pol *policy.Policy // Session grant policy

	program *tea.Program
	// sendFn overrides delivery. Tests set it; production leaves it nil and
	// goes through program.Send.
	sendFn func(any)

	rootCtx    context.Context
	rootCancel context.CancelFunc

	mu       sync.Mutex
	turnCtx  context.Context
	turnStop context.CancelFunc
	nextID   atomic.Int64
	inFlight sync.WaitGroup

	// opMu serialises the operations that start or replace session work:
	// Submit (through its inFlight.Add in RunTurn), ManualCompact (through
	// its own inFlight.Add) and NewSession (cancel, inFlight.Wait, swap).
	// Holding it across a Submit means NewSession's Wait either sees that
	// turn's Add or runs entirely before the Submit starts, so an Add can
	// never race a Wait at zero. It is never held by a turn goroutine.
	opMu sync.Mutex
	// sessionGen counts the sessions NewSession has installed (installSession
	// bumps it). The fenced senders, send and sendAsyncSequenceFenced, deliver a
	// message only if the generation they were given is still current; both go
	// through deliverIfCurrent, which holds genMu for reading across the check
	// and the delivery, and bumpGeneration takes genMu for writing to advance
	// sessionGen. The functions TestTurnPathsUseOnlyFencedSenders lists send
	// their messages only through the fenced senders, and that test pins it.
	//
	// Invariant: no goroutine that Bubble Tea's Update runs on may take genMu.
	// A reader can sit in genMu across a program.Send that waits for the loop,
	// and Go's RWMutex holds back new readers once a writer (bumpGeneration, in
	// NewSession) is waiting, so an Update that took genMu could freeze the
	// loop. sessionGen is therefore atomic and currentGen reads it without
	// genMu. Code on the Update goroutine reads the generation only through
	// currentGen: ClearGrants and Submit are TUI callbacks, and Submit calls
	// RunTurn. Outside tests, only deliverIfCurrent and bumpGeneration take
	// genMu.
	genMu      sync.RWMutex
	sessionGen atomic.Uint64
	// turnsLive counts RunTurn goroutines that have not yet returned, and
	// compacting reports a ManualCompact in progress. Both are protected by mu.
	// ManualCompact refuses while either is set; Submit refuses while
	// compacting is set, so conv has one writer at a time.
	turnsLive  int
	compacting bool

	// Approval state: tracks pending approvals.
	approveMu sync.Mutex
	approvals map[int64]*approval
	approveID atomic.Int64

	// providerBuilder constructs the live provider client from the active
	// endpoint. Production uses the Anthropic adapter; tests swap it for a fake.
	providerBuilder providerBuilder

	// modelInfo resolves the capability figures Submit budgets against.
	// Production uses provider.LookupModel. Tests replace it so a model with a
	// small context window can drive the compaction thresholds without
	// building hundreds of kilobytes of conversation.
	modelInfo func(id string) provider.ModelInfo

	// afterCompactHook, when non-nil, runs inside runCompaction after the
	// summary call has succeeded and before the pre-swap cancellation
	// re-check. It exists only so a test can cancel the turn inside that
	// window deterministically; production leaves it nil.
	afterCompactHook func()

	// sessionID is required for the opencode endpoint. main.go sets the first
	// one; NewSession rotates it (C52) so a provider that groups requests by
	// this id sees /new as a new conversation.
	sessionID string
	// version is the Kirsch version, passed to the provider for the User-Agent.
	version string

	// Session-scoped prompt state. Assembled once at session start and reused
	// for every turn; the chosen context file and size are kept for /status.
	// architecture.md §5 and kirsch-plan.md §6.2.
	//
	// promptMu protects systemPrompt, contextFile, contextSize and
	// promptLoaded. StartSession and NewSession write them under it;
	// production reads go through promptState and contextState.
	promptMu         sync.Mutex
	promptLoaded     bool
	systemPrompt     string
	contextFile      string
	contextSize      int
	loadProjectCtxFn func(candidates []string, maxBytes int, engine prompt.Engine) (string, string, int, []string, error)

	// conv holds the multi-turn conversation. It is mutated by agent.Turn on a
	// background goroutine and passed to every RunTurn call.
	conv *agent.Conversation

	// store persists the JSONL event log. It is lazily opened in Submit and
	// reused across turns so that every event for this session lands in one file.
	store *session.Store
	// layout computes session paths under the configured state root.
	layout *session.Layout
	// storeMu protects store and layout and the other fields marked below.
	// Session files are created under it (ensureStore, NewSession), and Close
	// takes it to set closed and abandon empty sessions, so a session file is
	// never created while Close is between those steps and the lock release.
	storeMu sync.Mutex

	// lock guards against concurrent instances in the same workspace. Acquired
	// at startup before session initialization. If ErrLockHeld is returned, the
	// app marks itself as running-concurrently and disables session grants.
	lock                  *session.Lock
	concurrentInstance    bool // true if another instance holds the lock
	sessionGrantsDisabled bool // true if session grants should be disabled

	// turnCount numbers turns for turn.started/completed events.
	turnCount atomic.Int64

	// Session token tracking. These accumulate across all turns in the session.
	// Protected by storeMu.
	sessionTokensIn    int64
	sessionTokensOut   int64
	sessionTokensRead  int64
	sessionTokensWrite int64
	// sessionCostUSD is the sum of each call's cost priced at that call's own
	// model, the same per-call figure usage.json accumulates. A flat or unknown
	// model adds nothing. Protected by storeMu.
	sessionCostUSD float64
	// currentSessionID tracks the ID of the current session for /status display.
	// Protected by storeMu.
	currentSessionID string
	// origin records how the live session was created (by ensureStore or
	// NewSession), so an empty one can be abandoned cleanly (see
	// abandonEmpty). Nil for a resumed session, which is never abandoned.
	// Protected by storeMu.
	origin *sessionOrigin
	// abandonMu serialises the session-file removals and index.json restores
	// (abandonEmpty, abandonEmptyLocked, discardFailedCreate), so two of them
	// cannot interleave. Lock order is storeMu, then abandonMu: Close, and
	// createSessionStore when the store open fails, take it while holding
	// storeMu, and installSession takes it before it releases storeMu and holds
	// it until the replaced session is closed and abandoned. Nothing that holds abandonMu takes storeMu. Because
	// installSession has released storeMu by then, readers of the store do not
	// wait on the old store's fsync; only a Close that runs at that moment does,
	// holding storeMu, and so do readers queued behind that Close.
	abandonMu sync.Mutex
	// afterAbandonRemoveHook, when set, is called by abandonEmpty right after
	// it removes a session file, before the index restore. Tests use it to
	// observe what Close still holds at that moment; production leaves it nil.
	// It runs with abandonMu held, and also with storeMu held when Close is the
	// caller (installSession has released storeMu by then), so it must not call
	// App methods: one that takes either lock would deadlock.
	afterAbandonRemoveHook func()
	// beforeCreateSessionHook, when set, is called by createSessionStore after
	// its closed check and before it creates anything. Tests use it to start
	// Close inside that interval; production leaves it nil. It runs with
	// storeMu held, so it must not call App methods and wait for them on the
	// same goroutine: one that takes storeMu would deadlock.
	beforeCreateSessionHook func()
	// afterCreateSessionHook, when set, is called by createSessionStore after
	// the session file is created and index.json names it, before the store
	// is returned. Tests use it to start Close at that moment; production
	// leaves it nil. It runs with storeMu held. opMu is held as well when the
	// caller is NewSession, or Submit through ensureStore; StartNewSession's
	// call to ensureStore holds no opMu. So it must not call App methods and
	// wait for them on the same goroutine: one that takes either lock would
	// deadlock.
	afterCreateSessionHook func()
	// openSessionStore, when set, replaces session.New in createSessionStore.
	// Tests use it to make the open fail after CreateSession has written
	// index.json; production leaves it nil.
	openSessionStore func(path string, opts ...session.Option) (*session.Store, error)
	// afterInstallUnlockHook, when set, is called by installSession right after
	// it releases storeMu and before it closes and abandons the replaced
	// session. It runs with abandonMu and opMu held. Tests use it to start
	// Close at that moment; production leaves it nil.
	afterInstallUnlockHook func()
	// beforeInstallHook, when set, is called by NewSession after the old turn
	// has stopped and before installSession. Tests use it to hold NewSession
	// at that point so Close provably runs first; production leaves it nil.
	beforeInstallHook func()
	// pending is the session NewSession has created but not yet installed
	// because it is still waiting for the old turn to stop. It lives on the
	// App, not in NewSession's locals, so Close can find and abandon it.
	// Protected by storeMu.
	pending *pendingSession
	// closed is set by Close. Once it is true createSessionStore (and so
	// ensureStore and NewSession) and ResumeLastSession open or create nothing,
	// and installSession installs nothing, so no store is opened that nobody
	// closes. Protected by storeMu.
	closed bool
	// lastEstimatedTokens and lastContextWindow track the most recent budget
	// estimates for compaction % display. Protected by storeMu.
	lastEstimatedTokens int
	lastContextWindow   int

	// touchedFiles tracks workspace-relative paths changed by successful apply_patch
	// calls in this process, in first-touched order, cleaned with path.Clean and
	// de-duplicated. It is in memory only: a resumed session starts empty, because
	// nothing here is read from or written to the session store. Protected by mu.
	touchedFiles []string
	// touchedFilesSeen maps each touched file to true for fast duplicate detection.
	// Protected by mu.
	touchedFilesSeen map[string]bool
	// diffTimeout bounds one /diff git run; zero means defaultDiffTimeout. Set
	// only before the App is used.
	diffTimeout time.Duration
}

// providerBuilder constructs a provider.Provider from an endpoint configuration.
type providerBuilder func(endpoint string, ep config.EndpointConfig, key, version, sessionID string, log *telemetry.Logger) (provider.Provider, error)

// New builds an App. The context hierarchy is rooted here: rootCtx outlives
// everything, a turn context is derived per turn, and cancelling a turn never
// touches the root.
func New(ws *workspace.Workspace, cfg config.Config, log *telemetry.Logger) *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{
		ws:               ws,
		cfg:              cfg,
		log:              log,
		reg:              tool.NewRegistry(),
		pol:              policy.New(cfg.Policy.AllowSessionScopedGrants, cfg.Policy.RequireApprovalForPatches, cfg.Policy.RequireApprovalForCommands),
		rootCtx:          ctx,
		rootCancel:       cancel,
		approvals:        make(map[int64]*approval),
		providerBuilder:  defaultProviderBuilder,
		modelInfo:        provider.LookupModel,
		loadProjectCtxFn: prompt.LoadProjectContext,
		conv:             &agent.Conversation{},
		touchedFilesSeen: make(map[string]bool),
	}
	a.reg.Register(&tool.ReadFile{WS: ws})
	a.reg.Register(&tool.ListFiles{WS: ws})
	a.reg.Register(&tool.SearchCode{WS: ws})
	a.reg.Register(&tool.GitStatus{WS: ws})
	a.reg.Register(&tool.GitDiff{WS: ws})
	a.reg.Register(&tool.ApplyPatch{WS: ws, Approver: &approvalAdapter{app: a}})
	a.reg.Register(&tool.RunCommand{WS: ws, Config: &a.cfg, Policy: a.pol, Approver: &approvalAdapter{app: a}, ProgressSink: nil})
	return a
}

// Registry exposes the tool registry. Used by main's start-up log (the tool
// names) and by tests; the TUI never sees it. `kirsch doctor` does not use it:
// it builds a zero-value tool.SearchCode for its backend line (internal/doctor,
// backendResult).
func (a *App) Registry() *tool.Registry { return a.reg }

// Grants returns the list of active session-scoped command grants.
// Each grant is a space-separated command prefix.
func (a *App) Grants() []string { return a.pol.Grants() }

// ClearGrants removes all active session-scoped command grants and records the
// clear in the session log so a resumed session does not restore them.
func (a *App) ClearGrants() {
	a.pol.ClearGrants()

	a.storeMu.Lock()
	store := a.store
	a.storeMu.Unlock()
	// A nil store means no session file is open, so no grant has been recorded
	// in this session and there is nothing to revoke on disk. ensureStore is
	// deliberately not called: a clear must not create a session file.
	if store != nil {
		store.RecordScopesCleared()
		// RecordScopesCleared only queues the line. Flush it now so a kill
		// before the next ticker flush cannot lose the clear, which would let
		// a resumed session restore the grants the user revoked. A failure is
		// reported to the TUI the way turn completion reports one.
		a.flushStore(a.currentGen(), store)
	}
}

// GetTouchedFiles returns the list of workspace-relative paths changed by
// successful apply_patch calls in this session. The list is ordered and
// de-duplicated.
func (a *App) GetTouchedFiles() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Return a copy to prevent external mutation.
	result := make([]string, len(a.touchedFiles))
	copy(result, a.touchedFiles)
	return result
}

// addTouchedFile records a path as touched by apply_patch. The path is
// normalised with path.Clean first, so "./a.go", "a.go" and "sub/../a.go" are
// one entry. An empty path is ignored. A path already recorded is not added
// again. The caller must hold mu.
func (a *App) addTouchedFile(p string) {
	if p == "" {
		return
	}
	p = path.Clean(p)
	if !a.touchedFilesSeen[p] {
		a.touchedFilesSeen[p] = true
		a.touchedFiles = append(a.touchedFiles, p)
	}
}

// recordTouchedFilesFromDiff parses the apply_patch input and records the
// target path of every file change, plus the source path of a rename (which no
// longer exists in the tree but was touched). Both parses repeat what the tool
// already did, so a failure here means the input shape and the tool's parse
// have diverged; it is logged and nothing is recorded.
func (a *App) recordTouchedFilesFromDiff(raw json.RawMessage) {
	var in struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		a.log.Warn("touched files: apply_patch input not decodable", "err", err.Error())
		return
	}

	changes, err := patch.Parse([]byte(in.Diff))
	if err != nil {
		a.log.Warn("touched files: apply_patch diff not parseable", "err", err.Error())
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	for _, change := range changes {
		a.addTouchedFile(change.Path)
		if change.OldPath != "" {
			a.addTouchedFile(change.OldPath)
		}
	}
}

// defaultDiffTimeout bounds the /diff git invocation when a.diffTimeout is unset.
const defaultDiffTimeout = 10 * time.Second

// GetDiff returns the unstaged working tree diff for /diff. It runs git with
// the same arguments as the git_diff tool (tool.DiffArgs), bounded by a
// timeout, and applies the tool's MaxDiffBytes cap before sanitising.
//
// A clean tree returns no Lines and no Err; the TUI turns that into a notice.
// Added and Removed count the whole git output, not the capped part, so the
// modal header reflects the real change size.
//
// The call blocks for as long as git runs; callers on the Bubble Tea Update
// goroutine must run it inside a tea.Cmd.
func (a *App) GetDiff() tui.DiffResult {
	if !a.ws.IsGit {
		return tui.DiffResult{Err: "not a Git repository"}
	}

	timeout := a.diffTimeout
	if timeout <= 0 {
		timeout = defaultDiffTimeout
	}
	ctx, cancel := context.WithTimeout(a.rootCtx, timeout)
	defer cancel()

	out, err := tool.RunGit(ctx, a.ws.Root, tool.DiffArgs(false, "")...)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return tui.DiffResult{Err: fmt.Sprintf("git diff timed out after %s", timeout)}
		}
		return tui.DiffResult{Err: fmt.Sprintf("git diff failed: %v", err)}
	}
	if out == "" {
		return tui.DiffResult{}
	}

	added, removed := countDiffLines(out)
	content, truncated := tool.Truncate(out, tool.MaxDiffBytes)
	// Sanitise for safe rendering (ui-spec §7.1); drop the final newline so the
	// modal has no blank last row.
	lines := tui.SanitizeLines(strings.TrimSuffix(content, "\n"))

	return tui.DiffResult{
		Lines:     lines,
		Added:     added,
		Removed:   removed,
		Truncated: truncated,
		CapBytes:  tool.MaxDiffBytes,
	}
}

// countDiffLines counts added and removed lines in unified git diff output. A
// file section starts at a "diff --git " line; everything up to its first "@@"
// is header (including the "--- " and "+++ " file lines) and is not counted.
// After the first "@@" every line starting "+" or "-" is content, so an added
// line whose text starts with "++" ("+++x") is counted as one addition.
func countDiffLines(out string) (added, removed int) {
	inHeader := true
	for len(out) > 0 {
		line := out
		if i := strings.IndexByte(out, '\n'); i >= 0 {
			line, out = out[:i], out[i+1:]
		} else {
			out = ""
		}
		switch {
		case strings.HasPrefix(line, "diff --git "):
			inHeader = true
		case inHeader:
			if strings.HasPrefix(line, "@@") {
				inHeader = false
			}
		case strings.HasPrefix(line, "+"):
			added++
		case strings.HasPrefix(line, "-"):
			removed++
		}
	}
	return added, removed
}

// Attach connects the Bubble Tea program. Results reach the TUI only through
// program.Send — architecture.md §3 rule 4, since TUI state may be mutated
// only inside Update.
func (a *App) Attach(p *tea.Program) { a.program = p }

// SetSessionID sets the per-invocation session identifier required by the
// opencode endpoint. Called once from cmd/kirsch/main.go before the first turn.
func (a *App) SetSessionID(id string) { a.sessionID = id }

// SetVersion sets the Kirsch version used in the provider User-Agent.
func (a *App) SetVersion(v string) { a.version = v }

// ConcurrentInstance reports whether another instance holds the workspace lock.
func (a *App) ConcurrentInstance() bool {
	return a.concurrentInstance
}

// SetConcurrentInstance sets the concurrent instance flag and disables session grants.
// This is provided for tests in packages that cannot access unexported fields and must
// not be used from production code.
func (a *App) SetConcurrentInstance(concurrent bool) {
	a.concurrentInstance = concurrent
	a.sessionGrantsDisabled = concurrent
}

// StartSession assembles the system prompt once per session and stores the
// chosen project-context file and size for /status. It is called from
// cmd/kirsch/main.go after the TUI is attached; subsequent turns reuse the
// cached prompt rather than re-reading the project context. kirsch-plan.md §6.2.
func (a *App) StartSession(getenv func(string) string) {
	a.promptMu.Lock()
	defer a.promptMu.Unlock()
	if a.promptLoaded {
		return
	}
	a.loadAndStoreSessionPrompt(getenv)
}

// promptState returns the session system prompt under promptMu.
func (a *App) promptState() string {
	a.promptMu.Lock()
	defer a.promptMu.Unlock()
	return a.systemPrompt
}

// contextState returns the loaded project-context file and its size under
// promptMu, for /status.
func (a *App) contextState() (string, int) {
	a.promptMu.Lock()
	defer a.promptMu.Unlock()
	return a.contextFile, a.contextSize
}

// loadAndStoreSessionPrompt reads the project context, assembles the system
// prompt and marks it loaded. The caller must hold promptMu.
func (a *App) loadAndStoreSessionPrompt(getenv func(string) string) {
	content, chosen, size, warnings, err := a.loadProjectCtxFn(
		a.cfg.Context.ProjectFiles,
		a.cfg.Context.MaxProjectContextBytes,
		a.ws,
	)
	for _, w := range warnings {
		a.log.Warn("project context", "warning", w)
	}
	if err != nil {
		a.log.Error("project context", "error", err)
	}

	env := prompt.Env{
		WorkspaceRoot: a.ws.Root,
		ProjectType:   projectTypeName(a.ws.Types()),
		Branch:        a.ws.Branch(),
		Dirty:         a.ws.IsDirty(),
		OS:            runtime.GOOS,
		HasRG:         a.hasRG(),
	}

	a.systemPrompt = prompt.Assemble(env, content)
	a.contextFile = chosen
	a.contextSize = size
	a.promptLoaded = true
}

// AcquireLock attempts to acquire an advisory flock on the workspace's session
// directory. If another instance holds the lock, it sets concurrentInstance to true
// (which skips resume and disables session grants), sends a status warning, and logs
// the event. Other errors are logged but do not block startup: the lock is advisory
// and a real error should not prevent the instance from running.
func (a *App) AcquireLock() error {
	root, err := a.stateRoot()
	if err != nil {
		// Log but do not block on state root errors. The lock is best-effort.
		a.log.Debug("lock state root error", "error", err)
		return nil
	}

	lock, err := session.Acquire(root, a.ws.CanonicalRoot)
	if err == nil {
		a.lock = lock
		return nil
	}

	// Check if the lock is held by another instance.
	if errors.Is(err, session.ErrLockHeld) {
		a.log.Debug("lock held by another instance")
		a.concurrentInstance = true
		a.sessionGrantsDisabled = true
		a.SendStatusWarning("another Kirsch is running here")
		return nil
	}

	// Real error: log at Warn level and send a status warning so the user knows
	// the single-instance guard is off. The lock is advisory and best-effort.
	a.log.Warn("lock unavailable — single-instance guard off", "error", err)
	a.SendStatusWarning("lock unavailable — single-instance guard off")
	return nil
}

// Close cancels everything and waits, briefly, for in-flight tools. It then
// closes the session store and abandons (see abandonEmpty) a session created
// by this process that never received an event, and a /new session still
// waiting to be installed. It releases the workspace lock after those files
// are gone. Setting closed and the abandon steps run under storeMu, which
// NewSession and ensureStore also hold while they create a session file, so no
// file they create can appear between those steps and the lock release. If
// installSession is still closing and abandoning the session that /new
// replaced, Close's abandonEmpty waits on abandonMu until that file is gone,
// so the lock release comes after that removal too.
//
// The wait is bounded on purpose. A tool goroutine finishing after the Bubble
// Tea program has stopped blocks in program.Send — the program's loop is no
// longer draining its message channel — so an unbounded Wait here deadlocks
// the whole process on quit. Shutdown paths must not be able to hang: a
// straggler goroutine is a smaller problem than a program that will not exit.
func (a *App) Close() {
	a.rootCancel()

	done := make(chan struct{})
	go func() {
		a.inFlight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(closeGrace):
	}

	a.storeMu.Lock()
	a.closed = true
	// C51: a /new session still waiting to be installed, then the live one,
	// are abandoned if they never held an event. The pending one goes first so
	// index.json is walked back through the chain in order. A session without
	// an origin (a resumed one) is never abandoned.
	if p := a.pending; p != nil {
		a.pending = nil
		_ = p.store.Close()
		a.abandonEmpty(p.layout, p.origin)
	}
	if a.store != nil {
		_ = a.store.Close()
		a.abandonEmpty(a.layout, a.origin)
		a.store = nil
		a.origin = nil
	}
	a.storeMu.Unlock()

	// The lock is released only now, after the abandon steps above. While it
	// is held no other Kirsch can start in this workspace, resume the session
	// being deleted and then lose its events to an unlinked file.
	if a.lock != nil {
		_ = a.lock.Release()
		a.lock = nil
	}

	_ = a.log.Close()
}

// closeGrace bounds how long Close waits for in-flight tools.
const closeGrace = 2 * time.Second

// stateRoot returns the session layout root derived from cfg.Session.StorageDir.
//
// The default StorageDir is ~/.local/share/kirsch/sessions, but Layout expects
// the parent of the sessions directory, so this function strips a trailing
// "sessions" basename. A leading "~" is expanded with os.UserHomeDir so tests
// that pass config.Defaults() unexpanded still get a usable path.
func (a *App) stateRoot() (string, error) {
	dir := a.cfg.Session.StorageDir
	if dir == "" {
		return session.DefaultStateRoot()
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if dir == "~" || strings.HasPrefix(dir, "~"+string(os.PathSeparator)) {
		dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve session storage dir: %w", err)
	}
	abs = filepath.Clean(abs)

	if filepath.Base(abs) == "sessions" {
		return filepath.Dir(abs), nil
	}
	return abs, nil
}

// ensureStore lazily creates the session store and layout for this App.
//
// The store is reused across turns and closed in App.Close. Errors opening or
// creating the session file are returned to the caller so Submit can surface
// them as a turn error.
func (a *App) ensureStore() (*session.Store, *session.Layout, error) {
	a.storeMu.Lock()
	defer a.storeMu.Unlock()

	if a.store != nil {
		return a.store, a.layout, nil
	}

	// createSessionStore returns errAppClosing once Close has run.
	store, layout, sessionID, origin, err := a.createSessionStore()
	if err != nil {
		return nil, nil, err
	}

	a.store = store
	a.layout = layout
	a.origin = origin
	a.currentSessionID = sessionID
	// Reset token totals for the new session
	a.sessionTokensIn = 0
	a.sessionTokensOut = 0
	a.sessionTokensRead = 0
	a.sessionTokensWrite = 0
	a.sessionCostUSD = 0
	return store, layout, nil
}

// StartNewSession creates a fresh session store for the current workspace.
// It is safe to call before the first turn; subsequent calls are no-ops.
func (a *App) StartNewSession() error {
	_, _, err := a.ensureStore()
	return err
}

// sessionOrigin records what index.json held for the workspace just before a
// session was created, so abandoning that session can put it back.
type sessionOrigin struct {
	// id and path name the session file the origin belongs to.
	id, path string
	// prevID and prevAt are the workspace's index entry before the session was
	// created; hadPrev is false when the workspace had none.
	prevID  string
	prevAt  time.Time
	hadPrev bool
	// supersedes is the origin of the empty session this one replaced via
	// /new, or nil. It lets abandonEmpty walk past a superseded session whose
	// file has been removed to the entry before it.
	supersedes *sessionOrigin
}

// pendingSession is a session store NewSession has created and is holding
// while it waits for the old turn to stop.
type pendingSession struct {
	store  *session.Store
	layout *session.Layout
	origin *sessionOrigin
}

// errAppClosing is returned by NewSession, ensureStore (and so StartNewSession)
// and ResumeLastSession once Close has run, and by NewSession when Close runs
// while its new session waits to be installed.
var errAppClosing = errors.New("app is closing")

// randomSessionID returns the provider session id: 16 random bytes as 32 hex
// characters, the same format cmd/kirsch/main.go's newSessionID produces. If
// the random source fails it falls back to the pid, as that function does.
func randomSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// abandonEmpty removes a session file that never received an event and, when
// index.json still points at it, puts the workspace's index entry back to what
// it was before the session was created (see restorePoint). When the
// workspace had no entry before, the entry is deleted. The store must already
// be closed so its buffer is flushed: an empty file then really means no event
// was written. A nil origin (a resumed session), a missing file or a non-empty
// file is left alone. Errors are ignored: this is best-effort tidying on a path
// (quit, /new) that must not fail or hang.
func (a *App) abandonEmpty(layout *session.Layout, origin *sessionOrigin) {
	a.abandonMu.Lock()
	defer a.abandonMu.Unlock()
	a.abandonEmptyLocked(layout, origin)
}

// abandonEmptyLocked is abandonEmpty for a caller that already holds
// abandonMu (installSession, which takes it before releasing storeMu).
func (a *App) abandonEmptyLocked(layout *session.Layout, origin *sessionOrigin) {
	if origin == nil || layout == nil {
		return
	}
	info, err := os.Stat(origin.path)
	if err != nil || info.Size() != 0 {
		return
	}
	if err := os.Remove(origin.path); err != nil {
		return
	}
	if h := a.afterAbandonRemoveHook; h != nil {
		h()
	}
	restoreIndexLocked(layout, origin)
}

// restoreIndexLocked puts the workspace's index.json entry back to what it was
// before origin's session was created (see restorePoint), or deletes it when
// there is nothing valid to restore. It does nothing when the entry no longer
// names origin's session. Errors are ignored, as in abandonEmpty. The caller
// holds abandonMu.
func restoreIndexLocked(layout *session.Layout, origin *sessionOrigin) {
	idx, err := session.ReadIndex(layout.Root)
	if err != nil {
		return
	}
	entry, ok := idx.Sessions[layout.Workspace]
	if !ok || entry.LastSessionID != origin.id {
		return // index already moved on (a later session replaced this one)
	}
	prevID, prevAt, hadPrev := restorePoint(layout, origin)
	if hadPrev {
		idx.Set(layout.Workspace, prevID, prevAt)
	} else {
		delete(idx.Sessions, layout.Workspace)
	}
	_ = session.WriteIndex(layout.Root, idx)
}

// restorePoint returns the index entry to put back when origin's session is
// abandoned: origin's own previous entry, unless that entry names an empty
// session that /new replaced and abandonEmpty has since removed, in which case
// the entry before that one, and so on. If the entry that results names a
// session file that no longer exists (another instance abandoned it), there is
// nothing valid to restore and had is false, so the caller deletes the entry
// rather than leave index.json pointing at a missing file. The caller holds
// abandonMu, so a removal is not half done while this looks at the files.
func restorePoint(layout *session.Layout, origin *sessionOrigin) (id string, at time.Time, had bool) {
	id, at, had = origin.prevID, origin.prevAt, origin.hadPrev
	for s := origin.supersedes; s != nil && had && id == s.id; s = s.supersedes {
		if _, err := os.Stat(s.path); !errors.Is(err, fs.ErrNotExist) {
			break // still on disk, so the entry is a valid restore point
		}
		id, at, had = s.prevID, s.prevAt, s.hadPrev
	}
	if had {
		target := session.SessionPath(layout.Root, layout.Workspace, id)
		if _, err := os.Stat(target); errors.Is(err, fs.ErrNotExist) {
			return "", time.Time{}, false
		}
	}
	return id, at, had
}

// createSessionStore creates a session file under the state root and opens a
// store on it. It touches no App state, so a failure leaves the App as it
// was. The returned origin records the workspace's previous index entry for
// abandonEmpty. Both callers, ensureStore and NewSession, hold storeMu while
// it runs.
//
// It returns errAppClosing, creating nothing, once Close has run. The check
// sits here, in the same storeMu hold as the creation, so that both callers
// get it and Close (which sets closed under storeMu) cannot run between the
// check and the creation and release the workspace lock first.
//
// If opening the store fails after CreateSession has pointed index.json at the
// new id, the half-made session is discarded (see discardFailedCreate), so the
// next start does not resume a session that was never opened.
func (a *App) createSessionStore() (*session.Store, *session.Layout, string, *sessionOrigin, error) {
	// Close has already run: a store created now would never be closed.
	if a.closed {
		return nil, nil, "", nil, errAppClosing
	}
	if h := a.beforeCreateSessionHook; h != nil {
		h()
	}

	root, err := a.stateRoot()
	if err != nil {
		return nil, nil, "", nil, err
	}

	layout, err := session.NewLayout(root, a.ws.CanonicalRoot)
	if err != nil {
		return nil, nil, "", nil, fmt.Errorf("create session layout: %w", err)
	}

	// Read the entry CreateSession is about to overwrite. A read failure just
	// means a later abandon cannot restore it.
	prevID, prevAt, hadPrev, _ := layout.LastSession()

	path, sessionID, err := layout.CreateSession(time.Now().UTC())
	if err != nil {
		return nil, nil, "", nil, fmt.Errorf("create session file: %w", err)
	}

	origin := &sessionOrigin{id: sessionID, path: path, prevID: prevID, prevAt: prevAt, hadPrev: hadPrev}
	open := session.New
	if a.openSessionStore != nil {
		open = a.openSessionStore
	}
	store, err := open(
		path,
		session.WithOnError(func(err error) {
			a.log.Debug("session store error", "error", err)
		}),
	)
	if err != nil {
		a.discardFailedCreate(layout, origin)
		return nil, nil, "", nil, fmt.Errorf("open session store: %w", err)
	}
	if h := a.afterCreateSessionHook; h != nil {
		h()
	}
	return store, layout, sessionID, origin, nil
}

// discardFailedCreate undoes a CreateSession whose store could not be opened:
// it removes the session file if it exists and is empty, then puts index.json
// back as restoreIndexLocked does. The index is restored even when there is no
// file, because CreateSession writes index.json before any file exists and an
// entry naming a missing file makes the next start fail. A non-empty file is
// left on disk. The caller holds storeMu; this takes abandonMu (storeMu, then
// abandonMu, as everywhere).
func (a *App) discardFailedCreate(layout *session.Layout, origin *sessionOrigin) {
	a.abandonMu.Lock()
	defer a.abandonMu.Unlock()
	if info, err := os.Stat(origin.path); err == nil && info.Size() == 0 {
		_ = os.Remove(origin.path)
	}
	restoreIndexLocked(layout, origin)
}

// NewSession abandons the current session and starts a fresh one. It blocks
// until the in-flight turn has stopped, so the TUI calls it from a tea.Cmd and
// never from Update.
//
// The new session store is created first. If that fails, NewSession returns
// the error and the App is unchanged: the running turn is not cancelled and
// the conversation, store, totals, grants and prompt are as they were. Once
// the store exists, NewSession:
//   - cancels the in-flight turn and waits for its goroutines to finish;
//   - installs the new store, session ID and an empty conversation, zeroes the
//     token totals and budget estimates, and closes the old store (see
//     installSession);
//   - clears touched files and session grants and resets the turn counter;
//   - rotates the provider session id to a fresh one;
//   - re-reads the project context and rebuilds the system prompt.
//
// The new session file and its index.json entry are written up front. If the
// process quits before the session receives an event (during the wait, or
// after /new with nothing sent), Close removes the empty file and puts
// index.json back to the entry it replaced (abandonEmpty). An empty session
// that /new replaces is removed the same way. A resumed session is never
// removed. If Close has already run, NewSession returns errAppClosing (from
// createSessionStore) before creating anything. If Close runs while NewSession
// waits for the old turn, it abandons the parked session and NewSession returns
// errAppClosing, installing nothing.
//
// The advisory lock is not released; this process keeps holding it.
// NewSession holds opMu throughout, so Submit and ManualCompact cannot start
// work between the Wait and the swap. It holds storeMu from the closed check
// (in createSessionStore) until the new session is parked; Close takes storeMu
// too, so it cannot run in between and release the lock while the new file
// exists unparked.
func (a *App) NewSession(getenv func(string) string) error {
	a.opMu.Lock()
	defer a.opMu.Unlock()

	// The closed check and the file creation (both in createSessionStore) and
	// the parking below run under one storeMu hold. Close sets closed, abandons
	// pending and live sessions under
	// storeMu, and releases the workspace lock only after that. So Close either
	// runs that section wholly before this one (closed is seen, no file is
	// made) or wholly after it (Close finds the new session, parked or by then
	// installed, and abandons it if it is still empty, while it still holds
	// the lock). So an empty file this creates is never on disk once Close has
	// released the lock. storeMu is held only for the file I/O here, as
	// ensureStore holds it, never across the turn wait below.
	a.storeMu.Lock()
	store, layout, sessionID, origin, err := a.createSessionStore()
	if err != nil {
		// errAppClosing (Close has run) or a create failure; either way
		// nothing was parked and storeMu must not stay held.
		a.storeMu.Unlock()
		return err
	}
	// Park the new session where Close can see it: if the process quits while
	// we wait below, Close abandons it instead of leaving an empty file that
	// index.json points at (C51).
	a.pending = &pendingSession{store: store, layout: layout, origin: origin}
	a.storeMu.Unlock()

	// Cancel any in-flight turn and wait for its goroutines to finish. The
	// turn goroutines hold the old store; after Wait none of them can write.
	a.CancelTurn()
	a.inFlight.Wait()
	if h := a.beforeInstallHook; h != nil {
		h()
	}

	if err := a.installSession(store, layout, sessionID, origin); err != nil {
		return err
	}

	// C52: a fresh provider session id, so the provider sees a new conversation.
	// No race with a turn reading it: opMu is held and the old turn has finished.
	a.sessionID = randomSessionID()

	a.promptMu.Lock()
	a.loadAndStoreSessionPrompt(getenv)
	a.promptMu.Unlock()
	return nil
}

// installSession makes store the live session and resets every piece of
// session-scoped state in one place: conversation, token totals, budget
// estimates, turn counter, touched files and grants. The previous store is
// closed and abandoned after storeMu is released, so the fsync and file I/O do
// not stall readers of the store. abandonMu is taken before storeMu is
// released and held until that is done: a Close that gets storeMu in between
// waits in abandonEmpty for the replaced file to be gone before it abandons
// the new session, and so releases the workspace lock only after both. The
// caller must have waited for in-flight work to finish.
//
// It returns errAppClosing, installing nothing, if Close has already run;
// Close has then dealt with the pending store itself. The old session is
// abandoned (see abandonEmpty) if it was created by this process and never
// received an event; a resumed session has no origin and is left alone.
func (a *App) installSession(store *session.Store, layout *session.Layout, sessionID string, origin *sessionOrigin) error {
	a.storeMu.Lock()
	if a.closed {
		a.storeMu.Unlock()
		return errAppClosing
	}
	a.pending = nil
	old, oldOrigin, oldLayout := a.store, a.origin, a.layout
	// Record the replaced session on the new origin before anyone else can
	// see it as live, so restorePoint can step over it if it is removed.
	origin.supersedes = oldOrigin
	a.store = store
	a.layout = layout
	a.origin = origin
	a.currentSessionID = sessionID
	a.conv = &agent.Conversation{}
	a.sessionTokensIn = 0
	a.sessionTokensOut = 0
	a.sessionTokensRead = 0
	a.sessionTokensWrite = 0
	a.sessionCostUSD = 0
	a.lastEstimatedTokens = 0
	a.lastContextWindow = 0
	// Lock order storeMu, then abandonMu. Taking abandonMu before letting go of
	// storeMu leaves no moment where Close can take storeMu and finish its
	// abandon steps while the replaced session's file is still on disk.
	a.abandonMu.Lock()
	a.storeMu.Unlock()
	if h := a.afterInstallUnlockHook; h != nil {
		h()
	}
	if old != nil {
		_ = old.Close()
		a.abandonEmptyLocked(oldLayout, oldOrigin)
	}
	a.abandonMu.Unlock()

	a.turnCount.Store(0)

	a.mu.Lock()
	a.touchedFiles = nil
	a.touchedFilesSeen = make(map[string]bool)
	a.mu.Unlock()

	a.pol.ClearGrants()

	// Last, so a late message from the old session is fenced out from here on.
	// bumpGeneration also waits for any delivery of the old generation that is
	// already handing a message to the TUI.
	a.bumpGeneration()
	return nil
}

// ResumeLastSession loads the workspace's last session from index.json,
// reconstructs the conversation and transcript, reopens the session store for
// append, restores session-scoped grants, and sends the recovered transcript
// into the TUI.
//
// It also seeds the session totals from the log's usage events
// (session.LoadUsageTotals), prices the summed tokens at the current model, and
// sends a SessionUsageMsg after the transcript. On a load error the totals stay
// zero and the zero-total message is still sent.
//
// If the workspace has no recorded session, it returns session.ErrNoSession so
// the caller can fall back to a fresh session. After Close it returns
// errAppClosing and opens nothing.
func (a *App) ResumeLastSession() error {
	a.storeMu.Lock()
	defer a.storeMu.Unlock()

	if a.store != nil {
		return nil
	}

	// Close has already run: a store opened now would never be closed.
	if a.closed {
		return errAppClosing
	}

	root, err := a.stateRoot()
	if err != nil {
		return err
	}

	layout, err := session.NewLayout(root, a.ws.CanonicalRoot)
	if err != nil {
		return fmt.Errorf("create session layout: %w", err)
	}

	id, _, ok, err := layout.LastSession()
	if err != nil {
		return fmt.Errorf("read session index: %w", err)
	}
	if !ok {
		return session.ErrNoSession
	}

	path := session.SessionPath(layout.Root, layout.Workspace, id)
	conv, transcript, err := session.Resume(path)
	if err != nil {
		return fmt.Errorf("resume session: %w", err)
	}

	store, err := session.New(
		path,
		session.WithOnError(func(err error) {
			a.log.Debug("session store error", "error", err)
		}),
	)
	if err != nil {
		return fmt.Errorf("open session store: %w", err)
	}

	if err := layout.RecordSession(id, time.Now().UTC()); err != nil {
		a.log.Debug("failed to update session index", "error", err)
	}

	a.restoreGrants(path)

	a.conv = conv
	a.store = store
	a.layout = layout
	a.currentSessionID = id
	// Seed the totals from this session's own log. usage.json holds the totals of every
	// session that uses the same data root, so it is not read here. A load error leaves the totals at zero.
	a.sessionTokensIn = 0
	a.sessionTokensOut = 0
	a.sessionTokensRead = 0
	a.sessionTokensWrite = 0
	a.sessionCostUSD = 0
	modelID := a.cfg.Provider.Endpoints[a.cfg.Provider.Default].Model
	if totals, err := session.LoadUsageTotals(path); err != nil {
		a.log.Debug("failed to load session usage", "error", err)
	} else {
		a.sessionTokensIn = totals.InputTokens
		a.sessionTokensOut = totals.OutputTokens
		a.sessionTokensRead = totals.CacheReadTokens
		a.sessionTokensWrite = totals.CacheWriteTokens
		// The log's usage events carry no model id, so the summed tokens are
		// priced once at the current model's rates, the same Cost call
		// recordUsage makes per call. A model without per-token pricing adds
		// nothing, as in recordUsage.
		if cost, ok := provider.LookupModel(modelID).Cost(
			totals.InputTokens, totals.OutputTokens,
			totals.CacheReadTokens, totals.CacheWriteTokens,
		); ok {
			a.sessionCostUSD = cost
		}
	}

	items := a.transcriptToItems(transcript)
	a.sendAsyncSequence([]tea.Msg{
		tui.NoticeMsg{Text: "session resumed"},
		tui.TranscriptLoadedMsg{Items: items, Recovered: store.Recovered()},
		sessionUsageMsgFor(
			a.sessionTokensIn, a.sessionTokensOut,
			a.sessionTokensRead, a.sessionTokensWrite,
			a.sessionCostUSD, modelID,
		),
	})
	return nil
}

// restoreGrants replays approval.scope_granted events from the session log
// through policy.Grant. Invalid or refused grants are logged and skipped.
// If session grants are disabled (e.g., because another instance is running),
// this method returns early without restoring any grants.
func (a *App) restoreGrants(path string) {
	if a.sessionGrantsDisabled {
		return
	}
	grants, err := session.LoadGrants(path)
	if err != nil {
		a.log.Debug("failed to load session grants", "error", err)
		return
	}
	for _, g := range grants {
		op, argv, ok := parseGrantAction(g.Action, g.Pattern)
		if !ok {
			a.log.Debug("skipping unknown grant action", "action", g.Action)
			continue
		}
		if err := a.pol.Grant(op, argv); err != nil {
			a.log.Debug("restored grant rejected by policy", "action", g.Action, "pattern", g.Pattern, "error", err)
		}
	}
}

// parseGrantAction converts the action/pattern stored in a session log back
// into a policy operation and argv. Only run_command grants are restorable;
// apply_patch is refused by policy.Grant, so it is skipped here.
func parseGrantAction(action, pattern string) (policy.Operation, []string, bool) {
	switch action {
	case "run_command":
		return policy.OperationCommand, strings.Fields(pattern), true
	case "apply_patch":
		// OperationPatch is never grantable, so there is nothing to restore.
		return 0, nil, false
	default:
		return 0, nil, false
	}
}

// transcriptToItems maps the session package's plain transcript to TUI items.
// Tool results are correlated to their calls by ID so completed cards render
// with their output.
func (a *App) transcriptToItems(trans session.Transcript) []tui.Item {
	results := make(map[string]agent.ResultBlock)
	for _, tm := range trans.Messages {
		if tm.Role != agent.RoleUser {
			continue
		}
		for _, b := range tm.Content {
			if b.Kind == agent.BlockToolResult && b.Result != nil {
				results[b.Result.CallID] = *b.Result
			}
		}
	}

	var items []tui.Item
	for _, tm := range trans.Messages {
		switch tm.Role {
		case agent.RoleUser:
			for _, b := range tm.Content {
				if b.Kind == agent.BlockText && b.Text != "" {
					items = append(items, tui.Item{
						Kind: tui.KindUser,
						Text: &tui.TextBlock{Lines: tui.SanitizeLines(b.Text)},
					})
				}
				// Tool results are rendered as part of the assistant's tool-call card.
			}

		case agent.RoleAssistant:
			for _, b := range tm.Content {
				switch b.Kind {
				case agent.BlockText:
					if b.Text != "" {
						items = append(items, tui.Item{
							Kind: tui.KindAssistant,
							Text: &tui.TextBlock{Lines: tui.SanitizeLines(b.Text)},
						})
					}
				case agent.BlockThinking:
					if b.Thinking != nil {
						text := b.Thinking.Text
						items = append(items, tui.Item{
							Kind: tui.KindThinking,
							Thinking: &tui.ThinkingCard{
								Tokens: len(strings.Fields(text)),
								Body:   tui.SanitizeLines(text),
							},
						})
					}
				case agent.BlockToolCall:
					if b.ToolCall != nil {
						tc := b.ToolCall
						state := tui.StateRunning
						out := ""
						if res, ok := results[tc.ID]; ok {
							state = tui.StateOK
							if res.IsError {
								state = tui.StateError
							}
							out = res.Content
						}
						items = append(items, tui.Item{
							Kind: tui.KindTool,
							Tool: &tui.ToolCard{
								Name:   tc.Name,
								Target: describeToolCallInput(tc.Input),
								State:  state,
								Out:    tui.SanitizeLines(out),
							},
						})
					}
				case agent.BlockToolResult:
					// Tool results are rendered as part of the assistant's tool-call card.
				}
			}
		}
	}
	return items
}

// ActiveModelInfo returns the model name and context window for the active
// endpoint as plain TUI status fields. It keeps provider.LookupModel on the
// app side of the boundary so internal/tui never imports internal/provider.
func (a *App) ActiveModelInfo() tui.Status {
	epName := a.cfg.Provider.Default
	ep, ok := a.cfg.Provider.Endpoints[epName]
	if !ok {
		return tui.Status{}
	}
	info := provider.LookupModel(ep.Model)
	return tui.Status{Model: info.ID, ContextWindow: info.ContextWindow}
}

// StatusInfo returns the plain facts the /status command needs. It never
// exposes the API key value, only the name of the variable that supplied it,
// and it reports only the proxy hostname, stripping any userinfo. ui-spec §6.
func (a *App) StatusInfo(getenv func(string) string) tui.StatusInfoMsg {
	epName := a.cfg.Provider.Default
	ep, ok := a.cfg.Provider.Endpoints[epName]
	if !ok {
		return tui.StatusInfoMsg{}
	}

	_, keySource := ep.ResolveKey(getenv)

	// Get the dirty count from the workspace
	dirtyCount := 0
	if a.ws != nil {
		count, err := a.ws.DirtyCount()
		if err != nil {
			dirtyCount = -1
		} else {
			dirtyCount = count
		}
	}

	// Get grants count
	grants := a.Grants()
	grantsCount := len(grants)

	// Get session ID, token totals, and compaction percent under lock
	a.storeMu.Lock()
	sessionID := a.currentSessionID
	if sessionID == "" {
		sessionID = "none"
	}
	tokensIn := int(a.sessionTokensIn)
	tokensOut := int(a.sessionTokensOut)
	tokensCache := int(a.sessionTokensRead + a.sessionTokensWrite)
	compactionPct := -1
	// A window no larger than outputReserve has no budget to measure against;
	// compactPercent would report 0%, which reads as "nothing used".
	if a.lastEstimatedTokens >= 0 && a.lastContextWindow-outputReserve > 0 {
		compactionPct = compactPercent(a.lastEstimatedTokens, a.lastContextWindow)
	}
	a.storeMu.Unlock()
	contextFile, contextSize := a.contextState()

	return tui.StatusInfoMsg{
		Endpoint:           epName,
		BaseURLHost:        hostOnly(ep.BaseURL),
		KeySource:          keySource,
		ProxyHost:          proxyHost(getenv),
		ContextFile:        contextFile,
		ContextSize:        contextSize,
		ShowDataFlowNotice: epName == "opencode",
		DirtyCount:         dirtyCount,
		SessionID:          sessionID,
		TokensIn:           tokensIn,
		TokensOut:          tokensOut,
		TokensCache:        tokensCache,
		CompactionPercent:  compactionPct,
		ActiveGrants:       grantsCount,
	}
}

// hostOnly returns the hostname from a URL string, or the original string if it
// cannot be parsed. The configured base_url is already validated, so the parse
// should not fail in practice.
func hostOnly(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Hostname()
}

// proxyHost returns the hostname of the active proxy, checking HTTPS_PROXY and
// then HTTP_PROXY. It deliberately discards scheme, port, userinfo, and path so
// that /status never leaks credentials. ADR 0008, /status box :313.
func proxyHost(getenv func(string) string) string {
	for _, key := range []string{"HTTPS_PROXY", "HTTP_PROXY"} {
		if raw := getenv(key); raw != "" {
			return hostOnly(raw)
		}
	}
	return ""
}

// CheckOnboarding inspects the active endpoint and sends an OnboardingStateMsg
// to the TUI when the user needs to see something before starting: a missing
// API key, an unknown model, or (for opencode) the ADR 0008 data-flow notice.
// It is safe to call from outside the Bubble Tea event loop.
func (a *App) CheckOnboarding(getenv func(string) string) {
	epName := a.cfg.Provider.Default
	ep, ok := a.cfg.Provider.Endpoints[epName]
	if !ok {
		return
	}

	key, _ := ep.ResolveKey(getenv)
	info := provider.LookupModel(ep.Model)
	if !info.Known {
		a.log.Warn("unknown model, using conservative fallback", "model", ep.Model, "endpoint", epName)
	}

	msg := tui.OnboardingStateMsg{
		NoAPIKey:     key == "",
		KeyVars:      [2]string{"KIRSCH_" + ep.APIKeyEnv, ep.APIKeyEnv},
		UnknownModel: !info.Known,
		Endpoint:     epName,
	}

	// Nothing to say for a fully configured non-opencode endpoint.
	if !msg.NoAPIKey && !msg.UnknownModel && msg.Endpoint != "opencode" {
		return
	}

	a.sendAsyncSequence([]tea.Msg{msg})
}

// Submit starts a real model-driven turn from the user's composer text.
//
// It resolves the active endpoint, builds a provider client, adapts it to
// agent.Model, assembles the system prompt, and hands everything to RunTurn.
// The conversation is preserved across calls so later turns retain context.
func (a *App) Submit(userText string, getenv func(string) string) {
	// opMu is held until RunTurn has registered its goroutine with inFlight,
	// so NewSession cannot Wait between the two. TryLock keeps a caller on the
	// Bubble Tea goroutine from freezing behind a NewSession that is waiting
	// for a turn to stop.
	//
	// The refusal below is sent without opMu, fenced to the generation read
	// after TryLock failed. If the holder is NewSession and has not bumped the
	// generation yet, the refusal is dropped if its delivery starts after the
	// bump; a delivery that starts before it is delivered, and the bump waits
	// for it. If the bump has already happened (NewSession holds opMu until it
	// returns), the refusal reads the new generation and is delivered into the
	// new session.
	if !a.opMu.TryLock() {
		a.sendAsyncSequenceFenced(a.currentGen(), []tea.Msg{tui.TurnErrorMsg{
			Kind:    "busy",
			Message: "A session operation is in progress. Try again in a moment.",
		}})
		return
	}
	defer a.opMu.Unlock()
	// Every message Submit sends from here on is fenced to this generation.
	// NewSession holds opMu across its bump, so it cannot change under us.
	gen := a.currentGen()

	// A manual compaction owns conv until it returns; a turn started now would
	// write the same Conversation.
	if a.isCompacting() {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
			Kind:    "busy",
			Message: "Compaction is running. Wait for it to finish, or press Esc to cancel it.",
		}})
		return
	}

	epName := a.cfg.Provider.Default
	ep, ok := a.cfg.Provider.Endpoints[epName]
	if !ok {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
			Kind:    "config",
			Message: fmt.Sprintf("unknown endpoint %q", epName),
		}})
		return
	}

	key, _ := ep.ResolveKey(getenv)
	if key == "" {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
			Kind:    "auth",
			Message: fmt.Sprintf("no API key for endpoint %q (set %s or %s)", epName, "KIRSCH_"+ep.APIKeyEnv, ep.APIKeyEnv),
		}})
		return
	}

	// Ensure the session prompt is assembled exactly once. In production this is
	// already done in startupPostAttach; the guard lets tests call Submit without
	// first calling StartSession while still reading the context only once.
	a.StartSession(getenv)

	store, _, err := a.ensureStore()
	if err != nil {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
			Kind:    "session_store",
			Message: fmt.Sprintf("failed to open session store: %v", err),
		}})
		return
	}

	prov, err := a.providerBuilder(epName, ep, key, a.version, a.sessionID, a.log)
	if err != nil {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
			Kind:    "provider",
			Message: fmt.Sprintf("failed to build provider client: %v", err),
		}})
		return
	}

	model := &providerModel{
		p:             prov,
		model:         ep.Model,
		promptCaching: ep.PromptCaching,
		thinking:      parseThinkingLevel(ep.Thinking),
	}

	info := a.modelInfo(ep.Model)

	ag := &agent.Agent{
		Model:     model,
		MaxTokens: info.MaxOutput,
		System:    a.promptState(),
		Recorder:  store,
	}

	// The tool specs are part of the request size, so attach the executor before
	// estimating the budget. RunTurn will set the same field again.
	ag.Tools = &toolExecutor{app: a}

	// There is deliberately no overflow gate here. A request over budget may
	// still fit once compacted, so RunTurn owns the overflow decision: it
	// compacts first and refuses only when compaction cannot help.
	estimated := a.estimateRequest(userText, a.conv, ag.Tools)

	// Store the estimated and context window for /status display
	a.storeMu.Lock()
	a.lastEstimatedTokens = estimated
	a.lastContextWindow = info.ContextWindow
	a.storeMu.Unlock()

	a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.BudgetMsg{
		EstimatedTokens: estimated,
		ContextWindow:   info.ContextWindow,
	}})

	compParams := compactionParams{
		estimatedTokens: estimated,
		contextWindow:   info.ContextWindow,
		maxOutput:       info.MaxOutput,
		modelID:         ep.Model,
	}
	a.RunTurn(ag, a.conv, userText, compParams)
}

// estimateRequest estimates the tokens of the request that would be built from
// the session system prompt, userText, conv and the tool definitions. Submit
// and RunTurn both go through it so the pre-turn and post-compaction figures
// cannot drift apart.
func (a *App) estimateRequest(userText string, conv *agent.Conversation, tools agent.Tools) int {
	return estimateTokens(a.promptState(), userText, conv, tools)
}

func (a *App) hasRG() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

func projectTypeName(types []workspace.ProjectType) string {
	if len(types) == 0 {
		return "unknown"
	}
	return string(types[0])
}

// defaultProviderBuilder constructs the real Anthropic/opencode HTTP client.
func defaultProviderBuilder(endpoint string, ep config.EndpointConfig, key, version, sessionID string, log *telemetry.Logger) (provider.Provider, error) {
	return anthropic.New(anthropic.Options{
		Endpoint:      endpoint,
		BaseURL:       ep.BaseURL,
		Auth:          ep.Auth,
		APIKey:        key,
		KeyEnv:        ep.APIKeyEnv,
		PromptCaching: ep.PromptCaching,
		Version:       version,
		SessionID:     sessionID,
		Log:           log,
	})
}

// WorkspaceInfo returns the header data for the TUI.
func (a *App) WorkspaceInfo() tui.WorkspaceInfoMsg {
	types := a.ws.Types()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = string(t)
	}
	return tui.WorkspaceInfoMsg{
		Project: a.ws.Name(),
		Branch:  a.ws.Branch(),
		Dirty:   a.ws.IsDirty(),
		Types:   names,
	}
}

// currentGen returns the session generation messages are being fenced against.
// It does not take genMu (see the invariant on genMu), so it never blocks and
// is safe to call from the Update goroutine.
func (a *App) currentGen() uint64 {
	return a.sessionGen.Load()
}

// turnGen returns the generation carried by ctx (set by RunTurn and RunTool),
// or the current generation when ctx carries none (a direct caller of Request
// or of the tool executor, as the tests are).
func (a *App) turnGen(ctx context.Context) uint64 {
	if g, ok := ctx.Value(turnGenKey{}).(uint64); ok {
		return g
	}
	return a.currentGen()
}

// bumpGeneration starts a new generation. It takes genMu for writing, so it
// waits for every delivery that passed its check under the old generation to
// be accepted by the TUI first; a delivery that starts afterwards fails its
// check and is dropped. For a sender that goes through deliverIfCurrent (send,
// sendAsyncSequenceFenced), messages from the old generation are therefore all
// either already queued ahead of anything NewSession's caller sends next, or
// never sent. Unfenced senders (sendAsyncSequence) get no such guarantee.
func (a *App) bumpGeneration() {
	a.genMu.Lock()
	a.sessionGen.Add(1)
	a.genMu.Unlock()
}

// deliverIfCurrent runs deliver only if gen is still the current generation.
// It holds genMu for reading across deliver, which is what lets bumpGeneration
// wait for a delivery that has passed the check: the check and the hand-over to
// the TUI are one step, so there is no window between them for the bump to fall
// into.
func (a *App) deliverIfCurrent(gen uint64, deliver func()) {
	a.genMu.RLock()
	defer a.genMu.RUnlock()
	if gen != a.sessionGen.Load() {
		return
	}
	deliver()
}

// send delivers a message to the TUI, or drops it if there is nowhere to put
// it or if gen is no longer the current session generation (a /new has
// replaced the session the sender belongs to).
//
// Delivery is abandoned once the root context is cancelled. Without that, a
// tool that finishes just as the user quits blocks forever handing its result
// to a program that has stopped reading.
//
// send is called by RunTool's goroutine, by the tool executor's Invoke and by
// Request. It returns only once the message has been accepted by the TUI,
// dropped by the generation check, or abandoned because the root context was
// cancelled.
func (a *App) send(gen uint64, msg tea.Msg) {
	if a.sendFn != nil {
		a.deliverIfCurrent(gen, func() { a.sendFn(msg) })
		return
	}
	if a.program == nil {
		return
	}
	delivered := make(chan struct{})
	go func() {
		a.deliverIfCurrent(gen, func() { a.program.Send(msg) })
		close(delivered)
	}()
	select {
	case <-delivered:
	case <-a.rootCtx.Done():
	}
}

// sendAsyncSequence delivers multiple messages to the TUI in order, from a
// single background goroutine, without blocking the caller.
//
// It is called from Resolve, which runs on the event loop goroutine and
// must never block. A blocking send from Update would deadlock Bubble Tea:
// Update cannot return until send returns, but send waits for the event loop
// to read from an unbuffered channel, and the event loop cannot run until
// Update returns.
//
// This is used when message ordering must be guaranteed (e.g., a notice
// followed by a resolved message). Sending each message from its own
// goroutine would let them race and arrive out of order; sendAsyncSequence
// sends all messages from one goroutine instead, so they arrive in the
// order provided.
//
// The non-blocking guarantee holds unconditionally, including when sendFn is
// set: sendAsyncSequence always spawns one goroutine and returns immediately,
// whether delivery goes through sendFn (tests) or program.Send (production).
// An earlier version called sendFn synchronously on the caller's goroutine
// when set, which was safe for the mutex-and-append collector in
// app_test.go but not in general — internal/app/integration_test.go's
// driveTUI wires sendFn to an unbuffered channel read from the same
// goroutine that calls Resolve synchronously from Update, deliberately, to
// catch this class of deadlock (see its comment). A synchronous sendFn call
// from inside that call chain would deadlock the harness built to catch the
// bug this mission exists to fix. Spawning unconditionally removes the
// asymmetry rather than documenting around it.
//
// If the program is shutting down or no program is attached, messages are
// abandoned silently.
//
// NOTE: The goroutine spawned here is not tracked by a.inFlight. The
// program's context cancellation during shutdown prevents it from blocking
// indefinitely (see Close and the program's event loop).
func (a *App) sendAsyncSequence(msgs []tea.Msg) {
	a.sendSequence(msgs, a.deliverOne)
}

// sendAsyncSequenceFenced is sendAsyncSequence with each message fenced to gen:
// a message is delivered only if gen is still the current session generation
// when its turn comes (see deliverIfCurrent), so what a cancelled turn still
// has queued never reaches the TUI's new session. Submit (its refusals and its
// BudgetMsg), RunTurn, flushStore and recordCompactionUsage send through it;
// send is the fenced sender for callers that wait for delivery.
// sendAsyncSequence stays for messages that belong to no turn, for example
// notices, approval resolutions and onboarding.
func (a *App) sendAsyncSequenceFenced(gen uint64, msgs []tea.Msg) {
	a.sendSequence(msgs, func(msg tea.Msg) {
		a.deliverIfCurrent(gen, func() { a.deliverOne(msg) })
	})
}

// sendSequence is the body shared by sendAsyncSequence and
// sendAsyncSequenceFenced: one goroutine, messages in order, deliver called for
// each.
func (a *App) sendSequence(msgs []tea.Msg, deliver func(tea.Msg)) {
	if len(msgs) == 0 {
		return
	}
	if a.sendFn == nil && a.program == nil {
		return
	}
	// Spawn a single goroutine to send all messages in order, whatever the
	// delivery path. This ensures ordering and keeps the non-blocking
	// guarantee true for both the test path (sendFn) and the real path
	// (program.Send) — see deliverOne.
	go func() {
		for _, msg := range msgs {
			deliver(msg)
		}
	}()
}

// deliverOne sends a single message via sendFn when a test has set one, or
// via program.Send otherwise. It exists so sendAsyncSequence can call it
// uniformly from inside its one goroutine without branching on the delivery
// path at the call site, which is what let a synchronous sendFn branch creep
// in previously (see sendAsyncSequence's docblock).
func (a *App) deliverOne(msg tea.Msg) {
	if a.sendFn != nil {
		a.sendFn(msg)
		return
	}
	if a.program != nil {
		a.program.Send(msg)
	}
}

// SendNotice queues a non-blocking informational notice to the TUI. It is used
// for best-effort messages that do not require user action, such as explaining
// that a session resume fallback is happening.
func (a *App) SendNotice(text string) {
	a.sendAsyncSequence([]tea.Msg{tui.NoticeMsg{Text: text}})
}

// SendStatusWarning queues a persistent warning to the status bar. It is safe
// to call from outside the Bubble Tea event loop.
func (a *App) SendStatusWarning(text string) {
	a.sendAsyncSequence([]tea.Msg{tui.StatusWarningMsg{Text: text}})
}

// SetSendFn installs the function used to deliver messages to the TUI. It is
// provided for tests in packages that cannot access unexported fields and must
// not be used from production code.
func (a *App) SetSendFn(fn func(any)) {
	a.sendFn = fn
}

// BeginTurn starts a new turn context, cancelling any previous one.
func (a *App) BeginTurn() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.turnStop != nil {
		a.turnStop()
	}
	ctx, cancel := context.WithCancel(a.rootCtx)
	a.turnCtx, a.turnStop = ctx, cancel
	return ctx
}

// CancelTurn cancels the in-flight turn. Safe to call when none is running.
//
// Wired now, with no model call to cancel, because cancellation retrofitted
// later is cancellation that does not work — the seams have to exist before
// anything long-running is threaded through them.
func (a *App) CancelTurn() {
	a.mu.Lock()
	stop := a.turnStop
	a.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// RunTurn starts a real agent turn and streams the events it emits into TUI
// messages. It returns immediately; the turn runs on a background goroutine.
//
// The adapter keeps the TUI-implementation boundary: every agent.Event is
// translated into one of the plain-field messages in internal/tui/messages.go
// before it crosses into the Bubble Tea program. architecture.md §3.
//
// compParams contains the estimated token count, context window, and max output
// needed to determine whether compaction should run before the turn. Compaction
// runs on the turn's goroutine, after BeginTurn, so CancelTurn/Esc cancels it.
func (a *App) RunTurn(ag *agent.Agent, conv *agent.Conversation, userText string, compParams compactionParams) {
	ctx := a.BeginTurn()
	// Submit holds opMu here and NewSession holds it across its generation
	// bump, so this is the generation the turn really starts under.
	gen := a.currentGen()

	a.mu.Lock()
	a.turnsLive++
	a.mu.Unlock()
	a.inFlight.Add(1)
	go func() {
		defer a.inFlight.Done()
		defer func() {
			a.mu.Lock()
			a.turnsLive--
			a.mu.Unlock()
		}()

		var textBlock, thinkingBlock int64
		callIDs := make(map[string]int64)
		ctx = context.WithValue(ctx, toolCallsKey{}, callIDs)
		ctx = context.WithValue(ctx, turnGenKey{}, gen)

		// The agent must use the app-side tool executor so that tool calls
		// driven by the model send ToolStartedMsg / ToolCompletedMsg and are
		// correlated with the IDs allocated for EventToolCallStart.
		ag.Tools = &toolExecutor{app: a}

		a.storeMu.Lock()
		store := a.store
		a.storeMu.Unlock()

		// Compaction runs here, on the turn goroutine after BeginTurn, so a
		// cancel reaches it, and before RecordTurnStarted, so the log reads
		// compaction.applied then turn.started. RunTurn owns the overflow
		// decision: Submit does not gate on the budget.
		//
		// Threshold is 75% of (contextWindow - outputReserve).
		budget := compParams.contextWindow - outputReserve
		if budget < 0 {
			budget = 0
		}
		threshold := budget * 75 / 100

		if compParams.estimatedTokens > threshold {
			// Announce compaction only when there is something to compact, so
			// the nothing-to-compact path does not flash the status.
			if _, _, ok := session.PlanCompaction(conv.Messages); ok {
				a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.CompactingMsg{}})
			}
			// The summary call is capped at the output reserve as well as the
			// model's own maximum: the context is already at least 75% full.
			maxTokens := min(compParams.maxOutput, outputReserve)
			comp, err := a.runCompaction(ctx, ag.Model, conv, maxTokens)
			// Record before the switch: the summary call is billed whether or
			// not the compaction was applied.
			a.recordCompactionUsage(gen, comp, compParams.modelID)
			switch {
			case errors.Is(err, context.Canceled):
				// Cancelled: runCompaction returned before touching conv or
				// the store. The turn is not started.
				a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnCancelledMsg{}})
				a.flushStore(gen, store)
				return
			case errors.Is(err, session.ErrNothingToCompact):
				if compParams.estimatedTokens <= budget {
					// Above the threshold but within budget: proceed.
					break
				}
				a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
					Kind:    "context_overflow",
					Message: "The request exceeds the model's context window.",
					Detail: fmt.Sprintf(
						"Estimated %d tokens against a budget of %d after the %d-token output reserve, and there is nothing to compact. Start a new session with /new.",
						compParams.estimatedTokens, budget, outputReserve),
				}})
				a.flushStore(gen, store)
				return
			case err != nil:
				a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
					Kind:    "context_overflow",
					Message: "Compaction failed. The request exceeds the model's context window.",
					Detail: fmt.Sprintf(
						"Compaction failed: %v. Start a new session with /new.", err),
				}})
				a.flushStore(gen, store)
				return
			default:
				// runCompaction replaced conv.Messages and recorded
				// compaction.applied. Tell the header now, so it is right
				// even if the compacted request is still too large.
				a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.SessionCompactedMsg{}})
				newEstimated := a.estimateRequest(userText, conv, ag.Tools)
				if newEstimated > budget {
					a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.TurnErrorMsg{
						Kind:    "context_overflow",
						Message: "The compacted request still exceeds the model's context window.",
						Detail: fmt.Sprintf(
							"Estimated %d tokens after compaction against a budget of %d. Start a new session with /new.",
							newEstimated, budget),
					}})
					a.flushStore(gen, store)
					return
				}
				// Update the stored estimate for /status display
				a.storeMu.Lock()
				a.lastEstimatedTokens = newEstimated
				a.lastContextWindow = compParams.contextWindow
				a.storeMu.Unlock()
				a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.BudgetMsg{
					EstimatedTokens: newEstimated,
					ContextWindow:   compParams.contextWindow,
				}})
			}
		}

		turn := int(a.turnCount.Add(1))
		if store != nil {
			store.RecordTurnStarted(turn)
		}

		emit := func(ev agent.Event) {
			for _, msg := range a.adaptTurnEvent(ev, &textBlock, &thinkingBlock, callIDs, compParams.modelID) {
				a.sendAsyncSequenceFenced(gen, []tea.Msg{msg})
			}
		}

		err := ag.Turn(ctx, conv, userText, emit)

		var status string
		var final tea.Msg
		switch {
		case err == nil:
			status = "success"
			final = tui.TurnCompleteMsg{}
		case errors.Is(err, context.Canceled):
			status = "cancelled"
			final = tui.TurnCancelledMsg{}
		default:
			status = "error"
			final = tui.TurnErrorMsg{
				Kind:    turnErrorKind(err),
				Message: err.Error(),
				Detail:  turnErrorDetail(err),
			}
		}
		if store != nil {
			store.RecordTurnCompleted(status)
		}

		a.sendAsyncSequenceFenced(gen, []tea.Msg{final})
		a.flushStore(gen, store)
	}()
}

// flushStore flushes store, if there is one, and reports a failure to the TUI
// as an ErrorMsg fenced to gen. RunTurn calls it on every exit, including the
// early returns of the compaction block, so a compaction.applied event is
// never left unflushed; ManualCompact and ClearGrants call it too.
func (a *App) flushStore(gen uint64, store *session.Store) {
	if store == nil {
		return
	}
	if flushErr := store.Flush(); flushErr != nil {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{tui.ErrorMsg{
			Kind:    "session_store",
			Message: fmt.Sprintf("Failed to flush session log: %v", flushErr),
		}})
	}
}

// recordCompactionUsage records the usage a compaction's summary call reported
// and sends the new session totals to the status bar (m4-d9), so the automatic
// and the manual compaction paths both update it. comp is what runCompaction
// returned alongside its error: non-nil on every failure after the model call
// began, carrying Usage only (or, on a cancel after the summary arrived, Usage
// plus an unapplied Messages). A nil comp or an all-zero Usage records and sends
// nothing. The SessionUsageMsg it sends is fenced to gen.
func (a *App) recordCompactionUsage(gen uint64, comp *session.Compaction, modelID string) {
	if comp == nil || !comp.Usage.Nonzero() {
		return
	}
	a.recordUsage(comp.Usage, modelID)
	a.sendAsyncSequenceFenced(gen, []tea.Msg{a.buildSessionUsageMsg(modelID)})
}

// buildSessionUsageMsg builds a SessionUsageMsg from the current session totals.
// Used after both turn and compaction usage events to update the status bar (m4-d9).
//
// The cost is the running session cost, not the cumulative tokens priced at
// modelID's rates. In a session started in this process it is the sum of the
// per-call costs; recordUsage adds each of those same figures to usage.json as
// well, which also holds other sessions' totals. After ResumeLastSession it
// starts from the log's summed tokens priced once at the model in use at
// resume, with per-call costs added after; for a session that switched models
// before the resume that base differs from the sum of its per-call costs.
// CostState and ModelKnown describe modelID, the model in use now.
func (a *App) buildSessionUsageMsg(modelID string) tui.SessionUsageMsg {
	a.storeMu.Lock()
	in, out := a.sessionTokensIn, a.sessionTokensOut
	read, write := a.sessionTokensRead, a.sessionTokensWrite
	cost := a.sessionCostUSD
	a.storeMu.Unlock()

	return sessionUsageMsgFor(in, out, read, write, cost, modelID)
}

// sessionUsageMsgFor builds a SessionUsageMsg from the given totals. CostState
// and ModelKnown describe modelID. It takes no lock, so ResumeLastSession, which
// already holds storeMu, can use it where buildSessionUsageMsg would deadlock.
func sessionUsageMsgFor(in, out, read, write int64, cost float64, modelID string) tui.SessionUsageMsg {
	modelInfo := provider.LookupModel(modelID)
	return tui.SessionUsageMsg{
		TokensIn:   int(in),
		TokensOut:  int(out),
		CacheRead:  int(read),
		CacheWrite: int(write),
		CostUSD:    cost,
		CostState:  provider.CostState(modelInfo.Pricing),
		ModelKnown: modelInfo.Known,
	}
}

// recordUsage records one model call's usage in three places: the session store
// (a usage event), the TUI session totals that /status reads, and, when a
// layout is present, usage.json with the cost for modelID. The same per-call
// cost is added to the running session cost that buildSessionUsageMsg reports,
// on top of whatever base the session started with: zero, or the priced log
// totals ResumeLastSession seeded.
//
// Turn calls and compaction calls both come through here, so the totals count
// every billed call. The header's per-call token figure is a separate message
// (tui.UsageMsg) that only turns send.
//
// A failed usage.json update is logged at Warn and dropped. So is the move of a
// corrupt usage.json: session.AddUsage reports it through its first return
// value, because that package must not write to the terminal.
func (a *App) recordUsage(usage agent.Usage, modelID string) {
	modelInfo := provider.LookupModel(modelID)
	costUSD, costOK := modelInfo.Cost(
		int64(usage.InputTokens),
		int64(usage.OutputTokens),
		int64(usage.CacheReadTokens),
		int64(usage.CacheWriteTokens),
	)

	a.storeMu.Lock()
	store := a.store
	layout := a.layout
	a.sessionTokensIn += int64(usage.InputTokens)
	a.sessionTokensOut += int64(usage.OutputTokens)
	a.sessionTokensRead += int64(usage.CacheReadTokens)
	a.sessionTokensWrite += int64(usage.CacheWriteTokens)
	if costOK {
		a.sessionCostUSD += costUSD
	}
	a.storeMu.Unlock()

	if store != nil {
		store.RecordUsage(usage)
	}
	if layout != nil {
		recovered, err := session.AddUsage(
			layout.Root,
			int64(usage.InputTokens),
			int64(usage.OutputTokens),
			int64(usage.CacheReadTokens),
			int64(usage.CacheWriteTokens),
			costUSD,
			costOK,
			time.Now().UTC(),
		)
		if recovered != "" {
			a.log.Warn("recovered from corrupt usage.json", "moved_to", recovered)
		}
		if err != nil {
			a.log.Warn("usage update failed", "error", err)
		}
	}
}

// adaptTurnEvent translates a single agent.Event into TUI messages. It mutates
// the adapter state (text/thinking block IDs and the provider-call-id map) so
// deltas for the same block reuse the transcript item they created.
func (a *App) adaptTurnEvent(ev agent.Event, textBlock, thinkingBlock *int64, callIDs map[string]int64, modelID string) []tea.Msg {
	switch ev.Type {
	case agent.EventTextDelta:
		if *textBlock == 0 {
			*textBlock = a.nextID.Add(1)
		}
		return []tea.Msg{tui.AssistantTextDeltaMsg{BlockID: *textBlock, Delta: ev.Text}}

	case agent.EventThinkingDelta:
		if *thinkingBlock == 0 {
			*thinkingBlock = a.nextID.Add(1)
		}
		return []tea.Msg{tui.ThinkingDeltaMsg{BlockID: *thinkingBlock, Delta: ev.Text}}

	case agent.EventThinkingDone:
		// The TUI card already shows the streamed deltas; the conversation stores the
		// authoritative final block because assembly.accept replaces the delta-built
		// block with the EventThinkingDone payload. Display fidelity and conversation
		// fidelity are deliberately separate, so no display message is needed.
		return nil

	case agent.EventToolCallStart:
		if ev.ToolCall == nil {
			return nil
		}
		id := a.nextID.Add(1)
		callIDs[ev.ToolCall.ID] = id
		return []tea.Msg{tui.ToolStartedMsg{
			ID:     id,
			Name:   ev.ToolCall.Name,
			Target: describeToolCallInput(ev.ToolCall.Input),
		}}

	case agent.EventToolCallDelta:
		// The agent forwards partial tool-call input, but the complete call
		// and its result are handled by EventToolCallEnd and the agent's
		// Tools implementation. No TUI update for a partial input fragment.
		return nil

	case agent.EventToolCallEnd:
		// The agent has finished receiving the complete tool call. The actual
		// tool execution and ToolCompletedMsg are produced by the agent.Tools
		// adapter (RunTool / toolExecutor), which owns the app-side tool ID.
		return nil

	case agent.EventMessageDone:
		// A block boundary: the next assistant text or thinking stream starts
		// a new transcript item.
		*textBlock, *thinkingBlock = 0, 0
		return nil

	case agent.EventUsage:
		if ev.Usage == nil {
			return nil
		}

		// recordUsage also adds the call to the session totals.
		a.recordUsage(*ev.Usage, modelID)

		return []tea.Msg{
			tui.UsageMsg{
				InputTokens:      ev.Usage.InputTokens,
				OutputTokens:     ev.Usage.OutputTokens,
				CacheReadTokens:  ev.Usage.CacheReadTokens,
				CacheWriteTokens: ev.Usage.CacheWriteTokens,
			},
			a.buildSessionUsageMsg(modelID),
		}

	case agent.EventError:
		// Turn returns the same error after the stream ends, so we surface it as TurnErrorMsg from the final return path rather than duplicating it here.
		return nil
	}

	return nil
}

// describeToolCallInput extracts a one-line target from a tool call's JSON input.
func describeToolCallInput(input json.RawMessage) string {
	var decoded map[string]any
	if err := json.Unmarshal(input, &decoded); err != nil {
		return ""
	}
	return describeInput(decoded)
}

// turnErrorKind maps agent sentinel errors to the plain strings the TUI renders.
func turnErrorKind(err error) string {
	switch {
	case errors.Is(err, agent.ErrMaxTurnsExceeded):
		return "max_turns_exceeded"
	case errors.Is(err, agent.ErrContextOverflow):
		return "context_overflow"
	default:
		return "turn_error"
	}
}

// turnErrorDetail returns a human-readable detail line for a turn error.
func turnErrorDetail(err error) string {
	switch {
	case errors.Is(err, agent.ErrMaxTurnsExceeded):
		return "The model kept requesting tools without producing a final answer."
	case errors.Is(err, agent.ErrContextOverflow):
		return "The assembled request exceeded the model's context window."
	default:
		return ""
	}
}

// RunTool invokes a tool and reports the result to the TUI. It returns
// immediately.
//
// Everything after the bookkeeping happens on the goroutine — including the
// ToolStartedMsg — and that is not tidiness, it is the difference between
// working and deadlocking. RunTool is called synchronously from the TUI's
// Update, and program.Send blocks until the Bubble Tea event loop reads the
// message. Sending from this function means Update waits for a loop that
// cannot run until Update returns: the TUI blocks on itself, no tool ever
// starts, and the interface simply stops responding with no error anywhere.
//
// This is exactly the deadlock architecture.md §5 names when it says the TUI
// loop must never block, on anything, ever. It is easy to reintroduce, because
// the offending line looks like ordinary sequencing.
func (a *App) RunTool(name string, input map[string]any) {
	ctx := a.BeginTurn()
	gen := a.currentGen()
	ctx = context.WithValue(ctx, turnGenKey{}, gen)
	id := a.nextID.Add(1)
	target := describeInput(input)

	a.inFlight.Add(1)
	go func() {
		defer a.inFlight.Done()

		a.send(gen, tui.ToolStartedMsg{ID: id, Name: name, Target: target})
		a.log.Debug("tool started", "tool", name, "id", id, "target", target)

		raw, err := json.Marshal(input)
		if err != nil {
			a.send(gen, tui.ToolCompletedMsg{
				ID: id, Name: name, Target: target,
				OK: false, ErrorKind: string(tool.KindToolInputInvalid),
				ErrorMsg: "could not encode tool input: " + err.Error(),
			})
			return
		}

		start := time.Now()
		ctx = tool.WithToolID(ctx, id)
		ctx = tool.WithProgressSink(ctx, func(chunk string) {
			a.send(gen, tui.CommandOutputChunkMsg{ID: id, Chunk: chunk})
		})
		res := a.reg.Invoke(ctx, name, raw)
		elapsed := time.Since(start)

		msg := tui.ToolCompletedMsg{
			ID: id, Name: name, Target: target,
			OK:        res.OK,
			Summary:   res.DisplaySummary,
			Content:   res.Content,
			Truncated: res.Truncated,
			Elapsed:   elapsed,
		}
		if res.Error != nil {
			msg.ErrorKind = string(res.Error.Kind)
			msg.ErrorMsg = res.Error.Message
		}
		a.log.Debug("tool completed",
			"tool", name, "id", id, "ok", res.OK,
			"duration_ms", res.DurationMS, "kind", msg.ErrorKind,
			telemetry.Content("summary", res.DisplaySummary))
		a.send(gen, msg)
	}()
}

// canApproveForSession determines whether a session-scope grant is possible
// by consulting policy and the sessionGrantsDisabled flag. Both the operation
// type AND the specific argv must be acceptable, and session grants must not
// be disabled (e.g. by concurrent instance contention).
func (a *App) canApproveForSession(op policy.Operation, argv []string) bool {
	return !a.sessionGrantsDisabled &&
		a.pol.CanApproveForSession(op) &&
		a.pol.CanGrant(op, argv)
}

// Request asks the user for approval, blocking until the user responds or a
// context is cancelled. It is called only from tool goroutines, never from
// Update or View. It sends ApprovalRequestedMsg to the TUI and waits for
// Resolve to be called. It returns the approval outcome and any grant error
// that occurred (nil if the approval was not for a session grant, or if the
// grant succeeded).
func (a *App) Request(ctx context.Context, req ApprovalRequest) (ApprovalOutcome, error) {
	// If the caller supplied an ID (e.g. mapped from a tool invocation), use it;
	// otherwise allocate a fresh approval ID. This keeps direct callers and the
	// tool adapter in the same ID space while preserving test callers that pass 0.
	id := req.ID
	if id == 0 {
		id = a.approveID.Add(1)
	}

	app := &approval{
		id:        id,
		decision:  make(chan ApprovalOutcome, 1), // buffered: capacity 1
		argv:      req.Argv,
		operation: req.Operation,
	}

	a.approveMu.Lock()
	a.approvals[id] = app
	a.approveMu.Unlock()

	// Calculate whether session approval is possible by consulting policy.
	// Both the operation type AND the specific argv must be acceptable.
	// If session grants are disabled (e.g. because another instance is running),
	// session approval is never offered.
	canApproveForSession := a.canApproveForSession(req.Operation, req.Argv)

	// Compute display strings for the card
	var subject, grantScope string
	var detail []string
	var diffFilename string
	var diffLines []string
	var diffAdded, diffRemoved int

	switch req.Operation {
	case policy.OperationPatch:
		subject, detail = formatPatchDisplay(req.Changes)
		diffFilename, diffLines, diffAdded, diffRemoved = computeDiffDisplay(req.Changes)
	case policy.OperationCommand:
		subject, grantScope = formatCommandDisplay(req.Argv)
		detail = formatCommandDetail(req.Description, req.Argv)
	case policy.OperationUnspecified:
		// Unspecified operation should not occur in practice.
	}

	// Send the approval message to the TUI.
	a.send(a.turnGen(ctx), tui.ApprovalRequestedMsg{
		ID:                   id,
		ToolID:               tool.ToolIDFrom(ctx),
		Description:          req.Description,
		Kind:                 kindString(req.Operation),
		CanApproveForSession: canApproveForSession,
		Subject:              subject,
		Detail:               detail,
		GrantScope:           grantScope,
		Argv:                 req.Argv,
		DiffFilename:         diffFilename,
		DiffLines:            diffLines,
		DiffAdded:            diffAdded,
		DiffRemoved:          diffRemoved,
	})

	// Block on three things:
	// 1. User decision via the buffered channel
	// 2. Turn cancellation (user pressed Esc)
	// 3. Root cancellation (process shutting down)
	select {
	case outcome := <-app.decision:
		a.approveMu.Lock()
		grantErr := app.grantError
		delete(a.approvals, id)
		a.approveMu.Unlock()
		return outcome, grantErr
	case <-ctx.Done():
		a.approveMu.Lock()
		delete(a.approvals, id)
		a.approveMu.Unlock()
		return ApprovalOutcomeCancelled, nil
	case <-a.rootCtx.Done():
		a.approveMu.Lock()
		delete(a.approvals, id)
		a.approveMu.Unlock()
		return ApprovalOutcomeCancelled, nil
	}
}

// Resolve records a user's decision on an approval. It must never block.
// It is called from Update when the user presses y/a/n or when the turn
// is cancelled. It is safe to call for an approval that has already been
// resolved or does not exist.
func (a *App) Resolve(id int64, outcome ApprovalOutcome) {
	a.approveMu.Lock()
	app, exists := a.approvals[id]
	a.approveMu.Unlock()

	if !exists {
		return
	}

	// If approving for session, try to record a grant. Route all approvals through
	// Grant so every validation failure is handled the same way (no bypass checks here).
	// Use the real operation from the request, not a bare literal.
	// If the grant fails, the actual outcome is Approved (allow-once), not the user's choice.
	confirmedOutcome := outcome
	grantAttempted := false
	var grantErr error
	if outcome == ApprovalOutcomeSession {
		grantAttempted = true
		if err := a.pol.Grant(app.operation, app.argv); err != nil {
			// Record the grant error in the approval struct so Request can return it
			// before the approval is deleted from the map.
			a.approveMu.Lock()
			app.grantError = err
			a.approveMu.Unlock()
			a.log.Debug("grant rejected", "argv", app.argv, "err", err)
			// Grant failed: the command runs allow-once, not for session.
			confirmedOutcome = ApprovalOutcomeOnce
			grantErr = err
		} else {
			// Record a successful session-scope grant in the durable log.
			a.storeMu.Lock()
			store := a.store
			a.storeMu.Unlock()
			if store != nil {
				action := "run_command"
				if app.operation == policy.OperationPatch {
					action = "apply_patch"
				}
				store.RecordScopeGranted(action, strings.Join(app.argv, " "))
			}
		}
	}

	// Send confirmation of the actual outcome to the TUI so it can update the card.
	// This is only necessary when the outcome might differ from what the user chose,
	// which happens when a session grant is attempted (and might fail).
	// Use sendAsyncSequence to ensure messages arrive in the correct order.
	// This avoids the race that would occur if each message were sent from
	// its own goroutine, which would let them arrive out of order.
	if grantAttempted {
		var msgs []tea.Msg
		// If the grant failed, queue a notice first.
		if grantErr != nil {
			msgs = append(msgs, tui.NoticeMsg{
				Text: fmt.Sprintf("Session grant denied: %s", grantErr.Error()),
			})
		}
		// Convert from app.ApprovalOutcome to tui.ApprovalOutcome. The numeric values
		// don't match: app uses 0-3 but tui has an extra Unresolved(0) at the start,
		// so we must use FromAppOutcome to map correctly.
		msgs = append(msgs, tui.ApprovalResolvedMsg{
			ID:      id,
			Outcome: tui.FromAppOutcome(int(confirmedOutcome)),
		})
		// Send all messages in sequence from a single goroutine, ensuring order.
		a.sendAsyncSequence(msgs)
	}

	// Send on the buffered channel. If it already has a value, this is a
	// no-op (the select will have taken the first one).
	select {
	case app.decision <- outcome:
	default:
		// Already resolved; drop the duplicate silently.
	}
}

// RequestToolApproval implements tool.Approver.Request for adapting between
// the internal ApprovalOutcome type and the public policy.Decision return type.
// The tool's apply_patch calls this method expecting policy.Decision back.
// This method converts the internal ApprovalOutcome to the decision the tool expects:
// - ApprovalOutcomeOnce → DecisionAllow (allow this operation once)
// - ApprovalOutcomeSession → DecisionSession (allow and record a grant, or DecisionAllow if grant failed)
// - ApprovalOutcomeDeny or ApprovalOutcomeCancelled → DecisionDeny (deny the operation)
func (a *App) RequestToolApproval(ctx context.Context, req tool.ApprovalRequest) policy.Decision {
	// Convert tool.ApprovalRequest to app.ApprovalRequest, carrying over the
	// invocation-mapped ID and the operation payload.
	appReq := ApprovalRequest{
		ID:          req.ID,
		Description: req.Description,
		Operation:   req.Operation,
		Argv:        req.Argv,
		Changes:     req.Changes,
	}

	// Get the approval outcome and any grant error (blocks until user responds or context is cancelled).
	outcome, grantErr := a.Request(ctx, appReq)

	// If session approval was chosen, check if the grant actually succeeded.
	// If the grant failed, downgrade to allow-once (DecisionAllow) so the tool
	// still runs but doesn't get a session grant.
	if outcome == ApprovalOutcomeSession {
		if grantErr != nil {
			a.log.Debug("grant failed for session approval", "err", grantErr)
			return policy.DecisionAllow // Downgrade to allow-once
		}
		return policy.DecisionSession
	}

	// Convert the outcome to a policy.Decision for the tool
	switch outcome {
	case ApprovalOutcomeOnce:
		return policy.DecisionAllow
	case ApprovalOutcomeDeny:
		return policy.DecisionDeny
	case ApprovalOutcomeCancelled:
		return policy.DecisionDeny // Cancellation is treated as denial to the tool
	default:
		return policy.DecisionDeny // Safeguard for unknown outcomes
	}
}

// ResolveToolApproval implements tool.Approver.Resolve for the tool layer.
// It's called when the TUI resolves an approval and records the decision.
func (a *App) ResolveToolApproval(id int64, decision policy.Decision) {
	// Convert policy.Decision back to ApprovalOutcome for internal handling
	outcome := ApprovalOutcomeDeny // default
	switch decision {
	case policy.DecisionAllow:
		outcome = ApprovalOutcomeOnce
	case policy.DecisionSession:
		outcome = ApprovalOutcomeSession
	case policy.DecisionDeny:
		outcome = ApprovalOutcomeDeny
	case policy.DecisionAskUser:
		// AskUser is not a resolution, so it resolves as deny.
		outcome = ApprovalOutcomeDeny
	}

	a.Resolve(id, outcome)
}

// runCompaction summarises the middle of conv.Messages with one model call,
// replaces conv.Messages with the compacted result, and records
// compaction.applied. maxTokens caps the summary request's output.
//
// It returns the applied Compaction (including its Usage) on success. It returns
// session.ErrNothingToCompact, with a nil Compaction, when there are fewer than 6
// turns, an error wrapping context.Canceled when the turn was cancelled, and another
// error on model or provider failure. On any error conv and the store are untouched.
// On every error after the model call began the Compaction is still non-nil, so the
// caller can record its Usage; it was not applied and must not be used for anything
// else. A cancel after the summary arrived returns a Compaction that also carries
// the unapplied Messages.
//
// The cancellation check immediately before the swap is what keeps a cancel
// that lands after session.Compact's own last check from half-applying: the
// swap and the store write both follow it with no further wait.
//
// conv is not guarded by a lock. Its single-writer guarantee comes from two
// callers: RunTurn's goroutine, which Submit starts only when no manual
// compaction is running, and ManualCompact, which starts only when no turn
// goroutine is live. NewSession waits for both before it replaces a.conv.
// Callers that bypass Submit and ManualCompact, as tests do, must keep to the
// same rule themselves.
func (a *App) runCompaction(ctx context.Context, model agent.Model, conv *agent.Conversation, maxTokens int) (*session.Compaction, error) {
	comp, err := session.Compact(ctx, model, conv.Messages, maxTokens)
	// comp may be non-nil even on error, carrying usage from the stream.
	// Return early on error, but allow callers to record usage from comp before
	// they switch on the error type.
	if err != nil {
		return comp, err
	}

	if a.afterCompactHook != nil {
		a.afterCompactHook()
	}
	if err := ctx.Err(); err != nil {
		return comp, fmt.Errorf("compaction cancelled before it was applied: %w", err)
	}

	conv.Messages = comp.Messages

	// compaction.applied is recorded here, before the caller records
	// turn.started, so the log reads: compaction.applied, turn.started, ...
	a.storeMu.Lock()
	store := a.store
	a.storeMu.Unlock()
	if store != nil {
		store.RecordCompaction(comp.From, comp.To, comp.Summary)
	}

	return comp, nil
}

// GetActiveModel builds the active model from the current endpoint configuration.
// It returns nil if the endpoint or API key is not available.
func (a *App) GetActiveModel(getenv func(string) string) agent.Model {
	epName := a.cfg.Provider.Default
	ep, ok := a.cfg.Provider.Endpoints[epName]
	if !ok {
		return nil
	}

	key, _ := ep.ResolveKey(getenv)
	if key == "" {
		return nil
	}

	prov, err := a.providerBuilder(epName, ep, key, a.version, a.sessionID, a.log)
	if err != nil {
		return nil
	}

	return &providerModel{
		p:             prov,
		model:         ep.Model,
		promptCaching: ep.PromptCaching,
		thinking:      parseThinkingLevel(ep.Thinking),
	}
}

// isCompacting reports whether a ManualCompact is running.
func (a *App) isCompacting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.compacting
}

// beginManualCompact claims the right to compact. It returns ok=false, with
// nothing claimed, when another session operation holds opMu, a turn goroutine
// is live, or a compaction is already running. On success it has started the
// turn context, marked compacting and registered with inFlight, so
// NewSession's Wait covers the compaction; the caller must call the returned
// done exactly once.
func (a *App) beginManualCompact() (ctx context.Context, conv *agent.Conversation, done func(), ok bool) {
	if !a.opMu.TryLock() {
		return nil, nil, nil, false
	}
	defer a.opMu.Unlock()

	a.mu.Lock()
	if a.compacting || a.turnsLive > 0 {
		a.mu.Unlock()
		return nil, nil, nil, false
	}
	a.compacting = true
	a.mu.Unlock()

	ctx = a.BeginTurn()
	a.inFlight.Add(1)
	return ctx, a.conv, func() {
		a.mu.Lock()
		a.compacting = false
		a.mu.Unlock()
		a.inFlight.Done()
	}, true
}

// ManualCompact compacts the current conversation without starting a turn and
// returns the real outcome as the one ManualCompactMsg the TUI will see. It
// blocks for the whole summary call, so the TUI runs it inside a tea.Cmd and
// keeps its busy state until the returned message arrives. The outcome is the
// return value; the only messages it sends are the ones recordCompactionUsage
// and flushStore send, fenced to the generation it started under.
//
// It gets its own turn context from BeginTurn, so CancelTurn (Esc) cancels
// it, and it goes through runCompaction, the path RunTurn uses, so the
// afterCompactHook seam, the cancellation re-check before the swap and the
// compaction.applied record are shared. Outcomes are classified with
// errors.Is. The store is flushed on every exit, as RunTurn does.
func (a *App) ManualCompact(getenv func(string) string) tui.ManualCompactMsg {
	ctx, conv, done, ok := a.beginManualCompact()
	if !ok {
		return tui.ManualCompactMsg{Outcome: tui.ManualCompactBusy}
	}
	defer done()
	// NewSession waits for this compaction (beginManualCompact registered it
	// with inFlight) before it bumps the generation, so this is stable here.
	gen := a.currentGen()

	model := a.GetActiveModel(getenv)
	if model == nil {
		return tui.ManualCompactMsg{
			Outcome: tui.ManualCompactFailed,
			Detail:  "no active model available",
		}
	}
	ep := a.cfg.Provider.Endpoints[a.cfg.Provider.Default]
	maxTokens := min(a.modelInfo(ep.Model).MaxOutput, outputReserve)

	a.storeMu.Lock()
	store := a.store
	a.storeMu.Unlock()

	comp, err := a.runCompaction(ctx, model, conv, maxTokens)
	// Record before classifying the outcome: the summary call is billed whether
	// or not the compaction was applied.
	a.recordCompactionUsage(gen, comp, ep.Model)
	a.flushStore(gen, store)

	switch {
	case err == nil:
		// conv now holds the compacted messages, so the estimate /status
		// reads is stale. No user text is pending. The window comes from
		// a.modelInfo, as Submit's estimate does. This stores the estimate
		// for StatusInfo and sends no message.
		window := a.modelInfo(ep.Model).ContextWindow
		estimated := a.estimateRequest("", conv, &toolExecutor{app: a})
		a.storeMu.Lock()
		a.lastEstimatedTokens = estimated
		a.lastContextWindow = window
		a.storeMu.Unlock()
		return tui.ManualCompactMsg{Outcome: tui.ManualCompactApplied}
	case errors.Is(err, session.ErrNothingToCompact):
		return tui.ManualCompactMsg{Outcome: tui.ManualCompactNothing}
	case errors.Is(err, context.Canceled):
		return tui.ManualCompactMsg{Outcome: tui.ManualCompactCancelled}
	default:
		return tui.ManualCompactMsg{Outcome: tui.ManualCompactFailed, Detail: err.Error()}
	}
}

// describeInput picks the field worth showing on a tool card.
func describeInput(input map[string]any) string {
	// Check string fields first: path and query.
	for _, key := range []string{"path", "query"} {
		if v, ok := input[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}

	// For run_command, show argv as a space-separated string, sanitised to prevent
	// terminal escape injection (internal/tui matches this approach for approval cards).
	// argv comes from the model/tool input as []string.
	if v, ok := input["argv"]; ok {
		// Handle both []string (production) and []any (JSON decoding).
		if strSlice, ok := v.([]string); ok && len(strSlice) > 0 {
			cmdStr := strings.Join(strSlice, " ")
			// Sanitise using the same function approval cards use (internal/tui/transcript.go).
			return tui.SanitizeSingleLine(cmdStr)
		}
		if anySlice, ok := v.([]any); ok && len(anySlice) > 0 {
			// Build a space-separated command string from argv.
			var parts []string
			for _, arg := range anySlice {
				if s, ok := arg.(string); ok {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				cmdStr := strings.Join(parts, " ")
				// Sanitise using the same function approval cards use (internal/tui/transcript.go).
				return tui.SanitizeSingleLine(cmdStr)
			}
		}
	}
	return ""
}
