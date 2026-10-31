# ADR 0010 — Model catalog and roles

- **Status:** Proposed
- **Date:** 2026-10-30
- **Deciders:** Project owner (the operator), via the build plan (filed 2026-10-30); acceptance pending

## Context

v0.1 uses one provider configuration at a time. ADR 0003 fixed one provider for v0.1, and its "one provider" statement was already superseded in part by ADR 0008 (see ADR 0003's Status line). ADR 0008 made each endpoint configuration rather than code. It ships built-in endpoints, `opencode` and `anthropic`, and lets a user define more in the global config. Model facts such as context window, output limit and pricing sit in Go code today, in `internal/provider/models.go`.

The build plan (`plan/build-plan-v0.2-v0.6.md`) proposes a model catalog and per-role model choice for Milestone 6. Its amendment A5 is filed here as ADR 0010. The build plan proposed this number as 0009; it was renumbered because ADR 0009 (Planning layout from Milestone 6) already exists and ADR numbers are never reused.

The inputs are these sections of the build plan:

- **Plan amendments and new ADRs**, row A5: supersedes ADR 0003's "one provider" and ADR 0008's two fixed endpoints. That is the plan's wording, and ADR 0008 does not match it (see the first paragraph of this section). Model data moves to `models.toml`; the roles are `main`, `compact` and `quick`, with no fourth; and fallback is an ordered list.
- **Plan amendments and new ADRs**, row A2: new package `internal/model` (catalog and roles, M6). `internal/agent` imports `internal/model` interfaces only. `internal/tui` never imports `internal/model` and receives role and model names as strings.
- **Plan amendments and new ADRs**, row A3: a new `[models]` table with the keys `main`, `compact`, `quick` and `fallback`. Project config may set `[models]`, `[policy]` and `[hooks]`. Secrets stay env-only.
- **Decisions already taken**: three roles and no fourth. `main` drives turns, `compact` summarises, and `quick` does sub-second jobs. There is no automatic "best model for this prompt" routing.
- **Operator decisions**, O30: the default catalog format is TOML, embedded plus a user file.
- **Milestone 6**, rows m6-d1 and m6-d2: the catalog moves into data, and `Resolve(role)` returns an endpoint and its model information.

## Decision

Once this ADR is accepted, Kirsch adopts the following.

1. **One neutral `Request`.** Model-facing code builds one request type. Wire-specific detail stays in adapters (see ADR 0011, when accepted).
2. **Model data lives in data, not Go.** Model rows live in `models.toml`. Kirsch embeds a snapshot and reads a user file at `~/.config/kirsch/models.toml`. A user row overrides an embedded row with the same `id`. An unknown model ID still gets the conservative budget.
3. **Three roles and no fourth.** The roles are `main`, `compact` and `quick`. `compact` and `quick` default to `main`. A role that names an unknown endpoint fails at config load.
4. **Fallback is a list the user wrote.** `fallback` is an ordered list of endpoints the user sets. Kirsch never picks a fallback by heuristic.
5. **Package boundaries.** `internal/model` owns the catalog and roles. `internal/agent` depends on its interfaces only. `internal/tui` never imports it. `internal/arch/imports_test.go` is updated in the same mission as the package.

**Supersession.** When accepted, this ADR supersedes ADR 0003 in part, on its "one provider" statement, and ADR 0008 in part, on its statement that `[provider].default` names the endpoint in use (`plan/adr/0008-provider-endpoints.md:58`). How `[models].main` relates to `[provider].default` is not settled (see Consequences). The build plan's row A5 says "two fixed endpoints" for ADR 0008; ADR 0008 does not say that (see Context). ADR 0008 still governs the trust boundary for `[provider]`. The Status lines of ADR 0003 and ADR 0008 are deliberately **not** edited by this ADR. Updating them on acceptance is the operator's call.

## Consequences

**Positive**

- Model figures change by editing a data file, without a code change.
- Role resolution happens in one package, so the agent and the front end see names and interfaces, not provider details.
- Fallback follows an order the user wrote, so a failure never moves work to a model the user did not choose.

**Negative / risks**

- **Project `[models]`.** A3 lets project config set `[models]`. ADR 0008 forbids a project file from choosing the endpoint, the credential, the model or the thinking level, and it names `[provider]` only. This ADR does not settle whether a project `[models]` entry may name an endpoint or a model. That question must be answered before m6-d2 is built, or the trust boundary has a gap. The same question applies to `[policy]` and `[hooks]`. A project-set `[hooks]` entry would run commands from an untrusted repository without an approval card (ADR 0007). Amendment 76 (spec §5, line 255) currently ignores every project-file key except `[context].project_files`.
- **Precedence between `[models]` and `[provider]`.** Precedence between `[models].main` and `[provider].default` (ADR 0008, global-only) is unspecified. Decide before m6-d2.
- **Embedded figures can drift.** The embedded `models.toml` snapshot is a copy of published figures. Its currency depends on the refresh path (O30 default: `kirsch models refresh` imports models.dev JSON).
- **Fixed role count.** The role set may not fit a future need. The plan allows no fourth role.
- Re-read against the code before the milestone that builds it. The catalog and role types are reshaped by M6, and the milestones after M6 build on them.
- The Status lines of ADR 0003 and ADR 0008 now disagree with this ADR's supersession until the operator updates them.
