# Milestone 4 — Instruction Set

> **Status: Outline drafted far ahead — expect to rewrite.** Written 2026-09-12
> alongside M2 and M3 at the owner's request. Plan §11 amendment 50 records the
> trade. This document is two milestones from execution, and the plan's own
> rules exist because detail at that distance is usually wrong: M1 produced
> seven corrections to instructions written *one* milestone ahead. Treat this as
> a scope statement with the hard problems identified — the task breakdown will
> need real work once M3 lands and the shape of the agent loop is known rather
> than assumed.

## Ground Rules (read first)

1. **Milestone 3 must be complete.** Every box ticked, CI green.
2. **This is the milestone where Kirsch becomes useful and dangerous at the same
   time.** It completes real tasks against real repositories. The smoke test is
   not a formality; it is the only thing that exercises the whole system with a
   real model.
3. **Locked decisions** (do not re-litigate):
   - The reconstruction rule (plan §7): resume rebuilds the provider message
     array from `user.message`, `assistant.message`, `assistant.thinking` and
     `tool.completed` — **never** from `assistant.delta`. Deltas exist to replay
     the *look* of streaming and are not the record of what was sent.
   - One goroutine owns all JSONL writes (architecture.md §3 rule 3).
   - Compaction rewrites only the *conversation*. The transcript and the log
     keep everything, which is why `/compact` does not make scrollback vanish.
4. Every task has a checkable result.

---

## What makes this milestone hard

**Three representations of one conversation, and they are not the same thing.**
architecture.md §6 names them: the transcript (what the user saw, untruncated),
the conversation (what the model sees, truncated and compacted), and the session
log (every event, append-only). Conflating any two is a bug, and this is the
milestone where all three exist simultaneously for the first time. Most of the
subtle failures here will be one of them standing in for another.

**Resume is not "load the file".** It has to reconstruct a *valid provider
message array* — every tool call answered, thinking blocks in place with their
signatures, compaction applied rather than replayed. A session that loads and
renders beautifully but produces an API-shape error on the next turn has failed.

**Durability has a cost and a deadline.** `fsync` every ~500ms and on turn
completion. A `kill -9` mid-write must leave a file that loads as `recovered`
rather than one that refuses to load at all.

---

## Task 1 — `internal/session`: the store

1. JSONL, one event per line, append-only, `"v":1` on every line (ADR 0002).
2. **Single writer goroutine.** Every other goroutine posts events to it. This
   is the rule that makes the file well-formed under concurrency, and it is
   easier to hold from the start than to retrofit.
3. Buffered, with `fsync` every ~500ms and on turn completion.
4. Storage layout per plan §7: `<workspace-hash>/<timestamp>-<ulid>.jsonl`,
   plus `index.json` (canonical path → last session) and `usage.json`.
5. **Repair on load.** A corrupt or torn final line truncates the session there
   and marks it `recovered`; startup never blocks on one bad file. Test it by
   truncating a real session mid-line, not by hand-crafting the damage.

## Task 2 — Resume

1. Rebuild the transcript **and** the conversation from the log, per the
   reconstruction rule above.
2. Restore pending state and session-scoped approval grants
   (`approval.scope_granted`), which M2 deliberately kept in memory only.
3. Apply `compaction.applied` rather than replaying the pre-compaction history.
4. `kirsch resume`, `--new`, and auto-resume of the last session per workspace
   via `index.json`.

**Check:** quit mid-task, resume, and get an identical transcript, identical
pending state, and identical grants — then *complete the turn*, which is the
part that proves the message array is valid.

## Task 3 — Single-instance guard

Advisory `flock` on `<workspace-hash>/.lock`. A second instance is **not
blocked**; it degrades (plan §7): starts fresh rather than auto-resuming, shows
`⚠ another Kirsch is running here`, and disables session-scoped grants so every
patch and command is approved individually. Two agents patching one working tree
is a foot-gun worth degrading for.

## Task 4 — Compaction

Per §6.4, and the thing to keep in view: **compaction is itself a model call.**

- Keeps the original task verbatim, the last four turns verbatim, and a
  structured summary of everything between.
- Cancellable. A cancelled compaction leaves the session uncompacted and the
  pending turn unstarted — never half-applied.
- On failure, Kirsch does **not** quietly send an oversized request. It surfaces
  `context_overflow` and tells the user to `/new`.
- Thinking blocks are dropped, not summarised.

## Task 5 — Slash commands, for real

`/status` `/diff` `/files` `/approvals` `/new` `/compact` `/quit` all operating
on real state. `/status` reports model, branch, dirty count, session id, tokens
(in/out/cache), compaction percentage, the project-context file loaded, and
active grants.

## Task 6 — Token and cost tracking

Added to `internal/telemetry`, which M1 built for this. Status bar and
`usage.json`. Unknown models degrade with a warning and a conservative budget
rather than failing.

## Task 7 — The final-task report

Enforced in the system prompt: summary, files changed, commands run with
pass/fail, limitations. This is what makes a completed task reviewable rather
than merely finished.

## Task 8 — Smoke test

`scripts/smoke-test.sh` against `testdata/repo-go-module` with a real model:

> "Add input validation to the Divide function and cover it with a test"

`calc/divide.go` has shipped without that validation since M1 precisely so this
test has something to do. The run must produce a shown patch, apply it after
approval, run `go test ./...`, and end with a report naming exactly the expected
files.

## Task 9 — Final acceptance

- [ ] Smoke test passes on a real model, end to end
- [ ] Quit mid-task, resume, and **complete the turn** — identical transcript,
      pending state and grants, and a valid message array
- [ ] A session truncated by `kill -9` loads as `recovered` and is resumable
- [ ] A second instance warns and degrades per §7
- [ ] Compaction is cancellable and never half-applies; failure surfaces
      `context_overflow` rather than an oversized request
- [ ] All slash commands operate on real state
- [ ] Token and cost tracking in the status bar and `usage.json`
- [ ] `go test -race ./...`, `gofmt`, `go vet`, `golangci-lint`, CI all clean

**Definition of done:** every box checked, CHANGELOG updated, progress set to
`☑ Complete`, commit tagged so Milestone 5 starts from a known point.

---

## Explicitly Out of Scope (reminder)

No `kirsch doctor`, no release tooling, no user documentation in `doc/`, no
session branching or search. Those are Milestone 5.

---

## Note for whoever executes this

Before starting, re-read this document against the code and rewrite what no
longer fits. It was written before Milestones 2 and 3 existed, and the honest
expectation is that the task breakdown is wrong in at least a few places —
particularly Tasks 2 and 4, which depend closely on the shape the agent loop
actually took. Amend the plan (§11) rather than working around the mismatch.
