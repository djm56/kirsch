package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/doctor"
	"github.com/djm56/kirsch/internal/session"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Session.StorageDir = filepath.Join(t.TempDir(), "sessions")
	return app.New(ws, cfg, telemetry.Disabled())
}

// newTestAppWithRoot returns a test app, the session state root, and the
// canonical workspace root so tests can seed broken session indexes.
func newTestAppWithRoot(t *testing.T) (*app.App, string, string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	stateRoot := t.TempDir()
	cfg.Session.StorageDir = filepath.Join(stateRoot, "sessions")
	return app.New(ws, cfg, telemetry.Disabled()), stateRoot, ws.CanonicalRoot
}

// noticeCollector captures the first tui.NoticeMsg sent through App.SetSendFn.
type noticeCollector struct {
	ch chan tui.NoticeMsg
}

func newNoticeCollector() *noticeCollector {
	return &noticeCollector{ch: make(chan tui.NoticeMsg, 1)}
}

func (c *noticeCollector) send(msg any) {
	if nm, ok := msg.(tui.NoticeMsg); ok {
		select {
		case c.ch <- nm:
		default:
		}
	}
}

func (c *noticeCollector) wait(t *testing.T, d time.Duration) tui.NoticeMsg {
	t.Helper()
	select {
	case nm := <-c.ch:
		return nm
	case <-time.After(d):
		t.Fatalf("timed out waiting for notice")
	}
	return tui.NoticeMsg{}
}

// TestChooseSessionInitResume verifies that the explicit `resume` subcommand
// routes to ResumeLastSession and surfaces ErrNoSession when there is none.
func TestChooseSessionInitResume(t *testing.T) {
	a := newTestApp(t)
	defer a.Close()

	init := chooseSessionInit("resume", false, true)
	if err := init(a); !errors.Is(err, session.ErrNoSession) {
		t.Fatalf("init error = %v, want ErrNoSession", err)
	}
}

// TestChooseSessionInitNewFlag verifies that --new bypasses auto-resume and
// creates a fresh session.
func TestChooseSessionInitNewFlag(t *testing.T) {
	a := newTestApp(t)
	defer a.Close()

	init := chooseSessionInit("", true, true)
	if err := init(a); err != nil {
		t.Fatalf("init error = %v", err)
	}
	// ResumeLastSession returns nil immediately when a store already exists.
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("store not created by --new: ResumeLastSession error = %v", err)
	}
}

// TestChooseSessionInitAutoResumeNoSession verifies that auto-resume falls back
// to a fresh session when there is no previous session and explains the
// fallback with a notice.
func TestChooseSessionInitAutoResumeNoSession(t *testing.T) {
	a := newTestApp(t)
	defer a.Close()

	c := newNoticeCollector()
	a.SetSendFn(c.send)

	init := chooseSessionInit("", false, true)
	if err := init(a); err != nil {
		t.Fatalf("init error = %v", err)
	}
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("store not created by auto-resume fallback: ResumeLastSession error = %v", err)
	}

	nm := c.wait(t, time.Second)
	if !strings.Contains(nm.Text, "no previous session") {
		t.Errorf("notice text = %q, want it to contain 'no previous session'", nm.Text)
	}
}

// TestChooseSessionInitAutoResumeResumeError verifies that auto-resume falls
// back to a fresh session on any resume error that is not ErrNoSession and
// sends a notice that includes the failure reason.
func TestChooseSessionInitAutoResumeResumeError(t *testing.T) {
	a, stateRoot, canonicalRoot := newTestAppWithRoot(t)
	defer a.Close()

	// Seed an index.json that points to a session file that does not exist so
	// ResumeLastSession fails with a non-ErrNoSession error.
	idx := &session.Index{
		Sessions: map[string]session.IndexEntry{
			canonicalRoot: {
				LastSessionID: "missing-session-id",
				UpdatedAt:     time.Now().UTC(),
			},
		},
	}
	if err := session.WriteIndex(stateRoot, idx); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	c := newNoticeCollector()
	a.SetSendFn(c.send)

	init := chooseSessionInit("", false, true)
	if err := init(a); err != nil {
		t.Fatalf("init error = %v", err)
	}
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("store not created by auto-resume fallback: ResumeLastSession error = %v", err)
	}

	nm := c.wait(t, time.Second)
	if !strings.Contains(nm.Text, "session resume failed") {
		t.Errorf("notice text = %q, want it to contain 'session resume failed'", nm.Text)
	}
	if !strings.Contains(nm.Text, "missing-session-id") {
		t.Errorf("notice text = %q, want it to contain the resume failure reason", nm.Text)
	}
}

