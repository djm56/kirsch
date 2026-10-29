# Milestone 5 — your test checklist

This file is yours. Each section tests one piece of Milestone 5 once it is finished. You paste a command, compare what you see, and tick a box.

## How to use this file

1. Only do a section when Bishop tells you it is ready. A ready section has numbered steps; a section that is not ready says *Not ready yet*.
2. Do *Before you start* first.
3. In a section, do each step in order: paste the command into Terminal, press Enter, and compare what you see with *You should see*.
4. If it matches, tick the box: change `- [ ]` to `- [x]`. A check mark (✅) inside the brackets also counts as a tick.
5. If it does not match, do not tick it. Write what you saw under *Anything odd?* at the end of the section and tell Bishop the section and step number.
6. Never paste an API key into this file or into a chat.

You never need to open `plan/testing/manual/milestone-5.md`. That is the agents' detailed version of these tests.

## Before you start (each time you test)

**Setup A — Open Terminal and go to the kirsch folder.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
```
- [✅] Done

**Setup B — Set up a throwaway test area.**
Paste:
```
source scripts/m5-test-env.sh
```
You should see `Ready.` followed by a few lines saying where your test files are, and `none of the four API key variables is set`.
This makes throwaway folders for the tests, clears any API keys in this Terminal window, and points kirsch at those folders instead of your real settings.
If you see `Not ready:` instead, do what that line says, then paste the setup line again.
- [✅] I saw Ready.

Opened a new Terminal window or tab? Do Setup A and Setup B again in that window.

## When you have finished testing

Paste:
```
m5_done
```
This forgets the test setup in this window. It also prints one line starting `rm -rf` that would delete the throwaway test folders. Pasting that line is optional. If you skip it, the folders stay on your Mac.

## Write down what you tested

| | |
|---|---|
| Date | |
| Which commit (paste the output of `git log --oneline -1`) | |
| Anything about your setup worth noting | |

## Sections

| Section | What it tests |
|---|---|
| 1 | The status bar on a narrow window |
| 2 | Version and help |
| 3 | Clearing approvals is saved |
| 4 | Terminal size and window resizing |
| 5 | The settings guide |
| 6 | Safety tests for running commands |
| 7 | The doctor command: its report and whether it says pass or fail |
| 8 | Pasting into the typing area, and the cursor |
| 9 | The build script and version check |
| 10 | Release settings and build check |
| 11 | Another safety test for running commands |
| 12 | The doctor command: tools and project checks |
| 13 | Wrapping long lines on screen |
| 14 | Paste delay |
| 15 | The automatic release steps on GitHub |
| 16 | A check that the settings guide lists every setting |
| 17 | The install guide |
| 18 | Fresh machine, no API key check |
| 19 | Starting a new session with `/new` (only if you choose to do it) |
| 20 | The doctor command: key, backend and model lines |
| 21 | Marker for cut text in the typing area |
| 22 | The usage guide corrections |
| 23 | The front page and progress tracker are current |
| 24 | The doctor command: state folder and lock |
| 25 | Project files (contributing, security, notices) and a note about reference screens |
| 26 | `/new` while busy (only if you choose to do it) |
| 27 | The troubleshooting guide and the list of docs pages |
| 28 | Resume totals and percent full (only if you choose to do it) |
| 29 | Record your decisions |
| 30 | The list of AI models (only if your decision calls for it) |
| 31 | The help footer and saved pictures of the screen (only if your decision calls for it) |
| 32 | The size limit on the debug log (only if your decision calls for it) |
| 33 | The notice about the debug log (only if your decision calls for it) |
| 34 | A library that makes unique IDs (only if your decision calls for it) |
| 35 | A written change to the specification (its section 7) (only if your decision calls for it) |
| 36 | What happens when two copies run at once (only if your decision calls for it) |
| 37 | Demo and final release check |

## If something does not match

Tell Bishop:
1. The section and step number.
2. The command you pasted.
3. What you saw (copy and paste it).
Never include an API key.

---

## 1 · The status bar on a narrow window (m5-d1)

**What this checks:** the bottom bar of kirsch on a narrow window. At 80 columns or wider the bar shows a token count (`0 tok` before any turn has run). Below 80 columns it shows no token count. Below 40 columns the main area is hidden and says `terminal too narrow`. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. No step moves you. kirsch starts from there and works on the practice project.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see two lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Run the automatic test for this piece.**
Paste:
```
go test -count=1 ./internal/tui -run TestSessionUsageNarrowLayoutFallsBackToSessionTotal
```
You should see one line starting with `ok` and ending with a time, like:
```
ok  	github.com/djm56/kirsch/internal/tui	1.2s
```
(The time will differ.)
- [✅] I saw `ok`

**Step 3 — Check that the window is at least 80 columns wide.**
Paste:
```
stty size
```
You should see two numbers: the rows, then the columns, for example `24 100`. The second number is the width. It must be 80 or more for step 4. If it is less, make the Terminal window wider and paste the line again.
- [✅] The second number is 80 or more

**Step 4 — Start kirsch on a wide window.**
Paste:
```
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
You should see this in the bar, the line with `·` between the words, just above the typing line:
```
minimax-m3 · idle · 0 tok
```
(`minimax-m3` is the model in the test setup. `0 tok` means no turn has run yet.)
Then type `/quit` and press Enter to close kirsch.
- [✅] I saw `minimax-m2.7 · idle · 0 tok`

