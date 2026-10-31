package doctor

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// lineGrammar is the shape every report line between the first and the last
// must have: "<status>  <check>: <detail>" on one line.
var lineGrammar = regexp.MustCompile(`^(ok|warn|fail)  [a-z]+: \S.*$`)

// lookWithout returns a lookPath that finds every tool except the named ones.
// It is an injected stand-in for PATH, used where the probe logic is under test;
// TestRunDefaultLookPathUsesPATH covers the real exec.LookPath through a shim.
func lookWithout(missing ...string) func(string) (string, error) {
	return func(file string) (string, error) {
		for _, m := range missing {
			if file == m {
				return "", exec.ErrNotFound
			}
		}
		return "/fake/bin/" + file, nil
	}
}

// find returns the results with the given check name.
func find(results []Result, name string) []Result {
	var out []Result
	for _, r := range results {
		if r.Name == name {
			out = append(out, r)
		}
	}
	return out
}

// hasDetail reports whether any result's detail contains want.
func hasDetail(results []Result, want string) bool {
	for _, r := range results {
		if strings.Contains(r.Detail, want) {
			return true
		}
	}
	return false
}

// writeFile writes content to path, creating its folder.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// gitRepo creates a Git repository in a temp dir and returns its path. It skips
// the test when git is not installed.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed; cannot create a repository")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

// chdirOutsideRepo makes a fresh temp dir the working directory for the test and
// sets GIT_CEILING_DIRECTORIES so git stops looking for a repository above it
// (and above each folder in others), whatever the temp root sits inside. It
// returns the new working directory.
func chdirOutsideRepo(t *testing.T, others ...string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	ceilings := []string{filepath.Dir(dir)}
	for _, o := range others {
		ceilings = append(ceilings, filepath.Dir(o))
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", strings.Join(ceilings, string(os.PathListSeparator)))
	return dir
}

// shimPathWithout builds a temp dir of symlinks to every executable on the
// current PATH except the named tool, points PATH at it for the test, and
// returns the shim dir. PATH is never emptied (DIR-009): the other tools stay
// findable, so a failure here is about the one missing tool.
func shimPathWithout(t *testing.T, tool string) string {
	t.Helper()
	shim := t.TempDir()
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Name() == tool {
				continue
			}
			full := filepath.Join(dir, e.Name())
			st, err := os.Stat(full) // follows symlinks
			if err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
				continue
			}
			// The first PATH entry wins, as it does for the shell.
			_ = os.Symlink(full, filepath.Join(shim, e.Name()))
		}
	}
	t.Setenv("PATH", shim)
	return shim
}

// TestRunRGMissing: a missing rg is a warn line that names the pure-Go
// fallback, git stays ok, and the control (rg present) is an ok line instead.
func TestRunRGMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	out := Run(Options{LookPath: lookWithout("rg")})
	rg := find(out.Results, "rg")
	if len(rg) != 1 || rg[0].Status != StatusWarn || !strings.Contains(rg[0].Detail, "pure-Go fallback") {
		t.Errorf("rg results = %+v, want one warn naming the pure-Go fallback", rg)
	}
	if git := find(out.Results, "git"); len(git) != 1 || git[0].Status != StatusOK {
		t.Errorf("git results = %+v, want one ok line (control)", git)
	}

	ctl := Run(Options{LookPath: lookWithout()})
	if rg := find(ctl.Results, "rg"); len(rg) != 1 || rg[0].Status != StatusOK {
		t.Errorf("control rg results = %+v, want one ok line", rg)
	}
}

