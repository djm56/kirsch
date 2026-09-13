# Manual test — Milestone 0 (the interface)

Checking the things automation cannot: whether it feels right, whether it is
legible on *your* terminal, whether the keys you press do what you expect.

**Time:** about 15 minutes.
**Prerequisite:** `npm test` green. This walkthrough assumes the logic works and
only asks whether the interface does.

```bash
npm start
```

Mark each check ✅ or ❌. A ❌ with a sentence about what you saw is worth more
than a long report.

---

## 1 · First run

The launch screen, before anything is typed.

- [ ] The wordmark renders as two rows of solid blocks — no gaps, no tofu boxes
- [ ] The tagline reads `v0.1.0-dev · terminal-native coding agent`
- [ ] Three suggestions appear under "Ask anything. Three things to try:"
- [ ] The header shows `Kirsch ─ <your repo> ─ <your branch>`
- [ ] A `●` follows the branch if the repo has uncommitted changes
- [ ] The composer shows the placeholder `Ask anything (Enter to send, /help for help)`
- [ ] Nothing is cut off, and no line wraps oddly

**Legibility — the one worth taking seriously.** The palette was retuned in
Milestone 0 after the first version measured below the readable contrast
threshold (plan §11 amendment 41). Look at it on your actual background:

- [ ] The three suggestions are readable, not washed-out grey
- [ ] The separator rules are visible but not shouting
- [ ] The header title reads as distinct from the body text

If any of that looks wrong on your theme, say which and the palette moves. It is
one line per token in `internal/tui/styles.go`, and §10.1 records the measured
contrast for each.

## 2 · Typing

- [ ] Type a few words — they appear, the placeholder disappears
- [ ] Type `?` — **it inserts a literal `?`**. No help overlay.

  That last one is the point. A help overlay that fires mid-sentence makes the
  composer unusable; ui-spec §5.1 calls it out as easy to get wrong.

- [ ] `Alt+Enter` (or `Shift+Enter`, depending on your terminal) inserts a
      newline instead of sending
- [ ] `Ctrl+J` also inserts a newline — this is the binding that always works
- [ ] The composer grows to fit, up to five lines, then scrolls internally
- [ ] `Esc` on a non-empty composer clears it

**Terminal-specific.** Which newline binding arrives is a property of your
terminal, not of Kirsch (plan §11 amendment 40). If neither `Alt+Enter` nor
`Shift+Enter` works in a terminal you use daily, note which one — `Ctrl+J` is
the fallback and a config-selectable binding becomes worth building.

## 3 · Sending, and the fake turn

Type anything and press `Enter`.

- [ ] The message appears under a `── you ───` rule
- [ ] The wordmark scrolls away and does not come back
- [ ] Tool cards appear one at a time: a `read_file`, then a `run_command`
- [ ] The running card shows `◐` while it runs
- [ ] The status bar shows a spinner and a changing verb
- [ ] Assistant text streams in a word at a time, with a `▌` caret
- [ ] The caret disappears when the text finishes
- [ ] The composer is dimmed and refuses input while the turn is live
- [ ] An approval card appears at the end

**Spinner.** The braille spinner (`⠋⠙⠹⠸⠼`) renders inconsistently in some fonts
— this is a named risk in ui-spec §14.

- [ ] The spinner animates smoothly with no gaps or replacement boxes

## 4 · The approval card

- [ ] The card is boxed, titled `approval required`
- [ ] A `┃` gutter marks it as focused
- [ ] The key row reads `[y] approve   [a] approve for session   [n] reject   [d] detail`
- [ ] Typing anything other than `y`/`a`/`n`/`d`/`?`/Esc does nothing at all —
      **and nothing leaks into the composer**

Press `d`:

- [ ] A modal opens with a unified diff
- [ ] `+` lines are green, `-` lines red, `@@` cyan
- [ ] `j`/`k` and the arrow keys scroll; `g`/`G` jump to top/bottom
- [ ] `Esc` returns to **the approval card**, not to the composer

  That is the binding most likely to be implemented as "always go back to the
  composer" (ui-spec §4).

