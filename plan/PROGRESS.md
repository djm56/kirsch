# Progress — Kirsch Milestones

This file is canonical for *live state* — status, owner, and blocked-by relationships as work actually changes. The Deliverables tables in each milestone document are the *static definition* — IDs, dependencies, and file ownership — and never carry status or owner.

## Milestone Status

| # | Milestone | Status | Instruction set |
|---|---|---|---|
| 0 | Repo skeleton + static TUI prototype | ☑ Complete | [milestones/milestone-0.md](milestones/milestone-0.md) |
| 1 | Workspace engine + read-only tools | ☑ Complete | [milestones/milestone-1.md](milestones/milestone-1.md) |
| 2 | Patches, commands, approvals | ☐ Not started | [milestones/milestone-2.md](milestones/milestone-2.md) |
| 3 | Provider + agent loop | ☐ Not started | [milestones/milestone-3.md](milestones/milestone-3.md) |
| 4 | Real task loop + sessions | ☐ Not started | [milestones/milestone-4.md](milestones/milestone-4.md) |
| 5 | Polish + release (v0.1.0) | ☐ Not started | Not yet written |

## Current Milestone — Deliverables

Milestone 2 work begins here. Each deliverable specifies what it owns and which other deliverables must be complete before work can start.

| ID | Title | Status | Owner | Blocked by |
|---|---|---|---|---|
| m2-d1 | Patch infrastructure | not started | — | — |
| m2-d2 | Policy | not started | — | — |
| m2-d3 | Approval flow | not started | — | m2-d2 |
| m2-d4 | Tool implementations | not started | — | m2-d1, m2-d2 |
| m2-d5 | TUI integration | not started | — | m2-d3, m2-d4 |
| m2-d6 | Testing & acceptance | not started | — | all |

## How to Claim a Deliverable

1. Check the `Blocked by` column — if it names any deliverables, they must all be marked `done` before you start.
2. If clear, edit this file: set `Owner` to your name and change `Status` to `in progress`.
3. Work on the tasks listed in the static table in [milestones/milestone-2.md](milestones/milestone-2.md). Branch name follows the convention `m2-d[N]-<short-title>` (e.g., `m2-d1-patch-infrastructure`).
4. When the deliverable's "done when" statement in the milestone document is fully satisfied, edit this file: change `Status` to `done`.

## Outstanding Verification

Eight manual walkthrough checks remain unticked despite Milestones 0 and 1 being marked complete:

- `testing/manual/milestone-0-donovan.md` — seven unticked items: all are the `Ctrl+G` / `g` / `G` re-pin checks. These were added by a later mission that built the re-pin feature, so they postdate the original walkthrough run rather than having been skipped.
- `testing/manual/milestone-1.md` — one unticked item: "Outside a repository it refuses with a message naming `--workspace`" — the failure path for workspace detection.

This section exists to keep the gap visible rather than lost. The operator has confirmed Milestones 0 and 1 are tested and is moving forward; close these unticked items as part of Milestone 2's acceptance work or afterward as a cleanup step.

## Update Discipline

This file is edited one cell at a time as work actually changes state — never batch-updated at the end of a milestone. Anyone may correct a cell that disagrees with reality, and if this file ever disagrees with reality, reality wins and the file gets fixed immediately. Treat this as the ground truth for what is in progress and what you can start next.
