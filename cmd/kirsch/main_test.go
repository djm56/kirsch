package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
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
// succeed, list the commands and every flag on stdout, and not mention doctor.
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
			for _, want := range []string{"resume", "--workspace", "--new", "--debug", "--version"} {
				if !strings.Contains(out, want) {
					t.Errorf("stdout lacks %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "doctor") {
				t.Errorf("stdout mentions doctor:\n%s", out)
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
