package policy

import (
	"testing"
)

// TestPrefixMatching verifies that allowlist patterns match exact argv
// correctly, and extended calls require approval (now that defaults are exact-match).
// Tests the default allowlist against various argv calls.
func TestPrefixMatching(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		expected Decision
	}{
		// Exact matches against default allowlist entries (extendable=false)
		{
			name:     "exact match go test",
			argv:     []string{"go", "test"},
			expected: DecisionAllow,
		},
		{
			name:     "exact match go build",
			argv:     []string{"go", "build"},
			expected: DecisionAllow,
		},
		// Extended calls now ask (since defaults are exact-match)
		{
			name:     "extended call go test .../... asks",
			argv:     []string{"go", "test", "./..."},
			expected: DecisionAskUser,
		},
		{
			name:     "extended call git diff HEAD~1 asks",
			argv:     []string{"git", "diff", "HEAD~1"},
			expected: DecisionAskUser,
		},
		{
			name:     "extended call go build -v asks",
			argv:     []string{"go", "build", "-v"},
			expected: DecisionAskUser,
		},
		// Near-misses that must NOT match
		{
			name:     "partial match go testfoo rejected",
			argv:     []string{"go", "testfoo"},
			expected: DecisionAskUser,
		},
		{
			name:     "partial match go buildfail rejected",
			argv:     []string{"go", "buildfail"},
			expected: DecisionAskUser,
		},
		{
			name:     "partial match gotk rejected",
			argv:     []string{"gotk"},
			expected: DecisionAskUser,
		},
		// Argv split property: semicolon in argv does not bypass matching
		{
			name:     "argv split does not bypass matcher go+semicolon+rm",
			argv:     []string{"go test; rm -rf /"},
			expected: DecisionAskUser,
		},
		{
			name:     "argv with pipes is not split",
			argv:     []string{"go test | cat"},
			expected: DecisionAskUser,
		},
		// Unknown commands
		{
			name:     "unknown command curl",
			argv:     []string{"curl"},
			expected: DecisionAskUser,
		},
		{
			name:     "unknown command docker",
			argv:     []string{"docker"},
			expected: DecisionAskUser,
		},
		// Git near-misses (git diff exists, git checkout does not)
		{
			name:     "git log exact matches",
			argv:     []string{"git", "log"},
			expected: DecisionAllow,
		},
		{
			name:     "git checkout rejected",
			argv:     []string{"git", "checkout"},
			expected: DecisionAskUser,
		},
		{
			name:     "git commit rejected",
			argv:     []string{"git", "commit"},
			expected: DecisionAskUser,
		},
		// ls cases (now exact-match)
		{
			name:     "ls -la asks (extended)",
			argv:     []string{"ls", "-la"},
			expected: DecisionAskUser,
		},
		{
			name:     "ls / asks (extended)",
			argv:     []string{"ls", "/"},
			expected: DecisionAskUser,
		},
		{
			name:     "ls exact matches",
			argv:     []string{"ls"},
			expected: DecisionAllow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			got, _ := p.ForCommand(tc.argv)
			if got != tc.expected {
				t.Errorf("ForCommand(%v) = %v, want %v", tc.argv, got, tc.expected)
			}
		})
	}
}

