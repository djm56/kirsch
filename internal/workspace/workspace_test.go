package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixtureWS opens a testdata fixture with an explicit root. The fixtures are
// plain directories, not Git repos — a nested .git cannot be committed — so
// root detection is tested separately against a real `git init`.
func fixtureWS(t *testing.T, name string) *Workspace {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := newWorkspace(root, false)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

// TestSymlinkTable covers every row of milestone-1 Task 1.3. The allowed rows
// matter as much as the blocked ones: a containment check that refuses
// legitimate internal symlinks is also a bug, and a much quieter one.
func TestSymlinkTable(t *testing.T) {
	ws := fixtureWS(t, "repo-symlink-escape")

	// A checkout that materialised symlinks as plain files would make the
	// blocked rows pass for the wrong reason. Skip loudly instead.
	info, err := os.Lstat(filepath.Join(ws.Root, "link-escape"))
	if err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Skip("fixture symlinks materialised as plain files; " +
			"check core.symlinks and re-clone before trusting this package")
	}

	cases := []struct {
		path    string
		want    string // "ok", "violation", or "notfound"
		comment string
	}{
		{"inside.txt", "ok", "an ordinary file"},
		{"sub/target.txt", "ok", "an ordinary nested file"},
		{"link-ok", "ok", "symlink to a file inside the workspace"},
		{"link-dir-ok", "ok", "symlink to a directory inside the workspace"},
		{"link-dir-ok/target.txt", "ok", "path *through* an internal directory symlink"},
		{"link-escape", "violation", "symlink to /etc/passwd"},
		{"link-parent", "violation", "symlink to the parent directory"},
		{"link-loop", "violation", "self-referential symlink; must not hang"},
		{"link-dangling", "notfound", "symlink to a missing file is absent, not forbidden"},
		{"../escape.txt", "violation", "plain .. traversal"},
		{"sub/../../escape.txt", "violation", ".. traversal hidden mid-path"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got, err := ws.Resolve(tc.path)
			switch tc.want {
			case "ok":
				if err != nil {
					t.Fatalf("%s: %v (%s)", tc.path, err, tc.comment)
				}
				if !contained(got, ws.CanonicalRoot) {
					t.Errorf("resolved outside the root: %s", got)
				}
			case "violation":
				if !IsViolation(err) {
					t.Fatalf("%s: got err=%v, want a violation (%s)", tc.path, err, tc.comment)
				}
			case "notfound":
				if IsViolation(err) {
					t.Fatalf("%s: reported as a violation; a dangling link is missing, not forbidden", tc.path)
				}
				if err != nil {
					t.Fatalf("%s: %v; want a clean resolve so the tool can report file_not_found", tc.path, err)
				}
				if _, statErr := os.Stat(got); !os.IsNotExist(statErr) {
					t.Errorf("%s resolved to something that exists: %s", tc.path, got)
				}
			}
		})
	}
}

// TestSiblingPrefixIsNotContainment is the trailing-separator bug: without it,
// /home/me/project-evil passes a prefix check against /home/me/project.
func TestSiblingPrefixIsNotContainment(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	evil := filepath.Join(parent, "project-evil")
	for _, d := range []string{root, evil} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(evil, "loot.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := newWorkspace(root, false)
	if err != nil {
		t.Fatal(err)
	}

	if contained(evil, ws.CanonicalRoot) {
		t.Fatal("project-evil is treated as contained by project; the trailing separator is missing")
	}
	if _, err := ws.Resolve("../project-evil/loot.txt"); !IsViolation(err) {
		t.Errorf("reaching a sibling directory was allowed: %v", err)
	}
}

// TestDenylist checks every entry, including the normalisation case: a
// denylist applied before cleaning is not a denylist.
func TestDenylist(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	denied := []string{
		".env",
		".env.local",
		".env.production",
		"./foo/../.env",
		"sub/.env",
		"key.pem",
		"certs/server.pem",
		"id_rsa.key",
		".git/config",
		".git/HEAD",
		"sub/.git/config",
		".kirsch/config.toml",
		".kirsch/sessions/abc.jsonl",
	}
	for _, p := range denied {
		t.Run("denied/"+p, func(t *testing.T) {
			if _, err := ws.Resolve(p); !IsViolation(err) {
				t.Errorf("%s was not refused (err=%v)", p, err)
			}
		})
	}

	allowed := []string{
		"README.md",
		"main.go",
		"pkg/util/strings.go",
		"environment.md",  // contains "env" but is not .env
		"keyboard.go",     // ends in neither .key nor .pem
		"docs/.gitignore", // a dotfile that is not .git/
		"unicode.md",
	}
	for _, p := range allowed {
		t.Run("allowed/"+p, func(t *testing.T) {
			if _, err := ws.Resolve(p); IsViolation(err) {
				t.Errorf("%s was refused but should be allowed: %v", p, err)
			}
		})
	}
}

func TestInvalidInput(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	if _, err := ws.Resolve(""); !IsInvalidPath(err) {
		t.Errorf("empty path: got %v, want tool_input_invalid", err)
	}
	if _, err := ws.Resolve("/etc/passwd"); !IsInvalidPath(err) {
		t.Errorf("absolute path: got %v, want tool_input_invalid", err)
	}
	// Absolute paths are malformed input, not an attack — the distinction is
	// what lets the model self-correct rather than think it hit a wall.
	if IsViolation(mustErr(ws.Resolve("/etc/passwd"))) {
		t.Error("an absolute path should be invalid input, not a violation")
	}
}

func mustErr(_ string, err error) error { return err }

// TestDetectAgainstRealRepo exercises root detection, which the fixtures
// cannot: they are plain directories with no .git.
func TestDetectAgainstRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "T"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !ws.IsGit {
		t.Error("a git init'd directory was not detected as a repo")
	}

	// The macOS case the instruction set calls out: /tmp is a symlink to
	// /private/tmp, so a non-canonical root rejects everything under TempDir.
	if ws.CanonicalRoot == "" {
		t.Fatal("no canonical root")
	}
	if _, err := ws.Resolve("a.txt"); err != nil {
		t.Errorf("resolving inside a TempDir workspace failed — canonical root not applied: %v", err)
	}

	if !ws.IsDirty() {
		t.Error("a repo with an untracked file should report dirty")
	}
	if b := ws.Branch(); b == "" {
		t.Error("no branch reported for a real repo")
	}
	if ws.Name() != filepath.Base(ws.CanonicalRoot) {
		t.Errorf("Name() = %q", ws.Name())
	}
}

func TestDetectRejectsNonDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Detect(f); err == nil {
		t.Error("a file was accepted as a workspace")
	}
}

func TestNoWorkspaceErrorIsActionable(t *testing.T) {
	msg := ErrNoWorkspace.Error()
	for _, want := range []string{"Git repository", "--workspace"} {
		if !strings.Contains(msg, want) {
			t.Errorf("onboarding error does not mention %q: %s", want, msg)
		}
	}
}

// TestNonGitWorkspaceDegrades: --workspace on a plain directory is supported,
// and the header omits the branch rather than showing an error.
func TestNonGitWorkspaceDegrades(t *testing.T) {
	dir := t.TempDir()
	ws, err := Detect(dir)
	if err != nil {
		t.Fatalf("a non-Git directory must be usable with --workspace: %v", err)
	}
	if ws.IsGit {
		t.Error("a plain directory reported as a git repo")
	}
	if ws.Branch() != "" {
		t.Errorf("branch = %q, want empty outside a repo", ws.Branch())
	}
	if ws.IsDirty() {
		t.Error("a non-Git workspace reported dirty")
	}
}

// TestSymlinkLoopTerminatesQuickly: "must not hang" is a timing claim, so
// assert on time rather than on merely getting an answer.
func TestSymlinkLoopTerminatesQuickly(t *testing.T) {
	ws := fixtureWS(t, "repo-symlink-escape")
	done := make(chan error, 1)
	go func() {
		_, err := ws.Resolve("link-loop")
		done <- err
	}()
	select {
	case err := <-done:
		if !IsViolation(err) {
			t.Errorf("link-loop: got %v, want a violation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resolving a symlink loop did not terminate within 2s")
	}
}

// TestDanglingSymlinkOutsideRootIsRefused pins the machine-dependent hole that
// the first implementation had: EvalSymlinks on a link whose target is both
// outside the workspace and missing returns ErrNotExist, which reads as an
// ordinary absent file. The escape is then invisible on this machine and
// caught on one where the target happens to exist.
func TestDanglingSymlinkOutsideRootIsRefused(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	// Points outside the workspace at something that does not exist.
	if err := os.Symlink("../outside/nothing-here.txt", filepath.Join(root, "sneaky")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	ws, err := newWorkspace(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Resolve("sneaky"); !IsViolation(err) {
		t.Fatalf("a dangling symlink pointing outside the workspace was allowed (err=%v); "+
			"containment must not depend on whether the target exists", err)
	}
}

// TestSymlinkChainIsFollowed proves the hop budget does not refuse legitimate
// chains of links inside the workspace.
func TestSymlinkChainIsFollowed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(root, "a")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, l := range [][2]string{{"a", "b"}, {"b", "c"}, {"c", "d"}} {
		if err := os.Symlink(l[0], filepath.Join(root, l[1])); err != nil {
			t.Fatal(err)
		}
	}
	ws, err := newWorkspace(root, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ws.Resolve("d")
	if err != nil {
		t.Fatalf("a four-link chain inside the workspace was refused: %v", err)
	}
	if filepath.Base(got) != "real.txt" {
		t.Errorf("chain resolved to %q, want real.txt", got)
	}
}

// TestDenylistIsCaseInsensitive is a regression test for a bypass found by
// FuzzResolveNeverEscapes, not by review.
//
// macOS and Windows filesystems are case-insensitive by default, so `.ENV`
// opens the same bytes as `.env`. A case-sensitive denylist is therefore
// bypassable on two of the three platforms people actually use, by nothing
// cleverer than pressing shift. Plan §11 amendment 51.
func TestDenylistIsCaseInsensitive(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	variants := []string{
		".ENV", ".Env", ".eNv",
		".ENV.LOCAL", ".Env.Production",
		"KEY.PEM", "key.PEM", "Key.Pem",
		"ID_RSA.KEY", "id_rsa.Key",
		".GIT/config", ".Git/HEAD", "sub/.GIT/config",
		".KIRSCH/config.toml", ".Kirsch/sessions/a.jsonl",
	}
	for _, p := range variants {
		t.Run(p, func(t *testing.T) {
			if _, err := ws.Resolve(p); !IsViolation(err) {
				t.Errorf("%s was not refused (err=%v); on a case-insensitive "+
					"filesystem this reads the real file", p, err)
			}
		})
	}

	// The rule must not over-reach: these merely contain the same letters.
	for _, p := range []string{"environment.md", "monkey.go", "keyboard.txt", "gitignore.md"} {
		t.Run("allowed/"+p, func(t *testing.T) {
			if _, err := ws.Resolve(p); IsViolation(err) {
				t.Errorf("%s was refused but is not denylisted: %v", p, err)
			}
		})
	}
}
