-- The harness's own kit, and there is one thing in it.
--
-- Base Factorio's only 1x1 loader is `loader-1x1`, a hidden 0.03125 prototype,
-- and a source that slow could not saturate anything. Every other piece in
-- these rigs is a stock express belt, splitter or chest, so what the suite
-- measures is the rate a real one delivers.
--
-- NOTHING HERE TOUCHES `balancer-part` OR ANY PROTOTYPE OF THIS MOD'S. This
-- file has to load in the phase where Better Belt Balancer is not installed at
-- all, which is the whole shape of the added-as-removed leg.

local util = require("util")

local EXPRESS = 0.09375

local loader = util.table.deepcopy(data.raw["loader-1x1"]["loader-1x1"])
loader.name = "bbbmig-loader"
loader.speed = EXPRESS
loader.minable = nil
loader.next_upgrade = nil
loader.fast_replaceable_group = nil

data:extend { loader }
