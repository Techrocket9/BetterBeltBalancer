# MaxPorts — the 64-port cap, what actually binds it, and the uncap path

Status, in two halves. **RAISING the cap: direction documented, deliberately not implemented** (user decision 2026-08-04) — §1–§3, so that the day someone raises it they start from the three real constraints rather than rediscovering them, and so that nobody "fixes" the cap by bumping the constant, which compiles, runs and is wrong twice over. **HITTING the cap: DONE** — §4 (2026-08-04) for a balancer that grows an edge it cannot have, and §5 (2026-08-05) for two balancers MERGED into one that is over the limit, which §4 left written down as uncovered. Both are shipped code with a measurement from each side of the change and an `edge`-suite leg.

**§3's "measure first" is DONE (2026-08-05) and it changed the answer.** The `mega` bench scenario built the first real 64×64 and timed a recompile of it: **155 ms empty, ~390 ms saturated**, against estimates in this file that said a network four times bigger would cost 30. The candidate range for a raised bound was 256 and 512; on the measurement it is **128 at most**, and the reasoning that got there is unchanged — only the numbers under it moved.

## 1. Where 64 comes from — two of our decisions, no engine limit

Factorio imposes nothing here: surfaces are effectively unbounded, splitter chains and linked-belt pairs are unlimited. The cap is ours, twice:

- **The slot geometry.** The hidden surface is carved into fixed **32×72-tile slots** (`compile.go`: `slotW`/`slotH`), one network per slot, and `slotOrigin` is a bijection the whole adoption path leans on — a hidden entity's *position is a slot number*, which is how `rebuildFromWorld` adopts a network without any stored state. A P-line network needs P rows × `Width(P)` columns: P=64 needs 19×64 and fits; P=128 needs 22×128 and the **72-row slot height binds**. 72 = 64 + 8 margin — the slot was sized *for* 64, not the other way round.
- **A judgment call** (`plan.go`): "64 lines is six stages and ~500 entities, which is already far past any balancer a player builds." (That "~500" undercounts, by the arithmetic that reproduces the measured 84 entities of an 8×8: head 2(N+Loop) + splitters kP/2 + jumper columns 2P(k−1) + sends + visible gives **~1,150 for a 64×64** — the jumper columns dominate at depth. The judgment stands; the number in the comment doesn't. **The game has since built one and it is `1152 entities`, exactly what that arithmetic gives** — §3.)

The engine fact that lives nearby — a transport line accepts one item per tick per lane, so the hidden tier's 0.25 speed is the compression ceiling — bounds per-line RATE, never port count. `P >= max(N, M)` keeps it irrelevant at any P.

## 2. What binds at P > 64, and the fix for each

| constraint | naive move | why it's wrong | the real fix |
|---|---|---|---|
| slot height | variable-size slot allocator | breaks the position→slot bijection adoption depends on; fragmentation; ~real complexity | **power-of-two size classes**: P≤64 keeps today's grid and code path byte for byte; each larger class gets its own band of taller cells at a disjoint coordinate range with its own LIFO free list — today's 15-line allocator stamped out per class, position→slot still computable (band → class, then the same divide). Coordinates are free: the hidden surface generates no chunks and entities on ungenerated chunks run at full rate (S1) |
| `plan.go`'s working buffers, sized `[MaxPorts]` | bump the constant | `ordBuf` at P=1024 is ~50–60 KB of **globals**, and the conservative collector rescans all globals at every paced step's termination attempt — this repo already paid for that lesson once ("The root scan that could not fit in a step"), and `test/run.sh` now FAILS the run on the collector's root-set warning line. A static-array bump trips the gate it was built to trip | **lazily-grown package-level slices**, high-water, grown only when a larger P than ever seen arrives. Heap contents are scanned once per collection when marked, not once per paced step forever. `TestBuildDoesNotAllocateOnAReusedBuffer` keeps passing — warm buffers stay zero-alloc at any P |
| the per-edit hitch | nothing — it is physics | recompile is full teardown-and-rebuild at ~12.6 µs per host call, linear in entities plus items in flight | **keep a loud refusal**, moved to where the hitch budget dies rather than where the old slot died. See §3 |

## 3. Why "remove the cap" should mean "raise the bound", not "no bound"

The cap does two jobs today. Protecting the neighbour's slot becomes obsolete under size classes. Bounding **the worst single-tick stall a player can construct** does not, ever: entities scale ~P·log2 P, and reinsertion scales with the thousands of items a big saturated network holds. Full-recompile is the architecture; an uncapped P is an uncapped multiplayer freeze that a pasted blueprint triggers silently, on every client, in lockstep. Incremental compilation across ticks is the only true escape and is exactly the "massively overcomplicating" trade that was declined.

### The measurement — 2026-08-05, and it retires the estimates that were here

**This paragraph used to say "nothing in this file is a measurement yet" and carry two projections. The `mega` bench scenario has now built a real 64×64 in a real game and timed a real recompile of it, and both projections were an order of magnitude optimistic.** Full method, tables and the surrounding megabase figures: CLAUDE.md, "The megabase cell". In short —

```sh
bench/run.sh --mod bbb --scenario mega      --hitch -n 40 --ticks 2400 --runs 1
bench/run.sh --mod bbb --scenario mega-idle --hitch -n 40 --ticks 2400 --runs 1
```

M2's tick-pair pattern (profiler opened in the mutating tick, closed in the flushing one), one input belt removed and restored, median of three, minus that run's own `idle tick pair, nothing pending` control:

| a 64×64 (P = 64 = `MaxPorts`) | measured |
|---|--:|
| entities in the compiled network | **1,152** |
| first compile, at create (audit-subtracted, 3 creates) | **138–162 ms**, median 147 |
| teardown-and-rebuild, network **EMPTY** | **155–159 ms** |
| teardown-and-rebuild, network **SATURATED** | **367–393 ms**, still rising at 410–427 |
| delivery afterwards | 64 outputs, 99.2% of a full express belt each, balance 1.0028 |

**Two things that measurement settles.** First, §1's entity arithmetic is exactly right: the derivation there gives 1,152 for P=64 and the game built 1,152. Second, the cost is **linear in entities** and the constant is the host boundary as always — an 8×8 is 84 entities and ~11.5 ms empty, a 64×64 is 13.7× the entities and 13.4× the time.

So the projection can be re-run on a measured base instead of a guessed one, using §1's own formula (`head 2(N+Loop) + kP/2 + 2P(k−1) + sends + visible`) and the measured 155 ms / 1,152 entities:

| P | entities | **empty hitch** | saturated, ≥2.5× (items scale too) |
|---|--:|--:|--:|
| 64 (today) | 1,152 | **155 ms** *(measured)* | **~390 ms** *(measured)* |
| 128 | 2,624 | ~350 ms | ~0.9 s |
| 256 | 5,888 | ~790 ms | ~2.0 s |
| 512 | 13,056 | ~1.8 s | ~4.4 s |
| 1024 | 28,672 | ~3.9 s | ~9.6 s |

The old estimates said ~30 ms at P=128 and ~300 ms at P=1024; the real numbers are ~350 ms and ~3.9 s empty. They were derived from a host-call count that undercounted entities the same way §1's "~500" comment did, and they are deleted rather than corrected in place, because their being cheap was the reason 256 and 512 read as a candidate range.

**They are not a candidate range any more.** At P=64 a player laying one belt at the edge of a full balancer already costs a **~25-tick single-tick freeze** — and that is today's shipped cap, not a projection. P=128 doubles it and P=256 is several seconds. So the shape of the change, when wanted, is unchanged — size-class slots + heap buffers + a raised `MaxPorts`, refusal staying loud past it — but the bound it is raised **to** is now a measured argument rather than an open question: **128 is the only defensible next step on a full-recompile architecture**, and anything past it needs incremental compilation across ticks first, which is the trade that was declined. Anyone who wants more should re-measure with `--hitch` rather than re-argue.

**And the cap being loud is worth more than it looked.** §4's refusal is what stands between a player and that freeze, and the `mega` save exercises it at bench scale: one 65-input cluster, exactly one `[BBB] alert:`, no `[BBB] error:`, and every other network in the save untouched and delivering.

## 4. What hitting the cap does — DONE 2026-08-04

**What it used to do**, verified against `compile.go` before the change: the fingerprint moves → `teardownForRebuild(root)` runs → **then** `plan.Build` returns `!fits`. So the 65th belt against a working 64-port balancer (a) demolished the running network, (b) built nothing, (c) spilled the entire contents beside the cluster, and (d) said so only in the log file. The refusal itself was correct; everything around it was hostile to the player who triggered it.

**Measured, on the `edge` suite's new `lim` leg** — a column of 32 parts carrying 64 input belts and one output, which is P = 64 = `MaxPorts` exactly, with a 65th input belt laid on it while it runs. Same rig, same schedule, the only difference being where the check sits:

| | check AFTER the teardown | **check BEFORE it** |
|---|--:|--:|
| the standing network | **torn down, 1,876 items drained** | untouched |
| rebuilt | nothing | nothing — and nothing needed to be |
| items on the ground | **1,690** (`ground` 336 → 2,026) | **0** |
| delivery over 246 ticks, before → after the edit | 184 → **10** | 184 → **185** |
| the audit afterwards | `nets=9 drift=0 unbuilt=1` | `nets=10 drift=1 unbuilt=0` |
| the log | `[BBB] error:` **three times** for one belt | one `[BBB] alert:` |
| `test/run.sh` | **killed the run** on the first error line | green |

The four items, all in `guest/go/limit.go` unless said otherwise:

1. **Refuse before teardown** — `compile.go`. `overLimitShape(edges)` mirrors `plan.Build`'s own two tests (a cluster with no inputs or no outputs is a legitimate half-built state at any size) and runs the moment `classifyEdges` returns. The old network is left standing and running; the over-limit belt stays unconnected, the same inert degradation as the documented failure envelope. `plan.Build`'s `!fits` branch is kept as an unreachable backstop and is still an `error:`, so a crack between the two mirrors fails a run.
2. **Tell the player.** `noteBuiltByPlayer` records `(surface, tile, force, player_index, isPart)` for every addition a PLAYER makes on or beside a cluster — scalars only, no host call, and nothing at all when `player_index` is zero, which is a robot, a script build and every event in every headless suite. At refusal time the player is resolved fresh with `game.get_player` and gets `create_local_flying_text` at the refused piece (the box centre until the 2026-08-05 interactive check: on a 32-part column that spawned the text seventeen tiles off the player's screen — sound heard, text unseen) plus `utility/cannot_build`; a robot or script build gets `force.print` with a different locale key, because "you got it back" would be a lie.
3. **Hand the piece back.** `revertOverLimit` re-finds each recorded entity at its tile, cross-checks it against the registry (a part stands ON a registered tile; a belt at an edge stands one tile outside the box by construction) and `mine_entity`s it into the player's inventory, `force = nil` so a full inventory declines rather than the item evaporating. **It runs from `flush()` after `endCarry()` and may not run any earlier**: `mine_entity` dispatches `on_player_mined_entity` synchronously, which re-enters the registry, frees nodes and refills the very queues the drain is iterating — and a claim recorded before `endCarry` would be settled against pools the miner had nothing to do with. The resulting mine takes the fingerprint back to what the `netInfo` already holds, so the next flush is a SKIP: nothing is ever rebuilt.
4. **The feedback gate.** A refused cluster is re-queued by every audit and by every event within two tiles, and its fingerprint can never match — so `overLimit` (root → refused fingerprint, point-queried only, nil until the first refusal) fires the message ONCE per distinct edge state. Without it the unfixed run's three `error:` lines for one belt would be three flying texts and three cannot-build sounds, and three more on every audit after that. The `edge` suite asserts exactly one.

**What the suite pins and what it cannot.** `assert-edge.py`'s LIM section asserts the refusal line and its numbers, one refusal per edit, zero ground items, an unmoved conserved total, the audit's `drift=1 unbuilt=0`, and delivery holding across the edit as a ratio between two equal-length windows.

**The robot arm of the feedback is verified; the player arm is not.** Every build in every suite is a script build, so `player_index` is zero and the fork always takes `force.print` — which means the LocalisedString really does cross the boundary and the `LuaForce` really is resolvable from a force INDEX (the registry keeps no handle; it comes off a part standing on the cluster). Neither is readable back from Lua — `force.print` goes to chat and `--benchmark` does not log it — so the guest logs the fact and the suite asserts that line, once, with no error. The flying text, the sound and the hand-back need a player: `game.get_player` resolves to nothing in a headless `--create`, so what the suite asserts there is the NEGATIVE — zero pieces handed back over the whole run — exactly as it does for the miner's pocket, and that trigger joins the pocket's on CLAUDE.md's interactive checklist.

**The one shape §4 did not cover was the MERGE, and §5 covers it.**

## 5. The MERGE past the cap — DONE 2026-08-05

**§4 shipped with one shape written down as uncovered, and this is that shape.** A part BRIDGING two clusters into an over-limit one is queued by `AddPart` as two DEAD roots and one live one, so `flushDead` had already brought both predecessors down by the time `flushLive` reached the merged cluster and §4's check had nothing left to protect. Items were conserved and the revert still handed the part back, so both balancers returned on the next tick — empty, with everything they had been holding in a heap on the floor between them.

**§4's own wording is what made it look expensive**: *"covering it means classifying the merged cluster's edges before `flushDead` runs, which is a second classification pass on every flush, on the guest's hot path."* True of the obvious implementation and not of the one that shipped. Two things make the pass free:

- **A dead root that is NO LONGER A ROOT has been absorbed**, and what absorbed it is the cluster the flush is about to build. That is an exact test for a merge and it costs one `find` per queued root. A shrink, a split, an audit repair, a clone reconcile and a surface deletion all queue roots that are still roots, and none of them is looked at twice.
- **A cluster of C parts has at most 4C exterior sides and therefore at most 4C edges**, so `4*csize[r] <= MaxPorts` is a *proof* that no classification could find enough of them. Sixteen parts is the largest cluster that can be proved safe that way, and every balancer any suite in this repo merges is smaller — the `mar` suite's merge leg is five parts. So the only merges that pay for a classification are the ones between balancers big enough to be worth asking about.

Being wrong about the bound costs exactly the old behaviour: the merge falls through to the refusal in `compile()`, where it always was. Nothing about that path changed.

`guest/go/limit.go`, "The merge", is the long form; `spareOverLimitMerges` runs from the top of `flushDead` and from nowhere else.

### Measured, both sides of the change

The `edge` suite's new `brdg` rig: two columns of sixteen parts with a belt on both sides of every one — thirty-two inputs and one output apiece, so each half is a real P=32 network of 434 entities — separated by ONE TILE that already carries two input belts of its own. A part in the gap makes one cluster of 66 inputs and 2 outputs, which would need P=128. Sixteen and sixteen is the cheapest shape that reaches it: a connected cluster of C parts has at most 2C+2 exterior sides, so sixty-five edges needs thirty-two parts however they are arranged, and splitting them evenly is what keeps each HALF at 32 ports instead of making one of them a second 64-port network like `lim`.

Same rig, same schedule, the only difference being the one call at the top of `flushDead`:

| | no pre-pass | **the pre-pass** |
|---|--:|--:|
| the two standing networks | **both torn down, 1,044 items each** | untouched |
| items on the ground | **+1,814** (`ground` 336 → 2,150) | **0** |
| delivery over 246 ticks, before → after the edit | 186 → **8** and 185 → **8** | 186 → **184** and 185 → **184** |
| the audit while the refusal stands | `clusters=11 nets=10 drift=0 unbuilt=1` | `clusters=11 nets=12 drift=1 unbuilt=0` |
| ... and the bridging part mined off again | both halves rebuilt from scratch | **0 teardowns, 0 builds** — a SKIP |
| visible interfaces of ours standing | 114 | **180** |
| `[BBB] error:` | 0 | 0 |
| the conserved total | unmoved | unmoved |

**The item total was conserved to the item in both arms**, which is exactly why this was invisible for a milestone. What moved is where the items were and whether the machines still existed.

Verbatim from the failing run, and note what the alert claims while it happens — true of the cluster being refused, which had no network of its own, and a lie about the two that did:

```
[BBB] merge 18+64->18 (17 parts)
[BBB] merge 18+80->18 (33 parts)
[BBB] torn down cluster 64, returned 1044 items
[BBB] torn down cluster 80, returned 1044 items
[BBB] alert: cluster 18 would need 128 ports for 66 inputs and 2 outputs, over
      the limit of 64; refused BEFORE the teardown, so the standing network is
      untouched
[BBB] spilled 1044 items beside cluster 64
[BBB] spilled 1044 items beside cluster 80
[BBB] audit clusters=11 parts=90 nets=10 drift=0 unbuilt=1
```

### What the registry believes while a refused merge stands

**One cluster, two networks, and at least one of them under a key that is not a root.** That is the only state in this guest where `nets` holds something `liveRootList` can never reach, so it is reported rather than left to be discovered: `strandedNets` tells `auditAll` how many networks a refused merge left behind, the audit counts them in `nets=`, and the merged cluster is reported as `drift=1 unbuilt=0` — an edge list past what the mod can build, and a guest that knows it. `drift=0 unbuilt=1` there is the signature of the defect.

Which key the survivor's own network is under is not fixed and must not be assumed: `newNode` reuses freed ids, so the merged root can be either predecessor's root OR the bridging part's own brand-new node with no network at all. The measurement above is the second case (`merge 18+64->18`), which is precisely why the audit asks limit.go instead of reading `nets[root]` and drawing a conclusion.

**It is stable, and that is asserted rather than argued**: the `edge` suite takes four audits while the refusal stands and requires the same report from all four, exactly one `alert:` for the whole edit, and no teardown, build or spill in between. The three ways out:

| what happens | what the pass does |
|---|---|
| the edge list shrinks back under the limit | queues every stranded network dead, INCLUDING the merged root's own — a cluster that tears itself down in `compile()` opens an `owned` pool and then declines every geometric one (carry.go, `takeCarry`), so the other predecessor's items would spill. Down in `flushDead` instead, both pools are unowned and the one network that goes up claims them both |
| the bridging part is mined — the revert, or a player | the cluster splits back into what it was, each component re-roots at its smallest node id (which is the root it already had), and each half's fingerprint is the one its netInfo never lost. **Both compiles are a SKIP; nothing is torn down and nothing is rebuilt** |
| the stranded node is FREED — the cluster dissolved in one event, a surface went | nothing can ever reclaim that network and its node id is on the free list, so `sweepStranded` brings it down on the next flush, before the id can be reused under it. A dissolve is a removal, so it spills, which is correct |

### What it costs

**Nothing measurable, and that is checked rather than asserted.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — and leg F *is* the merge leg, a balancer grown by a part and taken apart again a hundred times. Six parts is under the sixteen the arithmetic bound proves safe, so the pass never classifies anything and never allocates. `flushDead` gains one call that returns on two length tests whenever nothing merged and nothing is stranded.

### The merge shapes that are STILL not covered

Two, and they are the same one seen twice: **a clone reconcile (`reconcileArea`) and `game.merge_forces` both call `flushDead()` themselves, before the merging happens.** Their sequence is *bring every affected network down → reconcile the registry → rebuild*, so by the time the merged cluster exists its predecessors are already gone and there is nothing for the pass to spare. Items are conserved and both balancers come back empty, exactly as the bridging part used to do.

Covering them means the same decision a third and fourth time, in the middle of two reconcile paths whose whole design is a wholesale rewrite — and neither is a gesture a player makes. A clone big enough to bridge two 33-port balancers is a mod or the map editor; `merge_forces` is an administrator's keypress. Written down rather than done.
