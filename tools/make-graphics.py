#!/usr/bin/env python3
"""Generate the adaptive balancer-part sprite sheet and the item icon.

M5's whole visual claim is that a cluster of any shape reads as ONE structure.
That is done with a 47-variant "blob" tile set: every part is told which of its
eight neighbours are part of the same balancer, and picks the cell that carries
exactly the right piece of the outline. Connected sides have no border at all --
the plating, the plate seams and the glowing conduits run straight across the
tile boundary into the neighbour -- and the trim appears only where the blob
actually ends.

    python3 tools/make-graphics.py                    # or: make graphics
    python3 tools/make-graphics.py --priority          # or: make graphics-priority
    python3 tools/make-graphics.py --preview /tmp/look.png [--half]

Outputs (committed, so `make mod` works without Python):

    mod-data/graphics/entity/balancer-part-variants.png   512x768, 8x12 cells of 64
    mod-data/graphics/icons/balancer-part.png             64x64
    mod-data/graphics/entity/io-arrows.png                256x32, 8 cells of 32

THE SHEET IS 94 CELLS AND ONLY THE FIRST 47 ARE SHAPES. Cell 47+i is cell i with
a priority badge composited onto it, because the guest encodes a part's PRIORITY
flag in the same `graphics_variation` the shape lives in: variation = shape for
an ordinary part and shape + 47 for a priority one (guest/go/skin). One byte in
the entity carries both, the engine persists it, and a blueprint carries it --
which is the whole reason the flag is stored there and not only in the guest
heap, which every mod update throws away.

`--priority` is the mode that builds the second half, and it is the ONLY mode
that may be run against a sheet an artist drew: it READS the committed sheet's
first 47 cells and writes them back unchanged, so running it on a 94-cell sheet
is idempotent and running it on an artist's 47-cell sheet badges their work
rather than replacing it. The default mode below regenerates the placeholder
art from scratch and would overwrite an artist's cells, which is what it has
always done.

`--preview` writes nothing into the mod: it assembles the five named shapes into
one image the way the game would, which is the only honest way to judge whether
a change to the palette or the trim still looks fused. `--half` renders it at
the size `scale = 0.5` actually puts on screen. Factorio cannot take a
screenshot headlessly (`game.take_screenshot` writes nothing in `--benchmark`;
measured), so this is the tuning loop.

THE 47 AND ITS ORDER ARE A CONTRACT WITH `guest/go/skin`. Both sides enumerate
masks 0..255 ascending, keep the CANONICAL ones, and number them from 1. The
canonical rule is that a diagonal bit is meaningful only when both of the sides
touching it are set -- if a side is missing, that side's own trim already draws
the corner, so the diagonal cannot change the picture. 47 masks survive. Change
the rule here and `skin.Variation` disagrees with the sheet; the anchors at the
bottom of this file and `skin_test.go` assert the same three numbers so that a
one-sided change fails on both sides.

    bits: N=1 E=2 S=4 W=8   NE=16 SE=32 SW=64 NW=128
          (N is -y, E is +x, S is +y, W is -x; y grows downwards)

HOW THE PICTURE IS ACTUALLY DRAWN, because it is what makes 47 cheap: nothing
here draws "an edge piece" or "a corner piece". The nine cells of the local
neighbourhood are turned into a signed distance field -- for each pixel, the
distance to the nearest EMPTY cell -- and the trim is a band at a fixed distance
from that boundary. Every one of the 47 cells falls out of the same six lines.
Concave corners come out rounded (the distance to an empty cell's corner is
radial), which is exactly the fillet that makes a blob look fused; convex
corners come out sharp, which is what an armour plate should look like.

NOTHING FROM THE BASE GAME OR ANY THIRD-PARTY MOD IS COPIED INTO THIS REPO --
every pixel below is computed here, from numbers in this file. Written against
zlib and struct rather than Pillow, which is not installed on this machine and
is not worth a dependency.

FOR A FUTURE ARTIST: replace the PNGs, keep the cell order. The only contract is
"cell i is the shape whose canonical mask is the i-th ascending one, 64x64,
laid out 8 per row", and cell 47+i is that same shape MARKED. `variant_masks()`
prints the list. Draw the first 47 and run `--priority` to get the rest, or draw
all 94 by hand -- the badge below is a computed triangle and is the crudest
thing in this file. What it has to survive is `scale = 0.5`, which puts a 64 px
cell on screen as 32: whatever replaces it has to be readable at about eleven
pixels across, against plating that is already blue-grey and already lit.
"""

