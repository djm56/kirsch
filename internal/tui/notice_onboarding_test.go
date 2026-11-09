package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Session start-up sends notices before the user types (cmd/kirsch/main.go:
// "no previous session — starting fresh" on a fresh machine). A notice is not
// something said in the session, so it must not end the onboarding or the
// empty-state screen. These tests drive the model through Update and read the
// rendered frame (View), which is the entry point the terminal sees.

const noKeyLine = "No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY"

// noticeModel returns a 120x40 model, optionally with the no-key onboarding
// installed, that has received one NoticeMsg.
func noticeModel(withOnboarding bool, notice string) Model {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	msgs := []tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 40}}
	if withOnboarding {
		msgs = append(msgs, OnboardingStateMsg{
			NoAPIKey: true,
			KeyVars:  [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
			Endpoint: "opencode",
		})
	}
	msgs = append(msgs, NoticeMsg{Text: notice})
	return drive(m, msgs...)
}

// TestNoticeOnlyTranscriptKeepsOnboarding: a notice-only transcript with the
// no-key onboarding shows the key line and the notice text.
func TestNoticeOnlyTranscriptKeepsOnboarding(t *testing.T) {
	m := noticeModel(true, "no previous session — starting fresh")
	frame := stripSGR(m.View())
	if !strings.Contains(frame, noKeyLine) {
		t.Errorf("frame lacks the onboarding key line %q:\n%s", noKeyLine, frame)
	}
	if !strings.Contains(frame, "no previous session — starting fresh") {
		t.Errorf("frame lacks the notice text:\n%s", frame)
	}
}

// TestNoticeOnlyTranscriptKeepsEmptyState: the same with no onboarding installed
// shows the empty-state screen and the notice.
func TestNoticeOnlyTranscriptKeepsEmptyState(t *testing.T) {
	m := noticeModel(false, "no previous session — starting fresh")
	frame := stripSGR(m.View())
	if !strings.Contains(frame, "Ask anything. Three things to try:") {
		t.Errorf("frame lacks the empty-state lead-in:\n%s", frame)
	}
	if !strings.Contains(frame, "no previous session — starting fresh") {
		t.Errorf("frame lacks the notice text:\n%s", frame)
	}
}

// TestFirstNonNoticeItemEndsOnboarding: a notice followed by a user item shows
// neither the onboarding nor the empty state; the notice stays in the
// transcript. Each case first checks that its screen's text is in the frame
// before the user item (the positive control), so the absence check after it
// runs on a model that did draw that screen. The user item is appended through
// appendBlock and relayout, the two calls the Update handlers make, because the
// real submit path needs a send callback.
func TestFirstNonNoticeItemEndsOnboarding(t *testing.T) {
	cases := []struct {
		name           string
		withOnboarding bool
		screenText     string
	}{
		{"onboarding", true, "No API key"},
		{"empty state", false, "Three things to try"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := noticeModel(tc.withOnboarding, "no previous session — starting fresh")
			if before := stripSGR(m.View()); !strings.Contains(before, tc.screenText) {
				t.Fatalf("control: %q not in the frame before the user item:\n%s", tc.screenText, before)
			}
			m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"hello"}}})
			m.relayout(m.layout())
			frame := stripSGR(m.View())
			if strings.Contains(frame, tc.screenText) {
				t.Errorf("%q still showing after a user item:\n%s", tc.screenText, frame)
			}
			if !strings.Contains(frame, "hello") || !strings.Contains(frame, "no previous session") {
				t.Errorf("frame lacks the user item or the notice:\n%s", frame)
			}
		})
	}
}

