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

## v0.1.4 · Markdown replies and code highlighting

**Draft, not ready to run.** On 2026-11-21 the operator waived DIR-021 for this
section until the end-of-round capture mission attaches a screen capture from
Kirsch for each section. The steps were written from the code and the unit tests.
No screen has been captured yet, so the capture comes first.

**Removal condition:** remove this notice when that capture is attached for every
section.

The unit tests in `internal/tui` cover the logic behind the cues. No test starts
the interface in a terminal or reads a terminal's screen, so none captures the
running program. Elsewhere, `cmd/kirsch/main_test.go` builds the binary and runs
`--version` without a terminal. `TestMatchesScreenReference` in
`internal/tui/golden_test.go` renders screen 13, a markdown reply, from the Model's
`View()` in-process with colour off, and compares the text with the screen
reference. Other tests check the escape codes in rendered rows and in the `View()`
output, for example a keyword colour in a highlighted frame. None of them checks
how a terminal draws those colours.

**Context:** This section is run as part of the combined end-of-round test
(operator direction, 2026-11-20). It is not run on its own before the v0.1.4 tag.
Use the build and the test folder from the v0.1.2 section. `~/kirsch-ux-test`
must be a Git repository. Run `npm run build`. If Kirsch from section 11 of v0.1.3
is still running, press `Tab` to return to the composer, type `/quit` and press
`Enter`. Then start Kirsch in `~/kirsch-ux-test` with
`~/path/to/kirsch/bin/kirsch --new`. A section tagged *(model)* needs the model key
set up in the v0.1.2 section. The reply from section 1 stays in the transcript, and
later sections add replies below it. The *(model)* and *(no model)* tags on each
heading say whether a section sends a prompt. They do not say whether it uses an
earlier reply. `c` copies the selected reply, and `/copy` copies the last reply, so
`/copy` picks up a later reply, not the reply from section 1.

Use the markdown in the screen as the test. If a model's wording differs from the
prompt, judge the markdown features, not the words.

### 1. A markdown reply renders (model)

**Context:** A fresh session, with the composer empty and Kirsch idle. This section
ends in Composing, with the composer empty and the reply on screen.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > Reply with only this markdown: a heading `# Heading check`; a blank line; a
   > paragraph with **bold words**, `inline code` and ~~struck words~~; a blank
   > line; two bullets, `- first item` and `- second item`; a blank line; a go
   > code fence containing `func f() int { return 1 }`. Do not wrap the whole
   > reply in a code fence.

   You should see the reply drawn as markdown. The heading keeps its `#`. `bold
   words` is bold, with no `**`. `inline code` keeps its backticks. `struck words`
   is struck through, and its `~~` stays. The two items show `•` in place of `-`.
   The bullets assume a UTF-8 terminal; in an ASCII locale they show `-`.
   The code is indented two columns, its fence lines are not shown, and `func` and
   `return` are in the keyword colour.
   - [ ] The heading reads `# Heading check`.
   - [ ] `bold words` is bold, and no `**` shows.
   - [ ] `inline code` keeps its backticks.
   - [ ] `struck words` is struck through, and its `~~` shows.
   - [ ] The two items show `•` (or `-` in an ASCII locale).
   - [ ] The code has no fence lines, and `func` and `return` are in the keyword colour.

### 2. `render_markdown = false` shows the reply as plain text (no model)

**Context:** Section 1 is done and its reply is on screen, with Kirsch in Composing.
Type `/quit` and press `Enter` to quit Kirsch. Open the global config file,
`~/.config/kirsch/config.toml`, or `$XDG_CONFIG_HOME/kirsch/config.toml` when
`XDG_CONFIG_HOME` is set. Add these two lines, and save:

    [ui]
    render_markdown = false

Start Kirsch again from `~/kirsch-ux-test` with
`~/path/to/kirsch/bin/kirsch`, without `--new`. It resumes the session from
section 1.

