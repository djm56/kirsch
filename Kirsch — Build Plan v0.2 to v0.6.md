# Kirsch — Build Plan v0.2 to v0.6

Oct 30, 2026 · @Donovan Maidens

This is the post-v0.1.0 build plan for Kirsch: five milestones that take it from one provider and one model to a daily tool that chooses models per role, runs headless under Bishop, speaks to editors and remote machines, and works overnight, without changing what Kirsch is. It is written to be added to `plan/` as-is and read by Bishop one mission at a time.

## How to apply this document

Save this page as `plan/build-plan-v0.2-v0.6.md`, then make the changes below in one commit before any M6 mission starts. Everything else in `plan/` keeps working as it does today: `PROGRESS.md` holds live status, each milestone file holds static definition, Bishop claims one deliverable per mission, you test the matching checklist section.

| Action | File |
| --- | --- |
| Add amendments A1–A7 to the Amendment Log | `plan/spec/kirsch-plan.md` §11 |
| Create ADRs 0009, 0010, 0011 from the amendment text (TEMPLATE.md) | `plan/adr/` |
| Split the five milestone sections of this page into one file each | `plan/milestones/milestone-6.md` … `milestone-10.md` |
| Create the M6 start-here file from the prompt at the end of this page | `plan/milestones/milestone-6-start-here.md` |
| Add the M6 deliverable rows with empty Status and Owner; add "Milestone 6 — decisions recorded" | `plan/PROGRESS.md` |
| Add rows 6–10 to the milestone table (Executable) | `plan/milestones/README.md` |
| Add a sentence pointing at this plan; keep everything else | `plan/README.md`, `AGENTS.md` |

**Ground rules.** Unchanged from `plan/README.md`: one milestone at a time, in order; nothing from a later milestone early; the plan is the spec and the UI spec is law for anything visual; a task that needs a later package stops and amends; amend, don't drift. Two additions: a deliverable that adds a config key adds its `doc/configuration.md` line in the same mission (S36 enforces it), and one that adds a flag or slash command adds its `doc/usage.md` row.

**Mission filter.** Before building, every mission checks that Kirsch still is: one loop, one session, one pane; patch-only editing; approval-first; local-first; tool output as data. A mission that finds it needs a second loop, a forked context, a suppressed approval, or a non-loopback call in offline mode stops and files an amendment instead of building it.

**Confidence.** All five sets are *Executable*: written once before building, amendment expected. Re-read each set against the code before starting it; M7–M10 against the M6 code in particular, because the catalog and role types they build on will have been reshaped by then.

## Plan amendments and new ADRs

Record these in `plan/spec/kirsch-plan.md` §11 before M6; they are the places where the locked v0.1 text contradicts what the next milestones build.

| # | Section | Change |
| --- | --- | --- |
| A1 | §1 Out of scope | Remove "Multiple wire formats (OpenAI-format adapters)" (M7), "`git commit` tool" (M8; `git push` stays out) and "session branching" (M10; search stays out). Add as permanent: sub-agents, any `spawn` command, in-process delegation. Orchestration belongs to the caller (Bishop). |
| A2 | §2 Architecture | New packages: `internal/model` (catalog, roles; M6), `internal/command` (M6), `internal/provider/openaichat` (M7), `internal/hook` (M8), `internal/executor` (M9), `internal/acp` (M9). Hard rules extended: `internal/agent` imports `internal/model` interfaces only; `internal/tui` never imports `internal/model` and receives role and model names as strings. `internal/arch/imports_test.go` is updated in the same mission as each package. |
| A3 | §5 Configuration | `[provider.endpoints.<name>]` gains `api` (`anthropic-messages` default, `openai-chat` from M7), `discover`, `quota`. New `[models]` (`main`, `compact`, `quick`, `fallback`), `[hooks]` (M8), `[workspace] executor` (M9), `[session] max_cost_usd`, `[provider] offline`, `[review] second_opinion`. Project config may set `[policy]`, `[models]`, `[hooks]` in addition to `[context].project_files`. Secrets stay env-only. |
| A4 | §7 Session events | New: `model_changed`, `agent_changed`, `fallback_used`, `checkpoint`, `undo`, `hook_ran`, `nudge`, `patch_proposed`, `patch_queued`, `patch_reviewed`, and a `fork` header. Each lands with the mission that first writes it and §7 is amended in the same commit. |
| A5 | ADR 0009 — Model catalog and roles | Supersedes ADR 0003's "one provider" and ADR 0008's two fixed endpoints. Decision: one neutral `Request`; model data lives in `models.toml` (embedded snapshot plus user file), not in Go; three roles and no fourth; fallback is an ordered list the user wrote, never a heuristic. |
| A6 | ADR 0010 — Wire-format adapters | One adapter package per `api` value, each translating to and from the neutral types; per-model `compat` flags replace code branches; thinking from Chat-format servers is decoded, shown, never echoed. |
| A7 | ADR 0011 — Execution seam | An `Executor` interface behind every filesystem, search, git and command call; containment is checked on the path string before any executor call; patches travel as text; local first, ssh second, container only if cheap. Contract table row "Providers" becomes "any Messages- or Chat-format endpoint; model chosen per role" at the M6 tag. |

