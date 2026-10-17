package session

import "errors"

// ErrLockHeld is returned when an advisory flock cannot be acquired because
// another process already holds it (EWOULDBLOCK). It is defined here in an
// untagged file so it is accessible from both Unix and non-Unix platforms.
var ErrLockHeld = errors.New("lock held by another instance")
