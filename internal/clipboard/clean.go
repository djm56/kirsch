// Package clipboard turns a reply's source text into a safe clipboard payload
// and delivers it by every route available. ui-spec §6 (v0.1.3, amendment 101).
//
// It holds no UI state and does not import internal/tui: the TUI calls it
// through a function field on its Model, and cmd/kirsch supplies the real
// routes.
package clipboard

import (
	"regexp"
	"strings"
)

// ansiRe matches CSI in its 7-bit and 8-bit forms; OSC and two-byte escapes in
// their 7-bit (ESC-prefixed) form. A charset selection (ESC, an opening
// parenthesis and one letter or digit) is matched; it has no 8-bit form. It is
// the one pattern: StripANSI exposes it, and tui's Sanitize uses it through
// StripANSI. The 8-bit form is written as a rune (\x{009b}) because a bare 0x9b
// byte is not valid UTF-8 and Go's regexp rejects the pattern. A C1 control
// outside a sequence is not matched here: encoded as a code point
// (U+0080–U+009F, two bytes in UTF-8) it is dropped by Clean's removal pass,
// but a lone byte 0x80–0x9F is invalid UTF-8, so Clean turns it into U+FFFD and
// keeps that.
var ansiRe = regexp.MustCompile(
	"\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)" + // OSC ... BEL or ST
		"|\x1b\\[[0-?]*[ -/]*[@-~]" + // CSI
		"|\\x{009b}[0-?]*[ -/]*[@-~]" + // 8-bit CSI
		"|\x1b[@-Z\\\\-_]" + // two-byte escapes
		"|\x1b\\([A-Za-z0-9]", // charset selection
)

// StripANSI returns s with every escape sequence ansiRe matches removed, and
// nothing else changed.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// Clean returns s with everything removed that should not reach a clipboard.
// ui-spec §6, "Source text and payload".
//
// In order: ANSI escape sequences are removed; CRLF becomes LF; then these
// code points are removed: C0 controls other than tab and newline (a CR left
// after the CRLF step is one of them), DEL, the C1 controls U+0080–U+009F, the
// bidirectional controls U+202A–U+202E and U+2066–U+2069, and the Unicode tag
// characters U+E0000–U+E007F. Everything else is kept, including tabs, newlines,
// zero-width characters and the directional marks U+200E, U+200F and U+061C.
// No line-width cap applies.
//
// A byte that is not valid UTF-8 comes out as U+FFFD, because the removal pass
// walks runes. That includes a raw 0x9B byte: the 8-bit CSI pattern matches the
// code point U+009B, not the byte, so a bare 0x9B is kept as U+FFFD.
func Clean(s string) string {
	s = StripANSI(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Map(func(r rune) rune {
		if removedRune(r) {
			return -1
		}
		return r
	}, s)
}

// removedRune reports whether Clean drops r.
func removedRune(r rune) bool {
	switch {
	case r < 0x20:
		return r != '\t' && r != '\n'
	case r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f:
		return true
	case r >= 0x202a && r <= 0x202e:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	case r >= 0xe0000 && r <= 0xe007f:
		return true
	}
	return false
}