// TestChooseSessionInitAutoResumeDisabled verifies that auto_resume=false starts
// a fresh session.
func TestChooseSessionInitAutoResumeDisabled(t *testing.T) {
	a := newTestApp(t)
	defer a.Close()

	init := chooseSessionInit("", false, false)
	if err := init(a); err != nil {
		t.Fatalf("init error = %v", err)
	}
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("store not created with auto_resume=false: ResumeLastSession error = %v", err)
	}
}

// TestChooseSessionInitExplicitResumeConcurrentInstance verifies that explicit
// `resume` is skipped when another instance is running, and a fresh session
// is started instead with a notice.
func TestChooseSessionInitExplicitResumeConcurrentInstance(t *testing.T) {
	a := newTestApp(t)
	defer a.Close()

	c := newNoticeCollector()
	a.SetSendFn(c.send)

	// Simulate another instance holding the lock by marking the app as concurrent.
	a.SetConcurrentInstance(true)

	init := chooseSessionInit("resume", false, true)
	if err := init(a); err != nil {
		t.Fatalf("init error = %v", err)
	}

	// Verify a fresh session was created.
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("store not created by concurrent-instance fallback: ResumeLastSession error = %v", err)
	}

	nm := c.wait(t, time.Second)
	if !strings.Contains(nm.Text, "resume skipped") {
		t.Errorf("notice text = %q, want it to contain 'resume skipped'", nm.Text)
	}
	if !strings.Contains(nm.Text, "another Kirsch instance is running") {
		t.Errorf("notice text = %q, want it to contain 'another Kirsch instance is running'", nm.Text)
	}
}

// TestChooseSessionInitAutoResumeConcurrentInstance verifies that auto-resume
// is skipped when another instance is running, and a fresh session is started
// instead with a notice.
func TestChooseSessionInitAutoResumeConcurrentInstance(t *testing.T) {
	a := newTestApp(t)
	defer a.Close()

	c := newNoticeCollector()
	a.SetSendFn(c.send)

	// Simulate another instance holding the lock by marking the app as concurrent.
	a.SetConcurrentInstance(true)

	init := chooseSessionInit("", false, true)
	if err := init(a); err != nil {
		t.Fatalf("init error = %v", err)
	}

	// Verify a fresh session was created.
	if err := a.ResumeLastSession(); err != nil {
		t.Fatalf("store not created by concurrent-instance fallback: ResumeLastSession error = %v", err)
	}

	nm := c.wait(t, time.Second)
	if !strings.Contains(nm.Text, "auto-resume skipped") {
		t.Errorf("notice text = %q, want it to contain 'auto-resume skipped'", nm.Text)
	}
	if !strings.Contains(nm.Text, "another Kirsch instance is running") {
		t.Errorf("notice text = %q, want it to contain 'another Kirsch instance is running'", nm.Text)
	}
}

// TestRunVersionOutsideRepository drives run() with --version from a directory
// that is not a Git repository. It must print exactly "kirsch <version>\n" and
// succeed, which proves the flag is handled before workspace detection.
func TestRunVersionOutsideRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run --version: %v", err)
	}
	if want := "kirsch " + buildVersion() + "\n"; stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	// Shape check independent of buildVersion().
	if got := stdout.String(); !strings.HasPrefix(got, "kirsch ") || !strings.HasSuffix(got, "\n") {
		t.Errorf("stdout = %q, want prefix \"kirsch \" and suffix newline", got)
	}
}

