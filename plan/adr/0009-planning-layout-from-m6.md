# ADR 0009 — Planning layout from Milestone 6

- **Status:** Accepted
- **Date:** 2026-10-29
- **Deciders:** Project owner (the operator), accepting Bishop's recommendation (2026-10-29)

## Context

This is recorded as an ADR because it sets a structural decision that would be expensive to reverse, and the operator directed it. It is not a product-plan change: the amendment log in `plan/spec/kirsch-plan.md` §11 records changes to the product plan, and this ADR does not change the product plan.

The `plan/` folder holds these kinds of content, and they change at different speeds:

- **Decisions** — `adr/` and `spec/`. These change rarely.
- **Backlog and status** — `milestones/`, `PROGRESS.md` and `START-HERE.md`. These change with every step.
- **Manual test checklists** — `testing/manual/milestone-N.md`, plus a plain operator copy per milestone in `testing/manual/user-testing/`. Each milestone adds a new master checklist and a new operator copy.
- **Process** — `process/`. This changes rarely.

Test checklists are the largest part of the folder. Each milestone adds a new master checklist and a new operator copy, so the total grows with every milestone.

The sizes below are pinned to commit `686d3c4`, the last commit before this ADR was written. They are read from git, not from the working tree, so anyone can reproduce them:

```sh
git grep -c '' 686d3c4 -- 'plan/*.md' | awk -F: '{s+=$NF} END {print s}'
git grep -c '' 686d3c4 -- 'plan/testing/*.md' | awk -F: '{s+=$NF} END {print s}'
git ls-tree -r --name-only 686d3c4 -- plan | grep -c '\.md$'
```

At commit `686d3c4`, `plan/` holds 44 Markdown files and 21,470 lines. The testing folder holds 11,870 of those lines, which is more than half.

Status text is written by hand in these places: `PROGRESS.md`, `START-HERE.md`, the milestone pages and the testing indexes. Each copy can drift from the others and from the ticks in the checklists. Missions mission-20261029-02 and -03 fixed stale status text, and their DEBRIEFs record it. Mission -02 also fixed a hand-kept ID list, and -03 fixed hand-kept doctor counts. Missions -05 and -06 fixed hand-kept lists, closed ranges and counts, as their DEBRIEFs record. That is a recurring cost, not a one-off. This decision removes only the hand-kept tick counts and the "where you are" narrative. It does not cover the other classes those missions fixed: doctor counts, ID lists, closed ranges and summary counts. It does not claim that a script would have caught those fixes.

The options below were weighed.

- **GitHub Issues, Milestones and a Project board.** This gives a real board view and a shared tracker.
- **Backlog.md.** Per its description at https://github.com/MrLesk/Backlog.md, it turns a folder into a board powered by plain Markdown files and a zero-config CLI. Each task is a Markdown file in a project-local folder. It also has an MCP server. Git is optional for it.
- **An in-repo restructure with no new tool.** The folders are reorganised, and a small script builds status from the tick marks.

The GitHub and Backlog.md options are not chosen now.

GitHub Issues, Milestones and a Project board are not chosen, for these reasons:

- **The crew's checks read files in git.** These checks include tick preservation under DIR-020, before-and-after diffs against a committed baseline, and the write guard on existing files. None of them can see GitHub data.
- **Each agent write to GitHub is published at once.** A published write is one-way. Bishop's standing order in `.claude/SOUL.md` says that anything destructive or one-way waits for an explicit yes, so each such write would need the operator's approval. That order is Bishop's practice. No hook enforces it. It does not fit an agent crew that writes state repeatedly during a mission.
- **`gh` needs an upgrade, not an install.** The installed version is 2.74.2, checked with `gh --version` on 2026-10-29. Sub-issue commands arrived in gh v2.94.0, per GitHub's changelog of 2026-06-10, [Manage sub-issues, types and dependencies from GitHub CLI](https://github.blog/changelog/2026-06-10-manage-sub-issues-types-and-dependencies-from-github-cli/). The upgrade is the operator's call.

Backlog.md is not chosen, for these reasons:

- **It adds a tool, a CLI and an MCP server, for little gain over plain files.** It would be a new install. The operator's decision for this project is no new installs.
- **Whether its task format can reuse the current checklists has not been checked.** If it cannot, each checklist and operator copy would need reshaping.

Linear and Jira were also considered and are not chosen. These appear to be sized for more than a single-operator project needs. For a single-operator project, they add cost without a matching benefit.

## Decision

From Milestone 6 onwards, `plan/` uses this layout.

