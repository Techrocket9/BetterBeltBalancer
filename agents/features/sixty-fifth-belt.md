# The sixty-fifth belt — refusing without demolishing

**`plan.MaxPorts` is 64 and refusing past it was always correct. What the refusal did FIRST was tear down the working balancer it was refusing to replace.** `compile()` classified the edges, saw the fingerprint move, called `teardownForRebuild(root)`, and only then asked `plan.Build` whether the shape fit — so one belt too many against a 64-port balancer demolished it, built nothing, spilled the whole machine on the floor and said so only in the log file. Found by reading `agents/maxports.md` §4, which had it written down as a queued task; closed 2026-08-04.

**Everything below is one statement moving twenty lines up, and everything around it.** `plan.Shape(n, m)` needs the edge COUNTS and nothing else, and they are in hand the moment `classifyEdges` returns.

## Measured, both sides of the change

The `edge` suite's new `lim` rig: a column of thirty-two parts with a belt on both sides of every one of them pointing inwards — **sixty-four inputs and one output, which is P = `next_pow2(64)` = `MaxPorts` exactly**, 1,026 hidden entities, the biggest network this mod builds. A sixty-fifth input belt is laid on it while it runs and then mined off again. Same rig, same schedule, the only difference being where the check sits:

| | check AFTER the teardown | **check BEFORE it** |
|---|--:|--:|
| the standing network | **torn down, 1,876 items drained** | untouched |
| items on the ground | **1,690** (`ground` 336 → 2,026) | **0** |
| delivery over 246 ticks, before → after the edit | 184 → **10** | 184 → **185** |
| the audit afterwards | `nets=9 drift=0 unbuilt=1` | `nets=10 drift=1 unbuilt=0` |
| the log | `[BBB] error:` **three times** for one belt | one `[BBB] alert:` |
| `test/run.sh` | **killed the run** on the first error line | green |

The ten items in the "after" column of the failing run are the output belt draining; the machine had stopped. And **the item total was conserved to the item throughout**, which is exactly why eight suites were green over this for five milestones: the defect was never a count, it was where the items were and whether the machine still existed.

## The four pieces, and the one that is sharp

`guest/go/limit.go`, whose header is the long form.

1. **The check moves in front of the teardown** (`compile.go`). `overLimitShape(edges)` mirrors `plan.Build`'s own two tests — a cluster with no inputs or no outputs is a legitimate half-built state at ANY size, so refusing one here would refuse every single-sided cluster in the world. `plan.Build` keeps its `!fits` branch as an unreachable backstop and it is still an `error:`, so a crack between the two mirrors fails a run.
2. **The player is told.** `noteBuiltByPlayer` writes down `(surface, tile, force, player_index, isPart)` for every addition a PLAYER makes on or beside a cluster — the miner's-pocket pattern exactly, scalars only, **no host call**, and nothing at all when `player_index` is zero. `on_built_entity` is the only build event that carries one, so a robot, a script build, a revive and a clone all record nothing, and so does every event in every headless suite. At refusal time the player is resolved FRESH with `game.get_player` and gets `create_local_flying_text` at the refused piece — the box centre until the 2026-08-05 interactive check found that on a 32-part column it spawned seventeen tiles off the placing player's screen — in **vanilla's cannot-build red** rather than the default white, plus `utility/cannot_build`; a robot or script build gets `force.print` with a **different** locale key, because "you got it back" would be a lie. The sentence itself counts BELTS PER SIDE and not ports, and deliberately does not narrate the hand-back: both are the same playtest's, and all four are written up in "The wake race" below.
3. **The piece is handed back, and WHERE that runs is the whole design.** `revertOverLimit` is called from `flush()` **after `endCarry()`** — after every teardown, every build and the settling of the carry transaction — and may not run any earlier, for three reasons that are each sufficient: `mine_entity` dispatches `on_player_mined_entity` SYNCHRONOUSLY, so it re-enters the registry, frees nodes, re-roots components and refills the very queues `flushLive` is iterating; the compile buffers (`tileBuf`, `edgeBuf`, `opBuf`, `entBuf`) are package level and not re-entrant; and a claim recorded before `endCarry` would be settled against pools the miner had nothing to do with. Recorded after, it lives exactly one flush and is answered by the teardown the revert itself provoked. The mine takes the fingerprint back to what the `netInfo` already holds, so the next flush is a **SKIP**: at no point in the whole sequence is anything torn down or rebuilt. `force = nil` on the mine, which is vanilla's rule — a full inventory declines and the piece stays standing rather than evaporating.
4. **The feedback gate, which is not optional.** A refused cluster is re-queued by every audit and by every event within two tiles of it, and its fingerprint can never match its `netInfo`'s, so it reaches the refusal EVERY TIME — three times for one belt in the measurement above. `overLimit` is a root → refused-fingerprint map, **point-queried only, never ranged**, nil until the first refusal (a lookup and a `delete` on a nil map are both free, so a save that never hits the cap never allocates it), and it fires the message once per distinct edge state. The suite asserts **exactly one**. **A refusal issued from inside `rebuildFromWorld` does not arm it**, and that is not a detail: the memo is what silenced the one refusal that had never been delivered, the first time a player laid the sixty-fifth belt as the first event of a session. See "The wake race" below.

