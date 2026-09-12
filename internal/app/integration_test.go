package app

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// driveTUI wires a real App to a real TUI model and pumps messages between
// them, exactly as cmd/kirsch does — but without a tea.Program.
//
// Bubble Tea renders nothing to a non-TTY, so running a real program here
// produces an empty buffer and proves only that the harness is wrong. Feeding
// App's messages straight into Update tests the thing that actually matters:
// that an intent from the TUI reaches a tool, and that the tool's result comes
// back as something the TUI can render.
func driveTUI(t *testing.T, fixture string, typed string, settle time.Duration, expand bool) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	info := a.WorkspaceInfo()
	m := tui.New(tui.Options{
		Version: "0.1.0",
		Caps:    tui.Caps{Colour: false, Unicode: true},
		Session: tui.SessionInfo{Project: info.Project, Branch: info.Branch, Dirty: info.Dirty},
	})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn

	var mu sync.Mutex
	// Unbuffered on purpose. A buffered channel absorbs a message sent from
	// inside Update and so hides the deadlock the real Bubble Tea program hits,
	// where program.Send waits for an event loop that is itself inside Update.
	// With no buffer, a regression stalls the harness and the test times out
	// rather than passing.
	inbox := make(chan tea.Msg)
	a.sendFn = func(msg any) {
		if tm, ok := msg.(tea.Msg); ok {
			inbox <- tm
		}
	}

	apply := func(msg tea.Msg) {
		mu.Lock()
		defer mu.Unlock()
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}

	apply(tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, r := range typed {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if typed != "" {
		apply(tea.KeyMsg{Type: tea.KeyEnter})
	}

	deadline := time.After(settle)
	for {
		select {
		case msg := <-inbox:
			apply(msg)
		case <-deadline:
			mu.Lock()
			defer mu.Unlock()
			if expand {
				// A tool card is collapsed until Enter (ui-spec §3.3), so the
				// content is only on screen after focusing the transcript and
				// expanding the card the result selected.
				next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
				m = next.(tui.Model)
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = next.(tui.Model)
			}
			return m.View()
		}
	}
}

// TestDebugCommandReadsRealFile is Task 9's acceptance criterion: from a real
// repository, a debug command reads a file and renders it as a tool card.
func TestDebugCommandReadsRealFile(t *testing.T) {
	// Collapsed first: the card and its summary must be on screen.
	out := driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, false)
	for _, want := range []string{"read_file", "go.mod"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in the collapsed card:\n%s", want, out)
		}
	}
	// Expanded: the file's contents, line-numbered.
	out = driveTUI(t, "repo-go-module", "/read go.mod", 1500*time.Millisecond, true)
	for _, want := range []string{"module example.com/calc"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

func TestDebugCommandSearchesRealRepo(t *testing.T) {
	out := driveTUI(t, "repo-go-module", "/search Divide", 1500*time.Millisecond, true)
	for _, want := range []string{"search_code", "Divide", "calc/divide.go"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in rendered output:\n%s", want, out)
		}
	}
}

// TestDeniedPathRendersAsAToolCard is the other half of Task 9's check: a
// denylisted path must render an error, not panic and not silently do nothing.
func TestDeniedPathRendersAsAToolCard(t *testing.T) {
	out := driveTUI(t, "repo-small", "/read .env", 1500*time.Millisecond, false)
	if !strings.Contains(out, "read_file") {
		t.Errorf("no tool card rendered:\n%s", out)
	}
	if !strings.Contains(out, "denied") && !strings.Contains(out, "never readable") {
		t.Errorf("refusal not shown to the user:\n%s", out)
	}
	if strings.Contains(out, "hunter2") {
		t.Fatalf(".env contents reached the screen:\n%s", out)
	}
}

func TestHeaderShowsRealWorkspace(t *testing.T) {
	out := driveTUI(t, "repo-node", "", 400*time.Millisecond, false)
	if !strings.Contains(out, "repo-node") {
		t.Errorf("header does not show the real project name:\n%s", out)
	}
}

// TestCancelledToolRendersCancelledGlyph is the last of Task 8's checks:
// cancelling returns control and the card shows ⊘ rather than hanging on the
// spinner or vanishing.
func TestCancelledToolRendersCancelledGlyph(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-small"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	a := New(ws, config.Defaults(), telemetry.Disabled())
	defer a.Close()

	slow := &slowTool{started: make(chan struct{})}
	a.reg.Register(slow)

	m := tui.New(tui.Options{Version: "0.1.0", Caps: tui.Caps{Unicode: true}})
	m.RunTool = a.RunTool
	m.Cancel = a.CancelTurn

	inbox := make(chan tea.Msg)
	a.sendFn = func(msg any) { inbox <- msg.(tea.Msg) }

	apply := func(msg tea.Msg) {
		next, _ := m.Update(msg)
		m = next.(tui.Model)
	}
	apply(tea.WindowSizeMsg{Width: 100, Height: 30})

	m.RunTool("slow", map[string]any{})
	apply(<-inbox) // ToolStartedMsg

	select {
	case <-slow.started:
	case <-time.After(2 * time.Second):
		t.Fatal("tool never started")
	}
	if !strings.Contains(m.View(), "◐") {
		t.Errorf("running card does not show the running badge:\n%s", m.View())
	}

	begin := time.Now()
	apply(tea.KeyMsg{Type: tea.KeyEsc}) // Esc cancels the turn
	apply(<-inbox)                      // ToolCompletedMsg
	elapsed := time.Since(begin)

	if elapsed > time.Second {
		t.Errorf("cancel took %v; ui-spec §11 targets under 1s", elapsed)
	}
	view := m.View()
	if !strings.Contains(view, "⊘") {
		t.Errorf("cancelled card does not show ⊘:\n%s", view)
	}
	if strings.Contains(view, "◐") {
		t.Errorf("card still shows the running badge after cancellation:\n%s", view)
	}
}
