// Package patch parses and renders unified diffs.
//
// The parser is pure and testable, with no dependencies on the tool or
// workspace layers. It validates diffs at parse time to let the applier
// assume well-formed input. Error types are defined here; the tool layer
// maps them to tool.Kind when returning results to the model.
package patch

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Op represents a file operation type.
type Op uint8

// File operation constants.
const (
	OpModify Op = iota
	OpCreate
	OpDelete
	OpRename
)

// HunkLine represents a single line in a hunk, preserving its prefix.
// Prefix is one of: ' ' (context), '+' (addition), '-' (removal), '\'
// (special marker like "\ No newline at end of file").
type HunkLine struct {
	Prefix  byte
	Content string
}

// Hunk represents a contiguous block of changes with context.
// OldStart/OldLines describe the line range in the original file;
// NewStart/NewLines describe the line range in the new file.
// Lines contains the hunk's content with prefixes preserved.
type Hunk struct {
	OldStart, OldLines int
	NewStart, NewLines int
	Lines              []HunkLine
}

// FileChange represents changes to a single file.
type FileChange struct {
	Op       Op     // OpModify, OpCreate, OpDelete, OpRename
	Path     string // workspace-relative; the target for renames, stripped of a/b/ prefixes
	OldPath  string // rename source, else ""
	Hunks    []Hunk
	NewMode  os.FileMode // zero unless the mode changes
	IsBinary bool

	// Internal: preserved for round-trip rendering
	originalAPath string
	originalBPath string
	HasHeader     bool        // true if diff began with "diff --git"; false for header-less format
	CreateMode    os.FileMode // preserve mode for create operations (e.g., 0o755)
}

// ParseError is returned when a diff cannot be parsed.
type ParseError struct {
	Kind    ParseErrorKind
	Message string
}

// ParseErrorKind categorizes parse failures.
type ParseErrorKind string

const (
	ErrMissingHeader     ParseErrorKind = "missing_header"
	ErrBadMode           ParseErrorKind = "bad_mode"
	ErrHunkCountMismatch ParseErrorKind = "hunk_count_mismatch"
)

func (e *ParseError) Error() string {
	return string(e.Kind) + ": " + e.Message
}

// ── Parse ───────────────────────────────────────────────────────────────────

// Parse reads a unified diff and returns the file changes.
// It parses git diff output and hand-written unified diffs alike.
func Parse(data []byte) ([]FileChange, error) {
	p := &parser{
		lines: strings.Split(string(data), "\n"),
	}
	return p.parse()
}

type parser struct {
	lines []string
	pos   int
}

func (p *parser) hasLine() bool {
	return p.pos < len(p.lines)
}

func (p *parser) peek() string {
	if p.pos < len(p.lines) {
		return p.lines[p.pos]
	}
	return ""
}

func (p *parser) next() string {
	if p.pos < len(p.lines) {
		line := p.lines[p.pos]
		p.pos++
		return line
	}
	return ""
}

// peekTrimmed returns the peeked line with \r trimmed, for header/metadata comparison
func (p *parser) peekTrimmed() string {
	return strings.TrimSuffix(p.peek(), "\r")
}

func (p *parser) parse() ([]FileChange, error) {
	var changes []FileChange

	for p.hasLine() {
		line := p.peekTrimmed()

		// Skip empty lines
		if line == "" {
			p.next()
			continue
		}

		// Each file change starts with "diff --git" (traditional) or "--- " (header-less)
		if strings.HasPrefix(line, "diff --git ") {
			change, err := p.parseFileChange(true)
			if err != nil {
				return nil, err
			}
			changes = append(changes, change)
		} else if strings.HasPrefix(line, "--- ") {
			// Header-less diff: start with the --- line
			change, err := p.parseFileChange(false)
			if err != nil {
				return nil, err
			}
			changes = append(changes, change)
		} else {
			p.next()
		}
	}

	return changes, nil
}

