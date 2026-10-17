package main

import (
	"errors"
	"path/filepath"
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