**Step 5 — Start kirsch on a window narrower than 80 columns.**
Make the Terminal window narrower than 80 columns, but not narrower than 40. 60 columns is a good choice. Paste:
```
stty size
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
kirsch takes over the whole Terminal window, so while it is open you will not see the `stty size` output. The bar should read:
```
minimax-m3 · idle
```
There is no token count at all.
Then type `/quit` and press Enter to close kirsch. The `stty size` output comes back on screen after you quit. It should show a second number from 40 to 79 (for example `24 60`).
(Resize the window before you paste, so kirsch starts at the width you want. kirsch also redraws if you resize it while it is open.)
- [✅] I saw `minimax-m2.7 · idle` and no token count

**Step 6 — Start kirsch on a window narrower than 40 columns.**
Make the Terminal window narrower than 40 columns. 30 columns is a good choice. Paste the same commands as in step 5. kirsch takes over the window, so the `stty size` output comes back only after you quit. It should then show a second number below 40 (for example `24 30`).
You should see a dim line that says `terminal too narrow` in the main area, and the bar should still read:
```
minimax-m3 · idle
```
Then type `/quit` and press Enter to close kirsch.
- [✅] I saw `terminal too narrow`, and the bar reads `minimax-m2.7 · idle`

**Step 7 — Make the window wide again.**
Make the Terminal window 80 columns or wider. Paste the same commands as in step 5. After you quit kirsch, the `stty size` output should show a second number of 80 or more (for example `24 100`).
You should see the bar read `minimax-m3 · idle · 0 tok` again. Then type `/quit` and press Enter to close kirsch.
- [✅] I saw `minimax-m2.7 · idle · 0 tok` again

> Added by the crew 2026-10-29 (mission-20261029-01): the default model is now `minimax-m3`. The ticks above were made when it was `minimax-m2.7`. On a re-run the bar starts with `minimax-m3`.

**Anything odd?** Write it here:

_(nothing yet)_

---

## 2 · Version and help (m5-d2)

**What this checks:** kirsch can tell you its version and show its help screen, and it does both even when you are not inside a project folder.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Step 2 moves you into your test folder; Steps 3 to 7 work from there.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
go build -ldflags "-X main.version=9.9.9" -o bin/kirsch-999 ./cmd/kirsch
```
You should see two lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
(The second line builds kirsch. The third builds a test copy stamped with the made-up version 9.9.9, which step 3 uses.)
- [✅] Done, no errors

**Step 2 — Ask for the version, from outside any project.**
Paste:
```
cd "$T_CFG"
"$KIRSCH_BIN" --version
```
You should see:
```
kirsch 0.1.0-dev
```
If you are testing a release (your kirsch folder is exactly on a version tag like v0.1.0 with no changes), this line shows that version instead, for example `kirsch 0.1.0`. That also counts as a match.
(`$T_CFG` is your throwaway test folder. This step runs kirsch from there, not from inside the kirsch project.)
- [✅] I saw `kirsch 0.1.0-dev` (or the release version, as above)

**Step 3 — Ask the test copy for its version.**
Paste:
```
"${KIRSCH_BIN}-999" --version
```
You should see:
```
kirsch 9.9.9
```
There must be no `v` in front of 9.9.9.
- [✅] I saw `kirsch 9.9.9`

**Step 4 — Show the help screen.**
Paste:
```
"$KIRSCH_BIN" --help
```
You should see exactly:
```
Usage:
  kirsch [flags]
  kirsch resume [flags]

Commands:
  resume    resume the most recent session for this workspace

Flags:
  --workspace <path>  workspace root (defaults to the enclosing Git repository)
  --new               start a fresh session even when a previous session exists
  --debug             write a structured debug log
  --version           print the version and exit
  --help, -h          show this help and exit
```
- [✅] I saw the help screen

> **Note, 2026-10-28 (m5-d3a):** Since 2026-10-28 (m5-d3a) the help screen also shows a `kirsch doctor` line under Usage and a `doctor` line under Commands. Your tick above was for the help screen before that change. Section 7 checks the new lines.

**Step 5 — Check the help screen finishes cleanly.**
Paste:
```
"$KIRSCH_BIN" --help > /dev/null; echo "result: $?"
```
You should see:
```
result: 0
```
(0 means it finished without an error.)
- [✅] I saw `result: 0`

**Step 6 — The short form `-h` and the help for `resume` work too.**
Paste:
```
"$KIRSCH_BIN" -h > /dev/null; echo "result: $?"
"$KIRSCH_BIN" resume --help > /dev/null; echo "result: $?"
```
You should see:
```
result: 0
result: 0
```
- [✅] I saw `result: 0` twice