1. Look at the reply from section 1.
   You should see the reply as plain text, as the model wrote it. `# Heading
   check` keeps its `#`, `**bold words**` keeps its asterisks, and `~~struck
   words~~` keeps its tildes. The two fence lines of the code block show. No
   keyword or heading colour shows, and no text is struck through or bold.
   - [ ] The `#` and the `**` markers show.
   - [ ] The `~~` markers show, and no text is struck through.
   - [ ] The two fence lines of the code block show.
   - [ ] No keyword or heading colour shows, and no text is bold.

This section ends with Kirsch in Composing. Type `/quit` and press `Enter`. Then
remove the two lines from the config file, so that later sections run with the
default. Section 3 starts Kirsch again.

### 3. The `read_file` modal highlights a Go file (model)

**Context:** Section 2 is done, and its config lines are removed. Restart Kirsch
with `~/path/to/kirsch/bin/kirsch` from `~/kirsch-ux-test`. Do not use `--new`, so
the section 1 reply stays in the transcript for later sections. The folder must
hold `hello.go`. If it does not, create it by hand with these lines:

    package main

    func main() {
    	println("hi")
    }

Then check that Git tracks `hello.go`: run `git ls-files hello.go` in
`~/kirsch-ux-test`. If it prints nothing, run `git add hello.go`, then
`git -c user.name=test -c user.email=test@example.com commit -m "add hello.go"`.
The commit needs a git identity, and the `-c` settings above supply one for that
commit only. A commit made with your own identity works too. Section 5 changes
`hello.go`, and `/diff` shows only what changed after this commit.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > Use read_file on hello.go, then say done.

2. Press `Shift+↑` to enter Browsing. Press `↑` until the `read_file` card is
   selected, then press `d`.
   You should see the content modal, with the file's lines numbered. The modal's
   own line numbers are at the left, and they stay dim. The tool's number for each
   line follows them. `package` and `func` are in the keyword colour, and `"hi"`
   is in the string colour.
   - [ ] `package` and `func` are in the keyword colour.
   - [ ] `"hi"` is in the string colour.
   - [ ] The modal's own line numbers are dim.
3. Press `Esc`.
   You should see the modal close and Browsing stay on.
   - [ ] `Esc` closes the modal.

This section ends in Browsing, with the `read_file` card selected and no modal open.

### 4. A patch approval diff is highlighted, and the markers keep their colour (model)

**Context:** Section 3 is done, and it ends in Browsing. Press `Tab` to return to
the composer, because step 1 types a prompt. Approvals are on, as the default is.
The session is still the one from section 3. This section ends in Composing, with
the turn ended and no approval card showing.

1. Type this prompt, press `Enter`, and wait:

   > Add a new file notes.go with package main and a function func Hello() string that returns "hi". Propose it with apply_patch and wait for my approval.

   You should see an approval card for the patch.
2. Press `d` on the approval card.
   You should see the diff modal. Each added line starts with a green `+`. `package`
   and `func` are in the keyword colour, and `"hi"` is in the string colour.
   - [ ] Each added line starts with a green `+`.
   - [ ] `package` and `func` are in the keyword colour.
   - [ ] `"hi"` is in the string colour.
3. Press `Esc` to close the diff, then press `n` to reject the patch.
   You should see the patch rejected, and no `notes.go` in the folder.
   - [ ] The patch is rejected, and `notes.go` is not created.
   If another approval card appears, press `n` on it too, and wait until Kirsch is
   idle.

### 5. `/diff` highlights each file in its own language (no model)

**Context:** Section 4 is done, and Kirsch is idle in Composing. Wait until the
turn from section 4 has ended and Kirsch is idle before you type `/diff`. Make
these changes by hand in another terminal, and do not run anything in Kirsch for
them. In `~/kirsch-ux-test`, add the line `func extra() {}` at the end of
`hello.go`, the file that section 3 made sure Git tracks. Create `tool.py` with the
line `def extra(): pass`, and run `git add -N tool.py` so that `/diff` lists it.
Then, in Kirsch, go on from section 4. This section ends in Composing, with `/diff`
closed.

1. Type `/diff` and press `Enter`.
   You should see the `hello.go` section. Its `+` marker is green, and `func` is in
   the keyword colour. Its `diff --git` header is not coloured by language.
   - [ ] The `+` on `func extra() {}` is green.
   - [ ] `func` is in the keyword colour.
   - [ ] The `diff --git` header line has no keyword colour.
