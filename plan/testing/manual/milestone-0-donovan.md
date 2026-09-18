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
- [✅] The header is **two rows**: `Kirsch` alone on the first, with a rule running
      off to the right edge, and `<your repo> ─ <your branch>` on the second

  *Changed since the last run.* It used to be one row reading
  `Kirsch ─ <repo> ─ <branch>`, which at 40 columns spent most of its width on
  text rather than rule. The wordmark and the session identity now have a row
  each. Only the first row carries a rule; a second one directly beneath would
  read as the top edge of a box.
- [✅] A `●` follows the branch on **row 2** if the repo has uncommitted changes.

  *Re-marked: the dot is orange by design, not by accident.* `header.dirty` is
  the `warning` token — 215, `#ffaf5f` — in `kirsch-ui-screens.md` §00 and in the
  palette table there, the same token as `approval required`, the 200-line cap
  marker, and status-bar warnings. Nothing in the spec ever made it cyan. Recorded
  here so the expectation is written down rather than rediscovered next run.
- [✅] The composer shows the placeholder `Ask anything (Enter to send, /help for help)`
- [✅] Nothing is cut off, and no line wraps oddly

**The frame margin — check this at a glance.** Kirsch now keeps one blank column
at the left edge and one at the right, and none at the top or bottom. The easiest
way to see it is the header rule and the two separator rules: they are the only
full-width things on screen.

- [✅] The header rule on row 1 starts **one column in** from the left edge, not
      flush against it — every row does, but the rule is where it is obvious
- [✅] The same rule stops **one column short** of the right edge. Put a window
      border, a split pane or a coloured terminal background beside it and you
      should see a sliver of your own background at both ends, not Kirsch's
- [✅] There is **no** blank row above the header or below the composer. The
      margin is horizontal only — rows are too expensive to spend, columns are
      not
- [✅] Select the whole window and copy it, or run `npm start | cat -A` style
      inspection if your terminal offers it: no row should end in trailing
      spaces. The right column is *reserved*, not painted

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
- [✅] `End` re-pins and clears the indicator — **while browsing**, not from the
      composer
- [✅] `Ctrl+G` re-pins and clears the indicator **from the composer**, without
      leaving it and without sending anything

  *`Esc` was the cause of the first two, and was fixed last run.* `Esc` used to
  re-pin, and it is the only way back to the composer that does not first walk
  the selection past the last card — so the act of returning to type undid the
  scroll. `Esc` no longer re-pins (ui-spec §2.4).

  *`Ctrl+G` is new this run, and it is the composer's re-pin key.* `End` and `G`
  re-pin in **Browsing** mode only; in the composer `End` belongs to the text
  area and moves the cursor to the end of the line, which is why pressing it
  there looked like a broken re-pin. There was no single key that took the view
  back to the bottom without first sending something — `↑` then `End` did it in
  two, by way of a mode you did not want. `Ctrl+G` is that key. Re-pinning is now
  `Ctrl+G` from the composer, `End` or `G` while browsing, scrolling back to the
  bottom, sending a message, or submitting a slash command.

## 6 · Help

Press `?` while browsing (not in the composer).

- [✅] A two-column overlay opens: composing and browsing on the left, approval,
      modal and commands on the right
- [✅] A `debug (M1 only)` block lists `/read`, `/ls`, `/search`, `/gitstatus`,
      `/gitdiff`
- [✅] The footer shows the version and `Esc or ? closes`
- [✅] Nothing is cut off — if the bottom rows are missing, your terminal is
      under **32 rows**, which is the documented minimum for this overlay. It was
      31 before the header grew to two rows
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
- [✅] The one-column margin at each edge is still there at **every** width,
      including under the `terminal too narrow` notice — the frame does not
      change shape at the moment it is most degraded
- [✅] Widen it back — everything returns
- [✅] Make it very short — the header disappears **as a unit, both rows at once**,
      and the status bar and composer stay
