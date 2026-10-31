# Milestone 8 — Headless (v0.4.0)

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

Kirsch can be driven by a program: `kirsch run` executes one task without a TUI, streams session events as JSON lines, applies a written approval policy, can run in shadow mode that applies nothing, can commit what it was allowed to change, and can undo. After M8 Bishop runs a task end-to-end through Kirsch and the reviewer receives the exported session.

## Waits on

`v0.2.0` (M7 optional; if present, `--offline` works in `run`); O36–O38.

## Build order

`internal/session` (export, checkpoint) → `internal/policy` (file) → `internal/hook` → `internal/tool` (`git_commit`) → `internal/app` (headless) → `cmd/kirsch` → `doc/`.

## Deliverables

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m8-d1 | Headless driver | — | `internal/app/headless.go` + tests; `internal/arch` | 1 | `app` runs a session through a `Frontend` interface; the TUI implements it; a JSON frontend writes one event per line to stdout and reads nothing; import test proves no `tui` import |
| m8-d2 | `kirsch run` | m8-d1 | `cmd/kirsch/run.go` + tests | 1 | `kirsch run [--json] [--max-turns N] [--budget-usd X] [--agent P] [--model E/M] "task"` or `--task-file`/stdin; exit codes 0 done, 2 max-turns, 3 budget, 4 denied by policy, 5 provider, 130 interrupted; final report is the last text event |
| m8-d3 | Approval policy file | — | `internal/policy/file.go` + tests; `--approve path` | 1 | TOML answers every approval kind (`patches`, `commands` list split as m6-d11, `paths_deny`); anything unanswered is denied with exit 4; the TUI never reads it |
| m8-d4 | Shadow mode | m8-d2 | headless driver; `apply_patch` dry-run | 2 | `--shadow`: patches validated and recorded as `patch_proposed` with full diff, not applied; tree-modifying commands refused; report lists proposed patches |
| m8-d5 | Worktree | — | `internal/workspace/worktree.go` + tests; `--worktree [name]` | 2 | Creates or reuses `.kirsch/worktrees/<name>` via `git worktree add`; the root becomes the worktree; the instance lock is per worktree; recorded in the session |
| m8-d6 | Checkpoint and `/undo` | O38 | `internal/session/checkpoint.go` + tests; `apply_patch`; `/undo`; §7 | 2 | A checkpoint ref is recorded before each applied patch; `/undo` restores the last one and records `undo`; works after resume; refuses if the tree changed outside Kirsch |
| m8-d7 | Hooks, two events | O36 | `internal/hook/` + tests; `[hooks] post_patch`, `stop`; `hook_ran` event | 3 | Each hook runs with event JSON on stdin, 30s timeout, output captured and shown dimmed; non-zero exit reported, never a veto; same containment and env rules as `run_command` |
| m8-d8 | Verify-before-done nudge | — | `internal/agent/agent.go` + test; card | 3 | Text-only end after applied patches with no command since → a notice card offering the project test command; in `run` mode a `nudge` event only |
| m8-d9 | `git_commit` tool | O37 | `internal/tool/git_commit.go` + tests; `commit` approval kind; `system.md` | 3 | Commits only files touched by approved patches this session; model proposes the message, user approves; trailers `Kirsch-Session:` and `Kirsch-Model:`; never stages outside the touched set; never pushes |
| m8-d10 | Session export | — | `internal/session/export.go` + golden test; `kirsch session export --md`, `session list` | 3 | Markdown with task, models, files read, searches, patches (applied / rejected / proposed), commands with trimmed output, cost, final report |
| m8-d11 | Docs | all | `doc/headless.md` (new), `doc/configuration.md`, `doc/usage.md`, `CHANGELOG.md` | 4 | Event schema section generated from the Go types by a `scripts/` script and checked by a test; a 30-line Bishop driver example |
| m8-d12 | Acceptance and tag | all | repository | gate | Bishop walkthrough passes; tag `v0.4.0` |

## Acceptance (operator)

- [ ] `kirsch run --json --approve .kirsch/policy.toml --worktree t1 "add a test for X"` from a Bishop task; the harness parses the stream; exit 0.
- [ ] Same with `--shadow`; tree unchanged; `kirsch session export --md` shows the proposed diff.
- [ ] TUI: apply a patch, `/undo`; file restored; `git status` clean.
- [ ] `post_patch` hook running `gofmt -l` shows output as a card.
- [ ] `git_commit` after two patches; `git log -1` shows both trailers; no stray staged files.

## Not in this milestone

ACP, ssh or container executors, SDK, `pre_tool` veto hooks, fork.

## Operator decisions

| ID | Needed by | Question | Default |
| --- | --- | --- | --- |
| O36 | m8-d7 | Hook events in v0.4 | `post_patch` and `stop` only |
| O37 | m8-d9 | Ship `git_commit` in v0.4? | Yes, approval-gated; policies need explicit `commit = "allow"` |
| O38 | m8-d6 | Checkpoint mechanism | `git stash create` object + event; file copies when not a git tree |

## Documentation

| Milestone | Page(s) | Guard |
| --- | --- | --- |
| M8 | `doc/headless.md`, `doc/sessions.md` | event schema generated and tested |
