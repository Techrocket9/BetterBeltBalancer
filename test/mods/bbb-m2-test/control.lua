-- bbb-m2-test: drives the network compiler and reports what came out.
--
-- Deliberately plain Lua, and it ASSERTS NOTHING. It builds rigs, samples the
-- output chests at two ticks, and logs the numbers; test/assert-m2.py decides
-- whether they are right. A test mod that computed the expected balance in Lua
-- would be a second implementation of the thing under test.
--
-- The rigs, one per y band on a flat scratch surface:
--
--   ctrl     a bare express belt, chest to chest. The yardstick: whatever this
--            delivers in the sample window is what one saturated belt is worth,
--            so "full throughput" is a comparison against the engine rather
--            than against a number someone worked out on paper.
--   sat4     4 parts, 4 belts in, 4 belts out, everything saturated
--   sat8     the same at 8, which needs three butterfly stages and two jumper
--            blocks rather than two and one
--   a3to5    3 in, 5 out: N != M, and P=8 with loopbacks on the spare ports
--   a4to1    4 in, 1 out: the other asymmetry, where spare OUTPUT ports have to
--            dead-end because there are no spare input ports to loop them into
--   starve   4 in, 4 out, but only ONE input has a source. This is the case
--            that kills every chest-based design (Techrocket9 measured one
--            output draining >9,000 items while its peers got ~80)
--   block    4 in, 4 out, but the fourth output has nowhere to go
--   regrow   3 in, 4 out; a fourth input belt is added at tick 900, under load
--   xsurf    sat4 again on a SECOND surface, because the network lives on a
--            third one and linked belts are the only thing joining them
--
-- The SHAPE band, added because the six shapes above are six of the sixty-four
-- (n, m) pairs with n, m <= 8 and the pure-Go fixed-point model
-- (plan.PropagateLoop) is exact only for n <= m -- everything with dead-ended
-- spare outputs can only be covered in a real Factorio:
--
--   sq3      3 in, 3 out. P=4, Loop=1: the smallest square shape that is not a
--            power of two, and probably the most common balancer anyone builds
--   a2to3    2 in, 3 out. P=4, Loop=1; each output gets 2/3 of a belt
--   a5to3    5 in, 3 out. P=8, Loop=3 and TWO DEAD-ENDED output ports -- the
--            blocking regime no linear model can express
--   n9m9     9 in, 9 out. P=16: the first FOUR-stage butterfly, three jumper
--            blocks, ever built in a real game
--   fdbk     a literal feedback loop: a third output belt curls round through
--            the world and comes back into the cluster's north face, so the
--            machine sees 3 in / 3 out and one of each is itself
--   tslow    4 in, 4 out, but one OUTPUT ROW is a normal-tier belt. A
--            rate-LIMITED port rather than a fully blocked one
--   lane     the lane-fidelity rig. Both inputs are SIDE-LOADED, so each is
--            half a belt on ONE lane; chest totals cannot see the difference,
--            so this one is asserted on per-lane occupancy at the outputs
--
-- The EDGE-TYPE band. classifySide keys on the entity's `type` and names six of
-- them; until these rigs existed, only "transport-belt" had ever been run:
--
--   uio      2->2 through UNDERGROUND ends placed directly against the parts,
--            both arms of the belt_to_ground_type branch
--   spio     2->2 fed and drained by vanilla express SPLITTERS whose faces
--            span both parts, so each half is its own edge
--   lio      1->1 through LOADERS directly against the part -- and the first
--            1->1 (P=1, five-entity) flow rig in any suite
--   lsio     2->2 through LANE SPLITTERS against the parts. Base ships the
--            `lane-splitter` TYPE and not one buildable entity of it, so
--            data.lua clones the mod's own hidden one to have anything to place
--   pass     the NEGATIVE: a belt line running PAST the cluster's north face,
--            perpendicular to it, which must not be classified as an edge and
--            must not have anything stolen from it
--
-- Plus two measurements that are not rigs:
--   * a profiler around a forced full recompile of sat4 and sat8
--   * an item-conservation check around a recompile of a network that is FULL:
--     sample, recompile, sample again, all inside one tick so nothing moves for
--     any other reason.
--
-- FORCING THE FLUSH. The guest batches: a build or mine event updates its
-- registry inside the event and defers the recompile to the next tick
-- (`fk.defer`), so a measurement taken in the tick that laid the belt would see
-- nothing at all. `bbb-audit` -- a shipped marker prototype whose whole purpose
-- is "re-classify and repair everything, now" -- is the synchronous escape
-- hatch, and it is what both measurements below use. That is also why on_init
-- ends with one: `--create` never reaches a tick, so without it every network
-- in the save would be compiled on the first tick of the BENCHMARK instead.

