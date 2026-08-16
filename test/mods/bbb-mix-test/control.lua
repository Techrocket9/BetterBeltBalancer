-- bbb-mix-test: what happens when a balancer carries MORE THAN ONE KIND of item.
--
-- Every other suite ran iron plates through everything. That is the right
-- default -- it makes a throughput number mean something -- and it left the
-- whole multi-kind half of `guest/go/carry.go` unexercised: the pool's
-- (name, quality, stack size) key, the per-kind split, insertRemainder's walk
-- over several groups, and the BOUND on how many groups one pool can carry.
--
-- WHAT IT CANNOT REACH, and it is a property of the DLC rather than of the rigs:
-- `compile.go`'s `detailedTally` and `kindAt`. This suite is base only, and
-- below the stacking gate the drain takes the flat totals and that code is never
-- called at all -- so no rig here exercises it however many kinds it runs.
-- Multi-kind AND STACKED lives in the `plat` suite's `smix` band.
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. It builds rigs, counts items
-- BY NAME on both surfaces at named ticks, and logs the numbers; test/
-- assert-mix.py decides whether they are right. A test mod that computed the
-- expected answer in Lua would be a second implementation of the thing under
-- test.
--
-- THE RIGS, one per y band on a flat scratch surface:
--
--   ctrl     a bare express belt, chest to chest, iron plates. The yardstick,
--            exactly as in M2: "full throughput" is a comparison against the
--            engine rather than against arithmetic on a wiki number.
--   duo      a 2->2 balancer fed by two PURE belts -- one iron, one copper --
--            draining freely into two chests. Does a balancer carrying two
--            kinds still deliver two belts' worth, and does each output see
--            BOTH kinds?
--   mixfull  a 2->2 fed by two SUSHI belts, outputs DEAD-ENDED so the hidden
--            network fills and stays full. Then a forced recompile inside one
--            tick: is every kind conserved exactly?
--   many     a 4-in 4-out balancer fed by four sushi belts drawing on 48
--            distinct base items between them, outputs dead-ended. Past the
--            pool's bound of 32 groups the drain cannot CARRY them all, and
--            what it cannot carry has to reach the WORLD rather than stop
--            existing. This is the rig that fails on the guest that shipped.
--
-- HOW A SUSHI BELT IS MADE HERE, AND WHY IT IS NOT SIX FILTERS IN ONE CHEST.
-- The obvious rig is an infinity chest with six filters feeding one loader, and
-- the `probe` band below is that rig, kept precisely so this paragraph is a
-- measurement rather than an assertion. It does not produce a mixed belt: a
-- loader draws from the first stack it finds in the source inventory and the
-- infinity chest tops that same stack straight back up. See the `[BBB-MIX] t=…
-- rig=probe out1 kinds=…` line -- assert-mix.py prints it and the numbers are in
-- CLAUDE.md.
--
-- So a source here holds ONE filter at a time and ROTATES it, with
-- `remove_unfiltered_items` on so the previous kind is voided rather than left
-- for the loader to prefer. The result is a banded belt -- a short run of each
-- kind, four ticks of belt each -- which is what a real sushi bus looks like
-- anyway, and it is deterministic: the band boundaries are a function of
-- `game.tick` and nothing else.
--
-- FORCING THE FLUSH. The guest batches: a build event updates its registry
-- inside the event and defers the recompile to the next tick (`fk.defer`), so a
-- measurement taken in the tick that laid the belt would see nothing at all.
-- `bbb-audit` -- a shipped marker prototype whose whole purpose is "re-classify
-- and repair everything, now" -- is the synchronous escape hatch. That is also
-- why on_init ends with one: `--create` never reaches a tick, so without it
-- every network in the save would be compiled on the first tick of the
-- BENCHMARK instead.

local PART = "bbb-balancer-part"
local AUDIT = "bbb-audit"
local BELT = "express-transport-belt"
local LOADER = "bbbt-loader"
local E = defines.direction.east
local S = defines.direction.south

local SURF = "bbb-mix-a"

--------------------------------------------------------------------------------
-- the item lists
--
-- Every name is checked against `prototypes.item` at init and a missing one is a
-- hard error, so a rename in a future Factorio fails the run in the CREATE log
-- with the name in it rather than producing a rig that quietly carries fewer
-- kinds than it claims.
--------------------------------------------------------------------------------

