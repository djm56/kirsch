# Kirsch — TUI Spec v0.1

> **Status: Settled for v0.1.** This is the source of truth for "the feel of the
> UI". Milestone 0 prototypes all of it with fake data; later milestones wire it
> to real events without changing the shape. Where an instruction set and this
> spec conflict, this spec wins — flag the conflict rather than guessing.
>
> Sections 1–9 keep their original numbering because the build plan and
> milestone docs reference them. Sections 10–14 are reference appendices:
> palette, glyphs, timing constants, accessibility rules, and the golden-test
> surface. Read §10 before writing any styling code.
>
> **Companion:** [`kirsch-ui-screens.md`](kirsch-ui-screens.md) draws every
> state described here as a literal 80-column character grid with a per-region
> colour map. This document says what the rules are; that one shows what they
> produce. Read both before rendering anything — and if they disagree, this
> document wins and the screen is the bug.

## 1. Design Goals

1. **Readable transcript first.** The conversation viewport is the star; every
   other element yields space to it.
2. **No surprises.** Consequential actions are visible as cards *before* they
   happen; nothing applies silently.
3. **Keyboard-only, vi-adjacent feel.** No mouse required; nothing needs more
   than two keystrokes.
4. **Never frozen.** Spinner for any in-flight work; every turn cancellable;
   a stuck tool never sticks the UI.
5. **Terminal-agnostic.** Works from 40×10 to huge; correct with no-color, no
   Unicode, soft wrap, and zero-width resize.
6. **The terminal's own background shows through.** Kirsch paints foreground
   colors and accents, never a full-screen background fill. A TUI that repaints
   the background looks foreign in every theme but the one it was built in.

## 2. Full-Screen Layout

```
┌─ Kirsch ─ my-project ─ main ● ─────────────────┐  ① header (1 line)
│                                                │
│  [transcript viewport — scrolls]               │  ② transcript (fills)
│                                                │
│  ── you ───────────────────────                │
│  Fix the Divide validation                     │
│                                                │
│  ▸ read_file calc/divide.go · 4ms · ok         │  tool card (collapsed)
│                                                │
│  I found the issue in... ▌                     │  streaming assistant
│                                                │
├────────────────────────────────────────────────┤
│ claude-sonnet-5 · ⠋ thinking · 12.4k tok       │  ③ status bar (1 line)
├────────────────────────────────────────────────┤
│ > _                                            │  ④ composer (1–5 lines)
└────────────────────────────────────────────────┘
```

- **① Header** — `Kirsch ─ <project name> ─ <branch>[ ● if dirty]`. Append
  ` (compacted)` when the session has been compacted. Project name is the
  workspace directory's base name.
- **② Transcript viewport** — all conversation history. Scroll and pin rules
  in §2.4.
- **③ Status bar** — §2.3.
- **④ Composer** — multiline text area, 1–5 lines then internal scroll.
  Placeholder `Ask anything (Enter to send, /help for help)` when empty.
  Dimmed and input-disabled while a turn is in flight; `Esc`/`Ctrl+C` cancels
  the turn and re-enables it.

### 2.1 Sizing rules

- Header and status bar are fixed at 1 line each. Composer is its content
  height clamped to 1–5. Transcript takes everything left over.
- All width math is rune-aware (`go-runewidth`). A rune is not a cell: CJK and
  emoji are width 2, combining marks are width 0. Assuming otherwise corrupts
  every border on the screen.
- `View()` guards zero and negative dimensions and renders nothing rather than
  panicking.

### 2.2 Responsive breakpoints

| Width | Behaviour |
|---|---|
| ≥ 80 | Full layout. Modals at 80% of viewport. |
| 60–79 | Status bar drops the token count. Modals at 90%. |
| 40–59 | Header shows project name only (no branch). Tool cards drop duration. Modals full-screen. |
| < 40 | Transcript hidden. Status bar + composer only, with a dim `terminal too narrow` notice. No panic. |

| Height | Behaviour |
|---|---|
| ≥ 10 | Full layout. |
| 6–9 | Header hidden; transcript shrinks to at least 2 lines. |
| 5 | Header hidden; transcript 1 line. |
| < 5 | Status bar + composer only; both separator rules dropped. |
| ≤ 0 either axis | Render empty string. |