import math
import os
import struct
import sys
import zlib

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GFX = os.path.join(ROOT, "mod-data", "graphics")

CELL = 64          # px per cell; scale 0.5 in the prototype makes that one tile
COLS = 8           # cells per row in the sheet -- `line_length` in the prototype

# --- the theme ---------------------------------------------------------------
#
# One palette, and everything below is a lerp between two of these. A re-theme
# is this block plus the two glow colours.

OUTLINE   = (11, 14, 20)      # the hard line at the very edge of the blob
TRIM_HI   = (96, 152, 198)    # outer edge of the trim band
TRIM_LO   = (40, 68, 98)      # where the trim meets the body
BODY_HI   = (64, 76, 97)      # plating, lit
BODY_LO   = (34, 41, 54)      # plating, shaded
SEAM      = (26, 32, 43)      # the line between two plates
RIVET     = (88, 102, 128)
HUB_RING  = (86, 104, 132)
HUB_BODY  = (50, 62, 82)
CORE      = (146, 212, 248)   # the lit core in the middle of every part
VEIN      = (70, 124, 168)    # the conduit running to each connected side
VEIN_HI   = (128, 198, 240)

OUTLINE_D = 1.6    # px: how deep the hard outline is
TRIM_D    = 6.0    # px: how deep the trim band is
AO_D      = 11.0   # px: how far the inward shading from the trim reaches

# Fake lighting: which way an edge faces changes how bright its trim is. The
# key is the direction from the pixel to the empty cell that is nearest to it.
FACE_LIGHT = {(0, -1): 1.22, (0, 1): 0.80, (-1, 0): 1.08, (1, 0): 0.92}

# --- the 47 ------------------------------------------------------------------

N, E, S, W = 1, 2, 4, 8
NE, SE, SW, NW = 16, 32, 64, 128

# corner bit -> the two side bits that make it meaningful, and its (dx, dy)
CORNERS = (
    (NE, N, E, 1, -1),
    (SE, S, E, 1, 1),
    (SW, S, W, -1, 1),
    (NW, N, W, -1, -1),
)
SIDES = ((N, 0, -1), (E, 1, 0), (S, 0, 1), (W, -1, 0))


def canon(mask):
    """Drop the diagonal bits that cannot change the picture."""
    m = mask & 0x0F
    for bit, a, b, _, _ in CORNERS:
        if mask & bit and mask & a and mask & b:
            m |= bit
    return m


def variant_masks():
    """The canonical masks, ascending. Index i is variation i+1."""
    return [m for m in range(256) if canon(m) == m]


# --- pixels ------------------------------------------------------------------


def lerp(a, b, t):
    t = 0.0 if t < 0 else (1.0 if t > 1 else t)
    return tuple(int(round(a[i] + (b[i] - a[i]) * t)) for i in range(3))


def scale(c, f):
    return tuple(min(255, max(0, int(round(v * f)))) for v in c)


def dist_to_cell(px, py, cx, cy):
    """Distance from a pixel to the square of cell (cx, cy), in px."""
    x0, y0 = cx * CELL, cy * CELL
    dx = max(x0 - px, 0.0, px - (x0 + CELL))
    dy = max(y0 - py, 0.0, py - (y0 + CELL))
    return math.hypot(dx, dy)


