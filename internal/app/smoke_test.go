package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/agent"
	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/patch"
	"github.com/djm56/kirsch/internal/policy"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/telemetry"
	"github.com/djm56/kirsch/internal/tui"
	"github.com/djm56/kirsch/internal/workspace"
)

const smokePrompt = "Add input validation to the Divide function and cover it with a test"

// Compiled regexes for parseReportSummary.
var (
	reBullets    = regexp.MustCompile(`^[\s]*[-*•]\s+`)
	reNumbered   = regexp.MustCompile(`^[\s]*\d+\.\s+`)
	reHeaders    = regexp.MustCompile(`^[\s]*#+\s+`)
	reBold       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBackticks  = regexp.MustCompile("`(.+?)`")
	reTrailDash  = regexp.MustCompile(`\s*—\s+note\s*:?\s*$`)
	reTrailMinus = regexp.MustCompile(`\s*-\s+note\s*:?\s*$`)
	reTrailColon = regexp.MustCompile(`:\s+note\s*$`)
)

// smokeEnv is one App wired to a workspace copy of repo-go-module and a provider.
type smokeEnv struct {
	t    *testing.T
	a    *App
	c    *collector
	dir  string // temp workspace root
	msgs []any  // collected messages after settling
}

// smokeResult holds the analysis of a smoke test turn's outcome.
type smokeResult struct {
	FinalText        string                         // The final assistant message text
	ApprovalRequests map[string]int64               // ToolName -> ApprovalID; tracks each approval request
	CompletedTools   map[int64]tui.ToolCompletedMsg // ID -> ToolCompletedMsg; tracks each completion
	ReportLabels     map[string]string              // "Summary" -> "...", "Files changed" -> "...", etc.
	FilesChanged     []string                       // Parsed files from "Files changed" section
	CommandsRun      []string                       // Parsed commands from "Commands run" section
}

// newSmokeEnv creates a copy of testdata/repo-go-module in a temp dir, initializes
// it as a git repo, and builds an App on it.
//
// getenv selects the mode. A nil getenv is offline mode: config.Defaults() and
// os.Getenv for the session prompt. A non-nil getenv is live mode: the config
// is loaded with config.Load the same way cmd/kirsch/main.go does, and getenv is
// used for the session prompt.
//
// In both modes cfg.Session.StorageDir is pointed at a fresh temp directory
// named "sessions". App.stateRoot takes the parent of a "sessions" directory as
// the state root, so the session files, index.json, usage.json and
// usage.json.lock all land in that temp directory and not under ~/.local/share.
// (The harness never calls App.AcquireLock, so no .lock file is written either.)
//
// providerSetup is an optional callback that runs after New to wire a provider;
// when nil the provider comes from the config.
func newSmokeEnv(t *testing.T, getenv func(string) string, providerSetup func(a *App)) *smokeEnv {
	t.Helper()
	isLiveMode := getenv != nil
	if getenv == nil {
		getenv = os.Getenv
	}
	dir := t.TempDir()

	// Copy testdata/repo-go-module into the temp dir.
	srcRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-go-module"))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	srcWd, err := os.Open(srcRoot)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer srcWd.Close()
	entries, err := srcWd.Readdirnames(-1)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, name := range entries {
		src := filepath.Join(srcRoot, name)
		dst := filepath.Join(dir, name)
		if err := copyPath(src, dst); err != nil {
			t.Fatalf("copy %s: %v", name, err)
		}
	}

	// Initialize as a git repo.
	gitIn(t, dir, "init")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-m", "Initial commit")

	// Build the App on the copied workspace.
	ws, err := workspace.Detect(dir)
	if err != nil {
		t.Fatalf("workspace.Detect: %v", err)
	}
	c := newCollector(-1)

	// For live tests, load config the same way main.go does.
	// For offline tests, use defaults.
	var cfg config.Config
	if isLiveMode {
		home, _ := os.UserHomeDir()
		cfg, _, err = config.Load(config.Options{
			GlobalPath:  config.GlobalPath(getenv("XDG_CONFIG_HOME"), home),
			ProjectPath: config.ProjectPath(ws.Root),
			HomeDir:     home,
		})
		if err != nil {
			t.Fatalf("config.Load: %v", err)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("config.Validate: %v", err)
		}
	} else {
		cfg = config.Defaults()
	}

	// Isolate every file the App persists. This is a separate temp dir from the
	// workspace so session files never show up in the workspace's git status.
	cfg.Session.StorageDir = filepath.Join(t.TempDir(), "sessions")

	a := New(ws, cfg, telemetry.Disabled())
	a.sendFn = c.send
	t.Cleanup(a.Close)

	// Apply provider setup if provided.
	if providerSetup != nil {
		providerSetup(a)
	}

	// Start a session with the configured getenv.
	a.StartSession(getenv)

	e := &smokeEnv{t: t, a: a, c: c, dir: dir}
	return e
}

