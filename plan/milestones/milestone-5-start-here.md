# Milestone 5 — start here

This page tells you where Milestone 5 is, what to do next, and how each piece of work goes.

## Where you are now

*Updated at the end of every mission.*

- **Finished:** m5-d1 — the status bar shows no token count below 80 columns, as the spec says, so your Milestone 4 ❌ is resolved. You tested it (section 1) on 2026-10-27 and everything matched. m5-d2 — kirsch shows its version and help screen. You tested it (section 2) on 2026-10-27 and everything matched. m5-d9 — an empty `NO_COLOR` setting no longer turns colour off, as the no-colour convention says, and new tests check colour and window resizing. You tested m5-d9 (section 4) on 2026-10-27. Your note about the typing line is recorded for later. m5-d19a — the safety tests for running commands are repaired and pass on their own. You tested them (section 6) on 2026-10-28 and everything matched. m5-d13a — the settings guide, `doc/configuration.md`, lists every setting with its normal value and what it does, and a settings file with a token-like name stops kirsch from starting. You tested it (section 5) on 2026-10-28 and everything matched. m5-d13b — a check now fails if a setting kirsch reads is missing from the settings guide. You tested it (section 16) on 2026-10-28 and everything matched. m5-d10a — `bash scripts/build.sh 9.9.9` builds a kirsch that reports `kirsch 9.9.9`, and the build script drops a leading `v`. A build with no version number reports `0.1.0-dev`, unless your kirsch folder is exactly on a release tag with no changes. You tested it (section 9) on 2026-10-28 and everything matched. m5-d5 — clearing approvals with `/approvals` now sticks: an approval you cleared stays cleared after you quit and run `kirsch resume`. You tested it (section 3) on 2026-10-29. m5-d3a — `kirsch doctor` printed the version and a summary line, and it exited 0. At that point it still reported outside a project folder and with a broken settings file. Its checks came later, in m5-d3b (section 12, tested on 2026-11-03). You tested it (section 7) on 2026-10-29. m5-d10b — the release settings file, `.goreleaser.yaml`, and a check that builds kirsch for every release target (`npm run ci:targets`). You tested it (section 10) on 2026-10-29. m5-d6 — text you paste into the typing area lands as you copied it, except that tabs turn into spaces, and the typing area follows the cursor when the text is longer than the area. You tested it (section 8) on 2026-10-29, including the real-Terminal paste check. In that check the last pasted lines showed with the cursor after them, and the Up-arrow step moved the window a line at a time. On 2026-11-03 you tested m5-d3b, m5-d7a, m5-d8, m5-d13c, m5-d4a, m5-d7b, m5-d14a and m5-d14b (sections 12, 13, 14, 17, 20, 21, 22 and 27), and each matched. You also tested m5-d4b (section 24), and every step is ticked. On 2026-11-09 you tested m5-d19b, m5-d11, m5-d16, m5-d17a, m5-d15, m5-d12, m5-d17b, m5-d18, m5-d21a, m5-d21b, m5-d21d and m5-d21f (sections 11, 15, 18, 19, 23, 25, 26, 28, 30, 31, 33 and 35), and each matched. Private vulnerability reporting is on, and the GitHub dry-run release was green.
- **Ready for you to test:** nothing new is waiting. These remain open: section 27 has one optional box, the re-check of the *No API key* paragraph at 80×24, marked not yet run; the "quit and reopen kirsch" check in section 3 still waits for a model turn, and it is checked at the final release check (section 37 of the checklist, not written yet); and a blank approval card for an unusual case is a known product defect that is not fixed yet, and section 6 explains it.
- **Next:** every crew deliverable is done and tested. m5-d22 (demo, release smoke run, tag) is yours, and Bishop guides it. You confirmed the O23 reading (section 31) on 2026-11-11. Other small choices are yours and are listed in `plan/PROGRESS.md` under the *Open from …* sections. For the later doctor checks the rule is recorded (O5, later checks, 2026-10-29): a check is a fail if the problem would stop kirsch starting, and a warning if kirsch still starts but works less well. Bishop applies that rule and lists the result for you.
- **Can start now:** no crew piece is left to start. m5-d22 is yours, and Bishop guides it (see *Next*). The status table and the decisions table are in `plan/PROGRESS.md` (*Milestone 5 — deliverables* and *Milestone 5 — decisions recorded*). **m5-d20** was your own piece: your decisions were recorded on 2026-11-03 (mission-20261103-05). The name of the new event for m5-d5 is recorded as `approval.scopes_cleared` (O11 (a)).

