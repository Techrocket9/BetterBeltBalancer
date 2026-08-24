-- bbb-edge-test: the edits that arrive while a network is FULL AND MOVING.
--
-- The M2 suite proves item conservation across ONE recompile of a saturated
-- 2x2. This suite asks the questions that only a long multiplayer game asks:
-- does it hold on the hundredth cycle, does it hold when two full networks are
-- torn down in one flush, does it hold when a part is placed and removed inside
-- the same tick, and does it hold when two forces become one.
--
-- THE MEASUREMENT IS GLOBAL AND IT IS ATOMIC. `count_all` totals every item on
-- the visible surface AND on the hidden surface -- on the ground, on a belt, in
-- a splitter's transport lines, in a chest -- so nothing this mod does can move
-- an item out of the count. Every source is a FINITE steel chest, so the total
-- is a conserved quantity and any fall in it is a real loss. And every count is
-- taken immediately after a `bbb-audit` marker in the SAME tick: the marker is
-- the shipped synchronous "drain the queue and compile now" trigger, and
-- `create_entity{raise_built=true}` dispatches it before it returns, so the
-- count and the teardown it is about are one sample.
--
-- The rigs, one per 30-row band:
--
--   chn   a 2-part balancer whose OUTPUTS ARE DEAD-ENDED, so its hidden network
--         fills completely and stays that way, grown by a third part and shrunk
--         again ONE HUNDRED TIMES with the count taken every cycle. Blocking the
--         outputs is what makes this a hundred teardowns of a FULL network
--         rather than of an empty one; the M2 suite proves the single case and
--         this one proves it does not drift.
--   same  a part placed and removed inside one tick (the paste-then-undo
--         shape), and then a part placed on the tick a deferred flush from the
--         previous tick is pending.
--   mrg   two saturated 2-part balancers with a one-tile gap. A part in the gap
--         BRIDGES them: two full networks come down in one flush and one comes
--         up. Removing it again is the undo, and the split path under load.
--   rot   an edge belt turned around mid-flow, twice: once silently (which
--         raises nothing at all and is what an undone rotation does to the
--         world) and once through the event path.
--   frc   two forces' balancers touching. Same-tick edits interleaved between
--         them, and then `game.merge_forces` while both networks are full.
--   det1  four clusters of one, two, three and four parts, pasted in ONE tick
--   det2  in a deliberately scrambled order. The deferred flush must compile
--         them in the order their first part arrived, and det2 must produce
--         exactly the sequence det1 did -- which is the determinism a lockstep
--         game needs from the queue.
--   aout  a SATURATED 4x4, four in and four out, with a fifth OUTPUT belt (and
--         its sink) added while it runs. This is the field report: an edge edit
--         on an operating balancer. Nothing may reach the ground, and the 4->5
--         network it becomes has to balance over the window after it.
--   ain   the same shape, with a fifth INPUT belt (and its source) added while
--         it runs.
--   shrk  the same shape, with one OUTPUT BELT REMOVED while it runs. Four
--         outputs to three LOOKS like the case where reinsertion runs out of
--         room and it is not: P = next_pow2(max(N, M)) is 4 either side of that
--         edit, so the butterfly is the same size and everything fits. Reading
--         it as evidence about shrinks is what let the second field report
--         through; `bmin` below is the one that actually shrinks.
--   bmin  THE SECOND FIELD REPORT: a saturated 2-part balancer, two in and two
--         out and DEAD-ENDED, with a third OUTPUT BELT added while it runs and
--         then MINED AGAIN. That second edit is the one no other rig here
--         performs. `shrk` below removes an output too, but from four to three,
--         and P = next_pow2(max(N, M)) is 4 either way -- so the butterfly it
--         rebuilds is the same size and everything fits, which is why this
--         suite has been reporting "the capacity fallback was not needed" and
--         a player was watching items hit the floor. Two outputs to three and
--         back crosses the boundary: P goes 2 -> 4 -> 2, the machine doubles
--         and then halves, and the reinsertion into the half genuinely
--         overflows. What the overflow is FOR is the miner's pocket, and this
--         leg is what says the quantity is real.
--   lim   THE BIGGEST BALANCER THIS MOD BUILDS, one belt short of refusing: a
--         column of thirty-two parts with a belt on both sides of every one of
--         them pointing inwards, which is sixty-four inputs and P = 64 =
--         plan.MaxPorts exactly, plus one output. A sixty-fifth input belt is
--         laid on it while it runs and then mined off again. This is the only
--         leg here that is not about items: what it asserts is that the refusal
--         happens BEFORE the teardown, so a working network survives an edit the
--         mod cannot honour instead of being demolished for nothing.
--   brdg  THE OTHER SHAPE OF THE SAME REFUSAL, and the one `lim`'s fix could
--         not reach. Two WORKING balancers of sixteen parts each -- thirty-two
--         inputs and one output apiece -- with a one-tile gap between them that
--         already carries two input belts. A part in the gap makes one cluster
--         of 66 inputs and 2 outputs, which would need P = 128. A merge's
--         teardowns belong to `AddPart`, not to compile(), so they are queued
--         before the flush even starts: without the pass at the top of flushDead
--         both balancers are demolished and come back EMPTY on the next tick.
--   frepa FAST REPLACE, forward: a saturated two-part balancer with a plain
--         belt line running past it one tile south. A PART is fast-replaced onto
--         that belt while it runs, which is the gesture the feature exists for
--         -- dropping a balancer into a belt line you already have. The line is
--         cut, the balancer becomes three in and three out, and the belt plus
--         whatever it was carrying goes where a fast replace puts it.
--   frepb FAST REPLACE, reverse, and it is the half that needs guest code. A
--         fast-replaceable group is symmetric, so a BELT can be laid on a part
--         -- and the engine raises NO EVENT AT ALL for the part it destroys.
--         Four parts in a column, fed on the top and bottom rows only so the two
--         middle ones carry no interface; a south-facing belt is laid on one of
--         them, and the registry has to notice. It also probes the refusal: a
--         part that DOES carry an interface cannot be replaced, because
--         `bbb-linked-belt` is a belt-connectable standing on the same tile.
--   ntch  THE FIELD REPORT'S SHAPE: a 2x2 balancer with one corner missing,
--         saturated and flowing, two in and two out. The missing corner is the
--         only tile in this save that is enclosed by parts and is not one, and
--         it is where a visual artifact would land -- so it is the rig the
--         placement probe below is really about.
--
-- AND WHERE THE COMPILER PUT ITS VISIBLE ENTITIES. `probe_placement` enumerates
-- every entity of one of the mod's four hidden prototypes standing on the
-- VISIBLE surface and asks one question of each: is there a registered balancer
-- part on that exact tile? The compiler's contract is that the only thing it
-- ever puts where a player can see it is an edge interface, and an edge
-- interface sits on the cluster's own tile -- so the answer must be yes for
-- every one of them, on every sample. It is a structural guarantee rather than
-- a pixel one: an interface that landed on a bare tile is a sprite nothing
-- covers, which is exactly how the tan streak would come back.
--
-- GROUND ITEMS ARE COUNTED SEPARATELY FROM THE TOTAL, and that is the point of
-- the 2026-08-02 pass. Conservation was always exact; what was wrong was the
-- PLACEMENT -- a recompile drained the hidden network onto the floor beside the
-- cluster, because the drain could not tell a recompile from a removal. Every
-- count line carries `ground=`, and for a recompile it must be zero.
--
-- ASSERTS NOTHING. test/assert-edge.py decides.

local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local LOADER = "bbbt-loader"
local AUDIT = "bbb-audit"
local E = defines.direction.east
local W = defines.direction.west
local N = defines.direction.north
local S = defines.direction.south

local PITCH = 30
local SURF = "bbb-edge"
local OTHER_FORCE = "bbb-other"

