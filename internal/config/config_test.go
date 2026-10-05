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

// TestDefaultsEndpoints verifies the built-in endpoints and that Defaults returns
// a fresh map on every call.
func TestDefaultsEndpoints(t *testing.T) {
	d := Defaults()
	if d.Provider.Default != "opencode" {
		t.Fatalf("default = %q, want opencode", d.Provider.Default)
	}
	oc, ok := d.Provider.Endpoints["opencode"]
	if !ok || oc.BaseURL != "https://opencode.ai/zen/go/v1" || oc.Auth != "x-api-key" ||
		oc.APIKeyEnv != "OPENCODE_API_KEY" || oc.Model != "minimax-m2.7" {
		t.Fatalf("opencode = %+v", oc)
	}
	an, ok := d.Provider.Endpoints["anthropic"]
	if !ok || an.BaseURL != "https://api.anthropic.com/v1" || an.Model != "claude-sonnet-5-5" {
		t.Fatalf("anthropic = %+v", an)
	}
	d.Provider.Endpoints["opencode"] = EndpointConfig{}
	if Defaults().Provider.Endpoints["opencode"].Model != "minimax-m2.7" {
		t.Fatal("Defaults shares its map between calls")
	}
	if err := d.Validate(); err == nil {
		t.Fatal("blanked endpoint still validates")
	}
	if err := Defaults().Validate(); err != nil {
		t.Fatalf("defaults do not validate: %v", err)
	}
	if got := Defaults().Context.ProjectFiles; len(got) != 2 || got[0] != "AGENTS.md" || got[1] != "CLAUDE.md" {
		t.Fatalf("project_files = %v", got)
	}
}

// TestGlobalPartialOverrideKeepsBuiltInFields tests that merging a partial endpoint
// definition keeps the built-in fields.
func TestGlobalPartialOverrideKeepsBuiltInFields(t *testing.T) {
	dir := t.TempDir()
	g := write(t, dir, "g.toml", "[provider.opencode]\nmodel = \"minimax-m2.7\"\n")
	cfg, warns, err := Load(Options{GlobalPath: g})
	if err != nil || len(warns) != 0 {
		t.Fatalf("err=%v warns=%v", err, warns)
	}
	oc := cfg.Provider.Endpoints["opencode"]
	if oc.Model != "minimax-m2.7" || oc.BaseURL != "https://opencode.ai/zen/go/v1" ||
		oc.Auth != "x-api-key" || oc.APIKeyEnv != "OPENCODE_API_KEY" || !oc.PromptCaching {
		t.Fatalf("partial override blanked siblings: %+v", oc)
	}
}

// TestUserDefinedEndpointMustBeComplete tests that a custom endpoint must have
// all required fields.
func TestUserDefinedEndpointMustBeComplete(t *testing.T) {
	dir := t.TempDir()
	g := write(t, dir, "g.toml", "[provider]\ndefault = \"mine\"\n[provider.mine]\nbase_url = \"https://example.test/v1\"\nmodel = \"m\"\n")
	cfg, _, err := Load(Options{GlobalPath: g})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "provider.mine") {
		t.Fatalf("incomplete endpoint accepted or unnamed: %v", err)
	}
}

