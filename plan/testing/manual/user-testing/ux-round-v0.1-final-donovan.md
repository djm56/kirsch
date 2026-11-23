# UX round v0.1: combined test, runbook

This is the plain-language copy of the combined end-of-round test. It covers v0.1.3, v0.1.4 and v0.1.5. Each heading names the master section it follows, so you can look it up in `plan/testing/manual/ux-round-v0.1.md` if a step is unclear.

## What this checks

The copy and select changes (v0.1.3), the Markdown and colour changes (v0.1.4), and the scrollbar and long-line changes (v0.1.5), as they look on your screen in Kirsch.

Each step says what to do, what you should see, and a box to tick. Tick a box only when the screen shows what the step says. All boxes start empty. Only you tick them.

## Setup

Do this once, before v0.1.3 section 1.

**Terminal.** Use a Terminal window, not a panel inside an editor. Some sections use Unicode characters, so the Terminal must use UTF-8, unless a step says otherwise.

**Build.** From the top folder of Kirsch, run:

```bash
npm run build
```

It writes `bin/kirsch`. In the commands below, replace `~/path/to/kirsch` with the folder where you cloned Kirsch.

**Test folder.** Run these in a Terminal:

```bash
mkdir -p ~/kirsch-ux-test && cd ~/kirsch-ux-test && git init
printf 'A test folder for Kirsch.\n' > README.md
```

The folder must be a Git folder, so Kirsch starts inside it. The second line creates `README.md`, which some sections read.

**Long file (for v0.1.5).** Run these in `~/kirsch-ux-test`, once:

```bash
cd ~/kirsch-ux-test
{ printf 'x%.0s' $(seq 1 150); echo; seq 1 80; printf 'y%.0s' $(seq 1 150); echo; } > long.txt
git add long.txt && git commit -m "Test long.txt"
```

`long.txt` is one line 150 characters wide, then the lines that `seq 1 80` prints, then one more line 150 characters wide. Skip the `git add` and `git commit` lines if `long.txt` is already committed. If `git commit` asks for an identity, set one for this folder.

**Model key.** Set the model key you normally use. Sections marked *(model)* need it. A step that types a prompt sends that prompt to the model, which costs tokens.

**Window sizes.** Sections that need a set size name it in their **Before you start** line. The key-hint row at the bottom of the screen needs at least 24 rows, so keep the window at 24 rows or more. Check the size with `tput cols` and `tput lines` in a second Terminal tab.

**Starting Kirsch.** Start it from `~/kirsch-ux-test` with `~/path/to/kirsch/bin/kirsch`. Where a section says `--new`, add it. `--new` starts a fresh session, so the transcript starts empty. Without it, Kirsch resumes the last session.

---

## v0.1.3 · Copy and select replies

**Before you start:** Start a fresh session. In `~/kirsch-ux-test`, run `~/path/to/kirsch/bin/kirsch --new`. Then type these prompts, one after the other. Press `Enter` after each, and wait for Kirsch to be idle in the composer before you type the next:

> say hello in one sentence

> say goodbye in one sentence

The transcript now holds your first message, its reply, your second message and its reply, in that order. A start-up notice, such as `no previous session — starting fresh`, may sit above them. A notice is a selectable item too. The steps below assume the transcript starts with your first message. If a notice sits above it, write that beside section 1, step 3, because that is the step where the `↑` press reaches the notice.

The `┃` mark is the accent bar on the left of a selected item. In an ASCII locale it is `|`.

### v0.1.3 · master section 1: Arrows stop on replies and skip your messages (no model)

**Before you start:** Setup is done. The composer is empty, Kirsch is idle, and nothing is selected. This section ends in Composing, with the `┃` on the second reply.

1. Press `Shift+↑`. The focus moves to the transcript, and the last reply is selected.
   You should see the `┃` beside the second reply, on every row of that reply.
   - [ ] The `┃` marks the second reply, on each of its rows.
2. Press `↑`.
   You should see the `┃` move up to the first reply. Your second message sits between the two replies, and it gets no `┃`.
   - [ ] One `↑` moves the `┃` from the second reply to the first.
   - [ ] Your second message has no `┃` when the selection passes it.
3. Press `↑` again.
   You should see the `┃` stay on the first reply. Nothing lies above it. If a start-up notice sits above your first message, this press selects the notice instead. Write `notice above` beside this step, and leave the boxes below empty.
   - [ ] The `┃` stays on the first reply.
   - [ ] Your first message gets no `┃`.
4. Press `↓` twice. The first press moves the `┃` back to the second reply. The second press takes the focus back to the composer.
   - [ ] The second `↓` returns the cursor to the composer.
   - [ ] The `┃` stays on the second reply after the focus returns to the composer.

### v0.1.3 · master section 2: `Enter` and `d` do nothing on a reply (no model)

**Before you start:** Composing, with the composer empty and the `┃` on the second reply, as section 1 leaves it. This section ends in Browsing, with the second reply selected.

