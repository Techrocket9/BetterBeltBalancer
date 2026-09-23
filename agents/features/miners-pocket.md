# The miner's pocket — a player who mines a balancer keeps what was in it

**Mining a vanilla splitter puts what it was holding in your pocket.** Mining the last part of a balancer emptied a hidden network onto the ground beside it — conserved, placed correctly by the policy above (a dissolve *is* a removal, and a removal spills), and still not what the game does. This pass is the rest of that sentence: a removal a PLAYER caused offers its items to that player first, and only the remainder reaches the floor.

**The pool carries a `player_index` and nothing else.** A scalar, because the no-entity-references rule is absolute here and the gap is a whole tick wide: the mine arrives in one dispatch, the network is not drained until the deferred flush in the next, and a `LuaPlayer` handle does not survive that. `game.get_player` is called **fresh, at settle time**, and returning nothing is the ordinary case rather than an error — a player who left between the mine and the flush simply gets today's spill.

## Which removals get a beneficiary

| removal | beneficiary |
|---|---|
| `on_player_mined_entity` on **any part**, whether the removal dissolves the cluster or merely **shrinks** it | that player. See "The shrink was the whole feature" below — restricting this to the dissolve was the first cut and made the pocket almost inert in play |
| `on_player_mined_entity` on a **belt-connectable at a cluster's edge** — the belt beside the machine rather than a part of it | that player, **since 2026-08-02**. Removing an output takes a *port* off the machine, so this is a shrink like any other. See "A mine beside a machine is a mine of that machine" below — this row said *none* for one commit and a player found it |
| a **robot** deconstruction | none, deliberately, for a belt at the edge exactly as for a part. See below |
| `on_entity_died`, `script_raised_destroy`, a **surface** deleted or cleared, a network **forgotten** | none — nobody did those |

**The robot case is a decision, not an omission.** Vanilla sends a deconstruction robot's haul to a logistic storage chest, which needs either the robot itself — an entity reference the deferred flush cannot hold, by the rule this whole guest is built on — or a network-and-chest search the guest would have to do from scratch at settle time. Spilling is what every other non-player removal here already does and it is recoverable. Revisit only with the robot's own inventory in hand, which means doing the work **inside** the event, which means putting back the removal window `fk.Defer()` deleted.

## The shrink was the whole feature — the 2026-08-02 field report

**The pocket shipped crediting the miner only on the DISSOLVE, and a player reported that mining a full balancer in the map editor put ONE item in their inventory and the rest on the floor.** The report was accurate and the cause was not where anybody looked: not the boundary, not `insert`, not the count. It was the word *dissolves* in the table above.

**A player does not mine a balancer in one tick.** They mine a part, the machine recompiles **smaller** — fewer ports, a smaller butterfly, less transport line — they mine the next one, and so on down to the last. Every one of those steps is a SHRINK, every shrink hands back less than it drained, and the difference falls through to the spill by the fourth decision of "A recompile is not a removal". Only the final dissolve recorded a miner, and by then there was almost nothing left in the machine. Measured headlessly on a saturated four-part 4×4 with dead-ended outputs, mined one part per tick:

| step | what the guest did | cumulative on the ground |
|---|---|--:|
| mine part 0 | shrink: 232 drained, 224 reinserted, **8 spilled** | 7 |
| mine part 1 | shrink: 224 drained, 72 reinserted, **152 spilled** | 156 |
| mine part 2 | shrink: 72 drained, 26 reinserted, **46 spilled** | 200 |
| mine part 3 | **dissolve**: 26 drained — the only step the pocket ever saw | 224 |

**206 of 232 items were already on the floor before the beneficiary was consulted at all**, and a smaller or less loaded balancer leaves the dissolve holding a single item — which is exactly the report. **Editor mode is where it looks worst rather than where it is different**: the editor controller ships `mining_speed = 6` and `instant_deconstruction = true`, so a player rips the balancer apart fast enough that the whole machine appears on the ground at once.

**The fix is one call site**: `removePart` records the miner for every part a player mines, not only for the one that empties the cluster. Nothing else moves, and the precedence is what makes that safe — a claimed pool is never pocketed, so the survivor still claims the shrink's pool and `takeCarry` still reinserts everything that fits. The beneficiary is consulted by `settleCarry` over the **remainder alone**, so a shrink that fits entirely — the ordinary one, and every edge edit in the `edge` suite — reaches none of this and costs nothing.