The full status table is in `plan/PROGRESS.md` under *Milestone 5 — deliverables*.

## What to do next

1. Test the section named under *Ready for you to test* above, if there is one. Open `plan/testing/manual/user-testing/milestone-5-donovan.md`, do *Before you start* (Setup A and Setup B), then the section. Tick each box as it matches.
2. If something does not match, tell Bishop the section and step number and paste what you saw.
3. Commit and push your changes when you are ready, after you have tested them (`git add -A`, `git commit -m "…"`, `git push`). Bishop lists what changed at the end of every mission.
4. The next piece is m5-d22 (demo, release smoke run, tag), and it is yours. When you are ready, tell Bishop, and it guides you through it. Bishop asks you for any decision it needs.

## How each piece of work goes

1. You paste the prompt into the chat with Bishop.
2. Bishop plans the work. The crew builds it, checks it, and writes your test section.
3. Bishop tells you in plain words: "Section N is ready to test."
4. You test it, tick the boxes, and commit.

Some pieces need a decision only you can make. Bishop asks you before it starts and records your answer in `plan/PROGRESS.md`.

## The prompt to paste

```
/mission Deliver <m5-dX> from plan/milestones/milestone-5.md (its section for <m5-dX>). Use the decisions recorded in plan/PROGRESS.md and ask me for any that are missing. Apply any "fix before lifting" item in plan/PROGRESS.md that names <m5-dX>. Expand section <N> of plan/testing/manual/milestone-5.md. Then write section <N> of my test checklist, plan/testing/manual/user-testing/milestone-5-donovan.md, replacing its "Not ready yet" text. Copy the layout of section 2 exactly: the same headings, labels, step blocks and closing lines, in the same order, with this piece's own steps. Section 2's own commands, its 9.9.9 test copy, its release-tag note and its tidy-up are specific to m5-d2 and are not copied. The start-folder line is part of the layout, and so is section 2's build-step allowance (lines starting `go: downloading` or `npm notice` are fine). **Anything odd?** starts as `_(nothing yet)_`. Run every step yourself first. Save a verbatim copy of my checklist to .claude/memory/workspace/ before editing it, keep my ticks and notes, and start new or changed boxes unticked. When it is done, set <m5-dX> to done in plan/PROGRESS.md and update every "where we are" line: the top of plan/PROGRESS.md, README.md, plan/START-HERE.md, plan/README.md, plan/milestones/README.md, and "Where you are now" and "What to do next" in plan/milestones/milestone-5-start-here.md. No live provider runs. Tell me in plain words when my section is ready to test.
```

- `<m5-dX>` is the piece's ID from the table below, such as `m5-d3a`.
- `<N>` is the piece's test section number from the table below.

**m5-d20 and m5-d22 are yours, not the crew's.** Do not paste the prompt for them. m5-d20 is done: your decisions are in the *Milestone 5 — decisions recorded* table in plan/PROGRESS.md. m5-d22 is the final demo and release, which Bishop will guide you through when its turn comes.

## Which piece can I do next?

A piece can start once everything in *Needs first* is done or decided. O-numbers are decisions only you make; Bishop will ask.

