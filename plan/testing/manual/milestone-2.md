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

§2 and §4 create `newfile.txt` from the checkout. §11 now builds its own workspace, so it is unaffected either way.

---

## Findings Summary

**Code defects, now fixed:**
- Pressing `a` to approve for the session froze Kirsch — reported in §1, §4 and §7. It was a deadlock in the approval flow, fixed in `mission-20260928-01` and covered by `internal/app/deadlock_test.go`.
- Approval cards always showed a hardcoded `2.4s`. They now show the time from request to decision, using the model's clock (`resolveApproval`, `internal/tui/update.go`). Fixed in `mission-20261001-01`.
- Command result cards never named the command, because `describeInput` in `internal/app/app.go` ignored `argv`. They now name it, sanitised through `tui.SanitizeSingleLine`. Fixed in `mission-20261001-01`.

**Walkthrough wrong, corrected:**
- §9 `/run cat`: the third item now describes the result card and the approval card separately.
- §10 `/patch`: the setup now stands on its own instead of referring to §5, the containment item has a command behind it, and the first approval card is rejected before the second command is typed.
- §11: the section builds its own workspace, because a `hostile.txt` left by an earlier run makes the new-file patch fail; the diff is opened with `d` before `y`; the result card is expanded before its body is checked.
- §7 Part 2: notes that Part 1's grant already exists when both run in one session.

**Environment:**
- §10 `/help`: the help body clamps to the terminal height and `/run` sits alone on its last row, so at some heights it falls just below the fold. Scrolling reaches it.
- §6: the pasted `no Go files` card is what a bare `go test` reports at the Kirsch checkout root.

**Operator clarification:**
- §9 `/run cat`: the operator reports that an approval card did appear and they pressed `y`. The approval gate was verified by test. Their ❌ on the first item is not explained by the code and stands until they retest.

**Open questions, for the operator to decide:**
- Command output stays collapsed until opened. Current behaviour matches `plan/spec/ui-spec-v0.1.md` §3.3.
- One approved command produces two cards, the approval card and the result card.
- The first Up after a command completes selects the result card, not the approval card.
- An approval card's duration is the time taken to decide, shown where a result card shows its run time.

---

## 1 · The approval card for a blocked command

A command not on the allowlist raises a card. An allowlisted one does not.

```
/run echo hello
```

- [✅] A card appears with the box title `approval required` and the command shown in the body
- [✅] The card shows the exact command: `echo hello`
- [✅] There are rendered choice buttons: `[y]` approve, `[a]` approve for session (when applicable), `[n]` reject, and `[d]` detail. The `Esc` key works everywhere without being drawn.

Findings
When I click a the whole app crashes and I have to restart the terminal and the Kirsh once again

Press `n` to clear this card before typing the next command.

```
/run echo hello
# press y
```

- [✅] The command runs
- [✅] The output appears as the card's body
- [✅] A new prompt is ready

Now reject the command:

```
/run echo hello
# press n (or Esc)
# wait for the prompt
```

- [✅] The command does **not** run
- [✅] The card shows `rejected`
- [✅] A new prompt is ready

Try an allowlisted command (the allowlist is: `go build`, `go test`, `git diff`, `git log`, `ls` — exact matches only):

```
/run go test
```

- [✅] No approval card appears at all
- [✅] The command runs immediately
  At the Kirsch checkout root a bare `go test` finds no Go files, so the card shows `✗ exit status 1`. That is expected: this check is only that no approval card appears.
- [ ] The completed card appears collapsed, showing its head line; expand it by focusing the transcript (Up arrow) and pressing Enter (ui-spec §3.3)

Findings
I still have to go to the run_command and click eneter to view the command

## 2 · The patch card

A patch always asks, regardless of policy.

```
/patch create-file.diff
```

- [✅] A card appears with the box title `approval required`
- [✅] The card shows a file count: `[1 file]` or `[N files]`
- [✅] The card lists the paths that will be changed
- [✅] There are **exactly three rendered buttons**: `[y]` approve, `[n]` reject, `[d]` view diff. The `Esc` key works everywhere without being drawn.

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

- [✅] The command runs
- [✅] A second `/run echo approval-once-test` still asks for approval
- [✅] The approval status is not carried over

