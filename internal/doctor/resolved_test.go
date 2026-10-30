package doctor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
)

// envOf returns a getenv backed by a fixed map; names not in it read as empty.
func envOf(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// reportText renders results through Report, the way the command prints them.
func reportText(t *testing.T, results []Result) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Report(&buf, "v", results); err != nil {
		// Fixed text: tests that set a key variable call this helper, so no
		// error, result or output is printed from here.
		t.Fatal("Report returned an error (error withheld: a key variable may be set)")
	}
	return buf.String()
}

// one returns the single result with the given check name, failing the test when
// there is not exactly one. Tests that set a key variable call it, so its failure
// text is the check name, the count found and the statuses only: never a detail,
// a result or the results slice.
func one(t *testing.T, results []Result, name string) Result {
	t.Helper()
	got := find(results, name)
	if len(got) != 1 {
		statuses := make([]Status, len(got))
		for i, r := range got {
			statuses[i] = r.Status
		}
		t.Fatalf("results named %q: found %d with statuses %v, want exactly one (details withheld)", name, len(got), statuses)
	}
	return got[0]
}

// defaultEndpoint returns the name of the built-in default endpoint, so config
// files written by these tests name the same table kirsch's defaults use.
func defaultEndpoint() string {
	return config.Defaults().Provider.Default
}

// workspaceDir makes a folder to pass as --workspace, optionally holding an
// AGENTS.md with the given content (skipped when content is empty).
func workspaceDir(t *testing.T, agentsMD string) string {
	t.Helper()
	dir := t.TempDir()
	if agentsMD != "" {
		writeFile(t, filepath.Join(dir, "AGENTS.md"), agentsMD)
	}
	return dir
}

// TestKeyLineNamesWinner covers each combination of KIRSCH_OPENCODE_API_KEY and
// OPENCODE_API_KEY, set and unset: the line names the winning variable (the
// KIRSCH_ one when both are set) or says none, and the status is ok or warn.
func TestKeyLineNamesWinner(t *testing.T) {
	chdirOutsideRepo(t) // no enclosing repository, so no project config is read
	const prefixed, plain = "KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"
	for _, tc := range []struct {
		name   string
		env    map[string]string
		want   string
		status Status
	}{
		{"neither", nil, "endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt", StatusWarn},
		{"prefixed only", map[string]string{prefixed: "v1"}, "endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY", StatusOK},
		{"plain only", map[string]string{plain: "v2"}, "endpoint opencode: key found in OPENCODE_API_KEY", StatusOK},
		{"both", map[string]string{prefixed: "v3", plain: "v4"}, "endpoint opencode: key found in KIRSCH_OPENCODE_API_KEY", StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := Run(Options{LookPath: lookWithout(), Getenv: envOf(tc.env)})
			got := one(t, out.Results, "key")
			// The failure text carries only the expected line: the actual
			// detail is withheld because a set key value must never be printed.
			if got.Detail != tc.want || got.Status != tc.status {
				t.Errorf("key line differs from the expected {%v %q} (actual withheld; status matched: %v)",
					tc.status, tc.want, got.Status == tc.status)
			}
		})
	}
}

// TestKeyLineNilGetenvIsEmptyEnvironment: a nil Getenv reads as no variable set.
func TestKeyLineNilGetenvIsEmptyEnvironment(t *testing.T) {
	chdirOutsideRepo(t)
	out := Run(Options{LookPath: lookWithout()})
	if got := one(t, out.Results, "key"); got.Status != StatusWarn || !strings.Contains(got.Detail, "no key found") {
		t.Errorf("key line = %+v, want a warn saying no key found", got)
	}
}

