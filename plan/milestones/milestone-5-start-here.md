# Milestone 5 — start here

This page tells you where Milestone 5 is, what to do next, and how each piece of work goes.

## Where you are now

*Updated at the end of every mission.*

- **Finished:** m5-d1 — the status bar shows no token count below 80 columns, as the spec says, so your Milestone 4 ❌ is resolved. m5-d2 — kirsch shows its version and help screen. You tested it (section 2) on 2026-10-27 and everything matched.
- **Ready for you to test:** section 1 — the status bar on a narrow window. Open `plan/testing/manual/user-testing/milestone-5-donovan.md`, do *Before you start*, then section 1.
- **Next:** test section 1, then pick a piece from *Can start now*. You do not have to commit first.
- **Can start now:** m5-d9, m5-d10a, m5-d13a, m5-d19a — their decisions are recorded. **m5-d20** is your own piece: writing down your remaining decisions. You can do it at any time, and it unblocks several other pieces. m5-d5 needs one more answer from you (the name of the new event, part (a) of O11); Bishop will ask.

The full status table is in `plan/PROGRESS.md` under *Milestone 5 — deliverables*.

## What to do next

1. Test the section named under *Ready for you to test* above, if there is one. Open `plan/testing/manual/user-testing/milestone-5-donovan.md`, do *Before you start* (Setup A and Setup B), then the section. Tick each box as it matches.
2. If something does not match, tell Bishop the section and step number and paste what you saw.
3. Commit and push your changes when you are ready, after you have tested them (`git add -A`, `git commit -m "…"`, `git push`). Bishop lists what changed at the end of every mission.
4. Start a piece from *Can start now*: copy the prompt in *The prompt to paste*, fill in the piece, and paste it. Do not paste it for a piece the *Next* line says Bishop starts, or for m5-d20 and m5-d22 (see the note under the prompt). Bishop asks you for any decision it needs.

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

**m5-d20 and m5-d22 are yours, not the crew's.** Do not paste the prompt for them. For m5-d20, write your decisions in the *Milestone 5 — decisions recorded* table at the end of plan/PROGRESS.md (or ask Bishop to walk you through them). m5-d22 is the final demo and release, which Bishop will guide you through when its turn comes.

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
| m5-d6 Paste and cursor | Pasted text lands exactly as copied, and the typing area follows the cursor. | O10; m5-d1 (after m5-d1 finishes, because they change the same file, `view.go`; only if O10 is wrap) | 8 |
| m5-d10a Build script and version check | A build script prints the version with no `v`. | m5-d2, O4 | 9 |
| m5-d10b Release settings and build check | One settings file builds every target, and the automatic checks on GitHub run it. | m5-d2, O2, O4, O20 | 10 |
| m5-d19b Safety test repair, second part | A timing test is kept as it is or made less strict. | m5-d19a, O15 (C29), O13 | 11 |
| m5-d3b Doctor: tools and project checks | Doctor flags a missing `rg` or `git`, and checks the repository and settings. | m5-d3a, O5 | 12 |
| m5-d7a Soft wrap: wrapped text | Wrapped text keeps its indentation and spacing. | m5-d1, m5-d6, O23 | 13 |
| m5-d8 Paste delay | Paste joining is added, or the delay setting is removed and the spec changed. | m5-d6, O9 | 14 |
| m5-d11 Release workflow on GitHub | The release steps run when you push a version tag. | m5-d10b, O3 | 15 |
| m5-d13b Settings guide check | The guide is checked: a missing setting fails the check. | m5-d13a | 16 |
| m5-d13c Install guide (`doc/install.md`) | The install page says what a `go install` build reports. | m5-d2, m5-d10a, m5-d10b, O4, O6 | 17 |
| m5-d16 Fresh machine, no key | A first run with no API key shows clear next steps. | O14, O19; m5-d10a (builds the binary the script runs); m5-d2 and m5-d3a (after they finish, because they change the same file, `cmd/kirsch/main.go`; only if O14 makes "not in a repo" a screen) | 18 |
| m5-d17a (optional) Starting a new session with `/new` | No leftover session file when you quit, and `/new` gives a new session ID. | m5-d5, O13 | 19 |
| m5-d4a Doctor: key, backend, model and project lines | Doctor says which key is used, and never shows its value. | m5-d3b, O5 | 20 |
| m5-d7b Soft wrap: cut marker | Cut lines in the typing area show the marker you chose. | m5-d6, m5-d7a, O17, O23 | 21 |
| m5-d14a Usage guide (`doc/usage.md`) corrections | The usage page matches the code. | m5-d1, m5-d5, m5-d6, m5-d7a, O6, O10 | 22 |
| m5-d15 Front page and progress tracker | The README, changelog and trackers agree with the code. | m5-d1, m5-d10a, m5-d10b, m5-d13c, O22 | 23 |
| m5-d4b Doctor: state folder and lock | Doctor checks the lock and the state folder, and leaves live runs alone. | m5-d4a, O5 | 24 |
| m5-d12 Project files and reference-screen note | `CONTRIBUTING`, `SECURITY` and `NOTICE` exist and match the code. | m5-d10a, m5-d10b, O7, O8, O21; m5-d8 (after it finishes, because they change the same file, `ui-spec-v0.1.md`; only if O9 chose to change the spec) | 25 |
| m5-d17b (optional) `/new` while busy | Pressing Enter during `/new` shows no stray card, and late messages are dropped. | m5-d8, m5-d17a, O13 | 26 |
| m5-d14b Troubleshooting guide and docs list | Troubleshooting explains each doctor line, and the docs list matches the files. | m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a, m5-d13c, O6 | 27 |
| m5-d18 (optional) Resume totals and percent full | Totals show after resume, and the percent full updates after `/compact`. | m5-d17a (after it finishes, because they change the same file), O13 | 28 |
| m5-d20 Record your decisions | You write down your decisions before the m5-d21 pieces run. | None; records O12, O15, O16 and the decision list in its section | 29 |
| m5-d21a (conditional) Model list | The model list matches your recorded choice. | m5-d20 | 30 |
| m5-d21b (conditional) Help footer and saved screen pictures | The help footer matches O16, and moved screen pictures are listed. | m5-d20, O16, O23 | 31 |
| m5-d21c (conditional) Debug log size limit | The debug log writes at most the size you chose. | m5-d20 | 32 |
| m5-d21d (conditional) Debug log notice | `SECURITY.md` says how the debug log behaves. | m5-d20, m5-d12 (after it finishes, because they change the same file, `SECURITY.md`) | 33 |
| m5-d21e (conditional) Unique-ID library | The home-made ID code is swapped for a library, or kept. | m5-d20 | 34 |
| m5-d21f (conditional) Spec section 7 change | Spec section 7 lists the session events the code writes. | m5-d20, m5-d5 (after it finishes, because they change the same file, `plan/spec/kirsch-plan.md` §7) | 35 |
| m5-d21g (conditional) Two copies at once | How two copies running at the same time behave: changed, or kept as it is. | m5-d20 | 36 |
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
