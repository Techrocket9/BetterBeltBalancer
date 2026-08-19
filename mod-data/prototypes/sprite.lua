-- The I/O arrows: which sides of a balancer take items in and which give them
-- out, drawn on the edge tiles when the player holds Alt.
--
-- Eight `sprite` prototypes over one 256x32 strip, named `bbb-arrow-<in|out>-
-- <n|e|s|w>` where the letter is the SIDE THE BELT IS ON. The arrow itself
-- points the way the items go -- inwards on an input edge, outwards on an
-- output -- so `in-n` is a green chevron pointing SOUTH, sitting on the north
-- edge of its tile.
--
-- THE SHIFT AND THE ROTATION ARE BOTH BAKED IN, and that is the whole reason
-- there are eight prototypes rather than one drawn with an `orientation` and an
-- offset. The guest names one of these and passes a target and nothing else:
-- no orientation to compute from a `defines.direction` value whose number this
-- mod deliberately never writes down, and no offset table to marshal on every
-- one of the eight-or-more draws a compile makes.
--
-- The shift puts the arrow 0.3 tiles towards its edge, which also keeps the two
-- arrows apart on a tile that carries two edges -- legal, and normal on a
-- corner. It is applied as a per-family distance rather than one number; see
-- ART_BIAS below for why.
--
-- Rendering objects with an ENTITY target are destroyed by the engine when that
-- entity is (verified in the 2.0.77 runtime doc, `ScriptRenderTarget`), and the
-- entity these are drawn on is the visible linked belt the compiler places for
-- the edge. So a teardown removes them for free and the guest stores no
-- rendering ids at all. See guest/go/compile.go, drawArrow.

local STRIP = "__better-belt-balancer__/graphics/entity/io-arrows.png"
-- Unit direction per side; the DISTANCE is per family, just below.
local SIDES = {
  { "n", 0, -1 },
  { "e", 1, 0 },
  { "s", 0, 1 },
  { "w", -1, 0 },
}

-- THE ART IS NOT CENTRED IN ITS CELL, so the shift cannot be one number.
-- Each chevron is drawn flush against its TAIL edge, which puts the glyph's
-- centroid 0.104 tiles BEHIND its tip. An input points inwards, so that bias
-- pushes it further OUT and ADDS to the shift; an output points outwards, so
-- the same bias pulls it IN and SUBTRACTS. Uncompensated the two families land
-- 0.404 and 0.196 tiles from the tile centre: an output stops reading as an
-- edge marker and sits on the machine's own hub instead. Measured, both on the
-- generated placeholder and on the 2026-08-19 artist delivery -- the centroid
-- offset is +-6.6 px of a 32 px cell in BOTH, so this is a property of the
-- convention rather than of one sheet.
--
-- Compensating per family puts both at 0.3, which is what the shift always
-- meant. If the art is ever redrawn centred (Artboard C of the art spec asks
-- for exactly that), set ART_BIAS to 0 -- do not edit the two distances.
local ART_BIAS = 0.104
local DIST = { [0] = 0.3 - ART_BIAS, [1] = 0.3 + ART_BIAS }

local arrows = {}
for kind = 0, 1 do
  for i, side in ipairs(SIDES) do
    local d = DIST[kind]
    arrows[#arrows + 1] = {
      type = "sprite",
      name = "bbb-arrow-" .. (kind == 0 and "in" or "out") .. "-" .. side[1],
      filename = STRIP,
      priority = "extra-high-no-scale",
      width = 32,
      height = 32,
      x = (kind * 4 + i - 1) * 32,
      -- 32 px at scale 0.5 is half a tile: big enough to read at default zoom,
      -- small enough that two of them on one tile do not collide.
      scale = 0.5,
      shift = { side[2] * d, side[3] * d },
      flags = { "no-crop" },
    }
  end
end

data:extend(arrows)