**What it cost to find, and the lesson.** The first hypothesis was that the count on the `ItemStackDefinition` was not reaching the engine — `ItemStackDefinition.count` defaults to `1` when it is absent, so "exactly one item pocketed" is that defect's exact signature, and the ABI was the obvious suspect. It was wrong, and disproving it is what `probe.go` now is: a shipped diagnostic that asks a **chest** the question the pocket asks a player. That took the boundary off the table in one headless run and left the guest's own policy as the only place the defect could be. **The lesson is the one this file already records in another form: a path that no suite can reach is a path whose bugs a player finds.** The pocket was declared unverifiable headlessly and the declaration was half true — the trigger needs a player, the *arithmetic* and the *quantity* never did, and both are pinned now.

## A mine beside a machine is a mine of that machine — the second field report

**Place a downward-facing output belt on a running balancer, then mine it again: a small pile of items on the floor, with an inventory that had room for them.** Same reporter, same day, and the same shape as the shrink report one level out — the policy sentence already covered it and the implementation asked a narrower question.

**Nothing was lost and nothing was placed wrongly.** Removing an output takes a *port* off the machine, and `P = next_pow2(max(N, M))` is a step function: two outputs to three took P from 2 to 4 when the belt was laid, and mining it takes P back to 2. The butterfly halves, the network the recompile builds cannot hold what the one it drained was holding, and the difference falls through to the spill by decision 4 of "A recompile is not a removal". That part is correct and stays. What was wrong is where the difference went: `removePart` records a claim and `onNeighbour` did not, so a **belt** mined at the edge had no beneficiary and the overflow went to the floor rather than to the miner.

**Why seven suites were green while it happened, and it is not "no suite drove that edit".** The `edge` suite drives exactly that edit — `shrk`, an output belt mined off a saturated 4×4 — and CLAUDE.md quoted it as evidence for the old policy row: *"the `edge` suite's `shrk` leg measures that reinsertion fitting with room to spare"*. **The measurement was true and the generalisation was not.** `shrk` goes four outputs to three, and `next_pow2(max(4, 3))` is 4 exactly as `next_pow2(max(4, 4))` is: the machine it rebuilds is the *same size*, so of course everything fits. The suite's own bound (40 items) has never been approached because nothing was ever overflowing. **A shrink that does not shrink the butterfly is not evidence about shrinks**, and reading it as such is what let the report through.

**The fix is one call site and one tile.** `onNeighbour` records the miner too, and the claim goes on **the part tile the gate just looked up** — not on the mined belt's tile, which is one outside the network's box by construction and would answer to no pool at all. That is the trap the obvious fix falls into: the part path passes the tile the *event* reported, because for a part that tile is the network's, and reusing the same call on the neighbour path compiles, runs, records a claim per mine and changes nothing. Tile-keyed rather than root-keyed for the reason "The three decisions" already gives — `flushLive` re-resolves every queued root a tick later and a part mined elsewhere in the same tick re-roots the survivors — so the two paths agree by construction instead of by argument.

**It costs nothing on the guest's hottest path**, which is the one property that was never negotiable: the gate is entered for every belt anyone lays or picks up anywhere on the map, and the tile, the force (`pforce[id]`) and the player index are all already in hand. There is **no host call**, and the note is skipped outright unless a `player_index` arrived — which is every event but one. `carry.Claims.Add` also dedupes exact repeats now, because the gate calls it once per part tile in a 5×5 neighbourhood and a deconstruction planner dragged along a belt line beside a balancer would otherwise grow the store with the *sweep* instead of with the *machine*.

**The `bmin` rig, and it is a tripwire rather than a proof.** The `edge` suite gained the shape the report was made with: a saturated two-part balancer, two in and two out, dead-ended so it stays full, with a third output belt added while it runs and then mined again — P going 2 → 4 → 2, which is the boundary crossing `shrk` does not make. Measured 2026-08-02:

| | measured |
|---|---|
| the belt **laid** (P 2 → 4, the machine grows) | **0 items on the ground**, like every other edge edit |
| the belt **mined** (P 4 → 2, the machine halves) | **128 items on the ground** — the field report, reproduced |
| the same run on the guest **before** the fix | **128**, byte for byte |

