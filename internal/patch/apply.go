package patch

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// ApplyError represents an error during patch application.
type ApplyError struct {
	Kind    ApplyErrorKind
	Message string
	Hunk    *Hunk  // the hunk that failed (for patch_conflict)
	Content string // actual file content at the conflict location
}

// ApplyErrorKind categorizes application failures.
type ApplyErrorKind string

const (
	ErrPatchConflict      ApplyErrorKind = "patch_conflict"
	ErrLineEnding         ApplyErrorKind = "line_ending_mismatch"
	ErrBinaryNotSupported ApplyErrorKind = "binary_not_supported"
	ErrInvalidState       ApplyErrorKind = "invalid_state"
)

func (e *ApplyError) Error() string {
	return string(e.Kind) + ": " + e.Message
}

// LineEnding represents the line ending style detected in a file.
type LineEnding byte

const (
	LineEndingLF   LineEnding = '\n'
	LineEndingCRLF LineEnding = 0 // special value to represent \r\n
)

// pendingRename tracks a file change pending commit in Phase 2.
type pendingRename struct {
	tempPath    string // path to new content temp file
	finalPath   string // destination path
	oldPath     string // for renames: original path
	backupPath  string // backup of original finalPath (if it existed)
	hadOriginal bool   // whether finalPath existed before
	// oldBackupPath holds the pre-edit content of oldPath for a rename that
	// also edits content (tempPath != ""). A plain rename moves oldPath's
	// bytes to finalPath unchanged, so finalPath is its own record of what
	// oldPath held; an edited rename overwrites finalPath with different
	// bytes, so oldPath's original content needs its own backup to be
	// restorable on rollback.
	oldBackupPath string
	// disposition records what this entry represents after commit:
	// "rename", "restore", "create-or-modify", or "delete"
	disposition string
}

// commitRename moves an entry's content into its final path during Phase 2
// commit. It is a variable, not a direct call, so a test can inject a
// failure that fires only here — after Phase 1 has already created its temp
// file successfully — without resorting to filesystem permissions that would
// also block Phase 1's os.CreateTemp in the same directory. Production code
// never reassigns it.
//
// Not safe under t.Parallel(): reassigning a package-level var is a data
// race against any other test reading it concurrently. Nothing in this
// package calls t.Parallel() today, so there is no live race, but a test
// that reassigns commitRename must keep it that way — do not add
// t.Parallel() to a test in this file while commitRename or
// backupRenameSource may be overridden by another test running at the same
// time.
var commitRename = os.Rename

// backupRenameSource performs the os.Rename that moves a rename-with-edit's
// oldPath to its backup location (oldBackupPath) before commitRename
// overwrites finalPath with the edited content. It is a variable for the
// same reason commitRename is: a test needs to inject a failure at this
// exact point — after a pre-existing target at finalPath has already been
// backed up, before oldPath has been touched — without relying on
// filesystem permissions. Production code never reassigns it. Same
// t.Parallel() caution as commitRename above.
var backupRenameSource = os.Rename

// selfRepairFinalPath restores p.finalPath from p.backupPath when this
// entry already backed up a pre-existing file at finalPath but the calling
// site is about to return an error from somewhere other than the shared
// commitErr handling in Commit's Phase 2 loop (which performs this
// same repair for every failure that reaches it). Every early return in the
// rename-with-edit setup that can fire after the target has been backed up
// must call this before returning — that backup is otherwise invisible to
// rollback(), because this entry never reaches `committed`.
//
// Returns a slice (nil if nothing needed repair, or if there was nothing to
// repair) describing anything left inconsistent, in the same shape rollback()
// and the shared commitErr block use, so a caller can append it directly to
// an existing inconsistent-files list.
func selfRepairFinalPath(p pendingRename) []string {
	if !p.hadOriginal || p.backupPath == "" {
		return nil
	}
	if err := os.Rename(p.backupPath, p.finalPath); err != nil {
		return []string{fmt.Sprintf("%s (original content backed up at %s, restore failed: %v)", p.finalPath, p.backupPath, err)}
	}
	return nil
}

