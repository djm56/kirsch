# Manual tests — UX round v0.1

These are hands-on checks for the UX round (`plan/milestones/ux-round-v0.1.md`).
Each release has its own section. You run the steps by hand before its tag, and
you tick a box only when the screen shows what the step says.

How to read a section:

- **Context** says what state the program is in before the steps. Do the setup
  first, then the steps in order.
- Each **step** says what to type or press, then what you should see. The
  *Tick* line says what to tick when it matches.
- If a step does not match, leave the box empty and write what you saw beside it.
  Do not tick it to move on.

All boxes start empty. Agents write the steps; only you tick them.

---

## v0.1.2 · Output limit, cut-off notice, keys and help

**Context:** A Terminal window, not an IDE panel. Build from the repository root
with `npm run build`, which writes `bin/kirsch`. Use a test folder that is a Git
repository, so Kirsch starts inside it:

```bash
mkdir -p ~/kirsch-ux-test && cd ~/kirsch-ux-test && git init
printf 'A test folder for Kirsch.\n' > README.md
~/path/to/kirsch/bin/kirsch
```

The `printf` line creates `README.md`. Each section that reads it names it in its Context.

Replace `~/path/to/kirsch` with the folder where you cloned Kirsch. Set the model
key you normally use. Steps that send a prompt use a real model and cost tokens.
Steps marked *no model* do not send anything.

Setup comes before section 1. Type this prompt once, in the composer, and press
`Enter`. Leave it to finish, and start section 1 when Kirsch is idle in the
composer again:

> count from 1 to 300, one number per line

That gives a transcript tall enough to scroll.

### 1. Help overlay at 80×28 (no model)

**Context:** Start in Composing, after the setup reply has finished, with the
composer empty. The window is 80 columns by 28 rows. Check the size with
`tput cols` and `tput lines` in a second Terminal tab. Kirsch is idle in the
composer, with no turn running. Step 1 enters Browsing. This section ends in
Browsing, with the help box closed.

1. Press `Shift+↑`. The focus moves to the cards. Press `?`.
   You should see a box titled `help`, with two columns. The left column starts
   with the heading `composing`, then `browsing`.
   - [✅] The help box opens at 80×28.
2. Read the left column.
   You should see these rows, each with its key and its action text:
   - `Ctrl+G` with `bottom, re-pin`
   - `PgUp/PgDn` with `scroll` (under `composing`)
   - `PgUp/PgDn j/k` with `scroll` (under `browsing`)
   - `g/G End` with `top / bottom, re-pin`
   - `Shift+Enter` with `newline (or Ctrl+J)`
   - [✅] All the rows are present, with the text shown above.
3. Check the edges of the box.
   - [✅] No row runs past the right-hand border of the box, and the footer line
     at the bottom of the box is visible.
4. Press `Esc`. The help box closes, and the screen shows the transcript again.
   `Esc` closes only the box: you stay in Browsing.
   - [✅] `Esc` closes the help box.

If the footer is cut off at 28 rows, write that beside step 3. Do not tick it.

### 2. Page keys from the composer, while idle

**Context:** Section 1 ends in Browsing. If the help box is still open, press
`Esc` to close it. Then press `Esc` (or `Tab`) to return to the composer. Press
`Ctrl+G` to start at the bottom, whatever section 1's `Shift+↑` did. The setup
prompt has finished, so the transcript is tall enough to scroll. The bottom row
should show the composer hint, not card keys. The composer is empty. This
section ends in Composing.

1. Type `half a sentence` into the composer. Do not press Enter.
   Press `PgUp`.
   You should see the transcript move up by about one screen. The text
   `half a sentence` is still in the composer, and the cursor is still there.
   - [✅] The transcript scrolls up one page on `PgUp`.
   - [✅] `half a sentence` is still in the composer, unchanged.
2. Press `PgDn`.
   You should see the transcript move back down by a page, to the bottom.
   - [✅] The transcript scrolls back down on `PgDn`, and `half a sentence` is
     still in the composer.