That last row is the honest split and it is why the assertion is a **floor** (40) rather than an equality: `player_index` is 0 on every removal a headless run can produce, so the redirection is invisible here and only the *quantity* is not. The suite now has **three** spill windows and no more — the dissolve, the by-hand teardown's shrinks, and this — and a spill outside any of them still fails the run. What the fix does with those 128 items is checked by `go test ./carry/` for the identity, by the chest probe for the arithmetic, and by a player for the trigger.

## The three decisions

1. **Precedence: a claimed pool is never pocketed.** The four decisions of "A recompile is not a removal" run first and unchanged. If any cluster built in the same flush succeeds the network geometrically — a merge, a split, a shrink, a plain recompile — the items go **into it**, and the beneficiary is never consulted. `settleCarry` is the only caller, and it sits between the claim and the ground. So the ordering is: **network, then miner, then floor.**

2. **A claim is keyed by TILE, not by cluster root**, and that is the one thing the obvious implementation gets wrong. A decon planner mining a four-part balancer in a single tick removes three parts that merely *shrink* it — each re-rooting the survivors at the smallest surviving node id — and then a fourth that dissolves, so the root at the moment of the dissolve is **not** the root the `netInfo` is filed under, and a root-keyed claim would silently miss. The mined part's **tile** is inside the visible bounding box of the network coming down whichever root owns it, so a claim is `(surface, force, tile, player)` and `openPool` takes the first claim standing on a tile of the network it was handed. First in event order, which the engine makes deterministic, so two players mining into one dissolve resolve identically on every client. **The `force` term is the 2026-08-02 correction below and it was missing for two commits.**

   **And the tile is the NETWORK's, which is not the same as the mined entity's.** For a part the two readings coincide and the distinction never came up; for a **belt at the cluster's edge** they do not, because a belt adjacent to a cluster is by construction one tile *outside* its box — so the neighbour path passes the tile of the **part it was touching**, which is the registry key the gate just looked up. Handing over the tile the event reported would record a claim no pool can ever answer, silently, which is the whole of `carry/beside_test.go`.

3. **`LuaControl.insert`, plain counts, one host call per item kind.** The count may span many inventory stacks, so a 72-item pool of one kind is one call and it returns how many were actually taken. The **belt** stack size is dropped here on purpose: an inventory has no notion of one, and vanilla mining loses it too. "Stacked belts come back stacked" recovers stack density for the *reinsertion* path, which is the only path where it means anything.

**Why not `event.buffer`, which is how vanilla does it.** `on_player_mined_entity` carries a `LuaInventory` the engine then empties into the player, and it is the right answer for the entity being mined — which is the **part**, a `simple-entity-with-force` holding nothing at all. The items are in a hidden network on another surface, they are not read until the flush a tick later, and the buffer is valid only inside its own dispatch. Reaching it would mean draining inside the event, which is exactly the removal window `fk.Defer()` retired, for a cosmetic difference in where the items land.

## What it costs

One `uint32` on `carryPool`, one `(surface, force, tile, player)` entry per cluster a player shrank or dissolved in one tick — truncated by every settle, high-water like the recompile queues, and **bounded by the machine rather than by the gesture** since `Claims.Add` dedupes exact repeats (a belt line deconstructed beside a balancer walks the same handful of part tiles once per belt) — and **one bound member**, `LuaControl.insert`, taking the mod from 36 to 37 of 4,187. `game.get_player` was already bound for `on_undo_applied`. Nothing on any hot path: the claim list is empty on every tick nobody mined a balancer, `beneficiaryFor` is a scan of an empty slice, and the pocket itself costs one `get_player` plus one `insert` per item kind on the one dispatch where a machine was removed. Package built 2026-08-02, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 278,309 B | **281,920 B** | +1.30% |
| `fk_module.lua` | 2,112,476 B | **2,159,726 B** | +2.24% |
| members bound into the mod | 36 | **37** | of 4,187 |

**The 2026-08-02 correction added no runtime cost and one prototype.** The fix itself is a call site moved four lines up; what is not free is `probe.go`, which is diagnostic-and-test machinery that ships for the same reason `bbb-audit` does. Measured over the same round:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 281,920 B | **287,334 B** | +1.92% |
| `fk_module.lua` | 2,159,726 B | **2,222,176 B** | +2.89% |
| members bound into the mod | 37 | **38** | of 4,187 (`LuaEntity.type`) |
| shipped prototypes | 5 | **6** | `bbb-insert-probe`, hidden, script-only |

