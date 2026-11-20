# Using Kirsch

Kirsch is a terminal-native coding agent. This page covers the interface: how
the screen is laid out, every key it responds to, every slash command, and the
approval model that stands between a tool and your files.

> **What this page covers.** Kirsch runs real model turns against your
> repository, keeps a durable session log, and resumes it on launch or with
> `kirsch resume`. The layout, the key bindings, the slash commands and the
> approval flow all work as described below. Sections that describe a limit
> say so in place.
>
> The help overlay inside the program (`?`, or `/help`) prints its own short
> table of keys. That table is text only: the key handlers do not read it, so a
> row that is wrong has to be corrected by hand. The table also leaves out some
> keys the program accepts, such as `Home` and `Esc` in Browsing. Where this page
> and the overlay disagree, the program decides what a key does, and this page
> describes that.

---

## Starting it

```bash
kirsch                       # run in the enclosing Git repository
kirsch resume                # resume the most recent session for this workspace
kirsch --new                 # start a fresh session even when a previous session exists
kirsch --workspace ~/code/x  # run against a directory you name
kirsch --debug               # also write a structured debug log
```

`kirsch resume` picks up the most recent session for this workspace, including
its session grants. If another Kirsch is already running in the workspace, it
starts a fresh session instead and says so. `kirsch --new` starts a fresh
session even when a previous session exists.

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
 claude-sonnet-5 · ⠋ thinking · in 12.4k out 980 cache 8.9k/1.2k · $0.04   status bar
 ──────────────────────────────────────────────   separator rule
 > _                                              composer
