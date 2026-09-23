# A recompile is not a removal — where a teardown's items go

**The defect interactive play found that seven headless suites did not.** Every edge edit on an operating balancer — adding one output belt — emptied the hidden network onto the ground beside it. Every suite passed while it did, because every suite asked the wrong question: *were the items conserved?* They always were. The question nobody had asked is *where did they end up*, and the answer was "on the floor", on every recompile, forever.

**The policy, in one line: a recompile reinserts, a removal spills — and since 2026-08-02 a removal a PLAYER caused pockets first.** The implementation is `guest/go/carry.go` and its file header is the long form. The short form:

| what happened | what the drained items do |
|---|---|
| an **edge edit** on a standing cluster (a belt added, mined, rotated, retiered) | go back inside the network the same flush rebuilds |
| a **merge** — two clusters bridged | the survivor takes both halves' pools |
| a **split** — a cluster broken in two | both successors take even shares |
| a cluster **shrunk** (a part mined, parts remain) | the survivor takes them |
| a **clone reconcile**, a **force merge**, the **hidden surface** coming back | all recompiles; all reinsert |
| anything **MINED BY A PLAYER** that makes the machine smaller — a part, whether the cluster dissolves or merely shrinks, or a **belt at the cluster's edge**, which takes a port off it | whatever the successor network takes goes back inside it as always; **everything left over goes into that player's inventory**, and only the remainder of *that* to the ground — vanilla's rule for a mined machine. This row was narrower than the sentence above it twice: restricted to the dissolve it emptied a balancer onto the floor shrink by shrink, and restricted to PARTS it emptied one onto the floor when a player mined the output belt back off. Both were field reports on 2026-08-02. See "The miner's pocket" |
| a cluster **DISSOLVED** any other way, a surface **deleted**, a network **forgotten** | spill beside the cluster, exactly as before — nobody to credit |
| what a **materially smaller** network cannot hold | spills; this is the only way a recompile can still put anything on the floor |

## The four decisions, and why each is the one it is