// copyPath recursively copies a file or directory.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	// Copy file.
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()
	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstF.Close()
	_, err = io.Copy(dstF, srcF)
	if err != nil {
		dstF.Close()
		return err
	}
	return dstF.Chmod(info.Mode())
}

// submit runs Submit with the smoke prompt and returns the messages once the
// turn completes and the collector is quiet. If ctx has a deadline, it uses that;
// otherwise it uses a default 30-second deadline for offline tests.
func (e *smokeEnv) submit(ctx context.Context, getenv func(string) string) []any {
	e.t.Helper()
	if getenv == nil {
		getenv = func(string) string { return "fake-key" }
	}

	// Determine the deadline: use ctx's if present, otherwise use 30s for offline tests.
	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(30 * time.Second)
	}

	// Submit starts the turn and returns; poll the collector for the terminal
	// message below, up to the deadline.
	e.a.Submit(smokePrompt, getenv)

	// Wait for terminal message or deadline.
	for {
		msgs := e.c.snapshot()
		if findMsgType[tui.TurnCompleteMsg](msgs) || findMsgType[tui.TurnErrorMsg](msgs) || findMsgType[tui.TurnCancelledMsg](msgs) {
			break
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("no terminal message after deadline; got %d messages", len(msgs))
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for in-flight work and quiescence.
	e.a.inFlight.Wait()
	last, quietSince := e.c.count(), time.Now()
	for time.Since(quietSince) < 100*time.Millisecond {
		time.Sleep(10 * time.Millisecond)
		if n := e.c.count(); n != last {
			last, quietSince = n, time.Now()
		}
	}

	e.msgs = e.c.snapshot()
	return e.msgs
}

// toJSONRaw converts a map to json.RawMessage.
func toJSONRaw(m map[string]interface{}) json.RawMessage {
	b, _ := json.Marshal(m)
	return b
}

// diffHunk renders one unified-diff hunk whose header counts are computed from
// the lines, so a variant cannot carry a wrong count. ctx lines are kept as
// context and added lines are appended after them. The new start equals the old
// start, which holds for the hunks below because no earlier hunk in the same
// file changes the line count.
func diffHunk(oldStart int, ctx, added []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, len(ctx), oldStart, len(ctx)+len(added))
	for _, l := range ctx {
		b.WriteString(" " + l + "\n")
	}
	for _, l := range added {
		b.WriteString("+" + l + "\n")
	}
	return b.String()
}

// diffOpts selects the variant of the scripted patch.
type diffOpts struct {
	guard     bool // calc/divide.go gets the b == 0 guard
	zeroTest  bool // calc/divide_test.go gets TestDivideByZero (a zero-divisor test)
	extraFile bool // the patch also creates calc/extra.go
}

// goodDiffOpts is the variant a correct agent would produce.
var goodDiffOpts = diffOpts{guard: true, zeroTest: true}

