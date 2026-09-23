package main

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestApprovalOutcomeConversion verifies the ToAppOutcome conversion function
// handles all enum values correctly, preventing the enum trap. Uses actual app
// constants rather than bare literals to ensure reordering the iota block
// fails a test instead of silently inverting the mapping.
func TestApprovalOutcomeConversion(t *testing.T) {
	tests := []struct {
		tuiOutcome tui.ApprovalOutcome
		appOutcome app.ApprovalOutcome
		name       string
	}{
		{tui.Approved, app.ApprovalOutcomeOnce, "Approved -> Once"},
		{tui.ApprovedSession, app.ApprovalOutcomeSession, "ApprovedSession -> Session"},
		{tui.Rejected, app.ApprovalOutcomeDeny, "Rejected -> Deny"},
		{tui.Cancelled, app.ApprovalOutcomeCancelled, "Cancelled -> Cancelled"},
		{tui.Unresolved, app.ApprovalOutcomeCancelled, "Unresolved -> Cancelled (safety fallback)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tui.ToAppOutcome(tt.tuiOutcome)
			if got != int(tt.appOutcome) {
				t.Errorf("ToAppOutcome(%d) = %d, want %d", tt.tuiOutcome, got, int(tt.appOutcome))
			}
		})
	}
}

// TestResolveApprovalNilCheckWired verifies the ResolveApproval callback is
// properly wired and explicitly checked for nil before calling. This guards
// against accidental removal of the assignment inside wireCallbacks.
func TestResolveApprovalNilCheckWired(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	// Create a real TUI model.
	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Before wiring, ResolveApproval should be nil. This verifies we're testing
	// the wiring, not accidentally relying on a previous assignment.
	if m.ResolveApproval != nil {
		t.Fatal("ResolveApproval should be nil before wireCallbacks")
	}

	// Wire the callbacks.
	wireCallbacks(&m, a, log)

	// After wiring, ResolveApproval must not be nil.
	if m.ResolveApproval == nil {
		t.Fatal("ResolveApproval is nil after wireCallbacks; wiring failed")
	}

	// Verify the wired callback works by dispatching an approval and key through
	// the real Update path.
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result := a.Request(context.Background(), app.ApprovalRequest{
			Description:          "test patch",
			Kind:                 "patch",
			CanApproveForSession: false,
		})
		done <- result
	}()

	// Give the request goroutine a moment to register.
	time.Sleep(10 * time.Millisecond)

	// Dispatch the ApprovalRequestedMsg through Update.
	msg := tui.ApprovalRequestedMsg{
		ID:                   1,
		Description:          "test patch",
		Kind:                 "patch",
		CanApproveForSession: false,
	}
	model, _ := m.Update(msg)
	m = model.(tui.Model)

	// Dispatch Esc through Update to trigger cancellation via the wired callback.
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ = m.Update(keyMsg)
	m = model.(tui.Model)

	// Verify the App received ApprovalOutcomeCancelled through the wired callback.
	select {
	case outcome := <-done:
		if outcome != app.ApprovalOutcomeCancelled {
			t.Errorf("Request returned %d, want %d (ApprovalOutcomeCancelled)",
				outcome, app.ApprovalOutcomeCancelled)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Request() blocked; ResolveApproval callback may not be wired correctly")
	}
}

// TestRunToolCallbackWired verifies that the RunTool callback is properly wired
// through wireCallbacks. Deleting the m.RunTool assignment inside wireCallbacks
// causes this test to fail because RunTool will be nil.
func TestRunToolCallbackWired(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Before wiring, RunTool should be nil.
	if m.RunTool != nil {
		t.Fatal("RunTool should be nil before wireCallbacks")
	}

	// Wire the callbacks through wireCallbacks, which assigns m.RunTool.
	wireCallbacks(&m, a, log)

	// After wiring, RunTool must not be nil. If m.RunTool = ... is deleted
	// from inside wireCallbacks, this assertion will fail.
	if m.RunTool == nil {
		t.Fatal("RunTool is nil after wireCallbacks; wiring assignment is missing")
	}
}

// TestCancelCallbackWired verifies that the Cancel callback is properly wired
// through wireCallbacks. Deleting the m.Cancel assignment inside wireCallbacks
// causes this test to fail because Cancel will be nil.
func TestCancelCallbackWired(t *testing.T) {
	ws, err := workspace.Detect("")
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}

	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
		Session: tui.SessionInfo{
			Project: "test-project",
			Branch:  "main",
			Dirty:   false,
		},
	})

	// Before wiring, Cancel should be nil.
	if m.Cancel != nil {
		t.Fatal("Cancel should be nil before wireCallbacks")
	}

	// Wire the callbacks through wireCallbacks, which assigns m.Cancel.
	wireCallbacks(&m, a, log)

	// After wiring, Cancel must not be nil. If m.Cancel = ... is deleted
	// from inside wireCallbacks, this assertion will fail.
	if m.Cancel == nil {
		t.Fatal("Cancel is nil after wireCallbacks; wiring assignment is missing")
	}
}