func (p *parser) parseFileChange(hasHeader bool) (FileChange, error) {
	change := FileChange{
		Op:        OpModify,
		HasHeader: hasHeader,
	}

	// Parse "diff --git a/... b/..." if present
	if hasHeader {
		diffLine := p.peekTrimmed()
		p.next()

		aPath, bPath := extractPaths(diffLine)
		if aPath == "" || bPath == "" {
			return FileChange{}, &ParseError{
				Kind:    ErrMissingHeader,
				Message: "malformed diff --git header",
			}
		}

		change.Path = bPath
		change.originalAPath = aPath
		change.originalBPath = bPath
	}

	// Parse metadata lines
	seenMinusLine := false
	seenPlusLine := false

	for p.hasLine() {
		line := p.peekTrimmed()

		// Stop at next file (traditional format)
		if strings.HasPrefix(line, "diff --git ") {
			break
		}

		// For header-less diffs: in multi-file format, stop at a second --- line
		// (which marks the start of the next file's section)
		if !hasHeader && strings.HasPrefix(line, "--- ") && seenMinusLine && seenPlusLine {
			break
		}

		if line == "" {
			p.next()
			continue
		}

		// Check for missing +++ line before hunk
		if strings.HasPrefix(line, "@@") && seenMinusLine && !seenPlusLine {
			return FileChange{}, &ParseError{
				Kind:    ErrMissingHeader,
				Message: "missing +++ line after --- line",
			}
		}

		// new file mode / deleted file mode (traditional header format only)
		if strings.HasPrefix(line, "new file mode ") {
			mode := strings.TrimPrefix(line, "new file mode ")
			change.Op = OpCreate
			// Validate create mode: only 100644 and 100755 are allowed
			if mode != "100644" && mode != "100755" {
				return FileChange{}, &ParseError{
					Kind:    ErrBadMode,
					Message: fmt.Sprintf("invalid create mode: %s (only 100644 and 100755 are supported)", mode),
				}
			}
			if mode == "100755" {
				change.CreateMode = 0o755
			} else {
				change.CreateMode = 0o644
			}
			p.next()
			continue
		}

		if strings.HasPrefix(line, "deleted file mode ") {
			mode := strings.TrimPrefix(line, "deleted file mode ")
			change.Op = OpDelete
			// Validate deleted file mode: only 100644 and 100755 are allowed
			if mode != "100644" && mode != "100755" {
				return FileChange{}, &ParseError{
					Kind:    ErrBadMode,
					Message: fmt.Sprintf("invalid deleted file mode: %s (only 100644 and 100755 are supported)", mode),
				}
			}
			p.next()
			continue
		}

		// Mode changes
		if strings.HasPrefix(line, "old mode ") {
			oldMode := strings.TrimPrefix(line, "old mode ")
			p.next()

			if p.hasLine() && strings.HasPrefix(p.peekTrimmed(), "new mode ") {
				newModeLine := p.peekTrimmed()
				newMode := strings.TrimPrefix(newModeLine, "new mode ")
				p.next()

				if (oldMode == "100644" && newMode == "100755") || (oldMode == "100755" && newMode == "100644") {
					if newMode == "100755" {
						change.NewMode = 0o755
					} else {
						change.NewMode = 0o644
					}
				} else {
					return FileChange{}, &ParseError{
						Kind:    ErrBadMode,
						Message: fmt.Sprintf("unsupported mode change: %s -> %s", oldMode, newMode),
					}
				}
			}
			continue
		}

		// Rename
		if strings.HasPrefix(line, "rename from ") {
			change.Op = OpRename
			change.OldPath = stripPrefix(strings.TrimPrefix(line, "rename from "))
			change.originalAPath = "a/" + change.OldPath
			p.next()
			continue
		}

		if strings.HasPrefix(line, "rename to ") {
			change.Path = stripPrefix(strings.TrimPrefix(line, "rename to "))
			change.originalBPath = "b/" + change.Path
			p.next()
			continue
		}

		// Binary marker
		if strings.HasPrefix(line, "Binary files ") && strings.Contains(line, " differ") {
			change.IsBinary = true
			p.next()
			return change, nil
		}

		if strings.HasPrefix(line, "GIT binary patch") {
			change.IsBinary = true
			p.next()
			// Skip binary patch data
			for p.hasLine() {
				l := p.peekTrimmed()
				if strings.HasPrefix(l, "diff --git ") || l == "" {
					break
				}
				p.next()
			}
			return change, nil
		}

		// --- and +++ lines: detect file operation and preserve paths
		if strings.HasPrefix(line, "--- ") {
			// Extract path for round-trip
			minusPath := strings.TrimPrefix(line, "--- ")
			change.originalAPath = minusPath
			// For header-less diffs: derive Op from --- /dev/null, and set Path from --- line
			if !hasHeader && minusPath == "/dev/null" && change.Op == OpModify {
				change.Op = OpCreate
			}
			// For header-less diffs: set Path from --- if not /dev/null (for deletes/modifies)
			if !hasHeader && change.Path == "" && minusPath != "/dev/null" {
				change.Path = stripPrefix(minusPath)
			}
			seenMinusLine = true
			p.next()
			continue
		}

		if strings.HasPrefix(line, "+++ ") {
			// Extract path for round-trip
			plusPath := strings.TrimPrefix(line, "+++ ")
			change.originalBPath = plusPath
			// For header-less diffs: derive Op from +++ /dev/null
			if !hasHeader && plusPath == "/dev/null" && change.Op == OpModify {
				change.Op = OpDelete
			}
			// For header-less diffs: set Path if not already set (for creates/modifies)
			if !hasHeader && change.Path == "" && plusPath != "/dev/null" {
				change.Path = stripPrefix(plusPath)
			}
			seenPlusLine = true
			p.next()
			continue
		}

		// Hunks
		if strings.HasPrefix(line, "@@ ") {
			hunk, err := p.parseHunk(change.originalAPath)
			if err != nil {
				return FileChange{}, err
			}
			change.Hunks = append(change.Hunks, hunk)
			continue
		}

		// Unknown line, might be something we should skip
		p.next()
	}

	return change, nil
}

