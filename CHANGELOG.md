# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Planning baseline.** The locked v0.1 spec (`plan/spec/kirsch-plan.md`), the
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
  `plan/spec/kirsch-ui-screens.md` are parsed out of the markdown and compared
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
- **`internal/patch` — unified-diff parser, renderer, and atomic applier.**
  Standard library only, no in-module imports, so the parser stays a leaf and
  error mapping onto tool kinds belongs to the tool layer later. Parsing handles
  `git diff` output and hand-written header-less diffs, including multi-file
  header-less input; validates hunk arithmetic at parse time; preserves `\ No
  newline at end of file` in both directions; detects binary diffs rather than
  failing on them; honours `100644` ↔ `100755` mode changes and rejects any
  other; and leaves paths untouched beyond stripping `a/` and `b/` prefixes, so
  containment stays the workspace's job. Round-trip re-render is byte-for-byte
  on every well-formed corpus file, including two carrying CRLF — the renderer
  preserves carriage returns inside hunk content while keeping metadata lines
  LF-only. Applying is all-or-nothing: content is staged to temp files beside
  their targets, every file validated before any is committed, each target moved
  aside to a backup before its replacement lands, and every committed entry
  restored from that backup if a later one fails. Where a restore cannot
  complete, the error names each path left inconsistent and where its content
  now sits, rather than reporting a rollback that did not happen. Strict context
  matching with zero fuzz — a conflict returns the offending hunk and the actual
  file content at that location, enabling a model to re-read and retry rather
  than guess. Line endings are detected by dominance: a mixed-ending file is
  rewritten wholesale in its dominant ending (a tie becomes LF), and a patch
  that would convert endings is rejected. Trailing newlines are honoured, with
  two suspected edge cases open (`plan/milestones/milestone-2.md`, *Open
  defects*). Permission bits survive except a deliberate exec-bit change; a
  rename with edits keeps only the source's `Perm()` (no setuid, setgid or
  sticky). Test
  fixtures: `testdata/repo-patch/` (seven files across LF, CRLF,
  no-trailing-newline, executable, nested, multi-byte and binary) and
  `testdata/patches/` (twenty diffs, each named for the behaviour it pins). The
  rollback path is exercised by injected commit failures rather than only by
  inspection — that distinction cost this deliverable four fix rounds and is the
  thing most worth recording.
- **`internal/policy` — the command allowlist and session grants.** The default
  allowlist is `go build`, `go test`, `git diff`, `git log` and `ls`, each
  matched as that exact argv. Shells and anything whose job is to run another
  program — `sh`, `bash`, `zsh`, `dash`, `env`, `xargs`, `nohup`, matched by
  basename — always ask. `ForPatch` never allows, and no configuration changes
  that. A session grant records an argv prefix; it is refused for a bare
  wildcard, an empty prefix or a shell. The `allow_session_scoped_grants`,
  `require_approval_for_patches` and `require_approval_for_commands` settings
  are not yet read by the policy,
  so grants are always available, patches always ask, and a command asks unless it is allowlisted or granted.
- **The approval flow.** `App.Request` blocks the tool goroutine on a capacity-1
  decision channel; `Resolve`, called from `Update`, never blocks; the first
  answer on the channel wins; cancelling the turn resolves a pending approval as
  cancelled within a second. A rejected patch or command comes back to the
  caller as a `policy_denied` tool result.
- **`apply_patch` and `run_command`, the first tools that change the world.**
  `apply_patch` parses the diff, resolves every path through the workspace and
  dry-runs the whole patch before it asks, so a patch that cannot apply never
  produces a prompt. `run_command` takes an argv, never a shell string; builds
  its environment from `PATH`, `HOME`, `LANG` and the configured passthrough,
  then strips secret-shaped names; gives the child `/dev/null` for stdin; caps
  output at 200KB, head and tail; and kills the whole process group on timeout
  or cancellation. Its output appears in the card when the command finishes. Not
  yet wired: the `policy.default_command_timeout_seconds` setting (an omitted
  timeout defaults to 3600s with no maximum) and live output streaming.
- **The approval surface in the TUI.** Approval cards render real requests;
  `[a]` appears only when policy says a session grant is possible; the diff
  modal shows the parsed diff with per-file `+n −m` counts; `/approvals` lists
  grants and clears them through a confirm prompt; and two temporary debug
  commands, `/patch` and `/run`, sit under `debug (M1–M2)` in `/help`.
- **Milestone 3, m3-d1 — provider interface and fake.** `internal/provider` holds
  the provider-neutral contract the agent loop is tested against: `Provider`,
  request and message types (text, thinking with an opaque signature or redacted
  data, tool calls, tool results), streaming events, and the plan §5 model table
  with its unknown-model fallback. `provider.Fake` replays scripted turns — text,
  tool calls, thinking, mid-stream errors and a turn that blocks until cancelled —
  and records deep copies of every request, so tests can check what a second
  request echoed back. No vendor name appears in the package.