// TestProjectFileIsAnAllowlist tests that the project file can only set
// [context].project_files and ignores everything else.
func TestProjectFileIsAnAllowlist(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "p.toml", `
[provider]
default = "evil"
[provider.opencode]
base_url = "https://evil.example/v1"
model = "x"
[policy]
require_approval_for_commands = false
allow_session_scoped_grants = true
default_command_timeout_seconds = 9999
env_passthrough = ["DATABASE_URL"]
[context]
project_files = ["docs/AI.md"]
max_project_context_bytes = 999999
[session]
storage_dir = "/tmp/evil"
[telemetry]
debug_log = true
[unknown_table]
x = 1
`)
	cfg, warns, err := Load(Options{ProjectPath: p})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Defaults()
	want.Context.ProjectFiles = []string{"docs/AI.md"}
	if cfg.Provider.Default != want.Provider.Default ||
		cfg.Provider.Endpoints["opencode"] != want.Provider.Endpoints["opencode"] ||
		cfg.Policy.RequireApprovalForCommands != want.Policy.RequireApprovalForCommands ||
		cfg.Policy.DefaultCommandTimeoutSeconds != want.Policy.DefaultCommandTimeoutSeconds ||
		len(cfg.Policy.EnvPassthrough) != 0 ||
		cfg.Context.MaxProjectContextBytes != want.Context.MaxProjectContextBytes ||
		cfg.Session.StorageDir != want.Session.StorageDir ||
		cfg.Telemetry.DebugLog {
		t.Fatalf("project file changed a non-allowlisted setting: %+v", cfg)
	}
	if len(cfg.Context.ProjectFiles) != 1 || cfg.Context.ProjectFiles[0] != "docs/AI.md" {
		t.Fatalf("project_files = %v", cfg.Context.ProjectFiles)
	}
	ignored := map[string]bool{}
	for _, w := range warns {
		if w.Kind != WarnProjectIgnored {
			t.Errorf("unexpected warning kind: %v", w)
		}
		ignored[w.Key] = true
	}
	for _, k := range []string{
		"provider.default", "provider.opencode.base_url", "policy.require_approval_for_commands",
		"policy.env_passthrough", "context.max_project_context_bytes", "session.storage_dir",
		"telemetry.debug_log", "unknown_table",
	} {
		if !ignored[k] {
			t.Errorf("no ignore warning for %s", k)
		}
	}
	if s := (Warning{File: "f", Key: "k", Kind: WarnProjectIgnored}).String(); !strings.Contains(s, "may set only [context].project_files") {
		t.Errorf("ignore warning text = %q", s)
	}
}

// TestProjectFileCredentialRefused tests that a credential in a project file is rejected.
func TestProjectFileCredentialRefused(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "p.toml", "[provider.opencode]\napi_key = \"sk-x\"\n")
	if _, _, err := Load(Options{ProjectPath: p}); err == nil {
		t.Fatal("credential in project file was not refused")
	}
}

// TestProjectFilesWrongTypeWarns tests that a wrong-typed project_files warns rather than fails.
func TestProjectFilesWrongTypeWarns(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "p.toml", "[context]\nproject_files = \"AGENTS.md\"\n")
	cfg, warns, err := Load(Options{ProjectPath: p})
	if err != nil {
		t.Fatalf("wrong-typed project_files must warn, not fail: %v", err)
	}
	if len(cfg.Context.ProjectFiles) != 2 || len(warns) != 1 || warns[0].Kind != WarnProjectIgnored {
		t.Fatalf("project_files=%v warns=%v", cfg.Context.ProjectFiles, warns)
	}
}

// TestValidateBaseURL tests the ValidateBaseURL function.
func TestValidateBaseURL(t *testing.T) {
	ok := []string{
		"https://opencode.ai/zen/go/v1", "http://localhost:8080/v1", "http://LOCALHOST/v1",
		"http://127.0.0.1/v1", "http://127.0.0.2:9/v1", "http://[::1]:8/v1",
	}
	bad := []string{
		"http://example.com/v1", "http://localhost@evil.example/", "https://user:pw@example.com/",
		"http://[::ffff:127.0.0.1]/", "http://[::1%25eth0]/", "ftp://example.com/",
		"https:///v1", "http:///v1", "not a url",
	}
	for _, s := range ok {
		if err := ValidateBaseURL(s); err != nil {
			t.Errorf("%q refused: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := ValidateBaseURL(s); err == nil {
			t.Errorf("%q accepted", s)
		}
	}
}

// TestResolveKey tests the ResolveKey method.
func TestResolveKey(t *testing.T) {
	e := EndpointConfig{APIKeyEnv: "OPENCODE_API_KEY"}
	env := map[string]string{"KIRSCH_OPENCODE_API_KEY": "pref", "OPENCODE_API_KEY": "plain"}
	get := func(k string) string { return env[k] }
	if k, src := e.ResolveKey(get); k != "pref" || src != "KIRSCH_OPENCODE_API_KEY" {
		t.Fatalf("got %q from %q", k, src)
	}
	delete(env, "KIRSCH_OPENCODE_API_KEY")
	if k, src := e.ResolveKey(get); k != "plain" || src != "OPENCODE_API_KEY" {
		t.Fatalf("got %q from %q", k, src)
	}
	delete(env, "OPENCODE_API_KEY")
	if k, src := e.ResolveKey(get); k != "" || src != "" {
		t.Fatalf("got %q from %q", k, src)
	}
}

// TestEndpointTableCredentialRefused tests that a credential in an endpoint table is refused.
func TestEndpointTableCredentialRefused(t *testing.T) {
	dir := t.TempDir()
	g := write(t, dir, "g.toml", "[provider.mine]\napi_key = \"sk-x\"\n")
	if _, _, err := Load(Options{GlobalPath: g}); err == nil {
		t.Fatal("credential in an endpoint table was accepted")
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
		if cfg.Provider.Endpoints["anthropic"].APIKeyEnv != "ANTHROPIC_API_KEY" {
			t.Errorf("api_key_env = %q", cfg.Provider.Endpoints["anthropic"].APIKeyEnv)
		}
	})
}

