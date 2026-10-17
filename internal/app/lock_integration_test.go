//go:build unix

package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestAcquireLockNonHeldError verifies that when a lock file is unreadable
// (non-held error, not ErrLockHeld), AcquireLock sends a status warning and
// leaves ConcurrentInstance false. This represents the single-instance guard
// being offline due to a real filesystem error.
func TestAcquireLockNonHeldError(t *testing.T) {
	if testing.Short() {
		t.Skip("lock test creates temp state, skipping in short mode")
	}
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")

	wsRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(wsRoot)
	if err != nil {
		t.Fatal(err)
	}

	// Create the session directory and lock file, then make it unreadable.
	sessionDir := session.SessionDir(stateRoot, ws.CanonicalRoot)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatalf("create session dir: %v", err)
	}
	lockPath := filepath.Join(sessionDir, ".lock")
	if err := os.WriteFile(lockPath, nil, 0o000); err != nil {
		t.Fatalf("create lock file: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(lockPath, 0o600) }) // restore permissions for cleanup

	// Create an app and try to acquire the lock.
	a := New(ws, cfg, telemetry.Disabled())
	defer a.Close()

	c := newCollector(1) // collect at least one message
	a.sendFn = c.send

	if err := a.AcquireLock(); err != nil {
		t.Fatalf("AcquireLock: %v", err)
	}

	// ConcurrentInstance should be false (the error is not ErrLockHeld).
	if a.ConcurrentInstance() {
		t.Fatal("ConcurrentInstance should be false for non-held errors")
	}

	// The status warning should have been sent.
	msgs := c.wait(t, 2*time.Second)
	foundWarning := false
	for _, m := range msgs {
		if w, ok := m.(tui.StatusWarningMsg); ok && w.Text == "lock unavailable — single-instance guard off" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("StatusWarningMsg not found in messages: %+v", msgs)
	}

	// Verify the held-lock path does NOT emit the non-held-error warning.
	t.Run("held_lock_no_warning", func(t *testing.T) {
		// Restore lock permissions so a2 can test the held-lock path.
		if err := os.Chmod(lockPath, 0o600); err != nil {
			t.Fatalf("restore lock permissions: %v", err)
		}
		a1 := New(ws, cfg, telemetry.Disabled())
		defer a1.Close()
		if err := a1.AcquireLock(); err != nil {
			t.Fatalf("a1 AcquireLock: %v", err)
		}

		// Now a2 tries to acquire; should hit the held-lock path, not the non-held path.
		a2 := New(ws, cfg, telemetry.Disabled())
		defer a2.Close()
		// The collector is sized beyond what the held path should send, so
		// wait() cannot return on the first message; instead the test lets
		// the per-message delivery goroutines (sendAsyncSequence) settle and
		// inspects everything that arrived.
		c2 := newCollector(1 << 20)
		a2.sendFn = c2.send
		if err := a2.AcquireLock(); err != nil {
			t.Fatalf("a2 AcquireLock: %v", err)
		}

		if !a2.ConcurrentInstance() {
			t.Fatal("held path: ConcurrentInstance should be true")
		}
		const heldText = "another Kirsch is running here"
		const nonHeldText = "lock unavailable \u2014 single-instance guard off"
		deadline := time.Now().Add(2 * time.Second)
		for c2.count() < 1 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		// Settle window: a mutant that also emits the non-held warning does
		// so from a separate goroutine, after the first message.
		time.Sleep(200 * time.Millisecond)
		msgs2 := c2.snapshot()

		foundHeld := false
		for _, m := range msgs2 {
			w, ok := m.(tui.StatusWarningMsg)
			if !ok {
				continue
			}
			switch w.Text {
			case heldText:
				foundHeld = true
			case nonHeldText:
				t.Fatalf("held-lock path emitted the non-held-error warning: %+v", msgs2)
			}
		}
		if !foundHeld {
			t.Fatalf("held-lock path did not send %q: %+v", heldText, msgs2)
		}
	})
}

// TestLockSecondInstanceDegrades verifies that a second instance on the same
// workspace: acquires no lock, sets concurrentInstance to true, shows a warning,
// and has session grants disabled.
//
// This test uses two in-process App instances with the same workspace and state
// root. The first acquires the lock; the second detects contention and degrades
// to fresh-start with grants disabled.
func TestLockSecondInstanceDegrades(t *testing.T) {
	if testing.Short() {
		t.Skip("lock test creates temp state, skipping in short mode")
	}

	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")

	// Set up workspace to the same repo both instances will use.
	wsRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(wsRoot)
	if err != nil {
		t.Fatal(err)
	}

	// === FIRST INSTANCE ===
	// Create the first app, acquire the lock, start a session.
	a1 := New(ws, cfg, telemetry.Disabled())
	defer a1.Close()

	if err := a1.AcquireLock(); err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	if a1.ConcurrentInstance() {
		t.Fatal("first instance should not be concurrent")
	}
	if a1.lock == nil {
		t.Fatal("first instance should hold a lock")
	}

	// Start a session in the first instance.
	if err := a1.StartNewSession(); err != nil {
		t.Fatalf("first StartNewSession: %v", err)
	}

	// === SECOND INSTANCE ===
	// Create a second app with the same workspace and state root.
	a2 := New(ws, cfg, telemetry.Disabled())
	c := newCollector(1) // Just collect at least one message to verify sendFn works
	a2.sendFn = c.send   // Set sendFn before acquiring lock so the warning message is sent
	defer a2.Close()

	// The second instance tries to acquire the lock while the first holds it.
	if err := a2.AcquireLock(); err != nil {
		t.Fatalf("second AcquireLock: %v", err)
	}

	// Verify the second instance detected contention.
	if !a2.ConcurrentInstance() {
		t.Fatal("second instance should set concurrentInstance=true")
	}
	if !a2.sessionGrantsDisabled {
		t.Fatal("second instance should have sessionGrantsDisabled=true")
	}

	// Check that the status warning was sent. We give it a bit of time since
	// SendStatusWarning spawns a goroutine.
	msgs := c.wait(t, 2*time.Second)
	foundWarning := false
	for _, m := range msgs {
		if w, ok := m.(tui.StatusWarningMsg); ok && w.Text == "another Kirsch is running here" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("StatusWarningMsg not found in messages: %+v", msgs)
	}
}

// TestLockReleaseAllowsReacquisition verifies that after the first instance
// releases its lock, a subsequent instance can acquire it. This is a sanity
// check for the lock release mechanism.
func TestLockReleaseAllowsReacquisition(t *testing.T) {
	if testing.Short() {
		t.Skip("lock test creates temp state, skipping in short mode")
	}

	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")

	wsRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(wsRoot)
	if err != nil {
		t.Fatal(err)
	}

	// First instance acquires and closes.
	a1 := New(ws, cfg, telemetry.Disabled())
	if err := a1.AcquireLock(); err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	if a1.ConcurrentInstance() {
		t.Fatal("first instance should not be concurrent")
	}
	a1.Close() // This releases the lock

	// Third instance should now acquire the lock.
	a3 := New(ws, cfg, telemetry.Disabled())
	defer a3.Close()
	if err := a3.AcquireLock(); err != nil {
		t.Fatalf("third AcquireLock: %v", err)
	}
	if a3.ConcurrentInstance() {
		t.Fatal("third instance should not be concurrent after first released lock")
	}
	if a3.lock == nil {
		t.Fatal("third instance should hold a lock")
	}
}

// TestSessionGrantsDisabledPreventsApproval verifies the two places
// sessionGrantsDisabled is consulted: the restoreGrants gate and the Request
// approval gate (via canApproveForSession).
//
// restoreGrants subtests read one real log file holding an
// approval.scope_granted event and compare a.pol.Grants() — the stored grants,
// which is what restoreGrants writes to — not CanGrant, which never reads
// them. Request subtests drive Request itself and read
// ApprovalRequestedMsg.CanApproveForSession from the collector; the
// request_gate_* subtests exercise canApproveForSession directly.
func TestSessionGrantsDisabledPreventsApproval(t *testing.T) {
	if testing.Short() {
		t.Skip("lock test creates temp state, skipping in short mode")
	}

	tmp := t.TempDir()
	stateRoot := filepath.Join(tmp, "state")

	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")

	wsRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(wsRoot)
	if err != nil {
		t.Fatal(err)
	}

	// One log file shared by both restoreGrants subtests.
	logPath := filepath.Join(tmp, "grants.jsonl")
	grantLine := `{"v":1,"type":"approval.scope_granted","action":"run_command","pattern":"echo test"}` + "\n"
	if err := os.WriteFile(logPath, []byte(grantLine), 0o600); err != nil {
		t.Fatalf("write grant log: %v", err)
	}

	t.Run("restoreGrants_disabled", func(t *testing.T) {
		a := New(ws, cfg, telemetry.Disabled())
		defer a.Close()
		a.sessionGrantsDisabled = true

		a.restoreGrants(logPath)

		if got := a.pol.Grants(); len(got) != 0 {
			t.Fatalf("restoreGrants restored grants %q although sessionGrantsDisabled=true", got)
		}
	})

	t.Run("restoreGrants_enabled", func(t *testing.T) {
		a := New(ws, cfg, telemetry.Disabled())
		defer a.Close()

		a.restoreGrants(logPath)

		got := a.pol.Grants()
		if len(got) != 1 || got[0] != "echo test" {
			t.Fatalf("restoreGrants with grants enabled: Grants() = %q, want [\"echo test\"] (positive control for the disabled case)", got)
		}
	})

	// requestFlag drives Request once and returns the CanApproveForSession
	// field of the ApprovalRequestedMsg it sent.
	requestFlag := func(t *testing.T, a *App) bool {
		t.Helper()
		c := newCollector(1)
		a.sendFn = c.send
		const id = 41
		type result struct {
			out ApprovalOutcome
			err error
		}
		done := make(chan result, 1)
		go func() {
			out, err := a.Request(context.Background(), ApprovalRequest{
				ID:          id,
				Description: "run echo test",
				Operation:   policy.OperationCommand,
				Argv:        []string{"echo", "test"},
			})
			done <- result{out, err}
		}()
		msgs := c.wait(t, 2*time.Second)
		a.Resolve(id, ApprovalOutcomeOnce)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Request did not return after Resolve")
		}
		msg, ok := msgs[0].(tui.ApprovalRequestedMsg)
		if !ok {
			t.Fatalf("message is %T, want ApprovalRequestedMsg", msgs[0])
		}
		return msg.CanApproveForSession
	}

	t.Run("request_disabled", func(t *testing.T) {
		a := New(ws, cfg, telemetry.Disabled())
		defer a.Close()
		a.sessionGrantsDisabled = true
		if requestFlag(t, a) {
			t.Fatal("Request sent CanApproveForSession=true although sessionGrantsDisabled=true")
		}
	})

	t.Run("request_enabled", func(t *testing.T) {
		a := New(ws, cfg, telemetry.Disabled())
		defer a.Close()
		if !requestFlag(t, a) {
			t.Fatal("Request sent CanApproveForSession=false although sessionGrantsDisabled=false (positive control)")
		}
	})

	t.Run("request_gate_disabled", func(t *testing.T) {
		a := New(ws, cfg, telemetry.Disabled())
		defer a.Close()
		a.sessionGrantsDisabled = true
		if a.canApproveForSession(policy.OperationCommand, []string{"echo", "test"}) {
			t.Fatal("canApproveForSession should be false when sessionGrantsDisabled=true")
		}
	})

	t.Run("request_gate_enabled", func(t *testing.T) {
		a := New(ws, cfg, telemetry.Disabled())
		defer a.Close()
		if !a.canApproveForSession(policy.OperationCommand, []string{"echo", "test"}) {
			t.Fatal("canApproveForSession should be true when sessionGrantsDisabled=false")
		}
	})
}
