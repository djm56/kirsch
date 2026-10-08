package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeDriver scripts one turn: tool calls land, text streams, an approval
// arrives. It is a test fixture only — production submit() now hands text to
// the app via the Submit callback.
type fakeDriver struct {
	step      int
	target    ItemID
	toolCard  ItemID
	remaining []string
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
	// 50ms frame.
	fakeRunDwell = 2400 * time.Millisecond

	// fakeApprovalPoll is how often the scripted turn looks to see whether the
	// operator has answered the approval in front of it.
	fakeApprovalPoll = 250 * time.Millisecond
)

// tickFakeIn schedules the scripted turn's next step d from now.
func tickFakeIn(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return struct{}{} })
}

// advance walks one step of the scripted turn, exercising the same render path
// real events drive from Milestone 3.
func (f *fakeDriver) advance(m Model) (tea.Model, tea.Cmd) {
	if m.pendingApproval != 0 {
		return m, tickFakeIn(fakeApprovalPoll)
	}

	next := StreamCoalesce
	switch f.step {
	case 0: // a read lands
		m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
			Name: "read_file", Target: "calc/divide.go",
			State: StateOK, Elapsed: 4 * time.Millisecond,
			Out: SanitizeLines(fakeDivideSource),
		}})
		m.status.Tokens += 3200

	case 1: // a command starts running, and stays visibly running
		f.toolCard = m.appendBlock(Item{Kind: KindTool, Tool: &ToolCard{
			Name: "run_command", Target: "go test ./...", State: StateRunning,
		}})
		m.busy.Verb = "running go test"
		next = fakeRunDwell

	case 2: // and finishes, failing, with output long enough to hit the cap
		m.tr.MutateTool(f.toolCard, func(c *ToolCard) {
			c.State = StateError
			c.Summary = "exit 1"
			c.Elapsed = fakeRunDwell
			c.Trunc = "truncated — 200KB cap"
			c.Out = SanitizeLines(fakeDirtyOutput + "\n" + fakeTestOutput)
		})
		m.busy.Verb = "thinking"
		m.status.Tokens += 6100

	case 3: // the assistant starts replying
		f.target = m.appendBlock(Item{Kind: KindAssistant, Text: &TextBlock{
			Lines: []string{""}, Streaming: true,
		}})
		f.remaining = strings.Fields(
			"I found it in calc/divide.go — the zero check runs after the division, " +
				"so the panic fires before validation can return an error.")

	case 4: // ...one word at a time
		if len(f.remaining) > 0 {
			word := f.remaining[0]
			f.remaining = f.remaining[1:]
			sep := " "
			if it, _, ok := m.tr.Find(f.target); ok && len(it.Text.Lines) > 0 && it.Text.Lines[0] == "" {
				sep = ""
			}
			m.tr.AppendText(f.target, sep+word)
			m.relayout(m.layout())
			return m, tickFakeIn(StreamCoalesce)
		}
		if it, _, ok := m.tr.Find(f.target); ok {
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
		return m, nil

	default:
		m.busy = Busy{}
		return m, nil
	}

	f.step++
	m.relayout(m.layout())
	return m, tickFakeIn(next)
}
