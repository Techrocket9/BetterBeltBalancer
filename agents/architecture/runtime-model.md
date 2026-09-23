# The runtime model: what M1 to M4 left standing

The milestone narratives and the invariants they fixed, from CLAUDE.md's Status section. M5's is [`../features/adaptive-graphics.md`](../features/adaptive-graphics.md).

**M1 is done.** The mod skeleton, the visible `bbb-balancer-part` entity and the cluster registry in the Go guest, verified headlessly.

**M2 is done.** Clusters compile into hidden splitter networks and the exit criterion is met: saturated 4x4 and 8x8 rigs balance to 0.15% spread at 99.9% of one belt per output, and starvation, blocked outputs, asymmetry, recompile under load and cross-surface all hold (table above). What M3, M4 and M5 inherit:

- **The network is a butterfly over P = next_pow2(max(N, M)) lines**, log2(P) stages of P/2 splitters, rows permuted between stages so each stage's pairs land on adjacent rows, and a row that has to move does so through a linked-belt jumper pair. There are **no belts between the pieces** -- every element sits directly against the next -- which took a 4x4 from 50 entities to 32 and an 8x8 from 132 to 84. Spare output ports loop back into spare input ports where there are enough of them and dead-end otherwise.

| network | entities | of which visible |
|---|---|---|
| 1->1 | 5 | 2 |
| 2->2 | 11 | 4 |
| 4->4 | 32 | 8 |
| 8->8 | 84 | 16 |
| 3->5 | 72 | 8 |

- **Invariant: at most one linked belt per tile SIDE.** Two edges may share a tile -- S1 ran four on one at full rate -- but two same-direction inputs on a tile leave one of them silently dead. `TestOneLinkedBeltPerTileSide` pins it.
- **Invariant: every teardown for an event runs before any build.** A cluster that split leaves the old network's visible interfaces standing on tiles that now belong to a different cluster; a build that ran first would classify them as part of the world.
- **Retired invariant: "a mined entity is still valid during the event that reports it".** M2 and M3 compiled inside the event that reported a removal, so the classifier carried a one-position blind spot -- the belt on its way out -- armed and restored by every flush. **`fk.Defer()` retired the whole mechanism** and the reasoning is worth keeping: the engine destroys a mined entity when its own dispatch RETURNS, so a flush that happens on the next tick re-reads a world the belt is simply not in. There is nothing to ignore, and the case the window could never handle -- a mod raising `script_raised_destroy` and then not destroying -- now comes out right instead of wrong. See `compile.go`.
- **Recompile is full teardown and rebuild**, plus a fingerprint over the edge list so that a change which does not move an edge rebuilds nothing. Entity-diff minimisation is M3+.
- **Recompile cost is boundary-bound, not algorithm-bound.** A 4x4 rebuild is 4.4 ms against 0.13 ms for the same 32 `create_entity` calls made straight from Lua; an 8x8 is 9.6 ms. (Of an EMPTY network. One carrying items also pays to put them back — see "A recompile is not a removal", which is the same boundary at the same ~12.6 µs per call.) That is ~350 host calls at ~12.6 us each, and that 12.6 us is the tier-2 encode on the Lua side -- see the `--persist` section for what does and does not move it. The planner itself is microseconds and allocates nothing on a warm buffer.
- **Items are never deleted, and since 2026-08-02 they are not put on the floor either.** Teardown reads every transport line in the slot; a RECOMPILE puts the total back inside the network it rebuilds and only a REMOVAL spills it beside the cluster (`spill_item_stack`, item by item -- `drop_full_stack = true` was measured placing nothing at all). See "A recompile is not a removal" below. What is not recovered either way: fractional item positions, anything a splitter holds outside its transport lines, and how the items were STACKED.
- The hidden surface is created lazily by name, generates no chunks (S1: entities on ungenerated chunks run at full rate; re-verified here), and is carved into 32x72 slots keyed by cluster root id. `MaxPorts` is 64; beyond it a compile is refused loudly rather than overrunning its neighbour's slot -- and the refusal happens BEFORE the teardown, so the balancer that was already there keeps running ("The sixty-fifth belt").
- The hidden belt tier runs at **speed 0.25**, which M2 recorded as a ceiling and which is a FLOOR since 0.3.1: a belt above it runs fully compressed and delivers 480 x speed items/s, so there is no engine ceiling there at all and a modded belt faster than 0.25 was being throttled. It is enough by construction for a vanilla game -- P >= N means no hidden line ever carries more than one visible belt's rate, and nothing in base or Space Age reaches 0.25 -- and `deriveHiddenSpeed` raises it to the fastest belt installed. See "Cost, research and belt speed".