// rollback reverts all committed changes from backups and cleans up temp files.
// Returns a list of paths left in inconsistent state if any restore fails.
func rollback(committed []pendingRename, pending []pendingRename) []string {
	var inconsistent []string

	// Restore committed entries in reverse order
	for i := len(committed) - 1; i >= 0; i-- {
		p := committed[i]

		switch p.disposition {
		case "rename":
			if p.oldBackupPath != "" {
				// Rename-with-edit: finalPath holds edited content, and
				// oldPath's original bytes are held separately in
				// oldBackupPath (see pendingRename.oldBackupPath). Undo in
				// the opposite order: clear finalPath first, then restore
				// the source.
				if p.hadOriginal {
					if err := os.Rename(p.backupPath, p.finalPath); err != nil {
						inconsistent = append(inconsistent, fmt.Sprintf("%s (should hold pre-existing content backed up at %s)", p.finalPath, p.backupPath))
					}
				} else if err := os.Remove(p.finalPath); err != nil {
					inconsistent = append(inconsistent, fmt.Sprintf("%s (should not exist, remove failed)", p.finalPath))
				}
				if err := os.Rename(p.oldBackupPath, p.oldPath); err != nil {
					inconsistent = append(inconsistent, fmt.Sprintf("%s (original content backed up at %s)", p.oldPath, p.oldBackupPath))
				}
				continue
			}
			// Plain rename: finalPath holds oldPath's original bytes
			// unchanged, so moving it back is the exact inverse.
			if err := os.Rename(p.finalPath, p.oldPath); err != nil {
				inconsistent = append(inconsistent, fmt.Sprintf("%s (should be at %s, backup at %s)", p.finalPath, p.oldPath, p.backupPath))
				continue
			}
			// If original existed at finalPath, restore it from backup
			if p.hadOriginal {
				if err := os.Rename(p.backupPath, p.finalPath); err != nil {
					inconsistent = append(inconsistent, fmt.Sprintf("%s (backup at %s)", p.finalPath, p.backupPath))
				}
			}

		case "restore":
			// Original existed: restore from backup
			if err := os.Rename(p.backupPath, p.finalPath); err != nil {
				inconsistent = append(inconsistent, fmt.Sprintf("%s (backup at %s)", p.finalPath, p.backupPath))
			}

		case "create-or-modify":
			// No original: remove the file we created/modified
			if err := os.Remove(p.finalPath); err != nil {
				inconsistent = append(inconsistent, fmt.Sprintf("%s (remove failed)", p.finalPath))
			}

		case "delete":
			// Was a delete: restore from backup
			if err := os.Rename(p.backupPath, p.finalPath); err != nil {
				inconsistent = append(inconsistent, fmt.Sprintf("%s (backup at %s)", p.finalPath, p.backupPath))
			}

		default:
			// An unrecognised disposition must not silently no-op: that
			// would leave this entry unreverted with no record that
			// rollback ever saw it.
			inconsistent = append(inconsistent, fmt.Sprintf("%s (unknown rollback disposition %q, not reverted)", p.finalPath, p.disposition))
		}
	}

	// Clean up remaining temp files
	for _, p := range pending {
		if p.tempPath != "" {
			if err := os.Remove(p.tempPath); err != nil {
				inconsistent = append(inconsistent, fmt.Sprintf("%s (temp file, remove failed: %v)", p.tempPath, err))
			}
		}
	}

	return inconsistent
}

// StageHandleState represents the lifecycle state of a StageHandle.
type StageHandleState string

const (
	StateStaged    StageHandleState = "staged"    // changes staged, ready for commit or discard
	StateCommitted StageHandleState = "committed" // changes successfully committed
	StateDiscarded StageHandleState = "discarded" // changes discarded, handle spent
	StateFailed    StageHandleState = "failed"    // commit failed, changes rolled back
)

// StageHandle holds the result of a Stage operation.
// The staged changes are ready for commit or can be discarded.
// After calling Commit or Discard, the handle cannot be reused.
type StageHandle struct {
	pending []pendingRename
	moves   []HunkMove
	state   StageHandleState // tracks the handle's lifecycle state
}

// HunkMove records a hunk that was applied at a position other than the one its
// header declares. Positions are line numbers in the file before any hunk of the
// patch was applied, so a hunk after a hunk that added or removed lines is
// compared in the same frame as its header.
type HunkMove struct {
	Change    int  // index into the changes passed to Stage
	Hunk      int  // 1-based position of the hunk within its change
	Declared  int  // the hunk's old start, as its header gives it
	Applied   int  // where the hunk went: its first old line, or for an insertion the line it follows
	Insertion bool // the hunk has no old lines, so Declared and Applied are the line it follows
}

// Offset is how far the hunk moved from its declared position: negative when it
// went earlier in the file, positive when later.
func (m HunkMove) Offset() int { return m.Applied - m.Declared }

// Moves returns the hunks that were applied away from their declared position,
// in the order they were applied. It is empty when every hunk went where its
// header says.
func (h *StageHandle) Moves() []HunkMove { return h.moves }

// Stage runs Phase 1 of patch application: applies each change to a temp file
// in its directory, validating as it goes. On success, returns a handle whose
// Commit method will atomically commit the validated changes. On failure,
// cleans up any temp files and returns an error.
//
// The returned handle holds temp files in each target directory until Commit
// or Discard is called. If a caller holds a handle across an approval prompt,
// checkpoint, or other suspension point where an error or panic might occur,
// it MUST ensure Discard is called if the approval flow does not complete.
// Temp files are not automatically cleaned up if the handle is abandoned.
func Stage(changes []FileChange) (*StageHandle, error) {
	if len(changes) == 0 {
		return &StageHandle{state: StateStaged}, nil
	}

	// Phase 1: Apply each change to a temp file in its directory, validating as we go
	var pending []pendingRename
	var moves []HunkMove

	for i, change := range changes {
		if change.IsBinary {
			// Cleanup on failure
			rollback(nil, pending)
			return nil, &ApplyError{
				Kind:    ErrBinaryNotSupported,
				Message: fmt.Sprintf("binary files cannot be patched: %s", change.Path),
			}
		}

		var tempPath string
		var changeMoves []HunkMove
		var err error

		switch change.Op {
		case OpCreate:
			tempPath, err = applyCreate(&change)
		case OpModify:
			tempPath, changeMoves, err = applyModify(&change)
		case OpDelete:
			tempPath, err = applyDelete(&change)
		case OpRename:
			tempPath, changeMoves, err = applyRename(&change)
		default:
			rollback(nil, pending)
			return nil, &ApplyError{
				Kind:    "invalid_op",
				Message: fmt.Sprintf("unsupported operation: %v", change.Op),
			}
		}

		if err != nil {
			// Nothing has committed yet, so there is nothing to revert —
			// only pending temp files to clean up. Reuse rollback's cleanup
			// loop rather than duplicating it here: this file has already
			// drifted from that duplication once.
			rollback(nil, pending)
			return nil, err
		}

		for _, m := range changeMoves {
			m.Change = i
			moves = append(moves, m)
		}

		switch change.Op {
		case OpDelete:
			pending = append(pending, pendingRename{
				tempPath:    "",
				finalPath:   change.Path,
				disposition: "delete",
			})
		case OpRename:
			pending = append(pending, pendingRename{
				tempPath:    tempPath,
				finalPath:   change.Path,
				oldPath:     change.OldPath,
				disposition: "rename",
			})
		default:
			pending = append(pending, pendingRename{
				tempPath:    tempPath,
				finalPath:   change.Path,
				disposition: "create-or-modify",
			})
		}
	}

	return &StageHandle{pending: pending, moves: moves, state: StateStaged}, nil
}

