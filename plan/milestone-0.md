# Milestone 0 — Instruction Set

> **Status: Ready for execution — NOT started.** This is the complete,
> ordered instruction set for the builder of Milestone 0 (repo bootstrap +
> static TUI prototype). Execute tasks in order. Nothing in this milestone
> touches an LLM, the filesystem (agent-side), or a shell.

## Ground Rules (read first)

1. **One milestone at a time.** Do not build anything from Milestones 1–5.
   In particular: **no** file-reading tools, **no** subprocess execution,
   **no** provider/agent/session/policy/config/telemetry packages.
2. **The UI spec is law.** [`plan/ui-spec-v0.1.md`](ui-spec-v0.1.md) defines
   layout, cards, modals, keys, and edge cases. Where this instruction set
   and the spec seem to conflict, the spec wins — flag the conflict, don't
   guess.
3. **Locked decisions** (do not re-litigate):
   - Module path: `github.com/djm56/kirsch`
   - License: MIT (already in repo)
   - UI stack: Bubble Tea + bubbles + lipgloss + `go-runewidth` (ADR 0001)
   - Everything needed to build Kirsch lives in `plan/`. `doc/` (not `docs/`)
     is reserved for end-user documentation and stays empty until Milestone 5
   - Dark theme only; `Shift+Enter` newline with `Alt+Enter` fallback
   - Token display: 1 decimal, k/M units (`12.4k tok`)
4. Every task below has a checkable result. Do not move to the next task with
   the previous one failing.

---

## Task 1 — Go module bootstrap

1. `go mod init github.com/djm56/kirsch`
2. Add dependencies:
   - `github.com/charmbracelet/bubbletea`
   - `github.com/charmbracelet/bubbles` (textarea, viewport, spinner)
   - `github.com/charmbracelet/lipgloss`
   - `github.com/mattn/go-runewidth`
3. Create `cmd/kirsch/main.go` — a minimal `tea.NewProgram(...)` entry that
   runs the TUI model and exits cleanly.

**Check:** `go build ./...` and `go run ./cmd/kirsch` compile and open an
empty (or placeholder) full-screen TUI without panic.

## Task 2 — Repo housekeeping files

1. `AGENTS.md` — short guide for coding agents working on this repo: point to
   `plan/kirsch-plan.md` (spec), `plan/README.md` (progress + builder rules),
   the hard dependency rules in `plan/layout.md`, and the
   one-milestone-at-a-time rule. State that `doc/` is for end users and must
   not be used for build notes.
2. `CHANGELOG.md` — Keep a Changelog format, `## [Unreleased]` section with
   a "Planning" entry noting docs + spec + ADRs; Milestone 0 entry added when
   complete.
3. `README.md` — already exists; update the planning-phase banner to reflect
   Milestone 0 status when it starts, and remove the "no code exists" line
   once the prototype lands.
4. Promote `plan/layout.md` → `plan/architecture.md` (rename + expand with a
   short prose write-up; the package table and dependency rules carry over,
   plus the package **build order** table from plan §2).
   Update links in `README.md`, `plan/README.md`, and the ADRs.

**Check:** all markdown links resolve; no stale references to
`plan/layout.md` (except in git history); `doc/` still contains only its
placeholder.

## Task 3 — CI (GitHub Actions)

`.github/workflows/ci.yml`, triggered on push + PR to `main`:

1. Job `lint` (ubuntu-latest): checkout, setup-go (use `go.mod` version),
   `gofmt -l` (fail if output non-empty), `go vet ./...`,
   `golangci-lint run` (pin the version in the workflow).
2. Job `test` (ubuntu-latest): `go test ./...`.

**Check:** CI passes on a commit containing only whitespace changes (i.e.,
green on trivial input); `gofmt -l .` is empty locally.

## Task 4 — TUI package skeleton

Create `internal/tui/` with this structure (all fake data; no imports outside
bubbletea/bubbles/lipgloss/runewidth + stdlib):

```
internal/tui/
  model.go        root tea.Model: state, focus, mode (normal/modal), resize
  update.go       key handling per ui-spec §5, focus transfer, mode switching
  view.go         full-screen render per ui-spec §2, zero-width guard
  styles.go       lipgloss styles, dark palette, NO_COLOR fallback
  transcript.go   transcript state + card selection + expand/collapse
  cards.go        rendering: user msg, assistant, tool card, approval, error
  modal.go        diff modal + help overlay per ui-spec §4
  statusbar.go    status bar: model, state+spinner, tokens, warnings
  composer.go     textarea wrapper: growth clamp 1–5 lines, placeholder,
                  Shift+Enter/Alt+Enter newline, disabled-while-busy dim
  fake.go         fake transcript data per ui-spec §8 (2 user msgs, streaming
                  simulation, 3 tool cards incl. truncated+errored, 1
                  approval card, 1 error card), timer-driven stream ticks
```