// TestShellsCanNeverBeAllowlisted verifies that shells and shell runners
// always require approval, specifically DecisionAskUser, never DecisionAllow.
func TestShellsCanNeverBeAllowlisted(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		// Direct shell invocations
		{
			name: "sh",
			argv: []string{"sh"},
		},
		{
			name: "bash",
			argv: []string{"bash"},
		},
		{
			name: "zsh",
			argv: []string{"zsh"},
		},
		{
			name: "dash",
			argv: []string{"dash"},
		},
		// Runners that execute other commands
		{
			name: "env",
			argv: []string{"env"},
		},
		{
			name: "xargs",
			argv: []string{"xargs"},
		},
		{
			name: "nohup",
			argv: []string{"nohup"},
		},
		// Evasion attempts: absolute paths
		{
			name: "evasion /bin/sh",
			argv: []string{"/bin/sh"},
		},
		{
			name: "evasion /bin/bash",
			argv: []string{"/bin/bash"},
		},
		{
			name: "evasion /usr/bin/env",
			argv: []string{"/usr/bin/env"},
		},
		{
			name: "evasion /bin/env",
			argv: []string{"/bin/env"},
		},
		// Evasion attempts: via env
		{
			name: "evasion /usr/bin/env sh",
			argv: []string{"/usr/bin/env", "sh"},
		},
		{
			name: "evasion /usr/bin/env bash",
			argv: []string{"/usr/bin/env", "bash"},
		},
		{
			name: "evasion /bin/env sh",
			argv: []string{"/bin/env", "sh"},
		},
		// Evasion attempts: with flags
		{
			name: "evasion bash -lc",
			argv: []string{"bash", "-lc"},
		},
		{
			name: "evasion bash -c",
			argv: []string{"bash", "-c"},
		},
		{
			name: "evasion bash -ic",
			argv: []string{"bash", "-ic"},
		},
		{
			name: "evasion sh -c",
			argv: []string{"sh", "-c"},
		},
		{
			name: "evasion sh -lc",
			argv: []string{"sh", "-lc"},
		},
		{
			name: "evasion zsh -c",
			argv: []string{"zsh", "-c"},
		},
		{
			name: "evasion /bin/bash -c",
			argv: []string{"/bin/bash", "-c"},
		},
		{
			name: "evasion /bin/sh -c",
			argv: []string{"/bin/sh", "-c"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			got, _ := p.ForCommand(tc.argv)
			if got != DecisionAskUser {
				t.Errorf("ForCommand(%v) = %v, want DecisionAskUser (shells must always require approval)", tc.argv, got)
			}
		})
	}
}

// TestForPatchNeverAllows verifies that ForPatch never returns DecisionAllow,
// per the specification's principle #1. Patches always require approval.
func TestForPatchNeverAllows(t *testing.T) {
	cases := []struct {
		name  string
		files []string
	}{
		{
			name:  "ordinary path",
			files: []string{"main.go"},
		},
		{
			name:  "multiple files",
			files: []string{"main.go", "util.go", "config.go"},
		},
		{
			name:  "nested path",
			files: []string{"internal/app/app.go"},
		},
		{
			name:  "single file root",
			files: []string{"README.md"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			got, _ := p.ForPatch(tc.files)
			if got == DecisionAllow {
				t.Errorf("ForPatch(%v) = DecisionAllow, but patches must never be auto-allowed", tc.files)
			}
		})
	}
}