def neighbourhood(mask):
    """The empty cells of the 3x3 around this one, as (dx, dy)."""
    filled = {(0, 0)}
    for bit, dx, dy in SIDES:
        if mask & bit:
            filled.add((dx, dy))
    for bit, _, _, dx, dy in CORNERS:
        if mask & bit:
            filled.add((dx, dy))
    return [(dx, dy) for dy in (-1, 0, 1) for dx in (-1, 0, 1)
            if (dx, dy) not in filled]


def body_pixel(x, y):
    """The plating: a vertical gradient, plate seams, and a rivet per plate.

    Every feature is placed on a 16 px lattice offset by 8, so it lines up
    across a tile boundary: seams at 8, 24, 40, 56 are 16 apart in both
    directions including from 56 to the next tile's 8. A player looking at a
    2x2 block sees one continuous plated surface, not four squares.
    """
    c = lerp(BODY_HI, BODY_LO, y / float(CELL))
    # Brushed metal: a deterministic hash along the diagonal, +-3 levels. It is
    # below the threshold of "texture" and above the threshold of "flat".
    n = ((x * 7 + y * 13) * 2654435761) & 0xFFFFFFFF
    c = scale(c, 1.0 + (((n >> 13) & 7) - 3.5) * 0.012)
    if x % 16 == 8 or y % 16 == 8:
        c = lerp(c, SEAM, 0.8)
    elif x % 16 == 9 or y % 16 == 9:
        c = lerp(c, BODY_HI, 0.35)   # the lit lip on the far side of a seam
    # One rivet per plate, on a 32 px lattice so it tiles across a boundary.
    rx, ry = (x - 6) % 32, (y - 6) % 32
    if rx <= 2 and ry <= 2:
        c = RIVET if (rx + ry) <= 2 else lerp(RIVET, SEAM, 0.6)
    return c


def hub_pixel(x, y, mask):
    """The hub and its conduits, or None where they are not.

    The conduits are what carry the "these are one machine" reading: a vein
    leaves the hub towards every CONNECTED side and stops at the tile edge,
    where the neighbour's vein picks it up. Two parts side by side show one
    unbroken lit line between their two cores.
    """
    dx, dy = x - (CELL / 2.0 - 0.5), y - (CELL / 2.0 - 0.5)
    r = math.hypot(dx, dy)
    oct_r = max(abs(dx), abs(dy)) * 0.62 + (abs(dx) + abs(dy)) * 0.31

    if r <= 4.4:
        return lerp(CORE, VEIN_HI, r / 4.4)
    if r <= 5.6:
        return lerp(VEIN_HI, HUB_BODY, (r - 4.4) / 1.2)
    if oct_r <= 10.4:
        return lerp(HUB_BODY, HUB_RING, (oct_r - 5.6) / 4.8 * 0.45)
    if oct_r <= 12.6:
        return lerp(HUB_RING, OUTLINE, (oct_r - 10.4) / 2.2 * 0.55)

    for bit, sx, sy in SIDES:
        if not mask & bit:
            continue
        along = dx * sx + dy * sy
        perp = abs(dx * sy - dy * sx)
        if along < 9.0 or perp > 3.6:
            continue
        if perp <= 1.2:
            return lerp(VEIN_HI, VEIN, (along - 9.0) / 24.0 * 0.55)
        return lerp(VEIN, BODY_LO, (perp - 1.2) / 2.4)
    return None


