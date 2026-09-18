# Getting Started with Kirsch

You have cloned the repository and want to contribute. This file is your routing map — it points you at the right documents in the order that makes sense, without explaining them twice.

## Read This First — The Essentials

Get your bearings in this order. Stop here if you just want to understand what Kirsch is and what is happening right now.

1. **[../README.md](../README.md)** — What Kirsch is and why. *139 lines.* Read the first three sections; you can skip the Milestone Map if you only want the executive summary.

2. **[PROGRESS.md](PROGRESS.md)** — Where we are right now. *47 lines.* The milestone status table and current deliverables table. This is the single source of truth for what is in progress and what you can start next.

3. **[README.md](README.md)** — The build index. *98 lines.* The folder structure, what each document is for, and the rules for builders. Especially rule 3: "The UI spec is law."

## Understanding the Full Plan

Kirsch's instruction set is big — about 7,000 lines across the plan folder — so this section shows you what to read and what to skim.

**Always read these three** (about 1,200 lines together):

- **[spec/kirsch-plan.md](spec/kirsch-plan.md)** — The locked v0.1 spec. *1,034 lines.* Sections §1–§3 are essential: the contract, the architecture, and the tool contracts. Sections §4–§10 are reference; read them when you need them. §11 is the amendment log; skim it to understand what has been decided since the plan was written.

- **[spec/architecture.md](spec/architecture.md)** — Why the code is shaped this way. *336 lines.* Dependency rules (enforced by CI), the four interfaces that matter, concurrency model, error handling. If you touch `internal/`, read this first.

- **[spec/ui-spec-v0.1.md](spec/ui-spec-v0.1.md)** — The TUI specification. *837 lines.* **This is normative.** Everything visual must match this spec. Skim §1–§2 (layout, breakpoints), read §13 (the golden states), and refer to the rest as needed. When the spec and the code disagree, the spec is law.

**Read for the milestone you are working** (about 500 lines for Milestone 2):

- **Milestone instruction sets** — [milestones/milestone-2.md](milestones/milestone-2.md) (441 lines). Pick the milestone you are claiming work from, read the Ground Rules and the deliverables table, then read the tasks that belong to your deliverable. Later milestones are drafted further ahead; read them when they become current.

**Refer to on demand**:

- **[spec/kirsch-ui-screens.md](spec/kirsch-ui-screens.md)** — Screen reference. *842 lines.* **This is normative.** Every screen in the TUI is drawn here as a literal 80-column character grid. If you are building or testing anything visual, this is your render target and your acceptance test. Golden-test files are captured against these grids.

- **[testing/README.md](testing/README.md)** — How to verify Kirsch. *122 lines.* The commands and which to run before you commit. Includes pointers to the manual walkthroughs and the security tooling.

- **[adr/](adr/)** — Seven architecture decision records. *45–75 lines each.* Why specific decisions were made. Read one when you encounter a "why?" that the plan doesn't answer.

## Loading Files Into Your Assistant

When using an AI coding tool, hand this prioritized list to it as context. Load in this order:

**Always load** — these establish what you are building and where we are:
- `../AGENTS.md` — The five working rules and where every development rule lives. Many tools load this automatically.
- `../README.md`
- `PROGRESS.md`
- `README.md`

**Load for the current milestone** (Milestone 2, currently):
- `spec/kirsch-plan.md` (sections §1–§3 minimum; full file better)
- `spec/architecture.md`
- `spec/ui-spec-v0.1.md`
- `milestones/milestone-2.md`

**Load on demand**:
- `spec/kirsch-ui-screens.md` — required for any visual work; reference only otherwise
- `testing/README.md` — required before you run the test gate
- Any `adr/*.md` that answers a "why?" the plan doesn't cover
- `testing/automated-tests.md` and `testing/security.md` — required for understanding test coverage and security tooling

**Normative documents** — these are law, not advisory:
- `spec/ui-spec-v0.1.md` — everything visual must match this
- `spec/kirsch-ui-screens.md` — the render target and golden-test source

## Your First Contribution

1. **Verify the build works.** From the repo root:
   ```bash
   npm run check      # fmt + vet + lint + test + screens
   ```
   This must exit with status 0. If it doesn't, stop and report it.

2. **Read [PROGRESS.md](PROGRESS.md).** Find the current milestone and the deliverables table. Find a deliverable with status "not started" whose `Blocked by` column is empty (nothing blocking it).

3. **Read the milestone document** for that deliverable. Read the Ground Rules (all of them), the Deliverables section, and the Tasks for your deliverable only.

4. **Claim the deliverable** by editing [PROGRESS.md](PROGRESS.md): set the `Owner` column to your name and change `Status` to `in progress`. Create a branch following the naming convention in the milestone document.

5. **Do the work.** Complete the tasks for your deliverable. The milestone document is the spec; the `spec/ui-spec-v0.1.md` is law for anything visual. Refer to `spec/architecture.md` for dependency rules and error handling. Test early and often.

6. **Run the gate.** Before opening a pull request:
   ```bash
   npm run check      # fmt + vet + lint + test + screens
   npm run security   # vulnerabilities + static analysis
   npm run fuzz       # property tests
   ```
   All must exit 0.

7. **Open a pull request.** Reference the deliverable ID (e.g., "m2-d1") in the title. The [process/working-agreement.md](process/working-agreement.md) carries the full branch and PR conventions.

8. **Mark done** in [PROGRESS.md](PROGRESS.md) once the PR is merged. Update the `Status` to `done`.

## Where Things Live

- **[../AGENTS.md](../AGENTS.md)** — The five working rules: milestone order, dependency constraints, UI spec authority, amendment procedures, and the plan/doc split. Required reading for all builders and reviewers.
- **[plan/](.)** — Everything needed to **build** Kirsch. Three markdown files at the root (`README.md`, `START-HERE.md`, `PROGRESS.md`) and five directories (`spec/`, `adr/`, `milestones/`, `process/`, `testing/`). Scaffolding for the builder.
- **[../doc/](../doc/)** — Everything needed to **use** Kirsch: installation, configuration, troubleshooting. `usage.md` written now (363 lines, documenting layout, key bindings, slash commands, and the approval model); `install.md`, `configuration.md`, and `troubleshooting.md` arrive at Milestone 5.
- **[spec/](spec/)** — The locked v0.1 specification: what Kirsch is and how it must behave. See [`spec/README.md`](spec/README.md) for the index of the four specification documents.
- **[adr/](adr/)** — Seven architecture decision records explaining the big choices.
- **[testing/](testing/)** — Manual walkthroughs, test suite reference, security tooling.
- **[milestones/](milestones/)** — Per-milestone instruction sets. Read the current one before you code.
- **[process/](process/)** — Shared development practices: how two developers divide work and stay out of each other's files.
- **Repo root** ([../](../)) — Project README, license, and build automation.

---

**Questions?** Start with the document the prompt points you at. If it doesn't answer, the next step is usually the file the first one links to. The whole structure is built so you don't have to search.
