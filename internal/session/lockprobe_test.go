//go:build unix

package session

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestProbeLock drives ProbeLock against the real flock: no file, an unlocked
// file, and a file held by Acquire (a separate open file description in the
// same process, which flock treats as a different holder).
func TestProbeLock(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()
	lockPath := filepath.Join(SessionDir(root, ws), ".lock")

	t.Run("no session directory is free", func(t *testing.T) {
		st, err := ProbeLock(root, ws)
		if err != nil || st != LockFree {
			t.Fatalf("ProbeLock = %v, %v; want LockFree, nil", st, err)
		}
		// Creates-nothing: neither the directory nor the file appeared.
		if _, err := os.Stat(SessionDir(root, ws)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("session dir after probe: %v, want not exist", err)
		}
	})

	t.Run("directory without lock file is free and stays without", func(t *testing.T) {
		if err := os.MkdirAll(SessionDir(root, ws), 0o700); err != nil {
			t.Fatal(err)
		}
		st, err := ProbeLock(root, ws)
		if err != nil || st != LockFree {
			t.Fatalf("ProbeLock = %v, %v; want LockFree, nil", st, err)
		}
		if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
			t.Errorf(".lock after probe: %v, want not exist", err)
		}
	})

	t.Run("unlocked file is free", func(t *testing.T) {
		l, err := Acquire(root, ws)
		if err != nil {
			t.Fatal(err)
		}
		if err := l.Release(); err != nil {
			t.Fatal(err)
		}
		st, err := ProbeLock(root, ws)
		if err != nil || st != LockFree {
			t.Fatalf("ProbeLock = %v, %v; want LockFree, nil", st, err)
		}
	})

	t.Run("held lock is held, and stays held", func(t *testing.T) {
		l, err := Acquire(root, ws)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Release()
		for i := 0; i < 2; i++ { // twice: the first probe must not release the holder
			st, err := ProbeLock(root, ws)
			if err != nil || st != LockHeld {
				t.Fatalf("probe %d = %v, %v; want LockHeld, nil", i, st, err)
			}
		}
		if _, err := Acquire(root, ws); !errors.Is(err, ErrLockHeld) {
			t.Errorf("Acquire after probes = %v, want ErrLockHeld", err)
		}
	})
}

// TestProbeLockFreeLeavesNoLock: a probe that reports free must not keep its
// shared lock. The lock file exists and is unlocked; after ProbeLock returns
// LockFree, Acquire (an exclusive request on another open file description)
// succeeds at once. A leaked shared lock would make it fail with ErrLockHeld.
func TestProbeLockFreeLeavesNoLock(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()
	first, err := Acquire(root, ws) // creates the .lock file
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(SessionDir(root, ws), ".lock")); err != nil {
		t.Fatalf("control: .lock missing: %v", err)
	}

	st, err := ProbeLock(root, ws)
	if err != nil || st != LockFree {
		t.Fatalf("ProbeLock = %v, %v; want LockFree, nil", st, err)
	}
	l, err := Acquire(root, ws)
	if err != nil {
		t.Fatalf("Acquire right after a free probe = %v, want success (the probe left its lock behind)", err)
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
}

// TestProbeLockOtherError: a path component that is a file makes the open fail
// with something other than not-exist; that is LockUnknown plus an error, not
// free.
func TestProbeLockOtherError(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()
	// root/sessions is a file, so the lock path cannot exist (ENOTDIR).
	if err := os.WriteFile(filepath.Join(root, "sessions"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ProbeLock(root, ws)
	if err == nil || st != LockUnknown {
		t.Fatalf("ProbeLock = %v, %v; want LockUnknown and an error", st, err)
	}
}

// TestProbeLockInterruptHandling scripts the flockFn seam (a real EINTR cannot
// be provoked on demand): one EINTR is retried, a second is an error, and
// EWOULDBLOCK is held with no retry. The file is real; no signal is sent.
func TestProbeLockInterruptHandling(t *testing.T) {
	root, ws := t.TempDir(), t.TempDir()
	l, err := Acquire(root, ws)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}

	orig := flockFn
	t.Cleanup(func() { flockFn = orig })

	for _, tc := range []struct {
		name      string
		script    []error
		wantCalls int
		wantState LockState
		wantErr   bool
	}{
		{"one EINTR is retried then free", []error{syscall.EINTR, nil}, 2, LockFree, false},
		{"a second EINTR is an error", []error{syscall.EINTR, syscall.EINTR, nil}, 2, LockUnknown, true},
		{"EWOULDBLOCK is held with no retry", []error{syscall.EWOULDBLOCK, nil}, 1, LockHeld, false},
		{"EINTR then EWOULDBLOCK is held", []error{syscall.EINTR, syscall.EWOULDBLOCK}, 2, LockHeld, false},
		{"another error is not retried", []error{syscall.EACCES, nil}, 1, LockUnknown, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			flockFn = func(int, int) error {
				calls++
				if calls > len(tc.script) {
					t.Fatalf("flock called %d times, script has %d (retry is unbounded)", calls, len(tc.script))
				}
				return tc.script[calls-1]
			}
			st, err := ProbeLock(root, ws)
			if st != tc.wantState || (err != nil) != tc.wantErr {
				t.Errorf("ProbeLock = %v, %v; want %v, err=%v", st, err, tc.wantState, tc.wantErr)
			}
			if calls != tc.wantCalls {
				t.Errorf("flock called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}
