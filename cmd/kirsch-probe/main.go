// Command kirsch-probe probes the OpenCode Go endpoint to gather facts about
// models, streaming, thinking blocks, caching and error handling.
//
// The probe is opt-in: the operator runs it in their own shell with their own
// API key. It never runs without the -run flag, which makes this a dry-run tool
// by default — showing the plan and request count.
//
// SECURITY:
//   - The API key comes from the environment only: KIRSCH_OPENCODE_API_KEY
//     first, then OPENCODE_API_KEY. Neither variable may be passed as a flag
//     or read from a file.
//   - The key is redacted from stdout and from every returned error, and the
//     probe refuses to write any file whose contents include it.
//   - Redirects are never followed. A 3xx response stops the probe immediately.
//   - base_url defaults to https://opencode.ai/zen/go/v1. A -base-url flag
//     exists for tests, accepting only https:// URLs or literal loopback hosts.
//   - The probe stops immediately on any 3xx, 429, 401, or 403 response, and on
//     a 400 whose error type is MissingSessionID or contains "Session" (a
//     gateway policy error). Any other 400 does not stop the run.
//   - Every request sends User-Agent: kirsch-probe/0.1 and an x-opencode-session
//     ID, as the gateway requires. IDs are random per run, one per conversation.
//   - The key travels in exactly one auth header, chosen by -auth: x-api-key
//     (the default) or Authorization: Bearer. It is applied to every request.
//   - Only fixed literal prompts are sent; nothing is read from disk.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := run(context.Background(), parseFlags(), os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "kirsch-probe: %v\n", err)
		os.Exit(1)
	}
}

type options struct {
	run     bool
	baseURL string
	models  []string
	outDir  string
	auth    string // authXAPIKey or authBearer; empty means authXAPIKey
	key     string
	pause   time.Duration
	getenv  func(string) string // for testing
}

// Auth modes accepted by -auth.
const (
	authXAPIKey = "x-api-key"
	authBearer  = "bearer"
)

// resolveAuth returns the effective auth mode: the empty value means the
// default, authXAPIKey. Any value other than the two modes is an error naming
// the allowed values.
func resolveAuth(auth string) (string, error) {
	switch auth {
	case "", authXAPIKey:
		return authXAPIKey, nil
	case authBearer:
		return authBearer, nil
	}
	return "", fmt.Errorf("invalid -auth %q: allowed values are %q and %q", auth, authXAPIKey, authBearer)
}

// setAuth puts the key on req in the chosen mode. Exactly one auth header is
// set; the other is removed so that none can be left over.
func setAuth(req *http.Request, mode, key string) {
	req.Header.Del("x-api-key")
	req.Header.Del("Authorization")
	if key == "" {
		return
	}
	if mode == authBearer {
		req.Header.Set("Authorization", "Bearer "+key)
		return
	}
	req.Header.Set("x-api-key", key)
}

// defaultOutDir is the default output directory for a run started at now: one
// directory per run, so a re-run never mixes with an earlier run's files.
func defaultOutDir(now time.Time) string {
	return filepath.Join("testdata", "probe", "opencode", now.UTC().Format("2006-01-02T1504Z"))
}

// checkOutDirFree refuses an output directory that already holds files. A
// directory that does not exist, or that cannot be inspected because a parent
// is not a directory, is left for WriteResults to create or to report.
func checkOutDirFree(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		return fmt.Errorf("output path %s exists and is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading output directory %s: %w", dir, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("output directory %s already exists and is not empty; choose another with -out or remove it", dir)
	}
	return nil
}

// defaultModels is the default value of the -models flag.
const defaultModels = "minimax-m3,minimax-m2.7,qwen3.8-max,qwen3.8-flash,qwen3.7-plus"

// splitModels turns a comma-separated list into trimmed, non-empty names.
func splitModels(list string) []string {
	var models []string
	for _, m := range strings.Split(list, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			models = append(models, m)
		}
	}
	return models
}

func parseFlags() options {
	var (
		run        = flag.Bool("run", false, "execute requests (default: dry run)")
		baseURL    = flag.String("base-url", "https://opencode.ai/zen/go/v1", "base URL for the endpoint")
		modelsFlag = flag.String("models", defaultModels, "comma-separated list of models to probe")
		outDir     = flag.String("out", "", "output directory (default: testdata/probe/opencode/<YYYY-MM-DDTHHMMZ> UTC)")
		auth       = flag.String("auth", authXAPIKey, "auth header for the key: x-api-key or bearer")
	)
	flag.Parse()

	opts := options{
		run:     *run,
		baseURL: *baseURL,
		models:  splitModels(*modelsFlag),
		outDir:  *outDir,
		auth:    *auth,
		pause:   1 * time.Second,
		getenv:  os.Getenv,
	}

	opts.key = lookupKey(opts.getenv)

	return opts
}

// lookupKey retrieves the API key from the environment.
// KIRSCH_OPENCODE_API_KEY takes precedence over OPENCODE_API_KEY.
func lookupKey(getenv func(string) string) string {
	if key := getenv("KIRSCH_OPENCODE_API_KEY"); key != "" {
		return key
	}
	return getenv("OPENCODE_API_KEY")
}

