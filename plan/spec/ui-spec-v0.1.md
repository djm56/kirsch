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
 Kirsch ───────────────────────────────────────   ① header row 1 — wordmark + rule
 my-project ─ main ●                              ① header row 2 — session line

                                                  ② transcript (fills)
 ── you ──────────────────────
 Fix the Divide validation

 ▸ read_file calc/divide.go · 4ms · ok            tool card (collapsed)

 I found the issue in... ▌                        streaming assistant

 ──────────────────────────────────────────────   separator rule
 claude-sonnet-5 · ⠋ thinking · 12.4k tok         ③ status bar (1 line)
 ──────────────────────────────────────────────   separator rule
 > _                                              ④ composer (1–5 lines)
^                                              ^
one blank column here, and one here — the frame margin, §2.1 (the scrollbar may take the right one)
```

There is no box around the frame. Every row begins one column in from the left
edge and stops one column short of the right — the frame margin of §2.1, which the
scrollbar may take on transcript rows — and nothing else paints an outer border.

- **① Header — two rows.** Row 1 is the wordmark `Kirsch` alone, followed by a
  rule running to the last content column. Row 2 is the session line:
  `<project name>[ ─ <branch>][ ● if dirty][ (compacted)]`. Project name is the
  workspace directory's base name; below 60 columns §2.2 drops the branch and
  the dirty marker goes with it. Only row 1 carries a rule — a second full-width
  rule directly beneath the first would read as the top edge of a box rather
  than as a header. The header is shown or hidden **as a unit**, so the height
  ladder frees two rows at the step that drops it.
- **② Transcript viewport** — all conversation history. Scroll and pin rules
  in §2.4.
- **③ Status bar** — §2.3.
- **④ Composer** — multiline text area, 1–5 lines then internal scroll.
  Placeholder `Ask anything (Enter to send, /help for help)` while the onboarding
  or the empty-session screen is showing (§7.5), and not otherwise. Below 40
  columns, or when the transcript region has no rows, the empty session shows no
  placeholder.
  Dimmed, and ordinary typing is refused while a turn is in flight; `Esc`/`Ctrl+C`
  cancels the turn and re-enables it. Some keys still act during a turn; §5.2 lists them.

### 2.1 Sizing rules

- The header is fixed at **2 lines**; the status bar and each separator rule are
  fixed at 1. Composer is its content height clamped to 1–5. A key-hint row appears
  at terminal height ≥ 24, fixed at 1. Transcript takes everything left over.
- **Frame margin.** The frame keeps **one blank column at its left edge and one
  at its right**, and none at the top or bottom. The right one carries the scrollbar on
  transcript rows when the bar is drawn (below). Horizontal only, and
  deliberately so: the chrome budget in §2.2 spends every row the terminal has,
  so a blank row would come straight out of the transcript, which at the
  advertised 40×10 minimum is already down to four rows. A column is cheap at
  every width this app supports; a row is not.
  - **The minimum terminal is unchanged at 40×10** (§1). The margin does not
    raise it.
  - **Two widths, and they are not interchangeable.** The *terminal* width is
    what the §2.2 bands are read from. Every renderer draws into the content width
    alone; the margin columns on either side belong to the frame. The *content*
    width — terminal width less both margin columns — is what every renderer
    measures against, and the only width any of them sees. A 40-column terminal
    has 38 content columns. Banding on content width would make a 40-column
    terminal render the `terminal too narrow` notice and retire the advertised
    minimum as a side effect of adding a margin; the same reasoning holds at 60
    and 80, because a band is a statement about the terminal the user has.
  - **The right column is reserved, not written, except for the scrollbar.**
    Content is bounded at the content width and shifted right by one, so nothing
    can reach the last column. No space is emitted into it on a frame without the
    bar — writing one would put trailing whitespace on every row of every frame
    without changing a single rendered cell. On a frame without the bar the right
    column carries the terminal's own background, which is what §1's "the
    terminal's own background shows through" requires.
  - **The scrollbar is an exception to the blank margin (v0.1.5, amendment 103).**
    Its column is the right margin column, and the frame draws the bar into it. The
    bar is drawn after the left margin is inserted, on the transcript's rows only. The
    bar is drawn exactly when `scrollbarDrawn` returns true,
    and otherwise the column stays blank. The conditions live in `scrollbarDrawn` alone,
    and this spec does not restate them.
    The content width does not change, and every renderer still
    measures against the same content width. The left margin is not a condition of
    its own: `framePadding` gives no margin only below a 3-column terminal, and
    `ShowTranscript` is never set that low, so the margin is present whenever the bar
    can be drawn.
    - **Padding.** On a frame that draws the bar, each transcript row is padded to
      the content width with the ANSI-aware `pad`
      before the glyph is placed in the last column. The padding writes spaces into
      content cells that the row would otherwise leave empty. This is the deliberate
      trade against the trailing-whitespace rationale above: the row still ends in its
      glyph, so it has no trailing whitespace, but its empty cells are written rather
      than left. Frames without the bar are not padded.
    - **Flush against the bar.** The `↓ N new` row (§2.4) and full-width rows, such
      as a card border that runs to the last content column, end at the last content
      column, so they sit flush against the bar. That is intended. Nothing separates
      the bar from the content.
    - **Where it is drawn.** `View()` composites the overlays (the modal is
      composited by `overlayModal`) and then makes the frame call, which runs
      `padFrame` to insert the left margin. The bar is drawn on that same pass,
      after the insert.
  - **Applied once, last**, after the final overlay is composited. No renderer
    below the frame step knows the margin exists; an overlay's span is the content
    span, never the terminal's width (screens, layout invariant 3). The scrollbar is
    drawn into the margin by the frame step (above).
  - The margin is not gated on a §2.2 band: the 38-column `terminal too narrow`
    notice sits inside the same margin as an 80-column session, so the app does
    not change shape at the moment it is most degraded. It is dropped only at
    two columns and below, where the margin would be the whole frame; that seam
    sits far beneath the supported minimum, where nothing legible renders either
    way.
- All width math is rune-aware (`go-runewidth`). A rune is not a cell: CJK and
  emoji are width 2, combining marks are width 0. Assuming otherwise corrupts
  every border on the screen.
- `View()` guards zero and negative dimensions and renders nothing rather than
  panicking.

### 2.2 Responsive breakpoints

| Width | Behaviour |
|---|---|
| ≥ 80 | Full layout. Modals at 80% of viewport. |
| 60–79 | Status bar drops the token count: no token figure at all, not even `N tok`. Modals at 90%. |
| 40–59 | Header shows project name only (no branch). Tool cards drop duration. Status bar shows no token figure. Modals full-screen. |
| < 40 | Transcript hidden. Status bar + composer only, with a dim `terminal too narrow` notice, and no token figure. No panic. |

Token figures appear only at 80 columns and wider. Below 80 the status bar never shows a token count, not even the session total `N tok`.

| Height | Behaviour |
|---|---|
| ≥ 10 | Full layout. |
| 6–9 | Header hidden; transcript shrinks to at least 2 lines. |
| 5 | Header hidden; transcript 1 line. |
| < 5 | Status bar + composer only; both separator rules dropped. |
| ≤ 0 either axis | Render empty string. |

The width bands are read from the **terminal** width, the height bands from the terminal
height; neither is read from the content width the margin leaves (§2.1).

The bands follow from the chrome budget, so change them together. Chrome is header (2) +
separator (1) + status (1) + separator (1) + composer (1) = **6 rows with a header, 4
without**. So h=10 leaves 4 transcript rows, h=9 leaves 5, h=6 leaves 2, and h=5 leaves 1 —
which is why the "at least 2 lines" band starts at 6, not 5. Degrade in this order, never
another: composer (never dropped, minimum 1) → status bar (never dropped) → the two
separator rules (dropped **as a pair**, so the bar is never half-framed) → header →
transcript remainder.

**The header band still starts at h=10, and the transcript is briefly larger without it.**
§1 names 40×10 as the smallest supported terminal, and the smallest supported terminal is
exactly where the full layout has to still be the full layout — raising the band to 11 so
the two-row header could keep a five-row transcript would mean the advertised minimum no
longer renders a header at all. The cost is the one discontinuity above: shrinking from 10
rows to 9 *grows* the transcript from 4 rows to 5, as the two-row header goes and only
four rows of chrome remain. That is unavoidable once the header is a two-row unit; moving
the band relocates it rather than removing it.

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
(`sonnet-5`) → state verb reduced to the bare spinner. The token count is only
present at 80 columns and wider (§2.2), so below 80 the order starts at the
grants indicator. **Warnings are never
dropped**; at narrow widths they are the entire reason the bar exists.

### 2.4 Scroll and pinning

- The transcript is **pinned to the bottom** by default. New content while
  pinned keeps it pinned.
- Scrolling up **unpins**. A `↓ 3 new` indicator appears bottom-right of the
  viewport and counts blocks arrived since unpinning.
- Re-pin on any of: scrolling back to the bottom, `End` or `G` in Browsing
  mode, **`Ctrl+G` in Composing mode**, sending a message, or submitting a slash
  command.
- **`Ctrl+G` is the way back to the bottom from the composer.** It re-pins
  without leaving Composing and without putting anything in the transcript. A
  reader who scrolled up and then decided not to send needs a key that returns the
  view without sending. `PgDn` from the composer also re-pins, but only when its
  page reaches the bottom. `Ctrl+G` moves the viewport only: the composer's
  contents, its hint and the selection are all left alone, and the mode does
  not change. It works **while a turn is running**, as do `PgUp` and `PgDn`: a
  transcript streaming past is when getting back to the bottom is worth the most.
  `Ctrl+G` rather than `End` or `Ctrl+End`: all three spell "go to the bottom",
  but bubbles v1.0.0 already binds `End` to LineEnd and `Ctrl+End` to InputEnd
  in the textarea, so taking either would buy a scroll at the price of a cursor
  movement in a composer that can be five rows tall. `Ctrl+G` is also the
  sturdiest on the wire — it is BEL, a single C0 byte, where `Ctrl+End` is a
  modified-key escape sequence, and §5.2's note on `g`/`G` records that this
  family of keys is exactly where terminal disagreement lands. Bare `G` was
  never a candidate: it has to stay an ordinary character while typing.
- **A slash command re-pins.** Submitting one is a request, and §3.1 renders the
  invocation as a message, so the transcript goes to the bottom where its answer
  will be. The hint-only path is the exception — an unknown command, `/copy`, and `c`
  answer entirely in the dim hint under the composer and put nothing in the
  transcript, so re-pinning would cost the reader their scroll position to report
  a typo or a copy.
- **`Esc` does not re-pin.** Leaving Browsing hands focus back to the composer and
  leaves the viewport exactly where it is. Re-pinning there would make the rule
  below unreachable in practice: `Esc` is the key a reader takes back to the
  composer, so scrolling up to read something would be undone by the act of going
  to type about it. `Tab`, and the `↓` that moves past the last selectable item, hand focus
  back the same way and leave the offset alone. Walking the selection down with
  `↓` is different: each step scrolls only as far as needed to show the selected
  item, and if that scroll reaches the bottom it re-pins. So the walk can re-pin,
  and the keypress that returns focus cannot. (Amendment 100 corrected the earlier
  claim that `Esc` was the only such route.)
- **Typing does not re-pin.** Composing a message while reading scrollback must
  not yank the view to the bottom — the user is reading it deliberately.
- `PgUp`/`PgDn` scroll a page without moving the selection, in Browsing and
  in Composing. In Composing they run ahead of the busy guard, like `Ctrl+G`, so
  they work while a turn is running, and they do not change the mode. `j`/`k` in
  Browsing move one row and stop at the top and bottom. A page is the transcript
  height minus one row, and at least one row, the same in Browsing and Composing.
  Like `Ctrl+G`, `PgUp`/`PgDn` leave the composer's text and hint alone. Scrolling
  up unpins; scrolling down to the bottom re-pins, and scrolling back to the bottom
  clears the `↓ N new` indicator. `↑`/`↓` move the selection and scroll only
  as far as needed to keep it visible. Scrolling and selection are separate
  concerns and must not be collapsed into one.
- **The scrollbar (v0.1.5, amendment 103).** A one-column position indicator in
  the right margin (§2.1). It shows where the viewport sits in the transcript.
  - Inputs. `h` is the transcript height (`lay.TranscriptH`). `n` is the
    transcript's line count, `len(m.lines)`. `maxOff`
    is `n − h`, and `off` is `clamp(m.scroll.Offset, 0, maxOff)`. `setOffset` keeps
    the offset between `0` and `maxOff` when it is called, but not every path calls
    it. An unpinned resize only relayouts (the `tea.WindowSizeMsg` case of `update`),
    and expanding a non-tool item in `keyBrowsing` does not clamp, so the stored offset
    can sit above `maxOff`. The clamp keeps the thumb inside the track in that case.
    The pinned offset is `n − h`, re-derived only while pinned (in `relayout`).
  - Drawn exactly when `scrollbarDrawn` returns true. Otherwise the column is blank.
  - Thumb size, integer division: `thumbH = max(1, h*h / n)`.
  - Thumb top, integer division: `thumbTop = (off*(h − thumbH) + maxOff/2) / maxOff`.
    Transcript row `r` shows the thumb when `thumbTop ≤ r < thumbTop + thumbH`, and
    the track otherwise. Row numbers in this section, and in screen 14, count from
    0 at the top of the transcript.
  - Both ends. Offset `0` puts the thumb at the top. The pinned offset puts it flush
    at the bottom, because `maxOff/2` is below `maxOff` and adds nothing to the last
    step. Rounding can also put the thumb at either end for offsets near them, so the
    thumb at the bottom does not by itself mean pinned. For example, with `h` 10 and
    `n` 40 (`maxOff` 30, `thumbH` 2): offsets 0 and 1 both give `thumbTop` 0, and
    offsets 29 and 30 both give `thumbTop` 8, the bottom two rows. Offset 29 is not
    pinned.
  - Worked example, `h` 10 and `n` 40: `maxOff` is 30 and `thumbH` is 2. Offset 0
    gives `thumbTop` 0. Offset 15 gives `(15·8 + 15)/30` = 4. The pinned offset 30
    gives `(30·8 + 15)/30` = 8, so the thumb takes the last two rows, 8 and 9.
  - Worked example, `h` 12 and `n` 100: `maxOff` is 88 and `thumbH` is
    `max(1, 144/100)` = 1. Offset 44 gives `(44·11 + 44)/88` = 6. The pinned offset
    88 gives `(88·11 + 44)/88` = 11, row 11, the last row.
  - Colour roles. The track is `dim`, the decoration role (§10.1). The thumb is
    `muted`, which has the higher contrast of the two (§10.1 gives each role's
    contrast), so the position stays readable. No palette entry is added. The thumb is identified by
    its glyph, not only its colour (§12). With colour off the glyphs are drawn and
    no SGR is emitted (§13).
  - No overlay. While an overlay is open the transcript is drawn with the flattened
    styles (`m.sty.Flat()`, called in `relayout`), and the bar is not drawn
    at all. `Flat()` gives `muted` and `dim` the same style, so a thumb under an
    overlay could be told apart from the track by its glyph alone. The bar is left out
    instead. The margin is outside every overlay's span, because the frame step runs
    after the overlays in `View`.
  - The "↓ N new" indicator (in `transcriptRows`) sits on the last
    transcript row and ends at the last content column.
    The bar sits in the margin beside it, so they meet, flush, and do not overlap.
    That is intended (§2.1).
  - No mouse, no click and no drag. Those are milestone items 3 and 7, deferred in
    `plan/milestones/ux-round-v0.1.md`.

## 3. Transcript Elements

### 3.1 User message

```
── you ─────────────────────────
Fix the Divide validation
```

Verbatim, soft-wrapped. Slash-command invocations render the same way; their
result appears as a system notice or a card.

### 3.2 Assistant message

Streams with a `▌` caret; caret removed on completion. Soft-wrapped. (v0.1.4, amendment 102) While a reply streams it is shown plain and soft-wrapped, with the caret, as before. A finished reply is rendered as markdown when `ui.render_markdown` is true (the default; plan §5) and the reply is not streaming. Colour does not gate rendering. Whether colour is on is decided per `DetectCaps` (§7.3), and it affects styling only. A reply cancelled mid-turn, or one whose turn ended in an error, is rendered as it stands. Replies loaded on resume are rendered. When `ui.render_markdown` is false, the reply shows exactly as in v0.1.3: raw markdown text with no structural changes. When colour is unavailable, a rendered reply keeps its structure, wrapping and line count, with no styling. That is the unstyled form that `NO_COLOR` and `TERM=dumb` produce. Reading "fall back to plain text" as "unstyled text" is Bishop's reading, made under the operator's standing go-ahead and open to being overturned. The unstyled form is the same rendering code run with identity styles, as for every other item (the double pass in `relayout`), so `strip(styled) == plain` holds by construction (§13). User messages, cards, notices and the status bar are never rendered. (v0.1.4, amendment 102) A reply that finishes streaming re-flows once, from plain to rendered.

(v0.1.4, amendment 102) Each element renders as the table below. The ASCII fallback applies when the locale is not UTF-8 (§10.2).

| Element | Rendered form | ASCII fallback |
|---|---|---|
| Heading | Text in bold `accent`, with its `#` markers kept | As rendered |
| Bold | Bold, with its `**` markers dropped | As rendered |
| Italic | Italic, with its `*` markers dropped | As rendered |
| Strikethrough | Text between its `~~` markers, struck through | As rendered |
| Inline code | Text in `synString`, with its backticks kept; no background | As rendered |
| List item | Bullet `•`; numbered items keep their numbers; nested lists indent by two columns | Bullet `-` |
| Block quote | Prefix `│ ` in `muted` | Prefix `\| ` (a pipe and a space) |
| Link | The link text, then the URL in `muted` in parentheses; a bare URL is shown as written | As rendered |
| Horizontal rule | A rule of `─` in `border` across the item width | `-` across the item width |
| Fenced code block | Content indented two columns, its indentation kept exactly relative to the block; fence lines and the info string not shown; highlighted (below); no background; lines wrap at the item width, and a wrapped row keeps the block's indent | As rendered |
| Table | Columns aligned; `border` separators: `│` between columns, `─` under the header row | `\|` between columns, `-` under the header row |
| Raw HTML | Shown as literal text | As rendered |
| Image | The alt text | As rendered |