def draw_cell(mask, showcase=False):
    """One 64x64 cell as a list of rows of (r, g, b, a)."""
    empties = neighbourhood(mask)
    rows = []
    for y in range(CELL):
        row = []
        py = y + 0.5
        for x in range(CELL):
            px = x + 0.5
            d, face = 1e9, None
            for cx, cy in empties:
                dd = dist_to_cell(px, py, cx, cy)
                if dd < d:
                    d, face = dd, (cx, cy)

            c = body_pixel(x, y)
            h = hub_pixel(x, y, mask if not showcase else 0x0F)
            if h is not None and d > TRIM_D:
                c = h
            if d < AO_D:
                # Inward shading from the edge: depth, and it is what stops a
                # long straight run of parts from reading as flat lino.
                c = lerp(c, SEAM, (1.0 - d / AO_D) * 0.35)
            if d < TRIM_D:
                t = lerp(TRIM_HI, TRIM_LO, (d - OUTLINE_D) / (TRIM_D - OUTLINE_D))
                c = scale(t, FACE_LIGHT.get(face, 1.0))
            if d < OUTLINE_D:
                c = OUTLINE
            row.append((c[0], c[1], c[2], 255))
        rows.append(row)
    return rows


# --- PNG ---------------------------------------------------------------------


def read_png(path):
    """The committed sheet, back as rows of (r, g, b, a).

    Written here rather than reached for from Pillow for the reason the writer
    gives: Pillow is not installed on this machine and is not worth a dependency
    for one file. 8-bit RGBA, non-interlaced, which is what every PNG in this
    repository is and what the writer below emits; anything else is refused by
    name rather than half-read.
    """
    d = open(path, "rb").read()
    if d[:8] != b"\x89PNG\r\n\x1a\n":
        raise SystemExit("%s is not a PNG" % path)
    idat, w, h, srgb = bytearray(), 0, 0, None
    i = 8
    while i < len(d):
        n = struct.unpack(">I", d[i:i + 4])[0]
        tag, body = d[i + 4:i + 8], d[i + 8:i + 8 + n]
        if tag == b"IHDR":
            w, h, depth, colour, comp, filt, inter = struct.unpack(">IIBBBBB", body)
            if (depth, colour, comp, filt, inter) != (8, 6, 0, 0, 0):
                raise SystemExit(
                    "%s is depth %d colour %d interlace %d; this reader does "
                    "8-bit RGBA non-interlaced only" % (path, depth, colour, inter))
        elif tag == b"IDAT":
            idat += body
        elif tag == b"sRGB":
            srgb = body
        i += 12 + n

    raw = zlib.decompress(bytes(idat))
    stride = w * 4
    rows, prev = [], bytes(stride)
    at = 0
    for _ in range(h):
        ft = raw[at]
        line = bytearray(raw[at + 1:at + 1 + stride])
        at += 1 + stride
        # The five PNG filters. An artist's exporter picks per scanline and this
        # has to undo whatever it picked, so all five are here even though the
        # writer below only ever emits None.
        if ft == 1:
            for j in range(4, stride):
                line[j] = (line[j] + line[j - 4]) & 0xFF
        elif ft == 2:
            for j in range(stride):
                line[j] = (line[j] + prev[j]) & 0xFF
        elif ft == 3:
            for j in range(stride):
                a = line[j - 4] if j >= 4 else 0
                line[j] = (line[j] + ((a + prev[j]) >> 1)) & 0xFF
        elif ft == 4:
            for j in range(stride):
                a = line[j - 4] if j >= 4 else 0
                c = prev[j - 4] if j >= 4 else 0
                b = prev[j]
                pa, pb, pc = abs(b - c), abs(a - c), abs(a + b - 2 * c)
                pr = a if (pa <= pb and pa <= pc) else (b if pb <= pc else c)
                line[j] = (line[j] + pr) & 0xFF
        elif ft != 0:
            raise SystemExit("%s: unknown PNG filter %d" % (path, ft))
        prev = bytes(line)
        rows.append([tuple(line[j:j + 4]) for j in range(0, stride, 4)])
    return rows, srgb


