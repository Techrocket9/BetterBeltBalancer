# `prio` -- `test/assert-prio.py`, a port that is fed first

**The seventeenth suite, and the first time any priority network has run in Factorio.** Twenty rate rigs on a flat scratch surface plus a bare express belt as the yardstick, 7,200 ticks, base only. What it is about is two formulas: with S belts arriving and q of the M outputs flagged, a flagged port carries `min(S/q, 1)` and every other one carries `min(max(S - q, 0) / (M - q), 1)`, exact at every load. **THE LOAD IS SET WITH BELT TIERS**, which is `m2`'s `tslow` technique used forwards rather than as a limit case: a normal-tier run is exactly a third of an express one, so a rig walks through the three regimes -- the priority tier cannot fill, it is full and the rest share what is left, everything saturated -- by choosing how many input rows run and at what tier.

Measured 2026-09-15 on **Factorio 2.0.77**, the `release/2.0` arm at 0.2.4, shipped configuration. One saturated express belt delivered **1,306 items** over the window; every cell is that rig's per-port delta divided by it.

| rig | shape, q | fed | per port, measured | the formulas |
|---|---|---|---|---|
| `p4lo` | 4->4, q=1 | one normal belt, S = 1/3 | **0.332, 0, 0, 0** | 0.333, 0, 0, 0 |
| `p4mid` | 4->4, q=1 | two express, S = 2 | **0.998, 0.333, 0.332, 0.332** | 1, 1/3, 1/3, 1/3 |
| `p4sat` | 4->4, q=1 | four express | **0.999, 0.998, 1.000, 0.998** | 1, 1, 1, 1 |
| `p2lo` | 2->2, q=1 | one normal, S = 1/3 | **0.332, 0.000** | 0.333, 0 |
| `p2mid` | 2->2, q=1 | express + normal, S = 4/3 | **0.998, 0.334** | 1, 1/3 |
| `p2sat` | 2->2, q=1 | two express | **0.999, 0.999** | 1, 1 |
| `pq2` | 4->4, **q=2** | one express, S = 1 | **0.499, 0.499, 0.000, 0.000** | 1/2, 1/2, 0, 0 |
| `p35` | 3->5, q=1 | three express | **0.999, 0.501, 0.499, 0.501, 0.499** | 1, 1/2 x4 |
| `p32` | 3->2, q=1 | three express, saturated | **0.998, 1.000** | 1, 1 |
| `pblk` | 4->4, q=1 | one express, the PRIORITY port dead-ended | **-, 0.332, 0.334, 0.334** | -, 1/3 x3 |
| `plane` | 2->2, q=1 | both inputs SIDE-LOADED, S = 1 | **0.999, 0.000** | 1, 0 |
| `ptog` | 2->2, plain | express + normal, S = 4/3 | **0.668, 0.666** | 2/3, 2/3 |
| `pin`, `pbig`, `pgrow` | the refusal rigs | | **1.000** on every port | 1 |
| `pcol1`, `pcol2` | 1->1 and 2->2, plain | express | **0.998** and **1.000, 1.000** | 1 |

**EVERY ROW IS INSIDE 0.002 OF A BELT AND THE MODEL'S OWN FIGURES ARE WHAT IT IS COMPARED WITH**, which is the result this whole construction was built for: 1,568 cases of `guest/go/plan`'s flow model said the two formulas are exact, and nothing had ever run one in a game. The bound the script enforces is 3% per port and a port the formulas give nothing is bounded at 1% of a belt; every such port came out at **exactly zero**.

**Three of the rigs measure something a chest total cannot.**

