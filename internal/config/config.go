// Package config loads Kirsch's TOML configuration.
//
// Two rules shape this package. Nothing here reads $HOME or any other part of
// the environment: the caller resolves paths and passes them in, so a test can
// exercise every precedence layer without touching the real user's files. And
// no secret is ever accepted from a config file — see checkSecrets.
package config

import (
	"fmt"
	"maps"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the full v0.1 configuration. Blocks that nothing consumes until
// M3/M4 are still parsed and validated here: a key that silently does nothing
// is worse than one that does not exist.
type Config struct {
	Provider  ProviderConfig  `toml:"provider"`
	Policy    PolicyConfig    `toml:"policy"`
	Context   ContextConfig   `toml:"context"`
	Session   SessionConfig   `toml:"session"`
	Telemetry TelemetryConfig `toml:"telemetry"`
}

// ProviderConfig selects the active endpoint and configures every endpoint.
// It is decoded by hand (see UnmarshalTOML) because TOML puts `default` and the
// endpoint tables side by side under [provider].
type ProviderConfig struct {
	Default   string
	Endpoints map[string]EndpointConfig
	unknown   []string // endpoint keys this version does not recognise; reported as warnings
}

// EndpointConfig is one configured Messages-format endpoint (ADR 0008).
// A user-defined endpoint must set base_url, auth, api_key_env and model;
// thinking defaults to "off" and prompt_caching to false.
type EndpointConfig struct {
	BaseURL       string // https, or http on a literal loopback host only
	Auth          string // "x-api-key" or "bearer"
	APIKeyEnv     string // the NAME of the variable holding the key, never the key
	Model         string
	PromptCaching bool
	Thinking      string // off | low | medium | high
}

// PolicyConfig governs approvals and command execution (consumed from M2).
type PolicyConfig struct {
	DefaultCommandTimeoutSeconds int      `toml:"default_command_timeout_seconds"`
	RequireApprovalForPatches    bool     `toml:"require_approval_for_patches"`
	RequireApprovalForCommands   bool     `toml:"require_approval_for_commands"`
	AllowSessionScopedGrants     bool     `toml:"allow_session_scoped_grants"`
	EnvPassthrough               []string `toml:"env_passthrough"`
}

// ContextConfig governs project-context injection (consumed from M3).
type ContextConfig struct {
	ProjectFiles           []string `toml:"project_files"`
	MaxProjectContextBytes int      `toml:"max_project_context_bytes"`
}

// SessionConfig governs the session store (consumed from M4).
type SessionConfig struct {
	StorageDir string `toml:"storage_dir"`
	AutoResume bool   `toml:"auto_resume"`
}

// TelemetryConfig governs the debug log.
type TelemetryConfig struct {
	DebugLog bool `toml:"debug_log"`
}

// Defaults returns the built-in configuration — the first precedence layer.
// Every key has one, which is why a missing config file is not an error.
func Defaults() Config {
	return Config{
		Provider: ProviderConfig{
			Default: "opencode",
			Endpoints: map[string]EndpointConfig{
				"opencode": {
					BaseURL: "https://opencode.ai/zen/go/v1", Auth: "x-api-key",
					APIKeyEnv: "OPENCODE_API_KEY", // #nosec G101 -- variable name, not a secret
					Model:     "minimax-m3", PromptCaching: true, Thinking: "off",
				},
				"anthropic": {
					BaseURL: "https://api.anthropic.com/v1", Auth: "x-api-key",
					APIKeyEnv: "ANTHROPIC_API_KEY", // #nosec G101 -- variable name, not a secret
					Model:     "claude-sonnet-5-5", PromptCaching: true, Thinking: "off",
				},
			},
		},
		Policy: PolicyConfig{
			DefaultCommandTimeoutSeconds: 60,
			RequireApprovalForPatches:    true,
			RequireApprovalForCommands:   true,
			AllowSessionScopedGrants:     true,
			EnvPassthrough:               []string{},
		},
		Context: ContextConfig{
			ProjectFiles:           []string{"AGENTS.md", "CLAUDE.md"},
			MaxProjectContextBytes: 32768,
		},
		Session: SessionConfig{
			StorageDir: "~/.local/share/kirsch/sessions",
			AutoResume: true,
		},
		Telemetry: TelemetryConfig{DebugLog: false},
	}
}

// Options is what Load needs. Paths are supplied by the caller; the loader
// never consults the environment, so tests are hermetic.
type Options struct {
	GlobalPath  string  // may be empty or missing
	ProjectPath string  // may be empty or missing
	Flags       *Config // non-nil fields override everything; see ApplyFlags
	HomeDir     string  // for ~ expansion; empty disables it
}

// WarningKind categorizes configuration warnings.
type WarningKind int

const (
	WarnUnknownKey WarningKind = iota
	WarnProjectIgnored
)

// Warning is a non-fatal configuration problem. Unknown keys produce these;
// they never fail startup, because a config written for a newer Kirsch must
// still boot an older one.
type Warning struct {
	File string
	Key  string
	Msg  string
	Kind WarningKind
}

func (w Warning) String() string {
	switch w.Kind {
	case WarnProjectIgnored:
		if w.Msg != "" {
			return fmt.Sprintf("%s: %s", w.File, w.Msg)
		}
		return fmt.Sprintf("%s: key %q ignored (a project config may set only [context].project_files)", w.File, w.Key)
	case WarnUnknownKey:
		if w.Key != "" {
			return fmt.Sprintf("%s: unknown key %q (ignored)", w.File, w.Key)
		}
		return fmt.Sprintf("%s: %s", w.File, w.Msg)
	}
	return fmt.Sprintf("%s: %s", w.File, w.Msg)
}

// secretKeyRe matches key names that must never carry a value in a config
// file. plan §5.
var secretKeyRe = regexp.MustCompile(`(?i)(api_?key|token|secret|password)`)

// exemptKeys are names that match secretKeyRe but hold a variable *name*
// rather than a value. The distinction is the whole point: api_key_env says
// where to find the key, which is safe to commit; api_key would be the key.
var exemptKeys = map[string]bool{
	"api_key_env": true,
}

// UnmarshalTOML decodes [provider] with per-field merge. It is called by
// BurntSushi/toml with the decoded [provider] table as a map[string]any.
func (p *ProviderConfig) UnmarshalTOML(data any) error {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("provider must be a table")
	}

	// Iterate top-level keys in sorted order for determinism
	sortedKeys := slices.Sorted(maps.Keys(m))
	for _, key := range sortedKeys {
		val := m[key]
		if key == "default" {
			// default is a string at the top level
			if s, ok := val.(string); ok {
				p.Default = s
			} else {
				return fmt.Errorf("provider.default must be a string")
			}
			continue
		}

		// Every other key is an endpoint table
		endptTable, ok := val.(map[string]any)
		if !ok {
			// Non-table values are unknown keys; record them
			p.unknown = append(p.unknown, "provider."+key)
			continue
		}

		// Initialize endpoints map if nil
		if p.Endpoints == nil {
			p.Endpoints = make(map[string]EndpointConfig)
		}

		// Get existing endpoint, or start new user-defined endpoint with thinking defaulting to "off"
		endpt, exists := p.Endpoints[key]
		if !exists {
			endpt = EndpointConfig{Thinking: "off"}
		}

		// Merge field by field, in sorted order for determinism
		sortedFields := slices.Sorted(maps.Keys(endptTable))
		for _, fkey := range sortedFields {
			fval := endptTable[fkey]
			switch fkey {
			case "base_url":
				if s, ok := fval.(string); ok {
					endpt.BaseURL = s
				} else {
					return fmt.Errorf("provider.%s.base_url must be a string", key)
				}
			case "auth":
				if s, ok := fval.(string); ok {
					endpt.Auth = s
				} else {
					return fmt.Errorf("provider.%s.auth must be a string", key)
				}
			case "api_key_env":
				if s, ok := fval.(string); ok {
					endpt.APIKeyEnv = s
				} else {
					return fmt.Errorf("provider.%s.api_key_env must be a string", key)
				}
			case "model":
				if s, ok := fval.(string); ok {
					endpt.Model = s
				} else {
					return fmt.Errorf("provider.%s.model must be a string", key)
				}
			case "prompt_caching":
				if b, ok := fval.(bool); ok {
					endpt.PromptCaching = b
				} else {
					return fmt.Errorf("provider.%s.prompt_caching must be a bool", key)
				}
			case "thinking":
				if s, ok := fval.(string); ok {
					endpt.Thinking = s
				} else {
					return fmt.Errorf("provider.%s.thinking must be a string", key)
				}
			default:
				// Unknown fields in endpoint
				p.unknown = append(p.unknown, fmt.Sprintf("provider.%s.%s", key, fkey))
			}
		}

		p.Endpoints[key] = endpt
	}

	return nil
}

