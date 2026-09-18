# Kirsch — Build Plan & Progress

This folder holds **everything needed to build Kirsch**: the locked v0.1 spec,
the architecture and UI specs, the ADRs, and the per-milestone instruction
sets. It is scaffolding for the builder.

[`doc/`](../doc/) is reserved for **documentation for people using Kirsch** —
install, configuration reference, keybindings, troubleshooting. Usage documentation
([`usage.md`](../doc/usage.md)) landed early because the help overlay inside the
program points readers at that file, so leaving it absent meant shipping a pointer
to nothing. The rest of `doc/` arrives at Milestone 5 when there is a working
system to document. If you are wondering which folder something belongs in, ask:
*is this for someone building Kirsch, or someone using it?* Building → `plan/`.
Using → `doc/`.

## Contents

| Document | Purpose |
|---|---|
| [`START-HERE.md`](START-HERE.md) | Your entry point — read this first. Routes you through the documents in the right order. |
| [`PROGRESS.md`](PROGRESS.md) | Live status — who is working what, what is blocked, what is done. The single source of truth for "where are we". |
| [`spec/`](spec/) | The locked v0.1 specification — contract, architecture, TUI spec, and screen reference. What Kirsch is and how it must behave. See [`spec/README.md`](spec/README.md) for the index and what each document answers. |
| [`process/working-agreement.md`](process/working-agreement.md) | Practices for shared work — claiming deliverables, branch naming, commits, PRs, file ownership via the Owns column in deliverables, handover contract freezing, amendment procedure, board discipline. |
| [`testing/`](testing/) | How to verify Kirsch — manual walkthroughs per milestone, what the automated suite covers, and the security tooling. See [`testing/README.md`](testing/README.md) for detail. |
| [`adr/`](adr/) | Architecture decision records 0001–0007, documenting why major design decisions were made. See [`adr/README.md`](adr/README.md) for the index and what an ADR is. |
| [`milestones/`](milestones/) | Per-milestone instruction sets. See [`milestones/README.md`](milestones/README.md) for the confidence levels and how milestones map to shipped code. |

## Milestone Instruction Sets

Kirsch is built in five milestones. Live status lives in [`PROGRESS.md`](PROGRESS.md). The milestone instruction sets, their confidence levels, and the links to each milestone's document live in [`milestones/README.md`](milestones/README.md).

Milestone instruction sets are normally written **one at a time**, each authored
after the previous milestone lands, because detail written three milestones
ahead is usually wrong by the time anyone reads it.

M2, M3 and M4 were drafted together on 2026-09-12 at the owner's request, so
that trade has been made deliberately and is recorded as plan §11 amendment 50.
Each carries a status banner saying how far it can be trusted: **M2 is
executable as written**, M3 is settled in shape but provisional in detail, and
M4 is a scope statement whose task breakdown should be expected to need real
work. The evidence for the caution is close to hand — Milestone 1 produced seven
amendments correcting instructions written just *one* milestone ahead.

Re-read the next milestone's document against the code before starting it, and
amend rather than work around a mismatch.

## Live Status

Live status — who is working what, what is blocked, what is done — lives in [`PROGRESS.md`](PROGRESS.md). It is the single place to look for "where are we", and it is edited one cell at a time as work actually changes rather than batch-updated at the end of a milestone.

A milestone is **Complete** only when every box in its instruction set's final
acceptance checklist is ticked, `go test ./...` is green, and CI is green. Not
before. A milestone that is "basically done except for tests" is in progress.

## Verifying

```bash
npm run check      # fmt + vet + lint + test + screens — the pre-commit gate
npm run security   # vulnerabilities, static analysis, secret scan
npm run fuzz       # property tests over the untrusted-input surfaces
```

[`testing/`](testing/) has the detail: a manual walkthrough per milestone, a
reference for what the automated suite does and does not cover, and how to
triage what the security tooling reports. Three real defects have come out of
that tooling so far — all recorded in §11 amendments 51–53, and none of them
found by reading the code.

## Rules for the builder

1. **One milestone at a time, in order.** Complete the acceptance checklist
   before starting the next one.
2. **Do not build features from later milestones early**, and do not build
   anything from the plan's out-of-scope list (§1).
3. **The plan is the spec; the UI spec is law for anything visual.** Where an
   instruction set and [`spec/ui-spec-v0.1.md`](spec/ui-spec-v0.1.md) appear
   to conflict, the spec wins — flag the conflict rather than guessing.
   [`spec/kirsch-ui-screens.md`](spec/kirsch-ui-screens.md) is the render target for
   that law: build the TUI to match those grids, and capture golden files
   against them. A screen that disagrees with the spec is the screen's bug —
   fix the screen, never the spec, and never split the difference in code.
4. **If a task seems to need a package from a later milestone, stop.** Check
   the package build order table in plan §2. Either the dependency is real
   (and the plan needs amending) or the task is out of scope.
5. **Amend, don't drift.** A decision that contradicts the plan gets written
   into the plan's Amendment Log (§11), not left in a commit message.
