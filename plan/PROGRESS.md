# Progress — Kirsch

**Where we are:** Milestone 2 — patches, commands, approvals. 1 of 6 deliverables done.

**What's next:** m2-d2 Policy. Unblocked, no dependencies. Owns `internal/policy/`.

This file is the single source of truth for live state — what is in progress and what you can start next. Milestone instruction documents carry static definition only (IDs, tasks, dependencies, ownership of files) and never carry status or owner information.

## Milestones

| # | Milestone | Status |
|---|---|---|
| 0 | Repo skeleton + static TUI prototype | ☑ Complete |
| 1 | Workspace engine + read-only tools | ☑ Complete |
| 2 | Patches, commands, approvals | ◐ In progress |
| 3 | Provider + agent loop | ☐ Not started |
| 4 | Real task loop + sessions | ☐ Not started |
| 5 | Polish + release (v0.1.0) | ☐ Not started |

## Milestone 2 — what's left

| ID | Title | Status | Owner | Waits on |
|---|---|---|---|---|
| m2-d1 | Patch infrastructure | done | — | — |
| m2-d2 | Policy | not started | — | — |
| m2-d3 | Approval flow | not started | — | m2-d2 |
| m2-d4 | Tool implementations | not started | — | m2-d1, m2-d2 |
| m2-d5 | TUI integration | not started | — | m2-d3, m2-d4 |
| m2-d6 | Testing & acceptance | not started | — | all |

## What has landed

| Completed | Deliverable | What landed |
|---|---|---|
| 2026-09-22 | m2-d1 Patch infrastructure | `internal/patch`: unified-diff parser, renderer, and atomic applier (standard library only, 44 tests). Fixtures at `testdata/repo-patch/` and diff corpus at `testdata/patches/`. |

---

**Outstanding verification:** see [testing/README.md](testing/README.md#outstanding-verification).
