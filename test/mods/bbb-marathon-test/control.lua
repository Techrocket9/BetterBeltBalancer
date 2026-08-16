-- bbb-marathon-test: what a NET-ZERO world operation costs the guest heap,
-- forever.
--
-- Under `-gc=leaking` every transient guest allocation is permanent -- it is in
-- the linear memory, in every save, and in every multiplayer join, for the life
-- of the session. So the question a 300-hour game asks is not "does it leak"
-- (everything does) but "what is the SLOPE": how many bytes does one complete
-- place-and-remove cycle add that never come back, and is that slope flat.
--
-- The instrument is the guest's own heap probe. `[BBB] heap <what> sys=… alloc=…`
-- is written at every audit (guest/go/gc.go); under `-gc=leaking` `alloc` is
-- TinyGo's bump allocator reporting every byte it has ever handed out, which is
-- exactly the permanent heap. This mod drives a leg, places a `bbb-audit`
-- marker, and lets the guest print the number. test/assert-marathon.py fits the
-- slope.
--
-- THE LEGS, and each one is chosen so that ONE term dominates it:
--
--   cal   ten audits with the world untouched. The audit is not free -- it
--         re-classifies every cluster in the save -- and every other leg pays
--         exactly one of them per iteration, so this is the constant that gets
--         subtracted from all of them.
--   A     the headline cycle: place a 4-part balancer and its 14 belts in one
--         tick, run it under load, remove all 18 entities, audit. Net zero.
--   B     lay a belt two tiles from a finished balancer and pick it up again.
--         The cluster is queued and re-classified; the fingerprint says nothing
--         moved and NOTHING is rebuilt. This is the common case -- a player
--         building near a balancer -- and its slope is the one that multiplies
--         by the biggest number in a real game.
--   C     remove one of a balancer's input belts and put it back: an edge
--         really moves, so this is two full teardown-and-rebuilds per
--         iteration. A rotation costs the same, by construction: the
--         fingerprint covers direction.
--   D     lay a belt eighteen tiles from anything and pick it up again. No
--         cluster is within the two-tile gate, so no compile happens at all and
--         what is left is the raw per-EVENT cost of being entered. On a
--         multiplayer server this is the term with the largest multiplier of
--         all -- every belt anyone lays anywhere on the map pays it.
--   E     six entities placed in one tick and removed in one tick: a small
--         blueprint paste and its undo.
--   G     the same edit as C but on a 4x4 -- sixteen parts, four in, four out,
--         a 32-entity network. C and G together are what says the compile
--         term SCALES with the network rather than with the number of edits,
--         which is what a projection out to three hundred hours needs.
--   F     a saturated balancer grown by one part and taken apart again, with
--         every item on the surface counted every cycle. This is both a slope
--         leg and the conservation kill-test: 100 cycles of add-part /
--         remove-everything on a network that is FULL, and the count may never
--         rise and may not fall by more than the documented spill loss.
--
-- Every leg ends with the world in the state the calibration measured, so all
-- the audits cost the same and the subtraction is honest.
--
-- ASSERTS NOTHING. The guest's own lines are the assertion surface, exactly as
-- in the other five suites.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbt-loader"
local AUDIT = "bbb-audit"
local E = defines.direction.east

-- Rigs are 30 rows apart, which is more than `spill_item_stack`'s 12-tile
-- radius in both directions: leg F counts a band and a neighbour's spill
-- landing in it would read as items minted out of nothing.
local PITCH = 30
local SURF = "bbb-mar"

local KEEP, CYCLE, CHURN, FAR = 0, PITCH, 2 * PITCH, 3 * PITCH
local BIG = 4 * PITCH
local ROWS = 5 * PITCH

-- Leg F's finite stock. A steel chest holds 48 stacks, so this is also the most
-- an insert can put in one -- big enough that 100 cycles never starve the rig,
-- and a round number the count can be read against.
local CHURN_STOCK = 4800
local CHURN_BAND = { { -20, CHURN - 14 }, { 20, CHURN + 16 } }

--------------------------------------------------------------------------------
-- world helpers, the same shapes the M2 and M3 mods use
--------------------------------------------------------------------------------

local function P(x, y) return { x + 0.5, y + 0.5 } end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-marathon-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

