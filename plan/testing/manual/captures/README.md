# Screen captures for the manual walkthroughs

## What these are

Each file under `<release>/` is one screen of Kirsch, taken from Kirsch's own
render. `<name>.txt` is the frame as a terminal shows it, with colour off, under a
short header that names the state, the size, the glyph set and the keys or canned
arrivals that produced it. `<name>.annot.txt` exists for states where colour
matters. It lists, run by run, the palette role and SGR codes of the colour-on
frame of the same state. The form is documented at the top of
`internal/tui/capture_test.go`.

A walkthrough quotes these files, so a claim about a screen rests on output the
program drew.

There is a folder for each release whose walkthrough cites captures. The v0.1.5 files
have descriptive names. The v0.1.3 and v0.1.4 files start `sNN-`, where `NN` is the
number of the walkthrough section the state belongs to. A release folder may also hold
`NOT-CAPTURED.md`, written by hand: it names the cues of a section that no frame can
show, such as what a paste into another app holds, and why. It does not list every step
that has no capture. `TestCaptureSectionsAreAccountedFor` in `internal/tui` reads the
section headings of the v0.1.3 and v0.1.4 parts of `plan/testing/manual/ux-round-v0.1.md`
and fails for a section that has neither a file starting `sNN-` nor an entry starting
`- Section N` in its folder's `NOT-CAPTURED.md`.

The files must not be reformatted or edited by hand: the frames carry trailing
spaces and exact column positions, and an editor that trims whitespace or an
auto-formatter makes the comparison test fail. Regenerate them instead.

## Where they come from

The captures come from the real render with canned inputs and no model. Each state
is built in `internal/tui/capture_test.go` by sending canned transcript messages
and real key messages through the real `Model`'s `Update`, then calling `View()`.
No provider, no network and no model key are involved. The table at the top of that
file (`captureTable`) lists every state, its terminal size and the steps that reach
it.

A state's messages are sent in the order a live turn sends the ones it shows: the typed
text, then `Enter`, then the reply's text and the end of the turn. A live turn also sends
messages the captures leave out: a budget message before the reply and a usage message
and a session-usage message after it. A tool call's messages are the ones a model-driven
call of that tool sends. A read_file call sends the card's start with no target, then its
target once the whole call has arrived, then its result, which carries the target too. An
apply_patch call sends the card's start, then its approval request, then its result, and
no target message, because its input names no target. Each is pinned to what the real
path sends, from a stream the real Anthropic decoder reads, by
`internal/app/capture_drift_test.go` and `internal/app/live_tool_test.go`. Where a
walkthrough section is marked *(model)*, the reply text is canned and the file's header
has a `# canned:` line that says so. A state built for a walkthrough that sets an
environment or a config key records it in its header: `# env` for the environment that
colour and glyphs are detected from (the `NO_COLOR=1` states call `DetectCaps` under it)
and `# render_markdown` for the config key.

A key whose command the program runs in the background, such as the copy that `c` and
`/copy` start, has its command run by the harness and its result delivered after the key.
The copy route is a stand-in that reports the local tool worked and writes nowhere, so
no state touches a clipboard.

## What is canned

The frames are Kirsch's own render, and the values below are canned, so a screen you see
yourself shows other values in their place. The list names values that a reader is likely to
take for Kirsch's own. It is not a complete list, and no test checks it against the table.

- The version in the header: `0.1.5-dev`. The help footer in the v0.1.3 states shows it too,
  so it reads `kirsch v0.1.5-dev`, not the version of its release.
- The header's project and branch, `my-project` and `main`, the dirty marker `●` after the
  branch (`Options.Dirty` is true in the shared inputs file), and the status line's model
  name, `claude-sonnet-5-5`, with its `idle` state and token count. No usage message is
  sent, so the count does not move.
- File names and file text: `calc/divide.go`, `calc/usage.go`, `calc/long.txt`,
  `README.md`, `hello.go`, `tool.py`, and the lines of each. The words in a reply and in
  a user message, and the reply of every walkthrough section marked *(model)*.
- The footer of a tool card, for example `‹10 of 11 lines — d full output · Enter collapse›`:
  Kirsch draws its shape, and its numbers come from the canned file.
