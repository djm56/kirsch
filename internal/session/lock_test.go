//go:build unix

package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