1. Press `Shift+↑` to enter Browsing.
   - [ ] The `┃` is still on the second reply.
2. Press `Enter`. You should see none of these happen:
   - [ ] `Enter` does not expand or collapse the reply.
   - [ ] `Enter` opens no box.
   - [ ] `Enter` does not move the view.
3. Press `d`. You should see no box open, and the screen stays as it was.
   - [ ] `d` opens no box for the reply.
4. Read the bottom row of the screen, the key-hint row. You should see `↑↓ select · c copy · Tab/Esc prompt`.
   - [ ] The key-hint row reads `↑↓ select · c copy · Tab/Esc prompt`.

### v0.1.3 · master section 3: Copy a reply with `c`, then paste it into another app (no model)

**Before you start:** Browsing, with the second reply selected, as section 2 leaves it. Open a plain text window in another app, such as TextEdit. Before each paste, clear the text window. This section ends in Browsing, with the second reply selected.

1. Press `c`. You should see the hint under the composer read `copied N characters`, with `N` a plain number. Write `N` beside this step.
   If the hint reads `sent N characters (OSC 52)` or `copy failed: no clipboard route`, write that beside this step and leave the box empty.
   - [ ] The hint under the composer reads `copied N characters`.
2. Switch to the text window. Clear it, then paste. You should see the text of the second reply, with the same words and punctuation.
   - [ ] The paste holds the second reply's text.
3. Switch back to Kirsch. You should see the transcript where it was.
   - [ ] The view did not move.
   - [ ] The `┃` is still on the second reply.

### v0.1.3 · master section 4: `c` on a card gives the hint, and copies nothing (no model)

**Before you start:** Browsing, with the second reply selected, as section 3 leaves it. This section makes a notice card and selects it. This section ends in Browsing, with the notice selected.

1. Press `Tab` to return to the composer. Type `/status` and press `Enter`. You should see a notice card appear at the bottom of the transcript.
   - [ ] The status notice appears in the transcript.
2. Press `Shift+↑` to enter Browsing, then press `↓`. The `┃` moves from the second reply to the notice card.
   - [ ] The `┃` marks the status notice.
3. Press `c`. You should see the hint `c copies replies only` under the composer.
   - [ ] The hint reads `c copies replies only`.
   - [ ] Nothing is copied.

### v0.1.3 · master section 5: `/copy` copies the last reply (no model)

**Before you start:** Browsing, with the status notice selected, as section 4 leaves it. The last reply in the transcript is the second reply. The status notice is not a reply, so `/copy` does not use it. This section ends in Composing, with the composer empty. The copy notice may still show; typing clears it.

1. Press `Tab` to return to the composer. Type `/copy` and press `Enter`. You should see the hint under the composer read `copied N characters`. The transcript does not move.
   - [ ] The hint under the composer reads `copied N characters`.
   - [ ] The transcript stays where it was.
2. Clear the text window, switch to it, and paste. You should see the second reply's text, the same as in section 3.
   - [ ] The paste holds the second reply's text.

### v0.1.3 · master section 6: `/copy code` and `/copy code 2` on a reply with two code blocks (model)

**Before you start:** Composing, with the composer empty, as section 5 leaves it. This section sends a prompt, so it uses the model. Type this prompt, and press `Enter`:

> Write two short Go functions. Put each in its own fenced code block with the language tag go. Put one sentence of plain text between the two blocks.