- The duration on a tool card's head, `· <n>ms ·`, drawn where the layout sets
  `ShowDuration`: every tool completion the captures send carries the one canned
  value `toolElapsedMs` from `internal/tui/testdata/capture-inputs.json`. A live card shows the tool's own run, and the time the user takes
  to answer an approval is left out of it (`approvalWait` in `toolExecutor.Invoke`, `internal/app/app.go`).
- Notices: `one`, `two` and `three` in the `scrollbar-mid-new` state, and the `↓ 3 new`
  count they cause.
- The `/diff` text. It is git's own output for a canned file pair, with a canned
  function name in the hunk header and canned blob hashes in the index row. Real git
  cuts that header's function context at 80 bytes. The `/diff` text of v0.1.4 is git's
  output for a modified file and a new one registered with `git add -N`.
- The `/status` notice's field values in the v0.1.3 states (endpoint, key source,
  session, tokens and the rest). The real ones come from config and the environment. Its
  dirty count is 1 because the header's dirty marker is drawn, so the count and the
  marker agree.
- The `N` in `copied N characters`. It is the number of characters the stand-in copy
  route was handed, which is the canned reply or code block.
- The patch approval in v0.1.4: the model's description, and the patch it proposes. The
  messages built from them are pinned to what a model-driven `apply_patch` call sends,
  and the `read_file` messages of every state to what a model-driven `read_file` call
  sends; the messages of each are read from the shared file below.
- In the chain replay, the `new` count of section 4 and every offset. They follow from
  the canned replies' length, so a live reply gives other numbers.

The shape is not canned: where a bar and its thumb sit, which rows are blank, where a
marker or a number sits, which role a run of cells has.

The canned values and the patterns the real producers are held to are in
`internal/tui/testdata/capture-inputs.json`. The tests in `internal/tui` and in
`internal/app` read that file. A test in `internal/app` isolates git from the user's
git configuration, because a setting such as `core.abbrev` changes the blob hashes the
files hold.

## Annotation files

`<name>.annot.txt` lists the runs of cells of the colour-on frame, or for a state with an
`# env` header line the frame that environment gives. Most files list only the runs that
are neither plain nor a plain text, muted, dim or border colour (the header says `emph`).
`s03-readfile-modal-go` lists every run that has an attribute, the dim ones too, because
its cue is a dim line number. A line such as `fg=SynKeyword(38;5;141)` gives the palette
role and the SGR code the program emitted. It does not show how a terminal draws that
code. An annotation with no run says so in its last line: for the `NO_COLOR=1` states
that is the evidence that the frame has no attribute, and `TestCaptureNoColourStatesCarryNoEscapes`
also checks that the frame holds no escape and that the same steps without `NO_COLOR` do.

## The chain replay

`<release>/chain-replay.txt` is not a screen. It is a record of one session, the
walkthrough's sections run in order through one `Model`: for each section, the
selection, whether any row of the selected item is in the transcript window (the
gutter is drawn only then), the key-hint row, the mode, the scroll offset and pin, the
open modal, the size and the glyph set, before and after it. A step that says to press
an arrow until the gutter shows on a card is run as a search that stops when that card
is selected and in view, and the record says which key got there, how many presses it
took and whether Shift+↑ was needed to return to Browsing.

Section 4 is recorded in each of these orderings, because the order of its inputs changes
the result: the main path presses its keys before the reply's text arrives and leaves a
`↓ N new` count above 0; retry branch 1 presses them while the reply is streaming, and
retry branch 2 presses them after the reply has finished (the walkthrough's `reply finished
first` case), and each retry branch leaves `new 0`. The file's header says which is which,
and the chain goes on from the main path. The header also names the messages a real resume
sends that the replay does not. That list is held to `App.ResumeLastSession` by a test in
`internal/app`. The replay is written and compared like the captures, from
`internal/tui/chain_replay_test.go`.

## Regenerating

From the repository root:

```
KIRSCH_CAPTURE_WRITE=1 go test -count=1 -run TestCapture ./internal/tui
```

Without `KIRSCH_CAPTURE_WRITE=1`, the same tests compare each file with the current
render and fail, showing the differing lines, when one has gone stale. They also
fail for a file that no table row or extra produces and for a row or extra with no
file. Writing never deletes: remove a stray file by hand.

To add a release, add rows to `captureTable` with that release's folder name and
regenerate.
