-- bbb-interactive-setup: the gestures a headless Factorio cannot make, and the
-- scenes the mod portal is captured from.
--
-- Every trigger in the gesture bands needs a PLAYER -- game.get_player resolves
-- to nothing in a --create, and on_player_mined_entity/on_built_entity with a
-- player_index cannot be raised from script -- so the suites pin the arithmetic
-- and the quantities, and a human pins the trigger. This mod exists to make that
-- human check cost thirty seconds per gesture instead of ten minutes of setup:
-- it stages the rigs beside spawn on a fresh world, hands the player the pieces,
-- and prints where to walk. test/interactive/README.md is the checklist.
--
-- EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
-- A part carries at most one interface linked belt, so a part serving an input
-- on one side and an output on another is what 2.1's collision validator now
-- refuses. The consequence for a rig is geometry -- a 4-in/4-out balancer is
-- EIGHT parts, a west column carrying the inputs and an east column carrying
-- the outputs -- and it is why two of these bands are redesigns rather than
-- re-lays: under this rule a player's belt can only change a balancer's port
-- count by landing on a part that has no belt yet. See agents/single-edge.md.
--
-- It stages and ASSERTS NOTHING, deliberately: the assertions are the player's
-- eyes and the [BBB] log lines the README says to grep for. What IS asserted,
-- headlessly, is that everything below is LEGAL -- test/assert-interactive.py
-- over a --create of this world requires no failed placement, no compile error
-- and no refusal at all, because the gestures are what create the refusals and
-- the staging must not.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbi-loader"
local AUDIT = "bbb-audit"
local N, E, S, W =
  defines.direction.north, defines.direction.east,
  defines.direction.south, defines.direction.west

-- Everything sits in this box, east of spawn: the gesture bands in a column
-- around x = 20 and the demo scenes in a second column around x = 56, far
-- enough apart that no scene's belts are inside a gesture rig's neighbour gate.
local X0, X1, Y0, Y1 = 8, 76, -34, 104
local COL = 20 -- the gesture column
local DX = 56  -- the demo column

local function P(x, y) return { x + 0.5, y + 0.5 } end

-- One tile along a cardinal direction.
local function step(dir)
  if dir == N then return 0, -1 end
  if dir == S then return 0, 1 end
  if dir == E then return 1, 0 end
  return -1, 0
end