// TestRunGitMissing: a missing git leaves the repository check silent and its
// status follows kirsch's start path. With no --workspace kirsch cannot find the
// repository without git and refuses to start, so the git line fails; with an
// explicit --workspace kirsch starts with its Git features off, so the git line
// only warns. In neither case does the repository check claim "not in a Git
// repository". The control, the same call with git found, does produce a
// repository line, so the absence is not just a silent repository check: it is
// the fail line for no flag and the warn line for a non-repository --workspace.
func TestRunGitMissing(t *testing.T) {
	notRepo := t.TempDir()
	chdirOutsideRepo(t, notRepo)
	for _, tc := range []struct {
		name, flag  string
		want        Status // status of the git line, and of the control repository line
		controlText string // text the control repository line contains
	}{
		{"no flag", "", StatusFail, "not in a Git repository"},
		{"workspace flag", notRepo, StatusWarn, "is not a Git repository"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := Run(Options{WorkspaceFlag: tc.flag, LookPath: lookWithout("git")})
			git := find(out.Results, "git")
			if len(git) != 1 || git[0].Status != tc.want {
				t.Fatalf("git results = %+v, want one %v line", git, tc.want)
			}
			if hasDetail(out.Results, "not in a Git repository") || hasDetail(out.Results, "is not a Git repository") {
				t.Errorf("git missing was reported as not in a repository: %+v", out.Results)
			}
			if repo := find(out.Results, "repository"); len(repo) != 0 {
				t.Errorf("repository lines = %+v, want none when git is missing", repo)
			}

			ctl := Run(Options{WorkspaceFlag: tc.flag, LookPath: lookWithout()})
			repo := find(ctl.Results, "repository")
			if len(repo) != 1 || repo[0].Status != tc.want || !strings.Contains(repo[0].Detail, tc.controlText) {
				t.Errorf("control: repository results = %+v, want one %v line containing %q",
					repo, tc.want, tc.controlText)
			}
		})
	}
}

// TestRunNonGitWorkspaceVersusNoFlag is the control pair for the start-path rule.
// A non-repository folder given as --workspace is a warn, because kirsch starts
// in it with its Git features off; the same kind of folder as the working
// directory with no flag is a fail, because kirsch refuses to start. Statuses
// and the Failed verdict (the exit code) must differ between the two.
func TestRunNonGitWorkspaceVersusNoFlag(t *testing.T) {
	notRepo := t.TempDir()
	chdirOutsideRepo(t, notRepo)

	// HomeDir is a fixture home so the state line has a state folder to check;
	// with no home it fails, which is not what this test is about.
	flagged := Run(Options{WorkspaceFlag: notRepo, LookPath: lookWithout(), HomeDir: t.TempDir()})
	fr := find(flagged.Results, "repository")
	if len(fr) != 1 || fr[0].Status != StatusWarn {
		t.Fatalf("--workspace non-repository: repository results = %+v, want one warn", fr)
	}
	if !strings.Contains(fr[0].Detail, notRepo) || !strings.Contains(fr[0].Detail, "Git features are off") {
		t.Errorf("warn detail = %q, want the folder %q and that the Git features are off", fr[0].Detail, notRepo)
	}
	// Workspace.DirtyCount is behind IsGit, so it is among the features named.
	if !strings.Contains(fr[0].Detail, "dirty count") {
		t.Errorf("warn detail = %q, want it to name the dirty count", fr[0].Detail)
	}
	if Failed(flagged.Results) {
		t.Errorf("a non-repository --workspace must not fail doctor: %+v", flagged.Results)
	}
	if flagged.Workspace == nil || flagged.Workspace.IsGit {
		t.Errorf("Workspace = %+v, want non-nil with IsGit false", flagged.Workspace)
	}

	bare := Run(Options{LookPath: lookWithout()})
	br := find(bare.Results, "repository")
	if len(br) != 1 || br[0].Status != StatusFail || br[0].Detail != "not in a Git repository" {
		t.Fatalf("no flag, non-repository cwd: repository results = %+v, want one fail", br)
	}
	if !Failed(bare.Results) {
		t.Errorf("no --workspace outside a repository must fail doctor: %+v", bare.Results)
	}
}