## Decisions already taken

These were settled while writing the plan and are recorded here so no mission re-opens them.

- **Documentation lives in the repo.** Usage and reference in `doc/` (Diátaxis split: tutorial, how-to, reference, explanation), builder material in `plan/`. The GitHub wiki is a sidecar for a FAQ, a dev journal and notes only; it is a separate repository, cannot be tagged with a release, cannot change in the same PR as the code, cannot be covered by the coverage tests, and cannot be read by an agent with `read_file`. Browsable pages, if wanted later, are generated from `doc/`.
- **Extension surfaces, in this order:** Markdown slash commands (M6), agent profiles as config presets (M6), shell hooks on two events with no veto (M8), `SKILL.md` skills in the agentskills.io format (M10). No TypeScript or Lua extension language, no plugin marketplace, no MCP client; an MCP server that is really needed is wrapped as a CLI with a README and called through `run_command`.
- **Sub-agents are not Kirsch's job.** Kirsch stays one loop, one session, one pane, fully visible. It owes an orchestrator `kirsch run --json`, `--agent <profile>` and `--worktree`, nothing more. Bishop spawns and watches.
- **Model roles stop at three.** `main` drives turns, `compact` summarises, `quick` does sub-second jobs (second-opinion lines, later commit messages). No automatic "best model for this prompt" routing.
- **Wire formats in order:** Anthropic Messages (have), OpenAI Chat Completions (M7), OpenAI Responses only when a wanted model needs it. No Google format, no Bedrock or Vertex signing, no OAuth device-flow logins.
- **Isolation is policy plus container, not a home-grown sandbox.** `doc/sandboxing.md` documents devcontainer and Docker Sandbox use; protected paths become executor-enforced in M9; an opt-in `bwrap` / `sandbox-exec` wrapper is deferred until real use shows the approval model is not enough.
- **Permanently out of scope:** sub-agents and `spawn`, in-process delegation, session search, a web UI, Windows, a public Go SDK (`pkg/kirsch`), LSP, plan mode, built-in to-dos.

## Milestone 6 — More than one model (v0.2.0)

**Goal.** A session holds several configured endpoints, picks the model per role, switches models from the TUI, survives a transient provider failure, and refuses the three failure modes that waste quota overnight. Only the existing Messages-format adapter is used.

**Waits on.** `v0.1.0`; A1–A5 and A7 recorded; ADR 0009 accepted; O30–O32 answered.

**Build order.** `internal/model` → `internal/provider` (retry) → `internal/agent` → `internal/app` → `internal/tui` → `internal/config` → `cmd/kirsch` → `doc/`.

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
| m6-d14 | Docs and ADR | all | `doc/configuration.md`, `doc/usage.md`, `doc/providers.md` (new), `plan/adr/0009-*.md`, `README.md`, `CHANGELOG.md` | 5 | S36 passes; new slash-command coverage test passes; one CHANGELOG line per user-visible change |
| m6-d15 | Acceptance and tag | all | repository | gate | Checklist all-green; live smoke on two endpoints; CI green; tag `v0.2.0` |

**Acceptance (operator).**

- [ ] `anthropic` as `main`, `opencode` as `compact`; a task that compacts; `/status` shows both.
- [ ] `/model` to `opencode/minimax-m3` mid-session; quit; `kirsch resume` shows it.
- [ ] Invalid key on `main`, `fallback = ["opencode"]`; the turn completes on opencode and the card says so.
- [ ] Fake-provider loop fixture; third call shows the doom-loop card.
- [ ] Project policy allowlisting `go test ./...` runs silently; `go test ./... && rm -rf /tmp/x` prompts.
- [ ] `/test` from `.kirsch/commands/test.md` sends the expanded template.
- [ ] `kirsch --agent review` cannot propose a patch and says why.

