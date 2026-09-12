package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
)

// Placeholder is the composer's empty-state text. ui-spec §2.
const Placeholder = "Ask anything (Enter to send, /help for help)"

// Composer wraps the textarea with Kirsch's own prompt and hint rows.
type Composer struct {
	ta       textarea.Model
	Disabled bool   // mirrors Busy; the only thing Busy does here
	Hint     string // dim inline hint for an unknown /command, §6
}

func newComposer() Composer {
	ta := textarea.New()
	ta.Placeholder = Placeholder
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.Focus()
	return Composer{ta: ta}
}

// Value returns the composer's contents.
func (c *Composer) Value() string { return c.ta.Value() }

// SetValue replaces the composer's contents.
func (c *Composer) SetValue(s string) { c.ta.SetValue(s) }

// Reset clears the composer and any pending hint.
func (c *Composer) Reset() {
	c.ta.Reset()
	c.Hint = ""
}

// InsertString inserts literal text at the cursor. Used for bracketed paste,
// where the content must never be interpreted as keys. ui-spec §7.2.
func (c *Composer) InsertString(s string) { c.ta.InsertString(s) }

// contentLines is the composer's height in rows: its content clamped to 1..5,
// plus a row for the hint when one is showing.
//
// The hint occupies a row *of* the composer region rather than becoming a fifth
// region, because row order is always header, transcript, status, composer.
func (c *Composer) contentLines() int {
	n := clamp(strings.Count(c.ta.Value(), "\n")+1, 1, 5)
	if c.Hint != "" {
		n++
	}
	return n
}

// rows renders the composer to exactly lay.ComposerH lines.
func (c *Composer) rows(lay Layout, sty *Styles, g Glyphs, busy bool) []string {
	h := lay.ComposerH
	out := make([]string, 0, h)

	prompt := sty.Accent(">")
	if busy {
		prompt = sty.Dim(">")
	}

	value := c.ta.Value()
	var lines []string
	if value == "" {
		lines = []string{sty.Dim(Placeholder)}
	} else {
		lines = strings.Split(value, "\n")
		for i := range lines {
			lines[i] = sty.Text(lines[i])
		}
	}

	// First row carries the prompt; when a turn is live the right edge carries
	// the disabled hint instead of the caret.
	first := prompt + " " + lines[0]
	if busy {
		hint := "(input disabled, Esc cancels)"
		plain := "> " + stripFirst(value)
		first = prompt + " " + sty.Dim(stripFirst(value))
		if cellWidth(plain)+cellWidth(hint)+1 <= lay.W {
			gap := lay.W - cellWidth(plain) - cellWidth(hint)
			first += strings.Repeat(" ", gap) + sty.Dim(hint)
		}
	}
	out = append(out, first)
	for _, l := range lines[1:] {
		if len(out) >= h {
			break
		}
		out = append(out, "  "+l)
	}
	if c.Hint != "" && len(out) < h {
		out = append(out, sty.Dim("  "+c.Hint))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out[:h]
}

func stripFirst(v string) string {
	if v == "" {
		return ""
	}
	return strings.SplitN(v, "\n", 2)[0]
}

// parseSlash recognises a slash command.
//
// The grammar is deliberately strict: a single line beginning with `/`.
// Multi-line content starting with `/` is an ordinary message, so a pasted
// diff or stack trace is never mistaken for a command. ui-spec §6.
func parseSlash(v string) (cmd, args string, ok bool) {
	if !strings.HasPrefix(v, "/") || strings.Contains(v, "\n") {
		return "", "", false
	}
	body := strings.TrimSpace(v[1:])
	if body == "" {
		return "", "", false
	}
	parts := strings.SplitN(body, " ", 2)
	if len(parts) == 2 {
		return parts[0], strings.TrimSpace(parts[1]), true
	}
	return parts[0], "", true
}

// SlashCommands is the v0.1 command set. ui-spec §6.
var SlashCommands = []string{
	"help", "status", "diff", "files", "approvals", "new", "compact", "quit",
}

// completeSlash completes a unique prefix, returning the completion and whether
// exactly one candidate matched.
func completeSlash(prefix string) (string, bool) {
	var match string
	n := 0
	for _, c := range SlashCommands {
		if strings.HasPrefix(c, prefix) {
			match = c
			n++
		}
	}
	return match, n == 1
}