Notes:

- `main.go` runs the TUI directly. `internal/app` is **not** created in this
  milestone (nothing to wire yet); it arrives with agent integration in
  Milestone 3.
- Streaming simulation: a `tea.Tick` appends text to the assistant message,
  exercising the same render path real deltas will use.
- Spinner states (`thinking`, `running go test`, `applying patch`) cycle on a
  timer so the status bar's busy path is exercised.

**Check:** all files exist, `go vet` clean, no TODOs that hide unimplemented
spec behavior.

## Task 5 — Interaction wiring (per ui-spec)

Implement and verify each behavior:

1. Composer focus; `Enter` sends (appends a user message to transcript,
   starts a fake streaming turn); `Shift+Enter`/`Alt+Enter` newline.
2. Focus transfer: `↑` at composer top / `↓` at transcript bottom moves
   focus; visible selection cursor on cards; `↑`/`↓` move selection;
   `PgUp`/`PgDn` scroll; `Enter` expands/collapses the selected card.
3. Approval card: `y` → resolves to approved (card collapses to `✓ approved`);
   `a` → approved for session (only rendered on the fake `run_command` card,
   never on the fake `apply_patch` card — plan §4); `n` → rejected; `d` →
   diff modal. Exclusive input capture while pending.
4. Diff modal: canned unified diff, colour-coded (`+`/`-`/`@@`), no-color
   fallback, `j`/`k`/arrows scroll, `Esc` closes.
5. Help overlay `?` (bindings + slash commands), `Esc` closes.
6. Quit semantics: idle `q` with empty composer quits; `Ctrl+C` cancels the
   in-flight fake turn (spinner stops, composer re-enables); double `Ctrl+C`
   within 1s force-quits.
7. Auto-scroll with `↓ new content` indicator when scrolled up (ui-spec §2).
8. Slash commands: `/help` and `/quit` functional; others render as system
   notices (fake content is fine — behavior shape must match ui-spec §6);
   unknown `/command` → dim inline hint.

**Check:** each numbered item demonstrated by a test or manual script (see
Task 6).

## Task 6 — Tests

1. **Golden-file snapshots of `View()`**: at least — initial screen, mid-turn
   (spinner + partial assistant text), approval pending, diff modal open,
   help overlay open, narrow width (40 cols), NO_COLOR mode.
2. **Synthetic key tests**: drive `Update()` with `tea.KeyMsg` sequences —
   send a message; expand a card; approve (`y`); reject (`n`); open/close
   modal; cancel turn; force-quit path.
3. **Resize test**: `tea.WindowSizeMsg` down to 0×0 and back — no panic,
   no corruption.
4. Paste handling: bracketed-paste `tea.PasteMsg` inserts literal multiline
   text (ui-spec §7).

**Check:** `go test ./...` green; golden files reviewed by a human once.

## Task 7 — Final acceptance (from plan §Milestone 0)

Run the full checklist:

- [ ] `go run ./cmd/kirsch` opens full-screen with header/transcript/status/
      composer per ui-spec §2
- [ ] Typing, sending, fake streaming all render correctly
- [ ] Card selection, expand/collapse work
- [ ] Approval `y`/`a`/`n`/`d` flow works end to end, with `a` absent on the
      patch card
- [ ] Diff modal + help overlay open/close cleanly
- [ ] Resize (large ↔ small ↔ 0×0) never panics or corrupts
- [ ] Quit paths (`q`, double `Ctrl+C`) work
- [ ] NO_COLOR fallback renders sensibly
- [ ] `gofmt`/`go vet`/`golangci-lint`/`go test ./...` all clean; CI green
- [ ] **No LLM, file, or shell code exists anywhere in the repo**

**Definition of done:** every box checked, CHANGELOG updated, README banner
updated, and a commit (or PR) tagged so Milestone 1 can start from a known
point.

---

## Explicitly Out of Scope (reminder)

No workspace/path code, no tools, no policy, no provider (not even the
interface), no session store, no config parsing, no telemetry, no
`internal/app`. If a task seems to need any of these, stop — it belongs to a
later milestone; fake the data instead.
