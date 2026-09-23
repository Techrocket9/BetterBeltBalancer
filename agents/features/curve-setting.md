# The curve rule is a setting — `bbb-curved-exits`, on by default

**The curve rule changes what a STANDING factory means, so it is behind a map setting a player can turn off.** A belt line that merely started beside a balancer part was nothing at all before 0.3.3 -- the incumbent's accepted limitation, inherited -- and is that part's output now. So an update gives somebody ports they never laid, and on 2.1 a part that was already serving a belt is then asked for a second one and the whole balancer is refused until the line is taken away. That is a real cost paid by a player who was not asking for the feature; `bbb-curved-exits` is what they turn off, and with it off such a belt is left unconnected exactly as it was.

**THE DEFAULT IS THE NEW BEHAVIOUR, AND THE FIRST LOAD OF AN OLDER SAVE TURNS IT OFF.** The feature was asked for, the one shape it declines is the one that would half-fill a port, and a default that shipped off would be a feature nobody found -- so a new save gets the rule. A save that predates it does not, and which saves those are is read off the STATE VERSION FkLua stamps into every save rather than off any shape in the world: such a load decides at the first curve edge it classifies, reads the rest of the save with the rule off, writes `bbb-curved-exits = false` for it and says so once per force with a ping per balancer. So the DEFAULT is what a player chooses into and what a fresh world gets, and an existing factory is left exactly as it was until its owner asks otherwise. "A save from before the curve rule keeps the reading it was built to" is that pass, and it carries what happens without it -- including the 2.0 case where a curve turned one balancer into a multi-edge one and tripped the OTHER rule's grandfather.

`guest/go/curve.go` is the policy and its header is the long form; `guest/go/curveupg.go` is the upgrade; `guest/go/globalsetting.go` is the mechanics it shares with `bbb-multi-edge-parts`.

## Where it differs from the mod's other runtime-global

| | `bbb-multi-edge-parts` | **`bbb-curved-exits`** |
|---|---|---|
| defined on | 2.0.x only | **both engines** |
| default | false | **true** |
| what it is about | what the ENGINE permits, which 2.1 changed | what THIS MOD decides a belt at a face means |
| the write's gate | the `bbb-can-stack` marker, because writing an undefined key raises | **none, and none is needed** |
| a flip with balancers standing | VETOED, and the setting goes straight back on | **honoured**, and the save re-classifies |
| what the first load of an older save does | writes it ON, so the save keeps working | **writes it OFF**, so the save keeps working |
| driveable by a suite | on 2.0 alone | **on either**, which `m2` uses |

**HAND-ROLLED RATHER THAN DECLARED THROUGH FkRecipes, AND THAT IS FORCED.** The library has a `LegacyBoolSetting` that would carry the name across verbatim, and it emits `setting_type = "startup"` for every setting it declares (`go/settings.go`) with no way to ask for another kind. A startup setting can be neither flipped mid-save nor written by a script, so it is not this setting. It is declared beside the other hand-rolled bool in `guest/go/data/settings.go`, BEFORE that file's engine gate rather than after it, and `guest/go/tune`'s `HandRolledSettings` is what tells the library's locale checker that both exist.

**THE CONTROL GUEST MAY NOT IMPORT `guest/go/tune`, MEASURED.** That package imports FkRecipes, so a control guest that imported it would link the data stage's emit layer as well and the two `main` packages' exports collide: `tinygo build` dies on wasm-opt's `parse exception: duplicate export name`. So the name is a literal in `curve.go` exactly as `bbb-multi-edge-parts`' is in `sedge.go`, and `tune`'s copy is the one the prototype and the locale checker use. What that leaves open is stated rather than papered over: a rename that missed `curve.go` would leave the guest reading a setting nobody defines, which reads Absent, which is answered ON, which is the shipped default and therefore silent. There is no import that can close it.

## The read, and what a disabled rule costs

**A PER-HEAP TRI-STATE, AND `classifySide` ASKS IT BEFORE IT PROBES.** `curvedExitsAllowed()` is two host calls on the first call of a heap and one integer compare after that, thrown away by the three load hooks and by the setting-changed handler -- which are between them every moment the answer can move. The gate sits in front of `curvesFromCluster`, so a save that has turned the rule off makes NEITHER of its two `find_entities_filtered` calls and pays exactly what this guest paid before the rule existed.

