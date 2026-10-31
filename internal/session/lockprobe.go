//go:build unix

package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ProbeLock reports whether the workspace's lock is held, without taking it.
//
// It only reads. It never creates the session directory or the .lock file: a
// missing file means no instance has ever locked this workspace, so it returns
// LockFree. An existing file is opened read-only and given one non-blocking
// shared flock attempt. A shared request fails with EWOULDBLOCK while a holder
// has the exclusive lock Acquire takes, so that failure means LockHeld. If the
// attempt succeeds the lock is released and the file closed at once, so a
// holder is never disturbed and the call never waits. EINTR is retried once, as
// tryLock does; a second EINTR, or any other error, returns LockUnknown and the
// error.
//
// The probe is not invisible to a kirsch that is starting. For the moment the
// shared lock is held, Acquire in another process gets EWOULDBLOCK, which it
// reads as ErrLockHeld, so that kirsch starts in concurrent-instance mode: no
// resume, no session grants, and the warning "another Kirsch is running here"
// (App.AcquireLock). The window is a few system calls wide. Two probes at the
// same time share the lock and do not affect each other.
func ProbeLock(root, workspace string) (LockState, error) {
	lockPath := filepath.Join(SessionDir(root, workspace), ".lock")

	file, err := os.OpenFile(lockPath, os.O_RDONLY, 0) // #nosec G304 -- fixed ".lock" basename in a session dir named by a sha256 hex of the workspace
	if errors.Is(err, os.ErrNotExist) {
		return LockFree, nil
	}
	if err != nil {
		return LockUnknown, fmt.Errorf("open lock file: %w", err)
	}
	defer func() { _ = file.Close() }() // closing also drops any shared lock taken below

	for attempt := 0; ; attempt++ {
		err := flockFn(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
		if err == nil {
			_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) // best effort; Close releases it anyway
			return LockFree, nil
		}
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return LockHeld, nil
		}
		if errors.Is(err, syscall.EINTR) && attempt == 0 {
			continue
		}
		return LockUnknown, fmt.Errorf("probe lock: %w", err)
	}
}
