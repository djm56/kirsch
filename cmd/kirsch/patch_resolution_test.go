package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// TestResolvePatchFileWiring drives the installed m.ResolvePatchFile closure —
// the one wireCallbacks assigns in main.go, not a test double — against a
// real workspace rooted at a temporary directory containing
// testdata/patches/.
//
// It exists because nothing else exercises that wiring. internal/tui cannot
// import internal/workspace, so a test living there is necessarily a
// substitute for real resolution; step 10c's substitute went further and
// mocked the very call the gate had just forced into the production code,
// under a comment claiming to simulate what workspace.Resolve would do. This
// test calls the resolver wireCallbacks actually installs, so it fails if the
// wiring regresses, not just if the resolver package's own tests do.
func TestResolvePatchFileWiring(t *testing.T) {
	root := t.TempDir()
	patchDir := filepath.Join(root, "testdata", "patches")
	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", patchDir, err)
	}

	const wantDiff = "--- a/x\n+++ b/x\n@@ -0,0 +1 @@\n+hello\n"
	if err := os.WriteFile(filepath.Join(patchDir, "real.diff"), []byte(wantDiff), 0o644); err != nil {
		t.Fatalf("WriteFile(real.diff): %v", err)
	}

	// A file that genuinely exists one level above the patch directory, so
	// the parent-segment case below proves the confinement check refuses it
	// rather than merely observing that nothing was there to find.
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("top secret"), 0o644); err != nil {
		t.Fatalf("WriteFile(secret.txt): %v", err)
	}

	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatalf("workspace.Detect(%s): %v", root, err)
	}

	log, err := telemetry.New(telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}
	cfg, _, err := config.Load(config.Options{})
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	a := app.New(ws, cfg, log)
	defer a.Close()

	m := tui.New(tui.Options{Version: "0.1.0-test"})
	wireCallbacks(&m, a, log, ws)

	t.Run("legitimate file resolves", func(t *testing.T) {
		got, err := m.ResolvePatchFile("real.diff")
		if err != nil {
			t.Fatalf(`ResolvePatchFile("real.diff") = err %v, want nil`, err)
		}
		if got != wantDiff {
			t.Errorf(`ResolvePatchFile("real.diff") = %q, want %q`, got, wantDiff)
		}
	})

	t.Run("symlink inside the patch directory pointing outside the workspace is refused", func(t *testing.T) {
		outsideDir := t.TempDir()
		outside := filepath.Join(outsideDir, "secret.txt")
		if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
			t.Fatalf("WriteFile(outside): %v", err)
		}
		link := filepath.Join(patchDir, "evil.diff")
		if err := os.Symlink(outside, link); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		defer os.Remove(link)

		content, err := m.ResolvePatchFile("evil.diff")
		if err == nil {
			t.Fatalf(`ResolvePatchFile("evil.diff") = %q, nil error, want refusal`, content)
		}
	})

	t.Run("a parent segment is refused", func(t *testing.T) {
		content, err := m.ResolvePatchFile("../../secret.txt")
		if err == nil {
			t.Fatalf(`ResolvePatchFile("../../secret.txt") = %q, nil error, want refusal`, content)
		}
	})
}