// TestKeyValueNeverPrinted sets known key values in both variables and asserts
// neither value appears anywhere in the report. Every failure message here is a
// fixed string: none prints the report, a line of it, or a value. The positive
// control runs the same check on a report into which a value is deliberately
// printed, so the check is shown able to see it, and the winner's name is
// asserted present so the line was produced.
func TestKeyValueNeverPrinted(t *testing.T) {
	chdirOutsideRepo(t)
	const v1, v2 = "known-value-aaa-111", "known-value-bbb-222"
	leaks := func(text string) bool {
		return strings.Contains(text, v1) || strings.Contains(text, v2)
	}

	out := Run(Options{LookPath: lookWithout(), Getenv: envOf(map[string]string{
		"KIRSCH_OPENCODE_API_KEY": v1,
		"OPENCODE_API_KEY":        v2,
	})})
	text := reportText(t, out.Results)
	if !strings.Contains(text, "key found in KIRSCH_OPENCODE_API_KEY") {
		t.Fatal("the key line is missing, so the absence check below would prove nothing")
	}
	if leaks(text) {
		t.Error("the report contains a key value (value withheld from this message)")
	}

	// Positive control: the same check flags a report that does print a value.
	bad := append([]Result{{StatusOK, "key", "found " + v1}}, out.Results...)
	if !leaks(reportText(t, bad)) {
		t.Fatal("control: the leak check did not see a value that was printed")
	}
}

// TestKeyLineWithholdsNonNameAPIKeyEnv: api_key_env is validated only as
// non-empty, so a key pasted into it would be printed by a line that echoes the
// variable name. The line withholds a setting that does not look like a variable
// name (envNameRe), on the no-key branch and on the found branch, for three
// key shapes: hyphenated, underscore-and-mixed-case (name-shaped under the old
// pattern) and a long upper-case token (over the length limit). It still prints
// a valid name, including one at the length limit (the positive controls).
// Failure messages are fixed strings: no detail, no value and no setting is
// printed.
func TestKeyLineWithholdsNonNameAPIKeyEnv(t *testing.T) {
	chdirOutsideRepo(t)
	const envValue = "found-branch-value-ccc"
	ep := defaultEndpoint()
	cfgFor := func(name string) string {
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, "[provider."+ep+"]\napi_key_env = \""+name+"\"\n")
		return path
	}

	for _, shape := range []struct{ name, pasted string }{
		{"hyphenated", "sk-live-Zq81xPasted-0042"},
		{"underscore and mixed case", "Sk_Live_Zq81xPasted_0042"},
		{"long upper case", "SK_" + strings.Repeat("Z", 70)},
	} {
		pasted := shape.pasted
		leaks := func(text string) bool {
			return strings.Contains(text, pasted) || strings.Contains(text, envValue)
		}
		for _, tc := range []struct {
			name string
			env  map[string]string
		}{
			{"no key branch", nil},
			{"found branch", map[string]string{"KIRSCH_" + pasted: envValue, pasted: envValue}},
		} {
			t.Run(shape.name+"/"+tc.name, func(t *testing.T) {
				out := Run(Options{LookPath: lookWithout(), GlobalConfigPath: cfgFor(pasted), Getenv: envOf(tc.env)})
				if !out.ConfigLoaded {
					t.Fatal("control: the config with a key-shaped api_key_env did not load")
				}
				text := reportText(t, out.Results)
				got := one(t, out.Results, "key")
				if got.Status != StatusWarn || !strings.Contains(got.Detail, "does not look like an environment variable name") {
					t.Error("key line is not the withheld-setting warn (actual line withheld from this message)")
				}
				if leaks(text) {
					t.Error("the report contains the api_key_env setting or a key value (withheld from this message)")
				}
				// Control: the same check sees the setting when it is printed.
				if !leaks(reportText(t, append([]Result{{StatusWarn, "key", "set " + pasted}}, out.Results...))) {
					t.Fatal("control: the leak check did not see a printed setting")
				}
			})
		}
	}

	for _, valid := range []string{"MY_KEY_VAR", strings.Repeat("V", 64)} {
		t.Run("control: a valid name is still printed/"+fmt.Sprint(len(valid)), func(t *testing.T) {
			out := Run(Options{LookPath: lookWithout(), GlobalConfigPath: cfgFor(valid)})
			got := one(t, out.Results, "key")
			want := fmt.Sprintf("endpoint %s: no key found (set KIRSCH_%s or %s); kirsch starts but cannot send a prompt", ep, valid, valid)
			if got.Status != StatusWarn || got.Detail != want {
				t.Errorf("key line differs from the expected {warn %q}", want)
			}
		})
	}
}

