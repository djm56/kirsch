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
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/djm56/kirsch/internal/app"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

// patchFileDir is the fixed directory the temporary M1/M7 debug `/patch`
// command reads from — plan/milestones/milestone-2.md Task 7.5 specifies it
// verbatim: "/patch <file> (apply a diff from testdata/patches/)". The
// command takes a bare filename, so the directory has to be supplied by the
// wiring rather than by the caller.
const patchFileDir = "testdata/patches"

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
	wireCallbacks(&m, a, log, ws)

	// Input is normalised on the way in so that terminals which encode Home and
	// End as SS3 — macOS Terminal among them — reach Bubble Tea as the CSI forms
	// its key table actually carries. See tui.NormalizeInput.
	//
	// The option is appended rather than always passed because NormalizeInput
	// returns nil when stdin is not a terminal, and tea.WithInput(nil) disables
	// input outright. Omitting it leaves Bubble Tea to open /dev/tty for itself,
	// which is what keeps a piped or redirected invocation driveable.
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if in := tui.NormalizeInput(os.Stdin); in != nil {
		opts = append(opts, tea.WithInput(in))
	}
	p := tea.NewProgram(m, opts...)
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

// wireCallbacks wires the TUI model's callbacks to the app and telemetry logger.
// This function encapsulates the callback assignments that connect the TUI to the app.
// By extracting this, we enable tests to use the same wiring path instead of
// duplicating or hand-writing callbacks.
//
// Deleting the m.ResolveApproval assignment in this function breaks the approval flow
// and causes Request to block indefinitely, making user approvals impossible.
func wireCallbacks(m *tui.Model, a *app.App, log *telemetry.Logger, ws *workspace.Workspace) {
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
	// ResolveApproval wires the TUI's approval resolution to the app's decision channel.
	// Deleting this assignment breaks the approval flow and causes Request to block indefinitely.
	m.ResolveApproval = func(id int64, outcome tui.ApprovalOutcome) {
		log.Debug("tui resolved approval", "id", id, "outcome", outcome)
		appOutcome := tui.ToAppOutcome(outcome)
		// Defensive: ToAppOutcome returns 0..ApprovalOutcomeCancelled by construction; this bound satisfies G115 and fails safe to Cancelled.
		if appOutcome < 0 || appOutcome > int(app.ApprovalOutcomeCancelled) {
			log.Error("invalid approval outcome from TUI", "value", appOutcome)
			appOutcome = int(app.ApprovalOutcomeCancelled)
		}
		a.Resolve(id, app.ApprovalOutcome(appOutcome))
	}
	// GetGrants wires the TUI's grant listing to the app's policy.
	m.GetGrants = func() []string {
		return a.Grants()
	}
	// GetGrantCount wires the TUI's grant count to the app's policy.
	m.GetGrantCount = func() int {
		return len(a.Grants())
	}
	// ClearGrants wires the TUI's grant clearing to the app's policy.
	m.ClearGrants = func() {
		a.ClearGrants()
	}
	// ResolvePatchFile wires the TUI's patch file resolution to the
	// workspace. The filename is joined onto patchFileDir before it reaches
	// ws.Resolve, so /patch create-file.diff actually finds
	// testdata/patches/create-file.diff instead of a file that has never
	// existed at the workspace root.
	//
	// ws.Resolve is the workspace's own security boundary: it refuses
	// anything that resolves outside the workspace root, including through a
	// symlink resolved stepwise. That bounds the result to the workspace,
	// not to patchFileDir specifically — a filename such as
	// "../../internal/policy/policy.go" stays inside the workspace while
	// leaving the patch directory. Containment is re-checked below against
	// patchFileDir once ws.Resolve has settled the symlink question, so
	// /patch can only ever read what the milestone spec scoped it to.
	m.ResolvePatchFile = func(filename string) (string, error) {
		resolved, err := ws.Resolve(filepath.Join(patchFileDir, filename))
		if err != nil {
			return "", err
		}
		if rel := ws.Rel(resolved); rel != patchFileDir && !strings.HasPrefix(rel, patchFileDir+"/") {
			return "", fmt.Errorf("patch file %q is outside %s", filename, patchFileDir)
		}
		content, err := os.ReadFile(resolved) // #nosec G304 -- path is contained to testdata/patches by ws.Resolve (internal/workspace) + the prefix check above
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
}