// TestForPatchNoConfigurationAllows verifies that ForPatch never returns
// DecisionAllow across various file inputs and settings. This is a binding
// principle (#1) — no configuration can make ForPatch auto-allow patches.
func TestForPatchNoConfigurationAllows(t *testing.T) {
	cases := []struct {
		name                     string
		files                    []string
		allowSessionScopedGrants bool
		setupGrants              bool
	}{
		// Empty and basic file cases
		{
			name:                     "empty files list, grants disabled",
			files:                    []string{},
			allowSessionScopedGrants: false,
			setupGrants:              false,
		},
		{
			name:                     "single file, grants disabled",
			files:                    []string{"README.md"},
			allowSessionScopedGrants: false,
			setupGrants:              false,
		},
		{
			name:                     "multiple files, grants disabled",
			files:                    []string{"main.go", "util.go"},
			allowSessionScopedGrants: false,
			setupGrants:              false,
		},
		{
			name:                     "nested path, grants disabled",
			files:                    []string{"internal/app/app.go"},
			allowSessionScopedGrants: false,
			setupGrants:              false,
		},
		// Same cases with grants enabled (but no grants recorded)
		{
			name:                     "empty files list, grants enabled",
			files:                    []string{},
			allowSessionScopedGrants: true,
			setupGrants:              false,
		},
		{
			name:                     "single file, grants enabled",
			files:                    []string{"README.md"},
			allowSessionScopedGrants: true,
			setupGrants:              false,
		},
		// Same cases with grants enabled AND grants recorded
		{
			name:                     "single file, grants enabled with active grant",
			files:                    []string{"README.md"},
			allowSessionScopedGrants: true,
			setupGrants:              true,
		},
		{
			name:                     "multiple files, grants enabled with active grant",
			files:                    []string{"main.go", "util.go"},
			allowSessionScopedGrants: true,
			setupGrants:              true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			// Set the grants configuration flag
			p.allowSessionScopedGrants = tc.allowSessionScopedGrants
			// If requested, set up some grants (even though ForPatch should ignore them)
			if tc.setupGrants && tc.allowSessionScopedGrants {
				p.Grant([]string{"go", "test"})
				p.Grant([]string{"git", "diff"})
			}
			got, _ := p.ForPatch(tc.files)
			if got == DecisionAllow {
				t.Errorf("ForPatch(%v) with grants=%v, active=%v = DecisionAllow, but ForPatch must never allow",
					tc.files, tc.allowSessionScopedGrants, tc.setupGrants)
			}
		})
	}
}

// TestGrantRefusesInvalidPrefixes verifies that Grant refuses bare wildcards,
// empty prefixes, and shells (including all evasion forms).
func TestGrantRefusesInvalidPrefixes(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		{
			name: "bare wildcard",
			argv: []string{"*"},
		},
		{
			name: "empty argv list",
			argv: []string{},
		},
		// Direct shells
		{
			name: "shell sh",
			argv: []string{"sh"},
		},
		{
			name: "shell bash",
			argv: []string{"bash"},
		},
		{
			name: "shell zsh",
			argv: []string{"zsh"},
		},
		{
			name: "shell dash",
			argv: []string{"dash"},
		},
		// Absolute paths to shells
		{
			name: "shell /bin/sh",
			argv: []string{"/bin/sh"},
		},
		{
			name: "shell /bin/bash",
			argv: []string{"/bin/bash"},
		},
		// Shell runners
		{
			name: "runner env",
			argv: []string{"env"},
		},
		{
			name: "runner xargs",
			argv: []string{"xargs"},
		},
		{
			name: "runner nohup",
			argv: []string{"nohup"},
		},
		{
			name: "runner /usr/bin/env",
			argv: []string{"/usr/bin/env"},
		},
		// Evasion: env with shell
		{
			name: "evasion env sh",
			argv: []string{"env", "sh"},
		},
		{
			name: "evasion /usr/bin/env bash",
			argv: []string{"/usr/bin/env", "bash"},
		},
		// Evasion: shell with flags
		{
			name: "evasion bash -c",
			argv: []string{"bash", "-c"},
		},
		{
			name: "evasion sh -lc",
			argv: []string{"sh", "-lc"},
		},
		{
			name: "evasion /bin/bash -c",
			argv: []string{"/bin/bash", "-c"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			err := p.Grant(tc.argv)
			if err == nil {
				t.Errorf("Grant(%v) = nil, want an error for invalid prefix", tc.argv)
			}
		})
	}
}

// TestGrantAcceptsValidPrefixes verifies that Grant accepts legitimate command
// prefixes and stores them correctly.
func TestGrantAcceptsValidPrefixes(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		{
			name: "simple command go test",
			argv: []string{"go", "test"},
		},
		{
			name: "single element go",
			argv: []string{"go"},
		},
		{
			name: "git command git diff",
			argv: []string{"git", "diff"},
		},
		{
			name: "ls with flags ls -la",
			argv: []string{"ls", "-la"},
		},
		{
			name: "unknown command curl",
			argv: []string{"curl"},
		},
		{
			name: "docker with subcommand",
			argv: []string{"docker", "run"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			err := p.Grant(tc.argv)
			if err != nil {
				t.Errorf("Grant(%v) = %v, want nil for valid prefix", tc.argv, err)
			}
		})
	}
}

