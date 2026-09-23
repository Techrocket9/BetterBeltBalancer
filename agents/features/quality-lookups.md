# A part at uncommon quality is a part — the quality-blind lookups, closed

**The four bare-name `find_entity` call sites the migration pass wrote up as NOT fixed are fixed, and the tenth suite exists because none of them was reachable by any rig this repo had.** The defect class, once more in one sentence: `find_entity` takes an `EntityWithQualityID` and resolves a bare name as **normal quality only** (the probe table is in the migration section), so a lookup that used it worked on every normal-quality save this repo has ever run and silently failed on a part a player built from a quality-rolled item. Closed 2026-08-20; `guest/go/findpart.go` is the fix and its header is the long form.

**`findOnTile` is the fix stated once** — a one-tile area query with a name filter, which is the question every caller was actually asking (is a thing of OURS with this name standing on this tile), with the quality left out of it the way `setSearchBox` + `findByName` already leaves it out everywhere else. All five sites go through it now, `legacyRunBuilds` included, so a sixth question about the same identity asks the same code or does not ask at all. Two details are decisions rather than plumbing:

- **Dedicated buffers, not `findByName`'s.** One caller (`reapFastReplaced`) runs on the EVENT path and the shared `searchArea`/`nameFilter` scratch belongs to the flush; a private filter struct costs a few static bytes and removes the aliasing question instead of answering it.
- **`found` and `err` are separate returns, on purpose.** `reapFastReplaced` unregisters a part on the strength of a MISS, and the old code deliberately refused to edit the registry on a failed query. A helper that collapsed the two would have turned a host error into a registry edit — which is exactly the class of quiet semantic change a mechanical refactor ships.

## What each site did while it stood, measured rather than recited

The migration section's table said what a non-normal part would do to each site; the `qual` suite's red proof ran the pre-fix guest and watched all of it happen at once (below). The sharpest two: `reapFastReplaced` really does unregister a standing part when a script builds a colliding belt on it — the audit went `clusters=4` → `3` with the part still in the world — and `restyle` really is an eternal retry, not a one-time miss: **24 skin lines instead of 6** over one 2,160-tick run, every one `set=0`, because a part it can never find is re-queried by every flush that touches its cluster.

## The `qual` suite — the tenth, base plus the quality mod

Every part in every rig is uncommon (`[BBB-QUAL] quality rig=… value=uncommon` is asserted per rig, so a run where the quality silently failed to apply fails as vacuous). One 2,160-tick run, four rigs plus a control belt:

| rig | what it is | what came out |
|---|---|---|
| `qblk` | a 2x2 BLOCK, two in and two out, saturated | skin line `parts=4 set=4 vars=21,27,17,35` (m1's own literals for the shape) exactly ONCE — a mid-run poke inside the neighbour gate provokes the extra flush that the unfixed guest turns into another `set=0` line — and **900 900 against the control's 900: 2.000x one belt, 0.00% spread**, which is the first evidence anywhere that an uncommon balancer BALANCES |
| `qcol` | a 1→1 column of four, the fast-replace TRUE POSITIVE | `can_fast_replace` **true** over an uncommon interior part (the engine does not gate the gesture on quality either), the replace really removes it, the guest unregisters it exactly once, and the column splits 1 cluster → 2 with the halves' skin lines carrying the right variations (`5,2` with `set=1` — the bottom part's picture was already right and is not re-set) |
| `qlone` | one lone part, the TRIPWIRE | a script builds a COLLIDING belt on its very tile; the part is still standing, and the registry does not move — `clusters=4 parts=75` before and after, and **zero** `fast-replaced the part` lines for its tile |
| `qlim` | the edge suite's 64-input block, uncommon, given its sixty-fifth belt | **exactly one** refusal alert (128 ports for 65 inputs over the limit of 64) and **exactly one clean `told force 1` line** — the refusal is DELIVERED, which is the whole point: `forceOfCluster` reads the force off a part of the cluster, and every part is uncommon. Delivery holds across the edit (900 items over the window) and the audit stands at `drift=1 unbuilt=0 refused=1` for the rest of the run |

