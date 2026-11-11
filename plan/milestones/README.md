# Milestone Instruction Sets

This folder holds the per-milestone instruction sets for building Kirsch — one file per milestone, each containing the deliverables and the detailed tasks.

## Milestones At A Glance

| Milestone | Title | Confidence | Notes |
|-----------|-------|-----------|-------|
| 0 | Repo skeleton + static TUI prototype | Executed | Instructions tested in the field. All acceptance checklist items verified. |
| 1 | Workspace engine + read-only tools | Executed | Instructions tested in the field. Eight unticked walkthrough items noted in [`../testing/README.md`](../testing/README.md#outstanding-verification); see that section for detail. |
| 2 | Patches, commands, approvals | Executed | Complete. See [`../PROGRESS.md`](../PROGRESS.md) for deliverable status and task breakdown. |
| 3 | Provider + agent loop | Executed | Complete. Written before Milestone 2 landed; see [`../PROGRESS.md`](../PROGRESS.md) for deliverable status. |
| 4 | Real task loop + sessions | Executed | Complete and operator-accepted 2026-10-23 (tag `m4`). The narrow-width status-bar fallback, carried to Milestone 5 as m5-d1, was resolved on 2026-10-27 as a wrong expectation, not a code defect (see [`../PROGRESS.md`](../PROGRESS.md)). |
| 5 | Polish + release (v0.1.0) | Executable | Instruction set and testing skeletons written 2026-10-23–24 (mission-20261023-06). In progress. Which deliverables are done, and what is still open, is in [`../PROGRESS.md`](../PROGRESS.md), the live status file. Complete definition is in `milestone-5.md`. (See section below.) |
| 6 | More than one model (v0.2.0) | Executable | Written 2026-10-30 (mission-20261030-05) from [`../build-plan-v0.2-v0.6.md`](../build-plan-v0.2-v0.6.md). Not started. Definition: [`milestone-6.md`](milestone-6.md). Start: [`milestone-6-start-here.md`](milestone-6-start-here.md). |
| 7 | More than one wire (v0.3.0) | Executable | Written 2026-10-30 (mission-20261030-05) from [`../build-plan-v0.2-v0.6.md`](../build-plan-v0.2-v0.6.md). Not started. Definition: [`milestone-7.md`](milestone-7.md). |
| 8 | Headless (v0.4.0) | Executable | Written 2026-10-30 (mission-20261030-05) from [`../build-plan-v0.2-v0.6.md`](../build-plan-v0.2-v0.6.md). Not started. Definition: [`milestone-8.md`](milestone-8.md). |
| 9 | Front ends and machines (v0.5.0) | Executable | Written 2026-10-30 (mission-20261030-05) from [`../build-plan-v0.2-v0.6.md`](../build-plan-v0.2-v0.6.md). Not started. Definition: [`milestone-9.md`](milestone-9.md). |
| 10 | Review and memory (v0.6.0) | Executable | Written 2026-10-30 (mission-20261030-05) from [`../build-plan-v0.2-v0.6.md`](../build-plan-v0.2-v0.6.md). Not started. Definition: [`milestone-10.md`](milestone-10.md). |

Live status for each milestone lives in [`../PROGRESS.md`](../PROGRESS.md).

## How Confidence Levels Work

**Executed** (M0–M4) — These milestones have shipped. The instructions reflect what was actually built. Future reference purposes only.

**Executable** — Written once before building, with amendment expected as the builder's constraints meet the plan's shape. Unlike Executed, these instructions describe the planned approach before implementation reshapes the design. M2 and M3 were at this level before they shipped.

**Outline** — A scope statement rather than a detailed instruction set. Expect to do real architectural work to break it down further. No amendments needed — the whole thing may be substantially rewritten as understanding grows. M4 was at this level until its detailed rewrite on 2026-10-13.

## Milestone 5

The instruction set is written (mission-20261023-06). [`milestone-5.md`](milestone-5.md) carries the deliverables, the decisions table and the waves. Each deliverable is a seed for one mission, and the file's Deliverables table names them. Live status is in [`../PROGRESS.md`](../PROGRESS.md). Start with [`milestone-5-start-here.md`](milestone-5-start-here.md) — the plain-language front door: where Milestone 5 is, what to do next, the prompt to paste, and which piece can start.

## Deliverables

Each milestone carries a **Deliverables section** listing the discrete pieces of work, what each owns, and their dependencies. That section is *static definition* — it describes the structure and does not carry status or owner information.

Live status — who is working what, what is done, what is blocked — lives in [`../PROGRESS.md`](../PROGRESS.md), the single source of truth for current state. From Milestone 6, its tick counts come from [`scripts/plan-status.sh`](../../scripts/plan-status.sh).

When you claim a deliverable:

1. Check `PROGRESS.md`'s `Waits on` column — if it names deliverables, all of them must be marked `done` before you start.
2. Edit `PROGRESS.md`: set `Owner` to your name and change `Status` to `in progress`.
3. Read the Ground Rules and your deliverable's tasks in this milestone's file.
4. When your deliverable's "done when" statement is fully satisfied, edit `PROGRESS.md`: change `Status` to `done`.

The Deliverables table carries no status or owner and is not edited to record progress. Changing the split itself is a plan change, agreed between both developers, not an individual edit.

**From Milestone 6** (ADR 0009, 2026-10-29), a milestone page holds only its goal, its scope and a deliverables table. The tasks for each deliverable sit in a small file per deliverable. The exact file names are settled when Milestone 6 is planned. Milestones 0 to 5 keep their current layout. The claim procedure above still applies, and the Owner and Status cells in `PROGRESS.md` stay hand-kept. Details are in [`../adr/0009-planning-layout-from-m6.md`](../adr/0009-planning-layout-from-m6.md).
