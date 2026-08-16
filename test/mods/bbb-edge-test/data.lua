-- The same harness loader the M2, M3 and marathon suites use: base's
-- `loader-1x1` is a hidden 0.03125 prototype, far too slow to saturate
-- anything.
local util = require("util")

local EXPRESS = 0.09375

local function loader(name)
  local p = util.table.deepcopy(data.raw["loader-1x1"]["loader-1x1"])
  p.name = name
  p.speed = EXPRESS
  p.minable = nil
  p.next_upgrade = nil
  p.fast_replaceable_group = nil
  return p
end

data:extend { loader("bbbt-loader") }
