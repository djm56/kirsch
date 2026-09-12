package tui

// The empty-session state: the wordmark, the tagline and three example
// prompts. ui-spec §7.5 and screens 00 and 01.
//
// This is onboarding, not an error — no border, no red. It is shown only while
// the transcript is empty; it scrolls away with the first message and never
// comes back.

// wordmarkUTF8 is two rows of half-blocks, 21 cells wide.
var wordmarkUTF8 = []string{
	"█▄▀ █ █▀█ █▀▀ █▀▀ █ █",
	"█▀▄ █ █▀▄ ▄▄█ █▄▄ █▀█",
}

// wordmarkASCII is the non-UTF-8 fallback: letter-spaced plain text.
var wordmarkASCII = []string{"K I R S C H"}

// Suggestions are the three example prompts on the empty session. §7.5.
var Suggestions = []string{
	"What does the approval flow do when a patch conflicts?",
	"Add input validation to Divide and cover it with a test",
	"Where is the session lock taken?",
}

// emptyStateRows renders the onboarding screen into the transcript region.
//
// Returned top-aligned and padded to exactly h rows, because the empty state
// reads from the top while a populated transcript is pinned to the bottom.
//
// The wordmark is suppressed below 40 columns or 10 rows, where the transcript
// needs the lines more than the branding does (screen 00).
func (m Model) emptyStateRows(lay Layout) []string {
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

	showWordmark := lay.W >= 40 && lay.H >= 10
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
		add(m.sty.Dim("v" + m.sess.Version + " " + m.gly.Bullet + " terminal-native coding agent"))
		add("")
	}

	// The lead-in and suggestions survive at sizes where the wordmark does not:
	// they are the part that tells a new user what to do.
	if h-len(out) >= 5 {
		add("")
		add(m.sty.Muted("Ask anything. Three things to try:"))
		add("")
		for _, s := range Suggestions {
			add(m.sty.Dim(m.gly.Bullet + " " + truncEnd(s, lay.W-2, "…")))
		}
	}

	for len(out) < h {
		out = append(out, "")
	}
	return out[:h]
}
