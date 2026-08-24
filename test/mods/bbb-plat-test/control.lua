-- The SPACE AGE suite. Two legs, and they are here together because Space Age
-- is what they have in common and a second DLC-only suite would cost a second
-- Factorio run for one rig.
--
--   1. a balancer whose parts are on a SPACE PLATFORM surface while its network
--      is on the one global hidden surface, so its linked belts cross from a
--      moving platform to a surface that is not going anywhere;
--   2. BELT STACKING, which is a Space Age feature at the prototype level: a
--      loader's `max_belt_stack_size` is refused at load without the
--      `space_travel` feature flag, so no base-only suite can build a stacked
--      belt at all. This leg is what says a recompile hands a stacked network
--      back STACKED rather than merely conserved.
--
-- The stacking leg builds on its own flat scratch surface, on its own FORCE
-- (`bbb-stack`, `belt_stack_size_bonus = 3`), which is the other half of what
-- stacking needs and is also the guest's own gate: the platform leg's rigs are
-- on `player`, whose bonus stays 0, so one save exercises both arms of it.
--
-- THE `smix` BAND IS STACKED SUSHI, AND IT IS THE ONLY RIG IN THIS REPO THAT
-- REACHES `kindAt`'s MULTI-CANDIDATE BRANCH. `detailedTally` reads a stacked
-- line position by position and `kindAt` decides which (name, quality) total a
-- position belongs to; its `len(totals) == 1` branch is the only one any suite
-- had ever run, because every stacked rig above is single-kind iron plate and
-- every MULTI-kind rig lives in the base-only `mix` suite, where the stacking
-- gate is shut and `detailedTally` is never called at all. Multi-kind AND
-- stacked is Space Age, so it is here. See CLAUDE.md, "Stacked belts come back
-- stacked".
--
-- EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
-- Every column of parts is TWO columns -- a west part carrying the row's input
-- and an east part carrying its output -- because an edge is an interface
-- linked belt standing on the cluster's own tile and 2.1's collision validator
-- forbids two belt-connectables on one tile. What that costs a band is
-- GEOMETRY AND NOTHING ELSE: N, M, the stack sizes and the item kinds in
-- flight are properties of the BELTS, and the belts did not move.
--
-- Each band also carries ONE EXTRA EDGELESS PART below its west column, and it
-- is not decoration. Every recompile here is forced by laying a belt against
-- the cluster, and under this rule a working balancer HAS NO FREE FACE -- so
-- the belt each `recompile` used to put on the block's north face would now be
-- REFUSED and the band would measure a refusal instead of a teardown. The
-- edgeless part is the one tile a belt can still reach; `m2`'s conservation rig
-- and the interactive checklist's band B reached the same conclusion
-- independently. See agents/single-edge.md.
local PART = "bbb-balancer-part"
local BELT = "express-transport-belt"
local AUDIT = "bbb-audit"
local E = defines.direction.east

local function P(x, y) return { x + 0.5, y + 0.5 } end

--------------------------------------------------------------------------------
-- the belt-stacking leg
--------------------------------------------------------------------------------

local STK = "bbb-stk"       -- the scratch surface
local SFORCE = "bbb-stack"  -- the stacking force
local PITCH = 12

-- The five bands. `full`, `plain` and `smix` are DEAD-ENDED on purpose: a
-- network with nowhere to drain fills to capacity and then stops moving, which
-- is what makes a before/after sample taken inside one tick a measurement of
-- the teardown and of nothing else.
local BANDS = { ctrl = 0, full = PITCH, flow = 2 * PITCH, plain = 3 * PITCH,
                smix = 4 * PITCH }

-- How many ROWS each band's part block is, which under the one-belt rule is
-- half its part count -- and, with the edgeless part on the end, exactly where
-- `recompile` has to put its belt: (-1, BANDS[band] + ROWS[band]).
local ROWS = { full = 4, flow = 4, plain = 2, smix = 2 }