def write_png(path, rows, srgb=None):
    w, h = len(rows[0]), len(rows)
    raw = bytearray()
    for row in rows:
        raw.append(0)  # filter type 0 (None) for every scanline
        for r, g, b, a in row:
            raw += bytes((r, g, b, a))

    def chunk(tag, data):
        return (struct.pack(">I", len(data)) + tag + data +
                struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF))

    png = b"\x89PNG\r\n\x1a\n"
    png += chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0))
    if srgb is not None:
        # Carried through rather than dropped, so that a sheet that has been
        # through --priority differs from the artist's in its pixels and in
        # nothing else a viewer can see.
        png += chunk(b"sRGB", srgb)
    png += chunk(b"IDAT", zlib.compress(bytes(raw), 9))
    png += chunk(b"IEND", b"")

    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(png)
    print("wrote %s (%dx%d, %d bytes)"
          % (os.path.relpath(path, ROOT), w, h, len(png)))


# --- the I/O arrows ----------------------------------------------------------
#
# Eight 32 px cells, `in` then `out`, N E S W within each. The arrow always
# points the way the ITEMS go -- into the balancer on an input edge, out of it
# on an output edge -- and the colour says which of the two it is without having
# to work out which edge of which part you are looking at.
#
# The cell says which SIDE the belt is on; the prototype (guest/go/data/
# sprite.go) carries the shift that puts the arrow on that side of the tile, so
# the guest names a sprite and passes nothing but a target.

ARROW = 32
ARROW_IN = (120, 226, 148)
ARROW_OUT = (250, 186, 84)
ARROW_EDGE = (10, 14, 20)


def draw_arrow(kind, side):
    """One arrow cell. `side` is the edge the belt is on, 0..3 = N E S W."""
    point = side if kind else (side + 2) % 4   # out points away, in points in
    body = ARROW_OUT if kind else ARROW_IN
    rows = []
    for y in range(ARROW):
        row = []
        for x in range(ARROW):
            # Rotate the pixel into a frame where the arrow points north, so
            # the glyph below is written once. Each step is a quarter-turn
            # CLOCKWISE on screen (y grows downwards), so `point` walks
            # N -> E -> S -> W, matching the compass order of `side`.
            u, v = x - 15.5, y - 15.5
            for _ in range(point):
                u, v = v, -u
            # Two chevrons, one behind the other: `a` is constant along the arms
            # of a V whose apex is at v = -a, and 0.8 turns it into a real
            # distance for a slope of 0.75. The arms are cut at |u| = 10, which
            # keeps the whole glyph and its outline inside the 32 px cell in
            # every one of the four rotations.
            a = abs(u) * 0.75 - v
            near = min(abs(a + 6.0), abs(a - 0.5)) * 0.8 if abs(u) <= 10.0 else 99.0
            if near < 2.2:
                row.append(body + (255,))
            elif near < 3.5:
                row.append(ARROW_EDGE + (255,))
            else:
                row.append((0, 0, 0, 0))
        rows.append(row)
    return rows


def arrow_strip():
    strip = [[(0, 0, 0, 0)] * (8 * ARROW) for _ in range(ARROW)]
    for kind in (0, 1):
        for side in range(4):
            cell = draw_arrow(kind, side)
            ox = (kind * 4 + side) * ARROW
            for y in range(ARROW):
                strip[y][ox:ox + ARROW] = cell[y]
    return strip


# --- looking at it ------------------------------------------------------------


# --- the priority badge ------------------------------------------------------
#
# Cell 47+i is cell i with this drawn over its hub: an up-pointing triangle in
# the same amber the OUTPUT arrow uses, with a dark outline so it survives being
# on top of lit plating. It says "this port is served first" and it says nothing
# about a direction, which is why it is not a chevron -- the I/O arrows are
# chevrons and a second chevron on the same machine would read as one.
#
# It is CENTRED rather than tucked into a corner, and that is the one thing about
# it that is not arbitrary: the plating, the seams and the conduits all run
# across a tile boundary on purpose (that is the whole 47-cell trick), so a badge
# near an edge lands on the seam between two parts and reads as belonging to
# neither. The hub is the one place in the cell that is the part's own.
#
# Coverage is supersampled 4x4 and composited into the RGB rather than into the
# alpha: every one of the 47 cells is opaque, and a badge that introduced partial
# alpha would put a hole in the machine.

