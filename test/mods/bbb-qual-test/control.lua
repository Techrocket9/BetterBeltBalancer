-- bbb-qual-test: every part in every rig here is UNCOMMON quality, because the
-- defect class this suite exists for is invisible at normal.
--
-- `LuaSurface.find_entity` resolves a bare name as NORMAL QUALITY ONLY (the
-- pinned 2.0.77 API: "Normal quality will be used"), so a guest call site that
-- looks a part up that way works perfectly on every normal-quality save and
-- silently fails on a part a player built from a quality-rolled item. The mig
-- suite's fidelity rig found the first such site (guest/go/legacy.go); this
-- suite drives the other four (guest/go/findpart.go is the fix stated once):
--
--   qblk   skin.go's restyle: a 2x2 BLOCK of uncommon parts, saturated 2->2.
--          The guest's own `[BBB] skin` line is the assertion surface -- the
--          block's four variations are m1's known literals -- and the rig also
--          answers the question nothing anywhere had asked: does an uncommon
--          balancer BALANCE at all. (It does; every other lookup in the guest
--          was already quality-blind.)
--   qcol   fastreplace.go's TRUE POSITIVE: a 1->1 column of four uncommon
--          parts. A belt is fast-replaced onto an interior part mid-run; the
--          part really is gone, so the guest must unregister it -- at any
--          quality. This half passes on the unfixed guest too (nil for the
--          wrong reason); the tripwire is qlone.
--   qlone  fastreplace.go's TRIPWIRE: one lone uncommon part, and a script
--          builds a COLLIDING belt on its very tile (create_entity permits
--          that; a player cannot do it). The part is still standing, so the
--          registry must not be edited -- and the unfixed guest, asking at
--          normal quality only, reads the part as gone and unregisters it.
--   qlim   limit.go's forceOfCluster: the over-limit column (32 parts, 64
--          inputs, one output -- the edge suite's `lim`, uncommon) given its
--          sixty-fifth belt mid-run. The refusal must still be DELIVERED: the
--          `told force` line is the arm a headless run can reach, and on the
--          unfixed guest the lookup fails and the refusal is delivered to
--          nobody. (revertOne shares the same lookup and the same fix; its
--          observable needs a player, which no headless run has.)
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. test/assert-qual.py decides;
-- a test mod that computed the expected answer would be a second
-- implementation of the thing under test.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbqual-loader"
local AUDIT = "bbb-audit"
local FLOW_ITEM = "iron-plate"
local N = defines.direction.north
local E = defines.direction.east
local W = defines.direction.west

local SURF = "bbb-qual"

-- Band bases. Far enough apart that no rig's probe belt is inside another
-- rig's two-tile neighbour gate.
local CTRL = 0
local QBLK = 12
local QCOL = 26
local QLONE = 40
local QLIM = 52          -- rows QLIM-3 (sink chest) .. QLIM+32 (the 65th belt)
local ROWS = QLIM + 40

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

  local area = { { -20, -12 }, { 20, ROWS + 8 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -20, 20 do
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
    error(string.format("bbb-qual-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

-- Every part in this suite is uncommon. One helper, so a rig cannot forget.
local function part(s, x, y)
  return put(s, PART, x, y, { quality = "uncommon" })
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

local function surf() return game.surfaces[SURF] end

local function audit_now(tag)
  surf().create_entity {
    name = AUDIT, position = P(-16, 0), force = "player", raise_built = true,
  }
  log("[BBB-QUAL] audited tag=" .. tag)
end

-- The anti-vacuity line: what quality one part of a rig REALLY carries, read
-- back from the world. A run where `quality = "uncommon"` silently did not
-- take would pass every conservation check while proving nothing, so the
-- assertion script requires this to say uncommon before it believes anything.
local function report_quality(rig, x, y)
  local e = surf().find_entities_filtered {
    name = PART, area = { { x + 0.1, y + 0.1 }, { x + 0.9, y + 0.9 } },
  }[1]
  log(string.format("[BBB-QUAL] quality rig=%s value=%s",
    rig, e and e.quality.name or "MISSING"))
end

--------------------------------------------------------------------------------
-- the mid-run probes
--------------------------------------------------------------------------------

-- THE COLLIDE PROBE, qlone. `create_entity` with no `fast_replace` places a
-- belt on the part's tile regardless of the collision mask (a script can; a
-- player cannot). The part is STILL STANDING afterwards, so the guest's
-- appearance check must leave the registry alone -- the unfixed guest, asking
-- for the part at normal quality only, read it as gone and unregistered it.
local function collide_probe()
  local s = surf()
  local b = s.create_entity {
    name = BELT, position = P(0, QLONE), force = "player",
    direction = E, raise_built = true,
  }
  local standing = #s.find_entities_filtered {
    name = PART, area = { { 0.1, QLONE + 0.1 }, { 0.9, QLONE + 0.9 } },
  }
  log(string.format("[BBB-QUAL] collide created=%s part-standing=%d",
    tostring(b ~= nil), standing))
end

-- THE TRUE REPLACE, qcol. The engine's own answer first (`can_fast_replace`
-- is the question a player's cursor asks, and quality gating it would be worth
-- knowing), then the replace itself: the engine destroys the part with NO
-- EVENT and raises only the belt's build event, which is the whole reason
-- fastreplace.go exists.
local function replace_probe()
  local s = surf()
  local can = s.can_fast_replace {
    name = BELT, position = P(0, QCOL + 1), direction = E, force = "player",
  }
  log("[BBB-QUAL] frep-can value=" .. tostring(can))
  local b = s.create_entity {
    name = BELT, position = P(0, QCOL + 1), force = "player",
    direction = E, fast_replace = true, raise_built = true,
  }
  local gone = #s.find_entities_filtered {
    name = PART, area = { { 0.1, QCOL + 1.1 }, { 0.9, QCOL + 1.9 } },
  }
  log(string.format("[BBB-QUAL] frep created=%s parts-left-on-tile=%d",
    tostring(b ~= nil), gone))
