# Kirsch — Specification

This folder holds **the locked v0.1 specification** — the four documents that define what Kirsch is and how it must behave.

This is different from `milestones/`, which says what to build *next*. The spec says what to build and the invariants it must hold. Everything visual in the TUI is law here; everything structural is law here; and where an instruction set contradicts the spec, the spec wins.

## The Four Documents

| Document | Answers | When to read |
|---|---|---|
| [`kirsch-plan.md`](kirsch-plan.md) (1034 lines) | What is Kirsch? What are the contract, the tool contracts, the policy, and the context? | When starting the project or a new milestone. Sections §1–§3 are essential; §4–§10 are reference. §11 is the amendment log tracking all decisions made since the plan was locked. |
| [`architecture.md`](architecture.md) (336 lines) | Why is the code shaped this way? What are the dependency rules, the key interfaces, the concurrency model, error handling? | Before touching `internal/`. Explains why specific hard rules exist and how they enforce the invariants. |
| [`ui-spec-v0.1.md`](ui-spec-v0.1.md) (837 lines) | What does the TUI look like? Layout, breakpoints, modes, key bindings, elements, palette, timing? | Before rendering anything visual. §1–§2 cover layout and breakpoints; §13 defines the fourteen golden states. **Normative** — everything visual must match this. |
| [`kirsch-ui-screens.md`](kirsch-ui-screens.md) (842 lines) | What does every golden state look like as a literal grid? | As the render target and source of truth for golden test files. Every state in §13 of the UI spec is drawn here as an 80-column character grid with a per-region colour map. **Normative** — draw to match these grids exactly. |

## Normative Rules

Where specifications conflict:

- **The UI spec (`ui-spec-v0.1.md`) is law for anything visual.** It says what the rules are. Sections §13 defines the fourteen states.
- **The screen reference (`kirsch-ui-screens.md`) is the render target and source of truth for golden files.** It shows what the UI spec describes — every state as a literal 80-column grid with exact colours.
- **If a screen and the spec disagree, the spec wins and the screen is the bug.** Fix the screen, never the spec, and never split the difference in code.
- **If an instruction set and the spec disagree, the spec wins.** Flag the conflict in an amendment rather than working around it.

## The Pinned File Warning

`kirsch-ui-screens.md` is not just documentation — it is executable specification, parsed by two tools:

- `internal/tui/golden_test.go` — parses every grid from this file and compares it against `View()` on every test run.
- `scripts/lint-screens.py` — validates the screen format and structure.

Both hardcode this file's path. **If you move or rename it, you must repoint both tools.** The test suite fails immediately with `t.Fatalf("read screen reference: %v", err)` (line 55 of `golden_test.go`) when the path is wrong, and the screen linter fails too—the breakage is unmissable.

## The Amendment Log

Decisions and changes from the original locked spec are recorded in [`kirsch-plan.md`](kirsch-plan.md) **§11, Amendment Log**. Read it to understand what has been decided since the plan was locked on 2026-09-11. A decision that contradicts the plan goes there rather than into a commit message — that is where a reader looks when asking "why isn't this in the original spec?"