// Commit atomically commits all staged changes with backup and rollback.
// For each entry, preserves the original before replacing it, then restores
// on failure. After successful commit, cleans up all backups. It is an error
// to call Commit after the handle has been used (committed or discarded).
func (h *StageHandle) Commit() error {
	// Ensure this handle is in the staged state
	if h.state != StateStaged {
		return &ApplyError{
			Kind:    ErrInvalidState,
			Message: fmt.Sprintf("cannot commit: handle is in %q state (must be %q)", h.state, StateStaged),
		}
	}

	pending := h.pending
	h.pending = nil // Clear pending

	if len(pending) == 0 {
		h.state = StateCommitted
		return nil
	}

	// Phase 2: Commit all changes atomically with backup/restore
	// For each entry, preserve the original before replacing it, then restore on failure
	var committed []pendingRename
	for i, p := range pending {
		var commitErr error

		switch {
		case p.oldPath != "":
			// Rename operation: oldPath -> finalPath
			// Check if target exists first
			if _, err := os.Stat(p.finalPath); err == nil {
				// Target already exists, back it up using CreateTemp for atomic safety
				backupFile, err := os.CreateTemp(filepath.Dir(p.finalPath), ".patch-backup-")
				if err != nil {
					h.state = StateFailed
					inconsistent := rollback(committed, pending)
					return &ApplyError{
						Kind:    "backup_failed",
						Message: fmt.Sprintf("failed to create backup temp for rename: %v; inconsistent files: %v", err, inconsistent),
					}
				}
				backupPath := backupFile.Name()
				_ = backupFile.Close()

				if err := os.Rename(p.finalPath, backupPath); err != nil {
					_ = os.Remove(backupPath)
					h.state = StateFailed
					inconsistent := rollback(committed, pending)
					return &ApplyError{
						Kind:    "backup_failed",
						Message: fmt.Sprintf("failed to backup existing file before rename: %v; inconsistent files: %v", err, inconsistent),
					}
				}
				p.backupPath = backupPath
				p.hadOriginal = true
			}

			if p.tempPath != "" {
				// Rename with edits: applyRename built the edited content at
				// tempPath, in finalPath's own directory. oldPath still
				// holds the untouched original. Back oldPath up before
				// vacating it, so a failure here or in a sibling file's
				// commit can restore it byte-for-byte instead of trying to
				// reconstruct it from the edited copy.
				oldBackupFile, err := os.CreateTemp(filepath.Dir(p.oldPath), ".patch-oldbackup-")
				if err != nil {
					// p.finalPath may already have been backed up above (if a
					// target existed at this path) — that backup is not yet
					// in `committed`, so rollback() alone cannot see it.
					// Restore it here before falling back to rollback() for
					// everything committed by prior entries.
					h.state = StateFailed
					selfInconsistent := selfRepairFinalPath(p)
					inconsistent := rollback(committed, pending)
					inconsistent = append(inconsistent, selfInconsistent...)
					return &ApplyError{
						Kind:    "backup_failed",
						Message: fmt.Sprintf("failed to create backup temp for rename source: %v; inconsistent files: %v", err, inconsistent),
					}
				}
				oldBackupPath := oldBackupFile.Name()
				_ = oldBackupFile.Close()

				if err := backupRenameSource(p.oldPath, oldBackupPath); err != nil {
					_ = os.Remove(oldBackupPath)
					// Same reasoning as above: self-repair p.finalPath before
					// rolling back everything else, since this entry never
					// reaches `committed`.
					h.state = StateFailed
					selfInconsistent := selfRepairFinalPath(p)
					inconsistent := rollback(committed, pending)
					inconsistent = append(inconsistent, selfInconsistent...)
					return &ApplyError{
						Kind:    "backup_failed",
						Message: fmt.Sprintf("failed to back up rename source before edit commit: %v; inconsistent files: %v", err, inconsistent),
					}
				}

				commitErr = commitRename(p.tempPath, p.finalPath)
				if commitErr == nil {
					p.oldBackupPath = oldBackupPath
				} else if restoreErr := os.Rename(oldBackupPath, p.oldPath); restoreErr != nil {
					// Same directory, temp file we just created — expected
					// to succeed. Surface it rather than lose it silently.
					commitErr = fmt.Errorf("%v (and failed to restore rename source from %s: %v)", commitErr, oldBackupPath, restoreErr)
				}
			} else {
				commitErr = commitRename(p.oldPath, p.finalPath)
			}

		case p.tempPath == "":
			// Delete operation: move to backup instead of removing
			backupFile, err := os.CreateTemp(filepath.Dir(p.finalPath), ".patch-backup-")
			if err != nil {
				h.state = StateFailed
				inconsistent := rollback(committed, pending)
				return &ApplyError{
					Kind:    "backup_failed",
					Message: fmt.Sprintf("failed to create backup temp for delete: %v; inconsistent files: %v", err, inconsistent),
				}
			}
			backupPath := backupFile.Name()
			_ = backupFile.Close()

			commitErr = os.Rename(p.finalPath, backupPath)
			if commitErr == nil {
				p.backupPath = backupPath
				p.hadOriginal = true
			} else {
				_ = os.Remove(backupPath)
			}

		default:
			// Create or modify: backup the original first (if it exists)
			if _, err := os.Stat(p.finalPath); err == nil {
				// File exists, must back it up
				backupFile, err := os.CreateTemp(filepath.Dir(p.finalPath), ".patch-backup-")
				if err != nil {
					h.state = StateFailed
					inconsistent := rollback(committed, pending)
					return &ApplyError{
						Kind:    "backup_failed",
						Message: fmt.Sprintf("failed to create backup temp for file: %v; inconsistent files: %v", err, inconsistent),
					}
				}
				backupPath := backupFile.Name()
				_ = backupFile.Close()

				if err := os.Rename(p.finalPath, backupPath); err != nil {
					_ = os.Remove(backupPath)
					h.state = StateFailed
					inconsistent := rollback(committed, pending)
					return &ApplyError{
						Kind:    "backup_failed",
						Message: fmt.Sprintf("failed to backup file before update: %v; inconsistent files: %v", err, inconsistent),
					}
				}
				p.backupPath = backupPath
				p.hadOriginal = true
				p.disposition = "restore"
			} else {
				// File did not exist, this is a create
				p.disposition = "create-or-modify"
			}

			// Now rename the new content into place
			commitErr = commitRename(p.tempPath, p.finalPath)
		}

		if commitErr != nil {
			// This entry never reaches `committed` below, so rollback()
			// cannot see it — but a pre-existing target may already have
			// been moved to p.backupPath (the rename and create-or-modify
			// branches above both back up before they move new content in).
			// Put it back before touching anything else, or this file is
			// left with content missing and a backup nobody's told about.
			h.state = StateFailed
			selfInconsistent := selfRepairFinalPath(p)

			// Revert all previously committed entries.
			inconsistent := rollback(committed, pending)
			inconsistent = append(inconsistent, selfInconsistent...)
			if len(inconsistent) > 0 {
				return &ApplyError{
					Kind:    "partial_rollback",
					Message: fmt.Sprintf("failed to commit file change: %v; %d of %d files were already committed and are being rolled back; rollback incomplete, inconsistent files: %v", commitErr, len(committed), len(pending), inconsistent),
				}
			}
			return &ApplyError{
				Kind:    "rename_failed",
				Message: fmt.Sprintf("failed to commit file change: %v; %d of %d files were already committed and were rolled back cleanly", commitErr, len(committed), len(pending)),
			}
		}

		// Update the pending entry with backup info and mark as committed
		pending[i] = p
		committed = append(committed, p)
	}

	// All changes committed successfully, now remove all backups
	for _, p := range committed {
		if p.backupPath != "" {
			_ = os.Remove(p.backupPath)
		}
		if p.oldBackupPath != "" {
			_ = os.Remove(p.oldBackupPath)
		}
	}

	h.state = StateCommitted
	return nil
}

