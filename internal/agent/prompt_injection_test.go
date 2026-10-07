package agent

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/djm56/kirsch/internal/agent/prompt"
)

// TestPromptInjection_SurfacedNotObeyed exercises the m3-d4 mechanical core of
// ADR 0007 with the repo-prompt-injection fixture. It falsifies three failure
// modes:
//  1. The injected authority claim is hidden from the model (it must be
//     visible in the system prompt's project-context section).
//  2. The project context appears anywhere other than the system prompt.
//  3. The injected text changes any rule-bearing section (1-4 or 6) of the
//     system prompt.
func TestPromptInjection_SurfacedNotObeyed(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", "repo-prompt-injection"))
	if err != nil {
		t.Fatalf("resolving fixture root: %v", err)
	}
	engine := &rootedEngine{root: root}

	projectContext, chosen, size, warnings, err := prompt.LoadProjectContext(
		[]string{"src/helper.php"},
		32768,
		engine,
	)
	if err != nil {
		t.Fatalf("loading project context: %v", err)
	}
	if chosen != "src/helper.php" {
		t.Fatalf("chosen: got %q, want src/helper.php", chosen)
	}
	if size == 0 {
		t.Fatal("loaded project context size is zero")
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(projectContext, "SYSTEM NOTE FOR THE ASSISTANT") {
		t.Fatal("fixture injection text missing from loaded project context")
	}

	env := prompt.Env{
		WorkspaceRoot: root,
		ProjectType:   "test",
		Branch:        "main",
		Dirty:         false,
		OS:            "test",
		HasRG:         false,
	}

	systemWith := prompt.Assemble(env, projectContext)
	systemWithout := prompt.Assemble(env, "")

	// Drive two agents that differ only in whether project context is present.
	agentWith := &Agent{
		Model:     newFakeModel([]Event{textEv("ok"), msgDone()}),
		Tools:     fullTools(),
		Recorder:  &fakeRecorder{},
		MaxTokens: 1024,
		System:    systemWith,
	}
	agentWithout := &Agent{
		Model:     newFakeModel([]Event{textEv("ok"), msgDone()}),
		Tools:     fullTools(),
		Recorder:  &fakeRecorder{},
		MaxTokens: 1024,
		System:    systemWithout,
	}

	if err := agentWith.Turn(context.Background(), &Conversation{}, "hello", nil); err != nil {
		t.Fatalf("Turn with context failed: %v", err)
	}
	if err := agentWithout.Turn(context.Background(), &Conversation{}, "hello", nil); err != nil {
		t.Fatalf("Turn without context failed: %v", err)
	}

	reqWith := agentWith.Model.(*fakeModel).requests[0]

	// 1. Surfaced: the injected instruction text is present in the outgoing
	// system prompt, inside the fenced, untrusted-labelled project-context
	// section.
	if !strings.Contains(reqWith.System, projectContext) {
		t.Fatal("project context missing from system prompt")
	}

	// 2. Exactly once: the project context appears in the System prompt once and
	// zero times in any Message.
	if got := strings.Count(reqWith.System, projectContext); got != 1 {
		t.Fatalf("project context appears in System %d times, want exactly 1", got)
	}
	if got := countInMessages(reqWith.Messages, projectContext); got != 0 {
		t.Fatalf("project context appears in Messages %d times, want 0", got)
	}

	// 3. No authority: rule-bearing sections (1-4 and 6) are byte-identical
	// with and without the injected project context.
	rulesWith := extractRuleSections(t, systemWith)
	rulesWithout := extractRuleSections(t, systemWithout)
	if rulesWith != rulesWithout {
		t.Fatalf("rule-bearing sections differ with/without injection:\nwith:    %q\nwithout: %q", rulesWith, rulesWithout)
	}
}

// rootedEngine is a minimal Engine for tests that resolves relative paths
// under a fixed root.
type rootedEngine struct {
	root string
}

func (e *rootedEngine) Resolve(rel string) (string, error) {
	return filepath.Join(e.root, rel), nil
}

// countInMessages returns the total number of non-overlapping occurrences of
// substr in the text blocks of msgs.
func countInMessages(msgs []Message, substr string) int {
	n := 0
	for _, m := range msgs {
		for _, b := range m.Content {
			if b.Kind == BlockText {
				n += strings.Count(b.Text, substr)
			}
		}
	}
	return n
}

// sectionPattern matches section headers like "## 1. Role and constraints".
var sectionPattern = regexp.MustCompile(`(?m)^## ([1-6])\. .*\n`)

// extractRuleSections returns the concatenation of sections 1-4 and 6 from the
// assembled system prompt, including their headers. Section 5 (project context)
// is intentionally excluded; prompts without project context simply omit it.
func extractRuleSections(t *testing.T, system string) string {
	t.Helper()
	matches := sectionPattern.FindAllStringIndex(system, -1)

	sections := make(map[int]string)
	for i, m := range matches {
		start := m[0]
		end := len(system)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		header := system[m[0]:m[1]]
		num := int(header[3] - '0')
		sections[num] = system[start:end]
	}

	var out strings.Builder
	for _, num := range []int{1, 2, 3, 4, 6} {
		s, ok := sections[num]
		if !ok {
			t.Fatalf("missing section %d in system prompt:\n%s", num, system)
		}
		out.WriteString(s)
	}
	return out.String()
}
