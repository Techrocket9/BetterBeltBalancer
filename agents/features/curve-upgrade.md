# A save from before the curve rule keeps the reading it was built to

**The curve rule changed what a WORLD means and not only what an edit means, so the first load after the update classifies a factory somebody else built under a rule it was not built to.** The setting above is what a player turns off; this is what happens to a player who has not been asked yet. `guest/go/curveupg.go` is the pass and its header is the long form.

**What it does to a save that predates it**, measured by a throwaway spike on 2026-09-07, Factorio 2.0.77, over one save created by the guest that had no curve arm and loaded by the guest that has it:

| the rig | what the load did to it |
|---|---|
| a curve-eligible belt beside an EDGELESS part | the balancer is rebuilt **1 -> 2**. Its real output HALVES -- **712 items over a window, then 355** -- and **347** go onto a line the player never meant as an output |
| the same belt beside a part that already carries its one belt | two edges on one tile. The standing network is CONDEMNED, torn down and refused: **12 items on the ground**, and on 2.1 the balancer is dead for good |
| no such belt anywhere | adopts exactly as it stood |

**AND ON 2.0 THE SECOND ROW THEN TRIPS THE MULTI-EDGE GRANDFATHER**, which is the part of this worth writing down because nothing about it is obvious and it will be rediscovered otherwise. `refuseSingleEdge` announces, `settleEdgeMode` asks `GrandfatherNeeded`, the count is non-zero, and the mod writes `bbb-multi-edge-parts = true` and tells the force it has kept multiple belts per part working -- **for a save that never used multiple belts per part in its life**. The chain is correct at every link: the tile really does carry two edges under the new rule, and the grandfather pass carries no provenance by design ("The fold is asked the same question whatever made the clusters").

Neither row is a defect in the classifier either. The classifier is right about what those belts now mean; it is the player who has not been asked.

## The trigger is the save's own state version

**`fk_state_version` is stamped into every save beside the build id and handed back to `fk_migrate`, and every build up to 0.3.2 exported none -- so their saves read 0.** That is the whole question, asked as one integer compare: `oldVersion < stateVersion` means this world was written before a belt could turn as it left a balancer, and the load is UNDECIDED about curves until its first settling flush.

The rungs live in `guest/go/stateversion.go` and the next rule change adds one. **It is deliberately not a `bbb-curve-decided` setting**: a setting is a second thing to keep in step with the first, it is a row in a player's menu for a fact about the past, and it moves `mod_settings_sha256` -- which is what the data-stage gate hashes to say a load-time change moved nothing a player can see.

**A SAME-VERSION DEV REBUILD READS old == 1 AND DECIDES NOTHING**, and that is right rather than a gap. Since upstream's 2026-08-07 fix `fk_migrate` fires for ANY rebuild, same-version included, so this hook is reached twenty times an afternoon by whoever is working on the mod; a save stamped 1 was written by a guest that already had the curve rule, its standing networks ARE the curve-on reading, and there is nothing about it to keep.

## The signature it replaced, and the three ordinary saves that failed it

**Until 2026-09-07 the trigger was a SHAPE, and the shape was wrong.** `rebuildFromWorld` compares the edge list it re-derives against the interfaces standing; the pass asked that comparison twice, once with the curve arm's edges taken out, and adopted a cluster that matched the second reading. A cluster the mod itself compiled and nobody has touched since does answer that. **Anything else does not, and "anything else" is not exotic.** Found by review and then reproduced on 2.0.77, three shapes an old-rule save can perfectly well contain:

| the shape | what the load did with it |
|---|---|
| a HALF-BUILT cluster -- inputs, no output -- with a belt running past | `inspectNetwork` returns at `len(ents) == 0` before it compares anything, so no reading was ever taken. Compiled **1 -> 1** under the new rule and **807 items** went into a chest the player never connected |
| an old-rule cluster whose OUTPUT BELT was mined while the mod was uninstalled | the standing interfaces describe a belt that is gone, so neither reading is a bijection. Torn down and recompiled onto the curve belt, **819 items misrouted** |
| an old-rule cluster with one extra INPUT laid while the mod was uninstalled, whose curve belt lands on the tile already carrying its output | neither reading matches, so the curve edge survives: the tile carries two, the network is condemned and refused, **12 items reach the ground**, and on 2.0 `settleEdgeMode` then writes `bbb-multi-edge-parts = true` for a save with one belt on every part |

