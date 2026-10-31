# Milestone 7 — More than one wire (v0.3.0)

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

A second adapter speaks OpenAI Chat Completions, a local server is a first-class endpoint, Kirsch can prove it is offline, and it shows what caching and flat-rate quotas cost. After M7 a full session runs on LM Studio with no network.

## Waits on

`v0.2.0`; ADR 0011; O33–O35.

## Build order

`internal/provider/openaichat` → `internal/provider` (registry) → `internal/model` (discovery) → `internal/app` (offline) → `internal/doctor` → `internal/tui` → `cmd/kirsch` → `doc/`.

## Deliverables

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m7-d1 | Chat-format probe fixtures | — | `cmd/kirsch-probe` (`--api openai-chat`), `testdata/*.sse` for LM Studio, Ollama, one gateway | 1 | Recorded streams for text, tool call, reasoning content, usage-at-end, error; README says how to re-record |
| m7-d2 | `openai-chat` encode/decode | m7-d1 | `internal/provider/openaichat/{client,encode,decode}.go` + tests | 1 | Neutral `Request` ↔ chat request; partial tool-argument JSON salvaged only at `finish_reason`; usage from the final chunk; `reasoning_content` and `<think>` both decode to `BlockThinking` and are never echoed (O33) |
| m7-d3 | Adapter registry | m7-d2 | `internal/provider/registry.go` + tests; `api` config key | 1 | `api` selects the adapter; unknown value fails at load; built-ins keep `anthropic-messages` |
| m7-d4 | Local endpoint and discovery | m7-d3 | `internal/model/discover.go` + tests; `discover = true` | 2 | `kirsch models` and `/model` query `GET /v1/models` on demand, 2s timeout, cached per session; discovered rows are `Known: false`, pricing per O34 |
| m7-d5 | Compat flags | m7-d2, m6-d1 | `models.toml` `compat` table; both encoders | 2 | At least `no_between_tools`, `no_parallel_tools`, `max_tokens_field`, `strict_tools`; the Haiku `between_tools` 400 becomes a compat row, not a branch |
| m7-d6 | Offline mode | m7-d3 | `internal/app/offline.go` + tests; `--offline`; `[provider] offline`; doctor line; status-bar lock glyph | 2 | Non-loopback endpoints refused at load and at switch; `doctor` lists what is reachable; ASCII fallback per ui-spec |
| m7-d7 | Cache hygiene | — | `internal/agent/prompt/prompt.go` + test | 3 | Static block holds nothing per-day or per-repo; a test proves its bytes are identical across two repos |
| m7-d8 | Cache-waste line | m7-d7, m6-d1 | `internal/session/usage.go` + tests; `/status` | 3 | Cacheable-but-not-cached input tokens priced and summed; `/status` shows "cache waste $x.xx (n turns)"; flat-rate shows tokens |
| m7-d9 | Quota gauge | O35 | `internal/session/quota.go` + tests; `quota` config; status bar; `kirsch usage` | 3 | Per-endpoint consumption per rolling window persisted across sessions; `kirsch usage` prints it; a short gauge in the status bar when a quota is set |
| m7-d10 | Docs and ADR | all | `doc/providers.md` (LM Studio, Ollama, gateway, offline), `doc/configuration.md`, `doc/usage.md`, `plan/adr/0011-*.md`, `CHANGELOG.md` | 4 | Each recipe run once by the operator; S36 passes |
| m7-d11 | Acceptance and tag | all | repository | gate | Cable-out session passes; opencode live smoke still passes; tag `v0.3.0` |

## Acceptance (operator)

- [ ] LM Studio on the Mac Mini; `[provider.endpoints.local] api = "openai-chat" base_url = "http://127.0.0.1:1234/v1" discover = true`; `kirsch models` lists the served model.
- [ ] `kirsch --offline` with Wi-Fi off: read, patch, approve, `go test`, compact, resume; no message mentions the network.
- [ ] `kirsch --offline` with `main = anthropic`: refused at start with the reason.
- [ ] On `anthropic`, `/status` shows cache waste after three turns; an unchanged fourth turn does not grow it.
- [ ] OpenCode Go weekly cap configured; gauge appears; `kirsch usage` matches session totals.

## Not in this milestone

Responses format, Google format, OAuth, automatic routing.

## Operator decisions

| ID | Needed by | Question | Default |
| --- | --- | --- | --- |
| O33 | m7-d2 | Reasoning content from Chat-format servers | Decode to `BlockThinking`, show dimmed, never echo back |
| O34 | m7-d4 | Pricing kind for discovered models | `flat` on loopback, `unknown` remote |
| O35 | m7-d9 | Quota windows | Rolling 5-hour, 7-day, calendar month; tokens or requests |

## Documentation

| Milestone | Page(s) | Guard |
| --- | --- | --- |
| M7 | `doc/providers.md` (LM Studio, Ollama, gateway, offline) | recipes run by operator |

## Notes for whoever executes this

- ADR 0011 must be Accepted before Milestone 7 starts.