Nothing on any hot path moves: the probe's queue is empty on every tick nobody placed a marker, so `flush` gains one length test on a nil slice, and the entity filter gains a third name term that the engine decodes once at subscribe time.

## What is verified, and what the user has to check by hand

**Only the TRIGGER is interactive now.** The wall is real and unchanged — nothing headless can make this guest resolve a `LuaPlayer` — but the two things that were behind it with the trigger are not behind it any more, and the field report is why. Both were added 2026-08-02 and both live in the `edge` suite.

**The insert arithmetic, asked of a steel chest.** `insert` is a member of `LuaControl`, and a chest is a `LuaControl`, and so is a character: `LuaEntity.insert`, `LuaPlayer.insert` and `LuaControl.insert` are **one member id, one signature and one tier-2 encode of one table**. So the exact call the pocket makes to a player can be made to a chest with no player anywhere. The `bbb-insert-probe` marker (`guest/go/probe.go`, and a fifth shipped prototype alongside `bbb-audit`) is placed on a container, **deferred exactly as the pocket is** so the call happens inside `fk_on_deferred` rather than inside a build event, and offers it four legs through the very same `insertOne` the pocket uses:

    [BBB] insert probe container iron-gear-wheel asked=50 took=50 held=50
    [BBB] insert probe container iron-plate      asked=37 took=37 held=37
    [BBB] insert probe container copper-cable    asked=23 took=23 held=23
    [BBB] insert probe container steel-chest     asked=7  took=7  held=7

Four questions, one per leg, and the counts are distinct and none of them is 1 so that a wrong answer says which: a two-key `{name, count}` map; a three-key one with a **quality**, proving the optional third key does not displace the second; a two-key one **after** the three-key one, which is the shared-`carryKV`-buffer question; and one whose item name was **read out of the world** rather than written down in the guest, so it is a heap-allocated string out of `getStr` rather than a pointer into `.rodata`. The suite reads all four back from Lua as well, so the guest's own numbers are checked against something that did not cross the boundary. A count arriving as 1 — which is what an `ItemStackDefinition` whose `count` never reached the engine produces — fails the suite by name.

**The quantity, measured by taking a balancer apart the way a player does.** The `edge` suite's `hand` leg mines a saturated **ten-part** rig one part per tick and counts the ground at every step. Ten parts because a 4×4 is eight under the one-belt rule and this rig carries a spare row as well, and the order is the spare row and then row by row, west part then east part -- so every prefix leaves a CONNECTED cluster and eight of the nine shrinks leave a machine with at least one input and one output. On the shipped build:

    taking a saturated balancer apart ONE PART PER TICK:
      cumulative on the ground: pre-hand=0 ... hand-6=78 hand-7=85 hand-8=95
                                hand-9=110 hand-10=110
      the nine SHRINKS put 110 items there and the dissolve 0

Headless has no player, so those 110 items still land on the floor and every other number in the suite is what it was. What is pinned is that **the overflow is a real quantity**: the assertion is a floor rather than a ceiling, because a leg where every shrink happened to fit would satisfy every other check in the suite and would say nothing at all about the thing that was fixed. With a player, all 110 go to the miner before the ground. The dissolve gets nothing here and that is the rule showing through: the ninth step leaves ONE part, one part carries one belt, so the survivor has an input or an output and never both and `plan.Build` gives it no network at all to drain.

**...and the same quantity for the BELT at the edge**, which is the second field report's `bmin` leg. An output belt laid on a saturated two-part balancer and then mined again takes P from 2 to 4 and back:

    an OUTPUT BELT placed on a running balancer and then mined again:
      P went 2 -> 4 on the placement and 4 -> 2 on the removal
      items on the ground from the removal: 124 (floor 40), of 128 spilled

**0 on the placement, 128 on the removal, and 128 on the pre-fix guest too** — the fix is invisible to a headless run for the reason above, so what the leg is is a tripwire on the quantity and not on the redirection. It is also the leg the suite lacked: `shrk` mines an output belt as well, but four outputs to three leaves `next_pow2(max(N, M))` at 4, so nothing overflows and its bound has never been approached. See "A mine beside a machine is a mine of that machine".

**What is left, and it is only the trigger.** The `edge` suite carries a probe that runs in the same tick as the removal leg:

    [BBB-EDGE] player-mine-raise ok=false
        err=on_player_mined_entity (ID 76) (76) can't be raised through script.
    [BBB-EDGE] player-resolve p1=false players=0

