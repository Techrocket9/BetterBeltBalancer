-- bbb-mig-test: builds a Belt Balancer 2 shaped world in phase one and reports,
-- in phase two, what survived the swap.
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. test/assert-mig.py decides
-- whether the numbers are right; a test mod that computed the expected answer
-- would be a second implementation of the thing under test.
--
-- IT IS PRESENT IN BOTH PHASES AND BETTER BELT BALANCER IS NOT. That is the
-- shape of the suite: `on_init` runs once, in the phase where the incumbent
-- owns `balancer-part`, and everything after it runs on the far side of a mod
-- swap. So nothing in `on_init` may name a Better Belt Balancer prototype, and
-- everything that does is guarded on the prototype existing.
--
-- THE RIGS, one per band on a flat scratch surface, every part placed as
-- `balancer-part` because in phase one that is the only balancer prototype in
-- the game:
--
--   ctrl    a bare express belt, chest to chest. The yardstick: whatever this
--           delivers in the sample window is what one saturated belt is worth,
--           so "full throughput" is a comparison against the engine rather than
--           against a number worked out on paper.
--   m4x4    4 parts, 4 belts in, 4 belts out, saturated. The shape a migrating
--           player is most likely to have.
--   m3to5   5 parts, 3 in and 5 out: N != M, P=8, spare ports looped back. It
--           is here because adoption re-derives the edge list from the world,
--           and an asymmetric shape is where a wrong edge list shows up as a
--           rate rather than as a crash.
--   wit     THE CONSERVATION WITNESS. 2 parts, 2 in and 2 out, and NO source
--           and NO sink at all -- its belts are hand-loaded with COPPER PLATE
--           and nothing can add to them or take from them. Every other rig in
--           this save runs iron plate, so a count of copper across every
--           surface is exactly this rig's contents, before the swap and after
--           it. That is what makes "the items on the belts survived" an
--           equality rather than an estimate.
--   fid     THE FIDELITY RIG. 2 parts, belts either side, and nothing feeding
--           it: what it measures is not a rate but what the conversion CARRIED.
--           One part is DAMAGED and the other is built at UNCOMMON quality, and
--           those are the only two properties `legacyConvertOne` reads off the
--           old entity and writes onto the new one. Until this rig existed every
--           part in every leg was undamaged and normal, so both lines could have
--           been absent and every assertion in this suite would still have
--           passed.
--   frc     THE FORCE RIG, and it is the SHARP case rather than a fourth copy of
--           the others: FOUR PARTS IN ONE COLUMN, the top two on the player
--           force and the bottom two on a second force, TOUCHING. Clusters are
--           per force, so that is two balancers; a flood fill that lost its
--           force check fuses them into one, silently, with every item count in
--           this suite unmoved. It is also the only rig that makes the
--           conversion grant the technology to more than one force.
--
-- And a SECOND SURFACE, `bbb-mig-b`, carrying two more parts and their belts.
-- `legacyScan` walks every surface in index order; until that rig existed every
-- part in this suite stood on one surface, so a scan that stopped after the
-- first surface it converted anything on would have passed every leg.
--
-- Plus a steel chest holding a stack of the incumbent's ITEM, which is the
-- other half of what a removed mod takes with it.

local LEGACY_PART = "balancer-part"
local BBB_PART = "bbb-balancer-part"
local AUDIT = "bbb-audit"
local BELT = "express-transport-belt"
local LOADER = "bbbmig-loader"
local WITNESS_ITEM = "copper-plate"
local FLOW_ITEM = "iron-plate"
local E = defines.direction.east

local SURF = "bbb-mig-a"
local SURF_B = "bbb-mig-b"
local PITCH = 12
local HALFX = 20

-- The second force, and the list of forces every per-force report walks. An
-- ipairs over a written-down order rather than a pairs() over `game.forces`,
-- which is a hash: nothing here is host-visible, but a report line whose columns
-- swapped between runs would be read by an assertion.
local FORCE_B = "bbb-mig-force-b"
local FORCES = { "player", FORCE_B }

