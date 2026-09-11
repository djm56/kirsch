# ADR 0001 — Go + Bubble Tea for the TUI

- **Status:** Accepted (planning phase; no code yet)
- **Date:** 2026-09-11
- **Deciders:** Project owner

## Context

Kirsch needs a full-screen terminal UI: streaming text, scrollable
transcript, modals, multiline composer, spinner, resize safety. Options
considered:

1. **Web UI (Electron / local server + browser)** — richest rendering, but
   heavy, distribution-unfriendly, and contradicts "terminal-native, local
   first."
2. **TUI in another language** (Python/Textual, Rust/ratatui) — capable, but
   the project contract locks Go for the agent core; a second language adds
   build complexity and FFI/process boundaries.
3. **Go + Bubble Tea** (`bubbletea` + `bubbles` + `lipgloss`) — single
   language, Elm-architecture message loop that matches our
   agent↔TUI message-passing model, mature ecosystem.

## Decision

Build the TUI in Go using **Bubble Tea**, with `bubbles` (textarea, viewport,
spinner) and `lipgloss` for styling. All width math is rune-aware via
`go-runewidth`.

## Consequences

**Positive**

- One toolchain, one binary, easy cross-compile (Milestone 5 targets).
- Bubble Tea's `Cmd`/`Msg` model maps directly onto our rule that agent
  goroutines update the TUI only via `program.Send(msg)`.
- `Update()` can be driven by synthetic `tea.KeyMsg` in tests → golden-file
  snapshot testing of `View()`.

**Negative / risks**

- Some terminals don't send distinct Shift+Enter; we accept this and map
  Alt+Enter as fallback (see ui-spec §9).
- No mouse support in v0.1 (not required by spec).
- Colour/rendering edge cases across terminals; mitigated by NO_COLOR
  fallback and resize guards (ui-spec §7).
