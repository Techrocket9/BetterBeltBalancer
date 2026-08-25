-- THE FIXTURE OBSERVER: what a Factorio 2.0 balancer save looks like when it is
-- opened on Factorio 2.1, before and after this mod's migration runs on it.
--
-- It builds no rig and it asserts nothing. Every balancer in these saves was
-- built by a 2.0 binary this machine no longer has
-- (test/fixtures-2.0/README.md), so the world is the fixture's; this mod says
-- what is in it at named moments, and test/assert-mig21.py decides whether that
-- is right.
--
-- ---------------------------------------------------------------------------
-- THE ONE THING IT HAS TO GET RIGHT: WHEN "BEFORE" IS
-- ---------------------------------------------------------------------------
--
-- The migration does not wait for a tick. This mod's guest heap is declined --
-- the mod is a different build -- so `fk_migrate` fires from
-- `on_configuration_changed`, BEFORE THE FIRST TICK, and by tick 0 the condemned
-- remnants are already down and their contents already on the ground. A sample
-- taken from `on_tick` is a sample taken afterwards, and the only "before" any
-- script can reach is this mod's own `on_configuration_changed`.
--
-- WHETHER THAT ONE RUNS FIRST IS FACTORIO'S CHOICE. Handlers run in mod load
-- order, and `bbb-mig21-observer` sorts before `better-belt-balancer`, which is
-- why this mod deliberately declares NO DEPENDENCY on it -- a dependency would
-- put it after. Measured, and it does not have to be trusted: if the order ever
-- flipped, the seeding below would find nothing to seed and report zero, and the
-- assertion script fails on a zero. It cannot pass vacuously.
--
-- IT IS ALREADY POST-PRUNING WHATEVER HAPPENS. The ENGINE deletes all but one
-- belt-connectable per tile at LOAD, before any script of any mod runs, with no
-- log line at all -- measured, m2's 77 part tiles came back with 77 interfaces
-- of the ~140 the save was built with. Nothing here can see the world as the 2.0
-- binary left it, and the assertions are written knowing that.
--
-- ---------------------------------------------------------------------------
-- WHY IT SEEDS THE NETWORKS, WHICH IS THE PART THAT LOOKS LIKE CHEATING
-- ---------------------------------------------------------------------------
--
-- The fixtures are `--create` saves. A `--create` never reaches a tick, so the
-- rigs were built and the save was written with every belt in them EMPTY -- and
-- a migration that recovers nothing, spills nothing and conserves nothing
-- trivially would satisfy every count in this suite while proving none of them.
--
-- So the one moment before the migration is also where the items go in: one item
-- into every transport line of every entity this mod's compiler places, on every
-- surface. That is not a stand-in for a running balancer's contents, it is
-- better than one for this purpose -- it is a KNOWN NUMBER, so "what the
-- teardown recovered" can be asserted as an equality against it rather than as a
-- floor. The suite fails if it is zero, which is what happens if the handler
-- order ever changes.

local OURS = { "bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter" }
local HIDDEN = "bbb-hidden"
local AUDIT = "bbb-audit"
local SEED = "iron-plate"

-- One item per line, and `insert_at_back` reports whether it landed. An empty
-- line always takes one, so on these saves this is one per line and the count is
-- exact; a line that somehow already held something at the back is skipped and
-- not counted, which keeps the number honest rather than optimistic.
local function seed_entity(e)
  local ok, count = pcall(function() return e.get_max_transport_line_index() end)
  if not ok or not count then return 0 end
  local n = 0
  for i = 1, count do
    local line = e.get_transport_line(i)
    if line and line.insert_at_back { name = SEED, count = 1 } then n = n + 1 end
  end
  return n
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

-- Everything the compiler placed, wherever it is: the hidden network proper and
-- the edge interfaces standing on the visible part tiles. Both are drained by a
-- teardown, so both are seeded.
local function seed_all()
  local hidden, visible = 0, 0
  for _, s in pairs(game.surfaces) do
    for _, e in pairs(s.find_entities_filtered { name = OURS }) do
      local n = seed_entity(e)
      if s.name == HIDDEN then hidden = hidden + n else visible = visible + n end
    end
  end
  log(string.format("[MIG21] seeded hidden=%d visible=%d total=%d",
    hidden, visible, hidden + visible))
