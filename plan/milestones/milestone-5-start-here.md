# Milestone 5 — Start Here

> **Status: front door.** This file tells you where to start, what to do for each deliverable, and how to use the testing document without reading all of it. It does not replace the detailed documents. Those are unchanged and remain authoritative. Where this guide and `milestone-5.md` disagree, `milestone-5.md` wins.

Unfamiliar word? See *Words you will see*, below Step 2.

## Milestone 5 in one minute

**The goal, in the spec's words:** "Daily-driver quality; installable by others." The milestone's *Goal and acceptance* section quotes it.

**Done means all of this is true:**

- The smoke test passes. That means every Milestone 4 check, plus the Milestone 5 additions.
- A fresh machine with no API key gets clear onboarding.
- `kirsch doctor` prints correct output.
- Release binaries exist for each target the spec names.
- `doc/` holds the user-facing documents the spec names, and every config key is documented in them.

**Which document is for what:**

| Document | What it is for | When you use it |
|----------|----------------|-----------------|
| `plan/milestones/milestone-5.md` | What to build. One section per deliverable, with its acceptance checks. | Read it only for the section of the deliverable you are running. Bishop reads it too. |
| `plan/milestones/milestone-5-start-here.md` (this file) | Where to start, and the loop you repeat. | First, and whenever you lose the thread. |
| `plan/PROGRESS.md` | Live status, the decisions you have recorded, and the carried items. | After each mission closes. Its *Milestone 5 — deliverables* table is the real status. |
| `plan/testing/manual/milestone-5.md` | The master test script. Agents expand it. | Only to read the wording. You do not run from it. |
| `plan/testing/manual/user-testing/milestone-5-donovan.md` | Your copy. You run from it and tick its boxes. It also has the *Run record* and *Reporting* sections. | Every time you test. |

In `plan/PROGRESS.md`, a deliverable is marked *done* when its mission closes. Your test ticks are separate; they live in your copy of the testing document.

## Before you start (once)

1. **Run `git status`. Commit and push the planning documents it lists under `plan/` and `README.md`.**
2. **Make your wave-1 decisions.** The next section lists them.
3. **That is all you need before the first mission.** You do not have to read `milestone-5.md` from end to end. Bishop checks each deliverable against the code before it writes that deliverable's brief. The check is described in the *Note for whoever executes this* section of `milestone-5.md`.

## Step 1 — your first decisions

These are the decisions that block wave 1. They are the decisions in the table below. Each is named in the *Needs first* cells of the wave-1 rows of the route map. O13 comes from the cell for m5-d19a, which reads "O13 (scheduling)".

The *Unblocks* column names the deliverables whose *Needs first* cell names that decision.

| Decision | The question in plain words | Options (condensed from `milestone-5.md`) | Recommendation | Unblocks |
|----------|-----------------------------|-------------------------------------------|----------------|----------|
| O1 | What the narrow status bar shows, and at which widths. | (a) Keep the spec rule: no tokens below 80 columns, and correct the Milestone 4 acceptance text. (b) Keep an `N tok` rung at 60–79 columns and below. (c) Show `N tok` only below a width you choose. | none recorded — your call | m5-d1 |
| O4 | The version string, and what `--version` shows. Ground Rule 7 already fixes the bare form, `0.1.0` with no `v`. The open part is the choices in the next column. | (a) `--version` prints the version only, or also the commit and date. (b) `go install` builds get a real version from build information, or not. | none recorded — your call | m5-d2 (also m5-d10a, m5-d10b, m5-d13c) |
| O6 | Who writes the `doc/` pages. | The milestone gives no separate options. The question is whether `@lambert` writes them and `@apone` checks the claims against the code. | `@lambert` writes, `@apone` reviews the claims against the code. This is the milestone's own recommendation. | m5-d13a (also m5-d13c, m5-d14a, m5-d14b) |
| O11 | The name of the new session event that records a cleared approval, and the matching change to the spec's §7. | Each part is a question, not an alternative. (a) Name the event. The milestone's example is `approval.scopes_cleared`. (b) Whether `/new` records one too. The milestone's own note says no, because `/new` starts a new file. | none recorded — your call | m5-d5 |
| O13 | Whether to schedule the optional groups (m5-d17, m5-d18) and the safety-test group (m5-d19) into v0.1, or leave them in the backlog. | (a) Schedule them into v0.1. (b) Leave them in the backlog. You may choose differently for each group: m5-d17, m5-d18 and m5-d19. | Schedule m5-d19. The milestone's reason: C30 names tests that cannot fail. You may defer it, but the gap stays open. m5-d17 and m5-d18 have no recommendation; your call. | m5-d19a (also m5-d17a, m5-d17b, m5-d18, m5-d19b) |
| O18 | Empty `NO_COLOR`. The code treats an empty value as set. The no-color.org convention treats it as unset. Which behaviour is pinned? | (a) Keep the code: empty means set, colour off. (b) Treat empty as unset: colour on, if the other conditions allow. | none recorded — your call | m5-d9 |
| O23 | What happens to golden screens that move when a screen changes. | (a) Reconcile the moved golden fences by the golden procedure, then a person reviews the diff. (b) List each moved golden first and stop for your approval. Either way, fences 80 columns and wider do not move unless their step says so. | none recorded — your call | m5-d1 (also m5-d7a, m5-d7b, m5-d21b) |

