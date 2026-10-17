//go:build unix

package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Lock represents an advisory file lock on the session directory's .lock file.
// The lock is non-blocking exclusive; if it cannot be acquired immediately,
// it reports ErrLockHeld. Closing the lock releases it; the file handle is kept
// open for the lock to persist.
type Lock struct {
	file *os.File
}

// Acquire takes an advisory non-blocking exclusive flock on the .lock file
// in the session directory. If the lock is already held by another process,
// it returns ErrLockHeld (from EWOULDBLOCK). Other errors are reported as-is.
// The returned Lock holds the file open; call Release or Close to unlock.
//
// The lock is per session directory, protecting against concurrent instances
// writing to the same workspace. It is not enforced on non-Unix platforms
// (see lock_other.go); those platforms are not supported in v0.1.
func Acquire(root, workspace string) (*Lock, error) {
	sessionDir := SessionDir(root, workspace)

	// Create the session directory if it doesn't exist.
	// Use 0o700 (rwx------) to restrict access to the owning user.
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	lockPath := filepath.Join(sessionDir, ".lock")

	// Open or create the lock file. Use O_WRONLY|O_CREATE to create if needed,
	// but do not truncate (|O_TRUNC).
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	// Attempt non-blocking exclusive lock.
	// syscall.LOCK_EX = exclusive lock
	// syscall.LOCK_NB = non-blocking (fail immediately if held)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close() // Flock failed; close error is secondary
		// Check for EWOULDBLOCK, which means the lock is held by another process.
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLockHeld
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}

	return &Lock{file: file}, nil
}

// Release unlocks the lock and closes the file. It is safe to call multiple times.
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}
	// Unlock before closing. flock is automatically released on close, but
	// we unlock explicitly to free the lock immediately.
	file := l.file
	l.file = nil
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		_ = file.Close() // Unlock failed; close error is secondary
		return fmt.Errorf("release lock: %w", err)
	}
	return file.Close()
}

// Close is an alias for Release, providing a common cleanup interface.
func (l *Lock) Close() error {
	return l.Release()
}
