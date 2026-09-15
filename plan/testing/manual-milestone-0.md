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

- [✅] The wordmark renders as two rows of solid blocks — no gaps, no tofu boxes
- [✅] The tagline reads `v0.1.0-dev · terminal-native coding agent`
- [✅] Three suggestions appear under "Ask anything. Three things to try:"
- [✅] The header shows `Kirsch ─ <your repo> ─ <your branch>`
- [✅] A `●` follows the branch if the repo has uncommitted changes.

  *Re-marked: the dot is orange by design, not by accident.* `header.dirty` is
  the `warning` token — 215, `#ffaf5f` — in `kirsch-ui-screens.md` §00 and in the
  palette table there, the same token as `approval required`, the 200-line cap
  marker, and status-bar warnings. Nothing in the spec ever made it cyan. Recorded
  here so the expectation is written down rather than rediscovered next run.
- [✅] The composer shows the placeholder `Ask anything (Enter to send, /help for help)`
- [✅] Nothing is cut off, and no line wraps oddly

**Legibility — the one worth taking seriously.** The palette was retuned in
Milestone 0 after the first version measured below the readable contrast
threshold (plan §11 amendment 41). Look at it on your actual background:

- [✅] The three suggestions are readable, not washed-out grey
- [✅] The separator rules are visible but not shouting
- [✅] The header title reads as distinct from the body text

If any of that looks wrong on your theme, say which and the palette moves. It is
one line per token in `internal/tui/styles.go`, and §10.1 records the measured
contrast for each.

## 2 · Typing

- [✅] Type a few words — they appear, the placeholder disappears
- [✅] Type `?` — **it inserts a literal `?`**. No help overlay.

  That last one is the point. A help overlay that fires mid-sentence makes the
  composer unusable; ui-spec §5.1 calls it out as easy to get wrong.

- [✅] `Alt+Enter` (or `Shift+Enter`, depending on your terminal) inserts a
      newline instead of sending
- [✅] `Ctrl+J` also inserts a newline — this is the binding that always works
- [✅] The composer grows to fit, up to five lines, then scrolls internally
- [✅] `Esc` on a non-empty composer clears it

**Terminal-specific.** Which newline binding arrives is a property of your
terminal, not of Kirsch (plan §11 amendment 40). If neither `Alt+Enter` nor
`Shift+Enter` works in a terminal you use daily, note which one — `Ctrl+J` is
the fallback and a config-selectable binding becomes worth building.

## 3 · Sending, and the fake turn

Type anything and press `Enter`.

- [✅] The message appears under a `── you ───` rule
- [✅] The wordmark scrolls away and does not come back
- [✅] Tool cards appear one at a time: a `read_file`, then a `run_command`
- [✅] The `run_command` card shows `◐` and holds it for 2.4 seconds, with the
      status bar spinning on `running go test`, then flips to `✗ exit 1 · 2.4s`

  *Re-marked: the glyph was correct and unseeable.* The scripted turn used to
  advance on the stream tick alone, so `StateRunning` lasted a single 50ms frame.
  It now dwells deliberately — long enough to read.
- [✅] The status bar shows a spinner and a changing verb
- [✅] Assistant text streams in a word at a time, with a `▌` caret
- [✅] The caret disappears when the text finishes
- [✅] The composer is dimmed and refuses input while the turn is live
- [✅] An approval card appears at the end

**Spinner.** The braille spinner (`⠋⠙⠹⠸⠼`) renders inconsistently in some fonts
— this is a named risk in ui-spec §14.

- [✅] The spinner animates smoothly with no gaps or replacement boxes

## 4 · The approval cards — there are two

The turn ends on **two** approvals, one after the other. The first is the patch,
which never offers `[a]`; answering it releases the second, a `run_command`, which
does. ADR 0006 makes those two the whole approval surface, and until the scripted
turn built both, `[a]` appeared in the golden tests and nowhere a person could
reach — which is what the ❌ here last run was seeing.

**First card — `apply_patch — add input validation`:**

- [✅] The card is boxed, titled `approval required`
- [✅] A `┃` gutter marks it as focused
- [✅] The key row reads `[y] approve   [n] reject   [d] view diff` —
      **no `[a]`**, because a patch never offers a session grant
- [✅] Typing anything other than `y`/`n`/`d`/`?`/Esc does nothing at all —
      **and nothing leaks into the composer**

Press `d`:

- [✅] A modal opens with a unified diff
- [✅] `+` lines are green, `-` lines red, `@@` cyan
- [✅] `j`/`k` and the arrow keys scroll; `g`/`G` jump to top/bottom
- [✅] `Esc` returns to **the approval card**, not to the composer
- [✅] Closing the modal restores full colour **on the same keypress** — no
      second `Esc` needed

  That first one is the binding most likely to be implemented as "always go back
  to the composer" (ui-spec §4).

Press `y`:

- [✅] The card collapses to a single line ending `✓ approved`

**Second card — `run_command — go test ./...`, which arrives immediately:**

- [✅] The key row reads
      `[y] approve   [a] approve for session   [n] reject   [d] detail` —
      `[a]` is present this time
