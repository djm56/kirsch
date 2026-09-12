package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/djm56/kirsch/internal/workspace"
)

// MaxListEntries caps list_files output. plan §3.
const MaxListEntries = 1000

// ListFiles walks the workspace, honouring ignore rules.
type ListFiles struct{ WS *workspace.Workspace }

type listFilesInput struct {
	Path          string `json:"path,omitempty"`
	MaxDepth      *int   `json:"max_depth,omitempty"`
	IncludeHidden bool   `json:"include_hidden,omitempty"`
}

// Name implements Tool.
func (t *ListFiles) Name() string { return "list_files" }

// Description implements Tool.
func (t *ListFiles) Description() string {
	return "List files and directories in the workspace, respecting .gitignore " +
		"and Kirsch's built-in ignore list."
}

// Schema implements Tool.
func (t *ListFiles) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path":           {"type": "string",  "description": "Workspace-relative directory. Defaults to the root."},
    "max_depth":      {"type": "integer", "description": "Levels below path to descend. Defaults to 2; -1 is unlimited."},
    "include_hidden": {"type": "boolean", "description": "Include dotfiles. Denylisted paths stay hidden regardless."}
  },
  "additionalProperties": false
}`)
}

// Invoke implements Tool.
func (t *ListFiles) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in listFilesInput
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}
	if in.Path == "" {
		in.Path = "."
	}
	depth := 2
	if in.MaxDepth != nil {
		depth = *in.MaxDepth
	}
	if _, res, bad := resolve(t.WS, in.Path); bad {
		return res
	}

	var b strings.Builder
	files, dirs := 0, 0
	err := t.WS.Walk(in.Path, workspace.WalkOptions{
		MaxDepth:      depth,
		IncludeHidden: in.IncludeHidden,
		Limit:         MaxListEntries,
	}, func(e workspace.Entry) bool {
		if ctx.Err() != nil {
			return false
		}
		if e.IsDir {
			dirs++
			b.WriteString(e.Rel + "/\n")
		} else {
			files++
			fmt.Fprintf(&b, "%s\t%d\n", e.Rel, e.Size)
		}
		return true
	})
	if ctx.Err() != nil {
		return Fail(KindCancelled, "cancelled while listing %s", in.Path)
	}
	truncated := workspace.IsLimitReached(err)
	if err != nil && !truncated {
		if workspace.IsViolation(err) {
			return Fail(KindWorkspaceViolation, "%v", err)
		}
		return Fail(KindInternal, "walk %s: %v", in.Path, err)
	}

	summary := fmt.Sprintf("%s · %d files", in.Path, files)
	if dirs > 0 {
		summary += fmt.Sprintf(", %d dirs", dirs)
	}
	if truncated {
		summary += fmt.Sprintf(" (capped at %d)", MaxListEntries)
	}
	return OKResult(b.String(), summary, truncated)
}