**ONE CLEAN OLD-RULE CLUSTER ANYWHERE ELSE RESCUES ALL THREE, WHICH IS WHY THE FIRST `curv` SUITE WAS GREEN.** The decision is per SAVE, so one cluster that does match is enough to write the setting and the setting is what the others then get. A save whose every affected cluster is one of the shapes above takes no decision on any load, and the write is owed only by a note nothing records. The suite had three rigs and two of them matched.

## The signature the watermark leaves, and why it is complete

While a load is undecided it is **"this cluster's classification contains at least one curve edge"**, whatever else is true of the cluster -- half-built, drifted, refused, adopted, or never compiled at all. At the moment that save was written no such belt could have been an output, because the build that wrote it had no way to read one. So the reading is complete rather than heuristic, and it needs no comparison against what is standing.

**Its one false positive is the one the old signature had**: a player who laid one of these belts while the mod was uninstalled, meaning it as an output, and updated in the same step. They turn the rule back on. It is the benign direction and it is the only one available, because nothing in a save records what a belt was FOR.

## A cluster with no curve edge is identical under both rules

...which is what lets the decision be taken at the FIRST curve edge any classification in the load produces, wherever that happens -- the rebuild's inspection, a legacy conversion's flush, or an ordinary compile in the same dispatch. **There is no ordering problem to solve**: a cluster classified earlier in this load had no curve edge, or the decision would already have been taken, and a cluster with no curve edge reads the same either way. Nothing has to be re-done.

So `curveDecide` drops the curve edges from the list it was handed, `classifyEdges` is the one place it is called from, and every later classification in the load runs with the rule off. **That is what makes adoption, the port limit, the one-belt-per-part count and the compile itself all see the world the save was written to** -- by construction rather than by a second reading passed between them. `inspectNetwork`'s curve-free retry, the `curveFreeEdges` buffer and the `multiOver` recount it needed are deleted with it, and so is `curveScanDone`.

**THE PROBE STAYS ON FOR THE REST OF THE WINDOW, THOUGH**, and that is the one place the decision and its consequence are separate. What the probe finds is also what the player is HANDED: a save with seven affected balancers has to name seven and ping seven, so switching it off at the first would trade a checklist for two host calls per face on one load in the history of a save.

**AND THE PER-TILE COUNTS ARE RECOMPUTED RATHER THAN CLEARED**, which is what keeps the 2.1 migration honest in the same dispatch. A tile whose only second edge was the curve stops counting, which is right. A tile with two STRAIGHT edges goes on counting whatever the curve decision was, so a pruned multi-edge remnant is still condemned and the player still told. `mig21` asserts the other half from the other side: neither of its fixtures has a belt across a face, so no load of them may decide anything at all -- and its fixtures are the only genuinely pre-0.3.3 worlds this repository has, so every load of them is undecided, which is exactly the state a false positive would need.

## Where the window opens and closes

It opens in `fk_migrate` when the stamp is older than this guest's rung, and in `legacyScan` when a conversion actually converted something. It closes at the first flush of the load that settles, which is `settleCurveMode`; **that flush is guaranteed, because `rebuildFromWorld` asks for one whenever the window is open** -- not only when something was found, which is the change from the old pass and the reason for it: a save that found NOTHING still has to close the window, or a belt the player lays a minute later would be read as evidence about how the world was built.

**THE INIT CASE HAS NO WATERMARK TO READ AND CANNOT HAVE ONE.** A mod being ADDED to an existing save reaches `fk_on_init`, and Factorio raises no configuration change for a guest that was not there before, so no `fk_migrate` fires and no `oldVersion` exists. The hook order on that load is `fk_on_init` -> `legacyRecheck(init)`, and then `fk_on_configuration_changed` -> `ensureRegistry` -> `legacyRecheck(config)` in the same load; the rebuild's `rebuildingFromWorld` guard is what stops its own flush closing a window the conversion has not opened yet. What decides instead is what was converted: **the incumbent shared this mod's own old limitation** -- a belt curving away from one of its parts was not an output there either -- so a `legacyScan` that converted anything opens the window before its flush, and the same first-curve-edge rule decides. It asks no watermark even where one exists, because what those parts were built to is a fact about the INCUMBENT rather than about which of our builds wrote the save.

