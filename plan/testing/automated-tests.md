# The automated test suite

What it covers, how it is organised, and — as importantly — what it
deliberately does not cover.

```bash
npm test              # go test ./...
npm run test:race     # the same under the race detector
npm run test:cover    # coverage summary
```

Roughly 300 assertions across 100 test functions, all of which run in a few
seconds without a network, an API key, or a terminal. Statement coverage is
about 73% overall — see "What is deliberately not covered" for where the gap
is and why most of it is intentional.

| Package | Tests | Fuzz | Cover | What it is protecting |
|---|---|---|---|---|
| `internal/workspace` | 25 | 2 | 85% | The containment boundary — the security core |
| `internal/tui` | 23 | 2 | 67% | Rendering, the mode machine, text sanitisation |
| `internal/tool` | 22 | 2 | 85% | The tool envelope, truncation, backend parity |
| `internal/app` | 12 | — | 84% | Wiring, cancellation, the TUI↔tool boundary |
| `internal/config` | 9 | — | 85% | Precedence, and refusing credentials |
| `internal/telemetry` | 7 | — | 92% | Never writing to stdout/stderr; redaction |
| `internal/arch` | 2 | — | — | The dependency rules, enforced not reviewed |

`internal/tui` is the low one, and deliberately so: a large share of its
statements are rendering branches for terminal sizes and capability
combinations that the golden states do not enumerate. The parts that carry
*logic* — the mode machine, scroll and pin, sanitisation — are covered
directly. Chasing the number by asserting on more rendered strings would add
brittleness rather than confidence.

## The tests that earn their keep

Most tests here are ordinary. These are the ones worth understanding, because
each encodes something that was got wrong once.

### `workspace`: the symlink table

`TestSymlinkTable` drives every row of the `repo-symlink-escape` fixture. **The
allowed rows matter as much as the blocked ones** — a containment check that
refuses legitimate internal symlinks is also a bug, and a much quieter one,
because nobody files a ticket saying "my link works fine".

It caught a real hole. The algorithm the instruction set specified —
`EvalSymlinks` the whole path, then prefix-check — lets a symlink pointing
*outside* the workspace at a target that happens not to exist through, because
`EvalSymlinks` reports `ErrNotExist` which is indistinguishable from an ordinary
missing file. The same link on a machine where that target exists is correctly
refused. A containment check whose answer depends on whether the attacker's
target is present is not a containment check. Plan §11 amendment 45.

### `workspace`: `project-evil`

`TestSiblingPrefixIsNotContainment` exists for one character. Without a trailing
separator, `strings.HasPrefix("/home/me/project-evil", "/home/me/project")` is
true and a sibling directory passes containment.

### `workspace`: the case-insensitivity regression

`TestDenylistIsCaseInsensitive` was written *after* fuzzing found `.ENV` read the
real `.env` on macOS. It is in the suite so the fix cannot quietly regress, and
it checks both directions — the variants that must be refused, and lookalikes
(`environment.md`, `monkey.go`) that must not be.

### `workspace`: the ignore-file escape

`TestIgnoreFileSymlinkEscapeIsRefused` covers a gap `gosec` found:
`filepath.WalkDir` does not follow symlinks while traversing, but `os.ReadFile`
follows them when opening. **This test was verified by breaking the fix and
watching it fail** — see the note at the bottom.

### `tui`: the structural-equality property

`TestStyledStripsToPlain` renders every state twice, with colour and without,
and asserts that stripping the escapes from one yields the other byte for byte,
at six sizes in both glyph modes.

This is ui-spec §12's accessibility rule made checkable: colour is decoration,
structure is the contract. It caught a real bug on its first run — the width
primitives counted SGR bytes as visible content, so styled rows were padded too
little and every border on the row walked left.

### `tui`: the executable screen reference

`TestMatchesScreenReference` parses all thirteen character grids out of
`plan/kirsch-ui-screens.md` and compares them against `View()`. The design
document is not a description of the interface; it is an assertion about it.

When it fails, exactly one of two things is true — the renderer is wrong, or the
document is. Fix whichever, in the same commit.

### `tool`: backend parity

