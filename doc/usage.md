# Using Kirsch

Kirsch is a terminal-native coding agent. This page covers the interface: how
the screen is laid out, every key it responds to, every slash command, and the
approval model that stands between a tool and your files.

> **Where this document is, and is not, complete.** Kirsch is mid-build. The
> layout, the key bindings, the slash-command surface and the approval flow all
> exist and behave as described below. The model does not: nothing here talks to
> an LLM yet, and the transcript you see on first run is scripted. Sessions and
> resume do not exist yet either. Both arrive in later milestones, and this page
> gains them when they do — sections that describe something not yet wired say
> so in place rather than leaving you to find out.
>
> The help overlay inside the program (`?`, or `/help`) is generated from the
> same binding table the program dispatches on, so it cannot describe a key it
> does not have. Where this page and the overlay disagree about a binding, the
> overlay is right. Where they disagree about *why*, this page is.

---

## Starting it

```bash
kirsch                       # run in the enclosing Git repository
kirsch --workspace ~/code/x  # run against a directory you name
kirsch --debug               # also write a structured debug log
```

Kirsch works inside one directory and never outside it — the **workspace**. With
no flag it is the enclosing Git repository; `--workspace` names one explicitly.
The project name in the header is that directory's base name.

The debug log goes to `$XDG_STATE_HOME/kirsch/debug.log`, or
`~/.local/state/kirsch/debug.log` when that variable is unset. It never writes
to stdout or stderr, so it cannot corrupt the display. `KIRSCH_DEBUG=1` turns it
on without the flag.

---

## The screen

```
 Kirsch ───────────────────────────────────────   header row 1 — wordmark + rule
 my-project ─ main ●                              header row 2 — session line

                                                  transcript (fills)
 ── you ───────────────────────
 Fix the Divide validation

 ▸ read_file calc/divide.go · 4ms · ok            tool card (collapsed)

 I found the issue in... ▌                        streaming assistant

 ──────────────────────────────────────────────   separator rule
 claude-sonnet-5 · ⠋ thinking · 12.4k tok         status bar
 ──────────────────────────────────────────────   separator rule
 > _                                              composer
```

**Header — two rows.** `Kirsch` stands alone on the first, with a rule running
off to the right. The second is the session: project name, branch, a `●` if the
worktree is dirty, and `(compacted)` once the session has been compacted. Below
60 columns the branch is dropped, and the dirty marker goes with it. Only the
first row carries a rule.

**Transcript.** Everything that has been said and done, oldest at the top. It
takes whatever rows the rest of the frame does not need.

**Status bar.** Model, current state, token count, and any active session
grants. Persistent warnings — a recovered session, another Kirsch running in the
same workspace, a missing `rg` — sit at the right and are the last thing dropped
when the bar runs out of room.

**Composer.** One row, growing to five as you add lines, then scrolling
internally. While a turn is running it dims and stops accepting input, and shows
`(input disabled, Esc cancels)` at the right.

**The frame keeps one blank column at each side**, and none at the top or
bottom. A terminal's rows are scarcer than its columns: the layout already
spends every row it has, so a blank row would come out of the transcript, which
at the smallest supported terminal is four rows tall. The right-hand column is
reserved rather than painted, so no row ever ends in trailing whitespace, and
both edges show your own terminal background rather than Kirsch's.

**Sizes.** Kirsch runs from 40×10 upwards. As the window narrows it drops the
token count (below 80), then the branch (below 60), and below 40 replaces the
transcript with a `terminal too narrow` notice rather than rendering something
illegible. As it shortens it shrinks the composer, then drops the two separator
rules together, then the header — both of its rows at once. Nothing panics at
any size, and a zero-sized terminal draws nothing at all.

---

## Keys

Kirsch is modal. Which keys do what depends on where the focus is, and the mode
changes only by an explicit key.

### Composing — the default

| Key | Action |
|---|---|
| `Enter` | Send |
| `Shift+Enter` | Newline, where the terminal sends `ESC`+`CR` for it |
| `Alt+Enter` | Newline, same decode path — **Option+Enter on a Mac keyboard produces nothing** |
| `Ctrl+J` | Newline — the one form every terminal can produce |
| `Tab` | Complete a unique slash-command prefix |
| `Ctrl+G` | Scroll back to the bottom of the transcript and re-pin it |
| `↑` on the first line | Move focus to the transcript (→ Browsing) |
| `Esc` / `Ctrl+C` | Cancel the running turn; if nothing is running, clear the composer |
| `Ctrl+C` twice within 1s | Force quit |

