# Kirsch — TUI screen reference (v0.1)

> **Status: Settled for v0.1.** Companion to [`ui-spec-v0.1.md`](ui-spec-v0.1.md),
> which stays normative: it says what the rules are, this shows what they produce.
> Where a screen and the spec disagree, **the spec wins and the screen is the bug**.
> Built against in [Milestone 0](../milestones/milestone-0.md) (Tasks 4–7) and held to in
> [Milestone 1](../milestones/milestone-1.md) (Task 9); registered in the plan's amendment log
> ([`kirsch-plan.md`](kirsch-plan.md) §11, amendment 24).

Every screen below is a literal character grid at **80 columns** (narrow variants noted
per screen). These are not illustrations: `TestMatchesScreenReference` parses every grid
out of this file and compares it against `View()`, so each one is verified byte-for-byte
on every test run. The structure is the contract; colour is applied on top per the tables
at the end.

When that test fails, exactly one of two things is true — the renderer is wrong, or this
document is. Fix whichever it is, and if it is this document, in the same commit.

Screens 00–11 cover thirteen of the fourteen §13 golden states — the §13 table maps each
state to its screen. The fourteenth (onboarding: no API key, not a Git repo) is not drawn
yet: it is unreachable until the provider lands in M3, and its screen is drawn there
before its golden file is captured.

Reading notes:

- Each grid's fence carries its exact terminal size — a fence reading `text 80×18` means
  80 columns by 18 rows. The row count **is** the terminal height, so the golden harness
  reads the size from the fence rather than inferring it. Screen 00 carries none: it is a
  component, not a full-screen render.
- **The first column of every grid row is the frame's left margin, and the last is its
  right margin.** That is why every row below starts with one space: the frame keeps one
  blank column at each edge and none top or bottom (ui-spec §2.1, layout invariant 6
  below). So a `text 80×18` grid has 80 terminal columns but **78 content columns**, and
  the full-width `─` rules run to column 79, not 80. Measure any row against the content
  width, never the fence's width, or every rule in this document will look one cell short.
- **The right margin is reserved, not written.** Nothing draws into it and no space is
  emitted there, so it never shows up as trailing whitespace — which is also why the
  right-trimming below costs nothing.
- Trailing whitespace cannot survive in a markdown source file, so every grid is stored
  right-trimmed and the golden comparison right-trims both sides before diffing.
- Blank first/last transcript lines are padding inside the transcript viewport, not
  literal blank output — the transcript region flexes to the terminal height.
- The header is the **first two rows** of every full-screen grid: the wordmark `Kirsch`
  and its rule, then the session line. Only the first of the two carries a rule.
- Full-width `─` runs are the header, status-bar and composer separators.
- `▌` is the streaming/text cursor. `●` after the branch is the dirty-worktree marker.
- Tabs in captured tool output are expanded to 4 spaces before rendering (§3.3).
- Kirsch never paints the terminal background. Only `codeBg` and `selectionBg` set a
  background, and only within their own panels.

---

## How to read the colour maps

Each screen is followed by a `colours` block. Format:

```text
<region>            <the literal characters it covers>    <token>  <256>  <hex>
```

Regions are listed in top-to-bottom, left-to-right order. Anything a screen's map does
not mention falls through to these defaults:

| Fallback | Token | 256 | Hex |
|---|---|---|---|
| Body text (user, assistant, tool output) | `text` | 253 | `#dadada` |
| Every separator and unfocused border | `border` | 244 | `#808080` |
| Metadata between `·` delimiters (paths, timings, counts) | `muted` | 248 | `#a8a8a8` |
| Placeholders, hints, backgrounded cells | `dim` | 245 | `#8a8a8a` |

Three rules hold on every screen:

1. **Tool names are the only bold text.** Nothing else sets SGR 1.
2. **Backgrounds are opt-in.** Only `codeBg` (235) and `selectionBg` (236) set one, and
   only inside their own panel. Every other cell leaves the host background untouched.
3. **Colour is decoration, never information.** Each state is identifiable from its glyph
   alone, which is what makes screen 11 a valid golden fixture.

The hex column is the standard xterm-256 value for that index. Emit the 256 index (SGR
38;5;N) as the primary path; the hex is there for truecolor terminals and for anyone
porting the palette.

---

## 00 · Startup wordmark

```text
█▄▀ █ █▀█ █▀▀ █▀▀ █ █
█▀▄ █ █▀▄ ▄▄█ █▄▄ █▀█

v0.1.0-dev · terminal-native coding agent

— non-UTF-8 / ASCII fallback —

K I R S C H
v0.1.0-dev . terminal-native coding agent
```

**Colours**

```text
wordmark            "█▄▀ █ █▀█ █▀▀ █▀▀ █ █" (both rows)          accent    117  #87d7ff
tagline             "v0.1.0-dev · terminal-native coding agent"  dim       245  #8a8a8a
(ASCII fallback)    same assignment, both rows                   accent    117  #87d7ff
```