**M3 is done.** Every 2.0 lifecycle path that can change what the compiler compiled from is handled and kill-tested (table above), and the exit criterion is met: **the stale-reference crash class is impossible by construction, not by handling.** The rules that make it so, and which nothing here may break:

- **No entity reference outlives the event it arrived in.** Persistent state is a surface index, two integer tile coordinates, a force index and a slot number. Nothing dereferences a `LuaEntity` or a `LuaTransportLine` from a previous tick, because none is kept. That is not a discipline applied to a design that could go either way -- it is why every recovery path below can be written at all: recovering from a surprise is *re-reading the world*, never repairing a graph of handles. (`get-by-unit-number` is declared on the part prototype and is still unused; there has been no need.)
- **Clusters are per force**, and adjacency, the compiler's flood fill and the edge search all agree about it. Two forces' parts touching are two balancers; a belt of another force is never an edge, because `find_entities_filtered` applies the force filter in C++ and never returns it. This is the semantics the part prototype always claimed and the code did not have.
- **Every recompile is from the visible world.** The registry says which tiles are parts; everything else -- which belts, facing where, of what force -- is re-read at compile time.

## The failure envelope

The honest statement of what happens when nothing tells us. Another mod calling `entity.destroy()` on a belt beside a balancer raises **no event of any kind**, and no amount of subscribing can change that. What follows:

| | |
|---|---|
| **What breaks** | Nothing, immediately. The network is engine state: its linked belts do not care that a visible belt vanished, so items simply stop arriving on that port. The other ports keep balancing exactly. |
| **What is wrong** | Only the guest's fingerprint, which still describes an edge that is no longer there. A rebuild triggered for another reason would place an interface facing nothing, which is inert. |
| **What recovers it** | Any event that touches the cluster -- a belt laid within two tiles, a part added or removed, a rotation, a clone, an undo, a surface event. The edge list is re-derived from the world every time, so the first such event is correct. Also: any `bbb-audit` marker, which re-classifies every cluster and reports the drift before repairing it. M3's `noev` rig is exactly this sequence and comes back to 1.978x. |
| **What never happens** | A crash, a desync, or a duplicated network. There is no reference to go stale and no state that can disagree with itself. |

The same envelope covers a belt whose `direction` is assigned directly (what an undone rotation does to the world), and a mod that swaps a belt for another tier without raising. **A bot upgrade needs no handler at all** for the same reason: whatever the removal path was, the *build* event for the replacement fires, and that recompile reads the world as it now is. This was checked rather than assumed -- M3's `swap` rig fast-replaces an express belt with a fast one, which raises a build event and no mine event, and lands at exactly 1.667x.

`on_object_destroyed` is deliberately **not** used, though `agents/design.md` listed it. It requires registering every belt of interest by unit number, which is the per-entity bookkeeping this design exists to avoid, and it fires *after* the fact carrying only a registration number -- so it could not even say where to recompile without a unit-number-to-position table, i.e. a cache.

## Coming back on a heap this build did not write

The guest heap is discarded whenever the mod is rebuilt, and this mod deliberately keeps it that way. **It exports `fk_migrate` and must never export `fk_migrate_adopt`.** Those used to be one hook: exporting `fk_migrate` meant adopting the previous build's **entire linear memory**, `.rodata` and all, so a guest that exported it was reading its own string constants out of another program's image. Upstream split them ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 13, fixed) -- `fk_migrate(old_version)` is a **notification on a fresh heap** and `fk_migrate_adopt` is the opt-in that really hands the bytes over. This guest's state is Go maps and slices reachable only through package-level roots, which is exactly the shape adoption cannot carry across a rebuild, so the adopt half is permanently off the table.