// buildSmokeDiff builds the unified diff the scripted provider sends through
// apply_patch. It always modifies calc/divide.go (removes the two "no
// validation" comment lines; with opts.guard also adds an import, a sentinel
// error and the b == 0 guard) and calc/divide_test.go (always appends
// TestDivideNormal, which has no zero divisor; with opts.zeroTest also appends
// TestDivideByZero). With opts.extraFile it also creates calc/extra.go.
func buildSmokeDiff(opts diffOpts) string {
	var d strings.Builder

	// calc/divide.go: the fixture has 9 lines and the hunk spans all of them.
	// The two comment lines are always removed: old count 9, new count 7 plus
	// the added lines.
	newN := 9 - 2
	var body strings.Builder
	body.WriteString(" package calc\n \n")
	if opts.guard {
		body.WriteString("+import \"errors\"\n+\n+var ErrDivideByZero = errors.New(\"divide by zero\")\n+\n")
		newN += 4
	}
	body.WriteString(" // Divide returns a divided by b.\n //\n")
	body.WriteString("-// It has no input validation. Milestone 4's smoke test adds it; do not fix it\n")
	body.WriteString("-// here, or the smoke test has nothing to do.\n")
	body.WriteString(" func Divide(a, b float64) (float64, error) {\n")
	if opts.guard {
		body.WriteString("+\tif b == 0 {\n+\t\treturn 0, ErrDivideByZero\n+\t}\n")
		newN += 3
	}
	body.WriteString(" \treturn a / b, nil\n }\n")
	d.WriteString("--- a/calc/divide.go\n+++ b/calc/divide.go\n")
	fmt.Fprintf(&d, "@@ -1,9 +1,%d @@\n", newN)
	d.WriteString(body.String())

	// calc/divide_test.go: the fixture has 13 lines; the hunk is its last four.
	ctx := []string{
		"\tif got != 2 {",
		"\t\tt.Errorf(\"Divide(6, 3) = %v, want 2\", got)",
		"\t}",
		"}",
	}
	var tests []string
	if opts.zeroTest {
		tests = append(tests,
			"",
			"func TestDivideByZero(t *testing.T) {",
			"\tgot, err := Divide(10, 0)",
			"\tif err == nil {",
			"\t\tt.Errorf(\"Divide(10, 0) returned nil error; want non-nil\")",
			"\t}",
			"\tif got != 0 {",
			"\t\tt.Errorf(\"Divide(10, 0) returned %v; want 0\", got)",
			"\t}",
			"}",
		)
	}
	tests = append(tests,
		"",
		"func TestDivideNormal(t *testing.T) {",
		"\tgot, _ := Divide(10, 2)",
		"\tif got != 5 {",
		"\t\tt.Errorf(\"Divide(10, 2) = %v; want 5\", got)",
		"\t}",
		"}",
	)
	d.WriteString("--- a/calc/divide_test.go\n+++ b/calc/divide_test.go\n")
	d.WriteString(diffHunk(10, ctx, tests))

	if opts.extraFile {
		d.WriteString("--- /dev/null\n+++ b/calc/extra.go\n@@ -0,0 +1,3 @@\n+package calc\n+\n+const Extra = 1\n")
	}
	return d.String()
}

// buildSmokeTestDiff builds the diff a correct agent would produce.
func buildSmokeTestDiff() string { return buildSmokeDiff(goodDiffOpts) }

// TestParseReportSummary tests the parser with various markdown formats.
// It verifies stripping of bullets, backticks, bold markers, and trailing notes.
// It tests headers in forms: ## header, **header**, 2. header, header:
func TestParseReportSummary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "standard markdown headers",
			input: `## Summary
	This is a summary.

	## Files changed
	- calc/divide.go
	- calc/divide_test.go

	## Commands run
	go test ./...

	## Limitations
	None`,
			expected: map[string]string{
				"Summary":       "This is a summary.",
				"Files changed": "- calc/divide.go\n- calc/divide_test.go",
				"Commands run":  "go test ./...",
				"Limitations":   "None",
			},
		},
		{
			name: "bold headers",
			input: `**Summary**
	This is a summary.

	**Files changed**
	calc/divide.go
	calc/divide_test.go

	**Commands run**
	go test ./...`,
			expected: map[string]string{
				"Summary":       "This is a summary.",
				"Files changed": "calc/divide.go\ncalc/divide_test.go",
				"Commands run":  "go test ./...",
			},
		},
		{
			name: "numbered headers",
			input: `1. Summary
	This is a summary.

	2. Files changed
	* calc/divide.go
	* calc/divide_test.go

	3. Commands run
	go test ./...`,
			expected: map[string]string{
				"Summary":       "This is a summary.",
				"Files changed": "* calc/divide.go\n* calc/divide_test.go",
				"Commands run":  "go test ./...",
			},
		},
		{
			name: "headers with colons",
			input: `Summary:
	This is a summary.

	Files changed:
	calc/divide.go
	calc/divide_test.go

	Commands run:
	go test ./...`,
			expected: map[string]string{
				"Summary":       "This is a summary.",
				"Files changed": "calc/divide.go\ncalc/divide_test.go",
				"Commands run":  "go test ./...",
			},
		},
		{
			name: "mixed formats with trailing notes",
			input: `## Summary — note
	This is a summary.

	## Files changed - note
	calc/divide.go

	## Commands run: note
	go test ./...`,
			expected: map[string]string{
				"Summary":       "This is a summary.",
				"Files changed": "calc/divide.go",
				"Commands run":  "go test ./...",
			},
		},
		{
			name: "backtick stripping in content lines",
			input: `## Summary
	This has ` + "`code`" + `.

	## Files changed
	` + "`calc/divide.go`" + `
	` + "`calc/divide_test.go`" + ``,
			expected: map[string]string{
				"Summary":       "This has `code`.",
				"Files changed": "calc/divide.go\ncalc/divide_test.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseReportSummary(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("parseReportSummary returned %d labels, want %d; got %v", len(got), len(tt.expected), got)
			}
			for label, expectedContent := range tt.expected {
				gotContent, ok := got[label]
				if !ok {
					t.Errorf("missing label %q", label)
					continue
				}
				if gotContent != expectedContent {
					t.Errorf("label %q: got %q, want %q", label, gotContent, expectedContent)
				}
			}
		})
	}
}

