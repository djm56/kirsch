package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/djm56/kirsch/internal/workspace"
)

// MaxDiffBytes caps git_diff output. plan §3.
const MaxDiffBytes = 200 << 10

// GitStatus reports the working tree state.
type GitStatus struct{ WS *workspace.Workspace }

// Name implements Tool.
func (t *GitStatus) Name() string { return "git_status" }

// Description implements Tool.
func (t *GitStatus) Description() string {
	return "Show the working tree status (git status --porcelain)."
}

// Schema implements Tool.
func (t *GitStatus) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
}

// Invoke implements Tool.
func (t *GitStatus) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in struct{}
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}
	if !t.WS.IsGit {
		// Not a crash and not a violation: the workspace is simply not a
		// repository, which is a supported way to run.
		return Fail(KindToolInputInvalid,
			"this workspace is not a Git repository, so there is no status to report")
	}
	out, err := RunGit(ctx, t.WS.Root, "status", "--porcelain")
	if err != nil {
		return gitFailure(ctx, "git status", err, out)
	}
	n := 0
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	summary := fmt.Sprintf("%d changed", n)
	if n == 0 {
		summary = "clean"
		out = "working tree clean\n"
	}
	content, truncated := Truncate(out, MaxDiffBytes)
	return OKResult(content, summary, truncated)
}

// GitDiff shows changes in the working tree or the index.
type GitDiff struct{ WS *workspace.Workspace }

type gitDiffInput struct {
	Staged bool   `json:"staged,omitempty"`
	Path   string `json:"path,omitempty"`
}

// Name implements Tool.
func (t *GitDiff) Name() string { return "git_diff" }

// Description implements Tool.
func (t *GitDiff) Description() string {
	return "Show a unified diff of uncommitted changes, optionally staged only " +
		"or limited to one path."
}

// Schema implements Tool.
func (t *GitDiff) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "staged": {"type": "boolean", "description": "Diff the index against HEAD instead of the working tree."},
    "path":   {"type": "string",  "description": "Workspace-relative path to limit the diff to."}
  },
  "additionalProperties": false
}`)
}

// Invoke implements Tool.
func (t *GitDiff) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in gitDiffInput
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}
	if !t.WS.IsGit {
		return Fail(KindToolInputInvalid,
			"this workspace is not a Git repository, so there is nothing to diff")
	}

	if in.Path != "" {
		// Even a path handed straight to git crosses Resolve first: "git will
		// handle it" is exactly the reasoning that lets ../../etc through.
		if _, res, bad := resolve(t.WS, in.Path); bad {
			return res
		}
	}

	out, err := RunGit(ctx, t.WS.Root, DiffArgs(in.Staged, in.Path)...)
	if err != nil {
		return gitFailure(ctx, "git diff", err, out)
	}
	files := strings.Count(out, "\ndiff --git ")
	if strings.HasPrefix(out, "diff --git ") {
		files++
	}
	content, truncated := Truncate(out, MaxDiffBytes)
	summary := fmt.Sprintf("%d files changed", files)
	if in.Staged {
		summary = "staged · " + summary
	}
	if files == 0 {
		summary = "no changes"
		content = "no changes\n"
	}
	return OKResult(content, summary, truncated)
}

// DiffArgs builds the argument list for a git diff, shared by the git_diff tool
// and the TUI's /diff so the two cannot drift. --no-ext-diff and --no-textconv
// stop a diff driver named in repository config from running a command. A
// non-empty path is appended after a `--` terminator and must already have
// crossed workspace.Resolve.
func DiffArgs(staged bool, path string) []string {
	args := []string{"diff", "--no-color", "--no-ext-diff", "--no-textconv"}
	if staged {
		args = append(args, "--staged")
	}
	if path != "" {
		args = append(args, "--", path)
	}
	return args
}

// RunGit executes a git command in the given directory and returns its output.
// The executable is a literal, never a variable, so there is no command
// injection surface. Arguments must be pre-validated; git_diff's path argument
// crosses workspace.Resolve before calling here and is passed after a `--`
// terminator so it cannot be read as an option. gosec G204.
func RunGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...) // #nosec G204 -- literal binary, resolved args
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil && stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), err
}

func gitFailure(ctx context.Context, what string, err error, out string) Result {
	if ctx.Err() != nil {
		return Fail(KindCancelled, "cancelled during %s", what)
	}
	return Fail(KindInternal, "%s failed: %v", what, err)
}
