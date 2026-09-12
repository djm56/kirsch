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
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// App owns the root context and the wiring between the TUI and the tools.
type App struct {
	ws  *workspace.Workspace
	cfg config.Config
	log *telemetry.Logger
	reg *tool.Registry

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
		rootCtx:    ctx,
		rootCancel: cancel,
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
