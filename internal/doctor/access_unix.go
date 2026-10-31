//go:build unix

package doctor

import "syscall"

// accessROK, accessWOK and accessXOK are the access(2) mode bits for read,
// write and search permission. syscall does not export them under these names;
// the values are the same on every Unix.
const (
	accessROK = 0x4
	accessWOK = 0x2
	accessXOK = 0x1
)

// writableDir reports whether the calling user may create entries in dir, using
// access(2) with W_OK|X_OK. That is a permission query: it writes nothing, so
// doctor leaves no temporary file behind. checked is always true on Unix.
// access(2) checks against the real user and group IDs.
func writableDir(dir string) (writable, checked bool) {
	return syscall.Access(dir, accessWOK|accessXOK) == nil, true
}

// writableFile reports whether the calling user may open path for writing, using
// access(2) with W_OK. Like writableDir it writes nothing. checked is always
// true on Unix.
func writableFile(path string) (writable, checked bool) {
	return syscall.Access(path, accessWOK) == nil, true
}

// readableFile reports whether the calling user may open path for reading,
// using access(2) with R_OK. Like writableDir it writes nothing. checked is
// always true on Unix.
func readableFile(path string) (readable, checked bool) {
	return syscall.Access(path, accessROK) == nil, true
}
