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
| 20 | The doctor command: key, backend, model and project notes lines |
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

> **Note, 2026-10-29 (m5-d3b):** The usage line now reads `  kirsch doctor [flags]`, because m5-d3b added `[flags]`. The help text quoted in Step 4 is the m5-d2 text, as the note above says. Your tick above stands for the help screen you tested before this change.

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

> **Note, 2026-10-29 (m5-d3b):** Doctor now runs checks for git, rg, the repository and the settings file, so some lines in this section have changed. The ticks above stand for what you tested before this change, and they are not edited.
> - Step 2 and Step 4: the report now has check lines for git, rg, repository and config, before the summary. The sentence "every count is 0" is the m5-d3a text. On a Mac with git and rg installed, inside the kirsch folder, Step 2 ends `doctor: 4 ok, 0 warn, 0 fail`.
> - Step 4 is outside any project, so the repository check now fails. It prints `fail  repository: not in a Git repository`, the summary reads `doctor: 3 ok, 0 warn, 1 fail`, and the result is `result: 1`. That is correct: without `--workspace`, kirsch does not start outside a repository.
> - Step 5: the first doctor run now prints a `fail` line for config, and its result is `result: 1`, not `result: 0`. The plain kirsch run after it still refuses to start, with `result: 1`.
> - Step 4 leaves you in `$T_CFG`, which is outside any project, so Step 5 runs outside a repository too. The report therefore also has a repository fail, and it has two fails in all. Run from that folder on 2026-10-29, the first doctor command printed this, with the folder path shortened and the git and rg paths as they were on that Mac:
>
>   ```text
>   kirsch 0.1.0-dev
>   ok  git: found at /opt/local/bin/git
>   ok  rg: found at /opt/homebrew/bin/rg
>   fail  repository: not in a Git repository
>   fail  config: parse …/kirsch/config.toml: toml: line 2: expected '.' or ']' to end table name, but got '\n' instead
>   doctor: 2 ok, 0 warn, 2 fail
>   result: 1
>   ```
>
>   The plain run after it printed `kirsch: parse …/kirsch/config.toml: toml: line 2: expected '.' or ']' to end table name, but got '\n' instead`, then `result: 1`. That run left its cleanup line out, so the throwaway folder it made is still there. Your own run will show the same lines, with different folder names.
> - Step 6: the usage line now reads `  kirsch doctor [flags]`, so the first line the grep prints is `4:  kirsch doctor [flags]`.
> - `--workspace` given with `doctor` now applies. Section 12 shows it.
> - **Dated note, 2026-10-29 (m5-d4a):** since m5-d4a, `kirsch doctor` also prints `key`, `backend`, `model` and `context` lines after the config line, and the summary counts them. So the summary lines in the samples above, and the summaries in Steps 2 and 4, are out of date. The check lines and the results in this section are unchanged. Measured on 2026-10-29 from the kirsch folder, with no key set and an empty settings folder, Step 2 now ends `doctor: 7 ok, 1 warn, 0 fail`, and its one warning is the `key` line. Measured outside a repository with no flag, the summary reads `doctor: 5 ok, 2 warn, 1 fail`, and the result is `result: 1`. Your own settings may name a model kirsch does not list, and that adds one warning. The runs are in `.claude/memory/workspace/m20261029-06-step3b-runs.txt` (parts 4 and 6, scratch, archived at the next mission start).
> - **Dated note, 2026-10-30 (m5-d4b, step 4e):** since m5-d4b the summary also counts the `state` and `lock` lines, Measured at step 4c on the current code (`.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`), with `--workspace` set to the kirsch folder, no key set, a scratch settings folder and a writable state folder (run B; that run was started from a scratch folder), Step 2 now ends `doctor: 9 ok, 1 warn, 0 fail`. Measured outside a repository with no flag (run E), Step 4's summary reads `doctor: 5 ok, 4 warn, 1 fail`, and the result is `result: 1`. The `doctor: 0 ok` samples in Steps 2 and 4 are the m5-d3a text and are kept as recorded. The Step 5 sample (`doctor: 2 ok, 0 warn, 2 fail`, printed on 2026-10-29) is out of date for the same reason, and it was not measured again at step 4e, so no current figure is given for it.

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

**What this checks:** two timing tests for commands that kirsch runs. One checks that a command which stops when asked ends about at its time limit. The other checks that a command which ignores the request is still forced to stop within the allowed time. Step 2 looks at the milestone plan's boxes that changed, and step 3 is a decision for you. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Each step starts in the kirsch folder.

**Step 1 — Run the two timing tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tool -run 'TestRunCommandTerminateThenKill$|TestRunCommandTimeoutWithinDeadline$' -v; echo "exit=$?"
```
You should see lines starting `=== RUN` and `--- PASS`, and no line starting `--- FAIL`. The last line is `exit=0`. Here is the output of a run on 2026-11-05. Your times will differ a little, and that is normal:
```
=== RUN   TestRunCommandTimeoutWithinDeadline
--- PASS: TestRunCommandTimeoutWithinDeadline (1.02s)
=== RUN   TestRunCommandTerminateThenKill
--- PASS: TestRunCommandTerminateThenKill (3.02s)
PASS
ok  	github.com/djm56/kirsch/internal/tool	4.241s
exit=0
```
In plain words, the two PASS lines mean this:
- `TestRunCommandTimeoutWithinDeadline`: a command that stops when asked ends about at its time limit, which is 1 second in this test. The test accepts anything from 1 to 2 seconds.
- `TestRunCommandTerminateThenKill`: a command that ignores the request is forced to stop. The call ends about 2 seconds after the 1-second limit (3 seconds after the start, as the output shows). The test accepts up to 3 seconds after the limit, and fails above that.

A `--- FAIL` line, or an `exit=` value other than 0, means the check did not pass. Copy the last lines and tell Bishop.
- [✅] I saw `--- PASS` for both tests, the `ok` line, and `exit=0` as the last line

**Step 2 — Look at the three boxes that changed.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
sed -n '/Command timeout ends the call/,/^- \[x\] Env filtering/p' plan/milestones/milestone-2.md
```
You should see three boxes, each starting `- [ ]`, which means unticked. The last line of the output is one already-ticked box, and it only marks where the printout stops. Here is the output of that command, run after the edits in this milestone:
```
- [ ] Command timeout ends the call within 3s of the deadline, even when the
      command ignores SIGTERM. At the deadline SIGTERM goes to the process
      group; if the group leader is still alive 2s later, SIGKILL goes to the
      group. This box does not assert that every process in the group is gone
      when a process ignores SIGTERM: no test checks that, and the code may
      skip SIGKILL once the leader has exited (suspected, read from
      `internal/tool/run_command.go`, not run). (was: "Command timeout kills
      the process group within 500ms of the deadline", unticked; relaxed by
      O15, C29, 2026-10-29; the 3s value is in plan/PROGRESS.md row C29
      detail)
- [ ] A cancelled command's process is actually gone, not merely detached
      Note (2026-11-05, m5-d19b): m5-d19a (mission-20261027-05) made
      `TestRunCommandCancellationProcessGroupDead` record the PIDs out of band
      and assert they are gone (ESRCH). The test fails when `Setpgid` is
      set to `false` ("still alive 3s after the command ended",
      plan/testing/manual/milestone-5.md section 6). That meets the condition
      the status note below sets for ticking. Ticking is the operator's call.
- [ ] A command that reads input gets end-of-input at once and cannot
      wait for typing
      (was: "A command reading stdin fails immediately", ticked; reworded
      by C34, 2026-11-03)
- [x] Env filtering strips `*_KEY`/`*_TOKEN` even when allowlisted
```
Next to that output, the three boxes say the following:
- The first box is the timeout box. Its text now says `within 3s of the deadline`, and the old `500ms` wording is kept after `(was:`.
- The second box is the cancelled-process box. Its note says the test meets the condition for ticking, and that ticking is the operator's call.
- The third box is the stdin box. It has the reworded text, and the old wording is kept after `(was:`.

- [✅] I saw the three boxes, each unticked, and the timeout box says `within 3s of the deadline`

