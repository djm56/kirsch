package tui

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

// This file is the only place in the package that may name a colour. Every
// other file reaches colour through a semantic Style. ui-spec §10.1.

// Timing constants. ui-spec §11.
const (
	SpinnerFrame    = 100 * time.Millisecond // 10fps reads as smooth, costs little
	StreamCoalesce  = 50 * time.Millisecond  // or on newline, whichever comes first
	CmdOutCoalesce  = 100 * time.Millisecond // output is chunkier than tokens
	PasteDebounce   = 20 * time.Millisecond  // one update per paste
	DoubleInterrupt = time.Second            // double Ctrl+C window
)

// PasteWarnBytes is the paste size above which the user is asked first. §7.2.
const PasteWarnBytes = 8 << 10

// InlineExpandCap is the maximum number of lines a card expands to inline
// before the `d` modal takes over. ui-spec §3.3.
const InlineExpandCap = 200

// MaxRenderedLineWidth caps a single rendered line. ui-spec §7.1.
const MaxRenderedLineWidth = 2000

// Caps is what the terminal can do. It is resolved once, at construction, and
// injected — nothing in the render path reads the environment, so the fallback
// states are reachable in a test without touching process-global state.
type Caps struct {
	Colour  bool
	Unicode bool
}

// DetectCaps resolves terminal capabilities from the environment. It is called
// once, from main, and never from inside the package. ui-spec §7.3.
func DetectCaps(out *os.File) Caps {
	colour := true
	if _, set := os.LookupEnv("NO_COLOR"); set {
		colour = false
	}
	if os.Getenv("TERM") == "dumb" || os.Getenv("TERM") == "" {
		colour = false
	}
	if out != nil {
		if fi, err := out.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
			colour = false // not a TTY: piped or redirected
		}
	}
	return Caps{Colour: colour, Unicode: unicodeLocale()}
}

func unicodeLocale() bool {
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return strings.Contains(strings.ToUpper(v), "UTF-8") ||
				strings.Contains(strings.ToUpper(v), "UTF8")
		}
	}
	return false
}

// Style paints one fragment. A Style is only ever applied to text containing no
// newline, so it cannot change the line count — which is what makes "layout is
// identical with and without colour" (§12) a structural property rather than a
// convention.
type Style func(string) string

func identity(s string) string { return s }

// Styles is the semantic palette. ui-spec §10.1.
type Styles struct {
	Text        Style // 253 — body text, tool output, action labels
	Muted       Style // 248 — metadata between · delimiters
	Dim         Style // 245 — placeholders, suggestions, backgrounded cells
	Accent      Style // 117 — header title, tool glyphs, selection gutter
	Success     Style // 120 — ✓, [y], [a], diff additions
	Error       Style // 203 — ✗, [n], diff deletions, error card
	Warning     Style // 215 — dirty marker, approval required, cap marker
	Hunk        Style // 123 — diff @@ headers
	Border      Style // 244 — separators and unfocused borders
	BorderFocus Style // 117 — focused modal border
	Bold        Style // tool names, and nothing else
	CodeBg      Style // 235 — fenced tool-output panels
	SelectionBg Style // 236 — selected card interior
}

// PaletteSGR lists every SGR parameter string the package is allowed to emit.
// The golden tests assert that nothing outside this set reaches View(), which
// is the automated form of "no raw colour numbers outside styles.go".
var PaletteSGR = map[string]bool{
	"0": true, "": true, "1": true,
	"38;5;253": true, "38;5;248": true, "38;5;245": true, "38;5;117": true,
	"38;5;120": true, "38;5;203": true, "38;5;215": true, "38;5;123": true,
	"38;5;244": true, "48;5;235": true, "48;5;236": true,
}

// NewRenderer builds a renderer with an explicit colour profile.
//
// This must never fall back to lipgloss's package-level default: that renderer
// is built from os.Stdout, which is not a TTY under `go test`, so termenv
// resolves it to Ascii and silently strips every colour. A "coloured" golden
// captured that way is byte-identical to the plain one and the test passes
// forever while proving nothing.
func NewRenderer(colour bool) *lipgloss.Renderer {
	profile := termenv.Ascii
	if colour {
		profile = termenv.ANSI256
	}
	r := lipgloss.NewRenderer(io.Discard, termenv.WithProfile(profile))
	// SetColorProfile is required, not belt-and-braces: without it lipgloss
	// ignores the termenv option and falls back to EnvColorProfile(), which
	// inspects the writer — and the writer here is io.Discard, which is not a
	// TTY, so every style would silently resolve to Ascii and emit nothing.
	r.SetColorProfile(profile)
	r.SetHasDarkBackground(true) // never probe: HasDarkBackground writes OSC 11 and waits
	return r
}