- **`p32` reads its INPUTS.** Its three sources are finite steel chests, so what each was drawn OF is the difference between two readings: **870, 870, 870, a spread of 0.00%** over the window. Equal intake under saturation is what the balancing butterfly in front of the concentrator exists for -- the concentrator's tree is deliberately asymmetric, and without BALANCE the row an input happened to sit on would decide how much of it was taken once the network filled.
- **`plane` reads the LANES.** Both of its inputs are side-loaded, so each is half a belt on ONE lane, and S = 1 with q = 1 means the priority port must carry a full belt on BOTH. A lane-preserving network fills the same chest at the same rate, so the totals cannot tell them apart: the priority port's straight tail is sampled five times and reads **12 left and 12 right at every one of them**.
- **`pblk` blocks the PRIORITY port**, which is the tier absorbing a dead end rather than a neighbour doubling up. The three normal ports take a third of a belt each, which is the whole of the one belt fed.

**The toggle, on a running 2 -> 2, through `remote.call('better-belt-balancer', 'set-part-priority', surface, x, y, on)`.** The keybind is a path no script can take -- `script.raise_event` refuses a custom input and a headless run has no player -- so the mod's own second door is what a suite drives, reaching the same `setPartPriority` with no branch below it that can tell them apart. **A HEADLESS RUN HAS A PLAYER SINCE `curs`**, which loads a save a graphical client made, so "no player" is no longer the half of that sentence that holds; whether such a player can be made to press a custom input at all is not measured here, and `curs` is where a follow-up would ask. Flag ON: **1.000 and 0.333**; flag OFF again: **0.667 and 0.667**; and across each toggle, counted either side of an audit marker inside ONE TICK, **13,249 items before and 13,249 after, 0 on the ground**. One teardown and one compile per toggle, and the part's own `graphics_variation` crosses 47 and comes back.

**TICKING EVERY OUTPUT COSTS NOTHING, which is a leg because it very nearly cost a full rebuild.** `ShapeEdges` reports a side whose every port is flagged as having none -- two tiers where the second is empty is one tier -- so the network is the plain butterfly either way. `pcol1` is a 1 -> 1, the only shape where ticking the last port is a single toggle, and `pcol2` is the 2 -> 2 form, which takes two toggles in one tick because q = 1 in the middle is a real priority network and a flush between them would compile it. Both come out at **0 teardowns and 0 compiles**, with the PART badged all the same: the flag really did move, and what collapses is what the planner reads.

**The four refusals, each with its own sentence and the balancer still running.**

| | the rig | measured |
|---|---|---|
| a priority INPUT | `pin`, a 2 -> 2 | refused at every size: `... is a priority input, which this version does not build; the flag was not set`, the part unbadged |
| too big for a priority port | `pbig`, 33 -> 2, which is P = 64 | `... does not fit; the flag was not set`, the part unbadged |
| a BUILD reaching the same bound | `pgrow`, a 32 -> 3 with two priority ports given a 33rd input belt | `cluster 150 cannot be built with 2 priority outputs over 33->3 ports`, refused in front of its own teardown |
| ...and taking a flag OFF it | the same cluster, still not fitting either way | **ACCEPTED**, and refused again as `1 priority outputs over 33->3` -- a change that cannot make anything worse is the way out of a machine a build broke |

All three rigs deliver **2.000, 3.000 and 2.000 belts before the refusals and exactly the same after**, each against its own control window, which is what "the standing network is untouched" means as a measurement rather than as a sentence.

**The spill guard, from both sides.** `pfull` is a dead-ended 4 -> 4 with one priority port, fed from tick 0, so by the time it is asked it is stationary: clearing the flag is **refused**, with the guest's own two numbers -- `the balancer holds 578 items and the network this would build can take back 192` -- and **zero items on the ground**. `pbnd` is the same shape at 2 -> 2, opened with its feed cut and the clear asked every forty ticks as it drains, so the tick it is ACCEPTED on is the tick the machine crossed under the successor's bound: refused on the first attempt, accepted on the second, **0 items on the ground** and the items conserved to the unit across the recompile. Nothing anywhere in the run spills.

