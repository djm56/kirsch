# Milestone 5 — Instruction Set

> **Status: Detailed instruction set — written 2026-10-23 (mission-20261023-06).** Built from spec §8 (`plan/spec/kirsch-plan.md:472–492`), the open carried items in `plan/PROGRESS.md` (`:63–262`), and the mission-20261023-05 DEBRIEF (*Tracker And Reality*, `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:35–39`). Every code claim carries a `file:line` citation and was read against the code on 2026-10-23. Each citation is re-checked before its brief is written (see *Note for whoever executes this*). Every deliverable is written so the operator can lift it into a `/mission` brief. Live status and owners belong in `plan/PROGRESS.md`, not here. New to Milestone 5? Start with [`milestone-5-start-here.md`](milestone-5-start-here.md): where Milestone 5 is, what to do next, the prompt to paste, and which piece can start.

## Ground Rules (read first)

1. **Milestone 4 is complete.** Operator-accepted 2026-10-23, tag `m4`. The narrow-width status-bar fallback that the walkthrough marked ❌ was the first deliverable here (m5-d1). O1 resolved it on 2026-10-27 as a wrong expectation, not a code defect.
2. **Each deliverable is sized for one small junior mission** (a sequence of steps, each owned by one agent; one `@hicks` code step plus one `@apone` review, at most one fix round, as `plan/milestones/milestone-4.md` words it). This is a sizing target, not a cap on rounds. The two-round contract in `.claude/CREW-MANIFEST.md` (Delegation Constraints) and `.claude/skills/mission-lifecycle/SKILL.md` still governs every fix loop. Every code or docs step is followed immediately by an `@apone` review step, and that review covers exactly one step. A deliverable that needs more than one code step is split when its brief is written, not during execution.
3. **Every code step goes to `@hicks`.** `@vasquez` is not in any step. It arrives only by escalation under the delegation contract (CREW-MANIFEST.md, Delegation Constraints).
4. **Operator-owned steps are marked `(operator)`.** They are decisions, real-terminal checks, repository settings, token-spending runs, or the tag. Bishop does not delegate them. Their acceptance boxes are ticked by the operator.
5. **Decisions come first.** A deliverable whose *Depends on* cell names an open O-decision cannot start until that decision is recorded (see *Decisions the operator makes before starting*).
6. **Live state goes in `plan/PROGRESS.md`.** This document carries no status and no owner. Acceptance boxes here are unchecked seed text for a mission brief. Their ticks, if any, live with the mission's evidence.
7. **Locked decisions (do not re-litigate).** Release tooling is goreleaser (`plan/spec/kirsch-plan.md:481`). The targets are `darwin/amd64`, `darwin/arm64`, `linux/amd64` and `linux/arm64` (`:481`). The version string injected at build time is bare (`0.1.0`), not `v0.1.0`, because the TUI already writes the `v` (`internal/tui/empty.go:81`).
8. **Re-read before you start.** Each deliverable must be checked against the code before its brief is written. See *Note for whoever executes this*.
9. **Mission steps name the agent.** Each deliverable lists its steps with the agent for each. A step that changes a file is followed by its `@apone` review as the next step.

---

## Goal and acceptance

The spec's Milestone 5 section, quoted exactly (`plan/spec/kirsch-plan.md`):

> **Goal:** Daily-driver quality; installable by others.  *(`:474`)*

> **Acceptance:** `bash scripts/smoke-test.sh` passes all milestone-4 checks plus: fresh machine flow (no key → clear onboarding), `kirsch doctor` output correct, release binaries built for all four targets, and `doc/` contains the four user-facing documents with every config key in §5 documented.  *(`:487`)*

**Where the acceptance and the code, or the acceptance and itself, disagree:**

- **No-key flow versus the smoke test.** `:487` asks for a fresh-machine no-key flow inside the smoke-test acceptance. `scripts/smoke-test.sh:16–22` exits 2 without a key and runs its live test (`:28`). The no-key flow cannot live in that script as written. It is m5-d16, and its placement is O14.
- **"`kirsch doctor` output correct" has no format to check against.** The spec fixes no output format and no exit codes (`plan/spec/kirsch-plan.md:479, :487`). The format is O5.
- **Doctor must run before the existing aborts.** `cmd/kirsch/main.go:62–65` (no repository) and `:71–81` (bad config) return before anything could be reported. Doctor, `--version` and `--help` must be dispatched ahead of both.
- **`kirsch --help`.** The spec lists it (`:480`). Today the flag set prints flag defaults and exits 1 (`main.go:31–36, :58–60`). That path was read from the flag package's behaviour and not run, so the exit status is *unverified* until m5-d2 runs it.
- **Release binaries for every target.** `:487` requires binaries for every target in the list above. Nothing cross-builds today. Compilation is *plausible but unverified*: there is no cgo in the tree, and the POSIX calls exist on both OSes (`internal/tool/run_command.go:183, :353–369`; `internal/session/lock.go:50`). m5-d10b must prove it.
- **`doc/` and every §5 key.** `:487` requires every config key documented. The flags layer of §5 precedence is not wired (`internal/config/config.go:496–506`, called from `cmd/kirsch/main.go:71–75` with no flags). The document must not promise config flags. See the backlog (X6).
- **Smoke test on the release build.** The milestone-4 checks are re-run at release (S34). The live run spends tokens and is operator-owned.

---

## Golden mechanism and regeneration procedure

This section is the one reference for goldens. Every step that changes a golden (m5-d1, m5-d7a, m5-d7b, m5-d21b) and every decision that names goldens (O21, O23) points here. It was verified against the code on 2026-10-23.

- **What a golden is.** A golden is a sized fence in `plan/spec/kirsch-ui-screens.md`, opened by a line of the form ```` ```text W×H ````. The document holds 14 sized fences. Eleven are 80 columns or wider. Three are narrower than 80: the fences headed `40×11`, `38×6` and `40×6`.
- **What compares them.** `internal/tui/golden_test.go` reads that document (`screensDoc`, `golden_test.go:22`), parses each sized fence, and compares `View()` against it. No golden file exists under `internal/tui`, and there is no `testdata` directory.
- **What regeneration writes.** `TestGenerateGrids` runs only with `KIRSCH_REGEN=1` (the gate is `regen_test.go:13`). It writes the current render of every scenario to `/tmp/kirsch-grids.txt` (`regen_test.go:27`). It does not change the screens document. A person reconciles the grids by hand.

**Procedure for any step that changes a golden:**

1. Record the list of sized fences the step will move, by fence header, before any regeneration.
2. Save a verbatim copy of `plan/spec/kirsch-ui-screens.md` to `.claude/memory/workspace/` before the edit. Acceptance boxes compare against that copy.
3. Run `KIRSCH_REGEN=1 go test ./internal/tui -run TestGenerateGrids`.
4. Reconcile each affected grid in `plan/spec/kirsch-ui-screens.md` by hand, from `/tmp/kirsch-grids.txt`. A grid that differs from the render is either a renderer defect or a stale reference. The brief says which, and a human reviews the diff.
5. Run `go test ./internal/tui`, then `npm run screens`, which lints the screens document (`package.json:35`).

**Ownership.** Four deliverables own `plan/spec/kirsch-ui-screens.md`: m5-d1 (wave 1), m5-d7a (wave 3), m5-d7b (wave 4) and m5-d21b (wave 7). Each sits in its own wave, so no two of them run together. m5-d7a follows m5-d1, and m5-d7b follows m5-d7a, on the same grids.

## Decisions the operator makes before starting

A deliverable named in a *Blocks* cell cannot start until its decision is recorded in `plan/PROGRESS.md`. The *Recommendation* column is filled only where a recommendation has been made. The choices from O17 onward are ones the step's brief must record before the step runs. They were not in the operator's original list, and they are numbered here so none is decided silently.

| ID | Decision | Options | Recommendation | Blocks | Sources (file:line) |
|----|----------|---------|----------------|--------|---------------------|
| O1 | What the narrow status bar shows, and at which widths. | Keep ui-spec §2.2:124 (no tokens below 80) and correct the M4 acceptance text; or keep an `N tok` rung at 60–79 and below; or `N tok` only below some width. The walkthrough step (`plan/testing/manual/user-testing/milestone-4-donovan.md:1042`) says both "shows only `N tok`" and "dropping tokens first". | — | m5-d1 | `plan/PROGRESS.md:67`; `plan/spec/ui-spec-v0.1.md:124, :170–173`; `internal/tui/statusbar.go:44–48`; `internal/tui/turn_messages_test.go:619–627` |
| O2 | goreleaser shape: archive format and names; checksums; changelog source (git log, or `CHANGELOG.md` `[Unreleased]`); Homebrew tap or none; SBOM; signing (cosign or GPG) or none. The spec fixes only goreleaser, the targets listed at `:481`, and a GitHub workflow. | — | — | m5-d10b | `plan/spec/kirsch-plan.md:481` |
| O3 | Release workflow: trigger (`v*` tag; manual dispatch for dry runs); `permissions: contents: write`; goreleaser version and action pinning (CI pins by tag; the golangci-lint pin lives in `scripts/golangci-lint-version`); whether CI runs a snapshot build on pull requests. `GITHUB_TOKEN` covers release assets; extra secrets are needed only for signing or a tap. Repository Actions settings must allow write. | — | — | m5-d11 | `.github/workflows/ci.yml:9–10, :20, :25`; `plan/spec/kirsch-plan.md:481` |
| O4 | Version string: bare `0.1.0` in ldflags (the TUI adds `v`); whether `--version` also prints commit and date; whether a `debug.ReadBuildInfo` fallback gives `go install` builds a real version. | — | — | m5-d2, m5-d10a, m5-d10b, m5-d13c | `cmd/kirsch/main.go:27–28`; `internal/tui/empty.go:81`; `internal/tui/modal.go:308` |
| O5 | Doctor design: package placement (`internal/app`, or new `internal/doctor` with an `plan/spec/architecture.md:31–49` line and an `internal/arch` entry); output grammar; which failures are fail and which are warn (a missing `rg` has a fallback); exit codes; whether doctor may create directories or take the lock (`internal/session/lock.go:34, :42`) or must probe read-only; no network calls (the spec is silent). | — | — | m5-d3a, m5-d3b, m5-d4a, m5-d4b | `plan/spec/kirsch-plan.md:479, :487`; `internal/session/lock.go:29–60` |
| O6 | Who writes `doc/`. The spec asks for user documentation. The crew table gives READMEs and changelogs to `@lambert`, and every coding step to `@hicks`. | — | `@lambert` writes, `@apone` reviews the claims against the code. | m5-d13a, m5-d13c, m5-d14a, m5-d14b | `.claude/CREW-MANIFEST.md` (crew table, Delegation Constraints) |
| O7 | `SECURITY.md` reporting channel (GitHub private advisories or an email), supported versions, response promise. | — | — | m5-d12 | `plan/spec/kirsch-plan.md:482` |
| O8 | `NOTICE`: only Bubble Tea, Lip Gloss and Bubbles, as the spec says, or every direct dependency. | — | — | m5-d12 | `plan/spec/kirsch-plan.md:482`; `go.mod:6–15` |
| O9 | Paste debounce: implement the 20 ms coalescing, or amend ui-spec §7.2 and §11 because bracketed paste already gives a single message. | — | — | m5-d8 | `internal/tui/styles.go:22`; `github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118` |
| O10 | Composer past five rows: scroll to the caret, or wrap. ui-spec §2:72 says scroll. | — | — | m5-d6, m5-d14a | `internal/tui/composer.go:100–106, :199–205`; `internal/tui/view.go:461` |
| O11 | D3: name of the new session event (for example `approval.scopes_cleared`) and the §7 amendment. Whether `/new` also records one (it starts a new file, so no). | — | — | m5-d5 | `plan/adr/0006-session-scoped-approvals.md:45, :67–69`; `plan/spec/kirsch-plan.md:340–357` |
| O12 | Demo assets: asciinema or screenshots; stored in the repository or linked. | — | — | m5-d22 | `plan/spec/kirsch-plan.md:484` |
| O13 | Whether to schedule the optional groups (m5-d17, m5-d18) and the safety-test group (m5-d19) into v0.1, or leave them in the backlog. | — | Schedule m5-d19: C30 names tests that cannot fail (`run_command_test.go`); the operator may defer it, but the gap stays open. | m5-d17a, m5-d17b, m5-d18, m5-d19a, m5-d19b | `plan/PROGRESS.md:149`; `internal/tool/run_command_test.go` |
| O14 | Fresh-machine acceptance: does "no key gives onboarding" include "not in a repo" (stderr text today), and where it lives, given `smoke-test.sh` requires a key. | — | — | m5-d16 | `plan/spec/kirsch-plan.md:487`; `plan/spec/ui-spec-v0.1.md:597`; `scripts/smoke-test.sh:16–22` |
| O15 | Operator-only carried decisions: C24 (clip the 400 debug log at a byte bound the operator records with the choice, or disclose), C34 (stdin box wording), C40 (ULID dependency), C43 (§7 flattening), C46 (`session.started`/`session.resumed`), C29 (the 500 ms box: tighten or relax). | — | — | m5-d19b (C29); m5-d20 (record) | `plan/PROGRESS.md:138, :148, :153, :170, :186, :194` |
| O16 | The help footer's `docs: doc/usage.md` for installed binaries (X8). | — | — | m5-d20 (decision); m5-d21b (follow-through) | `internal/tui/modal.go:308` |
| O17 | Composer rows cut at the frame edge: show a marker, or end the row in `⋯`. | Marker `⋯` at the cut; or wrap the row and show no marker. | — | m5-d7b | `plan/spec/ui-spec-v0.1.md:565–571`; `internal/tui/view.go:602–613` |
| O18 | Empty `NO_COLOR`: the code treats an empty value as set; the no-color.org convention treats it as unset. Record which behaviour is pinned. | Keep the code (empty is set, colour off); or treat empty as unset (colour on, if the other conditions allow). | — | m5-d9 | `internal/tui/styles.go:44–60` (`LookupEnv` at `:48`); no-color.org convention (external) |
| O19 | No-key check method: the flow is interactive, so the script must drive a pseudo-terminal or an equivalent. | A pseudo-terminal driver the script starts (for example `script(1)`, or a small pty helper in the repository's language). A manual run is not an option: the check runs inside the temporary repository that the script creates (m5-d16, instruction 3). | — | m5-d16 | `scripts/smoke-test.sh:16–22`; `internal/app/app.go:1298–1324` |
| O20 | m5-d10b CI-check method: a snapshot build of every target through goreleaser, or a `GOOS=… GOARCH=… go build` loop over the target list. | `goreleaser build --snapshot` (needs goreleaser on CI); or a plain `go build` loop (no extra tool). | — | m5-d10b | `plan/spec/kirsch-plan.md:481`; `.github/workflows/ci.yml` |
| O21 | C37 flag versus doc: the regeneration mechanism is `KIRSCH_REGEN=1`, not `-update`. Correct the spec, or implement `-update`. | Correct the two sentences that name `-update` (`plan/spec/kirsch-plan.md:498` and `plan/spec/ui-spec-v0.1.md:824`) so that they describe *Golden mechanism and regeneration procedure* (no product change); or add a `-update` flag. | — | m5-d12 | `plan/spec/kirsch-plan.md:498`; `plan/spec/ui-spec-v0.1.md:824`; `internal/tui/regen_test.go:13` |
| O22 | The "multiple providers" wording: README says "Multiple providers"; the spec says "Multiple wire formats (OpenAI-format adapters)". ADR 0008 and the spec describe the configured endpoints (`internal/config/config.go:91, :96`). | Keep the README wording and state the configured endpoints; or adopt the spec's wording. | — | m5-d15 | `README.md:47, :53–58`; `plan/spec/kirsch-plan.md:17, :32–34`; `plan/adr/0008-provider-endpoints.md` |
| O23 | Golden regeneration handling for changed screens: regenerate, then a human reviews the golden diff; or list each moved golden first and stop for approval. | Reconcile the moved fences in `plan/spec/kirsch-ui-screens.md` by the procedure in *Golden mechanism and regeneration procedure*, then a human reviews the diff; or list-then-approve before reconciling. Either way, fences 80 columns and wider do not move unless their step says so. | — | m5-d1, m5-d7a, m5-d7b, m5-d21b | `plan/spec/kirsch-ui-screens.md`; `internal/tui/golden_test.go:22`; `internal/tui/regen_test.go:13` |

The choices from O17 onward were not numbered in the operator's original list. Each step's brief quotes the recorded choice.

---

## Deliverables

Each row is a seed for one mission. The *Wave* column gives the wave from *Parallel waves* below. The mission steps for each deliverable are in its section.

| ID | Deliverable | Depends on | File ownership | Wave | Acceptance summary |
|----|-------------|------------|----------------|------|--------------------|
| m5-d1 | Narrow-width status-bar fallback | O1, O23 | `internal/tui/view.go`, `statusbar.go`, `turn_messages_test.go`; `plan/spec/kirsch-ui-screens.md` (sized fences below 80 columns only); ui-spec §2.2–§2.3 (docs step) | 1 | Narrow bar shows what O1 records; pinned test matches; ui-spec agrees; goldens at 80 columns and wider do not move |
| m5-d2 | `--version` and `--help` | O4 | `cmd/kirsch/main.go`, `cmd/kirsch/main_test.go` | 1 | `--version` works outside a repository; `--help` lists commands and flags and exits 0 |
| m5-d3a | Doctor: dispatch, report grammar, version line, exit codes, arch entry | m5-d2, O5 | `cmd/kirsch/main.go`, `cmd/kirsch/main_test.go`, `internal/doctor/` (new), `internal/arch/imports_test.go`, `plan/spec/architecture.md` (docs step) | 2 | `kirsch doctor` runs ahead of repository and config checks; help names `doctor`; version line and exit status correct |
| m5-d3b | Doctor: `rg`, `git`, repository and config probes | m5-d3a, O5 | `internal/doctor/` | 3 | Missing `rg` and missing `git` are reported under a PATH shim; repository and config lines correct |
| m5-d4a | Doctor: key, backend, model and project-context lines; comment edits | m5-d3b, O5 | `internal/doctor/`, `internal/tool/searchcode.go` (comment only), `internal/app/app.go` (comment only at `:520–521`) | 4 | Key line names the winner and never prints a value; backend, model and context lines correct; skipped line when config did not load |
| m5-d4b | Doctor: state directory and lock state | m5-d4a, O5 | `internal/doctor/`, `internal/session/` (read-only probe helper, only if O5 requires one) | 5 | Non-writable state directory flagged, mode restored; lock reads held and free; live instance not disturbed |
| m5-d5 | Persist `/approvals` clear | O11 | `internal/session/event.go`, `store.go`, `grants.go`, `internal/app/app.go`, `plan/spec/kirsch-plan.md` §7 (docs step) | 1 | A cleared grant stays cleared after `kirsch resume` |
| m5-d6 | Composer paste path and caret rows | O10; m5-d1 (file ownership: `view.go`, if O10 is wrap) | `internal/tui/composer.go`, `update.go`, `composer_test.go`, `paste_path_test.go` (new); `view.go` only if O10 is wrap | 2 | Ordinary paste lands verbatim; large-paste prompt tested; composer rows follow the caret |
| m5-d7a | Soft-wrap fidelity: `wrap()` | m5-d1, m5-d6, O23 | `internal/tui/view.go`, `cards.go`, `onboarding.go` (only if its output moves), `wrap_test.go` (new), `plan/spec/kirsch-ui-screens.md` (the fences that move) | 3 | Indentation and interior spacing kept; `wrap` has direct tests |
| m5-d7b | Soft-wrap fidelity: composer cut marker | m5-d6, m5-d7a, O17, O23 | `internal/tui/composer.go`, `view.go` (`exactly`, `:602–613`), `plan/spec/kirsch-ui-screens.md` (the fences that move) | 4 | No composer row cut without the marker O17 records |
| m5-d8 | Paste debounce | m5-d6, O9 | `internal/tui/update.go`, `model.go`, `styles.go`; ui-spec §7.2 and §11 (amend branch, docs step) | 3 | Debounce implemented, or constant removed with the spec amended |
| m5-d9 | Terminal capability and resize tests | O18 | `internal/tui/styles_test.go` (new), `view_smoke_test.go` (optional); `styles.go` only if O18 changes code | 1 | `DetectCaps` covered, positive control included; resize invariants with a modal open (optional) |
| m5-d10a | Build wrapper and version proof | m5-d2, O4 | `scripts/build.sh` (new), `package.json` | 2 | `build.sh` reports the injected version with no leading `v` |
| m5-d10b | goreleaser config and target-compile check | m5-d2, O2, O4, O20 | `.goreleaser.yaml` (new), `scripts/ci.sh`, `.github/workflows/ci.yml` | 2 | Config builds only `./cmd/kirsch`; every target in the list compiles; CI check fails on a forced failure |
| m5-d11 | GitHub release workflow | m5-d10b, O3 | `.github/workflows/release.yml` (new) | 3 | Workflow triggers on `v*`; first live run is operator-owned |
| m5-d12 | Repository files and golden-regeneration note | m5-d10a, m5-d10b, O7, O8, O21; m5-d8 (file ownership: `ui-spec-v0.1.md`, amend branch) | `CONTRIBUTING.md`, `SECURITY.md`, `NOTICE` (new); `plan/spec/kirsch-plan.md:498`; `plan/spec/ui-spec-v0.1.md:824` | 5 | Each file exists; the regeneration note matches the code |
| m5-d13a | `doc/configuration.md` | O6 | `doc/configuration.md` (new) | 1 | Every §5 key documented with its default and valid values; no config flags promised |
| m5-d13b | Key-coverage test (S36) | m5-d13a | `internal/config/key_coverage_test.go` (new) | 3 | The check passes on the page and fails when a key is removed from it |
| m5-d13c | `doc/install.md` | m5-d2, m5-d10a, m5-d10b, O4, O6 | `doc/install.md` (new) | 3 | Install page states what a `go install` build reports, as O4 records it |
| m5-d14a | `doc/usage.md` corrections | m5-d1, m5-d5, m5-d6, m5-d7a, O6, O10 | `doc/usage.md` | 4 | Usage claims match the code; debug-commands section gone; no milestone or ADR language |
| m5-d14b | `doc/troubleshooting.md` and `doc/README.md` | m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a, m5-d13c, O6 | `doc/troubleshooting.md` (new), `doc/README.md` | 6 | Troubleshooting explains every doctor line; index status lines match the files |
| m5-d15 | Front-door and tracker freshness | m5-d1, m5-d10a, m5-d10b, m5-d13c, O22 | `README.md`, `CHANGELOG.md`, `plan/PROGRESS.md`, `plan/spec/kirsch-plan.md:391`, `plan/adr/README.md`, `plan/README.md:25`, `AGENTS.md:21` | 4 | README install section present; ADR range and script claim fixed; closed rows marked |
| m5-d16 | Fresh-machine no-key check | O14, O19; m5-d10a (builds the binary the script runs); m5-d2 and m5-d3a (file ownership: `cmd/kirsch/main.go`, only if O14 makes "not in a repo" a screen) | new script in `scripts/` (name in the brief); `cmd/kirsch/main.go` only if O14 makes "not in a repo" a screen | 3 | A no-key run shows onboarding naming the active endpoint's key pair |
| m5-d17a (optional) | `/new` lifecycle: C51, C52 | m5-d5, O13 | `internal/app/app.go` | 3 | No orphan empty session file on quit during `/new`; provider session id rotates on `/new` |
| m5-d17b (optional) | `/new` busy state and turn fencing: C48, C49 | m5-d8, m5-d17a, O13 | `internal/tui/update.go`; `internal/app/app.go` (generation counter, after m5-d17a) | 5 | Enter during `/new` shows no stray card; late messages from a cancelled turn dropped |
| m5-d18 (optional) | Resume totals and compaction % | m5-d17a (file ownership only), O13 | `internal/app/app.go`, `internal/session/` (new loader file) | 6 | Totals shown after resume; compaction % updates after `/compact` |
| m5-d19a | Safety-test repair for `run_command`: C30, C35, C36 | — (O13) | `internal/tool/run_command_test.go`; `internal/tui/approval_unspecified_test.go` (new) | 1 | Named tests fail under their mutants; lower-case names pinned; blank-card content asserted |
| m5-d19b | Safety-test repair: C29 | m5-d19a, O15, O13 | `internal/tool/run_command.go`, `run_command_test.go`, `plan/milestones/milestone-2.md` | 2 | The 500 ms box holds, or is relaxed, as O15 records |
| m5-d20 (operator) | Record the operator's decisions | None; records O12, O15, O16 and the decision list in its section | `plan/PROGRESS.md` only | gate | Each listed ID has a recorded decision |
| m5-d21a (conditional) | Model table (X5, C9) | m5-d20 | `internal/provider/models.go`, its test | 7 | Table matches the recorded decision; review recorded |
| m5-d21b (conditional) | Help footer and goldens (X8, O16) | m5-d20, O16, O23 | `internal/tui/modal.go` (footer); `plan/spec/kirsch-ui-screens.md` (the fences that move) | 7 | Footer matches O16; moved goldens listed; review of the golden diff |
| m5-d21c (conditional) | Debug log clip (C24) | m5-d20 | `internal/provider/anthropic/client.go` (`:453` call site; `logBadRequest`, `:506–512`), `client_test.go`, `SECURITY.md` (the 400 sentence only) | 7 | 400 log writes at most the bound O15 records; its test pairs absence with presence |
| m5-d21d (conditional) | Debug log disclosure (C24) | m5-d20, m5-d12 (file ownership: `SECURITY.md`) | `SECURITY.md` | 7 | `SECURITY.md` states the 400 log behaviour, traced to code |
| m5-d21e (conditional) | ULID dependency (C40) | m5-d20 | `internal/session/layout.go:96–140`; `go.mod`, `go.sum` | 7 | Stdlib ULID replaced, or no step if the operator keeps it |
| m5-d21f (conditional) | §7 amendment (C43, C46) | m5-d20, m5-d5 (file ownership: `plan/spec/kirsch-plan.md` §7) | `plan/spec/kirsch-plan.md` §7 | 7 | §7 lists the session events the code writes |
| m5-d21g (conditional) | Lock behaviour (C8) | m5-d20 | `internal/session/usage.go:28, :152`; `usage_test.go` | 7 | Lock behaviour changed, or no step if the operator accepts it |
| m5-d22 (operator) | Demo, release smoke run, tag | m5-d21 deliverables that ran, O12, O15, O16 | repository, GitHub settings | 8 | Demo linked; smoke test PASS on the release build; tag `v0.1.0` |
| backlog | Recorded, not scheduled for v0.1 | — | — | — | See *Backlog* |

### Grouping notes for the operator

Research grouping is followed except where it does not sit cleanly. The notes below give the reason for each split.

1. **m5-d2 and the help text.** The help text names `doctor` only in m5-d3a, which adds the line and its test. m5-d2 must not name `doctor`.
2. **C23 is in the backlog, but its documentation remedy is in m5-d14b.** Only the code change stays in the backlog, as its own line.
3. **m5-d13 is split into m5-d13a, m5-d13b and m5-d13c.** The docs (`doc/configuration.md`, `doc/install.md`) are `@lambert` steps, and the key-coverage test (S36) is a `@hicks` step. O4 sits on m5-d13c, which needs it.
4. **Docs edits inside code deliverables are their own steps.** m5-d1 (ui-spec §2.2–§2.3), m5-d3a (`architecture.md`), m5-d5 (§7), and m5-d8 (ui-spec on the amend branch) each carry a separate `@lambert` step in their mission steps, with `@apone` immediately after.
5. **m5-d4 is split into m5-d4a and m5-d4b.** m5-d4a takes the key, backend, model and context lines and the comment edits. m5-d4b takes the state-directory probe and the lock state, which have side effects that O5 must decide first.
6. **m5-d16 does not depend on m5-d4.** `CheckOnboarding` (`internal/app/app.go:1298–1324`) does not use doctor, so the dependency is dropped.
7. **m5-d17 is split into m5-d17a and m5-d17b.** m5-d17a (app lifecycle) does not depend on m5-d8. m5-d17b (turn race and TUI) owns `update.go` and depends on m5-d8 for that file.
8. **m5-d19 is split into m5-d19a and m5-d19b.** m5-d19a holds test-only work. It depends on O13 only because O13 can defer the whole safety group. m5-d19b holds C29, which needs O15.
9. **m5-d20 is split into an operator gate, the m5-d21 deliverables and m5-d22.** m5-d20 records the operator's decisions. The m5-d21 deliverables carry the agent follow-through for each decision, one deliverable per outcome. m5-d22 holds the operator's demo, release run and tag.
10. **m5-d3 is split into m5-d3a and m5-d3b.** m5-d3a takes the dispatch, the report grammar, the exit codes and the help line, and it shares `main.go` with m5-d2. m5-d3b takes the probes that read the environment, the repository and the config. Those probes need the report grammar from m5-d3a, so they sit one wave later.
11. **m5-d7 is split into m5-d7a and m5-d7b.** m5-d7a changes `wrap()`, which every card and the onboarding text pass through, and it moves the most fences. m5-d7b changes the composer cut in `exactly`, which needs O17 and the composer work from m5-d6. Splitting them keeps the wide-moving change in its own review.
12. **m5-d10 is split into m5-d10a and m5-d10b.** m5-d10a is the local build wrapper and needs only the ldflags variable from m5-d2. m5-d10b is the goreleaser configuration and the target-compile check, which need O2 and O20 and touch CI. Splitting them lets the local wrapper land in wave 2 without waiting on the release decisions.
13. **m5-d14 is split into m5-d14a and m5-d14b.** m5-d14a corrects `doc/usage.md` against the code, after the narrow-bar, grant, paste and wrap outcomes (waves 1 to 4). m5-d14b writes `doc/troubleshooting.md` from the doctor lines and links the install page, so it waits for the doctor steps and m5-d13c, which run in waves 1 to 5. Each page's dependencies then sit in earlier waves.

---

## Item → deliverable coverage (DIR-014 — mechanically checkable)

Every S, C, D and X item appears exactly once in the table below. The IDs are those first recorded in the mission-20261023-06 research snapshot. The *Source* column carries each item's `file:line`, so the table does not depend on that snapshot after it is archived. Backlog items map to "backlog" and are listed under *Backlog*. The expected ID set is the ranges typed once, in the script below. The check reads only this file.

Run with `bash` from any directory. It prints the expected-but-absent IDs, the IDs in the table that are not expected (each counted once), the duplicated IDs, and a verdict. The positive control runs in the same invocation, on a copy that is byte-identical to this file before one row is duplicated and another deleted. The script aborts if the copy differs from the source, and it aborts if the control does not report FAIL with both the absent S7 and the duplicated C5. The temporary directory is removed on exit.

```bash
export LC_ALL=C
F=/Volumes/DATA/Github/kirsch/plan/milestones/milestone-5.md
SCR=$(mktemp -d) || exit 1
trap 'rm -f "$SCR/expected" "$SCR/table" "$SCR/table.uniq" "$SCR/corrupt.md"; rmdir "$SCR"' EXIT

