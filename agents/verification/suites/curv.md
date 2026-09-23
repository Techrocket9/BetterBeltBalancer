# `curv` -- `test/assert-curve.py`, a save from before the curved exit

**The fifteenth suite, and the only world in the estate that is built to a rule the guest building it no longer has.** Two things make it that, and neither works alone. Its create phase runs a PRE-STATE build of the shipped guest -- `make prestate`, `-tags prestate`, whose `fk_state_version` reports 0 the way every build up to 0.3.2 did by not exporting it at all -- so the save carries the watermark of a world that predates the rule. And `guest/go/obs/curv` builds every balancer with no perpendicular belt anywhere near it, forces the compile with an audit marker, and only THEN lays the belts the new rule would read as outputs, with `create_entity` and **no `raise_built`** so the mod is never told. That is `m3`'s `noev` idiom used to FORGE a save rather than to provoke a defect: the prestate build's classifier is the shipped one, so a network compiled beside those belts would have taken them.

Eight named rigs and a control belt, over two surfaces and two forces, plus a BAND of forty more. The first three are the shape the pass was written for; the next three are the shapes an adversarial review found it could not see, and each one is a perfectly ordinary thing for an old save to contain.

| rig | what it is | what the load must do with it |
|---|---|---|
| `A` | a 1 -> 1 with a THIRD, EDGELESS part below it, and the curve belt against that part | adopt it whole. The new rule would make it 1 -> 2 and HALVE the port the player has |
| `B` | the same belt against the part that already carries the spine's own output | adopt it whole. The new rule would put two edges on that tile, which the single-edge rule forbids |
| `C` | no perpendicular belt anywhere | adopt it whole, exactly as it stood -- the control on the other seven |
| `D` | HALF-BUILT: an input and no output, so the old rule gave it no network at all, and a belt line running past | leave it half-built. There is nothing standing for an adoption to compare, so the pass that compared could never have seen it |
| `E` | an old-rule 1 -> 1 whose OUTPUT BELT was mined with no event, plus a curve belt | recompile it as an INPUT-ONLY cluster. The standing interfaces describe a belt that is gone, so no reading of the world is a bijection with them |
| `F` | an old-rule 2-in/1-out over three parts with an extra INPUT laid with no event, and its curve belt on the tile that already carries the output | recompile it 2 -> 1. Under the new rule that tile carries two edges, so the machine is condemned AND the multi-edge grandfather speaks for a save with one belt on every part |
| `G` | `A`'s shape on a SECOND FORCE | adopt it, and tell that force separately |
| `H` | `A`'s shape on a SECOND SURFACE | adopt it |
| the **band** | forty more of `A`'s shape, on the player force, fed and drained by one belt each and measured by nothing | adopt them -- and PING them. They are here to make one force's checklist longer than a chat line holds, which is what makes `pings == balancers` a question this suite can ask at all. See "The ping list names every balancer it counts" |

**THREE LEGS, AND EACH MOVES ONE VARIABLE.** Re-measured 2026-09-15 with the band in the world, Factorio 2.0.77 on the `release/2.0` arm, base only, 1,200 ticks; the control belt delivered **600** items over the window and every main is a 1 -> 1. Every cell but the four the band moves is the 2026-09-07 recording unchanged:

| | `kept` | `off` | `state1` |
|---|---|---|---|
| what created the save | the pre-state build | the pre-state build, with `bbb-curved-exits` written **false** into a `mod-settings.dat` before the map was made | **the shipped build** |
| `fk_migrate` was handed state version | **0** | **0** | **1** |
| the rebuild | 4 surfaces, 140 parts, **48 clusters, 45 adopted, 3 rebuilt** | identical | 48 clusters, **1 adopted, 47 rebuilt** |
| balancers kept | **47** | **none: no curved-exit line anywhere** | none |
| told | **force 1 about 46 with 46 pings IN 2 CHAT LINES charted 46; force 4 about 1 with 1 ping charted 1** | nothing | nothing |
| `settings.global bbb-curved-exits` | `true` in the create, **`false`** at every sample after the load | `false` throughout | **`true`** throughout |
| teardowns / spills / compiles | **2 / 1 (12 items) / 1** | identical | -- |
| items on the ground at the end | **12**, which is the spill exactly | 12 | -- |
| `A/B/C/F/G/H-main` | **600 items each, 1.000x one belt** | identical | -- |
| every `*-curve` chest | **exactly 0** | exactly 0 | **300 300 600 600 300 300 300** |
| the final audit | **`clusters=48 parts=140 nets=46 drift=0 unbuilt=0 refused=0`** | identical | -- |