**Not in M6.** OpenAI-format endpoints, discovery, offline mode, `kirsch run`, hooks, fork.

## Milestone 7 — More than one wire (v0.3.0)

**Goal.** A second adapter speaks OpenAI Chat Completions, a local server is a first-class endpoint, Kirsch can prove it is offline, and it shows what caching and flat-rate quotas cost. After M7 a full session runs on LM Studio with no network.

**Waits on.** `v0.2.0`; ADR 0010; O33–O35.

**Build order.** `internal/provider/openaichat` → `internal/provider` (registry) → `internal/model` (discovery) → `internal/app` (offline) → `internal/doctor` → `internal/tui` → `cmd/kirsch` → `doc/`.

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
| m7-d10 | Docs and ADR | all | `doc/providers.md` (LM Studio, Ollama, gateway, offline), `doc/configuration.md`, `doc/usage.md`, `plan/adr/0010-*.md`, `CHANGELOG.md` | 4 | Each recipe run once by the operator; S36 passes |
| m7-d11 | Acceptance and tag | all | repository | gate | Cable-out session passes; opencode live smoke still passes; tag `v0.3.0` |

**Acceptance (operator).**

- [ ] LM Studio on the Mac Mini; `[provider.endpoints.local] api = "openai-chat" base_url = "http://127.0.0.1:1234/v1" discover = true`; `kirsch models` lists the served model.
- [ ] `kirsch --offline` with Wi-Fi off: read, patch, approve, `go test`, compact, resume; no message mentions the network.
- [ ] `kirsch --offline` with `main = anthropic`: refused at start with the reason.
- [ ] On `anthropic`, `/status` shows cache waste after three turns; an unchanged fourth turn does not grow it.
- [ ] OpenCode Go weekly cap configured; gauge appears; `kirsch usage` matches session totals.

**Not in M7.** Responses format, Google format, OAuth, automatic routing.

## Milestone 8 — Headless (v0.4.0)

**Goal.** Kirsch can be driven by a program: `kirsch run` executes one task without a TUI, streams session events as JSON lines, applies a written approval policy, can run in shadow mode that applies nothing, can commit what it was allowed to change, and can undo. After M8 Bishop runs a task end-to-end through Kirsch and the reviewer receives the exported session.

**Waits on.** `v0.2.0` (M7 optional; if present, `--offline` works in `run`); O36–O38.

**Build order.** `internal/session` (export, checkpoint) → `internal/policy` (file) → `internal/hook` → `internal/tool` (`git_commit`) → `internal/app` (headless) → `cmd/kirsch` → `doc/`.

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m8-d1 | Headless driver | — | `internal/app/headless.go` + tests; `internal/arch` | 1 | `app` runs a session through a `Frontend` interface; the TUI implements it; a JSON frontend writes one event per line to stdout and reads nothing; import test proves no `tui` import |
| m8-d2 | `kirsch run` | m8-d1 | `cmd/kirsch/run.go` + tests | 1 | `kirsch run [--json] [--max-turns N] [--budget-usd X] [--agent P] [--model E/M] "task"` or `--task-file`/stdin; exit codes 0 done, 2 max-turns, 3 budget, 4 denied by policy, 5 provider, 130 interrupted; final report is the last text event |
| m8-d3 | Approval policy file | — | `internal/policy/file.go` + tests; `--approve path` | 1 | TOML answers every approval kind (`patches`, `commands` list split as m6-d11, `paths_deny`); anything unanswered is denied with exit 4; the TUI never reads it |
| m8-d4 | Shadow mode | m8-d2 | headless driver; `apply_patch` dry-run | 2 | `--shadow`: patches validated and recorded as `patch_proposed` with full diff, not applied; tree-modifying commands refused; report lists proposed patches |
| m8-d5 | Worktree | — | `internal/workspace/worktree.go` + tests; `--worktree [name]` | 2 | Creates or reuses `.kirsch/worktrees/<name>` via `git worktree add`; the root becomes the worktree; the instance lock is per worktree; recorded in the session |
| m8-d6 | Checkpoint and `/undo` | O38 | `internal/session/checkpoint.go` + tests; `apply_patch`; `/undo`; §7 | 2 | A checkpoint ref is recorded before each applied patch; `/undo` restores the last one and records `undo`; works after resume; refuses if the tree changed outside Kirsch |
| m8-d7 | Hooks, two events | O36 | `internal/hook/` + tests; `[hooks] post_patch`, `stop`; `hook_ran` event | 3 | Each hook runs with event JSON on stdin, 30s timeout, output captured and shown dimmed; non-zero exit reported, never a veto; same containment and env rules as `run_command` |
| m8-d8 | Verify-before-done nudge | — | `internal/agent/agent.go` + test; card | 3 | Text-only end after applied patches with no command since → a notice card offering the project test command; in `run` mode a `nudge` event only |
| m8-d9 | `git_commit` tool | O37 | `internal/tool/git_commit.go` + tests; `commit` approval kind; `system.md` | 3 | Commits only files touched by approved patches this session; model proposes the message, user approves; trailers `Kirsch-Session:` and `Kirsch-Model:`; never stages outside the touched set; never pushes |
| m8-d10 | Session export | — | `internal/session/export.go` + golden test; `kirsch session export --md`, `session list` | 3 | Markdown with task, models, files read, searches, patches (applied / rejected / proposed), commands with trimmed output, cost, final report |
| m8-d11 | Docs | all | `doc/headless.md` (new), `doc/configuration.md`, `doc/usage.md`, `CHANGELOG.md` | 4 | Event schema section generated from the Go types by a `scripts/` script and checked by a test; a 30-line Bishop driver example |
| m8-d12 | Acceptance and tag | all | repository | gate | Bishop walkthrough passes; tag `v0.4.0` |