(v0.1.4, amendment 102) The item's width is the content width (§2.1), less two columns when the reply is selected (§3.9). Rendered output wraps to that width. Post-condition: no rendered row exceeds the item's width. Wide tables, long words and long URLs are hard-wrapped at the width. A wrapped continuation row keeps its block's indent.

(v0.1.4, amendment 102) Any other element is drawn as its text content in `text`, with any markdown markers it has kept, within the width, background and SGR rules in this section and §13.

(v0.1.4, amendment 102) Known gap, list numbering: a numbered item takes the number after the item before it, not the number written in its own source line, when the text before its first content does not end in digits, then `.` or `)`, then spaces. That text is read from the source line on which the first content starts. The first item takes the list's start number. For example, the first content is a heading, a block quote, a fenced code block, a nested list, or nothing at all, or the content starts on the line after an empty marker. A thematic break as the first content falls under the same rule. For example, `1. # h` followed by `5. # i` draws as `1. # h` and then `2. # i`.

(v0.1.4, amendment 102) In a rendered reply, headings keep their `#` markers, inline code keeps its backticks, and strikethrough keeps its `~~` markers, in every mode. The styling is applied on top when colour is available. So with colour off, no meaning is lost to a missing marker, and the identical-layout invariant still holds (§13). Bold and italic drop their `**` and `*` markers. With colour off, bold and italic text shows as plain text; that is a known gap (plan amendment 102).