// Discard abandons the staged work and removes every temp file it created.
// After calling Discard, the handle cannot be reused. Discard reports any
// failure to clean up temp files, so the caller can decide whether to retry
// or escalate. A non-nil return means some temp files could not be removed.
func (h *StageHandle) Discard() error {
	// Ensure this handle is in the staged state
	if h.state != StateStaged {
		return &ApplyError{
			Kind:    ErrInvalidState,
			Message: fmt.Sprintf("cannot discard: handle is in %q state (must be %q)", h.state, StateStaged),
		}
	}

	// Clean up all pending temp files, collecting any failures
	inconsistent := rollback(nil, h.pending)
	h.pending = nil
	h.state = StateDiscarded

	// Report any cleanup failures
	if len(inconsistent) > 0 {
		return &ApplyError{
			Kind:    "cleanup_failed",
			Message: fmt.Sprintf("discard abandoned changes but failed to clean up temp files: %v", inconsistent),
		}
	}
	return nil
}

// ApplyToAbsPath applies a set of file changes to absolute paths.
// All paths in changes must be absolute. This function implements atomicity:
// changes are applied to temp files in the same directories, validated,
// then renamed into place. If any operation fails, previously renamed files
// are reverted from backups.
func ApplyToAbsPath(changes []FileChange) error {
	handle, err := Stage(changes)
	if err != nil {
		return err
	}
	return handle.Commit()
}

