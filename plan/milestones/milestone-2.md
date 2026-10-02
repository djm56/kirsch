# Milestone 2 — Instruction Set

> **Status: Complete — accepted by the operator on 2026-10-02, with the items listed after Task 9 deferred; 15 of 18 Task 9 boxes ticked.** See [`plan/PROGRESS.md`](../PROGRESS.md) for execution status. Milestone 1 completed 2026-09-12. This is the
> complete, ordered instruction set for Milestone 2 (patches, commands,
> approvals). Execute tasks in order. This is the milestone where Kirsch first
> changes something on disk, so the approval path is the point of the whole
> exercise, not a step within it.

## Ground Rules (read first)

1. **Milestone 1 must be complete.** Every box in [`milestone-1.md`](milestone-1.md)
   ticked, CI green.
2. **No provider, agent, or session packages.** See the build-order table in
   plan §2. This milestone makes side effects safe; nothing yet decides to
   cause them. Approvals are driven by the temporary debug commands, exactly as
   M1's tools were.
3. **Locked decisions** (do not re-litigate):
   - `apply_patch` **never** offers `a` (approve for session). Plan §4, ADR
     0006, principle #1. It is not a UI detail; enforce it in `internal/policy`
     so no call site can offer it by accident.
   - Session grants are argv **prefixes**, never a bare `sh -c`, never a lone
     wildcard.
   - Grants live in memory this milestone; persistence arrives with the session
     store in M4.
   - The *command* allowlist is `internal/policy`. The *path* denylist already
     lives in `internal/workspace` and is not duplicated.
4. **Read amendment 46 before writing Task 6.** The approval flow is the exact
   deadlock shape it describes, and the one architecture.md §5 was written to
   warn about. See the box below.
5. Every task has a checkable result. Do not move on with the previous one
   failing.

---

## Deliverables

This milestone breaks into six independently claimable deliverables. Each deliverable specifies which tasks it covers, what it owns in the codebase, and which other deliverables must be complete before work can start. Two developers can work in parallel on deliverables with no shared files and no dependency relationship.

| ID | Title | Tasks | Depends on | Parallel | Owns |
|---|---|---|---|---|---|
| m2-d1 | Patch infrastructure | 1–3 | — | Yes | `internal/patch/`, `testdata/repo-patch/`, `testdata/patches/` |
| m2-d2 | Policy | 4 | — | Yes | `internal/policy/` |
| m2-d3 | Approval flow | 6 | m2-d2 | Yes | `internal/app/`, `internal/tui/update.go`, `internal/tui/messages.go`, `internal/tui/model.go` (approval state) |
| m2-d4 | Tool implementations | 5 | m2-d1, m2-d2 | Yes | `internal/tool/` |
| m2-d5 | TUI integration | 7 | m2-d3, m2-d4 | No | `internal/tui/cards.go`, `internal/tui/modal.go`, `internal/tui/view.go`, `internal/tui/model.go` (modal state), `internal/tui/update.go` (slash commands) |
| m2-d6 | Testing & acceptance | 8–9 | all | No | — |

### Acceptance criteria per deliverable

**m2-d1 (Patch infrastructure):** Done when fixtures are created and tracked (Task 1.1, 1.2), all diff corpus files parse correctly with expected errors on malformed cases (Task 2), and patch application is atomic with correct handling of line endings, permissions, and trailing newlines (Task 3).

**m2-d2 (Policy):** Done when allowlist matching works correctly for prefix patterns without false matches, `sh -c` and shell evasions always require approval, `ForPatch` never returns `DecisionAllow`, grants are properly scoped and refused for bare wildcards, and `go test ./internal/policy/...` is green.

**m2-d3 (Approval flow):** Done when `Approver.Request` blocks correctly from the tool goroutine, `Approver.Resolve` returns immediately and never blocks, cancellation returns within 1s with no parked goroutines, and an integration test drives a real TUI model through approve, reject, and approve-for-session flows.

**m2-d4 (Tool implementations):** Done when `apply_patch` validates and dry-runs before requesting approval (no approval prompts for invalid patches), `run_command` filters environment correctly (stripping tokens even when allowlisted), handles timeouts with process-group kill, and both tools pass their adversarial checks (path escapes, env leaks, shell evasions).