**Acceptance (operator).**

- [ ] `kirsch run --json --approve .kirsch/policy.toml --worktree t1 "add a test for X"` from a Bishop task; the harness parses the stream; exit 0.
- [ ] Same with `--shadow`; tree unchanged; `kirsch session export --md` shows the proposed diff.
- [ ] TUI: apply a patch, `/undo`; file restored; `git status` clean.
- [ ] `post_patch` hook running `gofmt -l` shows output as a card.
- [ ] `git_commit` after two patches; `git log -1` shows both trailers; no stray staged files.

**Not in M8.** ACP, ssh or container executors, SDK, `pre_tool` veto hooks, fork.

## Milestone 9 — Front ends and machines (v0.5.0)

**Goal.** An editor drives Kirsch over the Agent Client Protocol, and the TUI on one machine works a repository on another through an execution seam. Approvals, the session log and containment behave identically in every configuration.

**Waits on.** `v0.4.0`; ADR 0011; O39–O40.

**Build order.** `internal/executor` (local) → `internal/workspace` + `internal/tool` (use it) → `internal/executor/ssh` → `internal/acp` → `cmd/kirsch` → `doc/`.

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m9-d1 | `Executor` interface, local | — | `internal/executor/{executor,local}.go` + tests; tools and workspace refactored to call it | 1 | Every read, list, `rg`, `git`, command and patch application goes through `Executor`; containment checked on the path string before the call; behaviour unchanged; fuzz suite still covers the untrusted surfaces |
| m9-d2 | Protected paths | m9-d1, O39 | `internal/policy`, `internal/executor/local.go` | 1 | Denylist paths and `.git/**` refused for writes by the executor itself, not only by the tool layer |
| m9-d3 | `ssh` executor | m9-d1, O40 | `internal/executor/ssh/` (commands via `ssh`, reads via `cat`/`ls`, search via remote `rg`, patches piped to remote `git apply --check` then `git apply`); `[workspace] executor = "ssh" host root` | 2 | M2 and M4 manual walkthroughs pass against a remote repo; session file, approvals and config stay local; missing remote `rg`/`git` is a doctor failure |
| m9-d4 | `kirsch acp` | m8-d1, O40 | `internal/acp/` on the chosen Go SDK + tests; `cmd/kirsch/acp.go` | 2 | `initialize`, `session/new`, `session/prompt`, `session/cancel`; text, thinking and tool calls stream as updates; each approval is an ACP permission request (allow-once / always / reject); `apply_patch` diffs sent as diff blocks; sessions are the same JSONL |
| m9-d5 | ACP conformance | m9-d4 | `internal/acp/conformance_test.go` with the SDK's example client | 3 | Scripted read-only task and approved-patch task pass; cancel mid-tool leaves no orphan process |
| m9-d6 | Docs and ADR | all | `doc/sandboxing.md` (devcontainer, Docker Sandbox, what approvals do not protect against), `doc/editors.md` (Zed, JetBrains), `doc/configuration.md`, `plan/adr/0011-*.md`, `CHANGELOG.md` | 3 | Each recipe run once by the operator |
| m9-d7 | Acceptance and tag | all | repository | gate | Zed and remote-repo walkthroughs pass; tag `v0.5.0` |

