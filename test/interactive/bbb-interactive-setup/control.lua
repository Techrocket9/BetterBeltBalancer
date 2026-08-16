-- bbb-interactive-setup: the four gestures a headless Factorio cannot make.
--
-- Every trigger below needs a PLAYER -- game.get_player resolves to nothing in
-- a --create, and on_player_mined_entity/on_built_entity with a player_index
-- cannot be raised from script -- so the suites pin the arithmetic and the
-- quantities, and a human pins the trigger. This mod exists to make that human
-- check cost thirty seconds per gesture instead of ten minutes of setup: it
-- stages the rigs beside spawn on a fresh world, hands the player the pieces,
-- and prints where to walk. test/interactive/README.md is the checklist.
--
-- It stages and ASSERTS NOTHING, deliberately: the assertions are the player's
-- eyes and the [BBB] log lines the README says to grep for.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbi-loader"
local N, E, S, W =
  defines.direction.north, defines.direction.east,
  defines.direction.south, defines.direction.west

-- Everything sits in this box, east of spawn.
local X0, X1, Y0, Y1 = 8, 32, -34, 84
local COL = 20 -- the part column

local function prep_ground(s)
  s.request_to_generate_chunks({ x = (X0 + X1) / 2, y = (Y0 + Y1) / 2 }, 4)
  s.force_generate_chunk_requests()
  for _, e in pairs(s.find_entities_filtered { area = { { X0 - 4, Y0 - 4 }, { X1 + 4, Y1 + 4 } } }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = { { X0 - 4, Y0 - 4 }, { X1 + 4, Y1 + 4 } } }
  local tiles = {}
  for x = X0 - 4, X1 + 4 do
    for y = Y0 - 4, Y1 + 4 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  s.set_tiles(tiles, true, false, false, false)
end

local function P(x, y) return { x + 0.5, y + 0.5 } end

-- One tile along a cardinal direction.
local function step(dir)
  if dir == N then return 0, -1 end
  if dir == S then return 0, 1 end
  if dir == E then return 1, 0 end
  return -1, 0
