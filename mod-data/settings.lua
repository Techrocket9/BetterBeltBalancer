-- THE SETTINGS STAGE, and there is exactly one setting in it.
--
-- `bbb-multi-edge-parts` is the per-save policy half of the 2.1 port's rule:
-- Factorio 2.1 allows one belt-connectable per tile, so a balancer part carries
-- one interface and therefore serves ONE BELT (guest/go/sedge.go,
-- agents/single-edge.md). On 2.0 the collision-mask loophole that permitted two
-- is still open, and a save built before the rule existed must not be broken by
-- an update -- so multi-edge survives there, opt-in, defaulting to off.
--
-- ---------------------------------------------------------------------------
-- RUNTIME-GLOBAL, AND THAT IS FORCED RATHER THAN PREFERRED
-- ---------------------------------------------------------------------------
--
-- The grandfather pass has THE MOD flip this setting -- a save updated from the
-- release that had no setting keeps its multi-edge balancers working, and the
-- only way to express that is for the guest to write `true` on the first load
-- (guest/go/sedge.go, `grandfatherMultiEdge`). A script can write
-- `settings.global` and can NEVER write a startup setting: measured on 2.1.14,
-- `settings.startup` answers `LuaCustomTable is read only`.
--
-- What used to force startup was the collision flag being a data-stage
-- decision, and that is dissolved by splitting the two questions the first
-- design conflated. CAN the engine stack is a fact about the Factorio version
-- and is answered by prototypes/hidden.lua's `bbb-can-stack` marker; MAY the
-- compiler use it is this setting. The effective rule is the AND, and
-- guest/go/edgemode is that fold with its eighteen states proved under `go
-- test`.
--
-- Runtime-global buys two more things startup could not: the player flips it
-- mid-save with no restart, and the flip arrives as an ordinary replicated
-- event (`on_runtime_mod_setting_changed`) instead of a whole load cycle.
--
-- ---------------------------------------------------------------------------
-- DEFINED ON 2.0.x AND NEVER ON 2.1.x
-- ---------------------------------------------------------------------------
--
-- No dead toggles: on 2.1 nothing this setting could say would change what the
-- engine permits, so it is not in the menu at all. Two consequences the guest
-- depends on, both measured on 2.1.14:
--
--   READING an undefined runtime setting returns nil and raises nothing, so the
--   guest's policy read needs no version gate -- nil IS the "not defined on this
--   engine" answer (guest/go/sedge.go, `settingMultiEdge`).
--
--   WRITING one RAISES (`LuaCustomTable doesn't contain key ...`), so the
--   grandfather pass's write is gated on the `bbb-can-stack` marker as a
--   CORRECTNESS matter and not as policy. A 2.0 save opened on 2.1 is full of
--   exactly the clusters that pass looks for, so a fold that forgot the marker
--   would raise inside the load of every save the migration exists for. That
--   negative is the one half of this feature a 2.1-only test estate can pin, and
--   `TestGrandfatherNeverWritesWhereTheKeyDoesNotExist` is where it is pinned.
--
-- `mods` IS visible in this stage (measured, along with `feature_flags`), which
-- is what lets the version branch be shared with the data stage instead of
-- duplicated. See mod-data/engine.lua.

if not require("engine").base_is_2_0() then return end

data:extend {
  {
    type = "bool-setting",
    name = "bbb-multi-edge-parts",
    -- Map, not global-per-user: what it controls is the geometry of machines
    -- standing in the save, so it has to be one answer for everybody in a
    -- multiplayer game and it has to travel with the save.
    setting_type = "runtime-global",
    -- FALSE, so that a 2.0 save which never used multi-edge is bit-compatible
    -- with a fresh single-edge world -- which is the save that upgrades to 2.1
    -- losing nothing. A save that DOES use it is flipped up by the grandfather
    -- pass on its first load under this version, once, with a warning.
    default_value = false,
    order = "a",
  },
}