// TestRunHelp drives run() with each spelling of a help request. Help must
// succeed and list the commands (doctor included) and every flag on stdout.
func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"resume", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Chdir(t.TempDir())
			var stdout, stderr bytes.Buffer
			if err := run(args, &stdout, &stderr); err != nil {
				t.Fatalf("run %v: %v", args, err)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty (help goes to stdout only)", stderr.String())
			}
			out := stdout.String()
			// The Commands entry, not just the word: the usage block also says
			// "kirsch doctor", which would satisfy a bare substring check.
			if !strings.Contains(out, "\n  doctor    check the installation") {
				t.Errorf("stdout lacks the doctor entry under Commands:\n%s", out)
			}
			// The usage line, not just the word: doctor takes flags (--workspace).
			if !strings.Contains(out, "\n  kirsch doctor [flags]\n") {
				t.Errorf("stdout lacks the usage line %q:\n%s", "kirsch doctor [flags]", out)
			}
			for _, want := range []string{"resume", "doctor", "--workspace", "--new", "--debug", "--version"} {
				if !strings.Contains(out, want) {
					t.Errorf("stdout lacks %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestRunBadFlag drives run() with an undefined flag. It must fail, with the
// flag package's error and usage on stderr and nothing on stdout.
func TestRunBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--nope"}, &stdout, &stderr); err == nil {
		t.Fatal("run --nope returned nil, want an error")
	}
	if !strings.Contains(stderr.String(), "nope") || !strings.Contains(stderr.String(), "--workspace") {
		t.Errorf("stderr lacks the error or usage:\n%s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

// TestResolveVersion covers the stamped-wins rule and each build-info shape the
// fallback must accept or reject.
func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name, stamped, mainVersion, want string
	}{
		{"stamped wins over release tag", "9.9.9", "v0.1.0", "9.9.9"},
		{"stamped wins over empty", "9.9.9", "", "9.9.9"},
		{"release tag", devVersion, "v0.1.0", "0.1.0"},
		{"pre-release tag", devVersion, "v0.2.0-rc.1", "0.2.0-rc.1"},
		{"devel", devVersion, "(devel)", devVersion},
		{"empty", devVersion, "", devVersion},
		{"pseudo-version", devVersion, "v0.0.0-20261026103300-6956cc7abcde", devVersion},
		{"pseudo-version after tag", devVersion, "v0.1.1-0.20261026103300-6956cc7abcde", devVersion},
		{"dirty", devVersion, "v0.1.0+dirty", devVersion},
		{"tag without leading v", devVersion, "0.1.0", "0.1.0"},
		{"incompatible build metadata", devVersion, "v2.0.0+incompatible", devVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.stamped, tt.mainVersion); got != tt.want {
				t.Errorf("resolveVersion(%q, %q) = %q, want %q", tt.stamped, tt.mainVersion, got, tt.want)
			}
		})
	}
}

// TestLdflagsVersionBuild builds the real binary with -X main.version and runs
// --version from outside any repository. The value must come out bare.
func TestLdflagsVersionBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary; skipped under -short")
	}
	bin := filepath.Join(t.TempDir(), "kirsch")
	// The test binary runs on the machine that built it, so GOROOT is the
	// toolchain that is running this test, not whatever "go" PATH finds first.
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go") //nolint:staticcheck // SA1019: same-machine lookup is intended
	build := exec.Command(goBin, "build", "-ldflags", "-X main.version=9.9.9", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	cmd := exec.Command(bin, "--version")
	cmd.Dir = t.TempDir()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("kirsch --version: %v", err)
	}
	if want := "kirsch 9.9.9\n"; string(out) != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

// brokenConfigEnv points XDG_CONFIG_HOME at a global config that is not valid
// TOML and HOME at an empty directory, and returns the config home.
func brokenConfigEnv(t *testing.T) string {
	t.Helper()
	cfgHome := t.TempDir()
	dir := filepath.Join(cfgHome, "kirsch")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("this = [is not toml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HOME", t.TempDir())
	return cfgHome
}

// TestRunDoctorBeforeWorkspaceAndConfig drives run() with "doctor" from a
// directory that is not a repository, with a broken global config. Doctor must
// still print the version line, report both problems as lines (the repository
// and the config) and return errDoctorFailed, not the config or workspace error
// the normal start would return. The control runs the same broken config without
// doctor against a real repository and must fail on the config, so the doctor
// result is not just a config that happens to load.
func TestRunDoctorBeforeWorkspaceAndConfig(t *testing.T) {
	brokenConfigEnv(t)

	t.Run("control: broken config without doctor fails", func(t *testing.T) {
		repo, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
		if err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		err = run([]string{"--workspace", repo}, &stdout, &stderr)
		if err == nil || errors.Is(err, errDoctorFailed) {
			t.Fatalf("run with broken config = %v, want a config error", err)
		}
		if !strings.Contains(err.Error(), "config") {
			t.Errorf("error = %q, want it to name the config", err)
		}
	})

	t.Run("doctor in a non-repository with broken config", func(t *testing.T) {
		chdirOutsideRepo(t)
		var stdout, stderr bytes.Buffer
		err := run([]string{"doctor"}, &stdout, &stderr)
		if !errors.Is(err, errDoctorFailed) {
			t.Fatalf("run doctor = %v, want errDoctorFailed", err)
		}
		first, _, _ := strings.Cut(stdout.String(), "\n")
		if want := "kirsch " + buildVersion(); first != want {
			t.Errorf("first line = %q, want %q", first, want)
		}
		out := stdout.String()
		if !strings.Contains(out, "fail  repository: not in a Git repository\n") {
			t.Errorf("stdout lacks the repository failure:\n%s", out)
		}
		if !strings.Contains(out, "fail  config: parse ") {
			t.Errorf("stdout lacks the config failure:\n%s", out)
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty", stderr.String())
		}
	})
}

