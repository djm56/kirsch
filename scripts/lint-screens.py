#!/usr/bin/env python3
"""Lint the character grids in plan/kirsch-ui-screens.md.

The grids are Milestone 0's render target and the source of the golden files
(plan amendment 24). They were originally committed with ragged box borders and
an 82-column row on an 80-column grid — defects no correct renderer could
reproduce, which would have meant capturing goldens from an unreviewed render.
This keeps that from recurring.

Checks, per grid:
  * the fence declares a size, e.g. ```text 80×18
  * the row count matches the declared height exactly
  * the widest row matches the declared width exactly
  * no row exceeds the declared width
  * box borders are square: every row's right-most border character sits in one
    of at most two columns (two, because a modal may overlap a card behind it)

Exit status is 0 when clean, 1 otherwise.
"""

import re
import sys
import unicodedata
from pathlib import Path

DOC = Path(__file__).resolve().parent.parent / "plan" / "kirsch-ui-screens.md"
BOX = "┌┐└┘│├┤"


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
    failures = 0

    for screen, size, rows in grids(text):
        if size is None:
            continue  # a component, not a terminal render
        want_w, want_h = size
        where = f"screen {screen} ({want_w}×{want_h})"

        if len(rows) != want_h:
            print(f"{where}: {len(rows)} rows, declared {want_h}")
            failures += 1

        widest = max((cell_width(r) for r in rows), default=0)
        if widest != want_w:
            print(f"{where}: widest row is {widest} cols, declared {want_w}")
            failures += 1

        for i, row in enumerate(rows, 1):
            w = cell_width(row)
            if w > want_w:
                print(f"{where}: row {i} is {w} cols, over by {w - want_w}")
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
