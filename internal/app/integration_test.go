package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
func driveTUI(t *testing.T, fixture string, typed string, settle time.Duration, collapse bool) string {
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
			// Select the card with Shift+Up
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
			m = next.(tui.Model)
			// If collapse=true, press Enter to collapse the card to head only
			if collapse {
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
	out := driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, true)
	for _, want := range []string{"read_file", "go.mod"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in the collapsed card:\n%s", want, out)
		}
	}
	// Expanded: the file's contents, line-numbered.
	out = driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, false)
	for _, want := range []string{"module example.com/calc"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

// TestReadFileHasNoEmptyLastRow verifies that tool output boxes do not render
// an empty row before the bottom border, even though the real producer writes
// content ending in \n.
func TestReadFileHasNoEmptyLastRow(t *testing.T) {
	out := driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, false)

	// Locate the read_file card by its head row: the row containing both
	// read_file and go.mod.
	rows := strings.Split(out, "\n")
	var cardHeadIdx int
	for i, row := range rows {
		if strings.Contains(row, "read_file") && strings.Contains(row, "go.mod") {
			cardHeadIdx = i
			break
		}
	}
	if cardHeadIdx == 0 && (len(rows) == 0 || !strings.Contains(rows[0], "read_file")) {
		t.Fatalf("could not find read_file card head in output:\n%s", out)
	}

	// From that row, walk down to the first row containing └ (or the ASCII +).
	// That row is this card's bottom border.
	var cardBottomIdx int
	for i := cardHeadIdx + 1; i < len(rows); i++ {
		if strings.Contains(rows[i], "└") || strings.Contains(rows[i], "+") {
			cardBottomIdx = i
			break
		}
	}
	if cardBottomIdx == 0 {
		t.Fatalf("could not find read_file card bottom border from row %d:\n%s", cardHeadIdx, out)
	}

	// Assert that the row directly above the bottom border contains the last
	// line of go.mod (which is 3 lines long, so last line is "go 1.25").
	// Read the fixture file to get the expected last line.
	fixturePath := filepath.Join("..", "..", "testdata", "repo-go-module", "go.mod")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("could not read fixture file: %v", err)
	}
	lines := strings.Split(string(fixtureBytes), "\n")
	// Remove the trailing empty string if the file ends with \n
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		t.Fatalf("fixture file is empty")
	}
	lastLine := lines[len(lines)-1]

	// The row directly above the bottom border should contain the last line of the file
	contentRowAboveBorder := rows[cardBottomIdx-1]
	if !strings.Contains(contentRowAboveBorder, lastLine) {
		t.Errorf("tool card content row above bottom border does not contain %q:\n%s", lastLine, contentRowAboveBorder)
	}
}

