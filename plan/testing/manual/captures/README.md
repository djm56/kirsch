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

## Regenerating

From the repository root:

```
KIRSCH_CAPTURE_WRITE=1 go test -count=1 -run TestCapture ./internal/tui
```

Without `KIRSCH_CAPTURE_WRITE=1`, the same tests compare each file with the current
render and fail, showing the differing lines, when one has gone stale. They also
fail for a file with no table row and a table row with no file. Writing never
deletes: remove a stray file by hand.

To add a release, add rows to `captureTable` with that release's folder name and
regenerate.
