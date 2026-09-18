# Working on Kirsch

Read this before changing anything. It is short on purpose; the documents it
points at are the real thing.

This is now a shared repository. [`plan/process/working-agreement.md`](plan/process/working-agreement.md) covers claiming work and staying out of each other's files.

## Where the rules live

| Question | Document |
|---|---|
| What am I building, and in what order? | [`plan/spec/kirsch-plan.md`](plan/spec/kirsch-plan.md) — the locked v0.1 spec |
| Where do I start? | [`plan/START-HERE.md`](plan/START-HERE.md) — ingestion order, first-contribution path, cross-tool index |
| What milestone are we on? | [`plan/PROGRESS.md`](plan/PROGRESS.md) — live status, deliverables, blocking |
| How do two developers share the work? | [`plan/process/working-agreement.md`](plan/process/working-agreement.md) — claiming, branches, commits, PRs, file ownership |
| Why is the code shaped this way? | [`plan/spec/architecture.md`](plan/spec/architecture.md) — dependency rules, concurrency, error model |
| How should the UI behave? | [`plan/spec/ui-spec-v0.1.md`](plan/spec/ui-spec-v0.1.md) — normative for everything visual |
| What should the UI *look* like? | [`plan/spec/kirsch-ui-screens.md`](plan/spec/kirsch-ui-screens.md) — literal character grids |
| How do I verify it? | [`plan/testing/`](plan/testing/) — walkthroughs, suite reference, security |
| Why was X decided? | [`plan/adr/`](plan/adr/) — ADRs 0001–0007 |

## Five rules

1. **One milestone at a time, in order.** Complete the current milestone's
   acceptance checklist before starting the next. Do not build features from
   later milestones early, and do not build anything on the plan's out-of-scope
   list (§1).

2. **Respect the dependency rules.** `internal/agent` imports no implementation
   package; `internal/tui` imports no `provider`, `tool`, `workspace` or
   `agent`. These are enforced by a CI import check from Milestone 1, not by
   review. See [`plan/spec/architecture.md`](plan/spec/architecture.md) §3.

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
cmd/kirsch/         entry point: flags, wiring, every environment read
internal/tui/       Bubble Tea program — renders state, emits intents (M0)
internal/app/       the wiring layer: root context, routing, adapters (M1)
internal/workspace/ root detection, containment, denylist, ignore, detect (M1)
internal/tool/      tool envelope, registry, and the read-only tools (M1)
internal/config/    TOML config, four-layer precedence, secret refusal (M1)
internal/telemetry/ structured debug log — never stdout or stderr (M1)
internal/arch/      architecture tests only; no production code (M1)
plan/               build instructions — spec, architecture, ADRs, milestones
doc/                end-user documentation (Milestone 5)
testdata/           six fixture repositories (M1)
```

Each package is created by exactly one milestone; the table in
[`plan/spec/kirsch-plan.md`](plan/spec/kirsch-plan.md) §2 says which. If a task seems to
need a package from a later milestone, stop and check that table — either the
dependency is real and the plan needs amending, or the task is out of scope.

## Three things that have already bitten

Each cost real time; each is now a test. They are listed because the mistakes
are easy to repeat, not because the code is fragile.

1. **Never block the TUI's `Update`.** It is the deadlock architecture.md §5
   warns about and Milestone 1 hit for real (plan §11 amendment 46):
   `program.Send` waits for the event loop, and inside `Update` that loop cannot
   run. Nothing renders, and no error appears anywhere.
2. **A test double must not be more forgiving than the thing it replaces.** A
   buffered channel standing in for `program.Send` cannot express blocking, so
   the deadlock above existed only in the real program and every test passed.
3. **Reach for `--debug` early.** Scraping a pseudo-terminal is misleading —
   Bubble Tea redraws differentially, so a string genuinely on screen can be
   missing from the byte stream. The debug log located that deadlock in one run.

## Before you commit

```
npm run check      # fmt + vet + lint + test + screens
npm run security   # govulncheck + gosec + secret scan
npm run ci         # both of the above, then the race-enabled test run
```

Or directly, if you would rather not go through npm. This is exactly what the
two commands above expand to, in order:

```
# npm run check
golangci-lint fmt --diff             # must print nothing
go vet ./...
golangci-lint run
go test ./...
python3 scripts/lint-screens.py      # the character grids are executable

# npm run security
govulncheck ./...
gosec -quiet -exclude-generated ./...
bash scripts/secret-scan.sh
```

**`gofmt -l .` is not the formatting bar.** It stopped being so when
`.golangci.yml` grew a `formatters:` block: formatting is now gofmt, gofumpt
and goimports together, all three inside the one `golangci-lint` binary.
`golangci-lint fmt` rewrites files, `--diff` reports without writing, and
`golangci-lint run` fails on a gofumpt violation as well. A tree that only
satisfies `gofmt` can therefore look clean locally and still be red in CI.

Two notes on running them directly. The npm scripts reach every Go tool through
`scripts/go-tool.sh`, which takes a copy already on your PATH first and falls
back to `GOBIN` or `GOPATH/bin` — so the npm commands need no PATH change — and
which refuses a `golangci-lint` it can read as older than v2, warning and
carrying on where the version string will not parse. Invoking the tools
yourself means putting the install directory on your PATH first; `npm run
tools` installs them and prints the directory it used. And `npm run check`
runs the tests without `-race`; `npm run test:race` is the stricter run, and
the one every milestone's acceptance checklist asks for.

Run `npm run fuzz` as well after touching anything that handles paths, ignore
rules or tool input. It has found a real bug there before.