| Piece | In plain words | Needs first | Test section |
|-------|----------------|-------------|--------------|
| m5-d1 Narrow status bar | The narrow status bar shows the option you chose. | O1, O23 | 1 |
| m5-d2 `--version` and `--help` | `--version` and `--help` work as specified. | O4 | 2 |
| m5-d5 Keep cleared approvals | A cleared approval stays cleared after you resume. | O11 | 3 |
| m5-d9 Terminal colour and resize tests | Tests for colour detection and window resizing. | O18 | 4 |
| m5-d13a Settings guide (`doc/configuration.md`) | Every setting is written down, with its default. | O6 | 5 |
| m5-d19a Safety tests for running commands (repair) | Safety tests fail when the code breaks. | O13 (scheduling) | 6 |
| m5-d3a Doctor, first part | `kirsch doctor` runs and reports the version and a pass or fail result. | m5-d2, O5 | 7 |
| m5-d6 Paste and cursor | Pasted text lands as copied, except that tabs become spaces, and the typing area follows the cursor. The real-Terminal check was run on 2026-10-29 and you tested it (section 8). | O10; m5-d1 (after m5-d1 finishes, because they change the same file, `view.go`; only if O10 is wrap) | 8 |
| m5-d10a Build script and version check | A build script prints the version with no `v`. | m5-d2, O4 | 9 |
| m5-d10b Release settings and build check | One settings file builds every target, and the automatic checks on GitHub run it. | m5-d2, O2, O4, O20 | 10 |
| m5-d19b Safety test repair, second part | A timing test is kept as it is or made less strict. Done; you tested it (section 11) on 2026-11-09. | m5-d19a, O15 (C29), O13 | 11 |
| m5-d3b Doctor: tools and project checks | Doctor flags a missing `rg` or `git`, and checks the repository and settings. Done; you tested it (section 12) on 2026-11-03. | m5-d3a, O5 | 12 |
| m5-d7a Soft wrap: wrapped text | Wrapped text keeps its indentation and spacing, and streamed replies are cleaned as they arrive. Done; you tested it (section 13) on 2026-11-03. | m5-d1, m5-d6, O23 | 13 |
| m5-d8 Paste delay | Done; you tested it (section 14) on 2026-11-03. The delay setting is removed and the spec changed (O9), because bracketed paste already gives one update. Open: multi-line paste with bracketed paste off. | m5-d6, O9 | 14 |
| m5-d11 Release workflow on GitHub | The release steps run when you push a version tag, and a manual run is a dry run. Done; you tested it (section 15) on 2026-11-09. | m5-d10b, O3 | 15 |
| m5-d13b Settings guide check | The guide is checked: a missing setting fails the check. | m5-d13a | 16 |
| m5-d13c Install guide (`doc/install.md`) | Done; you tested it (section 17) on 2026-11-03. The install page says what a `go install` build reports. | m5-d2, m5-d10a, m5-d10b, O4, O6 | 17 |
| m5-d16 Fresh machine, no key | A first run with no API key shows clear next steps. Done; you tested it (section 18) on 2026-11-09. | O14, O19; m5-d10a (builds the binary the script runs); m5-d2 and m5-d3a (after they finish, because they change the same file, `cmd/kirsch/main.go`; only if O14 makes "not in a repo" a screen) | 18 |
| m5-d17a (optional) Starting a new session with `/new` | No leftover session file when you quit, and `/new` gives a new session ID. Done; you tested it (section 19) on 2026-11-09. | m5-d5, O13 | 19 |
| m5-d4a Doctor: key, backend, model and project lines | Done; you tested it (section 20) on 2026-11-03. Doctor says which key is used, by name only, and never shows its value. It also reports the search backend, the model and the project notes file. | m5-d3b, O5 | 20 |
| m5-d7b Soft wrap: cut marker | Done; you tested it (section 21) on 2026-11-03. A line too long for the typing area ends in `⋯`, the marker you chose (O17). The spec's caret-cell sentence waits for your ratification. | m5-d6, m5-d7a, O17, O23 | 21 |
| m5-d14a Usage guide (`doc/usage.md`) corrections | Done; you tested it (section 22) on 2026-11-03. The usage page matches the code. | m5-d1, m5-d5, m5-d6, m5-d7a, O6, O10 | 22 |
| m5-d15 Front page and progress tracker | Done; you tested it (section 23) on 2026-11-09. The README, changelog and trackers agree with the code. | m5-d1, m5-d10a, m5-d10b, m5-d13c, O22 | 23 |
| m5-d4b Doctor: state folder and lock | Done; you tested it (section 24) on 2026-11-03. Doctor says whether the session folder can be written and whether another kirsch is running in this workspace. It leaves a running kirsch alone. | m5-d4a, O5 | 24 |
| m5-d12 Project files and reference-screen note | `CONTRIBUTING`, `SECURITY` and `NOTICE` exist and match the code. Done; you tested it (section 25) on 2026-11-09. | m5-d10a, m5-d10b, O7, O8, O21; m5-d8 (after it finishes, because they change the same file, `ui-spec-v0.1.md`; only if O9 chose to change the spec) | 25 |
| m5-d17b (optional) `/new` while busy | Pressing Enter during `/new` shows no stray card, and late turn messages from the cancelled reply are dropped (an approval answer is not; see "S-4, unfenced sends" under *Open from m5-d17b* in `plan/PROGRESS.md`). Done; you tested it (section 26) on 2026-11-09. | m5-d8, m5-d17a, O13 | 26 |
| m5-d14b Troubleshooting guide and docs list | Done; you tested it (section 27) on 2026-11-03. Troubleshooting explains each doctor line, and the docs list matches the files. | m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a, m5-d13c, O6 | 27 |
| m5-d18 (optional) Resume totals and percent full | Done; you tested it (section 28) on 2026-11-09. After you reopen a session, the status bar shows that session's totals. After `/compact`, the percent full updates. | m5-d17a (after it finishes, because they change the same file), O13 | 28 |
| m5-d20 Record your decisions | Done: your decisions were recorded on 2026-11-03. | None; records O12, O15, O16 and the decision list in its section | 29 |
| m5-d21a (conditional) Model list | Done; you tested it (section 30) on 2026-11-09. The model table matches Anthropic's pricing page as read on 2026-11-05. One price changed: the price for reading saved context with Sonnet 5.5, from $0.20 to $0.10 per million tokens. | m5-d20 | 30 |
| m5-d21b (conditional) Help footer and saved screen pictures | Done; you tested it (section 31) on 2026-11-09. The help box's last line now points to `github.com/djm56/kirsch` instead of `doc/usage.md`, which an installed Kirsch does not have. The link shows only when the window is wide enough to fit the whole line: at 80 columns it does not show. | m5-d20, O16, O23 | 31 |
| m5-d21c (conditional) Debug log size limit | The debug log writes at most the size you chose. Not needed: you chose to disclose (C24). | m5-d20 | 32 |
| m5-d21d (conditional) Debug log notice | Done; you tested it (section 33) on 2026-11-09. `SECURITY.md` now says that a 400 error writes the whole request to the debug log, and that this is a known limit: the request is not cut short. A 400 that names `MissingSessionID` is not written to the log. When the log is on, it also holds the text of each message you send, the workspace path at startup, and the command arguments of a refused session approval. It does not record each tool call. | m5-d20, m5-d12 (done) | 33 |
| m5-d21e (conditional) Unique-ID library | The home-made ID code is swapped for a library, or kept. Not needed: you chose to keep the home-made ID code (C40). | m5-d20 | 34 |
| m5-d21f (conditional) Spec section 7 change | Done; you tested it (section 35) on 2026-11-09. Section 7 of the spec now lists the session events the code writes, and it says which events the code does not write. | m5-d20, m5-d5 (after it finishes, because they change the same file, `plan/spec/kirsch-plan.md` §7) | 35 |
| m5-d21g (conditional) Two copies at once | How two copies running at the same time behave: changed, or kept as it is. Not needed: you accepted it as it is (C8). | m5-d20 | 36 |
| m5-d22 Demo and final release check | A demo, a final release check, then the `v0.1.0` tag. | m5-d21 pieces that ran, O12, O15, O16 | 37 |