(v0.1.4, amendment 102) No background is painted anywhere in a rendered reply, and no margins are added. Blocks are separated by one blank row, and runs of blank lines in the source collapse to one. That separator is not a margin. A block's indent, the two-column fence indent or list nesting, is not a margin either.

(v0.1.4, amendment 102) Rendering works from the reply's sanitised lines (§7.1), never from the unsanitised source, so model text cannot inject escape sequences. Only Kirsch's own styling sequences reach the screen.

(v0.1.4, amendment 102) The parse and the layout run at most once per consecutive run at the same width and glyph set (Unicode or ASCII): the cache holds the laid-out runs, each tagged with its §10.1 role and attributes, with no styling applied. Every drawing pass, whether live, flattened under an overlay, or identity for the plain form, draws from the same cached runs through its own `Styles`. Relayout runs on every update, so the cache is what keeps a long reply cheap. The cache key includes the width the reply renders at, which is the selected width (two columns narrower, above) when the reply is selected. The cache key also includes the reply's text, so a changed reply is laid out again. Only the latest width is kept for each reply, so a width that comes back after another is laid out again.

(v0.1.4, amendment 102) `c` and `/copy` still copy the source text (§6), not the rendered text. Code-block numbering for `/copy code N` follows the §6 grammar on the cleaned payload text (§6), whatever the rendering shows.

(v0.1.4, amendment 102) Highlighting is by language, foreground only, with one colour scheme for reply fences and the modals (§4.1). Token mapping: keywords, including keyword constants such as `true`, `nil` and `null`, → `synKeyword`; strings → `synString`; numbers and other literals → `synNumber`; type names, class names and keyword types → `synType`; function names and builtins → `accent`; comments → `muted`; everything else, operators and punctuation included, → `text`. `success` and `error` are never used for tokens, so a token cannot be mistaken for a diff marker. The `synKeyword`, `synString`, `synNumber` and `synType` roles are in §10.1. No background is painted. In a reply fence the language comes from the info string; an unknown or missing language is shown in `text`. With colour off, code is plain.

