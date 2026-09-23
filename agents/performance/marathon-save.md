# The marathon save — 300 hours, four players, and where the bill lands

**The heap diet asked what the guest allocates. This asks what it allocates PER EDIT, forever.** Under `-gc=leaking` a transient allocation is permanent: it is in the linear memory, in the save, in every multiplayer join, and in Lua's collector's walk for the rest of the session. So the question a marathon multiplayer game asks is not "does it leak" — everything does, by construction — but what the SLOPE is, whether any of it is superlinear, how much of it is ours, and what a 300-hour save therefore looks like.

Measured 2026-08-02 on Factorio 2.0.77 and re-recorded 2026-08-24 on **2.1.14** after the rigs were re-laid single-edge; base only, `-gc=leaking`, `--persist=packed`, `test/run.sh mar`: 680 net-zero world operations over 4,600 ticks, each followed by a `bbb-audit` marker so the guest prints its own `[BBB] heap post-audit sys=… alloc=…`. Under leaking `alloc` is TinyGo's bump allocator reporting every byte it has ever handed out, which IS the permanent heap.

## The slopes

**RE-RECORDED 2026-08-24 ON THE SINGLE-EDGE RIGS**, Factorio 2.1.14, `make GC=leaking test`. Every rig in this suite was re-laid one belt per part, which in three of the five legs that touch a balancer makes the CLUSTER bigger (two columns of parts instead of one) while leaving the NETWORK the same size, and in one makes the network smaller. The old figures are kept beside the new ones because the pairs are what say which term moved and why. Net of the audit (1,136 B, measured three times with the world untouched, **0.0% spread**, unchanged); each leg is shaped so that one term dominates it:

| one operation | 2.0.77, multi-edge | **2.1.14, single-edge** | what moved |
|---|--:|--:|---|
| a belt-connectable **mined or rotated** anywhere, no cluster within two tiles | 0 B | **0 B** | nothing; the guest rejects it from guest memory |
| a belt-connectable **built** anywhere, same | 32 B | **32 B** | nothing; it is the `name` string and nothing else |
| a belt laid **inside the two-tile gate** and picked up again | 352 B | **352 B** | nothing; the fingerprint still throws the re-classification away |
| one **teardown-and-rebuild of a 2→2** (11 entities) | 1,180 B | **1,209 B** | +29 B: the same network, over a four-part cluster instead of a two-part one |
| one **teardown-and-rebuild of a 4×4** (32 entities) | 3,736 B | **3,736 B** | **nothing at all** -- see below |
| a **whole 4-part balancer** in and out, run under load | 1,216 B | **1,280 B** | 16 entities each way instead of 18, over a 2x2 block instead of a 1x4 column |
| a **six-entity paste and its undo** | 736 B | **560 B** | the six entities now make a 1→1 (P=1, 5 entities) where they made a 2→2 (P=2, 11) |
| a full balancer **grown by a part, dissolved and rebuilt** | 1,712 B | **2,080 B** | five parts torn down and four put back, where it was three and two |
| a belt laid **ACROSS a face** and picked up again (leg H) | -- | **384 B** | new 2026-09-07, and see below |
| one **`bbb-audit`** over the save | 1,136 B | **1,136 B** | nothing |
| linear memory over the 680-operation run | 3.92 MiB | **3.92 MiB** | nothing |

**LEG H IS THE CURVE ARM'S GATE AND IT COST THE ARM NOTHING, MEASURED BOTH WAYS.** A perpendicular belt is the only thing that makes the classifier read tiles OTHER than the face itself (compile.go, `curvesFromCluster`), and no leg in this suite had one -- so that arm could have allocated per probe forever with every slope above green and blind. Leg H is leg B's shape one tile in: the belt is DECLINED, by its FAR tile rather than its rear, so both probes run and nothing is rebuilt. **384 B per iteration with the arm enabled and 384 B with it disabled**, byte-identical over 100 iterations in the same session, so the arm's own permanent cost is zero; the 32 B that separates leg H from leg B is `classifySide`'s existing `find_entities_filtered` returning a one-element result slice for a side that now has a belt on it, which leg B's tile never has. Linearity x1.00.

**THE 4×4 LEG IS THE CONTROL ON THE WHOLE RE-LAY AND IT CAME BACK IDENTICAL TO THE BYTE.** That is not luck: a 4x4 built the wide way -- inputs on the west column, outputs on the east, two interior columns carrying nothing -- was ALREADY single-edge and never had a tile with two belts on it, so leg G's rig is the one thing in this suite that did not move. 3,736 B before the port and 3,736 B after is the measurement that says the guest's compile path is untouched and every other row of this table is geometry.

**And every term the 300-hour projection uses is in that unmoved set.** The edit-rate model below multiplies exactly three of these -- the far-belt build (32 B), the belt inside the gate (176 B per primitive) and the 4×4 recompile (3,736 B) -- and all three are byte-identical across the port. **The projection stands exactly as recorded**, and it is worth saying out loud that this was checked rather than assumed.

