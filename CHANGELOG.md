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
- **The screen reference is executable.** All thirteen character grids in
  `plan/kirsch-ui-screens.md` are parsed out of the markdown and compared
  against `View()` byte-for-byte on every test run, so a rendering change and a
  stale design document cannot drift apart.

### Changed

- Corrected the render targets before building against them: five geometry
  defects in the screen reference, two arithmetically impossible height bands,
  a conflated spinner glyph, and an escape-sequence assertion that could never
  hold. Recorded as plan amendments 25–32.
- `Alt+Enter` replaces `Shift+Enter` as the primary newline binding: Bubble Tea
  v1 carries no shift modifier, so the original binding was unimplementable.

[Unreleased]: https://github.com/djm56/kirsch/commits/main
