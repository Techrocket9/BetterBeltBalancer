# The merge that would be over the limit — the other half of the same refusal

**"The sixty-fifth belt" moved the check in front of the teardown `compile()` does to ITSELF, and shipped with a note saying that a MERGE past the limit still demolished both balancers.** That note was accurate for a day. Closed 2026-08-05; `guest/go/limit.go`'s "The merge" and [`agents/maxports.md`](../maxports.md) §5 are the long form.

**Whose teardown it is, is the whole difference.** An edge edit's teardown belongs to `compile()`, which is why one statement moving twenty lines up fixed it. A merge's teardowns belong to `AddPart`, which marks BOTH predecessors' roots dead the moment the bridging part arrives — so `flushDead` had demolished two working balancers before `flushLive` so much as looked at the cluster they became. The items were conserved (the carry transaction settled them, unclaimed, onto the ground) and the revert still handed the part back, so both balancers returned on the following tick **empty**, with everything they had been holding in a heap on the floor between them. The alert said *"refused BEFORE the teardown, so the standing network is untouched"* while it happened: true of the cluster being refused, which had no network of its own, and a lie about the two that did. That clause has three arms since 0.3.3 and is `refusalFound`'s ("The alert that said the machine was fine"); against the guest above it selects the second on the two open pools, and against the spared merge it selects the third, which says only that the merged cluster had nothing to lose.

**`spareOverLimitMerges` runs at the top of `flushDead`** — the one choke point every teardown in this guest goes through — and takes the merge's teardowns back off the queue. `compile()` then reaches the merged cluster and refuses it through the path that was already there.

## Why it is free, which is what the old note got wrong

The note said covering it *"means classifying the merged cluster's edges before `flushDead` runs, which is a second classification pass on every flush, on the guest's hot path."* True of the obvious implementation. Two things make the pass cost nothing:

- **A dead root that is NO LONGER A ROOT has been absorbed**, and what absorbed it is the cluster the flush is about to build. Exact for a merge, one `find` per queued root, and it rejects every other queue filler for free: a shrink, a split, an audit repair, a clone reconcile and a surface deletion all queue roots that are still roots. Only a candidate that had something absorbed into it may have teardowns taken away, which is also what keeps the pass out of `dropSurface`'s way.
- **A cluster of C parts has at most 4C exterior sides and therefore at most 4C edges**, so a part count is a PROOF about how many ports the merged cluster can possibly have. The threshold is `4*csize[r] <= MaxPorts/2` since the priority pass and was `MaxPorts` before it: a priority network is two butterflies wide and stops fitting its slot at P = 64, which is thirty-three belts on a side, which nine parts can carry. **Eight parts is the largest cluster provable safe now, where it used to be sixteen** — and every balancer any suite in this repo merges is still smaller, the `mar` suite's merge leg being five parts. A merge of nine or more pays one classification it was going to pay a moment later in `compile()` anyway.

Being wrong about the bound costs exactly the old behaviour: the merge falls through to the refusal in `compile()`, where it always was.

## What the registry believes while a refused merge stands

**One cluster, two networks, and at least one of them keyed by a node that is no longer a root.** It is the only state in this guest where `nets` holds something `liveRootList` can never reach, so it is REPORTED rather than left to be discovered: `strandedNets` tells `auditAll` how many networks a refused merge left behind, the audit counts them in `nets=`, and the merged cluster comes out `drift=1 unbuilt=0` — an edge list past what the mod can build, and a guest that knows it. **`drift=0 unbuilt=1` there is the signature of the defect.**

Which key the survivor's network is under is not fixed and must not be assumed: `newNode` reuses freed ids, so the merged root can be either predecessor's root OR the bridging part's own brand-new node with no network at all. The measured run is the second case, which is exactly why the audit asks limit.go instead of reading `nets[root]` and drawing a conclusion.

**The state is stable and the suite asserts it**: four audits while the refusal stands, the same report from all four, exactly one `alert:` for the whole edit, and no teardown, build or spill in between. Three ways out:

| what happens | what the pass does |
|---|---|
| the edge list shrinks back under the limit | every stranded network is queued dead, INCLUDING the merged root's own — a cluster that tears itself down in `compile()` opens an `owned` pool and then declines every geometric one (carry.go, `takeCarry`), so the other predecessor's items would spill. Down in `flushDead` instead, both pools are unowned and the one network that goes up claims them both |
| the bridging part is mined — the revert, or a player | the cluster splits back into what it was, each component re-roots at its smallest node id (which is the root it already had), and each half's fingerprint is the one its `netInfo` never lost. **Both compiles are a SKIP; nothing is torn down and nothing is rebuilt** |
| the stranded node is FREED — a dissolve in one event, a surface deleted | nothing can ever reclaim that network and its id is on the free list, so `sweepStranded` brings it down on the next flush before the id can be reused under it. A dissolve is a removal, so it spills, which is correct |

## Measured, both sides of the change

The `edge` suite's new `brdg` rig, and the only difference between the columns is the one call at the top of `flushDead`:

| | no pre-pass | **the pre-pass** |
|---|--:|--:|
| the two standing networks | **both torn down, 1,044 items each** | untouched |
| items on the ground | **+1,814** (`ground` 336 → 2,150) | **0** |
| delivery over 246 ticks, before → after the edit | 186 → **8** and 185 → **8** | 186 → **184** and 185 → **184** |
| the audit while the refusal stands | `clusters=11 nets=10 drift=0 unbuilt=1` | `clusters=11 nets=12 drift=1 unbuilt=0` |
| the bridging part mined off again | both halves rebuilt from scratch | **0 teardowns, 0 builds** — a SKIP |
| visible interfaces of ours standing | 114 | **180** |
| the conserved total | unmoved | unmoved |

**The item total was conserved to the item in both arms**, which is why this was invisible for a milestone: what moved is where the items were and whether the machines still existed.

## What it costs

**Nothing on any hot path, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — and leg F *is* the merge leg, a balancer grown by a part and taken apart again a hundred times. Six parts is under the sixteen the bound proves safe, so the pass never classifies and never allocates. Package built 2026-08-05, shipped config:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 310,628 B | **316,749 B** | +1.97% |
| `fk_module.lua` | 2,432,188 B | **2,516,161 B** | +3.45% |
| `dist/bbb.wasm` | 1,076,928 B | 1,098,032 B | |
| members bound into the mod | 42 | **42** | of 4,257 — none added |

## The merge shapes that are STILL not covered

Two, and they are the same one seen twice: **a clone reconcile (`reconcileArea`) and `game.merge_forces` both call `flushDead()` themselves, before the merging happens.** Their sequence is *bring every affected network down → reconcile the registry → rebuild*, so by the time the merged cluster exists its predecessors are already gone and there is nothing to spare. Items are conserved and both balancers come back empty, exactly as the bridging part used to do. Neither is a gesture a player makes — a clone big enough to bridge two 33-port balancers is a mod or the map editor, and `merge_forces` is an administrator's keypress — and covering them means the same decision in the middle of two paths whose whole design is a wholesale rewrite. Written down rather than done; `agents/maxports.md` §5 carries it too.