end

-- THE POKE, qblk. A belt inside the block's two-tile neighbour gate but NOT
-- adjacent to it: the cluster is queued, the flush runs, the edge list has not
-- moved and the shape has not moved -- so a healthy restyle says NOTHING, and
-- the unfixed one (which can never find the uncommon parts and never records a
-- variation as set) emits one more `[BBB] skin ... set=0` line per flush,
-- forever. The assertion is on the COUNT of the block's skin lines.
local function poke_add()
  put(surf(), BELT, 0, QBLK - 2, { direction = E })
end

local function poke_remove()
  local e = surf().find_entities_filtered {
    name = BELT, area = { { 0.1, QBLK - 1.9 }, { 0.9, QBLK - 1.1 } },
  }[1]
  if e then e.destroy { raise_destroy = true } end
end

-- The sixty-fifth belt, exactly as the edge suite lays it.
local function lim_add()
  put(surf(), BELT, 0, QLIM + 32, { direction = N })
end

--------------------------------------------------------------------------------
-- reporting
--------------------------------------------------------------------------------

local function report(tick)
  local parts = {}
  for _, name in ipairs(storage.order) do
    local per = {}
    for _, c in ipairs(storage.rigs[name]) do per[#per + 1] = chest_count(c) end
    parts[#parts + 1] = name .. "=" .. table.concat(per, ",")
  end
  log(string.format("[BBB-QUAL] sample tick=%d %s", tick, table.concat(parts, " ")))
end

--------------------------------------------------------------------------------
-- the schedule
--------------------------------------------------------------------------------

local SCHEDULE = {
  [150] = collide_probe,
  [152] = function() audit_now("post-collide") end,
  [200] = replace_probe,
  [202] = function() audit_now("post-replace") end,
  [300] = lim_add,
  [302] = function() audit_now("post-lim") end,
  [500] = poke_add,
  [510] = poke_remove,
  [900] = function() report(900) end,
  [2100] = function() report(2100) end,
  [2140] = function() audit_now("final") end,
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

  -- ctrl: the yardstick.
  source(s, -5, CTRL)
  for x = -3, 3 do put(s, BELT, x, CTRL, { direction = E }) end
  rig("ctrl", { sink(s, 4, CTRL) })

  -- qblk: a 2x2 block, two in west and two out east, saturated.
  local out = {}
  for dy = 0, 1 do
    for dx = 0, 1 do part(s, dx, QBLK + dy) end
  end
  for dy = 0, 1 do
    source(s, -5, QBLK + dy)
    for x = -3, -1 do put(s, BELT, x, QBLK + dy, { direction = E }) end
    for x = 2, 4 do put(s, BELT, x, QBLK + dy, { direction = E }) end
    out[#out + 1] = sink(s, 5, QBLK + dy)
  end
  rig("qblk", out)
  report_quality("qblk", 0, QBLK)

  -- qcol: four parts in a column, one belt in at the top and one out at the
  -- bottom, so it compiles (1->1) and its two INTERIOR parts carry no
  -- interface -- which is what makes the fast replace at QCOL+1 legal.
  for dy = 0, 3 do part(s, 0, QCOL + dy) end
  put(s, BELT, -1, QCOL, { direction = E })
  put(s, BELT, 1, QCOL + 3, { direction = E })
  report_quality("qcol", 0, QCOL + 1)

  -- qlone: one part, nothing against it. A cluster with no edges is a
  -- legitimate half-built state; what matters is that it is REGISTERED, so
  -- the collide probe's wrong answer is visible as a cluster count.
  part(s, 0, QLONE)
  report_quality("qlone", 0, QLONE)

  -- qlim: the edge suite's `lim` column, uncommon. Thirty-two parts, a belt on
  -- both sides of each pointing inwards (64 inputs, P = MaxPorts exactly),
  -- three of them fed, one output leaving north off the top into a chest.
  for r = 0, 31 do part(s, 0, QLIM + r) end
  for r = 0, 31 do
    put(s, BELT, -1, QLIM + r, { direction = E })
    put(s, BELT, 1, QLIM + r, { direction = W })
  end
  for r = 0, 2 do
    source(s, -5, QLIM + r)
    for x = -3, -2 do put(s, BELT, x, QLIM + r, { direction = E }) end
  end
  put(s, BELT, 0, QLIM - 1, { direction = N })
  put(s, LOADER, 0, QLIM - 2, { direction = N, type = "input" })
  local lim_chest = s.create_entity {
    name = "steel-chest", position = P(0, QLIM - 3), force = "player",
  }
  rig("qlim", { lim_chest })
  report_quality("qlim", 0, QLIM)

  audit_now("t0")
  log("[BBB-QUAL] init complete")
end)
