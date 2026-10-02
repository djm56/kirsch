package patch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyModifySingleFile tests applying a single-file modification.
func TestApplyModifySingleFile(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("First line\nSecond line\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a simple modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 First line
-Second line
+Updated line
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Update paths to use temp directory
	changes[0].Path = testFile

	// Apply the patch
	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the result
	result, _ := os.ReadFile(testFile)
	expected := "First line\nUpdated line\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyMultiFileAtomicity tests that multi-file patches are atomic.
// Uses real corpus fixtures to verify atomicity when Phase 2 fails on the second file.
func TestApplyMultiFileAtomicity(t *testing.T) {
	tmpDir := t.TempDir()

	// Copy the repo-patch fixtures into the temp directory
	sourceDir := "../../testdata/repo-patch"
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("failed to read testdata/repo-patch: %v", err)
	}

	// Create directory structure and copy files
	if err := os.Mkdir(filepath.Join(tmpDir, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(sourceDir, entry.Name())
		dstPath := filepath.Join(tmpDir, entry.Name())

		// Handle nested directory
		if entry.IsDir() {
			subEntries, err := os.ReadDir(srcPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", srcPath, err)
			}
			for _, subEntry := range subEntries {
				subSrcPath := filepath.Join(srcPath, subEntry.Name())
				subDstPath := filepath.Join(dstPath, subEntry.Name())
				content, err := os.ReadFile(subSrcPath)
				if err != nil {
					t.Fatalf("failed to read %s: %v", subSrcPath, err)
				}
				if err := os.WriteFile(subDstPath, content, 0o644); err != nil {
					t.Fatalf("failed to write %s: %v", subDstPath, err)
				}
			}
		} else {
			content, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("failed to read %s: %v", srcPath, err)
			}
			if err := os.WriteFile(dstPath, content, 0o644); err != nil {
				t.Fatalf("failed to write %s: %v", dstPath, err)
			}
		}
	}

	// Hash all files before applying the patch
	beforeHashes := make(map[string]string)
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(tmpDir, path)
		content, _ := os.ReadFile(path)
		beforeHashes[relPath] = hashBytes(content)
		return nil
	})

	// Load the conflict fixture which will fail on the second file
	conflictDiff, err := os.ReadFile("../../testdata/patches/multi-file-second-conflicts.diff")
	if err != nil {
		t.Fatalf("failed to read conflict fixture: %v", err)
	}

	changes, err := Parse(conflictDiff)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Update paths to use temp directory
	for i := range changes {
		changes[i].Path = filepath.Join(tmpDir, changes[i].Path)
	}

	// Apply - should fail because unicode.txt content doesn't match
	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	// Verify atomicity: all files unchanged (byte-identical)
	afterHashes := make(map[string]string)
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(tmpDir, path)
		content, _ := os.ReadFile(path)
		afterHashes[relPath] = hashBytes(content)
		return nil
	})

	for file, beforeHash := range beforeHashes {
		afterHash, ok := afterHashes[file]
		if !ok {
			t.Errorf("file %s disappeared", file)
		} else if beforeHash != afterHash {
			t.Errorf("file %s was modified despite conflict", file)
		}
	}
}

// hashBytes returns a hash of byte content for byte-identical comparison
func hashBytes(data []byte) string {
	var sum uint64
	for _, b := range data {
		sum = sum*31 + uint64(b)
	}
	return string(rune(sum))
}

// TestApplyCRLFPreservation tests that CRLF line endings are preserved.
func TestApplyCRLFPreservation(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file with CRLF endings
	origContent := "Line1\r\nLine2\r\n"
	if err := os.WriteFile(testFile, []byte(origContent), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a diff with CRLF in hunks
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 Line1
-Line2
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	result, _ := os.ReadFile(testFile)
	expected := "Line1\r\nModified\r\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyGitStyleCRLFDiffPreservesCRLF applies a git-style CRLF diff to a
// CRLF file and verifies the file is correctly modified with CRLF preserved.
func TestApplyGitStyleCRLFDiffPreservesCRLF(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "crlf.txt")

	// Copy the fixture file byte-for-byte
	fixtureContent, err := os.ReadFile("../../testdata/repo-patch/crlf.txt")
	if err != nil {
		t.Fatalf("failed to read fixture crlf.txt: %v", err)
	}
	if err := os.WriteFile(testFile, fixtureContent, 0o644); err != nil {
		t.Fatalf("failed to copy fixture to temp: %v", err)
	}

	// Load the git-style CRLF diff
	diffContent, err := os.ReadFile("../../testdata/patches/crlf-preserving.diff")
	if err != nil {
		t.Fatalf("failed to read crlf-preserving.diff: %v", err)
	}

	changes, err := Parse(diffContent)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	// Apply the patch
	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the result
	result, _ := os.ReadFile(testFile)
	expected := "First line\r\nModified second line\r\nThird line\r\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyCRLFConvertingPatchRejected applies a converting diff (CRLF lines
// on an LF file, or vice versa) and expects ErrLineEnding rejection.
func TestApplyCRLFConvertingPatchRejected(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "crlf.txt")

	// Copy the fixture file byte-for-byte
	fixtureContent, err := os.ReadFile("../../testdata/repo-patch/crlf.txt")
	if err != nil {
		t.Fatalf("failed to read fixture crlf.txt: %v", err)
	}
	if err := os.WriteFile(testFile, fixtureContent, 0o644); err != nil {
		t.Fatalf("failed to copy fixture to temp: %v", err)
	}

	// Load the converting diff (CRLF in patch but will have LF added lines)
	diffContent, err := os.ReadFile("../../testdata/patches/crlf-converting.diff")
	if err != nil {
		t.Fatalf("failed to read crlf-converting.diff: %v", err)
	}

	changes, err := Parse(diffContent)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	// Apply the patch - should fail with ErrLineEnding
	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected ErrLineEnding, but apply succeeded")
	}

	applyErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T", err)
	}

	if applyErr.Kind != ErrLineEnding {
		t.Errorf("expected ErrLineEnding, got %v", applyErr.Kind)
	}

	// Verify file is unchanged
	result, _ := os.ReadFile(testFile)
	if string(result) != string(fixtureContent) {
		t.Errorf("file should be unchanged after rejection")
	}
}