1. **A living test checklist per feature area**, for example version and help, doctor, composer and paste. The checklist replaces the per-milestone master and operator copy. Each area keeps the split already in use: technical checks for the builder, and plain paste → see → tick steps for the operator. Checklists hold no ticks.
2. **A dated results file per test run.** Before a run, the crew creates the results file by copying the checklist sections for that run byte for byte, with every box unticked. Bishop selects the checklist sections for a run from the deliverables being tested, and the operator may add sections. Only the operator ticks. The results file keeps the operator's evidence separate from the editable checklist. DIR-020 applies to results files.
3. **Tick counts are built by `scripts/plan-status.sh` from tick marks, and never typed by hand.** From M6 onwards, the script replaces the hand-kept tick counts and the "where you are" narrative in `PROGRESS.md`, `START-HERE.md` and the testing indexes. Those files name `scripts/plan-status.sh` as the command to run instead. The script's output goes to stdout only, and nothing it generates is committed. The deliverables table in `PROGRESS.md` keeps its Owner and Status cells. Those cells stay hand-kept, and they are used for claiming work and for tying each branch to its row, as now. `PROGRESS.md` keeps decisions, carried items and that table. The M5 sections of `PROGRESS.md` keep their existing text as the record (item 6). The script does not exist yet. Its requirements are:
   - **What counts as a checkbox line.** A list item: optional indentation, then `- ` or `* `, then `[`, then a marker of 1 to 4 characters (characters, not bytes), then `]`. A longer bracket such as `- [Benefit 1]` in `plan/adr/TEMPLATE.md` is not a checkbox line and is not reported.
   - **What each marker means.** `[ ]` is open. `[x]` and `[✅]` are done. `[❌]` is failed. Any other 1 to 4 character marker is unrecognised. In `plan/`, as measured on 2026-10-29 with the first command in the block below, `[✅ ]` is the only such marker. It sits at `plan/testing/manual/user-testing/milestone-0-donovan.md:169`.
   - **Code fences.** Lines between fence lines are not counted. A fence line starts, after optional indentation, with ``` or ~~~. A fence closes on the same fence character.
   - **Grouping.** Counts are grouped under the nearest preceding `## ` heading. `### ` and deeper headings count under their enclosing `## `. Lines before the first `## ` are grouped under the file name itself.
   - **Aggregation.** Markers are compared as exact strings, whatever the locale. Any `sort` or `uniq` the script uses runs with `LC_ALL=C`, or the counting is done in the script's own code.
   - **Invocation.** One or more file paths. A directory or a missing file is a usage error with a non-zero exit.
   - **Output.** Plain text on stdout. One line per section with open, done, failed and unrecognised counts, plus a total line per file. Then `file:line` for each failed box and each unrecognised box. The exact layout is fixed by the script's tests.
   - **Exit status.** 0 when every file was read, whatever the counts. A usage error or an unreadable file gives a non-zero exit.
   - **Read-only.** The script writes nothing.
   - **Tests.** The fixture includes `[❌]`, `[✅]` and `[✅ ]` lines, and the test asserts a separate count for each. A sabotaged run that counts with a plain `sort | uniq -c` must make the test fail.

   The hazard was measured on 2026-10-29 in the session locale `en_ZA.UTF-8`, with the commands below. The `[❌]` lines at `plan/testing/manual/user-testing/milestone-4-donovan.md:1042` and `plan/testing/manual/user-testing/milestone-2-donovan.md:503` were counted in the `[✅]` row, which held 824 lines. The `[✅ ]` line at `milestone-0-donovan.md:169` was counted in the `[ ]` row, which held 967 lines. With `LC_ALL=C`, each marker is its own row: `[ ]` 966, `[x]` 177, `[✅ ]` 1, `[✅]` 822 and `[❌]` 2. The per-marker command finds the `[❌]` lines at the locations above.

   ```sh
   /usr/bin/grep -rhoE '^\s*[-*] \[[^]]{1,4}\]' plan | sed 's/^ *//' | sort | uniq -c
   /usr/bin/grep -rhoE '^\s*[-*] \[[^]]{1,4}\]' plan | sed 's/^ *//' | LC_ALL=C sort | LC_ALL=C uniq -c
   /usr/bin/grep -rhE "^\s*[-*] \[❌\]" plan | wc -l
   ```

4. **Milestone pages hold only the goal, the scope and a deliverables table.** The tasks for each deliverable sit in a small file per deliverable.
5. **`spec/` and existing ADRs do not change under this ADR.** This ADR makes no change to `process/`. The status rules in `process/working-agreement.md` stay true under item 3, because the Owner and Status cells they describe stay hand-kept in the deliverables table. When M6 is planned, those rules are checked against item 3 again, and changed only if one of them stops being true.
6. **M5 stays in its current layout until it closes.** The M5 parts of `PROGRESS.md`, `START-HERE.md` and the M5 files stay as they are until M5 closes. After M5 closes, its hand-kept text in `PROGRESS.md` stays as the record and is not rebuilt by the script. Item 3 applies from the M6 sections onward. Earlier milestones are never migrated, and they stay as the record. The M5 paths are already cited in mission files, in `FINDINGS.md` and in `PATTERNS.md`. Moving them mid-milestone would break those links. The grep behind this claim is `grep -rl "milestone-5" .claude/memory/`. It finds none in `DIRECTIVES.md`. The script may be run on any file, M5 included, because it only reads and counts.

The exact folder and file names for items 1, 2 and 4 are settled when Milestone 6 is planned. This ADR fixes the shape of the layout, not the paths. The script `scripts/plan-status.sh` is to be built by mission-20261029-07, the mission that records this ADR.

This layout is intended to keep a later move to GitHub Issues or Backlog.md open. A small file per deliverable maps to an issue or a task, so a later import would not need the content reshaped.

## Consequences

**Positive**

- No new master checklist or operator copy is needed per milestone.
- Tick counts are to match the ticks, because the script builds them from the ticks.
- Files are to be smaller, so agents read less per task.
- The layout stays in git, so the existing crew checks are intended to keep working.

**Negative / risks**

- The old and new layouts exist side by side until M5 closes. Agents and operators must know which layout a given milestone uses.
- A results file per run means more small files in the folder.
- `scripts/plan-status.sh` is to count only the lines it can parse, and to report any other marker as unrecognised. Tick marks must stay in the formats listed in Decision item 3.
- There is no board view. Progress is seen as text, not as columns on a board.
