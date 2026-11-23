package app

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/tui"
)

// These tests pin strings that internal/tui recognises to the code in other
// packages that produces them. internal/tui must not import internal/tool
// (internal/arch), so the pins live here, in a package that imports both.

// TestToolTruncMarkerMatchesTruncate: the line the /diff section pass treats as
// the tools' truncation marker is the line tool.Truncate writes. Entry point:
// tool.Truncate on a many-line string over its byte cap. Its output must hold
// exactly one line equal to tui.ToolTruncMarker. Positive control: Truncate
// reports that it cut, and the output holds lines from both the kept head and
// the kept tail, so the marker line sits where Truncate puts it.
func TestToolTruncMarkerMatchesTruncate(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "line %03d of the output\n", i)
	}
	out, cut := tool.Truncate(b.String(), 2000)
	if !cut {
		t.Fatal("positive control: Truncate did not cut a string over its cap")
	}
	lines := strings.Split(out, "\n")
	n := 0
	for _, ln := range lines {
		if ln == tui.ToolTruncMarker {
			n++
		}
	}
	if n != 1 {
		t.Errorf("Truncate output holds %d lines equal to tui.ToolTruncMarker %q, want 1:\n%s",
			n, tui.ToolTruncMarker, out)
	}
	if !strings.HasPrefix(out, "line 000") || !strings.Contains(out, "line 399") {
		t.Errorf("positive control: output lacks the kept head or tail:\n%s", out)
	}
}

// TestRenameTitleGivesTheNewPathToTheHighlighter: the title computeDiffDisplay
// builds for a renamed file is one the highlighter splits to the new path.
// Entry point: patch.Parse and computeDiffDisplay for a rename from a Python file
// to a Go file, then tui.DiffTitlePath, the function the diff modal's
// highlighting calls on its title to find the path its language comes from
// (that the modal draws by that path is TestApprovalDiffLanguageComesFromTheTitle
// in internal/tui). Positive control: a file that is not renamed keeps its own
// path whole.
func TestRenameTitleGivesTheNewPathToTheHighlighter(t *testing.T) {
	cases := []struct{ diff, wantPath string }{
		{"diff --git a/old.py b/new.go\nrename from old.py\nrename to new.go\n", "new.go"},
		{"--- a/calc/x.go\n+++ b/calc/x.go\n@@ -1 +1 @@\n-a\n+b\n", "calc/x.go"},
	}
	for _, c := range cases {
		changes, err := patch.Parse([]byte(c.diff))
		if err != nil || len(changes) == 0 {
			t.Fatalf("patch.Parse(%q): %d changes, err %v", c.diff, len(changes), err)
		}
		title, _, _, _ := computeDiffDisplay(changes)
		if got := tui.DiffTitlePath(title); got != c.wantPath {
			t.Errorf("title %q: highlighter path %q, want %q", title, got, c.wantPath)
		}
	}
}

// TestBenchCapsMatchTool: the three caps internal/tui's modal benchmark sizes its
// cases by are the values the tool producers cap by. internal/tui must not import
// internal/tool, in its test files either (internal/arch), so the benchmark keeps
// its own constants in internal/tui/modal_bench_test.go. An unexported constant of
// a _test.go file in package tui cannot be reached from this package, so the test
// does not call anything: it parses that source file, finds each constant by name
// and evaluates its one constant expression. Entry point: go/parser on the file,
// go/types.Eval on each expression. Positive control: every name is found, so a
// renamed constant fails the lookup and does not vanish from the comparison; the
// comparison is against tool.DefaultMaxLines, tool.MaxFileBytes and tool.MaxDiffBytes.
func TestBenchCapsMatchTool(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "../tui/modal_bench_test.go", nil, 0)
	if err != nil {
		t.Fatalf("parse internal/tui/modal_bench_test.go: %v", err)
	}
	local := map[string]string{} // constant name -> its expression's source text
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, s := range gd.Specs {
			vs := s.(*ast.ValueSpec)
			if len(vs.Names) != len(vs.Values) {
				continue // an implicit repeat of the previous expression; not used for these
			}
			for i, n := range vs.Names {
				local[n.Name] = types.ExprString(vs.Values[i])
			}
		}
	}
	for _, c := range []struct {
		name string
		want int64
	}{
		{"modalBenchFileLines", int64(tool.DefaultMaxLines)},
		{"modalBenchFileBytes", int64(tool.MaxFileBytes)},
		{"modalBenchDiffBytes", int64(tool.MaxDiffBytes)},
	} {
		expr, found := local[c.name]
		if !found {
			t.Errorf("internal/tui/modal_bench_test.go declares no constant %s", c.name)
			continue
		}
		tv, err := types.Eval(token.NewFileSet(), nil, token.NoPos, expr)
		if err != nil || tv.Value == nil {
			t.Errorf("%s = %q is not a constant expression: %v", c.name, expr, err)
			continue
		}
		got, exact := constant.Int64Val(tv.Value)
		if !exact || got != c.want {
			t.Errorf("%s = %s = %v, but the tool package's cap is %d", c.name, expr, tv.Value, c.want)
		}
	}
}