**Step 3 — A decision for you, not a check.**
Decide whether to tick the box "A cancelled command's process is actually gone, not merely detached", in `plan/milestones/milestone-2.md`. The test behind it is `TestRunCommandCancellationProcessGroupDead`, which you ran in section 6 of this checklist. When it was checked with the code broken on purpose (Bishop's crew turned process groups off), it failed with the line below; that run is recorded in the table in section 6 of `plan/testing/manual/milestone-5.md`. Ticking the box is your decision. Bishop has not ticked it, and Bishop will not tick it for you. Write your decision under *Anything odd?* at the end of this section.

```
run_command_test.go:963: still alive 3s after the command ended: grandchild sleep(pid 83373)
```

One gap is not covered by any test, and this section does not check it. A process that ignores the stop request may survive when the process it started exits on that request. It is written up in `plan/PROGRESS.md` under *Open from m5-d19b*, and fixing it needs a code change.

**Anything odd?** Write it here:

Yes Tick the box

---

## 12 · The doctor command: tools and project checks (m5-d3b)

**What this checks:** the `kirsch doctor` command now checks the two tools kirsch uses (git and rg), the folder kirsch works in, and your settings file. Each check prints one line. A line that says `fail` means kirsch will not start. A line that says `warn` means kirsch starts, but some things are switched off. Nothing here calls a model or uses an API key.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. Some steps below make throwaway folders. A tidy-up line at the end of this section removes them, or you can leave them for later.

> **Note, 2026-10-29 (m5-d4a):** Since m5-d4a, `kirsch doctor` prints more lines after the config line: `key`, `backend`, `model` and `context`. The summary line counts them too, so the summaries written in steps 2 to 6 below are out of date and read differently now. The check lines, the `repository` and `config` results, and the exit statuses those steps name are unchanged. Here is what step 2 prints now, from the kirsch folder, with no API key set. Paths differ on your Mac. The `context` size is the size of `AGENTS.md` on 2026-10-29, and it changes when that file changes. The model line names the default model, `minimax-m3`, because this sample was taken with an empty settings folder; your own settings may name another model.
> ```
> kirsch 0.1.0-dev
> ok  git: found at /opt/local/bin/git
> ok  rg: found at /opt/homebrew/bin/rg
> ok  repository: /Volumes/DATA/Github/kirsch
> ok  config: settings valid
> warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
> ok  backend: search_code uses ripgrep
> ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
> ok  context: AGENTS.md, 6687 bytes
> doctor: 7 ok, 1 warn, 0 fail
> exit=0
> ```
> Run steps 2 to 6 as written. The `key` warning is expected when no key is set. It is a warning, not a failure, and the run still ends `exit=0`. Write what you see under *Anything odd?* if a check line starts `fail` where this section says it should not, or if an exit status differs from the one a step names.
>
> **Dated note, 2026-10-30 (m5-d4b, step 4c):** the report now ends with two more lines, `state` and `lock`, and the summary counts them. So the `doctor: 7 ok, 1 warn, 0 fail` sample above is out of date. Measured on 2026-10-30 with `--workspace` set to the kirsch folder, a scratch settings folder, no key set, `rg` on PATH and a writable state folder, the run reads `doctor: 9 ok, 1 warn, 0 fail` and `exit=0` (`.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`, runs A, B and D). The count depends on the key (a warning when none is set), `rg` on PATH, the state folder (a `fail` there makes the run `exit=1`) and the lock.

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
cd /Volumes/DATA/Github/kirsch
"$KIRSCH_BIN" doctor; echo "exit=$?"
```
You should see the version line, then the check lines for `git`, `rg`, `repository` and `config`, then a summary line, then `exit=0`. The check lines start with `ok`, in this order: `git`, `rg`, `repository`, `config`. The repository line shows your kirsch folder. The summary reads `doctor: 4 ok, 0 warn, 0 fail`, as it did before m5-d4a. Since m5-d4a it reads `doctor: 7 ok, 1 warn, 0 fail` with no key set (measured 2026-10-29, see the note above). **Dated note, 2026-10-30 (m5-d4b, step 4c):** since m5-d4b it reads `doctor: 9 ok, 1 warn, 0 fail` with no key set and a writable state folder, because the `state` and `lock` lines count too (measured in `.claude/memory/workspace/m20261030-01-step4c-doctor-run.txt`, run B, with a scratch settings folder).
If rg is not installed on your Mac, its line starts with `warn` instead. The summary then reads `3 ok, 1 warn`, and `exit=0` still. A warning does not make doctor fail.
> **Dated note, 2026-10-29 (m5-d4a, step 3c):** the `3 ok, 1 warn` above is out of date. With `rg` off PATH, the `rg` line warns and the `backend` line warns too, and the summary counts both. Measured from the kirsch folder with no key set, under a PATH that holds only `git`, the run prints `warn  rg: not found on PATH; search_code uses the pure-Go fallback` and `warn  backend: search_code uses pure-go (rg is not on PATH)`. The summary reads `doctor: 5 ok, 3 warn, 0 fail`, and the run ends `exit=0`. The `key` warning is there because no key is set in that run. A warning still does not make doctor fail. The runs are in `.claude/memory/workspace/m20261029-06-step3c-runs.txt`, parts 7 and 8 (scratch, archived at the next mission start). **Dated note, 2026-10-30 (m5-d4b, step 4c):** since m5-d4b the summary also counts the `state` and `lock` lines, so the `doctor: 5 ok, 3 warn, 0 fail` above is out of date too. Those two lines were not measured in that run, and this PATH was not measured again at step 4e, so no current figure is given for it.
- [✅] I saw the check lines, the summary, and `exit=0`

**Step 3 — A folder outside any project, with `--workspace`.**
Paste:
```
d3=$(mktemp -d)
cd "$d3"
git -C "$d3" rev-parse --is-inside-work-tree; echo "exit=$?"
"$KIRSCH_BIN" doctor --workspace "$d3"; echo "exit=$?"
cd /Volumes/DATA/Github/kirsch
```
You should see `fatal: not a git repository` and `exit=128`. That shows the new folder is not inside a Git project. Then the report: the git and rg lines start with `ok`, and the repository line starts with `warn`. The repository line names your new folder and says it is not a Git repository. It also says kirsch starts, but the Git features are off, such as `/diff` and the branch display. The config line starts with `ok`. The summary reads one warning and no failures, and it ends with `exit=0`. The `--workspace` value tells kirsch which folder to check.
If rg is not installed on your Mac, the rg line starts with `warn` instead. The summary then reads two warnings and no failures, and `exit=0` still. This step's folder is named `$d3`, and the tidy-up line at the end of this section removes it.
- [✅] I saw the `warn` line for repository and `exit=0`

**Step 4 — The same kind of folder, with no `--workspace`.**
Paste:
```
d4=$(mktemp -d)
cd "$d4"
"$KIRSCH_BIN" doctor; echo "exit=$?"
cd /Volumes/DATA/Github/kirsch
```
You should see the repository line start with `fail` and say `not in a Git repository`. The summary reads one failure, and it ends with `exit=1`. Without `--workspace`, kirsch needs a Git project to start, so this is a failure. Step 3 and step 4 check each other: the folder is the same kind, and the only difference is the `--workspace` value. This step's folder is named `$d4`, and the tidy-up line at the end of this section removes it.
- [✅] I saw the `fail` line for repository and `exit=1`

**Step 5 — A settings file that does not read.**
Paste:
```
c5=$(mktemp -d)
mkdir -p "$c5/kirsch"
printf '[session\n' > "$c5/kirsch/config.toml"
cd /Volumes/DATA/Github/kirsch
XDG_CONFIG_HOME="$c5" "$KIRSCH_BIN" doctor; echo "exit=$?"
XDG_CONFIG_HOME="$c5" "$KIRSCH_BIN" --workspace /Volumes/DATA/Github/kirsch; echo "exit=$?"
```
You should see the report. The git, rg and repository lines start with `ok`. The config line starts with `fail  config: parse`, names a `config.toml` file, and includes `toml: line 2`. The summary reads one failure, and it ends with `exit=1`. The report still prints every line, because doctor reports the broken file and carries on.
Then the second command, which is kirsch itself with the same file, prints a line starting `kirsch: parse` that names the same file. It ends with `exit=1`. Kirsch refuses to start with this file, and doctor told you why. This step's folder is named `$c5`, and the tidy-up line at the end of this section removes it.
- [✅] Doctor printed a `fail` line for config, and plain kirsch also gave `exit=1`

**Step 6 — A settings file with a key kirsch does not know.**
Paste:
```
c6=$(mktemp -d)
mkdir -p "$c6/kirsch"
printf 'bogus_setting = 1\n' > "$c6/kirsch/config.toml"
cd /Volumes/DATA/Github/kirsch
XDG_CONFIG_HOME="$c6" "$KIRSCH_BIN" doctor; echo "exit=$?"
```
You should see a line starting `warn  config:`. It names the file and says `unknown key "bogus_setting" (ignored)`. The next config line starts with `ok` and says `settings valid`. The summary reads one warning and no failures, and it ends with `exit=0`. The unknown key is ignored, so kirsch still starts. This step's folder is named `$c6`, and the tidy-up line at the end of this section removes it.
- [✅] I saw the `warn` line for config and `exit=0`

**Step 7 — Run the automatic tests for this piece.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -list '^(TestRunDoctor.*|TestDoctorBinaryExitStatus|TestRun(RGMissing|GitMissing|NonGitWorkspaceVersusNoFlag|NotInRepositoryRunsOtherChecks|WorkspaceMissingDirectory|BrokenGlobalConfig|BrokenProjectConfig|InvalidConfigValues|UnknownConfigKey|ProjectConfigIgnoredKeyWarns|ConfigLoadedRecord|MultiLineConfigErrorStaysOnOneLine|DefaultLookPathUsesPATH|NeverReturnsUnknownStatus)|TestFailed|TestReport.*)$' ./cmd/kirsch ./internal/doctor | grep -c '^Test'
go test -count=1 -run '^(TestRunDoctor.*|TestDoctorBinaryExitStatus|TestRun(RGMissing|GitMissing|NonGitWorkspaceVersusNoFlag|NotInRepositoryRunsOtherChecks|WorkspaceMissingDirectory|BrokenGlobalConfig|BrokenProjectConfig|InvalidConfigValues|UnknownConfigKey|ProjectConfigIgnoredKeyWarns|ConfigLoadedRecord|MultiLineConfigErrorStaysOnOneLine|DefaultLookPathUsesPATH|NeverReturnsUnknownStatus)|TestFailed|TestReport.*)$' ./cmd/kirsch ./internal/doctor; echo "exit $?"
```
You should see two results, in this order.

1. The first command prints one number: how many tests the pattern selects. Expect a number that is not `0`. If it prints `0`, the pattern selected no tests, so the step has not run. Do not tick the box.
2. The second command runs those tests. Expect one `ok` line for `./cmd/kirsch` and one `ok` line for `./internal/doctor`, with no line starting `FAIL` and no compiler error. The last line reads `exit 0`. If any line starts `FAIL`, if a compiler error appears, or if the last line is not `exit 0`, do not tick the box.

- [✅] The first command printed a number that is not `0`, and the second command ended with `exit 0`.

**Optional — tidy up the test folders from this section.** Each step that made a folder gave it its own name (`$d3`, `$d4`, `$c5` and `$c6`), so no name is reused. When you have finished this section, this line removes them:
```
rm -r "$d3" "$d4" "$c5" "$c6"
```
The names exist only in the terminal where you pasted the steps. If you closed that terminal, the folders stay in the system temporary area until you remove them.

**Anything odd?** Write it here:

_(nothing yet)_

---

---

## 13 · Wrapping long lines on screen (m5-d7a)

**What this checks:** long lines of text on the screen wrap onto the next line without losing their indentation or their double spaces, and a reply that is still arriving is cleaned as each piece comes in. It also checks that the reference-screens file has not changed. None of these steps starts kirsch or needs an API key.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Every step below works from there.

**Step 1 — Run the wrapping tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tui -run 'TestWrap|TestTranscriptKeepsIndentAndSpaceRuns|TestTranscriptWrapsIndentedLines'
```
You should see one line starting with `ok` and ending with a time, like:
```
ok  	github.com/djm56/kirsch/internal/tui	0.3s
```
(The time will differ. Lines starting `go: downloading` are also fine; Go prints them the first time it fetches something. There must be no line starting with `FAIL`.)
- [✅] I saw `ok` and no `FAIL`

**Step 2 — Run the tests for reply text that arrives in pieces.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tui -run 'Streamed|AppendText|AppendThinking|StreamEndReleasesRaw|ThinkingBodyLines'
```
You should see one line starting with `ok` and ending with a time, like:
```
ok  	github.com/djm56/kirsch/internal/tui	0.3s
```
(The time will differ. Lines starting `go: downloading` are also fine. There must be no line starting with `FAIL`.)
- [✅] I saw `ok` and no `FAIL`

**Step 3 — Check that the reference-screens file has not changed, and that the screen tests and the screen check pass.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
git diff --stat 686d3c4 -- plan/spec/kirsch-ui-screens.md
git diff --stat 686d3c4 -- internal/tui/view.go
npm run screens
go test -count=1 ./internal/tui
```
You should see:
- The first command prints nothing at all. That means the reference-screens file has no changes since commit `686d3c4`, the pre-mission baseline commit.
- The second command prints one line that names `internal/tui/view.go`, then a summary line. This line is a control: it shows the same command does report a changed file. It compares with `686d3c4` too, so it stays the same after your commits.
- The third command prints two lines that start with `>` (that is npm saying what it is running), then `kirsch-ui-screens.md: grids OK, content OK`. Lines starting `npm notice` are also fine. Anything else means something went wrong.
- The fourth command prints `ok` and the package path with a time. It is the check that proves the screens still match, because it runs the whole `internal/tui` package, golden tests included. Lines starting `go: downloading` are also fine. There must be no line starting with `FAIL`.

If the first command prints anything, do not tick; write what you saw under *Anything odd?* and tell Bishop, because a later piece may have changed the reference-screens file.

- [✅] I saw nothing from the first command, `grids OK, content OK` from the third, `ok` with no `FAIL` from the fourth, and the second command printed a line naming `view.go`

**Anything odd?** Write it here:

_(nothing yet)_

---

## 14 · Paste delay (m5-d8)

**What this checks:** the paste delay setting has been removed from the code and from the spec, because pasting already arrives as one piece when your Terminal sends it that way. You will check that the old setting is gone, that the new automatic paste tests exist and pass, and you will read the limits that apply to some Terminals. No kirsch session starts, no model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder. Every step here runs from there. The first test run can take a minute, and lines starting `go: downloading` are fine.

**Step 1 — Check the old setting is gone from the code.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -rn PasteDebounce internal/; echo "exit $?"
```
You should see only one line: `exit 1`. There must be nothing above it. (`1` means the search found nothing, which is what we want.)
- [✅] Nothing above `exit 1`

**Step 2 — Check the spec no longer names the old setting.**
Paste:
```
grep -n 'Paste debounce' plan/spec/ui-spec-v0.1.md; echo "exit $?"
```
You should see only one line: `exit 1`. There must be nothing above it.
- [✅] Nothing above `exit 1`

**Step 3 — Check the committed copies from before the change still have the old setting.**
This proves the checks above can fail. The copies are old on purpose. They come from commit `686d3c4`, the last commit before the change, so they do not move.
Paste:
```
cd /Volumes/DATA/Github/kirsch && git show 686d3c4:internal/tui/styles.go | grep -n PasteDebounce; echo "exit $?"
cd /Volumes/DATA/Github/kirsch && git show 686d3c4:plan/spec/ui-spec-v0.1.md | grep -n 'Paste debounce'; echo "exit $?"
```
You should see this, in order:
```
22:	PasteDebounce   = 20 * time.Millisecond  // one update per paste
exit 0
565:- **Paste debounce** (§11) so a large paste is one update, not thousands.
758:| Paste debounce | 20ms | One update per paste |
exit 0
```
Do not tick the box below if the lines on screen differ from these. A different line means the command or the commit is wrong, and the check has not been run.
- [✅] The first command prints the line starting `22:` then `exit 0`, and the second prints the lines starting `565:` and `758:` then `exit 0`

**Step 4 — List and count the new paste tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -list '^TestUnbracketed' ./internal/tui | grep '^TestUnbracketed'
go test -list '^TestUnbracketed' ./internal/tui | grep -c '^TestUnbracketed'; echo "exit $?"
```
You should see two test names, then a number on its own line, then `exit 0`. The two names must be exactly `TestUnbracketed5KBPasteLandsWholeThroughKeyMessages` and `TestUnbracketedMultiLinePasteCurrentBehaviour`. The number after them must not be `0`. If no names appear, or the number is `0`, the tests were not found, and the step has not passed. Step 5 runs the tests these names come from.
- [✅] The two names on screen are the two listed above, and the number after them is not `0`

**Step 5 — Run the paste tests.**
Paste:
```
go test -count=1 -run '^TestUnbracketed' ./internal/tui; echo "exit $?"
```
It runs the tests named in Step 4, and every one of them must pass.
You should see a line starting `ok` and, as the very last line, `exit 0`. A line with `FAIL` in it means something is wrong.
- [✅] The last line says `exit 0`, and no line says `FAIL`

**Step 6 — Read the limits that apply to some Terminals.** Nothing to paste in this step. Do not start kirsch.

Normally your Terminal sends a paste to kirsch as one block, and long text lands whole. Some Terminals do not do that. Then kirsch reads the paste as if you had typed each key, and these things can go wrong:

1. **Text with line breaks.** Only the first line reaches kirsch, and kirsch sends it as your message. The rest of the paste is dropped while kirsch is busy. This is a known problem and is not fixed. The crew has written it down. Whether it is acceptable for version 0.1, or needs a fix, is your decision. It is open in `plan/PROGRESS.md`, under *Open from m5-d8*. The crew checked this with a test that feeds Enter key messages to kirsch's code, without a real terminal. The test does not run the library's reader; that a line break arrives as Enter is read from the library's source.
2. **Tabs.** A tab in your text arrives as the Tab key. What you would see: if the typing line holds a slash command name that matches exactly one command, Tab finishes the name. If the name matches several commands, or none, Tab does nothing and the rest of the paste is typed in as normal. Otherwise Tab moves the focus up to the transcript (the messages and cards above the typing line), and the line at the bottom of the screen changes. After that, the rest of the paste does not appear in the typing line. Its letters act on the transcript: `d` opens a detail view, `g` and `G` move the view, and `?` opens help. A second Tab in the same paste moves focus back to the typing line, and the text after it is typed in again, so one paste can end up split between the two. The crew read the Tab key's arrival from Bubble Tea's source and what kirsch does with it from kirsch's code. Neither was run. It is recorded in `plan/PROGRESS.md`, under *Open from m5-d8*.
3. **A rare accented or non-English letter** can be lost if the paste is cut at a terminal read boundary. The cause is in the library kirsch uses to read the keyboard, so kirsch cannot fix it alone. The crew read this from the library's source and did not test it.

- [✅] I have read the limits above

**Anything odd?** Write it here:

_(nothing yet)_

---

## 15 · The automatic release steps on GitHub (m5-d11)

**What this checks:** the steps that build kirsch's release files, and the notices that go into each one. Steps 2 to 4 run on your Mac. Inside the kirsch folder they change only the `dist` and `build` folders, which Git ignores. The first run also downloads tools into Go's own download folder on your Mac. Step 5 runs on GitHub after you commit and push. It starts a dry run, which publishes nothing. No kirsch session starts, no model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

Every step works from the kirsch folder.

**Step 1 — Go to the kirsch folder.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
```
You should see your prompt again, with no message.
- [✅] Done

**Step 2 — Build the release files on your Mac.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run release:snapshot; echo "result: $?"
```
The first run downloads tools, so it takes a few minutes. Wait for it to finish.

The run prints many lines, and their exact text can vary from one Mac to another. You may see lines like these: lines starting with `>` (npm saying what it is running), lines starting `go: downloading`, and lines starting `npm notice`. On this machine a healthy run also printed a line starting `go:` that says it is switching Go version, and a line that contains `error=invalid semantic version tag=m4`. That second line comes from an older tag in the project, and the run still succeeded. Two things matter:
- `release succeeded` near the end.
- `result: 0` as the last line.

Any other number on the last line means the run failed. If that happens, paste the same command again once; if the cause was a network problem, a second run may clear it. Running it twice is safe. If it still fails, tell Bishop.

You should see these as the last lines. The number of seconds will differ on your Mac:
```
  • release succeeded after 8s
  • thanks for using GoReleaser!
result: 0
```
- [✅] I saw `release succeeded` near the end and `result: 0` last

**Step 3 — Check that the archive holds the notices.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
tar -tzf dist/kirsch_*_darwin_arm64.tar.gz | sort | grep -E '^(kirsch|LICENSE|README\.md|CHANGELOG\.md|NOTICE|licenses/go/LICENSE)$'
tar -tzf dist/kirsch_*_darwin_arm64.tar.gz | grep -c '^licenses/'
```
You should see these names. The order may differ on your Mac; what matters is that all of them appear:
```
CHANGELOG.md
kirsch
LICENSE
licenses/go/LICENSE
NOTICE
README.md
```
Then a line with the number `29`.

What the names are:
- `kirsch` is the program.
- `LICENSE` is kirsch's own licence.
- `README.md` is the front page.
- `CHANGELOG.md` is the list of changes.
- `NOTICE` names the libraries kirsch uses, and their licences.
- `licenses/go/LICENSE` is the licence of the Go language that kirsch is built with.

The `29` counts the licence texts in the `licenses` folder. It can change when the libraries change. What matters is a number above zero, with no error message.
- [✅] I saw all of them, then a number above zero

**Step 4 — Check the checksums file.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
ls dist/*_checksums.txt
```
You should see one file name that ends in `_checksums.txt`, for example:
```
dist/kirsch_m4-SNAPSHOT-f6d0cd0_checksums.txt
```
The middle part of the name is the version. It can differ on your Mac, depending on the latest tag and the commit. The checksums file holds one fingerprint for each archive, so someone can check that an archive was not changed.
- [✅] I saw one `_checksums.txt` file name

**Step 5 — Start a dry run on GitHub (you do this in your browser).**
This step runs the copy of the project on GitHub, so commit your changes and push them first. Changes that are not pushed do not run. Publishing a real release is a later step, with a version tag, and it is not part of this section. The release steps are set up so that a tag works only on a commit that is already on `main`; if it is not, the run should fail with an error before it builds or publishes anything. That check has not been tried on GitHub yet.

1. Open the kirsch repository on GitHub, `https://github.com/djm56/kirsch`, signed in as the owner.
2. Click the *Actions* tab.
3. In the list on the left, click *Release*.
4. Click *Run workflow*. Choose the branch `main`, then click the green *Run workflow* button.
5. Wait for the run to finish. It takes a few minutes. Click the run to open its page.

If GitHub's buttons are named a little differently, look for the closest match; if you cannot find *Run workflow*, stop and tell Bishop.

You should see a green tick on the run. On its page, an artifact named `release-dry-run` is listed. The *Releases* page shows no new release. This step publishes nothing.

If the run is red, or a step says permission denied, stop and tell Bishop the section and step number.
- [ ] The run is green, the `release-dry-run` artifact is on its page, and the *Releases* page shows no new release

**Anything odd?** Write it here:

_(nothing yet)_

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

**What this checks:** the install guide tells you how to install Kirsch and how to check the install. These steps try the install routes that can run on this Mac, and they check that the version lines and the doctor output match what the guide shows.

**Before this section:** do *Before you start* in this Terminal window.

You start in the kirsch folder, and every step works from there.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Read the install guide.**
Paste:
```
open doc/install.md
```
(If you would rather read it in the Terminal, paste `less doc/install.md` instead, and press `q` to leave.)
Read the guide from the top. The part about installing from a release archive is checked at the final release check (section 37 of this checklist), not here. Read the rest now.
- [✅] I read the install guide

**Step 3 — Check the guide has no planning words.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n -i -w -E 'milestones?|ADRs?' doc/install.md
printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -i -w -E 'milestones?|ADRs?'
```
The first command should print nothing. That is the pass. The second command prints test words, so you can see the check works:
```
an ADR
ADRs
milestones
```
It must not print `headroom`. The words it looks for are project planning words, which should not appear in a guide for users.
- [✅] I saw no output from the first command, and the test words from the second, without `headroom`

**Step 4 — Install with `go install` into a throwaway folder.**
Paste:
```
GOBIN="$T_CFG/gobin" go install github.com/djm56/kirsch/cmd/kirsch@latest
"$T_CFG/gobin/kirsch" --version
```
This needs the internet. It downloads the source from Go's module proxy, and it sends nothing to a model. Lines starting `go: downloading` are fine. You should see:
```
kirsch 0.1.0-dev
```

Once Kirsch has a release, the version line shows that release instead, for example `kirsch 0.1.0`. That also counts as a match.

It says `0.1.0-dev` while Kirsch has no release, and a build of the public code without a release tag reports that development version.
- [✅] I saw `kirsch 0.1.0-dev` (or the release version, as above)

**Step 5 — Build from source, in a throwaway copy.**
Paste:
```
git clone -q /Volumes/DATA/Github/kirsch "$T_CFG/src"
bash "$T_CFG/src/scripts/build.sh"
"$T_CFG/src/bin/kirsch" --version
```
The build prints nothing when it works. The copy holds only what is committed in the kirsch folder, so a file that is not committed yet is not in it. That is fine for this step, because it checks the build route, not your working files. You should see:
```
kirsch 0.1.0-dev
```

Once Kirsch has a release, this line shows that release instead, for example `kirsch 0.1.0`, but only if the latest commit in the kirsch folder is exactly that release's tag. On any other commit it still shows `kirsch 0.1.0-dev`. Either counts as a match.

- [✅] I saw `kirsch 0.1.0-dev` (or the release version, as above)

**Step 6 — Run the doctor check.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
"$KIRSCH_BIN" doctor; echo "exit=$?"
```
Doctor only looks. It makes no network calls and sends nothing to a model. You should see these lines, in this order:
```
kirsch 0.1.0-dev
ok  git: found at <the path to git on your Mac>
ok  rg: found at <the path to rg on your Mac>
ok  repository: <the path of the kirsch folder>
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6854 bytes
ok  state: <a folder inside the sessions folder Setup B named> does not exist yet; kirsch will create it (<the nearest folder that already exists> is writable)
ok  lock: free (no lock file yet)
doctor: 9 ok, 1 warn, 0 fail
exit=0
```

Once Kirsch has a release, the first line (`kirsch 0.1.0-dev`) shows that release instead, for example `kirsch 0.1.0`, but only if the kirsch folder was exactly on that release's tag, with no uncommitted changes, when Step 1 built it. Otherwise it still shows `kirsch 0.1.0-dev`. Either counts as a match.

The lines in angle brackets depend on your Mac, so read them as descriptions, not exact text. The paths will differ. The `git` and `rg` paths are where those programs live on your Mac. The `state` line names a folder inside the sessions folder that Setup B printed as "Sessions will be saved in:". The folder in brackets is the nearest one that already exists. If you have already started kirsch inside the kirsch folder in this test setup, the line reads `exists and is writable` instead, and the `lock` line may read `free` without `(no lock file yet)`. Kirsch started in the practice project is a different project and does not count. Either way the line is `ok`. The `context` line names `AGENTS.md`, and its byte count changes if that file changes. If ripgrep is not installed, the `rg` and `backend` lines read `warn`, and the last count line changes. The guide says a warning does not make doctor fail. The `key` line reads `warn` here because this test window has no key set, as the guide's example shows.
- [✅] I saw the first line `kirsch 0.1.0-dev` (or the release version, as above), the `key` line as `warn`, the same lines in the same order, and the last line `exit=0`, apart from the differences named above

**Optional — tidy up the throwaway folders.**
```
rm -rf "${T_CFG:?}/gobin" "${T_CFG:?}/src"
```
This removes only the throwaway folders made by the install steps above. The other test folders are covered by `m5_done`, as the top of this file says.

**Anything odd?** Write it here:

_(nothing yet)_

---

## 18 · Fresh machine, no API key check (m5-d16)

**What this checks:** what Kirsch shows when you start it with no API key. The first step runs a check that does this for you. The later steps let you see the same start by hand, and see what changes once you type a command.

**Before this section:** do *Before you start* in this Terminal window. Step 1 works from the kirsch folder. Steps 2 to 4 start Kirsch in new temporary folders, so they do not use your own sessions, and each one needs the Terminal window at the size the step gives.

**Step 1 — Run the no-key check.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run check:no-key; echo "exit=$?"
```
The check takes about 35 seconds. It builds Kirsch first. If the build fails, the check stops and shows the build's output, and the `exit=` line at the end is not `exit=0`. Your own key settings do not reach the check, so it works even when a key is set in this window. Here is the whole output of a run from a normal shell, without `env -i`. The first lines are npm's own, and they are not part of the result:
```

> kirsch@0.1.0-dev check:no-key
> bash scripts/no-key-check.sh

 . No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY.
no-key-check: PASS (the line was emitted to the terminal)
exit=0
```
The key line starts with one space, then the dot, and both are part of the text Kirsch prints. The check passed when its output contains the `no-key-check: PASS (the line was emitted to the terminal)` line and `exit=0` is the last line. Lines between them do not change the result. If the PASS line is missing, or `exit=0` is not the last line, the check did not pass. Copy the last lines of the output and tell Bishop.
- [ ] I saw the `No API key` line, the `PASS` line, and `exit=0`

**Step 2 — Look at a fresh start with no key.**
Set this Terminal window to 80 columns wide and 24 rows tall. In Terminal, the Window settings for the profile hold the Columns and Rows fields.

Paste:
```
FAKE_HOME="$(mktemp -d)" && cd "$(mktemp -d)" && git init -q . && env -i HOME="$FAKE_HOME" PATH="$PATH" TERM=xterm-256color /Volumes/DATA/Github/kirsch/bin/kirsch
```
This starts Kirsch as on a new computer: no key, and an empty home folder. It does not use your own settings or your own sessions.

You should see, from the top:
- the word KIRSCH with a space between each letter (this test has no language setting, so Kirsch falls back to plain letters)
- the version line, `v0.1.0-dev . terminal-native coding agent`
- the key lines: `. No API key: set KIRSCH_OPENCODE_API_KEY → OPENCODE_API_KEY.`, then `. Keys are never read from config files.`
- one blank line, then the note that starts `. Using opencode sends prompts and file contents to`. It may wrap onto more than one row.
- one blank line, then `. no previous session - starting fresh`. The dash in that line is a plain dash here.
- near the bottom, the typing line: `> Ask anything (Enter to send, /help for help)`

Quit with two `Ctrl+C`.
- [ ] I saw the key lines with both key names, the `Keys are never read` line, then one blank line, the OpenCode note that starts `. Using opencode sends prompts`, one blank line, the `no previous session` line, and the typing line at the bottom

**Step 3 — Type a command at the same size.**
Paste the same command from step 2 again. Then type `/status` and press Enter.

The onboarding goes away. The screen shows the messages instead: the `no previous session` line, then the status text (model, endpoint, key source and so on). The typing line at the bottom shows only the cursor. That is expected at this size, because the status text is too tall to fit under the onboarding.

Quit with two `Ctrl+C`.
- [ ] After `/status`, the onboarding was gone and the typing line showed only the cursor

**Step 4 — The same command with a taller window.**
Set the window to 80 columns wide and 40 rows tall. Paste the same command from step 2 again, and type `/status` and press Enter.

This time the onboarding should stay, with the `no previous session` line and the status text beneath it. The typing line at the bottom should show `Ask anything (Enter to send, /help for help)` again.

Quit with two `Ctrl+C`.
- [ ] With 40 rows, the onboarding stayed, with the status text beneath it, and the typing line showed `Ask anything`

The folders made by steps 2 to 4 hold only test data, and this test does not remove them.

**Anything odd?** Write it here:

_(nothing yet)_

---

## 19 · Starting a new session with `/new` (only if you choose to do it) (m5-d17a)

**What this checks:** what happens to a session that Kirsch makes when it starts, or when you type `/new`, and you then quit with nothing written in it. Such a session is now removed on quit, so the next start no longer picks up an empty one. Steps 1 and 2 run tests. You do not type anything into Kirsch in this section. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Each step starts in the kirsch folder.

**Step 1 — Run the session tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/app/ -run 'TestCloseAfterNewSessionLeavesNoEmptySession$|TestNewSessionThatReceivedAnEventSurvivesClose$|TestFreshLaunchSessionIsAbandonedLikeANewOne$|TestResumedSessionIsNeverDeleted$|TestNewSessionRotatesTheProviderSessionID$' -v; echo "exit=$?"
```
You should see lines starting `=== RUN` and `--- PASS`, and no line starting `--- FAIL`. Here is the output of the run on 2026-11-05, copied as it was printed. Your times will differ a little, and that is normal:
```
=== RUN   TestCloseAfterNewSessionLeavesNoEmptySession
--- PASS: TestCloseAfterNewSessionLeavesNoEmptySession (0.08s)
=== RUN   TestNewSessionRotatesTheProviderSessionID
--- PASS: TestNewSessionRotatesTheProviderSessionID (0.08s)
=== RUN   TestNewSessionThatReceivedAnEventSurvivesClose
--- PASS: TestNewSessionThatReceivedAnEventSurvivesClose (0.06s)
=== RUN   TestResumedSessionIsNeverDeleted
--- PASS: TestResumedSessionIsNeverDeleted (0.08s)
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_quit
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_/new_then_quit
=== RUN   TestFreshLaunchSessionIsAbandonedLikeANewOne/startup_with_an_event_then_/new_then_quit_(control)
--- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne (0.17s)
    --- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_quit (0.05s)
    --- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne/empty_startup_then_/new_then_quit (0.06s)
    --- PASS: TestFreshLaunchSessionIsAbandonedLikeANewOne/startup_with_an_event_then_/new_then_quit_(control) (0.06s)
PASS
ok  	github.com/djm56/kirsch/internal/app	0.736s
exit=0
```
The output above ends with `exit=0`, as yours should.

In plain words, each PASS means this:
- `TestCloseAfterNewSessionLeavesNoEmptySession`: a session that already has something in it is replaced by `/new`, nothing is written to the new session, and Kirsch quits. Only the old session's file is left, and `index.json` points at that old session.
- `TestNewSessionRotatesTheProviderSessionID`: the first message is sent with the provider session id set before `/new`. After `/new`, the next message is sent with a different id, which is 32 lowercase hexadecimal characters.
- `TestNewSessionThatReceivedAnEventSurvivesClose`: a new session that receives one event after `/new` is kept on quit. The old session's file is also kept, and `index.json` points at the new session.
- `TestResumedSessionIsNeverDeleted`: an empty session that was resumed is kept on quit, and it is still kept after a later `/new` and quit. The empty session that `/new` made is removed, and `index.json` points at the resumed session.
- `TestFreshLaunchSessionIsAbandonedLikeANewOne`: it checks these situations in turn. A launch with nothing written, then quit, leaves no session file and no index entry. A launch with nothing written, then `/new`, then quit, leaves no session file and no index entry. A launch with one event written, then `/new`, then quit (the control), leaves only the first session's file, and `index.json` points at it.

A `--- FAIL` line, or an `exit=` value other than 0, means the check did not pass. Copy the last lines and tell Bishop.
- [ ] I saw `--- PASS` for each named test, the `ok` line, and `exit=0` as the last line

**Step 2 — Run the `/new` test through the real wiring.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./cmd/kirsch/ -run 'TestSlashNewThroughRealWiringStartsANewSession$' -v; echo "exit=$?"
```
This test starts a session, records one real event in it, then sends `/new` through the same wiring that Kirsch uses at the keyboard. It asserts that the session id that `/status` reports changes after `/new`, and that the old id is no longer on screen. It asserts that two session files exist, the old session's and the new one, and that exactly one of them holds content: the old session's event.
You should see `--- PASS` for the test, and an `ok` line. Here is the output of a run on 2026-11-05, copied as it was printed:
```
=== RUN   TestSlashNewThroughRealWiringStartsANewSession
--- PASS: TestSlashNewThroughRealWiringStartsANewSession (0.16s)
PASS
ok  	github.com/djm56/kirsch/cmd/kirsch	0.669s
exit=0
```
The output above ends with `exit=0`, as yours should.
- [ ] I saw `--- PASS` for the test, the `ok` line, and `exit=0` as the last line

**What the code does (read from the code, not seen in a run).** A session file that this Kirsch created and that nothing was written to is removed when you quit (`Close`, `internal/app/app.go:890`) or when `/new` replaces it (`installSession`, `internal/app/app.go:1303`). A session that received any event, or that was resumed, is never removed. When the removed session is the one `index.json` names for this workspace (at quit, the session you were in), the entry is put back to what it held before Kirsch created that session, skipping any earlier empty sessions that `/new` already removed, or deleted if there was nothing earlier or that earlier session's file is gone (`restoreIndexLocked`, `internal/app/app.go:1085`; `restorePoint`, `internal/app/app.go:1111`). After `/new`, the entry names the new session; at quit it goes back as described only if that session is still empty. So, after a normal quit, the next launch does not find an empty session that this Kirsch created. Which notice that launch shows depends on how it is started (`chooseSessionInit`, `cmd/kirsch/main.go:277–310`). `/new` also sets a new provider session id (`randomSessionID()`, `internal/app/app.go:1039`, set in `NewSession` at `internal/app/app.go:1281`). Nothing in this change needs a key or a model.

**Anything odd?** Write it here:

_(nothing yet)_

---

## 20 · The doctor command: key, backend, model and project notes lines (m5-d4a)

**What this checks:** `kirsch doctor` now also tells you which API key it found, by name only, whether it can use `rg` (the fast search tool), which model your settings name, and which project notes file kirsch will read. It never shows your key, and it never sends a message to a model. No API key is used: the one key check in this section sets a dummy value for a single command, and nothing is kept.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. The steps below run doctor from there, because the project notes line reads `AGENTS.md` from the kirsch folder.

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

**Step 2 — No key is set.** Setup B cleared the key variables, so this run has no key.
Paste:
```
cd /Volumes/DATA/Github/kirsch
"$KIRSCH_BIN" doctor; echo "exit=$?"
```
You should see the version line, then the check lines. Three of them matter here:
- A line starting `warn  key:` that ends with `kirsch starts but cannot send a prompt`. It names both `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY`. That is correct: no key is set, so kirsch starts, but it cannot send a message yet.
- A line starting `ok  model:` with the default model, `minimax-m3`, and its context window in tokens.
- A line starting `ok  context: AGENTS.md,` followed by a size in bytes. The size is the file's size today, so it may differ from the example you see.

With `rg` installed, the summary line reports warnings and no failure: no line starts `fail`, and the run ends `exit=0`. The `key` line is the warning, because no key is set. A warning does not stop doctor. If `rg` is not installed on your Mac, the `rg` line and the `backend` line both start with `warn` (the `backend` line says `pure-go`), and the run still ends `exit=0`.
- [✅] I saw the `warn  key:` line naming both variables, the `ok  model:` line, the `ok  context: AGENTS.md` line, and `exit=0`

**Step 3 — Your own settings.** This run does not use the test settings folder, so it reads the settings you normally use.
Paste:
```
cd /Volumes/DATA/Github/kirsch
env -u XDG_CONFIG_HOME "$KIRSCH_BIN" doctor; echo "exit=$?"
```
You should see one `model:` line for the model your own settings name. If kirsch knows that model, the line starts with `ok`. If it does not, the line starts with `warn  model:` and says `not in the table; conservative fallback`. That is a warning, not a failure: kirsch then uses a conservative context size for a model it does not know. On the run Bishop made, the settings named `claude-haiku-5-5`, which kirsch does not list, so this line warned. If yours names the same model, expect the same warning.
The run ends `exit=0`.
- [✅] I saw the `model:` line for my own model, and `exit=0`

**Step 4 — A dummy key, for one command at a time.** This step uses a dummy value, not your key. Each line below sets the dummy for that one command only, so nothing stays set in this Terminal window. Do not type your own key here, and do not paste one into this file or into a chat.
Paste:
```
cd /Volumes/DATA/Github/kirsch
KIRSCH_OPENCODE_API_KEY=dummy-not-a-key "$KIRSCH_BIN" doctor | grep "key:"
KIRSCH_OPENCODE_API_KEY=dummy-not-a-key "$KIRSCH_BIN" doctor | grep -c dummy-not-a-key
KIRSCH_OPENCODE_API_KEY=dummy-not-a-key "$KIRSCH_BIN" doctor | grep -c KIRSCH_OPENCODE_API_KEY
```
You should see three results, in this order.
1. One line starting `ok  key:`. It ends with `key found in KIRSCH_OPENCODE_API_KEY`. The variable's name is on that line, and the dummy value is not.
2. `0`. The dummy value does not appear anywhere in doctor's output.
3. `1`. This proves the search works: the check finds the variable's name once, so the `0` above means the search ran and found nothing to leak.

If the second result is not `0`, the value appears in doctor's output. Do not tick the box; tell Bishop the section and step.
- [✅] The key line ends with `KIRSCH_OPENCODE_API_KEY`, the second result is `0`, and the third is `1`

**Step 5 — Confirm no key is set.** This command unsets the key variables and then counts any that are still set. The check must pass before any later section.
Paste:
```
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
keycount_control
keycount
```
You should see `CONTROL PASS: a planted key line counts as 1`, then `0`. The `CONTROL PASS` line shows that the counter works, and the `0` shows that no key variable is still set.
- [✅] I saw `CONTROL PASS` and `0`

**Step 6 — Run the automatic tests for this piece.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
PAT='^(TestKeyLineNamesWinner|TestKeyLineNilGetenvIsEmptyEnvironment|TestKeyValueNeverPrinted|TestKeyLineWithholdsNonNameAPIKeyEnv|TestModelLine|TestContextLine|TestBackendLineUnderPathShim|TestResolvedLinesSkippedWhenConfigDidNotLoad|TestContextLineSkippedWithoutWorkspace|TestResolvedLinesNeverFail|TestRunDoctorKeyLine|TestRunDoctorModelLine|TestRunDoctorContextLine|TestRunDoctorBackendLine|TestRunDoctorSkippedLinesWhenConfigDidNotLoad|TestIsolatedEnvClearsKeyVariables|TestAllowedIsExact|TestImportRules)$'
printf '%s\n' "$PAT" | grep -o 'Test[A-Za-z0-9_]*' | wc -l
go test -list "$PAT" ./internal/doctor ./cmd/kirsch ./internal/arch | grep -c '^Test'
go test -count=1 -run "$PAT" ./internal/doctor ./cmd/kirsch ./internal/arch; echo "exit $?"
```
You should see the following, in this order.

1. The first two commands each print one number. The first is how many test names the pattern carries. The second is how many tests `go test -list` selects from it. The two numbers must be equal. This manual does not write the number down, so no number is expected in advance. If they differ, the pattern selects a different set of tests from the names it carries, so do not tick the box. A `0` means it selected no tests at all, so the step has not run.
2. The third command runs those tests. Expect one `ok` line for each of `./internal/doctor`, `./cmd/kirsch` and `./internal/arch`, no line starting `FAIL`, and the last line reads `exit 0`. If any line starts `FAIL`, if a compiler error appears, or if the last line is not `exit 0`, do not tick the box.

- [✅] The first two commands printed the same number, and the third command ended with `exit 0`.

**Step 7 — A pattern that selects nothing (on screen, nothing to save).** This shows why the count in step 6 comes first. Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 -run '^NoSuchTestXYZ$' ./internal/doctor; echo "exit $?"
```
You should see one line starting `ok` that ends `[no tests to run]`, and then `exit 0`. The test command ends `exit 0` even though it ran no test, because Go counts "no tests to run" as a success. The count in step 6 is what catches that.
- [✅] I saw `[no tests to run]` on the `ok` line and `exit 0` at the end

**Anything odd?** Write it here:

_(nothing yet)_

---

---

## 21 · Marker for cut text in the typing area (m5-d7b)

**What this checks:** a line you type in kirsch's typing area that is too long for the window ends in `⋯` where it is cut, so you can see that it was cut. A line that fits is not marked.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. The steps before the last one run tests and checks from there. The last step starts kirsch, and it needs no model and sends nothing.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build; echo "exit $?"
```
You should see a line starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. The last line must be `exit 0`. If it is `exit` followed by any other number, the build failed: do not tick, write what you saw under *Anything odd?* at the end of the section, and tell Bishop. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Check that the marker tests exist.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -list 'TestComposerCut|TestComposerFit|TestComposerBusyFirst|TestComposerCaretOver' ./internal/tui | grep '^Test'
```
You should see test names, one per line, each starting with `Test`, for example `TestComposerCutRowsEndInMarkerAt40`. An empty list means the names have changed, and you should tell Bishop.

Then paste the same command with a count at the end:
```
cd /Volumes/DATA/Github/kirsch
go test -list 'TestComposerCut|TestComposerFit|TestComposerBusyFirst|TestComposerCaretOver' ./internal/tui | grep -c '^Test'
```
You should see a number that is not 0. (`grep -c` finishes with a non-zero code when the number is 0, so read the number, not the code.)
- [✅] I saw test names, and the number was not 0

**Step 3 — Run the marker tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 -run 'TestComposerCut|TestComposerFit|TestComposerBusyFirst|TestComposerCaretOver' ./internal/tui; echo "exit $?"
```
You should see a line starting with `ok` and ending with a time, then:
```
exit 0
```
If you see `FAIL` and `exit 1`, do not tick. Write what you saw under *Anything odd?* at the end of the section.
If you see `[no tests to run]` anywhere in the output, do not tick, even with `exit 0`. That wording means the name filter matched no test, so no marker test ran. Write what you saw under *Anything odd?* and tell Bishop.
- [✅] I saw `ok` and `exit 0`

**Step 4 — Check the reference screens against the committed copy from before this work.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
git diff --stat 686d3c4 -- plan/spec/kirsch-ui-screens.md
git diff --stat 686d3c4 -- internal/tui/composer.go
npm run screens; echo "result: $?"
```
You should see nothing from the first command. The second command is a control: it should print two lines. The first names `internal/tui/composer.go` with its count of changed lines, because that file did change after commit `686d3c4`. The second is git's summary line. If it prints nothing, the comparison is not working, and do not tick; tell Bishop. The third command prints `kirsch-ui-screens.md: grids OK, content OK`, then `result: 0`.
Nothing printed by the first command means the saved screens are the same as commit `686d3c4`. Any line printed means they changed, so do not tick, and tell Bishop. If the third command prints anything other than `grids OK, content OK`, or ends with a result other than `0`, do not tick, either.
- [✅] Nothing printed by git, and I saw `grids OK, content OK` and `result: 0`

**Step 5 — Start kirsch and type a long line. Do not press Enter.** *The crew could not run this step; it needs your real Terminal.*
Make the Terminal window 40 columns wide. Then paste:
```
stty size
```
You should see two numbers, rows then columns (for example `24 40`). The second number must be from 40 to 47 for this step. Below 40 columns kirsch shows `terminal too narrow` instead of its screen; if that happens, widen the window to 40 and run `stty size` again. If the second number is 48 or more, narrow the window to 40 and run `stty size` again.

Then paste:
```
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
You should see kirsch's screen, with the typing area near the bottom. At this width the empty typing area is not blank. It shows the dim placeholder, cut to the window: `> Ask anything (Enter to send, /help ⋯`, with `⋯` as its last character. Check that before you type. In a window 48 or more columns wide the placeholder shows whole instead, as `> Ask anything (Enter to send, /help for help)`; this step does not use that width, but the typed-line check below still applies to it. Nothing calls a model unless you press Enter on a message, and you will not press Enter in this section.
Type one line of letters that is longer than the window, for example a run of `a` about twice as long as the window is wide. Do not press Enter.
You should see the line stop at the right edge of the typing area, with `⋯` as its last character. The `⋯` sits one column in from the window's right edge, not on it.
Press Esc once. The typing area should empty and show the same dim `> Ask anything (Enter to send, /help ⋯` line again, with the `>` at the left.
Only when the typing area shows the dim `Ask anything` line again, type `/quit` and press Enter. You should see your Terminal prompt again.
Known limit: when a line is cut, the cursor after the cut is not on screen. kirsch does not scroll sideways in v0.1, so this is expected and is not a defect in this section.
- [✅] The long line ends in `⋯` at the right edge of the typing area
- [✅] Esc emptied the typing area
- [✅] `/quit` closed kirsch, and I am back at my Terminal prompt

**Anything odd?** Write it here:

_(nothing yet)_

---

## 22 · The usage guide corrections (m5-d14a)

**What this checks:** the usage guide, the page `doc/usage.md`. The steps check that the old debug commands and the claim about a missing warning are gone, that the guide does not use planning words or code references, that the start-up commands match what kirsch prints, and that a sentence about approvals was replaced. Then you read the parts of the guide that changed. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

You start in the kirsch folder. Step 1 goes there, and every other step works from there. Step 5 shows kirsch's help screen, which starts nothing. Step 8 starts kirsch so you can compare its screen with the guide, and sends it no message.

**Step 1 — Go to the kirsch folder.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
```
You should see your prompt again, with no message.
- [✅] Done

**Step 2 — The old debug commands are gone.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff' doc/usage.md
printf '/read <path>\n' | grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff'
```
You should see nothing from the first command. The second command shows this line, which proves the search does find a command when it is there:
```
1:/read <path>
```
- [✅] Nothing from the first command, and `1:/read <path>` from the second

**Step 3 — No planning words or code references in the guide.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -i -w -E 'milestones?|ADRs?' doc/usage.md
printf 'two milestones\nan ADR\nheadroom\n' | grep -n -i -w -E 'milestones?|ADRs?'
grep -E 'internal/|cmd/kirsch/|Sources?:' doc/usage.md
printf 'see internal/tui\nSource: x\nplain\n' | grep -n -E 'internal/|cmd/kirsch/|Sources?:'
```
You should see nothing from the first command, then the lines from the second. The word `headroom` is not printed:
```
1:two milestones
2:an ADR
```
Then nothing from the third command, then the lines from the fourth. The word `plain` is not printed:
```
1:see internal/tui
2:Source: x
```
- [✅] Nothing from the first and third commands, and the control lines from the second and fourth

**Step 4 — Build kirsch.**
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

**Step 5 — The start-up commands match the guide.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep 'kirsch resume\|kirsch --new' doc/usage.md
"$KIRSCH_BIN" --help
```
You should see these grep lines. The ones under *Starting it* are the start-up commands and the lines of the paragraph just below them that mention `kirsch resume` or `kirsch --new`. The line near the top and the lines near the end are in other parts of the guide, and that is fine:
```
> `kirsch resume`. The layout, the key bindings, the slash commands and the
kirsch resume                # resume the most recent session for this workspace
kirsch --new                 # start a fresh session even when a previous session exists
`kirsch resume` picks up the most recent session for this workspace, including
starts a fresh session instead and says so. `kirsch --new` starts a fresh
`kirsch resume` (or auto-resume) restores them, unless another Kirsch is already
cleared after `kirsch resume`. A new session, including one started with
```
Then you should see this help screen. It names `kirsch resume` and the `--new` flag, the same names the guide uses:
```
Usage:
  kirsch [flags]
  kirsch resume [flags]
  kirsch doctor [flags]

Commands:
  resume    resume the most recent session for this workspace
  doctor    check the installation and report problems

Flags:
  --workspace <path>  workspace root (defaults to the enclosing Git repository)
  --new               start a fresh session even when a previous session exists
  --debug             write a structured debug log
  --version           print the version and exit
  --help, -h          show this help and exit
```
The `doctor` lines are checked in the doctor section, not here.
- [✅] I saw the grep lines, and the help screen names `kirsch resume` and `--new`

**Step 6 — The approval wording is changed.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep 'no setting turns that off' doc/usage.md
grep 'Nothing consequential happens silently' doc/usage.md
printf 'Nothing consequential happens silently.\n' | grep -n 'Nothing consequential happens silently'
```
You should see this line from the first command:
```
A patch appears as a card and waits, and no setting turns that off. A command
```
The second command prints nothing at all. The third prints the control line, which proves the search does find the sentence when it is there:
```
1:Nothing consequential happens silently.
```
- [✅] I saw the line from the first command, nothing from the second command, and the control line `1:Nothing consequential happens silently.` from the third

**Step 7 — The missing `rg` claim is gone.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep 'missing .rg' doc/usage.md
printf 'a missing `rg` warning\n' | grep -n 'missing .rg'
```
You should see nothing from the first command. The second command prints the control line, which proves the search does find the phrase when it is there:
```
1:a missing `rg` warning
```
- [✅] Nothing from the first command, and the control line ``1:a missing `rg` warning`` from the second

**Step 8 — Read the guide, then compare it with kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
open doc/usage.md
```
The guide opens in your text viewer. Read these parts, and tick each one when you have read it:
- [✅] *The screen*: the status bar, the composer, and *Sizes*
- [✅] *Everywhere*: the *Pasting* paragraph
- [✅] *The approval model*: the paragraph about session grants

Then compare the guide with kirsch. Paste:
```
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
kirsch opens on the practice project. Look at the bar and the typing line, and check them against the guide. Do not paste or type a message. Type `/quit` and press Enter to close kirsch. The main area may also show a short notice about the session, for example that it is starting fresh or that it resumed. The guide does not describe these notices, and that is fine. If another kirsch is already open in this project, the bar shows a warning and the session is not resumed; write that under *Anything odd?*, then close the other kirsch. Starting kirsch does not call a model.
- [✅] I compared the bar and the typing line with the guide, and wrote anything that differed under *Anything odd?*

**Anything odd?** Write it here:

_(nothing yet)_

---

## 23 · The front page and progress tracker are current (m5-d15)

**What this checks:** the front page of the project now has an Install section, the list of decision records in `AGENTS.md` names the ones that exist, the npm note and the narrow-width note are correct, and the progress tracker agrees with the work. Every step here only reads files. Nothing runs Kirsch, and no key is needed.

**Before this section:** nothing needs running first. Open a Terminal window.

**Step 1 — Go to the kirsch folder.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
```
You should see your prompt again, with no message.
- [ ] Done

**Step 2 — Check the Install section on the front page.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n -E '^## Install$|doc/install.md' README.md | cut -c1-160
test -f doc/install.md && echo "doc/install.md is there"
```
You should see these lines. The numbers are line numbers in `README.md`, and the last line of output says `doc/install.md is there`.
```
23:## Install
25:> **v0.1.0 is not published yet.** Until it is, build Kirsch from a clone of the repository. The release-archive route and `go install` by release tag, which
27:Build from a clone. These are the commands from the *Build from source* section of [`doc/install.md`](doc/install.md):
36:You need Go and `git`. [`doc/install.md`](doc/install.md) gives the Go version. The binary is written to `bin/kirsch` inside the checkout. The same page cove
doc/install.md is there
```
- [ ] The lines match

**Step 3 — Check the decision-record range in `AGENTS.md`.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'Why was X decided' AGENTS.md | cut -c1-160
ls plan/adr/
```
You should see line 21 ending in `ADRs 0001–0012`, and then the file names from `0001-go-bubble-tea.md` to `0012-execution-seam.md`, followed by `README.md` and `TEMPLATE.md`.
- [ ] The lines match

**Step 4 — Check the npm note in `README.md`.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'npm run screens' README.md | cut -c1-160
grep -n 'python3 scripts/lint-screens.py' README.md | cut -c1-160
grep -c 'every script is one line' README.md
```
You should see line 167, which ends with the words `npm run screens` runs, then line 168, which reads `python3 scripts/lint-screens.py`. The last line should be `0`, which means the old wording is gone.
- [ ] The lines match

**Step 5 — Check the narrow-width note in `CHANGELOG.md`.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'corrected 2026-11-06, m5-d15' CHANGELOG.md | cut -c1-100
```
You should see line 22, which starts `- **Milestone 4, m4-d9 — Token and Cost Tracking.**`. The correction sits after the original sentence, and the original is still there.
- [ ] The line matches

**Step 6 — Check the Providers row in `README.md`.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n '^| Providers' README.md
```
You should see line 69, which names `opencode` and `anthropic` as the built-in endpoints.
- [ ] The line matches

**Step 7 — Check the progress tracker.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n '^| m5-d15 |' plan/PROGRESS.md
grep -o 'm5-d15 is done ([^)]*)' plan/milestones/milestone-5-start-here.md
```
You should see line 292 of `plan/PROGRESS.md`. Its status cell reads `done (not yet operator-tested; section 23 ready to test)`, and its owner cell reads `mission-20261106-02`. The last line should be `m5-d15 is done (mission-20261106-02)`.
- [ ] The lines match

**Step 8 — Check the resolved marks in the tracker.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'Resolved 2026-11-06 (m5-d15)' plan/PROGRESS.md | cut -c1-60
```
You should see a line for each of these rows: the `max_turns_exceeded` row, the *Unwired settings and hooks* row, the *Configured command timeout ignored* row, the *Stale code comments* row, and the *Deferred review suggestion — `usage.json` cost field* row. The rows marked for part of their scope are the *Unwired settings and hooks* row and the *Stale code comments* row. Each row writes its open part in its own text.
- [ ] The lines match

**Step 9 — Check that a stale sentence is gone.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -c 'unless O4 changes' plan/milestones/milestone-5.md
```
You should see `0`.
- [ ] The count is 0

If a line differs from what you should see, do not tick the box. Tell Bishop which step and what the screen showed.

---

## 24 · The doctor command: state folder and lock (m5-d4b)

**What this checks:** `kirsch doctor` now shows extra lines about your test setup:

- `state` asks: can kirsch save your sessions in its folder?
- `lock` asks: is another kirsch already open in this project?

Doctor only looks. It changes nothing, it does not disturb a kirsch that is already running, and it needs no key. It never sends anything to a model. The `state` line can say `fail`. That means kirsch could not save a new session in the folder. A `fail` anywhere in the report makes the `exit=` line read `exit=1` instead of `exit=0`. Step 8 (optional) shows that on purpose, using a folder in your test area.

**Before this section:** do *Before you start* in this Terminal window. This is Terminal A. Setup B must have said `Ready.`. This section also needs a second Terminal window, Terminal B, so that two kirsch runs can be open at the same time. Do not run Setup B in Terminal B. Setup B makes a fresh test area, so running it there would give Terminal B a different test area, and the two windows would not see the same kirsch.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see lines starting `>`, like these (that is npm saying what it is doing):
```
> kirsch@0.1.0-dev build
> bash scripts/build.sh
```
Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — Start kirsch in Terminal A.**
Paste:
```
cd "$T_CFG"
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
You should see kirsch's usual screen. Because no key is set, kirsch also shows a short notice that no key is set. That is expected, and kirsch keeps running and waits for you. Leave Terminal A open.
- [✅] kirsch's screen is open in Terminal A

**Step 3 — Copy the test paths into Terminal B.**
Terminal B must use the same test folders as Terminal A. That is why you copy the paths across.

Paste this in Terminal A. It only prints text:
```
printf 'export T_CFG=%s\nexport T_STATE=%s\nexport WORKSPACE=%s\nexport KIRSCH_BIN=%s\n' "$T_CFG" "$T_STATE" "$WORKSPACE" "$KIRSCH_BIN"
```
You should see four lines, each starting with `export`. Select them and copy them.

Open a second Terminal window, Terminal B. In Terminal B, paste the four lines you copied. Then paste these three lines:
```
export XDG_CONFIG_HOME="$T_CFG"
export XDG_STATE_HOME="$T_STATE"
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY KIRSCH_LIVE_MODEL
```
Nothing is printed. Do not paste a key anywhere.

Optional check: paste `echo "$XDG_STATE_HOME"` in Terminal B. It should print the same path as the `export T_STATE=` line you pasted from Terminal A.
- [✅] I pasted the four `export` lines and the three lines in Terminal B, and I did not run Setup B there

**Step 4 — Ask doctor while kirsch is running (Terminal B).**

Paste:
```
cd "$T_CFG"
"$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
```
Among the lines doctor prints, you should see these:
- `warn  lock: held by another kirsch in this workspace; a new start would skip resume and session grants`. This means doctor noticed the kirsch open in Terminal A. That is what should happen.
- `ok  state: <path> exists and is writable`. This means kirsch can save sessions in your test folder. Compare the start of the path with the `T_STATE` path from step 3. The path starts with that same folder, then `/sessions/`, then a short code of 12 letters and digits.

A line starting `doctor:` counts the results. The very last line is `exit=0`. That means doctor found no `fail`. A `warn` still gives `exit=0`.
- [✅] I saw the `warn  lock:` line, the `ok  state:` line in my test folder, and `exit=0`

**Step 5 — Check that Terminal A is not disturbed.**
Look at Terminal A. kirsch's screen should still be open. In Terminal B, paste the step 4 command again.
You should see the same `warn  lock:` line again.
- [✅] kirsch in Terminal A is still open, and the lock line still says another kirsch is running

**Step 6 — Quit kirsch, then look again.**
**During this step and step 8, do not start kirsch in another window.** With kirsch closed, doctor holds the lock for an instant while it looks. A kirsch started at that exact instant could start without resuming your last session.
In Terminal A, type `/quit` and press Enter. kirsch closes and the prompt comes back. In Terminal B, paste the step 4 command again.
You should see `ok  lock: free` and `exit=0`. The line says `free` because kirsch is closed. A small lock file stays in your test folder after kirsch closes. That is normal.
- [✅] I saw `ok  lock: free` and `exit=0`

**Step 7 — Check the state folder is in your test area.**
Paste this in Terminal B:
```
ls "$T_STATE/sessions"
```
You should see a short list of names, each 12 letters and digits long. The short code at the end of the `ok  state:` path from step 4 (the part after `/sessions/`) should be one of them.
- [✅] The folder name matches the one on the `ok  state:` line

**Step 8 — Optional: make a test folder unwritable, then put its permissions back (Terminal B, after step 6).**
Kirsch must be closed first, so do this after step 6. The block below makes the folder `$T_STATE` read-only for a moment, runs doctor, then puts the folder's permissions back. Paste this block once only. If something interrupts it and you paste it again, the folder can stay read-only. The repair lines after the expected output fix that. Paste:
```
cd "$T_CFG"
M=$(stat -f %Lp "$T_STATE")
chmod 500 "$T_STATE"
"$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
chmod "$M" "$T_STATE"
ls -ld "$T_STATE"
```
You should see a line that reads `fail  state: <path> is not writable, so index.json cannot be updated; kirsch cannot create a new session, so any start that needs one fails until this is fixed`, where `<path>` is the full path of your `$T_STATE` folder. The summary line ends `1 fail`, and the `exit=` line after it reads `exit=1`. The last line shows the folder's permissions again, and it starts `drwx`.

If the last line starts `dr-x` instead, paste these lines, and then check the last line again:
```
chmod 700 "$T_STATE"
ls -ld "$T_STATE"
```
- [✅] (optional) I saw the `fail  state:` line, `exit=1`, and the folder's mode back to `drwx`

**Anything odd?** Write it here:

_(nothing yet)_

---

## 25 · Project files (contributing, security, notices) and a note about reference screens (m5-d12)

**What this checks:** the three project files that a newcomer or a person reporting a security problem reads first. `CONTRIBUTING.md` says how to build, test and check a change. `SECURITY.md` says how to report a security problem, and what kirsch does and does not protect against. `NOTICE` names the libraries kirsch uses and their licences. The section also checks the note on how the reference screens are regenerated. Step 6 changes a setting on GitHub. Step 7 runs commands on a throwaway copy of the project. Nothing else changes anything.

**Before this section:** do *Before you start* in this Terminal window. Setup B must have said `Ready.`.

Steps 1 to 5 work from the kirsch folder. Step 7 works from a copy in your throwaway area, and it comes back to the kirsch folder at the end.

**Step 1 — Go to the kirsch folder.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
```
You should see your prompt again, with no message.
- [ ] Done

**Step 2 — Read the three new files.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
open -t CONTRIBUTING.md
open -t SECURITY.md
open -t NOTICE
```
Each file opens in the app your Mac uses for plain text, because of the `-t` flag. You should see:
- in `CONTRIBUTING.md`, a table of commands under *Build, test and check*;
- in `SECURITY.md`, the headings *Reporting a vulnerability*, *Threat model* and *Known limits*;
- in `NOTICE`, the line *Third-party software*, then a list of libraries, each with its licence.

Read them. You run nothing from them yet; that starts in step 3.
- [ ] I opened all three files and read them

**Step 3 — The regeneration note names the right flag.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'KIRSCH_REGEN=1' CONTRIBUTING.md
grep -n -e '-update' CONTRIBUTING.md plan/spec/kirsch-plan.md plan/spec/ui-spec-v0.1.md
printf 'run with -update\n' | grep -n -e '-update'
git show 0d6bbe4:plan/spec/kirsch-plan.md | grep -c -e '-update'
git show 0d6bbe4:plan/spec/ui-spec-v0.1.md | grep -c -e '-update'
```
You should see:
- The first command prints a line that contains `KIRSCH_REGEN=1`.
- The second command prints nothing. That means no old `-update` flag is left in the three files.
- The third command prints `1:run with -update`. It is the control: it shows the second command would have printed a line if the old flag were there.
- The last two commands each print `1`. They read the spec as it was before this change, so the old flag was there and the check means something.
- [ ] The first command showed `KIRSCH_REGEN=1`, the second printed nothing, the third printed `1:run with -update`, and the last two each printed `1`

**Step 4 — The security notes say what they should.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'advisories/new' SECURITY.md
grep -n '7 days' SECURITY.md
grep -n 'latest release' SECURITY.md
grep -n 'full request body' SECURITY.md
grep -n 'An approved command' SECURITY.md
printf 'An approved command\n' | grep -n 'An approved command'
```
You should see one line from each of the first four commands: the reporting link, the seven-day reply line, the line about the latest release, and the line about the full request body in the debug log. The fifth command prints nothing, because the old sentence about approved commands has gone. The sixth command is the control, and it prints `1:An approved command`. That shows the fifth command would have printed a line if the old sentence were there.
- [ ] Four lines, nothing from the fifth command, and `1:An approved command` from the sixth

**Step 5 — Every library the program uses is in NOTICE.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
awk '/^require \(/{f++} f==1&&/^\t/{print $1}' go.mod | while read m; do grep -q -F "$m" NOTICE && echo "ok $m" || echo "MISSING $m"; done
```
Every line should start with `ok`, followed by the name of a library. A line that starts with `MISSING` means a library kirsch uses is not in `NOTICE`. Stop there and tell Bishop the section and step number.

The control plants a made-up library that `NOTICE` does not list. Paste:
```
printf 'require (\n\tgithub.com/example/planted v1.0.0\n)\n' > "$T_CFG/planted.mod"
awk '/^require \(/{f++} f==1&&/^\t/{print $1}' "$T_CFG/planted.mod" | while read m; do grep -q -F "$m" NOTICE && echo "ok $m" || echo "MISSING $m"; done
```
It should print `MISSING github.com/example/planted`.
- [ ] Every line from the first loop starts with `ok`, and the control prints `MISSING github.com/example/planted`

**Step 6 — Turn on private vulnerability reporting (you do this on GitHub).**
This needs your GitHub login as the owner of the repository. The crew cannot do it for you.
1. Open the repository page, `https://github.com/djm56/kirsch`, in your browser, signed in as the owner.
2. Click *Settings*. In the left-hand list, under *Security*, click *Advanced Security*. (GitHub moves this page from time to time. If you do not see it, look for *Code security* in the same list.)
3. Find *Private vulnerability reporting* and click *Enable*.
4. Check it from outside. Open a private (incognito) browser window, where you are not signed in to GitHub, and go to `https://github.com/djm56/kirsch/security`. You should see a button labelled *Report a vulnerability*. Before you turn the setting on, that button is not there, so this check shows whether the setting took effect.

Until this is on, the link in `SECURITY.md` does not open the report form.
- [ ] I turned on private vulnerability reporting, and a signed-out window shows the *Report a vulnerability* button

**Step 7 — Run the commands from CONTRIBUTING on a clean copy (after you commit).**
This step copies the project from its last commit, so commit your changes first. Changes that are not committed are not in the copy.

Read these warnings before you start:
- `npm run tools` installs the latest versions of `govulncheck`, `gosec`, `staticcheck`, `golangci-lint` and `osv-scanner` into your Go bin folder, replacing any copies already there. `npm run ci:local` does the same for `govulncheck` and `gosec`. Other programs on your Mac that use these tools get the new versions too.
- `npm run fuzz` runs each fuzz target for about 30 seconds, so it takes several minutes in all.
- `npm start` opens kirsch. Type `/quit` to leave. It sends nothing to a model.

Run one line at a time. Wait for its `exit=` line before you paste the next one. Each must end with `exit=0`:
```
git clone -q /Volumes/DATA/Github/kirsch "$T_CFG/clean"; echo "exit=$?"
cd "$T_CFG/clean"
npm run tools; echo "exit=$?"
npm run build; echo "exit=$?"
npm start; echo "exit=$?"
npm test; echo "exit=$?"
npm run test:race; echo "exit=$?"
npm run check; echo "exit=$?"
npm run security; echo "exit=$?"
npm run fuzz; echo "exit=$?"
npm run ci:local; echo "exit=$?"
cd /Volumes/DATA/Github/kirsch
```
If a line prints anything other than `exit=0`, stop and tell Bishop the section, the step and the line. `npm start` waits for you: type `/quit` to leave, and then its `exit=` line prints.

Then tidy up the throwaway copy. This line is optional:
```
rm -rf "${T_CFG:?}/clean"
```
- [ ] Every command ended with `exit=0`, kirsch opened and closed on `/quit`, and I went back to the kirsch folder

**Anything odd?** Write it here:

_(nothing yet)_

---

## 26 · `/new` while busy (only if you choose to do it) (m5-d17b)

**What this checks:** what happens when you press Esc during a reply, and then type `/new` while that cancelled reply is still finishing. Steps 1 and 2 run tests. You do not type anything into Kirsch in this section. No model is called and no API key is used.

**Before this section:** do *Before you start* in this Terminal window. Each step starts in the kirsch folder.

**Step 1 — Run the busy-state tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/tui/ -run 'TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages$|TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction$|TestNewSessionDoneEndsBusyAfterLateTurnMessages$' -v; echo "exit=$?"
```
You should see `=== RUN` and `--- PASS` lines for each test named in the plain words below, the `ok` line, and `exit=0` as the last line. Here is the output of a run on 2026-11-05, copied as it was printed. Your times will differ a little, and that is normal:
```
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/control:_UsageMsg_touches_no_busy_state
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCompleteMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCancelledMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnErrorMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ErrorMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ManualCompactMsg_(delivered_directly,_no_/compact_run)
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ToolStartedMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/AssistantTextDeltaMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/CompactingMsg
=== RUN   TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/positive_control:_no_/new_pending,_Enter_reaches_Submit
--- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages (0.05s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/control:_UsageMsg_touches_no_busy_state (0.01s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCompleteMsg (0.01s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnCancelledMsg (0.01s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/TurnErrorMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ErrorMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ManualCompactMsg_(delivered_directly,_no_/compact_run) (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/ToolStartedMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/AssistantTextDeltaMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/CompactingMsg (0.00s)
    --- PASS: TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages/positive_control:_no_/new_pending,_Enter_reaches_Submit (0.00s)
=== RUN   TestNewSessionDoneEndsBusyAfterLateTurnMessages
=== RUN   TestNewSessionDoneEndsBusyAfterLateTurnMessages/success
=== RUN   TestNewSessionDoneEndsBusyAfterLateTurnMessages/failure
--- PASS: TestNewSessionDoneEndsBusyAfterLateTurnMessages (0.01s)
    --- PASS: TestNewSessionDoneEndsBusyAfterLateTurnMessages/success (0.00s)
    --- PASS: TestNewSessionDoneEndsBusyAfterLateTurnMessages/failure (0.00s)
=== RUN   TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction
--- PASS: TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/tui	0.689s
exit=0
```

In plain words, the lines above mean this:

- `TestEnterWhileNewSessionPendingStaysSwallowedAfterLateTurnMessages`: while `/new` is waiting, a late message from the cancelled reply does not clear the busy state, and Enter does not send your typed text to the app. There is one subtest for each kind of late message: `TurnCompleteMsg`, `TurnCancelledMsg`, `TurnErrorMsg`, `ErrorMsg`, `ManualCompactMsg`, `ToolStartedMsg`, `AssistantTextDeltaMsg` and `CompactingMsg`. `ManualCompactMsg` is delivered directly, not a late message from the cancelled reply. The `control` line sends a message that should change nothing. The `positive control` line checks that, when `/new` is not waiting, Enter does send the text, so the other lines cannot pass just because Enter never works.
- `TestManualCompactMsgDuringNewAfterBusyClearedMidCompaction`: it runs `/compact`, then a session-log save error clears busy while the compaction is still going. Then `/new` is typed and Enter is pressed. When the compaction's own message arrives, the test checks that `/new` is still waiting, the busy state still says "new session", the compaction notice shows once, and Enter does not reach the app. Then the message that ends `/new` clears busy.
- `TestNewSessionDoneEndsBusyAfterLateTurnMessages`: the message that ends `/new` clears the busy state even after late messages have arrived. There are two cases. `success` empties the transcript. `failure` keeps the transcript, and the late error box is still there.

A `--- FAIL` line means the check did not pass. Copy the last lines and tell Bishop.
- [ ] I saw `--- PASS` for each test named in step 1, the `ok` line, and `exit=0` as the last line, with no `--- FAIL` line

**Step 2 — Run the session-fence tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 ./internal/app/ -run 'TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns$|TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne$|TestTurnPathsUseOnlyFencedSenders$' -v; echo "exit=$?"
```
You should see `=== RUN` and `--- PASS` lines for each test named in the plain words below, the `ok` line, and `exit=0` as the last line. Here is the output of a run on 2026-11-05, copied as it was printed. Your times will differ a little, and that is normal:
```
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/BudgetMsg
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/AssistantTextDeltaMsg
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/final_turn_message
=== RUN   TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/Submit_refusal
--- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns (1.66s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/BudgetMsg (0.38s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/AssistantTextDeltaMsg (0.52s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/final_turn_message (0.47s)
    --- PASS: TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns/Submit_refusal (0.29s)
=== RUN   TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne
--- PASS: TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne (0.22s)
=== RUN   TestTurnPathsUseOnlyFencedSenders
=== RUN   TestTurnPathsUseOnlyFencedSenders/package_app
=== RUN   TestTurnPathsUseOnlyFencedSenders/control
--- PASS: TestTurnPathsUseOnlyFencedSenders (0.01s)
    --- PASS: TestTurnPathsUseOnlyFencedSenders/package_app (0.01s)
    --- PASS: TestTurnPathsUseOnlyFencedSenders/control (0.00s)
PASS
ok  	github.com/djm56/kirsch/internal/app	2.406s
exit=0
```

In plain words, the lines above mean this:

- `TestNewSessionDeliversNothingFromTheOldTurnAfterItReturns`: a reply is started and cancelled, then `/new` runs. Once `/new` has returned, nothing from the old reply shows up in the test's record of messages. The subtests are `BudgetMsg`, `AssistantTextDeltaMsg`, `final_turn_message` and `Submit_refusal`. For `Submit_refusal`, the endpoint is unknown, so no reply is started, and the refusal message must not arrive after `/new`. Each one has a check that the message really exists at that point, so an empty result afterwards means the message was held back, not that nothing was sent.
- `TestEverySenderDropsAStaleGenerationAndDeliversACurrentOne`: after `/new`, it sends one message through each sender that carries the old session's number, and one through the same sender with the new number. The old message must not arrive. The new one must arrive, which shows that the sender works at all.
- `TestTurnPathsUseOnlyFencedSenders`: it reads the Go source of the app package, not the tests, and checks that the functions in package app that send a turn's messages use the checked senders, unless the function is on a short list of known exceptions. Its own comment lists what it cannot see. It does not run the program. Its `control` line plants three kinds of bad call and one allowed one in a small piece of text, and checks that the test reports the three bad ones.

- [ ] I saw `--- PASS` for each test named in step 2, the `ok` line, and `exit=0` as the last line, with no `--- FAIL` line

**What changed for you (read from the code and tests, not seen in a run).** When you press Esc during a reply and then type `/new` while that reply is still finishing, the status bar stays on "new session" until the new session is ready. Enter does not send your text, and no error box appears, until then. Messages from the old reply no longer appear in the new session; an approval answer from the old reply is not covered (see *Open from m5-d17b*, S-4, in `plan/PROGRESS.md`). The status bar reads the busy verb through `stateVerb` (`internal/tui/statusbar.go:196`, used at `:52`); `/new` sets it at `internal/tui/update.go:1204`; Esc reaches `cancelTurn` at `internal/tui/update.go:792`.

**Anything odd?** Write it here:

_(nothing yet)_

---

## 27 · The troubleshooting guide and the list of docs pages (m5-d14b)

**What this checks:** the troubleshooting guide explains what you see when kirsch will not start or cannot send a message, and the list of docs pages names the pages that exist. Each step makes one of those problems happen, then checks the guide's advice against what kirsch prints.

**Before this section:** do *Before you start* in this Terminal window.

Steps that need a particular folder start with a `cd` line. The others work from any folder, because they point kirsch at the test project with `--workspace`. Steps 7 and 8 open kirsch in a window; type `/quit` to leave it.

**Step 1 — Build kirsch.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
npm run build
```
You should see lines starting with `>` (that is npm saying what it is running). Lines starting `go: downloading` or `npm notice` are also fine. Anything else means something went wrong.
- [✅] Done, no errors

**Step 2 — The pages have no planning words and no code references.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n -i -w -E 'milestones?|ADRs?' doc/troubleshooting.md doc/README.md
printf 'an ADR\nADRs\nmilestones\nheadroom\n' | grep -n -i -w -E 'milestones?|ADRs?'
grep -n -E '\.go\b|internal/|cmd/kirsch/|Sources?:' doc/troubleshooting.md doc/README.md
printf 'cmd/kirsch/main.go:85\ninternal/doctor/probes.go:138\n*Source:* x\n' | grep -n -E '\.go\b|internal/|cmd/kirsch/|Sources?:'
```
The first search should print nothing, because the guide is for people using kirsch and does not talk about planning. The second command proves the search does catch those words. It prints each matching line with its line number in front: `1:an ADR`, `2:ADRs` and `3:milestones`, and no line for `headroom`. The third search should print nothing. The fourth command proves that search works too (made-up examples). It prints `1:cmd/kirsch/main.go:85`, `2:internal/doctor/probes.go:138` and `3:*Source:* x`.
- [✅] The searches printed nothing, and the test commands printed the lines named above

**Step 3 — The docs list matches the files.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
sed -n '/^| Document/,/^$/p' doc/README.md
ls doc/
```
You should see the table of docs pages. Every page row has `Written` in the last column, and every file in the list other than `README.md` has a row. Every file the table names appears in the list from `ls doc/`, and so does `README.md`.
- [✅] Every row says `Written`, and each file named in the table is there

**Step 4 — No API key: kirsch says so, and the guide explains it.**
Paste, in a window where no key is set:
```
keycount_control
keycount
"$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
```
You should see `CONTROL PASS: a planted key line counts as 1`, then `0`. If the line after `CONTROL PASS` is not `0`, a key is set in this window, so stop and clear it first (the clear line at the end of this section does it).

The doctor run should print a line that reads `warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt`, and then `exit=0`. Other lines appear too, and they are fine.

Then open `doc/troubleshooting.md` and find the section headed *No API key*. Its text should name `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY`.
- [✅] I saw the `key` warning and `exit=0`, and the *No API key* section names the variables above

**Step 5 — Not in a repository.**
Paste:
```
cd "$T_CFG"
git rev-parse --show-toplevel 2>/dev/null || echo "not in a repository"
```
The second line should print `not in a repository`. If it prints a folder path instead, stop: that folder is inside a repository, and this step cannot test the case. Then paste `cd /Volumes/DATA/Github/kirsch` to go back to the kirsch folder.

If it printed `not in a repository`, paste:
```
"$KIRSCH_BIN"; echo "exit=$?"
"$KIRSCH_BIN" doctor; echo "exit=$?"
cd /Volumes/DATA/Github/kirsch
```

You should see, from the first kirsch run, `kirsch: Kirsch runs inside a Git repository.`, a blank line, `Either cd into a repository, or point Kirsch at a directory explicitly:`, `    kirsch --workspace /path/to/project`, and then `exit=1`. From doctor, you should see a line that reads `fail  repository: not in a Git repository`, then `exit=1`. Doctor prints other lines too, and they are fine. Then open *Not in a repository* in the guide. It should quote the message above and say to `cd` into a repository or to use `--workspace`.
- [✅] I saw the message, the `repository` fail line and `exit=1` each time, and *Not in a repository* quotes the message

**Step 6 — Missing ripgrep: kirsch still works, with warnings.**
Paste:
```
NO_RG_PATH="$(dirname "$(command -v git)"):/usr/bin:/bin"
PATH="$NO_RG_PATH" command -v rg || echo no-rg
PATH="$NO_RG_PATH" "$KIRSCH_BIN" doctor --workspace "$WORKSPACE"; echo "exit=$?"
```
The second line should print `no-rg`. If it prints a path, stop: ripgrep sits in the folder where git lives, and this step cannot test the case.

You should see these warning lines from doctor: `warn  rg: not found on PATH; search_code uses the pure-Go fallback` and `warn  backend: search_code uses pure-go (rg is not on PATH)`. Then `exit=0`.

Then open *Missing ripgrep* in the guide. It should quote the warning lines above and say how to install ripgrep.
- [✅] The warning lines appeared, `exit=0` appeared, and *Missing ripgrep* quotes them

**Step 7 — (Optional) The no-key error card. The crew could not run this.**
Paste, in a window where no key is set (check with the key count in step 4 first):
```
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
Type `hi` and press Enter. Kirsch refuses before it sends anything, so nothing reaches the provider.

You should see an error card titled `auth`, reading `no API key for endpoint "opencode" (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY)`. Type `/quit` to leave. Then open *No API key* in the guide. It should show the same card text.
- [✅] I saw the `auth` card with the variable names shown above, and *No API key* shows the same text
- [ ] (not yet run) Re-check the first claim of the *No API key* paragraph in the guide, against the screen. In a new Terminal window with no key set, first set this window to 80 columns wide and 24 rows tall. In Terminal, the Window settings for the profile hold the Columns and Rows fields. Then start a fresh session with this command, which uses a new temporary folder and a new temporary home: `FAKE_HOME="$(mktemp -d)" && cd "$(mktemp -d)" && git init -q . && env -i HOME="$FAKE_HOME" PATH="$PATH" TERM=xterm-256color /Volumes/DATA/Github/kirsch/bin/kirsch`. The onboarding should be on screen, with the `no previous session` notice beneath it. A much shorter window shows the ordinary transcript instead of the onboarding. That is expected, not a failure. Quit with two `Ctrl+C`. Send no message.

**Step 8 — (Optional) The rejected-key card. The crew could not run this.**
Paste:
```
export OPENCODE_API_KEY=dummy-invalid-key
"$KIRSCH_BIN" --workspace "$WORKSPACE"
```
Type `hi` and press Enter. This contacts the provider, which refuses the dummy key. Never put a real key here.

You should see an error card titled `turn_error`. Its text includes `provider "opencode" rejected the API key (HTTP 401)`, or the same line with `HTTP 403` if the provider answers that way, and names the variables shown above. Type `/quit` to leave. Then open *401 or 403: rejected the API key* in the guide. It should show the same message and say the key is not always the cause.

Then clear every key and check the count:
```
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
keycount_control
keycount
```
You should see `CONTROL PASS: a planted key line counts as 1`, then `0`.
- [✅] I saw the `turn_error` card with HTTP 401 or 403, the guide's *401 or 403* section matches it, and the key count reads `0`

The lock case, which the guide also covers, is in section 24. It starts a second kirsch, so this section does not repeat it.

**If you stopped part way through step 8 — clear the keys from this window.**
```
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
```

**Anything odd?** Write it here:

_(nothing yet)_

---

## 28 · Resume totals and percent full (only if you choose to do it) (m5-d18)

**What this checks:** two things. When you reopen an earlier session, the totals in the status bar are that session's own totals, straight away. After you use `/compact`, the percent full that `/status` shows changes to match. You do not type anything into Kirsch in this section. Step 1 runs tests only. No model is called, no API key is used, and Kirsch is not started.

**Before this section:** do *Before you start* in this Terminal window. Step 1 starts in the kirsch folder.

**Step 1 — Run the resume, session and compact tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 -v -run 'TestResumeLastSession|TestResumeCompletesPartialTurn' ./internal/app/; echo exit=$?
go test -count=1 -v -run 'TestLoadUsageTotals' ./internal/session/; echo exit=$?
go test -count=1 -v -run 'TestManualCompactUpdatesStatusCompactionPercent' ./internal/app/; echo exit=$?
```
You should see, in this order, the lines below. Each `exit=` line comes straight after the `ok` line it belongs to. Your times will differ a little, and that is normal:
```
ok  	github.com/djm56/kirsch/internal/app	1.189s
exit=0
ok  	github.com/djm56/kirsch/internal/session	0.638s
exit=0
ok  	github.com/djm56/kirsch/internal/app	0.384s
exit=0
```
Above those lines you should also see `--- PASS` lines for each test. The last line of each run is `exit=0`. In the output of the third command you should also see the two lines `compaction % before /compact: 60` and `compaction % after /compact: 37 (estimate 9727 tokens)`. A `--- FAIL` line means a check did not pass. Copy the last lines and tell Bishop.

- [ ] I ran the three commands in step 1, I saw `ok` and `exit=0` after each one, I saw no `--- FAIL` line, and I saw both `compaction % before /compact: 60` and `compaction % after /compact: 37 (estimate 9727 tokens)` in the output of the third command

---

## 29 · Record your decisions (m5-d20)

Not ready yet. This section will be written in the same step-by-step style when m5-d20 is finished.

---

## 30 · The list of AI models (only if your decision calls for it) (m5-d21a)

**What this checks:** the prices Kirsch uses to work out what a session costs. One price changed. Reading saved context with Claude Sonnet 5.5 is now $0.10 per million tokens, not $0.20, to match Anthropic's pricing page as it read on 2026-11-05. You do not type anything into Kirsch in this section. Step 1 runs tests only. It uses no key, no model is called, and Kirsch is not started.

**Before this section:** do *Before you start* in this Terminal window. Step 1 starts in the kirsch folder.

**Step 1 — Run the price tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 -v -run 'TestPublishedPerTokenFigures|TestCostPerTokenModel|TestCostAllPerTokenModels' ./internal/provider/; echo exit=$?
```
You should see, at the end, these two lines. Your times will differ a little, and that is normal:
```
ok  	github.com/djm56/kirsch/internal/provider	0.640s
exit=0
```
Above those lines you should also see `--- PASS` lines for each test. The last line is `exit=0`. A `--- FAIL` line means a check did not pass. Copy the last lines and tell Bishop.

- [ ] I ran the command in step 1, I saw `ok` and `exit=0`, and I saw no `--- FAIL` line

---

## 31 · The help footer and saved pictures of the screen (only if your decision calls for it) (m5-d21b)

**What this checks:** the last line of the Help box, the box you see when you press `?`. That line used to say `docs: doc/usage.md`. An installed Kirsch does not have that folder, so the line now says `docs: github.com/djm56/kirsch`. At 80 columns the link does not show. It shows when the window is wide enough to fit the whole line. Step 1 runs tests and the screen check. Step 2 is a look at the line that changed. You do not type anything into Kirsch in this section. It uses no key, no model is called, and Kirsch is not started.

**Before this section:** do *Before you start* in this Terminal window. Step 1 starts in the kirsch folder.

**Step 1 — Run the help and screen tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 -v -run 'TestHelpFooterNamesScrollKeysWhenClipped' ./internal/tui/; echo exit=$?
go test -count=1 ./internal/tui/; echo exit=$?
npm run screens; echo exit=$?
```
You should see lines like these in the output. The sample below shows selected lines from the runs, not every line. Your times will differ a little, and that is normal:
```
--- PASS: TestHelpFooterNamesScrollKeysWhenClipped (0.04s)
ok  	github.com/djm56/kirsch/internal/tui	0.705s
exit=0
ok  	github.com/djm56/kirsch/internal/tui	9.723s
exit=0
kirsch-ui-screens.md: grids OK, content OK
exit=0
```
A run that passes ends with the line `exit=0`. A `--- FAIL` line means a check did not pass. Copy the last lines and tell Bishop.

- [ ] I ran the commands in step 1, I saw an `exit=0` line after every command, `ok` after each test command and `grids OK` after the screen check, and I saw no `--- FAIL` line

**Step 2 — Look at the line that changed.**
The saved picture of the Help box is in `plan/spec/kirsch-ui-screens.md`, under the heading `06 · Help overlay`. It is drawn 80 columns wide and 28 rows tall. Its footer line, the line under the divider, changed. In plain words:

- Before: the line said `kirsch v0.1.0-dev · docs: doc/usage.md · Esc or ? closes`.
- After: the line says `kirsch v0.1.0-dev · Esc or ? closes`.

So `docs: doc/usage.md` is gone from the picture, and the new link does not take its place. At 80 columns the picture shows no link. It shows one when the window is wide enough to fit the whole line.

Bishop's own diff of the saved picture before this change against the tree now, with the removed line marked `-` and the added line marked `+`:
```
-         │ kirsch v0.1.0-dev · docs: doc/usage.md · Esc or ? closes   │
+         │ kirsch v0.1.0-dev · Esc or ? closes                        │
```

O16 decided that the help footer's `docs: doc/usage.md` becomes `docs: github.com/djm56/kirsch`, because an installed program does not have the `doc/` folder. Tick the box below only if the change above is what you decided. The 80-column picture above shows no link. That is a fact to look at, not something O16 decided. Whether the 80-column picture should keep a link is open: see *Open from m5-d21b* item a in `plan/PROGRESS.md`. Bishop has a recommendation there, for you to decide.

- [ ] The change above is what I decided under O16: the help footer's docs line now reads `docs: github.com/djm56/kirsch` instead of `docs: doc/usage.md`

Question for you (O23): your answer goes in the decisions table in `plan/PROGRESS.md`. The O23 row there reads `Go with a`. Bishop read `a` as regenerate the moved pictures, then have a person review the golden diff. Please confirm that reading; see *Open from m5-d21b* item d in `plan/PROGRESS.md`.

---

## 32 · The size limit on the debug log (only if your decision calls for it) (m5-d21c)

Not ready yet. This section will be written in the same step-by-step style when m5-d21c is finished.

---

## 33 · The notice about the debug log (only if your decision calls for it) (m5-d21d)

**What this checks:** the notice about the debug log in `SECURITY.md`, the project's security notes. The debug log is off unless you turn it on. When it is on, it holds the text of each message you send, in full. It also holds the workspace folder path. When a configuration warning occurs, it holds the configuration file path too. It holds the command arguments of a session approval that is refused. It does not record each tool call or each slash command. The notice says the 400 request is a disclosed limit, and that the request is not cut short. When a provider turns a request away with a 400 error (a request the provider says is badly formed), the log holds the whole request. A 400 error that names `MissingSessionID` is reported as a configuration error, and it is not written to the log. A secret typed into a message, or that appears in a path or command, is written to the log too. Steps 1 to 3 search the file for the notice, for the exception and for the message part. They check the wording only. They use no key, no model is called, and Kirsch is not started.

**Before this section:** do *Before you start* in this Terminal window. Step 1 starts in the kirsch folder.

**Step 1 — Find the notice in the file.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'full request body' SECURITY.md; echo exit=$?
```
You should see a line of text, then `exit=0`. Your line number can differ if the file has changed since this was written. The words after the number should be the same:
```
100:  full request body. This is a disclosed limit, and the body is not clipped. It
exit=0
```
The last line is `exit=0`. If you see `exit=1` instead, or no line above it, copy the output and tell Bishop.

**Step 2 — Find the exception in the file.**
Paste (still in the kirsch folder):
```
grep -n 'MissingSessionID' SECURITY.md; echo exit=$?
```
You should see a line of text that names `MissingSessionID`, then `exit=0`. The words after the number should be the same:
```
105:  `MissingSessionID` is the exception: Kirsch reports it as a configuration
exit=0
```
The last line is `exit=0`. If you see `exit=1` instead, or no line above it, copy the output and tell Bishop.

**Step 3 — Find the message part of the notice.**
Paste (still in the kirsch folder):
```
grep -n 'writes the text of each message you send' SECURITY.md; echo exit=$?
```
You should see a line that names `writes the text of each message you send`, then `exit=0`. The words after the number should be the same:
```
84:  log writes the text of each message you send, in full. It also writes the
exit=0
```
The last line is `exit=0`. If you see `exit=1` instead, or no line above it, copy the output and tell Bishop.

- [ ] I ran the commands in steps 1, 2 and 3. The line from step 1 contains the word `disclosed`, the line from step 2 contains `MissingSessionID`, the line from step 3 contains `writes the text of each message you send`, the last line I saw after each was `exit=0`, and I did not edit `SECURITY.md`

---

## 34 · A library that makes unique IDs (only if your decision calls for it) (m5-d21e)

Not ready yet. This section will be written in the same step-by-step style when m5-d21e is finished.

---

## 35 · A written change to the specification (its section 7) (only if your decision calls for it) (m5-d21f)

**What this checks:** the list of session events in section 7 of the specification, `plan/spec/kirsch-plan.md`. Kirsch writes its conversation to a session file, with a line for each event. Section 7 now lists only the kinds of line Kirsch writes. Step 1 lists the kinds the code writes. Step 2 searches section 7 for an event that was taken off the list, and expects no match. Step 3 searches for an event that is still on the list, as a control, and expects a match. The steps only read files. They use no key, no model is called, and Kirsch is not started.

**Before this section:** do *Before you start* in this Terminal window. Step 1 starts in the kirsch folder.

**Step 1 — List the kinds of event the code writes.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
grep -n 'Type: *eventType' internal/session/event.go; echo exit=$?
```
You should see a line for each kind below, then `exit=0`. The kinds are: user message, assistant message, assistant thinking, tool completed, turn started, turn completed, usage, approval scope granted, approval scopes cleared, and compaction applied. The user message kind is on the lines that start `181:` and `211:`. The line numbers can differ if the file has changed since this was written. The rest of each line should be the same:
```
181:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
211:			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeUserMessage},
225:				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeToolCompleted},
261:				baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantThinking},
284:			baseEvent: baseEvent{Version: eventVersion, Type: eventTypeAssistantMessage},
308:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnStarted},
316:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeTurnCompleted},
324:		baseEvent:    baseEvent{Version: eventVersion, Type: eventTypeUsage},
335:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeApprovalScopeGranted},
344:		baseEvent: baseEvent{Version: eventVersion, Type: eventTypeApprovalScopesCleared},
351:		baseEvent:    baseEvent{Version: eventVersion, Type: eventTypeCompactionApplied},
exit=0
```
The last line is `exit=0`. If you see `exit=1` instead, or no lines above it, copy the output and tell Bishop.

**Step 2 — Check that a removed event is gone from section 7.**
Paste (still in the kirsch folder):
```
awk '/^## 7\. Session Events/{p=1} /^## 8\. /{p=0} p' plan/spec/kirsch-plan.md | grep -n -F '"type":"session.started"'; echo exit=$?
```
You should see no line of text, only `exit=1`. Kirsch does not write `session.started`, so the list should not show it.
```
exit=1
```
If you see a line that contains `session.started` in a JSON example, copy the output and tell Bishop.

**Step 3 — Control: a kept event is still in section 7.**
Paste (still in the kirsch folder):
```
awk '/^## 7\. Session Events/{p=1} /^## 8\. /{p=0} p' plan/spec/kirsch-plan.md | grep -n -F '"type":"turn.started"'; echo exit=$?
```
You should see a line that contains `turn.started`, then `exit=0`. The number before it can differ. The words after it should be the same:
```
6:{"v":1,"type":"turn.started","turn":3}
exit=0
```
If you see `exit=1` instead, or no line above it, copy the output and tell Bishop. Step 2 proves nothing unless this step finds its line.

- [ ] I ran steps 1, 2 and 3 in the kirsch folder. Step 1 listed a line for each kind of event named above, and its last line was `exit=0`. Step 2 printed no line, only `exit=1`. Step 3 printed a line that contains `turn.started`, and its last line was `exit=0`. I did not edit `plan/spec/kirsch-plan.md`.

---

## 36 · What happens when two copies run at once (only if your decision calls for it) (m5-d21g)

Not ready yet. This section will be written in the same step-by-step style when m5-d21g is finished.

---

## 37 · Demo and final release check (m5-d22)

Not ready yet. This section will be written in the same step-by-step style when m5-d22 is finished.