2. Look at the `tool.py` section.
   You should see `def` in the keyword colour on `+def extra(): pass`, with a green
   `+`. The `hello.go` section keeps its Go colours.
   - [ ] `def` is in the keyword colour.
   - [ ] The `+` on the `tool.py` line is green.
3. Press `Esc` to close `/diff`.
   - [ ] `Esc` closes `/diff`.

### 6. `NO_COLOR=1` shows everything plain, with the same layout (no model)

**Context:** Section 5 is done, and Kirsch is in Composing. Type `/quit` and press
`Enter`. Start Kirsch again from `~/kirsch-ux-test` with
`NO_COLOR=1 ~/path/to/kirsch/bin/kirsch`. It resumes the last session. This section
ends in Composing, with `/diff` closed.

1. Type `/diff` and press `Enter`.
   You should see one file section for each changed file, as in section 5. The
   `hello.go` section has the line `+func extra() {}`, and the `tool.py` section has
   the line
   `+def extra(): pass`. Each `+` is plain text, with no colour, and so are `func`
   and `def`.
   - [ ] No line in `/diff` has any colour.
   - [ ] The `hello.go` section has the line `+func extra() {}`.
   - [ ] The `tool.py` section has the line `+def extra(): pass`.
2. Press `Esc` to close `/diff`.
   - [ ] `Esc` closes `/diff`.
3. Press `Shift+↑`, then `↑` until the reply from section 1 is selected. You should
   see its `#`, its backticks and its `~~`, and no `**`. Bold text is plain. Press
   `Tab` to return to the composer.
   - [ ] The reply from section 1 shows `#`, backticks and `~~`, and no `**`.

### 7. A `postgresql` fence is plain, and a `sql` fence is highlighted (model)

**Context:** Section 6 is done, and Kirsch is in Composing. Type `/quit` and press
`Enter`. Start Kirsch again with `~/path/to/kirsch/bin/kirsch` from
`~/kirsch-ux-test`, without `NO_COLOR`, so that colour is back. It resumes the
session, so the section 1 reply stays for section 8. This section ends in
Composing, with the composer empty.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > Reply with only two code fences. The first is tagged postgresql and the second is tagged sql. Each holds the line SELECT 1;

   You should see two code blocks. In the `postgresql` block, `SELECT` is plain
   text. In the `sql` block, `SELECT` is in the keyword colour.
   - [ ] `SELECT` in the `postgresql` block is plain.
   - [ ] `SELECT` in the `sql` block is in the keyword colour.

### 8. `/copy` still copies the source markdown (no model)

**Context:** Section 7 is done, and Kirsch is in Composing with the composer empty.
The reply from section 1 is still in the transcript, and so is the reply from
section 7, which is the last reply. Open a plain text app, such as TextEdit, before
each paste, and clear it before each paste. This section ends in Composing, with
the composer empty.

1. Press `Shift+↑` to enter Browsing. Press `↑` until the reply from section 1 is
   selected, then press `c`. Paste into the text app.
   You should see the source markdown: `# Heading check`, `**bold words**`,
   `` `inline code` ``, `~~struck words~~`, `- first item`, and the fence lines. You
   should not see `•`.
   - [ ] The paste keeps `# Heading check` with its `#`.
   - [ ] The paste keeps `**bold words**` with its asterisks.
   - [ ] The paste keeps the fence lines, and has no `•`.
2. Press `Tab` to return to the composer. Type `/copy` and press `Enter`. Paste into
   the text app. `/copy` takes the last reply, which is the reply from section 7,
   not the reply from section 1. You should see that reply's source: its two fence
   lines, `postgresql` and `sql`, with `SELECT 1;` in each block.
   - [ ] The `/copy` paste holds the section 7 reply's source, with its fence lines.
   - [ ] The `/copy` paste does not hold the section 1 reply's text.

---

## v0.1.5 · Scrollbar and long lines in the modal

