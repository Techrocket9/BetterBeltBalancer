# `lane` -- `test/assert-lane.py`, the lane-keeping part never mixes a belt's two lanes

The eighteenth suite, base only, both engines (every rig is built to the one-belt rule and the package takes the arm's own base pin). Every rig is fed one express belt side-loaded with iron from the north and copper from the south, so iron rides the left lane and copper the right, and the observer reads BOTH LANES of one tile on every output every eight ticks (an express tile takes under eleven to cross, so every item is seen) in three windows: before tick 600, the window a recompile under load lands in, and after. The benchmark phase loads the save as an UPGRADE (`bump_build`), so it also runs the rebuild-from-world.

| rig | what it is | what is asserted |
|---|---|---|
| `o12` | ORDINARY 1->2 on the same feed, the control | copper reaches its left lanes in every window (it swaps the lanes outright) |
| `l12` | lane 1->2; a third output laid on its edgeless part at tick 600 while it runs | every output clean in every window; the carry took items back |
| `ljam` | lane 1->2 with dead-end outputs, so the network jams; a flowing output added at tick 600 | the jammed network's whole contents (46 items on 2.1.24) come back on their own lanes |
| `lprio` | lane 1->2 with a priority output, flagged at tick 5 | clean; all flow to the priority output, none to its sibling |
| `l22` | lane 2->2, a column of seven | clean; both outputs equal |
| `ladj` / `oadj` | an ordinary part beside a lane part / beside an ordinary part, belt in one side, belt out the other | `ladj` delivers nothing (two balancers, one input-only and one output-only); `oadj` delivers |
| `lorph` | a lane 1->1 compiled in `on_init`, then destroyed with no event | the upgrade's rebuild sweeps exactly its 3 hidden entities |
| `lfor` | two lane 1->1s on adjacent tiles, one on the player force fed iron/copper and one on a second force fed steel/plastic; `merge_forces` at tick 1200 | two balancers before (each delivers only its pair), one lane-keeping 2->2 after (left lanes iron and steel, right lanes copper and plastic, never crossed; all four items in both sinks; 2 parts remapped) |
| `frep` | an ordinary 1->1 whose two parts are script-fast-replaced by lane parts at tick 300, and back at tick 900 | it keeps delivering after each swap and the final audit is clean |

Plus: every feed probe is split (anti-vacuity), each lane sink holds equal iron and copper, and the rebuild adopts the nine standing networks and rebuilds the two one-sided `ladj` clusters, `(9 networks adopted, 2 rebuilt)`; the final audit reads `clusters=10 parts=33 nets=8 drift=0 unbuilt=0`. Carry on 2.1.24: `[12, 16, 46]` (l12, the force merge, ljam).

Red proof (2026-11-10, 2.1.24): with `drain`'s lane tally forced off, copper appears on the left lane of `l12-west` and `ljam-west` in window 1. See `agents/features/lane-part.md`.
