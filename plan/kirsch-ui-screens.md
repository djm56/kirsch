# Kirsch — TUI screen reference (v0.1)

> **Status: Settled for v0.1.** Companion to [`ui-spec-v0.1.md`](ui-spec-v0.1.md),
> which stays normative: it says what the rules are, this shows what they produce.
> Where a screen and the spec disagree, **the spec wins and the screen is the bug**.
> Built against in [Milestone 0](milestone-0.md) (Tasks 4–7) and held to in
> [Milestone 1](milestone-1.md) (Task 9); registered in the plan's amendment log
> ([`kirsch-plan.md`](kirsch-plan.md) §11, amendment 24).

Every screen below is a literal character grid at **80 columns** (narrow variants noted
per screen). Use these as render targets and as fixtures for the ui-spec §13 golden
tests: the byte-for-byte structure is the contract, colour is applied on top per the
tables at the end.

Screens 00–11 cover thirteen of the fourteen §13 golden states — the §13 table maps each
state to its screen. The fourteenth (onboarding: no API key, not a Git repo) is not drawn
yet: it is unreachable until the provider lands in M3, and its screen is drawn there
before its golden file is captured.

Reading notes:

- Each grid's fence carries its exact terminal size — a fence reading `text 80×18` means
  80 columns by 18 rows. The row count **is** the terminal height, so the golden harness
  reads the size from the fence rather than inferring it. Screen 00 carries none: it is a
  component, not a full-screen render.
- Trailing whitespace cannot survive in a markdown source file, so every grid is stored
  right-trimmed and the golden comparison right-trims both sides before diffing.
- Blank first/last transcript lines are padding inside the transcript viewport, not
  literal blank output — the transcript region flexes to the terminal height.
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
| Body text (user, assistant, tool output) | `text` | 252 | `#d0d0d0` |
| Every separator and unfocused border | `border` | 238 | `#444444` |
| Metadata between `·` delimiters (paths, timings, counts) | `muted` | 244 | `#808080` |
| Placeholders, hints, backgrounded cells | `dim` | 240 | `#585858` |

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

v0.1.0 · terminal-native coding agent

— non-UTF-8 / ASCII fallback —

K I R S C H
v0.1.0 . terminal-native coding agent
```

**Colours**

```text
wordmark            "█▄▀ █ █▀█ █▀▀ █▀▀ █ █" (both rows)      accent    111  #87afff
tagline             "v0.1.0 · terminal-native coding agent"  dim       240  #585858
(ASCII fallback)    same assignment, both rows               accent    111  #87afff
```

- Two rows of half-blocks in `accent` (111). No background fill, so it sits on any host theme.
- 22 columns wide, 2 rows. Non-UTF-8 locale falls back to letter-spaced plain text.
- Shown on the empty session only. It scrolls away with the first message and never reappears.
- Suppressed below 40 cols or 10 rows, where the transcript needs the lines more.

---

## 01 · Empty session (first run)

```text 80×18
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

█▄▀ █ █▀█ █▀▀ █▀▀ █ █
█▀▄ █ █▀▄ ▄▄█ █▄▄ █▀█

v0.1.0 · terminal-native coding agent


Ask anything. Three things to try:

· What does the approval flow do when a patch conflicts?
· Add input validation to Divide and cover it with a test
· Where is the session lock taken?

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · idle · 0 tok
────────────────────────────────────────────────────────────────────────────────
> Ask anything (Enter to send, /help for help)
```

**Colours**

```text
header.title        "Kirsch ─ my-project ─ main"             accent    111  #87afff
header.dirty        "●"                                      warning   179  #dfaf5f
header.rule         trailing "─" fill                        border    238  #444444
wordmark            both block rows                          accent    111  #87afff
tagline             "v0.1.0 · terminal-native coding agent"  dim       240  #585858
lead_in             "Ask anything. Three things to try:"     muted     244  #808080
suggestions         the three "· …" lines                    dim       240  #585858
status.bar          "claude-sonnet-5 · idle · 0 tok"         muted     244  #808080
composer.prompt     ">"                                      accent    111  #87afff
composer.placeholder "Ask anything (Enter to send, …)"       dim       240  #585858
separators          both full-width "─" rules                border    238  #444444
```

- Onboarding, not an error: no border, no red. §7.5.
- Composer placeholder is the literal string from §2.
- Suggestions are `dim` (240) with `·` bullets; the lead-in line is `muted` (244).

---

## 02 · Mid-turn (streaming + running tool)

```text 80×15
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

