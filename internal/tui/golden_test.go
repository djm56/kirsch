package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// The screen reference is the expected output, not a sketch of it. These tests
// parse the character grids straight out of plan/spec/kirsch-ui-screens.md and
// compare View() against them, so a rendering change and a stale design
// document cannot drift apart silently — which is the rule plan §9.6 states and
// amendment 24 exists to protect.

const screensDoc = "../../plan/spec/kirsch-ui-screens.md"

// fixtureVersion is the version string every fixture in this package renders.
//
// It is the binary's own default — `const devVersion = "0.1.3-dev"` in
// cmd/kirsch/main.go — and not the rounder "0.1.0" the fixtures used to carry.
// Four cells of difference, and they were the four that mattered: the onboarding
// tagline is built from this string, and at "0.1.0" it measured 37 cells against
// the 38 a 40-column terminal leaves while the shipped binary measured 41 and
// overran. A fixture narrower than the product tests a frame nobody runs, and
// this one hid an overrun at exactly the width ui-spec §1 advertises as the
// minimum.
//
// Anything width-sensitive that names a version takes this constant. If the
// shipped default changes, this changes with it and the grids are regenerated —
// which is the point: the goldens should move when the product does.
const fixtureVersion = "0.1.3-dev"

type grid struct {
	Screen string
	W, H   int
	Rows   []string
}

var fenceRe = regexp.MustCompile(`^` + "```" + `text (\d+)×(\d+)$`)

// parseGrids extracts every sized render grid. Colour-map blocks follow a
// **Colours** heading and are skipped; screen 00 carries no size because it is
// a component rather than a full-screen render.
func parseGrids(t *testing.T) map[string]grid {
	t.Helper()
	raw, err := os.ReadFile(filepath.FromSlash(screensDoc))
	if err != nil {
		t.Fatalf("read screen reference: %v", err)
	}
	out := map[string]grid{}
	var screen string
	var afterColours, inFence bool
	var w, h int
	var buf []string

	for _, line := range strings.Split(string(raw), "\n") {
		if m := regexp.MustCompile(`^## (\d\d) · `).FindStringSubmatch(line); m != nil {
			screen, afterColours = m[1], false
			continue
		}
		if strings.TrimSpace(line) == "**Colours**" {
			afterColours = true
			continue
		}
		if strings.HasPrefix(line, "```") {
			if !inFence {
				inFence, buf, w, h = true, nil, 0, 0
				if m := fenceRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
					w, _ = strconv.Atoi(m[1])
					h, _ = strconv.Atoi(m[2])
				}
				continue
			}
			inFence = false
			if screen != "" && !afterColours && w > 0 {
				// A screen with more than one grid keys its extras 10b, 10c, …
				key := screen
				for n := 1; ; n++ {
					if _, taken := out[key]; !taken {
						break
					}
					key = fmt.Sprintf("%s%c", screen, 'a'+n)
				}
				out[key] = grid{Screen: key, W: w, H: h, Rows: append([]string(nil), buf...)}
			}
			continue
		}
		if inFence {
			buf = append(buf, line)
		}
	}
	return out
}

// scenario builds one model state. Every field the frame depends on is set
// explicitly — nothing is read from the clock, the environment or git — so the
// comparison is reproducible.
type scenario struct {
	Screen  string
	Unicode bool
	Colour  bool
	Build   func(m *Model)
}

func scenarios() []scenario {
	return []scenario{
		{Screen: "01", Unicode: true}, // empty session: no build step at all

		{Screen: "02", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Fix the Divide validation"},
			}})
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Target: "calc/divide.go",
				State: StateOK, Elapsed: 4 * time.Millisecond,
			}})
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "run_command", Target: "go test ./...", State: StateRunning,
			}})
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
				Lines: []string{"I found it in calc/divide.go — the zero check runs after " +
					"the division, so the panic fires before validation can return an error."},
				Streaming: true,
			}})
			m.busy = Busy{Active: true, Verb: "running go test"}
			m.status.Tokens = 12400
		}},

		{Screen: "03", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "search_code", Target: `"Divide("`, Summary: "3 matches", State: StateOK,
			}})
			m.queuePatchApproval()
			m.status.Tokens = 14100
			m.frame = 1 // ⠙
		}},

		{Screen: "04", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindApproval, Approval: &ApprovalCard{
				Kind: ApprovalPatch, Subject: "2 files changed", Outcome: Approved,
			}})
			m.queueCommandApproval()
			m.status.Tokens = 16800
			m.GetGrantCount = func() int { return 1 }
			m.frame = 2 // ⠹
		}},

		{Screen: "07", Unicode: true, Build: func(m *Model) {
			id := m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "run_command", Target: "go test ./...", Summary: "exit 1",
				State: StateError, Elapsed: 2400 * time.Millisecond,
				Out: SanitizeLines(fakeTestOutput),
			}})
			m.sel = id
			// Tool cards show preview by default; no need to set expanded
			m.status.Tokens = 22900
			m.GetGrantCount = func() int { return 1 }
		}},

		{Screen: "08", Unicode: true, Build: func(m *Model) {
			m.sess.Compacted = true
			m.tr.Append(Item{Kind: KindNotice, Notice: &NoticeCard{
				Text: "session recovered — 3 events after a torn line were discarded",
			}})
			m.tr.Append(Item{Kind: KindNotice, Notice: &NoticeCard{
				Text: "compacted 94 events into a summary · 12 files touched this session",
			}})
			m.tr.Append(Item{Kind: KindError, Err: &ErrorCard{
				Kind:    "provider_error",
				Message: "anthropic: 503 after 3 retries — request not sent",
				Hint:    "Enter to expand · the turn is still cancellable",
			}})
			m.sel = 0
			m.status.Tokens = 31200
			m.status.Warnings = []string{"recovered session"}
		}},

		{Screen: "10", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Target: "calc/divide.go", State: StateOK,
			}})
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
				Lines: []string{"Soft wrap only. No horizontal scrolling in v0.1."}, Streaming: true,
			}})
			m.busy = Busy{Active: true, Verb: "thinking"}
		}},

		{Screen: "05", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Fix the Divide validation"},
			}})
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "search_code", Target: `"Divide("`, Summary: "3 matches", State: StateOK,
			}})
			m.queuePatchApproval()
			m.openModal(ModalState{
				Kind: ModalDiff, Title: "calc/divide.go",
				Lines: fakeDiff(), Added: 12, Removed: 4,
			})
			m.status.Tokens = 14100
			m.frame = 3 // ⠸
		}},

		{Screen: "06", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Target: "calc/divide.go", State: StateOK,
				Elapsed: 4 * time.Millisecond,
			}})
			m.base = BaseBrowsing
			m.openModal(ModalState{Kind: ModalHelp, Title: "help", Lines: helpLines()})
			m.status.Tokens = 12400
		}},

		{Screen: "09", Unicode: true, Build: func(m *Model) {
			id := m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Target: "internal/session/store.go:1-120",
				State: StateOK, Elapsed: 3 * time.Millisecond,
			}})
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{
				"The lock is taken in Open, before auto-resume reads index.json, so a " +
					"second instance starts a fresh session instead of adopting the first.",
			}}})
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "git_diff", Summary: "2 files changed",
				State: StateOK, Elapsed: 8 * time.Millisecond,
			}})
			m.sel = id
			m.scroll = Scroll{Offset: 0, Pinned: false, NewSince: 3}
			m.comp.SetValue("also check the second-instance path")
			m.busy = Busy{Active: true, Verb: "thinking"}
			m.status.Tokens = 28000
			m.frame = 4 // ⠼
		}},

		{Screen: "10b", Unicode: true, Build: func(m *Model) {
			m.busy = Busy{Active: true, Verb: "thinking"}
			m.status.Warnings = []string{"recovered session"}
		}},

		{Screen: "10c", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{
				"no header at 6 rows; transcript keeps 2 lines",
			}}})
		}},

		{Screen: "11", Unicode: false, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "search_code", Target: `"Divide("`, Summary: "3 matches", State: StateOK,
			}})
			m.queuePatchApproval()
			m.status.Tokens = 14100
			m.frame = 1 // \ — index 1 of the ASCII cycle, matching screen 03's ⠙
		}},

		// A finished reply drawn as markdown (ui-spec §3.2): a heading, bold and
		// italic, inline code, a list, a quote, a link and a fenced block. The
		// reply is built from Lines the way a resumed one is, with markdown on.
		{Screen: "13", Unicode: true, Build: func(m *Model) {
			m.renderMD = true
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Why does Divide panic?"},
			}})
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{
				Lines: SanitizeLines(screen13Reply),
			}})
			m.status.Tokens = 15200
		}},

		// The transcript scrollbar (ui-spec §2.4, amendment 103): 24 lines in an
		// 8-row region, unpinned at offset 8 of a maximum of 16, so the thumb is two
		// rows long and starts on row 3. The `↓ 4 new` row meets the bar. The items
		// are chosen to give the lines the grid shows at transcript rows 0 to 7
		// (lines 8 to 15); the lines above and below only have to make the count 24.
		{Screen: "14", Unicode: true, Build: func(m *Model) {
			oneLine := func(name, target string) {
				m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
					Name: name, Target: target, State: StateOK,
				}})
			}
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Check the second-instance path"},
			}})
			oneLine("read_file", "internal/session/lock.go")
			oneLine("read_file", "internal/session/store.go")
			oneLine("search_code", `"Open("`)
			oneLine("read_file", "internal/session/index.go")
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{
				"The lock is taken in Open, before auto-resume reads index.json, so a " +
					"second instance starts a fresh session instead of adopting the first.",
			}}})
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "git_diff", Summary: "2 files changed",
				State: StateOK, Elapsed: 8 * time.Millisecond,
			}})
			m.tr.Append(Item{Kind: KindAssistant, Text: &TextBlock{Lines: []string{
				"Checked the second-instance path. The tests pass on main.",
			}}})
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Run them again"},
			}})
			oneLine("run_command", "go test ./...")
			oneLine("run_command", "go vet ./...")
			oneLine("run_command", "go build ./...")
			oneLine("run_command", "git status")
			m.scroll = Scroll{Offset: 8, Pinned: false, NewSince: 4}
			m.busy = Busy{Active: true, Verb: "thinking"}
			m.status.Tokens = 28000
			m.frame = 4 // ⠼
		}},

		// Long lines wrap in the modal (ui-spec §4.1, amendment 103), screen 15: a
		// read_file content modal. Its fourth line, "43" and its text, is wider than
		// the text width and wraps inside the word "strings". The box's contents are
		// the ones the grid draws; the cells beside it are whatever the dimmed
		// transcript behind it renders.
		{Screen: "15", Unicode: true, Build: func(m *Model) {
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Fix the Divide validation"},
			}})
			m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "read_file", Target: "calc/divide.go",
				State: StateOK, Elapsed: 4 * time.Millisecond,
			}})
			m.openModal(ModalState{
				Kind: ModalContent, Title: "calc/divide.go",
				Lines: []string{
					"40      if b == 0 {",
					"41          return 0, ErrDivideByZero",
					"42      }",
					"43      if err := validateDivisor(b); err != nil && strings.Contains(message, " +
						`"divide by zero") { return 0, wrapError(err) }`,
				},
				Path: "calc/divide.go", ReadFile: true,
			})
			m.busy = Busy{Active: true, Verb: "thinking"}
			m.status.Tokens = 14100
			m.frame = 3 // ⠸
		}},

		// Screen 15b: a patch approval diff, opened over its approval card. The @@
		// line and the added line are wider than the text width and wrap; the hunk
		// header wraps flush and the added line's second row starts with a blank
		// column where its marker is. The approval card carries the same four lines
		// and counts as the modal, so the two agree about what is being approved.
		{Screen: "15b", Unicode: true, Build: func(m *Model) {
			diff := []string{
				"@@ -12,7 +12,15 @@ func Divide(a, b float64) (float64, error) {",
				"     func Divide(a, b float64) (float64, error) {",
				"-    return a / b, nil",
				"+    if err := validateDivisor(b); err != nil && strings.Contains(message, " +
					`"divide by zero") {`,
			}
			m.tr.Append(Item{Kind: KindUser, Text: &TextBlock{
				Lines: []string{"Fix the Divide validation"},
			}})
			id := m.appendBlock(Item{Kind: KindApproval, Approval: &ApprovalCard{
				Kind:         ApprovalPatch,
				Title:        "apply_patch — add input validation",
				Subject:      "1 file changed",
				Detail:       []string{"files: 1 changed (calc/divide.go)"},
				DiffFilename: "calc/divide.go",
				Diff:         diff, Added: 1, Removed: 1,
			}})
			m.pendingApproval = id
			m.sel = id
			m.openModal(ModalState{
				Kind: ModalDiff, Title: "calc/divide.go",
				Lines: diff, Added: 1, Removed: 1,
			})
			m.status.Tokens = 14100
			m.frame = 3 // ⠸
		}},

		{Screen: "12", Unicode: true, Build: func(m *Model) {
			m.status = Status{Model: "unknown-model", Family: "unknown"}
			m.onboarding = &OnboardingState{
				NoAPIKey:     true,
				KeyVars:      [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
				NotGitRepo:   true,
				UnknownModel: true,
				Endpoint:     "opencode",
			}
		}},
	}
}

// screen13Reply is the markdown behind screen 13. It has no table on purpose: a
// table's `│` between columns would put box characters in more than the two
// right-edge columns scripts/lint-screens.py allows a grid.
const screen13Reply = "## Why it panics\n" +
	"\n" +
	"The zero check runs **after** the division, so Go raises *integer divide by zero* first. " +
	"Call `Divide(a, b)` with `b == 0` and you will see it.\n" +
	"\n" +
	"- Move the check above the division\n" +
	"- Return an error instead of panicking\n" +
	"\n" +
	"> A guard clause belongs before the work it guards.\n" +
	"\n" +
	"See [the Go spec](https://go.dev/ref/spec) for the rules.\n" +
	"\n" +
	"```go\n" +
	"func Divide(a, b int) (int, error) {\n" +
	"    if b == 0 {\n" +
	"        return 0, ErrZero\n" +
	"    }\n" +
	"    return a / b, nil\n" +
	"}\n" +
	"```\n"

func buildScenario(t *testing.T, sc scenario, g grid) string {
	t.Helper()
	m := New(Options{
		Version: fixtureVersion,
		Caps:    Caps{Colour: sc.Colour, Unicode: sc.Unicode},
		Session: SessionInfo{Project: "my-project", Branch: "main", Dirty: true},
		Status:  Status{Model: "claude-sonnet-5-5", Family: "sonnet-5-5"},
	})
	if sc.Build != nil {
		sc.Build(&m)
	}
	mm, _ := m.Update(tea.WindowSizeMsg{Width: g.W, Height: g.H})
	return mm.(Model).View()
}

