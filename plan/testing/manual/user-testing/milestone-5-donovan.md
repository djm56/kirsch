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
- [ ] Done, no errors

**Step 2 — Ask for the report from inside the project.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
"$KIRSCH_BIN" doctor; echo "exit=$?"
```
You should see the version line, then the check lines for `git`, `rg`, `repository` and `config`, then a summary line, then `exit=0`. The check lines start with `ok`, in this order: `git`, `rg`, `repository`, `config`. The repository line shows your kirsch folder. The summary reads `doctor: 4 ok, 0 warn, 0 fail`, as it did before m5-d4a. Since m5-d4a it reads `doctor: 7 ok, 1 warn, 0 fail` with no key set (measured 2026-10-29, see the note above).
If rg is not installed on your Mac, its line starts with `warn` instead. The summary then reads `3 ok, 1 warn`, and `exit=0` still. A warning does not make doctor fail.
> **Dated note, 2026-10-29 (m5-d4a, step 3c):** the `3 ok, 1 warn` above is out of date. With `rg` off PATH, the `rg` line warns and the `backend` line warns too, and the summary counts both. Measured from the kirsch folder with no key set, under a PATH that holds only `git`, the run prints `warn  rg: not found on PATH; search_code uses the pure-Go fallback` and `warn  backend: search_code uses pure-go (rg is not on PATH)`. The summary reads `doctor: 5 ok, 3 warn, 0 fail`, and the run ends `exit=0`. The `key` warning is there because no key is set in that run. A warning still does not make doctor fail. The runs are in `.claude/memory/workspace/m20261029-06-step3c-runs.txt`, parts 7 and 8 (scratch, archived at the next mission start).
- [ ] I saw the check lines, the summary, and `exit=0`

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
- [ ] I saw the `warn` line for repository and `exit=0`

**Step 4 — The same kind of folder, with no `--workspace`.**
Paste:
```
d4=$(mktemp -d)
cd "$d4"
"$KIRSCH_BIN" doctor; echo "exit=$?"
cd /Volumes/DATA/Github/kirsch
```
You should see the repository line start with `fail` and say `not in a Git repository`. The summary reads one failure, and it ends with `exit=1`. Without `--workspace`, kirsch needs a Git project to start, so this is a failure. Step 3 and step 4 check each other: the folder is the same kind, and the only difference is the `--workspace` value. This step's folder is named `$d4`, and the tidy-up line at the end of this section removes it.
- [ ] I saw the `fail` line for repository and `exit=1`

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
- [ ] Doctor printed a `fail` line for config, and plain kirsch also gave `exit=1`

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
- [ ] I saw the `warn` line for config and `exit=0`

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

- [ ] The first command printed a number that is not `0`, and the second command ended with `exit 0`.

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
- [ ] I saw `ok` and no `FAIL`

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
- [ ] I saw `ok` and no `FAIL`

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

- [ ] I saw nothing from the first command, `grids OK, content OK` from the third, `ok` with no `FAIL` from the fourth, and the second command printed a line naming `view.go`

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
- [ ] Nothing above `exit 1`

**Step 2 — Check the spec no longer names the old setting.**
Paste:
```
grep -n 'Paste debounce' plan/spec/ui-spec-v0.1.md; echo "exit $?"
```
You should see only one line: `exit 1`. There must be nothing above it.
- [ ] Nothing above `exit 1`

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
- [ ] The first command prints the line starting `22:` then `exit 0`, and the second prints the lines starting `565:` and `758:` then `exit 0`

**Step 4 — List and count the new paste tests.**
Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -list '^TestUnbracketed' ./internal/tui | grep '^TestUnbracketed'
go test -list '^TestUnbracketed' ./internal/tui | grep -c '^TestUnbracketed'; echo "exit $?"
```
You should see two test names, then a number on its own line, then `exit 0`. The two names must be exactly `TestUnbracketed5KBPasteLandsWholeThroughKeyMessages` and `TestUnbracketedMultiLinePasteCurrentBehaviour`. The number after them must not be `0`. If no names appear, or the number is `0`, the tests were not found, and the step has not passed. Step 5 runs the tests these names come from.
- [ ] The two names on screen are the two listed above, and the number after them is not `0`

**Step 5 — Run the paste tests.**
Paste:
```
go test -count=1 -run '^TestUnbracketed' ./internal/tui; echo "exit $?"
```
It runs the tests named in Step 4, and every one of them must pass.
You should see a line starting `ok` and, as the very last line, `exit 0`. A line with `FAIL` in it means something is wrong.
- [ ] The last line says `exit 0`, and no line says `FAIL`

**Step 6 — Read the limits that apply to some Terminals.** Nothing to paste in this step. Do not start kirsch.