- [✅] Shrinking from 10 rows to 9 makes the transcript *one row taller*, not
      shorter — **expected, not a defect.** The two-row header goes at that step
      and only four rows of chrome remain. ui-spec §2.2 records why the band
      stays at 10: 40×10 is the advertised minimum and has to keep its header
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

Everything below covers what changed in mission-20260914-02 and
mission-20260915-02, and nothing else. A box left empty is one this run has to
answer; a ✅ is a check already carried out and kept here for context.
Sections §1–§10 remain the record of what was fixed and why it was fixed that
way; §11 is the runnable pass — the shortest route to confirming none of it has
come undone. Start from a fresh `npm start` unless a check says otherwise.

### 11.1 · Start-up and input handling

New this round, and the group with the least automated cover behind it.

- [✅] The app responds to keystrokes immediately — no `Enter` needed to make a
      keypress register
- [✅] Typed characters do not echo over the interface
- [✅] `Ctrl+C` is handled by the app rather than killing it from the shell
- [✅] `kirsch < /dev/null` starts, and still reads keys from the terminal
- [✅] `echo hi | npm start` starts, and does **not** consume the piped text as
      keystrokes
- [✅] Quitting leaves the terminal sane — prompt intact, echo working, no stuck
      colours

  *Why this group exists.* A wrapper defect found in review would have left the
  terminal in canonical mode — input line-buffered, typed characters echoing
  over the interface, `Ctrl+C` arriving as a signal rather than a key — and the
  same wrapper was delivering piped stdin as keystrokes. Neither defect shipped.
  Both are worth a look anyway, because no automated test covers either one; the
  only way to know is to sit in front of it.

### 11.2 · Overlays close cleanly

One `Esc`, not two.

- [✅] The diff modal restores full colour on the same keypress that closes it
- [✅] The help overlay restores full colour on the same keypress that closes it
- [✅] A confirm prompt restores full colour on the same keypress that closes it
- [✅] Opening an overlay dims the transcript on the same frame the overlay
      appears — no bright flash first

### 11.3 · Transcript navigation

- [✅] `g` jumps to the top of the transcript
- [✅] `G` jumps to the bottom
- [✅] `Home` jumps to the top
- [✅] `End` jumps to the bottom
- [✅] `PgUp`/`PgDn` still scroll **without** moving the selection
- [✅] Pressing `↑` from the composer draws the `┃` selection gutter on that same
      frame, not one keypress later

### 11.4 · Scroll and pin

- [✅] Scroll up with `PgUp`, then press `Esc` to return to the composer — the
      view stays where it was
- [✅] Type — it still stays where it was
- [✅] New content arriving while scrolled up raises a `↓ n new` indicator
      bottom-right
- [ ] Press `↑` to browse, then `End` — it re-pins and clears the indicator
- [ ] Still browsing, scroll up again and press `G` — same result
- [ ] Back in the composer (`Esc`), scroll up with `PgUp`, then press **`Ctrl+G`**
      — the view re-pins and the indicator clears, and you are still in the
      composer
- [ ] `Ctrl+G` leaves the composer's text exactly as it was — type `hello`,
      scroll up, press `Ctrl+G`, and `hello` is still there, uncommitted
- [ ] `Ctrl+G` sends nothing — no new user message appears in the transcript
- [ ] `Ctrl+G` **during a running turn** re-pins too. Send a message, scroll up
      while the assistant is streaming, press `Ctrl+G`: the view returns to the
      bottom even though the composer is input-disabled
- [ ] `g` and `G` typed in the composer are still ordinary characters — they do
      not scroll anything
- [✅] Scroll up, then submit `/status` — the view re-pins so the answer is visible
- [✅] Scroll up, then submit `/nonsense` — the view does **not** re-pin, because
      a hint is not transcript content

*Both ❌s from the last run were the same misunderstanding, and it was the
program's, not the operator's.* `End` and `G` re-pin in **Browsing** mode, and
they always did; pressed in the composer, `End` is the text area's own
end-of-line key and `G` is a literal character, so neither could have re-pinned
there. What was actually missing was any re-pin key reachable from the composer
at all — every one that existed put something in the transcript first. The fix
is a dedicated binding rather than a restoration: `Ctrl+G`, live even during a
turn. The checks above are split by mode so the two can no longer be confused.

