// Package arch holds the architecture tests.
//
// Plan §2's dependency rules are enforced here rather than by code review. The
// value is entirely in the timing: the day internal/provider first appears is
// precisely the day nobody is thinking about import rules, and this test is
// already in place to catch it.
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
	"internal/tui": {
		"internal/provider",
		"internal/tool",
		"internal/workspace",
		"internal/agent",
	},
}

// TestImportRules fails if any forbidden edge exists.
//
// It passes vacuously today for agent and provider, which do not exist yet.
// That is the point: a test that only starts working once the package it
// guards is written has already missed its moment.
func TestImportRules(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps=false",
		"-f", "{{.ImportPath}} {{join .Imports \" \"}}", "../...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	checked := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := strings.TrimPrefix(fields[0], modulePrefix)
		banned, guarded := forbidden[pkg]
		if !guarded {
			continue
		}
		checked++
		for _, imp := range fields[1:] {
			dep := strings.TrimPrefix(imp, modulePrefix)
			if dep == imp {
				continue // not one of ours
			}
			for _, b := range banned {
				if dep == b || strings.HasPrefix(dep, b+"/") {
					t.Errorf("illegal import: %s must not import %s\n\n"+
						"plan §2 / architecture.md §3. %s receives typed messages and "+
						"emits typed intents; it never reaches into an implementation "+
						"package directly.", pkg, dep, pkg)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no guarded package was inspected; the go list invocation is probably wrong")
	}
	t.Logf("checked %d guarded package(s) of %d rules", checked, len(forbidden))
}

// TestGuardedPackagesAreReal keeps the rule table honest: a typo in a package
// name would make TestImportRules pass by never matching anything.
func TestGuardedPackagesAreReal(t *testing.T) {
	out, err := exec.Command("go", "list", "../...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	exists := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		exists[strings.TrimPrefix(line, modulePrefix)] = true
	}
	// Only packages that exist today need to be present; the rest are
	// deliberately pre-registered for later milestones.
	if !exists["internal/tui"] {
		t.Error("internal/tui not found; the rule table or the module path is wrong")
	}
}
