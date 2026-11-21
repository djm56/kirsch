#!/usr/bin/env python3
"""Lint the character grids in plan/spec/kirsch-ui-screens.md.

The grids are Milestone 0's render target and the source of the golden files
(plan amendment 24). They were originally committed with ragged box borders and
an 82-column row on an 80-column grid — defects no correct renderer could
reproduce, which would have meant capturing goldens from an unreviewed render.
This keeps that from recurring.

The declared size is the TERMINAL size, not the content's. The frame keeps a
margin at its left and right edges — framePad columns of it, read out of
internal/tui/view.go rather than repeated here — so a grid of a W-column
terminal carries at most W - 2*framePad columns of content, offset framePad
columns from the left. Markdown cannot hold trailing whitespace and the
generator trims it, so the right margin is absent from the file and every row
measures at most W - framePad — with one exception. The transcript scrollbar
(plan amendment 103) is drawn into the right margin, so a row may end in a
scrollbar glyph in the grid's last column, column W. Only the scrollbar glyphs
are admitted there: `│` and `┃` in a Unicode grid, `|` and `#` in a grid that is
all ASCII, where "all ASCII" is judged on the grid with each last-column glyph
set aside, so a `│` in the last column of an otherwise ASCII grid is not
admitted. Any other character in that column is still a row that has written
into the margin.

The lint cannot tell a transcript row from another, but it checks the shape the
bar has. The rows that end in a glyph form one contiguous run, and within that
run the thumb glyph (`┃` or `#`) forms one contiguous run. A glyph on a lone row
away from the run, a gap in the run, or two thumbs fails.

What the checks guarantee, taken together:

  * the grid is a render of the terminal it declares — the fence states a size,
    the row count is that terminal's height, and no row is wider than that
    terminal less its right margin, once a scrollbar glyph in the last column
    is set aside;
  * the frame's left margin is on every row that has content, so a grid cannot
    be committed from a renderer that has stopped insetting the frame;
  * the frame's content reaches the last column available to it on at least one
    row — the widest row, scrollbar glyph and the padding before it set aside,
    is exactly W - framePad and never less — so a grid
    cannot be committed from a renderer that has quietly lost columns, which is
    the check the original "widest row equals the declared width" was making
    before the margin existed;
  * no row carries trailing whitespace, which markdown cannot hold;
  * box borders are square: every row's right-most border character sits in one
    of at most two columns (two, because a modal may overlap a card behind it).
    A scrollbar glyph in the last column is not a border and is set aside first;
  * non-grid content that a raw-grid regeneration would silently discard is still
    present: the colour-map format header, screen 00's wordmark and ASCII
    fallback, every screen's colour map, screen 05's collapsed grant list, and
    the NOTE annotations.

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
# The scrollbar glyphs (ui-spec §10.2): track then thumb, in the Unicode form and
# in the ASCII fallback. They are the only characters admitted in a grid's last
# column, the right margin.
BAR_UNICODE = "│┃"
BAR_ASCII = "|#"


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


def bar_glyphs(rows: list[str], want_w: int) -> str:
    """The scrollbar glyphs a grid may carry: ASCII ones if it is all ASCII.

    A row exactly want_w cells wide has its last character set aside before the
    test, since that is where a glyph would sit. Otherwise a `│` there would make
    an ASCII grid look Unicode and admit itself.
    """
    bodies = [r[:-1] if cell_width(r) == want_w else r for r in rows]
    return BAR_ASCII if all(b.isascii() for b in bodies) else BAR_UNICODE


def runs(flags: list[bool]) -> int:
    """The number of maximal runs of True in flags."""
    return sum(1 for i, f in enumerate(flags) if f and (i == 0 or not flags[i - 1]))


def split_bar(row: str, want_w: int, glyphs: str) -> tuple[str, str]:
    """Split a scrollbar glyph off the end of a row.

    Returns (body, glyph). The glyph is taken only when the row is exactly want_w
    cells wide, so it sits in the grid's last column, and it is one of glyphs.
    Anything else comes back whole with an empty glyph, so the margin rules see
    it as they did before the scrollbar existed.
    """
    if row and row[-1] in glyphs and cell_width(row) == want_w:
        return row[:-1], row[-1]
    return row, ""


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


def colour_blocks(text: str):
    """Yield (screen, lines) for every **Colours** block in screen order.

    A screen's colour map is the ```text fence that follows its **Colours**
    heading. Screen 00 is included even though it carries no sized grid.
    """
    screen = None
    in_colours = False
    in_fence = False
    buf: list[str] = []

    for line in text.split("\n"):
        heading = re.match(r"^## (\d\d) · ", line)
        if heading:
            screen = heading.group(1)
            continue
        if line.strip() == "**Colours**":
            in_colours = True
            continue
        if not in_colours:
            continue
        if line.startswith("```"):
            if not in_fence:
                in_fence, buf = True, []
            else:
                in_fence = False
                in_colours = False
                if screen is not None:
                    yield screen, buf
            continue
        if in_fence:
            buf.append(line)


def content_checks(text: str) -> list[str]:
    """Verify non-grid content a raw-grid regeneration would strip.

    These are positive controls paired with the absence checks: each one names
    the literal text it guards and fails with a concrete message when that text
    is missing.
    """
    problems: list[str] = []

    # The format-example header must introduce every colour map.
    format_header = "<region>            <the literal characters it covers>    <token>  <256>  <hex>"
    if format_header not in text:
        problems.append("colour-map format-example header is missing")

    # Screen 00 is a component: it must carry the wordmark and ASCII fallback.
    if '"█▄▀ █ █▀█ █▀▀ █▀▀ █ █"' not in text:
        problems.append("screen 00 wordmark row 1 is missing")
    if "K I R S C H" not in text:
        problems.append("screen 00 ASCII fallback is missing")

    # Screen 05 records the collapsed grant list.
    if "▸ apply_patch 2 files changed · ✓ approved" not in text:
        problems.append("screen 05 collapsed apply_patch grant is missing")
    if "▸ run_command go test ./... · 2.4s · ✓ approved · session grant: go test" not in text:
        problems.append("screen 05 collapsed run_command grant is missing")

    # NOTE annotations live inside colour-map blocks.
    note_count = sum(1 for line in text.split("\n") if line.startswith("NOTE"))
    if note_count < 4:
        problems.append(f"only {note_count} NOTE annotation(s) in colour maps, want at least 4")

    # Every screen (00 through 12) must have a non-empty colour map.
    blocks = dict(colour_blocks(text))
    for screen in [f"{i:02d}" for i in range(13)]:
        if screen not in blocks:
            problems.append(f"screen {screen}: no **Colours** block found")
        elif not any(line.strip() for line in blocks[screen]):
            problems.append(f"screen {screen}: **Colours** block is empty")

    return problems


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

        glyphs = bar_glyphs(rows, want_w)
        # Each row with its scrollbar glyph split off. A row that has one is
        # padded out to the glyph, so the padding is trimmed too: the widest-row
        # rule is about where content ends, not where the bar sits.
        split = [split_bar(r, want_w, glyphs) for r in rows]
        bodies = [body for body, _ in split]
        has_bar = [glyph != "" for _, glyph in split]
        # The bar is one vertical run, and its thumb is one run inside it.
        if runs(has_bar) > 1:
            print(f"{where}: scrollbar glyphs are not one contiguous run of rows")
            failures += 1
        thumb = glyphs[1]
        in_run = [g for g in (gl for _, gl in split) if g]
        if runs([g == thumb for g in in_run]) > 1:
            print(f"{where}: the scrollbar thumb is not one contiguous run")
            failures += 1
        widest = max(
            (cell_width(b.rstrip() if bar else b) for b, bar in zip(bodies, has_bar)),
            default=0,
        )
        if widest != want_widest:
            print(f"{where}: widest row is {widest} cols, want {want_widest} — "
                  f"{want_w} columns less the {pad}-column right margin")
            failures += 1

        for i, (row, body) in enumerate(zip(rows, bodies), 1):
            w = cell_width(body)
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
        for row in bodies:
            cols = [cell_width(row[:i]) for i, ch in enumerate(row) if ch in BOX]
            if cols:
                right_edges.add(cols[-1])
        if len(right_edges) > 2:
            print(f"{where}: ragged box borders — right edges at {sorted(right_edges)}")
            failures += 1

    content_problems = content_checks(text)
    for msg in content_problems:
        print(f"content: {msg}")
    failures += len(content_problems)

    if failures:
        print(f"\n{failures} problem(s) in {DOC.name}")
        return 1
    print(f"{DOC.name}: grids OK, content OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