{ seq -f 'S%g' 1 36; seq -f 'C%g' 1 61; seq -f 'D%g' 1 8; seq -f 'X%g' 1 8; } | sort > "$SCR/expected"

check_coverage() {
  awk -F'|' '/^\| [SCDX][0-9]+ \|/ { gsub(/ /, "", $2); print $2 }' "$1" | sort > "$SCR/table"
  uniq "$SCR/table" > "$SCR/table.uniq"
  echo "== $1"
  echo "-- expected but absent from the table:"
  comm -23 "$SCR/expected" "$SCR/table.uniq"
  echo "-- in the table but not expected:"
  comm -13 "$SCR/expected" "$SCR/table.uniq"
  echo "-- duplicated in the table:"
  uniq -d "$SCR/table"
  if [ -z "$(comm -3 "$SCR/expected" "$SCR/table.uniq")" ] && [ -z "$(uniq -d "$SCR/table")" ]; then
    echo "VERDICT: PASS"
  else
    echo "VERDICT: FAIL"
  fi
}

check_coverage "$F"

# Positive control, same invocation. The copy must be identical to the source
# before corruption. The corruption duplicates the C5 row and deletes the S7 row.
cp "$F" "$SCR/corrupt.md" && cmp -s "$F" "$SCR/corrupt.md" || { echo "ABORT: control copy differs from source"; exit 1; }
echo "control copy identical to source before corruption"
awk '/^\| C5 \|/ { print; print; next } /^\| S7 \|/ { next } { print }' "$F" > "$SCR/corrupt.md"
control_out=$(check_coverage "$SCR/corrupt.md")
echo "$control_out"
echo "$control_out" | grep -qx 'VERDICT: FAIL' || { echo "ABORT: control did not FAIL"; exit 1; }
absent_section() { awk '/^-- expected but absent/ { on = 1; next } /^-- / { on = 0 } on'; }
dup_section() { awk '/^-- duplicated/ { on = 1; next } /^-- / { on = 0 } on'; }
echo "$control_out" | absent_section | grep -qx 'S7' || { echo "ABORT: control did not report S7 absent"; exit 1; }
echo "$control_out" | dup_section | grep -qx 'C5' || { echo "ABORT: control did not report C5 duplicated"; exit 1; }
```

The check depends only on this file. A pass means every expected ID appears once. The control shows the check reports a missing ID and a duplicate and returns FAIL, so a PASS on the real file means something.

| ID | Item | Maps to | Source (file:line) |
|----|------|---------|---------------------|
| S1 | Bracketed paste: multi-line stack traces literal | m5-d6 | `internal/tui/update.go:707–709`; `internal/tui/composer.go:61–63`; `plan/spec/kirsch-plan.md:478` |
| S2 | Soft-wrap long lines, keep indentation | m5-d7a, m5-d7b | `internal/tui/view.go:165–197, :602–613`; `plan/spec/ui-spec-v0.1.md:227, :232–234, :565–571` |
| S3 | Zero-width resize safety | m5-d9 | `internal/tui/view.go:434–436, :554–558`; `internal/tui/view_smoke_test.go:32–55` |
| S4 | No-colour fallback detection | m5-d9 | `internal/tui/styles.go:44–60`; `plan/spec/ui-spec-v0.1.md:572–575, :763` |
| S5 | Paste debouncing | m5-d8 | `internal/tui/styles.go:22`; `plan/spec/ui-spec-v0.1.md:561, :753` |
| S6 | `doctor`: binary version | m5-d3a | `cmd/kirsch/main.go:27–28`; `plan/spec/kirsch-plan.md:479` |
| S7 | `doctor`: `rg` present | m5-d3b | `internal/tool/searchcode.go:161–163`; `internal/app/app.go:1450–1453` |
| S8 | `doctor`: `git` present | m5-d3b | `internal/workspace/workspace.go:94, :116, :395` |
| S9 | `doctor`: repository detected | m5-d3b | `internal/workspace/workspace.go:78–103`; `plan/spec/kirsch-plan.md:479` |
| S10 | `doctor`: key resolvable (name, never value) | m5-d4a | `internal/config/config.go:521–532` |
| S11 | `doctor`: config parses | m5-d3b | `internal/config/config.go:270, :604` |
| S12 | `doctor`: session directory writable | m5-d4b | `internal/app/app.go:831–855`; `internal/session/layout.go:43–49` |
| S13 | `doctor`: active `search_code` backend | m5-d4a | `internal/tool/searchcode.go:116–123` |
| S14 | `doctor`: resolved model and whether in the table | m5-d4a | `internal/provider/models.go:16–20, :106` |
| S15 | `doctor`: project-context file loaded | m5-d4a | `internal/agent/prompt/loader.go:50`; `internal/app/app.go:724–750` |
| S16 | `doctor`: lock state | m5-d4b | `internal/session/lock.go:29–60` |
| S17 | `doctor`: dispatch, report format, exit codes | m5-d3a | `cmd/kirsch/main.go:44–51`; `plan/spec/kirsch-plan.md:479, :487` |
| S18 | `kirsch --version` via ldflags | m5-d2 | `cmd/kirsch/main.go:27–28, :55–57`; `plan/spec/kirsch-plan.md:480` |
| S19 | `kirsch --help` | m5-d2; doctor line in m5-d3a | `cmd/kirsch/main.go:31–36, :54, :58–60`; `plan/spec/kirsch-plan.md:480` |
| S20 | goreleaser release config | m5-d10b | `plan/spec/kirsch-plan.md:481`; `.gitignore:10` |
| S21 | `scripts/build.sh` wrapper | m5-d10a | `plan/spec/kirsch-plan.md:481`; `package.json:9` |
| S22 | Builds for every target, and the CI check | m5-d10b | `plan/spec/kirsch-plan.md:481, :487`; `internal/tool/run_command.go:183, :353–369`; `internal/session/lock.go:50` |
| S23 | GitHub release workflow | m5-d11 | `plan/spec/kirsch-plan.md:481`; `.github/workflows/ci.yml:9–10` |
| S24 | `CONTRIBUTING.md` | m5-d12 | `plan/spec/kirsch-plan.md:482`; `README.md:62–64` |
| S25 | `SECURITY.md` | m5-d12 | `plan/spec/kirsch-plan.md:482`; `internal/tool/run_command.go:395–401`; `internal/provider/anthropic/client.go:453` (the call site; the request body is written in `logBadRequest`, `client.go:506–512`) |
| S26 | `NOTICE` | m5-d12 | `plan/spec/kirsch-plan.md:482`; `go.mod:6–15` |
| S27 | `doc/install.md` | m5-d13c | `plan/spec/kirsch-plan.md:483`; `doc/README.md:13` |
| S28 | `doc/configuration.md` | m5-d13a | `plan/spec/kirsch-plan.md:483`; `internal/config/config.go:27–118` |
| S29 | `doc/usage.md` corrections | m5-d14a, m5-d14b | `plan/spec/kirsch-plan.md:483`; `doc/usage.md:259`; `doc/README.md:3, :19` |
| S30 | `doc/troubleshooting.md` | m5-d14b | `plan/spec/kirsch-plan.md:483`; `doc/README.md:16` |
| S31 | README install section | m5-d15; m5-d22 (banner) | `plan/spec/kirsch-plan.md:484`; `README.md:5–7, :99–101` |
| S32 | Screenshots or asciinema demo | m5-d22 | `plan/spec/kirsch-plan.md:484`; O12 |
| S33 | Tag `v0.1.0` | m5-d22 | `plan/spec/kirsch-plan.md:485`; `.git/refs/tags/` |
| S34 | Smoke test re-run on the release build | m5-d22 | `plan/spec/kirsch-plan.md:487`; `plan/PROGRESS.md:117` |
| S35 | Fresh-machine flow: no key gives onboarding | m5-d16 | `plan/spec/kirsch-plan.md:487`; `internal/app/app.go:1298–1324`; `scripts/smoke-test.sh:16–22` |
| S36 | Every §5 config key documented (mechanical check) | m5-d13b | `plan/spec/kirsch-plan.md:487`; `internal/config/config.go:57–80, :219–259` |
| C1 | Narrow-width status-bar fallback | m5-d1 | `plan/PROGRESS.md:67`; `internal/tui/view.go:447–448` |
| C2 | Absence-check tests use fixed sleeps | backlog | `plan/PROGRESS.md:78` |
| C3 | Out-of-order `SessionUsageMsg` can leave stale totals | backlog | `plan/PROGRESS.md:99` |
| C4 | Mid-session pricing change can understate or hide cost | backlog | `plan/PROGRESS.md:99` |
| C5 | Resumed session shows no totals until new usage | m5-d18 | `plan/PROGRESS.md:99`; `internal/app/app.go:1058–1063` |
| C6 | `applyCompaction` error arm in `Compact` untested | backlog | `plan/PROGRESS.md:99` |
| C7 | Rename-aside-then-write-failure consequence undocumented | backlog | `plan/PROGRESS.md:99` |
| C8 | Usage-lock timeout drops an increment | m5-d20 (decision); m5-d21g (if changed) | `plan/PROGRESS.md:99`; `internal/session/usage.go:28, :152` |
| C9 | Cache-write prices unverified | m5-d20 (decision); m5-d21a (model table); m5-d22 (re-read) | `plan/PROGRESS.md:99`; `internal/provider/models.go:33–36` |
| C10 | Race mutant for the resume test killed in 1 of 20 runs | backlog | `plan/PROGRESS.md:99` |
| C11 | Final-report survival is instruction-level only | m5-d20 (decision); m5-d22 (re-check) | `plan/PROGRESS.md:105` |
| C12 | "Limitations" pinned as a bare word | backlog | `plan/PROGRESS.md:106` |
| C13 | `system.md` mutants not independently rerun | backlog | `plan/PROGRESS.md:107` |
| C14 | Stray `.bak` files | m5-d20 (decision); m5-d22 (delete) | `plan/PROGRESS.md:108` |
| C15 | Smoke test: approval and TestDivide-baseline branches lack an offline negative control | backlog | `plan/PROGRESS.md:114` |
| C16 | Comment in `smoke_live_test.go` overstates containment | backlog | `plan/PROGRESS.md:115` |
| C17 | Zero-divisor check is a source regex | backlog | `plan/PROGRESS.md:116` |
| C18 | `stop_reason`, `tool`, `duration_ms` omitted from session events | backlog | `plan/PROGRESS.md:126, :178, :216` |
| C19 | Other agent edge cases | backlog | `plan/PROGRESS.md:127` |
| C20 | Calibration warnings | backlog | `plan/PROGRESS.md:128` |
| C21 | `max_turns_exceeded` error card (closed in code) | m5-d15 | `plan/PROGRESS.md:129`; `internal/app/app.go:1999–2000, :2011` |
| C22 | Cancellation goroutine not joined | backlog | `plan/PROGRESS.md:130` |
| C23 | Every 401/403 reads "rejected the API key" | m5-d14b (doc remedy); code change in Backlog | `plan/PROGRESS.md:137`; `internal/provider/anthropic/client.go:447` |
| C24 | 400 debug log writes the full request | m5-d20 (decision); m5-d21c (clip) or m5-d21d (disclose) | `plan/PROGRESS.md:138`; `internal/provider/anthropic/client.go:453` |
| C25 | Config review items | backlog | `plan/PROGRESS.md:139` |
| C26 | Wire-format review items | backlog | `plan/PROGRESS.md:140` |
| C27 | Live-test review items | backlog | `plan/PROGRESS.md:141` |
| C28 | Default-model review items | backlog | `plan/PROGRESS.md:142` |
| C29 | 500 ms box for SIGTERM-ignoring processes | m5-d19b | `plan/PROGRESS.md:148`; `plan/milestones/milestone-2.md:418–419`; `internal/tool/run_command.go:353–369` |
| C30 | `run_command` process-group tests cannot fail | m5-d19a | `plan/PROGRESS.md:149`; `internal/tool/run_command_test.go` |
| C31 | Suspected trailing-newline defect in `internal/patch` | backlog | `plan/PROGRESS.md:150` |
| C32 | Unwired settings and hooks (closed in code) | m5-d15 | `plan/PROGRESS.md:151`; `internal/app/app.go:500` |
| C33 | Configured command timeout ignored (closed in code) | m5-d15 | `plan/PROGRESS.md:152`; `internal/tool/run_command.go:144–151` |
| C34 | Stdin wording in the Task 9 box | m5-d20 (operator wording) | `plan/PROGRESS.md:153` |
| C35 | Secret-name patterns partly untested | m5-d19a | `plan/PROGRESS.md:154`; `internal/tool/run_command_test.go:523–539`; `internal/tool/run_command.go:395–401` |
| C36 | Blank approval card for an unspecified operation | m5-d19a | `plan/PROGRESS.md:155` |
| C37 | No `-update` flag for goldens | m5-d12 | `plan/PROGRESS.md:156`; `plan/spec/kirsch-plan.md:498`; `plan/spec/ui-spec-v0.1.md:824`; `internal/tui/regen_test.go:13` |
| C38 | Walkthrough checks not retested (M2 §10, §1) | m5-d20 (decision); m5-d22 (retest) | `plan/PROGRESS.md:157`; `plan/testing/manual/user-testing/milestone-2-donovan.md` |
| C39 | Stale code comments | m5-d15 | `plan/PROGRESS.md:158`; `internal/policy/policy.go:57–58`; `internal/tool/run_command.go:334–341, :355–357` |
| C40 | ULID dependency decision | m5-d20 (decision); m5-d21e (if changed) | `plan/PROGRESS.md:170`; `internal/session/layout.go:96–140` |
| C41 | Fuzz seeds with `..` for `FuzzSessionPathContained` | backlog | `plan/PROGRESS.md:171` |
| C42 | `usage.json` cost field (closed in code) | m5-d15 | `plan/PROGRESS.md:172`; `internal/session/usage.go:81–85` |
| C43 | §7 flattening of interleaved text and tool calls | m5-d20 (decision); m5-d21f (§7 amendment) | `plan/PROGRESS.md:186`; `plan/spec/kirsch-plan.md:340–357` |
| C44 | Pending-map leniency | backlog | `plan/PROGRESS.md:187–188` |
| C45 | Per-event fuzz seeds; deep-copy comment | backlog | `plan/PROGRESS.md:188` |
| C46 | `session.started` / `session.resumed` not written | m5-d20 (decision); m5-d21f (§7 amendment) | `plan/PROGRESS.md:194`; `plan/spec/kirsch-plan.md:342–343` |
| C47 | No auto-resume happy-path test in `main_test.go` | backlog | `plan/PROGRESS.md:195` |
| C48 | `/new` clears busy while pending; stray error card | m5-d17b | `plan/PROGRESS.md:201`; `internal/tui/update.go:1152–1163` |
| C49 | Late async messages from a cancelled turn not fenced | m5-d17b | `plan/PROGRESS.md:202`; `internal/app/app.go`; `internal/tui/` |
| C50 | `/status` compaction % stale after `/compact` | m5-d18 | `plan/PROGRESS.md:203` |
| C51 | Quit during `/new` can leave an orphan empty session file | m5-d17a | `plan/PROGRESS.md:204`; `internal/app/app.go:947, :954–957` |
| C52 | Provider session header id persists across `/new` | m5-d17a | `plan/PROGRESS.md:205`; `cmd/kirsch/main.go:199`; `internal/app/app.go:943–997` |
| C53 | `/diff` shows unstaged changes only | backlog | `plan/PROGRESS.md:206` |
| C54 | `/files` resets on resume | backlog | `plan/PROGRESS.md:207` |
| C55 | `update_test.go` and `status_test.go` overlap | backlog | `plan/PROGRESS.md:208` |
| C56 | Flush-ordering edge (500 ms window) | backlog | `plan/PROGRESS.md:214` |
| C57 | 100 ms settle window | backlog | `plan/PROGRESS.md:215` |
| C58 | Per-field mutant evidence for Signature only | backlog | `plan/PROGRESS.md:217` |
| C59 | `held_lock_no_warning` fixed 200 ms settle | backlog | `plan/PROGRESS.md:223` |
| C60 | Lock-unavailable fallback has no automated test | backlog | `plan/PROGRESS.md:224` |
| C61 | Outstanding verification (M0 and M1 walkthrough checks) | m5-d20 (decision); m5-d22 (tick) | `plan/PROGRESS.md:262`; `plan/testing/README.md:97–101` |
| D1 | `usage.md` debug-commands section | m5-d14a | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:35`; `doc/usage.md:257–270` |
| D2 | Status-bar mock omits cache | m5-d14a | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:36`; `doc/usage.md:54`; `internal/tui/statusbar.go:225` |
| D3 | `/approvals` clear not persisted | m5-d5 | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:37`; `internal/app/app.go:528–529` |
| D4 | `/files` empty after resume, undocumented | m5-d14a | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:38`; `doc/usage.md:247`; `internal/app/app.go:1006–1070` |
| D5 | "0001–0007" ADR range | m5-d15 | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:39`; `plan/spec/kirsch-plan.md:391`; `plan/adr/README.md:64` |
| D6 | `usage.md` says every action waits on a card | m5-d14a | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:39`; `doc/usage.md:299–302` |
| D7 | "Starting it" omits `kirsch resume` and `--new` | m5-d14a | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:39`; `doc/usage.md:22–26` |
| D8 | README says every script is one line of go or bash | m5-d15 | `.claude/memory/missions/mission-20261023-05/DEBRIEF.md:39`; `README.md:142–143`; `package.json:35` |
| X1 | `usage.md` claims a missing-`rg` warning nothing emits | m5-d14a | `doc/usage.md:69–71` |
| X2 | CHANGELOG and PROGRESS claim an `N tok` narrow fallback | m5-d15 | `CHANGELOG.md:16`; `plan/PROGRESS.md:231` |
| X3 | README says one provider only | m5-d15 | `README.md:47`; `plan/spec/kirsch-plan.md:17` |
| X4 | Composer shows only its first rows; caret can be off screen | m5-d6 | `internal/tui/composer.go:100–106, :138–143, :199–205`; `internal/tui/view.go:461` |
| X5 | Model-table figures dated 2026-10-02 | m5-d20 (decision); m5-d21a (model table); m5-d22 (re-read, if m5-d21a did not) | `internal/provider/models.go:33–36`; `plan/spec/kirsch-plan.md:270, :274` |
| X6 | §5 command-line flags precedence layer unwired | m5-d13a (documents no flags); wiring in Backlog | `internal/config/config.go:496–506`; `cmd/kirsch/main.go:71–75` |
| X7 | "Not in a Git repo" is stderr text, not an onboarding screen | m5-d16 | `cmd/kirsch/main.go:62–65, :34`; `plan/spec/ui-spec-v0.1.md:597` |
| X8 | Help overlay footer says `docs: doc/usage.md` | m5-d20 (decision); m5-d21b (footer and goldens) | `internal/tui/modal.go:308` |

