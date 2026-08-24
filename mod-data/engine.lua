-- WHICH ENGINE THIS IS, asked once and answered the same way in both stages.
--
-- The whole 2.1 port turns on one fact -- can this Factorio put two
-- belt-connectables on one tile -- and TWO STAGES have to agree about it:
--
--   the DATA stage      emits `not_colliding_with_itself` and the `bbb-can-stack`
--                       marker prototype on 2.0.x and never on 2.1.x
--                       (prototypes/hidden.lua)
--   the SETTINGS stage  defines `bbb-multi-edge-parts` on 2.0.x and never on
--                       2.1.x, because a setting that cannot do anything is a
--                       dead toggle in the player's menu (settings.lua)
--
-- They are separate Lua states with separate globals and no shared runtime, so
-- "agree" can only mean ONE FILE. Two copies of a version match would compile,
-- would work, and would be one edit away from a game where the compiler believes
-- it may stack and the prototype it would stack cannot -- which is a silent nil
-- from `create_entity` on every second interface, forever.
--
-- `mods` is what makes one file possible: it is the dictionary of every
-- installed mod's version, base included, and it is visible in BOTH stages
-- (measured on 2.1.14 -- the settings stage sees `mods` and `feature_flags`).
--
-- IT FAILS SAFE TOWARDS 2.1. Anything unreadable is treated as 2.1, because
-- emitting the flag on 2.1 refuses the mod at load, while not emitting it on 2.0
-- merely costs the multi-edge geometry -- which the guest then refuses to build
-- rather than building wrongly. The match is on MAJOR.MINOR alone, so every
-- 2.0.x point release is 2.0 and everything from 2.1.0 on is not.
--
-- See agents/single-edge.md and guest/go/sedge.go for the runtime half.

local M = {}

function M.base_is_2_0()
  local v = mods and mods["base"]
  if type(v) ~= "string" then return false end
  local major, minor = v:match("^(%d+)%.(%d+)")
  return tonumber(major) == 2 and tonumber(minor) == 0
end

return M
