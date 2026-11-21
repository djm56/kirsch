package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Screen captures for the manual walkthroughs (DIR-021).
//
// A walkthrough that tells the operator what a screen looks like has to quote a
// screen the program drew. Each row of captureTable builds one model state with
// canned transcript items and real key messages through Update, the way the
// golden scenarios do: no provider, no network, no model key. Its frame is
// written to plan/testing/manual/captures/<release>/<name>.txt, and the test
// compares every file against the current render so a capture cannot go stale
// silently.
//
// Later releases add rows to captureTable with their own Release string. The file
// locations, the write and compare modes and the inventory check all follow from
// the rows. A new key name for capKeys is the one other edit: it goes in
// captureKeyNames, and capKeys fails on a multi-character name that is not there.
//
// Inputs are the producers' own formats. The read_file card is fed the summary and
// the numbered content that internal/tool's read_file returns, and /diff is fed a
// header with the index line git prints. internal/tui may not import internal/tool,
// so TestCaptureInputFormats pins these inputs to literals here and
// TestCaptureInputsMatchProducers in internal/app pins the same literals to the
// real read_file tool and a real git diff.
//
// Modes. KIRSCH_CAPTURE_WRITE=1 (re)writes the files. Without it the test fails,
// with the differing rows, when a file is missing or differs. Write mode never
// deletes: a stray file is reported by TestCaptureInventory and removed by hand.
//
// Which frame is captured. The .txt file is View() from a colour-off model, so
// it is the exact text a terminal shows and reads without escapes. A state that
// sets Annotate is built a second time with colour on and also gets a
// <name>.annot.txt listing the colour roles; colour-off and colour-on frames are
// asserted to carry the same text (TestCaptureStyledStripsToPlain), so the two
// files describe one frame. The .annot.txt header ends with the line
// "=== annotations ===" and holds no frame; the frame is in the .txt file.
//
// Annotation form. After a short header and a legend, one line per cell run:
//
//	row 07 cols 12-15 fg=Success(38;5;120) "+foo"
//
// Rows count from 0 at the top of the frame and columns from 0 at its left edge,
// margin included. A run is consecutive cells on one row with the same attributes.
// An attribute is fg=<role>(<SGR>), bg=<role>(<SGR>) or bold, italic, strike, with
// the SGR parameters as the program emitted them. The role name is looked up from
// the SGR number alone (capturePalette), so two roles that share a number read as
// the first one listed there: the modal border (BorderFocus, 117) is reported as
// Accent. An SGR number outside capturePalette fails the test.
// Annotate "bar" lists, for every row whose last cell is in the last column, that
// cell alone, marking one with no attributes "plain"; a row that stops short of the
// last column has no line. Annotate "emph" lists every run except plain runs and
// runs whose only attribute is a foreground of Text, Muted, Dim or Border: the
// diff colours, the syntax roles, the backgrounds, bold and the rest.

const capturesRoot = "../../plan/testing/manual/captures"

// captureVersion is the version the captured frames show in the header. It is
// fixed here, not taken from fixtureVersion or the release, so a version bump
// leaves the capture files alone.
const captureVersion = "0.1.5-capture"

// Separator lines that end a file's header: the .txt file's frame follows its
// line, the .annot.txt file's run listing follows its line.
const (
	captureFrameSep = "=== frame ===\n"
	captureAnnotSep = "=== annotations ===\n"
)

// captureStep is one input to the model. Msgs builds the messages from the
// current model, because some inputs depend on it (the /diff result carries the
// sequence number the model issued).
type captureStep struct {
	Name string // what the walkthrough quotes: a key, or a canned arrival
	Msgs func(m Model) []tea.Msg
}

// captureState is one row of the table.
type captureState struct {
	Release  string // the folder under captures/, e.g. "v0.1.5"
	Name     string // the file name stem, and the name the walkthrough cites
	W, H     int
	Unicode  bool
	Annotate string // "", "bar" or "emph"; see the annotation form above
	Setup    func(m *Model)
	Steps    []captureStep
}

var captureKeyNames = map[string]tea.KeyType{
	"Shift+↑": tea.KeyShiftUp, "PgUp": tea.KeyPgUp, "PgDn": tea.KeyPgDown,
	"Home": tea.KeyHome, "End": tea.KeyEnd, "Esc": tea.KeyEsc,
	"Enter": tea.KeyEnter, "Tab": tea.KeyTab,
}

// capKeyMsg is the message for one key name: a name from captureKeyNames, or a
// single character. Any other name is an error, so a misspelt key cannot type its
// first letter.
func capKeyMsg(n string) (tea.Msg, error) {
	if kt, ok := captureKeyNames[n]; ok {
		return keyType(kt), nil
	}
	if r := []rune(n); len(r) == 1 {
		return key(r[0]), nil
	}
	return nil, fmt.Errorf("unknown key name %q: not in captureKeyNames and not a single character", n)
}

// capKeys builds a step that presses the named keys in order, by capKeyMsg. A name
// capKeyMsg rejects panics when the step runs, which fails the test that built the
// model.
func capKeys(names ...string) captureStep {
	return captureStep{Name: strings.Join(names, " "), Msgs: func(Model) []tea.Msg {
		var out []tea.Msg
		for _, n := range names {
			msg, err := capKeyMsg(n)
			if err != nil {
				panic("capKeys: " + err.Error())
			}
			out = append(out, msg)
		}
		return out
	}}
}