So a mod update starts with an empty registry and a world full of parts and running networks. Nothing breaks in the meantime, and that is the point: **a compiled network does not need a script to keep running.** What must not happen is the guest deciding a cluster has no network and building it a second one.

Two mechanisms, and the second is not redundant:

- **`fk_migrate` names the moment.** It fires from `on_configuration_changed` -- after `on_load`, before the first tick, with `game` fully available -- so the rebuild happens at a deterministic point rather than inside whichever event happened to arrive first. The `upg` suite asserts the hook ran *and* that it, rather than the fallback, drove the rebuild.
- **`registryReady` stays** as the fallback: a plain bool that is false in a freshly initialised heap, so the first event of any session rebuilds before it decides anything. It covers what the hook does not -- a mod added to a save that already contains parts, a `--persist` mode changed underneath it. One bool test on the hot path, and it is not going away.

The rebuild **adopts** rather than rebuilds wherever the evidence is complete. For each cluster it finds the visible interfaces standing in the cluster's box, follows one of them to its hidden partner (whose position *is* a slot number), and compares the set of (tile, direction) it found with the edge list it just re-derived. An exact match means the network is correct -- no tick has passed since it was built -- and the cluster is adopted for about thirty host calls instead of a teardown and ~350. Anything less falls back to a rebuild. Slots no cluster claims are then swept, which is also where a hidden entity orphaned by a prototype rename would go: our four prototypes can only be renamed by a version of this mod, which is a rebuilt guest, which is exactly this path.

**`game.surfaces` binds now** ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 15, fixed upstream): a dictionary return whose key is a dyn value comes back as an ordered pair slice, and `pairs()` over this one yields the surface NAME, so the hidden surface falls out of the same walk instead of costing its own `get_surface` by name. The scan is **one host call plus one `index` read per surface** -- two on a base save. It used to probe `get_surface(1), get_surface(2), …` and stop after 64 consecutive misses: ~65 host calls on that same save, and a guess about how sparse an index can get.

**The list is sorted by index before anything walks it**, which the probe gave for free and this does not. `fk_abi.lua` explicitly declines to promise an iteration order for a dictionary return -- it walks `pairs()` -- and surface order decides the order parts are registered, which decides node ids, which decide cluster roots and slot claims. Two clients of a lockstep game disagreeing about that is a desync. It is an insertion sort because a big save has a dozen surfaces.

## How many recompiles a batch costs, and why it is now one

**`fk.Defer()`, and the bound is one build per affected CLUSTER per tick** -- not per entity, not per part. `fk_on_deferred` is a one-shot `on_tick` the host registers when the guest asks and tears down again from inside the flush, so an idle guest still pays zero registrations and zero per-tick calls; that is the M4 measurement this could not be allowed to break, and it did not ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 12, fixed upstream).

**The split is: the registry updates inside the event, the compile does not.** `AddPart`/`RemovePart` read nothing from the world -- the tile and the force are already in hand from the event -- so they cannot be deferred and are not; an entity is valid only inside its own dispatch. What is queued is a cluster id, deduplicated, with no host call behind it.

Measured, not claimed (`test/assert-m3.py` counts `compiled cluster` lines between markers, and asserts both halves):

| | before | after |
|---|---|---|
| 12-entity blueprint paste, 2-part balancer | 2 builds | **1** (0 inside the paste tick, 1 by the flush on the next) |
| `--create` of the n=200 k=4 bench save | 800 compiles | **200** |
| 100 belts pasted along a finished balancer's edge | up to 100 classifications | **1** |

The last row is where the win actually lives, and it is why `onNeighbour` matters as much as `onPart`: a belt laid next to a cluster used to cost a full edge re-classification (~16 `find_entities_filtered` for a 4×4) that the fingerprint then threw away 99 times out of 100. The paste row is the one that is easy to under-sell -- 2 → 1 looks small because a 2-part balancer is small.

Two things that did not change:

- Built parts-first (the M2 rigs), a cluster still becomes buildable partway through and the bound is per tick, so a rig built across several ticks costs one build per tick that moved an edge.
- A cluster whose edge list did not move rebuilds nothing at all -- the fingerprint still skips, and skips are still most of a churn run.

**`markDead` and `markLive` are O(N²) in the queue, deliberately, and the bound is worth writing down rather than removing.** Both dedupe by scanning the queue linearly, so queuing N clusters in one tick costs N²/2 `uint32` comparisons. N is not the number of clusters in the save, it is **the clusters touched in ONE tick**, and the queue is truncated to `[:0]` by every flush. The worst case this mod has a name for is the n=200 bench `--create`: 200 clusters in one dispatch is 20,000 comparisons, which is a handful of microseconds against the 45.8 s that create takes — three orders of magnitude below the compile it is deduplicating for. A map would allocate per tick on the guest's hottest path to save that, which is the trade "The heap diet" spent a milestone learning to refuse. **Left as it is; revisit only if something starts queuing thousands of clusters in one tick, and re-measure rather than re-argue.**

**The one-tick latency is real and is stated rather than hidden.** A network appears on the tick after its last part, and anything that has to measure the result in the tick it caused it -- which is every item-conservation check in `test/` -- has to force the flush. The shipped `bbb-audit` marker is the synchronous escape hatch and it is what the test mods use; it is also what their `on_init` uses, because `--create` never reaches a tick at all and the compiling would otherwise land on the first tick of the benchmark.

## Which events reach the guest at all

`script.on_event(id, handler, filters)` applies its filter list **in C++ before the handler runs**, and `fk.subscribe` carries one now ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 3, fixed upstream). All eleven per-entity subscriptions use it, with the same five-term list:

| term | covers |
|---|---|
| `{filter = "transport-belt-connectable"}` | every belt that can be an edge — `classifySide`'s six types — **and all four of our own hidden prototypes**, each being a clone of a base belt-connectable |
| `{filter = "name", name = "bbb-balancer-part"}` | the part, a `simple-entity-with-force` and therefore not belt-connectable |
| `{filter = "name", name = "bbb-audit"}` | the marker, a `simple-entity`, same reason |
| `{filter = "name", name = "bbb-insert-probe"}` | the insert probe, the same |
| `{filter = "name", name = "balancer-part"}` | the INCUMBENT'S part, which this mod's data stage keeps alive as a stub once the incumbent is gone ("Adopting a Belt Balancer 2 or 3 save"). A name filter for a prototype that does not exist is accepted and matches nothing, measured, so the list does not branch on the mod set |

**Five terms, not the dozen names and types the obvious version needs.** `transport-belt-connectable` is a category the filter grammar has built in, and finding it is the difference between a filter list that has to be revisited every time a prototype is added and one that does not. Verified against the pinned `runtime-api.json`: all eleven of this mod's per-entity events accept it, and `on_player_rotated_entity`/`on_player_flipped_entity` accept no filters at all and keep the in-guest gate.

What this removes is everything else on the map — an assembler placed, a tree mined, **a biter killed**. Each of those used to enter the guest and pay a dispatch plus three host calls (`name`, `position`, `surface_index`) to be rejected. `on_entity_died` is the one that mattered: on a map under attack it is the highest-frequency event there is, and none of it was ever ours.

**The in-guest name read stays, and that is not a half-measure.** The filter says "one of the things this mod cares about"; the handler still has to know *which* — a part, the marker, a piece of our own network being copied or destroyed, or a belt near a cluster. What changed is how often that is paid.

**And WHEN it is paid moved once more, for a reason that is about the heap rather than about the clock.** `onEvent` reads `position` and `surface_index` first now and buys the `name` last, because a name comes back as a Go string — `getStr` copies the host's bytes, since the arena under them is released when the call returns — and under `-gc=leaking` that copy is permanent, 32 B for one `express-transport-belt`, in the save and in every multiplayer join. A DISAPPEARANCE or a ROTATION can only concern this mod if it is on a registered tile, inside the two-tile neighbour gate, or on the hidden surface, and all three are answered from guest memory — so those events do not buy a name at all. An APPEARANCE still buys it unconditionally and always will: a part placed alone in the middle of nowhere has no neighbours to be recognised by. Measured at half of the largest-multiplier term this guest has; see "The marathon save".