The bands follow from the chrome budget, so change them together. Chrome is header (1) +
separator (1) + status (1) + separator (1) + composer (1) = **5 rows with a header, 4
without**. So h=10 leaves 5 transcript rows, h=6 leaves 2, and h=5 leaves 1 — which is why
the "at least 2 lines" band starts at 6, not 5. Degrade in this order, never another:
composer (never dropped, minimum 1) → status bar (never dropped) → the two separator rules
(dropped **as a pair**, so the bar is never half-framed) → header → transcript remainder.

### 2.3 Status bar

```
left   <model> · <state> · <tokens> [· <n grants>]
right  [⚠ warnings]
```

- **State** is `idle`, or a spinner plus a verb: `thinking`, `running go test`,
  `applying patch`, `compacting`, `cancelling`.
- **Tokens** use 1 decimal and k/M units (`12.4k tok`).
- **Warnings** are persistent conditions, not transient errors:
  `⚠ recovered session`, `⚠ another Kirsch is running here`,
  `⚠ no rg — using fallback search`.

**Truncation priority.** When the left segment does not fit, drop in this
order: token count → grants indicator → model name shortened to its family
(`sonnet-5`) → state verb reduced to the bare spinner. **Warnings are never
dropped**; at narrow widths they are the entire reason the bar exists.

### 2.4 Scroll and pinning

- The transcript is **pinned to the bottom** by default. New content while
  pinned keeps it pinned.
- Scrolling up **unpins**. A `↓ 3 new` indicator appears bottom-right of the
  viewport and counts blocks arrived since unpinning.
- Re-pin on any of: scrolling back to the bottom, `End` or `G` in Browsing
  mode, sending a message, or submitting a slash command.
- **A slash command re-pins.** Submitting one is a request, and §3.1 renders the
  invocation as a message, so the transcript goes to the bottom where its answer
  will be. The two hint-only paths are the exception — an unknown command, and a
  debug command called without its argument. Both answer entirely in the dim hint
  under the composer and put nothing in the transcript, so re-pinning would cost
  the reader their scroll position to report a typo.
- **`Esc` does not re-pin.** Leaving Browsing hands focus back to the composer and
  leaves the viewport exactly where it is. Re-pinning there would make the rule
  below unreachable in practice: `Esc` is the only way back to the composer that
  does not first walk the selection past the last card, so scrolling up to read
  something would be undone by the act of going to type about it.
- **Typing does not re-pin.** Composing a message while reading scrollback must
  not yank the view to the bottom — the user is reading it deliberately.
- `PgUp`/`PgDn` scroll without moving the card selection. `↑`/`↓` move the
  selection and scroll only as far as needed to keep it visible. Scrolling and
  selection are separate concerns and must not be collapsed into one.

## 3. Transcript Elements

### 3.1 User message

```
── you ─────────────────────────
Fix the Divide validation
```

Verbatim, soft-wrapped. Slash-command invocations render the same way; their
result appears as a system notice or a card.

### 3.2 Assistant message

Streams with a `▌` caret; caret removed on completion. Soft-wrapped. Triple-
backtick regions render with the code background (§10) and preserve indentation
exactly. **No syntax highlighting in v0.1** — it is a large dependency and an
open-ended rabbit hole; explicitly out of scope.

### 3.3 Tool card

**Collapsed (default):**

```
▸ read_file calc/divide.go · 4ms · ok
```

Glyph, bold name, dimmed target, result summary, duration. Tool-specific
summaries:

| Tool | Summary |
|---|---|
| `read_file` | path + line range |
| `list_files` | path + `n files` |
| `search_code` | query + `n matches` |
| `apply_patch` | `n files changed` + status |
| `run_command` | command + exit status |
| `git_status` / `git_diff` | changed-file count |

**Expanded (`Enter`):** full output inline, **capped at 200 rendered lines**
with a `‹200 of 4,181 lines — press d for full output›` marker. `d` opens the
content modal (§4.1) for everything. An unbounded inline expansion makes the
scrollback unusable, which is a worse failure than truncating it.

Truncation markers where caps were hit upstream: `‹truncated — 200KB cap›`,
`‹truncated — 4000 tok›`.

