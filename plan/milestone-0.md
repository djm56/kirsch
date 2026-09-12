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
   layout, breakpoints, cards, modals, the mode state machine, keys per mode,
   text sanitisation, palette, timing constants, and accessibility rules.
   Where this instruction set and the spec seem to conflict, the spec wins —
   flag the conflict, don't guess. Read §5.1 (mode state machine) and §10
   (palette and glyphs) before writing any code.
3. **The screens are the render target.**
   [`plan/kirsch-ui-screens.md`](kirsch-ui-screens.md) draws twelve states
   (00–11) as literal 80-column character grids, each with a colour map naming
   the token, 256 index and hex for every span. Do not design the layout — it
   is already designed. Build to those grids, and capture the Task 6 golden
   files against them rather than against whatever your first pass renders; a
   golden file recorded from an unreviewed render just freezes the bug.
   Structure is the contract, colour is applied on top: line counts and box
   positions must be identical with and without colour. If a screen and the
   ui-spec disagree, the spec wins and the screen is the bug — flag it.
4. **Locked decisions** (do not re-litigate):
   - Module path: `github.com/djm56/kirsch`
   - License: MIT (already in repo)
   - UI stack: Bubble Tea + bubbles + lipgloss + `go-runewidth` (ADR 0001)
   - Everything needed to build Kirsch lives in `plan/`. `doc/` (not `docs/`)
     is reserved for end-user documentation and stays empty until Milestone 5
   - Dark theme only; `Alt+Enter` newline with `Ctrl+J` fallback (`Shift+Enter`
     is not representable on Bubble Tea v1 — see ui-spec §5.2)
   - Token display: 1 decimal, k/M units (`12.4k tok`)
5. Every task below has a checkable result. Do not move to the next task with
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
   the hard dependency rules in `plan/architecture.md`, and the
   one-milestone-at-a-time rule. State that `doc/` is for end users and must
   not be used for build notes.
2. `CHANGELOG.md` — Keep a Changelog format, `## [Unreleased]` section with
   a "Planning" entry noting docs + spec + ADRs; Milestone 0 entry added when
   complete.
3. `README.md` — already exists; update the planning-phase banner to reflect
   Milestone 0 status when it starts, and remove the "no code exists" line
   once the prototype lands.
4. `plan/architecture.md` already exists and is settled — **do not rewrite
   it.** Read it before Task 4; it defines the dependency rules, the mode the
   TUI must implement, and the concurrency constraints. At the end of the
   milestone, review it against what you actually built and correct any drift
   in a separate commit, flagging anything that contradicts it rather than
   silently diverging.

**Check:** all markdown links resolve; `doc/` still contains only its
placeholder; any architecture drift is either fixed in code or recorded in
`plan/architecture.md`.

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
                  Alt+Enter / Ctrl+J newline, disabled-while-busy dim
  fake.go         fake transcript data per ui-spec §8 (2 user msgs, streaming
                  simulation, 3 tool cards incl. truncated+errored, two
                  approval cards — apply_patch without [a], run_command with
                  it — 1 error card, 1 system notice), timer-driven stream
                  ticks. Use the literal strings from kirsch-ui-screens.md
                  (calc/divide.go, `go test ./...`, the same timings and
                  token counts) so the golden files match the screens