## `on_forces_merged` — the one event with no per-entity form at all

`game.merge_forces(source, destination)` moves every entity of one force onto another and then destroys the source force, and it raises **no per-entity event for any of it**. Nothing else in this guest can see it, and clusters are per force, so without a handler:

- every node of the source force keeps a `pforce` naming a force that no longer exists — and that index is what `classifyEdges` hands the engine as a filter and what `createArgs` puts in a `create_entity` table;
- two clusters that touch and were two balancers only BECAUSE their forces differed are now one balancer, and the registry still says two. Two networks over tiles that belong to one cluster is the shape "Two bugs M3 found" is about, twice;
- a belt of the source force beside a balancer of the destination force was never an edge and now is — anywhere on the map.

`lifecycle.go`'s handler takes the **MERGED** half rather than the merging one: `on_forces_merged` carries `source_index` as a number, which is the only thing the registry needs, and it fires after the transfer, when the world already agrees with what the handler is about to write down. The order inside it is the same order `reconcileArea` uses and for the same reason — **a network has to come down while `nets` is still keyed by the root that owns it**, because a cluster absorbed by a merge stops being a root, `liveRootList` stops returning it, and its hidden network and its visible interfaces would stand for the rest of the session with nothing left that knows where they are.

It then remaps, re-derives every component of the surviving force by flood fill in node id order (the remap can only ADD adjacencies, so a fill per unvisited node is complete, and the smallest id in each component is the root exactly as a split does it), and rebuilds. **The rebuild sweep is over every cluster of that force rather than only the ones near the merge**, because of the third bullet above; most of them are fingerprint skips, and a merge is an administrator's keypress. The `edge` suite drives it with both forces' networks full: six clusters over sixteen parts afterwards, `drift=0`, both halves still delivering, and the items conserved.

Two caveats worth carrying:

- **The filter list is a load-time cost, not a per-event one.** It crosses as a tier-2 value and is decoded once, at subscribe time, inside `_initialize`.
- **The event ids must stay literal constants at the call site.** FkLua's constant scan prunes 218 event descriptors to the 22 this guest names; an id it cannot prove ships all of them, which is a bigger mod and nothing tells you. A loop over a slice of ids would compile and would be wrong.

  **AND IT ONCE STOPPED BEING ENOUGH ON ITS OWN, WHICH IS WHY `make mod` NOW FAILS ON A LOST PRUNE** ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 30, fixed upstream): the 2026-08-30 regen widened `fk.subscribe` 3 -> 5 operands for SubscribeNamed, `fkapi.SubscribeFiltered` stopped inlining under `-gc=custom` and ONLY there, so the eleven filtered ids reached the import as a parameter, the scan gave up and the packaged mod carried all 225 descriptors -- `fk_api_gen.lua` 23,332 -> 73,101 B parsed on every load, zip +5,513 B -- with every literal at every call site still a literal. **The `defer` in `SubscribeFilteredMasked` was a red herring**: dropping it fixed THIS shape and upstream measured it changing nothing on theirs, so what closes the CLASS is the scan -- one level interprocedural since FkLua d7c0d09, proving an id that arrives as a parameter when every direct call site of its function passes a constant -- with `//go:inline` on all six wrappers beside it as a size optimisation rather than as what correctness rests on (TinyGo lowers it to a hint an inliner may decline; Rust's `#[inline(always)]` is obeyed). Regenerated here 2026-08-31: both arms are back to `23 events subscribed, of 225` and `fk_api_gen.lua` to 23,332 B, the whole substantive bindings diff over 178,268 lines being those six pragmas and that defer, with no id and no signature moved -- and all 23 `hostSubscribe` call sites in the collected `bbb.wasm` are `i32.const`-fed inside `_initialize`. (24 since the same day's `on_force_created` subscription landed beside this in an unrelated round -- the subscribed count moves with the guest; the `of 225` bound, the prune and the tripwire do not.) **THE TRIPWIRE IS THE NEW PART**: `make mod` and `make zip` capture the packager's log (`dist/.fklua-mod.log`; the `--report` JSON sits beside it as `dist/.fklua-mod.report.json` since the sync pass) and FAIL on `was not a compile-time constant`, the fragment members, events and defines all share, red-proven the same day against the pre-regen bindings. A lost prune is a build failure here rather than one line of a report a few hundred long.