### 3.4 Approval card

```
┌─ approval required ─────────────────────────────┐
│ apply_patch — add input validation              │
│                                                 │
│ files: 2 changed (calc/divide.go,               │
│        calc/divide_test.go)                     │
│                                                 │
│ [y] approve   [n] reject   [d] view diff        │
└─────────────────────────────────────────────────┘
```

- For `run_command`: shows command, args, cwd, timeout, and why approval is
  needed ("not on allowlist"), and offers a fourth action:

  ```
  [y] approve   [a] approve for session   [n] reject   [d] detail
  ```

- `y` → approved; card collapses to `✓ approved`.
- `a` → approved **and** a session grant recorded for the argv prefix shown on
  the card (plan §4). Collapses to `✓ approved · session grant: go test`.
  **Never offered for `apply_patch`**, never for `sh -c`.
- `n` → rejected; returned to the model as a tool result so it can adapt.
- Only one approval is pending at a time; input capture is exclusive (§5.1).

### 3.5 Thinking card

```
▸ thinking · 412 tok
```

Collapsed and dimmed throughout so it never competes with assistant text.
Expands like a tool card. Default off in v0.1; render path exists from M3.

### 3.6 Error card

Red-bordered, with the error kind and a one-line detail; `Enter` expands.
Reserved for **infrastructure** errors (provider unreachable, config invalid,
session unwritable) and for tool errors the model could not recover from.
A tool error the model handles and works around stays a tool card with an `✗`
badge — promoting every recovered failure to an error card trains the user to
ignore error cards.

### 3.7 System notice

Single dim line, no border, for `/status`, `/files`, `/approvals`, `/compact`
output, session recovery points, and unknown-command hints.

```
· session recovered — 3 events after a torn line were discarded
```

### 3.8 Card lifecycle

Every card moves through: `pending` → `running` → one terminal state.

| State | Badge | Notes |
|---|---|---|
| pending | `◌` dim | Awaiting approval or queued |
| running | `◐` spinner | Spinner lives on the card, not only in the status bar |
| ok | `✓` success | |
| error | `✗` error | |
| cancelled | `⊘` dim | Turn cancelled mid-tool |
| truncated | `⋯` warning | Suffix on `ok`, not a state of its own |

A card never disappears or is rewritten in place once terminal. Partial
assistant text from a cancelled turn **stays** in the transcript, marked
cancelled — erasing what the user watched arrive is worse than leaving it.

### 3.9 Selection

Only **cards** are selectable: tool, approval, thinking, error, system notice.
User and assistant text blocks are skipped by `↑`/`↓`, because `Enter` does
nothing on them and stopping there is pure friction.

The selected card shows an accent-colored `┃` in the left gutter. Selection
survives new content arriving; it does not jump to the newest card.

## 4. Modals

Modals overlay the transcript, clamp to the §2.2 breakpoint sizes, and trap all
input. `Esc` closes and returns to the **previous mode** — closing a diff modal
opened from an approval card returns to the pending approval, not to the
composer (§5.1).

### 4.1 Content / diff modal (`d`)

One modal serves both jobs. Opened with `d` on any card, or from an approval
card.

- Centered, scrollable, header names the source.
- **Diff content** is colour-coded: `+` green, `-` red, `@@` cyan, context
  dimmed; header shows per-file stats (`+12 −4`). No-color fallback uses the
  prefix characters alone.
- **Plain content** renders with line numbers.
- `j`/`k`/arrows scroll, `PgUp`/`PgDn` page, `g`/`G` top/bottom, `Esc` closes.

### 4.2 Help overlay (`?`)

Single screen: key bindings grouped by mode (§5.2), then slash commands (§6),
then a footer with version and the docs path. `Esc` or `?` closes.

### 4.3 Confirm prompt

A one-line modal for destructive-ish actions: `/new` while a turn is in flight,
and clearing session grants from `/approvals`. `y`/`n` only, `Esc` cancels.
Quitting does **not** confirm — the session log is durable, so there is nothing
to lose and a confirm-on-quit is a tax on every exit.

## 5. Key Bindings

### 5.1 Mode state machine

