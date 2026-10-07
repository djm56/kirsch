package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestOnboardingStateMsgUpdatesModel verifies that the app-to-TUI onboarding
// message installs the state the onboarding renderer consumes. The message
// carries only plain fields; the TUI never touches provider or config tables.
func TestOnboardingStateMsgUpdatesModel(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})

	next, _ := m.Update(OnboardingStateMsg{
		NoAPIKey:     true,
		KeyVars:      [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
		NotGitRepo:   false,
		UnknownModel: true,
		Endpoint:     "opencode",
	})

	mm := next.(Model)
	if mm.onboarding == nil {
		t.Fatal("onboarding state was not set")
	}
	if !mm.onboarding.NoAPIKey {
		t.Errorf("NoAPIKey = %v, want true", mm.onboarding.NoAPIKey)
	}
	wantVars := [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"}
	if mm.onboarding.KeyVars != wantVars {
		t.Errorf("KeyVars = %v, want %v", mm.onboarding.KeyVars, wantVars)
	}
	if mm.onboarding.NotGitRepo {
		t.Errorf("NotGitRepo = true, want false")
	}
	if !mm.onboarding.UnknownModel {
		t.Errorf("UnknownModel = %v, want true", mm.onboarding.UnknownModel)
	}
	if mm.onboarding.Endpoint != "opencode" {
		t.Errorf("Endpoint = %q, want opencode", mm.onboarding.Endpoint)
	}
}

// TestOnboardingStateMsgOverwritesExistingState verifies that later onboarding
// messages replace earlier ones rather than accumulating. This is the normal
// Update pattern for value messages.
func TestOnboardingStateMsgOverwritesExistingState(t *testing.T) {
	m := New(Options{
		Version: "0.1.0",
		Caps:    Caps{Colour: false, Unicode: true},
	})

	apply := func(msg tea.Msg) Model {
		next, _ := m.Update(msg)
		m = next.(Model)
		return m
	}

	apply(OnboardingStateMsg{
		NoAPIKey:     true,
		KeyVars:      [2]string{"KIRSCH_OLD_KEY", "OLD_KEY"},
		UnknownModel: false,
		Endpoint:     "anthropic",
	})
	apply(OnboardingStateMsg{
		NoAPIKey:     false,
		KeyVars:      [2]string{"KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"},
		UnknownModel: true,
		Endpoint:     "opencode",
	})

	if m.onboarding.NoAPIKey {
		t.Errorf("NoAPIKey = true, want false after overwrite")
	}
	if !m.onboarding.UnknownModel {
		t.Errorf("UnknownModel = false, want true after overwrite")
	}
	if m.onboarding.Endpoint != "opencode" {
		t.Errorf("Endpoint = %q, want opencode", m.onboarding.Endpoint)
	}
}
