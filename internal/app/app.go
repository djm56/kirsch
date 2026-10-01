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
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

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
			if line.Prefix == '\\' {
				prefixedLine = `\ ` + line.Content
			} else {
				prefixedLine = string(line.Prefix) + line.Content
			}
			diffLines = append(diffLines, prefixedLine)

			// Count added and removed lines.
			if line.Prefix == '+' {
				added++
			} else if line.Prefix == '-' {
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
}

// New builds an App. The context hierarchy is rooted here: rootCtx outlives
// everything, a turn context is derived per turn, and cancelling a turn never
// touches the root.
func New(ws *workspace.Workspace, cfg config.Config, log *telemetry.Logger) *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{
		ws:         ws,
		cfg:        cfg,
		log:        log,
		reg:        tool.NewRegistry(),
		pol:        policy.New(),
		rootCtx:    ctx,
		rootCancel: cancel,
		approvals:  make(map[int64]*approval),
	}
	a.reg.Register(&tool.ReadFile{WS: ws})
	a.reg.Register(&tool.ListFiles{WS: ws})
	a.reg.Register(&tool.SearchCode{WS: ws})
	a.reg.Register(&tool.GitStatus{WS: ws})
	a.reg.Register(&tool.GitDiff{WS: ws})
	a.reg.Register(&tool.ApplyPatch{WS: ws, Approver: &approvalAdapter{app: a}})
	a.reg.Register(&tool.RunCommand{WS: ws, Config: &a.cfg, Policy: a.pol, Approver: &approvalAdapter{app: a}})
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
	id := a.approveID.Add(1)

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

	if req.Operation == policy.OperationPatch {
		subject, detail = formatPatchDisplay(req.Changes)
		diffFilename, diffLines, diffAdded, diffRemoved = computeDiffDisplay(req.Changes)
	} else if req.Operation == policy.OperationCommand {
		subject, grantScope = formatCommandDisplay(req.Argv)
		detail = formatCommandDetail(req.Description, req.Argv)
	}

	// Send the approval message to the TUI.
	a.send(tui.ApprovalRequestedMsg{
		ID:                   id,
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
	// If the grant fails, the actual outcome is Rejected, not the user's choice.
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
			// Grant failed: the actual outcome is Rejected, not ApprovalOutcomeSession
			confirmedOutcome = ApprovalOutcomeDeny
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
	// Convert tool.ApprovalRequest to app.ApprovalRequest, carrying over the Argv and Changes
	appReq := ApprovalRequest{
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
	// argv comes from shellSplit in update.go, which returns []string.
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
