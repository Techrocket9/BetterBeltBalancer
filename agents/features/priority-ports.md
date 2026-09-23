# Priorities: a port that is fed first

**A player ticks a balancer part and the belt it serves is filled before the others. It is a compile-time decision and it runs no script**, which is the rule above all of this: the ticked port is a vanilla splitter with `splitter_output_priority` set on it at create time, and a shape that cannot be expressed that way is REFUSED rather than emulated. There is no `on_tick` handler, there never was one, and this feature did not add one.

**The construction is two butterflies and a pair of tier balancers.** The plain butterfly BALANCES; the same schedule with every splitter's output priority on the smaller y CONCENTRATES, because a splitter that fills one side first is a merge and a tree of merges is a sorter; so the network balances, then sorts, then taps rank `r` into one of two square sub-balancers, priority ports first. [`agents/priority.md`](../priority.md) is the whole of it, including the mirror probe that says what INPUT priority would take and why it is not built.

**The semantics are two formulas and they are exact at every load**, which is the whole reason this mod exists rather than an approximation. With S belts arriving and q of the M outputs ticked, a ticked port carries `min(S/q, 1)` and every other port carries `min(max(S - q, 0) / (M - q), 1)`. 1,568 cases over every shape with n, m <= 8, every legal q and seven loads deliver that to 1e-9 with the intake spread exactly 0, against a flow model with capacity and back-pressure in it that reproduces `PropagateLoop` on every plain shape. **AND TWENTY OF THOSE CASES HAVE RUN IN A GAME SINCE**, on both engines: the `prio` suite's rigs come out inside 0.002 of a belt of what the model says, which is the sentence this paragraph could not carry until 2026-09-15.

## The flag, and why a toggle is a recompile

`pprio`, one byte a node in `cluster.go`'s parallel slices beside `pvar`. It is the PART'S rather than the belt's: `classifyEdges` reads it once per tile and stamps it onto every edge that tile carries, which on an engine that lets a part hold two belts is both of them.

**It is in the compile FINGERPRINT**, and it has to be. The flag is the only thing that moves when a player presses the key and nothing in the world does, so a hash blind to it would make the gesture a silent no-op -- the same shape as a belt turned around with no event, which `m3`'s `swap` rig is about. It costs the `Dir` field one more bit of shift and no more mixing.

**WHAT GOES IN IS THE FLAG THE PLANNER READS, NOT THE ONE THE PART CARRIES**, which is the same collapse the refusals get: every output ticked is the plain butterfly byte for byte, so a hash over the raw flags moved on the tick a player ticked the LAST port and tore an identical network down to build it again, draining and reinserting a full one for no change. `plan.PrioCollapses` is that rule, kept beside `ShapeEdges` because that is where it is decided.

**So a toggle takes the path an edge edit takes and it is NOT A REMOVAL.** `compile` sees the fingerprint move, calls `teardownForRebuild`, and the pool that teardown opens is claimed by the network the same flush builds, matched by root because a cluster that had a network is its own successor. The items go back INSIDE the balancer. `noteMinedByPlayer` is never called from `priority.go`, so `settleCarry` has nobody to offer anything to and the reinsertion is the only outcome -- a claim is what makes a removal's leftovers somebody's property, and a toggle is not a removal.

## Where the flag survives, measured

**In `graphics_variation`**, which is the one piece of per-entity state this mod already writes and the engine already persists. Cells 1..47 are the shapes and 48..94 are the same shapes badged, so the byte is the in-world indicator at the same time. The guest heap is declined on every rebuilt guest, so a flag kept only there would be lost on every release of this mod and a player's factory would quietly go back to balancing evenly.

Measured on 2.0.77 against a prototype declaring `variation_count = 94`:

| written | reads back |
|---|---|
| 94 | 94 |
| 95 | **1** |
| 255 | **67** |
| 0 | refused, `allowed values are from 1 to 256` |

so the engine wraps modulo the count and there is no way to store a byte it will not give back.

**The blueprint round trip keeps the flag with nothing asked of the guest.** A blueprint over a `simple-entity-with-force` with a `placeable_by` carries `variation = 50` and a revived ghost comes back at 50. Driven end to end on 2.0.77 through the remote method on a 2x2: the part at 10,10 flips and `restyle` writes variation **21 -> 68**, which is 21 + 47; off again writes **68 -> 21**, one teardown and one rebuild each way. A blueprint taken over the result carries **68** for the flagged part and 17, 27, 35 for the other three; pasted elsewhere and revived, the cluster comes up `skin cluster=5 parts=4 set=0 vars=68,27,17,35` -- `recoverPriority` found the world already showing what it wanted and made no write at all. **With `recoverPriority` stubbed out the same paste comes up `set=4 vars=21,...` and the flag is gone**, which is the defect it exists for.