- Two rows of half-blocks in `accent` (117). No background fill, so it sits on any host theme.
- **21 columns wide**, 2 rows — count the grid above, and note the single-cell `█` for `I`
  and for the two stems of `H` on row 1. Non-UTF-8 locale falls back to letter-spaced
  plain text.
- Shown on the empty session only. It scrolls away with the first message and never reappears.
- Suppressed below 40 cols or 10 rows, where the transcript needs the lines more. The band is
  read from the **terminal** width, not from the content width the frame's margin leaves — at
  40 and 41 columns the two disagree, and reading content width would retire the wordmark from
  the advertised minimum as a side effect of the margin.
- The tagline is truncated to the content width with the `⋯` marker (`...` in ASCII). Its width
  is not fixed: the version is a build-time string, and the shipped `0.1.0-dev` already spends
  41 of the 38 columns a 40-column terminal leaves. At that width the row reads
  `v0.1.0-dev · terminal-native coding a⋯`. The version leads because it is the part
  worth keeping when the row is cut, and the marker is there because the frame's own
  bound is a hard edge with no marker — unbounded, the row broke off mid-word with
  nothing to say it had been cut.

---

## 01 · Empty session (first run)

```text 80×18
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●

 █▄▀ █ █▀█ █▀▀ █▀▀ █ █
 █▀▄ █ █▀▄ ▄▄█ █▄▄ █▀█

 v0.1.0-dev · terminal-native coding agent


 Ask anything. Three things to try:

 · What does the approval flow do when a patch conflicts?
 · Add input validation to Divide and cover it with a test
 · Where is the session lock taken?
 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · idle · 0 tok
 ──────────────────────────────────────────────────────────────────────────────
 > Ask anything (Enter to send, /help for help)
```

**Colours**

```text
header.wordmark     "Kirsch" (row 1)                         accent    117  #87d7ff
header.rule         trailing "─" fill on row 1               border    244  #808080
header.title        "my-project ─ main" (row 2)              accent    117  #87d7ff
header.dirty        "●"                                      warning   215  #ffaf5f
wordmark            both block rows                              accent    117  #87d7ff
tagline             "v0.1.0-dev · terminal-native coding agent"  dim       245  #8a8a8a
lead_in             "Ask anything. Three things to try:"         muted     248  #a8a8a8
suggestions         the three "· …" lines                        dim       245  #8a8a8a
status.bar          "claude-sonnet-5 · idle · 0 tok"             muted     248  #a8a8a8
composer.prompt     ">"                                          accent    117  #87d7ff
composer.placeholder "Ask anything (Enter to send, …)"           dim       245  #8a8a8a
separators          both full-width "─" rules                    border    244  #808080
```

- Onboarding, not an error: no border, no red. §7.5.
- Composer placeholder is the literal string from §2. It appears **only here**: an empty
  composer shows the placeholder while the session is empty, the cursor `▌` once there is a
  transcript and the composer has focus, and nothing at all while an approval or modal holds
  capture (screens 03–06). A placeholder standing behind a pending approval reads as an
  invitation to type into a composer that is deliberately refusing input.
- Suggestions are `dim` (240) with `·` bullets; the lead-in line is `muted` (244).

---

## 02 · Mid-turn (streaming + running tool)

```text 80×15
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●
 ── you ───────────────────────────────────────────────────────────────────────
 Fix the Divide validation

 ▸ read_file calc/divide.go · 4ms · ✓ ok
 ◐ run_command go test ./... · running

 I found it in calc/divide.go — the zero check runs after the division, so the
 panic fires before validation can return an error. ▌

 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · ⠋ running go test · 12.4k tok
 ──────────────────────────────────────────────────────────────────────────────
 >                                                (input disabled, Esc cancels)
```

**Colours**

```text
header.*            as screen 01
speaker.rule        "── you ───…"                            border    244  #808080
speaker.label       "you"                                    muted     248  #a8a8a8
user.text           "Fix the Divide validation"              text      253  #dadada
tool.glyph          "▸"                                      accent    117  #87d7ff
tool.glyph.running  "◐"                                      accent    117  #87d7ff
tool.name           "read_file" / "run_command"    text 253 #dadada + BOLD
tool.args+timing    " calc/divide.go · 4ms · "               muted     248  #a8a8a8
tool.status.ok      "✓ ok"                                   success   120  #87ff87
tool.status.running "running"                                muted     248  #a8a8a8
assistant.text      the two wrapped lines                    text      253  #dadada
cursor              "▌"                                      dim       245  #8a8a8a
status.model        "claude-sonnet-5 · "                     muted     248  #a8a8a8
status.spinner      "⠋"                                      accent    117  #87d7ff
status.verb+tokens  "running go test · 12.4k tok"            muted     248  #a8a8a8
composer.prompt     ">" (disabled state)                     dim       245  #8a8a8a
```

