package patch

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The diffs below are the exact Go %q strings the model sent in a live run,
// kept quoted so that liveUnquote decodes them to the exact bytes sent.

// liveCall3Quoted is call 3: a hunk whose final line is a lone space with no
// newline after it.
const liveCall3Quoted = `"--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -6,3 +6,6 @@\n // here, or the smoke test has nothing to do.\n func Divide(a, b float64) (float64, error) {\n+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n \treturn a / b, nil\n }\n "`

// liveCall5Quoted is call 5: a hunk with no ---/+++ file header.
const liveCall5Quoted = `"@@ -7,2 +7,5 @@ func Divide(a, b float64) (float64, error) {\n+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n \treturn a / b, nil\n }\n"`

// liveCall7Quoted is call 7: an unprefixed "// Divide returns a divided by b."
// line inside the hunk body, at line 11 of the diff.
const liveCall7Quoted = `"--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -1,3 +1,8 @@\n package calc\n \n+import \"errors\"\n+\n+// ErrDivideByZero is returned when division by zero is attempted.\n+var ErrDivideByZero = errors.New(\"division by zero\")\n+\n// Divide returns a divided by b.\n-func Divide(a, b float64) (float64, error) {\n-\treturn a / b, nil\n+//\n+// It returns ErrDivideByZero if b is zero.\n+func Divide(a, b float64) (float64, error) {\n+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n+\treturn a / b, nil\n }\n"`

// liveDivideGo is testdata/repo-go-module/calc/divide.go, ending "}\n".
// TestParseLiveDivideBaseFixture pins it against the file on disk.
const liveDivideGo = "package calc\n\n// Divide returns a divided by b.\n//\n// It has no input validation. Milestone 4's smoke test adds it; do not fix it\n// here, or the smoke test has nothing to do.\nfunc Divide(a, b float64) (float64, error) {\n\treturn a / b, nil\n}\n"

// liveCall3Want is liveDivideGo after call 3 applies: the guard sits after line 7.
const liveCall3Want = "package calc\n\n// Divide returns a divided by b.\n//\n// It has no input validation. Milestone 4's smoke test adds it; do not fix it\n// here, or the smoke test has nothing to do.\nfunc Divide(a, b float64) (float64, error) {\n\tif b == 0 {\n\t\treturn 0, ErrDivideByZero\n\t}\n\treturn a / b, nil\n}\n"

// liveUnquote decodes a quoted Go string literal to its exact bytes.
func liveUnquote(t *testing.T, quoted string) string {
	t.Helper()
	s, err := strconv.Unquote(quoted)
	if err != nil {
		t.Fatalf("unquote: %v", err)
	}
	return s
}

// expectLiveParseError fails the test unless Parse rejects diff with a
// *ParseError of the given kind whose message contains substr.
func expectLiveParseError(t *testing.T, diff string, kind ParseErrorKind, substr string) {
	t.Helper()
	_, err := Parse([]byte(diff))
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T (%v)", err, err)
	}
	if pe.Kind != kind {
		t.Errorf("kind mismatch: got %s, want %s (message %q)", pe.Kind, kind, pe.Message)
	}
	if !strings.Contains(pe.Message, substr) {
		t.Errorf("message %q does not contain %q", pe.Message, substr)
	}
}

// TestParseLiveDivideBaseFixture pins liveDivideGo against the testdata file on
// disk, so the fixtures below are built from the real base.
func TestParseLiveDivideBaseFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "repo-go-module", "calc", "divide.go"))
	if err != nil {
		t.Fatalf("read base fixture: %v", err)
	}
	if string(data) != liveDivideGo {
		t.Errorf("liveDivideGo differs from testdata\n got: %q\nwant: %q", data, liveDivideGo)
	}
}