// Load resolves the four precedence layers: defaults → global → project →
// flags, merged per key.
func Load(o Options) (Config, []Warning, error) {
	cfg := Defaults()
	var warns []Warning

	if o.GlobalPath != "" {
		w, err := mergeFile(&cfg, o.GlobalPath)
		warns = append(warns, w...)
		if err != nil {
			return cfg, warns, err
		}
	}

	if o.ProjectPath != "" {
		w, err := mergeProjectFile(&cfg, o.ProjectPath)
		warns = append(warns, w...)
		if err != nil {
			return cfg, warns, err
		}
	}

	if o.Flags != nil {
		ApplyFlags(&cfg, o.Flags)
	}
	if o.HomeDir != "" {
		cfg.Session.StorageDir = ExpandHome(cfg.Session.StorageDir, o.HomeDir)
	}
	return cfg, warns, nil
}

// mergeFile decodes one file over cfg.
//
// Decoding *into the already-populated struct* is what makes the merge
// per-key: BurntSushi/toml only assigns fields the file actually mentions, so
// a project file that sets one key leaves the global's other keys standing.
// Unmarshalling into a fresh struct and copying it over would blank them —
// the naive approach the instruction set warns about.
func mergeFile(cfg *Config, path string) ([]Warning, error) {
	// The path is supplied by the caller, which is the only component allowed
	// to read the environment (see Options). Nothing model- or
	// repository-controlled reaches here. gosec G304.
	data, err := os.ReadFile(path) // #nosec G304 -- caller-supplied config path
	if os.IsNotExist(err) {
		return nil, nil // a missing config file is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Decode to throwaway map to check secrets before decoding into cfg
	var md toml.MetaData
	var throwaway map[string]any
	md, err = toml.Decode(string(data), &throwaway)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if err := checkSecrets(md, path); err != nil {
		return nil, err
	}

	// Now decode into cfg
	md, err = toml.Decode(string(data), cfg)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var warns []Warning
	for _, key := range md.Undecoded() {
		warns = append(warns, Warning{File: path, Key: key.String(), Kind: WarnUnknownKey})
	}

	// Convert provider.unknown entries to warnings and clear the slice
	if cfg.Provider.unknown != nil {
		for _, key := range cfg.Provider.unknown {
			warns = append(warns, Warning{File: path, Key: key, Kind: WarnUnknownKey})
		}
		cfg.Provider.unknown = nil
	}

	return warns, nil
}

// mergeProjectFile decodes a project config file over cfg. It enforces an
// allowlist: only [context].project_files is permitted. All other keys are
// ignored with a warning.
func mergeProjectFile(cfg *Config, path string) ([]Warning, error) {
	// The path is supplied by the caller, which is the only component allowed
	// to read the environment (see Options). Nothing model- or
	// repository-controlled reaches here. gosec G304.
	data, err := os.ReadFile(path) // #nosec G304 -- caller-supplied config path
	if os.IsNotExist(err) {
		return nil, nil // a missing config file is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Decode to throwaway map first
	var md toml.MetaData
	var decoded map[string]any
	md, err = toml.Decode(string(data), &decoded)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// Check secrets before proceeding
	if err := checkSecrets(md, path); err != nil {
		return nil, err
	}

	var warns []Warning

	// Known top-level config sections
	knownSections := map[string]bool{
		"provider":  true,
		"policy":    true,
		"context":   true,
		"session":   true,
		"telemetry": true,
	}

	// Only [context].project_files is allowed. Walk all leaf keys and warn about
	// everything else (except Hash-type keys, which are table headers).
	for _, keyPath := range md.Keys() {
		keyStr := keyPath.String()

		// Skip if it's the allowed key
		if keyStr == "context.project_files" {
			// Check if it's valid (list of strings)
			contextVal, hasContext := decoded["context"]
			if hasContext {
				contextTable, ok := contextVal.(map[string]any)
				if ok {
					if projectFilesVal, hasProjectFiles := contextTable["project_files"]; hasProjectFiles {
						if projectFilesList, ok := projectFilesVal.([]any); ok {
							allStrings := true
							projectFiles := []string{}
							for _, v := range projectFilesList {
								if s, ok := v.(string); ok {
									projectFiles = append(projectFiles, s)
								} else {
									allStrings = false
									break
								}
							}
							if allStrings {
								cfg.Context.ProjectFiles = projectFiles
								continue // Don't warn about valid project_files
							}
						}
					}
				}
			}
			// If we get here, project_files is invalid or missing
			warns = append(warns, Warning{File: path, Key: keyStr, Kind: WarnProjectIgnored, Msg: "context.project_files must be a list of strings; ignored"})
			continue
		}

		// Skip Hash-type keys (table headers)
		if md.Type(keyPath...) == "Hash" {
			continue
		}

		// Warn about everything else
		warns = append(warns, Warning{File: path, Key: keyStr, Kind: WarnProjectIgnored})
	}

	// Also warn about top-level unknown tables (sections that aren't recognized)
	// Iterate in sorted order for determinism
	for _, k := range slices.Sorted(maps.Keys(decoded)) {
		if !knownSections[k] && md.Type(k) == "Hash" {
			warns = append(warns, Warning{File: path, Key: k, Kind: WarnProjectIgnored})
		}
	}

	// Sort warns by Key and drop exact duplicates for determinism
	slices.SortFunc(warns, func(a, b Warning) int {
		return strings.Compare(a.Key, b.Key)
	})
	// Deduplicate by Key
	seen := make(map[string]bool)
	var deduped []Warning
	for _, w := range warns {
		if !seen[w.Key] {
			deduped = append(deduped, w)
			seen[w.Key] = true
		}
	}

	return deduped, nil
}

// checkSecrets refuses a config file that carries a credential.
//
// Checked against every key the file defines, decoded or not, so an unknown
// key named `api_key` is refused rather than merely warned about — being
// unrecognised is not a reason to tolerate a secret sitting in a file that
// people commit.
func checkSecrets(md toml.MetaData, path string) error {
	for _, key := range append(md.Keys(), md.Undecoded()...) {
		parts := key
		leaf := parts[len(parts)-1]
		if exemptKeys[strings.ToLower(leaf)] {
			continue
		}
		if !secretKeyRe.MatchString(leaf) {
			continue
		}
		if md.Type(key...) == "Hash" {
			continue // a table named e.g. [secrets] carries no value itself
		}
		// Multi-line and punctuated on purpose: printed to the user as
		// onboarding text rather than wrapped into another error.
		//nolint:staticcheck // ST1005: user-facing message, not an error fragment
		return fmt.Errorf(
			"%s: key %q looks like a credential\n\n"+
				"Kirsch never reads secrets from config files, because config files get "+
				"committed and shared. Set the value in an environment variable and name "+
				"that variable with `api_key_env` instead.",
			path, key.String())
	}
	return nil
}

// ApplyFlags overlays command-line values. Only non-zero fields override, so
// an unset flag leaves the file-derived value alone.
func ApplyFlags(cfg *Config, f *Config) {
	if f.Provider.Default != "" {
		cfg.Provider.Default = f.Provider.Default
	}
	if f.Session.StorageDir != "" {
		cfg.Session.StorageDir = f.Session.StorageDir
	}
	if f.Telemetry.DebugLog {
		cfg.Telemetry.DebugLog = true
	}
}

// ExpandHome replaces a leading ~ with home. Path-valued keys only.
func ExpandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// ResolveKey returns the endpoint's API key and the name of the variable that
// supplied it: KIRSCH_ + APIKeyEnv first, then APIKeyEnv. Both empty: "", "".
func (e EndpointConfig) ResolveKey(getenv func(string) string) (key, source string) {
	// Try KIRSCH_ prefix first
	prefixedKey := "KIRSCH_" + e.APIKeyEnv
	if val := getenv(prefixedKey); val != "" {
		return val, prefixedKey
	}
	// Fall back to unprefixed
	if val := getenv(e.APIKeyEnv); val != "" {
		return val, e.APIKeyEnv
	}
	return "", ""
}

// ValidateBaseURL checks an endpoint base_url. ADR 0008.
//
// The URL must parse and carry no userinfo. The scheme must be https or
// http (net/url lower-cases it, so HTTP:// is accepted). https accepts any
// non-empty host. http is accepted only on loopback: the name localhost
// (any case, no trailing dot), an IPv4 address in 127.0.0.0/8 written as a
// full dotted quad, or [::1]. Everything else over http is refused,
// including 0.0.0.0, [::], short or zero-padded IPv4 forms (127.1,
// 127.00.0.1), IPv4-mapped IPv6 and zoned addresses. The port is not
// checked.
func ValidateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base_url parse error: %w", err)
	}

	// Reject userinfo
	if u.User != nil {
		return fmt.Errorf("base_url must not include userinfo")
	}

	// Scheme must be https or http
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("base_url scheme must be https or http, got %q", u.Scheme)
	}

	// Host must be non-empty
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("base_url must have a host")
	}

	// https requires just a non-empty host (already checked)
	if u.Scheme == "https" {
		return nil
	}

	// http is allowed only on loopback
	// Check localhost (case-insensitive)
	if strings.EqualFold(host, "localhost") {
		return nil
	}

	// Try to parse as netip address
	addr, err := netip.ParseAddr(host)
	if err == nil {
		// IPv4-mapped (::ffff:127.0.0.1) and zoned (::1%eth0) forms fall through:
		// a mapped address is not Is4, and neither equals IPv6Loopback.
		// TestValidateBaseURL and TestValidateBaseURLEdgeCases pin this.

		// Check if loopback
		if addr.Is4() {
			// IPv4: must start with 127
			if addr.AsSlice()[0] == 127 {
				return nil
			}
		} else if addr.Is6() {
			// IPv6: must be ::1 (IPv6 loopback)
			if addr == netip.IPv6Loopback() {
				return nil
			}
		}
		return fmt.Errorf("http is allowed only on loopback addresses")
	}

	// Not a valid address and not localhost
	return fmt.Errorf("http is allowed only on localhost or loopback addresses")
}