// TestGrantSemantics verifies that a grant for a prefix matches extended calls
// with that prefix, but not partial-word matches. Grants are always extendable.
func TestGrantSemantics(t *testing.T) {
	cases := []struct {
		name          string
		grantedPrefix []string
		testCall      []string
		shouldMatch   bool
		description   string
	}{
		{
			name:          "grant go test matches go test ./...",
			grantedPrefix: []string{"go", "test"},
			testCall:      []string{"go", "test", "./..."},
			shouldMatch:   true,
			description:   "extended call with prefix grant should match",
		},
		{
			name:          "grant go test does not match go testfoo",
			grantedPrefix: []string{"go", "test"},
			testCall:      []string{"go", "testfoo"},
			shouldMatch:   false,
			description:   "partial word match must not succeed",
		},
		{
			name:          "grant git diff matches git diff HEAD~1",
			grantedPrefix: []string{"git", "diff"},
			testCall:      []string{"git", "diff", "HEAD~1"},
			shouldMatch:   true,
			description:   "extended call with more arguments should match",
		},
		{
			name:          "grant git diff does not match git checkout",
			grantedPrefix: []string{"git", "diff"},
			testCall:      []string{"git", "checkout"},
			shouldMatch:   false,
			description:   "different subcommand should not match",
		},
		{
			name:          "grant go does not match git",
			grantedPrefix: []string{"go"},
			testCall:      []string{"git"},
			shouldMatch:   false,
			description:   "different command should not match",
		},
		{
			name:          "grant go matches go test",
			grantedPrefix: []string{"go"},
			testCall:      []string{"go", "test"},
			shouldMatch:   true,
			description:   "grant for command root matches subcommands",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			// Record the grant
			err := p.Grant(tc.grantedPrefix)
			if err != nil {
				t.Fatalf("Grant(%v) failed: %v", tc.grantedPrefix, err)
			}

			// Check if the test call matches the grant
			got, _ := p.ForCommand(tc.testCall)
			matches := got == DecisionAllow

			if matches != tc.shouldMatch {
				if tc.shouldMatch {
					t.Errorf("ForCommand(%v) after Grant(%v) = %v, want DecisionAllow. %s",
						tc.testCall, tc.grantedPrefix, got, tc.description)
				} else {
					t.Errorf("ForCommand(%v) after Grant(%v) = DecisionAllow, want different. %s",
						tc.testCall, tc.grantedPrefix, tc.description)
				}
			}
		})
	}
}

// TestGrantsReturnsRecordedGrants verifies that Grants returns the list of
// active session-scoped grants with correct content and order.
func TestGrantsReturnsRecordedGrants(t *testing.T) {
	p := New()

	// Initially empty
	grants := p.Grants()
	if len(grants) != 0 {
		t.Errorf("Grants() = %v, want empty initially", grants)
	}

	// Record a grant
	p.Grant([]string{"go", "test"})
	grants = p.Grants()
	if len(grants) != 1 {
		t.Errorf("after Grant([go test]), Grants() has len %d, want 1 grant", len(grants))
	}
	if grants[0] != "go test" {
		t.Errorf("after Grant([go test]), Grants()[0] = %q, want %q", grants[0], "go test")
	}

	// Record another
	p.Grant([]string{"git", "diff"})
	grants = p.Grants()
	if len(grants) != 2 {
		t.Errorf("after second Grant, Grants() has len %d, want 2 grants", len(grants))
	}
	// Verify both grants are present and in order
	if grants[0] != "go test" {
		t.Errorf("Grants()[0] = %q, want %q", grants[0], "go test")
	}
	if grants[1] != "git diff" {
		t.Errorf("Grants()[1] = %q, want %q", grants[1], "git diff")
	}

	// Record a grant with flags
	p.Grant([]string{"ls", "-la"})
	grants = p.Grants()
	if len(grants) != 3 {
		t.Errorf("after third Grant, Grants() has len %d, want 3 grants", len(grants))
	}
	if grants[2] != "ls -la" {
		t.Errorf("Grants()[2] = %q, want %q", grants[2], "ls -la")
	}
}

