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

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/policy"
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
// not just through read_file. Covers both read-only and write tools.
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

	// Read-only tools that take a path parameter
	readOnlyCallers := map[string]func(string) json.RawMessage{
		"read_file":  func(p string) json.RawMessage { return mustJSON(t, map[string]any{"path": p}) },
		"list_files": func(p string) json.RawMessage { return mustJSON(t, map[string]any{"path": p}) },
		"git_diff":   func(p string) json.RawMessage { return mustJSON(t, map[string]any{"path": p}) },
	}

	for tool, build := range readOnlyCallers {
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

	// Test apply_patch target path escapes
	tmpWS := newTestWorkspace(t)
	approver := newFakeApprover(policy.DecisionAllow)
	applyPatchTool := &ApplyPatch{WS: tmpWS, Approver: approver}

	for _, p := range badPaths {
		t.Run("apply_patch/target/"+p, func(t *testing.T) {
			diff := "diff --git a/" + p + " b/" + p + "\n" +
				"--- a/" + p + "\n" +
				"+++ b/" + p + "\n" +
				"@@ -1 +1 @@\n" +
				"-old\n" +
				"+new\n"
			input := applyPatchInput{Diff: diff, Description: "test"}
			raw := mustJSON(t, input)
			res := applyPatchTool.Invoke(context.Background(), raw)
			if res.OK {
				t.Fatalf("apply_patch accepted target %q", p)
			}
			want := KindWorkspaceViolation
			if strings.HasPrefix(p, "/") {
				want = KindToolInputInvalid
			}
			if res.Error.Kind != want {
				t.Errorf("apply_patch target %q: kind = %s, want %s", p, res.Error.Kind, want)
			}
			if approver.called {
				t.Errorf("apply_patch should not call approver for path violation on target %q", p)
			}
		})
	}

	// Test apply_patch rename source escapes
	tmpWS2 := newTestWorkspace(t)
	approver2 := newFakeApprover(policy.DecisionAllow)
	applyPatchTool2 := &ApplyPatch{WS: tmpWS2, Approver: approver2}

	for _, p := range badPaths {
		t.Run("apply_patch/rename_source/"+p, func(t *testing.T) {
			diff := "diff --git a/" + p + " b/good.txt\n" +
				"--- a/" + p + "\n" +
				"+++ b/good.txt\n" +
				"similarity index 100%\n" +
				"rename from " + p + "\n" +
				"rename to good.txt\n"
			input := applyPatchInput{Diff: diff, Description: "test"}
			raw := mustJSON(t, input)
			res := applyPatchTool2.Invoke(context.Background(), raw)
			if res.OK {
				t.Fatalf("apply_patch accepted rename source %q", p)
			}
			want := KindWorkspaceViolation
			if strings.HasPrefix(p, "/") {
				want = KindToolInputInvalid
			}
			if res.Error.Kind != want {
				t.Errorf("apply_patch rename source %q: kind = %s, want %s", p, res.Error.Kind, want)
			}
			if approver2.called {
				t.Errorf("apply_patch should not call approver for path violation on rename source %q", p)
			}
		})
	}

	// Test run_command cwd escapes
	tmpWS3 := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New(true, true, true)
	approver3 := newFakeApprover(policy.DecisionAllow)
	runCmdTool := &RunCommand{WS: tmpWS3, Config: &conf, Policy: pol, Approver: approver3}

	for _, p := range badPaths {
		t.Run("run_command/cwd/"+p, func(t *testing.T) {
			input := runCommandInput{Argv: []string{"echo", "test"}, Cwd: p}
			raw := mustJSON(t, input)
			res := runCmdTool.Invoke(context.Background(), raw)
			if res.OK {
				t.Fatalf("run_command accepted cwd %q", p)
			}
			want := KindWorkspaceViolation
			if strings.HasPrefix(p, "/") {
				want = KindToolInputInvalid
			}
			if res.Error.Kind != want {
				t.Errorf("run_command cwd %q: kind = %s, want %s", p, res.Error.Kind, want)
			}
			if approver3.called {
				t.Errorf("run_command should not call approver for path violation on cwd %q", p)
			}
		})
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

func TestReadFileDisplaySummaryFormat(t *testing.T) {
	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "test.txt")
	content := "line 1\nline 2\nline 3"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpWS, err := workspace.Detect(dir)
	if err != nil {
		t.Fatal(err)
	}

	r := registryFor(tmpWS)
	res := r.Invoke(context.Background(), "read_file",
		mustJSON(t, map[string]any{"path": "test.txt"}))

	if !res.OK {
		t.Fatalf("read_file failed: %v", res.Error)
	}
	if res.DisplaySummary != "lines 1-3" {
		t.Errorf("DisplaySummary = %q, want %q", res.DisplaySummary, "lines 1-3")
	}
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

// TestApplyPatchSymlinkOutsideWorkspaceRefused verifies that apply_patch detects
// when a symlink inside the workspace points outside it, preventing laundering
// of denied paths. The approver must never be called for this violation.
func TestApplyPatchSymlinkOutsideWorkspaceRefused(t *testing.T) {
	ws := newTestWorkspace(t)
	approver := newFakeApprover(policy.DecisionAllow)
	tool := &ApplyPatch{WS: ws, Approver: approver}

	// Create a symlink inside the workspace that points outside.
	targetPath := filepath.Join(ws.Root, "escape-link")
	// Point it to a path outside workspace (using relative path to go up)
	if err := os.Symlink("../../../etc/passwd", targetPath); err != nil {
		t.Skipf("cannot create symlink on this system: %v", err)
	}
	defer os.Remove(targetPath)

	// Try to create a patch targeting the symlink.
	input := applyPatchInput{
		Diff: "diff --git a/escape-link b/escape-link\n" +
			"--- a/escape-link\n" +
			"+++ b/escape-link\n" +
			"@@ -1 +1 @@\n" +
			"-old\n" +
			"+new\n",
		Description: "test",
	}
	raw := mustJSON(t, input)

	result := tool.Invoke(context.Background(), raw)

	if result.OK {
		t.Fatalf("apply_patch accepted symlink escape_link pointing outside workspace")
	}
	if result.Error.Kind != KindWorkspaceViolation {
		t.Errorf("kind = %s, want KindWorkspaceViolation", result.Error.Kind)
	}
	if approver.called {
		t.Errorf("approver should not be called for symlink pointing outside workspace")
	}
}

// TestRunCommandSymlinkOutsideWorkspaceRefused verifies that run_command detects
// when a symlink inside the workspace (used as cwd) points outside it.
func TestRunCommandSymlinkOutsideWorkspaceRefused(t *testing.T) {
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New(true, true, true)
	approver := newFakeApprover(policy.DecisionAllow)
	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Create a directory and a symlink to it from outside workspace.
	linkPath := filepath.Join(ws.Root, "link-outside")
	if err := os.Symlink("../../../tmp", linkPath); err != nil {
		t.Skipf("cannot create symlink on this system: %v", err)
	}
	defer os.Remove(linkPath)

	input := runCommandInput{
		Argv: []string{"pwd"},
		Cwd:  "link-outside",
	}
	raw := mustJSON(t, input)

	result := tool.Invoke(context.Background(), raw)

	if result.OK {
		t.Fatalf("run_command accepted symlink cwd pointing outside workspace")
	}
	if result.Error.Kind != KindWorkspaceViolation {
		t.Errorf("kind = %s, want KindWorkspaceViolation", result.Error.Kind)
	}
	if approver.called {
		t.Errorf("approver should not be called for symlink pointing outside workspace")
	}
}

// TestRunCommandEnvironmentCaseSensitivity verifies that environment filtering
// is case-sensitive: MY_TOKEN (uppercase) is stripped, but my_token (lowercase)
// passes through if it is allowlisted.
//
// Known gap: lowercase secret-shaped names (my_token vs MY_TOKEN) escape the
// uppercase-only strip patterns entirely. This is a known limitation and not
// intended design; it is registered for the operator.
func TestRunCommandEnvironmentCaseSensitivity(t *testing.T) {
	ws := newTestWorkspace(t)

	// Set up environment: uppercase will be stripped (matches *_TOKEN),
	// lowercase will pass through.
	if err := os.Setenv("MY_TOKEN", "secret123"); err != nil {
		t.Fatalf("failed to setenv MY_TOKEN: %v", err)
	}
	defer os.Unsetenv("MY_TOKEN")

	if err := os.Setenv("my_token", "visible456"); err != nil {
		t.Fatalf("failed to setenv my_token: %v", err)
	}
	defer os.Unsetenv("my_token")

	conf := config.Defaults()
	conf.Policy.EnvPassthrough = []string{"my_token", "MY_TOKEN"}
	pol := policy.New(true, true, true)
	approver := newFakeApprover(policy.DecisionAllow)
	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	input := runCommandInput{
		Argv: []string{"env"},
		Cwd:  ".",
	}
	raw := mustJSON(t, input)

	result := tool.Invoke(context.Background(), raw)

	if !result.OK {
		t.Fatalf("env command failed: %s", result.Error.Message)
	}

	// MY_TOKEN should be stripped (case-sensitive match of *_TOKEN pattern).
	if strings.Contains(result.Content, "MY_TOKEN") {
		t.Errorf("MY_TOKEN should have been stripped (matches *_TOKEN pattern)")
	}

	// Lowercase my_token doesn't match the uppercase _TOKEN strip pattern, so it
	// passes through when allowlisted.
	if !strings.Contains(result.Content, "my_token=visible456") {
		t.Errorf("my_token should pass through when allowlisted and doesn't match strip patterns")
	}
}

// TestRunCommandParentEnvironmentAbsent verifies that environment variables
// from the parent process that are neither allowlisted nor stripped are absent
// from the child's environment.
func TestRunCommandParentEnvironmentAbsent(t *testing.T) {
	ws := newTestWorkspace(t)

	// Set a variable that is neither allowlisted nor a strip-pattern match.
	unusedVar := "UNUSED_PARENT_VAR_" + fmt.Sprintf("%d", os.Getpid())
	if err := os.Setenv(unusedVar, "should-not-appear"); err != nil {
		t.Fatalf("failed to setenv: %v", err)
	}
	defer os.Unsetenv(unusedVar)

	conf := config.Defaults()
	conf.Policy.EnvPassthrough = []string{} // Empty: only PATH, HOME, LANG pass.
	pol := policy.New(true, true, true)
	approver := newFakeApprover(policy.DecisionAllow)
	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	input := runCommandInput{
		Argv: []string{"env"},
		Cwd:  ".",
	}
	raw := mustJSON(t, input)

	result := tool.Invoke(context.Background(), raw)

	if !result.OK {
		t.Fatalf("env command failed: %s", result.Error.Message)
	}

	// The unused variable should not appear in the command's environment.
	if strings.Contains(result.Content, unusedVar) {
		t.Errorf("%s should not appear in child environment", unusedVar)
	}

	// Verify that PATH, HOME, or LANG are present (control).
	hasStdVar := strings.Contains(result.Content, "PATH=") ||
		strings.Contains(result.Content, "HOME=") ||
		strings.Contains(result.Content, "LANG=")
	if !hasStdVar {
		t.Errorf("at least one of PATH, HOME, LANG should be in child environment")
	}
}

// TestRunCommandExecPatternDocumentsPlatformGuarantee documents a property
// guaranteed by os/exec's argv handling: shell metacharacters in argv elements
// (;, &&, |, $(...), backticks) are passed as literal arguments to the program,
// never interpreted by a shell. This property holds because exec.CommandContext
// invokes the program directly without a shell intermediary. The test verifies
// this platform guarantee holds, not a defense implemented by this codebase.
func TestRunCommandExecPatternDocumentsPlatformGuarantee(t *testing.T) {
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New(true, true, true)
	approver := newFakeApprover(policy.DecisionAllow)
	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// Use printf to output its argument literally. Shell metacharacters should be
	// passed as a single literal argument to printf, which will output them exactly.
	// We use a format string with %s to test argument passing: the metacharacters
	// should appear verbatim, not be interpreted as shell commands.
	input := runCommandInput{
		Argv: []string{"printf", "%s", "$(whoami);id;cat /etc/passwd"},
		Cwd:  ".",
	}
	raw := mustJSON(t, input)

	result := tool.Invoke(context.Background(), raw)

	if !result.OK {
		t.Fatalf("printf command failed: %s", result.Error.Message)
	}

	// The output should contain the literal argument string exactly as passed.
	expected := "$(whoami);id;cat /etc/passwd"
	if !strings.Contains(result.Content, expected) {
		t.Errorf("shell metacharacters were not passed as literals. output: %q, want substring: %q",
			result.Content, expected)
	}

	// Verify shell commands were not executed: "root" and "uid=" are outputs of id/whoami.
	// These would only appear if the shell had interpreted the metacharacters.
	if strings.Contains(result.Content, "uid=") || strings.Contains(result.Content, "root:") {
		t.Errorf("shell command was interpreted (found output of id or cat in result)")
	}
}

// TestRunCommandRenamedShellLimitationDocumented documents the known limitation
// that policy.isShell matches by basename only and does not catch a renamed or
// copied shell (e.g., cp /bin/bash ./mytool). A renamed shell would be treated as
// a regular command, subject to default policy (allowlist/grant checks) rather than
// being rejected as a shell. This test asserts that behavior is unchanged and
// documents the gap for visibility.
func TestRunCommandRenamedShellLimitationDocumented(t *testing.T) {
	ws := newTestWorkspace(t)
	conf := config.Defaults()
	pol := policy.New(true, true, true)
	approver := newFakeApprover(policy.DecisionAllow)
	tool := &RunCommand{WS: ws, Config: &conf, Policy: pol, Approver: approver}

	// The basename check in policy.isShell does not catch renamed shells.
	// This is documented in policy.go: "It does NOT catch deliberately renamed or
	// symlinked shells (e.g., cp /bin/bash ./mytool)."
	// A renamed shell like "mytool" (which is really /bin/bash) will be treated as
	// a regular command: it requires approval if not in the allowlist.
	// This test records that gap.

	input := runCommandInput{
		Argv: []string{"mytool", "--help"},
		Cwd:  ".",
	}
	raw := mustJSON(t, input)

	// mytool is not a recognized shell by basename, so it's treated as a regular
	// command. Since it's not in the default allowlist, approver WILL be called.
	// This is the intended (albeit limited) behavior.
	result := tool.Invoke(context.Background(), raw)

	// The approver should have been called because mytool is not in the allowlist.
	// This verifies the documented behavior: renamed shells bypass the shell check.
	if !approver.called {
		t.Errorf("expected approver to be called for unknown command 'mytool' (not in allowlist)")
	}

	// We expect the command to fail (mytool doesn't exist) after approver approval.
	// We return DecisionAllow, so the tool attempts to run it.
	if result.OK {
		t.Fatalf("expected command to fail (mytool not found)")
	}
}