// doctorRun calls run() with "doctor" plus args and returns stdout, stderr and
// run's error. It is the entry point the doctor tests below drive: run itself,
// with the real doctorProbes, not the probes called directly.
func doctorRun(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run(append([]string{"doctor"}, args...), &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

// isolatedEnv points HOME, XDG_CONFIG_HOME and XDG_STATE_HOME at fresh empty
// temp dirs so a doctor run never reads the developer's own config, and unsets
// every API key variable the built-in endpoints read (keyVarNames), so an
// ambient key never reaches a test. Each is restored when the test ends. It
// returns the config home.
func isolatedEnv(t *testing.T) string {
	t.Helper()
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	for _, name := range keyVarNames() {
		setEnv(t, name, "")
	}
	return cfgHome
}

// keyVarNames returns every variable ResolveKey can read a key from for the
// built-in endpoints: each endpoint's api_key_env and its KIRSCH_-prefixed form.
func keyVarNames() []string {
	var names []string
	for _, ep := range config.Defaults().Provider.Endpoints {
		names = append(names, ep.APIKeyEnv, "KIRSCH_"+ep.APIKeyEnv)
	}
	return names
}

// TestIsolatedEnvClearsKeyVariables sets every key variable to a known value,
// calls isolatedEnv and asserts each reads empty afterwards. Control: before the
// call each variable reads the value just set, so an empty read afterwards is the
// helper's doing. Failure text names variables only, never a value.
func TestIsolatedEnvClearsKeyVariables(t *testing.T) {
	names := keyVarNames()
	if len(names) == 0 {
		t.Fatal("control: the built-in endpoints name no key variable")
	}
	t.Run("clears", func(t *testing.T) {
		const ambient = "ambient-key-value-zzz"
		for _, name := range names {
			t.Setenv(name, ambient)
			if os.Getenv(name) != ambient {
				t.Fatalf("control: %s did not read the value just set", name)
			}
		}
		isolatedEnv(t)
		for _, name := range names {
			if os.Getenv(name) != "" {
				t.Errorf("%s is still set after isolatedEnv (value withheld)", name)
			}
		}
	})
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

// gitRepoOrSkip creates a Git repository in a temp dir and returns its path, or
// skips the test when git is not installed.
func gitRepoOrSkip(t *testing.T) string {
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

// shimPathWithout builds a temp dir of symlinks to every executable on the
// current PATH except the named tool and points PATH at it for the test. PATH is
// never emptied (DIR-009): everything else stays findable, so a failure is about
// the one missing tool. Callers assert the controls themselves.
func shimPathWithout(t *testing.T, tool string) {
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
}

// TestRunDoctorRGMissing drives run() with rg removed from PATH by a shim: doctor
// must print a warn line naming the pure-Go fallback and, since a warning does
// not fail the run, return nil. Controls under the same shim PATH: git is found
// and rg is not.
func TestRunDoctorRGMissing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed; the control cannot hold")
	}
	isolatedEnv(t)
	repo := gitRepoOrSkip(t)
	shimPathWithout(t, "rg")
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatalf("control: LookPath(git) under the shim = %v, want found", err)
	}
	if _, err := exec.LookPath("rg"); err == nil {
		t.Fatal("control: LookPath(rg) under the shim succeeded, want not found")
	}
	out, _, err := doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v, want nil (a missing rg only warns)\n%s", err, out)
	}
	if !strings.Contains(out, "warn  rg: not found on PATH; search_code uses the pure-Go fallback\n") {
		t.Errorf("stdout lacks the rg warning:\n%s", out)
	}
	if !strings.Contains(out, "ok  git: ") {
		t.Errorf("stdout lacks the ok git line (control):\n%s", out)
	}
}

// TestRunDoctorGitMissing drives run() with git removed from PATH by a shim from
// a folder that is not a repository. Doctor must print a git failure line, must
// not print "not in a Git repository", and must return errDoctorFailed. The
// controls: the same folder with git still on PATH does print "not in a Git
// repository", and under the shim LookPath(git) fails while LookPath(ls)
// succeeds.
func TestRunDoctorGitMissing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed; the control cannot hold")
	}
	isolatedEnv(t)
	chdirOutsideRepo(t)

	ctl, _, _ := doctorRun(t)
	if !strings.Contains(ctl, "fail  repository: not in a Git repository\n") {
		t.Fatalf("control: with git on PATH stdout lacks the repository failure:\n%s", ctl)
	}

	shimPathWithout(t, "git")
	if _, err := exec.LookPath("git"); err == nil {
		t.Fatal("control: LookPath(git) under the shim succeeded, want not found")
	}
	if _, err := exec.LookPath("ls"); err != nil {
		t.Fatalf("control: LookPath(ls) under the shim = %v, want found", err)
	}
	out, _, err := doctorRun(t)
	if !errors.Is(err, errDoctorFailed) {
		t.Fatalf("run doctor = %v, want errDoctorFailed", err)
	}
	if !strings.Contains(out, "fail  git: not found on PATH") {
		t.Errorf("stdout lacks the git-missing line:\n%s", out)
	}
	if strings.Contains(out, "not in a Git repository") {
		t.Errorf("git missing was reported as not in a repository:\n%s", out)
	}
}