// TestRunNotInRepositoryRunsOtherChecks: outside a repository the repository
// check fails and git, rg and config all still report. The control is a real
// repository, where the same call reports the root as ok.
func TestRunNotInRepositoryRunsOtherChecks(t *testing.T) {
	chdirOutsideRepo(t)
	out := Run(Options{LookPath: lookWithout()})
	repo := find(out.Results, "repository")
	if len(repo) != 1 || repo[0].Status != StatusFail || repo[0].Detail != "not in a Git repository" {
		t.Errorf("repository results = %+v, want one fail \"not in a Git repository\"", repo)
	}
	for _, name := range []string{"git", "rg", "config"} {
		if len(find(out.Results, name)) == 0 {
			t.Errorf("no %q line outside a repository: %+v", name, out.Results)
		}
	}
	if !out.ConfigLoaded {
		t.Errorf("ConfigLoaded = false, want true for default config")
	}
	// No workspace was found, so no project file was looked for; the line must
	// say so. It must not claim a global file was loaded either: Options names
	// none here, so none was read.
	cfg := find(out.Results, "config")
	if len(cfg) != 1 || cfg[0].Status != StatusOK ||
		cfg[0].Detail != "settings valid; project file not read (no workspace)" {
		t.Errorf("config results = %+v, want one ok line saying the project file was not read", cfg)
	}

	root := gitRepo(t)
	ctl := Run(Options{WorkspaceFlag: root, LookPath: lookWithout()})
	r := find(ctl.Results, "repository")
	if len(r) != 1 || r[0].Status != StatusOK || r[0].Detail != root {
		t.Errorf("control repository results = %+v, want one ok line with %q", r, root)
	}
	if c := find(ctl.Results, "config"); len(c) != 1 || c[0].Detail != "settings valid" {
		t.Errorf("control config results = %+v, want the plain \"settings valid\" line", c)
	}
}

// TestRunWorkspaceMissingDirectory: an explicit --workspace that does not exist
// is a repository failure carrying Detect's error, not "not in a Git repository".
func TestRunWorkspaceMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	out := Run(Options{WorkspaceFlag: missing, LookPath: lookWithout()})
	repo := find(out.Results, "repository")
	if len(repo) != 1 || repo[0].Status != StatusFail || !strings.Contains(repo[0].Detail, missing) {
		t.Errorf("repository results = %+v, want one fail naming %q", repo, missing)
	}
	if hasDetail(repo, "not in a Git repository") {
		t.Errorf("a missing --workspace was reported as not in a repository: %+v", repo)
	}
	if out.Workspace != nil {
		t.Errorf("Workspace = %+v, want nil", out.Workspace)
	}
}

// TestRunBrokenGlobalConfig: a config that is not valid TOML is one fail line
// that names the file and the parse error; every other check still reports and
// ConfigLoaded stays false.
func TestRunBrokenGlobalConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, "this = [is not toml\n")
	out := Run(Options{GlobalConfigPath: path, LookPath: lookWithout()})
	cfg := find(out.Results, "config")
	if len(cfg) != 1 || cfg[0].Status != StatusFail {
		t.Fatalf("config results = %+v, want one fail", cfg)
	}
	if !strings.Contains(cfg[0].Detail, path) || !strings.Contains(cfg[0].Detail, "parse") {
		t.Errorf("config detail = %q, want the file %q and the parse error", cfg[0].Detail, path)
	}
	if out.ConfigLoaded {
		t.Error("ConfigLoaded = true after a parse error")
	}
	for _, name := range []string{"git", "rg", "repository"} {
		if len(find(out.Results, name)) == 0 {
			t.Errorf("no %q line after a config failure: %+v", name, out.Results)
		}
	}

	// Control: the same call with no broken file loads.
	if ctl := Run(Options{LookPath: lookWithout()}); !ctl.ConfigLoaded {
		t.Error("control: ConfigLoaded = false without a broken file")
	}
}

