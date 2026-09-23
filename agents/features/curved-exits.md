# A belt that turns as it leaves — the exit the classifier could not see

**The mod portal's second feature request (Gamer433): a belt may not corner directly at a balancer's exit.** Inputs already worked, because a corner belt entering a balancer points AT the part and `d == back` holds. Exits did not: an exit corner is perpendicular to the face, and a perpendicular belt fell straight through `classifySide`. That is `agents/design.md`'s inherited limitation, *a belt curving away at the edge is not an output*, and it is retired.

**THE ENGINE DOES THE CURVE AND THE WHOLE FEATURE IS CLASSIFICATION.** Measured on 2.0.77 before a line was written: a belt with an empty rear and exactly ONE perpendicular feeder curves towards that feeder, any belt-connectable counts as the feeder -- an interface linked belt included -- and the curved run carries BOTH LANES (26 and 29 items per lane over a seven-belt dead-ended run, against 28 and 28 straight). So an output interface on the cluster tile is all it takes; the compiler's only job is to decide whether that tile gets one.

## The rule, and the four tests

In `classifySide`'s `transport-belt` case alone, because nothing else curves: an underground end, a loader, a splitter and a lane splitter all have a fixed input face and no amount of feeding turns them round. A belt B beside cluster tile T, running perpendicular to the face, is an OUTPUT if and only if all four hold, over B's REAR (where a straight run would come from) and its FAR perpendicular tile (the face opposite ours):

| | |
|---|---|
| 1, 2 | **neither tile is a registered part tile of ANY cluster.** Not an optimisation: the edge query filters on the six belt-connectable type names, which deliberately omit linked-belt, so a foreign cluster's own output interface standing on its own tile is INVISIBLE to tests 3 and 4 and the registry is the only thing that can see it |
| 3, 4 | **nothing standing on either tile points into B.** `classifySide`'s own reading asked at the probe tile, so an underground's input end, a loader's input type and a belt facing away all correctly fail to count, with no second list of types to keep in step -- **plus a LINKED BELT since 0.3.4**, which is the one type the probe knows and the edge query must not: its output end feeds and its input end does not, both measured. See the 2026-09-15 subsection |

**A SIDE-LOADED EXIT IS EXCLUDED ON PURPOSE and it is the one shape a player might expect and not get.** A belt whose rear is fed shares itself between two sources, so our port would deliver half a lane -- and a half-lane port backs the butterfly up, which breaks the exact balance this mod exists for. A belt that merely STARTS beside the machine is accepted, and that is vanilla's own splitter semantics: a splitter feeds the head of a line.

**THE UX CONSEQUENCE IS IN THE CHANGELOG BECAUSE A PLAYER WILL MEET IT.** A belt line that starts beside a part which already carries its belt used to sit there inert; it is a second belt on that part now, so it is refused and handed back. Starting it one tile further out, or giving it a belt behind it, is the old behaviour. The `sedge` suite's `scrv` rig is that gesture driven by script.

**AND SINCE 2026-09-07 THE WHOLE RULE IS BEHIND A MAP SETTING, `bbb-curved-exits`, ON BY DEFAULT.** That consequence is a change to what a STANDING factory means -- an update turns a belt line somebody laid years ago into a port, and on 2.1 into a refusal -- and this is what the player who did not ask for the feature turns off. See "The curve rule is a setting" below, which is where the flip, the re-classification and what it costs a save live.

## The three things it did not need, and the one measurement that decided the probe

- **The fingerprint is unchanged.** A curved output and a straight one on the same face are the same `plan.Edge` -- same tile, same direction, same `Out` -- because the interface is identical and only the belt beyond it differs. The planner is unchanged for the same reason.
- **The neighbour gate is unchanged.** The classification now depends on two tiles that are not adjacent to the cluster, so an edit there has to reach the same recompile. `nearCluster` already walks Chebyshev distance 2, B is 1 from the cluster tile, its rear is 1 diagonally and its far tile is 2. Nothing widens.
- **`belt_shape` and `belt_neighbours` are the signal this must NEVER use**, and it is a correctness matter rather than a preference. `classifyEdges` runs BEFORE `teardownForRebuild`, so on every recompile after the first the engine would report the shape our OWN standing interface produced, and the edge would oscillate in and out forever. The signal has to be blind to our own network, which a positional probe over the six belt types is by construction (linked-belt is not among them, and the engine applies the filter in C++).
- **The probe is FORCE-BLIND while the edge query keeps its force filter**, measured rather than argued: items pushed onto one force's belt arrive on another force's belt three tiles down the same line, and a belt whose only perpendicular feeder belongs to a different force reads `belt_shape = right` exactly as the same-force control does. So a rear tile holding a foreign feeder is a rear tile that is fed. It errs safely in both directions -- a foreign feeder means B has two feeders and does not curve at all -- and the same probe's negative is this rule's own: give that belt a fed rear as well and the shape goes back to `straight`.

