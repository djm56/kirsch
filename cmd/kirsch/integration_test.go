package main

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestIntegrationApprovalFlow tests the complete approval flow with a real TUI
// model and real App, verifying that the ResolveApproval callback is properly
// wired and drives the three approval decision flows: approve once, reject, and
// approve for session.
func TestIntegrationApprovalFlow(t *testing.T) {
	tests := []struct {
		name           string
		decision       tui.ApprovalOutcome
		wantAppOutcome app.ApprovalOutcome
		description    string
	}{
		{
			name:           "approve once",
			decision:       tui.Approved,
			wantAppOutcome: app.ApprovalOutcomeOnce,
			description:    "approve this once",
		},
		{
			name:           "reject",
			decision:       tui.Rejected,
			wantAppOutcome: app.ApprovalOutcomeDeny,
			description:    "reject this proposal",
		},
		{
			name:           "approve for session",
			decision:       tui.ApprovedSession,
			wantAppOutcome: app.ApprovalOutcomeSession,
			description:    "approve for this session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a real App with the production callbacks.
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

			// Wire the callbacks exactly as run() does in main.go via wireCallbacks().
			wireCallbacks(&m, a, log)

			// Run an approval request in a goroutine.
			done := make(chan app.ApprovalOutcome, 1)
			go func() {
				result := a.Request(context.Background(), app.ApprovalRequest{
					Description:          tt.description,
					Kind:                 "patch",
					CanApproveForSession: tt.decision == tui.ApprovedSession,
					Argv:                 []string{"go", "test", "./..."},
				})
				done <- result
			}()

			// Give the request goroutine time to set up. The first Request() call
			// will be assigned ID 1.
			time.Sleep(10 * time.Millisecond)

			// Simulate the user decision via the Model callback.
			m.ResolveApproval(1, tt.decision)

			// Wait for the approval goroutine to return with the decision.
			select {
			case outcome := <-done:
				if outcome != tt.wantAppOutcome {
					t.Errorf("Request returned %d, want %d", outcome, tt.wantAppOutcome)
				}
			case <-time.After(1 * time.Second):
				t.Fatal("Request() blocked; callback may not be wired correctly")
			}
		})
	}
}

// TestIntegrationCancelledOutcome tests that escape/Ctrl+C during an approval
// resolves as Cancelled (not Rejected) and flows correctly through the real
// dispatch path. This exercises the new Cancelled outcome that distinguishes
// turn cancellation from explicit rejection.
func TestIntegrationCancelledOutcome(t *testing.T) {
	// Set up a real App.
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

	// Wire the callbacks exactly as run() does in main.go via wireCallbacks().
	wireCallbacks(&m, a, log)

	// Run an approval request.
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result := a.Request(context.Background(), app.ApprovalRequest{
			Description:          "test patch",
			Kind:                 "patch",
			CanApproveForSession: false,
		})
		done <- result
	}()

	// Give the request goroutine time to set up.
	time.Sleep(10 * time.Millisecond)

	// Simulate Esc/Ctrl+C by resolving as Cancelled.
	m.ResolveApproval(1, tui.Cancelled)

	// Verify the outcome reaches the App as Cancelled, not Rejected.
	select {
	case outcome := <-done:
		if outcome != app.ApprovalOutcomeCancelled {
			t.Errorf("Request returned %d, want %d (ApprovalOutcomeCancelled)",
				outcome, app.ApprovalOutcomeCancelled)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Request() blocked; Cancelled outcome may not be flowing through")
	}
}

// TestIntegrationCancellationWithinOneSecond tests that a turn cancelled with
// an approval pending returns within one second, with no parked goroutines.
func TestIntegrationCancellationWithinOneSecond(t *testing.T) {
	// Set up a real App.
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

	// Measure goroutine count before the test to verify no leaks.
	baseGoroutines := runtime.NumGoroutine()

	// Run an approval request with a context that we will cancel.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result := a.Request(ctx, app.ApprovalRequest{
			Description:          "test patch",
			Kind:                 "patch",
			CanApproveForSession: false,
		})
		done <- result
	}()

	// Give the request goroutine a moment to register the approval.
	// We use a short sleep here to let the goroutine reach the blocking point.
	time.Sleep(10 * time.Millisecond)

	// Measure the time before cancellation.
	cancelTime := time.Now()

	// Cancel the turn.
	cancel()

	// Measure the time until Request returns.
	select {
	case outcome := <-done:
		elapsed := time.Since(cancelTime)

		// Verify the outcome is Cancelled.
		if outcome != app.ApprovalOutcomeCancelled {
			t.Errorf("Request returned %d, want %d (ApprovalOutcomeCancelled)",
				outcome, app.ApprovalOutcomeCancelled)
		}

		// Verify it returned within one second.
		if elapsed > 1*time.Second {
			t.Errorf("Cancellation took %v, want under 1 second", elapsed)
		}

		// Give goroutines a moment to clean up, then verify no stragglers.
		time.Sleep(10 * time.Millisecond)
		finalGoroutines := runtime.NumGoroutine()

		// Goroutine count should return to baseline (or very close).
		// Allow a small tolerance for runtime goroutines.
		if finalGoroutines > baseGoroutines+2 {
			t.Errorf("Goroutine leak: started at %d, ended at %d (delta: +%d)",
				baseGoroutines, finalGoroutines, finalGoroutines-baseGoroutines)
		}

	case <-time.After(1500 * time.Millisecond):
		t.Fatal("Cancellation did not return within 1.5 seconds; likely blocked or parked")
	}
}

// TestIntegrationResolveTwiceIsHarmless tests that resolving the same approval
// twice through the real path (Resolve -> buffered channel) is harmless and
// does not cause a goroutine to hang.
func TestIntegrationResolveTwiceIsHarmless(t *testing.T) {
	// Set up a real App.
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

	// Run an approval request.
	done := make(chan app.ApprovalOutcome, 1)
	go func() {
		result := a.Request(context.Background(), app.ApprovalRequest{
			Description:          "test patch",
			Kind:                 "patch",
			CanApproveForSession: false,
		})
		done <- result
	}()

	// Give the request goroutine time to register. We poll with a short sleep
	// since we can only observe the app's behavior indirectly through the channel.
	time.Sleep(10 * time.Millisecond)

	// Resolve with Approved.
	a.Resolve(1, app.ApprovalOutcomeOnce)

	// Wait for the result.
	select {
	case outcome := <-done:
		if outcome != app.ApprovalOutcomeOnce {
			t.Errorf("First resolve returned %d, want %d", outcome, app.ApprovalOutcomeOnce)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("First resolve blocked")
	}

	// Resolve again with a different outcome. This should be harmless.
	a.Resolve(1, app.ApprovalOutcomeSession)

	// Give time to see if anything breaks.
	time.Sleep(10 * time.Millisecond)

	// If we get here without hanging, the test passes.
}