## Where each half runs, and the order that had to move

- **the decision is taken inside `classifyEdges`**, which is unmissable: every path that asks the world what a cluster's edges are goes through that one function, so there is no call site to forget on the day somebody adds a fifth;
- **the mode is held in a FLAG and not in the cache**, which is measured rather than reasoned: `curveRecheck` runs from three load hooks and `fk_migrate` and `fk_on_configuration_changed` both fire on this load, so a primed cache is thrown away between the decision and the flush that writes. `curveForcedOff` is what `curvedExitsAllowed` answers on until the setting itself carries the answer;
- **the write and the message run where `settleEdgeMode`'s do**: `flush()`, after `endCarry()`, never from inside the rebuild. The write raises `on_runtime_mod_setting_changed` synchronously, and the rebuild may not address a player at all (`refuseAdmit`, the wake race);
- **and it runs BEFORE `settleEdgeMode` since the third shape above.** A tile whose second edge is a curve is not a tile carrying two belts in a save that is keeping the old reading, so a multi-edge summary built before this had settled would speak for a rule the save never used. Both settles also read the same `affected` buffers through `gatherAnnounced`, so the one whose answer the other depends on has to come first either way.

**AND THE ANCHOR IS WRITTEN IMMEDIATELY BEFORE THE SETTING**, which is `grandfatherMultiEdge`'s ordering exactly. `onCurvedExitsSettingChanged` reads the cached value before invalidating it and returns when the two agree, so a cache put to Off in the statement above the write makes the write's own synchronous re-entry a no-op.

**Anywhere earlier is too early, and the first passing run of this suite is where that was measured.** With the decision held in the cache and primed by the scan, `fk_on_configuration_changed`'s `curveRecheck` cleared it before the flush, `before` was therefore "not asked yet", and the handler acted on the guest's own write:

    [BBB] curved exits: a belt across a balancer's face is left unconnected;
    3 clusters re-queued, and every one whose edges did not move skips

A player told they had changed something they had not, and a whole-save re-queue from inside the flush that was settling it. Every cluster skipped on the fingerprint it never lost, so the run was green and every number in it was right -- which is what makes it worth writing down. `assert-curve.py` asserts that line is ABSENT now.

**AND THE FLIP HANDLER CLEARS THE FLAG, which was the second half of that finding and was missed.** `curvedExitsAllowed` returned early on `curveForcedOff` without populating the cache, so after a write that FAILED the handler's before/after comparison read "not asked yet" -- which compares equal to neither answer. A player flipping the setting off was told they had changed something they had not, and a player flipping it back ON was told it was "left unconnected" and then ignored, because the classifier went on answering on the flag. The forced-off arm populates the cache now, and the handler clears the flag before it asks: somebody has said what they want, and this load's own held answer is superseded.

## The latch is the watermark, and it needs no anchor

The pass writes the setting, so the classifier and the standing networks agree from then on -- and the SAVE is stamped at this guest's rung the next time it is written, so `oldVersion < stateVersion` is false forever after. Either half alone would do it; both is what makes a player who turns curves back on safe, because from then on the standing networks match the curve-on reading exactly and no load is undecided about them.

**`edgemode.CurveKeepNeeded`'s `SettingOff` arm is a SHAPE GUARD and not that latch**, which is worth saying because it reads like one. A note is recorded only for an edge the curve arm produced, and the curve arm is behind the setting -- so a save whose setting is already off produces no curve edge, no note, and the count is zero before the fold is reached. It is there so the fold agrees with the gate rather than resting on it, which is the same standing this repo gives `tune`'s unknown-option arm. **The fold itself did not move when the count's meaning did**, and there is deliberately no fourth `Setting` for "already decided": that would be a second place to keep in step with the watermark. `go test ./edgemode/` proves all six states.

## What the player is told

