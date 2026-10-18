package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// wiredDiffModel builds a real App over repo and a real TUI model wired by
// wireCallbacks (the production wiring), sized so keys reach the composer.
func wiredDiffModel(t *testing.T, repo string) tea.Model {
	t.Helper()
	ws, err := workspace.Detect(repo)
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}
	log := telemetry.Disabled()
	a := app.New(ws, config.Config{}, log)
	t.Cleanup(func() { a.Close() })

	m := tui.New(tui.Options{
		Version: "0.1.0-test",
		Caps:    tui.Caps{Colour: false, Unicode: false},
	})
	wireCallbacks(&m, a, log)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return tm
}

// runDiffThroughWiring types /diff, presses Enter, runs the returned command
// and feeds its message back, returning the rendered frame.
func runDiffThroughWiring(t *testing.T, tm tea.Model) string {
	t.Helper()
	for _, r := range "/diff" {
		tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	var cmd tea.Cmd
	tm, cmd = tm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("/diff returned no tea.Cmd through the wired model")
	}
	tm, _ = tm.Update(cmd())
	return tm.View()
}

func gitHere(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{
		"-c", "user.name=t", "-c", "user.email=t@example.com",
		"-c", "commit.gpgsign=false",
	}, args...)
	c := exec.Command("git", full...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestDiffCommandThroughRealWiring runs /diff with the real app.GetDiff behind
// the real wireCallbacks, in a real repo, in both states. A clean tree must
// show the "working tree clean" notice and no modal; a dirty tree must open the
// modal. The modal's " close" footer is the discriminator: the notice text alone
// would also appear if a clean tree opened a one-line modal.
func TestDiffCommandThroughRealWiring(t *testing.T) {
	repo := t.TempDir()
	gitHere(t, repo, "init")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitHere(t, repo, "add", ".")
	gitHere(t, repo, "commit", "-m", "initial")

	t.Run("dirty tree opens the modal", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		view := runDiffThroughWiring(t, wiredDiffModel(t, repo))
		if !strings.Contains(view, "+two") {
			t.Errorf("dirty tree frame has no diff line \"+two\":\n%s", view)
		}
		if !strings.Contains(view, " close") {
			t.Errorf("dirty tree frame has no modal footer:\n%s", view)
		}
	})

	t.Run("clean tree shows the notice and no modal", func(t *testing.T) {
		gitHere(t, repo, "commit", "-am", "second")
		view := runDiffThroughWiring(t, wiredDiffModel(t, repo))
		if !strings.Contains(view, "working tree clean") {
			t.Errorf("clean tree frame has no notice:\n%s", view)
		}
		if strings.Contains(view, " close") {
			t.Errorf("clean tree opened a modal (footer present):\n%s", view)
		}
	})
}