Press `y`:

- [ ] The card collapses to a single line ending `✓ approved`
- [ ] The composer becomes usable again

Run it again and try `n` — the card should collapse to `✗ rejected`. And once
more with a `run_command` approval and `a` — it should read
`✓ approved · session grant: go test`, and the status bar should show `1 grant`.

## 5 · Browsing the transcript

Press `↑` from the composer.

- [ ] Focus moves to the transcript; a `┃` gutter appears on a card
- [ ] `↑`/`↓` move the selection, **skipping plain text blocks**
- [ ] `Enter` on a tool card expands it into a bordered panel
- [ ] A long output ends with `‹200 of N lines — press d for full output›`
- [ ] `d` opens the full content with line numbers
- [ ] `PgUp`/`PgDn` scroll **without** moving the selection
- [ ] `End` jumps to the bottom
- [ ] `Esc` returns to the composer

**Scroll and pin.** Scroll up with `PgUp`, then return to the composer and type:

- [ ] The view does **not** jump to the bottom while you type
- [ ] A `↓ n new` indicator appears bottom-right when new content arrives
- [ ] `End` re-pins and clears the indicator

## 6 · Help

Press `?` while browsing (not in the composer).

- [ ] A two-column overlay opens: composing and browsing on the left, approval,
      modal and commands on the right
- [ ] A `debug (M1 only)` block lists `/read`, `/ls`, `/search`, `/gitstatus`,
      `/gitdiff`
- [ ] The footer shows the version and `Esc or ? closes`
- [ ] Nothing is cut off — if the bottom rows are missing, your terminal is
      under 30 rows, which is the documented minimum for this overlay
- [ ] Both `Esc` and `?` close it

## 7 · Slash commands

- [ ] `/help` opens the overlay from the composer
- [ ] `/status` prints a one-line notice with model and branch
- [ ] `/nonsense` shows a dim hint under the composer — **not** an error card
- [ ] `Tab` after `/hel` completes to `/help`

## 8 · Resizing

Drag the window around while it is running.

- [ ] Narrow it to about 60 columns — the token count disappears
- [ ] Narrow to about 45 — the branch disappears from the header
- [ ] Narrow below 40 — the transcript is replaced by `terminal too narrow`
- [ ] Widen it back — everything returns
- [ ] Make it very short — the header disappears, the status bar and composer stay
- [ ] At no point does anything panic, corrupt, or leave stray characters

## 9 · Fallbacks

```bash
NO_COLOR=1 npm start
```

- [ ] Identical layout, no colour at all
- [ ] Status is still readable from the glyphs alone (`✓`, `✗`, `◐`)

```bash
LANG=C npm start
```

- [ ] ASCII glyphs: `>` for `▸`, `[ok]` for `✓ ok`, `+`/`-`/`|` for box drawing
- [ ] The wordmark falls back to `K I R S C H`
- [ ] **The layout is the same shape** — same line count, same box positions

## 10 · Quitting

- [ ] `q` on an empty composer quits
- [ ] `Ctrl+C` once during a turn cancels it — spinner stops, composer returns
- [ ] A cancelled tool card shows `⊘`
- [ ] `Ctrl+C` twice quickly force-quits from anywhere
- [ ] Your terminal is left in a sane state — prompt intact, no stuck colours,
      echo working

That last one matters more than it looks. A TUI that corrupts the terminal on
exit is the single most annoying kind of bug, and it is invisible to every
automated test in the suite.

---

## Reporting

For anything marked ❌:

1. Which step, and what you saw instead
2. Your terminal and its size (`echo $TERM`, `tput cols`/`tput lines`)
3. A screenshot if it is visual, or the last few lines if it is textual

Rendering problems reproduce best with the debug log:

```bash
npm run start:debug
# then: cat ~/.local/state/kirsch/debug.log
```