Press `n` to clear the second card.

**Outcome: `n` rejects**

```
/run echo rejection-test
# press n
```

- [✅] The command does not run
- [✅] The card shows `rejected`
- [✅] The prompt returns immediately

**Outcome: `a` approves for the session**

```
/run echo session-approval-test
# press a
```

- [✅] The command runs
- [✅] A second `/run echo session-approval-test` runs **without asking**
- [✅] A third `/run echo session-approval-test` also runs without asking


**Outcome: `Esc` cancels**

```
/run echo cancellation-test
# press Esc
```

- [✅] The command does not run
- [✅] The card shows a cancellation state
- [✅] The prompt returns immediately
- [✅] A second `/run echo cancellation-test` still asks for approval (not remembered)

Press `n` to clear the second card.

## 4 · The `[a]` button (approval for session) is never on patches

Start with a unapproved command that accepts `a`, then try it on a patch:

```
/run sleep 0.1
# press a
/run sleep 0.1
```

- [✅] The second command runs without asking

Findings
As above when clicking a the entire Kirsh crashes and I have to close the terminal

```
/patch create-file.diff
```

- [✅] The patch card appears
- [✅] There are **exactly three rendered buttons**: `[y]` approve, `[n]` reject, `[d]` view diff. The `Esc` key works everywhere without being drawn.
- [✅] The `a` button is absent

Findings
I see the y , n and there is d for diff there is now escape

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

**Note:** This setup was derived from the code at `cmd/kirsch/main.go` in the previous mission
and has since been confirmed working by the operator's own pass through §5 above.

Then in the running Kirsch session:

```
/patch modify-single-hunk.diff
# press d
```

- [✅] A modal appears with the filename in the title
- [✅] The modal shows the line count in the header: `+N −M` (Unicode minus) for that file
- [✅] The modal body shows the diff with its line prefixes (` ` for context, `+`
      for added, `-` for removed in the actual diff lines; the header uses Unicode `−`)
- [✅] `Esc` closes the modal and returns to the card
- [✅] The card is still visible and the buttons still work

Press `n` to clear the card. Then quit Kirsch with `/quit` and restart it from the checkout with `npm start` before §6, which needs the checkout as its workspace.

## 6 · The diff modal on a command card

The detail view (pressed with `d`) appears only when a command card is in focus. After
a tool completes, the focus is in the composer input field. Press Up arrow to move focus
to the transcript (where the tool card sits), then press `d` to open the detail view.

```
/run go test -v ./internal/...
# An approval card appears
# press y (approve the command)
# Wait for the output to appear as a card
# press Up arrow (moves focus from composer to transcript)
# press d (opens detail view of the output)
```

- [✅] An approval card appears because the command has extra arguments beyond the two-token `go test` pattern
- [✅] After approval, the command runs and output appears as a card
- [✅] A modal appears showing a detail view
- [✅] `Esc` closes the modal and returns to the card

**Why approval is needed:** The allowlist entry for `go test` is exactly two tokens (`go` and `test`),
non-extendable. Any command with additional arguments — like `-v ./internal/...` — raises an approval
card before running.

**Why the Up arrow is needed:** After a tool completes, the text input focus sits in the
composer at the prompt. Keyboard input would be inserted as text at the cursor. The `d`
key only triggers a detail view when a message card has focus in the transcript — pressing
Up moves focus there, so the key binding works.

Findings
I am seeing this

 ┃ ▾ run_command · 30ms · ✗ exit status 1
 ┃   ┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
 ┃   │ # .                                                                                                                                        │
 ┃   │ no Go files in /Volumes/DATA/Github/kirsch                                                                                                 │
 ┃   │ FAIL    . [setup failed]

Confirmed and documented: the original instructions were missing the navigation step required to reach the transcript and make the `d` key functional.

**Status:** The card pasted above reads `no Go files in /Volumes/DATA/Github/kirsch`, which is what a bare `go test` reports at the Kirsch checkout root, where there are no Go files. It most likely came from §1's `/run go test` rather than this section's `go test -v ./internal/...` — an inference from the paste, not confirmed by rerunning. A bare `go test` failing there is expected and is not what this section tests.

## 7 · Session grants — listing and clearing