// TestParseLiveWhitespaceFragmentAtEOF covers a final line that is whitespace
// only with no newline after it. That fragment is not a blank context line. The
// mirrors are a final line with text and no newline, which is kept, and a
// whitespace-only line that ends in a newline, met mid-hunk, which is still
// blank context.
func TestParseLiveWhitespaceFragmentAtEOF(t *testing.T) {
	call3 := liveUnquote(t, liveCall3Quoted)

	t.Run("call 3 parses to a hunk of four old and seven new lines", func(t *testing.T) {
		changes, err := Parse([]byte(call3))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if len(changes) != 1 || len(changes[0].Hunks) != 1 {
			t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
		}
		if h := changes[0].Hunks[0]; h.OldLines != 4 || h.NewLines != 7 || len(h.Lines) != 7 {
			t.Errorf("hunk: got old %d, new %d, %d lines; want old 4, new 7, 7 lines", h.OldLines, h.NewLines, len(h.Lines))
		}
	})

	t.Run("call 3 applies and inserts the guard after line 7", func(t *testing.T) {
		expectApplied(t, liveDivideGo, call3, liveCall3Want)
	})

	t.Run("tab-only final fragment is ignored the same way", func(t *testing.T) {
		diff := strings.TrimSuffix(call3, " ") + "\t"
		expectApplied(t, liveDivideGo, diff, liveCall3Want)
	})

	t.Run("multi-space final fragment is ignored the same way", func(t *testing.T) {
		diff := strings.TrimSuffix(call3, " ") + "  "
		expectApplied(t, liveDivideGo, diff, liveCall3Want)
	})

	t.Run("mirror: call 3 without the fragment applies the same", func(t *testing.T) {
		diff := strings.TrimSuffix(call3, " ")
		expectApplied(t, liveDivideGo, diff, liveCall3Want)
	})

	t.Run("mirror: a final line with text and no newline is kept", func(t *testing.T) {
		// "+// end" is the last line and has no newline after it. Dropping it
		// would leave only the context line, and the file would not change.
		diff := "--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -9,1 +9,2 @@\n }\n+// end"
		expectApplied(t, liveDivideGo, diff, liveDivideGo+"// end\n")
	})

	t.Run("mirror: whitespace-only line ending in newline mid-hunk stays blank context", func(t *testing.T) {
		// The " \n" line is the single-space context line for the blank line 2.
		diff := "--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -1,3 +1,4 @@\n package calc\n \n+// added\n // Divide returns a divided by b.\n"
		want := strings.Replace(liveDivideGo, "package calc\n\n", "package calc\n\n// added\n", 1)
		expectApplied(t, liveDivideGo, diff, want)
	})
}

// TestParseLiveHunkWithoutFileHeader covers a hunk that comes before any
// ---/+++ header, and a hunk in a diff --git block that has no ---/+++ pair.
// Such a hunk names no file, so Parse refuses it. The mirrors are the same hunk
// after its headers, and a diff --git block with its ---/+++ pair, which parse.
func TestParseLiveHunkWithoutFileHeader(t *testing.T) {
	call5 := liveUnquote(t, liveCall5Quoted)

	t.Run("call 5 hunk with no file header is rejected by line", func(t *testing.T) {
		expectLiveParseError(t, call5, ErrMissingHeader, "hunk at line 1 has no file header")
	})

	t.Run("diff --git line alone does not name the file for a hunk", func(t *testing.T) {
		diff := "diff --git a/calc/divide.go b/calc/divide.go\n" + call5
		expectLiveParseError(t, diff, ErrMissingHeader, "hunk at line 2 has no file header")
	})

	t.Run("mirror: the same hunk after its file headers parses", func(t *testing.T) {
		diff := "--- a/calc/divide.go\n+++ b/calc/divide.go\n" + call5
		changes, err := Parse([]byte(diff))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if len(changes) != 1 || len(changes[0].Hunks) != 1 {
			t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
		}
	})

	t.Run("mirror: a diff --git block with its --- and +++ lines parses", func(t *testing.T) {
		diff := "diff --git a/calc/divide.go b/calc/divide.go\n--- a/calc/divide.go\n+++ b/calc/divide.go\n" + call5
		changes, err := Parse([]byte(diff))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if len(changes) != 1 || len(changes[0].Hunks) != 1 {
			t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
		}
	})
}

