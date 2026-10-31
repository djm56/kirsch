//go:build !unix

package session

// ProbeLock reports LockUnknown on non-Unix platforms: the lock is never
// enforced there (see lock_other.go), so there is no holder to detect. It
// touches nothing on disk.
func ProbeLock(_, _ string) (LockState, error) {
	return LockUnknown, nil
}
