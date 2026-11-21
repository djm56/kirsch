package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/workspace"
)

// The screen captures under plan/testing/manual/captures are drawn from canned
// inputs in internal/tui/capture_test.go. internal/tui must not import
// internal/tool (internal/arch), so these tests, in a package that imports both,
// hold the real producers to the literals and patterns that
// TestCaptureInputFormats in internal/tui holds the canned inputs to. Keep the two
// in step: a change to one set of literals is a change to the other.

// TestCaptureInputsMatchProducers: entry points are the real read_file tool's
// Invoke on a temporary file, and GetDiff on a temporary git repo with one
// modified file. The tool's DisplaySummary and Content for "a\nb\n" must equal
// the literals the capture test pins capReadFile to, and the first five rows of the
// real diff must match the patterns the capture test pins capDiff to. Positive
// control: a one-line file gives the range "lines 1-1", and the diff has more rows
// than the five header patterns, so the patterns were read off a real diff with a
// hunk. The committed scrollbar-bottom capture must show a summary of the real
// shape ("lines 1-N") on its read_file row.
func TestCaptureInputsMatchProducers(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"two.txt": "a\nb\n", "one.txt": "a\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := workspace.Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	rf := &tool.ReadFile{WS: ws}
	read := func(path string) tool.Result {
		in, err := json.Marshal(map[string]string{"path": path})
		if err != nil {
			t.Fatal(err)
		}
		res := rf.Invoke(context.Background(), in)
		if !res.OK {
			t.Fatalf("read_file %s failed: %+v", path, res.Error)
		}
		return res
	}
	two := read("two.txt")
	if two.DisplaySummary != "lines 1-2" || two.Content != "1\ta\n2\tb\n" {
		t.Errorf("read_file gave summary %q content %q, want \"lines 1-2\" and \"1\\ta\\n2\\tb\\n\" (the capture test's literals)",
			two.DisplaySummary, two.Content)
	}
	if one := read("one.txt"); one.DisplaySummary != "lines 1-1" {
		t.Errorf("control: one-line file summary %q, want \"lines 1-1\"", one.DisplaySummary)
	}

	// The same patterns as capDiffHeaderShapes in internal/tui/capture_test.go.
	shapes := []string{
		`^diff --git a/\S+ b/\S+$`,
		`^index [0-9a-f]{7,}\.\.[0-9a-f]{7,} [0-7]{6}$`,
		`^--- a/\S+$`,
		`^\+\+\+ b/\S+$`,
		`^@@ -\d+,\d+ \+\d+,\d+ @@`,
	}
	repo := newGitRepo(t, map[string]string{"test.txt": "line1\nline2\nline3\n"})
	writeRepoFile(t, repo, "test.txt", "line1\nline2 changed\nline3\nline4\n")
	res := newDiffApp(repo).GetDiff()
	if res.Err != "" {
		t.Fatalf("GetDiff: %s", res.Err)
	}
	if len(res.Lines) <= len(shapes) {
		t.Fatalf("control: the real diff has %d rows, want more than the %d header rows", len(res.Lines), len(shapes))
	}
	for i, shape := range shapes {
		if !regexp.MustCompile(shape).MatchString(res.Lines[i]) {
			t.Errorf("real git diff row %d is %q, want it to match %s", i, res.Lines[i], shape)
		}
	}

	// The committed capture shows the real summary shape: the digits of the tool's
	// own summary replaced by \d+.
	shape := regexp.MustCompile(`\d+`).ReplaceAllString(two.DisplaySummary, `\d+`)
	committed, err := os.ReadFile(filepath.Join("..", "..", "plan", "testing", "manual", "captures", "v0.1.5", "scrollbar-bottom.txt"))
	if err != nil {
		t.Fatalf("read committed capture: %v", err)
	}
	row := regexp.MustCompile(`read_file \S+ · ` + shape + ` · `)
	if !row.Match(committed) {
		t.Errorf("scrollbar-bottom.txt has no read_file row with a summary of the real shape %q", shape)
	}
	if strings.Contains(string(committed), " · read · ") {
		t.Error("scrollbar-bottom.txt still shows the bare summary \"read\"")
	}
}