`TestSearchBackendParity` runs nine queries through both `search_code` backends
— ripgrep and the pure-Go walk — and requires byte-identical output. Three of
the queries probe files that must never surface (`.env`, a gitignored file,
`node_modules`).

A divergence here is not a performance detail. It means the same question
returns different answers on two developers' machines, and from Milestone 3 the
model's behaviour diverges with it.

### `app`: cancellation and the leak check

`TestCancelReturnsWithinOneSecond` asserts ui-spec §11's target rather than
eyeballing it. `TestNoGoroutineLeak` runs twenty app lifecycles and compares
goroutine counts — the app is the first real concurrency in the codebase, so a
leak here is the kind that survives to production.

### `arch`: the import rules

`TestImportRules` fails if `internal/tui` imports `tool`, `workspace` or
`provider`, or if `internal/agent` imports an implementation package. It passes
*vacuously* today for `agent` and `provider`, which do not exist yet — and that
is the entire point. The day `internal/provider` first appears is precisely the
day nobody is thinking about import rules.

Verified by adding a deliberate illegal import and watching it fail with the
offending edge named.

## Fuzz targets

```bash
npm run fuzz          # 30s per target
npm run fuzz:long     # 5m per target
```

`go test` runs each target's **seed corpus** as ordinary tests, which is what CI
does. The fuzzing engine itself — which generates new inputs — runs on demand.

| Target | Property asserted |
|---|---|
| `workspace.FuzzResolveNeverEscapes` | Whatever `Resolve` returns is inside the workspace and not denylisted |
| `workspace.FuzzResolveTerminates` | No input makes resolution hang |
| `tui.FuzzSanitize` | No escape byte, no cursor-moving control, valid UTF-8, no over-wide line |
| `tui.FuzzSanitizeIsIdempotent` | Sanitising twice equals sanitising once |
| `tool.FuzzTruncate` | Never splits a rune, never exceeds its budget |
| `tool.FuzzDecodeInput` | Never panics; every rejection carries a message |

These assert **properties**, not cases, and that is why they find things a table
test does not. `FuzzResolveNeverEscapes` reached `.ENV` within seconds; the
hand-written table of denied paths had encoded the same case-sensitivity
assumption the code made, so review could not catch it either.

A crashing input is written to the package's `testdata/fuzz/` directory.
**Commit it** — it becomes a regression test that runs on every `go test`.

## What is deliberately not covered

Worth stating plainly, so nobody mistakes silence for coverage.

- **No live model calls.** Nothing in the suite needs an API key. A test suite
  that costs money and needs a credential is a test suite that stops being run.
  The agent loop is tested against `provider.Fake` from Milestone 3; the only
  live exercise is `scripts/smoke-test.sh` in Milestone 4.
- **No real terminal.** Tests drive `Update` with synthetic messages and assert
  on `View()`. Terminal behaviour — whether `Alt+Enter` arrives, whether braille
  renders — cannot be tested this way and lives in the manual walkthroughs.
- **No `internal/app` sub-tests in the count above.** Its tests are
  coarse-grained integration tests by design; each drives a real TUI model and a
  real App together.
- **Windows.** Unsupported in v0.1 (plan §1); process-group kill is POSIX.

## Two rules this suite is built on

**A test double must not be more forgiving than the thing it replaces.**

Milestone 1 shipped a deadlock that existed only in the real program: the double
for `program.Send` was a buffered channel and could not block, so `Update`
calling into it looked fine in every test and hung in reality. The harness in
`internal/app/integration_test.go` now uses an **unbuffered** channel, which
reproduces `Send`'s blocking discipline — a regression becomes a timeout rather
than a pass. Plan §11 amendment 46.

**Verify that a test bites.**

When a test is written to catch a specific bug, break the fix and confirm it
fails. Two tests here were checked that way —
`TestIgnoreFileSymlinkEscapeIsRefused` and `TestRendererEmitsPaletteSGR` — and
both would otherwise have been decorative. The second is a particularly good
example: lipgloss silently resolves to no-colour when its output is not a TTY,
so a "coloured" golden captured under `go test` is byte-identical to the plain
one, and every colour assertion passes while proving nothing.