**Step 7 — A mistyped option is refused.**
Paste:
```
"$KIRSCH_BIN" --nope; echo "result: $?"
```
You should see the message `flag provided but not defined: -nope`, then the help screen, then `kirsch: flag provided but not defined: -nope` again, and last:
```
result: 1
```
(1 means kirsch refused the option. The message appearing twice is known. It is not part of this test.)
- [✅] I saw the refusal and `result: 1`

**Step 8 — Run the automatic tests for this piece.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./cmd/kirsch -run 'TestRunVersionOutsideRepository|TestRunHelp|TestRunBadFlag|TestResolveVersion|TestLdflagsVersionBuild'
```
You should see one line starting with `ok` and ending with a time, like:
```
ok  	github.com/djm56/kirsch/cmd/kirsch	1.2s
```
(The time will differ.)
- [✅] I saw `ok`

**Optional — tidy up the test copy.**
```
rm -f /Volumes/DATA/Github/kirsch/bin/kirsch-999
```

**Anything odd?** Write it here:

_(nothing yet)_

---

## 3 · Clearing approvals is saved (m5-d5)

**What this checks:** when you clear your approvals with `/approvals`, the clear is saved. After you quit and reopen kirsch, the approvals you cleared stay cleared. These steps run the tests that prove it. They do not start kirsch, so they need no key. The "quit and reopen kirsch" check uses kirsch for real, which needs a model turn. The crew did not find a way to do it without your API key, so it stays open for now. It is checked at the final release check (section 37 of this checklist).

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Every step runs from there.

**Step 1 — Run the saved-clear checks for the session file.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/session/ -run 'LoadGrants|RecordScopesCleared' -v
```
You should see:
```
=== RUN   TestLoadGrantsReadsScopeGrantedEvents
--- PASS: TestLoadGrantsReadsScopeGrantedEvents (0.02s)
=== RUN   TestLoadGrantsStopsAtTornLine
--- PASS: TestLoadGrantsStopsAtTornLine (0.01s)
=== RUN   TestLoadGrantsEmptyFile
--- PASS: TestLoadGrantsEmptyFile (0.00s)
=== RUN   TestLoadGrantsMissingFile
--- PASS: TestLoadGrantsMissingFile (0.00s)
=== RUN   TestLoadGrantsClearThenGrantKeepsOnlyLater
--- PASS: TestLoadGrantsClearThenGrantKeepsOnlyLater (0.01s)
=== RUN   TestLoadGrantsClearLeavesNone
--- PASS: TestLoadGrantsClearLeavesNone (0.01s)
=== RUN   TestLoadGrantsWithoutClearEventUnchanged
--- PASS: TestLoadGrantsWithoutClearEventUnchanged (0.01s)
=== RUN   TestRecordScopesClearedWritesEvent
--- PASS: TestRecordScopesClearedWritesEvent (0.01s)
PASS
ok  	github.com/djm56/kirsch/internal/session	0.223s
```
(The times will differ. Every test should say `PASS`, and the last line should start with `ok`.)
- [✅] I saw `ok` on the last line, and no `FAIL` anywhere

**Step 2 — Run the saved-clear checks for the app.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/app/ -run 'ClearGrants|ResolveSessionGrantThenClear' -v
```
You should see:
```
=== RUN   TestClearGrantsRoundTripAndEvent
--- PASS: TestClearGrantsRoundTripAndEvent (0.12s)
=== RUN   TestClearGrantsAfterResumeBeforeAnyTurn
--- PASS: TestClearGrantsAfterResumeBeforeAnyTurn (0.09s)
=== RUN   TestClearGrantsWithoutStoreCreatesNoSession
--- PASS: TestClearGrantsWithoutStoreCreatesNoSession (0.02s)
=== RUN   TestClearGrantsIsDurableWithoutClose
--- PASS: TestClearGrantsIsDurableWithoutClose (0.02s)
=== RUN   TestResolveSessionGrantThenClearNotRestored
--- PASS: TestResolveSessionGrantThenClearNotRestored (0.09s)
PASS
ok  	github.com/djm56/kirsch/internal/app	0.537s
```
(The times will differ.)
- [✅] I saw `ok` on the last line, and no `FAIL` anywhere

**Step 3 — Run the race checks.** This takes a few minutes.
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -race -count=1 ./internal/session/ ./internal/app/
```
You should see the `ok` line for each package, like these (the times will differ):
```
ok  	github.com/djm56/kirsch/internal/session	6.373s
ok  	github.com/djm56/kirsch/internal/app	98.164s
```
- [✅] I saw an `ok` line for each package, and no `FAIL` or `DATA RACE`