Two walls, and the suite **asserts both**, so if either ever falls the run fails and says to write the real test instead of documenting it: a headless `--create` has no players, so `game.get_player(1)` is nil; and `on_player_mined_entity` is not one of the events `script.raise_event` will raise — `LuaBootstrap` carries a `raise_*` helper for each of the eleven that can be, and there is none for this one. This is the same shape as the undo/redo entry in the table below.

What the suites **do** pin:

- **the fallback**, which is the entire existing dissolve assertion: the `edge` suite's removal leg reaches the spill by a `script_raised_destroy`, records no beneficiary, and comes out at exactly the numbers it always did — **118 items spilled, 90 of them on the ground**, and every recompile still at `ground=0`. There are **three** spill windows in the suite now and no more: that removal, the `hand` leg's shrinks, and `bmin`'s port-boundary removal. A spill outside any of them fails the run;
- **the negative**, which is the half with teeth. `edge` and `m3` between them drive every removal path there is except a player mining — `die()`, `destroy()` with and without an event, a fast-replace, a shrink, a split, a force merge, a clone reconcile, a surface deleted, the hidden surface deleted, ~100 randomised stress teardowns and 200 churn teardowns — and both assert that **zero** teardowns credited a player. A claim leaking across a removal path it does not belong to fails the suite. **That assertion still holds after the shrink and then the neighbour path began recording claims**, and it is not a weaker statement: `player_index` is zero on every removal a headless run can produce, and `noteMinedByPlayer` returns on zero before it appends, so the suites' numbers did not move by a single item;
- **the log line**, `cluster N dissolved, mined by player P`, which is the only evidence a headless run could ever give that `player_index` reached the registry. `assert-log.py` counts dissolves on the word rather than on the end of the line so that the suffix cannot silently break a counter that has been green since M1. Its sibling, `cluster N offered X items to player P before the floor`, is written by `handBack` at the exact point the decision is taken and is what an interactive check greps for — it cannot fire in any suite, for the same reason;
- **the identity**, by `go test ./carry/`, which is the only machine in this repo that can check any of this. `carry/beside_test.go` is the neighbour case: that the claim goes on a tile of the NETWORK and that the mined belt's own tile answers to nobody, that one belt between two clusters credits both, that a force boundary is respected, and that a decon sweep does not grow the store.

