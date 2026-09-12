package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPrecedenceAcrossFourLayers walks defaults → global → project → flags and
// checks that each layer wins over the one before it, on a different key each
// time so a layer cannot pass by accident.
func TestPrecedenceAcrossFourLayers(t *testing.T) {
	dir := t.TempDir()
	global := write(t, dir, "global.toml", `
[provider.anthropic]
model = "from-global"
thinking = "low"

[session]
storage_dir = "/from/global"
`)
	project := write(t, dir, "project.toml", `
[provider.anthropic]
thinking = "high"
`)

	cfg, warns, err := Load(Options{
		GlobalPath:  global,
		ProjectPath: project,
		Flags:       &Config{Session: SessionConfig{StorageDir: "/from/flag"}},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}

	// defaults survive where nothing overrides
	if got := cfg.Provider.Default; got != "anthropic" {
		t.Errorf("provider.default = %q, want the default %q", got, "anthropic")
	}
	// global beats defaults
	if got := cfg.Provider.Anthropic.Model; got != "from-global" {
		t.Errorf("model = %q, want from-global", got)
	}
	// project beats global
	if got := cfg.Provider.Anthropic.Thinking; got != "high" {
		t.Errorf("thinking = %q, want high (project overrides global)", got)
	}
	// flags beat everything
	if got := cfg.Session.StorageDir; got != "/from/flag" {
		t.Errorf("storage_dir = %q, want /from/flag", got)
	}
}

// TestProjectFileDoesNotBlankGlobalSiblings is the merge bug the instruction
// set warns about: unmarshalling a nested table into a fresh struct and
// assigning it wipes the keys the project file did not mention.
func TestProjectFileDoesNotBlankGlobalSiblings(t *testing.T) {
	dir := t.TempDir()
	global := write(t, dir, "g.toml", `
[provider.anthropic]
model = "keep-me"
api_key_env = "KEEP_THIS_TOO"
prompt_caching = true
`)
	project := write(t, dir, "p.toml", `
[provider.anthropic]
thinking = "medium"
`)
	cfg, _, err := Load(Options{GlobalPath: global, ProjectPath: project})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider.Anthropic.Model != "keep-me" {
		t.Errorf("model = %q; the project file blanked a sibling key it never mentioned",
			cfg.Provider.Anthropic.Model)
	}
	if cfg.Provider.Anthropic.APIKeyEnv != "KEEP_THIS_TOO" {
		t.Errorf("api_key_env = %q; blanked by the project layer", cfg.Provider.Anthropic.APIKeyEnv)
	}
	if !cfg.Provider.Anthropic.PromptCaching {
		t.Error("prompt_caching was blanked by the project layer")
	}
	if cfg.Provider.Anthropic.Thinking != "medium" {
		t.Errorf("thinking = %q, want medium", cfg.Provider.Anthropic.Thinking)
	}
}

// TestSecretShapedKeysRefused checks both sides of the api_key_env line: a
// variable *name* is fine, a value is not.
func TestSecretShapedKeysRefused(t *testing.T) {
	refused := []struct{ name, body string }{
		{"api_key", "[provider.anthropic]\napi_key = \"sk-ant-real-key\"\n"},
		{"apikey", "[provider.anthropic]\napikey = \"sk-ant-real-key\"\n"},
		{"token", "[provider]\ntoken = \"ghp_xxx\"\n"},
		{"secret", "[policy]\nsecret = \"hunter2\"\n"},
		{"password", "[session]\npassword = \"hunter2\"\n"},
		{"mixed case", "[provider]\nAPI_KEY = \"sk-ant\"\n"},
	}
	for _, tc := range refused {
		t.Run("refused/"+tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := write(t, dir, "c.toml", tc.body)
			_, _, err := Load(Options{GlobalPath: p})
			if err == nil {
				t.Fatal("a credential in a config file was accepted")
			}
			if !strings.Contains(err.Error(), p) {
				t.Errorf("error does not name the file: %v", err)
			}
		})
	}

	t.Run("accepted/api_key_env", func(t *testing.T) {
		dir := t.TempDir()
		p := write(t, dir, "c.toml", "[provider.anthropic]\napi_key_env = \"ANTHROPIC_API_KEY\"\n")
		cfg, _, err := Load(Options{GlobalPath: p})
		if err != nil {
			t.Fatalf("api_key_env holds a variable name, not a value, and must be allowed: %v", err)
		}
		if cfg.Provider.Anthropic.APIKeyEnv != "ANTHROPIC_API_KEY" {
			t.Errorf("api_key_env = %q", cfg.Provider.Anthropic.APIKeyEnv)
		}
	})
}

