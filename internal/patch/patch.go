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
// NewStart/NewLines describe the line range in the new file. OldLines and
// NewLines are counted from the body, not taken from the @@ header.
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
//
// Parse returns an error for a hunk that comes before any file header, for a
// hunk with no --- line or no +++ line earlier in its file change, and for
// input that yields no file change at all. A whitespace-only final fragment
// with no newline after it is dropped before parsing, so it cannot become a
// context line.
func Parse(data []byte) ([]FileChange, error) {
	lines := strings.Split(string(data), "\n")
	// The last element has no newline after it. Whitespace only there is a
	// fragment, not a body line, so it is dropped. A whitespace-only line that
	// ends in a newline is not the last element and is kept.
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "" {
		lines = lines[:n-1]
	}
	p := &parser{
		lines: lines,
	}
	changes, err := p.parse()
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, &ParseError{
			Kind:    ErrMissingHeader,
			Message: "no file changes found: expected --- a/… / +++ b/… or diff --git headers",
		}
	}
	return changes, nil
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
		} else if strings.HasPrefix(line, "@@ ") {
			// A hunk outside any file change belongs to no file, so it cannot be applied
			return nil, &ParseError{
				Kind:    ErrMissingHeader,
				Message: fmt.Sprintf("hunk at line %d has no file header (--- a/… / +++ b/…)", p.pos+1),
			}
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
			// A hunk needs both its --- and +++ lines before it. Without both, the
			// diff does not name the file the hunk applies to, so the hunk is refused.
			if !seenMinusLine || !seenPlusLine {
				return FileChange{}, &ParseError{
					Kind:    ErrMissingHeader,
					Message: fmt.Sprintf("hunk at line %d has no file header (--- a/… / +++ b/…)", p.pos+1),
				}
			}
			hunk, err := p.parseHunk(change.originalAPath, hasHeader)
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

// parseHunk reads one hunk, its @@ header and body. hasHeader reports whether
// the file change began with "diff --git ", which decides how a header-like
// pair inside the body is read.
func (p *parser) parseHunk(filePath string, hasHeader bool) (Hunk, error) {
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

	// Parse hunk lines - preserve CRLF in content.
	// The body is counted, not trusted: the header counts never decide where
	// the body ends. OldLines and NewLines are replaced with the counted values
	// below, as git apply --recount does.
	var oldCount, newCount int

hunkLoop:
	for p.hasLine() {
		rawLine := p.peek()
		trimmedLine := strings.TrimSuffix(rawLine, "\r")
		st := hunkState{
			hasHeader: hasHeader,
			declOld:   oldLines,
			declNew:   newLines,
			oldCount:  oldCount,
			newCount:  newCount,
		}

		// Stop at next hunk, next file, or end of input
		if p.hunkEndsAt(p.pos, st) {
			break
		}

		// A fully empty line is a blank context line when more body lines follow
		// before the hunk ends. Otherwise it is trailing and adds nothing.
		if trimmedLine == "" {
			if !p.bodyContinuesAt(p.pos, st) {
				break
			}
			// A bare blank line keeps the CR it had, as non-empty lines do
			content := ""
			if len(rawLine) > len(trimmedLine) {
				content = "\r"
			}
			oldCount++
			newCount++
			hunk.Lines = append(hunk.Lines, HunkLine{Prefix: ' ', Content: content})
			p.next()
			continue
		}

		// Handle "\ No newline at end of file" — special marker that doesn't count towards line arithmetic.
		// It is kept in the hunk and never counted, so it neither ends the hunk nor shifts the counts.
		if strings.HasPrefix(trimmedLine, `\ No newline at end of file`) {
			hunk.Lines = append(hunk.Lines, HunkLine{Prefix: '\\', Content: `No newline at end of file`})
			p.next()
			continue
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
			// A line with another prefix is not part of the patch text. If body
			// lines follow it before the hunk ends, ending the hunk here would drop
			// them and report the patch as applied, so it is an error. With no body
			// line after it, it is trailing text and ends the hunk.
			if p.bodyFollowsAt(p.pos+1, st) {
				return Hunk{}, &ParseError{
					Kind:    ErrHunkCountMismatch,
					Message: fmt.Sprintf("in %s %s: line %d is not a hunk line (it starts with neither ' ', '+', '-' nor '\\'): %q", filePath, header, p.pos+1, trimmedLine),
				}
			}
			break hunkLoop
		}
	}

	// A body with no ' ', '+' or '-' line has nothing to recount into a hunk
	if oldCount == 0 && newCount == 0 {
		return Hunk{}, &ParseError{
			Kind:    ErrHunkCountMismatch,
			Message: fmt.Sprintf("in %s %s: hunk has no body lines", filePath, header),
		}
	}

	// The declared counts are replaced by the counted values
	hunk.OldLines = oldCount
	hunk.NewLines = newCount

	return hunk, nil
}

// hunkState is what the body loop has counted when it asks whether a line
// ends the hunk. The counts are compared against the counts declared in the @@
// header in two places only: a format-patch signature, in any file change, and
// a header-like pair, in a file change that has no "diff --git " line.
type hunkState struct {
	hasHeader          bool // the file change began with "diff --git "
	declOld, declNew   int  // counts declared in the @@ header
	oldCount, newCount int  // body lines counted so far
}

// hunkEndsAt reports whether the line at index i ends the hunk body, given the
// counts in st. The hunk ends at the end of input, at a "@@" or "diff --git "
// line, at a format-patch signature, or at a file header: a "--- " line, a
// "+++ " line, then an "@@" line.
//
// A header-like pair inside a "diff --git " change is always body, because git
// starts the next file with "diff --git ". A header-like pair in a header-less
// change is body until the counted lines reach the declared counts. Only then
// does it end the hunk.
//
// A "-- " line is a signature only when the counted lines have reached the
// declared counts and exactly one non-empty line follows it before the end of
// input, that line looking like a git version (see signatureAt). A real
// signature follows a complete hunk. A "-- " line met before the counts are
// reached, or with no version-like line after it, is the removal of the file
// line "- ". A format-patch with a custom --signature is therefore not
// recognised, and its "-- " line is read as that removal.
//
// Known limitations, grouped by the mechanism.
//
// Counts short when the line is reached, because the header over-declares:
//   - In a header-less change, the next file's "--- " and "+++ " lines are
//     swallowed as body, so that hunk attaches to the earlier file and apply
//     fails with patch_conflict on it, not with a parse error.
//   - A format-patch signature after the hunk is read as the removal of "- ",
//     and apply fails the same way.
//
// Counts already reached when the line is reached, because the header
// under-declares. This is the silent case: a genuine removal of "- " as the
// final body line, followed by exactly one non-empty line that looks like a git
// version, is read as a signature and dropped without an error.
func (p *parser) hunkEndsAt(i int, st hunkState) bool {
	if i >= len(p.lines) {
		return true
	}
	line := strings.TrimSuffix(p.lines[i], "\r")
	if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff --git ") {
		return true
	}
	// A format-patch signature is a "-- " line that follows a hunk whose counted
	// lines have reached the declared counts, with exactly one non-empty line after
	// it, a version-like one.
	if line == "-- " && st.oldCount >= st.declOld && st.newCount >= st.declNew && p.signatureAt(i+1) {
		return true
	}
	if strings.HasPrefix(line, "--- ") && i+2 < len(p.lines) &&
		strings.HasPrefix(strings.TrimSuffix(p.lines[i+1], "\r"), "+++ ") &&
		strings.HasPrefix(strings.TrimSuffix(p.lines[i+2], "\r"), "@@") {
		if st.hasHeader {
			return false
		}
		return st.oldCount >= st.declOld && st.newCount >= st.declNew
	}
	return false
}