**WHAT A PLAYER MINING DOES IS MEASURED SINCE 2026-09-14** and the `curs` suite is where: both gestures below are driven by `player.mine_entity` against a committed save that has a player in it, and both come out with `offered ... before the floor`, everything in the inventory and nothing on the floor. What a human is left checking is how it LOOKS, and the checklist keeps them for that. Two gestures, one per field report. Mine a balancer that is carrying items **part by part** and check that the items arrive in the inventory rather than on the ground at *every* step, not only the last; and **lay a belt on the spare edgeless part attached to a running balancer and mine it again**, which is the gesture that halves the machine — those items must arrive in the inventory too. (Under the one-belt-per-part rule a working balancer has no free face for a belt at all, so the checklist's band B stages a spare part to receive one; measured on that rig, the mine drains 200 items and 128 will not fit.) Grep for `offered … before the floor` either way. A full inventory must still spill the remainder, and an edge edit that does not shrink the machine must still put everything back inside the network and pocket nothing.

## A claim is a Region — the force the claim did not carry

**The claim test compared the surface and the box and not the FORCE, while the successor test over the same pool — three hundred lines away in the same file — compared all three.** Found by review, not by play, and it is the third time this repo has met the same shape: a force check that every neighbour of a predicate has and that one predicate does not (see "Two bugs M3 found", where `collectCluster` was the one).

What it could do, and it is narrow but real. Clusters are per force, so **two forces' parts touching are two balancers whose bounding boxes are adjacent by construction** — and around an L or a diagonal they overlap outright. Two such networks coming down in one tick, with a player mining one of them, and the pool of the *other* force's network could find that claim inside its own box and credit that player. Conservation was never at risk — the pool is settled either way, into a successor or a pocket or the floor — and no suite could see it, both because no headless run has a player and because the wrong player is not a wrong count. **The wrong pocket is the whole feature being wrong.**

**The fix is not a comparison, it is deleting the second predicate.** A claim is the one-tile `carry.Region` it always was — surface, force, and a box with both corners on the mined tile — and `beneficiaryFor` and `carryPool.matches` are both `Region.Overlaps` now. There is no second place left to keep in step, and a third question about the same identity asks the same code or does not ask at all.

**The first failing test this repo could write for the miner's pocket.** The trigger needs a player and always will; the *predicate* needs nothing at all, so it moved to `guest/go/carry` — pure Go, no fkapi, the third package to earn that treatment after `plan` and `skin` — and `make check` runs it. Transcribing the shipped guest's two predicates into one function and running the new tests against it fails on three of them, and all three are the same defect seen from a different side:

    --- FAIL: TestAClaimOfAnotherForceIsNotThisNetworksClaim
        network of force 1 accepted a claim on force 2's part at (12,12):
        one force's miner would pocket the other's items
    --- FAIL: TestAClaimIsJustADegenerateRegion
    --- FAIL: TestOverlapsIsSymmetric
        Overlaps(1,2) is not symmetric

The last one is worth keeping for its own sake: a pool asks the predicate one way (*does this new cluster succeed me*) and a claim the other (*is this tile mine*), so **asymmetry is exactly what "written out twice" looks like** from the outside, and it is a property a future third caller would break again.

**It costs nothing.** The force is `pforce[id]` at the mine event — registry state already in hand at the call site, read before the node is freed — so `noteMinedByPlayer` still makes no host call and cannot fail; the claim carries one more `uint32` on a slice that is empty on every tick nobody mined a balancer; and `Region` is a struct of scalars that TinyGo inlines away. Measured over the round, shipped config: the zip went **287,334 → 287,422 B (+88)** and `fk_module.lua` **2,222,176 → 2,224,002 B (+1,826)**, and no member, define or prototype was added. All seven suites are green and **their numbers did not move by a single item**, which is the expected result rather than a weak one: `player_index` is zero on every removal a headless run can produce, so `beneficiaryFor` scans an empty slice in every one of them.

## ...and a force in the key is a force that can be DESTROYED

**The same round, one level down: `remapCarryForce` followed the merge into the pools and not into the claims.** A claim carries a force index only since the section above made it a one-tile `Region`, and `game.merge_forces` is the one event that can destroy the force something wrote down. `onForcesMerged` tears the affected networks down *before* it remaps the registry — it has to, because a cluster absorbed by a merge stops being a root — so in that single tick the drained pool and any claim over it both name a force on its way out. The pool was remapped; the claim was not; the survivor's `Overlaps` then found no claim at all, and a player who mined a source-force part in the merge tick watched the items go on the floor.

**It fails closed, which is the whole reason a review found it and nothing else could.** Conservation is untouched — the pool settles into a successor, a pocket or the ground either way — and the wrong answer is the *absence* of a pocket, which no count anywhere can see. It is the narrowest window in the feature (a merge tick, a mine, the source force) and it is exactly the window a multiplayer administrator's keypress opens.

**The claim store moved to `guest/go/carry` with the predicate**, which is what makes the merge machine-checkable at all: `carry.Claims` is `Add`, `BeneficiaryFor`, `FollowMerge` and `Reset` over pure data, and `carry.Region.FollowMerge` is the one statement of the merge rule that both `remapCarryForce` loops now call. Transcribing the shipped remap — the pools alone — fails two of the eight new tests:

    --- FAIL: TestAMergeCarriesTheClaimWithThePool
        the surviving network credited player 0, not 7: a claim left naming
        the destroyed force is a claim nobody's pool can answer
    --- FAIL: TestNothingKeepsNamingTheDestroyedForce
        claim 0 still names force 2, which no longer exists

The `edge` suite's `merge_forces` leg is unchanged and could not have caught this: a headless `--create` has no player, so `player_index` is 0 on every removal any suite can produce and the claim list is empty in all seven. The leg carries a comment saying so and pointing at the test that does cover it — `go test ./carry/`, which `make check` runs.

Zip **287,422 → 287,782 B (+360)**, `fk_module.lua` **2,224,002 → 2,228,940 B**; all seven suites green from clean in the shipped config and green again in the `GC=leaking` arm, with every number byte-identical for the reason above.

**And what the second field report cost, over the same round**: zip **287,782 → 288,684 B (+902, +0.31%)**, `fk_module.lua` **2,228,940 → 2,246,630 B (+0.79%)**. No member, define or prototype was added — the whole change is a call site in the neighbour gate, a dedupe loop in `Claims.Add` and one log line.
