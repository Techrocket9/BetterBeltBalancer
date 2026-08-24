-- bbb-bench-setup: builds N identical belt-balancer test rigs into a fresh save.
--
-- Everything happens in on_init, which Factorio runs while `--create` builds the
-- map, so the resulting save already contains the whole rig array and can be fed
-- straight to `--benchmark`. Deliberately plain Lua: this is bench
-- infrastructure, not the product.
--
-- Rig layout (one row per input/output belt, K rows, x grows east):
--
--   col 0        source infinity-chest (spawns `item`; empty in the idle scenarios)
--   col 1..2     source loader, type "output"  (chest -> belt)
--   col 3..5     input transport belts, facing east
--   col 6..6+K-1 K balancer-part columns  (plain belts in the control scenarios)
--                the part prototype is cfg.part_name -- belt-balancer-2 and -3
--                both call theirs "balancer-part", ours is "bbb-balancer-part"
--   col 6+K..+2  output transport belts, facing east
--   col +3..+4   sink loader, type "input"  (belt -> chest)
--   col +5       sink steel-chest, drained by the meter
--
-- The K x K part block is one balancer per rig: the K belts on the west face are
-- its inputs, the K on the east face its outputs.

local cfg = require("config")

local TIERS = {
  normal  = { belt = "transport-belt",         loader = "loader" },
  fast    = { belt = "fast-transport-belt",    loader = "fast-loader" },
  express = { belt = "express-transport-belt", loader = "express-loader" },
}

local EAST = defines.direction.east

-- Rig geometry, all in tile units.
local function rig_width(k) return k + 12 end
local function rig_height(k) return k end
local PAD = 3

local function tile_center(tx, ty)
  return { x = tx + 0.5, y = ty + 0.5 }
end

--------------------------------------------------------------------------------
-- surface
--------------------------------------------------------------------------------

