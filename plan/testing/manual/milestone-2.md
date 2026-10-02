# Manual test — Milestone 2 (approval flow, policy engine, write tools)

Checking that Kirsch asks before it changes the world, shows you honestly what
it is asking about, and refuses what it should. Milestone 2 is where the write
tools landed, so this walkthrough is mostly verifying that they carry the
approval surface back to the policy engine and the TUI.

**Time:** about 40 minutes.
**Prerequisite:** `npm test` and `npm run security` both green.

```bash
cd /path/to/kirsch
npm start
```

**Workspace:** run every section in a session started this way, from the Kirsch checkout. §5, the environment part of §9, §10 and §11 start Kirsch against a scratch workspace instead, and each says so in its own setup. While an approval card is waiting it takes every key you type, so resolve each card before typing the next command.

**Important:** Typing anything that is not a slash command (e.g., `hello`, arbitrary text)
triggers the fixture demo — scripted approval cards appear as if the model sent them.
This is expected behaviour during testing. Avoid typing non-command text at the prompt
to prevent test confusion.

**Before you start:** a previous run of this walkthrough leaves files behind that make later runs fail. A new-file patch is refused when its target already exists, so remove them from the Kirsch checkout first:

```bash
cd /path/to/kirsch
rm -f -- newfile.txt hostile.txt
```

§2 and §4 create `newfile.txt` from the checkout. §11 builds its own workspace, so it is unaffected either way.

**To run this walkthrough:**

1. Copy this file: `cp manual/milestone-2.md manual/user-testing/milestone-2-<yourname>.md`
2. Mark each check ✅ or ❌ in your copy. A ❌ with a sentence about what you saw is worth more than a long report.
3. The master stays blank. Commit both files: the blank master and your completed run.

---

## 1 · The approval card for a blocked command

A command not on the allowlist raises a card. An allowlisted one does not.

```
/run echo hello
```

- [ ] A card appears with the box title `approval required` and the command shown in the body
- [ ] The card shows the exact command: `echo hello`
- [ ] There are rendered choice buttons: `[y]` approve, `[a]` approve for session (when applicable), `[n]` reject, and `[d]` detail. The `Esc` key works everywhere without being drawn.

Press `n` to clear this card before typing the next command.

```
/run echo hello
# press y
```

- [ ] The command runs
- [ ] The output appears as the card's body
- [ ] A new prompt is ready

Now reject the command:

```
/run echo hello
# press n (or Esc)
# wait for the prompt
```

- [ ] The command does **not** run
- [ ] The card shows `rejected`
- [ ] A new prompt is ready

Try an allowlisted command (the allowlist is: `go build`, `go test`, `git diff`, `git log`, `ls` — exact matches only):

```
/run go test
```

- [ ] No approval card appears at all
- [ ] The command runs immediately
  At the Kirsch checkout root a bare `go test` finds no Go files, so the card shows `✗ exit status 1`. That is expected: this check is only that no approval card appears.
- [ ] The completed card is already open, showing its output without any key press (up to 10 lines; `d` opens the full output).

## 2 · The patch card

A patch always asks, regardless of policy.

```
/patch create-file.diff
```

- [ ] A card appears with the box title `approval required`
- [ ] The card shows a file count: `[1 file]` or `[N files]`
- [ ] The card lists the paths that will be changed
- [ ] There are **exactly three rendered buttons**: `[y]` approve, `[n]` reject, `[d]` view diff. The `Esc` key works everywhere without being drawn.

Confirm that `[a]` never appears on a patch card, even though the command tool
shows it.

Then press `n`. Pressing `y` here creates `newfile.txt` in the checkout, and §4's `/patch create-file.diff` would then fail before its card appears.

## 3 · The four outcomes

Test each outcome on a command that requires approval. Start fresh each time:

**Outcome: `y` approves once**

```
/run echo approval-once-test
# press y
```

- [ ] The command runs
- [ ] A second `/run echo approval-once-test` still asks for approval
- [ ] The approval status is not carried over

Press `n` to clear the second card.

**Outcome: `n` rejects**