Count the lines that open a fence, which look like ```` ```go ````, in the reply. Models vary, so the count may differ. The steps below assume two, which is the number the prompt asks for. If the count is not two, write the count beside step 1, and do not tick the boxes that depend on it. Wait until the reply has finished and Kirsch is idle in the composer, with no turn running, before step 1. This section ends in Composing, with the composer empty.

1. Type `/copy code` and press `Enter`. You should see the hint under the composer read `copied N characters`. Clear the text window, switch to it, and paste. You should see the lines of the second code block only, with no fence line and no `go`.
   - [ ] The hint under the composer reads `copied N characters`.
   - [ ] The paste holds the second block's lines.
   - [ ] The paste has no fence line.
   - [ ] The paste has no `go`.
2. Type `/copy code 2` and press `Enter`. Clear the text window, switch to it, and paste. Block 2 is the last block, so you should see the same text as step 1.
   - [ ] `/copy code 2` pastes the same text as `/copy code`.
3. Type `/copy code 1` and press `Enter`. Clear the text window, switch to it, and paste. You should see the lines of the first code block, and not the second.
   - [ ] The paste holds the first block's lines.
   - [ ] The paste does not hold the second block's lines.

### v0.1.3 · master section 7: `/copy code 9` gives the range notice (no model)

**Before you start:** Composing, with the composer empty, as section 6 leaves it. Kirsch is idle, and the reply has finished. The reply with two code blocks is still the last reply. The clipboard holds the first block's lines from section 6, step 3. This section ends in Composing, with the composer empty.

1. Type `/copy code 9` and press `Enter`. You should see the hint under the composer read `the last reply has 2 code blocks`.
   - [ ] The hint under the composer reads `the last reply has 2 code blocks`.
2. Clear the text window, switch to it, and paste. You should see the first block's lines again, because the out-of-range request copied nothing.
   - [ ] The paste still holds the first block's lines.

### v0.1.3 · master section 8: `/copy foo` gives the usage hint (no model)

**Before you start:** Composing, with the composer empty, as section 7 leaves it. This section ends in Composing, with the composer empty and the usage hint showing.

1. Type `/copy foo` and press `Enter`. You should see the hint under the composer read `usage: /copy [code [N]]`.
   - [ ] The hint under the composer reads `usage: /copy [code [N]]`.
   - [ ] No new item appears in the transcript.

### v0.1.3 · master section 9: The copy notice clears when you type (no model)

**Before you start:** Composing, with the composer empty, as section 8 leaves it. The usage hint is showing. This section ends in Composing, with the composer empty.

1. Type `/copy` and press `Enter`. You should see the hint under the composer read `copied N characters`.
   - [ ] The hint under the composer reads `copied N characters`.
2. Type the letter `x`. You should see the copy notice disappear, and the `x` appear in the composer.
   - [ ] One keypress removes the copy notice.
   - [ ] The `x` appears in the composer.
3. Press `Esc` to clear the composer.
   - [ ] The composer is empty.

### v0.1.3 · master section 10: `/c` and `/co` do not complete, and `/cop` does (no model)

**Before you start:** Composing, with the composer empty, as section 9 leaves it. This section ends in Composing, with the composer empty.

1. Type `/c` and press `Tab`. The composer still reads `/c`.
   - [ ] `/c` does not complete.
   Type the letter `x`. The composer reads `/cx`, so the focus is still in the composer. Press `Esc` to clear it.
   - [ ] After `Tab`, the focus is still in the composer.
   - [ ] The `x` lands in the composer.
2. Type `/co` and press `Tab`. The composer still reads `/co`.
   - [ ] `/co` does not complete.
   Press `Esc` to clear the composer.
3. Type `/cop` and press `Tab`. You should see the composer read `/copy`.
   - [ ] `/cop` completes to `/copy`.
   Press `Esc` to clear the composer.
   - [ ] The composer is empty.

### v0.1.3 · master section 11: The help overlay shows the copy rows, and `?` closes it (no model)

**Before you start:** Composing, with the composer empty and no notice showing, as section 10 leaves it. Section 10 ends with `Esc`, which clears the composer and the hint under it. The status notice card from section 4 is still the selected item. Kirsch is idle. The window is 80 columns by 28 rows. This section ends in Browsing, with the help box closed and the hint under the composer reading `c copies replies only`, as step 5 leaves it.

1. Press `Shift+↑` to enter Browsing, then press `?`. You should see a box titled `help`, with its footer `Esc or ? closes` at the bottom.
   - [ ] The help box opens.
   - [ ] Its footer reads `Esc or ? closes`.
2. Read the left column of the help box. With the composer empty and no notice showing, the whole box shows on the first view at 80 by 28, with the footer `kirsch v<version> · Esc or ? closes`, where `<version>` is the version of the build you are testing. Until the v0.1.5 tag is made, a build from this tree prints `kirsch v0.1.3-dev`. Under `browsing`, you should see `↑/↓` with `select card/reply`, and `c` with `copy reply`. There is no `?` row. If a notice is showing under the composer, the box is one row short, and `G`, which moves to the bottom of the help box, shows the last rows.
   - [ ] The first view shows the whole box, from its top border to its bottom border.
   - [ ] The first view has no `↓ N more` marker.
   - [ ] The browsing block shows `↑/↓` with `select card/reply`.
   - [ ] The browsing block shows `c` with `copy reply`.
   - [ ] The browsing block has no `?` row.
3. Read the right column. Its last row in the commands list should read `/exit  /copy`.
   - [ ] The last commands row reads `/exit  /copy`.
4. Press `?`. The help box closes, and you stay in Browsing.
   - [ ] `?` closes the help box.
5. Press `c`. The card is still selected. You should see the hint under the composer read `c copies replies only`.
   - [ ] The hint under the composer reads `c copies replies only`.
   If nothing appears, write `no hint after help` beside this box. Do not tick it.

If a notice is showing and a row is still cut off after `G`, write that beside step 2. Do not tick it.

---

## v0.1.4 · Markdown replies and code highlighting

**Before you start:** If Kirsch from v0.1.3 section 11 is still running, press `Tab` to return to the composer, type `/quit` and press `Enter`. Then start Kirsch in `~/kirsch-ux-test` with `~/path/to/kirsch/bin/kirsch --new`. Sections tagged (model) need the model key from setup. The reply from section 1 stays in the transcript, and later sections add replies below it.

The (model) and (no model) tags on each heading say whether a section types a prompt. They do not say whether it uses an earlier reply. `c` copies the selected reply, and `/copy` copies the last reply, so `/copy` picks up a later reply, not the reply from section 1.

Use the markdown on the screen as the test. If a model's wording differs from the prompt, judge the markdown features, not the words.

### v0.1.4 · master section 1: A markdown reply renders (model)

**Before you start:** A fresh session, with the composer empty and Kirsch idle. This section ends in Composing, with the composer empty and the reply on screen.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > Reply with only this markdown: a heading `# Heading check`; a blank line; a paragraph with **bold words**, `inline code` and ~~struck words~~; a blank line; two bullets, `- first item` and `- second item`; a blank line; a go code fence containing `func f() int { return 1 }`. Do not wrap the whole reply in a code fence.

   You should see the reply drawn as markdown. The heading keeps its `#`. `bold words` is bold, with no `**`. `inline code` keeps its backticks. `struck words` is struck through, and its `~~` stays. The two items show `•` in place of `-`. The bullets assume a UTF-8 terminal; in an ASCII locale they show `-`. The code is indented two columns, its fence lines are not shown, and `func` and `return` are in the keyword colour.
   - [ ] The heading reads `# Heading check`.
   - [ ] `bold words` is bold, and no `**` shows.
   - [ ] `inline code` keeps its backticks.
   - [ ] `struck words` is struck through, and its `~~` shows.
   - [ ] The two items show `•` (or `-` in an ASCII locale).
   - [ ] The code has no fence lines, and `func` and `return` are in the keyword colour.

