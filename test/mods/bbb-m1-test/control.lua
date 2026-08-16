-- bbb-m1-test: drives the M1 cluster registry through every merge and split
-- shape, and marks each phase in the log so the assertions can be positional.
--
-- Deliberately plain Lua: this is test infrastructure, not the product. It
-- asserts NOTHING itself -- the guest's own `[BBB] state` lines are the
-- assertion surface, and test/assert-log.py checks them. A test mod that
-- computed the expected answer in Lua would be a second implementation of the
-- thing under test.
--
-- Phase 1 runs in on_init, which Factorio runs while `--create` builds the map,
-- so the save already contains the clusters. Phases 2..5 run on later ticks
-- during `--benchmark`, which means the registry has to survive a save and a
-- reload to get there -- that is not incidental, it is what proves
-- `--persist=packed` carries the guest heap.

local PART = "bbb-balancer-part"

--------------------------------------------------------------------------------
-- surfaces
--------------------------------------------------------------------------------

-- Two surfaces with parts at IDENTICAL coordinates. A registry that forgot to
-- key by surface merges them, and every phase after this one is wrong -- which
-- is the space-platform bug, found on a flat map.
local function make_surface(name)
  local mgs = {
    width = 256,
    height = 256,
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
  local surface = game.create_surface(name, mgs)
  surface.always_day = true
  surface.request_to_generate_chunks({ x = 32, y = 8 }, 4)
  surface.force_generate_chunk_requests()

  local area = { { -8, -8 }, { 72, 24 } }
  for _, e in pairs(surface.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  surface.destroy_decoratives { area = area }
  local tiles = {}
  for x = -8, 72 do
    for y = -8, 24 do
      tiles[#tiles + 1] = { name = "grass-1", position = { x, y } }
    end
  end
  surface.set_tiles(tiles, true, false, false, false)
  return surface
end

--------------------------------------------------------------------------------
-- placement / removal
--------------------------------------------------------------------------------

local function place(surface, tx, ty)
  local e = surface.create_entity {
    name = PART,
    position = { x = tx + 0.5, y = ty + 0.5 },
    force = "player",
    raise_built = true,
  }
  if not e then
    error(string.format("BBB-TEST: could not place %s at %s (%d,%d)", PART, surface.name, tx, ty))
  end
  return e
end

local function remove(surface, tx, ty)
  local found = surface.find_entities_filtered {
    name = PART,
    area = { { tx, ty }, { tx + 1, ty + 1 } },
  }
  if #found ~= 1 then
    error(string.format("BBB-TEST: expected 1 part at %s (%d,%d), found %d",
      surface.name, tx, ty, #found))
  end
  found[1].destroy { raise_destroy = true }
end

local SURFACES = { a = "bbb-m1-a", b = "bbb-m1-b", skin = "bbb-m1-skin" }

local function surf(which)
  return game.surfaces[SURFACES[which]]
end

local function phase(n, what)
  log(string.format("[BBB-TEST] phase=%d %s", n, what))
end

--------------------------------------------------------------------------------
-- the patterns
--------------------------------------------------------------------------------
--
--            x=0        x=10..13      x=20..22        x=30..34
--   y=0      A single   A line        A L (corner)    A bridged pair
--   y=1                               |
--   y=2                               |
--
-- Surface B carries a single at (0,0) and a pair at (10,0)-(11,0): the same
-- tiles surface A uses, so nothing may join across them.

local function phase1()
  phase(1, "build: single, line, L, bridge, cross-surface")
  local a = make_surface("bbb-m1-a")
  local b = make_surface("bbb-m1-b")

  -- 1. a lone part -- a cluster of one
  place(a, 0, 0)

  -- 2. a line, built left to right: three successive merges
  place(a, 10, 0); place(a, 11, 0); place(a, 12, 0); place(a, 13, 0)

  -- 3. an L: the corner merges two arms that are not in line
  place(a, 20, 0); place(a, 21, 0); place(a, 22, 0)
  place(a, 22, 1); place(a, 22, 2)

  -- 4. two separate clusters, THEN the tile that bridges them: one placement
  --    that merges two existing clusters rather than growing one
  place(a, 30, 0); place(a, 31, 0)
  place(a, 33, 0); place(a, 34, 0)
  place(a, 32, 0)

  -- 5. surface B, at coordinates surface A is already using
  place(b, 0, 0)
  place(b, 10, 0); place(b, 11, 0)
end

-- Remove the bridge: one cluster of 5 becomes two of 2.
local function phase2()
  phase(2, "remove bridge a(32,0): expect split 2+2")
  remove(surf("a"), 32, 0)
end

-- Remove the second tile of the line: 4 becomes 1 and 2.
local function phase3()
  phase(3, "remove line middle a(11,0): expect split 1+2")
  remove(surf("a"), 11, 0)
end

-- Remove the lone part: its cluster disappears entirely.
local function phase4()
  phase(4, "remove lone a(0,0): expect dissolve")
  remove(surf("a"), 0, 0)
end

-- Remove the L's corner: the two arms come apart.
local function phase5()
  phase(5, "remove L corner a(22,0): expect split 2+2")
  remove(surf("a"), 22, 0)
end

-- A DIFFERENT removal path. Everything above goes through
-- `destroy{raise_destroy=true}`, so nothing has yet proved the guest is
-- listening to on_entity_died -- and a part that dies to a biter or a nuke and
-- stays in the registry is a cluster the compiler will build a network for
-- twice.
local function phase6()
  phase(6, "kill b(11,0) via die(): expect shrink, on_entity_died path")
  local s = surf("b")
  local found = s.find_entities_filtered { name = PART, area = { { 11, 0 }, { 12, 1 } } }
  if #found ~= 1 then error("BBB-TEST: expected 1 part at b(11,0)") end
  found[1].die()
end

--------------------------------------------------------------------------------
-- M5: the adaptive sprite
--------------------------------------------------------------------------------
--
-- The five named shapes, on their own surface so nothing above is disturbed.
-- What is asserted is the guest's `[BBB] skin ... vars=` line -- the variation
-- it put on every part of a cluster, in (y, x) order -- against the numbers
-- guest/go/skin/skin_test.go proves in pure Go for the same five shapes. The Go
-- test says the mapping is right; this says the mapping reached the entities in
-- a real Factorio.
--
-- ALL OF PHASE 7 IS ONE TICK, deliberately: the guest defers its work to the
-- next tick, so thirty placements produce five skin lines, one per finished
-- shape, rather than a line per intermediate shape.
--
--    x=0..3   line       x=10..12  L        x=20..22  plus
--    x=30..31 2x2 block  x=40..43  donut (4x4 ring around a 2x2 hole)

local function phase7()
  phase(7, "build the five named shapes: line, L, plus, 2x2, donut")
  local s = make_surface("bbb-m1-skin")
  place(s, 0, 0); place(s, 1, 0); place(s, 2, 0); place(s, 3, 0)

  place(s, 10, 0); place(s, 10, 1); place(s, 10, 2); place(s, 11, 2); place(s, 12, 2)

  place(s, 21, 0); place(s, 20, 1); place(s, 21, 1); place(s, 22, 1); place(s, 21, 2)

  place(s, 30, 0); place(s, 31, 0); place(s, 30, 1); place(s, 31, 1)

  for y = 0, 3 do
    for x = 40, 43 do
      if not (x >= 41 and x <= 42 and y >= 1 and y <= 2) then place(s, x, y) end
    end
  end
end

-- Grow the line by one tile at its east end. The new part and the part that USED
-- to be the end change picture; the other three do not, and the guest must say
-- so -- `set=2` out of `parts=5` is the claim that a part added to a balancer
-- costs host calls for the pictures that moved and not for the ones that did
-- not. That is what keeps this affordable on a 200-part balancer.
local function phase8()
  phase(8, "grow the line east: expect set=2 of parts=5")
  place(surf("skin"), 4, 0)
end

-- Take the plus apart at its centre. Four lone parts, each of which must go back
-- to the lone-part picture (variation 1): the update path in the other
-- direction, and through RemovePart rather than AddPart.
local function phase9()
  phase(9, "remove the plus centre: expect four lone parts at variation 1")
  remove(surf("skin"), 21, 1)
end

--------------------------------------------------------------------------------
-- schedule
--------------------------------------------------------------------------------

-- Phases 2..5 land on separate ticks so the log is unambiguously ordered, and
-- so that each one is a fresh event dispatch rather than a continuation of the
-- one before.
local SCHEDULE = {
  [10] = phase2, [20] = phase3, [30] = phase4, [40] = phase5, [50] = phase6,
  [60] = phase7, [70] = phase8, [80] = phase9,
}

script.on_init(function()
  phase1()
  log("[BBB-TEST] init complete")
end)

script.on_event(defines.events.on_tick, function(e)
  local f = SCHEDULE[e.tick]
  if f then
    f()
    if e.tick == 80 then log("[BBB-TEST] phases complete") end
  end
end)
