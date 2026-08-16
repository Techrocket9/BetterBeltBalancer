local util = require("util")
local p = util.table.deepcopy(data.raw["loader-1x1"]["loader-1x1"])
p.name = "bbbt-loader"
p.speed = 0.09375
p.minable = nil
p.next_upgrade = nil
p.fast_replaceable_group = nil

-- The STACKING source. `max_belt_stack_size` is refused at load without the
-- `space_travel` feature flag -- "Belt stacking is disabled and can not be
-- used" -- which is exactly why the stacking leg lives in this suite and not in
-- a base-only one. The flag's key is `space_travel`; the error message spells it
-- with a hyphen and the table does not.
--
-- The prototype is only half of it: a force whose `belt_stack_size_bonus` is 0
-- receives SINGLES from this loader. Both are needed, and control.lua sets the
-- other half.
local q = util.table.deepcopy(p)
q.name = "bbbt-stackloader"
if feature_flags and feature_flags["space_travel"] then
  q.max_belt_stack_size = 4
end

data:extend { p, q }