```
                 ┌──────────────────────────────────────┐
                 │              Modal                    │  traps all input
                 └───────▲──────────────────────┬────────┘
                    d/?  │                 Esc  │ returns to previous
                 ┌───────┴──────────┐           │
                 │ ApprovalPending  │◄──────────┘  exclusive capture
                 └───────▲──────────┘
      approval requested │ │ y/a/n resolves
                 ┌───────┴─▼────────┐   ↑ at top      ┌──────────┐
                 │    Composing     │────────────────►│ Browsing │
                 │ (composer focus) │◄────────────────│(card sel)│
                 └──────────────────┘   ↓ at bottom   └──────────┘
                          │                                │
                          └────────► Confirm ◄─────────────┘
```

`Busy` is an orthogonal flag, not a mode: it dims and disables the composer and
drives the status-bar spinner, but does not change which mode is active.

**Precedence when states overlap:** `Modal` > `ApprovalPending` > `Confirm` >
`Browsing` / `Composing`.

Two consequences worth stating because they are easy to get wrong:

- An approval that arrives **while a modal is open** appends its card and
  renders it, but exclusive capture begins only when the modal closes.
  Otherwise the user's modal keystrokes get silently eaten by an approval they
  have not seen.
- `?` opens help only in `Browsing` and `ApprovalPending`. **In `Composing`,
  `?` is a literal character** — a help overlay that fires mid-sentence makes
  the composer unusable. `/help` is the route while composing.

### 5.2 Bindings by mode

**Composing**

| Key | Action |
|---|---|
| `Enter` | Send (turn starts) |
| `Shift+Enter` | Newline, where the terminal sends `ESC`+`CR` for it |
| `Alt+Enter` | Newline, same decode path — **not produced by Option on a Mac keyboard** |
| `Ctrl+J` | Newline — the one binding that is always representable |
| `Tab` | Complete a unique slash-command prefix |
| `↑` at first line | Focus transcript (→ Browsing) |
| `Esc` / `Ctrl+C` | Cancel the in-flight turn if busy; otherwise clear composer |
| `Ctrl+C` ×2 within 1s | Force quit |

There is no bare-key quit. `q` is an ordinary character in the composer, and
quitting is `/quit`, `/exit`, or `Ctrl+C` twice — see §6.

**On the newline binding.** No binding here is "primary", because which one reaches the
program is a property of the terminal rather than of Kirsch. Bubble Tea v1's `tea.Key` is
`{Type, Runes, Alt, Paste}` and carries **no shift modifier**, so the toolkit cannot name
Shift+Enter — but terminals that emit `ESC`+`CR` for it decode down the same path as
Alt+Enter, and in practice that is what most send. Conversely, Option+Enter on a Mac
keyboard produces nothing at all. So the composer accepts `ESC`+`CR` however the terminal
produces it, and `Ctrl+J` (a literal line feed), which is the only newline that is always
representable. Reasoning from the toolkit's API alone got this backwards once already —
see plan amendment 40.

**Browsing**

| Key | Action |
|---|---|
| `↑` / `↓` | Move card selection |
| `PgUp` / `PgDn` | Scroll without moving selection |
| `Home` / `End` | Top / bottom (`End` re-pins) |
| `g` / `G` | Top / bottom — the keyboard-reachable form of `Home` / `End` (`G` re-pins) |
| `Enter` | Expand / collapse selected card |
| `d` | Open content or diff modal for selected card |
| `?` | Help overlay |
| `Esc` | Return to Composing (does **not** re-pin — §2.4) |
| `↓` at last card | Return to Composing |

**On `g` / `G`.** They mirror the modal's own `g`/`G` rather than inventing a second
vocabulary for the same gesture, and they exist because `Home` and `End` are not
universally reachable: on macOS Terminal both arrive as SS3, which Bubble Tea v1.3.10
does not decode. They are deliberately absent from the **`browsing`** block of the help
overlay — the overlay's `modal` block does list them — because that grid is pinned by
screen 06 in `plan/kirsch-ui-screens.md`, so adding a row there is a spec amendment
rather than a code change. The overlay is a one-screen summary, not this table: its
`browsing` block also omits `Home` and `Esc`, both of which are bound.

**ApprovalPending**

