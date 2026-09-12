# Kirsch — Build Plan & Progress

This folder holds **everything needed to build Kirsch**: the locked v0.1 spec,
the architecture and UI specs, the ADRs, and the per-milestone instruction
sets. It is scaffolding for the builder.

[`doc/`](../doc/) is reserved for **documentation for people using Kirsch** —
install, configuration reference, keybindings, troubleshooting. It is written
at Milestone 5 and is deliberately empty until then. If you are wondering which
folder something belongs in, ask: *is this for someone building Kirsch, or
someone using it?* Building → `plan/`. Using → `doc/`.

## Contents

| Document | Purpose |
|---|---|
| [`kirsch-plan.md`](kirsch-plan.md) | The locked v0.1 spec: contract, architecture, tool contracts, policy, config, context, events, milestones, testing |
| [`architecture.md`](architecture.md) | Why the codebase is shaped this way — dependency rules and their enforcement, the four interfaces that matter, concurrency and cancellation, the three representations of a conversation, error model, anti-goals |
| [`ui-spec-v0.1.md`](ui-spec-v0.1.md) | Full TUI spec — layout and breakpoints, transcript elements, mode state machine, bindings per mode, slash commands, text sanitisation, palette, timing, accessibility, golden-test surface |
| [`kirsch-ui-screens.md`](kirsch-ui-screens.md) | Screen reference — every ui-spec §13 golden state drawn as a literal 80-column character grid, each with a per-region colour map. What the spec describes in prose, this draws |
| [`adr/`](adr/) | Architecture decision records 0001–0007 |
| [`milestone-0.md`](milestone-0.md) | Instruction set — repo bootstrap + static TUI prototype |
| [`milestone-1.md`](milestone-1.md) | Instruction set — workspace engine + read-only tools |
| [`milestone-2.md`](milestone-2.md) | Instruction set — patches, commands, approvals |
| [`milestone-3.md`](milestone-3.md) | Instruction set — provider + agent loop *(drafted ahead; refine before executing)* |
| [`milestone-4.md`](milestone-4.md) | Outline — real task loop + sessions *(drafted far ahead; expect to rewrite)* |
| `milestone-5.md` | *Written when Milestone 4 completes* |

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

## Progress

Update this table as milestones complete. It is the single place to look for
"where are we".

| # | Milestone | Status | Instruction set |
|---|---|---|---|
| 0 | Repo skeleton + static TUI prototype | ☑ Complete | [done](milestone-0.md) |
| 1 | Workspace engine + read-only tools | ☑ Complete | [done](milestone-1.md) |
| 2 | Patches, commands, approvals | ☐ Not started | [ready](milestone-2.md) |
| 3 | Provider + agent loop | ☐ Not started | [draft](milestone-3.md) |
| 4 | Real task loop + sessions | ☐ Not started | [outline](milestone-4.md) |
| 5 | Polish + release (`v0.1.0`) | ☐ Not started | not written |

Status values: `☐ Not started` · `◐ In progress` · `☑ Complete`.

A milestone is **Complete** only when every box in its instruction set's final
acceptance checklist is ticked, `go test ./...` is green, and CI is green. Not
before. A milestone that is "basically done except for tests" is in progress.

## Rules for the builder

1. **One milestone at a time, in order.** Complete the acceptance checklist
   before starting the next one.
2. **Do not build features from later milestones early**, and do not build
   anything from the plan's out-of-scope list (§1).
3. **The plan is the spec; the UI spec is law for anything visual.** Where an
   instruction set and [`ui-spec-v0.1.md`](ui-spec-v0.1.md) appear
   to conflict, the spec wins — flag the conflict rather than guessing.
   [`kirsch-ui-screens.md`](kirsch-ui-screens.md) is the render target for
   that law: build the TUI to match those grids, and capture golden files
   against them. A screen that disagrees with the spec is the screen's bug —
   fix the screen, never the spec, and never split the difference in code.
4. **If a task seems to need a package from a later milestone, stop.** Check
   the package build order table in plan §2. Either the dependency is real
   (and the plan needs amending) or the task is out of scope.
5. **Amend, don't drift.** A decision that contradicts the plan gets written
   into the plan's Amendment Log (§11), not left in a commit message.