// TestModelLine: a known model prints the endpoint, id and window; an unknown id
// prints "not in the table" and the conservative fallback window.
func TestModelLine(t *testing.T) {
	t.Run("known", func(t *testing.T) {
		chdirOutsideRepo(t)
		// The expected line is derived from the built-in default and the model
		// table, not pinned to a model id or window.
		cfg := config.Defaults()
		epName := cfg.Provider.Default
		info := provider.LookupModel(cfg.Provider.Endpoints[epName].Model)
		if !info.Known {
			t.Fatal("control: the default model is not in the table")
		}
		out := Run(Options{LookPath: lookWithout()})
		got := one(t, out.Results, "model")
		want := fmt.Sprintf("endpoint %s, model %s: context window %d tokens", epName, info.ID, info.ContextWindow)
		if got.Status != StatusOK || got.Detail != want {
			t.Errorf("model line = {%v %q}, want {ok %q}", got.Status, got.Detail, want)
		}
	})
	t.Run("not in the table", func(t *testing.T) {
		chdirOutsideRepo(t)
		path := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, path, "[provider."+defaultEndpoint()+"]\nmodel = \"no-such-model\"\n")
		out := Run(Options{LookPath: lookWithout(), GlobalConfigPath: path})
		got := one(t, out.Results, "model")
		want := fmt.Sprintf("endpoint %s, model no-such-model: not in the table; conservative fallback, context window %d tokens",
			defaultEndpoint(), provider.ConservativeBudget())
		if got.Status != StatusWarn || got.Detail != want {
			t.Errorf("model line = {%v %q}, want {warn %q}", got.Status, got.Detail, want)
		}
		if provider.LookupModel("no-such-model").Known {
			t.Fatal("control: the test model id is in the table")
		}
	})
}

