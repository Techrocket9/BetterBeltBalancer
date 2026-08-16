-- Test-rig prototypes. Nothing here ships; this is the harness's own kit.
--
-- The only thing needed is a loader that runs at express speed, because base's
-- `loader-1x1` is a hidden 0.03125 prototype and a source that slow could not
-- saturate anything. The rigs are otherwise built from stock express belts, so
-- what they measure is the rate a real belt delivers.
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
