-- bbb-sedge-test: every balancer here is built to Factorio 2.1's rule, ONE BELT
-- PER BALANCER PART, and three rigs exist to break it in the three ways an edit
-- can.
--
-- WHY THE RULE. Every edge of a cluster is an interface linked belt standing on
-- the cluster's own tile, so a part carrying an input on one side and an output
-- on another carried TWO belt-connectables on one tile. 2.1 closed the
-- collision-mask loophole that permitted it, and the engine's own reason is that
-- belt-to-belt connections are re-derived at load rather than saved -- one
-- belt-connectable per tile is what makes that unambiguous. See
-- agents/single-edge.md and guest/go/sedge.go.
--
-- The consequence for a rig is geometry: a 4-in/4-out balancer is EIGHT parts
-- (a 4x2 block, four west parts carrying the inputs and four east parts
-- carrying the outputs), not four. The smallest balancer is two.
--
--   ctrl   a bare express belt, the yardstick -- so "full throughput" is a
--          comparison against the engine rather than arithmetic on a wiki
--   s11    1 -> 1 over TWO parts. P = 1, no butterfly stages at all
--   s22    2 -> 2 over FOUR parts (a 2x2 block). P = 2
--   s44    4 -> 4 over EIGHT parts (a 4x2 block). P = 4
--   s35    3 -> 5 over TEN parts (a 5x2 block, two west parts carrying nothing).
--          P = 8 with loopbacks -- the asymmetric shape, where a wrong edge list
--          reads as a rate rather than as a crash
--
-- And the three refusals, each a working balancer that an edit asks for a
-- second belt on one of its parts:
--
--   sbld   a second belt BUILT against an occupied part, by script. No player
--          index, so the force.print arm fires and nothing is handed back --
--          which is also the standing negative for the whole run
--   srot   a belt ROTATED onto an occupied part. `entity.direction = ...` raises
--          nothing at all, so the audit is what finds it -- and rotating it back
--          must be a SKIP, because the fingerprint is the one the netInfo never
--          lost
--   smrg   a part BRIDGING two working balancers into one whose bridging tile
--          would carry two belts. The teardowns belong to AddPart and are queued
--          before the compiler ever sees the cluster they make, so the merge
--          pre-pass is what has to refuse it -- with both standing networks
--          untouched
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. test/assert-sedge.py decides;
-- a test mod that computed the expected answer would be a second implementation
-- of the thing under test.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbs-loader"
local AUDIT = "bbb-audit"
local FLOW_ITEM = "iron-plate"
local E = defines.direction.east
local S = defines.direction.south

local SURF = "bbb-sedge"

-- Band bases. Far enough apart that no rig's belts are inside another rig's
-- two-tile neighbour gate.
local CTRL = 0
local S11 = 6
local S22 = 12         -- rows 12..13
local S44 = 20         -- rows 20..23
local S35 = 30         -- rows 30..34
local SBLD = 42        -- rows 42..43, and the refused belt at row 41
local SROT = 50        -- rows 50..51, and the rotated belt at row 49
local SMRG = 58        -- 58..59 = A, 60 = the gap, 61..62 = B
local ROWS = 72

--------------------------------------------------------------------------------
-- surface and pieces
--------------------------------------------------------------------------------

