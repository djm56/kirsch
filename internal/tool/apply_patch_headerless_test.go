package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/policy"
)

// liveCall3Quoted is the exact Go %q string of model call 3: a hunk whose final
// line is a lone space with no newline after it. It is the same string as the
// one in internal/patch/parse_live_test.go.
const liveCall3Quoted = `"--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -6,3 +6,6 @@\n // here, or the smoke test has nothing to do.\n func Divide(a, b float64) (float64, error) {\n+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n \treturn a / b, nil\n }\n "`

// liveCall3Want is liveDivideGo after call 3 applies: the guard sits after line 7.
const liveCall3Want = "package calc\n\n// Divide returns a divided by b.\n//\n// It has no input validation. Milestone 4's smoke test adds it; do not fix it\n// here, or the smoke test has nothing to do.\nfunc Divide(a, b float64) (float64, error) {\n\tif b == 0 {\n\t\treturn 0, ErrDivideByZero\n\t}\n\treturn a / b, nil\n}\n"

// liveCall5Quoted is the exact Go %q string of model call 5: a hunk with no
// ---/+++ file header. It is decoded with strconv.Unquote in the test.
const liveCall5Quoted = `"@@ -7,2 +7,5 @@ func Divide(a, b float64) (float64, error) {\n+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n \treturn a / b, nil\n }\n"`

// liveCall7Quoted is the exact Go %q string of model call 7: an unprefixed
// "// Divide returns a divided by b." line inside the hunk body, at line 11.
const liveCall7Quoted = `"--- a/calc/divide.go\n+++ b/calc/divide.go\n@@ -1,3 +1,8 @@\n package calc\n \n+import \"errors\"\n+\n+// ErrDivideByZero is returned when division by zero is attempted.\n+var ErrDivideByZero = errors.New(\"division by zero\")\n+\n// Divide returns a divided by b.\n-func Divide(a, b float64) (float64, error) {\n-\treturn a / b, nil\n+//\n+// It returns ErrDivideByZero if b is zero.\n+func Divide(a, b float64) (float64, error) {\n+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n+\treturn a / b, nil\n }\n"`

// liveDivideGo is testdata/repo-go-module/calc/divide.go, ending "}\n".
const liveDivideGo = "package calc\n\n// Divide returns a divided by b.\n//\n// It has no input validation. Milestone 4's smoke test adds it; do not fix it\n// here, or the smoke test has nothing to do.\nfunc Divide(a, b float64) (float64, error) {\n\treturn a / b, nil\n}\n"

