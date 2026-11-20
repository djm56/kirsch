package clipboard

import "testing"

// TestCleanRemoves runs one removed character or sequence per case through
// Clean, each embedded between two letters so a removal shows as "ab".
func TestCleanRemoves(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"U+0000", "a\x00b", "ab"},
		{"U+001F", "a\x1fb", "ab"},
		{"U+007F DEL", "a\x7fb", "ab"},
		{"U+0080", "a\u0080b", "ab"},
		{"U+009F", "a\u009fb", "ab"},
		{"U+202A", "a\u202ab", "ab"},
		{"U+202E", "a\u202eb", "ab"},
		{"U+2066", "a\u2066b", "ab"},
		{"U+2069", "a\u2069b", "ab"},
		{"U+E0000", "a\U000e0000b", "ab"},
		{"U+E007F", "a\U000e007fb", "ab"},
		{"ANSI CSI", "a\x1b[31mb\x1b[0m", "ab"},
		{"ANSI OSC", "a\x1b]0;title\x07b", "ab"},
		{"8-bit CSI", "a\u009b31mb", "ab"},
		{"ANSI OSC ST-terminated", "a\x1b]0;title\x1b\\b", "ab"},
		{"charset selection ESC ( B", "a\x1b(Bb", "ab"},
		{"CR in CRLF", "a\r\nb", "a\nb"},
		{"lone CR", "a\rb", "ab"},
		{"lone ESC", "a\x1bb", "ab"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Clean(c.in); got != c.want {
				t.Errorf("Clean(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestCleanKeeps is the positive control for TestCleanRemoves: the same
// function, the same embedding, and characters that must survive, including the
// neighbour just outside each removed range. A Clean that dropped everything
// would pass the removal cases and fail here.
func TestCleanKeeps(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"U+0009 tab", "a\tb"},
		{"U+000A newline", "a\nb"},
		{"U+00A0 no-break space (just past C1)", "a b"},
		{"U+200B zero-width space", "a\u200bb"},
		{"U+200E left-to-right mark", "a\u200eb"},
		{"U+200F right-to-left mark", "a\u200fb"},
		{"U+061C Arabic letter mark", "a\u061cb"},
		{"U+2029 (just before U+202A)", "a b"},
		{"U+202F (just after U+202E)", "a b"},
		{"U+2065 (just before U+2066)", "a⁥b"},
		{"U+206A (just after U+2069)", "a\u206ab"},
		{"U+DFFFF (just before U+E0000)", "a\U000dffffb"},
		{"U+E0080 (just after U+E007F)", "a\U000e0080b"},
		{"U+0020 space (just past C0)", "a b"},
		{"U+007E tilde (just before DEL)", "a~b"},
		{"markdown fence", "```go\nx := 1\n```\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Clean(c.in); got != c.in {
				t.Errorf("Clean(%q) = %q, want it unchanged", c.in, got)
			}
		})
	}
}

// TestCleanLoneC1ByteBecomesReplacement pins what Clean does with a C1 control
// written as a lone byte rather than as a code point. A lone 0x9B (or any lone
// 0x80–0x9F) is invalid UTF-8: ansiRe's 8-bit CSI branch matches the code point
// U+009B, not the byte, and the removal pass sees U+FFFD, which it keeps. The
// control rows run the same C1 controls as code points through the same call
// and expect them removed, so the byte rows cannot pass because Clean returned
// its input or dropped everything.
func TestCleanLoneC1ByteBecomesReplacement(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"lone byte 0x9B", "\x9b", "�"},
		{"lone byte 0x9B before CSI parameters", "a\x9b31mb", "a�31mb"},
		{"lone byte 0x80", "a\x80b", "a�b"},
		{"control: code point U+009B before CSI parameters", "a\u009b31mb", "ab"},
		{"control: code point U+0080", "a\u0080b", "ab"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Clean(c.in); got != c.want {
				t.Errorf("Clean(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestCleanIsIdempotent: cleaning cleaned text changes nothing, which is what
// lets the TUI clean a code block that was found in already-cleaned text. The
// inputs try to build an escape sequence or a CRLF out of what the first pass
// removes; they cannot, because every escape introducer (ESC, U+009B) is itself
// removed. The first pass must change at least one input, or the loop is not
// exercising anything.
func TestCleanIsIdempotent(t *testing.T) {
	inputs := []string{
		"a\x1b[3\x001mb",
		"\x1b\x1b[31m",
		"\x1b[\u202e31m",
		"\r\r\n",
		"\x1b]0;t\x00\x07x",
		"\u009b31m",
		"plain",
	}
	// The 8-bit CSI input is removed whole by the first pass.
	if got := Clean("\u009b31m"); got != "" {
		t.Errorf("Clean(%q) = %q, want empty", "\u009b31m", got)
	}
	changed := 0
	for _, in := range inputs {
		once := Clean(in)
		if once != in {
			changed++
		}
		if twice := Clean(once); twice != once {
			t.Errorf("Clean(Clean(%q)) = %q, want %q", in, twice, once)
		}
	}
	if changed == 0 {
		t.Error("no input was changed by Clean, so the idempotence check proved nothing")
	}
}