// TestRunDoctorOutsideRepository drives run() from a folder that is not a
// repository, with and without --workspace pointing at another such folder. It is
// a control pair: kirsch refuses to start in the first case and starts (without
// its Git features) in the second, and doctor's status and result follow. With no
// flag doctor prints the "not in a Git repository" failure and returns
// errDoctorFailed; with --workspace it prints a warning naming the folder and
// returns nil. Both still print the git, rg and config lines.
func TestRunDoctorOutsideRepository(t *testing.T) {
	isolatedEnv(t)
	notRepo := t.TempDir()
	chdirOutsideRepo(t, notRepo)
	for _, tc := range []struct {
		name    string
		args    []string
		wantErr bool
		want    []string
	}{
		{"cwd", nil, true, []string{
			"fail  repository: not in a Git repository\n",
			"ok  config: settings valid; project file not read (no workspace)\n",
		}},
		{"workspace", []string{"--workspace", notRepo}, false, []string{
			"warn  repository: " + notRepo + " is not a Git repository; kirsch starts, but the Git features are off",
			"ok  config: settings valid\n",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := doctorRun(t, tc.args...)
			if tc.wantErr && !errors.Is(err, errDoctorFailed) {
				t.Fatalf("run doctor = %v, want errDoctorFailed", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("run doctor = %v, want nil (a non-Git --workspace only warns)\n%s", err, out)
			}
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("stdout lacks %q:\n%s", want, out)
				}
			}
			for _, prefix := range []string{"ok  git: ", "rg: "} {
				if !strings.Contains(out, prefix) {
					t.Errorf("stdout lacks a line with %q:\n%s", prefix, out)
				}
			}
		})
	}
}