3. Press `PgUp` again, so the view is scrolled up. Then press `Ctrl+G`.
   The view must be scrolled up before you press `Ctrl+G`, or this step cannot
   fail. You should see the view jump to the bottom, and `half a sentence` still
   in the composer.
   - [✅] `Ctrl+G` brings the view to the bottom from a scrolled-up view, and the
     composer is unchanged.
   If the view stays where it is, write that beside step 3. Do not tick it.
4. Clear the composer: press `Ctrl+C` once, or select and delete the text.
   - [✅] The composer is empty.

### 3. Page keys while a reply is streaming (model)

**Context:** Start in Composing, with the composer empty, as section 2 leaves
it. Kirsch is idle, scrolled to the bottom. The setup prompt has finished, so the
transcript is tall enough to scroll. This section ends in Composing.

1. Type `count from 1 to 500, one number per line` and press `Enter`. While the
   numbers are still streaming, press `PgUp`.
   You should see the view move up while the reply keeps streaming. The
   composer is dimmed and shows `(input disabled, Esc cancels)`.
   - [✅] `PgUp` scrolls the transcript while the reply is still streaming.
2. Press `Esc`, or wait for the reply to finish.
   - [✅] The turn ends (cancelled by `Esc`, or finished), and nothing is left
     half-typed in the composer.

### 4. Line keys in Browsing (no model)

**Context:** Start in Composing, with the composer empty, as section 3 leaves
it. Press `Ctrl+G` first, so the view is at the bottom. Kirsch is idle. The
setup prompt has finished, so the transcript is long. Step 1 enters Browsing.
This section ends in Browsing.

1. Press `Shift+↑` to enter Browsing. The view may move when you do; that is
   expected. Then press `G` so the view is at the bottom. Then press `k` five
   times. You should see the transcript move up five lines, one per press.
   - [✅] Five `k` presses move the view up five lines.
2. Press `j` five times.
   - [✅] Five `j` presses move the view back down five lines.
3. Hold `k` until the top of the transcript shows, or press `g` to go there. Then
   press `k` once more.
   - [✅] The view stops at the top and does not move further.
4. Press `G`.
   - [✅] `G` moves the view to the bottom.
5. Press `j` once more.
   - [✅] The view stays at the bottom and nothing moves.

Card selection is not checked by hand in this section. That `j` and `k` do not
change the selected card is checked by the automated test
`TestBrowsingJKScrollOneRowWithoutMovingSelection` (`internal/tui/keyscroll_test.go`),
so there is nothing to check by hand here.

### 5. Scrolled up while a reply streams, then back down (model)

**Context:** Press `Esc` to return to the composer. Section 4 ends in Browsing,
and step 1 below types a prompt, which needs the composer. Kirsch is idle,
scrolled to the bottom, in the test folder that has `README.md`. The transcript
must already be taller than the screen before step 1. The setup reply, or an
earlier long reply, provides that. Step 1 scrolls up as soon as the first tool
card shows, before its own reply has arrived, so the transcript cannot be short
at that point. This section ends in Browsing.

1. Type `read README.md, then count from 1 to 300, one number per line` and press
   `Enter`. As soon as the first tool card shows, press `Shift+↑`, then `k` three
   times, to scroll up while the reply keeps going.
   When the reply has finished, you should see the `↓ N new` indicator at the
   bottom right. `N` counts the blocks that arrived while you were scrolled up
   (tool cards and messages). It is a small number. It is not a count of lines.
   If `N` is near the number of lines in the reply, the build is counting lines,
   and this step fails.
   If no tool card appears, the model answered without calling a tool. Write
   `no tool call` beside this step, and send the prompt again.
   If no indicator appears, write `reply finished first` beside this step and
   try again with a longer prompt.
   - [✅] The `↓ N new` indicator appears while scrolled up, with `N` a small
     count of blocks.
2. After the reply has finished, reach the bottom: press `PgDn` until the view
   stops moving. Do not press keys while the reply is still streaming.
   - [✅] The `↓ N new` indicator goes away at the bottom.

### 6. Approval card: `j`, `k`, `PgUp`, `PgDn` (model)

**Context:** Press `Esc` to return to the composer. Section 5 ends in Browsing.
Kirsch is idle, in the test folder, with the composer empty. The approval card
appears when a change is requested, so type this prompt in the composer and
press `Enter`:

> create a file named ux-test.txt containing the word hello