**Step 4 — Check the spec lists the new line.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep 'approval.scopes_cleared' plan/spec/kirsch-plan.md
```
You should see:
```
{"v":1,"type":"approval.scopes_cleared"}
Rules: single writer goroutine; buffered flush with `fsync` every ~500ms and on turn completion; on load, a corrupt line truncates the session there and marks it `recovered` (never block startup on one bad file); `kirsch resume` restores transcript, pending state, and session-scoped approval grants. A later `approval.scopes_cleared` cancels the grants recorded before it.
```
(Look for the line in the session-events list, and for the sentence about resume.)
- [✅] I saw the line in the session-events list, and the sentence about resume

**Anything odd?** Write it here:

_(nothing yet)_

---

## 4 · Terminal size and window resizing (m5-d9)

**What this checks:** when kirsch uses colour, and whether the help box stays on screen when you resize the window. Colour is turned off when `NO_COLOR` is set to something. An empty `NO_COLOR` (set, but with nothing after the `=`) does not turn colour off.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. No step moves you away from it.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see two lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Start kirsch with colour turned off.**
Paste:
```
script -q "$T_CFG/colour-off.log" env NO_COLOR=1 "$KIRSCH_BIN" --workspace "$WORKSPACE"
```
kirsch takes over the window. Type `/quit` and press Enter to close it. Then paste:
```
grep -a -o $'\033\[[0-9;:]*m' "$T_CFG/colour-off.log" | wc -l
```
You should see `0`. (The number counts the colour codes kirsch wrote to the screen. macOS may put spaces in front of the number.)
- [✅] I saw `0`

**Step 3 — Start kirsch with `NO_COLOR` set but empty.**
Paste:
```
script -q "$T_CFG/colour-on.log" env NO_COLOR= "$KIRSCH_BIN" --workspace "$WORKSPACE"
```
kirsch takes over the window. Type `/quit` and press Enter to close it. Then paste:
```
grep -a -o $'\033\[[0-9;:]*m' "$T_CFG/colour-on.log" | wc -l
```
You should see a number bigger than `0`. The exact number can differ. An empty `NO_COLOR` counts as not set, so kirsch uses colour.
- [✅] I saw a number bigger than `0`

**Step 4 — Open the help box, then resize the window.**
Make the Terminal window at least 100 columns wide and 30 rows tall before you start. Paste:
```
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
kirsch takes over the window. Press Tab once: the line at the bottom changes. Then press `?`.
You should see a box with `help` in its top border, words such as `composing` and `browsing` inside it, and `Esc or ? closes` at the bottom of it.
- [✅] I saw the box at full size

Now make the Terminal window narrower, to about 60 columns, and keep its height the same. The box should still be there, with `help` in its top border. At this width the help box is clipped, so some rows end in `⋯`, and that is normal.
- [✅] At about 60 columns, the box was still there and nothing stuck out past the right edge

Make the window wider again. The box should still be there.
- [✅] When I made the window wider again, the box was still there

Press Esc: the box closes. Press Esc again: the typing line comes back. Type `/quit` and press Enter.

**Step 5 — Run the automatic tests for this piece.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tui -run 'TestDetectCaps|TestFrameInvariants'
```
You should see one line starting with `ok` and ending with a time, like:
```
ok  	github.com/djm56/kirsch/internal/tui	1.4s
```
(The time will differ.)
- [✅] I saw `ok`

**Anything odd?** Write it here:

When I did step 2 the terminal line did some strange things when typing example, but I think this could be an anomally but you can see the issue below:
 ───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
 minimax-m2.7 · idle · 0 tok
 ───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
 > ▌
 > /▌
 > /q▌
 > /qu▌
 > /qui▌
 > /quit▌

---

## 5 · The settings guide (m5-d13a)

**What this checks:** the settings guide, a page in the `doc/` folder. It lists every setting kirsch reads, its normal value, and what it does. It also checks that the guide does not promise settings you type on the command line, and that a settings file with a name that looks like a token or password stops kirsch from starting. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. No step moves you away from it.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see two lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Look at the settings sections in the guide.**
Paste:
```
grep -n '^### ' doc/configuration.md
```
You should see one line for each of these headings, in this order: `[provider]`, `[policy]`, `[context]`, `[session]` and `[telemetry]`. Each line starts with a line number and a colon.
- [✅] I saw a heading for each settings section

**Step 3 — Check the guide does not use the words "milestone" or "ADR".**
Paste:
```
grep -n -i -w -E 'milestones?|ADRs?' doc/configuration.md
```
You should see nothing at all: no lines and no message. That means the guide has none of those words.
- [✅] I saw nothing

**Step 4 — A settings file with a token-like name stops kirsch.**
Paste this one line. It writes a throwaway settings file in a new folder, so your test settings are not changed:
```
( d=$(mktemp -d); mkdir -p "$d/kirsch"; printf '[provider]\nmax_tokens = 1\n' > "$d/kirsch/config.toml"; XDG_CONFIG_HOME="$d" "$KIRSCH_BIN" --workspace "$WORKSPACE" ; echo "result: $?"; rm -f "$d/kirsch/config.toml"; rmdir "$d/kirsch" "$d" )
```
kirsch does not open. You should see a line starting `kirsch:` that names a `config.toml` file and the key `provider.max_tokens`, followed by the words `looks like a credential`. Then a short paragraph starting `Kirsch never reads secrets from config files`, and last:
```
result: 1
```
(1 means kirsch refused to start.)
- [✅] I saw the message about `provider.max_tokens` and `result: 1`

**Anything odd?** Write it here:

_(nothing yet)_

---

## 6 · Safety tests for running commands (m5-d19a)

**What this checks:** the automatic safety tests for commands that kirsch runs. They check that a command gets no keyboard input, that a command which times out or is cancelled leaves no child process running, that secret-looking setting names such as `MY_TOKEN` are removed from what the command sees, and that a name written in lower case (such as `my_token`) is not removed. They also check the approval card for a request whose type was not stated, which kirsch shows like a file-change approval. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Each step starts there too.

**Step 1 — Run the command-safety tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tool -run 'TestRunCommandStdinIsDevNull|TestRunCommandAllStripPatterns|TestRunCommandLowerCaseSecretNamesPassThrough|TestRunCommandTimeoutProcessGroupDead|TestRunCommandCancellationProcessGroupDead' -v
```
You should see lines starting `=== RUN` and `--- PASS`, and no line starting `--- FAIL`. Some lines are indented. Those are the smaller checks inside one test, and they are fine. The whole run takes about a second or two, and that is normal. A first run that has to build first can take longer. The last line starts with `ok` and ends with a time, like:
```
ok  	github.com/djm56/kirsch/internal/tool	1.6s
```
(The time will differ.)