**Three of the four rigs were single-edge already, which is why this suite cost the 2.1 port so little.** `qblk`'s west column carries the inputs and its east column the outputs, `qcol`'s two interior parts carry nothing, `qlone` has no belts. `qlim` was the exception -- a belt on both sides of every part is exactly what 2.1 forbids -- and it is **sixty-six parts** now: one output part above, a 2x32 input block, and one EDGELESS part below for the sixty-fifth belt. That spare part is forced rather than free, and for the reason the interactive checklist's band C records: under the rule all sixty-four input parts already carry their belt, so a sixty-fifth belt against any of them would ask the SINGLE-EDGE bound instead and this would stop being a test of `forceOfCluster` at all.

The audit walk is asserted at every step — `(4, 75, nets=0, 0, unbuilt=3, refused=0)` at t0 (the audit inside `on_init`'s marker dispatch reports BEFORE the drain it forces compiles anything, and qlone's edgeless cluster is never "unbuilt"), `(4, 75, 3, 0, 0, 0)` post-collide, `(5, 74, 2, 0, 0, 0)` post-replace, and `(5, 74, 2, 1, 0, 1)` from the refusal to the end. **The tuple is the assertion and `unbuilt=0` alone would not be**: a cluster with no inputs or no outputs is a legitimate half-built state and never counts as unbuilt, so a rig that lost half its belts would read `unbuilt=0` while delivering nothing. `nets` is written down per tag rather than compared against `clusters` here, because three of this save's five clusters are legitimately network-free by the end. The skin assertion is an **exact multiset of six lines** for the whole run, which is what makes both halves of the restyle claim — found once, and never retried — one comparison; qlim's line is **truncated by the guest at 32 variations** with a literal `...`, which no cluster in any other suite is big enough to reach, so the truncation is part of what is matched rather than parsed away.

## Red-proven, one pre-fix build, every family firing at once

The whole fix stashed, the guest rebuilt, the same suite run. Three rigs, three sites, three failure families, each naming itself:

| site | what fired on the pre-fix guest |
|---|---|
| `restyle` | **24 skin lines instead of 6, every one `set=0`, every variation 0** — the retry-forever half and the never-found half in one number |
| `reapFastReplaced` | *"the guest unregistered the lone part at (0, 40) under a COLLIDING belt"*, plus the audit walk failing at every post-collide sample (`clusters=3 parts=74` where the world holds 4 and 75) |
| `forceOfCluster` | *"the force was told about the refusal 0 time(s)"* — the alert fired and nobody was told |

**Re-run 2026-08-24 against the single-edge rigs**, with `findOnTile`'s filter given `Quality = "normal"` -- which is the pre-fix `find_entity` semantics stated exactly -- and all three families fire again with the same shapes: **24 skin lines instead of 6, every one `set=0`**, the lone part unregistered under the colliding belt with the audit walk failing at all four post-collide tags, and the force told **0 times**. The true positives stay true in that arm too.

And the true positives stayed true on BOTH arms, which is what makes them controls rather than assertions of the fix: the colliding belt was created, the real fast replace removed its part and was reaped exactly once, and the uncommon block delivered 2.000x at 0.00% in the pre-fix run too — conservation and throughput were never the defect.

## What is still behind the player wall

`revertOne`'s isPart arm shares the lookup and the fix, and its observable — the over-limit part arriving back in the inventory — needs a player, which no headless run has. Same wall as always ("The sixty-fifth belt"); the suite asserts the standing negative (**zero hand-backs**), and the interactive gesture, if anyone wants it, is the over-limit gesture from the checklist done with a quality part in hand.

## What it costs

Package built 2026-08-20, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 413,608 B | **414,060 B** | +0.11% |
| `fk_module.lua` | 2,746,726 B | **2,747,461 B** | +0.03% |
| `dist/bbb.wasm` | 1,162,312 B | 1,163,064 B | |
| members bound into the mod | 51 | **51** | none added — `find_entities_filtered` was always bound and `find_entity` stays bound for nothing (the binding is generated either way) |

**Nothing on any hot path moves, and the `restyle` design question dissolved under measurement.** The migration write-up priced the repair as "a second byte or an allocating query on the flush path" against `mar` slopes asserted to the byte — and the seven slopes came back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B, 3.92 MiB of linear memory), because the one-element slice `find_entities_filtered` returns lands in the same TinyGo allocation size class as the boxed Object `find_entity` returned. The retry cost the old code was PAYING — one host call per unfound part per flush, forever, on any save with a quality part — is gone with the defect. `reapFastReplaced`'s ordinary-play cost is unchanged: the point-query miss returns before any host call, and `mar`'s leg B (352 B) and leg D (32 B) say so. `make test` went 1m27.8s → 1m34.9s for the tenth suite.
