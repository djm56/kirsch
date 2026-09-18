# Working Agreement — Shared Development

This document is how two developers divide work on the same milestone and stay out of each other's files. It introduces a practice rather than describing an existing one — until now, Kirsch has been built solo, with commits going straight to `main`. Read this before claiming a deliverable and before opening a pull request.

## Claiming Work

Work is claimed through [`PROGRESS.md`](../PROGRESS.md), which is the single source of truth for what is in progress and what you can start next. See the claim procedure in `PROGRESS.md` for the process.

Two people want the same deliverable? They coordinate synchronously. If everything unblocked is already claimed and you want to move forward, the honest answer is to pair on the next unblocked deliverable — solving two problems at once — or to work the sequential tail together. The board exists to prevent thrashing, not to create queues; use it that way.

## Branches

Name your branch **`m<milestone>-d<deliverable>-<short-title>`** — matching the deliverable ID so it is traceable to a row on `PROGRESS.md`. One branch per deliverable.

If a deliverable turns out to be bigger than expected and should have been split, stop and raise it — the split is a plan change, not something to work around by hiding two deliverables in one branch. The `Owns` column in each milestone's Deliverables table is the contract; changing it silently breaks things for the other developer.

## Commits

Use imperative mood and sentence case, matching the existing commit style in this repository (e.g., "Add a frame margin, a two-row header, and Ctrl+G re-pin"). Reference the deliverable ID somewhere a reader can grep for it — either in the commit message or in the PR title.

A decision that contradicts the plan goes in the Amendment Log (`spec/kirsch-plan.md` §11), never a commit message. See [../../AGENTS.md](../../AGENTS.md) rule 4 for what "contradicts" means.

## Pull Requests

The title must carry the deliverable ID (e.g., "m2-d1"). CI already runs on pull requests targeting `main` — it must be green before you merge.

Review expectation: with a team of two, the other person reviews when they have space. If they are deep in their own deliverable, they might not have capacity right now — that is real and is not a blocker. The realistic options are to merge once you have verified everything locally and documented your choices in the PR, or to pair with them on review if the change is high-risk. Either is fine; review-queue backlog is not a blocker in a two-person team.

## Staying Out of Each Other's Files

The **`Owns` column** in each milestone's Deliverables table is the contract. Before you edit a file, check whose deliverable owns it.

Your task genuinely needs a file you do not own? Ask first, do not just edit. Changing a file you do not own silently breaks the other person's work.

If two concurrent deliverables need to share a file, that is a defect in the split — it should have been one deliverable or the owners should have been clear. Raise it rather than working around it. A temporary shared edit is tempting and invisible — and two people editing the same file in different branches is how silent conflicts happen, even when they "don't overlap".

**Milestone 2's case:** The genuinely concurrent pairs are m2-d1 paired with m2-d2, and m2-d3 paired with m2-d4. Neither pair shares a file. The tail — m2-d5 and m2-d6 — runs sequentially. This milestone has no concurrent file collision.

## Handover Between Dependent Deliverables

When deliverable B depends on deliverable A, B starts its work against A's finished work. Two things matter:

1. **A is done when its "done when" statement is met**, not when it is nearly done or almost done. "Nearly" is a reason to ask; it is not a reason to start work against half-finished types. A depends on stability, and B needs to know what it depends on won't change tomorrow.

2. **Once B has started, the types A defined are frozen.** The handover contract is a snapshot. In Milestone 2, m2-d3 defines `ApprovalRequestedMsg` and other types in `messages.go` that m2-d5 consumes. Once m2-d5 starts writing code against those types, changing them breaks m2-d5's work silently — commit messages will not warn. Agree on the contract before B starts, and both hold it.

## When You Disagree with the Plan

A decision that changes the plan goes in the Amendment Log (`spec/kirsch-plan.md` §11). See [../../AGENTS.md](../../AGENTS.md) rule 4 for the full procedure.

The two-person dimension: a decision that changes a shared contract — a type signature, a file boundary, or what a deliverable owns — gets agreed before it is written, not discovered in a diff. "Discovered in a diff" means the other person has to rewrite their work.

## Verifying and Testing

How two developers divide the testing burden is in [`testing/README.md`](../testing/README.md). Read it before claiming a deliverable to understand who runs what, when.

The one rule that belongs in a working agreement rather than a testing guide: **you do not merge on someone else's green.** CI on the pull request is the gate. Your local test run passing is validation, not permission. CI on the PR must be green before you merge, and CI is the same for both developers — no shortcuts, no exceptions.

## Keeping the Board Honest

Update [`PROGRESS.md`](../PROGRESS.md) when state actually changes — when you move from "not started" to "in progress" or from "in progress" to "blocked" or "done" — not at the end of the week. A board nobody trusts is worse than no board.