// TestDoctorBinaryExitStatus builds the real binary and runs `kirsch doctor` as a
// subprocess, which is the only path through main() and its os.Exit. Outside a
// repository with no --workspace it must exit 1 and print no "kirsch:" line on
// stderr (errDoctorFailed is exited on silently; the report on stdout already
// says what failed). The control is the same binary in a fresh Git repository,
// which must exit 0. HOME and the XDG directories are isolated so the
// developer's own config is never read.
func TestDoctorBinaryExitStatus(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on PATH; cannot build the binary")
	}
	repo := gitRepoOrSkip(t)

	// Built before the environment is isolated: with HOME pointed at a temp dir
	// the go command would default its module cache there and populate it.
	bin := filepath.Join(t.TempDir(), "kirsch")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	isolatedEnv(t)

	runDoctor := func(dir string) (stdout, stderr string, code int) {
		t.Helper()
		var so, se bytes.Buffer
		cmd := exec.Command(bin, "doctor")
		cmd.Dir = dir
		// The ceiling stops git looking above the folder, so a temp dir that
		// happens to sit inside some repository still counts as outside one.
		cmd.Env = append(os.Environ(), "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir))
		cmd.Stdout, cmd.Stderr = &so, &se
		err := cmd.Run()
		var ee *exec.ExitError
		switch {
		case err == nil:
		case errors.As(err, &ee):
			code = ee.ExitCode()
		default:
			t.Fatalf("run %s doctor in %s: %v", bin, dir, err)
		}
		return so.String(), se.String(), code
	}

	out, errOut, code := runDoctor(repo)
	if code != 0 {
		t.Fatalf("control: doctor in a Git repository exited %d, want 0\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "ok  repository: ") {
		t.Errorf("control: stdout lacks the ok repository line:\n%s", out)
	}

	out, errOut, code = runDoctor(t.TempDir())
	if code != 1 {
		t.Errorf("doctor outside a repository exited %d, want 1\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "fail  repository: not in a Git repository\n") {
		t.Errorf("stdout lacks the repository failure:\n%s", out)
	}
	if strings.Contains(errOut, "kirsch:") {
		t.Errorf("stderr has a kirsch: line, want none for a doctor failure:\n%s", errOut)
	}

	// Positive control for the absence check above: the same capture, on a
	// failure that main does print. An unknown command makes run return an error
	// that main writes to stderr as `kirsch: unknown command "bogus"`, so if
	// stderr were not wired to the buffer this check would fail.
	cmd := exec.Command(bin, "bogus")
	var se bytes.Buffer
	cmd.Stderr = &se
	if err := cmd.Run(); err == nil {
		t.Fatal("control: kirsch bogus exited 0, want a failure")
	}
	if !strings.Contains(se.String(), `kirsch: unknown command "bogus"`) {
		t.Errorf("control: stderr = %q, want the kirsch: unknown command line", se.String())
	}
}

// TestRunDoctorBrokenConfigDoesNotExitEarly drives run() with a broken global
// config.toml inside a real repository: the config line names the file and the
// parse error, and the git, rg and repository lines and the summary still
// print. The control is the same run without the broken file.
func TestRunDoctorBrokenConfigDoesNotExitEarly(t *testing.T) {
	repo := gitRepoOrSkip(t)
	cfgHome := isolatedEnv(t)

	ctl, _, _ := doctorRun(t, "--workspace", repo)
	if !strings.Contains(ctl, "ok  config: settings valid\n") {
		t.Fatalf("control: stdout lacks the ok config line:\n%s", ctl)
	}

	path := filepath.Join(cfgHome, "kirsch", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("this = [is not toml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err := doctorRun(t, "--workspace", repo)
	if !errors.Is(err, errDoctorFailed) {
		t.Fatalf("run doctor = %v, want errDoctorFailed", err)
	}
	if !strings.Contains(out, "fail  config: parse "+path+": ") {
		t.Errorf("stdout lacks the config line naming %s:\n%s", path, out)
	}
	for _, want := range []string{"ok  git: ", "rg: ", "ok  repository: " + repo + "\n", "doctor: "} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout lacks %q after the config failure:\n%s", want, out)
		}
	}
}

// TestRunDoctorMultiLineConfigErrorIsOneLine drives run() with a global config
// whose error message spans several lines (a credential-looking key) and checks
// that no report line breaks the "<status>  <check>: <detail>" grammar.
func TestRunDoctorMultiLineConfigErrorIsOneLine(t *testing.T) {
	cfgHome := isolatedEnv(t)
	t.Chdir(t.TempDir())
	path := filepath.Join(cfgHome, "kirsch", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("api_key = \"abc\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Control: the real config error does span lines, so the test input is not
	// accidentally single-line.
	if _, _, err := config.Load(config.Options{GlobalPath: path}); err == nil || !strings.Contains(err.Error(), "\n") {
		t.Fatalf("control: config.Load error = %v, want a multi-line error", err)
	}
	out, _, _ := doctorRun(t)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	found := false
	for _, l := range lines[1 : len(lines)-1] {
		if !strings.HasPrefix(l, "ok  ") && !strings.HasPrefix(l, "warn  ") && !strings.HasPrefix(l, "fail  ") {
			t.Errorf("line %q does not start with a status word", l)
		}
		if strings.HasPrefix(l, "fail  config: ") && strings.Contains(l, "looks like a credential") {
			found = true
		}
	}
	if !found {
		t.Errorf("stdout lacks one config line with the whole error:\n%s", out)
	}
}

// TestRunDoctorUnknownConfigKeyWarns drives run() with a config holding an
// unknown key: doctor prints a warn line naming the file and key and, a warning
// not being a failure, run returns nil in a real repository.
func TestRunDoctorUnknownConfigKeyWarns(t *testing.T) {
	repo := gitRepoOrSkip(t)
	cfgHome := isolatedEnv(t)
	path := filepath.Join(cfgHome, "kirsch", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("bogus_setting = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err := doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor = %v, want nil (an unknown key only warns)\n%s", err, out)
	}
	want := "warn  config: " + path + ": unknown key \"bogus_setting\" (ignored)\n"
	if !strings.Contains(out, want) {
		t.Errorf("stdout lacks %q:\n%s", want, out)
	}
}

// TestRunDoctorHonoursWorkspaceFlag drives run() from a folder that is not a
// repository with --workspace pointing at a real one: the repository line is the
// flag's root. The control is the same folder without the flag, which fails.
func TestRunDoctorHonoursWorkspaceFlag(t *testing.T) {
	repo := gitRepoOrSkip(t)
	isolatedEnv(t)
	chdirOutsideRepo(t)

	ctl, _, err := doctorRun(t)
	if !errors.Is(err, errDoctorFailed) || !strings.Contains(ctl, "fail  repository: not in a Git repository\n") {
		t.Fatalf("control: without the flag run = %v, stdout:\n%s", err, ctl)
	}
	out, _, err := doctorRun(t, "--workspace", repo)
	if err != nil {
		t.Fatalf("run doctor --workspace = %v, want nil\n%s", err, out)
	}
	if want := "ok  repository: " + repo + "\n"; !strings.Contains(out, want) {
		t.Errorf("stdout lacks %q:\n%s", want, out)
	}
}

// TestRunDoctorExitSentinel injects results through the doctorProbes seam and
// calls run(). A fail must return errDoctorFailed after printing the report; a
// warn-only set must return nil; a result with an unknown status must fail the
// run too (DIR-002), with the real Report and Failed doing the counting.
func TestRunDoctorExitSentinel(t *testing.T) {
	orig := doctorProbes
	t.Cleanup(func() { doctorProbes = orig })
	t.Chdir(t.TempDir())

	doctorProbes = func(doctor.Options) doctor.Outcome {
		return doctor.Outcome{Results: []doctor.Result{
			{Status: doctor.StatusOK, Name: "a", Detail: "fine"},
			{Status: doctor.StatusFail, Name: "b", Detail: "broken"},
		}}
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"doctor"}, &stdout, &stderr)
	if !errors.Is(err, errDoctorFailed) {
		t.Fatalf("run with a fail = %v, want errDoctorFailed", err)
	}
	if want := "fail  b: broken\n"; !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout lacks %q:\n%s", want, stdout.String())
	}
	if !strings.HasSuffix(stdout.String(), "doctor: 1 ok, 0 warn, 1 fail\n") {
		t.Errorf("stdout lacks the summary:\n%s", stdout.String())
	}

	doctorProbes = func(doctor.Options) doctor.Outcome {
		return doctor.Outcome{Results: []doctor.Result{{Status: doctor.StatusWarn, Name: "rg", Detail: "missing"}}}
	}
	stdout.Reset()
	if err := run([]string{"doctor"}, &stdout, &stderr); err != nil {
		t.Errorf("run with only a warn = %v, want nil", err)
	}

	doctorProbes = func(doctor.Options) doctor.Outcome {
		return doctor.Outcome{Results: []doctor.Result{{Status: doctor.Status(9), Name: "x", Detail: "odd"}}}
	}
	stdout.Reset()
	if err := run([]string{"doctor"}, &stdout, &stderr); !errors.Is(err, errDoctorFailed) {
		t.Errorf("run with an unknown status = %v, want errDoctorFailed", err)
	}
}

// TestRunDoctorPassesOptions injects a probe through the doctorProbes seam and
// checks what run hands it: the --workspace value, the global config path built
// from XDG_CONFIG_HOME, the home directory, and a LookPath that finds a tool
// known to be on PATH. This is the wiring the environment-read-in-main
// convention depends on.
func TestRunDoctorPassesOptions(t *testing.T) {
	orig := doctorProbes
	t.Cleanup(func() { doctorProbes = orig })
	cfgHome := isolatedEnv(t)
	t.Chdir(t.TempDir())

	var got doctor.Options
	doctorProbes = func(o doctor.Options) doctor.Outcome {
		got = o
		return doctor.Outcome{}
	}
	if _, _, err := doctorRun(t, "--workspace", "/some/where"); err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceFlag != "/some/where" {
		t.Errorf("WorkspaceFlag = %q, want /some/where", got.WorkspaceFlag)
	}
	if want := filepath.Join(cfgHome, "kirsch", "config.toml"); got.GlobalConfigPath != want {
		t.Errorf("GlobalConfigPath = %q, want %q", got.GlobalConfigPath, want)
	}
	if got.HomeDir != os.Getenv("HOME") || got.HomeDir == "" {
		t.Errorf("HomeDir = %q, want HOME %q", got.HomeDir, os.Getenv("HOME"))
	}
	if got.LookPath == nil {
		t.Fatal("LookPath is nil, want exec.LookPath")
	}
	want, wantErr := exec.LookPath("ls")
	if wantErr != nil {
		t.Skipf("ls is not on PATH, so LookPath cannot be exercised: %v", wantErr)
	}
	if path, err := got.LookPath("ls"); err != nil || path != want {
		t.Errorf("LookPath(ls) = %q, %v, want %q", path, err, want)
	}
	if _, err := got.LookPath("kirsch-no-such-tool-xyz"); err == nil {
		t.Error("LookPath(kirsch-no-such-tool-xyz) succeeded, want an error")
	}
}

// TestRunDoctorVersionAndHelpFlags checks that --version and --help given with
// doctor are answered by the flag handling: the version alone, or the help text,
// with no doctor report.
func TestRunDoctorVersionAndHelpFlags(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := run([]string{"doctor", "--version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if want := "kirsch " + buildVersion() + "\n"; stdout.String() != want {
		t.Errorf("--version stdout = %q, want %q", stdout.String(), want)
	}
	stdout.Reset()
	if err := run([]string{"doctor", "--help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != usageText {
		t.Errorf("--help stdout = %q, want the usage text", stdout.String())
	}
}

// fileState is what the changes-nothing test records about one path: its kind,
// and for a regular file its size, modification time and content hash.
type fileState struct {
	dir   bool
	size  int64
	mtime int64
	sum   [sha256.Size]byte
}

// snapshotTree walks root and records a fileState for every path under it,
// including root and every directory (a directory's mtime moves when an entry is
// created or removed). Any walk, stat or read error fails the test: an error
// swallowed here would make a blind snapshot look like an unchanged tree.
func snapshotTree(t *testing.T, root string) map[string]fileState {
	t.Helper()
	got := map[string]fileState{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		st := fileState{dir: d.IsDir(), size: info.Size(), mtime: info.ModTime().UnixNano()}
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(p) // #nosec G304 -- test walks its own temp dirs
			if err != nil {
				return err
			}
			st.sum = sha256.Sum256(data)
		}
		got[p] = st
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return got
}

// diffTrees lists, one line each, what differs between two snapshots.
func diffTrees(before, after map[string]fileState) []string {
	var diffs []string
	for p, b := range before {
		a, ok := after[p]
		switch {
		case !ok:
			diffs = append(diffs, "removed "+p)
		case a != b:
			diffs = append(diffs, "changed "+p)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			diffs = append(diffs, "added "+p)
		}
	}
	sort.Strings(diffs)
	return diffs
}

// TestRunDoctorChangesNothing runs `kirsch doctor --workspace <ws>` with HOME,
// XDG_CONFIG_HOME, XDG_STATE_HOME, the working directory and the workspace each
// in their own temp directory, and asserts that no path under any of those five
// roots was added, removed or changed. A path is compared by kind, size, mtime
// and SHA-256, and a walk error fails the test rather than being skipped.
//
// The roots hold real input so doctor reads files rather than finding nothing:
// a valid global config, a valid project config under the workspace, and a Git
// repository in the workspace (when git is installed).
//
// The test is built to be able to fail, per DIR-026 and DIR-001:
//   - positive control per root: a file placed there must appear in that root's
//     snapshot, so a walker blind to a root is caught;
//   - comparison control: writing one more file into each root must make
//     diffTrees report it, so a comparison that cannot see a change is caught;
//   - the write mutant (a probe that writes under HOME) is run via
//     `go test -overlay` and recorded in the step report, not in this file.
//
// Doctor has no network code; that is by construction, not asserted here. The
// entry point is run(), with the real doctorProbes.
func TestRunDoctorChangesNothing(t *testing.T) {
	home, cfg, state, cwd, ws := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	roots := map[string]string{"home": home, "xdg-config": cfg, "xdg-state": state, "cwd": cwd, "workspace": ws}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_STATE_HOME", state)
	t.Chdir(cwd)

	// Real input for doctor to read.
	if _, err := exec.LookPath("git"); err == nil {
		if out, err := exec.Command("git", "-C", ws, "init", "-q").CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
	}
	for _, f := range []string{
		filepath.Join(cfg, "kirsch", "config.toml"),
		filepath.Join(ws, ".kirsch", "config.toml"),
	} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("[session]\nauto_resume = false\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Positive control per root: a file placed there must be seen.
	for name, root := range roots {
		p := filepath.Join(root, "control.txt")
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok := snapshotTree(t, root)[p]; !ok {
			t.Fatalf("control: the snapshot of root %s does not contain %s", name, p)
		}
	}

	// Comparison control: one added file per root must show up as a difference.
	for name, root := range roots {
		before := snapshotTree(t, root)
		extra := filepath.Join(root, "extra-control.txt")
		if err := os.WriteFile(extra, []byte("y"), 0o600); err != nil {
			t.Fatal(err)
		}
		if d := diffTrees(before, snapshotTree(t, root)); len(d) == 0 {
			t.Fatalf("control: adding %s to root %s produced no difference", extra, name)
		}
		if err := os.Remove(extra); err != nil {
			t.Fatal(err)
		}
	}

	before := map[string]map[string]fileState{}
	for name, root := range roots {
		before[name] = snapshotTree(t, root)
	}

	var stdout, stderr bytes.Buffer
	err := run([]string{"doctor", "--workspace", ws}, &stdout, &stderr)
	if err != nil && !errors.Is(err, errDoctorFailed) {
		// A failing check is acceptable (git may be missing here); any other
		// error means doctor did not run to the report.
		t.Fatalf("run doctor: %v", err)
	}
	if !strings.Contains(stdout.String(), "ok  config: settings valid\n") {
		t.Fatalf("doctor did not read the config it was given:\n%s", stdout.String())
	}

	for name, root := range roots {
		if d := diffTrees(before[name], snapshotTree(t, root)); len(d) != 0 {
			t.Errorf("doctor changed root %s (%s): %v", name, root, d)
		}
	}
}
