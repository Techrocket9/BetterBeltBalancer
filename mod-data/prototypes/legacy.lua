-- THE LEGACY STUB: what keeps a Belt Balancer 2/3 balancer alive long enough
-- for the guest to adopt it.
--
-- WHY A STUB AND NOT A migrations/*.json RENAME. Factorio's prototype-migration
-- files are applied ONCE PER SAVE PER FILE and the fact that they ran is
-- remembered by file name
-- (https://lua-api.factorio.com/latest/auxiliary/migrations.html). So a rename
-- file shipped by this mod would be recorded as applied on the FIRST load after
-- this mod is installed -- which, for the player this feature is for, is a load
-- on which the incumbent is still present and its balancers must not be touched
-- -- and could never fire again on the later load where the incumbent is gone.
-- It also has no way to express "do nothing while that other mod is installed",
-- and what a rename does when BOTH the old and the new prototype exist is not
-- documented. The whole feature is a decision taken at RUNTIME, from
-- script.active_mods, so it belongs in the guest; this file exists only to stop
-- the engine deleting the evidence before the guest can look at it.
--
-- WHAT THE ENGINE DOES WITHOUT THIS FILE. When a mod is removed, every entity
-- whose prototype went with it is deleted at load, silently, before any script
-- runs. A player who swaps Belt Balancer 2 for this mod would find every
-- balancer part gone and every belt around them still standing. Defining a
-- prototype of the SAME NAME and a COMPATIBLE TYPE is what makes the entities
-- survive that load -- it is the same mechanism a rename file uses, applied by
-- existence rather than by a one-shot file.
--
-- WHEN IT DEFINES ANYTHING. Only when nobody else has. `data-final-fixes` runs
-- after every mod's data and data-updates stages, so an incumbent that is still
-- installed has already defined `balancer-part` and this file does nothing at
-- all -- which is the "leave it alone while it is installed" half of the
-- feature, enforced by the engine's own load order rather than by a list of mod
-- names. The runtime half is `bbb-legacy-stub` below.
--
-- THE TYPE TEST IS ON `simple-entity-with-force` AND THAT IS THE RIGHT TABLE.
-- Belt Balancer, Belt Balancer Performance, Belt Balancer 2 and Belt Balancer 3
-- all define `balancer-part` as exactly that type, and the stub has to be that
-- type anyway: an entity survives its prototype's disappearance only when the
-- name comes back under a compatible type. If some other mod were to define an
-- entity named `balancer-part` under a DIFFERENT entity type, Factorio refuses
-- to load at all -- entity names are unique across entity types -- so that case
-- is loud and immediate rather than silently wrong.

local ENTITY = "balancer-part"
local MARKER = "bbb-legacy-stub"

local existing_entity = data.raw["simple-entity-with-force"][ENTITY]
local existing_item = data.raw.item[ENTITY]

if not existing_entity then
  data:extend {
    {
      type = "simple-entity-with-force",
      name = ENTITY,

      -- WHAT IS COPIED FROM THE INCUMBENT AND WHY. The type and the collision
      -- box are what make a standing entity survive the load; the collision
      -- mask is what keeps it refusing to share a tile with a belt for the one
      -- load it is still standing; the mining result and `placeable_by` are
      -- what make a player who mines one, or a construction robot that revives
      -- an old blueprint's ghost of one, end up holding this mod's item
      -- instead of nothing.
      collision_box = { { -0.35, -0.35 }, { 0.35, 0.35 } },
      selection_box = { { -0.5, -0.5 }, { 0.5, 0.5 } },
      collision_mask = {
        layers = {
          floor = true,
          meltable = true,
          object = true,
          transport_belt = true,
          water_tile = true,
        },
      },
      max_health = 170,
      corpse = "splitter-remnants",
      resistances = { { type = "fire", percent = 60 } },
      minable = { mining_time = 0.1, result = "bbb-balancer-part" },
      placeable_by = { item = "bbb-balancer-part", count = 1 },

      -- A PLAYER CANNOT BUILD ONE, AND NO FLAG IS WHAT STOPS THEM. Nothing
      -- places this prototype: the stub item below has `place_result =
      -- "bbb-balancer-part"`, there is no recipe and no technology, and the
      -- only item whose `placeable_by` names it is this mod's own part item,
      -- which places this mod's own part. That is structural and stronger than
      -- any flag.
      --
      -- `not-blueprintable` is DELIBERATELY ABSENT, and the reason is the one
      -- case this whole file exists for. A migrating player's blueprint book is
      -- full of `balancer-part`, and those books keep working: a ghost of this
      -- prototype asks for a `bbb-balancer-part` item through `placeable_by`, a
      -- robot builds the stub, and the guest swaps it for a real part inside
      -- the build event (guest/go/legacy.go). Refusing the blueprint would
      -- break the one path that makes an old book useful, in exchange for
      -- stopping a capture of a prototype that exists for at most one load.
      --
      -- `not-upgradable` is present because there is no upgrade path onto or
      -- off this prototype and there must not be one; the guest's conversion is
      -- the only route out.
      flags = {
        "placeable-neutral",
        "player-creation",
        "not-upgradable",
      },
      hidden = true,
      hidden_in_factoriopedia = true,

      -- THIS MOD'S OWN LONE-PART PICTURE, cell (0,0) of the 47-variant sheet,
      -- so that the moment of conversion is not a visible change. A stub and
      -- the part that replaces it draw the same pixels.
      icon = "__better-belt-balancer__/graphics/icons/balancer-part.png",
      icon_size = 64,
      picture = {
        filename = "__better-belt-balancer__/graphics/entity/balancer-part-variants.png",
        priority = "high",
        width = 64,
        height = 64,
        x = 0,
        y = 0,
        scale = 0.5,
      },
    },

    -- THE MARKER, and it is defined HERE rather than unconditionally because
    -- its whole meaning is "the `balancer-part` prototype in this game is the
    -- stub above". The guest cannot tell an incumbent's prototype from ours by
    -- looking at `balancer-part` itself, and it must not convert some unknown
    -- fifth mod's entities; a point lookup of this name against
    -- `prototypes.entity` answers that in two host calls and no allocation.
    -- See guest/go/legacy.go, `legacyStubPresent`.
    --
    -- Nothing ever places one. It is a prototype that exists to be looked up.
    {
      type = "simple-entity",
      name = MARKER,
      hidden = true,
      hidden_in_factoriopedia = true,
      flags = { "not-blueprintable", "not-deconstructable", "not-on-map",
        "no-copy-paste", "not-selectable-in-game", "not-upgradable",
        "not-in-kill-statistics", "not-in-made-in", "hide-alt-info" },
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

-- THE ITEM, tested separately because the two can in principle come apart.
--
-- A stack of the incumbent's parts in a chest, in a player's inventory, in a
-- logistic request or on a belt is deleted with its prototype exactly as an
-- entity is, and there is no runtime pass that could get them back -- so the
-- item has to survive the same load. It does not need renaming and is
-- deliberately not renamed: `place_result` points at this mod's part, so a
-- legacy stack simply places this mod's balancers. Walking every inventory in
-- the game to rewrite stacks would be a scan of the whole world for a cosmetic
-- difference in what the stack is called.
--
-- The stack size is the incumbent's 50, so a full stack stays a full stack.
if not existing_item then
  data:extend {
    {
      type = "item",
      name = ENTITY,
      icon = "__better-belt-balancer__/graphics/icons/balancer-part.png",
      icon_size = 64,
      subgroup = "belt",
      order = "c[splitter]-y[bbb-balancer]-z[legacy]",
      place_result = "bbb-balancer-part",
      stack_size = 50,
      hidden = true,
      hidden_in_factoriopedia = true,
    },
  }
end