-- Two sushi bands of six, so `mixfull` can hold up to twelve distinct kinds --
-- comfortably inside the pool's bound of 32, which is the point: this rig
-- exercises the multi-kind tally, split and reinsertion paths WITHOUT
-- overflowing anything.
local MIXFULL_ITEMS = {
  { "iron-plate", "copper-plate", "steel-plate", "iron-gear-wheel",
    "copper-cable", "electronic-circuit" },
  { "stone", "stone-brick", "coal", "wood", "pipe", "plastic-bar" },
}

-- Four bands of twelve = 48 distinct kinds, comfortably PAST the pool's bound of
-- 32 groups. That is the whole purpose of the `many` rig, and 48 rather than 33
-- because the bound is on GROUPS: a base-only run has one quality and no belt
-- stacking, so a group is exactly a name here, but a rig sized at the bound plus
-- one would stop overflowing the day either of those changed.
--
-- No two bands share a name, so the count below is the count of distinct kinds
-- and `check_items` says so out loud rather than leaving it to be counted from
-- the source. Every name is a base 2.0.77 item.
local MANY_ITEMS = {
  { "iron-plate", "copper-plate", "steel-plate", "iron-ore", "copper-ore",
    "coal", "stone", "stone-brick", "wood", "iron-gear-wheel", "copper-cable",
    "iron-stick" },
  { "electronic-circuit", "advanced-circuit", "processing-unit", "plastic-bar",
    "sulfur", "battery", "explosives", "engine-unit", "electric-engine-unit",
    "flying-robot-frame", "low-density-structure", "rocket-fuel" },
  { "pipe", "pipe-to-ground", "transport-belt", "fast-transport-belt",
    "express-transport-belt", "underground-belt", "splitter", "inserter",
    "fast-inserter", "long-handed-inserter", "small-electric-pole",
    "medium-electric-pole" },
  { "automation-science-pack", "logistic-science-pack", "military-science-pack",
    "chemical-science-pack", "production-science-pack", "utility-science-pack",
    "speed-module", "efficiency-module", "productivity-module", "solid-fuel",
    "uranium-ore", "barrel" },
}

