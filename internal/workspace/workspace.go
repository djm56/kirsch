// Package workspace is the security boundary: it decides which paths Kirsch
// may touch, and nothing else in the codebase is allowed to decide otherwise.
//
// Every filesystem path a tool uses crosses Resolve. That is architecture.md
// §3 rule 5, and it exists so that "can this tool read .env?" has exactly one
// answer in exactly one place rather than one answer per tool.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// Violation is returned when a path escapes the workspace or hits the
// denylist. Callers map it to the workspace_violation tool error kind.
type Violation struct {
	Path   string
	Reason string
}

func (v *Violation) Error() string {
	return fmt.Sprintf("path %q is outside the workspace or denied: %s", v.Path, v.Reason)
}

// InvalidPath is returned for input that is malformed rather than forbidden —
// an empty path, or an absolute one. Callers map it to tool_input_invalid.
type InvalidPath struct {
	Path   string
	Reason string
}

func (e *InvalidPath) Error() string {
	return fmt.Sprintf("invalid path %q: %s", e.Path, e.Reason)
}

// Workspace is a resolved repository root.
//
// Root and CanonicalRoot are both kept deliberately. On macOS /tmp is a
// symlink to /private/tmp, so a containment check against a non-canonical root
// rejects every path under t.TempDir() — which is every test in this package.
type Workspace struct {
	Root          string
	CanonicalRoot string
	IsGit         bool

	mu        sync.Mutex
	branch    string
	dirty     bool
	metaAt    time.Time
	metaValid bool

	ignore      gitignore.Matcher
	ignoreValid bool
}

// ErrNoWorkspace is returned when neither --workspace nor a Git root is found.
var ErrNoWorkspace = errors.New(
	"Kirsch runs inside a Git repository.\n\n" +
		"Either cd into a repository, or point Kirsch at a directory explicitly:\n" +
		"    kirsch --workspace /path/to/project")

// Detect resolves the workspace root: an explicit --workspace if given,
// otherwise the enclosing Git repository.
func Detect(flagWorkspace string) (*Workspace, error) {
	if flagWorkspace != "" {
		st, err := os.Stat(flagWorkspace)
		if err != nil {
			return nil, fmt.Errorf("--workspace %s: %w", flagWorkspace, err)
		}
		if !st.IsDir() {
			return nil, fmt.Errorf("--workspace %s is not a directory", flagWorkspace)
		}
		abs, err := filepath.Abs(flagWorkspace)
		if err != nil {
			return nil, err
		}
		return newWorkspace(abs, isGitRepo(abs))
	}

	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, ErrNoWorkspace
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return nil, ErrNoWorkspace
	}
	return newWorkspace(root, true)
}

func newWorkspace(root string, isGit bool) (*Workspace, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root %s: %w", root, err)
	}
	return &Workspace{Root: root, CanonicalRoot: canonical, IsGit: isGit}, nil
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// denyPatterns are paths no tool may touch, whatever it is asked.
//
// Matched against the cleaned workspace-relative path, so ./foo/../.env is
// caught. A denylist checked before normalisation is not a denylist.
var denyPatterns = []string{
	".env",
	".env.*",
	"*.pem",
	"*.key",
}

// denyPrefixes are directory subtrees excluded entirely.
var denyPrefixes = []string{
	".git",
	".kirsch",
}

// Resolve turns a workspace-relative path into an absolute one, or refuses.
//
// The order matters: input validity, then the denylist on the *cleaned*
// relative path, then canonical containment. Checking containment first would
// let a symlinked .env inside the workspace through, because it is,
// technically, contained.
func (w *Workspace) Resolve(rel string) (string, error) {
	if rel == "" {
		return "", &InvalidPath{Path: rel, Reason: "path is empty"}
	}
	if filepath.IsAbs(rel) {
		return "", &InvalidPath{
			Path:   rel,
			Reason: "absolute paths are not accepted; give a path relative to the workspace root",
		}
	}

	clean := filepath.Clean(rel)
	if err := checkDenylist(clean); err != nil {
		return "", err
	}

	resolved, err := w.resolveWithin(clean)
	if err != nil {
		var v *Violation
		if errors.As(err, &v) {
			v.Path = rel
			return "", v
		}
		return "", err
	}

	if !contained(resolved, w.CanonicalRoot) {
		return "", &Violation{Path: rel, Reason: "resolves outside the workspace root"}
	}
	// Re-checked on the resolved path, so a symlink aimed at a denied file
	// inside the workspace cannot launder it past the first check.
	if inside, relErr := filepath.Rel(w.CanonicalRoot, resolved); relErr == nil {
		if err := checkDenylist(inside); err != nil {
			return "", err
		}
	}
	return resolved, nil
}

// maxSymlinkHops bounds link following. EvalSymlinks would return ELOOP for a
// cycle, but this resolver follows links itself, so it needs its own budget.
const maxSymlinkHops = 40

