# Milestone 6 — More than one model (v0.2.0)

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

A session holds several configured endpoints, picks the model per role, switches models from the TUI, survives a transient provider failure, and refuses the three failure modes that waste quota overnight. Only the existing Messages-format adapter is used.

## Waits on

`v0.1.0`; A1–A5 and A7 recorded; ADR 0010 accepted; O30–O32 answered.

## Build order

`internal/model` → `internal/provider` (retry) → `internal/agent` → `internal/app` → `internal/tui` → `internal/config` → `cmd/kirsch` → `doc/`.

## Deliverables

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m6-d1 | Model catalog in data | O30 | `internal/model/catalog.go` + tests, embedded `models.toml`, loader for `~/.config/kirsch/models.toml`; `internal/provider/models.go` removed | 1 | Every row of today's `models.go` exists in `models.toml` with identical figures; unknown IDs still get the conservative budget; a user row overrides an embedded row by `id`; each field incl. `compat` is pinned by a test |
| m6-d2 | Roles and resolution | m6-d1 | `internal/model/roles.go` + tests; `[models]` keys in config | 1 | `Resolve(role)` returns endpoint + `ModelInfo`; `compact` and `quick` default to `main`; an unknown endpoint in a role fails at config load |
| m6-d3 | `kirsch models` | m6-d1, m6-d2 | `cmd/kirsch/models.go` + tests | 1 | Lists endpoint, model, window, max output, pricing kind, role(s); `--json`; works outside a repo |
| m6-d4 | Retry with backoff | O31 | `internal/provider/anthropic/retry.go` + tests | 2 | 429/529/408 and connection errors retry per O31 honouring `Retry-After`; 400/401/403 never; each retry is an `EventError` of kind `retrying`; cancel aborts a waiting retry at once |
| m6-d5 | Fallback endpoint | m6-d2, m6-d4 | `internal/app/provider_model.go` + tests; `fallback_used` event; §7 | 2 | After retries on `main` are exhausted the turn is re-sent once to the next `fallback` entry; thinking blocks from the failed attempt are discarded; a card names the endpoint that answered |
| m6-d6 | Truncation guard | — | `internal/agent/agent.go` + test | 2 | On stop reason `max_tokens` every tool call in the message is failed unexecuted with a fixed message; the next turn tells the model why |
| m6-d7 | Doom-loop approval card | m6-d6 | `internal/agent/loop_guard.go` + tests; new card kind in `internal/tui/cards.go` | 2 | SHA-256 of tool name + canonical args; the third identical consecutive call raises an approval card instead of running; reject ends the turn; window resets on user input |
| m6-d8 | Budget cap | — | `internal/app/budget.go`; `--budget-usd`; `[session] max_cost_usd` | 2 | A turn that would exceed the cap is not started and a card says so; flat-rate endpoints are exempt and `/status` says so |
| m6-d9 | `/model` picker and `--model` | m6-d2, m6-d3 | `internal/tui/modal.go`, `update.go`, goldens; `model_changed` event; `cmd/kirsch` | 3 | Picker shows only endpoints with a key present; changes `main` for this session; refused while a turn runs; `kirsch resume` restores it; status bar shows `endpoint/model` |
| m6-d10 | Compaction on `compact` | m6-d2 | `internal/session/compact.go` + tests | 3 | Summariser uses the `compact` role; the original task text is re-inserted verbatim after the summary; `/status` names the compact model |
| m6-d11 | Per-project policy, command splitting | O32 | `internal/config` (project `[policy]`), `internal/policy/split.go` + fuzz | 3 | `.kirsch/config.toml` may set `[policy]`; commands split on `&&`, `\|\|`, `;`, `\|` (+O32) and each segment matches independently |
| m6-d12 | Markdown slash commands | — | `internal/command/` + tests; composer completion; `doc/usage.md` | 3 | `.kirsch/commands/*.md` and `~/.config/kirsch/commands/*.md` load at start and on `/reload`; `/name args` substitutes `$ARGUMENTS`; built-ins win; the five M1 debug commands are removed |
| m6-d13 | Agent profiles | m6-d2, m6-d11 | `internal/agent/profile.go` + tests; `.kirsch/agents/*.md` loader; `--agent`, `/agent`; `agent_changed` event | 4 | Frontmatter sets role, allowed tools, policy preset, prompt overlay; built-in `review` (read-only, `quick`, no `apply_patch`); import test proves `tui` sees only the name |
| m6-d14 | Docs and ADR | all | `doc/configuration.md`, `doc/usage.md`, `doc/providers.md` (new), `plan/adr/0010-*.md`, `README.md`, `CHANGELOG.md` | 5 | S36 passes; new slash-command coverage test passes; one CHANGELOG line per user-visible change |
| m6-d15 | Acceptance and tag | all | repository | gate | Checklist all-green; live smoke on two endpoints; CI green; tag `v0.2.0` |