In plain words, these tests check this: a command gets no keyboard input; a command that times out or is cancelled leaves no child process running; secret-looking setting names (such as `MY_TOKEN` and `API_KEY`) are removed; and the same names written in lower case are kept, because the match is case-sensitive.
- [✅] I saw `ok` and no `--- FAIL` line

**Step 2 — Run the approval-card tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tui -run 'TestApprovalUnspecifiedCard' -v
```
You should see lines starting `--- PASS`, and no line starting `--- FAIL`. The last line starts with `ok` and ends with a time, like:
```
ok  	github.com/djm56/kirsch/internal/tui	0.3s
```
(The time will differ.)

A test with *known defect* in its name passes on purpose. It records that the approval card for a request whose type was not stated still shows blank rows and a `[d] view diff` action with no diff behind it. Kirsch shows that card like a file-change approval, so it offers `[d] view diff`. That is a problem in the program, not in the test. It is written up in `plan/PROGRESS.md` under *Carried from Milestone 5 planning*, and it is not yet known whether you can reach that card in normal use. The test is meant to fail once the card is fixed; that is the time to update it.
- [✅] I saw `ok` and no `--- FAIL` line

**Anything odd?** Write it here:

_(nothing yet)_

---

## 7 · The doctor command: its report and whether it says pass or fail (m5-d3a)

**What this checks:** the `kirsch doctor` command. It prints the version, a line for each check, and a summary line. It still prints that report outside a project folder, and when the settings file is broken. It exits 0 unless a check fails. Nothing here calls a model or uses an API key.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. Step 4 moves you into your test folder, and Step 5 creates its own throwaway settings folder.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see lines starting with `>`, like these (that is npm saying what it is doing):
```
> kirsch@0.1.0-dev build
> bash scripts/build.sh
```
Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Ask for the report from inside the project.**
Paste:
```
"$KIRSCH_BIN" doctor
```
You should see:
```
kirsch 0.1.0-dev
doctor: 0 ok, 0 warn, 0 fail
```
The line starting `kirsch` is the version. The line starting `doctor:` is the summary: it counts the checks that passed, warned or failed. This section's version of doctor has no checks, so every count is 0. If your kirsch folder is exactly on a release tag with no changes, the version line shows that tag instead, as in section 2.
- [✅] I saw the version line and the summary line

**Step 3 — Show the result code.**
Paste:
```
"$KIRSCH_BIN" doctor > /dev/null; echo "result: $?"
```
You should see:
```
result: 0
```
(0 means doctor finished and no check failed. The `> /dev/null` in this step hides the report, which Step 2 already showed.)
- [✅] I saw `result: 0`

**Step 4 — Ask for the report from the test folder, outside the project.**
Paste:
```
cd "$T_CFG"
"$KIRSCH_BIN" doctor; echo "result: $?"
```
You should see:
```
kirsch 0.1.0-dev
doctor: 0 ok, 0 warn, 0 fail
result: 0
```
(`$T_CFG` is your throwaway test folder. It is not a Git project, and doctor still reports.)
- [✅] I saw the same report and `result: 0` outside the project

**Step 5 — Run it with a broken settings file.**
Paste this line. It writes a broken settings file in a new throwaway folder, so your test settings are not changed, and it removes that folder when it finishes:
```
( d=$(mktemp -d); mkdir -p "$d/kirsch"; printf '[session\nstorage_dir = "%s"\n' "$T_STATE/sessions" > "$d/kirsch/config.toml"; XDG_CONFIG_HOME="$d" "$KIRSCH_BIN" doctor; echo "result: $?"; XDG_CONFIG_HOME="$d" "$KIRSCH_BIN" --workspace "$WORKSPACE"; echo "result: $?"; rm -f "$d/kirsch/config.toml"; rmdir "$d/kirsch" "$d" )
```
You should see the report again, then `result: 0`. Then a line starting `kirsch: parse` that names a `config.toml` file and `toml: line 2`, then `result: 1`. The folder name in that line will be different.
(Doctor still reports, because it does not read the settings file. Plain kirsch refuses to start, and `result: 1` shows that it did.)
- [✅] Doctor still reported, and plain kirsch refused with `result: 1`

**Step 6 — See the doctor lines in the help screen.**
Paste:
```
"$KIRSCH_BIN" --help | grep -n doctor
```
You should see:
```
4:  kirsch doctor
8:  doctor    check the installation and report problems
```
- [✅] I saw the doctor line under Usage and the doctor line under Commands

**Step 7 — Run the automatic tests for this piece.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./cmd/kirsch -run 'Doctor|TestRunHelp'
go test -count=1 ./internal/doctor ./internal/arch
```
You should see lines starting with `ok`, each ending with a time, like these:
```
ok  	github.com/djm56/kirsch/cmd/kirsch	0.293s
ok  	github.com/djm56/kirsch/internal/doctor	0.257s
ok  	github.com/djm56/kirsch/internal/arch	1.601s
```
(The times will differ.)
- [✅] I saw `ok` on each of those lines