Wait until the approval card shows, with `[y]`, `[n]` and `[d]` in it. `d` and
`?` open a box over the card, and `Esc` closes that box and returns to the card.
This section ends in Composing, once the turn has ended and no approval card is
showing.

1. Press `j`.
   You should see the bottom of the composer say `approval pending — Tab to return to the card`.
   - [✅] `j` releases the card, with the message shown above.
2. Press `Tab`.
   You should see the card armed again.
   - [✅] `Tab` re-arms the card.
3. Press `PgDn`, then `PgUp`.
   You should see the card still armed, and the transcript does not move.
   - [✅] `PgUp` and `PgDn` neither release the card nor scroll the transcript.
4. Press `n` to reject the change.
   - [✅] `n` rejects the change.
   If another approval card appears, the model has asked for the change again.
   That is not a failure. Press `n` on each further card until the turn ends, and
   wait until Kirsch is idle in the composer before section 7.

### 7. Reply cut off at the output limit (model)

**Context:** Start in Composing, with the composer empty, as section 6 leaves it:
the turn has ended and no approval card is showing. Kirsch is idle, using the
model you normally use. The output limit
depends on that model. Models in Kirsch's table have their own limit, and a
model that is not in the table gets 16384 tokens. Note the model name in the
margin before you start.

There is no reliable way to force a cut-off. A long request is the best try. If
the reply finishes before the limit, this section is **not reached**: write
`not reached` beside it, leave the boxes empty, and try again with a longer
request.

1. Type `count from 1 to 3000, one number per line` and press `Enter`.
   If the reply stops early, you should see this dim line directly under the
   reply's text:
   `· reply cut off — output limit reached (N tokens)`
   `N` is a plain number, with no commas or spaces.
   - [✅] The line appears under the reply, after its text.
   - [✅] `N` is a plain number, and it matches the output limit for your model
     (from the table, or 16384 for a model not in the table).
2. If the line is missing, check the reply's end. Did it finish with the last
   number, `3000`?
   - [ ] If the reply finished with `3000`, this step is **not reached**. Write
     that beside it and leave the boxes empty.
3. Type `/quit` and press `Enter`. Then, in the same test folder, run:
   ```bash
   ~/path/to/kirsch/bin/kirsch resume
   ```
   You should see the same reply again, with the same line under it.
   - [✅] The cut-off line is shown again after resume.

**Not checked by hand:** a reply made only of thinking blocks that is cut off
shows the line live but not after resume. Its test is
`TestThinkingOnlyReplyCutOffShowsNoticeLiveButNotOnResume`
(`internal/app/cutoff_notice_test.go`). There is no safe manual trigger for it.

### 8. Replies appear in order (model)

**Context:** Start in Composing, in Kirsch running in the test folder. Section 7
ends with Kirsch running in Composing, in the test folder. If Kirsch is still
running in that terminal, stay in it. If it has exited, start Kirsch with
`~/path/to/kirsch/bin/kirsch` in the test folder. Kirsch is idle, and the test
folder has `README.md`, which the `printf` line in the opening Context creates.
The order check depends on the model, so write the model name beside this
section. This section ends in Composing.

1. Type `read README.md, then say in one sentence what it is for` and press
   `Enter`. Watch the screen as the reply comes in.
   You should see the text the model wrote before a tool call above that tool's
   card, and the text after the call below it. No line appears out of place.
   - [✅] Reply text, tool cards and the final text appear in the order the model
     wrote them.

This check cannot reliably tell a correct build from a broken one by eye. The
order problem was timing-dependent, so a pass shows only that the order looked
right on this run. Say so in the report, and do not treat one pass as proof.

---

## Known gaps after v0.1.2

These are not checked by the steps above. They are listed here so the operator
knows they are open. The full list is in `plan/milestones/ux-round-v0.1.md`.

- The context budget reserves 4096 output tokens, while a request may ask for
  16384. After compaction, input plus `max_tokens` can exceed the window. Not
  changed in v0.1.2.
- On a terminal too small to show the transcript, `PgUp` and `PgDn` from the
  composer clear the `↓ N new` indicator.
- The no-scroll test for the approval prompt depends on a mutant, not on a
  control run through the same path.