**m2-d5 (TUI integration):** Done when approval cards render real `ApprovalRequest` data, the `[a]` button appears only when `policy.ForCommand` permits session grants, the diff modal renders parsed diffs with file statistics, `/approvals` lists and clears grants correctly, debug commands `/patch` and `/run` are implemented and labeled `(debug)`, and golden screens still match the reference or the reference is updated with correct visual changes.

**m2-d6 (Testing & acceptance):** Done when all Task 8 tests pass (unit, atomicity, process, approval integration, adversarial), all Task 9 checklist items are ticked, CHANGELOG is updated, `go test -race ./...` is green, `golangci-lint` and `go vet` are clean, and no provider, agent, or session code exists in the repo.

### How two developers work this milestone

The parallel path follows this sequence of dependencies and unblocking:

1. **Stage 1.** Dev A claims m2-d1 (patch infrastructure) and Dev B claims m2-d2 (policy). Both work independently.
2. **Once m2-d1 and m2-d2 are done,** the sequential tail begins. Dev A moves to m2-d3 (approval flow, depends on d2) while Dev B starts m2-d4 (tool implementations, depends on d1 and d2). These overlap but do not collide in files—m2-d3 owns `internal/app/` and specific `internal/tui/` files, m2-d4 owns `internal/tool/`.
3. **Once both m2-d3 and m2-d4 are done,** m2-d5 (TUI integration) can start. This task depends on both and owns the remaining `internal/tui/` files that render approval cards and diffs.
4. **Once m2-d5 is done,** either developer can run m2-d6 (testing & acceptance).

This sequence describes task ordering only and makes no estimate of how long any deliverable takes. The milestone is not fully parallel—the last two deliverables are inherently sequential because m2-d5 depends on m2-d3 and m2-d4 both completing, and m2-d6 depends on all others.

### Warning: `internal/tui` file boundary

Concurrent deliverables — m2-d1 with m2-d2, and m2-d3 with m2-d4 — own separate packages with no shared files and no collision risk.

However, m2-d3 and m2-d5 are sequential, not concurrent: m2-d5 depends on m2-d3, and both edit `internal/tui/model.go` and `internal/tui/update.go`. The risk is not a merge conflict, but that m2-d5 starts against a nearly-complete m2-d3 and inherits its half-built state:

- **m2-d3 builds** approval state in `model.go` (the approval struct and its decision channel) and approval-handling callbacks in `update.go` (Resolve calls from `y`/`a`/`n` key presses).
- **m2-d5 builds** modal state in `model.go` (diff display state) and debug command handlers in `update.go` (`/approvals`, `/patch`, `/run` commands).

The types in `messages.go` form the handover contract: m2-d3 defines `ApprovalRequestedMsg`, and m2-d5 consumes it. **Once m2-d5 has begun, do not change `ApprovalRequestedMsg` or any type it depends on** — changes break m2-d5's rendering logic.

---

## The one hard problem in this milestone

An approval is a **synchronous question asked of an asynchronous UI**. The tool
runner must stop and wait for a human; the human answers through a Bubble Tea
`Update`; and `Update` must never block, on anything, ever (architecture.md §5).

Milestone 1 hit the mirror image of this and lost an afternoon to it (amendment
46): `RunTool` sent a message from inside `Update`, so `Update` waited for the
event loop that could not run until `Update` returned. Nothing rendered and no
error appeared anywhere.

The shape that works:

```
tool goroutine                     TUI goroutine (Update)
──────────────                     ──────────────────────
Approver.Request(ctx, req)
  registers a pending approval
  sends ApprovalRequestedMsg  ───▶ appends the card, sets pendingApproval
  blocks on  <-decision                     … user presses y ─┐
                                                              │
        ◀────────────────────────  Approver.Resolve(id, y) ◀──┘
  returns Decision                        (returns immediately)
```

Three rules make it safe, and each one is a test:

1. **`Approver.Request` is only ever called from a tool goroutine.** Never from
   `Update`, never from `View`.
2. **`Approver.Resolve` never blocks.** It writes to a buffered channel of
   capacity 1 and returns. The TUI calls it from inside `Update`.
3. **`Request` selects on the turn context as well as the decision channel.**
   Cancelling a turn with an approval pending must resolve it as
   `policy_denied`/`cancelled`, not leave a goroutine parked forever.

And one testing rule, learned the same way: **a test double for the approval
channel must be able to block.** An unbuffered or capacity-1 channel reproduces
the real discipline; a large buffer hides the bug entirely.

