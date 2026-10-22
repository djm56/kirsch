package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/policy"
)

// invokePatch applies diff to files in a fresh workspace through Invoke, with an
// approver that allows it, and returns the result.
func invokePatch(t *testing.T, files map[string]string, diff string) Result {
	t.Helper()
	ws := newTestWorkspace(t)
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(ws.Root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	tool := &ApplyPatch{WS: ws, Approver: newFakeApprover(policy.DecisionAllow)}
	raw, _ := json.Marshal(applyPatchInput{Diff: diff, Description: "test"})
	return tool.Invoke(context.Background(), raw)
}

// TestApplyPatchResultReportsMovedHunks checks the success text for patches that
// land away from their declared lines, and for one that does not. The unmoved row
// is the control: the same tool, file and hunk shape give plain "patch applied".
func TestApplyPatchResultReportsMovedHunks(t *testing.T) {
	const header = "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n"
	for _, tc := range []struct {
		name  string
		files map[string]string
		diff  string
		want  string
	}{
		{
			name:  "control: hunk at its declared line",
			files: map[string]string{"file.txt": "line1\nline2\nline3\nline4\n"},
			diff:  header + "@@ -2,1 +2,1 @@\n-line2\n+X\n",
			want:  "patch applied",
		},
		{
			name:  "hunk one line before its declared line",
			files: map[string]string{"file.txt": "line1\nline2\nline3\nline4\n"},
			diff:  header + "@@ -3,1 +3,1 @@\n-line2\n+X\n",
			want:  "patch applied; file.txt hunk 1 applied at old line 2 (declared 3, offset -1)",
		},
		{
			name:  "hunk after its declared line",
			files: map[string]string{"file.txt": "line1\nline2\nline3\nline4\n"},
			diff:  header + "@@ -1,1 +1,1 @@\n-line3\n+X\n",
			want:  "patch applied; file.txt hunk 1 applied at old line 3 (declared 1, offset +2)",
		},
		{
			name:  "second hunk moved, first not",
			files: map[string]string{"file.txt": "line1\nline2\nline3\nline4\n"},
			diff:  header + "@@ -1,1 +1,1 @@\n-line1\n+A\n@@ -4,1 +4,1 @@\n-line3\n+C\n",
			want:  "patch applied; file.txt hunk 2 applied at old line 3 (declared 4, offset -1)",
		},
		{
			name:  "pure insertion carried by an earlier shift",
			files: map[string]string{"file.txt": "l0\nl1\nP\nl3\nl4\nl5\nl6\n"},
			diff:  header + "@@ -1,1 +1,1 @@\n-P\n+P2\n@@ -4,0 +4,1 @@\n+X\n",
			want:  "patch applied; file.txt hunk 1 applied at old line 3 (declared 1, offset +2); file.txt hunk 2 applied after old line 6 (declared 4, offset +2)",
		},
		{
			name:  "move in the second file names the second file",
			files: map[string]string{"one.txt": "a\nb\nc\n", "two.txt": "a\nb\nc\nd\n"},
			diff: "--- a/one.txt\n+++ b/one.txt\n@@ -2,1 +2,1 @@\n-b\n+B\n" +
				"--- a/two.txt\n+++ b/two.txt\n@@ -4,1 +4,1 @@\n-c\n+C\n",
			want: "patch applied; two.txt hunk 1 applied at old line 3 (declared 4, offset -1)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := invokePatch(t, tc.files, tc.diff)
			if !result.OK {
				t.Fatalf("expected OK, got error: %s", result.Error.Message)
			}
			if result.Content != tc.want {
				t.Errorf("content mismatch\n got: %q\nwant: %q", result.Content, tc.want)
			}
			if result.DisplaySummary != "applied" {
				t.Errorf("display summary = %q, want %q", result.DisplaySummary, "applied")
			}
		})
	}
}

// TestApplyPatchWindowConflictMessage drives a hunk whose only match is 11 lines
// from its expected position through Invoke. It is a patch_conflict whose text
// names the window and the match outside it.
func TestApplyPatchWindowConflictMessage(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		if i == 20 {
			b.WriteString("K\n")
		} else {
			fmt.Fprintf(&b, "l%02d\n", i)
		}
	}
	diff := "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -32,1 +32,1 @@\n-K\n+K2\n"
	result := invokePatch(t, map[string]string{"file.txt": b.String()}, diff)
	if result.OK {
		t.Fatalf("expected failure, got OK: %q", result.Content)
	}
	if result.Error.Kind != KindPatchConflict {
		t.Fatalf("expected KindPatchConflict, got %s", result.Error.Kind)
	}
	want := `patch cannot be applied: hunk 1 (old line 32): removal mismatch: expected "K", got "l31"; no matching position was found within 10 lines of line 32, but the old lines match at line 21`
	if result.Error.Message != want {
		t.Errorf("message mismatch\n got: %q\nwant: %q", result.Error.Message, want)
	}
}