-- A blank, deterministic surface: no water, no cliffs, no resources, no
-- decoratives, no enemies. Anything the generator still manages to produce is
-- bulldozed and paved over below.
local function make_surface(width, height)
  local mgs = {
    width = width + 64,
    height = height + 64,
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

  local surface = game.create_surface("bbb-bench", mgs)
  surface.always_day = true
  surface.freeze_daytime = true

  -- Generate every chunk the rigs will touch, plus a ring of slack.
  local cx = math.ceil((width + 64) / 32 / 2) + 1
  local cy = math.ceil((height + 64) / 32 / 2) + 1
  surface.request_to_generate_chunks({ x = width / 2, y = height / 2 }, math.max(cx, cy))
  surface.force_generate_chunk_requests()

  local area = { { -32, -32 }, { width + 32, height + 32 } }
  for _, e in pairs(surface.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  surface.destroy_decoratives { area = area }

  -- Pave, so nothing the generator did can block a build.
  local tiles = {}
  for x = -32, width + 32 do
    for y = -32, height + 32 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  surface.set_tiles(tiles, true, false, false, false)

  return surface
end

--------------------------------------------------------------------------------
-- rig construction
--------------------------------------------------------------------------------

-- scenario -> what the rig contains. Two independent switches: whether the
-- balancer parts exist, and whether items flow.
local SCENARIOS = {
  saturated      = { parts = true,  items = true  },
  idle           = { parts = true,  items = false },
  control        = { parts = false, items = true  },
  ["control-idle"] = { parts = false, items = false },
  -- The megabase scenarios build a HETEROGENEOUS population instead of n
  -- identical rigs; same two switches, a different builder. See the MEGA
  -- section near the bottom of this file.
  mega                  = { parts = true,  items = true,  mega = true },
  ["mega-idle"]         = { parts = true,  items = false, mega = true },
  ["mega-control"]      = { parts = false, items = true,  mega = true },
  ["mega-control-idle"] = { parts = false, items = false, mega = true },
}

local function build_rig(surface, force, ox, oy, spec)
  local k, tier = spec.k, spec.tier
  local sc = SCENARIOS[spec.scenario]
  local belt_name, loader_name = TIERS[tier].belt, TIERS[tier].loader
  local sinks = {}

  local part_x0 = ox + 6
  local out_x0  = part_x0 + k

  for row = 0, k - 1 do
    local ty = oy + row

    -- source: infinity chest -> loader -> three belts
    local src = surface.create_entity {
      name = "infinity-chest", position = tile_center(ox, ty), force = force,
      raise_built = true,
    }
    if sc.items then
      src.set_infinity_container_filter(1, { index = 1, name = spec.item, count = 200, mode = "at-least" })
    end
    src.remove_unfiltered_items = true

    surface.create_entity {
      name = loader_name, position = { x = ox + 2, y = ty + 0.5 }, direction = EAST,
      type = "output", force = force, raise_built = true,
    }

    -- Belt run. With no parts the belt is continuous across the columns the
    -- parts would have occupied, so items still reach the sink.
    local x_from, x_to = ox + 3, out_x0 + 2
    for tx = x_from, x_to do
      local is_part_column = tx >= part_x0 and tx < out_x0
      if not sc.parts or not is_part_column then
        surface.create_entity {
          name = belt_name, position = tile_center(tx, ty), direction = EAST,
          force = force, raise_built = true,
        }
      end
    end

    -- sink: three belts (built above) -> loader -> chest
    surface.create_entity {
      name = loader_name, position = { x = out_x0 + 4, y = ty + 0.5 }, direction = EAST,
      type = "input", force = force, raise_built = true,
    }
    sinks[#sinks + 1] = surface.create_entity {
      name = "steel-chest", position = tile_center(out_x0 + 5, ty), force = force,
      raise_built = true,
    }
  end

  -- Parts last: belt-balancer-2 links a part to its belts at build time, and
  -- linking from either side works, but building parts into finished belt rows
  -- exercises the path a player actually takes.
  if sc.parts then
    for px = 0, k - 1 do
      for py = 0, k - 1 do
        surface.create_entity {
          name = spec.part_name, position = tile_center(part_x0 + px, oy + py),
          force = force, raise_built = true,
        }
      end
    end
  end

  return sinks
end

--------------------------------------------------------------------------------
-- MEGA: a heterogeneous population at megabase scale
--------------------------------------------------------------------------------
--
-- Everything above builds `n` copies of one shape. The mega scenarios build a
-- MIX, because a megabase is a mix: square balancers, non-power-of-two ones
-- whose spare ports loop back, dead-ended ones whose spare outputs back up, and
-- -- for the first time in a real game rather than in `plan`'s simulator --
-- the three sizes past P=8, up to the P=64 that IS `plan.MaxPorts`.
--
-- A rig here is `n` inputs on the west face and `m` outputs on the east face of
-- a part block `cols` wide and `max(n, m)` tall. Rows past `n` have no source
-- and rows past `m` have no sink, which is the whole of what makes a 3->5 a
-- 3->5. Otherwise the column layout is the uniform rig's, tile for tile.
--
--   `--n` is the number of BLOCKS. Each block is the ten small rigs below; the
--   four giants are built once regardless, because one 64x64 is a measurement
--   and forty of them is a different benchmark.

local MEGA_BLOCK = {
  { name = "2->2", n = 2, m = 2, cols = 2 },
  { name = "2->2", n = 2, m = 2, cols = 2 },
  { name = "2->2", n = 2, m = 2, cols = 2 },
  { name = "3->3", n = 3, m = 3, cols = 3 },
  { name = "3->3", n = 3, m = 3, cols = 3 },
  { name = "4x4",  n = 4, m = 4, cols = 4 },
  { name = "4x4",  n = 4, m = 4, cols = 4 },
  { name = "3->5", n = 3, m = 5, cols = 4 },  -- loopback: spare outputs feed spare inputs
  { name = "5->3", n = 5, m = 3, cols = 4 },  -- dead end: spare outputs back up
  { name = "8x8",  n = 8, m = 8, cols = 4 },
}

-- Built once each, whatever --n says. `defer` on the 64x64 is what lets the
-- create time its very first compile against an otherwise-identical audit; see
-- mega_init.  `over_limit` is the 65-input cluster: P would be 128, the guest
-- must refuse it BEFORE tearing anything down, and it must deliver nothing.
local MEGA_GIANTS = {
  { name = "16x16", n = 16, m = 16, cols = 4 },
  { name = "32x32", n = 32, m = 32, cols = 4 },
  { name = "64x64", n = 64, m = 64, cols = 4, defer = true },
  { name = "65->1", n = 65, m = 1,  cols = 2, over_limit = true },
}

local function shape_height(s) return math.max(s.n, s.m) end
local function shape_width(s)  return s.cols + 12 end

local function build_mega_rig(surface, force, ox, oy, shape, spec)
  local sc = SCENARIOS[spec.scenario]
  local belt_name, loader_name = TIERS[spec.tier].belt, TIERS[spec.tier].loader
  local nin, mout, pc = shape.n, shape.m, shape.cols
  local h = shape_height(shape)
  local part_x0 = ox + 6
  local out_x0  = part_x0 + pc
  local sinks = {}

  for row = 0, h - 1 do
    local ty = oy + row

    if row < nin then
      local src = surface.create_entity {
        name = "infinity-chest", position = tile_center(ox, ty), force = force,
        raise_built = true,
      }
      if sc.items then
        src.set_infinity_container_filter(1, { index = 1, name = spec.item, count = 200, mode = "at-least" })
      end
      src.remove_unfiltered_items = true
      surface.create_entity {
        name = loader_name, position = { x = ox + 2, y = ty + 0.5 }, direction = EAST,
        type = "output", force = force, raise_built = true,
      }
      for tx = ox + 3, part_x0 - 1 do
        surface.create_entity {
          name = belt_name, position = tile_center(tx, ty), direction = EAST,
          force = force, raise_built = true,
        }
      end
    end

    -- The part columns carry belts in the control scenarios, so a row that has
    -- both a source and a sink still delivers. A row with a source and no sink
    -- backs up -- which is what the dead-ended shapes do in the balancer arm
    -- too, so the two arms agree about that row rather than differing.
    if not sc.parts then
      for tx = part_x0, out_x0 - 1 do
        surface.create_entity {
          name = belt_name, position = tile_center(tx, ty), direction = EAST,
          force = force, raise_built = true,
        }
      end
    end

    if row < mout then
      for tx = out_x0, out_x0 + 2 do
        surface.create_entity {
          name = belt_name, position = tile_center(tx, ty), direction = EAST,
          force = force, raise_built = true,
        }
      end
      surface.create_entity {
        name = loader_name, position = { x = out_x0 + 4, y = ty + 0.5 }, direction = EAST,
        type = "input", force = force, raise_built = true,
      }
      sinks[#sinks + 1] = surface.create_entity {
        name = "steel-chest", position = tile_center(out_x0 + 5, ty), force = force,
        raise_built = true,
      }
    end
  end

  if sc.parts then
    for px = 0, pc - 1 do
      for py = 0, h - 1 do
        surface.create_entity {
          name = spec.part_name, position = tile_center(part_x0 + px, oy + py),
          force = force, raise_built = true,
        }
      end
    end
  end

  return sinks, part_x0
end

-- The shipped synchronous "drain the deferred queue and re-classify now"
-- trigger. It destroys itself, and it is the only thing that makes a network
-- exist inside `--create`, which never reaches a tick.
local function audit_marker(surface, force, x, y)
  if not prototypes.entity["bbb-audit"] then return false end
  surface.create_entity {
    name = "bbb-audit", position = { x + 0.5, y + 0.5 }, force = force, raise_built = true,
  }
  return true
end

--------------------------------------------------------------------------------
-- init
--------------------------------------------------------------------------------

local function bench_init()
  local spec = {
    scenario = cfg.scenario, n = cfg.n, k = cfg.k, tier = cfg.tier, item = cfg.item,
    part_name = cfg.part_name or "balancer-part",
  }
  assert(TIERS[spec.tier], "bbb-bench: unknown tier " .. tostring(spec.tier))
  assert(SCENARIOS[spec.scenario], "bbb-bench: unknown scenario " .. tostring(spec.scenario))
  -- A scenario that places parts against a mod that does not define that
  -- prototype fails here rather than at create_entity, 16 x n times over.
  if SCENARIOS[spec.scenario].parts then
    assert(prototypes.entity[spec.part_name],
      "bbb-bench: no such entity prototype: " .. tostring(spec.part_name)
      .. " (is the balancer mod enabled?)")
  end

  local w, h = rig_width(spec.k), rig_height(spec.k)
  local pitch_x, pitch_y = w + PAD, h + PAD
  local cols = math.ceil(math.sqrt(spec.n))
  local rows = math.ceil(spec.n / cols)

  local surface = make_surface(cols * pitch_x, rows * pitch_y)
  local force = game.forces.player

  -- Nothing here should ever be attacked, polluted or evolved.
  game.map_settings.pollution.enabled = false
  game.map_settings.enemy_evolution.enabled = false
  game.map_settings.enemy_expansion.enabled = false
  for _, s in pairs(game.surfaces) do
    for _, e in pairs(s.find_entities_filtered { force = "enemy" }) do e.destroy() end
  end

  storage.sinks = {}
  storage.spec = spec
  storage.totals = {}
  for i = 1, spec.k do storage.totals[i] = 0 end

  for i = 0, spec.n - 1 do
    local ox = (i % cols) * pitch_x
    local oy = math.floor(i / cols) * pitch_y
    local sinks = build_rig(surface, force, ox, oy, spec)
    storage.sinks[i + 1] = sinks
  end

  -- BetterBeltBalancer batches its recompiles onto the following tick
  -- (`fk.defer`), and `--create` never reaches a tick -- so without this every
  -- network in the save would be compiled on the FIRST TICK OF THE BENCHMARK,
  -- which is the one measurement this harness exists to keep clean. `bbb-audit`
  -- is that mod's own synchronous "drain and re-classify now" marker; it
  -- destroys itself. Guarded by the prototype's existence, so the bb2, bb3 and
  -- no-mod cells are untouched.
  if prototypes.entity["bbb-audit"] then
    surface.create_entity {
      name = "bbb-audit", position = { -40.5, -40.5 }, force = force, raise_built = true,
    }
  end

  log(string.format(
    "BENCH-SETUP scenario=%s n=%d k=%d tier=%s item=%s part=%s rigs_built=%d surface=%dx%d",
    spec.scenario, spec.n, spec.k, spec.tier, spec.item, spec.part_name,
    #storage.sinks, cols * pitch_x, rows * pitch_y))
end

-- The floor the mega scenario exists to clear. A save that does not carry a
-- megabase's worth of hidden network is not a megabase measurement, and a
-- silently-shrunk population would look like a performance win.
local MEGA_MIN_SPLITTERS = 3000

local function mega_init()
  local spec = {
    scenario = cfg.scenario, n = cfg.n, k = cfg.k, tier = cfg.tier, item = cfg.item,
    part_name = cfg.part_name or "balancer-part",
  }
  assert(TIERS[spec.tier], "bbb-bench: unknown tier " .. tostring(spec.tier))
  local sc = SCENARIOS[spec.scenario]
  if sc.parts then
    assert(prototypes.entity[spec.part_name],
      "bbb-bench: no such entity prototype: " .. tostring(spec.part_name)
      .. " (is the balancer mod enabled?)")
  end

  local nblocks = spec.n
  assert(nblocks >= 1, "bbb-bench: mega needs at least one block")

  -- Block metrics: the ten small shapes stacked vertically, PAD between them.
  local block_h, block_w = 0, 0
  for _, s in ipairs(MEGA_BLOCK) do
    block_h = block_h + shape_height(s) + PAD
    block_w = math.max(block_w, shape_width(s))
  end
  local pitch_x = block_w + PAD
  local bcols = math.ceil(math.sqrt(nblocks))
  local brows = math.ceil(nblocks / bcols)

  -- The giants get their own column to the right of the block grid: they are
  -- 16 to 65 rows tall and would otherwise dictate every cell's height.
  local giant_h, giant_w = 0, 0
  for _, s in ipairs(MEGA_GIANTS) do
    giant_h = giant_h + shape_height(s) + PAD
    giant_w = math.max(giant_w, shape_width(s))
  end

  local width  = bcols * pitch_x + giant_w + PAD
  local height = math.max(brows * block_h, giant_h)

  local surface = make_surface(width, height)
  local force = game.forces.player

  game.map_settings.pollution.enabled = false
  game.map_settings.enemy_evolution.enabled = false
  game.map_settings.enemy_expansion.enabled = false
  for _, s in pairs(game.surfaces) do
    for _, e in pairs(s.find_entities_filtered { force = "enemy" }) do e.destroy() end
  end

  storage.spec = spec
  storage.rigs = {}
  storage.mega = true

  local counts = {}
  local order = {}
  local function add(shape, ox, oy)
    local sinks, part_x0 = build_mega_rig(surface, force, ox, oy, shape, spec)
    local totals = {}
    for i = 1, #sinks do totals[i] = 0 end
    storage.rigs[#storage.rigs + 1] = {
      class = shape.name, sinks = sinks, totals = totals,
      -- Which rigs the worst-balance search is allowed to look at.
      --
      -- The over-limit cluster is REFUSED, so it has no network and delivers
      -- nothing; counting it would read as a broken balancer. And in the
      -- CONTROL scenarios there is no balancer at all -- a 3->5's rows 3 and 4
      -- have no source and physically cannot be fed by a straight belt -- so
      -- only the square shapes are asked to be even there. Every rig still
      -- reports its own class aggregate on a BENCH-SHAPE line either way.
      deliver = (not shape.over_limit) and (sc.parts or shape.n == shape.m),
    }
    if counts[shape.name] == nil then counts[shape.name] = 0; order[#order + 1] = shape.name end
    counts[shape.name] = counts[shape.name] + 1
    return #storage.rigs, part_x0
  end

  for b = 0, nblocks - 1 do
    local ox = (b % bcols) * pitch_x
    local oy = math.floor(b / bcols) * block_h
    local y = oy
    for _, s in ipairs(MEGA_BLOCK) do
      add(s, ox, y)
      y = y + shape_height(s) + PAD
    end
  end

  local gx, gy = bcols * pitch_x, 0
  local deferred = nil
  for _, s in ipairs(MEGA_GIANTS) do
    if s.defer then
      deferred = { shape = s, ox = gx, oy = gy }
    else
      add(s, gx, gy)
    end
    gy = gy + shape_height(s) + PAD
  end

  -- Everything except the 64x64 is standing. Flush it, then time an audit with
  -- NOTHING pending, then build the 64x64 and time the audit that compiles it.
  -- The difference is the first compile of a P=64 network, measured the same
  -- way M2 measures a recompile: two probes and a subtraction, never one
  -- number on its own. (The audit re-classifies every cluster in the save, so
  -- the control is not small and cannot be skipped.)
  local mx, my = -40, -40
  local audited = audit_marker(surface, force, mx, my)
  if audited then
    local p = helpers.create_profiler()
    audit_marker(surface, force, mx, my)
    p.stop()
    log { "", "[BENCH-MEGA] timing audit only, nothing pending ", p }
  end

  local gi, gpx = add(deferred.shape, deferred.ox, deferred.oy)
  if audited then
    local q = helpers.create_profiler()
    audit_marker(surface, force, mx, my)
    q.stop()
    log { "", "[BENCH-MEGA] timing audit + FIRST COMPILE of the 64x64 ", q }
  end

  -- Where the hitch schedule reaches to remove and restore one input belt of
  -- the 64x64: the tile immediately west of the top-left part.
  storage.hitch = { surface = surface.index, x = gpx - 1, y = deferred.oy, rig = gi }

  for _, name in ipairs(order) do
    log(string.format("BENCH-MEGA-SHAPE class=%s rigs=%d", name, counts[name]))
  end

  -- The floor. `find_entities_filtered` with no area walks the whole surface,
  -- which is what we want: the hidden surface holds nothing else.
  local nsplit, nlane = 0, 0
  local hid = game.surfaces["bbb-hidden"]
  if hid then
    nsplit = #hid.find_entities_filtered { name = "bbb-splitter" }
    nlane  = #hid.find_entities_filtered { name = "bbb-lane-splitter" }
  end
  log(string.format(
    "BENCH-MEGA blocks=%d rigs=%d hidden_splitters=%d (bbb-splitter=%d bbb-lane-splitter=%d)",
    nblocks, #storage.rigs, nsplit + nlane, nsplit, nlane))
  if sc.parts and hid and nsplit + nlane < MEGA_MIN_SPLITTERS then
    error(string.format(
      "bbb-bench: mega built only %d hidden splitters, floor is %d -- raise --n",
      nsplit + nlane, MEGA_MIN_SPLITTERS))
  end

  log(string.format(
    "BENCH-SETUP scenario=%s n=%d k=%d tier=%s item=%s part=%s rigs_built=%d surface=%dx%d",
    spec.scenario, spec.n, spec.k, spec.tier, spec.item, spec.part_name,
    #storage.rigs, width, height))
end

--------------------------------------------------------------------------------
-- metering: throughput + balance sanity check
--------------------------------------------------------------------------------

local function sum_of(t)
  local s = 0
  for _, v in ipairs(t) do s = s + v end
  return s
end

-- Drains every sink chest and accumulates per-output-index totals. Runs rarely
-- (default every 600 ticks) so its cost is well under a microsecond per tick
-- amortised, and it is identical across scenarios so it cancels in any delta.
-- It also keeps the sinks from backing up, which is what keeps the rigs
-- saturated.
local function meter(e)
  local totals = storage.totals
  local window = 0
  for _, sinks in ipairs(storage.sinks) do
    for idx, chest in ipairs(sinks) do
      if chest.valid then
        local inv = chest.get_inventory(defines.inventory.chest)
        local n = inv.get_item_count()
        if n > 0 then
          totals[idx] = totals[idx] + n
          window = window + n
          inv.clear()
        end
      end
    end
  end

  local parts = {}
  for i = 1, storage.spec.k do parts[i] = tostring(totals[i]) end
  log(string.format("BENCH-METER tick=%d window=%d cumulative=%d per_output=%s",
    e.tick, window, sum_of(totals), table.concat(parts, ",")))
end

-- The mega population has a different number of outputs per rig, so there is no
-- single per-output-index vector to report. What goes on the BENCH-METER line
-- instead is the per-output vector of the WORST-balanced rig in the save, which
-- makes run.sh's own max/min the worst per-rig balance anywhere -- a strictly
-- sharper gate than the uniform case's, and it needs no change over there.
-- Per-class aggregates go on their own BENCH-SHAPE lines for the write-up.
local function mega_meter(e)
  local window, grand = 0, 0
  local classes, order = {}, {}
  local worst_ratio, worst_vec, worst_class = 0, nil, nil

  for _, rig in ipairs(storage.rigs) do
    local t = rig.totals
    for idx, chest in ipairs(rig.sinks) do
      if chest.valid then
        local inv = chest.get_inventory(defines.inventory.chest)
        local n = inv.get_item_count()
        if n > 0 then
          t[idx] = t[idx] + n
          window = window + n
          inv.clear()
        end
      end
    end

    local c = classes[rig.class]
    if not c then
      c = { outs = {}, rigs = 0, total = 0, delivered = rig.deliver }
      classes[rig.class] = c
      order[#order + 1] = rig.class
    end
    c.rigs = c.rigs + 1
    local mn, mx = nil, 0
    for idx, v in ipairs(t) do
      c.outs[idx] = (c.outs[idx] or 0) + v
      c.total = c.total + v
      grand = grand + v
      if mn == nil or v < mn then mn = v end
      if v > mx then mx = v end
    end
    if rig.deliver and mn ~= nil then
      local r = (mn > 0) and (mx / mn) or 999
      if r > worst_ratio then worst_ratio, worst_vec, worst_class = r, t, rig.class end
    end
  end

  for _, name in ipairs(order) do
    local c = classes[name]
    local mn, mx = nil, 0
    for _, v in ipairs(c.outs) do
      if mn == nil or v < mn then mn = v end
      if v > mx then mx = v end
    end
    log(string.format(
      "BENCH-SHAPE tick=%d class=%s rigs=%d outputs=%d total=%d min=%d max=%d balance=%.4f",
      e.tick, name, c.rigs, #c.outs, c.total, mn or 0, mx,
      (mn and mn > 0) and (mx / mn) or 0))
  end

  -- No rig has delivered anything yet (the idle scenarios, or the first sample
  -- of a saturated run): report a single zero rather than an empty field, which
  -- would make run.sh's balance parser read 999.
  local vec = { "0" }
  if worst_vec and worst_ratio > 0 then
    vec = {}
    for i, v in ipairs(worst_vec) do vec[i] = tostring(v) end
  end
  log(string.format(
    "BENCH-METER tick=%d window=%d cumulative=%d per_output=%s worst_class=%s",
    e.tick, window, grand, table.concat(vec, ","), worst_class or "none"))
end

--------------------------------------------------------------------------------
-- the 64x64 recompile hitch (--hitch only)
--------------------------------------------------------------------------------
--
-- M2's tick-pair pattern, verbatim in shape: the guest defers its recompile to
-- the following tick, so the profiler is opened in the tick that MUTATES and
-- closed in the tick that FLUSHES. Each window therefore contains one whole
-- engine tick as well as the recompile, and the `idle tick pair` probe measures
-- exactly that and nothing else. SUBTRACT IT.
--
-- Three reps, at ticks that are not multiples of the 600-tick meter interval --
-- a meter sample inside a profiler window would be measured as recompile.

local HITCH_AT = { 1210, 1510, 1810 }

-- Module-level, not `storage`: a profiler is a live handle and one benchmark
-- run is one session, so there is nothing to carry across a save.
local hitch = {}

local function hitch_belt(h)
  local s = game.surfaces[h.surface]
  return s.find_entities_filtered { position = { h.x + 0.5, h.y + 0.5 },
                                    type = "transport-belt" }[1], s
end

local function hitch_tick(e)
  local h = storage.hitch
  if not h then return end
  local t = e.tick
  for _, base in ipairs(HITCH_AT) do
    if t == base - 2 then
      hitch.p = helpers.create_profiler()
      hitch.label = "idle tick pair, nothing pending"
      return
    elseif t == base or t == base + 2 or t == base + 4 then
      if hitch.p then
        hitch.p.stop()
        log { "", "[BENCH-MEGA] hitch " .. hitch.label .. " ", hitch.p }
        hitch.p = nil
      end
      if t == base then
        local belt = hitch_belt(h)
        if not belt then log("[BENCH-MEGA] hitch: no input belt at the 64x64") return end
        hitch.p = helpers.create_profiler()
        hitch.label = "64x64 teardown+rebuild(-1 input)"
        belt.destroy { raise_destroy = true }
      elseif t == base + 2 then
        local s = game.surfaces[h.surface]
        hitch.p = helpers.create_profiler()
        hitch.label = "64x64 teardown+rebuild(full)"
        s.create_entity { name = TIERS[cfg.tier].belt,
                          position = { h.x + 0.5, h.y + 0.5 }, direction = EAST,
                          force = game.forces.player, raise_built = true }
      end
      return
    end
  end
end

local function register_meter()
  if cfg.meter_interval and cfg.meter_interval > 0 then
    local m = (SCENARIOS[cfg.scenario] and SCENARIOS[cfg.scenario].mega) and mega_meter or meter
    script.on_nth_tick(cfg.meter_interval, m)
  end
  if cfg.hitch then script.on_event(defines.events.on_tick, hitch_tick) end
end

script.on_init(function()
  if SCENARIOS[cfg.scenario] and SCENARIOS[cfg.scenario].mega then
    mega_init()
  else
    bench_init()
  end
  register_meter()
end)
script.on_load(register_meter)