## What it costs, which on the permanent heap is nothing

**The arm allocates nothing and that is measured both ways.** `find_entities_filtered` writes into a buffer the guest keeps (`FindEntitiesFilteredInto`), and the type is compared on the HOST (`type_is`) rather than read as a string, so no result slice and no name reaches the guest heap. The `mar` suite gained **leg H** for it -- no leg had a perpendicular-adjacent belt, so every existing slope would have been green and blind -- and it comes out **384 B per iteration with the arm enabled and 384 B with it disabled**, byte-identical. The seven standing slopes are unmoved: **1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B** over **3.92 MiB** of linear memory, calibration 1,136 B at 0.0% spread.

**The one thing that did change on an existing path is a string that stopped crossing.** `classifyStraight` asks an underground end and a loader `is "output"` instead of reading the type as a string: `BeltConnectionType` is a two-literal union in both pinned API descriptions, so one bool answers both arms at the same one host call, and the guest heap stops taking a copy. No `mar` leg has an underground or a loader as an edge, which is why no slope moved; `m2`'s `uio`, `lio` and `lsio` are unchanged to the item.

## Red-proven, and the three families fire on three different things

The arm disabled (`curvesFromCluster` returning false), everything else in place, built and run:

| suite | what fired |
|---|---|
| `m2` | **five**: `curve` delivering **0.000x** at 100% spread, both of its exits reading `left=0 right=0` over five samples, and `1 of 23 clusters have no network at all` |
| `sedge` | **four**: three refusals where four are expected, three `told force` where four are, and the `post-curve` and `final` audits at `drift=1` against the expected 2 |
| `mar` | **nothing, and that is the attribution**: leg H is 384 B/iter in both arms, which is what says the 32 B over leg B is the existing side query and not the probe |

**`sload` correctly did NOT fire**, which is worth saying: it delivers 0 with the arm on and with it off, because it is a tripwire on the DECLINE rather than on the accept. A rig that fails in both arms would be measuring nothing.

## What it costs to ship

Both packages from `make clean`, 2026-09-07, shipped config (`--persist=packed --gc=collected`), the 2.1.17 pin, same FkLua, same machine:

| | before (0.3.2) | after (0.3.3) | |
|---|--:|--:|---|
| `dist/better-belt-balancer_*.zip` | 664,013 B | **667,587 B** | +0.54% |
| `fk_module.lua` (the control guest) | 3,148,568 B | **3,192,745 B** | +1.40% |
| `dist/bbb.wasm` | 1,305,440 B | 1,318,916 B | |
| `fk_data_module.lua` (the data guest) | 3,123,134 B | **3,123,134 B** | **byte-identical** |
| `dist/bbbdata.wasm` | 573,682 B | **573,682 B** | **byte-identical** |
| members bound into the mod | 56 | **57** | of 4,870 |
| events subscribed / defines read | 24 / 4 | 24 / 4 | unmoved |

**The data guest is byte-identical and that is the shape of the whole change**: this is a runtime classification and the load path is not touched. **Net ONE member**, and the arithmetic is worth the line: `LuaEntity::type` gains its PREDICATE form (`type_is`) beside the string read `classifySide` still uses, while `belt_to_ground_type` and `loader_type` SWAP for theirs one for one. `find_entities_filtered` adds nothing at all -- the buffer form and the allocating one are the same member id, and only the guest-side wrapper differs.

## "Belt corners on exit", 2026-09-15 -- the rule is unchanged and here is the measurement

**A second report from Gamer433, who asked for the feature: "You have to be very careful not to place a straight belt adjacent to the balancer, as it immediately curves it."** The mod's author read it as a PRIORITY defect -- that the classifier grabs belts which have a straight feeder behind them, "belts lower-priority than splitter entrances". **That hypothesis is false, the rule does not change, and what the report describes is the first belt of a line.**

**EVERY GESTURE WAS DRIVEN FROM `player.build_from_cursor`, ONE BELT PER TICK**, on Factorio 2.0.77 (build 84539) against release 0.2.3 plus trunk's cursor harness, with the mod's own `[BBB]` lines and `LuaEntity.belt_shape` read at every step. **78 samples, 18 grabs, and NOT ONE of the eighteen had a fed rear:**

