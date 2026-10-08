// Package app wires the pieces together and routes between them.
//
// It exists because it is the only place that knows about both sides, which
// makes it the only place adapters can live. It holds no business logic: if a
// rule about what Kirsch may do ends up here, it is in the wrong package.
// architecture.md §3.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
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
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// toolCallsKey is the context key for carrying the provider-call-id → app-side
// tool ID map from RunTurn into the toolExecutor that completes agent-driven
// tool calls. architecture.md §4.
type toolCallsKey struct{}

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
		te.app.send(tui.ToolStartedMsg{ID: id, Name: c.Name, Target: target})
	}

	res := te.app.reg.Invoke(ctx, c.Name, c.Input)

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
	te.app.send(msg)

	return agent.ToolResult{
		OK:        res.OK,
		Content:   res.Content,
		Truncated: res.Truncated,
		ErrorKind: msg.ErrorKind,
	}
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

	// Approval state: tracks pending approvals.
	approveMu sync.Mutex
	approvals map[int64]*approval
	approveID atomic.Int64

	// providerBuilder constructs the live provider client from the active
	// endpoint. Production uses the Anthropic adapter; tests swap it for a fake.
	providerBuilder providerBuilder

	// sessionID is required for the opencode endpoint and generated once per
	// invocation in main.go.
	sessionID string
	// version is the Kirsch version, passed to the provider for the User-Agent.
	version string

	// Session-scoped prompt state. Assembled once at session start and reused
	// for every turn; the chosen context file and size are kept for /status.
	// architecture.md §5 and kirsch-plan.md §6.2.
	systemPrompt     string
	contextFile      string
	contextSize      int
	startOnce        sync.Once
	loadProjectCtxFn func(candidates []string, maxBytes int, engine prompt.Engine) (string, string, int, []string, error)

	// conv holds the multi-turn conversation. It is mutated by agent.Turn on a
	// background goroutine and passed to every RunTurn call.
	conv *agent.Conversation
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
		loadProjectCtxFn: prompt.LoadProjectContext,
		conv:             &agent.Conversation{},
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

// Registry exposes the tool registry. Used by `kirsch doctor` in M5 and by
// tests; the TUI never sees it.
func (a *App) Registry() *tool.Registry { return a.reg }

// Grants returns the list of active session-scoped command grants.
// Each grant is a space-separated command prefix.
func (a *App) Grants() []string { return a.pol.Grants() }

// ClearGrants removes all active session-scoped command grants.
func (a *App) ClearGrants() { a.pol.ClearGrants() }

// Attach connects the Bubble Tea program. Results reach the TUI only through
// program.Send — architecture.md §3 rule 4, since TUI state may be mutated
// only inside Update.
func (a *App) Attach(p *tea.Program) { a.program = p }

// SetSessionID sets the per-invocation session identifier required by the
// opencode endpoint. Called once from cmd/kirsch/main.go before the first turn.
func (a *App) SetSessionID(id string) { a.sessionID = id }

// SetVersion sets the Kirsch version used in the provider User-Agent.
func (a *App) SetVersion(v string) { a.version = v }

// StartSession assembles the system prompt once per session and stores the
// chosen project-context file and size for /status. It is called from
// cmd/kirsch/main.go after the TUI is attached; subsequent turns reuse the
// cached prompt rather than re-reading the project context. kirsch-plan.md §6.2.
func (a *App) StartSession(getenv func(string) string) {
	a.startOnce.Do(func() { a.loadAndStoreSessionPrompt(getenv) })
}

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
}

// Close cancels everything and waits, briefly, for in-flight tools.
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
	_ = a.log.Close()
}