func (p *parser) parseHunk(filePath string) (Hunk, error) {
	header := p.peekTrimmed()
	p.next()

	// Parse @@ -oldStart,oldCount +newStart,newCount @@
	start := strings.Index(header, "@@")
	if start == -1 {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s: malformed hunk header (no opening @@)", filePath),
		}
	}

	end := strings.Index(header[start+2:], "@@")
	if end == -1 {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s: malformed hunk header (no closing @@)", filePath),
		}
	}

	rangeStr := header[start+2 : start+2+end]
	parts := strings.Fields(rangeStr)
	if len(parts) < 2 {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: malformed hunk ranges", filePath, header),
		}
	}

	// Parse old range: -start,count
	oldRange := parts[0]
	if !strings.HasPrefix(oldRange, "-") {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: malformed old range", filePath, header),
		}
	}
	oldRange = strings.TrimPrefix(oldRange, "-")

	oldParts := strings.Split(oldRange, ",")
	oldStart, err := strconv.Atoi(oldParts[0])
	if err != nil {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: invalid old start: %s", filePath, header, oldParts[0]),
		}
	}

	oldLines := 1
	if len(oldParts) == 2 {
		oldLines, err = strconv.Atoi(oldParts[1])
		if err != nil {
			return Hunk{}, &ParseError{
				Kind:    ErrHunkCountMismatch,
				Message: fmt.Sprintf("in %s %s: invalid old count: %s", filePath, header, oldParts[1]),
			}
		}
	}

	// Parse new range: +start,count
	newRange := parts[1]
	if !strings.HasPrefix(newRange, "+") {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: malformed new range", filePath, header),
		}
	}
	newRange = strings.TrimPrefix(newRange, "+")

	newParts := strings.Split(newRange, ",")
	newStart, err := strconv.Atoi(newParts[0])
	if err != nil {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: invalid new start: %s", filePath, header, newParts[0]),
		}
	}

	newLines := 1
	if len(newParts) == 2 {
		newLines, err = strconv.Atoi(newParts[1])
		if err != nil {
			return Hunk{}, &ParseError{
				Kind:    ErrHunkCountMismatch,
				Message: fmt.Sprintf("in %s %s: invalid new count: %s", filePath, header, newParts[1]),
			}
		}
	}

	hunk := Hunk{
		OldStart: oldStart,
		OldLines: oldLines,
		NewStart: newStart,
		NewLines: newLines,
	}

	// Parse hunk lines - preserve CRLF in content
	var oldCount, newCount int