---

## Parallel waves

Within a wave the deliverables touch disjoint files. A deliverable runs only after its *Depends on* cell is satisfied and its O-decisions are recorded.

- **Wave 1:** m5-d1, m5-d2, m5-d5, m5-d9, m5-d13a, m5-d19a.
- **Wave 2:** m5-d3a (after m5-d2), m5-d6 (after O10), m5-d10a (after m5-d2), m5-d10b (after m5-d2), m5-d19b (after m5-d19a and O15's C29 item).
- **Wave 3:** m5-d3b (after m5-d3a), m5-d7a (after m5-d1 and m5-d6), m5-d8 (after m5-d6), m5-d11 (after m5-d10b), m5-d13b (after m5-d13a), m5-d13c (after m5-d2, m5-d10a and m5-d10b), m5-d16 (after O14, O19 and m5-d10a; after m5-d2 and m5-d3a for file ownership), m5-d17a (after m5-d5).
- **Wave 4:** m5-d4a (after m5-d3b), m5-d7b (after m5-d6 and m5-d7a), m5-d14a (after m5-d1, m5-d5, m5-d6 and m5-d7a), m5-d15 (after m5-d1, m5-d10a, m5-d10b and m5-d13c).
- **Wave 5:** m5-d4b (after m5-d4a), m5-d12 (after m5-d10a and m5-d10b; after m5-d8 for `ui-spec-v0.1.md`, amend branch), m5-d17b (after m5-d8 and m5-d17a).
- **Wave 6:** m5-d14b (after m5-d3a, m5-d3b, m5-d4a, m5-d4b, m5-d13a and m5-d13c), m5-d18 (after m5-d17a, for file ownership only).
- **Gate:** m5-d20 (operator). Must be complete before any m5-d21 deliverable runs. The C29 item of O15 is the exception: it must be recorded before m5-d19b runs in wave 2.
- **Wave 7:** m5-d21a, m5-d21b, m5-d21c, m5-d21d, m5-d21e, m5-d21f, m5-d21g (conditional; only those the m5-d20 decisions call for).
- **Wave 8:** m5-d22 (operator).

**Known file conflicts and how the waves handle them.**

- `internal/app/app.go` has five owners, each in a different wave: m5-d5 (wave 1), m5-d17a (wave 3), the comment edit at `:520–521` by m5-d4a (wave 4), m5-d17b (wave 5, generation counter after m5-d17a) and m5-d18 (wave 6).
- m5-d19a and m5-d19b (the safety-test group) touch only the files named in their rows. They share no file with m5-d5, m5-d1 or m5-d6.
- `view_smoke_test.go` belongs to m5-d9 alone. m5-d6's tests go in the new `paste_path_test.go`.
- m5-d6 touches `view.go` only if O10 is wrap. It sits in wave 2, where no other deliverable touches `view.go`.
- m5-d8 (wave 3) owns `update.go` after m5-d6 (wave 2). m5-d17b (wave 5) owns it after m5-d8.
- m5-d8 (amend branch) and m5-d12 both edit `plan/spec/ui-spec-v0.1.md`. m5-d12 therefore sits in wave 5, after m5-d8.

**If O10 is wrap.** m5-d6 still sits in wave 2. It then also owns `view.go`, which no other wave-2 deliverable touches. No other wave changes.

**Wave check (mechanical).** The wave lists and the *Wave* column are kept by hand. The check below compares them. It also checks that each deliverable named in a *Depends on* cell sits in an earlier wave, with the gate ranked between waves 6 and 7. It reads only this file. The control moves the m5-d15 *Wave* cell to 2 in a scratch copy. The check must then report a mismatch and an order violation for m5-d15, and return FAIL.

```bash
export LC_ALL=C
F=/Volumes/DATA/Github/kirsch/plan/milestones/milestone-5.md
SCR=$(mktemp -d) || exit 1
trap 'rm -f "$SCR/facts" "$SCR/ctrl.md"; rmdir "$SCR"' EXIT

check_waves() {
  awk -F'|' '
    /^\| m5-d[0-9]+[a-z]? / {
      id = $2; match(id, /m5-d[0-9]+[a-z]?/); id = substr(id, RSTART, RLENGTH)
      wave = $6; gsub(/ /, "", wave)
      print "ROW", id, wave
      dep = $4
      while (match(dep, /m5-d[0-9]+[a-z]?/)) { print "DEP", id, substr(dep, RSTART, RLENGTH); dep = substr(dep, RSTART + RLENGTH) }
    }
    /^- \*\*(Wave [0-9]+|Gate):\*\*/ {
      label = $0; sub(/^- \*\*/, "", label); sub(/:\*\*.*/, "", label)
      w = label; if (label == "Gate") w = "gate"; else sub(/Wave /, "", w)
      body = $0; sub(/^- \*\*[^*]*\*\*/, "", body); sub(/\. .*/, "", body); gsub(/\([^)]*\)/, "", body)
      while (match(body, /m5-d[0-9]+[a-z]?/)) { print "MEM", substr(body, RSTART, RLENGTH), w; body = substr(body, RSTART + RLENGTH) }
    }' "$1" > "$SCR/facts"
  awk '
    function rank(w) { return (w == "gate") ? 6.5 : w + 0 }
    $1 == "ROW" { wave[$2] = $3; order[++nrow] = $2 }
    $1 == "MEM" { cnt[$2]++; where[$2] = where[$2] " " $3 }
    $1 == "DEP" { deps[++nd] = $2 " " $3 }
    END {
      bad = 0
      for (i = 1; i <= nrow; i++) {
        id = order[i]
        if (cnt[id] != 1 || where[id] != " " wave[id]) { print "MISMATCH " id ": Wave cell " wave[id] ", wave lists:" where[id]; bad++ }
      }
      for (j = 1; j <= nd; j++) {
        split(deps[j], p, " "); a = p[1]; b = p[2]
        if ((b in wave) && rank(wave[b]) >= rank(wave[a])) { print "ORDER " a " (wave " wave[a] ") depends on " b " (wave " wave[b] ")"; bad++ }
      }
      print (bad == 0 ? "VERDICT: PASS" : "VERDICT: FAIL")
      exit (bad == 0 ? 0 : 1)
    }' "$SCR/facts"
}

echo "== real file"
check_waves "$F"; echo "exit=$?"
echo "== control: m5-d15 Wave cell set to 2 in a scratch copy"
awk -F'|' -v OFS='|' '$2 ~ /^ m5-d15 $/ { $6 = " 2 " } { print }' "$F" > "$SCR/ctrl.md"
check_waves "$SCR/ctrl.md"; echo "exit=$?"
```

The real file must print `VERDICT: PASS` with exit 0. The control must print a `MISMATCH m5-d15` line, an `ORDER m5-d15` line, and `VERDICT: FAIL` with exit 1.

---

## Manual test hooks

The file `plan/testing/manual/milestone-5.md` does not exist yet. It is created at step 4 of mission-20261023-06. Each `§ m5-dNx` anchor below is a section in that file. Each split part has its own anchor.

---

## m5-d1 — Narrow-width status-bar fallback

**Scope:** Decide what the status bar shows below 80 columns (O1), then change the band rule, the ladder and the pinned test to match. The ui-spec text is amended to agree with the code.

**Items covered:**
- C1 — `plan/PROGRESS.md:67`. Tokens are shown only in the ≥80 band: `internal/tui/view.go:447–448`, `internal/tui/statusbar.go:149`, comment `statusbar.go:44–45`. Pinned by `internal/tui/turn_messages_test.go:619–627`. The `N tok` rung exists only inside the wide band (`statusbar.go:154`).

**Dependencies:** O1 (recorded first) and O23 (golden handling).

**File ownership:** `internal/tui/view.go`, `internal/tui/statusbar.go`, `internal/tui/turn_messages_test.go`, `plan/spec/kirsch-ui-screens.md` (the sized fences below 80 columns that instruction 1 lists), `plan/spec/ui-spec-v0.1.md` §2.2 and §2.3.

**Parallel execution:** Wave 1 (see *Parallel waves*). Disjoint files. m5-d7a follows it (`view.go`).

**Mission steps:**
1. `@hicks` — code: band rule, ladder, comment, pinned test, and goldens if instruction 1 finds any.
2. `@apone` — review of step 1.
3. `@lambert` — docs: ui-spec §2.2 (`plan/spec/ui-spec-v0.1.md:124`) and §2.3 (`:170–173`).
4. `@apone` — review of step 3.

**Instruction items:**
1. Record O1 in `plan/PROGRESS.md` before writing the brief. Quote the chosen behaviour into the brief. Save `plan/spec/kirsch-ui-screens.md` verbatim to `.claude/memory/workspace/m5-d1-kirsch-ui-screens.pre.md` before any change. Then list the sized fences below 80 columns with `grep -n '^```text [0-9]*×' plan/spec/kirsch-ui-screens.md | awk -F'[ ×]' '$2+0 < 80'`, and record them by fence header. If the list is empty, record "no golden below 80 columns" and change no fence.
2. Change the band rule. Tokens are gated to the wide band at `view.go:447–448` and `statusbar.go:149`. Move or add the gate so the narrow bands show what O1 chose.
3. Change the ladder. The `N tok` rung is currently inside the wide band (`statusbar.go:154`). Add or move it as O1 requires.
4. Update the comment at `statusbar.go:44–45`, which says "Narrower than the wide layout no token figure is shown", so it matches the new rule.
5. Flip or rewrite `turn_messages_test.go:619–627` ("below the spec width: no tokens at all") to pin O1's behaviour at 60–79 columns and below 60.
6. Amend `plan/spec/ui-spec-v0.1.md` §2.2 (`:124`) and §2.3 (`:170–173`) to match the code and O1.
7. Change only the fences that the list in instruction 1 names, as O23 records and by the procedure in *Golden mechanism and regeneration procedure*. Fences at 80 columns and wider must not move. The acceptance box checks this.

**Acceptance:**
- [ ] O1 is recorded in `plan/PROGRESS.md` before this brief runs.
- [ ] At 60–79 columns and below 60, the status bar shows exactly what O1 specifies.
- [ ] `TestSessionUsageNarrowLayoutFallsBackToSessionTotal` in `internal/tui/turn_messages_test.go` pins O1's behaviour and passes.
- [ ] ui-spec §2.2 and §2.3 agree with the code.
- [ ] Sized fences at 80 columns and wider are unchanged. The command extracts them from the committed copy and from the file, and compares them. The committed copy is the pre-change baseline only until the next commit that touches `plan/spec/kirsch-ui-screens.md`; after that commit, point `$PRE` at the commit before the change instead. Empty diff output means pass. The first output line is the count of wide fences in the file, so a count of zero shows that the extraction read nothing.

  ```bash
  PRE=$(mktemp); git -C /Volumes/DATA/Github/kirsch show HEAD:plan/spec/kirsch-ui-screens.md > "$PRE"
  NOW=/Volumes/DATA/Github/kirsch/plan/spec/kirsch-ui-screens.md
  wide() { awk '/^```text [0-9]+×[0-9]+$/ { split($2, d, "×"); keep = (d[1] + 0 >= 80); if (keep) print "@@FENCE " $2; infence = 1; next }
       /^```$/ { infence = 0; keep = 0; next }
       infence && keep { print }' "$1"; }
  wide "$NOW" | grep -c '^@@FENCE'
  diff <(wide "$PRE") <(wide "$NOW")
  # control: one character appended to the first line of the first wide fence in a scratch copy
  CTRL=$(mktemp)
  awk 'BEGIN { d = 0; m = 0 } /^```text 80×/ && !d { on = 1; d = 1; print; next } on && /^```$/ { on = 0 } on && !m { print $0 "X"; m = 1; next } { print }' "$PRE" > "$CTRL"
  diff <(wide "$PRE") <(wide "$CTRL")
  rm -f "$CTRL" "$PRE"
  ```

  The control's `diff` must print a difference. Without that, an empty diff proves nothing.
- [ ] `npm run check` exits 0.
- [ ] The step report ends with the IMPROVEMENT-NOTE and STEP COMPLETE lines.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d1 (Walkthrough).

**Must-know-before:**
- O1 is the blocker. The M4 acceptance, `CHANGELOG.md:16`, `plan/PROGRESS.md:231` and the walkthrough step (`plan/testing/manual/user-testing/milestone-4-donovan.md:1042`) disagree with ui-spec §2.2:124. Do not pick a side in the brief. Record O1.
- X2 (the CHANGELOG and PROGRESS claims) is corrected in m5-d15, after this lands.
- This is the item the M4 walkthrough marked ❌ (`plan/PROGRESS.md:67`). O1 resolved it as a wrong expectation, not a code defect.

---

## m5-d2 — `--version` and `--help`

**Scope:** A `--version` flag that works outside a repository and prints the ldflags-injected value. A `--help` that names the commands and flags and exits 0.

**Items covered:**
- S18 — `cmd/kirsch/main.go:27–28` (the variable); `main.go:55–57` (flags today: `workspace`, `new`, `debug`). The TUI prefixes its own `v` (`internal/tui/empty.go:81`, `internal/tui/onboarding.go:37`, `internal/tui/modal.go:308`).
- S19 — `main.go:54` (`flag.NewFlagSet("kirsch", flag.ContinueOnError)`); `main.go:31–36, :58–60` (the help path, not yet run); `main.go:49–50` (`kirsch help` is an unknown command).

**Dependencies:** O4 (recorded first). m5-d10a, m5-d10b and m5-d13c depend on this one.

**File ownership:** `cmd/kirsch/main.go`, `cmd/kirsch/main_test.go`.

**Parallel execution:** Wave 1 (see *Parallel waves*). Disjoint files. m5-d3a follows (both touch `main.go`).

**Mission steps:**
1. `@hicks` — code: the flag, the usage text, the exit status, and the tests.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O4 first: the bare value in ldflags, whether `--version` prints commit and date, and whether a `debug.ReadBuildInfo` fallback is wanted.
2. Add the `--version` flag. It must exit before `workspace.Detect` (`main.go:62`), so it works outside a Git repository.
3. Keep the variable at `main.go:27–28`. The injected value must be bare (`0.1.0`). A leading `v` would show as `vv0.1.0`.
4. Replace the default usage on the `flag.NewFlagSet` at `main.go:54` with a custom usage that names the commands and flags. Name `resume` (`main.go:46–48`) and `--version`. Do not name `doctor` here; m5-d3a adds it.
5. Make `-h` and `--help` exit 0. Today they exit 1 (`main.go:31–36, :58–60`). This is unverified until run; record the observed status in the step report.
6. Add tests in `cmd/kirsch/main_test.go` for `--version` and `--help`.

**Acceptance:**
- [ ] `kirsch --version` run outside a Git repository prints the version and exits 0.
- [ ] A build with `-ldflags "-X main.version=9.9.9"` prints `9.9.9`, with no leading `v`.
- [ ] `kirsch --help` lists the commands and the flags.
- [ ] `echo $?` after `kirsch --help` prints 0.
- [ ] `cmd/kirsch/main_test.go` covers both flags.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d2 (Walkthrough).

**Must-know-before:**
- O4 is the blocker.
- `main.go:62–65` (no repository) and `:71–81` (bad config) return before anything else, so both flags must sit ahead of them.
- The `--help` exit status is unverified until this step runs it.

---

## m5-d3a — Doctor: dispatch, report grammar, version line, exit codes, arch entry

**Scope:** The `doctor` subcommand, its dispatch ahead of repository and config checks, its report grammar and exit codes (O5), the version line, and the `doctor` line in the help text with its test. Also the `internal/arch` entry and the layout line for a new package.

**Items covered:**
- S6 — version: `cmd/kirsch/main.go:27–28`.
- S17 — dispatch and report: `main.go:44–51` accepts only `resume`; anything else is `unknown command` (`:50`). The spec fixes no format or exit codes (`plan/spec/kirsch-plan.md:479, :487`).
- The help text gains a `doctor` line, owned here, per grouping note 1.

**Dependencies:** m5-d2 (the help text and flag dispatch order), O5 (recorded first).

**File ownership:** `cmd/kirsch/main.go`, `cmd/kirsch/main_test.go`, `internal/doctor/` (new package), `internal/arch/imports_test.go`, `plan/spec/architecture.md` (layout line at `:31–49`, docs step).

**Parallel execution:** Wave 2 (see *Parallel waves*). m5-d3b follows this one.

**Mission steps:**
1. `@hicks` — code: dispatch, version line, report grammar and exit codes per O5, the `doctor` help line and its test in `main_test.go`, the `internal/doctor` package, and the `internal/arch/imports_test.go` entry.
2. `@apone` — review of step 1.
3. `@lambert` — docs: the layout line at `plan/spec/architecture.md:31–49`.
4. `@apone` — review of step 3.

**Instruction items:**
1. Record O5 first: package placement, output grammar, fail versus warn, exit codes, and whether doctor may create directories or take the lock.
2. Add `doctor` to the dispatch at `main.go:44–51`. It runs ahead of `workspace.Detect` (`main.go:62`) and ahead of config load (`main.go:71–81`), so it reports when there is no repository and when the config is broken.
3. Print the version first (S6), from `main.version` at `main.go:27–28`.
4. Add `doctor` to the help text, and add a test in `cmd/kirsch/main_test.go` that asserts the help output names it.
5. Set the report grammar and exit codes exactly as O5 records them. Every check line in later deliverables uses this grammar.
6. If O5 chooses a new package, add its `internal/arch/imports_test.go` entry and the layout line in `plan/spec/architecture.md:31–49`.
7. Make no network calls (O5 records this).

**Acceptance:**
- [ ] `kirsch doctor` is dispatched ahead of `workspace.Detect` and config load (test).
- [ ] The first doctor line shows the version (test).
- [ ] The help output names `doctor` (test in `cmd/kirsch/main_test.go`).
- [ ] Exit status matches O5 for each status O5 defines (test).
- [ ] The `internal/arch` test passes with the new package entry.
- [ ] `plan/spec/architecture.md:31–49` names the new package.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d3a (Walkthrough).

**Must-know-before:**
- O5 is the blocker. The spec fixes no output format and no exit codes.
- The comments that already name doctor (`internal/tool/searchcode.go:116–117` and `internal/app/app.go:520–521`) become true when doctor calls the backend. m5-d4a updates them.

---

## m5-d3b — Doctor: `rg`, `git`, repository and config probes

**Scope:** The checks that need only the environment, the repository and the config file: `rg`, `git`, the repository root, and config parsing.

**Items covered:**
- S7 — `rg` present: `internal/tool/searchcode.go:161–163` (`ripgrepAvailable`, unexported); `internal/app/app.go:1450–1453` (`App.hasRG`, unexported). A missing `rg` degrades to the pure-Go backend (`searchcode.go:118–122`), so it is a warning.
- S8 — `git` present: no `LookPath("git")` exists. Git runs through `exec.Command` (`internal/workspace/workspace.go:94, :116, :395`). A missing `git` makes `Detect` return `ErrNoWorkspace` (`workspace.go:94–96`), which reads as "not in a repo".
- S9 — repository detected: `workspace.Detect` (`workspace.go:78–103`); `ErrNoWorkspace` (`:71–74`); `main.go` returns on error at `:62–65`; `--workspace` is handled inside `Detect`.
- S11 — config parses: `config.Load` returns `(cfg, warnings, err)` (`internal/config/config.go:270`); `Validate` (`:604`). `main.go:71–81` returns on either error. Warnings go only to the debug log (`main.go:128–130`).

**Dependencies:** m5-d3a (dispatch and report grammar), O5.

**File ownership:** `internal/doctor/`.

**Parallel execution:** Wave 3 (see *Parallel waves*). m5-d4a follows.

**Mission steps:**
1. `@hicks` — code: the probes, the config-failure capture, and the PATH-shim tests.
2. `@apone` — review of step 1.

**Instruction items:**
1. Probe `rg` with doctor's own `exec.LookPath("rg")`. Do not reuse `ripgrepAvailable`. A missing `rg` is a warn line that names the pure-Go fallback (`searchcode.go:118–122`).
2. Probe `git` with `exec.LookPath("git")`. "git missing" is a line separate from "not in a Git repository" (`workspace.go:94–96`).
3. Report the repository: the detected root, or "not in a Git repository". Keep running the other checks.
4. Call `config.Load`, catch its error, and print the failing file and message. List unknown-key warnings as warn lines. Doctor does not exit on the error. Record whether config loaded, because m5-d4a reads that record.
5. Honour `--workspace` through `workspace.Detect`.
6. The missing-tool tests use a PATH shim (DIR-009): a temporary directory holding symlinks to every executable on the original PATH except the tool under test. Do not empty PATH. The control: under the same shim PATH, `LookPath("git")` succeeds in the `rg` test, and `LookPath("rg")` still fails.

**Acceptance:**
- [ ] With the PATH shim lacking `rg`, doctor prints a warn line that names the pure-Go fallback (test).
- [ ] With the PATH shim lacking `git`, doctor prints a git-missing line and not "not in a Git repository" (test).
- [ ] Outside a repository, doctor prints "not in a Git repository" and every check that does not need one (test).
- [ ] A broken `config.toml` produces a line naming the file and the parse error, and doctor does not exit on it (test).
- [ ] An unknown config key produces a warn line (test).
- [ ] `--workspace` is honoured (test).

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d3b (Walkthrough).

**Must-know-before:**
- m5-d3a is complete.
- The PATH shim is the method DIR-009 requires for a missing tool. An emptied PATH would also remove the shell.

---

## m5-d4a — Doctor: key, backend, model and project-context lines

**Scope:** The doctor checks that need the resolved configuration: the key, the active search backend, the resolved model, and the project-context file. Also the comment edits that describe doctor.

**Items covered:**
- S10 — key resolvable: `internal/config/config.go:521–532` (`EndpointConfig.ResolveKey`, returns the key and the name of the variable). Used at `internal/app/app.go:1218` and `:1305`.
- S13 — active `search_code` backend: `(*SearchCode).Backend()` (`internal/tool/searchcode.go:116–123`) returns `ripgrep` or `pure-go`.
- S14 — resolved model: `provider.LookupModel` returns `ModelInfo` with `Known` (`internal/provider/models.go:16–20, :106`). `app.go:1198–1206` and `:1306–1309` already use it.
- S15 — project-context file: `prompt.LoadProjectContext` (`internal/agent/prompt/loader.go:50`). The app calls it through the injected `a.loadProjectCtxFn` (`internal/app/app.go:725`; field at `:432`). The inputs are `cfg.Context.ProjectFiles` and `MaxProjectContextBytes` (`app.go:724–750`).
- The comments at `internal/tool/searchcode.go:116–117` and `internal/app/app.go:520–521` describe doctor. They are updated here.

**Dependencies:** m5-d3b (the config-load result), O5.

**File ownership:** `internal/doctor/`, `internal/tool/searchcode.go` (comment only), `internal/app/app.go` (comment only at `:520–521`).

**Parallel execution:** Wave 4 (see *Parallel waves*). m5-d4b follows this one.

**Mission steps:**
1. `@hicks` — code: the lines, the skipped-line handling, and the comment edits.
2. `@apone` — review of step 1.

**Before any edit.** Save verbatim copies of `internal/tool/searchcode.go` to `.claude/memory/workspace/m5-d4a-searchcode.pre.go` and of `internal/app/app.go` to `.claude/memory/workspace/m5-d4a-app.pre.go`. The acceptance box compares against these copies, not against HEAD. m5-d5 (wave 1) also edits `app.go`, so a diff against HEAD would show its code.

**Instruction items:**
1. Key line: print whether `ResolveKey` found a key and the winning variable name. Never print the value (`config.go:521–532`). If config did not load (m5-d3b), print `skipped: config did not load` in place of the key line, the model line and the context line. Do not guess a default.
2. Backend line: call `(*SearchCode).Backend()` and print `ripgrep` or `pure-go`. Update the comments at `searchcode.go:116–117` and `app.go:520–521` so they describe the doctor line that now exists (DIR-015).
3. Model line: print the endpoint, the model id, whether `LookupModel` knows it, and its context window (`models.go:16–20`). An unknown model prints `not in the table` and the conservative fallback.
4. Project-context line: doctor calls `prompt.LoadProjectContext` (`internal/agent/prompt/loader.go:50`), which takes an Engine. Build the Engine the way the app builds it. The brief names that construction after reading `internal/app/app.go:724–750`. Print the file and its size, or `none`. If config did not load, print `skipped: config did not load`.

**Acceptance:**
- [ ] The key line names the winning variable or says none, and never prints the key value (a test sets a known value and asserts the output does not contain it).
- [ ] With `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY` each set and unset, the key line names the winner or says none (test).
- [ ] A model id not in the table prints `not in the table` and the conservative fallback (test).
- [ ] With `AGENTS.md` present, the context line names it and its size; with it absent, the line says `none` (test).
- [ ] The backend line reads `ripgrep` with `rg` on PATH and `pure-go` without, under the PATH shim (test).
- [ ] With config not loaded, the key, model and context lines each read `skipped: config did not load` (test).
- [ ] The comments describe doctor lines that exist, and the two files change only in comments. The check compares each file with its saved copy. Every changed content line must match `^[-+][[:space:]]*//`. The first two commands print nothing on a pass. The last two print a count, and each count must be at least 1, because the edit changes both files.

  ```bash
  PRE=/Volumes/DATA/Github/kirsch/.claude/memory/workspace
  check_comments() { diff -U0 "$1" "$2" | grep -E '^[-+]' | grep -vE '^(---|\+\+\+) ' | grep -vE '^[-+][[:space:]]*//'; }
  changed_lines() { diff -U0 "$1" "$2" | grep -E '^[-+]' | grep -vcE '^(---|\+\+\+) '; }
  check_comments "$PRE/m5-d4a-searchcode.pre.go" /Volumes/DATA/Github/kirsch/internal/tool/searchcode.go
  check_comments "$PRE/m5-d4a-app.pre.go" /Volumes/DATA/Github/kirsch/internal/app/app.go
  changed_lines "$PRE/m5-d4a-searchcode.pre.go" /Volumes/DATA/Github/kirsch/internal/tool/searchcode.go
  changed_lines "$PRE/m5-d4a-app.pre.go" /Volumes/DATA/Github/kirsch/internal/app/app.go
  # control: a scratch copy of the saved app.go with one non-comment line appended
  CTRL=$(mktemp); cp "$PRE/m5-d4a-app.pre.go" "$CTRL"; printf 'var controlProbe = 1\n' >> "$CTRL"; check_comments "$PRE/m5-d4a-app.pre.go" "$CTRL"; rm -f "$CTRL"
  ```

  The control's `check_comments` must print `+var controlProbe = 1`. The reviewer confirms that the hunks sit at `searchcode.go:116–117` and `app.go:520–521` in the saved copies.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d4a (Walkthrough).

**Must-know-before:**
- m5-d3b and O5 are complete.
- S10 prints only the variable name. Never the value.

---

## m5-d4b — Doctor: state directory and lock state

**Scope:** The checks with side effects: whether the session state directory is writable, and whether the lock is held.

**Items covered:**
- S12 — session directory writable: no probe exists. The state root is `App.stateRoot` (`internal/app/app.go:831–855`, unexported), which strips a trailing `sessions` (`app.go:851–853`). `SessionDir` joins `<root>/sessions/<workspace-hash>` (`internal/session/layout.go:78–80`). `session.DefaultStateRoot` is at `layout.go:43–49`. The lock path logs and degrades (`app.go:780–784`).
- S16 — lock state: `session.Acquire` (`internal/session/lock.go:29–60`) returns `ErrLockHeld`. It also creates the session directory and `.lock` (`lock.go:34, :42`), so calling it has side effects.

**Dependencies:** m5-d4a, O5 (side-effect policy and lock method).

**File ownership:** `internal/doctor/`. `internal/session/` only for a read-only probe helper, if O5 requires one.

**Parallel execution:** Wave 5 (see *Parallel waves*). m5-d14b follows this one.

**Mission steps:**
1. `@hicks` — code: the state-directory probe, the lock report, and their tests.
2. `@apone` — review of step 1.

**Instruction items:**
1. Probe path: `<root>/sessions/<hash>`. The root comes from the configured storage directory with a trailing `sessions` stripped (`app.go:851–853`), and `~` is expanded.
2. Writability: follow O5. If O5 allows a write probe, create and remove a temporary file in `<root>/sessions/<hash>`, creating the directory only if O5 allows it. A non-writable directory is flagged. The probe's failure stays out of the lock path's degradation (`app.go:780–784`). If config did not load, print `skipped: config did not load`.
3. Lock: follow O5. Report held or free without disturbing a live instance. A held lock reads held; a free lock reads free.

**Acceptance:**
- [ ] A non-writable state directory is flagged; the test restores the directory mode afterwards (test).
- [ ] While the test holds the lock with `session.Acquire`, doctor reports held; after the release, doctor reports free (test, one process).
- [ ] The held lock is still held after the doctor run, so the live instance is not disturbed (test).
- [ ] The probe path printed is `<root>/sessions/<hash>` for a fixture root (test).
- [ ] With config not loaded, both lines read `skipped: config did not load` (test).

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d4b (Walkthrough). A second live instance is operator-run.

**Must-know-before:**
- O5's side-effect policy decides which probe runs. S12 and S16 both have side effects in the current code (`lock.go:34, :42`).

---

## m5-d5 — Persist `/approvals` clear

**Scope:** Clearing grants with `/approvals` must be recorded in the session log, so that a cleared grant does not come back on `kirsch resume`.

**Items covered:**
- D3 — `internal/app/app.go:528–529` (`App.ClearGrants` clears the policy only); `cmd/kirsch/main.go:274–277` (wiring); `app.go:992` (`installSession` clears at `/new`); `internal/session/grants.go:25–66` (`LoadGrants` replays every `approval.scope_granted`); `app.go:1077–1096` (resume replay). Resume ignores unknown types (`internal/session/resume.go:161–163`), so a new event is additive. ADR 0006 (`plan/adr/0006-session-scoped-approvals.md:45`) promises grants are "visible and revocable"; `:67–69` names a permissive restore as a security bug.

**Dependencies:** O11 (event name and §7 amendment; recorded first).

**File ownership:** `internal/session/event.go`, `internal/session/store.go`, `internal/session/grants.go`, `internal/app/app.go`, `plan/spec/kirsch-plan.md` §7 (`:340–357`, docs step).

**Parallel execution:** Wave 1 (see *Parallel waves*). m5-d17a follows this one.

**Mission steps:**
1. `@hicks` — code: the event type, `ClearGrants`, `LoadGrants`, and the tests.
2. `@apone` — review of step 1.
3. `@lambert` — docs: the §7 amendment (`plan/spec/kirsch-plan.md:340–357`) listing the event the code writes.
4. `@apone` — review of step 3.

**Instruction items:**
1. Record O11 first: the event name, the §7 amendment, and whether `/new` records anything new (it does not, unless O11 says otherwise).
2. Add the event type in `internal/session/event.go`.
3. Make `ClearGrants` (`app.go:528–529`) record the event through the store. The store is lazily created under `App.storeMu` (`internal/app/app.go:812`), so take that lock the way the existing code does.
4. Make `LoadGrants` (`grants.go:25–66`) reset its list when it reads the new event.
5. Leave `/new` (`app.go:992`) as it is, unless O11 says otherwise. A new session file needs no clear event.

**Acceptance:**
- [ ] A round-trip test grants a command, clears the grants, reopens the session log, and asserts that no grants are returned.
- [ ] A clear writes the new event into the session log (test reads the file).
- [ ] `LoadGrants` returns the grants before the clear event and none after it (test).
- [ ] A session log with no clear event restores its grants as before (backward-compatibility test).
- [ ] The §7 amendment lists the new event type (step 3).
- [ ] `go test -race ./internal/session/ ./internal/app/` passes in full, with no exclusion. `TestRunTurnApprovalToolUsesApprovalPath` is included: its race is RESOLVED (`plan/PROGRESS.md:180`), so a failure there is a returning race, not a known one.
- [ ] (operator) Grant with `a`, clear with `/approvals`, quit, run `kirsch resume`, run `/approvals`: no grants listed.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d5 (Walkthrough).

**Must-know-before:**
- O11 is the blocker.
- ADR 0006 (`plan/adr/0006-session-scoped-approvals.md:45, :67–69`) makes revocability a requirement. A permissive restore is named there as a security bug. This is the reason the item is not optional.
- D3 is a product defect, not only a documentation gap (`.claude/memory/missions/mission-20261023-05/DEBRIEF.md:37`).
- `TestRunTurnApprovalToolUsesApprovalPath` had a race, now fixed. The test reads the approval ID from the observed `ApprovalRequestedMsg` (`internal/app/turn_adapter_test.go:371–393`). `plan/PROGRESS.md:180` and `CHANGELOG.md:278` record it as RESOLVED. Do not exclude it from the `-race` run.

---

## m5-d6 — Composer paste path and caret rows

**Scope:** The composer's paste path is tested end to end, and the composer shows the caret row when the input is taller than five rows (O10).

**Items covered:**
- S1 — paste path: `internal/tui/update.go:707–709` routes `k.Paste` to `handlePaste`. `update.go:1113–1133` inserts through `composer.go:61–63` (`InsertString`). `update.go:1124–1129` raises `ConfirmLargePaste` above `styles.go:27` (`PasteWarnBytes`, 8 KB); the accept arm is at `update.go:1140–1141`. A multi-line text starting `/` is not a command (`composer.go:351–353`). The existing tests cover only the oversize prompt (`internal/tui/view_smoke_test.go:888–896`) and card-mode drops (`internal/tui/pending_card_test.go:683, :722`).
- X4 — the composer shows only its first rows. `contentLines` clamps to 1–5 (`composer.go:100–106`). `rows` appends logical lines until the height is reached and cuts the rest (`composer.go:138–143, :199–205, :212`). A second clamp sits in the layout at `internal/tui/view.go:461` (`l.ComposerH = clamp(composerLines, 1, 5)`). ui-spec §2:72 and `doc/usage.md:73–75` say the composer scrolls internally.

**Dependencies:** O10 (recorded first); m5-d1 (file ownership: `view.go`, only if O10 is wrap). m5-d7a, m5-d7b and m5-d8 follow this one.

**File ownership:** `internal/tui/composer.go`, `internal/tui/update.go`, `internal/tui/composer_test.go`, `internal/tui/paste_path_test.go` (new). `internal/tui/view.go` only if O10 is wrap.

**Parallel execution:** Wave 2, whatever O10 decides. It sits after m5-d1 so the `view.go` conditional cannot collide with m5-d1 (wave 1) or m5-d7a (wave 3). `view_smoke_test.go` is not touched here; m5-d9 owns it.

**Mission steps:**
1. `@hicks` — code: the paste tests, the O10 implementation, and the clamp changes O10 requires.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O10: scroll to the caret, or wrap.
2. Add a test in `internal/tui/paste_path_test.go` that an ordinary multi-line paste with indentation lands in the composer verbatim.
3. Add tests that accept and decline the large-paste prompt (`update.go:1124–1129`; accept arm `:1140–1141`).
4. Add a test that a multi-line paste starting with `/` is not treated as a command (`composer.go:351–353`).
5. Implement O10 for input taller than five rows. The current cut is at `composer.go:138–143, :199–205, :212`. Relax the clamp at `composer.go:100–106` if O10 requires it. Change `view.go:461` only if O10 is wrap, and then record it.
6. Add a test with a 12-line paste where the caret row is on screen. `composer_test.go` has no such test (X4).

**Acceptance:**
- [ ] A multi-line paste with indentation lands verbatim (test).
- [ ] A paste over 8 KB shows the y/n prompt; accept and decline are both tested.
- [ ] A pasted multi-line text starting with `/` is not a command (test).
- [ ] A 12-line paste leaves the caret row visible (test).
- [ ] Existing composer tests pass.
- [ ] (operator) Paste a 20-line trace in Terminal.app and iTerm2: newlines kept, nothing acts as a key.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d6 (Walkthrough). The real-terminal check is operator-owned.

**Must-know-before:**
- O10 is the blocker.
- The real-terminal part of S1 is operator-owned.
- Indentation in the transcript is S2, handled in m5-d7a. Do not fix it here.

---

## m5-d7a — Soft-wrap fidelity: `wrap()`

**Scope:** The wrap function keeps leading whitespace and interior spacing.

**Items covered:**
- S2 (wrap part) — `wrap` (`internal/tui/view.go:165–197`) serves user and assistant text (`internal/tui/cards.go:112–116, :123–132`), notices (`cards.go:335, :348`) and onboarding (`internal/tui/onboarding.go:52–77`). It splits with `strings.Fields` (`view.go:176`), which collapses leading indentation and runs of spaces. No test calls `wrap(`. ui-spec §3.1:227 and §3.2:232–234 say indentation is preserved.

**Dependencies:** O23 (golden handling, recorded first); m5-d1 (both touch `view.go`) and m5-d6 (`composer.go`, and `update.go` order).

**File ownership:** `internal/tui/view.go`, `internal/tui/cards.go`, `internal/tui/onboarding.go` (only if its output moves), `internal/tui/wrap_test.go` (new), and `plan/spec/kirsch-ui-screens.md` (the fences that move).

**Parallel execution:** Wave 3 (see *Parallel waves*). m5-d7b follows this one.

**Mission steps:**
1. `@hicks` — code: `wrap()`, its callers, the direct tests, and the goldens that move.
2. `@apone` — review of step 1.

**Instruction items:**
1. Change `wrap` (`view.go:165–197`) so it keeps leading whitespace and interior spacing. Hard-break only words wider than the frame. Replace the `strings.Fields` split at `view.go:176`.
2. Check each caller: `cards.go:112–116, :123–132, :335, :348` and `onboarding.go:52–77`. Record each caller whose output moves.
3. Add tests in `internal/tui/wrap_test.go` that call `wrap(` directly.
4. Change the golden fences that move, as O23 records, by the procedure in *Golden mechanism and regeneration procedure*. Save the pre-change copy of `plan/spec/kirsch-ui-screens.md` first, and record which fences moved.

**Acceptance:**
- [ ] A pasted 200-column indented trace keeps its indentation in the transcript (test).
- [ ] Runs of interior spaces survive in the transcript (test).
- [ ] `wrap` has direct tests in `internal/tui/wrap_test.go`, and they pass.
- [ ] Every golden that moved is listed in the step report.
- [ ] (operator) A human reviews the golden diff before the step closes.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d7a (Walkthrough).

**Must-know-before:**
- m5-d1 and m5-d6 are complete.
- The spec's "verbatim, soft-wrapped" and "preserve indentation exactly" (ui-spec §3.1:227, §3.2:232–234) are the reference. The code disagrees.
- Goldens move. A human review is part of the acceptance, not optional.

---

## m5-d7b — Soft-wrap fidelity: composer cut marker

**Scope:** The composer no longer cuts a row without a marker.

**Items covered:**
- S2 (composer part) — the composer does not wrap (`internal/tui/composer.go:87–88`). `rows` (`composer.go:138–205`) emits each logical line whole. `exactly` (`internal/tui/view.go:602–613`) cuts at the frame edge (`:610`) with no marker. ui-spec §7.3:565–571.

**Dependencies:** m5-d6 (after it), m5-d7a (both touch `view.go`), O17 (the marker choice), O23 (golden handling).

**File ownership:** `internal/tui/composer.go`, `internal/tui/view.go` (`exactly`), and `plan/spec/kirsch-ui-screens.md` (the fences that move).

**Parallel execution:** Wave 4 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: the cut marker or wrap per O17, its test, and the goldens that move.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O17: the cut marker, or wrap.
2. Make each composer row that is cut show the marker O17 records, at the cut in `exactly` (`view.go:602–613`, `:610`).
3. Add a test at 40 columns that asserts no cut row lacks the marker.
4. Change the golden fences that move, as O23 records, by the procedure in *Golden mechanism and regeneration procedure*. Record which fences moved.

**Acceptance:**
- [ ] At 40 columns no composer row is cut without the marker O17 records (test).
- [ ] Every golden that moved is listed in the step report.
- [ ] (operator) A human reviews the golden diff before the step closes.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d7b (Walkthrough).

**Must-know-before:**
- O17 is the blocker.
- ui-spec §7.3:565–571 is the reference for the marker.

---

## m5-d8 — Paste debounce

**Scope:** Decide the paste debounce (O9). Either implement the 20 ms coalescing, or amend the spec and delete the unused constant.

**Items covered:**
- S5 — `internal/tui/styles.go:22` (`PasteDebounce = 20 * time.Millisecond`), with no other reference. `github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118` already delivers bracketed paste as one message. ui-spec §7.2:561 and §11:753 name the debounce.

**Dependencies:** m5-d6 (the paste path), O9 (recorded first).

**File ownership:** `internal/tui/update.go`, `internal/tui/model.go`, `internal/tui/styles.go`, and `plan/spec/ui-spec-v0.1.md` §7.2 and §11 on the amend branch.

**Parallel execution:** Wave 3 (see *Parallel waves*). m5-d17b follows this one (shared `update.go`). The amend branch's edit to `plan/spec/ui-spec-v0.1.md` is why m5-d12 sits in wave 5.

**Mission steps:**
- Implement branch: 1. `@hicks` — coalescing and its test. 2. `@apone` — review of step 1.
- Amend branch: 1. `@hicks` — delete `PasteDebounce` at `styles.go:22` and its references. 2. `@apone` — review of step 1. 3. `@lambert` — ui-spec §7.2:561 and §11:753. 4. `@apone` — review of step 3.

**Before any edit.** Save verbatim copies of `internal/tui/styles.go` to `.claude/memory/workspace/m5-d8-styles.pre.go` and of `plan/spec/ui-spec-v0.1.md` to `.claude/memory/workspace/m5-d8-ui-spec.pre.md`. The acceptance box compares against them.

**Instruction items:**
1. Record O9 first.
2. Implement branch: coalesce back-to-back `KeyRunes` that arrive within the window into one insert, using the constant at `styles.go:22`. The debounce applies only when the terminal does not send bracketed paste.
3. Amend branch: delete `PasteDebounce` at `styles.go:22`. Amend the two lines of `plan/spec/ui-spec-v0.1.md` that contain `Paste debounce`, one in §7.2 and one in §11, to match. Leave the "Approval card appearance" row alone. It reads "Never debounced" and is unrelated.
4. Add a test for whichever branch was chosen.

**Acceptance:**
- [ ] O9 is recorded in `plan/PROGRESS.md`.
- [ ] Implement branch: a test with runes arriving within 20 ms inserts once. Amend branch: `grep -rn PasteDebounce internal/` prints nothing, and `grep -n 'Paste debounce' plan/spec/ui-spec-v0.1.md` prints nothing. The pattern is case-sensitive on purpose. `grep -n -i debounce` also matches the unrelated "Never debounced" row. Controls, run on the saved copies before the edit:

  ```bash
  PRE=/Volumes/DATA/Github/kirsch/.claude/memory/workspace
  grep -n PasteDebounce "$PRE/m5-d8-styles.pre.go"
  grep -n 'Paste debounce' "$PRE/m5-d8-ui-spec.pre.md"
  ```

  The first prints one line, the declaration, in the form `<n>:<text>`. The second prints two lines, both containing `Paste debounce`: one in §7.2 and one in §11. Neither prints the "Never debounced" row.
- [ ] With bracketed paste off, a 5 KB paste lands whole (test).

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d8 (Walkthrough): bracketed paste off, paste 5 KB, and the UI stays responsive (operator).

**Must-know-before:**
- O9 is the blocker.
- Bracketed paste is already one message in the normal case (`github.com/charmbracelet/bubbletea@v1.3.10/key_sequences.go:106–118`). The debounce matters only without it.

---

## m5-d9 — Terminal capability and resize tests

**Scope:** Tests for `DetectCaps` (no-colour and ASCII fallback), and an optional check that the resize invariants hold with a modal open.

**Items covered:**
- S3 — zero-width resize safety is already present: `computeLayout` returns early for `w<=0 ‖ h<=0` (`internal/tui/view.go:434–436`); `View` returns `""` (`view.go:554–558`); `wrap` guards `w<=0` (`view.go:168–171`). `TestFrameInvariants` sweeps widths and heights down to 0 and −1 (`internal/tui/view_smoke_test.go:32–55`). Nothing is required; the modal sweep is optional.
- S4 — `DetectCaps` (`internal/tui/styles.go:44–60`): colour is off for `NO_COLOR` set (tested with `os.LookupEnv`, `styles.go:48`), for `TERM` equal to `dumb` or empty, and for stdout that is not a character device. Locale is in `unicodeLocale` (`styles.go:62–70`). It is called once at `cmd/kirsch/main.go:104`. No test calls it (`internal/tui/composer_test.go:434` is a comment only).

**Dependencies:** O18 (recorded first).

**File ownership:** `internal/tui/styles_test.go` (new), `internal/tui/view_smoke_test.go` (optional), and `internal/tui/styles.go` only if O18 changes code.

**Parallel execution:** Wave 1 (see *Parallel waves*). Disjoint files. m5-d8 follows (`styles.go`).

**Mission steps:**
1. `@hicks` — code: `styles_test.go`, and `styles.go` only if O18 changes behaviour.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O18 first.
2. Write `internal/tui/styles_test.go`. For the cases that set environment variables, use `t.Setenv`. For the non-terminal case, use a pipe.
3. Cover these cases:
   - `NO_COLOR` set to a non-empty value: colour off.
   - Positive control: `DetectCaps` takes an `*os.File` (`internal/tui/styles.go:46`) and reads its argument, not the global `os.Stdout`. Open `/dev/null` with `os.Open` and pass that file directly, with `TERM=xterm` and `NO_COLOR` unset: colour on. The control must return colour on, or the negative cases prove nothing.
   - `NO_COLOR` empty, set with `t.Setenv("NO_COLOR", "")`: this sets the variable to empty and `LookupEnv` reports it set, so the case pins O18's behaviour.
   - `NO_COLOR` truly unset: `os.Unsetenv("NO_COLOR")`, with a `t.Cleanup` that restores the prior value. `t.Setenv` cannot express unset.
   - `TERM=dumb` and `TERM` empty: colour off.
   - Stdout that is not a character device (a pipe): colour off.
   - `LANG=C`: ASCII glyphs. First clear the higher-precedence variables with `t.Setenv("LC_ALL", "")` and `t.Setenv("LC_CTYPE", "")`. `unicodeLocale` (`internal/tui/styles.go:62–70`) reads `LC_ALL`, then `LC_CTYPE`, then `LANG`, and uses the first non-empty value, so a host `LC_ALL=en_US.UTF-8` would otherwise shadow `LANG=C`. Positive control for this case: `LANG=en_US.UTF-8` with the same two cleared gives the Unicode glyphs.
4. Record the O18 decision in the brief and pin it with the empty-`NO_COLOR` case.
5. Optional: run `TestFrameInvariants` (`view_smoke_test.go:32–55`) with a modal and a confirm prompt open.

**Acceptance:**
- [ ] `internal/tui/styles_test.go` covers each case in instruction 3, and the positive control returns colour on.
- [ ] The truly-unset case passes, and the test asserts that the prior `NO_COLOR` value is restored.
- [ ] The O18 behaviour is recorded in the brief and pinned by the empty-`NO_COLOR` case.
- [ ] (optional) The invariants sweep also runs with a modal and a confirm prompt open.
- [ ] (operator) A real resize with a modal open.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d9 (Walkthrough).

**Must-know-before:**
- The `NO_COLOR` decision (O18) is not in the operator's §7 list and must be recorded in the brief.
- The no-colour fallback is already live (`cmd/kirsch/main.go:104`). The step adds tests, not behaviour, unless O18 changes it.

---

## m5-d10a — Build wrapper and version proof

**Scope:** A local build wrapper that injects the bare version, and `package.json` pointed at it.

**Items covered:**
- S21 — `scripts/` has no `build.sh`. `package.json:9` `build` is `go build -o bin/kirsch ./cmd/kirsch` with no ldflags.

**Dependencies:** m5-d2 (the ldflags variable), O4 (recorded first).

**File ownership:** `scripts/build.sh` (new), `package.json`.

**Parallel execution:** Wave 2 (see *Parallel waves*). m5-d11, m5-d12 and m5-d13c follow.

**Mission steps:**
1. `@hicks` — code: `scripts/build.sh`, and the `package.json` build line.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O4 first.
2. Write `scripts/build.sh`: builds `./cmd/kirsch` with `-X main.version=<bare>` in ldflags. The version comes from the first argument, or from `git describe` with any leading `v` removed. Output is `bin/kirsch`.
3. Point `package.json:9` (`build`) at `bash scripts/build.sh`.
4. Prove the version with a known value.

**Acceptance:**
- [ ] `bash scripts/build.sh 9.9.9 && bin/kirsch --version` prints `9.9.9`, with no leading `v`.
- [ ] `npm run build` runs `scripts/build.sh`.
- [ ] With no argument, the version falls back the way O4 records.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d10a (Walkthrough).

**Must-know-before:**
- O4 is the blocker.

---

## m5-d10b — goreleaser config and target-compile check

**Scope:** The goreleaser configuration and a CI check that fails when a target does not compile.

**Items covered:**
- S20 — goreleaser config absent (no `.goreleaser*`). `.gitignore:10` already ignores `/dist/`. `cmd/kirsch-probe` is a second `main` and must not ship. There is no cgo (no `import "C"`).
- S22 — the targets are listed at `plan/spec/kirsch-plan.md:481` and `:487`. Nothing cross-builds today. Compilation is unverified until the check runs. The POSIX calls exist on both OSes (`internal/tool/run_command.go:183, :353–369`; `internal/session/lock.go:50`), and `lock_other.go` covers non-unix. ADR 0001:33 names the cross-compile targets.

**Dependencies:** m5-d2 (the ldflags variable), O2, O4, O20 (all recorded first).

**File ownership:** `.goreleaser.yaml` (new), `scripts/ci.sh`, `.github/workflows/ci.yml`.

**Parallel execution:** Wave 2 (see *Parallel waves*). m5-d11, m5-d12 and m5-d13c follow.

**Mission steps:**
1. `@hicks` — code: the goreleaser config, the CI target check per O20, and the forced-failure proof.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O2, O4 and O20 first.
2. Write `.goreleaser.yaml`: builds `./cmd/kirsch` only; `CGO_ENABLED=0`; ldflags `-X main.version={{.Version}}` (bare); the targets listed at `plan/spec/kirsch-plan.md:481`; archives; checksums; the changelog source O2 chooses.
3. Add a build of every listed target to `scripts/ci.sh` and `.github/workflows/ci.yml`, using the method O20 records. The check must fail when any target does not compile.
4. Prove that each listed target compiles locally with the chosen method. Record each target's result in the step report.
5. Force one target to fail, confirm the CI check exits non-zero, then revert the forced failure and record both results.

**Acceptance:**
- [ ] `.goreleaser.yaml` builds only `./cmd/kirsch`, with `CGO_ENABLED=0` and `-X main.version={{.Version}}`.
- [ ] A forced failure in one target makes the CI check exit non-zero. The forced failure is reverted and recorded.
- [ ] Every target listed at `plan/spec/kirsch-plan.md:481` compiles. The step report lists each target and its result.
- [ ] (operator) `goreleaser release --snapshot --clean` produces `dist/` with archives and a checksums file, and the native binary reports its version.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d10b (Walkthrough). The snapshot run is operator-owned.

**Must-know-before:**
- O2, O4 and O20 are the blockers.
- The target compile is unverified. Do not claim it from reading.
- Only `./cmd/kirsch` ships. `cmd/kirsch-probe` must not.

---

## m5-d11 — GitHub release workflow

**Scope:** A release workflow that runs on a version tag and builds the archives through goreleaser.

**Items covered:**
- S23 — `.github/workflows/` holds only `ci.yml` (jobs lint, test, security, screens; `permissions: contents: read` at `:9–10`; actions pinned by tag at `:20, :25`, as `actions/checkout@v4` and `actions/setup-go@v5`).
- No actionlint-style check exists in this repository. `grep -rn actionlint .github scripts package.json .claude` prints nothing (run 2026-10-23). This step adds none; O3 decides whether one is wanted.

**Dependencies:** m5-d10b (goreleaser config), O3 (recorded first).

**File ownership:** `.github/workflows/release.yml` (new).

**Parallel execution:** Wave 3 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: `release.yml`, per O3.
2. `@apone` — review of step 1.

**Instruction items:**
1. Record O3 first: trigger, permissions, pinning, and whether a manual dispatch exists for dry runs.
2. Trigger on `v*` tags. Add a dispatch trigger only if O3 says so.
3. Check out with full history. Set up Go from `go.mod`. Run the goreleaser action.
4. Set `permissions: contents: write`, scoped as O3 says.
5. Pin actions the way `ci.yml` does (by tag, `:20, :25`), unless O3 picks otherwise.

**Acceptance:**
- [ ] `release.yml` triggers on `v*` tags, and on dispatch only if O3 says so (grep the `on:` block).
- [ ] The write permission is scoped as O3 records (grep `permissions:`).
- [ ] Action pinning matches O3.
- [ ] (operator) A dispatch run, or a throwaway pre-release tag on a fork, produces a release with archives and checksums.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d11 (Walkthrough). The live run is operator-owned.

**Must-know-before:**
- O3 is the blocker.
- Repository Actions settings must allow write. That is an operator setting.
- The first real tag run is operator-owned.

---

## m5-d12 — Repository files and golden-regeneration note

**Scope:** `CONTRIBUTING.md`, `SECURITY.md`, and `NOTICE`. The golden-regeneration instruction in the spec is corrected to match the code.

**Items covered:**
- S24 — `CONTRIBUTING.md` absent. The build-time rules live in `plan/process/working-agreement.md` and `plan/process/code-standards.md`, which `README.md:62–64` says go away after v0.1.0.
- S25 — `SECURITY.md` absent. The limits below must be stated, each traced to code: `run_command` strips env names ending `_TOKEN`, `_KEY` or `_SECRET`, and names prefixed `AWS_`, by case-sensitive match (`internal/tool/run_command.go:395–401`); and a 400 writes the full request body to the debug log (`internal/provider/anthropic/client.go:453`, C24).
- S26 — `NOTICE` absent. The named libraries are Bubble Tea, Lip Gloss and Bubbles (`go.mod:7–9`). The other direct requires are at `go.mod:6, :10–15`. Licence: `LICENSE` (MIT).
- C37 — `plan/spec/kirsch-plan.md:498` says `go test ./internal/tui -update`, and `plan/spec/ui-spec-v0.1.md:824` says the goldens are regenerated with `-update`. The code gates regeneration on `KIRSCH_REGEN=1` (`internal/tui/regen_test.go:13`), and a person reconciles the grids by hand (see *Golden mechanism and regeneration procedure*). O21 decides whether the spec is corrected or a flag is added.

**Dependencies:** m5-d10a and m5-d10b (build commands), O7 and O8 (`SECURITY.md` and `NOTICE`), O21 (C37). All recorded first. m5-d8 (file ownership: `plan/spec/ui-spec-v0.1.md`, amend branch only).

**File ownership:** `CONTRIBUTING.md` (new), `SECURITY.md` (new), `NOTICE` (new), `plan/spec/kirsch-plan.md:498` and `plan/spec/ui-spec-v0.1.md:824` (docs step).

**Parallel execution:** Wave 5 (see *Parallel waves*). It sits in wave 5 because m5-d8's amend branch also edits `plan/spec/ui-spec-v0.1.md`. m5-d15 (wave 4) edits a different line of `kirsch-plan.md`.

**Mission steps:**
1. `@lambert` — docs: `CONTRIBUTING.md`, `SECURITY.md`, `NOTICE`, and the corrections at `plan/spec/kirsch-plan.md:498` and `plan/spec/ui-spec-v0.1.md:824` as O21 records.
2. `@apone` — review of step 1, including each `SECURITY.md` claim against its cited line.

**Instruction items:**
1. Record O7, O8 and O21.
2. `CONTRIBUTING.md`: the build, test and check commands (`npm run check`, `scripts/ci.sh`); how goldens are regenerated, following *Golden mechanism and regeneration procedure* (the `KIRSCH_REGEN=1` dump, the hand reconciliation of `plan/spec/kirsch-ui-screens.md`, `go test ./internal/tui` and `npm run screens`) (C37); the project's non-negotiables (`plan/spec/kirsch-plan.md:23–30`); pull-request expectations. Do not link to `plan/process/` files, which go away after v0.1.0 (`README.md:62–64`).
3. `SECURITY.md`: supported versions and the reporting channel (O7), a one-page threat model, and the limits in S25. Read it against `plan/testing/security.md`. Every claim must trace to a code location or a test. The 400 sentence states today's behaviour, that a 400 writes the full request body, because that is the behaviour until O15 decides C24. m5-d21c or m5-d21d amends that sentence.
4. `NOTICE`: credit Bubble Tea, Lip Gloss and Bubbles. List further direct dependencies only as O8 says.
5. Unless O21 chose to implement `-update`, correct the sentence in `plan/spec/kirsch-plan.md:498` that names `go test ./internal/tui -update`, and the sentence in `plan/spec/ui-spec-v0.1.md:824` that says the goldens are regenerated with `-update`. Each corrected sentence must describe *Golden mechanism and regeneration procedure*. Record which branch was taken. Before the edit, save verbatim copies of both spec files to `.claude/memory/workspace/m5-d12-kirsch-plan.pre.md` and `.claude/memory/workspace/m5-d12-ui-spec.pre.md`. The acceptance box uses them as its control.

**Acceptance:**
- [ ] (operator) Each command in `CONTRIBUTING.md` is run once on a clean clone and exits 0.
- [ ] `CONTRIBUTING.md` names `KIRSCH_REGEN=1` (`grep -n 'KIRSCH_REGEN=1' CONTRIBUTING.md` prints a line). `grep -n -e '-update' CONTRIBUTING.md plan/spec/kirsch-plan.md plan/spec/ui-spec-v0.1.md` prints nothing. Control, run on each saved copy before the edit: `grep -n -e '-update'` prints the line that contains `go test ./internal/tui -update` in the kirsch-plan copy, and the line that contains `regenerated with` and `-update` in the ui-spec copy. Each printed line is in the form `<n>:<text>`.
- [ ] If O21 chose `-update` instead, this box is replaced by two checks: `grep -n -e '-update' internal/tui/regen_test.go` prints the flag definition, and the two spec sentences name the flag as O21 records.
- [ ] `SECURITY.md` names the supported versions and the reporting channel O7 records.
- [ ] Every claim in `SECURITY.md` traces to a code location or a test. The review step checks each one.
- [ ] `NOTICE` credits Bubble Tea, Lip Gloss and Bubbles, and lists further direct dependencies only as O8 records. Its list matches the direct requires in `go.mod`.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d12 (Walkthrough).

**Must-know-before:**
- O7, O8 and O21 are the blockers.
- The limits in `SECURITY.md` state the current behaviour. C24 is an operator decision (O15), and `SECURITY.md` states the behaviour until the operator decides.
- The reporting channel and response promise are operator choices. Do not invent them.

---

## m5-d13a — `doc/configuration.md`

**Scope:** `doc/configuration.md`, written for a user, not a builder. It lists every §5 config key with its default and valid values.

**Items covered:**
- S28 — `doc/configuration.md` absent (`doc/README.md:14`). Keys are in `internal/config/config.go:27–118`. Precedence and the project allowlist are at `plan/spec/kirsch-plan.md:253–255`. `KIRSCH_DEBUG` is at `cmd/kirsch/main.go:88`. The config path is at `internal/config/config.go:665–681`. Defaults come from `Defaults()` (`config.go:84–118`).
- X6 — the §5 command-line flags precedence layer is unwired. `ApplyFlags` (`internal/config/config.go:496–506`) is called by `Load` only when `Options.Flags` is set (`:291`). `cmd/kirsch/main.go:71–75` passes no flags. This page does not promise config flags. Wiring them is a backlog item (see *Backlog*, X6).

**Dependencies:** O6 (recorded first).

**File ownership:** `doc/configuration.md` (new).

**Parallel execution:** Wave 1 (see *Parallel waves*). m5-d13b follows.

**Mission steps:**
1. `@lambert` — docs: `doc/configuration.md`.
2. `@apone` — review of step 1 against `internal/config/config.go`.

**Instruction items:**
1. Record O6 first.
2. Every key, with its default and valid values, its scope (global or project), and what it does. Derive the key list from the struct tags (`config.go:57–80`) and from `UnmarshalTOML` for endpoint fields (`config.go:219–259`).
3. Include `KIRSCH_DEBUG` (`cmd/kirsch/main.go:88`) and the config path (`config.go:665–681`).
4. Note that the project file may set only `context.project_files` (`config.go:355`, `plan/spec/kirsch-plan.md:255`).
5. Note that API keys are read from the environment variables named by `api_key_env`, never from config (`config.go:521–532`).
6. Make no promise of command-line config flags (X6). State it outright, in one sentence, that v0.1 reads no configuration from command-line flags.
7. Do not use the words "milestone" or "ADR".

**Acceptance:**
- [ ] Every key in `config.go:57–80` and every endpoint field in `config.go:219–259` is listed, with its default and valid values. The m5-d13b test checks this mechanically.
- [ ] `grep -n -i 'flag' doc/configuration.md` prints at least one line (an empty result would mean the grep did not read the page), and every line it prints is a sentence that promises no config flag. The output is pasted.
- [ ] `grep -n -i -w -E 'milestone|ADR' doc/configuration.md` prints nothing. Control: `printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'` prints `an ADR` and not `headroom`. The word boundary keeps words such as `headroom` out of the match.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d13a (Walkthrough).

**Must-know-before:**
- O6 is the blocker.
- The spec requires "every config key in §5" (`plan/spec/kirsch-plan.md:483, :487`).

---

## m5-d13b — Key-coverage test (S36)

**Scope:** A mechanical check that every §5 config key appears in `doc/configuration.md`.

**Items covered:**
- S36 — no check exists. Keys are in the struct tags (`internal/config/config.go:57–80`) and in `UnmarshalTOML` for endpoint fields (`config.go:219–259`).

**Dependencies:** m5-d13a (the page the test reads).

**File ownership:** `internal/config/key_coverage_test.go` (new).

**Parallel execution:** Wave 3 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: the key-coverage test.
2. `@apone` — review of step 1.

**Instruction items:**
1. Derive the key list from the struct tags (`config.go:57–80`) and from `UnmarshalTOML` for endpoint fields (`config.go:219–259`).
2. Read `doc/configuration.md`. Resolve the path from the module root, found by walking up from the test's working directory to `go.mod`. Fail for any key missing from the page.
3. Control: the test must fail when a key is removed from the page. Demonstrate it once, revert the removal, and record both in the step report.

**Acceptance:**
- [ ] The test passes against the current page. The output is pasted into the step report.
- [ ] Removing one key from `doc/configuration.md` makes the test fail. The removal is demonstrated once and reverted, and the step report records it.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d13b (Walkthrough).

**Must-know-before:**
- m5-d13a is complete.

---

## m5-d13c — `doc/install.md`

**Scope:** `doc/install.md`, written for a user. Install from a release archive, install with `go install`, install from source, the supported platforms, key setup, and `kirsch doctor`.

**Items covered:**
- S27 — `doc/install.md` absent (`doc/README.md:13`). The module path is at `go.mod:1`. A `go install` build reports `0.1.0-dev` (`cmd/kirsch/main.go:27–28`) unless O4 changes that. Supported platforms are at `plan/spec/kirsch-plan.md:21`.

**Dependencies:** m5-d2 (the `--version` text), m5-d10a and m5-d10b (archive names and build wrapper), O4 (the `go install` version text), O6 (recorded first).

**File ownership:** `doc/install.md` (new).

**Parallel execution:** Wave 3 (see *Parallel waves*).

**Mission steps:**
1. `@lambert` — docs: `doc/install.md`.
2. `@apone` — review of step 1 against the code and the release names.

**Instruction items:**
1. Record O4 and O6 first.
2. Cover: install from a release archive (names from m5-d10b); `go install github.com/djm56/kirsch/cmd/kirsch@…`, which reports the version O4 records; install from source (`scripts/build.sh`, from m5-d10a); supported platforms (`plan/spec/kirsch-plan.md:21`); key setup (the `api_key_env` variables named in `internal/config/config.go:91` and `:96`); and `kirsch doctor`.
3. No ADRs and no milestone language.

**Acceptance:**
- [ ] `doc/install.md` states what a `go install` build reports, as O4 records it.
- [ ] `grep -n -i -w -E 'milestone|ADR' doc/install.md` prints nothing. Control: `printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'` prints `an ADR` and not `headroom`.
- [ ] (operator) `doc/install.md` is followed on a clean machine or container, and it works.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d13c (Walkthrough). The clean-machine run is operator-owned.

**Must-know-before:**
- O4 and O6 are the blockers.
- `go install` builds report `0.1.0-dev` (`cmd/kirsch/main.go:27–28`) unless O4 changes it.

---

## m5-d14a — `doc/usage.md` corrections

**Scope:** Correct `doc/usage.md` against the code, remove its debug-command section and its milestone wording, and audit that the page covers key bindings, slash commands and the approval model.

**Items covered:**
- D1 — `doc/usage.md:257–270` documents debug commands (`/read`, `/ls`, `/search`, `/gitstatus`, `/gitdiff`). `SlashCommands` (`internal/tui/composer.go:372–374`) has none of them. The milestone wording is at `doc/usage.md:259` ("Milestone 1 added five commands").
- D2 — the status-bar mock at `doc/usage.md:54` omits cache. The code prints `in %s out %s cache %s/%s` (`internal/tui/statusbar.go:225`). The narrow-band wording at `:84–89` depends on m5-d1.
- D4 — `doc/usage.md:247` says nothing about `/files` after resume. `ResumeLastSession` restores no touched files (`internal/app/app.go:1006–1070`). `CHANGELOG.md:17` says "a resumed session starts empty". If C54 is built, D4 changes.
- D6 — `doc/usage.md:299–302` says every action waits on a card. The exact-match allowlist and session grants let some commands run without one (`README.md:105–108`; `internal/policy/policy_test.go:620–660`).
- D7 — `doc/usage.md:22–26` ("Starting it") omits `kirsch resume` and `--new`. Both exist at `cmd/kirsch/main.go:46–48, :56`.
- X1 — `doc/usage.md:69–71` claims a missing-`rg` warning in the status bar. The warning is listed at ui-spec §2.3:168. The step checks the `.go` files for an emitter before removing the claim.

**Dependencies:** m5-d1 (narrow bar, for D2 wording), m5-d5 (grant clearing, for D6 wording), m5-d6 (composer text, with O10), m5-d7a (wrap), O6 and O10 (recorded first).

**File ownership:** `doc/usage.md`.

**Parallel execution:** Wave 4 (see *Parallel waves*). m5-d14b follows.

**Mission steps:**
1. `@lambert` — docs: the corrections below, in `doc/usage.md`.
2. `@apone` — review of step 1 against the code and the audit.

**Before any edit.** Save a verbatim copy of `doc/usage.md` to `.claude/memory/workspace/m5-d14a-usage.pre.md`. The controls below run on that copy.

**Instruction items:**
1. Audit first: confirm that `doc/usage.md` has a section for key bindings, one for slash commands and one for the approval model. Record each heading in the step report.
2. Delete the section headed `### Debug commands — temporary` (`doc/usage.md:257–270`), including its table, whose first row is `| \`/read <path>\` | \`read_file\` |`.
3. Status-bar mock at `:54`: show cache read and write, matching `internal/tui/statusbar.go:225`. Update the narrow-band wording at `:84–89` to match what m5-d1 landed.
4. The `/files` row at `:247`: add that a resumed session starts empty, unless C54 has been built.
5. Replace the sentence that reads "Nothing consequential happens silently. Before Kirsch writes a file or runs a command, the action appears as a card and waits:". The replacement names the exact-match allowlist and session grants as running without a card, and says that other writes and commands appear as a card and wait (`README.md:105–108`).
6. "Starting it" at `:22–26`: add `kirsch resume` and `kirsch --new`, from `cmd/kirsch/main.go:46–48, :56`.
7. Remove the missing-`rg` claim at `:69–71`, after the `.go` files are checked for an emitter. Removing the claim is enough for v0.1. Implementing the warning is not in this step.
8. The milestone sentence at `:259` is inside the section deleted in instruction 2, so it goes with it.
9. The composer paragraph at `doc/usage.md:73–75` says the composer grows to five rows "then scrolling internally". Reword it to the behaviour O10 records: scroll to the caret, or wrap.

**Acceptance:**
- [ ] The audit names a section covering key bindings, one covering slash commands, and one covering the approval model.
- [ ] The status-bar mock at `doc/usage.md:54` shows cache read and write.
- [ ] The `/files` row at `doc/usage.md:247` says a resumed session starts empty, or C54 has been built and D4 is removed.
- [ ] The section headed `### Debug commands — temporary` is gone, and so is its table, whose first row is `| \`/read <path>\` | \`read_file\` |`. `grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff' doc/usage.md` prints nothing. Control, same pattern: `printf '/read <path>\n' | grep -n '/read <path>\|/ls \[path\]\|/search <query>\|/gitstatus\|/gitdiff'` prints `1:/read <path>`.
- [ ] "Starting it" names `kirsch resume` and `kirsch --new`.
- [ ] The sentence "Nothing consequential happens silently. Before Kirsch writes a file or runs a command, the action appears as a card and waits:" is gone. `grep -n 'Nothing consequential happens silently' doc/usage.md` prints nothing. Control: the same grep on the pre-change copy prints one line, `301:Nothing consequential happens silently. Before Kirsch writes a file or runs a`, in the form `grep -n` gives for a single file. The replacement names the allowlist and session grants.
- [ ] The composer paragraph at `doc/usage.md:73–75` describes the behaviour O10 records.
- [ ] `doc/usage.md` makes no claim of a missing-`rg` warning: `grep -n 'missing .rg' doc/usage.md` prints nothing. Control: the same grep on the pre-change copy prints the line that contains ``missing `rg` `` in the form `<n>:<text>`.
- [ ] `grep -n -i -w -E 'milestone|ADR' doc/usage.md` prints nothing. Control: `printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'` prints `an ADR` and not `headroom`.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d14a (Walkthrough).

**Must-know-before:**
- m5-d1, m5-d5, m5-d6 and m5-d7a are complete. Their outcomes change what the page must say.
- D4's wording changes if C54 is built. C54 is in the backlog, so this step documents the limit.
- D6's wording about grants must match the D3 fix in m5-d5.

---

## m5-d14b — `doc/troubleshooting.md` and `doc/README.md`

**Scope:** Write `doc/troubleshooting.md`, which explains every doctor line and the common failures. Update the index in `doc/README.md`, and remove its milestone and ADR wording.

**Items covered:**
- S30 — `doc/troubleshooting.md` absent (`doc/README.md:16`). Material comes from the code: the no-key text (`internal/app/app.go:1298–1324`), the not-a-repo text (`internal/workspace/workspace.go:71–74`), the 401/403 text (`internal/provider/anthropic/client.go:447`, C23), the missing-`rg` fallback (`internal/tool/searchcode.go:118–122`), and the lock warning.
- C23 (doc remedy) — every 401 or 403 reads "rejected the API key" (`client.go:447`). The page explains it. The code change stays in the backlog.
- S29 (README part) — `doc/README.md:3` ("**Mostly written at Milestone 5**") and `doc/README.md:19` ("v0.1 spec, architecture, UI spec, ADRs, and the per-milestone instruction sets.") carry milestone and ADR wording.
- Index status lines: `doc/README.md:13` (install, now written by m5-d13c), `:14` (configuration, now written by m5-d13a), `:16` (troubleshooting).

**Dependencies:** m5-d3a, m5-d3b, m5-d4a and m5-d4b (the doctor lines the page explains), m5-d13a and m5-d13c (the pages the index names), O6 (recorded first).

**File ownership:** `doc/troubleshooting.md` (new), `doc/README.md`.

**Parallel execution:** Wave 6 (see *Parallel waves*).

**Mission steps:**
1. `@lambert` — docs: `doc/troubleshooting.md`, then `doc/README.md`.
2. `@apone` — review of step 1 against the doctor lines in the code.

**Instruction items:**
1. Write `doc/troubleshooting.md`: each doctor line that m5-d3a, m5-d3b, m5-d4a and m5-d4b emit, what it means, and what to do. Quote each line verbatim from the code (DIR-005).
2. Cover the common failures in S30: no key (quote the onboarding text from `internal/tui/onboarding.go:51`), not in a repository (`workspace.go:71–74`), 401 or 403 "rejected the API key" (`client.go:447`), the missing-`rg` fallback, and the lock held warning.
3. In `doc/README.md`, make the status lines at `:13`, `:14` and `:16` match the files as they now exist.
4. In `doc/README.md`, remove the milestone wording at `:3` and the words "ADRs, and the per-milestone instruction sets" at `:19`.
5. Run `grep -n -i -w -E 'milestone|ADR' doc/troubleshooting.md doc/README.md`. It must print nothing, and the control in the acceptance box must print its line.

**Acceptance:**
- [ ] Every doctor line that m5-d3a, m5-d3b, m5-d4a and m5-d4b emit is explained in `doc/troubleshooting.md`. The step report lists the emitted lines and the page section for each.
- [ ] The status lines at `doc/README.md:13`, `:14` and `:16` match the files that exist.
- [ ] `grep -n -i -w -E 'milestone|ADR' doc/troubleshooting.md doc/README.md` prints nothing. Control: `printf 'an ADR\nheadroom\n' | grep -i -w -E 'milestone|ADR'` prints `an ADR` and not `headroom`.
- [ ] (operator) Each failure the troubleshooting page covers is reproduced, and the advice fixes it.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d14b (Walkthrough).

**Must-know-before:**
- m5-d4b is complete, so the doctor lines are final.

---

## m5-d15 — Front-door and tracker freshness

**Scope:** The README front door, the CHANGELOG and PROGRESS claims that the code contradicts, the ADR range, and the closed carried rows in `plan/PROGRESS.md`.

**Items covered:**
- S31 — the README has no install section. The banner and "What works today" start from `go run ./cmd/kirsch` (`README.md:5–7, :99–101`).
- D5 — "0001–0007" appears at `plan/spec/kirsch-plan.md:391`, `plan/README.md:25` and `AGENTS.md:21`. Each states the range as a live fact, and each is out of date, because `plan/adr/` holds eight ADRs. At `plan/adr/README.md:64` ("ADRs 0001–0007 were written during planning") the sentence may be true as written, so the executor judges it (DIR-004). `CHANGELOG.md:21` is a dated history line and stays. `0008-provider-endpoints.md` exists (`plan/adr/README.md:60`). The ADR status note at `plan/adr/README.md:64` is the operator's decision.
- D8 — `README.md:142–143` says "every script is one line of `go` or `bash`". `package.json:35` runs `python3 scripts/lint-screens.py` as `screens`.
- X2 — `CHANGELOG.md:16` ("with fallback `N tok` on narrow terminals") and `plan/PROGRESS.md:231` ("narrow terminal fallback `N tok`") claim a narrow fallback. Corrected after m5-d1.
- X3 — `README.md:47` reads "| Providers | One only for v0.1 (Anthropic) |". `plan/spec/kirsch-plan.md:17` and ADR 0008 describe the configured endpoints (`internal/config/config.go:91, :96`).
- Closed rows to mark Resolved in `plan/PROGRESS.md`:
  - C21 — `plan/PROGRESS.md:129`. `max_turns_exceeded` is mapped at `internal/app/app.go:1999–2000, :2011`, rendered and tested (`internal/tui/turn_messages_test.go:103–106`; `internal/app/turn_adapter_test.go:297`).
  - C32 — `plan/PROGRESS.md:151`. `policy.New(cfg…)` reads the settings (`internal/app/app.go:500`).
  - C33 — `plan/PROGRESS.md:152`. The command timeout is read at `internal/tool/run_command.go:144–151`; amendment 85.
  - C39 — `plan/PROGRESS.md:158`. Closed for the named comments (`internal/policy/policy.go:57–58`; `internal/tool/run_command.go:334–341, :355–357`). Checked 2026-10-23: `internal/config/config.go:44` reads "EndpointConfig is one configured Messages-format endpoint (ADR 0008)." It makes no policy-wiring claim, so it does not block C39.
  - C42 — `plan/PROGRESS.md:172`. `internal/session/usage.go:81–85` carries `CostUSD` and `CostUnknownTokens`.

**Dependencies:** m5-d1 (X2 wording), m5-d10a and m5-d10b (release tooling the README refers to), m5-d13c (the install page the README links to), O22 (recorded first).

**File ownership:** `README.md`, `CHANGELOG.md`, `plan/PROGRESS.md`, `plan/spec/kirsch-plan.md:391`, `plan/adr/README.md`, `plan/README.md:25`.

**Parallel execution:** Wave 4 (see *Parallel waves*).

**Mission steps:**
1. `@lambert` — docs: the edits below, across the files named in its row.
2. `@apone` — review of step 1 against the code and the cited lines.

**Before any edit.** Save verbatim copies to `.claude/memory/workspace/`: `plan/spec/kirsch-plan.md` as `m5-d15-kirsch-plan.pre.md`, `plan/adr/README.md` as `m5-d15-adr-README.pre.md`, `plan/README.md` as `m5-d15-plan-README.pre.md`, `AGENTS.md` as `m5-d15-AGENTS.pre.md` and `README.md` as `m5-d15-README.pre.md`. The controls in the acceptance boxes run on these copies.

**Instruction items:**
1. Add an install section to `README.md` that links `doc/install.md`. Keep the banner's `go run` line only as a developer path (`README.md:5–7, :99–101`).
2. Judge each site of "0001–0007" (DIR-004), and record the judgement for each site in the step report. Change `plan/spec/kirsch-plan.md:391`, `plan/README.md:25` and `AGENTS.md:21` to the range that matches `ls plan/adr/`. At `plan/adr/README.md:64`, edit only if "ADRs 0001–0007 were written during planning" is false as written; a true sentence stays. Leave `CHANGELOG.md:21` alone, because it is history. Leave the ADR status wording alone (the operator's call).
3. Reword `README.md:142–143` to match `package.json:35`: name the Python script that `npm run screens` runs, and drop "every script is one line of `go` or `bash`".
4. Correct `CHANGELOG.md:16` and `plan/PROGRESS.md:231` to describe the narrow-width behaviour that m5-d1 built. Per DIR-020, the corrected sentence goes beside the original, with the verdict beside it, and the original is kept.
5. Correct `README.md:47` to state the configured endpoints, in the wording O22 records. Read it with the out-of-scope bullet at `README.md:53–58`.
6. Mark C21, C32, C33 and C42 Resolved in `plan/PROGRESS.md`. C39 may be marked, because `internal/config/config.go:44` was checked on 2026-10-23 (see the scope section). Before editing `plan/PROGRESS.md`, save a verbatim copy to the workspace (DIR-020). Mark each row Resolved beside its original text, and do not replace that text.

**Acceptance:**
- [ ] `README.md` has an install section that links `doc/install.md`.
- [ ] Every line that `grep -n '0001–0007'` prints in `plan/spec/kirsch-plan.md`, `plan/adr/README.md`, `plan/README.md` and `AGENTS.md` is a site that the step report records as kept, with its reason (DIR-004). Repository-wide, `grep -rln '0001–0007' --include='*.md' . | grep -v '\.claude/memory/' | grep -v 'plan/milestones/'` lists only files whose sites the report records. The exclusions match path text, because BSD `grep -r` prints paths without a leading `./`. The workspace and the milestone instruction sets are excluded, because they quote the range as history. Control: the same grep on each saved copy prints one line in the form `<n>:<text>` for each of the four files.
- [ ] `grep -n 'lint-screens.py' README.md` prints a line, and `grep -n 'every script is one line' README.md` prints nothing. Control: the second grep on `.claude/memory/workspace/m5-d15-README.pre.md` prints one line in the form `<n>:<text>`.
- [ ] `CHANGELOG.md:16` and `plan/PROGRESS.md:231` describe the narrow-width behaviour that m5-d1 built.
- [ ] `README.md:47` states the configured endpoints, in the wording O22 records.
- [ ] C21, C32, C33 and C42 are marked Resolved in `plan/PROGRESS.md`. C39 is marked Resolved with the 2026-10-23 check of `internal/config/config.go:44` cited beside it.
- [ ] `plan/adr/README.md:64` status wording is unchanged. `grep -c 'Accepted (planning phase' plan/adr/README.md` prints the same count as the same command on `.claude/memory/workspace/m5-d15-adr-README.pre.md`.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d15 (Walkthrough).

**Must-know-before:**
- m5-d1 must have landed before the X2 wording is written.
- The README banner and "What works today" are updated at release, in m5-d22. This step does not touch them.
- ADR status wording is the operator's (`plan/adr/README.md:64`).

---

## m5-d16 — Fresh-machine no-key check

**Scope:** A no-key run on a clean environment, driven through a pseudo-terminal or the equivalent (O19), showing that onboarding names the active endpoint's key pair. It sits beside `scripts/smoke-test.sh`, which cannot host it (O14).

**Items covered:**
- S35 — onboarding for a missing key exists (`CheckOnboarding`, `internal/app/app.go:1298–1324`; the screen landed at m3-d5). `KeyVars` is set at `app.go:1313` as `"KIRSCH_" + ep.APIKeyEnv` and `ep.APIKeyEnv`. The onboarding text renders at `internal/tui/onboarding.go:51` ("No API key: set … → …"). `scripts/smoke-test.sh:16–22` refuses to start without a key, so a no-key flow cannot live in it as written.
- X7 — `cmd/kirsch/main.go:62–65, :34` prints `kirsch: Kirsch runs inside a Git repository…` and exits 1. ui-spec §7.5:597 lists it among onboarding screens.

**Dependencies:** O14 and O19 (both recorded first); m5-d10a (builds the binary the script runs); m5-d2 and m5-d3a (file ownership: `cmd/kirsch/main.go`, only if O14 makes "not in a repo" a screen). This deliverable does not depend on m5-d4: `CheckOnboarding` does not use doctor.

**File ownership:** a new script in `scripts/` (name in the brief). `cmd/kirsch/main.go` only if O14 makes "not in a repo" a screen.

**Parallel execution:** Wave 3 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: the no-key script, and the X7 screen only if O14 requires it.
2. `@apone` — review of step 1.

**Before any edit.** Save `scripts/smoke-test.sh` verbatim to `.claude/memory/workspace/m5-d16-smoke-test.pre.sh`. The acceptance box compares against it.

**Instruction items:**
1. Record O14 and O19 first: whether "not in a repo" is included and where the check lives, and how the interactive TUI is driven.
2. Do not add a no-key run to `scripts/smoke-test.sh`. It exits 2 without a key (`:16–22`), and that behaviour stays.
3. Write the no-key check as a separate script. The command shape is `env -i HOME=<temporary directory> PATH=$PATH TERM=xterm-256color <binary>`, where `<binary>` is `bin/kirsch` built by `bash scripts/build.sh` (m5-d10a). The script runs it inside a temporary Git repository the script creates and removes.
4. Drive the TUI by the method O19 records.
5. Check that the onboarding text names the active endpoint's key pair: `KIRSCH_<api_key_env>` and `<api_key_env>`, from `app.go:1313`, rendered at `internal/tui/onboarding.go:51`. With the default endpoint (`internal/config/config.go:87, :91`) the pair is `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY`. The check asserts that pair.
6. If O14 includes "not in a repo", decide whether it becomes a screen (`cmd/kirsch/main.go:62–65`), and record the decision.

**Acceptance:**
- [ ] A no-key run in a clean environment shows onboarding that names `KIRSCH_OPENCODE_API_KEY` and `OPENCODE_API_KEY`. The output is pasted into the step report.
- [ ] The check runs in a temporary Git repository, which it creates and removes.
- [ ] `scripts/smoke-test.sh` is unchanged against its saved copy: `cmp -s .claude/memory/workspace/m5-d16-smoke-test.pre.sh scripts/smoke-test.sh` exits 0. `git ls-files --error-unmatch -- scripts/smoke-test.sh` prints the path, so the comparison is against a tracked file. Control: the same `cmp` against a copy with one byte appended exits 1.
- [ ] `scripts/smoke-test.sh` still exits 2 without a key. Run it with all four key variables unset. With any one of them set, the script runs `TestSmokeLive` against the live provider and spends tokens:

  ```bash
  env -u ANTHROPIC_API_KEY -u KIRSCH_ANTHROPIC_API_KEY -u OPENCODE_API_KEY -u KIRSCH_OPENCODE_API_KEY bash scripts/smoke-test.sh; echo "exit=$?"
  ```

  The printed `exit=` line is pasted.
- [ ] The "not in a repo" case is handled as O14 records.
- [ ] (operator) The check is run on a clean machine.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d16 (Walkthrough).

**Must-know-before:**
- O14 and O19 are the blockers.
- `scripts/smoke-test.sh` is the milestone-4 smoke test. Keep its checks as they are.

---

## m5-d17a (optional) — `/new` lifecycle: C51, C52

**Scope:** The carried `/new` items: the orphan empty session file, and the provider session id across `/new`.

**Items covered:**
- C51 — quit during `/new` can leave an orphan empty session file. `NewSession` creates the file first (`internal/app/app.go:947`, `createSessionStore`), then cancels and swaps (`:954–957`). `plan/PROGRESS.md:204`.
- C52 — the provider session header id persists across `/new`. `SetSessionID` is called once (`cmd/kirsch/main.go:199`). `NewSession` and `installSession` (`internal/app/app.go:943–997`) never touch `a.sessionID`. `plan/PROGRESS.md:205`.

**Dependencies:** m5-d5 (the session layer), O13 (recorded first).

**File ownership:** `internal/app/app.go`.

**Parallel execution:** Wave 3 (see *Parallel waves*). m5-d4a (wave 4) and m5-d17b (wave 5) follow it because both touch `app.go`.

**Mission steps:**
1. `@hicks` — code: the lazy file creation or cleanup, and the session-id rotation, with tests.
2. `@apone` — review of step 1.

**Instruction items:**
1. C51: create the session file lazily, or remove the empty file on cancel (`app.go:947, :954–957`). Choose one, and record it.
2. C52: rotate the provider session id on `/new` (`app.go:943–997`). Today `SetSessionID` is called once, at `cmd/kirsch/main.go:199`.

**Acceptance:**
- [ ] A test shows that quitting during `/new` leaves no empty session file.
- [ ] A test shows that after `NewSession` the provider session id differs from the id before it.
- [ ] `go test ./internal/app/` passes.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d17a (Walkthrough).

**Must-know-before:**
- Optional. Scheduled only if O13 says so.
- m5-d5 is complete.

---

## m5-d17b (optional) — `/new` busy state and turn fencing: C48, C49

**Scope:** The busy state while `/new` is pending, and late async messages from a cancelled turn.

**Items covered:**
- C48 — while `/new` is pending the TUI can clear busy and show a stray error card. `internal/tui/update.go:1152–1163` sets busy and `newSessionPending`. `plan/PROGRESS.md:201`.
- C49 — late async messages from the cancelled turn are not fenced by session generation (`internal/app/app.go`, `internal/tui/`). `plan/PROGRESS.md:202`.

**Dependencies:** m5-d8 (shared `update.go`), m5-d17a (shared `app.go`), O13 (recorded first).

**File ownership:** `internal/tui/update.go`. `internal/app/app.go` for the generation counter, after m5-d17a lands.

**Parallel execution:** Wave 5 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — reproduce C48 and C49 first, then fix and test.
2. `@apone` — review of step 1.

**Instruction items:**
1. Reproduce C48 and C49 on the current code before any change. Record each reproduction in the step report: the command or test run, and its output.
2. C48: fix the busy state while `/new` is pending (`internal/tui/update.go:1152–1163`). Add a test that pressing Enter during `/new` shows no error card. Record that the test fails on the pre-fix code.
3. C49: add a generation counter in `internal/app/app.go` so that late messages from a cancelled turn are dropped. Add a test that a message carrying the cancelled turn's generation is dropped.

**Acceptance:**
- [ ] A test shows that pressing Enter while `/new` is pending shows no error card.
- [ ] A test shows that a message carrying a cancelled turn's generation is dropped.
- [ ] The reproduction for C48 and for C49 is in the step report, with the command and output pasted.
- [ ] The C48 test fails on the pre-fix code, and the failing output is pasted.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d17b (Walkthrough).

**Must-know-before:**
- Optional. Scheduled only if O13 says so.
- C48 and C49 have not been re-derived against the code. Reproduce them first; the reproduction is the evidence.

---

## m5-d18 (optional) — Resume totals and compaction %

**Scope:** The status bar shows the session's totals straight after resume. `/status` updates the compaction estimate after a manual `/compact`.

**Items covered:**
- C5 — `ResumeLastSession` zeroes the totals (`internal/app/app.go:1058–1063`) and sends only `NoticeMsg` and `TranscriptLoadedMsg` (`:1066–1069`). `usage.json` is workspace-wide, not per session (`internal/session/layout.go:236`; `internal/session/usage.go:69`), so it cannot seed the totals. `usage` events are in the session log (`internal/session/event.go:315`). `plan/PROGRESS.md:99`.
- C50 — `/status` compaction % is stale after a manual `/compact`. `plan/PROGRESS.md:203`.

**Dependencies:** m5-d17a (file ownership only: both own `internal/app/app.go`). If O13 defers the optional group, this dependency falls away. O13 (recorded first).

**File ownership:** `internal/app/app.go`, `internal/session/` (a new loader file beside `grants.go`).

**Parallel execution:** Wave 6 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: the resume loader, the compaction update, and their tests.
2. `@apone` — review of step 1.

**Instruction items:**
1. At resume, sum the session's `usage` events and send `SessionUsageMsg` (C5). The loader sits beside `internal/session/grants.go`.
2. Stop relying on `usage.json` for per-session totals. It is workspace-wide.
3. After a manual `/compact`, update the compaction estimate that `/status` shows (C50).
4. Compaction check, with values: a fixture session whose `/status` compaction % reads 60% before `/compact`. After `/compact`, the test asserts that `/status` shows the value the compaction returns for that fixture. Record both values in the step report. The after-value must differ from 60%.

**Acceptance:**
- [ ] A test shows that after resume the status bar shows the session's totals before any new usage.
- [ ] A test shows that the totals equal the sum of the session's `usage` events in a fixture log.
- [ ] A test shows the compaction % before `/compact` is 60% and after it is the value `Compact` returns, which differs from 60%. Both values are pasted.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d18 (Walkthrough): run a turn, quit, `kirsch resume`; then `/compact` and `/status`.

**Must-know-before:**
- Optional. Scheduled only if O13 says so.
- m5-d17a is complete (file ownership).

---

## m5-d19a — Safety-test repair for `run_command`: C30, C35, C36

**Scope:** The `run_command` tests that cannot fail, the secret-name patterns whose lower-case behaviour is unpinned, and the blank approval card. None of this needs an operator decision.

**Items covered:**
- C30 — `TestRunCommandTimeoutProcessGroupDead`, `TestRunCommandCancellationProcessGroupDead` and `TestRunCommandStdinIsDevNull` in `internal/tool/run_command_test.go` (`:769`, `:832`, `:208`) cannot fail (`plan/PROGRESS.md:149`).
- C35 — secret-name patterns (`plan/PROGRESS.md:154`). `TestRunCommandAllStripPatterns` (`internal/tool/run_command_test.go:523–539`) already covers `MY_TOKEN`, `API_KEY`, `DB_SECRET` and `AWS_*`, with negative cases (`TOKEN_PREFIX`, `MY_TOKENIZE`, `SECRET_PASSWORD`). Only lower-case names are unpinned. The logic at `internal/tool/run_command.go:395–401` is a case-sensitive suffix and prefix test.
- C36 — blank approval card for an unspecified operation (`plan/PROGRESS.md:155`). Test what renders, and assert that the content is correct, not only that it matches a stored string.

**Dependencies:** O13 (the scheduling of the safety group). Nothing else.

**File ownership:** `internal/tool/run_command_test.go`; `internal/tui/approval_unspecified_test.go` (new). No non-test file changes.

**Parallel execution:** Wave 1 (see *Parallel waves*). Disjoint files. m5-d19b follows (same test file).

**Mission steps:**
1. `@hicks` — tests: the rewritten C30 tests, the lower-case C35 cases, and the C36 test.
2. `@apone` — review of step 1.

**Instruction items:**
1. C30: rewrite each named test so that a mutant breaking its property makes it fail. For each test, record the named mutant, the test that fails on it, and the failing output. Record the pass on the real code.
2. C35: pin that lower-case names are not stripped: `my_token`, `api_key` and `db_secret` pass through, because matching is case-sensitive (`run_command.go:395–401`). A change to case-insensitive matching would then fail this test and need an explicit decision. The `AWS_` prefix and the other patterns are already covered and need no change.
3. C36: add a test that records what renders for an unspecified operation on the blank approval card. The test asserts that the card shows the operation name the request carries and that no field is empty.

**Acceptance:**
- [ ] Each C30 test fails under its named mutant and passes on the real code. The mutant names and the failing output are pasted.
- [ ] The lower-case cases assert that `my_token`, `api_key` and `db_secret` are not stripped, and the suite passes.
- [ ] The C36 test asserts that the blank approval card for an unspecified operation shows the operation name and no empty field.
- [ ] `go test ./internal/tool/ ./internal/tui/` passes.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d19a (Walkthrough): run `run_command` with `MY_TOKEN` exported in the kirsch process environment and listed in `Policy.EnvPassthrough`. `buildEnvironment` reads it with `os.Getenv` (`run_command.go:386–393`). Set `LANG` in the environment too. The child sees `LANG` (always passed, `internal/tool/run_command.go:379–384`) and does not see `MY_TOKEN` (stripped even when listed, `run_command.go:386–401`). The `LANG` arrival is the control: an absent `MY_TOKEN` alone would also appear if nothing arrived at all.

**Must-know-before:**
- Mutant evidence is the test of these tests. A test that passes on a broken build is the defect this deliverable repairs.

---

## m5-d19b — Safety-test repair: C29, after O15

**Scope:** The 500 ms box for a process that ignores SIGTERM. The outcome follows O15's C29 decision.

**Items covered:**
- C29 — the 500 ms box cannot hold for a process that ignores SIGTERM. `run_command` sends SIGTERM and always waits 2 s before SIGKILL (`internal/tool/run_command.go:353–369`, comment `:355–357`). The unticked boxes are at `plan/milestones/milestone-2.md:418–419`. `plan/PROGRESS.md:148`.

**Dependencies:** O15 (the C29 decision, recorded before this step runs; m5-d20 records it, and this step runs in wave 2, ahead of the gate's other items); m5-d19a (same test file).

**File ownership:** `internal/tool/run_command.go`, `internal/tool/run_command_test.go`, `plan/milestones/milestone-2.md`.

**Parallel execution:** Wave 2 (see *Parallel waves*). m5-d19a is in wave 1.

**Mission steps:**
1. `@hicks` — code and test, for the tighten branch only. Skip if O15 relaxes.
2. `@apone` — review of step 1, if step 1 ran.
3. `@lambert` — the milestone-2 box at `plan/milestones/milestone-2.md:418–419`, for the relax branch only.
4. `@apone` — review of step 3, if step 3 ran.

**Instruction items:**
1. Record O15's C29 decision in `plan/PROGRESS.md` before any step runs.
2. If O15 tightens: change the code and test so that the 500 ms box holds for a process that ignores SIGTERM.
3. If O15 relaxes: change the box at `plan/milestones/milestone-2.md:418–419` to the value O15 records. Record which branch was taken. Before the edit, save a verbatim copy of `plan/milestones/milestone-2.md` to the workspace (DIR-020). Changed text starts unticked, and the verdict goes beside the old text, not in place of it.

**Acceptance (one box per O15 outcome):**
- [ ] If O15 tightens: a test with a child that ignores SIGTERM passes within 500 ms. The test output is pasted.
- [ ] If O15 relaxes: `plan/milestones/milestone-2.md:418–419` states the value O15 records, and no test asserts 500 ms for that case.
- [ ] O15's C29 decision is recorded in `plan/PROGRESS.md` before step 1 runs.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d19b (Walkthrough).

**Must-know-before:**
- O15's C29 decision is the blocker.

---

## m5-d20 (operator) — Record the operator's decisions

**Scope:** The decisions only the operator can make, recorded in `plan/PROGRESS.md`. No agent delegates this step. Its outcomes drive the m5-d21 deliverables.

**Items covered (decisions):**
- C8 (accept or change), C9 (cache-write prices against the vendor price sheets), C11 (accept, or re-check in m5-d22), C14 (confirm deleting the `.bak` files in m5-d22), C24 (clip, or disclose in `SECURITY.md`), C34 (the Task 9 box wording, which is the operator's), C38 (retest in m5-d22), C40 (keep the stdlib ULID, or add `oklog/ulid`), C43 (amend §7, or accept), C46 (keep the omission, or amend §7), C61 (tick in m5-d22), X5 (model table, after the vendor pages are read), X8 (footer, see O16).
- The open O-decisions: O12 (demo format and storage), O15 (C24, C34, C40, C43, C46 and C29), O16 (the help footer).

**Dependencies:** none. This step records O12, O15 and O16, and the decisions listed above.

**File ownership:** `plan/PROGRESS.md` only.

**Parallel execution:** A gate. It sits before the m5-d21 deliverables and may be done at any time before them, except that the C29 item of O15 must be recorded before m5-d19b runs (wave 2).

**Mission steps:** operator only. No agent delegates this step.

**Instruction items:**
1. Record one decision per ID in the list above, in `plan/PROGRESS.md`, with the option chosen.
2. Record O12, O15 and O16.
3. Record each outcome before the m5-d21 deliverable that depends on it runs.
4. If C24 is clip, record the byte bound with the choice. m5-d21c quotes that bound into its brief, and no other step defines it.

**Acceptance:**
- [ ] Each ID in the list above has a recorded decision in `plan/PROGRESS.md`.
- [ ] O12, O15 and O16 are recorded.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d20 (Walkthrough, operator-run).

**Must-know-before:**
- Operator-owned. No agent runs this deliverable.

---

## m5-d21 group (conditional) — Agent follow-through on recorded decisions

**Scope:** The agent work that follows from the m5-d20 decisions. The group is split into one deliverable per outcome, so each deliverable holds one code or docs step and its review. A deliverable runs only if its decision calls for it. Each deliverable takes the same sync and review rules as any other. C34 (the Task 9 box wording) is operator-owned and has no agent deliverable.

## m5-d21a (conditional) — Model table (X5, C9)

**Scope:** The model figures, dates and cache-write price, if the operator changes them under m5-d20.

**Items covered:**
- X5 — `internal/provider/models.go:33–36`; `plan/spec/kirsch-plan.md:270, :274`.
- C9 — `internal/provider/models.go:33–36` (cache-write prices).

**Dependencies:** m5-d20 (the recorded decision).

**File ownership:** `internal/provider/models.go` and its test.

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: figures, dates and cache-write price in `internal/provider/models.go`, and its test.
2. `@apone` — review of step 1.

**Instruction items:**
1. Run only if the operator changed the figures or dates under m5-d20. If not, record "no step" in `plan/PROGRESS.md`.
2. Quote the recorded decision into the brief.

**Acceptance:**
- [ ] The table matches the recorded decision, and `go test ./internal/provider/` passes.
- [ ] The review step is recorded in `plan/PROGRESS.md`.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21a (Walkthrough).

**Must-know-before:**
- m5-d20 is complete, and its X5 and C9 decisions are recorded.

## m5-d21b (conditional) — Help footer and goldens (X8, O16)

**Scope:** The help overlay footer, if the operator changes it under m5-d20, and the goldens that move with it.

**Items covered:**
- X8 — `internal/tui/modal.go:308`, the footer that says `docs: doc/usage.md`.

**Dependencies:** m5-d20 (O16 recorded); O23 (golden handling, recorded first).

**File ownership:** `internal/tui/modal.go`; `plan/spec/kirsch-ui-screens.md` (the fences that move).

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: the footer at `internal/tui/modal.go:308`, then the golden fences that move, reconciled by the procedure in *Golden mechanism and regeneration procedure*, as O23 records. Record which fences moved.
2. `@apone` — review of step 1, including the golden diff. The operator also reviews the golden diff.

**Instruction items:**
1. Run only if the operator changed the footer under m5-d20. If not, record "no step".
2. Quote the O16 and O23 choices into the brief.

**Acceptance:**
- [ ] The footer matches O16, and every moved golden is listed in the step report.
- [ ] The review step covers the golden diff and is recorded.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21b (Walkthrough).

**Must-know-before:**
- O16 and O23 are recorded.

## m5-d21c (conditional) — Debug log clip (C24)

**Scope:** Clip the 400 debug log to the byte bound the operator records under m5-d20, if the operator chooses to clip. The alternative, disclosure, is m5-d21d.

**Items covered:**
- C24 — `internal/provider/anthropic/client.go:453` is the call site. `logBadRequest` (`client.go:506–512`) writes the request body (`:511`).

**Dependencies:** m5-d20 (the C24 decision).

**File ownership:** `internal/provider/anthropic/client.go`, `internal/provider/anthropic/client_test.go`, and the sentence in `SECURITY.md` that states the 400 log behaviour (the "full request body" wording m5-d12 wrote).

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: clip the request body that `logBadRequest` writes, and its test.
2. `@apone` — review of step 1.
3. `@lambert` — docs: the `SECURITY.md` sentence, as instruction 4 says.
4. `@apone` — review of step 3.

**Instruction items:**
1. Run only if the operator clipped under m5-d20. If not, record "no step".
2. Quote the recorded bound, in bytes, into the brief.
3. The test pairs presence with absence: the logged 400 body contains the first N bytes of the request, where N is the bound, and it does not contain the bytes after N.
4. Amend the `SECURITY.md` sentence that says a 400 writes the full request body, so that it states the bound quoted in instruction 2. Before the edit, save `SECURITY.md` to `.claude/memory/workspace/m5-d21c-SECURITY.pre.md`.

**Acceptance:**
- [ ] The 400 log writes at most the bound quoted in the brief. The test asserts presence of the first N bytes and absence of the bytes after N, and it passes.
- [ ] `grep -c 'full request body' SECURITY.md` prints 0. Control: the same grep on `.claude/memory/workspace/m5-d21c-SECURITY.pre.md` prints a count of at least 1.
- [ ] The review step is recorded.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21c (Walkthrough).

**Must-know-before:**
- The C24 decision is recorded, and the byte bound is recorded with it.

## m5-d21d (conditional) — Debug log disclosure (C24)

**Scope:** Disclose the 400 debug log in `SECURITY.md`, if the operator chooses disclosure under m5-d20 instead of clipping (m5-d21c).

**Items covered:**
- C24 — the disclosure line, traced to `internal/provider/anthropic/client.go:453` (call site) and `logBadRequest` (`client.go:506–512`).

**Dependencies:** m5-d20 (the C24 decision); m5-d12 (file ownership: `SECURITY.md`).

**File ownership:** `SECURITY.md`.

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@lambert` — docs: the disclosure line in `SECURITY.md`, traced to the code locations above.
2. `@apone` — review of step 1, checking the claim against the cited lines.

**Instruction items:**
1. Run only if the operator disclosed under m5-d20. If not, record "no step".

**Acceptance:**
- [ ] `SECURITY.md` states the 400 log behaviour, and each claim traces to a cited line.
- [ ] The sentence in `SECURITY.md` that says a 400 writes the full request body now says that this is a disclosed limit. `grep -n 'full request body' SECURITY.md` prints a line, and that line contains `disclosed`. On the file that m5-d12 leaves, the sentence has no `disclosed`, so this box fails until the update is made.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21d (Walkthrough).

**Must-know-before:**
- The C24 decision is recorded, and m5-d12 is complete.

## m5-d21e (conditional) — ULID dependency (C40)

**Scope:** Replace the stdlib ULID with `oklog/ulid`, only if the operator adds the dependency under m5-d20.

**Items covered:**
- C40 — `internal/session/layout.go:96–140`.

**Dependencies:** m5-d20 (the C40 decision).

**File ownership:** `internal/session/layout.go`, `go.mod`, `go.sum`.

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: replace the stdlib ULID at `internal/session/layout.go:96–140` and add the dependency.
2. `@apone` — review of step 1.

**Instruction items:**
1. Run only if the operator adds `oklog/ulid`. If the operator keeps the stdlib ULID, record "no step".

**Acceptance:**
- [ ] `go test ./internal/session/` passes, and the review step is recorded.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21e (Walkthrough).

**Must-know-before:**
- The C40 decision is recorded.

## m5-d21f (conditional) — §7 amendment (C43, C46)

**Scope:** Amend §7 of the spec, if the operator amends it under m5-d20.

**Items covered:**
- C43 — §7 flattening of interleaved text and tool calls (`plan/spec/kirsch-plan.md:340–357`).
- C46 — the session events `session.started` and `session.resumed` (`plan/spec/kirsch-plan.md:342–343`).

**Dependencies:** m5-d20 (the C43 and C46 decisions); m5-d5 (file ownership: `plan/spec/kirsch-plan.md` §7).

**File ownership:** `plan/spec/kirsch-plan.md` §7 (`:340–357`).

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@lambert` — docs: `plan/spec/kirsch-plan.md` §7 (`:340–357`), including the session events at `:342–343`.
2. `@apone` — review of step 1.

**Instruction items:**
1. Run only if the operator amends §7. If not, record "no step".

**Acceptance:**
- [ ] §7 matches the events the code writes, and the review step is recorded.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21f (Walkthrough).

**Must-know-before:**
- The C43 and C46 decisions are recorded, and m5-d5 is complete.

## m5-d21g (conditional) — Lock behaviour (C8)

**Scope:** Change the usage-lock behaviour, only if the operator changes it under m5-d20.

**Items covered:**
- C8 — `internal/session/usage.go:28, :152`, the lock timeout that drops an increment.

**Dependencies:** m5-d20 (the C8 decision).

**File ownership:** `internal/session/usage.go`, `internal/session/usage_test.go`.

**Parallel execution:** Wave 7 (see *Parallel waves*).

**Mission steps:**
1. `@hicks` — code: `internal/session/usage.go:28, :152`, and its test.
2. `@apone` — review of step 1.

**Instruction items:**
1. Run only if the operator changes the lock behaviour. If the operator accepts it, record "no step".

**Acceptance:**
- [ ] The lock behaviour matches the recorded decision, and `go test ./internal/session/` passes.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d21g (Walkthrough).

**Must-know-before:**
- The C8 decision is recorded.

---

## m5-d22 (operator) — Demo, release smoke run, tag

**Scope:** The steps only the operator can take: the demo, the operator-run smoke test on the release build, the tag, and the release-day tracker updates. **No agent delegates this deliverable.**

**Items covered:**
- S32 — no screenshots or asciinema demo. It needs a real terminal and is blocked by O12 (`plan/spec/kirsch-plan.md:484`).
- S33 — tag `v0.1.0`. `git tag -l` prints `m3` and `m4` (checked 2026-10-23). Tagging triggers the release workflow (m5-d11).
- S34 — re-run `bash scripts/smoke-test.sh` on the release build. It spends tokens. The live run passed on 2026-10-23 (`plan/PROGRESS.md:117`). Re-run at release.
- C9 — re-read the cache-write prices against the vendor pages before tagging, if m5-d21a did not already do it (`internal/provider/models.go:33–36`).
- C11 — re-check final-report survival in the release smoke run (`plan/PROGRESS.md:105`).
- C14 — delete the stray `.bak` files `opencode.json.bak` and `.claude/memory/state/FLIGHT-RECORDER.md.bak`.
- C38 — retest the walkthrough checks (`plan/testing/manual/user-testing/milestone-2-donovan.md`).
- C61 — tick the outstanding verification boxes (`plan/testing/README.md:97–101`).
- X5 — re-read the vendor pricing pages before tagging, if m5-d21a did not already do it.
- S31 (banner) — the README banner and "What works today" name v0.1.0.

**Dependencies:** the m5-d21 deliverables that ran; O12, O15 and O16 recorded. All deliverables above are complete.

**File ownership:** the repository, and GitHub settings.

**Parallel execution:** Wave 8, after everything else.

**Mission steps:** operator only. No agent delegates this step.

**Instruction items (operator):**
1. Make the demo (O12) on a real terminal. Store or link it, and embed or link it in `README.md`.
2. Run `bash scripts/smoke-test.sh` on the release build with a key. Read the transcript.
3. Delete the `.bak` files (C14).
4. Re-read the vendor price pages and confirm the model table (C9, X5).
5. Run the walkthrough retests (C38) and tick the outstanding verification boxes (C61).
6. Update the `README.md` banner and "What works today" to name v0.1.0 (S31).
7. Tag `v0.1.0` (S33). The tag triggers the release workflow (m5-d11).

**Acceptance:**
- [ ] O12, O15 and O16 are recorded in `plan/PROGRESS.md`.
- [ ] The demo is stored or linked, and `README.md` embeds or links it.
- [ ] `bash scripts/smoke-test.sh` on the release build prints PASS. The transcript is pasted.
- [ ] `git tag -l v0.1.0` prints `v0.1.0`, and the release page lists the artefacts.
- [ ] `ls opencode.json.bak .claude/memory/state/FLIGHT-RECORDER.md.bak` reports both files missing (`No such file or directory`). Control: `ls scripts/smoke-test.sh` lists its file, so the error is not a wrong path. The output is pasted.
- [ ] The model table and its dates match the vendor pages as read on the day of tagging.
- [ ] The C61 outstanding-verification boxes are ticked by the operator. The C38 retest is recorded.
- [ ] The `README.md` banner names v0.1.0.

**Manual test hook:** `plan/testing/manual/milestone-5.md` § m5-d22 (Walkthrough, operator-run).

**Must-know-before:**
- Operator-owned. No agent runs this deliverable.
- S34 spends tokens. Confirm the budget first.
- The release build must exist (m5-d10a, m5-d10b, m5-d11) before S34 and S33.

---

## Backlog

Recorded, not scheduled for v0.1. Each line is an ID, its title, and its source. The coverage table maps each to "backlog". The full item text is in `plan/PROGRESS.md` at the line given.

- **C2** — absence-check tests use fixed sleeps (`plan/PROGRESS.md:78`).
- **C3** — out-of-order `SessionUsageMsg` can leave stale totals (`:99`).
- **C4** — mid-session pricing change can understate or hide cost (`:99`).
- **C6** — `applyCompaction` error arm in `Compact` untested (`:99`).
- **C7** — rename-aside-then-write-failure consequence undocumented (`:99`).
- **C10** — race mutant for the resume test killed in 1 of 20 runs (`:99`).
- **C12** — "Limitations" pinned as a bare word (`:106`).
- **C13** — `system.md` mutants not independently rerun (`:107`; evidence only).
- **C15** — smoke test approval and TestDivide-baseline branches lack an offline negative control (`:114`).
- **C16** — comment in `smoke_live_test.go` overstates containment (`:115`).
- **C17** — zero-divisor check is a source regex (`:116`).
- **C18** — `stop_reason`, `tool`, `duration_ms` omitted from session events (`:126, :178, :216`).
- **C19** — other agent edge cases: empty user text, nil Model or Tools, synchronous `emit`, resume on an assistant `tool_use` (`:127`).
- **C20** — calibration warnings: EventError after MessageDone; predictions file edit (`:128`).
- **C22** — cancellation goroutine not joined (`:130`).
- **C23 (code change)** — every 401/403 reads "rejected the API key" (`:137`; `internal/provider/anthropic/client.go:447`). Only the code change is here. The documentation remedy is in m5-d14b.
- **C25** — config review items (`:139`).
- **C26** — wire-format review items (`:140`).
- **C27** — live-test review items (`:141`).
- **C28** — default-model review items (`:142`).
- **C31** — suspected trailing-newline defect in `internal/patch`, not yet reproduced (`:150`).
- **C41** — fuzz seeds with `..` for `FuzzSessionPathContained` (`:171`).
- **C44** — pending-map leniency (`:187, :188`).
- **C45** — per-event fuzz seeds; deep-copy comment on `cloneBlocks`/`cloneRaw` (`:188`).
- **C47** — no auto-resume happy-path test in `main_test.go` (`:195`).
- **C53** — `/diff` shows unstaged changes only (`:206`).
- **C54** — `/files` resets on resume (`:207`). If built, D4 changes (m5-d14a).
- **C55** — `update_test.go` and `status_test.go` overlap (`:208`).
- **C56** — flush-ordering edge, 500 ms window (`:214`).
- **C57** — 100 ms settle window (`:215`).
- **C58** — per-field mutant evidence shown for Signature only (`:217`; evidence only).
- **C59** — `held_lock_no_warning` fixed 200 ms settle (`:223`).
- **C60** — lock-unavailable fallback has no automated test (`:224`).
- **X6 (wiring)** — the §5 command-line flags precedence layer is unwired (`internal/config/config.go:496–506`; `cmd/kirsch/main.go:71–75`). Whether to wire it is a separate decision, not numbered here. m5-d13a documents no flags.

---

## Out of scope

Carried from the README's *Explicitly Out of Scope for v0.1* (`README.md:53–58`), verbatim:

> Multiple providers, `git commit`/`git push` tools, MCP, LSP, subagents, plugins, OS keychain, web UI, session branching/search, automatic compaction UI tuning. Do not build these early.

The spec's list at `plan/spec/kirsch-plan.md:32–34` differs at the provider item. It says "Multiple wire formats (OpenAI-format adapters)" where the README says "Multiple providers". The README list is carried here. The difference is a contradiction to settle with the operator (O22), at m5-d15.

The M4 out-of-scope items `kirsch doctor` and release tooling are Milestone 5 items (`plan/milestones/milestone-4.md`, *Explicitly Out of Scope (reminder)*).

---

## Note for whoever executes this

The code citations in this document were read against the code on 2026-10-23. No shell-based build or run was part of that read, so nothing here was compiled or run. Before a deliverable starts:

1. Re-read the deliverable against the current code. Check each `file:line` it cites still says what the section says. Code may have moved since this document was written.
2. Check the O-decisions named in its *Dependencies*. If one is still open, the deliverable waits.
3. Check the carried rows it covers in `plan/PROGRESS.md`. A row may have been closed by another mission.
4. Write the mission brief from this section. Keep the acceptance boxes as the brief's acceptance. Record any change in the brief, and say which claims you verified and which you took from this document.
5. Give every file the brief names as an absolute path, and name the repository's absolute path when more than one checkout exists (DIR-006).

Where this document and the code disagree, the code is right and this document is wrong. Fix the document in the step that changes it.
