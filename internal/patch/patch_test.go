package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseCorpusFiles tests all 20 corpus files with expected outcomes.
func TestParseCorpusFiles(t *testing.T) {
	type testCase struct {
		name     string
		wantErr  bool
		errKind  ParseErrorKind
		validate func(*testing.T, []FileChange)
	}

	tests := []testCase{
		// Parse succeeds (17 files)
		{
			name:    "binary.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if !changes[0].IsBinary {
					t.Error("expected IsBinary=true")
				}
			},
		},
		{
			name:    "chmod-exec.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if changes[0].NewMode != 0o755 {
					t.Errorf("expected NewMode=0o755, got %o", changes[0].NewMode)
				}
			},
		},
		{
			name:    "context-mismatch.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
			},
		},
		{
			name:    "create-file.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if changes[0].Op != OpCreate {
					t.Errorf("expected OpCreate, got %v", changes[0].Op)
				}
			},
		},
		{
			name:    "crlf-converting.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
			},
		},
		{
			name:    "crlf-preserving.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
			},
		},
		{
			name:    "delete-file.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if changes[0].Op != OpDelete {
					t.Errorf("expected OpDelete, got %v", changes[0].Op)
				}
			},
		},
		{
			name:    "denied-path.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
			},
		},
		{
			name:    "escape-path.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
			},
		},
		{
			name:    "modify-multi-hunk.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if len(changes[0].Hunks) != 2 {
					t.Errorf("expected 2 hunks, got %d", len(changes[0].Hunks))
				}
			},
		},
		{
			name:    "modify-single-hunk.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if len(changes[0].Hunks) != 1 {
					t.Errorf("expected 1 hunk, got %d", len(changes[0].Hunks))
				}
			},
		},
		{
			name:    "multi-file.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 3 {
					t.Fatalf("expected 3 changes, got %d", len(changes))
				}
			},
		},
		{
			name:    "multi-file-second-conflicts.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 3 {
					t.Fatalf("expected 3 changes, got %d", len(changes))
				}
			},
		},
		{
			name:    "no-eol-add.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				// Check that the "\ No newline" marker is preserved
				if len(changes[0].Hunks) > 0 {
					hasBackslash := false
					for _, line := range changes[0].Hunks[0].Lines {
						if line.Prefix == '\\' {
							hasBackslash = true
						}
					}
					if !hasBackslash {
						t.Error("expected backslash marker for no-eol")
					}
				}
			},
		},
		{
			name:    "no-eol-remove.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
			},
		},
		{
			name:    "rename.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if changes[0].Op != OpRename {
					t.Errorf("expected OpRename, got %v", changes[0].Op)
				}
				if len(changes[0].Hunks) != 0 {
					t.Errorf("expected 0 hunks for rename, got %d", len(changes[0].Hunks))
				}
			},
		},
		{
			name:    "rename-with-edit.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 {
					t.Fatalf("expected 1 change, got %d", len(changes))
				}
				if changes[0].Op != OpRename {
					t.Errorf("expected OpRename, got %v", changes[0].Op)
				}
				if len(changes[0].Hunks) == 0 {
					t.Error("expected hunks for rename-with-edit")
				}
			},
		},

		// Parse fails (3 files)
		{
			name:    "chmod-other.diff",
			wantErr: true,
			errKind: ErrBadMode,
			validate: func(t *testing.T, changes []FileChange) {
				// Should not reach here on error
			},
		},
		{
			name:    "malformed-header.diff",
			wantErr: true,
			errKind: ErrMissingHeader,
			validate: func(t *testing.T, changes []FileChange) {
				// Should not reach here on error
			},
		},
		// Recounted, not rejected: the header declares 5 old and 5 new lines,
		// but the body holds 3 of each, and the hunk parses to the counted values.
		{
			name:    "malformed-hunk-counts.diff",
			wantErr: false,
			validate: func(t *testing.T, changes []FileChange) {
				if len(changes) != 1 || len(changes[0].Hunks) != 1 {
					t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
				}
				if h := changes[0].Hunks[0]; h.OldLines != 3 || h.NewLines != 3 {
					t.Errorf("expected counted old 3, new 3, got old %d, new %d", h.OldLines, h.NewLines)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("../../testdata/patches", tc.name))
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			changes, err := Parse(data)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				pErr, ok := err.(*ParseError)
				if !ok {
					t.Errorf("expected ParseError but got %T: %v", err, err)
					return
				}
				if pErr.Kind != tc.errKind {
					t.Errorf("expected error kind %v but got %v", tc.errKind, pErr.Kind)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			tc.validate(t, changes)
		})
	}
}

// TestRoundTrip tests that parsing and re-rendering produces identical output.
func TestRoundTrip(t *testing.T) {
	testFiles := []string{
		"binary.diff",
		"chmod-exec.diff",
		"context-mismatch.diff",
		"create-file.diff",
		"crlf-converting.diff",
		"crlf-preserving.diff",
		"delete-file.diff",
		"denied-path.diff",
		"escape-path.diff",
		"modify-multi-hunk.diff",
		"modify-single-hunk.diff",
		"multi-file.diff",
		"multi-file-second-conflicts.diff",
		"no-eol-add.diff",
		"no-eol-remove.diff",
		"rename.diff",
		"rename-with-edit.diff",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("../../testdata/patches", filename))
			if err != nil {
				t.Fatalf("failed to read test file: %v", err)
			}

			// Parse the diff
			changes, err := Parse(original)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			// Re-render it
			rendered := Render(changes)

			// Compare byte-for-byte
			if string(original) != string(rendered) {
				t.Errorf("round-trip mismatch\noriginal:\n%s\nrendered:\n%s",
					string(original), string(rendered))
			}
		})
	}
}

// TestHunkLinePreservation tests that hunk line prefixes are preserved.
func TestHunkLinePreservation(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,3 @@
 context line
-removed line
+added line
+another added line
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(changes[0].Hunks))
	}

	hunk := changes[0].Hunks[0]
	if len(hunk.Lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(hunk.Lines))
	}

	tests := []struct {
		idx         int
		expectedKey byte
	}{
		{0, ' '}, // context
		{1, '-'}, // removed
		{2, '+'}, // added
		{3, '+'}, // another added
	}

	for _, tc := range tests {
		if hunk.Lines[tc.idx].Prefix != tc.expectedKey {
			t.Errorf("line %d: expected prefix %c, got %c", tc.idx, tc.expectedKey, hunk.Lines[tc.idx].Prefix)
		}
	}
}

// TestParseCreateFileWithNewFileMode tests that new file mode is recognized.
func TestParseCreateFileWithNewFileMode(t *testing.T) {
	diff := `diff --git a/newfile.txt b/newfile.txt
new file mode 100644
--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,1 @@
+This is new
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Op != OpCreate {
		t.Errorf("expected OpCreate, got %v", changes[0].Op)
	}
}

// TestParseDeleteFileWithDeletedFileMode tests that deleted file mode is recognized.
func TestParseDeleteFileWithDeletedFileMode(t *testing.T) {
	diff := `diff --git a/oldfile.txt b/oldfile.txt
deleted file mode 100644
--- a/oldfile.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-This is deleted
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Op != OpDelete {
		t.Errorf("expected OpDelete, got %v", changes[0].Op)
	}
}

// TestPathStripsPrefixes tests that a/ and b/ prefixes are removed.
func TestPathStripsPrefixes(t *testing.T) {
	diff := `diff --git a/some/path.txt b/some/path.txt
--- a/some/path.txt
+++ b/some/path.txt
@@ -1,1 +1,1 @@
-old
+new
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if changes[0].Path != "some/path.txt" {
		t.Errorf("expected path 'some/path.txt', got '%s'", changes[0].Path)
	}
}

// TestBinaryFilesMarker tests that "Binary files ... differ" is parsed correctly.
func TestBinaryFilesMarker(t *testing.T) {
	diff := `diff --git a/binary.dat b/binary.dat
--- a/binary.dat
+++ b/binary.dat
Binary files a/binary.dat and b/binary.dat differ
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !changes[0].IsBinary {
		t.Error("expected IsBinary=true")
	}

	if len(changes[0].Hunks) != 0 {
		t.Errorf("expected 0 hunks for binary file, got %d", len(changes[0].Hunks))
	}
}