`q` is an ordinary character here — there is no bare-key quit. `?` is an
ordinary character too: a help overlay that fired mid-sentence would make the
composer unusable. Quitting is `/quit`, `/exit`, or `Ctrl+C` twice.

**`Ctrl+G` is worth knowing.** It is the only way back to the bottom that does
not send something first, and it is the only composing key that works *during* a
running turn — which is when watching the bottom of a stream is worth the most.
It moves the viewport and nothing else: your half-typed message, the card
selection and the mode are all left exactly as they were. `End` would have been
the obvious key and is not available: the text area already uses it for
end-of-line, as it uses `Ctrl+End` for end-of-input.

### Browsing — after `↑`

| Key | Action |
|---|---|
| `↑` / `↓` | Move the card selection (plain text blocks are skipped) |
| `PgUp` / `PgDn` | Scroll without moving the selection |
| `Home` / `End` | Top / bottom — `End` re-pins |
| `g` / `G` | Top / bottom, for terminals that swallow `Home` and `End` |
| `Enter` | Expand or collapse the selected card |
| `d` | Open the full content or diff in a modal |
| `?` | Help overlay |
| `Esc` | Back to the composer — does **not** move the view |
| `↓` past the last card | Back to the composer |

Scrolling and selection are separate: `PgUp`/`PgDn` move the view and leave the
selection where it is, `↑`/`↓` move the selection and scroll only as far as they
must to keep it visible.

`g`/`G` exist because `Home` and `End` are not universally reachable — macOS
Terminal sends both as a sequence the toolkit does not decode. Kirsch normalises
that on the way in, so both pairs work; `g`/`G` are the belt to that braces.

### An approval is pending

| Key | Action |
|---|---|
| `y` | Approve, this once |
| `a` | Approve and grant for the rest of the session — `run_command` only |
| `n` | Reject |
| `d` | Open the detail or diff |
| `?` | Help overlay |
| `Esc` / `Ctrl+C` | Cancel the turn — counts as a rejection |

Every other key is swallowed. Nothing reaches the composer while an approval is
up, which is deliberate: an approval is the one moment where a keystroke meant
for something else must not be interpreted as an answer.

### A modal is open

| Key | Action |
|---|---|
| `j` / `k` / `↑` / `↓` | Scroll a line |
| `PgUp` / `PgDn` | Scroll a page |
| `g` / `G` / `Home` / `End` | Top / bottom |
| `Esc` | Close, returning to whatever mode you were in |
| `?` | Also closes the help overlay |

Closing returns you to the *previous* mode, so `Esc` from a modal opened over a
pending approval puts you back at the approval, not at the composer.

### A confirm prompt is up

One-line questions — "Discard the running turn and start a new session?",
"Clear all session grants?", "Paste 12KB into the composer?" — appear
immediately above the composer, not over the status bar.

| Key | Action |
|---|---|
| `y` | Yes |
| `n` / `Esc` | No |

`Esc` is not spelt out on the prompt itself: it does the same thing as `n`, the
row already names a way out, and the cells it would cost come straight out of
the question — which at 40 columns is the part being truncated.

### Everywhere

`Ctrl+C` is context-dependent by design: it cancels a running turn, and
otherwise arms a force quit. Two presses inside a second always quit, from any
mode.

**Pasting** is literal — pasted text is never interpreted as keys, and newlines
in it are preserved. A paste over 8KB asks first. A paste that arrives while an
overlay is open is discarded rather than queued, so text cannot appear in a
composer you could not see it landing in.

---

## Scrolling and pinning

The transcript is **pinned to the bottom**. New content while pinned keeps it
pinned, so a running turn stays in view without you touching anything.

Scrolling up **unpins** it, and a `↓ 3 new` indicator appears at the bottom
right counting what has arrived since. The view then stays exactly where you put
it: typing does not drag it back, and neither does returning to the composer
with `Esc`. Reading scrollback is a deliberate act and Kirsch does not undo it
for you.

It re-pins when you:

- scroll back to the bottom yourself,
- press `Ctrl+G` in the composer,
- press `End` or `G` while browsing,
- send a message, or
- submit a slash command.

A slash command re-pins because it is a request and its answer belongs on
screen. The two exceptions are the commands that answer entirely in the dim hint
under the composer — an unknown command, and a debug command missing its
argument. Neither puts anything in the transcript, so re-pinning would cost you
your place to report a typo.

---

## Slash commands

Type `/` at the start of a single line. A command must be the whole line:
multi-line text beginning with `/` is an ordinary message, so a pasted diff or
stack trace is never mistaken for one. `Tab` completes a unique prefix.

