-- The harness's own kit, and there is one thing in it -- the same one thing
-- every other suite's kit has, for the same reason.
--
-- Base Factorio's only 1x1 loader is `loader-1x1`, a hidden 0.03125 prototype,
-- and a source that slow could not saturate anything. Every other piece in
-- these rigs is a stock express belt or chest, so what the suite measures is
-- the rate a real one delivers.

local util = require("util")

local EXPRESS = 0.09375

local loader = util.table.deepcopy(data.raw["loader-1x1"]["loader-1x1"])
loader.name = "bbbs-loader"
loader.speed = EXPRESS
loader.minable = nil
loader.next_upgrade = nil
loader.fast_replaceable_group = nil

data:extend { loader }