(v0.1.4, amendment 102) Preprocessor directives (chroma's `CommentPreproc` and `CommentPreprocFile` token types) draw in `text`, not `muted`, so a directive is not read as commented out.

(v0.1.4, amendment 102) `NameConstant` draws as `synKeyword`. That covers `true`, `nil` and `NULL` in the lexers that tag them as names. It also covers some names that are not keyword constants: in some lexers, for example Ruby's capitalised names and all-caps names in Rust, Perl and Crystal. Those draw as keywords too. This is a known gap: a name that is not a constant can read as a keyword.

(v0.1.4, amendment 102) Headings stay bold `accent`. A link's URL is `muted`, not `dim`, because a URL is unique information (§12 rule 5).

(v0.1.3 and earlier showed the reply as raw text: markdown markers appeared as typed.)

### 3.3 Tool card

**Collapsed (default):**

```
▸ read_file calc/divide.go · 4ms · ok
```

Glyph, bold name, dimmed target, result summary, duration. Tool-specific
summaries:

| Tool | Summary |
|---|---|
| `read_file` | `lines N-M` |
| `list_files` | path + `n files` |
| `search_code` | query + `n matches` |
| `apply_patch` | `n files changed` + status |
| `run_command` | command + exit status |
| `git_status` / `git_diff` | changed-file count |

**Preview (default):** first 10 lines in a fenced panel. When lines are hidden, a
muted marker reads `‹10 of 4,181 lines — d full output · Enter collapse›`. `Enter`
toggles between preview and collapsed (head only). `d` opens the content modal
(§4.1) for everything. An unbounded inline expansion makes the scrollback unusable,
which is a worse failure than truncating it. Empty output shows no box. The trailing
empty output row is trimmed.

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

- `y` → approved; once you approve, the approval card folds away. The result card's
  head shows ` · approved` or ` · approved for session`, plus `session grant: <scope>`.
  `d` on the folded patch card opens the diff. Rejected and cancelled approvals keep
  their own card.
- `a` → approved **and** a session grant recorded for the argv prefix shown on
  the card (plan §4). Collapses to `✓ approved · session grant: go test`.
  **Never offered for `apply_patch`**, never for `sh -c`.
- `n` → rejected; returned to the model as a tool result so it can adapt.
- A refused session grant now confirms as approved (allow-once), because the command runs.
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
output, session recovery points, unknown-command hints, and replies cut off at
the output limit.

```
· session recovered — 3 events after a torn line were discarded
```

A reply that stops because it reached the output limit gets a notice directly
under it, exactly `· reply cut off — output limit reached (N tokens)`, where N
is the `max_tokens` value sent with the request. N is a plain integer with no
separators (`16384`). The notice appears whenever the stop reason is `max_tokens`,
including when the reply text is empty or the reply ends in tool calls. It sits
after the reply's text and before any tool cards from the same message. It is shown
live and on resume, with one stated exception. A reply made only of thinking
blocks that is cut off shows the notice live but not on resume, because no
`assistant.message` line is written for it (pinned by
`TestThinkingOnlyReplyCutOffShowsNoticeLiveButNotOnResume`, `internal/app/cutoff_notice_test.go`).
A session written before v0.1.2 has no stored stop reason and
shows no notice on resume (`assistant.message` fields, plan §7, amendment 100).
Every stored `stop_reason` comes with its `max_tokens`. If a stored message has
`max_tokens` as its stop reason but no count, the notice is shown without it,
exactly `· reply cut off — output limit reached`.

The fallbacks for `·` and `—` are in §10.2: on a non-UTF-8 locale the leading `·`
is drawn as `.` and the `—` as `-`.

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

A card never disappears or is rewritten in place once terminal, except an
approved approval card that is linked to its tool card: it folds away when the
approval is resolved as approved or approved for session. Partial assistant text
from a cancelled turn **stays** in the transcript, marked cancelled — erasing what
the user watched arrive is worse than leaving it.

### 3.9 Selection

Selectable items are **cards and assistant replies**. A reply is one assistant text block in the transcript
(v0.1.3, amendment 101). User messages are skipped by `↑`/`↓`, because nothing
can be done with them. A reply with no lines is skipped. `Enter` and `d` do nothing on a reply (v0.1.3, amendment 101).

The selected card or reply shows an accent-colored `┃` in the left gutter, on every
row of the item, whenever it is the selection, in every mode, Composing included, exactly as for a selected card; its 2-column-narrower wrap holds for as long as it stays selected (v0.1.3, amendment 101).
The gutter takes 2 columns. Selection survives new content arriving; it does not jump to the newest item.
Entering Browsing keeps the current selection; when nothing is selected, it selects the last selectable item, which may now be a reply.

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
  muted (corrected, amendment 102: the code has always drawn context lines
  muted, and this text said dimmed); header shows per-file stats (`+12 −4`).
  No-color fallback uses the prefix characters alone.
- **Plain content** renders with line numbers.
- (v0.1.4, amendment 102) **Diff content** is highlighted by language, with the scheme in §3.2. The diff modal's feeders are below; each sets the language from its own lines.
  - **Patch approval diff**, opened from a patch approval card or from a folded approval's tool card. Its lines are hunks only, starting at the first `@@`, with no `diff --git`, `---` or `+++` lines. It may also hold single descriptive lines for a file with no hunks (renamed, empty file created or deleted, binary, mode change) and `\ No newline at end of file` lines; none begins with `+` or `-`, so none is highlighted. The language comes from the file path in the modal's title. For a rename the title is `old → new`, and the new path, after ` → `, is used.
  - **`/diff`**, titled "working tree". Its lines are full `git diff` output, which may cover several files. The language is set per file section, from the `b/` path of that section's `diff --git a/… b/…` line. The `b/` path is present for deleted files too. When a section's `diff --git` line does not parse as `a/<path> b/<path>` with one unambiguous ` b/` split, as with git's mnemonic or no-prefix settings or with quoted paths, that section is not highlighted. A truncation marker, such as `‹… output truncated …›` or the trailing `⋯ truncated` marker, ends the current section's highlighting, and a file header run ends there too (`internal/tui/highlight.go:412-413`). Lines after it are not highlighted until the next `diff --git` (`internal/tui/highlight.go:408-410`), and a line after it is a code line by its first character, not a header line.
  - **The `git_diff` card** opens the plain content modal, not the diff modal. Its content is not highlighted (see the `read_file` rule below).
- (v0.1.4, amendment 102) **Header rule, `/diff` only.** A file's header runs from its `diff --git` line to its first `@@` line, or to a truncation marker if that comes first (`internal/tui/highlight.go:412-413`). Header lines are never highlighted and keep their current colours (today `---` lines draw in `error` and `+++` lines in `success`, by their leading character). This is the same rule the per-file stats (`+12 −4`) count by. After the first `@@`, a line beginning `+` or `-` is a code line, even if it begins `+++` or `---`. Approval diffs have no header lines. `@@` headers keep `hunk`. On `+` and `-` code lines the marker keeps its `success` or `error` colour, and the rest of the line is highlighted. Each line is lexed on its own. Context lines keep `muted` and are not highlighted, as before. The `+` and `-` characters stay the line markers, so colour is never the only signal.
- (v0.1.4, amendment 102) **Plain content** on a `read_file` card is highlighted by language, from the extension of the card's target path. Highlighting applies to no other card's content. An inline `<script>` on one line is highlighted as JavaScript, as the case pinned in `internal/tui/highlight_test.go` does. A script that spans lines is not, because each of its lines is lexed as HTML. A `read_file` line counts as numbered when it begins with one or more ASCII digits followed by at least N spaces, where N = 4 − (number of digits mod 4), that is 1 to 4. The prefix is the digits and exactly the first N spaces after them. Everything after the prefix is the line's text, whatever it starts with, including its own indentation. For example, `5` followed by seven spaces and `foo` has the prefix `5` plus three spaces, and the text is four spaces and `foo`. The tool writes `N<TAB>text`, and the sanitiser (§7.1) expands the tab to the next four-column stop, so the number and its spaces end on a stop. On a numbered line the prefix is shown `muted` and unhighlighted, and only the text after it is lexed. Any other line is shown in `text` and not lexed. That covers a truncation marker, a line cut mid-way by truncation, and the text of a failed `read_file` card. An unknown extension is plain, and so is an extension that chroma knows but Kirsch excludes, for example `.raku` or `.svelte` (see the excluded lexers below). The modal's own line numbers stay `dim`, as today.
- (v0.1.4, amendment 102) Modal highlighting depends only on colour being available. `ui.render_markdown` does not turn it off. With colour off, everything is plain, as in v0.1.3. (v0.1.4, amendment 102) The content view, the diff view and reply fences use one language mapping: a file's name or extension (including names such as `Makefile` and `Dockerfile`), or a fence's info string, maps to a lexer; anything unmapped is plain. The excluded lexers, which are the keys of `hlUnsafeLexers` in `internal/tui/highlight.go`, are known to chroma but always drawn without syntax colours: Raku, the PostgreSQL SQL dialect, the Materialize SQL dialect and Svelte. Every name, alias or file pattern that chroma maps to them is excluded, with one exception: the fence words that `hlInfoAlias` (`internal/tui/highlight.go`) maps to `sql`, at present `postgresql` and `postgres`. For example, a fence named `raku` is plain, a `.svelte` or `.raku` file is plain, and a `sql` fence is highlighted. Those fence words are looked up as `sql`, so their fences are highlighted as SQL (`TestHighlightPostgresInfoStringsMapToSQL`, `internal/tui/highlight_test.go`).
- (v0.1.4, amendment 102) **Limits.** Wherever highlighting applies, a line wider than 512 columns is drawn without syntax colours. Each line has 50 ms to be lexed; a line that runs over is drawn the same way. Each lexer that runs over is plain from then on, for the rest of the session, and nothing resets that. On a diff line drawn without syntax colours, the `+` or `-` marker keeps its colour. Over the width cutoff, and on a line with no lexer, that colour covers the whole line, which is drawn the plain way. A diff line has no lexer, and so draws whole in its colour, in cases such as an unmapped language, an excluded lexer, an unparsed `diff --git` section, lines before the first `diff --git`, or lines after a truncation marker in `/diff` before the next `diff --git`. Whenever lexing gives the line back as one text run, for example after a deadline miss or once a language has missed one, only the marker keeps its colour and the rest of the line is text. Known gaps: a slow moment, such as a GC pause or heavy load, during one line can turn highlighting off for that language for the rest of the session; and a line abandoned at the deadline keeps one background goroutine busy until chroma's own per-rule timeouts end it, at most once per language.
- (v0.1.5, amendment 103, UX-D7 and ux-d8) **Long lines wrap.** There is no sideways
  scroll. A body line wider than the modal's text width continues on the rows below
  it. The cut is hard, at a cell boundary: a wide rune is never split, and there is
  no word wrapping, because the body is code. The `⋯` cut that a modal body line had
  before is replaced by wrapping. The text width comes from `wrapWidths`: `inner-5` for a
  content line, after its number column, and `inner` for a diff line, with `inner-1` on a
  diff code line's later rows for the marker column. `inner` is the box width less 4
  (`modalFrameFor`), and a diff line's `inner` includes its marker. `wrapBounds` gives the
  point where each row breaks, and `modalRows` draws the rows. The sanitiser's 2000-column
  cap and its `⋯` (§7.1) run first, as before.
  - **Content modal.** The modal's number column (drawn in `modalRows`) shows the number on
    the first row of a source line only. A highlighted line takes its runs from `contentRuns`. A continuation row leaves those columns
    blank, so the text stays in its column. On a `read_file` line the number prefix
    is part of the line's text (the rule above), so it is drawn once, on the first
    row.
  - **Diff modal.** The `+`, `-` or space marker is on the first row only. A line is a
    code line when it is not a header line and begins with `+`, `-` or a space. Header
    status decides that, not the first character. In `/diff`, `hlSections`
    marks as header every line from `diff --git` up to the first `@@` or a truncation
    marker, and `diffIsCode` reads that flag as `h.lines[idx].header`, checked only when
    `md.WorkTree` is set. So a
    `---` or `+++` header line wraps flush though it begins `-` or `+`. A continuation
    row of a code line starts with one blank column in the marker's place, then the rest
    of the line, so the code stays aligned. A `@@` line, a header line and a descriptive
    line wrap flush, with no blank column. Approval diffs have no header lines, so there
    a line is a code line by its first character alone. Colour continues on
    continuation rows: a highlighted line keeps its syntax runs, and a line drawn the
    plain way keeps its whole-line colour (Limits, below). Colour is by prefix, as
    `modalRows` gives it: `@@` is `hunk`, a
    `+` line is `success`, a `-` line is `error`, and anything else is `muted`. Header
    lines follow the same rule by their leading character, as the header rule above
    says. On a highlighted `+` or `-` line the marker takes `success` or `error`
    (`diffRuns`).
  - **Scroll position.** `md.Off` counts display rows. `j`/`k`, the arrows, `PgUp`/`PgDn`
    and `g`/`G` move it by display row, through `keyModal`.
    - **One helper gives the range.** `modalExtent` returns the modal's total display rows
      and its body height for a layout. Both come from `modalSizing`, which runs the box's
      own height ladder (`modalFrameFor`). `modalSizing` is the single source of the extent.
      `modalBox` calls it directly, and `keyModal` reads it through `modalExtent`, so the keys
      and the render take their numbers from the same function.
    - **Keys clamp at both ends.** Each scrolling key first clamps `md.Off` to
      `[0, max(0, totalRows − bodyH)]`, then applies its step, then clamps again (the
      `scroll` closure in `keyModal`). The first clamp is needed because a resize leaves
      the stored offset alone (the `tea.WindowSizeMsg` case of `update`), so it can sit past
      the range; a step taken from there would move nothing on screen and the press would
      be lost. Where the height ladder draws no body, `bodyH` is 0, and the same formulas
      apply, giving `[0, totalRows]`. `modalBox` draws no body in that case, so the offset
      has no visible effect. At the rungs that show a notice instead of a box (`modalFrameFor`,
      `notice`), `modalExtent` returns the display-row count as `totalRows` and 0 as `bodyH`,
      and the same formulas apply. The same clamps apply to the help overlay, which shares
      `keyModal`. In `ModeModal`, a key reaches `keyModal` unless `handleKey` consumed it
      before the mode switch. `handleKey` is the reference for which input that is.
    - `g`/`Home` set it to 0. `G`/`End` set it to `max(0, totalRows − bodyH)`, the top of
      the last page.
    - Rendering clamps it to `[0, max(0, totalRows − bodyH)]`, so the last page is full.
      `modalBox` clamps a local copy, not the stored offset, and the rule is measured in
      display rows.
    - The box height is `totalRows` plus the chrome, in display rows (`modalSizing`).
      The clipping checks in `modalBox` and the marker (`buildClippingMarker`) count
      display rows too.
    - The page is `max(1, TranscriptH-4)` display rows (`keyModal`). It is not the body
      height.
    - A resize does not touch the modal (the `tea.WindowSizeMsg` case of `update`). The render
      draws a clamped copy of the offset, and the next key press re-clamps the stored
      offset. The top source line is not kept. That is a known
      gap, listed in `plan/spec/kirsch-plan.md` amendment 103.
    - A modal whose lines all fit the text width has one display row per source line,
      so it behaves as it did before.
  - **Help overlay.** Its lines are not wrapped. They keep the cut that `styleHelpLine`
    makes with `truncEnd`. No
    help line wraps, so for help a display row is a source line and `Off` means what it
    did before.
- **Unchanged.** The header, the per-file stats and the title. The highlighting
  cutoff (`hlMaxLineCols`, 512 columns) and the
  50 ms deadline still decide colour. A line over the cutoff now wraps instead of
  being cut, and it keeps its plain colouring.
- `j`/`k`/arrows scroll, `PgUp`/`PgDn` page, `g`/`G` top/bottom, `Esc` closes. The
  movement is by display row from v0.1.5 (amendment 103).

### 4.2 Help overlay (`?`)

Single screen: key bindings grouped by mode (§5.2), then slash commands (§6),
then a footer with version and the docs path. `Esc` or `?` closes. The minimum
terminal for the full-help overlay is 80×28. From v0.1.2 the left column is 17 rows,
the same as the right, so the overlay keeps this minimum (amendment 100).
From v0.1.3 (amendment 101) the browsing block's movement row reads
`select card/reply`, its `?  help` row is replaced by `c  copy reply`, and the
commands column's last row is `/exit  /copy`. The left column keeps its height,
so the minimum stays 80×28. `?` is still bound, and the footer's `Esc or ?
closes` names it.

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
                 ┌───────┴─▼────────┐   ⇧↑ / Tab     ┌───────────┐
                 │    Composing     │───────────────►│ Browsing  │
                 │ (composer focus) │◄───────────────│(selection)│
                 └──────────────────┘   ↓ at bottom  └───────────┘
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
| `↑` / `↓` | On the first / last visual row: recall previous / next entry from history (up to 100 entries per session; consecutive duplicates and blank entries skipped); past the newest, back to the unsent draft. Ignored while a turn is running |
| `Shift+↑` | Move to the cards |
| `Tab` | While typing a slash-command name (no whitespace yet), complete a unique prefix; an ambiguous or unknown prefix does nothing. In any other text, move to the cards |
| `Ctrl+G` | Re-pin the transcript to the bottom — works during a live turn (§2.4) |
| `PgUp` / `PgDn` | Scroll the transcript a page without leaving Composing — works during a live turn (§2.4) |
| `Esc` / `Ctrl+C` | Cancel the in-flight turn if busy; otherwise clear composer |
| `Ctrl+C` ×2 within 1s | Force quit |

There is no bare-key quit. `q` is an ordinary character in the composer, and
quitting is `/quit`, `/exit`, or `Ctrl+C` twice — see §6.

**Keys that act while a turn runs.** These run *ahead* of the busy guard in
`keyComposing` (`internal/tui/update.go`), so they stay live while a turn is in
flight: `Esc` and `Ctrl+C` (cancel the turn), `Ctrl+J` and `Alt+Enter` (newline),
`Shift+Enter` where the terminal sends it as `Alt+Enter` does (newline), `Tab`
(slash-command completion, or into Browsing), `Shift+↑` (into Browsing), `Ctrl+G`
(re-pin), and `PgUp`/`PgDn` (scroll). Ordinary typing and plain `Enter` are refused
while a turn runs.
The help overlay lists `Ctrl+G` and `PgUp`/`PgDn` in its **`composing`** block,
each in its own row. Its `Shift+Enter` row also names `Ctrl+J` (screen 06,
amendment 100).
The overlay is a one-screen summary, not this table.

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
| `↑` / `↓` | Move selection (cards and assistant replies) (v0.1.3, amendment 101) |
| `PgUp` / `PgDn` | Scroll without moving selection |
| `j` / `k` | Scroll one line without moving selection |
| `Home` / `End` | Top / bottom (`End` re-pins) |
| `g` / `G` | Top / bottom — the keyboard-reachable form of `Home` / `End` (`G` re-pins) |
| `Enter` | Tool card: toggle between the 10-line preview and collapsed. Assistant reply: nothing (v0.1.3, amendment 101) |
| `d` | Open content or diff modal for selected card. Assistant reply: nothing (v0.1.3, amendment 101) |
| `c` | Copy the selected reply's source text (§6; v0.1.3, amendment 101). On a selected card it copies nothing and shows the hint `c copies replies only`. With nothing selected it gives `nothing to copy` |
| `?` | Help overlay |
| `Tab` / `Esc` | Return to Composing (does **not** re-pin — §2.4) |
| `↓` at the last selectable item | Return to Composing (v0.1.3, amendment 101) |

**Selected reply (v0.1.3, amendment 101).** On a selected reply the key-hint row (§2.1) reads `↑↓ select · c copy · Tab/Esc prompt`, built from the same glyphs as today's row. On a card the row is unchanged.

**On `g` / `G`.** They mirror the modal's own `g`/`G` rather than inventing a second
vocabulary for the same gesture, and they exist because `Home` and `End` are not
universally reachable: on macOS Terminal both arrive as SS3, which Bubble Tea v1.3.10
does not decode. The help overlay's **`browsing`** block has no separate `g`/`G` row: its `g/G End`
row names them, and `j/k` shares the `PgUp/PgDn` row (screen 06, amendment 100).
The overlay is a summary, not these tables: some bound keys are not listed in it.
From v0.1.3 (amendment 101) the overlay's `?` row is replaced by `c`, `?` stays
bound, and the footer's `Esc or ? closes` names it.

**ApprovalPending**

| Key | Action |
|---|---|
| `y` / `n` / `a` / `d` / `?` | Act immediately |
| Navigation keys (`↑` `↓` `←` `→` `PgUp` `PgDn` `Home` `End`) | Never release |
| `Tab` / `Shift+↑` | Re-arm a released card |
| `Esc` / `Ctrl+C` | Cancel the turn (resolves as Cancelled, not Rejected) |
| Any other non-navigation key | Release the card and show `approval pending — Tab to return to the card`. While released, `y`/`a`/`n` do nothing until the card is re-armed. |
| Bracketed paste | Dropped, does not release |

`j`/`k` are Browsing bindings. In ApprovalPending they are ordinary keys and release the card. `c` is a Browsing binding too (v0.1.3, amendment 101): in ApprovalPending it is an ordinary key and releases the card, like `j`/`k`. `PgUp`/`PgDn` are navigation keys there: they do not release the card, and they do not scroll.

When armed, `y`, `a`, and `n` act immediately. `a` is inert unless the approval
offers a session grant. When released, `d`, `?`, `Tab`/`Shift+↑` (re-arm) and
`Esc`/`Ctrl+C` (cancel) act; `y`/`a`/`n` and every other non-navigation key are ignored.

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
trimmed, unparsed, except for `/copy` (below).

| Command | Behaviour |
|---|---|
| `/help` | Help overlay |
| `/status` | Notice: model, branch, dirty file count, session id, tokens (in/out/cache), compaction %, project-context file loaded, active grants |
| `/diff` | Diff modal on the working tree (`git diff`) (corrected, amendment 102) |
| `/files` | Notice listing files touched this session |
| `/approvals` | Notice listing active session grants; offers to clear (confirm prompt) |
| `/new` | New session; confirm if a turn is in flight |
| `/compact` | Manual compaction (plan §6.4); notice confirms, status bar shows `compacting` |
| `/quit` | Quit |
| `/exit` | Quit — an alias of `/quit`, sharing one arm rather than a second behaviour |
| `/copy` | Copies the last assistant reply's source text (v0.1.3, amendment 101) |
| `/copy code` | Copies the last fenced code block in the last assistant reply (v0.1.3, amendment 101) |
| `/copy code N` | Copies block N, counted from 1, in the last assistant reply (v0.1.3, amendment 101) |

**Source text and payload (v0.1.3, amendment 101).** A reply's **source text** is the
text as the model sent it, markdown source included, before the screen sanitiser
(`Sanitize`, §7.1) runs. It is not the wrapped screen lines. Live, the source is kept
when the stream ends, cancelled or not. On resume, it is the text stored in the
session's `assistant.message`. A reply that is still streaming copies the text
received so far.

The clipboard payload is the source text, cleaned for safe pasting. ANSI escape
sequences are removed; CRLF becomes LF; then these are removed: C0 controls other
than tab and newline; DEL; C1 controls (U+0080–U+009F); and the Unicode
bidirectional controls U+202A–U+202E and U+2066–U+2069. Unicode tag characters U+E0000–U+E007F are removed too. Other invisible characters, such as zero-width characters and the directional marks U+200E, U+200F and U+061C, are kept. Tabs are kept; no line-width cap applies. A payload that cleans to the empty
string gives `nothing to copy`.

**Code blocks (v0.1.3, amendment 101).** Fences are found in the cleaned payload
text, after the cleaning above, not in the source. An opening fence is a line whose
first characters that are not spaces or tabs are three or more backticks or three or more tildes. Any
indentation is allowed, because models indent fences inside list items. A backtick
fence whose info string contains a backtick is not a fence. In indentation, a tab
counts as advancing to the next multiple of 4 columns. A content line loses leading
spaces and tabs up to the opening fence's indentation, measured in columns. A tab that straddles that cut is replaced by spaces for the columns past the cut. A closing
fence uses the same character, is at least as long as the opening fence, may be
indented, and may be followed only by spaces or tabs. An unclosed fence runs to the
end of the reply, and the newline that ends the reply does not add an empty content
line to it. What is copied is the content lines only: no fence lines, no info string,
no trailing newline. A block is empty when its copied text is the empty string: it has no content lines, or its only content line is empty. Whitespace-only lines are content, so a block of spaces is not empty. An empty block copies nothing, and its notice is
`code block K is empty`.

**Arguments (v0.1.3, amendment 101).** `/copy` splits its arguments on whitespace. The accepted forms are:

- nothing: the last reply's source text;
- `code`: the last code block in the last reply;
- `code` followed by one token of ASCII digits only: block N, counted from 1. Leading
  zeros are allowed; no sign is allowed.

`code` is lower case only. Anything else gives `usage: /copy [code [N]]`. If the last
reply has no code block, the notice is `no code block in the last reply`, whatever N
is. If N is 0, greater than the block count, or too large to parse, the notice is the
range notice in the table below. `/copy` is refused while a turn runs, like every
slash command; that is existing behaviour.

**Precedence (v0.1.3, amendment 101).** When more than one condition holds, the
first that applies decides the notice:

1. an argument that is not an accepted form gives the usage hint;
2. no assistant reply, or `c` with nothing selected, gives `nothing to copy`, and `c` on a selected card gives `c copies replies only`;
3. for the `code` forms, a last reply with no code block gives
   `no code block in the last reply`;
4. N of 0, greater than the block count, or too large to parse, gives the range notice;
5. an empty block gives `code block K is empty`.

After that, a payload that cleans to the empty string gives `nothing to copy`.

Placeholders. Each placeholder stands for one kind of number, and the notice table
uses them as follows. `C` is the copied character count, in Unicode code points of the
payload: `copied C characters` and `sent C characters (OSC 52)`. `K` is a block number:
`code block K is empty`. `M` is the block count in the range notice, and it counts
every block, empty ones included. A bare `/copy code` refers to the last block, so its
`K` is that block's number. All numbers print as plain decimal with no leading zeros,
so `code 01` refers to block 1.

**Copy routes and notices (v0.1.3, amendment 101).** `c` and every `/copy` form share
one copy path. It runs in a background command, never on the update loop, and its
result comes back as a message that sets the notice. Each copy:

- The payload goes to the clipboard tool that `github.com/atotto/clipboard` finds
  on the host. That tool gets 3 seconds; if it has not finished by then, the route
  counts as failed.
- An OSC 52 sequence (`ESC ] 52 ; c ; <base64> BEL`) is written as one write to the
  terminal output the screen is drawn on. It is sent only when that output is a
  terminal and `TERM` is not `dumb`. Over SSH it can reach the clipboard of the
  machine the user sits at.

The notice shows in the composer's dim inline hint, the same place as the
unknown-command hint. `/copy` and `c` are hint-only paths (§2.4): they do not re-pin
or scroll the transcript, and the hint is the whole answer. A copy notice is a
composer hint like any other. It is cleared the same way: by any key that reaches the composer's text area, typing or cursor movement; when the composer is reset, on submit or `Esc`; and when an approval card arrives. A newer copy notice replaces it. A copy result that arrives while another kind of hint is showing (the unknown-command hint, the approval-release hint, or a usage hint) does not replace it. The
viewport keys that §2.4 says leave the hint alone (`Ctrl+G`, `PgUp`/`PgDn`) leave a
copy notice alone too. Notices are ASCII and fit the 40-column minimum (36 cells
after the indent); one that still does not fit is cut from the end, as other hints
are.

| Result | Notice |
|---|---|
| Local tool succeeded (whether or not OSC 52 was also sent) | `copied C characters` |
| Local tool failed or timed out, OSC 52 sent | `sent C characters (OSC 52)` |
| Neither route worked | `copy failed: no clipboard route` |
| No assistant reply, or `c` with nothing selected | `nothing to copy` |
| Last reply has no code block | `no code block in the last reply` |
| N is 0, above the block count, or too large to parse | `the last reply has 1 code block`, or `the last reply has M code blocks` otherwise |
| Block K is empty | `code block K is empty` |
| Argument not accepted | `usage: /copy [code [N]]` |
| `c` on a selected card | `c copies replies only` |

K, M and C are plain integers with no separators, as in §3.7. C counts the payload's
Unicode code points. The program cannot confirm that OSC 52 arrived, so the OSC 52
notice says the sequence was sent, not that the clipboard changed.

Copies in flight: the result of the most recent copy wins, and an older result that
arrives later is ignored.

Unknown `/command` → a dim inline hint under the composer, not an error card,
and **never sent to the model**. `Tab` completes a unique prefix. `/copy` is appended
last in the command list, so `/exit  /copy` is the last row of the help overlay's
commands column. `/c` and `/co` are now ambiguous between `/compact` and `/copy`, and complete
nothing.

## 7. Edge Cases

### 7.1 Text sanitisation (do this before rendering anything)

- **Strip every ANSI escape sequence** from tool output before it reaches the
  renderer — SGR colours, cursor movement, and OSC sequences alike. `git` and
  many test runners emit colour when they detect a TTY, and an unstripped
  sequence does not merely look wrong: it hands arbitrary terminal control to
  command output. Kirsch colours diffs itself, from parsed structure, never by
  passing escapes through.
- **Carriage returns**: in multi-line text, a `\r`-separated run (progress bars
  from `npm`, `go test`, `curl`) renders as its final segment only — that is what
  the writer intended the user to see. In a single-row field (next bullet), each
  `\r` becomes a space and a CRLF pair becomes one space, so the text before a
  `\r` is kept rather than hidden.
- **Single-row fields** are drawn on one row, so their line breaks are turned
  into spaces. A field is single-row by role: text the renderer draws on one row,
  stored through `SanitizeSingleLine` (`internal/tui/transcript.go`). For example,
  each line of the `/diff` working-tree modal is its own single-row value
  (`internal/tui/update.go`). Text kept as separate rows stays multi-line. Examples
  of multi-line text are the detail lines of an error card and the detail and diff
  lines of an approval card (`internal/tui/update.go`).
- **Tabs** expand to the next multiple of 4 columns, consistently everywhere, so code
  indentation lines up (corrected, amendment 102: the code has always expanded to the next 4-column stop).
- **Control characters** that are still present once escape sequences are stripped,
  tabs are expanded and lines are split are replaced with `·`. The set is the one
  `replaceControls` in `internal/tui/transcript.go` replaces. On the Sanitize path,
  tabs, line feeds and carriage returns are handled before that point and never
  become `·`. The markdown render runs `replaceControls` again on text that a
  character reference resolves to (`mdText`, `internal/tui/markdown.go`), so a
  reference that resolves to a control character is shown as `·`; the test
  `internal/tui/markdown_test.go` pins `&#1;` rendering as `·`.
