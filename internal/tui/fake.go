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

// fakeDriver scripts one turn: tool calls land, text streams, an approval
// arrives. It exists so the running prototype reaches every state the screens
// document through interaction, rather than by pre-baking a transcript nobody
// can get back to.
type fakeDriver struct {
	step      int
	target    ItemID
	toolCard  ItemID
	remaining []string
}

// loadFixture builds the transcript described in ui-spec §8: two user
// messages, streaming assistant text, three tool cards (one truncated, one
// errored), two approval cards (apply_patch without [a], run_command with it),
// one error card and one system notice.
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
		State: StateOK, // no duration: screen 03 shows the summary in that slot
		Out:   SanitizeLines("calc/divide.go:12\ncalc/divide_test.go:8\ncalc/divide_test.go:31"),
	}})
	m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "run_command", Target: "go test ./...", Summary: "exit 1",
		State: StateError, Elapsed: 2400 * time.Millisecond,
		Trunc: "truncated — 200KB cap",
		Out:   SanitizeLines(fakeDirtyOutput + "\n" + fakeTestOutput),
	}})
	m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
		Lines: []string{
			"I found it in calc/divide.go — the zero check runs after the division, " +
				"so the panic fires before validation can return an error.",
		},
	}})
	m.queuePatchApproval()
	m.tr.Append(Item{Kind: KindNotice, Notice: &NoticeCard{
		Text: "session recovered — 3 events after a torn line were discarded",
	}})
	m.status.Tokens = 14100
}

// queuePatchApproval appends the apply_patch variant, which never offers [a].
// ADR 0006.
func (m *Model) queuePatchApproval() {
	id := m.tr.Append(Item{Kind: KindApproval, Approval: &ApprovalCard{
		Kind:    ApprovalPatch,
		Title:   "apply_patch — add input validation",
		Subject: "2 files changed",
		Detail: []string{
			"files: 2 changed (calc/divide.go,",
			"       calc/divide_test.go)",
		},
		Diff: fakeDiff(), Added: 12, Removed: 4,
	}})
	m.pendingApproval = id
	m.sel = id
}

// queueCommandApproval appends the run_command variant — the one that does
// offer [a].
func (m *Model) queueCommandApproval() {
	id := m.tr.Append(Item{Kind: KindApproval, Approval: &ApprovalCard{
		Kind:    ApprovalCommand,
		Title:   "run_command — go test ./...",
		Subject: "go test ./...",
		Detail: []string{
			"cwd: .        timeout: 60s",
			"reason: not on allowlist",
		},
		GrantScope: "go test",
	}})
	m.pendingApproval = id
	m.sel = id
}

// begin starts the scripted turn.
func (f *fakeDriver) begin(m *Model) {
	*f = fakeDriver{}
	m.busy = Busy{Active: true, Verb: "thinking"}
}

// advanceFake walks one step of the scripted turn per tick, exercising the same
// render path real events will drive from Milestone 3.
func (m Model) advanceFake() (tea.Model, tea.Cmd) {
	switch m.fake.step {
	case 0: // a read lands
		m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
			Name: "read_file", Target: "calc/divide.go",
			State: StateOK, Elapsed: 4 * time.Millisecond,
			Out: SanitizeLines(fakeDivideSource),
		}})
		m.status.Tokens += 3200

	case 1: // a command starts running
		m.fake.toolCard = m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
			Name: "run_command", Target: "go test ./...", State: StateRunning,
		}})
		m.busy.Verb = "running go test"

	case 2: // and finishes, failing, with output long enough to hit the cap
		m.tr.MutateTool(m.fake.toolCard, func(c *ToolCard) {
			c.State = StateError
			c.Summary = "exit 1"
			c.Elapsed = 2400 * time.Millisecond
			c.Trunc = "truncated — 200KB cap"
			c.Out = SanitizeLines(fakeDirtyOutput + "\n" + fakeTestOutput)
		})
		m.busy.Verb = "thinking"
		m.status.Tokens += 6100

	case 3: // the assistant starts replying
		m.fake.target = m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
			Lines: []string{""}, Streaming: true,
		}})
		m.fake.remaining = strings.Fields(
			"I found it in calc/divide.go — the zero check runs after the division, " +
				"so the panic fires before validation can return an error.")

	case 4: // ...one word at a time
		if len(m.fake.remaining) > 0 {
			word := m.fake.remaining[0]
			m.fake.remaining = m.fake.remaining[1:]
			sep := " "
			if it, _, ok := m.tr.Find(m.fake.target); ok && len(it.Text.Lines) > 0 && it.Text.Lines[0] == "" {
				sep = ""
			}
			m.tr.AppendText(m.fake.target, sep+word)
			m.relayout(m.layout())
			return m, tickStream() // stay on this step until the text runs out
		}
		if it, _, ok := m.tr.Find(m.fake.target); ok {
			it.Text.Streaming = false
		}
		m.busy.Verb = "applying patch"
		m.status.Tokens += 1800

	case 5: // an approval blocks the turn
		m.queuePatchApproval()
		m.busy = Busy{}
		m.relayout(m.layout())
		return m, nil

	default:
		m.busy = Busy{}
		return m, nil
	}

	m.fake.step++
	m.relayout(m.layout())
	return m, tickStream()
}

// fakeDirtyOutput exercises the §7.1 sanitisation path from the running
// prototype and not only from its tests: ANSI colour, a \r progress bar, tabs,
// a NUL byte and a 5,000-column line.
var fakeDirtyOutput = "\x1b[32mPASS\x1b[0m\n" +
	"10%\r50%\r100% done\n" +
	"\tindented by a tab\n" +
	"nul\x00byte\n" +
	strings.Repeat("m", 5000)

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
		fmt.Fprintf(&b, "    case %d: ok\n", i)
	}
	b.WriteString("FAIL    example.com/calc    0.004s")
	return b.String()
}()