One message per owning force, once, with a `[gps=]` per balancer and the map charted around each one -- the third producer of that shape after the 2.1 migration summary and the 2.0 grandfather warning:

    [BBB] curved exits: kept the old reading for this save -- 7 balancers have
    a belt across a face that was not an output when they were built;
    settings.global bbb-curved-exits = false
    [BBB] curved exits: told force 1 about 6 balancers built before a belt
    could turn as it left one, 6 pings, first [gps=0,1,bbb-curv], charted 6 ...
    [BBB] curved exits: told force 4 about 1 balancers ... 1 pings ...

**`tellAffected` TAKES ITS HEADING AND ITS NOUN FROM THE CALLER SINCE THIS PASS**, and that is a fix rather than plumbing: the line was `single-edge: ... balancers built to the multi-edge rule` whoever called it, which was true while the only callers were single-edge's two and is a lie the moment a third rule speaks. What is shared is the force resolve, the ping list, the charting and the counts; what is not is the sentence. The three assertion scripts that match the old wording (`assert-mig.py`, `assert-mig21.py`, `assert-flip.py`) match it byte for byte still, because `sedgeHeading` and `sedgeWhat` are the strings they always were.

The locale entry names the menu row verbatim so the player can turn the rule on if they want it, and `go test ./tune/` checks that the row it quotes is a row that exists -- the same check the grandfather warning gets, now table-driven over both messages, because both are entries that can be perfectly present and still disagree with each other.

## The ping list names every balancer it counts -- 2026-09-15

**A checklist that names N machines and points at the first thirty-five of them.** Reported on the mod portal by Gamer433, against this message: *"You have included a ping function that you trigger after applying the patch, but unfortunately it doesn't show all the problems. Some locations where this also happened were not pinged."*

**All three producers, one buffer, one cap.** `tellAffected` builds a force's pings into `gpsBuf [900]byte` and `gpsAdd` refused one once 772 bytes were used. That is about **36 pings** for a short nauvis coordinate, **29** for a megabase's and **22** for a long space-platform name -- and the sentence above the list kept the EXACT count, which is what made a short list look whole. Only the guest's own `told force` line said otherwise. The charting hangs off `gpsAdd`'s answer, so a balancer the list did not name was not revealed on the map either, and clicking one of the pings it did carry was the only way to find out.

**WHICH balancers fell off is the rebuild's cluster-walk order**, which a player cannot predict. Measured in the `curv` suite's own world with extra affected rigs, before the fix: **30 extra -> 36 named and 36 pinged, 40 extra -> 46 named and 36 pinged, 60 extra -> 66 named and 35 pinged** -- the ping count going DOWN as the balancer count goes up, because a longer coordinate is a longer ping. In the 40-extra run the ten that were dropped included rig `H`, which is the whole of the second surface. The 40-extra row is the one the band re-takes.

**It is CHUNKED now and the buffer has not moved.** What does not fit opens the next chat line: the first carries the producer's message and its first chunk, the rest carry `[bbb] ping-list-continued`, one key for all three producers. **`force.print` was measured accepting a 789-byte parameter with no error**, so the 900 bytes are a readability bound rather than an engine one and they stay one. The `told force` line gains ` in N lines`, written only when there was more than one -- so `flip`'s 2-ping and `mig21`'s 21-ping messages are byte for byte the lines they have always been -- and ` (list truncated)` is left for the one ping that can still be lost, a cluster whose surface name cannot be read.

**The band is what makes the suite able to ask.** No suite in the estate had more than seven affected balancers, so `pings == balancers` had never been asked of a list long enough to be cut, and `assert-curve.py` and `assert-flip.py` both asserted it. Forty more of rig `A`'s shape puts **46 on the player force's checklist**: measured on 2.0.77, **46 pings in 2 chat lines, charted 46**, against the same world on the unfixed guest at **36 pings in one line, charted 36, `(list truncated)`**. The charted count agrees with the ping count in both, which is why the charting assertion could not see this.

**And two regexes could never have caught it either**: `assert-mig.py` and `assert-mig21.py` spelled the marker `, list truncated` where the guest writes ` (list truncated)`, so their optional truncation groups had never matched anything.
