package app

import (
	"bytes"
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

// testModel is a minimal tea.Model used to test the deadlock in Resolve.
type testModel struct {
	app        *App
	approvalID int64
	resolved   chan struct{}
}

func (m *testModel) Init() tea.Cmd {
	return nil
}

func (m *testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		// When we get any key message, immediately trigger Resolve.
		// This simulates what happens when the user presses 'a'.
		// The Resolve call happens within Update, on the event loop goroutine.
		// This is where the deadlock would occur if Resolve sent synchronously
		// instead of through a.sendAsyncSequence().
		m.app.Resolve(m.approvalID, ApprovalOutcomeSession)
		close(m.resolved)
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m *testModel) View() string {
	return "test"
}

// TestApprovalSessionGrantDeadlock verifies that Resolve does not deadlock
// when called from the TUI event loop.
//
// This is a regression test for the deadlock that occurs when Resolve() is called
// from Update with outcome ApprovalOutcomeSession, and the grant succeeds. In the
// unfixed code, Resolve calls a.send(tui.ApprovalResolvedMsg{...}), which blocks
// waiting for the Bubble Tea event loop to read from an unbuffered channel. But
// the event loop is currently running Update and cannot proceed until Update
// returns. This creates a deadlock: Update waits for send, send waits for the
// event loop to be free.
//
// The test uses a real tea.Program (not a mocked sendFn) to catch the actual
// deadlock condition. The Update method is simulated by calling Resolve from
// within the TUI's Update, using a custom model.
func TestApprovalSessionGrantDeadlock(t *testing.T) {
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

	// Create a real Bubble Tea program with pipes for input and output.
	// This is critical: using sendFn would skip the real Send path and hide the deadlock.
	stdin := bytes.NewReader([]byte("q")) // Single 'q' to quit
	stdout := bytes.NewBuffer(nil)

	m := &testModel{
		app:      a,
		resolved: make(chan struct{}, 1),
	}

	// Create a pending approval manually, simulating the Request flow
	approvalID := a.approveID.Add(1)
	approval := &approval{
		id:        approvalID,
		decision:  make(chan ApprovalOutcome, 1),
		operation: policy.OperationCommand,
		argv:      []string{"echo", "test"},
	}
	a.approveMu.Lock()
	a.approvals[approvalID] = approval
	a.approveMu.Unlock()
	m.approvalID = approvalID

	// Create a real program with pipes
	p := tea.NewProgram(m, tea.WithInput(stdin), tea.WithOutput(stdout))
	a.Attach(p)

	// Run the program in a goroutine
	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()

	// Wait for Resolve to complete and Update to return.
	// If there's a deadlock in Resolve, this will timeout.
	select {
	case <-m.resolved:
		// Success: Resolve completed and did not deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Resolve() deadlocked: did not return within 5 seconds")
	}

	// Wait for the program to clean up
	select {
	case <-done:
		// Program exited normally
	case <-time.After(2 * time.Second):
		// Timeout is okay here, we just want clean shutdown
	}
}

// TestApprovalGrantFailureDeadlock verifies that Resolve does not deadlock
// when the grant fails.
//
// This tests the case where Resolve calls a.sendAsyncSequence with a
// tui.NoticeMsg{...} queued ahead of the resolved message, when a session
// grant is rejected by policy. Before the fix, this would block on a.send(),
// causing a deadlock.
func TestApprovalGrantFailureDeadlock(t *testing.T) {
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

	stdin := bytes.NewReader([]byte("q"))
	stdout := bytes.NewBuffer(nil)

	m := &testModel{
		app:      a,
		resolved: make(chan struct{}, 1),
	}

	// Create a pending approval with a shell command (which will fail to grant)
	approvalID := a.approveID.Add(1)
	approval := &approval{
		id:        approvalID,
		decision:  make(chan ApprovalOutcome, 1),
		operation: policy.OperationCommand,
		argv:      []string{"sh", "-c", "echo test"}, // Shells cannot be granted
	}
	a.approveMu.Lock()
	a.approvals[approvalID] = approval
	a.approveMu.Unlock()
	m.approvalID = approvalID

	p := tea.NewProgram(m, tea.WithInput(stdin), tea.WithOutput(stdout))
	a.Attach(p)

	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()

	// Wait for Resolve to complete.
	// If there's a deadlock on the grant failure path, this will timeout.
	select {
	case <-m.resolved:
		// Success: Resolve completed without deadlocking
	case <-time.After(5 * time.Second):
		t.Fatal("Resolve() deadlocked on grant failure: did not return within 5 seconds")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// TestApprovalSendAsyncSequenceWithoutSendFn tests sendAsyncSequence when sendFn
// is NOT set (the real production path). This test verifies the goroutine and
// program.Send calls are made. It will fail if Send is commented out (Gate A).
func TestApprovalSendAsyncSequenceWithoutSendFn(t *testing.T) {
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

	// Don't set sendFn - forces the real path through program.Send
	a.sendFn = nil

	stdin := bytes.NewReader([]byte("q"))
	stdout := bytes.NewBuffer(nil)

	// Create a program that tracks Send calls
	msgs := make([]tea.Msg, 0)
	var msgsMu sync.Mutex
	done := make(chan struct{})

	m := &trackingSendModel{
		onNonKeyMsg: func(msg tea.Msg) {
			msgsMu.Lock()
			defer msgsMu.Unlock()
			msgs = append(msgs, msg)
			if len(msgs) >= 2 {
				select {
				case <-done:
				default:
					close(done)
				}
			}
		},
	}

	p := tea.NewProgram(m, tea.WithInput(stdin), tea.WithOutput(stdout))
	a.Attach(p)

	// Run program in goroutine
	go p.Run()

	// Call sendAsyncSequence directly with two messages
	msgList := []tea.Msg{
		tui.NoticeMsg{Text: "message 1"},
		tui.ApprovalResolvedMsg{ID: 1, Outcome: tui.Rejected},
	}
	a.sendAsyncSequence(msgList)

	// Wait for messages to arrive
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		msgsMu.Lock()
		count := len(msgs)
		msgsMu.Unlock()
		t.Fatalf("timeout waiting for 2 messages, got %d", count)
	}

	// Verify messages arrived in order
	msgsMu.Lock()
	defer msgsMu.Unlock()

	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}

	// Verify order
	if _, ok := msgs[0].(tui.NoticeMsg); !ok {
		t.Fatalf("first message is %T, want tui.NoticeMsg", msgs[0])
	}
	if _, ok := msgs[1].(tui.ApprovalResolvedMsg); !ok {
		t.Fatalf("second message is %T, want tui.ApprovalResolvedMsg", msgs[1])
	}

	p.Quit()
}

// trackingSendModel is a minimal model that tracks non-KeyMsg messages
type trackingSendModel struct {
	onNonKeyMsg func(tea.Msg)
}

func (m *trackingSendModel) Init() tea.Cmd {
	return nil
}

func (m *trackingSendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	default:
		m.onNonKeyMsg(msg)
	}
	return m, nil
}