- [✅] `d` opens a detail modal (cwd, timeout, reason), not a diff
- [✅] Pressing `a` collapses it to `✓ approved · session grant: go test`, and
      the status bar shows `1 grant`
- [✅] The composer becomes usable again once both are answered

Worth a second run: answer the first card with `n` and check it collapses to
`✗ rejected`, and the second with `y` to see the plain `✓ approved` form without
a grant.

## 5 · Browsing the transcript

Press `↑` from the composer.

- [✅] Focus moves to the transcript; a `┃` gutter appears on a card
- [✅] `↑`/`↓` move the selection, **skipping plain text blocks**
- [✅ ] `Enter` on a tool card expands it into a bordered panel
- [✅] A long output ends with `‹200 of N lines — press d for full output›`
- [✅] `d` opens the full content with line numbers
- [✅] `PgUp`/`PgDn` scroll **without** moving the selection
- [✅] `Home`/`End` jump to top/bottom — these now work
- [✅] `g`/`G` do the same, and are the ones to use if your terminal swallows
      `Home`/`End`
- [✅] `Esc` returns to the composer

  *Both were fixed since the last run.* On macOS Terminal, `Home` and `End`
  arrive as SS3 sequences that Bubble Tea v1.3.10 does not decode, so they were
  reaching no handler at all — hence "the home and end keys did not work on the
  keyboard" last time. They are decoded now. `g`/`G` were added alongside, mirroring
  the modal's own bindings rather than inventing a second vocabulary, so there is a
  binding that works regardless of terminal.

**Scroll and pin.** Scroll up with `PgUp`, then return to the composer with `Esc`
and type:

- [✅] The view does **not** jump to the bottom while you type
- [✅] The view does **not** jump to the bottom when you press `Esc` either
- [✅] A `↓ n new` indicator appears bottom-right when new content arrives
- [✅] `End` re-pins and clears the indicator

  *All of these were ❌ last run, and `Esc` was the cause of it.* `Esc` used to
  re-pin, and it is the only way back to the composer that does not first walk
  the selection past the last card — so the act of returning to type undid the
  scroll, and every check below it looked broken. `Esc` no longer re-pins
  (ui-spec §2.4). Re-pinning is now `End`, `G`, scrolling back to the bottom,
  sending a message, or submitting a slash command.

## 6 · Help

Press `?` while browsing (not in the composer).

- [✅] A two-column overlay opens: composing and browsing on the left, approval,
      modal and commands on the right
- [✅] A `debug (M1 only)` block lists `/read`, `/ls`, `/search`, `/gitstatus`,
      `/gitdiff`
- [✅] The footer shows the version and `Esc or ? closes`
- [✅] Nothing is cut off — if the bottom rows are missing, your terminal is
      under **31 rows**, which is the documented minimum for this overlay
- [✅] The commands block lists `/quit` and `/exit`
- [✅] Both `Esc` and `?` close it, and full colour returns on that same keypress

## 7 · Slash commands

- [✅] `/help` opens the overlay from the composer
- [✅] `/status` prints a one-line notice with model and branch
- [✅] `/nonsense` shows a dim hint under the composer — **not** an error card
- [✅] `Tab` after `/hel` completes to `/help`
- [✅] `Tab` after `/e` completes to `/exit`
- [✅] Scroll up, then run `/status` — the view re-pins so the notice is on screen
- [✅] Scroll up, then run `/nonsense` — the view stays exactly where it is,
      because a hint is not transcript content

*The grey-out is fixed.* Closing any overlay used to leave the whole window
dimmed with nothing drawn over it until the next keypress rebuilt the frame,
which read as "Esc twice to get the colours back". `dispatchKey` now rebuilds
whenever a handler changes the overlay state, so one `Esc` is enough everywhere.

## 8 · Resizing

Drag the window around while it is running.

- [✅] Narrow it to about 60 columns — the token count disappears
- [✅] Narrow to about 45 — the branch disappears from the header
- [✅] Narrow below 40 — the transcript is replaced by `terminal too narrow`
- [✅] Widen it back — everything returns
- [✅] Make it very short — the header disappears, the status bar and composer stay
- [✅] At no point does anything panic, corrupt, or leave stray characters

## 9 · Fallbacks

```bash
NO_COLOR=1 npm start
```

- [✅] Identical layout, no colour at all
- [✅] Status is still readable from the glyphs alone (`✓`, `✗`, `◐`)

```bash
LANG=C npm start
```

- [✅] ASCII glyphs: `>` for `▸`, `[ok]` for `✓ ok`, `+`/`-`/`|` for box drawing
- [✅] The wordmark falls back to `K I R S C H`
- [✅] **The layout is the same shape** — same line count, same box positions

## 10 · Quitting

- [✅] Typing `q` on an empty composer **types a `q`** — it does not quit
- [✅] `/quit` quits
- [✅] `/exit` quits too — it is an alias, sharing one arm rather than a second
      behaviour
- [✅] `Ctrl+C` once during a turn cancels it — spinner stops, composer returns
- [✅] A cancelled tool card shows `⊘`
- [✅] `Ctrl+C` twice quickly force-quits from anywhere
- [✅] Your terminal is left in a sane state — prompt intact, no stuck colours,
      echo working