// ── Operation-Specific Handlers (Phase 1) ───────────────────────────────────

// applyCreate applies a create operation, returning the path to the temp file.
func applyCreate(change *FileChange) (string, error) {
	// Check if file already exists - creates should fail on existing files
	// (this check happens in Phase 2 backup, but let's catch it early)
	if _, err := os.Stat(change.Path); err == nil {
		return "", &ApplyError{
			Kind:    "file_exists",
			Message: fmt.Sprintf("cannot create file, already exists: %s", change.Path),
		}
	}

	// Create file with hunks
	content := bytes.Buffer{}

	for _, hunk := range change.Hunks {
		for _, line := range hunk.Lines {
			switch line.Prefix {
			case '+':
				content.WriteString(line.Content)
				content.WriteByte('\n')
			case '\\':
				// This is a "\ No newline at end of file" marker
				// Don't add newline
			}
		}
	}

	// Handle trailing newline: if the last line is a backslash marker,
	// the file should not end with newline
	shouldHaveTrailingNewline := true
	if len(change.Hunks) > 0 {
		lastHunk := change.Hunks[len(change.Hunks)-1]
		if len(lastHunk.Lines) > 0 && lastHunk.Lines[len(lastHunk.Lines)-1].Prefix == '\\' {
			shouldHaveTrailingNewline = false
		}
	}

	// If content ends with newline but shouldn't, trim it
	contentBytes := content.Bytes()
	if !shouldHaveTrailingNewline && len(contentBytes) > 0 && contentBytes[len(contentBytes)-1] == '\n' {
		contentBytes = contentBytes[:len(contentBytes)-1]
	}

	// Write to temp file in same directory as target
	dir := filepath.Dir(change.Path)
	if dir == "" {
		dir = "."
	}
	tmpFile, err := os.CreateTemp(dir, ".patch-")
	if err != nil {
		return "", &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to create temp file: %v", err),
		}
	}

	_, err = tmpFile.Write(contentBytes)
	_ = tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to write temp file: %v", err),
		}
	}

	// Apply mode: use specified mode or default to 0644 for consistency
	// (CreateTemp creates at 0600, but created files should generally be readable)
	if change.CreateMode != 0 {
		_ = os.Chmod(tmpFile.Name(), change.CreateMode)
	} else {
		_ = os.Chmod(tmpFile.Name(), 0o644) // #nosec G302 -- created by CreateTemp in filepath.Dir(change.Path); change.Path is workspace-resolved by resolve(t.WS, …) in internal/tool/apply_patch.go before Stage; 0644 is the mode for a new source file
	}

	return tmpFile.Name(), nil
}

// maxRelocation is how many lines a hunk may move from its expected position.
// A hunk whose old lines match nowhere within this distance is a patch_conflict,
// even when they match farther away: a short block can match exactly at a
// distant, unrelated place.
const maxRelocation = 10