| Key | Action |
|---|---|
| `y` | Approve once |
| `a` | Approve + session grant (`run_command` only) |
| `n` | Reject |
| `d` | Open detail / diff modal |
| `?` | Help overlay |
| `Esc` / `Ctrl+C` | Cancel the turn (counts as rejection) |

All other keys are swallowed with no effect — never forwarded to the composer.

**Modal**

| Key | Action |
|---|---|
| `j` / `k` / `↑` / `↓` | Scroll |
| `PgUp` / `PgDn` | Page |
| `g` / `G` | Top / bottom |
| `Esc` (or `?` in help) | Close, return to previous mode |

### 5.3 Global

`Ctrl+C` is context-dependent by design: cancel a turn if one is running,
otherwise arm force-quit. Two presses within 1s always force-quit, from any
mode. The session log stays intact because of the flush-on-turn rule
(plan §7).

## 6. Slash Commands

**Grammar.** The composer content must be a **single line beginning with `/`**.
Multi-line content starting with `/` is an ordinary message — pasting a path or
a diff must not be mistaken for a command. Arguments are the rest of the line,
trimmed, unparsed.

| Command | Behaviour |
|---|---|
| `/help` | Help overlay |
| `/status` | Notice: model, branch, dirty file count, session id, tokens (in/out/cache), compaction %, project-context file loaded, active grants |
| `/diff` | Content modal on the working tree (`git diff`) |
| `/files` | Notice listing files touched this session |
| `/approvals` | Notice listing active session grants; offers to clear (confirm prompt) |
| `/new` | New session; confirm if a turn is in flight |
| `/compact` | Manual compaction (plan §6.4); notice confirms, status bar shows `compacting` |
| `/quit` | Quit |
| `/exit` | Quit — an alias of `/quit`, sharing one arm rather than a second behaviour |

Unknown `/command` → a dim inline hint under the composer, not an error card,
and **never sent to the model**. `Tab` completes a unique prefix.

Milestone 1 adds temporary debug commands — `/read`, `/ls`, `/search`,
`/gitstatus`, `/gitdiff` — labelled `(debug)` in `/help` and removed in M3.

## 7. Edge Cases

### 7.1 Text sanitisation (do this before rendering anything)

- **Strip every ANSI escape sequence** from tool output before it reaches the
  renderer — SGR colours, cursor movement, and OSC sequences alike. `git` and
  many test runners emit colour when they detect a TTY, and an unstripped
  sequence does not merely look wrong: it hands arbitrary terminal control to
  command output. Kirsch colours diffs itself, from parsed structure, never by
  passing escapes through.
- **Carriage returns**: a `\r`-separated run (progress bars from `npm`, `go
  test`, `curl`) renders as its final segment only — that is what the writer
  intended the user to see.
- **Tabs** expand to 4 spaces, consistently everywhere, so code indentation
  lines up.
- **Other C0 control characters** are replaced with a dim `·`.
- **Very long lines**: any single rendered line is capped at 2,000 columns with
  a `⋯` marker. A minified bundle on one line must not be able to hang the
  wrapper.

### 7.2 Input

- **Bracketed paste**: pasted text is literal, never interpreted as keys;
  newlines preserved. Paste over 8KB warns before inserting.
- **Paste debounce** (§11) so a large paste is one update, not thousands.

### 7.3 Rendering

- **Soft wrap** everywhere; no horizontal scrolling in v0.1.
- **No-color fallback**: `NO_COLOR` set, `TERM=dumb`, or not a TTY → styling
  degrades to plain prefixes (`+`/`-`, `[ok]`, `[err]`); layout identical.
- **No-Unicode fallback**: when the locale is not UTF-8, every glyph falls back
  per the §10.2 table.
- **Resize**: reflow at any size; modals re-clamp; selection and scroll position
  preserved where the content still exists.
- **Spammy streaming**: repaints coalesced (§11) so a fast stream does not
  thrash the terminal. The same path carries incremental `run_command` output.

### 7.4 Session and process

- **Force quit**: two `Ctrl+C` within 1s exits immediately, no confirm.
- **Recovered session**: `⚠ recovered session` in the status bar plus a system
  notice at the truncation point in the transcript.
- **Second instance**: `⚠ another Kirsch is running here`; session grants
  disabled (plan §7).