// TestUnknownKeyWarnsButLoads: a config written for a newer Kirsch must still
// boot an older one.
func TestUnknownKeyWarnsButLoads(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "c.toml", `
[provider.anthropic]
model = "claude-sonnet-5"
future_option = "from a newer version"

[brand_new_block]
enabled = true
`)
	cfg, warns, err := Load(Options{GlobalPath: p})
	if err != nil {
		t.Fatalf("an unknown key must not fail startup: %v", err)
	}
	if cfg.Provider.Anthropic.Model != "claude-sonnet-5" {
		t.Error("known keys were not applied alongside the unknown one")
	}
	if len(warns) == 0 {
		t.Fatal("no warning for unknown keys")
	}
	var joined string
	for _, w := range warns {
		joined += w.String() + "\n"
	}
	for _, want := range []string{"future_option", "brand_new_block", p} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings do not mention %q:\n%s", want, joined)
		}
	}
}

// TestMissingFilesAreFine — every key has a default, so no file is required.
func TestMissingFilesAreFine(t *testing.T) {
	cfg, warns, err := Load(Options{
		GlobalPath:  filepath.Join(t.TempDir(), "nope.toml"),
		ProjectPath: filepath.Join(t.TempDir(), "also-nope.toml"),
	})
	if err != nil {
		t.Fatalf("missing config files must not be an error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("missing files should not warn: %v", warns)
	}
	if cfg.Provider.Anthropic.Model != Defaults().Provider.Anthropic.Model {
		t.Error("defaults not applied when no file exists")
	}
}

// TestLoaderNeverReadsEnvironment guards the hermeticity rule: the loader must
// not consult $HOME or XDG variables, so tests never depend on the machine.
func TestLoaderNeverReadsEnvironment(t *testing.T) {
	t.Setenv("HOME", "/nonexistent-home")
	t.Setenv("XDG_CONFIG_HOME", "/nonexistent-xdg")
	cfg, _, err := Load(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Session.StorageDir != Defaults().Session.StorageDir {
		t.Errorf("storage_dir = %q; the loader expanded ~ without being given a home",
			cfg.Session.StorageDir)
	}
}

func TestExpandHome(t *testing.T) {
	cases := []struct{ in, home, want string }{
		{"~/a/b", "/home/me", "/home/me/a/b"},
		{"~", "/home/me", "/home/me"},
		{"/absolute", "/home/me", "/absolute"},
		{"relative/path", "/home/me", "relative/path"},
		{"~notme/x", "/home/me", "~notme/x"}, // ~user is not expanded
	}
	for _, c := range cases {
		if got := ExpandHome(c.in, c.home); got != c.want {
			t.Errorf("ExpandHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidate(t *testing.T) {
	bad := Defaults()
	bad.Provider.Anthropic.Thinking = "sometimes"
	if err := bad.Validate(); err == nil {
		t.Error("invalid thinking level accepted")
	}
	bad = Defaults()
	bad.Policy.DefaultCommandTimeoutSeconds = 0
	if err := bad.Validate(); err == nil {
		t.Error("zero command timeout accepted")
	}
	if err := Defaults().Validate(); err != nil {
		t.Errorf("defaults must validate: %v", err)
	}
}

func TestPathResolution(t *testing.T) {
	if got := GlobalPath("/xdg", "/home/me"); got != "/xdg/kirsch/config.toml" {
		t.Errorf("XDG_CONFIG_HOME ignored: %q", got)
	}
	if got := GlobalPath("", "/home/me"); got != "/home/me/.config/kirsch/config.toml" {
		t.Errorf("home fallback wrong: %q", got)
	}
	if got := GlobalPath("", ""); got != "" {
		t.Errorf("no home should yield no path, got %q", got)
	}
	if got := ProjectPath("/work"); got != "/work/.kirsch/config.toml" {
		t.Errorf("project path wrong: %q", got)
	}
}