**What the cues rest on.** Each *You should see* cue is backed by a screen capture in
`plan/testing/manual/captures/v0.1.5/`. The captures come from Kirsch's own render, with
canned inputs and no model. The file names, the hunk text, the version, the notices and
the footer in the captures are canned, so your screen will show different words. The cues
therefore describe structure only: where the bar and the thumb are, which column they are
in, which rows are blank, and where a marker or a number sits. They never describe text.
The index of the captures is `plan/testing/manual/captures/README.md`.

`modal-diff-wrapped` is cited for wrap structure only. Its hunk header is canned: it is not
the header git prints for the change in section 8, and its counts do not match the lines under it.

**Context:** This section is run as part of the combined end-of-round test. Use the build
and the test folder from the v0.1.2 section, `~/kirsch-ux-test`, and run `npm run build`
first. These commands set up the folder once, before section 1:

```bash
cd ~/kirsch-ux-test
{ printf 'x%.0s' $(seq 1 150); echo; seq 1 80; printf 'y%.0s' $(seq 1 150); echo; } > long.txt
git add long.txt && git commit -m "Test long.txt"
```

`long.txt` is one line 150 characters wide, then the lines that `seq 1 80` prints, then one
more line 150 characters wide. Skip the `git add` and `git commit` lines if `long.txt` is
already committed. If `git commit` asks for an identity, set one for this folder.

Sections use Unicode, which needs a UTF-8 locale, unless they say otherwise. Each section's
Context names the state it starts from, which is the state the section before it ends in.
Section 1 starts a new session.

The hint texts quoted in these sections are the Unicode forms. Section 10 of this group quotes the ASCII form.

**Reproduction.** The sections that need an overflowing transcript use the setup prompt
from the v0.1.2 section, `count from 1 to 300, one number per line`, which needs a model
key. The sections marked *(no model)* need that transcript on screen. They also run in any
later session that has it: start Kirsch without `--new` to resume the session. If no session
has one yet, run section 2 first.

### 1. A short transcript has no bar (model)

**Context:** Before you start Kirsch, set the window to 80 columns by 24 rows, by dragging a corner of the
window or using your terminal's window-size setting. Run `tput cols` and `tput lines` in that
window, and check that they print `80` and `24`. Then start Kirsch with
`~/path/to/kirsch/bin/kirsch --new` in `~/kirsch-ux-test`. Kirsch is idle in Composing, with
the composer empty.
This section ends in Composing, with the composer empty and a transcript that fits the
screen.

1. Type `read README.md and reply in one short sentence`, press `Enter`, and wait until
   Kirsch is idle.
   You should see a transcript shorter than the screen. The right-hand column is blank on
   every row, beside the header, the transcript, the status line and the composer.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-hidden-short.txt`, lines 9 to 27
   (`# steps:`, line 5, reads `(canned short transcript loads) | (a read_file card finishes)`).
   Lines 11 to 20 are the short transcript and no line has a glyph in its last column.
   - [ ] No bar is drawn beside the transcript.

### 2. The bar at the bottom of an overflowing transcript (model)

**Context:** Section 1 ends in Composing with the composer empty. The window is 80 by 24.
Unicode. This section ends in Composing, pinned to the bottom, with the composer empty.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > count from 1 to 300, one number per line

   You should see a bar in the right-hand column, running from the first transcript row
   down to the last one, above the rule that separates the transcript from the status line.
   A thumb, drawn `┃`, sits at the bottom of the bar, on the last transcript row. The track
   above it is drawn `│`. The length of the thumb depends on how long the transcript is,
   so match where it sits, not how long it is.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-bottom.txt`, lines 11 to 27
   (`# steps:`, line 5, reads `(canned transcript loads) | (canned second turn loads) | (a read_file card finishes)`).
   Lines 23 to 27 hold the thumb, at the bottom of the bar, and line 28 is the rule, with no bar.
   - [ ] The bar runs from the first transcript row to the last one.
   - [ ] The thumb sits at the bottom of the bar.

### 3. The bar at the top (no model)