**Part 1: Approve a command for the session**

```
/run echo grant-test
# press a
/run echo grant-test
```

- [✅] The first command runs
- [✅] A second `/run echo grant-test` runs without asking
- [✅] A third `/run echo grant-test` also runs without asking

Findings
Preessing a crashes Kirsch

**Status:** Previously reported as crashing. This was a deadlock in the approval flow,
fixed in `mission-20260928-01`. Now covered by regression tests in `internal/app/deadlock_test.go`.

**Part 2: Clear grants and verify re-prompting**

When you have a session grant, `/approvals` opens a modal listing the approved commands.
The `c` key clears all grants. Test the full flow:

```
/run echo grant-test
# press a (approves for this session)
/approvals
# (a modal opens listing the grant)
# press c (clears all grants)
# press y to confirm
```

- [✅] The modal closes
- [✅] The grants list is now empty
- [✅] A new `/run echo grant-test` asks for approval again

Press `n` to clear the card.

**Status:** The operator has since run this part and ticked all three items. If you have just run Part 1 in the same session, `echo grant-test` is already granted, so the first `/run echo grant-test` runs with no approval card and the `a` keypress lands in the composer. Run Part 2 in a fresh session, or skip straight to `/approvals`.

**Note:** The footer renders `c to clear (confirm)` in progressively shorter variants as the terminal narrows (`internal/tui/modal.go:308-310`).
At 39 columns and above, some variant of the instruction fits; as the terminal widens, more scroll bindings appear alongside it.
Below 39 columns, no variant carrying the instruction fits, and the footer shows only the exit key (`Esc`), read from `modalExit` at line 311.
The `c` key does not appear in any help binding.

Findings
Maybe the c should appear in the modal so we know how to clear the session approvals

## 8 · The boundary — patch directory containment

A patch whose path escapes the patch directory (`testdata/patches/`) is refused before any prompt appears.
Note that `../escape.txt` resolves to `testdata/escape.txt`, which is outside `testdata/patches/`,
triggering the containment check at `cmd/kirsch/main.go:180-182`:

```
/patch ../escape.txt
```

- [✅] A card or message appears **without an approval prompt**
- [✅] It explains that the path is outside the patch directory

A shell-escaping command always asks, even if it is on the allowlist:

```
/run sh -c "echo hello"
```

- [✅] An approval card appears
- [✅] The command text shows `sh -c "echo hello"`

Press `n` to clear the card.

A grant for a command covers extensions of that command by prefix matching, but shells are never granted.
Grants match with `extendable: true`, so any argv that begins with the granted prefix is covered:

```
/run echo grant-demo
# press a
/run echo grant-demo with more args
```

- [✅] The first command is approved for the session
- [✅] The second command runs without asking (prefix match — both start with `echo grant-demo`)

```
/run sh -c "echo test"
```

- [✅] An approval card appears (shells are never granted, regardless of prior grants)

Press `n` to clear the card.

## 9 · The command tool — timeout and process handling

A command that runs longer than its timeout has its process group killed:

```
/run sleep 10
# immediately (before it finishes) press Esc on the card
```

- [✅] The card shows a cancellation state
- [✅] Control returns to the prompt promptly
- [✅] No `sleep` process is left behind (`ps aux | grep sleep`)

**Status:** `sleep` is not on the allowlist, so this command raises an approval card, and Esc on that card cancels the request before any process starts. These items check that cancelling an approval works. They do not exercise the timeout kill this section's heading describes, and nothing else in this walkthrough does either.

A command that reads from standard input gets `/dev/null` and exits immediately:

```
/run cat
# press y
```

- [✅] An approval card appears (cat is not allowlisted)
- [✅] The command does not block
- [✅] After `y`, the result card reads `▸ run_command cat · completed · <duration> · ✓ ok`: it names the command and carries no `✗`. It is a separate card from the approval card, which collapses to `▸ run_command cat · <duration> · ✓ approved`.
- [✅] The prompt returns