`recoverPriority` (`skin.go`) reads the variation back only for parts whose picture this guest has never written, and takes ONLY the flag: a pasted part's neighbourhood is not its source's, so the shape is recomputed from the registry as always. One host call for the whole cluster through the bulk getter, skipped entirely when every candidate's picture is already known.

## The four refusals

`plan.ShapeEdges` answers one question, can this be built, and three bounds sit behind it; the fourth is the guest's own and is about the MOMENT rather than the shape. `refuseShape` (`limit.go`) is the one place a refused `Ports` chooses, and each refusal has one sentence for a toggle or a pair for a build, plus one log line.

| | when it fires | the toggle's key | the build's |
|---|---|---|---|
| over the port limit | more than `plan.MaxPorts` belts, whatever is flagged | `over-port-limit`, for a flag going ON | `over-port-limit` |
| too big for a priority port | the construction does not fit the slot, which is P = 64 | `priority-refused` | `priority-too-big` |
| a priority input | `QIn > 0` after the collapse, at ANY size | `priority-input-refused` | `priority-input` |
| too full to shrink | the successor holds fewer positions than the balancer is carrying | `priority-holding` | unreachable: a build cannot make a standing network smaller |

Each build key has a second with `-unconnected` on it, for the robot or script build that leaves the piece standing rather than handing it back, and `tellRefusal`'s hand-back line names the bound that fired for both.

**THE PORT CAP IS NAMED FIRST ON BOTH SIDES**, which `refuseShape` had the other way round: it dispatched on the flag, so a sixty-five-belt balancer with one flagged port was told about the priority port, and a player who took the flag off was refused again for a reason nobody had mentioned. A cluster can break more than one bound at once and the cap is the only one taking a flag off does not fix. Among the priority bounds a priority INPUT wins over a size, because there taking the flag off is the fix either way and it is the flag the player last touched.

**A TOGGLE IS REFUSED BEFORE THE FLAG MOVES.** `ShapeEdges` is asked with the flag speculatively flipped, so a shape the compiler could not build leaves the flag, the network and the items exactly as they were. Refusing after the flip would save the network too -- the check is in front of the teardown -- and would leave the cluster standing refused until the player guessed to toggle back, which is a worse answer to the same question.

**TAKING A FLAG OFF IS THE EXCEPTION, AND IT HAS TO BE, BECAUSE IT IS THE WAY OUT EVERY DOCUMENT NAMES.** The fit bound is about the SHAPE, so at P = 64 every q >= 1 is refused -- and a balancer that arrived there with two flags on it, by growing past P = 32 with them already set, could then not have either one taken off, each single un-flag landing on the q = 1 the bound refuses. It stayed refused for good. A toggle that turns a flag OFF is allowed whenever the shape before it moved did not fit either: the pre-toggle shape costs no host call to ask, and a change that cannot make things worse than they already are is not one to refuse. The spill guard does not run on that path, for the reason the refusal leaves the network alone -- nothing is torn down, so nothing can spill.

**TICKING EVERY PORT COLLAPSES RATHER THAN REFUSING.** `ShapeEdges` reports `QOut = 0` for M flagged outputs and `QIn = 0` for N flagged inputs, so a player who ticks all 64 outputs of a 64-port balancer gets the plain network to the byte instead of a refusal, and a balancer whose every input is flagged is not an input refusal at all. Two tiers where the second is empty is one tier.

**INPUT PRIORITY IS THE ONE THAT IS NOT ABOUT SIZE, and it is a decision.** The mirror construction is measured to be the same shape -- a de-concentrator, the tiers moved to the input side -- and what does not fit is the COMBINATION: input and output priority in one network is `DE-CONCENTRATE + BALANCE + CONCENTRATE` in series, 9k-3 columns, which is 33 at P=16 against a 32-column slot. An approximation would be the one outcome this repository's own rule forbids, a network that compiled, ran, balanced and quietly ignored the flag a player set.

## The spill guard, and the one number it protects

**A TOGGLE never puts anything on the ground**, which is the gesture the guard covers and the whole of what it covers. The fourth decision of "A recompile is not a removal" spills what a materially smaller successor cannot hold, which is right for a machine a player MINED and wrong for a flag they ticked. So `prioFitsWhatIsStanding` asks `plan.Reinsertable` what the successor could take back and what the predecessor could, and when the successor is smaller it counts what the standing network is holding and refuses the change rather than making it. Nothing is torn down, nothing is said in chat, and the player is told to let the balancer empty. **A flag arriving any other way is not a toggle and does not get this** -- a blueprint revive or a clone that raises one on a part whose cluster is then smaller takes the ordinary recompile-and-spill rule, because the BUILD it came in on is what recompiles the cluster and a build that shrinks a machine has spilled the difference since long before priorities existed.