// applyModify applies a modify operation, returning the path to the temp file
// and the hunks that were applied away from their declared position. Each hunk
// is placed at its declared position when its old lines match there exactly.
// Otherwise it goes at the nearest exact match within maxRelocation lines of that
// position that does not overlap the previous hunk (see locateHunk). A pure
// insertion is placed at its declared point and never moved from it, though a
// shift found by an earlier hunk carries into where that point is.
func applyModify(change *FileChange) (string, []HunkMove, error) {
	// Read the original file
	content, err := os.ReadFile(change.Path)
	if err != nil {
		return "", nil, &ApplyError{
			Kind:    "file_not_found",
			Message: fmt.Sprintf("cannot read file: %v", err),
		}
	}

	// Detect line ending
	le := detectLineEnding(content)

	// Split into lines (preserving the line ending style)
	lines := splitLines(content)

	// Apply each hunk. A hunk's old side is its ' ' and '-' lines. locateHunk
	// finds where that block stands in the current lines: at the expected
	// position first, then at the nearest exact match within maxRelocation lines.
	// offset is the shift from declared positions to current positions, resize is
	// the net change in line count made by the hunks already applied, and floor is
	// the end of the previous hunk, the first index this hunk may touch.
	var moves []HunkMove
	offset := 0
	resize := 0
	floor := 0
	for i := range change.Hunks {
		hunk := &change.Hunks[i]
		hunkNo := i + 1
		old := oldSide(hunk.Lines)

		// pos is where the old side begins in lines. declaredEnd is where the old
		// side ends in declared coordinates: OldStart-1 plus its length, or
		// OldStart for a pure insertion, which goes after line OldStart. The next
		// hunk's offset is pos plus the new length, less declaredEnd, so any shift
		// found here carries into the hunks after it. applied is pos in the line
		// numbers of the file before any hunk ran: the first old line, or for an
		// insertion the line it follows. It is compared with OldStart to find a
		// hunk applied away from its declared position.
		var pos, declaredEnd, applied int
		if len(old) == 0 {
			// Pure insertion: "-N,0" inserts after old line N, and N=0 is the
			// start of the file. An insertion is never relocated, but it follows
			// any shift an earlier hunk carried.
			pos = hunk.OldStart + offset
			declaredEnd = hunk.OldStart
			applied = pos - resize
			if pos > len(lines) {
				return "", nil, &ApplyError{
					Kind:    ErrPatchConflict,
					Message: fmt.Sprintf("hunk %d (old line %d): insertion runs past end of file (file has %d lines)", hunkNo, hunk.OldStart, len(lines)),
					Hunk:    hunk,
					Content: strings.Join(lines, lineSep(le)),
				}
			}
			if pos < floor {
				return "", nil, &ApplyError{
					Kind:    ErrPatchConflict,
					Message: fmt.Sprintf("hunk %d (old line %d): insertion point is before the start of the file or inside the previous hunk", hunkNo, hunk.OldStart),
					Hunk:    hunk,
					Content: strings.Join(lines, lineSep(le)),
				}
			}
		} else {
			expected := hunk.OldStart - 1 + offset
			found, alt, status := locateHunk(lines, old, expected, floor, maxRelocation)
			switch status {
			case locateAmbiguous:
				return "", nil, &ApplyError{
					Kind:    ErrPatchConflict,
					Message: fmt.Sprintf("hunk %d (old line %d): ambiguous: the old lines match at lines %d and %d", hunkNo, hunk.OldStart, found+1, alt+1),
					Hunk:    hunk,
					Content: getContextWindow(lines, found, len(old)),
				}
			case locateNone:
				// Look past the window for the message only: a match found
				// there is reported and never applied.
				outside, _, _ := locateHunk(lines, old, expected, floor, noWindow)
				return "", nil, &ApplyError{
					Kind:    ErrPatchConflict,
					Message: missMessage(lines, old, expected, outside, hunkNo, hunk.OldStart),
					Hunk:    hunk,
					Content: getContextWindow(lines, clampIndex(expected, len(lines)), len(old)),
				}
			case locateFound:
				// The match is applied below.
			}
			pos = found
			declaredEnd = hunk.OldStart - 1 + len(old)
			applied = pos - resize + 1
		}

		// Build the new lines for this hunk: context as the file has it, added
		// lines from the patch, removed lines dropped.
		newLines := make([]string, 0, len(hunk.Lines))
		k := pos
		for _, line := range hunk.Lines {
			switch line.Prefix {
			case ' ':
				newLines = append(newLines, lines[k])
				k++
			case '-':
				k++
			case '+':
				newLines = append(newLines, strings.TrimSuffix(line.Content, "\r"))
			}
		}

		// Replace the old side with the new lines
		rebuilt := make([]string, 0, len(lines)-len(old)+len(newLines))
		rebuilt = append(rebuilt, lines[:pos]...)
		rebuilt = append(rebuilt, newLines...)
		rebuilt = append(rebuilt, lines[pos+len(old):]...)
		lines = rebuilt

		floor = pos + len(newLines)
		offset = floor - declaredEnd
		resize += len(newLines) - len(old)
		if applied != hunk.OldStart {
			moves = append(moves, HunkMove{Hunk: hunkNo, Declared: hunk.OldStart, Applied: applied, Insertion: len(old) == 0})
		}
	}

	// Conversion check. A patch "carries endings" if any ' ', '-' or '+' line
	// ends in \r, as a git diff of a CRLF file does. Such a patch must add lines
	// in the file's own ending:
	//   - CRLF file: every '+' line must end in \r
	//   - LF file: no '+' line may end in \r
	// A '+' line directly followed by "\ No newline at end of file" has no line
	// terminator at all, so it is exempt. A patch with no \r anywhere (the form a
	// model writes, or a git diff of an LF file) is not checked: its added lines
	// take the file's ending. What is rejected: a CRLF-file patch that mixes \r
	// and bare added lines, and an added line ending in \r on an LF file.
	patchCarriesEndings := false
	for _, hunk := range change.Hunks {
		for _, line := range hunk.Lines {
			if line.Prefix == ' ' || line.Prefix == '-' || line.Prefix == '+' {
				if strings.HasSuffix(line.Content, "\r") {
					patchCarriesEndings = true
					break
				}
			}
		}
		if patchCarriesEndings {
			break
		}
	}

	if patchCarriesEndings {
		isCRLFFile := le == LineEndingCRLF
		for _, hunk := range change.Hunks {
			for i, line := range hunk.Lines {
				if line.Prefix == '+' {
					// Skip this line if it's followed by a "\ No newline at end of file" marker.
					// Such a line has no terminator, so it cannot violate the ending rule.
					if i+1 < len(hunk.Lines) && hunk.Lines[i+1].Prefix == '\\' {
						continue
					}
					hasEnding := strings.HasSuffix(line.Content, "\r")
					if isCRLFFile && !hasEnding {
						return "", nil, &ApplyError{
							Kind:    ErrLineEnding,
							Message: fmt.Sprintf("patch adds line without CRLF ending to CRLF file: %q", line.Content),
						}
					}
					if !isCRLFFile && hasEnding {
						return "", nil, &ApplyError{
							Kind:    ErrLineEnding,
							Message: fmt.Sprintf("patch adds line with CRLF ending to LF file: %q", line.Content),
						}
					}
				}
			}
		}
	}

	// Check for trailing newline marker in last hunk
	shouldHaveTrailingNewline := true
	if len(change.Hunks) > 0 {
		lastHunk := change.Hunks[len(change.Hunks)-1]
		if len(lastHunk.Lines) > 0 && lastHunk.Lines[len(lastHunk.Lines)-1].Prefix == '\\' {
			shouldHaveTrailingNewline = false
		}
	}

	// Reconstruct file content
	result := bytes.Buffer{}
	for i, line := range lines {
		result.WriteString(line)
		if i < len(lines)-1 || shouldHaveTrailingNewline {
			result.WriteString(lineSep(le))
		}
	}

	// Backstop check: verify line ending did not change.
	// The explicit conversion check above rejects converting patches before
	// writing; this block is a safety backstop to catch any unexpected changes.
	if len(change.Hunks) > 0 {
		// Check if applying the patch would change the line ending style
		newContent := result.Bytes()
		newLE := detectLineEnding(newContent)
		// Report if the detected line ending has changed
		if newLE != le {
			return "", nil, &ApplyError{
				Kind:    ErrLineEnding,
				Message: fmt.Sprintf("patch would change line ending style from %s to %s", leString(le), leString(newLE)),
			}
		}
	}

	// Write to temp file
	dir := filepath.Dir(change.Path)
	if dir == "" {
		dir = "."
	}
	tmpFile, err := os.CreateTemp(dir, ".patch-")
	if err != nil {
		return "", nil, &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to create temp file: %v", err),
		}
	}

	_, err = tmpFile.Write(result.Bytes())
	_ = tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to write temp file: %v", err),
		}
	}

	// Preserve permissions
	if info, err := os.Stat(change.Path); err == nil {
		_ = os.Chmod(tmpFile.Name(), info.Mode())
	}

	// Apply mode change if specified
	if change.NewMode != 0 {
		_ = os.Chmod(tmpFile.Name(), change.NewMode)
	}

	return tmpFile.Name(), moves, nil
}