// TestHunkArithmeticValidation tests that a hunk with no body is rejected with anchored messages.
// Header counts that disagree with a non-empty body are recounted, not rejected (see TestRecount*).
func TestHunkArithmeticValidation(t *testing.T) {
	// This hunk declares 2 old and 3 new lines and has no body, so there is nothing to recount
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,3 @@
`

	_, err := Parse([]byte(diff))
	if err == nil {
		t.Error("expected error for hunk count mismatch")
		return
	}

	pErr, ok := err.(*ParseError)
	if !ok {
		t.Errorf("expected ParseError, got %T", err)
		return
	}

	if pErr.Kind != ErrHunkCountMismatch {
		t.Errorf("expected ErrHunkCountMismatch, got %v", pErr.Kind)
	}

	// Assert message content includes file path and hunk signature
	if !strings.Contains(pErr.Message, "a/file.txt") {
		t.Errorf("expected error message to contain file path 'a/file.txt', got: %s", pErr.Message)
	}
	if !strings.Contains(pErr.Message, "@@") {
		t.Errorf("expected error message to contain hunk signature '@@', got: %s", pErr.Message)
	}
}

// TestNoNewlineAtEndOfFile tests the special "\ No newline" marker.
func TestNoNewlineAtEndOfFile(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,1 +1,1 @@
-line without newline
\ No newline at end of file
+line with newline
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	hunk := changes[0].Hunks[0]

	// Find the backslash marker
	found := false
	for _, line := range hunk.Lines {
		if line.Prefix == '\\' && line.Content == `No newline at end of file` {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected to find backslash marker for no newline at end of file")
	}
}

// TestRenameOperation tests that rename operations are parsed correctly.
func TestRenameOperation(t *testing.T) {
	diff := `diff --git a/oldname.txt b/newname.txt
rename from oldname.txt
rename to newname.txt
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if changes[0].Op != OpRename {
		t.Errorf("expected OpRename, got %v", changes[0].Op)
	}

	if changes[0].OldPath != "oldname.txt" {
		t.Errorf("expected OldPath='oldname.txt', got '%s'", changes[0].OldPath)
	}

	if changes[0].Path != "newname.txt" {
		t.Errorf("expected Path='newname.txt', got '%s'", changes[0].Path)
	}
}

// TestModeChangeExecBit tests that 100644 ↔ 100755 changes are honored.
func TestModeChangeExecBit(t *testing.T) {
	diff := `diff --git a/script.sh b/script.sh
old mode 100644
new mode 100755
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if changes[0].NewMode != 0o755 {
		t.Errorf("expected NewMode=0o755, got %o", changes[0].NewMode)
	}
}

// TestModeChangeUnsupported tests that non-exec mode changes are rejected.
func TestModeChangeUnsupported(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
old mode 100644
new mode 100600
`

	_, err := Parse([]byte(diff))
	if err == nil {
		t.Error("expected error for unsupported mode change")
		return
	}

	pErr, ok := err.(*ParseError)
	if !ok {
		t.Errorf("expected ParseError, got %T", err)
		return
	}

	if pErr.Kind != ErrBadMode {
		t.Errorf("expected ErrBadMode, got %v", pErr.Kind)
	}
}

// TestMultiFileChanges tests parsing multiple files in one diff.
func TestMultiFileChanges(t *testing.T) {
	diff := `diff --git a/file1.txt b/file1.txt
--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old1
+new1
diff --git a/file2.txt b/file2.txt
--- a/file2.txt
+++ b/file2.txt
@@ -1,1 +1,1 @@
-old2
+new2
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	if changes[0].Path != "file1.txt" {
		t.Errorf("expected first file to be file1.txt, got %s", changes[0].Path)
	}

	if changes[1].Path != "file2.txt" {
		t.Errorf("expected second file to be file2.txt, got %s", changes[1].Path)
	}
}

// TestEmptyLinesInDiff tests handling of empty lines in diff.
func TestEmptyLinesInDiff(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line1
-removed
+added
 line2
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes[0].Hunks[0].Lines) != 4 {
		t.Errorf("expected 4 lines, got %d", len(changes[0].Hunks[0].Lines))
	}
}

// TestHeaderlessDiffModify tests parsing a header-less modify diff.
func TestHeaderlessDiffModify(t *testing.T) {
	diff := `--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,2 @@
 context
-old
+new
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Op != OpModify {
		t.Errorf("expected OpModify, got %v", change.Op)
	}
	if change.Path != "file.txt" {
		t.Errorf("expected path 'file.txt', got '%s'", change.Path)
	}
	if change.HasHeader {
		t.Errorf("expected HasHeader=false for header-less diff")
	}
	if len(change.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(change.Hunks))
	}
}

// TestHeaderlessDiffCreate tests parsing a header-less create diff.
func TestHeaderlessDiffCreate(t *testing.T) {
	diff := `--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,1 @@
+new content
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Op != OpCreate {
		t.Errorf("expected OpCreate, got %v", change.Op)
	}
	if change.Path != "newfile.txt" {
		t.Errorf("expected path 'newfile.txt', got '%s'", change.Path)
	}
	if change.HasHeader {
		t.Errorf("expected HasHeader=false for header-less diff")
	}
}

// TestHeaderlessDiffDelete tests parsing a header-less delete diff.
func TestHeaderlessDiffDelete(t *testing.T) {
	diff := `--- a/oldfile.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-old content
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Op != OpDelete {
		t.Errorf("expected OpDelete, got %v", change.Op)
	}
	if change.Path != "oldfile.txt" {
		t.Errorf("expected path 'oldfile.txt', got '%s'", change.Path)
	}
}

// TestHeaderlessDiffRoundTrip tests that a header-less diff round-trips correctly.
func TestHeaderlessDiffRoundTrip(t *testing.T) {
	original := `--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,2 @@
 context
-old
+new
`

	changes, err := Parse([]byte(original))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	rendered := Render(changes)

	if original != string(rendered) {
		t.Errorf("round-trip mismatch\noriginal:\n%s\nrendered:\n%s",
			original, string(rendered))
	}
}

// TestHunkBodyHeaderLike tests that a --- line inside a hunk body is not mistaken
// for a file section header. It asserts line count, prefix, and content directly.
func TestHunkBodyHeaderLike(t *testing.T) {
	// This diff has a line that reads "--- a/foo.txt" inside the hunk body.
	// The hunk declares 4 old lines and 3 new lines.
	// Structure: 2 context lines, 2 removals (one of which looks like a file header),
	// then 1 addition.
	// That's 4 old (2 context + 2 removals) and 3 new (2 context + 1 addition).
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,4 +1,3 @@
 context line 1
 context line 2
-This is a line
---- a/foo.txt
+This is replacement
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if len(change.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(change.Hunks))
	}

	hunk := change.Hunks[0]

	// Verify hunk metadata
	if hunk.OldLines != 4 {
		t.Errorf("expected OldLines=4, got %d", hunk.OldLines)
	}
	if hunk.NewLines != 3 {
		t.Errorf("expected NewLines=3, got %d", hunk.NewLines)
	}

	// Verify line count
	if len(hunk.Lines) != 5 {
		t.Fatalf("expected 5 hunk lines, got %d", len(hunk.Lines))
	}

	// Verify each line's prefix and content
	expectedLines := []struct {
		prefix  byte
		content string
	}{
		{' ', "context line 1"},
		{' ', "context line 2"},
		{'-', "This is a line"},
		{'-', "--- a/foo.txt"},
		{'+', "This is replacement"},
	}

	for i, expected := range expectedLines {
		if hunk.Lines[i].Prefix != expected.prefix {
			t.Errorf("line %d: expected prefix %c, got %c", i, expected.prefix, hunk.Lines[i].Prefix)
		}
		if hunk.Lines[i].Content != expected.content {
			t.Errorf("line %d: expected content %q, got %q", i, expected.content, hunk.Lines[i].Content)
		}
	}
}

