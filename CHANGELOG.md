# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Planning baseline.** The locked v0.1 spec (`plan/kirsch-plan.md`), the
  architecture rationale, the TUI spec, a screen-by-screen render reference,
  and ADRs 0001–0007.
- **Milestone 0 — repo bootstrap and static TUI prototype.** Go module, CI, and
  `internal/tui` rendering header, transcript, tool and approval cards, modals,
  status bar and composer against fake data. The full mode state machine, card
  selection and expansion with the 200-line cap, approval flow, scroll/pin
  behaviour, text sanitisation, and both the no-colour and ASCII fallbacks. No
  LLM, filesystem or subprocess code exists in the binary.
- **Milestone 1 — workspace engine and read-only tools.** Git root detection
  with a `--workspace` override, canonical-path containment, the path denylist,
  `.gitignore`-aware walking and project-type detection (`internal/workspace`);
  TOML configuration with four-layer precedence and credential refusal
  (`internal/config`); a structured debug log that never touches stdout or
  stderr (`internal/telemetry`); the tool envelope and registry
  (`internal/tool`) with `read_file`, `list_files`, `search_code`, `git_status`
  and `git_diff`; and the wiring layer (`internal/app`) that lets the TUI drive
  them without importing them. Six fixture repositories under `testdata/`.
- **`search_code` has two backends that provably agree.** ripgrep when it is on
  `PATH`, a pure-Go walk otherwise, with a differential test over nine queries
  asserting byte-identical output.
- **Plan §2's dependency rules are enforced by a test in CI**, not by review —
  in place before the packages it guards exist.
- **The screen reference is executable.** All thirteen character grids in
  `plan/kirsch-ui-screens.md` are parsed out of the markdown and compared
  against `View()` byte-for-byte on every test run, so a rendering change and a
  stale design document cannot drift apart.

- **Instruction sets for Milestones 2, 3 and 4**, at deliberately decreasing
  fidelity (plan §11 amendment 50): M2 executable as written, M3 settled in
  shape but provisional in detail, M4 a scope statement identifying the hard
  problems. M5 is left unwritten.
- **`/exit`**, an alias of `/quit` sharing one arm rather than a second
  behaviour, and tab-completable like every other command.
- **`scripts/go-tool.sh`**, which every npm script now goes through to reach a
  Go tool. `go install` writes into a directory that is not on the PATH of the
  non-interactive shell npm hands a script — so `npm run check`
  used to die at the lint stage with `sh: golangci-lint: command not found`,
  and a missing tool read as a failing check. It takes a copy already on PATH
  first, falls back to `GOBIN` or `GOPATH/bin`, names the command that installs
  a tool that genuinely is not there, and refuses a `golangci-lint` it can
  identify as older than v2 rather than letting it fail on a flag error,
  reporting the version it found beside the version needed. A version string it
  cannot parse warns and carries on: an unreadable version is not evidence of a
  wrong one, and the tool's own exit code still decides the check.
- **`scripts/go-bin-dir.sh`**, which answers the one question both of the other
  scripts need — the directory `go install` writes to, which is `GOBIN` when set
  and `GOPATH/bin` otherwise. They each used to answer it themselves and
  disagreed: `install-tools.sh` assumed `GOPATH/bin` unconditionally, so with
  `GOBIN` set it reported the binary it had just written as a foreign copy
  shadowing the install and advised removing it. It asks `go` rather than
  reading the environment, because both variables can come from the `go env`
  config file rather than from an export.
- **`npm run lint:fix`** — `golangci-lint run --fix`, the auto-fixable subset of
  the lint findings, beside the formatting `npm run fmt` already applies.

### Changed

- Corrected the render targets before building against them: five geometry
  defects in the screen reference, two arithmetically impossible height bands,
  a conflated spinner glyph, and an escape-sequence assertion that could never
  hold. Recorded as plan amendments 25–32.
- Newline binding settled by testing on real terminals rather than by reading
  the toolkit's key table: `Shift+Enter` works wherever the terminal emits
  `ESC`+`CR`, Option+Enter on a Mac produces nothing, and `Ctrl+J` always
  works. The composer accepts the sequence rather than the key name.
- Palette retuned for legibility on dark backgrounds. The original `dim` (2.3:1)
  and `border` (1.7:1) were below the threshold at which anything is readable,
  so placeholders, onboarding suggestions and every separator rule appeared as
  washed-out grey. Every foreground token now clears 3:1 on black, `#1e1e1e`,
  One Dark and Nord; every token carrying words clears 4.5:1.
- **Key bindings corrected by the first operator walkthrough of
  `plan/testing/manual-milestone-0.md`.** `Esc` no longer re-pins the transcript,
  which had made "typing does not re-pin" unreachable — returning to the composer
  undid the scroll. Submitting a slash command now re-pins; the two hint-only
  paths do not. `Home`/`End` are decoded where terminals send them as SS3, and
  `g`/`G` are bound in the transcript pane beside them. Closing an overlay
  restores full colour on the same keypress instead of needing a second `Esc`.
  Recorded as plan amendments 55–59.
- The scripted Milestone 0 turn now holds its `run_command` card visibly in the
  running state and ends on two approvals — a patch, then a command — so the
  `◐` glyph and the `[a]` session-grant row are both reachable by a person and
  not only by the golden tests.
- Help overlay grew to 31 rows; screen 06 in `plan/kirsch-ui-screens.md` redrawn
  at 80×31.
- **`npm run fmt` has changed meaning, and every contributor's local formatting
  step changes with it.** It was `gofmt -w .`. It is now `golangci-lint fmt`,
  which applies gofmt, gofumpt and goimports from the one binary, sorting
  `github.com/djm56/kirsch` imports into their own group after the third-party
  block. `npm run fmt:check` reports the same set as a diff without writing,
  and `golangci-lint run` now fails on a gofumpt violation too — so a tree that
  satisfies `gofmt` alone is no longer formatted. `AGENTS.md` had been stating
  the old bar, which would have given a contributor a green local check and a
  red CI; it now states this one.

### Removed

- **The bare `q` quit binding.** A composer where `q` on an empty line quits is
  one where beginning a message with a word starting in `q` quits instantly.
  `q` is now an ordinary character; quitting is `/quit`, the new `/exit` alias,
  or `Ctrl+C` twice.

### Fixed

- **The CI lint job, which had been failing on every run on `main`.** The
  pairing of `golangci-lint-action@v6` with `golangci-lint v1.62.2` never
  reached `.golangci.yml`: a linter built with go1.23 refuses a module
  targeting go1.25, so the job died on the Go version gate before any rule was
  read — and a v1 binary could not have read a `version: "2"` config had it got
  that far. The action is now v9, the linter pin v2.13.2. Both are concrete
  versions rather than floating tags, so a linter release cannot turn a green
  branch red on its own.

[Unreleased]: https://github.com/djm56/kirsch/commits/main
