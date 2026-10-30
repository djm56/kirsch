package doctor

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/djm56/kirsch/internal/agent/prompt"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/tool"
	"github.com/djm56/kirsch/internal/workspace"
)

// Names of the checks this file reports. Tests and the report share them.
// Report lines match "[a-z]+" for the name, so each is one lower-case word.
const (
	checkGit        = "git"
	checkRG         = "rg"
	checkRepository = "repository"
	checkConfig     = "config"
	checkKey        = "key"
	checkBackend    = "backend"
	checkModel      = "model"
	checkContext    = "context"
)

// envNameRe is what a name printed by the key line must match: upper-case
// letters, digits and underscores, starting with a letter or underscore, at most
// 64 characters. That is the usual shape of an environment variable name and
// rules out the usual shapes of a pasted key (mixed case, hyphens, long tokens).
// It is a heuristic, not a proof: a pasted key that is itself 64 or fewer
// upper-case letters, digits and underscores still matches and would be printed.
var envNameRe = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,63}$`)

// skippedNoConfig is the detail of the key, model and context lines when the
// config did not load: there is no endpoint or project-file list to read, and
// guessing the built-in default would report a setting kirsch never used.
const skippedNoConfig = "skipped: config did not load"

// skippedNoWorkspace is the detail of the context line when the config loaded
// but no workspace was detected (so kirsch itself would not have started): the
// loader needs a workspace to resolve the candidate files against.
const skippedNoWorkspace = "skipped: no workspace"

// noRepoDetail is the detail of the repository check's failure when no
// --workspace was given and workspace.Detect found no enclosing repository.
const noRepoDetail = "not in a Git repository"

// Options carries what Run takes from the caller instead of reading it itself:
// the --workspace value, the config paths, the home directory, the PATH lookup
// used for the git and rg presence probes and the environment reader used for
// the key line. Only those are injected.
// workspace.Detect is not: it reads the process working directory and runs the
// real git found on PATH, whatever LookPath says. A test that fakes a missing
// git through LookPath therefore changes the git line, not what Detect sees.
type Options struct {
	// WorkspaceFlag is the value of --workspace; empty means "the enclosing
	// Git repository". It goes to workspace.Detect unchanged.
	WorkspaceFlag string
	// GlobalConfigPath is the global config file location; empty means none.
	GlobalConfigPath string
	// HomeDir is used for ~ expansion in the config; empty disables it.
	HomeDir string
	// LookPath finds an executable on PATH for the git and rg presence probes
	// only. Nil means exec.LookPath.
	LookPath func(file string) (string, error)
	// Getenv reads an environment variable for the key line only (main passes
	// os.Getenv). Nil means an empty environment: no variable is set. It is
	// not the whole environment seam: the backend line's tool.Backend reads
	// PATH itself, and the default LookPath (exec.LookPath) and
	// workspace.Detect read the process environment too.
	Getenv func(key string) string
}

// Outcome is what Run returns: the results to report, plus the facts that the
// lines reading the configuration (key, model, context) use, so the config is
// not loaded a second time.
type Outcome struct {
	// Results are the check results, in report order.
	Results []Result
	// ConfigLoaded is true only when config.Load and Config.Validate both
	// succeeded. When it is false, Config holds no usable configuration and
	// must not be read.
	ConfigLoaded bool
	// Config is the loaded, validated configuration. Meaningful only when
	// ConfigLoaded is true.
	Config config.Config
	// Workspace is the detected workspace, or nil when detection failed. It can
	// be non-nil while the repository check only warned (an explicit
	// --workspace that is not a Git repository).
	Workspace *workspace.Workspace
}

// Run performs the environment, repository and config probes: git, rg, the
// repository and the config file, then the key, search_code backend, model and
// project-context lines (resolvedResults). It never stops at the first failure; every
// check that can run does run. It only reads: it creates nothing, takes no lock
// and uses no network.
//
// The pass/fail rule is O5: a check fails when the problem would stop kirsch
// starting, and warns when kirsch still starts but works less well. "Would stop
// kirsch starting" is read off the start-up steps of main.run that come before
// the TUI is built: workspace.Detect, config.Load and Config.Validate. Those are
// the only failing outcomes here (the git line fails only in the case where
// Detect would fail for want of git). main.run can also return an error from
// flag parsing, an unknown command, startupPostAttach and the program itself;
// doctor does not probe those.
func Run(o Options) Outcome {
	lookPath := o.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	var out Outcome

	gitPath, gitErr := lookPath("git")
	gitOK := gitErr == nil
	switch {
	case gitOK:
		out.Results = append(out.Results, Result{StatusOK, checkGit, "found at " + gitPath})
	case o.WorkspaceFlag == "":
		// Without --workspace, Detect asks git for the repository root and
		// returns ErrNoWorkspace when it cannot, so kirsch will not start.
		out.Results = append(out.Results, Result{
			StatusFail, checkGit,
			"not found on PATH; kirsch needs git to find the repository",
		})
	default:
		// An explicit --workspace skips that question: Detect marks the folder
		// as not a repository and kirsch starts without its Git features.
		out.Results = append(out.Results, Result{
			StatusWarn, checkGit,
			"not found on PATH; kirsch starts with --workspace but " + gitFeaturesOff,
		})
	}

	if rgPath, err := lookPath("rg"); err == nil {
		out.Results = append(out.Results, Result{StatusOK, checkRG, "found at " + rgPath})
	} else {
		out.Results = append(out.Results, Result{
			StatusWarn, checkRG,
			"not found on PATH; search_code uses the pure-Go fallback",
		})
	}

	ws, wsErr := workspace.Detect(o.WorkspaceFlag)
	out.Workspace = ws
	out.Results = append(out.Results, repositoryResults(ws, wsErr, gitOK)...)

	projectPath := ""
	if ws != nil {
		projectPath = config.ProjectPath(ws.Root)
	}
	out.Results = append(out.Results, loadConfig(&out, o, projectPath)...)
	out.Results = append(out.Results, resolvedResults(out, o.Getenv)...)
	return out
}

// resolvedResults returns the four lines that need the resolved configuration,
// in report order: key, backend, model, context. The backend line does not read
// the config, so it is always produced; the other three read the default
// endpoint or the project-file list, so with no loaded config they each say
// skippedNoConfig. A skipped line is a warn: nothing was checked, which is not
// a pass, and it cannot fail the run because the config line has already failed
// it (ConfigLoaded is false only when that line is a fail).
//
// Statuses follow the O5 rule against main.run's start path: none of these four
// can stop kirsch starting (run reads no key, model or context file before the
// TUI is built), so none is a fail. Each is a warn when kirsch starts but works
// less well (no key; pure-Go search; a model not in the table; an unusable
// project-context file).
func resolvedResults(out Outcome, getenv func(string) string) []Result {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	backend := backendResult()
	if !out.ConfigLoaded {
		return []Result{
			{StatusWarn, checkKey, skippedNoConfig},
			backend,
			{StatusWarn, checkModel, skippedNoConfig},
			{StatusWarn, checkContext, skippedNoConfig},
		}
	}
	// Validate has accepted the config, so the default endpoint exists.
	epName := out.Config.Provider.Default
	ep := out.Config.Provider.Endpoints[epName]
	return []Result{
		keyResult(epName, ep, getenv),
		backend,
		modelResult(epName, ep),
		contextResult(out.Config.Context, out.Workspace),
	}
}

// keyResult reports whether the default endpoint's API key can be resolved and
// which variable wins (EndpointConfig.ResolveKey: KIRSCH_<name> before <name>).
// Only the variable's name is printed. The key value is discarded where
// ResolveKey returns it and never reaches a Result. No key is a warn: kirsch
// starts, shows its onboarding notice (App.CheckOnboarding) and refuses each
// prompt with an auth error (App.Submit).
//
// config.Validate only checks that api_key_env is non-empty and its secret-key
// guard exempts that field, so the setting can hold anything, including a key
// pasted by mistake. When it does not match envNameRe it is never printed, on
// either branch; the line is a warn saying the setting does not look like a
// variable name.
func keyResult(epName string, ep config.EndpointConfig, getenv func(string) string) Result {
	if !envNameRe.MatchString(ep.APIKeyEnv) {
		return Result{
			StatusWarn, checkKey,
			fmt.Sprintf("endpoint %s: api_key_env does not look like an environment variable name (value withheld); fix the setting", epName),
		}
	}
	_, source := ep.ResolveKey(getenv)
	if source == "" {
		return Result{
			StatusWarn, checkKey,
			fmt.Sprintf("endpoint %s: no key found (set %s or %s); kirsch starts but cannot send a prompt",
				epName, "KIRSCH_"+ep.APIKeyEnv, ep.APIKeyEnv),
		}
	}
	return Result{StatusOK, checkKey, fmt.Sprintf("endpoint %s: key found in %s", epName, source)}
}

// backendResult reports which implementation search_code would use, from the
// tool's own Backend method on a zero-value tool (ForcePureGo unset, as the app
// registers it). Backend looks rg up through the process PATH, not through
// Options.LookPath, so this line and the rg line agree only when LookPath is
// exec.LookPath, as main passes it. pure-go is a warn: the same condition as the
// rg line's warn, search still works.
func backendResult() Result {
	b := (&tool.SearchCode{}).Backend()
	if b == "ripgrep" {
		return Result{StatusOK, checkBackend, "search_code uses " + b}
	}
	return Result{StatusWarn, checkBackend, "search_code uses " + b + " (rg is not on PATH)"}
}

// modelResult reports the default endpoint's model as provider.LookupModel
// resolves it: the id, the context window and whether the table knows the model.
// An unknown model is a warn: kirsch starts and uses the table's conservative
// fallback window (App.CheckOnboarding shows the same notice).
func modelResult(epName string, ep config.EndpointConfig) Result {
	info := provider.LookupModel(ep.Model)
	if !info.Known {
		return Result{
			StatusWarn, checkModel,
			fmt.Sprintf("endpoint %s, model %s: not in the table; conservative fallback, context window %d tokens",
				epName, ep.Model, info.ContextWindow),
		}
	}
	return Result{
		StatusOK, checkModel,
		fmt.Sprintf("endpoint %s, model %s: context window %d tokens", epName, info.ID, info.ContextWindow),
	}
}

// contextResult reports which project-context file the app would load, calling
// prompt.LoadProjectContext with the inputs App.loadAndStoreSessionPrompt gives
// it: cfg.ProjectFiles, cfg.MaxProjectContextBytes and the *workspace.Workspace
// as the Engine. The size is the file's size on disk. No file is ok ("none").
// A refused or non-regular candidate, or an error from the loader (it could not
// stat or read a candidate), is a warn: the app logs it and starts anyway,
// without that context. On an error no candidate was chosen, so the detail says
// a candidate could not be read and carries the loader's error, not "none". A
// nil workspace is skipped.
func contextResult(cfg config.ContextConfig, ws *workspace.Workspace) Result {
	if ws == nil {
		return Result{StatusWarn, checkContext, skippedNoWorkspace}
	}
	_, chosen, size, warnings, err := prompt.LoadProjectContext(cfg.ProjectFiles, cfg.MaxProjectContextBytes, ws)
	detail := "none; looked for " + strings.Join(cfg.ProjectFiles, ", ")
	if len(cfg.ProjectFiles) == 0 {
		detail = "none; no project files configured"
	}
	if chosen != "" {
		detail = fmt.Sprintf("%s, %d bytes", chosen, size)
	}
	status := StatusOK
	if len(warnings) > 0 {
		status = StatusWarn
		detail += "; " + strings.Join(warnings, "; ")
	}
	if err != nil {
		status = StatusWarn
		// The loader returns no chosen file with an error, so the "none; looked
		// for" prefix would be untrue: a candidate was there but unreadable.
		detail = "a candidate could not be read: " + err.Error()
		if len(warnings) > 0 {
			detail += "; " + strings.Join(warnings, "; ")
		}
	}
	return Result{status, checkContext, detail}
}

// gitFeaturesOff finishes the sentence of the warnings issued when kirsch will
// run in a folder it does not treat as a Git repository. The features named are
// examples of what Workspace.IsGit guards, not a complete list: the /diff
// command (app.GetDiff), the git_status and git_diff tools (internal/tool/git.go),
// the branch readout (refreshMeta) and the dirty count (DirtyCount, which
// returns 0 outside a repository).
const gitFeaturesOff = "the Git features are off, such as /diff, git_status, git_diff, the branch display and the dirty count"

// repositoryResults turns the outcome of workspace.Detect into zero or one
// result.
//
// The status follows kirsch's own start path (main.run):
//   - no --workspace and no repository found (ErrNoWorkspace): fail, kirsch
//     refuses to start.
//   - --workspace naming a folder that does not exist or is not a directory
//     (any other Detect error): fail, kirsch returns that error.
//   - --workspace naming an existing folder that is not a Git repository
//     (IsGit false, no error): warn, kirsch starts with its Git features off.
//   - a repository: ok, with its root.
//
// When git is missing the repository cannot be judged: Detect reports
// ErrNoWorkspace, or IsGit false, for a missing git exactly as it does for a
// folder that is not a repository. Saying "not in a Git repository" then would
// be a guess, so the check produces no line (DIR-002); the git line already
// carries the status. An error that is not about git (an explicit --workspace
// that does not exist) is still reported.
//
// Limit: Detect treats any failure of the git command, including a git that is
// present but errors (for example "dubious ownership"), as "no repository", so
// doctor cannot tell that case from a folder that really is not a repository.
func repositoryResults(ws *workspace.Workspace, err error, gitOK bool) []Result {
	switch {
	case errors.Is(err, workspace.ErrNoWorkspace):
		if !gitOK {
			return nil
		}
		return []Result{{StatusFail, checkRepository, noRepoDetail}}
	case err != nil:
		return []Result{{StatusFail, checkRepository, err.Error()}}
	case !ws.IsGit:
		if !gitOK {
			return nil
		}
		return []Result{{
			StatusWarn, checkRepository,
			ws.Root + " is not a Git repository; kirsch starts, but " + gitFeaturesOff,
		}}
	}
	return []Result{{StatusOK, checkRepository, ws.Root}}
}

// loadConfig runs config.Load and Validate, records the outcome on out, and
// returns the config results: one warn per config warning, then either one fail
// naming the problem or one ok line. Both kinds of failure are ones that stop
// main.run, so they fail here. projectPath is empty exactly when workspace
// detection failed (config.ProjectPath of a real root is never empty); the ok
// line then says the project file was not read.
//
// The ok lines say "settings valid", not "file loaded": config.Load treats a
// missing file as empty and does not report which files existed, so doctor
// cannot say whether the global or project file was actually read.
func loadConfig(out *Outcome, o Options, projectPath string) []Result {
	cfg, warnings, err := config.Load(config.Options{
		GlobalPath:  o.GlobalConfigPath,
		ProjectPath: projectPath,
		HomeDir:     o.HomeDir,
	})
	var results []Result
	for _, w := range warnings {
		results = append(results, Result{StatusWarn, checkConfig, w.String()})
	}
	if err != nil {
		// Load errors already name the file ("parse <path>: ...").
		return append(results, Result{StatusFail, checkConfig, err.Error()})
	}
	if err := cfg.Validate(); err != nil {
		// Validate checks the merged result, so it cannot name one file; it
		// names the files that were consulted instead.
		return append(results, Result{
			StatusFail, checkConfig,
			"invalid settings (" + consulted(o.GlobalConfigPath, projectPath) + "): " + err.Error(),
		})
	}
	out.ConfigLoaded = true
	out.Config = cfg
	if projectPath == "" {
		return append(results, Result{
			StatusOK, checkConfig,
			"settings valid; project file not read (no workspace)",
		})
	}
	return append(results, Result{StatusOK, checkConfig, "settings valid"})
}

// consulted names the config files Load was pointed at, skipping empty paths.
func consulted(global, project string) string {
	var parts []string
	if global != "" {
		parts = append(parts, "global "+global)
	}
	if project != "" {
		parts = append(parts, "project "+project)
	}
	if len(parts) == 0 {
		return "no config files"
	}
	return strings.Join(parts, ", ")
}