### v0.1.4 · master section 2: `render_markdown = false` shows the reply as plain text (no model)

**Before you start:** Section 1 is done and its reply is on screen, with Kirsch in Composing. Type `/quit` and press `Enter` to quit Kirsch. Open the global config file, `~/.config/kirsch/config.toml`, or `$XDG_CONFIG_HOME/kirsch/config.toml` when `XDG_CONFIG_HOME` is set. Add these lines, and save:

```toml
[ui]
render_markdown = false
```

Start Kirsch again from `~/kirsch-ux-test` with `~/path/to/kirsch/bin/kirsch`, without `--new`. It resumes the session from section 1.

1. Look at the reply from section 1.
   You should see the reply as plain text, as the model wrote it. `# Heading check` keeps its `#`, `**bold words**` keeps its asterisks, and `~~struck words~~` keeps its tildes. The two fence lines of the code block show. No keyword or heading colour shows, and no text is struck through or bold.
   - [ ] The `#` and the `**` markers show.
   - [ ] The `~~` markers show, and no text is struck through.
   - [ ] The two fence lines of the code block show.
   - [ ] No keyword or heading colour shows, and no text is bold.

This section ends with Kirsch in Composing. Type `/quit` and press `Enter`. Then remove those lines from the config file, so that later sections run with the default. Section 3 starts Kirsch again.

### v0.1.4 · master section 3: The `read_file` modal highlights a Go file (model)

**Before you start:** Section 2 is done, and its config lines are removed. Restart Kirsch with `~/path/to/kirsch/bin/kirsch` from `~/kirsch-ux-test`. Do not use `--new`, so the section 1 reply stays in the transcript for later sections. The folder must hold `hello.go`. If it does not, create it by hand with these lines:

```go
package main

func main() {
	println("hi")
}
```

Then check that Git tracks `hello.go`: run `git ls-files hello.go` in `~/kirsch-ux-test`. If it prints nothing, run `git add hello.go`, then `git -c user.name=test -c user.email=test@example.com commit -m "add hello.go"`. The commit needs a git identity, and the `-c` settings above supply one for that commit only. A commit made with your own identity works too. Section 5 changes `hello.go`, and `/diff` shows only what changed after this commit.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > Use read_file on hello.go, then say done.

2. Press `Shift+↑` to enter Browsing. The `read_file` card is still selected, so press `d`.
   You should see the content modal, with the file's lines numbered. The modal's own line numbers are at the left, and they stay dim. The tool's number for each line follows them. `package` and `func` are in the keyword colour, and `"hi"` is in the string colour.
   - [ ] `package` and `func` are in the keyword colour.
   - [ ] `"hi"` is in the string colour.
   - [ ] The modal's own line numbers are dim.
3. Press `Esc`.
   You should see the modal close and Browsing stay on.
   - [ ] `Esc` closes the modal.

This section ends in Browsing, with the `read_file` card selected and no modal open.

### v0.1.4 · master section 4: A patch approval diff is highlighted, and the markers keep their colour (model)

**Before you start:** Section 3 is done, and it ends in Browsing. Press `Tab` to return to the composer, because step 1 types a prompt. Approvals are on, as the default is. The session is still the one from section 3. This section ends in Composing, with the turn ended and no approval card showing.

1. Type this prompt, press `Enter`, and wait:

   > Add a new file notes.go with package main and a function func Hello() string that returns "hi". Propose it with apply_patch and wait for my approval.

   You should see an approval card for the patch, with `[y] approve`, `[n] reject` and `[d] view diff` in it.