This table is based on plan/milestones/milestone-5.md. If they ever disagree, milestone-5.md is right.

## Words you will see

| Word | What it means |
|------|---------------|
| **Bishop** | The commander agent. It plans each mission, hands the work to the crew, and writes no code itself. |
| **Mission** | One `/mission` run. Bishop gives each run an ID such as `mission-20261024-01`. |
| **Piece (`m5-dN`)** | One piece of Milestone 5 work, sized for one mission. Pieces that share a number but have different letters, such as `m5-d3a` and `m5-d3b`, are parts of one larger job; each row in the table says what it needs first. |
| **O-number** | A decision only you make. Bishop asks you for it. |
| **Crew** | `@hicks` writes code. `@apone` checks it. `@lambert` writes documents and keeps records. `@vasquez` is the senior developer who steps in if a fix keeps failing. |
| **(optional) / (conditional)** | *(optional)* means the piece happens only if you schedule it. *(conditional)* means it happens only if a decision you recorded calls for it. |
| **Test checklist** | Your copy: `plan/testing/manual/user-testing/milestone-5-donovan.md`. |
| **`plan/PROGRESS.md`** | The full status record, including your recorded decisions. |

## When you are stuck

- **You are not sure what is next.** Read *Where you are now* at the top of this page, then ask Bishop.
- **A test step does not match.** Do not tick the box. Tell Bishop the section and the step number, and paste what you saw.
- **Bishop asks a decision you do not understand.** Ask Bishop to explain the options in plain words.
- **A fix keeps failing.** Bishop brings in the senior developer. Bishop asks you only if that also fails twice.

The detailed plan is `plan/milestones/milestone-5.md`. You do not need to read it; Bishop does.