Normally your Terminal sends a paste to kirsch as one block, and long text lands whole. Some Terminals do not do that. Then kirsch reads the paste as if you had typed each key, and these things can go wrong:

1. **Text with line breaks.** Only the first line reaches kirsch, and kirsch sends it as your message. The rest of the paste is dropped while kirsch is busy. This is a known problem and is not fixed. The crew has written it down. Whether it is acceptable for version 0.1, or needs a fix, is your decision. It is open in `plan/PROGRESS.md`, under *Open from m5-d8*. The crew checked this with a test that feeds Enter key messages to kirsch's code, without a real terminal. The test does not run the library's reader; that a line break arrives as Enter is read from the library's source.
2. **Tabs.** A tab in your text arrives as the Tab key. What you would see: if the typing line holds a slash command name that matches exactly one command, Tab finishes the name. If the name matches several commands, or none, Tab does nothing and the rest of the paste is typed in as normal. Otherwise Tab moves the focus up to the transcript (the messages and cards above the typing line), and the line at the bottom of the screen changes. After that, the rest of the paste does not appear in the typing line. Its letters act on the transcript: `d` opens a detail view, `g` and `G` move the view, and `?` opens help. A second Tab in the same paste moves focus back to the typing line, and the text after it is typed in again, so one paste can end up split between the two. The crew read the Tab key's arrival from Bubble Tea's source and what kirsch does with it from kirsch's code. Neither was run. It is recorded in `plan/PROGRESS.md`, under *Open from m5-d8*.
3. **A rare accented or non-English letter** can be lost if the paste is cut at a terminal read boundary. The cause is in the library kirsch uses to read the keyboard, so kirsch cannot fix it alone. The crew read this from the library's source and did not test it.

- [ ] I have read the limits above

**Anything odd?** Write it here:

_(nothing yet)_

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
- [ ] Done, no errors

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
- [ ] I saw the `warn  key:` line naming both variables, the `ok  model:` line, the `ok  context: AGENTS.md` line, and `exit=0`

**Step 3 — Your own settings.** This run does not use the test settings folder, so it reads the settings you normally use.
Paste:
```
cd /Volumes/DATA/Github/kirsch
env -u XDG_CONFIG_HOME "$KIRSCH_BIN" doctor; echo "exit=$?"
```
You should see one `model:` line for the model your own settings name. If kirsch knows that model, the line starts with `ok`. If it does not, the line starts with `warn  model:` and says `not in the table; conservative fallback`. That is a warning, not a failure: kirsch then uses a conservative context size for a model it does not know. On the run Bishop made, the settings named `claude-haiku-5-5`, which kirsch does not list, so this line warned. If yours names the same model, expect the same warning.
The run ends `exit=0`.
- [ ] I saw the `model:` line for my own model, and `exit=0`

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
- [ ] The key line ends with `KIRSCH_OPENCODE_API_KEY`, the second result is `0`, and the third is `1`

**Step 5 — Confirm no key is set.** This command unsets the key variables and then counts any that are still set. The check must pass before any later section.
Paste:
```
unset ANTHROPIC_API_KEY KIRSCH_ANTHROPIC_API_KEY OPENCODE_API_KEY KIRSCH_OPENCODE_API_KEY
keycount_control
keycount
```
You should see `CONTROL PASS: a planted key line counts as 1`, then `0`. The `CONTROL PASS` line shows that the counter works, and the `0` shows that no key variable is still set.
- [ ] I saw `CONTROL PASS` and `0`

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

- [ ] The first two commands printed the same number, and the third command ended with `exit 0`.

**Step 7 — A pattern that selects nothing (on screen, nothing to save).** This shows why the count in step 6 comes first. Paste:
```
cd /Volumes/DATA/Github/kirsch
go test -count=1 -run '^NoSuchTestXYZ$' ./internal/doctor; echo "exit $?"
```
You should see one line starting `ok` that ends `[no tests to run]`, and then `exit 0`. The test command ends `exit 0` even though it ran no test, because Go counts "no tests to run" as a success. The count in step 6 is what catches that.
- [ ] I saw `[no tests to run]` on the `ok` line and `exit 0` at the end

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
- [ ] Done, no errors

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
- [ ] I saw test names, and the number was not 0

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
- [ ] I saw `ok` and `exit 0`

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
- [ ] Nothing printed by git, and I saw `grids OK, content OK` and `result: 0`

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
- [ ] The long line ends in `⋯` at the right edge of the typing area
- [ ] Esc emptied the typing area
- [ ] `/quit` closed kirsch, and I am back at my Terminal prompt

**Anything odd?** Write it here:

_(nothing yet)_

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
