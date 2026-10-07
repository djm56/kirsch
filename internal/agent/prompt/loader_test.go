package prompt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// fakeEngine is a test double for the workspace containment engine. It resolves
// relative paths under root, and can be configured to refuse named candidates
// with an error, matching the real engine's behaviour for absolute paths,
// traversal, symlink escape, and denylisted entries.
type fakeEngine struct {
	root    string
	refused map[string]error
}

func newFakeEngine(root string) *fakeEngine {
	return &fakeEngine{root: root, refused: make(map[string]error)}
}

func (f *fakeEngine) refuse(rel string, err error) *fakeEngine {
	f.refused[rel] = err
	return f
}

// Resolve implements the Engine interface.
func (f *fakeEngine) Resolve(rel string) (string, error) {
	if err, ok := f.refused[rel]; ok {
		return "", err
	}
	if filepath.IsAbs(rel) {
		return rel, nil
	}
	return filepath.Join(f.root, rel), nil
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func writeDir(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
	return path
}

func writeFIFO(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if runtime.GOOS == "windows" {
		t.Skip("FIFO test skipped on windows")
	}
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Fatalf("mkfifo %s: %v", name, err)
	}
	return path
}

func hasWarningNamed(warnings []string, name string) bool {
	for _, w := range warnings {
		if strings.Contains(w, fmt.Sprintf("%q", name)) {
			return true
		}
	}
	return false
}

// TestLoadProjectContext_FirstExistingCandidateWins falsifies any loader that
// does not treat the candidate list as FIFO. The final sub-case covers the
// no-candidates-remain situation, asserting empty content, no chosen file,
// zero size, and no warnings.
func TestLoadProjectContext_FirstExistingCandidateWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "AGENTS.md", "AGENTS content\n")
	writeFile(t, dir, "CLAUDE.md", "CLAUDE content\n")
	engine := newFakeEngine(dir)

	content, chosen, size, warnings, err := LoadProjectContext([]string{"AGENTS.md", "CLAUDE.md"}, 1024, engine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chosen != "AGENTS.md" {
		t.Fatalf("chosen: got %q, want AGENTS.md", chosen)
	}
	if content != "AGENTS content\n" {
		t.Fatalf("content: got %q, want AGENTS content\\n", content)
	}
	if size != len("AGENTS content\n") {
		t.Fatalf("size: got %d, want %d", size, len("AGENTS content\n"))
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings: got %v, want none", warnings)
	}

	// With only CLAUDE.md present, it wins.
	os.Remove(filepath.Join(dir, "AGENTS.md"))
	content, chosen, _, warnings, err = LoadProjectContext([]string{"AGENTS.md", "CLAUDE.md"}, 1024, engine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chosen != "CLAUDE.md" {
		t.Fatalf("chosen after removing AGENTS.md: got %q, want CLAUDE.md", chosen)
	}
	if content != "CLAUDE content\n" {
		t.Fatalf("content after removing AGENTS.md: got %q", content)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings after removing AGENTS.md: got %v, want none", warnings)
	}

	// With no candidates present, there is no project context and no error.
	os.Remove(filepath.Join(dir, "CLAUDE.md"))
	content, chosen, size, warnings, err = LoadProjectContext([]string{"AGENTS.md", "CLAUDE.md"}, 1024, engine)
	if err != nil {
		t.Fatalf("unexpected error when empty: %v", err)
	}
	if chosen != "" {
		t.Fatalf("chosen when empty: got %q, want empty", chosen)
	}
	if content != "" {
		t.Fatalf("content when empty: got %q, want empty", content)
	}
	if size != 0 {
		t.Fatalf("size when empty: got %d, want 0", size)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings when empty: got %v, want none", warnings)
	}
}

// TestLoadProjectContext_RefusedCandidatesProduceNamedWarnings falsifies any
// loader that treats an engine refusal as a hard error, that swallows the
// refused candidate's name, or that injects content from a refused path.
func TestLoadProjectContext_RefusedCandidatesProduceNamedWarnings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "valid.md", "valid content\n")
	engine := newFakeEngine(dir).
		refuse("/abs.md", errors.New("absolute path")).
		refuse("../outside.md", errors.New("path outside workspace")).
		refuse("escaped-link", errors.New("symlink resolves outside root")).
		refuse(".env", errors.New("denylisted path"))

	candidates := []string{"/abs.md", "../outside.md", "escaped-link", ".env", "valid.md"}
	content, chosen, size, warnings, err := LoadProjectContext(candidates, 1024, engine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chosen != "valid.md" {
		t.Fatalf("chosen: got %q, want valid.md", chosen)
	}
	if content != "valid content\n" {
		t.Fatalf("content: got %q", content)
	}
	if size != len("valid content\n") {
		t.Fatalf("size: got %d, want %d", size, len("valid content\n"))
	}

	wantRefused := []string{"/abs.md", "../outside.md", "escaped-link", ".env"}
	if len(warnings) != len(wantRefused) {
		t.Fatalf("warnings count: got %d, want %d: %v", len(warnings), len(wantRefused), warnings)
	}
	for _, name := range wantRefused {
		if !hasWarningNamed(warnings, name) {
			t.Fatalf("missing warning naming %q in %v", name, warnings)
		}
	}
}