**Context:** Section 2 ends in Composing, pinned to the bottom, with the composer empty.
The window is 80 by 24. Unicode. This section ends in Browsing, with the view at the top of
the transcript and no modal open.

1. Press `Shift+↑` to enter Browsing. Then press `g`, which moves the view to the top
   whichever item is selected.
   You should see the thumb at the top of the bar, starting on the first transcript row,
   with the track below it down to the last transcript row.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-top.txt`: the thumb is on
   lines 11 to 15 and the track on lines 16 to 27. `# steps:`, line 5, reads
   `(canned transcript loads) | (canned second turn loads) | (a read_file card finishes) | Shift+↑ | g`.
   The capture's steps match the key presses here, so this section checks the screen only.
   - [ ] The thumb is at the top of the bar.
   - [ ] The track runs below the thumb to the last transcript row.

### 4. The bar in the middle, with `↓ N new` (model)

**Context:** Section 3 ends in Browsing at the top of the transcript, with no modal open.
The window is 80 by 24. Unicode. The transcript still
overflows. This section ends in Browsing, scrolled up and not pinned, with a `↓ N new`
indicator on the last transcript row.

1. Press `Tab` to return to the composer. Type `read README.md, then count from 1 to 300, one number per line`
   and press `Enter`. As soon as the first tool card shows, press `Shift+↑`, then `g`, then
   `PgDn`. Wait until the reply has finished.
   `Shift+↑` returns to Browsing. `g` moves the view to the top, and `PgDn` moves it down
   one page. The view is then not pinned, so the blocks that arrive below it are counted in
   `↓ N new`. `N` counts those blocks, not lines.
   You should see the thumb on a row above the bottom of the bar, with track below it. On the
   last transcript row, the `↓ N new` indicator ends flush against the bar's column.
   If no tool card appears, write `no tool call` beside this step and send the prompt again.
   If no indicator appears, write `reply finished first` beside this step and try again with
   a longer prompt.
   The capture's own steps are `Shift+↑`, `g`, `PgDn`, the same as here. Its thumb sits where
   its canned transcript length puts it, so this section checks the thumb as a rule, not
   against the capture's rows.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-mid-new.txt` for the indicator's
   place: line 27 holds `↓ 3 new` on the last transcript row, with the bar at its right.
   `# steps:`, line 5, reads
   `(canned transcript loads) | (canned second turn loads) | (a read_file card finishes) | Shift+↑ | g | PgDn | (three notices arrive)`.
   The capture's number and its notices are canned.
   - [ ] The thumb is not on the bottom row of the bar, and track shows below it.
   - [ ] The `↓ N new` indicator ends on the last transcript row, flush against the bar.

### 5. No bar under an open modal (model)

**Context:** Section 4 ends in Browsing, scrolled up and not pinned. The window is 80 by 24.
Unicode. This section ends in Browsing, with no modal open.

1. Press `↑` or `↓` until a `read_file` card for `README.md` shows the `┃` gutter and the
   hint row begins `↑↓ select`. For example, the card from section 1 or the card from section 4
   will do. The cues below hold whichever `README.md` card is opened, so do not try to tell
   them apart. If the gutter goes off screen, press the other arrow. If the hint row changes
   to `↑↓ history · ⇧↑/Tab cards · /help`, which is the composer's hint and not the Browsing
   one, you have stepped past the last item: press `Shift+↑` to return to the cards and carry
   on. Then press `d`.
   If a `d` reaches the composer by mistake, press `Esc` to clear it (no turn is running here,
   and `Esc` clears the composer in Composing when none is). You are still in the composer:
   press `Shift+↑`, bring the gutter back to the card with the arrows, and press `d` again.
   You should see a box over the transcript, with a title in the form `read_file <file>`. No
   bar is drawn in the right-hand column on any row. Fragments of the transcript behind the
   box show at the box's right edge.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-hidden-modal.txt`, lines 11 to 26.
   `scrollbar-hidden-modal.annot.txt`, line 10, states that no row of the frame has a cell in
   column 79. `# steps:`, line 5, reads `(canned transcript loads) | (canned second turn loads) | (a read_file card finishes) | Shift+↑ | d`. The capture's steps do not include the arrow presses.
   - [ ] No bar is drawn while the modal is open.
