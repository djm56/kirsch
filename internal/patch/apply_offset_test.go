package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// divideBase is testdata/repo-go-module/calc/divide_test.go, ending "}\n".
// TestDivideBaseFixture pins it against the file on disk.
const divideBase = "package calc\n\nimport \"testing\"\n\nfunc TestDivide(t *testing.T) {\n\tgot, err := Divide(6, 3)\n\tif err != nil {\n\t\tt.Fatalf(\"Divide() error = %v\", err)\n\t}\n\tif got != 2 {\n\t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n\t}\n}\n"

// divideAdded is the added lines of the second hunk of captured call 2:
// the tests appended after the final "}" of the base.
const divideAdded = "\nfunc TestDivideByZero(t *testing.T) {\n\t_, err := Divide(5, 0)\n\tif err == nil {\n\t\tt.Error(\"Divide(5, 0) expected error, got nil\")\n\t}\n\tif err.Error() != \"division by zero\" {\n\t\tt.Errorf(\"Divide(5, 0) error = %v, want 'division by zero'\", err)\n\t}\n}\n\nfunc TestDivideZeroByNonZero(t *testing.T) {\n\tgot, err := Divide(0, 5)\n\tif err != nil {\n\t\tt.Fatalf(\"Divide(0, 5) unexpected error = %v\", err)\n\t}\n\tif got != 0 {\n\t\tt.Errorf(\"Divide(0, 5) = %v, want 0\", got)\n\t}\n}\n"

// divideAddedDiff is divideAdded as the '+' body of a hunk.
const divideAddedDiff = "+\n+func TestDivideByZero(t *testing.T) {\n+\t_, err := Divide(5, 0)\n+\tif err == nil {\n+\t\tt.Error(\"Divide(5, 0) expected error, got nil\")\n+\t}\n+\tif err.Error() != \"division by zero\" {\n+\t\tt.Errorf(\"Divide(5, 0) error = %v, want 'division by zero'\", err)\n+\t}\n+}\n+\n+func TestDivideZeroByNonZero(t *testing.T) {\n+\tgot, err := Divide(0, 5)\n+\tif err != nil {\n+\t\tt.Fatalf(\"Divide(0, 5) unexpected error = %v\", err)\n+\t}\n+\tif got != 0 {\n+\t\tt.Errorf(\"Divide(0, 5) = %v, want 0\", got)\n+\t}\n+}\n"

const divideDiffHeader = "--- a/calc/divide_test.go\n+++ b/calc/divide_test.go\n"

// divideCall2 is captured call 2 byte-exact (call 3 is identical). Hunk 1 is
// correct. Hunk 2 declares old line 12, but its context starts at old line 11,
// so in the shifted file it sits above its declared position.
const divideCall2 = divideDiffHeader +
	"@@ -1,6 +1,8 @@\n package calc\n \n-import \"testing\"\n+import (\n+\t\"testing\"\n+)\n \n func TestDivide(t *testing.T) {\n \tgot, err := Divide(6, 3)\n" +
	"@@ -12,3 +14,23 @@ func TestDivide(t *testing.T) {\n \t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n \t}\n }\n" +
	divideAddedDiff

// The capture gives calls 4 to 6 as placeholders, not literal text. Their
// added lines are reconstructed here as call 2's second hunk body, which is
// the span the placeholder names. The placeholder's "18" does not match that
// span. Only context, '-' lines and the header position drive the offset
// search, so the reconstruction does not change what these tests check.

// divideCall4 is a single hunk, declared old line 12, context at old line 11.
const divideCall4 = divideDiffHeader +
	"@@ -12,3 +12,21 @@ func TestDivide(t *testing.T) {\n \t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n \t}\n }\n" +
	divideAddedDiff

// divideCall5 is a single hunk, declared old line 14, context at old line 11.
const divideCall5 = divideDiffHeader +
	"@@ -14,3 +14,19 @@ func TestDivide(t *testing.T) {\n \t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n \t}\n }\n" +
	divideAddedDiff

// divideCall6 is a single hunk with only its last two context lines, declared old line 13, context at old line 12.
const divideCall6 = divideDiffHeader +
	"@@ -13,3 +13,18 @@ func TestDivide(t *testing.T) {\n \t}\n }\n" +
	divideAddedDiff