**Status:** On 2026-10-01 the operator confirmed that an approval card did appear for `/run cat` and that they pressed `y`, so the approval gate held. Their ❌ on the first item stands until they retest it. The third item was rewritten: before this mission the result card never named the command, because `describeInput` in `internal/app/app.go` ignored `argv`, and the approval card always showed a hardcoded `2.4s`. Both are fixed. One approved command produces two cards: the result card is assembled by `renderTool` in `internal/tui/cards.go` and carries the `completed` summary and the `✓ ok` badge; the approval card is assembled by `renderApproval` and carries `✓ approved` with no summary. The operator had marked the original third item ❌. It is blank because its text changed.

**Why no exit status is shown:** On success `run_command.go` sets the summary to `completed` and records no exit code; the string `exit status N` is produced only on the failure branch, so an exit code is never visible in a successful run.

Shells always require approval, even if a command itself is allowlisted. The `/run`
debug command's argument parser handles double quotes but not single quotes or
backslash escapes — those are parsed as literal characters:

```
/run echo hello
# press a to approve for this and all similar commands
/run echo hello world
# verify the grant works and the extension does not ask
```

- [✅] The first command is approved for the session
- [✅] The second command runs without asking (prefix match — the grant covers any argv starting with `echo hello`)

A command with a variable named `*_KEY`, `*_TOKEN`, `*_SECRET`, or `AWS_*` does not pass that
variable to the child, even if allowlisted or approved. This filtering is applied unconditionally
in `internal/tool/run_command.go:374-381`. The test must allowlist a secret-shaped variable to demonstrate that it is stripped regardless of allowlisting.

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

Config file location and structure (read from `internal/config/config.go:50` and `internal/config/config.go:301`):
- **Location:** `.kirsch/config.toml` in the workspace root
- **TOML key:** `env_passthrough` under `[policy]` section
- **Default:** empty list `[]`

The `env_passthrough` list (built at `internal/tool/run_command.go:365-372`) is combined with the permanent allowlist (`PATH`, `HOME`, `LANG`), then the secret patterns are applied at lines 374–381.

Then in the running Kirsch session:

```
/run env
# press y (shell commands require approval; isShell() in internal/policy/policy.go)
```

- [✅] An approval card appears
- [✅] The environment output appears in a card
- [✅] `ORDINARY_VAR` is visible in the output (proving the allowlist mechanism works)
- [✅] `SECRET_KEY` is absent from the output (stripped by pattern `*_KEY` even though allowlisted)
- [✅] Other variables like `PATH` and `HOME` are visible (permanent allowlist items)

**Why the ordinary variable is present:** It demonstrates that the passthrough mechanism is working — the variable reaches the subprocess because it is allowlisted and not secret-shaped. If `ORDINARY_VAR` were absent, the test would not be discriminating: a secret's absence could be either the strip or a broken allowlist. With both variables allowlisted and only the secret stripped, the test proves the stripping code is active and effective.

Quit Kirsch with `/quit` and restart it from the checkout with `npm start` before §10.

## 10 · Debug commands

The debug commands are labelled `debug (M1–M2)` in `/help`, rendered by `helpLines()` at `internal/tui/modal.go:585`. The help modal displays two columns of bindings; the debug commands sit at the bottom of the right-hand column.

```
/help
```

- [✅] `/patch` appears with the label `debug (M1–M2)`
- [❌] `/run` appears with the label `debug (M1–M2)` (scroll with `j`/`k`, `g`/`G`, or PgUp/PgDn if the modal clamps content to available height — see Known Limitations)

Findings
No I am not seeing debug (M1–M2) I am seeing M1-M2 with patch only, also I am not seeing anything with run

**Status:** `helpLines()` in `internal/tui/modal.go` renders the label as `debug (M1–M2)` with an en dash, which can read as a hyphen. It lays the debug commands two to a row, so `/patch` shares a row with `/gitdiff` and `/run` sits alone on the last row of the help body. At some terminal heights the fold falls between those two rows, so the label and `/patch` show and `/run` does not. `TestHelpScrollingReachesDebugCommands` in `internal/tui/header_test.go` reproduces this and confirms scrolling brings `/run` into view. The ❌ above is the operator's, and it stands until they retest with scrolling.

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

