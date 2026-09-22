# Code Standards — Kirsch Go Codebase

Two developers, one codebase. This document describes how Go code in this project is written, organised, and commented, so that the work reads as one hand.

This is a **project standard**, not a tool-enforced rule. Code that violates this standard is a review finding when the violation occurs in code being changed. Unmodified pre-existing code is never a violation and is never reported as one. When a rule is newly established, existing code comes into line as it is touched, not retroactively.

## Scope Of Binding

Three binding clauses apply to every rule here:

1. **Scope of binding.** A rule binds the code being added or modified. It does not bind unmodified pre-existing code.
2. **Rule over example.** Code that differs from an example while satisfying the rule is compliant.
3. **Intent tests, not site lists.** Where a rule states a test of role or intent, apply that test. No exemption list overrides the test.

## What This Document Does Not Own

The following areas are covered elsewhere and should not be duplicated here:

- **Package layout, dependency rules, error model, testing seams** — [plan/spec/architecture.md](../spec/architecture.md) (§2 layouts, §3 dependencies, §8 error model, §10 testing seams)
- **Formatting, import grouping, lint rules** — [.golangci.yml](../../.golangci.yml) (gofmt, gofumpt, goimports, local-prefix ordering, enabled linters)
- **Work division, branches, commits, pull requests** — [working-agreement.md](./working-agreement.md)

A rule a machine already checks does not need a human rule beside it. Duplicated facts drift apart.

## File Layout and Decomposition

### One File, One Role

Each `.go` file holds a single architectural role. One file might implement a tool, render a component, orchestrate a subsystem, or provide a service — it does not split its focus across multiple concerns.

**Examples:**
- `internal/tool/git.go` — the Git tool and only the Git tool
- `internal/tool/readfile.go` — the file-reading tool and only the file-reading tool
- `internal/tui/composer.go` — the message composer component
- `internal/workspace/workspace.go` — the workspace detection and containment service

This holds regardless of a file's line count. **Line count is not a trigger for splitting a file.** A file split by role remains on-role even at 999 lines, like `internal/tui/update.go`. When a file grows large, decomposition happens inside it — through functions, methods, and sections — not by splitting it into multiple roles.

The exception to "one role" is the envelope file for each role or package: `tool.go` in `internal/tool`, `model.go` in `internal/tui`, `workspace.go` in `internal/workspace`. These files are allowed to hold the role's entry points, public interface, and registry because they coordinate their implementation.

### Declaration Order Within a File

Declarations follow a consistent order:

