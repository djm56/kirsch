# Progress — Kirsch

**Where we are:** Milestone 2 — patches, commands, approvals. 3 of 6 deliverables done.

**What's next:** m2-d4, Tool implementations — its dependencies are m2-d1 and m2-d2, both done, so it is unblocked.

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
| m2-d2 | Policy | done | — | — |
| m2-d3 | Approval flow | done | — | m2-d2 |
| m2-d4 | Tool implementations | not started | — | m2-d1, m2-d2 |
| m2-d5 | TUI integration | not started | — | m2-d3, m2-d4 |
| m2-d6 | Testing & acceptance | not started | — | all |

## What has landed

| Completed | Deliverable | What landed |
|---|---|---|
| 2026-09-23 | m2-d3 Approval flow | `internal/app`: `Approver.Request` blocks on a per-approval buffered channel from the tool goroutine and selects on the decision, the turn context and the root context, deleting its map entry on every path; `Resolve` never blocks, taking the lock only for the lookup and sending under `select`/`default`. Session approvals call `policy.Grant`. The TUI resolves through `wireCallbacks` in `cmd/kirsch`, with a `Cancelled` outcome distinct from rejection so an abandoned turn is not recorded as a refusal. Approve, reject, approve-for-session and escape-to-cancel are proven through real `Model.Update` dispatch; cancellation returns inside a second with no parked goroutines. |
| 2026-09-23 | m2-d2 Policy | `internal/policy`: the command allowlist and session-grant engine. Argv-slice matching so a semicolon-bearing element cannot be split; shells and runners refused at any install location by basename; `ForPatch` never auto-allows; grants refuse wildcards, empty prefixes and shells, and are disabled wholesale by config. Shipped allowlist entries are exact-match, so `go build -toolexec=` and `git diff --output=` require approval. |
| 2026-09-22 | m2-d1 Patch infrastructure | `internal/patch`: unified-diff parser, renderer, and atomic applier (standard library only, 44 tests). Fixtures at `testdata/repo-patch/` and diff corpus at `testdata/patches/`. |

---

**Outstanding verification:** see [testing/README.md](testing/README.md#outstanding-verification).