// autoApprove allows every approval request the App raises, once each, until
// the test ends. The polling goroutine is stopped by a context cancelled in
// t.Cleanup, and the cleanup waits for it to exit. Cleanups run last-in
// first-out, so this one runs before the App's own Close registered in
// newSmokeEnv. Live and offline tests share this helper.
func (e *smokeEnv) autoApprove() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	e.t.Cleanup(func() {
		cancel()
		<-done
	})
	go func() {
		defer close(done)
		seen := make(map[int64]bool)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, m := range e.c.snapshot() {
					if req, ok := m.(tui.ApprovalRequestedMsg); ok && !seen[req.ID] {
						seen[req.ID] = true
						e.a.ResolveToolApproval(req.ID, policy.DecisionAllow)
					}
				}
			}
		}
	}()
}

// smokeGoTestArgv is the command the scripted agent runs. It runs only the
// original TestDivide, so it passes against every patch variant below and the
// agent's own go test is not what flags a variant.
var smokeGoTestArgv = []string{"go", "test", "-run", "^TestDivide$", "./calc/"}

// smokeReportText renders a final report with the four labels. files is the
// "Files changed" body; limitationsLabel false replaces the Limitations header
// with prose that still contains the word.
func smokeReportText(files []string, limitationsLabel bool) string {
	limits := "## Limitations\nNone."
	if !limitationsLabel {
		limits = "The Limitations of this change: none."
	}
	return "## Summary\nValidation added.\n\n## Files changed\n" + strings.Join(files, "\n") +
		"\n\n## Commands run\ngo test -run ^TestDivide$ ./calc/\n\n" + limits
}

// smokeScript is the scripted provider conversation: apply_patch with diff,
// optionally one run_command turn, then the final report.
func smokeScript(diff string, runCommand bool, report string) []provider.Turn {
	script := []provider.Turn{{ToolCalls: []provider.ToolCall{{
		ID:   "call-1",
		Name: "apply_patch",
		Input: toJSONRaw(map[string]interface{}{
			"diff":        diff,
			"description": "Add divide-by-zero validation and tests",
		}),
	}}}}
	if runCommand {
		script = append(script, provider.Turn{ToolCalls: []provider.ToolCall{{
			ID:    "call-2",
			Name:  "run_command",
			Input: toJSONRaw(map[string]interface{}{"argv": smokeGoTestArgv}),
		}}})
	}
	return append(script, provider.Turn{Text: report})
}

