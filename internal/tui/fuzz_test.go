package tui

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Fuzz targets for the surfaces an attacker controls.
//
// ADR 0007 says tool output is untrusted input. In a repository Kirsch did not
// write, every byte that reaches Sanitize came from somewhere else — a file, a
// command's stdout, a search hit — so these functions are the boundary between
// hostile bytes and a terminal that will faithfully execute whatever control
// sequences it is handed.
//
// Run the corpus in CI (`go test`), and run them properly now and then:
//   go test ./internal/tool/ -run=Fuzz -fuzz=FuzzSanitize -fuzztime=60s

// FuzzSanitize asserts the four properties §7.1 exists to guarantee, against
// arbitrary input.
func FuzzSanitize(f *testing.F) {
	seeds := []string{
		"",
		"plain text",
		"\x1b[31mred\x1b[0m",
		"\x1b]0;title\x07payload",
		"\x1b[2J\x1b[1;1H",
		"10%\r50%\r100%",
		"a\r\nb\r\nc",
		"\tindented\t\ttwice",
		"nul\x00byte",
		"\x9b31m8-bit CSI",
		strings.Repeat("x", 5000),
		strings.Repeat("漢", 1500),
		"👩‍💻 zwj sequence",
		"é combining",
		"\x1b\x1b\x1b[[[",
		"\x1b]8;;http://evil\x1b\\link\x1b]8;;\x1b\\",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		out := Sanitize(in)

		// 1. No escape byte survives. This is the property that keeps a
		//    malicious file from driving the user's terminal.
		if strings.ContainsRune(out, 0x1b) {
			t.Fatalf("ESC survived sanitisation: %q -> %q", in, out)
		}
		// 2. No control character that could move the cursor or corrupt the
		//    display survives. \n is the sole exception: it is the line
		//    separator the renderer splits on.
		for _, r := range out {
			if r == '\n' {
				continue
			}
			if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
				t.Fatalf("control %#U survived: %q -> %q", r, in, out)
			}
		}
		// 3. Output is always valid UTF-8. Invalid bytes reaching lipgloss
		//    produce width miscalculations and corrupted borders.
		if !utf8.ValidString(out) {
			t.Fatalf("produced invalid UTF-8 from %q", in)
		}
		// 4. No single line exceeds the column cap, however wide its runes.
		for i, line := range strings.Split(out, "\n") {
			if w := runewidth.StringWidth(line); w > MaxRenderedLineWidth {
				t.Fatalf("line %d is %d columns, cap is %d", i, w, MaxRenderedLineWidth)
			}
		}
	})
}

// FuzzSanitizeIsIdempotent: sanitising twice must equal sanitising once.
//
// Not a stylistic nicety. Content is sanitised on ingest and may be re-rendered
// through the same path; a function that keeps changing its own output would
// corrupt text on the second pass — and would mean the first pass had not
// actually reached a safe state.
func FuzzSanitizeIsIdempotent(f *testing.F) {
	for _, s := range []string{"", "a\tb", "\x1b[31mx", "10%\r20%", "\x00\x01\x02", "漢字\r\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		once := Sanitize(in)
		if twice := Sanitize(once); twice != once {
			t.Fatalf("not idempotent:\n  in:    %q\n  once:  %q\n  twice: %q", in, once, twice)
		}
	})
}

// FuzzNormalizeSS3 covers the rewrite that sits in the path of every keystroke.
//
// Different threat model from Sanitize: these bytes come from the user's own
// terminal, not from a hostile file, so the risk is not injection but a mangled
// input stream. A rewrite that changed the length would short-read Bubble Tea's
// input buffer; one that touched a byte other than the `O` of an SS3 Home or End
// would corrupt some other key; one that was not idempotent would mean the first
// pass had not finished. The bounds are the part worth fuzzing — the loop indexes
// b[i+1] and b[i+2] — and a panic there happens inside the input goroutine, where
// it takes the program down with the terminal still in raw mode.
func FuzzNormalizeSS3(f *testing.F) {
	for _, s := range []string{
		"", "\x1b", "\x1bO", "\x1bOH", "\x1bOF", "\x1bOA", "\x1bOP",
		"\x1b[H", "\x1b\x1bOH", "\x1bOH\x1bOF", "\x1bOHOH", "OH",
		"go to the end", "\x1bO\x1bOH", "\x00\x1bOH\x00",
		strings.Repeat("\x1bOH", 400),
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		b := []byte(in)
		normalizeSS3(b) // must not panic on any slice, however truncated

		// 1. Length is never changed. ss3File.Read returns the file's own
		//    count, so a rewrite that grew or shrank the buffer would hand
		//    Bubble Tea bytes it never read or drop bytes it did.
		if len(b) != len(in) {
			t.Fatalf("length changed from %d to %d", len(in), len(b))
		}
		// 2. The only edit ever made is `O` -> `[`. Any other difference means
		//    the rewrite reached a byte belonging to some other key.
		for i := range b {
			if b[i] == in[i] {
				continue
			}
			if in[i] != 'O' || b[i] != '[' {
				t.Fatalf("byte %d changed %q -> %q; the only permitted edit is 'O' -> '['",
					i, in[i], b[i])
			}
		}
		// 3. Idempotent: a second pass changes nothing. `\x1b[H` is not a
		//    match, so a rewritten sequence cannot be rewritten again — and if
		//    it could, the first pass had left work undone.
		again := make([]byte, len(b))
		copy(again, b)
		normalizeSS3(again)
		if !bytes.Equal(again, b) {
			t.Fatalf("not idempotent:\n  in:    %q\n  once:  %q\n  twice: %q", in, b, again)
		}
	})
}