2. Press `Esc`.
   You should see the screen return to Browsing, with the bar back in the right-hand column.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-mid-new.txt`, lines 16 to 20, for
   the bar's structure only.
   - [ ] The bar comes back when the modal closes.

### 6. A wrapped line in a `read_file` modal (model)

**Context:** Section 5 ends in Browsing, with no modal open. The window is 80 by 24. Unicode.
This section ends with the `read_file` modal for `long.txt` open at its top.

1. Press `Tab`. Type `read long.txt and reply with one word`, press `Enter`, and wait until
   Kirsch is idle. Press `Shift+↑`, which enters Browsing from the composer. Then press `↑` or
   `↓` until the `read_file` card for `long.txt` shows the `┃` gutter and the hint row begins
   `↑↓ select`. If the gutter goes off screen, press the other arrow. If the hint row changes
   to `↑↓ history · ⇧↑/Tab cards · /help`, which is the composer's hint and not the Browsing
   one, you have stepped past the last item: press `Shift+↑` to return to the cards and carry
   on. Then press `d`.
   If a `d` reaches the composer by mistake, press `Esc` to clear it (no turn is running here,
   and `Esc` clears the composer in Composing when none is). You are still in the composer:
   press `Shift+↑`, bring the gutter back to the `long.txt` card with the arrows, and press
   `d` again.
   You should see a box whose title is `read_file long.txt`. Its first line is 150 characters wide, so it
   runs over several rows inside the box. The line number shows on the first row only. The
   rows below it leave the number columns blank, and their text starts in the column where
   the tool's own number sits on the first row. The modal's own number and the tool's number
   both show on that first row. That is how the capture draws it, and it is described here
   as seen, not as designed.
   The rule and the footer are drawn inside the box, as before: the footer sits on the row
   above the box's bottom edge.
   Backed by `plan/testing/manual/captures/v0.1.5/modal-readfile-wrapped.txt`. The box is lines 11
   to 21, and the wrapped line is lines 14 to 16. `modal-readfile-wrapped.annot.txt`, rows 05
   to 07, shows the first row's text at column 20 and the continuation at column 16. Lines 13
   to 19 show fragments of the card behind the box at the right edge, and lines 12 to 19 show
   fragments at the left edge. `# steps:`, line 5, reads `(a read_file card finishes) | Shift+↑ | d`. The capture's steps
   are a subset of the key presses here, so this section checks the screen only.
   - [ ] The wrapped line runs over several rows, and the line number shows on the first row only.
   - [ ] The modal's number and the tool's number both show on the first row.
   - [ ] The footer `j/k scroll · g/G top/bottom · Esc close` is drawn inside the box, above its bottom edge.

   Pieces of the transcript behind the box show at its left and right edges, as the capture shows.

### 7. `G` lands on the last full page (model)

**Context:** Section 6 ends with the `read_file` modal for `long.txt` open at its top. The
window is 80 by 24. Unicode. This section ends in Browsing, with the modal closed.

1. Press `G`.
   You should see the body filled down to the last line of `long.txt`, with no blank row under
   it. The rule above the footer shows `↑ N above`, where `N` counts the display rows above the
   top of the body. It is a count of rows, not a line number.
   Backed by `plan/testing/manual/captures/v0.1.5/modal-after-G.txt`: the body is lines 12 to 24,
   and its last row, line 24, is the last row of the file's last line. Line 25 is the rule with `↑ 47 above`.
   `# steps:`, line 5, reads `(a read_file card finishes) | Shift+↑ | d | G`.
   - [ ] The last body row is the last row of the file's last line, which may be a wrapped row. No blank row sits under it.
   - [ ] The rule shows `↑` with a count of rows above.