**The 2.0.77 column itself carries the 2026-08-02 item-placement policy** — a recompile reinserts its drained items instead of spilling them ("A recompile is not a removal"), which is more host calls on the two recompile legs and fewer allocations on every leg that drains. Its own before/after pair (681 → 1,180 for the 2→2, 1,517 → 3,736 for the 4×4, 1,921 → 1,712 for the grown balancer) is in that section. **Under the shipped `--gc=collected` none of it persists**: on 2.1.14 the same 680 operations end on **0.46 MiB of linear memory with a 10,192 B live set**, 9 collections in 6 paced steps and **0 forward-progress deadlines** (2026-08-02 on 2.0.77: 0.46 MiB and 8,736 B; 2026-08-03: 0.71 MiB and 16,448 B). All of them are orders inside the suite's 4 MiB and 256 KiB ceilings.

**Nothing is superlinear, and that is the finding rather than the arithmetic.** Every leg's second hundred iterations cost what its first hundred did, to within 7% — the suite fails on 1.35×. Measured over the re-laid rigs: A ×1.00, B ×1.00, C ×1.03, D ×1.00, E ×1.00, G ×1.07, F ×1.00. That is the classic 300-hour killer ruled out directly rather than argued: it means a place/remove cycle that nets zero entities really does cost a constant, and three hundred hours is multiplication rather than a curve.

**Why it is flat, checked shape by shape.** Every monotonic-growth candidate in the guest was audited and every one of them is bounded by a HIGH-WATER MARK rather than by operations ever performed:

| shape | verdict |
|---|---|
| registry node ids | **reused.** `freeNode` pushes the slot onto `freeID` and `newNode` pops it, LIFO. Ids do not grow with parts ever placed, and the seven parallel slices (`parent`, `csize`, `ppos`, `pforce`, `alive`, `mark`, `pvar`) grow only to the most parts the map has ever held at once |
| the `pvar` byte | in those slices, so the same answer. A reused slot is reset to 0, which is "we do not know what is drawn there" |
| `index`, `nets`, `rfwClaim` | Go maps, all three point-queried only. `delete` is called on every removal path; the bucket array is high-water, and under leaking each doubling's predecessor stays — so ~2× high-water, not ~1× operations |
| slot grid | **reused.** `releaseSlot` pushes onto `freeSlots`, `allocSlot` prefers it. `nextSlot` rises only to the most networks standing at once, and `sweepOrphanSlots` rebuilds the allocator around what is actually claimed |
| the defer queue | `deadRoots`/`liveRoots` are truncated to `[:0]` by every flush; high-water is the clusters touched in one tick |
| edge lists, plans, tile buffers, the `create_entity` table | package-level and reused; `netInfo` stores a 64-bit FINGERPRINT rather than the edge list, exactly so that a stored slice per cluster is not a leak that grows with play |
| a map only ever written | there is none |

The linearity check is what proves this rather than the reading: a registry whose ids grew monotonically would show up as a slope that climbs as the slices double, and it does not.

## Whose bytes they are

**Every byte in the compile terms is a generated binding's return value, and none of it is BBB's own.** A `find_entities_filtered` return is `out := make([]Object, n)`; an entity return is `return &v`; a `type` or `name` read is `string(b)` out of `getStr`, which must copy because the marshalling arena underneath is released when the call returns. A 2→2 teardown-and-rebuild makes about thirty such calls, which is the 681 B. The arena (FKLUA-GAPS.md item 10) fixed the ABI's own side; this is the caller's, and upstream's own note says so — "the 48 bytes that remain are the caller's, not the ABI's".

**The one term that WAS ours is taken.** `onEvent` used to read `name`, `position` and `surface_index` in that order for every belt-connectable event on the map, and the engine's filter admits every belt there is — so every belt anyone lays or picks up anywhere paid 32 B of permanent heap for a string the guest usually threw away. Position and surface index are scalars decoded out of the return block and cost nothing, so they go first now, and a **disappearance or a rotation** that is not on a registered tile, not inside the two-tile gate and not on the hidden surface returns without buying the name at all. All four questions are answered from guest memory (`nearCluster`, 25 point queries, no host call).

An **appearance** still buys it unconditionally and always will: a part placed alone in the middle of nowhere has no neighbours to be recognised by, so the name is the only thing that can identify it. Measured, same suite, before → after:

| | before | after |
|---|--:|--:|
| a belt laid and picked up 18 tiles from anything | 64 B | **32 B** |
| a whole 4-part balancer placed and removed (18 entities each way) | 1,664 B | **1,216 B** |
| a six-entity paste and its undo | 864 B | **736 B** |

The 448 B the second row saved is exactly the 14 belts' mine events at 32 B each, which is the arithmetic working.

## The projection

**The edit-rate model, stated so it can be argued with.** Per player-hour, for a player who is actively building rather than idling:

| | events/player-hour | why |
|---|--:|---|
| belt-connectables **built** anywhere | 250 | a busy hour lays a few hundred belt pieces; every one enters the guest |
| belt-connectables **mined** anywhere | 150 | free now |
| belts laid **inside a balancer's gate**, no edge moved | 10 | balancers are a small fraction of a base's tiles |
| edits that **move an edge** on a 4×4 | 6 | |
| whole balancers built or removed | 1 | ~3.6 KB each at 4×4 size |

