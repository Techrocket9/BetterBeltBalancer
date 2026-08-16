-- The harness loader, cloned exactly as the headless suites clone it: base's
-- loader-1x1 is a hidden 0.03125 prototype and a source that slow could not
-- fill anything. Nothing here ships; this mod exists to be enabled for ten
-- minutes and disabled again.
local util = require("util")

local EXPRESS = 0.09375

local p = util.table.deepcopy(data.raw["loader-1x1"]["loader-1x1"])
p.name = "bbbi-loader"
p.speed = EXPRESS
p.minable = nil
p.next_upgrade = nil
p.fast_replaceable_group = nil

data:extend { p }