2. Press `Esc` to close the modal.
   You should see Browsing again. The hint line reads
   `↑↓ select · Enter preview · d detail · Tab/Esc prompt` while the `┃` gutter is on a card,
   and `↑↓ select · c copy · Tab/Esc prompt` while it is on a reply.
   The hint is drawn by `hintRow` in `internal/tui/view.go`, which names `c copy` in place of
   `Enter preview` and `d detail` only on a reply.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-mid-new.txt`, line 32, which shows
   the card hint text.

### 8. A wrapped `+` line and a wrapped `@@` line in `/diff` (no model)

**Context:** Section 7 ends in Browsing with no modal open. Before this section, make the
window 80 columns by 32 rows, the size of the diff capture. Check it with `tput cols` and
`tput lines`. Then make the unstaged change this section diffs, in the test folder:

```bash
cd ~/kirsch-ux-test
{ printf 'x%.0s' $(seq 1 150); echo; seq 1 80; printf 'z%.0s' $(seq 1 150); echo; } > long.txt
```

The last line of `long.txt` is now 150 `z` characters, where the setup made it 150 `y`
characters. `/diff` lists every unstaged change in the folder, not only `long.txt`. For example,
the `hello.go` change and the `tool.py` file that section 5 of v0.1.4 leaves in this folder
may appear as well. This section checks only the `long.txt` hunk. Git puts the
start of `long.txt`'s first line in that hunk's header as its function context, 80 characters
of it, so the header is wider than the text. This section ends in Composing, with the composer
empty and no modal open.

1. Press `Tab` to return to the composer. Type `/diff` and press `Enter`.
   You should see a box titled `working tree`, with a stat on its top rule.
2. Press `j` or `PgDn` until the `long.txt` hunk is in view. The diff for `hello.go` can sit
   above it, so scroll past that part first.
   You should see a code line wider than the text wrap. Its `+` or `-` marker shows on the
   first row only. The row below it starts with a blank column where the marker is, and the
   code continues from there. A `@@` line wraps flush, with no blank column. A short `@@` line
   does not wrap, and that is not a failure: it wraps only when it is wider than the text.
   Backed by `plan/testing/manual/captures/v0.1.5/modal-diff-wrapped.txt` for wrap structure only.
   The `+` line is lines 21 to 24, the `-` line lines 25 to 28, and the wrapped `@@` line lines 16 to
   19. `modal-diff-wrapped.annot.txt` shows the marker at column 11 on the first row, and the
   continuation rows starting at column 12 with column 11 blank. `# steps:`, line 5, reads
   `/diff | Enter | (the git diff result arrives)`. The capture is 80 columns by 32 rows. The
   capture's diff holds `calc/divide.go` and nothing else, so it has no `hello.go` part to scroll past.
   - [ ] A code line wider than the text shows its `+` or `-` on the first row only.
   - [ ] The row below a wrapped code line starts with a blank column where the marker is.
   - [ ] A wrapped `@@` line starts flush with the left edge of the text.
3. Press `Esc`.
   You should see the modal close, and Kirsch return to the composer, because `/diff` was
   typed there. The hint line under the composer shows the composer keys.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-bottom.txt`, lines 31 and 32 (the
   composer and its hint).

### 9. Help lines are cut, not wrapped, at 60×24 (no model)

**Context:** Section 8 ends in Composing, with the composer empty, at 80 by 32. Before this
section, make the window 60 columns by 24 rows, and check it with `tput cols` and `tput lines`.
This section ends in Browsing, with the help overlay closed, at 60 by 24.

1. Press `Shift+↑`, then `?`.
   You should see a box titled `help`, with two columns. Each binding is one row of the screen.
   A row longer than the box's text width ends in `⋯` at the right edge. It does not wrap onto
   the next row. The footer rule shows a `↓ N more` marker, where `N` counts the help rows below
   the ones showing.
   Backed by `plan/testing/manual/captures/v0.1.5/help-cut.txt`, lines 11 to 27 (the box), line 24
   (a cut row, `PgUp/PgDn j/k scroll` with `/help  /st⋯`), and line 25 (the footer rule with
   `↓ 4 more`). `# steps:`, line 5, reads `Shift+↑ | ?`. The file's header, line 3, gives
   the size as `60x24`.
   - [ ] Each help row is one row of the screen, and a long one ends in `⋯`.
   - [ ] The footer rule shows a `↓ N more` marker.
