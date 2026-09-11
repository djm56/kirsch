# Kirsch — TUI Spec v0.1 (Draft)

> **Status: Planning-phase draft.** This spec defines the UI that Milestone 0
> must prototype (with fake data) and that later milestones will wire to real
> agent events. It is the source of truth for "the feel of the UI."

## 1. Design Goals

1. **Readable transcript first.** The conversation viewport is the star; every
   other element yields space to it.
2. **No surprises.** Consequential actions (patches, commands) are visible as
   cards *before* they happen; nothing applies silently.
3. **Keyboard-only, vi-adjacent feel.** No mouse required; everything
   reachable in one or two keystrokes.
4. **Never frozen.** Spinner for any in-flight work; every turn cancellable;
   stuck tools don't stick the UI.
5. **Terminal-agnostic.** Works from 60×20 to huge; safe with no-color, soft
   wrap, and zero-width resize.

## 2. Full-Screen Layout

```
┌─ Kirsch ─ my-project ─ main ● ─────────────────┐  ① header (1 line)
│                                                │
│  [transcript viewport — scrolls]               │  ② transcript (fills)
│                                                │
│  ── user ──────────────────────                │
│  Fix the Divide validation                     │
│                                                │
│  ▸ read_file calc/divide.go            4ms ✓   │  tool card (collapsed)
│                                                │
│  I found the issue in... ▌                     │  streaming assistant
│                                                │
├────────────────────────────────────────────────┤
│ claude-sonnet-5 · ⠋ thinking · 12.4k tok       │  ③ status bar (1 line)
├────────────────────────────────────────────────┤
│ > _                                            │  ④ composer (1–5 lines)
└────────────────────────────────────────────────┘
```

- **① Header** — `Kirsch ─ <project name> ─ <branch>[ ● if dirty]`. When the
  session is compacted, append ` (compacted)`.
- **② Transcript viewport** — all conversation history (user messages,
  assistant text, tool cards, approval cards, error cards, system notices).
  Auto-scrolls to bottom on new content unless the user has scrolled up
  (then shows a `↓ new content` indicator; any key returns to bottom).
- **③ Status bar** — left: model name, state (`idle` / spinner + word:
  `thinking` / `running go test` / `applying patch`), cumulative token count.
  Right: warning/error glyphs (e.g. `⚠ recovered session`).
- **④ Composer** — multiline text area. Grows to 5 lines, then scrolls
  internally. Placeholder `Ask anything (Enter to send, ? for help)` when
  empty. Disabled (dimmed) while a turn is in flight; Esc/Ctrl+C cancels the
  turn and re-enables it.

### Sizing rules

- Transcript gets all leftover height; header/status are fixed at 1 line each.
- Composer height = current content, clamped 1–5 lines.
- On resize, reflow; on **zero-width/zero-height**, render nothing and do not
  panic (guard in `View()`).
- All width math is rune-aware (`go-runewidth`); East-Asian text never
  corrupts borders.

## 3. Transcript Elements

### 3.1 User message