// TestRunBrokenProjectConfig: the project config under the workspace root is
// read too, and a break in it names that file.
func TestRunBrokenProjectConfig(t *testing.T) {
	root := gitRepo(t)
	path := filepath.Join(root, ".kirsch", "config.toml")
	writeFile(t, path, "this = [is not toml\n")
	out := Run(Options{WorkspaceFlag: root, LookPath: lookWithout()})
	cfg := find(out.Results, "config")
	if len(cfg) != 1 || cfg[0].Status != StatusFail || !strings.Contains(cfg[0].Detail, path) {
		t.Errorf("config results = %+v, want one fail naming %q", cfg, path)
	}
}

// TestRunInvalidConfigValues: config that parses but fails Validate is a fail
// line naming the files consulted and the validation error.
func TestRunInvalidConfigValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, "[provider]\ndefault = \"nowhere\"\n")
	out := Run(Options{GlobalConfigPath: path, LookPath: lookWithout()})
	cfg := find(out.Results, "config")
	if len(cfg) != 1 || cfg[0].Status != StatusFail ||
		!strings.Contains(cfg[0].Detail, path) || !strings.Contains(cfg[0].Detail, "nowhere") {
		t.Errorf("config results = %+v, want one fail naming %q and the bad value", cfg, path)
	}
	if out.ConfigLoaded {
		t.Error("ConfigLoaded = true after a validation error")
	}
}

// TestRunUnknownConfigKey: an unknown key is a warn line that names the file and
// key, and config still counts as loaded. The control is a file with only known
// keys, which has no warn line.
func TestRunUnknownConfigKey(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.toml")
	writeFile(t, bad, "bogus_setting = 1\n")
	out := Run(Options{GlobalConfigPath: bad, LookPath: lookWithout()})
	var warns []Result
	for _, r := range find(out.Results, "config") {
		if r.Status == StatusWarn {
			warns = append(warns, r)
		}
	}
	if len(warns) != 1 || !strings.Contains(warns[0].Detail, bad) || !strings.Contains(warns[0].Detail, "bogus_setting") {
		t.Errorf("config warns = %+v, want one naming %q and bogus_setting", warns, bad)
	}
	if !out.ConfigLoaded {
		t.Error("ConfigLoaded = false for a config with only an unknown key")
	}

	good := filepath.Join(dir, "good.toml")
	writeFile(t, good, "[session]\nauto_resume = false\n")
	ctl := Run(Options{GlobalConfigPath: good, LookPath: lookWithout()})
	for _, r := range ctl.Results {
		if r.Status == StatusWarn && r.Name == "config" {
			t.Errorf("control: unexpected config warn %+v", r)
		}
	}
}

// TestRunProjectConfigIgnoredKeyWarns: a key a project config may not set (here
// telemetry.debug_log) is a WarnProjectIgnored warning, shown as a warn line that
// names the project file and the key, and config still counts as loaded. The
// control is a project file that sets only [context].project_files, which gets
// no warn line.
func TestRunProjectConfigIgnoredKeyWarns(t *testing.T) {
	root := gitRepo(t)
	path := filepath.Join(root, ".kirsch", "config.toml")
	writeFile(t, path, "[telemetry]\ndebug_log = true\n")
	// HomeDir as in TestRunNonGitWorkspaceVersusNoFlag: the state line needs one.
	out := Run(Options{WorkspaceFlag: root, LookPath: lookWithout(), HomeDir: t.TempDir()})
	var warns []Result
	for _, r := range find(out.Results, "config") {
		if r.Status == StatusWarn {
			warns = append(warns, r)
		}
	}
	if len(warns) != 1 || !strings.Contains(warns[0].Detail, path) ||
		!strings.Contains(warns[0].Detail, "telemetry.debug_log") || !strings.Contains(warns[0].Detail, "ignored") {
		t.Errorf("config warns = %+v, want one naming %q and the ignored key", warns, path)
	}
	if !out.ConfigLoaded {
		t.Error("ConfigLoaded = false for a project config with an ignored key")
	}
	if Failed(out.Results) {
		t.Errorf("an ignored project key must only warn: %+v", out.Results)
	}

	writeFile(t, path, "[context]\nproject_files = [\"AGENTS.md\"]\n")
	ctl := Run(Options{WorkspaceFlag: root, LookPath: lookWithout()})
	for _, r := range ctl.Results {
		if r.Status == StatusWarn && r.Name == "config" {
			t.Errorf("control: unexpected config warn %+v", r)
		}
	}
}