**Anything odd?** Write it here:

_(nothing yet)_

---

## 8 · Pasting into the typing area, and the cursor (m5-d6)

**What this checks:** text you paste into kirsch's typing area lands as you copied it, except that tabs turn into spaces, and the typing area follows the cursor when the text is longer than the area.

**Before this section:** do *Before you start* in this Terminal window. Setup must have said `Ready.`.

Step 1 runs the automatic tests in your kirsch folder. Steps 2 to 6 are the paste check. Do steps 3 to 6 in Terminal.app, then again in iTerm2.

**Step 1 — Run the automatic tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tui/ -run 'TestPasteLandsVerbatim|TestLargePasteConfirmAcceptAndDecline|TestMultiLinePasteStartingWithSlashIsNotACommand|TestComposerScrollsToTheCaret|TestComposerWindowFollowsTheCaretLineByLine|TestComposerWindowStaysAtTheTopWithoutACaret|TestComposerHintYieldsToAFullWindow' -v
```
You should see a line starting `--- PASS:` for each test name, then a last line starting `ok`. A line with `FAIL` means something is wrong.
- [✅] I saw a `--- PASS:` line for each of the test names listed above, `ok` at the end, and no `FAIL`

**Step 2 — Make a 20-line sample to copy.**
Paste:
```
for i in {1..20}; do printf '  trace %02d\n' "$i"; done
```
You should see 20 lines, from `  trace 01` to `  trace 20`. Each line starts with 2 spaces. Copy the 20 lines: drag across them, then press Cmd-C.
- [✅] I copied the 20 lines

**Step 3 — Start kirsch in this window.**
Paste:
```
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
You should see kirsch's screen, with the typing area near the bottom. kirsch calls no model unless you send a message, and you will not send a message in this section.
- [✅]] I saw kirsch's screen

**Step 4 — Paste the 20 lines.** *The crew could not run this step; it needs your real Terminal.*
The typing area is already active. Press Cmd-V, and do not press Enter.
You should see the typing area show the last lines you pasted, with the cursor after the last line, on the bottom row. Nothing has been sent: no message from you appears above the typing area.
Your copy may end with a blank line. The bottom row may then be empty with the cursor on it and four trace lines above. That is fine.
If your pasted text used tabs, they show as spaces. That is known and waiting for your decision.
If kirsch stops responding after the paste (keys do nothing), close this Terminal tab or window, then write what you saw under *Anything odd?* and tell Bishop. Nothing in this step touches a model or your files.
- [✅] The typing area shows the last lines I pasted, with the cursor after the last line
- [✅] The pasted lines are on separate lines, not joined together
- [✅] Nothing was sent: no message from me appeared above the typing area

**Step 5 — Move up with the arrow key.**
Press the Up arrow key again and again, a press at a time, until the top line shows.
You should see the lines in the typing area move up a line for each press, with the cursor staying on the bottom row. The `>` at the left appears once the top line is visible. Stop pressing when the `>` appears at the left of the top line. If you press again, the cursor moves up inside the typing area instead of the lines moving.
- [✅] The lines moved up a line at a time and the cursor stayed on the bottom row

**Step 6 — Clear the typing area and quit.**
Press Esc once. The typing area should empty, and the `>` should show at the left.
Then type `/quit` and press Enter.
Only type `/quit` when the typing area is empty. If the pasted text is still there, Enter sends it to kirsch as a message, and kirsch would try to reach a model. That is not wanted in this section. If it happens, stop and tell the crew.
You should see your Terminal prompt again.
- [✅] The typing area was empty after Esc
- [✅] `/quit` closed kirsch, and I am back at my Terminal prompt