- **Unicode bidi controls** U+061C, U+200E, U+200F, U+202A–U+202E and
  U+2066–U+2069 are replaced with `·`, in all sanitised text,
  single-row fields included. This differs from the clipboard payload, which
  removes U+202A–U+202E and U+2066–U+2069 and keeps the directional marks U+200E,
  U+200F and U+061C; see the clipboard payload paragraph under the `/copy` table
  in §6.
- **Very long lines**: any single rendered line is capped at 2,000 columns with
  a `⋯` marker. Notices and error card text are capped too. The cap applies to
  each line split at a line feed, before a single-row field joins its lines with
  spaces, so a single-row value whose source has several lines can be wider than
  the cap once joined. In a single-row field a carriage return is already a space
  before the cap, because `SanitizeSingleLine` in `internal/tui/transcript.go`
  turns it into one. A minified bundle on one line must not be able to hang the wrapper.

### 7.2 Input

- **Bracketed paste**: pasted text is literal, never interpreted as keys;
  newlines preserved. Paste over 8KB warns before inserting.
- **A paste is one update.** With bracketed paste on (the normal case), the terminal sends a paste as one message, so the composer takes it in one update and no debounce is needed (§11). With bracketed paste off, a paste reaches kirsch as key presses, and these limits apply. Each is current behaviour, not a fix: a multi-line paste sends its first line to the model as a message and drops the rest while a turn is busy; a tab arrives as the Tab key (`tea.KeyTab`), which in the typing area (`keyComposing`, `internal/tui/update.go`) completes a unique slash-command name, does nothing when the typing area holds a slash name that matches several commands or none, and otherwise moves focus to the transcript. When focus has moved, the rest of the paste is read as browsing keys (`keyBrowsing`), not typed into the composer, and a later Tab in the same paste moves focus back to the typing area (`keyBrowsing` handles `tea.KeyTab` and calls `setBase(BaseComposing)`), so the pasted text is split between the two; a multi-byte character (for example `é`) that falls on a terminal read boundary can be lost. That a tab arrives as `tea.KeyTab` is read from Bubble Tea's source (`keyHT`, `key.go`, bubbletea v1.3.10) and is not run; what kirsch does with it is read from `internal/tui/update.go` and is not tested. Status, test provenance and the open operator decision are in `plan/PROGRESS.md`, *Open from m5-d8*.

