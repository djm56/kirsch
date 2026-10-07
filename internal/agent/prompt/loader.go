// Package prompt assembles the Kirsch system prompt from an embedded template
// and caller-supplied environment and project context.
package prompt

import (
	"fmt"
	"io"
	"os"
)

// maxProjectContextBytes is the hard upper bound for project-context loading.
// A caller-supplied cap above this value is clamped to this ceiling.
const maxProjectContextBytes = 32768

// Engine is the narrow interface the project-context loader needs from the
// workspace containment layer. It is declared here in the agent tree so that
// internal/agent and its subpackages do not import any implementation package.
// The real *workspace.Workspace satisfies this interface directly; the app
// layer wires it with a trivial adapter.
type Engine interface {
	// Resolve turns a workspace-relative path into an absolute filesystem path,
	// or returns an error if the path is absolute, traverses above the root,
	// resolves outside the root, or matches the denylist.
	Resolve(rel string) (string, error)
}

// LoadProjectContext selects the first usable file from candidates, reads it
// with a bounded reader, and returns the raw content together with metadata the
// caller needs for reporting and for assembling the system prompt.
//
// Selection is first-existing-wins. Non-existent candidates are skipped
// silently. Candidates refused by the engine (absolute paths, paths containing
// "..", paths that resolve outside the workspace, or denylisted paths such as
// ".env" and ".kirsch") are skipped with a warning that names the candidate.
// Candidates that are not regular files (directories, FIFOs, devices, etc.) are
// skipped with a warning and are never opened, so a FIFO cannot block the
// loader.
//
// The effective byte cap is min(maxBytes, 32768). The read is bounded by
// construction: the file is never trusted to be the size Stat reports. Content
// that exceeds the cap is truncated at the last complete line boundary within
// the cap and a visible truncation marker is appended.
//
// LoadProjectContext is a pure function of its inputs. It holds no cache and
// no session state; callers enforce any once-per-session or re-read policies.
//
// Returns the content (raw, possibly truncated), the chosen candidate name, the
// file's byte size, any warnings, and an error only for unexpected filesystem
// failures.
func LoadProjectContext(candidates []string, maxBytes int, engine Engine) (content string, chosen string, size int, warnings []string, err error) {
	cap := maxBytes
	if cap < 0 {
		cap = 0
	}
	if cap > maxProjectContextBytes {
		cap = maxProjectContextBytes
	}

	for _, candidate := range candidates {
		resolved, resolveErr := engine.Resolve(candidate)
		if resolveErr != nil {
			warnings = append(warnings, fmt.Sprintf("project context candidate %q refused: %s", candidate, resolveErr))
			continue
		}

		st, statErr := os.Stat(resolved)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return "", "", 0, warnings, fmt.Errorf("stat project context candidate %q: %w", candidate, statErr)
		}
		if !st.Mode().IsRegular() {
			warnings = append(warnings, fmt.Sprintf("project context candidate %q is not a regular file", candidate))
			continue
		}

		// From here on the candidate is the chosen file. The size is reported
		// from Stat so /status can show the on-disk size even when the content
		// is truncated.
		chosen = candidate
		size = int(st.Size())

		readContent, readErr := readBounded(resolved, cap)
		if readErr != nil {
			return "", "", 0, warnings, fmt.Errorf("read project context candidate %q: %w", candidate, readErr)
		}
		return readContent, chosen, size, warnings, nil
	}

	return "", "", 0, warnings, nil
}

// readBounded reads at most cap bytes from path and, if the file exceeds the
// cap, truncates at the last complete line boundary within the cap and appends
// a visible marker. The cap+1 read proves truncation was detected without
// trusting the file's Stat size.
func readBounded(path string, cap int) (_ string, err error) {
	f, err := os.Open(path) // #nosec G304 -- path is Engine.Resolve output: LoadProjectContext, the only caller, passes the result of engine.Resolve, which rejects absolute paths, upward traversal, out-of-root resolution, and denylisted paths
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// LimitReader caps the read at cap+1 bytes. If the file is larger than the
	// cap we get cap+1 bytes and no EOF; if it is cap bytes or smaller we get
	// at most cap bytes and EOF.
	limited := io.LimitReader(f, int64(cap)+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}

	if len(buf) <= cap {
		return string(buf), nil
	}

	// The file exceeded the cap. Truncate at the last complete line boundary
	// within the capped region and append a visible marker.
	buf = buf[:cap]
	cut := lastNewlineIndex(buf)
	if cut >= 0 {
		buf = buf[:cut+1]
	} else {
		buf = buf[:0]
	}

	marker := fmt.Sprintf("[project context truncated after %d bytes]\n", cap)
	return string(buf) + marker, nil
}

// lastNewlineIndex returns the index of the last '\n' in b, or -1 if none.
func lastNewlineIndex(b []byte) int {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			return i
		}
	}
	return -1
}