local CHN, SAME, MRG, ROT, FRC = 0, PITCH, 2 * PITCH, 3 * PITCH, 4 * PITCH
local DET1, DET2 = 5 * PITCH, 5 * PITCH + 15
-- The three recompile-under-load rigs. Each is four in and four out over EIGHT
-- parts, saturated, plus a FIFTH ROW OF PARTS CARRYING NOTHING -- which is the
-- one thing the one-belt rule forced on this suite. Two of the three exist to
-- take a fifth belt while they run, and under the rule every part of a working
-- balancer already has its belt, so a belt laid on any of them would be REFUSED
-- rather than compiled. The spare row is where a player's belt can still change
-- a balancer's port count, and it is the same trick m2's conservation rig and
-- the interactive checklist's band B arrived at independently.
local AOUT, AIN, SHRK = 6 * PITCH, 7 * PITCH, 8 * PITCH
-- The field report's own shape, in its own band because it is the only rig with
-- a hole in it.
local NTCH = 9 * PITCH
-- The port-boundary rig: a 2->2 over four parts plus an attached EDGELESS fifth,
-- and the only rig in the suite whose edge count crosses a power of two in both
-- directions.
local BMIN = 10 * PITCH
-- THE PORT LIMIT, and it is the only rig in this suite that is not about items.
-- A 2x32 BLOCK carrying SIXTY-FOUR input belts -- one on each part, which is the
-- rule -- plus one part below it carrying the single output and one spare part
-- above it carrying nothing. P = next_pow2(max(64, 1)) = 64, which is
-- plan.MaxPorts exactly, over sixty-six parts.
--
-- THE SPARE PART IS FORCED RATHER THAN FREE. Every one of the sixty-five parts
-- that carry a belt already has its one belt, so a sixty-fifth INPUT has nowhere
-- legal to land: laid against any of them it would reach the one-belt-per-part
-- bound instead, which is a different refusal in a different suite. The spare
-- part above the block is what keeps this the PORT-limit gesture.
local LIM = 11 * PITCH
local LIM_ROWS = 32
-- THE MERGE THE MOD CANNOT HONOUR. Two 2x16 BLOCKS carrying thirty-two input
-- belts each -- one per part -- with an output part of their own, so each half
-- is thirty-three parts at P = next_pow2(32) = 32 and a real network in its own
-- right. ONE TILE between them, with ONE input belt standing beside that tile
-- from t=0: the part that goes into the gap therefore brings one edge of its own
-- and takes the merged cluster to 65 inputs and 2 outputs, which is P = 128,
-- twice plan.MaxPorts.
--
-- ONE FLANKING BELT AND NOT TWO, and that is the rule rather than economy. A
-- second belt would have to stand on the bridging part's OWN east face to be an
-- edge of it at all, and it would then take that tile to two belts as well as
-- taking the merged cluster to sixty-six inputs -- a shape that is illegal
-- twice. Measured: compile() asks the port bound first and returns, so the
-- refusal still reads as a port refusal and the one-belt break is never
-- reported. One belt keeps the rig exactly the shape this leg names, and the
-- input count in test/assert-edge.py is what says so.
--
-- Sixteen and sixteen is the cheapest shape that reaches it: under the rule a
-- cluster of C parts has at most C edges, so sixty-five needs sixty-five parts
-- carrying belts however they are arranged, and splitting them evenly keeps each
-- HALF at P = 32 instead of one of them being a second 64-port network.
local BRDG = LIM + LIM_ROWS + 12
local BRDG_HALF = 16
local BRDG_FED = 3
-- FAST REPLACE, two rigs in one band, and they are the only rigs in this suite
-- that are BUILT MID-RUN. Everything above is standing from on_init and every
-- baseline in test/assert-edge.py is a statement about that world; these two are
-- placed after the last of those assertions has been made, so not one of them
-- moves. Their SOURCE CHESTS and their stock are created in on_init all the
-- same, because `count_all` is a conserved quantity and inserting twenty-four
-- thousand items into it mid-run would read as this mod minting matter.
--
--   frepa  the forward gesture: a saturated 2->2 over four parts with a PLAIN
--          BELT LINE running east one tile below it that ENDS on the tile the
--          part is dropped onto. From the parts above, that belt is neither
--          `dir` nor `back`, so it is not an edge at all (M2's `pass` shape) --
--          until a part is fast-replaced onto its last tile, at which point the
--          line feeds the balancer and it is three in and two out.
--
--          THE LINE ENDS THERE RATHER THAN RUNNING PAST, and that is the one
--          belt rule again: a part dropped into the MIDDLE of a line takes the
--          belt behind it as an input and the belt ahead as an output, which is
--          two belts on one tile and is refused. Dropping a part into a line is
--          a gesture that works at the line's end now.
--   frepb  the reverse: a 1->1 row of two parts, a THREE-PART NECK, and another
--          1->1 row -- nine parts in a T-less column whose middle three carry no
--          belt at all. The belt that replaces the middle of the neck becomes an
--          edge of the part above it and of the part below it, so BOTH of those
--          have to be otherwise edgeless or the split would hand one half a part
--          with two belts and that half would be refused instead of built. The
--          neck is also what makes the split a split: a second column of parts
--          beside the target would keep the cluster connected around it.
local FREP = BRDG + 2 * BRDG_HALF + 12
local FREPB = FREP + 8
local ROWS = FREPB + 12

-- Everything the compiler is allowed to put on the visible surface. All four are
-- named rather than just the linked belt: the assertion is about the CONTRACT
-- ("only edge interfaces are visible, and they sit on cluster tiles"), so a
-- hidden splitter appearing out there has to fail rather than go uncounted.
local OURS = { "bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter" }

local STOCK = 6000

-- The hidden surface's slot grid is 32x72 per slot, 64 columns. Twenty columns
-- and four rows is eighty slots, which is far more than this save can use and
-- far less than a query over the whole address space.
local HIDDEN_AREA = { { 0, 0 }, { 640, 288 } }
local VIS_AREA = { { -24, -18 }, { 28, ROWS + 18 } }

-- The determinism rigs: {x of the west part column, rows}, and the order their
-- first parts are placed. Each is a 2-wide block with its input belt at x-1 and
-- its output at x+2, and the columns are five apart so that no rig's output belt
-- is adjacent to the next rig's input.
local DET_SHAPES = { { 8, 1 }, { 13, 2 }, { 18, 3 }, { 23, 4 } }
local DET_ORDER = { 3, 1, 4, 2 }

--------------------------------------------------------------------------------
-- world helpers
--------------------------------------------------------------------------------

local function P(x, y) return { x + 0.5, y + 0.5 } end
local function surf() return game.surfaces[SURF] end

local function put(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = "player", raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-edge-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

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

local function kill(s, x, y, filter)
  local e = at(s, x, y, filter)
  if e and e.valid then e.destroy { raise_destroy = true } end
end

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

  for _, e in pairs(s.find_entities_filtered { area = VIS_AREA }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = VIS_AREA }
  local tiles = {}
  for x = -24, 28 do
    for y = -18, ROWS + 18 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  s.set_tiles(tiles, true, false, false, false)
  return s
end

local function source(s, y, force)
  local c = s.create_entity {
    name = "steel-chest", position = P(-6, y), force = force or "player",
  }
  c.get_inventory(defines.inventory.chest).insert { name = "iron-plate", count = STOCK }
  put(s, LOADER, -5, y, { direction = E, type = "output", force = force or "player" })
end

local function sink(s, y, force)
  put(s, LOADER, 5, y, { direction = E, type = "input", force = force or "player" })
  return s.create_entity {
    name = "steel-chest", position = P(6, y), force = force or "player",
  }
end

-- ONE BELT PER PART, so a row is TWO parts: the west one carries the input and
-- the east one the output.
--
--   x=-6 source chest   -5 loader   -4..-1 belts   0 WEST PART   1 EAST PART
--   x=2..4 belts        5 sink loader              6 chest
local function feed(s, y, force)
  source(s, y, force)
  for x = -4, -1 do put(s, BELT, x, y, { direction = E, force = force or "player" }) end
  for x = 2, 4 do put(s, BELT, x, y, { direction = E, force = force or "player" }) end
  return sink(s, y, force)
end

-- The input side alone, for a row whose output is somewhere other than due east.
local function feed_in(s, y, force)
  source(s, y, force)
  for x = -4, -1 do put(s, BELT, x, y, { direction = E, force = force or "player" }) end
end

-- ... and the output side alone.
local function drain_out(s, y, force)
  for x = 2, 4 do put(s, BELT, x, y, { direction = E, force = force or "player" }) end
  return sink(s, y, force)
end

--------------------------------------------------------------------------------
-- counting
--
-- Everything, on both surfaces. An item this mod can lose is an item that left
-- this total, and there is nowhere else for one to be.
--------------------------------------------------------------------------------

-- Returns TWO numbers: everything, and the part of it that is lying on the
-- ground. The second is what the recompile policy is about -- `spill_item_stack`
-- puts an item on a belt when there is one under it and on the floor otherwise,
-- so a spill that lands on the output belts is invisible to the total and very
-- visible to a player.
local function count_area(s, area)
  if not (s and s.valid) then return 0, 0 end
  local total, ground = 0, 0
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid then
      if e.type == "item-entity" then
        if e.stack and e.stack.valid_for_read then
          total = total + e.stack.count
          ground = ground + e.stack.count
        end
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
  return total, ground
end

local function count_all()
  local a, ag = count_area(surf(), VIS_AREA)
  local b, bg = count_area(game.surfaces["bbb-hidden"], HIDDEN_AREA)
  return a + b, ag + bg
end

-- audit_and_count is the atomic sample: the marker drains the queue and
-- re-classifies inside this very dispatch, so the count that follows describes
-- the world the audit just reported on.
local function audit_and_count(tag)
  log(string.format("[BBB-EDGE] mark tag=%s", tag))
  surf().create_entity {
    name = AUDIT, position = P(24, 4), force = "player", raise_built = true,
  }
  local total, ground = count_all()
  log(string.format("[BBB-EDGE] count tag=%s total=%d ground=%d", tag, total, ground))
end

--------------------------------------------------------------------------------
-- WHERE the compiler put its visible entities
--
-- The visual half of the mod rests on one structural fact: the only thing the
-- compiler ever creates on a surface a player looks at is an edge interface,
-- and an edge interface stands on a tile of the cluster itself -- under a
-- balancer part's own sprite, which is one opaque tile. Anything of ours on a
-- bare tile is a picture with nothing over it.
--
-- So: enumerate every entity of our four hidden prototypes on the visible
-- surface, and require a registered part on that exact tile. The tile is taken
-- with `floor`, which is exact for the 1x1 entities involved (both sit at a
-- tile centre) and would round a 2-tile splitter's centre onto one of its tiles
-- -- generous in the direction that could only hide a failure, never invent one.
--------------------------------------------------------------------------------

local function probe_placement(tag)
  local s = surf()
  local parts, nparts = {}, 0
  for _, e in pairs(s.find_entities_filtered { area = VIS_AREA, name = PART }) do
    if e.valid then
      parts[math.floor(e.position.x) .. ":" .. math.floor(e.position.y)] = true
      nparts = nparts + 1
    end
  end
  local total, off, strays = 0, 0, {}
  for _, e in pairs(s.find_entities_filtered { area = VIS_AREA, name = OURS }) do
    if e.valid then
      total = total + 1
      local tx, ty = math.floor(e.position.x), math.floor(e.position.y)
      if not parts[tx .. ":" .. ty] then
        off = off + 1
        strays[#strays + 1] = string.format("%s@%d,%d", e.name, tx, ty)
      end
    end
  end
  table.sort(strays)
  log(string.format("[BBB-EDGE] place tag=%s ours=%d onpart=%d offpart=%d parts=%d",
    tag, total, total - off, off, nparts))
  for i = 1, math.min(#strays, 8) do
    log(string.format("[BBB-EDGE] stray tag=%s %s", tag, strays[i]))
  end
end

--------------------------------------------------------------------------------
-- P1: a hundred cycles of add-part / remove-part on a network that is full
--------------------------------------------------------------------------------

-- Sixteen ticks, and the number is chosen rather than convenient: a rebuild
-- starts on an EMPTY network and refills it out of the backed-up input belts,
-- so the removal has to be late enough that what comes down is full again.
local CHN_ITERS, CHN_PERIOD = 100, 16

local function chn_step(phase, i)
  local s = surf()
  if phase == 0 then
    put_soft(s, PART, 0, CHN + 2)
  elseif phase == 10 then
    kill(s, 0, CHN + 2, { name = PART })
  elseif phase == 13 then
    audit_and_count("chn" .. i)
  end
end

--------------------------------------------------------------------------------
-- the one-off scenarios
--------------------------------------------------------------------------------

-- A part placed and removed inside ONE dispatch chain. The registry gains a
-- node and gives it straight back; the flush on the next tick is handed a root
-- whose slot has been freed and must drop it rather than compile it.
local function p_same_tick()
  local s = surf()
  log("[BBB-EDGE] sametick begin")
  put_soft(s, PART, 0, SAME + 2)
  kill(s, 0, SAME + 2, { name = PART })
  log("[BBB-EDGE] sametick end")
end

-- A part placed on the tick BEFORE, and another on the tick the deferred flush
-- for it lands. Whether this mod's on_tick runs before or after the guest's
-- one-shot is engine order and is deterministic either way; what must not
-- happen is a network built under a root that has stopped being one.
local function p_pending_a()
  put_soft(surf(), PART, 0, SAME + 2)
  log("[BBB-EDGE] pending first")
end

local function p_pending_b()
  put_soft(surf(), PART, 0, SAME + 3)
  log("[BBB-EDGE] pending second")
end

local function p_pending_undo()
  local s = surf()
  kill(s, 0, SAME + 2, { name = PART })
  kill(s, 0, SAME + 3, { name = PART })
end

-- The bridge: one part in the gap between two SATURATED balancers. Two whole
-- networks are torn down in one flush and one is built.
local function p_merge()
  log("[BBB-EDGE] merge begin")
  put_soft(surf(), PART, 0, MRG + 2)
end

local function p_unmerge()
  log("[BBB-EDGE] split begin")
  kill(surf(), 0, MRG + 2, { name = PART })
end

-- An edge belt turned around with `entity.direction = …`, which raises NOTHING.
-- The audit is the documented recovery, and it must SEE the drift before it
-- repairs it.
local function p_rot_silent()
  local b = at(surf(), -1, ROT, { type = "transport-belt" })
  b.direction = W
  log("[BBB-EDGE] rot silent")
end

local function p_rot_restore()
  local b = at(surf(), -1, ROT, { type = "transport-belt" })
  b.direction = E
  log("[BBB-EDGE] rot restored")
end

-- The same edge, through the event path: the belt is mined and a belt facing
-- the other way is laid on the same tile inside the same tick.
local function p_rot_event()
  local s = surf()
  kill(s, -1, ROT + 1, { type = "transport-belt" })
  put_soft(s, BELT, -1, ROT + 1, { direction = W })
  log("[BBB-EDGE] rot event flipped")
end

local function p_rot_event_back()
  local s = surf()
  kill(s, -1, ROT + 1, { type = "transport-belt" })
  put_soft(s, BELT, -1, ROT + 1, { direction = E })
  log("[BBB-EDGE] rot event restored")
end

-- Two forces editing in one tick, alternating. Their parts touch and must stay
-- two balancers; each edit must reach only its own.
-- A WHOLE ROW EACH, because a row is two parts under the one-belt rule: a west
-- part taking the new input and an east part giving the new output. The old
-- version added one part and put a belt on both sides of it, which is the shape
-- the rule forbids.
local function p_forces_interleaved()
  local s = surf()
  log("[BBB-EDGE] forces interleaved begin")
  put_soft(s, PART, 0, FRC - 1)
  put_soft(s, PART, 0, FRC + 4, { force = OTHER_FORCE })
  put_soft(s, PART, 1, FRC - 1)
  put_soft(s, PART, 1, FRC + 4, { force = OTHER_FORCE })
  put_soft(s, BELT, -1, FRC - 1, { direction = E })
  put_soft(s, BELT, -1, FRC + 4, { direction = E, force = OTHER_FORCE })
  put_soft(s, BELT, 2, FRC - 1, { direction = E })
  put_soft(s, BELT, 2, FRC + 4, { direction = E, force = OTHER_FORCE })
  log("[BBB-EDGE] forces interleaved end")
end

-- And then the two forces become one, while both networks are full. No
-- per-entity event is raised for any of it.
--
-- What this leg CANNOT reach is the other half of the merge: a player mining a
-- source-force part in the same tick, whose claim names a force the merge is
-- about to destroy. A headless --create has no player, so player_index is 0 on
-- every removal any suite can produce and the claim list is empty in all seven.
-- That half is proved by `go test ./carry/` instead, which `make check` runs --
-- see guest/go/carry/claims_test.go.
local function p_forces_merge()
  log("[BBB-EDGE] forces merge begin")
  game.merge_forces(OTHER_FORCE, "player")
  log("[BBB-EDGE] forces merge end")
end

-- Four clusters in one tick, their parts interleaved in a scrambled order. The
-- flush must compile them in the order their FIRST part arrived.
local function det_paste(base, tag)
  local s = surf()
  log(string.format("[BBB-EDGE] det-begin tag=%s", tag))
  for step = 1, 4 do
    for _, k in ipairs(DET_ORDER) do
      local cx, n = DET_SHAPES[k][1], DET_SHAPES[k][2]
      if step <= n then
        put_soft(s, PART, cx, base + step - 1)
        put_soft(s, PART, cx + 1, base + step - 1)
      end
    end
  end
  for _, k in ipairs(DET_ORDER) do
    local cx, n = DET_SHAPES[k][1], DET_SHAPES[k][2]
    for r = 0, n - 1 do
      put_soft(s, BELT, cx - 1, base + r, { direction = E })
      put_soft(s, BELT, cx + 2, base + r, { direction = E })
    end
  end
  log(string.format("[BBB-EDGE] det-end tag=%s", tag))
end

local function det_flushed(tag)
  log(string.format("[BBB-EDGE] det-flushed tag=%s", tag))
end

local function det_clear(base)
  local s = surf()
  for _, shape in ipairs(DET_SHAPES) do
    local cx, n = shape[1], shape[2]
    for r = 0, n - 1 do
      kill(s, cx, base + r, { name = PART })
      kill(s, cx + 1, base + r, { name = PART })
      kill(s, cx - 1, base + r, { type = "transport-belt" })
      kill(s, cx + 2, base + r, { type = "transport-belt" })
    end
  end
end

-- A balancer part built ON THE HIDDEN SURFACE, which is what an area clone or a
-- careless script can do and what nothing else in the suite reaches. The guest
-- must refuse to register it: a cluster there would put its bounding box inside
-- the slot grid, and a teardown spills a network's items beside the CLUSTER --
-- which would be a surface no player can ever reach.
local function p_hidden_part()
  local h = game.surfaces["bbb-hidden"]
  log("[BBB-EDGE] hidden-part begin")
  h.create_entity {
    name = PART, position = P(1000, 500), force = "player", raise_built = true,
  }
  local e = h.find_entities_filtered { position = P(1000, 500), name = PART }[1]
  log(string.format("[BBB-EDGE] hidden-part placed=%s", tostring(e ~= nil)))
  if e then e.destroy { raise_destroy = true } end
end

--------------------------------------------------------------------------------
-- The three recompile-under-load rigs: an edge edit on an OPERATING balancer.
--
-- All three are four-in four-out over EIGHT parts and saturated. `aout` and
-- `ain` carry a ninth and tenth part below them that hold nothing at all, and
-- the belt each of those legs adds goes THERE: under the one-belt rule every
-- part of a working balancer already has its belt, so a belt laid against one of
-- them would be refused rather than compiled and the leg would measure the wrong
-- thing entirely. What changes is only what the edit does to the network:
--
--   aout  4->5 outputs: the network is rebuilt over eight ports instead of four
--   ain   5->4 inputs:  the same, from the other side
--   shrk  4->3 outputs: fewer ports, and P stays 4 -- so the network is NOT
--         smaller and nothing overflows. See the header, and `bmin`
--------------------------------------------------------------------------------

-- The fifth OUTPUT, with a loader and a chest to drain it, so the rig after the
-- edit is a real 4->5 balancer rather than one with a blocked port. It leaves
-- SOUTH off the spare row's west part, which is the only tile of this cluster
-- with a free face.
local function p_add_output()
  local s = surf()
  log("[BBB-EDGE] add-out begin")
  local c = s.create_entity {
    name = "steel-chest", position = P(0, AOUT + 7), force = "player",
  }
  put_soft(s, LOADER, 0, AOUT + 6, { direction = S, type = "input" })
  put_soft(s, BELT, 0, AOUT + 5, { direction = S })
  storage.out.aout[#storage.out.aout + 1] = c
  log("[BBB-EDGE] add-out end")
end

-- The fifth INPUT: one south-facing belt, and NOTHING FEEDING IT. The edit under
-- test is the edge list going 4->5 inputs and the network being rebuilt over
-- eight ports; a source chest would put 4,800 new items into a total whose whole
-- job is to be conserved, and the classifier does not care whether an input belt
-- is carrying anything.
local function p_add_input()
  log("[BBB-EDGE] add-in begin")
  put_soft(surf(), BELT, 0, AIN + 5, { direction = N })
  log("[BBB-EDGE] add-in end")
end

-- One OUTPUT BELT mined off a running 4x4. Four in, three out afterwards.
local function p_remove_output()
  log("[BBB-EDGE] shrink begin")
  kill(surf(), 2, SHRK, { type = "transport-belt" })
  log("[BBB-EDGE] shrink end")
end

-- THE OTHER HALF OF THE POLICY, in the same suite so the two cannot drift
-- apart: every part of the shrk balancer mined, which DISSOLVES the cluster.
-- There is no successor network for the drained items to go into and the
-- machine has genuinely been removed, so they must come back to the world --
-- onto the belts under them where there is room, and onto the ground where
-- there is not, which is what a mined machine does in vanilla.
local function p_remove_cluster()
  local s = surf()
  log("[BBB-EDGE] remove begin")
  for r = 0, 3 do
    kill(s, 0, SHRK + r, { name = PART })
    kill(s, 1, SHRK + r, { name = PART })
  end
  log("[BBB-EDGE] remove end")
end

-- TAKING A BALANCER APART BY HAND: ONE PART PER TICK, WHICH IS THE FIELD REPORT.
--
-- `p_remove_cluster` above mines every part in ONE tick, so it is one teardown
-- of one network and one removal. A PLAYER cannot do that: they mine a part,
-- the machine recompiles SMALLER, they mine the next one, and so on down to the
-- dissolve. Each of those shrinks is a recompile into a network with fewer ports
-- and less line, so each one hands back less than it drained -- and until
-- 2026-08-02 the difference went on the floor, because only the final DISSOLVE
-- recorded the miner.
--
-- Measured on this rig one part per tick: the shrinks are where the items are,
-- and the dissolve gets the dregs. That quantity is what a player now receives
-- instead of the floor, and the assertion is that it is a REAL quantity -- a leg
-- where the shrinks happened to fit would satisfy every other check in this
-- suite and would prove nothing about the thing that was fixed.
--
-- Headless has no player, so the items still spill here and every number in this
-- suite is what it was. What is pinned is the MAGNITUDE; what the arithmetic
-- does with it is pinned by the insert probe below.
-- TEN PARTS AND TEN MINES, because a 4x4 is eight parts under the one-belt rule
-- and this rig carries the spare row as well.
--
-- THE ORDER IS THE SPARE ROW AND THEN ROW BY ROW, WEST PART THEN EAST PART, and
-- it is chosen rather than convenient. Every prefix of it leaves a CONNECTED
-- cluster, so no step is a split -- a split is a different carry decision and a
-- different measurement. And every step but the last two leaves a machine with
-- at least one input and one output, so it is a real shrink into a smaller
-- butterfly: N and M come down together, P crosses 4 -> 2 and 2 -> 1 on the way,
-- and each rebuild has less room than the teardown it followed. Taking a whole
-- column first instead would strip every input in four mines and leave a
-- five-output machine with nothing feeding it, which is a balancer dying rather
-- than a balancer shrinking.
--
-- THE NINTH STEP HAS NO NETWORK AND THAT IS STRUCTURAL. One part can carry one
-- belt, so the last part standing has an input or an output and never both --
-- which `plan.Build` reads as a legitimate half-built cluster, not an unbuilt
-- one. Under the old rule the last part carried both and the leg never saw it.
local HAND_ORDER = { { 0, 4 }, { 1, 4 } }
for r = 0, 3 do
  HAND_ORDER[#HAND_ORDER + 1] = { 0, r }
  HAND_ORDER[#HAND_ORDER + 1] = { 1, r }
end

local function p_hand_mine(i)
  return function()
    local xy = HAND_ORDER[i]
    log("[BBB-EDGE] hand mine " .. i)
    kill(surf(), xy[1], AOUT + xy[2], { name = PART })
  end
end

-- THE SECOND FIELD REPORT: AN OUTPUT BELT PLACED ON A RUNNING BALANCER AND THEN
-- MINED AGAIN.
--
-- `bmin` is two parts, two in, two out and dead-ended, so it is full and stays
-- full. Adding a SOUTH-facing output belt off the bottom part takes it to two in
-- and three out, and P = next_pow2(max(N, M)) goes 2 -> 4: the butterfly roughly
-- triples and the recompile has room for everything it drained, exactly as the
-- `aout` rig does. Mining that belt again takes it back to P = 2, and THAT is
-- the edit nothing else in this suite performs -- the machine halves, the
-- reinsertion legitimately runs out of room, and carry.go's fourth decision
-- ("what does not fit") sends the difference somewhere.
--
-- Where it goes is the whole point. A player caused this removal, so the
-- overflow is theirs before the floor's -- and headless has no player, so it
-- lands on the floor here and is COUNTED. The assertion is a floor rather than a
-- ceiling, for the same reason the by-hand leg's is: a shrink that happened to
-- fit would satisfy every other check in this suite and would say nothing about
-- the quantity the pocket redirects. Which is precisely what `shrk` did.
local function p_bmin_add()
  log("[BBB-EDGE] bmin-add begin")
  put_soft(surf(), BELT, 0, BMIN + 3, { direction = S })
  log("[BBB-EDGE] bmin-add end")
end

local function p_bmin_remove()
  log("[BBB-EDGE] bmin-remove begin")
  kill(surf(), 0, BMIN + 3, { type = "transport-belt" })
  log("[BBB-EDGE] bmin-remove end")
end

--------------------------------------------------------------------------------
-- THE SIXTY-FIFTH BELT: the edit the mod cannot honour
--
-- `lim` is a 2x32 BLOCK of parts with one input belt against each of them --
-- sixty-four inputs, which is the rule -- plus one part below the block carrying
-- the single output and one SPARE PART above it carrying nothing at all.
-- P = next_pow2(max(64, 1)) = 64, which is plan.MaxPorts exactly, so the network
-- is the largest one this mod builds and it is a real one: 1,026 entities on the
-- hidden surface, compiled during `--create` like everything else here.
--
-- Then one more input belt goes against the SPARE PART, and P would have to be
-- 128. The spare part is what makes that possible: every other part in the rig
-- has its one belt, so a sixty-fifth belt laid anywhere else would break the
-- one-belt-per-part rule instead and this leg would be measuring the other
-- bound.
--
-- WHAT USED TO HAPPEN, and it is the whole reason this leg exists: compile()
-- classified the edges, saw the fingerprint move, tore the network down, and
-- only then asked plan.Build whether the new shape fit. So laying that belt
-- demolished a working sixty-four-port balancer, built nothing, and spilled its
-- entire contents on the floor beside it. Verified on the unfixed guest, which
-- is what makes this leg evidence: see test/assert-edge.py's LIM section.
--
-- THREE INPUTS ARE FED AND NO MORE. This rig is about the refusal, not about
-- throughput -- sixty-four saturated sources would be sixty-four more chests in
-- a total whose whole job is to be conserved, and the assertion below is that
-- the machine KEPT RUNNING across the refusal, which a trickle proves as well as
-- a flood. Sixty-three of its output ports dead-end (M is 1), so it back-fills
-- for the whole run and the share reaching the live port climbs as they do --
-- which is why the two measurement windows are the same length and are compared
-- as a ratio rather than against a constant.
--------------------------------------------------------------------------------

local LIM_FED = 3

-- The sixty-fifth input: one belt below the spare part pointing NORTH, into it.
-- Nothing feeds it, exactly as `ain`'s fifth input is unfed -- the classifier
-- does not care whether an input belt is carrying anything, and the edit under
-- test is the edge COUNT going 64 -> 65.
local function p_lim_add()
  log("[BBB-EDGE] lim-add begin")
  put_soft(surf(), BELT, 0, LIM + LIM_ROWS + 1, { direction = N })
  log("[BBB-EDGE] lim-add end")
end

-- ... and mined again. The edge list goes back to sixty-four, which is the
-- fingerprint the guest's netInfo already holds, so the compile is a SKIP: the
-- network that was never torn down is never rebuilt either.
local function p_lim_remove()
  log("[BBB-EDGE] lim-remove begin")
  kill(surf(), 0, LIM + LIM_ROWS + 1, { type = "transport-belt" })
  log("[BBB-EDGE] lim-remove end")
end

--------------------------------------------------------------------------------
-- THE BRIDGE THAT WOULD BE OVER THE LIMIT
--
-- `lim` is one cluster growing an edge it cannot have. `brdg` is the other shape
-- of the same refusal and it is the one the fix for `lim` could not reach: two
-- WORKING balancers with a one-tile gap, and a part in the gap.
--
-- The difference is whose teardown it is. An edge edit is refused in front of
-- `teardownForRebuild`, which is the teardown compile() does to itself. A merge's
-- teardowns belong to `AddPart`, which marks BOTH predecessors' roots dead
-- before the flush starts -- so until 2026-08-05 flushDead demolished two
-- running balancers and flushLive then discovered that what they became could
-- not be built. Both came back on the next tick EMPTY, with everything they had
-- been holding in a heap on the floor between them.
--
-- Each half is thirty-two inputs and one output (P = 32); together with the ONE
-- belt already standing beside the gap tile they are 65 inputs and 2 outputs, so
-- P would have to be 128. One belt and not two: a second would put two on the
-- bridging part's own tile, and the merge would then be refused for the
-- one-belt-per-part bound before it ever reached the port limit. Three of each
-- half's thirty-two inputs are fed, for the same reason `lim` feeds three: this
-- leg is about the refusal, not about throughput, and thirty-two more source
-- chests would be thirty-two more numbers in a total whose whole job is to be
-- conserved.
--------------------------------------------------------------------------------

-- The bridging part. The one input belt beside its tile has been standing since
-- t=0, so this one placement takes the merged cluster from 64 edges to 65 --
-- the smallest step over the limit this rig can make.
local function p_brdg_add()
  log("[BBB-EDGE] brdg-add begin")
  put_soft(surf(), PART, 0, BRDG + BRDG_HALF)
  log("[BBB-EDGE] brdg-add end")
end

-- ... and mined off again, which is what the revert does for a player. The
-- cluster splits back into the two it was, each component re-roots at its
-- smallest node id -- which is the root it already had -- and each half's
-- fingerprint matches the netInfo it never lost.
local function p_brdg_remove()
  log("[BBB-EDGE] brdg-remove begin")
  kill(surf(), 0, BRDG + BRDG_HALF, { name = PART })
  log("[BBB-EDGE] brdg-remove end")
end

local function brdg_report(tag)
  local n = { 0, 0 }
  for i, c in ipairs { storage.brdg_a, storage.brdg_b } do
    if c and c.valid then
      for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
        n[i] = n[i] + item.count
      end
    end
  end
  log(string.format("[BBB-EDGE] brdg tag=%s tick=%d a=%d b=%d", tag, game.tick, n[1], n[2]))
end

local function lim_report(tag)
  local c = storage.lim_chest
  local n = 0
  if c and c.valid then
    for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
      n = n + item.count
    end
  end
  log(string.format("[BBB-EDGE] lim tag=%s tick=%d delivered=%d", tag, game.tick, n))
end

-- THE INSERT PROBE: the miner's-pocket arithmetic, asked of a steel chest.
--
-- `insert` is a member of LuaControl and a chest is a LuaControl, so the call
-- the pocket makes to a player can be made to a chest -- same member id, same
-- signature, same tier-2 encode of the same table -- in a headless run with no
-- player anywhere. The guest reports what it asked for, what the engine said it
-- took and what the entity holds afterwards; this reads the same three numbers
-- back from Lua so the guest's own arithmetic is checked against something that
-- did not come through the boundary.
--
-- The marker is deferred by the guest and runs inside the next flush, which is
-- where the pocket runs, so the report is one tick behind the placement.
local PROBE_AT = { x = 24, y = 10 }
local PROBE_WANT = {
  { "iron-gear-wheel", 50 }, { "iron-plate", 37 },
  { "copper-cable", 23 }, { "steel-chest", 7 },
}

local function p_insert_probe()
  local s = surf()
  log("[BBB-EDGE] insert-probe begin")
  storage.probe_chest = s.create_entity {
    name = "steel-chest", position = P(PROBE_AT.x, PROBE_AT.y), force = "player",
  }
  s.create_entity {
    name = "bbb-insert-probe", position = P(PROBE_AT.x, PROBE_AT.y),
    force = "player", raise_built = true,
  }
end

local function p_insert_probe_read()
  local c = storage.probe_chest
  for _, want in ipairs(PROBE_WANT) do
    log(string.format("[BBB-EDGE] insert-probe-lua %s want=%d held=%d",
      want[1], want[2], (c and c.valid) and c.get_item_count(want[1]) or -1))
  end
  -- AND THEN THE CHEST GOES, CONTENTS AND ALL. The probe's items are MINTED by
  -- the probe -- they came from nowhere, which is the point of asking for a
  -- known count -- and this suite's whole instrument is a global total that is
  -- conserved. Leaving 117 invented items inside the counted area would read as
  -- the mod creating them.
  if c and c.valid then c.destroy() end
  storage.probe_chest = nil
end

-- THE PROBE THAT SAYS WHY THE REMOVAL ABOVE IS THE ONLY HALF THAT CAN BE TESTED.
--
-- A dissolve started by a PLAYER hands the drained network to that player's
-- inventory before anything reaches the ground (carry.go, "the beneficiary"),
-- and this suite cannot reach that path. Two walls, and this probe measures both
-- rather than asserting them from the documentation:
--
--   * a headless `--create` has no players at all, so `game.get_player(1)` is
--     nil and the beneficiary would fall back to the spill even if one were set;
--   * `on_player_mined_entity` is not one of the events `script.raise_event`
--     will raise -- LuaBootstrap carries a `raise_*` helper for each of the
--     eleven that can be, and there is none for this one.
--
-- The entity handed to the raise is a belt FOUR tiles from the nearest part, so
-- the probe is inert whichever way it goes: a vanish event outside the two-tile
-- neighbour gate is rejected in-guest before anything is looked up, and the raise
-- does not destroy anything either way.
local function p_probe_player_mine()
  local s = surf()
  local e = at(s, -4, NTCH, { type = "transport-belt" })
  local ok, err = pcall(script.raise_event, defines.events.on_player_mined_entity,
    { player_index = 1, entity = e })
  log(string.format("[BBB-EDGE] player-mine-raise ok=%s err=%s",
    tostring(ok), (tostring(err):gsub("%s+", " "))))
  log(string.format("[BBB-EDGE] player-resolve p1=%s players=%d",
    tostring(game.get_player(1) ~= nil), #game.players))
end

-- THE OPERATOR SEAM: the console command and the remote interface the guest
-- registers in `init` (guest/go/commands.go).
--
-- A console command CANNOT be triggered from script -- 2.0.77 has no
-- `commands.run_command` -- so the command leg is asserted as far as Factorio's
-- OWN registry and no further: `commands.commands` is the engine's table, not
-- the mod's claim about itself, so a name in it is the engine confirming the
-- registration reached it.
--
-- The remote leg is what drives the handler end to end, and it is evidence about
-- the command leg because ONE export serves both: `fk_on_call` is id-dispatched
-- and has no branch that can tell which door a call came through. What comes
-- back is the audit's own cluster count, so a wrong answer here is a wrong
-- answer to the diagnostic a player would be running.
local function p_probe_operators()
  log(string.format("[BBB-EDGE] command registered=%s",
    tostring(commands.commands["bbb-audit"] ~= nil)))
  local methods = {}
  for m in pairs(remote.interfaces["better-belt-balancer"] or {}) do
    methods[#methods + 1] = m
  end
  table.sort(methods)
  log(string.format("[BBB-EDGE] remote iface methods=%s",
    table.concat(methods, ",")))
  local ok, got = pcall(remote.call, "better-belt-balancer", "audit")
  log(string.format("[BBB-EDGE] remote audit ok=%s clusters=%s",
    tostring(ok), tostring(got)))
end

--------------------------------------------------------------------------------
-- FAST REPLACE
--
-- `bbb-balancer-part` carries `fast_replaceable_group = "transport-belt"`, so a
-- part held over a belt replaces it the way a splitter does. The group is
-- symmetric, which buys the reverse for free and is why the guest has to watch
-- for it: measured on 2.0.77, a belt laid on a part destroys that part and
-- raises NO EVENT of any kind, so without guest/go/fastreplace.go the registry
-- keeps a tile it calls a part which is holding somebody's belt.
--
-- Both rigs are built at CHN_END + 2680 rather than in on_init -- see the FREP
-- comment at the top of this file.
--------------------------------------------------------------------------------

-- The source half of `feed`, minus the chest: the chests are made in on_init so
-- their stock is inside the conserved total from t=0.
-- The input side minus the chest: the frep chests are made in on_init so their
-- stock is inside the conserved total from t=0, and `feed_in` would try to build
-- a second one on the same tile.
local function frep_in(s, y)
  put(s, LOADER, -5, y, { direction = E, type = "output" })
  for x = -4, -1 do put(s, BELT, x, y, { direction = E }) end
end

local function frep_feed(s, y)
  frep_in(s, y)
  return drain_out(s, y)
end

local function p_frep_build()
  local s = surf()
  log("[BBB-EDGE] frep-build begin")
  for r = 0, 1 do put(s, PART, 0, FREP + r); put(s, PART, 1, FREP + r) end
  storage.out.frepa = { frep_feed(s, FREP), frep_feed(s, FREP + 1) }
  -- The belt line, x = -4 through 0, ENDING on the tile the part is dropped
  -- onto. It is an edge of nothing until then: an east-facing belt on a part's
  -- SOUTH face is neither `dir` nor `back` and falls through classifySide.
  put(s, LOADER, -5, FREP + 2, { direction = E, type = "output" })
  for x = -4, 0 do put(s, BELT, x, FREP + 2, { direction = E }) end

  -- frepb: a 1->1 row, a three-part NECK carrying nothing, and another 1->1 row.
  -- The target is the middle of the neck; its two vertical neighbours must be
  -- edgeless, or the belt that lands on it would be a second belt for one of
  -- them and that half would be refused rather than built.
  for r = 0, 4 do put(s, PART, 0, FREPB + r) end
  put(s, PART, 1, FREPB)
  put(s, PART, 1, FREPB + 4)
  local b = {}
  for _, r in ipairs { 0, 4 } do b[#b + 1] = frep_feed(s, FREPB + r) end
  storage.out.frepb = b
  log("[BBB-EDGE] frep-build end")
end

-- A fast replace hands the replaced ENTITY back as an item, and with no player
-- to hand it to the engine spills it. That item is new matter -- the engine
-- minted it, exactly as the insert probe mints its own -- and `count_all` is a
-- conserved quantity, so the machine items are logged and then removed. What the
-- belt was CARRYING is left exactly where it fell: that is a real spill and it
-- belongs in the count.
local MACHINE_ITEMS = { ["express-transport-belt"] = true, [PART] = true }

-- The engine hands the replaced entity back the way `spill_item_stack` does:
-- onto a belt if there is one under it and onto the floor if there is not. Both
-- have to be looked at, and the reverse gesture is exactly the case where the
-- item lands on a BELT -- the one that was just created on that tile.
local function frep_sweep(tag, y0, y1)
  local s, seen, took, where = surf(), {}, 0, "ground"
  local area = { { -8, y0 }, { 8, y1 } }
  for _, e in pairs(s.find_entities_filtered { area = area, type = "item-entity" }) do
    if e.valid and e.stack and e.stack.valid_for_read then
      local n, c = e.stack.name, e.stack.count
      seen[n] = (seen[n] or 0) + c
      if MACHINE_ITEMS[n] then
        took = took + c
        e.destroy()
      end
    end
  end
  for _, e in pairs(s.find_entities_filtered {
    area = area, type = { "transport-belt", "underground-belt", "splitter",
                          "lane-splitter", "loader-1x1", "loader" },
  }) do
    if e.valid then
      for i = 1, e.get_max_transport_line_index() do
        local line = e.get_transport_line(i)
        for name in pairs(MACHINE_ITEMS) do
          local n = line.get_item_count(name)
          if n > 0 then
            seen[name] = (seen[name] or 0) + n
            took = took + n
            where = "on a belt"
            line.remove_item { name = name, count = n }
          end
        end
      end
    end
  end
  local parts = {}
  for k, v in pairs(seen) do parts[#parts + 1] = k .. "x" .. v end
  table.sort(parts)
  log(string.format(
    "[BBB-EDGE] frep-spill tag=%s handed-back=[%s] machine-removed=%d where=%s",
    tag, table.concat(parts, ","), took, where))
end

-- FORWARD: a part onto the belt line, while everything is running.
--
-- THE CREATE IS GATED ON `can_fast_replace` AND THAT IS NOT BELT AND BRACES.
-- `create_entity{fast_replace = true}` does not ask that question: handed a
-- gesture the engine would refuse it falls back to CREATING, and a
-- simple-entity-with-force is created whatever it collides with -- so a guest
-- without the prototype line would end up with a part and a belt on one tile,
-- and the next compile would try to put an interface there and fail. Asking
-- first is what makes the red arm of this leg an assertion rather than a
-- compile error.
local function p_frep_forward()
  local s = surf()
  log("[BBB-EDGE] frep-fwd begin")
  local can = s.can_fast_replace {
    name = PART, position = P(0, FREP + 2), force = "player", direction = N,
  }
  log(string.format("[BBB-EDGE] frep-can what=part-over-belt value=%s",
    tostring(can)))
  local made = can and s.create_entity {
    name = PART, position = P(0, FREP + 2), force = "player",
    fast_replace = true, raise_built = true,
  } or nil
  log(string.format("[BBB-EDGE] frep-fwd created=%s belt-left=%s part-there=%s",
    tostring(made ~= nil),
    tostring(at(s, 0, FREP + 2, { type = "transport-belt" }) ~= nil),
    tostring(at(s, 0, FREP + 2, { name = PART }) ~= nil)))
  frep_sweep("fwd", FREP - 3, FREP + 5)
  log("[BBB-EDGE] frep-fwd end")
end

-- THE REFUSAL: a part that carries an edge interface cannot be belt-replaced,
-- because `bbb-linked-belt` is a belt-connectable standing on that same tile.
-- `can_fast_replace` is the engine's answer to a PLAYER and it is the assertion.
--
-- `create_entity` does not ask that question -- it mines the part and only then
-- discovers it cannot place the belt -- so the part is put back. A player cannot
-- reach that state and a phantom left standing here would be an artefact in
-- every audit after it rather than the thing under test.
local function p_frep_edge()
  local s = surf()
  log("[BBB-EDGE] frep-edge begin")
  log(string.format("[BBB-EDGE] frep-can what=belt-over-edge-part value=%s",
    tostring(s.can_fast_replace {
      name = BELT, position = P(0, FREPB), force = "player", direction = S })))
  local made = s.create_entity {
    name = BELT, position = P(0, FREPB), direction = S, force = "player",
    fast_replace = true, raise_built = true,
  }
  local still = at(s, 0, FREPB, { name = PART })
  log(string.format("[BBB-EDGE] frep-edge created=%s part-survived=%s",
    tostring(made ~= nil), tostring(still ~= nil)))
  if not still then
    s.create_entity { name = PART, position = P(0, FREPB), force = "player" }
  end
  frep_sweep("edge", FREPB - 3, FREPB + 8)
  log("[BBB-EDGE] frep-edge end")
end

-- REVERSE: a south-facing belt onto an INTERIOR part, which splits the column
-- into a two-part cluster above and a one-part cluster below, with the new belt
-- an OUTPUT of the first and an INPUT of the second.
--
-- Nothing tells the guest this happened except the belt's own build event, so
-- without guest/go/fastreplace.go the registry keeps four parts in one cluster,
-- the belt is INTERIOR and therefore never classified, the fingerprint never
-- moves, and nothing at all is rebuilt.
local function p_frep_reverse()
  local s = surf()
  log("[BBB-EDGE] frep-rev begin")
  local can = s.can_fast_replace {
    name = BELT, position = P(0, FREPB + 2), force = "player", direction = S,
  }
  log(string.format("[BBB-EDGE] frep-can what=belt-over-interior-part value=%s",
    tostring(can)))
  local made = can and s.create_entity {
    name = BELT, position = P(0, FREPB + 2), direction = S, force = "player",
    fast_replace = true, raise_built = true,
  } or nil
  log(string.format("[BBB-EDGE] frep-rev created=%s part-left=%s belt-there=%s",
    tostring(made ~= nil),
    tostring(at(s, 0, FREPB + 2, { name = PART }) ~= nil),
    tostring(at(s, 0, FREPB + 2, { type = "transport-belt" }) ~= nil)))
  frep_sweep("rev", FREPB - 3, FREPB + 8)
  log("[BBB-EDGE] frep-rev end")
end

local function report(tag)
  local names = {}
  for name in pairs(storage.out) do names[#names + 1] = name end
  table.sort(names)
  for _, name in ipairs(names) do
    local chests = storage.out[name]
    local parts = {}
    for i, c in ipairs(chests) do
      local n = 0
      if c and c.valid then
        for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do
          n = n + item.count
        end
      end
      parts[i] = tostring(n)
    end
    log(string.format("[BBB-EDGE] t=%s rig=%s out=[%s]", tag, name,
      table.concat(parts, " ")))
  end
end

--------------------------------------------------------------------------------
-- the schedule
--------------------------------------------------------------------------------

local CHN_T0 = 300
local CHN_END = CHN_T0 + CHN_ITERS * CHN_PERIOD

local ONE_OFFS = {
  [CHN_END + 20]  = function() audit_and_count("pre-sametick") end,
  [CHN_END + 24]  = p_same_tick,
  [CHN_END + 28]  = function() audit_and_count("post-sametick") end,
  [CHN_END + 32]  = p_pending_a,
  [CHN_END + 33]  = p_pending_b,
  [CHN_END + 38]  = function() audit_and_count("post-pending") end,
  [CHN_END + 42]  = p_pending_undo,
  [CHN_END + 46]  = function() audit_and_count("post-pending-undo") end,

  [CHN_END + 70]  = function() audit_and_count("pre-merge") end,
  [CHN_END + 72]  = p_merge,
  [CHN_END + 76]  = function() audit_and_count("post-merge"); probe_placement("post-merge") end,
  [CHN_END + 110] = p_unmerge,
  [CHN_END + 114] = function() audit_and_count("post-split") end,

  [CHN_END + 140] = function() audit_and_count("pre-rot") end,
  [CHN_END + 142] = p_rot_silent,
  [CHN_END + 144] = function() audit_and_count("post-rot-silent") end,
  [CHN_END + 148] = p_rot_restore,
  [CHN_END + 150] = function() audit_and_count("post-rot-restored") end,
  [CHN_END + 154] = p_rot_event,
  [CHN_END + 158] = function() audit_and_count("post-rot-event") end,
  [CHN_END + 162] = p_rot_event_back,
  [CHN_END + 166] = function() audit_and_count("post-rot-event-back") end,

  [CHN_END + 200] = function() audit_and_count("pre-forces") end,
  [CHN_END + 202] = p_forces_interleaved,
  [CHN_END + 206] = function() audit_and_count("post-forces-interleaved") end,
  [CHN_END + 240] = p_forces_merge,
  [CHN_END + 244] = function() audit_and_count("post-forces-merge") end,

  [CHN_END + 300] = function() det_paste(DET1, "1") end,
  [CHN_END + 303] = function() det_flushed("1") end,
  [CHN_END + 310] = function() det_clear(DET1) end,
  [CHN_END + 320] = function() det_paste(DET2, "2") end,
  [CHN_END + 323] = function() det_flushed("2") end,
  [CHN_END + 330] = function() det_clear(DET2) end,

  [CHN_END + 350] = p_hidden_part,
  [CHN_END + 354] = function() audit_and_count("post-hidden-part") end,

  -- The field report, and the three shapes of it. Nothing here changes the
  -- number of parts or of clusters: every one of them is an EDGE edit on a
  -- balancer that is running, which is exactly the case a recompile must not
  -- put on the floor.
  [CHN_END + 400]  = function() audit_and_count("pre-add-out") end,
  [CHN_END + 404]  = p_add_output,
  [CHN_END + 408]  = function() audit_and_count("post-add-out"); probe_placement("post-add-out") end,
  -- ... and then the balance property of what it became, over a 500-tick window
  -- that opens well after the new port's own belt and loader have filled. It is
  -- a RATE, not a total: the four original ports carry two thousand items from
  -- before the recompile and the fifth carries none, and a window that opened on
  -- the edit would read the fifth one's pipeline filling as throughput.
  [CHN_END + 560]  = function() report("aout-a") end,

  -- bmin's GROWING half. It has to happen here rather than beside its own
  -- shrinking half, because the network it grows into must be FULL before that
  -- one comes down and a P=4 butterfly takes a few hundred ticks to back up out
  -- of two express belts. Nothing may reach the ground: this is an edge edit
  -- that makes the machine bigger, which is the `aout` case.
  [CHN_END + 596]  = function() audit_and_count("pre-bmin") end,
  [CHN_END + 600]  = p_bmin_add,
  [CHN_END + 604]  = function() audit_and_count("post-bmin-add") end,

  [CHN_END + 1060] = function() report("aout-b") end,
  -- The notch rig has been saturated and moving for the whole run by now, which
  -- is the condition the field report was made under: the artifact was
  -- invisible until items were flowing.
  [CHN_END + 1062] = function() probe_placement("flowing") end,

  [CHN_END + 1080] = function() audit_and_count("pre-add-in") end,
  [CHN_END + 1084] = p_add_input,
  [CHN_END + 1088] = function() audit_and_count("post-add-in") end,

  [CHN_END + 1120] = function() audit_and_count("pre-shrink") end,
  [CHN_END + 1124] = p_remove_output,
  [CHN_END + 1128] = function() audit_and_count("post-shrink") end,

  [CHN_END + 1136] = p_probe_player_mine,
  -- The operator seam. It runs BEFORE the removal legs below, because
  -- `remote.call('...','audit')` really does audit -- it drains the deferred
  -- queue and recompiles -- so it must not land between a leg that mines
  -- something and the count that describes it.
  [CHN_END + 1138] = p_probe_operators,
  [CHN_END + 1140] = p_remove_cluster,
  [CHN_END + 1144] = function() audit_and_count("post-remove") end,

  -- ... and bmin's SHRINKING half, five hundred ticks after the belt was laid,
  -- which is the field report itself. It is scheduled after every tag whose
  -- assertion is that the ground is EMPTY -- ground is cumulative over the run,
  -- and this leg deliberately puts items on it.
  [CHN_END + 1160] = function() audit_and_count("pre-bmin-remove") end,
  [CHN_END + 1164] = p_bmin_remove,
  [CHN_END + 1168] = function() audit_and_count("post-bmin-remove") end,

  -- The field report's own gesture: the `aout` rig taken apart ONE PART PER
  -- TICK. Its balance windows closed 140 ticks ago, so it is free, and it is a
  -- saturated four-part rig that has been running for the whole suite.
  -- TEN STEPS, because the rig is ten parts: eight for a 4x4 under the one-belt
  -- rule and two more for the spare row `p_add_output` needed. Nine shrinks and
  -- a dissolve, one part per four ticks with an audit between each pair.
  [CHN_END + 1200] = function() audit_and_count("pre-hand") end,
  [CHN_END + 1204] = p_hand_mine(1),
  [CHN_END + 1208] = function() audit_and_count("hand-1") end,
  [CHN_END + 1212] = p_hand_mine(2),
  [CHN_END + 1216] = function() audit_and_count("hand-2") end,
  [CHN_END + 1220] = p_hand_mine(3),
  [CHN_END + 1224] = function() audit_and_count("hand-3") end,
  [CHN_END + 1228] = p_hand_mine(4),
  [CHN_END + 1232] = function() audit_and_count("hand-4") end,
  [CHN_END + 1236] = p_hand_mine(5),
  [CHN_END + 1240] = function() audit_and_count("hand-5") end,
  [CHN_END + 1244] = p_hand_mine(6),
  [CHN_END + 1248] = function() audit_and_count("hand-6") end,
  [CHN_END + 1252] = p_hand_mine(7),
  [CHN_END + 1256] = function() audit_and_count("hand-7") end,
  [CHN_END + 1260] = p_hand_mine(8),
  [CHN_END + 1264] = function() audit_and_count("hand-8") end,
  [CHN_END + 1268] = p_hand_mine(9),
  [CHN_END + 1272] = function() audit_and_count("hand-9") end,
  [CHN_END + 1276] = p_hand_mine(10),
  [CHN_END + 1280] = function() audit_and_count("hand-10") end,

  [CHN_END + 1288] = p_insert_probe,
  [CHN_END + 1294] = p_insert_probe_read,

  [CHN_END + 1298] = function()
    audit_and_count("final"); report("final"); probe_placement("final")
  end,

  -- THE SIXTY-FIFTH BELT, last because it is the only leg whose rig is thirty-two
  -- parts and because the tail of this suite is quiet -- every other rig has
  -- finished being edited, so the two delivery windows either side of the
  -- refusal measure the lim rig and nothing else.
  --
  -- The two windows are the same length (246 ticks) and are compared as a ratio.
  -- They have to be: sixty-three of this rig's output ports dead-end, so the
  -- share of the input that reaches the live one climbs all run as they fill,
  -- and a constant would be a number copied from a passing run.
  [CHN_END + 1304] = function() lim_report("before-open") end,
  [CHN_END + 1548] = function() audit_and_count("pre-lim") end,
  [CHN_END + 1550] = function() lim_report("before-close") end,
  [CHN_END + 1552] = p_lim_add,
  [CHN_END + 1556] = function()
    audit_and_count("post-lim"); lim_report("after-open")
  end,
  [CHN_END + 1802] = function() lim_report("after-close") end,
  [CHN_END + 1806] = function() audit_and_count("post-lim-window") end,
  [CHN_END + 1810] = p_lim_remove,
  [CHN_END + 1814] = function() audit_and_count("post-lim-back") end,

  -- THE BRIDGE THAT WOULD BE OVER THE LIMIT, last of all and for the same
  -- reason `lim` is next-to-last: nothing else in the save is being edited any
  -- more, so the delivery windows either side of the merge measure these two
  -- balancers and nothing else. All three windows are 246 ticks and are
  -- compared as ratios, because both halves back-fill all run (thirty-one of
  -- each half's thirty-two output ports dead-end) and a constant would be a
  -- number copied from a passing run.
  [CHN_END + 1848] = function() brdg_report("before-open") end,
  [CHN_END + 2092] = function() audit_and_count("pre-brdg") end,
  [CHN_END + 2094] = function() brdg_report("before-close") end,
  [CHN_END + 2096] = p_brdg_add,
  [CHN_END + 2100] = function()
    audit_and_count("post-brdg"); brdg_report("after-open"); probe_placement("brdg")
  end,
  -- Two more audits while the refusal STANDS. The state a refused merge leaves
  -- is the only one in this guest where `nets` holds a network under a key that
  -- is no longer a root, and it has to be stable: the same report, no second
  -- alert, and nothing torn down or rebuilt in between.
  [CHN_END + 2198] = function() audit_and_count("brdg-hold-1") end,
  [CHN_END + 2298] = function() audit_and_count("brdg-hold-2") end,
  [CHN_END + 2346] = function() brdg_report("after-close") end,
  [CHN_END + 2350] = function() audit_and_count("post-brdg-window") end,
  [CHN_END + 2354] = p_brdg_remove,
  [CHN_END + 2358] = function() audit_and_count("post-brdg-back") end,
  -- The recovery window opens a hundred ticks after the un-merge: the half whose
  -- root the merged cluster had been using is torn down and rebuilt from its own
  -- carry pool, and a rebuild puts every reinserted item back at the HEAD of the
  -- butterfly, so its output is briefly starved by construction.
  [CHN_END + 2458] = function() brdg_report("back-open") end,
  [CHN_END + 2704] = function() brdg_report("back-close") end,
  [CHN_END + 2708] = function() audit_and_count("post-brdg-final") end,

  -- FAST REPLACE, after everything else, and that is deliberate: these two rigs
  -- are BUILT HERE rather than in on_init so that every baseline above -- the
  -- fifteen clusters, the one hundred and ninety-eight parts, the
  -- placement-probe counts -- is a statement about the same world it has always
  -- been about. Only the tags below see the frep rigs at all.
  [CHN_END + 2728] = p_frep_build,
  -- Three hundred and twenty ticks to fill: source chest, loader, four belts, a
  -- P=2 hidden network and three more belts before the first item reaches a
  -- chest.
  [CHN_END + 3048] = function()
    audit_and_count("frep-built"); report("frep-before-open")
  end,
  [CHN_END + 3398] = function() report("frep-before-close") end,
  [CHN_END + 3402] = function() audit_and_count("pre-frep") end,
  [CHN_END + 3406] = p_frep_forward,
  [CHN_END + 3410] = function() audit_and_count("post-frep-fwd") end,
  [CHN_END + 3414] = p_frep_edge,
  [CHN_END + 3418] = function() audit_and_count("post-frep-edge") end,
  [CHN_END + 3422] = p_frep_reverse,
  [CHN_END + 3426] = function() audit_and_count("post-frep-rev") end,
  -- The after-window opens 162 ticks past the last edit and is the same length
  -- as the before-window, so the two are a ratio. A rebuild puts every
  -- reinserted item back at the HEAD of the butterfly, so an output is briefly
  -- starved by construction -- the same reason the brdg recovery window waits.
  [CHN_END + 3588] = function() report("frep-after-open") end,
  [CHN_END + 3938] = function() report("frep-after-close") end,
  [CHN_END + 3942] = function()
    audit_and_count("frep-final"); probe_placement("frep")
  end,

  [CHN_END + 3946] = function() log("[BBB-EDGE] done") end,
}

--------------------------------------------------------------------------------

script.on_init(function()
  local s = make_surface()
  game.create_force(OTHER_FORCE)
  storage.out = {}

  -- One rig: `rows` rows of TWO parts, of which the first `fed` are fed and
  -- drained. A row that is not fed carries nothing at all on either of its
  -- parts, which is the only place in a working balancer a player's belt can
  -- still land.
  local function rig(name, base, rows, force, fed)
    fed = fed or rows
    for r = 0, rows - 1 do
      put(s, PART, 0, base + r, { force = force })
      put(s, PART, 1, base + r, { force = force })
    end
    local chests = {}
    for r = 0, fed - 1 do chests[r + 1] = feed(s, base + r, force) end
    if name then storage.out[name] = chests end
    return chests
  end

  -- chn's outputs DEAD-END: belts, and nothing to take the items off them. The
  -- whole hidden network backs up and stays full, so every one of the two
  -- hundred teardowns below drains a network that is carrying items.
  for r = 0, 1 do put(s, PART, 0, CHN + r); put(s, PART, 1, CHN + r) end
  for r = 0, 1 do
    source(s, CHN + r)
    for x = -4, -1 do put(s, BELT, x, CHN + r, { direction = E }) end
    for x = 2, 4 do put(s, BELT, x, CHN + r, { direction = E }) end
  end
  rig("same", SAME, 2)
  -- Two clusters with a one-tile gap at MRG+2. Row MRG+2 gets no belts, so the
  -- bridging part brings no edge of its own -- the whole change is the merge.
  for r = 0, 1 do put(s, PART, 0, MRG + r); put(s, PART, 1, MRG + r) end
  for r = 3, 4 do put(s, PART, 0, MRG + r); put(s, PART, 1, MRG + r) end
  local mrg = {}
  for _, r in ipairs { 0, 1, 3, 4 } do mrg[#mrg + 1] = feed(s, MRG + r) end
  storage.out.mrg = mrg
  rig("rot", ROT, 2)
  rig("frcA", FRC, 2)
  rig("frcB", FRC + 2, 2, OTHER_FORCE)
  -- The three recompile-under-load rigs. Four in and four out over eight parts,
  -- saturated -- and `aout` and `ain` carry a fifth row that holds nothing, so
  -- that the belt each of them is given mid-run has somewhere legal to land.
  -- The part count never moves in any of the three: every edit is a belt.
  rig("aout", AOUT, 5, nil, 4)
  rig("ain", AIN, 5, nil, 4)
  rig("shrk", SHRK, 4)

  -- ntch: the field report's own shape, saturated and left running for the whole
  -- suite. A C of five parts around a HOLE at (1, b+1) -- the one tile in this
  -- save enclosed by parts on three sides and not one itself, so it is where a
  -- stray visible entity would be most visible. Two inputs on the west parts of
  -- rows b and b+1, two outputs on the east parts of rows b and b+2, and
  -- (0, b+2) carrying nothing, which is what keeps every tile to one belt.
  do
    local b = NTCH
    put(s, PART, 0, b)
    put(s, PART, 1, b)
    put(s, PART, 0, b + 1)
    put(s, PART, 0, b + 2)
    put(s, PART, 1, b + 2)
    for _, r in ipairs { 0, 1 } do feed_in(s, b + r) end
    local north = drain_out(s, b)
    local south = drain_out(s, b + 2)
    storage.out.ntch = { north, south }
  end

  -- bmin: a 2->2 over four parts, DEAD-ENDED like chn -- the outputs are belts
  -- with nothing to take items off them, so the hidden network fills and stays
  -- full -- plus an attached FIFTH PART carrying nothing. That fifth part is
  -- where the third output belt goes: under the one-belt rule the four working
  -- parts have no free face between them, and this rig's whole subject is a port
  -- count crossing a power of two in both directions. It is registered in no
  -- `storage.out` because it has no chests; what is measured about it is the
  -- ground.
  for r = 0, 1 do put(s, PART, 0, BMIN + r); put(s, PART, 1, BMIN + r) end
  put(s, PART, 0, BMIN + 2)
  for r = 0, 1 do
    source(s, BMIN + r)
    for x = -4, -1 do put(s, BELT, x, BMIN + r, { direction = E }) end
    for x = 2, 4 do put(s, BELT, x, BMIN + r, { direction = E }) end
  end

  -- lim: THE BIGGEST BALANCER THIS MOD BUILDS, one belt short of refusing. A
  -- 2x32 block with one input belt against each of its sixty-four parts, one
  -- part above it carrying the single output, and one SPARE PART below it
  -- carrying nothing -- sixty-six parts, sixty-four inputs, P = 64 =
  -- plan.MaxPorts exactly. Only three of the sixty-four are fed: see the header
  -- above p_lim_add for why.
  do
    put(s, PART, 0, LIM - 1)
    put(s, BELT, 0, LIM - 2, { direction = N })
    put(s, LOADER, 0, LIM - 3, { direction = N, type = "input" })
    storage.lim_chest = s.create_entity {
      name = "steel-chest", position = P(0, LIM - 4), force = "player",
    }
    for r = 0, LIM_ROWS - 1 do
      put(s, PART, 0, LIM + r)
      put(s, PART, 1, LIM + r)
      put(s, BELT, -1, LIM + r, { direction = E })
      put(s, BELT, 2, LIM + r, { direction = W })
    end
    put(s, PART, 0, LIM + LIM_ROWS)
    for r = 0, LIM_FED - 1 do
      source(s, LIM + r)
      for x = -4, -2 do put(s, BELT, x, LIM + r, { direction = E }) end
    end
  end

  -- brdg: TWO balancers with a one-tile gap, whose MERGE is over the limit. A
  -- 2x16 block each with one input belt per part (thirty-two inputs, P = 32) and
  -- an output part of its own -- A's above, B's below, so neither is anywhere
  -- near the gap. The GAP TILE gets ONE input belt beside it here and keeps it
  -- for the whole run: it is an edge of nothing until a part stands there, and
  -- then it is the one that takes the merged cluster to sixty-five.
  do
    local gap = BRDG + BRDG_HALF
    local foot = BRDG + 2 * BRDG_HALF
    put(s, PART, 0, BRDG - 1)
    put(s, BELT, 0, BRDG - 2, { direction = N })
    put(s, LOADER, 0, BRDG - 3, { direction = N, type = "input" })
    storage.brdg_a = s.create_entity {
      name = "steel-chest", position = P(0, BRDG - 4), force = "player",
    }
    for r = 0, 2 * BRDG_HALF do
      if r ~= BRDG_HALF then
        put(s, PART, 0, BRDG + r)
        put(s, PART, 1, BRDG + r)
        put(s, BELT, -1, BRDG + r, { direction = E })
        put(s, BELT, 2, BRDG + r, { direction = W })
      end
    end
    put(s, BELT, -1, gap, { direction = E })
    put(s, PART, 0, foot + 1)
    put(s, BELT, 0, foot + 2, { direction = S })
    put(s, LOADER, 0, foot + 3, { direction = S, type = "input" })
    storage.brdg_b = s.create_entity {
      name = "steel-chest", position = P(0, foot + 4), force = "player",
    }
    for r = 0, BRDG_FED - 1 do
      for _, y in ipairs { BRDG + r, gap + 1 + r } do
        source(s, y)
        for x = -4, -2 do put(s, BELT, x, y, { direction = E }) end
      end
    end
  end

  -- THE FAST-REPLACE RIGS' SOURCE CHESTS, AND NOTHING ELSE OF THEM. The rigs
  -- themselves are built at CHN_END + 2680 so that no baseline above moves, but
  -- their stock has to be inside the conserved total from t=0: `count_all` is
  -- the instrument this whole suite rests on, and thirty thousand items
  -- appearing in it halfway through would read as the mod minting matter.
  for _, y in ipairs { FREP, FREP + 1, FREP + 2, FREPB, FREPB + 4 } do
    local c = s.create_entity {
      name = "steel-chest", position = P(-6, y), force = "player",
    }
    c.get_inventory(defines.inventory.chest).insert {
      name = "iron-plate", count = STOCK,
    }
  end

  -- `--create` never reaches a tick, so the flush every build armed would land
  -- on the first tick of the benchmark. The marker drains it here.
  log("[BBB-EDGE] mark tag=init")
  s.create_entity { name = AUDIT, position = P(24, 4), force = "player", raise_built = true }
  -- Every network in the save now exists, so this is the widest sample the
  -- placement probe ever gets.
  probe_placement("init")
  -- The real number, not the requested one: a steel chest holds 48 stacks, so
  -- an insert of STOCK stops at 4,800 whatever STOCK says.
  log(string.format("[BBB-EDGE] plan chn_end=%d end_tick=%d stock=%d",
    CHN_END, CHN_END + 3950, count_all()))
end)

script.on_event(defines.events.on_tick, function(ev)
  local t = ev.tick
  if t >= CHN_T0 and t < CHN_END then
    local d = t - CHN_T0
    chn_step(d % CHN_PERIOD, math.floor(d / CHN_PERIOD) + 1)
    return
  end
  local f = ONE_OFFS[t]
  if f then f() end
end)