**How to record them.** You record each decision yourself, in `plan/PROGRESS.md`. Paste this block at the very end of the file, once, and fill in your choices:

```
## Milestone 5 — decisions recorded

| ID | Choice | Date |
|----|--------|------|
| O4 | <your choice> | <YYYY-MM-DD> |
```

Then add one row per decision to that table. Add each new row directly under the last row of that table, never in the middle. Other documents cite `plan/PROGRESS.md` by line number. For example, the O1 row of `milestone-5.md` cites `plan/PROGRESS.md:67`. Inserting a row in the middle would move those lines and break the citations.

You do not have to decide every O-number now. Later decisions go in the same table, before the deliverable that needs them. One exception: the C29 part of O15 must be recorded before m5-d19b runs.

## Step 2 — your first build mission

**Suggestion, and you may choose another wave-1 deliverable.** Start with **m5-d2** (`--version` and `--help`). Its decision is O4. It has a code step and its review. The milestone says the current `--help` exit status has not been verified, so this is where it gets checked.

Finishing m5-d2 clears it from the *Needs first* cell of every deliverable that lists it. The route map shows which. Those deliverables may still wait on other decisions.

Alternatives that need only a recommendation accepted: **m5-d19a** (decision O13, accepting the schedule recommendation) and **m5-d13a** (decision O6, accepting the recommendation to have `@lambert` write the page).

Paste this prompt, with the O4 choice filled in:

```
/mission Deliver m5-d2 from plan/milestones/milestone-5.md (section "## m5-d2 — `--version` and `--help`"). Recorded decisions it needs: O4 = <your choice>. Apply any "fix before lifting" item in plan/PROGRESS.md that names m5-d2. Expand section 2 of plan/testing/manual/milestone-5.md (its TO EXPAND blocks), then copy that section into plan/testing/manual/user-testing/milestone-5-donovan.md. Before editing my copy, save a verbatim copy to `.claude/memory/workspace/`. Keep any ticks and notes already in my copy. New or changed boxes start unticked. Set m5-d2 to done in the Milestone 5 deliverables table in plan/PROGRESS.md. No live provider runs.
```

If no carried item in `plan/PROGRESS.md` names m5-d2, Bishop says so in the brief.

## Words you will see