// runOfflineSmoke drives App.Submit with a scripted fake provider
// (provider.NewFake, wired through a.providerBuilder) and returns the
// checkSmokeRun errors. Entry point is App.Submit, the same as TestSmokeLive.
func runOfflineSmoke(t *testing.T, script []provider.Turn) []string {
	t.Helper()
	e := newSmokeEnv(t, nil, func(a *App) {
		fake := provider.NewFake(script...)
		a.providerBuilder = func(string, config.EndpointConfig, string, string, string, *telemetry.Logger) (provider.Provider, error) {
			return fake, nil
		}
	})
	e.autoApprove()
	msgs := e.submit(context.Background(), nil)
	if !findMsgType[tui.TurnCompleteMsg](msgs) {
		t.Fatalf("no TurnCompleteMsg in %#v", msgs)
	}
	return checkSmokeRun(t, e, collectSmokeResult(t, e))
}

// TestSmokeOffline runs the smoke checks against a scripted fake provider (no
// API spend). The positive control must produce no errors. Each negative
// control breaks exactly one property of the otherwise-correct run and must
// produce exactly the listed error kinds and no others, so a shared failure
// cannot stand in for a path's own check.
func TestSmokeOffline(t *testing.T) {
	good := []string{"calc/divide.go", "calc/divide_test.go"}

	t.Run("positive control: correct run has no errors", func(t *testing.T) {
		diff := buildSmokeTestDiff()
		changes, err := patch.Parse([]byte(diff))
		if err != nil {
			t.Fatalf("patch.Parse: %v", err)
		}
		var changedFiles []string
		for _, c := range changes {
			changedFiles = append(changedFiles, c.Path)
		}
		if !slices.Equal(changedFiles, good) {
			t.Errorf("scripted patch touches %v, want %v", changedFiles, good)
		}

		errs := runOfflineSmoke(t, smokeScript(diff, true, smokeReportText(good, true)))
		for _, errMsg := range errs {
			t.Errorf("smoke check failed: %s", errMsg)
		}
	})

	// Each case changes one thing. want lists the error prefixes the checker
	// must emit; every error must match one, and every prefix must be matched.
	negatives := []struct {
		name   string
		script []provider.Turn
		want   []string
	}{
		{
			name:   "report lists an extra file",
			script: smokeScript(buildSmokeTestDiff(), true, smokeReportText(append(slices.Clone(good), "extra_file.go"), true)),
			want:   []string{"files changed mismatch"},
		},
		{
			name:   "report omits calc/divide_test.go",
			script: smokeScript(buildSmokeTestDiff(), true, smokeReportText(good[:1], true)),
			want:   []string{"files changed mismatch"},
		},
		{
			name:   "patch omits the b == 0 guard",
			script: smokeScript(buildSmokeDiff(diffOpts{guard: false, zeroTest: true}), true, smokeReportText(good, true)),
			want:   []string{"behaviour probe failed", "go test ./... failed"},
		},
		{
			name:   "patch keeps the guard but adds no zero-divisor test",
			script: smokeScript(buildSmokeDiff(diffOpts{guard: true, zeroTest: false}), true, smokeReportText(good, true)),
			want:   []string{"no test exercises a zero divisor"},
		},
		{
			name:   "agent never runs go test",
			script: smokeScript(buildSmokeTestDiff(), false, smokeReportText(good, true)),
			want:   []string{"agent did not complete a go test run_command"},
		},
		{
			name:   "report has no Limitations label, only the word in prose",
			script: smokeScript(buildSmokeTestDiff(), true, smokeReportText(good, false)),
			want:   []string{"missing report label: Limitations"},
		},
		{
			name:   "patch touches an extra file while the report lists two",
			script: smokeScript(buildSmokeDiff(diffOpts{guard: true, zeroTest: true, extraFile: true}), true, smokeReportText(good, true)),
			want:   []string{"touched files mismatch"},
		},
	}
	for _, tt := range negatives {
		t.Run("negative control: "+tt.name, func(t *testing.T) {
			errs := runOfflineSmoke(t, tt.script)
			if len(errs) == 0 {
				t.Fatalf("checker accepted a run that should fail; want error kinds %v", tt.want)
			}
			matched := make(map[string]bool)
			for _, errMsg := range errs {
				known := false
				for _, w := range tt.want {
					if strings.HasPrefix(errMsg, w) {
						matched[w] = true
						known = true
					}
				}
				if !known {
					t.Errorf("unexpected error kind: %s", errMsg)
				}
			}
			for _, w := range tt.want {
				if !matched[w] {
					t.Errorf("missing expected error kind %q; got %v", w, errs)
				}
			}
		})
	}
}

