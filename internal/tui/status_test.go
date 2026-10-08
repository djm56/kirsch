package tui

import (
	"strings"
	"testing"
)

// flatten collapses whitespace and line breaks so tests can assert on phrases
// that may have been wrapped across rows.
func flatten(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestStatusCommandRendersAllFields verifies /status builds a notice containing
// endpoint, base_url host, key source, proxy host, context file/size, branch,
// and the shared ADR 0008 data-flow notice for opencode.
func TestStatusCommandRendersAllFields(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
		Status:  Status{Model: "claude-sonnet-5-5", Tokens: 42},
	})
	m.StatusInfo = func() StatusInfoMsg {
		return StatusInfoMsg{
			Endpoint:           "opencode",
			BaseURLHost:        "opencode.ai",
			KeySource:          "KIRSCH_OPENCODE_API_KEY",
			ProxyHost:          "proxy.example",
			ContextFile:        "ctx.md",
			ContextSize:        123,
			ShowDataFlowNotice: true,
		}
	}

	out, _ := m.runSlash("status", "", Layout{TermW: 120, ContentW: 116, Pad: 2, H: 30, ShowTranscript: true, TranscriptH: 20})
	mm := out.(Model)
	text := flatten(strings.Join(mm.plainLines, "\n"))

	want := []string{
		"model claude-sonnet-5-5",
		"endpoint opencode",
		"base_url opencode.ai",
		"key source KIRSCH_OPENCODE_API_KEY",
		"proxy proxy.example",
		"context ctx.md 123 bytes",
		"branch main",
		DataFlowNotice,
	}
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("/status output missing %q\n%s", w, text)
		}
	}
}

// TestStatusCommandOmitsProxyWhenAbsent verifies /status does not mention a
// proxy when no proxy host is reported.
func TestStatusCommandOmitsProxyWhenAbsent(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
		Status:  Status{Model: "claude-sonnet-5-5"},
	})
	m.StatusInfo = func() StatusInfoMsg {
		return StatusInfoMsg{
			Endpoint:    "anthropic",
			BaseURLHost: "api.anthropic.com",
			KeySource:   "ANTHROPIC_API_KEY",
		}
	}

	out, _ := m.runSlash("status", "", Layout{TermW: 120, ContentW: 116, Pad: 2, H: 30, ShowTranscript: true, TranscriptH: 20})
	mm := out.(Model)
	text := flatten(strings.Join(mm.plainLines, "\n"))

	if strings.Contains(text, "proxy") {
		t.Errorf("/status output mentions proxy when none is configured\n%s", text)
	}
}

// TestStatusCommandOmitsDataFlowNoticeForNonOpencode verifies the ADR 0008
// notice is absent for endpoints other than opencode.
func TestStatusCommandOmitsDataFlowNoticeForNonOpencode(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
		Status:  Status{Model: "claude-sonnet-5-5"},
	})
	m.StatusInfo = func() StatusInfoMsg {
		return StatusInfoMsg{
			Endpoint:    "anthropic",
			BaseURLHost: "api.anthropic.com",
			KeySource:   "ANTHROPIC_API_KEY",
		}
	}

	out, _ := m.runSlash("status", "", Layout{TermW: 120, ContentW: 116, Pad: 2, H: 30, ShowTranscript: true, TranscriptH: 20})
	mm := out.(Model)
	text := flatten(strings.Join(mm.plainLines, "\n"))

	if strings.Contains(text, DataFlowNotice) {
		t.Errorf("non-opencode /status output contains data-flow notice\n%s", text)
	}
}

// TestStatusCommandShowsCleanContextWhenNoneLoaded verifies /status renders the
// "context none" state when no project context file was chosen.
func TestStatusCommandShowsCleanContextWhenNoneLoaded(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
		Status:  Status{Model: "claude-sonnet-5-5"},
	})
	m.StatusInfo = func() StatusInfoMsg {
		return StatusInfoMsg{
			Endpoint:    "opencode",
			BaseURLHost: "opencode.ai",
			KeySource:   "KIRSCH_OPENCODE_API_KEY",
		}
	}

	out, _ := m.runSlash("status", "", Layout{TermW: 120, ContentW: 116, Pad: 2, H: 30, ShowTranscript: true, TranscriptH: 20})
	mm := out.(Model)
	text := flatten(strings.Join(mm.plainLines, "\n"))

	if !strings.Contains(text, "context none") {
		t.Errorf("/status output missing 'context none'\n%s", text)
	}
}
