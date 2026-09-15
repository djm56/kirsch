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

// loadFixture builds the static transcript NewWithFixture starts from: one
// user message; three tool cards — read_file, search_code, and a run_command
// that failed and carries an upstream truncation note; a finished, no longer
// streaming assistant message; the apply_patch approval, which never offers
// [a]; and one system notice.
//
// A subset of ui-spec §8, deliberately — and the list above is read off the
// body below, not copied from the spec. §8 describes the whole Milestone 0
// surface; the states this function omits are reached elsewhere. A running
// card, its terminal transition and streaming text come from advanceFake's
// scripted turn; the run_command approval from queueCommandApproval; the
// error card from an ErrorMsg, or from a golden that builds its own. The
// fixture moved behind NewWithFixture and shrank when it did — plan
// amendment 33.
func (m *Model) loadFixture() {
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{
		Lines: []string{"Fix the Divide validation"},
	}})
	m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "read_file", Target: "calc/divide.go",
		State: StateOK, Elapsed: 4 * time.Millisecond,
		Out: SanitizeLines(fakeDivideSource),
	}})
	m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "search_code", Target: `"Divide("`, Summary: "3 matches",
		State: StateOK, // no duration: screen 03 shows the summary in that slot
		Out:   SanitizeLines("calc/divide.go:12\ncalc/divide_test.go:8\ncalc/divide_test.go:31"),
	}})
	m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
		Name: "run_command", Target: "go test ./...", Summary: "exit 1",
		State: StateError, Elapsed: 2400 * time.Millisecond,
		Trunc: "truncated — 200KB cap",
		Out:   SanitizeLines(fakeDirtyOutput + "\n" + fakeTestOutput),
	}})
	m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{
		Lines: []string{
			"I found it in calc/divide.go — the zero check runs after the division, " +
				"so the panic fires before validation can return an error.",
		},
	}})
	m.queuePatchApproval()
	m.appendBlock(Item{Kind: KindNotice, Notice: &NoticeCard{
		Text: "session recovered — 3 events after a torn line were discarded",
	}})
	m.status.Tokens = 14100
}

// queuePatchApproval appends the apply_patch variant, which never offers [a].
// ADR 0006.
func (m *Model) queuePatchApproval() {
	id := m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{
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
	id := m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{
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

// Fixture pacing. Neither constant is product behaviour: a real tool card is
// moved by ToolStartedMsg and ToolCompletedMsg, whose timing belongs to the
// tool, and nothing outside this file reads either value.
const (
	// fakeRunDwell is how long the scripted run_command card stays in
	// StateRunning, and — because case 2 reports it as the card's Elapsed — how
	// long that card then claims to have taken. The scripted turn used to
	// advance on StreamCoalesce alone, so the running state lasted a single
	// 50ms frame: the ◐ glyph, the running gutter and the "running go test"
	// verb all existed, were rendered correctly, and were gone before a reader
	// could see any of them.
	fakeRunDwell = 2400 * time.Millisecond

	// fakeApprovalPoll is how often the scripted turn looks to see whether the
	// operator has answered the approval in front of it. A tick is the only
	// thing that re-enters advanceFake, and resolving an approval is a key
	// press that schedules none — so the chain waits by ticking rather than by
	// being restarted from the key handler, which would mean reaching into the
	// product path to serve a fixture. The interval is coarse because the thing
	// it waits on is a human: it costs four idle frames a second instead of
	// twenty, and no repaint, since the poll returns the model untouched.
	fakeApprovalPoll = 250 * time.Millisecond
)

// tickFakeIn schedules the scripted turn's next step d from now.
//
// The scripted turn is the only caller that varies its interval. tickStream's
// fixed StreamCoalesce is what submit starts the chain with and what real
// streamed text will coalesce on, and that is a product decision the fixture
// must not reach into by making it a parameter.
func tickFakeIn(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return streamTickMsg{} })
}

// advanceFake walks one step of the scripted turn per tick, exercising the same
// render path real events will drive from Milestone 3.
//
// The turn ends on two approvals rather than one: the patch, which never offers
// [a], and then a run_command, which does. ADR 0006 makes those two the whole of
// the approval surface, and screen 04 pins exactly the state the second one
// produces — an approved patch above a live run_command card with a grant on
// offer. Until this existed, the running app only ever built the patch variant,
// so [a] rendered in the golden tests and nowhere a person could reach.
//
// The second approval is why the poll below exists. Model.pendingApproval holds
// one ID, so queueing the second while the first is unanswered would silently
// replace it — the card the operator is reading would stop being the card the
// keys resolve.
func (m Model) advanceFake() (tea.Model, tea.Cmd) {
	// A scripted approval blocks the turn exactly as a real one blocks a real
	// turn: the chain keeps ticking, the step counter does not move, and the
	// model is returned untouched until the operator has answered.
	if m.pendingApproval != 0 {
		return m, tickFakeIn(fakeApprovalPoll)
	}

	next := StreamCoalesce
	switch m.fake.step {
	case 0: // a read lands
		m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
			Name: "read_file", Target: "calc/divide.go",
			State: StateOK, Elapsed: 4 * time.Millisecond,
			Out: SanitizeLines(fakeDivideSource),
		}})
		m.status.Tokens += 3200

	case 1: // a command starts running, and stays visibly running
		m.fake.toolCard = m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
			Name: "run_command", Target: "go test ./...", State: StateRunning,
		}})
		m.busy.Verb = "running go test"
		next = fakeRunDwell

	case 2: // and finishes, failing, with output long enough to hit the cap
		m.tr.MutateTool(m.fake.toolCard, func(c *ToolCard) {
			c.State = StateError
			c.Summary = "exit 1"
			c.Elapsed = fakeRunDwell
			c.Trunc = "truncated — 200KB cap"
			c.Out = SanitizeLines(fakeDirtyOutput + "\n" + fakeTestOutput)
		})
		m.busy.Verb = "thinking"
		m.status.Tokens += 6100

	case 3: // the assistant starts replying
		m.fake.target = m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{
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
			return m, tickFakeIn(StreamCoalesce) // stay on this step until the text runs out
		}
		if it, _, ok := m.tr.Find(m.fake.target); ok {
			it.Text.Streaming = false
		}
		m.busy.Verb = "applying patch"
		m.status.Tokens += 1800

	case 5: // the patch approval blocks the turn
		m.queuePatchApproval()
		m.busy = Busy{}
		next = fakeApprovalPoll

	case 6: // and the command it wants to run next needs its own
		m.queueCommandApproval()
		m.busy = Busy{}
		m.relayout(m.layout())
		return m, nil // the last beat: nothing is waiting behind this card

	default:
		m.busy = Busy{}
		return m, nil
	}

	m.fake.step++
	m.relayout(m.layout())
	return m, tickFakeIn(next)
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

// fakeTestOutput is long enough to exercise the 200-line inline cap and the
// "‹200 of N lines›" marker beneath it. On its own it sanitises to 4,176
// lines, which is what screen 07's grid pins; loadFixture's run_command card
// prepends fakeDirtyOutput's five lines and so renders 4,181.
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