-- put_soft is put where a collision is a legitimate outcome of the schedule
-- rather than a broken test.
local function put_soft(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  return s.create_entity(args)
end

local function at(s, x, y, filter)
  local f = filter or {}
  f.position = P(x, y)
  return s.find_entities_filtered(f)[1]
end

-- kill removes an entity the way a player mining it does, as far as the guest
-- is concerned: an event is raised and the entity is gone when the dispatch
-- returns.
local function kill(s, x, y, filter)
  local e = at(s, x, y, filter)
  if e and e.valid then e.destroy { raise_destroy = true } end
  return e ~= nil
end

local function surf() return game.surfaces[SURF] end

local function make_surface()
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
  local s = game.create_surface(SURF, mgs)
  s.always_day = true
  s.request_to_generate_chunks({ x = 0, y = ROWS / 2 }, math.ceil(ROWS / 32) + 3)
  s.force_generate_chunk_requests()

  local area = { { -22, -16 }, { 26, ROWS + 16 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -22, 26 do
    for y = -16, ROWS + 16 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  s.set_tiles(tiles, true, false, false, false)
  return s
end

-- An infinity source and a chest sink, both permanent. Only the parts and the
-- belts between them move during a leg.
local function source_inf(s, y)
  local c = s.create_entity { name = "infinity-chest", position = P(-6, y), force = "player" }
  c.infinity_container_filters = {
    { index = 1, name = "iron-plate", count = 1000, mode = "at-least" },
  }
  put(s, LOADER, -5, y, { direction = E, type = "output" })
end

local function source_finite(s, y, count)
  local c = s.create_entity { name = "steel-chest", position = P(-6, y), force = "player" }
  c.get_inventory(defines.inventory.chest).insert { name = "iron-plate", count = count }
  put(s, LOADER, -5, y, { direction = E, type = "output" })
end

local function sink(s, y)
  put(s, LOADER, 4, y, { direction = E, type = "input" })
  return s.create_entity { name = "steel-chest", position = P(5, y), force = "player" }
end

-- count_band totals every item standing in a box: on the ground, on a belt, in
-- a splitter's transport lines, in a chest. The same routine the M3 suite
-- counts its stress surface with.
local function count_band(s, area)
  if not (s and s.valid) then return 0 end
  local total = 0
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid then
      if e.type == "item-entity" then
        if e.stack and e.stack.valid_for_read then total = total + e.stack.count end
      else
        local ok, n = pcall(function() return e.get_max_transport_line_index() end)
        if ok and n then
          for i = 1, n do total = total + e.get_transport_line(i).get_item_count() end
        end
        local inv = e.get_inventory(defines.inventory.chest)
        if inv then
          for _, item in pairs(inv.get_contents()) do total = total + item.count end
        end
      end
    end
  end
  return total
end

--------------------------------------------------------------------------------
-- the probe
--
-- The audit marker is the shipped synchronous "drain the queue and re-classify
-- now" trigger, and it is also what makes the guest print its heap. The
-- [BBB-MAR] line goes FIRST so the assertion script can attribute the
-- `post-audit` line that follows it to this leg and this iteration.
--------------------------------------------------------------------------------

local function probe(leg, i)
  log(string.format("[BBB-MAR] leg=%s i=%d", leg, i))
  surf().create_entity {
    name = AUDIT, position = P(20, 4), force = "player", raise_built = true,
  }
end

--------------------------------------------------------------------------------
-- the legs
--
-- Each is a function of (phase, i): `phase` counts ticks within the iteration,
-- `i` counts iterations. Every one of them leaves the world exactly as it found
-- it, which is what makes the audits comparable.
--------------------------------------------------------------------------------

-- cal: nothing but the probe. The constant every other leg pays once.
local function leg_cal(phase, i, name)
  if phase == 0 then probe(name, i) end
end

-- A: a whole balancer, built and removed. 4 parts, 8 input belts, 6 output
-- belts, all in one tick each way -- so one deferred flush builds it and one
-- takes it down, which is the batching the guest is designed around.
local A_PARTS, A_IN, A_OUT = 4, 2, 2

local function leg_A(phase, i)
  local s = surf()
  if phase == 0 then
    for r = 0, A_PARTS - 1 do put_soft(s, PART, 0, CYCLE + r) end
    for r = 0, A_IN - 1 do
      for x = -4, -1 do put_soft(s, BELT, x, CYCLE + r, { direction = E }) end
    end
    for r = 0, A_OUT - 1 do
      for x = 1, 3 do put_soft(s, BELT, x, CYCLE + r, { direction = E }) end
    end
  elseif phase == 9 then
    for r = 0, A_PARTS - 1 do kill(s, 0, CYCLE + r, { name = PART }) end
    for r = 0, A_IN - 1 do
      for x = -4, -1 do kill(s, x, CYCLE + r, { type = "transport-belt" }) end
    end
    for r = 0, A_OUT - 1 do
      for x = 1, 3 do kill(s, x, CYCLE + r, { type = "transport-belt" }) end
    end
  elseif phase == 11 then
    probe("A", i)
  end
end

-- B: a belt two tiles above the KEEP balancer's top part. Inside the guest's
-- two-tile neighbour gate, so the cluster is queued and re-classified; outside
-- the edge, so the fingerprint matches and nothing is rebuilt.
local function leg_B(phase, i)
  local s = surf()
  if phase == 0 then
    put_soft(s, BELT, 0, KEEP - 2, { direction = E })
  elseif phase == 1 then
    kill(s, 0, KEEP - 2, { type = "transport-belt" })
  elseif phase == 3 then
    probe("B", i)
  end
end

-- C: one of KEEP's input belts, removed and put back. The edge list really
-- moves both times, so an iteration is TWO full teardown-and-rebuilds.
local function leg_C(phase, i)
  local s = surf()
  if phase == 0 then
    kill(s, -1, KEEP, { type = "transport-belt" })
  elseif phase == 1 then
    put_soft(s, BELT, -1, KEEP, { direction = E })
  elseif phase == 3 then
    probe("C", i)
  end
end

-- D: a belt eighteen tiles from the nearest balancer part. The guest is entered
-- -- the engine's filter admits every belt-connectable on the map -- and the
-- in-guest position gate rejects it without a compile. What is left is the cost
-- of being entered at all, which is the term a busy server multiplies by the
-- largest number.
local function leg_D(phase, i)
  local s = surf()
  if phase == 0 then
    put_soft(s, BELT, 18, FAR, { direction = E })
  elseif phase == 1 then
    kill(s, 18, FAR, { type = "transport-belt" })
    probe("D", i)
  end
end

-- E: six entities in one tick and six out in one tick -- the event shape of a
-- small blueprint paste and its undo.
local function leg_E(phase, i)
  local s = surf()
  if phase == 0 then
    put_soft(s, PART, 14, FAR)
    put_soft(s, PART, 14, FAR + 1)
    put_soft(s, BELT, 13, FAR, { direction = E })
    put_soft(s, BELT, 13, FAR + 1, { direction = E })
    put_soft(s, BELT, 15, FAR, { direction = E })
    put_soft(s, BELT, 15, FAR + 1, { direction = E })
  elseif phase == 4 then
    kill(s, 14, FAR, { name = PART })
    kill(s, 14, FAR + 1, { name = PART })
    kill(s, 13, FAR, { type = "transport-belt" })
    kill(s, 13, FAR + 1, { type = "transport-belt" })
    kill(s, 15, FAR, { type = "transport-belt" })
    kill(s, 15, FAR + 1, { type = "transport-belt" })
  elseif phase == 7 then
    probe("E", i)
  end
end

-- G: leg C on a 4x4 -- the M2 `sat4` shape, sixteen parts and a 32-entity
-- network, against C's two parts and eleven. Two full teardown-and-rebuilds per
-- iteration, exactly as C, so the ratio between them is the compile term's
-- dependence on network size and nothing else.
local function leg_G(phase, i)
  local s = surf()
  if phase == 0 then
    kill(s, -1, BIG, { type = "transport-belt" })
  elseif phase == 1 then
    put_soft(s, BELT, -1, BIG, { direction = E })
  elseif phase == 3 then
    probe("G", i)
  end
end

-- F: the conservation leg. A saturated 2-part balancer on a FINITE stock is
-- grown by a third part (merge -> teardown and rebuild while the network is
-- full), then taken apart entirely, then rebuilt. Every item in the band is
-- counted while the network is DOWN -- which is the only moment the count is
-- complete, because a teardown spills the hidden network onto the visible
-- ground.
local function leg_F(phase, i)
  local s = surf()
  if phase == 0 then
    put_soft(s, PART, 0, CHURN + 2)
  elseif phase == 6 then
    for r = 0, 2 do kill(s, 0, CHURN + r, { name = PART }) end
  elseif phase == 7 then
    probe("F", i)
  elseif phase == 8 then
    log(string.format("[BBB-MAR] churn i=%d total=%d", i, count_band(s, CHURN_BAND)))
    for r = 0, 1 do put_soft(s, PART, 0, CHURN + r) end
  end
end

--------------------------------------------------------------------------------
-- the schedule
--------------------------------------------------------------------------------

local LEGS = {
  { name = "cal",  iters = 10,  period = 3,  run = function(p, i) leg_cal(p, i, "cal") end },
  { name = "A",    iters = 100, period = 12, run = leg_A },
  { name = "calA", iters = 10,  period = 3,  run = function(p, i) leg_cal(p, i, "calA") end },
  { name = "B",    iters = 100, period = 4,  run = leg_B },
  { name = "C",    iters = 100, period = 4,  run = leg_C },
  { name = "D",    iters = 100, period = 2,  run = leg_D },
  { name = "E",    iters = 50,  period = 8,  run = leg_E },
  { name = "G",    iters = 100, period = 4,  run = leg_G },
  { name = "F",    iters = 100, period = 10, run = leg_F },
  { name = "calZ", iters = 10,  period = 3,  run = function(p, i) leg_cal(p, i, "calZ") end },
}

-- A gap between legs, so a leg's last flush cannot land inside the next one's
-- first iteration.
local GAP = 20
local START = 90

local PLAN = {}
do
  local t = START
  for _, leg in ipairs(LEGS) do
    PLAN[#PLAN + 1] = { name = leg.name, t0 = t, run = leg.run,
      iters = leg.iters, period = leg.period }
    t = t + leg.iters * leg.period + GAP
  end
  PLAN.end_tick = t
end

--------------------------------------------------------------------------------

script.on_init(function()
  local s = make_surface()

  -- KEEP: never structurally touched. Legs B and C happen at its edge.
  for r = 0, 1 do put(s, PART, 0, KEEP + r) end
  for r = 0, 1 do
    source_inf(s, KEEP + r)
    for x = -4, -1 do put(s, BELT, x, KEEP + r, { direction = E }) end
    for x = 1, 3 do put(s, BELT, x, KEEP + r, { direction = E }) end
    sink(s, KEEP + r)
  end

  -- CYCLE: only the ends are permanent. Leg A places and removes everything
  -- between them.
  for r = 0, A_IN - 1 do source_inf(s, CYCLE + r) end
  for r = 0, A_OUT - 1 do sink(s, CYCLE + r) end

  -- BIG: the M2 `sat4` shape. Four inputs on the west face of a 4x4 block of
  -- parts, four outputs on the east -- a 32-entity network, against KEEP's
  -- eleven.
  for r = 0, 3 do
    for c = 0, 3 do put(s, PART, c, BIG + r) end
  end
  for r = 0, 3 do
    local ch = s.create_entity {
      name = "infinity-chest", position = P(-4, BIG + r), force = "player",
    }
    ch.infinity_container_filters = {
      { index = 1, name = "iron-plate", count = 1000, mode = "at-least" },
    }
    put(s, LOADER, -3, BIG + r, { direction = E, type = "output" })
    for x = -2, -1 do put(s, BELT, x, BIG + r, { direction = E }) end
    put(s, BELT, 4, BIG + r, { direction = E })
    put(s, LOADER, 5, BIG + r, { direction = E, type = "input" })
    s.create_entity { name = "steel-chest", position = P(6, BIG + r), force = "player" }
  end

  -- CHURN: a finite stock, so leg F's count is a conservation statement.
  for r = 0, 1 do put(s, PART, 0, CHURN + r) end
  for r = 0, 1 do
    source_finite(s, CHURN + r, CHURN_STOCK)
    for x = -4, -1 do put(s, BELT, x, CHURN + r, { direction = E }) end
    for x = 1, 3 do put(s, BELT, x, CHURN + r, { direction = E }) end
    sink(s, CHURN + r)
  end

  -- `--create` never reaches a tick, so the flush every build event armed would
  -- otherwise land on the first tick of the benchmark. The marker drains it
  -- here, exactly as the other suites' on_init does.
  probe("init", 0)
  log(string.format("[BBB-MAR] plan legs=%d end_tick=%d stock=%d",
    #LEGS, PLAN.end_tick, count_band(s, CHURN_BAND)))
end)

script.on_event(defines.events.on_tick, function(ev)
  local t = ev.tick
  for _, leg in ipairs(PLAN) do
    if t >= leg.t0 then
      local span = leg.iters * leg.period
      if t < leg.t0 + span then
        local d = t - leg.t0
        leg.run(d % leg.period, math.floor(d / leg.period) + 1)
        return
      end
    end
  end
  if t == PLAN.end_tick then
    log(string.format("[BBB-MAR] done end_tick=%d churn_final=%d",
      PLAN.end_tick, count_band(surf(), CHURN_BAND)))
  end
end)