── you ─────────────────────────────────────────────────────────────────────────
Fix the Divide validation

▸ read_file calc/divide.go · 4ms · ✓ ok
◐ run_command go test ./... · running

I found it in calc/divide.go — the zero check runs after the division,
so the panic fires before validation can return an error. ▌

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · ⠋ running go test · 12.4k tok
────────────────────────────────────────────────────────────────────────────────
>                                                  (input disabled, Esc cancels)
```

**Colours**

```text
header.*            as screen 01
speaker.rule        "── you ───…"                            border    238  #444444
speaker.label       "you"                                    muted     244  #808080
user.text           "Fix the Divide validation"              text      252  #d0d0d0
tool.glyph          "▸"                                      accent    111  #87afff
tool.glyph.running  "◐"                                      accent    111  #87afff
tool.name           "read_file" / "run_command"    text 252 #d0d0d0 + BOLD
tool.args+timing    " calc/divide.go · 4ms · "               muted     244  #808080
tool.status.ok      "✓ ok"                                   success   114  #87d787
tool.status.running "running"                                muted     244  #808080
assistant.text      the two wrapped lines                    text      252  #d0d0d0
cursor              "▌"                                      dim       240  #585858
status.model        "claude-sonnet-5 · "                     muted     244  #808080
status.spinner      "⠋"                                      accent    111  #87afff
status.verb+tokens  "running go test · 12.4k tok"            muted     244  #808080
composer.prompt     ">" (disabled state)                     dim       240  #585858
```

- The spinner appears twice by design: `◐` on the running tool card, and the verb in the
  status bar (`running go test`). §3.8.
- Composer is dimmed and input-disabled while the turn is live; Esc cancels.
- Assistant text soft-wraps at the viewport width. No horizontal scrolling in v0.1 (§2.2).

---

## 03 · Approval — apply_patch (no `[a]`)

```text 80×17
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

▸ search_code "Divide(" · 3 matches · ✓ ok

┃ ┌─ approval required ────────────────────────────────────────────────────────┐
┃ │ apply_patch — add input validation                                         │
┃ │                                                                            │
┃ │ files: 2 changed (calc/divide.go,                                          │
┃ │        calc/divide_test.go)                                                │
┃ │                                                                            │
┃ │ [y] approve   [n] reject   [d] view diff                                   │
┃ └────────────────────────────────────────────────────────────────────────────┘

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · ⠙ awaiting approval · 14.1k tok
────────────────────────────────────────────────────────────────────────────────
>
```

**Colours**

```text
header.*            as screen 01
tool.glyph          "▸"                                      accent    111  #87afff
tool.name           "search_code"                  text 252 #d0d0d0 + BOLD
tool.args           ' "Divide(" · 3 matches · '              muted     244  #808080
tool.status.ok      "✓ ok"                                   success   114  #87d787
gutter              "┃" on every card row                    accent    111  #87afff
card.border         "┌ ─ ┐ │ └ ┘"                            border    238  #444444
card.title          "approval required"                      warning   179  #dfaf5f
card.bg             card interior cells             selectionBg 236 #303030 (background)
card.summary        "apply_patch — add input validation"  text 252 #d0d0d0, tool name BOLD
card.detail         "files: 2 changed (…" both lines         muted     244  #808080
action.approve      "[y]"                                    success   114  #87d787
action.reject       "[n]"                                    error     203  #ff5f5f
action.diff         "[d]"                                    accent    111  #87afff
action.labels       "approve" "reject" "view diff"           text      252  #d0d0d0
status.spinner      "⠙"                                      accent    111  #87afff
status.rest        "claude-sonnet-5 · awaiting approval · …" muted     244  #808080
```

- Three actions only. `[a]` is **never** offered for `apply_patch`. §4.
- `┃` gutter marks selection; the card interior sits on `selectionBg` (236).
- Input capture is exclusive: every key that is not `y`/`n`/`d`/`?`/Esc is swallowed,
  not forwarded to the composer.

---

## 04 · Approval — run_command (with `[a]`)

```text 80×17
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

▸ apply_patch 2 files changed · ✓ approved