---

## Task 1 — Fixtures for patching

Extend `testdata/`; do not modify the M1 fixtures other than as listed, because
M1's tests assert against them.

### 1.1 `testdata/repo-patch/`

A fixture whose only job is to exercise diff application.

```
plain.txt              three short lines, LF endings
crlf.txt               three short lines, CRLF endings
no-eol.txt             one line, NO trailing newline
exec.sh                mode 0755 (executable bit must survive)
nested/deep.txt        for a rename across directories
unicode.txt            multi-byte content, for offset maths
binary.dat             NUL bytes — patching must be refused
```

`git ls-files -s testdata/repo-patch` must show `100755` for `exec.sh`. Verify
it rather than assuming: a checkout on a filesystem without the executable bit
makes that test meaningless, and the test should skip loudly rather than pass.

### 1.2 Diff corpus — `testdata/patches/`

Plain `.diff` files, loaded by name in table tests. Each is a fact about the
parser, so name them for the behaviour rather than numbering them.

```
modify-single-hunk.diff
modify-multi-hunk.diff
create-file.diff              --- /dev/null
delete-file.diff              +++ /dev/null
rename.diff                   rename from / rename to
rename-with-edit.diff
chmod-exec.diff               old mode 100644 / new mode 100755
chmod-other.diff              a non-exec mode change → tool_input_invalid
context-mismatch.diff         → patch_conflict
crlf-preserving.diff
crlf-converting.diff          would change line endings → rejected
no-eol-add.diff               adds a trailing newline
no-eol-remove.diff            removes one
binary.diff                   → tool_input_invalid
escape-path.diff              targets ../outside → workspace_violation
denied-path.diff              targets .env → workspace_violation
multi-file.diff               three files, all valid
multi-file-second-conflicts.diff   the atomicity test
malformed-header.diff
malformed-hunk-counts.diff    @@ counts that disagree with the body
```

**Check:** every file is tracked; `go test ./internal/patch/...` will load them
by name, so a missing one fails a test rather than being silently skipped.

## Task 2 — `internal/patch`: parse

A separate package from `internal/tool`, because parsing a diff is pure,
testable, and has nothing to do with the tool envelope.

```go
type Op uint8   // OpModify, OpCreate, OpDelete, OpRename

type FileChange struct {
    Op        Op
    Path      string   // workspace-relative; the target for renames
    OldPath   string   // rename source, else ""
    Hunks     []Hunk
    NewMode   os.FileMode  // zero unless the mode changes
    IsBinary  bool
}

type Hunk struct {
    OldStart, OldLines int
    NewStart, NewLines int
    Lines    []HunkLine  // prefix preserved: ' ', '+', '-', '\'
}
```

1. Parse unified diffs only. `git diff` output and hand-written diffs must both
   work; the model produces the latter.
2. **Validate hunk arithmetic at parse time.** If `@@ -a,b +c,d @@` disagrees
   with the number of context/removal/addition lines that follow, that is
   `tool_input_invalid` with a message naming the hunk. Catching it here means
   the applier can assume well-formed input.
3. `\ No newline at end of file` is a hunk line with its own prefix, in both
   directions. Losing it silently rewrites the file.
4. Binary diffs (`GIT binary patch`, or `Binary files … differ`) parse to
   `IsBinary` so the applier can refuse with a clear message rather than a
   parse error.
5. Mode changes: only `100644` ↔ `100755` are honoured. Anything else is
   `tool_input_invalid`.
6. Paths come out of the diff **stripped of `a/` and `b/` prefixes** and are
   otherwise untouched — no cleaning, no resolution. Resolution is the
   workspace's job and must not be pre-empted here.

**Check:** `go test ./internal/patch/...` parses every corpus file to the
expected structure, and every malformed one produces an error naming the
problem. Round-trip test: parse then re-render a diff and compare to the input
byte-for-byte for the well-formed cases.

## Task 3 — `internal/patch`: apply

1. **Strict context, zero fuzz.** A hunk applies at exactly the offsets it
   claims, with context matching byte-for-byte. No searching nearby, no
   whitespace tolerance. On mismatch return `patch_conflict` carrying the
   offending hunk and the actual file content at that location — that text is
   what lets the model re-read and retry rather than guess.