- The spinner appears twice by design: `◐` on the running tool card, and the verb in the
  status bar (`running go test`). §3.8.
- Composer is dimmed and input-disabled while the turn is live; Esc cancels.
- Assistant text soft-wraps at the viewport width. No horizontal scrolling in v0.1 (§2.2).

---

## 03 · Approval — apply_patch (no `[a]`)

```text 80×17
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●
 ▸ search_code "Divide(" · 3 matches · ✓ ok

 ┃ ┌─ approval required ──────────────────────────────────────────────────────┐
 ┃ │ apply_patch — add input validation                                       │
 ┃ │                                                                          │
 ┃ │ files: 2 changed (calc/divide.go,                                        │
 ┃ │        calc/divide_test.go)                                              │
 ┃ │                                                                          │
 ┃ │ [y] approve   [n] reject   [d] view diff                                 │
 ┃ └──────────────────────────────────────────────────────────────────────────┘

 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · ⠙ awaiting approval · 14.1k tok
 ──────────────────────────────────────────────────────────────────────────────
 >
```

**Colours**

```text
header.*            as screen 01
tool.glyph          "▸"                                      accent    117  #87d7ff
tool.name           "search_code"                  text 253 #dadada + BOLD
tool.args           ' "Divide(" · 3 matches · '              muted     248  #a8a8a8
tool.status.ok      "✓ ok"                                   success   120  #87ff87
gutter              "┃" on every card row                    accent    117  #87d7ff
card.border         "┌ ─ ┐ │ └ ┘"                            border    244  #808080
card.title          "approval required"                      warning   215  #ffaf5f
card.bg             card interior cells             selectionBg 236 #303030 (background)
card.summary        "apply_patch — add input validation"  text 253 #dadada, tool name BOLD
card.detail         "files: 2 changed (…" both lines         muted     248  #a8a8a8
action.approve      "[y]"                                    success   120  #87ff87
action.reject       "[n]"                                    error     203  #ff5f5f
action.diff         "[d]"                                    accent    117  #87d7ff
action.labels       "approve" "reject" "view diff"           text      253  #dadada
status.spinner      "⠙"                                      accent    117  #87d7ff
status.rest        "claude-sonnet-5 · awaiting approval · …" muted     248  #a8a8a8
```

- Three actions only. `[a]` is **never** offered for `apply_patch`. §4.
- `┃` gutter marks selection; the card interior sits on `selectionBg` (236).
- Input capture is exclusive: every key that is not `y`/`n`/`d`/`?`/Esc is swallowed,
  not forwarded to the composer.

---

## 04 · Approval — run_command (with `[a]`)

```text 80×17
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●
 ▸ apply_patch 2 files changed · ✓ approved

 ┃ ┌─ approval required ──────────────────────────────────────────────────────┐
 ┃ │ run_command — go test ./...                                              │
 ┃ │                                                                          │
 ┃ │ cwd: .        timeout: 60s                                               │
 ┃ │ reason: not on allowlist                                                 │
 ┃ │                                                                          │
 ┃ │ [y] approve   [a] approve for session   [n] reject   [d] detail          │
 ┃ └──────────────────────────────────────────────────────────────────────────┘

 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · ⠹ awaiting approval · 16.8k tok · 1 grant
 ──────────────────────────────────────────────────────────────────────────────
 >
```

**Colours**

```text
all rows            identical assignment to screen 03
action.session      "[a]"                                    success   120  #87ff87
card.detail         "cwd: … timeout: …" / "reason: …"        muted     248  #a8a8a8
status.grants       "1 grant"                                muted     248  #a8a8a8
(collapsed form)    "✓ approved · session grant: go test"    success   120  #87ff87
```

- Four actions. `reason:` states why approval was required.
- Grant count (`1 grant`) appears in the status bar, dropped right after tokens at 60–79 cols.
- On `[a]` the card collapses to a single line and the grant is recorded:

```text
▸ apply_patch 2 files changed · ✓ approved
▸ run_command go test ./... · 2.4s · ✓ approved · session grant: go test
```

---

## 05 · Diff modal over a pending approval

```text 80×20
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●
 ── you ─┌─ calc/divide.go ─────────────────────────────────── +12 −4 ┐────────
 Fix the │ @@ -12,7 +12,15 @@ func Divide(a, b float64) (float64, er⋯ │
         │  func Divide(a, b float64) (float64, error) {              │
 ▸ search│ -    return a / b, nil                                     │
         │ +    if b == 0 {                                           │
 ┃ ┌─ app│ +        return 0, ErrDivideByZero                         │───────┐
 ┃ │ appl│ +    }                                                     │       │
 ┃ │     │ +    return a / b, nil                                     │       │
 ┃ │ file│  }                                                         │       │
 ┃ │     ├────────────────────────────────────────────────────────────┤       │
 ┃ │     │ j/k scroll · g/G top/bottom · Esc back to approval         │       │
 ┃ │ [y] └────────────────────────────────────────────────────────────┘       │
 ┃ └──────────────────────────────────────────────────────────────────────────┘

 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · ⠸ awaiting approval · 14.1k tok
 ──────────────────────────────────────────────────────────────────────────────
 >
```