// capMsgs builds a step that delivers canned messages, as the app would.
func capMsgs(name string, msgs ...tea.Msg) captureStep {
	return captureStep{Name: name, Msgs: func(Model) []tea.Msg { return msgs }}
}

// capText is a step that types text into the composer, one key per character.
func capText(text string) captureStep {
	return captureStep{Name: text, Msgs: func(Model) []tea.Msg {
		var out []tea.Msg
		for _, r := range text {
			out = append(out, key(r))
		}
		return out
	}}
}

// capTurn is a canned user message and a reply, loaded the way a resumed session
// loads them: the assistant block carries Lines and the unsanitised Source, as the
// resume loader in internal/app builds it.
func capTurn(user string, reply ...string) tea.Msg {
	text := strings.Join(reply, "\n")
	return TranscriptLoadedMsg{Items: []Item{
		{Kind: KindUser, Text: &TextBlock{Lines: []string{user}}},
		{Kind: KindAssistant, Text: &TextBlock{Lines: SanitizeLines(text), Source: text}},
	}}
}

// capReadFile is a finished read_file call: the tool starting, then completing
// the way the app sends it (toolExecutor.Invoke passes the tool's DisplaySummary
// and Content). The content is the file's text numbered "%d\t%s\n" per line and
// the summary is "lines 1-N", both as internal/tool/readfile.go writes them for a
// whole file under its line and byte caps. TestCaptureInputFormats pins this
// output to literals, and TestCaptureInputsMatchProducers in internal/app pins the
// same literals to the real tool.
func capReadFile(target, text string) []tea.Msg {
	var numbered strings.Builder
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, line := range lines {
		fmt.Fprintf(&numbered, "%d\t%s\n", i+1, line)
	}
	return []tea.Msg{
		ToolStartedMsg{ID: 1, Name: "read_file", Target: target},
		ToolCompletedMsg{
			ID: 1, Name: "read_file", Target: target, OK: true,
			Summary: fmt.Sprintf("lines 1-%d", len(lines)), Content: numbered.String(),
		},
	}
}

// capReplyLines is n short reply rows, ASCII only so the ASCII state shows glyphs
// from the bar and not from the text.
func capReplyLines(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("Step %d: the zero check in Divide runs after the division.", i+1)
	}
	return out
}

const capDivideFile = "package calc\n\nimport \"errors\"\n\n" +
	"// Divide returns a/b, or an error when b is zero.\n" +
	"func Divide(a, b float64) (float64, error) {\n" +
	"\tif b == 0 {\n\t\treturn 0, errors.New(\"division by zero\")\n\t}\n" +
	"\treturn a / b, nil\n}\n"

// capOverflow loads a transcript longer than any viewport in the table and ends
// with a read_file card, the card the modal states open with `d`.
func capOverflow() []captureStep {
	return []captureStep{
		capMsgs("(canned transcript loads)",
			capTurn("Why does Divide panic on zero?", capReplyLines(14)...)),
		capMsgs("(canned second turn loads)",
			capTurn("And where is it called from?", capReplyLines(14)...)),
		capMsgs("(a read_file card finishes)", capReadFile("calc/divide.go", capDivideFile)...),
	}
}

func appendSteps(base []captureStep, more ...captureStep) []captureStep {
	return append(append([]captureStep(nil), base...), more...)
}

const capLongGo = "var usage = \"" +
	"alpha-0123456789 bravo-0123456789 charlie-0123456789 delta-0123456789 " +
	"echo-0123456789 foxtrot-0123456789 golf-0123456789 hotel-0123456789\"\n"

// capDiff is the canned /diff result: a file header with the index line git
// prints, a hunk header long enough to wrap, a long added line, a long removed
// line and context. capDiffHeaderShapes pins the header rows to git's layout.
func capDiff() DiffResult {
	long := strings.Repeat("0123456789", 14)
	return DiffResult{
		Lines: []string{
			"diff --git a/calc/divide.go b/calc/divide.go",
			"index 3f2a1c4..9b8e7d6 100644",
			"--- a/calc/divide.go",
			"+++ b/calc/divide.go",
			"@@ -12,7 +12,15 @@ func Divide(a, b float64) (float64, error) { // " + long,
			" \tif b == 0 {",
			"+\treturn divideChecked(a, b, \"" + long + "\")",
			"-\treturn legacyDivide(a, b, \"" + long + "\")",
			" \t}",
		},
		Added: 1, Removed: 1,
	}
}

// capDiffHeaderShapes is the layout of the first five rows of a git diff for one
// modified file, one pattern per row. The same patterns are repeated in
// TestCaptureInputsMatchProducers (internal/app), which checks them against a
// real git diff.
var capDiffHeaderShapes = []string{
	`^diff --git a/\S+ b/\S+$`,
	`^index [0-9a-f]{7,}\.\.[0-9a-f]{7,} [0-7]{6}$`,
	`^--- a/\S+$`,
	`^\+\+\+ b/\S+$`,
	`^@@ -\d+,\d+ \+\d+,\d+ @@`,
}

