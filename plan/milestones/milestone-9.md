# Milestone 9 — Front ends and machines (v0.5.0)

> **Status: Executable — written 2026-10-30 (mission-20261030-05) from `plan/build-plan-v0.2-v0.6.md`.** Live status lives in `plan/PROGRESS.md`; this page carries none.

## Ground rules

- The build plan's **Ground rules** paragraph applies unchanged (`plan/build-plan-v0.2-v0.6.md`, "How to apply this document"). One milestone at a time, in order.
- **Mission filter** (`plan/build-plan-v0.2-v0.6.md`, "How to apply this document"). Every mission checks that Kirsch still is: one loop, one session, one pane; patch-only editing; approval-first; local-first; tool output as data.
- **Confidence:** Re-read each set against the code before starting it; M7–M10 against the M6 code in particular, because the catalog and role types they build on will have been reshaped by then.
- Every code step goes to `@hicks`, and `@apone` reviews it as the very next step. `@vasquez` arrives only by escalation.
- Operator decisions (O-numbers) are recorded in `plan/PROGRESS.md` before the deliverable that names them. The build plan's default applies if the operator has not answered when the mission starts.
- Layout: ADR 0009 item 4 says a milestone page holds only its goal, scope and deliverables table. This page was filed from the build plan as written and carries more than that. Trimming it, and settling the per-deliverable and checklist paths, is open (see `plan/PROGRESS.md`, *Carried from the build plan filing*).
- Acceptance boxes here are unticked seed text. Ticks live in the checklists.

## Goal

An editor drives Kirsch over the Agent Client Protocol, and the TUI on one machine works a repository on another through an execution seam. Approvals, the session log and containment behave identically in every configuration.

## Waits on

`v0.4.0`; ADR 0012; O39–O40.

## Build order

`internal/executor` (local) → `internal/workspace` + `internal/tool` (use it) → `internal/executor/ssh` → `internal/acp` → `cmd/kirsch` → `doc/`.

## Deliverables

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m9-d1 | `Executor` interface, local | — | `internal/executor/{executor,local}.go` + tests; tools and workspace refactored to call it | 1 | Every read, list, `rg`, `git`, command and patch application goes through `Executor`; containment checked on the path string before the call; behaviour unchanged; fuzz suite still covers the untrusted surfaces |
| m9-d2 | Protected paths | m9-d1, O39 | `internal/policy`, `internal/executor/local.go` | 1 | Denylist paths and `.git/**` refused for writes by the executor itself, not only by the tool layer |
| m9-d3 | `ssh` executor | m9-d1, O40 | `internal/executor/ssh/` (commands via `ssh`, reads via `cat`/`ls`, search via remote `rg`, patches piped to remote `git apply --check` then `git apply`); `[workspace] executor = "ssh" host root` | 2 | M2 and M4 manual walkthroughs pass against a remote repo; session file, approvals and config stay local; missing remote `rg`/`git` is a doctor failure |
| m9-d4 | `kirsch acp` | m8-d1, O40 | `internal/acp/` on the chosen Go SDK + tests; `cmd/kirsch/acp.go` | 2 | `initialize`, `session/new`, `session/prompt`, `session/cancel`; text, thinking and tool calls stream as updates; each approval is an ACP permission request (allow-once / always / reject); `apply_patch` diffs sent as diff blocks; sessions are the same JSONL |
| m9-d5 | ACP conformance | m9-d4 | `internal/acp/conformance_test.go` with the SDK's example client | 3 | Scripted read-only task and approved-patch task pass; cancel mid-tool leaves no orphan process |
| m9-d6 | Docs and ADR | all | `doc/sandboxing.md` (devcontainer, Docker Sandbox, what approvals do not protect against), `doc/editors.md` (Zed, JetBrains), `doc/configuration.md`, `plan/adr/0012-*.md`, `CHANGELOG.md` | 3 | Each recipe run once by the operator |
| m9-d7 | Acceptance and tag | all | repository | gate | Zed and remote-repo walkthroughs pass; tag `v0.5.0` |

## Acceptance (operator)

- [ ] Zed with `kirsch acp`: prompt, tool call, approve a patch from Zed's dialog, see the diff.
- [ ] TUI on the Mac, `executor = "ssh"` to the Ubuntu box: read, search, patch, `go test`, resume; the remote sessions directory stays empty.
- [ ] Patch `.env` on the remote: refused before any ssh call.

## Not in this milestone

Container executor (`m9-d3b` only if ssh lands early), an OS sandbox of Kirsch's own, HTTP/WebSocket ACP, SDK package.

## Operator decisions

| ID | Needed by | Question | Default |
| --- | --- | --- | --- |
| O39 | m9-d2 | Read-only protected paths mechanism | Policy-only on local; executors with mounts enforce |
| O40 | m9-d3, m9-d4 | ACP SDK; ssh transport | `coder/acp-go-sdk`; system `ssh` with `ControlMaster` |

## Documentation

| Milestone | Page(s) | Guard |
| --- | --- | --- |
| M9 | `doc/sandboxing.md`, `doc/editors.md`, `doc/troubleshooting.md` (doctor lines) | doctor-line coverage test |

## Notes for whoever executes this

- O39's default ("Policy-only on local") and m9-d2's Done when (the executor itself refuses writes) disagree. The operator decides before m9-d2. Symlink handling for path-string containment is unspecified; see ADR 0012.
- ADR 0012 must be Accepted before Milestone 9 starts.
