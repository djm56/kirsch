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

	// A §2.2-shaped band, so it reads the terminal width like every other one.
	// On ContentW it would have moved when the frame gained its margin, and a
	// 40-column terminal — the advertised minimum, where screen 00 puts the
	// wordmark — would have lost it to two columns of padding.
	//
	// Pinned by TestEmptyStateWordmarkBandReadsTerminalWidth. It needs its own
	// test rather than riding on the golden grids: computeLayout's §2.2 switch
	// is covered by them because every band boundary changes a grid, but this
	// band is one comparison in a renderer, and no golden sits at a width where
	// TermW and ContentW disagree about it. The discriminating widths are 40
	// and 41 — the only two where TermW >= 40 and ContentW >= 40 differ.
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
		// Bound to the content width, like every other row the package emits.
		// The version is a build-time value — "0.1.0-dev" in the shipped
		// binary, and longer again at a release candidate — so this row's
		// width is not a constant and cannot be eyeballed against the 38
		// content columns a 40-column terminal leaves. Unbounded it reached
		// View's final truncEnd instead, which carries an empty marker because
		// its job is the frame's hard edge rather than an elision, so the row
		// read as a word broken off mid-air.
		//
		// A bare truncEnd rather than a fitWidest ladder, unlike the modal
		// footer: nothing at the tail of this row is load-bearing. That footer
		// needed a ladder because the span it lost first was the one naming the
		// key that closes the modal. Here the version leads and the prose after
		// it is decoration, so cutting from the right drops the least.
		tagline := "v" + m.sess.Version + " " + m.gly.Bullet + " terminal-native coding agent"
		add(m.sty.Dim(truncEnd(tagline, lay.ContentW, m.gly.Trunc)))
		add("")
	}

	// The lead-in and suggestions survive at sizes where the wordmark does not:
	// they are the part that tells a new user what to do.
	if h-len(out) >= 5 {
		add("")
		add(m.sty.Muted("Ask anything. Three things to try:"))
		add("")
		for _, s := range Suggestions {
			add(m.sty.Dim(m.gly.Bullet + " " + truncEnd(s, lay.ContentW-2, m.gly.Trunc)))
		}
	}

	for len(out) < h {
		out = append(out, "")
	}
	return out[:h]
}