1. **Package comment** (for the file that carries the package's primary type)
2. **`package` clause**
3. **`import` block** (standard library, third-party, local, in that grouping)
4. **Constants** (grouped by semantic concern)
5. **Type definitions** (structs, interfaces)
6. **Constructors** (e.g., `NewRegistry`, `NewStyles`)
7. **Methods** (grouped by receiver type)
8. **Unexported helper functions**
9. **Test code** (in `*_test.go` files)

### Function-Level Decomposition

Within a large file or method, break concerns into helper functions rather than nesting logic deeply. Two patterns apply:

**Exported helpers** — use when the abstraction is useful beyond the calling file. Example: `internal/tool/tool.go` exports `Fail()` and `OKResult()` so that multiple tool implementations (`git.go`, `listfiles.go`, and others) can build results uniformly without duplicating the failure or success template.

**Unexported helpers** — use for logic shared within a file but not exported. Example: `internal/tool/git.go` has `runGit()` at line 137 and `gitFailure()` at line 153; both are called only within `git.go` itself (`runGit` at :45 and :117; `gitFailure` at :47 and :119) and remain unexported. Similarly, `internal/tui/update.go` has `normalizeSS3()` for input normalization, `tickSpinner()` and `tickStream()` for command generation, and `debugToolCall()` and `debugUsage()` for debug output. These are implementation details of the update orchestration and remain unexported.

**Methods on the main type** — use for significant behaviours that belong to the type's lifecycle. Example: `internal/tui/update.go` defines `dispatchKey()`, `applyToolResult()`, `keyComposing()`, `keyBrowsing()`, etc. as methods on `Model` because they orchestrate state transitions specific to the model's responsibility.

Guidance: extract when a function or method exceeds ~40 lines of logic, or when two paths repeat the same concern. This is advisory, not binding — the test is readability for that function's purpose.

### Package Comments

A package comment appears once per package, in the file holding the package's central type or interface — not necessarily the alphabetically first file.

**Examples:**
- `internal/tui/model.go` (not cards.go, which is alphabetically first)
- `internal/tool/tool.go` (not git.go, which is alphabetically first)
- `cmd/kirsch/main.go` carries the pattern `// Command <name> ...` rather than `// Package main ...`

The comment is a 5–7 line block (measured: 5–7 lines across all production packages). It describes the package's role and primary abstractions, not just a restatement of the name.

**Presence:** All 8 packages carry a package comment. Test-only packages also carry a package comment: `internal/arch` demonstrates this at `internal/arch/imports_test.go:1`.

### Section Dividers in Large Files

Non-test source files exceeding 400 lines of code use section dividers to mark major sections. A divider takes the form `// ── <Label> ` padded with `─` to exactly **79 runes** (the classic 80-column terminal minus one).

**Example:** `// ── Scroll ──────────────────────────────────────────────────────────────────`

This width includes the comment prefix, the label, and the padding, and is counted in runes (Unicode characters), not bytes. UTF-8 box-drawing characters are multi-byte, so a byte count will be higher.

**Trigger:** 400 lines, non-test files only. Test files are exempt because their structure serves verification, not production organisation. Files at or exceeding 400 lines benefit from dividers; files below it rarely need them.

**Current sites:** Three dividers exist, all in `internal/tui`: `transcript.go:292`, `transcript.go:339`, `view.go:297`.

### Naming Files

File names are **lowercase, no underscores except the `_test` suffix**.

Examples: `git.go`, `readfile.go`, `view.go`, `model.go`, `empty_test.go`, `tool_test.go`.

## Comments

### Package Comment Shape and Placement

See "Package Comments" under File Layout and Decomposition, above.

### Doc Comments on Exported Declarations

Every exported identifier carries a doc comment.

**Predicate:** Coverage counts apply this predicate: godoc shows a comment for the declaration — it carries its own doc comment, or is explained by a comment from its enclosing declaration group. Under this predicate, 251 exported declarations are examined; 219 carry explanation (own or inherited), 32 do not.

**Actual coverage:** The 32 declarations not carrying explanation follow two conventions, below:

**Standard-interface exemption:** Methods satisfying a standard interface (`error`, `fmt.Stringer`, `io.Writer`) need no explicit doc comment. The interface name supplies the contract. Six methods qualify:

1. `internal/config/config.go:123` — `func (w Warning) String() string` (satisfies `fmt.Stringer`)
2. `internal/telemetry/telemetry.go:178` — `func (s *sizeCounter) Write(p []byte) (int, error)` (satisfies `io.Writer`)
3. `internal/tool/tool.go:49` — `func (e *Error) Error() string` (satisfies `error`)
4. `internal/workspace/walk.go:133` — `func (l *limitReached) Error() string` (satisfies `error`)
5. `internal/workspace/workspace.go:30` — `func (v *Violation) Error() string` (satisfies `error`)
6. `internal/workspace/workspace.go:41` — `func (e *InvalidPath) Error() string` (satisfies `error`)

**Enum exemption:** Exported enum constants are documented collectively by their type's doc comment, or individually through the `const (` block comment or per-constant trailing comments. The type always carries the primary doc comment. Constants follow three patterns:
- **26 carry no comment** at any level — covered by type comment only
- **5 carry trailing per-constant comments** (as in `CardState`, where constants are followed by `// ◌ awaiting approval or queued` style remarks)
- **18 carry a `const (` block comment** that documents all constants in the block together, plus the type-level comment

This pattern holds across 10 exported enums, totalling 49 constants: 26 + 5 + 18 = 49.

The 10 enums and their constant counts:

- `ItemKind` — 7 constants (`internal/tui/transcript.go:19`)
- `CardState` — 5 constants (`internal/tui/transcript.go:65`)
- `ApprovalKind` — 2 constants (`internal/tui/transcript.go:103`)
- `ApprovalOutcome` — 4 constants (`internal/tui/transcript.go:111`)
- `BaseMode` — 2 constants (`internal/tui/model.go:17`)
- `Mode` — 5 constants (`internal/tui/model.go:25`)
- `ModalKind` — 3 constants (`internal/tui/modal.go:9`)
- `ConfirmAction` — 3 constants (`internal/tui/modal.go:34`)
- `Kind` — 14 constants, with block comment `// The full plan §3.3 set…` (`internal/tool/tool.go:20`)
- `ProjectType` — 4 constants, with block comment `// The four kinds v0.1 recognises.` (`internal/workspace/detect.go:11`)

**Trailing comments on tuning constants:** Beyond enums, six exported constants carry trailing comments that document their purpose or value: `MaxFileBytes` (file read limit), and five timing constants (`SpinnerFrame`, `StreamCoalesce`, `CmdOutCoalesce`, `PasteDebounce`, `DoubleInterrupt`). These are not enum members but follow the same trailing-comment pattern as a standard Go idiom for explaining tuning values. All are in `internal/tool/readfile.go` and `internal/tui/styles.go`.

**Comment form:** A doc comment opens with the identifier's own name, is a full sentence or clause, and ends with a full stop.

Example: `// Model is the root Bubble Tea model.` (from `internal/tui/model.go:78`)

### Comments Explain Why, Not What

A comment that restates what the code obviously does is noise. A comment that explains a decision, a constraint, or an architectural consequence is necessary.

**Example of why (good):**
```go
// Callbacks into internal/app. Function fields rather than an interface
// so the TUI depends on behaviour it names itself and cannot be handed a
// package it is forbidden to import.
```

This explains a dependency-rule consequence that a reader could not derive from the code alone.

**Example of what (avoid):**
```go
// i is the loop counter
for i := 0; i < 10; i++ {
```

The code is self-evident. The comment adds nothing.

### No TODO, FIXME, HACK, or XXX

Outstanding work is tracked in `plan/PROGRESS.md`, not in code comments. Zero instances across the entire codebase.

This is deliberate discipline, not an oversight.

## Naming

### Receiver Names

Method receivers are **one lowercase letter**, matching the type's initial or a conventional abbreviation.

Examples: `(r *Registry)`, `(m Model)`, `(e *Error)`, `(t *SearchCode)`.

**Unnamed receivers:** When a method body does not use the receiver, the receiver name is omitted. This applies only to marker methods that seal an interface:

```go
func (RunToolIntent) isIntent() {}    // Unnamed receiver
func (CancelIntent) isIntent() {}     // Unnamed receiver
```

These are interface-sealing methods with empty bodies; omitting the receiver name signals the intent.

### Sentinel Errors

Sentinel errors are declared as module-level `var`, not `const`, using `errors.New`. One exists in the codebase:

```go
//nolint:staticcheck // ST1005: user-facing message, not an error fragment
var ErrNoWorkspace = errors.New(
	"Kirsch runs inside a Git repository.\n\n" +
		"Either cd into a repository, or point Kirsch at a directory explicitly:\n" +
		"    kirsch --workspace /path/to/project")
```

**Naming:** `Err<Concept>` (not `Error<Concept>`).

**Linter exemptions:** Linter suppression comments (`//nolint`) name the linter rule and the reason for suppression in the same comment. This general rule applies to all suppressions. `staticcheck` rule `ST1005` is suppressed for user-facing message text — text printed directly to the user as onboarding guidance, not wrapped into another error as a fragment. This applies whether the text is a sentinel error or constructed at the point of failure with `fmt.Errorf(...)` inside a function. Examples: `internal/workspace/workspace.go:70` (sentinel, `errors.New`), `internal/config/config.go:223` (inline, `fmt.Errorf`). Non-ST1005 suppressions follow the same pattern; an example is `internal/telemetry/telemetry.go:116`, which suppresses `staticcheck` with the reason "nil ctx is fine here".

### Constructors

Constructors follow the pattern `New` or `New<Type>`:

- `NewRegistry()`
- `NewRenderer()`
- `NewStyles()`
- `NewGlyphs()`

Alternative factory patterns are rare. `NewWithFixture()` at `internal/tui/model.go:170` adds a descriptive suffix where additional configuration is needed; this variant is accepted alongside the `New<Type>` pattern.

## Error Handling

Error model, classes, and semantics are documented in [plan/spec/architecture.md](../spec/architecture.md) §8. This section covers construction only.

**Error construction** follows these patterns:

- **Custom error type** (`tool.Error`) — for tool results returned to the model.
- **`errors.New`** — for sentinel errors.
- **`fmt.Errorf` with `%w`** — for wrapping infrastructure errors that need context.

**Linter suppression:** See Sentinel Errors, above.

## Tests

### File Naming and Organisation

Test file mirrors its source file, or is a named exception.

**Mirror pattern:** `foo.go` is tested by `foo_test.go`. Eight source files follow this pattern:

- `internal/app/app_test.go` mirrors `app.go`
- `internal/config/config_test.go` mirrors `config.go`
- `internal/telemetry/telemetry_test.go` mirrors `telemetry.go`
- `internal/tool/tool_test.go` mirrors `tool.go`
- `internal/tui/composer_test.go` mirrors `composer.go`
- `internal/tui/empty_test.go` mirrors `empty.go`
- `internal/workspace/walk_test.go` mirrors `walk.go`
- `internal/workspace/workspace_test.go` mirrors `workspace.go`

**Named exceptions** — five ruled categories. Eight test files fall under these exceptions:

- **Architecture** — `internal/arch/imports_test.go`
- **Fuzz testing** — `internal/tool/fuzz_test.go`, `internal/tui/fuzz_test.go`, `internal/workspace/fuzz_test.go`
- **Golden-file tests** — `internal/tui/golden_test.go`, `internal/tui/regen_test.go` (regenerates golden fixtures under `KIRSCH_REGEN`)
- **Smoke tests** — `internal/tui/view_smoke_test.go`
- **Integration tests** — `internal/app/integration_test.go`

**Existing files outside the ruled set** (grandfathered, not violations):

These three files exist and are not to be renamed:

- `internal/tui/header_test.go` — dedicated component tests
- `internal/tui/scroll_test.go` — dedicated component tests
- `internal/tool/tools_test.go` — multi-tool registry tests

**Coverage:** 19 test files: 8 mirror, 8 exception, 3 grandfathered.

Grandfathered files are not violations and are not reported as findings. The document binds only changes under review.

**Coverage gap:** `internal/workspace/detect.go` has no test file. This is a coverage gap, not a violation.

### Package Declaration in Tests

All test files declare the same package as their source (`package foo`, not `package foo_test`). This allows direct access to unexported types and functions.

### Test Function Naming

Test functions are named `Test<Concept>` and read as descriptive phrases:

- `TestPanickingToolDoesNotEscape`
- `TestUnknownToolSelfCorrects`
- `TestTheCaretOverstrikesAndNeverWidensTheRow`
- `TestEveryPathTakingToolRefusesEscapes`

Subtests via `t.Run()` are named for the case being tested, either computed dynamically or from a struct field like `name`.

### Fuzz Test Naming

Fuzz functions follow Go's built-in convention and name the property they hold:

- `FuzzSanitizeIsIdempotent`
- `FuzzResolveNeverEscapes`
- `FuzzResolveTerminates`

### Table-Driven vs. Sequential Style

Both styles are used; there is no rule for preferring one over the other.

**Table-driven tests** work well when cases are orthogonal variations of the same behaviour (multiple inputs, same logic, various outputs).

**Sequential tests** work well when each test names a distinct property or scenario.

Guidance: use a table when cases are independent variations on one concept, and separate functions when each tests a distinct property. This is advisory, not binding — both styles are acceptable, and the choice rests on readability for that test case.

**Measured:** 163 test functions and 48 `t.Run()` subtests across 19 test files show both styles in use.

### Test Helper Discipline

Every test helper function carries `t.Helper()` as its first statement. This marks the helper, so failure lines point to the test call site rather than the helper definition.

**Measured:** 31 uses across the codebase.

## Miscellaneous

### Context as First Parameter

Every function that can be cancelled or has a timeout takes `context.Context` as its first parameter.

### No Build Tags

No build tags (`//go:build`, `// +build`) are used anywhere in the codebase.

### Line Length

There is no line-length limit. The longest code line (excluding comment dividers) is 139 runes at `internal/workspace/workspace.go:376`. Comment dividers reach 79 runes (215 bytes with multi-byte UTF-8 characters). This is a deliberate absence, not an oversight — lines are wrapped where they improve readability, not at an arbitrary column.

### Longest Files Over 400 Lines

Six source files exceed 400 lines and are compliant as one-role files:

- `internal/tui/update.go` — 999 lines (state update orchestration)
- `internal/tui/view.go` — 691 lines (terminal layout and rendering)
- `internal/tui/modal.go` — 640 lines (modal dialog component)
- `internal/tui/model.go` — 449 lines (state model definition)
- `internal/tui/transcript.go` — 441 lines (transcript rendering)
- `internal/workspace/workspace.go` — 411 lines (workspace detection and containment)

Each is one role and remains intact. Two of the six currently use section dividers (`transcript.go` 2, `view.go` 1). The 400-line divider rule binds only code being added or modified; unmodified pre-existing files come into line as they are touched.

Five test files also exceed 400 lines and are exempt by the rule — the trigger explicitly specifies non-test files only, because test structure serves verification rather than production organisation:

- `internal/tui/header_test.go` — 1681 lines (dedicated component tests)
- `internal/tui/view_smoke_test.go` — 1185 lines (smoke tests)
- `internal/tui/scroll_test.go` — 961 lines (dedicated component tests)
- `internal/tui/composer_test.go` — 512 lines (mirror of `composer.go`)
- `internal/tool/tools_test.go` — 401 lines (multi-tool registry tests)

## Pre-PR Checklist

Before opening a pull request, run through this list:

- [ ] Each new `.go` file represents one architectural role (not split by line count).
- [ ] Package comment in place (if this is a new package).
- [ ] Section dividers present in files over 400 lines.
- [ ] All exported identifiers have doc comments (except standard-interface methods).
- [ ] Receiver names are one lowercase letter (or unnamed for marker methods).
- [ ] Test file mirrors source file, or is a named exception (architecture, fuzz, golden, smoke, integration).
- [ ] All test helpers carry `t.Helper()` as the first statement.
- [ ] No `TODO`, `FIXME`, `HACK`, or `XXX` comments.
- [ ] Context is the first parameter of cancellable functions.
- [ ] Error wrapping uses `fmt.Errorf` with `%w` where appropriate.
- [ ] No build tags or conditional compilation directives.
- [ ] Formatted with `golangci-lint fmt` (enforces gofmt, gofumpt, goimports).
- [ ] All linters pass (`golangci-lint run`).
