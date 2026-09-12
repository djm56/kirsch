// Package config loads Kirsch's TOML configuration.
//
// Two rules shape this package. Nothing here reads $HOME or any other part of
// the environment: the caller resolves paths and passes them in, so a test can
// exercise every precedence layer without touching the real user's files. And
// no secret is ever accepted from a config file — see checkSecrets.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// ProviderConfig selects and configures the model provider.
type ProviderConfig struct {
	Default   string                  `toml:"default"`
	Anthropic AnthropicProviderConfig `toml:"anthropic"`
}

// AnthropicProviderConfig is the only provider in v0.1.
type AnthropicProviderConfig struct {
	Model         string `toml:"model"`
	APIKeyEnv     string `toml:"api_key_env"`
	PromptCaching bool   `toml:"prompt_caching"`
	Thinking      string `toml:"thinking"` // off | low | medium | high
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
			Default: "anthropic",
			Anthropic: AnthropicProviderConfig{
				Model:         "claude-sonnet-5",
				APIKeyEnv:     "ANTHROPIC_API_KEY",
				PromptCaching: true,
				Thinking:      "off",
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
			ProjectFiles:           []string{"AGENTS.md", "CLAUDE.md", ".kirsch/context.md"},
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

// Warning is a non-fatal configuration problem. Unknown keys produce these;
// they never fail startup, because a config written for a newer Kirsch must
// still boot an older one.
type Warning struct {
	File string
	Key  string
	Msg  string
}

func (w Warning) String() string {
	if w.Key != "" {
		return fmt.Sprintf("%s: unknown key %q (ignored)", w.File, w.Key)
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

// Load resolves the four precedence layers: defaults → global → project →
// flags, merged per key.
func Load(o Options) (Config, []Warning, error) {
	cfg := Defaults()
	var warns []Warning

	for _, layer := range []struct{ path string }{{o.GlobalPath}, {o.ProjectPath}} {
		if layer.path == "" {
			continue
		}
		w, err := mergeFile(&cfg, layer.path)
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
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil // a missing config file is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	md, err := toml.Decode(string(data), cfg)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if err := checkSecrets(md, path); err != nil {
		return nil, err
	}

	var warns []Warning
	for _, key := range md.Undecoded() {
		warns = append(warns, Warning{File: path, Key: key.String()})
	}
	return warns, nil
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
	if f.Provider.Anthropic.Model != "" {
		cfg.Provider.Anthropic.Model = f.Provider.Anthropic.Model
	}
	if f.Provider.Anthropic.Thinking != "" {
		cfg.Provider.Anthropic.Thinking = f.Provider.Anthropic.Thinking
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

// Validate reports configuration that parses but cannot be honoured.
func (c Config) Validate() error {
	switch c.Provider.Anthropic.Thinking {
	case "off", "low", "medium", "high":
	default:
		return fmt.Errorf("provider.anthropic.thinking = %q; want one of off, low, medium, high",
			c.Provider.Anthropic.Thinking)
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