The number it protects against is the last column of `agents/priority.md`'s table: **250 items on a 4->4 and 741 on an 8->8** at q = 1. It needs a balancer whose every output is blocked, and a jammed one really is near full.

**THE BOUND IS NOT THE TILE ARITHMETIC, and the first cut of the guard had that wrong.** Eight positions a belt tile, sixteen a splitter, is what the network holds when every line in it is compressed, and it is over what a real teardown gets back: a saturated dead-ended 4x4 drains **232 items** where the tiles say 288, and M2's full 2x2 drains **72** where they say 96. A guard on the arithmetic passes a toggle on a 2->2 holding anything from 65 to 96 items and spills the difference, with the refusal's own advice -- let it empty -- walking the player into the band rather than out of it. `plan.Reinsertable` is therefore the arithmetic less a third: under both measured fractions (75.0% and 80.6%) and under the pessimistic per-tile reading of them on every shape the planner builds, tightest at 1->4 with q=2 and 2.48 points to spare. Being under costs a refusal a player did not need; being over costs items on the floor.

**THE DIRECTION IS NOT THE QUESTION**, which is what the design had wrong and a test now holds. "Turning a priority port ON never spills" is true of the plain network against a priority one and false of q against q+1: a 4x4 takes back **442** items at q=1 and **362** at q=2, because one priority port leaves three normal ports and a square butterfly over three ports is four rows with a loopback in it, where two and two are a pair of single-splitter blocks. So a toggle ON shrinks that network, and the guard compares two capacities rather than asking which way the flag went (`TestAPriorityToggleCanShrinkTheNetworkInEitherDirection`).

**The count is items and the bound counts belt positions**, and under belt stacking those are not the same unit: a stacked position holds up to four items, so the count can exceed the positions occupied and the guard can refuse a change that would have fitted. That is the side to be wrong on, and it is the side the bound is on for every force in any case.

**The reading is the drain's own, made without draining**: the hidden slot's box and the visible cluster's box, through `sweep`'s four names and its prebuilt filter, one host call per transport line and nothing crossing the boundary. It runs on a keypress, only when the successor could take back less than the predecessor, and only when a network is standing. It is proportional to the network, though, and on the largest shape the fit rule allows that is a few thousand host calls -- 1,267 entities, a line count each and up to eight line reads -- which at the ~12.6 us this repo measures for a tier-2 call is the same order as the recompile it stands in front of. The product is not measured, and it belongs to the open decision above rather than beside it.

**WHAT IT DOES NOT COVER**, because neither is a flag changing: a belt MINED off a priority port shrinks the machine too and keeps the miner's-pocket contract, offering what will not fit to the miner before the floor; and a blueprint or paste that lands a flagged part into a balancer that is then too big is refused by `compile` in front of its own teardown, so there is nothing standing to lose. The only other writer of the flag is `recoverPriority`, which raises one on a part that has just arrived in the world with its belts, so the cluster it lands in is recompiled by the BUILD it came in on.

## The fit, and the bound that is the slot

| P | band 0 width | worst extent | fits |
|---|--:|--:|---|
| 2 | 5 | 6 x 3 | yes |
| 4 | 11 | 11 x 8 | yes |
| 8 | 17 | 17 x 16 | yes |
| 16 | 23 | 23 x 32 | yes |
| 32 | 29 | 29 x 64 | yes |
| **64** | **35** | **35 x 128** | **no** |

Against a 32 x 72 slot, so **a priority port is allowed at every size under P = 64 and refused at 64**, and the boundary is not a constant anybody chose: P=64 wants 35 columns of a 32-column slot and there is no packing of two 64-row blocks that is not also 128 rows of a 72-row slot. The tests measure the extent off the ops rather than off the rule and check it is exactly tight, because a loose bound would refuse shapes that fit.

**AN OPEN DECISION, NOT TAKEN HERE.** The fit rule admits a 32->32 with one priority port, which is **1,267 entities** -- larger than the 1,152-entity P=64 plain network `agents/maxports.md` measures at 155 ms empty and ~390 ms saturated for one teardown-and-rebuild. A recompile is linear in entities, so the largest network this mod would build under the shipped rule is one a belt edit rebuilds in about four hundred milliseconds, and the HITCH rather than the slot may be the bound that should refuse. The bound shipped is the slot. Deciding otherwise means measuring a priority recompile in a game first, which nothing has.

## The player's side

**ALT + P over the part under the cursor**, and the letter is the only part of that with a choice in it: a `--dump-data` of base, quality, elevated-rails and space-age on 2.0.77 has **14 custom inputs, twelve of them `ALT + <letter>`** over A B C D E F G L R T U Y, one on TAB and one unbound. P is free. The engine's own compiled-in bindings are not prototypes and no dump lists them, which is said in the prototype's comment rather than glossed.

