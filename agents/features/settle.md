# The settle: a burst of builds or mines on a balancer is compiled once

Code: `guest/go/settle.go` (the rule), `onEvent` and `onDeferred` in `guest/go/main.go` (which dispatch is a gesture, and where the rule is asked), `markDead`/`markLive` and `flush` in `guest/go/compile.go` (the counters, the flush depth, the end of a burst), `removePart` in `guest/go/cluster.go` (a removed part's tile, recorded for the cut).

**Why.** A mod-portal report (GriimVT, 2026-11-06): lag when placing ten or more connected parts. `fk.Defer()` bounds a recompile at one per cluster per TICK, and a player dragging parts beside belts places one every few ticks, so a twenty-part drag was a teardown-and-rebuild per tile of a network one part larger each time. Taking a running balancer apart by hand, and robots building or deconstructing one, are the same shape. What one rebuild costs is `bench/baselines/RESULTS.md`, "What one edit costs": a running 4x4 27 ms on 0.3.7, a 16x16 123 ms.

**The rule.**

- Only a gesture is held: a build or a mine by a player, a construction robot or a space platform (`isGesture`: `on_built_entity`, `on_robot_built_entity`, `on_space_platform_built_entity` and the three `*_mined_entity`). Every `markDead`/`markLive` made from any other dispatch, or from inside a flush (`flushDepth`, which covers an over-limit revert's own `mine_entity`), sets `foreignTouch`, and a queue with a foreign touch in it drains on the next tick as it always did. Scripts (`script_raised_*`), clones, rotations, flips and deaths keep the next-tick contract. The first cut held everything, and the `m1` suite, which edits by script ten ticks apart, read its part sprites a phase late; every suite in the estate is written against that contract, and so may other mods be.
- An isolated build is unchanged: with nothing compiled by a deferred flush in the last `burstWindow` (12) ticks, the flush drains on the tick after it.
- A build within `burstWindow` ticks of a deferred flush that drained work opens a burst. So does a mine with no build beside it in the queue, on its own. The queues are kept and the one-shot re-armed; nothing is torn down, so the standing network keeps running the shape it had.
- A mine with a build beside it (a belt fast-replaced onto a part) is answered like a build: the belt is a port the player is waiting for. The `curs` suite's column gesture failed its audit until this was so.
- A part mined inside a burst takes its own tile with it on the next tick (`cutRemoved`): the visible interfaces on that one tile are drained and destroyed outside a carry transaction, so the pool goes to the miner's pocket by the claim `noteMinedByPlayer` made on that tile, or to the floor, and the claims are NOT reset (the flush that ends the burst still needs them for the rest of the machine). Nothing stands on a tile a player has just cleared. No item is held across ticks: a cut tile's pool is settled in the tick it is cut, and everything else is still on a belt.
- A burst is compiled once `settleQuiet` (12) ticks pass with nothing new queued, or `settleCap` (600) ticks after it opened. When a robot's or a platform's build or mine is in the queue, `robotWindow` and `robotQuiet` (60 each) replace `burstWindow` and `settleQuiet`.
- Any other pending work drains at once: a legacy conversion, an insert probe, an over-limit revert.
- Every synchronous flush (audit marker, clone, surface reconcile, a priority keypress) drains both queues and ends the burst.

**Why 12 for a player.** At 30 the `curs` suite's separate clicks, 15 and 29 ticks after an earlier gesture, were held, and four of its samples (a refusal's hand-back, a curve classification) read the world a third of a second early. A click is a gesture of its own and its answer belongs on the next tick. A character running while dragging lays a tile every 6 or 7 ticks, which 12 covers with room; a slower drag than one tile per 12 ticks rebuilds per tile as before.

**Why a mine opens a burst on its own.** With the opening rule alone, the first mine of a removal drag drained on the next tick, and that was the drag's dearest tick: the whole running 10->10 torn down and rebuilt as a 10->9 with 819 items carried, 89.0 ms in the `curs` removal drag. A mine's visible answer is its tile going empty, which the cut gives on the next tick either way, so holding the rest costs a player nothing they can see. A lone mine's survivors are rebuilt 12 ticks later instead of 1.

**Why 60 for robots.** Measured by the bench's robot phase at 12: eighteen of twenty parts landed within 92 ticks and were held into one compile, then the last two arrived 37 and 48 ticks after it, and each rebuilt the whole running network (58.0 and 77.3 ms). No cursor is waiting on a robot. A second covers the jitter inside one wave of robots; a robot's round trip back to a roboport is not covered.

"Nothing new queued" is a counter, `queueTouches`, bumped by every `markDead`/`markLive` whether it deduplicates or not, compared once per deferred tick. The clock is `game.tick`, one host call per deferred tick and only on a tick with queued work. Deterministic: tick and guest state only. An idle guest still registers nothing.

FkLua's `flush_deferred` (`runtime/lua/fk_mod.lua`) unregisters the one-shot BEFORE dispatching `fk_on_deferred`, so `fk.Defer()` from inside it re-arms; that is what a held tick does.

**What it costs a player.** The first click of a drag connects on the next tick; the finished shape about a fifth of a second after the last click. While a burst is held the new parts carry the sprite they were placed with and a belt laid against one is not yet connected. A mined part's tile is empty on the next tick; the rest of the machine keeps running its old shape until the burst settles, and a machine taken apart by hand no longer spills at each shrink, because its items are carried into the survivors (or the pocket) once, at the end.

**Measured** (Factorio 2.1.24, M3 Pro). The "off" rows are the same tree with `holdFlush` returning false.

The `curs` suite's band (k): a player in the committed fixture clicks twenty parts from the cursor, one every 6 ticks, beside ten fed and drained rows, ending on a 10->10; 171 ticks profiled from just before the first click, the rest of the suite's world running underneath. Band (k2): after the final audit the same player mines the twenty parts off again, one every 6 ticks, east column first, 171 ticks profiled the same way. `make test SUITES=curs`:

| | compiles | teardowns | the 171 ticks in all | worst tick | ticks over 16.7 ms |
|---|--:|--:|--:|--:|--:|
| drag, off | 10 (10->1 up to 10->10) | | 617.0 ms / 804.5 ms | 56.9 / 64.2 ms | 10 / 10 |
| drag, on | 1 (10->10 over 16 ports) | | 181.2 to 245.2 ms | 25.5 to 30.1 ms | 1 |
| removal drag, off | 9 (10->9 down to 10->1) | 10 | 1070.5 ms | 97.2 ms | 10 |
| removal drag, on | 0 | 1 | 248.7 / 283.4 ms | 29.2 / 31.0 ms | 1 |

The off runs are the bands' red proof: they fail on exactly the two compile assertions and nothing else. The drag's one compile is one rather than two because the west column alone has inputs and no outputs and builds no network. The removal drag's one teardown carries 1271 items into the player's inventory and none to the floor, with each cut tile's two or so items pocketed on the tick after its mine; the off run pocketed 1305 in one go at the dissolve, after nine shrinks had carried the rest. The remaining worst tick in both directions is the one teardown or rebuild that a burst ends in.

The bench's drag rig, `bench/run.sh --mod <zip> --scenario saturated --hitch -n 4 -k 4 --ticks 2700 --runs 1`, three phases, added time over the idle tick median:

| | compiles | added | worst tick | ticks over 16.7 ms |
|---|--:|--:|--:|--:|
| scripted drag (`raise_built`), not held by design | 10 | 564.7 to 585.0 ms | 61.9 to 65.4 ms | 10 to 11 |
| robots deconstruct the 20 parts, off | 4 | 426.1 ms | 96.5 ms | 5 |
| robots deconstruct, on | 0 | 81.9 / 110.6 ms | 28.7 / 27.5 ms | 1 |
| robots build them from ghosts, off | 11 | 443.7 ms | 95.4 ms | 7 |
| robots build, on, quiet 12 | 3 | 210.5 ms | 77.3 ms | 3 |
| robots build, on, quiet 60 | 1 | 110.3 / 159.0 ms | 30.3 ms | 1 |

The pairs are two runs of the same build; "added" moves with the idle median it is taken over (0.116 to 0.187 ms per tick across these runs, times 561 ticks). The scripted drag is unbatched under the rule above; the first cut, which held scripted builds too, took it to 78.0 / 74.8 ms and 25.1 / 24.7 ms over one compile.

**Not done.** A player's drag slower than one tile per 12 ticks is not held, and robots arriving more than a second apart (a second wave from the roboport) rebuild once per wave.