// resolveWithin walks rel one component at a time from root, following
// symlinks by hand and checking containment at every hop.
//
// The obvious implementation — EvalSymlinks the whole path, then check the
// result — has a hole that is both serious and machine-dependent. A symlink
// pointing outside the workspace at a target that happens not to exist makes
// EvalSymlinks return ErrNotExist, which is indistinguishable from an ordinary
// missing file; the path is then treated as an in-workspace file that simply
// is not there, and the escape is never noticed. On a machine where the same
// target does exist, the identical link is correctly refused. A containment
// check whose answer depends on whether the attacker's target is present is
// not a containment check.
//
// Resolving stepwise closes that: every symlink's target is checked for
// containment when the link is read, whether or not the target exists.
func (w *Workspace) resolveWithin(rel string) (string, error) {
	cur := w.CanonicalRoot
	todo := strings.Split(filepath.ToSlash(rel), "/")
	hops := 0

	deny := func(reason string) (string, error) {
		return "", &Violation{Path: rel, Reason: reason}
	}

	for len(todo) > 0 {
		seg := todo[0]
		todo = todo[1:]

		switch seg {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			if !contained(cur, w.CanonicalRoot) {
				return deny("path traverses above the workspace root")
			}
			continue
		}

		next := filepath.Join(cur, seg)
		info, err := os.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				// A non-existent tail cannot hide a symlink — nothing exists to
				// be one — so it is safe to append, and the caller reports
				// file_not_found rather than a violation.
				out := filepath.Join(append([]string{next}, todo...)...)
				if !contained(out, w.CanonicalRoot) {
					return deny("resolves outside the workspace root")
				}
				return out, nil
			}
			return deny("path could not be inspected: " + err.Error())
		}

		if info.Mode()&os.ModeSymlink == 0 {
			cur = next
			if !contained(cur, w.CanonicalRoot) {
				return deny("resolves outside the workspace root")
			}
			continue
		}

		// A symlink: replace it with its target's components and keep walking,
		// so a link to a link to a link is followed the same way — and a link
		// to itself burns the hop budget instead of spinning.
		hops++
		if hops > maxSymlinkHops {
			return deny("symlink chain is too deep or cyclic")
		}
		target, err := os.Readlink(next)
		if err != nil {
			return deny("unreadable symlink: " + err.Error())
		}
		parts := strings.Split(filepath.ToSlash(target), "/")
		if filepath.IsAbs(target) {
			// Restart from the filesystem root; containment then rejects it
			// unless the target happens to lie inside the workspace.
			cur = string(filepath.Separator)
		}
		todo = append(parts, todo...)
	}

	if !contained(cur, w.CanonicalRoot) {
		return deny("resolves outside the workspace root")
	}
	return cur, nil
}

// contained reports whether p is the root or lives beneath it.
//
// The trailing separator is load-bearing: without it, /home/me/project-evil
// passes a prefix check against /home/me/project.
func contained(p, root string) bool {
	return p == root || strings.HasPrefix(p, root+string(filepath.Separator))
}

// checkDenylist refuses paths that no tool may touch.
func checkDenylist(rel string) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return nil
	}
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return &Violation{Path: rel, Reason: "path traverses above the workspace root"}
	}

	segments := strings.Split(rel, "/")
	for _, seg := range segments {
		for _, prefix := range denyPrefixes {
			if seg == prefix {
				return &Violation{Path: rel, Reason: prefix + "/ is never readable"}
			}
		}
	}

	base := segments[len(segments)-1]
	for _, pattern := range denyPatterns {
		if ok, _ := path.Match(pattern, base); ok {
			return &Violation{Path: rel, Reason: "matches the denied pattern " + pattern}
		}
	}
	return nil
}

// IsViolation reports whether err is a containment or denylist refusal.
func IsViolation(err error) bool {
	var v *Violation
	return errors.As(err, &v)
}

// IsInvalidPath reports whether err is malformed input rather than a refusal.
func IsInvalidPath(err error) bool {
	var e *InvalidPath
	return errors.As(err, &e)
}

// Rel returns the workspace-relative form of an absolute path.
func (w *Workspace) Rel(abs string) string {
	if r, err := filepath.Rel(w.CanonicalRoot, abs); err == nil {
		return filepath.ToSlash(r)
	}
	return abs
}

// Name is the workspace directory's base name, shown in the TUI header.
func (w *Workspace) Name() string { return filepath.Base(w.CanonicalRoot) }

// metaTTL bounds how stale branch and dirty state may be. The TUI header reads
// them on every render; shelling out to git that often would be absurd.
const metaTTL = 2 * time.Second

// Branch returns the current branch, or "" outside a Git repository.
func (w *Workspace) Branch() string {
	w.refreshMeta(false)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.branch
}

// IsDirty reports uncommitted changes.
func (w *Workspace) IsDirty() bool {
	w.refreshMeta(false)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dirty
}

// RefreshMeta forces a re-read of branch and dirty state.
func (w *Workspace) RefreshMeta() { w.refreshMeta(true) }

func (w *Workspace) refreshMeta(force bool) {
	if !w.IsGit {
		return
	}
	w.mu.Lock()
	fresh := w.metaValid && time.Since(w.metaAt) < metaTTL
	w.mu.Unlock()
	if fresh && !force {
		return
	}

	// rev-parse --abbrev-ref HEAD fails in a repository with no commits, which
	// is a normal state (git init, then start work). symbolic-ref still knows
	// the branch name, so fall back rather than reporting none.
	branch := ""
	if out, err := exec.Command("git", "-C", w.Root, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		branch = strings.TrimSpace(string(out))
	}
	if branch == "" || branch == "HEAD" {
		if out, err := exec.Command("git", "-C", w.Root, "symbolic-ref", "--short", "HEAD").Output(); err == nil {
			branch = strings.TrimSpace(string(out))
		}
	}
	dirty := false
	if out, err := exec.Command("git", "-C", w.Root, "status", "--porcelain").Output(); err == nil {
		dirty = len(strings.TrimSpace(string(out))) > 0
	}

	w.mu.Lock()
	w.branch, w.dirty, w.metaAt, w.metaValid = branch, dirty, time.Now(), true
	w.mu.Unlock()
}