local function P(x, y) return { x + 0.5, y + 0.5 } end

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

  local area = { { -16, -12 }, { 16, ROWS + 8 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -16, 16 do
    for y = -12, ROWS + 8 do
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
    error(string.format("bbb-sedge-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

-- Items flow EAST out of every source, so an input belt is one facing east on
-- the west side of a part and an output belt is one facing east on its east
-- side. Nothing in this suite ever needs a second direction except the two
-- refusal belts, which point SOUTH into a part's north face.
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

-- A source at x = -5 feeding east along the row, up to but not including x0.
local function feed(s, y, x0)
  source(s, -5, y)
  for x = -3, x0 - 1 do put(s, BELT, x, y, { direction = E }) end
end

-- A sink at x = 3 (chest at 4) drained from x0 eastwards.
local function drain(s, y, x0)
  for x = x0, 1 do put(s, BELT, x, y, { direction = E }) end
  return sink(s, 2, y)
end

local function chest_count(c)
  if not (c and c.valid) then return -1 end
  local total = 0
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
    total = total + item.count
  end
  return total
end

local function surf() return game.surfaces[SURF] end

local function audit_now(tag)
  surf().create_entity {
    name = AUDIT, position = P(-12, 0), force = "player", raise_built = true,
  }
  log("[BBB-SEDGE] audited tag=" .. tag)
end

--------------------------------------------------------------------------------
-- the mid-run edits
--------------------------------------------------------------------------------

-- sbld: a SECOND belt against a part that already has one. A script build, so
-- `player_index` is zero, the force.print arm of the refusal fires, and nothing
-- may be handed back anywhere in this run.
local function sbld_add()
  log("[BBB-SEDGE] sbld-add begin")
  put(surf(), BELT, 0, SBLD - 1, { direction = S })
  log("[BBB-SEDGE] sbld-add end")
end

-- srot: the same second belt, arrived at by ROTATION. Assigning `direction`
-- raises no event of any kind -- it is the failure envelope's own case -- so the
-- audit that follows is what finds the drift, and the repair pass behind it is
-- what reaches the refusal.
local function rot(dir, tag)
  local e = surf().find_entities_filtered {
    name = BELT, area = { { -0.4, SROT - 1 + 0.1 }, { 0.4, SROT - 1 + 0.9 } },
  }[1]
  if not e then error("bbb-sedge-test: the srot belt is missing") end
  e.direction = dir
  log("[BBB-SEDGE] srot " .. tag)
end

-- smrg: the bridging part, into a one-tile gap whose OWN tile already has a
-- belt on each side. Both halves are running and full; their teardowns are
-- AddPart's and are queued before the compiler sees the cluster they make.
local function merge_add()
  log("[BBB-SEDGE] merge-add begin")
  put(surf(), PART, 0, SMRG + 2)
  log("[BBB-SEDGE] merge-add end")
end

local function merge_remove()
  log("[BBB-SEDGE] merge-remove begin")
  local e = surf().find_entities_filtered {
    name = PART, area = { { -0.4, SMRG + 2.1 }, { 0.4, SMRG + 2.9 } },
  }[1]
  if e then e.destroy { raise_destroy = true } end
  log("[BBB-SEDGE] merge-remove end")
end

--------------------------------------------------------------------------------
-- reporting
--------------------------------------------------------------------------------

local function report(tag)
  local parts = {}
  for _, name in ipairs(storage.order) do
    local per = {}
    for _, c in ipairs(storage.rigs[name]) do per[#per + 1] = chest_count(c) end
    parts[#parts + 1] = name .. "=" .. table.concat(per, ",")
  end
  log(string.format("[BBB-SEDGE] sample tag=%s tick=%d %s",
    tag, game.tick, table.concat(parts, " ")))
end

-- THE ANTI-VACUITY LINE. Every rig in this suite is built to the rule it is
-- about, so a run in which some part quietly ended up with two belts against it
-- would refuse a rig the assertions expect to be running -- and a run in which a
-- refusal rig's extra belt failed to be placed would pass every rate check while
-- proving nothing. This reports what is really standing on the two tiles the
-- refusals are about.
local function report_tile(tag, x, y)
  local names = {}
  for _, e in pairs(surf().find_entities_filtered {
    area = { { x + 0.1, y + 0.1 }, { x + 0.9, y + 0.9 } },
  }) do
    names[#names + 1] = e.name
  end
  table.sort(names)
  log(string.format("[BBB-SEDGE] tile tag=%s at=%d,%d holds=[%s]",
    tag, x, y, table.concat(names, ",")))
end

--------------------------------------------------------------------------------
-- the schedule
--------------------------------------------------------------------------------

local SCHEDULE = {
  [60]   = function() audit_now("built") end,
  [500]  = sbld_add,
  [502]  = function() audit_now("post-sbld"); report_tile("post-sbld", 0, SBLD - 1) end,
  [600]  = function() rot(S, "on") end,
  [602]  = function() audit_now("post-rot"); report_tile("post-rot", 0, SROT - 1) end,
  [700]  = function() report("a") end,
  [1000] = function() report("b") end,
  [1200] = function() rot(E, "off") end,
  [1202] = function() audit_now("post-rot-back") end,
  [1400] = merge_add,
  [1402] = function() audit_now("post-merge"); report_tile("post-merge", 0, SMRG + 2) end,
  [1500] = function() report("c") end,
  [1800] = function() report("d") end,
  [1900] = merge_remove,
  [1902] = function() audit_now("post-unmerge") end,
  [2100] = function() report("e") end,
  [3400] = function() report("f") end,
  [3450] = function() audit_now("final") end,
}

script.on_event(defines.events.on_tick, function(e)
  local f = SCHEDULE[e.tick]
  if f then f() end
end)

--------------------------------------------------------------------------------
-- the rigs
--------------------------------------------------------------------------------

script.on_init(function()
  local s = make_surface()
  storage.rigs, storage.order = {}, {}
  local function rig(name, chests)
    storage.order[#storage.order + 1] = name
    storage.rigs[name] = chests
  end

  -- ctrl: the yardstick. A bare express belt from the same source to the same
  -- kind of sink.
  feed(s, CTRL, 0)
  rig("ctrl", { drain(s, CTRL, 0) })

  -- s11: 1 -> 1 over TWO parts. The west part carries the input, the east part
  -- carries the output, and neither carries both -- which is the rule.
  put(s, PART, 0, S11)
  put(s, PART, 1, S11)
  feed(s, S11, 0)
  rig("s11", { drain(s, S11, 2) })

  -- s22: 2 -> 2 over a 2x2 block. Two west parts, two east parts.
  do
    local out = {}
    for dy = 0, 1 do
      put(s, PART, 0, S22 + dy)
      put(s, PART, 1, S22 + dy)
      feed(s, S22 + dy, 0)
      out[#out + 1] = drain(s, S22 + dy, 2)
    end
    rig("s22", out)
  end

  -- s44: 4 -> 4 over a 4x2 block -- EIGHT parts for the shape four used to
  -- build, which is the footprint the rule costs.
  do
    local out = {}
    for dy = 0, 3 do
      put(s, PART, 0, S44 + dy)
      put(s, PART, 1, S44 + dy)
      feed(s, S44 + dy, 0)
      out[#out + 1] = drain(s, S44 + dy, 2)
    end
    rig("s44", out)
  end

  -- s35: 3 -> 5 over a 5x2 block. The bottom two west parts carry NOTHING,
  -- which is what makes the shape asymmetric without breaking the rule: P = 8
  -- with loopbacks, and a wrong edge list reads as a rate rather than a crash.
  do
    local out = {}
    for dy = 0, 4 do
      put(s, PART, 0, S35 + dy)
      put(s, PART, 1, S35 + dy)
      if dy < 3 then feed(s, S35 + dy, 0) end
      out[#out + 1] = drain(s, S35 + dy, 2)
    end
    rig("s35", out)
  end

  -- sbld: a working 2 -> 2, whose north-west part gets a second belt at t=500.
  do
    local out = {}
    for dy = 0, 1 do
      put(s, PART, 0, SBLD + dy)
      put(s, PART, 1, SBLD + dy)
      feed(s, SBLD + dy, 0)
      out[#out + 1] = drain(s, SBLD + dy, 2)
    end
    rig("sbld", out)
  end

  -- srot: the same shape, with a belt already standing on the north-west
  -- part's north face and pointing EAST -- which is not an edge, because a belt
  -- flowing past a cluster is not pointing at it. Rotating it south at t=600
  -- makes it one, silently.
  do
    local out = {}
    for dy = 0, 1 do
      put(s, PART, 0, SROT + dy)
      put(s, PART, 1, SROT + dy)
      feed(s, SROT + dy, 0)
      out[#out + 1] = drain(s, SROT + dy, 2)
    end
    put(s, BELT, 0, SROT - 1, { direction = E })
    rig("srot", out)
  end

  -- smrg: TWO 1 -> 1 balancers in one column with a one-tile gap between them,
  -- and a belt on BOTH SIDES OF THE GAP TILE. Bridging the gap makes one
  -- cluster whose bridging part would carry two belts -- so the merge must be
  -- refused with both standing networks untouched.
  --
  -- Neither gap belt is adjacent to either half: a belt beside the gap is
  -- DIAGONAL from the nearest part, and adjacency here is four-way.
  do
    put(s, PART, 0, SMRG)
    put(s, PART, 0, SMRG + 1)
    feed(s, SMRG, 0)
    local a = drain(s, SMRG + 1, 1)

    put(s, PART, 0, SMRG + 3)
    put(s, PART, 0, SMRG + 4)
    feed(s, SMRG + 3, 0)
    local b = drain(s, SMRG + 4, 1)

    put(s, BELT, -1, SMRG + 2, { direction = E })
    put(s, BELT, 1, SMRG + 2, { direction = E })
    rig("smrg", { a, b })
  end

  report_tile("init", 0, SMRG + 2)
  audit_now("t0")
  log("[BBB-SEDGE] init complete")
end)