// TestContextLine covers the context line per project-file situation: AGENTS.md
// present (names it and its size); absent (says none); a candidate that is a
// directory (NOTES.md, listed first); a candidate Resolve refuses (a denylisted
// .env and an absolute path, listed first); an empty project_files list; and a
// candidate that exists but cannot be read (chmod 000). An unusable candidate
// listed ahead of AGENTS.md is a warn that still names the file that was chosen.
func TestContextLine(t *testing.T) {
	const body = "# rules\nbe brief\n"
	t.Run("present", func(t *testing.T) {
		ws := workspaceDir(t, body)
		got := one(t, Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws}).Results, "context")
		want := fmt.Sprintf("AGENTS.md, %d bytes", len(body))
		if got.Status != StatusOK || got.Detail != want {
			t.Errorf("context line = {%v %q}, want {ok %q}", got.Status, got.Detail, want)
		}
	})
	t.Run("absent", func(t *testing.T) {
		ws := workspaceDir(t, "")
		got := one(t, Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws}).Results, "context")
		if got.Status != StatusOK || !strings.HasPrefix(got.Detail, "none") || strings.Contains(got.Detail, "bytes") {
			t.Errorf("context line = %+v, want an ok line starting with none", got)
		}
	})
	t.Run("candidate is a directory", func(t *testing.T) {
		ws := workspaceDir(t, body)
		if err := os.Mkdir(filepath.Join(ws, "NOTES.md"), 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, cfg, "[context]\nproject_files = [\"NOTES.md\", \"AGENTS.md\"]\n")
		got := one(t, Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, GlobalConfigPath: cfg}).Results, "context")
		if got.Status != StatusWarn || !strings.HasPrefix(got.Detail, "AGENTS.md, ") ||
			!strings.Contains(got.Detail, "NOTES.md") {
			t.Errorf("context line = %+v, want a warn naming AGENTS.md and the refused NOTES.md", got)
		}
	})
	t.Run("candidate is refused", func(t *testing.T) {
		// Resolve refuses a denylisted name (.env) and an absolute path before it
		// looks at the disk; both are listed ahead of AGENTS.md, which is chosen.
		ws := workspaceDir(t, body)
		writeFile(t, filepath.Join(ws, ".env"), "X=1\n")
		abs := filepath.Join(t.TempDir(), "abs.md")
		cfg := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, cfg, "[context]\nproject_files = [\".env\", \""+abs+"\", \"AGENTS.md\"]\n")
		out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, GlobalConfigPath: cfg})
		if !out.ConfigLoaded {
			t.Fatal("control: the config did not load")
		}
		got := one(t, out.Results, "context")
		if got.Status != StatusWarn || !strings.HasPrefix(got.Detail, "AGENTS.md, ") ||
			!strings.Contains(got.Detail, `".env" refused`) || !strings.Contains(got.Detail, abs+`" refused`) {
			t.Errorf("context line = %+v, want a warn naming AGENTS.md and both refused candidates", got)
		}
	})
	t.Run("no project files configured", func(t *testing.T) {
		ws := workspaceDir(t, body)
		cfg := filepath.Join(t.TempDir(), "config.toml")
		writeFile(t, cfg, "[context]\nproject_files = []\n")
		out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, GlobalConfigPath: cfg})
		if !out.ConfigLoaded || len(out.Config.Context.ProjectFiles) != 0 {
			t.Fatal("control: the config did not load with an empty project_files")
		}
		got := one(t, out.Results, "context")
		if got.Status != StatusOK || got.Detail != "none; no project files configured" {
			t.Errorf("context line = %+v, want {ok \"none; no project files configured\"}", got)
		}
	})
	t.Run("candidate exists but cannot be read", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: chmod 000 does not stop a read, so the unreadable case cannot be built")
		}
		ws := workspaceDir(t, body)
		// Control: the same file, readable, is reported with its size.
		got := one(t, Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws}).Results, "context")
		if want := fmt.Sprintf("AGENTS.md, %d bytes", len(body)); got.Status != StatusOK || got.Detail != want {
			t.Fatalf("control: the readable file is not reported as %q: %+v", want, got)
		}

		target := filepath.Join(ws, "AGENTS.md")
		if err := os.Chmod(target, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(target, 0o600) })
		if f, err := os.Open(target); err == nil {
			_ = f.Close()
			t.Skip("this filesystem still lets the file be opened after chmod 000")
		}

		got = one(t, Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws}).Results, "context")
		if got.Status != StatusWarn || !strings.HasPrefix(got.Detail, "a candidate could not be read: ") ||
			strings.HasPrefix(got.Detail, "none") || strings.Contains(got.Detail, "looked for") {
			t.Errorf("context line = %+v, want a warn starting %q and not claiming none or looked for", got, "a candidate could not be read: ")
		}
	})
}