**Colours**

```text
modal.border        "┌ ─ ┐ │ ├ ┤ └ ┘" of the modal      borderFocus 117  #87d7ff
modal.filename      "calc/divide.go"                         accent    117  #87d7ff
modal.added.count   "+12"                                    success   120  #87ff87
modal.removed.count "−4"                                     error     203  #ff5f5f
diff.hunk           lines starting "@@"                      hunk      123  #87ffff
diff.context        lines starting " " (space)               muted     248  #a8a8a8
diff.removed        lines starting "-"                       error     203  #ff5f5f
diff.added          lines starting "+"                       success   120  #87ff87
modal.footer        "j/k scroll · g/G … · Esc …"             muted     248  #a8a8a8
BACKGROUND CELLS    every cell not owned by the modal        dim       245  #8a8a8a
                    (approval card + transcript behind it lose their own colours
                     entirely while the modal is open — one flat dim pass)
status+composer     unchanged, still live                    muted     248  #a8a8a8
```

- The modal overwrites cells; the approval card and transcript behind it render at
  `dim` and are not redrawn until it closes.
- **The modal covers most of the approval card's key row.** Screen 03 ends that card with
  `[y] approve   [n] reject   [d] view diff`; here the box leaves `[y]` visible and takes
  the rest. That is the overlay landing where §4.1 puts it, recorded so the difference
  between the two screens reads as a consequence rather than a defect. Nothing is lost
  but the reminder: the modal traps every key while it is open, so `y`, `n` and `d` are
  inert at the moment they are hidden, and the footer already says
  `Esc back to approval`.
- Esc returns to the **pending approval**, not the composer. §4.1.
- Modal border uses `borderFocus` (111); unfocused panels use `border` (238).
- Colour comes from parsed diff structure. All ANSI in tool output is stripped first (§5.1).

---

## 06 · Help overlay

```text 80×32
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●
         ┌─ help ─────────────────────────────────────────────────────┐
 ▸ read_f│ composing                            approval              │
         │ Enter         send                   y         approve     │
         │ Shift+Enter   newline                a         + session   │
         │ Ctrl+J        newline (alt)          n         reject      │
         │ Tab           complete /cmd          d         detail      │
         │ ↑ at line 1   browse                                       │
         │ Esc           cancel turn            modal                 │
         │                                      j/k ↑/↓   scroll      │
         │ browsing                             g/G       top/bottom  │
         │ ↑/↓           select card            Esc       close       │
         │ PgUp/PgDn     scroll                                       │
         │ Enter         expand                 commands              │
         │ d             diff / content         /help  /status        │
         │ End           bottom, re-pin         /diff  /files         │
         │ ?             help                   /approvals  /new      │
         │                                      /compact  /quit       │
         │                                      /exit                 │
         │                                                            │
         │                                      debug (M1 only)       │
         │                                      /read  /ls            │
         │                                      /search  /gitstatus   │
         │                                      /gitdiff              │
         ├────────────────────────────────────────────────────────────┤
         │ kirsch v0.1.0-dev · docs: doc/usage.md · Esc or ? closes   │
         └────────────────────────────────────────────────────────────┘
 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · idle · 12.4k tok
 ──────────────────────────────────────────────────────────────────────────────
 >
```

**Colours**

```text
modal.border        box borders + "├ ┤" divider         borderFocus 117  #87d7ff
modal.title         "help"                                   accent    117  #87d7ff
section.headings    "composing" "browsing" "approval"        warning   215  #ffaf5f
                    "modal" "commands"
bindings            key + description columns                muted     248  #a8a8a8
footer              "kirsch v0.1.0-dev · docs: … · Esc or ? closes"  dim 245  #8a8a8a
BACKGROUND CELLS    header row behind the overlay            dim       245  #8a8a8a
```

- Opens from Browsing and ApprovalPending only. In the composer, `?` is a literal character. §4.2.
- One screen, no scrolling: bindings by mode, then slash commands, then the version footer.
- Mode headings are `warning` (179); bindings are `muted` (244).
- Two columns, both generated from the same binding table `update.go` dispatches on, so a
  binding cannot change behaviour while keeping its old description here.
