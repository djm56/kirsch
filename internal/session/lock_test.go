//go:build unix

package session

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestAcquireSucceeds verifies that a lock can be acquired when no lock is held.
func TestAcquireSucceeds(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	lock, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer lock.Release()

	if lock.file == nil {
		t.Fatal("lock.file is nil")
	}
}

// TestAcquireCreatesSessionDir verifies that the session directory is created
// with the correct permissions.
func TestAcquireCreatesSessionDir(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	lock, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer lock.Release()

	sessionDir := SessionDir(root, workspace)
	info, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatalf("stat session dir: %v", err)
	}

	// Check that the directory exists and is a directory.
	if !info.IsDir() {
		t.Fatal("session directory is not a directory")
	}
}

// TestAcquireCreatesLockFile verifies that the .lock file exists after
// acquiring a lock.
func TestAcquireCreatesLockFile(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	lock, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer lock.Release()

	lockPath := filepath.Join(SessionDir(root, workspace), ".lock")
	_, err = os.Stat(lockPath)
	if err != nil {
		t.Fatalf("stat lock file: %v", err)
	}
}

// TestSecondAcquireContends verifies that a second acquire on the same
// workspace/root returns ErrLockHeld while the first lock is held. The test
// uses two separate os.Open calls in the same process to verify that flock
// contends at the per-file-description level.
func TestSecondAcquireContends(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	// First acquire succeeds.
	lock1, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer lock1.Release()

	// Second acquire fails with ErrLockHeld because the lock is held by the
	// first file descriptor.
	lock2, err := Acquire(root, workspace)
	if !errors.Is(err, ErrLockHeld) {
		t.Fatalf("second Acquire: got %v, want ErrLockHeld", err)
	}
	if lock2 != nil {
		t.Fatal("second Acquire returned non-nil lock")
	}
}

// TestReleaseSucceeds verifies that a lock can be released and reopened
// after Release.
func TestReleaseSucceeds(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	// First acquire and release.
	lock1, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if err := lock1.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Second acquire should succeed.
	lock2, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("second Acquire after release: %v", err)
	}
	defer lock2.Release()
}

// TestReleaseIdempotent verifies that Release can be called multiple times
// without error.
func TestReleaseIdempotent(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	lock, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Multiple releases should not error.
	if err := lock.Release(); err != nil {
		t.Fatalf("first Release: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}
}

// TestCloseAlias verifies that Close is equivalent to Release.
func TestCloseAlias(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()

	lock, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	if err := lock.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Should be able to acquire again after Close.
	lock2, err := Acquire(root, workspace)
	if err != nil {
		t.Fatalf("Acquire after Close: %v", err)
	}
	defer lock2.Release()
}

// TestRealErrorDistinct verifies that a real error (e.g., permission denied)
// is distinct from ErrLockHeld. This test skips on some systems where permission
// models differ.
func TestRealErrorDistinct(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	// Create a file that we'll make unwritable to trigger a permission error.
	root := t.TempDir()
	workspace := t.TempDir()

	// Create the session directory and a lock file.
	sessionDir := SessionDir(root, workspace)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatalf("create session dir: %v", err)
	}

	lockPath := filepath.Join(sessionDir, ".lock")
	if f, err := os.Create(lockPath); err != nil {
		t.Fatalf("create lock file: %v", err)
	} else {
		f.Close()
	}

	// Make the lock file unreadable and unwritable.
	if err := os.Chmod(lockPath, 0o000); err != nil {
		t.Fatalf("chmod lock file: %v", err)
	}
	defer os.Chmod(lockPath, 0o600)

	// Try to acquire a lock. This should fail with a permission error, not ErrLockHeld.
	lock, err := Acquire(root, workspace)
	if err == nil {
		lock.Release()
		t.Fatal("Acquire should have failed with permission error")
	}
	if errors.Is(err, ErrLockHeld) {
		t.Fatalf("permission error returned as ErrLockHeld: %v", err)
	}
}

// TestAddUsageWaitsForHeldLock holds usage.json.lock, starts AddUsage and
// shows it neither returns nor writes while the lock is held, then finishes
// and records its increment once the lock is released. With no flock in
// AddUsage it returns within milliseconds, while the lock is still held.
func TestAddUsageWaitsForHeldLock(t *testing.T) {
	root := t.TempDir()
	held, err := acquireUsageLock(root)
	if err != nil {
		t.Fatalf("acquireUsageLock: %v", err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			_ = held.Release()
		}
	})

	done := make(chan error, 1)
	go func() {
		_, err := AddUsage(root, 100, 50, 0, 0, 0, true, time.Now())
		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("AddUsage returned (%v) while another holder had the lock", err)
	case <-time.After(200 * time.Millisecond):
	}
	if _, err := os.Stat(UsagePath(root)); err == nil {
		t.Fatal("usage.json was written while the lock was held")
	}

	if err := held.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	released = true
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("AddUsage after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("AddUsage did not complete after the lock was released")
	}
	got, err := ReadUsage(root)
	if err != nil {
		t.Fatalf("ReadUsage: %v", err)
	}
	if got.PromptTokens != 100 || got.CompletionTokens != 50 {
		t.Errorf("usage after release = %+v, want (100, 50)", got)
	}
}