PRIO_FILL = ARROW_OUT          # the amber an output arrow already uses
PRIO_EDGE = (10, 14, 20)

# Apex and base half-width of the two triangles, in px from the cell centre. The
# outer one is the outline; the difference between them is its thickness.
PRIO_OUTER = (12.2, 9.8, 13.4)   # up, down, half-width
PRIO_INNER = (9.6, 7.6, 10.6)
PRIO_SS = 4                      # supersamples per axis


def in_triangle(px, py, tri):
    """Is (px, py) inside the up-pointing triangle, centre-relative."""
    up, down, half = tri
    if py < -up or py > down:
        return False
    # Linear taper from the apex to the base.
    t = (py + up) / (up + down)
    return abs(px) <= half * t


def prio_coverage(x, y, tri):
    """How much of pixel (x, y) the triangle covers, 0..1."""
    n = 0
    for sy in range(PRIO_SS):
        py = y + (sy + 0.5) / PRIO_SS - (CELL / 2.0)
        for sx in range(PRIO_SS):
            px = x + (sx + 0.5) / PRIO_SS - (CELL / 2.0)
            if in_triangle(px, py, tri):
                n += 1
    return n / float(PRIO_SS * PRIO_SS)


def badge_cell(cell):
    """One cell with the badge over it. The input is not modified."""
    out = []
    for y in range(CELL):
        row = []
        for x in range(CELL):
            r, g, b, a = cell[y][x]
            co = prio_coverage(x, y, PRIO_OUTER)
            if co <= 0.0:
                row.append((r, g, b, a))
                continue
            c = lerp((r, g, b), PRIO_EDGE, co)
            ci = prio_coverage(x, y, PRIO_INNER)
            if ci > 0.0:
                c = lerp(c, PRIO_FILL, ci)
            row.append((c[0], c[1], c[2], a))
        out.append(row)
    return out