## Which FIELDS reach the guest — the undo mask

A filter decides whether the guest is entered. Once it is, the encode is **eager and complete**: every field of the payload is marshalled before the handler runs. That is the right trade for a flat event — the mean one has under five scalar fields and a host call per field would cost more — and it is the wrong one for exactly one event this mod subscribes to.

`on_undo_applied` carries `actions`, an unbounded array of tier-2 dynamic values: an undo of a 200-entity blueprint deep-copied every `BlueprintEntity` in it across the boundary so that this guest could read one `uint32`. Upstream measured that dispatch at **7.49 ms**.

`fk.subscribe` takes a **field mask** now, resolved once at subscribe time exactly as a filter is, and the two subscriptions carry it:

```go
fkapi.SubscribeMasked(fkapi.EventOnUndoApplied, fkapi.SkipOnUndoAppliedActions)
fkapi.SubscribeMasked(fkapi.EventOnRedoApplied, fkapi.SkipOnRedoAppliedActions)
```

**7.49 ms → 2.7 µs on the same subscription** ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 14, fixed upstream). Three properties make it safe to use rather than merely fast, and they are why the hand-derived offset could be deleted instead of kept as a belt-and-braces:

- **The layout does not move.** A masked container is written as `(ptr, count) = (0, 0)`, which is a reading every generated decoder already produces, so `player_index` keeps the offset the guest was compiled against and `fkapi.ReadOnUndoApplied(ptr).PlayerIndex` is now the obvious call.
- **A masked field is written EMPTY, not skipped.** The scratch buffer is reused across dispatches; leaving the bytes alone would show the guest the *previous* event's data.
- **Only optionals and containers are maskable**, and a `Skip…` constant exists only for those. Being wrong about a mask costs a value that reads absent, never a zero indistinguishable from a real one.

## `defines.*` — asked for by name, never written down

`gen-bindings` emits a `Defines<Path>()` accessor per define value now ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 11, fixed upstream), and the four cardinal directions are read through it:

```go
dirOf = [4]uint32{
    fkapi.DefinesDirectionNorth(), fkapi.DefinesDirectionEast(),
    fkapi.DefinesDirectionSouth(), fkapi.DefinesDirectionWest(),
}
plan.SetCompass(dirOf[0], dirOf[1], dirOf[2], dirOf[3])
```

**There is no constant to generate and that is the design.** A define's number is Factorio's own and is not stable across versions, and it is not in `runtime-api.json` at all — the description carries names and an order, not values. So the generated table carries the dotted PATH, `control.lua` resolves it against the running game once at load, and the guest holds a per-build id. The accessor caches on first use: one host call for the life of the mod.

**Each accessor is called directly, and computing an id would silently ship all 1137.** The pruning machinery is the same constant scan that prunes members and events — a scan for a literal reaching the `fk.define` import — so a table of ids, a loop, or an offset would compile, would work, and would add ~45 KB of paths to every save. Named directly, `fklua mod` reports **`API: 4 defines read, of 1137`**, and the packaged `fk_api_gen.lua` carries four lines.

`guest/go/plan` has no `fkapi` import, which is what lets `go test ./plan/` prove the balance property under an ordinary toolchain, so the compass is pushed into it by `initBuffers` rather than pulled. `plan.Opposite` was `(d + 8) & 15` and is now a lookup on the installed compass: the arithmetic was one more thing assumed about numbers this package no longer knows.

## Semantics fixed at M3