| Word | What it means |
|------|---------------|
| **Bishop** | The commander agent. It plans each mission, hands the work to the crew and writes no code itself. |
| **Mission** | One `/mission` run. Bishop gives each run an ID such as `mission-20261024-01`. |
| **Deliverable, `m5-dN`** | One piece of Milestone 5 work, sized for one small mission. Where a letter follows the number (`m5-d3a`, `m5-d3b`), the piece is larger and is cut into parts that run in order. |
| **O-number** | A decision only you make. They are listed in *Decisions the operator makes before starting* in `milestone-5.md`. |
| **Wave** | A batch of deliverables whose files do not overlap. A deliverable starts when its *Needs first* items are done and its decisions are recorded. The wave number is the suggested order. |
| **Gate** (`m5-d20`) | The point where you record the decisions listed in m5-d20's section in `milestone-5.md`, before the m5-d21 deliverables run. Exception: the C29 part of O15 is recorded before m5-d19b runs. |
| **(operator)** | Yours. Bishop does not delegate it. |
| **(optional)** | Scheduled only if you choose to (decision O13). |
| **(conditional)** | Runs only if a recorded decision calls for it. |
| **Fix before lifting** | A known correction listed in *Carried from Milestone 5 planning* in `plan/PROGRESS.md`. Bishop applies it to the brief when that deliverable starts. You do not fix it yourself. |
| **TO EXPAND** | A placeholder in the testing document where a detail is not yet known. The deliverable's own mission fills it in. |
| **Golden** | A reference screen grid, kept in `plan/spec/kirsch-ui-screens.md`. It changes only by the procedure in *Golden mechanism and regeneration procedure* in `milestone-5.md`. |
| **C, S, D and X items** | Carried items, spec tasks and other tracked IDs. The milestone maps each one to a deliverable. You can ignore them. The *Item → deliverable coverage* section of `milestone-5.md` has a check that proves each one is mapped. |
| **Automated / Hands-on / Operator** | The kinds of check in the testing document. *Automated* is a command with a clear pass result. *Hands-on* is steps you do in a terminal. *Operator* is a check only you can make. |
| **Live check** | A check that spends tokens or publishes something. You run it only when you choose to. |
| **Crew agents** | `@hicks` writes code. `@apone` reviews it. `@lambert` writes documentation and state. `@vasquez` is the senior developer, and joins only after two junior fix rounds. |

## The loop for every deliverable

