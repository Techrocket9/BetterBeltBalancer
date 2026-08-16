-- Default bench configuration.
--
-- bench/run.sh OVERWRITES this file inside the staged copy of the mod
-- (bench/tmp/mods/bbb-bench-setup/config.lua) before creating the save. The
-- values here are only the defaults you get if you run the mod by hand.
return {
  scenario = "saturated", -- "saturated" | "idle" | "control" | "control-idle"
  n = 1,                  -- number of independent test rigs
  k = 4,                  -- balancer size: K input belts, K x K parts, K output belts
  tier = "express",       -- "normal" | "fast" | "express"
  item = "iron-ore",
  part_name = "balancer-part", -- the balancer mod's part prototype:
                          -- "balancer-part" for belt-balancer-2 and -3,
                          -- "bbb-balancer-part" for BetterBeltBalancer

  meter_interval = 600,   -- ticks between throughput samples (0 disables metering)
}