- **Clusters are per force.** Documented above; the hidden network is built with the cluster's force index, and the edge search is filtered by it.
- **An area or brush clone always recompiles the clusters in its destination box.** The reconcile brings them down (returning their items), destroys anything of ours the clone copied (whose contents were minted by the clone and must NOT be returned), reconciles the registry against the world -- including parts that `clear_destination_entities` destroyed without an event -- and rebuilds. It is a superset of what the per-entity `on_entity_cloned` handler already did, and it is the layer that catches what that handler does not see.
- **A surface being deleted or cleared is handled in the PRE event**, where the parts and the hidden entities are still valid, so slots are freed and items handed back. The POST event exists for one case: the surface that just went was the hidden one, and every network has to be rebuilt on its replacement.
- **Items are lost only to fractional positions and splitter internals.** Measured at 0.90% over ~100 teardowns under M3's randomised churn — and at **0.00% over 200 teardowns of a network deliberately kept FULL** by the `edge` suite, which is the sharper version of the same claim: what M3 loses is the fractional positions of a network in motion, not a fixed tax per teardown. **Where they go was fixed later** — see "A recompile is not a removal".
- **Nothing on the hidden surface is ever a balancer.** `AddPart` refuses a part whose surface index is the hidden one, whatever put it there. The guard is in the registry rather than in a caller because the consequence is item loss a player cannot see: `teardownNet` spills a network's contents beside the CLUSTER, so a cluster registered there would hand its items back to a surface nobody can reach — and it would give that cluster a bounding box inside the slot grid, which a teardown sweeps. Exercised by the `edge` suite.

## Two bugs M3 found, both silent, both from the same shape

Recorded because the shape will recur: **a cluster that keeps working while being wrong, because its own edge list never changed.**

1. `collectCluster` -- the compiler's flood fill -- had no force check when the registry's flood fills got one. A second force's cluster therefore had a bounding box that swallowed the first's, and the first's visible interfaces were destroyed by the second's very next teardown. The victim's fingerprint still matched, so it never rebuilt: it delivered **nothing at all** for 450 ticks until an unrelated event happened to rebuild it.
2. `find_entities_filtered` returns everything whose bounding box *touches* the area, and a 1x1 entity on tile n occupies exactly `[n, n+1]` -- so the obvious sweep box (`right_bottom = x1+1, y1+1`) also returns everything on tiles `x1+1` and `y1+1`. `setSearchBox` insets by a tenth of a tile. Clusters are adjacent whenever two forces build against each other and diagonally whenever anyone builds an L.

The lesson for M4 and beyond: **the fingerprint is a statement about the belts AROUND a cluster and says nothing about the interfaces ON it.** Anything that can remove an interface must put its cluster on the teardown queue, not just the build queue. That is why a build or death event naming one of our own prototypes -- which the compiler never raises, so it is always somebody else -- forces a rebuild rather than a re-classification.

## What the fingerprint covers, and the one thing it deliberately does not

FNV-1a over the edge list: for each edge, the CLUSTER TILE it sits on, the direction the interface must face, and whether it is an input or an output; then the count. A cluster tile plus a side is in bijection with the neighbouring belt tile, so position is covered; direction and in/out-ness are covered directly, so a rotation, a flip, an underground's ends being swapped and a belt appearing or vanishing all move it.

**The belt's prototype NAME is not in it, and that is the right answer rather than an omission.** The interface the compiler places sits on the cluster's own tile, not on the belt's; its whole contract with the belt is *a belt-connectable is there, facing that way, on that force*. A belt swapped for another tier at the same position and direction needs no change to the network at all -- and `P >= N` means the hidden lines have capacity for a faster one.

M3's `swap` rig is the evidence. Fast-replacing an express input belt with a fast one raises a build event, the cluster is re-classified, the fingerprint matches, and **nothing is rebuilt** -- and throughput moves from 2.000x to exactly 1.667x, which is (1 + 0.0625/0.09375) belts, purely because the belt itself is slower. The network did the right thing by not changing.

The one case this gives up is a belt whose *name* changes in a way that changes its CLASSIFICATION without changing its direction -- and there is none: `classifySide` keys on the entity's `type` and, for undergrounds and loaders, on which end it is, and every one of those changes the direction or the in/out flag too.