2. **Atomicity is the hard requirement.** A patch touching N files is
   all-or-nothing:
   - apply every file to a temp copy in the same directory (same filesystem, so
     `os.Rename` is atomic);
   - validate all of them;
   - then rename each into place;
   - if any rename fails, rename back every file already committed.

   The test is not "does the happy path work". It is: **a three-file patch whose
   second file conflicts leaves the working tree byte-identical to before**,
   compared with a hash of every file.
3. **Line endings.** Detect the file's dominant existing ending and preserve it.
   A patch that would convert CRLF to LF or the reverse is rejected rather than
   silently applied — it produces a diff touching every line, which is a
   reviewability disaster.
4. **Trailing newline** honoured in both directions.
5. **Permissions** are preserved except for a deliberate exec-bit change.
6. `Apply` takes an already-resolved absolute path per file. It never calls
   `workspace.Resolve` itself — resolution happens once, in the tool, before
   the approval prompt.

**Check:** the atomicity test above; CRLF survives; the exec bit survives and
can be changed; `no-eol` handled both ways; applying the same patch twice fails
the second time with `patch_conflict` rather than corrupting the file.

## Task 4 — `internal/policy`

The command allowlist and the session-grant engine. Small, pure, heavily
tested — it is the thing standing between a model and your shell.

```go
type Decision uint8  // DecisionAllow, DecisionAskUser, DecisionDeny

type Policy struct { /* config + in-memory grants */ }

func (p *Policy) ForCommand(argv []string) (Decision, string)  // reason
func (p *Policy) ForPatch(files []string) (Decision, string)
func (p *Policy) Grant(argv []string) error   // records a session grant
func (p *Policy) Grants() []string
func (p *Policy) ClearGrants()
```

1. **Pattern matching.** Allowlist entries are argv-prefix patterns with a
   trailing `*` permitted (`go test*`, `git diff*`). Match against the argv
   slice, never against a joined string: `go test; rm -rf /` is one argv
   element in a well-formed call and must not be split into two by the matcher.
2. **`sh -c` and friends can never be allowlisted or granted.** `sh`, `bash`,
   `zsh`, `dash`, `env`, `xargs`, `nohup` — anything whose job is to run
   something else — always require approval, every time. Test the obvious
   evasions: `/bin/sh`, `/usr/bin/env sh`, `bash -lc`.
3. **`ForPatch` never returns `DecisionAllow`.** Principle #1. There is no
   config key that changes this, and there is a test asserting that no
   combination of settings produces an allow.
4. `Grant` refuses a bare wildcard, an empty prefix, and any shell.
5. `allow_session_scoped_grants = false` in config disables `Grant` entirely.

**Check:** `go test ./internal/policy/...` green, including: every default
allowlist entry matches what it should and nothing more; `sh -c` evasions all
require approval; a grant for `go test` matches `go test ./...` on the next
call and does not match `go testfoo`; `ForPatch` never allows.

## Task 5 — `apply_patch` and `run_command` tools

### 5.1 `apply_patch`

Input: `{"diff": "...", "description": "..."}`.

Order of operations, and the order is the security property:

1. Parse the diff (Task 2).
2. Resolve **every** path in it — rename sources and targets both — through
   `workspace.Resolve`. Any failure is `workspace_violation`, returned before
   anything else happens.
3. **Dry-run the application** against temp copies. A patch that cannot apply
   fails here with `patch_conflict`.
4. *Only now* request approval.
5. On approval, commit the already-validated result.

The user is never asked to approve a patch that would then be refused or fail —
plan §3.1. An approval prompt that can be followed by "actually, no" teaches
people to stop reading approval prompts.

### 5.2 `run_command`

Input: `{"argv": ["go","test","./..."], "cwd": ".", "timeout_seconds": 60}`.

- **No shell.** `exec.CommandContext` with explicit argv. Reject a single-string
  command outright with a message telling the model to pass argv.
- **Environment filtering**, and the order matters: build the pass-through set
  (`PATH`, `HOME`, `LANG`, plus `[policy].env_passthrough`), then **strip last**
  anything matching `*_TOKEN`, `*_KEY`, `*_SECRET`, `AWS_*`. An allowlisted
  project var that also matches a strip pattern is still stripped. Test that
  exact case.
- **stdin is `/dev/null`.** A command that prompts must fail immediately, not
  hang to the timeout. Test with something that reads stdin.
