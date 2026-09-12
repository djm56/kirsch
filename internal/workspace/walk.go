package workspace

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// builtinIgnore is always applied and is deliberately not overridable by a
// .gitignore negation.
//
// A repository that writes `!node_modules` has made a choice about its own
// tooling, not about what belongs in a language model's context window. vendor/
// is here for the same reason and is the one entry with a real cost — it hides
// genuine source in a Go module that vendors its dependencies. Plan amendment
// 44 records that trade and the intent to make it configurable later.
var builtinIgnore = []string{
	".git",
	".kirsch",
	"node_modules",
	"vendor",
	".DS_Store",
}

// Entry is one item found by Walk.
type Entry struct {
	Rel   string // workspace-relative, slash-separated
	IsDir bool
	Size  int64
}

// WalkOptions configures a walk.
type WalkOptions struct {
	// MaxDepth counts from the start path, not from the workspace root. Zero
	// means the start directory's immediate children only; negative is
	// unlimited.
	MaxDepth      int
	IncludeHidden bool
	// Limit caps the number of entries returned. Zero means unlimited.
	Limit int
}

// Walk visits files under rel in lexical order.
//
// Deterministic ordering is not cosmetic: non-deterministic tool output makes
// golden tests flaky and, from M3, makes the model's behaviour irreproducible
// between identical runs.
//
// Symlinked directories are never followed. That prevents cycles and escapes in
// one rule, and is far simpler than detecting either mid-walk.
func (w *Workspace) Walk(rel string, o WalkOptions, fn func(Entry) bool) error {
	start, err := w.Resolve(rel)
	if err != nil {
		return err
	}
	matcher, err := w.ignoreMatcher()
	if err != nil {
		return err
	}
	count := 0
	return w.walkDir(start, 0, o, matcher, &count, fn)
}

func (w *Workspace) walkDir(dir string, depth int, o WalkOptions, m gitignore.Matcher, count *int, fn func(Entry) bool) error {
	if o.MaxDepth >= 0 && depth > o.MaxDepth {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // an unreadable directory is skipped, never fatal
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, de := range entries {
		name := de.Name()
		if !o.IncludeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if isBuiltinIgnored(name) {
			continue
		}

		abs := filepath.Join(dir, name)
		rel := w.Rel(abs)
		parts := strings.Split(rel, "/")

		// A symlink is reported but never descended into.
		info, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		isSymlink := info.Mode()&os.ModeSymlink != 0
		isDir := de.IsDir() && !isSymlink

		if m != nil && m.Match(parts, isDir) {
			continue
		}
		if checkDenylist(rel) != nil {
			continue
		}

		if o.Limit > 0 && *count >= o.Limit {
			return errLimit
		}
		*count++

		size := int64(0)
		if !isDir {
			size = info.Size()
		}
		if !fn(Entry{Rel: rel, IsDir: isDir, Size: size}) {
			return nil
		}
		if isDir {
			if err := w.walkDir(abs, depth+1, o, m, count, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// errLimit signals that WalkOptions.Limit was reached. Callers translate it to
// a truncated result rather than an error.
var errLimit = &limitReached{}

type limitReached struct{}

func (l *limitReached) Error() string { return "walk limit reached" }

// IsLimitReached reports whether a walk stopped because of WalkOptions.Limit.
func IsLimitReached(err error) bool {
	_, ok := err.(*limitReached)
	return ok
}

func isBuiltinIgnored(name string) bool {
	for _, b := range builtinIgnore {
		if name == b {
			return true
		}
	}
	return false
}

// ignoreMatcher builds a matcher from every .gitignore in the tree.
//
// go-git's implementation is used rather than a hand-rolled one because the
// cases it gets right — nested files, negation, anchored vs floating patterns,
// ** — are exactly the cases where a simpler matcher fails *open*, silently
// including a file. Plan amendment 43.
func (w *Workspace) ignoreMatcher() (gitignore.Matcher, error) {
	w.mu.Lock()
	if w.ignoreValid {
		m := w.ignore
		w.mu.Unlock()
		return m, nil
	}
	w.mu.Unlock()

	var patterns []gitignore.Pattern
	err := filepath.WalkDir(w.CanonicalRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isBuiltinIgnored(d.Name()) && p != w.CanonicalRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != ".gitignore" {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		domain := strings.Split(strings.TrimPrefix(filepath.ToSlash(w.Rel(filepath.Dir(p))), "./"), "/")
		if len(domain) == 1 && domain[0] == "." {
			domain = nil
		}
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			patterns = append(patterns, gitignore.ParsePattern(line, domain))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	m := gitignore.NewMatcher(patterns)
	w.mu.Lock()
	w.ignore, w.ignoreValid = m, true
	w.mu.Unlock()
	return m, nil
}

// InvalidateIgnoreCache forces .gitignore files to be re-read.
func (w *Workspace) InvalidateIgnoreCache() {
	w.mu.Lock()
	w.ignore, w.ignoreValid = nil, false
	w.mu.Unlock()
}

// IsIgnored reports whether a workspace-relative path is excluded from the
// walk, by either the built-in list or a .gitignore.
func (w *Workspace) IsIgnored(rel string, isDir bool) bool {
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if isBuiltinIgnored(seg) {
			return true
		}
	}
	m, err := w.ignoreMatcher()
	if err != nil || m == nil {
		return false
	}
	return m.Match(strings.Split(filepath.ToSlash(rel), "/"), isDir)
}