// applyFixture writes content to a file named divide_test.go in a fresh
// directory, parses diff, and applies it through ApplyToAbsPath (Stage, then
// Commit, which runs applyModify). It returns the file's bytes after the call,
// which equal content whenever the apply fails, and the apply error.
func applyFixture(t *testing.T, content, diff string) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "divide_test.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = path
	applyErr := ApplyToAbsPath(changes)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(got), applyErr
}

// expectApplied fails the test unless applyFixture applies diff to content
// and the result equals want byte for byte.
func expectApplied(t *testing.T, content, diff, want string) {
	t.Helper()
	got, err := applyFixture(t, content, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if got != want {
		t.Errorf("result mismatch\n got: %q\nwant: %q", got, want)
	}
}

// expectConflict fails the test unless applyFixture rejects diff with
// patch_conflict, leaves the file byte-identical to content, and reports
// exactly message. An empty message skips the text check.
func expectConflict(t *testing.T, content, diff, message string) {
	t.Helper()
	got, err := applyFixture(t, content, diff)
	applyErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected *ApplyError, got %T (%v)", err, err)
	}
	if applyErr.Kind != ErrPatchConflict {
		t.Errorf("expected patch_conflict, got %v", applyErr.Kind)
	}
	if message != "" && applyErr.Message != message {
		t.Errorf("message mismatch\n got: %q\nwant: %q", applyErr.Message, message)
	}
	if got != content {
		t.Errorf("file changed on a rejected apply: %q", got)
	}
}

// TestDivideBaseFixture pins divideBase against the testdata file on disk, so
// every fixture below is built from the real base and not from a stale copy.
func TestDivideBaseFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "repo-go-module", "calc", "divide_test.go"))
	if err != nil {
		t.Fatalf("read base fixture: %v", err)
	}
	if string(data) != divideBase {
		t.Errorf("divideBase differs from testdata\n got: %q\nwant: %q", data, divideBase)
	}
}

// TestApplyOffsetCall2TwoHunks applies captured call 2 through ApplyToAbsPath.
// Hunk 2 sits above its declared position, so the offset search must find it.
func TestApplyOffsetCall2TwoHunks(t *testing.T) {
	want := "package calc\n\nimport (\n\t\"testing\"\n)\n\nfunc TestDivide(t *testing.T) {\n\tgot, err := Divide(6, 3)\n\tif err != nil {\n\t\tt.Fatalf(\"Divide() error = %v\", err)\n\t}\n\tif got != 2 {\n\t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n\t}\n}\n" + divideAdded
	expectApplied(t, divideBase, divideCall2, want)
}

// TestApplyOffsetCall4ShiftMinusOne applies captured call 4: a single hunk,
// declared old line 12, context at old line 11.
func TestApplyOffsetCall4ShiftMinusOne(t *testing.T) {
	expectApplied(t, divideBase, divideCall4, divideBase+divideAdded)
}

// TestApplyOffsetCall5ShiftMinusThree applies reconstructed call 5: a single hunk,
// declared old line 14, context at old line 11, so the match lies further above
// the declared position than it does in call 4.
func TestApplyOffsetCall5ShiftMinusThree(t *testing.T) {
	expectApplied(t, divideBase, divideCall5, divideBase+divideAdded)
}

// TestApplyOffsetCall6TwoContextLines applies reconstructed call 6: a single hunk
// with only its trailing context lines, declared old line 13, context at old line 12.
func TestApplyOffsetCall6TwoContextLines(t *testing.T) {
	expectApplied(t, divideBase, divideCall6, divideBase+divideAdded)
}

// TestApplyInsertAtStartOfFile applies "-0,0": N=0 inserts before the first line.
func TestApplyInsertAtStartOfFile(t *testing.T) {
	diff := divideDiffHeader + "@@ -0,0 +1,1 @@\n+// Code generated for tests.\n"
	expectApplied(t, divideBase, diff, "// Code generated for tests.\n"+divideBase)
}

// TestApplyInsertAtEndOfFile applies "-13,0" to the base: N equal to the file
// length appends after the last line.
func TestApplyInsertAtEndOfFile(t *testing.T) {
	diff := divideDiffHeader + "@@ -13,0 +14,20 @@\n" + divideAddedDiff
	expectApplied(t, divideBase, diff, divideBase+divideAdded)
}

