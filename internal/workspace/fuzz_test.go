package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Fuzzing the containment boundary.
//
// Resolve is the single place that decides whether Kirsch may touch a path, and
// from Milestone 3 its input is written by a language model against a
// repository Kirsch did not author. Both halves of that sentence are reasons to
// throw hostile input at it rather than only the cases someone thought of:
// plan §11 amendment 45 records a containment hole that the fixtures caught and
// review did not.
//
// Run properly now and then:
//   go test ./internal/workspace/ -run='^FuzzResolve$' -fuzz='^FuzzResolve$' -fuzztime=60s

// FuzzResolveNeverEscapes is the property that matters more than any other in
// this codebase: whatever Resolve returns, it is inside the workspace.
func FuzzResolveNeverEscapes(f *testing.F) {
	seeds := []string{
		"", ".", "..", "a.txt", "./a.txt", "a/../b.txt",
		"../escape", "../../etc/passwd", "/etc/passwd",
		".env", ".env.local", "x.pem", "x.key", ".git/config", ".kirsch/x",
		"a/./b/../c", strings.Repeat("../", 64) + "etc/passwd",
		"a\x00b", "a\nb", "日本語.txt", "a//b///c",
		strings.Repeat("a/", 512) + "deep.txt",
		"./././././.env",
		"sub/../../.env",
		" .env", ".env ", ".ENV",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	root := f.TempDir()
	for _, d := range []string{"sub", "sub/nested", ".git", ".kirsch"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			f.Fatal(err)
		}
	}
	for _, name := range []string{"a.txt", "sub/b.txt", ".env", "key.pem", ".git/config"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			f.Fatal(err)
		}
	}
	ws, err := newWorkspace(root, false)
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, rel string) {
		got, err := ws.Resolve(rel)
		if err != nil {
			return // a refusal is always an acceptable answer
		}

		// Accepted. Now it must genuinely be inside the workspace — checked
		// independently of the logic under test, on the returned path.
		if !filepath.IsAbs(got) {
			t.Fatalf("Resolve(%q) returned a relative path %q", rel, got)
		}
		clean := filepath.Clean(got)
		if clean != ws.CanonicalRoot &&
			!strings.HasPrefix(clean, ws.CanonicalRoot+string(filepath.Separator)) {
			t.Fatalf("Resolve(%q) escaped the workspace: %q", rel, got)
		}

		// And it must not be something the denylist forbids, however the input
		// spelled it. `./foo/../.env` is still `.env`.
		inside, relErr := filepath.Rel(ws.CanonicalRoot, clean)
		if relErr != nil {
			t.Fatalf("Resolve(%q) returned an unrelatable path %q", rel, got)
		}
		base := strings.ToLower(filepath.Base(inside))
		if base == ".env" || strings.HasPrefix(base, ".env.") ||
			strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") {
			t.Fatalf("Resolve(%q) allowed a denylisted file: %q", rel, inside)
		}
		for _, seg := range strings.Split(filepath.ToSlash(inside), "/") {
			if seg == ".git" || seg == ".kirsch" {
				t.Fatalf("Resolve(%q) allowed a path through %s: %q", rel, seg, inside)
			}
		}
	})
}

// FuzzResolveTerminates guards against the other failure mode. A path that
// makes Resolve loop is a hang, and a hang in a tool is a hung interface with
// no error to report.
func FuzzResolveTerminates(f *testing.F) {
	f.Add("a")
	f.Add(strings.Repeat("../", 1000))
	f.Add(strings.Repeat("a/", 4000))

	root := f.TempDir()
	// A cycle, so the fuzzer can reach the link-following budget.
	_ = os.Symlink("loop", filepath.Join(root, "loop"))
	_ = os.Symlink("loop2", filepath.Join(root, "loop1"))
	_ = os.Symlink("loop1", filepath.Join(root, "loop2"))

	ws, err := newWorkspace(root, false)
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, rel string) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = ws.Resolve(rel)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Resolve(%q) did not terminate", rel)
		}
	})
}