**Acceptance (operator).**

- [ ] Zed with `kirsch acp`: prompt, tool call, approve a patch from Zed's dialog, see the diff.
- [ ] TUI on the Mac, `executor = "ssh"` to the Ubuntu box: read, search, patch, `go test`, resume; the remote sessions directory stays empty.
- [ ] Patch `.env` on the remote: refused before any ssh call.

**Not in M9.** Container executor (`m9-d3b` only if ssh lands early), an OS sandbox of Kirsch's own, HTTP/WebSocket ACP, SDK package.

## Milestone 10 — Review and memory (v0.6.0)

**Goal.** Kirsch works while you are away and shows what it did: proposed patches queue in an inbox you review in the morning, a cheap model annotates each before you decide, a session can fork for a side quest, project knowledge lives in skills, and a replay suite proves behaviour without spending tokens.

**Waits on.** `v0.3.0` and `v0.4.0`; O41–O42.

**Build order.** `internal/session` (inbox, fork) → `internal/model` (opinion) → `internal/agent/prompt` (skills) → `internal/tui` (inbox mode) → `cmd/kirsch` → `scripts/` → `doc/`.

| ID | Title | Waits on | Owns | Wave | Done when |
| --- | --- | --- | --- | --- | --- |
| m10-d1 | Inbox store | m8-d4 | `internal/session/inbox.go` + tests; `patch_queued`, `patch_reviewed` events | 1 | `--shadow`/`--queue` runs write proposed patches with session id, base commit, diff; stale when base is no longer HEAD; JSONL under the state directory |
| m10-d2 | `kirsch review` | m10-d1 | `internal/tui/inbox.go` (new mode, ui-spec amendment, goldens); `cmd/kirsch/review.go` | 1 | Approve (normal checkpoint), reject, skip, edit-in-`$EDITOR`-then-approve; each decision recorded; stale items only reject or re-run |
| m10-d3 | Second-opinion line | m6-d2, m10-d1, O41 | `internal/model/opinion.go` + tests; `[review] second_opinion` (default false) | 2 | Before an approval card or inbox item, the `quick` role gives one line (≤ 120 chars) shown dimmed; timeout or error shows nothing and never blocks |
| m10-d4 | Session fork | O42 | `internal/session/fork.go` + tests; `/fork`, `/branches`; resume picks a branch; `fork` header | 2 | A child file names parent id and event index; pre-fork events read from the parent, never copied; compaction of a child never touches the parent |
| m10-d5 | Skills folder | — | `internal/agent/prompt/skills.go` + tests; `.kirsch/skills/*/SKILL.md`, `~/.config/kirsch/skills/`; `system.md` | 3 | Names and descriptions in a prompt section after the cache breakpoint; bodies read with `read_file` on demand; frontmatter errors are warnings; a skill cannot change policy |
| m10-d6 | `kirsch replay` | — | `internal/session/replay.go` + tests; fake provider plays recorded streams | 3 | Replays a session's provider events against real tools in a temp copy at the base commit with the session's policy; reports divergence in tool results or patches |
| m10-d7 | Nightly task set | m10-d6, m8-d2 | `scripts/nightly.sh`, `testdata/tasks/*.toml` (8–10 tasks on a Go repo and a WordPress plugin) | 4 | Runs each task with `kirsch run --shadow --json`, exports sessions, writes a summary table (pass, patches, cost, duration) Bishop can read; operator-triggered first, then cron on the Mac Mini |
| m10-d8 | Docs | all | `doc/review.md`, `doc/sessions.md`, `doc/skills.md`, `doc/usage.md`, `CHANGELOG.md` | 4 | The overnight recipe run once end-to-end by the operator |
| m10-d9 | Acceptance and tag | all | repository | gate | Overnight walkthrough passes; replay suite green in CI; tag `v0.6.0` |

**Acceptance (operator).**

- [ ] Three tasks queued overnight with `--shadow --queue`; `kirsch review` in the morning shows patches with opinion lines; approve two, reject one; `git status` matches.
- [ ] `/fork` mid-session, fix a test on the branch, return; `/branches` lists both.
- [ ] A `wordpress-plugin` skill appears in `/status`; its body shows in a tool card only after the model reads it.
- [ ] `kirsch replay` of yesterday's session reports no divergence; a deliberate tool change reports one.