**AND THE FILL FRACTION THE GUARD'S BOUND RESTS ON IS MEASURED HERE FOR THE FIRST TIME.** Three rigs are dead-ended, jammed, and then opened with their feed cut; what lands in their chests, less what their own visible belts were carrying, is what the MACHINE was holding:

| | jammed hold, measured | `plan.Reinsertable` | tile positions |
|---|--:|--:|--:|
| plain 2 -> 2 | **72** | 64 | 112 |
| priority 2 -> 2, q = 1 | **128** | 106 | 192 |
| plain 4 -> 4 | **232** | 192 | 320 |
| priority 4 -> 4, q = 1 | **578** (the guard's own reading) | 442 | 736 |

**Two of those reproduce this repository's own records to the item**: M2 measures a full 2x2 draining 72, and the `edge` suite a saturated dead-ended 4x4 draining 232. The suite measures them independently, by a different route, in a different save.

**Two legs, and the second is `upg`'s question asked of a priority save.** Under `bump_build` the guest heap is discarded, so the registry is re-derived from the world with `pprio` zero for every node -- and the only place a flag survives is the part's own `graphics_variation`. **22 clusters adopted and 0 rebuilt** on the fresh heap, `drift=0` at the audit after it, and every rate above unchanged. A rebuild that did not read the badges back would adopt every network anyway, because the interfaces are in the same places, and the loss would be silent until the first audit found the fingerprint moved -- so the three assertions fail together and none of them alone would.

The save is **twenty-two clusters over one hundred and eighty-five parts**, and the audits are exact tuples taken at TAGGED moments: `(22, 185, 22, 0, 0, 0)` at rest and after the settings paste, `(22, 185, 22, 1, 0, 1)` from the grow to the end, where `pgrow` is a cluster that still HAS its network and knows its edge list has moved past what the mod can build. Every audit a tuple is asserted against is named by the observer, because a run takes a dozen of them and half are inside a toggle, where the marker is what makes the two item counts one atomic sample.

**WHAT THE SUITE MEASURED ABOUT THE SETTINGS PASTE IS THE DOOR AND NOT THE HANDLER.** `LuaEntity::copy_settings` declares no `raises` in either pinned runtime description -- 89 methods there do and this is not one of them -- and `on_entity_settings_pasted` carries a mandatory `player_index`. So a scripted copy moves the destination's `graphics_variation` and tells this guest nothing, and shift-click joins the keybind and the miner's pocket behind the player wall. **THE POCKET CAME OUT FROM BEHIND IT THE SAME DAY**, in `curs`, which drives both of its field reports from a real cursor; whether that suite's fixture player can drive a shift-click paste or the keybind is not measured here, and it is the route a follow-up would take. The leg asserts what really happens: the engine copies the picture, the registry does not move, nothing is refused, and the mod is not told. `onSettingsPasted`'s own refusal path is the interactive checklist's.

**Red-proven twice, and the two catch different halves.**

| injected | what fired |
|---|---|
| **the executor never sets a splitter's priority** (`compile.go`'s `setPriority` call made unreachable, so every op reaches the engine with no `splitter_output_priority`) | **every rate row, by port and by number**: `p4lo` at 0.083 on all four where the priority port wants 0.333 and the rest want nothing, `p4mid` at 0.499 x4 against 1 and 1/3, `plane` at 0.499/0.499 against 1 and 0, `pq2` at 0.250 x4, `p35` at 0.600 x5, and both toggle windows. The network balances perfectly and ignores the flag, which is the one outcome this repository's own rule forbids |
| **the spill guard disabled** (`prioFitsWhatIsStanding` returning true) | **346 items on the ground** clearing `pfull`'s flag and **36** clearing `pbnd`'s, two spill lines where the run allows none, the holding refusal absent from the log, and the boundary leg naming it: *the guard's arithmetic being wrong in the direction it exists to prevent* |

Both were injected on the arm, observed, reverted, and the revert confirmed by `shasum -a 256 -c`.
