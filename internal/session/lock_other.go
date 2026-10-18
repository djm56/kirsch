//go:build !unix

package session

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock represents an advisory file lock. On non-Unix platforms (Windows,
// plan9, js, wasip1), the single-instance guard is not enforced: the lock is
// always acquired and held as a stub. This allows cross-platform compilation
// without requiring Unix-specific syscalls.
//
// These platforms are not supported in v0.1. Two instances may patch the
// same working tree on a non-Unix platform simultaneously; the guard is
// advisory and Unix-only by design.
type Lock struct {
	file *os.File
}

// Acquire returns a lock stub that is always acquired successfully. The lock is
// never enforced on non-Unix platforms; two processes may write to the same
// workspace without contention.
//
// The session directory is still created, but no flock is taken.
func Acquire(root, workspace string) (*Lock, error) {
	sessionDir := SessionDir(root, workspace)

	// Create the session directory if it doesn't exist.
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	// On non-Unix platforms, create the file but do not lock it. This is a stub
	// to allow cross-platform compilation; the lock is never enforced.
	lockPath := filepath.Join(sessionDir, ".lock")
	file, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	return &Lock{file: file}, nil
}

// Release closes the lock file and is idempotent. On non-Unix platforms,
// this is a no-op lock release since flock was never acquired.
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	return file.Close()
}

// Close is an alias for Release.
func (l *Lock) Close() error {
	return l.Release()
}

// tryLock is a no-op stub on non-Unix platforms; the lock is always acquired.
func tryLock(_ *os.File) error {
	return nil
}