```

The `cache` figures are tokens read from the cache, then tokens written to it.

**Header — two rows.** `Kirsch` stands alone on the first, with a rule running
off to the right. The second is the session: project name, branch, a `●` if the
worktree is dirty, and `(compacted)` once the session has been compacted. Below
60 columns the branch is dropped, and the dirty marker goes with it. Only the
first row carries a rule.

**Transcript.** Everything that has been said and done, oldest at the top. It
takes whatever rows the rest of the frame does not need.

**Status bar.** Model, current state, session token totals and cost, and any
active session grants. Persistent warnings, for example a recovered session or
another Kirsch running in the same workspace, sit at the right. No narrowing
step removes them, but on a very narrow terminal a long one is cut at the right
edge.

**Composer.** One row to start, growing as you add lines up to a fixed maximum
height. Past that maximum the composer scrolls up and down to keep the line you
are typing on in view. A line wider than the terminal is cut at the edge with a
`⋯` marker; the composer does not scroll sideways. While a turn is running it
dims, refuses ordinary typing, and shows `(input disabled, Esc cancels)` at the
right. Some keys still act during a turn, for example `Esc`, `Ctrl+J`, `Alt+Enter`,
`Tab`, `Ctrl+G` and `PgUp`/`PgDn`; the Composing table marks the ones it lists.

**The frame keeps one blank column at each side**, and none at the top or
bottom. A terminal's rows are scarcer than its columns: the layout already
spends every row it has, so a blank row would come out of the transcript, which
at the smallest supported terminal is four rows tall. The right-hand column is
reserved rather than painted, so no row ever ends in trailing whitespace, and
both edges show your own terminal background rather than Kirsch's.

**Sizes.** Kirsch runs from 40×10 upwards. As the window narrows it drops the
status bar's token figures (below 80 columns), then the branch (below 60), and
below 40 replaces the transcript with a `terminal too narrow` notice rather than
rendering something illegible. Below 80 columns the status bar also shortens its
left-hand text as the width shrinks, starting with the grant count, and keeps the
state for as long as it can. As the height shrinks it shrinks the composer, then
drops the two separator rules together, then the header — both of its rows at
once. Nothing panics at any size, and a zero-sized terminal draws nothing at all.

---

## Keys

Kirsch is modal. Which keys do what depends on where the focus is, and the mode
changes only by an explicit key.

### Composing — the default

| Key | Action |
|---|---|
| `Enter` | Send |
| `Shift+Enter` | Newline, where the terminal sends `ESC`+`CR` for it. Works while a turn is running |
| `Alt+Enter` | Newline, same decode path — **Option+Enter on a Mac keyboard produces nothing**. Works while a turn is running |
| `Ctrl+J` | Newline — the one form every terminal can produce. Works while a turn is running |
| `Tab` | While typing a slash-command name (no space yet), complete a unique prefix; an ambiguous or unknown prefix does nothing. In any other text, move to the cards (→ Browsing). Works while a turn is running |
| `Ctrl+G` | Scroll back to the bottom of the transcript and re-pin it. Works while a turn is running |
| `PgUp` / `PgDn` | Scroll the transcript one page. Your typed text, the mode and the selection stay as they are. Works while a turn is running |
| `↑` / `↓` on the first / last row | Previous / next entry from this session's history (up to 100); past the newest, back to your unsent draft. Ignored while a turn is running |
| `Shift+↑` | Move focus to the cards (→ Browsing). Works while a turn is running |
| `Esc` / `Ctrl+C` | Cancel the running turn; if nothing is running, clear the composer |
| `Ctrl+C` twice within 1s | Force quit |

`q` is an ordinary character here — there is no bare-key quit. `?` is an
ordinary character too: a help overlay that fired mid-sentence would make the
composer unusable. Quitting is `/quit`, `/exit`, or `Ctrl+C` twice.

**`Ctrl+G` is worth knowing.** It re-pins the transcript from the composer without
sending anything, so it is the key to reach for when you have scrolled up, decided
not to send, and want the bottom back. It works *during* a running turn, which is
when watching the bottom of a stream is worth the most, and `PgUp` and `PgDn` work
then too. While a turn runs, the composer refuses ordinary typing, but `Ctrl+J` and
`Alt+Enter` still add a newline, and `Esc` still cancels the turn.
`Ctrl+G` moves the viewport and nothing else: your half-typed message, the
selection and the mode are all left exactly as they were. `End` would have been
the obvious key and is not available: the text area already uses it for
end-of-line, as it uses `Ctrl+End` for end-of-input.

**The key-hint line.** On a terminal at least 24 rows tall, the bottom row lists the keys that act in the current focus — `↑↓ history · ⇧↑/Tab cards · /help` in the composer, for example. It goes blank while a modal or confirm prompt is open, and is dropped on shorter terminals.

### Browsing — after `Shift+↑` or `Tab`

| Key | Action |
|---|---|
| `↑` / `↓` | Move the selection between cards and assistant replies; your own messages are skipped |
| `c` | Copy the selected reply (see [Copying](#copying)) |
| `PgUp` / `PgDn` | Scroll without moving the selection |
| `j` / `k` | Scroll one line without moving the selection. Stops at the top and the bottom |
| `Home` / `End` | Top / bottom — `End` re-pins |
| `g` / `G` | Top / bottom, for terminals that swallow `Home` and `End` |
| `Enter` | Tool cards: switch between the 10-line preview and the head line only. On a reply: nothing |
| `d` | Open the full content or diff in a modal. On a reply: nothing |
| `?` | Help overlay |
| `Tab` / `Esc` | Back to the composer — does **not** move the view |
| `↓` past the last selectable item | Back to the composer |

Scrolling and selection are separate: `PgUp`/`PgDn` move the view and leave the
selection where it is, `↑`/`↓` move the selection and scroll only as far as they
must to keep it visible. `PgDn`, `j`, `End` and `G` re-pin the transcript and clear
the `↓ N new` indicator when they scroll it to the bottom. `↓` re-pins only if the
scroll it causes reaches the bottom, so a `↓` that stops one row short leaves the
transcript unpinned.

`g`/`G` exist because `Home` and `End` are not universally reachable — macOS
Terminal sends both as a sequence the toolkit does not decode. Kirsch normalises
that on the way in, so the pairs work; `g`/`G` are the belt to that braces.

### An approval is pending

| Key | Action |
|---|---|
| `y` | Approve, this once |
| `a` | Approve and grant for the rest of the session — `run_command` only |
| `n` | Reject |
| `d` | Open the detail or diff |
| `?` | Help overlay |
| `Esc` / `Ctrl+C` | Cancel the turn — recorded as cancelled, not rejected |
| `Tab` / `Shift+↑` | Re-arm a card that a stray key released |

Every other key is swallowed, and none reaches the composer. A key that is not one of the
above and not a navigation key also *releases* the card: the composer shows
`approval pending — Tab to return to the card`, and `y`, `a` and `n` do nothing until `Tab`
or `Shift+↑` re-arms it. That is deliberate: typing a command while a card waits — the `n`
in `/new` — must not be read as an answer. Navigation keys never release; pasted text is
dropped. `j` and `k` are not navigation keys here, so they release the card.
`PgUp` and `PgDn` are navigation keys: they neither release the card nor scroll
the transcript.

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
"Clear 2 session grants?", "Paste 12KB into the composer?" — appear
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
in it are preserved. Pasted tabs become spaces in the composer, so tab-indented
text arrives with space indentation. A terminal that does not send a paste as
one block delivers each tab as a Tab key press instead: the first moves focus
to the cards, where the text that follows is read as card keys and not typed.
A paste over 8KB asks first. A paste that arrives while an overlay is open is
discarded rather than queued, so text cannot appear in a composer you could not
see it landing in.

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
screen. The exception is an unknown command. It answers in the dim hint under
the composer and puts nothing in the transcript, so re-pinning would cost you
your place to report a typo.

---

## A reply that stops early

A model stops a reply when it reaches the output limit Kirsch sent with that
request. When that happens Kirsch shows a dim line under the reply, after its
text and above any tool cards from the same reply:

```text
· reply cut off — output limit reached (N tokens)
```

On a terminal whose locale is not UTF-8, the line starts with `.` instead of `·`,
and `-` stands in for `—`.

`N` is the limit sent with that request, as a plain number. The line appears as
soon as the reply stops, and it appears again when you resume the session.
Sessions written before v0.1.2 do not record whether a reply was cut off, so they
show no line.

The line means the model had more to say and the limit stopped it. These help:

- Ask for less in one message, or split the work into two requests.
- Use a model with a larger output limit. Set the model with
  `provider.<name>.model` (see [configuration.md](configuration.md)). Each model
  in Kirsch's table has its own limit. A model that is not in the table gets
  16384 tokens.

A reply made only of thinking, with no text, that is cut off shows the line as
it stops but not after a resume.

---

## Slash commands

Type `/` at the start of a single line. A command must be the whole line:
multi-line text beginning with `/` is an ordinary message, so a pasted diff or
stack trace is never mistaken for one. `Tab` completes a unique prefix. `/c` and
`/co` do not complete, because `/compact` and `/copy` share those prefixes.

| Command | What it does |
|---|---|
| `/help` | The help overlay |
| `/status` | A notice summarising the session — for example the model, endpoint, branch, dirty count, token totals and compaction level |
| `/diff` | The unstaged working-tree diff, in a modal |
| `/files` | A notice listing the files changed by approved patches this session. A resumed session starts this list empty and does not rebuild it from the earlier transcript |
| `/approvals` | Lists active session grants, and offers to clear them |
| `/new` | Start a new session — confirms first if a turn is running |
| `/compact` | Compact the conversation history |
| `/quit` | Quit |
| `/exit` | Quit — the same thing, spelt the other way |
| `/copy` | Copy the last reply's text to the clipboard. See [Copying](#copying) |
| `/copy code` | Copy the last code block in the last reply |
| `/copy code N` | Copy code block N, counted from 1, in the last reply |

An unknown command produces a dim hint under the composer, never an error card,
and is **never sent to the model**.

---

## Copying

You can copy a model reply to your clipboard in two ways: select the reply in
Browsing and press `c`, or type a `/copy` command in the composer.

- `c` copies the selected reply. Select a reply with `↑` or `↓` in Browsing first.
- `/copy` copies the last reply.
- `/copy code` copies the last code block in the last reply.
- `/copy code 2` copies code block 2 in the last reply. Blocks are counted from 1.

**What is copied.** The reply's text as the model sent it, so markdown is kept:
`**bold**` stays as `**bold**`, and the screen's line wrapping is not. Kirsch
cleans the text before it copies it, so it pastes safely. It removes terminal
escape sequences and control characters, and it turns Windows line ends into
plain ones. Tabs are kept, and long lines are not cut. A reply that is still
streaming, or whose turn was cancelled, copies the text received so far.

**Code blocks.** A code block starts with a line of three or more backticks
(` ``` `) or tildes (`~~~`). That line may be indented, as a fence inside a list
item often is. The block ends with a closing line of the same character, at least
as long as the opening run, with nothing after it but spaces or tabs. A block that
is never closed runs to the end of the reply. Count the opening lines from the top
of the reply to find a block's number; Kirsch does not number them on screen.
The copy holds only the text between the fence lines, without the fence lines or
the language name after the opening fence. An empty block copies
nothing. `/copy code` and `/copy code N` look only at the last reply. To copy a
code block from an older reply, select that reply and press `c`, which copies
the whole reply.