- **Incremental output** streams into the tool card, coalesced per ui-spec §11.
  A 60-second `go test` shows progress, not a frozen spinner.
- **Output cap** 200KB combined, head+tail with the middle elided.
- **Cancellation is a process-group kill**: `SIGTERM`, 2s grace, then `SIGKILL`.
  Closing pipes is not cancellation — the child keeps running and keeps its
  file handles. Use `Setpgid` and signal the negative pid.
- **cwd** resolved and confined exactly like any other path.

**Check:** timeout kills the process group within 500ms of the deadline; a
command reading stdin fails immediately; env filtering strips a token even when
allowlisted; a `sleep 60` cancelled mid-run dies rather than lingering (assert
the pid is gone, not merely that the call returned).

## Task 6 — The approval flow

Read the box near the top of this document first.

1. `internal/app` gains an `Approver`:

   ```go
   type approval struct {
       id       int64
       decision chan Outcome   // capacity 1
   }
   func (a *App) Request(ctx context.Context, req ApprovalRequest) Outcome
   func (a *App) Resolve(id int64, outcome Outcome)   // never blocks
   ```

2. `Request` sends `ApprovalRequestedMsg` to the TUI **from the tool goroutine**
   and then selects on the decision channel, the turn context, and the root
   context.
3. `Resolve` is called from `Update` when the user presses `y`/`a`/`n`, and from
   the cancellation path. It must never block: send on a buffered channel, and
   drop silently if the approval has already been resolved.
4. A rejected action returns `policy_denied` **as a tool result**, so from M3 the
   model can adapt rather than treating it as a wall.
5. Grants recorded through `policy.Grant` on `a`.

**Check:** an integration test that drives a real TUI model and a real App
through approve, reject, and approve-for-session; a test that cancels a turn
with an approval pending and asserts the tool goroutine returns within 1s; a
test that resolving twice is harmless. Use an unbuffered channel in any test
double — see ground rule 4.

## Task 7 — TUI: real approval and diff

1. Approval cards render real `ApprovalRequest` data — the M0 fake cards become
   fixtures for golden tests only.
2. `[a]` appears only when `policy.ForCommand` says a grant is possible. The
   renderer asks policy; it does not decide.
3. The diff modal renders the real parsed diff with per-file `+n −m` stats.
4. `/approvals` lists real grants and clears them through the confirm prompt.
5. Two temporary debug commands, this milestone only, removed in M3:
   `/patch <file>` (apply a diff from `testdata/patches/`) and `/run <argv…>`.
   Label them `(debug)` in `/help` alongside M1's.

**Check:** the golden states in [`spec/kirsch-ui-screens.md`](../spec/kirsch-ui-screens.md)
still match — screens 03, 04 and 05 are the approval and diff surfaces, and real
data must render in the same shape. If it does not, the screen reference is
wrong and gets updated in this milestone's commit.

## Task 8 — Tests

1. **Unit:** diff parse and apply (the corpus), allowlist matching, grant
   scoping, env filtering, line-ending detection.
2. **Atomicity:** the three-file partial-failure test, comparing a hash of every
   file before and after.
3. **Process:** timeout, cancellation, process-group death, stdin.
4. **Approval:** approve / reject / grant / cancel-while-pending, through a real
   TUI model.
5. **Adversarial:** a diff targeting `../`, a diff targeting `.env`, an argv
   attempting `sh -c`, an allowlisted command with a token in its environment.
6. `go test -race ./...` green.

## Task 9 — Final acceptance

- [x] Every diff corpus file parses to the expected structure; malformed ones
      name their problem
- [x] A three-file patch whose second file conflicts leaves the tree
      byte-identical
- [x] CRLF files survive patching; a line-ending-converting patch is rejected
- [x] The executable bit survives, and can be changed deliberately
- [x] Applying the same patch twice fails the second time with `patch_conflict`
- [x] Every path in a diff is containment- and denylist-checked **before** the
      approval prompt
- [x] `apply_patch` never offers `[a]`, and no config makes it allow
- [x] Command allowlist matches correctly; `sh -c` and its evasions always ask
- [x] A session grant for `go test` matches `go test ./...` and never `sh -c`
- [ ] Command timeout kills the process group within 500ms of the deadline
- [ ] A cancelled command's process is actually gone, not merely detached
- [x] A command reading stdin fails immediately
- [x] Env filtering strips `*_KEY`/`*_TOKEN` even when allowlisted
- [x] Approve, reject and approve-for-session all work from the TUI
- [x] Cancelling a turn with an approval pending returns within 1s and leaves
      no parked goroutine
