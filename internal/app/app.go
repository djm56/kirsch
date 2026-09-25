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
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/config"
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
	Description string           // Human-readable text for the approval card
	Operation   policy.Operation // The operation type; drives both policy enforcement and the TUI's rendering kind
	Argv        []string         // Command argv for session grant; nil if not applicable
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

// ApprovalOutcome represents a user's decision on an approval.
type ApprovalOutcome uint8

const (
	ApprovalOutcomeOnce ApprovalOutcome = iota
	ApprovalOutcomeSession
	ApprovalOutcomeDeny
	ApprovalOutcomeCancelled
)

// approval tracks a pending approval request.
type approval struct {
	id        int64
	decision  chan ApprovalOutcome // capacity 1; Resolve drops writes after first
	argv      []string             // Command argv for session grant; nil if not applicable
	operation policy.Operation     // The operation type for this approval
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
	return a
}

// Registry exposes the tool registry. Used by `kirsch doctor` in M5 and by
// tests; the TUI never sees it.
func (a *App) Registry() *tool.Registry { return a.reg }

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
// Resolve to be called.
func (a *App) Request(ctx context.Context, req ApprovalRequest) ApprovalOutcome {
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
	// The policy decides based on the operation type, not the caller.
	canApproveForSession := a.pol.CanApproveForSession(req.Operation)

	// Send the approval message to the TUI.
	a.send(tui.ApprovalRequestedMsg{
		ID:                   id,
		Description:          req.Description,
		Kind:                 kindString(req.Operation),
		CanApproveForSession: canApproveForSession,
	})

	// Block on three things:
	// 1. User decision via the buffered channel
	// 2. Turn cancellation (user pressed Esc)
	// 3. Root cancellation (process shutting down)
	select {
	case outcome := <-app.decision:
		a.approveMu.Lock()
		delete(a.approvals, id)
		a.approveMu.Unlock()
		return outcome
	case <-ctx.Done():
		a.approveMu.Lock()
		delete(a.approvals, id)
		a.approveMu.Unlock()
		return ApprovalOutcomeCancelled
	case <-a.rootCtx.Done():
		a.approveMu.Lock()
		delete(a.approvals, id)
		a.approveMu.Unlock()
		return ApprovalOutcomeCancelled
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

	// If approving for session with a command argv, grant it now.
	// Use the real operation from the request, not a bare literal.
	if outcome == ApprovalOutcomeSession && len(app.argv) > 0 {
		if err := a.pol.Grant(app.operation, app.argv); err != nil {
			// Log the rejection silently; the approval still succeeds, but the
			// grant does not record. The TUI will not re-prompt; the tool gets
			// the decision it asked for.
			a.log.Debug("grant rejected", "argv", app.argv, "err", err)
		}
	}

	// Send on the buffered channel. If it already has a value, this is a
	// no-op (the select will have taken the first one).
	select {
	case app.decision <- outcome:
	default:
		// Already resolved; drop the duplicate silently.
	}
}

// describeInput picks the field worth showing on a tool card.
func describeInput(input map[string]any) string {
	for _, key := range []string{"path", "query"} {
		if v, ok := input[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
