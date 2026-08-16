-- ONE recipe and ONE technology for M1.
--
-- The incumbent ships three tiers whose only difference is the recipe cost --
-- the part behaves identically at every tier. Copying that now would be
-- copying a wishlist item the forum record says was never delivered
-- (meaningful tier differences), so M1 ships the single tier and the question
-- stays open.
--
-- Cheap on purpose: this is infrastructure, not a reward. Iron plates, gears
-- and belts, all available the moment `logistics` is researched.
data:extend {
  {
    type = "recipe",
    name = "bbb-balancer-part",
    enabled = false,
    energy_required = 1,
    ingredients = {
      { type = "item", name = "iron-plate",     amount = 4 },
      { type = "item", name = "iron-gear-wheel", amount = 2 },
      { type = "item", name = "transport-belt",  amount = 2 },
    },
    results = {
      { type = "item", name = "bbb-balancer-part", amount = 1 },
    },
    order = "c[splitter]-y[bbb-balancer]",
  },
}
