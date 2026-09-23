// Package policy defines the command allowlist, session grants, and approval
// decisions. It stands between a model and the shell, enforcing that every
// write and execution passes a policy decision the user can see.
package policy

import (
	"errors"
	"path/filepath"
	"strings"
)

// Grant validation errors.
var (
	ErrGrantsDisabled      = errors.New("session grants are disabled")
	ErrEmptyPrefix         = errors.New("grant prefix cannot be empty")
	ErrBareWildcard        = errors.New("bare wildcard '*' cannot be granted")
	ErrShellGrantForbidden = errors.New("shells cannot be granted")
)

// Decision is a policy outcome: allow, ask the user, or deny.
type Decision uint8

const (
	DecisionAllow Decision = iota
	DecisionAskUser
	DecisionDeny
)

// allowlistEntry represents a command pattern in the allowlist.
type allowlistEntry struct {
	prefix     []string // the argv prefix to match (e.g., []string{"go", "test"})
	extendable bool     // if true, matches extended argv; if false, matches only exact prefix
}

// Policy holds the command allowlist, session-scoped grants, and configuration.
// Note: Policy is not goroutine-safe. Single-goroutine use is load-bearing.
type Policy struct {
	// allowlist holds default command patterns that are allowed.
	allowlist []allowlistEntry
	// allowSessionScopedGrants controls whether Grant() is enabled (config: allow_session_scoped_grants).
	allowSessionScopedGrants bool
	// grants records session-scoped command prefixes approved by the user.
	grants [][]string
}

// New creates a Policy with the default allowlist and grants enabled.
func New() *Policy {
	return &Policy{
		allowlist: []allowlistEntry{
			{prefix: []string{"go", "build"}, extendable: false},
			{prefix: []string{"go", "test"}, extendable: false},
			{prefix: []string{"git", "diff"}, extendable: false},
			{prefix: []string{"git", "log"}, extendable: false},
			{prefix: []string{"ls"}, extendable: false},
		},
		allowSessionScopedGrants: true,
		grants:                   make([][]string, 0),
	}
}

// ForCommand decides whether a command is allowed, requires user approval, or
// is denied. It returns the decision and a human-readable reason.
func (p *Policy) ForCommand(argv []string) (Decision, string) {
	// Shells always require approval, never allowed
	if isShell(argv) {
		return DecisionAskUser, "shell commands require approval"
	}

	// Check allowlist
	for _, entry := range p.allowlist {
		if matchesPattern(entry.prefix, entry.extendable, argv) {
			return DecisionAllow, "matched allowlist"
		}
	}

	// Check session grants (only if enabled)
	if p.allowSessionScopedGrants {
		for _, grant := range p.grants {
			if matchesPattern(grant, true, argv) { // grants always extend
				return DecisionAllow, "matched session grant"
			}
		}
	}

	// Default: ask user
	return DecisionAskUser, "not in allowlist or grants"
}

// ForPatch decides whether a patch may be applied. Per the specification,
// ForPatch never returns DecisionAllow. All patches require explicit approval.
func (p *Policy) ForPatch(files []string) (Decision, string) {
	return DecisionAskUser, "file patches always require approval"
}

// Grant records a session-scoped grant for a command prefix. It returns an
// error if the prefix is invalid (bare wildcard, empty, or a shell).
func (p *Policy) Grant(argv []string) error {
	// Grants must be enabled
	if !p.allowSessionScopedGrants {
		return ErrGrantsDisabled
	}

	// Refuse empty argv
	if len(argv) == 0 {
		return ErrEmptyPrefix
	}

	// Refuse bare wildcard
	if len(argv) == 1 && argv[0] == "*" {
		return ErrBareWildcard
	}

	// Refuse shells
	if isShell(argv) {
		return ErrShellGrantForbidden
	}

	// Record the grant
	p.grants = append(p.grants, argv)
	return nil
}

// Grants returns the list of active session-scoped grants as formatted strings.
func (p *Policy) Grants() []string {
	result := make([]string, len(p.grants))
	for i, grant := range p.grants {
		result[i] = strings.Join(grant, " ")
	}
	return result
}

// ClearGrants removes all active session-scoped grants.
func (p *Policy) ClearGrants() {
	p.grants = make([][]string, 0)
}

// isShell checks if a command (first argv element) is a shell or shell runner.
// It detects direct shell invocations (sh, bash, zsh, etc.) by basename,
// absolute paths to shells, and shell runners (env, xargs, nohup).
// It does NOT catch deliberately renamed or symlinked shells (e.g., cp /bin/bash ./mytool).
func isShell(argv []string) bool {
	if len(argv) == 0 {
		return false
	}

	cmd := argv[0]
	basename := filepath.Base(cmd)

	// Define the set of shell names and runners by basename
	shells := map[string]bool{
		"sh":    true,
		"bash":  true,
		"zsh":   true,
		"dash":  true,
		"env":   true,
		"xargs": true,
		"nohup": true,
	}

	// Check if the basename is a known shell or runner
	if shells[basename] {
		return true
	}

	return false
}

// matchesPattern checks if argv matches an allowlist/grant pattern.
// If extendable is true, matches if argv starts with the prefix pattern.
// If extendable is false, matches only if argv exactly equals the pattern.
func matchesPattern(pattern []string, extendable bool, argv []string) bool {
	if len(pattern) == 0 {
		return false
	}

	// Check if argv has the pattern as a prefix
	if len(argv) < len(pattern) {
		return false
	}

	for i, p := range pattern {
		if argv[i] != p {
			return false
		}
	}

	// If extendable, any argv with this prefix matches
	if extendable {
		return true
	}

	// If not extendable, argv must be exactly this length
	return len(argv) == len(pattern)
}