// TestHeaderlessCreateWithMode tests preserving mode in header-less create.
func TestHeaderlessCreateWithMode(t *testing.T) {
	// Header-less diffs don't have "new file mode" lines, so mode is not preserved
	// This test just verifies the create operation is detected correctly
	diff := `--- /dev/null
+++ b/script.sh
@@ -0,0 +1,1 @@
+#!/bin/bash
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if changes[0].Op != OpCreate {
		t.Errorf("expected OpCreate, got %v", changes[0].Op)
	}
	// For header-less creates, CreateMode should not be set (no mode information available)
	if changes[0].CreateMode != 0 {
		t.Errorf("expected CreateMode=0 for header-less create, got %o", changes[0].CreateMode)
	}
}

// TestCreateFileMode100755 tests that "new file mode 100755" is preserved (W2 fix).
func TestCreateFileMode100755(t *testing.T) {
	diff := `diff --git a/script.sh b/script.sh
new file mode 100755
--- /dev/null
+++ b/script.sh
@@ -0,0 +1,1 @@
+#!/bin/bash
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.Op != OpCreate {
		t.Errorf("expected OpCreate, got %v", change.Op)
	}
	if change.CreateMode != 0o755 {
		t.Errorf("expected CreateMode=0o755, got %o", change.CreateMode)
	}

	// Test round-trip: rendering should preserve the 100755 mode
	rendered := Render(changes)
	if !strings.Contains(string(rendered), "new file mode 100755") {
		t.Errorf("expected 'new file mode 100755' in rendered output, got:\n%s", string(rendered))
	}
}

// TestCreateFileModeRoundTrip tests that create file mode round-trips correctly.
func TestCreateFileModeRoundTrip(t *testing.T) {
	original := `diff --git a/script.sh b/script.sh
new file mode 100755
--- /dev/null
+++ b/script.sh
@@ -0,0 +1,1 @@
+#!/bin/bash
`

	changes, err := Parse([]byte(original))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	rendered := Render(changes)

	if original != string(rendered) {
		t.Errorf("round-trip mismatch\noriginal:\n%s\nrendered:\n%s",
			original, string(rendered))
	}
}

// TestHeaderlessMultiFileTwoFiles tests that a header-less two-file diff parses correctly.
func TestHeaderlessMultiFileTwoFiles(t *testing.T) {
	diff := `--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old1
+new1
--- a/file2.txt
+++ b/file2.txt
@@ -1,1 +1,1 @@
-old2
+new2
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	if changes[0].Path != "file1.txt" {
		t.Errorf("expected first file path 'file1.txt', got %q", changes[0].Path)
	}
	if changes[1].Path != "file2.txt" {
		t.Errorf("expected second file path 'file2.txt', got %q", changes[1].Path)
	}

	if len(changes[0].Hunks) != 1 {
		t.Errorf("expected 1 hunk for file1, got %d", len(changes[0].Hunks))
	}
	if len(changes[1].Hunks) != 1 {
		t.Errorf("expected 1 hunk for file2, got %d", len(changes[1].Hunks))
	}
}

// TestHeaderlessMultiFileThreeFiles tests that a header-less three-file diff parses correctly.
func TestHeaderlessMultiFileThreeFiles(t *testing.T) {
	diff := `--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old1
+new1
--- a/file2.txt
+++ b/file2.txt
@@ -1,1 +1,1 @@
-old2
+new2
--- a/file3.txt
+++ b/file3.txt
@@ -1,1 +1,1 @@
-old3
+new3
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	expectedPaths := []string{"file1.txt", "file2.txt", "file3.txt"}
	for i, expectedPath := range expectedPaths {
		if changes[i].Path != expectedPath {
			t.Errorf("file %d: expected path %q, got %q", i, expectedPath, changes[i].Path)
		}
		if len(changes[i].Hunks) != 1 {
			t.Errorf("file %d: expected 1 hunk, got %d", i, len(changes[i].Hunks))
		}
	}
}

// TestHeaderlessSilentMisSegmentation tests the silent variant where counts would absorb
// the second file's --- and +++ lines if not for the early termination fix.
// The first hunk declares 2 old and 2 new, which would normally consume 2 lines.
// If mis-segmentation occurs, the --- and +++ of file2 (2 lines) would be absorbed,
// and a second @@ would be needed to detect the error. With the fix, the hunk
// stops when counts are satisfied, leaving file2's section for a separate FileChange.
func TestHeaderlessSilentMisSegmentation(t *testing.T) {
	diff := `--- a/file1.txt
+++ b/file1.txt
@@ -1,2 +1,2 @@
-line1
-line2
+modified1
+modified2
--- a/file2.txt
+++ b/file2.txt
@@ -1,2 +1,2 @@
-orig1
-orig2
+changed1
+changed2
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d (silent mis-segmentation would give 1)", len(changes))
	}

	if changes[0].Path != "file1.txt" {
		t.Errorf("expected first file 'file1.txt', got %q", changes[0].Path)
	}
	if changes[1].Path != "file2.txt" {
		t.Errorf("expected second file 'file2.txt', got %q", changes[1].Path)
	}
}

// TestHeaderlessMultiFileRoundTrip tests that a header-less multi-file diff round-trips correctly.
func TestHeaderlessMultiFileRoundTrip(t *testing.T) {
	original := `--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old1
+new1
--- a/file2.txt
+++ b/file2.txt
@@ -1,1 +1,1 @@
-old2
+new2
`

	changes, err := Parse([]byte(original))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	rendered := Render(changes)

	if original != string(rendered) {
		t.Errorf("round-trip mismatch\noriginal:\n%s\nrendered:\n%s",
			original, string(rendered))
	}
}

// TestCreateFileInvalidMode tests that invalid create file modes are rejected.
func TestCreateFileInvalidMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"symlink", "120000"},
		{"garbage", "100700"},
		{"octal", "0644"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diff := fmt.Sprintf(`diff --git a/newfile.txt b/newfile.txt
new file mode %s
--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,1 @@
+content
`, tc.mode)

			_, err := Parse([]byte(diff))
			if err == nil {
				t.Errorf("expected error for create mode %q", tc.mode)
				return
			}

			pErr, ok := err.(*ParseError)
			if !ok {
				t.Errorf("expected ParseError, got %T", err)
				return
			}

			if pErr.Kind != ErrBadMode {
				t.Errorf("expected ErrBadMode, got %v", pErr.Kind)
			}
		})
	}
}

// TestDeleteFileInvalidMode tests that invalid deleted file modes are rejected.
func TestDeleteFileInvalidMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"symlink", "120000"},
		{"garbage", "100700"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diff := fmt.Sprintf(`diff --git a/oldfile.txt b/oldfile.txt
deleted file mode %s
--- a/oldfile.txt
+++ /dev/null
@@ -1,1 +0,0 @@
-content
`, tc.mode)

			_, err := Parse([]byte(diff))
			if err == nil {
				t.Errorf("expected error for deleted file mode %q", tc.mode)
				return
			}

			pErr, ok := err.(*ParseError)
			if !ok {
				t.Errorf("expected ParseError, got %T", err)
				return
			}

			if pErr.Kind != ErrBadMode {
				t.Errorf("expected ErrBadMode, got %v", pErr.Kind)
			}
		})
	}
}

// TestNoEolMarkerAtBoundaryWithSecondFile (W1) tests the intersection of two seams:
// a \ No newline marker at the exact point a hunk's declared counts are satisfied,
// immediately followed by another file section in a header-less diff.
// This tests that the marker handler runs before the completion check, and that
// the completion check doesn't absorb the second file's --- line.
func TestNoEolMarkerAtBoundaryWithSecondFile(t *testing.T) {
	diff := `--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old
+new
\ No newline at end of file
--- a/file2.txt
+++ b/file2.txt
@@ -1,1 +1,1 @@
-file2old
+file2new
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// File 1 should have the marker on its last (and only) hunk
	if changes[0].Path != "file1.txt" {
		t.Errorf("expected first file 'file1.txt', got %q", changes[0].Path)
	}
	if len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 hunk for file1, got %d", len(changes[0].Hunks))
	}

	hunk := changes[0].Hunks[0]
	if len(hunk.Lines) != 3 {
		t.Fatalf("expected 3 lines in file1 hunk (removal + addition + marker), got %d", len(hunk.Lines))
	}

	// Last line should be the backslash marker with correct prefix and content
	if hunk.Lines[2].Prefix != '\\' {
		t.Errorf("expected last line prefix '\\', got %c", hunk.Lines[2].Prefix)
	}
	if hunk.Lines[2].Content != `No newline at end of file` {
		t.Errorf("expected marker content %q, got %q", `No newline at end of file`, hunk.Lines[2].Content)
	}

	// File 2 should be intact
	if changes[1].Path != "file2.txt" {
		t.Errorf("expected second file 'file2.txt', got %q", changes[1].Path)
	}
	if len(changes[1].Hunks) != 1 {
		t.Fatalf("expected 1 hunk for file2, got %d", len(changes[1].Hunks))
	}

	// Round-trip test
	rendered := Render(changes)
	if string(rendered) != diff {
		t.Errorf("round-trip mismatch\noriginal:\n%s\nrendered:\n%s", diff, string(rendered))
	}
}