- [x] Golden screens still match, or the reference is updated in this commit
- [ ] `go test -race ./...`, `golangci-lint fmt --diff`, `go vet`,
      `golangci-lint run`, CI all clean
- [x] **No provider, agent, or session code exists anywhere in the repo**

**Definition of done:** every box checked, CHANGELOG updated, `../PROGRESS.md`
set to `☑ Complete`, and a commit tagged so Milestone 3 starts from a
known point.

**Status as of 2026-10-02: complete, with deferred items.** The operator tested Milestone 2 by hand and accepted it on 2026-10-02. The items below are deferred, not done. Each one is carried to *Carried from Milestone 2* in [`plan/PROGRESS.md`](../PROGRESS.md), to be picked up in Milestone 3 or a follow-up mission.

- **CI.** The first CI run on Milestone 2 code (run 36985475148, commit `485e1dd`) failed only in two tests that passed a single argument over 128KB to `printf`, which Linux refuses. Mission-20261002-02 fixed them. Every other gate in the second-to-last box is clean locally. The box is ticked once CI is green on the operator's push of that fix.
- **The timeout kill is not proven.**
  - `TestRunCommandTimeoutWithinDeadline` bounds the call at 1–2s after a 1s deadline, which is not the 500ms the box requires.
  - `TestRunCommandTimeoutProcessGroupDead` never checks that the process died. On timeout the result carries no content, so its PID parse fails and the test returns before its assertion (`internal/tool/run_command_test.go:~804`).
  - To close this, fix that test (record the PID out of band) and tighten the bound, or relax the requirement.
- **Cancelled-process death is not proven.** Walkthrough §9 cancels an approval card before any process starts (`plan/testing/manual/user-testing/milestone-2-donovan.md:411`), and `TestRunCommandCancellationProcessGroupDead` has no PID check. To close this, record the PID out of band and assert `ESRCH` after cancelling, then tick the box.
- **The configured command timeout is ignored.** See the last item under *Where execution departed*.
- **Two more settings and output streaming are not wired.** See *Where execution departed* (plan amendments 72 and 73).
- **Some ticked boxes rest on the operator's walkthrough, not on a test that can fail.** The evidence table below says which. The tests that cannot detect a regression are listed under *Open defects*.
- **Trailing-newline handling (Task 3.4) is suspect.** See *Open defects*. Deferred: reproduce it, and fix it if it is real, in a follow-up.
- **Four checks in the operator's walkthrough run were not retested** (`plan/testing/manual/user-testing/milestone-2-donovan.md`); deferred:
  - `/run` in `/help` (§10). It was ❌ before this close-out; mission-20261002-01 fixed it, and it awaits retest.
  - The two `/patch` checks in §10.
  - The expand-with-Up check in §1. It was written before tool cards opened as previews.
- **Release steps:** `plan/PROGRESS.md` is set to `☑ Complete`, and the CHANGELOG carries the Milestone 2 entries. The tag is the operator's.

### Evidence for Task 9