┃ ┌─ approval required ────────────────────────────────────────────────────────┐
┃ │ run_command — go test ./...                                                │
┃ │                                                                            │
┃ │ cwd: .        timeout: 60s                                                 │
┃ │ reason: not on allowlist                                                   │
┃ │                                                                            │
┃ │ [y] approve  [a] approve for session  [n] reject  [d] detail               │
┃ └────────────────────────────────────────────────────────────────────────────┘

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · ⠹ awaiting approval · 16.8k tok · 1 grant
────────────────────────────────────────────────────────────────────────────────
>
```

**Colours**

```text
all rows            identical assignment to screen 03
action.session      "[a]"                                    success   114  #87d787
card.detail         "cwd: … timeout: …" / "reason: …"        muted     244  #808080
status.grants       "1 grant"                                muted     244  #808080
(collapsed form)    "✓ approved · session grant: go test"    success   114  #87d787
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
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

── you ─────────────────────────────────────────────────────────────────────────
Fix t┌─ calc/divide.go ─────────────────────────────────────────── +12 −4 ┐
     │ @@ -12,7 +12,15 @@ func Divide(a, b float64)                       │
▸ sea│  func Divide(a, b float64) (float64, error) {                      │
     │ -    return a / b, nil                                             │
┃ ┌─ │ +    if b == 0 {                                                   │─── ┐
┃ │ a│ +        return 0, ErrDivideByZero                                 │    │
┃ │  │ +    }                                                             │    │
┃ │ f│ +    return a / b, nil                                             │    │
┃ │  │  }                                                                 │    │
┃ │  ├────────────────────────────────────────────────────────────────────┤    │
┃ │ [│ j/k scroll · g/G top/bottom · Esc back to approval                 │    │
┃ └──└────────────────────────────────────────────────────────────────────┘────┘

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · ⠸ awaiting approval · 14.1k tok
────────────────────────────────────────────────────────────────────────────────
>
```

**Colours**

```text
modal.border        "┌ ─ ┐ │ ├ ┤ └ ┘" of the modal      borderFocus 111  #87afff
modal.filename      "calc/divide.go"                         accent    111  #87afff
modal.added.count   "+12"                                    success   114  #87d787
modal.removed.count "−4"                                     error     203  #ff5f5f
diff.hunk           lines starting "@@"                      hunk      116  #87d7d7
diff.context        lines starting " " (space)               muted     244  #808080
diff.removed        lines starting "-"                       error     203  #ff5f5f
diff.added          lines starting "+"                       success   114  #87d787
modal.footer        "j/k scroll · g/G … · Esc …"             muted     244  #808080
BACKGROUND CELLS    every cell not owned by the modal        dim       240  #585858
                    (approval card + transcript behind it lose their own colours
                     entirely while the modal is open — one flat dim pass)
status+composer     unchanged, still live                    muted     244  #808080
```

- The modal overwrites cells; the approval card and transcript behind it render at
  `dim` and are not redrawn until it closes.
- Esc returns to the **pending approval**, not the composer. §4.1.
- Modal border uses `borderFocus` (111); unfocused panels use `border` (238).
- Colour comes from parsed diff structure. All ANSI in tool output is stripped first (§5.1).

---

## 06 · Help overlay

```text 80×25
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

 ┌─ help ──────────────────────────────────────────────────────────────────────┐
 │ composing                           approval                                │
 │ Enter        send                   y  approve     n  reject                │
 │ Alt+Enter    newline                a  + session   d  detail                │
 │ Tab          complete /cmd                                                  │
 │ ↑ at line 1  browse                 modal                                   │
 │ Esc          cancel turn            j/k ↑/↓  scroll                         │
 │ q            quit (idle)            g/G      top / bottom                   │
 │                                     Esc      close                          │
 │ browsing                                                                    │
 │ ↑/↓          select card            commands                                │
 │ PgUp/PgDn    scroll                 /help   /status   /diff                 │
 │ Enter        expand                 /files  /approvals                      │
 │ d            diff / content         /new    /compact  /quit                 │
 │ End          bottom, re-pin                                                 │
 ├─────────────────────────────────────────────────────────────────────────────┤
 │ kirsch v0.1.0 · docs: doc/usage.md · Esc or ? closes                        │
 └─────────────────────────────────────────────────────────────────────────────┘

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · idle · 12.4k tok
────────────────────────────────────────────────────────────────────────────────
>
```

**Colours**

```text
modal.border        box borders + "├ ┤" divider         borderFocus 111  #87afff
modal.title         "help"                                   accent    111  #87afff
section.headings    "composing" "browsing" "approval"        warning   179  #dfaf5f
                    "modal" "commands"
bindings            key + description columns                muted     244  #808080
footer              "kirsch v0.1.0 · docs: … · Esc or ? closes"  dim   240  #585858
BACKGROUND CELLS    header row behind the overlay            dim       240  #585858
```

- Opens from Browsing and ApprovalPending only. In the composer, `?` is a literal character. §4.2.
- One screen, no scrolling: bindings by mode, then slash commands, then the version footer.
- Mode headings are `warning` (179); bindings are `muted` (244).

---

## 07 · Tool card expanded at the 200-line cap

```text 80×17
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

┃ ▾ run_command go test ./... · 2.4s · ✗ exit 1
┃   ┌──────────────────────────────────────────────────────────────────────┐
┃   │ === RUN   TestDivide                                                 │
┃   │     divide_test.go:31: Divide(1, 0) = +Inf, want ErrDivideByZero     │
┃   │ --- FAIL: TestDivide (0.00s)                                         │
┃   │ === RUN   TestDivide_Table                                           │
┃   │ --- PASS: TestDivide_Table (0.00s)                                   │
┃   │ FAIL    example.com/calc    0.004s                                   │
┃   └──────────────────────────────────────────────────────────────────────┘
┃   ‹200 of 4,181 lines — press d for full output›

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · idle · 22.9k tok · 1 grant
────────────────────────────────────────────────────────────────────────────────
> ▌
```

**Colours**

```text
gutter              "┃" on every card row                    accent    111  #87afff
tool.glyph          "▾" (expanded)                           accent    111  #87afff
tool.name           "run_command"                  text 252 #d0d0d0 + BOLD
tool.args+timing    " go test ./... · 2.4s · "               muted     244  #808080
tool.status.fail    "✗ exit 1"                               error     203  #ff5f5f
output.border       inner "┌ ─ ┐ │ └ ┘"                      border    238  #444444
output.bg           inner panel cells                 codeBg    235 #262626 (background)
output.text         captured stdout/stderr verbatim          text      252  #d0d0d0
cap.marker          "‹200 of 4,181 lines — press d …›"       warning   179  #dfaf5f
card.bg             card interior (selected)         selectionBg 236 #303030 (background)
NOTE                captured output is NOT syntax-coloured; ANSI is stripped (§5.1)
```

- A command failure the model can handle stays a tool card with `✗` — it is **not** promoted
  to an error card. §3.6.
- Output panel sits on `codeBg` (235) with indentation preserved.
- Cap marker in `warning` (179). `d` opens the full output in a content modal.

---

## 08 · Error card + system notices

```text 80×14
Kirsch ─ my-project ─ main ● (compacted) ───────────────────────────────────────

· session recovered — 3 events after a torn line were discarded
· compacted 94 events into a summary · 12 files touched this session

  ┌─ provider_error ───────────────────────────────────────────────────────────┐
  │ anthropic: 503 after 3 retries — request not sent                          │
  │ Enter to expand · the turn is still cancellable                            │
  └────────────────────────────────────────────────────────────────────────────┘

────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · idle · 31.2k tok                           ⚠ recovered session
────────────────────────────────────────────────────────────────────────────────
> ▌
```

**Colours**

```text
header.compacted    "(compacted)"                            muted     244  #808080
notices             both "· …" lines                         dim       240  #585858
error.border        "┌ ─ ┐ │ └ ┘" of the error card          error     203  #ff5f5f
error.title         "provider_error"                         error     203  #ff5f5f
error.message       "anthropic: 503 after 3 retries — …"     text      252  #d0d0d0
error.hint          "Enter to expand · the turn is still …"  muted     244  #808080
status.left         "claude-sonnet-5 · idle · 31.2k tok"     muted     244  #808080
status.warning      "⚠ recovered session"                    warning   179  #dfaf5f
NOTE                the error card sets NO background — border and title only
```

- Error cards are reserved for infrastructure failures (provider, session, transport), so red
  borders stay rare enough to mean something. §3.6.
- Notices are one `dim` line with a `·` prefix and no border. §3.7.
- Persistent conditions live in the status bar and are never truncated away.

---

## 09 · Scrolled up (unpinned)

```text 80×14
Kirsch ─ my-project ─ main ● ───────────────────────────────────────────────────

┃ ▸ read_file internal/session/store.go:1-120 · 3ms · ✓ ok

The lock is taken in Open, before auto-resume reads index.json, so a
second instance starts a fresh session instead of adopting the first.

▸ git_diff 2 files changed · 8ms · ✓ ok

                                                                        ↓ 3 new
────────────────────────────────────────────────────────────────────────────────
claude-sonnet-5 · ⠼ thinking · 28.0k tok
────────────────────────────────────────────────────────────────────────────────
> also check the second-instance path▌
```

**Colours**

```text
gutter              "┃" (selected card)                      accent    111  #87afff
card.bg             selected card row               selectionBg 236 #303030 (background)
tool.glyph          "▸"                                      accent    111  #87afff
tool.name           "read_file" / "git_diff"       text 252 #d0d0d0 + BOLD
tool.args+timing    " internal/session/store.go:1-120 · 3ms · "  muted 244  #808080
tool.status.ok      "✓ ok"                                   success   114  #87d787
assistant.text      the two wrapped lines                    text      252  #d0d0d0
new.counter         "↓ 3 new"                                accent    111  #87afff
composer.prompt     ">"                                      accent    111  #87afff
composer.text       "also check the second-instance path"    text      252  #d0d0d0
cursor              "▌"                                      dim       240  #585858
```

- Typing while unpinned does not yank the view to the bottom; the counter keeps rising. §2.4.
- `↓ n new` renders on the last transcript row, right-aligned, in `accent` (111).
- Selection stays on the card the user chose. New content never steals it. §3.9.
- `End` jumps to the bottom and re-pins.

---

## 10 · Narrow and short terminals

**40 columns** — the minimum supported width:

```text 40×11
Kirsch ─ my-project ────────────────────

▸ read_file calc/divide.go · ✓

Soft wrap only. No horizontal
scrolling in v0.1. ▌

────────────────────────────────────────
sonnet-5 · ⠋ thinking
────────────────────────────────────────
> ▌
```

**38 columns** — below the transcript threshold:

```text 38×6
terminal too narrow

──────────────────────────────────────
⠋                                    ⚠
──────────────────────────────────────
> ▌
```

**6 rows** — below the header threshold:

```text 40×6
no header at 6 rows;
transcript keeps 2 lines
────────────────────────────────────────
sonnet-5 · idle
────────────────────────────────────────
> ▌
```

**Colours**

```text
(applies to all three widths)
header.title        "Kirsch ─ my-project"                    accent    111  #87afff
too_narrow.notice   "terminal too narrow"                    dim       240  #585858
status.spinner      "⠋"                                      accent    111  #87afff
status.rest         "sonnet-5 · thinking" / "idle"           muted     244  #808080
status.warning      "⚠" (38-col form, glyph only)            warning   179  #dfaf5f
tool.status.ok      "✓" (glyph only, label dropped)          success   114  #87d787
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
Kirsch - my-project - main * ---------------------------------------------------

> search_code "Divide(" . 3 matches . [ok]

| +- approval required --------------------------------------------------------+
| | apply_patch - add input validation                                         |
| |                                                                            |
| | files: 2 changed (calc/divide.go,                                          |
| |        calc/divide_test.go)                                                |
| |                                                                            |
| | [y] approve   [n] reject   [d] view diff                                   |
| +----------------------------------------------------------------------------+

--------------------------------------------------------------------------------
claude-sonnet-5 . \ awaiting approval . 14.1k tok
--------------------------------------------------------------------------------
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
| `accent` | 111 | `#87afff` | Header title, tool glyphs, selection gutter, focused border, `↓ n new`, `[d]` |
| `text` | 252 | `#d0d0d0` | Assistant and user body text, tool output, action labels |
| `muted` | 244 | `#808080` | Metadata (paths, timings, counts), key bindings, status bar |
| `dim` | 240 | `#585858` | Placeholders, suggestions, backgrounded cells, version footer, cursor |
| `success` | 114 | `#87d787` | `✓`, `[y]`, `[a]`, diff additions, grant confirmations |
| `error` | 203 | `#ff5f5f` | `✗`, `[n]`, diff deletions, error card border and title |
| `warning` | 179 | `#dfaf5f` | Dirty marker, `approval required`, cap marker, status-bar warnings |
| `hunk` | 116 | `#87d7d7` | Diff `@@` hunk headers |
| `border` | 238 | `#444444` | Unfocused borders, all separators |
| `borderFocus` | 111 | `#87afff` | Focused modal border |
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
2. The header, status bar and composer are one row each. The composer grows to a maximum of
   5 rows on `Alt+Enter`, taking rows from the transcript.
3. Modals overwrite the transcript region only. The status bar and composer stay visible.
4. Line counts are identical with and without colour. Only styling degrades.
5. Selection and scroll position are user-owned: streaming content never moves either.