// TestApplyInsertPastEndOfFile applies "-14,0" to the base: N one past the file
// length is rejected, not appended. This is captured call 7's case.
func TestApplyInsertPastEndOfFile(t *testing.T) {
	diff := divideDiffHeader + "@@ -14,0 +14,20 @@\n" + divideAddedDiff
	expectConflict(t, divideBase, diff,
		"hunk 1 (old line 14): insertion runs past end of file (file has 13 lines)")
}

// TestApplyRemovalPastEndOfFile is the step-4 case: a hunk whose context lines
// fill the whole base and whose last body line removes a line past the end of
// the base. The removal must be rejected.
func TestApplyRemovalPastEndOfFile(t *testing.T) {
	ctx := " package calc\n \n import \"testing\"\n \n func TestDivide(t *testing.T) {\n \tgot, err := Divide(6, 3)\n \tif err != nil {\n \t\tt.Fatalf(\"Divide() error = %v\", err)\n \t}\n \tif got != 2 {\n \t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n \t}\n }\n"
	diff := divideDiffHeader + "@@ -1,14 +1,15 @@\n" + ctx + "+// trailer\n-\n"
	expectConflict(t, divideBase, diff,
		"hunk 1 (old line 1): removal runs past end of file (file has 13 lines); no matching position was found in the file")
}

// TestApplyRemovesLastLine is the positive control for TestApplyRemovalPastEndOfFile:
// the final line of the base is in range and its removal applies.
func TestApplyRemovesLastLine(t *testing.T) {
	diff := divideDiffHeader + "@@ -12,2 +12,1 @@\n \t}\n-}\n"
	want := "package calc\n\nimport \"testing\"\n\nfunc TestDivide(t *testing.T) {\n\tgot, err := Divide(6, 3)\n\tif err != nil {\n\t\tt.Fatalf(\"Divide() error = %v\", err)\n\t}\n\tif got != 2 {\n\t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)\n\t}\n"
	expectApplied(t, divideBase, diff, want)
}

// TestApplyNoMatchContextMismatch is a negative control: a context line that
// occurs nowhere in the file must be rejected as patch_conflict.
func TestApplyNoMatchContextMismatch(t *testing.T) {
	diff := divideDiffHeader + "@@ -5,2 +5,2 @@\n func TestDividX(t *testing.T) {\n-\tgot, err := Divide(9, 9)\n+\tgot, err := Divide(7, 7)\n"
	expectConflict(t, divideBase, diff, "")
}

// TestApplyNoMatchPastEndOfFile is a negative control: a hunk whose context runs
// past the end of the file, with no matching position anywhere, must be
// rejected as patch_conflict.
func TestApplyNoMatchPastEndOfFile(t *testing.T) {
	diff := divideDiffHeader + "@@ -13,3 +13,3 @@\n }\n X\n Y\n"
	expectConflict(t, divideBase, diff, "")
}

// TestApplyNoMatchContextMismatchMessage pins the mismatch wording for a hunk
// with no matching position: the first mismatching line is named by its
// expected and got text, and the message says the search found nothing.
func TestApplyNoMatchContextMismatchMessage(t *testing.T) {
	diff := divideDiffHeader + "@@ -5,2 +5,2 @@\n func TestDividX(t *testing.T) {\n-\tgot, err := Divide(9, 9)\n+\tgot, err := Divide(7, 7)\n"
	expectConflict(t, divideBase, diff,
		`hunk 1 (old line 5): context mismatch: expected "func TestDividX(t *testing.T) {", got "func TestDivide(t *testing.T) {"; no matching position was found in the file`)
}

// TestApplyNoMatchPastEndOfFileMessage pins the past-end wording for hunk 2,
// which must be named by its index (2), not by its NewStart (14).
func TestApplyNoMatchPastEndOfFileMessage(t *testing.T) {
	diff := divideDiffHeader +
		"@@ -1,1 +1,1 @@\n-package calc\n+package calc\n" +
		"@@ -13,3 +14,3 @@\n }\n X\n Y\n"
	expectConflict(t, divideBase, diff,
		"hunk 2 (old line 13): context runs past end of file (file has 13 lines); no matching position was found in the file")
}

