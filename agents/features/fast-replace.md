# Fast replace — a part goes over a belt, and a belt goes over a part

**A balancer part can be placed over a piece of belt now, the way a vanilla splitter can.** It is one data-stage line and one guest check, and the two halves are not the same size at all: the line is the whole feature, and the check is the whole risk.

```lua
fast_replaceable_group = "transport-belt",   -- guest/go/data/entity.go
```

That string is base's own group for **every transport belt, underground belt, splitter and lane splitter** (`data/base/prototypes/entity/transport-belts.lua`; loaders are `"loader"` and linked belts are `"linked-belts"`, so neither is touched). `fast_replaceable_group` is an `EntityPrototype` property, so a `simple-entity-with-force` may carry one — checked against the pinned `prototype-api.json` rather than assumed.

**THE COLLISION MASK DID NOT MOVE AND MUST NOT.** `transport_belt` is still in it, which is what stops a player laying a belt THROUGH a balancer; fast replace is an exception the engine makes for the entity being REPLACED and for nothing else. `can_place_entity` for a part over a belt is still **false** after the change, and `can_fast_replace` is what went true. Measured, both before and after.

## What the engine actually does, measured

Probed on Factorio 2.0.77, base only, with a scratch mod. `can_fast_replace` is the question a player's cursor asks, so it is the one the table is about.

| the cursor holds | over | before the line | after |
|---|---|---|---|
| a balancer part | a transport belt | false | **true** |
| a balancer part | an underground belt end | false | **true** |
| a balancer part | a lane splitter | false | **true** |
| a balancer part | a splitter | false | false (two tiles wide) |
| a balancer part | a loader | false | false (another group) |
| a belt / underground / lane splitter | a part with **no** interface on its tile | false | **true** |
| a splitter or a loader | a part | false | false |
| a belt / underground / lane splitter | a part that **carries an edge interface** | false | **false**, until 0.3.3 |

**The last row was the whole of the portal report and it is `true` since 0.3.3.** `bbb-linked-belt` is a belt-connectable standing on the cluster's own tile, so a part carrying an edge holds TWO colliding entities — and the engine wants **every collider on the tile in the group**, which one prototype line alone could never satisfy. Giving the interface the part's own group is the fix and it is one line; see "A belt over a part the balancer is using" below, which is the whole pass.

**The other three hidden prototypes stay out of it, and the reason `strip()` gave was wrong twice over.** It said dropping the group keeps our belt from being offered as a replacement for a real one. First, the engine does not read a dropped group as "no group": it defaults an entity with no declared group to a **singleton group named after itself**, which is what the runtime reports for all of them (`bbb-splitter` → `bbb-splitter`, and so on). Second, and this is the half that matters, a group is only one side of a fast replace — the other is an ITEM IN A CURSOR, and no item places any of these, so none of them could ever BE the replacement whatever group it carried. The upgrade planner shuts the same door for the same reason: an interface offered as an upgrade destination comes back *"is not a valid upgrade destination entity"*, measured. What a group on one of ours decides is the REVERSE direction alone, which is why exactly one of the four has one.

**The upgrade planner cannot pair a part with a belt.** Two entities are upgrades of each other only with the same group, the same collision BOX and the same collision MASK. The masks are identical (`floor+meltable+object+transport_belt+water_tile`, measured); the boxes are not — the part is ±0.35 and every belt is ±0.4 — so the box is the only thing keeping them apart, and it is worth knowing that it is what is doing the work.

**`create_entity{fast_replace = true}` is not the gesture and must not be used as one.** Handed a replace the engine would refuse, it falls back to CREATING, and a `simple-entity-with-force` is created whatever it collides with — so a script gets a part and a belt on one tile. It was worse in the other direction while an interface-carrying part could not be replaced: `create_entity` **mined the part and then failed to place the belt**, returning nil and leaving neither. The `edge` suite's rigs therefore ask `can_fast_replace` first and build only when the answer is yes, which is the discipline that survives that state going away.

## The forward direction needs almost no guest code, and the one case it needs is a refusal

