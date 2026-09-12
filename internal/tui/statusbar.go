package tui

import (
	"fmt"
	"strings"
	"time"
)

// Status is the status bar's data. ui-spec §2.3.
type Status struct {
	Model    string // "claude-sonnet-5"
	Family   string // "sonnet-5" — precomputed for the truncation ladder
	Tokens   int
	Grants   int
	Warnings []string // persistent conditions, never transient errors
}

// statusRow renders exactly one line.
//
// The truncation ladder is tokens → grants → model family → bare spinner, and
// warnings are never dropped: at narrow widths they are the entire reason the
// bar exists. ui-spec §2.3.
func (m Model) statusRow(lay Layout) string {
	dot := " " + m.gly.Bullet + " "

	state, stateHasSpinner := m.stateVerb()
	spinner := ""
	if stateHasSpinner {
		spinner = m.gly.Spinner[m.frame%len(m.gly.Spinner)]
	}

	warn := ""
	warnPlain := ""
	if len(m.status.Warnings) > 0 {
		w := m.gly.Warn + " " + strings.Join(m.status.Warnings, "  "+m.gly.Warn+" ")
		warn, warnPlain = m.sty.Warning(w), w
	}

	// Build the left segment at descending detail until it fits.
	budget := lay.W - cellWidth(warnPlain)
	if warnPlain != "" {
		budget-- // at least one space between segments
	}

	type variant struct{ plain, styled string }
	build := func(model string, tokens, grants bool) variant {
		p := model
		s := m.sty.Muted(model)
		if spinner != "" {
			p += dot + spinner + " " + state
			s += m.sty.Muted(dot) + m.sty.Accent(spinner) + m.sty.Muted(" "+state)
		} else {
			p += dot + state
			s += m.sty.Muted(dot + state)
		}
		if tokens {
			t := formatTokens(m.status.Tokens)
			p += dot + t
			s += m.sty.Muted(dot + t)
		}
		if grants && m.status.Grants > 0 {
			g := fmt.Sprintf("%d grant%s", m.status.Grants, plural(m.status.Grants))
			p += dot + g
			s += m.sty.Muted(dot + g)
		}
		return variant{p, s}
	}

	candidates := []variant{
		build(m.status.Model, lay.ShowTokens, true),
		build(m.status.Model, false, true),
		build(m.status.Model, false, false),
		build(m.status.Family, false, false),
	}
	if spinner != "" {
		// Last resort: the bare spinner, no verb.
		candidates = append(candidates, variant{
			plain:  spinner,
			styled: m.sty.Accent(spinner),
		})
	}

	chosen := candidates[len(candidates)-1]
	for _, c := range candidates {
		if cellWidth(c.plain) <= budget {
			chosen = c
			break
		}
	}
	if warn == "" {
		return chosen.styled
	}
	gap := lay.W - cellWidth(chosen.plain) - cellWidth(warnPlain)
	if gap < 1 {
		gap = 1
	}
	return chosen.styled + strings.Repeat(" ", gap) + warn
}

// stateVerb returns the status word and whether it animates.
//
// A pending approval is a waiting state, not an idle one: the turn is blocked
// on the user. Screens 03 and 04 both show the spinner running against
// "awaiting approval".
func (m Model) stateVerb() (string, bool) {
	if m.pendingApproval != 0 {
		return "awaiting approval", true
	}
	if !m.busy.Active {
		return "idle", false
	}
	return m.busy.Verb, true
}

// formatTokens renders a token count with one decimal and k/M units.
func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM tok", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk tok", float64(n)/1000)
	default:
		return fmt.Sprintf("%d tok", n)
	}
}

// formatDuration renders a tool card's elapsed time.
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