--------------------------------------------------------------------------------
-- the stacked-sushi band
--
-- Two sushi sources over SIX distinct names, none of which is `iron-plate` --
-- and that disjointness is load-bearing rather than tidy. Every other band in
-- this suite runs iron plate, so a name is enough to say which network an item
-- came out of, and the four bands above keep the numbers they were recorded
-- with because every count and every profile in this file filters by it. One
-- save, two independent measurements, no second Factorio run.
--
-- SEVEN NAMES AND NOT FORTY-EIGHT. The carry pool's bound is 32 (name, quality,
-- STACK SIZE) groups, and belt stacking multiplies a name by every stack size
-- standing on the lines at once -- so seven names is up to 28 groups, inside the
-- bound with room. Overflowing it is the `mix` suite's job ("More than
-- thirty-two kinds"); overflowing it HERE would spill, and a spill is what this
-- band asserts does not happen.
--------------------------------------------------------------------------------

-- ...AND ONE NAME AT THREE QUALITIES, which is the other branch. `kindAt`
-- settles a position by `name_is` and reaches for `LuaItemStack.quality` only
-- when the line's totals carry the SAME NAME twice -- the one thing a name
-- cannot decide. `plastic-bar` is on source 1's list three times over, at
-- normal, uncommon and rare, and the three are CONSECUTIVE bands so two of them
-- land on one hidden line. `quality` is an optional key of
-- `InfinityInventoryFilter` in the pinned 2.0.77 runtime API and the `quality`
-- mod is already loaded for this suite, so no new dependency is taken.
local SMIX_ITEMS = {
  {
    { name = "copper-plate" }, { name = "steel-plate" }, { name = "iron-gear-wheel" },
    { name = "plastic-bar" },
    { name = "plastic-bar", quality = "uncommon" },
    { name = "plastic-bar", quality = "rare" },
  },
  {
    { name = "copper-cable" }, { name = "electronic-circuit" }, { name = "stone-brick" },
  },
}

local SMIX_SET = {}
for _, band in ipairs(SMIX_ITEMS) do
  for _, it in ipairs(band) do SMIX_SET[it.name] = true end
end

-- `mine` splits every count in this file into the two independent measurements
-- the one save carries: false is the four iron-plate bands, true is `smix`.
local function mine(name, smix)
  if smix then return SMIX_SET[name] == true end
  return SMIX_SET[name] ~= true
end

-- The rotation period, and it is a MEASURED constant rather than a preference.
-- What `kindAt` needs is a hidden transport LINE carrying two names at once, and
-- a hidden belt tile holds four item positions per lane: a band longer than that
-- gives single-kind lines and the multi-candidate branch is never entered. A
-- stacking loader at express rate emits ~3 items/tick over two lanes, so four
-- ticks is ~1.5 stacked positions per lane -- comfortably shorter than a line.
-- The `smixlines` sample is what proves it landed, and assert-plat.py fails the
-- run if it did not. Longer periods were measured; see CLAUDE.md.
local SROTATE = 4

local function make_surface(name, rows)
  local s = game.create_surface(name, {
    width = 512, height = 512, water = 0, peaceful_mode = true,
    default_enable_all_autoplace_controls = false,
    autoplace_settings = {
      tile       = { treat_missing_as_default = false, settings = { ["grass-1"] = {} } },
      decorative = { treat_missing_as_default = false, settings = {} },
      entity     = { treat_missing_as_default = false, settings = {} },
    },
    cliff_settings = { richness = 0 },
    starting_points = {},
    property_expression_names = { cliffiness = "0" },
  })
  s.always_day = true
  s.request_to_generate_chunks({ x = 0, y = rows / 2 }, math.ceil(rows / 32) + 3)
  s.force_generate_chunk_requests()
  local area = { { -12, -8 }, { 12, rows + 8 } }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end
  s.destroy_decoratives { area = area }
  local tiles = {}
  for x = -12, 12 do
    for y = -8, rows + 8 do tiles[#tiles + 1] = { name = "grass-1", position = { x, y } } end
  end
  s.set_tiles(tiles, true, false, false, false)
  return s
end

local function sput(s, name, x, y, extra)
  local args = { name = name, position = P(x, y), force = SFORCE, raise_built = true }
  if extra then for k, v in pairs(extra) do args[k] = v end end
  local e = s.create_entity(args)
  if not e then
    error(string.format("bbb-plat-test: could not place %s at (%d,%d)", name, x, y))
  end
  return e
end

-- A stacking source: an infinity chest behind a loader whose prototype allows a
-- stack of 4. The force's bonus is the other half; without it this delivers
-- singles, which is the `plain` band's whole point.
local function source(s, x, y, stacking)
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = SFORCE }
  c.infinity_container_filters = { { index = 1, name = "iron-plate", count = 2000, mode = "at-least" } }
  sput(s, stacking and "bbbt-stackloader" or "bbbt-loader", x + 1, y, { direction = E, type = "output" })