// invokeLiveDivide writes calc/divide.go into a fresh workspace, then runs diff
// through Invoke with the given approver. It returns the result and the file's
// bytes after the call.
func invokeLiveDivide(t *testing.T, approver *fakeApprover, diff string) (Result, string) {
	t.Helper()
	ws := newTestWorkspace(t)
	if err := os.MkdirAll(filepath.Join(ws.Root, "calc"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(ws.Root, "calc", "divide.go")
	if err := os.WriteFile(path, []byte(liveDivideGo), 0o644); err != nil {
		t.Fatalf("write divide.go: %v", err)
	}
	tool := &ApplyPatch{WS: ws, Approver: approver}
	raw, _ := json.Marshal(applyPatchInput{Diff: diff, Description: "test"})
	result := tool.Invoke(context.Background(), raw)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read divide.go: %v", err)
	}
	return result, string(got)
}

// TestApplyPatchCall3WhitespaceFragmentApplies covers call 3 end to end. The
// lone-space final fragment is ignored, the guard is written after line 7, and
// the tool reports the plain success text after approval.
func TestApplyPatchCall3WhitespaceFragmentApplies(t *testing.T) {
	call3, err := strconv.Unquote(liveCall3Quoted)
	if err != nil {
		t.Fatalf("unquote: %v", err)
	}
	approver := newFakeApprover(policy.DecisionAllow)
	result, got := invokeLiveDivide(t, approver, call3)

	if !result.OK {
		t.Fatalf("expected OK, got error %+v", result.Error)
	}
	if !approver.called {
		t.Errorf("approver was not called for call 3")
	}
	if result.Content != "patch applied" {
		t.Errorf("content %q, want %q", result.Content, "patch applied")
	}
	if got != liveCall3Want {
		t.Errorf("file mismatch\n got: %q\nwant: %q", got, liveCall3Want)
	}
}

// TestApplyPatchHunkWithoutFileHeaderNeverApplies covers a hunk with no file
// header. The tool must fail it before approval, print no success text, and
// leave the file unchanged.
func TestApplyPatchHunkWithoutFileHeaderNeverApplies(t *testing.T) {
	call5, err := strconv.Unquote(liveCall5Quoted)
	if err != nil {
		t.Fatalf("unquote: %v", err)
	}
	approver := newFakeApprover(policy.DecisionAllow)
	result, got := invokeLiveDivide(t, approver, call5)

	if result.OK {
		t.Fatalf("expected failure, got OK with content %q", result.Content)
	}
	if result.Error == nil || result.Error.Kind != KindToolInputInvalid {
		t.Fatalf("expected KindToolInputInvalid, got %+v", result.Error)
	}
	if !strings.Contains(result.Error.Message, "hunk at line 1 has no file header") {
		t.Errorf("message %q does not name the missing header", result.Error.Message)
	}
	if approver.called {
		t.Errorf("approver was called for a hunk with no file header")
	}
	if got != liveDivideGo {
		t.Errorf("file changed on a rejected patch: %q", got)
	}
	if strings.Contains(result.Content, "patch applied") || strings.Contains(result.DisplaySummary, "patch applied") {
		t.Errorf("success text printed for a rejected patch: content %q, summary %q", result.Content, result.DisplaySummary)
	}
}

// TestApplyPatchHunkWithHeadersStillApplies is the mirror of the test above: the
// same hunk after its file headers is accepted, goes to approval, and changes the file.
func TestApplyPatchHunkWithHeadersStillApplies(t *testing.T) {
	call5, err := strconv.Unquote(liveCall5Quoted)
	if err != nil {
		t.Fatalf("unquote: %v", err)
	}
	approver := newFakeApprover(policy.DecisionAllow)
	diff := "--- a/calc/divide.go\n+++ b/calc/divide.go\n" + call5
	result, got := invokeLiveDivide(t, approver, diff)

	if !result.OK {
		t.Fatalf("expected OK, got error %+v", result.Error)
	}
	if !approver.called {
		t.Errorf("approver was not called for a patch with headers")
	}
	if !strings.Contains(got, "\tif b == 0 {") {
		t.Errorf("guard not written to file: %q", got)
	}
}

// TestApplyPatchZeroFileChangesNeverApplies covers input with no file change in
// it. The tool must not return success for it, and must not ask for approval.
func TestApplyPatchZeroFileChangesNeverApplies(t *testing.T) {
	approver := newFakeApprover(policy.DecisionAllow)
	result, got := invokeLiveDivide(t, approver, "Here is the patch, nothing else.\n")

	if result.OK {
		t.Fatalf("expected failure, got OK with content %q", result.Content)
	}
	if result.Error == nil || result.Error.Kind != KindToolInputInvalid {
		t.Fatalf("expected KindToolInputInvalid, got %+v", result.Error)
	}
	if approver.called {
		t.Errorf("approver was called for input with no file changes")
	}
	if got != liveDivideGo {
		t.Errorf("file changed on a rejected patch: %q", got)
	}
	if strings.Contains(result.Content, "patch applied") || strings.Contains(result.DisplaySummary, "patch applied") {
		t.Errorf("success text printed for a rejected patch: content %q, summary %q", result.Content, result.DisplaySummary)
	}
}

// TestApplyPatchUnprefixedHunkLineNeverApplies covers call 7, a hunk body with
// an unprefixed line at line 11 and body lines after it. The tool must fail it
// naming the line, before approval, print no success text, and leave the file
// unchanged.
func TestApplyPatchUnprefixedHunkLineNeverApplies(t *testing.T) {
	call7, err := strconv.Unquote(liveCall7Quoted)
	if err != nil {
		t.Fatalf("unquote: %v", err)
	}
	approver := newFakeApprover(policy.DecisionAllow)
	result, got := invokeLiveDivide(t, approver, call7)

	if result.OK {
		t.Fatalf("expected failure, got OK with content %q", result.Content)
	}
	if result.Error == nil || result.Error.Kind != KindToolInputInvalid {
		t.Fatalf("expected KindToolInputInvalid, got %+v", result.Error)
	}
	if !strings.Contains(result.Error.Message, "line 11 is not a hunk line") {
		t.Errorf("message %q does not name line 11", result.Error.Message)
	}
	if approver.called {
		t.Errorf("approver was called for a hunk with an unprefixed line")
	}
	if got != liveDivideGo {
		t.Errorf("file changed on a rejected patch: %q", got)
	}
	if strings.Contains(result.Content, "patch applied") || strings.Contains(result.DisplaySummary, "patch applied") {
		t.Errorf("success text printed for a rejected patch: content %q, summary %q", result.Content, result.DisplaySummary)
	}
}
