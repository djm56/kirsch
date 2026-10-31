package session

// LockState is what ProbeLock found out about a workspace's single-instance
// lock. It lives in an untagged file so callers compile on every platform.
type LockState int

// The outcomes of ProbeLock.
const (
	// LockFree means no other process holds the lock, or there is no lock file.
	LockFree LockState = iota
	// LockHeld means another open file description holds the lock.
	LockHeld
	// LockUnknown means the lock could not be checked: either the platform does
	// not enforce it (see lock_other.go) or the probe returned an error.
	LockUnknown
)