Three doors reach `setPartPriority` and nothing else does: the keybind, `remote.call('better-belt-balancer', 'set-part-priority', surface, x, y, on)`, and a settings paste. The remote method is not a convenience -- a keypress cannot be issued from a script and a headless run has no player to press one, so without it the whole feature is reachable by a human and by nothing else, which is the condition `commands.go` exists to end.

**The settings paste takes the flag from the REGISTRY and clears `pvar`**, which is two statements for two things the engine did. Measured on 2.0.77, a script `copy_settings` between two of these entities moves `graphics_variation` itself, so the destination is wearing the SOURCE's shape on a neighbourhood that is not the source's; zeroing `pvar` is what makes `restyle` look rather than compare against a memory that is now wrong. Reading the flag back out of that variation would work today and would rest on an engine behaviour nobody promised.

**A REFUSED PASTE THEN HAS TO PUT THE PICTURE BACK**, and that is the price of the clearing. The flag is restored on every refusal, and the destination is still wearing the source's badged variation with `pvar` at 0 -- so the next restyle hands it to `recoverPriority`, which reads a badge as a flag and raises one with no guard and no message. The player is told the change did not happen and the flag arrives a flush later anyway, on a cluster the compiler then refuses: a hole through both the fit check and the spill guard. The handler writes the shape the registry says that part should draw, and `pvar` with it, in the dispatch that caused it.

**The editor's variation picker gains a second half, and it is not a harmless one.** M5 records that a variation picked by hand in the map editor persists until something changes the cluster's shape, because `restyle` compares its own answer against `pvar` rather than reading the entity. That is still true and it now has a way to be wrong that matters: a hand-picked cell in 48..94 is a priority badge. Within the session it is only a picture -- `recoverPriority` reads parts whose picture this guest has never written and an editor pick leaves `pvar` set -- **but `pvar` is guest heap, and the heap is declined on every rebuilt guest.** So the next release of this mod zeroes it for every part in the world, the rebuild-from-world hands every one of them to `recoverPriority`, and the hand-picked badge becomes a real priority port nobody asked for. An editor pick is a picture until the next update and a port afterwards, which is the direction to be wrong in for a session and the wrong one for a save.

**THE DUMP GOLDENS MOVE AND ARE NOT RE-CAPTURED HERE.** Two prototype changes reach the data stage: the custom input `bbb-toggle-priority`, which is a prototype like any other, and `variation_count` going 47 to 94 on `bbb-balancer-part`. Both are `data.raw`, so **both `data_raw_sha256` arms move on every engine and `mod_settings_sha256` does not** -- this feature declares no setting. A golden whose engine does not match the binary is a SKIP by construction, so the capture belonged to whoever next ran `test/check-datastage.py --capture` on a binary, beside the `prio` suite. **BOTH ENGINES ARE CAPTURED**: 2.0.77 on 2026-09-15 and 2.1.17 on 2026-09-17, each with master's own tree dumped first as the control and the move read out of a `jq -S` diff of the two normalised dumps -- twelve lines on each arm of each engine, the seven-line custom input and the one changed count, with the settings dumps byte-identical. The 2.1.16 rows are still owed and need a 2.1.16 binary.

## What is NOT done

- **Input priority.** Refused at every size, with its own sentence. The mirror construction and what building it would take are in `agents/priority.md`.
- ~~**Nothing has been run in Factorio.**~~ **DONE 2026-09-15**: the `prio` suite is the seventeenth, it is green in both `-gc` arms on Factorio 2.0.77 and, since 2026-09-17, on 2.1.17, and every rate in this section is now a measurement as well as the model's -- twenty rigs inside 0.002 of a belt of the two formulas, with a saturated 3 -> 2 drawing 870, 870, 870 from its three inputs at 0.00% spread and a side-loaded priority port carrying twelve items on each lane. Its section is under Verification and it is what the three holds in the table above come from.
- **The 94-cell sheet has not been measured at native scale in a graphical client.** `skin/sheet_test.go` reads the committed PNG's header and compares its cell count against `skin.Cells`, which is the class of defect a headless gate can catch; whether the badge reads as a badge at native scale is a human's. The mod's author reports exercising ALT + P and the badge in a graphical 2.0.77 client beside Vladance's mod on 2026-09-17, and both worked -- a report rather than a measurement, and the only look either has had.
- **Two of the three doors are behind the player wall, and that is measured rather than assumed.** A keybind cannot be issued from a script. And `LuaEntity::copy_settings` declares no `raises` in either pinned runtime description -- 89 methods there do and this is not one of them -- while `on_entity_settings_pasted` carries a mandatory `player_index`, so a scripted copy moves the destination's `graphics_variation` and tells this guest nothing at all. The `prio` suite asserts that outcome and `onSettingsPasted`'s own refusal path is the interactive checklist's.
- **The recompile hitch at the fit rule's ceiling**, above.