### 7.3 Rendering

- **Soft wrap** everywhere; no horizontal scrolling in v0.1.
- **Every row that is cut is cut with a marker.** A row too wide for the content
  width ends in `⋯` (`...` in ASCII, §10.2), never mid-word with nothing to say
  it was shortened. `View()`'s final bound on the frame is a hard edge rather
  than an elision and carries no marker, so anything whose width is not a
  constant — a build-time version string, a project name, a branch — is
  truncated by its own renderer before it reaches that bound. A row whose text
  fits is not a cut row: when the caret sits at the end of its line, with no
  character under it, the caret's own cell can make the row one cell wider, and
  the hard edge drops that cell with no marker. A caret over a zero-width
  character mid-text takes a cell of its own, so a row that holds one can be cut
  and marked even though its text fits. Read from the code, not measured: in
  `internal/tui/composer.go` the extra cell is counted only when the cluster
  under the caret is empty. The measured evidence is
  `TestComposerCaretOverWideAndZeroWidthAtTheBoundary` in
  `internal/tui/composer_cut_test.go`, which passes in both Unicode and ASCII
  modes. *Pending the operator's ratification
  (m5-d7b; `plan/PROGRESS.md`, Open from m5-d7b).*
- **No-color fallback**: `NO_COLOR` set to a non-empty value (an empty value is
  treated as unset), `TERM=dumb`, `TERM` empty, or not a TTY → styling degrades
  to plain prefixes (`+`/`-`, `[ok]`, `[err]`); layout identical. (corrected,
  amendment 102: an empty `TERM` was missing from this list, though
  `DetectCaps` has always treated it as no colour.) (v0.1.4, amendment 102)
  Colour does not gate Markdown rendering. A rendered reply keeps its structure
  with no styling, and its unstyled form is the same rendering code run with
  identity styles (§3.2, §13).
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
| Empty session | Wordmark, tagline, and three example prompts |
| Model unknown to the model table | Dim notice: cost display unavailable, conservative budget in use |