**The three that are rebuilt are rebuilt because the WORLD changed under them and not because the rule did**, which is what the leg asserts rather than merely reports: D was never compiled at all, E's output belt is gone, F gained an input. The two of them that HAD a network pay a teardown, and only E spills -- its successor is input-only and cannot take its items back, where F's is a working 2 -> 1 and does. `nets` is two short of `clusters` because D and E end with an input and no output, which is a legitimate half-built state and is never counted `unbuilt`, which is exactly why it is asserted beside it.

**`off` IS THE SAME WORLD COMING OUT THE SAME WAY WITH NOTHING SAID ABOUT IT**, and that is a stronger negative than a quieter world would be: every teardown, every spill and every rate is identical, so what the leg isolates is the decision and nothing else. **`state1` is the same world again with only the watermark moved**, and its curve belts are ordinary outputs -- 300 on the halved 1 -> 2s, 600 where the main was already dead -- which is the correct answer for a save this build wrote and is what says the trigger is the version rather than the shape of the world.

**THE FRESH-WORLD NEGATIVE IS THE CREATE LOG AND COSTS NOTHING.** The create phase builds a world from nothing, so a create that logged a curve-kept line would be one firing where there was no earlier rule for anything to have been built to. The script asserts no such line, the create audit as an exact tuple, and the setting reading what that leg staged.

**564 items are seeded into the compiler's own entities before the save is written, and the create log's own `inside` line reads 564 back**, which is `mig21`'s trap met for `mig21`'s reason and then made an identity: a `--create` never reaches a tick, so nothing can have moved between the seeding and the reading, and a run in which the two disagree is one where the seeding did not take.

**And the flip handler's own line must be ABSENT**, which is a tooth the first passing run of this suite put there; so must any `single-edge:` line at all, which is the tooth rig `F` put there.

**Red-proven three times, and the three catch three different things.**

| injected defect | what fired |
|---|---|
| **`fk_state_version` not exported** -- the `//go:wasmexport` line removed, so FkLua stores 0 for both builds | **the `state1` leg, five assertions**, led by *the save this leg loaded was stamped state version 0 and it has to be 1*: the load decides, the setting reads `false` at `t1` and at `final`, and every curve chest takes 0. **The `kept` and `off` legs stay green**, which is the point -- with no watermark every save looks old, and the leg that says so is the one whose save is new |
| **the signature narrowed back to the bijection** -- `curveDecide` gated on the curve-free list being in one-to-one correspondence with what is standing, which is the pass as it shipped | **the `kept` leg, five assertions**, and they are the review's three shapes reproduced inside a suite. The guest keeps the old reading for **4** balancers rather than 7 and tells the forces about **[3, 1]** rather than [6, 1]; D is compiled 1 -> 1 onto its curve belt, E is torn down and recompiled onto its, and F is condemned and refused -- **3 teardowns, 2 spills of 24 items, 14 still on the ground**. And the log carries `single-edge: kept multiple belts per part enabled for this save -- 1 balancers use it; settings.global bbb-multi-edge-parts = true`, with a ping at F: the other rule's grandfather, fired by a curve, for a save that never used multiple belts per part |
| **the per-tile recount dropped** -- the strip left in place and `recountEdgesPerTile` returning early, so the counts go on describing the reading WITH the curves | **the `kept` leg, three assertions.** Every rate and every ping is right -- 6 and 1 balancers told, all six mains at 1.000x, every curve chest at 0 -- and F is condemned anyway on a tile whose second edge the load has already decided not to see: **2 spills of 24 items where the rigs allow 1**, and `single-edge: kept multiple belts per part enabled for this save -- 2 balancers use it`. It is the half a rate cannot see, and it is what the settle order below is often mistaken for. Taken 2026-09-07, before the band: its "6 and 1 balancers told" is 46 and 1 in today's world |
| **the CHUNKING reverted** -- `gpsAdd` refusing a ping once the buffer is nearly full, which is the guest through 0.3.3, with the band left in place | **the `kept` leg, three assertions**, and the numbers are the defect: force 1 is told about **46 balancers and handed 36 pings in one chat line**, the log carries `(list truncated)`, and the band's own anti-vacuity line fires because nothing chunked. **`charted` reads 36 against 36 pings**, agreeing with itself throughout, which is why the charting assertion this suite has had since 2026-08-24 could never have seen it |

**AND ONE INJECTION THAT DOES NOT FIRE, WHICH IS WORTH MORE THAN A FOURTH THAT DOES.** Reverting the settle order -- `settleEdgeMode` before `settleCurveMode` -- moves **not one number in the suite**, on either engine. The decision is taken inside `classifyEdges`, which is upstream of both settles, so by the time either runs there is no curve edge left for a multi-edge count to have seen. The order is kept because it reads in the direction the answers depend, and the comment in `flush()` says plainly that it is not what protects anything: `recountEdgesPerTile` is, and the row above is the proof of it.