-- Every name is checked against `prototypes.item` and a missing one is a HARD
-- ERROR naming it, in the create log, rather than a rig that quietly carries
-- fewer kinds than it claims -- which for `many` would mean an overflow rig that
-- never overflows and passes vacuously. `distinct` is logged for the same
-- reason: assert-mix.py reads it and requires the number it was promised.
local function check_items()
  local missing, seen, distinct = {}, {}, 0
  for _, band in ipairs(MIXFULL_ITEMS) do
    for _, name in ipairs(band) do
      if not prototypes.item[name] then missing[#missing + 1] = name end
    end
  end
  for _, band in ipairs(MANY_ITEMS) do
    for _, name in ipairs(band) do
      if not prototypes.item[name] then missing[#missing + 1] = name end
      if not seen[name] then seen[name] = true; distinct = distinct + 1 end
    end
  end
  if #missing > 0 then
    error("bbb-mix-test: no such item prototype: " .. table.concat(missing, ", "))
  end
  log(string.format("[BBB-MIX] item lists ok: many distinct=%d mixfull=%d",
    distinct, #MIXFULL_ITEMS[1] + #MIXFULL_ITEMS[2]))
end

--------------------------------------------------------------------------------
-- surface
--------------------------------------------------------------------------------

local function make_surface(name, rows)
  local mgs = {
    width = 512,
    height = 512,
    water = 0,
    peaceful_mode = true,
    default_enable_all_autoplace_controls = false,
    autoplace_settings = {
      tile       = { treat_missing_as_default = false, settings = { ["grass-1"] = {} } },
      decorative = { treat_missing_as_default = false, settings = {} },
      entity     = { treat_missing_as_default = false, settings = {} },
    },
    cliff_settings = { richness = 0 },
    starting_points = {},
    property_expression_names = { cliffiness = "0" },
  }
  local s = game.create_surface(name, mgs)
  s.always_day = true
  s.request_to_generate_chunks({ x = 0, y = rows / 2 }, math.ceil(rows / 32) + 3)
  s.force_generate_chunk_requests()

  local area = { { -34, -20 }, { 20, rows + 20 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -34, 20 do
    for y = -20, rows + 20 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  s.set_tiles(tiles, true, false, false, false)
  return s
end

local function surf() return game.surfaces[SURF] end

--------------------------------------------------------------------------------
-- pieces
--------------------------------------------------------------------------------

local function P(x, y) return { x + 0.5, y + 0.5 } end

-- audit_now asks the guest to drain its deferred queue and re-classify every
-- cluster, synchronously, inside this call. See "forcing the flush" above.
local function audit_now()
  surf().create_entity {
    name = AUDIT, position = P(-30, 0), force = "player", raise_built = true,
  }
end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-mix-test: could not place %s at %s (%d,%d)", name, s.name, x, y))
  end
  return e
end

-- A PURE source: one item, one filter, never rotated.
local function source(s, x, y, item)
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = "player" }
  c.infinity_container_filters = {
    { index = 1, name = item, count = 1000, mode = "at-least" },
  }
  put(s, LOADER, x + 1, y, { direction = E, type = "output" })
  return c
end

-- A SUSHI source: the same chest and loader, but its single filter is rewritten
-- every ROTATE ticks and `remove_unfiltered_items` voids what the last band left
-- behind. `offset` staggers the bands so that four of these do not all switch to
-- their first item on the same tick.
local ROTATE = 4

local function sushi_source(s, x, y, items, offset)
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = "player" }
  c.remove_unfiltered_items = true
  c.infinity_container_filters = {
    { index = 1, name = items[1], count = 1000, mode = "at-least" },
  }
  put(s, LOADER, x + 1, y, { direction = E, type = "output" })
  storage.sushi[#storage.sushi + 1] = { chest = c, items = items, offset = offset }
  return c
end

local function rotate_sushi(tick)
  if tick % ROTATE ~= 0 then return end
  -- Factorio is Lua 5.2.1: there is no `//` operator, and `/` is always a float.
  local step = math.floor(tick / ROTATE)
  for _, sr in ipairs(storage.sushi) do
    if sr.chest.valid then
      local i = (step + sr.offset) % #sr.items + 1
      sr.chest.infinity_container_filters = {
        { index = 1, name = sr.items[i], count = 1000, mode = "at-least" },
      }
    end
  end
end

local function sink(s, x, y)
  put(s, LOADER, x, y, { direction = E, type = "input" })
  return s.create_entity { name = "steel-chest", position = P(x + 1, y), force = "player" }
end

--------------------------------------------------------------------------------
-- rigs
--
--   x=-5 source chest   -4 loader   -3..-1 belts   0 PART   1..3 belts
--   x=4 sink loader     5 chest        (dead-ended rigs stop after x=3)
--
-- The tile NORTH of a rig's first part is deliberately left empty on every rig:
-- it is the one free face, and laying a south-facing belt on it is a real edge
-- change, so the fingerprint moves and the network is torn down and rebuilt.
-- That is how every conservation check here forces a recompile.
--------------------------------------------------------------------------------

local PITCH = 14

local RIGS = {
  { name = "ctrl" },
  -- The multi-filter source, with NO balancer at all: chest to belt to chest.
  -- It is here to keep the paragraph at the top of this file honest -- what one
  -- infinity chest with six filters actually puts on a belt is a measurement,
  -- and this is where it is taken.
  { name = "probe", multi = MIXFULL_ITEMS[1] },
  { name = "duo", parts = 2, feeds = { "iron-plate", "copper-plate" }, drained = true },
  { name = "mixfull", parts = 2, sushi = MIXFULL_ITEMS },
  -- Four parts, four in, four out, dead-ended -- M2's `sat4` shape, which is the
  -- one this repo has already measured holding ~230 items when it is full. That
  -- matters: 48 kinds have to be SIMULTANEOUSLY standing on the lines for the
  -- pool to overflow, and a network that only holds a dozen items could not do
  -- it however many kinds went past.
  { name = "many", parts = 4, sushi = MANY_ITEMS },
}

-- A chest carrying EVERY filter at once. Only the probe band uses it.
local function multi_source(s, x, y, items)
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = "player" }
  local f = {}
  for i, name in ipairs(items) do
    f[i] = { index = i, name = name, count = 1000, mode = "at-least" }
  end
  c.infinity_container_filters = f
  put(s, LOADER, x + 1, y, { direction = E, type = "output" })
  return c
end

local function build_rig(cfg, base)
  local s = surf()
  local r = { name = cfg.name, base = base, out = {}, cfg = cfg }

  if not cfg.parts then -- a bare belt, chest to chest
    if cfg.multi then
      multi_source(s, -5, base, cfg.multi)
    else
      source(s, -5, base, "iron-plate")
    end
    for x = -3, 3 do put(s, BELT, x, base, { direction = E }) end
    r.out[1] = sink(s, 4, base)
    return r
  end

  -- Parts FIRST, belts after, so that the belt events are what drive the
  -- compiles -- the same order M2 builds in and for the same reason.
  for i = 0, cfg.parts - 1 do put(s, PART, 0, base + i) end

  for i = 1, cfg.parts do
    local y = base + i - 1
    if cfg.sushi then
      sushi_source(s, -5, y, cfg.sushi[i], (i - 1) * 3)
    else
      source(s, -5, y, cfg.feeds[i])
    end
    for x = -3, -1 do put(s, BELT, x, y, { direction = E }) end
    for x = 1, 3 do put(s, BELT, x, y, { direction = E }) end
    -- A dead-ended output backs up, which is what fills the hidden network and
    -- keeps it full: the conservation rigs need a network that is holding as
    -- much as it can hold.
    if cfg.drained then r.out[i] = sink(s, 4, y) end
  end
  return r
end

--------------------------------------------------------------------------------
-- counting, BY ITEM NAME
--
-- Everything, on both surfaces: on the ground, on a belt, inside a splitter's
-- transport lines, in a chest. An item this mod can lose is an item that left
-- this total, and there is nowhere else for one to be.
--
-- Counting only the visible side would not be conservation at all -- the point
-- of the network is that most of the items are somewhere the player cannot see,
-- so a teardown that deleted them would look like a gain on the visible side.
-- The whole hidden surface is counted rather than one slot, because no tick
-- passes between the two samples and every other rig's network is therefore
-- frozen.
--
-- PER NAME rather than as one total, which is the whole point of this suite: a
-- teardown that dropped one KIND and reinserted the rest conserves nothing, and
-- a single total would have to lose the same number of items twice in opposite
-- directions to hide it.
--------------------------------------------------------------------------------

local VIS_AREA = { { -34, -20 }, { 20, 80 } }
local HIDDEN_AREA = { { -16, -16 }, { 2200, 400 } }

local function count_area(s, area, kinds)
  if not (s and s.valid) then return 0 end
  local ground = 0
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid then
      if e.type == "item-entity" then
        if e.stack and e.stack.valid_for_read then
          kinds[e.stack.name] = (kinds[e.stack.name] or 0) + e.stack.count
          ground = ground + e.stack.count
        end
      else
        local ok, n = pcall(function() return e.get_max_transport_line_index() end)
        if ok and n then
          for i = 1, n do
            for _, item in pairs(e.get_transport_line(i).get_contents()) do
              kinds[item.name] = (kinds[item.name] or 0) + item.count
            end
          end
        end
        local inv = e.get_inventory(defines.inventory.chest)
        if inv then
          for _, item in pairs(inv.get_contents()) do
            kinds[item.name] = (kinds[item.name] or 0) + item.count
          end
        end
      end
    end
  end
  return ground
end

-- The infinity chests are DELIBERATELY EXCLUDED. They mint and void items every
-- tick by design, so anything they hold is not a conserved quantity -- including
-- them would make every count a measurement of the source rather than of the
-- balancer. They are found by type and their contents subtracted back out.
-- The HIDDEN surface is counted into its own table as well as the total, and
-- that second number is the anti-vacuity one: it is exactly the compiled
-- networks' contents and nothing else, so "how many distinct kinds were really
-- inside a balancer when it was torn down" is answerable without trusting the
-- guest's own account of it.
local function count_kinds()
  local vis, hid = {}, {}
  local ground = count_area(surf(), VIS_AREA, vis)
  ground = ground + count_area(game.surfaces["bbb-hidden"], HIDDEN_AREA, hid)
  for _, c in pairs(surf().find_entities_filtered {
    area = VIS_AREA, type = "infinity-container",
  }) do
    if c.valid then
      for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
        vis[item.name] = (vis[item.name] or 0) - item.count
      end
    end
  end
  local kinds, total = {}, 0
  for _, t in ipairs { vis, hid } do
    for name, n in pairs(t) do kinds[name] = (kinds[name] or 0) + n end
  end
  for name, n in pairs(kinds) do
    if n == 0 then kinds[name] = nil else total = total + n end
  end
  local htotal, hkinds = 0, 0
  for _, n in pairs(hid) do
    if n > 0 then htotal = htotal + n; hkinds = hkinds + 1 end
  end
  return kinds, total, ground, htotal, hkinds
end

-- Sorted by name, so two samples of the same world produce the same lines in the
-- same order on every machine. `pairs` order over a Lua table is not a promise.
local function emit(tag)
  local kinds, total, ground, htotal, hkinds = count_kinds()
  local names = {}
  for name in pairs(kinds) do names[#names + 1] = name end
  table.sort(names)
  log(string.format(
    "[BBB-MIX] count tag=%s total=%d ground=%d kinds=%d hidden=%d hkinds=%d",
    tag, total, ground, #names, htotal, hkinds))
  for _, name in ipairs(names) do
    log(string.format("[BBB-MIX] kind tag=%s name=%s count=%d", tag, name, kinds[name]))
  end
end

--------------------------------------------------------------------------------
-- the conservation check
--
-- Inside a single tick, with no other movement possible: count every item by
-- name on both surfaces, lay a belt on the cluster's one free face, force the
-- recompile with an audit marker, count again. Nothing else can have moved, so
-- the difference is exactly what the teardown did with what it drained.
--------------------------------------------------------------------------------

local function conserve(rig_name)
  local r = storage.rigs[rig_name]
  if not r then error("bbb-mix-test: no rig " .. rig_name) end
  log(string.format("[BBB-MIX] mark rig=%s tick=%d", rig_name, game.tick))
  emit(rig_name .. "-before")
  -- The north face is the only free one; a belt facing SOUTH there is a new
  -- INPUT edge, so the port count goes up and the network the rebuild produces
  -- is at least as big as the one it replaced. That matters: a shrink would
  -- spill legitimately (carry.go, decision 4) and this check is about the
  -- kinds, not about capacity.
  put(surf(), BELT, 0, r.base - 1, { direction = S })
  audit_now()
  emit(rig_name .. "-after")
end

--------------------------------------------------------------------------------
-- reporting
--------------------------------------------------------------------------------

local function chest_kinds(c)
  if not (c and c.valid) then return nil end
  local parts, total = {}, 0
  local names, counts = {}, {}
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
    names[#names + 1] = item.name
    counts[item.name] = (counts[item.name] or 0) + item.count
    total = total + item.count
  end
  table.sort(names)
  for _, n in ipairs(names) do parts[#parts + 1] = n .. ":" .. counts[n] end
  return total, table.concat(parts, ",")
end

local function report(tick)
  for _, name in ipairs { "ctrl", "probe", "duo" } do
    local r = storage.rigs[name]
    local outs = {}
    for i = 1, #r.out do
      local total, detail = chest_kinds(r.out[i])
      outs[i] = tostring(total or -1)
      log(string.format("[BBB-MIX] t=%d rig=%s out%d kinds=%s",
        tick, name, i, detail or ""))
    end
    log(string.format("[BBB-MIX] t=%d rig=%s out=[%s]", tick, name, table.concat(outs, " ")))
  end
end

--------------------------------------------------------------------------------
-- schedule
--
-- Every edit lands BEFORE the throughput window opens, so `duo`'s rate is
-- measured over a stretch in which nothing was touched -- the same discipline
-- M2's `regrow` rig follows.
--------------------------------------------------------------------------------

local SCHEDULE = {
  [800] = function() conserve("duo") end,
  [1000] = function() conserve("mixfull") end,
  -- `many` last of the three, and 200 ticks after `mixfull`, because it is the
  -- one whose result depends on how FULL it is: 48 kinds only overflow a
  -- 32-group pool if 48 kinds are simultaneously standing on the lines, and a
  -- dead-ended network freezes holding whatever was in flight when it filled.
  -- Every source cycles its twelve items every 48 ticks, so by t=1200 each of
  -- them has run its list twenty-five times over.
  [1200] = function() conserve("many") end,
  [1400] = function() report(1400) end,
  [3140] = function() report(3140) end,
}

script.on_init(function()
  check_items()
  storage.sushi = {}
  local rows = (#RIGS + 1) * PITCH
  make_surface(SURF, rows)
  storage.rigs = {}
  for i, cfg in ipairs(RIGS) do
    storage.rigs[cfg.name] = build_rig(cfg, (i - 1) * PITCH)
  end
  -- Compile everything NOW rather than on the first tick after the save is
  -- loaded. See "forcing the flush" at the top of this file.
  audit_now()
  log(string.format("[BBB-MIX] init complete: %d rigs", #RIGS))
end)

script.on_event(defines.events.on_tick, function(e)
  rotate_sushi(e.tick)
  local f = SCHEDULE[e.tick]
  if f then f() end
end)