= 8,000 + 0 + 1,760 + 9,102 + 3,600 ≈ **21.9 KiB per player-hour**.

**Re-run 2026-08-02 with the item-placement policy's 4×4 term (3,736 B): ~34.9 KiB per player-hour, 1.60×.** The table below is the 21.9 KiB model as measured; multiply the heap rows by 1.6 for today's guest, which moves the busy column to ~41 MiB of permanent heap and one rung up (32 → 64 MiB) of linear memory. Both are `-gc=leaking` numbers and the shipped build is `collected`, where the live set did not move at all.

**The single-edge port did not move any of this, and that was checked rather than assumed.** All three multiplied terms -- 32 B for a far belt built, 176 B for a belt inside the gate, 3,736 B for a 4×4 recompile -- came back byte-identical on 2.1.14 over the re-laid rigs (see "The slopes" above). The projection stands as recorded.

| after 300 hours | quiet (1 player, ¼ rate) | **busy (4 players)** | extreme (8 players, 2× rate) |
|---|--:|--:|--:|
| permanent guest heap | 1.6 MiB | **25.7 MiB** | 103 MiB |
| linear memory (TinyGo's `growHeap` DOUBLES) | 2 MiB | **32 MiB** | 128 MiB |
| save size added (`packed`, 113 KiB/MiB) | 0.2 MB | **3.5 MB** | 14 MB |
| host RAM, per client (5.00 MiB/MiB) | 10 MiB | **160 MiB** | 640 MiB |
| join / load time added (~26 ms/MiB, flat) | 0.05 s | **0.8 s** | 3.3 s |
| Lua GC total per cycle (0.202 ms/MiB) | 0.3 ms | **5.2 ms** | 21 ms |
| Lua GC **worst tick** (sharded) | ~0.5 ms | **~0.5 ms** | ~0.5 ms |
| worst **`memory.grow`** tick, the last doubling | ~0 ms | **~420 ms** | **~1.8 s** |

Rates from `../FkLua/agents/guests.md`, "the guest heap budget". The `packed` save number is the same 0.44 B/word the heap slope implies: **the heap slope IS the save-size slope**, because packed pages track the heap's high-water mark.

**And the row that matters is the last one, which is not the row anybody expected.** Sharding made Lua's collector's worst tick FLAT at ~0.5 ms whatever the heap size — the 0.2 ms/MiB penalty this repo spent two milestones attributing is a per-cycle TOTAL now and not a pause. The save is 3.5 MB. The join is under a second. All three are documented bounds a mod can ship with.

What is not is `memory.grow`. TinyGo's `growHeap` grows by exactly the current size, so a leaking guest is always on the ladder, and `mem_grow` writes a zero into every new word at ~107 ns a word in Factorio's Lua. The step from 16 MiB to 32 writes 4,194,304 fresh words — **~450 ms in one tick**, less the ~25 ms the runtime's paced pre-build can have ready. Upstream measured the same shape directly: **491 ms worst grow tick for a leaking guest at a 16 MiB target and 974 ms at 40 MiB**, against a collected guest's flat 23–25 ms. The pre-build is capped at 1 MiB deliberately — a lookahead of "one grow" would be unbounded for a doubling guest — so it removes a fixed ~25 ms and 2.5% of a 32 MiB doubling.

**Nothing downstream can bound that.** It is a fact about the growth LAW, not about collecting; guests.md says so in as many words. A busy 300-hour multiplayer server under `-gc=leaking` pays about eight of these stalls over its life, the last two being roughly 225 ms and 450 ms — single-tick freezes every client feels at once.

> **MEASURED SINCE, AND WORSE THAN THIS.** The rung above was arithmetic; driven up the ladder for real on 2026-08-02 it came out **48.7 ms at 2→4 MiB, 120.3 at 4→8, 226.1 at 8→16 and 782.4 ms at 16→32** — the two middle rungs on the 107 ns/word model almost exactly, and the last one 1.74× over it, because a 16 MiB grow also pays eight new shards' last array-part reallocation (`../FkLua/agents/sharding.md` §15). **This is the row that flipped the `-gc` decision**; the mod ships `--gc=collected` since that pass, and the same 3,400 operations end on 0.52 MiB of linear memory instead of 31.9 with a worst tick of 71 ms instead of 782. See "The third decision" above. Everything else in this section is a statement about the OTHER arm, which stays buildable and stays measured.

## What this does NOT say

- **It is not a leak in the ordinary sense.** Nothing is retained: the collected arm of the same run reports a **live set of 8,736 B** at the end of 680 operations (16,448 B on the 2026-08-03 re-measurement). Every byte above that is reclaimable and simply is not reclaimed.
- **A single-player game is unaffected.** The quiet column costs 0.2 MB of save and no stall at all, which is what the mod has been measured at all along.
- **The compile slope is not reachable from here.** It is generated binding return values, [`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 17, and moving it means changing how the classifier reads the world — a correctness surface, not a place to economise.