2. Press `d` on the approval card.
   You should see the diff modal. Each added line starts with a green `+`. `package` and `func` are in the keyword colour, and `"hi"` is in the string colour.
   - [ ] Each added line starts with a green `+`.
   - [ ] `package` and `func` are in the keyword colour.
   - [ ] `"hi"` is in the string colour.
3. Press `Esc` to close the diff, then press `n` to reject the patch.
   You should see the patch rejected, and no `notes.go` in the folder.
   - [ ] The patch is rejected, and `notes.go` is not created.
   If another approval card appears, press `n` on it too, and wait until Kirsch is idle.

### v0.1.4 · master section 5: `/diff` highlights each file in its own language (no model)

**Before you start:** Section 4 is done, and Kirsch is idle in Composing. Wait until the turn from section 4 has ended and Kirsch is idle before you type `/diff`. Make these changes by hand in another terminal, and do not run anything in Kirsch for them. In `~/kirsch-ux-test`, add the line `func extra() {}` at the end of `hello.go`, the file that section 3 made sure Git tracks. Create `tool.py` with the line `def extra(): pass`, and run `git add -N tool.py` so that `/diff` lists it. Then, in Kirsch, go on from section 4. This section ends in Composing, with `/diff` closed.

1. Type `/diff` and press `Enter`.
   You should see the `hello.go` section. Its `+` marker is green, and `func` is in the keyword colour. Its `diff --git` header is not coloured by language. The `index` hashes and the `@@` lines are git's output and may differ on your machine.
   - [ ] The `+` on `func extra() {}` is green.
   - [ ] `func` is in the keyword colour.
   - [ ] The `diff --git` header line has no keyword colour.
2. Look at the `tool.py` section.
   You should see `def` in the keyword colour on `+def extra(): pass`, with a green `+`. The `hello.go` section keeps its Go colours.
   - [ ] `def` is in the keyword colour.
   - [ ] The `+` on the `tool.py` line is green.
3. Press `Esc` to close `/diff`.
   - [ ] `Esc` closes `/diff`.

### v0.1.4 · master section 6: `NO_COLOR=1` shows everything plain, with the same layout (no model)

**Before you start:** Section 5 is done, and Kirsch is in Composing. Type `/quit` and press `Enter`. Start Kirsch again from `~/kirsch-ux-test` with `NO_COLOR=1 ~/path/to/kirsch/bin/kirsch`. It resumes the last session. This section ends in Composing, with `/diff` closed.

1. Type `/diff` and press `Enter`.
   You should see one file section for each changed file, as in section 5. The `hello.go` section has the line `+func extra() {}`, and the `tool.py` section has the line `+def extra(): pass`. Each `+` is plain text, with no colour, and so are `func` and `def`. The `index` hashes and the `@@` lines are git's output and may differ on your machine.
   - [ ] No line in `/diff` has any colour.
   - [ ] The `hello.go` section has the line `+func extra() {}`.
   - [ ] The `tool.py` section has the line `+def extra(): pass`.
2. Press `Esc` to close `/diff`.
   - [ ] `Esc` closes `/diff`.
3. Press `Shift+↑`, then `↑` until the reply from section 1 is selected. You should see its `#`, its backticks and its `~~`, and no `**`. Bold text is plain. Press `Tab` to return to the composer.
   - [ ] The reply from section 1 shows `#`, backticks and `~~`, and no `**`.

### v0.1.4 · master section 7: A `postgresql` fence is highlighted as SQL, and a `sql` fence is highlighted (model)

**Before you start:** Section 6 is done, and Kirsch is in Composing. Type `/quit` and press `Enter`. Start Kirsch again with `~/path/to/kirsch/bin/kirsch` from `~/kirsch-ux-test`, without `NO_COLOR`, so that colour is back. It resumes the session, so the section 1 reply stays for section 8. This section ends in Composing, with the composer empty.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > Reply with only two code fences. The first is tagged postgresql and the second is tagged sql. Each holds the line SELECT 1;

   You should see two code blocks. In each one, `SELECT` is in the keyword colour. The screen does not show the language tags. The prompt asks for the `postgresql` fence first, so the first block should be the `postgresql` one and the second the `sql` one.
   - [ ] `SELECT` in the `postgresql` block is in the keyword colour.
   - [ ] `SELECT` in the `sql` block is in the keyword colour.

### v0.1.4 · master section 8: `/copy` still copies the source markdown (no model)

**Before you start:** Section 7 is done, and Kirsch is in Composing with the composer empty. The reply from section 1 is still in the transcript, and so is the reply from section 7, which is the last reply. Open a plain text app, such as TextEdit, before each paste, and clear it before each paste. This section ends in Composing, with the composer empty.

1. Press `Shift+↑` to enter Browsing. Press `↑` until the reply from section 1 is selected, then press `c`. Paste into the text app.
   You should see the source markdown: `# Heading check`, `**bold words**`, `` `inline code` ``, `~~struck words~~`, `- first item`, and the fence lines. You should not see `•`.
   - [ ] The paste keeps `# Heading check` with its `#`.
   - [ ] The paste keeps `**bold words**` with its asterisks.
   - [ ] The paste keeps the fence lines, and has no `•`.
