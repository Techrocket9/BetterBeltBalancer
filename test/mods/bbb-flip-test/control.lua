-- bbb-flip-test: `bbb-multi-edge-parts`, driven through every transition it has.
--
-- THIS SUITE CANNOT RUN ON FACTORIO 2.1 AND THAT IS THE POINT. The setting is a
-- runtime-global bool that `mod-data/settings.lua` defines on 2.0.x and never on
-- 2.1.x, so on trunk's own engine there is nothing to flip: the marker prototype
-- is absent, the AND short-circuits, and the whole flip half of
-- guest/go/sedge.go is unreachable. `guest/go/edgemode` proves the FOLD under
-- `go test`; this is the only place anything drives the WORLD it decides about.
--
-- `settings.global` IS SCRIPT-WRITABLE, which is what makes a headless leg
-- possible at all -- it is the same write the grandfather pass makes, and the
-- reason the setting is runtime-global rather than startup.
--
-- ---------------------------------------------------------------------------
-- WHAT THE FLIP-OFF ACTUALLY DOES, WHICH IS NOT WHAT THE DESIGN TEXT SAID
-- ---------------------------------------------------------------------------
--
-- The design called a flip OFF with multi-edge balancers standing a SWEEP: tear
-- them down, spill, and tell the player. It is not. Turning the setting off puts
-- `settings.global` at false with those clusters still standing, and the very
-- next thing `settleEdgeMode` asks is `edgemode.GrandfatherNeeded(marker=true,
-- setting=Off, n>0)` -- which is TRUE, so the grandfather pass writes the
-- setting straight back ON and tells each owning force why. The flip is
-- VETOED, and the sweep can never stick: the two conditions are the same
-- condition.
--
-- That is the behaviour a player reported from a live 2.0.77 session and it is
-- the behaviour this suite pins. See agents/single-edge.md.
--
-- ---------------------------------------------------------------------------
-- THE RIGS
-- ---------------------------------------------------------------------------
--
--   ctrl   a bare express belt, the yardstick
--   sok    2 -> 2 over FOUR parts, a 2x2 block laid ONE BELT PER PART. It is
--          legal under both modes and must not be touched by any flip -- the
--          single-edge neighbour, and the control on every window
--   me1    2 -> 2 over TWO parts, the incumbent's idiom: each part carries an
--          input on its west face and an output on its east. Built at on_init,
--          so it is REFUSED at the false default; it compiles when the setting
--          goes on, and it drains freely, so its delivery is what says a vetoed
--          flip left it running
--   me2    the same shape, DEAD-ENDED, and BUILT AFTER the setting went on --
--          which is the field report's own rig. It fills and stays full, so
--          whatever a flip-off does to a standing multi-edge network's items
--          shows up here as a number rather than as a rounding error
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. test/assert-flip.py decides.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbf-loader"
local AUDIT = "bbb-audit"
local FLOW_ITEM = "iron-plate"
local SETTING = "bbb-multi-edge-parts"
local OURS = { "bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter" }
local E = defines.direction.east
local S = defines.direction.south

local SURF = "bbb-flip"

