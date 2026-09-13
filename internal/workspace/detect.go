package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectType is a detected project kind.
type ProjectType string

// The four kinds v0.1 recognises.
const (
	TypeWordPressPlugin ProjectType = "wordpress-plugin"
	TypeWordPressTheme  ProjectType = "wordpress-theme"
	TypeGo              ProjectType = "go"
	TypeNode            ProjectType = "node"
)

// detectHeaderBytes bounds how much of a PHP file is read looking for a plugin
// header. WordPress itself reads 8KB; matching that keeps detection cheap and
// predictable on a directory full of large PHP files.
const detectHeaderBytes = 8 << 10

// DetectTypes returns the project kinds found at the workspace root, primary
// first.
//
// Root level only, which bounds the cost on a large repository. Multiple
// matches are normal and are all reported — a WordPress plugin with a
// package.json for its build step is two true things, not an ambiguity to
// resolve. The order is fixed so callers and tests can rely on it.
func DetectTypes(root string) []ProjectType {
	var out []ProjectType
	add := func(t ProjectType) {
		for _, existing := range out {
			if existing == t {
				return
			}
		}
		out = append(out, t)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(entries))
	var phpFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names[e.Name()] = true
		if strings.EqualFold(filepath.Ext(e.Name()), ".php") {
			phpFiles = append(phpFiles, e.Name())
		}
	}

	// 1. WordPress plugin: a `Plugin Name:` header in any root-level PHP file.
	for _, name := range phpFiles {
		if headerContains(filepath.Join(root, name), "Plugin Name:") {
			add(TypeWordPressPlugin)
			break
		}
	}
	// 2. WordPress theme: `Theme Name:` in root style.css.
	if names["style.css"] && headerContains(filepath.Join(root, "style.css"), "Theme Name:") {
		add(TypeWordPressTheme)
	}
	// 3. Go.
	if names["go.mod"] {
		add(TypeGo)
	}
	// 4. Node.
	if names["package.json"] && isJSONObject(filepath.Join(root, "package.json")) {
		add(TypeNode)
	}
	return out
}

// headerContains reports whether the first detectHeaderBytes of a file contain
// needle.
func headerContains(path, needle string) bool {
	// Called only with a path built from the canonical workspace root plus a
	// root-level entry name from os.ReadDir — never with caller input. gosec G304.
	f, err := os.Open(path) // #nosec G304 -- root-relative, not caller-supplied
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, detectHeaderBytes)
	n, _ := f.Read(buf)
	return strings.Contains(string(buf[:n]), needle)
}

// isJSONObject guards against a package.json that is present but unparseable,
// which should not count as a Node project.
func isJSONObject(path string) bool {
	// Root-level path built from the canonical root plus a literal filename.
	// gosec G304.
	body, err := os.ReadFile(path) // #nosec G304 -- root-relative, not caller-supplied
	if err != nil {
		return false
	}
	var v map[string]any
	return json.Unmarshal(body, &v) == nil
}

// Types returns the workspace's detected project types.
func (w *Workspace) Types() []ProjectType { return DetectTypes(w.CanonicalRoot) }