1. **Which network inherits a pool.** The pool remembers the surface, the force and the visible bounding box of the cluster it came out of, and a cluster built in the same flush claims it when all three agree — boxes *overlapping* is the test. That single predicate covers all three shapes above: a plain recompile matches itself, a merged cluster's box contains both old ones, and a split's successors are both inside the old one. The **force** check is not decoration (two forces' parts touching are two balancers whose boxes are adjacent by construction) and it is the one thing `on_forces_merged` has to patch: it tears the networks down *before* it remaps the registry, so `remapCarryForce` follows the merge into the pools already drained. Without it the surviving cluster failed to recognise the absorbed half's pool and 52 items went on the floor — measured, in exactly that shape. **It follows the merge into the CLAIMS too since 2026-08-02**, which was the same omission one level down: a claim carries a force since it became a one-tile `carry.Region`, a player mining a source-force part in the merge tick wrote one down under the force about to be destroyed, and the survivor's remapped pool then failed to match it — the pocket becoming the floor. Both are `carry.Region.FollowMerge` now, and the claim store moved into the pure package with the predicate so `make check` covers it. See "The miner's pocket".

2. **How a pool divides between several claimants** (the split): even shares per item kind, the earlier claimant taking the odd one — `ceil(remaining / claimants-left)`, which is exact for one and splits 24 as 12/12. The counting has to happen **before the first build**, which is why `claimCarry` is a separate pass over the flush's queue: the first successor cannot know to take half unless something has already found the second. It is a flood fill per queued cluster and **not one host call**, and it runs at all only when a teardown left something behind.

3. **Where in the new network they go.** In **plan order** — `plan.Build` emits input side to output side — filling each transport line before the next, and **interior lines (splitters, lane splitters, straight belts) before any linked belt**. The second half is the one that matters: every stage still in front of a reinserted item rebalances it, whereas an item put on an edge line reaches a player's belt without passing through the butterfly at all. Measured on the 4→5 network a recompile built under load: **300 300 300 300 300 over the next 500 ticks, 0.00% spread.** `entBuf` is parallel to `ops`, so the order needs no second look at the world and no sort.

4. **`insert_at`, not `insert_at_back`, and that is the difference between 32 items and all of them.** The back of a line is ONE position, so `insert_at_back` succeeds once per line per tick and then refuses until the belt has moved. The first cut of this file used it and recovered exactly 32 items into a network with 32 transport lines, dropping the other 40 of a 72-item drain straight through to the spill. This is the same property `agents/design.md` records as an incumbent sin — *"script `insert_at_back` cannot produce compressed belts"* — met from the other side. `insert_at(position)` places at a named point, `line_length` says how far that goes, and walking a line at the belt's own **0.25-tile item pitch** fills it the way the drain found it.

**The transaction, and why it is a counter.** Pools live for one flush: `flush()` opens one around teardown-then-build and settles what nobody claimed when it closes. The three reconcile paths that split their own flush in half — `reconcileArea`, `hiddenSurfaceGone`, `onForcesMerged` — open one around the whole sequence, and the nesting counter absorbs the inner `flush()`. **Every path that opens one must close it**: the items are in guest memory until it does, and a pool that outlived its dispatch would be items in no transport line, no inventory and no ground stack. `dropSurface` and `hiddenSurfaceGoing` deliberately do *not* open one — there is nothing left to rebuild onto — so they spill immediately, which is what they always did.

## What it costs, measured

**The recompile hitch went up and this is the trade.** Interleaved `old / new / old / new / old / new` in one session, `test/run.sh m2`, medians of three, each minus that run's own `idle tick pair, nothing pending` control (0.36 old / 0.48 new), shipped `--persist=packed --gc=collected`:

| forced teardown-and-rebuild of a SATURATED rig | before | **after** | |
|---|--:|--:|---|
| 4×4, full recompile | 7.69 ms | **11.55 ms** | 1.50× |
| 4×4, one input removed | 9.40 ms | **11.76 ms** | 1.25× |
| 8×8, full recompile | 14.38 ms | **25.72 ms** | 1.79× |
| 8×8, one input removed | 18.10 ms | **26.92 ms** | 1.49× |

**It is boundary-bound like everything else here and it is proportional to the items actually in flight.** A 4×4 recompile hands back ~80 items and an 8×8 ~230, and the reinsertion is about 1.4 host calls per item recovered plus two per transport line touched, at the same ~12.6 µs per call the rest of the compile pays. The line walk is **lazy** for exactly this reason: it asks an entity how many lines it has, and a line how long it is, only as far as the items reach, so a recompile that recovered twenty items touches two entities and pays for two. An empty or lightly loaded network pays almost nothing, and **steady state is still zero** — there is no `on_tick` handler and there must never be one.

An 8×8 saturated recompile is now over one engine tick. That is stated rather than hidden, and it is the right side to be wrong on: the alternative is several hundred items on the floor every time a player touches a belt near a big balancer.

**The heap, and the one thing this pass gave back.** Under `-gc=leaking` a transient is permanent, so the `mar` suite's per-operation slopes move. Measured 2026-08-02, same suite, same method as "The marathon save":

| one operation | before | after | |
|---|--:|--:|---|
| teardown-and-rebuild of a **2→2** | 681 B | **1,180 B** | 1.73× |
| teardown-and-rebuild of a **4×4** | 1,517 B | **3,736 B** | 2.46× |
| a balancer grown by a part, dissolved and rebuilt | 1,921 B | **1,712 B** | **0.89×** |
| a whole 4-part balancer in and out | 1,216 B | 1,216 B | — |
| a belt inside the gate, laid and picked up | 352 B | 352 B | — |
| a belt 18 tiles from anything | 32 B | 32 B | — |
| linear memory over the 680-operation run | 1.92 MiB | **3.92 MiB** | one more rung |

Every leg is still **linear** (second half against first, ×1.00–×1.07; the suite fails at ×1.35), which is the property that makes a 300-hour projection multiplication rather than a curve. Re-running the edit-rate model with the new 4×4 term gives **~34.9 KiB per player-hour against 21.9**, so the busy four-player 300-hour figure moves 25.7 MiB → ~41 MiB and the linear-memory rung 32 → 64 MiB. **In the shipped `--gc=collected` build none of it persists**: the same 680 operations end on 0.77 MiB of linear memory against 0.46 before, and the **live set — what a save actually carries — is 8,960 B against 8,736**. This is the defect class "The third decision" flipped the mode for, arriving on schedule.

**And one term went the other way.** `drain` reads a line's contents with `GetContentsInto` on a package-level buffer now instead of `GetContents`, which was `make([]ItemWithQualityCount, n)` per non-empty line. That mattered more once a recompile started reinserting — a network handed its items back is *full* the next time it comes down, so far more lines take that branch — and it is worth ~2.2 KB per 4×4 recompile: the 4×4 leg was **6,517 B** before the buffer and is 3,736 B after, and the whole-balancer leg came out *below* its pre-policy figure. It is the only allocation on that path BBB owns; the rest is generated-binding return values ("What is left, quantified").

## What is still lost, honestly

Fractional item positions, and whatever a splitter holds outside its transport lines. **The stack sizes are no longer on this list** — see the section below, which is the pass that took them off it.

One known imprecision, recorded rather than fixed: a cluster that claims a pool geometrically but is then **not built** (it has no adjacent belts at all, or its compile fails) leaves its share to spill at the end of the flush. It needs two same-force clusters with overlapping boxes recompiling in the same tick, one of them beltless; conservation is unaffected and the fallback is the correct one. **Since the beneficiary pass that share is offered to the miner before the floor**, if a player is what caused the dissolve, so the fallback got slightly better without the imprecision going away.

## More than thirty-two kinds — the bound that was an item sink

**A drained pool holds `maxCarryKinds` = 32 (name, quality, stack) groups, and the thirty-third was logged and DROPPED.** `drain()` had already read that group off a transport line and `sweep()` destroys the entity a few statements later, so those items ceased to exist — the one thing this mod is not allowed to do, in the one file whose header says so twice. Found by review on 2026-08-04, not by play, and then reproduced in a headless run before a line was changed.

**What a player would see, and why it is not exotic.** Thirty-three distinct groups through one balancer is a **sushi belt or a mall bus** — a mixed line feeding a balancer is one of the things people build balancers for. Under Space Age it is nearer eight item kinds than thirty-three, because a group is a (name, quality, stack size) and belt stacking multiplies the first by the last. The symptom is silent: the balancer keeps working, the counts nobody is taking are smaller than they were, and the only trace is a `[BBB] error:` line in a log nobody reads.

**Why eight suites were green while it happened**, and it is the same shape as the two field reports above: **all of them run iron plates.** One kind, one quality, no stacking, so the pool never held more than a handful of groups and the branch was unreachable from every rig in the repo. `mix` is the suite that can reach it, and it is red-proven against the guest that shipped: **16 kinds and 72 items destroyed** on one recompile of a saturated 4×4 carrying 48 kinds.

**The fix is that the overflow spills, and WHEN it spills is the part that was measured twice.** `tally` appends what it cannot carry to a small buffer and `closePool` puts it on the ground at the centre of the cluster's box — the same place, the same call and the same arithmetic as the `spillPool` every removal path has always used. The first cut spilled at the moment of the decision, inside `tally`, which is appealing because there is then nothing to remember; it lost **one stone-brick of 4,336**, because `teardownNet` sweeps the hidden half FIRST and the visible cluster box SECOND, `spill_item_stack` allows belts, and the visible sweep re-drained what had just been put in the middle of it into a pool already too full to take it. The control that proved it: raise the bound to 64 so that nothing overflows and the same rig conserves exactly. Buffering to the end of the teardown, after both sweeps, gives **4,336 in and 4,336 out**.

**What it gives up is placement, not conservation**, which is the doctrine `compile.go`'s `detailedTally` already states in as many words: an overflow group is not carried into the successor network and is not offered to a miner's pocket, because both of those are what the POOL is for and a buffer general enough to reach either would be the pool with no bound at all. Past thirty-two groups this guest stops promising WHERE a teardown's items land and goes on promising THAT they land. The bound itself stays: it bounds package-level memory that is never given back.

**The line is an `alert:`, and the level is load-bearing rather than a matter of taste.** `test/run.sh` fails any run in which a `[BBB] error:` line appears at all, because an error from this guest means one thing — a compile did not produce a network. An overflow is not that, and at `error:` the `mix` suite could not have asserted it: the runner would have killed the run before a single number was read. `alert:` is this guest's existing level for "something outside the ordinary happened and the mod coped", it is never switched off by `QUIET=1`, and the suite **requires** it in the overflow window.

**What it costs.** Nothing on any path base single-kind play reaches: the branch is past thirty-two groups, `carryOverflow` is empty on every other tick, and the common path gains one slice truncation in `openPool` and one `len()` test in `closePool`. Measured rather than asserted — the `mar` suite's seven per-operation heap slopes came back **identical to the byte** in the `-gc=leaking` arm (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B, 3.92 MiB of linear memory), and a pre-fix/post-fix run of the other six suites differs in **no asserted number at all** — only in profiler milliseconds and in which function `fklua mod`'s NaN report attributes a hoisted `f64` to. Package built 2026-08-04, shipped config:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 298,874 B | **300,422 B** | +0.52% |
| `fk_module.lua` | 2,282,383 B | **2,316,009 B** | +1.47% |
| members bound into the mod | 38 | **38** | of 4,257 |