**`/copy` is refused while a turn is running**, like every other slash command.
`c` works from Browsing.

**Two routes.** Kirsch sends the text two ways. It hands it to the clipboard tool
on the computer Kirsch runs on, and it waits up to 3 seconds for that tool. It
also writes an escape sequence, OSC 52, to the terminal, which a terminal can use
to set the clipboard. Kirsch sends the sequence only when it is writing to a real
terminal and `TERM` is not `dumb`.

**Notices.** The result appears as a hint under the composer:

| Notice | What it means |
|---|---|
| `copied N characters` | The clipboard tool took the text. `N` counts characters, not bytes |
| `sent N characters (OSC 52)` | The clipboard tool failed or took longer than 3 seconds, and the sequence went to the terminal. Kirsch cannot tell whether the terminal used it, so paste to check |
| `copy failed: no clipboard route` | Neither route worked |
| `nothing to copy` | There is no reply yet, `c` was pressed with nothing selected, or the text is empty once cleaned |
| `c copies replies only` | `c` was pressed on a selected card. Only replies can be copied |
| `no code block in the last reply` | `/copy code` or `/copy code N` was used, and the last reply has no code block |
| `the last reply has 1 code block`, or `the last reply has M code blocks` | The block number was 0, larger than the number of blocks, or too large to read. `M` counts every block, empty ones included |
| `code block K is empty` | Block `K` has no text between its fences |
| `usage: /copy [code [N]]` | The text after `/copy` is not one of the accepted forms |