```
/run echo rejection-test
# press n
```

- [ ] The command does not run
- [ ] The card shows `rejected`
- [ ] The prompt returns immediately

**Outcome: `a` approves for the session**

```
/run echo session-approval-test
# press a
```

- [ ] Kirsch stays responsive after `a` — no freeze, and the next command can be typed
- [ ] The command runs
- [ ] A second `/run echo session-approval-test` runs **without asking**
- [ ] A third `/run echo session-approval-test` also runs without asking


**Outcome: `Esc` cancels**

```
/run echo cancellation-test
# press Esc
```

- [ ] The command does not run
- [ ] The card shows a cancellation state
- [ ] The prompt returns immediately
- [ ] A second `/run echo cancellation-test` still asks for approval (not remembered)

Press `n` to clear the second card.

## 4 · The `[a]` button (approval for session) is never on patches

Start with an unapproved command that accepts `a`, then try it on a patch:

```
/run sleep 0.1
# press a
/run sleep 0.1
```

- [ ] The second command runs without asking

```
/patch create-file.diff
```

- [ ] The patch card appears
- [ ] There are **exactly three rendered buttons**: `[y]` approve, `[n]` reject, `[d]` view diff. The `Esc` key works everywhere without being drawn.
- [ ] The `a` button is absent

Press `n` to clear the card.

## 5 · The diff modal on a patch card

The patch file and the files it targets must be reachable from the same workspace root.
Create a scratch workspace with both paths satisfied:

Quit the current Kirsch session with `/quit` first, then run these from the Kirsch checkout so the relative `testdata/` paths resolve:

```bash
# Set up a test workspace containing both the patch files and the files they target
TESTWS=$(mktemp -d)
mkdir -p "$TESTWS/testdata/patches"
# Copy the file the patch targets
cp testdata/repo-patch/plain.txt "$TESTWS/"
# Copy the patch file itself
cp testdata/patches/modify-single-hunk.diff "$TESTWS/testdata/patches/"
# Start Kirsch pointing at this workspace
npm --prefix /path/to/kirsch run start -- -workspace "$TESTWS"
```

Then in the running Kirsch session:

```
/patch modify-single-hunk.diff
# press d
```

- [ ] A modal appears with the filename in the title
- [ ] The modal shows the line count in the header: `+N −M` (Unicode minus) for that file
- [ ] The modal body shows the diff with its line prefixes (` ` for context, `+`
      for added, `-` for removed in the actual diff lines; the header uses Unicode `−`)
- [ ] `Esc` closes the modal and returns to the card
- [ ] The card is still visible and the buttons still work

Press `n` to clear the card. Then quit Kirsch with `/quit` and restart it from the checkout with `npm start` before §6, which needs the checkout as its workspace.

## 6 · The diff modal on a command card

The detail view (pressed with `d`) appears only when a command card is in focus. After
a tool completes, the focus is in the composer input field. Press Shift+↑ (or Tab) to move focus
to the transcript (where the tool card sits), then press `d` to open the detail view.

```
/run go test -v ./internal/...
# An approval card appears
# press y (approve the command)
# Wait for the output to appear as a card
# press Shift+↑ (moves focus from composer to transcript)
# press d (opens detail view of the output)
```

- [ ] An approval card appears because the command has extra arguments beyond the two-token `go test` pattern
- [ ] After approval, the command runs and output appears as a card
- [ ] A modal appears showing a detail view
- [ ] `Esc` closes the modal and returns to the card

**Why approval is needed:** The allowlist entry for `go test` is exactly two tokens (`go` and `test`),
non-extendable. Any command with additional arguments — like `-v ./internal/...` — raises an approval
card before running.

**Why Shift+↑ is needed:** After a tool completes, the text input focus sits in the
composer at the prompt. Keyboard input would be inserted as text at the cursor. The `d`
key only triggers a detail view when a message card has focus in the transcript — pressing
Shift+↑ (or Tab) moves focus there, so the key binding works.

## 7 · Session grants — listing and clearing

**Part 1: Approve a command for the session**

```
/run echo grant-test
# press a
/run echo grant-test
```