// TestBackendLineUnderPathShim: with rg on PATH the backend line reads ripgrep,
// without it pure-go. PATH is a shim of the real one (DIR-009), never emptied;
// the with-rg case adds a stub rg to the shim, so neither case needs ripgrep
// installed. Controls: LookPath agrees with each case before Run is called.
func TestBackendLineUnderPathShim(t *testing.T) {
	chdirOutsideRepo(t)
	shim := shimPathWithout(t, "rg")
	if _, err := exec.LookPath("ls"); err != nil {
		t.Fatalf("control: ls not found under the shim: %v", err)
	}

	if _, err := exec.LookPath("rg"); err == nil {
		t.Fatal("control: rg found under the shim before the stub was added")
	}
	got := one(t, Run(Options{}).Results, "backend")
	if got.Status != StatusWarn || got.Detail != "search_code uses pure-go (rg is not on PATH)" {
		t.Errorf("backend line without rg = {%v %q}, want the pure-go warn", got.Status, got.Detail)
	}

	writeFile(t, filepath.Join(shim, "rg"), "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(shim, "rg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("rg"); err != nil {
		t.Fatalf("control: stub rg not found after adding it: %v", err)
	}
	got = one(t, Run(Options{}).Results, "backend")
	if got.Status != StatusOK || got.Detail != "search_code uses ripgrep" {
		t.Errorf("backend line with rg = {%v %q}, want the ripgrep ok", got.Status, got.Detail)
	}
}

// TestResolvedLinesSkippedWhenConfigDidNotLoad: with a config that fails to
// parse, the key, model and context lines each read exactly "skipped: config did
// not load" (a warn), even though a key variable is set and a workspace with
// AGENTS.md is given, so no default is guessed. The backend line is still
// produced. The control: the same call with a good config produces real lines.
func TestResolvedLinesSkippedWhenConfigDidNotLoad(t *testing.T) {
	ws := workspaceDir(t, "# rules\n")
	env := envOf(map[string]string{"KIRSCH_OPENCODE_API_KEY": "skipped-case-value"})
	good := Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, Getenv: env})
	for _, name := range []string{"key", "model", "context"} {
		if got := one(t, good.Results, name); strings.HasPrefix(got.Detail, "skipped") {
			t.Fatalf("control: %s line is skipped with a good config (line withheld: a key variable is set)", name)
		}
	}

	bad := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, bad, "this is = not [valid toml\n")
	out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: ws, GlobalConfigPath: bad, Getenv: env})
	if out.ConfigLoaded {
		t.Fatal("control: the broken config loaded")
	}
	for _, name := range []string{"key", "model", "context"} {
		got := one(t, out.Results, name)
		if got.Status != StatusWarn || got.Detail != "skipped: config did not load" {
			t.Errorf("%s line differs from the expected {warn \"skipped: config did not load\"} (actual withheld: a key variable is set)", name)
		}
	}
	if got := one(t, out.Results, "backend"); strings.HasPrefix(got.Detail, "skipped") {
		t.Error("backend line is skipped, want a real line (it needs no config; line withheld: a key variable is set)")
	}
}

// TestContextLineSkippedWithoutWorkspace: a --workspace that does not exist
// leaves the config loaded but no workspace, so the context line is skipped for
// that reason while the key and model lines are real.
func TestContextLineSkippedWithoutWorkspace(t *testing.T) {
	out := Run(Options{LookPath: lookWithout(), WorkspaceFlag: filepath.Join(t.TempDir(), "missing")})
	if !out.ConfigLoaded {
		t.Fatal("control: config did not load")
	}
	if got := one(t, out.Results, "context"); got.Status != StatusWarn || got.Detail != "skipped: no workspace" {
		t.Errorf("context line = %+v, want {warn \"skipped: no workspace\"}", got)
	}
	if got := one(t, out.Results, "model"); strings.HasPrefix(got.Detail, "skipped") {
		t.Errorf("model line = %+v, want a real line", got)
	}
}

// TestResolvedLinesNeverFail: none of the four lines can stop kirsch starting
// (O5), so across the cases above none is a fail.
func TestResolvedLinesNeverFail(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, bad, "this is = not [valid toml\n")
	chdirOutsideRepo(t)
	unknown := filepath.Join(t.TempDir(), "unknown.toml")
	writeFile(t, unknown, "[provider."+defaultEndpoint()+"]\nmodel = \"no-such-model\"\n")
	for _, o := range []Options{
		{LookPath: lookWithout()},
		{LookPath: lookWithout(), GlobalConfigPath: bad},
		{LookPath: lookWithout(), GlobalConfigPath: unknown},
		{LookPath: lookWithout(), WorkspaceFlag: filepath.Join(t.TempDir(), "missing")},
	} {
		out := Run(o)
		for _, name := range []string{"key", "backend", "model", "context"} {
			if got := one(t, out.Results, name); got.Status == StatusFail {
				t.Errorf("%s line failed: %+v", name, got)
			}
		}
	}
}