// signatureAt reports whether the lines from index i to the end of input hold
// a format-patch signature body: exactly one non-empty line, which looks like a
// git version (see isVersionLine), and any number of empty lines. Called just
// after a "-- " line. With no non-empty line after it, the "-- " line is not a
// signature.
func (p *parser) signatureAt(i int) bool {
	found := 0
	for ; i < len(p.lines); i++ {
		line := strings.TrimSuffix(p.lines[i], "\r")
		if line == "" {
			continue
		}
		found++
		if !isVersionLine(line) {
			return false
		}
	}
	return found == 1
}

// isVersionLine reports whether line starts with a version number: one or more
// digits, then one or more groups of a dot and one or more digits. It matches
// "2.39.1" and "2.39.1.windows.1", and anything after the number is ignored.
func isVersionLine(line string) bool {
	digits := func(i int) int {
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		return i
	}
	i := digits(0)
	if i == 0 {
		return false
	}
	groups := 0
	for i < len(line) && line[i] == '.' {
		end := digits(i + 1)
		if end == i+1 {
			break
		}
		groups++
		i = end
	}
	return groups > 0
}

// bodyFollowsAt reports whether a line starting with ' ', '+', '-' or '\\'
// occurs from index i up to where the hunk ends, given the counts in st. It is
// the test for text a stray line would cut off: lines after the hunk end are
// not scanned, and empty lines are skipped.
func (p *parser) bodyFollowsAt(i int, st hunkState) bool {
	for ; i < len(p.lines) && !p.hunkEndsAt(i, st); i++ {
		line := strings.TrimSuffix(p.lines[i], "\r")
		if line == "" {
			continue
		}
		switch line[0] {
		case ' ', '+', '-', '\\':
			return true
		}
	}
	return false
}

// bodyContinuesAt reports whether a hunk body line follows the position i
// before the hunk ends, given the counts in st. Empty lines from i onward are
// skipped while looking ahead, and each is counted as context, as the body loop
// counts it. The first non-empty line decides: a body line ' ', '+', '-' or '\'
// continues the hunk, and anything else ends it.
func (p *parser) bodyContinuesAt(i int, st hunkState) bool {
	for ; i < len(p.lines); i++ {
		line := strings.TrimSuffix(p.lines[i], "\r")
		if line == "" {
			st.oldCount++
			st.newCount++
			continue
		}
		if p.hunkEndsAt(i, st) {
			return false
		}
		switch line[0] {
		case ' ', '+', '-', '\\':
			return true
		}
		return false
	}
	return false
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