**When a notice goes.** A copy notice is a hint under the composer, like the
unknown-command hint. Any key that reaches the composer's text area clears it:
typing, or moving the cursor. Sending a message, pressing `Esc` in the composer,
and an approval card arriving also clear it. A newer copy notice replaces the
one showing. If another hint is showing, such as the unknown-command hint or the
usage hint, a copy result that arrives later does not replace it. `Ctrl+G` and
`PgUp`/`PgDn` leave a copy notice alone.

**Known gaps that matter to you.**

- OSC 52 reaches the clipboard only when tmux allows it. Kirsch writes a bare
  OSC 52 sequence, so tmux needs `set-clipboard on`, and the terminal must accept
  the sequence.
- macOS Terminal.app ignores OSC 52. On that terminal only the clipboard tool
  on the computer can copy.
- Over SSH, `copied` can mean the text reached a clipboard tool on the remote
  machine, not on yours. The OSC 52 route is the one that reaches your own
  computer.
- Some terminals cap or drop large OSC 52 copies, so a very long copy can fail
  on that route. The notice still names the route Kirsch used, not what your
  screen shows. To copy less, use `/copy code N` for one block.
- On a very short terminal, the layout gives the composer no spare row for its
  hint, so a copy notice in the hint under the composer is not visible.
- A draft of five or more lines fills the composer, and then a copy notice is not
  shown under it. A draft of four lines or fewer shows the notice.
- When any notice is showing under the composer, the help overlay at 80 by 28 is
  one row short, and `G` shows its last rows.
- A very large copy can garble the screen once. The next redraw clears it.

See [Copy did not reach my clipboard](troubleshooting.md#copy-did-not-reach-my-clipboard)
for what to do when a copy fails.

---

## Tool cards

Every tool call appears in the transcript as a card, collapsed by default:

```
▸ read_file calc/divide.go · 4ms · ok
```

Glyph, tool name, what it acted on, the result, and how long it took. By default,
a tool card opens as a preview showing the first 10 lines. When lines are hidden,
a marker reads `‹10 of 4,181 lines — d full output · Enter collapse›`. `Enter`
toggles between preview and collapsed (head only). `d` opens the whole thing in a
modal. The 10-line preview keeps the scrollback legible while tools are running.

Where a cap was hit further upstream — a 200KB read, a 4000-token result — the
card says so in the same shape: `‹truncated — 200KB cap›`.

Tool output is sanitised before it is rendered. Escape sequences are stripped
rather than passed through, because an unstripped sequence hands arbitrary
terminal control to whatever the tool printed. Carriage-return runs render as
their final segment, tabs become four spaces, and other control characters
become a dim `·`.

---

## The approval model

A patch appears as a card and waits, and no setting turns that off. A command
runs without a card when it is exactly one of the built-in allowlist commands,
with no further arguments, or when a session grant covers it. Every other
command appears as a card and waits, unless
`policy.require_approval_for_commands` is `false` in your configuration; shells
and runners wait even then (see [`configuration.md`](configuration.md)). Check
that setting when you expect a card and none appears:

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

Grants live and die with the session. The session log records each grant, so
`kirsch resume` (or auto-resume) restores them, unless another Kirsch is already
running in the workspace, in which case a fresh session starts. A grant you
clear with `/approvals` is recorded in the session log as cleared, so it stays
cleared after `kirsch resume`. A new session, including one started with
`/new`, starts with none.

---

## When colour or Unicode is not available

Set `NO_COLOR` to a non-empty value, run under `TERM=dumb`, or pipe the output
somewhere that is not a terminal, and Kirsch emits **no colour codes and no
style sequences** — not merely no visible colour. An empty `NO_COLOR`
(`NO_COLOR=`) is treated as unset, as the no-color.org convention asks, so it
does not turn colour off by itself. The layout is identical: same rows, same
columns, same box positions. Every state is identified by its glyph rather
than by its colour, so nothing becomes ambiguous.

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
