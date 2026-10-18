package tui

import (
	"fmt"
	"strings"
	"time"
)

// Status is the status bar's data. ui-spec §2.3.
type Status struct {
	Model           string   // e.g. "claude-sonnet-5-5"
	Family          string   // precomputed for the truncation ladder; derived from Model if empty
	Tokens          int      // last call's total tokens; shown only while no session totals exist (SessionTokens* all zero)
	EstimatedTokens int      // chars/4 estimate for the pending request; 0 when no turn is active
	ContextWindow   int      // model max context in tokens; 0 means unknown
	Warnings        []string // persistent conditions, never transient errors

	// Session totals from the status bar's perspective (wide layout).
	SessionTokensIn     int
	SessionTokensOut    int
	SessionTokensCacheR int // cache read
	SessionTokensCacheW int // cache write
	SessionCostUSD      float64
	SessionCostState    string // "unknown", "flat", "per_token"
}

// outputReserve is the headroom reserved for the model's response.
// plan §6.3.
const outputReserve = 4096

// statusRow renders exactly one line.
//
// The truncation ladder follows ui-spec §2.3: token count → grants → model
// family → bare spinner. The token count itself narrows before it goes:
//
//	wide layout, session totals present:
//	  1. model, state, "in X out Y cache R/W", cost, grants
//	  2. model, state, "in X out Y cache R/W", grants
//	  3. model, state, session total "N tok", grants
//	wide layout, no session totals: model, state, last call's "N tok", grants
//	then, at every width:
//	  no tokens, grants → no tokens, no grants → family only → bare spinner
//
// Narrower than the wide layout no token figure is shown. The per-call figure
// (Status.Tokens) is never a fallback once session totals exist.
//
// Warnings are never dropped: at narrow widths they are the entire reason the
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

	budgetPlain := ""
	if m.status.EstimatedTokens > 0 && m.status.ContextWindow > outputReserve {
		pct := m.status.EstimatedTokens * 100 / (m.status.ContextWindow - outputReserve)
		budgetPlain = fmt.Sprintf("context %d%% full", pct)
	}

	rightPlain := warnPlain
	rightStyled := warn
	if budgetPlain != "" {
		if rightPlain != "" {
			rightPlain += "  " + budgetPlain
			rightStyled += "  " + m.sty.Warning(budgetPlain)
		} else {
			rightPlain = budgetPlain
			rightStyled = m.sty.Warning(budgetPlain)
		}
	}

	// Build the left segment at descending detail until it fits.
	leftBudget := lay.ContentW - cellWidth(rightPlain)
	if rightPlain != "" {
		leftBudget-- // at least one space between segments
	}

	type variant struct{ plain, styled string }

	// Session totals exist once any usage has been reported for this session.
	hasSessionTokens := m.status.SessionTokensIn > 0 || m.status.SessionTokensOut > 0 ||
		m.status.SessionTokensCacheR > 0 || m.status.SessionTokensCacheW > 0

	// tokenForm selects which token figure a variant carries.
	type tokenForm int
	const (
		tokensNone       tokenForm = iota
		tokensLastCall             // Status.Tokens, the pre-totals figure
		tokensSessionSum           // session total, "N tok"
		tokensDetail               // "in X out Y cache R/W"
		tokensDetailCost           // tokensDetail plus the cost
	)

	build := func(model string, form tokenForm, grants bool) variant {
		p := model
		s := m.sty.Muted(model)
		if spinner != "" {
			p += dot + spinner + " " + state
			s += m.sty.Muted(dot) + m.sty.Accent(spinner) + m.sty.Muted(" "+state)
		} else {
			p += dot + state
			s += m.sty.Muted(dot + state)
		}
		var t string
		switch form {
		case tokensNone:
			// No token figure on this rung.
		case tokensLastCall:
			t = formatTokens(m.status.Tokens)
		case tokensSessionSum:
			t = formatTokens(m.status.SessionTokensIn + m.status.SessionTokensOut +
				m.status.SessionTokensCacheR + m.status.SessionTokensCacheW)
		case tokensDetail, tokensDetailCost:
			t = m.formatSessionTokens()
		}
		if t != "" {
			p += dot + t
			s += m.sty.Muted(dot + t)
		}
		if form == tokensDetailCost {
			if cost := m.formatCost(); cost != "" {
				p += dot + cost
				s += m.sty.Muted(dot) + cost
			}
		}
		if grants && m.GetGrantCount != nil {
			count := m.GetGrantCount()
			if count > 0 {
				g := fmt.Sprintf("%d grant%s", count, plural(count))
				p += dot + g
				s += m.sty.Muted(dot + g)
			}
		}
		return variant{p, s}
	}

	candidates := []variant{}
	if lay.ShowTokens {
		if hasSessionTokens {
			candidates = append(candidates,
				build(m.status.Model, tokensDetailCost, true),
				build(m.status.Model, tokensDetail, true),
				build(m.status.Model, tokensSessionSum, true),
			)
		} else {
			candidates = append(candidates, build(m.status.Model, tokensLastCall, true))
		}
	}
	candidates = append(candidates,
		build(m.status.Model, tokensNone, true),
		build(m.status.Model, tokensNone, false),
		build(m.status.Family, tokensNone, false),
	)

	if spinner != "" {
		// Last resort: the bare spinner, no verb.
		candidates = append(candidates, variant{
			plain:  spinner,
			styled: m.sty.Accent(spinner),
		})
	}

	chosen := candidates[len(candidates)-1]
	for _, c := range candidates {
		if cellWidth(c.plain) <= leftBudget {
			chosen = c
			break
		}
	}
	if rightStyled == "" {
		return chosen.styled
	}
	gap := lay.ContentW - cellWidth(chosen.plain) - cellWidth(rightPlain)
	if gap < 1 {
		gap = 1
	}
	return chosen.styled + strings.Repeat(" ", gap) + rightStyled
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

// formatSessionTokens renders session token totals as "in X out Y cache Z/W" using k/M units.
func (m Model) formatSessionTokens() string {
	in := formatShortTokens(m.status.SessionTokensIn)
	out := formatShortTokens(m.status.SessionTokensOut)
	cacheR := formatShortTokens(m.status.SessionTokensCacheR)
	cacheW := formatShortTokens(m.status.SessionTokensCacheW)

	return fmt.Sprintf("in %s out %s cache %s/%s", in, out, cacheR, cacheW)
}

// formatShortTokens renders a token count with one decimal and k/M units, without "tok" suffix.
// Used for the detailed session token breakdown.
func formatShortTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// formatCost renders the cost component for the status bar.
// Returns empty string if cost should not be shown.
func (m Model) formatCost() string {
	switch m.status.SessionCostState {
	case "per_token":
		if m.status.SessionCostUSD < 0.01 {
			return fmt.Sprintf("$%.4f", m.status.SessionCostUSD)
		}
		return fmt.Sprintf("$%.2f", m.status.SessionCostUSD)
	case "flat":
		return "" // Don't show cost for flat-rate models
	case "unknown":
		return "$?"
	default:
		return ""
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
