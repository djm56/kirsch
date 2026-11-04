# Contributing to Kirsch

This file covers how to build, test and check a change, how the golden screens
are regenerated, the principles every change must keep, and what a pull request
should contain.

## Requirements

- Go, at the version in the `go` line of `go.mod`.
- Node.js, at the version in the `engines` field of `package.json`. It brings
  npm, which runs the `npm` scripts.
- Python 3, for `npm run screens`.
- The lint and security tools. Run `npm run tools` once to install them.

## Build, test and check

Run these from the repository root.

| Task | Command |
|------|---------|
| Build the binary | `npm run build` |
| Run from source | `npm start` |
| Unit tests | `npm test` |
| Tests with the race detector | `npm run test:race` |
| Format, vet, lint, test and screens checks | `npm run check` |
| Security scans | `npm run security` |
| Fuzz targets | `npm run fuzz` |
| Every job CI runs, locally | `npm run ci:local` |

`npm run check` is the minimum before you open a pull request. `npm run ci:local`
runs the same jobs as the CI workflow, and is the full check. It installs
tools as it runs: it overwrites `govulncheck` and `gosec` in your Go bin
directory with their latest releases, and downloads the pinned version of
`golangci-lint` into `bin/`, which git ignores.

## Golden screens

The golden screens are character grids in `plan/spec/kirsch-ui-screens.md`. A
test compares the renderer's output with each grid. Nothing in CI regenerates
them, and nothing in the repository updates them automatically.

To change a screen:

1. Run the regeneration test with the environment variable set:

   ```bash
   KIRSCH_REGEN=1 go test ./internal/tui -run TestGenerateGrids
   ```

   The test is skipped unless the `KIRSCH_REGEN` environment variable is set to
   a non-empty value; use `1`. When it runs, it writes the current render of
   every scenario to `/tmp/kirsch-grids.txt`. It does not change the screens
   document.

2. Reconcile each affected grid in `plan/spec/kirsch-ui-screens.md` by hand,
   from `/tmp/kirsch-grids.txt`. A grid that differs from the render is either a
   renderer defect or a stale reference. Decide which, and say which in the pull
   request.

3. Run `go test ./internal/tui`, then `npm run screens`.

4. A person reviews the diff of the screens document. Golden changes are never
   accepted without that review.

## Non-negotiable principles

Every change must keep these principles. A change that breaks one is not ready
to merge, whatever its tests say.

1. The agent never writes to a file without an approved patch.
2. The agent never runs a command without a policy decision (allowlist or user approval).
3. The agent never accesses paths outside the workspace root, including via symlinks.
4. Every consequential action is visible in the TUI before it happens.
5. Every session is durable and resumable.
6. Everything is cancellable — no UI hang on a stuck tool or model call.

With the default settings, the file tools and the policy gate enforce principles
1 to 3 for tool calls. A command that runs, whether you approved it or it is on
the built-in allowlist, is outside the file tools' limits: it can write files and
reach paths outside the workspace. The allowlist includes `go test`, which runs
the workspace's own test code. Turning off command approval in the global
configuration also lets most commands run without asking.
[SECURITY.md](SECURITY.md) states these limits, and a change must not widen them
without saying so.

## Pull requests

- Keep each pull request to one logical change. Small changes are easier to review.
- Say what changed, why, and what you checked. List the exact commands you ran
  and what they returned.
- Run `npm run check` before you open the pull request.
- New behaviour gets a test. A bug fix gets a test that fails without the fix.
- If a golden screen changed, say in the description which screens moved and
  whether the render or the reference was wrong.
- If a public surface changed, update the README, the `doc/` pages or the
  CHANGELOG to match.
- Do not report a security problem in a public pull request or issue. Follow
  [SECURITY.md](SECURITY.md).