**THE CURVED EXIT NEEDED NOTHING HERE, AND THAT IS THE PROPERTY THAT MADE IT CHEAP.** A belt across a face that the engine bends towards us is an OUTPUT on that face, and an output is `{tile, dir, Out}` -- the same three values a straight output on the same face has, because the interface placed is the same interface and only the belt beyond it differs. So the hash is unchanged, the netInfo is unchanged, and a belt that starts curving or stops moves the edge LIST exactly as a rotation does. What DID widen is the radius the classification reads: a perpendicular belt's answer depends on two tiles that are not adjacent to the cluster at all (its rear and its far perpendicular tile), so an edit there has to reach the same recompile. It does, with nothing added -- `nearCluster` already walks Chebyshev distance 2 and both of those tiles are inside it. See compile.go, `curvesFromCluster`.

**M4 is done.** The head-to-head against both incumbents, the table above and [`bench/baselines/RESULTS.md`](../../bench/baselines/RESULTS.md) in full. The exit criterion (≥10× at scale) is met with margin on every saturated cell; the ~100× stretch target is not met on whole-tick cost and never could have been, because what is left is the hidden network's own engine cost rather than overhead — on the axis that target was about, mod Lua, there is nothing left to measure. What M5 and anything after it inherit from it:

- **The idle claim held literally**: `scriptUpdate` came out AT the control's, not near it, and `[BBB]` lines inside a benchmark window are zero. There is no `on_tick` handler and there must never be one.
- **The verbose default cost nothing to benchmark against.** `make QUIET=1` exists and benchmark builds were expected to want it; they do not, because the guest runs no code at steady state and so logs nothing. Keeping the verbose build is what makes "zero `[BBB]` lines" an assertion rather than a tautology.
- **The one regression M4 found is the GC tail** (above). It is a property of the LIVE heap's size — which is the guest's linear memory as a Lua word table, and is the same object in every persisting mode — so anything that grows what the registry stores per part makes it worse, and that is a performance consideration and not only a save-size one. M4 attributed it to `--persist=table` specifically; the persist re-measurement showed `packed` has the same tail, so the attribution was to the wrong half of the mode. **It was not the registry either** — see "The heap diet": the heap was log lines, and the rule that survives all three attributions is the general one, *anything permanently allocated per event is a worst tick*. The per-part discipline stays; it was simply never the biggest term.
- **`--create` cost is real and is where the whole architecture's bill lands**, and batching took a **4× bite out of it**: 4 compiles per 4×4 rig as the part block grew, 800 of them for the n=200 save — now **200**, one per cluster, because the whole `on_init` is one tick. Nobody builds 200 balancers in one tick, but a large blueprint paste is the same shape and gets the same reduction.
- **`bbb-audit`** is a shipped prototype, and it earned a second job. A script placing one asks for a full re-classification and gets `[BBB] audit clusters=… drift=… unbuilt=…` plus `[BBB] stats … compiles=… skipped=… builds=… teardowns=… creates=…`. It is also the only **synchronous** "drain the deferred queue and compile now" trigger there is, which is what every test mod's `on_init` uses to keep the compiling inside `--create` where `--benchmark` cannot see it. M4 did not need it; M4's successor could not have been measured without it.

## What M5 inherited, and what it did with it

Kept because the four notes below are what shaped the answer, and the last two were written before there was one:

- **Event-time cost went DOWN, in two ways.** Filters mean the guest is not entered at all for anything that is not a belt-connectable or one of our two named prototypes, and `fk.Defer()` means a build or mine event does a registry update and a queue insertion and nothing else -- no classification, no host call on the neighbour path.
- **A benchmark that builds its rigs in `on_init` pays a one-off surface scan** at the first event. `fk_after_load` exists upstream now and is **deliberately not adopted**: it fires only after a LOAD, never on a new map, and the two loads that matter are already covered — an adopted heap needs nothing, and a rebuilt guest gets `fk_migrate`.
- **Work that reads only the event's own payload happens in the event; work that reads the world happens in the flush.** M5 obeyed it: the mask is computed from the registry (no host call, so it could have gone either way) and the entities are touched in the flush, which is what makes the sprite correct one tick after the part is placed rather than inside the event.
- **The merged tile set is free**: `collectCluster(root)` produces it, in a deterministic order, with no host call, and that is what `restyle` walks.
