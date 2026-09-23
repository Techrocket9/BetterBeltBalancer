# `mig` -- `test/assert-mig.py`, a save that used to be somebody else's

The ninth suite, **seven legs and two name probes**, and **the only one whose two phases run under different mod sets** -- a mod is installed or uninstalled between `--create` and `--benchmark`. Its rigs, its numbers, its seventeen red proofs and the run against the real Belt Balancer 2 are in "Adopting a Belt Balancer 2 or 3 save", which is where the whole feature lives; what is worth knowing here is that `guest/go/obs/bb2data` is a DATA-STAGE-ONLY stand-in carrying the real mod's own name and version, so `script.active_mods` sees what it would really see -- and that it is staged under **all four** of `legacyIncumbents`' names, by copying the one package and rewriting its `info.json` at staging time.

**The legs cover TWO AXES and neither is a subset of the other**: WHICH MOD owns `balancer-part` (four incumbent names and a stranger who is none of them), and WHICH TRANSITION of `legacy.go`'s state machine the load makes. The second axis is the one that was empty: until 2026-08-20 only `Blocked -> Done` by removal was ever driven, so the `Done -> Blocked` recheck that `fk_on_configuration_changed` exists for had no test at all, and neither did the promise `legacyCheck` makes to a stranger in as many words. `readd` and `fgone` are those two. **Both axes survived the single-edge port untouched**, because the state machine knows nothing about belts.

**AND SINCE 2026-08-24 THE OUTCOME IS SPLIT, WHICH IS THE PORT.** This is the one suite whose rigs were deliberately NOT re-laid one belt per part: its world is the INCUMBENT'S, and Belt Balancer's idiom is a single column of parts with a belt on every free face, so re-laying them would have been re-laying the thing under test. They convert exactly as they always did and are then REFUSED, and what was added is the **`sok` band** -- the same balancer laid two columns wide, which is a shape one of their users could genuinely have -- which converts into a network that runs. Nine clusters over thirty-one parts, and one audit line carries the whole answer, identically in all five conversion legs:

    clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7

Measured 2026-08-24, Factorio 2.1.14, base plus the quality mod. One saturated express belt delivered **1,306 items** over t=1800..3540:

| rig | laid | what it is on 2.1 |
|---|---|---|
| `m4x4` | the incumbent's way, 4 parts | refused. **0 0 0 0** |
| `m3to5` | the incumbent's way, 5 parts | refused. **0 0 0 0 0** |
| **`sok2`** | **two columns, 4 parts** | **1306 1306 -- 2.000x one belt, spread 0.00%** |
| **`sok4`** | **two columns, 8 parts, P=4** | **1304 1306 1306 1304 -- 3.997x, spread 0.15%** |

**The zeros are asserted as zeros rather than as a loosened bound**, which is the discipline the rest of the estate kept: a refused cluster has no network at all, so an item in one of those chests would be a balancer that got built when the rule says it cannot be. And **nothing is torn down and nothing is spilled**, which is where this suite asserts the opposite of `mig21`'s: there the balancers were STANDING when the save opened, so the remnant had to come down and everything it held reached the ground; here the clusters are seconds old, `hadNet` is false, and there is no teardown for the refusal to be in front of. The items are where they always were -- on the player's own belts, which is what the copper witness measures from the other side, at **48 at every one of four samples**.

**The refusal SHAPE is a multiset**, `[2, 2, 2, 2, 2, 3, 4]` parts carrying more than one belt, and `m3to5` is the row that makes it a statement about a classification rather than about a constant: three inputs and five outputs over five parts means three of its parts carry two belts and two carry one. Nothing else in the world has a count that is neither zero nor its whole size.

**And the `added` leg's rebuild-from-world ADOPTS the two working clusters beside the seven it refuses** -- `9 clusters, 2 adopted, 7 rebuilt` -- which closes a gap [`agents/single-edge.md`](../../single-edge.md)'s phase-2 section records against `mig21` and could not close there: neither committed fixture has a single-edge cluster in it.

**The one defect this pass found and did not fix is FIXED, 2026-08-24.** A converted-and-refused balancer used to be announced with the ORDINARY per-piece message -- the one that says the extra piece was left in place unconnected, when nobody placed anything -- and the migration summary with its GPS checklist arrived only in the `added` leg, where a rebuild-from-world happens to follow the conversion. **A legacy conversion is the THIRD PRODUCER of that summary now**, so every conversion shape speaks the checklist once, per force, and no leg speaks the per-piece copy at all: `told per cluster: 0` and one summary in all six. `EXPECT_SUMMARY` moved from a measurement of the defect to a statement of the rule, and `EXPECT_TOLDPIECE = 0` is the half with teeth. The whole pass -- including the second false sentence it found on the way, and the 2.0 grandfather arm it settled -- is [`agents/single-edge.md`](../../single-edge.md)'s phase-8 section.