- **Milestone 3, m3-d2 — Messages adapter and endpoint config.** `internal/provider/anthropic`
  speaks the Messages wire format: request encoding, SSE decoding with event
  framing and bounded buffers, and a streaming HTTP client that refuses redirects,
  retries 5xx, network errors and transient 429s with `Retry-After` back-off,
  surfaces a usage-limit 429 as a lockout, fails a 401 or 403 at once, and logs a
  400's request body, never the key, to the debug log. Its tests replay the
  recorded probe streams. `internal/config` gains the endpoint map with two
  built-ins (`opencode`, the default, and `anthropic`), `base_url` validation
  (https, or http on a literal loopback host; userinfo and IPv4-mapped forms
  refused) and a project-file allowlist: only `[context].project_files` is
  honoured, every other key is ignored with a warning, and a credential-shaped
  key is refused. An opt-in live smoke test (`npm run test:live`, build tag
  `live`) sends one streamed request through the adapter; `KIRSCH_LIVE_MODEL`
  points it at another Messages-format model.
- **Milestone 3, m3-d3 (part 1) — the agent turn loop.** `internal/agent` declares provider-neutral message and event types and the `Model`, `Tools` and `Recorder` interfaces, and imports no implementation package. `Agent.Turn` appends the user's text, streams the model, runs any tool calls one after another in the order returned, and feeds the results back until the model answers. A message completes only on `MessageDone`, a protocol violation returns `ErrStreamProtocol`, and every error is wrapped. Payloads are copied, never kept by reference, and consecutive thinking blocks stay separate. A failed turn keeps everything that happened, and the next turn's text merges into a trailing user message; `Recorder` receives each append and merge so the conversation can be rebuilt exactly. The guards (short-circuit, retries, the 25-round guard, cancellation) are not built yet.

### Changed

- **The `opencode` default model is `minimax-m2.7`** (plan amendment 83). The endpoint refused `minimax-m3` for the operator's account on 2026-10-05, and `minimax-m2.7` passed the live smoke test through the adapter. The model table gains a known, flat-rate `minimax-m2.7` row; the `minimax-m3` row stays. The live smoke test allows 256 output tokens, because `minimax-m2.7` thinks before every answer.
- **Tool card preview mode.** Tool cards now open as a preview showing the first
  10 lines by default, collapsible to the head with `Enter`. When lines are hidden,
  a marker reads `‹10 of N lines — d full output · Enter collapse›`. `d` opens the
  full output in a modal. The 200-line inline expansion is gone. Empty output shows
  no box; the trailing empty output row is trimmed. The preview keeps the transcript
  legible while tools are running.