| Check | Proven by |
|---|---|
| Corpus parses; malformed ones name their problem | `TestParseCorpusFiles`, `TestRoundTrip` (`internal/patch/patch_test.go`) |
| Three-file partial failure leaves the tree byte-identical | `TestApplyMultiFileAtomicity`, `TestApplyMultiFileAtomicityPhase2Failure` |
| CRLF survives; converting patch rejected | `TestApplyGitStyleCRLFDiffPreservesCRLF`, `TestApplyCRLFPreservation`, `TestApplyCRLFConvertingPatchRejected`, `TestApplyLFFileRejectsCRLFAddedLine`, `TestApplyPatchCRLFConvertingNoApproval` |
| Exec bit survives and can be changed | `TestApplyExecBitPreservation`, `TestModeChangeExecBit`, `TestApplyRenameWithEditPreservesExecBit` |
| Same patch twice fails with `patch_conflict` | `TestApplySamePatchTwiceFails` |
| Paths checked before the approval prompt | `TestApplyPatchWorkspaceViolationTargetNoApproval`, `TestApplyPatchRenameSourceEscapeNoApproval`, `TestApplyPatchConflictNoApproval`; `TestEveryPathTakingToolRefusesEscapes` (its `apply_patch` target and rename-source subtests cover `.env` and `../`, and check the approver is not called) |
| `apply_patch` never offers `[a]` | `TestForPatchNeverAllows`, `TestForPatchNoConfigurationAllows`, `TestApprovalPatchNeverOffersSessionApproval`; walkthrough §4 |
| Allowlist; `sh -c` and evasions ask | `TestShellsCanNeverBeAllowlisted`, `TestShellDetectionByBasename`, `TestRunCommandShellCommandRequiresApproval` |
| `go test` grant scope | `TestGrantSemantics`, `TestGrantRefusesInvalidPrefixes` |
| Cancelled process is gone | **Not proven — box open.** Walkthrough §9 cancels an approval card before any process starts (`milestone-2-donovan.md:411`). `TestRunCommandCancellationProcessGroupDead` checks only the error kind and a 3s return |
| A command reading stdin fails immediately | Walkthrough §9 (`/run cat`, approved: "The command does not block"); `TestRunCommandStdinIsDevNull` shows a stdin-reading command returns at once. A stdin reader sees EOF and exits; it does not error. The test would also pass without the explicit `/dev/null` assignment, because `exec` reads `/dev/null` when `Stdin` is nil |
| Env stripping even when allowlisted | `TestRunCommandEnvironmentFiltering` (`*_TOKEN`, with a real environment and passthrough); walkthrough §9 (`SECRET_KEY` stripped although allowlisted). `*_SECRET` and `AWS_*` have no test |
| Approve, reject and grant from the TUI | Walkthrough §3 and §7 (operator run). `TestApprovalSessionGrantDeadlock` drives a real `tea.Program` with a stub model and proves only that resolving does not deadlock |
| Cancel with an approval pending returns within 1s | `TestApprovalCancellationReleasesRequest` cancels the context passed to `Request` and asserts `Cancelled` within 1s. By inspection, `Request` parks only in its `select` (`internal/app/app.go:~626–643`), so that return releases it. No test drives an Esc-cancelled turn or asserts goroutine count or `a.approvals` cleanup; the "no parked goroutine" half rests on that reasoning, not on a test that can fail |
| Golden screens | `TestMatchesScreenReference`; screens 06 and 07 updated |
| No provider, agent or session code | `TestImportRules` (`internal/arch`) |

### Where execution departed from this document

Each departure is recorded as a plan amendment in [`spec/kirsch-plan.md`](../spec/kirsch-plan.md) §11 (amendments 61–73).

- **The allowlist is exact-match.** Task 4 describes argv-prefix patterns with a trailing `*`. The shipped default entries (`go build`, `go test`, `git diff`, `git log`, `ls`) match only that exact argv, so `go test ./...` and `ls cmd` ask for approval. Session grants remain prefix matches. This is stricter than the document, and the operator kept it on 2026-10-02.
- **Secret stripping is case-sensitive.** `*_TOKEN`, `*_KEY`, `*_SECRET` and `AWS_*` match upper-case names only, so `my_token` passes through. On 2026-10-02 the operator chose to record this rather than change it.
- **Renamed shells are documented, not blocked.** Shell detection is by basename, so a shell copied under another name is not detected (`TestRunCommandRenamedShellLimitationDocumented`).
- **Mixed line endings are normalised.** A file with mixed endings is rewritten wholesale in its dominant ending, and a tie becomes LF. That includes lines the patch did not touch. A bare `\r` is treated as a line break. This departs from Task 3.3's "preserve" rule.
- **Rename-with-edit keeps permission bits only.** It carries the source file's `Perm()` but not setuid, setgid or sticky. A plain modify and a pure rename keep all of them.
- **A patch that carries line endings is checked.** A patch's hunk lines can carry `\r`, as a `git diff` of a CRLF file does. Such a patch must add lines in the file's own ending, or it is rejected with `line_ending_mismatch`. A `+` line directly followed by `\ No newline at end of file` is exempt, because it has no terminator. A patch with no `\r` anywhere takes the file's ending.
- **The `run_command` timeout ignores the configured default.** `policy.default_command_timeout_seconds` (default 60) is validated by `internal/config/config.go:~273` but never read by `run_command`. When `timeout_seconds` is omitted, the tool falls back to 3600 (`internal/tool/run_command.go:~143–146`) and enforces no maximum, while its schema tells the model "Maximum 3600" (`run_command.go:~83`). Plan §3 specifies a default of 60 and a maximum of 300.
- **A command reading stdin sees EOF; it does not error.** `run_command` gives the child `/dev/null`, so a reader such as `cat` exits at once, normally with success. The Task 9 box and plan §3.2 say "fails immediately"; the delivered guarantee is that it never hangs. Rewording the box is the operator's decision.
- **Three policy settings are not read.** `allow_session_scoped_grants`, `require_approval_for_patches` and `require_approval_for_commands` exist in `internal/config` but nothing passes them to the policy, which always enables session grants (`internal/app/app.go:~305`). Task 4.5's "`allow_session_scoped_grants = false` disables `Grant` entirely" is not met.
- **Command output is not streamed.** `RunCommand.ProgressSink` is never set in production (`internal/app/app.go:~316`), so output appears when the command finishes. Task 5.2's incremental streaming is not met.