end

-- A STACKED SUSHI source: the same stacking loader, but its chest holds ONE
-- filter at a time and rewrites it every SROTATE ticks, with
-- `remove_unfiltered_items` on so the previous kind is voided rather than left
-- for the loader to prefer.
--
-- One filter and not six. The `mix` suite measured the naive rig -- an infinity
-- chest carrying every filter at once -- and it delivers ONE kind: a loader
-- draws from the first stack it finds and the chest tops that same stack
-- straight back up (2,292 items in 1 of 6 kinds). Rotating a single filter gives
-- a BANDED belt, which is what a real sushi bus looks like anyway, and it is
-- deterministic: the band boundaries are a function of `game.tick` and nothing
-- else.
local function set_filter(c, it)
  c.infinity_container_filters = {
    { index = 1, name = it.name, quality = it.quality or "normal",
      count = 2000, mode = "at-least" },
  }
end

local function sushi_source(s, x, y, items, offset)
  local c = s.create_entity { name = "infinity-chest", position = P(x, y), force = SFORCE }
  c.remove_unfiltered_items = true
  set_filter(c, items[1])
  sput(s, "bbbt-stackloader", x + 1, y, { direction = E, type = "output" })
  storage.sushi[#storage.sushi + 1] = { chest = c, items = items, offset = offset }
end

local function rotate_sushi(tick)
  if not storage.sushi or tick % SROTATE ~= 0 then return end
  -- Factorio is Lua 5.2.1: there is no `//` operator, and `/` is always a float.
  local step = math.floor(tick / SROTATE)
  for _, sr in ipairs(storage.sushi) do
    if sr.chest.valid then
      set_filter(sr.chest, sr.items[(step + sr.offset) % #sr.items + 1])
    end
  end
end

local function sink(s, x, y)
  sput(s, "bbbt-loader", x, y, { direction = E, type = "input" })
  return s.create_entity { name = "steel-chest", position = P(x + 1, y), force = SFORCE }
end

local function chest_of(c)
  if not (c and c.valid) then return -1 end
  local n = 0
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do n = n + item.count end
  return n
end

-- The one thing an infinity chest is not is a conserved quantity: it mints and
-- voids items every tick by design, and `smix`'s sources void a whole band's
-- worth on every rotation. Counting one would make every total a measurement of
-- the source rather than of the balancer.
local function conserved(e)
  return e.type ~= "infinity-container"
end

local function stk_audit()
  game.surfaces[STK].create_entity {
    name = AUDIT, position = P(-30, 0), force = SFORCE, raise_built = true,
  }
end

-- Everything countable: items on the ground, items on every transport line, and
-- items in every chest. Conservation is only a claim if the sinks are counted
-- too -- `flow` has delivered thousands by the time it is recompiled.
-- `smix` selects which of the two independent measurements this save carries is
-- being counted, and `kinds`, when given, collects the per-NAME breakdown --
-- which is what the stacked-sushi band asserts on. A teardown that dropped one
-- kind and reinserted the rest conserves nothing, and a single total would have
-- to lose the same number of items twice in opposite directions to hide it.
-- A KIND IS A (NAME, QUALITY) PAIR, which is exactly the key `get_contents`
-- returns and exactly what `kindAt` has to tell apart. Counting by name alone
-- would let a teardown that handed back three normal plastic bars for three rare
-- ones pass every check in this file.
-- `quality` is a plain string on the (name, quality, count) rows `get_contents`
-- returns and a LuaQualityPrototype -- userdata -- on a LuaItemStack. Both
-- readings reach here, so both are named.
local function kindkey(item)
  local q = item.quality
  if q ~= nil and type(q) ~= "string" then q = q.name end
  return item.name .. "/" .. (q or "normal")
end

local function count_all(s, area, smix, kinds)
  local n = 0
  local function take(item, c)
    if not mine(item.name, smix) then return end
    n = n + c
    if kinds then
      local k = kindkey(item)
      kinds[k] = (kinds[k] or 0) + c
    end
  end
  for _, e in pairs(s.find_entities_filtered { area = area, type = "item-entity" }) do
    if e.valid and e.stack and e.stack.valid_for_read then take(e.stack, e.stack.count) end
  end
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and conserved(e) then
      local ok, m = pcall(function() return e.get_max_transport_line_index() end)
      if ok and m then
        for i = 1, m do
          for _, item in pairs(e.get_transport_line(i).get_contents()) do
            take(item, item.count)
          end
        end
      end
      local inv = e.get_inventory(defines.inventory.chest)
      if inv then
        for _, item in pairs(inv.get_contents()) do take(item, item.count) end
      end
    end
  end
  return n
end

-- The stack PROFILE of a set of transport lines: how many items, how many belt
-- POSITIONS they occupy, and the histogram of stack sizes over those positions.
--
-- This is the measurement the leg exists for. Conservation compares item
-- counts and was already exact before stacking was recovered; what was wrong is
-- that 72 items came back as 72 positions of 1 instead of 18 positions of 4.
local function hist_str(hist)
  local keys = {}
  for k in pairs(hist) do keys[#keys + 1] = k end
  table.sort(keys)
  local parts = {}
  for _, k in ipairs(keys) do parts[#parts + 1] = string.format("%d:%d", k, hist[k]) end
  return table.concat(parts, ",")
end

local function profile(s, area, smix)
  local items, positions, hist = 0, 0, {}
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and conserved(e) then
      local ok, m = pcall(function() return e.get_max_transport_line_index() end)
      if ok and m then
        for i = 1, m do
          for _, d in pairs(e.get_transport_line(i).get_detailed_contents()) do
            if mine(d.stack.name, smix) then
              local c = d.stack.count
              items, positions = items + c, positions + 1
              hist[c] = (hist[c] or 0) + 1
            end
          end
        end
      end
    end
  end
  return items, positions, hist_str(hist)
end

-- THE ANTI-VACUITY SAMPLE, and without it the band proves nothing.
--
-- `kindAt`'s cheap branch is `len(totals) == 1` -- one (name, quality) on the
-- line, no host call at all -- and it is the branch every stacked rig this repo
-- has ever built takes. What the multi-candidate branch needs is a single
-- transport LINE carrying two names at once, and it needs at least one of those
-- positions to be a STACK, because an unstacked line never reaches
-- detailedTally in the first place. Neither is something a rotation period can
-- be assumed into producing: if the sushi bands are longer than a line, every
-- line is single-kind and the whole leg passes while exercising nothing.
--
-- So this walks the hidden surface and reports, over the lines that carry
-- anything of `smix`'s at all: how many carry two or more distinct NAMES, how
-- many of those also carry a stacked position, and the richest line seen.
-- assert-plat.py requires both counts to be non-zero.
local function line_mix(s, area)
  local r = { lines = 0, multi = 0, multistacked = 0, qtie = 0, qtiestacked = 0,
              maxnames = 0, maxkinds = 0, maxstack = 0 }
  for _, e in pairs(s.find_entities_filtered { area = area }) do
    if e.valid and conserved(e) then
      local ok, m = pcall(function() return e.get_max_transport_line_index() end)
      if ok and m then
        for i = 1, m do
          local names, kinds, n, nk, stacked, big = {}, {}, 0, 0, 0, 0
          for _, d in pairs(e.get_transport_line(i).get_detailed_contents()) do
            if mine(d.stack.name, true) then
              if not names[d.stack.name] then names[d.stack.name] = 0; n = n + 1 end
              local k = kindkey(d.stack)
              if not kinds[k] then
                kinds[k] = true
                nk = nk + 1
                names[d.stack.name] = names[d.stack.name] + 1
              end
              if d.stack.count > 1 then stacked = stacked + 1 end
              if d.stack.count > big then big = d.stack.count end
            end
          end
          if nk > 0 then
            r.lines = r.lines + 1
            if n > r.maxnames then r.maxnames = n end
            if nk > r.maxkinds then r.maxkinds = nk end
            if big > r.maxstack then r.maxstack = big end
            if n >= 2 then
              r.multi = r.multi + 1
              if stacked >= 1 then r.multistacked = r.multistacked + 1 end
            end
            -- The quality tiebreak's own precondition: ONE name on this line
            -- carrying more than one quality. Nothing a `name_is` can settle.
            for _, q in pairs(names) do
              if q >= 2 then
                r.qtie = r.qtie + 1
                if stacked >= 1 then r.qtiestacked = r.qtiestacked + 1 end
                break
              end
            end
          end
        end
      end
    end
  end
  return r
end

-- The hidden surface is where the network lives, and it is the only place a
-- teardown can take items from. Its slots are laid out 32x72 from the origin.
local HIDDEN_AREA = { { -16, -16 }, { 2200, 400 } }

local function hidden()
  return game.surfaces["bbb-hidden"]
end

local VIS_AREA = { { -12, -8 }, { 12, 5 * PITCH + 8 } }

local function sample(tag)
  local s = game.surfaces[STK]
  local vis = count_all(s, VIS_AREA, false)
  local hid = hidden()
  local hn, hi, hp, hh = 0, 0, 0, ""
  if hid then
    hn = count_all(hid, HIDDEN_AREA, false)
    hi, hp, hh = profile(hid, HIDDEN_AREA, false)
  end
  log(string.format("[BBB-STK] %s total=%d visible=%d hidden=%d hitems=%d hpos=%d hist=%s",
    tag, vis + hn, vis, hn, hi, hp, hh))
end

-- The same sample for the stacked-sushi band, PER ITEM NAME, plus the profile
-- and the mixed-line evidence. Sorted by name, so two samples of the same world
-- produce the same lines in the same order on every machine: `pairs` order over
-- a Lua table is not a promise.
local function smix_sample(tag)
  local s = game.surfaces[STK]
  local kinds = {}
  local vis = count_all(s, VIS_AREA, true, kinds)
  local hid = hidden()
  local hn, hi, hp, hh = 0, 0, 0, ""
  local r = { lines = 0, multi = 0, multistacked = 0, qtie = 0, qtiestacked = 0,
              maxnames = 0, maxkinds = 0, maxstack = 0 }
  if hid then
    hn = count_all(hid, HIDDEN_AREA, true, kinds)
    hi, hp, hh = profile(hid, HIDDEN_AREA, true)
    r = line_mix(hid, HIDDEN_AREA)
  end
  local names = {}
  for name, n in pairs(kinds) do if n > 0 then names[#names + 1] = name end end
  table.sort(names)
  log(string.format(
    "[BBB-STK] smix tag=%s total=%d visible=%d hidden=%d kinds=%d hitems=%d hpos=%d hist=%s",
    tag, vis + hn, vis, hn, #names, hi, hp, hh))
  log(string.format(
    "[BBB-STK] smixlines tag=%s lines=%d multi=%d multistacked=%d qtie=%d " ..
    "qtiestacked=%d maxnames=%d maxkinds=%d maxstack=%d",
    tag, r.lines, r.multi, r.multistacked, r.qtie, r.qtiestacked,
    r.maxnames, r.maxkinds, r.maxstack))
  for _, name in ipairs(names) do
    log(string.format("[BBB-STK] smixkind tag=%s name=%s count=%d", tag, name, kinds[name]))
  end
end

-- A belt arriving against the band's EDGELESS part is a genuine new edge -- a
-- new INPUT, so the port count goes UP -- and the fingerprint moves, so the
-- network comes down and goes back up. It goes there rather than on the block's
-- north face because under the one-belt rule every part of a working balancer
-- already carries its belt and a belt on any of them would be REFUSED. The
-- audit marker in the same tick is what makes the rebuild happen inside this
-- dispatch rather than on the next tick, so "before" and "after" are one atomic
-- sample apart.
-- The profiler is around the AUDIT, not around a tick pair, because the sample
-- either side of it has to be atomic. That means it carries a whole-save
-- re-classification as well as the recompile -- the same trade `assert-m2.py`
-- documents -- so the number is only ever compared with the `audit only,
-- nothing pending` line below it and with the same measurement from another
-- build. It is NOT comparable with M2's tick-pair recompile timings.
local function recompile(band, tag)
  sample(tag .. " before")
  sput(game.surfaces[STK], BELT, -1, BANDS[band] + ROWS[band], { direction = E })
  local p = helpers.create_profiler()
  stk_audit()
  p.stop()
  log { "", "[BBB-STK] timing " .. tag .. " recompile (audit-forced) ", p }
  sample(tag .. " after")
end

-- The same gesture for `smix`, sampled per NAME. A belt arriving against the
-- edgeless part is a new INPUT edge, so the port count goes UP --
-- P = next_pow2(max(N,M)) takes a 2->2 from 2 to 4 -- and the network the
-- rebuild produces is strictly bigger than the one it replaced. That matters:
-- a SHRINK would legitimately spill whatever no longer fits (carry.go,
-- decision 4), and this band is about what the drain did with the kinds, not
-- about capacity.
local function smix_recompile()
  smix_sample("before")
  sput(game.surfaces[STK], BELT, -1, BANDS.smix + ROWS.smix, { direction = E })
  local p = helpers.create_profiler()
  stk_audit()
  p.stop()
  log { "", "[BBB-STK] timing smix recompile (audit-forced) ", p }
  smix_sample("after")
end

-- Every name is checked against `prototypes.item` and a missing one is a HARD
-- ERROR naming it, in the create log, rather than a band that quietly carries
-- fewer kinds than it claims -- which here would be a multi-kind rig that is
-- single-kind and passes every conservation check while proving nothing.
local function check_items()
  local missing, names, kinds = {}, 0, 0
  for name in pairs(SMIX_SET) do
    if not prototypes.item[name] then missing[#missing + 1] = "item " .. name end
    names = names + 1
  end
  for _, band in ipairs(SMIX_ITEMS) do
    for _, it in ipairs(band) do
      kinds = kinds + 1
      if it.quality and not prototypes.quality[it.quality] then
        missing[#missing + 1] = "quality " .. it.quality
      end
    end
  end
  table.sort(missing)
  if #missing > 0 then
    error("bbb-plat-test: no such prototype: " .. table.concat(missing, ", "))
  end
  log(string.format("[BBB-STK] smix item list ok: names=%d kinds=%d rotate=%d",
    names, kinds, SROTATE))
end

local function build_stacking()
  local s = make_surface(STK, 5 * PITCH)
  local f = game.create_force(SFORCE)
  -- The other half of belt stacking. It is a research result in a real game and
  -- it only ever goes up, which is why the guest may cache it for a dispatch.
  f.belt_stack_size_bonus = 3
  log("[BBB-STK] force " .. SFORCE .. " belt_stack_size_bonus=" .. tostring(f.belt_stack_size_bonus))

  -- ctrl: one uninterrupted stacked belt. The yardstick, and the thing that
  -- fails loudly if belt stacking silently did not happen at all.
  local b = BANDS.ctrl
  source(s, -5, b, true)
  for x = -3, 3 do sput(s, BELT, x, b, { direction = E }) end
  storage.stk_ctrl = sink(s, 4, b)

  -- Every band below is TWO COLUMNS OF PARTS plus one EDGELESS part on the end;
  -- `parts_for` lays them, so no band can forget either half of the rule.
  --
  --   x=-5 chest  -4 loader  -3..-1 belts  0 WEST PART  1 EAST PART
  --   x=2..4 belts   5 sink loader   6 chest      (dead-ended bands stop at 4)
  local function parts_for(band)
    local base, rows = BANDS[band], ROWS[band]
    for i = 0, rows - 1 do
      sput(s, PART, 0, base + i)
      sput(s, PART, 1, base + i)
    end
    sput(s, PART, 0, base + rows) -- the edgeless one; see `recompile`
  end

  -- full: 4 in, dead-ended out. Fills the hidden network with stacks and stops.
  b = BANDS.full
  parts_for("full")
  for i = 0, 3 do
    source(s, -5, b + i, true)
    for x = -3, -1 do sput(s, BELT, x, b + i, { direction = E }) end
    for x = 2, 4 do sput(s, BELT, x, b + i, { direction = E }) end
  end

  -- flow: 4 in, 4 out, running. Recompiled while stacked items are moving
  -- through it, and then measured for another 700 ticks.
  b = BANDS.flow
  storage.stk_out = {}
  parts_for("flow")
  for i = 0, 3 do
    source(s, -5, b + i, true)
    for x = -3, -1 do sput(s, BELT, x, b + i, { direction = E }) end
    for x = 2, 4 do sput(s, BELT, x, b + i, { direction = E }) end
    storage.stk_out[i + 1] = sink(s, 5, b + i)
  end

  -- plain: the same force, an ordinary loader, so the lines are UNSTACKED while
  -- the gate is open. This is the branch that costs one host call per non-empty
  -- line and then hands the flat totals back, and it has to conserve exactly.
  b = BANDS.plain
  parts_for("plain")
  for i = 0, 1 do
    source(s, -5, b + i, false)
    for x = -3, -1 do sput(s, BELT, x, b + i, { direction = E }) end
    for x = 2, 4 do sput(s, BELT, x, b + i, { direction = E }) end
  end

  -- smix: 2 in, 2 out, dead-ended, fed by STACKED SUSHI. The only rig in this
  -- repo whose hidden transport lines carry more than one item name AND more
  -- than one item per position at the same time, which is the pair of
  -- conditions `kindAt`'s multi-candidate branch needs to be reached at all.
  b = BANDS.smix
  parts_for("smix")
  for i = 0, 1 do
    -- The offset staggers the two sources so they do not switch to their first
    -- item on the same tick, which would band the whole rig in lockstep and
    -- leave every line single-kind after all.
    sushi_source(s, -5, b + i, SMIX_ITEMS[i + 1], i * 2)
    for x = -3, -1 do sput(s, BELT, x, b + i, { direction = E }) end
    for x = 2, 4 do sput(s, BELT, x, b + i, { direction = E }) end
  end

  stk_audit()
  log("[BBB-STK] init complete")
end

script.on_init(function()
  check_items()
  storage.sushi = {}
  build_stacking()
  local force = game.forces.player
  force.unlock_space_platforms()
  local ok, plat = pcall(function()
    return force.create_space_platform {
      name = "bbb-plat", planet = "nauvis",
      starter_pack = "space-platform-starter-pack",
    }
  end)
  if not ok or not plat then
    log("[BBB-PLAT] could not create a space platform: " .. tostring(plat))
    return
  end
  plat.apply_starter_pack()
  local s = plat.surface
  log(string.format("[BBB-PLAT] platform state=%s surface=%s",
    tostring(plat.state), s and s.name or "NIL"))
  if not s then return end

  local tiles = {}
  for x = -14, 14 do
    for y = -6, 6 do tiles[#tiles + 1] = { name = "space-platform-foundation", position = { x, y } } end
  end
  s.set_tiles(tiles, true, false, false, false)
  for _, e in pairs(s.find_entities_filtered { area = { { -14, -6 }, { 15, 7 } } }) do
    if e.valid and e.type ~= "character" then e.destroy() end
  end

  -- FOUR parts, not two: a west part carrying each row's input and an east part
  -- carrying its output, because one tile may carry one belt. The machine is
  -- the same 2->2 it always was -- N, M and P are properties of the belts.
  storage.out = {}
  for i = 0, 1 do
    for x = 0, 1 do
      local part = s.create_entity { name = PART, position = P(x, i), force = "player", raise_built = true }
      if not part then log("[BBB-PLAT] could not place a part") return end
    end
  end
  for i = 0, 1 do
    local c = s.create_entity { name = "infinity-chest", position = P(-6, i), force = "player" }
    c.infinity_container_filters = { { index = 1, name = "iron-plate", count = 1000, mode = "at-least" } }
    s.create_entity { name = "bbbt-loader", position = P(-5, i), direction = E, type = "output", force = "player", raise_built = true }
    for x = -4, -1 do s.create_entity { name = BELT, position = P(x, i), direction = E, force = "player", raise_built = true } end
    for x = 2, 3 do s.create_entity { name = BELT, position = P(x, i), direction = E, force = "player", raise_built = true } end
    s.create_entity { name = "bbbt-loader", position = P(4, i), direction = E, type = "input", force = "player", raise_built = true }
    storage.out[i + 1] = s.create_entity { name = "steel-chest", position = P(5, i), force = "player" }
  end
  -- The yardstick: one uninterrupted belt on the same platform.
  local c = s.create_entity { name = "infinity-chest", position = P(-6, 4), force = "player" }
  c.infinity_container_filters = { { index = 1, name = "iron-plate", count = 1000, mode = "at-least" } }
  s.create_entity { name = "bbbt-loader", position = P(-5, 4), direction = E, type = "output", force = "player", raise_built = true }
  for x = -4, 3 do s.create_entity { name = BELT, position = P(x, 4), direction = E, force = "player", raise_built = true } end
  s.create_entity { name = "bbbt-loader", position = P(4, 4), direction = E, type = "input", force = "player", raise_built = true }
  storage.ctrl = s.create_entity { name = "steel-chest", position = P(5, 4), force = "player" }
  -- The guest defers every recompile to the next tick (`fk.defer`), and
  -- `--create` never reaches one, so the network would otherwise be compiled on
  -- the first tick of the benchmark instead of into the save. `bbb-audit` is
  -- the marker that drains the queue synchronously.
  s.create_entity { name = "bbb-audit", position = P(8, 0), force = "player", raise_built = true }
  log("[BBB-PLAT] init complete")
end)

local function count(c)
  if not (c and c.valid) then return -1 end
  local n = 0
  for _, item in pairs(c.get_inventory(defines.inventory.chest).get_contents()) do n = n + item.count end
  return n
end

local function stk_report(tick)
  local outs = {}
  for i = 1, 4 do outs[i] = tostring(chest_of(storage.stk_out and storage.stk_out[i])) end
  log(string.format("[BBB-STK] t=%d ctrl=%d flow=[%s]", tick,
    chest_of(storage.stk_ctrl), table.concat(outs, " ")))
end

-- The schedule. 400 ticks is long enough for a dead-ended 4x4 to fill and stop;
-- the two recompiles are 100 ticks apart so that the guest's own log lines
-- cannot be confused between them, and `flow`'s measurement window is the 700
-- ticks after its own recompile.
local SCHEDULE = {
  [400] = function() sample("formed") end,
  -- The control every timing below is read against: the same whole-save
  -- re-classification with nothing to rebuild.
  [500] = function()
    local p = helpers.create_profiler()
    stk_audit()
    p.stop()
    log { "", "[BBB-STK] timing audit only, nothing pending ", p }
  end,
  [600] = function() recompile("full", "full") end,
  [700] = function() recompile("plain", "plain") end,
  [800] = function() recompile("flow", "flow") end,
  -- `smix` last of the four, and 100 ticks after `flow` for the same reason the
  -- others are spaced: the guest's own log lines cannot then be confused between
  -- two recompiles. By t=900 each sushi source has run its three-item list
  -- seventy-five times, so the dead-ended network is frozen holding a full
  -- cross-section of bands.
  [900] = function() smix_recompile() end,
  -- 200 ticks after its own recompile, so the window measures a network that is
  -- running rather than one still refilling: a rebuild puts every drained item
  -- back at the HEAD of the butterfly, so the outputs are briefly starved by
  -- construction. The `edge` suite measures the same shape the same way.
  [1000] = function() stk_report(1000) end,
  [1500] = function() stk_report(1500) end,
  -- After the last sample, so it cannot disturb one, and after every recompile
  -- this suite makes. Nothing has touched either surface since tick 900, so the
  -- registry and the world must agree exactly -- and the cluster and part
  -- counts are what say the bands are the shape this suite thinks they are,
  -- which no stack profile and no rate can. assert-plat.py reads the LAST audit
  -- line in the run, which is this one.
  [1520] = stk_audit,
}

script.on_event(defines.events.on_tick, function(e)
  rotate_sushi(e.tick)
  if e.tick == 600 or e.tick == 1500 then
    log(string.format("[BBB-PLAT] t=%d ctrl=%d out=[%d %d]", e.tick,
      count(storage.ctrl), count(storage.out and storage.out[1]), count(storage.out and storage.out[2])))
  end
  local f = SCHEDULE[e.tick]
  if f then f() end
end)