// captureTable is every state a manual walkthrough cites. Steps name the keys the
// operator presses; the walkthrough quotes the Name column.
var captureTable = []captureState{
	{
		Release: "v0.1.5", Name: "scrollbar-bottom", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: capOverflow(),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-top", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: appendSteps(capOverflow(), capKeys("Shift+↑"), capKeys("g")),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-mid-new", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: appendSteps(capOverflow(), capKeys("Shift+↑"), capKeys("g"), capKeys("PgDn"),
			capMsgs("(three notices arrive)", NoticeMsg{Text: "one"}, NoticeMsg{Text: "two"}, NoticeMsg{Text: "three"})),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-hidden-short", W: 80, H: 24, Unicode: true,
		Steps: []captureStep{
			capMsgs("(canned short transcript loads)", capTurn("Hello", "Hi there.")),
			capMsgs("(a read_file card finishes)", capReadFile("calc/divide.go", "package calc\n")...),
		},
	},
	{
		Release: "v0.1.5", Name: "scrollbar-hidden-modal", W: 80, H: 24, Unicode: true, Annotate: "bar",
		Steps: appendSteps(capOverflow(), capKeys("Shift+↑"), capKeys("d")),
	},
	{
		Release: "v0.1.5", Name: "scrollbar-ascii", W: 80, H: 24, Unicode: false, Annotate: "bar",
		Steps: capOverflow(),
	},
	{
		Release: "v0.1.5", Name: "modal-readfile-wrapped", W: 80, H: 24, Unicode: true, Annotate: "emph",
		Steps: []captureStep{
			capMsgs("(a read_file card finishes)",
				capReadFile("calc/usage.go", "package calc\n\n"+capLongGo+"\nfunc Usage() string { return usage }\n")...),
			capKeys("Shift+↑"), capKeys("d"),
		},
	},
	{
		Release: "v0.1.5", Name: "modal-diff-wrapped", W: 80, H: 32, Unicode: true, Annotate: "emph",
		Setup: func(m *Model) { m.GetDiff = capDiff },
		Steps: []captureStep{
			capText("/diff"), capKeys("Enter"),
			{Name: "(the git diff result arrives)", Msgs: func(m Model) []tea.Msg {
				return []tea.Msg{DiffResultMsg{Result: capDiff(), Seq: m.pendingDiffSeq}}
			}},
		},
	},
	{
		Release: "v0.1.5", Name: "modal-after-G", W: 80, H: 24, Unicode: true,
		Steps: []captureStep{
			capMsgs("(a read_file card finishes)",
				capReadFile("calc/long.txt", strings.Repeat("line of the file\n", 60))...),
			capKeys("Shift+↑"), capKeys("d"), capKeys("G"),
		},
	},
	{
		Release: "v0.1.5", Name: "help-cut", W: 60, H: 24, Unicode: true,
		Steps: []captureStep{capKeys("Shift+↑"), capKeys("?")},
	},
}

// captureModel builds a state's model with colour on or off and returns it after
// every step.
func captureModel(t *testing.T, cs captureState, colour bool) Model {
	t.Helper()
	m := New(Options{
		Version: captureVersion,
		Caps:    Caps{Colour: colour, Unicode: cs.Unicode},
		Session: SessionInfo{Project: "my-project", Branch: "main", Dirty: true},
		Status:  Status{Model: "claude-sonnet-5-5", Family: "sonnet-5-5"},
	})
	if cs.Setup != nil {
		cs.Setup(&m)
	}
	m = drive(m, tea.WindowSizeMsg{Width: cs.W, Height: cs.H})
	for _, st := range cs.Steps {
		m = drive(m, st.Msgs(m)...)
	}
	return m
}

// captureHeader is the file header: the state, the size, the glyph set and the
// inputs, then sep, the separator line that ends the header.
func captureHeader(cs captureState, kind, sep string) string {
	glyphs := "unicode"
	if !cs.Unicode {
		glyphs = "ascii"
	}
	var steps []string
	for _, st := range cs.Steps {
		steps = append(steps, st.Name)
	}
	return fmt.Sprintf("# state: %s\n# release: %s\n# size: %dx%d\n# glyphs: %s\n# steps: %s\n# %s\n# no model, no network, no key: canned messages through the real Model\n%s",
		cs.Name, cs.Release, cs.W, cs.H, glyphs, strings.Join(steps, " | "), kind, sep)
}

// capturePlain is the text file's content for a state.
func capturePlain(t *testing.T, cs captureState) string {
	t.Helper()
	m := captureModel(t, cs, false)
	return captureHeader(cs, "frame text: View() with colour off", captureFrameSep) + m.View() + "\n"
}

// capturePalette maps the foreground and background SGR parameters to the role
// names of the palette comments in styles.go.
var capturePalette = map[string]string{
	"38;5;253": "Text", "38;5;248": "Muted", "38;5;245": "Dim", "38;5;117": "Accent",
	"38;5;120": "Success", "38;5;203": "Error", "38;5;215": "Warning", "38;5;123": "Hunk",
	"38;5;244": "Border", "38;5;141": "SynKeyword", "38;5;180": "SynString",
	"38;5;216": "SynNumber", "38;5;152": "SynType",
	"48;5;235": "CodeBg", "48;5;236": "SelectionBg",
}

// capRun is a run of cells on one row sharing one attribute set.
type capRun struct {
	Row, From, To int
	Attr          string // "" for no attributes
	Quiet         bool   // the only attribute is a foreground of Text, Muted, Dim or Border
	Text          string
}

