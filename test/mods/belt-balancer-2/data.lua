-- A STAND-IN FOR BELT BALANCER 2, and only its data stage.
--
-- WHY A STAND-IN AND NOT THE REAL MOD. The suite has to run from a clean
-- checkout on any machine, and vendoring somebody else's mod into this
-- repository is a distribution question this project does not need to answer.
-- What the migration actually depends on is the PROTOTYPES -- a
-- `simple-entity-with-force` named `balancer-part`, an `item` of the same name
-- that places it, and a technology that dies with the mod -- and those are
-- reproduced here exactly, under the real mod's own name and version so that
-- `script.active_mods` and `mods[...]` see what they would really see. The real
-- Belt Balancer 2 was run through the same flow by hand once, and the numbers
-- are recorded in CLAUDE.md.
--
-- IT HAS NO CONTROL STAGE AT ALL, deliberately. The real mod's runtime moves
-- items through a Lua FIFO in its own `storage`, and none of that survives its
-- removal -- Factorio deletes a removed mod's `storage` with the mod. So a
-- stand-in that balanced would be modelling the one thing the migration cannot
-- recover. What matters is that the ENTITIES are standing and the BELTS around
-- them are full, and both of those are the world's, not the mod's.
--
-- NONE OF BELT BALANCER 2'S ART IS USED. The entity draws the engine's own
-- empty sprite; a headless run never opens a sprite file and the rigs are
-- measured, not looked at.

local util = require("util")

data:extend {
  {
    type = "simple-entity-with-force",
    name = "balancer-part",
    icon = "__base__/graphics/icons/splitter.png",
    icon_size = 64,
    flags = { "placeable-neutral", "player-creation" },
    minable = { mining_time = 0.1, result = "balancer-part" },
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
    picture = util.empty_sprite(),
  },

  {
    type = "item",
    name = "balancer-part",
    icon = "__base__/graphics/icons/splitter.png",
    icon_size = 64,
    subgroup = "belt",
    order = "c[splitter]-x[balancer]",
    place_result = "balancer-part",
    stack_size = 50,
  },

  {
    type = "recipe",
    name = "belt-balancer-normal-belt",
    enabled = false,
    energy_required = 3,
    ingredients = {
      { type = "item", name = "iron-gear-wheel", amount = 20 },
      { type = "item", name = "electronic-circuit", amount = 15 },
      { type = "item", name = "transport-belt", amount = 5 },
    },
    results = { { type = "item", name = "balancer-part", amount = 1 } },
    order = "g[balancer]-a[balancer]",
  },

  -- The technology is here because it is the one thing the migration has to
  -- REPLACE rather than preserve: it goes with the mod, and a player left
  -- holding fifty balancers and no way to craft a part would have been given a
  -- worse save than they started with.
  {
    type = "technology",
    name = "belt-balancer-1",
    icon = "__base__/graphics/icons/splitter.png",
    icon_size = 64,
    effects = { { type = "unlock-recipe", recipe = "belt-balancer-normal-belt" } },
    prerequisites = { "logistics" },
    unit = {
      count = data.raw.technology["logistics"].unit.count,
      ingredients = data.raw.technology["logistics"].unit.ingredients,
      time = data.raw.technology["logistics"].unit.time,
    },
  },
}