// TestApplyLFFileRejectsCRLFAddedLine applies a patch with CRLF-ending added
// lines to an LF file and expects ErrLineEnding rejection.
func TestApplyLFFileRejectsCRLFAddedLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "lf.txt")

	// Create an LF file
	if err := os.WriteFile(testFile, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Build an inline diff with context lines (no \r) but a + line with \r
	diff := "--- a/lf.txt\n+++ b/lf.txt\n@@ -1,3 +1,3 @@\n a\n-b\n+" +
		"b" + "\r" + "\n c\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	// Apply - should fail with ErrLineEnding
	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected ErrLineEnding, but apply succeeded")
	}

	applyErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T", err)
	}

	if applyErr.Kind != ErrLineEnding {
		t.Errorf("expected ErrLineEnding, got %v", applyErr.Kind)
	}

	// Verify file is unchanged
	result, _ := os.ReadFile(testFile)
	if string(result) != "a\nb\nc\n" {
		t.Errorf("file should be unchanged after rejection, got %q", string(result))
	}
}

// TestApplyGitStyleCRLFEditsLastLineWithoutNewline applies a git-style CRLF diff
// that edits the last line of a CRLF file when that line has no final newline.
// The diff emits the + line without \r and then a "\ No newline at end of file"
// marker, which should not be rejected as a converting patch.
func TestApplyGitStyleCRLFEditsLastLineWithoutNewline(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "crlf-noeol.txt")

	// Create file with CRLF endings, last line has no newline
	if err := os.WriteFile(testFile, []byte("a"+"\r"+"\n"+"b"+"\r"+"\n"+"last"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Diff that edits the last line (no newline). The + line has no \r because
	// it's not followed by a line ending; only context and - lines have \r.
	diff := "--- a/crlf-noeol.txt\n+++ b/crlf-noeol.txt\n@@ -1,3 +1,3 @@\n " +
		"a" + "\r" + "\n " + "b" + "\r" + "\n" +
		"-last\n" +
		"\\ No newline at end of file\n" +
		"+new last\n" +
		"\\ No newline at end of file\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	// Apply should succeed (+ line without \r is exempt because it's before a marker)
	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the result
	result, _ := os.ReadFile(testFile)
	expected := "a" + "\r" + "\n" + "b" + "\r" + "\n" + "new last"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyGitStyleCRLFAppendsAfterLastLineWithoutNewline applies a git-style
// CRLF diff that appends to a CRLF file after a line without a final newline.
// The original last line has no newline; the diff removes it, adds it back with
// a newline (now with \r), then adds another line.
func TestApplyGitStyleCRLFAppendsAfterLastLineWithoutNewline(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "crlf-noeol-append.txt")

	// Create file with CRLF on first line, last line has no newline
	if err := os.WriteFile(testFile, []byte("a"+"\r"+"\n"+"b"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Diff that removes the last line (no newline), then adds it back with a
	// newline (so with \r), then adds another line.
	diff := "--- a/crlf-noeol-append.txt\n+++ b/crlf-noeol-append.txt\n@@ -1,2 +1,3 @@\n " +
		"a" + "\r" + "\n" +
		"-b\n" +
		"\\ No newline at end of file\n" +
		"+b" + "\r" + "\n" +
		"+c\n" +
		"\\ No newline at end of file\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	// Apply should succeed
	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the result
	result, _ := os.ReadFile(testFile)
	expected := "a" + "\r" + "\n" + "b" + "\r" + "\n" + "c"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyGitStyleCRLFRenameWithEdit applies a rename-with-edit whose hunks
// carry \r, applied to a CRLF source. The renamed file keeps CRLF with no \r\r.
func TestApplyGitStyleCRLFRenameWithEdit(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.txt")
	// Write CRLF content to source
	if err := os.WriteFile(oldFile, []byte("Keep this line\r\nOld content\r\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	newFile := filepath.Join(tmpDir, "dst.txt")

	// Build a git-style diff with CRLF in hunks
	diff := "diff --git a/src.txt b/dst.txt\nrename from src.txt\nrename to dst.txt\n" +
		"--- a/src.txt\n+++ b/dst.txt\n@@ -1,2 +1,2 @@\n Keep this line\r\n" +
		"-Old content\r\n+New content\r\n"

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile

	if err := ApplyToAbsPath(changes); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify src is gone
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("src.txt should no longer exist after rename")
	}

	// Verify new file has correct content (CRLF, not \r\r\n)
	result, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("dst.txt should exist after rename: %v", err)
	}
	expected := "Keep this line\r\nNew content\r\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyExecBitPreservation tests that executable bit is preserved.
func TestApplyExecBitPreservation(t *testing.T) {
	tmpDir := t.TempDir()
	scriptFile := filepath.Join(tmpDir, "script.sh")

	// Create executable script
	if err := os.WriteFile(scriptFile, []byte("#!/bin/bash\necho hi\n"), 0o755); err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	// Check if the filesystem supports executable bits
	// (some filesystems like FAT don't support permission modes)
	info, _ := os.Stat(scriptFile)
	if info.Mode()&0o111 == 0 {
		t.Skip("filesystem does not support executable bit")
	}

	// Parse a modification diff
	diff := `--- a/script.sh
+++ b/script.sh
@@ -1,2 +1,2 @@
 #!/bin/bash
-echo hi
+echo hello
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = scriptFile

	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Check executable bit is preserved
	info, _ = os.Stat(scriptFile)
	if info.Mode()&0o111 == 0 {
		t.Error("executable bit was lost")
	}
}

// TestApplyNoEOLRemoval tests removing a trailing newline.
func TestApplyNoEOLRemoval(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file matching the fixture format
	originalContent := "First line\nSecond line\nThird line\n"
	if err := os.WriteFile(testFile, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Use actual diff that removes trailing newline
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 First line
 Second line
-Third line
+Third line
\ No newline at end of file
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	result, _ := os.ReadFile(testFile)
	expected := "First line\nSecond line\nThird line"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplyNoEOLAddition tests adding a trailing newline.
func TestApplyNoEOLAddition(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file without trailing newline
	if err := os.WriteFile(testFile, []byte("Content no newline"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse diff that adds trailing newline
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,1 +1,1 @@
-Content no newline
\ No newline at end of file
+Content no newline
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	result, _ := os.ReadFile(testFile)
	expected := "Content no newline\n"
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

// TestApplySamePatchTwiceFails tests that applying the same patch twice fails.
func TestApplySamePatchTwiceFails(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("Original\nContent\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 Original
-Content
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	// First application should succeed
	err = ApplyToAbsPath(changes)
	if err != nil {
		t.Fatalf("first apply failed: %v", err)
	}

	// Second application should fail (content no longer matches)
	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Error("expected error on second apply, got nil")
	}

	// Verify the error is a patch conflict with content
	applyErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T", err)
	}

	if applyErr.Kind != ErrPatchConflict {
		t.Errorf("expected ErrPatchConflict, got %v", applyErr.Kind)
	}

	if applyErr.Content == "" {
		t.Error("expected Content field to be populated in conflict error")
	}

	// Verify file was not corrupted by the failed second apply
	result, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file after failed apply: %v", err)
	}
	expected := "Original\nModified\n"
	if string(result) != expected {
		t.Errorf("file was corrupted: expected %q, got %q", expected, string(result))
	}
}

// TestApplyConflictReturnsContent tests that conflict errors include actual file content.
func TestApplyConflictReturnsContent(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with different content
	if err := os.WriteFile(testFile, []byte("Unexpected content\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Try to apply a patch that expects different content
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,1 +1,1 @@
-Expected content
+New content
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected conflict error")
	}

	applyErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T", err)
	}

	// Content field should be populated
	if applyErr.Content == "" {
		t.Error("Content field should be populated for conflict errors")
	}

	// Content should mention the actual file content
	if !strings.Contains(applyErr.Content, "Unexpected") {
		t.Errorf("Content field should contain actual file content, got: %s", applyErr.Content)
	}
}

// TestApplyContextMismatch tests that strict context matching is enforced.
func TestApplyContextMismatch(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a file
	if err := os.WriteFile(testFile, []byte("Line1\nContext\nLine3\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Try to apply a patch with wrong context
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 Line1
-WrongContext
+NewLine
 Line3
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	changes[0].Path = testFile

	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected context mismatch error")
	}

	applyErr, ok := err.(*ApplyError)
	if !ok {
		t.Fatalf("expected ApplyError, got %T", err)
	}

	if applyErr.Kind != ErrPatchConflict {
		t.Errorf("expected ErrPatchConflict, got %v", applyErr.Kind)
	}
}

// TestDetectLineEndingTie tests the tie case where crlfCount == lfCount.
// When counts are equal, the function should resolve to LF.
func TestDetectLineEndingTie(t *testing.T) {
	// Content with one CRLF and one LF: equal counts
	content := []byte("Line1\r\nLine2\n")
	le := detectLineEnding(content)
	if le != LineEndingLF {
		t.Errorf("expected LineEndingLF on tie, got %v", le)
	}

	// Verify splitLines handles this correctly (no stray \r)
	lines := splitLines(content)
	expectedLines := []string{"Line1", "Line2"}
	if len(lines) != len(expectedLines) {
		t.Errorf("expected %d lines, got %d", len(expectedLines), len(lines))
	}
	for i, line := range lines {
		if line != expectedLines[i] {
			t.Errorf("line %d: expected %q, got %q", i, expectedLines[i], line)
		}
	}
}

// TestApplyMultiFileAtomicityPhase2Failure tests that multi-file patches are
// atomic when Phase 2 fails, and that the failure actually happens in Phase 2
// — after Phase 1 has already produced a valid temp file for every entry —
// rather than in Phase 1 before rollback() is ever reached.
//
// A locked target directory cannot produce that scenario: applyModify's
// os.CreateTemp runs in that same directory, so a permission lock blocks
// Phase 1 first and rollback() is never called (TestApplyMultiFileAtomicity
// already covers that Phase-1-level case). Forcing a Phase-2-only failure
// needs a seam, so this test overrides the package-level commitRename hook
// for exactly one file's commit, restoring it via t.Cleanup, and confirms via
// sawTempFileForTarget that Phase 1's temp file for that entry already
// existed on disk at the moment the injected failure fired.
//
// The changeset is a create (which commits, then must be rolled back via
// remove) and a rename (which commits, then must be rolled back to restore
// the original path), followed by the entry whose commit is forced to fail.
// Verifies that the "create-or-modify" and "rename" rollback dispositions
// execute correctly by asserting the created file is removed and the renamed
// file is restored to its original path.
func TestApplyMultiFileAtomicityPhase2Failure(t *testing.T) {
	tmpDir := t.TempDir()

	// Existing file to be renamed
	oldFile := filepath.Join(tmpDir, "original.txt")
	if err := os.WriteFile(oldFile, []byte("To be renamed\n"), 0o644); err != nil {
		t.Fatalf("failed to create original file: %v", err)
	}

	// Path the file will be renamed to (does not exist yet)
	newPath := filepath.Join(tmpDir, "renamed.txt")

	// Existing file whose Phase 2 commit will be forced to fail
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("Original target\n"), 0o644); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	// Path where a new file will be created
	createPath := filepath.Join(tmpDir, "created.txt")

	// Build a multi-file patch with create, rename, and a failing modify.
	// Order: create first, rename second (both must commit), then the
	// forced-failure modify. All three sections carry "diff --git" headers:
	// mixing a headered rename section into an otherwise header-less diff
	// silently merges the trailing header-less sections' hunks into the
	// rename's change (verified by parsing this diff both ways) — a
	// pre-existing parser property (patch.go), not something this file
	// changes, but the fixture has to respect it to parse into three
	// separate changes rather than one.
	diff := `diff --git a/created.txt b/created.txt
new file mode 100644
--- /dev/null
+++ b/created.txt
@@ -0,0 +1,1 @@
+New file content
diff --git a/original.txt b/renamed.txt
rename from original.txt
rename to renamed.txt
diff --git a/target.txt b/target.txt
--- a/target.txt
+++ b/target.txt
@@ -1,1 +1,1 @@
-Original target
+Modified target
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Update paths to use temp directory
	for i := range changes {
		switch changes[i].Path {
		case "created.txt":
			changes[i].Path = createPath
		case "renamed.txt":
			changes[i].Path = newPath
			changes[i].OldPath = oldFile
		case "target.txt":
			changes[i].Path = targetFile
		}
	}

	// Inject a Phase-2-only failure for target.txt's commit.
	// NOTE: this test reassigns the package-level commitRename var — do not
	// add t.Parallel() to this test (see commitRename's doc comment).
	origCommitRename := commitRename
	var sawTempFileForTarget bool
	commitRename = func(oldname, newname string) error {
		if newname == targetFile {
			if _, statErr := os.Stat(oldname); statErr == nil {
				sawTempFileForTarget = true
			}
			return fmt.Errorf("injected phase 2 failure for %s", newname)
		}
		return origCommitRename(oldname, newname)
	}
	t.Cleanup(func() { commitRename = origCommitRename })

	// Apply - should fail because target.txt's commit is forced to fail
	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected error from injected Phase 2 failure, got nil")
	}
	if !sawTempFileForTarget {
		t.Fatal("commitRename never saw target.txt's Phase 1 temp file on disk — the injected failure did not exercise Phase 2 as intended")
	}

	// Verify atomicity: all three files in their original state
	// 1. created.txt should not exist (it was created, then rolled back)
	if _, err := os.Stat(createPath); err == nil {
		t.Errorf("created.txt should not exist after rollback, but it does")
	}

	// 2. original.txt should still exist (rename was rolled back, content restored)
	if _, err := os.Stat(oldFile); err != nil {
		t.Errorf("original.txt should exist after rollback, but got error: %v", err)
	}
	originalContent, _ := os.ReadFile(oldFile)
	if string(originalContent) != "To be renamed\n" {
		t.Errorf("original.txt content should be unchanged after rollback, got %q", string(originalContent))
	}

	// 3. renamed.txt should not exist (rename was rolled back)
	if _, err := os.Stat(newPath); err == nil {
		t.Errorf("renamed.txt should not exist after rollback, but it does")
	}

	// 4. target.txt should be unchanged (its own backup was self-repaired)
	targetContent, _ := os.ReadFile(targetFile)
	if string(targetContent) != "Original target\n" {
		t.Errorf("target.txt should be unchanged after rollback, got %q", string(targetContent))
	}
}

// TestApplyRenameWithEditCommitsEditedContent tests that a rename combined
// with content edits in a single operation commits the *edited* content at
// the new path, not the untouched original — and that the temp file built to
// hold the edit is not left behind on the success path.
func TestApplyRenameWithEditCommitsEditedContent(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(oldFile, []byte("Keep this line\nOld content\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	newFile := filepath.Join(tmpDir, "dst.txt")

	diff := `diff --git a/src.txt b/dst.txt
rename from src.txt
rename to dst.txt
--- a/src.txt
+++ b/dst.txt
@@ -1,2 +1,2 @@
 Keep this line
-Old content
+New content
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile

	if err := ApplyToAbsPath(changes); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	if _, err := os.Stat(oldFile); err == nil {
		t.Error("src.txt should no longer exist after rename")
	}

	result, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("dst.txt should exist after rename: %v", err)
	}
	expected := "Keep this line\nNew content\n"
	if string(result) != expected {
		t.Errorf("dst.txt should hold the edited content, expected %q, got %q", expected, string(result))
	}

	leftovers, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	if len(leftovers) != 0 {
		t.Errorf("no temp or backup files should remain after a clean commit, found: %v", leftovers)
	}
}

// TestApplyRenameWithEditRollback tests that a rename-with-edit which has
// already committed is correctly reverted — original bytes restored at the
// old path, edited content removed from the new path — when a later entry's
// Phase 2 commit fails. This is the rollback-side counterpart to
// TestApplyRenameWithEditCommitsEditedContent and exercises the
// oldBackupPath restoration branch in rollback() directly.
func TestApplyRenameWithEditRollback(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(oldFile, []byte("Keep this line\nOld content\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	newFile := filepath.Join(tmpDir, "dst.txt")

	otherFile := filepath.Join(tmpDir, "other.txt")
	if err := os.WriteFile(otherFile, []byte("Original other\n"), 0o644); err != nil {
		t.Fatalf("failed to create other file: %v", err)
	}

	diff := `diff --git a/src.txt b/dst.txt
rename from src.txt
rename to dst.txt
--- a/src.txt
+++ b/dst.txt
@@ -1,2 +1,2 @@
 Keep this line
-Old content
+New content
diff --git a/other.txt b/other.txt
--- a/other.txt
+++ b/other.txt
@@ -1,1 +1,1 @@
-Original other
+Modified other
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for i := range changes {
		switch changes[i].Path {
		case "dst.txt":
			changes[i].Path = newFile
			changes[i].OldPath = oldFile
		case "other.txt":
			changes[i].Path = otherFile
		}
	}

	// NOTE: this test reassigns the package-level commitRename var — do not
	// add t.Parallel() to this test (see commitRename's doc comment).
	origCommitRename := commitRename
	commitRename = func(oldname, newname string) error {
		if newname == otherFile {
			return fmt.Errorf("injected phase 2 failure for %s", newname)
		}
		return origCommitRename(oldname, newname)
	}
	t.Cleanup(func() { commitRename = origCommitRename })

	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected error from injected Phase 2 failure, got nil")
	}

	srcContent, statErr := os.ReadFile(oldFile)
	if statErr != nil {
		t.Fatalf("src.txt should exist after rollback: %v", statErr)
	}
	if expected := "Keep this line\nOld content\n"; string(srcContent) != expected {
		t.Errorf("src.txt should hold its original content after rollback, expected %q, got %q", expected, string(srcContent))
	}

	if _, err := os.Stat(newFile); err == nil {
		t.Error("dst.txt should not exist after rollback")
	}

	otherContent, _ := os.ReadFile(otherFile)
	if expected := "Original other\n"; string(otherContent) != expected {
		t.Errorf("other.txt should be unchanged after rollback, expected %q, got %q", expected, string(otherContent))
	}

	leftovers, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	if len(leftovers) != 0 {
		t.Errorf("no temp or backup files should remain after a clean rollback, found: %v", leftovers)
	}
}

// TestApplyRenameWithEditBackupSourceFailureSelfRepairs tests the CRITICAL
// closed in mission-20260921-01, senior fix round 2: a rename-with-edit onto
// a target that already exists, where backing up the rename SOURCE (oldPath)
// fails after the TARGET (finalPath) has already been backed up.
//
// Before this fix, that failure returned directly from inside the
// oldBackupPath setup — a path that never reaches the shared commitErr
// self-repair block below it, and never reaches `committed` either — so
// finalPath was left missing on disk with its content stranded in an
// untracked .patch-backup-* file, and the returned error named neither. This
// test injects the failure via the backupRenameSource seam (distinct from
// commitRename, which only covers the later, already-self-repairing step)
// and asserts finalPath is restored and no backup or temp file is left
// behind.
//
// NOTE: this test reassigns the package-level backupRenameSource var — do
// not add t.Parallel() to this test (see backupRenameSource's doc comment).
func TestApplyRenameWithEditBackupSourceFailureSelfRepairs(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(oldFile, []byte("Keep this line\nOld content\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Target already exists — this is the branch where finalPath gets
	// backed up BEFORE the oldPath backup is attempted.
	newFile := filepath.Join(tmpDir, "dst.txt")
	if err := os.WriteFile(newFile, []byte("Pre-existing target\n"), 0o644); err != nil {
		t.Fatalf("failed to create pre-existing target: %v", err)
	}

	diff := `diff --git a/src.txt b/dst.txt
rename from src.txt
rename to dst.txt
--- a/src.txt
+++ b/dst.txt
@@ -1,2 +1,2 @@
 Keep this line
-Old content
+New content
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile

	// Inject a failure at the oldBackupPath rename step — after the target
	// has already been backed up, before commitRename ever runs.
	origBackupRenameSource := backupRenameSource
	backupRenameSource = func(_, _ string) error {
		return fmt.Errorf("injected failure backing up rename source")
	}
	t.Cleanup(func() { backupRenameSource = origBackupRenameSource })

	err = ApplyToAbsPath(changes)
	if err == nil {
		t.Fatal("expected error from injected backup-source failure, got nil")
	}

	// The pre-existing target must be restored, not left missing with its
	// content stranded in an untracked backup file.
	result, statErr := os.ReadFile(newFile)
	if statErr != nil {
		t.Fatalf("dst.txt should exist after self-repair, got error: %v", statErr)
	}
	if expected := "Pre-existing target\n"; string(result) != expected {
		t.Errorf("dst.txt should hold its original pre-existing content, expected %q, got %q", expected, string(result))
	}

	// The source file must be untouched (the injected failure fires before
	// oldPath is ever moved).
	srcContent, err := os.ReadFile(oldFile)
	if err != nil {
		t.Fatalf("src.txt should still exist: %v", err)
	}
	if expected := "Keep this line\nOld content\n"; string(srcContent) != expected {
		t.Errorf("src.txt should be unchanged, expected %q, got %q", expected, string(srcContent))
	}

	// No stray backup/temp files should remain: the backup made for
	// finalPath must have been consumed by self-repair, and the temp files
	// created for the edited content and for the (never-completed) oldPath
	// backup must both have been cleaned up.
	leftovers, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	if len(leftovers) != 0 {
		t.Errorf("no temp or backup files should remain after self-repair, found: %v", leftovers)
	}
}

// TestStageAndDiscardNoTempFiles tests that Stage followed by Discard leaves no temp files.
func TestStageAndDiscardNoTempFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("First line\nSecond line\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 First line
-Second line
+Updated line
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Count temp files before
	beforeTempFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	beforeCount := len(beforeTempFiles)

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Count temp files after Stage (should have created temp files)
	afterStageTempFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	afterStageCount := len(afterStageTempFiles)
	if afterStageCount <= beforeCount {
		t.Errorf("Stage should have created temp files, expected > %d temp files, got %d", beforeCount, afterStageCount)
	}

	// Discard the staged changes
	if err := handle.Discard(); err != nil {
		t.Fatalf("Discard failed: %v", err)
	}

	// Count temp files after Discard (should be back to before count)
	afterDiscardTempFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	afterDiscardCount := len(afterDiscardTempFiles)
	if afterDiscardCount != beforeCount {
		t.Errorf("Discard should clean up all temp files, expected %d temp files, got %d", beforeCount, afterDiscardCount)
		t.Logf("Leftover temp files: %v", afterDiscardTempFiles)
	}
}

// TestStageMidwayFailureNoTempFiles tests that a Stage failing partway leaves no temp files.
func TestStageMidwayFailureNoTempFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	// Create both files
	if err := os.WriteFile(file1, []byte("File 1 line 1\nFile 1 line 2\n"), 0o644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("File 2 line 1\nFile 2 line 2\n"), 0o644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Parse a diff with two files, where the second will fail due to context mismatch
	diff := `--- a/file1.txt
+++ b/file1.txt
@@ -1,2 +1,2 @@
 File 1 line 1
-File 1 line 2
+Modified 1
--- a/file2.txt
+++ b/file2.txt
@@ -1,2 +1,2 @@
 Wrong context that won't match
-File 2 line 2
+Modified 2
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = file1
	changes[1].Path = file2

	// Count temp files before
	beforeTempFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	beforeCount := len(beforeTempFiles)

	// Stage the changes - should fail on second file
	_, stageErr := Stage(changes)
	if stageErr == nil {
		t.Fatal("expected Stage to fail on conflicting patch")
	}

	// Count temp files after failed Stage (should be back to before count)
	afterTempFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".patch-*"))
	afterCount := len(afterTempFiles)
	if afterCount != beforeCount {
		t.Errorf("Stage failure should clean up all temp files, expected %d temp files, got %d", beforeCount, afterCount)
		t.Logf("Leftover temp files: %v", afterTempFiles)
	}

	// The first file should be unchanged (Stage never modifies disk on failure)
	content, _ := os.ReadFile(file1)
	if string(content) != "File 1 line 1\nFile 1 line 2\n" {
		t.Errorf("File 1 should be unchanged after Stage failure, got: %q", string(content))
	}
}

// TestCommitStagedBytes tests that the bytes committed are the bytes staged,
// not re-applied against a potentially modified file.
func TestCommitStagedBytes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Line 1\nLine 2\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 Line 1
-Line 2
+Modified Line 2
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Modify the target file on disk to have different content
	if err := os.WriteFile(testFile, []byte("Line 1\nDifferent Line 2\n"), 0o644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	// Commit - it should commit the staged content, not re-apply
	if err := handle.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify the committed content is what was staged, not a re-application
	result, _ := os.ReadFile(testFile)
	expected := "Line 1\nModified Line 2\n"
	if string(result) != expected {
		t.Errorf("expected committed content %q, got %q", expected, string(result))
	}
}

// TestCommitTwiceFails tests that calling Commit twice on the same handle fails.
func TestCommitTwiceFails(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Original\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1 +1 @@
-Original
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// First Commit should succeed
	if err := handle.Commit(); err != nil {
		t.Fatalf("first Commit failed: %v", err)
	}

	// Second Commit should fail
	if err := handle.Commit(); err == nil {
		t.Fatal("expected second Commit to fail, but it succeeded")
	}
}

// TestCommitAfterDiscard tests that calling Commit after Discard returns an error.
// This ensures the handle cannot be reused after discarding.
func TestCommitAfterDiscard(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Original\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1 +1 @@
-Original
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Discard the changes
	if err := handle.Discard(); err != nil {
		t.Fatalf("Discard failed: %v", err)
	}

	// Commit after Discard should fail
	if err := handle.Commit(); err == nil {
		t.Fatal("expected Commit after Discard to fail, but it succeeded")
	}

	// File should be unchanged
	result, _ := os.ReadFile(testFile)
	expected := "Original\n"
	if string(result) != expected {
		t.Errorf("expected file unchanged to %q, got %q", expected, string(result))
	}
}

// TestDiscardAfterCommit tests that calling Discard after Commit returns an error.
// The committed state indicates the handle is no longer staged, so discard should fail.
func TestDiscardAfterCommit(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Original\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1 +1 @@
-Original
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Commit the changes
	if err := handle.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Discard after Commit should fail
	if err := handle.Discard(); err == nil {
		t.Fatal("expected Discard after Commit to fail, but it succeeded")
	}

	// File should be modified
	result, _ := os.ReadFile(testFile)
	expected := "Modified\n"
	if string(result) != expected {
		t.Errorf("expected file modified to %q, got %q", expected, string(result))
	}
}

// TestFailedCommitState tests that a failed commit leaves the handle in a
// distinguishable state, and that subsequent operations on the handle report
// the true state rather than succeeding silently.
func TestFailedCommitState(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Original\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1 +1 @@
-Original
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Inject a failure in commitRename to simulate a commit failure
	oldCommitRename := commitRename
	commitRename = func(oldpath, newpath string) error {
		return fmt.Errorf("injected failure")
	}
	defer func() { commitRename = oldCommitRename }()

	// Commit should fail
	commitErr := handle.Commit()
	if commitErr == nil {
		t.Fatal("expected Commit to fail but it succeeded")
	}

	// Second Commit attempt should also fail (not silently succeed)
	// and report the actual state, not that it's already committed
	secondErr := handle.Commit()
	if secondErr == nil {
		t.Fatal("expected second Commit to fail after first failure, but it succeeded")
	}

	// The error message should mention the failed state, not a successful commit
	if strings.Contains(secondErr.Error(), "already been committed") {
		t.Errorf("error message should not claim successful commit: %v", secondErr)
	}

	// Discard should also fail with state error
	discardErr := handle.Discard()
	if discardErr == nil {
		t.Fatal("expected Discard to fail after failed commit, but it succeeded")
	}
	if !strings.Contains(discardErr.Error(), "invalid_state") || !strings.Contains(discardErr.Error(), "failed") {
		t.Errorf("error message should indicate failed state: %v", discardErr)
	}

	// File should be unchanged
	result, _ := os.ReadFile(testFile)
	expected := "Original\n"
	if string(result) != expected {
		t.Errorf("expected file unchanged to %q, got %q", expected, string(result))
	}
}

// TestDiscardReportsCleanupFailures tests that Discard reports when it cannot
// clean up temp files, rather than swallowing the error. This is tested by
// creating a handle with a pending entry that points to a path that cannot
// be removed (a directory or a file in a read-only location).
func TestDiscardReportsCleanupFailures(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("Original\n"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse a modification diff
	diff := `--- a/test.txt
+++ b/test.txt
@@ -1 +1 @@
-Original
+Modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = testFile

	// Stage the changes
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}

	// Manually inject a bad temp path into the handle's pending list
	// This simulates a scenario where cleanup will fail because the path
	// is invalid or inaccessible. We'll use a path that doesn't exist,
	// which rollback will try to remove and fail on.
	if len(handle.pending) > 0 {
		// Replace the temp path with one that's in a non-existent directory
		nonExistentDir := filepath.Join(tmpDir, "nonexistent", "subdir")
		handle.pending[0].tempPath = filepath.Join(nonExistentDir, ".patch-test")
	}

	// Discard should report cleanup failure
	discardErr := handle.Discard()
	if discardErr == nil {
		t.Fatal("expected Discard to report cleanup failure, but it succeeded")
	}

	if !strings.Contains(discardErr.Error(), "cleanup_failed") {
		t.Errorf("error kind should be cleanup_failed, got: %v", discardErr)
	}
}

// TestEmptyChangeSetStillCommits tests that an empty change set (zero files)
// still commits cleanly, which is the correct behavior for a no-op.
func TestEmptyChangeSetStillCommits(t *testing.T) {
	// Create an empty change set
	changes := []FileChange{}

	// Stage should succeed with an empty set
	handle, err := Stage(changes)
	if err != nil {
		t.Fatalf("Stage failed on empty change set: %v", err)
	}

	// Commit should succeed on an empty set
	if err := handle.Commit(); err != nil {
		t.Fatalf("Commit failed on empty change set: %v", err)
	}

	// Second Commit should fail (handle already committed)
	if err := handle.Commit(); err == nil {
		t.Fatal("expected second Commit to fail on empty set, but it succeeded")
	}
}

// TestApplyRenameWithEditPreservesExecBit tests that a rename-with-edit operation
// preserves the executable bit (0755) on the renamed file.
func TestApplyRenameWithEditPreservesExecBit(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.sh")
	content := []byte("#!/bin/bash\necho hello\n")
	if err := os.WriteFile(oldFile, content, 0o755); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	// Ensure the mode is set (umask-safe)
	if err := os.Chmod(oldFile, 0o755); err != nil {
		t.Fatalf("failed to chmod source file: %v", err)
	}
	// Verify the mode took
	info, err := os.Stat(oldFile)
	if err != nil {
		t.Fatalf("failed to stat source file: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Skipf("filesystem does not support the exec bit: got %v", info.Mode().Perm())
	}

	newFile := filepath.Join(tmpDir, "dst.sh")

	diff := `diff --git a/src.sh b/dst.sh
rename from src.sh
rename to dst.sh
--- a/src.sh
+++ b/dst.sh
@@ -1,2 +1,2 @@
 #!/bin/bash
-echo hello
+echo goodbye
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile

	if err := ApplyToAbsPath(changes); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the renamed file preserves the exec bit
	info, err = os.Stat(newFile)
	if err != nil {
		t.Fatalf("dst.sh should exist after rename: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("expected exec bit (0755), got %o", info.Mode().Perm())
	}

	// Verify edited content
	result, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("failed to read dst.sh: %v", err)
	}
	expected := "#!/bin/bash\necho goodbye\n"
	if string(result) != expected {
		t.Errorf("expected content %q, got %q", expected, string(result))
	}

	// Verify src no longer exists
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("src.sh should no longer exist after rename")
	}
}

// TestApplyRenameWithEditPreservesMode0644 tests that a rename-with-edit operation
// preserves the regular file mode (0644) on the renamed file.
func TestApplyRenameWithEditPreservesMode0644(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.txt")
	content := []byte("Line 1\nLine 2\n")
	if err := os.WriteFile(oldFile, content, 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	// Ensure the mode is set (umask-safe)
	if err := os.Chmod(oldFile, 0o644); err != nil {
		t.Fatalf("failed to chmod source file: %v", err)
	}
	// Verify the mode took
	info, err := os.Stat(oldFile)
	if err != nil {
		t.Fatalf("failed to stat source file: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("filesystem does not support the required file mode")
	}

	newFile := filepath.Join(tmpDir, "dst.txt")

	diff := `diff --git a/src.txt b/dst.txt
rename from src.txt
rename to dst.txt
--- a/src.txt
+++ b/dst.txt
@@ -1,2 +1,2 @@
 Line 1
-Line 2
+Line 2 modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile

	if err := ApplyToAbsPath(changes); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the renamed file preserves the 0644 mode (catches the 0600 defect)
	info, err = os.Stat(newFile)
	if err != nil {
		t.Fatalf("dst.txt should exist after rename: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("expected mode 0644, got %o", info.Mode().Perm())
	}

	// Verify edited content
	result, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("failed to read dst.txt: %v", err)
	}
	expected := "Line 1\nLine 2 modified\n"
	if string(result) != expected {
		t.Errorf("expected content %q, got %q", expected, string(result))
	}

	// Verify src no longer exists
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("src.txt should no longer exist after rename")
	}
}

// TestApplyRenameWithEditAppliesModeChange tests that a rename-with-edit operation
// that also carries a mode change applies the new mode to the file.
func TestApplyRenameWithEditAppliesModeChange(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "src.sh")
	content := []byte("#!/bin/bash\necho hello\n")
	if err := os.WriteFile(oldFile, content, 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	// Ensure the mode is set (umask-safe)
	if err := os.Chmod(oldFile, 0o644); err != nil {
		t.Fatalf("failed to chmod source file: %v", err)
	}
	// Verify the mode took
	info, err := os.Stat(oldFile)
	if err != nil {
		t.Fatalf("failed to stat source file: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("filesystem does not support the required file mode")
	}

	newFile := filepath.Join(tmpDir, "dst.sh")

	diff := `diff --git a/src.sh b/dst.sh
old mode 100644
new mode 100755
rename from src.sh
rename to dst.sh
--- a/src.sh
+++ b/dst.sh
@@ -1,2 +1,2 @@
 #!/bin/bash
-echo hello
+echo goodbye
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = newFile
	changes[0].OldPath = oldFile

	if err := ApplyToAbsPath(changes); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the renamed file has the new mode (0755)
	info, err = os.Stat(newFile)
	if err != nil {
		t.Fatalf("dst.sh should exist after rename: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("expected mode 0755 from mode change, got %o", info.Mode().Perm())
	}

	// Verify edited content
	result, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("failed to read dst.sh: %v", err)
	}
	expected := "#!/bin/bash\necho goodbye\n"
	if string(result) != expected {
		t.Errorf("expected content %q, got %q", expected, string(result))
	}

	// Verify src no longer exists
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("src.sh should no longer exist after rename")
	}
}

// TestApplyModifyPreservesExecBit tests that a plain modify operation
// (without rename) preserves the executable bit (0755) on the modified file.
func TestApplyModifyPreservesExecBit(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "script.sh")
	content := []byte("#!/bin/bash\necho hello\n")
	if err := os.WriteFile(srcFile, content, 0o755); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	// Ensure the mode is set (umask-safe)
	if err := os.Chmod(srcFile, 0o755); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}
	// Verify the mode took
	info, err := os.Stat(srcFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Skipf("filesystem does not support the exec bit: got %v", info.Mode().Perm())
	}

	diff := `diff --git a/script.sh b/script.sh
--- a/script.sh
+++ b/script.sh
@@ -1,2 +1,2 @@
 #!/bin/bash
-echo hello
+echo modified
`

	changes, err := Parse([]byte(diff))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	changes[0].Path = srcFile

	if err := ApplyToAbsPath(changes); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	// Verify the modified file preserves the exec bit
	info, err = os.Stat(srcFile)
	if err != nil {
		t.Fatalf("file should exist after modify: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("expected exec bit (0755) preserved on modify, got %o", info.Mode().Perm())
	}

	// Verify edited content
	result, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("failed to read script.sh: %v", err)
	}
	expected := "#!/bin/bash\necho modified\n"
	if string(result) != expected {
		t.Errorf("expected content %q, got %q", expected, string(result))
	}
}