// captureRuns is captureRunsErr that fails the test on an error.
func captureRuns(t *testing.T, row int, s string) []capRun {
	t.Helper()
	runs, err := captureRunsErr(row, s)
	if err != nil {
		t.Fatal(err)
	}
	return runs
}

// captureRunsErr parses one styled row into runs. It returns an error for an
// escape it does not handle and for a foreground or background colour that is not
// in capturePalette, since a colour without a role would print an empty role name.
// The attribute groups it handles are the ones the program emits: reset, bold,
// italic, strike, their off codes, default foreground, and 256-colour fg and bg.
func captureRunsErr(row int, s string) ([]capRun, error) {
	var fg, bg string
	var bold, italic, strike bool
	var runs []capRun
	col := 0
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "\x1b[") {
			end := strings.IndexByte(s[i:], 'm')
			if end < 0 {
				return nil, fmt.Errorf("row %d: unterminated escape", row)
			}
			switch p := s[i+2 : i+end]; {
			case p == "0":
				fg, bg, bold, italic, strike = "", "", false, false, false
			case p == "1":
				bold = true
			case p == "3":
				italic = true
			case p == "9":
				strike = true
			case p == "22":
				bold = false
			case p == "23":
				italic = false
			case p == "29":
				strike = false
			case p == "39":
				fg = ""
			case strings.HasPrefix(p, "38;5;"):
				fg = p
			case strings.HasPrefix(p, "48;5;"):
				bg = p
			default:
				return nil, fmt.Errorf("row %d: unhandled SGR %q", row, p)
			}
			i += end + 1
			continue
		}
		r := []rune(s[i:])[0]
		i += len(string(r))
		w := cellWidth(string(r))
		var attrs []string
		if fg != "" {
			if _, ok := capturePalette[fg]; !ok {
				return nil, fmt.Errorf("row %d col %d: foreground SGR %q is not in capturePalette", row, col, fg)
			}
			attrs = append(attrs, fmt.Sprintf("fg=%s(%s)", capturePalette[fg], fg))
		}
		if bg != "" {
			if _, ok := capturePalette[bg]; !ok {
				return nil, fmt.Errorf("row %d col %d: background SGR %q is not in capturePalette", row, col, bg)
			}
			attrs = append(attrs, fmt.Sprintf("bg=%s(%s)", capturePalette[bg], bg))
		}
		for _, f := range []struct {
			on   bool
			name string
		}{{bold, "bold"}, {italic, "italic"}, {strike, "strike"}} {
			if f.on {
				attrs = append(attrs, f.name)
			}
		}
		attr := strings.Join(attrs, " ")
		quiet := bg == "" && !bold && !italic && !strike && fg != "" &&
			map[string]bool{"Text": true, "Muted": true, "Dim": true, "Border": true}[capturePalette[fg]]
		if n := len(runs); n > 0 && runs[n-1].Attr == attr && runs[n-1].To == col-1 {
			runs[n-1].To = col + w - 1
			runs[n-1].Text += string(r)
		} else {
			runs = append(runs, capRun{Row: row, From: col, To: col + w - 1, Attr: attr, Quiet: quiet, Text: string(r)})
		}
		col += w
	}
	return runs, nil
}

// captureAnnot is the annotated file's content for a state: the colour-on frame
// described by its runs.
func captureAnnot(t *testing.T, cs captureState) string {
	t.Helper()
	m := captureModel(t, cs, true)
	var b strings.Builder
	b.WriteString(captureHeader(cs, "annotations: colour roles of the colour-on frame ("+cs.Annotate+")", captureAnnotSep))
	b.WriteString("legend: row/cols count from 0 at the top-left of the frame; fg/bg=<role>(<SGR>)\n")
	listed := 0
	for r, row := range strings.Split(m.View(), "\n") {
		for _, run := range captureRuns(t, r, row) {
			switch cs.Annotate {
			case "bar":
				if run.To != cs.W-1 {
					// a run reaching the last column is the bar cell; a run ending
					// earlier is a row with no cell there.
					continue
				}
				attr := run.Attr
				if attr == "" {
					attr = "plain"
				}
				listed++
				fmt.Fprintf(&b, "row %02d col %d %s %q\n", r, cs.W-1, attr, run.Text[len(run.Text)-lastRuneLen(run.Text):])
			case "emph":
				if run.Attr == "" || run.Quiet {
					continue
				}
				fmt.Fprintf(&b, "row %02d cols %d-%d %s %q\n", r, run.From, run.To, run.Attr, run.Text)
			}
		}
	}
	if cs.Annotate == "bar" && listed == 0 {
		fmt.Fprintf(&b, "no row of the frame has a cell in column %d\n", cs.W-1)
	}
	return b.String()
}

// lastRuneLen is the byte length of the final rune of s.
func lastRuneLen(s string) int {
	r := []rune(s)
	return len(string(r[len(r)-1]))
}

// captureDir is the folder a release's files live in.
func captureDir(release string) string {
	return filepath.Join(filepath.FromSlash(capturesRoot), release)
}

// captureFiles is the file name to content for one state.
func captureFiles(t *testing.T, cs captureState) map[string]string {
	t.Helper()
	out := map[string]string{cs.Name + ".txt": capturePlain(t, cs)}
	if cs.Annotate != "" {
		out[cs.Name+".annot.txt"] = captureAnnot(t, cs)
	}
	return out
}

