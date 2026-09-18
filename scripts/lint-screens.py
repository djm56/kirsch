#!/usr/bin/env python3
"""Lint the character grids in plan/spec/kirsch-ui-screens.md.

The grids are Milestone 0's render target and the source of the golden files
(plan amendment 24). They were originally committed with ragged box borders and
an 82-column row on an 80-column grid — defects no correct renderer could
reproduce, which would have meant capturing goldens from an unreviewed render.
This keeps that from recurring.

The declared size is the TERMINAL size, not the content's. The frame keeps a
blank margin at its left and right edges — framePad columns of it, read out of
internal/tui/view.go rather than repeated here — so a grid of a W-column
terminal carries at most W - 2*framePad columns of content, offset framePad
columns from the left. Markdown cannot hold trailing whitespace and the
generator trims it, so the right margin is never present in the file and every
row measures at most W - framePad.

What the checks guarantee, taken together:

  * the grid is a render of the terminal it declares — the fence states a size,
    the row count is that terminal's height, and no row is wider than that
    terminal less its right margin;
  * the frame's left margin is on every row that has content, so a grid cannot
    be committed from a renderer that has stopped insetting the frame;
  * the frame's content reaches the last column available to it on at least one
    row — the widest row is exactly W - framePad and never less — so a grid
    cannot be committed from a renderer that has quietly lost columns, which is
    the check the original "widest row equals the declared width" was making
    before the margin existed;
  * no row carries trailing whitespace, which markdown cannot hold;
  * box borders are square: every row's right-most border character sits in one
    of at most two columns (two, because a modal may overlap a card behind it).

Exit status is 0 when clean, 1 otherwise.
"""

import re
import sys
import unicodedata
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DOC = ROOT / "plan" / "spec" / "kirsch-ui-screens.md"
VIEW = ROOT / "internal" / "tui" / "view.go"
BOX = "┌┐└┘│├┤"


def frame_pad() -> int:
    """Read framePad out of the renderer.

    Parsed rather than repeated: the margin is the renderer's decision, and a
    copy of it here would go stale silently — the grids would be regenerated
    from the new margin and linted against the old one, which is the drift this
    script exists to catch rather than to commit.
    """
    m = re.search(r"^const framePad = (\d+)$", VIEW.read_text(encoding="utf-8"), re.M)
    if not m:
        sys.exit(f"{VIEW}: no `const framePad = N` to read the frame margin from")
    return int(m.group(1))


def cell_width(s: str) -> int:
    """Display width in terminal cells: wide runes count 2, combining 0."""
    w = 0
    for ch in s:
        if unicodedata.combining(ch):
            continue
        w += 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1
    return w


def grids(text: str):
    """Yield (screen, declared_size, rows) for each render grid.

    Colour-map blocks follow a **Colours** heading and are skipped; screen 00 is
    a component rather than a full-screen render and carries no size.
    """
    screen = None
    after_colours = False
    in_fence = False
    size = None
    buf: list[str] = []

    for line in text.split("\n"):
        heading = re.match(r"^## (\d\d) · ", line)
        if heading:
            screen, after_colours = heading.group(1), False
            continue
        if line.strip() == "**Colours**":
            after_colours = True
            continue
        if line.startswith("```"):
            if not in_fence:
                in_fence, buf = True, []
                m = re.match(r"^```text (\d+)×(\d+)$", line.strip())
                size = (int(m.group(1)), int(m.group(2))) if m else None
            else:
                in_fence = False
                if screen and not after_colours:
                    yield screen, size, buf
            continue
        if in_fence:
            buf.append(line)


def main() -> int:
    text = DOC.read_text(encoding="utf-8")
    pad = frame_pad()
    margin = " " * pad
    failures = 0

    for screen, size, rows in grids(text):
        if size is None:
            continue  # a component, not a terminal render
        want_w, want_h = size
        where = f"screen {screen} ({want_w}×{want_h})"
        # The widest a row can measure once the right margin is trimmed away:
        # the left margin plus every content column.
        want_widest = want_w - pad

        if len(rows) != want_h:
            print(f"{where}: {len(rows)} rows, declared {want_h}")
            failures += 1

        widest = max((cell_width(r) for r in rows), default=0)
        if widest != want_widest:
            print(f"{where}: widest row is {widest} cols, want {want_widest} — "
                  f"{want_w} columns less the {pad}-column right margin")
            failures += 1

        for i, row in enumerate(rows, 1):
            w = cell_width(row)
            if w > want_widest:
                print(f"{where}: row {i} is {w} cols, over by {w - want_widest} — "
                      f"it has written into the right margin")
                failures += 1
            if row and not row.startswith(margin):
                print(f"{where}: row {i} does not open with the {pad}-column "
                      f"left margin: {row!r}")
                failures += 1
            if row != row.rstrip():
                print(f"{where}: row {i} has trailing whitespace")
                failures += 1

        right_edges = set()
        for row in rows:
            cols = [cell_width(row[:i]) for i, ch in enumerate(row) if ch in BOX]
            if cols:
                right_edges.add(cols[-1])
        if len(right_edges) > 2:
            print(f"{where}: ragged box borders — right edges at {sorted(right_edges)}")
            failures += 1

    if failures:
        print(f"\n{failures} problem(s) in {DOC.name}")
        return 1
    print(f"{DOC.name}: grids OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