// validateModelName checks that a model name is safe to use as a path segment (F11).
func validateModelName(model string) error {
	if model == "" {
		return fmt.Errorf("empty model name")
	}
	// "_global" is the output directory for S0; a model of that name would
	// collide with it. Compared case-insensitively, as macOS and Windows
	// filesystems are.
	if strings.EqualFold(model, "_global") {
		return fmt.Errorf("invalid model name: %s (reserved for the S0 output directory)", model)
	}
	// Only allow alphanumeric, dots, hyphens, underscores.
	for _, ch := range model {
		isValid := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '_'
		if !isValid {
			return fmt.Errorf("invalid model name: %s (contains invalid character %c)", model, ch)
		}
	}
	return nil
}

// redactWriter redacts the API key from everything written through it. It is
// the backstop behind the per-site redaction in the prober: a print that
// forgets to redact still cannot put the key on the terminal.
type redactWriter struct {
	w   io.Writer
	key string
}

// Write redacts the key from p and writes the result. Each call is one whole
// print, so a key is never split across calls. It reports len(p) on success.
func (r redactWriter) Write(p []byte) (int, error) {
	if r.key == "" {
		return r.w.Write(p)
	}
	if _, err := r.w.Write(bytes.ReplaceAll(p, []byte(r.key), []byte("<redacted>"))); err != nil {
		return 0, err
	}
	return len(p), nil
}

// redactError returns err with the key redacted from its message. An error
// that does not mention the key is returned as is, so wrapping is preserved.
func redactError(err error, key string) error {
	if err == nil || key == "" || !strings.Contains(err.Error(), key) {
		return err
	}
	return errors.New(redactKey(err.Error(), key))
}

// run executes the probe. Whatever it prints to stdout or returns as an error
// has the API key redacted from it; main prints the returned error to stderr.
func run(ctx context.Context, opts options, stdout io.Writer) error {
	return redactError(runProbe(ctx, opts, redactWriter{w: stdout, key: opts.key}), opts.key)
}

// runProbe is run without the redaction backstop.
func runProbe(ctx context.Context, opts options, stdout io.Writer) error {
	// Validate the auth mode before anything else, so a typo costs nothing.
	auth, err := resolveAuth(opts.auth)
	if err != nil {
		return err
	}
	opts.auth = auth

	// Validate key presence for -run mode.
	if opts.run && opts.key == "" {
		return fmt.Errorf("no API key set; set one of: KIRSCH_OPENCODE_API_KEY, OPENCODE_API_KEY")
	}

	// Validate base_url.
	if err := validateBaseURL(opts.baseURL); err != nil {
		return err
	}

	// Set default output directory.
	if opts.outDir == "" {
		opts.outDir = defaultOutDir(time.Now())
	}
	// A live run writes into a fresh directory; refuse before any request.
	if opts.run {
		if err := checkOutDirFree(opts.outDir); err != nil {
			return err
		}
	}

	// Validate model names (F11) and refuse duplicates: a repeated model would
	// overwrite its own results and double the request count.
	// Names are compared case-insensitively: output directories collide on
	// case-insensitive filesystems.
	seen := make(map[string]bool)
	for _, model := range opts.models {
		if err := validateModelName(model); err != nil {
			return err
		}
		folded := strings.ToLower(model)
		if seen[folded] {
			return fmt.Errorf("duplicate model in -models: %s (names are compared case-insensitively)", model)
		}
		seen[folded] = true
	}

	// Create prober and plan scenarios.
	prober := NewProber(opts)
	scenarios := prober.Plan()
	totalRequests := len(scenarios)

	// Dry run: print plan and exit (F10 - show index, model, scenario, method, path).
	if !opts.run {
		if _, err := fmt.Fprintf(stdout, "Planned %d requests (auth: %s):\n", totalRequests, opts.auth); err != nil {
			return err
		}
		for i, s := range scenarios {
			model := s.Model
			if model == "" {
				model = "_global"
			}
			if _, err := fmt.Fprintf(stdout, "  %2d. %-15s %-20s %s %s\n", i+1, model, s.Scenario, s.Method, s.Path); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(stdout, "Total: %d requests\n", totalRequests); err != nil {
			return err
		}
		return nil
	}

	// Live run: execute scenarios and capture error.
	runErr := prober.Run(ctx, stdout)

	// Write results to disk, including on error (F1).
	results := prober.Results()
	if writeErr := WriteResults(opts.outDir, opts.models, results, scenarios, opts.key, opts.auth); writeErr != nil {
		return errors.Join(runErr, fmt.Errorf("writing results: %w", writeErr))
	}

	// Print the summary table to stdout (N4). It is rendered from the in-memory
	// results, the same source WriteResults used for SUMMARY.md, so the file
	// never has to be read back.
	if _, err := fmt.Fprint(stdout, generateSummaryMarkdown(opts.models, results, opts.auth)); err != nil {
		return errors.Join(runErr, fmt.Errorf("printing summary: %w", err))
	}

	// Return the run error if any (results already written).
	return runErr
}