end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    log(string.format("[BBB-INTERACTIVE] could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

local function source(s, x, y, dir)
  -- A chest and a loader pushing a full belt in `dir`; the belt itself is the
  -- caller's. Infinity chests, because the player should never see these run dry.
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = "player" }
  c.infinity_container_filters = { { index = 1, name = "iron-plate", count = 1000, mode = "at-least" } }
  local ox, oy = step(dir)
  put(s, LOADER, x + ox, y + oy, { direction = dir, type = "output" })
end

-- A type="input" loader's container side is the tile its arrow points into, so
-- the chest goes ONE STEP ALONG dir from the loader -- in every direction, not
-- only east. The first cut of this file offset only x, which parked the three
-- north/south outputs' chests beside their loaders where nothing could reach
-- them; a player's screenshot found it.
local function sink(s, x, y, dir)
  put(s, LOADER, x, y, { direction = dir, type = "input" })
  local ox, oy = step(dir)
  s.create_entity { name = "steel-chest", position = P(x + ox, y + oy), force = "player" }
end

--------------------------------------------------------------------------------
-- The bands. Each is one gesture from the README, staged so the gesture is the
-- only thing left to do.
--------------------------------------------------------------------------------

-- A: THE MINER'S POCKET. A saturated dead-ended 4-part balancer; the gesture is
-- mining it part by part and watching the items land in the inventory at EVERY
-- step, not only the last (the 2026-08-02 field report).
local function band_pocket(s)
  local b = -24
  for i = 0, 3 do put(s, PART, COL, b + i) end
  for i = 0, 3 do
    source(s, COL - 4, b + i, E)
    for x = COL - 2, COL - 1 do put(s, BELT, x, b + i, { direction = E }) end
    for x = COL + 1, COL + 3 do put(s, BELT, x, b + i, { direction = E }) end
    -- no sink: the outputs dead-end, so the network fills and stays full
  end
end

-- B: THE SHRINK AT THE EDGE. A saturated dead-ended 2-part balancer with its
-- south face free; the gesture is laying one south-facing belt there (P 2->4),
-- waiting a breath, and mining it again (P 4->2) -- the overflow must reach the
-- inventory, not the floor (the second field report, bmin).
local function band_shrink(s)
  local b = -12
  for i = 0, 1 do put(s, PART, COL, b + i) end
  for i = 0, 1 do
    source(s, COL - 4, b + i, E)
    for x = COL - 2, COL - 1 do put(s, BELT, x, b + i, { direction = E }) end
    for x = COL + 1, COL + 3 do put(s, BELT, x, b + i, { direction = E }) end
  end
  -- the free face is (COL, b+2) = (COL, -10); the README sends the belt there
end

-- C: THE SIXTY-FIFTH BELT. Thirty-two parts carrying 64 input belts and one
-- output -- P = 64 = the limit exactly -- and RUNNING, so the player can see
-- that the refusal leaves it running. The gesture is a south-facing belt on the
-- north face (COL, -1): flying text, the cannot-build sound, the belt back in
-- the inventory, and not one item on the ground.
local function band_limit(s)
  local b = 0
  for i = 0, 31 do put(s, PART, COL, b + i) end
  for i = 0, 31 do
    put(s, BELT, COL - 1, b + i, { direction = E }) -- west inputs
    put(s, BELT, COL + 1, b + i, { direction = W }) -- east inputs
  end
  -- feed four of them so items visibly flow (chest 16, loader 17, belt 18
  -- meets the standing input belt at 19)
  for _, i in ipairs { 4, 12, 20, 28 } do
    source(s, COL - 4, b + i, E)
    put(s, BELT, COL - 2, b + i, { direction = E })
  end
  -- the one output, south face, into a visible sink
  put(s, BELT, COL, b + 32, { direction = S })
  put(s, BELT, COL, b + 33, { direction = S })
  sink(s, COL, b + 34, S)
end

-- D: THE BRIDGE. Two 16-part, 32-input balancers with a one-tile gap whose
-- flanks already carry two more input belts: the part that joins them asks for
-- 66 inputs, P = 128, over the limit. The gesture is placing one balancer part
-- in the gap -- both machines must keep running, the part must come back, and
-- nothing may spill (the merge the sixty-fifth-belt fix could not reach).
local function band_bridge(s)
  local function column(b)
    for i = 0, 15 do put(s, PART, COL, b + i) end
    for i = 0, 15 do
      put(s, BELT, COL - 1, b + i, { direction = E })
      put(s, BELT, COL + 1, b + i, { direction = W })
    end
    for _, i in ipairs { 4, 11 } do
      source(s, COL - 4, b + i, E)
      put(s, BELT, COL - 2, b + i, { direction = E })
    end
  end
  column(40) -- parts y 40..55
  column(57) -- parts y 57..72, gap at (COL, 56)
  -- each column's one output, pointing away from the gap, visibly running
  put(s, BELT, COL, 39, { direction = N })
  put(s, BELT, COL, 38, { direction = N })
  sink(s, COL, 37, N)
  put(s, BELT, COL, 73, { direction = S })
  put(s, BELT, COL, 74, { direction = S })
  sink(s, COL, 75, S)
  -- the gap tile's flanking belts: inert today, two extra inputs the moment a
  -- part lands on (COL, 56)
  put(s, BELT, COL - 1, 56, { direction = E })
  put(s, BELT, COL + 1, 56, { direction = W })
end

--------------------------------------------------------------------------------

script.on_init(function()
  -- No crash site and no intro: the rigs are the scenario.
  if remote.interfaces["freeplay"] then
    pcall(remote.call, "freeplay", "set_disable_crashsite", true)
    pcall(remote.call, "freeplay", "set_skip_intro", true)
  end
  local s = game.surfaces["nauvis"]
  prep_ground(s)
  band_pocket(s)
  band_shrink(s)
  band_limit(s)
  band_bridge(s)
  log("[BBB-INTERACTIVE] rigs staged: pocket y-24, shrink y-12, limit y0, bridge y40; gap at (20,56)")
end)

script.on_event(defines.events.on_player_created, function(e)
  local p = game.get_player(e.player_index)
  if not p then return end
  p.teleport({ COL - 8, -16 }, "nauvis")
  -- The pieces every gesture needs, so nothing has to be crafted.
  p.insert { name = BELT, count = 50 }
  p.insert { name = PART, count = 10 }
  local tags = {
    { pos = { COL, -23 }, text = "A: mine me part by part" },
    { pos = { COL, -12 }, text = "B: belt on my south face, then mine it" },
    { pos = { COL, 16 },  text = "C: 65th belt on my north face (20,-1)" },
    { pos = { COL, 56 },  text = "D: one part in this gap" },
  }
  p.force.chart(p.surface, { { X0 - 32, Y0 - 32 }, { X1 + 32, Y1 + 32 } })
  for _, t in ipairs(tags) do
    pcall(function()
      p.force.add_chart_tag(p.surface, { position = P(t.pos[1], t.pos[2]), text = t.text })
    end)
  end
  p.print("[BBB] Four rigs east of you. A: y=-24 mine the balancer part by part.")
  p.print("[BBB] B: y=-12 lay a south-facing belt on the free south face, then mine it.")
  p.print("[BBB] C: y=0 lay a 65th belt against the big one, at (20,-1) facing south.")
  p.print("[BBB] D: y=56 place a balancer part in the one-tile gap between the two big ones.")
  p.print("[BBB] Expected outcomes and the log lines to grep: test/interactive/README.md")
end)