// TestApplyNoFuzzTrailingSpace is a negative control for exactness: a removal
// line that differs from the file only by a trailing space must not match at
// any offset.
func TestApplyNoFuzzTrailingSpace(t *testing.T) {
	diff := divideDiffHeader + "@@ -6,1 +6,1 @@\n-\tgot, err := Divide(6, 3) \n+\tgot, err := Divide(6, 3)\n"
	expectConflict(t, divideBase, diff, "")
}

// TestApplyInsertInsidePreviousHunk applies a pure insertion whose point falls
// inside the region the previous hunk already changed. It must be rejected.
func TestApplyInsertInsidePreviousHunk(t *testing.T) {
	content := "a\nb\nc\n"
	diff := divideDiffHeader +
		"@@ -1,2 +1,2 @@\n a\n-b\n+B\n" +
		"@@ -1,0 +2,1 @@\n+x\n"
	expectConflict(t, content, diff,
		"hunk 2 (old line 1): insertion point is before the start of the file or inside the previous hunk")
}

// TestApplyInsertAdjacentToPreviousHunk applies a pure insertion whose point is
// exactly the end of the previous hunk. That position is allowed.
func TestApplyInsertAdjacentToPreviousHunk(t *testing.T) {
	content := "a\nb\nc\n"
	diff := divideDiffHeader +
		"@@ -1,1 +1,1 @@\n-a\n+A\n" +
		"@@ -1,0 +1,1 @@\n+x\n"
	expectApplied(t, content, diff, "A\nx\nb\nc\n")
}

// TestApplyOverlapWithPreviousHunkMessage pins the wording for a hunk whose
// old lines match only at a position inside the previous hunk's region. Such a
// position is never taken, so the hunk is rejected.
func TestApplyOverlapWithPreviousHunkMessage(t *testing.T) {
	content := "A\nB\nC\n"
	diff := divideDiffHeader +
		"@@ -2,1 +2,1 @@\n-B\n+B2\n" +
		"@@ -1,2 +1,2 @@\n A\n-B2\n+B3\n"
	expectConflict(t, content, diff,
		"hunk 2 (old line 1): the old lines match only inside the previous hunk")
}

// TestApplyOffsetNearestAboveWins gives one match earlier in the file, farther
// from the declared position, and one match later in it, nearer to the declared
// position. The nearer later match must be chosen. This kills earlier-first,
// farthest, and ascending first-found searches.
func TestApplyOffsetNearestAboveWins(t *testing.T) {
	content := "K\nx1\nx2\nx3\nK\nx5\nx6\nx7\n"
	diff := divideDiffHeader + "@@ -4,1 +4,1 @@\n-K\n+K2\n"
	expectApplied(t, content, diff, "K\nx1\nx2\nx3\nK2\nx5\nx6\nx7\n")
}

// TestApplyOffsetNearestBelowWins gives one match earlier in the file, nearer to
// the declared position, and one match later in it, farther from it. The nearer
// earlier match must be chosen. This kills later-first, farthest, and
// descending first-found searches.
func TestApplyOffsetNearestBelowWins(t *testing.T) {
	content := "x0\nx1\nK\nx3\nx4\nx5\nK\nx7\n"
	diff := divideDiffHeader + "@@ -4,1 +4,1 @@\n-K\n+K2\n"
	expectApplied(t, content, diff, "x0\nx1\nK2\nx3\nx4\nx5\nK\nx7\n")
}

// TestApplyOffsetBelowOnly places the only match earlier in the file than the
// declared position: the search must reach earlier lines.
func TestApplyOffsetBelowOnly(t *testing.T) {
	content := "K\nx1\nx2\nx3\nx4\nx5\nx6\nx7\n"
	diff := divideDiffHeader + "@@ -4,1 +4,1 @@\n-K\n+K2\n"
	expectApplied(t, content, diff, "K2\nx1\nx2\nx3\nx4\nx5\nx6\nx7\n")
}

// TestApplyOffsetAboveOnly places the only match later in the file than the
// declared position, well past what a search one line out would reach.
func TestApplyOffsetAboveOnly(t *testing.T) {
	content := "x0\nx1\nx2\nx3\nx4\nx5\nx6\nK\n"
	diff := divideDiffHeader + "@@ -2,1 +2,1 @@\n-K\n+K2\n"
	expectApplied(t, content, diff, "x0\nx1\nx2\nx3\nx4\nx5\nx6\nK2\n")
}