// TestParseLiveHunkWithOneFileHeaderLine covers a hunk that has one of its two
// file header lines and not the other. Parse refuses it, since a hunk needs both
// its --- and +++ lines earlier in its file change. Two checks refuse it, on
// different paths:
//   - In a diff --git block with only +++, the hunk guard in parseFileChange is
//     reached, and it names the hunk line.
//   - With only ---, the missing +++ check runs on the @@ line before the hunk
//     guard, so the guard is never reached. That check names the --- line, not
//     the hunk. The header-less case takes the same path.
//
// The mirror is the diff --git block with both lines in
// TestParseLiveHunkWithoutFileHeader, which parses.
func TestParseLiveHunkWithOneFileHeaderLine(t *testing.T) {
	const hunk = "@@ -1,1 +1,1 @@\n-a\n+b\n"

	t.Run("diff --git block with only +++ is refused by the hunk guard at the hunk line", func(t *testing.T) {
		diff := "diff --git a/f b/f\n+++ b/f\n" + hunk
		expectLiveParseError(t, diff, ErrMissingHeader, "hunk at line 3 has no file header")
	})

	t.Run("diff --git block with only --- is refused by the missing +++ check", func(t *testing.T) {
		diff := "diff --git a/f b/f\n--- a/f\n" + hunk
		expectLiveParseError(t, diff, ErrMissingHeader, "missing +++ line after --- line")
	})

	t.Run("header-less --- with no +++ is refused by the missing +++ check", func(t *testing.T) {
		diff := "--- a/f\n" + hunk
		expectLiveParseError(t, diff, ErrMissingHeader, "missing +++ line after --- line")
	})
}

// TestParseLiveZeroFileChanges covers input that yields no file change at all.
// Parse refuses it rather than return an empty result that reads as success.
// The mirror is TestParseLiveHunkWithoutFileHeader's headered case, which
// yields a change and parses.
func TestParseLiveZeroFileChanges(t *testing.T) {
	t.Run("commentary with no headers is rejected", func(t *testing.T) {
		expectLiveParseError(t, "Here is the patch.\n", ErrMissingHeader, "no file changes")
	})

	t.Run("empty input is rejected", func(t *testing.T) {
		expectLiveParseError(t, "", ErrMissingHeader, "no file changes")
	})
}

// TestParseLiveUnprefixedHunkLine covers a line inside a hunk body that begins
// with none of ' ', '+', '-', '\' and is not a terminator. Parse refuses it and
// names its line, instead of ending the hunk and dropping what follows. Each of
// the four body prefixes is tested as the line that follows it. The mirror is
// TestParseLiveHunkTerminators.
func TestParseLiveUnprefixedHunkLine(t *testing.T) {
	t.Run("call 7 unprefixed line is rejected by name at line 11", func(t *testing.T) {
		expectLiveParseError(t, liveUnquote(t, liveCall7Quoted), ErrHunkCountMismatch, "line 11 is not a hunk line")
	})

	t.Run("a bare line before a further body line is rejected by name", func(t *testing.T) {
		diff := "--- a/one.txt\n+++ b/one.txt\n@@ -1,2 +1,2 @@\n-a\nstray\n+b\n"
		expectLiveParseError(t, diff, ErrHunkCountMismatch, "line 5 is not a hunk line")
	})

	t.Run("a bare line before a minus line is rejected by name", func(t *testing.T) {
		diff := "--- a/one.txt\n+++ b/one.txt\n@@ -1,2 +1,1 @@\n-a\nstray\n-b\n"
		expectLiveParseError(t, diff, ErrHunkCountMismatch, "line 5 is not a hunk line")
	})

	t.Run("a bare line before a context line is rejected by name", func(t *testing.T) {
		diff := "--- a/one.txt\n+++ b/one.txt\n@@ -1,2 +1,2 @@\n-a\nstray\n b\n"
		expectLiveParseError(t, diff, ErrHunkCountMismatch, "line 5 is not a hunk line")
	})

	t.Run("a bare line before a backslash marker is rejected by name", func(t *testing.T) {
		diff := "--- a/one.txt\n+++ b/one.txt\n@@ -1,1 +1,1 @@\n-a\n+b\nstray\n\\ No newline at end of file\n"
		expectLiveParseError(t, diff, ErrHunkCountMismatch, "line 6 is not a hunk line")
	})

	t.Run("mirror: a bare line with no body line after it is trailing text and ends the hunk", func(t *testing.T) {
		diff := "--- a/one.txt\n+++ b/one.txt\n@@ -1,1 +1,1 @@\n-a\n+b\nstray\n"
		changes, err := Parse([]byte(diff))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if len(changes) != 1 || len(changes[0].Hunks[0].Lines) != 2 {
			t.Errorf("expected 1 change with a 2-line hunk, got %+v", changes)
		}
	})
}

