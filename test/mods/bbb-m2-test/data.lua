-- Test-rig prototypes. Nothing here ships; this is the harness's own kit.
--
-- Two of them, and both exist because BASE FACTORIO HAS NO BUILDABLE VERSION of
-- the thing the rig needs to put against a balancer part:
--
--   bbbt-loader         base's `loader-1x1` is a hidden 0.03125 prototype, and
--                       a source that slow could not saturate anything
--   bbbt-lane-splitter  base ships the `lane-splitter` TYPE and not one entity
--                       of it -- the type exists for Space Age's turbo lane
--                       splitter, and there is nothing in `data.raw` to place
--
-- The rigs are otherwise built from stock express belts, splitters,
-- undergrounds and loaders, so what they measure is the rate a real one
-- delivers.

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

-- The mod's OWN hidden lane splitter, made placeable.
--
-- Cloning ours rather than writing one from scratch is the point: it is a real
-- `lane-splitter` prototype that a real Factorio validated, so what `lsio`
-- exercises is the engine's type and `classifySide`'s reading of it, not a
-- prototype the harness invented and might have got wrong. This mod depends on
-- `better-belt-balancer`, so its data stage has already run.
--
-- Its inherited speed (0.25, the hidden network's ceiling) is left alone: the
-- rig is about CLASSIFICATION, and an edge slower than express would cap it and
-- turn a throughput assertion into a measurement of this prototype.
local function lane_splitter(name)
  local p = util.table.deepcopy(data.raw["lane-splitter"]["bbb-lane-splitter"])
  p.name = name
  p.minable = nil
  p.next_upgrade = nil
  p.fast_replaceable_group = nil
  return p
end

data:extend { loader("bbbt-loader"), lane_splitter("bbbt-lane-splitter") }