**The empty session, in detail** (screens 00 and 01). The block wordmark is 21
cells by 2 rows, suppressed below 40 columns or 10 rows where the transcript
needs the lines more; the band is read from the terminal width, not the content
width the margin leaves (§2.1), so it survives at the advertised 40-column
minimum. Beneath it sits the tagline
`v<version> · terminal-native coding agent`, truncated to the content width with
the §7.3 marker. Its width is **not** a constant — the version is a build-time
string, and the shipped `0.1.3-dev` already makes the tagline 41 cells against
the 38 a 40-column terminal leaves — so it is cut by its own renderer and reads
`v0.1.3-dev · terminal-native coding a⋯` rather than breaking off mid-air. The
version leads because it is the part worth keeping when the row is cut. The
lead-in and the three suggestions survive at sizes where the wordmark does not:
they are the part that tells a new user what to do.

**Notices under the onboarding.** Start-up notices, such as
`no previous session — starting fresh`, arrive before the first message. They do
not end the onboarding or the empty session. While the transcript holds nothing
but notices, the onboarding (or the empty session) stays, with one blank row and
then the notices beneath it, provided every notice row fits below it. This includes
notices from user commands, such as `/status`. If any notice row would not fit, the
normal scrollable transcript shows instead, with nothing cut off. The first message,
a tool card, an error card or any other item ends the onboarding, as before.

The rule is decided afresh for each frame, so the onboarding can come back. `/new`
empties the transcript and brings it back, and so does a larger window while the
transcript still holds only notices that now fit.

## 8. Milestone 0 Prototype Scope

The prototype implements everything above with **fake data**:

- Fake transcript: 2 user messages, timer-driven streaming assistant text,
  3 tool cards (one truncated, one errored), **two** approval cards (one
  `apply_patch` without `[a]`, one `run_command` with `[a]`), 1 error card,
  1 system notice.
- Working: composer typing, the full mode state machine (§5.1), card selection
  and expansion including the 10-line preview mode, approval `y`/`a`/`n` flow,
  content/diff modal, help overlay, confirm prompt, resize, quit paths,
  scroll/pin behaviour, all §10 styling with both fallbacks.
- **No** LLM, file, shell, or session code. No real policy. Spinner states and
  streaming are simulated.

**Acceptance:** `go run ./cmd/kirsch` opens; typing, scrolling, selection,
modals, resize, and quit all work with no panic or visual corruption.

## 9. Resolved Decisions

- **Newline binding:** no binding is primary (§5.2). The composer accepts the
  `ESC`+`CR` that `Shift+Enter` or `Alt+Enter` sends, depending on the terminal,
  and `Ctrl+J`, the one always representable. `Shift+Enter` was the original
  preference but is not representable on Bubble Tea v1, which
  carries no shift modifier — and most terminals send it as a bare CR anyway.
  M0 tests both bindings it can actually construct. See §5.2 and plan
  amendment 29.
- **Palette:** dark only for v0.1. No light theme. See §10.
- **Background:** never painted; the terminal's own background shows through.
- **Token display:** 1 decimal, k/M units (`12.4k tok`).
- **Syntax highlighting:** out of scope for v0.1. (v0.1.4, amendment 102) From v0.1.4 it is in scope, for reply fences and the content and diff modals (§3.2, §4.1).
- **Mouse support:** out of scope for v0.1.
- **Transcript search:** deferred to v0.2.
- **Inline preview:** 10 lines by default, full content via the modal (`d`).
- **`?` in the composer is a literal character**, not a help key.
- **Typing never re-pins** a scrolled-up transcript.
- **No bare-key quit.** `q` is an ordinary character in the composer. Quitting is
  `/quit`, `/exit`, or `Ctrl+C` twice. See §5.2 and §6.
- **`Esc` does not re-pin.** Leaving Browsing returns focus to the composer and
  keeps the viewport where it is. See §2.4.
- **Slash commands re-pin** on submission; the hint-only paths do not, because
  they put nothing in the transcript. See §2.4.
- **`g`/`G` are bound in the transcript pane as well as in modals** — one vocabulary
  for the gesture, and the reachable form of `Home`/`End` where those arrive as
  undecoded SS3. See §5.2.
- **`j`/`k` scroll one line in Browsing** (UX-D2, v0.1.2). They move the transcript,
  not the selection. See §2.4 and plan amendment 100.
- **Composing `PgUp`/`PgDn` scroll a page** (UX-D3, v0.1.2). They run ahead of the
  busy guard and do not change the mode. See §2.4 and plan amendment 100.
- **Cut-off notice on resume** (v0.1.2). A reply that stops on `max_tokens` gets a
  system notice with the `max_tokens` sent, shown live and on resume from the
  `stop_reason` and `max_tokens` fields of `assistant.message`. See §3.7 and plan
  amendment 100.

**Open questions (not blocking Milestone 0):**