// TestNothingSaidYetByKind: each item kind other than a notice ends the
// onboarding on its own, and notices alone do not. Calls nothingSaidYet, the
// rule shared by transcriptRows and View.
func TestNothingSaidYetByKind(t *testing.T) {
	notice := Item{Kind: KindNotice, Notice: &NoticeCard{Text: "n"}}
	others := map[string]Item{
		"user":      {Kind: KindUser, Text: &TextBlock{}},
		"assistant": {Kind: KindAssistant, Text: &TextBlock{}},
		"tool":      {Kind: KindTool, Tool: &ToolCard{}},
		"approval":  {Kind: KindApproval, Approval: &ApprovalCard{}},
		"error":     {Kind: KindError, Err: &ErrorCard{}},
		"thinking":  {Kind: KindThinking, Thinking: &ThinkingCard{}},
	}
	m := New(Options{Version: "0.1.0"})
	if !m.nothingSaidYet() {
		t.Error("empty transcript: nothingSaidYet = false, want true")
	}
	m.appendBlock(notice)
	m.appendBlock(notice)
	if !m.nothingSaidYet() {
		t.Error("notices only: nothingSaidYet = false, want true")
	}
	for name, it := range others {
		m := New(Options{Version: "0.1.0"})
		m.appendBlock(notice)
		m.appendBlock(it)
		if m.nothingSaidYet() {
			t.Errorf("notice then %s: nothingSaidYet = true, want false", name)
		}
	}
}

// TestComposerPlaceholderFollowsTheSameRule: comp.Onboarding (the composer's
// placeholder) is on for a notice-only transcript and off after a user item.
func TestComposerPlaceholderFollowsTheSameRule(t *testing.T) {
	m := noticeModel(true, "no previous session — starting fresh")
	if !strings.Contains(stripSGR(m.View()), Placeholder) {
		t.Errorf("notice-only transcript: composer lacks the placeholder %q", Placeholder)
	}
	m.appendBlock(Item{Kind: KindUser, Text: &TextBlock{Lines: []string{"hello"}}})
	m.relayout(m.layout())
	if strings.Contains(stripSGR(m.View()), Placeholder) {
		t.Errorf("after a user item: composer still shows the placeholder %q", Placeholder)
	}
}

