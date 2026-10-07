package tui

// onboardingRows renders the provider-aware onboarding notices into the
// transcript region. It is the reachable-on-first-run counterpart to
// emptyStateRows: the session is still empty, but there are conditions the user
// needs to know about before they start typing. ui-spec §7.5, screen 12.
//
// Onboarding notices are rendered with dim/muted styling and no border, never
// via the error-card path, because a new user's first experience must not be a
// red border.
func (m Model) onboardingRows(lay Layout) []string {
	h := lay.TranscriptH
	if h <= 0 {
		return nil
	}

	out := make([]string, 0, h)
	add := func(s string) {
		if len(out) < h {
			out = append(out, s)
		}
	}

	// The wordmark and tagline are shared with the ordinary empty-session
	// screen; they scroll away once the user sends the first message.
	showWordmark := lay.TermW >= 40 && lay.H >= 10
	if showWordmark {
		mark := wordmarkUTF8
		if !m.caps.Unicode {
			mark = wordmarkASCII
		}
		add("")
		for _, row := range mark {
			add(m.sty.Accent(row))
		}
		add("")
		tagline := "v" + m.sess.Version + " " + m.gly.Bullet + " terminal-native coding agent"
		add(m.sty.Dim(truncEnd(tagline, lay.ContentW, m.gly.Trunc)))
		add("")
	}

	ob := m.onboarding
	if ob == nil {
		for len(out) < h {
			out = append(out, "")
		}
		return out[:h]
	}

	if ob.NoAPIKey {
		line := m.gly.Bullet + " No API key: set " + ob.KeyVars[0] + " → " + ob.KeyVars[1] + "."
		for _, w := range wrap(line, lay.ContentW) {
			add(m.sty.Dim(w))
		}
		add(m.sty.Dim(m.gly.Bullet + " Keys are never read from config files."))
		add("")
	}

	if ob.NotGitRepo {
		line := m.gly.Bullet + " Not a Git repository: run inside a repository, or use --workspace <dir>."
		for _, w := range wrap(line, lay.ContentW) {
			add(m.sty.Dim(w))
		}
		add("")
	}

	if ob.UnknownModel {
		line := m.gly.Bullet + " Unknown model: cost display unavailable; conservative budget in use."
		for _, w := range wrap(line, lay.ContentW) {
			add(m.sty.Dim(w))
		}
		add("")
	}

	if ob.Endpoint == "opencode" {
		line := m.gly.Bullet + " Using opencode sends prompts and file contents to OpenCode's gateway and the model host, not Anthropic."
		for _, w := range wrap(line, lay.ContentW) {
			add(m.sty.Dim(w))
		}
		add("")
	}

	for len(out) < h {
		out = append(out, "")
	}
	return out[:h]
}