// TestApplyOffsetAmbiguousRejected gives two matches at equal distance from the
// declared position, one earlier in the file and one later. The hunk must be
// rejected as ambiguous.
func TestApplyOffsetAmbiguousRejected(t *testing.T) {
	content := "x0\nK\nx2\nx3\nx4\nK\nx6\n"
	diff := divideDiffHeader + "@@ -4,1 +4,1 @@\n-K\n+K2\n"
	expectConflict(t, content, diff,
		"hunk 1 (old line 4): ambiguous: the old lines match at lines 2 and 6")
}

// TestApplyOffsetPreviousHunkFloor applies a two-hunk patch in which hunk 2's
// nearest match sits before the end of hunk 1. That match is not allowed, so the
// farther match after hunk 1 is taken. This kills a search that ignores the floor.
func TestApplyOffsetPreviousHunkFloor(t *testing.T) {
	content := "A\nB\nC\nD\nE\nA\n"
	diff := divideDiffHeader +
		"@@ -2,1 +2,1 @@\n-B\n+B2\n" +
		"@@ -3,1 +3,1 @@\n-A\n+A2\n"
	expectApplied(t, content, diff, "A\nB2\nC\nD\nE\nA2\n")
}

// TestApplyOffsetAdjacentHunks applies two hunks where hunk 2's old lines begin
// exactly at the end of hunk 1. That position is allowed, and the search floor
// must not exclude it.
func TestApplyOffsetAdjacentHunks(t *testing.T) {
	content := "a\nb\nc\nd\n"
	diff := divideDiffHeader +
		"@@ -2,1 +2,1 @@\n-b\n+b2\n" +
		"@@ -3,1 +3,1 @@\n-c\n+c2\n"
	expectApplied(t, content, diff, "a\nb2\nc2\nd\n")
}

// TestApplyOffsetAdjacentRelocatedHunk applies a two-hunk patch in which hunk 2
// is declared one line late and its old lines begin exactly at the end of hunk 1.
// The match is not at the expected position, so it is the scan that must reach
// the first index hunk 1 leaves free.
func TestApplyOffsetAdjacentRelocatedHunk(t *testing.T) {
	content := "a\nb\nc\nd\n"
	diff := divideDiffHeader +
		"@@ -2,1 +2,1 @@\n-b\n+b2\n" +
		"@@ -4,1 +4,1 @@\n-c\n+c2\n"
	expectApplied(t, content, diff, "a\nb2\nc2\nd\n")
}

// TestApplyOffsetCarriedShift applies a two-hunk patch in which hunk 1 is found
// later in the file than its declared position. Hunk 2's declared position is in
// the same shifted frame, so the shift must carry into hunk 2's expected
// position. Y occurs at the exact expected position, and also occurs earlier and
// later in the file. A search that drops the carried shift lands on the earlier
// Y, and one that counts the shift twice lands on the later Y.
func TestApplyOffsetCarriedShift(t *testing.T) {
	content := "h1\nh2\nP\nm\nY\nn\nY\no\nY\np\n"
	diff := divideDiffHeader +
		"@@ -1,1 +1,1 @@\n-P\n+P2\n" +
		"@@ -5,1 +5,1 @@\n-Y\n+Y2\n"
	expectApplied(t, content, diff, "h1\nh2\nP2\nm\nY\nn\nY2\no\nY\np\n")
}