- **The overlay needs 32 rows.** §4.2 says one screen, no scrolling, and the content now
  runs to 22 body lines plus four rows of modal chrome — a 26-row box. The frame spends
  six more around it: the two-row header, the two rules, the status bar and the composer.
  Below that height it scrolls rather than truncating silently, but the grid is drawn at
  the height where the rule actually holds. The `debug (M1 only)` block is Milestone 1
  scaffolding and leaves with the debug commands in M3, which buys back five rows — the
  blank spacer, the heading, and the three command rows beneath it. Body height is the
  taller of the two columns, so dropping them takes it from 22 to
  `max(left 15, right 17)` = 17, a 21-row box and a 27-row screen. The right-hand
  column uses a narrower key field than the left because its keys are single characters —
  a shared field pushes its descriptions past the modal's edge at §2.2's 80% width.

---

## 07 · Tool card expanded at the 200-line cap

```text 80×17
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●
 ┃   │     case 187: ok                                                       │
 ┃   │     case 188: ok                                                       │
 ┃   │     case 189: ok                                                       │
 ┃   │     case 190: ok                                                       │
 ┃   │     case 191: ok                                                       │
 ┃   │     case 192: ok                                                       │
 ┃   │     case 193: ok                                                       │
 ┃   │     case 194: ok                                                       │
 ┃   └────────────────────────────────────────────────────────────────────────┘
 ┃   ‹200 of 4,176 lines — press d for full output›

 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · idle · 22.9k tok · 1 grant
 ──────────────────────────────────────────────────────────────────────────────
 > ▌
```

**Colours**

```text
gutter              "┃" on every card row                    accent    117  #87d7ff
tool.glyph          "▾" (expanded)                           accent    117  #87d7ff
tool.name           "run_command"                  text 253 #dadada + BOLD
tool.args+timing    " go test ./... · 2.4s · "               muted     248  #a8a8a8
tool.status.fail    "✗ exit 1"                               error     203  #ff5f5f
output.border       inner "┌ ─ ┐ │ └ ┘"                      border    244  #808080
output.bg           inner panel cells                 codeBg    235 #262626 (background)
output.text         captured stdout/stderr verbatim          text      253  #dadada
cap.marker          "‹200 of 4,181 lines — press d …›"       warning   215  #ffaf5f
card.bg             card interior (selected)         selectionBg 236 #303030 (background)
NOTE                captured output is NOT syntax-coloured; ANSI is stripped (§5.1)
```

- A command failure the model can handle stays a tool card with `✗` — it is **not** promoted
  to an error card. §3.6.
- Output panel sits on `codeBg` (235) with indentation preserved.
- Cap marker in `warning` (179). `d` opens the full output in a content modal.
- **The grid shows the tail of the expansion, and that is not an omission.** A capped card
  is 204 rows — head, border, 200 lines, border, marker — so at any usable terminal height
  the head is above the fold. The earlier drawing showed a head, six output lines and the
  `‹200 of …›` marker together, which no renderer can produce: six lines and a 200-line cap
  are different claims. What you see here is what the state actually looks like, and the
  marker — the thing this state exists to show — is in frame.

---

## 08 · Error card + system notices

```text 80×14
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ● (compacted)
 · session recovered — 3 events after a torn line were discarded
 · compacted 94 events into a summary · 12 files touched this session

 ┌─ provider_error ───────────────────────────────────────────────────────────┐
 │ anthropic: 503 after 3 retries — request not sent                          │
 │ Enter to expand · the turn is still cancellable                            │
 └────────────────────────────────────────────────────────────────────────────┘

 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · idle · 31.2k tok                         ⚠ recovered session
 ──────────────────────────────────────────────────────────────────────────────
 > ▌
```

**Colours**

```text
header.compacted    "(compacted)"                            muted     248  #a8a8a8
notices             both "· …" lines                         dim       245  #8a8a8a
error.border        "┌ ─ ┐ │ └ ┘" of the error card          error     203  #ff5f5f
error.title         "provider_error"                         error     203  #ff5f5f
error.message       "anthropic: 503 after 3 retries — …"     text      253  #dadada
error.hint          "Enter to expand · the turn is still …"  muted     248  #a8a8a8
status.left         "claude-sonnet-5 · idle · 31.2k tok"     muted     248  #a8a8a8
status.warning      "⚠ recovered session"                    warning   215  #ffaf5f
NOTE                the error card sets NO background — border and title only
```

- Error cards are reserved for infrastructure failures (provider, session, transport), so red
  borders stay rare enough to mean something. §3.6.
- Notices are one `dim` line with a `·` prefix and no border. §3.7.
- Persistent conditions live in the status bar and are never truncated away.

---

## 09 · Scrolled up (unpinned)

```text 80×14
 Kirsch ───────────────────────────────────────────────────────────────────────
 my-project ─ main ●

 ┃ ▸ read_file internal/session/store.go:1-120 · 3ms · ✓ ok

 The lock is taken in Open, before auto-resume reads index.json, so a second
 instance starts a fresh session instead of adopting the first.

 ▸ git_diff · 2 files changed · 8ms · ✓ ok
                                                                        ↓ 3 new
 ──────────────────────────────────────────────────────────────────────────────
 claude-sonnet-5 · ⠼ thinking · 28.0k tok
 ──────────────────────────────────────────────────────────────────────────────
 > also check the second-instance path            (input disabled, Esc cancels)
```

