package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/djm56/kirsch/internal/workspace"
)

// Read limits. plan §3.
const (
	MaxFileBytes    = 2 << 20 // 2MB
	DefaultMaxLines = 500
	BinarySniffLen  = 8 << 10
)

// ReadFile reads a text file from the workspace.
type ReadFile struct{ WS *workspace.Workspace }

type readFileInput struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

// Name implements Tool.
func (t *ReadFile) Name() string { return "read_file" }

// Description implements Tool.
func (t *ReadFile) Description() string {
	return "Read a UTF-8 text file from the workspace. Output is line-numbered. " +
		"Optionally limit to a 1-based inclusive line range."
}

// Schema implements Tool.
func (t *ReadFile) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path":       {"type": "string",  "description": "Workspace-relative path. Absolute paths are rejected."},
    "start_line": {"type": "integer", "description": "First line to return, 1-based inclusive."},
    "end_line":   {"type": "integer", "description": "Last line to return, 1-based inclusive."}
  },
  "required": ["path"],
  "additionalProperties": false
}`)
}

// Invoke implements Tool.
func (t *ReadFile) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in readFileInput
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}
	abs, res, bad := resolve(t.WS, in.Path)
	if bad {
		return res
	}

	st, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return Fail(KindFileNotFound, "%s does not exist", in.Path)
	}
	if err != nil {
		return Fail(KindInternal, "stat %s: %v", in.Path, err)
	}
	if st.IsDir() {
		return Fail(KindToolInputInvalid, "%s is a directory; use list_files", in.Path)
	}
	if st.Size() > MaxFileBytes {
		return Fail(KindFileTooLarge, "%s is %.1fMB; the limit is %dMB",
			in.Path, float64(st.Size())/(1<<20), MaxFileBytes>>20)
	}

	body, err := os.ReadFile(abs)
	if err != nil {
		return Fail(KindInternal, "read %s: %v", in.Path, err)
	}
	if isBinary(body) {
		return Fail(KindBinaryFile, "%s looks like a binary file (NUL byte in the first %dKB)",
			in.Path, BinarySniffLen>>10)
	}

	// Line splitting keeps \r attached to the line, so a CRLF file round-trips
	// unchanged rather than being silently normalised.
	lines := strings.Split(string(body), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1] // trailing newline is a terminator, not an empty line
	}

	start, end := 1, len(lines)
	if in.StartLine > 0 {
		start = in.StartLine
	}
	if in.EndLine > 0 {
		end = in.EndLine
	}
	if start > len(lines) {
		return Fail(KindToolInputInvalid, "start_line %d is past the end of %s (%d lines)",
			start, in.Path, len(lines))
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return Fail(KindToolInputInvalid, "start_line %d is after end_line %d", start, end)
	}

	truncated := false
	if in.EndLine == 0 && end-start+1 > DefaultMaxLines {
		end = start + DefaultMaxLines - 1
		truncated = true
	}

	// Line numbers are part of the contract: the model needs stable references
	// to cite, and from M2 to anchor a patch against.
	var b strings.Builder
	for i := start; i <= end; i++ {
		fmt.Fprintf(&b, "%d\t%s\n", i, lines[i-1])
	}
	content, cut := Truncate(b.String(), MaxFileBytes)
	truncated = truncated || cut

	summary := fmt.Sprintf("%s:%d-%d", in.Path, start, end)
	if truncated {
		summary += fmt.Sprintf(" (of %d lines)", len(lines))
	}
	return OKResult(content, summary, truncated)
}

// isBinary reports a NUL byte in the first BinarySniffLen bytes.
func isBinary(body []byte) bool {
	n := len(body)
	if n > BinarySniffLen {
		n = BinarySniffLen
	}
	for _, c := range body[:n] {
		if c == 0 {
			return true
		}
	}
	return false
}

// resolve maps a workspace refusal onto the right tool error kind.
//
// The distinction matters: a violation means "never, stop asking", while
// invalid input means "you wrote it wrong, try again" — and from M3 the model
// reads that difference and acts on it.
func resolve(ws *workspace.Workspace, rel string) (string, Result, bool) {
	abs, err := ws.Resolve(rel)
	if err == nil {
		return abs, Result{}, false
	}
	if workspace.IsInvalidPath(err) {
		return "", Fail(KindToolInputInvalid, "%v", err), true
	}
	return "", Fail(KindWorkspaceViolation, "%v", err), true
}
