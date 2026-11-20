# Manual tests — UX round v0.1

These are hands-on checks for the UX round (`plan/milestones/ux-round-v0.1.md`).
Each release has its own section. The v0.1.2 steps were run by hand before its
tag. The v0.1.3 to v0.1.5 steps run together as one combined test at the end of
the round, as the operator directed on 2026-11-20. You tick a box only when the
screen shows what the step says.

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
   - [ ] The help box opens at 80×28.
2. Read the left column.
   You should see these rows, each with its key and its action text:
   - `Ctrl+G` with `bottom, re-pin`
   - `PgUp/PgDn` with `scroll` (under `composing`)
   - `PgUp/PgDn j/k` with `scroll` (under `browsing`)
   - `g/G End` with `top / bottom, re-pin`
   - `Shift+Enter` with `newline (or Ctrl+J)`
   - [ ] All the rows are present, with the text shown above.
3. Check the edges of the box.
   - [ ] No row runs past the right-hand border of the box, and the footer line
     at the bottom of the box is visible.
4. Press `Esc`. The help box closes, and the screen shows the transcript again.
   `Esc` closes only the box: you stay in Browsing.
   - [ ] `Esc` closes the help box.

If the footer is cut off at 28 rows, write that beside step 3. Do not tick it.

### 2. Page keys from the composer, while idle

**Context:** Section 1 ends in Browsing. If the help box is still open, press
`Esc` to close it. Then press `Esc` (or `Tab`) to return to the composer. Press
`Ctrl+G` to start at the bottom, whatever section 1's `Shift+↑` did. The setup
prompt has finished, so the transcript is tall enough to scroll. The key-hint row
should show the composer's keys, not card keys. The composer is empty. This
section ends in Composing.

1. Type `half a sentence` into the composer. Do not press Enter.
   Press `PgUp`.
   You should see the transcript move up by about one screen. The text
   `half a sentence` is still in the composer, and the cursor is still there.
   - [ ] The transcript scrolls up one page on `PgUp`.
   - [ ] `half a sentence` is still in the composer, unchanged.
2. Press `PgDn`.
   You should see the transcript move back down by a page, to the bottom.
   - [ ] The transcript scrolls back down on `PgDn`, and `half a sentence` is
     still in the composer.
3. Press `PgUp` again, so the view is scrolled up. Then press `Ctrl+G`.
   The view must be scrolled up before you press `Ctrl+G`, or this step cannot
   fail. You should see the view jump to the bottom, and `half a sentence` still
   in the composer.
   - [ ] `Ctrl+G` brings the view to the bottom from a scrolled-up view, and the
     composer is unchanged.
   If the view stays where it is, write that beside step 3. Do not tick it.
4. Clear the composer: press `Ctrl+C` once, or select and delete the text.
   - [ ] The composer is empty.

### 3. Page keys while a reply is streaming (model)

**Context:** Start in Composing, with the composer empty, as section 2 leaves
it. Kirsch is idle, scrolled to the bottom. The setup prompt has finished, so the
transcript is tall enough to scroll. This section ends in Composing.

1. Type `count from 1 to 500, one number per line` and press `Enter`. While the
   numbers are still streaming, press `PgUp`.
   You should see the view move up while the reply keeps streaming. The
   composer is dimmed and shows `(input disabled, Esc cancels)`.
   - [ ] `PgUp` scrolls the transcript while the reply is still streaming.
2. Press `Esc`, or wait for the reply to finish.
   - [ ] The turn ends (cancelled by `Esc`, or finished), and nothing is left
     half-typed in the composer.

### 4. Line keys in Browsing (no model)

**Context:** Start in Composing, with the composer empty, as section 3 leaves
it. Press `Ctrl+G` first, so the view is at the bottom. Kirsch is idle. The
setup prompt has finished, so the transcript is long. Step 1 enters Browsing.
This section ends in Browsing.

1. Press `Shift+↑` to enter Browsing. The view may move when you do; that is
   expected. Then press `G` so the view is at the bottom. Then press `k` five
   times. You should see the transcript move up five lines, one per press.
   - [ ] Five `k` presses move the view up five lines.
2. Press `j` five times.
   - [ ] Five `j` presses move the view back down five lines.
3. Hold `k` until the top of the transcript shows, or press `g` to go there. Then
   press `k` once more.
   - [ ] The view stops at the top and does not move further.
4. Press `G`.
   - [ ] `G` moves the view to the bottom.
5. Press `j` once more.
   - [ ] The view stays at the bottom and nothing moves.

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
   - [ ] The `↓ N new` indicator appears while scrolled up, with `N` a small
     count of blocks.