```

Notes:

- `main.go` runs the TUI directly. `internal/app` is **not** created in this
  milestone (nothing to wire yet); it arrives in **Milestone 1**, the moment the
  TUI drives a real tool — see the plan §2 build-order table and amendment 2.
- Streaming simulation: a `tea.Tick` appends text to the assistant message,
  exercising the same render path real deltas will use.
- Spinner states (`thinking`, `running go test`, `applying patch`) cycle on a
  timer so the status bar's busy path is exercised.

**Check:** all files exist, `go vet` clean, no TODOs that hide unimplemented
spec behavior.

## Task 5 — Interaction wiring (per ui-spec)

Build the **mode state machine of ui-spec §5.1 first**, then hang behaviour off
it. Modes bolted onto ad-hoc key handling is how TUIs end up with keys that
work in some states and silently vanish in others.

1. Modes `Composing` / `Browsing` / `ApprovalPending` / `Modal` / `Confirm`,
   with `Busy` as an orthogonal flag, and the precedence order
   `Modal > ApprovalPending > Confirm > Browsing/Composing`.
2. Composer: `Enter` sends (appends a user message, starts a fake streaming
   turn); `Alt+Enter`/`Ctrl+J` newline; `Tab` completes a unique slash
   prefix. **`?` is a literal character here, not the help key** (ui-spec
   §5.1) — get this right in M0 or it is a bug report later.
3. Focus transfer: `↑` at composer top / `↓` past the last card; accent `┃`
   gutter on the selected card; `↑`/`↓` move selection; `PgUp`/`PgDn` scroll
   **without** moving selection; `Enter` expands/collapses.
4. Card expansion honours the **200-line inline cap** with the
   `‹200 of N lines — press d for full output›` marker; `d` opens the content
   modal with everything (ui-spec §3.3, §4.1).
5. Approval cards: `y` approve; `a` approve-for-session — rendered on the fake
   `run_command` card only, never on the fake `apply_patch` card (plan §4);
   `n` reject; `d` detail modal. Exclusive capture while pending, **except**
   that an approval arriving while a modal is open defers capture until the
   modal closes (ui-spec §5.1).
6. Content/diff modal: canned unified diff, colour-coded, no-color fallback,
   `j`/`k`/arrows scroll, `g`/`G`, `Esc` closes **to the previous mode** — from
   an approval card that means back to the pending approval, not the composer.
7. Help overlay via `?` (Browsing/ApprovalPending) and `/help` (anywhere).
8. Confirm prompt (ui-spec §4.3) wired for `/new` mid-turn. Quitting never
   confirms.
9. Scroll and pin per ui-spec §2.4: pinned by default; scrolling up unpins and
   shows `↓ n new`; re-pin on `End`, `Esc`, bottom, or send. **Typing must not
   re-pin.**
10. Quit semantics: idle `q` with empty composer quits; `Ctrl+C` cancels the
    in-flight fake turn (spinner stops, composer re-enables); double `Ctrl+C`
    within 1s force-quits.
11. Slash commands: `/help` and `/quit` functional; others render as system
    notices (fake content fine — the *shape* must match ui-spec §6); unknown
    `/command` → dim inline hint, never an error card.
12. Text sanitisation (ui-spec §7.1) on the fake tool output: strip ANSI, keep
    only the last `\r` segment, expand tabs to 4, cap lines at 2,000 columns.
    Seed one fake tool card with escape sequences and a `\r` progress bar so
    this path is actually exercised.
13. ASCII glyph fallback (ui-spec §10.2) when the locale is not UTF-8.

**Check:** each numbered item demonstrated by a test or manual script (Task 6).
Every §10 colour is referenced through a semantic name in `styles.go` — no raw
colour numbers anywhere else.

## Task 6 — Tests

1. **Golden-file snapshots of `View()`** — all fourteen states enumerated in
   ui-spec §13. Do not invent a shorter list; that appendix exists so the
   coverage question is already answered. Thirteen of the fourteen are drawn
   as character grids in
   [`kirsch-ui-screens.md`](kirsch-ui-screens.md) (the §13 table maps state →
   screen); compare against those grids before recording a golden file.
   **State 14 (onboarding — no API key, not a Git repo) has no screen and is
   not reachable in M0** — there is no provider until M3. Capture the other
   thirteen now and leave 14 to M3, which draws its screen first. Screen 11 is
   the `NO_COLOR` fixture: strip SGR sequences from `View()` and it must match
   byte-for-byte, which is also the test that line counts do not change
   between the coloured and uncoloured paths.
2. **Synthetic key tests**: drive `Update()` with `tea.KeyMsg` sequences —
   send a message; expand a card past the 200-line cap; approve (`y`);
   approve-for-session (`a`); reject (`n`); open/close modal and confirm it
   returns to the *previous* mode; cancel turn; force-quit path; `?` typed in
   the composer inserts a literal `?`.
3. **Resize test**: `tea.WindowSizeMsg` down to 0×0 and back, plus each §2.2
   breakpoint boundary — no panic, no corruption.
4. **Paste handling**: bracketed paste inserts literal multiline text, never
   interpreted as keys; an 8KB+ paste warns first (ui-spec §7.2). On Bubble Tea
   v1 this arrives as `tea.KeyMsg{Type: KeyRunes, Paste: true}` — there is no
   `tea.PasteMsg`, which is a v2 type.
5. **Sanitisation**: a fake tool result containing ANSI escapes, a `\r`
   progress bar, tabs, a NUL byte, and a 5,000-column line renders correctly.
   Assert on the *rendered* string, since this is the test that catches terminal
   corruption — but assert the right thing. "No escape sequence in `View()`" is
   false whenever colour is on, because Kirsch emits its own SGR. The two
   checkable claims (ui-spec §13) are: **zero** `0x1b` bytes under `NO_COLOR`,
   and under colour, every escape is an SGR drawn from the §10.1 palette. The
   second doubles as the automated form of "no raw colour numbers outside
   `styles.go`".
6. **Scroll/pin**: scrolling up unpins and shows the indicator; typing does
   not re-pin; `End` does.

**Check:** `go test ./...` green; golden files reviewed by a human once.

## Task 7 — Final acceptance (from plan §Milestone 0)

Run the full checklist. Every box below was verified by a test or a
capture from the real binary, not by inspection:

- [x] `go run ./cmd/kirsch` opens full-screen with header/transcript/status/
      composer per ui-spec §2
- [x] Typing, sending, fake streaming all render correctly
- [x] Mode state machine implemented with ui-spec §5.1 precedence; an approval
      arriving during an open modal defers capture
- [x] `?` in the composer inserts a literal `?`; `/help` opens help from
      anywhere
- [x] Card selection (`┃` gutter), expand/collapse, and the 200-line inline cap
      with `d` → full content modal
- [x] `PgUp`/`PgDn` scroll without moving selection
- [x] Approval `y`/`a`/`n`/`d` flow works end to end, with `a` absent on the
      patch card
- [x] Modal `Esc` returns to the *previous* mode, not always the composer
- [x] Confirm prompt wired for `/new` mid-turn; quit never confirms
- [x] Scroll/pin per §2.4 — typing does not re-pin
- [x] Text sanitisation (§7.1) proven: zero `0x1b` bytes under `NO_COLOR`, and
      under colour only §10.1 palette SGR reaches `View()` (ui-spec §13)
- [x] Resize (large ↔ small ↔ 0×0, plus every §2.2 breakpoint) never panics
      or corrupts
- [x] Quit paths (`q`, double `Ctrl+C`) work
- [x] NO_COLOR and ASCII-glyph fallbacks both render sensibly
- [x] Rendered output matches every reachable grid in
      [`kirsch-ui-screens.md`](kirsch-ui-screens.md) (screens 00–11), colour
      maps included — spot-checked at 80 cols against each screen
- [x] Thirteen of the fourteen ui-spec §13 golden states captured from those
      screens and human-reviewed; state 14 (onboarding) deferred to M3 with
      its screen, and the deferral noted in the PR rather than silently skipped
- [x] Stripped (`NO_COLOR`) output matches screen 11 byte-for-byte, proving
      line counts are identical with and without colour
- [x] No raw colour numbers outside `styles.go`
- [x] `gofmt`/`go vet`/`golangci-lint`/`go test ./...` all clean; CI green
- [x] **No LLM, file, or shell code exists anywhere in the repo**

**Definition of done:** every box checked, CHANGELOG updated, README banner
updated, and a commit (or PR) tagged so Milestone 1 can start from a known
point.

**Completed 2026-09-12.** 43 tests green under `-race`; all thirteen character
grids in `plan/kirsch-ui-screens.md` verified byte-for-byte against `View()` on
every run. Golden state 14 (onboarding — no API key, not a Git repo) remains
deferred to M3 with its screen, per amendment 24: there is no provider to be
missing a key for until then. Findings from execution are recorded as plan
amendments 33–39 — most usefully, screen 01 was never implemented and only
surfaced when the binary was run, which is the argument for the checkpoint.

---

## Explicitly Out of Scope (reminder)

No workspace/path code, no tools, no policy, no provider (not even the
interface), no session store, no config parsing, no telemetry, no
`internal/app`. If a task seems to need any of these, stop — it belongs to a
later milestone; fake the data instead.