- [ ] The first command runs
- [ ] A second `/run echo grant-test` runs without asking
- [ ] A third `/run echo grant-test` also runs without asking

**Part 2: Clear grants and verify re-prompting**

When you have a session grant, `/approvals` opens a modal listing the approved commands.
The `c` key clears all grants. Test the full flow:

```
/run echo grant-test
# press a (approves for this session)
/approvals
```

- [ ] At 80 columns or wider, the `/approvals` modal footer names `c to clear (confirm)`

```
# press c (clears all grants)
# press y to confirm
```

- [ ] The modal closes
- [ ] The grants list is now empty
- [ ] A new `/run echo grant-test` asks for approval again

Press `n` to clear the card.

**Note:** If you have just run Part 1 in the same session, `echo grant-test` is already granted, so the first `/run echo grant-test` runs with no approval card. Run Part 2 in a fresh session, or skip straight to `/approvals`.

Below 39 columns, the footer has room only for the exit key (`Esc`).

## 8 · The boundary — patch directory containment

A patch whose path escapes the patch directory (`testdata/patches/`) is refused before any prompt appears.
Note that `../escape.txt` resolves to `testdata/escape.txt`, which is outside `testdata/patches/`,
triggering the containment check:

```
/patch ../escape.txt
```

- [ ] A card or message appears **without an approval prompt**
- [ ] It explains that the path is outside the patch directory

A shell-escaping command always asks, even if it is on the allowlist:

```
/run sh -c "echo hello"
```

- [ ] An approval card appears
- [ ] The command text shows `sh -c "echo hello"`

Press `n` to clear the card.

A grant for a command covers extensions of that command by prefix matching, but shells are never granted.
Grants match with `extendable: true`, so any argv that begins with the granted prefix is covered:

```
/run echo grant-demo
# press a
/run echo grant-demo with more args
```

- [ ] The first command is approved for the session
- [ ] The second command runs without asking (prefix match — both start with `echo grant-demo`)

```
/run sh -c "echo test"
```

- [ ] An approval card appears (shells are never granted, regardless of prior grants)

Press `n` to clear the card.

## 9 · The command tool — timeout and process handling

A command that runs longer than its timeout has its process group killed. In this example, pressing Esc on an approval card cancels the request before any process starts:

```
/run sleep 10
# immediately (before it finishes) press Esc on the card
```

- [ ] The card shows a cancellation state
- [ ] Control returns to the prompt promptly
- [ ] No `sleep` process is left behind (`ps aux | grep sleep`)

**Note:** `sleep` is not on the allowlist, so this command raises an approval card, and Esc on that card cancels the request before any process starts. This demonstrates that cancelling an approval works. The timeout kill is covered by automated tests (`TestRunCommandTimeoutProcessGroupDead` in `internal/tool/run_command_test.go`), not by this walkthrough.

A command that reads from standard input gets `/dev/null` and exits immediately:

```
/run cat
# press y
```

- [ ] An approval card appears (cat is not allowlisted)
- [ ] The command does not block
- [ ] After `y`, one card appears: it names the command and shows `completed · approved · <duration> · ✓ ok` 
- [ ] The prompt returns

Shells always require approval, even if a command itself is allowlisted. The `/run`
debug command's argument parser handles double quotes but not single quotes or
backslash escapes — those are parsed as literal characters:

```
/run echo hello
# press a to approve for this and all similar commands
/run echo hello world
# verify the grant works and the extension does not ask
```

- [ ] The first command is approved for the session
- [ ] The second command runs without asking (prefix match — the grant covers any argv starting with `echo hello`)

A command with a variable named `*_KEY`, `*_TOKEN`, `*_SECRET`, or `AWS_*` does not pass that
variable to the child, even if allowlisted or approved. This filtering is applied unconditionally. The test must allowlist a secret-shaped variable to demonstrate that it is stripped regardless of allowlisting.

Create a test workspace with a Kirsch config file:

Quit the current Kirsch session with `/quit` first. These commands use no relative paths, so they can run from any directory.