That last one matters more than it looks. A TUI that corrupts the terminal on
exit is the single most annoying kind of bug, and it is invisible to every
automated test in the suite.

Other Findings
I dont think typing a q on its own should quit the terminal what if I am trying in the command are and I start with a word that starts with q then it wouyld quit instantly, it makes more sense to have a /quit command or a /exit command. Also when closing modals the modal closes but the existing text remains dull grey I have to eascape a second time for proper colors to come back.

**Both actioned.** The bare `q` binding is gone — `q` is now an ordinary character
— and `/quit` and `/exit` are the quit commands, listed in the help overlay. The
second `Esc` is no longer needed anywhere: closing an overlay rebuilds the frame
on the same keypress. See ui-spec §5.2 and §6, and plan §11 amendments 55–59.

## 11 · Regression pass

Everything below covers what changed in mission-20260914-02, and nothing else.
Sections §1–§10 remain the record of what was fixed and why it was fixed that
way; §11 is the runnable pass — the shortest route to confirming none of it has
come undone. Start from a fresh `npm start` unless a check says otherwise.

### 11.1 · Start-up and input handling

New this round, and the group with the least automated cover behind it.

- [ ] The app responds to keystrokes immediately — no `Enter` needed to make a
      keypress register
- [ ] Typed characters do not echo over the interface
- [ ] `Ctrl+C` is handled by the app rather than killing it from the shell
- [ ] `kirsch < /dev/null` starts, and still reads keys from the terminal
- [ ] `echo hi | npm start` starts, and does **not** consume the piped text as
      keystrokes
- [ ] Quitting leaves the terminal sane — prompt intact, echo working, no stuck
      colours

  *Why this group exists.* A wrapper defect found in review would have left the
  terminal in canonical mode — input line-buffered, typed characters echoing
  over the interface, `Ctrl+C` arriving as a signal rather than a key — and the
  same wrapper was delivering piped stdin as keystrokes. Neither defect shipped.
  Both are worth a look anyway, because no automated test covers either one; the
  only way to know is to sit in front of it.

### 11.2 · Overlays close cleanly

One `Esc`, not two.

- [ ] The diff modal restores full colour on the same keypress that closes it
- [ ] The help overlay restores full colour on the same keypress that closes it
- [ ] A confirm prompt restores full colour on the same keypress that closes it
- [ ] Opening an overlay dims the transcript on the same frame the overlay
      appears — no bright flash first

### 11.3 · Transcript navigation

- [ ] `g` jumps to the top of the transcript
- [ ] `G` jumps to the bottom
- [ ] `Home` jumps to the top
- [ ] `End` jumps to the bottom
- [ ] `PgUp`/`PgDn` still scroll **without** moving the selection
- [ ] Pressing `↑` from the composer draws the `┃` selection gutter on that same
      frame, not one keypress later

### 11.4 · Scroll and pin

- [ ] Scroll up with `PgUp`, then press `Esc` to return to the composer — the
      view stays where it was
- [ ] Type — it still stays where it was
- [ ] New content arriving while scrolled up raises a `↓ n new` indicator
      bottom-right
- [ ] `End` re-pins and clears the indicator
- [ ] `G` re-pins and clears the indicator
- [ ] Scroll up, then submit `/status` — the view re-pins so the answer is visible
- [ ] Scroll up, then submit `/nonsense` — the view does **not** re-pin, because
      a hint is not transcript content

### 11.5 · The scripted turn

- [ ] The `run_command` card holds `◐` for about 2.4 seconds with the spinner
      running, then flips to `✗ exit 1 · 2.4s`
- [ ] The turn ends on two approvals, `apply_patch` first and `run_command` after
- [ ] The patch card's key row reads `[y] approve   [n] reject   [d] view diff`,
      with no `[a]`
- [ ] Answering the patch card releases a `run_command` approval whose key row
      reads `[y] approve   [a] approve for session   [n] reject   [d] detail`
- [ ] `d` on the patch card opens a diff
- [ ] `d` on the command card opens a detail modal with no diff in it
- [ ] `a` on the command card collapses it to
      `✓ approved · session grant: go test`, and the status bar shows `1 grant`

### 11.6 · Quitting

- [ ] `q` on an empty composer types a `q`
- [ ] `/quit` quits
- [ ] `/exit` quits
- [ ] `Tab` after `/e` completes to `/exit`
- [ ] `Ctrl+C` twice quickly force-quits from anywhere

### 11.7 · Help overlay

- [ ] At 31 rows or more, nothing is cut off and `/gitdiff` is visible at the
      bottom
- [ ] At exactly 30 rows, `/gitdiff` falls below the fold — **expected, not a
      defect.** Adding `/exit` took the overlay's body past what 30 rows hold, and
      plan §11 amendment 59 raises the documented minimum to 31. Confirm the
      behaviour; do not report it.
- [ ] The commands block lists `/quit`
- [ ] The commands block lists `/exit`

A ❌ anywhere in §11 is reported exactly as a ❌ anywhere else in this document —
follow the **Reporting** block below.

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