// NewStyles resolves the palette. With colour off every Style is the identity
// function, so not a single escape byte is emitted.
func NewStyles(r *lipgloss.Renderer, colour bool) *Styles {
	if !colour {
		return &Styles{
			Text: identity, Muted: identity, Dim: identity, Accent: identity,
			Success: identity, Error: identity, Warning: identity, Hunk: identity,
			Border: identity, BorderFocus: identity, Bold: identity,
			CodeBg: identity, SelectionBg: identity,
		}
	}
	fg := func(n string) Style {
		st := r.NewStyle().Foreground(lipgloss.Color(n))
		return func(x string) string { return st.Render(x) }
	}
	bg := func(n string) Style {
		st := r.NewStyle().Background(lipgloss.Color(n))
		return func(x string) string { return st.Render(x) }
	}
	bold := r.NewStyle().Bold(true)
	// Every foreground token clears 3:1 against a dark ground, and every token
	// carrying words clears 4.5:1. The first palette did not: dim at 240 was
	// 2.3:1 and border at 238 was 1.7:1, which made placeholders, the
	// onboarding suggestions and every separator rule read as washed-out grey
	// on a real dark terminal. ui-spec §10.1.
	return &Styles{
		Text:        fg("253"),
		Muted:       fg("248"),
		Dim:         fg("245"),
		Accent:      fg("117"),
		Success:     fg("120"),
		Error:       fg("203"),
		Warning:     fg("215"),
		Hunk:        fg("123"),
		Border:      fg("244"),
		BorderFocus: fg("117"),
		Bold:        func(x string) string { return bold.Render(x) },
		CodeBg:      bg("235"),
		SelectionBg: bg("236"),
	}
}

// Flat collapses every role to dim. Used for the transcript behind an open
// modal (screens 05 and 06): a different style set over the same structure, so
// the line count is untouched.
func (s *Styles) Flat() *Styles {
	d := s.Dim
	return &Styles{
		Text: d, Muted: d, Dim: d, Accent: d, Success: d, Error: d,
		Warning: d, Hunk: d, Border: d, BorderFocus: d, Bold: d,
		CodeBg: identity, SelectionBg: identity,
	}
}

// Glyphs is the glyph table with its ASCII fallback. ui-spec §10.2.
//
// Several fallbacks are multi-cell strings ("[ok]" is four), so every width
// calculation must go through cellWidth and never assume 1.
type Glyphs struct {
	Collapsed string // ▸ / >
	Expanded  string // ▾ / v
	Running   string // ◐ / *  — the card badge, not the status-bar spinner
	Pending   string // ◌ / .
	OK        string // ✓ / [ok]
	Err       string // ✗ / [err]
	Cancelled string // ⊘ / [canc]
	Trunc     string // ⋯ / ...
	Dirty     string // ● / *
	Warn      string // ⚠ / !
	New       string // ↓ / v
	Caret     string // ▌ / _
	CaretAlt  string // ▐ / #  — the caret's stand-in, see caretFor in composer.go
	Gutter    string // ┃ / |
	Bullet    string // · / .
	Sep       string // ─ / -  (header fill and separator rules)

	BoxTL, BoxTR, BoxBL, BoxBR string
	BoxH, BoxV, BoxLT, BoxRT   string

	Spinner []string // status-bar animation frames
}

var (
	spinnerUTF8  = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerASCII = []string{"-", "\\", "|", "/"}
)

// NewGlyphs returns the Unicode table, or the ASCII fallback when the locale is
// not UTF-8.
//
// CaretAlt is drawn only where the text under the cursor is already the caret
// glyph, so the two only have to differ from one another — nothing else about
// the pair is load-bearing. Both are one cell, which is what lets either of
// them be padded to the width of what it covers by the same rule.
func NewGlyphs(unicode bool) Glyphs {
	if !unicode {
		return Glyphs{
			Collapsed: ">", Expanded: "v", Running: "*", Pending: ".",
			OK: "[ok]", Err: "[err]", Cancelled: "[canc]", Trunc: "...",
			Dirty: "*", Warn: "!", New: "v", Caret: "_", CaretAlt: "#",
			Gutter: "|",
			Bullet: ".", Sep: "-",
			BoxTL: "+", BoxTR: "+", BoxBL: "+", BoxBR: "+",
			BoxH: "-", BoxV: "|", BoxLT: "+", BoxRT: "+",
			Spinner: spinnerASCII,
		}
	}
	return Glyphs{
		Collapsed: "▸", Expanded: "▾", Running: "◐", Pending: "◌",
		OK: "✓", Err: "✗", Cancelled: "⊘", Trunc: "⋯",
		Dirty: "●", Warn: "⚠", New: "↓", Caret: "▌", CaretAlt: "▐",
		Gutter: "┃",
		Bullet: "·", Sep: "─",
		BoxTL: "┌", BoxTR: "┐", BoxBL: "└", BoxBR: "┘",
		BoxH: "─", BoxV: "│", BoxLT: "├", BoxRT: "┤",
		Spinner: spinnerUTF8,
	}
}

// okLabel is "✓ ok" in Unicode and "[ok]" in ASCII — four cells either way,
// which is what lets screen 11 be a width-preserving twin of screen 03.
func (g Glyphs) okLabel() string {
	if g.OK == "[ok]" {
		return g.OK
	}
	return g.OK + " ok"
}

// asciiFold downgrades typographic characters that a non-UTF-8 terminal cannot
// display. The §10.2 table covers Kirsch's own glyphs; this covers characters
// that arrive inside content — an em dash in a card title, smart quotes in a
// commit message — which would otherwise render as mojibake on exactly the
// terminals the fallback exists to serve.
var asciiFold = strings.NewReplacer(
	"—", "-", "–", "-", "‘", "'", "’", "'", "“", `"`, "”", `"`,
	"…", "...", "·", ".", "‹", "<", "›", ">", "−", "-",
)

// Fold returns s with non-ASCII typography downgraded when the terminal cannot
// render it, and unchanged otherwise.
func (g Glyphs) Fold(s string) string {
	if g.BoxH != "-" { // unicode table in use
		return s
	}
	return asciiFold.Replace(s)
}

func init() {
	// East Asian ambiguous-width characters must measure as 1, matching what
	// the reference grids assume. Set once, package-wide.
	runewidth.DefaultCondition.EastAsianWidth = false
}