| Command | What it does |
|---|---|
| `/help` | The help overlay |
| `/status` | A one-line notice: model, branch, token count |
| `/diff` | The working tree as a diff, in a modal |
| `/files` | A notice listing the files touched this session |
| `/approvals` | Lists active session grants, and offers to clear them |
| `/new` | Start a new session — confirms first if a turn is running |
| `/compact` | Compact the conversation history |
| `/quit` | Quit |
| `/exit` | Quit — the same thing, spelt the other way |

An unknown command produces a dim hint under the composer, never an error card,
and is **never sent to the model**.

**In the current build**, `/diff`, `/files` and `/compact` answer from the
scripted Milestone 0 fixture rather than from your repository — a placeholder
diff, a fixed file list, and `nothing to compact yet`. The command surface,
the completion, the re-pin behaviour and the modals around them are real; the
content behind those three is not yet. `/read`, `/ls`, `/search`, `/gitstatus`
and `/gitdiff` below *do* read your actual workspace.

### Debug commands — temporary

Milestone 1 added five commands so a real repository can be read from inside the
interface before the model drives the tools itself. They are labelled
`debug (M1 only)` in the help overlay and **will be removed** once the model
drives tools directly. Do not build a habit on them.

| Command | Tool |
|---|---|
| `/read <path>` | `read_file` |
| `/ls [path]` | `list_files` |
| `/search <query>` | `search_code` |
| `/gitstatus` | `git_status` |
| `/gitdiff` | `git_diff` |

---

## Tool cards

Every tool call appears in the transcript as a card, collapsed by default:

```
▸ read_file calc/divide.go · 4ms · ok
```

Glyph, tool name, what it acted on, the result, and how long it took. `Enter`
expands it inline, capped at 200 rendered lines with a
`‹200 of 4,181 lines — press d for full output›` marker at the cut; `d` opens
the whole thing in a modal. The cap is there because an unbounded expansion
makes the scrollback unusable, which is worse than truncating it.

Where a cap was hit further upstream — a 200KB read, a 4000-token result — the
card says so in the same shape: `‹truncated — 200KB cap›`.

Tool output is sanitised before it is rendered. Escape sequences are stripped
rather than passed through, because an unstripped sequence hands arbitrary
terminal control to whatever the tool printed. Carriage-return runs render as
their final segment, tabs become four spaces, and other control characters
become a dim `·`.

---

## The approval model

Nothing consequential happens silently. Before Kirsch writes a file or runs a
command, the action appears as a card and waits:

```
┌─ approval required ─────────────────────────────┐
│ apply_patch — add input validation              │
│                                                 │
│ files: 2 changed (calc/divide.go,               │
│        calc/divide_test.go)                     │
│                                                 │
│ [y] approve   [n] reject   [d] view diff        │
└─────────────────────────────────────────────────┘
```

`y` approves this one action. `n` rejects it. `d` shows you exactly what it
would do — the diff for a patch, the full command and its environment for a
command — before you answer.

**`a` grants for the session, and only commands offer it.** A session grant
means "stop asking me about this command for the rest of this session"; it is
offered on `run_command` cards and never on `apply_patch` ones, because a patch
is a different change every time and a blanket yes to it would mean nothing. The
status bar shows a count of active grants, `/approvals` lists them, and it
offers to clear them.

Grants live and die with the session. Nothing about them is written to disk, so
starting Kirsch again starts from no grants.

---

## When colour or Unicode is not available

Set `NO_COLOR`, run under `TERM=dumb`, or pipe the output somewhere that is not
a terminal, and Kirsch emits **no escape sequences at all** — not merely no
visible colour. The layout is identical: same rows, same columns, same box
positions. Every state is identified by its glyph rather than by its colour, so
nothing becomes ambiguous.

Where the locale is not UTF-8, every glyph falls back to an ASCII form of the
same width — `>` for `▸`, `[ok]` for `✓`, `...` for `⋯`, `+-|` for box drawing,
and the wordmark to `K I R S C H`. The layout does not move, because every
substitution preserves width.

Kirsch never paints a full-screen background. It sets foreground colours and
accents only, so it sits inside your terminal theme rather than replacing it.

---

## If something looks wrong

Run with the debug log on and read it afterwards:

```bash
kirsch --debug
cat ~/.local/state/kirsch/debug.log
```

Rendering problems are worth reporting with your terminal (`echo $TERM`) and its
size (`tput cols`, `tput lines`) — most of the ones found so far have been
specific to one terminal's encoding of one key, and the size is what makes a
layout defect reproducible.
