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
minimax-m2.7 · idle · 0 tok
```
(`minimax-m2.7` is the model in the test setup. `0 tok` means no turn has run yet.)
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
minimax-m2.7 · idle
```
There is no token count at all.
Then type `/quit` and press Enter to close kirsch. The `stty size` output comes back on screen after you quit. It should show a second number from 40 to 79 (for example `24 60`).
(Resize the window before you paste, so kirsch starts at the width you want. kirsch also redraws if you resize it while it is open.)
- [✅] I saw `minimax-m2.7 · idle` and no token count

**Step 6 — Start kirsch on a window narrower than 40 columns.**
Make the Terminal window narrower than 40 columns. 30 columns is a good choice. Paste the same commands as in step 5. kirsch takes over the window, so the `stty size` output comes back only after you quit. It should then show a second number below 40 (for example `24 30`).
You should see a dim line that says `terminal too narrow` in the main area, and the bar should still read:
```
minimax-m2.7 · idle
```
Then type `/quit` and press Enter to close kirsch.
- [✅] I saw `terminal too narrow`, and the bar reads `minimax-m2.7 · idle`

**Step 7 — Make the window wide again.**
Make the Terminal window 80 columns or wider. Paste the same commands as in step 5. After you quit kirsch, the `stty size` output should show a second number of 80 or more (for example `24 100`).
You should see the bar read `minimax-m2.7 · idle · 0 tok` again. Then type `/quit` and press Enter to close kirsch.
- [✅] I saw `minimax-m2.7 · idle · 0 tok` again

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

Not ready yet. This section will be written in the same step-by-step style when m5-d5 is finished.

---

## 4 · Terminal size and window resizing (m5-d9)

Not ready yet. This section will be written in the same step-by-step style when m5-d9 is finished.

---

## 5 · The settings guide (m5-d13a)

Not ready yet. This section will be written in the same step-by-step style when m5-d13a is finished.

---

## 6 · Safety tests for running commands (m5-d19a)

Not ready yet. This section will be written in the same step-by-step style when m5-d19a is finished.

---

## 7 · The doctor command: its report and whether it says pass or fail (m5-d3a)

Not ready yet. This section will be written in the same step-by-step style when m5-d3a is finished.

---

## 8 · Pasting into the typing area, and the cursor (m5-d6)

Not ready yet. This section will be written in the same step-by-step style when m5-d6 is finished.

---

## 9 · The build script and version check (m5-d10a)

Not ready yet. This section will be written in the same step-by-step style when m5-d10a is finished.

---

## 10 · Release settings and build check (m5-d10b)

Not ready yet. This section will be written in the same step-by-step style when m5-d10b is finished.

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

Not ready yet. This section will be written in the same step-by-step style when m5-d13b is finished.

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
