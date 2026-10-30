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
			if isAllowed(pkg, dep) {
				continue
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
	if !exists["internal/doctor"] {
		t.Error("internal/doctor not found; the rule table or the module path is wrong")
	}
}
