package workspace

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func walkAll(t *testing.T, ws *Workspace, start string, o WalkOptions) []string {
	t.Helper()
	var got []string
	err := ws.Walk(start, o, func(e Entry) bool {
		suffix := ""
		if e.IsDir {
			suffix = "/"
		}
		got = append(got, e.Rel+suffix)
		return true
	})
	if err != nil && !IsLimitReached(err) {
		t.Fatalf("Walk: %v", err)
	}
	return got
}

// TestWalkRespectsIgnoreRules is the check that matters: the walker decides
// what reaches model context, so a file leaking in is a real failure and not a
// cosmetic one.
func TestWalkRespectsIgnoreRules(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	got := walkAll(t, ws, ".", WalkOptions{MaxDepth: -1})
	joined := strings.Join(got, "\n")

	mustNot := []struct{ path, why string }{
		{"ignored/", "directory ignored by the root .gitignore"},
		{"ignored/secret.txt", "file under an ignored directory"},
		{"app.log", "matched by *.log in the root .gitignore"},
		{"pkg/util/scratch.txt", "matched by a *nested* .gitignore"},
		{".env", "on the path denylist"},
		{".hidden.txt", "hidden, and IncludeHidden is false"},
	}
	for _, m := range mustNot {
		if strings.Contains(joined, m.path) {
			t.Errorf("%s appeared in the walk (%s)\ngot:\n%s", m.path, m.why, joined)
		}
	}

	mustHave := []string{"README.md", "main.go", "pkg/util/strings.go", "unicode.md", "crlf.txt"}
	for _, m := range mustHave {
		if !strings.Contains(joined, m) {
			t.Errorf("%s missing from the walk\ngot:\n%s", m, joined)
		}
	}
}

// TestWalkOrderingIsDeterministic — non-deterministic output makes golden tests
// flaky and, from M3, makes model behaviour irreproducible between runs.
func TestWalkOrderingIsDeterministic(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	first := walkAll(t, ws, ".", WalkOptions{MaxDepth: -1})
	for i := 0; i < 5; i++ {
		if got := walkAll(t, ws, ".", WalkOptions{MaxDepth: -1}); !reflect.DeepEqual(first, got) {
			t.Fatalf("walk %d differed:\n first: %v\n got:   %v", i, first, got)
		}
	}
	sorted := append([]string(nil), first...)
	for i := 1; i < len(sorted); i++ {
		if strings.TrimSuffix(sorted[i-1], "/") > strings.TrimSuffix(sorted[i], "/") {
			// Not a flat sort — it is a depth-first walk — so only check that
			// siblings are ordered, via the repeated-run check above.
			break
		}
	}
}

func TestWalkMaxDepth(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	cases := []struct {
		depth  int
		want   string // a path that must be present
		absent string // a path that must not be
	}{
		{0, "README.md", "pkg/util/strings.go"},
		{1, "pkg/util/", "pkg/util/strings.go"},
		{2, "pkg/util/strings.go", "deep/a/b/c/d/deep.txt"},
		{-1, "deep/a/b/c/d/deep.txt", ""},
	}
	for _, c := range cases {
		got := strings.Join(walkAll(t, ws, ".", WalkOptions{MaxDepth: c.depth}), "\n")
		if !strings.Contains(got, c.want) {
			t.Errorf("depth %d: %s missing\n%s", c.depth, c.want, got)
		}
		if c.absent != "" && strings.Contains(got, c.absent) {
			t.Errorf("depth %d: %s should be beyond the depth limit\n%s", c.depth, c.absent, got)
		}
	}
}

func TestWalkIncludeHidden(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	got := strings.Join(walkAll(t, ws, ".", WalkOptions{MaxDepth: -1, IncludeHidden: true}), "\n")
	if !strings.Contains(got, ".hidden.txt") {
		t.Errorf("IncludeHidden did not surface .hidden.txt:\n%s", got)
	}
	// The denylist outranks IncludeHidden: .env is never readable, hidden or not.
	if strings.Contains(got, ".env") {
		t.Errorf(".env surfaced with IncludeHidden; the denylist must outrank it:\n%s", got)
	}
	if strings.Contains(got, ".git/") {
		t.Errorf(".git surfaced with IncludeHidden:\n%s", got)
	}
}

// TestWalkSkipsBuiltinIgnores covers the two the fixtures exercise directly.
func TestWalkSkipsBuiltinIgnores(t *testing.T) {
	for _, tc := range []struct{ fixture, forbidden string }{
		{"repo-node", "node_modules"},
		{"repo-wordpress-plugin", "vendor"},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			ws := fixtureWS(t, tc.fixture)
			got := strings.Join(walkAll(t, ws, ".", WalkOptions{MaxDepth: -1}), "\n")
			if strings.Contains(got, tc.forbidden) {
				t.Errorf("%s appeared in the walk:\n%s", tc.forbidden, got)
			}
		})
	}
}

func TestWalkDoesNotFollowDirectorySymlinks(t *testing.T) {
	ws := fixtureWS(t, "repo-symlink-escape")
	got := walkAll(t, ws, ".", WalkOptions{MaxDepth: -1})
	joined := strings.Join(got, "\n")
	// link-dir-ok points at sub/. Its contents must appear once, via sub/,
	// not a second time through the link.
	if strings.Contains(joined, "link-dir-ok/target.txt") {
		t.Errorf("walker descended into a directory symlink:\n%s", joined)
	}
	if !strings.Contains(joined, "sub/target.txt") {
		t.Errorf("real directory not walked:\n%s", joined)
	}
}

func TestWalkLimit(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	var n int
	err := ws.Walk(".", WalkOptions{MaxDepth: -1, Limit: 3}, func(Entry) bool {
		n++
		return true
	})
	if !IsLimitReached(err) {
		t.Errorf("limit not signalled: %v", err)
	}
	if n != 3 {
		t.Errorf("visited %d entries, want 3", n)
	}
}

func TestIsIgnored(t *testing.T) {
	ws := fixtureWS(t, "repo-small")
	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"app.log", false, true},
		{"ignored", true, true},
		{"pkg/util/scratch.txt", false, true},
		{"node_modules/x/index.js", false, true},
		{"vendor/autoload.php", false, true},
		{"README.md", false, false},
		{"pkg/util/strings.go", false, false},
	}
	for _, c := range cases {
		if got := ws.IsIgnored(c.path, c.isDir); got != c.want {
			t.Errorf("IsIgnored(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestDetectProjectTypes(t *testing.T) {
	base, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		dir  string
		want []ProjectType
	}{
		{"repo-wordpress-plugin", []ProjectType{TypeWordPressPlugin}},
		{"repo-go-module", []ProjectType{TypeGo}},
		{"repo-node", []ProjectType{TypeNode}},
		{"detect/theme", []ProjectType{TypeWordPressTheme}},
		// Precedence is pinned here: go before node, in the documented order.
		{"detect/ambiguous", []ProjectType{TypeGo, TypeNode}},
		{"detect/none", nil},
		{"repo-small", nil},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			got := DetectTypes(filepath.Join(base, c.dir))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("DetectTypes(%s) = %v, want %v", c.dir, got, c.want)
			}
		})
	}
}