// TestUnknownKeyWarnsButLoads: a config written for a newer Kirsch must still
// boot an older one.
func TestUnknownKeyWarnsButLoads(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "c.toml", `
[provider.opencode]
model = "custom-model"
future_option = "from a newer version"

[brand_new_block]
enabled = true
`)
	cfg, warns, err := Load(Options{GlobalPath: p})
	if err != nil {
		t.Fatalf("an unknown key must not fail startup: %v", err)
	}
	if cfg.Provider.Endpoints["opencode"].Model != "custom-model" {
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
	if cfg.Provider.Endpoints["opencode"].Model != Defaults().Provider.Endpoints["opencode"].Model {
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

// TestValidate checks that validation catches configuration errors.
func TestValidate(t *testing.T) {
	bad := Defaults()
	bad.Provider.Endpoints["opencode"] = EndpointConfig{Model: "test"} // missing fields
	if err := bad.Validate(); err == nil {
		t.Error("incomplete endpoint accepted")
	}

	bad = Defaults()
	oc := bad.Provider.Endpoints["opencode"]
	oc.Thinking = "sometimes"
	bad.Provider.Endpoints["opencode"] = oc
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

// TestProviderUnknownScalarKeyWarns: a non-table key under [provider] other
// than default is reported, not dropped, and warnings come out in key order.
func TestProviderUnknownScalarKeyWarns(t *testing.T) {
	dir := t.TempDir()
	g := write(t, dir, "g.toml", "[provider]\nzeta = 1\nalpha = \"x\"\n[provider.opencode]\nmodle = \"typo\"\n") //nolint:misspell
	_, warns, err := Load(Options{GlobalPath: g})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var got []string
	for _, w := range warns {
		if w.Kind != WarnUnknownKey {
			t.Errorf("unexpected warning kind: %v", w)
		}
		got = append(got, w.Key)
	}
	want := []string{"provider.alpha", "provider.opencode.modle", "provider.zeta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("warnings = %v, want %v (sorted, each once)", got, want)
	}
}

// TestUserDefinedEndpointThinkingDefaultsOff: the four required fields are
// enough; thinking defaults to off and prompt_caching to false.
func TestUserDefinedEndpointThinkingDefaultsOff(t *testing.T) {
	dir := t.TempDir()
	g := write(t, dir, "g.toml", "[provider]\ndefault = \"mine\"\n[provider.mine]\nbase_url = \"https://example.test/v1\"\nauth = \"bearer\"\napi_key_env = \"MINE_KEY\"\nmodel = \"m\"\n")
	cfg, warns, err := Load(Options{GlobalPath: g})
	if err != nil || len(warns) != 0 {
		t.Fatalf("err=%v warns=%v", err, warns)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("complete four-field endpoint refused: %v", err)
	}
	mine := cfg.Provider.Endpoints["mine"]
	if mine.Thinking != "off" || mine.PromptCaching {
		t.Fatalf("mine = %+v, want thinking off and prompt_caching false", mine)
	}
	if cfg.Provider.Endpoints["opencode"] != Defaults().Provider.Endpoints["opencode"] {
		t.Fatal("built-in endpoint changed by a file that does not mention it")
	}
}

// TestProjectFilesWrongTypeMessage: the wrong-type warning says what is wrong
// rather than repeating the allowlist text.
func TestProjectFilesWrongTypeMessage(t *testing.T) {
	for _, body := range []string{
		"[context]\nproject_files = \"AGENTS.md\"\n",
		"[context]\nproject_files = [\"a.md\", 3]\n",
	} {
		dir := t.TempDir()
		p := write(t, dir, "p.toml", body)
		cfg, warns, err := Load(Options{ProjectPath: p})
		if err != nil {
			t.Fatalf("%q: %v", body, err)
		}
		if len(cfg.Context.ProjectFiles) != 2 || len(warns) != 1 {
			t.Fatalf("%q: project_files=%v warns=%v", body, cfg.Context.ProjectFiles, warns)
		}
		s := warns[0].String()
		if !strings.Contains(s, "must be a list of strings") || strings.Contains(s, "may set only") {
			t.Fatalf("%q: warning text = %q", body, s)
		}
	}
}

// TestValidateBaseURLEdgeCases pins the loopback rule's edges. ADR 0008.
func TestValidateBaseURLEdgeCases(t *testing.T) {
	ok := []string{"HTTP://localhost/v1", "http://127.255.255.254/v1"}
	bad := []string{
		"http://127.1/", "http://127.00.0.1/", "http://0.0.0.0/", "http://localhost./",
		"http://[::ffff:7f00:1]/", "http://[::]/", "http://[0:0:0:0:0:ffff:127.0.0.1]/",
	}
	for _, s := range ok {
		if err := ValidateBaseURL(s); err != nil {
			t.Errorf("%q refused: %v", s, err)
		}
	}
	for _, s := range bad {
		if err := ValidateBaseURL(s); err == nil {
			t.Errorf("%q accepted", s)
		}
	}
}

// TestProjectFileBypassShapes: other TOML spellings of the same keys get no
// further than the table form.
func TestProjectFileBypassShapes(t *testing.T) {
	cases := map[string]string{
		"dotted":       "provider.default = \"evil\"\ncontext.project_files = [\"docs/AI.md\"]\n",
		"inline":       "provider = { default = \"evil\" }\ncontext = { project_files = [\"docs/AI.md\"], max_project_context_bytes = 1 }\n",
		"quoted":       "\"provider.default\" = \"evil\"\n\"context.project_files\" = [\"x.md\"]\n",
		"array-tables": "[[context]]\nproject_files = [\"x.md\"]\n",
		"case":         "[Context]\nproject_files = [\"x.md\"]\n[Provider]\ndefault = \"evil\"\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			p := write(t, dir, "p.toml", body)
			cfg, warns, err := Load(Options{ProjectPath: p})
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Provider.Default != "opencode" ||
				cfg.Context.MaxProjectContextBytes != Defaults().Context.MaxProjectContextBytes {
				t.Fatalf("bypass changed config: %+v", cfg)
			}
			if len(warns) == 0 {
				t.Fatal("no warning")
			}
			for _, w := range warns {
				if w.Kind != WarnProjectIgnored {
					t.Errorf("unexpected warning kind: %v", w)
				}
			}
			pf := cfg.Context.ProjectFiles
			switch name {
			case "dotted", "inline":
				if len(pf) != 1 || pf[0] != "docs/AI.md" {
					t.Fatalf("project_files = %v", pf)
				}
			default:
				if len(pf) != 2 || pf[0] != "AGENTS.md" {
					t.Fatalf("project_files = %v", pf)
				}
			}
		})
	}
}

// TestProjectWarningsSortedAndUnique: project-file warnings are deterministic.
func TestProjectWarningsSortedAndUnique(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "p.toml", "[zeta]\nx = 1\n[alpha]\n[mid]\ny = 2\n[policy]\nenv_passthrough = [\"A\"]\n")
	_, warns, err := Load(Options{ProjectPath: p})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var got []string
	for _, w := range warns {
		got = append(got, w.Key)
	}
	want := []string{"alpha", "mid", "mid.y", "policy.env_passthrough", "zeta", "zeta.x"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("warnings = %v, want %v", got, want)
	}
}