// Validate reports configuration that parses but cannot be honoured.
func (c Config) Validate() error {
	// Validate provider.default
	if _, ok := c.Provider.Endpoints[c.Provider.Default]; !ok {
		return fmt.Errorf("provider.default = %q; endpoint does not exist", c.Provider.Default)
	}

	// Validate each endpoint in sorted order
	endpoints := make([]string, 0, len(c.Provider.Endpoints))
	for name := range c.Provider.Endpoints {
		endpoints = append(endpoints, name)
	}
	sort.Strings(endpoints)

	for _, name := range endpoints {
		e := c.Provider.Endpoints[name]

		// Check required fields
		if e.BaseURL == "" {
			return fmt.Errorf("provider.%s.base_url is required", name)
		}
		if e.Auth == "" {
			return fmt.Errorf("provider.%s.auth is required", name)
		}
		if e.APIKeyEnv == "" {
			return fmt.Errorf("provider.%s.api_key_env is required", name)
		}
		if e.Model == "" {
			return fmt.Errorf("provider.%s.model is required", name)
		}

		// Check Auth is valid
		if e.Auth != "x-api-key" && e.Auth != "bearer" {
			return fmt.Errorf("provider.%s.auth = %q; want x-api-key or bearer", name, e.Auth)
		}

		// Check Thinking is valid
		switch e.Thinking {
		case "off", "low", "medium", "high":
		default:
			return fmt.Errorf("provider.%s.thinking = %q; want one of off, low, medium, high", name, e.Thinking)
		}

		// Validate BaseURL
		if err := ValidateBaseURL(e.BaseURL); err != nil {
			return fmt.Errorf("provider.%s.base_url: %w", name, err)
		}
	}

	if c.Policy.DefaultCommandTimeoutSeconds <= 0 {
		return fmt.Errorf("policy.default_command_timeout_seconds = %d; want a positive number",
			c.Policy.DefaultCommandTimeoutSeconds)
	}
	if c.Context.MaxProjectContextBytes < 0 {
		return fmt.Errorf("context.max_project_context_bytes = %d; want zero or more",
			c.Context.MaxProjectContextBytes)
	}
	return nil
}

// GlobalPath returns the global config location for the given environment.
// Resolved by the caller, never inside Load.
func GlobalPath(xdgConfigHome, home string) string {
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "kirsch", "config.toml")
	}
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "kirsch", "config.toml")
}

// ProjectPath returns the per-workspace config location.
func ProjectPath(workspaceRoot string) string {
	if workspaceRoot == "" {
		return ""
	}
	return filepath.Join(workspaceRoot, ".kirsch", "config.toml")
}