| gesture | grabbed? | what the mod said | plates pushed onto the player's line |
|---|---|---|--:|
| one belt on a free face, nothing else | **yes, permanent** | `compiled cluster 1 2->3 over 4 ports, 27 entities` | 7 at +58 ticks |
| a line built FORWARD past two free faces, one belt per tick | **never** | nothing | 0 |
| a line built BACKWARD, head first, one per tick | transient: each face belt grabbed until the belt behind it lands | 2x `2->3`, 1x `2->2` | 0 |
| face belt, then extended EAST (ahead) | **yes, permanent** | `2->3` | 12 |
| face belt, then extended WEST (behind) | one flush, then RELEASED | `2->3`, `2->2` | 0 |
| rear = underground OUTPUT end / express splitter / a belt of ANOTHER FORCE, all pointing in | **never** | nothing | 0 |
| a belt on an OCCUPIED part's face, multi-edge off | refused and handed back | `alert: ... worst 2`, `handed the refused piece at 0,11 ... back to player 1` | 0 |
| a settled forward line, then the belt BEHIND the face tile MINED | **yes, permanent**, until it is put back (`2->2` + `spilled 1 items`) | | 14 at +28 ticks |
| a grabbed face belt, then a belt at its FAR tile pointing in | RELEASED (`spilled 2 items`) | | 1 |

And the engine control with no balancer within ten tiles, which is what says the classifier is a faithful model of it rather than a rule of its own: a belt B with a south feeder reads `right` when its rear tile is EMPTY, holds an underground INPUT end, or holds a belt pointing away; and `straight` when its rear holds a belt, an underground OUTPUT end, or a splitter pointing into it. **That is `feedsTile` exactly**, and it is why the answer to the report is not a priority order.

**THE HEAD OF A LINE AND A DELIBERATE CORNER ARE ONE WORLD STATE.** A belt with an empty rear and one perpendicular feeder is what the engine curves, and it is what a player laying the first belt of a line produces -- so there is nothing for any rule to tell apart, at any priority. A rule that declined it would decline the feature it was asked for. **What says which the player meant is the SECOND belt**, and the row above measures both directions of that: extend the line backwards and the port goes away on the next flush, extend it forwards and it stays. The cost of a release is a teardown into a smaller machine, which spills, and that is the ordinary policy rather than anything this rule adds.

**TWO THINGS THE DEEP-DIVE MEASURED THAT THE REPORT DID NOT NAME, and neither is a defect either.** A SETTLED passing line starts curving when the belt BEHIND its face tile is mined: nothing about the face belt moved, its rear simply went empty, and 14 plates were on the player's line 28 ticks later. And a line clicked one belt at a time from HEAD TO TAIL across parts that already carry their belt is refused per belt and left with holes -- each face belt lands with an empty rear, is classified, takes that part to two belts, and the whole cluster is refused, so the piece is handed back while the player is still dragging.

**THE ONE REAL GAP IT FOUND IS A LINKED BELT AT THE REAR**, and that is 0.3.4. `feedsTile` walked `beltTypeNames`, the six EDGE types, which omit linked-belt because every visible interface this mod places IS one standing on a part tile -- a linked belt in the edge query would make a cluster's own output an edge of itself. The probe inherited the omission, so a player-placeable linked belt from another mod feeding a belt's rear was invisible to us and visible to the engine: the mod grabbed the belt, the engine kept it straight, and the interface side-loaded half a lane into a port. WormholeBelts is one such mod and one the author runs. `findAnyByPos` gets its own seven-type span now and `probeFeeds` reads the seventh, with the end type deciding it as an underground's does. The four rows behind that are measured on 2.0.77 and are in the commit: **a connected output end at the rear reads `straight`, an input end reads `right`, an UNCONNECTED output end reads `straight` too** (so nothing reads `linked_belt_neighbour`), **and an output end facing away reads `right`** (so the `d == back` gate holds for the seventh type as for the six). **Our own interfaces are structurally out of reach** rather than filtered out: a probe tile that is a registered part tile is refused two tests earlier.

**`mar`'s leg H is the gate and it did not move**: **384 B/iter, 192 B/primitive, x1.00**, with the other seven slopes at 1,280 / 176 / 1,209 / 16 / 560 / 3,736 / 2,080 B per primitive over **3.92 MiB** of linear memory, 1,136 B of calibration at 0.0% spread and 0 items lost over 200 teardowns. What that says exactly is that the WIDER FILTER costs nothing: leg H's probe tile holds no linked belt, so what is not measured there is the cost of finding one, which is one `type_is` and one `linked_belt_type` on a path that runs once per edit.

**The `curs` suite's bands (h) and (i) are the three gestures and the red proof**, and their numbers are in the `curs` section under Verification. One new member, **61 -> 62 of 4,870**: `LuaEntity::linked_belt_type`, the predicate form.
