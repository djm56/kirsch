package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/workspace"
)

func fixtureWS(t *testing.T, name string) *workspace.Workspace {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func registryFor(ws *workspace.Workspace) *Registry {
	r := NewRegistry()
	r.Register(&ReadFile{WS: ws})
	r.Register(&ListFiles{WS: ws})
	r.Register(&SearchCode{WS: ws})
	r.Register(&GitStatus{WS: ws})
	r.Register(&GitDiff{WS: ws})
	return r
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestEveryPathTakingToolRefusesEscapes is the table the acceptance checklist
// asks for: the denylist and containment must hold through every entry point,
// not just through read_file.
func TestEveryPathTakingToolRefusesEscapes(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	r := registryFor(ws)

	badPaths := []string{
		"../escape.txt",
		"../../etc/passwd",
		"sub/../../escape.txt",
		".env",
		".env.local",
		"key.pem",
		"id_rsa.key",
		".git/config",
		".kirsch/config.toml",
		"/etc/passwd", // absolute: invalid input rather than a violation
	}
	callers := map[string]func(string) json.RawMessage{
		"read_file":  func(p string) json.RawMessage { return mustJSON(t, map[string]any{"path": p}) },
		"list_files": func(p string) json.RawMessage { return mustJSON(t, map[string]any{"path": p}) },
		"git_diff":   func(p string) json.RawMessage { return mustJSON(t, map[string]any{"path": p}) },
	}

	for tool, build := range callers {
		for _, p := range badPaths {
			t.Run(tool+"/"+p, func(t *testing.T) {
				res := r.Invoke(context.Background(), tool, build(p))
				if res.OK {
					t.Fatalf("%s accepted %q", tool, p)
				}
				want := KindWorkspaceViolation
				if strings.HasPrefix(p, "/") {
					want = KindToolInputInvalid
				}
				if res.Error.Kind != want {
					t.Errorf("%s(%q): kind = %s, want %s (%s)",
						tool, p, res.Error.Kind, want, res.Error.Message)
				}
			})
		}
	}
}

func TestReadFile(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	r := registryFor(ws)
	call := func(in map[string]any) Result {
		return r.Invoke(context.Background(), "read_file", mustJSON(t, in))
	}

	t.Run("line numbered", func(t *testing.T) {
		res := call(map[string]any{"path": "main.go"})
		if !res.OK {
			t.Fatalf("read failed: %v", res.Error)
		}
		if !strings.HasPrefix(res.Content, "1\tpackage main") {
			t.Errorf("output is not line-numbered:\n%s", res.Content[:60])
		}
	})

	t.Run("line range", func(t *testing.T) {
		res := call(map[string]any{"path": "main.go", "start_line": 3, "end_line": 5})
		if !res.OK {
			t.Fatalf("%v", res.Error)
		}
		lines := strings.Split(strings.TrimRight(res.Content, "\n"), "\n")
		if len(lines) != 3 {
			t.Errorf("got %d lines, want 3:\n%s", len(lines), res.Content)
		}
		if !strings.HasPrefix(lines[0], "3\t") {
			t.Errorf("range does not start at line 3: %q", lines[0])
		}
	})

	t.Run("crlf preserved", func(t *testing.T) {
		res := call(map[string]any{"path": "crlf.txt"})
		if !res.OK {
			t.Fatalf("%v", res.Error)
		}
		if !strings.Contains(res.Content, "\r") {
			t.Error("CRLF was normalised away; the file must round-trip unchanged")
		}
	})

	t.Run("binary refused", func(t *testing.T) {
		res := call(map[string]any{"path": "binary.dat"})
		if res.OK || res.Error.Kind != KindBinaryFile {
			t.Errorf("kind = %v, want binary_file", res.Error)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		res := call(map[string]any{"path": "no-such-file.txt"})
		if res.OK || res.Error.Kind != KindFileNotFound {
			t.Errorf("kind = %v, want file_not_found", res.Error)
		}
	})

	t.Run("directory", func(t *testing.T) {
		res := call(map[string]any{"path": "pkg"})
		if res.OK || res.Error.Kind != KindToolInputInvalid {
			t.Errorf("kind = %v, want tool_input_invalid", res.Error)
		}
		if !strings.Contains(res.Error.Message, "list_files") {
			t.Errorf("message should point at the right tool: %q", res.Error.Message)
		}
	})

	t.Run("too large", func(t *testing.T) {
		// Generated at test time: the instruction set is explicit that a >2MB
		// file must not be committed.
		dir := t.TempDir()
		big := filepath.Join(dir, "big.txt")
		if err := os.WriteFile(big, make([]byte, 3<<20), 0o644); err != nil {
			t.Fatal(err)
		}
		tmpWS, err := workspace.Detect(dir)
		if err != nil {
			t.Fatal(err)
		}
		res := registryFor(tmpWS).Invoke(context.Background(), "read_file",
			mustJSON(t, map[string]any{"path": "big.txt"}))
		if res.OK || res.Error.Kind != KindFileTooLarge {
			t.Errorf("kind = %v, want file_too_large", res.Error)
		}
	})

	t.Run("no trailing newline", func(t *testing.T) {
		res := call(map[string]any{"path": "no-trailing-newline.txt"})
		if !res.OK {
			t.Fatalf("%v", res.Error)
		}
		if strings.Count(res.Content, "\n") != 1 {
			t.Errorf("a file without a trailing newline produced %d lines:\n%q",
				strings.Count(res.Content, "\n"), res.Content)
		}
	})
}

func TestListFiles(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	r := registryFor(ws)
	res := r.Invoke(context.Background(), "list_files",
		mustJSON(t, map[string]any{"path": ".", "max_depth": -1}))
	if !res.OK {
		t.Fatalf("%v", res.Error)
	}
	for _, forbidden := range []string{".env", "app.log", "ignored/", "scratch.txt"} {
		if strings.Contains(res.Content, forbidden) {
			t.Errorf("%s leaked into list_files output:\n%s", forbidden, res.Content)
		}
	}
	if !strings.Contains(res.Content, "main.go") {
		t.Errorf("main.go missing:\n%s", res.Content)
	}
	if !strings.Contains(res.DisplaySummary, "files") {
		t.Errorf("summary = %q, want a file count", res.DisplaySummary)
	}
}

func TestSearchCode(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	r := registryFor(ws)
	call := func(in map[string]any) Result {
		return r.Invoke(context.Background(), "search_code", mustJSON(t, in))
	}

	res := call(map[string]any{"query": "Divide"})
	if !res.OK {
		t.Fatalf("%v", res.Error)
	}
	if !strings.Contains(res.Content, "pkg/util/strings.go:") {
		t.Errorf("expected match missing:\n%s", res.Content)
	}
	for _, line := range strings.Split(strings.TrimSpace(res.Content), "\n") {
		if line == "no matches" {
			continue
		}
		if !regexpPathLine.MatchString(line) {
			t.Errorf("result line is not path:line: text — %q", line)
		}
	}

	if res := call(map[string]any{"query": "["}); !res.OK {
		t.Errorf("literal search of '[' should be treated as text, not a regex: %v", res.Error)
	}
	if res := call(map[string]any{"query": "[", "regex": true}); res.OK {
		t.Error("an invalid regex was accepted")
	} else if res.Error.Kind != KindToolInputInvalid {
		t.Errorf("kind = %s, want tool_input_invalid", res.Error.Kind)
	}
	if res := call(map[string]any{"query": ""}); res.OK {
		t.Error("empty query accepted")
	}
	// Search must never surface an ignored or denied file.
	if res := call(map[string]any{"query": "hunter2"}); res.OK &&
		strings.Contains(res.Content, ".env") {
		t.Errorf(".env content surfaced through search:\n%s", res.Content)
	}
}

var regexpPathLine = regexp.MustCompile(`^[^:]+:\d+: `)

// TestSearchBackendParity is the differential test milestone-1 asks for while
// both implementations are fresh. A divergence found in M4 is a debugging
// afternoon; found here it is a diff.
func TestSearchBackendParity(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not on PATH; parity cannot be checked on this machine")
	}
	ws := fixtureWS(t, "repo-small")
	rgTool := &SearchCode{WS: ws}
	goTool := &SearchCode{WS: ws, ForcePureGo: true}
	if rgTool.Backend() != "ripgrep" || goTool.Backend() != "pure-go" {
		t.Fatalf("backends not distinct: %s vs %s", rgTool.Backend(), goTool.Backend())
	}

	queries := []map[string]any{
		{"query": "Divide"},
		{"query": "func"},
		{"query": "package"},
		{"query": "hunter2"},  // only in .env, which is denied
		{"query": "scratch"},  // only in a gitignored file
		{"query": "left-pad"}, // only under node_modules
		{"query": "Divide", "glob": "*.go"},
		{"query": "func.*string", "regex": true},
		{"query": "日本語"},
	}
	for _, q := range queries {
		name := fmt.Sprint(q)
		t.Run(name, func(t *testing.T) {
			raw := mustJSON(t, q)
			a := rgTool.Invoke(context.Background(), raw)
			b := goTool.Invoke(context.Background(), raw)
			if a.OK != b.OK {
				t.Fatalf("OK differs: rg=%v go=%v", a.OK, b.OK)
			}
			if a.Content != b.Content {
				t.Errorf("backends disagree for %s\n--- ripgrep ---\n%s\n--- pure-go ---\n%s",
					name, a.Content, b.Content)
			}
			if a.DisplaySummary != b.DisplaySummary {
				t.Errorf("summaries differ: %q vs %q", a.DisplaySummary, b.DisplaySummary)
			}
		})
	}
}

// TestGitToolsOnNonRepo needs a directory that is genuinely outside any
// repository. The testdata fixtures cannot serve: they live inside Kirsch's own
// checkout, so `git -C testdata/repo-small rev-parse --git-dir` correctly finds
// the parent repo. That is right behaviour — a subdirectory of a repo is in a
// repo — and it means only a TempDir is really non-Git.
func TestGitToolsOnNonRepo(t *testing.T) {
	ws, err := workspace.Detect(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ws.IsGit {
		t.Skip("TempDir is inside a repository on this machine")
	}
	r := registryFor(ws)
	for _, name := range []string{"git_status", "git_diff"} {
		res := r.Invoke(context.Background(), name, json.RawMessage(`{}`))
		if res.OK {
			t.Errorf("%s succeeded outside a repository", name)
			continue
		}
		if res.Error == nil || !strings.Contains(res.Error.Message, "not a Git repository") {
			t.Errorf("%s: unhelpful message %v", name, res.Error)
		}
	}
}

func TestGitToolsOnRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@e.com"}, {"config", "user.name", "T"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "init"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := workspace.Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := registryFor(ws)

	st := r.Invoke(context.Background(), "git_status", json.RawMessage(`{}`))
	if !st.OK {
		t.Fatalf("git_status: %v", st.Error)
	}
	if !strings.Contains(st.Content, "a.txt") {
		t.Errorf("status missing the modified file:\n%s", st.Content)
	}

	df := r.Invoke(context.Background(), "git_diff", json.RawMessage(`{}`))
	if !df.OK {
		t.Fatalf("git_diff: %v", df.Error)
	}
	if !strings.Contains(df.Content, "+two") {
		t.Errorf("diff missing the added line:\n%s", df.Content)
	}
	if !strings.Contains(df.DisplaySummary, "1 files changed") {
		t.Errorf("summary = %q", df.DisplaySummary)
	}
}

func TestSchemasAreValidJSON(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	for _, tool := range registryFor(ws).List() {
		var v map[string]any
		if err := json.Unmarshal(tool.Schema(), &v); err != nil {
			t.Errorf("%s: schema is not valid JSON: %v", tool.Name(), err)
			continue
		}
		if v["type"] != "object" {
			t.Errorf("%s: schema type = %v, want object", tool.Name(), v["type"])
		}
		if _, ok := v["additionalProperties"]; !ok {
			t.Errorf("%s: schema does not set additionalProperties", tool.Name())
		}
		if tool.Description() == "" {
			t.Errorf("%s: no description", tool.Name())
		}
	}
}

func TestRegistryListsAllFive(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	got := registryFor(ws).Names()
	want := []string{"git_diff", "git_status", "list_files", "read_file", "search_code"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tools = %v, want %v", got, want)
	}
}