// diffText names the rows where got differs from want, one - and + pair each, or
// the row counts when they differ in length. It returns "" when the two are equal.
func diffText(want, got string) string {
	if want == got {
		return ""
	}
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		wok, gok := i < len(w), i < len(g)
		if wok {
			wl = w[i]
		}
		if gok {
			gl = g[i]
		}
		if wok && gok && wl == gl {
			continue
		}
		if wok {
			fmt.Fprintf(&b, "line %d - %q\n", i+1, wl)
		}
		if gok {
			fmt.Fprintf(&b, "line %d + %q\n", i+1, gl)
		}
	}
	return b.String()
}

// checkCaptureFile compares the file at path with got and returns the
// differences, or the read error, or "" when they match. It is the one compare
// the test uses, and the control test calls it on an edited copy.
func checkCaptureFile(path, got string) string {
	want, err := os.ReadFile(path)
	if err != nil {
		return "cannot read: " + err.Error()
	}
	return diffText(string(want), got)
}

// TestCaptureFiles is the capture itself. With KIRSCH_CAPTURE_WRITE=1 it writes
// every state's files; without it, it fails for each file that is missing or
// differs from the current render.
func TestCaptureFiles(t *testing.T) {
	write := os.Getenv("KIRSCH_CAPTURE_WRITE") == "1"
	for _, cs := range captureTable {
		cs := cs
		t.Run(cs.Release+"/"+cs.Name, func(t *testing.T) {
			for name, got := range captureFiles(t, cs) {
				path := filepath.Join(captureDir(cs.Release), name)
				if write {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if d := checkCaptureFile(path, got); d != "" {
					t.Errorf("%s differs from the render (regenerate with KIRSCH_CAPTURE_WRITE=1 go test -count=1 -run TestCapture ./internal/tui):\n%s", path, d)
				}
			}
		})
	}
}

// duplicateRows names the "release/name" pairs that more than one row of table
// carries.
func duplicateRows(table []captureState) []string {
	seen := map[string]bool{}
	var dup []string
	for _, cs := range table {
		id := cs.Release + "/" + cs.Name
		if seen[id] {
			dup = append(dup, id)
		}
		seen[id] = true
	}
	return dup
}

// foldersWithoutRows names the folders directly under root that are not in
// releases, the set of release names the table has rows for.
func foldersWithoutRows(root string, releases map[string]bool) ([]string, error) {
	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, d := range dirs {
		if d.IsDir() && !releases[d.Name()] {
			out = append(out, d.Name())
		}
	}
	return out, nil
}

// TestCaptureInventory checks the table against the folders: every file a row
// produces exists, every file under a release folder (dotfiles aside) belongs to a
// row, no two rows share a name, and every release folder has rows. A stray, a
// duplicate or a missing file fails.
func TestCaptureInventory(t *testing.T) {
	want := map[string][]string{}
	for _, id := range duplicateRows(captureTable) {
		t.Errorf("table has two rows named %s", id)
	}
	for _, cs := range captureTable {
		for name := range captureFiles(t, cs) {
			want[cs.Release] = append(want[cs.Release], name)
		}
	}
	for release, names := range want {
		missing, stray := inventoryGap(captureDir(release), names)
		for _, n := range missing {
			t.Errorf("table row has no file: %s/%s", release, n)
		}
		for _, n := range stray {
			t.Errorf("file belongs to no table row: %s/%s", release, n)
		}
	}
	releases := map[string]bool{}
	for r := range want {
		releases[r] = true
	}
	empty, err := foldersWithoutRows(filepath.FromSlash(capturesRoot), releases)
	if err != nil {
		t.Fatalf("read captures folder: %v", err)
	}
	for _, d := range empty {
		t.Errorf("folder %s has no rows in captureTable", d)
	}
}

// TestCaptureInventoryDetectsStrayAndMissing is the control for the inventory
// test: the same comparison, on a temporary folder, fails for a stray file and for
// a missing one, passes when the sets match, ignores a dotfile (.DS_Store), flags
// a duplicate row and flags a folder with no rows, each beside a case that passes.
func TestCaptureInventoryDetectsStrayAndMissing(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.txt", "b.txt", ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if m, s := inventoryGap(dir, []string{"a.txt", "b.txt"}); len(m)+len(s) != 0 {
		t.Fatalf("control: matching sets reported missing %v stray %v", m, s)
	}
	if m, s := inventoryGap(dir, []string{"a.txt", "b.txt", "c.txt"}); len(m) != 1 || len(s) != 0 {
		t.Errorf("a missing file: missing %v stray %v, want one missing", m, s)
	}
	if m, s := inventoryGap(dir, []string{"a.txt"}); len(m) != 0 || len(s) != 1 {
		t.Errorf("a stray file: missing %v stray %v, want one stray", m, s)
	}
	// .DS_Store is in dir in every call above and in no want list. The matching-sets
	// call reports nothing and the stray call reports b.txt alone, so the dotfile is
	// ignored while a non-dot stray is still found.

	rows := []captureState{{Release: "v1", Name: "a"}, {Release: "v1", Name: "b"}, {Release: "v2", Name: "a"}}
	if d := duplicateRows(rows); len(d) != 0 {
		t.Errorf("control: rows with distinct release/name reported duplicates %v", d)
	}
	if d := duplicateRows(append(rows, captureState{Release: "v1", Name: "a"})); len(d) != 1 || d[0] != "v1/a" {
		t.Errorf("a repeated release/name: duplicates %v, want [v1/a]", d)
	}

	root := t.TempDir()
	for _, n := range []string{"v1", "v9"} {
		if err := os.Mkdir(filepath.Join(root, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if f, err := foldersWithoutRows(root, map[string]bool{"v1": true, "v9": true}); err != nil || len(f) != 0 {
		t.Errorf("control: folders %v err %v with rows for every folder, want none", f, err)
	}
	if f, err := foldersWithoutRows(root, map[string]bool{"v1": true}); err != nil || len(f) != 1 || f[0] != "v9" {
		t.Errorf("a folder with no rows: %v err %v, want [v9]", f, err)
	}
}

// inventoryGap compares the file names in dir with want and returns the missing
// names and the stray ones. A name starting with "." is never stray (.DS_Store and
// the like), and every other file in dir that is not in want is.
func inventoryGap(dir string, want []string) (missing, stray []string) {
	files, _ := os.ReadDir(dir)
	have := map[string]bool{}
	for _, f := range files {
		have[f.Name()] = true
	}
	w := map[string]bool{}
	for _, n := range want {
		w[n] = true
		if !have[n] {
			missing = append(missing, n)
		}
	}
	for n := range have {
		if !w[n] && !strings.HasPrefix(n, ".") {
			stray = append(stray, n)
		}
	}
	return missing, stray
}

// TestCaptureCompareFailsOnOneCellEdit is the control for the compare mode, run on
// a .txt file and on an .annot.txt file. On a temporary copy of a committed
// capture, an unedited copy passes; a copy with one cell added after the header
// separator fails and the report names that line as one - and one + row; a copy
// with one trailing space appended fails. The committed files are never touched.
func TestCaptureCompareFailsOnOneCellEdit(t *testing.T) {
	cs := captureTable[0]
	for _, c := range []struct{ file, sep string }{
		{cs.Name + ".txt", captureFrameSep},
		{cs.Name + ".annot.txt", captureAnnotSep},
	} {
		t.Run(c.file, func(t *testing.T) {
			got := captureFiles(t, cs)[c.file]
			committed, err := os.ReadFile(filepath.Join(captureDir(cs.Release), c.file))
			if err != nil {
				t.Fatalf("read committed capture: %v", err)
			}
			tmp := filepath.Join(t.TempDir(), c.file)
			check := func(content string) string {
				if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
				return checkCaptureFile(tmp, got)
			}
			if d := check(string(committed)); d != "" {
				t.Fatalf("control: an unedited copy differs from the render:\n%s", d)
			}

			edited := strings.Replace(string(committed), c.sep, c.sep+"X", 1)
			if edited == string(committed) {
				t.Fatal("setup: the edit changed nothing")
			}
			d := check(edited)
			if d == "" {
				t.Fatal("a copy with one cell added after the header separator passed the compare")
			}
			if strings.Count(d, "\n") != 2 {
				t.Errorf("the report is not one - and one + line:\n%s", d)
			}

			// A trailing space on the last row: the frames carry trailing spaces on
			// purpose, so the compare must not trim.
			if d := check(strings.TrimSuffix(string(committed), "\n") + " \n"); d == "" {
				t.Error("a copy with one trailing space added to the last row passed the compare")
			}
		})
	}
}

// TestCaptureKeyNames is the control for capKeys: a name in captureKeyNames and a
// single character give a message, an unknown name longer than one character gives
// an error and does not fall back to its first letter, and running a step built
// with such a name panics.
func TestCaptureKeyNames(t *testing.T) {
	for _, n := range []string{"PgDn", "Esc", "g", "d"} {
		if msg, err := capKeyMsg(n); err != nil || msg == nil {
			t.Errorf("control: capKeyMsg(%q) = %v, %v, want a message", n, msg, err)
		}
	}
	for _, n := range []string{"PageDown", "Shift+Up", "Entr", ""} {
		if msg, err := capKeyMsg(n); err == nil {
			t.Errorf("capKeyMsg(%q) = %v, want an error", n, msg)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("running capKeys(\"PageDown\") did not panic")
			}
		}()
		capKeys("PageDown").Msgs(Model{})
	}()
	if n := len(capKeys("PgDn", "g").Msgs(Model{})); n != 2 {
		t.Errorf("control: capKeys(PgDn, g) built %d messages, want 2", n)
	}
}

// TestCaptureRunsRejectUnknownColour is the control for captureRunsErr: a
// foreground and a background that are in capturePalette parse, one that is not
// fails for each of fg and bg, and an unknown parameter group still fails.
func TestCaptureRunsRejectUnknownColour(t *testing.T) {
	if runs, err := captureRunsErr(0, "\x1b[38;5;117m\x1b[48;5;235mx\x1b[0m"); err != nil || len(runs) != 1 ||
		runs[0].Attr != "fg=Accent(38;5;117) bg=CodeBg(48;5;235)" {
		t.Fatalf("control: palette colours gave %+v, %v", runs, err)
	}
	for _, row := range []string{"\x1b[38;5;99mx\x1b[0m", "\x1b[48;5;99mx\x1b[0m", "\x1b[7mx\x1b[0m"} {
		if runs, err := captureRunsErr(0, row); err == nil {
			t.Errorf("captureRunsErr(%q) = %+v, want an error", row, runs)
		}
	}
}

// TestCaptureInputFormats pins the canned inputs to the producers' formats: the
// read_file summary and numbered content, and the first five rows of the /diff
// text. The literals and patterns repeat in TestCaptureInputsMatchProducers in
// internal/app, which runs the real tool and a real git diff against them. Entry
// points: capReadFile and capDiff, called directly. Positive control: a one-line
// file, whose summary is the range "lines 1-1", not a bare count.
func TestCaptureInputFormats(t *testing.T) {
	msgs := capReadFile("x.go", "a\nb\n")
	done, ok := msgs[1].(ToolCompletedMsg)
	if !ok {
		t.Fatalf("capReadFile's second message is %T, want ToolCompletedMsg", msgs[1])
	}
	if done.Summary != "lines 1-2" || done.Content != "1\ta\n2\tb\n" {
		t.Errorf("capReadFile summary %q content %q, want \"lines 1-2\" and \"1\\ta\\n2\\tb\\n\"", done.Summary, done.Content)
	}
	one := capReadFile("x.go", "a\n")[1].(ToolCompletedMsg)
	if one.Summary != "lines 1-1" {
		t.Errorf("control: one-line file summary %q, want \"lines 1-1\"", one.Summary)
	}

	lines := capDiff().Lines
	if len(lines) < len(capDiffHeaderShapes) {
		t.Fatalf("capDiff has %d lines, want at least %d", len(lines), len(capDiffHeaderShapes))
	}
	for i, shape := range capDiffHeaderShapes {
		if !regexp.MustCompile(shape).MatchString(lines[i]) {
			t.Errorf("capDiff line %d is %q, want it to match %s", i, lines[i], shape)
		}
	}
}

// TestCaptureStyledStripsToPlain: the colour-on frame of every annotated state,
// with its escapes stripped, equals the colour-off frame, so the .annot.txt file
// describes the frame in the .txt file.
func TestCaptureStyledStripsToPlain(t *testing.T) {
	n := 0
	for _, cs := range captureTable {
		if cs.Annotate == "" {
			continue
		}
		n++
		plain := captureModel(t, cs, false).View()
		styled := stripSGR(captureModel(t, cs, true).View())
		if plain != styled {
			t.Errorf("%s: stripped colour frame differs from plain:\n%s", cs.Name, diffText(plain, styled))
		}
	}
	if n == 0 {
		t.Fatal("no annotated state in the table")
	}
}

// capState finds a row by name.
func capState(t *testing.T, name string) captureState {
	t.Helper()
	for _, cs := range captureTable {
		if cs.Name == name {
			return cs
		}
	}
	t.Fatalf("no capture state named %q", name)
	return captureState{}
}

// capHasBar reports whether any of the cells is a bar glyph.
func capHasBar(cells []string) bool {
	for _, c := range cells {
		if c == "│" || c == "┃" || c == "|" || c == "#" {
			return true
		}
	}
	return false
}

// TestCaptureScrollbarStates: entry point View() through captureModel. Each
// scrollbar capture shows what its name says, read from the last column of the
// transcript rows. The bottom capture is the positive control for the modal
// capture, which has the same transcript and the same size and draws a bar when
// no modal is open (scrollbar-hidden-modal adds only the Shift+Up and d keys). The
// short capture has a different, shorter transcript; its control is the same
// short transcript at a height that does not fit it, which does draw a bar.
func TestCaptureScrollbarStates(t *testing.T) {
	bottom := barCells(t, captureModel(t, capState(t, "scrollbar-bottom"), false))
	if !capHasBar(bottom) {
		t.Fatalf("control: the overflowing transcript draws no bar: %q", bottom)
	}
	last := len(bottom) - 1
	if bottom[last] != "┃" || bottom[0] != "│" {
		t.Errorf("pinned at the bottom: first %q last %q, want the track on top and the thumb at the bottom", bottom[0], bottom[last])
	}

	top := barCells(t, captureModel(t, capState(t, "scrollbar-top"), false))
	if top[0] != "┃" || top[len(top)-1] != "│" {
		t.Errorf("scrolled to the top: first %q last %q, want the thumb at the top", top[0], top[len(top)-1])
	}

	midM := captureModel(t, capState(t, "scrollbar-mid-new"), false)
	mid := barCells(t, midM)
	if mid[0] != "│" || mid[len(mid)-1] != "│" && mid[len(mid)-1] != "┃" {
		t.Errorf("scrolled mid-way: first %q last %q", mid[0], mid[len(mid)-1])
	}
	thumb := -1
	for i, c := range mid {
		if c == "┃" {
			thumb = i
		}
	}
	if thumb <= 0 || thumb >= len(mid)-1 {
		t.Errorf("scrolled mid-way: the thumb's last row is %d of %d, want it strictly inside the track: %q", thumb, len(mid), mid)
	}
	lay := midM.layout()
	rows := strings.Split(midM.View(), "\n")
	band := stripSGR(rows[lay.HeaderH()+lay.TranscriptH-1])
	if !strings.HasSuffix(band, "↓ 3 new"+mid[len(mid)-1]) {
		t.Errorf("mid-way: the last transcript row is %q, want it to end \"↓ 3 new\" and then the bar cell", band)
	}

	short := capState(t, "scrollbar-hidden-short")
	if capHasBar(barCells(t, captureModel(t, short, false))) {
		t.Error("the short transcript capture draws a bar")
	}
	tight := short
	tight.H = 9
	if !capHasBar(barCells(t, captureModel(t, tight, false))) {
		t.Error("control: the short transcript at a 9-row height draws no bar, so the hidden-short capture does not show the bar being hidden for lack of overflow")
	}
	hm := captureModel(t, capState(t, "scrollbar-hidden-modal"), false)
	if hm.modal == nil {
		t.Fatal("the hidden-under-modal capture has no modal open")
	}
	if capHasBar(barCells(t, hm)) {
		t.Error("a bar glyph is drawn in the last column under an open modal")
	}

	asc := barCells(t, captureModel(t, capState(t, "scrollbar-ascii"), false))
	var sawPipe, sawHash bool
	for _, c := range asc {
		switch c {
		case "|":
			sawPipe = true
		case "#":
			sawHash = true
		case "│", "┃":
			t.Errorf("the ASCII capture draws the Unicode glyph %q", c)
		}
	}
	if !sawPipe || !sawHash {
		t.Errorf("the ASCII capture: track | seen %v, thumb # seen %v: %q", sawPipe, sawHash, asc)
	}
}

// TestCaptureModalStates: entry point modalBox on the captured models. The read_file
// capture numbers a wrapped line on its first row only; the /diff capture leaves
// the marker column blank on code continuations and wraps the @@ line flush; the
// G capture is the last full page; the help capture cuts lines, never wraps them.
func TestCaptureModalStates(t *testing.T) {
	// read_file: the long line is numbered 3; its continuation rows start with
	// five blanks and then text. The line before it is the control that a one-row
	// line does carry its number.
	rm := captureModel(t, capState(t, "modal-readfile-wrapped"), false)
	rows := bodyText(t, mustBox(rm))
	at := -1
	for i, r := range rows {
		if regexp.MustCompile(`^ {3}3 3 +var usage`).MatchString(r) {
			at = i
		}
	}
	if at < 0 || at+1 >= len(rows) {
		t.Fatalf("read_file capture: no row starts with the number 3 twice and then \"var usage\": %q", rows)
	}
	if !strings.HasPrefix(rows[at-1], "   2 2") {
		t.Errorf("control: the one-row line before the long one is %q, want it numbered 2", rows[at-1])
	}
	if !regexp.MustCompile(`^ {5}\S`).MatchString(rows[at+1]) {
		t.Errorf("the continuation row is %q, want five blank columns then text", rows[at+1])
	}
	if regexp.MustCompile(`^ *\d+ `).MatchString(rows[at+1]) {
		t.Errorf("the continuation row carries a line number: %q", rows[at+1])
	}

	// /diff.
	dm := captureModel(t, capState(t, "modal-diff-wrapped"), false)
	if dm.modal == nil || dm.modal.Kind != ModalDiff {
		t.Fatalf("diff capture: modal %+v, want the diff modal", dm.modal)
	}
	drows := bodyText(t, mustBox(dm))
	// find is the first row that starts with marker and holds word. The code
	// lines hold a tab in the canned diff, which the modal shows as spaces.
	find := func(marker, word string) int {
		for i, r := range drows {
			if strings.HasPrefix(r, marker) && strings.Contains(r, word) {
				return i
			}
		}
		t.Fatalf("diff capture: no row starts %q and holds %q: %q", marker, word, drows)
		return -1
	}
	if i := find("@@ -12,7", "Divide"); !regexp.MustCompile(`^\S`).MatchString(drows[i+1]) {
		t.Errorf("the @@ continuation is %q, want it flush", drows[i+1])
	}
	for _, c := range []struct{ marker, word string }{{"+", "divideChecked"}, {"-", "legacyDivide"}} {
		i := find(c.marker, c.word)
		if i+1 >= len(drows) || !regexp.MustCompile(`^ \S`).MatchString(drows[i+1]) {
			t.Errorf("the %q line's next row is %q, want a blank marker column then code", c.marker, drows[i+1])
		}
	}

	// G: the last page, full.
	gm := captureModel(t, capState(t, "modal-after-G"), false)
	total, bodyH := gm.modalExtent(gm.layout())
	if total <= bodyH || gm.modal.Off != total-bodyH {
		t.Errorf("after G: offset %d with %d rows in a %d-row body, want the offset at total-body", gm.modal.Off, total, bodyH)
	}
	grows := bodyText(t, mustBox(gm))
	if len(grows) != bodyH || !strings.HasPrefix(grows[len(grows)-1], fmt.Sprintf("%4d ", total)) {
		t.Errorf("after G: %d body rows ending %q, want %d ending numbered %d", len(grows), grows[len(grows)-1], bodyH, total)
	}

	// help: every visible row is a prefix of the help line it shows, and at least
	// one is shorter than its line (cut). A wrapped line would show its tail on the
	// next row, so the rows would not be prefixes of consecutive help lines.
	hm := captureModel(t, capState(t, "help-cut"), false)
	help := helpLines()
	hrows := bodyText(t, mustBox(hm))
	cut := 0
	for i, r := range hrows {
		src := strings.TrimRight(help[hm.modal.Off+i], " ")
		if !strings.HasPrefix(src, strings.TrimSuffix(r, "⋯")) {
			t.Errorf("help row %d is %q, not a prefix of line %q", i, r, src)
		}
		if cellWidth(r) < cellWidth(src) {
			cut++
		}
	}
	if cut == 0 {
		t.Errorf("no help row is shorter than its source line at this size: %q", hrows)
	}
}