// TestClearGrants verifies that ClearGrants removes all active grants both
// from the Grants() list and from enforcement (ForCommand).
func TestClearGrants(t *testing.T) {
	p := New()

	// Record grants
	p.Grant([]string{"go", "test"})
	p.Grant([]string{"git", "diff"})

	// Verify grants are in the list
	grants := p.Grants()
	if len(grants) != 2 {
		t.Fatalf("setup: expected 2 grants, got %d", len(grants))
	}

	// Verify grants are enforced (grants always extend)
	got, _ := p.ForCommand([]string{"go", "test", "./..."})
	if got != DecisionAllow {
		t.Errorf("before ClearGrants(), ForCommand([go test ./...]) = %v, want DecisionAllow", got)
	}

	// Clear them
	p.ClearGrants()

	// Verify Grants() is now empty
	grants = p.Grants()
	if len(grants) != 0 {
		t.Errorf("after ClearGrants(), Grants() = %v, want empty", grants)
	}

	// Verify that exact commands in the default allowlist still work
	got, _ = p.ForCommand([]string{"go", "test"})
	if got != DecisionAllow {
		t.Errorf("after ClearGrants(), ForCommand([go test]) = %v, want DecisionAllow (from default allowlist)", got)
	}

	// But extended calls now ask (no longer in a grant, and defaults are exact-match)
	got, _ = p.ForCommand([]string{"go", "test", "./..."})
	if got != DecisionAskUser {
		t.Errorf("after ClearGrants(), ForCommand([go test ./...]) = %v, want DecisionAskUser (no grant, exact-match default)", got)
	}

	// Test a command not in the default allowlist (the grant we just cleared)
	// Grant something that's not in default allowlist
	p.Grant([]string{"curl"})
	got, _ = p.ForCommand([]string{"curl", "-v"})
	if got != DecisionAllow {
		t.Errorf("ForCommand([curl -v]) after Grant([curl]) = %v, want DecisionAllow", got)
	}

	// Now clear and verify it's no longer granted
	p.ClearGrants()
	got, _ = p.ForCommand([]string{"curl"})
	if got != DecisionAskUser {
		t.Errorf("after ClearGrants(), ForCommand([curl]) = %v, want DecisionAskUser", got)
	}
}

// TestGrantsDisabledByConfiguration verifies that setting
// allowSessionScopedGrants = false disables Grant entirely, and
// commands don't match session grants (though they match the allowlist).
func TestGrantsDisabledByConfiguration(t *testing.T) {
	p := New()
	// Disable session grants
	p.allowSessionScopedGrants = false

	// Attempt to grant a command
	err := p.Grant([]string{"curl"})
	if err != ErrGrantsDisabled {
		t.Errorf("Grant([curl]) with grants disabled = %v, want ErrGrantsDisabled", err)
	}

	// Verify the grant was not recorded
	grants := p.Grants()
	if len(grants) != 0 {
		t.Errorf("after failed Grant, Grants() = %v, want empty", grants)
	}

	// Try to call an unknown command (not in default allowlist)
	got, _ := p.ForCommand([]string{"curl"})
	if got != DecisionAskUser {
		t.Errorf("ForCommand([curl]) with grants disabled = %v, want DecisionAskUser", got)
	}
}