### Added beyond this document

- **The approval deadlock fix** (mission-20260928-01). `app.Resolve` runs on the event loop, so the session-grant path now delivers its message sequence from one goroutine (`sendAsyncSequence`). `internal/app/deadlock_test.go` holds the first tests to drive a real `tea.Program`.
- **Walkthrough fixes** (missions 20260930-01 and 20261001-01): the `c` clear instruction in the grants modal, real approval elapsed time, and command cards that name their command.
- **The UX pass** (mission-20261001-02):
  - tool cards open as a 10-line preview;
  - ↑/↓ give composer history, and Shift+↑ or Tab reach the cards;
  - an approved command shows one card;
  - a key-hint row;
  - a pending approval card that stray typing releases rather than answers;
  - a clipping marker on modals.
- **The close-out** (mission-20261002-01):
  - `/run` shares the first debug row with `/patch` in `/help`, and the clipped help footer names the scroll keys;
  - lint and format are clean, with 13 findings fixed and none suppressed;
  - rename-with-edit no longer leaves files at 0600;
  - real `git diff` output for CRLF files now applies.

### Open defects and notes for Milestone 3

- **Trailing-newline handling: found by reading the code, not yet reproduced by a test.**
  - `applyModify` (`internal/patch/apply.go:~809–816`) decides the final newline from the last hunk's last line alone.
  - A file with no final newline gains one when a hunk stops short of the end.
  - A hunk that deletes the last line, with the `\ No newline` marker on the deleted line, drops the newline from the new last line.
  - Either would break Task 3.4. Deferred: reproduce and fix if real, in a follow-up.
- **Tests that cannot detect a regression:**
  - `TestRunCommandTimeoutProcessGroupDead` returns before its assertion.
  - `TestRunCommandCancellationProcessGroupDead` checks no PID.
  - `TestRunCommandStdinIsDevNull` passes without the code it names.
- **Untested secret patterns:** `*_SECRET` and `AWS_*` stripping, and `*_KEY` outside the walkthrough.
- **Unverified blank approval card:** an approval request with an unspecified operation sends empty `Subject` and `Detail`. Whether the card renders blank is unverified.
- **A comment contradicts behaviour:** `internal/tool/run_command.go:~316–321` and `~336–337` say the grace poll always waits the full 2s. `TestRunCommandTimeoutWithinDeadline` (`run_command_test.go:~755`) bounds a 1s timeout at 2s, so the poll must exit early once `Wait` reaps the process. The comment is probably wrong; confirm and fix.
- **The configured command timeout is ignored:** wire `policy.default_command_timeout_seconds` into `run_command`, enforce a ceiling, and make the schema text match.
- **Unwired policy settings and streaming:** wire `allow_session_scoped_grants` and the two `require_approval_for_*` keys into the policy and `ProgressSink` into `internal/app`, or remove the keys and the requirement.
- **No regeneration flag for screen grids:** `golden_test.go` has no `-update` flag, so screen grids must be regenerated from the renderer by hand.
- **Help overlay height:** the help overlay needs 34 rows to show without scrolling.

---

## Explicitly Out of Scope (reminder)

No provider, no agent loop, no session store, no compaction, no token or cost
tracking, no `git commit`/`git push` tools (not in v0.1 at all). Approvals are
still driven by debug commands; nothing decides to patch or run anything yet.
If a task seems to need one of these, stop and check the build-order table in
plan §2.