2. Press `Tab` to return to the composer. Type `/copy` and press `Enter`. Paste into the text app. `/copy` takes the last reply, which is the reply from section 7, not the reply from section 1. You should see that reply's source: its two fence lines, `postgresql` and `sql`, with `SELECT 1;` in each block.
   - [ ] The `/copy` paste holds the section 7 reply's source, with its fence lines.
   - [ ] The `/copy` paste does not hold the section 1 reply's text.

---

## v0.1.5 · Scrollbar and long lines in the modal

**Before you start:** The steps in this part use Unicode, which needs a UTF-8 Terminal, unless a section says otherwise. The hint texts quoted here are the Unicode forms. Section 10 quotes the ASCII form.

Section 1 starts a new session. Each section's **Before you start** line names the state it begins from, which is the state the section before it ends in.

Sections marked (no model) need a long transcript on screen. Section 2 makes one, by typing the long count prompt. A later session that already has one can be resumed by starting Kirsch without `--new`. If no session has one yet, run section 2 first.

### v0.1.5 · master section 1: A short transcript has no bar (model)

**Before you start:** If Kirsch from v0.1.4 section 8 is still running, type `/quit` and press `Enter`. Set the window to 80 columns by 24 rows, by dragging a corner of the window or using your terminal's window-size setting. Run `tput cols` and `tput lines` in that window, and check that they print `80` and `24`. Then start Kirsch with `~/path/to/kirsch/bin/kirsch --new` in `~/kirsch-ux-test`. Kirsch is idle in Composing, with the composer empty. This section ends in Composing, with the composer empty and a transcript that fits the screen.

1. Type `read README.md and reply in one short sentence`, press `Enter`, and wait until Kirsch is idle.
   You should see a transcript shorter than the screen. The right-hand column is blank on every row, beside the header, the transcript, the status line and the composer.
   - [ ] No bar is drawn beside the transcript.

### v0.1.5 · master section 2: The bar at the bottom of an overflowing transcript (model)

**Before you start:** Section 1 ends in Composing with the composer empty. The window is 80 by 24. Unicode. This section ends in Composing, pinned to the bottom, with the composer empty.

1. Type this prompt, press `Enter`, and wait until Kirsch is idle:

   > count from 1 to 300, one number per line

   You should see a bar in the right-hand column, running from the first transcript row down to the last one, above the rule that separates the transcript from the status line. A thumb, drawn `┃`, sits at the bottom of the bar, on the last transcript row. The track above it is drawn `│`. The length of the thumb depends on how long the transcript is, so match where it sits, not how long it is.
   - [ ] The bar runs from the first transcript row to the last one.
   - [ ] The thumb sits at the bottom of the bar.

### v0.1.5 · master section 3: The bar at the top (no model)

**Before you start:** Section 2 ends in Composing, pinned to the bottom, with the composer empty. The window is 80 by 24. Unicode. This section ends in Browsing, with the view at the top of the transcript and no modal open.

1. Press `Shift+↑` to enter Browsing. Then press `g`, which moves the view to the top whichever item is selected.
   You should see the thumb at the top of the bar, starting on the first transcript row, with the track below it down to the last transcript row.
   - [ ] The thumb is at the top of the bar.
   - [ ] The track runs below the thumb to the last transcript row.

### v0.1.5 · master section 4: The bar in the middle, with `↓ N new` (model)

**Before you start:** Section 3 ends in Browsing at the top of the transcript, with no modal open. The window is 80 by 24. Unicode. The transcript still overflows. This section ends in Browsing, scrolled up and not pinned, with a `↓ N new` indicator on the last transcript row.

1. Press `Tab` to return to the composer. Type `read README.md, then count from 1 to 300, one number per line` and press `Enter`. After the tool card finishes and before the reply's first text, press `Shift+↑`, then `g`, then `PgDn`. Wait until the reply has finished.
   `Shift+↑` returns to Browsing. `g` moves the view to the top, and `PgDn` moves it down one page. The view is then not pinned, so the blocks that arrive below it are counted in `↓ N new`. `N` counts those blocks, not lines.
   You should see the thumb on a row above the bottom of the bar, with track below it. On the last transcript row, the `↓ N new` indicator ends flush against the bar's column.
   If no tool card appears, write `no tool call` beside this step and send the prompt again.
   If no indicator appears, the keys came in an order that gives no count. Write `reply started first` or `reply finished first` beside this step, as applies, and try again with a longer prompt.
   - [ ] The thumb is not on the bottom row of the bar, and track shows below it.
   - [ ] The `↓ N new` indicator ends on the last transcript row, flush against the bar.

### v0.1.5 · master section 5: No bar under an open modal (model)

**Before you start:** Section 4 ends in Browsing, scrolled up and not pinned. The window is 80 by 24. Unicode. This section ends in Browsing, with no modal open.