2. After the reply has finished, reach the bottom: press `PgDn` until the view
   stops moving. Do not press keys while the reply is still streaming.
   - [ ] The `↓ N new` indicator goes away at the bottom.

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
   - [ ] `j` releases the card, with the message shown above.
2. Press `Tab`.
   You should see the card armed again.
   - [ ] `Tab` re-arms the card.
3. Press `PgDn`, then `PgUp`.
   You should see the card still armed, and the transcript does not move.
   - [ ] `PgUp` and `PgDn` neither release the card nor scroll the transcript.
4. Press `n` to reject the change.
   - [ ] `n` rejects the change.
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
   - [ ] The line appears under the reply, after its text.
   - [ ] `N` is a plain number, and it matches the output limit for your model
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
   - [ ] The cut-off line is shown again after resume.

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
   - [ ] Reply text, tool cards and the final text appear in the order the model
     wrote them.

This check cannot reliably tell a correct build from a broken one by eye. The
order problem was timing-dependent, so a pass shows only that the order looked
right on this run. Say so in the report, and do not treat one pass as proof.

---

## v0.1.3 · Copy and select replies

**Context:** This section is run as part of the combined end-of-round test
(operator direction, 2026-11-20). It is not run on its own before the v0.1.3 tag.
Use the build and the test folder from the v0.1.2 section. Run `npm run build`,
then start Kirsch in `~/kirsch-ux-test` with `~/path/to/kirsch/bin/kirsch --new`.
`--new` starts a fresh session, so the transcript starts empty. Plain `kirsch`
resumes the last session instead. Check the window size with `tput cols` and
`tput lines`. The key-hint row at the bottom of the screen needs at least 24 rows,
and section 11 needs 80 columns by 28 rows.

Setup comes first, and it uses the model. Type these two prompts, one after the
other. Press `Enter` after each, and wait for Kirsch to be idle in the composer
before you type the next:

> say hello in one sentence

> say goodbye in one sentence

The transcript now holds your first message, its reply, your second message and
its reply, in that order. A start-up notice, such as `no previous session —
starting fresh`, may sit above them. A notice is a selectable item too. The
expected results below assume the transcript starts with your first message. If
a notice sits above it, write that beside section 1, step 3, because that is the
step where the `↑` press reaches the notice.

For each step, the `┃` gutter is the accent bar on the left of a selected item.
In an ASCII locale it is `|`.

### 1. Arrows stop on replies and skip your messages (no model)

**Context:** Setup is done. The composer is empty, Kirsch is idle, and nothing is
selected. This section ends in Composing, with the `┃` on the second reply.

1. Press `Shift+↑`. The focus moves to the transcript, and the last reply is
   selected.
   You should see the `┃` beside the second reply, on every row of that reply.
   - [ ] The `┃` marks the second reply, on each of its rows.
2. Press `↑`.
   You should see the `┃` move up to the first reply. Your second message sits
   between the two replies, and it gets no `┃`.
   - [ ] One `↑` moves the `┃` from the second reply to the first.
   - [ ] Your second message has no `┃` when the selection passes it.
3. Press `↑` again.
   You should see the `┃` stay on the first reply. Nothing lies above it. If a
   start-up notice sits above your first message, this press selects the notice
   instead. Write `notice above` beside this step, and leave both boxes below empty.
   - [ ] The `┃` stays on the first reply.
   - [ ] Your first message gets no `┃`.
4. Press `↓` twice. The first press moves the `┃` back to the second reply. The
   second press takes the focus back to the composer.
   - [ ] The second `↓` returns the cursor to the composer.
   - [ ] The `┃` stays on the second reply after the focus returns to the composer.

### 2. `Enter` and `d` do nothing on a reply (no model)

**Context:** Composing, with the composer empty and the `┃` on the second reply,
as section 1 leaves it. This section ends in Browsing, with the second reply
selected.

1. Press `Shift+↑` to enter Browsing.
   - [ ] The `┃` is still on the second reply.
2. Press `Enter`.
   - [ ] `Enter` does not expand or collapse the reply.
   - [ ] `Enter` opens no box.
   - [ ] `Enter` does not move the view.
3. Press `d`. You should see no box open, and the screen stays as it was.
   - [ ] `d` opens no box for the reply.
4. Read the bottom row of the screen, the key-hint row. You should see
   `↑↓ select · c copy · Tab/Esc prompt`.
   - [ ] The key-hint row reads `↑↓ select · c copy · Tab/Esc prompt`.

### 3. Copy a reply with `c`, then paste it into another app (no model)

**Context:** Browsing, with the second reply selected, as section 2 leaves it.
Open a plain text window in another app, such as TextEdit. Before each paste,
clear the text window. This section ends in Browsing, with the second reply
selected.