// TestRunConfigLoadedRecord: ConfigLoaded and Config carry the loaded values for
// the probes that follow.
func TestRunConfigLoadedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, "[session]\nauto_resume = false\n")
	out := Run(Options{GlobalConfigPath: path, LookPath: lookWithout()})
	if !out.ConfigLoaded {
		t.Fatal("ConfigLoaded = false")
	}
	if out.Config.Session.AutoResume {
		t.Error("Config.Session.AutoResume = true, want the file's false")
	}
}

// TestRunMultiLineConfigErrorStaysOnOneLine drives a config error whose message
// spans several lines (a credential-looking key) through Run and Report, and
// checks every report line still has the one-line grammar.
func TestRunMultiLineConfigErrorStaysOnOneLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeFile(t, path, "api_key = \"abc\"\n")
	out := Run(Options{GlobalConfigPath: path, LookPath: lookWithout()})
	cfg := find(out.Results, "config")
	if len(cfg) != 1 || !strings.Contains(cfg[0].Detail, "\n") {
		t.Fatalf("config results = %+v, want one result whose raw detail spans lines (the test input)", cfg)
	}
	var buf bytes.Buffer
	if err := Report(&buf, "v", out.Results); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if want := 1 + len(out.Results) + 1; len(lines) != want {
		t.Errorf("report has %d lines, want %d:\n%s", len(lines), want, buf.String())
	}
	for _, l := range lines[1 : len(lines)-1] {
		if !lineGrammar.MatchString(l) {
			t.Errorf("line %q does not match the report grammar", l)
		}
	}
}

// TestRunDefaultLookPathUsesPATH leaves Options.LookPath nil so Run uses
// exec.LookPath, and removes one tool from PATH with a symlink shim. Controls in
// the same test: under the shim LookPath finds ls, a tool that is always present
// on the platforms CI runs on, and does not find the removed one; and when the
// other probed tool (git or rg) is installed at all, its line stays ok. The
// removed-tool assertions never depend on rg or git being installed.
func TestRunDefaultLookPathUsesPATH(t *testing.T) {
	for _, tc := range []struct {
		missing, other string
		want           Status
	}{
		{"rg", "git", StatusWarn},
		{"git", "rg", StatusFail},
	} {
		t.Run("without "+tc.missing, func(t *testing.T) {
			_, otherErr := exec.LookPath(tc.other) // before the shim
			t.Chdir(t.TempDir())
			shimPathWithout(t, tc.missing)
			if _, err := exec.LookPath("ls"); err != nil {
				t.Fatalf("control: LookPath(ls) under the shim = %v, want found", err)
			}
			if _, err := exec.LookPath(tc.missing); err == nil {
				t.Fatalf("control: LookPath(%q) under the shim succeeded, want not found", tc.missing)
			}
			out := Run(Options{})
			got := find(out.Results, tc.missing)
			if len(got) != 1 || got[0].Status != tc.want {
				t.Errorf("%s results = %+v, want one %v line", tc.missing, got, tc.want)
			}
			if otherErr == nil {
				other := find(out.Results, tc.other)
				if len(other) != 1 || other[0].Status != StatusOK {
					t.Errorf("%s results = %+v, want one ok line", tc.other, other)
				}
			}
		})
	}
}

// TestRunNeverReturnsUnknownStatus guards the probe output: every result carries
// one of the three known statuses.
func TestRunNeverReturnsUnknownStatus(t *testing.T) {
	t.Chdir(t.TempDir())
	out := Run(Options{LookPath: lookWithout("rg", "git")})
	if len(out.Results) == 0 {
		t.Fatal("no results")
	}
	for _, r := range out.Results {
		if r.Status.String() == "unknown" {
			t.Errorf("result %+v has an unknown status", r)
		}
	}
}
