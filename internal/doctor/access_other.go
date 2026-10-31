//go:build !unix

package doctor

// writableDir cannot query permissions without a write probe on this platform,
// and doctor never writes, so it reports checked as false. The caller turns that
// into a warn saying the folder was not checked.
func writableDir(_ string) (writable, checked bool) {
	return false, false
}

// writableFile cannot query permissions without opening the file for writing,
// which doctor never does, so it reports checked as false.
func writableFile(_ string) (writable, checked bool) {
	return false, false
}

// readableFile is not checked on this platform, to match writableDir and
// writableFile, so it reports checked as false.
func readableFile(_ string) (readable, checked bool) {
	return false, false
}
