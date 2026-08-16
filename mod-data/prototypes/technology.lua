-- One technology, hanging off `logistics` -- the same place the incumbent's
-- first tier hangs, so a player who knows that mod finds this one where they
-- expect it.
--
-- The unit is COPIED FROM `logistics` rather than written out, so the mod
-- follows whatever the base game (or an overhaul mod) says that tier costs
-- instead of pinning a number that is wrong in every modpack.
local logistics = data.raw.technology["logistics"]

data:extend {
  {
    type = "technology",
    name = "bbb-balancer",
    icon = "__better-belt-balancer__/graphics/icons/balancer-part.png",
    icon_size = 64,
    effects = {
      { type = "unlock-recipe", recipe = "bbb-balancer-part" },
    },
    prerequisites = { "logistics" },
    unit = {
      count = logistics.unit.count,
      ingredients = logistics.unit.ingredients,
      time = logistics.unit.time,
    },
    order = "a-b-bbb",
  },
}
