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
-- The shift is 0.3 tiles towards the edge, which also keeps the two arrows
-- apart on a tile that carries two edges -- legal, and normal on a corner.
--
-- Rendering objects with an ENTITY target are destroyed by the engine when that
-- entity is (verified in the 2.0.77 runtime doc, `ScriptRenderTarget`), and the
-- entity these are drawn on is the visible linked belt the compiler places for
-- the edge. So a teardown removes them for free and the guest stores no
-- rendering ids at all. See guest/go/compile.go, drawArrow.

local STRIP = "__better-belt-balancer__/graphics/entity/io-arrows.png"
local SIDES = {
  { "n", 0, -0.3 },
  { "e", 0.3, 0 },
  { "s", 0, 0.3 },
  { "w", -0.3, 0 },
}

local arrows = {}
for kind = 0, 1 do
  for i, side in ipairs(SIDES) do
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
      shift = { side[2], side[3] },
      flags = { "no-crop" },
    }
  end
end

data:extend(arrows)