// TestGrantDisableSeam verifies that when allowSessionScopedGrants is toggled
// from true to false, previously-granted commands stop being allowed.
// This tests the critical seam where the configuration flag gates grant enforcement.
func TestGrantDisableSeam(t *testing.T) {
	p := New()

	// Start with grants enabled (default)
	if !p.allowSessionScopedGrants {
		t.Fatalf("setup: grants should be enabled by default")
	}

	// Grant a prefix (not in default allowlist)
	err := p.Grant([]string{"curl"})
	if err != nil {
		t.Fatalf("Grant([curl]) with grants enabled = %v, want nil", err)
	}

	// Verify the grant is recorded
	grants := p.Grants()
	if len(grants) != 1 {
		t.Fatalf("after Grant, Grants() has len %d, want 1", len(grants))
	}

	// Verify a matching command is allowed by the grant
	got, reason := p.ForCommand([]string{"curl", "-v"})
	if got != DecisionAllow {
		t.Errorf("ForCommand([curl -v]) with grant enabled = %v, want DecisionAllow. reason: %s", got, reason)
	}

	// Now disable grants
	p.allowSessionScopedGrants = false

	// Verify the grant is still in the list (the toggle doesn't clear them)
	grants = p.Grants()
	if len(grants) != 1 {
		t.Errorf("after disabling grants, Grants() has len %d, want 1 (grants persist)", len(grants))
	}

	// Verify the same command now asks (grant is no longer enforced)
	got, reason = p.ForCommand([]string{"curl", "-v"})
	if got != DecisionAskUser {
		t.Errorf("ForCommand([curl -v]) after disabling grants = %v, want DecisionAskUser. reason: %s", got, reason)
	}

	// Verify that exact-match allowlist commands still work
	got, _ = p.ForCommand([]string{"go", "test"})
	if got != DecisionAllow {
		t.Errorf("ForCommand([go test]) with grants disabled = %v, want DecisionAllow (from allowlist)", got)
	}
}

// TestAllowlistAndGrantDistinction verifies that allowlist entries
// and session grants work independently and that allowlist entries
// may not be extendable while grants are always extendable.
func TestAllowlistAndGrantDistinction(t *testing.T) {
	p := New()

	// The default allowlist includes "go test*" (extendable=false after fix)
	// so "go test" matches but "go test foo" now asks
	got, _ := p.ForCommand([]string{"go", "test"})
	if got != DecisionAllow {
		t.Errorf("ForCommand([go test]) = %v, want DecisionAllow (allowlist exact match)", got)
	}

	// But with the grant, extended calls match
	p.Grant([]string{"go", "test"})
	got, _ = p.ForCommand([]string{"go", "test", "foo"})
	if got != DecisionAllow {
		t.Errorf("ForCommand([go test foo]) after grant = %v, want DecisionAllow (grant always extends)", got)
	}

	// Clear the grant
	p.ClearGrants()

	// Now "go test" still works (from allowlist), but "go test foo" does not
	got, _ = p.ForCommand([]string{"go", "test"})
	if got != DecisionAllow {
		t.Errorf("after ClearGrants(), ForCommand([go test]) = %v, want DecisionAllow (allowlist)", got)
	}

	got, _ = p.ForCommand([]string{"go", "test", "foo"})
	if got != DecisionAskUser {
		t.Errorf("after ClearGrants(), ForCommand([go test foo]) = %v, want DecisionAskUser (non-extendable)", got)
	}
}