The engine mines the belt before it creates the part, so the build event arrives in a world the belt has already left; `AddPart` registers the part inside the event and the compiler re-reads the world from the deferred flush a tick later ("THE REMOVAL WINDOW IS GONE", compile.go). A PLAYER's fast replace DOES also raise the mine for the belt -- measured 2026-09-14, see below -- and it reaches `onNeighbour`, which records a claim and queues the cluster, so either ordering ends in the same recompile.

**The one case that is not free is a part dropped into the MIDDLE of a line**, where the refusal leaves a hole in a working belt line; that is "A part dropped into a running belt line, and the belt that never came back" below.

## The reverse direction is what `guest/go/fastreplace.go` is for

**A fast-replaceable group is symmetric.** The same line that lets a part replace a belt lets a belt replace a part, and **a SCRIPT's** `create_entity{fast_replace = true}` raises **no event at all** for the part it destroys: measured, the only event in the whole dispatch is the BUILD event for the belt. **A PLAYER'S BELT-OVER-PART IS NOT THAT**, and that is the 2026-09-14 correction: driven through `build_from_cursor` against a save with a connected player, the engine raises `on_pre_player_mined_item` and `on_player_mined_entity` for the PART, with the part item in the buffer, before `on_built_entity` for the belt -- so the ordinary removal path handles it (`cluster N dissolved, mined by player P`) and `reapFastReplaced` finds the tile already unregistered. The file is the SCRIPT path's guard, and the "mine event FIRST" ordering it enumerates is the one a real player takes.

So without a check the registry keeps a **phantom** — a tile it calls a balancer part which is holding somebody's belt. Measured on the guest before the file existed: three parts became two in the world and the audit went on reporting `parts=3 drift=0 unbuilt=0`, because a phantom tile is INTERIOR, the belt standing on it is never classified, and the fingerprint therefore never moves. The cluster is then wrong for the rest of the session: **the belt a player laid does nothing at all**, and the tile is inside the box every teardown of that cluster sweeps.

**The check is one map probe on the appearance path and a host call only on a hit.** Is the appearing belt-connectable's own tile a registered part tile — the centre of the 5×5 neighbourhood `onNeighbour` walks anyway — and if so, is that part still standing? In ordinary play the first answer is always no, because the part's collision mask carries `transport_belt` and the only way a belt reaches a part's tile is by replacing the part.

**It is correct under every event ordering**, and two of the three are measured rather than enumerated for safety. No mine event, which is what a SCRIPT does — the check removes the part. The mine event FIRST, which is what a PLAYER does — `onPart` has already removed it, the tile lookup misses, and no host call is made. The build first and the mine after, which nothing observed produces — the check removes it, and `removePart` on an unregistered tile is a no-op. All three end in the same registry.

**Who is credited is `builtBy`**, the player who placed the belt: a fast replace hands the replaced entity to the player doing the replacing, so if the shrinking machine cannot take back everything it was holding, that player is the one the overflow belongs to. It is the miner's pocket's rule (carry.go) reached through the other door, and `RemovePartMinedBy` is literally the same call `onPart` makes for a mine.

## The consequence a player will meet, stated rather than hidden

**Dragging a belt line across a balancer replaces every part it crosses.** A drag is a sequence of builds and each one is subject to the same check, so a belt dragged over a balancer takes out the parts under it — all of them, since 0.3.3. That is exactly what vanilla does when a belt is dragged over a row of splitters, it is the price of the group being one string in both directions, and it is recoverable: the parts arrive in the inventory as items and the machine recompiles around what is left. **No balancer is immune to a drag any more**, and the sentence that used to stand here — that one whose every part carries an edge is immune by construction — was a property of the defect rather than of the design.

## Verified, and the two red proofs

Four rigs in the `edge` suite -- `frepa` and `frepb` here, and `frepc` and `frepd` in the section below -- and they are the only rigs in that suite **built mid-run** — after the last of its existing assertions has been made, so that not one baseline in it moved. Their source chests and their stock are created in `on_init` all the same, because `count_all` is a conserved quantity and inserting twenty-four thousand items into it halfway through would read as this mod minting matter.

