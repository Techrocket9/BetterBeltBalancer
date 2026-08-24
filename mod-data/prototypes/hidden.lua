-- The hidden network's prototypes: everything the compiler places, and nothing
-- a player can ever see, hold, blueprint or clone.
--
-- Four prototypes, all clones of base ones with the speed raised and the
-- player-facing surface removed:
--
--   bbb-linked-belt   the edge interface AND the network's row-crossing jumper
--   bbb-belt          the wire between stages
--   bbb-splitter      the balancing element (2 tiles tall)
--   bbb-lane-splitter the 1x1 lane-fidelity stage at the inputs
--
-- ---------------------------------------------------------------------------
-- THE COLLISION MASK ON bbb-linked-belt WAS THE WHOLE ARCHITECTURE, AND ON
-- FACTORIO 2.1 THE DOOR IS SHUT.
--
-- Spike S1 probed 14 masks on 2.0.77. The engine VALIDATES any belt-connectable
-- whose `layers` differ from the type default -- and the validation demands the
-- mask collide with transport-belt's and with itself, so every deviation fails
-- at load. But when `layers` is EXACTLY the type default the validation was
-- skipped entirely, while `not_colliding_with_itself` was still honoured at
-- runtime. That was the one and only door to putting two (or four)
-- belt-connectables on one tile, which is what let a 1x1 balancer part carry an
-- input interface on one side and an output interface on another.
--
-- It was a loophole and it is closed. 2.1 fixed the equals-compare that skipped
-- the validation, so the check now runs on every belt-connectable and demands
-- the mask collide with itself; probed exhaustively on 2.1.14, no mask design
-- passes and no runtime bypass exists (`create_entity` nils, `teleport` returns
-- false). boskid's answer to the interface request (forums t=135830) explains
-- the invariant it protects: belt-to-belt connections are NOT SAVED, they are
-- re-derived at load, and one belt-connectable per tile is what makes that
-- re-derivation unambiguous.
--
-- SO THE FLAG IS EMITTED ON 2.0.x AND NEVER ON 2.1.x, and with it the marker
-- prototype `bbb-can-stack` that tells the guest which world it is in. The rule
-- the guest enforces when the marker is absent is at most ONE BELT PER BALANCER
-- PART, because one part tile may carry at most one interface linked belt.
-- agents/single-edge.md is the whole port; guest/go/sedge.go is the runtime
-- half of this file.
--
-- THE VERSION GUARD IS WHAT KEEPS A MISPACKAGED ZIP FROM BRICKING A 2.1 LOAD,
-- and it fails SAFE: anything this file cannot read as 2.0.x is treated as 2.1,
-- because emitting the flag on 2.1 refuses the mod at load while not emitting
-- it on 2.0 merely costs the multi-edge geometry the guest would then refuse to
-- build anyway.
--
-- Do not "tidy" the layer list. On 2.0 adding or removing one entry moves this
-- prototype from the unvalidated path to the validated one, and the mod stops
-- loading.
-- ---------------------------------------------------------------------------
--
-- SPEED = 0.25 and that number is a ceiling, not a preference. A transport line
-- accepts at most one item per tick per lane, so 0.25 tiles/tick -- one item
-- width per tick -- is the fastest a belt can still run COMPRESSED: 60
-- items/s/lane, 120/s/belt, 2.67x express and 1.33x turbo. Going higher buys
-- nothing (the line cannot be fed any faster) and starts to open gaps.
--
-- 0.25 is enough by construction: the network spreads N input belts over
-- P >= N lines, so no hidden line ever carries more than one visible belt's
-- rate. A modded belt faster than 0.25 would bottleneck here; nothing in base
-- or Space Age is.