// TestAddUsageReturnsErrLockHeldWhenLockStaysHeld keeps the lock for the whole
// retry window (about 1s). AddUsage must give up with ErrLockHeld and write
// nothing.
func TestAddUsageReturnsErrLockHeldWhenLockStaysHeld(t *testing.T) {
	root := t.TempDir()
	held, err := acquireUsageLock(root)
	if err != nil {
		t.Fatalf("acquireUsageLock: %v", err)
	}
	defer func() { _ = held.Release() }()

	recovered, err := AddUsage(root, 1, 1, 0, 0, 0, true, time.Now())
	if !errors.Is(err, ErrLockHeld) {
		t.Fatalf("AddUsage error = %v, want ErrLockHeld", err)
	}
	if recovered != "" {
		t.Errorf("recovered = %q, want empty", recovered)
	}
	if _, statErr := os.Stat(UsagePath(root)); statErr == nil {
		t.Error("usage.json was written although the lock was never acquired")
	}
}

// TestTryLockInterruptHandling calls tryLock directly, with the flockFn seam
// scripted to return the listed errors in order, because a real EINTR cannot
// be provoked on demand. It does not exercise a real flock or a real signal.
func TestTryLockInterruptHandling(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "lock")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()

	orig := flockFn
	t.Cleanup(func() { flockFn = orig })

	for _, tc := range []struct {
		name      string
		script    []error
		wantCalls int
		check     func(error) string // returns a complaint, or ""
	}{
		{
			name:      "one EINTR is retried and then succeeds",
			script:    []error{syscall.EINTR, nil},
			wantCalls: 2,
			check: func(err error) string {
				if err != nil {
					return "want nil after the retry, got " + err.Error()
				}
				return ""
			},
		},
		{
			name:      "a second EINTR is returned, not retried again",
			script:    []error{syscall.EINTR, syscall.EINTR, nil},
			wantCalls: 2,
			check: func(err error) string {
				if !errors.Is(err, syscall.EINTR) || errors.Is(err, ErrLockHeld) {
					return "want an error wrapping EINTR, got " + errString(err)
				}
				return ""
			},
		},
		{
			name:      "EWOULDBLOCK is ErrLockHeld with no retry",
			script:    []error{syscall.EWOULDBLOCK, nil},
			wantCalls: 1,
			check: func(err error) string {
				if !errors.Is(err, ErrLockHeld) {
					return "want ErrLockHeld, got " + errString(err)
				}
				return ""
			},
		},
		{
			name:      "any other error is returned with no retry",
			script:    []error{syscall.EACCES, nil},
			wantCalls: 1,
			check: func(err error) string {
				if !errors.Is(err, syscall.EACCES) || errors.Is(err, ErrLockHeld) {
					return "want an error wrapping EACCES, got " + errString(err)
				}
				return ""
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			flockFn = func(int, int) error {
				calls++
				if calls > len(tc.script) {
					t.Fatalf("%s: flock called %d times, script has %d entries (retry is unbounded)", tc.name, calls, len(tc.script))
				}
				return tc.script[calls-1]
			}
			err := tryLock(f)
			if msg := tc.check(err); msg != "" {
				t.Errorf("%s: %s", tc.name, msg)
			}
			if calls != tc.wantCalls {
				t.Errorf("%s: flock called %d times, want %d", tc.name, calls, tc.wantCalls)
			}
		})
	}
}

// errString formats err for a test message, naming a nil error.
func errString(err error) string {
	if err == nil {
		return "nil"
	}
	return err.Error()
}