func TestDebugCommandSearchesRealRepo(t *testing.T) {
	out := driveTUI(t, "repo-go-module", "/search Divide", 1500*time.Millisecond, false)
	for _, want := range []string{"search_code", "Divide", "calc/divide.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

// TestReadFileDisplaySummaryFormat verifies that the read_file summary shows
// "lines N-M" and the file path appears exactly once in the card head.
func TestReadFileDisplaySummaryFormat(t *testing.T) {
	out := driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, false)

	// Find the read_file card head
	rows := strings.Split(out, "\n")
	var cardHead string
	for _, row := range rows {
		if strings.Contains(row, "read_file") && strings.Contains(row, "go.mod") {
			cardHead = row
			break
		}
	}

	if cardHead == "" {
		t.Fatalf("could not find read_file card head in output:\n%s", out)
	}

	// The summary in the card head should match the "lines N-M" format
	lineSummaryRegex := regexp.MustCompile(`lines 1-\d+`)
	if !lineSummaryRegex.MatchString(cardHead) {
		t.Errorf("card head summary should match 'lines 1-\\d+' format:\n%s", cardHead)
	}
}

// TestDeniedPathRendersAsAToolCard is the other half of Task 9's check: a
// denylisted path must render an error, not panic and not silently do nothing.
func TestDeniedPathRendersAsAToolCard(t *testing.T) {
	out := driveTUI(t, "repo-small", "/read .env", 1500*time.Millisecond, true)
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
	out := driveTUI(t, "repo-node", "", 400*time.Millisecond, true)
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

// TestToolCardPreviewAppLevel verifies that the real /read tool shows a preview
// of 10 lines by default. It reads a fixture file and asserts that lines 1–10
// are visible, line 11 is absent, and the marker shows the correct count.
func TestToolCardPreviewAppLevel(t *testing.T) {
	out := driveTUI(t, "repo-go-module", "/read calc/divide_test.go", 1500*time.Millisecond, false)

	// Read the fixture to get the line count
	fixturePath := filepath.Join("..", "..", "testdata", "repo-go-module", "calc", "divide_test.go")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("could not read fixture file: %v", err)
	}
	lines := tui.SanitizeLines(string(fixtureBytes))
	// SanitizeLines may add trailing empty line; trim it if fixture ended with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lineCount := len(lines)

	// Verify lines 1-10 are visible (using sanitized form)
	for i := 1; i <= 10 && i <= lineCount; i++ {
		if !strings.Contains(out, tui.Sanitize(lines[i-1])) {
			t.Errorf("preview should show line %d; view:\n%s", i, out)
		}
	}

	// Verify line 11 is NOT visible (if it exists, using same normalized form)
	if lineCount >= 11 {
		if strings.Contains(out, tui.Sanitize(lines[10])) {
			t.Errorf("line 11 should not be visible in preview; view:\n%s", out)
		}
	}

	// Verify the exact marker text with the correct line count
	expectedMarker := fmt.Sprintf("‹10 of %d lines — d full output · Enter collapse›", lineCount)
	if !strings.Contains(out, expectedMarker) {
		t.Errorf("marker should be %q; view:\n%s", expectedMarker, out)
	}
}

// TestOneCardAppCtxLink verifies that approval messages are linked to tool messages via context.
func TestOneCardAppCtxLink(t *testing.T) {
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
	inbox := make(chan tea.Msg, 100)
	var capturedTool *tui.ToolStartedMsg
	var capturedApproval *tui.ApprovalRequestedMsg
	done := make(chan struct{})

	a.sendFn = func(msg any) {
		select {
		case <-done:
			return
		default:
		}

		if tm, ok := msg.(tea.Msg); ok {
			// Capture ToolStartedMsg
			if tool, ok := tm.(tui.ToolStartedMsg); ok {
				mu.Lock()
				if capturedTool == nil {
					capturedTool = &tool
				}
				mu.Unlock()
			}
			// Capture ApprovalRequestedMsg
			if approval, ok := tm.(tui.ApprovalRequestedMsg); ok {
				mu.Lock()
				if capturedApproval == nil {
					capturedApproval = &approval
				}
				mu.Unlock()
			}
			select {
			case inbox <- tm:
			case <-done:
			default:
			}
		}
	}

	apply := func(msg tea.Msg) {
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}

	apply(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range "/run cat" {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	apply(tea.KeyMsg{Type: tea.KeyEnter})

	pumpUntil(time.After(1500*time.Millisecond), inbox, apply)
	close(done)

	// Verify both messages were captured
	mu.Lock()
	tool := capturedTool
	approval := capturedApproval
	mu.Unlock()

	if tool == nil {
		t.Fatal("no ToolStartedMsg received")
	}
	if approval == nil {
		t.Fatal("no ApprovalRequestedMsg received")
	}

	// Assert ToolID is non-zero and equals tool ID
	if tool.ID == 0 {
		t.Error("tool ID should be non-zero")
	}
	if approval.ToolID == 0 {
		t.Error("approval ToolID should be non-zero")
	}
	if approval.ToolID != tool.ID {
		t.Errorf("approval.ToolID = %d, want %d (tool.ID)", approval.ToolID, tool.ID)
	}

	// Approve the tool with 'a' and check the final view
	apply(key('a'))
	apply(tui.ApprovalResolvedMsg{ID: approval.ID, Outcome: tui.ApprovedSession})
	apply(tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()

	// Verify the final view has exactly one line containing both "run_command" and " · approved"
	lines := strings.Split(view, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "run_command") && strings.Contains(line, "· approved") {
			count++
		}
	}

	if count == 0 {
		t.Error("view should contain a line with both 'run_command' and '· approved'")
	}
	if count > 1 {
		t.Errorf("view should have exactly one line with 'run_command' and '· approved', got %d", count)
	}

	// Verify no line contains "✓ approved" (folded approval)
	for _, line := range lines {
		if strings.Contains(line, "✓ approved") {
			t.Errorf("no line should contain '✓ approved' for folded approval: %s", line)
		}
	}
}

// Helper to create a KeyMsg for a rune (same as in view_smoke_test.go)
func key(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
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