| | measured |
|---|---|
| **forward**: a part dropped onto a belt of a live line beside a saturated 2-part balancer | `can_fast_replace` **true**, the belt gone, the part registered, the cluster 2 → 3 parts |
| ...and what it handed back | `express-transport-belt` ×1 and the belt's eight iron plates, on the ground; the conserved total unmoved once the engine's own machine item is taken out of it |
| ...and the balancer it became | **3 → 2 over four ports, 262 262 over 350 ticks, 0.00% spread**. The line ENDS on the replaced tile under the one-belt rule, so the part it becomes takes an input and gives no output |
| **reverse**: a belt laid on the middle of a nine-part column's three-part NECK | `can_fast_replace` **true**, the part gone, the belt on the tile, the part item handed back **onto the belt that replaced it** |
| ...and the registry | 14 clusters / 196 parts → **15 clusters / 195 parts, drift=0 unbuilt=0**: the column SPLIT, and the new belt is an output of the upper half and an input of the lower one. The neck is what makes the split possible under the one-belt rule -- the target's two vertical neighbours must be edgeless, or the belt would be a second belt for one of them |
| ...and the guest said so | `a belt-connectable fast-replaced the part at 0,418`, **exactly once** |
| ...and the column kept running | `[262, 262]` → `[132, 264]`, 76% — see below |
| spills, ground items and `[BBB] error:` across these two rigs | **0**, **0** and **0** |

**The leg that used to sit here was the REFUSAL, and it is retired rather than re-recorded.** It drove a belt over `frepb`'s top part, asserted `can_fast_replace` **false** and repaired what `create_entity` mined behind the engine's own check. 0.3.3 makes that answer true, so the leg would now be asserting the defect; the gesture it named is the `frepc`/`frepd` band's subject and the tuple it contributed (`post-frep-edge`, 14 clusters of 196 parts) is gone from `EXPECT` with it.

**The column delivers less after the split and that is the correct answer.** Before: two independent belts in and two out, 2.0 belts. After: the lower cluster has TWO inputs — its own belt and the one coming down from upstairs — and one output, and a balancer equalises its inputs, so it draws half a belt from each and delivers 1.0, while the upper splits its one belt between its own output and the belt feeding downstairs and delivers 0.5. 1.5 belts of 2.0. The bound is set at 70% knowing that; what may not happen is a half that STOPS.

**RED PROOF 1, the data-stage line ON THE PART.** Remove `fast_replaceable_group` from `bbb-balancer-part` and rebuild: every `can_fast_replace` answer comes back **false**, the rigs correctly do nothing, and the suite fails with

    FAIL: post-frep-fwd: the guest saw 14 clusters of 195 parts, expected 14 of 196

**RED PROOF 2, the guest check.** Put the line back and make `reapFastReplaced` return immediately: the belt is created, the part is gone from the world, and

    FAIL: post-frep-rev: the guest saw 14 clusters of 196 parts, expected 15 of 195

with `frepb` going on delivering its pre-split figure unchanged, because nothing was rebuilt and the belt is inert. That is the phantom, and it is what a player would be left with.

## What is interactive, and since 2026-09-14 it is the PREVIEW alone

The trigger was behind the same wall as the miner's pocket and the over-limit hand-back -- a `--create` has no player, so nothing it builds can make a CURSOR do this -- and the `curs` suite is what took it down: it LOADS a save a graphical client made rather than generating one, and makes all four of these clicks from that player's own cursor. `test/interactive/` still stages gesture **E** (`make interactive-install`; [`test/interactive/README.md`](../../test/interactive/README.md) is the checklist), and the rig's tiles are verified headlessly so the coordinates in it are not a guess:

    part-over-belt(20,92)=true    at=[express-transport-belt]
    belt-over-part(20,96)                at=[bbb-balancer-part,bbb-linked-belt]
    belt-over-part(20,98)=true           at=[bbb-balancer-part]
    belt-over-part(20,100)               at=[bbb-balancer-part,bbb-linked-belt]