// numberedFile returns n lines named l00, l01 and so on, each ending in "\n",
// with the lines at the indexes in marks replaced by the given text.
func numberedFile(n int, marks map[int]string) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		if m, ok := marks[i]; ok {
			b.WriteString(m)
		} else {
			fmt.Fprintf(&b, "l%02d", i)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// TestApplyOffsetCapBoundary puts the only matching line at index 20 of a
// 40-line file and declares the hunk so its expected index is 10 or 11 lines
// away on either side. A match 10 lines from expected is applied. A match 11
// lines from expected is rejected, and the message names the match outside the
// window without applying it.
func TestApplyOffsetCapBoundary(t *testing.T) {
	content := numberedFile(40, map[int]string{20: "K"})
	applied := numberedFile(40, map[int]string{20: "K2"})
	for _, tc := range []struct {
		name     string
		oldStart int
		want     string // the result when applied, "" when rejected
		message  string // the exact conflict message when rejected
	}{
		{name: "match 10 lines after expected", oldStart: 11, want: applied},
		{name: "match 10 lines before expected", oldStart: 31, want: applied},
		{
			name: "match 11 lines after expected", oldStart: 10,
			message: `hunk 1 (old line 10): removal mismatch: expected "K", got "l09"; no matching position was found within 10 lines of line 10, but the old lines match at line 21`,
		},
		{
			name: "match 11 lines before expected", oldStart: 32,
			message: `hunk 1 (old line 32): removal mismatch: expected "K", got "l31"; no matching position was found within 10 lines of line 32, but the old lines match at line 21`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diff := divideDiffHeader + fmt.Sprintf("@@ -%d,1 +%d,1 @@\n-K\n+K2\n", tc.oldStart, tc.oldStart)
			if tc.want != "" {
				expectApplied(t, content, diff, tc.want)
				return
			}
			expectConflict(t, content, diff, tc.message)
		})
	}
}

// TestApplyOffsetCapFromExpectedPosition applies a two-hunk patch in which hunk 1
// is found 9 lines after its declared position. Hunk 2 then sits exactly 10
// lines from the position that shift implies and 19 lines from its declared
// one. It is applied, so the window is measured from the expected position and
// not from the declared one.
func TestApplyOffsetCapFromExpectedPosition(t *testing.T) {
	content := numberedFile(40, map[int]string{9: "P", 30: "Y"})
	diff := divideDiffHeader +
		"@@ -1,1 +1,1 @@\n-P\n+P2\n" +
		"@@ -12,1 +12,1 @@\n-Y\n+Y2\n"
	expectApplied(t, content, diff, numberedFile(40, map[int]string{9: "P2", 30: "Y2"}))
}

// TestApplyOffsetAmbiguousAtCapEdge puts one match 10 lines before expected and
// one 10 lines after. Both are inside the window and equally near, so the hunk is
// ambiguous and not rejected as out of range.
func TestApplyOffsetAmbiguousAtCapEdge(t *testing.T) {
	content := numberedFile(40, map[int]string{10: "K", 30: "K"})
	diff := divideDiffHeader + "@@ -21,1 +21,1 @@\n-K\n+K2\n"
	expectConflict(t, content, diff,
		"hunk 1 (old line 21): ambiguous: the old lines match at lines 11 and 31")
}

// TestApplyOffsetOverlapAndOutsideWindow applies a hunk whose old lines match at
// its expected position only inside the previous hunk, and match again 19 lines
// away. The message says both: the position inside the previous hunk, and the
// match outside the window. The farther match is not applied.
func TestApplyOffsetOverlapAndOutsideWindow(t *testing.T) {
	content := numberedFile(30, map[int]string{20: "X", 21: "l02"})
	diff := divideDiffHeader +
		"@@ -2,1 +2,1 @@\n-l01\n+X\n" +
		"@@ -2,2 +2,2 @@\n X\n-l02\n+Y\n"
	expectConflict(t, content, diff,
		"hunk 2 (old line 2): the old lines match inside the previous hunk; no matching position was found within 10 lines of line 2, but the old lines match at line 21")
}

// TestApplyOffsetExactMatchBesideNeighbours gives three adjacent matching lines.
// The one at the expected position is applied, and neither neighbour is, nor is
// the hunk ambiguous.
func TestApplyOffsetExactMatchBesideNeighbours(t *testing.T) {
	expectApplied(t, "K\nK\nK\n", divideDiffHeader+"@@ -2,1 +2,1 @@\n-K\n+K2\n", "K\nK2\nK\n")
}

// TestApplyAfterPureInsertion applies a pure insertion of two lines and then a
// hunk declared in the file's old line numbers. Three identical lines stand where
// the hunk goes, so a hunk expected one line early or late takes a neighbour of
// the correct line and the result differs.
func TestApplyAfterPureInsertion(t *testing.T) {
	content := "a\nb\nd\nd\nd\nf\n"
	diff := divideDiffHeader +
		"@@ -2,0 +3,2 @@\n+x\n+y\n" +
		"@@ -4,1 +6,1 @@\n-d\n+D\n"
	expectApplied(t, content, diff, "a\nb\nx\ny\nd\nD\nd\nf\n")
}