end

-- WHAT THE RIGS HAVE DELIVERED, which only matters on one engine and is
-- reported on both.
--
-- On Factorio 2.1 every balancer in these fixtures is refused and its remnant
-- torn down, so this number moves only because the rigs' own bare control belts
-- and pass-through lines are still running -- it is reported and not asserted.
-- On 2.0 nothing is torn down at all: the clusters are ADOPTED whole and the
-- grandfather pass keeps them working, and "keeps working" has to mean items
-- arriving somewhere rather than merely entities standing. Every sink in these
-- worlds is an ordinary chest; the infinity chests are the SOURCES and are
-- excluded, because their contents are held at a filter level and say nothing.
local function delivered()
  local total = 0
  for _, s in pairs(game.surfaces) do
    for _, e in pairs(s.find_entities_filtered { type = "container" }) do
      local inv = e.get_inventory(defines.inventory.chest)
      if inv then
        for _, item in pairs(inv.get_contents()) do total = total + item.count end
      end
    end
  end
  return total
end

local function sample(tag)
  local tot_parts, tot_iface, tot_stacked, tot_ground = 0, 0, 0, 0
  local tot_hidden, tot_hitems, tot_vitems = 0, 0, 0
  for _, s in pairs(game.surfaces) do
    local parts = s.count_entities_filtered { name = "bbb-balancer-part" }
    local ours = s.find_entities_filtered { name = OURS }
    local ground = 0
    for _, e in pairs(s.find_entities_filtered { name = "item-on-ground" }) do
      ground = ground + e.stack.count
    end

    if s.name == HIDDEN then
      -- The hidden surface: the networks themselves, and what is standing in
      -- their transport lines. That second number is where a teardown's
      -- recovered items come FROM, and it is what has to reach zero for a
      -- network this mod has condemned.
      local items = 0
      for _, e in pairs(ours) do items = items + line_items(e) end
      tot_hidden = tot_hidden + #ours
      tot_hitems = tot_hitems + items
      log(string.format(
        "[MIG21] tag=%s surface=%s hidden_entities=%d hidden_items=%d ground=%d",
        tag, s.name, #ours, items, ground))
    else
      -- A visible surface: the parts a player can see, the edge interfaces
      -- standing on their tiles, and whether any TILE carries two
      -- belt-connectables -- which is the thing 2.1 forbids and the engine has
      -- already dealt with by the time anything here can look.
      local by_tile, stacked, worst, items = {}, 0, 0, 0
      for _, e in pairs(ours) do
        local k = math.floor(e.position.x) .. "," .. math.floor(e.position.y)
        by_tile[k] = (by_tile[k] or 0) + 1
        items = items + line_items(e)
      end
      for _, n in pairs(by_tile) do
        if n > 1 then stacked = stacked + 1 end
        if n > worst then worst = n end
      end
      tot_parts = tot_parts + parts
      tot_iface = tot_iface + #ours
      tot_stacked = tot_stacked + stacked
      tot_ground = tot_ground + ground
      tot_vitems = tot_vitems + items
      log(string.format(
        "[MIG21] tag=%s surface=%s parts=%d ours=%d stacked_tiles=%d " ..
        "worst_per_tile=%d iface_items=%d ground=%d",
        tag, s.name, parts, #ours, stacked, worst, items, ground))
    end
  end
  log(string.format(
    "[MIG21] total tag=%s parts=%d ours=%d stacked_tiles=%d ground=%d " ..
    "hidden_entities=%d hidden_items=%d iface_items=%d delivered=%d",
    tag, tot_parts, tot_iface, tot_stacked, tot_ground,
    tot_hidden, tot_hitems, tot_vitems, delivered()))
end

-- The audit marker is a shipped prototype and the only SYNCHRONOUS "re-classify
-- the world and report" trigger there is: placing one drains the guest's
-- deferred queue inside this dispatch, so the `[BBB] audit` line it writes
-- describes the world at this tick rather than at some tick after it. Every
-- other suite's test mod uses exactly this.
local function audit(tag)
  game.surfaces[1].create_entity {
    name = AUDIT, position = { x = 0.5, y = 0.5 }, force = "player",
    raise_built = true,
  }
  log("[MIG21] audited tag=" .. tag)
end

-- WHETHER THE FORCE CAN SEE THE GROUND ITS BALANCERS STAND ON, AND THE WALL
-- THAT MAKES THAT UNANSWERABLE HERE.
--
-- A `[gps=]` is a coordinate and nothing else: clicking one opens the map there
-- whether or not the force has charted it, and an uncharted point is BLACK. So
-- the mod charts what it pings, and the obvious check is `is_chunk_charted`
-- after the message.
--
-- IT ANSWERS FALSE FOR EVERYTHING ON A HEADLESS RUN. Measured on 2.0.77 and not
-- assumed: with no players, `force.chart` charts nothing, `force.chart_all` over
-- a fully generated surface charts nothing, a radar charts nothing, and NAUVIS'S
-- OWN ORIGIN CHUNK -- which every real game charts at world creation -- reads
-- uncharted too. A force with no players has no chart to write into. That puts
-- the EFFECT behind the same player wall as the flying text and the hand-back;
-- what is on this side of it is the guest's own `charted N from x,y to x,y`.
--
-- SO THIS IS A TRIPWIRE, NOT A MEASUREMENT OF THE FIX: it reports zero before
-- and zero after, and test/assert-mig21.py asserts exactly that -- so the day a
-- Factorio charts headlessly the run fails and asks for the real assertion
-- instead of this one.
--
-- ONE CHUNK PER PART TILE, counted per surface. `is_chunk_charted` takes a CHUNK
-- position, so a part at tile (x, y) is asked about at
-- (floor(x/32), floor(y/32)); several parts share a chunk and the count is over
-- DISTINCT chunks, which is what makes the samples comparable.
local function chart_state(tag)
  for _, force in pairs(game.forces) do
    local per_surface = {}
    for _, s in pairs(game.surfaces) do
      if s.name ~= HIDDEN then
        local chunks, charted = {}, 0
        for _, e in pairs(s.find_entities_filtered { name = "bbb-balancer-part", force = force }) do
          local cx = math.floor(e.position.x / 32)
          local cy = math.floor(e.position.y / 32)
          local k = cx .. "," .. cy
          if not chunks[k] then
            chunks[k] = true
            if force.is_chunk_charted(s, { x = cx, y = cy }) then charted = charted + 1 end
          end
        end
        local n = 0
        for _ in pairs(chunks) do n = n + 1 end
        if n > 0 then
          per_surface[#per_surface + 1] =
            string.format("%s:%d/%d", s.name, charted, n)
        end
      end
    end
    if #per_surface > 0 then
      -- THE CONTROL, in the same line: nauvis's origin chunk, generated by world
      -- creation and charted by nothing, for the same force.
      local nau = game.surfaces.nauvis
      log(string.format("[MIG21] chart tag=%s force=%d %s nauvis_origin=%s players=%d",
        tag, force.index, table.concat(per_surface, " "),
        tostring(nau.is_chunk_generated({ x = 0, y = 0 }) and
          force.is_chunk_charted(nau, { x = 0, y = 0 })),
        #game.players))
    end
  end
end

-- BEFORE, as near as any script can get to it -- and the one moment the
-- networks can be given something to lose. See the header for both.
script.on_configuration_changed(function()
  seed_all()
  sample("cfg")
  chart_state("cfg")
end)

script.on_event(defines.events.on_tick, function(ev)
  if ev.tick == 1 then
    -- The migration's own flush lands on the first deferred tick, so this is the
    -- first moment everything it does has happened.
    sample("t1")
    chart_state("t1")
    audit("t1")
  elseif ev.tick == 2 then
    -- One tick later, because the audit above forces a flush of its own and what
    -- matters is that the state settled rather than that the audit ran.
    sample("post-audit")
  elseif ev.tick == 300 then
    -- And a long way after nothing has been touched: a refused cluster has to be
    -- a STABLE state, not one that oscillates between teardown and rebuild.
    sample("final")
    chart_state("final")
    audit("final")
  end
end)
