package tool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/djm56/kirsch/internal/workspace"
)

// Search limits. plan §3.
const (
	DefaultMaxResults = 50
	MaxMatchTextLen   = 200
)

// SearchCode searches file contents.
//
// Two backends: ripgrep when it is on PATH, and a pure-Go walk otherwise. They
// must agree — see the differential test. A divergence here is not a
// performance detail; it means the same question returns different answers on
// two developers' machines, and from M3 the model's behaviour diverges with it.
type SearchCode struct {
	WS *workspace.Workspace
	// ForcePureGo disables the ripgrep backend. Used by the differential test
	// to run both on one machine.
	ForcePureGo bool
}

type searchCodeInput struct {
	Query      string `json:"query"`
	Regex      bool   `json:"regex,omitempty"`
	Glob       string `json:"glob,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

// Match is one search hit.
type Match struct {
	Path string
	Line int
	Text string
}

// Name implements Tool.
func (t *SearchCode) Name() string { return "search_code" }

// Description implements Tool.
func (t *SearchCode) Description() string {
	return "Search file contents across the workspace. Literal by default; set " +
		"regex to interpret the query as a regular expression."
}

// Schema implements Tool.
func (t *SearchCode) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "query":       {"type": "string",  "description": "Text to find. Literal unless regex is true."},
    "regex":       {"type": "boolean", "description": "Interpret query as a regular expression."},
    "glob":        {"type": "string",  "description": "Limit to paths matching this glob, e.g. *.go"},
    "max_results": {"type": "integer", "description": "Maximum matches to return. Defaults to 50."}
  },
  "required": ["query"],
  "additionalProperties": false
}`)
}

// Invoke implements Tool.
func (t *SearchCode) Invoke(ctx context.Context, raw json.RawMessage) Result {
	var in searchCodeInput
	if e := DecodeInput(raw, &in); e != nil {
		return Result{OK: false, Error: e, DisplaySummary: e.Message}
	}
	if strings.TrimSpace(in.Query) == "" {
		return Fail(KindToolInputInvalid, "query is required and cannot be empty")
	}
	if in.MaxResults <= 0 {
		in.MaxResults = DefaultMaxResults
	}
	if in.Regex {
		if _, err := regexp.Compile(in.Query); err != nil {
			return Fail(KindToolInputInvalid, "query is not a valid regular expression: %v", err)
		}
	}

	matches, truncated, err := t.search(ctx, in)
	if ctx.Err() != nil {
		return Fail(KindCancelled, "cancelled while searching")
	}
	if err != nil {
		return Fail(KindInternal, "search failed: %v", err)
	}

	var b strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&b, "%s:%d: %s\n", m.Path, m.Line, m.Text)
	}
	summary := fmt.Sprintf("%q · %d matches", in.Query, len(matches))
	if truncated {
		summary += " (capped)"
	}
	if len(matches) == 0 {
		b.WriteString("no matches\n")
	}
	return OKResult(b.String(), summary, truncated)
}

// Backend reports which implementation a search would use. Surfaced by
// `kirsch doctor` in M5 and by the differential test.
func (t *SearchCode) Backend() string {
	if t.ForcePureGo || !ripgrepAvailable() {
		return "pure-go"
	}
	return "ripgrep"
}

func (t *SearchCode) search(ctx context.Context, in searchCodeInput) ([]Match, bool, error) {
	var (
		matches []Match
		err     error
	)
	if t.Backend() == "ripgrep" {
		matches, err = t.searchRipgrep(ctx, in)
		if err != nil {
			// A ripgrep failure falls back rather than failing the tool: the
			// answer matters more than which backend produced it.
			matches, err = t.searchPureGo(ctx, in)
		}
	} else {
		matches, err = t.searchPureGo(ctx, in)
	}
	if err != nil {
		return nil, false, err
	}

	// Both backends are sorted here rather than trusting either one's output
	// order, which is what makes them comparable at all.
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Path != matches[j].Path {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Line < matches[j].Line
	})

	truncated := false
	if len(matches) > in.MaxResults {
		matches = matches[:in.MaxResults]
		truncated = true
	}
	return matches, truncated, nil
}

func ripgrepAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

func (t *SearchCode) searchRipgrep(ctx context.Context, in searchCodeInput) ([]Match, error) {
	args := []string{"--json", "--no-config"}
	if !in.Regex {
		args = append(args, "--fixed-strings")
	}
	if in.Glob != "" {
		args = append(args, "--glob", in.Glob)
	}
	// Kirsch's built-in ignores are passed explicitly so ripgrep and the Go
	// walker exclude the same set; rg honours .gitignore natively.
	for _, ig := range []string{".git", ".kirsch", "node_modules", "vendor"} {
		args = append(args, "--glob", "!"+ig+"/**")
	}
	args = append(args, "--", in.Query, ".")

	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = t.WS.CanonicalRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		// Exit 1 means no matches, which is a normal answer, not a failure.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var out []Match
	sc := bufio.NewScanner(&stdout)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		var ev struct {
			Type string `json:"type"`
			Data struct {
				Path       struct{ Text string } `json:"path"`
				Lines      struct{ Text string } `json:"lines"`
				LineNumber int                   `json:"line_number"`
			} `json:"data"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Type != "match" {
			continue
		}
		rel := strings.TrimPrefix(filepathToSlash(ev.Data.Path.Text), "./")
		if t.excluded(rel) {
			continue
		}
		out = append(out, Match{
			Path: rel,
			Line: ev.Data.LineNumber,
			Text: trimMatch(ev.Data.Lines.Text),
		})
	}
	return out, sc.Err()
}

func (t *SearchCode) searchPureGo(ctx context.Context, in searchCodeInput) ([]Match, error) {
	var re *regexp.Regexp
	if in.Regex {
		var err error
		if re, err = regexp.Compile(in.Query); err != nil {
			return nil, err
		}
	}

	var out []Match
	err := t.WS.Walk(".", workspace.WalkOptions{MaxDepth: -1}, func(e workspace.Entry) bool {
		if ctx.Err() != nil {
			return false
		}
		if e.IsDir || t.excluded(e.Rel) {
			return true
		}
		if in.Glob != "" {
			if ok, _ := path.Match(in.Glob, path.Base(e.Rel)); !ok {
				return true
			}
		}
		abs, err := t.WS.Resolve(e.Rel)
		if err != nil {
			return true
		}
		// Resolved immediately above; see readfile.go. gosec G304.
		body, err := os.ReadFile(abs) // #nosec G304 -- resolved by workspace.Resolve
		if err != nil || isBinary(body) {
			return true
		}
		for i, line := range strings.Split(string(body), "\n") {
			hit := false
			if re != nil {
				hit = re.MatchString(line)
			} else {
				hit = strings.Contains(line, in.Query)
			}
			if hit {
				out = append(out, Match{Path: e.Rel, Line: i + 1, Text: trimMatch(line)})
			}
		}
		return true
	})
	if err != nil && !workspace.IsLimitReached(err) {
		return nil, err
	}
	return out, nil
}

// excluded applies the checks that must hold whichever backend ran.
func (t *SearchCode) excluded(rel string) bool {
	if t.WS.IsIgnored(rel, false) {
		return true
	}
	// The denylist is re-applied here rather than assumed: ripgrep knows about
	// .gitignore but nothing about Kirsch's rules.
	_, err := t.WS.Resolve(rel)
	return err != nil
}

func trimMatch(s string) string {
	s = strings.TrimRight(s, "\r\n")
	if len(s) > MaxMatchTextLen {
		return s[:MaxMatchTextLen] + "…"
	}
	return s
}

func filepathToSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }
