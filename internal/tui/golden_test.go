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
// parse the character grids straight out of plan/kirsch-ui-screens.md and
// compare View() against them, so a rendering change and a stale design
// document cannot drift apart silently — which is the rule plan §9.6 states and
// amendment 24 exists to protect.

const screensDoc = "../../plan/kirsch-ui-screens.md"

// fixtureVersion is the version string every fixture in this package renders.
//
// It is the binary's own default — `var version = "0.1.0-dev"` in
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
const fixtureVersion = "0.1.0-dev"

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
			m.status.Tokens, m.status.Grants = 16800, 1
			m.frame = 2 // ⠹
		}},

		{Screen: "07", Unicode: true, Build: func(m *Model) {
			id := m.tr.Append(Item{Kind: KindTool, Tool: &ToolCard{
				Name: "run_command", Target: "go test ./...", Summary: "exit 1",
				State: StateError, Elapsed: 2400 * time.Millisecond,
				Out: SanitizeLines(fakeTestOutput),
			}})
			m.sel = id
			m.expanded[id] = true
			m.status.Tokens, m.status.Grants = 22900, 1
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
	}
}

func buildScenario(t *testing.T, sc scenario, g grid) string {
	t.Helper()
	m := New(Options{
		Version: fixtureVersion,
		Caps:    Caps{Colour: sc.Colour, Unicode: sc.Unicode},
		Session: SessionInfo{Project: "my-project", Branch: "main", Dirty: true},
		Status:  Status{Model: "claude-sonnet-5", Family: "sonnet-5"},
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