local PART = "bbb-balancer-part"
local AUDIT = "bbb-audit"
local BELT = "express-transport-belt"
local SLOWBELT = "transport-belt" -- normal tier: exactly 1/3 of express
local UNDER = "express-underground-belt"
local SPLIT = "express-splitter"
local LSPLIT = "bbbt-lane-splitter" -- see data.lua: base has no buildable one
local LOADER = "bbbt-loader"
local E = defines.direction.east
local N = defines.direction.north
local S = defines.direction.south
local W = defines.direction.west

local PITCH = 12 -- rows between rigs
local HALFX = 16 -- how far either side of x=0 the scratch surface is cleared

--------------------------------------------------------------------------------
-- surface
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

--------------------------------------------------------------------------------
-- pieces
--------------------------------------------------------------------------------

local function P(x, y) return { x + 0.5, y + 0.5 } end

-- audit_now asks the guest to drain its deferred queue and re-classify every
-- cluster, synchronously, inside this call. See "forcing the flush" above.
local function audit_now(y)
  game.surfaces["bbb-m2-a"].create_entity {
    name = AUDIT, position = P(-30, y or 0), force = "player", raise_built = true,
  }
end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-m2-test: could not place %s at %s (%d,%d)", name, s.name, x, y))
  end
  return e
end

-- put_at is put() for the things whose position is not a tile centre. A
-- two-tile splitter facing east sits on the boundary BETWEEN its two rows, so
-- its y is an integer and not an integer-plus-a-half.
local function put_at(s, name, px, py, extra)
  local args = { name = name, position = { px, py }, force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-m2-test: could not place %s at %s (%.1f,%.1f)", name, s.name, px, py))
  end
  return e
end

-- A run of belts, inclusive, all facing the same way. `along` is "x" or "y".
local function belts(s, name, dir, along, from, to, fixed)
  local step = (to >= from) and 1 or -1
  for i = from, to, step do
    if along == "x" then put(s, name, i, fixed, { direction = dir })
    else put(s, name, fixed, i, { direction = dir }) end
  end
end

-- source/sink take the direction the ITEMS travel. `dir` defaults to east,
-- which is every rig but `lane`, whose feeds come down from the north.
local function source(s, x, y, dir)
  dir = dir or E
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = "player" }
  c.infinity_container_filters = {
    { index = 1, name = "iron-plate", count = 1000, mode = "at-least" },
  }
  if dir == S then
    put(s, LOADER, x, y + 1, { direction = S, type = "output" })
  else
    put(s, LOADER, x + 1, y, { direction = E, type = "output" })
  end
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
-- rigs
--
--   x=-5 source chest   -4 loader   -3..-1 belts   0 PART   1..3 belts
--   x=4 sink loader     5 chest
--------------------------------------------------------------------------------