**Anything odd?** Write it here:

_(nothing yet)_

---

## 9 · The build script and version check (m5-d10a)

**What this checks:** the build script, `scripts/build.sh`. It builds kirsch and can put a version number into it. You will check that the number you give it is the one kirsch shows, that a leading `v` is dropped, that a badly shaped number is refused, and that a normal build with no number shows `0.1.0-dev`. No kirsch session starts, no model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. Every step works from there. Step 4 is the last build in this section. It leaves kirsch built normally, with no version number, which the other sections expect. Step 5 checks that build.

**Step 1 — Build kirsch with the version 9.9.9.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
bash scripts/build.sh 9.9.9
bin/kirsch --version
```
You should see:
```
kirsch 9.9.9
```
There must be no `v` in front of 9.9.9. Lines starting `go: downloading` are also fine. When the build works, it prints nothing else.
- [✅] I saw `kirsch 9.9.9`

**Step 2 — A leading `v` is dropped.**
Paste:
```
bash scripts/build.sh v9.9.8
bin/kirsch --version
```
You should see:
```
kirsch 9.9.8
```
You typed `v9.9.8`, and kirsch shows `9.9.8`, with no `v`.
- [✅] I saw `kirsch 9.9.8`

**Step 3 — A badly shaped version is refused.**
A version needs three numbers, such as `1.2.3`. Paste:
```
bash scripts/build.sh 1.2; echo "result: $?"
```
You should see this line:
```
build.sh: version argument '1.2' is not X.Y.Z[-suffix|+suffix]
```
and then:
```
result: 2
```
(2 means the script refused the number and did not build. The kirsch from step 2 is still there.)
- [✅] I saw the refusal line and `result: 2`

**Step 4 — Build kirsch normally, with no version number.**
Paste:
```
npm run build
"$KIRSCH_BIN" --version
```
You should see two lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong. Then you should see:
```
kirsch 0.1.0-dev
```
(With no version number, kirsch shows a release version only when your kirsch folder is exactly on a version tag like v0.1.0 with no changes at all. Then this line shows that version instead, for example `kirsch 0.1.0`. That also counts as a match.)
- [✅] I saw `kirsch 0.1.0-dev` (or the release version, as above)

**Step 5 — Check that this build has no version built into it.**
Paste:
```
go version -m bin/kirsch | grep -c -- -ldflags
```
You should see `0` if step 4 showed `kirsch 0.1.0-dev`. You should see `1` if step 4 showed a release version such as `kirsch 0.1.0`. On a release build the script stamps the version into the binary, and that stamp appears as a `-ldflags` line, so the count is 1.
(0 means no version number was built into this kirsch. That is the expected result when step 4 showed `kirsch 0.1.0-dev`.)
- [✅] I saw `0`, or `1` after a release version (as above)

**Anything odd?** Write it here:

_(nothing yet)_

---

## 10 · Release settings and build check (m5-d10b)

**What this checks:** the release settings file, `.goreleaser.yaml`, and a check that every release target builds. The check builds kirsch for each target and prints a line for each target. No kirsch session starts, no model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Every step works from there.

**Step 1 — Check that every release target builds.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run ci:targets
```
You should see a line for each release target, each ending in `: ok`, then a line saying all the targets compile:
```
target darwin/amd64: ok
target darwin/arm64: ok
target linux/amd64: ok
target linux/arm64: ok
ci.sh: all 4 targets compile
```
Lines starting `>` (that is npm echoing the command it runs) and lines starting `go version` or `ci.sh: GOTOOLCHAIN` are also fine. A line saying `FAILED` means a target did not build. Tell Bishop.
- [✅] I saw a `: ok` line for each target, and the line saying all the targets compile

**Step 2 — See the result code.**
Paste:
```
npm run ci:targets; echo "result: $?"
```
You should see this as the last line:
```
result: 0
```
(0 means the check passed. Any other number means it failed.)
- [✅] I saw `result: 0`

**Step 3 — Look at the release settings.**
Paste:
```
grep -E "^ +main:" .goreleaser.yaml
```
You should see this line:
```
    main: ./cmd/kirsch
```
This says the release builds the program in `cmd/kirsch`. A check looks for any other build, but it does not catch every way of writing one.
- [✅] I saw `main: ./cmd/kirsch`

Building the real release files needs a tool called goreleaser. Installing it is part of the final release check (section 37). It was not installed on this machine when this section was written on 2026-10-28, so that check waits for that step.

**Anything odd?** Write it here:

_(nothing yet)_

---

## 11 · Another safety test for running commands (m5-d19b)

Not ready yet. This section will be written in the same step-by-step style when m5-d19b is finished.

---

## 12 · The doctor command: tools and project checks (m5-d3b)

Not ready yet. This section will be written in the same step-by-step style when m5-d3b is finished.