-- ---------------------------------------------------------------------------
-- AND THE VISUALS ARE BLANKED, WHICH IS NOT TIDYING. A clone of a base belt
-- keeps every picture the base prototype had, and `bbb-linked-belt` is the one
-- hidden prototype that stands on a surface a PLAYER LOOKS AT: the compiler
-- puts one on a balancer part's own tile for every edge of the cluster.
--
-- The part sprite is 64 px at scale 0.5 -- exactly one tile, opaque in all 47
-- cells (checked pixel by pixel) -- and it draws at render layer `object`, so
-- it covers everything the interface draws ON ITS OWN TILE. What it cannot
-- cover is what the interface draws OUTSIDE that tile, and base's linked belt
-- draws a lot of it:
--
--   structure           192x192 at scale 0.5 = THREE TILES BY THREE, at
--                       `structure_render_layer = "object"` -- the same layer
--                       the part is on, spilling onto all eight neighbours
--   belt_animation_set  the running belt, plus the starting/ending patches,
--                       which are drawn past the tile edge by design
--
-- On a solid rectangle of parts the neighbours' own sprites hide most of that.
-- On a shape with a NOTCH -- a 2x2 with one corner missing, which is what the
-- field report was about -- the empty tile is covered by nothing, and every
-- interface around it paints into it. Blanking is the fix, and it is free:
-- nothing about the hidden network is meant to be looked at.
--
-- Every replacement is "a valid sprite that happens to be one transparent
-- pixel of `__core__/graphics/empty.png`" rather than `nil`. Both are legal --
-- `belt_animation_set` and every `structure` field below are optional in
-- 2.0.77's prototype API -- but headless Factorio never opens a sprite file and
-- `test/check-sprites.py` only checks that the ones we name exist, so the
-- GRAPHICAL client is the first thing that would notice a shape the engine did
-- not like. A drawable-but-empty sprite is the shape this repo has already
-- proved (the audit marker below, and `util.empty_sprite` is core's own idiom);
-- an absent one asks the renderer to take a branch nothing here has exercised.
--
-- `direction_count = 20` on the blank animation set is the one number that is
-- not arbitrary: a belt animation set is indexed by direction and by patch, and
-- the defaults run up to `ending_east_index = 20`. Base's own belt sets are
-- `direction_count = 20` for exactly that reason. `empty.png` is 64x64, so 20
-- rows of one pixel fit inside it with room to spare.
--
-- WHAT THIS DOES NOT FIX: the ITEMS. 2.0.77 has no prototype field anywhere
-- that suppresses the drawing of items on a belt-connectable, and no
-- linked-belt equivalent of `LoaderPrototype::belt_length` to shorten the
-- stretch they are drawn over. See CLAUDE.md, "The tan streak".
-- ---------------------------------------------------------------------------

local util = require("util")

local SPEED = 0.25

local EMPTY_PNG = "__core__/graphics/empty.png"

-- One transparent pixel, as an Animation. Valid wherever an Animation,
-- Animation4Way, Sprite or Sprite4Way is wanted; `util.empty_sprite()` is the
-- same thing for the sprite-shaped fields.
local function blank_animation()
  return {
    filename = EMPTY_PNG,
    priority = "extra-high",
    width = 1,
    height = 1,
    frame_count = 1,
    line_length = 1,
  }
end

-- A TransportBeltAnimationSet (and, with the same fields, the WithCorners form:
-- its extra indices are 5..12 and are covered by the same 20 directions) whose
-- every direction and every patch is that one pixel. `animation_set` is the one
-- non-optional member of the set, which is why the set is replaced whole rather
-- than emptied field by field.
local function blank_belt_animation_set()
  return {
    animation_set = {
      filename = EMPTY_PNG,
      priority = "extra-high",
      width = 1,
      height = 1,
      frame_count = 1,
      line_length = 1,
      direction_count = 20,
    },
  }
end

-- The type default for a belt-connectable, verified against 2.0.77's
-- prototypes/collision-layers.lua. See the block comment above.
local BELT_LAYERS = {
  floor = true,
  meltable = true,
  object = true,
  transport_belt = true,
  water_tile = true,
}

-- WHICH ENGINE THIS IS. `mods` is the data stage's own dictionary of every
-- installed mod's version, base included, and it is the only thing here that
-- can tell 2.0 from 2.1 -- the two data stages are otherwise identical.
--
-- Anything unreadable is treated as 2.1, which is the safe direction: see the
-- header. The match is on MAJOR.MINOR alone, so every 2.0.x point release is
-- 2.0 and everything from 2.1.0 on is not.
local function base_is_2_0()
  local v = mods and mods["base"]
  if type(v) ~= "string" then return false end
  local major, minor = v:match("^(%d+)%.(%d+)")
  return tonumber(major) == 2 and tonumber(minor) == 0
end

local CAN_STACK = base_is_2_0()

-- Flags that keep a hidden entity out of every path that could copy it.
-- The network is ALWAYS recompiled from visible state; an entity that survived
-- into a blueprint or a clone would be a second, untracked network.
local HIDDEN_FLAGS = {
  "placeable-neutral",
  "player-creation",
  "not-blueprintable",
  "not-deconstructable",
  "not-on-map",
  "no-copy-paste",
  "not-selectable-in-game",
  "not-upgradable",
  "not-in-kill-statistics",
  "not-in-made-in",
  "hide-alt-info",
}

-- strip turns a cloned prototype into a hidden one: no item, no upgrade path,
-- no fast-replace group (it would offer our belt as a replacement for a real
-- one), no selection box, and none of the per-tick cost of being visible.
local function strip(p)
  p.minable = nil
  p.placeable_by = nil
  p.hidden = true
  p.hidden_in_factoriopedia = true
  p.next_upgrade = nil
  p.fast_replaceable_group = nil
  p.flags = util.table.deepcopy(HIDDEN_FLAGS)
  p.selection_box = nil
  p.selectable_in_game = false
  p.max_health = 1
  p.corpse = nil
  p.dying_explosion = nil
  p.working_sound = nil
  p.open_sound = nil
  p.close_sound = nil
  p.impact_category = nil
  return p
end

-- blank removes every picture a cloned belt-connectable draws unconditionally.
-- The per-type structures are handled at each prototype below; this is the one
-- field all four share.
local function blank(p)
  p.belt_animation_set = blank_belt_animation_set()
  return p
end

local function clone(t, base, name)
  local p = util.table.deepcopy(data.raw[t][base])
  p.name = name
  p.speed = SPEED
  return blank(strip(p))
end

-- The edge interface and the row-crossing jumper.
local linked = clone("linked-belt", "linked-belt", "bbb-linked-belt")
linked.collision_mask = {
  layers = util.table.deepcopy(BELT_LAYERS),
}
if CAN_STACK then
  -- The door, on the engine that still has one. Without it a second linked belt
  -- cannot share the tile, and without that a 1x1 part cannot be both an input
  -- and an output -- which is the rule guest/go/sedge.go enforces everywhere
  -- else.
  linked.collision_mask.not_colliding_with_itself = true
end
-- A linked belt remembers its partner in a blueprint and re-links on paste, and
-- can be carried through a clone the same way. Both would resurrect a network
-- the compiler does not know about, so both are refused at the prototype.
linked.allow_blueprint_connection = false
linked.allow_clone_connection = false
-- THE THREE-BY-THREE SPRITE, gone. Every field of LinkedBeltStructure is
-- optional in 2.0.77 and every one of them is replaced rather than dropped.
-- This is the one that was actually being seen: it is the only prototype here
-- that stands on the visible surface.
linked.structure = {
  direction_in = util.empty_sprite(),
  direction_out = util.empty_sprite(),
  direction_in_side_loading = util.empty_sprite(),
  direction_out_side_loading = util.empty_sprite(),
  back_patch = util.empty_sprite(),
  front_patch = util.empty_sprite(),
}

local belt = clone("transport-belt", "express-transport-belt", "bbb-belt")
belt.related_underground_belt = nil

local splitter = clone("splitter", "express-splitter", "bbb-splitter")
splitter.related_transport_belt = "bbb-belt"
-- Belt-and-braces: these three never leave the hidden surface, and a surface
-- with no generated chunks is not a place anything is looked at. Blanked anyway
-- so that the rule is "nothing the compiler places draws anything" rather than
-- "nothing the compiler places draws anything WHERE IT MATTERS", which is the
-- kind of qualification the tan streak got in through.
splitter.structure = blank_animation()
splitter.structure_patch = blank_animation()
splitter.frozen_patch = util.empty_sprite()

local lane = clone("lane-splitter", "lane-splitter", "bbb-lane-splitter")
-- LaneSplitterPrototype::structure is the one MANDATORY picture in this file
-- (`optional = false` in the prototype API), so it is replaced, never dropped.
lane.structure = blank_animation()
lane.structure_patch = blank_animation()

-- The audit marker. Not part of the network: a one-shot request.
--
-- Placing one (only a script can -- there is no item, no recipe and no
-- technology that yields it) asks the guest to compare every cluster's stored
-- edge fingerprint against a fresh classification of the world, log the result
-- and repair whatever drifted. It destroys itself on the way out.
--
-- It exists because the alternative is a Lua reimplementation of the compiler
-- inside the test harness, which would assert that two implementations agree
-- rather than that one is right. It ships rather than living in the test mod so
-- that the same question can be asked of a real save that is behaving oddly.
--
-- Collision mask is EMPTY on purpose: the marker has to be placeable on a tile
-- that already holds a belt or a balancer part.
local audit = {
  type = "simple-entity",
  name = "bbb-audit",
  hidden = true,
  hidden_in_factoriopedia = true,
  flags = util.table.deepcopy(HIDDEN_FLAGS),
  icon = "__better-belt-balancer__/graphics/icons/balancer-part.png",
  icon_size = 64,
  max_health = 1,
  collision_mask = { layers = {} },
  collision_box = { { 0, 0 }, { 0, 0 } },
  selectable_in_game = false,
  -- Never rendered (selectable_in_game = false, zero-size boxes), but the
  -- GRAPHICAL client still validates every sprite path at load -- headless
  -- does not, which is how a stale filename here survived every suite and
  -- benchmark and then refused to load in the real game. An invisible marker
  -- gets the engine's empty sprite and no file dependency of ours at all.
  picture = util.empty_sprite(),
}

-- The insert probe. The same shape as the audit marker and for the same reason:
-- a one-shot request a script places, which the guest answers in the log and
-- then destroys.
--
-- What it asks is "does the boundary hand this engine the item COUNT I gave
-- it?" -- placed on a container, the guest offers that container a known number
-- of a known item through the very function the miner's pocket uses, and reports
-- what came back. It exists because that arithmetic was unverifiable headlessly
-- (a --create has no player) and a player found a defect in it that seven
-- suites could not; `insert` is a LuaControl member, and a chest is a LuaControl,
-- so a chest can be asked the question a player's pockets could not be.
--
-- UNLIKE THE AUDIT MARKER IT IS DEFERRED: the answer arrives on the NEXT tick,
-- because the pocket runs inside the deferred flush and a probe that answered
-- inside its own build event would be testing the call from a place the pocket
-- never makes it. A `--create` never reaches a tick, so a marker placed in
-- on_init reports on the first tick of the run rather than into the save.
--
-- Empty collision mask, like the audit marker: it has to be placeable on a tile
-- that already holds the container it is about to fill.
--
-- THE COLLISION BOX IS A TILE, AND UNLIKE THE AUDIT MARKER'S THAT MATTERS.
-- Factorio snaps a placed entity to the grid its bounding box implies, so the
-- audit marker's zero-size box snaps to a tile CORNER: one created at
-- {x + 0.5, y + 0.5} reports its position as {x + 1, y + 1}. The audit does not
-- care where it landed, and this does -- it has to find the container under
-- itself. A tile-sized box snaps to the tile CENTRE, exactly as the balancer
-- part does, so the probe's position is the tile it was asked for.
local probe = {
  type = "simple-entity",
  name = "bbb-insert-probe",
  hidden = true,
  hidden_in_factoriopedia = true,
  flags = util.table.deepcopy(HIDDEN_FLAGS),
  icon = "__better-belt-balancer__/graphics/icons/balancer-part.png",
  icon_size = 64,
  max_health = 1,
  collision_mask = { layers = {} },
  collision_box = { { -0.35, -0.35 }, { 0.35, 0.35 } },
  selectable_in_game = false,
  picture = util.empty_sprite(),
}

data:extend { linked, belt, splitter, lane, audit, probe }

-- THE CAPABILITY MARKER, and it is defined here rather than unconditionally
-- because its whole meaning is "the linked belt above carries
-- `not_colliding_with_itself`, so this engine can stack two of them on one
-- tile". The guest cannot read a prototype's collision mask and must not carry
-- a version number of its own; a point lookup of this name against
-- `prototypes.entity` answers the question in two host calls and no allocation,
-- and the guest's belief cannot drift from the prototype's actual capability
-- because the two are one `if`.
--
-- It is the `bbb-legacy-stub` idiom exactly (prototypes/legacy.lua), for the
-- same reason and with the same shape: a prototype that exists to be looked up
-- and that nothing ever places. See guest/go/sedge.go, `multiEdgeAllowed`.
if CAN_STACK then
  data:extend {
    {
      type = "simple-entity",
      name = "bbb-can-stack",
      hidden = true,
      hidden_in_factoriopedia = true,
      flags = util.table.deepcopy(HIDDEN_FLAGS),
      max_health = 1,
      collision_mask = { layers = {} },
      collision_box = { { 0, 0 }, { 0, 0 } },
      selectable_in_game = false,
      -- The graphical client validates every sprite path at load and headless
      -- does not, which is how a stale filename here once survived every suite
      -- and then refused to load in the real game. The engine's own empty
      -- sprite has no file dependency of ours at all.
      picture = util.empty_sprite(),
    },
  }
end