// TestParseLiveUnprefixedLineBeforeNextHunk is a mirror of TestParseLiveUnprefixedHunkLine:
// a bare line before a "@@" terminator is trailing text and ends its hunk.
func TestParseLiveUnprefixedLineBeforeNextHunk(t *testing.T) {
	// The body lines after the stray line belong to the next hunk, past the
	// terminator, so the stray line cuts nothing off.
	diff := "--- a/one.txt\n+++ b/one.txt\n@@ -1,1 +1,1 @@\n-a\n+b\nstray\n@@ -5,1 +5,1 @@\n-c\n+d\n"
	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 2 {
		t.Fatalf("expected 1 change with 2 hunks, got %+v", changes)
	}
}

// TestParseLiveUnprefixedLineBeforeNextFile is a mirror of TestParseLiveUnprefixedHunkLine
// for the two file-level terminators: a "diff --git " line, and a header-less
// "--- " and "+++ " pair. A bare line before either is trailing text, so it ends
// its hunk and the next file parses as its own change.
func TestParseLiveUnprefixedLineBeforeNextFile(t *testing.T) {
	for _, tc := range []struct {
		name string
		diff string
	}{
		{
			name: "diff --git line",
			diff: "--- a/one.txt\n+++ b/one.txt\n@@ -1,1 +1,1 @@\n-a\n+b\nstray\ndiff --git a/two b/two\n--- a/two\n+++ b/two\n@@ -1,1 +1,1 @@\n-c\n+d\n",
		},
		{
			name: "header-less --- and +++ pair",
			diff: "--- a/one.txt\n+++ b/one.txt\n@@ -1,1 +1,1 @@\n-a\n+b\nstray\n--- a/two.txt\n+++ b/two.txt\n@@ -1,1 +1,1 @@\n-c\n+d\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes, err := Parse([]byte(tc.diff))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if len(changes) != 2 || len(changes[0].Hunks) != 1 || len(changes[1].Hunks) != 1 {
				t.Fatalf("expected 2 changes with 1 hunk each, got %+v", changes)
			}
		})
	}
}

// TestParseLiveHunkTerminators is the mirror of TestParseLiveUnprefixedHunkLine.
// Each line here is a legitimate hunk terminator, so the hunk ends at it with no
// error and the following content parses. The "@@" case also has a blank line
// before the terminator, which must not be counted as a body line.
func TestParseLiveHunkTerminators(t *testing.T) {
	const head = "--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -1,1 +1,1 @@\n-package calc\n+package calc2\n"
	for _, tc := range []struct {
		name        string
		diff        string
		wantChanges int
		wantHunks   int
	}{
		{
			name:        "next @@ hunk",
			diff:        head + "@@ -9,1 +9,1 @@\n-}\n+}\n",
			wantChanges: 1,
			wantHunks:   2,
		},
		{
			name:        "blank line then next @@ hunk",
			diff:        head + "\n@@ -9,1 +9,1 @@\n-}\n+}\n",
			wantChanges: 1,
			wantHunks:   2,
		},
		{
			name:        "next diff --git file",
			diff:        head + "diff --git a/calc/other.go b/calc/other.go\n--- a/calc/other.go\n+++ b/calc/other.go\n@@ -1,1 +1,1 @@\n-x\n+y\n",
			wantChanges: 2,
			wantHunks:   1,
		},
		{
			name:        "next header-less file pair",
			diff:        head + "--- a/calc/other.go\n+++ b/calc/other.go\n@@ -1,1 +1,1 @@\n-x\n+y\n",
			wantChanges: 2,
			wantHunks:   1,
		},
		{
			name:        "format-patch signature and version",
			diff:        head + "-- \n2.39.1\n\n",
			wantChanges: 1,
			wantHunks:   1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes, err := Parse([]byte(tc.diff))
			if err != nil {
				t.Fatalf("expected a clean parse, got %v", err)
			}
			if len(changes) != tc.wantChanges {
				t.Fatalf("changes: got %d, want %d", len(changes), tc.wantChanges)
			}
			if len(changes[0].Hunks) != tc.wantHunks {
				t.Fatalf("hunks in first change: got %d, want %d", len(changes[0].Hunks), tc.wantHunks)
			}
			if h := changes[0].Hunks[0]; len(h.Lines) != 2 {
				t.Errorf("first hunk: got %d body lines, want 2 (the terminator must not be counted)", len(h.Lines))
			}
		})
	}
}