### 7.5 Empty and first-run states

These are onboarding screens, not error cards — a new user's first experience
must not be a red border.

| Condition | Screen |
|---|---|
| No API key | Which env vars are read, in precedence order, and that keys are never read from config files |
| Not in a Git repo | Run inside a repository, or use `--workspace <dir>` |
| Empty session | Placeholder plus three example prompts |
| Model unknown to the model table | Dim notice: cost display unavailable, conservative budget in use |

## 8. Milestone 0 Prototype Scope

The prototype implements everything above with **fake data**:

- Fake transcript: 2 user messages, timer-driven streaming assistant text,
  3 tool cards (one truncated, one errored), **two** approval cards (one
  `apply_patch` without `[a]`, one `run_command` with `[a]`), 1 error card,
  1 system notice.
- Working: composer typing, the full mode state machine (§5.1), card selection
  and expansion including the 200-line inline cap, approval `y`/`a`/`n` flow,
  content/diff modal, help overlay, confirm prompt, resize, quit paths,
  scroll/pin behaviour, all §10 styling with both fallbacks.
- **No** LLM, file, shell, or session code. No real policy. Spinner states and
  streaming are simulated.

**Acceptance:** `go run ./cmd/kirsch` opens; typing, scrolling, selection,
modals, resize, and quit all work with no panic or visual corruption.

## 9. Resolved Decisions

- **Newline binding:** `Alt+Enter` primary, `Ctrl+J` fallback. `Shift+Enter` was
  the original preference but is not representable on Bubble Tea v1, which
  carries no shift modifier — and most terminals send it as a bare CR anyway.
  M0 tests both bindings it can actually construct. See §5.2 and plan
  amendment 29.
- **Palette:** dark only for v0.1. No light theme. See §10.
- **Background:** never painted; the terminal's own background shows through.
- **Token display:** 1 decimal, k/M units (`12.4k tok`).
- **Syntax highlighting:** out of scope for v0.1.
- **Mouse support:** out of scope for v0.1.
- **Transcript search:** deferred to v0.2.
- **Inline expansion cap:** 200 lines, full content via the modal.
- **`?` in the composer is a literal character**, not a help key.
- **Typing never re-pins** a scrolled-up transcript.
- **No bare-key quit.** `q` is an ordinary character in the composer. Quitting is
  `/quit`, `/exit`, or `Ctrl+C` twice. See §5.2 and §6.
- **`Esc` does not re-pin.** Leaving Browsing returns focus to the composer and
  keeps the viewport where it is. See §2.4.
- **Slash commands re-pin** on submission; the two hint-only paths do not, because
  they put nothing in the transcript. See §2.4.
- **`g`/`G` are bound in the transcript pane as well as in modals** — one vocabulary
  for the gesture, and the reachable form of `Home`/`End` where those arrive as
  undecoded SS3. See §5.2.

**Open questions (not blocking Milestone 0):**

- Whether `/approvals` deserves a direct key binding.
- Whether the 200-line inline cap should be configurable.

---

## 10. Appendix — Visual Language

### 10.1 Palette (dark, 256-colour)

Semantic roles only. Never reference a raw colour number outside `styles.go`.
The palette and glyph tables are repeated at the end of
[`kirsch-ui-screens.md`](kirsch-ui-screens.md) so the screens can be read
without flipping back here. §10 is the source of truth; change the two
together or they drift.

| Role | 256 | Hex | Contrast on `#1e1e1e` | Used for |
|---|---|---|---|---|
| `text` | 253 | `#dadada` | 11.9:1 | Default body text |
| `muted` | 248 | `#a8a8a8` | 7.0:1 | Metadata: paths, durations, counts |
| `dim` | 245 | `#8a8a8a` | 4.8:1 | Placeholders, suggestions, decoration |
| `accent` | 117 | `#87d7ff` | 10.5:1 | Header, card glyphs, selection gutter, focus |
| `success` | 120 | `#87ff87` | 13.2:1 | `✓`, diff additions |
| `error` | 203 | `#ff5f5f` | 5.6:1 | `✗`, diff deletions, error borders |
| `warning` | 215 | `#ffaf5f` | 9.2:1 | `⋯` truncation, status-bar warnings |
| `hunk` | 123 | `#87ffff` | 14.1:1 | Diff `@@` headers |
| `border` | 244 | `#808080` | 4.2:1 | Card and modal borders, separators |
| `borderFocus` | 117 | `#87d7ff` | 10.5:1 | Focused modal border |
| `codeBg` | 235 | `#262626` | — | Fenced code block background |
| `selectionBg` | 236 | `#303030` | — | Selected card background |