```bash
# Set up a temporary directory with a config file
TESTWS=$(mktemp -d)
mkdir -p "$TESTWS/.kirsch"

# Create a config that allows both a secret-shaped variable and an ordinary variable
cat > "$TESTWS/.kirsch/config.toml" << 'EOF'
[policy]
env_passthrough = ["SECRET_KEY", "ORDINARY_VAR"]
EOF

# Export both variables to the shell
export SECRET_KEY="test-secret"
export ORDINARY_VAR="test-ordinary"

# Start Kirsch pointing at this workspace
npm --prefix /path/to/kirsch run start -- -workspace "$TESTWS"
```

Then in the running Kirsch session:

```
/run env
# press y (shell commands require approval)
```

- [ ] An approval card appears
- [ ] The environment output appears in a card
- [ ] `ORDINARY_VAR` is visible in the output (proving the allowlist mechanism works)
- [ ] `SECRET_KEY` is absent from the output (stripped by pattern `*_KEY` even though allowlisted)
- [ ] Other variables like `PATH` and `HOME` are visible (permanent allowlist items)

**Why the ordinary variable is present:** It demonstrates that the passthrough mechanism is working — the variable reaches the subprocess because it is allowlisted and not secret-shaped. If `ORDINARY_VAR` were absent, the test would not be discriminating: a secret's absence could be either the strip or a broken allowlist. With both variables allowlisted and only the secret stripped, the test proves the stripping code is active and effective.

Quit Kirsch with `/quit` and restart it from the checkout with `npm start` before §10.

## 10 · Debug commands

The debug commands are labelled `debug (M1–M2)` in `/help`. The help modal displays two columns of bindings; the debug commands sit at the bottom. The `/run` command may fall below the fold if your terminal is short, requiring scrolling to reach it.

```
/help
```

- [ ] `/patch` appears with a label containing the text `M1–M2` (or `M1-M2`)
- [ ] `/run` appears with a label containing the text `M1–M2` (scroll with `j`/`k`, `g`/`G`, or PgUp/PgDn if the modal clamps content to available height)

Patch commands need the workspace setup from §5 to work correctly. Close the help modal with `Esc`, then quit Kirsch with `/quit` before running the setup below.

This section repeats that setup in self-contained form so §10 does not depend on §5 having run in the same session.

```bash
# Run these from the Kirsch checkout, so the relative testdata/ paths resolve
# Set up a fresh test workspace for §10 (independent of §5)
TESTWS_SEC10=$(mktemp -d)
mkdir -p "$TESTWS_SEC10/testdata/patches"
# Copy the file the patch targets
cp testdata/repo-patch/plain.txt "$TESTWS_SEC10/"
# Copy the patch file itself
cp testdata/patches/modify-single-hunk.diff "$TESTWS_SEC10/testdata/patches/"
# Start Kirsch pointing at this workspace
npm --prefix /path/to/kirsch run start -- -workspace "$TESTWS_SEC10"
```

Then in the running Kirsch session:

```
/patch modify-single-hunk.diff
# an approval card appears; check it lists plain.txt, then press n to reject it
/patch ../escape.txt
# refused before any card appears; the reason shows as a hint at the composer
```

- [ ] An approval card appears listing `plain.txt` as the file the patch changes
- [ ] `/patch ../escape.txt` raises no approval card; the hint at the composer reads `patch: patch file "../escape.txt" is outside testdata/patches`

Test the `/run` debug command's quote handling:

Each of these commands raises an approval card. Press `y` on each card before typing the next command. The card and its output show how the argument was parsed.

```
/run echo "double quoted"
/run echo 'single quoted'
/run echo back\slash
```

- [ ] Double quotes work as expected
- [ ] Single quotes are treated as literal characters, not quote delimiters
- [ ] Backslash escapes are treated as literal backslash followed by the
      character, not as escape sequences

## 11 · Hostile content — terminal escapes and injection

Create a patch file with escape sequences in the content. Keyboard input produces literal
characters only — `\033` at the prompt is four literal characters, never an ESC byte.
A real ESC byte must come from a file.

This section builds its own workspace. A new-file patch is refused when its target already
exists, so it must not run in a directory where that file may already be.