def priority_sheet():
    """Rewrite the committed sheet as 94 cells: 47 shapes, then 47 badged.

    IDEMPOTENT, and that is what makes it safe to run against whatever is
    committed: it takes the FIRST 47 cells whether the file it reads holds 47 or
    94, so a second run re-badges the same unbadged source and produces the same
    bytes. Cells 1..47 come back pixel for pixel as they went in, which is the
    property that matters when they are an artist's and not this file's.
    """
    path = os.path.join(GFX, "entity", "balancer-part-variants.png")
    src, srgb = read_png(path)
    have = (len(src[0]) // CELL) * (len(src) // CELL)
    if len(src[0]) != COLS * CELL or have < len(variant_masks()):
        raise SystemExit(
            "%s is %dx%d (%d cells of %d); --priority wants %d columns and at "
            "least %d cells" % (path, len(src[0]), len(src), have, CELL, COLS,
                                len(variant_masks())))

    n = len(variant_masks())
    lines = (2 * n + COLS - 1) // COLS
    sheet = [[(0, 0, 0, 0)] * (COLS * CELL) for _ in range(lines * CELL)]
    for i in range(n):
        sx, sy = (i % COLS) * CELL, (i // COLS) * CELL
        cell = [src[sy + y][sx:sx + CELL] for y in range(CELL)]
        for half, art in ((0, cell), (n, badge_cell(cell))):
            ox, oy = ((i + half) % COLS) * CELL, ((i + half) // COLS) * CELL
            for y in range(CELL):
                sheet[oy + y][ox:ox + CELL] = art[y]
    write_png(path, sheet, srgb)
    print("%d shapes, %d cells, %d per row" % (n, 2 * n, COLS))


def preview(path, half=False):
    """Assemble the five named shapes into one image, as the game would.

    THE POINT OF THIS IS THE TUNING LOOP. A cell on its own says nothing about
    whether a blob looks fused; a 2x2 next to a donut says everything, and
    launching Factorio to find out is a two-minute round trip. `half=True`
    halves it to what `scale = 0.5` puts on screen at default zoom, which is the
    size the art actually has to work at.

        python3 tools/make-graphics.py --preview /tmp/look.png
    """
    shapes = [
        [(0, 0), (1, 0), (2, 0), (3, 0)],
        [(0, 0), (0, 1), (0, 2), (1, 2), (2, 2)],
        [(1, 0), (0, 1), (1, 1), (2, 1), (1, 2)],
        [(0, 0), (1, 0), (0, 1), (1, 1)],
        [(x, y) for y in range(4) for x in range(4)
         if not (1 <= x <= 2 and 1 <= y <= 2)],
        [(0, 0), (1, 0), (2, 0), (1, 1), (2, 1), (3, 1), (4, 1),
         (1, 2), (2, 2), (0, 2)],
    ]
    cells, ox = {}, 0
    for tiles in shapes:
        got = set(tiles)
        for x, y in tiles:
            m = 0
            for bit, dx, dy in SIDES:
                if (x + dx, y + dy) in got:
                    m |= bit
            for bit, _, _, dx, dy in CORNERS:
                if (x + dx, y + dy) in got:
                    m |= bit
            cells[(x + ox, y)] = canon(m)
        ox += max(x for x, _ in tiles) + 3

    w, h = ox, max(y for _, y in cells) + 1
    img = [[(20, 22, 26, 255)] * (w * CELL) for _ in range(h * CELL)]
    drawn = {}
    for (x, y), m in cells.items():
        if m not in drawn:
            drawn[m] = draw_cell(m)
        for j in range(CELL):
            img[y * CELL + j][x * CELL:(x + 1) * CELL] = drawn[m][j]
    if half:
        img = [[img[2 * j][2 * i] for i in range(w * CELL // 2)]
               for j in range(h * CELL // 2)]
    write_png(path, img)


def main():
    masks = variant_masks()
    # The anchors skin_test.go asserts. A change to the enumeration that is not
    # made on both sides fails here and there.
    assert len(masks) == 47, len(masks)
    assert masks[0] == 0 and masks[-1] == 255
    assert masks.index(N | E | S | W) + 1 == 16, masks.index(N | E | S | W)

    lines = (len(masks) + COLS - 1) // COLS
    sheet = [[(0, 0, 0, 0)] * (COLS * CELL) for _ in range(lines * CELL)]
    for i, mask in enumerate(masks):
        cell = draw_cell(mask)
        ox, oy = (i % COLS) * CELL, (i // COLS) * CELL
        for y in range(CELL):
            sheet[oy + y][ox:ox + CELL] = cell[y]
    write_png(os.path.join(GFX, "entity", "balancer-part-variants.png"), sheet)
    # ...and then the priority half, over what was just written. The prototype
    # declares 94 cells (guest/go/data/entity.go, skin.Cells), so a default run
    # that stopped at 47 would leave a sheet the engine refuses to load.
    priority_sheet()

    # The icon is the lone part with its conduits lit anyway, so that an item in
    # an inventory says "this connects" without a neighbour to connect to.
    write_png(os.path.join(GFX, "icons", "balancer-part.png"),
              draw_cell(0, showcase=True))

    write_png(os.path.join(GFX, "entity", "io-arrows.png"), arrow_strip())

    print("%d variants, %d per row" % (len(masks), COLS))


if __name__ == "__main__":
    if "--preview" in sys.argv:
        i = sys.argv.index("--preview")
        preview(sys.argv[i + 1], half="--half" in sys.argv)
    elif "--priority" in sys.argv:
        priority_sheet()
    else:
        main()