// TestLoadProjectContext_NonRegularFilesSkippedWithWarning falsifies any loader
// that opens a FIFO or directory candidate. The regular-file check must come
// before Open, so this test cannot hang.
func TestLoadProjectContext_NonRegularFilesSkippedWithWarning(t *testing.T) {
	dir := t.TempDir()
	writeDir(t, dir, "adir")
	writeFIFO(t, dir, "afifo")
	writeFile(t, dir, "valid.md", "valid content\n")
	engine := newFakeEngine(dir)

	content, chosen, _, warnings, err := LoadProjectContext([]string{"adir", "afifo", "valid.md"}, 1024, engine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chosen != "valid.md" {
		t.Fatalf("chosen: got %q, want valid.md", chosen)
	}
	if content != "valid content\n" {
		t.Fatalf("content: got %q", content)
	}
	if !hasWarningNamed(warnings, "adir") {
		t.Fatalf("missing warning naming directory candidate in %v", warnings)
	}
	if !hasWarningNamed(warnings, "afifo") {
		t.Fatalf("missing warning naming FIFO candidate in %v", warnings)
	}
}

// TestLoadProjectContext_CapTruncationAndClamp falsifies any loader that trusts
// the Stat size, that does not truncate at a line boundary, that omits the
// truncation marker, or that honours a cap above the global ceiling.
func TestLoadProjectContext_CapTruncationAndClamp(t *testing.T) {
	t.Run("exact cap is not marked", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "ctx.md", "line1\nline2")
		content, _, _, _, err := LoadProjectContext([]string{"ctx.md"}, 12, newFakeEngine(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "line1\nline2" {
			t.Fatalf("content: got %q, want no truncation", content)
		}
		if strings.Contains(content, "truncated") {
			t.Fatalf("exact-cap content must not carry a truncation marker: %q", content)
		}
	})

	t.Run("over cap truncates at line boundary and marks", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "ctx.md", "line1\nline2\nline3\n")
		content, _, _, _, err := LoadProjectContext([]string{"ctx.md"}, 10, newFakeEngine(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "line1\n[project context truncated after 10 bytes]\n"
		if content != want {
			t.Fatalf("content: got %q, want %q", content, want)
		}
	})

	t.Run("cap above 32768 is clamped", func(t *testing.T) {
		dir := t.TempDir()
		// Build content well over 32768 bytes with clear line boundaries.
		line := strings.Repeat("x", 100) + "\n" // 101 bytes per line
		var b strings.Builder
		for b.Len() <= 40000 {
			b.WriteString(line)
		}
		writeFile(t, dir, "ctx.md", b.String())
		content, _, _, _, err := LoadProjectContext([]string{"ctx.md"}, 100000, newFakeEngine(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(content, "[project context truncated after 32768 bytes]") {
			t.Fatalf("cap clamp marker missing in content of length %d", len(content))
		}
		// The returned content must not exceed the clamped cap plus the marker.
		if len(content) > 32768+len("[project context truncated after 32768 bytes]\n") {
			t.Fatalf("content length %d exceeds clamped cap plus marker", len(content))
		}
	})

	t.Run("lower cap is honoured", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "ctx.md", "line1\nline2\nline3\n")
		content, _, _, _, err := LoadProjectContext([]string{"ctx.md"}, 10, newFakeEngine(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(content, "[project context truncated after 10 bytes]") {
			t.Fatalf("lower-cap marker missing: %q", content)
		}
	})
}

// TestLoadProjectContext_BoundedReadIgnoresStatSize falsifies any loader that
// relies on Stat to size the read buffer. The bounded reader makes the limit
// structural, so this test uses a positive control: a file larger than the cap
// is truncated regardless of what Stat reports.
func TestLoadProjectContext_BoundedReadIgnoresStatSize(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("word\n", 5000) // well over 1024 bytes
	writeFile(t, dir, "big.md", big)
	content, _, size, _, err := LoadProjectContext([]string{"big.md"}, 1024, newFakeEngine(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size <= 1024 {
		t.Fatalf("test setup error: file size %d should be larger than cap", size)
	}
	if !strings.Contains(content, "[project context truncated after 1024 bytes]") {
		t.Fatalf("bounded read did not truncate: %q", content)
	}
	if len(content) > 1024+len("[project context truncated after 1024 bytes]\n") {
		t.Fatalf("content length %d exceeds cap plus marker", len(content))
	}
}

// TestLoadProjectContext_PureFunctionOfInputs falsifies any loader that caches,
// mutates state, or produces different outputs for identical inputs.
func TestLoadProjectContext_PureFunctionOfInputs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ctx.md", "content\n")
	engine := newFakeEngine(dir)

	first := loadAll(LoadProjectContext([]string{"ctx.md"}, 100, engine))
	second := loadAll(LoadProjectContext([]string{"ctx.md"}, 100, engine))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same inputs produced different outputs:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// loadAll collapses the loader's multiple return values into a single comparable
// value for the purity test.
func loadAll(content, chosen string, size int, warnings []string, err error) loadResult {
	return loadResult{content, chosen, size, warnings, err}
}

type loadResult struct {
	Content  string
	Chosen   string
	Size     int
	Warnings []string
	Err      error
}