**Colours**

```text
gutter              "┃" (selected card)                      accent    117  #87d7ff
card.bg             selected card row               selectionBg 236 #303030 (background)
tool.glyph          "▸"                                      accent    117  #87d7ff
tool.name           "read_file" / "git_diff"       text 253 #dadada + BOLD
tool.args+timing    " internal/session/store.go:1-120 · 3ms · "  muted 248  #a8a8a8
tool.status.ok      "✓ ok"                                   success   120  #87ff87
assistant.text      the two wrapped lines                    text      253  #dadada
new.counter         "↓ 3 new"                                accent    117  #87d7ff
composer.prompt     ">"                                      accent    117  #87d7ff
composer.text       "also check the second-instance path"    text      253  #dadada
cursor              "▌"                                      dim       245  #8a8a8a
```

- Typing while unpinned does not yank the view to the bottom; the counter keeps rising. §2.4.
- `↓ n new` renders on the last transcript row, right-aligned, in `accent` (111).
- Selection stays on the card the user chose. New content never steals it. §3.9.
- `End` jumps to the bottom and re-pins.

---

## 10 · Narrow and short terminals

**40 columns** — the minimum supported width:

```text 40×11
 Kirsch ───────────────────────────────
 my-project
 ▸ read_file calc/divide.go · ✓ ok

 Soft wrap only. No horizontal
 scrolling in v0.1. ▌

 ──────────────────────────────────────
 claude-sonnet-5 · ⠋ thinking
 ──────────────────────────────────────
 >        (input disabled, Esc cancels)
```

**38 columns** — below the transcript threshold:

```text 38×6
 terminal too narrow

 ────────────────────────────────────
 ⠋                ⚠ recovered session
 ────────────────────────────────────
 >      (input disabled, Esc cancels)
```

**6 rows** — below the header threshold:

```text 40×6
 2 lines

 ──────────────────────────────────────
 claude-sonnet-5 · idle
 ──────────────────────────────────────
 > ▌
```

**Colours**

```text
(applies to all three widths)
header.wordmark     "Kirsch" (row 1, 40-col form only)       accent    117  #87d7ff
header.title        "my-project" (row 2, branch dropped)     accent    117  #87d7ff
too_narrow.notice   "terminal too narrow"                    dim       245  #8a8a8a
status.spinner      "⠋"                                      accent    117  #87d7ff
status.rest         "sonnet-5 · thinking" / "idle"           muted     248  #a8a8a8
status.warning      "⚠" (38-col form, glyph only)            warning   215  #ffaf5f
tool.status.ok      "✓" (glyph only, label dropped)          success   120  #87ff87
NOTE                no token changes at any width — only which SPANS are emitted.
                    Truncation never substitutes a colour.
```

- Status-bar truncation order: tokens → grants → model family → bare spinner. Warnings
  survive to the end. §2.2.
- Below 40 cols the transcript hides; below 5 rows the header goes. Neither panics.
- Long paths and commands ellipsize in the middle (`internal/…/store.go`), never wrap
  inside a card title.

---

## 11 · NO_COLOR + ASCII fallback

```text 80×17
 Kirsch -----------------------------------------------------------------------
 my-project - main *
 > search_code "Divide(" . 3 matches . [ok]

 | +- approval required ------------------------------------------------------+
 | | apply_patch - add input validation                                       |
 | |                                                                          |
 | | files: 2 changed (calc/divide.go,                                        |
 | |        calc/divide_test.go)                                              |
 | |                                                                          |
 | | [y] approve   [n] reject   [d] view diff                                 |
 | +--------------------------------------------------------------------------+

 ------------------------------------------------------------------------------
 claude-sonnet-5 . \ awaiting approval . 14.1k tok
 ------------------------------------------------------------------------------
 >
```

**Colours**

```text
ALL CELLS           default terminal foreground — no SGR colour sequences at all.
                    Kirsch emits zero escape codes when NO_COLOR is set.
BOLD                also dropped: no SGR 1. Status is carried by [ok] / [err] only.
NOTE                this screen is the golden-test fixture. Compare Kirsch's stripped
                    output against it byte-for-byte; screen 03 is the same structure
                    with the token assignment above applied.
```

- This screen is a **character-for-character transliteration of screen 03** through the
  §10.2 fallback table: every substitution is width-preserving (`✓ ok` → `[ok]` is 4 cells
  either way), so the two grids have identical line counts and identical box columns.
- That is what makes the pair testable. The golden assertion is
  `strip(screen 03 rendered) == screen 11`, byte for byte — a piped capture sees the same
  structure with or without colour, which is ui-spec §12's accessibility rule made checkable.