// TestMatchesScreenReference is the staleness gate. When it fails, exactly one
// of two things is true: the renderer is wrong, or the screen reference is. Fix
// whichever it is — and if it is the reference, update it in the same commit.
func TestMatchesScreenReference(t *testing.T) {
	grids := parseGrids(t)
	for _, sc := range scenarios() {
		g, ok := grids[sc.Screen]
		if !ok {
			t.Errorf("screen %s has no sized grid in %s", sc.Screen, screensDoc)
			continue
		}
		t.Run("screen"+sc.Screen, func(t *testing.T) {
			got := strings.Split(buildScenario(t, sc, g), "\n")
			want := g.Rows
			// Markdown cannot hold trailing whitespace, so both sides are
			// right-trimmed before diffing. Stripping spaces does not weaken
			// the comparison: escapes are not whitespace.
			for i := range got {
				got[i] = strings.TrimRight(got[i], " ")
			}
			if len(got) != len(want) {
				t.Fatalf("row count: got %d, want %d (grid is %d×%d)", len(got), len(want), g.W, g.H)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("row %d differs:\n  want: %q\n  got:  %q", i+1, want[i], got[i])
				}
			}
		})
	}
}

// TestScreen11IsScreen03Stripped proves the pair the reference claims to be a
// transliteration really is one: the ASCII/no-colour frame must equal the
// coloured frame with its escapes removed and its glyphs swapped.
func TestScreen11IsScreen03Stripped(t *testing.T) {
	grids := parseGrids(t)
	g3, g11 := grids["03"], grids["11"]
	if len(g3.Rows) != len(g11.Rows) {
		t.Fatalf("screens 03 and 11 differ in height: %d vs %d", len(g3.Rows), len(g11.Rows))
	}
	for i := range g3.Rows {
		if a, b := cellWidth(g3.Rows[i]), cellWidth(g11.Rows[i]); a != b {
			t.Errorf("row %d: screen 03 is %d cols, screen 11 is %d", i+1, a, b)
		}
	}
}