// closeGrace bounds how long Close waits for in-flight tools.
const closeGrace = 2 * time.Second

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

	return tui.StatusInfoMsg{
		Endpoint:           epName,
		BaseURLHost:        hostOnly(ep.BaseURL),
		KeySource:          keySource,
		ProxyHost:          proxyHost(getenv),
		ContextFile:        a.contextFile,
		ContextSize:        a.contextSize,
		ShowDataFlowNotice: epName == "opencode",
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
	epName := a.cfg.Provider.Default
	ep, ok := a.cfg.Provider.Endpoints[epName]
	if !ok {
		a.sendAsyncSequence([]tea.Msg{tui.TurnErrorMsg{
			Kind:    "config",
			Message: fmt.Sprintf("unknown endpoint %q", epName),
		}})
		return
	}

	key, _ := ep.ResolveKey(getenv)
	if key == "" {
		a.sendAsyncSequence([]tea.Msg{tui.TurnErrorMsg{
			Kind:    "auth",
			Message: fmt.Sprintf("no API key for endpoint %q (set %s or %s)", epName, "KIRSCH_"+ep.APIKeyEnv, ep.APIKeyEnv),
		}})
		return
	}

	// Ensure the session prompt is assembled exactly once. In production this is
	// already done in startupPostAttach; the guard lets tests call Submit without
	// first calling StartSession while still reading the context only once.
	a.StartSession(getenv)

	prov, err := a.providerBuilder(epName, ep, key, a.version, a.sessionID, a.log)
	if err != nil {
		a.sendAsyncSequence([]tea.Msg{tui.TurnErrorMsg{
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

	info := provider.LookupModel(ep.Model)

	ag := &agent.Agent{
		Model:     model,
		MaxTokens: info.MaxOutput,
		System:    a.systemPrompt,
	}

	// The tool specs are part of the request size, so attach the executor before
	// estimating the budget. RunTurn will set the same field again.
	ag.Tools = &toolExecutor{app: a}

	estimated := estimateTokens(a.systemPrompt, userText, a.conv, ag.Tools)
	budget := info.ContextWindow - outputReserve
	if budget < 0 {
		budget = 0
	}
	if estimated > budget {
		a.sendAsyncSequence([]tea.Msg{tui.TurnErrorMsg{
			Kind:    "context_overflow",
			Message: "The request exceeds the model's context window.",
			Detail: fmt.Sprintf(
				"Estimated %d tokens against a budget of %d after the %d-token output reserve. Start a new session with /new.",
				estimated, budget, outputReserve),
		}})
		return
	}

	a.sendAsyncSequence([]tea.Msg{tui.BudgetMsg{
		EstimatedTokens: estimated,
		ContextWindow:   info.ContextWindow,
	}})
	a.RunTurn(ag, a.conv, userText)
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

// send delivers a message to the TUI, or drops it if no program is attached.
// send delivers a message to the TUI, or drops it if there is nowhere to put
// it.
//
// Delivery is abandoned once the root context is cancelled. Without that, a
// tool that finishes just as the user quits blocks forever handing its result
// to a program that has stopped reading.
//
// send is called only from tool goroutines spawned in RunTool. Those goroutines
// need the delivery guarantee: they wait for the message to be accepted so they
// know the result reached the TUI.
func (a *App) send(msg tea.Msg) {
	if a.sendFn != nil {
		a.sendFn(msg)
		return
	}
	if a.program == nil {
		return
	}
	delivered := make(chan struct{})
	go func() {
		a.program.Send(msg)
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
			a.deliverOne(msg)
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
func (a *App) RunTurn(ag *agent.Agent, conv *agent.Conversation, userText string) {
	ctx := a.BeginTurn()

	a.inFlight.Add(1)
	go func() {
		defer a.inFlight.Done()

		var textBlock, thinkingBlock int64
		callIDs := make(map[string]int64)
		ctx = context.WithValue(ctx, toolCallsKey{}, callIDs)

		// The agent must use the app-side tool executor so that tool calls
		// driven by the model send ToolStartedMsg / ToolCompletedMsg and are
		// correlated with the IDs allocated for EventToolCallStart.
		ag.Tools = &toolExecutor{app: a}

		emit := func(ev agent.Event) {
			for _, msg := range a.adaptTurnEvent(ev, &textBlock, &thinkingBlock, callIDs) {
				a.sendAsyncSequence([]tea.Msg{msg})
			}
		}

		err := ag.Turn(ctx, conv, userText, emit)

		var final tea.Msg
		switch {
		case err == nil:
			final = tui.TurnCompleteMsg{}
		case errors.Is(err, context.Canceled):
			final = tui.TurnCancelledMsg{}
		default:
			final = tui.TurnErrorMsg{
				Kind:    turnErrorKind(err),
				Message: err.Error(),
				Detail:  turnErrorDetail(err),
			}
		}
		a.sendAsyncSequence([]tea.Msg{final})
	}()
}

// adaptTurnEvent translates a single agent.Event into TUI messages. It mutates
// the adapter state (text/thinking block IDs and the provider-call-id map) so
// deltas for the same block reuse the transcript item they created.
func (a *App) adaptTurnEvent(ev agent.Event, textBlock, thinkingBlock *int64, callIDs map[string]int64) []tea.Msg {
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
		return []tea.Msg{tui.UsageMsg{
			InputTokens:      ev.Usage.InputTokens,
			OutputTokens:     ev.Usage.OutputTokens,
			CacheReadTokens:  ev.Usage.CacheReadTokens,
			CacheWriteTokens: ev.Usage.CacheWriteTokens,
		}}

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
	id := a.nextID.Add(1)
	target := describeInput(input)

	a.inFlight.Add(1)
	go func() {
		defer a.inFlight.Done()

		a.send(tui.ToolStartedMsg{ID: id, Name: name, Target: target})
		a.log.Debug("tool started", "tool", name, "id", id, "target", target)

		raw, err := json.Marshal(input)
		if err != nil {
			a.send(tui.ToolCompletedMsg{
				ID: id, Name: name, Target: target,
				OK: false, ErrorKind: string(tool.KindToolInputInvalid),
				ErrorMsg: "could not encode tool input: " + err.Error(),
			})
			return
		}

		start := time.Now()
		ctx = tool.WithToolID(ctx, id)
		ctx = tool.WithProgressSink(ctx, func(chunk string) {
			a.send(tui.CommandOutputChunkMsg{ID: id, Chunk: chunk})
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
		a.send(msg)
	}()
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
	// CanApproveForSession checks the operation type; CanGrant checks whether
	// the specific argv would be accepted (shells, bare wildcards, empty argv, etc).
	canApproveForSession := a.pol.CanApproveForSession(req.Operation) &&
		a.pol.CanGrant(req.Operation, req.Argv)

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
	a.send(tui.ApprovalRequestedMsg{
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