- Both fallbacks are applied at once: no SGR at all (`NO_COLOR`) *and* ASCII glyphs
  (non-UTF-8 locale). Each is independently switchable; this grid shows the floor.
- Substitutions visible here: `-` rules, `*` dirty marker, `>` collapsed glyph, `|` selection
  gutter and box verticals, `+` box corners, `.` metadata separator, `[ok]` status, `\`
  spinner (frame 1 of the `- \ | /` cycle, matching screen 03's `⠙`).

---

## Palette (§10.1)

| Token | 256 | Hex | Used for |
|---|---|---|---|
| `accent` | 117 | `#87d7ff` | Header title, tool glyphs, selection gutter, focused border, `↓ n new`, `[d]` |
| `text` | 253 | `#dadada` | Assistant and user body text, tool output, action labels |
| `muted` | 248 | `#a8a8a8` | Metadata (paths, timings, counts), key bindings, status bar |
| `dim` | 245 | `#8a8a8a` | Placeholders, suggestions, backgrounded cells, version footer, cursor |
| `success` | 120 | `#87ff87` | `✓`, `[y]`, `[a]`, diff additions, grant confirmations |
| `error` | 203 | `#ff5f5f` | `✗`, `[n]`, diff deletions, error card border and title |
| `warning` | 215 | `#ffaf5f` | Dirty marker, `approval required`, cap marker, status-bar warnings |
| `hunk` | 123 | `#87ffff` | Diff `@@` hunk headers |
| `border` | 244 | `#808080` | Unfocused borders, all separators |
| `borderFocus` | 117 | `#87d7ff` | Focused modal border |
| `codeBg` | 235 | `#262626` | Fenced tool-output panels (background) |
| `selectionBg` | 236 | `#303030` | Selected card interior (background) |

Nothing sets a default background. `codeBg` and `selectionBg` apply only inside their panels.

## Glyphs (§10.2)

| Meaning | UTF-8 | ASCII |
|---|---|---|
| Tool call, collapsed | `▸` | `>` |
| Tool call, expanded | `▾` | `v` |
| Tool running | `◐` | `*` |
| Success | `✓ ok` | `[ok]` |
| Failure | `✗` | `[err]` |
| Selection gutter | `┃` | `|` |
| Dirty worktree | `●` | `*` |
| Warning | `⚠` | `!` |
| Notice bullet | `·` | `.` |
| Spinner | `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` | `- \ | /` |
| Cursor | `▌` | `_` |
| Box drawing | `┌─┐│└┘├┤` | `+-|` |

## Layout invariants

1. Row order is always: header, transcript, status bar, composer. Only the transcript flexes.
2. The header is two rows — the wordmark and its rule, then the session line. The status bar
   and composer are one row each. The composer grows to a maximum of 5 rows on `Alt+Enter`,
   taking rows from the transcript. The header is shown or hidden **as a unit**, so the
   height ladder frees two rows at the step that drops it.
