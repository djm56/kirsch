# v0.1.3: cues the captures do not show

This file is written by hand. The inventory test in `internal/tui/capture_test.go` does not
treat it as a stray file, and `TestCaptureSectionsAreAccountedFor` requires that every
section of the walkthrough's v0.1.3 part either has a capture here, whose name starts
`sNN-`, or has an entry below that starts `- Section N`.

An entry names a cue that no frame of Kirsch can show, and the reason. A step that has no
entry here is not thereby claimed to be captured: the captures are the files in this
folder, and each one's header lists the steps that built it.

- Section 3, step 1: the hint variants `sent N characters (OSC 52)` and `copy failed: no
  clipboard route`. Which one the operator sees depends on the clipboard route their
  terminal and machine give. The capture wires a route that reports the local tool
  worked, so it shows `copied N characters` only.
- Section 3, step 2: the paste into another app holds the second reply's text. The system
  clipboard is outside Kirsch's screen. The `N` in `s03-copy-reply` is the number of
  characters the capture's copy route was handed, not the content of any clipboard.
- Section 3, step 3: after switching to another app and back, the view did not move and
  the `┃` is still on the second reply. Kirsch gets no event from an app switch, so no
  frame records one. `s03-copy-reply` is the frame Kirsch drew right after the copy.
- Section 4, step 3: nothing is copied. That is a statement about the system clipboard.
  `s04-copy-card-hint` shows the hint `c copies replies only` and no `copied` notice.
- Section 5, step 2: the paste holds the second reply's text. The system clipboard again.
- Section 6, steps 1 to 3: what each paste holds, and that it has no fence line and no `go`.
  The system clipboard again. `s06-copy-code` shows the notice after `/copy code`.
- Section 7, step 2: the clipboard still holds the first block's lines. The system
  clipboard again. `s07-copy-code-range` shows the range notice.
