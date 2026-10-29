package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
// still print the version line and succeed. The control runs the same broken
// config without doctor against a real repository and must fail on the config,
// so the doctor pass is not just a config that happens to load.
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
		t.Chdir(t.TempDir())
		var stdout, stderr bytes.Buffer
		if err := run([]string{"doctor"}, &stdout, &stderr); err != nil {
			t.Fatalf("run doctor: %v", err)
		}
		first, _, _ := strings.Cut(stdout.String(), "\n")
		if want := "kirsch " + buildVersion(); first != want {
			t.Errorf("first line = %q, want %q", first, want)
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty", stderr.String())
		}
	})
}

// TestRunDoctorSkeletonReport pins the whole report when no probe exists yet:
// the version line and a zero summary.
func TestRunDoctorSkeletonReport(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := run([]string{"doctor"}, &stdout, &stderr); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	want := "kirsch " + buildVersion() + "\ndoctor: 0 ok, 0 warn, 0 fail\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

// TestRunDoctorExitSentinel injects results through the doctorProbes seam and
// calls run(). A fail must return errDoctorFailed after printing the report; a
// warn-only set must return nil.
func TestRunDoctorExitSentinel(t *testing.T) {
	orig := doctorProbes
	t.Cleanup(func() { doctorProbes = orig })
	t.Chdir(t.TempDir())

	doctorProbes = func() []doctor.Result {
		return []doctor.Result{
			{Status: doctor.StatusOK, Name: "a", Detail: "fine"},
			{Status: doctor.StatusFail, Name: "b", Detail: "broken"},
		}
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

	doctorProbes = func() []doctor.Result {
		return []doctor.Result{{Status: doctor.StatusWarn, Name: "rg", Detail: "missing"}}
	}
	stdout.Reset()
	if err := run([]string{"doctor"}, &stdout, &stderr); err != nil {
		t.Errorf("run with only a warn = %v, want nil", err)
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

// TestRunDoctorChangesNothing runs doctor with HOME, XDG_CONFIG_HOME and
// XDG_STATE_HOME pointing at empty temp directories and the working directory
// in another, and asserts nothing was created in any of them. The control is a
// file written into the working directory beforehand: the snapshot must see it,
// so an empty snapshot cannot be mistaken for a blind one. Doctor has no network
// code in this step; that is by construction, not asserted here.
func TestRunDoctorChangesNothing(t *testing.T) {
	home, cfg, state, cwd := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_STATE_HOME", state)
	t.Chdir(cwd)
	if err := os.WriteFile(filepath.Join(cwd, "control.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := func() []string {
		var got []string
		for _, root := range []string{home, cfg, state, cwd} {
			_ = filepath.WalkDir(root, func(p string, _ os.DirEntry, err error) error {
				if err == nil && p != root {
					got = append(got, p)
				}
				return nil
			})
		}
		return got
	}
	before := snapshot()
	if len(before) != 1 {
		t.Fatalf("control snapshot = %v, want exactly the control file", before)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"doctor"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(); strings.Join(after, "\n") != strings.Join(before, "\n") {
		t.Errorf("doctor changed the filesystem: before %v, after %v", before, after)
	}
}