-- Every rig gets a y band of `span` rows, defaulting to PITCH, and the bands
-- are laid out in table order. The first nine all take the default, so their
-- bases are the (i-1)*PITCH they have always been; a rig that needs more room
-- than one pitch says so rather than silently overlapping its neighbour.
--
-- `build` is for the shapes the uniform builder below cannot express -- a
-- feedback loop, side-loaded inputs, undergrounds, splitters, loaders. It gets
-- (surface, base) and returns the list of output chests, in the order
-- assert-m2.py reads them.
local RIGS = {
  { name = "ctrl" },
  { name = "sat4",   parts = 4, ins = 4, outs = 4 },
  { name = "sat8",   parts = 8, ins = 8, outs = 8 },
  { name = "a3to5",  parts = 5, ins = 3, outs = 5 },
  { name = "a4to1",  parts = 4, ins = 4, outs = 1 },
  { name = "starve", parts = 4, ins = 4, outs = 4, fed = { [1] = true } },
  { name = "block",  parts = 4, ins = 4, outs = 4, blocked = { [4] = true } },
  { name = "regrow", parts = 4, ins = 3, outs = 4, grow_to = 4 },
  { name = "xsurf",  parts = 4, ins = 4, outs = 4, other_surface = true },

  -- the shape band
  { name = "sq3",    parts = 3, ins = 3, outs = 3 },
  { name = "a2to3",  parts = 3, ins = 2, outs = 3 },
  { name = "a5to3",  parts = 5, ins = 5, outs = 3 },
  { name = "n9m9",   parts = 9, ins = 9, outs = 9, span = 16 },
  { name = "tslow",  outs = 4, build = "tslow" },
  { name = "fdbk",   outs = 2, build = "fdbk" },
  { name = "lane",   outs = 2, build = "lane" },

  -- the edge-type band
  { name = "uio",    outs = 2, build = "uio" },
  { name = "spio",   outs = 2, build = "spio" },
  { name = "lio",    outs = 1, build = "lio" },
  { name = "lsio",   outs = 2, build = "lsio" },
  { name = "pass",   outs = 3, build = "pass" },
}

local function build_input_row(s, y, fed)
  if fed then source(s, -5, y) end
  for x = -3, -1 do put(s, BELT, x, y, { direction = E }) end
end

local function build_output_row(s, y, blocked)
  for x = 1, 3 do put(s, BELT, x, y, { direction = E }) end
  if not blocked then return sink(s, 4, y) end
  return nil
end

--------------------------------------------------------------------------------
-- the custom builders
--
-- Every one of them places its PARTS first and its belts after, for the reason
-- build_rig gives: the belt-adjacency trigger is then on the critical path of
-- every rig rather than only of the ones the uniform builder makes.
--------------------------------------------------------------------------------

local BUILD = {}

