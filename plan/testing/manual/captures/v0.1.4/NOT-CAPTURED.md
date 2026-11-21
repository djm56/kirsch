# v0.1.4: cues the captures do not show

This file is written by hand. The inventory test in `internal/tui/capture_test.go` does not
treat it as a stray file, and `TestCaptureSectionsAreAccountedFor` requires that every
section of the walkthrough's v0.1.4 part either has a capture here, whose name starts
`sNN-`, or has an entry below that starts `- Section N`.

An entry names a cue that no frame of Kirsch can show, and the reason. A step that has no
entry here is not thereby claimed to be captured: the captures are the files in this
folder, and each one's header lists the steps that built it.

A colour cue is shown by the `.annot.txt` file, which lists the palette role and the SGR
code of each run of cells the program emitted, for example `fg=SynKeyword(38;5;141)`. It
does not show how a terminal draws that code, and no capture does.

- Section 4, step 3: no `notes.go` in the folder after the patch is rejected. That is the
  state of the file system, not of the screen. `s04-approval-rejected` shows the rejected
  card.
- Section 8, steps 1 and 2: what each paste holds, which is the source markdown of a reply,
  with its `#`, its asterisks and its fence lines, and no `•`. The system clipboard is
  outside Kirsch's screen, so no frame records it.
