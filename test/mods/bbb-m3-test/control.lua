-- bbb-m3-test: drives every lifecycle path that can change what the compiler
-- compiled from, and reports what happened.
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. The guest's own [BBB] lines
-- are the assertion surface and test/assert-m3.py decides whether they are
-- right; a test mod that computed the expected answer in Lua would be a second
-- implementation of the thing under test. Where the guest cannot see the answer
-- -- what a blueprint captured, how many items are on a surface -- the numbers
-- are logged raw under [BBB-M3] and judged there too.
--
-- EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
-- Every edge of a cluster is an interface linked belt standing on the cluster's
-- own tile, so a part carrying an input on its west side and an output on its
-- east carried TWO belt-connectables on one tile, which 2.1's collision
-- validator forbids. See agents/single-edge.md and guest/go/sedge.go.
--
-- What that costs this suite is GEOMETRY AND NOTHING ELSE. Every column of parts
-- becomes TWO columns -- a west part carrying the row's input and an east part
-- carrying its output -- so a 2-in/2-out rig is four parts and `live` is eight.
-- The lifecycle path each rig exists to drive is untouched; what moved is where
-- the belts stand. Per row:
--
--   x=-6 source chest   -5 loader   -4..-1 belts   0 WEST PART   1 EAST PART
--   x=2..4 belts        5 sink loader              6 chest
--
-- TWO PLACES NEEDED MORE THAN A RE-LAY, and both are the same shape: an edit
-- that used to land on a working balancer's free face has no free face to land
-- on any more. `phase_silent_notice` lays its belt DIAGONALLY from the cluster
-- instead -- inside the two-tile neighbour gate, so the cluster is re-classified,
-- and adjacent to nothing, so no tile gains a second belt. `died` kills the EAST
-- part of the second row rather than the west one, which is what keeps that
-- row's output orphaned and exactly frozen.
--
-- AND THE STRESS CHURN AVOIDS THE REFUSAL RATHER THAN EMBRACING IT. Its six
-- randomised edits are aimed so that no tile can ever carry two belts: the two
-- belt edits are the row's own single input and its own single output, and the
-- part edit adds and removes an EDGELESS part below the west column. That is a
-- decision and assert-m3.py asserts the negative -- zero one-belt-per-part
-- refusals over the whole run. This suite's subject is the twelve lifecycle
-- paths and its sharpest assertion is `drift=0 unbuilt=0` after 600 ticks of
-- churn; a churn that generated refusals would make its compile, build and
-- teardown counters a function of the rule rather than of the path under test,
-- and would leave clusters standing refused at the final audit. The refusal has
-- its own suite (`sedge`), which drives all three ways of reaching it.
--
-- The rigs, one per y band on a flat scratch surface:
--
--   ctrl    a bare express belt, chest to chest: the throughput yardstick
--   live    4 in / 4 out over EIGHT parts, saturated, and NOTHING is ever done
--           to it. It is the witness: every phase below happens around it,
--           including deleting the surface its hidden network lives on, and it
--           has to still be delivering four belts at the end
--   clone   2 in / 2 out over four parts; the source of clone_area/clone_brush
--   died    2 in / 2 out; the EAST part of the second row is killed with die()
--           while the network is full, which orphans that row's output
--   bdie    2 in / 2 out; an input BELT is killed with die()
--   noev    2 in / 2 out; an input belt is destroy()ed with NO EVENT AT ALL --
--           the incumbent's killer -- and then put back at the same tile, which
--           is also the regression test for the removal window leaking
--   swap    2 in / 2 out; a belt's direction is changed with no event (what an
--           undone rotation does), found by re-validation, then the same belt
--           is fast-replaced with a slower tier
--   forceA  2 in / 2 out on force "player", with forceB immediately below it on
--   forceB  force "bbb-other". They touch and must NOT become one cluster
--   bp      2 in / 2 out; the area a blueprint is taken of
--   paste   the same blueprint pasted as real entities, all in one tick
--   ghost   ghosts of the same four parts, then revived
--   bots    a roboport with construction robots builds ghosts of four parts
--
-- Plus three surfaces that are not rigs: bbb-m3-b (clone destination),
-- bbb-m3-doomed (deleted mid-run), bbb-m3-s (the stress surface, whose items
-- are counted against what was put into it after 600 ticks of randomised churn
-- and a full teardown).

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local BELT2 = "fast-transport-belt"
local LOADER = "bbbt-loader"
local AUDIT = "bbb-audit"
local E = defines.direction.east
local S = defines.direction.south
local W = defines.direction.west