**That is measured rather than argued, and leg H is where.** The `mar` suite's leg H is the only leg with a perpendicular-adjacent belt, and it is **384 B per iteration with the rule on and 384 B with it off** -- the same byte-identical figure the arm's own red proof produced, because the arm allocates nothing either way. The seven standing slopes are unmoved.

**A SETTING THIS GUEST CANNOT READ IS ON**, and the two directions are not symmetric: answering off would silently disable a rule a player's factory is built on, where answering on is the shipped default. It is also what a 0.3.2 save produces by a different route -- the engine supplies a prototype's default for a key `mod-settings.dat` has no entry for, so such a save reads true on its first load here and is classified under the new rule with nothing extra to do. **Observed rather than inferred**: only one leg of one suite writes a `mod-settings.dat` -- `curv`'s second, which is what makes it a save that was already decided -- so every other run in the estate is on exactly that state, and `m2`'s curve rig compiles at load and `sedge`'s `scrv` refusal fires, both of which need this answer to be ON.

## The flip, which re-classifies the save

**EVERY CLUSTER IS RE-QUEUED AND ALMOST ALL OF THEM SKIP.** The rule decides which belts are EDGES, so what a flip moves is the edge list of any cluster with a belt across one of its faces and nothing else. The handler does not try to work out which those are: it hands the whole save to the same `requeueEveryCluster` the grandfather pass uses, affordable for the same reason -- this is a keypress rather than a tick -- and every cluster whose edge list did not move skips on the fingerprint it never lost. One line says so:

    [BBB] curved exits: a belt across a balancer's face is left unconnected;
    23 clusters re-queued, and every one whose edges did not move skips

**A CLUSTER MAY END UP WITH NO NETWORK, AND THAT IS THE HONEST OUTCOME RATHER THAN A HOLE.** A balancer whose only outputs were curves has inputs and nothing else once the rule is off, which is a legitimate half-built state: `plan.Build` declines it, the audit counts it in neither `unbuilt` nor `refused`, and what its network was holding is spilled beside it by the ordinary teardown -- because a machine that no longer exists is a REMOVAL, and this mod's rule for a removal's items is that they go back to the world. Turning the rule on again rebuilds it.

**NOTHING FLUSHES IN THE HANDLER**, for the multi-edge handler's reason: the remote method's write is dispatched at the outermost level, but a player's keypress can arrive anywhere, so it queues and asks for the next tick's flush exactly as an ordinary event does. And there is no ANCHOR here, unlike the multi-edge fold: the handler reads the cached value BEFORE invalidating it, so a write of the value already there -- which Factorio raises this event for -- costs a re-read and nothing else.

**`set-curved-exits` JOINS THE REMOTE INTERFACE**, for `set-multi-edge-parts`' reason and one more. The reason is that Factorio refuses `settings.global[k] = v` from anybody but the mod that DEFINED the setting and a runtime-global has no owning player, so without the method the flip handler is reachable by a human and by nothing else. The one more is that this setting exists on both engines, so it is the first method there that a suite can drive on the engine trunk targets. `edge` asserts the method list as an exact SET and it is `{audit, set-multi-edge-parts, set-curved-exits}` (plus `set-part-priority` since the priorities feature, which is the fourth method and the only one that is not a setting).

## What the `m2` suite measures

The band runs AFTER everything else has reported, so no figure that suite has ever recorded is taken over a save whose classification rule has moved. Measured 2026-09-07 on Factorio 2.0.77 over the twenty-three-cluster save; the control belt delivered 300 items in each 400-tick window:

| | with the rule OFF | with it back ON |
|---|---|---|
| the `curve` rig's two corners | **16 and 19 items** -- 0.053x and 0.063x of one belt | **300 and 300** -- 1.000x each, **0.00% spread** |
| the audit | **`clusters=23 parts=163 nets=22 drift=0 unbuilt=0 refused=0`** | **`nets=23`**, and the rest unmoved |
| `sat4` beside it | 4.000x | 4.000x |
| what the guest did | **1 teardown, 1 spill, 0 compiles** | **1 compile**, of that one cluster |

