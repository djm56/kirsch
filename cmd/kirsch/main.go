// Command kirsch is a terminal-native coding agent.
//
// Milestone 0 renders the interface against fake data: there is no model,
// no filesystem access and no subprocess execution anywhere in the binary.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/tui"
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
	m := tui.New(tui.Options{
		Version: version,
		Caps:    tui.DetectCaps(os.Stdout),
	})

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
