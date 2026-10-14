package session

import (
	"path/filepath"
	"strings"
	"testing"
)

// FuzzSessionPathContained verifies that generated session paths stay inside
// the state root and never escape through "..".
func FuzzSessionPathContained(f *testing.F) {
	f.Add("/home/alice/.local/share/kirsch", "/home/alice/kirsch", "20261014T120000-01ARZ3NDEKTSV4RRFFQ69G5FAV")
	f.Add("/tmp/root", ".", "id")
	f.Add("/tmp/root", "../../etc", "id")

	f.Fuzz(func(t *testing.T, root, workspace, id string) {
		if root == "" {
			t.Skip("empty root")
		}
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			t.Fatalf("Abs: %v", err)
		}

		ws := workspace
		if canonical, err := Canonicalize(workspace); err == nil {
			ws = canonical
		}

		path := SessionPath(rootAbs, ws, id)
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			t.Fatalf("Rel: %v", err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			t.Fatalf("path escapes root: root=%q path=%q rel=%q", rootAbs, path, rel)
		}
	})
}