-- What the fidelity rig is built with. The health is a real wound rather than a
-- round number for its own sake -- it has to be BELOW max_health, or an equality
-- across the swap is satisfied by a guest that copies nothing at all.
local FID_HEALTH = 85
local FID_QUALITY = "uncommon"

--------------------------------------------------------------------------------
-- surface and pieces
--------------------------------------------------------------------------------

local function P(x, y) return { x + 0.5, y + 0.5 } end

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

  local area = { { -HALFX, -8 }, { HALFX, rows + 8 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -HALFX, HALFX do
    for y = -8, rows + 8 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  s.set_tiles(tiles, true, false, false, false)
  return s
end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-mig-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

local function belts(s, dir, from, to, y)
  for x = from, to do put(s, BELT, x, y, { direction = dir }) end
end

local function source(s, x, y)
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = "player" }
  c.infinity_container_filters = {
    { index = 1, name = FLOW_ITEM, count = 1000, mode = "at-least" },
  }
  put(s, LOADER, x + 1, y, { direction = E, type = "output" })
  return c
end

local function sink(s, x, y)
  put(s, LOADER, x, y, { direction = E, type = "input" })
  return s.create_entity { name = "steel-chest", position = P(x + 1, y), force = "player" }
end

local function chest_count(c)
  if not (c and c.valid) then return -1 end
  local total = 0
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
    total = total + item.count
  end
  return total
end

--------------------------------------------------------------------------------
-- counting
--
-- Over EVERY surface, so the hidden one this mod's compiler works on is included
-- the moment it exists -- and it does not exist at all in the phase where the
-- incumbent is installed, which is exactly why the count has to be written this
-- way rather than against a named pair of surfaces.
--------------------------------------------------------------------------------

local function count_item(name)
  local total = 0
  for _, s in pairs(game.surfaces) do
    for _, e in pairs(s.find_entities_filtered { force = "player" }) do
      if e.valid then
        if e.type == "item-entity" then
          if e.stack and e.stack.valid_for_read and e.stack.name == name then
            total = total + e.stack.count
          end
        else
          local ok, n = pcall(function() return e.get_max_transport_line_index() end)
          if ok and n then
            for i = 1, n do total = total + e.get_transport_line(i).get_item_count(name) end
          end
          local inv = e.get_inventory(defines.inventory.chest)
          if inv then total = total + inv.get_item_count(name) end
        end
      end
    end
  end
  return total
end

-- How many of each balancer prototype are standing, anywhere. The migration's
-- own headline in one line: `balancer-part` must reach zero and
-- `bbb-balancer-part` must reach the number that were there.
-- `find_entities_filtered{name = ...}` RAISES for a prototype that does not
-- exist, so both names are guarded: in phase one of the added-as-removed leg
-- `bbb-balancer-part` is not a prototype at all, and after the swap
-- `balancer-part` is only a prototype because this mod's data stage kept it
-- alive.
local function count_named(name)
  if not prototypes.entity[name] then return 0 end
  local n = 0
  for _, s in pairs(game.surfaces) do
    n = n + #s.find_entities_filtered { name = name }
  end
  return n
end

-- Every surface in the game, in INDEX order. `game.surfaces` is a hash and the
-- lines below it are read by an assertion, so the walk is sorted rather than
-- left to pairs() -- the same habit `collectSurfaces` follows in the guest, and
-- there for a much harder reason (surface order decides node ids, which decide
-- slots, which is a desync).
local function surfaces_in_order()
  local list = {}
  for _, s in pairs(game.surfaces) do list[#list + 1] = s end
  table.sort(list, function(a, b) return a.index < b.index end)
  return list
end

-- Guarded on the prototype for the reason `count_named` is: in phase one of the
-- swap leg `bbb-balancer-part` is not a prototype at all, and after the swap
-- `balancer-part` is only a prototype because this mod's data stage kept it
-- alive. `find_entities_filtered{name = ...}` RAISES for a name the game does
-- not have.
local function count_named_on(s, name, force)
  if not prototypes.entity[name] then return 0 end
  local f = { name = name }
  if force then f.force = force end
  return #s.find_entities_filtered(f)
end

-- WHERE the parts are, which the whole-world census cannot say. The summary line
-- the guest writes counts surfaces SCANNED rather than surfaces that had
-- anything on them, so a scan that stopped after the first surface it converted
-- something on would report the same number it reports now. This is the line
-- that can see it.
local function report_surfaces(phase)
  local out = {}
  for _, s in ipairs(surfaces_in_order()) do
    out[#out + 1] = string.format("%s:%d/%d", s.name,
      count_named_on(s, LEGACY_PART), count_named_on(s, BBB_PART))
  end
  log(string.format("[BBB-MIG] surfaces phase=%s %s", phase, table.concat(out, " ")))
end

-- WHOSE the parts are. The anti-vacuity half of the force rig: a run where the
-- second force's parts were never built would satisfy "two forces stayed two
-- clusters" by having only one force in it.
local function report_force_parts(phase)
  local out = {}
  for _, fname in ipairs(FORCES) do
    local n = 0
    for _, s in ipairs(surfaces_in_order()) do
      n = n + count_named_on(s, LEGACY_PART, fname) + count_named_on(s, BBB_PART, fname)
    end
    out[#out + 1] = string.format("%s=%d", fname, n)
  end
  log(string.format("[BBB-MIG] force-parts phase=%s %s", phase, table.concat(out, " ")))
end

-- The part standing on one tile, whichever prototype it is now. Both names are
-- guarded, and the name is REPORTED rather than assumed: which of the two is
-- there is exactly what the fidelity rig's tiles are being asked.
local function part_at(x, y)
  local s = game.surfaces[SURF]
  local area = { { x + 0.1, y + 0.1 }, { x + 0.9, y + 0.9 } }
  for _, name in ipairs { LEGACY_PART, BBB_PART } do
    if prototypes.entity[name] then
      local es = s.find_entities_filtered { name = name, area = area }
      if es[1] and es[1].valid then return es[1], name end
    end
  end
  return nil, "none"
end

-- THE TWO PROPERTIES THE CONVERSION CARRIES. `legacyConvertOne` reads the health
-- and the quality off the entity it is about to destroy and writes them onto the
-- one it creates; both are one call each and both are invisible on an undamaged
-- normal-quality part, which is what every other part in every leg is.
local function report_fidelity(phase)
  local t = storage.fid
  local e, name = part_at(t.health[1], t.health[2])
  log(string.format("[BBB-MIG] health phase=%s name=%s value=%s max=%s", phase, name,
    (e and e.health) and string.format("%.1f", e.health) or "none",
    e and string.format("%.1f", e.max_health) or "none"))
  e, name = part_at(t.quality[1], t.quality[2])
  log(string.format("[BBB-MIG] quality phase=%s name=%s value=%s", phase, name,
    (e and e.quality) and e.quality.name or "none"))
end

local function census(phase)
  log(string.format("[BBB-MIG] census phase=%s %s=%d %s=%d",
    phase, LEGACY_PART, count_named(LEGACY_PART), BBB_PART, count_named(BBB_PART)))
end

-- The item half. A stack of the incumbent's item in a chest is deleted with its
-- prototype exactly as an entity is, and `place_result` is what makes a
-- surviving stack useful rather than merely present.
local function report_item(phase)
  local held = -1
  local proto = prototypes.item[LEGACY_PART]
  local place = "nil"
  -- GUARDED ON THE PROTOTYPE, and not for tidiness: without the data stage's
  -- stub there is no `balancer-part` item at all after the swap, and
  -- `get_item_count` on a name the game does not have RAISES. That is the
  -- failure this suite is red-proved against, so it has to be reportable rather
  -- than a crash.
  if proto then
    local c = storage.item_chest
    if c and c.valid then held = c.get_item_count(LEGACY_PART) end
    if proto.place_result then place = proto.place_result.name end
  else
    held = 0
  end
  log(string.format("[BBB-MIG] legacy-item phase=%s held=%d place_result=%s",
    phase, held, place))
end

local function report_tech(phase)
  local f = game.forces["player"]
  local ours = f.technologies["bbb-balancer"]
  local theirs = f.technologies["belt-balancer-1"]
  log(string.format("[BBB-MIG] tech phase=%s bbb-balancer=%s belt-balancer-1=%s",
    phase,
    ours and tostring(ours.researched) or "absent",
    theirs and tostring(theirs.researched) or "absent"))
  -- THE SECOND FORCE, on a line of its own so the one above keeps the exact
  -- shape every assertion in this suite already reads. `legacyScan` grants the
  -- technology PER FORCE that owned a converted part, and a force left without
  -- it is a player holding balancers they cannot craft a spare for -- which is
  -- the thing the grant exists to prevent, one force further along than any leg
  -- could see before the force rig existed.
  local g = game.forces[FORCE_B]
  local gours = g and g.technologies["bbb-balancer"]
  log(string.format("[BBB-MIG] tech-force phase=%s force=%s bbb-balancer=%s",
    phase, FORCE_B, gours and tostring(gours.researched) or "absent"))
end

local function report_counts(phase)
  log(string.format("[BBB-MIG] count phase=%s %s=%d", phase, WITNESS_ITEM,
    count_item(WITNESS_ITEM)))
end

-- THE LATE BUILD: one `balancer-part` placed in phase two, well clear of every
-- rig, long after any scan has run.
--
-- It is the BUILD path rather than the scan, and it is the one probe that can
-- tell "this game's balancer-part is mine" apart from "somebody else's". A
-- migrated save must swap it; a save where a stranger owns the prototype must
-- leave it exactly where it is. Placed after the final audit so no count, no
-- rate and no audit in this suite moves because of it.
local LATE_X, LATE_Y = 12, 0

local function late_build()
  if not prototypes.entity[LEGACY_PART] then
    log("[BBB-MIG] late-build unavailable")
    return
  end
  local e = game.surfaces[SURF].create_entity {
    name = LEGACY_PART, position = P(LATE_X, LATE_Y), force = "player", raise_built = true,
  }
  log("[BBB-MIG] late-build placed=" .. tostring(e ~= nil))
end

local function late_probe()
  local s = game.surfaces[SURF]
  local area = { { LATE_X - 0.1, LATE_Y - 0.1 }, { LATE_X + 1.1, LATE_Y + 1.1 } }
  local legacy, ours = 0, 0
  if prototypes.entity[LEGACY_PART] then
    legacy = #s.find_entities_filtered { name = LEGACY_PART, area = area }
  end
  if prototypes.entity[BBB_PART] then
    ours = #s.find_entities_filtered { name = BBB_PART, area = area }
  end
  log(string.format("[BBB-MIG] late-build legacy=%d ours=%d", legacy, ours))
end

local function audit_now()
  if not prototypes.entity[AUDIT] then
    log("[BBB-MIG] audit unavailable (this mod is not installed in this phase)")
    return
  end
  game.surfaces[SURF].create_entity {
    name = AUDIT, position = P(-16, 0), force = "player", raise_built = true,
  }
end

--------------------------------------------------------------------------------
-- rigs
--------------------------------------------------------------------------------

local RIGS = {
  { name = "ctrl" },
  { name = "m4x4", parts = 4, ins = 4, outs = 4 },
  { name = "m3to5", parts = 5, ins = 3, outs = 5 },
}

local function build_rig(cfg, base)
  local s = game.surfaces[SURF]
  if cfg.name == "ctrl" then
    source(s, -5, base)
    belts(s, E, -3, 3, base)
    return { sink(s, 4, base) }
  end
  -- Parts first, belts after: the belt-adjacency trigger is then on the
  -- critical path of every rig rather than only of some.
  for i = 0, cfg.parts - 1 do put(s, LEGACY_PART, 0, base + i) end
  for i = 0, cfg.ins - 1 do
    source(s, -5, base + i)
    belts(s, E, -3, -1, base + i)
  end
  local out = {}
  for i = 0, cfg.outs - 1 do
    belts(s, E, 1, 3, base + i)
    out[#out + 1] = sink(s, 4, base + i)
  end
  return out
end

-- THE WITNESS. Two parts, two belts in and two out, nothing feeding it and
-- nothing draining it, hand-loaded with copper plate. `insert_at` is used rather
-- than `insert_at_back` for the reason this mod's own carry path uses it: the
-- back of a line is ONE position and accepts one item per tick, where a named
-- position fills the line the way a compressed belt is filled.
local function build_witness(base)
  local s = game.surfaces[SURF]
  for i = 0, 1 do put(s, LEGACY_PART, 0, base + i) end
  local loaded = 0
  for i = 0, 1 do
    for x = -3, -1 do
      local b = put(s, BELT, x, base + i, { direction = E })
      for lane = 1, 2 do
        local line = b.get_transport_line(lane)
        for k = 0, 3 do
          if line.insert_at(k * 0.25, { name = WITNESS_ITEM, count = 1 }) then
            loaded = loaded + 1
          end
        end
      end
    end
    for x = 1, 3 do put(s, BELT, x, base + i, { direction = E }) end
  end
  log(string.format("[BBB-MIG] witness loaded=%d", loaded))
end

-- THE FIDELITY RIG. Two parts and four belts, nothing feeding it: what is
-- measured here is not a rate. One part is DAMAGED and the other is created at
-- UNCOMMON quality, so the two properties `legacyConvertOne` carries across the
-- swap are both non-default on exactly one tile each, and the tiles are written
-- into `storage` rather than into a constant -- the rig bases are computed.
--
-- IT HAS BELTS EITHER SIDE and it needs them: a cluster with no inputs or no
-- outputs is a legitimate half-built state that compiles to nothing, and the
-- audit's `nets == clusters` would then read as a cluster the classifier never
-- saw. Every rig this suite adds carries one belt in and one belt out per part
-- for that reason, fed or not.
local function build_fidelity(base)
  local s = game.surfaces[SURF]
  local hurt = put(s, LEGACY_PART, 0, base)
  hurt.health = FID_HEALTH
  -- Guarded on the quality existing rather than on the mod list: `mig_list`
  -- enables the quality mod for both phases of every leg, and a guard is what
  -- turns a mod list that stopped doing so into a failed assertion instead of a
  -- failed script.
  local extra = nil
  if prototypes.quality and prototypes.quality[FID_QUALITY] then
    extra = { quality = FID_QUALITY }
  end
  put(s, LEGACY_PART, 0, base + 1, extra)
  for i = 0, 1 do
    put(s, BELT, -1, base + i, { direction = E })
    put(s, BELT, 1, base + i, { direction = E })
  end
  storage.fid = { health = { 0, base }, quality = { 0, base + 1 } }
end

-- THE FORCE RIG. Four parts in one column, the top two on the player force and
-- the bottom two on a second force, so the pair in the middle TOUCHES across a
-- force boundary. Two forces' parts touching are two balancers -- the flood
-- fill, the compiler's own fill and the edge search all agree about it -- and a
-- fusion is invisible to every count in this suite: the same parts, the same
-- items, one fewer cluster.
local function build_forces(base)
  local s = game.surfaces[SURF]
  for i = 0, 1 do
    put(s, LEGACY_PART, 0, base + i)
    put(s, BELT, -1, base + i, { direction = E })
    put(s, BELT, 1, base + i, { direction = E })
  end
  for i = 2, 3 do
    put(s, LEGACY_PART, 0, base + i, { force = FORCE_B })
    put(s, BELT, -1, base + i, { direction = E, force = FORCE_B })
    put(s, BELT, 1, base + i, { direction = E, force = FORCE_B })
  end
end

-- THE SECOND SURFACE. `legacyScan` walks every surface in index order and the
-- summary line it writes counts surfaces SCANNED, so a scan that stopped after
-- the first surface it converted something on would report the number it reports
-- now and leave these two parts standing as the incumbent's forever.
local function build_surface_b()
  local s = make_surface(SURF_B, 4)
  for i = 0, 1 do
    put(s, LEGACY_PART, 0, i)
    put(s, BELT, -1, i, { direction = E })
    put(s, BELT, 1, i, { direction = E })
  end
end

--------------------------------------------------------------------------------
-- the schedule
--------------------------------------------------------------------------------

local function report(tick)
  local parts = {}
  for _, name in ipairs(storage.order) do
    local per = {}
    for _, c in ipairs(storage.rigs[name]) do per[#per + 1] = chest_count(c) end
    parts[#parts + 1] = name .. "=" .. table.concat(per, ",")
  end
  log(string.format("[BBB-MIG] sample tick=%d %s", tick, table.concat(parts, " ")))
end

local SCHEDULE = {
  -- The opening statement of phase two, taken as early as a script can: one
  -- tick of belt movement after a load, and the conversion (if it happened)
  -- happened before the first tick, at on_init or at
  -- on_configuration_changed.
  [1] = function()
    census("t1")
    report_counts("t1")
    report_item("t1")
    report_tech("t1")
    report_fidelity("t1")
    report_surfaces("t1")
    report_force_parts("t1")
  end,
  [60] = function()
    audit_now()
    census("post-audit")
    report_counts("post-audit")
  end,
  [1800] = function() report(1800) end,
  [3540] = function() report(3540) end,
  [3560] = function()
    audit_now()
    census("final")
    report_counts("final")
    report_tech("final")
    report_fidelity("final")
    report_surfaces("final")
    report_force_parts("final")
  end,
  -- After everything else, so nothing above can see it. The swap is deferred by
  -- one tick like every other build, hence the gap before the probe.
  [3570] = late_build,
  [3580] = late_probe,
}

script.on_init(function()
  local base, order = 0, {}
  for _, cfg in ipairs(RIGS) do
    cfg.base = base
    base = base + PITCH
    order[#order + 1] = cfg.name
  end
  local wit_base = base
  local fid_base = wit_base + PITCH
  local frc_base = fid_base + PITCH
  make_surface(SURF, frc_base + PITCH)

  -- BEFORE ANY PART OF ITS OWN IS PLACED, so `game.forces[FORCE_B]` is there for
  -- the whole life of the save and every per-force report can name it without a
  -- guard.
  game.create_force(FORCE_B)

  storage.rigs = {}
  storage.order = order
  for _, cfg in ipairs(RIGS) do
    storage.rigs[cfg.name] = build_rig(cfg, cfg.base)
  end
  build_witness(wit_base)
  build_fidelity(fid_base)
  build_forces(frc_base)
  -- LAST, so the surface indices are fixed and deterministic: nauvis 1,
  -- bbb-mig-a 2, bbb-mig-b 3, and the hidden surface -- when it exists at all --
  -- after them.
  build_surface_b()

  -- The other half of what a removed mod takes with it.
  local s = game.surfaces[SURF]
  local c = s.create_entity {
    name = "steel-chest", position = P(10, wit_base + PITCH), force = "player",
  }
  c.insert { name = LEGACY_PART, count = 50 }
  storage.item_chest = c

  census("create")
  report_counts("create")
  report_item("create")
  report_tech("create")
  report_fidelity("create")
  report_surfaces("create")
  report_force_parts("create")
  -- Only ever available in the phase where this mod IS installed, which is the
  -- coexistence leg's phase one. It is what makes that leg's "nothing was
  -- converted" a statement about a guest that was asked to look.
  audit_now()
  log(string.format("[BBB-MIG] init complete: %d rigs plus the witness, the "
    .. "fidelity pair, the force column and a second surface", #RIGS))
end)

script.on_event(defines.events.on_tick, function(e)
  local f = SCHEDULE[e.tick]
  if f then f() end
end)