-- EVERY PLACEMENT GOES THROUGH HERE so that a rig which failed to land says so
-- in the log rather than only in the player's confusion. The headless staging
-- check fails a run on one of these lines.
local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    log(string.format("[BBB-INTERACTIVE] could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

-- Containers are placed without raise_built: nothing of this mod's subscribes
-- to them (the entity filter admits belt-connectables and three named
-- prototypes), so raising for them would be a dispatch that decides nothing.
local function box(s, name, x, y)
  local e = s.create_entity { name = name, position = P(x, y), force = "player" }
  if not e then
    log(string.format("[BBB-INTERACTIVE] could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

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

-- A chest and a loader pushing a full belt along `dir`; the belt itself is the
-- caller's. Infinity chests, because the player should never see these run dry.
local function source(s, x, y, dir)
  local c = box(s, "infinity-chest", x, y)
  if c then
    c.infinity_container_filters = {
      { index = 1, name = "iron-plate", count = 1000, mode = "at-least" },
    }
  end
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
  box(s, "steel-chest", x + ox, y + oy)
end

-- feed_to(s, x, y, dir, len): a source pushing along `dir` down a run of `len`
-- belts that ENDS at (x, y). (x, y) is the tile touching the part, so the part
-- it feeds is one step further along `dir`.
local function feed_to(s, x, y, dir, len)
  len = len or 2
  local dx, dy = step(dir)
  local fx, fy = x - dx * (len - 1), y - dy * (len - 1)
  source(s, fx - 2 * dx, fy - 2 * dy, dir)
  for i = 0, len - 1 do put(s, BELT, fx + dx * i, fy + dy * i, { direction = dir }) end
end

-- drain_from(s, x, y, dir, len): `len` belts STARTING at (x, y) and running
-- along `dir` into a sink. (x, y) is the tile touching the part, so the part it
-- drains is one step BACK along `dir`.
local function drain_from(s, x, y, dir, len)
  len = len or 2
  local dx, dy = step(dir)
  for i = 0, len - 1 do put(s, BELT, x + dx * i, y + dy * i, { direction = dir }) end
  sink(s, x + dx * len, y + dy * len, dir)
end

--------------------------------------------------------------------------------
-- The gesture bands. Each is one gesture from the README, staged so the gesture
-- is the only thing left to do.
--------------------------------------------------------------------------------

-- A: THE MINER'S POCKET. A saturated dead-ended 4 -> 4, which under the rule is
-- EIGHT parts: four west parts carrying the inputs, four east parts carrying the
-- outputs. The gesture is mining it part by part and watching the items land in
-- the inventory at EVERY step, not only the last (the 2026-08-02 field report).
-- Eight parts rather than four means eight steps, which is more of the same
-- evidence rather than a different check.
local function band_pocket(s)
  local b = -24
  for i = 0, 3 do
    put(s, PART, COL, b + i)
    put(s, PART, COL + 1, b + i)
    feed_to(s, COL - 1, b + i, E)
    -- No sink: the outputs dead-end, so the network fills and stays full.
    put(s, BELT, COL + 2, b + i, { direction = E })
    put(s, BELT, COL + 3, b + i, { direction = E })
  end
end

-- B: THE BELT AT THE EDGE, and it is a REDESIGN rather than a re-lay. The old
-- gesture was "lay a belt on a free face of a 2-part balancer", and under the
-- one-belt-per-part rule there is no such face: every part of a working
-- balancer already has its belt. So the rig carries an ATTACHED EDGELESS PART
-- -- a fifth part hanging off the 2x2 block with nothing against it -- and the
-- belt goes there. That takes P from 2 to 4, and mining it takes P back to 2,
-- which is the same power-of-two boundary crossing the `edge` suite's `bmin`
-- leg pins: the machine halves, the reinsertion overflows, and the overflow is
-- what must reach the miner rather than the floor.
--
-- The same rig carries the SINGLE-EDGE refusal, because it is the only bound
-- that can be reached here without also crossing the port limit: a belt laid
-- against a part that already has one is refused, and the balancer keeps
-- running.
local function band_edge(s)
  local b = -12
  for i = 0, 1 do
    put(s, PART, COL, b + i)
    put(s, PART, COL + 1, b + i)
    feed_to(s, COL - 1, b + i, E)
    put(s, BELT, COL + 2, b + i, { direction = E })
    put(s, BELT, COL + 3, b + i, { direction = E })
  end
  -- The edgeless fifth part. Its west and east neighbours are empty and the
  -- input belts one row up are DIAGONAL from it, so nothing touches it until
  -- the player's belt does.
  put(s, PART, COL, b + 2)
end

-- C: THE SIXTY-FIFTH BELT. Sixty-four input parts in a 2x32 block, one part
-- below them carrying the single output, and one EDGELESS part above them
-- waiting for the player's belt -- so P = 64, the limit exactly, and the
-- sixty-fifth belt has somewhere legal to land. Without that spare part the
-- gesture could only ask an occupied part for a second belt, which is the other
-- bound and band B's job.
local function band_limit(s)
  local b = 0
  put(s, PART, COL, b - 1) -- the edgeless part the 65th belt lands on
  for i = 0, 31 do
    put(s, PART, COL, b + i)
    put(s, PART, COL + 1, b + i)
    put(s, BELT, COL - 1, b + i, { direction = E }) -- west inputs
    put(s, BELT, COL + 2, b + i, { direction = W }) -- east inputs
  end
  -- Feed eight of the sixty-four so items visibly flow: everything funnels to
  -- the one output, which therefore runs saturated.
  for _, i in ipairs { 4, 12, 20, 28 } do
    source(s, COL - 4, b + i, E)
    put(s, BELT, COL - 2, b + i, { direction = E })
    source(s, COL + 5, b + i, W)
    put(s, BELT, COL + 3, b + i, { direction = W })
  end
  -- The one output, on a part of its own below the block.
  put(s, PART, COL, b + 32)
  drain_from(s, COL, b + 33, S)
end

-- D: THE BRIDGE. Two balancers of thirty-two inputs and one output each -- a
-- 2x16 block plus an output part apiece -- with a one-tile gap between them and
-- ONE belt already standing against the gap tile. A part in that gap merges
-- them into a machine wanting 65 inputs, which is over the limit; the gap tile
-- itself carries a single belt, so the merge fails the port bound rather than
-- the one-belt-per-part bound and this stays the over-limit merge gesture.
local function band_bridge(s)
  local function block(top)
    for i = 0, 15 do
      put(s, PART, COL, top + i)
      put(s, PART, COL + 1, top + i)
      put(s, BELT, COL - 1, top + i, { direction = E })
      put(s, BELT, COL + 2, top + i, { direction = W })
    end
    for _, i in ipairs { 4, 11 } do
      source(s, COL - 4, top + i, E)
      put(s, BELT, COL - 2, top + i, { direction = E })
    end
  end

  block(45) -- block A: parts y 45..60
  block(62) -- block B: parts y 62..77, gap at (COL, 61)

  -- Each block's one output, on its own part, pointing away from the gap.
  put(s, PART, COL, 44)
  drain_from(s, COL, 43, N)
  put(s, PART, COL, 78)
  drain_from(s, COL, 79, S)

  -- The gap tile's belt: inert today (it is diagonal from every part), one more
  -- input the moment a part lands on (COL, 61). One belt and not two, because
  -- two on that tile would be the other refusal.
  put(s, BELT, COL - 1, 61, { direction = E })
end

-- E: FAST REPLACE, both directions.
--
-- The forward half is a belt line that ENDS one tile below a running 2 -> 2, and
-- that ending is the rule's doing: a part dropped into the MIDDLE of a line
-- takes the belt behind it as an input and the belt ahead of it as an output,
-- which is two belts on one part and is refused. Replacing the line's last tile
-- gives the new part one input, and the balancer becomes three in and two out.
--
-- The reverse half is a five-part column with edges on its two ends only, so the
-- three middle parts carry no interface and a belt can be laid on them. Only the
-- MIDDLE one splits cleanly: the new belt is an output of the half above and an
-- input of the half below, and each half's other part already carries its own
-- belt, so the two parts either side of the new belt must be the ones with
-- nothing on them.
local function band_fastreplace(s)
  local b = 90
  for i = 0, 1 do
    put(s, PART, COL, b + i)
    put(s, PART, COL + 1, b + i)
    feed_to(s, COL - 1, b + i, E)
    drain_from(s, COL + 2, b + i, E)
  end
  -- The line, ending on (COL, b+2). An east-facing belt against a part's SOUTH
  -- face is neither `dir` nor `back` from that side, so it is not an edge of the
  -- balancer above it -- the `pass` rig's rule, met from the other side.
  source(s, COL - 5, b + 2, E)
  for x = COL - 3, COL do put(s, BELT, x, b + 2, { direction = E }) end

  local c = b + 6
  for i = 0, 4 do put(s, PART, COL, c + i) end
  feed_to(s, COL - 1, c, E)
  drain_from(s, COL + 1, c + 4, E)
end

--------------------------------------------------------------------------------
-- The demo band: the mod portal's capture scenes, versioned here so a capture
-- is reproducible instead of living in a save nobody kept.
--
-- All five are single-edge, which changed four of them and retired the fifth:
-- `single-part-1-to-3-fanout` asked one part to carry four belts and cannot
-- exist under this rule, so the cross form -- a plus of five parts whose four
-- arms carry one belt each and whose centre carries none -- is the 1 -> 3 read
-- now.
--------------------------------------------------------------------------------

-- cross 1 -> 3. The centre part touches four parts and no belt at all, which is
-- the shape of the rule in one picture.
local function demo_cross(s)
  local cy = -24
  put(s, PART, DX, cy)
  put(s, PART, DX, cy - 1)
  put(s, PART, DX + 1, cy)
  put(s, PART, DX, cy + 1)
  put(s, PART, DX - 1, cy)
  feed_to(s, DX - 2, cy, E)
  drain_from(s, DX, cy - 2, N)
  drain_from(s, DX + 2, cy, E)
  drain_from(s, DX, cy + 2, S)
end

-- compact-column 8 -> 8: sixteen parts in a 2x8 block, the smallest 8 -> 8 the
-- rule allows.
local function demo_compact(s)
  local b = -10
  for i = 0, 7 do
    put(s, PART, DX, b + i)
    put(s, PART, DX + 1, b + i)
    feed_to(s, DX - 1, b + i, E)
    drain_from(s, DX + 2, b + i, E)
  end
end

-- The C: a ten-part spine with two arms, eight inputs down the spine's west
-- face and the outputs off the arms' free faces. `toparm` is how many parts the
-- top arm has, which is the only difference between the 8 -> 8 and the 8 -> 9:
-- four gives eight outputs, five gives nine and a P = 16 butterfly with
-- loopbacks. The spine's two corner parts carry nothing -- the arms turn there.
local function demo_cshape(s, b, toparm)
  for i = 0, 9 do put(s, PART, DX, b + i) end
  for x = DX + 1, DX + toparm do put(s, PART, x, b) end
  for x = DX + 1, DX + 4 do put(s, PART, x, b + 9) end
  for i = 1, 8 do feed_to(s, DX - 1, b + i, E) end
  for x = DX + 1, DX + toparm do drain_from(s, x, b - 1, N) end
  for x = DX + 1, DX + 4 do drain_from(s, x, b + 10, S) end
end

-- long-run 8 -> 8: sixteen parts in a single row, taking their inputs from the
-- north and giving their outputs to the south, alternately. Belt orientation
-- alone decides which is which, and here that is the whole picture.
local function demo_longrun(s)
  local y = 60
  for i = 0, 15 do
    put(s, PART, DX + i, y)
    if i % 2 == 0 then
      feed_to(s, DX + i, y - 1, S, 3)
    else
      drain_from(s, DX + i, y + 1, S)
    end
  end
end

--------------------------------------------------------------------------------

-- NOT `put`, and that is not an oversight. The audit marker is a shipped
-- prototype whose whole contract is that it destroys itself inside the dispatch
-- its own placement raises, so `create_entity` hands back nil for it every time
-- -- which `put` would report as a rig that failed to land.
local function audit(s, tag)
  s.create_entity { name = AUDIT, position = P(COL - 10, -30), force = "player",
                    raise_built = true }
  log("[BBB-INTERACTIVE] audited " .. tag)
end

script.on_init(function()
  -- No crash site and no intro: the rigs are the scenario.
  if remote.interfaces["freeplay"] then
    pcall(remote.call, "freeplay", "set_disable_crashsite", true)
    pcall(remote.call, "freeplay", "set_skip_intro", true)
  end
  local s = game.surfaces["nauvis"]
  prep_ground(s)

  band_pocket(s)
  band_edge(s)
  band_limit(s)
  band_bridge(s)
  band_fastreplace(s)

  demo_cross(s)
  demo_compact(s)
  demo_cshape(s, 6, 4)  -- c-shape 8 -> 8
  demo_cshape(s, 30, 5) -- c-shape 8 -> 9
  demo_longrun(s)

  -- TWO MARKERS, AND THE SECOND IS THE ONE WORTH READING. The audit reports the
  -- registry as it stands when its own dispatch begins, and the compiling is
  -- what that dispatch then does -- so the first marker reports every cluster
  -- unbuilt and the second, placed after the first has drained the queue,
  -- reports them built. A --create never reaches a tick, so there is no other
  -- way to see the compiled state from here.
  audit(s, "staged")
  audit(s, "compiled")

  log("[BBB-INTERACTIVE] gestures staged: pocket y-24, edge y-12, limit y0 " ..
      "(spare part at (20,-1)), bridge y45 (gap at (20,61)), fast replace y90")
  log("[BBB-INTERACTIVE] demo scenes staged at x56: cross y-24, compact y-10, " ..
      "c-shape 8->8 y6, c-shape 8->9 y30, long run y60")
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
    { pos = { COL, -10 }, text = "B: belt on (20,-9), mine it, then try (20,-13)" },
    { pos = { COL, 16 },  text = "C: 65th belt at (20,-2), facing south" },
    { pos = { COL, 61 },  text = "D: one part in this gap" },
    { pos = { COL, 92 },  text = "E: part onto (20,92); belt onto (20,98)" },
    { pos = { DX, -24 },  text = "demo: cross 1-to-3" },
    { pos = { DX, -10 },  text = "demo: compact column 8-to-8" },
    { pos = { DX, 10 },   text = "demo: c-shape 8-to-8" },
    { pos = { DX, 34 },   text = "demo: c-shape 8-to-9" },
    { pos = { DX + 7, 60 }, text = "demo: long run 8-to-8" },
  }
  p.force.chart(p.surface, { { X0 - 32, Y0 - 32 }, { X1 + 32, Y1 + 32 } })
  for _, t in ipairs(tags) do
    pcall(function()
      p.force.add_chart_tag(p.surface, { position = P(t.pos[1], t.pos[2]), text = t.text })
    end)
  end
  p.print("[BBB] Five gesture rigs east of you and five demo scenes further east.")
  p.print("[BBB] A: y=-24 mine the balancer part by part.")
  p.print("[BBB] B: y=-12 lay a south-facing belt at (20,-9), mine it, then try (20,-13).")
  p.print("[BBB] C: y=0 lay a south-facing belt at (20,-2), against the spare part.")
  p.print("[BBB] D: y=61 place a balancer part in the one-tile gap between the two big ones.")
  p.print("[BBB] E: y=90 drop a part onto (20,92); then lay a belt on (20,98).")
  p.print("[BBB] Expected outcomes and the log lines to grep: test/interactive/README.md")
end)
