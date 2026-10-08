// Command kirsch is a terminal-native coding agent.
//
// Milestone 1: the TUI drives real read-only tools against a real repository.
// There is no model, no patching and no arbitrary command execution anywhere in
// the binary — the only subprocesses are read-only git and ripgrep.
package main

import (
	"crypto/rand"
	"encoding/hex"
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
		Status: a.ActiveModelInfo(),
	})
	wireCallbacks(&m, a, log)

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

	for _, w := range warnings {
		log.Warn("config", "detail", w.String())
	}
	log.Info("starting",
		"version", version,
		"workspace", ws.Root,
		"git", ws.IsGit,
		"tools", a.Registry().Names())

	// Generate one session ID per invocation for the opencode endpoint and tell
	// the app the version before any turn can start. This must happen after the
	// program is attached so onboarding messages have somewhere to go.
	startupPostAttach(a, p, version)

	_, err = p.Run()
	return err
}

// startupPostAttach runs the sequence that must happen after the TUI program
// is attached: identity, version, and onboarding. It is extracted so that
// run() and the startup tests share the exact same ordering.
func startupPostAttach(a *app.App, p *tea.Program, version string) {
	a.Attach(p)
	a.SetSessionID(newSessionID())
	a.SetVersion(version)
	startSessionCheck(a)
	onboardingCheck(a)
}

// startSessionCheck is a package-level hook so tests can verify that startup
// assembles the session prompt without needing to run the full TUI.
var startSessionCheck = func(a *app.App) {
	a.StartSession(os.Getenv)
}

// onboardingCheck is a package-level hook so tests can verify that startup
// actually calls CheckOnboarding without needing to run the full TUI.
var onboardingCheck = func(a *app.App) {
	a.CheckOnboarding(os.Getenv)
}

// newSessionID returns a 16-byte hex string suitable for the opencode endpoint.
func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a non-cryptographic but still unique identifier. This
		// path is extremely unlikely and is better than failing to start.
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// wireCallbacks wires the TUI model's callbacks to the app and telemetry logger.
// This function encapsulates the callback assignments that connect the TUI to the app.
// By extracting this, we enable tests to use the same wiring path instead of
// duplicating or hand-writing callbacks.
//
// Deleting the m.ResolveApproval assignment in this function breaks the approval flow
// and causes Request to block indefinitely, making user approvals impossible.
func wireCallbacks(m *tui.Model, a *app.App, log *telemetry.Logger) {
	// Submit hands ordinary composer text to the app, which starts a real
	// model-driven turn. Slash commands never reach this callback.
	m.Submit = func(text string) {
		log.Debug("tui submitted", "text", text)
		a.Submit(text, os.Getenv)
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
	// StatusInfo wires the /status command to the app's config/Env summary.
	m.StatusInfo = func() tui.StatusInfoMsg {
		return a.StatusInfo(os.Getenv)
	}
}