3. Modals overwrite the transcript region only. The status bar and composer stay visible.
   This binds every overlay in §4, the one-line confirm prompt of §4.3 included — it is
   bottom-aligned **inside** the region, immediately above the composer it asks about,
   not on the rule/status/rule band below it. It has exactly one exception, under
   **The floor** below, and that exception is a frame of two rows or fewer.

   **Height.** The overlay's own chrome yields rather than the region. The footer rule goes
   first, then the body; the two borders and the footer never go, because an overlay traps
   every key and the footer is what names the way out — a box holding one line of content
   and no exit key is worth less than one holding no content and an exit key. At 40×10 —
   the smallest supported terminal, §1 — the two-row header leaves four rows, one short of
   the full box, so that is where the rule goes, and a three-row region drops the body too.
   A box with no body rows also drops the scroll bindings from its footer: there is nothing
   left to scroll, and a binding named where it does nothing is worse than a shorter footer.
   Below three rows no bordered box holds a footer, so the overlay folds out of the box
   form into a single row: the modal's names the modal and the key that closes it, the
   confirm prompt's names the question and the keys that answer it. **There is no rung at
   which nothing is drawn.** An overlay that draws nothing while trapping every key is a
   frame that reads as an idle session and cannot be left, and that is the one outcome the
   ladder exists to prevent — so it is not a rung of the ladder, it is the thing the ladder
   is measured against.

   **Reservation.** The region an overlay needs is reserved by the layout, not asked for by
   the overlay. While one is open the height ladder holds a region row back, and pays for it
   out of the composer: an overlay traps every key, so the composer accepts no input while
   one is up and its rows are the cheapest in the frame. This is a floor on the existing
   order rather than a second ladder — the composer is already what gives way first — and it
   costs nothing at any size that had a region already, so no frame reflows on opening an
   overlay that fitted.

   Reserving it anywhere else was the alternative, and it is worth naming why not. A region
   that only the overlay knows it needs is two components deciding one number: the layout
   was never asked for a region, the overlay concluded it had none, each was locally correct,
   and the frame between them showed an idle session with every key trapped. One number, one
   owner.

   **The floor.** A frame of two rows or fewer has a status bar and a single composer row and
   nothing left to reserve from. There the overlay leaves the region and takes the frame's
   last row, and takes the whole of it — the first content column to the last, under
   **Coverage** below. The single exception to this invariant, and stated here rather than left to the
   code. The rule protects the status bar from an overlay that had somewhere else to go; at
   two rows there is nowhere else, and the choice is not between a correct frame and a
   corrupted one but between a frame that states the question and a frame whose only rows say
   `idle` while every key is trapped. A status bar that is wrong about the state of the
   session has not earned the row it is protected for. Below 40×10 (§1) in any case.

   **Coverage.** An overlay row occupies every cell of its span, from its left edge to its
   right one. The span is the bordered box's width wherever the overlay is inside the region —
   a one-row form stands in for the box and owns exactly the cells the box would have owned —
   and a whole frame row on the floor, where it is replacing one outright rather than sitting
   inside a region. A frame row is the content span, not the terminal's width: the frame keeps
   one blank column at each edge, applied after the last overlay is composited, so the margin
   is outside every span a renderer or an overlay can reach. Invariant 6 states the margin
   in full.

   This is the half of "the floor row spans the frame" that starting at column zero does not
   state, and it is separate from **Width** below because the two rules pull opposite ways:
   the width ladders choose the widest variant that FITS a span, which is almost never a
   variant as wide as the span, and what is spliced in replaces only the cells it occupies.
   An unpadded one-row form therefore leaves the row beneath showing beside it — the status
   bar finishing a confirm prompt with its token count, a composer placeholder running into
   the answer keys, the transcript continuing a help notice — and a row that says two things
   is worse than either of them alone. Bordered forms satisfy this by construction, being
   built at exactly the box width with a border at each end; the one-row forms satisfy it by
   padding at the point they are placed.

   **Width.** Both of an overlay's forms carry the same obligation, and neither may discharge
   it by truncating. The spans that name keys sit at the *end* of a row, which is exactly
   where truncation falls, so every overlay row is a ladder of whole variants rather than one
   long string cut to fit. What a row gives up, in order: the modal footer sheds the version
   and the docs pointer before the exit key, and the confirm prompt sheds the long spelling of
   its answer keys (`[y] yes   [n] no` → `y/n`) before it starts cutting the question, because
   both spellings name the same two keys while a dropped question is a destructive action the
   user cannot see. At 40 columns — 38 of them content, once the frame's margin is
   paid — the longest prompt reads `Discard the running turn an⋯  y/n`: cut, but both legible
   and answerable. `Esc` is not named on the confirm prompt: it and
   `n` do the same thing, so the row already names a way out, and the cells `Esc` would cost
   come straight out of the question.
4. Line counts are identical with and without colour. Only styling degrades.
5. Selection and scroll position are user-owned: streaming content never moves either.
6. **The frame keeps one blank column at each side, and none top or bottom.** ui-spec §2.1.
   Numbered last only because invariants 3 and 4 are cited by number from the code; read it
   alongside invariant 1, because it applies to every row of every grid in this document.

   **What it is.** A terminal `w` columns wide gives `w - 2` columns of content, and every
   renderer in the package measures against that content width. The left column is written
   as a space; the **right one is reserved, not written** — content is bounded at the content
   width and shifted right by one, so nothing can reach the last column, and emitting a space
   there would put trailing whitespace on every row of every frame without changing a single
   rendered cell. Both edges therefore show the terminal's own background, which is what the
   third reading note above requires of the whole frame.

   **Where it is applied.** Once, at the very end, after the last overlay is composited. It
   is the single place in the package that knows the margin exists; every layer beneath it
   renders at the content width and is unaware it is being inset. A renderer that padded
   itself would have to be right about the margin, and there are a dozen of them. It also
   has to run after the overlays rather than before: an overlay's span is the content span
   (invariant 3, **Coverage**), and padding first would make the margin an overlay's to fill.

   **What it does not change.** The minimum terminal is still 40×10 (ui-spec §1), and the
   §2.2 bands are still read from the **terminal** width — a 40-column terminal is a
   full-layout terminal with 38 content columns, not a 38-column one showing
   `terminal too narrow`. The margin is not gated on a band either: the `terminal too
   narrow` notice sits inside the same margin as an 80-column session, so the app does not
   change shape at the moment it is most degraded. It is dropped only at two columns and
   below, where the margin would be the whole frame and the result a blank terminal rather
   than a narrow one — far beneath anything supported, where nothing legible renders either
   way.

   **Why horizontal only.** The chrome budget spends every row the terminal has, so a blank
   row at the top or bottom would come straight out of the transcript; at 40×10 the
   transcript is already down to four rows. A column is cheap at every width this app
   supports; a row is not.