local CTRL = 0
local SOK = 6          -- rows 6..7, x = 0..1
local ME1 = 14         -- rows 14..15, x = 0
local ME2 = 22         -- rows 22..23, x = 0
local ROWS = 32

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
    error(string.format("bbb-flip-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
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

local function surf() return game.surfaces[SURF] end

local function chest_count(c)
  if not (c and c.valid) then return -1 end
  local total = 0
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
    total = total + item.count
  end
  return total
end

--------------------------------------------------------------------------------
-- the setting, read and written from script
--------------------------------------------------------------------------------

-- READING AN UNDEFINED RUNTIME SETTING RETURNS NIL AND RAISES NOTHING, so
-- `absent` is the ordinary answer on an engine that does not define it -- which
-- is exactly what this suite is skipped on. It is reported rather than assumed,
-- because a run whose setting never existed would satisfy several counts below
-- while proving none of them.
local function setting_value()
  local ok, v = pcall(function() return settings.global[SETTING] end)
  if not ok or v == nil then return "absent" end
  return tostring(v.value)
end

local function report_setting(tag)
  log(string.format("[FLIP] setting tag=%s value=%s", tag, setting_value()))
end

-- THE WRITE, AND WHY IT IS A `remote.call` AND NOT AN ASSIGNMENT.
--
-- `settings.global[k] = v` from THIS mod raises, measured on 2.0.77:
-- "Settings can only be changed by the owning player or the mod that made the
-- setting." A runtime-global has no owning player, so the mod that DEFINED the
-- setting is the only script in the game that may write it -- which means no
-- test mod can ever flip it and the whole flip half of guest/go/sedge.go would
-- be reachable by a human and by nothing else.
--
-- So better-belt-balancer opens the door itself, beside the audit and for the
-- same reason (guest/go/commands.go). What crosses is the same
-- `writeMultiEdgeSetting` a player's keypress reaches, so this drives the real
-- path rather than a stand-in: the write raises
-- `on_runtime_mod_setting_changed` SYNCHRONOUSLY, inside the assigning
-- statement, so everything the guest does about the flip has happened by the
-- time `remote.call` returns -- except the deferred flush it asks for, which
-- lands on the next tick.
local function flip(on, tag)
  log(string.format("[FLIP] writing tag=%s value=%s", tag, tostring(on)))
  local ok = remote.call("better-belt-balancer", "set-multi-edge-parts", on)
  log(string.format("[FLIP] wrote tag=%s accepted=%s value=%s",
    tag, tostring(ok), setting_value()))
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
  log(string.format("[FLIP] sample tag=%s tick=%d %s",
    tag, game.tick, table.concat(parts, " ")))
end

local function line_items(e)
  local ok, count = pcall(function() return e.get_max_transport_line_index() end)
  if not ok or not count then return 0 end
  local n = 0
  for i = 1, count do
    local line = e.get_transport_line(i)
    if line then n = n + #line end
  end
  return n
end

-- WHERE THE ITEMS ARE, which is the whole of what the field report was about.
-- `inside` is everything standing in a transport line of anything the COMPILER
-- placed, hidden surface and visible alike; `ground` is every loose item on
-- every surface. A vetoed flip has to move neither.
local function report_world(tag)
  local ground, inside, ours, hidden = 0, 0, 0, 0
  for _, s in pairs(game.surfaces) do
    for _, e in pairs(s.find_entities_filtered { name = "item-on-ground" }) do
      ground = ground + e.stack.count
    end
    for _, e in pairs(s.find_entities_filtered { name = OURS }) do
      inside = inside + line_items(e)
      if s.name == "bbb-hidden" then hidden = hidden + 1 else ours = ours + 1 end
    end
  end
  log(string.format(
    "[FLIP] world tag=%s tick=%d ground=%d inside=%d interfaces=%d hidden=%d",
    tag, game.tick, ground, inside, ours, hidden))
end

local function audit_now(tag)
  surf().create_entity {
    name = AUDIT, position = P(-12, 0), force = "player", raise_built = true,
  }
  log("[FLIP] audited tag=" .. tag)
end

--------------------------------------------------------------------------------
-- the mid-run edits
--------------------------------------------------------------------------------

-- The incumbent's idiom, in two parts: each one carries an input on its west
-- face and an output on its east, which is two belts on one tile.
local function build_multi(y0, dead_ended)
  local s = surf()
  local out = {}
  for dy = 0, 1 do
    put(s, PART, 0, y0 + dy)
    feed(s, y0 + dy, 0)
    if dead_ended then
      -- One belt tile and nothing after it: the network fills and stays full,
      -- so what it is holding when a flip lands is a real quantity.
      put(s, BELT, 1, y0 + dy, { direction = E })
      out[#out + 1] = false
    else
      out[#out + 1] = drain(s, y0 + dy, 1)
    end
  end
  return out
end

local function build_me2()
  log("[FLIP] me2-build begin")
  storage.rigs.me2 = build_multi(ME2, true)
  log("[FLIP] me2-build end")
end

-- Take the multi-edge clusters out of the world entirely, so that the flip OFF
-- that follows has nothing to veto. A dissolve is a REMOVAL, so its items spill
-- -- which is why every ground number in this suite is read as a DELTA over the
-- window it belongs to rather than as an absolute.
local function strip_multi()
  log("[FLIP] strip begin")
  local n = 0
  for _, y0 in ipairs { ME1, ME2 } do
    for dy = 0, 1 do
      for _, e in pairs(surf().find_entities_filtered {
        name = PART,
        area = { { -0.4, y0 + dy + 0.1 }, { 0.4, y0 + dy + 0.9 } },
      }) do
        e.destroy { raise_destroy = true }
        n = n + 1
      end
    end
  end
  log("[FLIP] strip end removed=" .. n)
end

-- A SECOND BELT against a part of `sok` that already has one: the ordinary
-- refusal, asked once at the false default the save starts at and once after
-- the second flip-off has stuck. Same gesture, same bound, two different reasons
-- for the mode to be single.
local function second_belt()
  log("[FLIP] second-belt begin")
  put(surf(), BELT, 0, SOK - 1, { direction = S })
  log("[FLIP] second-belt end")
end

--------------------------------------------------------------------------------
-- the schedule
--------------------------------------------------------------------------------

local SCHEDULE = {
  -- The save opens at the false default: me1 is built the incumbent's way and
  -- has to be refused.
  [60]   = function() report_setting("default"); audit_now("default") end,

  -- ON. The refused cluster's fingerprint never matched, so the requeue the
  -- handler makes is what finally compiles it.
  [200]  = function() flip(true, "on") end,
  [210]  = function() report_setting("post-on"); audit_now("post-on") end,

  -- ...and a multi-edge balancer BUILT while it is on, which is the shape the
  -- field report was made with.
  [250]  = build_me2,
  [270]  = function() audit_now("post-me2") end,

  [900]  = function() report("a"); report_world("a") end,
  [1300] = function() report("b") end,

  -- OFF, with both multi-edge balancers standing and me2 full. THE VETO.
  [1400] = function() report_world("pre-veto") end,
  [1405] = function() flip(false, "off-vetoed") end,
  [1406] = function() report_world("post-veto") end,
  [1410] = function() report_setting("post-veto"); audit_now("post-veto") end,
  [1420] = function() report_world("veto-settled") end,

  -- ...and the balancers are still running, which is what "a no-op on the
  -- world" has to mean.
  [1800] = function() report("c") end,
  [2600] = function() report("d"); report_world("d") end,

  -- Now take the multi-edge clusters away and flip OFF again. With nothing to
  -- veto the flip STICKS, and single-edge is enforced from there.
  [2700] = strip_multi,
  [2720] = function() audit_now("post-strip"); report_world("post-strip") end,
  [2800] = function() flip(false, "off-sticks") end,
  [2810] = function() report_setting("post-sticks"); audit_now("post-sticks") end,
  [2820] = function() report_world("post-sticks") end,

  -- And the bound is back: a second belt on a working single-edge part is
  -- refused again, now because the player turned the setting off rather than
  -- because it had never been on.
  [2900] = second_belt,
  [2910] = function() audit_now("post-second-belt") end,

  [3100] = function() report("f"); report_world("final") end,
  [3120] = function() report_setting("final"); audit_now("final") end,
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
  storage.rigs, storage.order = {}, { "ctrl", "sok", "me1", "me2" }

  feed(s, CTRL, 0)
  storage.rigs.ctrl = { drain(s, CTRL, 0) }

  -- sok: ONE BELT PER PART, and therefore legal in both modes. Two west parts
  -- carry the inputs, two east parts carry the outputs.
  do
    local out = {}
    for dy = 0, 1 do
      put(s, PART, 0, SOK + dy)
      put(s, PART, 1, SOK + dy)
      feed(s, SOK + dy, 0)
      out[#out + 1] = drain(s, SOK + dy, 2)
    end
    storage.rigs.sok = out
  end

  -- me1: the incumbent's idiom, draining freely. Refused until the flip.
  storage.rigs.me1 = build_multi(ME1, false)

  -- me2 does not exist yet: it is built at t=250, after the setting is on.
  storage.rigs.me2 = {}

  report_setting("init")
  audit_now("t0")
  log("[FLIP] init complete")
end)
