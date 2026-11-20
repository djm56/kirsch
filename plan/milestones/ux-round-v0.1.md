# UX round — v0.1.2 to v0.1.5

> **Status: Executable — written 2026-11-19 (mission-20261119-03) from `plan/pre-m6-ux-notes.md`.** Live status lives in `plan/PROGRESS.md`; this page carries none.

## Goal

Fix the daily usability problems found after v0.1.1, before Milestone 6 starts. The work ships in small releases.

## Ground rules

- One mission per release, in release order. v0.1.2 closes before v0.1.3 starts.
- Every code step goes to `@hicks`. `@apone` reviews it as the very next step. `@vasquez` arrives only by escalation.
- The mission filter in `plan/build-plan-v0.2-v0.6.md` ("How to apply this document") applies to every mission in this round.
- Tagging and releasing are operator-owned. Bishop does not tag.
- The docs website lives in a separate repository. The operator updates it after each tag.
- Mouse support stays out of this round. UI spec §12 rule 4 ("no mouse, ever") is unchanged by it.

## Releases

| Release | ID | Title | Notes items | Done when |
| --- | --- | --- | --- | --- |
| v0.1.2 | ux-d1 | Output limit and cut-off notice | 6, 8 | Unknown models get 16384 output tokens; model-table rows changed only with a cited source. The `minimax-m3` and `minimax-m2.7` rows stay at 4096 unless a published source is found, so v0.1.2 does not lift the cut-off on the default model by itself; the 16384 default helps models not in the table, such as `claude-haiku-5-5`. A reply that stops on `max_tokens` shows the notice from ui-spec §3.7, and the stop reason and the limit sent are written on `assistant.message` (§7, amendment 100), so the notice shows on resume. Tests pin all of it. |
| v0.1.2 | ux-d2 | Keyboard scrolling and help screen | 9, 10, 13 | `PgUp`/`PgDn` scroll from Composing; `j`/`k` scroll one line in Browsing; the help overlay, in the 17-row layout of amendment 100, lists `Ctrl+G`, `PgUp`/`PgDn` (composing) and `j`/`k`, `g`/`G` (browsing); screen 06 and the goldens match. |
| v0.1.3 | ux-d3 | Copy the last reply | 5 | `/copy` in all its forms (`/copy`, `/copy code`, `/copy code N`) puts source text on the clipboard through the local clipboard tool and OSC 52, with the routes and notices of ui-spec §6 and plan amendment 101; tests pin the code-block grammar and the notices. |
| v0.1.3 | ux-d4 | Select a reply and copy it with `c` | 12, 5 | Assistant replies can be selected in Browsing, and `↑`/`↓` stop on them; `c` copies the selected reply; the key-hint row and the help-overlay redraw show both (ui-spec §3.9, §4.2, §5.2; plan amendment 101). |
| v0.1.4 | ux-d5 | Markdown rendering | 1 | Assistant replies render bold, headings, lists and code fences; no background is painted; `NO_COLOR` and `TERM=dumb` fall back to plain text; a config key turns it off. |
| v0.1.4 | ux-d6 | Code highlighting | 4 | Code in the content view, in diff lines and in reply code fences is coloured by language; `+`/`-` stay the line markers. |
| v0.1.5 | ux-d7 | Scrollbar | 2 | A one-column position indicator on the transcript's right edge, with a non-Unicode fallback; no mouse. |
| v0.1.5 | ux-d8 | Long lines in the modal | 11 | Long lines in the content and diff modal can be read in full (wrap or sideways scroll, per UX-D7). |

## Manual tests

The operator's hands-on checks for each release are in [`plan/testing/manual/ux-round-v0.1.md`](../testing/manual/ux-round-v0.1.md). The v0.1.2 section was run on 2026-11-20 before its tag. On the operator's direction (2026-11-20), the sections for v0.1.3 to v0.1.5 are run together as one test at the end of the round.

## Deferred

- **Notes item 3 (mouse wheel) and item 7 (scrollbar click and drag)** are past this round. Both need a ui-spec amendment to §1 goal 3 and §12 rule 4 first.
- Turning on mouse reporting breaks the terminal's own text selection. That is the reason these items wait. The per-reply and per-block copies in ux-d3 and ux-d4 cover the copying need.

## Decisions

| ID | Question | Answer | Needed before |
| --- | --- | --- | --- |
| UX-D1 | Output limit for a model not in the model table | Answered 2026-11-19: a model not in the table gets 16384 output tokens. Its context window stays the cautious figure it gets today. A model-table row gets new figures only from a published source, cited by URL. The `minimax-m3` and `minimax-m2.7` rows stay at 4096 unless a published source is found, so v0.1.2 does not lift the cut-off on the default model by itself; the 16384 default helps models not in the table, such as `claude-haiku-5-5`. | v0.1.2 (answered) |
| UX-D2 | Line scroll keys in Browsing | Answered 2026-11-19: `j`/`k` scroll the transcript one line in Browsing mode. | v0.1.2 (answered) |
| UX-D3 | Page scroll from Composing | Answered 2026-11-19: `PgUp`/`PgDn` in Composing scroll the transcript a page, without changing mode or moving the card selection. | v0.1.2 (answered) |
| UX-D4 | How many missions per release | Answered 2026-11-19: one mission per release. | — (answered) |
| UX-D5 | Copy keys, and how a code block is picked | Answered 2026-11-20 by Bishop's recommendation (operator's standing go-ahead; the operator may overturn it): `↑`/`↓` stop on assistant replies as well as cards, `c` in Browsing copies the selected reply's source text, and `/copy`, `/copy code` and `/copy code N` copy the last reply, its last fenced code block, or block N. Details in plan amendment 101. | v0.1.3 (answered) |
| UX-D6 | Markdown while a reply streams | Open: plain text until the reply completes, or format each finished block as it closes. | v0.1.4 |
| UX-D7 | Long lines in the modal | Open: wrap them, or add sideways scroll with `h`/`l`. | v0.1.5 |
| UX-D8 | Config key name and default for markdown rendering | Open. Suggested: `ui.render_markdown = true`. | v0.1.4 |

## Spec amendments

Each release amends the UI spec before its code step.

- **v0.1.2** by plan amendment 100: the keys, the help overlay, the cut-off notice, and the §7 stop-reason fields on `assistant.message`.
- **v0.1.3** by plan amendment 101: selecting replies (ui-spec §3.9), copy keys (§5.2), `/copy` (§6), and a second help-overlay redraw within the same height rule as v0.1.2.
- **v0.1.4** must reverse "no syntax highlighting in v0.1" (ui-spec §3.2 and §9).
- **v0.1.5** adds the scrollbar glyphs to §10.2, and amends §4.1 (modal) for UX-D7.

## Relation to Milestone 6

- **m6-d1** replaces `internal/provider/models.go` with `models.toml`. ux-d1 changes figures only, not structure, so the two do not collide. m6-d1 must carry over the 16384 unknown-model output default from ux-d1.
- **m6-d6** deals with tool calls in a cut-off reply. It does not add the notice; ux-d1 does.
- **m6-d9** adds a `/model` picker. It removes the need to edit config to change model.

## Known gaps after v0.1.2

- The context budget reserves 4096 output tokens (`internal/app/budget.go`, `outputReserve`), while a request may ask for 16384. After compaction, input plus `max_tokens` can exceed the window. Not changed in v0.1.2.
- On a terminal too small to show the transcript, `PgUp` and `PgDn` from the composer clear the `↓ N new` indicator.
- The no-scroll test for the approval prompt relies on a mutant rather than a same-path control.