Quit the current Kirsch session with `/quit` before running the setup below.

```bash
# Build a fresh workspace for this section
TESTWS_SEC11=$(mktemp -d)
mkdir -p "$TESTWS_SEC11/testdata/patches"
# Create a patch with an actual ESC byte in the added content
printf 'diff --git a/hostile.txt b/hostile.txt\nnew file mode 100644\n--- /dev/null\n+++ b/hostile.txt\n@@ -0,0 +1 @@\n+\033[31mred text\033[0m\n' > "$TESTWS_SEC11/testdata/patches/escape-content.diff"
# Start Kirsch pointing at this workspace
npm --prefix /path/to/kirsch run start -- -workspace "$TESTWS_SEC11"
```

Then in the running Kirsch session:

```
/patch escape-content.diff
# (an approval card appears — /patch always asks)
# press d (opens the diff while the approval is still pending)
# press Esc (closes the diff and returns to the approval card)
# press y (approves; the patch applies)
```

- [ ] An approval card appears (patches always ask, regardless of content)
- [ ] A detail modal opens showing the diff with the title `hostile.txt`
- [ ] In the diff, the added line shows `+red text` with no visible escape sequences and no terminal colour rendering
- [ ] After `y`, a card appears showing `applied · ok`
- [ ] The result card body shows `patch applied` without escape codes or colour changes

**What happens to escape bytes:** The code strips ANSI sequences from patch content, matching patterns for CSI escape sequences and related forms. The sequences
are removed entirely by `Sanitize`, so patch content of `+\033[31mred text\033[0m` reaches the operator as
`+red text` — legible, unmangled, and posing no terminal-control risk.

---

## 12 · UX pass (terminal width and height)

Run this section in a fresh session from the Kirsch checkout (`npm start`). The terminal must be
at least **120 columns wide** and **between 24 and 33 rows tall**; check with `stty size` (it
prints rows, then columns) before starting. Some card heads checked below are wider than 80
columns, and Kirsch cuts a line at the right edge without an ellipsis, so a narrower terminal
would hide the text being checked.

### 12.1 · Card preview

The line count below is `go.mod`'s as of 2026-10-01. If `wc -l < go.mod` now prints a
different number, expect that number wherever `54` appears.

In the prompt:

```
/read go.mod
```

- [ ] The card opens without any key press, glyph `▾`, showing the first 10 lines of `go.mod`
- [ ] Under the box a dim line reads `‹10 of 54 lines — d full output · Enter collapse›`
- [ ] The card head reads `read_file go.mod · lines 1-54 …` — the path appears once

```
# press Shift+↑ (focus moves to the cards; the go.mod card is selected)
# press Enter
```

- [ ] The card collapses to its head line only, glyph `▸`

```
# press Enter again
```

- [ ] The 10-line preview and its marker return

```
# press d
```

- [ ] A modal opens showing all 54 lines (scroll with `j`/`k`)

```
# press Esc (closes the modal), then Tab (focus returns to the prompt)
```

### 12.2 · No empty last row

In the prompt (`ls` with an argument is not allowlisted, so an approval card appears):

```
/run ls cmd
# press y
```

- [ ] The output box holds exactly one row, `kirsch`, directly above its bottom border — no
      empty row between them

### 12.3 · Command history

In the prompt, type each line and press Enter (each shows `unknown command …`):

```
/zz1
/zz2
```

Then press ↑, ↑, ↓, ↓, checking the prompt after each press.

- [ ] The first ↑ fills the prompt with `/zz2`
- [ ] The second ↑ shows `/zz1`
- [ ] The first ↓ shows `/zz2` again
- [ ] The second ↓ leaves the prompt empty

Type `abc` without pressing Enter, press ↑, then ↓.

- [ ] ↑ shows `/zz2`
- [ ] ↓ brings `abc` back

Press Esc to clear the prompt.

### 12.4 · Moving between the prompt and the cards

The `┃` gutter marks the selected card whichever part has focus, so it does not show where
focus is; the bottom row does.