2. Press `Esc` to close the help box. It is not checked by a capture; section 10 starts from
   Browsing, with the help box closed.

### 10. The bar with ASCII glyphs (no model)

**Context:** Section 9 ends in Browsing, with the help box closed, at 60 by 24. Set the window
to 80 columns by 24 rows. Press `Tab`, type `/quit` and press `Enter`. Then start Kirsch from
`~/kirsch-ux-test` with the locale set to C: `LC_ALL=C ~/path/to/kirsch/bin/kirsch`. Leave out
`--new`, so the session resumes with its long transcript. This section ends in Composing,
pinned to the bottom, with the composer empty and the glyph set ASCII.

1. Wait until the transcript shows.
   You should see a bar in the right-hand column, its track drawn `|` and its thumb `#`, with
   the thumb at the bottom of the bar. The status line and the hint use `.` in place of the
   Unicode separators, and the composer caret is `_`. The frame is not pure ASCII. The footer that
   `renderTool` draws for a tool card keeps its own Unicode glyphs, for example `‹` and `›`.
   Backed by `plan/testing/manual/captures/v0.1.5/scrollbar-ascii.txt`. The bar is lines 11 to 27,
   with `|` on lines 11 to 22 and `#` on lines 23 to 27. The composer caret is on line 31, and the
   hint, `Up/Down history . Shift+Up/Tab cards . /help`, is on line 32. `# glyphs:`, line 4, reads
   `ascii`, and `# steps:`, line 5, reads `(canned transcript loads) | (canned second turn loads) | (a read_file card finishes)`.
   - [ ] The track is `|` and the thumb is `#`, in the right-hand column.
   - [ ] The status and hint lines use `.` for the separators, and the composer caret is `_`.

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

## Known gaps after v0.1.4

These are not checked by the v0.1.4 sections above. They are listed here so the
operator knows they are open. The full list is in `plan/spec/kirsch-plan.md`,
amendment 102.

- A numbered item takes the number after the item before it, not the number written
  in its source, when the text before its first content does not end in digits,
  then `.` or `)`, then spaces. That text is read from the source line on which the
  first content starts. The first item takes the list's start number. For example,
  the first content is a heading, a block quote, a fenced code block, a nested list,
  or nothing at all, or the content starts on the line after an empty marker. A
  thematic break as the first content falls under the same rule.
- Some names that are not keyword constants draw as keywords. In some lexers, for
  example capitalised names in Ruby and all-caps names in Rust, Perl and Crystal,
  the lexer tags such names as constants.
- With colour off, bold and italic text shows as plain text.
- A line wider than 512 columns is drawn without syntax colours, so a long minified
  line has none.
- One slow moment, such as a GC pause or heavy load, during a single line can turn
  highlighting off for that language for the rest of the session. Nothing resets
  it.
- A line abandoned at the 50 ms deadline keeps one background goroutine busy until
  chroma's own per-rule timeouts end it, at most once per language.
- The excluded lexers are Raku, the PostgreSQL and Materialize SQL dialects and
  Svelte, the keys of `hlUnsafeLexers` in `internal/tui/highlight.go`. They are
  always drawn without syntax colours. Every name, alias and file pattern that chroma
  maps to them is excluded too, for example the fences `postgresql`, `raku` and
  `svelte`, and the files `.raku` and `.svelte`.
- The binary is about 5.7 MB larger, from about 15.1 MB to about 20.8 MB. Warm
  `kirsch --version` takes a median of about 5.5 ms before the change and about
  10.5 ms after it. These are rough figures. Source: measured during
  mission-20261120-03; raw values in
  `.claude/memory/missions/mission-20261120-03/DEBRIEF.md`. The times are `time`
  readings on the build machine, with the first (cold) run of each set dropped. The
  warm runs span 0.004–0.010 s before the change and 0.009–0.014 s after it.

## Known gaps after v0.1.5

This block lists no gaps. The known gaps of v0.1.5 are in `plan/spec/kirsch-plan.md`,
amendment 103, under *Known gaps (v0.1.5)*. The sections above do not check them.