1. **Pick a deliverable** from the [route map](#route-map). Its *Needs first* items must be done, and its decisions recorded.
2. **Paste the mission prompt** from the template below.
3. **Bishop runs it.** The order is: plan, then `@hicks` or `@lambert` writes, then `@apone` reviews, then the mission closes. Answer any question Bishop asks.
4. **When it closes, run that deliverable's testing section.** See [Testing a deliverable — the short path](#testing-a-deliverable--the-short-path). Tick the boxes in your copy.
5. **Commit and push.**

The prompt template:

```
/mission Deliver <m5-dX> from plan/milestones/milestone-5.md (section "## <m5-dX> — <title>"). Recorded decisions it needs: <O-numbers and choices, or "see plan/PROGRESS.md, Milestone 5 — decisions recorded">. Apply any "fix before lifting" item in plan/PROGRESS.md that names <m5-dX>. Expand section <N> of plan/testing/manual/milestone-5.md (its TO EXPAND blocks), then copy that section into plan/testing/manual/user-testing/milestone-5-donovan.md. Before editing my copy, save a verbatim copy to `.claude/memory/workspace/`. Keep any ticks and notes already in my copy. New or changed boxes start unticked. Set <m5-dX> to done in the Milestone 5 deliverables table in plan/PROGRESS.md. No live provider runs.
```

What each placeholder means:

- **`<m5-dX>`** is the deliverable's ID from the route map, such as `m5-d3a`.
- **The section heading in quotes** must match the heading in `milestone-5.md` exactly, including any backticks.
- **The recorded decisions** are the O-numbers the deliverable needs, with your choices. If you recorded them in the *Milestone 5 — decisions recorded* section, write the pointer shown in the template.
- **`<N>`** is the deliverable's section number in the *Testing §* column of the route map. It matches the *Order of work* table in the testing document.

## Testing a deliverable — the short path

This is the part to keep to hand. You do not need the rest of the testing document to run a section.

1. Open a terminal, then `cd /Volumes/DATA/Github/kirsch`. Type `bash`. The command blocks are written for bash.
2. In your copy (`milestone-5-donovan.md`), fill in the *Run record* before your first check. Then paste the **Setup** block, then paste the **Guard functions** block. It runs `guard` and prints the controls. Run `guard` again whenever you change the config. Expect `GUARD PASS` and `CONTROL PASS` lines. In a scratch run on 2026-10-24, Setup printed one `Initialized empty Git repository in …` line from `git init`. The Guard functions block then printed `CONTROL PASS`, `GUARD PASS` and two more `CONTROL PASS` lines. Reported by @lambert; Bishop did not re-run it. The two `grep` lines under *Guard* are covered by `guard`; skip them. **If you see `GUARD FAIL`, stop.** A kirsch launch could then write to your real session store or reach the provider.
3. Find the deliverable's section number in the route map. Jump to the heading `## N ·`.
4. Read the **Context** line. It says which terminal, directory, workspace, key state and config to be in. Never run kirsch with the checkout as the workspace; use `--workspace "$WORKSPACE"` as the Context says.
5. If the section has a block headed *Key-state…*, run it first. Expect `CONTROL PASS`, then `0`. The `0` can come with exit status 1, because `grep -c` exits 1 when it counts nothing. That is expected. Section 2's block also prints git lines, and the text under its heading says what to expect.
6. Work down **Automated checks**, then **Hands-on checks**. Mark each one `- [✅]` when it passes or `- [❌]` when it fails. Write anything odd under **Notes / evidence**.
7. A step is live if it says it contacts the provider or needs a key. Live steps can sit inside Hands-on lists, so read each step before you run it. Enter a key only with the `read -rs` line under *Live key entry* in Setup, never in a file or chat, and run the *Model check before any live step* block first. A release run or a tag publishes publicly and **cannot be undone**; those steps say ONE-WAY.
8. At the end of the session: if you ran section 7, first remove its extra folder with `rm -rf -- "$T_BROKEN"`, because Cleanup unsets that variable. Then paste the **Cleanup** block. Expect `CONTROL PASS`, then `0`. Then remove the other scratch folders yourself with the line from `plan/testing/manual/milestone-4.md:60` (a recursive `rm` of `$T_CFG`, `$T_STATE` and `$WORKSPACE`). Only after that, repeat step 2 if you want to keep testing in that terminal: step 2 creates new scratch folders and overwrites the variables that name the old ones.

A new terminal window must repeat step 2 before it runs anything.

**What you can skip:**

- **Mechanical checks.** They check the document, not your machine. The block is the paragraph headed **Mechanical checks.** inside *How to read this document* in your copy. Run it under bash before a test run, and after any mission expands a section. Skip it otherwise.
- **The rest of How to read this document.** It is reference material.
- **Any section whose deliverable has not landed yet.**

**If a section still says TO EXPAND after its mission has closed,** ask Bishop to expand that section. That is a mission, so you do not write it yourself.

## Route map

Every deliverable in *Deliverables* in `milestone-5.md`, in wave order. The *Testing §* column is the section number in the testing document's *Order of work*.

*Who* names the agents for the work. Code goes to `@hicks` and is reviewed by `@apone`. Documentation goes to `@lambert`, reviewed by `@apone`. A code deliverable with a docs step names `@lambert` for that step too. **You** marks the parts only you do or check.

| Wave | Deliverable | In plain words | Needs first | Testing § | Who |
|------|-------------|----------------|-------------|-----------|-----|
| 1 | m5-d1 Narrow-width status-bar fallback | Narrow status bar shows what you chose. | O1, O23 | 1 | `@hicks`, `@lambert` (docs step), `@apone` |
| 1 | m5-d2 `--version` and `--help` | `--version` and `--help` work as specified. | O4 | 2 | `@hicks`, `@apone` |
| 1 | m5-d5 Persist `/approvals` clear | A cleared approval stays cleared after resume. | O11 | 3 | `@hicks`, `@lambert` (docs step), `@apone`. You: operator check |
| 1 | m5-d9 Terminal capability and resize tests | Colour detection and resize behaviour get tests. | O18 | 4 | `@hicks`, `@apone`. You: operator check |
| 1 | m5-d13a `doc/configuration.md` | Every config key documented, with defaults. | O6 | 5 | `@lambert`, `@apone` |
| 1 | m5-d19a Safety-test repair for `run_command`: C30, C35, C36 | Safety tests that fail when the code breaks. | O13 (scheduling) | 6 | `@hicks`, `@apone` |
| 2 | m5-d3a Doctor: dispatch, report grammar, version line, exit codes, arch entry | `kirsch doctor` runs and reports version and exit code. | m5-d2, O5 | 7 | `@hicks`, `@lambert` (docs step), `@apone` |
| 2 | m5-d6 Composer paste path and caret rows | Pasted text lands verbatim; composer follows the caret. | O10; m5-d1 (file ownership: `view.go`, if O10 is wrap) | 8 | `@hicks`, `@apone`. You: operator check |
| 2 | m5-d10a Build wrapper and version proof | A build script prints the version with no `v`. | m5-d2, O4 | 9 | `@hicks`, `@apone` |
| 2 | m5-d10b goreleaser config and target-compile check | One config compiles every target. CI checks it. | m5-d2, O2, O4, O20 | 10 | `@hicks`, `@apone`. You: operator check |
| 2 | m5-d19b Safety-test repair: C29 | The 500 ms box test holds, or is relaxed. | m5-d19a, O15 (C29), O13 | 11 | `@hicks`, `@lambert` (relax branch only), `@apone` |
| 3 | m5-d3b Doctor: `rg`, `git`, repository and config probes | Doctor flags missing `rg` or `git`; checks repo and config. | m5-d3a, O5 | 12 | `@hicks`, `@apone` |
| 3 | m5-d7a Soft-wrap fidelity: `wrap()` | Wrapped text keeps its indentation and spacing. | m5-d1, m5-d6, O23 | 13 | `@hicks`, `@apone`. You: operator check |
| 3 | m5-d8 Paste debounce | Paste coalescing added, or the constant removed and spec amended. | m5-d6, O9 | 14 | `@hicks`, `@lambert` (amend branch), `@apone`. You: operator check |
| 3 | m5-d11 GitHub release workflow | A release workflow runs on `v*` tags. | m5-d10b, O3 | 15 | `@hicks`, `@apone`. You: first live run |
| 3 | m5-d13b Key-coverage test (S36) | The page is checked: a missing config key fails it. | m5-d13a | 16 | `@hicks`, `@apone` |
| 3 | m5-d13c `doc/install.md` | The install page says what a `go install` build reports. | m5-d2, m5-d10a, m5-d10b, O4, O6 | 17 | `@lambert`, `@apone`. You: operator check |
| 3 | m5-d16 Fresh-machine no-key check | A run with no key shows onboarding. | O14, O19; m5-d10a (builds the binary the script runs); m5-d2 and m5-d3a (file ownership: `cmd/kirsch/main.go`, only if O14 makes "not in a repo" a screen) | 18 | `@hicks`, `@apone`. You: operator check |
| 3 | m5-d17a (optional) `/new` lifecycle: C51, C52 | No orphan session file on quit; `/new` rotates session id. | m5-d5, O13 | 19 | `@hicks`, `@apone` |
| 4 | m5-d4a Doctor: key, backend, model and project-context lines | Doctor names which key wins, and never prints its value. | m5-d3b, O5 | 20 | `@hicks`, `@apone` |
| 4 | m5-d7b Soft-wrap fidelity: composer cut marker | Cut composer rows show the marker you chose. | m5-d6, m5-d7a, O17, O23 | 21 | `@hicks`, `@apone`. You: operator check |
| 4 | m5-d14a `doc/usage.md` corrections | The usage page matches the code. | m5-d1, m5-d5, m5-d6, m5-d7a, O6, O10 | 22 | `@lambert`, `@apone` |
| 4 | m5-d15 Front-door and tracker freshness | README, changelog and trackers agree with the code. | m5-d1, m5-d10a, m5-d10b, m5-d13c, O22 | 23 | `@lambert`, `@apone` |
| 5 | m5-d4b Doctor: state directory and lock state | Doctor checks the lock and state folder; live runs untouched. | m5-d4a, O5 | 24 | `@hicks`, `@apone`. You: operator check |
| 5 | m5-d12 Repository files and golden-regeneration note | `CONTRIBUTING`, `SECURITY` and `NOTICE` exist and match the code. | m5-d10a, m5-d10b, O7, O8, O21; m5-d8 (file ownership: `ui-spec-v0.1.md`, amend branch) | 25 | `@lambert`, `@apone`. You: operator check |
| 5 | m5-d17b (optional) `/new` busy state and turn fencing: C48, C49 | Enter during `/new` shows no stray card; late messages drop. | m5-d8, m5-d17a, O13 | 26 | `@hicks`, `@apone`. You: live check, if you choose |
| 6 | m5-d14b `doc/troubleshooting.md` and `doc/README.md` | Troubleshooting explains each doctor line; the index matches the files. | m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a, m5-d13c, O6 | 27 | `@lambert`, `@apone`. You: live and operator checks |
| 6 | m5-d18 (optional) Resume totals and compaction % | Totals show after resume; compaction % updates after `/compact`. | m5-d17a (file ownership only), O13 | 28 | `@hicks`, `@apone`. You: live check, if you choose |
| gate | m5-d20 (operator) Record the operator's decisions | You record the decisions before the m5-d21 deliverables run. | None; records O12, O15, O16 and the decision list in its section | 29 | You |
| 7 | m5-d21a (conditional) Model table (X5, C9) | The model table matches your recorded choice. | m5-d20 | 30 | `@hicks`, `@apone` |
| 7 | m5-d21b (conditional) Help footer and goldens (X8, O16) | The help footer matches O16; moved goldens are listed. | m5-d20, O16, O23 | 31 | `@hicks`, `@apone`. You: operator check |
| 7 | m5-d21c (conditional) Debug log clip (C24) | The 400 debug log writes at most your chosen bound. | m5-d20 | 32 | `@hicks`, `@lambert` (`SECURITY.md` sentence), `@apone` |
| 7 | m5-d21d (conditional) Debug log disclosure (C24) | `SECURITY.md` states the 400 log behaviour. | m5-d20, m5-d12 (file ownership: `SECURITY.md`) | 33 | `@lambert`, `@apone`. You: operator check |
| 7 | m5-d21e (conditional) ULID dependency (C40) | Swap the home-made ULID for the library, or keep it. | m5-d20 | 34 | `@hicks`, `@apone` |
| 7 | m5-d21f (conditional) §7 amendment (C43, C46) | Spec §7 lists the session events the code writes. | m5-d20, m5-d5 (file ownership: `plan/spec/kirsch-plan.md` §7) | 35 | `@lambert`, `@apone`. You: operator check |
| 7 | m5-d21g (conditional) Lock behaviour (C8) | Lock behaviour changes, or you accept it as it is. | m5-d20 | 36 | `@hicks`, `@apone` |
| 8 | m5-d22 (operator) Demo, release smoke run, tag | Demo, release smoke test, then the `v0.1.0` tag. | m5-d21 deliverables that ran, O12, O15, O16 | 37 | You |

## When you are stuck

- **A deliverable is blocked by a decision.** Record the decision in the *Milestone 5 — decisions recorded* table in `plan/PROGRESS.md`, as Step 1 shows. Then run the deliverable.
- **Bishop escalates to `@vasquez` after two junior fix rounds.** That is the rule, not a failure. For documentation-only deliverables, Bishop tells you how it is handling a failed fix. The senior developer gets two fix rounds. If the senior developer also fails twice, Bishop stops and brings the mission to you. Your call is needed at that point.
- **A testing command fails.** Mark the check `- [❌]`, write what happened under **Notes / evidence**, and tell Bishop in the next mission. For the report format, see the **Reporting** section of your copy.
- **You are not sure what is next.** Open the *Milestone 5 — deliverables* table in `plan/PROGRESS.md`, and the route map in this file.