Background is never set on `body`.

**Contrast floor.** Every foreground token clears 3:1 against a dark ground,
and every token carrying words clears 4.5:1. The original palette did not: at
`dim` 240 (2.3:1) and `border` 238 (1.7:1), placeholders, the three onboarding
suggestions and every separator rule were effectively invisible on a dark
terminal — the whole interface read as washed-out grey. §12's rule that dim
text carries no unique information is a reason it may be *quieter*, never a
licence for it to be unreadable.

### 10.2 Glyphs and fallbacks

| Meaning | Unicode | ASCII fallback |
|---|---|---|
| Collapsed card | `▸` | `>` |
| Expanded card | `▾` | `v` |
| Tool running (card badge) | `◐` | `*` |
| Spinner (status bar, animated) | `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` | `- \ \| /` cycle |
| Pending | `◌` | `.` |
| OK | `✓` | `[ok]` |
| Error | `✗` | `[err]` |
| Cancelled | `⊘` | `[canc]` |
| Truncated | `⋯` | `...` |
| Dirty repo | `●` | `*` |
| Warning | `⚠` | `!` |
| New content | `↓` | `v` |
| Stream caret | `▌` | `_` |
| Selection gutter | `┃` | `\|` |
| Box drawing | `┌─┐│└┘` | `+-+\|+ +` |

Fallback triggers when the locale is not UTF-8. **Every state is identified by
its glyph, not only by its colour** (§12).

**Two distinct running indicators, one clock.** The card badge (`◐`) is a static glyph
marking a tool's lifecycle state (§3.8); the status-bar spinner is animated, advancing one
frame per §11's 100 ms tick. They appear together — screen 02 shows `◐` on the running tool
card while the status bar reads `⠋ running go test` — and both are driven by the same frame
counter, so they never drift apart. Do not collapse them into one glyph.

## 11. Appendix — Timing Constants

Collected here so they are not scattered magic numbers.

| Constant | Value | Rationale |
|---|---|---|
| Spinner frame | 100ms | 10fps reads as smooth, costs little |
| Stream repaint coalescing | 50ms, or on newline | Below perceptible lag, above thrash |
| Command output coalescing | 100ms | Output is chunkier than tokens |
| Paste debounce | 20ms | One update per paste |
| Double `Ctrl+C` window | 1000ms | Long enough to be deliberate |
| Cancel → composer usable | < 1000ms (target) | Tested, not eyeballed (plan §8 M3) |
| Approval card appearance | immediate | Never debounced |

## 12. Appendix — Accessibility

1. **Never signal state by colour alone.** Every state carries a glyph; colour
   reinforces. This is what makes the no-color fallback a downgrade rather than
   a loss of information.
2. `NO_COLOR`, `TERM=dumb`, and non-TTY output are all honoured.
3. No blinking, no flashing, no animation other than the spinner.
4. No binding requires more than two keys; no mouse, ever.
5. Dim and faint text never carry unique information.
6. Layout is identical with and without colour — only styling degrades, so a
   screen reader or a piped capture sees the same structure.

## 13. Appendix — Golden Test Surface

`View()` snapshots required (milestone-0 Task 6, extended in later milestones).
Each is drawn as a literal character grid in
[`kirsch-ui-screens.md`](kirsch-ui-screens.md) — build to that grid and capture
the golden file from it, rather than capturing whatever the first
implementation renders:

| # | State | Screen |
|---|---|---|
| 1 | Initial empty session | [01](kirsch-ui-screens.md#01--empty-session-first-run) (with the [00](kirsch-ui-screens.md#00--startup-wordmark) wordmark) |
| 2 | Mid-turn: spinner, partial assistant text, one running tool card | [02](kirsch-ui-screens.md#02--mid-turn-streaming--running-tool) |
| 3 | Approval pending — `apply_patch` variant (no `[a]`) | [03](kirsch-ui-screens.md#03--approval--apply_patch-no-a) |
| 4 | Approval pending — `run_command` variant (with `[a]`) | [04](kirsch-ui-screens.md#04--approval--run_command-with-a) |
| 5 | Content modal open over a pending approval | [05](kirsch-ui-screens.md#05--diff-modal-over-a-pending-approval) |
| 6 | Help overlay open | [06](kirsch-ui-screens.md#06--help-overlay) |
| 7 | Card expanded at the 200-line cap | [07](kirsch-ui-screens.md#07--tool-card-expanded-at-the-200-line-cap) |
| 8 | Error card and system notice | [08](kirsch-ui-screens.md#08--error-card--system-notices) |
| 9 | Narrow width (40 cols) and very narrow (38 cols, transcript hidden) | [10](kirsch-ui-screens.md#10--narrow-and-short-terminals) — first two grids |
| 10 | Short height (6 rows) | [10](kirsch-ui-screens.md#10--narrow-and-short-terminals) — third grid |
| 11 | `NO_COLOR` mode | [11](kirsch-ui-screens.md#11--no_color--ascii-fallback) |
| 12 | ASCII-fallback mode | [11](kirsch-ui-screens.md#11--no_color--ascii-fallback) |
| 13 | Scrolled up with `↓ n new` indicator | [09](kirsch-ui-screens.md#09--scrolled-up-unpinned) |
| 14 | Onboarding: no API key; not a Git repo | **not yet drawn** — see below |

State 14 is the one gap. Onboarding is not reachable until the provider lands
in M3, so its screen is drawn then, *before* the golden file is captured
(plan §11, amendment 24). Every other state has a grid to build against today.

Structure is the contract and colour is applied on top: line counts and box
positions are identical with and without colour. Screen 11 is a
character-for-character transliteration of screen 03 through the §10.2 fallback
table — every substitution is width-preserving — so the pair is what makes the
property testable rather than merely asserted.

**Four properties hold for every state, and are worth more than the snapshots.**
They are what the snapshots exist to protect:

1. `strip(styled) == plain`, byte for byte. This is the real form of "identical
   with and without colour", and it holds for all thirteen states, not only the
   two drawn as a pair.
2. Zero `0x1b` bytes are emitted under `NO_COLOR`. Not "no visible colour" —
   *no escape bytes at all*.
3. Every escape in styled output is an SGR sequence drawn from the §10.1
   palette. Anything else — cursor movement, an OSC sequence, a colour index
   that is not in the table — means either tool output reached the terminal
   unfiltered (§7.1) or a raw colour number escaped `styles.go`.
4. Row count equals the terminal height and no row exceeds the terminal width,
   at every size in §2.2.

Property 3 supersedes the looser instruction to assert that "no escape sequence
survives into `View()`". That is false as stated: Kirsch emits its own SGR
sequences whenever colour is on. The checkable claims are 2 and 3 — no escapes
at all in the no-colour path, and only palette SGR in the coloured one.

Golden files are regenerated with `-update` and reviewed by a human in the
diff. CI never auto-accepts them. When a golden diff is reviewed and found
*correct*, the screen reference has gone stale — update it in the same commit.

## 14. Appendix — Open Design Risks

Named so they are watched, not discovered:

- **~~`Shift+Enter` detection~~ — settled by testing, amendments 40 and 42.**
  Shift+Enter works wherever the terminal emits `ESC`+`CR`; Option+Enter on a Mac
  produces nothing; `Ctrl+J` always works. A config-selectable binding is not
  needed. Worth remembering how this resolved: reading the toolkit's key table
  produced a confident and wrong conclusion, and a person pressing the key in a
  real terminal produced the right one.
- **~~Braille spinner glyphs~~ — settled, amendment 42.** They render correctly in
  the target terminals. The ASCII cycle stays as the non-UTF-8 fallback, not as a
  default.
- **200-line inline cap** is a guess at the right number. M0 is the moment to
  find out whether it feels right with real-shaped content.
- **Exclusive approval capture** is correct but can feel abrupt when an
  approval interrupts scrollback reading. Watch for it during M2.