// TestExactMatchAllowlist verifies that the exact-match defaults work correctly.
// They should allow the exact prefix but ask for extended versions.
func TestExactMatchAllowlist(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		expected Decision
	}{
		// go build exact and extended
		{
			name:     "go build exact",
			argv:     []string{"go", "build"},
			expected: DecisionAllow,
		},
		{
			name:     "go build with args",
			argv:     []string{"go", "build", "./..."},
			expected: DecisionAskUser,
		},
		{
			name:     "go build with toolexec",
			argv:     []string{"go", "build", "-toolexec=/tmp/payload"},
			expected: DecisionAskUser,
		},
		// go test exact and extended
		{
			name:     "go test exact",
			argv:     []string{"go", "test"},
			expected: DecisionAllow,
		},
		{
			name:     "go test with args",
			argv:     []string{"go", "test", "./..."},
			expected: DecisionAskUser,
		},
		{
			name:     "go test with exec",
			argv:     []string{"go", "test", "-exec=/tmp/payload"},
			expected: DecisionAskUser,
		},
		// git diff exact and extended
		{
			name:     "git diff exact",
			argv:     []string{"git", "diff"},
			expected: DecisionAllow,
		},
		{
			name:     "git diff with args",
			argv:     []string{"git", "diff", "HEAD~1"},
			expected: DecisionAskUser,
		},
		{
			name:     "git diff with output",
			argv:     []string{"git", "diff", "--output=/tmp/patch"},
			expected: DecisionAskUser,
		},
		// git log exact and extended
		{
			name:     "git log exact",
			argv:     []string{"git", "log"},
			expected: DecisionAllow,
		},
		{
			name:     "git log with args",
			argv:     []string{"git", "log", "--oneline"},
			expected: DecisionAskUser,
		},
		// ls exact and extended
		{
			name:     "ls exact",
			argv:     []string{"ls"},
			expected: DecisionAllow,
		},
		{
			name:     "ls with args",
			argv:     []string{"ls", "-la"},
			expected: DecisionAskUser,
		},
		// Near-misses per default entry
		{
			name:     "go buildx near-miss",
			argv:     []string{"go", "buildx"},
			expected: DecisionAskUser,
		},
		{
			name:     "go testfoo near-miss",
			argv:     []string{"go", "testfoo"},
			expected: DecisionAskUser,
		},
		{
			name:     "git diffx near-miss",
			argv:     []string{"git", "diffx"},
			expected: DecisionAskUser,
		},
		{
			name:     "git logx near-miss",
			argv:     []string{"git", "logx"},
			expected: DecisionAskUser,
		},
		{
			name:     "lsx near-miss",
			argv:     []string{"lsx"},
			expected: DecisionAskUser,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			got, _ := p.ForCommand(tc.argv)
			if got != tc.expected {
				t.Errorf("ForCommand(%v) = %v, want %v", tc.argv, got, tc.expected)
			}
		})
	}
}

// TestForCommandEdgeCases verifies edge cases like nil and empty argv.
func TestForCommandEdgeCases(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		expected Decision
	}{
		{
			name:     "nil argv",
			argv:     nil,
			expected: DecisionAskUser,
		},
		{
			name:     "empty argv",
			argv:     []string{},
			expected: DecisionAskUser,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			got, _ := p.ForCommand(tc.argv)
			if got != tc.expected {
				t.Errorf("ForCommand(%v) = %v, want %v", tc.argv, got, tc.expected)
			}
		})
	}
}

// TestShellDetectionByBasename verifies that shells are detected regardless
// of their installation path, by matching the basename.
func TestShellDetectionByBasename(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		// Homebrew install location (macOS)
		{
			name: "homebrew bash",
			argv: []string{"/opt/homebrew/bin/bash"},
		},
		{
			name: "homebrew zsh",
			argv: []string{"/opt/homebrew/bin/zsh"},
		},
		// Custom location
		{
			name: "custom location bash",
			argv: []string{"/custom/path/to/bash"},
		},
		// Environment variable reference (still basename)
		{
			name: "env with homebrew bash",
			argv: []string{"/opt/homebrew/bin/env", "/opt/homebrew/bin/bash"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			got, _ := p.ForCommand(tc.argv)
			if got != DecisionAskUser {
				t.Errorf("ForCommand(%v) = %v, want DecisionAskUser (shell detected by basename)", tc.argv, got)
			}
		})
	}
}