hunkLoop:
	for p.hasLine() {
		rawLine := p.peek()
		trimmedLine := strings.TrimSuffix(rawLine, "\r")

		// Stop at next hunk or file
		if strings.HasPrefix(trimmedLine, "@@") || strings.HasPrefix(trimmedLine, "diff --git ") {
			break
		}

		if trimmedLine == "" {
			break
		}

		// Handle "\ No newline at end of file" — special marker that doesn't count towards line arithmetic.
		// This handler must run *before* the completion check below: the marker does not increment either
		// oldCount or newCount, so if completion ran first, a hunk would terminate when counts matched,
		// leaving the marker to fall through the unknown-line skip at line 354. That silently drops it,
		// violating Task 2 rule 3 (do not silently rewrite files). Running the handler first preserves
		// the marker and lets the completion check run after.
		if strings.HasPrefix(trimmedLine, `\ No newline at end of file`) {
			hunk.Lines = append(hunk.Lines, HunkLine{Prefix: '\\', Content: `No newline at end of file`})
			p.next()
			continue
		}

		// Stop if hunk is complete (counts satisfied) — but only for regular content lines
		if oldCount == oldLines && newCount == newLines {
			break
		}

		prefix := trimmedLine[0]

		// Regular lines - preserve content including any CR
		content := ""
		if len(trimmedLine) > 1 {
			content = trimmedLine[1:]
		}

		// If original rawLine had \r at the end, add it back to content
		if len(rawLine) > len(trimmedLine) {
			content = content + "\r"
		}

		switch prefix {
		case ' ':
			oldCount++
			newCount++
			hunk.Lines = append(hunk.Lines, HunkLine{Prefix: ' ', Content: content})
			p.next()
		case '+':
			newCount++
			hunk.Lines = append(hunk.Lines, HunkLine{Prefix: '+', Content: content})
			p.next()
		case '-':
			oldCount++
			hunk.Lines = append(hunk.Lines, HunkLine{Prefix: '-', Content: content})
			p.next()
		case '\\':
			// Backslash lines are already handled above
			p.next()
		default:
			// Unknown prefix, end of hunk
			break hunkLoop
		}
	}

	// Validate hunk arithmetic
	if oldCount != oldLines {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: old line count mismatch: declared %d, found %d", filePath, header, oldLines, oldCount),
		}
	}
	if newCount != newLines {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: new line count mismatch: declared %d, found %d", filePath, header, newLines, newCount),
		}
	}

	return hunk, nil
}

// ── Helper functions ────────────────────────────────────────────────────────

// extractPaths pulls the a/ and b/ paths from a "diff --git a/... b/..." line
func extractPaths(line string) (string, string) {
	// Format: diff --git a/path b/path
	if !strings.HasPrefix(line, "diff --git ") {
		return "", ""
	}

	rest := strings.TrimPrefix(line, "diff --git ")

	// Find the split between a/ and b/
	// Look for " b/" - the last occurrence
	lastSpace := strings.LastIndex(rest, " b/")
	if lastSpace == -1 {
		return "", ""
	}

	aPath := rest[:lastSpace]
	bPath := rest[lastSpace+1:]

	// Remove quotes if present (for paths with spaces)
	aPath = strings.Trim(aPath, `"`)
	bPath = strings.Trim(bPath, `"`)

	// Strip a/ and b/ prefixes
	aPath = strings.TrimPrefix(aPath, "a/")
	bPath = strings.TrimPrefix(bPath, "b/")

	return aPath, bPath
}