1. Press `c`. You should see the hint under the composer read
   `copied N characters`, with `N` a plain number. Write `N` beside this step.
   If the hint reads `sent N characters (OSC 52)` or
   `copy failed: no clipboard route`, write that beside this step and leave the
   box empty.
   - [ ] The hint under the composer reads `copied N characters`.
2. Switch to the text window. Clear it, then paste. You should see the text of the
   second reply, with the same words and punctuation.
   - [ ] The paste holds the second reply's text.
3. Switch back to Kirsch. You should see the transcript where it was.
   - [ ] The view did not move.
   - [ ] The `┃` is still on the second reply.

### 4. `c` on a card gives the hint, and copies nothing (no model)

**Context:** Browsing, with the second reply selected, as section 3 leaves it.
This section makes a notice card and selects it. This section ends in Browsing,
with the notice selected.

1. Press `Tab` to return to the composer. Type `/status` and press `Enter`.
   You should see a notice card appear at the bottom of the transcript.
   - [ ] The status notice appears in the transcript.
2. Press `Shift+↑` to enter Browsing, then press `↓`. The `┃` moves from the second
   reply to the notice card.
   - [ ] The `┃` marks the status notice.
3. Press `c`. You should see the hint `c copies replies only` under the composer.
   - [ ] The hint reads `c copies replies only`.
   - [ ] Nothing is copied.

### 5. `/copy` copies the last reply (no model)

**Context:** Browsing, with the status notice selected, as section 4 leaves it.
The last reply in the transcript is the second reply. The status notice is not a
reply, so `/copy` does not use it. This section ends in Composing, with the
composer empty. The copy notice may still show; typing clears it.

1. Press `Tab` to return to the composer. Type `/copy` and press `Enter`.
   You should see the hint under the composer read `copied N characters`. The
   transcript does not move.
   - [ ] The hint under the composer reads `copied N characters`.
   - [ ] The transcript stays where it was.
2. Clear the text window, switch to it, and paste. You should see the second reply's
   text, the same as in section 3.
   - [ ] The paste holds the second reply's text.

### 6. `/copy code` and `/copy code 2` on a reply with two code blocks (model)

**Context:** Composing, with the composer empty, as section 5 leaves it. This
section sends a prompt, so it uses the model. Type this prompt, and press `Enter`:

> Write two short Go functions. Put each in its own fenced code block with the language tag go. Put one sentence of plain text between the two blocks.

