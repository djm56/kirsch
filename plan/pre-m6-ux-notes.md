# Pre-Milestone 6 UX Notes

> **Status 2026-11-19:** turned into a plan. See [milestones/ux-round-v0.1.md](milestones/ux-round-v0.1.md). Items 3 and 7 (mouse) are deferred past the round. Item 4's colour check is closed: the operator confirmed diffs show green and red.

> **Operator notes first captured 2026-11-19, researched against the code the same day.** Items 1–7 are the operator's observations from using kirsch after v0.1.1. Items 8–13 were found in the research round. Each item says what was checked in the code. Nothing here is built. This work sits before Milestone 6 and is separate from it.

## Summary

| # | Item | Size | Needs spec amendment | Status |
|---|------|------|----------------------|--------|
| 1 | Markdown rendering for LLM output | Large | Yes | Planned — v0.1.4 |
| 2 | Scrollbar in results pane | Medium | Yes | Planned — v0.1.5 |
| 3 | Mouse wheel scroll in results pane | Medium | Yes | Deferred |
| 4 | Code highlighting in the detail modal | Medium | Yes | Planned — v0.1.4 |
| 5 | Copy LLM output to clipboard | Medium | Yes (help screen) | Planned — v0.1.3 |
| 6 | Output cut off by the model table cap | Small | No *(superseded 2026-11-19: amendment 100 changes plan §5 and ui-spec §3.7)* | Planned — v0.1.2 |
| 7 | Scrollbar click and drag | Medium | Yes | Deferred |
| 8 | No notice when a reply is cut off | Small | No *(superseded 2026-11-19: amendment 100 adds the §3.7 notice to the spec)* | Planned — v0.1.2 |
| 9 | No line-by-line scroll keys in the transcript | Small | Yes (help screen) | Planned — v0.1.2 |
| 10 | Page scroll does nothing while typing | Small | Yes (help screen) | Planned — v0.1.2 |
| 11 | Long lines in the modal are cut, not wrapped | Small | Yes | Planned — v0.1.5 |
| 12 | Text replies cannot be selected | Small | Yes | Planned — v0.1.3 |
| 13 | Scroll keys missing from the help screen | Small | Yes (help screen) | Planned — v0.1.2 |

Notes added 2026-11-20 beside the table; the status cells above are unchanged:

- Row 6: *(now built for v0.1.2, ux-d1, pending the operator's tag)*
- Row 8: *(now built for v0.1.2, ux-d1, pending the operator's tag)*
- Rows 9, 10 and 13: *(now built for v0.1.2, ux-d2, pending the operator's tag)*

All thirteen are treated as high priority by the operator. "Size" is a rough guess, not an estimate.

## Before any of this is built

### The UI spec currently forbids part of it

These lines in `plan/spec/ui-spec-v0.1.md` have to be amended before the matching items start:

- **Mouse.** Line 798 (§12 rule 4) *(now line 833)*: "No binding requires more than two keys; no mouse, ever." Line 25 (§1) *(now line 25)* makes keyboard-only a design goal, and line 676 (§9) *(now line 701)* lists mouse support as out of scope for v0.1. Items 2, 3 and 7 need this changed. *(superseded 2026-11-19: item 2 ships without mouse in v0.1.5; only items 3 and 7 need this)* The amendment should keep "nothing *requires* the mouse": every mouse action keeps a keyboard equivalent.
- **Syntax highlighting.** Lines 241 and 675 (§3.2 and §9) *(now lines 248 and 700)* put it out of scope for v0.1 as "a large dependency and an open-ended rabbit hole". Items 1 and 4 reverse that.
- **Pinned screens.** The screen layouts and the help overlay are fixed by reference grids in `plan/spec/kirsch-ui-screens.md`, and golden tests compare against them. A scrollbar column, new help rows, or styled markdown all change those grids, so the grids and goldens are regenerated as part of the work. The code comment at `internal/tui/update.go:28` says the same about adding help rows.
- **Background rule.** Kirsch never paints the terminal background (ui-spec design goal 6). Only the code and selection panels set one. Any markdown or highlighting style has to obey that.

### Mouse scroll and copying pull against each other

When a terminal program turns on mouse reporting, the terminal's own click-and-drag text selection stops working. The user then has to hold a modifier key (Option in Terminal.app and iTerm2, Shift in most others) to select text. So item 3 on its own makes copying harder than it is today.

Decision for planning: items 3 and 5 ship together, and a config switch turns the mouse off for anyone who prefers the terminal's own selection. Suggested key: `ui.mouse` (boolean, default `true`). *(superseded 2026-11-19: mouse is deferred past the UX round; item 5 ships without it)*

## Items

### 1. Markdown rendering for LLM output

LLM replies are shown as raw markdown: `**bold**`, `# heading` and backtick fences appear as literal characters. Add a setting that renders them as formatted terminal text.

**Checked in the code:**
- Assistant text is stored as plain lines and soft-wrapped at draw time (`internal/tui/transcript.go:79`, `internal/tui/view.go:199`).
- While a reply streams, the lines are recomputed from the whole raw text on every chunk (`internal/tui/transcript.go:88`).
- `go.mod` has lipgloss and termenv but neither `glamour` nor `chroma`.

**Notes for the build:**
- `glamour` is the usual Bubble Tea companion for this. It depends on `chroma`, so items 1 and 4 share one new dependency. (From general knowledge; confirm the versions when the work starts.)
- Glamour's stock styles add margins and background colours. Kirsch needs its own style that paints no background and uses the existing palette.
- Streaming needs a decision. Rendering markdown on every chunk will flicker on a half-finished fence or list. Options: render plain while streaming and format when the reply completes, or format each finished block as it closes.
- `NO_COLOR` and `TERM=dumb` must still work: formatting degrades to plain text, with the same layout.
- Config key suggestion: `ui.render_markdown` (boolean, default `true`). `false` gives today's behaviour.
- Affects assistant text only. Cards, the status bar and user messages stay as they are.

---

### 2. Scrollbar in results pane

The results pane has no position indicator, so in a long session it is hard to tell where you are.

**Checked in the code:**
- Kirsch does not use Bubble Tea's `viewport` component. It keeps its own scroll state (`m.scroll`, with `setOffset` and `repin`) and draws the pane itself in `internal/tui/view.go`. The scrollbar is drawn there.

**Notes for the build:**
- One column on the right edge: a thumb showing position and how much of the transcript is visible, on a faint track.
- Needs a fallback glyph pair for terminals without Unicode, like every other glyph in the spec.
- Costs one column of transcript width, which changes every pinned screen grid.

---

### 3. Mouse wheel scroll in results pane

The scroll wheel does nothing useful in the results pane.

**Checked in the code:**
- The program starts with `tea.WithAltScreen()` only (`cmd/kirsch/main.go:245`). No mouse option is set, and no mouse message is handled anywhere.

**Notes for the build:**
- Add `tea.WithMouseCellMotion()` when `ui.mouse` is on, and handle wheel events in the update loop.
- The wheel scrolls two or three lines per notch, never a full page.
- Wheel scrolling moves the view only. It must not move the card selection, and scrolling up must unpin from the bottom exactly as PgUp does (ui-spec §2.4).
- The wheel works in every mode, including while typing and while a turn is running, and scrolls the modal when one is open.
- Ships together with item 5; see "Mouse scroll and copying pull against each other" above.
  - *(Superseded 2026-11-19: item 3 is deferred past the UX round, and item 5 is ux-d3 in v0.1.3, with no mouse. See `plan/milestones/ux-round-v0.1.md`, Deferred.)*

---

### 4. Code highlighting in the detail modal

The operator asked for the code shown by `d` to be coloured.

**Checked in the code:**
- The diff view is already coloured by line: added lines green, removed lines red, `@@` headers in their own colour, context muted (`internal/tui/modal.go:383`).
- The content view (`d` on a file read or command output) is plain text with dim line numbers.
- There is no language-aware highlighting anywhere.

**So this item is two things:**
- **First, check why the diff looked uncoloured.** If the operator's diffs show no red and green, that is a bug or a terminal setting (`NO_COLOR`, or a `TERM` value kirsch treats as no-colour), not a missing feature. Check with `kirsch doctor` and a known patch before any build work.
  - **Result 2026-11-19:** the operator confirmed diff lines show green and red. Not a bug; closed.
- **Then, language highlighting.** Colour code by language in the content view, inside diff lines (keeping the red and green as the line marker), and in fenced code blocks in replies (with item 1).

**Notes for the build:**
- `chroma` does this in Go and comes in with glamour anyway.
- Pick the language from the file name the card already carries.
- Colour must never be the only signal: the `+` and `-` characters stay.

---

### 5. Copy LLM output to clipboard

There is no way to copy text out of kirsch other than the terminal's own selection, which picks up borders and wrapped lines and breaks across the screen edge.

**Checked in the code:**
- No clipboard code exists in `internal/`.
- `github.com/atotto/clipboard` is already an indirect dependency (it arrives with the text input component). It calls `pbcopy` on macOS and `xclip`, `xsel` or `wl-copy` on Linux.

**Notes for the build:**
- **Two routes, used together.** Write an OSC 52 escape so copying works over SSH, and also call the local clipboard tool so it works in terminals that ignore OSC 52. Apple's Terminal.app is believed not to support OSC 52 (from general knowledge; re-check when building), so OSC 52 alone would fail on a stock Mac.
- **What gets copied is the raw text**, the markdown as the model wrote it, not the rendered or wrapped screen rows.
- **Minimum useful version:** a `/copy` command that copies the last assistant reply, plus a key in browsing mode that copies the selected item. That key depends on item 12.
  - *(Now planned as ux-d3 and ux-d4 in v0.1.3, `plan/milestones/ux-round-v0.1.md`. The copy-key choice is UX-D5, still open.)*
- **For code:** a way to copy just one fenced code block from a reply is the most common real need and worth planning at the same time.
- Show a one-line notice after a copy ("copied 1,204 characters"), and a clear notice when no clipboard route worked.
- Free-form mouse selection inside kirsch (click, drag, highlight) is a much larger piece of work. It is not planned here; the per-reply and per-block copies cover the need, and `ui.mouse = false` gives back the terminal's own selection.

---

### 6. Output cut off by the model table cap

Replies stop part-way on some models. Kirsch causes this itself.

**Checked in the code:**
- Kirsch sends `max_tokens` on every request, taken from its model table (`internal/app/app.go:1836`, `internal/provider/anthropic/encode.go:96`).
- The table is in `internal/provider/models.go`. `minimax-m3` and `minimax-m2.7` are set to 4096 output tokens, marked "fallback".
- A model that is not in the table gets the most cautious row's figures, which is also 4096 (`conservativeLimits`, `internal/provider/models.go:144`).
- The table holds six models: `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1`, `claude-haiku-4-5-20251001`, `minimax-m3`, `minimax-m2.7`. Everything else on opencode is unknown to it, including `claude-haiku-5-5`, the Qwen models and the GLM models. All of those are capped at 4096.

**What to do before M6:**
- Replace the two minimax fallback figures with the published ones, and add rows for the models the operator actually uses. *(2026-11-19: only if a cited source is found, UX-D1; see ux-round-v0.1.md ux-d1)*
- Raise the unknown-model output default to something workable, and keep the cautious context window. The right figure needs deciding; a cap that is too high is refused by some endpoints. *(superseded 2026-11-19: decided, 16384, UX-D1)*

**What not to do:** add a `provider.<name>.max_output` config key. Milestone 6 deliverable m6-d1 replaces this table with a `models.toml` the user can override per model, which is the lasting fix. A config key added now would be removed a milestone later. If m6-d1 is still far off when this is picked up, revisit that call.

---

### 7. Scrollbar click and drag

With the scrollbar (item 2) and the mouse (item 3) in place:

- Clicking the track jumps to that position.
- Dragging the thumb scrolls continuously.

This is the lowest-value part of the mouse work and can follow items 2 and 3 instead of shipping with them. The keyboard and the wheel already cover the need.

---

### 8. No notice when a reply is cut off

When a reply hits the output cap it simply ends, and nothing tells the user it was cut short.
  - *(Superseded for v0.1.2: the cut-off notice ships as ux-d1.)*

**Checked in the code:**
- The stop reason `max_tokens` is decoded (`internal/provider/anthropic/decode.go:613`) and mapped to `provider.StopMaxTokens`, but nothing outside the provider package reads that value.

**What to do:** when a reply ends for this reason, show a notice line under it saying it was cut off at the output limit, and name the limit. Milestone 6 deliverable m6-d6 deals with tool calls in a cut-off reply; it does not add this notice.

Build with item 6. Together they are the smallest change with the biggest daily effect.

---

### 9. No line-by-line scroll keys in the transcript

**Checked in the code:**
- In browsing mode, `↑` and `↓` move between cards, not lines. `PgUp` and `PgDn` move a page. `Home`, `End`, `g` and `G` jump to the ends (`internal/tui/update.go:915` onward).
- `j` and `k` scroll by line only inside the modal.
  - *(Superseded for v0.1.2: `j` and `k` also scroll one line in Browsing, ux-d2.)*
- So there is no way to move the transcript by one line.

**What to do:** add line scroll keys in browsing mode. `j` and `k` would match the modal. *(superseded 2026-11-19: decided, `j`/`k`, UX-D2)* `Ctrl+↑` and `Ctrl+↓` are an alternative, but many terminals do not send them distinctly, so they need testing before being chosen.

---

### 10. Page scroll does nothing while typing

**Checked in the code:**
- `PgUp` and `PgDn` are handled in browsing, approval and modal modes. In composing mode they are passed to the text box, where they do nothing useful (`internal/tui/update.go:788`).
  - *(Superseded for v0.1.2: `PgUp` and `PgDn` scroll the transcript from composing too, ux-d2.)*
  - *(Correction, 2026-11-20: in the approval card `PgUp` and `PgDn` are handled only as navigation keys. They neither release the card nor scroll the transcript. `isNavigationKey` lists them (`internal/tui/update.go:1122`), and `keyApproval` (`internal/tui/update.go:1061`) does not scroll on them. The composing half is what the v0.1.2 note above supersedes.)*
- To scroll, the user first presses `Shift+↑` or `Tab` to enter browsing mode. This is why the operator has been "using shift and up arrow".

**What to do:** make `PgUp` and `PgDn` scroll the transcript from the composer, without changing mode or moving the card selection. `Ctrl+G` already returns to the bottom from the composer.

---

### 11. Long lines in the modal are cut, not wrapped

**Checked in the code:**
- The modal trims every line to the modal width and ends it with `…` (`truncEnd` in `internal/tui/modal.go:389` and `:394`). There is no wrapping and no sideways scroll.

A wide diff line or a long line of command output cannot be read in full anywhere in kirsch.

**What to do:** either wrap long lines in the modal, or add sideways scroll with `h` and `l`. Wrapping is simpler; sideways scroll keeps diffs aligned. One of the two needs choosing.

---

### 12. Text replies cannot be selected

**Checked in the code:**
- Only cards can be selected. `↑` and `↓` skip user and assistant text on purpose, because nothing could be done with a selected text block (`internal/tui/transcript.go:52`).

Once copying exists (item 5) there is something to do with one. Either let `↑` and `↓` stop on assistant replies, or add a separate pair of keys that jump between replies. This also gives a quick way to move through a long session reply by reply.

---

### 13. Scroll keys missing from the help screen

**Checked in the code:**
- `Home`, `g` and `G` work in the transcript but are left out of the help overlay on purpose, because the overlay's layout is pinned by the spec (`internal/tui/update.go:28`). `End` is listed.
- `Ctrl+G` (back to the bottom from the composer) is not listed either. *(superseded 2026-11-19: amendment 100 lists it in the composing block)*

**What to do:** amend the help screen once, in the same change that adds the keys from items 5, 9, 10 and 12, so the grid is regenerated one time. *(superseded 2026-11-19: screen 06 is redrawn in v0.1.2 and again in v0.1.3)*

---

## Overlap with Milestone 6

`plan/build-plan-v0.2-v0.6.md` already covers some of this ground:

| M6 deliverable | What it does | Effect on these notes |
|---|---|---|
| m6-d1 Model catalog in data | Moves the model table to `models.toml`, with a user file that overrides it | The lasting fix for item 6. Reason not to add a `max_output` config key now. |
| m6-d3 `kirsch models` | Lists each model's window and max output | Makes a wrong cap visible. |
| m6-d6 Truncation guard | Fails tool calls in a reply cut off by `max_tokens` | Related to item 8 but adds no user notice. |
| m6-d9 `/model` picker and `--model` | Switches model inside the TUI | Removes the need to edit `config.toml` to change model. |

Nothing in M6 to M10 covers mouse support, markdown rendering, highlighting, clipboard copy or the scroll keys. Transcript search is listed as permanently out of scope in the build plan.

## Suggested order

1. **Items 6 and 8** — fix the output cap and say when a reply is cut off. Small, no spec change. *(superseded 2026-11-19: items 6 and 8 now change the spec, amendment 100)*
2. **Items 9, 10 and 13** — keyboard scrolling and the help screen. Small, one help-screen amendment.
  - *(Now built for v0.1.2 as ux-d2.)*
3. **Items 5 and 12** — copying, with `/copy` for the last reply first.
4. **Items 2 and 3** — mouse wheel and scrollbar, with the `ui.mouse` switch, after the spec amendment. Item 7 follows if still wanted.
  - *(Superseded 2026-11-19: item 3 is deferred past the UX round. Item 2 is ux-d7 in v0.1.5, with no mouse, per `plan/milestones/ux-round-v0.1.md`.)*
5. **Items 1 and 4** — markdown rendering and code highlighting together, since they share the new dependency. Check the diff-colour question in item 4 first; it costs nothing.
6. **Item 11** — long lines in the modal, once the wrap-or-scroll choice is made.

## Related

- `plan/spec/ui-spec-v0.1.md` — the UI spec these items amend.
- `plan/spec/kirsch-ui-screens.md` — the pinned screen grids.
- `plan/build-plan-v0.2-v0.6.md` — Milestone 6 and beyond.
- `plan/milestones/` — per-milestone deliverable tracking.
