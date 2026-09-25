# Progress — Kirsch

**Where we are:** Milestone 2 — patches, commands, approvals. 4 of 6 deliverables done.

**What's next:** m2-d6, testing and acceptance; it depends on every other deliverable in the milestone and they are now all done.

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
| m2-d4 | Tool implementations | done | — | m2-d1, m2-d2 |
| m2-d5 | TUI integration | done | — | m2-d3, m2-d4 |
| m2-d6 | Testing & acceptance | not started | — | all |

## What has landed

| Completed | Deliverable | What landed |
|---|---|---|
| 2026-09-25 | m2-d5 TUI integration | `internal/tui` and `internal/app`: the approval surface now shows what it is asking about. Cards render the real request — a patch's file list, a command's argv, a rename named the same way on the card and in the modal title — and every text field is sanitised where the message is received, because the one screen that asks for consent was the only one in the transcript rendering untrusted content unchanged. The `[a]` key appears only where policy permits a session grant, and a refused grant corrects the card through a confirmed-outcome message correlated by approval id. The diff modal renders the parsed patch with per-file counts, and a change carrying no hunks says what it is rather than showing an empty body. `/approvals` lists the grants the policy actually holds and clears them through it, replacing a counter that production wrote and only tests read. `/patch` and `/run` are wired, labelled, and registered — the two tools that change the world were reachable from nowhere until this deliverable. Golden screen 06 is reconciled at 33 rows, the minimum the layout arithmetic allows. **Four items are recorded open, none of them regressions:** the help overlay clamps its body at common terminal heights with nothing on screen to say content is below the fold; a tool card's name and target reach the renderer unsanitised, the same defect this deliverable closed one card along; the box builder does not bound a row against its own width; and several tests carry names that outrun what their assertions can discriminate. See findings dated 2026-09-25. |
| 2026-09-25 | m2-d4 Tool implementations | `internal/tool`: `apply_patch` and `run_command`, the first tools that change the world and so the first that must ask first. `apply_patch` parses, resolves every path including rename sources, stages into temp files and only then requests approval, committing the bytes it validated rather than re-applying — so an invalid patch never produces a prompt. `run_command` takes an argv slice and never a shell, builds the child's environment from scratch and strips secrets last, reads stdin from the null device, confines the working directory, caps output at 200KB head-and-tail, and bounds cancellation with a process-group kill plus a `WaitDelay` backstop for descendants that leave the group. Supporting changes: a mutex and an operation-typed grant path in `internal/policy`, and a `Stage`/`Commit`/`Discard` seam in `internal/patch`. **Adversarial coverage is partial** — path escapes are solid, environment filtering has a known case-sensitivity gap, and the renamed-shell evasion is documented rather than tested. See findings dated 2026-09-25. |
| 2026-09-23 | m2-d3 Approval flow | `internal/app`: `Approver.Request` blocks on a per-approval buffered channel from the tool goroutine and selects on the decision, the turn context and the root context, deleting its map entry on every path; `Resolve` never blocks, taking the lock only for the lookup and sending under `select`/`default`. Session approvals call `policy.Grant`. The TUI resolves through `wireCallbacks` in `cmd/kirsch`, with a `Cancelled` outcome distinct from rejection so an abandoned turn is not recorded as a refusal. Approve, reject, approve-for-session and escape-to-cancel are proven through real `Model.Update` dispatch; cancellation returns inside a second with no parked goroutines. |
| 2026-09-23 | m2-d2 Policy | `internal/policy`: the command allowlist and session-grant engine. Argv-slice matching so a semicolon-bearing element cannot be split; shells and runners refused at any install location by basename; `ForPatch` never auto-allows; grants refuse wildcards, empty prefixes and shells, and are disabled wholesale by config. Shipped allowlist entries are exact-match, so `go build -toolexec=` and `git diff --output=` require approval. |
| 2026-09-22 | m2-d1 Patch infrastructure | `internal/patch`: unified-diff parser, renderer, and atomic applier (standard library only, 44 tests). Fixtures at `testdata/repo-patch/` and diff corpus at `testdata/patches/`. |

---

**Outstanding verification:** see [testing/README.md](testing/README.md#outstanding-verification).
