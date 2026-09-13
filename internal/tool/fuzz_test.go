package tool

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Fuzz targets for the tool layer's input handling.
//
// Truncate guards the terminal against malformed output; DecodeInput guards
// against malformed tool calls, which from M3 are written by a model and will
// sometimes be wrong. Run properly now and then:
//   go test ./internal/tool/ -run=Fuzz -fuzz=FuzzTruncate -fuzztime=60s

// FuzzTruncate asserts that truncation never produces invalid UTF-8 and never
// exceeds its budget by more than the marker.
//
// A cut mid-rune emits an invalid byte sequence, which corrupts the terminal —
// and the failure presents as a rendering bug, so it gets diagnosed as anything
// but truncation.
func FuzzTruncate(f *testing.F) {
	f.Add("hello world", 5)
	f.Add(strings.Repeat("漢", 100), 17)
	f.Add(strings.Repeat("👍", 50), 9)
	f.Add("line one\nline two\nline three\n", 12)
	f.Add("", 0)

	f.Fuzz(func(t *testing.T, in string, max int) {
		if max < 0 || max > 1<<20 {
			t.Skip()
		}
		out, truncated := Truncate(in, max)
		if !utf8.ValidString(out) && utf8.ValidString(in) {
			t.Fatalf("truncating valid UTF-8 to %d produced invalid UTF-8", max)
		}
		if !truncated && out != in {
			t.Fatalf("reported untruncated but changed the input: %q -> %q", in, out)
		}
		if truncated && len(out) > max+64 {
			t.Fatalf("result is %d bytes for a cap of %d", len(out), max)
		}
	})
}

// FuzzDecodeInput asserts the tool-input decoder never panics and always
// explains itself.
//
// From M3 a model writes this JSON, and a model will write malformed JSON. The
// decoder's message is what lets it correct itself, so an empty message is a
// failure even when the rejection is right.
func FuzzDecodeInput(f *testing.F) {
	for _, s := range []string{
		`{}`, `{"path":"a.txt"}`, `{"path":123}`, `{"unknown":1}`,
		`[]`, `null`, `{`, `{"path":"a","path":"b"}`, `{"path":"\ud800"}`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		var in struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
		}
		if err := DecodeInput([]byte(raw), &in); err != nil {
			if err.Kind != KindToolInputInvalid {
				t.Fatalf("decode failure reported as %s, want tool_input_invalid", err.Kind)
			}
			if strings.TrimSpace(err.Message) == "" {
				t.Fatal("rejection carries no message; a model cannot correct itself from silence")
			}
		}
	})
}