**The sixteen and nineteen are not the rule and they are not zero**, which is worth saying because the design asked for a zero. The interface is gone the tick the flush lands, and what was already standing on the two corner runs walks into the chests over the next few seconds. A sixth of a belt is the bound, which separates draining-out from still-a-port by a factor of six in both directions.

**THE ON WINDOW OPENS 210 TICKS AFTER THE REBUILD**, which is `plat`'s rule for the same reason: a rebuild puts every drained item back at the HEAD of the butterfly, so the outputs are starved by construction until the pipeline refills. Measured at 20 ticks the two corners deliver 0.927x and 0.923x, which is a statement about the window rather than about the ports.

**ONE TEARDOWN, ONE SPILL AND ONE COMPILE OVER THE WHOLE BAND, AND EACH OF THE THREE IS ONE FOR A DIFFERENT REASON.** One teardown because `curve` is the only cluster in the save whose edge list the rule moves. One spill because that cluster has no successor to reinsert into -- 24 items drained and 24 spilled, beside the same cluster. One compile, on the way back, and none on the way out. `assert-m2.py` asserts all three as exact counts, so a flip that quietly recompiled the save would fail as loudly as one that recompiled nothing.

## Red-proven twice, and the two catch different things

| injected defect | what fired |
|---|---|
| **the setting read stubbed to always-true**, so the policy is never consulted | **seven assertions**: `the guest logged 0 curved-exit flips and the observer made 2`, the band's teardown/spill/compile counts all at 0 against 1, the OFF audit reading `nets=23` where it should read 22, and both corners taking **300 items (1.000 of a belt)** over the OFF window -- named one per corner, because a port that is still running is the whole defect |
| **the handler's re-queue removed**, so the flip lands and nothing is re-classified | **six**, and the shape is the defect exactly: the flip is accepted and `settings.global` reads false, `a curved-exit flip re-queued 0 clusters and the save holds 23` twice, and the OFF audit at **`drift=1`** rather than one network short -- the classification has moved and the standing network does not match it. The audit's own flush is what eventually re-queues the cluster, one audit late, which leaves the ON side at **`nets=22 unbuilt=1`** and fails the final-audit assertion that has been in this suite since M1 |

## What it costs to ship

Both packages from `make clean`, 2026-09-07, shipped config (`--persist=packed --gc=collected`), the 2.1.17 pin, same FkLua, same machine:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.3.zip` | 671,065 B | **673,187 B** | +0.32% |
| `fk_module.lua` (the control guest) | 3,232,716 B | **3,255,399 B** | +0.70% |
| `fk_data_module.lua` (the data guest) | 3,129,350 B | **3,152,767 B** | +0.75% |
| `dist/bbb.wasm` | 1,325,076 B | 1,330,155 B | |
| `dist/bbbdata.wasm` | 574,508 B | 575,837 B | |
| members bound into the mod | 57 | **57** | of 4,870, **none added** |
| events subscribed / defines read | 24 / 4 | 24 / 4 | unmoved |

121 bytes of the zip's delta is the changelog entry, which is in the package. **`fk_api_gen.lua` is BYTE-IDENTICAL either side**, which is the strongest form of "no new member": the same 57 members, at the same ids. Everything the setting needs was already bound for the multi-edge one, and the remote method is a third id in a switch this guest already had.

**`mod_settings_sha256` MOVES AND `data_raw_sha256` DOES NOT**, on both mod sets and on every engine. A setting PROTOTYPE lives in the settings stage's own Lua state, so it reaches `mod-settings-dump.json` and never `data-raw-dump.json`, and the engine's own prototype list checksum does not move either. Re-captured on the 2.0 arm, which is where a 2.0 golden can be taken: **`196275f867f7f8b5` -> `cc79d368f2711dd6`** on `base` and on `incumbent` alike -- one hash, because a settings dump carries no per-mod-set content -- with `data_raw` at `414aa4796d21afaf` and `f67c4bf8544a9010` unmoved and the checksums at 3427049257 and 223071962 unmoved. That arm now carries FOUR settings, two startup cost dropdowns and two runtime-global bools; the 2.1 arm carries three.