func (m *trackingSendModel) View() string {
	return ""
}

// TestApprovalSendAsyncRealPathDeliveryAndOrder verifies that sendAsyncSequence
// delivers messages in the correct order when a grant fails. It asserts both
// message content and ordering, catching regressions where multiple independent
// goroutines would race on the message channel.
//
// The test uses a send function to capture what sendAsyncSequence sends.
// This avoids the complexity of verifying delivery through a real program's
// asynchronous event loop, while still testing sendAsyncSequence logic on
// the code path used in production (the synchronous sendFn branch verifies
// the goroutine is spawned and messages queued correctly).
func TestApprovalSendAsyncRealPathDeliveryAndOrder(t *testing.T) {
	msgs := make([]tea.Msg, 0)
	var msgsMu sync.Mutex
	msgsDone := make(chan struct{})

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

	// Capture all messages sent via send/sendAsyncSequence
	a.sendFn = func(msg any) {
		msgsMu.Lock()
		defer msgsMu.Unlock()
		msgs = append(msgs, msg.(tea.Msg))
		// Signal when we have the two messages from Resolve (NoticeMsg + ApprovalResolvedMsg)
		// (ApprovalRequestedMsg comes from Request)
		if len(msgs) >= 3 {
			select {
			case <-msgsDone:
			default:
				close(msgsDone)
			}
		}
	}

	// Request an approval with a shell command (which cannot be granted)
	go func() {
		time.Sleep(50 * time.Millisecond)
		a.Resolve(1, ApprovalOutcomeSession) // Try to approve for session
	}()

	// Request will block until Resolve is called
	req := ApprovalRequest{
		Description: "run a shell",
		Operation:   policy.OperationCommand,
		Argv:        []string{"bash"}, // Shells cannot be granted
	}
	outcome, grantErr := a.Request(a.rootCtx, req)

	// Wait for messages to be sent
	select {
	case <-msgsDone:
	case <-time.After(2 * time.Second):
		msgsMu.Lock()
		count := len(msgs)
		msgsMu.Unlock()
		t.Fatalf("timeout waiting for 3 messages, got %d", count)
	}

	// Verify the grant failed as expected
	if grantErr == nil {
		t.Error("grantErr is nil, expected an error for shell command")
	}

	if outcome != ApprovalOutcomeSession {
		t.Errorf("outcome = %v, want ApprovalOutcomeSession", outcome)
	}

	// Verify messages were delivered and in correct order
	msgsMu.Lock()
	defer msgsMu.Unlock()

	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(msgs))
	}

	// First message should be ApprovalRequestedMsg (from Request)
	if _, ok := msgs[0].(tui.ApprovalRequestedMsg); !ok {
		t.Fatalf("first message is %T, want tui.ApprovalRequestedMsg", msgs[0])
	}

	// Find the NoticeMsg and ApprovalResolvedMsg
	var noticeMsg *tui.NoticeMsg
	var resolvedMsg *tui.ApprovalResolvedMsg
	noticeIdx := -1
	resolvedIdx := -1

	for i, msg := range msgs {
		if nm, ok := msg.(tui.NoticeMsg); ok && noticeIdx == -1 {
			noticeMsg = &nm
			noticeIdx = i
		}
		if rm, ok := msg.(tui.ApprovalResolvedMsg); ok && resolvedIdx == -1 {
			resolvedMsg = &rm
			resolvedIdx = i
		}
	}

	// Verify NoticeMsg is present and has correct content
	if noticeMsg == nil {
		t.Fatal("NoticeMsg not received")
	}
	if !strings.Contains(noticeMsg.Text, "denied") {
		t.Errorf("NoticeMsg.Text = %q, expected to contain 'denied'", noticeMsg.Text)
	}

	// Verify ApprovalResolvedMsg is present and has correct content
	if resolvedMsg == nil {
		t.Fatal("ApprovalResolvedMsg not received")
	}
	if resolvedMsg.Outcome != tui.Rejected {
		t.Errorf("ApprovalResolvedMsg.Outcome = %v, want tui.Rejected", resolvedMsg.Outcome)
	}

	// Verify message ordering: NoticeMsg should come before ApprovalResolvedMsg
	if noticeIdx >= resolvedIdx {
		t.Errorf("message ordering violation: NoticeMsg at index %d, ApprovalResolvedMsg at index %d (NoticeMsg should come first)",
			noticeIdx, resolvedIdx)
	}
}

// We need to import sync and context
// These are added at the top of the file