## What is verified, and the one thing that is not

The `edge` suite's LIM section asserts the refusal line and its numbers (128 ports for 65 inputs against a limit of 64), one refusal per edit, zero ground items, an unmoved conserved total, the audit's `drift=1 unbuilt=0`, and delivery holding across the edit as a ratio between two equal-length windows — with a `before` window that must be non-zero, so a dead control fails rather than passing vacuously.

**The ROBOT arm of the feedback is verified and the PLAYER arm is not**, and the split is exactly where the wall is. Every build in every suite is a script build, so `player_index` is zero and the fork always takes `force.print` — which means the LocalisedString really does cross the boundary and the `LuaForce` really is resolvable from a force INDEX, which is the part that could realistically fail (the registry keeps no force handle; it comes off a part standing on the cluster). Neither is readable back from Lua — `force.print` goes to the game's chat and `--benchmark` does not log it — so the guest writes one line saying it happened and the suite asserts that line, **once, with no error**.

**The flying text, the sound and the hand-back need a player, and a `--create` has none**: `game.get_player` resolves to nothing and `revertOne` returns before it mines anything, so what the `edge` suite asserts there is the NEGATIVE — **zero pieces handed back over the whole run** — exactly as it asserts zero pockets, and a revert firing for a script build fails the run. **THE HAND-BACK ITSELF IS MEASURED SINCE 2026-09-14**, in the `curs` suite, which loads a save a client made rather than generating one: the sixty-fifth belt laid from a cursor comes back to the player, and the same gesture with the inventory full leaves it standing and says so. The flying text and the sound stay on the interactive checklist, which is about how the gesture looks.

**What the user still tests interactively is how it LOOKS**, the gesture itself being driven headlessly by the `curs` suite since 2026-09-14: build a balancer with sixty-four belts against it, leave one part of it with no belt, and lay a sixty-fifth belt against THAT part. (Under the one-belt-per-part rule a sixty-fifth belt laid anywhere else is the other refusal, so the spare part is what makes this the port-limit gesture; the checklist's band C stages it.) Expected — the flying text naming the limit, the standard cannot-build sound, the belt back in the inventory, and the balancer still running exactly as it was. Then do it again with the inventory full: the message still appears and the belt stays standing unconnected, with a `[BBB] alert: player … could not be handed back` in the log. Then have a construction robot place it from a ghost: a force-wide chat message instead, and the belt stands. Grep for `handed the refused piece` for the first case; the second and third are the two lines either side of it. Both hand-back lines NAME THE BOUND THAT FIRED -- `(over the port limit)` here, `(past the one-belt-per-part rule)` for the belt gesture -- because `tellRefusal` and `revertOne` are shared by both bounds and the sentence used to name only this one.

## The shape this left uncovered, and the pass that covered it

**A part bridging two clusters into an over-limit one demolished both**, and it shipped that way for a day. `AddPart` marks the two predecessors DEAD, so `flushDead` had already brought their networks down by the time `flushLive` reached the merged cluster and the check above had nothing left to protect. It is closed — see "The merge that would be over the limit" below, and `agents/maxports.md` §5.

## What it costs

Package built 2026-08-04, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 300,459 B | **310,630 B** | +3.38% |
| `fk_module.lua` | 2,319,175 B | **2,432,188 B** | +4.87% |
| `dist/bbb.wasm` | 1,039,715 B | 1,076,918 B | |
| members bound into the mod | 38 | **42** | of 4,257 |

The four new members are `LuaPlayer.create_local_flying_text`, `LuaPlayer.play_sound`, `LuaControl.mine_entity` and `LuaForce.print`. No new FkLua gap: every one was reachable through the generated bindings as they stood, and `LuaSurface.find_entity` and `game.get_player` were already bound.

**Nothing on any hot path moves, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — which is the check that says a build note recorded on the guest's highest-multiplier path costs nothing where no player builds. `flush()` gains one length test on a nil slice; `compile()` gains a loop over an edge list it has just walked anyway, and a `delete` on an empty map.
