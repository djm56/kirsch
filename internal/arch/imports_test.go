// Package arch holds the architecture tests.
//
// Plan §2's dependency rules are enforced here rather than by code review, and
// they bind a guarded package's test files (in-package and external) as well as
// its own. The value is entirely in the timing: the day internal/provider first
// appears is precisely the day nobody is thinking about import rules, and this
// test is already in place to catch it.
package arch

import (
	"os/exec"
	"strings"
	"testing"
)

const modulePrefix = "github.com/djm56/kirsch/"

// forbidden lists the edges plan §2 and architecture.md §3 rule out.
//
// architecture.md is stricter than the plan for internal/agent — it adds policy
// and session — and architecture.md wins on structure, so the stricter form is
// what is enforced.
var forbidden = map[string][]string{
	"internal/agent": {
		"internal/tui",
		"internal/provider",
		"internal/tool",
		"internal/policy",
		"internal/session",
	},
	"internal/agent/prompt": {
		"internal/tui",
		"internal/provider",
		"internal/tool",
		"internal/policy",
		"internal/session",
	},
	"internal/tui": {
		"internal/provider",
		"internal/tool",
		"internal/workspace",
		"internal/agent",
	},
	// Doctor only reports. The rule forbids the UI, the app and the agent loop,
	// and leaves session, tool and provider open for its probes. Because the
	// match is by prefix, "internal/agent" also covers internal/agent/prompt;
	// doctor's one exception, listed in allowed below, is internal/agent/prompt
	// itself (the project-context loader). internal/agent and every other
	// package under it stay forbidden.
	"internal/doctor": {
		"internal/tui",
		"internal/app",
		"internal/agent",
	},
	// The clipboard package is a leaf: it imports no internal package at all, so
	// the one prefix "internal" bans every package of ours (architecture.md §3
	// rule 6).
	"internal/clipboard": {
		"internal",
	},
	"internal/session": {
		"internal/tui",
		"internal/provider",
		"internal/tool",
		"internal/policy",
		"internal/workspace",
	},
}

// allowed lists, per guarded package, imports that a prefix in forbidden would
// otherwise ban. Each entry is matched exactly: allowing internal/agent/prompt
// does not allow internal/agent or any other package under it.
var allowed = map[string][]string{
	"internal/doctor": {"internal/agent/prompt"},
}

// isAllowed reports whether pkg may import dep despite a forbidden prefix.
func isAllowed(pkg, dep string) bool {
	for _, a := range allowed[pkg] {
		if dep == a {
			return true
		}
	}
	return false
}

// importSource names where in a package an import was found, for the error
// message: the package's own files, its in-package tests (package x, _test.go)
// or its external tests (package x_test).
type importSource string

const (
	fromPackage       importSource = "the package"
	fromInPackageTest importSource = "its in-package tests"
	fromExternalTest  importSource = "its external tests"
)

// TestImportRules fails if any forbidden edge exists.
//
// It runs go list over the module and checks, for each package the forbidden
// table guards, the imports of the package, of its in-package tests
// (TestImports) and of its external tests (XTestImports), against the same
// table and the same allowed mechanism. An entry whose package is missing is
// skipped. The test fails when no guarded package is inspected at all, and when
// the total count of in-package test imports across the guarded packages is
// zero: go list reports in-package test imports for the guarded packages in
// this tree, so a zero total means the TestImports field is not being read.
func TestImportRules(t *testing.T) {
	// "|" cannot occur in an import path, so it separates the four lists safely.
	out, err := exec.Command("go", "list", "-deps=false", "-f",
		"{{.ImportPath}}|{{join .Imports \" \"}}|{{join .TestImports \" \"}}|{{join .XTestImports \" \"}}",
		"../...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	checked, testImports := 0, 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Split(line, "|")
		if len(parts) != 4 {
			t.Fatalf("go list line has %d |-separated fields, want 4: %q", len(parts), line)
		}
		pkg := strings.TrimPrefix(strings.TrimSpace(parts[0]), modulePrefix)
		banned, guarded := forbidden[pkg]
		if !guarded {
			continue
		}
		checked++
		testImports += len(strings.Fields(parts[2]))
		for _, src := range []struct {
			from importSource
			imps string
		}{
			{fromPackage, parts[1]},
			{fromInPackageTest, parts[2]},
			{fromExternalTest, parts[3]},
		} {
			for _, imp := range strings.Fields(src.imps) {
				dep := strings.TrimPrefix(imp, modulePrefix)
				if dep == imp {
					continue // not one of ours
				}
				if isAllowed(pkg, dep) {
					continue
				}
				for _, b := range banned {
					if dep == b || strings.HasPrefix(dep, b+"/") {
						t.Errorf("illegal import: %s must not import %s, found in %s "+
							"(rule %q in the forbidden table, internal/arch/imports_test.go)",
							pkg, dep, src.from, b)
					}
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no guarded package was inspected; the go list invocation is probably wrong")
	}
	if testImports == 0 {
		t.Fatal("no in-package test import of a guarded package was inspected; the go list template is probably wrong")
	}
	t.Logf("checked %d guarded package(s) of %d rules", checked, len(forbidden))
}

// TestAllowedIsExact pins the one exception to the doctor rule: doctor may
// import internal/agent/prompt and nothing else under internal/agent, and no
// other guarded package gains the exception. Called directly, because the
// subpackages that must stay forbidden do not exist to import.
func TestAllowedIsExact(t *testing.T) {
	for _, tc := range []struct {
		pkg, dep string
		want     bool
	}{
		{"internal/doctor", "internal/agent/prompt", true},
		{"internal/doctor", "internal/agent", false},
		{"internal/doctor", "internal/agent/prompt/sub", false},
		{"internal/doctor", "internal/agent/other", false},
		{"internal/doctor", "internal/agent/promptx", false},
		{"internal/tui", "internal/agent/prompt", false},
		{"internal/session", "internal/agent/prompt", false},
	} {
		if got := isAllowed(tc.pkg, tc.dep); got != tc.want {
			t.Errorf("isAllowed(%q, %q) = %v, want %v", tc.pkg, tc.dep, got, tc.want)
		}
	}
}

// TestGuardedPackagesAreReal keeps part of the rule table honest: a typo in the
// package name of one of the entries it names would make TestImportRules skip
// that entry and pass.
func TestGuardedPackagesAreReal(t *testing.T) {
	out, err := exec.Command("go", "list", "../...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	exists := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		exists[strings.TrimPrefix(line, modulePrefix)] = true
	}
	// The packages below must exist. TestImportRules skips a table
	// entry whose package is missing, so a typo in one of these names would
	// pass silently. The other entries are not checked here.
	if !exists["internal/tui"] {
		t.Error("internal/tui not found; the rule table or the module path is wrong")
	}
	if !exists["internal/clipboard"] {
		t.Error("internal/clipboard not found; the rule table or the module path is wrong")
	}
	if !exists["internal/doctor"] {
		t.Error("internal/doctor not found; the rule table or the module path is wrong")
	}
}