**Not in M10.** Session search, inbox web UI, automatic approval of inbox items, MCP.

## Operator decisions, documentation plan and the M6 start-here

**Operator decisions.** Numbered after M5's O-series. Bishop asks each before the deliverable that names it; the answer is recorded in `PROGRESS.md` under the milestone's "decisions recorded". The default applies if you have not answered when the mission starts.

| ID | Needed by | Question | Default |
| --- | --- | --- | --- |
| O30 | m6-d1 | Catalog format | TOML embedded + user file; `kirsch models refresh` imports models.dev JSON |
| O31 | m6-d4 | Retry budget | 3 attempts, 1s / 4s / 16s, `Retry-After` wins, 60s ceiling |
| O32 | m6-d11 | Split on newlines and `$(...)` too? | Newlines yes; `$(`, backticks or `eval` never match the allowlist |
| O33 | m7-d2 | Reasoning content from Chat-format servers | Decode to `BlockThinking`, show dimmed, never echo back |
| O34 | m7-d4 | Pricing kind for discovered models | `flat` on loopback, `unknown` remote |
| O35 | m7-d9 | Quota windows | Rolling 5-hour, 7-day, calendar month; tokens or requests |
| O36 | m8-d7 | Hook events in v0.4 | `post_patch` and `stop` only |
| O37 | m8-d9 | Ship `git_commit` in v0.4? | Yes, approval-gated; policies need explicit `commit = "allow"` |
| O38 | m8-d6 | Checkpoint mechanism | `git stash create` object + event; file copies when not a git tree |
| O39 | m9-d2 | Read-only protected paths mechanism | Policy-only on local; executors with mounts enforce |
| O40 | m9-d3, m9-d4 | ACP SDK; ssh transport | `coder/acp-go-sdk`; system `ssh` with `ControlMaster` |
| O41 | m10-d3 | Second-opinion prompt ownership | Embedded `internal/model/opinion.md`, like `system.md` |
| O42 | m10-d4 | Fork storage | Separate file per branch, parent read-through |

**Documentation plan.** One page per milestone, written in the milestone, guarded by a coverage test where one exists.

| Milestone | Page(s) | Guard |
| --- | --- | --- |
| M5 (now) | `README.md` front door reordered: recording, install, what is different, link to tutorial; `doc/getting-started.md` tutorial | — |
| M6 | `doc/providers.md` (Messages endpoints, roles), `doc/configuration.md`, `doc/usage.md`, `doc/how-it-works.md` (from `architecture.md` and the ADRs) | S36; slash-command coverage test |
| M7 | `doc/providers.md` (LM Studio, Ollama, gateway, offline) | recipes run by operator |
| M8 | `doc/headless.md`, `doc/sessions.md` | event schema generated and tested |
| M9 | `doc/sandboxing.md`, `doc/editors.md`, `doc/troubleshooting.md` (doctor lines) | doctor-line coverage test |
| M10 | `doc/review.md`, `doc/skills.md`; `doc/llms.txt` generated by `scripts/` | — |

**Order of first missions.** `m6-d1` and `m6-d2` start the day `v0.1.0` is tagged. `m6-d4`, `m6-d6`, `m6-d8` and `m6-d12` have no dependencies inside M6 and are good overnight-queue candidates while you test wave 1.

**`plan/milestones/milestone-6-start-here.md` — paste this in Claude Code to begin:**

```markdown
# Milestone 6 — start here

You are Bishop working on Kirsch. Read, in order: plan/README.md, plan/PROGRESS.md,
plan/build-plan-v0.2-v0.6.md (amendments and the Milestone 6 section),
plan/adr/0009-model-catalog-and-roles.md, then the code under internal/provider,
internal/config and internal/app.

Mission: m6-d1 — Model catalog in data.

Before writing code: confirm amendments A1–A5 and A7 are in plan/spec/kirsch-plan.md §11
and that O30 is recorded in PROGRESS.md (default: TOML). If either is missing, stop and
ask the operator.

Run the mission filter: the change must keep Kirsch one loop, one session, one pane,
patch-only, approval-first, local-first, tool output as data. If it cannot, stop and
file an amendment instead.

Deliver exactly what the m6-d1 row says under Owns and Done when. Add every new config
key to doc/configuration.md in this mission. Update internal/arch/imports_test.go if
a new package is introduced. Run npm run check and npm run security. Record the
mission in PROGRESS.md (Status: done, mission id, commit) and list carried items under
"Carried from m6-d1". Do not start m6-d2.
```