// applyDelete applies a delete operation.
func applyDelete(change *FileChange) (string, error) {
	// For delete, we don't create a temp file
	// Just verify the file exists
	if _, err := os.Stat(change.Path); err != nil {
		return "", &ApplyError{
			Kind:    "file_not_found",
			Message: fmt.Sprintf("cannot delete file: %v", err),
		}
	}
	return "", nil
}

// applyRename applies a rename operation.
func applyRename(change *FileChange) (string, []HunkMove, error) {
	// For rename, we might have hunks to apply
	if len(change.Hunks) == 0 {
		// Pure rename: no hunks to apply
		if _, err := os.Stat(change.OldPath); err != nil {
			return "", nil, &ApplyError{
				Kind:    "file_not_found",
				Message: fmt.Sprintf("cannot rename file: %v", err),
			}
		}
		return "", nil, nil
	}

	// Rename with edits: apply hunks to temp file, then rename
	// Create temp file in the target directory
	dir := filepath.Dir(change.Path)
	if dir == "" {
		dir = "."
	}
	tmpFile, err := os.CreateTemp(dir, ".patch-")
	if err != nil {
		return "", nil, &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to create temp file: %v", err),
		}
	}
	_ = tmpFile.Close()

	// Copy old file to temp
	oldContent, err := os.ReadFile(change.OldPath)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, &ApplyError{
			Kind:    "file_not_found",
			Message: fmt.Sprintf("cannot read old file for rename: %v", err),
		}
	}

	err = os.WriteFile(tmpFile.Name(), oldContent, 0o644) // #nosec G703,G306 -- tmpFile already exists (CreateTemp, 0600) so the perm argument is inert; path is CreateTemp in filepath.Dir(change.Path), workspace-resolved by resolve(t.WS, …) in internal/tool/apply_patch.go before Stage; the copy's mode is set explicitly below
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to write temp file: %v", err),
		}
	}

	// CreateTemp makes the copy 0600, and applyModify copies whatever mode the
	// copy has, so set the source file's permission bits on the copy first. Only
	// Perm() is carried: an edit written through apply_patch does not carry
	// setuid, setgid or sticky forward, unlike a plain modify or a pure rename.
	info, err := os.Stat(change.OldPath)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to stat old file for permissions: %v", err),
		}
	}

	err = os.Chmod(tmpFile.Name(), info.Mode().Perm())
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, &ApplyError{
			Kind:    "io_error",
			Message: fmt.Sprintf("failed to set file permissions: %v", err),
		}
	}

	// Apply hunks to temp file
	// Temporarily change change.Path to temp file path for applyModify logic
	origPath := change.Path
	change.Path = tmpFile.Name()
	editedPath, moves, applyErr := applyModify(change)
	change.Path = origPath

	if applyErr != nil {
		_ = os.Remove(tmpFile.Name())
		if editedPath != "" {
			_ = os.Remove(editedPath)
		}
		return "", nil, applyErr
	}

	// Return the edited temp file path, clean up the unedited copy
	_ = os.Remove(tmpFile.Name())
	return editedPath, moves, nil
}

// ── Helper Functions ────────────────────────────────────────────────────────

// detectLineEnding detects the dominant line ending in content.
// Counts \r\n pairs and bare \n, returns the dominant style.
func detectLineEnding(content []byte) LineEnding {
	crlfCount := 0
	lfCount := 0

	for i := 0; i < len(content); i++ {
		if i+1 < len(content) && content[i] == '\r' && content[i+1] == '\n' {
			crlfCount++
			i++ // Skip the \n since we've counted the pair
		} else if content[i] == '\n' {
			lfCount++
		}
	}

	// If no line endings found, default to LF
	if crlfCount == 0 && lfCount == 0 {
		return LineEndingLF
	}

	// Return dominant style
	if crlfCount > lfCount {
		return LineEndingCRLF
	}
	return LineEndingLF
}