// TestHeaderlessMultiFileMultiHunk (W3) tests that a header-less diff with multiple hunks
// on the first file is correctly segmented from a second file section.
// The reviewer's trace shows that hunkLoop's @@ stop condition closes each hunk
// before the line-194 guard, so the guard only fires once file 1's hunks are exhausted.
func TestHeaderlessMultiFileMultiHunk(t *testing.T) {
	diff := `--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,1 @@
-old1
+new1
@@ -3,1 +3,1 @@
-old3
+new3
--- a/file2.txt
+++ b/file2.txt
@@ -1,1 +1,1 @@
-file2old
+file2new
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// File 1 should have 2 hunks
	if changes[0].Path != "file1.txt" {
		t.Errorf("expected first file 'file1.txt', got %q", changes[0].Path)
	}
	if len(changes[0].Hunks) != 2 {
		t.Fatalf("expected 2 hunks for file1, got %d", len(changes[0].Hunks))
	}

	// Check first hunk
	if changes[0].Hunks[0].OldStart != 1 || changes[0].Hunks[0].OldLines != 1 {
		t.Errorf("hunk 0: expected old 1,1, got %d,%d", changes[0].Hunks[0].OldStart, changes[0].Hunks[0].OldLines)
	}
	if changes[0].Hunks[0].NewStart != 1 || changes[0].Hunks[0].NewLines != 1 {
		t.Errorf("hunk 0: expected new 1,1, got %d,%d", changes[0].Hunks[0].NewStart, changes[0].Hunks[0].NewLines)
	}

	// Check second hunk
	if changes[0].Hunks[1].OldStart != 3 || changes[0].Hunks[1].OldLines != 1 {
		t.Errorf("hunk 1: expected old 3,1, got %d,%d", changes[0].Hunks[1].OldStart, changes[0].Hunks[1].OldLines)
	}
	if changes[0].Hunks[1].NewStart != 3 || changes[0].Hunks[1].NewLines != 1 {
		t.Errorf("hunk 1: expected new 3,1, got %d,%d", changes[0].Hunks[1].NewStart, changes[0].Hunks[1].NewLines)
	}

	// File 2 should be intact
	if changes[1].Path != "file2.txt" {
		t.Errorf("expected second file 'file2.txt', got %q", changes[1].Path)
	}
	if len(changes[1].Hunks) != 1 {
		t.Fatalf("expected 1 hunk for file2, got %d", len(changes[1].Hunks))
	}
}

// ── Hunk recounting ─────────────────────────────────────────────────────────
//
// The hunk body decides the hunk's extent and counts, not the @@ header, as
// git apply --recount does. Most tests below give header counts that the body
// does not satisfy. The rest pin a body shape the parser must not misread. The
// zero-body test and the negative controls check outcomes that the header
// counts do not decide.

// recountOriginal is calc/divide.go as a 9-line file. Line 2 is empty.
const recountOriginal = "package calc\n\n// Divide returns the quotient of a and b.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil\n}\n"

// applyDiffToText writes original to a temp file, parses diff, applies it to
// that file, and returns the file's resulting content. Parse and apply errors
// are returned unchanged so a test can assert their kind.
func applyDiffToText(t *testing.T, original, diff string) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	changes, err := Parse([]byte(diff))
	if err != nil {
		return "", err
	}
	for i := range changes {
		changes[i].Path = path
	}
	if err := ApplyToAbsPath(changes); err != nil {
		return "", err
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}
	return string(got), nil
}

// TestRecountDeclaredOldCountTooLarge: hunk 1 declares 4 old lines but its
// body holds 1. Hunk 2 sits after hunk 1, so it only applies if the parser
// keeps the counted (not declared) old count for hunk 1's offset.
func TestRecountDeclaredOldCountTooLarge(t *testing.T) {
	diff := "diff --git a/calc/divide.go b/calc/divide.go\n" +
		"--- a/calc/divide.go\n" +
		"+++ b/calc/divide.go\n" +
		"@@ -3,4 +3,2 @@\n" +
		"-// Divide returns the quotient of a and b.\n" +
		"+// Divide returns the integer quotient of a and b.\n" +
		"+// Integer division only.\n" +
		"@@ -8,1 +9,1 @@\n" +
		"-\treturn a / b, nil\n" +
		"+\treturn a / b, nil // quotient\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got := changes[0].Hunks[0]; got.OldLines != 1 || got.NewLines != 2 {
		t.Errorf("hunk 0: expected counted old 1, new 2, got old %d, new %d", got.OldLines, got.NewLines)
	}

	got, err := applyDiffToText(t, recountOriginal, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package calc\n\n// Divide returns the integer quotient of a and b.\n// Integer division only.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil // quotient\n}\n"
	if got != want {
		t.Errorf("declared-old-too-large content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountDeclaredCountTooSmall: the header declares 1 old and 1 new line,
// but the body runs to 3 old and 4 new. Every body line must be applied.
func TestRecountDeclaredCountTooSmall(t *testing.T) {
	diff := "diff --git a/calc/divide.go b/calc/divide.go\n" +
		"--- a/calc/divide.go\n" +
		"+++ b/calc/divide.go\n" +
		"@@ -7,1 +7,1 @@\n" +
		" \t}\n" +
		"-\treturn a / b, nil\n" +
		"+\treturn a / b, nil // quotient\n" +
		"+\t// truncates toward zero\n" +
		" }\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got := changes[0].Hunks[0]; got.OldLines != 3 || got.NewLines != 4 {
		t.Errorf("hunk 0: expected counted old 3, new 4, got old %d, new %d", got.OldLines, got.NewLines)
	}
	if n := len(changes[0].Hunks[0].Lines); n != 5 {
		t.Errorf("hunk 0: expected 5 body lines, got %d", n)
	}

	got, err := applyDiffToText(t, recountOriginal, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package calc\n\n// Divide returns the quotient of a and b.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil // quotient\n\t// truncates toward zero\n}\n"
	if got != want {
		t.Errorf("declared-count-too-small content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountWrongDeclaredNewCount: the header declares 3 new lines, but the
// body holds 1. The old count and body are correct and must apply.
func TestRecountWrongDeclaredNewCount(t *testing.T) {
	diff := "diff --git a/calc/divide.go b/calc/divide.go\n" +
		"--- a/calc/divide.go\n" +
		"+++ b/calc/divide.go\n" +
		"@@ -8,1 +8,3 @@\n" +
		"-\treturn a / b, nil\n" +
		"+\treturn a / b, nil // quotient\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got := changes[0].Hunks[0]; got.OldLines != 1 || got.NewLines != 1 {
		t.Errorf("hunk 0: expected counted old 1, new 1, got old %d, new %d", got.OldLines, got.NewLines)
	}

	got, err := applyDiffToText(t, recountOriginal, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package calc\n\n// Divide returns the quotient of a and b.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil // quotient\n}\n"
	if got != want {
		t.Errorf("wrong-declared-new-count content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountBlankContextLineMidHunk: the empty line 2 of the 9-line file is
// sent as a bare empty line, with no leading space. It sits mid-hunk, so it is
// a blank context line and must be kept.
func TestRecountBlankContextLineMidHunk(t *testing.T) {
	diff := "diff --git a/calc/divide.go b/calc/divide.go\n" +
		"--- a/calc/divide.go\n" +
		"+++ b/calc/divide.go\n" +
		"@@ -1,9 +1,9 @@\n" +
		" package calc\n" +
		"\n" +
		" // Divide returns the quotient of a and b.\n" +
		" func Divide(a, b int) (int, error) {\n" +
		" \tif b == 0 {\n" +
		" \t\treturn 0, errors.New(\"divide by zero\")\n" +
		" \t}\n" +
		"-\treturn a / b, nil\n" +
		"+\treturn a / b, nil // quotient\n" +
		" }\n"

	got, err := applyDiffToText(t, recountOriginal, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package calc\n\n// Divide returns the quotient of a and b.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil // quotient\n}\n"
	if got != want {
		t.Errorf("mid-hunk blank context content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountTrailingEmptyLineAtEOF: the body ends with an empty line at end
// of input. That line is trailing and must add nothing. The declared new count
// (4) is wrong. The recounted body (3 new lines) applies.
func TestRecountTrailingEmptyLineAtEOF(t *testing.T) {
	diff := "diff --git a/calc/divide.go b/calc/divide.go\n" +
		"--- a/calc/divide.go\n" +
		"+++ b/calc/divide.go\n" +
		"@@ -7,3 +7,4 @@\n" +
		" \t}\n" +
		"-\treturn a / b, nil\n" +
		"+\treturn a / b, nil // quotient\n" +
		" }\n" +
		"\n"

	got, err := applyDiffToText(t, recountOriginal, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package calc\n\n// Divide returns the quotient of a and b.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil // quotient\n}\n"
	if got != want {
		t.Errorf("trailing-empty-at-EOF content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountEmptyLineBeforeNextHunk: hunk 1 is followed by an empty line and
// then hunk 2's @@. The empty line is trailing to hunk 1 and must add nothing.
// Hunk 2's declared old count (2) is wrong. The recounted body (1 old line)
// applies. Kept as context, the empty line would have to match the
// "func Divide" line that follows hunk 1 in the file, and the diff would
// conflict.
func TestRecountEmptyLineBeforeNextHunk(t *testing.T) {
	diff := "diff --git a/calc/divide.go b/calc/divide.go\n" +
		"--- a/calc/divide.go\n" +
		"+++ b/calc/divide.go\n" +
		"@@ -3,1 +3,2 @@\n" +
		"-// Divide returns the quotient of a and b.\n" +
		"+// Divide returns the integer quotient of a and b.\n" +
		"+// Integer division only.\n" +
		"\n" +
		"@@ -8,2 +9,1 @@\n" +
		"-\treturn a / b, nil\n" +
		"+\treturn a / b, nil // quotient\n"

	got, err := applyDiffToText(t, recountOriginal, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package calc\n\n// Divide returns the integer quotient of a and b.\n// Integer division only.\nfunc Divide(a, b int) (int, error) {\n\tif b == 0 {\n\t\treturn 0, errors.New(\"divide by zero\")\n\t}\n\treturn a / b, nil // quotient\n}\n"
	if got != want {
		t.Errorf("empty-before-next-hunk content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountRemovalOfDoubleDashLine: removing the SQL line "-- comment"
// gives a body line "--- comment". It is not followed by a "+++ " line, so it
// stays a removal. The declared old count (4) is wrong. The recounted body
// (3 old lines) applies.
func TestRecountRemovalOfDoubleDashLine(t *testing.T) {
	diff := "--- a/q.sql\n" +
		"+++ b/q.sql\n" +
		"@@ -1,4 +1,2 @@\n" +
		" SELECT 1;\n" +
		"--- comment\n" +
		" SELECT 2;\n"

	got, err := applyDiffToText(t, "SELECT 1;\n-- comment\nSELECT 2;\n", diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "SELECT 1;\nSELECT 2;\n"
	if got != want {
		t.Errorf("double-dash removal content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountZeroBodyHunkRejected: a hunk header with no body lines cannot be
// recounted into anything, so it is rejected as ErrHunkCountMismatch.
func TestRecountZeroBodyHunkRejected(t *testing.T) {
	diff := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1,2 +1,2 @@\n"

	_, err := Parse([]byte(diff))
	if err == nil {
		t.Fatal("expected error for zero-body hunk, got none")
	}
	pErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}
	if pErr.Kind != ErrHunkCountMismatch {
		t.Errorf("expected ErrHunkCountMismatch, got %v", pErr.Kind)
	}
	want := "in a/file.txt @@ -1,2 +1,2 @@: hunk has no body lines"
	if pErr.Message != want {
		t.Errorf("zero-body message mismatch:\nwant %q\n got %q", want, pErr.Message)
	}
}

// TestRecountHeaderlessSecondFile: two header-less files with no "diff --git"
// lines. The first hunk declares -1,3 +1,3 and its body counts 3 old and 3 new
// lines, so the declared counts are met. A "--- " and "+++ " pair followed by
// "@@" then ends the hunk, and the second file parses as its own change. Were
// the counts short, the pair would be body.
func TestRecountHeaderlessSecondFile(t *testing.T) {
	diff := "--- a/one.txt\n" +
		"+++ b/one.txt\n" +
		"@@ -1,3 +1,3 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		" three\n" +
		"--- a/two.txt\n" +
		"+++ b/two.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		"-alpha\n" +
		"+ALPHA\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Path != "one.txt" || len(changes[0].Hunks) != 1 {
		t.Fatalf("file 1: expected one.txt with 1 hunk, got %q with %d", changes[0].Path, len(changes[0].Hunks))
	}
	if n := len(changes[0].Hunks[0].Lines); n != 4 {
		t.Errorf("file 1 hunk: expected 4 body lines, got %d", n)
	}
	if changes[1].Path != "two.txt" || len(changes[1].Hunks) != 1 {
		t.Fatalf("file 2: expected two.txt with 1 hunk, got %q with %d", changes[1].Path, len(changes[1].Hunks))
	}
	if got := changes[1].Hunks[0].Lines; len(got) != 2 || got[0].Prefix != '-' || got[0].Content != "alpha" || got[1].Prefix != '+' || got[1].Content != "ALPHA" {
		t.Errorf("file 2 hunk: expected -alpha +ALPHA, got %+v", got)
	}
}

// TestRecountNegativeContextMismatch: a context line that does not match the
// file is rejected as patch_conflict. The header counts agree with the body,
// so the rejection comes from the content.
func TestRecountNegativeContextMismatch(t *testing.T) {
	diff := "--- a/abc.txt\n" +
		"+++ b/abc.txt\n" +
		"@@ -1,3 +1,3 @@\n" +
		" a\n" +
		"-X\n" +
		"+Y\n" +
		" c\n"

	_, err := applyDiffToText(t, "a\nb\nc\n", diff)
	aErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T: %v", err, err)
	}
	if aErr.Kind != ErrPatchConflict {
		t.Errorf("expected patch_conflict, got %v", aErr.Kind)
	}
}

// TestRecountNegativeHunkPastEOF: a hunk that starts beyond the end of the
// file is rejected as patch_conflict. The header counts agree with the body,
// so the rejection comes from the position.
func TestRecountNegativeHunkPastEOF(t *testing.T) {
	diff := "--- a/abc.txt\n" +
		"+++ b/abc.txt\n" +
		"@@ -5,1 +5,1 @@\n" +
		"-x\n" +
		"+y\n"

	_, err := applyDiffToText(t, "a\nb\nc\n", diff)
	aErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T: %v", err, err)
	}
	if aErr.Kind != ErrPatchConflict {
		t.Errorf("expected patch_conflict, got %v", aErr.Kind)
	}
}

// TestRecountDashAndPlusPairMidHunk: a removed line "-- x" and an added line
// "++ y" read as a "--- " / "+++ " pair. Only a "--- " and "+++ " pair followed
// by an "@@" line is a file header, so here the pair is followed by body and
// stays in the hunk. The declared counts are correct, so the body applies in full.
func TestRecountDashAndPlusPairMidHunk(t *testing.T) {
	diff := "--- a/q.sql\n" +
		"+++ b/q.sql\n" +
		"@@ -1,4 +1,4 @@\n" +
		" SELECT 1;\n" +
		"--- x\n" +
		"+++ y\n" +
		" SELECT 2;\n" +
		" SELECT 3;\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if n := len(changes[0].Hunks[0].Lines); n != 5 {
		t.Errorf("hunk 0: expected 5 body lines, got %d", n)
	}

	got, err := applyDiffToText(t, "SELECT 1;\n-- x\nSELECT 2;\nSELECT 3;\n", diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "SELECT 1;\n++ y\nSELECT 2;\nSELECT 3;\n"
	if got != want {
		t.Errorf("dash-and-plus pair content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountBlankLineBeforeNoNewlineMarker: an empty line followed by a
// "\ No newline at end of file" line is a blank context line, and the marker
// stays in the hunk after it. The blank line is neither a removal nor an
// addition, so both counts are 3.
func TestRecountBlankLineBeforeNoNewlineMarker(t *testing.T) {
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,3 +1,3 @@\n" +
		" a\n" +
		"-b\n" +
		"+B\n" +
		"\n" +
		"\\ No newline at end of file\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	h := changes[0].Hunks[0]
	if h.OldLines != 3 || h.NewLines != 3 {
		t.Errorf("hunk 0: expected counted old 3, new 3, got old %d, new %d", h.OldLines, h.NewLines)
	}
	want := []HunkLine{
		{Prefix: ' ', Content: "a"},
		{Prefix: '-', Content: "b"},
		{Prefix: '+', Content: "B"},
		{Prefix: ' ', Content: ""},
		{Prefix: '\\', Content: "No newline at end of file"},
	}
	if len(h.Lines) != len(want) {
		t.Fatalf("hunk 0: expected %d body lines, got %d: %+v", len(want), len(h.Lines), h.Lines)
	}
	for i := range want {
		if h.Lines[i] != want[i] {
			t.Errorf("hunk 0 line %d: want %+v, got %+v", i, want[i], h.Lines[i])
		}
	}
}

// TestRecountBlankLineBeforeHeaderlessSecondFile: an empty line sits between
// the first file's last body line and the second file's "--- " header. There
// is no "diff --git" line. The declared counts (2 old, 2 new) are met before
// the empty line, so the header-like pair ends the hunk. The empty line is not
// a body line, so the first hunk keeps its two old lines and the second file
// parses as its own change.
func TestRecountBlankLineBeforeHeaderlessSecondFile(t *testing.T) {
	diff := "--- a/one.txt\n" +
		"+++ b/one.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		"\n" +
		"--- a/two.txt\n" +
		"+++ b/two.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		"-alpha\n" +
		"+ALPHA\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Path != "one.txt" || len(changes[0].Hunks) != 1 {
		t.Fatalf("file 1: expected one.txt with 1 hunk, got %q with %d", changes[0].Path, len(changes[0].Hunks))
	}
	if h := changes[0].Hunks[0]; h.OldLines != 2 || h.NewLines != 2 || len(h.Lines) != 3 {
		t.Errorf("file 1 hunk: expected old 2, new 2, 3 body lines, got old %d, new %d, %d lines", h.OldLines, h.NewLines, len(h.Lines))
	}
	if changes[1].Path != "two.txt" || len(changes[1].Hunks) != 1 {
		t.Fatalf("file 2: expected two.txt with 1 hunk, got %q with %d", changes[1].Path, len(changes[1].Hunks))
	}
	if got := changes[1].Hunks[0].Lines; len(got) != 2 || got[0].Prefix != '-' || got[0].Content != "alpha" || got[1].Prefix != '+' || got[1].Content != "ALPHA" {
		t.Errorf("file 2 hunk: expected -alpha +ALPHA, got %+v", got)
	}
}

// TestRecountBlankLineBeforeDiffGitHeader: an empty line sits between a hunk
// and a "diff --git" line. The empty line is not a body line, so the first
// hunk keeps its two old lines and the second file parses as its own change.
func TestRecountBlankLineBeforeDiffGitHeader(t *testing.T) {
	diff := "diff --git a/one.txt b/one.txt\n" +
		"--- a/one.txt\n" +
		"+++ b/one.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		"\n" +
		"diff --git a/two.txt b/two.txt\n" +
		"--- a/two.txt\n" +
		"+++ b/two.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		"-alpha\n" +
		"+ALPHA\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Path != "one.txt" || len(changes[0].Hunks) != 1 {
		t.Fatalf("file 1: expected one.txt with 1 hunk, got %q with %d", changes[0].Path, len(changes[0].Hunks))
	}
	if h := changes[0].Hunks[0]; h.OldLines != 2 || h.NewLines != 2 || len(h.Lines) != 3 {
		t.Errorf("file 1 hunk: expected old 2, new 2, 3 body lines, got old %d, new %d, %d lines", h.OldLines, h.NewLines, len(h.Lines))
	}
	if changes[1].Path != "two.txt" || len(changes[1].Hunks) != 1 {
		t.Fatalf("file 2: expected two.txt with 1 hunk, got %q with %d", changes[1].Path, len(changes[1].Hunks))
	}
	if got := changes[1].Hunks[0].Lines; len(got) != 2 || got[0].Prefix != '-' || got[0].Content != "alpha" || got[1].Prefix != '+' || got[1].Content != "ALPHA" {
		t.Errorf("file 2 hunk: expected -alpha +ALPHA, got %+v", got)
	}
}

// TestRecountBlankLineBeforeUnknownLine: an empty line followed by a line
// with an unknown prefix ends the hunk. The empty line is not kept, and the
// unknown line is skipped as it is outside any hunk.
func TestRecountBlankLineBeforeUnknownLine(t *testing.T) {
	diff := "--- a/one.txt\n" +
		"+++ b/one.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		"\n" +
		"Trailing note\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if h := changes[0].Hunks[0]; h.OldLines != 2 || h.NewLines != 2 || len(h.Lines) != 3 {
		t.Errorf("hunk 0: expected old 2, new 2, 3 body lines, got old %d, new %d, %d lines", h.OldLines, h.NewLines, len(h.Lines))
	}

	got, err := applyDiffToText(t, "one\ntwo\n", diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nTWO\n"
	if got != want {
		t.Errorf("blank-before-unknown-line content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountCRLFBareCRBlankContext: in a CRLF file, the empty line 2 is sent
// as a bare "\r" line with no leading space. It is a blank context line mid-hunk.
// The test pins the parsed Content of that line to "\r". The applied output is
// also checked, byte for byte, against the expected CRLF text.
func TestRecountCRLFBareCRBlankContext(t *testing.T) {
	original := "package\r\n\r\nx\r\ny\r\n"
	diff := "--- a/c.txt\n" +
		"+++ b/c.txt\n" +
		"@@ -1,4 +1,4 @@\n" +
		" package\r\n" +
		"\r\n" +
		"-x\r\n" +
		"+X\r\n" +
		" y\r\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	lines := changes[0].Hunks[0].Lines
	if len(lines) != 5 {
		t.Fatalf("hunk 0: expected 5 body lines, got %d: %+v", len(lines), lines)
	}
	if want := (HunkLine{Prefix: ' ', Content: "\r"}); lines[1] != want {
		t.Errorf("bare CR blank line: want %+v, got %+v", want, lines[1])
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "package\r\n\r\nX\r\ny\r\n"
	if got != want {
		t.Errorf("bare CR blank context content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountHeaderedPairIsBody: inside a "diff --git " change, a "--- " and
// "+++ " pair followed by "@@" is body. The removal "-- x" and the addition
// "++ y" stay in the first hunk, and the "@@" starts the second. Both hunks
// apply and no line is lost. Entry point: Parse, then apply via applyDiffToText.
func TestRecountHeaderedPairIsBody(t *testing.T) {
	original := "one\n-- x\ntwo\nthree\n"
	diff := "diff --git a/f.txt b/f.txt\n" +
		"--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"--- x\n" +
		"+++ y\n" +
		"@@ -3,2 +3,2 @@\n" +
		" two\n" +
		"-three\n" +
		"+THREE\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 2 {
		t.Fatalf("expected 1 change with 2 hunks, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 3 || got[1] != (HunkLine{Prefix: '-', Content: "-- x"}) || got[2] != (HunkLine{Prefix: '+', Content: "++ y"}) {
		t.Errorf("hunk 1: expected ' one', '-- x', '++ y', got %+v", got)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\n++ y\ntwo\nTHREE\n"
	if got != want {
		t.Errorf("headered pair content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountHeaderlessPairBelowDeclaredCounts: in a header-less diff, a
// "--- " and "+++ " pair followed by "@@" is body while the counted lines are
// short of the declared counts (1 of 2 old, 1 of 2 new). It is only a header
// once they reach them. Entry point: Parse, then apply via applyDiffToText.
func TestRecountHeaderlessPairBelowDeclaredCounts(t *testing.T) {
	original := "one\n-- x\ntwo\nthree\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"--- x\n" +
		"+++ y\n" +
		"@@ -3,2 +3,2 @@\n" +
		" two\n" +
		"-three\n" +
		"+THREE\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 2 {
		t.Fatalf("expected 1 change with 2 hunks, got %d changes", len(changes))
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\n++ y\ntwo\nTHREE\n"
	if got != want {
		t.Errorf("header-less pair content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountHeaderlessBlankBeforeBodyPair: a blank context line sits before a
// header-like pair in a header-less diff, while the counts are still short of
// the declared ones. The blank line is counted, so it stays in the hunk, and
// the pair is body. Entry point: Parse, then apply via applyDiffToText.
func TestRecountHeaderlessBlankBeforeBodyPair(t *testing.T) {
	original := "one\n\n-- x\nfour\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,3 +1,3 @@\n" +
		" one\n" +
		"\n" +
		"--- x\n" +
		"+++ y\n" +
		"@@ -4,1 +4,1 @@\n" +
		"-four\n" +
		"+FOUR\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 2 {
		t.Fatalf("expected 1 change with 2 hunks, got %d changes", len(changes))
	}
	if n := len(changes[0].Hunks[0].Lines); n != 4 {
		t.Errorf("hunk 1: expected 4 body lines, got %d", n)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\n\n++ y\nFOUR\n"
	if got != want {
		t.Errorf("blank-before-pair content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountFormatPatchSignature: a git format-patch output ends with a "-- "
// line, a version line and a blank line. The declared counts (2 old, 2 new) are
// met when the "-- " line is reached. The signature is not a removal, and the
// hunk keeps its three body lines. Entry point: Parse, then apply via
// applyDiffToText.
func TestRecountFormatPatchSignature(t *testing.T) {
	original := "one\ntwo\n"
	diff := "diff --git a/f.txt b/f.txt\n" +
		"--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		"-- \n" +
		"2.39.1\n" +
		"\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if n := len(changes[0].Hunks[0].Lines); n != 3 {
		t.Errorf("hunk 1: expected 3 body lines, got %d: %+v", n, changes[0].Hunks[0].Lines)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nTWO\n"
	if got != want {
		t.Errorf("format-patch content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountFormatPatchSignatureAppleTrailer: a macOS git writes its version
// line with a trailer after the number, "2.39.5 (Apple Git-154)". The counts are
// met at the "-- " line, so the signature is dropped and the hunk keeps its
// three body lines. Entry point: Parse, then apply via applyDiffToText.
func TestRecountFormatPatchSignatureAppleTrailer(t *testing.T) {
	original := "one\ntwo\n"
	diff := "diff --git a/f.txt b/f.txt\n" +
		"--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		"-- \n" +
		"2.39.5 (Apple Git-154)\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if n := len(changes[0].Hunks[0].Lines); n != 3 {
		t.Errorf("hunk 1: expected 3 body lines, got %d: %+v", n, changes[0].Hunks[0].Lines)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nTWO\n"
	if got != want {
		t.Errorf("Apple trailer content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountRemovalOfDashSpaceLine: a removal of the file line "- " is the
// body line "-- ". Three more non-empty body lines follow it, so it is not a
// signature and stays a removal. The counts are short as well, as one old and
// one new line are counted of the 4 and 3 declared. Entry point: Parse, then apply via
// applyDiffToText.
func TestRecountRemovalOfDashSpaceLine(t *testing.T) {
	original := "one\n- \ntwo\nthree\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,4 +1,3 @@\n" +
		" one\n" +
		"-- \n" +
		"-two\n" +
		"+TWO\n" +
		" three\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 5 || got[1] != (HunkLine{Prefix: '-', Content: "- "}) {
		t.Errorf("hunk 1: expected removal of \"- \" at line 2, got %+v", got)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nTWO\nthree\n"
	if got != want {
		t.Errorf("dash-space removal content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountDashSpaceBeforeTwoLines: a "-- " body line followed by two
// non-empty lines, "-two" and "+TWO", is not a signature: a signature has
// exactly one non-empty line after it, and that line looks like a git version.
// The "-- " is the removal of "- ". The header under-declares (1 old, 1 new), so
// the counts are already met at the "-- " line and the lines after it alone
// decide. Entry point: Parse, then apply via applyDiffToText.
func TestRecountDashSpaceBeforeTwoLines(t *testing.T) {
	original := "one\n- \ntwo\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		" one\n" +
		"-- \n" +
		"-two\n" +
		"+TWO\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if n := len(changes[0].Hunks[0].Lines); n != 4 {
		t.Errorf("hunk 1: expected 4 body lines, got %d: %+v", n, changes[0].Hunks[0].Lines)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nTWO\n"
	if got != want {
		t.Errorf("dash-space before two lines content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountDashSpaceRemovalBeforeCountsMet: a removal of the file line "- " is
// the body line "-- ". It is followed by one non-empty version-like line at end
// of input, which is the shape of a signature. The counted lines are 1 old and
// 1 new of the 3 and 2 declared, so the counts are short and the line is a
// removal, not a signature. The version line has no body prefix and ends the
// hunk. Entry point: Parse, then apply via applyDiffToText.
func TestRecountDashSpaceRemovalBeforeCountsMet(t *testing.T) {
	original := "one\n- \ntwo\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,3 +1,2 @@\n" +
		" one\n" +
		"-- \n" +
		"2.39.1\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 2 || got[1] != (HunkLine{Prefix: '-', Content: "- "}) {
		t.Fatalf("hunk 1: expected removal of \"- \" at line 2, got %+v", got)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\ntwo\n"
	if got != want {
		t.Errorf("removal before counts met content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountRemovalOfDashSpaceAsLastLine: a removal of the file line "- " is the
// body line "-- " and is the LAST line of the diff. The header under-declares
// (1 old, 1 new), so the counts are already reached at the "-- " line, and no
// line follows it, so it is not a signature. It stays a removal with the exact
// content "- ". Entry point: Parse, then apply via applyDiffToText.
func TestRecountRemovalOfDashSpaceAsLastLine(t *testing.T) {
	original := "one\n- \nthree\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		" one\n" +
		"-- \n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 2 || got[1] != (HunkLine{Prefix: '-', Content: "- "}) {
		t.Fatalf("hunk 1: expected removal of \"- \" as last line, got %+v", got)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nthree\n"
	if got != want {
		t.Errorf("last-line removal content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountDashSpaceBeforeNonVersionLine: a "-- " line followed by one
// non-empty line that does not look like a git version ("cheers") is not a
// signature, even with the counts already reached (1 old, 1 new declared and
// counted). The "-- " is the removal of "- ". The "cheers" line has no body
// prefix, so it ends the hunk. Entry point: Parse, then apply via
// applyDiffToText.
func TestRecountDashSpaceBeforeNonVersionLine(t *testing.T) {
	original := "one\n- \nthree\n"
	diff := "--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		" one\n" +
		"-- \n" +
		"cheers\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 2 || got[1] != (HunkLine{Prefix: '-', Content: "- "}) {
		t.Fatalf("hunk 1: expected removal of \"- \" at line 2, got %+v", got)
	}

	got, err := applyDiffToText(t, original, diff)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	want := "one\nthree\n"
	if got != want {
		t.Errorf("non-version line content mismatch:\nwant %q\n got %q", want, got)
	}
}

// TestRecountSignatureAfterOverDeclaredHunkFailsLoud: a format-patch signature
// after a hunk that over-declares its counts (5 and 5, against a body of 2 old
// and 2 new) is read as the removal of "- ", because the counts are short when
// the "-- " line is reached. That is the cost of gating the signature on the
// counts. The result is loud, not silent: the apply fails as patch_conflict and
// the file is left unchanged. Entry point: Parse, then ApplyToAbsPath.
func TestRecountSignatureAfterOverDeclaredHunkFailsLoud(t *testing.T) {
	original := "one\ntwo\n"
	diff := "diff --git a/f.txt b/f.txt\n" +
		"--- a/f.txt\n" +
		"+++ b/f.txt\n" +
		"@@ -1,5 +1,5 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		"-- \n" +
		"2.39.1\n" +
		"\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 4 || got[3] != (HunkLine{Prefix: '-', Content: "- "}) {
		t.Fatalf("hunk 1: expected the signature read as removal of \"- \", got %+v", got)
	}

	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	changes[0].Path = path
	applyErr, ok := ApplyToAbsPath(changes).(*ApplyError)
	if !ok || applyErr.Kind != ErrPatchConflict {
		t.Fatalf("expected patch_conflict for signature read as removal, got %v", applyErr)
	}
	if got, _ := os.ReadFile(path); string(got) != original {
		t.Errorf("file changed despite failed apply: %q", got)
	}
}

// TestRecountHeaderlessOverDeclaredFirstHunkFailsLoud: in a header-less diff,
// the first hunk declares -1,5 +1,5 but its body holds 3 old and 3 new lines.
// The counts are short at the second file's "--- " and "+++ " pair, so the pair
// is body: it is read as the removal "-- a/two.txt" and the addition
// "++ b/two.txt", and the second file's "@@" starts a second hunk of the first
// file. This is the known limitation named on hunkEndsAt. The failure is loud,
// not silent: Parse returns one change, and the apply fails as patch_conflict
// on the first file, which is left unchanged. Entry point: Parse, then
// ApplyToAbsPath.
func TestRecountHeaderlessOverDeclaredFirstHunkFailsLoud(t *testing.T) {
	original := "one\ntwo\nthree\n"
	diff := "--- a/one.txt\n" +
		"+++ b/one.txt\n" +
		"@@ -1,5 +1,5 @@\n" +
		" one\n" +
		"-two\n" +
		"+TWO\n" +
		" three\n" +
		"--- a/two.txt\n" +
		"+++ b/two.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		"-alpha\n" +
		"+ALPHA\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || changes[0].Path != "one.txt" || len(changes[0].Hunks) != 2 {
		t.Fatalf("expected 1 change one.txt with 2 hunks, got %d changes: %+v", len(changes), changes)
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 6 || got[4] != (HunkLine{Prefix: '-', Content: "-- a/two.txt"}) || got[5] != (HunkLine{Prefix: '+', Content: "++ b/two.txt"}) {
		t.Fatalf("hunk 1: expected the second file's header lines as body, got %+v", got)
	}

	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	changes[0].Path = path
	applyErr, ok := ApplyToAbsPath(changes).(*ApplyError)
	if !ok || applyErr.Kind != ErrPatchConflict {
		t.Fatalf("expected patch_conflict on the first file, got %v", applyErr)
	}
	if got, _ := os.ReadFile(path); string(got) != original {
		t.Errorf("file changed despite failed apply: %q", got)
	}
}

// TestRecountBlankCompletesCountsBeforeHeaderlessPair: the declared counts (2
// old, 2 new) are met only because a blank line is counted. One context line
// is counted before the blank line, and the blank line is the second. The
// look-ahead from the blank line counts it, finds the header-like pair with the
// counts met, and ends the hunk, so the blank line is dropped as trailing. The
// first hunk keeps one body line and the second file parses as its own change.
// Entry point: Parse.
func TestRecountBlankCompletesCountsBeforeHeaderlessPair(t *testing.T) {
	diff := "--- a/one.txt\n" +
		"+++ b/one.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" one\n" +
		"\n" +
		"--- a/two.txt\n" +
		"+++ b/two.txt\n" +
		"@@ -1,1 +1,1 @@\n" +
		"-alpha\n" +
		"+ALPHA\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 1 || got[0] != (HunkLine{Prefix: ' ', Content: "one"}) {
		t.Errorf("file 1 hunk: expected the single line \" one\", got %+v", got)
	}
	if changes[1].Path != "two.txt" || len(changes[1].Hunks) != 1 || len(changes[1].Hunks[0].Lines) != 2 {
		t.Errorf("file 2: expected two.txt with 1 hunk of 2 lines, got %+v", changes[1])
	}
}

// TestRecountCRLFBareCRMixedEndingRejected: the bare "\r" blank context line is
// the only line in the patch that carries a CR, and the added line "+TWO" has
// none. The CR is what makes the patch carry endings, so the bare added line
// mixes endings in a CRLF file and is rejected as line_ending_mismatch. Without
// the CR the patch would not be checked and would apply. Entry point: Parse,
// then apply via applyDiffToText.
func TestRecountCRLFBareCRMixedEndingRejected(t *testing.T) {
	original := "one\r\n\r\ntwo\r\n"
	diff := "--- a/c.txt\n" +
		"+++ b/c.txt\n" +
		"@@ -1,3 +1,3 @@\n" +
		" one\n" +
		"\r\n" +
		"-two\n" +
		"+TWO\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(changes) != 1 || len(changes[0].Hunks) != 1 {
		t.Fatalf("expected 1 change with 1 hunk, got %d changes", len(changes))
	}
	if got := changes[0].Hunks[0].Lines; len(got) != 4 || got[1] != (HunkLine{Prefix: ' ', Content: "\r"}) {
		t.Fatalf("hunk 1: expected bare CR context at line 2, got %+v", got)
	}

	_, err = applyDiffToText(t, original, diff)
	applyErr, ok := err.(*ApplyError)
	if !ok || applyErr.Kind != ErrLineEnding {
		t.Fatalf("expected line_ending_mismatch, got %v", err)
	}
}

// TestHunkEndsAtTable tests hunkEndsAt directly, with no Parse call. Each row
// gives the lines, the index tested (always 0), the state and the expected
// result. The rows are grouped by what they exercise:
//   - the "@@" and "diff --git " ends;
//   - the header-like pair, with the counts met and with the counts short,
//     in a header-less change and in a change with a "diff --git " header;
//   - the format-patch signature, with the counts met and with the counts
//     short of the old or of the new declaration;
//   - "-- " lines with and without a version-like line after them, including
//     trailers after the number;
//   - ordinary body lines.
func TestHunkEndsAtTable(t *testing.T) {
	cases := []struct {
		name  string
		lines string
		st    hunkState
		want  bool
	}{
		{"@@ line ends", "@@ -1,1 +1,1 @@\n", hunkState{}, true},
		{"diff --git line ends", "diff --git a/f b/f\n", hunkState{hasHeader: true}, true},
		{"headerless pair with counts met ends", "--- x\n+++ y\n@@ -1,1 +1,1 @@\n", hunkState{declOld: 1, declNew: 1, oldCount: 1, newCount: 1}, true},
		{"headerless pair short of old count is body", "--- x\n+++ y\n@@ -1,2 +1,2 @@\n", hunkState{declOld: 2, declNew: 2, oldCount: 1, newCount: 2}, false},
		{"headerless pair short of new count is body", "--- x\n+++ y\n@@ -1,2 +1,2 @@\n", hunkState{declOld: 2, declNew: 2, oldCount: 2, newCount: 1}, false},
		{"headered pair with counts met is body", "--- x\n+++ y\n@@ -1,1 +1,1 @@\n", hunkState{hasHeader: true, declOld: 1, declNew: 1, oldCount: 1, newCount: 1}, false},
		{"headered pair short of counts is body", "--- x\n+++ y\n@@ -1,2 +1,2 @@\n", hunkState{hasHeader: true, declOld: 2, declNew: 2, oldCount: 1, newCount: 1}, false},
		{"pair not followed by @@ is body", "--- x\n+++ y\n z\n", hunkState{declOld: 1, declNew: 1, oldCount: 1, newCount: 1}, false},
		{"format-patch signature ends", "-- \n2.39.1\n", hunkState{}, true},
		{"-- with two non-empty lines after is body", "-- \n-two\n+TWO\n", hunkState{}, false},
		{"signature with a blank line after the version ends", "-- \n2.39.1\n\n", hunkState{}, true},
		{"signature with a windows version line ends", "-- \n2.39.1.windows.1\n", hunkState{}, true},
		{"signature with an Apple Git trailer and counts met ends", "-- \n2.39.5 (Apple Git-154)\n", hunkState{declOld: 2, declNew: 2, oldCount: 2, newCount: 2}, true},
		{"-- with only a blank line after is body", "-- \n\n", hunkState{}, false},
		{"-- with two version lines after is body", "-- \n2.39.1\n2.40.0\n", hunkState{}, false},
		{"-- with a non-version line after is body", "-- \ncheers\n", hunkState{}, false},
		{"-- with a v-prefixed version line after is body", "-- \nv2.39.1\n", hunkState{}, false},
		{"-- with a one-part number after is body", "-- \n2\n", hunkState{}, false},
		{"-- with a dangling dot after is body", "-- \n2.\n", hunkState{}, false},
		{"signature with CR on the -- line ends", "-- \r\n2.39.1\r\n", hunkState{}, true},
		{"-- with counts met and no line after is body", "-- \n", hunkState{declOld: 2, declNew: 2, oldCount: 2, newCount: 2}, false},
		{"-- short of old count is body", "-- \n2.39.1\n", hunkState{declOld: 3, declNew: 2, oldCount: 1, newCount: 2}, false},
		{"-- short of new count is body", "-- \n2.39.1\n", hunkState{declOld: 2, declNew: 3, oldCount: 2, newCount: 1}, false},
		{"removal line is body", "-x\n", hunkState{}, false},
		{"addition line is body", "+y\n", hunkState{}, false},
		{"context line is body", " z\n", hunkState{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &parser{lines: strings.Split(c.lines, "\n")}
			if got := p.hunkEndsAt(0, c.st); got != c.want {
				t.Errorf("hunkEndsAt(0) = %v, want %v", got, c.want)
			}
		})
	}
}