// stripPrefix removes a/ or b/ prefix from a path
func stripPrefix(path string) string {
	return strings.TrimPrefix(strings.TrimPrefix(path, "a/"), "b/")
}

// ── Render ──────────────────────────────────────────────────────────────────

// Render converts file changes back to unified diff format.
// For round-trip testing, it renders changes back to text form.
func Render(changes []FileChange) []byte {
	var buf bytes.Buffer

	for _, change := range changes {
		// For --- and +++ lines, we use the original paths (which might include /dev/null)
		minusPath := change.originalAPath
		plusPath := change.originalBPath

		// Only render diff --git header if the original had one
		if change.HasHeader {
			// diff --git line always uses a/path and b/path, never /dev/null
			// For creates: a/newfile b/newfile
			// For deletes: a/file b/file
			// For renames: a/oldname b/newname

			diffAPath := "a/" + change.Path
			diffBPath := "b/" + change.Path

			if change.Op == OpRename {
				diffAPath = "a/" + change.OldPath
				diffBPath = "b/" + change.Path
			}

			fmt.Fprintf(&buf, "diff --git %s %s\n", diffAPath, diffBPath)

			// Mode lines
			if change.NewMode != 0 {
				if change.NewMode == 0o755 {
					buf.WriteString("old mode 100644\n")
					buf.WriteString("new mode 100755\n")
				} else {
					buf.WriteString("old mode 100755\n")
					buf.WriteString("new mode 100644\n")
				}
			}

			// Rename lines
			if change.Op == OpRename {
				fmt.Fprintf(&buf, "rename from %s\n", change.OldPath)
				fmt.Fprintf(&buf, "rename to %s\n", change.Path)
			}

			// new file / deleted file
			if change.Op == OpCreate {
				// Render the preserved create mode
				if change.CreateMode == 0o755 {
					buf.WriteString("new file mode 100755\n")
				} else {
					buf.WriteString("new file mode 100644\n")
				}
			}
			if change.Op == OpDelete {
				buf.WriteString("deleted file mode 100644\n")
			}
		}

		// --- and +++ lines (for hunks, creates, deletes, and binary files with --- markers)
		if len(change.Hunks) > 0 || change.Op == OpCreate || change.Op == OpDelete ||
			(change.IsBinary && change.originalAPath != "") {
			// Use the preserved original paths
			displayMinusPath := minusPath
			displayPlusPath := plusPath

			if displayMinusPath == "" || displayMinusPath == "a/" {
				if change.Op == OpCreate {
					displayMinusPath = "/dev/null"
				} else {
					displayMinusPath = "a/" + change.Path
				}
			}

			if displayPlusPath == "" || displayPlusPath == "b/" {
				if change.Op == OpDelete {
					displayPlusPath = "/dev/null"
				} else {
					displayPlusPath = "b/" + change.Path
				}
			}

			fmt.Fprintf(&buf, "--- %s\n", displayMinusPath)
			fmt.Fprintf(&buf, "+++ %s\n", displayPlusPath)
		}

		// Binary marker
		if change.IsBinary {
			// Construct diff paths for binary marker
			diffAPath := "a/" + change.Path
			diffBPath := "b/" + change.Path
			if change.Op == OpRename {
				diffAPath = "a/" + change.OldPath
				diffBPath = "b/" + change.Path
			}
			fmt.Fprintf(&buf, "Binary files %s and %s differ\n", diffAPath, diffBPath)
			continue
		}

		// Hunks
		for _, hunk := range change.Hunks {
			fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)
			for _, line := range hunk.Lines {
				if line.Prefix == '\\' {
					// Special handling for backslash prefix
					buf.WriteString(`\ `)
					buf.WriteString(line.Content)
				} else {
					buf.WriteByte(line.Prefix)
					buf.WriteString(line.Content)
				}
				buf.WriteByte('\n')
			}
		}
	}

	return buf.Bytes()
}