**Those tiles moved when the band was rebuilt single-edge and two of them answer differently since 0.3.3**, so re-derive them from `guest/go/obs/iact/main.go` rather than from memory. A part dropped into the MIDDLE of a running line takes the belt behind it as an input and the belt ahead as an output, which is two belts on one part and is refused -- so the forward half's line ENDS on the target tile and the balancer it joins becomes **3->2**, not 3->3. The reverse half's column is FIVE parts rather than four, because the belt that splits it cleanly is an edge of the part above it and of the part below it and both of those must be otherwise edgeless.

**AND THAT COLUMN STAGES THREE OUTCOMES ON FIVE TILES, WHICH IS WHY IT NEEDS NO REBUILD FOR THIS FEATURE.** All five parts are belt-replaceable now. The MIDDLE one splits cleanly; either END leaves a machine that keeps running one part shorter, because the belt lands where that part's own edge was and feeds its neighbour; and either tile BESIDE the middle hands its surviving neighbour a second belt, so that half is refused and stops. Three tiles, three answers, one rig.

What a human has to see is the PREVIEW rather than a red block over a belt, and the DRAG taking out every part it crosses -- a drag being a sequence of clicks that no `build_from_cursor` reproduces. The belt and its cargo arriving in the INVENTORY rather than on the ground, and the reverse gesture handing the PART back the same way, are measured by the `curs` suite. Grep for `a belt-connectable fast-replaced the part at`, for `compiled cluster … 3->2` and for `carrying more than one belt`.

## What it costs

Package built 2026-08-16, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 327,613 B | **330,778 B** | +0.97% |
| `fk_module.lua` | 2,534,887 B | **2,559,428 B** | +0.97% |
| `dist/bbb.wasm` | 1,106,413 B | 1,114,847 B | |
| members bound into the mod | 42 | **42** | of 4,257 — none added |

