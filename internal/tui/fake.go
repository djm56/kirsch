package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Milestone 0 fixture data.
//
// The literal strings come from plan/kirsch-ui-screens.md — same paths, same
// commands, same timings and token counts — so the golden files can be compared
// against the reference grids rather than against whatever the first render
// happened to produce.
//
// Nothing here reaches a filesystem, a shell or a model. It is data.

// fakeDriver scripts the streaming simulation.
type fakeDriver struct {
	target    ItemID
	remaining []string
}

// loadFixture builds the transcript described in ui-spec §8: two user messages,
// streaming assistant text, three tool cards (one truncated, one errored), two
// approval cards (apply_patch without [a], run_command with it), one error card
// and one system notice.
func (m *Model) loadFixture() {
	m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
		Lines: []string{"Fix the Divide validation"},
	}})

	m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "read_file", Target: "calc/divide.go",
		State: StateOK, Elapsed: 4 * time.Millisecond,
		Out: SanitizeLines(fakeDivideSource),
	}})

	m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "search_code", Target: `"Divide("`, Summary: "3 matches",
		State: StateOK, // no duration: screen 03 shows summary in that slot
		Out:   SanitizeLines("calc/divide.go:12\ncalc/divide_test.go:8\ncalc/divide_test.go:31"),
	}})

	// A deliberately filthy result, so the §7.1 sanitisation path is exercised
	// by the running prototype and not only by its tests: ANSI colour, a \r
	// progress bar, tabs, a NUL byte and a 5,000-column line.
	dirty := "\x1b[32mPASS\x1b[0m\n" +
		"10%\r50%\r100% done\n" +
		"\tindented by a tab\n" +
		"nul\x00byte\n" +
		strings.Repeat("m", 5000)
	m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "run_command", Target: "go test ./...", Summary: "exit 1",
		State: StateError, Elapsed: 2400 * time.Millisecond,
		Trunc: "truncated — 200KB cap",
		Out:   append(SanitizeLines(dirty), SanitizeLines(fakeTestOutput)...),
	}})

	m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
		Lines: []string{
			"I found it in calc/divide.go — the zero check runs after the division, " +
				"so the panic fires before validation can return an error.",
		},
	}})

	// apply_patch: never offers [a]. ADR 0006.
	patch := m.tr.Append(Item{Kind: KindApproval, Approval: &ApprovalCard{
		Kind:  ApprovalPatch,
		Title: "apply_patch — add input validation",
		Detail: []string{
			"files: 2 changed (calc/divide.go,",
			"       calc/divide_test.go)",
		},
		Diff: fakeDiff(), Added: 12, Removed: 4,
	}})
	m.pendingApproval = patch
	m.sel = patch

	m.tr.Append(Item{Kind: KindNotice, Notice: &NoticeCard{
		Text: "session recovered — 3 events after a torn line were discarded",
	}})

	m.status.Tokens = 14100
}

// queueCommandApproval appends the run_command approval variant — the one that
// does offer [a]. Reached by sending a message in the prototype.
func (m *Model) queueCommandApproval() {
	id := m.tr.Append(Item{Kind: KindApproval, Approval: &ApprovalCard{
		Kind:  ApprovalCommand,
		Title: "run_command — verify the fix",
		Detail: []string{
			"command: go test ./...",
			"cwd:     /Users/you/my-project",
			"timeout: 120s",
			"reason:  not on the allowlist",
		},
		GrantScope: "go test",
	}})
	m.pendingApproval = id
	m.sel = id
}

// begin starts a scripted streaming reply.
func (f *fakeDriver) begin(m *Model) {
	id := m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
		Lines: []string{""}, Streaming: true,
	}})
	f.target = id
	f.remaining = strings.Fields(
		"Looking at the call site now — the validation needs to run before the " +
			"division, and the test should cover the zero case explicitly.")
}

// advanceFake appends one word per tick, exercising the same render path real
// deltas will use in Milestone 3.
func (m Model) advanceFake() (tea.Model, tea.Cmd) {
	if len(m.fake.remaining) == 0 {
		if m.fake.target != 0 {
			if it, _, ok := m.tr.Find(m.fake.target); ok {
				it.Text.Streaming = false
			}
			m.fake.target = 0
			m.busy = Busy{}
			m.status.Tokens += 1200
			// Show the other approval variant once the reply lands, so both
			// cards are reachable from the running prototype.
			if m.pendingApproval == 0 {
				m.queueCommandApproval()
			}
			m.relayout(m.layout())
		}
		return m, nil
	}
	word := m.fake.remaining[0]
	m.fake.remaining = m.fake.remaining[1:]
	sep := " "
	if it, _, ok := m.tr.Find(m.fake.target); ok && len(it.Text.Lines) > 0 && it.Text.Lines[0] == "" {
		sep = ""
	}
	m.tr.AppendText(m.fake.target, sep+word)
	switch len(m.fake.remaining) % 3 {
	case 0:
		m.busy.Verb = "thinking"
	case 1:
		m.busy.Verb = "running go test"
	default:
		m.busy.Verb = "applying patch"
	}
	m.relayout(m.layout())
	return m, tickStream()
}

func fakeDiff() []string {
	return []string{
		"@@ -12,7 +12,15 @@ func Divide(a, b float64) (float64, error) {",
		" func Divide(a, b float64) (float64, error) {",
		"-    return a / b, nil",
		"+    if b == 0 {",
		"+        return 0, ErrDivideByZero",
		"+    }",
		"+    return a / b, nil",
		" }",
	}
}

const fakeDivideSource = `package calc

import "errors"

// ErrDivideByZero is returned when the divisor is zero.
var ErrDivideByZero = errors.New("divide by zero")

func Divide(a, b float64) (float64, error) {
	return a / b, nil
}`

// fakeTestOutput is long enough to exercise the 200-line inline cap and its
// "‹200 of 4,181 lines›" marker.
var fakeTestOutput = func() string {
	var b strings.Builder
	b.WriteString("=== RUN   TestDivide\n")
	b.WriteString("    divide_test.go:31: Divide(1, 0) = +Inf, want ErrDivideByZero\n")
	b.WriteString("--- FAIL: TestDivide (0.00s)\n")
	b.WriteString("=== RUN   TestDivide_Table\n")
	b.WriteString("--- PASS: TestDivide_Table (0.00s)\n")
	for i := 0; i < 4170; i++ {
		b.WriteString(fmt.Sprintf("    case %d: ok\n", i))
	}
	b.WriteString("FAIL    example.com/calc    0.004s")
	return b.String()
}()
