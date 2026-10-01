package app

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// driveTUI wires a real App to a real TUI model and pumps messages between
// them, exactly as cmd/kirsch does — but without a tea.Program.
//
// Bubble Tea renders nothing to a non-TTY, so running a real program here
// produces an empty buffer and proves only that the harness is wrong. Feeding
// App's messages straight into Update tests the thing that actually matters:
// that an intent from the TUI reaches a tool, and that the tool's result comes
// back as something the TUI can render.
func driveTUI(t *testing.T, fixture string, typed string, settle time.Duration, expand bool) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	info := a.WorkspaceInfo()
	m := tui.New(tui.Options{
		Version: "0.1.0",
		Caps:    tui.Caps{Colour: false, Unicode: true},
		Session: tui.SessionInfo{Project: info.Project, Branch: info.Branch, Dirty: info.Dirty},
	})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn

	var mu sync.Mutex
	// Unbuffered on purpose. A buffered channel absorbs a message sent from
	// inside Update and so hides the deadlock the real Bubble Tea program hits,
	// where program.Send waits for an event loop that is itself inside Update.
	// With no buffer, a regression stalls the harness and the test times out
	// rather than passing.
	inbox := make(chan tea.Msg)
	a.sendFn = func(msg any) {
		if tm, ok := msg.(tea.Msg); ok {
			inbox <- tm
		}
	}

	apply := func(msg tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}

	apply(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range typed {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if typed != "" {
		apply(tea.KeyMsg{Type: tea.KeyEnter})
	}

	deadline := time.After(settle)
	for {
		select {
		case msg := <-inbox:
			apply(msg)
		case <-deadline:
			mu.Lock()
			defer mu.Unlock()
			if expand {
				// A tool card is collapsed until Enter (ui-spec §3.3), so the
				// content is only on screen after focusing the transcript and
				// expanding the card the result selected.
				next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
				m = next.(tui.Model)
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = next.(tui.Model)
			}
			return m.View()
		}
	}
}

// TestDebugCommandReadsRealFile is Task 9's acceptance criterion: from a real
// repository, a debug command reads a file and renders it as a tool card.
func TestDebugCommandReadsRealFile(t *testing.T) {
	// Collapsed first: the card and its summary must be on screen.
	out := driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, false)
	for _, want := range []string{"read_file", "go.mod"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in the collapsed card:\n%s", want, out)
		}
	}
	// Expanded: the file's contents, line-numbered.
	out = driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, true)
	for _, want := range []string{"module example.com/calc"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

func TestDebugCommandSearchesRealRepo(t *testing.T) {
	out := driveTUI(t, "repo-go-module", "/search Divide", 1500*time.Millisecond, true)
	for _, want := range []string{"search_code", "Divide", "calc/divide.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

// TestDeniedPathRendersAsAToolCard is the other half of Task 9's check: a
// denylisted path must render an error, not panic and not silently do nothing.
func TestDeniedPathRendersAsAToolCard(t *testing.T) {
	out := driveTUI(t, "repo-small", "/read .env", 1500*time.Millisecond, false)
	if !strings.Contains(out, "read_file") {
		t.Errorf("no tool card rendered:\n%s", out)
	}
	if !strings.Contains(out, "denied") && !strings.Contains(out, "never readable") {
		t.Errorf("refusal not shown to the user:\n%s", out)
	}
	if strings.Contains(out, "hunter2") {
		t.Fatalf(".env contents reached the screen:\n%s", out)
	}
}

func TestHeaderShowsRealWorkspace(t *testing.T) {
	out := driveTUI(t, "repo-node", "", 400*time.Millisecond, false)
	if !strings.Contains(out, "repo-node") {
		t.Errorf("header does not show the real project name:\n%s", out)
	}
}

// TestCancelledToolRendersCancelledGlyph is the last of Task 8's checks:
// cancelling returns control and the card shows ⊘ rather than hanging on the
// spinner or vanishing.
func TestCancelledToolRendersCancelledGlyph(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	slow := &slowTool{started: make(chan struct{})}
	a.reg.Register(slow)

	m := tui.New(tui.Options{Version: "0.1.0", Caps: tui.Caps{Unicode: true}})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn

	inbox := make(chan tea.Msg)
	a.sendFn = func(msg any) { inbox <- msg.(tea.Msg) }

	apply := func(msg tea.Msg) {
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}
	apply(tea.WindowSizeMsg{Width: 100, Height: 30})

	m.RunTool("slow", map[string]any{})
	apply(<-inbox) // ToolStartedMsg

	select {
	case <-slow.started:
	case <-time.After(2 * time.Second):
		t.Fatal("tool never started")
	}
	if !strings.Contains(m.View(), "◐") {
		t.Errorf("running card does not show the running badge:\n%s", m.View())
	}

	begin := time.Now()
	apply(tea.KeyMsg{Type: tea.KeyEsc}) // Esc cancels the turn
	apply(<-inbox)                      // ToolCompletedMsg
	elapsed := time.Since(begin)

	if elapsed > time.Second {
		t.Errorf("cancel took %v; ui-spec §11 targets under 1s", elapsed)
	}
	view := m.View()
	if !strings.Contains(view, "⊘") {
		t.Errorf("cancelled card does not show ⊘:\n%s", view)
	}
	if strings.Contains(view, "◐") {
		t.Errorf("card still shows the running badge after cancellation:\n%s", view)
	}
}

// driveTUIWithApprovalCapture drives the TUI and captures approval messages.
// It returns the rendered output, the first ApprovalRequestedMsg received (or nil),
// and the TUI model for further interaction.
func driveTUIWithApprovalCapture(t *testing.T, fixture string, typed string, settle time.Duration) (string, *tui.ApprovalRequestedMsg, tui.Model) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	info := a.WorkspaceInfo()
	m := tui.New(tui.Options{
		Version: "0.1.0",
		Caps:    tui.Caps{Colour: false, Unicode: true},
		Session: tui.SessionInfo{Project: info.Project, Branch: info.Branch, Dirty: info.Dirty},
	})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn

	// mu guards capturedApproval and nothing else. It is written by the
	// goroutine that calls sendFn and read by this one. The model m is touched
	// only by this goroutine, so it needs no lock, and apply takes none.
	// No code path holds mu while calling apply, so the helper cannot
	// self-deadlock: the only critical sections are the two short ones below,
	// and neither calls out to anything that locks.
	var mu sync.Mutex
	inbox := make(chan tea.Msg, 100) // Buffer to prevent goroutine blocks when test finishes
	var capturedApproval *tui.ApprovalRequestedMsg
	done := make(chan struct{})
	a.sendFn = func(msg any) {
		// Give up sending if the test is done (prevents indefinite block on unbuffered send)
		select {
		case <-done:
			return
		default:
		}

		if tm, ok := msg.(tea.Msg); ok {
			// Capture ApprovalRequestedMsg on first receipt.
			if approval, ok := tm.(tui.ApprovalRequestedMsg); ok {
				mu.Lock()
				if capturedApproval == nil {
					capturedApproval = &approval
				}
				mu.Unlock()
			}
			// Non-blocking send; if buffer is full, drop the message gracefully
			select {
			case inbox <- tm:
			case <-done:
			default:
				// Buffer full and test is moving on; drop message
			}
		}
	}

	apply := func(msg tea.Msg) {
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}

	apply(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range typed {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if typed != "" {
		apply(tea.KeyMsg{Type: tea.KeyEnter})
	}

	pumpUntil(time.After(settle), inbox, apply)
	close(done) // Signal to sendFn to stop trying to send

	// Copy under the lock: a sendFn call already past its done check may still
	// be writing capturedApproval.
	mu.Lock()
	var captured *tui.ApprovalRequestedMsg
	if capturedApproval != nil {
		c := *capturedApproval
		captured = &c
	}
	mu.Unlock()
	return m.View(), captured, m
}

// pumpUntil applies every message arriving on inbox until deadline fires, then
// applies whatever is still buffered and returns. It holds no lock and takes
// none itself, so whatever apply does cannot deadlock against it.
func pumpUntil(deadline <-chan time.Time, inbox <-chan tea.Msg, apply func(tea.Msg)) {
	for {
		select {
		case msg := <-inbox:
			apply(msg)
		case <-deadline:
			for {
				select {
				case msg := <-inbox:
					apply(msg)
				default:
					return
				}
			}
		}
	}
}

// TestRunCatApprovalRequestWithoutGrant verifies that /run cat requests approval
// when no session grant is present. This tests the full real path: TUI dispatch →
// App.RunTool → registry → RunCommand.Invoke → Policy → real Approver.
func TestRunCatApprovalRequestWithoutGrant(t *testing.T) {
	view, approvalMsg, _ := driveTUIWithApprovalCapture(t, "repo-small", "/run cat", 1500*time.Millisecond)

	// Verify an approval message was received
	if approvalMsg == nil {
		t.Fatal("no ApprovalRequestedMsg received; approval gate was not reached")
	}

	// Verify the approval request is for the cat command
	if approvalMsg.Kind != "command" {
		t.Errorf("approval kind: expected 'command', got %q", approvalMsg.Kind)
	}

	if len(approvalMsg.Argv) != 1 || approvalMsg.Argv[0] != "cat" {
		t.Errorf("approval argv: expected ['cat'], got %v", approvalMsg.Argv)
	}

	if !strings.Contains(approvalMsg.Description, "cat") {
		t.Errorf("approval description should mention cat, got %q", approvalMsg.Description)
	}

	// Verify the TUI shows an approval card (not a tool result)
	if !strings.Contains(view, "approval") {
		t.Errorf("approval card not visible in TUI:\n%s", view)
	}
}

// TestRunCatWithSessionGrant verifies that /run cat does NOT request approval
// when a session grant covers it.
func TestRunCatWithSessionGrant(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	info := a.WorkspaceInfo()
	m := tui.New(tui.Options{
		Version: "0.1.0",
		Caps:    tui.Caps{Colour: false, Unicode: true},
		Session: tui.SessionInfo{Project: info.Project, Branch: info.Branch, Dirty: info.Dirty},
	})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn

	var mu sync.Mutex
	inbox := make(chan tea.Msg)
	var sawApproval bool
	a.sendFn = func(msg any) {
		if tm, ok := msg.(tea.Msg); ok {
			if _, ok := tm.(tui.ApprovalRequestedMsg); ok {
				sawApproval = true
			}
			inbox <- tm
		}
	}

	apply := func(msg tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}

	// Grant "cat" to the session
	if err := a.pol.Grant(policy.OperationCommand, []string{"cat"}); err != nil {
		t.Fatalf("failed to grant cat: %v", err)
	}

	apply(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range "/run cat" {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	apply(tea.KeyMsg{Type: tea.KeyEnter})

	deadline := time.After(1500 * time.Millisecond)
	for {
		select {
		case msg := <-inbox:
			apply(msg)
		case <-deadline:
			mu.Lock()
			defer mu.Unlock()
			if sawApproval {
				t.Fatal("ApprovalRequestedMsg received; grant should have covered cat")
			}
			// With grant, we should see a tool card, not an approval card
			view := m.View()
			if !strings.Contains(view, "run_command") {
				t.Errorf("tool card not visible; cat may not have executed:\n%s", view)
			}
			return
		}
	}
}

// TestPumpUntilDrainsBufferedMessagesAfterDeadline checks that pumpUntil
// applies every message already buffered in inbox once the deadline has
// fired, and returns without hanging. It pre-fills 50 messages and passes an
// already-fired deadline, then asserts all 50 were applied. A pump that
// returned on the deadline without draining would apply far fewer.
func TestPumpUntilDrainsBufferedMessagesAfterDeadline(t *testing.T) {
	inbox := make(chan tea.Msg, 100)
	for i := 0; i < 50; i++ {
		inbox <- tui.NoticeMsg{Text: "buffered"}
	}
	fired := make(chan time.Time, 1)
	fired <- time.Now()

	var mu sync.Mutex
	applied := 0
	apply := func(tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		applied++
	}

	finished := make(chan struct{})
	go func() {
		pumpUntil(fired, inbox, apply)
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("pumpUntil hung with messages buffered at the deadline")
	}
	mu.Lock()
	defer mu.Unlock()
	if applied != 50 {
		t.Errorf("applied %d messages, want 50", applied)
	}
}