local PITCH = 14
local OTHER_FORCE = "bbb-other"
local STRESS_ROWS = { 0, 14, 28, 42 }
local STRESS_STOCK = 2000
local STRESS_AREA = { { -30, -20 }, { 30, 80 } }

--------------------------------------------------------------------------------
-- deterministic randomness
--
-- An LCG carried in `storage`, not `math.random`: the churn schedule has to be
-- identical on every run for the assertions to mean anything, and it has to
-- survive the save/reload between --create and --benchmark.
--------------------------------------------------------------------------------

local function rnd(n)
  storage.seed = (storage.seed * 1103515245 + 12345) % 2147483648
  return (math.floor(storage.seed / 65536) % n) + 1
end

--------------------------------------------------------------------------------
-- surfaces and pieces
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

  local area = { { -18, -12 }, { 22, rows + 12 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -18, 22 do
    for y = -12, rows + 12 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  s.set_tiles(tiles, true, false, false, false)
  return s
end

local function P(x, y) return { x + 0.5, y + 0.5 } end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-m3-test: could not place %s at %s (%d,%d)", name, s.name, x, y))
  end
  return e
end

-- put_soft is put for the churn, where a collision is a legitimate outcome and
-- must not end the run.
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

local function source(s, x, y, force, count)
  local c
  if count then
    c = s.create_entity { name = "steel-chest", position = P(x, y), force = force or "player" }
    c.get_inventory(defines.inventory.chest).insert { name = "iron-plate", count = count }
  else
    c = s.create_entity { name = "infinity-chest", position = P(x, y), force = force or "player" }
    c.infinity_container_filters = {
      { index = 1, name = "iron-plate", count = 1000, mode = "at-least" },
    }
  end
  put(s, LOADER, x + 1, y, { direction = E, type = "output", force = force or "player" })
  return c
end

local function sink(s, x, y, force)
  put(s, LOADER, x, y, { direction = E, type = "input", force = force or "player" })
  return s.create_entity { name = "steel-chest", position = P(x + 1, y), force = force or "player" }
end

-- The sink loader's column, one tile east of the last output belt. Under the
-- one-belt rule the output belts start at x=2 rather than x=1, because x=1 is
-- the east part.
local SINKX = 5

local function chest_count(c)
  if not (c and c.valid) then return -1 end
  local total = 0
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
    total = total + item.count
  end
  return total
end

--------------------------------------------------------------------------------
-- rigs
--
--   x=-6 source chest  -5 loader  -4..-1 belts  0 WEST PART  1 EAST PART
--   x=2..4 belts       5 sink loader            6 chest
--
-- `rows` is the number of ROWS; the part count is twice it, because one tile may
-- carry only one belt and a row's input and output cannot stand against the same
-- one.
--------------------------------------------------------------------------------

local RIGS = {
  { name = "ctrl" },
  { name = "live",   rows = 4, ins = 4, outs = 4 },
  { name = "clone",  rows = 2, ins = 2, outs = 2 },
  { name = "died",   rows = 2, ins = 2, outs = 2 },
  { name = "bdie",   rows = 2, ins = 2, outs = 2 },
  { name = "noev",   rows = 2, ins = 2, outs = 2 },
  { name = "swap",   rows = 2, ins = 2, outs = 2 },
  { name = "forceA", rows = 2, ins = 2, outs = 2 },
  { name = "bp",     rows = 2, ins = 2, outs = 2 },
  { name = "paste",  outs = 2 },
  { name = "ghost",  outs = 2 },
  { name = "bots",   outs = 2 },
}

local function rigbase(name)
  for i, cfg in ipairs(RIGS) do
    if cfg.name == name then return (i - 1) * PITCH end
  end
end

local function build_rig(s, base, rows, ins, outs, force, finite)
  local r = { surface = s.name, base = base, outs = outs, out = {} }
  -- TWO PER ROW: the west one carries the row's input and the east one its
  -- output. Parts first, belts after, so the belt-adjacency trigger is on the
  -- critical path of every rig.
  for i = 0, rows - 1 do
    put(s, PART, 0, base + i, { force = force })
    put(s, PART, 1, base + i, { force = force })
  end
  for i = 0, ins - 1 do
    source(s, -6, base + i, force, finite)
    for x = -4, -1 do put(s, BELT, x, base + i, { direction = E, force = force }) end
  end
  for i = 0, outs - 1 do
    for x = 2, 4 do put(s, BELT, x, base + i, { direction = E, force = force }) end
    r.out[i + 1] = sink(s, SINKX, base + i, force)
  end
  return r
end

-- feed_and_drain gives a rig that was built some other way (a paste, a revive,
-- a robot) the belts and chests that make it measurable.
local function feed_and_drain(name, rows, belts)
  local s = game.surfaces["bbb-m3-a"]
  local r = storage.rigs[name]
  for i = 0, rows - 1 do
    source(s, -6, r.base + i)
    if belts then
      for x = -4, -1 do put(s, BELT, x, r.base + i, { direction = E }) end
      for x = 2, 4 do put(s, BELT, x, r.base + i, { direction = E }) end
    else
      -- The paste brought its own belts from x=-2 to x=4 (create_blueprint takes
      -- every entity whose box INTERSECTS the area, so the belt one tile outside
      -- each end came too); only the run-up from the loader is missing.
      for x = -4, -3 do put(s, BELT, x, r.base + i, { direction = E }) end
    end
    r.out[i + 1] = sink(s, SINKX, r.base + i)
  end
end

--------------------------------------------------------------------------------
-- counting
--------------------------------------------------------------------------------

local function count_surface(s, area)
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
-- phases
--------------------------------------------------------------------------------

local function mark(n, what)
  log(string.format("[BBB-M3] phase=%d %s", n, what))
end

local function audit_marker(y)
  game.surfaces["bbb-m3-a"].create_entity {
    name = AUDIT, position = P(16, y), force = "player", raise_built = true,
  }
end

-- 1. What a blueprint of a compiled balancer captures.
--
-- The hidden prototypes carry `not-blueprintable`, so the claim is that a
-- blueprint over a live balancer contains the visible parts and belts and NONE
-- of the network -- including the visible linked-belt interfaces, which sit on
-- the cluster's own tiles and would otherwise travel with the blueprint and
-- resurrect a network the compiler does not know about.
local function phase_blueprint_capture()
  mark(1, "blueprint capture over a compiled balancer")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("bp")
  local inv = game.create_inventory(1)
  inv[1].set_stack { name = "blueprint" }
  inv[1].create_blueprint {
    surface = s, force = "player",
    area = { { -1, base }, { 4, base + 2 } },
  }
  local names = {}
  for _, e in pairs(inv[1].get_blueprint_entities() or {}) do
    names[#names + 1] = e.name
  end
  table.sort(names)
  log("[BBB-M3] bp captured=[" .. table.concat(names, " ") .. "]")
  storage.bp = inv[1].get_blueprint_entities()
  inv.destroy()
end

-- 2. Ghosts, then a revive.
--
-- A ghost is what a blueprint actually produces in a real game, and a ghost is
-- NOT a part: the build event carries an `entity-ghost` whose name is
-- "entity-ghost", so the registry must not grow. Reviving it with raise_revive
-- is the script_raised_revive path; the bots phase below is the
-- on_robot_built_entity one.
local function phase_ghost_revive()
  mark(2, "ghosts placed, then revived")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("ghost")
  local ghosts = {}
  for i = 0, 1 do
    for x = 0, 1 do
      ghosts[#ghosts + 1] = s.create_entity {
        name = "entity-ghost", inner_name = PART, position = P(x, base + i),
        force = "player", raise_built = true,
      }
    end
  end
  log(string.format("[BBB-M3] ghosts placed=%d", #ghosts))
  local revived = 0
  for _, g in pairs(ghosts) do
    if g and g.valid then
      local _, e = g.revive { raise_revive = true }
      if e then revived = revived + 1 end
    end
  end
  log(string.format("[BBB-M3] ghosts revived=%d", revived))
  feed_and_drain("ghost", 2, true)
end

-- 3. A blueprint pasted as REAL entities, all inside one tick.
--
-- What an editor paste, and any mod that builds a blueprint directly, produces.
-- THREE markers now, because the guest batches (`fk.defer`, FKLUA-GAPS.md item
-- 12, fixed upstream): `paste-begin` .. `paste-end` brackets the tick the
-- entities arrive in and must contain NO compile at all, and `paste-flushed`
-- (tick 92, two ticks later so no handler-ordering question arises) closes the
-- window the single deferred compile has to land in. assert-m3.py counts
-- compiles in both.
local function phase_instant_paste()
  mark(3, "blueprint pasted as real entities in one tick")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("paste")
  -- Blueprint entity positions are relative to the blueprint's own origin,
  -- which is not the source rig's origin. Anchoring on the PART column rather
  -- than on the bounding box puts every piece exactly where the source rig had
  -- it, so the pasted rig can be fed and drained like any other.
  local px, py = math.huge, math.huge
  for _, e in pairs(storage.bp) do
    if e.name == PART then
      px = math.min(px, math.floor(e.position.x))
      py = math.min(py, math.floor(e.position.y))
    end
  end
  local placed = 0
  log("[BBB-M3] paste-begin")
  for _, e in pairs(storage.bp) do
    if put_soft(s, e.name, math.floor(e.position.x) - px,
          math.floor(e.position.y) - py + base, { direction = e.direction }) then
      placed = placed + 1
    end
  end
  log("[BBB-M3] paste-end")
  log(string.format("[BBB-M3] paste placed=%d of=%d", placed, #storage.bp))
  feed_and_drain("paste", 2, false)
end

local function phase_paste_flushed()
  log("[BBB-M3] paste-flushed")
end

-- 4/5. A construction robot builds a ghost part.
local function phase_bots_start()
  mark(4, "roboport and construction robots given ghosts to build")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("bots")
  local port = s.create_entity { name = "roboport", position = P(12, base + 1), force = "player" }
  port.get_inventory(defines.inventory.roboport_robot).insert {
    name = "construction-robot", count = 6,
  }
  local chest = s.create_entity { name = "storage-chest", position = P(9, base), force = "player" }
  chest.get_inventory(defines.inventory.chest).insert { name = PART, count = 8 }
  storage.port = port
  for i = 0, 1 do
    for x = 0, 1 do
      s.create_entity {
        name = "entity-ghost", inner_name = PART, position = P(x, base + i),
        force = "player", raise_built = true,
      }
    end
  end
end

local function phase_bots_check()
  mark(5, "what the robots built")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("bots")
  local n = s.count_entities_filtered {
    area = { { -1, base }, { 2, base + 2 } }, name = PART,
  }
  log(string.format("[BBB-M3] bots built=%d", n))
  if n > 0 then feed_and_drain("bots", 2, true) end
end

-- 6/7. Cloning an area, and cloning a brush.
--
-- The claims: the cloned parts form their OWN cluster with a freshly compiled
-- network; the visible interfaces that the clone copied along with them are
-- destroyed rather than left standing as a second, untracked network; and no
-- hidden-surface entity is cloned at all.
local function phase_clone_area()
  mark(6, "clone_area of a compiled balancer onto another surface")
  local a, b = game.surfaces["bbb-m3-a"], game.surfaces["bbb-m3-b"]
  local base = rigbase("clone")
  a.clone_area {
    source_area = { { -7, base }, { 7, base + 2 } },
    destination_area = { { -7, 2 }, { 7, 4 } },
    destination_surface = b,
    clone_entities = true, clone_tiles = false, clear_destination_entities = true,
  }
  local box = { { -12, -4 }, { 14, 12 } }
  log(string.format("[BBB-M3] clone-area parts=%d leaked=%d",
    b.count_entities_filtered { area = box, name = PART },
    b.count_entities_filtered {
      area = box, name = { "bbb-belt", "bbb-splitter", "bbb-lane-splitter" },
    }))
end

local function phase_clone_brush()
  mark(7, "clone_brush of the same balancer")
  local a, b = game.surfaces["bbb-m3-a"], game.surfaces["bbb-m3-b"]
  local base = rigbase("clone")
  local positions = {}
  for x = -7, 7 do
    for y = base, base + 1 do positions[#positions + 1] = { x = x, y = y } end
  end
  a.clone_brush {
    source_offset = { x = 0, y = base },
    destination_offset = { x = 0, y = 24 },
    source_positions = positions,
    destination_surface = b,
    clone_entities = true, clone_tiles = false, clear_destination_entities = true,
  }
  local box = { { -12, 20 }, { 14, 34 } }
  log(string.format("[BBB-M3] clone-brush parts=%d leaked=%d",
    b.count_entities_filtered { area = box, name = PART },
    b.count_entities_filtered {
      area = box, name = { "bbb-belt", "bbb-splitter", "bbb-lane-splitter" },
    }))
end

-- 8/9. A belt whose direction changes with NO EVENT, and the re-validation.
--
-- `entity.direction = x` raises nothing at all. That is exactly what an undone
-- ROTATION does to the world, and the reason the undo handler re-validates
-- rather than trusting the actions array it is handed.
--
-- The undo EVENT itself is out of reach here and the harness does not pretend
-- otherwise: a headless --create has no player to scope by, LuaUndoRedoStack
-- can be read and edited but never APPLIED, and script.raise_event refuses
-- ("on_undo_applied can't be raised through script"). What is tested is the
-- machinery the handler calls -- a pass that re-classifies every cluster from
-- the world -- reached here through the audit marker, which runs the same
-- classify-and-repair and additionally reports what it found.
local function phase_silent_rotate()
  mark(8, "belt direction changed with no event")
  local s = game.surfaces["bbb-m3-a"]
  at(s, -1, rigbase("swap"), { type = "transport-belt" }).direction = W
  log("[BBB-M3] swap: input belt turned around silently")
end

local function phase_revalidate()
  mark(9, "re-validation finds the silent rotation")
  audit_marker(0)
end

-- 10. Fast-replacing a belt with a different tier.
--
-- create_entity{fast_replace} destroys the belt that was there and raises no
-- mine event for it, so the BUILD event alone has to land the recompile. It is
-- the same shape as a construction robot performing an upgrade order, which is
-- why the bot-upgrade path needs no handler of its own.
local function phase_fast_replace()
  mark(10, "input belt fast-replaced with a slower tier")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("swap")
  at(s, -1, base, { type = "transport-belt" }).direction = E
  s.create_entity {
    name = BELT2, position = P(-1, base), direction = E, force = "player",
    fast_replace = true, raise_built = true,
  }
  local now = at(s, -1, base, { type = "transport-belt" })
  log(string.format("[BBB-M3] swap: belt is now %s", now and now.name or "gone"))
end

-- 11/12/13. A belt destroyed with NO EVENT WHATSOEVER: the incumbent's killer.
--
-- destroy{} with no raise_destroy is what a badly-behaved mod does, and no
-- amount of event handling can see it. The claim is structural: the guest holds
-- no reference to that belt so nothing goes stale, the network keeps running
-- (its linked belts do not care that a visible belt vanished -- items simply
-- stop arriving on that port), and the next event that touches the cluster
-- re-derives the edge list from the world and puts it right.
--
-- Phase 13 then puts a belt back on the SAME TILE. If the removal window ever
-- leaked -- if a position armed by an earlier removal were still armed -- that
-- belt would be classified as absent and this rig would never come back to full
-- rate. It is the M3 regression test for FKLUA item 9.
local function phase_silent_destroy()
  mark(11, "input belt destroyed with destroy{} -- no event of any kind")
  local s = game.surfaces["bbb-m3-a"]
  at(s, -1, rigbase("noev"), { type = "transport-belt" }).destroy()
  log("[BBB-M3] noev: belt gone, no event raised")
end

-- The placement is DIAGONAL from the nearest part and that is the whole of what
-- the one-belt rule changed here. It used to be a south-facing belt on the top
-- west part's north face, which was an extra input -- and under the rule that
-- part already carries its own input on its west side, so the same belt would
-- now be REFUSED and this phase would measure a refusal instead of a
-- re-classification. A belt at (-1, base-1) is inside the two-tile neighbour
-- gate, so the cluster is queued and re-derived from the world; it is
-- orthogonally adjacent to no part at all, so no tile gains a second belt. The
-- fingerprint moves anyway, because the belt phase 11 destroyed silently is
-- missing from the classification this placement provokes -- which is the thing
-- under test.
local function phase_silent_notice()
  mark(12, "an unrelated placement inside the cluster's neighbour gate")
  local s = game.surfaces["bbb-m3-a"]
  put(s, BELT, -1, rigbase("noev") - 1, { direction = E })
  log("[BBB-M3] noev: unrelated placement made, cluster re-classified")
end

local function phase_silent_recover()
  mark(13, "the destroyed belt is put back on the same tile")
  local s = game.surfaces["bbb-m3-a"]
  put(s, BELT, -1, rigbase("noev"), { direction = E })
  log("[BBB-M3] noev: belt replaced")
end

-- 14/15. Death.
--
-- The teardown that takes the hidden network's items back out is DEFERRED to the
-- next tick like every other recompile, so an audit marker forces the flush
-- inside this tick and the two counts stay one atomic sample apart. (Counting a
-- tick later would work for the assertion and would stop being a measurement of
-- the teardown: items would have moved for other reasons in between.)
--
-- BOTH SURFACES, and that is not belt and braces. Killing ONE part of a two-part
-- balancer leaves a cluster behind, so it is a recompile and not a removal: the
-- drained items go back INSIDE the network the flush rebuilds (2026-08-02,
-- `guest/go/carry.go`) instead of onto the floor beside it. A visible-only count
-- reads that correct behaviour as a loss -- it did, by two items, which is what
-- found this line. What is asserted is conservation across the pair.
local HIDDEN_BOX = { { 0, 0 }, { 2048, 720 } }

local function count_everywhere(s, box)
  return count_surface(s, box)
       + count_surface(game.surfaces["bbb-hidden"], HIDDEN_BOX)
end

local function phase_part_died()
  mark(14, "a balancer part killed with die() while its network is full")
  local s = game.surfaces["bbb-m3-a"]
  local base = rigbase("died")
  local box = { { -20, base - 6 }, { 20, base + 10 } }
  local before = count_everywhere(s, box)
  -- THE EAST PART OF THE SECOND ROW, not the west one. Under the one-belt rule
  -- a row's output stands against its east part, so killing that part is what
  -- takes the row's OUTPUT off the machine and leaves its chest orphaned -- the
  -- property this rig has always been about. Killing the west part would take an
  -- INPUT off instead and leave both outputs live at half a belt each, which is
  -- a different measurement. Three parts survive and stay one cluster: the row
  -- above is intact and the east part below it is still attached to it.
  at(s, 1, base + 1, { name = PART }).die()
  audit_marker(1)
  log(string.format("[BBB-M3] died: part killed, items %d -> %d",
    before, count_everywhere(s, box)))
end

local function phase_belt_died()
  mark(15, "an input belt killed with die()")
  local s = game.surfaces["bbb-m3-a"]
  at(s, -1, rigbase("bdie"), { type = "transport-belt" }).die()
  log("[BBB-M3] bdie: input belt killed")
end

-- 16/17. Surfaces.
local function phase_delete_surface()
  mark(16, "the surface a balancer stands on is deleted")
  game.delete_surface("bbb-m3-doomed")
end

local function phase_delete_hidden()
  mark(17, "the HIDDEN surface is deleted by another mod")
  game.delete_surface("bbb-hidden")
end

-- 18/19. The stress: 600 ticks of randomised churn around four clusters, then
-- everything comes down so that every item in every hidden network is handed
-- back and the total can be compared with what was put in.
local function phase_stress_start()
  mark(18, "stress begins")
  log(string.format("[BBB-M3] stress inserted=%d", #STRESS_ROWS * 2 * STRESS_STOCK))
end

-- EVERY ONE OF THE SIX IS AIMED SO THAT NO TILE CAN CARRY TWO BELTS. The two
-- belt edits are the row's own single input (west of the west part) and its own
-- single output (east of the east part), so a tile that gains a belt had none;
-- the part edit adds and removes an EDGELESS part below the west column, whose
-- three free faces are bare ground. See the header for why this churn avoids the
-- one-belt-per-part refusal rather than embracing it.
local function stress_step()
  local s = game.surfaces["bbb-m3-s"]
  local row = STRESS_ROWS[rnd(#STRESS_ROWS)]
  local action = rnd(6)
  if action == 1 then
    local b = at(s, -1, row, { type = "transport-belt" })
    if b then b.destroy { raise_destroy = true } end
  elseif action == 2 then
    if not at(s, -1, row, { type = "transport-belt" }) then
      put_soft(s, BELT, -1, row, { direction = E })
    end
  elseif action == 3 then
    local b = at(s, 2, row + 1, { type = "transport-belt" })
    if b then b.destroy { raise_destroy = true } end
  elseif action == 4 then
    if not at(s, 2, row + 1, { type = "transport-belt" }) then
      put_soft(s, BELT, 2, row + 1, { direction = E })
    end
  elseif action == 5 then
    if not at(s, 0, row + 2, { name = PART }) then put_soft(s, PART, 0, row + 2) end
  else
    local p = at(s, 0, row + 2, { name = PART })
    if p then p.destroy { raise_destroy = true } end
  end
end

-- Every teardown here is deferred, so the audit marker drains the queue before
-- the items are counted. Without it this would count the surface a tick before
-- the networks came down and report the run as having lost nearly everything.
local function phase_stress_end()
  mark(19, "stress ends: every part removed, so every network is torn down")
  local s = game.surfaces["bbb-m3-s"]
  for _, p in pairs(s.find_entities_filtered { name = PART }) do
    if p.valid then p.destroy { raise_destroy = true } end
  end
  audit_marker(2)
  log(string.format("[BBB-M3] stress recovered=%d", count_surface(s, STRESS_AREA)))
end

local function phase_audit()
  mark(20, "final audit")
  audit_marker(4)
end

-- M5: the I/O arrows must not leak.
--
-- Every visible interface the compiler places carries exactly one rendering
-- object, and the guest stores no id for it -- it relies on the engine
-- destroying a rendering object whose TARGET ENTITY is destroyed. This suite is
-- where that claim is worth testing: it has taken ~100 networks down, deleted a
-- surface with balancers on it, deleted the HIDDEN surface under every network
-- at once, cloned interfaces and killed some from outside. If any of those
-- paths left a rendering object behind, the count is higher than the number of
-- interfaces standing.
--
-- Counting entities and counting rendering objects is an independent
-- observation, not a second implementation of anything the guest does.
local function render_check(tick)
  local objs = rendering.get_all_objects("better-belt-balancer")
  local ifaces = 0
  for _, s in pairs(game.surfaces) do
    if s.name ~= "bbb-hidden" then
      ifaces = ifaces + #s.find_entities_filtered { name = "bbb-linked-belt" }
    end
  end
  log(string.format("[BBB-M3] t=%d renders=%d visible_interfaces=%d", tick, #objs, ifaces))
end

local function report(tick)
  local names = {}
  for name in pairs(storage.rigs) do names[#names + 1] = name end
  table.sort(names)
  for _, name in ipairs(names) do
    local r = storage.rigs[name]
    local outs = {}
    for i = 1, (r.outs or 1) do outs[i] = tostring(chest_count(r.out[i])) end
    log(string.format("[BBB-M3] t=%d rig=%s out=[%s]", tick, name, table.concat(outs, " ")))
  end
end

--------------------------------------------------------------------------------
-- schedule
--------------------------------------------------------------------------------

local SCHEDULE = {
  [30]   = phase_blueprint_capture,
  [60]   = phase_ghost_revive,
  [90]   = phase_instant_paste,
  [92]   = phase_paste_flushed,
  [120]  = phase_bots_start,
  [150]  = phase_clone_area,
  [180]  = phase_clone_brush,
  [210]  = phase_silent_rotate,
  [225]  = phase_revalidate,
  [240]  = phase_fast_replace,
  [270]  = phase_silent_destroy,
  [300]  = phase_silent_notice,
  [330]  = phase_silent_recover,
  [360]  = phase_part_died,
  [390]  = phase_belt_died,
  [420]  = phase_delete_surface,
  [450]  = phase_delete_hidden,
  [480]  = phase_bots_check,
  [540]  = function() report(540) end,
  [546]  = phase_stress_start,
  [1200] = phase_stress_end,
  [1260] = phase_audit,
  [1500] = function() report(1500); render_check(1500) end,
}

script.on_init(function()
  storage.seed = 20260801
  local rows = #RIGS * PITCH + 8
  local a = make_surface("bbb-m3-a", rows)
  make_surface("bbb-m3-b", 48)
  local doomed = make_surface("bbb-m3-doomed", 16)
  local stress = make_surface("bbb-m3-s", 60)

  game.create_force(OTHER_FORCE)

  storage.rigs = {}
  for i, cfg in ipairs(RIGS) do
    local base = (i - 1) * PITCH
    if cfg.name == "ctrl" then
      source(a, -6, base)
      for x = -4, 4 do put(a, BELT, x, base, { direction = E }) end
      storage.rigs.ctrl = { base = base, outs = 1, out = { sink(a, SINKX, base) } }
    elseif cfg.rows then
      storage.rigs[cfg.name] = build_rig(a, base, cfg.rows, cfg.ins, cfg.outs, "player")
    else
      storage.rigs[cfg.name] = { base = base, outs = cfg.outs, out = {} }
    end
  end

  -- The second force, directly below forceA's parts so the two touch. They must
  -- not merge, and each must get its own network out of its own belts.
  local fb = rigbase("forceA") + 2
  for i = 0, 1 do
    put(a, PART, 0, fb + i, { force = OTHER_FORCE })
    put(a, PART, 1, fb + i, { force = OTHER_FORCE })
  end
  storage.rigs.forceB = { base = fb, outs = 2, out = {} }
  for i = 0, 1 do
    source(a, -6, fb + i, OTHER_FORCE)
    for x = -4, -1 do put(a, BELT, x, fb + i, { direction = E, force = OTHER_FORCE }) end
    for x = 2, 4 do put(a, BELT, x, fb + i, { direction = E, force = OTHER_FORCE }) end
    storage.rigs.forceB.out[i + 1] = sink(a, SINKX, fb + i, OTHER_FORCE)
  end

  -- A rig on the surface that gets deleted.
  build_rig(doomed, 0, 2, 2, 2, "player")

  -- The stress rigs: finite sources, so the items can be counted.
  for _, row in ipairs(STRESS_ROWS) do
    for i = 0, 1 do
      put(stress, PART, 0, row + i)
      put(stress, PART, 1, row + i)
    end
    for i = 0, 1 do
      source(stress, -6, row + i, "player", STRESS_STOCK)
      for x = -4, -1 do put(stress, BELT, x, row + i, { direction = E }) end
      for x = 2, 4 do put(stress, BELT, x, row + i, { direction = E }) end
      sink(stress, SINKX, row + i)
    end
  end

  -- `--create` never reaches a tick, so the deferred flush the build events
  -- armed would otherwise run on the first tick of the BENCHMARK. The audit
  -- marker drains it here, which is also what keeps `--create` doing the
  -- compiling exactly as it did before the guest learned to batch.
  audit_marker(3)
  log(string.format("[BBB-M3] init complete: %d rigs, players=%d", #RIGS, #game.players))
end)

script.on_event(defines.events.on_tick, function(e)
  local f = SCHEDULE[e.tick]
  if f then f() end
  -- The roboport has no electric network out here, so it is topped up directly
  -- for as long as the bots have work to do.
  if storage.port and storage.port.valid and e.tick <= 480 then
    storage.port.energy = storage.port.electric_buffer_size
  end
  if e.tick > 546 and e.tick <= 1146 and e.tick % 6 == 0 then stress_step() end
end)