- **Approved approval cards fold away.** Once you approve a command or patch, the
  approval card folds away and the tool card's head shows ` · approved` or ` · approved
  for session`, plus `session grant: <scope>` when applicable. `d` on the folded patch
  card opens the diff. Rejected and cancelled approvals keep their own card. A refused
  session grant now confirms as approved (allow-once) because the command runs.
- **Composer history navigation.** Up/Down arrow at the first/last line of the composer
  now recalls previous/next entries from session history (up to 100 entries; consecutive
  duplicates and blank entries are skipped), rather than moving to the transcript.
  History survives `/new`. Shift+↑ and Tab move to the cards instead.
- **Tab in slash-command completion.** While typing a command name with no whitespace
  yet, Tab completes a unique slash-command prefix; an ambiguous or unknown prefix does
  nothing. In any other text, including an empty prompt, Tab moves to the cards.
- **Approval card release.** Any non-navigation key except y/a/n/d/? releases the card
  and shows `approval pending — Tab to return to the card`. While released, those keys
  do nothing until Tab or Shift+↑ re-arms the card. Navigation keys never release.
  Bracketed paste is dropped and does not release. Esc/Ctrl+C cancel whether armed or
  released.
- **Modal clipping marker.** When a modal body is clipped, its rule row shows
  `↑ N above` / `↓ N more` (ASCII `^`/`v`).
- **Key-hint row.** At terminal height ≥ 24, the last row is a dim key-hint line
  for the current focus (e.g. "↑↓ history · ⇧↑/Tab cards · /help" in composing
  mode). It is blank under a modal or confirm, so the frame never reflows. The full-help
  overlay minimum is now 80×34.
- **`read_file` summary.** Now shows `lines N-M` instead of repeating the path.
- **The header is two rows.** `Kirsch` and its rule on the first, the project
  name, branch, dirty marker and compaction note on the second. The single-row
  form spent most of a 40-column terminal on text rather than rule, and a long
  project name, a long branch and the compaction note all competed for the same
  row. Chrome goes from 5 rows with a header to 6. The §2.2 height bands are
  unchanged — the full-layout band has to keep starting at the advertised 40×10
  minimum — so the one visible consequence is that shrinking from 10 rows to 9
  makes the transcript *one row taller*, as the header goes as a unit and only
  four rows of chrome remain.
- **The frame keeps one blank column at its left and right edges**, and none at
  the top or bottom. The operator asked for padding on all four sides; a
  terminal row is not a window pixel, and the chrome budget already spends every
  row there is, so a blank row would come straight out of a transcript that is
  four rows tall at the supported minimum. The minimum terminal is still 40×10
  and the width bands are still read from the terminal width, so a 40-column
  terminal stays a full-layout terminal — with 38 content columns. The right
  column is *reserved, not written*: nothing can reach it, and emitting a space
  there would put trailing whitespace on every row of every frame without
  changing a single rendered cell.
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
  `plan/testing/manual/milestone-0.md`.** `Esc` no longer re-pins the transcript,
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
- Help overlay grew to 32 rows; screen 06 in `plan/spec/kirsch-ui-screens.md` is
  drawn at 80×32. Two causes: adding `/exit` took the body from 21 lines to 22,
  and the two-row header below cost one more.
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

- **Approving a command for the session froze Kirsch.** `app.Resolve` runs on
  the event loop and sent its follow-up messages with a blocking send that
  waited for that same loop. The whole sequence is now delivered from one
  goroutine, and `internal/app/deadlock_test.go` drives a real `tea.Program`
  through it.
- **A `git diff` of a CRLF file could not be applied.** The parser kept each
  hunk line's `\r` while the applier stripped it from the file, so every context
  line mismatched. Lines are now compared without the trailing `\r`, added lines
  are written in the file's own ending, and a patch that carries endings must
  add lines in the file's ending or is rejected.
- **A rename with edits left the new file at mode 0600.** The edited copy took
  its mode from a temp file; it now takes the source file's permission bits.
- **`/run` was hidden at the bottom of `/help`.** The Milestone 2 debug
  commands now lead their section, so `/run` shares a row with `/patch`, and a
  clipped help overlay's footer names the scroll keys.
- **Lint and format had been failing.** Thirteen `golangci-lint` findings and
  one formatting issue are fixed, none suppressed; one was a test whose
  condition repeated itself.
- **There was no way back to the bottom of the transcript from the composer.**
  Every re-pin reachable from Composing put something in the transcript first —
  sending a message, or submitting a slash command — so a reader who scrolled up
  and then decided not to send had no single key that returned the view; `↑` then
  `End` did it in two, by way of a mode they did not want. `Ctrl+G` is now bound
  in the composer and does exactly that: it moves the viewport and nothing else,
  leaving the composer's text, its hint, the card selection and the mode alone.
  It sits ahead of the busy guard, so it works **during a live turn**, which is
  when getting back to the bottom is worth the most. `End` and `Ctrl+End` were
  not candidates — bubbles binds both inside the text area — and bare `G` has to
  stay an ordinary character while typing. `End` and `G` still re-pin in Browsing
  mode, as they always did. The help overlay does not list `Ctrl+G` yet; its grid
  is a byte-for-byte test oracle, so that is a redraw, tracked in plan §11.
- **The onboarding tagline was cut with no truncation marker** between 40 and 42
  columns, reading as a word broken off mid-air. Its width is not a constant —
  the version is a build-time string, and `0.1.0-dev` already makes the row 41
  cells against the 38 a 40-column terminal leaves — so it was reaching `View()`'s
  final bound on the frame, which is a hard edge rather than an elision and
  carries no marker. It is truncated by its own renderer now, with the standard
  `⋯` (`...` in ASCII).
- **The CI lint job, which had been failing on every run on `main`.** The
  pairing of `golangci-lint-action@v6` with `golangci-lint v1.62.2` never
  reached `.golangci.yml`: a linter built with go1.23 refuses a module
  targeting go1.25, so the job died on the Go version gate before any rule was
  read — and a v1 binary could not have read a `version: "2"` config had it got
  that far. The action is now v9, the linter pin v2.13.2. Both are concrete
  versions rather than floating tags, so a linter release cannot turn a green
  branch red on its own.

### Security

- **Go toolchain pinned to 1.25.13.** `govulncheck` reported four standard-library
  advisories against go1.25.12, all fixed in go1.25.13: GO-2026-6218 (`net/url`),
  GO-2026-6090 (`crypto/tls`), GO-2026-5972 (`encoding/asn1`) and GO-2026-5026
  (`net/http`). Two were reachable from existing code (`internal/tool`,
  `internal/patch`) and two through `cmd/kirsch-probe`. `go.mod` now reads
  `go 1.25.13`; CI follows it through `go-version-file`.

[Unreleased]: https://github.com/djm56/kirsch/commits/main
