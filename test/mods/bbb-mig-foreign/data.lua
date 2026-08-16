-- A STRANGER WHO OWNS THE NAME.
--
-- This mod is not Belt Balancer, not Belt Balancer 2, not Belt Balancer 3 and
-- not Belt Balancer Performance. It defines `balancer-part` anyway, exactly as
-- they do, and its entities must never be converted by this mod's migration.
--
-- WHAT IT PROVES, and it is the guard with the real blast radius. The runtime
-- decision cannot be taken from the prototype's existence, because a prototype
-- says nothing about whose it is; and it cannot be taken from
-- `script.active_mods` alone, because that is a list of four names and this mod
-- is not on it. It is taken from `bbb-legacy-stub`, a marker prototype the data
-- stage defines IF AND ONLY IF it also defined the stub -- which it does not
-- here, because this mod got there first.
--
-- The mig suite's third leg runs this mod alongside the real one, through the
-- same swap the other two legs use, and asserts that nothing at all happened.

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
}