-- tslow: 4 in, 4 out, all express EXCEPT the last output row, which is a
-- normal-tier belt -- exactly a third of express. A RATE-LIMITED port, which is
-- a different question from `block`'s fully dead one: the three express outputs
-- must stay at a full belt each while the fourth trickles, and the balancer
-- must not throttle itself to the slow port's rate. The sink loader stays
-- express so the belt is the only limiter.
function BUILD.tslow(s, base)
  local out = {}
  for i = 0, 3 do put(s, PART, 0, base + i) end
  for i = 0, 3 do
    source(s, -5, base + i)
    belts(s, BELT, E, "x", -3, -1, base + i)
  end
  for i = 0, 2 do
    belts(s, BELT, E, "x", 1, 3, base + i)
    out[#out + 1] = sink(s, 4, base + i)
  end
  belts(s, SLOWBELT, E, "x", 1, 3, base + 3)
  out[4] = sink(s, 4, base + 3)
  return out
end

-- fdbk: a literal feedback loop.
--
--     row base      <- <- <- <- <- <- +      the return run
--     row base+1    v                 ^
--     row base+2  ->[P]->  sink       ^      real in/out
--     row base+3  ->[P]->  sink       ^      real in/out
--     row base+4    [P]-> -> -> -> -> +      the loop's own output
--
-- The machine sees 3 in and 3 out, and one of each is itself. In steady state
-- the loop carries L, every output carries (2+L)/3, and the loop's output IS
-- its input, so L = 1: each real output ends up at exactly one belt and the two
-- of them together at exactly the two belts that went in. The interesting part
-- is that the loop is a physical belt in the world, so it fills, and a network
-- that jams instead of settling shows up as a rate collapse.
--
-- No loop tile is orthogonally adjacent to a part except the two that are meant
-- to be edges: (1, base+4), the loop's output, and (0, base+1), its input into
-- the north face. The return run is kept two rows clear of the parts for that
-- reason and not for tidiness.
function BUILD.fdbk(s, base)
  local out = {}
  for i = 2, 4 do put(s, PART, 0, base + i) end
  for i = 2, 3 do
    source(s, -5, base + i)
    belts(s, BELT, E, "x", -3, -1, base + i)
    belts(s, BELT, E, "x", 1, 3, base + i)
    out[#out + 1] = sink(s, 4, base + i)
  end
  -- east along the bottom, north up column 6, west along the top, south in.
  belts(s, BELT, E, "x", 1, 5, base + 4)
  belts(s, BELT, N, "y", base + 4, base + 1, 6)
  belts(s, BELT, W, "x", 6, 1, base)
  belts(s, BELT, S, "y", base, base + 1, 0)
  return out
end

-- lane: the lane-fidelity rig, and the only one in any suite that chest totals
-- cannot judge.
--
-- Both inputs are fed by SIDE-LOADING and by nothing else -- a belt joining
-- from the NORTH -- so each input row carries exactly half a belt, all of it on
-- the same (left) lane. A vanilla splitter is lane-PRESERVING (spike S1), so a
-- network without the lane-splitter stage would deliver every one of those
-- items on one lane of every output and the chest totals would be IDENTICAL.
-- What separates the two is per-lane occupancy, which sample_lanes() reads.
--
-- THE DEAD BELT BEHIND EACH TARGET IS LOAD-BEARING AND IS NOT SCENERY. A belt
-- whose ONLY input is from a side is a CURVE, and a curve carries both lanes at
-- full rate -- which would feed the rig a whole belt across both lanes and
-- quietly turn the assertion into a tautology. An unfed belt in line behind it
-- makes the target a STRAIGHT belt, and a perpendicular belt joining a straight
-- belt fills exactly one lane. That is the whole rig.
--
--     row pb-3   [chest]        [chest]
--     row pb-2   [loader S]     [loader S]
--     row pb-1     v              v
--     row pb       v          ->  ->->[P]-> sink      x=-2 side-loads here
--     row pb+1   ->->->->->->->->[P]-> sink           x=-4 side-loads here
--                x=-5  -4  -3  -2
function BUILD.lane(s, base)
  local pb = base + 3
  storage.lane_base = pb
  local out = {}
  for i = 0, 1 do put(s, PART, 0, pb + i) end

  -- row pb: (-3,pb) is fed by nothing and exists only to keep (-2,pb) straight.
  source(s, -2, pb - 3, S)
  belts(s, BELT, S, "y", pb - 1, pb - 1, -2)
  belts(s, BELT, E, "x", -3, -1, pb)

  -- row pb+1: same shape one column further out, so its feed column is clear of
  -- row pb's chain. (-5,pb+1) is the dead belt here.
  source(s, -4, pb - 3, S)
  belts(s, BELT, S, "y", pb - 1, pb, -4)
  belts(s, BELT, E, "x", -5, -1, pb + 1)

  for i = 0, 1 do
    belts(s, BELT, E, "x", 1, 3, pb + i)
    out[#out + 1] = sink(s, 4, pb + i)
  end
  return out
end

-- uio: 2->2 where all four connections are UNDERGROUND ends placed directly
-- against the parts -- the output half of a pair on the input side, the input
-- half of a pair on the output side, which is both arms of classifySide's
-- belt_to_ground_type branch.
--
-- The halves are created west to east so each pair takes the nearest partner:
-- a pair created out of order could span the part and link the wrong two ends.
function BUILD.uio(s, base)
  local out = {}
  for i = 0, 1 do put(s, PART, 0, base + i) end
  for i = 0, 1 do
    local y = base + i
    source(s, -7, y)
    belts(s, BELT, E, "x", -5, -4, y)
    put(s, UNDER, -3, y, { direction = E, type = "input" })
    put(s, UNDER, -1, y, { direction = E, type = "output" })
    put(s, UNDER, 1, y, { direction = E, type = "input" })
    put(s, UNDER, 3, y, { direction = E, type = "output" })
    belts(s, BELT, E, "x", 4, 4, y)
    out[#out + 1] = sink(s, 5, y)
  end
  return out
end

-- spio: 2->2 fed by ONE vanilla express splitter whose output face spans both
-- parts and drained by a second whose input face does. A splitter is two tiles
-- wide and the per-tile search finds it once from each cluster tile it touches,
-- so each half is its own edge -- a claim that lived only in a comment in
-- classifySide until this rig.
function BUILD.spio(s, base)
  local out = {}
  for i = 0, 1 do put(s, PART, 0, base + i) end
  for i = 0, 1 do
    local y = base + i
    source(s, -7, y)
    belts(s, BELT, E, "x", -5, -2, y)
    belts(s, BELT, E, "x", 2, 3, y)
    out[#out + 1] = sink(s, 4, y)
  end
  -- An east-facing splitter's position is the boundary between its two rows.
  put_at(s, SPLIT, -0.5, base + 1.0, { direction = E })
  put_at(s, SPLIT, 1.5, base + 1.0, { direction = E })
  return out
end

-- lio: 1->1 through LOADERS directly against the part, which is the loader arm
-- of classifySide and also the smallest network this compiler can build --
-- P=1, no stages at all, five entities.
function BUILD.lio(s, base)
  put(s, PART, 0, base)
  source(s, -2, base) -- chest at -2, loader at -1, against the part
  return { sink(s, 1, base) } -- loader at 1, against the part; chest at 2
end

-- lsio: 2->2 fed and drained through LANE SPLITTERS placed directly against the
-- parts. A lane splitter is 1x1 and directional, so it classifies exactly as a
-- transport belt does -- d == back on the way in, d == dir on the way out -- and
-- until `classifySide` named the type it was the one belt-connectable family
-- that could stand against a balancer and be silently invisible to it. This rig
-- is its own red proof: on the guest before the case existed the cluster has no
-- recognised edges at all, so it compiles to nothing and both chests stay empty.
--
-- The entity is `bbbt-lane-splitter`, data.lua's clone of the mod's own hidden
-- one, because base ships the TYPE and no buildable instance of it.
function BUILD.lsio(s, base)
  local out = {}
  for i = 0, 1 do put(s, PART, 0, base + i) end
  for i = 0, 1 do
    local y = base + i
    source(s, -7, y)
    belts(s, BELT, E, "x", -5, -2, y)
    put(s, LSPLIT, -1, y, { direction = E })
    put(s, LSPLIT, 1, y, { direction = E })
    belts(s, BELT, E, "x", 2, 3, y)
    out[#out + 1] = sink(s, 4, y)
  end
  return out
end

-- pass: the NEGATIVE. A working 2->2, plus a belt line running east along row
-- `base` -- directly over the north face of the top part and perpendicular to
-- it. classifySide keys on the belt's direction: from the part's north side
-- `dir` is north and `back` is south, and an EAST-facing belt is neither, so it
-- falls through and is not an edge. That is the incumbent's accepted limitation
-- ("a belt curving away is not an output") met from the other side, and until
-- this rig nothing asserted it.
--
-- The passing line has its own source and its own chest: the balancer must
-- deliver its own two belts exactly, and must not take a single item from a
-- line that merely goes past.
function BUILD.pass(s, base)
  local out = {}
  for i = 1, 2 do put(s, PART, 0, base + i) end
  for i = 1, 2 do
    local y = base + i
    source(s, -5, y)
    belts(s, BELT, E, "x", -3, -1, y)
    belts(s, BELT, E, "x", 1, 3, y)
    out[#out + 1] = sink(s, 4, y)
  end
  source(s, -5, base)
  belts(s, BELT, E, "x", -3, 3, base)
  out[3] = sink(s, 4, base)
  return out
end

local function build_rig(cfg, base)
  local s = cfg.other_surface and game.surfaces["bbb-m2-b"] or game.surfaces["bbb-m2-a"]
  local r = { name = cfg.name, surface = s.name, base = base, out = {}, cfg = cfg }

  if cfg.build then
    r.out = BUILD[cfg.build](s, base)
    return r
  end

  if not cfg.parts then -- the control: one uninterrupted belt
    source(s, -5, base)
    for x = -3, 3 do put(s, BELT, x, base, { direction = E }) end
    r.out[1] = sink(s, 4, base)
    return r
  end

  -- Parts FIRST, belts after, so that the belt events are what drive the
  -- compiles. Building the belts first would work too and would compile once;
  -- this way the belt-adjacency trigger is on the critical path of every rig.
  for i = 0, cfg.parts - 1 do put(s, PART, 0, base + i) end

  for i = 1, cfg.ins do
    build_input_row(s, base + i - 1, cfg.fed == nil or cfg.fed[i] == true)
  end
  for i = 1, cfg.outs do
    r.out[i] = build_output_row(s, base + i - 1, cfg.blocked and cfg.blocked[i])
  end
  return r
end

--------------------------------------------------------------------------------
-- the item-conservation rig
--
-- A 2-in 2-out balancer fed hard with NOTHING draining it, so within a few
-- hundred ticks every belt and every splitter in the hidden network is full and
-- the whole thing is stationary. Then, inside a single tick: count everything
-- countable, force a recompile, count again. Nothing else can have moved, so
-- the difference is exactly what the teardown handed back -- and the guest logs
-- what it thinks it handed back, so the two numbers have to agree.
--------------------------------------------------------------------------------

local function build_loss_rig(base)
  storage.loss_base = base
  local s = game.surfaces["bbb-m2-a"]
  for i = 0, 1 do put(s, PART, 0, base + i) end
  for i = 0, 1 do
    source(s, -5, base + i)
    for x = -3, -1 do put(s, BELT, x, base + i, { direction = E }) end
    for x = 1, 3 do put(s, BELT, x, base + i, { direction = E }) end
  end
end

-- Everything a spilled item can end up in or on. Wide enough to contain the
-- guest's spill radius: items that landed outside it would read as loss and the
-- test would be lying about which side the bug was on.
local function loss_area()
  return { { -20, storage.loss_base - 14 }, { 20, storage.loss_base + 16 } }
end

local function count_area(s, area)
  local ground, lines = 0, 0
  for _, e in pairs(s.find_entities_filtered { area = area, type = "item-entity" }) do
    if e.valid and e.stack and e.stack.valid_for_read then ground = ground + e.stack.count end
  end
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid then
      local ok, n = pcall(function() return e.get_max_transport_line_index() end)
      if ok and n then
        for i = 1, n do lines = lines + e.get_transport_line(i).get_item_count() end
      end
    end
  end
  return ground, lines
end

-- Every item this rig can be holding, on BOTH surfaces.
--
-- Counting only the visible side would not be conservation at all: the point of
-- the network is that most of the items are somewhere the player cannot see, so
-- a teardown that deleted them would look like a gain on the visible side. The
-- whole hidden surface is counted rather than one slot, because no tick passes
-- between the two samples and every other rig's network is therefore frozen.
local function count_visible_items()
  local ga, la = count_area(game.surfaces["bbb-m2-a"], loss_area())
  local gh, lh = 0, 0
  local hid = game.surfaces["bbb-hidden"]
  if hid then
    gh, lh = count_area(hid, { { -16, -16 }, { 2200, 400 } })
  end
  return ga + la + gh + lh, ga, la + lh
end

--------------------------------------------------------------------------------
-- reporting
--------------------------------------------------------------------------------

-- In TABLE order, not `pairs` order. Nothing downstream depends on it -- the
-- assertion script keys on the rig name -- but a log that reorders itself
-- between runs is a diff nobody can read, and this file is the assertion
-- surface.
local function report(tick)
  for _, name in ipairs(storage.order) do
    local r = storage.rigs[name]
    -- `block` leaves a hole in r.out; make the slot explicit rather than short.
    local n = r.cfg and (r.cfg.outs or 1) or 1
    local outs = {}
    for i = 1, n do outs[i] = tostring(chest_count(r.out[i])) end
    log(string.format("[BBB-M2] t=%d rig=%s out=[%s]", tick, r.name, table.concat(outs, " ")))
  end
end

--------------------------------------------------------------------------------
-- per-lane occupancy, for the `lane` rig
--
-- The one thing a chest total cannot see. A lane-PRESERVING network -- which is
-- what this compiler builds if the lane-splitter stage is dropped -- delivers a
-- one-lane feed as a one-lane output, and the chests fill at exactly the same
-- rate either way. So the assertion is on where the items are STANDING: both
-- transport lines of both output rows, summed over the three visible output
-- belts, at several ticks of steady flow.
--------------------------------------------------------------------------------

local function sample_lanes(tick)
  local pb = storage.lane_base
  if not pb then return end
  local s = game.surfaces["bbb-m2-a"]
  for row = 0, 1 do
    local l1, l2 = 0, 0
    for x = 1, 3 do
      local b = s.find_entities_filtered {
        position = P(x, pb + row), type = "transport-belt" }[1]
      if b and b.valid then
        l1 = l1 + b.get_transport_line(defines.transport_line.left_line).get_item_count()
        l2 = l2 + b.get_transport_line(defines.transport_line.right_line).get_item_count()
      end
    end
    log(string.format("[BBB-M2] lane t=%d out=%d left=%d right=%d", tick, row + 1, l1, l2))
  end
end

--------------------------------------------------------------------------------
-- schedule
--------------------------------------------------------------------------------

-- A full recompile, timed ACROSS THE TICK BOUNDARY, because that is where the
-- work is now.
--
-- Removing an input belt and putting it back is two complete teardown/rebuild
-- cycles of a network already at its final size -- exactly the cost a player
-- pays for laying one belt at the edge of a finished balancer. The guest
-- batches, so `belt.destroy` costs a registry update and a one-shot
-- registration and nothing else; the compile happens when `fk_on_deferred`
-- drains the queue on the FOLLOWING tick.
--
-- So the profiler is opened in the tick that mutates and closed in the tick
-- that flushes. This mod declares `better-belt-balancer` as a dependency, which
-- fixes the load order, which fixes the handler order: the guest's flush has
-- run by the time the schedule below is entered. The window therefore contains
-- one whole engine tick as well as the recompile, and the `idle tick pair` line
-- measures exactly that and nothing else. SUBTRACT IT.
--
-- (The alternative -- forcing the flush with an audit marker, as loss_check
-- does -- was measured and rejected for timing: the audit re-classifies every
-- cluster in the save, which is 16 ms of its own against a 5 ms recompile.)
local timing = {}

local function timed_begin(label)
  timing.label, timing.p = label, helpers.create_profiler()
end

local function timed_end()
  if not timing.p then return end
  timing.p.stop()
  log { "", "[BBB-M2] timing " .. timing.label .. " ", timing.p }
  timing.p = nil
end

-- The belt at the west edge of a rig's first input row: destroying it is a real
-- edge change, so the fingerprint moves and the network is rebuilt.
local function input_belt(name)
  local r = storage.rigs[name]
  if not r then return nil end
  local s = game.surfaces[r.surface]
  return s.find_entities_filtered { position = P(-1, r.base), type = "transport-belt" }[1], s, r.base
end

local function time_drop(name)
  timed_end()
  local belt = input_belt(name)
  if not belt then
    log("[BBB-M2] timing " .. name .. ": no input belt found")
    return
  end
  timed_begin(name .. " teardown+rebuild(-1 input)")
  belt.destroy { raise_destroy = true }
end

local function time_restore(name)
  timed_end()
  local r = storage.rigs[name]
  if not r then return end
  timed_begin(name .. " teardown+rebuild(full)")
  put(game.surfaces[r.surface], BELT, -1, r.base, { direction = E })
end

local function grow()
  local r = storage.rigs["regrow"]
  local s = game.surfaces[r.surface]
  local i = r.cfg.grow_to
  build_input_row(s, r.base + i - 1, true)
  log(string.format("[BBB-M2] regrow: input %d added at tick %d", i, game.tick))
end

local function loss_check()
  local before, gb, lb = count_visible_items()
  local s = game.surfaces["bbb-m2-a"]
  -- A belt arriving on the cluster's NORTH side is a genuine new edge, so the
  -- fingerprint moves and the network is torn down and rebuilt. (The west sides
  -- are already occupied; this is the one free face.)
  put(s, BELT, 0, storage.loss_base - 1, { direction = defines.direction.south })
  -- Same tick, so that "before" and "after" are one atomic sample apart and the
  -- difference can only be the teardown. The audit marker is what makes that
  -- possible now that the recompile is deferred; it costs a full
  -- re-classification of every cluster, which is why this timing line is not
  -- comparable with the two above.
  local p = helpers.create_profiler()
  audit_now(storage.loss_base)
  p.stop()
  local after, ga, la = count_visible_items()
  log(string.format("[BBB-M2] loss before=%d after=%d returned=%d", before, after, after - before))
  log(string.format("[BBB-M2] loss detail ground %d->%d lines(both surfaces) %d->%d",
    gb, ga, lb, la))
  log { "", "[BBB-M2] timing loss recompile (audit-forced, whole-save re-classification) ", p }
end

-- What the engine itself charges for the same work, so that a slow compile can
-- be attributed to the right side of the boundary.
local function raw_create_cost()
  local hid = game.surfaces["bbb-hidden"]
  if not hid then log("[BBB-M2] raw: no hidden surface") return end
  local p = helpers.create_profiler()
  for i = 1, 32 do
    hid.create_entity { name = "bbb-belt", position = P(200 + i, 200), direction = E,
                        force = "player" }
  end
  p.stop()
  log { "", "[BBB-M2] timing raw 32 create_entity on the hidden surface ", p }
  local q = helpers.create_profiler()
  local found = hid.find_entities_filtered { area = { { 190, 190 }, { 250, 210 } } }
  q.stop()
  log { "", "[BBB-M2] timing raw find_entities_filtered (" .. #found .. " hits) ", q }
  for _, e in pairs(found) do e.destroy() end
end

local SCHEDULE = {
  [30] = raw_create_cost,
  -- Each probe spans two ticks; see the timing block above.
  [598] = function() timed_begin("idle tick pair, nothing pending") end,
  [600] = function() time_drop("sat4") end,
  [602] = function() time_restore("sat4") end,
  [604] = timed_end,
  [660] = function() time_drop("sat8") end,
  [662] = function() time_restore("sat8") end,
  [664] = timed_end,
  [900] = grow,
  [1200] = loss_check,
  [1800] = function() report(1800) end,
  -- Five lane samples spread over the measurement window. One would be a
  -- snapshot of a belt that happens to have a gap in it; five is a statement
  -- about where the items live.
  [1900] = function() sample_lanes(1900) end,
  [2300] = function() sample_lanes(2300) end,
  [2700] = function() sample_lanes(2700) end,
  [3100] = function() sample_lanes(3100) end,
  [3500] = function() sample_lanes(3500) end,
  [3540] = function() report(3540) end,
  -- After the last sample, so it cannot disturb one. Every rig has been
  -- standing untouched since tick 900, so the world and the registry must agree
  -- exactly -- and `pass`'s belt line running over a cluster's north face is
  -- the reason this audit is worth taking: a classifier that decided it WAS an
  -- edge would have rebuilt that network and reported the drift.
  [3560] = function() audit_now(0) end,
}

script.on_init(function()
  -- Bands in table order, each `span` rows tall (PITCH unless it says
  -- otherwise), and the loss rig two pitches clear of the last of them. That
  -- gap is not decoration: loss_area() reaches 14 rows ABOVE loss_base, and a
  -- rig inside it would put its own items into a count that is supposed to
  -- describe one teardown.
  local base, order = 0, {}
  for _, cfg in ipairs(RIGS) do
    cfg.base = base
    base = base + (cfg.span or PITCH)
    order[#order + 1] = cfg.name
  end
  local loss_base = base + 2 * PITCH
  make_surface("bbb-m2-a", loss_base + 2 * PITCH)
  -- Surface b carries `xsurf` alone and does not need the other rigs' rows.
  local brows = 0
  for _, cfg in ipairs(RIGS) do
    if cfg.other_surface then brows = cfg.base + 2 * PITCH end
  end
  make_surface("bbb-m2-b", brows)
  storage.rigs = {}
  storage.order = order
  for _, cfg in ipairs(RIGS) do
    storage.rigs[cfg.name] = build_rig(cfg, cfg.base)
  end
  build_loss_rig(loss_base)
  -- Compile everything NOW rather than on the first tick after the save is
  -- loaded. See "forcing the flush" at the top of this file.
  audit_now(0)
  log(string.format("[BBB-M2] init complete: %d rigs", #RIGS))
end)

script.on_event(defines.events.on_tick, function(e)
  local f = SCHEDULE[e.tick]
  if f then f() end
end)
