package tui

import (
	"fmt"
	"strings"
	"time"
)

// Milestone 0 fixture data.
//
// The literal strings come from plan/spec/kirsch-ui-screens.md — same paths, same
// commands, same timings and token counts — so the golden files can be compared
// against the reference grids rather than against whatever the first render
// happened to produce.
//
// Nothing here reaches a filesystem, a shell or a model. It is data.

// loadFixture builds the static transcript NewWithFixture starts from: one
// user message; three tool cards — read_file, search_code, and a run_command
// that failed and carries an upstream truncation note; a finished, no longer
// streaming assistant message; the apply_patch approval, which never offers
// [a]; and one system notice.
//
// A subset of ui-spec §8, deliberately — and the list above is read off the
// body below, not copied from the spec. §8 describes the whole Milestone 0
// surface; the states this function omits are reached elsewhere. The
// run_command approval comes from queueCommandApproval; the error card from an
// ErrorMsg, or from a golden that builds its own. The fixture moved behind
// NewWithFixture and shrank when it did — plan amendment 33.
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
		DiffFilename: "calc/divide.go",
		Diff:         fakeDiff(), Added: 12, Removed: 4,
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

// fakeTestOutput is long enough to exercise the 10-line preview cap and the
// "‹10 of N lines›" marker beneath it. On its own it sanitises to 4,176
// lines. When used in screen 07's golden test scenario, those 4,176 lines are
// rendered in preview mode, showing the first 10 lines with a marker.
// loadFixture prepends fakeDirtyOutput's five lines for other uses.
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