func lineSep(le LineEnding) string {
	if le == LineEndingCRLF {
		return "\r\n"
	}
	return "\n"
}

func leString(le LineEnding) string {
	if le == LineEndingCRLF {
		return "CRLF"
	}
	return "LF"
}

func splitLines(content []byte) []string {
	text := string(content)

	// Normalize all line endings to \n for consistent splitting.
	// This handles files with mixed line endings (where detectLineEnding
	// may have resolved a tie), ensuring stray \r characters don't fuse
	// to line content.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")

	// Remove empty trailing element if content ended with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

func getContextWindow(lines []string, centerLine int, hunkOldLines int) string {
	start := centerLine
	if start > 0 {
		start--
	}
	end := centerLine + hunkOldLines
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// locateStatus is the outcome of locateHunk.
type locateStatus int

const (
	locateNone      locateStatus = iota // no index at or after the floor matches
	locateFound                         // one matching index is nearest
	locateAmbiguous                     // two matching indexes are equally near
)

// oldSide returns the lines a hunk expects in the file, in order: its ' ' and
// '-' lines, with a trailing '\r' removed. It is empty for a pure insertion.
func oldSide(hunkLines []HunkLine) []HunkLine {
	var old []HunkLine
	for _, line := range hunkLines {
		if line.Prefix == ' ' || line.Prefix == '-' {
			old = append(old, HunkLine{Prefix: line.Prefix, Content: strings.TrimSuffix(line.Content, "\r")})
		}
	}
	return old
}

// matchesAt reports whether every old line matches lines exactly, with the
// first of them at index pos. A block that would run past either end of lines
// does not match.
func matchesAt(lines []string, old []HunkLine, pos int) bool {
	if pos < 0 || pos+len(old) > len(lines) {
		return false
	}
	for k, line := range old {
		if lines[pos+k] != line.Content {
			return false
		}
	}
	return true
}

// noWindow is the locateHunk window that excludes no distance.
const noWindow = math.MaxInt

// locateHunk finds where a hunk's old side begins in lines. It considers every
// index from floor to the last index that leaves room for the whole block and
// lies within window lines of expected, and keeps the one nearest to expected
// among those where the block matches exactly. A match at expected is returned
// at once, since only one index lies at that distance. When two matching indexes
// are equally near, the result is locateAmbiguous, found is the lower of the two,
// and alt is the higher. When nothing matches, found is -1. Nothing is fuzzed: a
// matching index must match every old line byte for byte.
func locateHunk(lines []string, old []HunkLine, expected, floor, window int) (found, alt int, status locateStatus) {
	if expected >= floor && matchesAt(lines, old, expected) {
		return expected, -1, locateFound
	}
	found, alt, status = -1, -1, locateNone
	bestDist := 0
	for p := floor; p <= len(lines)-len(old); p++ {
		dist := p - expected
		if dist < 0 {
			dist = -dist
		}
		if dist > window || !matchesAt(lines, old, p) {
			continue
		}
		switch {
		case status == locateNone || dist < bestDist:
			found, alt, bestDist, status = p, -1, dist, locateFound
		case dist == bestDist:
			alt, status = p, locateAmbiguous
		}
	}
	return found, alt, status
}

// firstMismatch describes the first old line that does not match lines when the
// block begins at index pos. It returns "" when every old line matches there.
func firstMismatch(lines []string, old []HunkLine, pos int) string {
	for k, line := range old {
		p := pos + k
		kind := "context"
		if line.Prefix == '-' {
			kind = "removal"
		}
		switch {
		case p < 0:
			return kind + " starts before the start of the file"
		case p >= len(lines):
			return fmt.Sprintf("%s runs past end of file (file has %d lines)", kind, len(lines))
		case lines[p] != line.Content:
			return fmt.Sprintf("%s mismatch: expected %q, got %q", kind, line.Content, lines[p])
		}
	}
	return ""
}

// missMessage explains why a hunk's old side matched nowhere within maxRelocation
// lines of expected. It describes the first failing line at expected. outside is
// the index of a match found beyond that window, or -1 when none exists at or
// after the end of the previous hunk. When no line fails at expected, the old
// side matches there but the search excluded that position, so it lies before the
// end of the previous hunk. The message says so instead.
func missMessage(lines []string, old []HunkLine, expected, outside, hunkNo, oldStart int) string {
	head := fmt.Sprintf("hunk %d (old line %d): ", hunkNo, oldStart)
	detail := firstMismatch(lines, old, expected)
	if detail == "" {
		if outside < 0 {
			return head + "the old lines match only inside the previous hunk"
		}
		detail = "the old lines match inside the previous hunk"
	}
	if outside < 0 {
		return head + detail + "; no matching position was found in the file"
	}
	return fmt.Sprintf("%s%s; no matching position was found within %d lines of line %d, but the old lines match at line %d", head, detail, maxRelocation, expected+1, outside+1)
}

// clampIndex limits index to the closed range from 0 to n.
func clampIndex(index, n int) int {
	if index < 0 {
		return 0
	}
	if index > n {
		return n
	}
	return index
}
