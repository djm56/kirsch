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

[Unreleased]: https://github.com/djm56/kirsch/commits/main