- Whether `/approvals` deserves a direct key binding.

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
| `codeBg` | 235 | `#262626` | — | Tool-card output panels. (v0.1.4, amendment 102) Rendered replies paint no background (§3.2). |
| `selectionBg` | 236 | `#303030` | — | Selected card background |
| `synKeyword` (v0.1.4, amendment 102) | 141 | `#af87ff` | 6.1:1 | Syntax highlighting: keywords and keyword constants (§3.2, §4.1) |
| `synString` (v0.1.4, amendment 102) | 180 | `#d7af87` | 8.2:1 | Syntax highlighting: strings; inline code in replies (§3.2, §4.1) |
| `synNumber` (v0.1.4, amendment 102) | 216 | `#ffaf87` | 9.3:1 | Syntax highlighting: numbers and other literals (§3.2, §4.1) |
| `synType` (v0.1.4, amendment 102) | 152 | `#afd7d7` | 10.7:1 | Syntax highlighting: type, class and keyword-type names (§3.2, §4.1) |

(v0.1.4, amendment 102) Markdown rendering and syntax highlighting use these roles as set out in §3.2 and §4.1.

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
| Composer caret | `▌` | `_` |
| Composer caret, over its own glyph | `▐` | `#` |
| Selection gutter | `┃` | `\|` |
| Box drawing | `┌─┐│└┘` | `+-+\|+ +` |
| Bullet (notices) | `·` | `.` |
| Dash in notice text | `—` | `-` |
| Markdown bullet (v0.1.4, amendment 102) | `•` | `-` |
| Markdown quote prefix (v0.1.4, amendment 102) | `│` | `\|` |
| Markdown table column separator (v0.1.4, amendment 102) | `│` | `\|` |
| Markdown table header rule (v0.1.4, amendment 102) | `─` | `-` |
| Markdown horizontal rule (v0.1.4, amendment 102) | `─` | `-` |
| Scrollbar track (v0.1.5, amendment 103) | `│` | `\|` |
| Scrollbar thumb (v0.1.5, amendment 103) | `┃` | `#` |

Fallback triggers when the locale is not UTF-8. **Every state is identified by
its glyph, not only by its colour** (§12).

**The composer caret has two glyphs because it is drawn over the text rather than
beside it.** It takes the cell of the character the cursor is on, so wherever that
character is already the caret's own glyph the row renders byte-identical to the
plain text and there is no cursor on screen. The second glyph is drawn in exactly
that case and never otherwise. No single glyph avoids it: the composer accepts
arbitrary text, so any character chosen is one the user can type — and in the
ASCII table `_` is not a corner case but `parse_slash` and `my_file.go`.

**The stream caret shares the first glyph and none of this behaviour.** It is
drawn beside the streaming text rather than over it, and is removed when the
message completes (§3.2), so it never covers a character, never collides with
one, and never uses the second glyph. Only the composer caret has a stand-in.

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
| Paste (bracketed) | none | One update per paste; no delay is added (§7.2) |
| Double `Ctrl+C` window | 1000ms | Long enough to be deliberate |
| Cancel → composer usable | < 1000ms (target) | Tested, not eyeballed (plan §8 M3) |
| Approval card appearance | immediate | Never debounced |

## 12. Appendix — Accessibility

1. **Never signal state by colour alone.** Every state carries a glyph; colour
   reinforces. This is what makes the no-color fallback a downgrade rather than
   a loss of information.
2. A non-empty `NO_COLOR`, `TERM=dumb`, an empty `TERM`, and non-TTY output are
   all honoured. An empty `NO_COLOR` is treated as unset. (corrected, amendment
   102: an empty `TERM` was missing here, though `DetectCaps` has always treated
   it as no colour.)
3. No blinking, no flashing, no animation other than the spinner.
4. No binding requires more than two keys; no mouse, ever.
5. Dim and faint text never carry unique information.
6. Layout is identical with and without colour — only styling degrades, so a
   screen reader or a piped capture sees the same structure. (v0.1.4, amendment
   102) This holds for rendered replies too, because their unstyled form is the same rendering code run with identity styles (§13).

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
| 7 | Tool card in preview (10-line default) | [07](kirsch-ui-screens.md#07--tool-card-in-preview-10-line-default) |
| 8 | Error card and system notice | [08](kirsch-ui-screens.md#08--error-card--system-notices) |
| 9 | Narrow width (40 cols) and very narrow (38 cols, transcript hidden) | [10](kirsch-ui-screens.md#10--narrow-and-short-terminals) — first two grids |
| 10 | Short height (6 rows) | [10](kirsch-ui-screens.md#10--narrow-and-short-terminals) — third grid |
| 11 | `NO_COLOR` mode | [11](kirsch-ui-screens.md#11--no_color--ascii-fallback) |
| 12 | ASCII-fallback mode | [11](kirsch-ui-screens.md#11--no_color--ascii-fallback) |
| 13 | Scrolled up with `↓ n new` indicator | [09](kirsch-ui-screens.md#09--scrolled-up-unpinned) |
| 14 | Onboarding: no API key; not a Git repo | [12](kirsch-ui-screens.md#12--onboarding-no-api-key-not-a-git-repo) |
| 15 | (v0.1.4, amendment 102) A finished reply rendered as markdown | [13](kirsch-ui-screens.md#13--rendered-reply) |
| 16 | (v0.1.5, amendment 103) Scrollbar drawn, transcript overflowing and unpinned | [14](kirsch-ui-screens.md#14--scrollbar-transcript-overflowing-unpinned) |
| 17 | (v0.1.5, amendment 103) Content modal with a wrapped `read_file` line | [15](kirsch-ui-screens.md#15--long-lines-wrap-in-the-modal) |
| 18 | (v0.1.5, amendment 103) Approval-diff modal with a wrapped hunk header and a wrapped code line | [15](kirsch-ui-screens.md#15--long-lines-wrap-in-the-modal), grid 15b |

Every state in the table has a grid; screen 12 was drawn in Milestone 3 before its
golden file was captured (plan §11, amendment 24). Screen 13 is generated from
the render, not drawn first (plan §11, amendment 102). The rows marked amendment 103
have grids and golden scenarios, and follow the properties below like every other state.
Wrapping also changes screen 05, not only the rows above: its diff hunk header is wider
than the modal's text width (`modalFrameFor`, and the `fakeDiff` hunk line in
`internal/tui/fake.go`), so the modal-wrap code redraws that grid as well.

Structure is the contract and colour is applied on top: line counts and box
positions are identical with and without colour. Screen 11 is a
character-for-character transliteration of screen 03 through the §10.2 fallback
table — every substitution is width-preserving — so the pair is what makes the
property testable rather than merely asserted.

**Four properties hold for every state, and are worth more than the snapshots.**
They are what the snapshots exist to protect:

1. `strip(styled) == plain`, byte for byte. This is the real form of "identical
   with and without colour", and it holds for every state in the table, not only the
   two drawn as a pair. (v0.1.4, amendment 102) For a rendered reply it holds by
   construction: the plain form is the same rendering code run with identity styles (the double pass in `relayout`). The checks that can fail for a rendered reply include: property 2 (no `0x1b` bytes
   in the plain path); property 3 (the SGR allow-list, below); the width post-condition
   (§3.2: no rendered row exceeds the item's width); and a test that renders the same reply with the live palette and with identity styles and compares them after stripping; it fails if rendering branches on the styles.
2. Zero `0x1b` bytes are emitted under `NO_COLOR`. Not "no visible colour" —
   *no escape bytes at all*.
3. Every escape in styled output is an SGR sequence drawn from the §10.1
   palette. Anything else — cursor movement, an OSC sequence, a colour index
   that is not in the table — means either tool output reached the terminal
   unfiltered (§7.1) or a raw colour number escaped `styles.go`. (v0.1.4,
   amendment 102) In a rendered reply the permitted SGRs are bold, italic,
   strikethrough, and foreground colours from §10.1 (`38;5;N`, with N in that table). The allow-list also holds the reset (`0`) and the codes that close those attributes: `22` (bold off), `23` (italic off), `29` (strikethrough off) and `39` (foreground default). No other sequence is emitted:
   Kirsch draws every rendered row through `Styles`, and a property-3 test
   asserts the allow-list, including that no OSC 8 hyperlink and no colour
   outside the palette appears. In a rendered reply, no background SGR is emitted.
4. Row count equals the terminal height and no row exceeds the terminal width,
   at every size in §2.2.

Property 3 supersedes the looser instruction to assert that "no escape sequence
survives into `View()`". That is false as stated: Kirsch emits its own SGR
sequences whenever colour is on. The checkable claims are 2 and 3 — no escapes
at all in the no-colour path, and only palette SGR in the coloured one.

Golden grids are regenerated with `KIRSCH_REGEN=1 go test ./internal/tui -run
TestGenerateGrids`, which writes the current render to `/tmp/kirsch-grids.txt`;
a person reconciles the grids in `kirsch-ui-screens.md` by hand, and a human
reviews the diff. CI never auto-accepts them. When a golden diff is reviewed and
found *correct*, the screen reference has gone stale — update it in the same
commit.

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
- **10-line preview mode** — tool cards open as a preview by default, collapsible
  to the head, and expandable to a modal. Approved approval cards fold away,
  linking to their tool card. These changes keep the transcript legible while
  tools are running.
- **Exclusive approval capture** is correct but can feel abrupt when an
  approval interrupts scrollback reading. Watch for it during M2.