**Why `Ctrl+G` and not `End`.** All three of `End`, `Ctrl+End` and `Ctrl+G` spell
"go to the bottom", but bubbles already binds the first two inside the text area,
so taking either would buy a scroll at the price of a cursor movement in a
composer that can be five rows tall. `Ctrl+G` is also a single control byte,
where `Ctrl+End` is a modified-key escape sequence — and this walkthrough has
already found one terminal (macOS Terminal) that does not send `Home`/`End` in a
form Bubble Tea decodes.

**Known gap:** the help overlay's `composing` block does not list `Ctrl+G` yet.
Its grid is a byte-for-byte test oracle, so adding the row is a spec amendment
and a redraw rather than a code change. Do not report it as a defect.

### 11.5 · The scripted turn

- [✅] The `run_command` card holds `◐` for about 2.4 seconds with the spinner
      running, then flips to `✗ exit 1 · 2.4s`
- [✅] The turn ends on two approvals, `apply_patch` first and `run_command` after
- [✅] The patch card's key row reads `[y] approve   [n] reject   [d] view diff`,
      with no `[a]`
- [✅] Answering the patch card releases a `run_command` approval whose key row
      reads `[y] approve   [a] approve for session   [n] reject   [d] detail`
- [✅] `d` on the patch card opens a diff
- [✅] `d` on the command card opens a detail modal with no diff in it
- [✅] `a` on the command card collapses it to
      `✓ approved · session grant: go test`, and the status bar shows `1 grant`

### 11.6 · Quitting

- [✅] `q` on an empty composer types a `q`
- [✅] `/quit` quits
- [✅] `/exit` quits
- [✅] `Tab` after `/e` completes to `/exit`
- [✅] `Ctrl+C` twice quickly force-quits from anywhere

### 11.7 · Help overlay

- [✅] At 32 rows or more, nothing is cut off and `/gitdiff` is visible at the
      bottom
- [✅] At exactly 31 rows, `/gitdiff` falls below the fold and `j`/`↓` scrolls it
      into view — **expected, not a defect.** The overlay is a 26-row box: two
      borders, 22 body lines, a divider rule and the footer. The frame spends six
      more rows around it — the **two-row** header, the two separator rules, the
      status bar and the composer — so 32 is where the whole box fits. The
      documented minimum was 31 when the header was one row; the header is two
      rows now, so it is 32. Below that the box keeps its borders and its footer
      and gives up body rows, which is why the overlay stays leaveable at any
      height. Confirm the behaviour; do not report it.
- [✅] The commands block lists `/quit`
- [✅] The commands block lists `/exit`

A ❌ anywhere in §11 is reported exactly as a ❌ anywhere else in this document —
follow the **Reporting** block below.

Additional Reporting
I am thinking of adding some padding around the entire window so it si not touching the sides so the entire window has 20px or whatever the equivalent is around the entire kirsh terminal window so top, left, bottom and right.
Also maybe a line rather below the top kirsh that has the folder name and the repo it has a line to the right of the text but maybe lets add it below.

**Both actioned, the first one partly.** The header is two rows now — `Kirsch`
and its rule on the first, the folder name and the branch on the second, exactly
as asked. The frame has a one-column margin at the left and right edges.

The part that was not done is the top and bottom, and it was a deliberate call
rather than an oversight. A terminal's rows are not like a window's pixels: the
chrome budget already spends every row the terminal has, so a blank row at the
top or the bottom comes straight out of the transcript — and at the smallest
supported terminal, 40×10, the transcript is down to four rows already. A column
is cheap at every width Kirsch supports; a row is not. If the top and bottom
still feel worth a row at a comfortable window size, say so and it can be made
conditional on height rather than unconditional.

Check both under **§1 · First run** above, and the margin at every width under
**§8 · Resizing**.

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
