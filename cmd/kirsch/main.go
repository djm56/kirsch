// Command kirsch is a terminal-native coding agent.
//
// Milestone 1: the TUI drives real read-only tools against a real repository.
// There is no model, no patching and no arbitrary command execution anywhere in
// the binary — the only subprocesses are read-only git and ripgrep.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// version is overridden at release time via -ldflags (Milestone 5).
var version = "0.1.0-dev"

func main() {
	if err := run(); err != nil {
		// The TUI owns the alternate screen until Run returns, so by the time
		// we get here stderr is safe to write to again.
		fmt.Fprintln(os.Stderr, "kirsch:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		flagWorkspace = flag.String("workspace", "", "workspace root (defaults to the enclosing Git repository)")
		flagDebug     = flag.Bool("debug", false, "write a structured debug log")
	)
	flag.Parse()

	ws, err := workspace.Detect(*flagWorkspace)
	if err != nil {
		return err
	}

	// Every environment read happens here, in main, and is passed inward. The
	// config and telemetry packages never consult the environment themselves,
	// which is what keeps their tests hermetic.
	home, _ := os.UserHomeDir()
	cfg, warnings, err := config.Load(config.Options{
		GlobalPath:  config.GlobalPath(os.Getenv("XDG_CONFIG_HOME"), home),
		ProjectPath: config.ProjectPath(ws.Root),
		HomeDir:     home,
	})
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	debug := *flagDebug || os.Getenv("KIRSCH_DEBUG") == "1" || cfg.Telemetry.DebugLog
	log, err := telemetry.New(telemetry.Options{
		Path:    telemetry.DefaultPath(os.Getenv("XDG_STATE_HOME"), home),
		Enabled: debug,
	})
	if err != nil {
		// A debug log that cannot be opened is not worth refusing to run over.
		log = telemetry.Disabled()
	}

	a := app.New(ws, cfg, log)
	defer a.Close()

	info := a.WorkspaceInfo()
	m := tui.New(tui.Options{
		Version: version,
		Caps:    tui.DetectCaps(os.Stdout),
		Session: tui.SessionInfo{
			Project: info.Project,
			Branch:  info.Branch,
			Dirty:   info.Dirty,
		},
	})
	// Wrapped rather than assigned directly so the debug log records what the
	// interface asked for, separately from what the tool layer then did. When
	// the two disagree, that gap is the bug.
	m.RunTool = func(name string, input map[string]any) {
		log.Debug("tui requested tool", "tool", name, "input", input)
		a.RunTool(name, input)
	}
	m.Cancel = func() {
		log.Debug("tui requested cancel")
		a.CancelTurn()
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	a.Attach(p)

	for _, w := range warnings {
		log.Warn("config", "detail", w.String())
	}
	log.Info("starting",
		"version", version,
		"workspace", ws.Root,
		"git", ws.IsGit,
		"tools", a.Registry().Names())

	_, err = p.Run()
	return err
}