```
── you ──────────────────────
Fix the Divide validation
```
Verbatim; soft-wrapped at viewport width. Slash-command invocations render
the same way (the command's result appears as a system notice or tool card).

### 3.2 Assistant message

Streams token-by-token with a `▌` caret. Once complete, caret removed and the
block stays. Soft-wrapped. Code spans (triple backticks) render with a
distinct background if color is available; otherwise plain indentation is
preserved exactly.

### 3.3 Tool card (core UI element)

**Collapsed (default):**

```
▸ read_file calc/divide.go · 4ms · ok
```

- `▸` glyph; name in bold; path/description dimmed; result summary
  (`ok` / `error` / `truncated`); duration.
- Tool-specific summary line, e.g.:
  - `read_file` → path + line range
  - `list_files` → path + `n files`
  - `search_code` → query + `n matches`
  - `apply_patch` → files changed count + status
  - `run_command` → command + exit status

**Expanded (`Enter` on card):** full `content` / `display_summary` output in a
scrollable region, with `‹truncated — 200KB cap›` / `‹truncated — 4000 tok›`
markers where caps hit. `Esc`/`Enter` collapses.

**State badges:** running (`◐` spinner on the card itself), ok (`✓` green),
error (`✗` red), cancelled (`⊘` dim), truncated (`⋯`).

### 3.4 Approval card (before an action runs)

```
┌─ approval required ────────────────────────────┐
│ apply_patch — add input validation              │
│                                                 │
│ files: 2 changed (calc/divide.go,               │
│        calc/divide_test.go)                     │
│                                                 │
│ [y] approve   [n] reject   [d] view diff       │
└─────────────────────────────────────────────────┘
```

- For `run_command`: shows command, args, cwd, timeout, and why it needs
  approval ("not on allowlist"), and offers a fourth action:

  ```
  [y] approve   [a] approve for session   [n] reject   [d] detail
  ```

- `y` → `approval granted` recorded, card collapses to resolved state
  (`✓ approved` / `✗ rejected`), action proceeds.
- `a` → approved **and** a session-scoped grant is recorded for the argv
  prefix shown on the card (plan §4). Card collapses to
  `✓ approved · session grant: go test`. **Never offered for `apply_patch`**,
  and never for `sh -c`.
- `n` → rejection is sent back to the model as a tool result so it can adapt.
- Only one approval pending at a time; input is captured exclusively.

### 3.5 Thinking card

When extended thinking is enabled (plan §6.5), thinking blocks render as a
collapsed, dimmed card:

```
▸ thinking · 412 tok
```

Expand/collapse exactly like a tool card. Dimmed styling throughout so it
never competes with assistant text. Default is off in v0.1, but the render
path exists from Milestone 3.

### 3.6 Error card

Distinct red-bordered card for `error` events: kind (`command_timeout`,
`patch_conflict`, …) + one-line detail; `Enter` expands full detail.

## 4. Modals

### 4.1 Diff modal (`d`, or from approval card)

- Centered box, 80% width/height, scrollable.
- Colour-coded unified diff: `+` green, `-` red, `@@` cyan, context dimmed.
  No-color fallback: prefix characters only.
- Header: files changed + per-file stats (`+12 −4`).
- `Esc` closes; `j`/`k` or arrows scroll.

### 4.2 Help overlay (`?`)

Single-screen cheat sheet: key bindings + slash commands (below), `Esc` to
close.

## 5. Key Bindings

| Key | Context | Action |
|---|---|---|
| `Enter` | composer | Send message (turn starts) |
| `Shift+Enter` | composer | Newline (primary binding) |
| `Alt+Enter` | composer | Newline (compatibility fallback — some terminals don't send Shift+Enter distinctly) |
| `Enter` | transcript | Expand/collapse selected tool card |
| `↑` / `↓` | transcript | Move card selection |
| `PgUp` / `PgDn` | transcript | Page scroll |
| `y` / `n` | approval pending | Approve / reject |
| `a` | approval pending (`run_command` only) | Approve + grant for this session |
| `d` | approval pending / card | Open diff modal |
| `Esc` / `Ctrl+C` | turn in flight | Cancel turn (model call + tools) |
| `Esc` | modal open | Close modal |
| `?` | anywhere | Help overlay |
| `q` | idle, empty composer | Quit |
| `Ctrl+C` twice fast | anywhere | Force quit (no prompt) |

Composer has focus by default; `↑`/`↓` at composer top/bottom transfers focus
to the transcript (with a visible selection cursor on cards).

## 6. Slash Commands

| Command | Behavior |
|---|---|
| `/help` | Help overlay |
| `/status` | System notice: model, branch, dirty files, session id, tokens, compaction % |
| `/diff` | Diff modal on current working tree (git diff) |
| `/files` | System notice listing touched files this session |
| `/approvals` | System notice listing active session-scoped grants; offers to clear them |
| `/new` | New session (confirm if mid-task) |
| `/compact` | Manual compaction per plan §6; system notice confirms |
| `/quit` | Quit (same as idle `q`) |

Unknown `/command` → inline dim hint, not an error card.

## 7. Edge Cases (binding Milestones 0 and 5)

- **Bracketed paste:** pasted text is literal (never interpreted as keys);
  multiline paste keeps newlines; paste of > 8KB warns before insert.
- **Soft wrap** everywhere; no horizontal scrolling in v0.1.
- **No-color fallback:** `NO_COLOR` env or non-TTY → all styling degrades to
  plain prefixes (`+`/`-`, `[ok]`, `[error]`); layout identical.
- **Zero/narrow width:** minimum useful width 40 cols; below that render only
  status bar + composer, no panic.
- **Resize:** reflow at any size; modals clamp to 80% of new size.
- **Spammy streaming:** delta rendering coalesced (batch repaints) so huge
  fast streams don't thrash the terminal. The same coalescing path carries
  incremental `run_command` output (plan §3.2), so a long `go test` shows
  progress inside its tool card rather than a frozen spinner.
- **Force quit:** two `Ctrl+C` within 1s kills everything without confirmation
  (session JSONL still intact due to flush-on-turn rules).

## 8. Milestone 0 Prototype Scope

The prototype implements everything above with **fake data**:

- Fake transcript: 2 user messages, streaming-simulated assistant text
  (timer-driven), 3 tool cards (one truncated, one errored), **two** approval
  cards (one `apply_patch` without `[a]`, one `run_command` with `[a]`), and
  1 error card.
- Working: composer typing, card selection/expand, approval `y`/`a`/`n` flow,
  diff modal with a canned diff, help overlay, resize, quit paths.
- **No** LLM, file, shell, or session code. No real policy. Spinner states
  are simulated.

**Acceptance:** `go run ./cmd/kirsch` opens; typing, scrolling, modal
open/close, resize, and quit all work with no panic or visual corruption.

## 9. Resolved Decisions

- **Newline binding:** `Shift+Enter` is the primary binding (user
  preference); `Alt+Enter` remains mapped as a compatibility fallback because
  terminal support for distinct Shift+Enter varies. Milestone 0 tests both.
- **Palette:** dark theme only for v0.1. No light palette switch — deferred
  beyond v0.1.
- **Token display:** 1 decimal, k/M units, exactly as mocked in §2
  (`12.4k tok`).
- **Transcript search:** deferred to v0.2 — not in v0.1 scope.
