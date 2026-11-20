# Kirsch — website ingest file

> Written 2026-11-18 from the Kirsch repository (`/Volumes/DATA/Github/kirsch`, GitHub `https://github.com/djm56/kirsch`). Part 1 is the brief. Part 2 is a verbatim snapshot of the user documentation as it stood in the working tree on 2026-11-18. If you can read the Kirsch repository, the files there win over the snapshot.

## 1. Your task

You are working in an Eleventy (11ty) website repository. Add Kirsch to it:

1. **A product page** for Kirsch (section 3 is the brief).
2. **A help section** of several pages built from Kirsch's user documentation (section 4 is the page plan, Part 2 is the source text).

Read this whole file before you change anything. Look at how the 11ty site already lays out product pages, collections, navigation, layouts and images, and follow it. Do not add a new static site generator or a new theme.

## 2. Rules

1. **Kirsch's `doc/` folder is the source of truth.** The help pages say what `doc/` says. Do not add features, flags, commands, providers, models, prices or platforms that the source text does not name. If something is missing, leave a visible `TODO` and list it in your report; do not fill it from general knowledge.
2. **Only describe released behaviour.** On 2026-11-18, **v0.1.0 is the only published release** (2026-11-17, https://github.com/djm56/kirsch/releases/tag/v0.1.0). **v0.1.1 is prepared but not yet tagged.** The snapshot in Part 2 already says "v0.1.1 is the current release" and pins `0.1.1` in commands. Before publishing, check the releases page. If v0.1.1 is not there, the site says v0.1.0. Better still, version commands use a placeholder (`VERSION=<version>`) and a line pointing at the releases page.
3. **No links into `plan/`.** `plan/` is the Kirsch build scaffolding and is not for users. Drop such links, or turn them into plain text.
4. **Rewrite links.** A relative link such as `configuration.md` or `configuration.md#provider` becomes the matching help-page route on this site. Absolute `https://github.com/djm56/kirsch...` links stay as they are. Check every internal link and anchor resolves after the build.
5. **Screenshots.** Use only the images listed in section 5. Never show an API key, a key's value, or a personal home path in any image or code sample.
6. **Roadmap is labelled as planned.** Anything in section 3.6 is "planned", never "available".
7. **Plain, direct language.** Short sentences. No hype.
8. **Licence:** MIT, © 2026 Donovan Maidens.

## 3. Product page brief

### 3.1 Name and one-line description
**Kirsch** — an open-source, terminal-native coding agent written in Go.

### 3.2 Short pitch
Kirsch is a full-screen terminal app that works inside a Git repository. It reads, lists, searches and diffs your code, proposes small patches you review and approve, and runs verification commands under a policy you control. Every session is saved on your own disk and can be resumed.

### 3.3 Core ideas (from README.md "Core Ideas")
- The agent is a partner, not an autoclicker: no file is written without an approved patch, and no command runs without a policy decision.
- The workspace is a boundary: nothing outside the repository root is touched.
- Everything is visible before it happens; every session is durable and resumable.
- Everything is cancellable.
- Small, reviewable patches (unified diffs you approve or reject).
- Tool output is data, never instructions.
- Local-first: sessions are JSONL files on local disk. No server, no telemetry, no cloud storage.

### 3.4 What v0.1.0 does (feature list)
- Investigate code: read, list, search files, and diff.
- Propose patches; every patch waits for your approval.
- Run commands under a policy: exact-match allowlist, session-scoped grants, or explicit approval. Shell commands always ask.
- Durable session log; `kirsch resume` or auto-resume on launch restores the transcript and session grants.
- Long conversations are compacted to fit the context window; `/compact` does it on demand.
- Status bar with session token counts and cost.
- Slash commands: `/help`, `/status`, `/diff`, `/files`, `/approvals`, `/new`, `/compact`, `/quit`, `/exit`.
- `kirsch doctor` checks the install without sending a prompt.
- A second Kirsch in the same workspace is detected and runs in a safe, degraded mode.
- A path denylist (`.env`, `*.pem`, `*.key`, `.git/**`, `.kirsch/**`) is always enforced.

### 3.5 Facts table
| Item | Value |
|---|---|
| Language | Go |
| Interface | Full-screen terminal UI (Bubble Tea) |
| Platforms | macOS and Linux, amd64 and arm64. No Windows. |
| Model endpoints (built in) | `opencode` (OpenCode Go, the default, model `minimax-m3`) and `anthropic` (model `claude-sonnet-5-5`). Both use the Anthropic Messages format. You can define your own endpoint. |
| API keys | Read from environment variables only (`OPENCODE_API_KEY`, `ANTHROPIC_API_KEY`, or their `KIRSCH_` forms). Never from a config file. |
| Storage | Local JSONL session files only |
| Licence | MIT |
| Source | https://github.com/djm56/kirsch |
| Releases | https://github.com/djm56/kirsch/releases |

### 3.6 Planned (label as "planned", with no dates)
Milestone release names from the Kirsch plan:
- v0.2.0 — More than one model
- v0.3.0 — More than one wire format
- v0.4.0 — Headless
- v0.5.0 — Front ends and machines
- v0.6.0 — Review and memory

Out of scope for v0.1: multiple providers at once, `git commit`/`git push` tools, MCP, LSP, subagents, plugins, OS keychain, web UI, session branching or search.

### 3.7 Calls to action
- "Get started" → the help section's quick start page.
- "View on GitHub" → https://github.com/djm56/kirsch
- "Download" → https://github.com/djm56/kirsch/releases

## 4. Help section — page plan

Kirsch's own docs plan (Diátaxis model) proposes these pages. Build them from the Part 2 source sections named in each row.

| Help page | Type | Built from (Part 2) |
|---|---|---|
| Overview (landing) | — | `doc/README.md` and the pitch in section 3 |
| Getting started (quick start) | Tutorial | `doc/install.md`: the release-archive download and verify steps, the API key step, then the first run from `doc/usage.md` "Starting it". Shortest path only. |
| Install | Reference/how-to | `doc/install.md`: platforms, Git and ripgrep, each install route, `kirsch doctor`. |
| Set up a provider and model | How-to | `doc/install.md` API key section; `doc/configuration.md` "Secrets" and `[provider]`. Explain the `opencode` default and the `anthropic` route, how to export the key, how to check with `kirsch doctor`. |
| Everyday tasks | How-to | `doc/usage.md`: resume a session, start a fresh one (`/new`), approve or reject a command, approve for the session, compact (`/compact`), read `/status`. |
| Using Kirsch | Reference | `doc/usage.md`: the screen, keys, slash commands, the approval model. |
| Configuration | Reference | `doc/configuration.md` |
| `kirsch doctor` checks | Reference | `doc/troubleshooting.md`, from "Reading the report" to the end of the per-check sections (before "Common failures"). |
| Troubleshooting | Problem → fix | `doc/troubleshooting.md` "Common failures" and the closing section. |
| Security | Reference | `SECURITY.md` |
| Changelog / release notes | — | Link to https://github.com/djm56/kirsch/releases rather than copying `CHANGELOG.md`. |

Text that is a screen of output (`kirsch --help`, `kirsch doctor`) stays a code block, not a picture.

## 5. Images

The images are in the Kirsch repository at `doc/images/` (local path `/Volumes/DATA/Github/kirsch/doc/images/`; on GitHub `https://github.com/djm56/kirsch/tree/main/doc/images`). Copy them into the 11ty site's own image folder.

| File | Use on | Alt text |
|---|---|---|
| `start-screen.png` | Product page hero; Getting started | Kirsch start screen showing the KIRSCH title, version line, provider notice and the typing line with its placeholder text |
| `help-overlay.png` | Using Kirsch, slash commands | Kirsch /help overlay listing the keys and slash commands |
| `approval-card.png` | **Do not use yet.** | The captured picture shows a garbled assistant line (`the requestedI'll run command.`). It is waiting for a retake. |

The two usable screenshots were taken on a pre-release build and show the version as `v0.1.0-dev`. A caption may say so.

## 6. How to keep the site in step with Kirsch

Pick one and say which in your report:
- **Snapshot (simplest):** copy the text in Part 2 into the 11ty content now. It will drift; re-run this ingest after each Kirsch release.
- **Pull at build (recommended if the build can reach GitHub):** at build time, fetch `doc/` from a pinned Kirsch release tag (for example the release tarball, or a sparse `git` checkout of `doc/` at the tag) into the content folder, then apply the link rules in section 2. Changing the pinned tag is then the only step per release.

## 7. What to report back

List: the pages created and their URLs; the images copied; every `TODO` left; every link you could not resolve; which option in section 6 you chose; and anything in Part 2 that contradicts this brief.

---

# Part 2 — Source snapshot (verbatim, 2026-11-18 working tree)

---

## SOURCE: README.md

# Kirsch

> **Status: v0.1.0 is the first release. v0.1.1 is the current release (v0.1.0 rebuilt with Go 1.26.11; see [`CHANGELOG.md`](CHANGELOG.md)). See the [v0.1.0 release page](https://github.com/djm56/kirsch/releases/tag/v0.1.0). While `plan/` remains, the items carried after the release are in plan/PROGRESS.md.** Kirsch runs real model turns against a repository: it reads, lists, searches and diffs the workspace through its tools, proposes small reviewable patches (every patch waits for your approval), and runs commands under a policy (exact-match allowlist, session-scoped grants, or explicit approval — shells always ask). Every turn is recorded to a durable local JSONL session log, and `kirsch resume` (or auto-resume on launch) restores the transcript and session-scoped grants. A second instance in the same workspace is detected and degraded gracefully (fresh start, status-bar warning, grants disabled). Long conversations are compacted to stay within the context window, and `/compact` does so on demand. The slash commands work on real session state, the status bar shows session token and cost totals, and the system prompt asks the agent to end each task that took actions with a final-task report. The authoritative build spec is [`plan/spec/kirsch-plan.md`](plan/spec/kirsch-plan.md); progress is tracked in [`plan/PROGRESS.md`](plan/PROGRESS.md).
>
> Developer path, from a clone of the repository (for a built binary, see [Install](#install)):
> ```
> go run ./cmd/kirsch
> ```

Kirsch is an open-source, **terminal-native coding agent written in Go**. Its
primary interface is a simple full-screen Bubble Tea TUI, in the spirit of
Claude Code / OpenCode / Pi.

It works inside a Git repository, where it can:

- Investigate code (read, list, search files)
- Propose and apply small, reviewable patches
- Run controlled verification commands (tests, builds, linters)
- Keep a durable, resumable local session record

**First user:** a solo developer working on real projects (WordPress/PHP, Go).

## Install

> **v0.1.0 is the first release.** [`doc/install.md`](doc/install.md) covers installing from a release archive on the [releases page](https://github.com/djm56/kirsch/releases), installing with `go install github.com/djm56/kirsch/cmd/kirsch@v0.1.1`, and building from a clone. v0.1.1 is the current release: it is v0.1.0 built with Go 1.26.11, which fixes the standard-library advisory GO-2026-6859. Upgrade from v0.1.0; see [`CHANGELOG.md`](CHANGELOG.md).

These are the commands for building from a clone, from the *Build from source* section of [`doc/install.md`](doc/install.md):

```bash
git clone https://github.com/djm56/kirsch.git
cd kirsch
bash scripts/build.sh
./bin/kirsch --version
```

You need Go and `git`. [`doc/install.md`](doc/install.md) gives the Go version. The binary is written to `bin/kirsch` inside the checkout. The same page covers what Kirsch needs on the machine, setting up an API key, and checking the install with `kirsch doctor`.

Developers working on Kirsch itself can run it from the clone without building a binary:

```bash
go run ./cmd/kirsch
```

## Core Ideas

1. **The agent is a partner, not an autoclicker.** It never writes to a file
   without an approved patch, and never runs a command without a policy
   decision (allowlist match, session grant, or explicit user approval).
2. **The workspace is a boundary.** Nothing outside the workspace root is
   ever touched — symlink escapes and `..` traversal are blocked outright.
3. **Everything is visible.** Every consequential action is shown in the TUI
   before it happens; every session is durable and resumable.
4. **Everything is cancellable.** No UI hang on a stuck tool or model call.
5. **Small, reviewable patches.** The unit of change is a unified diff the
   user can inspect, approve, or reject.
6. **Tool output is data, never instructions.** File contents and command
   output are treated as untrusted input, not as direction.
7. **Local-first.** JSONL session events on local disk only. No server, no
   telemetry, no cloud storage.

## Project Contract (locked for v0.1)

| Item | Value |
|---|---|
| Name | Kirsch |
| Language | Go |
| UI | Bubble Tea full-screen TUI (primary); non-interactive `kirsch run` later |
| License | MIT |
| Providers | One wire format for v0.1 (Anthropic Messages), with the built-in endpoints `opencode` (OpenCode Go, the default) and `anthropic` |
| Storage | JSONL session events, local disk only |
| CLI | stdlib `flag` + hand-rolled subcommands (no cobra) |
| Platforms | macOS + Linux, amd64/arm64 (no Windows in v0.1) |
| First user | Solo developer on real projects (WordPress/PHP, Go) |

## Explicitly Out of Scope for v0.1

Multiple providers, `git commit`/`git push` tools, MCP, LSP, subagents,
plugins, OS keychain, web UI, session branching/search, automatic compaction
UI tuning. Do not build these early.

## Repository Layout

```
plan/     everything needed to BUILD Kirsch — spec, architecture, ADRs,
          UI spec, per-milestone instruction sets. Scaffolding; plan/ stays
          for now and will be removed in a later release.
doc/      everything needed to USE Kirsch — install, configuration, usage,
          troubleshooting. The pages are in place, and `doc/README.md`
          indexes them.
```

If you are unsure where something belongs: *is it for someone building Kirsch,
or someone using it?* Building → `plan/`. Using → `doc/`.

| Document | Purpose |
|---|---|
| [`plan/README.md`](plan/README.md) | Build index, rules for the builder |
| [`plan/PROGRESS.md`](plan/PROGRESS.md) | Live milestone status, deliverables, blocking, and claiming |
| [`plan/START-HERE.md`](plan/START-HERE.md) | Ingestion order for humans and AI assistants, first-contribution path |
| [`plan/spec/kirsch-plan.md`](plan/spec/kirsch-plan.md) | Complete v0.1 build spec with milestones and amendments |
| [`plan/spec/architecture.md`](plan/spec/architecture.md) | Why the codebase is shaped this way: dependency rules, interfaces, concurrency, cancellation, error model |
| [`plan/spec/ui-spec-v0.1.md`](plan/spec/ui-spec-v0.1.md) | Full TUI spec: layout, cards, modals, mode state machine, palette, timing, accessibility |
| [`plan/spec/kirsch-ui-screens.md`](plan/spec/kirsch-ui-screens.md) | Screen reference: every UI state as an 80-column character grid with per-region colour maps — the render target and golden-test source |
| [`plan/testing/`](plan/testing/) | Manual walkthroughs, automated-test reference, security tooling |
| [`plan/adr/`](plan/adr/) | Architecture decision records |
| [`plan/process/working-agreement.md`](plan/process/working-agreement.md) | Claiming, branches, commits, PRs, file ownership, and board discipline for shared work |
| [`plan/process/code-standards.md`](plan/process/code-standards.md) | Go code standards — file layout, naming, comments, error handling, testing, and pre-PR checklist |
| [`plan/milestones/milestone-0.md`](plan/milestones/milestone-0.md) | Instruction set — repo bootstrap + static TUI prototype |
| [`plan/milestones/milestone-1.md`](plan/milestones/milestone-1.md) | Instruction set — workspace engine + read-only tools |
| [`plan/milestones/milestone-2.md`](plan/milestones/milestone-2.md) | Instruction set — patches, commands, approvals |
| [`plan/milestones/milestone-3.md`](plan/milestones/milestone-3.md) | Instruction set — provider + agent loop |
| [`plan/milestones/milestone-4.md`](plan/milestones/milestone-4.md) | Instruction set — real task loop + sessions |
| [`plan/milestones/milestone-5.md`](plan/milestones/milestone-5.md) | Instruction set — polish + release (v0.1.0) |
| [`doc/`](doc/) | End-user documentation: install, configuration, usage, troubleshooting |

## Progress

Milestone 4 (the real task loop + sessions) was operator-accepted on 2026-10-23 and tagged `m4`. Milestone 5 (polish and release) ends with the `v0.1.0` tag. While `plan/` remains, live status and the items carried after the release are in [`plan/PROGRESS.md`](plan/PROGRESS.md).

## What works today

This is what v0.1.0 does. The screenshots were taken on the build made just before the tag, so the start screen and the /help overlay show the version as `v0.1.0-dev`.

![Kirsch start screen showing the KIRSCH title, version line, provider notice and the typing line with its placeholder text](doc/images/start-screen.png)

![Kirsch approval card for the command echo demo, with the choices approve, approve for session, reject and detail](doc/images/approval-card.png)

![Kirsch /help overlay listing the keys and slash commands](doc/images/help-overlay.png)

Developer path, from a clone:

```
go run ./cmd/kirsch
```

Kirsch opens on a real repository, shows its project name, branch and dirty
state, and runs real model turns: the agent investigates (read, list, search,
diff), proposes small unified-diff patches — every patch waits for your approval
— and runs verification commands under the policy (exact-match allowlist,
session grant, or explicit approval; shells always ask). Every path crosses a
containment check, and the path denylist (`.env`, `*.pem`, `*.key`, `.git/**`,
`.kirsch/**`) is enforced in one place. Every turn is recorded to the local JSONL
session log; quit mid-task and `kirsch resume` (or auto-resume) restores the
transcript and grants. Apart from approved commands, the only subprocesses are
read-only `git` and `rg`.

When a conversation outgrows the context budget, Kirsch compacts it: the
original task and the most recent turns are kept verbatim, and the turns
between them are replaced by a structured summary. `/compact` runs the same
step on demand. The status bar shows the session's token counts and cost. A
model Kirsch has no pricing or context-window data for shows a notice once per
session (a fresh session after `/new` shows it again), and Kirsch falls back to
the smallest known context window. The system prompt asks the agent to end each
task that took actions with a final-task report.

The slash commands work on your real session. For example, `/status`
reports the session's state and `/diff` shows the unstaged diff in a modal.
The command table in [`doc/usage.md`](doc/usage.md) lists the rest.

## Verifying it

```bash
npm run tools      # install the lint and security tooling (one-off)
npm run check      # fmt + vet + lint + test + screens
npm run security   # govulncheck + gosec + secret scan
npm run ci:local   # every CI job, run locally by scripts/ci.sh
npm run fuzz       # property tests over the untrusted-input surfaces
```

The smoke test runs one real task against `testdata/repo-go-module` with a
real model. It needs an API key (`ANTHROPIC_API_KEY` or `OPENCODE_API_KEY`, or
their `KIRSCH_` forms), and it spends tokens, so run it by hand with
`bash scripts/smoke-test.sh`. It is not one of the `npm` scripts.

`npm` is a front door, not a dependency. `npm run screens` runs
`python3 scripts/lint-screens.py`.
Nothing in the build needs Node: `scripts/build.sh` and `.goreleaser.yaml` call
no `npm` or `node`. See
[`plan/testing/`](plan/testing/) for the manual walkthroughs and for what the
suite deliberately does not cover.

## License

[MIT](LICENSE) © 2026 Donovan Maidens

---

## SOURCE: doc/README.md

# Kirsch — Documentation

> The help overlay's footer points readers at the project page on GitHub, `github.com/djm56/kirsch`, where the terminal is wide enough to show the link. [`usage.md`](usage.md) is the usage guide.

`doc/` is for **people using Kirsch**. It contains:

| Document | Covers | State |
|---|---|---|
| [`install.md`](install.md) | Install (release binary, `go install`, build from source), first-run setup, API key | Written |
| [`configuration.md`](configuration.md) | Every key in `config.toml`, its default, and what it does — global vs project precedence | Written |
| [`usage.md`](usage.md) | Key bindings, slash commands, the approval model, sessions and resume | Written |
| [`troubleshooting.md`](troubleshooting.md) | `kirsch doctor` output explained, common failures and what they mean | Written |

**Everything about building Kirsch lives in [`plan/`](../plan/) instead.**
Build notes do not belong in this folder.

---

## SOURCE: doc/install.md

# Install Kirsch

Kirsch is a terminal program. This page covers the supported platforms, what
Kirsch needs on the machine, the ways to install it, how to set up an API key,
and how to check the install with `kirsch doctor`.

> **What this page covers.** Installing the `kirsch` program and checking that
> it works. It does not cover choosing a model provider or changing settings.
> Those are in [configuration.md](configuration.md).

---

## Supported platforms

| Operating system | Architectures |
|---|---|
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |

Windows is not supported. Kirsch stops a running command by stopping its
process group, and Windows has no Unix process groups.

## What Kirsch needs on the machine

### Git

Kirsch works on a Git repository. Without `--workspace`, Kirsch finds the
workspace by asking `git` for the repository that encloses the folder you start
it in. So `git` must be on your `PATH`, and you must start Kirsch inside a
repository. If either is missing, Kirsch does not start, and `kirsch doctor`
reports a failure.

With `--workspace <path>`, Kirsch uses that folder and does not ask `git` for
anything. If the folder is not a Git repository, Kirsch still starts, but its
Git features are off.

### ripgrep (optional)

Kirsch's code search uses `rg` (ripgrep) when `rg` is on your `PATH`. Without
`rg`, Kirsch uses its own built-in search, and search still works. Kirsch does
not stop for the lack of `rg`.

`kirsch doctor` reports the difference. Without `rg`, it prints these warnings:

    warn  rg: not found on PATH; search_code uses the pure-Go fallback
    warn  backend: search_code uses pure-go (rg is not on PATH)

A warning does not make `kirsch doctor` fail. To use ripgrep, install it with
your system's package manager.

## Install from a release archive

Release archives are on the
[releases page](https://github.com/djm56/kirsch/releases).

Each release has a `.tar.gz` archive for every supported platform, and a
checksum file for the whole release. The names follow this pattern, where
`<version>` is the release number without the leading `v`, such as `0.1.1`:

- `kirsch_<version>_<os>_<arch>.tar.gz`, where `<os>` is `darwin` or `linux`
  and `<arch>` is `amd64` or `arm64`
- `kirsch_<version>_checksums.txt`

The steps below use version `0.1.1` on macOS with Apple silicon. Change the
variables at the top for your release and platform.

```bash
VERSION=0.1.1
OS=darwin
ARCH=arm64
ARCHIVE="kirsch_${VERSION}_${OS}_${ARCH}.tar.gz"

curl -fLO "https://github.com/djm56/kirsch/releases/download/v${VERSION}/${ARCHIVE}"
curl -fLO "https://github.com/djm56/kirsch/releases/download/v${VERSION}/kirsch_${VERSION}_checksums.txt"
```

Check the archive against the checksum file before you unpack it. The checksum
file lists every platform, so the command selects the one line for your archive
and passes it to the checker.

On macOS, use `shasum`:

```bash
grep -F "${ARCHIVE}" "kirsch_${VERSION}_checksums.txt" | shasum -a 256 -c -
```

On Linux, use `sha256sum`:

```bash
grep -F "${ARCHIVE}" "kirsch_${VERSION}_checksums.txt" | sha256sum -c -
```

Each command prints a line ending in `OK` when the archive matches. A line
ending in `FAILED` means the download is damaged or incomplete. Download it
again, and do not run it.

Unpack the archive. It contains the `kirsch` program, and may also hold
licence and readme files:

```bash
tar -xzf "${ARCHIVE}"
```

Move `kirsch` onto your `PATH`. Either of these works:

```bash
sudo mkdir -p /usr/local/bin && sudo mv kirsch /usr/local/bin/
```

```bash
mkdir -p ~/.local/bin
mv kirsch ~/.local/bin/
```

If you use `~/.local/bin`, make sure that folder is on your `PATH`.

Then check the version:

```bash
kirsch --version
```

A release binary prints the release version, for example:

```text
kirsch 0.1.1
```

## Install with `go install`

`go install` builds Kirsch from its source on the module proxy. You need Go
1.26.11 or later, the version named in the module's `go` line. Run `go version`
to see what you have. If your Go is older, recent Go releases fetch the
required toolchain automatically. They do this unless `GOTOOLCHAIN` is set to
`local`.

Install the latest version:

```bash
go install github.com/djm56/kirsch/cmd/kirsch@latest
```

Or install one release, by tag:

```bash
go install github.com/djm56/kirsch/cmd/kirsch@v0.1.1
```

The binary goes to `$(go env GOBIN)` when `GOBIN` is set. Otherwise it goes to
`$(go env GOPATH)/bin`. Make sure that folder is on your `PATH`.

The version text depends on what you built from:

| You installed | `kirsch --version` prints |
|---|---|
| A release tag, such as `@v0.1.1`, or `@latest`, which is the newest release | The tag without its leading `v`, such as `kirsch 0.1.1` |
| Anything else, such as a branch (`@main`) or a commit | `kirsch 0.1.0-dev` |

Go records a commit as a pseudo-version, which is not a release tag, so a
commit build prints `kirsch 0.1.0-dev`.

## Build from source

Build from a clone of the repository. You need Go, as above, and `git`.

```bash
git clone https://github.com/djm56/kirsch.git
cd kirsch
bash scripts/build.sh
./bin/kirsch --version
```

The binary is written to `bin/kirsch` inside the checkout. The script stamps a
version into the binary:

- `bash scripts/build.sh 1.2.3` stamps `1.2.3`. A leading `v` is removed, so
  `v1.2.3` also stamps `1.2.3`.
- `bash scripts/build.sh` with no argument stamps the version only when the
  checkout is exactly on a `v` release tag and has no uncommitted changes.
  Otherwise the binary reports `kirsch 0.1.0-dev`.

## Set up your API key

Kirsch sends model requests through a provider. The default provider is
`opencode`, and it reads its key from the environment variable
`OPENCODE_API_KEY`. The built-in `anthropic` provider reads
`ANTHROPIC_API_KEY`.

Export the key in your shell. Replace the placeholder with your own key:

```bash
export OPENCODE_API_KEY="<your-opencode-key>"
```

Add the line to your shell's startup file if you want it set in every session.

Kirsch also accepts the same name with a `KIRSCH_` prefix, such as
`KIRSCH_OPENCODE_API_KEY`. It checks the prefixed name first. If both are set,
the prefixed one is used. An empty variable counts as unset. The
[secrets rules](configuration.md#secrets) explain the lookup order.

**Never put an API key in a config file.** Kirsch refuses to start when a
config file holds a key whose name looks like a credential. To use another
provider, see [configuration.md](configuration.md).

## Check the install with `kirsch doctor`

Run `kirsch doctor` inside your project folder:

```bash
cd /path/to/your/project
kirsch doctor
```

To check a folder other than the current one, pass `--workspace <path>`.

Doctor checks what Kirsch depends on, such as `git`, `rg`, your project and
settings, your API key, the model, the session lock, and the folder where
Kirsch saves sessions. It creates no files and makes no network calls. It never
sends a prompt to a model.

The report starts with the version line. Each check then gets one line with its
status (`ok`, `warn`, or `fail`), the check name, and a detail. The last line
counts the results.

This is an example. Your lines and paths depend on your machine:

```text
kirsch 0.1.1
ok  git: found at /usr/bin/git
ok  rg: found at /usr/local/bin/rg
ok  repository: /home/you/project
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6854 bytes
ok  state: /home/you/.local/share/kirsch/sessions/3f9a1c07b2d4 exists and is writable
ok  lock: free
doctor: 9 ok, 1 warn, 0 fail
```

The first time you run doctor, before Kirsch has run in that project, the
`state` line says the folder does not exist yet and that Kirsch will create it,
and the `lock` line says `free (no lock file yet)`. Each is `ok`.

Doctor exits with status 0 when no check fails, and with status 1 when at
least one check fails. A warning does not cause a failure. A `fail` line names
the problem, and it is the one to fix before you use Kirsch.

---

## SOURCE: doc/usage.md

# Using Kirsch

Kirsch is a terminal-native coding agent. This page covers the interface: how
the screen is laid out, every key it responds to, every slash command, and the
approval model that stands between a tool and your files.

> **What this page covers.** Kirsch runs real model turns against your
> repository, keeps a durable session log, and resumes it on launch or with
> `kirsch resume`. The layout, the key bindings, the slash commands and the
> approval flow all work as described below. Sections that describe a limit
> say so in place.
>
> The help overlay inside the program (`?`, or `/help`) prints its own short
> table of keys. That table is text only: the key handlers do not read it, so a
> row that is wrong has to be corrected by hand. The table also leaves out some
> keys the program accepts, such as `Home` and `Esc` in Browsing. Where this page
> and the overlay disagree, the program decides what a key does, and this page
> describes that.

---

## Starting it

```bash
kirsch                       # run in the enclosing Git repository
kirsch resume                # resume the most recent session for this workspace
kirsch --new                 # start a fresh session even when a previous session exists
kirsch --workspace ~/code/x  # run against a directory you name
kirsch --debug               # also write a structured debug log
```

`kirsch resume` picks up the most recent session for this workspace, including
its session grants. If another Kirsch is already running in the workspace, it
starts a fresh session instead and says so. `kirsch --new` starts a fresh
session even when a previous session exists.

Kirsch works inside one directory and never outside it — the **workspace**. With
no flag it is the enclosing Git repository; `--workspace` names one explicitly.
The project name in the header is that directory's base name.

The debug log goes to `$XDG_STATE_HOME/kirsch/debug.log`, or
`~/.local/state/kirsch/debug.log` when that variable is unset. It never writes
to stdout or stderr, so it cannot corrupt the display. `KIRSCH_DEBUG=1` turns it
on without the flag.

---

## The screen

```
 Kirsch ───────────────────────────────────────   header row 1 — wordmark + rule
 my-project ─ main ●                              header row 2 — session line

                                                  transcript (fills)
 ── you ───────────────────────
 Fix the Divide validation

 ▸ read_file calc/divide.go · 4ms · ok            tool card (collapsed)

 I found the issue in... ▌                        streaming assistant

 ──────────────────────────────────────────────   separator rule
 claude-sonnet-5 · ⠋ thinking · in 12.4k out 980 cache 8.9k/1.2k · $0.04   status bar
 ──────────────────────────────────────────────   separator rule
 > _                                              composer
```

The `cache` figures are tokens read from the cache, then tokens written to it.

**Header — two rows.** `Kirsch` stands alone on the first, with a rule running
off to the right. The second is the session: project name, branch, a `●` if the
worktree is dirty, and `(compacted)` once the session has been compacted. Below
60 columns the branch is dropped, and the dirty marker goes with it. Only the
first row carries a rule.

**Transcript.** Everything that has been said and done, oldest at the top. It
takes whatever rows the rest of the frame does not need.

**Status bar.** Model, current state, session token totals and cost, and any
active session grants. Persistent warnings, for example a recovered session or
another Kirsch running in the same workspace, sit at the right. No narrowing
step removes them, but on a very narrow terminal a long one is cut at the right
edge.

**Composer.** One row to start, growing as you add lines up to a fixed maximum
height. Past that maximum the composer scrolls up and down to keep the line you
are typing on in view. A line wider than the terminal is cut at the edge with a
`⋯` marker; the composer does not scroll sideways. While a turn is running it
dims and stops accepting input, and shows `(input disabled, Esc cancels)` at
the right.

**The frame keeps one blank column at each side**, and none at the top or
bottom. A terminal's rows are scarcer than its columns: the layout already
spends every row it has, so a blank row would come out of the transcript, which
at the smallest supported terminal is four rows tall. The right-hand column is
reserved rather than painted, so no row ever ends in trailing whitespace, and
both edges show your own terminal background rather than Kirsch's.

**Sizes.** Kirsch runs from 40×10 upwards. As the window narrows it drops the
status bar's token figures (below 80 columns), then the branch (below 60), and
below 40 replaces the transcript with a `terminal too narrow` notice rather than
rendering something illegible. Below 80 columns the status bar also shortens its
left-hand text as the width shrinks, starting with the grant count, and keeps the
state for as long as it can. As the height shrinks it shrinks the composer, then
drops the two separator rules together, then the header — both of its rows at
once. Nothing panics at any size, and a zero-sized terminal draws nothing at all.

---

## Keys

Kirsch is modal. Which keys do what depends on where the focus is, and the mode
changes only by an explicit key.

### Composing — the default

| Key | Action |
|---|---|
| `Enter` | Send |
| `Shift+Enter` | Newline, where the terminal sends `ESC`+`CR` for it |
| `Alt+Enter` | Newline, same decode path — **Option+Enter on a Mac keyboard produces nothing** |
| `Ctrl+J` | Newline — the one form every terminal can produce |
| `Tab` | While typing a slash-command name (no space yet), complete a unique prefix; an ambiguous or unknown prefix does nothing. In any other text, move to the cards (→ Browsing) |
| `Ctrl+G` | Scroll back to the bottom of the transcript and re-pin it |
| `↑` / `↓` on the first / last row | Previous / next entry from this session's history (up to 100); past the newest, back to your unsent draft. Ignored while a turn is running |
| `Shift+↑` | Move focus to the cards (→ Browsing) |
| `Esc` / `Ctrl+C` | Cancel the running turn; if nothing is running, clear the composer |
| `Ctrl+C` twice within 1s | Force quit |

`q` is an ordinary character here — there is no bare-key quit. `?` is an
ordinary character too: a help overlay that fired mid-sentence would make the
composer unusable. Quitting is `/quit`, `/exit`, or `Ctrl+C` twice.

**`Ctrl+G` is worth knowing.** It is the only way back to the bottom that does
not send something first, and it is the only composing key that works *during* a
running turn — which is when watching the bottom of a stream is worth the most.
It moves the viewport and nothing else: your half-typed message, the card
selection and the mode are all left exactly as they were. `End` would have been
the obvious key and is not available: the text area already uses it for
end-of-line, as it uses `Ctrl+End` for end-of-input.

**The key-hint line.** On a terminal at least 24 rows tall, the bottom row lists the keys that act in the current focus — `↑↓ history · ⇧↑/Tab cards · /help` in the composer, for example. It goes blank while a modal or confirm prompt is open, and is dropped on shorter terminals.

### Browsing — after `Shift+↑` or `Tab`

| Key | Action |
|---|---|
| `↑` / `↓` | Move the card selection (plain text blocks are skipped) |
| `PgUp` / `PgDn` | Scroll without moving the selection |
| `Home` / `End` | Top / bottom — `End` re-pins |
| `g` / `G` | Top / bottom, for terminals that swallow `Home` and `End` |
| `Enter` | Tool cards: switch between the 10-line preview and the head line only |
| `d` | Open the full content or diff in a modal |
| `?` | Help overlay |
| `Tab` / `Esc` | Back to the composer — does **not** move the view |
| `↓` past the last card | Back to the composer |

Scrolling and selection are separate: `PgUp`/`PgDn` move the view and leave the
selection where it is, `↑`/`↓` move the selection and scroll only as far as they
must to keep it visible.

`g`/`G` exist because `Home` and `End` are not universally reachable — macOS
Terminal sends both as a sequence the toolkit does not decode. Kirsch normalises
that on the way in, so both pairs work; `g`/`G` are the belt to that braces.

### An approval is pending

| Key | Action |
|---|---|
| `y` | Approve, this once |
| `a` | Approve and grant for the rest of the session — `run_command` only |
| `n` | Reject |
| `d` | Open the detail or diff |
| `?` | Help overlay |
| `Esc` / `Ctrl+C` | Cancel the turn — recorded as cancelled, not rejected |
| `Tab` / `Shift+↑` | Re-arm a card that a stray key released |

Every other key is swallowed, and none reaches the composer. A key that is not one of the
above and not a navigation key also *releases* the card: the composer shows
`approval pending — Tab to return to the card`, and `y`, `a` and `n` do nothing until `Tab`
or `Shift+↑` re-arms it. That is deliberate: typing a command while a card waits — the `n`
in `/new` — must not be read as an answer. Navigation keys never release; pasted text is
dropped.

### A modal is open

| Key | Action |
|---|---|
| `j` / `k` / `↑` / `↓` | Scroll a line |
| `PgUp` / `PgDn` | Scroll a page |
| `g` / `G` / `Home` / `End` | Top / bottom |
| `Esc` | Close, returning to whatever mode you were in |
| `?` | Also closes the help overlay |

Closing returns you to the *previous* mode, so `Esc` from a modal opened over a
pending approval puts you back at the approval, not at the composer.

### A confirm prompt is up

One-line questions — "Discard the running turn and start a new session?",
"Clear 2 session grants?", "Paste 12KB into the composer?" — appear
immediately above the composer, not over the status bar.

| Key | Action |
|---|---|
| `y` | Yes |
| `n` / `Esc` | No |

`Esc` is not spelt out on the prompt itself: it does the same thing as `n`, the
row already names a way out, and the cells it would cost come straight out of
the question — which at 40 columns is the part being truncated.

### Everywhere

`Ctrl+C` is context-dependent by design: it cancels a running turn, and
otherwise arms a force quit. Two presses inside a second always quit, from any
mode.

**Pasting** is literal — pasted text is never interpreted as keys, and newlines
in it are preserved. Pasted tabs become spaces in the composer, so tab-indented
text arrives with space indentation. A terminal that does not send a paste as
one block delivers each tab as a Tab key press instead: the first moves focus
to the cards, where the text that follows is read as card keys and not typed.
A paste over 8KB asks first. A paste that arrives while an overlay is open is
discarded rather than queued, so text cannot appear in a composer you could not
see it landing in.

---

## Scrolling and pinning

The transcript is **pinned to the bottom**. New content while pinned keeps it
pinned, so a running turn stays in view without you touching anything.

Scrolling up **unpins** it, and a `↓ 3 new` indicator appears at the bottom
right counting what has arrived since. The view then stays exactly where you put
it: typing does not drag it back, and neither does returning to the composer
with `Esc`. Reading scrollback is a deliberate act and Kirsch does not undo it
for you.

It re-pins when you:

- scroll back to the bottom yourself,
- press `Ctrl+G` in the composer,
- press `End` or `G` while browsing,
- send a message, or
- submit a slash command.

A slash command re-pins because it is a request and its answer belongs on
screen. The exception is an unknown command. It answers in the dim hint under
the composer and puts nothing in the transcript, so re-pinning would cost you
your place to report a typo.

---

## Slash commands

Type `/` at the start of a single line. A command must be the whole line:
multi-line text beginning with `/` is an ordinary message, so a pasted diff or
stack trace is never mistaken for one. `Tab` completes a unique prefix.

| Command | What it does |
|---|---|
| `/help` | The help overlay |
| `/status` | A notice summarising the session — for example the model, endpoint, branch, dirty count, token totals and compaction level |
| `/diff` | The unstaged working-tree diff, in a modal |
| `/files` | A notice listing the files changed by approved patches this session. A resumed session starts this list empty and does not rebuild it from the earlier transcript |
| `/approvals` | Lists active session grants, and offers to clear them |
| `/new` | Start a new session — confirms first if a turn is running |
| `/compact` | Compact the conversation history |
| `/quit` | Quit |
| `/exit` | Quit — the same thing, spelt the other way |

An unknown command produces a dim hint under the composer, never an error card,
and is **never sent to the model**.

---

## Tool cards

Every tool call appears in the transcript as a card, collapsed by default:

```
▸ read_file calc/divide.go · 4ms · ok
```

Glyph, tool name, what it acted on, the result, and how long it took. By default,
a tool card opens as a preview showing the first 10 lines. When lines are hidden,
a marker reads `‹10 of 4,181 lines — d full output · Enter collapse›`. `Enter`
toggles between preview and collapsed (head only). `d` opens the whole thing in a
modal. The 10-line preview keeps the scrollback legible while tools are running.

Where a cap was hit further upstream — a 200KB read, a 4000-token result — the
card says so in the same shape: `‹truncated — 200KB cap›`.

Tool output is sanitised before it is rendered. Escape sequences are stripped
rather than passed through, because an unstripped sequence hands arbitrary
terminal control to whatever the tool printed. Carriage-return runs render as
their final segment, tabs become four spaces, and other control characters
become a dim `·`.

---

## The approval model

A patch appears as a card and waits, and no setting turns that off. A command
runs without a card when it is exactly one of the built-in allowlist commands,
with no further arguments, or when a session grant covers it. Every other
command appears as a card and waits, unless
`policy.require_approval_for_commands` is `false` in your configuration; shells
and runners wait even then (see [`configuration.md`](configuration.md)). Check
that setting when you expect a card and none appears:

```
┌─ approval required ─────────────────────────────┐
│ apply_patch — add input validation              │
│                                                 │
│ files: 2 changed (calc/divide.go,               │
│        calc/divide_test.go)                     │
│                                                 │
│ [y] approve   [n] reject   [d] view diff        │
└─────────────────────────────────────────────────┘
```

`y` approves this one action. `n` rejects it. `d` shows you exactly what it
would do — the diff for a patch, the full command and its environment for a
command — before you answer.

**`a` grants for the session, and only commands offer it.** A session grant
means "stop asking me about this command for the rest of this session"; it is
offered on `run_command` cards and never on `apply_patch` ones, because a patch
is a different change every time and a blanket yes to it would mean nothing. The
status bar shows a count of active grants, `/approvals` lists them, and it
offers to clear them.

Grants live and die with the session. The session log records each grant, so
`kirsch resume` (or auto-resume) restores them, unless another Kirsch is already
running in the workspace, in which case a fresh session starts. A grant you
clear with `/approvals` is recorded in the session log as cleared, so it stays
cleared after `kirsch resume`. A new session, including one started with
`/new`, starts with none.

---

## When colour or Unicode is not available

Set `NO_COLOR` to a non-empty value, run under `TERM=dumb`, or pipe the output
somewhere that is not a terminal, and Kirsch emits **no colour codes and no
style sequences** — not merely no visible colour. An empty `NO_COLOR`
(`NO_COLOR=`) is treated as unset, as the no-color.org convention asks, so it
does not turn colour off by itself. The layout is identical: same rows, same
columns, same box positions. Every state is identified by its glyph rather
than by its colour, so nothing becomes ambiguous.

Where the locale is not UTF-8, every glyph falls back to an ASCII form of the
same width — `>` for `▸`, `[ok]` for `✓`, `...` for `⋯`, `+-|` for box drawing,
and the wordmark to `K I R S C H`. The layout does not move, because every
substitution preserves width.

Kirsch never paints a full-screen background. It sets foreground colours and
accents only, so it sits inside your terminal theme rather than replacing it.

---

## If something looks wrong

Run with the debug log on and read it afterwards:

```bash
kirsch --debug
cat ~/.local/state/kirsch/debug.log
```

Rendering problems are worth reporting with your terminal (`echo $TERM`) and its
size (`tput cols`, `tput lines`) — most of the ones found so far have been
specific to one terminal's encoding of one key, and the size is what makes a
layout defect reproducible.

---

## SOURCE: doc/configuration.md

# Configuration

Kirsch reads its settings from TOML files. Every setting has a built-in
default, so Kirsch runs with no configuration file at all. This page lists
every key, what it does, and where it may be set.

> **What this page covers.** The config file locations, the order in which
> the layers apply, the key reference, and the rules for secrets. Kirsch
> reads no configuration from command-line flags in v0.1. Options such as
> `--new`, `--debug`, and `--workspace` change one run and set no key on
> this page.

---

## Where the config files live

Kirsch reads a global file and a project file. Either may be missing; a
missing file is not an error.

| Scope | Path | Notes |
|---|---|---|
| Global | `$XDG_CONFIG_HOME/kirsch/config.toml` | Used when `XDG_CONFIG_HOME` is set and not empty. |
| Global | `~/.config/kirsch/config.toml` | Used otherwise. |
| Project | `.kirsch/config.toml` at the workspace root | The workspace is the enclosing Git repository, or the directory named with `--workspace`. |

The project file is read from the repository you are working in. Read the
[project file rules](#what-the-project-file-may-set) before you rely on it.

## Precedence

Kirsch applies these layers, in this order. Each layer changes only the keys
it sets. Keys that a layer does not mention keep the value from the layer
before it.

1. **Built-in defaults.** The values in the key reference below.
2. **Global file.** Your personal settings.
3. **Project file.** Accepted only for `context.project_files`. Every other
   key in it is ignored, with a warning that names the key. The warning goes
   to the [debug log](#debug-log).

Command-line flags are not a configuration layer. Kirsch reads no
configuration from command-line flags in v0.1, and no key on this page can
be set that way.

## What the project file may set

A project file may set `context.project_files` and nothing else. A
repository is untrusted input, and a project file overrides the global one,
so anything it could set, a hostile repository could set. Kirsch therefore
ignores every other key in a project file and names each ignored key in a
warning. That warning is written to the [debug log](#debug-log).

The rules for `context.project_files` itself are in the
[context table](#context). In short, each entry must be a relative path
inside the workspace.

A project file that sets a key whose name looks like a credential stops
startup. See [Secrets](#secrets).

## Secrets

**API keys are never read from a config file.** Name the environment variable
that holds the key with `api_key_env`, and export that variable in your shell.

```bash
export OPENCODE_API_KEY="..."   # or KIRSCH_OPENCODE_API_KEY, see below
```

Kirsch looks for the key in this order, and an empty value counts as unset:

1. `KIRSCH_` followed by the `api_key_env` name, for example
   `KIRSCH_OPENCODE_API_KEY`.
2. The `api_key_env` name itself, for example `OPENCODE_API_KEY`.

If both are set, the prefixed variable wins without a message. This lets
Kirsch run alongside other tools that use the unprefixed name. The
`/status` command names the variable that supplied the key, never its value.

Kirsch also refuses to start when any config file, global or project,
contains a key whose name looks like a credential. The match is
case-insensitive and looks at the last part of each key's name. `api_key`,
`apikey`, `token`, `secret`, and `password` all count, wherever they appear in
that part. The check covers keys Kirsch does not otherwise recognise, so
`max_tokens` stops startup as well as `api_key` does. Only the last part of a
dotted key is checked, so `provider.x.max_tokens` is checked as `max_tokens`
and stops startup. The error names the key.
The one exception is `api_key_env`, which holds a variable name, not a value.

## Key reference

Keys are written with their full dotted name, as they appear in the file.
`<name>` in an endpoint key stands for the endpoint's name, such as
`opencode`.

### `[provider]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `provider.default` | `"opencode"` | The name of an endpoint defined under `[provider]`, built-in or your own | Global | Chooses the endpoint that serves model turns. Startup fails when no endpoint has this name. |

Each endpoint has the fields below. The built-in endpoints are
`opencode` and `anthropic`, and the table gives their defaults. You may define
endpoints under other names. A new endpoint has no defaults for the fields the
table marks as required, so those must be set. For a new endpoint, `thinking`
and `prompt_caching` default to off.

| Key | Default for `opencode` / `anthropic` | Valid values | Scope | What it does |
|---|---|---|---|---|
| `provider.<name>.base_url` | `https://opencode.ai/inference/go/anthropic/v1` / `https://api.anthropic.com/v1` | Required for a new endpoint. An `https` URL with any host. An `http` URL only on `localhost`, an IPv4 address in the `127.x.x.x` range, written in full, such as `127.0.0.1`, or `[::1]`. No username or password in the URL. | Global | The address Kirsch sends model requests to. |
| `provider.<name>.auth` | `x-api-key` / `x-api-key` | Required for a new endpoint. `x-api-key` or `bearer`. | Global | How the API key is sent with each request. |
| `provider.<name>.api_key_env` | `OPENCODE_API_KEY` / `ANTHROPIC_API_KEY` | Required for a new endpoint. A variable name. | Global | The name of the environment variable that holds the key. Never the key itself. |
| `provider.<name>.model` | `minimax-m3` / `claude-sonnet-5-5` | Required for a new endpoint. A model name the endpoint accepts. | Global | The model named in each request. |
| `provider.<name>.prompt_caching` | `true` / `true` (a new endpoint defaults to `false`) | `true` or `false` | Global | Adds cache markers to the parts of the system prompt that are marked for caching. |
| `provider.<name>.thinking` | `off` / `off` (a new endpoint defaults to `off`) | `off`, `low`, `medium`, or `high` | Global | The thinking level requested from the model. The exact request fields depend on the model. |

Startup checks every endpoint in the file, not only the one in use. A
malformed endpoint therefore stops startup even when `provider.default`
names a different one.

### `[policy]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `policy.default_command_timeout_seconds` | `60` | A whole number from `1` to `300` | Global | The timeout for a command the agent runs, when the agent names no timeout of its own. |
| `policy.require_approval_for_patches` | `true` | `true` or `false` | Global | Whether applying a file change asks for your approval first. In this version every file change asks, whatever this is set to. See below. |
| `policy.require_approval_for_commands` | `true` | `true` or `false` | Global | Whether running a command asks for your approval first. Some commands always ask. See below. |
| `policy.allow_session_scoped_grants` | `true` | `true` or `false` | Global | Whether an approval can be granted for the rest of the session, not only once. Only commands can be granted: a file change and a shell are never granted. Grants are also off while another Kirsch is running in the same workspace. |
| `policy.env_passthrough` | `[]` | A list of variable names | Global | Extra environment variables passed to commands the agent runs, on top of `PATH`, `HOME`, and `LANG`. Names ending in `_TOKEN`, `_KEY`, or `_SECRET`, and names starting with `AWS_`, are removed even when listed. The match is case-sensitive: only the upper-case forms are matched, so a listed `my_token` reaches the command, and other secret-looking names, such as `GITHUB_PAT`, are not matched either. |

The 300-second limit applies to every command. A timeout above 300 is refused,
whether the agent names it or it comes from `default_command_timeout_seconds`.
The refusal comes after any approval prompt, so you may be asked to approve a
command that is then refused. Kirsch still starts with a larger value here, but
then every command that names no timeout of its own is refused. Keep this
setting at 300 or below.

**Command approvals.** `require_approval_for_commands` governs commands outside
a short built-in allowlist. Kirsch decides each command in this order:

1. Shells and runners always ask, even when the setting is `false`. The
   complete set is `sh`, `bash`, `zsh`, `dash`, `env`, `xargs`, and `nohup`,
   matched by file name, so `/bin/bash` counts too. Other shells, such as
   `fish`, `ksh`, and `csh`, and interpreters, such as `python`, are not in
   this set. They are treated like any other command: with the setting
   `false`, they run without asking.
2. The built-in allowlist runs without asking. It matches exact commands only,
   with nothing after them. The complete list is `go build`, `go test`,
   `git diff`, `git log`, and `ls`, and nothing else. So `go test ./...` and
   `ls -la` still ask.
3. Any other command asks when the setting is `true`, unless a session grant
   covers it. When the setting is `false`, every other command runs without
   asking.

**File-change approvals.** `require_approval_for_patches` is read at startup,
but the code that applies file changes never checks it. In this version every
file change asks for your approval, whatever the value. Setting it to `false`
does not turn that prompt off.

### `[context]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `context.project_files` | `["AGENTS.md", "CLAUDE.md"]` | A list of paths relative to the workspace root | Global or project | The files Kirsch reads into the system prompt. Kirsch reads the first entry that is an ordinary file, and does not read the rest. A missing entry is skipped without a message. An entry is refused, with a warning, when it is an absolute path, when it leaves the workspace, or when it is a path Kirsch never reads, such as `.env` or `certs/server.key`, or anything inside a `.git` or `.kirsch` folder. An entry that is a folder or other special item is skipped with a warning. A list in a file replaces the default list; it does not add to it. |
| `context.max_project_context_bytes` | `32768` | A whole number of zero or more. Values above `32768` are treated as `32768`. | Global | The most bytes read from the chosen file. A longer file is cut at its last complete line within the limit, and a notice giving the limit is added. If no line break falls within the limit, only the notice is sent. At `0` that is always the case: Kirsch still picks the file, but sends only the notice. An empty file sends nothing. |

### `[session]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `session.storage_dir` | `~/.local/share/kirsch/sessions` | A path. Only `~` and a path that starts with `~/` expand to your home folder. A relative path is taken from the folder you start Kirsch in. | Global | The folder for session data. Session logs go in `<folder>/sessions/<workspace-hash>`, one folder per workspace, and `index.json` and `usage.json` sit directly in `<folder>`. If the path ends in a folder named `sessions`, Kirsch treats that folder as the sessions folder and uses its parent as `<folder>`. |
| `session.auto_resume` | `true` | `true` or `false` | Global | When `true`, starting Kirsch resumes the most recent session for this workspace. If there is none, Kirsch starts a new session and shows the notice `no previous session — starting fresh`. If the resume fails for another reason, the notice says so and gives the reason. When `false`, or when another Kirsch is already running in this workspace, Kirsch starts a new session. The `--new` option on the command line also starts a new session. |

**Worked example, with the default.** Session logs go in
`~/.local/share/kirsch/sessions/<hash>`, one folder per workspace, where
`<hash>` is a 12-character hash of the workspace path. Each session is an
`<id>.jsonl` file in that folder. `index.json` and `usage.json` sit in
`~/.local/share/kirsch`, the folder above `sessions`. An empty
`session.storage_dir` falls back to that same folder, `~/.local/share/kirsch`.

### `[telemetry]`

| Key | Default | Valid values | Scope | What it does |
|---|---|---|---|---|
| `telemetry.debug_log` | `false` | `true` or `false` | Global | Writes the debug log on every run. Off by default. The `KIRSCH_DEBUG` variable and the `--debug` option turn it on for one run. See [Debug log](#debug-log). |

## Environment variables

The variables below change how Kirsch behaves. Kirsch reads others too, such as
`HOME`, to find your home folder.

| Variable | Effect |
|---|---|
| `KIRSCH_DEBUG` | Set to `1` to write the debug log for this run. Any other value does nothing. |
| `XDG_CONFIG_HOME` | Changes the global config directory, as described above. |
| `XDG_STATE_HOME` | Changes the directory for the debug log. |
| `KIRSCH_<NAME>` and `<NAME>` | The API key variables named by `api_key_env`. See [Secrets](#secrets). |

## Debug log

Kirsch writes a debug log only when you ask for one. It is off by default.
Turn it on for a single run with `--debug` or `KIRSCH_DEBUG=1`, or for every run
with `telemetry.debug_log = true`.

The log goes to `$XDG_STATE_HOME/kirsch/debug.log`, or to
`~/.local/state/kirsch/debug.log` when `XDG_STATE_HOME` is unset or empty. It
never writes to the screen, so it cannot disturb the display.

Warnings about configuration and project context go only to this log. With the
log off, a warning such as an ignored key in a project file, or a context file
that was refused, is not shown anywhere. Turn the log on to see those warnings.
[Using Kirsch](usage.md#starting-it) describes the same log.

## Startup checks

Kirsch checks the merged configuration before the interface opens. A failed
check stops startup and names the key at fault. The checks include, for
example:

- `provider.default` names an endpoint that exists.
- Each endpoint has the fields the table marks as required, and its `auth` and
  `thinking` hold valid values.
- Each `base_url` passes the rules in the endpoint table.
- `policy.default_command_timeout_seconds` is above zero. A value above 300
  passes this check. See the policy section.
- `context.max_project_context_bytes` is zero or more.

The credential check in [Secrets](#secrets) runs while the files are read,
before these checks.

An unknown key is a warning, not an error, unless its name looks like a
credential. A config file written for a newer Kirsch still starts an older one.

## Example

A global file that switches the default endpoint and turns on the debug log:

```toml
[provider]
default = "anthropic"

[telemetry]
debug_log = true
```

A project file that changes only the context files Kirsch reads:

```toml
[context]
project_files = ["AGENTS.md"]
```

---

## SOURCE: doc/troubleshooting.md

# Troubleshooting

When Kirsch will not start, or starts and then cannot send a prompt, begin
with `kirsch doctor`. It prints a line for each result its checks give. It sends
no prompt to a model and makes no network calls, so you can run it at any
time.

This page explains each line doctor can print, the common failures behind
them, and what to do about each one. For installing Kirsch, see
[install.md](install.md). For the settings named here, see
[configuration.md](configuration.md).

> **What this page covers.** The `kirsch doctor` report line by line, and the
> messages you see in the program for the failures people meet most often.
> Placeholders in angle brackets, such as `<path>`, stand for values that
> depend on your machine. Each one is explained where it first appears in its
> section.

---

## Reading the report

Run doctor inside your project folder, or pass `--workspace <path>` to check
another folder:

```bash
cd /path/to/your/project
kirsch doctor
```

The report has this shape:

```text
kirsch <version>
<status>  <check>: <detail>
doctor: <ok> ok, <warn> warn, <fail> fail
```

- `kirsch <version>` is the opening line. The version is the release version,
  or `0.1.0-dev` in a build that carries no release version. See
  [install.md](install.md) for how a build gets its version.
- `<status>` is `ok`, `warn`, or `fail`. Spaces follow it, as in the examples below.
- `<check>` is the name of the check, in lower case.
- `<detail>` is the message. Each one is quoted below.
- The last line counts the statuses.

The checks print in the order of the headings below. A `warn` line that
begins `skipped:` means the check did not run, because an earlier check left
it nothing to read. A config that did not load skips every check that reads
it, and a run with no workspace skips the checks that need a folder.

**Exit status.** Doctor exits `0` when no check fails, and `1` when at least
one check fails. A warning never causes a failure. Fix the `fail` lines first.

Example output from a repository with no key variables set. Your lines and
paths will differ:

```text
kirsch 0.1.0-dev
ok  git: found at /usr/bin/git
ok  rg: found at /usr/local/bin/rg
ok  repository: /home/you/project
ok  config: settings valid
warn  key: endpoint opencode: no key found (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY); kirsch starts but cannot send a prompt
ok  backend: search_code uses ripgrep
ok  model: endpoint opencode, model minimax-m3: context window 128000 tokens
ok  context: AGENTS.md, 6854 bytes
ok  state: /home/you/.local/share/kirsch/sessions/3f9a1c07b2d4 exists and is writable
ok  lock: free
doctor: 9 ok, 1 warn, 0 fail
```

---

## git

```text
ok  git: found at <path>
```

`<path>` is the full path of the `git` program found on your `PATH`. Nothing
to do.

```text
fail  git: not found on PATH; kirsch needs git to find the repository
```

This appears when `git` is not on your `PATH` and you did not pass
`--workspace`. Kirsch will not start. Install Git, then open a new terminal so
the `PATH` change applies.

```text
warn  git: not found on PATH; kirsch starts with --workspace but the Git features are off, such as /diff, git_status, git_diff, the branch display and the dirty count
```

This appears when `git` is not on your `PATH` and you did pass
`--workspace`. Kirsch starts with its Git features off. Install Git to turn
them on.

---

## rg

```text
ok  rg: found at <path>
```

`<path>` is the full path of `rg`, which is ripgrep. Nothing to do.

```text
warn  rg: not found on PATH; search_code uses the pure-Go fallback
```

Kirsch's code search works without ripgrep. It uses its own built-in search
instead. This is a warning, not a failure. To use ripgrep, install it with
your system's package manager. See [install.md](install.md#ripgrep-optional).
The `search_code` part of the message is explained under
[Missing ripgrep](#missing-ripgrep) below.

---

## repository

```text
ok  repository: <root>
```

`<root>` is the folder Kirsch treats as the project root. That is the
enclosing Git repository, or the folder you named with `--workspace`.

```text
fail  repository: not in a Git repository
```

This appears when you run doctor outside a Git repository and did not pass
`--workspace`. Kirsch will not start. See
[Not in a repository](#not-in-a-repository) below.

```text
fail  repository: --workspace <path>: <error>
```

This appears when the folder named with `--workspace` cannot be used. The
detail is the error from the operating system, with the path shown in front
of it. For example, for a folder that does not exist:

```text
fail  repository: --workspace /nonexistent-kirsch-path: stat /nonexistent-kirsch-path: no such file or directory
```

If the path is a file and not a folder, the detail is
`--workspace <path> is not a directory`. Give Kirsch the path of a folder
that exists.

A rarer detail, `resolve workspace root <root>: <error>`, can also appear.

```text
warn  repository: <root> is not a Git repository; kirsch starts, but the Git features are off, such as /diff, git_status, git_diff, the branch display and the dirty count
```

This appears when `--workspace` names an existing folder that is not a Git
repository. Kirsch starts with its Git features off. `<root>` is that folder.

When `git` is missing, doctor leaves out the repository line, because the
`git` line already reports the cause.

---

## config

Kirsch reads a global settings file and a project settings file. The global file is
`$XDG_CONFIG_HOME/kirsch/config.toml`, or `~/.config/kirsch/config.toml` when
`XDG_CONFIG_HOME` is not set. The project file is `.kirsch/config.toml` in the
workspace. Each is described in [configuration.md](configuration.md).

The config check prints a `warn` line for each warning the loader raises.
It then prints either a `fail` line or an `ok` line.

```text
warn  config: <file>: unknown key "<key>" (ignored)
```

`<file>` is the config file the key is in. `<key>` is a setting Kirsch does not
know. It may be a typo, or a setting that a newer Kirsch understands. Kirsch
ignores it. Correct the spelling if the setting was meant to take effect.

```text
warn  config: <file>: key "<key>" ignored (a project config may set only [context].project_files)
```

This appears when the project file sets a key other than
`context.project_files`. Kirsch ignores the key. A project file may set only
that setting. To change any other setting, put it in your global file. The
same text appears when the project file holds a table Kirsch does not
recognise. In that case `<key>` is the table's name.

```text
warn  config: <file>: context.project_files must be a list of strings; ignored
```

This appears when `context.project_files` in the project file is not a list
of strings. Kirsch ignores the setting. Write it as a list, for example
`project_files = ["AGENTS.md"]`.

```text
fail  config: read <path>: <error>
fail  config: parse <path>: <error>
```

`read` appears when a settings file exists but cannot be read, usually
because of its permissions. `parse` appears when a file is not valid TOML.
`<path>` is the settings file's path. `<error>` is the operating system's message for `read`, and the parser's message for `parse`.
A type error also reaches you through
this line. For example, a `provider.default` that is not a string is reported
as `parse <path>: provider.default must be a string`. Fix the named file.

```text
fail  config: <file>: key "<key>" looks like a credential; Kirsch never reads secrets from config files, because config files get committed and shared. Set the value in an environment variable and name that variable with `api_key_env` instead.
```

This appears when a settings file holds a key whose name looks like a
credential, such as `api_key` or `token`. Kirsch refuses to start. Remove the
key from the file. Put the value in an environment variable, and name that
variable with `api_key_env`. The details are in [Secrets](configuration.md#secrets).

The message is longer than one line. Doctor joins its parts onto one line.

```text
fail  config: invalid settings (<consulted>): <error>
```

This appears when the settings parse, but the combined result is not valid.
`<consulted>` names the files that were checked: `global <path>` for the global
file, and `project <path>` for the project file when there is a workspace. It
reads `no config files` when no path is set. `<error>` is a message from the list below.

- `provider.default = "<name>"; endpoint does not exist`. Set
  `provider.default` to the name of an endpoint defined under `[provider]`.
- `provider.<name>.base_url is required`, and the same form for each other
  required field. Only an endpoint you define needs these, because
  the built-in endpoints supply them.
- `provider.<name>.auth = "<value>"; want x-api-key or bearer`.
- `provider.<name>.thinking = "<value>"; want one of off, low, medium, high`.
- `provider.<name>.base_url: <reason>`. The reason says which rule the
  address breaks, for example `base_url scheme must be https or http, got
  "ftp"` or `http is allowed only on localhost or loopback addresses`. The
  rules are in the Valid values column of `provider.<name>.base_url` in
  [configuration.md](configuration.md#provider).
- `policy.default_command_timeout_seconds = <n>; want a positive number`.
- `context.max_project_context_bytes = <n>; want zero or more`.

`<name>` in these messages is the endpoint's name, such as `opencode`.
`<value>` and `<n>` are the setting's value as you wrote it.

```text
ok  config: settings valid
ok  config: settings valid; project file not read (no workspace)
```

`settings valid` means the files parsed and the combined settings pass the
checks. Doctor cannot tell whether a given file was found, so the line does
not say. The `project file not read` form appears when no workspace was found, so the project
file was never read.

Config warnings appear in doctor's report. They do not appear in the program
unless you turn on the debug log. See [Debug log](configuration.md#debug-log).

---

## key

The key check reads the variable named by `api_key_env` for the default
endpoint. It prints the variable's name, and never the key itself.

```text
ok  key: endpoint <endpoint>: key found in <variable>
```

`<endpoint>` is the endpoint, `opencode` by default. `<variable>` is the
variable that supplied the key. It is `KIRSCH_<NAME>` when that variable is set,
and `<NAME>` otherwise. `<NAME>` is the endpoint's `api_key_env` value, which is
`OPENCODE_API_KEY` for the default endpoint.

```text
warn  key: endpoint <endpoint>: no key found (set KIRSCH_<NAME> or <NAME>); kirsch starts but cannot send a prompt
```

This is the line from the example report above, with `<endpoint>` as
`opencode` and `<NAME>` as `OPENCODE_API_KEY`. Kirsch starts, but it cannot send a prompt. The error you
see when you send one is described under [No API key](#no-api-key). Set the
variable as described there.

```text
warn  key: endpoint <endpoint>: api_key_env does not look like an environment variable name (value withheld); fix the setting
```

This appears when `api_key_env` holds something that is not a variable name.
A common cause is a key pasted into the setting by mistake. Doctor withholds
the value, so the line does not show it. Set `api_key_env` to a variable name
such as `OPENCODE_API_KEY`. If the value you pasted was a real key, treat that
key as exposed, and replace it at the provider. A settings file may have
been committed or shared.

```text
warn  key: skipped: config did not load
```

The config did not load, so there is no endpoint to check. Fix the `config`
line first.

---

## backend

```text
ok  backend: search_code uses ripgrep
```

Kirsch's `search_code` tool, which the model uses to search your code, is
using ripgrep. Nothing to do.

```text
warn  backend: search_code uses pure-go (rg is not on PATH)
```

`search_code` is using Kirsch's built-in search, because ripgrep is not on
your `PATH`. Search works. See [Missing ripgrep](#missing-ripgrep).

This line is never skipped. It does not read the config.

---

## model

```text
ok  model: endpoint <endpoint>, model <id>: context window <tokens> tokens
```

`<endpoint>` is the endpoint's name. `<id>` is the model's name as Kirsch's model table records it. That can
differ from the name in your config. `<tokens>` is the context window Kirsch
assumes for the model. Nothing to do.

```text
warn  model: endpoint <endpoint>, model <model>: not in the table; conservative fallback, context window <tokens> tokens
```

`<model>` is the model name from your config. Kirsch does not know this
model, so it uses a conservative context window. The program shows the same
situation as `Unknown model: cost display unavailable; conservative budget in use.`
The warning does not stop Kirsch. Check the model name in your config for a typo. If the name is correct, the warning remains.

```text
warn  model: skipped: config did not load
```

The config did not load, so there is no model to check. Fix the `config` line
first.

---

## context

The context check looks for the project files Kirsch reads into the model's
instructions. The default list is `AGENTS.md` and `CLAUDE.md`. See
[configuration.md](configuration.md#context).

```text
ok  context: <file>, <size> bytes
```

`<file>` is the first entry in `context.project_files` that is an ordinary
file in the workspace. `<size>` is its size on disk in bytes.

```text
ok  context: none; looked for <files>
```

None of the listed files is present. `<files>` is the list from
`context.project_files`, separated by commas. Kirsch runs without project
context. This is not a warning.

```text
ok  context: none; no project files configured
```

`context.project_files` is an empty list, so there is nothing to look for.

```text
warn  context: <file>, <size> bytes; <warning>
```

A file was chosen, but a candidate in the list was skipped. `<warning>` is one
from the list below. More than one can appear. Kirsch reads the
chosen file.

- `project context candidate "<name>" refused: <reason>`. Kirsch refuses a
  name that matches a secrets pattern such as `.env`, `*.pem` or `*.key`, a
  path inside `.git` or `.kirsch`, a path that leads outside the workspace, an
  absolute path, and an empty path. `<reason>` says which.
- `project context candidate "<name>" is not a regular file`. The candidate
  is a folder or another special item.

```text
warn  context: a candidate could not be read: <error>
```

Kirsch could not read a candidate, so it chose none. `<error>` is one of
the forms below, and any warnings from above follow it:
`stat project context candidate "<name>": <error>` or
`read project context candidate "<name>": <error>`.

Fix a context problem by correcting the name in `context.project_files`, then
checking that the file exists, is an ordinary file, and sits inside the
workspace. A path in `context.project_files` must be relative to the
workspace root.

```text
warn  context: skipped: config did not load
warn  context: skipped: no workspace
```

The `skipped: config did not load` form means the config did not load. The `skipped: no workspace` form means no workspace
was found, so there is no folder to look in. Fix the `config` or `repository`
line first.

---

## state

The state check looks at the folder where Kirsch saves sessions. That folder
is `session.storage_dir` in your config, and the default is
`~/.local/share/kirsch/sessions`. Kirsch keeps one folder for each project
inside the `sessions` folder, and keeps `index.json` in the folder above it.
The check looks at the project's folder, then at the folder above `sessions`
and its `index.json` file. See [configuration.md](configuration.md#session).

The `ok` lines say the folder is usable:

```text
ok  state: <dir> exists and is writable
ok  state: <dir> does not exist yet; kirsch will create it (<ancestor> is writable)
```

`<dir>` is the project's session folder. `<ancestor>` is the nearest folder
above it that already exists. A first run usually prints this form,
because Kirsch has not created the folder for that project. That is
normal.

The `warn` lines say the check could not be made on this system, or that
the folder has not been made. Kirsch starts.

```text
warn  state: <dir> exists; writability not checked on this platform
warn  state: <dir> does not exist yet; writability of <ancestor> not checked on this platform
warn  state: <folder> exists; writability not checked on this platform
warn  state: <index> exists; readability not checked on this platform
```

`<folder>` is the folder above `sessions`. `<index>` is `index.json` inside it.

The state check can also print these lines, when an earlier check left it
nothing to read:

```text
warn  state: skipped: config did not load
warn  state: skipped: no workspace
```

Fix the `config` line, or the `repository` line, first.

The `fail` lines are the ones to fix. Each one ends with a reason Kirsch
cannot start, or cannot create a new session:

```text
fail  state: cannot work out the state folder: home directory unknown; kirsch cannot start until this is fixed
```

Kirsch cannot find your home folder, and the session folder is under it.
Check that the `HOME` variable is set in the shell you start Kirsch from.
The `lock` line repeats the first part of the message, as a warning.

```text
fail  state: <dir> is not a folder; kirsch cannot start until this is fixed
```

Something that is not a folder sits where the session folder should be.
Rename that file, so Kirsch can make the folder.

```text
fail  state: <dir> is not writable; kirsch cannot create a new session, so any start that needs one fails until this is fixed
```

You do not have write access to the session folder. Give yourself write access
to it, or change `session.storage_dir` to a folder you can write to.

```text
fail  state: cannot read <dir>: <error>; kirsch cannot start until this is fixed
```

Kirsch cannot read the session folder. `<error>` is the operating system's
message, with the path left out. Check the folder's permissions.

```text
fail  state: <dir> does not exist and <ancestor> is not a folder; kirsch cannot start until this is fixed
fail  state: <dir> does not exist and <ancestor> is not writable; kirsch cannot start until this is fixed
fail  state: <dir> does not exist and <ancestor> cannot be read: <error>; kirsch cannot start until this is fixed
```

The session folder does not exist, and Kirsch cannot create it, because the
nearest folder above it cannot take a new folder.
Fix that folder, or change `session.storage_dir`.

```text
fail  state: cannot read <folder>: <error>; kirsch cannot start until this is fixed
fail  state: <folder> is not a folder; kirsch cannot start until this is fixed
fail  state: <folder> is not writable, so index.json cannot be updated; kirsch cannot create a new session, so any start that needs one fails until this is fixed
fail  state: cannot read <index>: <error>; kirsch cannot start until this is fixed
fail  state: <index> is a folder, not a file; kirsch cannot start until this is fixed
fail  state: <index> is not readable; kirsch cannot start until this is fixed
```

These cover the storage root and its `index.json` file, which Kirsch reads
and writes when it starts a session. Fix the named folder or file: give it the
permissions it needs, or move the item that is in its way. Rename rather
than delete a file you are unsure of.

---

## lock

The lock check asks whether another Kirsch is running in the same project.
Kirsch guards each project with a lock, so a second copy starts
with fewer features. See [Lock held](#lock-held) below.

```text
ok  lock: free
ok  lock: free (no lock file yet)
```

No other Kirsch holds the lock for this project. The `(no lock file yet)` form
appears before the lock file has been created. Each is normal.

```text
warn  lock: held by another kirsch in this workspace; a new start would skip resume and session grants
```

Another Kirsch is running in this project. A new start would skip resume and
session grants. Doctor does not disturb the other copy. See
[Lock held](#lock-held).

```text
warn  lock: free, but <lockPath> is not writable; the single-instance guard will be off
```

`<lockPath>` is the `.lock` file in the project's session folder. Kirsch can
take the lock only if it can write to that file. Otherwise it runs without
the single-instance guard. Give yourself write access to the file.

```text
warn  lock: could not check the lock: <error>
warn  lock: not checked on this platform
warn  lock: skipped: config did not load
warn  lock: skipped: no workspace
```

These mean the lock could not be checked, so the state is unknown. The
`could not check` form gives the operating system's message. The `skipped` forms have the
same meaning as on the other checks above.

---

## Common failures

### No API key

**What you see.** Kirsch opens, but it cannot send a prompt. When you send
one, an error card titled `auth` reads:

```text
no API key for endpoint "opencode" (set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY)
```

On a fresh start, the onboarding screen also shows this, with the variable
names. It stays while the transcript holds only notices, such as `no previous
session — starting fresh`, and every notice fits beneath it. A resumed session
that already has messages in its transcript does not show the onboarding, so
check the key with `kirsch doctor`, below, instead.

**What to do.** Set the variable in the shell you start Kirsch from. Use the
bare name, or the prefixed one if you already use it for other tools:

```bash
export OPENCODE_API_KEY="<your-opencode-key>"
```

[install.md](install.md#set-up-your-api-key) has the details. Run
`kirsch doctor` to confirm: the `key` line should read `key found in`.

### Not in a repository

**What you see.** When you start Kirsch outside a Git repository, and without
`--workspace`, it stops before the screen opens. Kirsch prints its own message
after the program name:

```text
kirsch: Kirsch runs inside a Git repository.

Either cd into a repository, or point Kirsch at a directory explicitly:
    kirsch --workspace /path/to/project
```

This is the output of a run from an empty folder. Kirsch exits with status `1`.

`doctor` says the same thing in its own words:
`fail  repository: not in a Git repository` (see [repository](#repository)).

**What to do.** Either `cd` into the repository, or pass the folder with
`--workspace`:

```bash
kirsch --workspace /path/to/project
```

With `--workspace`, a folder that is not a Git repository works. Kirsch
starts with its Git features off. Doctor shows this as a `warn` line for the
`repository` check.

If you are sure you are inside a repository, check that
`git rev-parse --show-toplevel` works there. Kirsch treats any failure of that
command as 'not in a repository'.

### 401 or 403: rejected the API key

**What you see.** A prompt ends in an error that names the endpoint and the
status. Kirsch builds this text for a 401 and for a 403:

```text
provider "<endpoint>" rejected the API key (HTTP <status>); set KIRSCH_<NAME> or <NAME>
```

`<endpoint>` is the endpoint's name, `opencode` by default. `<status>` is
`401` or `403`. `<NAME>` is the endpoint's `api_key_env` value. The program
shows it in an error card titled `turn_error`, with extra text in
front of that message. For the default endpoint the card reads:

```text
agent: model stream: provider_error: auth: provider "opencode" rejected the API key (HTTP 401); set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY
```

**Why it can mislead.** The same message appears for a 401 and for a 403, whatever
the cause. The key is not always the problem. The endpoint may be
wrong, or the account may not have permission to use the model. The message
gives no way to tell these apart. So check in this order:

1. **The key.** Run `kirsch doctor`, and check the `key` line. Then confirm
   the variable is set in the shell you start Kirsch from, and that the value
   is the key you meant.
2. **The endpoint.** Check `base_url` and `auth` for the endpoint in your
   config. See [configuration.md](configuration.md#key-reference).
3. **The model.** Check the `model` setting. Your account may not allow the
   model you named.

Doctor cannot show a `401` or `403`, because it makes no network calls and
never sends a prompt.

### Missing ripgrep

**What happens.** Without ripgrep, `search_code` uses Kirsch's own search.
Search works. If ripgrep is on your `PATH` but fails during a search,
Kirsch also falls back to its own search, rather than failing the tool.

**What doctor shows.** Warnings, which do not fail doctor:

```text
warn  rg: not found on PATH; search_code uses the pure-Go fallback
warn  backend: search_code uses pure-go (rg is not on PATH)
```

These are the lines from the `rg` and `backend` checks.

**What to do.** Install ripgrep with your system's package manager. See
[install.md](install.md#ripgrep-optional).

### Lock held

**What you see.** A second Kirsch in the same project does not resume the
last session, and it turns off session grants. The program shows these
messages. Each line reads as follows:

- In the status bar, as a persistent warning: `another Kirsch is running here`
- When it would resume on launch: `auto-resume skipped — another Kirsch instance is running`
  (the notice that starts a new session in its place)
- When you run `kirsch resume`: `resume skipped — another Kirsch instance is running`

Session grants are switched off in this copy. The policy setting that
controls them is explained in
[configuration.md](configuration.md#policy).

If the lock cannot be taken for another reason, the status bar shows
`lock unavailable — single-instance guard off`.
In this case the guard is off, and Kirsch runs as normal.

**What to do.** If you want resume and grants in this project, close the
other Kirsch and start again. If you see the warning and expected no other
Kirsch to be running, look for one in another terminal. Running
`kirsch doctor` does not take the lock from a copy that is already running.

---

## If the report does not explain it

Run `kirsch doctor` and read the `fail` lines first. Fix each one, then run
doctor again. A `warn` line does not stop Kirsch, but it often explains
behaviour that looks wrong.

Config and project-context warnings do not show in the program. To see them,
turn on the debug log for one run with `kirsch --debug`, or with
`KIRSCH_DEBUG=1`. The log is written under `$XDG_STATE_HOME/kirsch/debug.log`,
or `~/.local/state/kirsch/debug.log`. See
[Debug log](configuration.md#debug-log).

---

## SOURCE: SECURITY.md

# Security Policy

## Supported versions

Only the latest release of Kirsch receives security fixes. If you run an older
release, upgrade to the latest one before you report a problem with it.

## Reporting a vulnerability

Report security problems privately. Do not open a public issue or pull request
for them.

Use GitHub's private vulnerability reporting form:

https://github.com/djm56/kirsch/security/advisories/new

You can reach the form from the Security tab of the repository. Include the
version you ran, your operating system, and the steps that reproduce the
problem.

We aim to reply within 7 days.

## Threat model

Kirsch is a terminal coding assistant. It reads the repository you open, sends
your prompts and tool results to the model provider you configure, and can
change files and run commands in that repository. The things worth protecting
are:

- the files in your workspace, including `.env` files and private keys;
- your API keys and tokens;
- your user account, which any command Kirsch runs would have.

The controls below are the ones Kirsch relies on. Some of them have known
limits, which the next section lists.

- **Changes need your approval.** The file tools apply a patch only after you
  approve it.
- **Commands need a policy decision.** By default, a command runs only if it is
  on the built-in allowlist, a session grant you gave covers it, or you approve
  it. The approval prompt shows the full command. Kirsch runs the command
  directly, without a shell.
- **Paths stay inside the workspace.** Kirsch refuses paths outside the
  workspace root, including paths that reach outside through a symlink. It also
  refuses `.env` files, key files such as `*.pem`, and the `.git/` and `.kirsch/`
  directories. The match ignores case.
- **Commands do not receive secret-named variables.** Before a command runs,
  Kirsch removes environment variables whose names end in `_TOKEN`, `_KEY` or
  `_SECRET`, or start with `AWS_`. It removes them even when they are on the
  passthrough list.
- **Credentials stay out of config files.** Kirsch refuses to load a config file
  that holds a credential-shaped key. A project's config file can set only the
  list of context files. Its other settings, including the debug log, the
  providers and the policy, are ignored.
- **Remote endpoints are checked.** A base URL that uses plain `http` must point
  at a loopback address. A base URL that carries a user name or password is
  refused. Kirsch does not follow redirects from the provider.

## Known limits

- **A command that runs can write files.** Your approval, or the built-in
  allowlist, is the only check on what a command does. The allowlist runs
  `go build`, `go test`, `git diff`, `git log` and `ls` without asking, each
  exactly as written with nothing after it, and `go test` runs the workspace's
  own test code. A command that runs can write anywhere your account can,
  including the `.kirsch/` directory and the global configuration directory. The
  file tools refuse those paths, but a shell command is not limited that way.
- **Commands run with your privileges.** Kirsch does not sandbox them. The
  allowlist and the approval prompt are the controls.
- **Command approval can be turned off.** Setting
  `policy.require_approval_for_commands` to `false` in your global configuration
  lets every command run without asking, except shells and runners (`sh`,
  `bash`, `zsh`, `dash`, `env`, `xargs` and `nohup`), which always ask. Other
  shells and interpreters, such as `fish` or `python`, are not in that set. A
  project's configuration cannot change this setting.
- **Environment filtering works on names.** A variable whose name does not match
  one of the patterns above is passed through when it is on the passthrough list,
  even if its value is a secret. The patterns are matched case-sensitively, so a
  lowercase name such as `my_token` is not removed.
- **The debug log can hold your messages, paths, commands and, on a 400, the
  whole conversation.** The debug log is off unless you turn it on with
  `--debug`, with `KIRSCH_DEBUG=1`, or with `telemetry.debug_log` in your global
  configuration. A project's configuration cannot turn it on. When it is on, the
  log writes the text of each message you send, in full. It also writes the
  workspace path at startup, and the paths and key names in a configuration
  warning. It writes the arguments of a command when you approve it for the
  session and the grant is refused, and the saved pattern of a grant that the
  policy refuses to restore. It writes the names of context files it refuses to
  load, and the name of a context file whose read or stat fails. It writes
  the base name of a usage file it moves aside after corruption. It writes
  the error text of a session, session index, usage or lock file that cannot be
  opened, read or written, and that text can name a file path. The log does not
  record each tool call or each slash command. Redaction replaces a value only
  when its attribute name contains one of `api_key`, `apikey`, `token`, `secret`,
  `password` or `env_value`, matched without regard to case, and it checks no
  other text. So a secret you type into a message, or that appears in a path
  or command, is logged.

  On a 400 response from the provider, the debug log records the
  full request body. This is a disclosed limit, and the body is not clipped. It
  is the request sent to the model: the system prompt, your messages, the tool
  definitions, the model's earlier `thinking` blocks, the tool calls in the
  conversation (for example commands and patches), and their results (for
  example file contents and command output). A 400 whose error type is
  `MissingSessionID` is the exception: Kirsch reports it as a configuration
  error and does not log the request body. A response body longer than 256
  bytes is cut to its first 256 bytes, followed by `…(truncated)`. Keep the
  debug log off when you work on material you cannot share.
- **Windows is not supported.** Process-group handling is POSIX-only.
- **Dependencies are pinned, not attested.** The module checksums in `go.sum`
  pin each dependency. Kirsch does not verify where a dependency came from, and
  its releases carry no supply-chain attestation.
