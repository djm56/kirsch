# Kirsch — User Documentation Website Plan

> **Status: Planning only; nothing built.** Dated 2026-11-17. Mission-20261117-03, step 2, revised at step 4 (fix round 1 of 2) against the review at step 3. Repository facts were re-checked at step 4; each citation is `file:line` as it stands on 2026-11-17. Paths are relative to the repository root, `/Volumes/DATA/Github/kirsch`. Facts from the step 1 research are carried here with their source, a repository `file:line`, a command, or a URL in [Sources](#sources). The step 1 research notes are working notes, archived with the mission; nothing here depends on them.

## Summary

Kirsch has user pages in `doc/`, but no website. The front page people see is the repository README, which is written for someone who clones the code. This plan proposes a documentation website built from `doc/`, so the words in the repo and the words on the site are the same words. The build plan already chose `doc/` as the home for user documentation (`plan/build-plan-v0.2-v0.6.md:47`), and this plan keeps to that choice. The site is built only when a release tag is pushed, so it never shows a feature that has not shipped. The pages are reshaped into a quick start, a full install, a provider-and-model guide, task guides, reference and troubleshooting. Screenshots go onto the doc pages now. Recordings made in CI would go beyond decision O12 and are left for a later phase.

Bishop recommends Astro Starlight and a subdomain. They depend on answers the operator has not given yet (see [Operator decisions](#operator-decisions)). Nothing here is built.

## What exists today

| File | What it covers | Gap against the topic list |
|---|---|---|
| `README.md` | Repo front page: pitch, build-from-clone install (`README.md:23`), project contract, layout, screenshots (`README.md:124`, `:126`, `:128`), verify commands | Not a site page. Carries links into `plan/` (command below). |
| `doc/README.md` | Index table of the user pages (`doc/README.md:7–12`) | Index only. A link into `plan/` (`doc/README.md:14`). Needs rewriting as a site landing page. |
| `doc/install.md` | Platforms (`:13`), Git and ripgrep (`:23`), install from a release archive (`:51`), `go install` (`:129`), build from source (`:162`), API key (`:182`), `kirsch doctor` (`:206`) | Quick install and full install in one page. Pinned to one release at `:68` and `:146`. |
| `doc/configuration.md` | Config file locations (`:15`), precedence (`:29`), Secrets (`:60`), key reference (`:89`), `[provider]` (`:95`), environment variables (`:186`), debug log (`:198`) | Reference. Holds the provider keys, but no guide for choosing a provider or model. |
| `doc/usage.md` | Starting it (`:20`), the screen (`:46`), keys (`:108`), slash commands (`:254`), approval model (`:302`) | Reference-shaped. Resume is prose only, at `:30–33`. No heading names resume. |
| `doc/troubleshooting.md` | `kirsch doctor` output explained one check at a time (`:21` to just before `:572`), common failures (`:572`), and a closing section (`:710`) | Largest page. Organised by doctor check, not by problem. Needs splitting. |
| `doc/images/` | PNG screenshots: `start-screen.png`, `help-overlay.png`, `approval-card.png` | No `doc/` page embeds them. Referenced from `README.md` only. |

Commands behind the figures above, run from the repository root. Each count is a value to re-run, not a number to keep. Re-run before relying on one.

```bash
wc -l README.md doc/*.md
grep -oE '\]\(plan/[^)]*\)' README.md | wc -l
grep -oE '\]\(\.\./plan[^)]*\)' doc/*.md | wc -l
grep -oE '\]\([A-Za-z_-]+\.md[^)]*\)' doc/*.md | wc -l
grep -oE '\]\(#[^)]*\)' doc/*.md | wc -l
grep -n 'images/' doc/*.md                  # no output: no doc page embeds a screenshot
grep -n 'images/' README.md                 # control: the three screenshot references
grep -c -i '^#.*resume' doc/usage.md        # 0: no heading names resume
```

The `grep -n 'images/' doc/*.md` result is an absence. Its control is the `README.md` line beside it, which shows the same pattern does find references where they exist.

### Against the operator's topic list

| Topic | Where it is today | Gap |
|---|---|---|
| Quick install | No dedicated page. The README "Install" section (`README.md:23`, up to `## Core Ideas` at `:44`) gives build-from-clone commands only. | No short path to a release binary. |
| Full install | `doc/install.md` | Covered. Its download commands move out to the quick start. |
| Models, API keys, providers | Split between the key export (`doc/install.md:182–205`), the secrets rules (`doc/configuration.md:60–88`), and the `[provider]` keys (`doc/configuration.md:95–119`). `doc/install.md:7–9` says provider choice is in `configuration.md`. | No guide. Nothing explains how to choose or switch an endpoint. |
| How-to guides | None task-based. | Missing. |
| Troubleshooting | `doc/troubleshooting.md` | Present, but too long and organised by check. |
| Screenshots | README only. | Not on any doc page. |

### Stale text that would travel with the docs

Stale text is found by a search, not a hand list. The search:

```bash
grep -n '0\.1\.0' README.md doc/*.md
```

At revision the command returns 18 lines. Each is classed below. A class marked **keep** stays, and the reason is given.

| Class | Lines (today) | What happens |
|---|---|---|
| Pinned to a release, must change | `doc/install.md:64` (prose: "use version `0.1.0`"), `:68` (`VERSION=0.1.0`), `:146` (`go install ...@v0.1.0`) | Moved and rewritten by ds-d1. The quick start carries a `<version>` placeholder, not a pin (see ds-d1). |
| Status text that will age | `README.md:3`, `:25`, `:86` ("goes away after v0.1.0", describing `plan/`), `:118`, `:122`; `doc/install.md:126`, `:156` (the "`@latest` once a release exists" half of the line), `:229` (example output of a release binary) | Reworded by ds-d7 (README: `:3`, `:25`, `:86`, `:118`, `:122`) and ds-d1 or ds-d15 (install: `:126`, `:156`, `:229`). The `:86` wording names a release that has already shipped, so it is reworded, not kept. |
| Keep: a milestone name | `README.md:113` ("polish + release (v0.1.0)") | Keep. It names a milestone. |
| Keep: a format example | `doc/install.md:58` (`<version>` is "such as `0.1.0`"), `:156` (`@v0.1.0` as a tag example) | Keep. Examples of the pattern. |
| Keep: dev builds, not a pin | `doc/install.md:157`, `:160`, `:180`; `doc/troubleshooting.md:40`, `:59` (`kirsch 0.1.0-dev`) | Keep. These are the version string a build from a branch or commit prints, not a release pin. |

Other text that will change with the milestones:

| Location | Text (short) | Why it matters |
|---|---|---|
| `README.md:69` | "One wire format for v0.1 (Anthropic Messages)" | Milestone 7 changes it: "More than one wire (v0.3.0)" (`plan/milestones/milestone-7.md:1`). |
| `README.md:75–77` | "Explicitly Out of Scope for v0.1: Multiple providers" (text at `:77`) | Milestone 6 changes it: "More than one model (v0.2.0)" (`plan/milestones/milestone-6.md:1`). |
| `doc/configuration.md:101–102` | "The built-in endpoints are `opencode` and `anthropic`" | Milestone 6 adds to the set. |
| `CHANGELOG.md:12` | "says v0.1.0 is not yet published" | False now that v0.1.0 is released. See ds-d17. |

### Links that break outside the repo

- The `plan/` links in `README.md`, and the `../plan/` link at `doc/README.md:14`. Commands are in the block above.
- The cross-file `.md` links inside `doc/`. Each needs a site route. The link check in ds-d9 is the acceptance for each one, not a hand count.
- The in-page anchor links in `doc/`. Whether a site generator makes the same anchor slugs as GitHub is UNVERIFIED. The anchors that cross the troubleshooting split are listed in ds-d5.
- The absolute GitHub URLs (`github.com/djm56/kirsch`) in `doc/`. These survive a move. Command: `grep -n 'github.com/djm56/kirsch' doc/*.md`.
- The in-app footer text `docs: github.com/djm56/kirsch` at `internal/tui/modal.go:300` and `:308`. Changing it is a code change, not a doc change (ds-d16, decision D10). The same string is asserted in `internal/tui/header_test.go:1755–1767`, and printed by `plan/spec/kirsch-ui-screens.md:428`.

## Options considered

| Option | How it works | Main risk |
|---|---|---|
| **a.** Docs as code in `doc/`, built by a static site generator and hosted on GitHub Pages | A build reads `doc/`, writes HTML, and a host serves it. The build runs in a GitHub Actions workflow. | A second toolchain in a Go repo. Unless the build is gated on tags, it shows unreleased behaviour on `main`. |
| **b.** GitHub wiki | A separate wiki repository, edited in the browser or as local files. | Cannot be tagged with a release, and cannot change in the same PR as the code (`plan/build-plan-v0.2-v0.6.md:47`). Search indexing is limited to wikis with 500 or more stars that block public editing ([wikis page](https://docs.github.com/en/communities/documenting-your-project-with-wikis/about-wikis)). |
| **c.** The operator's other site pulls the repo's markdown | A build-time checkout of `doc/`, a release artefact, or a content loader in the site's own framework. | If the site is WordPress, it does not render markdown, so every release is a republish and the copies drift. A JavaScript framework path is cleaner, but the site's stack is unknown (decision D1). |
| **d.** Other routes | Hosted docs platforms; GitHub's own rendering of `doc/`; an HTML bundle on each release. | Hosted platforms keep content outside the repo; cost, sync and export are UNVERIFIED. Bundle URLs change per release and the site cannot search them. GitHub rendering has no site search and no custom domain. |

### Site generators, with the facts this plan relies on

| Generator | Facts used here | Trade-off |
|---|---|---|
| **Astro Starlight** (recommended) | Releases dated 06 to 13 November: 0.43.1, 0.43.0, 0.42.11, 0.42.10. Day and month only; the year is inferred from the page order and the context date ([releases page](https://github.com/withastro/starlight/releases)). Starlight is built on Astro, a JavaScript project, so it needs Node. That Node requirement is UNVERIFIED in this plan; no page stating a minimum version was read. | Mature enough to use and actively released. Needs Node in `site/`. |
| **Docusaurus** | Latest release 3.10.2, dated 2026-07-10, year shown; the stable line ([releases page](https://github.com/facebook/docusaurus/releases)). `baseUrl` and `url` are documented ([deployment page](https://docusaurus.io/docs/deployment)). A versioning feature is named in the research but was not read: UNVERIFIED. React and Node are the reviewer's own knowledge, UNVERIFIED here. | Mature, versioning built in (UNVERIFIED), larger than Starlight, and built with React on Node. Heavier to maintain. |
| **Hugo** | One Go binary and no Node (UNVERIFIED: no page read states this). The configuration page read did not mention `baseURL` ([configuration page](https://gohugo.io/configuration/)). | No Node at all. Fewer docs features out of the box than Starlight or Docusaurus (UNVERIFIED). Its subpath behaviour does not matter on the subdomain route, because the site sits at the root of the subdomain. |
| **MkDocs Material** | Bishop recommends against it. Its own README gives an end-of-life date of 2027-05-05 and names Zensical as the successor ([repository](https://github.com/squidfunk/mkdocs-material)). | Its maintenance ends within the life of this plan. |

### Bishop's recommendation

Pending the operator's decisions. Nothing below is decided. [Operator decisions](#operator-decisions) gives the answer each choice needs.

1. **Source stays `doc/` in this repo.** The build plan already decided this at `plan/build-plan-v0.2-v0.6.md:47`. The same line gives the reasons against the GitHub wiki: it cannot be tagged with a release, cannot change in the same PR as the code, cannot be covered by the coverage tests, and cannot be read by an agent with `read_file`. The wiki is not used for user docs (decision D6).
2. **Build with Astro Starlight** (decision D3). The reasons: it is actively released, and it sits on the Astro deployment route that is documented: `base` for a subpath and `site` for the GitHub URL ([Astro GitHub deployment page](https://docs.astro.build/en/guides/deploy/github/)). That page is Astro's, not Starlight's; no Starlight-specific deployment page was read. The runner-up is Docusaurus if the operator wants versioning built in and accepts a heavier toolchain. Hugo is the choice if no Node at all matters more than docs features out of the box. The site toolchain lives in `site/`, so the Go build stays Node-free. `README.md:177–178` gets a wording update (ds-d7).
3. **Join the operator's site by the route that matches decision D1.**
   - Route (i): the other site is Astro or another JavaScript static generator. It loads `doc/` from a pinned tag as a content collection, with no separate host. This route is not tested. The research recorded it as a possibility for Astro content collections and Docusaurus docs plugins, and the test was not run (UNVERIFIED).
   - Route (ii): the other site is WordPress or anything else. The site deploys to GitHub Pages on a docs subdomain, such as `docs.<site>`, and the main site links to it from its navigation. A subpath (`<site>/kirsch`) is possible only if the host can reverse-proxy. That is UNVERIFIED.
   - Recommended default: route (ii) with a subdomain. It needs a DNS record and a link on the other site, and no change to the other site beyond that. GitHub's custom-domain page says a subdomain is a CNAME record to `<user>.github.io`, with no repository name ([custom domains](https://docs.github.com/en/pages/configuring-a-custom-domain-for-your-github-pages-site/managing-a-custom-domain-for-your-github-pages-site)).

## Proposed page structure

The split follows the Diátaxis model, as `plan/build-plan-v0.2-v0.6.md:47` names it. Tutorial is the quick start. How-to is the provider guide and the task guide. Reference is usage, configuration and the doctor checks. Explanation is empty for v0.1.

| Page | Purpose (one line) | Comes from | New or existing |
|---|---|---|---|
| `doc/README.md` | Landing page: what Kirsch is, and where to start. | Rewritten. Pitch from `README.md:3`. | Existing, rewritten (ds-d7) |
| `doc/getting-started.md` | Quick start: download, verify, set a key, first run. The shortest path. | New. Download and verify commands move from `doc/install.md:51–128`. Key step links to `set-up-a-provider.md`. Name planned at `plan/build-plan-v0.2-v0.6.md:237`. | New (ds-d1) |
| `doc/install.md` | Full install: platforms, Git and ripgrep, each install route, `doctor`. | Keeps platforms, Git and ripgrep, each route's heading and prose, and `doctor`. Its download commands move to `getting-started.md`, and its key section moves to `set-up-a-provider.md`. | Existing, trimmed (ds-d1, ds-d2) |
| `doc/set-up-a-provider.md` | Choose an endpoint and model, export its key, check with `doctor`. Covers the `opencode` default with `minimax-m3` and the `anthropic` route. | New. Written from `doc/install.md:182–205` (moved there by ds-d2), `doc/configuration.md:60–88` (Secrets) and `:95–119` (`[provider]`). Defaults are taken from the table at `doc/configuration.md:107–114`. | New (ds-d2) |
| `doc/how-to.md` | Task guides for v0.1 only: resume the last session, start a fresh one, approve or reject a command, approve for the session, compact the history, read `/status`. | New. Resume prose moves from `doc/usage.md:30–33`. Commands are checked against the table at `doc/usage.md:260–270`. | New (ds-d3) |
| `doc/usage.md` | Reference: the screen, keys, scrolling, slash commands, tool cards, the approval model. | Resume prose removed (the code block at `:22–28` stays). | Existing, trimmed (ds-d4) |
| `doc/configuration.md` | Reference: config files, precedence, every key, environment variables, debug log. | Stays as it is. Links to `set-up-a-provider.md` are added (ds-d2). Links rewritten (ds-d6). | Existing |
| `doc/doctor.md` | Reference: what each `kirsch doctor` check means. | New. Takes `doc/troubleshooting.md:21–571`, from "Reading the report" to the lock section. | New (ds-d5) |
| `doc/troubleshooting.md` | Common failures and what to do, starting from what the user sees. | Keeps `:572–709` (common failures) and `:710` onward ("If the report does not explain it"). | Existing, trimmed (ds-d5) |
| `doc/images/` | Screenshots for the pages that embed them. | Hand-taken PNGs now (see Screenshots and demos). | Existing |

Proposed tree:

```
doc/
  README.md              landing page                         (rewritten)
  getting-started.md     quick start, tutorial                (new)
  install.md             full install                         (trimmed)
  set-up-a-provider.md   provider and model, how-to           (new)
  how-to.md              task guides, how-to                  (new)
  usage.md               screen, keys, slash commands         (reference, trimmed)
  configuration.md       config keys                          (reference)
  doctor.md              doctor checks                        (reference, new)
  troubleshooting.md     common failures                      (trimmed)
  images/                screenshots
```

Reviewer's expectation for a generator (UNVERIFIED): the doc pages carry no frontmatter today. Each opens on its H1 (`head -1 doc/*.md` shows `# Configuration`, `# Install Kirsch`, `# Kirsch — Documentation`, `# Troubleshooting`, `# Using Kirsch`). A generator may need a title per page, a sidebar order, and image paths relative to the site. ds-d8 records what the chosen generator needs, from that generator's own documentation, and adds any frontmatter itself. It does so after the pages are written (ds-d1 to ds-d5), because the pages come first.

Name clash to resolve before ds-d2 is written: `plan/milestones/milestone-6.md:44` plans `doc/providers.md` and `:73` also plans `doc/how-it-works.md`. Decision D9 names the provider guide's file. ds-d2 waits for the answer in the decisions table. If D9 is left blank, the name in the table above applies, and the report says so.

Explanation pages are not in v0.1. `doc/how-it-works.md` is planned for Milestone 6 (`plan/milestones/milestone-6.md:73`).

## Screenshots and demos

### Phase 1 — now

- **Keep the hand-taken PNGs.** Decision O12 (`plan/PROGRESS.md:410`, accepted 2026-11-03) reads "Two or three screenshots stored in the repository and linked from the README; no asciinema recording." Putting them on doc pages is a wider use than O12 names. Decision D5 asks the operator to confirm that.
- **Retake `doc/images/approval-card.png` after the cause is triaged.** The captured picture shows the assistant line as `the requestedI'll run command.` (seen in `doc/images/approval-card.png` at step 4). The cause is not established. It could be a rendering defect in the TUI, which would need a code fix and review, or a capture artefact, which needs only a retake. That is the triage in ds-d12.
- **Copy the current PNG into the workspace before replacing it (DIR-020).** The copy and its check are in ds-d13.
- **Capture process.** The repo's record of the process is `plan/testing/manual/user-testing/milestone-5-donovan.md:3031–3042` (steps 2 to 4 re-read at step 4): macOS Cmd+Shift+4, then Space, then a click on the Terminal window. The window is at least 80 columns by 24 rows (`plan/testing/manual/milestone-5.md:5330`).
- **Use a neutral workspace.** The current approval card shows the temporary workspace name `m5-test.Wo98TJ` (seen in the picture at step 4), not a home path. The retake uses a workspace named for the docs, with no personal path visible.
- **Embed the images on doc pages.** `start-screen.png` on `doc/getting-started.md`; `help-overlay.png` on the slash-commands section of `doc/usage.md`; `approval-card.png` under "An approval is pending" in `doc/usage.md` (`doc/usage.md:164`). Doc pages reference the images as `images/…`; the README keeps `doc/images/…`. The link check in ds-d9 covers the doc-page references and the README references.
- **Text-only screens are code blocks, not pictures.** `kirsch --help` and `kirsch doctor` output is pasted verbatim from a real run, into a fenced block. A picture of text cannot be searched or copied.

### Key-leak rules that already exist

- The operator checks, by eye, that there is "no API key text anywhere in the picture, including the scrollback" (`plan/testing/manual/milestone-5.md:5330`). This is a human check, not a mechanism.
- `kirsch doctor` never sends a prompt and makes no network calls (`doc/install.md:217–220`). Its key line names the variable, not the value (`doc/install.md:234`).
- API keys are never read from a config file (`doc/configuration.md:62–63`). A config file therefore cannot put a key into a capture.
- The scrollback of the existing PNGs has not been checked. The visible frame of `approval-card.png` was viewed at step 4, and `start-screen.png` and `help-overlay.png` were not viewed. The scrollback is UNVERIFIED for each of them.

### Phase 2 — later, code work, beyond O12

Phase 2 goes beyond decision O12. O12 accepts stored screenshots and no recording. The automated recordings below are a new kind of artefact. D5 must be answered before Phase 2 starts.

- **What it is.** VHS tape files in the repo (proposed under `site/tapes/`), regenerated in CI through the VHS GitHub Action, which the VHS README names ([VHS README](https://raw.githubusercontent.com/charmbracelet/vhs/main/README.md)). The Action itself was not fetched.
- **How no key is needed.** The tape drives the real binary against a local scripted endpoint on a loopback `base_url`. The loopback rule is in the code comment at `internal/config/config.go:538–542`. It is also documented at `doc/configuration.md:109` and `doc/troubleshooting.md:246`. The Anthropic client posts to `<base_url>/messages` (`internal/provider/anthropic/client.go:282`). The `api_key_env` variable can hold a dummy value. No key is typed, so none appears in a tape.
- **What is new.** The existing scripted provider, `Fake` (type at `internal/provider/fake.go:30`, `NewFake` at `:41`), is reachable from Go tests. The step 1 research recorded that no CLI flag or config key selects it. That was not re-checked at step 4. Phase 2 therefore needs a small piece of new code and a review by `@apone`. This is code work, not doc work.
- **The honest-screenshot rule.** A script can emit event sequences the real API never sends. A picture made from one shows a state that may not occur in real use. Every capture made this way is labelled as scripted on the page that shows it.
- **Overlap.** `plan/build-plan-v0.2-v0.6.md:199` plans `kirsch replay` (m10-d6), in which "a fake provider plays recorded streams". That is Milestone 10. Phase 2 should either wait for that decision or build its own harness. Two fake providers should not be built.
- **Tapes never type a key.** A tape file is plain text in the repo. Anything typed into it, including a key, is stored in the repo.

## Publishing pipeline

1. **Where the code lives.** Content stays in `doc/`. Generator configuration and the build live in a new `site/` folder. `site/` is the only place with Node dependencies. The build (`scripts/build.sh` and `.goreleaser.yaml`) needs no Node, and `npm` is a front door, not a dependency (`README.md:175–176`). The Node sentence at `README.md:177–178` is reworded to say so (ds-d7).
2. **Pull requests.** A new workflow file, `.github/workflows/docs.yml`, runs on pull requests that touch `doc/` or `site/`. It builds the site and runs the link check. It does not deploy. The existing `.github/workflows/ci.yml` runs on pushes and pull requests to `main` (`ci.yml:3–8`). The docs job is kept in its own file so the Go gate is not changed. The link-check tool is not chosen. It was not researched in step 1 and is UNVERIFIED.
3. **Tags.** The same workflow runs on `push` of a tag matching `v*`, the same trigger as the release workflow (`.github/workflows/release.yml:8`), and on `workflow_dispatch` for a dry run. The deploy job runs only on a tag. Whether the deploy waits for the release job is decision D11. A check that the deploy step is guarded by the tag ref is part of ds-d9's acceptance.
4. **Deploy.** GitHub Pages, from the Actions workflow. Pages limits as read on 2026-11-17 ([Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits)): sites up to 1 GB; a soft limit of 10 builds per hour that the page says does not apply to a custom Actions workflow; a soft bandwidth limit of 100 GB a month. Enabling Pages and the domain are operator actions (ds-d10).
5. **Join the other site.** Route (i) or (ii), by decision D1. See Bishop's recommendation above.
6. **Link check, minimum.** The check fails on a broken internal link in the built output. It also fails if any link points into `plan/`, if any internal link ends in `.md` when the site route has no `.md`, and if any `images/` reference names a file that does not exist. These are greps on the built output, run after the link checker. ds-d9 proves each one with a planted defect.

## Versioning and the rule against unreleased behaviour

- **Start with the latest release only** (decision D4). One site, rebuilt on each tag.
- **Add per-version folders only when v0.2 ships and v0.1 users still need their docs.** Its cost is build size and a version selector. Whether Starlight supports versioning is UNVERIFIED. Docusaurus's versioning feature is named in the research and was not read (UNVERIFIED).
- **Versioned by branch** is also possible. It was not checked against any tool. Its cost is that docs fixes need backporting by hand. UNVERIFIED.
- **The rule.** A page describes behaviour that is in the tagged release. Documentation for a feature lands in the same PR as the feature, but the site builds only from a tag, so the text reaches readers only once the release is out.
- **Text that will change with the milestones.** The release tag names are in each milestone title: Milestone 6 "More than one model (v0.2.0)" (`plan/milestones/milestone-6.md:1`); Milestone 7 "More than one wire (v0.3.0)" (`plan/milestones/milestone-7.md:1`); Milestone 8 "Headless (v0.4.0)" (`plan/milestones/milestone-8.md:1`); Milestone 9 "Front ends and machines (v0.5.0)" (`plan/milestones/milestone-9.md:1`); Milestone 10 "Review and memory (v0.6.0)" (`plan/milestones/milestone-10.md:1`). The sentences that will change are listed in the stale-text table above, and the pinned lines in `doc/install.md` move under ds-d1.
- **The in-app footer.** `internal/tui/modal.go:300` and `:308` print `docs: github.com/djm56/kirsch`. Decision D10 asks whether the footer follows the new docs address. The footer text was fixed by accepted decision O16 (`plan/PROGRESS.md:411`, accepted 2026-11-03), so changing it reopens an accepted decision. `plan/README.md:81–85` says the UI spec is law for anything visual, so the change needs a UI spec amendment first (ds-d16).

## Deliverables

Each deliverable that changes files runs under the normal loop: a brief, a review where it produces code, and a state sync after each step. Owners are the crew: `@lambert` for doc files, `@hicks` for code, `@apone` for review. The operator owns anything outside the repo.

Acceptance is an artefact that exists or a command whose output can be read. Every command that checks an absence is run first on a control: a copy of the file with a planted hit, or the pre-edit file, where the grep must find the hit. The control result is pasted beside the real result (DIR-001, DIR-026).

| ID | What | Files touched | Depends on | Who | Acceptance (artefact that exists or does not) |
|---|---|---|---|---|---|
| ds-d1 | Quick start page; download commands move out of install | `doc/getting-started.md` (new); `doc/install.md` (`:51–128` commands move; route heading and prose stay, with a link to the quick start; `:64` and `:146` pins become `<version>`; `:156` "`@latest` once a release exists" is reworded) | ds-d2 (order, see Order of work); D7 for the name; D8 for which mission writes it | `@lambert` | (a) `doc/getting-started.md` exists. (b) No pinned number: `grep -n '0\.1\.0' doc/getting-started.md` prints nothing. Control: the same grep on a copy with `VERSION=0.1.0` planted prints that line. The page's version line is `VERSION=<version>`, with one line saying the number is on the [releases page](https://github.com/djm56/kirsch/releases). `doc/install.md:58` already uses `<version>` for the pattern. (c) Every line inside a fenced `bash` block in `getting-started.md` is one of: (i) found verbatim with `grep -Fxq` in `doc/set-up-a-provider.md`, `doc/install.md`, `doc/configuration.md` or `doc/usage.md`, after `sed 's/ *# .*$//'` has removed any trailing `# …` comment from the line; (ii) the placeholder line `VERSION=<version>`; (iii) the first-run line `kirsch` or `kirsch --workspace <path>`, which is the `doc/usage.md:23` and `:25` form with `<path>` in place of `~/code/x`; or (iv) one of the download commands the pasted diff of `doc/install.md` shows removed. Control: one planted command that is none of (i) to (iv) must be rejected, and the pasted result must show it rejected. (d) `grep -n 'curl -fLO\|VERSION=' doc/install.md` prints nothing after the edit; before the edit it prints the lines at `:68`, `:73`, `:74`. (e) `grep -n 'use version\|go install .*@v0\.1\.0\|once a release exists' doc/install.md` prints nothing after the edit. Control: before the edit it prints `:64`, `:146` and `:156`. |
| ds-d2 | Provider-and-model guide; key section moves out of install | `doc/set-up-a-provider.md` (new); `doc/install.md` (`:182–205` moves out; the summary at `:3–5` and the blockquote at `:7–9` now point here); `doc/troubleshooting.md:596` (link `install.md#set-up-your-api-key` rewritten); `doc/configuration.md` (link added, nothing else) | D9 for the name | `@lambert` | (a) `doc/set-up-a-provider.md` exists. (b) Every default the guide states is in the table at `doc/configuration.md:107–114`. Pasted: for each default value in the guide's table, `grep -Fq` against `doc/configuration.md`. Control: a wrong default planted in a copy must be reported missing. (c) `grep -n 'install.md#set-up-your-api-key' doc/*.md` prints nothing. Control: before the edit it prints `troubleshooting.md:596`. (d) `grep -n 'OPENCODE_API_KEY' doc/set-up-a-provider.md` shows the export line from `doc/install.md:192`, verbatim. (e) `grep -n 'set up an API key' doc/install.md` prints nothing after the edit. Control: before the edit it prints `:4`. |
| ds-d3 | Task guides | `doc/how-to.md` (new) | None | `@lambert` | (a) `doc/how-to.md` exists. (b) Every slash command it names is in the table at `doc/usage.md:260–270`. Pasted: `` grep -o '`/[a-z]*`' doc/how-to.md ``, each result checked with `grep -Fq` against that range. Control: a name not in the table (`/nosuch`) planted in a copy must be reported missing. (c) `grep -c '/model' doc/how-to.md` prints 0. Control: the same grep on a copy with `/model` planted prints 1. Reason: the command list at `internal/tui/composer.go:453–455` holds these names (help, status, diff, files, approvals, new, compact, quit, exit). There is no `/model` command in v0.1. Where the guide tells a reader how to change the model, it points to `set-up-a-provider.md`, not to a command. |
| ds-d4 | Reference trim: resume prose | `doc/usage.md` only | ds-d3 (the prose moves) | `@lambert` | The paragraph that begins "`kirsch resume` picks up" (`doc/usage.md:30–33`) is absent from `doc/usage.md` and present in `doc/how-to.md`. Pasted: `grep -c 'kirsch resume` picks up' doc/usage.md` prints 0 and `grep -c 'kirsch resume` picks up' doc/how-to.md` prints 1. Control: the greps run on the pre-edit copy first, where the usage count is 1. The code block at `:22–28` stays in `usage.md`. |
| ds-d5 | Split troubleshooting | `doc/troubleshooting.md` (keeps `:572–709` and `:710` onward; `:1–20` is rewritten: a new H1, the intro at `:3–11` made true of this page, and the "What this page covers" box at `:13–17` rewritten to name the common failures and the page's own scope); `doc/doctor.md` (new, takes `:21–571`, with a new H1 and an intro that takes the doctor sentences at `:3–6`) | None | `@lambert` | (a) `doc/doctor.md` exists. (b) No `##` or `###` heading is lost or added. Pasted: `diff` of the sorted `^## ` and `^### ` headings of the pre-edit file against the sorted headings of `doc/doctor.md` and `doc/troubleshooting.md` together. The diff is empty. Control: the same `diff` on a copy with one heading deleted is not empty. H1 lines are checked on their own: pasted `grep -n '^# '` of the pre-edit file and of `doc/doctor.md` and `doc/troubleshooting.md` together. The only changes are the rewritten H1 of `doc/troubleshooting.md` and the new H1 of `doc/doctor.md`. (c) The in-page anchors that now cross files are rewritten to `<file>#<slug>` and listed in the report. They are: `doc/troubleshooting.md:118` (`#missing-ripgrep`), `:137` (`#not-in-a-repository`), `:290` (`#no-api-key`), `:327` (`#missing-ripgrep`), `:533` (`#lock-held`), `:549` (`#lock-held`), `:615` (`#repository`). Those before `:615` sit in the doctor sections and point at common failures; `:615` sits in common failures and points at the doctor section `## repository`. Local check at this step, not left to ds-d9: pasted `grep -n 'doctor.md#\|troubleshooting.md#' doc/*.md`; `ls doc/doctor.md doc/troubleshooting.md`; and pasted `grep -n` of the heading each anchor names in its target file. Control: a copy with `troubleshooting.md#nosuch` planted is reported as missing. ds-d9 re-confirms on the built output later. |
| ds-d6 | Link clean-up inside `doc/` | All `doc/*.md`, including the `../plan/` link at `doc/README.md:14` (rewritten again by ds-d7) | ds-d1 to ds-d5 | `@lambert` | (a) `grep -n '\.\./plan' doc/*.md` prints nothing. Control: the same grep on a copy with `[x](../plan/y.md)` planted prints that line. (b) Every cross-file `.md` link resolves to a page that exists after the split. Local check at this step: pasted `grep -oh '](\([^)#]*\.md\)' doc/*.md \| sort -u` lists each link target, and `ls doc/<target>` is run for each one, pasted. Control: a copy with `](nosuch.md)` planted is reported as missing by that `ls`. ds-d9 re-confirms on the built output later. (c) The anchors from ds-d5 are rewritten and shown by `grep -n 'doctor.md#\|troubleshooting.md#' doc/*.md`. |
| ds-d7 | README and index wording | `README.md` (`:3`, `:25`, `:69`, `:77`, `:86`, `:118`, `:122`, `:177–178`); `doc/README.md` (rewritten as landing page) | ds-d6 (they edit `doc/README.md` together); ds-d1 to ds-d5 (the landing page links the new pages); D3 and D4 for wording | `@lambert` | (a) `doc/README.md` links to the other doc pages, by name: `getting-started`, `install`, `set-up-a-provider`, `how-to`, `usage`, `configuration`, `doctor`, `troubleshooting`. Pasted: `grep -o '](\([a-z-]*\.md\))' doc/README.md \| sort`. Control: the same command on the pre-edit file lists the old pages (`install`, `configuration`, `usage`, `troubleshooting`). (b) The sentence at `README.md:177–178` says `site/` is the only place with Node dependencies, and that the build needs no Node. Pasted `grep -n 'site/' README.md` shows it. (c) `grep -n '0\.1\.0' README.md doc/README.md` returns only lines classed keep in the stale-text table. Control: the same grep run before the edit returns the known hits. (d) `grep -n 'after v0\.1\.0' README.md` prints nothing after the edit. Control: before the edit it prints `:86`. |
| ds-d8 | Site scaffold and build check | `site/` (new): generator config, package file, content config that reads `doc/*.md`; `.gitignore` gets `site/dist/` and `site/node_modules/`; `doc/*.md` gets only the frontmatter lines the chosen generator needs, if any (see (d)) | D3; ds-d1 to ds-d5 (the pages must exist to be built) | `@hicks` (code); `@apone` reviews | (a) `cd site && npm run build` exits 0. (b) A built page for each doc page, by name. A loop runs `test -f` on `site/dist/index.html` (the landing page) and on `site/dist/<name>/index.html` for each of `getting-started`, `install`, `set-up-a-provider`, `how-to`, `usage`, `configuration`, `doctor`, `troubleshooting`. A bare scaffold with no doc pages must make the loop fail. That run is the control, and its output is pasted. (c) `grep -n 'site/' .gitignore` shows the entries; before the edit it prints nothing. (d) A note in `.claude/memory/workspace/` names what the chosen generator needs for frontmatter, sidebar order and image paths, from its own documentation, with a URL. Any frontmatter the generator needs is added to `doc/*.md` in this step, and the note lists each line added. Until that note exists, the expectation in Proposed page structure is UNVERIFIED. |
| ds-d9 | Docs workflow: link check on pull requests, deploy on tags | `.github/workflows/docs.yml` (new) | ds-d6; ds-d7; ds-d8; ds-d14 (which waits on ds-d13 for its last picture); D11 | `@hicks` writes; `@apone` reviews; operator runs the first dry run | (a) The file exists. Triggers: `pull_request` with paths `doc/**` and `site/**`; `push` of tags `v*`; `workflow_dispatch`. (b) The deploy job's `if:` names the tag ref. `grep -n 'refs/tags/v' .github/workflows/docs.yml` shows it on the deploy job. A `workflow_dispatch` run skips the deploy job, and the operator pastes the run summary that shows the deploy job as skipped. (c) The dry run, in one invocation, runs the branches below. Branch 1 is a copy of the built output with planted defects: an internal link to a missing page, a link into `plan/`, an internal link ending in `.md`, and an `images/` reference to a missing file. The link check must reject each one. Branch 2 is the real build, which must pass. The `plan/` and `.md` greps must print the planted lines on branch 1 and nothing on branch 2. The operator pastes the output of each branch. (d) If the link checker is not the one the grep greps for, the report says so. |
| ds-d10 | Hosting and domain | GitHub Pages settings; DNS record; `site/public/CNAME` if a custom domain is used | D2 | Operator | The address answers over HTTPS. The operator pastes the output of `curl -sI https://<address>/`, and its status line reads `200`. |
| ds-d11 | Join the other site | The operator's site (outside this repo) | D1, D2, ds-d9, ds-d10 | Operator (site); crew for any repo side | Route (i): the other site's build pins a tag and renders the pages; the operator pastes the build log line that shows the pinned tag. Route (ii): the link appears in the live navigation; the operator pastes the output of a grep for the docs address in the live navigation HTML. |
| ds-d12 | Triage the garbled approval line | Workspace note only, under `.claude/memory/workspace/` | None | Operator takes the second capture. `@hicks` reproduces the garble in code only if the operator's capture shows it again; `@apone` reviews any fix. `@lambert` writes the note from Bishop's brief. Bishop reads the captured frame and the code and does not write files. | The note quotes the garbled text verbatim (`the requestedI'll run command.`), and names the cause as either a TUI defect (with a fix step in the plan) or a capture artefact. The cause is backed by pasted evidence: for a TUI defect, the command output of a test or a reproduction; for a capture artefact, a second capture that does or does not show the garble. A cause named without that evidence is not accepted. |
| ds-d13 | Retake `approval-card.png` | `doc/images/approval-card.png` | ds-d12; D5 | Operator (capture) | (a) Before anything is replaced, `cp doc/images/approval-card.png .claude/memory/workspace/docs-site/approval-card-before.png`, then `cmp` of the copy and the original exits 0. The workspace copy is kept. (b) `ls -la doc/images/approval-card.png` shows a size other than 152825 bytes. (c) The operator's verdict is pasted into the workspace note, beside the evidence, not in place of it: "approval-card.png: assistant line reads cleanly yes/no; no key text visible yes/no; no other window's text visible yes/no". Nothing is ticked on the operator's behalf (DIR-020). (d) The retake's workspace name is neutral: no personal path in the frame. |
| ds-d14 | Embed screenshots on doc pages | `doc/getting-started.md`; `doc/usage.md` | ds-d1, ds-d3, ds-d4; ds-d13 for the approval picture | `@lambert` | (i) After ds-d4: `grep -n 'images/' doc/getting-started.md doc/usage.md` shows a reference to `start-screen.png` and a reference to `help-overlay.png`. Control: before the edit the same grep prints nothing. (ii) After ds-d13: `grep -n 'images/' doc/*.md` shows a reference to each of `start-screen.png`, `help-overlay.png` and `approval-card.png`. Control: before the edit the same grep prints nothing. The link check in ds-d9 resolves each reference later. |
| ds-d15 | Text screens as code blocks | `doc/getting-started.md`; `doc/doctor.md` | ds-d1, ds-d5; an operator run | `@lambert`; operator runs the commands | The pasted output in each block matches the raw output saved in the workspace. Pasted: the `diff` of each block against its saved file, which is empty. Control: the same `diff` on a copy with one line changed is not empty. The report names each saved file by its full path. |
| ds-d16 | In-app footer and help link (code) | `internal/tui/modal.go:273`, `:300`, `:308`; `internal/tui/header_test.go:817`, `:1755`, `:1756`, `:1766`, `:1767`; `plan/spec/kirsch-ui-screens.md:428` (after the UI spec amendment) | D2, D10; the UI spec check (`plan/README.md:81–85`) | `@hicks`; `@apone` reviews | `grep -rn 'docs: github.com/djm56/kirsch' internal plan/spec` prints nothing after the change. Control: the same grep before the change prints the `internal` lines and the spec line listed in Files touched. The grep string includes `docs: ` because the module path `github.com/djm56/kirsch` also appears in import lines that must stay. Golden files: `grep -rl 'docs: github' internal` lists only `internal/tui/modal.go` and `internal/tui/header_test.go` today, so no golden carries the footer text. If the change alters a golden, that golden is re-captured in the same step and named in the report. The comments at `internal/tui/modal.go:273` and `internal/tui/header_test.go:817` quote the footer as `docs: github.com…`, with an ellipsis, so the grep above does not catch them. If the footer text changes, `grep -n 'docs: github.com…' internal/tui/modal.go internal/tui/header_test.go` must print nothing, and the comments must quote the new text. Control: before the change it prints the comment lines. |
| ds-d17 | CHANGELOG stale line | `CHANGELOG.md:12` (wording kept), and the `## [0.1.0] — 2026-11-17` heading the Milestone 5 close-out adds | Milestone 5 close-out mission makes the cut (recommended owner; not started) | `@lambert` runs the check | (a) The heading date is 2026-11-17. `gh release view v0.1.0 --json publishedAt` returned `2026-11-17T14:57:18Z` on 2026-11-17, and the cut uses that date. (b) Pasted: `grep -n '^## \[' CHANGELOG.md` shows `## [0.1.0] — 2026-11-17` below `## [Unreleased]`, and `grep -n 'not yet published' CHANGELOG.md` prints a line that falls between that heading and the next `## [` heading, or the end of the file. Control: the same greps on the pre-cut file find no `## [0.1.0]` heading, so the check fails there. (c) The line's wording is not changed. Pasted: `grep -n 'not yet published' CHANGELOG.md` before and after the cut, with the text identical. The history is not reworded. |
| ds-d18 | Phase 2: scripted screenshots in CI | `site/tapes/*.tape` (new); a CI job; a loopback scripted endpoint (new code) | D5; the Milestone 10 decision on replay | `@hicks` and `@apone` for code; operator | Later, not scheduled. Done when the tape files exist and the CI job that regenerates them exists. Beyond O12 (`plan/PROGRESS.md:410`). |
| ds-d19 | Per-version folders | `site/` config | D4; the v0.2 tag | `@hicks`; operator | Later, not scheduled. Done when the v0.2 tag has built and a v0.1 folder exists in the site output. |

## Operator decisions

Answer each question in a word or a line. Bishop's recommendation is given for each. Nothing is decided until you fill in the last column.

If you want to go with every recommendation, write "all recommended" — you will still need to answer D1 and D2 yourself.

| ID | Question | Options | Bishop's recommendation | Your answer |
|---|---|---|---|---|
| D1 | What is your other website built with? Answer in one word. | (a) Astro. (b) WordPress. (c) Something else; name it. (d) Not sure. | Tell us. The route depends on the answer. If you are not sure, plan for route (ii), which needs only a link on your site and a DNS record. | |
| D2 | What web address should the docs have? Answer a, b or c, and give the address for a. | (a) A subdomain: the part before your domain, such as `docs.<site>`. If you want no subdomain, GitHub's own free address, `djm56.github.io/kirsch`, works. (b) A folder on your main site, such as `<site>/kirsch`. This needs your host to pass the folder on to the docs. (c) No separate docs site; people read the docs on GitHub, at `github.com/djm56/kirsch/tree/main/doc`. | (a), a subdomain. You give the name. | |
| D3 | Which site generator should build the docs? Answer a, b or c. | (a) Astro Starlight. (b) Docusaurus: mature, versioning built in (not yet checked), needs Node, heavier. (c) Hugo: one program, no Node, fewer docs features out of the box (not yet checked). | (a) Starlight. Bishop recommends against MkDocs Material, whose maker gives an end of life of 2027-05-05 ([repository](https://github.com/squidfunk/mkdocs-material)). Glossary: Node is the JavaScript runtime the generator needs to build. A subpath is a folder after the domain. | |
| D4 | Should the site show one version or several? Answer a or b. | (a) The latest release only. (b) A folder per version, starting when v0.2 is out. | (a) now. Add folders when v0.2 ships. | |
| D5 | How should screenshots be made? Answer a, b or c. | (a) Your own pictures now, with the approval picture retaken; recordings later. (b) Recordings made automatically now, which needs new code. (c) No pictures. | (a). Recordings wait for the Milestone 10 decision on replay (`plan/build-plan-v0.2-v0.6.md:199`). Option (b) and the later recordings go beyond the screenshot decision made on 2026-11-03 [O12] (`plan/PROGRESS.md:410`). | |
| D6 | Should the GitHub wiki hold any docs? Answer a or b. | (a) No. (b) A FAQ or notes page only. | (a). A wiki cannot be tagged with a release or changed in the same change as the code (`plan/build-plan-v0.2-v0.6.md:47`). | |
| D7 | What should the quick start page be called? Answer a or b; for b, give the name. | (a) `getting-started.md`, the name the build plan gave it (`plan/build-plan-v0.2-v0.6.md:237`). (b) Another name. | (a). | |
| D8 | v0.1.0 was released on 2026-11-17. Milestone 5 is being closed out in a separate mission that has not started yet. Should the quick start tutorial be written as part of that close-out, or as part of this docs work? Answer a or b. | (a) This docs work. (b) The Milestone 5 close-out. | (a). The quick start is documentation, and it does not need to wait for the close-out. | |
| D9 | What should the provider guide's file be called? Answer a or b. | (a) `set-up-a-provider.md`, as this plan names it. (b) `providers.md`, the name Milestone 6 gives its page (`plan/milestones/milestone-6.md:44`). | (a). Milestone 6 then adds its own reference page; the pages coexist. Its instruction set is amended when Milestone 6 is written, not in this plan. | |
| D10 | Should the in-app footer link follow the new docs address? Answer a or b. This reopens an accepted decision. | (a) Yes, once D2 is answered. It is a code change with review, and the written screen design must change first (`plan/README.md:81–85`). (b) No; keep the GitHub link. | (a), after D2. Context: on 2026-11-03 the footer text was fixed as `docs: github.com/djm56/kirsch` [O16] (`plan/PROGRESS.md:411`). Answering (a) reopens that decision. | |
| D11 | Should the docs site publish only after the release finishes? Answer a or b. | The docs build and the step on GitHub that builds the release start when a version tag (`v*`) is pushed (`.github/workflows/release.yml:8`). The site could go live before the release files are attached. (a) Yes: the site goes live only after the release files are built. (b) No: the site goes live as soon as the tag is pushed. | (a). The site should not link to a download that is not there yet. The mechanism is part of ds-d9. | |

## Order of work

**Can start now, with no decision needed except where a row names one:**

- ds-d2 (provider guide) first. It is written before ds-d1, because the key line it holds is the source ds-d1 checks against. It starts with the name in D9; a blank D9 means the table's name.
- ds-d1 (quick start), after ds-d2. It starts under the name in D7; a rename is cheap if D7 changes. Which mission writes it is D8; a blank D8 means this docs work.
- ds-d3 (task guides), then ds-d4 (usage trim), which depends on it.
- ds-d5 (split troubleshooting).
- ds-d6 (link clean-up), after ds-d1 to ds-d5, so links are fixed once.
- ds-d7 (README and index wording), after ds-d6, because they edit `doc/README.md` together.
- ds-d12 (triage of the garbled line).
- ds-d14 for `start-screen.png` and `help-overlay.png` only, after ds-d1, ds-d3 and ds-d4 (check (i) of its acceptance).
**Waits on a decision or a step:**

- ds-d8 (site scaffold) waits on D3 and on ds-d1 to ds-d5.
- ds-d9 (docs workflow) waits on ds-d6, ds-d7, ds-d8, ds-d14 and D11.
- ds-d10 (hosting) waits on D2.
- ds-d11 (join the other site) waits on D1, D2, ds-d9 and ds-d10.
- ds-d13 (retake) waits on ds-d12 and on D5.
- ds-d14 for `approval-card.png` waits on ds-d13.
- ds-d15 waits on ds-d1 and ds-d5, and on an operator run of the commands.
- ds-d16 (footer) waits on D2, D10, and the UI spec check.
- ds-d17 (CHANGELOG check) waits on the Milestone 5 close-out mission's cut of the `## [0.1.0]` section.
- ds-d18 and ds-d19 are later and not scheduled.

## Risks and open questions

1. **The operator's site stack is unknown.** Decision D1 answers it. No stack is assumed in this plan.
2. **Hugo's folder behaviour.** UNVERIFIED. The configuration page read did not mention `baseURL` ([Hugo configuration](https://gohugo.io/configuration/)). It does not matter on the subdomain route.
3. **Starlight's release year.** Inferred from the page order and the context date, not shown on the page ([releases page](https://github.com/withastro/starlight/releases)).
4. **VitePress and Zensical.** Latest stable VitePress release UNVERIFIED (npm returned HTTP 403, [npm page](https://www.npmjs.com/package/vitepress)). Zensical version, stability and licence UNVERIFIED ([Zensical](https://zensical.org)).
5. **Serving a project site under a path of a custom domain.** Not covered by the GitHub custom-domain page. UNVERIFIED.
6. **Search features for each generator.** UNVERIFIED.
7. **Whether the wiki is a separate repository, and whether its changes go through PRs.** The repo says so (`plan/build-plan-v0.2-v0.6.md:47`). GitHub's wiki page does not confirm it. UNVERIFIED.
8. **Whether `doc/` is included in the release archives.** UNVERIFIED. Not checked, and the plan does not rely on it.
9. **Sizes and release years of VHS, freeze, asciinema and agg output.** Not measured. Years partly inferred or missing.
10. **Hosted docs platforms.** Cost, sync and export UNVERIFIED.
11. **The cause of the garbled approval line.** Only the symptom is confirmed (seen in `doc/images/approval-card.png`). Triage is ds-d12.
12. **The scrollback of the existing PNGs.** Not checked. `start-screen.png` and `help-overlay.png` were not viewed (see Key-leak rules).
13. **Link checker.** Not chosen and not researched. UNVERIFIED.
14. **Name clash with Milestone 6.** Resolved by D9 before ds-d2 is written.
15. **Docs and release timing.** The docs workflow and the release workflow start on a `v*` tag (`.github/workflows/release.yml:8`). Decision D11 answers whether the deploy waits for the release.
16. **Footer change and the UI spec.** `plan/README.md:81–85` says the UI spec is law for anything visual. The footer change needs a spec amendment first, and reopens O16. Decision D10 and ds-d16.
17. **Phase 2 overlaps Milestone 10.** `plan/build-plan-v0.2-v0.6.md:199`. Two fake providers should not be built.
18. **Captured images show environment details.** The current approval card shows a temporary workspace name (`m5-test.Wo98TJ`) and the model name (`minimax-m3`). A retake should not show anything personal, and the operator checks each picture for key text.
19. **Generator needs for frontmatter, sidebar order and image paths.** UNVERIFIED. The doc pages carry no frontmatter today. ds-d8 records what the chosen generator needs.
20. **CHANGELOG has no v0.1.0 heading.** The release date is 2026-11-17 (`gh release view v0.1.0`, 14:57:18 UTC). The cut that adds the heading belongs to the Milestone 5 close-out mission, not this plan. ds-d17 checks the line after that cut.
21. **Route (i) is not tested.** Its mechanism is a possibility recorded in the research, not a run. UNVERIFIED.

## Sources

The URLs the step 1 research read, with the date it read them and the note it recorded. None were re-fetched at step 2 or step 4. Each fact in this plan that rests on one carries its URL where it is used.

| URL | Used for | Read date and note |
|---|---|---|
| https://github.com/withastro/starlight/releases | Starlight versions and dates | 2026-11-17. Versions, 06 to 13 Nov, day and month only; year inferred. |
| https://docs.astro.build/en/guides/deploy/github/ | Astro `base` and `site` for GitHub Pages | 2026-11-17. No date on page. Astro's page, not Starlight's. |
| https://starlight.astro.build/ | Starlight self-description | 2026-11-17. No version on page. |
| https://github.com/facebook/docusaurus/releases | Docusaurus versions | 2026-11-17. Years shown. |
| https://docusaurus.io/docs/deployment | `baseUrl` and `url` | 2026-11-17. |
| https://github.com/squidfunk/mkdocs-material | End-of-life notice (2027-05-05), Zensical successor | 2026-11-17. README notice. |
| https://zensical.org | Zensical maintainers | 2026-11-17. No version or licence on page. |
| https://www.mkdocs.org/about/release-notes/ | MkDocs core latest release | 2026-11-17. Latest shown 1.6.1, 2024-08-30. |
| https://github.com/vuejs/vitepress/releases | VitePress alphas | 2026-11-17. Day and month only. |
| https://github.com/vuejs/vitepress/tags | VitePress tags | 2026-11-17. No stable tag seen. |
| https://www.npmjs.com/package/vitepress | VitePress stable on npm | 2026-11-17. HTTP 403. UNVERIFIED. |
| https://github.com/gohugoio/hugo/releases | Hugo versions | 2026-11-17. Years inferred from asset timestamps. |
| https://gohugo.io/configuration/ | Hugo `baseURL` | 2026-11-17. No `baseURL` on the page read. |
| https://github.com/charmbracelet/vhs/releases | VHS versions | 2026-11-17. Years partly inferred. |
| https://raw.githubusercontent.com/charmbracelet/vhs/main/README.md | VHS features, requirements, GitHub Action | 2026-11-17. |
| https://github.com/charmbracelet/freeze | freeze features and examples | 2026-11-17. |
| https://github.com/charmbracelet/freeze/releases | freeze versions | 2026-11-17. Year of v0.2.2 not on page. UNVERIFIED. |
| https://github.com/asciinema/agg | agg features | 2026-11-17. No release data on the repo page. |
| https://github.com/asciinema/asciinema/releases | asciinema versions | 2026-11-17. Day and month only. |
| https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits | Pages limits | 2026-11-17. |
| https://docs.github.com/en/pages/configuring-a-custom-domain-for-your-github-pages-site/managing-a-custom-domain-for-your-github-pages-site | Custom domains, subdomain CNAME | 2026-11-17. Project-path case not covered. |
| https://docs.github.com/en/communities/documenting-your-project-with-wikis/about-wikis | Wiki access, search, limits | 2026-11-17. PR and separate-repo questions not answered on the page. |

Repository facts in this plan were re-checked at step 4 against the files named, with the `file:line` at each citation. Where a fact could not be re-checked, it is marked UNVERIFIED.