`LuaSurface.find_entity` was already bound (limit.go's revert uses it), so nothing new crosses the boundary and `make check` needed no re-pin.

**Nothing on any hot path moves, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — which is the gate a check that runs for every belt built anywhere on the map has to clear. Leg D (a belt laid 18 tiles from anything) is 32 B and leg B (a belt laid inside the neighbour gate) is 352 B, both unmoved: a `map[key]uint32` point query allocates nothing.

The `edge` suite is 4,650 → 5,850 ticks and still runs in about fifteen seconds. Its placement probe is **191 / 191 / 196 / 197 / 180 / 180, 0 off a part tile**, byte-identical to the run before this pass, plus one new `frep` sample at 192 — which is what building the rigs late buys.

## A belt over a part the balancer is using — the portal report, 0.3.3

**A part went over a belt and a belt would not go over a part, on any tile the balancer was actually using.** Reported on the mod portal by Gamer433. The asymmetry was real and it had a cause this file already recorded as a consequence: `bbb-linked-belt` stands on the cluster's own tile for every edge, so a part carrying one holds TWO colliding entities, and `can_fast_replace` was false there. What was NOT recorded is that this is a one-line fix.

**THE ENGINE WANTS EVERY COLLIDER ON THE TILE IN THE GROUP, and that is the whole measurement.** Probed on 2.0.77 with a scratch mod modelled on the real prototypes:

| group on the part | group on the interface | `can_fast_replace` for a belt over that tile |
|---|---|---|
| yes | no | **false** — what shipped through 0.3.2 |
| no | yes | false |
| **yes** | **yes** | **true**, in all four directions |

So `guest/go/data/hidden.go` gives `bbb-linked-belt` the part's own `"transport-belt"` group and nothing else changes. A transport belt, an underground and a lane splitter all replace the tile; a splitter does not (two tiles wide) and a linked belt does not (another group). **The replace destroys BOTH of ours and leaves the belt**, the partner at the other end of the link survives with a nil `linked_belt_neighbour`, and what the interface was carrying rides onto the belt that replaced it. Identical on the 2.0 multi-edge flavour, where both interfaces on a stacked tile die.

**It is safe on the other side of the group, and the reason `strip()` gave for dropping it was wrong.** A group is one half of a fast replace; the other half is an ITEM IN A CURSOR, and no item places any hidden prototype, so none of them can ever be the replacement whatever group it carries. The upgrade planner shuts the same door by itself: an interface offered as an upgrade destination comes back *"is not a valid upgrade destination entity"*, measured, because no item places it. `strip()`'s header says both of those now.

**THE WHOLE DATA-STAGE BLAST RADIUS IS ONE LINE, and that is a measurement rather than an argument.** `test/check-datastage.py --diff` either side, normalised through `jq -S`: the dump differs by `"fast_replaceable_group": "transport-belt"` on one prototype and by nothing else. Both `data_raw_sha256` arms move (base `f4fbcaa603abc93b` → `058da3c95203e4dc`, incumbent `e7001bf98d6c6771` → `f67c4bf8544a9010`) and `mod_settings_sha256` does not, on either mod set. **The engine's own `prototype_list_checksum` does not move either** — 790230733 and 223071962, unchanged — which is that instrument's documented blindness to a field value, met in this repo for the third time.

## What it costs a player, stated rather than hidden

**A belt laid where an EDGE part was becomes an edge of the part next to it along the same axis.** That is what makes this different from mining a part, where the belts that were its edges simply stop being anybody's: here the belt takes the removed part's own tile, which is orthogonally adjacent to its neighbours. On Factorio 2.1 that neighbour may already carry its one belt, and then the cluster the removal leaves cannot be built and is refused.

**AND THE TEARDOWN IS NOT THE COMPILE'S, so the check in front of `compile()` cannot save the machine.** `removePart` marks the old root dead unconditionally, `flushDead` brings the network down, and only then does `flushLive` discover that what is left is illegal — so the balancer's contents reach the ground. That is the shape "The merge that would be over the limit" describes for a MERGE, met one door along at a SHRINK, and **it is not sparable the way a merge is**: the interface standing on the replaced tile has already been destroyed by the engine, so the network is damaged whatever this guest decides. A player gets their part and their belt, an inert machine and a pile of iron beside it; mining the belt clears the refusal, and putting a part back on the tile is what makes the column a balancer again.

**And no balancer is immune to a belt DRAG any more.** Every part a drag crosses is replaced, exactly as a drag across a row of splitters replaces those. The sentence this file used to carry — that a balancer whose every part carries an edge is immune by construction — described the defect rather than the design.

The first two are in the changelog for a player and in `README.md`, in those words.

## The alert that said the machine was fine

**The alert said something false on this path and it is fixed.** Both bounds' log lines ended in *"refused BEFORE the teardown, so the standing network is untouched"*, a sentence written out once in `guest/go/limit.go` and again in `guest/go/sedge.go`. It is true of an EDIT to a working balancer, which is what both bounds were built for and what every suite drives, and false of this gesture: a belt replacing an edge part is a REMOVAL, `removePart` marks the old root dead, `flushDead` brings the network down, and `flushLive` refuses afterwards. Measured verbatim in the `frepd` window, six items handed back two lines below it. The two copies are why finding it false here did not fix it there -- and there it was false for a second reason nobody had connected to this one: a 2.0 multi-edge save opened on 2.1 CONDEMNS its remnant and tears it down twenty lines above the check, so the rebuild's own refusal there called the machine it had just demolished untouched -- on every cluster of both `mig21` fixtures. `limit.go`'s `refusalFound` is one clause for both bounds and three arms instead of one: `nets[root]` still holding the network is *BEFORE the teardown, so the standing network is untouched*, byte for byte what it always said; an open carry pool covering the cluster's tiles is *AFTER the network on these tiles came down, so its contents are handed back*; and neither is *and this cluster had no network to lose*, which is what a `mig` conversion, a spared merge and `mig21`'s own informed retry a tick later get -- the retry being a different dispatch, in which nothing came down. No assertion script's regex reaches past `worst (\d+)` or past `over the limit of (\d+)`, so the clause is behind every one of them and none moved.

**THE PLAYER-FACING COPY MAKES NO SUCH CLAIM AND IS LEFT ALONE, WHICH IS A CHECK RATHER THAN AN OMISSION.** All four refusal strings state the rule and stop -- `over-port-limit` and `single-edge-refused` are *"A balancer connects at most __1__ belts per side; this one would have __2__"* and *"Each balancer part connects to one belt. A part in this balancer would have __1__"*, and the two `-unconnected` variants add one clause about the PIECE (*"was left in place, unconnected"*), which is true of the belt on every path. That silence is deliberate and 2026-08-05's field report is why: the string used to narrate the hand-back and read as a transaction to go looking for. A machine that stopped is worth telling a player about, and the message for it already exists and is `single-edge-migrated`'s pair -- one sentence per outcome, chosen by the same fact `refusalFound` reads. Wiring the per-piece refusal to a third sentence is a feature and not a correction, so it is written down here rather than done in passing.

## The `frepc` band, measured

Two arms in the `edge` suite, built mid-run beside `frepa` and `frepb` so that no baseline in that suite moved. Both replace the TOP part of a column — the one carrying the input interface — with a south-facing belt; what differs is the part underneath. Measured 2026-09-07 on Factorio 2.0.77, both `-gc` arms:

| | `frepc`, neighbour EDGELESS | `frepd`, neighbour already has its belt |
|---|---|---|
| `can_fast_replace` | **true** | **true** |
| the part, the interface, the belt | part gone, **interface gone**, belt standing | the same |
| handed back | `bbb-balancer-part` ×1, on the belt that replaced it | the same |
| the registry | 5 parts → 4, still ONE cluster, `nets=17 drift=0 unbuilt=0 refused=0` | 3 parts → 2, `nets=16 over 17 clusters, drift=0 unbuilt=0 refused=1` |
| over equal 350-tick windows either side | **264 → 262 items**: the machine keeps its rate | **264 → 0**: a refused cluster has no network at all |
| the force | not told; nothing was refused | **told once**, cleanly |
| spilled | 0 | **6 items**, the network's own contents |

**`drift=0` on the refused arm is the signature and it is not the sixty-fifth belt's.** There the refusal happens in front of the teardown, so the cluster still HAS its network and its stored fingerprint disagrees with the world: `drift=1 unbuilt=0`. Here the network came down first, so there is no fingerprint left to disagree — `drift=0`, `nets` one short of `clusters`, `refused=1`. A suite that asserted only `unbuilt=0` would see nothing.

**ONE MACHINE ITEM, not two.** The interface is `minable`-less by construction, so destroying it yields no item and the player is handed the part alone. The assertion is an equality for that reason: a second machine item would mean one of ours had become something a player can hold.

**And the `edge` suite's one-belt-per-part negative changed sides.** It asserted ZERO refusals over the whole run, because every rig there is laid so that no tile ever carries two belts. `frepd` deliberately produces one, so the assertion is now exactly one AND inside `frepd`'s own window — which is a strictly stronger statement than the zero it replaces, and it keeps the property the zero was for: a refusal anywhere else still fails the run.

**The retired leg.** `frep-edge` drove a belt over `frepb`'s top part and asserted `can_fast_replace` **false**. That answer is the report, so the leg is gone rather than re-recorded, along with its `post-frep-edge` tuple. Its rig is untouched and every `frepb` number is what it was.

## Red-proven, and it names itself

The one line removed from `guest/go/data/hidden.go`, rebuilt, same suite. `test/run.sh` stops on the first audit tuple:

    FAIL: post-frepc-ok: the guest saw 17 clusters of 203 parts, expected 17 of 202

and run against the logs by hand, **twenty-two assertions fire**, led by the one that names the cause:

    FAIL: the engine refuses a belt over a part that carries an edge interface.
    That is the portal report itself, and it is what `fast_replaceable_group` on
    bbb-linked-belt exists to fix: with that line removed from
    guest/go/data/hidden.go this reads false and neither arm below can happen

with both `frep-can` lines reading **false**, both arms reporting `created=false part-left=true iface-left=true`, `frepd` still delivering **262** items after the edit it never took, the force told **0** times, one fast-replace removal where three are expected, and no spill in the refused window at all.

## What 0.3.3 costs

Both packages from clean, 2026-09-07, shipped config (`--persist=packed --gc=collected`), same FkLua (a1fcd04), same pin, same machine:

| | before (0.3.2) | after (0.3.3) | |
|---|--:|--:|---|
| `dist/better-belt-balancer_*.zip` | 664,010 B | **664,505 B** | +0.07% |
| `fk_module.lua` (the control guest) | 3,148,568 B | **3,148,568 B** | **byte-identical** |
| `fk_data_module.lua` (the data guest) | 3,123,134 B | **3,129,344 B** | +0.20% |
| `dist/bbb.wasm` | 1,305,428 B | 1,305,427 B | 1 B, the build stamp |
| `dist/bbbdata.wasm` | 573,674 B | 574,470 B | +796 B |
| `fk_api_gen.lua` | 24,077 B | **24,077 B** | **byte-identical** |
| members / events / defines | 56 / 24 / 4 | 56 / 24 / 4 | unmoved |

**The control guest's emitted Lua is byte-identical and so is the API table**, which is the shape of the whole change: one prototype field, and the guest half of the feature was already written. `guest/go/fastreplace.go` did not move a statement — it keys on the PART having gone, and the interface goes WITH the part rather than instead of it.

## A part dropped into a running belt line, and the belt that never came back

**A part held over the middle of a belt line is refused on Factorio 2.1, handed back, and left a HOLE in the line.** Reported by the mod's author on 2026-09-14, from a game. Every step of what the guest did was correct on its own: the engine fast-replaces (a data-stage decision nothing at runtime can veto), the part's tile then carries the belt behind it as an input and the belt ahead as an output, sedge.go refuses, limit.go mines the part back into the inventory. What was missing is the last step, which is that a refused click has to put the world back.

**The belt is restored and the player is CHARGED for it**, which is the half that is not decoration: the engine's own mine refunded them one belt item a tick earlier, so a restore that did not take it back would print belts, one per repetition of the gesture. `RemoveItem` returning 0 -- a full inventory, an item already spent, a belt spilled to the ground -- means there is nothing to pay with, so the gap stays and an alert names the item. **The belt comes back EMPTY and the cargo stays in the inventory**: the mine's buffer carried it there before any of this guest's code ran, measured at six iron plates in and six out with 97 items tracked either side and none on the ground, and reinserting them would mean taking them out of an inventory this mod never filled.

**THE ORDERING IS MEASURED, AND MEASURING IT IS THE ROUND'S OTHER FINDING.** The pass was designed with `on_pre_build`'s ordering against the mine as an ASSUMPTION that fails safe, because nothing headless could show it; it is a measurement now. Loading a save made by a graphical client under `--benchmark` keeps that save's player connected, and `player.build_from_cursor{position = ...}` drives a genuine cursor build. Five identical runs on 2.0.77, one tick, all before the call returns:

| | |
|---|---|
| `on_pre_build` | position 33.5,-45.5, no entity, the cursor's direction, player 1 |
| `on_pre_player_mined_item` | transport-belt, unit 1018 |
| `on_player_mined_entity` | the same belt, still valid, buffer holding one transport-belt and its cargo |
| `on_player_mined_item` | |
| `on_built_entity` | `bbb-balancer-part`, unit 1021 |

`on_object_destroyed` for the belt arrives AFTER the build event, which is what rules it out as the signal. A build the engine refuses raises **nothing at all**, `on_pre_build` included, so a rejected click cannot leave a stale slot.

**WHAT THAT SAYS ABOUT THE PLAYER WALL, and it is bigger than this bug.** This file had recorded since M3 that the cursor path is unverifiable headlessly, and the wall is narrower than that: it is a KEYBOARD, not a `LuaPlayer`. A save a client made keeps a player when a headless run loads it, and `build_from_cursor` is a real build with a real `player_index` -- so a COMMITTED FIXTURE SAVE WITH A PLAYER IN IT would let the estate assert the hand-back's destination, the refusal's audience, the miner's pocket's trigger and this bug's regression, none of which any suite could reach then. **IT WAS BUILT THE SAME DAY AND IS THE `curs` SUITE**, which is where this bug's own regression now lives: `test/fixtures-player/` is a client save cut to one player on nine chunks by `make player-fixture`, and gesture (a) of that suite is this gesture -- a part onto the middle of a loaded belt line -- asserted on the restore line, the tile, the direction, the force, the emptiness and the belt count. The suite was written and run BEFORE this fix, so what the unfixed guest does is a run rather than a description: seven assertions fail, the tile reads `empty` at +1, +2 and +8 ticks, and the player ends one belt richer than they started. Only the pixels stay behind the wall.

**The mechanism, and it is one slot and one slice.** `on_pre_build` is subscribed (the twenty-fifth subscription, no filter available and nothing maskable on it) and its (tick, player, tile) go into one package-level slot with no host call behind them. A player's mine that matches that slot is a fast replace, and it SKIPS the cheap gate -- a part dropped into a line far from every balancer is the common shape of this bug, and the far-belt mine would otherwise return before the name was bought. The belt is read once, there and nowhere else: name, direction, quality (through `name_is` first, so a normal-quality belt copies no string), the underground end where there is one, and the ITEM that places it; no force is read, because the restored belt takes the force of the part that replaced it, which the build note already carries. And a record from an earlier tick is dropped by the next record rather than only by the flush, because the common producer of these records is a belt-over-belt upgrade drag far from any balancer, which asks for no flush at all. `onPart` attaches it to the build note, and `revertOne` restores it after the part has been mined back out of the tile, which is the only order the collision box allows.

**What it costs.** Nothing at all on any headless path: `on_pre_build` carries a player index, there are none, and every match fails on the zero player before it compares anything. On a real server it is one payload-only dispatch per tile a build gesture touches, and about six host calls once per fast replace that is actually refused. `buildNote` grows by a comparable struct, which is a player keypress path and is zero on every note a headless run can make.

## What the restore costs to ship

Both packages from `make clean`, 2026-09-14, shipped config (`--persist=packed --gc=collected`), the 2.1.17 pin, same FkLua, same machine, same directory:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.3.zip` | 919,462 B | **930,315 B** | +1.18% |
| `fk_module.lua` (the control guest) | 3,268,293 B | **3,414,526 B** | +4.47% |
| `fk_data_module.lua` (the data guest) | 5,093,568 B | **5,093,568 B** | **byte-identical** |
| `dist/bbb.wasm` | 1,340,079 B | 1,371,743 B | |
| `dist/bbbdata.wasm` | 1,266,453 B | **1,266,453 B** | **byte-identical** |
| `fk_api_gen.lua` | 24,283 B | 25,426 B | |
| members bound into the mod | 57 | **61** | of 4,870 |
| events subscribed / defines read | 24 / 4 | **25** / 4 | `on_pre_build` |

**The data guest is byte-identical**, which is the shape of the whole change: this is a runtime path and the prototypes are untouched, so no dump golden may move and `make datastage-check` is the control that says so. **The four new members are read out of the generated table's own diff** rather than counted: `LuaControl::remove_item`, `LuaEntity::prototype`, `LuaEntityPrototype::items_to_place_this` and `LuaPrototypeBase::name`. Nothing was regenerated and the api pin did not move; all four were reachable through the committed bindings as they stood. **A fifth, `LuaControl::force_index`, was in the first cut and is not here**: the restored belt takes the force of the PART that replaced it, which the build note already carries, and a player cannot fast-replace another force's entity.

<!-- ========================== END FAST REPLACE ========================== -->

<!-- BEGIN: the 2.1 single-edge rule (2026-08-24) -->
