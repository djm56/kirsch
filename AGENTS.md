# Working on Kirsch

Read this before changing anything. It is short on purpose; the documents it
points at are the real thing.

## Where the rules live

| Question | Document |
|---|---|
| What am I building, and in what order? | [`plan/kirsch-plan.md`](plan/kirsch-plan.md) — the locked v0.1 spec |
| What milestone are we on? | [`plan/README.md`](plan/README.md) — progress table and builder rules |
| Why is the code shaped this way? | [`plan/architecture.md`](plan/architecture.md) — dependency rules, concurrency, error model |
| How should the UI behave? | [`plan/ui-spec-v0.1.md`](plan/ui-spec-v0.1.md) — normative for everything visual |
| What should the UI *look* like? | [`plan/kirsch-ui-screens.md`](plan/kirsch-ui-screens.md) — literal character grids |
| Why was X decided? | [`plan/adr/`](plan/adr/) — ADRs 0001–0007 |

## Five rules

1. **One milestone at a time, in order.** Complete the current milestone's
   acceptance checklist before starting the next. Do not build features from
   later milestones early, and do not build anything on the plan's out-of-scope
   list (§1).

2. **Respect the dependency rules.** `internal/agent` imports no implementation
   package; `internal/tui` imports no `provider`, `tool`, `workspace` or
   `agent`. These are enforced by a CI import check from Milestone 1, not by
   review. See [`plan/architecture.md`](plan/architecture.md) §3.

3. **The UI spec is law; the screens are the render target.** Where an
   instruction set and the spec conflict, the spec wins — flag it rather than
   guessing. Where a screen and the spec conflict, the spec wins and the screen
   is the bug.

4. **Amend, don't drift.** A decision that contradicts the plan goes in the
   Amendment Log (`plan/kirsch-plan.md` §11), not in a commit message.

5. **`plan/` is for building Kirsch; `doc/` is for using it.** `doc/` stays
   empty until Milestone 5. Never put build notes there.

## Layout

```
cmd/kirsch/      entry point
internal/tui/    Bubble Tea program (Milestone 0)
plan/            build instructions — spec, architecture, ADRs, milestones
doc/             end-user documentation (Milestone 5)
```

Each package is created by exactly one milestone; the table in
[`plan/kirsch-plan.md`](plan/kirsch-plan.md) §2 says which. If a task seems to
need a package from a later milestone, stop and check that table — either the
dependency is real and the plan needs amending, or the task is out of scope.

## Before you commit

```
gofmt -l .          # must be empty
go vet ./...
go test ./...
```