// TestApplyPureInsertionAfterRelocatedHunk applies a hunk found 2 lines after its
// declared position and then a pure insertion. The insertion point is carried by
// that shift and by nothing else.
func TestApplyPureInsertionAfterRelocatedHunk(t *testing.T) {
	content := "l0\nl1\nP\nl3\nl4\nl5\nl6\n"
	diff := divideDiffHeader +
		"@@ -1,1 +1,1 @@\n-P\n+P2\n" +
		"@@ -4,0 +4,1 @@\n+X\n"
	expectApplied(t, content, diff, "l0\nl1\nP2\nl3\nl4\nl5\nX\nl6\n")
}

// TestApplyOffsetCRLFShiftedHunk applies a hunk found one line before its
// declared position in a CRLF file, once with a patch that carries \r on every
// line, as a git diff of a CRLF file does, and once with a patch that carries
// none, as a model writes it. Both keep the file's CRLF endings.
func TestApplyOffsetCRLFShiftedHunk(t *testing.T) {
	content := "a\r\nb\r\nc\r\nd\r\n"
	want := "a\r\nB\r\nc\r\nd\r\n"
	t.Run("patch carries CR", func(t *testing.T) {
		expectApplied(t, content, divideDiffHeader+"@@ -3,1 +3,1 @@\n-b\r\n+B\r\n", want)
	})
	t.Run("patch carries no CR", func(t *testing.T) {
		expectApplied(t, content, divideDiffHeader+"@@ -3,1 +3,1 @@\n-b\n+B\n", want)
	})
}