1. Press `↑` or `↓` until a `read_file` card for `README.md` shows the `┃` gutter and the hint row begins `↑↓ select`. For example, the card from section 1 or the card from section 4 will do. The cues below hold whichever `README.md` card is opened, so do not try to tell them apart. If the gutter goes off screen, press the other arrow. If the hint row changes to `↑↓ history · ⇧↑/Tab cards · /help`, which is the composer's hint and not the Browsing one, you have stepped past the last item: press `Shift+↑` to return to the cards and carry on. Then press `d`.
   If a `d` reaches the composer by mistake, press `Esc` to clear it (no turn is running here, and `Esc` clears the composer in Composing when none is). You are still in the composer: press `Shift+↑`, bring the gutter back to the card with the arrows, and press `d` again.
   You should see a box over the transcript, with a title in the form `read_file <file>`. No bar is drawn in the right-hand column on any row. Fragments of the transcript behind the box show at the box's right edge.
   - [ ] No bar is drawn while the modal is open.
2. Press `Esc`.
   You should see the screen return to Browsing, with the bar back in the right-hand column.
   - [ ] The bar comes back when the modal closes.

### v0.1.5 · master section 6: A wrapped line in a `read_file` modal (model)

**Before you start:** Section 5 ends in Browsing, with no modal open. The window is 80 by 24. Unicode. This section ends with the `read_file` modal for `long.txt` open at its top.

1. Press `Tab`. Type `read long.txt and reply with one word`, press `Enter`, and wait until Kirsch is idle. Press `Shift+↑`, which enters Browsing from the composer. Then press `↑` or `↓` until the `read_file` card for `long.txt` shows the `┃` gutter and the hint row begins `↑↓ select`. If the gutter goes off screen, press the other arrow. If the hint row changes to `↑↓ history · ⇧↑/Tab cards · /help`, which is the composer's hint and not the Browsing one, you have stepped past the last item: press `Shift+↑` to return to the cards and carry on. Then press `d`.
   If a `d` reaches the composer by mistake, press `Esc` to clear it (no turn is running here, and `Esc` clears the composer in Composing when none is). You are still in the composer: press `Shift+↑`, bring the gutter back to the `long.txt` card with the arrows, and press `d` again.
   You should see a box whose title is `read_file long.txt`. Its first line is 150 characters wide, so it runs over several rows inside the box. The line number shows on the first row only. The rows below it leave the number columns blank, and their text starts in the column where the tool's own number sits on the first row. The modal's own number and the tool's number both show on that first row. That is how it is drawn now. The rule and the footer are drawn inside the box, as before: the footer sits on the row above the box's bottom edge.
   - [ ] The wrapped line runs over several rows, and the line number shows on the first row only.
   - [ ] The modal's number and the tool's number both show on the first row.
   - [ ] The footer `j/k scroll · g/G top/bottom · Esc close` is drawn inside the box, above its bottom edge.

   Pieces of the transcript behind the box show at its left and right edges.

### v0.1.5 · master section 7: `G` lands on the last full page (model)

**Before you start:** Section 6 ends with the `read_file` modal for `long.txt` open at its top. The window is 80 by 24. Unicode. This section ends in Browsing, with the modal closed.

1. Press `G`.
   You should see the body filled down to the last line of `long.txt`, with no blank row under it. The rule above the footer shows `↑ N above`, where `N` counts the display rows above the top of the body. It is a count of rows, not a line number.
   - [ ] The last body row is the last row of the file's last line, which may be a wrapped row. No blank row sits under it.
   - [ ] The rule shows `↑` with a count of rows above.
2. Press `Esc` to close the modal.
   You should see Browsing again. The hint line reads `↑↓ select · Enter preview · d detail · Tab/Esc prompt` while the `┃` gutter is on a card, and `↑↓ select · c copy · Tab/Esc prompt` while it is on a reply.

### v0.1.5 · master section 8: A wrapped `+` line and a wrapped `@@` line in `/diff` (no model)

**Before you start:** Section 7 ends in Browsing with no modal open. Before this section, make the window 80 columns by 32 rows, the size this section needs. Check it with `tput cols` and `tput lines`. Then make the unstaged change this section diffs, in the test folder:

```bash
cd ~/kirsch-ux-test
{ printf 'x%.0s' $(seq 1 150); echo; seq 1 80; printf 'z%.0s' $(seq 1 150); echo; } > long.txt
```

The last line of `long.txt` is now 150 `z` characters, where the setup made it 150 `y` characters. `/diff` lists every unstaged change in the folder, not only `long.txt`. For example, the `hello.go` change and the `tool.py` file that v0.1.4 section 5 leaves in this folder may appear as well. This section checks only the `long.txt` hunk. Git puts the start of `long.txt`'s first line in that hunk's header as its function context, 80 characters of it, so the header is wider than the text. This section ends in Composing, with the composer empty and no modal open.

