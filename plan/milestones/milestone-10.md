# Milestone 10 — Review and memory (v0.6.0)

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

Kirsch works while you are away and shows what it did: proposed patches queue in an inbox you review in the morning, a cheap model annotates each before you decide, a session can fork for a side quest, project knowledge lives in skills, and a replay suite proves behaviour without spending tokens.

## Waits on

`v0.3.0` and `v0.4.0`; O41–O42.

## Build order

`internal/session` (inbox, fork) → `internal/model` (opinion) → `internal/agent/prompt` (skills) → `internal/tui` (inbox mode) → `cmd/kirsch` → `scripts/` → `doc/`.

## Deliverables

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m10-d1 | Inbox store | m8-d4 | `internal/session/inbox.go` + tests; `patch_queued`, `patch_reviewed` events | 1 | `--shadow`/`--queue` runs write proposed patches with session id, base commit, diff; stale when base is no longer HEAD; JSONL under the state directory |
| m10-d2 | `kirsch review` | m10-d1 | `internal/tui/inbox.go` (new mode, ui-spec amendment, goldens); `cmd/kirsch/review.go` | 1 | Approve (normal checkpoint), reject, skip, edit-in-`$EDITOR`-then-approve; each decision recorded; stale items only reject or re-run |
| m10-d3 | Second-opinion line | m6-d2, m10-d1, O41 | `internal/model/opinion.go` + tests; `[review] second_opinion` (default false) | 2 | Before an approval card or inbox item, the `quick` role gives one line (≤ 120 chars) shown dimmed; timeout or error shows nothing and never blocks |
| m10-d4 | Session fork | O42 | `internal/session/fork.go` + tests; `/fork`, `/branches`; resume picks a branch; `fork` header | 2 | A child file names parent id and event index; pre-fork events read from the parent, never copied; compaction of a child never touches the parent |
| m10-d5 | Skills folder | — | `internal/agent/prompt/skills.go` + tests; `.kirsch/skills/*/SKILL.md`, `~/.config/kirsch/skills/`; `system.md` | 3 | Names and descriptions in a prompt section after the cache breakpoint; bodies read with `read_file` on demand; frontmatter errors are warnings; a skill cannot change policy |
| m10-d6 | `kirsch replay` | — | `internal/session/replay.go` + tests; fake provider plays recorded streams | 3 | Replays a session's provider events against real tools in a temp copy at the base commit with the session's policy; reports divergence in tool results or patches |
| m10-d7 | Nightly task set | m10-d6, m8-d2 | `scripts/nightly.sh`, `testdata/tasks/*.toml` (8–10 tasks on a Go repo and a WordPress plugin) | 4 | Runs each task with `kirsch run --shadow --json`, exports sessions, writes a summary table (pass, patches, cost, duration) Bishop can read; operator-triggered first, then cron on the Mac Mini |
| m10-d8 | Docs | all | `doc/review.md`, `doc/sessions.md`, `doc/skills.md`, `doc/usage.md`, `CHANGELOG.md` | 4 | The overnight recipe run once end-to-end by the operator |
| m10-d9 | Acceptance and tag | all | repository | gate | Overnight walkthrough passes; replay suite green in CI; tag `v0.6.0` |

## Acceptance (operator)

- [ ] Three tasks queued overnight with `--shadow --queue`; `kirsch review` in the morning shows patches with opinion lines; approve two, reject one; `git status` matches.
- [ ] `/fork` mid-session, fix a test on the branch, return; `/branches` lists both.
- [ ] A `wordpress-plugin` skill appears in `/status`; its body shows in a tool card only after the model reads it.
- [ ] `kirsch replay` of yesterday's session reports no divergence; a deliberate tool change reports one.

## Not in this milestone

Session search, inbox web UI, automatic approval of inbox items, MCP.

## Operator decisions

| ID | Needed by | Question | Default |
| --- | --- | --- | --- |
| O41 | m10-d3 | Second-opinion prompt ownership | Embedded `internal/model/opinion.md`, like `system.md` |
| O42 | m10-d4 | Fork storage | Separate file per branch, parent read-through |

## Documentation

| Milestone | Page(s) | Guard |
| --- | --- | --- |
| M10 | `doc/review.md`, `doc/skills.md`; `doc/llms.txt` generated by `scripts/` | — |