Count the lines that open a fence, which look like ```` ```go ````, in the reply.
Models vary, so the count may differ. The steps below assume two, which is the
number the prompt asks for. If the count is not two, write the count beside step 1,
and do not tick the boxes that depend on it. Wait until the reply has finished
and Kirsch is idle in the composer, with no turn running, before step 1. This
section ends in Composing, with the composer empty.

1. Type `/copy code` and press `Enter`. You should see the hint under the composer
   read `copied N characters`. Clear the text window, switch to it, and paste. You
   should see the lines of the second code block only, with no fence line and no
   `go`.
   - [ ] The hint under the composer reads `copied N characters`.
   - [ ] The paste holds the second block's lines.
   - [ ] The paste has no fence line.
   - [ ] The paste has no `go`.
2. Type `/copy code 2` and press `Enter`. Clear the text window, switch to it, and
   paste. Block 2 is the last block, so you should see the same text as step 1.
   - [ ] `/copy code 2` pastes the same text as `/copy code`.
3. Type `/copy code 1` and press `Enter`. Clear the text window, switch to it, and
   paste. You should see the lines of the first code block, and not the second.
   - [ ] The paste holds the first block's lines.
   - [ ] The paste does not hold the second block's lines.

### 7. `/copy code 9` gives the range notice (no model)

**Context:** Composing, with the composer empty, as section 6 leaves it. Kirsch is idle, and the reply has finished. The reply
with two code blocks is still the last reply. The clipboard holds the first block's
lines from section 6, step 3. This section ends in Composing, with the composer empty.

1. Type `/copy code 9` and press `Enter`. You should see the hint under the
   composer read `the last reply has 2 code blocks`.
   - [ ] The hint under the composer reads `the last reply has 2 code blocks`.
2. Clear the text window, switch to it, and paste. You should see the first block's
   lines again, because the out-of-range request copied nothing.
   - [ ] The paste still holds the first block's lines.

### 8. `/copy foo` gives the usage hint (no model)

**Context:** Composing, with the composer empty, as section 7 leaves it. This
section ends in Composing, with the composer empty and the usage hint showing.

1. Type `/copy foo` and press `Enter`. You should see the hint under the composer
   read `usage: /copy [code [N]]`.
   - [ ] The hint under the composer reads `usage: /copy [code [N]]`.
   - [ ] No new item appears in the transcript.

### 9. The copy notice clears when you type (no model)

**Context:** Composing, with the composer empty, as section 8 leaves it. The usage
hint is showing. This section ends in Composing, with the composer empty.

1. Type `/copy` and press `Enter`. You should see the hint under the composer read
   `copied N characters`.
   - [ ] The hint under the composer reads `copied N characters`.
2. Type the letter `x`. You should see the copy notice disappear, and the `x` appear
   in the composer.
   - [ ] One keypress removes the copy notice.
   - [ ] The `x` appears in the composer.
3. Press `Esc` to clear the composer.
   - [ ] The composer is empty.

### 10. `/c` and `/co` do not complete, and `/cop` does (no model)

**Context:** Composing, with the composer empty, as section 9 leaves it. This
section ends in Composing, with the composer empty.

1. Type `/c` and press `Tab`. The composer still reads `/c`.
   - [ ] `/c` does not complete.
   Type the letter `x`. The composer reads `/cx`, so the focus is still in the
   composer. Press `Esc` to clear it.
   - [ ] After `Tab`, the focus is still in the composer.
   - [ ] The `x` lands in the composer.
2. Type `/co` and press `Tab`. The composer still reads `/co`.
   - [ ] `/co` does not complete.
   Press `Esc` to clear the composer.
3. Type `/cop` and press `Tab`. You should see the composer read `/copy`.
   - [ ] `/cop` completes to `/copy`.
   Press `Esc` to clear the composer.
   - [ ] The composer is empty.

### 11. The help overlay shows the copy rows, and `?` closes it (no model)

**Context:** Composing, with the composer empty and no notice showing, as section
10 leaves it. Section 10 ends with `Esc`, which clears the composer and the hint
under it. The status notice card from section 4 is still the selected item. Kirsch
is idle. The window is 80 columns by 28 rows, as in v0.1.2 section 1. This section
ends in Browsing, with the help box closed and the hint under the composer
reading `c copies replies only`, as step 5 leaves it.

1. Press `Shift+↑` to enter Browsing, then press `?`. You should see a box titled
   `help`, with its footer `Esc or ? closes` at the bottom.
   - [ ] The help box opens.
   - [ ] Its footer reads `Esc or ? closes`.
2. Read the left column of the help box. With the composer empty and no notice
   showing, the whole box shows on the first view at 80 by 28, with the footer
   `kirsch v0.1.3-dev · Esc or ? closes`. Under `browsing`, you should see `↑/↓`
   with `select card/reply`, and `c` with `copy reply`. There is no `?` row. If a
   notice is showing under the composer, the box is one row short, and `G`, which
   moves to the bottom of the help box, shows the last rows.
   - [ ] The first view shows the whole box, from its top border to its bottom border.
   - [ ] The first view has no `↓ N more` marker.
   - [ ] The browsing block shows `↑/↓` with `select card/reply`.
   - [ ] The browsing block shows `c` with `copy reply`.
   - [ ] The browsing block has no `?` row.
3. Read the right column. Its last row in the commands list should read `/exit  /copy`.
   - [ ] The last commands row reads `/exit  /copy`.
4. Press `?`. The help box closes, and you stay in Browsing.
   - [ ] `?` closes the help box.
5. Press `c`. The card is still selected. You should see the hint under the composer
   read `c copies replies only`.
   - [ ] The hint under the composer reads `c copies replies only`.
   If nothing appears, write `no hint after help` beside this box. Do not tick it.

If a notice is showing and a row is still cut off after `G`, write that beside
step 2. Do not tick it.

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

## Known gaps after v0.1.3

These are not checked by the v0.1.3 steps above. They are listed here so the
operator knows they are open. The full list is in `plan/spec/kirsch-plan.md`,
amendment 101.

- Copying through OSC 52 works only when tmux allows it (`set-clipboard on`), and
  the terminal accepts the sequence.
- Some terminals drop a very large OSC 52 copy. The notice then names the route
  Kirsch used, not what reached the clipboard.
- `/copy code` reads only the last reply. To copy a block from an older reply,
  select that reply with `c`, which copies the whole reply.
- The screen does not number code blocks, so you count the fences yourself. From
  v0.1.4, markdown rendering may stop drawing fences, and that release revisits
  this.
- macOS Terminal.app ignores OSC 52, so the clipboard tool is the only route there.
- Over SSH, `copied N characters` can mean the text reached a clipboard on the
  remote machine, not on yours.
- On a very short terminal, the layout gives the composer no spare row for its
  hint, so a copy notice in the hint under the composer is not visible.
- A draft of five or more lines fills the composer, and then a copy notice is not
  shown under it. A draft of four lines or fewer shows the notice.
- When any notice is showing under the composer, the help overlay at 80 by 28 is
  one row short, and `G` shows its last rows.
- A very large copy can garble the screen once, until the next redraw.
