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
		{
			name:    "malformed-hunk-counts.diff",
			wantErr: true,
			errKind: ErrHunkCountMismatch,
			validate: func(t *testing.T, changes []FileChange) {
				// Should not reach here on error
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

// TestHunkArithmeticValidation tests that hunk line counts are validated with anchored messages.
func TestHunkArithmeticValidation(t *testing.T) {
	// This diff declares 2 lines in old and 3 lines in new, but only provides 2 total
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,3 @@
 line1
 line2
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