Findings

 ┃ ▾ apply_patch · 0ms · ✗ file not found during patch: cannot read file: open /Volumes/DATA/Github/kirsch/plain.txt: no such file or directory
 ┃   ┌────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
 ┃   │                                                                                                                                        │
 ┃   └────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
 Same result again

 I see this at the bottom > Ask anything (Enter to send, /help for help)
   patch: open /private/var/folders/nb/46t8bdcd3050lnvz_6066k_40000gn/T/tmp.0Mgkak3Qu5/testdata/patches/modify-single-hunl.diff: no such file or di

**Status:** The operator's output shows the patch was not found at the Kirsch checkout root because the workspace setup from §5 was not repeated. The workspace setup above (using `TESTWS_SEC10`) ensures both the patch file and its target are in the correct locations relative to the workspace root, which should resolve this path-resolution issue. The operator's retest will confirm it. The operator had marked both of this section's original patch items ❌. Those items were replaced by the two above, which are blank for retest.

Test the `/run` debug command's quote handling:

Each of these commands raises an approval card. Press `y` on each card before typing the next command. The card and its output show how the argument was parsed.

```
/run echo "double quoted"
/run echo 'single quoted'
/run echo back\slash
```

- [✅] Double quotes work as expected
- [✅] Single quotes are treated as literal characters, not quote delimiters
- [✅] Backslash escapes are treated as literal backslash followed by the
      character, not as escape sequences

## 11 · Hostile content — terminal escapes and injection

Create a patch file with escape sequences in the content. Keyboard input produces literal
characters only — `\033` at the prompt is four literal characters, never an ESC byte.
A real ESC byte must come from a file:

This section builds its own workspace. A new-file patch is refused when its target already
exists, and an earlier run of this section leaves `hostile.txt` behind, so it must not run
in a directory where that file may already be.

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
# press Up (moves focus to the transcript; the result card is selected)
# press Enter (expands the result card to show its body)
```

- [✅] An approval card appears (patches always ask, regardless of content)
- [✅] A detail modal opens showing the diff with the title `hostile.txt`
- [✅] In the diff, the added line shows `+red text` with no visible escape sequences and no terminal colour rendering
- [✅] After `y`, a card appears showing `applied · ok`
- [✅] The result card body shows `patch applied` without escape codes or colour changes

**What happens to escape bytes:** The code strips ANSI sequences at `internal/tui/transcript.go:413`,
matching patterns CSI (`\x1b\[[0-?]*[ -/]*[@-~]`), OSC, and two-byte escapes. The sequences
are removed entirely by `Sanitize`, so patch content of `+\033[31mred text\033[0m` reaches the operator as
`+red text` — legible, unmangled, and posing no terminal-control risk. Approval cards' detail modals are opened via `keyApproval` at `internal/tui/update.go:722–724`, which sets the selection to the approval card and calls `openDetail`; the approval card's `Diff` field carries the sanitised lines.

Findings
I am not sure what I am looking for here but this is what I am seeing

 ▾ apply_patch · applied · 5.3s · ✓ ok
   ┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
   │ patch applied                                                                                                                                │
   └──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

 ┃ ▸ apply_patch 1 file changed · 2.4s · ✓ approved

**Status:** This section now builds its own workspace. A new-file patch is refused when its target already exists (`applyCreate`, `internal/patch/apply.go`), and an earlier run left `hostile.txt` in the Kirsch checkout, so running here would fail before any approval card appeared. Because the setup and the key sequence changed, all five items are blank for retest. The operator had ticked three of the original four items and marked the fourth ❌. The diff is opened with `d` while the approval is pending (`keyApproval`, `internal/tui/update.go:722–724`); a result card's body stays hidden until the card is expanded with Enter (`KeyEnter`, `internal/tui/update.go`). The `2.4s` on the operator's pasted card is the old hardcoded duration, since fixed.





---

## Known limitations

The following are not bugs; they are recorded limitations of the current
implementation:

- The help overlay clamps its body to the terminal height with nothing on
  screen to say content is below the fold. At 80×24 neither debug command is
  visible. The overlay does scroll, but no footer names the keys.
- A tool card's target is sanitised through `describeInput` and `SanitizeSingleLine` (`internal/app/app.go:770` and `internal/tui/transcript.go:449`), removing ANSI escapes and newlines. The tool card's name (`c.Name`) and the `path` and `query` fields for other tool types are not sanitised, and remain unsanitised for a later milestone.
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