---

## 13 · Wrapping long lines on screen (m5-d7a)

Not ready yet. This section will be written in the same step-by-step style when m5-d7a is finished.

---

## 14 · Paste delay (m5-d8)

Not ready yet. This section will be written in the same step-by-step style when m5-d8 is finished.

---

## 15 · The automatic release steps on GitHub (m5-d11)

Not ready yet. This section will be written in the same step-by-step style when m5-d11 is finished.

---

## 16 · A check that the settings guide lists every setting (m5-d13b)

**What this checks:** the settings guide, `doc/configuration.md`, lists every setting kirsch reads. A test fails if a setting is missing from the guide. This section runs only tests. No kirsch program starts, no model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Step 1 runs from there.

**Step 1 — Run the settings guide check.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/config -run KeyCoverage -v
```
You should see:
```
=== RUN   TestKeyCoverage_DerivedListsAreSane
--- PASS: TestKeyCoverage_DerivedListsAreSane (0.00s)
=== RUN   TestKeyCoverage_PageDocumentsEveryKey
--- PASS: TestKeyCoverage_PageDocumentsEveryKey (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/config	0.4s
```
(Both `--- PASS` lines and `ok` must appear. The `=== RUN` lines and the bare `PASS` line are printed as well, as shown above. The times will differ.)
If a line says `FAIL` and names a setting, the guide is missing that setting. Tell Bishop the name.
- [✅] I saw both `--- PASS` lines, the bare `PASS` line and `ok`, with no `FAIL`

**Anything odd?** Write it here:

_(nothing yet)_

---

## 17 · The install guide (m5-d13c)

Not ready yet. This section will be written in the same step-by-step style when m5-d13c is finished.

---

## 18 · Fresh machine, no API key check (m5-d16)

Not ready yet. This section will be written in the same step-by-step style when m5-d16 is finished.

---

## 19 · Starting a new session with `/new` (only if you choose to do it) (m5-d17a)

Not ready yet. This section will be written in the same step-by-step style when m5-d17a is finished.

---

## 20 · The doctor command: key, backend and model lines (m5-d4a)

Not ready yet. This section will be written in the same step-by-step style when m5-d4a is finished.

---

## 21 · Marker for cut text in the typing area (m5-d7b)

Not ready yet. This section will be written in the same step-by-step style when m5-d7b is finished.

---

## 22 · The usage guide corrections (m5-d14a)

Not ready yet. This section will be written in the same step-by-step style when m5-d14a is finished.

---

## 23 · The front page and progress tracker are current (m5-d15)

Not ready yet. This section will be written in the same step-by-step style when m5-d15 is finished.

---

## 24 · The doctor command: state folder and lock (m5-d4b)

Not ready yet. This section will be written in the same step-by-step style when m5-d4b is finished.

---

## 25 · Project files (contributing, security, notices) and a note about reference screens (m5-d12)

Not ready yet. This section will be written in the same step-by-step style when m5-d12 is finished.

---

## 26 · `/new` while busy (only if you choose to do it) (m5-d17b)

Not ready yet. This section will be written in the same step-by-step style when m5-d17b is finished.

---

## 27 · The troubleshooting guide and the list of docs pages (m5-d14b)

Not ready yet. This section will be written in the same step-by-step style when m5-d14b is finished.

---

## 28 · Resume totals and percent full (only if you choose to do it) (m5-d18)

Not ready yet. This section will be written in the same step-by-step style when m5-d18 is finished.

---

## 29 · Record your decisions (m5-d20)

Not ready yet. This section will be written in the same step-by-step style when m5-d20 is finished.

---

## 30 · The list of AI models (only if your decision calls for it) (m5-d21a)

Not ready yet. This section will be written in the same step-by-step style when m5-d21a is finished.

---

## 31 · The help footer and saved pictures of the screen (only if your decision calls for it) (m5-d21b)

Not ready yet. This section will be written in the same step-by-step style when m5-d21b is finished.

---

## 32 · The size limit on the debug log (only if your decision calls for it) (m5-d21c)

Not ready yet. This section will be written in the same step-by-step style when m5-d21c is finished.

---

## 33 · The notice about the debug log (only if your decision calls for it) (m5-d21d)

Not ready yet. This section will be written in the same step-by-step style when m5-d21d is finished.

---

## 34 · A library that makes unique IDs (only if your decision calls for it) (m5-d21e)

Not ready yet. This section will be written in the same step-by-step style when m5-d21e is finished.

---

## 35 · A written change to the specification (its section 7) (only if your decision calls for it) (m5-d21f)

Not ready yet. This section will be written in the same step-by-step style when m5-d21f is finished.

---

## 36 · What happens when two copies run at once (only if your decision calls for it) (m5-d21g)

Not ready yet. This section will be written in the same step-by-step style when m5-d21g is finished.

---

## 37 · Demo and final release check (m5-d22)

Not ready yet. This section will be written in the same step-by-step style when m5-d22 is finished.