// stageMoves writes content to a fresh file, parses diff, stages it and returns
// the moves the handle reports. The handle is discarded, so nothing is written.
func stageMoves(t *testing.T, content, diff string) []HunkMove {
	t.Helper()
	path := filepath.Join(t.TempDir(), "divide_test.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = path
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	moves := handle.Moves()
	if err := handle.Discard(); err != nil {
		t.Fatalf("discard failed: %v", err)
	}
	return moves
}

// TestStageMoves checks the moves Stage reports against the old line each hunk
// was applied at, in the file's line numbers before any hunk. Rows with no moves
// are the controls for rows with moves: the same arithmetic runs in both.
func TestStageMoves(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		diff    string
		want    []HunkMove
	}{
		{
			name:    "call 2: hunk 2 one line early after a hunk that grew the file",
			content: divideBase,
			diff:    divideCall2,
			want:    []HunkMove{{Hunk: 2, Declared: 12, Applied: 11}},
		},
		{
			name:    "call 4: single hunk one line early",
			content: divideBase,
			diff:    divideCall4,
			want:    []HunkMove{{Hunk: 1, Declared: 12, Applied: 11}},
		},
		{
			name:    "call 5: single hunk three lines early",
			content: divideBase,
			diff:    divideCall5,
			want:    []HunkMove{{Hunk: 1, Declared: 14, Applied: 11}},
		},
		{
			name:    "hunk later than declared reports a positive offset",
			content: "x0\nx1\nx2\nx3\nx4\nx5\nx6\nK\n",
			diff:    divideDiffHeader + "@@ -2,1 +2,1 @@\n-K\n+K2\n",
			want:    []HunkMove{{Hunk: 1, Declared: 2, Applied: 8}},
		},
		{
			name:    "control: both hunks at their declared lines after a hunk that grew the file",
			content: "a\nb\nc\nd\ne\n",
			diff: divideDiffHeader +
				"@@ -1,1 +1,3 @@\n-a\n+a1\n+a2\n+a3\n" +
				"@@ -4,1 +6,1 @@\n-d\n+D\n",
			want: nil,
		},
		{
			name:    "control: both hunks at their declared lines after a hunk that shrank the file",
			content: "a\nb\nc\nd\ne\n",
			diff: divideDiffHeader +
				"@@ -1,2 +1,1 @@\n-a\n-b\n+ab\n" +
				"@@ -4,1 +3,1 @@\n-d\n+D\n",
			want: nil,
		},
		{
			name:    "control: pure insertion and the hunk after it at their declared lines",
			content: "a\nb\nd\nd\nd\nf\n",
			diff: divideDiffHeader +
				"@@ -2,0 +3,2 @@\n+x\n+y\n" +
				"@@ -4,1 +6,1 @@\n-d\n+D\n",
			want: nil,
		},
		{
			name:    "a shift carries into the hunk after it, and both are reported",
			content: "h1\nh2\nP\nm\nY\nn\nY\no\nY\np\n",
			diff: divideDiffHeader +
				"@@ -1,1 +1,1 @@\n-P\n+P2\n" +
				"@@ -5,1 +5,1 @@\n-Y\n+Y2\n",
			want: []HunkMove{{Hunk: 1, Declared: 1, Applied: 3}, {Hunk: 2, Declared: 5, Applied: 7}},
		},
		{
			name:    "an insertion after a relocated hunk that grew the file is reported in the old file's lines",
			content: "l0\nl1\nP\nl3\nl4\nl5\nl6\n",
			diff: divideDiffHeader +
				"@@ -1,1 +1,2 @@\n-P\n+P2\n+P3\n" +
				"@@ -4,0 +5,1 @@\n+X\n",
			want: []HunkMove{{Hunk: 1, Declared: 1, Applied: 3}, {Hunk: 2, Declared: 4, Applied: 6, Insertion: true}},
		},
		{
			name:    "a pure insertion after a relocated hunk is reported as an insertion",
			content: "l0\nl1\nP\nl3\nl4\nl5\nl6\n",
			diff: divideDiffHeader +
				"@@ -1,1 +1,1 @@\n-P\n+P2\n" +
				"@@ -4,0 +4,1 @@\n+X\n",
			want: []HunkMove{{Hunk: 1, Declared: 1, Applied: 3}, {Hunk: 2, Declared: 4, Applied: 6, Insertion: true}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := stageMoves(t, tc.content, tc.diff)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("moves mismatch\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

// TestHunkMoveOffset pins the sign of Offset: negative for a hunk applied before
// its declared line, positive for one applied after it.
func TestHunkMoveOffset(t *testing.T) {
	if got := (HunkMove{Declared: 12, Applied: 11}).Offset(); got != -1 {
		t.Errorf("early hunk offset = %d, want -1", got)
	}
	if got := (HunkMove{Declared: 2, Applied: 8}).Offset(); got != 6 {
		t.Errorf("late hunk offset = %d, want 6", got)
	}
}

// TestStageMovesNamesTheChange stages a two-file patch in which only the second
// file's hunk is moved. The move must name change index 1.
func TestStageMovesNamesTheChange(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.txt")
	second := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(first, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := os.WriteFile(second, []byte("a\nb\nc\nd\n"), 0o644); err != nil {
		t.Fatalf("write second: %v", err)
	}
	diff := "--- a/first.txt\n+++ b/first.txt\n@@ -2,1 +2,1 @@\n-b\n+B\n" +
		"--- a/second.txt\n+++ b/second.txt\n@@ -4,1 +4,1 @@\n-c\n+C\n"
	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = first
	changes[1].Path = second
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	moves := handle.Moves()
	if err := handle.Discard(); err != nil {
		t.Fatalf("discard failed: %v", err)
	}
	want := []HunkMove{{Change: 1, Hunk: 1, Declared: 4, Applied: 3}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("moves mismatch\n got: %+v\nwant: %+v", moves, want)
	}
}

// TestStageMovesRenameWithEdit stages a rename that also edits, with its hunk
// one line before its declared position. The move comes through the rename path.
func TestStageMovesRenameWithEdit(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "src.txt")
	newFile := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(oldFile, []byte("a\nb\nc\nd\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	diff := "diff --git a/src.txt b/dst.txt\nrename from src.txt\nrename to dst.txt\n" +
		"--- a/src.txt\n+++ b/dst.txt\n@@ -3,1 +3,1 @@\n-b\n+B\n"
	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}
	moves := handle.Moves()
	if err := handle.Discard(); err != nil {
		t.Fatalf("discard failed: %v", err)
	}
	want := []HunkMove{{Hunk: 1, Declared: 3, Applied: 2}}
	if !reflect.DeepEqual(moves, want) {
		t.Errorf("moves mismatch\n got: %+v\nwant: %+v", moves, want)
	}
}