In the prompt, with the prompt empty, press Shift+↑:

- [ ] Focus moves to the cards — the bottom row reads
      `↑↓ select · Enter preview · d detail · Tab/Esc prompt`

Press Tab:

- [ ] Focus returns to the prompt — the bottom row reads `↑↓ history · ⇧↑/Tab cards · /help`

Press Shift+↑, then Esc:

- [ ] Focus returns to the prompt again, with the same bottom row

In the prompt, type `/ru` and press Tab:

- [ ] It completes to `/run` and focus stays in the prompt

Press Esc to clear it, type `hello`, then press Tab:

- [ ] Focus moves to the cards — the bottom row reads
      `↑↓ select · Enter preview · d detail · Tab/Esc prompt`

Press Tab to return, then Esc to clear `hello`.

### 12.5 · One card per approved command

In the prompt (`echo` is not allowlisted, so an approval card appears):

```
/run echo one-card
# press y
```

- [ ] One card remains for this command; its head reads
      `run_command echo one-card · completed · approved · <duration> · ✓ ok`
- [ ] No separate `✓ approved` line is left in the transcript

```
/run echo grant-ux
# press a
```

- [ ] One card remains for this command, with no separate approval line
- [ ] Its head contains `approved for session · session grant: echo grant-ux`
- [ ] A second `/run echo grant-ux` runs with no approval card

```
/run echo reject-ux
# press n
```

- [ ] The approval card stays and reads `rejected`
- [ ] The command's own card is shown as well — two cards for this command

### 12.6 · A stray key cannot resolve a pending card

In the prompt:

```
/run echo release-ux
# an approval card appears
# type x
```

- [ ] The prompt area shows `approval pending — Tab to return to the card`
- [ ] Pressing `y` now does nothing — the card stays pending

Press Tab:

- [ ] The pending message disappears

Press `n`:

- [ ] The card is rejected

### 12.7 · Key-hint line

At 24 rows or more:

- [ ] In the prompt, the bottom row reads `↑↓ history · ⇧↑/Tab cards · /help`
- [ ] After Shift+↑ (the cards), it reads `↑↓ select · Enter preview · d detail · Tab/Esc prompt`

Press Tab to return to the prompt.

```
/run echo hint-ux
# an approval card appears — do not resolve it yet
```

- [ ] The bottom row reads `y approve · a session · n reject · d detail · Esc cancel`
- [ ] Type `x`: it changes to `Tab back to the card · Esc cancel`

Press Tab, then `n`, to clear the card. This leaves a rejected approval card and the command's own card in the transcript; that is expected.

```
/help
```

- [ ] With help open, the bottom row is blank
- [ ] Nothing else on the screen moves up or down when help opens

Press Esc. Shrink the terminal to about 20 rows:

- [ ] The key-hint row disappears

Restore the terminal to between 24 and 33 rows.

### 12.8 · Help clipping marker

The terminal should be between 24 and 33 rows tall: help needs 34 rows to fit without clipping.
In the prompt:

```
/help
```

- [ ] The line just above the help footer shows `↓ N more`
- [ ] Press `G`: that line now shows `↑ N above`
- [ ] It no longer shows `↓`

Press Esc.

---

## Known limitations

The following are not bugs; they are recorded limitations of the current
implementation:

- The help overlay is taller than the terminal below 34 rows. It scrolls, and a marker on
  its rule row (`↓ N more`, `↑ N above`) shows that content is clipped, but no footer names
  the scroll keys.
- A tool card's target is sanitised, removing ANSI escapes and newlines. The tool card's name and the `path` and `query` fields for other tool types are not sanitised, and remain unsanitised for a later milestone.
- The box builder does not bound a row against its own width, so a very long
  path can overrun a card's right border.

---

## Reporting

For anything marked ❌:

1. The exact command and what happened
2. Your terminal size and color support (`echo $COLUMNS $LINES` and
   `tput colors`)
3. Whether you are running in tmux or another multiplexer (they affect
   rendering and signal handling)
4. The debug log from `npm run start:debug`

A finding about the approval flow or the command execution is worth reporting
even if you are unsure.
