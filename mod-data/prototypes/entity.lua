-- The visible part: a 1x1 tile that clicks together with its neighbours.
--
-- `simple-entity-with-force` mirrors the incumbent, and the choice is not
-- cosmetic. The part must be force-owned (clusters are per force), must not be
-- an active entity (it costs nothing per tick, which is the whole point of the
-- architecture), and must occupy a tile the way a belt does so that a player
-- cannot put a belt through one.
--
-- THE COLLISION MASK IS THE LOAD-BEARING LINE. Verified against the 2.0.77
-- prototype API and base's own `prototypes/collision-layers.lua`: all five
-- layer names below exist in 2.0.77.
--
--   floor, meltable, object, water_tile   what a normal building collides with
--   transport_belt                        what makes a belt refuse to overlap it
--
-- Without `transport_belt` a player can lay a belt straight through the
-- balancer, and the belt-orientation-decides-I/O rule stops making sense.

data:extend {
  {
    type = "simple-entity-with-force",
    name = "bbb-balancer-part",
    icon = "__better-belt-balancer__/graphics/icons/balancer-part.png",
    icon_size = 64,

    -- `get-by-unit-number` is what lets the runtime resolve a part from a
    -- stored unit number instead of from a cached LuaEntity. Caching entity
    -- references is the dominant crash class in every fork of the incumbent;
    -- this flag is the alternative, and it has to be declared at data stage or
    -- it is not available later.
    flags = { "placeable-neutral", "player-creation", "get-by-unit-number" },

    minable = { mining_time = 0.1, result = "bbb-balancer-part" },
    placeable_by = { item = "bbb-balancer-part", count = 1 },

    -- FAST REPLACE, and it is the one line that makes a part behave like the
    -- machine it is. `"transport-belt"` is base's own group for every belt,
    -- underground, splitter and lane splitter, so holding a part over a belt
    -- replaces it the way holding a splitter over one does: the belt and
    -- whatever it was carrying go to the player, the part takes the tile.
    --
    -- IT DOES NOT WEAKEN THE COLLISION MASK ABOVE and must not be read as
    -- doing so. Fast replace is an exception the engine makes for the entity
    -- being REPLACED and for nothing else, so a belt still cannot be laid
    -- THROUGH a balancer -- it can only be laid ON one, one tile at a time,
    -- which is the reverse gesture and is handled in guest/go/fastreplace.go.
    --
    -- The group is symmetric by construction, so this also buys the reverse:
    -- a belt held over a part replaces the part. Only a part with NO EDGE
    -- INTERFACE on its tile can be replaced that way -- `bbb-linked-belt` is a
    -- belt-connectable of its own and blocks the placement, measured -- which
    -- means the gesture reaches interior parts and is refused on edges.
    fast_replaceable_group = "transport-belt",
    max_health = 170,
    corpse = "splitter-remnants",
    resistances = { { type = "fire", percent = 60 } },

    collision_box = { { -0.35, -0.35 }, { 0.35, 0.35 } },
    selection_box = { { -0.5, -0.5 }, { 0.5, 0.5 } },
    collision_mask = {
      layers = {
        floor = true,
        meltable = true,
        object = true,
        transport_belt = true,
        water_tile = true,
      },
    },

    -- M5: THE ADAPTIVE SPRITE. `pictures` is a variation set and the runtime
    -- picks one per entity through `LuaEntity.graphics_variation`, which
    -- `simple-entity-with-force` inherits from `simple-entity-with-owner`. The
    -- guest sets it whenever a cluster's SHAPE changes (guest/go/skin.go), so a
    -- balancer of any shape draws as one continuous structure with trim only
    -- along its real outline. Zero per-tick cost, no extra entities, no
    -- rendering objects, nothing stored per part but one byte.
    --
    -- 47 cells, laid out 8 per row, in the order `tools/make-graphics.py` and
    -- `guest/go/skin` both enumerate: canonical neighbour masks ascending. 64 px
    -- at scale 0.5 is exactly one 32 px tile. `pictures` takes priority over
    -- `picture` and `animations`, so this is the only art the entity has.
    pictures = {
      sheet = {
        filename = "__better-belt-balancer__/graphics/entity/balancer-part-variants.png",
        priority = "high",
        width = 64,
        height = 64,
        variation_count = 47,
        line_length = 8,
        scale = 0.5,
      },
    },

    -- The engine would otherwise pick a RANDOM cell for a part the moment it is
    -- created, which would be the wrong shape for one tick (the guest sets the
    -- right one on the next) and would flicker on every blueprint paste. Cell 1
    -- is the lone part -- the correct picture for a part with no neighbours,
    -- which is what a freshly placed one usually is.
    random_variation_on_create = false,
  },
}