## Acceptance (operator)

- [ ] `anthropic` as `main`, `opencode` as `compact`; a task that compacts; `/status` shows both.
- [ ] `/model` to `opencode/minimax-m3` mid-session; quit; `kirsch resume` shows it.
- [ ] Invalid key on `main`, `fallback = ["opencode"]`; the turn completes on opencode and the card says so.
- [ ] Fake-provider loop fixture; third call shows the doom-loop card.
- [ ] Project policy allowlisting `go test ./...` runs silently; `go test ./... && rm -rf /tmp/x` prompts.
- [ ] `/test` from `.kirsch/commands/test.md` sends the expanded template.
- [ ] `kirsch --agent review` cannot propose a patch and says why.

## Not in this milestone

OpenAI-format endpoints, discovery, offline mode, `kirsch run`, hooks, fork.

## Operator decisions

| ID | Needed by | Question | Default |
| --- | --- | --- | --- |
| O30 | m6-d1 | Catalog format | TOML embedded + user file; `kirsch models refresh` imports models.dev JSON |
| O31 | m6-d4 | Retry budget | 3 attempts, 1s / 4s / 16s, `Retry-After` wins, 60s ceiling |
| O32 | m6-d11 | Split on newlines and `$(...)` too? | Newlines yes; `$(`, backticks or `eval` never match the allowlist |

## Documentation

| Milestone | Page(s) | Guard |
| --- | --- | --- |
| M6 | `doc/providers.md` (Messages endpoints, roles), `doc/configuration.md`, `doc/usage.md`, `doc/how-it-works.md` (from `architecture.md` and the ADRs) | S36; slash-command coverage test |

## Notes for whoever executes this

- Retry with back-off already exists in `internal/provider/anthropic/client.go`: constants at :143–145 (`maxAttempts` 4, `baseBackoff` 500ms, `maxRetryAfter` 60s), the loop in `Stream` from :194, `isRetryable` at :303 and `retryWait` at :320. m6-d4 reshapes it to O31 and moves it to `retry.go`. Existing behaviour the m6-d4 row does not mention: every 5xx is retried today, not only 529; a 429 recognised as a lockout is not retried; a 402 (billing, ADR 0008) is not retried. And `Stream` promises exactly one terminal `EventError` (:162–167), so retry events sent as `EventError` change that contract. Settle these by amendment before m6-d4.
- A3 lets project config set `[models]`. Amendment 76 makes project config an allowlist (`plan/spec/kirsch-plan.md:255`), and ADR 0008 bars project files from `[provider]` keys, which choose the endpoint or model (`plan/adr/0008-provider-endpoints.md:100–108`). That conflict needs an operator decision before m6-d2 (and before m6-d11 for `[policy]`, m8-d7 for `[hooks]`). See ADR 0010 risks.
- O22 (recorded 2026-10-29; row O22 under *Milestone 5 — decisions recorded* in `plan/PROGRESS.md`) kept README's "Multiple providers" wording, while `plan/spec/kirsch-plan.md:34` lists "Multiple wire formats (OpenAI-format adapters)" as out of scope. Amendment 91 (A1) removes that phrase from §1 at M7. Before m6-d14, check that README's provider wording does not contradict the spec's §1 line.
- First missions, per `plan/build-plan-v0.2-v0.6.md`, *Order of first missions*: `m6-d1` and `m6-d2` start the day `v0.1.0` is tagged. `m6-d4`, `m6-d6`, `m6-d8` and `m6-d12` have no dependencies inside M6 and are good overnight-queue candidates while you test wave 1.
- m6-d12's "the five M1 debug commands are removed" is already satisfied: `TestDeletedDebugCommandsAreUnknown` (`internal/tui/view_smoke_test.go:1164`) asserts they are unknown. The test's list includes the M1 debug commands among other removed names. Re-derive Done-when against the code before briefing.
