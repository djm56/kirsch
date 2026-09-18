# Milestone Instruction Sets

This folder holds the per-milestone instruction sets for building Kirsch — one file per milestone, each containing the deliverables and the detailed tasks.

## Milestones At A Glance

| Milestone | Title | Confidence | Notes |
|-----------|-------|-----------|-------|
| 0 | Repo skeleton + static TUI prototype | Executed | Instructions tested in the field. All acceptance checklist items verified. |
| 1 | Workspace engine + read-only tools | Executed | Instructions tested in the field. Eight unticked walkthrough items noted in [`../PROGRESS.md`](../PROGRESS.md); see that section for detail. |
| 2 | Patches, commands, approvals | Executable | Instructions written once, before building. Ready to execute as written. May need amendments as work progresses; a pattern that is now expected. |
| 3 | Provider + agent loop | Provisional | Settled in shape and structure. Detail is provisional and may change during M2 based on what is learned. Re-read against the code before starting. |
| 4 | Real task loop + sessions | Outline | A scope statement rather than detailed instructions. Task breakdown should expect to need real work during execution. |
| 5 | Polish + release | — | Written when Milestone 4 completes. (See section below.) |

Live status for each milestone lives in [`../PROGRESS.md`](../PROGRESS.md).

## How Confidence Levels Work

**Executed** (M0, M1) — These milestones have shipped. The instructions reflect what was actually built. Future reference purposes only.

**Executable** (M2) — Written once before building, never yet executed. Expected to need amendment as the builder's constraints meet the plan's shape. This is the only confidence level for instructions that have never been built.

**Provisional** (M3) — Settled in shape at the architectural level, but detail — task decomposition, specific APIs, error handling — may shift after M2 is complete. Re-read the milestone instructions against the actual Milestone 2 code before starting M3 work. The amendments from M2 will inform them.

**Outline** (M4) — A scope statement rather than a detailed instruction set. Expect to do real architectural work to break it down further. No amendments needed — the whole thing may be substantially rewritten as understanding grows.

## Milestone 5 — Not Yet Written

Kirsch's instruction sets are normally written **one at a time**, each after the previous milestone lands, because detail written three milestones ahead is almost always wrong by the time anyone reads it.

M2, M3, and M4 were drafted together on 2026-09-12 at the owner's request, which was a deliberate trade. The evidence for caution is close to hand: Milestone 1 produced seven amendments correcting instructions written just one milestone ahead, including a security hole in the containment algorithm.

M5 will be written after M4 ships, following the one-at-a-time pattern. It will cover:

- **Final polish** — refinement, edge cases, performance tuning
- **Release engineering** — binary distribution, version tagging, release notes
- **User documentation** — [`doc/`](../../doc/), the installer, first-run, configuration reference, troubleshooting

No task breakdown yet. The instruction set will describe the steps when the time comes.

## Deliverables

Each milestone carries a **Deliverables section** listing the discrete pieces of work, what each owns, and their dependencies. That section is *static definition* — it describes the structure and does not carry status or owner information.

Live status — who is working what, what is done, what is blocked — lives in [`../PROGRESS.md`](../PROGRESS.md), the single source of truth for current state.

When you claim a deliverable:

1. Check `PROGRESS.md`'s `Blocked by` column — if it names deliverables, all of them must be marked `done` before you start.
2. Edit `PROGRESS.md`: set `Owner` to your name and change `Status` to `in progress`.
3. Read the Ground Rules and your deliverable's tasks in this milestone's file.
4. When your deliverable's "done when" statement is fully satisfied, edit `PROGRESS.md`: change `Status` to `done`.

The Deliverables table carries no status or owner and is not edited to record progress. Changing the split itself is a plan change, agreed between both developers, not an individual edit.