// cleanSectionLines joins one section's body lines: each line is trimmed, a
// line wholly wrapped in a single pair of backticks is unwrapped, and empty
// lines are dropped.
func cleanSectionLines(content []string) string {
	var cleaned []string
	for _, l := range content {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "`") && strings.HasSuffix(l, "`") && strings.Count(l, "`") == 2 {
			l = strings.TrimPrefix(l, "`")
			l = strings.TrimSuffix(l, "`")
		}
		if l != "" {
			cleaned = append(cleaned, l)
		}
	}
	return strings.Join(cleaned, "\n")
}

// parseReportSummary extracts all labeled sections from a report.
// It handles various markdown formats: ## headers, **bold** headers, numbered headers (e.g., "1. Files changed"),
// and headers followed by colons. Header recognition strips bullets, numbering, bold, backticks and trailing
// " — note" / " - note" / ": note" markers; a line is a header only if what remains equals a known label exactly,
// so the label word inside a prose sentence is not a header.
func parseReportSummary(text string) map[string]string {
	result := make(map[string]string)

	// stripForLabel normalises a line so it can be compared with a label.
	stripForLabel := func(s string) string {
		s = reBullets.ReplaceAllString(s, "")
		s = reNumbered.ReplaceAllString(s, "")
		s = reHeaders.ReplaceAllString(s, "")
		s = reBold.ReplaceAllString(s, "$1")
		s = reBackticks.ReplaceAllString(s, "$1")
		s = reTrailDash.ReplaceAllString(s, "")
		s = reTrailMinus.ReplaceAllString(s, "")
		s = reTrailColon.ReplaceAllString(s, "")
		s = strings.TrimSuffix(s, ":")
		return strings.TrimSpace(s)
	}

	labelPatterns := []string{"Files changed", "Commands run", "Summary", "Limitations"}
	isHeaderLine := func(line string) (string, bool) {
		stripped := stripForLabel(line)
		for _, label := range labelPatterns {
			if stripped == label {
				return label, true
			}
		}
		return "", false
	}

	var currentLabel string
	var currentContent []string
	flush := func() {
		if currentLabel != "" {
			result[currentLabel] = cleanSectionLines(currentContent)
		}
	}

	for _, line := range strings.Split(text, "\n") {
		if label, isHeader := isHeaderLine(line); isHeader {
			flush()
			currentLabel = label
			currentContent = nil
		} else if currentLabel != "" {
			currentContent = append(currentContent, line)
		}
	}
	flush()

	return result
}

// parseListLines splits a section body into one item per line, stripping
// leading bullets or numbering and inline backticks, and dropping blank lines
// and lines starting with "#". It serves both the file list and the command
// list.
func parseListLines(section string) []string {
	var items []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		line = reBullets.ReplaceAllString(line, "")
		line = reNumbered.ReplaceAllString(line, "")
		line = reBackticks.ReplaceAllString(line, "$1")
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			items = append(items, line)
		}
	}
	return items
}