// noKeyModel80 returns an 80x24 model with the default no-key opencode
// onboarding and the start-up notice a fresh machine gets.
func noKeyModel80() Model {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	return drive(m,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		OnboardingStateMsg{
			NoAPIKey: true,
			KeyVars:  [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
			Endpoint: "opencode",
		},
		NoticeMsg{Text: "no previous session — starting fresh"},
	)
}

// runStatus types /status and Enter into m, through Update, with a StatusInfo
// of the length a real run produces (opencode, so the data-flow notice is
// appended).
func runStatus(m Model) Model {
	m.StatusInfo = func() StatusInfoMsg {
		return StatusInfoMsg{
			Endpoint:           "opencode",
			BaseURLHost:        "opencode.ai",
			KeySource:          "none",
			ShowDataFlowNotice: true,
			SessionID:          "20261105-101500-abcdef",
			CompactionPercent:  -1,
		}
	}
	for _, r := range "/status" {
		m = drive(m, key(r))
	}
	return drive(m, keyType(tea.KeyEnter))
}

// TestStartupNoticeAt80x24ShowsKeyLineAndNotice: at the default 80x24, the
// no-key onboarding and the start-up notice are both on screen, and the
// composer shows its placeholder.
func TestStartupNoticeAt80x24ShowsKeyLineAndNotice(t *testing.T) {
	m := noKeyModel80()
	frame := stripSGR(m.View())
	if !strings.Contains(frame, noKeyLine) {
		t.Errorf("frame lacks the key line:\n%s", frame)
	}
	if !strings.Contains(frame, "no previous session — starting fresh") {
		t.Errorf("frame lacks the start-up notice:\n%s", frame)
	}
	if !strings.Contains(frame, Placeholder) {
		t.Errorf("composer lacks the placeholder %q while the onboarding shows:\n%s", Placeholder, frame)
	}
	if got := len(strings.Split(frame, "\n")); got != 24 {
		t.Errorf("frame has %d rows, want 24", got)
	}
}

// TestStatusNoticeWhoseRowsDoNotFitFallsBack: at 80x24 the start-up notice plus
// a /status notice of real length (typed through Update) do not fit under the
// onboarding. The frame is then the normal transcript: no onboarding key line,
// no placeholder (the composer follows the same answer), and every notice row
// whole.
func TestStatusNoticeWhoseRowsDoNotFitFallsBack(t *testing.T) {
	m := runStatus(noKeyModel80())
	frame := stripSGR(m.View())
	if strings.Contains(frame, "No API key") {
		t.Errorf("onboarding still showing although the notices do not fit:\n%s", frame)
	}
	if strings.Contains(frame, Placeholder) {
		t.Errorf("composer shows the placeholder while the transcript region shows no onboarding:\n%s", frame)
	}
	rows := 0
	for _, l := range m.plainLines {
		l = strings.TrimRight(l, " ")
		if l == "" {
			continue
		}
		rows++
		if !strings.Contains(frame, l) {
			t.Errorf("notice row cut from the frame: %q\n%s", l, frame)
		}
	}
	if rows < 4 {
		t.Errorf("only %d notice rows; the /status text is meant to wrap to several", rows)
	}
	if !strings.Contains(frame, "no previous session — starting fresh") {
		t.Errorf("frame lacks the start-up notice:\n%s", frame)
	}
	if got := len(strings.Split(frame, "\n")); got != 24 {
		t.Errorf("frame has %d rows, want 24", got)
	}
}

// TestManyNoticesFallBackToTranscript: thirty notices cannot fit under the
// onboarding. The normal transcript is shown, scrollable, with the newest
// notice visible; the frame stays the window height.
func TestManyNoticesFallBackToTranscript(t *testing.T) {
	m := noKeyModel80()
	for i := 0; i < 30; i++ {
		m = drive(m, NoticeMsg{Text: "notice number " + itoa(i)})
	}
	frame := stripSGR(m.View())
	if strings.Contains(frame, "No API key") {
		t.Errorf("onboarding showing although the notices do not fit:\n%s", frame)
	}
	if !strings.Contains(frame, "notice number 29") {
		t.Errorf("newest notice not visible:\n%s", frame)
	}
	if got := len(strings.Split(frame, "\n")); got != 24 {
		t.Errorf("frame has %d rows, want 24", got)
	}
}

// fitModel returns a model of width 80 and the given height with the no-key
// onboarding installed and k notices, sent through Update. Each notice is one
// short line, so each takes one transcript row (consecutive one-line notices
// get no blank between them; relayout's grouping).
func fitModel(height, k int) Model {
	m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
	msgs := []tea.Msg{
		tea.WindowSizeMsg{Width: 80, Height: height},
		OnboardingStateMsg{
			NoAPIKey: true,
			KeyVars:  [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
			Endpoint: "opencode",
		},
	}
	for i := 0; i < k; i++ {
		msgs = append(msgs, NoticeMsg{Text: "fit notice " + itoa(i)})
	}
	return drive(m, msgs...)
}

// TestIntroFitBoundary pins both sides of the fit rule in introRows: the
// onboarding rows, one separator and the notice rows filling the transcript
// region exactly keep the onboarding; one notice row more falls back to the
// normal transcript. k is chosen from the onboarding's own row count at this
// size, and the exact-fit case then checks independently, on the rendered
// frame, that the newest notice sits on the region's last row, so a k that is
// one off fails here rather than testing the wrong side.
func TestIntroFitBoundary(t *testing.T) {
	const height = 40
	probe := fitModel(height, 0)
	lay := probe.layout()
	top := probe.onboardingRows(lay)
	used := len(top)
	for used > 0 && top[used-1] == "" {
		used--
	}
	k := lay.TranscriptH - used - 1
	if k < 2 {
		t.Fatalf("setup: only %d notice rows fit at 80x%d (region %d, onboarding %d)", k, height, lay.TranscriptH, used)
	}
	lastRow := lay.HeaderH() + lay.TranscriptH - 1 // frame row of the region's last row

	t.Run("exact fit keeps the onboarding", func(t *testing.T) {
		m := fitModel(height, k)
		if got := len(m.lines) - 2; got != k {
			t.Fatalf("setup: %d notices made %d transcript rows, want %d", k, got, k)
		}
		frame := stripSGR(m.View())
		rows := strings.Split(frame, "\n")
		if !strings.Contains(frame, noKeyLine) {
			t.Errorf("onboarding key line missing at exact fit:\n%s", frame)
		}
		if !strings.Contains(frame, Placeholder) {
			t.Errorf("placeholder missing at exact fit:\n%s", frame)
		}
		newest := "fit notice " + itoa(k-1)
		if lastRow >= len(rows) || !strings.Contains(rows[lastRow], newest) {
			t.Errorf("newest notice %q not on the region's last row (frame row %d):\n%s", newest, lastRow, frame)
		}
	})

	t.Run("one row over falls back", func(t *testing.T) {
		m := fitModel(height, k+1)
		if got := len(m.lines) - 2; got != k+1 {
			t.Fatalf("setup: %d notices made %d transcript rows, want %d", k+1, got, k+1)
		}
		frame := stripSGR(m.View())
		if strings.Contains(frame, "No API key") {
			t.Errorf("onboarding still showing one row over the fit:\n%s", frame)
		}
		if strings.Contains(frame, Placeholder) {
			t.Errorf("placeholder still showing one row over the fit:\n%s", frame)
		}
		if newest := "fit notice " + itoa(k); !strings.Contains(frame, newest) {
			t.Errorf("newest notice %q not visible after the fall-back:\n%s", newest, frame)
		}
	})
}

// TestIntroOffWhenRegionCannotShowIt: when the transcript region draws no
// transcript (below 40 columns it draws "terminal too narrow"; at a height
// whose chrome takes every row it has none), introRows says no, so the
// composer placeholder is off. The positive control renders the same composer
// at the same layout with the placeholder forced on and finds it, so the
// absence is the rule's doing and not a composer too narrow to show the text.
func TestIntroOffWhenRegionCannotShowIt(t *testing.T) {
	const shown = "Ask anything" // the placeholder's head, which survives the cut at 30 columns
	cases := []struct {
		name          string
		width, height int
		notice        bool
		narrowLine    bool // "terminal too narrow" expected in the frame
	}{
		{"narrow with a notice", 30, 24, true, true},
		{"narrow and empty", 30, 24, false, true},
		{"no transcript rows and empty", 80, 2, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New(Options{Version: "0.1.0", Caps: Caps{Colour: false, Unicode: true}})
			msgs := []tea.Msg{
				tea.WindowSizeMsg{Width: tc.width, Height: tc.height},
				OnboardingStateMsg{
					NoAPIKey: true,
					KeyVars:  [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
					Endpoint: "opencode",
				},
			}
			if tc.notice {
				msgs = append(msgs, NoticeMsg{Text: "no previous session — starting fresh"})
			}
			m = drive(m, msgs...)
			lay := m.layout()

			c := m.comp
			c.Onboarding = true
			if ctl := stripSGR(strings.Join(c.rows(lay, m.sty, m.gly, false), "\n")); !strings.Contains(ctl, shown) {
				t.Fatalf("control: composer with the placeholder forced on lacks %q at %dx%d: %q", shown, tc.width, tc.height, ctl)
			}

			if m.showIntro(lay) {
				t.Errorf("showIntro = true at %dx%d (ShowTranscript=%v, TranscriptH=%d)", tc.width, tc.height, lay.ShowTranscript, lay.TranscriptH)
			}
			frame := stripSGR(m.View())
			if strings.Contains(frame, shown) {
				t.Errorf("placeholder shown while the region cannot show the onboarding:\n%s", frame)
			}
			if tc.narrowLine && !strings.Contains(frame, "terminal too narrow") {
				t.Errorf("frame lacks the narrow-terminal line:\n%s", frame)
			}
		})
	}
}
