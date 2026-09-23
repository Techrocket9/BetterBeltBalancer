# Adopting a Belt Balancer 2 or 3 save — the migration

**The rule, in one sentence: while an incumbent is INSTALLED this mod never touches its entities, and once it is GONE this mod converts every `balancer-part` left standing into one of its own, once per save.** Requested 2026-08-16 in exactly those terms; `guest/go/legacy.go` and `guest/go/data/legacy.go` are the two halves and their headers are the long form.

**The incumbents are FOUR NAMES and one prototype.** `belt-balancer` (the original), `belt-balancer-performance`, `belt-balancer-2` and `belt-balancer-3` all define a `simple-entity-with-force` called **`balancer-part`** and an `item` of the same name that places it -- the original was renamed into that name by Belt Balancer 2.1.0's own `migrations/2020-02-28_Belt-balancer_2.1.0.json`. All four declare `!` conflicts against each other, so **at most one is ever active**, and their I/O model is this mod's: adjacent belts, orientation decides in from out. That is what makes adoption possible at all -- the world already says what each balancer's ports are, and `classifyEdges` re-derives them the way it does after any other edit.

## Why NOT a `migrations/*.json` rename, which is the obvious answer

**A prototype-migration file is applied ONCE PER SAVE PER FILE and the engine remembers by FILE NAME** (<https://lua-api.factorio.com/latest/auxiliary/migrations.html>). And when a mod is ADDED to an existing save, all of its migrations RUN on that first load -- measured on 2.1.16, not merely recorded as this file used to claim -- which, for the player this feature is for, is the load where the incumbent is still present and its balancers must not be touched: the rename would actively execute at exactly the wrong moment, and its filename is then burned so it could never fire again on the later load where the incumbent is gone. It also has no way to express "do nothing while that other mod is installed", and what a rename does when BOTH the old and the new prototype exist is undocumented. The decision is taken at RUNTIME, from `script.active_mods` and a marker prototype, so it belongs in the guest.

## The two halves, and neither works alone

**The DATA half stops the engine deleting the evidence.** When a mod is removed, Factorio deletes every entity whose prototype went with it, **at load, before any script runs** -- so without a prototype of the same name and a compatible type there is nothing left for a guest to find. The data guest exports `fk_data_final_fixes`, and `guest/go/data/legacy.go` is that hook: it defines a stub `balancer-part` entity, a stub `balancer-part` item and a marker prototype **only when nobody else has**. `data-final-fixes` is the stage that can see that: it runs after every mod's data and data-updates stages, so an incumbent that is still installed has already defined its own and this file does nothing at all. **The "leave it alone while it is installed" half is enforced by the engine's own load order rather than by a list of mod names.**

**The RUNTIME half decides WHOSE the prototype is**, because a prototype existing says nothing about that. `guest/go/legacy.go` reads `script.active_mods` for the four names, and `prototypes.entity["bbb-legacy-stub"]` for the marker the data half defines **if and only if** it defined the stub. The marker is the guard with the blast radius: without it a mod nobody has heard of that happens to define `balancer-part` would have its entities eaten. See the red proof below, which is exactly that.

**Two things about the stub prototype are decisions rather than copies:**

- **`placeable_by = { item = "bbb-balancer-part" }` and `minable.result = "bbb-balancer-part"`**, so a player who mines a stub and a robot that revives a ghost of one both end up holding THIS mod's item. That is what makes a migrating player's **blueprint book** keep working: every blueprint they took names `balancer-part`, its ghosts ask for our item, and the part the robot builds is queued by `legacyBuilt` and swapped by the next flush.
- **`not-blueprintable` is deliberately ABSENT.** A player cannot get a stub out of nothing and no flag is what stops them: there is no recipe, no technology, and the item is hidden, so the only way to hold a `balancer-part` is to have held one when the incumbent left. That is structural and stronger than a flag -- and the flag would break the one path above in exchange for refusing a capture of a prototype that exists for at most one load. `not-upgradable` IS set: there is no upgrade path onto or off it and there must not be one.
- **The stub item's `place_result` is the STUB ENTITY, not `bbb-balancer-part`**, since 0.3.3, and it is the only field of that prototype which is not simply the incumbent's own. Pointing it at this mod's part put the two names into each other's `items_to_place_this` and hung a player's game. See "The two-element cycle that froze a game" below.

**And the item is not renamed, which is a decision too.** A stack of the incumbent's part in a chest, a hand, a logistic request or a bot survives because the prototype survives, and it goes on placing `balancer-part` -- which is the stub now, and which the next flush swaps for one of this mod's parts. Walking every inventory in the game to rewrite stacks would be a scan of the whole world for a cosmetic difference in what a stack is called.

**The locale entry for `balancer-part` is the INCUMBENT'S OWN ENGLISH TEXT, verbatim** ("Balancer Part"), and only in `en`. This mod loads after both Belt Balancer 2 and 3, so a different string would rename a working mod's entity in its own player's UI; identical text makes the override a no-op, and shipping no non-English file leaves every translation they have winning by fallback.

## The state machine, and the re-arm is why it is a machine and not a bool

`legacyPhase` lives in the guest heap and therefore in the save. Its zero value is `legacyUnchecked`, which is what a fresh heap must mean.

| state | what it means |
|---|---|
| `legacyUnchecked` | nothing decided on this heap. The gate every event passes is one compare against this |
| `legacyDone` | this game's `balancer-part` is ours (or there is none) and the scan has run |
| `legacyBlocked` | an incumbent is active, or a STRANGER owns `balancer-part`, or the mod list could not be read. Nothing is converted and nothing is touched, **including by the build path** |

| trigger | what it does | reported as |
|---|---|---|
| `fk_on_init` | a new save, **or this mod added to one that already exists** -- the common half of the swap | `trigger=init` |
| **`fk_on_configuration_changed`** | the MOD SET moved: a neighbour added, **removed**, or re-versioned. **It also fires on the load that ADDS this mod, right after `fk_on_init`** (a newly added mod is itself a mod-set change; measured in the `mig` suite's `added` leg, where the guest's `rebuilt from world` line lands before tick 0), so that load decides twice: `init` converts, and the second decision finds nothing and says nothing -- one extra by-name scan per surface, once | `trigger=configuration_changed` |
| `fk_migrate` | a rebuilt guest, on a fresh heap | `trigger=migrate` |
| `onEventBody`, after `ensureRegistry` | a fresh heap that reached an event before any hook | `trigger=first-dispatch` |
| the tail of `fk_on_deferred` | re-tests a **Blocked** state only | `trigger=deferred` |
| the top of `flush()` | not a decision at all: it swaps the stubs a BUILD EVENT queued | one line per stub |
| `auditAll` | the same, and it is what makes `/bbb-audit` a door onto the conversion | `trigger=audit` |

**The re-arm is the case a bool cannot serve.** A player can install this mod first, install the incumbent second, build fifty balancers with it, and remove it a month later; a save that had latched "scan done" would never look again. `fk_on_configuration_changed` is the trigger that case deserves and it is upstream's, landed 2026-08-16 for this shape ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 22): it is exactly the event that reports a neighbour being uninstalled, it is REPLICATED (it runs on the peer that loaded, before the first tick, so the conversion is already inside the state a joining client downloads), and it fires AFTER `fk_migrate` on a load that is both. It is the opposite side of the peer-local rule from `fk_after_load`, which this mod must never export.

**The deferred-flush re-arm behind it is belt and braces and is kept because it is free.** The test there is the MARKER PROTOTYPE and not `script.active_mods`: `prototypes.entity` is a `LuaCustomTable`, so the raw handle plus its index operator is a POINT query returning an Object or nil -- **two host calls and zero allocation**, where reading the mod list allocates a Go string per mod. `script.active_mods` is reached once per decision, never from a re-test, and it earns its place by NAMING the mod in the log line.

**Both gates were measured before either was written**, on 2.0.77 rather than assumed, because the design turned on them:

| probe | result |
|---|---|
| `script.on_event(id, f, {{filter="name", name=<a prototype that does not exist>}})` | **accepted**, no error. So `balancer-part` is in this mod's ordinary subscription filter unconditionally, and there is no branch on the mod set at subscribe time |
| `prototypes.entity[<a name that does not exist>]` | **nil**, and raises nothing. So the marker test is a value read rather than a guarded call |
| `find_entities_filtered{name=<a name that does not exist>}` | **RAISES**. Which is why the test mod guards its census and why the scan runs only when the marker says the prototype is ours |
| `mods[...]` at runtime | **nil** -- it is a data-stage global. `script.active_mods` is the runtime form and it carries versions |
| `find_entities_filtered` and `technologies[...].researched = true` from `on_init` | both legal |

## What the conversion does per part

Read position, force index, **quality** (as the prototype HANDLE, since `QualityID` is `string or LuaQualityPrototype` -- one host call and no string copied into the guest heap) and **health**; `destroy` with **no** `raise_destroy`; `create_entity` with the same position, force and quality; copy the health back; and `AddPart` straight into the registry, which is the same call a build event makes. So the clusters form, merge and queue themselves through the one code path and the flush restyles and compiles them exactly as it would after a blueprint paste.

**Read everything before destroying anything**, because after the destroy there is nothing left to ask -- and the order is forced the other way round by the collision box, since both prototypes collide on `object` and `transport_belt`.

**Health is preserved rather than reset**, at two host calls per part on a once-per-save path, because both prototypes carry `max_health = 170` and the alternative is silently repairing a building the player did not ask us to touch.

**Surfaces in INDEX ORDER**, through the same `collectSurfaces` `rebuildFromWorld` uses -- factored out of it for this, which is the one shared edit this pass makes to `lifecycle.go`. Registration order decides node ids, node ids decide cluster roots, roots decide hidden-surface slots: two clients walking surfaces in two orders would place one network in two different slots, which is a desync. `legacyIncumbents` is walked in ITS OWN order too, because `ActiveMods` is a `pairs()` walk over a Lua hash table.

**And the scan FLUSHES before it returns.** The ordinary build path defers to the next tick and the triggers that matter here cannot wait for one: a `--create` never reaches a tick at all, and `fk_on_init` on a save this mod was just added to is exactly that case. So it ends where the audit marker ends, with the networks built inside the same dispatch.

**The technology, per force that owned a converted part.** The incumbent's three technologies die with it, and a player left holding fifty balancers and no way to craft a replacement part would have been given a worse save than they started with. A point query again, and deliberately silent about a force that already has it and about a game where an overhaul mod removed it.

## What survives and what does not

| | |
|---|---|
| the balancers | as this mod's, at the same tiles, the same force, the same quality and the same health, compiled into networks on the load that adopts them |
| every item ON THE BELTS | untouched: those belts are vanilla and nothing here reads or writes them |
| every input and output | re-derived from the world by `classifyEdges`, so the ports are the ports the belts already implied |
| a stack of the incumbent's ITEM | survives, keeps its name, and places the STUB -- which the next flush swaps for one of this mod's parts, the same tick of latency an ordinary part placement has. It stopped placing this mod's part directly in 0.3.3 and the reason is a hang: "The two-element cycle that froze a game" |
| a BLUEPRINT BOOK of the incumbent's balancers | keeps working: ghosts ask for this mod's item, and each part a robot builds is swapped by the next flush |
| the ability to craft | any force that owned a balancer is given `bbb-balancer` |
| **the incumbent's own item BUFFER** | **gone, and there is no mechanism that could get it back** |

**The buffer, precisely, because "some items are lost" is not good enough.** `objects/balancer.lua` in Belt Balancer 2 takes items OFF the belts with `lane.remove_item(item)` and holds them in `balancer.buffer`, a plain Lua array in the mod's own `storage`, refilled each tick to `output_lane_count * 2`. So a 4-output balancer holds up to **16 items** that are not in the world at all, and Factorio deletes a removed mod's `storage` **with the mod**, before any script of ours could read it. A handful of items per balancer, stated in the README rather than glossed.

## The `mig` suite — seven legs, two name probes, and two axes

The **ninth** suite, and the only one whose two phases run under **different mod sets**: `test/run.sh`'s `stage_mig` and its `BETWEEN` hooks rewrite `mod-list.json` and add or delete a mod DIRECTORY between `--create` and `--benchmark` (a directory that is present but not listed is added back by Factorio as enabled, so "removed" has to mean both).

**The suite covers two axes and neither is a subset of the other**: WHICH MOD owns `balancer-part`, and WHICH TRANSITION of the state machine above the load makes. Until 2026-08-20 the first axis was one name of five and the second was one transition of several — only `belt-balancer-2` had ever been in front of the guest, and only `Blocked -> Done` by removal was ever driven.

| | the legs |
|---|---|
| **which mod** | `belt-balancer-2` (legs 1, 2, 5), `belt-balancer-3` (leg 3), `belt-balancer` and `belt-balancer-performance` (the two probes), a STRANGER who is none of them (legs 6, 7), and nobody at all (leg 4) |
| **which transition** | `Unchecked -> Done` on a new save (1), `Blocked -> Done` when an incumbent is removed (2, 3), Done with nothing to find and the build path doing the work (4), **`Done -> Blocked` when an incumbent ARRIVES (5)**, Blocked and staying Blocked (6), **Blocked -> Done when the STRANGER is removed (7)** |

`guest/go/obs/bb2data` is a **DATA-STAGE-ONLY stand-in** under the real mod's own name and version, so `script.active_mods` sees what it would really see. It has no control stage AT ALL -- not an inert one, none: `fklua mod --data-module` with no control positional, which is what it always wanted and could not have until 2026-08-25 ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 26). That is deliberate rather than incidental: the real mod's runtime is the one thing the migration cannot recover, so a stand-in that balanced would be modelling it. None of its art is used.

**IT IS STAGED UNDER ALL FOUR INCUMBENT NAMES AND THERE IS ONE COPY OF IT.** What differs between the four rows of `legacyIncumbents` is the NAME and nothing else — the prototypes are the same prototypes — so `mig_standin` copies the one directory and rewrites `info.json`'s `name` and `version` at staging time, into a directory named for the target mod (Factorio requires that). The rewrite is checked by two greps, because a silently unrenamed copy would stage belt-balancer-2 under every name and pass every leg.

`guest/go/obs/mig` -- a compiled observer since 2026-08-25, `test/mods/bbb-mig-test/control.lua` before that -- is present in BOTH phases and builds everything in `fk_on_init` with `balancer-part`, which in phase one is the only balancer prototype in the game. Six rigs, and the last three are about what the conversion CARRIES rather than about a rate:

| rig | what it is |
|---|---|
| `ctrl` | a bare express belt, the yardstick |
| `m4x4` | 4 parts, 4 in, 4 out, saturated -- the shape a migrating player is most likely to have |
| `m3to5` | P=8 with loopbacks. Adoption re-derives the edge list, and an asymmetric shape is where a wrong one reads as a rate rather than as a crash |
| `wit` | **the conservation witness**: 2 parts, 2 in and 2 out, **no source and no sink at all**, its belts hand-loaded with **COPPER PLATE** while every other rig runs iron. A copper count across every surface is therefore exactly that rig's contents, before the swap and after it, which is what makes "the items on the belts survived" an equality rather than an estimate |
| `fid` | **the fidelity pair**: one part DAMAGED to 85 of 170 and one built at UNCOMMON quality. Those are the only two properties `legacyConvertOne` reads off the old entity and writes onto the new one, and both are invisible on an undamaged normal-quality part |
| `frc` | **the force column**: four parts in one column, the top two on the player force and the bottom two on a second force, TOUCHING. Two forces' parts touching are two balancers |
| **`sok2`, `sok4`** | **THE SINGLE-EDGE BAND, added 2026-08-24**: the same balancer laid TWO COLUMNS wide -- inputs down the west column, outputs down the east, one belt per part -- at 2 in / 2 out over four parts and 4 in / 4 out over eight. A shape a Belt Balancer user could genuinely have, and the only rigs in this suite that Factorio 2.1 can build |

Plus **a second surface**, `bbb-mig-b`, carrying two more parts and their belts; and a steel chest holding 50 of the incumbent's item.

**Every rig this suite adds carries a belt in and a belt out per part, fed or not**, and that is a constraint rather than decoration: a cluster with no inputs or no outputs compiles to nothing, which is a legitimate half-built state, and a cluster the classifier never saw would then be invisible.

**EVERY RIG BUT THE `sok` BAND IS LAID THE INCUMBENT'S WAY AND THAT IS DELIBERATE.** One column of parts with a belt on both free faces of each row is two belts per part, which is what Belt Balancer builds and what Factorio 2.1 forbids -- so on 2.1 all seven of those clusters convert and are then REFUSED. This is the one suite in the estate whose rigs were NOT re-laid for the port, because its world is somebody else's and re-laying it would have been re-laying the thing under test. See the `mig` entry under Verification for the split outcome and its numbers.

**Thirty-one parts, nine clusters, three surfaces, two forces**, and the whole suite is measured against that world.

**The three conversion legs, measured** — re-measured 2026-08-24 against the eight-rig single-edge-band world on Factorio **2.1.14**, base plus the `quality` mod, shipped configuration. **Every conversion row below is what it always was and only the OUTCOME rows moved**, which is the whole finding of the port: `legacy.go` was not touched: 

| | leg 1 `added` | leg 2 `later` | leg 3 `bb3` |
|---|---|---|---|
| the incumbent | `belt-balancer-2 2.0.9` | the same | **`belt-balancer-3 1.0.1`** |
| phase one | the incumbent alone, this mod absent | the incumbent AND this mod | the incumbent AND this mod |
| between the phases | incumbent out, this mod in | incumbent out | incumbent out |
| coexistence in phase one | n/a | **exactly one** `belt-balancer-2 2.0.9 is active; its balancers are left alone`, **0 converted** | **exactly one**, and it NAMES `belt-balancer-3 1.0.1`, **0 converted** |
| parts adopted | **31 from 3 surfaces into 9 clusters** | the same | the same |
| trigger | **`init`** | **`configuration_changed`** | **`configuration_changed`** |
| forces given the technology | **2** | 2 | 2 |
| census, before -> after | 31 / 0 -> **0 / 31** | the same | the same |
| the witness's copper | **48 -> 48 -> 48 -> 48** over four samples | the same | the same |
| the item stack | **50 held**, `place_result` `balancer-part` -> `bbb-balancer-part` | the same | the same |
| technology after | `bbb-balancer=true`, `belt-balancer-1=absent` | the same | the same |
| `ctrl` over t=1800..3540 | 1306 items | 1306 | 1306 |
| **`sok2`** | **1306 1306, 2.000x**, spread **0.00%** | identical | identical |
| **`sok4`** | **1304 1306 1306 1304, 3.997x**, spread **0.15%** | identical | identical |
| `m4x4`, `m3to5` | **0 0 0 0 and 0 0 0 0 0** -- refused, and a refused cluster has no network | identical | identical |
| refused | **7 clusters**, parts-carrying-two-belts **[2, 2, 2, 2, 2, 3, 4]** | identical | identical |
| the fidelity pair | **85.0 of 170.0 health** and quality **uncommon**, both on a `bbb-balancer-part` at t1 and again at final | the same | the same |
| parts per force | **player=29, bbb-mig-force-b=2**, and the second force's `bbb-balancer` **researched** | the same | the same |
| where they are | `nauvis:0/0 bbb-mig-a:0/29 bbb-hidden:0/0 bbb-mig-b:0/2` -- nothing left as the incumbent's anywhere, on either surface that had any | the same | the same |
| the late build | `legacy=0 ours=1` | the same | the same |
| final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` | the same | the same |

**The trigger word on the summary line is an assertion, not decoration.** Leg 1 must be driven by `init` and legs 2 and 3 by `configuration_changed`; a leg that came out `first-dispatch` or `deferred` fails, because **a feature whose fallback silently does its primary trigger's work passes every test and ships broken.** That is the same thing `upg` asserts about which trigger drove a rebuild-from-world, and it is asserted here for the same reason. **Leg 2's ONE blocked line is the latch**: its create phase places an audit marker, which is a door onto the re-arm, and a Blocked state that re-decided every time it was asked would say so twice.

**THE RATES ARE THE `sok` BAND'S NOW, AND THE OTHERS ARE ZERO.** An adopted balancer laid one belt per part delivers exactly what a built one does, against a control belt in the same save; one laid the incumbent's way is refused and delivers nothing at all, which is the honest outcome rather than a regression. And in leg 1 `rebuildFromWorld` runs AFTER the conversion and **adopts the two working clusters beside the seven it refuses** -- `9 clusters, 2 adopted, 7 rebuilt` -- which is the only place in the estate a single-edge cluster is adopted next to refused ones.

**Leg 3, `bb3`, is the LIVE SUCCESSOR, and it is the coexistence shape rather than the swap shape on purpose.** Belt Balancer 3 is the one a player is most likely to be migrating off today and the one name in `legacyIncumbents` a typo would be most expensive in — and the reason the leg has to have both mods installed in phase one is that **a name this guest does not recognise does not fail loudly.** It falls through `legacyIncumbentActive` to the STRANGER branch, which is Blocked and **silent**, and then the removal load converts everything exactly as it should. Every conversion number in the column above is what a misspelled `belt-balancer-3` produces too. The NAMED blocked line from phase one is the only observable in this suite that can see a row of that list, which is what the leg is for and all it adds over leg 2.

**Leg 4, `built`, is the BUILD path and the PLAIN RELOAD.** No incumbent is ever installed, so `balancer-part` is this mod's own stub from the first byte and the observer's nineteen parts arrive one at a time through build events -- which is the path an old blueprint's ghosts take, minus the robot. Measured: **the whole-world scan runs and finds nothing** (it is silent, as it is on every other save this mod has ever been benchmarked on), **31 parts swapped after their own build events**, the `sok` band delivering 2.000x and 3.997x while the seven incumbent-idiom clusters are refused and deliver nothing, and the witness at 48 throughout. **This is the leg that found the quality defect below**: all of them is what it reports now and one short is what it reported the day the rig was written. It is also **the only leg whose save is written AFTER a conversion** and whose second phase changes no mod at all, so it is the only place the once-per-save flag can be seen surviving a save: on the reload the guest **says nothing about the migration at all**, which is asserted as the absence of any `[BBB] legacy:` line in the benchmark log.

**Leg 5, `readd`, is `Done -> Blocked` — the transition the hook exists for and the one nothing drove.** Its phase one is leg 4's: no incumbent, our own stub, thirty-one parts swapped through the build path, and the save written with the phase **Done**. Then `belt-balancer-2 2.0.9` is INSTALLED beside us. This is not an exotic order — a player installs this mod, uses it, and later installs a Belt Balancer to compare — and a save that had latched "scan done" would go on converting. Measured 2026-08-20:

| | measured |
|---|---|
| phase one | **31 parts swapped through the build path**, no scan summary line, no blocked line |
| the incumbent arrives | **exactly one** `belt-balancer-2 2.0.9 is active; its balancers are left alone`, and **0 converted** in phase two — no scan line and no build-path line |
| the world | `balancer-part=0 bbb-balancer-part=31` at **t1, post-audit and final**: the balancers this mod already owns do not move in either direction |
| the witness's copper | **48 -> 48 -> 48 -> 48** |
| the item stack | **50 held**, `place_result` **`bbb-balancer-part` -> `balancer-part`** — the other way round from every other leg, and correct: our stub owned the name while nobody else did, and the incumbent owns it again now |
| the technology | `bbb-balancer` **false -> true** (granted by phase one's own conversion, and an incumbent arriving takes nothing away) and `belt-balancer-1` **absent -> false** — PRESENT and unresearched, where every other leg requires it absent |
| the standing networks | `ctrl` 1306, **`sok2` 1306 1306 at 2.000x / 0.00% and `sok4` 1306 1304 1306 1304 at 3.997x / 0.15%** — unmoved across the incumbent's arrival. The seven refused clusters stay refused and deliver nothing, which is also unmoved |
| the fidelity pair | **85.0 of 170.0** and **uncommon**, on a `bbb-balancer-part` — here they crossed a SAVE as well as a swap, the conversion having happened in phase one |
| final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` |
| **the late build, which is the one with teeth** | it places the **INCUMBENT'S** `balancer-part` now, and comes out **`legacy=1 ours=0`** |

**That last row is the whole reason the leg exists.** `legacyBuilt` is gated on the phase being Done, and a gate reading the wrong phase does not crash, does not lose an item and does not move a rate: it **silently swaps a working mod's freshly built entity out from under it**. Nothing else in this suite can see it.

**Leg 6, `foreign`, is the stranger.** `guest/go/obs/foreigndata` defines `balancer-part` exactly as the incumbents do under a name this mod has never heard of, and it STAYS installed while this mod arrives beside it. Measured: **0 converted**, census `31 / 0` unmoved at every sample, the stranger's item still placing the stranger's entity, and the audit at **`clusters=0 parts=0 nets=0 drift=0 unbuilt=0 refused=0`** -- this mod owns nothing at all in that save, which since the port includes REFUSING nothing: a refusal there would mean a stranger's balancer had been converted and then declined. Its damaged part is still at **85.0 of 170.0**, still **uncommon**, and still a `balancer-part`; the second force's `bbb-balancer` is **false**, because a grant there would mean something of the stranger's had been converted.

**Leg 7, `fgone`, is the stranger UNINSTALLED**, which `legacyCheck` promises in as many words — *"the stranger can be uninstalled too, and on that load the stub appears and their balancers become ours, which is the same promise the incumbents get"* — and which nothing tested until it existed. It is not leg 6 with a different hook, and the difference is **when this mod is installed**: leg 6 has no guest at all in phase one, so the only thing it can watch a stranger's entities do is stand still. Here both mods are installed from the first byte, so the observer BUILDS nineteen of the stranger's `balancer-part` entities with a guest watching — and the guest must not touch one of them. Measured 2026-08-20:

| | measured |
|---|---|
| phase one, with the stranger installed | **ZERO blocked lines** (the stranger branch is silent by design, and a line here would mean the guest had decided bbb-mig-foreign is a Belt Balancer), **0 converted by the scan** and **0 swapped through the build path** over thirty-one build events |
| phase two, the stranger gone | **31 parts from 3 surfaces into 9 clusters, 2 forces, trigger=`configuration_changed`** |
| census | 31 / 0 -> **0 / 31** |
| the witness's copper | **48 -> 48 -> 48 -> 48** |
| the item stack | **50 held**, `place_result` `balancer-part` -> `bbb-balancer-part` |
| the technology | `bbb-balancer` **false -> true**, `belt-balancer-1` **absent in both phases** |
| rates | `ctrl` 1306, **`sok2` 2.000x** at 0.00% and **`sok4` 3.997x** at 0.15%; the seven refused clusters deliver nothing |
| the late build | `legacy=0 ours=1` |
| final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` |

**The technology check in this leg is its own, and that is not tidiness.** `belt-balancer-1` is absent in BOTH phases here — the stranger never defined one — so the assertion every other conversion leg makes (*the incumbent's technology is gone*) says nothing at all and would pass on a guest that granted nothing. What IS a statement is `bbb-balancer` going **false -> true**: unresearched while the stranger stood, researched by the conversion that followed it out.

**The two NAME PROBES, `belt-balancer` and `belt-balancer-performance`.** One `--create` each and nothing else. What a full leg would add over leg 2 is nothing — the conversion side of the feature is identical whichever name blocked it — and what it would cost is a benchmark phase, so the probe asserts the one thing that is not identical: the named blocked line, over a world that really does contain nineteen balancers the guest declined to touch. Measured 2026-08-24: **`belt-balancer 3.4.4`** and **`belt-balancer-performance 1.0.5`**, one blocked line each naming exactly that, **31 standing and 0 of ours**, **0 converted** and **0 refused**. The two versions are the harness's and are plausible rather than real; what they pin is that the guest read them back out of `script.active_mods`, not that any release carries them.

**And every leg ends with a LATE BUILD**, which is the probe that found the one defect review missed. One `balancer-part` is placed by script in phase two, well clear of every rig and after the final audit so nothing else moves, and what it separates is "this game's `balancer-part` is MINE" from "somebody else's": the four conversion legs come out **`legacy=0 ours=1`** and the two legs where somebody else owns the name -- the stranger, and the incumbent that arrived after us -- come out **`legacy=1 ours=0`**. The scan alone could not tell them apart, because the scan is gated on the marker either way; the BUILD path is gated on the PHASE, and the first cut of `legacyCheck` put the stranger case in `Done` -- which is the guest saying the name is its own. It says `Blocked` now, and Blocked also gets the marker re-test, which is what leg 7 is about.

**What the suite costs**: 12.7 s for the original four legs, 27.0 s for seven legs and two probes, 30.6 s once the world grew to nineteen parts on three surfaces, and **32.0 s** for the thirty-one-part world the single-edge band made, all on the same machine. The probes are ~1.5 s each because they stop after `--create`.

Re-measured on the review gate, back to back on one machine and against the SAME `dist/`, which is the only comparison worth having: **12.4 s** for master's four legs and **29.1 s** for the seven and two here. Master's four are green against this branch's guest, which is the expected result and is worth stating -- the quality fix can only help a leg that never built a quality part.

**And it is the one base suite that is not base-only.** `mig_list` enables the **`quality`** mod, in BOTH phases of every leg, because `legacyConvertOne` passes the old entity's quality through to the new one and a game with only `normal` in it cannot tell a guest that carries it apart from one that drops the key. A constant across the two phases is not a mod-set change, so no leg's trigger moves; `quality` depends on `base` alone, so this stays base-plus-one rather than becoming a second Space Age run.

## What the conversion CARRIES, and where the parts are

Three things `legacy.go` claims in as many words and nothing measured until 2026-08-20: that a converted part keeps its **health** and its **quality**, that the technology is granted **per force**, and that the scan walks **every surface**. Every part in every leg was an undamaged, normal-quality, player-force part on one surface, so all four claims were satisfied by a guest that did none of them.

| what | how it is measured | measured |
|---|---|---|
| **health** | the `fid` rig damages one part to 85 of 170 in `on_init` and the observer reports the value it reads back, in phase one and again after the swap | **85.0 -> 85.0**, on a `bbb-balancer-part`, at t1 and at final, in every conversion leg |
| **quality** | the other `fid` part is CREATED at `uncommon`, which needs the quality mod, which is why `mig_list` enables it | **uncommon -> uncommon** |
| **per-force technology** | the `frc` rig puts two parts on a second force; the summary line's own force count, plus a per-force technology line for that force | **2 forces given the balancer technology**, and `bbb-mig-force-b`'s `bbb-balancer` **researched** |
| **two forces' parts touching are two balancers** | the audit's cluster count, against **the number written down in `assert-mig.py`** rather than against the guest's own summary | **9 clusters** out of 31 parts; a fusion reads as 8 |
| **every surface was scanned** | the summary line's surface count against the number of non-hidden surfaces the observer can see | **3 and 3** |
| **parts on more than one surface were really converted** | the per-surface census | `bbb-mig-a:0/29 bbb-mig-b:0/2` -- nothing left as the incumbent's on either |

**The anti-vacuity guards are the point of three of those rows.** A part that was never damaged sits at `max_health`, and an equality across the swap is then satisfied by a guest that copies nothing -- so phase one must report a value BELOW the maximum or the leg fails as vacuous. A game with only `normal` quality in it cannot tell a carried quality from a dropped key -- so phase one must report `uncommon`. And a fusion check over one force says nothing -- so the second force must own exactly the two parts the rig builds, at every sample.

## A CHECK THAT SKIPS IS A CHECK THAT PASSED -- the review gate, 2026-08-20

**Three of this suite's shared checks read `if got is None: continue`, and a phase the observer stopped reporting therefore took its whole assertion with it, silently, while the leg went on printing the phases that were still there and exiting zero.** Found by reviewing the coverage pass rather than by a run, and closed the same day; two of the three are the shape the original inline code had, and the third arrived with the fidelity rig.

**It is a measurement, not a worry.** Delete three lines from the observer's tick-1 handler -- `report_counts("t1")`, `report_item("t1")`, `report_fidelity("t1")` -- and the pre-fix script prints

    witness: 48 copper plates in phase one
    witness: 48 at phase=post-audit
    witness: 48 at phase=final
    fidelity in phase one: 85.0 of 170.0 health, quality uncommon
    fidelity at phase=final  bbb-balancer-part at 85.0 of 170.0 health
    ...
    the incumbent's balancers were adopted and they balance

and **exits 0**, on a run in which the copper count at the one sample taken straight after the load was never made, the item stack was never compared with itself at all, and the health and the quality were never checked across the swap. Every one of those lines is written unconditionally by the observer at every phase named, so an absent one is a broken harness rather than a legitimate shape, and the run that says so is worth more than the run that does not. All three fail now, by name and by phase, and so does the `foreign` leg's own inline item check.

**And the same review found the probe's success message telling a lie.** The two name probes asserted the SCAN had converted nothing (`if adopted`) and said nothing about the BUILD path -- and those two names are in front of the guest nowhere else in this repo, while the observer builds nineteen of the incumbent's entities with the guest listening. With `legacyBuilt`'s phase gate removed the probe printed *"belt-balancer is recognised by name and its balancers were left alone"* and exited 0 over a create log carrying **nineteen** `legacy: adopted a balancer-part built at` lines. One line of assertion, and the more expensive of the two failures is watched on the whole name list rather than on half of it.

**The pattern to carry forward**, because this suite is entirely built of log lines and every future check here will be too: a missing line and a wrong line are not the same failure, and only one of them fails on its own. Assert the presence, then assert the value.

**THE CLUSTER COUNT IS THE ONE NUMBER THAT HAD TO STOP COMING FROM THE GUEST.** `check_audit` compared the audit's cluster count against the count on the guest's own summary line, and both come out of the same flood fill -- so a fill that fused two forces' touching parts moves them together and neither says anything, while every census, every copper count and every rate stays exactly what it was. The expected count is a constant in `assert-mig.py` now, derived from the rigs the observer builds, and the summary line is checked against it as well.

**AND THE SUMMARY LINE'S SURFACE NUMBER COUNTS SURFACES SCANNED, NOT SURFACES THAT HAD PARTS.** `legacyScan` increments it once per non-hidden surface before it looks at one, so `adopted 31 parts from 3 surfaces` on a save whose parts are all on two of them is the literal truth and reads as something else. It is left as it is and the reading is written down here rather than changed, because the line is the assertion surface for this whole suite and its format is read by `assert-mig.py` and quoted a dozen times in this file. What the pass added instead is the observer's own **per-surface census**, which is the only thing that can see a scan that visited a surface and converted nothing on it -- red-proven below, and the two statements fail differently.

## The quality nobody had ever built — what the fidelity rig found

**A `balancer-part` standing at any quality but `normal` was invisible to the migration's build path, and stood there unconverted, unregistered and unlogged for the rest of the save.** Found on the first run of the `fid` rig, 2026-08-20, before a line of the assertion had been written: leg 4 swapped **18 of 19** parts and the one it missed was the uncommon one.

**`LuaSurface.find_entity` takes an `EntityWithQualityID`, and the pinned runtime API says of a bare name that "Normal quality will be used".** So `find_entity("balancer-part", p)` is a query for a NORMAL-quality part at `p` and nothing else. Measured on 2.0.77 with a scratch mod, against a real uncommon entity at a known position:

| | result |
|---|---|
| `find_entity("iron-chest", p)`, normal chest at p | the object |
| `find_entity("iron-chest", p)`, **uncommon** chest at p | **nil** |
| `find_entity({name = "iron-chest", quality = "uncommon"}, p)` | the object |
| `find_entities_filtered{name = "iron-chest", position = p}` | **1**, whatever the quality |

**The whole-world SCAN never had it**, which is why six of the nine legs were green over it: `legacyConvertOn` filters on the name alone through `findByNameAll`, and the engine returns every quality. Only `legacyRunBuilds` -- the path that re-finds a stub one tick after its build event -- asked the quality-scoped question, and that path is **the blueprint book's**: a migrating player's ghosts revive as stubs and are swapped a tick later. An uncommon balancer-part in an old blueprint would have revived and stayed a stub.

The fix is that call, and it is `setSearchBox` + `findByName` -- the idiom `registerPartsIn` already uses. One slice per stub built, on a once-per-save path.

**FOUR MORE CALL SITES PASS A BARE NAME TO `find_entity` AND ARE NOT FIXED HERE**, because none of them is this pass's subject, none can be red-proven by this suite, and the cheap repair for the two that matter is not cheap:

| call site | what a non-normal-quality part does to it |
|---|---|
| `skin.go`, `restyle` | the part is never found, so `graphics_variation` is never set and an uncommon balancer draws **cell 1, the lone-part picture, forever** -- and because `pvar` is written only after the engine takes the value, that part is RETRIED on every restyle of its cluster, which is one host call per flush that touches it rather than the zero the M5 budget is priced at. The mechanism's whole budget is *one byte per part*, and the two repairs are a second byte (the quality) or an allocating area query on the flush path -- and the `mar` suite asserts those slopes to the byte. **A design question, not a one-line fix**. The `mig` save has carried one such part since the `fid` rig existed, so the cost is being paid in this repo today |
| `limit.go`, `forceOfCluster` | the cluster's force cannot be resolved, so an over-limit refusal on a balancer of uncommon parts is logged and **nobody is told** |
| `limit.go`, `revertOne` | the over-limit piece is not found, so it is **not handed back** -- the negative the `edge` suite asserts (zero hand-backs) is unmoved, because a headless run has no player |
| `fastreplace.go`, `reapFastReplaced` | it reads a standing uncommon part as GONE and **unregisters a part that is still there**. It needs a foreign belt-connectable to APPEAR on a part's tile, which in ordinary play only a fast replace does -- and a fast replace really did remove it, so the answer is right for the wrong reason on the gesture, and wrong for a script that builds a colliding belt |

None of the four is reachable by any rig in this repo, because nothing outside `mig` builds a quality part at all. Fixing them is a pass of its own and it starts with a rig, not with a call site.

> **ALL FOUR ARE FIXED SINCE 2026-08-20**, by exactly that pass: `findOnTile` (guest/go/findpart.go) is the fix stated once, the `qual` suite is the rig it starts with, and "A part at uncommon quality is a part" below is the write-up -- including the answer to the `restyle` design question, which turned out to cost nothing the `mar` suite can see. The table above is kept as the record of what each site did while it stood.

## The build path DEFERS, and that is a correction the harness forced

The obvious version of `legacyBuilt` converts in place, inside the build event. It works, and **`create_entity` then returns NIL to whoever placed the entity**, because the entity was destroyed during the event it raised. Measured within minutes of writing it: the harness's own `create_entity{name = "balancer-part", raise_built = true}` came back nil and the test mod raised on it -- which is exactly what another mod scripting a `balancer-part` into the world would do.

So the event half writes down **one tile** and asks for a flush, and `legacyRunBuilds` at the top of `flush()` re-finds the stub and swaps it. The stub stands for one tick, which is invisible (it draws this mod's own lone-part picture) and is the same latency every ordinary part placement already has. It is also where this guest does everything else that reads the world, and it means the `AddPart` lands in the drain that is about to compile it -- including the synchronous drain a `bbb-audit` marker forces, which is the only one a `--create` ever reaches.

## Red-proven seventeen times, and every proof catches a different thing

The first three are 2026-08-16 and are about the feature; the next four are 2026-08-20 and are about the three legs and two probes added that day; the next seven are the same day's fidelity pass; the last three are the same day's REVIEW gate, and they are the odd ones out -- the defect they inject is in the HARNESS rather than in the mod, because what they prove is that three checks which had never been able to fail now can. Every one is an injected defect, built, run, and reverted.

**The first seven rows were measured against the FOUR-RIG, ELEVEN-PART world and are kept as measured, and the rest against the NINETEEN-PART one.** The suite builds thirty-one parts on three surfaces now, and it runs on Factorio 2.1 where seven of its nine clusters are refused -- so the counts inside every row below are the world of the day they were taken and not today's. Which assertion fires is the claim; the count beside it is the evidence that was taken for it. **None of the seventeen retires under the port**, because every one of them is about the CONVERSION and the port did not touch `legacy.go`; five were re-derived against the reworked suite on 2026-08-24 and they are in [`agents/single-edge.md`](../single-edge.md)'s phase-7 section.

| injected defect | what came out |
|---|---|
| **`data-final-fixes.lua`'s require commented out** -- no stub prototype | phase two's census is `balancer-part=0 bbb-balancer-part=0`: **all 11 entities deleted by the engine at load**, and the 50-item stack with them (`held=0 place_result=nil`). The suite fails on "nothing was adopted at all". **The witness's 48 copper plates survive**, which is the honest detail: the belts are vanilla, so what the missing stub loses is the machine and not the goods |
| **`legacyStubPresent()` removed from `legacyCheck`** -- the marker guard | the foreign leg converts **11 of a stranger's entities** (`balancer-part=0 bbb-balancer-part=11`, `trigger=init`, audit `clusters=3 parts=11`) and **four assertions fire**. The other legs stay green, which is the whole point: the guard is invisible to every leg that is entitled to convert |
| **the stranger case landing in `Done` rather than `Blocked`** | the SCAN still leaves the stranger alone -- `balancer-part=11 bbb-balancer-part=0` at every census, audit `clusters=0` -- and the **BUILD PATH does not**: one part built beside the stranger in phase two comes out `legacy=0 ours=1`. Exactly one assertion fires, and it is the late-build probe, which exists because of it |
| **`"belt-balancer-3"` misspelled in `legacyIncumbents`** | leg 3 fails on **exactly one assertion** -- *"the guest said it was leaving the incumbent alone 0 times"* -- and **every other number in that leg is byte-identical to the green run**: 11 parts, 2 surfaces, 3 clusters, 1 force, `trigger=configuration_changed`, witness 48 at four samples, 3.997x at 0.15% and 2.995x at 0.13%, late build `legacy=0 ours=1`, audit `clusters=3 parts=11 nets=3 drift=0 unbuilt=0`. Leg 2 stays green with `belt-balancer-2 2.0.9` still named. **That is the proof, not a caveat**: an unrecognised name takes the silent stranger path, the removal converts everything anyway, and the blocked line is the only thing that ever knew |
| **`"belt-balancer"` and `"belt-balancer-performance"` misspelled**, the same way | legs 1--7 all stay green and **each probe fails on its own line** -- *"...0 times and it is once per decision -- and ZERO means it did not recognise `belt-balancer` at all, which is the silent stranger path"*, and the same for `belt-balancer-performance`. Both probes still report **11 standing, 0 of ours**, so the failure is recognition and not staging |
| **`legacyBuilt`'s phase gate removed** -- the build path stops asking whether this game's `balancer-part` is ours | leg 5 fires **exactly the two assertions that are about it**: *"a `balancer-part` was converted in phase two, with an incumbent installed"* and *"a `balancer-part` built while belt-balancer-2 is installed came out legacy=0 ours=1"*, with `[BBB] legacy: adopted a balancer-part built at 12,0` in the log to say so. Everything else in the leg is unmoved -- 11 swapped in phase one, one blocked line, census `0 / 11` throughout, witness 48, 3.997x/2.996x, audit `3 / 11 / 3 / 0 / 0`. **The injection is caught EARLIER in the suite too**, by leg 2, whose phase one has an incumbent installed and whose eleven live entities the ungated build path converts -- so leg 5 was re-run in isolation to see its own assertions fire |
| **`legacyStubPresent()` removed from `legacyCheck`**, again -- but read by leg 7 rather than by leg 6 | leg 7 fires **exactly one assertion**, *"a `balancer-part` was swapped through the build path while the stranger owned the prototype"*, over **11** `adopted a balancer-part built at` lines in the CREATE log. **This is a moment leg 6 cannot see at all**: it has no guest in phase one, so the only thing it can watch a stranger's entities do is stand still, where leg 7 watches eleven of them being BUILT with the guest listening |
| **the bare-name `find_entity` put back in `legacyRunBuilds`** -- the defect as it shipped | legs 1--3 stay **green**, which is the honest half: the whole-world scan is quality-agnostic and never had the bug. **Leg 4 fails**, on *"19 parts were placed and 18 were swapped through the build path"*, *"1 incumbent parts are still standing"*, *"19 parts went in and 18 of ours came out"*, *"the audit counted 18 parts"*, and -- naming the tile -- *"the quality tile holds a balancer-part at phase=t1 and should hold a bbb-balancer-part"*, twice |
| **the `SetHealth` copy skipped in `legacyConvertOne`** | leg 1 fires **exactly the two health assertions**, one per phase: *"the damaged part is at 170.0 health at phase=t1 and was at 85.0 before the swap"*. Nothing else in the leg moves -- the part is converted, registered, compiled and balancing, and **silently repaired to full**, which is a building this mod was not asked to touch |
| **the `quality` key dropped from `legacyCreateArgs`** | leg 1 fires **exactly the two quality assertions**: *"the uncommon part came back at quality 'normal' at phase=t1"*. Every other number is unmoved: a quality is not a rate and not a count |
| **only the first force granted the technology** in `legacyScan` | leg 1 fires **exactly three**: *"1 force(s) were given the balancer technology and the force rig puts parts on 2"*, and the second force's `bbb-balancer` being false at t1 and at final. The parts, the clusters, the items and the rates are all unmoved -- a force that cannot craft a spare part is not a number any of them carry |
| **the force check removed from `AddPart`'s adjacency loop** -- two forces' touching parts fuse | leg 1 fires **exactly two, and both are the cluster count**: *"the conversion made 6 clusters out of the observer's 7"* and *"the audit finds 6 clusters and the observer built 7"*. **19 parts, 48 copper, 3.997x, 2.995x, drift=0, unbuilt=0 -- every other number in the leg is byte-identical to the green run.** That is the whole reason the expected count stopped being the guest's own |
| **`legacyScan` breaks out after the first surface it converts anything on** | leg 1 fires the surface cross-check by name -- *"the scan reports 2 surfaces and the world has 3 that are not the hidden one (nauvis, bbb-mig-a, bbb-mig-b)"* -- plus a cascade (17 adopted of 19, 6 clusters, and *"the migration ran 2 times in one save"*, because the `added` leg decides twice and the second decision finishes the job the first abandoned) |
| **`legacyScan` VISITS every surface and converts on only the first one that had parts** | the same leg, and now the per-surface census is the assertion with the name on it: *"this mod's parts are standing on 1 surface(s) after the conversion and were built on 2"*. The surface COUNT is correct here -- 3 scanned, 3 in the world -- so the cross-check above says nothing and the census is the only thing that can see it. **The two surface statements fail on different defects, which is why there are two** |
| **the observer's `report_counts("t1")`, `report_item("t1")` and `report_fidelity("t1")` deleted** -- the harness stops reporting a phase rather than the guest doing something wrong | the fixed script fires **exactly three**, by name and by phase: *"no copper count for phase=t1"*, *"no legacy-item line for phase=t1"*, *"no health line for phase=t1"*. **The same logs against the pre-fix script exit 0** and print the cheerful *"the incumbent's balancers were adopted and they balance"*, having skipped the copper count at the one sample straight after the load, the item comparison entirely, and the health and quality across the swap. That is the proof: not that a defect was caught, but that three checks which had never been able to fail now can |
| **the observer reports the quality line only in phase one** (health untouched) | the fixed script fires **exactly two**: *"no quality line for phase=t1"* and *"...for phase=final"*. Pre-fix, **exit 0** again. The finer injection is needed because both lines come out of one function, so deleting the call reaches the health branch first -- what this one covers is the quality line's FORMAT drifting out from under its regex while the health line's does not |
| **`legacyBuilt`'s phase gate removed**, read by the two NAME PROBES rather than by leg 5 | each probe fires **exactly one assertion**, the new one: *"19 `balancer-part` entities were swapped through the BUILD path while belt-balancer was still installed"*, over 19 `adopted a balancer-part built at` lines in the create log. **Pre-fix the same log exits 0 and prints "belt-balancer is recognised by name and its balancers were left alone"** -- a success message that was false in its own second clause. The probes were run in isolation, as leg 5's proof was and for the same reason |

**The fourth and fifth rows are the ones worth reading twice**, because they are the shape this suite exists to catch and the shape that is hardest to catch: the defect changes **nothing about what the mod does**. A misspelled incumbent name converts the same eleven parts into the same three clusters at the same rates on the same trigger. What it changes is that the guest stopped recognising a real mod by name — and the day that mod's balancers are standing in a save while it is still installed, it would convert them.

## The real Belt Balancer 2, once, by hand

The stand-in is a stand-in, so the added-as-removed flow was run once against the **actual cloned Belt Balancer 2 source** (github IThundxr/belt-balancer-2, MIT, info.json version corrected to 2.0.9), with its real control stage running during phase one. Not committed and not a suite dependency.

**It does not load as cloned**, and that is worth recording: `objects/part.lua` has a `goto continue` at line 266 with **no `::continue::` label**, which is a load-time Lua error (*"no visible label 'continue' for <goto> at line 266"*). One label inserted at the end of that loop body and it loads. The released 2.0.9 presumably carries the fix; the repository at HEAD (`456da9d`) does not.

With that one line added, the same `assert-mig.py --leg added` passes against the real mod with **numbers identical to the stand-in's, to the item** -- measured 2026-08-16 against the four-rig world, and not re-run since: 11 parts adopted from 2 surfaces into 3 clusters, 1 force researched, `trigger=init`, witness 48/48/48/48, 50 held with `place_result` flipped, `bbb-balancer=true`, control 1306, `m4x4` 3.997x at 0.15% and `m3to5` 2.995x at 0.13%, final audit `clusters=3 parts=11 nets=3 drift=0 unbuilt=0`. That is what makes the stand-in evidence rather than a model.

## What it costs

**Nothing on any hot path, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** -- 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory -- which is the gate a pass that puts a gate on `onEventBody` and another on `fk_on_deferred` has to clear. Both are one integer compare in every save that has never seen an incumbent, and the marker re-test costs two host calls and no allocation in the only state that reaches it. The collected arm ended on 0.46 MiB with a 9,184 B live set, 9 collections in 5 paced steps and **0 forward-progress deadlines**.

Package built 2026-08-16, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 327,615 B | **347,170 B** | +5.97% |
| `fk_module.lua` | 2,534,887 B | **2,732,022 B** | +7.78% |
| `dist/bbb.wasm` | 1,106,373 B | 1,153,588 B | |
| members bound into the mod | 42 | **51** | of 4,257 |
| prototypes added | — | **3** | a `balancer-part` entity, a `balancer-part` item and the `bbb-legacy-stub` marker. ALL THREE CONDITIONAL: none is defined when another mod owns the name |
| sprite references checked | 7 | **10** | |

The nine new members are `LuaBootstrap.active_mods`, `LuaPrototypes.entity` (the handle-returning form) and `LuaCustomTable`'s index operator, `LuaEntity.quality`, `LuaEntity.health` and its setter, `LuaForce.technologies` (the handle-returning form) and `LuaTechnology.researched` and its setter. **No new FkLua gap and no re-pin**: every one was reachable through the generated bindings as they stood, `fklua gen-bindings --check` and `fklua lock --check` are unmoved, and the API pin does not move. The one thing that DID need upstream is the hook, and it landed in the same round.

## Verification, and the one gesture that is interactive

All nine suites green in **both arms** from clean, 2026-08-16, and **no other suite's numbers moved at all** -- M2's control is 1,306 with every rig rate unchanged, `edge`'s baseline is fifteen clusters over ninety-five parts with 0 lost over 200 teardowns, `mix` still reports its 72-item overflow, `plat` still reports its stacked-sushi band. That is the expected result rather than a weak one: no save in any other suite contains a `balancer-part` prototype, so the gate is a compare and the scan is never entered.

**What a headless run does NOT reach is a ROBOT reviving a ghost**, and the look of the thing. The code path a revive takes IS exercised -- the `built` leg drives `legacyBuilt` nineteen times through `script_raised_built`, and the late-build probe drives it once more in every leg -- so what is behind the wall is the construction network and the blueprint, not the swap. Both they and the pixels are on the interactive checklist as gesture F ([`test/interactive/README.md`](../../test/interactive/README.md)): take a real save with a real incumbent, swap the mods, and check the summary line, the plating, the belts, the chest, the technology -- then place one of the old blueprints and watch each revived ghost become one of this mod's parts.

**And verified again 2026-08-20, after the COVERAGE PASS that took the suite from four legs to seven plus two probes.** No guest line changed -- `dist/bbb.wasm` and `fk_module.lua` (2,745,246 B) are the bytes the merge left, `make check` is green with bindings and lock unmoved, and the sprite checker is green at 10 references. **All nine suites green in BOTH arms**, the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0), and **no other suite's numbers moved at all**, which is the only result a test-only pass may have. Inside `mig`, the four pre-existing legs report exactly what they always did -- 11 parts, 3 clusters, `trigger=init` and `trigger=configuration_changed`, witness 48 at every sample, 3.997x and 2.995x, audit `clusters=3 parts=11 nets=3 drift=0 unbuilt=0` -- and what the pass adds is three legs, two probes and four red proofs. The suite went 12.7 s to 27.0 s. **Those eleven-part counts are that pass's world**; the fidelity pass below took it to nineteen parts on three surfaces, and every current number in this section is the later one.

**And verified again the same day, after the FIDELITY PASS.** The suite claimed four things `legacyConvertOne` and `legacyScan` do -- health, quality, per-force technology and every surface -- and measured none of them, because every part in every leg was an undamaged normal-quality player-force part on one surface. Three rigs and a second surface later it measures all four, and **the first run of the first rig found a defect**: a legacy part at any quality but `normal` was invisible to the build path's `find_entity`, so leg 4 swapped 18 of 19. Fixed at that call site; the four other bare-name `find_entity` calls in the guest are written up above and were NOT fixed here -- they got their own pass and their own suite the same week ("A part at uncommon quality is a part" below).

`make check` green with bindings and lock unmoved, sprite checker green at 10 references, and **all nine suites green in BOTH arms**. The leaking arm's seven slopes came back **identical to the byte** -- 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0 -- which is the gate a guest change has to clear, and it clears it for the structural reason: the call that moved is on a once-per-save path that no other suite reaches. **No other suite's numbers moved at all.** Shipped config, forced clean rebuild either side of the fix:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 413,460 B | **413,608 B** | +0.04% |
| `fk_module.lua` | 2,745,246 B | **2,746,726 B** | +0.05% |
| `dist/bbb.wasm` | 1,162,033 B | 1,162,312 B | |
| members bound into the mod | 51 | **51** | of 4,257 -- none added; `find_entities_filtered` was already bound |

**What the pass closed, in one sentence each.** `belt-balancer-3` was never in front of the guest and neither were the other two names, so three of the four rows of `legacyIncumbents` were unpinned and a typo in any of them would have converted that mod's balancers out from under it while it was still installed. `Done -> Blocked` was never driven, so the build path's phase gate -- the one thing standing between an incumbent that arrives late and having its freshly built entities swapped -- had no test. And `legacyCheck`'s promise to a stranger, that uninstalling gets them the same adoption an incumbent gets, was a comment.

**And verified once more the same day, after the REVIEW GATE over both passes above.** Nothing in the guest, the observer or `run.sh` moved -- the whole change is four assertions in `test/assert-mig.py` that could not previously fail, plus two stale prose numbers in this file -- so `dist/` is byte-for-byte what the fidelity pass left (zip 413,608 B, `fk_module.lua` 2,746,726 B, 51 members) and no rebuild was needed to reach it. `make check` green with bindings and lock unmoved; **all nine suites green in BOTH arms**, each arm one invocation; the leaking arm's seven slopes back **identical to the byte** -- 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0; **no other suite's numbers moved**, which for an assertion-only change is the only result available. Three red proofs, all of them injections into the HARNESS, are the last three rows of the table above.

**What the gate found, and it is one shape seen twice.** Three shared checks and one probe could pass while measuring nothing -- a missing log line was a skipped assertion rather than a failed one, and the probe's own success sentence (*"...and its balancers were left alone"*) printed over a create log carrying nineteen conversions of a still-installed mod's entities. Both are written up in "A CHECK THAT SKIPS IS A CHECK THAT PASSED". What it did NOT find: nothing was weakened by either pass, the original four legs assert everything they always did and five things more, `bench/` was untouched, and every claim in the two passes' tables re-ran to the number recorded.

## And on Factorio 2.0 the outcome splits the other way -- the conversion is grandfathered

**A player who uninstalls Belt Balancer on 2.0 keeps their base, and until 2026-08-24 nothing had ever run that branch.** The conversion is byte-identical on both engines -- `legacy.go` knows nothing about the rule -- and so is the first flush after it: the setting defaults to false, so all seven incumbent-idiom clusters are refused with the same seven alert lines and the same shape multiset `[2, 2, 2, 2, 2, 3, 4]`. What happens NEXT is the whole difference. The capability marker is present, so `settleEdgeMode` asks `edgemode.GrandfatherNeeded(marker, Off, 7)`, gets true, writes `bbb-multi-edge-parts` ON, re-queues every cluster and tells each owning force -- and the very next flush compiles all seven.

| | 2.1 | **2.0** |
|---|---|---|
| the final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` | **`nets=9 ... refused=0`** |
| `sok2` / `sok4` | 2.000x / 3.997x | **2.000x / 3.997x** |
| `m4x4` / `m3to5` | **0 0 0 0** and **0 0 0 0 0** | **3.997x at 0.15% and 2.996x at 0.26%** -- the pre-port records to the digit |
| what the player is told | the migration checklist | **the grandfather warning**, both once per force: 6 and 1 |
| teardowns, spills | 0, 0 | **0, 0** |

**Two constants had to be SPLIT rather than switched**, and both splits are the kind one engine cannot see. The number of refusal LINES is 7 on both; the audit's own `refused=` column is 7 on 2.1 and **0** on 2.0, because the grandfather compiles all seven a flush later and a successful compile clears the feedback memo. And the `added` leg's rebuild-from-world reports **2 adopted / 7 rebuilt on BOTH** -- it runs in the dispatch after the conversion's flush, before the grandfather's re-queue has been flushed -- so that is its own constant and not the audit's. **The per-force log line is SHARED by both messages** (`tellAffected` writes it whichever sentence it is delivering), so what it pins is asserted on both engines; only the summary SENTENCE is 2.1-only, and on 2.0 it would be false of every balancer a flush later.

## And verified again 2026-08-24, on Factorio 2.1, where the outcome splits

**The single-edge port did not touch this feature and changed its answer completely.** `legacy.go` is byte-identical: 31 parts still convert from 3 surfaces into 9 clusters at their health and their quality, 2 forces are still granted the technology, the item stack still survives and still flips its `place_result`, the witness's 48 copper plates are still 48 at every sample, and the state machine's two axes are untouched because it knows nothing about belts. What moved is what the compiler then does with the clusters: **seven of the nine are laid the incumbent's way -- one column of parts with a belt on both free faces, which is two belts per part -- and Factorio 2.1 refuses them.**

**This is the one suite whose rigs were deliberately NOT re-laid.** Every other suite in the estate obeyed the rule because its rigs are ours; this one's world is somebody else's, and re-laying it would have been re-laying the thing under test. What was added instead is the **`sok` band**: the same balancer two columns wide, one belt per part, which is a shape a Belt Balancer user could genuinely have and which converts into a network that runs. One world, both outcomes, which is the portal story rather than a hypothetical.

    clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7

identical in all five conversion legs, with `sok2` at **1306 1306 -- 2.000x one belt, 0.00% spread**, `sok4` at **1304 1306 1306 1304 -- 3.997x, 0.15%**, and `m4x4` and `m3to5` at **exactly zero**, asserted as zeros rather than as a loosened bound. **Nothing is torn down and nothing is spilled** -- a cluster the conversion just created never had a network, so the refusal in front of the teardown has no teardown to be in front of -- which is where this suite asserts the opposite of `mig21`'s for the same refusal, and correctly. And leg 1's rebuild-from-world **adopts the two working clusters beside the seven it refuses**, which closes a gap the port's phase-2 section records against `mig21` and could not close there.

`make check` green with bindings and lock unmoved (no guest line changed and no package was rebuilt); **all thirteen suites green in BOTH arms**, each arm one invocation, with `mig` back in the default; the leaking arm's slopes **identical to the byte** to the port's phase-4 record; and **no other suite's numbers moved**. Five red proofs, two about the new outcome and three regressions across three families of the conversion, are in [`agents/single-edge.md`](../single-edge.md)'s phase-7 section, along with the one defect this pass found and did not fix -- a converted-and-refused balancer gets the ORDINARY per-piece message rather than the migration summary, unless a rebuild-from-world happens to follow the conversion.

<!-- END: adopting an incumbent's save -->