1. Press `Tab` to return to the composer. Type `/diff` and press `Enter`.
   You should see a box titled `working tree`, with a stat on its top rule.
2. Press `j` or `PgDn` until the `long.txt` hunk is in view. The diff for `hello.go` can sit above it, so scroll past that part first.
   You should see a code line wider than the text wrap. Its `+` or `-` marker shows on the first row only. The row below it starts with a blank column where the marker is, and the code continues from there. A `@@` line wraps flush, with no blank column. A short `@@` line does not wrap, and that is not a failure: it wraps only when it is wider than the text.
   - [ ] A code line wider than the text shows its `+` or `-` on the first row only.
   - [ ] The row below a wrapped code line starts with a blank column where the marker is.
   - [ ] A wrapped `@@` line starts flush with the left edge of the text.
3. Press `Esc`.
   You should see the modal close, and Kirsch return to the composer, because `/diff` was typed there. The hint line under the composer shows the composer keys.

### v0.1.5 · master section 9: Help lines are cut, not wrapped, at 60×24 (no model)

**Before you start:** Section 8 ends in Composing, with the composer empty, at 80 by 32. Before this section, make the window 60 columns by 24 rows, and check it with `tput cols` and `tput lines`. This section ends in Browsing, with the help overlay closed, at 60 by 24.

1. Press `Shift+↑`, then `?`.
   You should see a box titled `help`, with two columns. Each binding is one row of the screen. A row longer than the box's text width ends in `⋯` at the right edge. It does not wrap onto the next row. The footer rule shows a `↓ N more` marker, where `N` counts the help rows below the ones showing.
   - [ ] Each help row is one row of the screen, and a long one ends in `⋯`.
   - [ ] The footer rule shows a `↓ N more` marker.
2. Press `Esc` to close the help box.

### v0.1.5 · master section 10: The bar with ASCII glyphs (no model)

**Before you start:** Section 9 ends in Browsing, with the help box closed, at 60 by 24. Set the window to 80 columns by 24 rows. Press `Tab`, type `/quit` and press `Enter`. Then start Kirsch from `~/kirsch-ux-test` with the locale set to C: `LC_ALL=C ~/path/to/kirsch/bin/kirsch`. Leave out `--new`, so the session resumes with its long transcript. This section ends in Composing, pinned to the bottom, with the composer empty and the glyph set ASCII.

1. Wait until the transcript shows.
   You should see a bar in the right-hand column, its track drawn `|` and its thumb `#`, with the thumb at the bottom of the bar. The status line and the hint use `.` in place of the Unicode separators, and the composer caret is `_`. The hint line reads `Up/Down history . Shift+Up/Tab cards . /help`. The frame is not pure ASCII. The footer of a tool card keeps its own Unicode glyphs, for example `‹` and `›`.
   - [ ] The track is `|` and the thumb is `#`, in the right-hand column.
   - [ ] The status and hint lines use `.` for the separators, and the composer caret is `_`.

---

## If a step does not match

Leave the box empty. Write what you saw beside the step, in a few words, and tell Bishop. Do not tick the box to move on. Some steps tell you to write a specific word or phrase, such as `notice above` or `no tool call`. Write that exact text when the step says to.

## If everything passed

When every box you were asked to check is ticked, this is what happens next, and who does it.

1. **You tell Bishop the test passed.** An agent then makes one local release commit on `main`. The commit changes the `[Unreleased]` heading in `CHANGELOG.md` to `[0.1.5]`, with the date, and brings the mission branches into `main` on this machine. Bishop reports the commit hash.
2. **You create the tag on that commit.** From the Kirsch folder, run the command below with the hash Bishop reports. Replace `...` with a short message, and `<hash>` with that hash:

   ```bash
   git tag -a v0.1.5 -m "..." <hash>
   ```

3. **You push main and the tag `v0.1.5`.** Run these two commands only after the tag command has worked. From the Kirsch folder, run:

   ```bash
   git push origin main
   git push origin v0.1.5
   ```

   After the tag is out, an agent raises the version the build reports to the next development number.
4. **Other actions you owe.** These are changes only you make, because they touch files or settings that the agents do not change. Bishop has the exact wording for each one.
   - Make the pending edit to the rule on counts and lists in `.claude/memory/reference/DIRECTIVES.md`.
   - Add a rule to the execution steps of `.claude/agents/bishop.md`: before a brief goes out, check any claim it relays against the current files, and mark or drop a claim that has not been checked. There is more than one wording, and you choose which one stands.
   - In `.claude/memory/findings/FINDINGS.md`, set the Status of each finding you have ratified. Only you set that field.
   - In `plan/PROGRESS.md`, fix the stale notes: the RD12 note, which refers to a function that has since been removed (its question about logging tool calls stays open), and the note at line 830, which gives old line numbers.
   - In `plan/PROGRESS.md`, line 5 still describes the manual-test waiver for the v0.1.4 section as in force. That waiver has been removed, so update the line.
   - Delete the temporary folders from this round once the round is closed. Bishop names them first.
