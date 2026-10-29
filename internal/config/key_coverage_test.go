package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// moduleRoot walks up from the test's working directory to the directory
// holding go.mod and returns it.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod found above the test's working directory")
		}
		dir = parent
	}
}

// documentedKeys derives, from the code, every name doc/configuration.md must
// list. It returns the section names ("provider", "policy", ...) and the full
// dotted key names, with endpoint fields written as "provider.<name>.<field>".
//
// Two sources, because the code keeps keys in two shapes:
//   - reflect over Config: each field's `toml` tag is a section, and each
//     tagged field of that section's struct is a key. This covers every section
//     except provider, whose struct carries no tags.
//   - go/ast over config.go: ProviderConfig is decoded by hand, so its keys
//     exist only as string literals in ProviderConfig.UnmarshalTOML. The
//     `case "…":` labels are the endpoint fields; the `key == "…"` comparison
//     is the top-level key ("default"). reflect cannot see either.
func documentedKeys(t *testing.T) (sections, keys []string) {
	t.Helper()

	ct := reflect.TypeOf(Config{})
	for i := 0; i < ct.NumField(); i++ {
		section := ct.Field(i).Tag.Get("toml")
		if section == "" {
			continue
		}
		sections = append(sections, section)
		st := ct.Field(i).Type
		for j := 0; j < st.NumField(); j++ {
			if k := st.Field(j).Tag.Get("toml"); k != "" {
				keys = append(keys, section+"."+k)
			}
		}
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "config.go", nil, 0)
	if err != nil {
		t.Fatalf("parse config.go: %v", err)
	}
	unquote := func(e ast.Expr) (string, bool) {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(lit.Value)
		return s, err == nil
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "UnmarshalTOML" || fd.Recv == nil || fd.Body == nil {
			continue
		}
		star, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
		if !ok || star.X.(*ast.Ident).Name != "ProviderConfig" {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CaseClause:
				for _, e := range n.List {
					if s, ok := unquote(e); ok {
						keys = append(keys, "provider.<name>."+s)
					}
				}
			case *ast.BinaryExpr:
				if id, ok := n.X.(*ast.Ident); ok && id.Name == "key" && n.Op == token.EQL {
					if s, ok := unquote(n.Y); ok {
						keys = append(keys, "provider."+s)
					}
				}
			}
			return true
		})
	}
	return sections, keys
}

// missingFromPage returns every section and key that page does not document.
//
// "Appears on the page" means the exact forms the page uses, not a substring:
//   - a section needs a heading line "### `[<section>]`";
//   - a key needs a table row whose first cell is the backticked dotted name,
//     i.e. a line starting "| `<key>` |".
//
// Prose mentions of a key in backticks elsewhere do not count, so a key cannot
// pass by being named in a paragraph while its reference row is gone.
func missingFromPage(page string, sections, keys []string) []string {
	lines := strings.Split(page, "\n")
	has := func(match func(line string) bool) bool {
		for _, l := range lines {
			if match(l) {
				return true
			}
		}
		return false
	}
	var missing []string
	for _, s := range sections {
		heading := "### `[" + s + "]`"
		if !has(func(l string) bool { return strings.TrimSpace(l) == heading }) {
			missing = append(missing, "section "+heading)
		}
	}
	for _, k := range keys {
		row := "| `" + k + "` |"
		if !has(func(l string) bool { return strings.HasPrefix(l, row) }) {
			missing = append(missing, "key "+k)
		}
	}
	return missing
}

// TestKeyCoverage_DerivedListsAreSane is the positive control for the
// extraction: a broken derivation that finds nothing would otherwise let
// TestKeyCoverage_PageDocumentsEveryKey pass vacuously. It also checks the
// matcher itself: an empty page misses everything, and a prose-only mention
// does not count as documenting a key.
func TestKeyCoverage_DerivedListsAreSane(t *testing.T) {
	sections, keys := documentedKeys(t)
	if len(sections) == 0 || len(keys) == 0 {
		t.Fatalf("empty extraction: sections=%v keys=%v", sections, keys)
	}
	contains := func(list []string, want string) bool {
		for _, s := range list {
			if s == want {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"policy", "telemetry"} {
		if !contains(sections, want) {
			t.Errorf("sections %v lack %q", sections, want)
		}
	}
	// One key from each extraction route: struct tag, UnmarshalTOML case label,
	// UnmarshalTOML key comparison.
	for _, want := range []string{"policy.env_passthrough", "provider.<name>.base_url", "provider.default"} {
		if !contains(keys, want) {
			t.Errorf("keys %v lack %q", keys, want)
		}
	}

	// Matcher controls: an empty page misses every item; a page that only
	// mentions a key in prose still misses it.
	if got := missingFromPage("", sections, keys); len(got) != len(sections)+len(keys) {
		t.Errorf("empty page: %d missing, want %d", len(got), len(sections)+len(keys))
	}
	prose := "The `policy.env_passthrough` key is described elsewhere.\n"
	if got := missingFromPage(prose, nil, []string{"policy.env_passthrough"}); len(got) != 1 {
		t.Errorf("prose-only mention was accepted as documentation: missing=%v", got)
	}
	row := "| `policy.env_passthrough` | `[]` |\n"
	if got := missingFromPage(row, nil, []string{"policy.env_passthrough"}); len(got) != 0 {
		t.Errorf("table row was not accepted: missing=%v", got)
	}
}

// TestKeyCoverage_PageDocumentsEveryKey fails, naming every one, for each
// config section and key in the code that doc/configuration.md does not list
// (see missingFromPage for what counts as listed). The page is read from the
// module root, found by walking up from the working directory to go.mod.
func TestKeyCoverage_PageDocumentsEveryKey(t *testing.T) {
	sections, keys := documentedKeys(t)
	data, err := os.ReadFile(filepath.Join(moduleRoot(t), "doc", "configuration.md"))
	if err != nil {
		t.Fatalf("read doc/configuration.md: %v", err)
	}
	if missing := missingFromPage(string(data), sections, keys); len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("doc/configuration.md does not document:\n  %s", strings.Join(missing, "\n  "))
	}
}