// TestParseListLines checks bullet, numbering and backtick stripping for list
// items, including the bullet-plus-backtick form that cleanSectionLines does
// not unwrap because the line does not start with a backtick.
func TestParseListLines(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		{"plain", "calc/divide.go\ncalc/divide_test.go", []string{"calc/divide.go", "calc/divide_test.go"}},
		{"bullets with backticks", "- `calc/divide.go`\n* `calc/divide_test.go`", []string{"calc/divide.go", "calc/divide_test.go"}},
		{"numbered with backticks", "1. `calc/divide.go`\n2. calc/divide_test.go", []string{"calc/divide.go", "calc/divide_test.go"}},
		{"blank and header lines dropped", "\n# note\ncalc/divide.go\n", []string{"calc/divide.go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseListLines(tt.in); !slices.Equal(got, tt.want) {
				t.Errorf("parseListLines(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// collectSmokeResult collects the turn outcome and analyzes the result.
// It extracts the final text, tracks approval requests and completions, and parses report sections.
func collectSmokeResult(t *testing.T, e *smokeEnv) *smokeResult {
	t.Helper()

	result := &smokeResult{
		ApprovalRequests: make(map[string]int64),
		CompletedTools:   make(map[int64]tui.ToolCompletedMsg),
		ReportLabels:     make(map[string]string),
	}

	if len(e.a.conv.Messages) == 0 {
		t.Fatal("conversation is empty after turn")
	}
	lastMsg := e.a.conv.Messages[len(e.a.conv.Messages)-1]
	if lastMsg.Role != agent.RoleAssistant {
		t.Fatalf("last message is not from assistant: %v", lastMsg.Role)
	}
	for _, block := range lastMsg.Content {
		if block.Kind == agent.BlockText {
			result.FinalText += block.Text
		}
	}
	if result.FinalText == "" {
		t.Fatal("final assistant message has no text")
	}

	for _, m := range e.msgs {
		switch msg := m.(type) {
		case tui.ApprovalRequestedMsg:
			result.ApprovalRequests[msg.Kind] = msg.ID
		case tui.ToolCompletedMsg:
			result.CompletedTools[msg.ID] = msg
		}
	}

	result.ReportLabels = parseReportSummary(result.FinalText)
	if filesSection, ok := result.ReportLabels["Files changed"]; ok {
		result.FilesChanged = parseListLines(filesSection)
	}
	if cmdSection, ok := result.ReportLabels["Commands run"]; ok {
		result.CommandsRun = parseListLines(cmdSection)
	}

	return result
}

// sameSet reports whether a and b hold the same strings, ignoring order and
// keeping multiplicity.
func sameSet(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

// touchedFiles returns the paths git reports as modified, added, deleted or
// untracked in dir, from `git status --porcelain --untracked-files=all`. A
// rename line contributes both of its paths.
func touchedFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		files = append(files, strings.Split(line[3:], " -> ")...)
	}
	return files, nil
}

var (
	rePassedTest = regexp.MustCompile(`(?m)^--- PASS: (\S+)`)
	// reZeroDivisorCall matches Divide(<anything>, 0) / Divide(x, 0.0) / Divide(x, zero).
	reZeroDivisorCall = regexp.MustCompile(`Divide\(\s*[^,]+,\s*(?:0(?:\.0+)?|zero)\s*\)`)
	// reZeroDivisorField matches a table-test field such as `b: 0,` or `b: 0}`.
	reZeroDivisorField = regexp.MustCompile(`\bb\s*:\s*0(?:\.0+)?\s*[,}]`)
)

// zeroDivisorTestExists reports whether any calc/*_test.go in dir contains a
// call or table entry that passes a zero divisor. It is a source heuristic:
// it cannot see a divisor held in an unrelated variable or an unkeyed table
// row.
func zeroDivisorTestExists(dir string) (bool, error) {
	files, err := filepath.Glob(filepath.Join(dir, "calc", "*_test.go"))
	if err != nil {
		return false, err
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return false, err
		}
		if reZeroDivisorCall.Match(src) || reZeroDivisorField.Match(src) {
			return true, nil
		}
	}
	return false, nil
}

// probeDivideZero checks the behaviour of the patched calc package. It copies
// go.mod and the non-test .go files of calc/ from dir into a scratch directory,
// adds a probe test, and runs it there. The probe asserts Divide(1, 0) returns
// a non-nil error and, as a control through the same path, that Divide(6, 3)
// still returns 2 with a nil error. A scratch copy keeps the probe out of the
// workspace, so it cannot enter the touched-files check or collide with the
// agent's own test names. The returned string is the go test output.
func probeDivideZero(t *testing.T, dir string) (string, error) {
	t.Helper()
	scratch := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scratch, "calc"), 0o755); err != nil {
		return "", err
	}
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(scratch, "go.mod"), mod, 0o644); err != nil {
		return "", err
	}
	srcs, err := filepath.Glob(filepath.Join(dir, "calc", "*.go"))
	if err != nil {
		return "", err
	}
	for _, src := range srcs {
		if strings.HasSuffix(src, "_test.go") {
			continue
		}
		b, err := os.ReadFile(src)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(scratch, "calc", filepath.Base(src)), b, 0o644); err != nil {
			return "", err
		}
	}
	probe := "package calc\n\nimport \"testing\"\n\n" +
		"func TestKirschSmokeProbe(t *testing.T) {\n" +
		"\tif _, err := Divide(1, 0); err == nil {\n\t\tt.Errorf(\"Divide(1, 0) returned a nil error\")\n\t}\n" +
		"\tif got, err := Divide(6, 3); err != nil || got != 2 {\n\t\tt.Errorf(\"Divide(6, 3) = %v, %v; want 2, nil\", got, err)\n\t}\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(scratch, "calc", "zz_probe_test.go"), []byte(probe), 0o644); err != nil {
		return "", err
	}
	cmd := exec.Command("go", "test", "-count=1", "-run", "^TestKirschSmokeProbe$", "./calc/")
	cmd.Dir = scratch
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// checkSmokeRun validates the outcome of a smoke turn against ground truth in
// e.dir and returns the failures; an empty slice means every check passed.
// Each failure kind starts with its own prefix, which the negative controls in
// TestSmokeOffline match on. The checks are:
//   - the report has all four labels;
//   - the report's "Files changed" set equals the expected set;
//   - git status in the workspace shows exactly the expected set touched;
//   - apply_patch was approval-requested and completed OK;
//   - the agent itself completed a run_command whose argv began "go test";
//   - a probe in a scratch copy shows Divide(1, 0) returns an error;
//   - `go test ./...` in the workspace exits 0 and passes TestDivide plus at
//     least one other top-level test;
//   - some calc/*_test.go passes a zero divisor.
func checkSmokeRun(t *testing.T, e *smokeEnv, result *smokeResult) []string {
	t.Helper()

	var errs []string
	expectedFiles := []string{"calc/divide.go", "calc/divide_test.go"}

	for _, label := range []string{"Summary", "Files changed", "Commands run", "Limitations"} {
		if _, ok := result.ReportLabels[label]; !ok {
			errs = append(errs, "missing report label: "+label)
		}
	}

	if !sameSet(result.FilesChanged, expectedFiles) {
		errs = append(errs, "files changed mismatch: report lists "+strings.Join(result.FilesChanged, ", ")+", want "+strings.Join(expectedFiles, ", "))
	}

	touched, err := touchedFiles(e.dir)
	if err != nil {
		errs = append(errs, "touched files check could not run git status: "+err.Error())
	} else if !sameSet(touched, expectedFiles) {
		errs = append(errs, "touched files mismatch: git status shows "+strings.Join(touched, ", ")+", want "+strings.Join(expectedFiles, ", "))
	}

	if _, ok := result.ApprovalRequests["patch"]; !ok {
		errs = append(errs, "apply_patch approval was not requested")
	}

	patchCompleted, ranGoTest := false, false
	for _, tc := range result.CompletedTools {
		if tc.Name == "apply_patch" && tc.OK {
			patchCompleted = true
		}
		// run_command's Target is its argv joined by spaces.
		argv := strings.Fields(tc.Target)
		if tc.Name == "run_command" && tc.OK && len(argv) >= 2 && argv[0] == "go" && argv[1] == "test" {
			ranGoTest = true
		}
	}
	if !patchCompleted {
		errs = append(errs, "apply_patch did not complete successfully")
	}
	if !ranGoTest {
		errs = append(errs, "agent did not complete a go test run_command")
	}

	if out, err := probeDivideZero(t, e.dir); err != nil {
		errs = append(errs, "behaviour probe failed: "+err.Error()+"; output: "+out)
	}

	cmd := exec.Command("go", "test", "-count=1", "-v", "./...")
	cmd.Dir = e.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		errs = append(errs, "go test ./... failed: "+err.Error()+"; output: "+string(out))
	} else {
		var names []string
		for _, m := range rePassedTest.FindAllStringSubmatch(string(out), -1) {
			names = append(names, m[1])
		}
		if !slices.Contains(names, "TestDivide") {
			errs = append(errs, "go test ./... did not pass the original TestDivide: "+string(out))
		}
		if len(names) < 2 {
			errs = append(errs, "go test ./... passed no test beyond TestDivide: "+string(out))
		}
	}

	if found, err := zeroDivisorTestExists(e.dir); err != nil {
		errs = append(errs, "zero-divisor test scan failed: "+err.Error())
	} else if !found {
		errs = append(errs, "no test exercises a zero divisor")
	}

	return errs
}
