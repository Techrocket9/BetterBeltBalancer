# The FkRecipes migration, and what the library cost to adopt

BBB's two startup dropdowns (`bbb-recipe-cost` and `bbb-tech-cost`) are declared through [FkRecipes](https://github.com/Techrocket9/FkRecipes) since 2026-09-01, and its PROTOTYPES followed in **round two** the same day, on a library that had answered every refusal round one measured. Everything else this mod emits at the data stage stays hand-rolled, and this note records why, with the measurement behind each decision. **Read "Round three" for the current state, and "Round two" for how the prototypes crossed**; everything before round two is round one's record and is kept as it was measured, because the refusals it names are what the library's new surface is answering. It is also the dogfood report for the library: a graded list of every piece of friction met adopting it, in the shape FkRecipes' own `agents/implementation-notes.md` uses, so the asks land somewhere the library's maintainer can act on them.

The decision was taken on two measurement records made before a shipping line was written: a green baseline on the untouched tree, and five scratch-clone experiments (A through E) that built the settings-only shape, built the full-library shape, and drove the pure planner against a stub. Both are cited throughout. **Every verdict here rests on the Factorio 2.0.77 golden row**, because that is the only engine this machine has; see "What is NOT RUN on this machine".

## What migrated and what stayed, and why

**THIS TABLE IS ROUND ONE'S AND THREE OF ITS ROWS ARE SUPERSEDED.** The item, the recipe and the technology are declarations too since round two; the rows are kept as they were measured, because the refusals they record are what the library's new surface answers.

| what | verdict | why | evidence |
|---|---|---|---|
| `bbb-recipe-cost` (startup string dropdown, default `vanilla`, order `a`, six values) | **MIGRATED**, `fkrecipes.LegacyDropdownSettingNeedingLocale` | The Legacy constructors take a full name and an explicit order and emit both verbatim, which is exactly the shape a mod that already shipped needs: Factorio persists startup values in `mod-settings.dat` by name with no rename mechanism | `mod_settings_sha256` unmoved at `196275f867f7f8b5` on both mod sets (exp C); `PlanSettings` against a stub World emits the six fields verbatim (exp E) |
| `bbb-tech-cost` (startup string dropdown, default `logistics`, order `b`, three values) | **MIGRATED**, the same constructor | The same, and the two are the whole of what this mod asks a player at startup | the same two |
| `bbb-multi-edge-parts` (bool, **runtime-global**, defined on Factorio 2.0 only, written by the mod itself at runtime) | **HAND-ROLLED**, beside the plan | The library generates startup settings only and reads startup settings only, which `docs/migration.md` states in as many words. A runtime-global that this mod's own `writeMultiEdgeSetting` assigns at runtime is not a thing FkRecipes has a verb for, and it must stay absent on 2.1 where the key does not exist | `docs/migration.md`'s worked example names this setting and prescribes exactly this arrangement |
| recipe `bbb-balancer-part` | **HAND-ROLLED**, resolved by `guest/go/tune`'s `RecipePlan` and `Resolve` | FkRecipes prefixes every prototype name with `better-belt-balancer-` and has no Legacy form for items, recipes or technologies; it has no `order` slot on `RecipeSpec`; and `Recipe` takes an `ItemRef`, so it can only produce an item the same plan declares, which would drag this mod's item and therefore its `place_result` through the same rename | exp D and D2, below: measured, not argued |
| technology `bbb-balancer` | **HAND-ROLLED**, resolved by `guest/go/tune`'s `TechLadder` | The same prefix wall, plus no `order` slot on `TechSpec`. `CostBy` itself produced the right unit and the right prerequisite when driven by hand, so this is a naming refusal and not a capability one | exp D2, driven by hand |
| item `bbb-balancer-part` | **HAND-ROLLED** | `ItemSpec` carries no `place_result`, and an item that cannot place the balancer part is not this mod's item | exp D2's field-by-field DROPPED list |

**The plan lives in `guest/go/tune/plan.go`** and is emitted by `Emit()` from the data guest's `fk_settings` hook alone. That is the one place this arrangement departs from every published example: the library's own quickstart, and `docs/usage.md`'s "The shape of a mod", both route `Emit` into `fk_settings` **and** one data-family hook. A plan whose only declarations are settings has nothing to emit at a data stage, and routing it into one anyway would be a second pass over prototypes it never declared. It works, it is silent, and it is documented nowhere; see the friction entry.

**The locale tripwire is `guest/go/tune/locale_test.go`**, and it is the library's check plus three of this mod's own that the library does not make (**four since fix round 1's commit 3**, which adds `TestEveryCustomDropdownDescriptionNamesTheCustomOption`; see "Commit 3" below):

```go
Plan().CheckLocaleWith("better-belt-balancer", cfg, []string{"bbb-multi-edge-parts"})
```

That call is `docs/migration.md`'s own prescription for this mod, and passing the hand-rolled name is what makes the orphan scan complete rather than prefix-shaped (every name this mod ships carries the historical `bbb-` prefix and none carries `better-belt-balancer-`, so the plain `CheckLocale` orphan rule would see nothing at all). What BBB asserts beside it: that both cost settings carry **non-blank descriptions**, which the library deliberately never demands because a description is optional; that the hand-rolled bool carries its own `[mod-setting-name]` entry, which the library suppresses rather than requires (the list "suppresses orphans; it does not create obligations"); and that the label quoted mid-sentence in the `[bbb] single-edge-grandfathered` message ("Allow multiple belts per balancer part") is a **prefix** of that bool's `[mod-setting-name]` entry ("Allow multiple belts per balancer part (Factorio 2.0 only)"), so a rename of the setting's label cannot leave the grandfather warning telling a player to look for a menu row that no longer reads that way. Nothing in the library can know about that third one; it is a fact about one of this mod's own sentences.

**A pure-Go test pins `PlanSettings` against a stub World** to the six fields per setting, transcribed rather than derived, which is what makes the plan checkable with no wasm toolchain and no Factorio. It is the same discipline `guest/go/tune`'s existing tests already have and it is the reason the settings half could be accepted before a dump was run.

**`make check`'s vet of the data guest gains `-tags tinygo.wasm`.** That is not a preference: without it the vet fails, and the failure is in the friction list below.

## Round two, 2026-09-01: the prototypes follow

Round one closed with four asks at the end of this note, and FkRecipes head `b43c2c7` answers all four: `LegacyItem`, `LegacyRecipe` and `LegacyTechnology`; an `Order` field on all three specs; a `PlaceResult` field on `ItemSpec`; and `RecipeSpec.ResultNamed` for a recipe whose result the plan does not declare. It also split `Emit` into `EmitSettings` and `EmitData`, added host stubs so a consumer's ordinary `go vet` type-checks its own data guest, narrowed `PlanSettings` to a one-method `Named`, and exported `LocaleEntries`.

**The baseline this round was measured from**, on the untouched tree at `8038187` against FkRecipes `b43c2c7` and FkLua `32eb628` (`vcs.modified=false`), Factorio 2.0.77:

| gate | |
|---|---|
| `make check` | **exit 2**, and the failure is a library change: `tune/plan_test.go:166:30: cannot use planWorld{} (value of struct type planWorld) as fkrecipes.World value in argument to Plan().PlanData: planWorld does not implement fkrecipes.World (missing method EntityExists)`. See the friction list |
| `make datastage-check` | exit 0, eleven arms ok -- **over a STALE package**, see below |
| `go mod tidy -diff` | exit 0, no change needed |
| `make test`, the 2.1 golden rows | **NOT RUN**, and for round one's reasons unchanged |

**The stale package is the first finding of the round and it is the Makefile's.** `DATA_GUEST_SRC` listed this repository's own files, so `make mod` did not relink `dist/bbbdata.wasm` when the LIBRARY moved -- and the library is consumed from a working tree through a directory `replace`. The green baseline above was therefore a measurement of a data guest built against a library six commits older than the one on disk. `DATA_GUEST_SRC` tracks `../FkRecipes/go`'s non-test sources now. FkLua's `guest/go` is deliberately left untracked: the same hazard exists for it, its pin moves through `fklua.lock` and `gen-bindings --check`, and one dependency at a time is one change at a time.

### Commit 1: the item

`guest/go/data/item.go` is deleted and the item is one call in the plan:

```go
lib.LegacyItem(PartName, fkrecipes.ItemSpec{
	Icon: PartIcon, IconSize: 64, StackSize: 50, Subgroup: "belt",
	Order: PartOrder, PlaceResult: PartName,
})
```

**All eight fields cross and the dump does not move**: `make datastage-check` is green on all eleven arms with `base data_raw_sha256 f4fbcaa603abc93b`, `base mod_settings_sha256 196275f867f7f8b5`, `incumbent data_raw_sha256 e7001bf98d6c6771` and `incumbent mod_settings_sha256 196275f867f7f8b5` -- the same four figures round one's settings arm produced, which is the whole claim. On the host, `TestTheItemIsTheOneThatShipped` compares every one of the eight against literals transcribed from `ref-item-bbb-balancer-part.json`, and `checkOnly` names a ninth if one appears.

**`PlaceResult` is presence probed, which makes the `fk_data` hook's order a requirement rather than the old file order surviving.** `entity()` defines `bbb-balancer-part` and `EmitData` runs after it; a pass that reorders those two gets a refusal naming the declaration rather than the engine's assignID abort naming the item, and `TestThePlaceResultProbeRefusesAnAbsentEntity` pins the sentence verbatim:

```
fkrecipes: the item bbb-balancer-part names a place_result bbb-balancer-part that does not exist
```

**`EmitSettings` and `EmitData` replace `Emit`, and the vet flag is gone.** Each names the half it runs, so the linker can drop the other; each raises if it is routed from the wrong stage. `make check`'s data-guest vet is a plain `go vet ./data/` again.

**The names live in `guest/go/tune` now** -- `PartName`, `TechName`, `PartIcon`, `PartOrder`, `TechOrder` -- because three files have to agree about them and the plan is the one that emits them. `entity.go` and `legacy.go` name the constants rather than repeating the literals, which is what makes "an entity whose `minable.result` names an item nobody defined" unwritable rather than merely unlikely.

**Red-proven.** `PartOrder` moved from `c[splitter]-y[bbb-balancer]` to `c[splitter]-z[bbb-balancer]`:

```
plandata_test.go:120: bbb-balancer-part's order is "c[splitter]-z[bbb-balancer]" and shipped as "c[splitter]-y[bbb-balancer]"
```

and the gate, with **`mod_settings_sha256` still ok on both arms**, which is what says the two hashes watch different halves:

```
FAIL base data_raw_sha256:
  golden f4fbcaa603abc93b96c025f7af5b280d685395462d040c77cece6192cc8ff12a
  got    347d908ca439096fc02ed24f2259a303f1affc54269071cd744ea6a9301f4087
  note base: prototype list checksum moved 790230733 -> 143489279 (a prototype appeared or vanished)
FAIL incumbent data_raw_sha256:
  golden e7001bf98d6c6771b75be53361fda6d52587cac10cd9a7ba035a2a8261de4edf
  got    8db3a55f473972a346cdb373d0436d8cc241937cf9d09a61d23436a702022de7
```

Restored, and green again on all eleven arms. **One make hazard met on the way and worth writing down**: `sed -i.bak` leaves the restored file with the ORIGINAL mtime, which is older than the wasm built from the drifted one, so `make mod` declined to rebuild and the gate reported the injected failure over a restored source. `touch` before rebuilding, or the red proof's own restoration is what the next run measures.

### Commit 2: the recipe and the technology, together

They cross in ONE commit and the coupling is the library's `enabled`: it is emitted as `!unlocked`, where unlocked means a technology IN THE SAME PLAN lists the recipe in `Unlocks`. `RecipeSpec` has no `Enabled` field and `Extra` refuses a key the library emits itself, so a recipe migrated while its technology is still hand-rolled comes out `enabled = true` -- craftable from the first minute -- and moves the data hash. Graded AWKWARD below.

**The two verbs this mod wanted the library for.** `IngredientsBy` takes one `IngredientChoice` per allowed value of `bbb-recipe-cost`, each ingredient an `IngredientNamed` ladder built from `RecipePlan` rung for rung; `CostBy` takes one `CostChoice` per value of `bbb-tech-cost` whose sources are `TechLadder`, plus the fallback unit. `guest/go/data/recipe.go` and `technology.go` are deleted, and with them the predicate, the twenty-one-entry item type list, and the two `[BBB]` fallback log lines round one recorded as unwatched by any gate. The library's own sentences replace them.

**Both hashes unmoved, all eleven arms ok**, and the field-by-field statement is stronger than the hash this time: the three prototypes read out of the engine's own dump and normalised with `jq -S` are **IDENTICAL to the pre-migration references**, `ref-item-bbb-balancer-part.json`, `ref-recipe-bbb-balancer-part.json` and `ref-technology-bbb-balancer.json`, all three. **And the load says nothing**: a `--dump-data` over the packaged mod carries no `fkrecipes:` line, no `fklua:` line and no `[BBB]` line at any of the three stages.

**One behaviour is narrower and it is on the record.** The library validates `CostChoices.Fallback` before it resolves anything, so the fallback's science pack is presence probed whether or not the fallback can be reached. Measured:

```
fkrecipes: the technology bbb-balancer prices itself in automation-science-pack, which does not exist
```

in a game whose `logistics` is present and carries a unit, so the fallback is unreachable by construction. The hand-rolled technology loaded there. It is not a new failure class -- a fallback that DID fire in such a pack emitted a unit naming a missing item, which is the engine's own assignID abort -- and there is no way to declare a `CostBy` with no fallback, a zero `UnitSpec` being refused for its count. `TestTheFallbackPackIsProbedEvenWhenUnreachable` pins the sentence, so the day the library probes lazily this repo is told. **That day is 2026-09-07, FkRecipes c7a806e**: the `Fallback` is resolved only where it applies, the sentence above is unreachable, and the pin is INVERTED into `TestAnUnreachedFallbacksPackIsNeverProbed` (a game with no science pack at all loads on logistics' own unit) beside `TestAReachedFallbackWithNoPackInTheGameIsRefused`, which is where the pack question moved: `fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one`.

**Red-proven three times, and the three catch different things.**

| injected | what fired |
|---|---|
| `cheap`'s iron plate 2 -> 3 in `RecipePlan` | the host test names the list, `plandata_test.go:250: cheap is made of [{iron-plate 3} {transport-belt 1}] and the gate asserts [{iron-plate 2} {transport-belt 1}]`, AND the gate's own arm: `FAIL recipe-cheap: the recipe is [('iron-plate', 3), ('transport-belt', 1)] and `cheap` should be [('iron-plate', 2), ('transport-belt', 1)]`, with both golden hashes still ok -- the variant arms and the goldens watch different things |
| `Unlocks` removed from the technology | `plandata_test.go:210: the recipe's enabled is true and shipped as false` and the technology's effects gone, AND **both** `data_raw_sha256` arms move (`got 1242246ce26f215e...` on base, `e779bfbc694771d4...` on incumbent) with `mod_settings_sha256` ok on both and every variant arm still green, because they read the ingredient list rather than `enabled`. That pair is the coupling above, measured |
| the fallback `Count` 20 -> 25 | `plandata_test.go:460: the fallback unit is {count=25, time=15, ingredients=[["automation-science-pack", 1]]}, want {count=20, ...}`. The goldens cannot see it: the fallback is unreachable in base |

Each injected, observed and restored, with the restored file byte-compared against the copy taken before the injection.

### Commit 3: the resolver the library owns is gone

`tune.Resolve`, `tune.ResolveRecipe` and the `Ingredient` type are deleted, and with them the five tests that pinned what the library now pins: `TestResolveNeverEmitsAnUnprovenName`, `TestResolveWithNoPredicateEmitsNothing`, `TestEveryLadderResolvesInTheWorstWorldThereIs`, `TestNothingAtAllIsAnEmptyRecipeRatherThanAnInventedOne` and `TestTheIdentityPlanIsTheLastResort`. Every one of their properties is asserted in `plandata_test.go` against a fixture World, one layer out, on the code that actually ships.

**What stays, and one of the two is not redundant with anything the library does.** `RecipePlan` and `TechLadder` are the ladder DATA. `RecipeOptions`, `TechOptions` and the two defaults are the option lists whose head IS each default. `FallbackUnit` is the vanilla research cost. `speed.go` is untouched, being no library concern at all. And **`TestEveryLadderTerminates` stays because the library cannot make its claim**: what FkRecipes guarantees is that it emits no name the game lacks, which an EMPTY recipe satisfies vacuously; a ladder ending at something every game with belts in it has is what stops that being the answer. `TestVanillaIsTodaysRecipe` stays too, re-pointed at the plan, so the same transcribed literal is now compared against a different machine.

**No emitted byte moves here and the goldens therefore cannot**, which is the honest shape of this commit: `Resolve` was already unreferenced from the guest after commit 2, so the packaged data module comes out at **3,122,875 bytes of Lua**, the same figure to the byte, and `make datastage-check` is green on all eleven arms with the same four hashes. The gates are run anyway, because "cannot move" is a prediction until it is a measurement.

### Commit 4: one parser over the locale file

`guest/go/tune/locale_test.go` carried a thirty-line INI reader for the three assertions the library deliberately does not make: that both cost settings have non-blank descriptions, that the hand-rolled bool has its own `[mod-setting-name]`, and that the label quoted inside the `[bbb] single-edge-grandfathered` message is a prefix of that bool's menu row. Round one graded the missing accessor AWKWARD; `fkrecipes.LocaleEntries` is it, and the reader is deleted.

**The point is not thirty lines, it is that two parsers over one grammar can disagree silently**: the checker would pass and this mod's own assertions would be made against a different reading of the same bytes. `LocaleEntries` is `CheckLocale`'s own parser's output, exported, skipping a malformed line exactly as the checker skips it, so the two readings are one by construction. `CheckLocaleWith` and the manifest-name test are untouched.

**Red-proven twice**, each injected, observed, restored, with `git diff --quiet -- mod-data` exit 0 afterwards:

| injected | what fired |
|---|---|
| the `[mod-setting-description]` line for `bbb-tech-cost` deleted | `locale_test.go:196: no [mod-setting-description] bbb-tech-cost: the settings menu shows the label with no tooltip under it` |
| the label quoted inside `single-edge-grandfathered` changed to "Multiple belts per part" | `locale_test.go:256: [bbb] single-edge-grandfathered tells the player to turn off "Multiple belts per part" and the row in the menu is called "Allow multiple belts per balancer part (Factorio 2.0 only)": the message names an entry that is not there` |

### What round two cost

**The control is round one's code rebuilt on TODAY's library, at the real path**, because FkRecipes moved six commits between the rounds and grew every surface round two consumes. Round one's `guest/go/data`, `guest/go/tune` and `Makefile` were checked out into the real tree, built from `make clean`, measured and put back. The path is part of the control: the same round-one code built in a scratch clone under a longer path gives a 1,304,595 B `bbb.wasm` against the real tree's 1,302,885, TinyGo writing the module path into the debug sections.

| | round one, recorded | the control | round two | vs the control |
|---|--:|--:|--:|--:|
| the zip | 665,986 | 673,731 | **662,803** | **-10,928 B, -1.62%** |
| `fk_data_module.lua` | 3,216,828 | 3,349,529 | **3,123,134** | **-226,395 B, -6.76%** |
| `dist/bbbdata.wasm` | 545,651 | 584,688 | **571,382** | -13,306 B, -2.28% |
| `fk_module.lua` | 3,148,568 | 3,148,568 | **3,148,568** | byte-identical |
| `dist/bbb.wasm` | 1,302,885 | 1,302,885 | **1,302,885** | byte-identical, `dfeb456a58f88e2d` |
| members / events / defines | 56 / 24 / 4 | 56 / 24 / 4 | 56 / 24 / 4 | unmoved |
| data-module functions | 85 | 92 | **84** | |

`PlanData` is **23,288** emitted Lua lines, 32.9% of the module, against the control's **23,493** and round one's recorded 21,047: the growth is the LIBRARY's between the two rounds and round two is 205 lines under the control. It is 82.8% of the way to the 28,139-line function that once failed Lua 5.2's parser, no jump-span advisory fired, and the consumer still has no remedy. What changed about it is that it RUNS now.

`PlanSettings` is absent from the map, inlined into `main.onSettings` (3,460 lines). `tune.Plan` is 4,129, `main.settings` 440 against the control's 4,982, `main.onData` 173.

**Stage timing**, five interleaved `--dump-data` runs per package, medians of the engine's own stage timestamps:

| stage | the control | round two | |
|---|--:|--:|--:|
| `settings.lua` | 0.069 s | **0.065 s** | -0.004 |
| `data.lua` | 0.079 s | **0.102 s** | **+0.023** |
| `data-final-fixes.lua` | 0.177 s | **0.176 s** | -0.001 |
| settings start to the prototype checksum | 0.435 s | **0.449 s** | **+0.014** |

The settings stage got cheaper because a 6.8% smaller module is parsed three times; the data stage pays for the whole plan, including the cycle walk over every technology in the game. Net, about fourteen milliseconds per game load, once, on a path with no tick in it.

**No host-call count is given because there is no instrument.** Neither `fklua mod`'s report nor Factorio's log counts what a data stage calls, and `fkdata` keeps no counter, so the walk's cost is visible only as the 23 ms above.

### Round two friction, graded

Same grades as round one: CLEAN worked as documented, AWKWARD worked but cost something, MISLED pointed the wrong way, BLOCKED needed a workaround or could not be done.

**NOTHING IS BLOCKED THIS ROUND, and four of round one's five BLOCKED entries are closed by name.**

#### `World` grew a method and broke the consumer's host stub: AWKWARD

The round-two baseline's `make check` exits 2 on the untouched tree:

```
tune/plan_test.go:166:30: cannot use planWorld{} (value of struct type planWorld) as fkrecipes.World value in argument to Plan().PlanData: planWorld does not implement fkrecipes.World (missing method EntityExists)
```

`World` is an EXPORTED INTERFACE a consumer must implement to hold its plan up to the light on the host, so every method added to it is a compile break in every consumer's test tree. That is the ordinary Go cost of an interface a library asks consumers to implement, it is what an unreleased library is for, and the fix here was six lines. It is graded rather than waved through because the same addition after a tag would be a breaking change in a library whose version says otherwise, and because the mitigation is cheap: an embedded `Named` already exists for the settings half, and the same shape (a small required core plus optional probes the library can default) would make the next method additive.

#### `PlanSettings` takes `Named`, one method: CLEAN, and it closes round one's finding

Round one graded "`World` is ten methods and `PlanSettings` asks one of them" AWKWARD, with nine stub methods kept only to satisfy the interface. `planWorld` is one method now. The fix is exactly the one the note asked for.

#### Host stubs for the emit layer: CLEAN, and it closes round one's BLOCKED

`go/guest_host.go` gives `Emit`, `EmitSettings` and `EmitData` host counterparts that panic naming the cause, so `make check`'s data-guest vet dropped `-tags tinygo.wasm`. Round one graded the absence BLOCKED because the failure was a compile error naming a documented method in the one gate that exists to catch compile errors. The header of that file cites this mod's Makefile line by name, which is the ask landing.

#### `EmitSettings` and `EmitData`: CLEAN, and it no longer helps this consumer

The split does what round one asked for: name the half you use and the linker can drop the other. **This mod stopped being the consumer it was written for in the same round**, because a mod with both a settings plan and a data plan links both planners whatever it calls. `PlanData` is 23,288 lines of the packaged module and it RUNS now, which is the honest resolution of round one's "one function that never runs".

#### `LegacyItem`, `LegacyRecipe`, `LegacyTechnology`, `Order`, `PlaceResult`: CLEAN, and they close three BLOCKED entries

Round one's three fatal refusals were the prefix, the missing `order` and the missing `place_result`. All three are library surface now and all three carried this mod's prototypes across with **both golden hashes unmoved and the three prototypes byte-identical to their pre-migration references after `jq -S`**. Nothing had to be worked around and nothing had to be repointed.

#### The recipe and the technology cannot migrate one at a time: AWKWARD

`enabled` is emitted as `!unlocked`, where unlocked means a technology IN THE SAME PLAN lists the recipe in `Unlocks`. `RecipeSpec` has no `Enabled` field, and `Extra` refuses a key the library emits itself, so there is no way to say "enabled = false, the technology that unlocks me is somebody else's problem for one commit". A recipe migrated alone comes out craftable from the first minute and moves the data hash.

The RULE is documented (`docs/usage.md`: "A recipe some technology unlocks is emitted with `enabled = false`... A recipe nothing unlocks is emitted enabled"). What is not is its consequence for `docs/migration.md`'s step 2, which reads "Move the prototypes across with `LegacyItem`, `LegacyRecipe` and `LegacyTechnology`" as though a prototype at a time were available. One sentence saying a recipe and the technology that unlocks it are one step would have saved this round a measurement.

#### `PlaceResult`'s probe constrains the consumer's own hook body, and the docs say "beside": AWKWARD, mild

The probe is documented and correct, and its refusal names the declaration:

```
fkrecipes: the item bbb-balancer-part names a place_result bbb-balancer-part that does not exist
```

What it means for a consumer whose entity is hand-rolled IN THE SAME HOOK is that the entity must be emitted BEFORE `EmitData` runs, which is a constraint on the order of statements inside somebody's `fk_data`. `docs/migration.md` says "The entity stays hand-rolled beside the `Emit` call"; "before" is the word that would have said it. The cost is small because the failure is loud, and this mod's `main.go` now carries the reason at the call site.

#### The `CostBy` fallback is validated whether or not it is reachable: AWKWARD

`docs/usage.md` says so plainly: "a science pack that does not exist is refused before anything is emitted", which is why this is not MISLED. Measured here, in a game whose `logistics` is present and carries a unit, so the fallback cannot be reached:

```
fkrecipes: the technology bbb-balancer prices itself in automation-science-pack, which does not exist
```

**It is a LOAD THAT USED TO SUCCEED AND NOW REFUSES**, in a real if rare world: an overhaul pack that keeps `logistics` with a unit and has no `automation-science-pack` item. The hand-rolled stage loaded there and copied logistics' own unit; the branch refuses at plan time. That is the one class of change `docs/migration.md` exists to warn about, and it warns about names.

**And a migrating consumer cannot avoid it**, which is what moves the grade off CLEAN: a zero `UnitSpec` is refused for its count, so a `CostBy` always carries a fallback, and a fallback is always probed. There is no way to say "this is the cost of last resort, so ask about its pack only if it is reached".

**The ask, precisely.** Either probe `Fallback`'s packs only when the fallback is REACHED, or accept a pack LADDER in `Pack.Name` the way `IngredientNamed` does for an ingredient, so that an unreachable fallback cannot refuse a load and a reachable one still cannot name something absent. `TestTheFallbackPackIsProbedEvenWhenUnreachable` is the pin and its header carries the grade. **ANSWERED IN FULL ON 2026-09-07, FkRecipes c7a806e, and it is BOTH halves of the ask rather than either**: the packs are probed only when the fallback is reached -- "THE FALLBACK IS RESOLVED ONLY HERE, which is the point: its packs are probed when the fallback is what applies, and never when a source answered" (`go/data.go:598`) -- and `Pack` grew a `Fallbacks` ladder tried in order through `ToolExists`, "a pack no rung resolves is DROPPED with a log line, exactly as an ingredient is" (`go/lib.go:649`). The fallback's NUMBERS are still validated eagerly (`go/data.go:330`), which is the right split: a count of zero is the consumer's own mistake and needs no World to find. The failure class did not vanish, it moved to the only game that has to answer for it -- a unit that loses every pack is REFUSED rather than emitted free, `fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one` -- and this mod declares its fallback pack with no ladder under it, so a game without red science is a load it fails, deliberately. The pin is now two tests, `TestAnUnreachedFallbacksPackIsNeverProbed` and `TestAReachedFallbackWithNoPackInTheGameIsRefused`.

#### `max_level` travels with the copied unit: AWKWARD

`CostBy` copies the source technology's `max_level` beside its unit. It lives on the TECHNOLOGY rather than in the unit, and the hand-rolled `researchUnit` this replaced read three fields of the unit and nothing else, so this is a behaviour change and not a fidelity improvement.

**Measured on the engine**, with a scratch mod setting `data.raw.technology["logistics-3"].max_level = 3` and `bbb-tech-cost = "logistics-3"`: the hand-rolled stage emits `bbb-balancer` with no `max_level`, the branch emits `"max_level":3`. In that pack this mod's research becomes a three-level technology: the unlock fires at level one and the other two are no-ops a player pays for.

It is the library's documented behaviour and it is the right default for `CostOf`, whose whole promise is one named point for cost and position. For a technology that unlocks ONE recipe it is not what the consumer wants, and there is no way to decline it. **The ask**: an opt-out on `TechSpec`, or copying `max_level` only when the source's unit carries a `count_formula`, a fixed-count multi-level source being the odd shape rather than the ordinary one. `TestTheCopiedUnitCarriesTheSourceMaxLevel` is the pin; the stock-game counterpart is already covered by `TestTheTechnologyIsTheOneThatShipped`'s field list.

#### The verbatim unit copy fixed a load this mod used to break: an IMPROVEMENT, recorded

The same copy that brings `max_level` also brings everything else, and the hand-rolled three-field read could not. **Measured on the engine**, with `logistics-2` given `unit.count_formula = "100"` and no `count`:

```
Error while loading technology prototype "bbb-balancer" (technology): Key "count_formula" not found in property tree at ROOT.technology.bbb-balancer.unit
```

`rc=1` on the hand-rolled stage; `rc=0` on the branch, with `"unit":{"count_formula":"100",...}`. A multi-level source technology is exactly the shape that produced it, so the two findings above and this one are one library decision seen from three sides. `TestACountFormulaUnitSurvivesTheCopy` pins it, half against the source and half against transcribed literals, because a verbatim-copy claim compared only against its own source cannot fail by moving the source.

#### An unknown stored option reaches the library rather than this mod's default arm: a finding about the comments, and a small library ask

`tune/recipe.go` said an unknown option "falls back to vanilla rather than to nothing" and `tune/tech.go` said it "gets the default's ladder". Neither is true of the library path: `recipeChoices()` and `techChoices()` build one choice per allowed value, so an unknown string reaches `choiceFor`/`sourcesFor`, gets nil, and comes out as a recipe with NO ingredients and no log line, and a technology on the `Fallback` unit with no prerequisite. Measured on the host against the real path:

```
RECIPE ingredients=[] len=0
TECH unit={count=20, time=15, ingredients=[["automation-science-pack", 1]]} prereq present=false
LOG 0: fkrecipes: bbb-balancer: no source for the not-an-option cost carries a unit, so the fallback cost applies and the technology has no prerequisite
```

**THE ENGINE IS WHAT MAKES IT UNREACHABLE, measured on both packages.** A `mod-settings.dat` carrying `bbb-recipe-cost = "not-an-option"` and `bbb-tech-cost = "not-an-option"` is silently RESET to the defaults by Factorio 2.0.77 before the data stage runs, with no log line, on the branch and on master alike: the recipe comes out vanilla and the unit 20 x 15 automation science after `logistics`. A wrong TYPE is the loud case:

```
Error StringSetting.cpp:71: Failed to load mod mod setting (bbb-recipe-cost): Value must be a string in property tree at ROOT.startup.bbb-recipe-cost.value.
```

So both comments are rewritten to say that, the `"a-value-from-a-newer-build"` case is deleted from `TestTechLaddersWalkDown` (it asserted data no shipped path consults), and `TestAnUnknownOptionIsWhatTheLibraryDoesToday` records what the library does behind the engine's guard. **The library ask below was CLOSED on 2026-09-07 by FkRecipes c7a806e**, and not by the arm asked for: a stored value no dropdown offers is REFUSED BY NAME rather than routed to the default's plan -- `fkrecipes: bbb-recipe-cost holds "not-an-option", which is not one of its values`, the emitted name, which for these two Legacy settings is the unprefixed one -- with `go/data.go:1298` naming this mod as the reason ("This is the pilot's own finding, closed."). A refusal beats the default's plan for the same reason it beats the empty recipe: the state is reachable only by hand-editing `mod-settings.dat`, and a balancer part craftable out of nothing is worse than a mod that does not load. `TestAnUnknownOptionIsWhatTheLibraryDoesToday` is deleted with the behaviour it recorded and `TestAnUnofferedStoredValueIsRefusedByName` pins the sentence, for both dropdowns, one at a time because the library reports the first refusal its walk found.

**The library ask is one arm**: `choiceFor` returning nil for a non-default value should take the "the default applies" arm and its log line, the way a plan that resolved to nothing does. Today that arm is skipped, because it is gated on `len(declared) > 0` and an unknown value declares nothing -- so the one state that produces a silently empty recipe is the one state nothing says anything about.

#### `LocaleEntries`: CLEAN, and it closes round one's AWKWARD

Round one asked for a parsed reading a consumer's own assertions could use, and got one. `guest/go/tune/locale_test.go`'s thirty-line INI reader is deleted; the checker and this mod's three extra assertions are one reading of one file now.

#### `Extra` and `ResultNamed` were not needed, which is worth saying

Every field of all three prototypes has a named slot. Nothing went through `Extra` and nothing was left unplaced, so round one's field-by-field DROPPED list is empty this round. `ResultNamed` is not used either: this recipe produces an item the same plan declares, so it takes the handle.

#### Consuming a library from a working tree is a build-graph hazard: not the library's, recorded anyway

`DATA_GUEST_SRC` listed this repository's own files, and FkRecipes resolves through a directory `replace`, so `make mod` did not relink the data guest when the library moved. The round-two baseline's `make datastage-check` was green over a package built against a library six commits older than the one on disk. The Makefile tracks `../FkRecipes/go` now. It is BBB's defect and it is written down here because it is a consequence of the arrangement the library's own README prescribes for an unpublished module, so the next consumer will meet it.

## Round three, 2026-09-07 and 2026-09-08: both dropdowns gain a Custom value

FkRecipes' customizer round (its `agents/customizer-design.md`, resolved 2026-09-07, head `c7a806e`) added a text setting holding an ingredient language, a pack-list twin of it for research, `IntSetting`/`DoubleSetting` bindings for a research count and time, a `Custom` arm on `IngredientChoices` and `CostChoices`, and `UnimplementedWorld`. This round is the pilot taking all of it: `bbb-recipe-cost` gains `custom` and `bbb-recipe-ingredients` beside it (commit 50c0db3), `bbb-tech-cost` gains `custom` and `bbb-tech-packs`, `bbb-tech-count` and `bbb-tech-seconds` beside it (this commit). The target state was one sentence: a player on the last public release, 0.3.2, with any stored value of either dropdown updates and sees the same recipe and research they had, and can now write both. Every claim below is on Factorio 2.0.77 (build 84539), FkRecipes `c7a806e`, FkLua `a1fcd04`, both sibling checkouts clean.

**The baseline this round was measured from**, before a line was written: the untouched tree at cb89891 against the same siblings.

| gate | |
|---|---|
| `make check` | **exit 1**, at `fklua gen-bindings --check`: the committed bindings were generated by an older fklua than the one `../FkLua/bin/fklua` is built from (a1fcd04). Regenerated as 08320d9; nothing hand-written moved, `fklua.lock`'s api and api_sha256 unchanged |
| `make datastage-check` | exit 0, eleven arms, base data_raw `1e1fcf4f56f5ef22`, incumbent `e7001bf98d6c6771`, mod_settings `196275f867f7f8b5` on both |
| `go test ./tune/` | **compile failure**: `fixtureWorld does not implement fkrecipes.World (missing method FluidExists)`, once per `PlanData` call site. The fixture now embeds `UnimplementedWorld`, which is what the library built for exactly this (0f8a70f) |

### What migrated, and what still cannot

| what | verdict | why |
|---|---|---|
| `bbb-recipe-cost` gains `custom` (seventh, last); `bbb-recipe-ingredients`, a text setting, order `aa` | **MIGRATED**, `IngredientChoices.Custom` with `LegacyIngredientsSetting` (superseded in the sync pass, below: `IngredientsSetting` behind `OrderAfter("a")`, emitted `better-belt-balancer-recipe-ingredients` at `aab`) | The dropdown keeps its six values in their positions, so every stored value survives; the text ships as the word `default`, which the library resolves to the declared list WITH its ladders, so a silent player's recipe cannot drift across releases |
| `bbb-tech-cost` gains `custom` (fourth, last); `bbb-tech-packs` (`ba`), `bbb-tech-count` (`bb`, int, 1 to 1,000,000), `bbb-tech-seconds` (`bc`, double, 1 to 3600) | **MIGRATED**, `CostChoices.Custom` with `LegacyPacksSetting`, `LegacyIntSetting`, `LegacyDoubleSetting` and a `Position` ladder `logistics-3, logistics-2, logistics` (superseded twice, below: in the sync pass by the three ordinary constructors behind `OrderAfter("b")`, emitted `better-belt-balancer-tech-packs` at `bad`, `-tech-count` at `bae`, `-tech-seconds` at `baf`; and in fix round 1 by the flip, where `Position` becomes `TechOptions()` itself, `logistics, logistics-2, logistics-3`) | The three tiers keep their positions; the fields default to `FallbackUnit` (20 automation science over 15 s), so Custom untouched is the base game's Logistics cost; a written cost hangs off the first rung the game has or off nothing with a line |
| the four new settings' NAMES | **`Legacy`, deliberately**, though nothing has shipped them | A generated order is two letters from the declaration index and the consumer cannot choose where it lands among legacy orders (measured at indices 0, 2, 25, 26, 52); for the research fields it would put all three ABOVE their dropdown. Graded below |
| `bbb-multi-edge-parts` | **HAND-ROLLED**, as in round one | Runtime-global, 2.0 only, written by the mod at runtime; the library has no verb for it |
| `Extra`-carried `enabled`; a `Fallback` naming no pack | **NOTHING TO MIGRATE** | The plan carries no `Extra` at all (the recipe and the technology crossed in one commit in round two, so no hand-rolled `enabled` ever existed), and `FallbackUnit` has always named `automation-science-pack` |

### The release-to-head proof

The baseline is the last public release, tag 0.3.2 (cf5a78e), rebuilt in a worktree on FkLua a1fcd04 and dumped under every stored preference it can hold, on both mod sets, by the gate's own `run_arm` (a scratchpad driver over `test/check-datastage.py`, so the staging, the private user directory, the stamping and the `jq -S` normalisation are the gate's and not a reimplementation). The `mod-settings.dat` the engine REWROTE after each of those runs was kept, one per preference. The head package was then dumped with THOSE NINE FILES installed byte for byte into the staged mods directory, `write_mod_settings` replaced by a file copy, so what the head read is the file a 0.3.2 player's game wrote and not a re-encoding of it. Every prototype this mod owns (every type whose name starts `bbb-`, plus `balancer-part`) was extracted from both dumps and compared field for field, and the whole settings dump beside it.

| stored preference | base: 22 owned prototypes | incumbent: 21 owned prototypes | incumbent `data_raw_sha256` |
|---|---|---|---|
| no file at all (the defaults) | identical but `place_result` (below) | **identical** | `e7001bf98d6c6771`, = 0.3.2 |
| `bbb-recipe-cost` = `vanilla` | identical but `place_result` (below) | **identical** | `e7001bf98d6c6771`, = 0.3.2 |
| `cheap` | the same one field | **identical** | `4959eeb52f1feb4a`, = 0.3.2 |
| `belt-fast` | the same one field | **identical** | `c639ae2fd720447c`, = 0.3.2 |
| `belt-express` | the same one field | **identical** | `5c47f80996635f48`, = 0.3.2 |
| `splitter` | the same one field | **identical** | `82abcb3e853ed223`, = 0.3.2 |
| `splitter-express` | the same one field | **identical** | `a0d927f3d3474299`, = 0.3.2 |
| `bbb-tech-cost` = `logistics` | the same one field | **identical** | `e7001bf98d6c6771`, = 0.3.2 |
| `logistics-2` | the same one field | **identical** | `429788ebaeff6b65`, = 0.3.2 |
| `logistics-3` | the same one field | **identical** | `3bbdb6d95ace4722`, = 0.3.2 |

That is ten configurations, twenty (configuration, mod set) pairs: the nine stored values a release could hold plus the no-file default, which is the same state as `vanilla` and `logistics` and is kept as the control that the installed files were read.

**The one moved field is explained, not re-recorded**: on the base arm `item/balancer-part/place_result` reads `balancer-part` where 0.3.2 read `bbb-balancer-part`. That is cb89891, the legacy stub item placing the stub entity instead of this mod's part, the freeze fix already in the 0.3.3 changelog; the incumbent arm defines no stub and has nothing to move. Nothing else on any prototype of this mod's moved for any stored preference. The base-arm `data_raw_sha256` values differ from 0.3.2's by exactly that field (the recipe and the unit under each preference are byte-identical), and the installed files were read: `recipe-cheap` on the head hashes `633884a749689dea`, not `recipe-vanilla`'s `1e1fcf4f56f5ef22`.

**The settings dump moves in eight places, the same eight on all twenty (configuration, mod set) pairs**, and nowhere else: `bbb-recipe-cost/allowed_values` gains `custom`; `bbb-recipe-cost/localised_description` appears, composed from the six presets rendered in the language; `bbb-tech-cost/allowed_values` gains `custom`; `bbb-tech-cost/localised_description` appears, one line per tier as `cost of <source>`; and `bbb-recipe-ingredients` (`aa`), `bbb-tech-packs` (`ba`), `bbb-tech-count` (`bb`) and `bbb-tech-seconds` (`bc`) appear. `bbb-multi-edge-parts` and every other field of the two dropdowns are the release's. The whole-dump hash on both mod sets is `0bea93fa1594052e`, the golden (moved to `e5e7d159e88f4a6b` by the sync pass below, on a change of the library's and not of this plan's, and then to `df4d6c3fc7bf854d` when the four customizer settings took generated names).

**The four custom rows are new goldens**, dumped with texts of this round's own rather than the gate's: `recipe-custom` with `2 iron-plate, 1 fast-transport-belt, 1 iron-stick` (the library's line: `fkrecipes: bbb-balancer-part takes its ingredients from bbb-recipe-ingredients: 2 iron-plate, 1 fast-transport-belt, 1 iron-stick`), `tech-custom` with `2 automation-science-pack, 1 logistic-science-pack` at 40 units of 30 s (`... takes its research cost from bbb-tech-packs: count 40, time 30, packs 2 automation-science-pack, 1 logistic-science-pack`), and both `-default` rows, whose data-raw is the preset's (`recipe-custom-default` hashes `1e1fcf4f56f5ef22`, vanilla's) or the fallback unit after `logistics-3` (`tech-custom-default`, `afb19d7fa0cfd0a5` base; fix round 1's flip moves both to `logistics` and `1e1fcf4f56f5ef22`, the default dump's own hash). The gate's arms keep asserting the transcribed tuples, units and prerequisites: `recipe-custom` expects `[("iron-plate", 3), ("splitter", 1)]`, `tech-custom` expects `{count 50, time 20, [automation 1, logistic 1]}` after `logistics-3` (after `logistics` since fix round 1), `tech-ignored` expects logistics-2's own unit after `logistics-2` beside the ignored line (three lines, whole and in order, since the sync pass). The four customizer settings are named in this paragraph as round three declared them; the sync pass renamed them and the arms assert the generated names now.

### What round three cost

Against the last public release rather than against round two, because that is what a player updates from; both packages on the same machine, the 0.3.2 worktree rebuilt on today's FkLua. The table and the stage timings are in CLAUDE.md ("Round three"); the headline: the zip 584,887 to 822,248 B (+40.6%), `fk_data_module.lua` 1,763,788 to 4,904,124 B (39,056 to 122,031 lines), `bbbdata.wasm` 174,225 to 1,085,173 B, the control guest the same 3,148,568 B with four lines differing (the api signature and the build stamp), and about a fifth of a second more per game load (settings start to the prototype checksum 0.279 to 0.478 s, medians of five interleaved runs).

**The jump relay, which the packager does not report.** `PlanData` is 26,418 emitted Lua lines now, 93.9% of the 28,139-line function that once failed Lua 5.2's parser, and it loads because FkLua's `relayJumps` (`internal/luagen/funclimit.go`) breaks every over-long jump into trampolines. Nothing says so: `fklua mod` prints no line about it, its `--report` JSON has no field for it at a1fcd04 (`data_lua_bytes` is the nearest), and the only instrument is a grep for the labels it emits, `::LT<n>::` and `::LTs<n>::`. The head module carries 26 of them, thirteen stations, every one inside `PlanData` (lines 44,159 to 70,576); the release carries none. A consumer who wants to know whether the relay fired, how many stations it placed, or how close a function is to the span the relay cannot bridge (a single basic block longer than the limit, which the check refuses) has to read the emitted Lua. Filed below.

### Round three friction, graded

Same grades: CLEAN worked as documented, AWKWARD worked but cost something, MISLED pointed the wrong way, BLOCKED needed a workaround or could not be done. **Nothing is BLOCKED.** The first three entries are 50c0db3's, carried forward as that commit's message filed them.

#### A consumer holding legacy orders cannot choose where a generated setting lands: AWKWARD

`orderString(i)` (go/settings.go) makes a generated setting's order from its declaration index alone, `'a'+i/26` then `'a'+i%26`, so with this mod's legacy `a` and `b` every generated setting up to the twenty-sixth sorts between the two dropdowns and the twenty-seventh onward past `b`, measured through the library at indices 0, 2, 25, 26 and 52. For the recipe's text field that is the right place by accident; for the research's three it puts packs, count and seconds above the dropdown that switches them on, and no ordering of the declarations can fix it. The escape is the `Legacy` constructors, which take an explicit order and force the consumer to hand-write the prefixed name too, which this mod did for all four (`bbb-` rather than `better-belt-balancer-`, one namespace instead of a seam). The ask: a way to give a generated setting an explicit order without giving up the generated name.

**Answered at FkRecipes b2e47b6 and adopted in the sync pass, below.** `OrderAfter` places every generated setting declared after it behind the named order with its own two letters, so the four customizer settings are generated names now and the menu reads `a`, `aab`, `b`, `bad`, `bae`, `baf`.

#### `docs/usage.md`: "Settings are emitted in declaration order and given `order` strings from that order, so the settings screen shows them the way you wrote them": MISLED

True for a plan of generated settings only. For a plan that mixes legacy and generated settings the screen shows the generated ones wherever the index letters fall relative to the legacy orders, which is what the entry above measured, and the sentence says nothing about it.

#### `docs/migration.md`'s worked example for this mod uses `IngredientsSetting`: MISLED

The example under "Adding a customizer to a dropdown you already ship" is written around BetterBeltBalancer and declares the text setting with `lib.IngredientsSetting("balancer-part-ingredients", ...)`, and line 265 states the outcome: "The text setting is a new name, so it carries the generated prefix and the mod's default as its text." A consumer following it for the recipe alone lands where the example says; a consumer who then takes step 6's research half lands on a settings menu whose research dropdown sits below its own three custom fields, with no sentence anywhere warning them. This mod measured it and declined the example.

#### `LegacyPacksSetting`, `LegacyIntSetting`, `LegacyDoubleSetting`, `CustomCost` with `Position`, and the two bound refusals: CLEAN

All three constructors take a full name and an explicit order and emit both verbatim; `CustomCost` binds the three handles and a `Position` ladder exactly as `docs/usage.md` says; the count and seconds minima are checked on the DECLARED spec with sentences that name the engine's rule (`... backs a research count but declares no minimum of at least 1 (the engine refuses a unit count of 0)`), and the engine's reset-not-clamp rule makes every readable value legal, as the library measured and this mod's `tech-custom` arm confirms through a type-2 double read as an int. The unit lands in the short tuple form, the log line reads as documented, and the prerequisite follows the ladder rung by rung to none with a line (round three's wording was "down", which was this mod's ladder direction then; fix round 1 turned that ladder round and the library walks whatever order it is handed).

#### An edited count or seconds under a tier draws no line: AWKWARD, mild

`noteIgnoredText` (go/data.go:558) runs for the pack TEXT under a tier and says `fkrecipes: bbb-tech-packs is edited, but bbb-tech-cost is not on custom, so the text is ignored`. The two numbers beside it draw nothing when moved under a tier: measured on the host with the count at 50 and the seconds at 20 under `logistics-2`, the stream is exactly the one pack line (`TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo`), and at the gate (`tech-ignored`). So a player who moves the unit count, sees the research unchanged and opens the log finds nothing about the count. The library's reason is sound for a number EQUAL to its default, which is indistinguishable from untouched; a number that differs from its declared default is not. The ask: the ignored line for a numeric setting whose stored value differs from its declared default while the dropdown is on a tier.

**Answered at FkRecipes da11cf5 and adopted in the sync pass, below.** The measurement in this entry is round three's and is kept as it was made; on the head the same fixture draws three lines, count, seconds, packs, and the host test and the gate arm named above pin all three, whole and in order.

#### The composed description of a cost dropdown shows the source technology's internal name: AWKWARD, mild

**THE TIER LINE QUOTED IN THIS ENTRY IS A DATED RECORD AND BOTH OF ITS HALVES HAVE SINCE MOVED.** The tail was localised at `da11cf5` (the paragraph under this one), and fix round 2's third commit, 2026-09-14, rewrote the LABEL as a name, so the line this entry quotes as `Default: Logistics, alongside transport belts: cost of logistics` renders today as `Logistics 2: cost of logistics-2`. The entry stands as the friction it was, on the shape it was found on. `costPresetText` (go/customize.go:575) renders a tier's line as `cost of <first source>`, so the research dropdown's tooltip ends `Default: Logistics, alongside transport belts: cost of logistics` and `... : cost of logistics-2`. For the recipe dropdown the internal-name rendering is right, because the player types that vocabulary into the field beside it; for a cost dropdown the player never types a technology name, so the internal name buys nothing and reads as a raw key beside a localised label. The ask: render the source as its localised name (`{"technology-name.<name>"}`), or let a consumer opt out of the composition for a cost dropdown. The composition itself, and the checker demanding the description that anchors it, are correct.

**Answered at FkRecipes da11cf5 and adopted in the sync pass, below.** `costPresetText` is `costPresetTail` on the head (go/customize.go:725) and a tier's line ends `": cost of ", {"technology-name.<first source>"}`; the transcription in `tune/plan_test.go` moved with it and the settings golden moved on both mod sets, which is the whole of what this entry cost to close.

#### `UnimplementedWorld`: CLEAN, and it closes round two's AWKWARD

`World` grew two methods this round and the fixture stopped compiling with a Go error naming the first of them, which is the round-two finding again; the embed was the answer the library had built for it, one line, and the next method costs nothing until a plan of this mod's asks it. `FluidExists` did get asked, by the recipe field, and the fixture stocks `water` for the refusal it buys.

#### The relay is silent and unreported: AWKWARD

Recorded above under "What round three cost". The ask: a report field (stations placed, and the largest function's span against the limit) and a line from `fklua mod` when a relay fires, so a consumer can watch a library function approach the span the relay cannot bridge without grepping the emitted module. This is FkLua's rather than FkRecipes', and is filed here because this consumer is where it was measured.

**Answered at FkLua 2a541a7 and b88965d, adopted in the sync pass, below.** `fklua mod` prints the span, the stations, the widest block and the block room under each module's line, `--report` carries them as a `jumps` object per module, and `make mod` keeps the line in `dist/.fklua-mod.log`.

#### The engine writes an int setting as a type-2 double: noted, not friction

`tools/mod-settings.py` wrote every number as a property-tree double (retired in the sync pass, below, for `fklua modsettings write`, which spells an int setting's number as the signed 64-bit); the library measured that the engine reads a type-6 int and a type-2 double alike for an int setting, and the `tech-custom` arm confirms it on this package. Nothing to ask.

**Corrected in the sync pass, below**: the heading names the wrong writer. It was this mod's `tools/mod-settings.py` that wrote the double; the engine reads it and writes the setting back as a signed 64-bit (type 6), read out of its own rewrite.

### Red proofs, in the real tree

Fifteen host injections and two gate injections by the implementer, and ten more host injections by the adversarial review (its report is the round's scratchpad `r3/review-c.md`), each in this mod's own code or locale and none in the library, each observed, restored and cmp-verified; the table is in CLAUDE.md ("Round three"). The review's two MUST-FIX findings (the pack field's fluid refusal reachable and unpinned, with the fixture header silent on the pack list's fluid rule; and two present-tense CLAUDE.md claims left false, the "eleven-arm gate" under the goldens table and the status row saying two settings whose ladders guarantee no unknown name) and its three SHOULD-FIX findings (the tooltip and changelog promising "the highest logistics tier your mods have" where the ladder is three fixed names; "nine preferences" against a measurement of ten configurations; Custom untouched refusing in a pack without the base science pack, unsaid in the player's terms) are all taken in the commit. What the two gate proofs add to the host's: with the `Position` ladder reduced to `logistics`, `tech-custom-default` and `tech-custom` FAIL on the prerequisite while every tier arm stays green; with the count default at 25, `tech-custom-default` FAILS on the unit (`count 25`) and both `mod_settings_sha256` arms fail beside it (a declared default is a field of the prototype the golden hashes), while `tech-custom`, which writes 50, stays green, which is what says the one arm reads the declaration and the other reads the file.

### The client run: NOT REACHABLE from this session, and owed

The round asked for the Mod Settings screen on the Startup tab and a craft in a game. The 0.3.3 package was staged into the user's mods directory with its manifest stamped for 2.0 the way the gate stamps its own (`factorio_version` 2.0, `base >= 2.0.0`; the binary is 2.0.77 and the package is pinned 2.1), `bbb-interactive-setup` disabled for the run, and `mod-list.json` and `mod-settings.dat` backed up first. Control of the Factorio application (`com.factorio`) and of Steam was then requested through the desktop-control channel and DENIED for this session (`request_access` answered `user_denied` for both), so no screenshot, click or keystroke could be made. The staging was reverted: the 0.3.3 directory removed, both files restored and cmp-verified, and no Factorio process was running before or after (pgrep exit 1).

Each observation the round asked for is therefore NOT REACHABLE here, reported as such rather than guessed: whether `\n` renders as a line break in a setting's description tooltip; whether the composed multi-line descriptions are readable there; whether the settings-screen text field has a length limit of its own; whether rich text renders as an icon inside it; whether shift-clicking an item with the field focused inserts a tag (FkRecipes withdrew its own sentence claiming so, because startup settings are edited from the main menu where no inventory exists, and the in-game Mod Settings dialog, which has one beside it, is exactly the case a client run would settle); whether the six labels and the four descriptions agree as rendered; and whether a custom recipe crafts as written. FkRecipes' `agents/customizer-design.md` lists the first five as "claims that a headless probe cannot settle, carried as assumptions until a client run checks them", and they stay carried.

What IS settled without a client, and is what the labels rest on: every locale key the six settings and their eleven dropdown values need exists, in both directions (the library's checker); every sentence a refusal can show a player is pinned word for word on the host, and the recipe field's three are reproduced on the engine; and the two composed descriptions carry exactly the keys and texts the transcription pins, read out of the engine's own settings dump.

The maintainer's checklist for the first client at hand: install `dist/better-belt-balancer_0.3.3` (stamped for the binary's series if it is 2.0), open Settings > Mod settings > Startup and read the six rows in the order `a`, `aab`, `b`, `bad`, `bae`, `baf`; hover each description and look for the line breaks the composed ones carry; set the recipe dropdown to Custom, type `2 iron-plate, 1 splitter` into the field, restart; start a freeplay game, research the balancer (or `/c game.player.force.technologies["bbb-balancer"].researched=true`), and craft a part by hand: the crafting tooltip should list two iron plates and a splitter. Then, in that game, open Mod settings again with an inventory beside it and shift-click an item with the field focused.

### What is NOT RUN on this machine, round three

Unchanged from round two: `make test` (fourteen suites) and the 2.1.16 and 2.1.17 golden rows, because the packaged mod is pinned 2.1 and the binary is 2.0.77; the 2.1 rows carry `_stale` notes naming the three moves (the stub's `place_result`, the recipe customizer's settings, the research customizer's settings) for the next 2.1 session (five on the base arm and four on the incumbent since the sync pass, below). The `bench/` matrix, out of scope.

## Sync pass, 2026-09-08: this mod on FkLua b88965d and FkRecipes 9754f49

Both siblings moved after round three and this is the round that puts the mod on their heads: FkLua master `b88965d`, twelve commits past the `a1fcd04` round three built on, and FkRecipes master `9754f49`, eight past the `c7a806e` round three adopted. Every claim below is on Factorio 2.0.77 (build 84539, re-asked with `"$FACTORIO_BIN" --version` before every gate), both sibling checkouts clean and read-only (a defect or a gap is filed here with its evidence, never patched next door), and `../FkLua/bin/fklua` rebuilt from the FkLua head the way its own Makefile builds it (`cd ../FkLua && go build -o bin/fklua ./cmd/fklua`; `go version -m bin/fklua` reads `vcs.revision=b88965d0875de1771babb0c8fc86c7f767c5a935`, `vcs.modified=false`; `bin/` is that repository's gitignored build output and its `git status` stayed empty). The `a1fcd04` binary was copied aside before the rebuild, as insurance; the toolchain-only column of the size table below is round three's exact wasm files repackaged with the NEW binary, set against round three's recorded figures.

### What moved in the siblings, in this mod's terms

**FkLua, a1fcd04 to b88965d.** A data-stage string is BYTES in both guest libraries (b647414, 9591650): the Go guest's `fkdata.go` moves 38 lines in and 4 out between the two heads (`git -C ../FkLua diff --stat a1fcd04 b88965d -- guest/go/fkdata/fkdata.go`) and FkRecipes' own sync pass read every one of them as a doc comment or a gofmt realignment, so this mod's data guest needed nothing but a relink. `fklua mod --report` carries a `jumps` object per module (2a541a7, b88965d) and the packager PRINTS the relay's figures under each module's line, which is the answer to round three's "the relay is silent and unreported". `fklua modsettings write --from FILE.json --out mod-settings.dat` and `fklua modsettings read` (c21ff07, f80f61b) were a toolchain twin of `tools/mod-settings.py`, which this round's third commit retires for them. Three luagen size commits (58bc28a, 6310005, cf08bd8, gated by a2149b6) shrink the emitted Lua; measured below on this mod's own guests. A 41-case engine probe (7cc0d25) is what the library's measured corrections rest on.

**FkRecipes, c7a806e to 9754f49.** The ingredient language links only when a text setting is declared (3601004; this plan declares two, so it stays linked). `OrderAfter` (b2e47b6) is the answer to round three's "a consumer holding legacy orders cannot choose where a generated setting lands", decided in this round's second commit. An edited research count or seconds under a tier draws a line, and a cost preset's composed line names its technology through the technology's own locale key (da11cf5), which are round three's two mild AWKWARDs, both re-pinned in this round's first commit. Three measured corrections (7026e52), the Rust half on the byte-string surface (947ce44, not this mod's concern), `go/internal/modsettings` retired for `fklua modsettings write` (bf95d92), and both of the library's gates reading the `jumps` row (9754f49).

### The baseline, the untouched tree at 235c1da against both heads

| gate | |
|---|---|
| `../FkLua/bin/fklua gen-bindings --check` | **exit 0**: `every one of them is bound or deferred: go 4865 + 5, rust 4865 + 5`; the committed `guest/go/fkapi/fkapi.go` is unmoved and nothing was regenerated |
| `../FkLua/bin/fklua lock --check` | **exit 0**: `fklua.lock is up to date (api 2.1.17)`; the api pin, the api hash and the bindings hash all stay |
| the Makefile | records no sibling revision (its stamps are the GC and persist MODES), so there is nothing there to move |
| `make check` | **exit 2**, at `go test ./tune/`, on exactly the two library texts: `TestEverySettingPrototypeIsTheOneThatShipped` (`bbb-tech-cost's localised_description is got [..., ": cost of ", ["technology-name.logistics"]] want [..., ": cost of logistics"]`) and `TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo` (`the plan's log stream is [count ... seconds ... packs] want exactly one line`) |
| `make mod` | exit 0 |
| `make datastage-check` | **make exit 2** (the script's own 1): both `data_raw_sha256` arms ok (`1e1fcf4f56f5ef22` base, `e7001bf98d6c6771` incumbent); BOTH `mod_settings_sha256` arms FAIL, golden `0bea93fa1594052e`, got `e5e7d159e88f4a6b9ea8f8f1ca2a1490c5c77e06183d4836c64c6adf5f0b9ded` on both mod sets; all fourteen variant arms and the speed arm ok, `tech-ignored` included, because it asserted its one line by containment |
| the release-to-head proof, re-run | below |

**The release-to-head proof, re-run on the untouched tree against both heads** (scratchpad `sync2/prefs-head2.py`, round three's `prefs-head.py` with one addition: it keeps the `mod-settings.dat` the ENGINE rewrote after each run and reads it back with `fklua modsettings read`). The nine release `.dat` files are installed byte for byte, the no-file default and round three's four custom rows go through the writer, on both mod sets: 28 (configuration, mod set) pairs, `compare-prefs.py` over the owned extracts. Against the 0.3.2 release (`baseline/rel032`): what round three found and nothing else, every owned prototype the release's on all twenty pairs but `item/balancer-part/place_result` on the base arm (cb89891, explained there), and the settings dump moving in the same eight places. Against round three's own head extracts (`r3/head033`): **every owned prototype IDENTICAL on all 28 pairs, every `data_raw_sha256` the same to the digit (`recipe-cheap` base `633884a749689dea`, `tech-custom` base `1cc1d661c9f4874c`, `tech-custom-default` base `afb19d7fa0cfd0a5`, and the rest), every library log line the same**, and the settings dump differing in exactly ONE place per pair: `bbb-tech-cost/localised_description`, each tier's line ending `": cost of ", {"technology-name.<source>"}` where round three's ended `": cost of <source>"`. The library's eight commits reach this mod's prototypes nowhere and its settings dump in one field.

**And the engine writes an int setting as a signed 64-bit, not as a double**, which corrects round three's heading "The engine writes an int setting as a type-2 double". The `tech-custom` row's `.dat` was written by `tools/mod-settings.py` with the count as a type-2 double (40.0); the file the engine rewrote after the run holds `bbb-tech-count` as type 6 (`xxd` of `sync2/head-b0/tech-custom/base-settled.dat`: `... 76 616c 7565 0600 2800 0000 0000 0000 ...`, the `06` after `value` and 0x28 = 40 as a little-endian int64), and `fklua modsettings read` renders it `"bbb-tech-count": 40` beside `"bbb-tech-seconds": 30.0`. The engine READS either encoding, which is what round three's body said and what FkLua's own table measures; what it WRITES is the setting's own type. (Measured on commit 1's package, before the rename; the same rewrite of commit 2's package carries `"better-belt-balancer-tech-count": 40`, `sync2/head-c/tech-custom/base-settled.json`.)

### Commit 1: the two moved texts are re-pinned, and the golden moves with the library

`tune/plan_test.go` transcribes the research dropdown's composed description in the head's shape, through a `costPresetLine(setting, value, source)` helper beside `presetLine` (a second helper because the LINE'S SHAPE differs: a recipe preset's line ends in one rendered string, a cost preset's in two, the words and then the game's own `technology-name` entry for the first rung of that tier's ladder; the name is not `costLine` because two tests already hold a const by that name). `tune/plandata_test.go`'s `TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo` asserts the stream WHOLE and IN ORDER over three arms: both numbers moved (50, 20) gives count, seconds, packs; the count at its declared default (20) gives seconds, packs; the seconds at its declared default (15) gives count, packs. The last two are the half this mod is on the hook for: a number's "untouched" is its DECLARED default and the declaration is `plan.go`'s. `test/check-datastage.py`'s `tech-ignored` arm carries the three lines and compares the whole `fkrecipes:` stream against them in order (`lines == want_lines`), printing one `said` per line. The 2.0.77 golden is re-captured: only `mod_settings_sha256` moved, `0bea93fa1594052e` to `e5e7d159e88f4a6b` on both mod sets, both `data_raw_sha256` unmoved with the normalised dumps byte-identical either side, the `_note` and the four 2.1 `_stale` notes extended in place. Every library line citation in `guest/go/tune` (`plan.go`, `tech.go`, `recipe.go`, `world_test.go`, `plan_test.go` and `plandata_test.go`; twenty-nine added lines carrying a citation, 31 occurrences and 24 distinct strings, by `git diff -U0 -- guest/go/tune | grep '^+' | grep -o 'go/[a-z]*\.go:[0-9]*'`, each opened on the head and confirmed to name the thing it names) is re-pinned to the head's line numbers, and one sentence of `plan.go`'s is corrected: under a tier the three custom fields are READ, to say whether the player moved them, and no longer "not read at all".

**Red-proven six times, each injected in this mod's code or test, observed, restored and the restore cmp-verified** (the implementer's five, the adversarial review's one):

| injected | what fired |
|---|---|
| `tech.go`: the logistics-2 ladder reversed, so its first rung is `logistics` | `TestEverySettingPrototypeIsTheOneThatShipped`: `got ... ["string-mod-setting.bbb-tech-cost-logistics-2"], ": cost of ", ["technology-name.logistics"]] ... want ... ["technology-name.logistics-2"]]` |
| the three-line arm's count written at 20, its declared default | `both numbers moved: the plan's log stream is got [seconds, packs] want the ignored fields in the library's order, [count, seconds, packs]` |
| the expected order swapped, seconds before count | `got [count, seconds, packs] want ... [seconds, count, packs]` |
| at the gate, `tech-ignored`'s `.dat` count set to 20 | `FAIL tech-ignored: the library's log lines are [seconds, packs] and the whole stream, in order, has to be [count, seconds, packs]`, on a real 2.0.77 dump, which is the engine-side confirmation that a count at its declared default draws no line |
| at the gate, the three expected lines reordered | `FAIL tech-ignored: the library's log lines are [count, seconds, packs] and the whole stream, in order, has to be [seconds, count, packs]` |
| `plan.go`: the count's declared default 20 to 21, the tests untouched (the review's own) | the "count at its declared default" arm (`got [count, seconds, packs] want [seconds, packs]`), beside fourteen other tests that transcribe or store the 20 (the settings transcription's `default_value is 21 and shipped as 20`, the fallback-unit identity, the unreadable-number test's unit, and every everything-world arm, whose fixture stores 20 and now reads as an edit); what the new arm adds is that it pins the DECLARATION and not the fixture's value, which the fixture-side injection alone could not say |

The adversarial review (scratchpad `sync2/reviewB/`) found three MUST-FIX items, all taken: the new test's anti-vacuity comment quoted the real game's logistics-2 unit where the fixture's charges one logistic science pack; `plan.go`'s declaration comment said the three fields are "not read at all" under a tier and cited a line the library no longer has; and this note's two round-three entries recorded as open what the head grants. Its SHOULD-FIX items, the stale `presetLine` rationale and the eleven drifted citations, are taken too.

### Commit 2: the four customizer settings are generated names placed by `OrderAfter`, and the engine loads its first three-letter order from the library

**The decision is ADOPT, and it is the one judgment call of this pass.** Round three declared `bbb-recipe-ingredients` (`aa`), `bbb-tech-packs` (`ba`), `bbb-tech-count` (`bb`) and `bbb-tech-seconds` (`bc`) through the `Legacy` constructors although nothing had shipped them, for one measured reason: a generated order came from the declaration index alone and the three research fields sorted above their dropdown (the entry above, and `plan.go`'s header, which keeps the measurement). FkRecipes b2e47b6 answered exactly that ask. Four things decided it, in the order they weigh: the deviation rested on the placement alone and the placement is answered; nothing has shipped these names and the day 0.3.3 ships they are frozen forever (mod-settings.dat is keyed by name, no rename), so this is the only moment the generated names cost nothing; this mod is the pilot `OrderAfter` was built for and the engine had never loaded a three-letter order from the library (`grep -rn 'OrderAfter\|order_after' go/examples rust/examples testdata/mirror` in FkRecipes is empty), so this mod's dump gate is where it first meets one; and the seam a reader sees, four rows spelled `better-belt-balancer-` beside two spelled `bbb-`, is invisible in the settings screen, which renders the localised labels, and appears only in the library's log lines and in mod-settings.dat, where every migrated mod that grows a setting will carry it. The cost, stated: round three's four custom-row extracts in the scratchpad were dumped under the `bbb-` names and are reset by construction, and anybody holding an unshipped 0.3.3 build with a stored custom value loses it, which is nobody outside this machine (round three's staging was reverted).

**What the plan emits**, read out of the plan (`go test ./tune/` before the transcription moved) and out of the engine's own dump, on both mod sets:

| declaration | constructor | emitted name | order |
|---|---|---|---|
| index 0 | `LegacyDropdownSettingNeedingLocale`, order `a` | `bbb-recipe-cost` | `a` |
| `OrderAfter("a")`, index 1 | `IngredientsSetting("recipe-ingredients", ...)` | `better-belt-balancer-recipe-ingredients` | `aab` |
| index 2 | `LegacyDropdownSettingNeedingLocale`, order `b` | `bbb-tech-cost` | `b` |
| `OrderAfter("b")`, index 3 | `PacksSetting("tech-packs", ...)` | `better-belt-balancer-tech-packs` | `bad` |
| index 4 | `IntSetting("tech-count", 20, Between(1, 1000000))` | `better-belt-balancer-tech-count` | `bae` |
| index 5 | `DoubleSetting("tech-seconds", 15, Between(1, 3600))` | `better-belt-balancer-tech-seconds` | `baf` |

The bare names are the ones FkRecipes docs/migration.md works through for this mod, and the two letters are the library's arithmetic (index 1 is `ab`, indices 3 to 5 are `ad`, `ae`, `af`) behind the order named by the `OrderAfter` in force, captured at declaration. The three refusals the library makes of a mixed plan (an empty order; a generated order equal to a legacy one; a placed order sorting past a legacy order that extends the named one) are unreachable from one-letter legacy orders, which `plan.go`'s header says and the review checked against go/settings.go.

**What moved and what did not.** The 2.0.77 golden: `mod_settings_sha256` `e5e7d159e88f4a6b` to `df4d6c3fc7bf854d` on both mod sets; both `data_raw_sha256` unmoved, and the normalised data-raw dumps compared byte for byte either side through a stash (`cmp` exit 0 on both arms). The settings dump moves in eight places per pair, the four old settings gone and the four new ones present with the same fields but the name, the order and the description key. The locale file: eight keys renamed, every label and description byte for byte what it was (the values diffed alone, 56 lines each side, identical). The release-to-head proof re-run on this package (`sync2/head-c`, 28 pairs): against the 0.3.2 release, what round three found (`place_result` on the base arm, eight settings places); against commit 1's extracts (`head-b0`), every owned prototype IDENTICAL on all 28 pairs, every data-raw hash the same to the digit, the settings dump differing in exactly those eight places. README and the changelog say nothing, because a player sees the four labels, which did not move.

**Where the three-letter order meets the engine.** The gate's two golden arms gain a settings-order probe (`check_settings_order`, beside `check_legacy_stub`, one engine run for both): the six orders are read out of the engine's own `mod-settings-dump.json`, asserted by name against a literal table, and their sort by `(order, name)`, the Startup tab's sort, against the declaration order. On 2.0.77 both arms print `ok base settings orders a, aab, b, bad, bae, baf, sorted as declared` and `ok incumbent ...`. The engine accepted and stored all four. The engine's own rewrite of a `tech-custom` `.dat` carries `"better-belt-balancer-tech-count": 40` as a signed 64-bit, as it carried the old name (`fklua modsettings read` of `sync2/head-c/tech-custom/base-settled.dat`).

**The cost of the rename, measured rather than assumed** (scratchpad `sync2/agentC/rename-cost.py`: the gate's own writer and base arm against this package, on 2.0.77). A `.dat` holding `bbb-recipe-cost: custom` and `bbb-recipe-ingredients: 3 iron-plate, 1 splitter` yields `[('iron-plate', 4), ('iron-gear-wheel', 2), ('transport-belt', 2)]`, the vanilla list, and no `fkrecipes:` line at all; one holding `bbb-tech-cost: custom`, `bbb-tech-packs: 2 automation-science-pack`, count 40, seconds 30 yields `{count 20, time 15, [automation-science-pack 1]}` after `logistics-3` (that placement is the sync pass's own; it is `logistics` since fix round 1 flipped the `Position` ladder) with the line `fkrecipes: bbb-balancer takes its research cost from better-belt-balancer-tech-packs: count 20, time 15, packs 1 automation-science-pack`. The dropdown's stored `custom` is honoured, because its name did not move; the four values under names no mod declares are ignored by the engine, silently (FkLua's measured table: "a value under a name no mod declares: ignores it, and leaves it in the file"), which is why a rename is free only before a release.

**Red-proven eight times, each injected in this mod's code, locale or gate, observed, restored and cmp-verified** (the implementer's six, the adversarial review's two):

| injected | what fired |
|---|---|
| `lib.OrderAfter("b")` deleted | `better-belt-balancer-tech-packs's order is "aad" and shipped as "bad"` (and `aae`, `aaf`), the sort test with the dropdown below its three fields, and `... carries the order "aad" and the dropdown it belongs to carries "b"`; at the gate, on a real dump, both `mod_settings_sha256` arms FAIL (`got 9cda532b13b672e6`), both `data_raw_sha256` still ok, and `FAIL base: the setting \`better-belt-balancer-tech-packs\` carries the order 'aad' and has to carry 'bad'` with the sort line, which is what says the probe reads the engine and not the plan |
| `lib.OrderAfter("a")` deleted | `better-belt-balancer-recipe-ingredients's order is "ab" and shipped as "aab"`; the sort test fires only through its literal table, since `a, ab, b, ...` is still the declaration order and `ab` still extends `a`, which is recorded rather than hidden |
| `techPacks` declared above `lib.OrderAfter("b")` | `... order is "aad" and shipped as "bad"`, the sort test, and `better-belt-balancer-tech-packs carries the order "aad" and the dropdown it belongs to carries "b": a generated order that does not extend its dropdown's own is somewhere after it rather than under it`; the other two fields stay at `bae` and `baf` |
| `techPacks` put back on `LegacyPacksSetting("bbb-tech-packs", ..., "bb")` | the prefix test (`0 setting(s) are emitted as "better-belt-balancer-tech-packs" and exactly one has to be`), the transcription on the name and the order, and the locale checker's four sentences: `the setting bbb-tech-packs has no [mod-setting-name] entry`, `... has no [mod-setting-description] entry, and a text setting needs one to tell the player the format`, `the [mod-setting-name] entry better-belt-balancer-tech-packs matches no setting this plan declares`, and the description twin; eleven tests in all |
| the old `bbb-tech-count=` left in the locale file beside the new key | `the [mod-setting-name] entry bbb-tech-count matches no setting this plan declares`, the rename leftover the complete-list reading exists to catch; then the new `[mod-setting-name]` entry deleted: `the setting better-belt-balancer-tech-count has no [mod-setting-name] entry, though one sits under [mod-setting-description]` |
| the gate table's `bad` changed to `bae` for the packs setting | `--golden-only`: both hashes ok, `FAIL base: the setting \`better-belt-balancer-tech-packs\` carries the order 'bad' and has to carry 'bae'` and the same on incumbent; the sort line silent, which says the two assertions are independent |
| the two `OrderAfter` arguments swapped (the review's) | the library ACCEPTS the plan (no tie, no extension), so the mod's tests are the only guard: `... recipe-ingredients's order is "bab" and shipped as "aab"`, the three research fields at `aad`, `aae`, `aaf`, and the under-its-dropdown check firing on all four rows |
| `techSeconds` declared before `techCount` (the review's) | the orders stay well-formed (`bae`, `baf`) and only the pairing moves; the transcription reports it as field mismatches on both rows and the sort test names it directly |

With `ModName` set to `better-belt-balancer-x`, the locale check reports fourteen findings where round three's Legacy plan reported zero, which is what says the prefix now reaches this plan; `locale_test.go`'s "the prefix reaches NOTHING here" paragraph was rewritten around that measurement.

**The adversarial review found no MUST-FIX.** Its five SHOULD-FIX items are taken: three comment lines the rename had pushed past ninety columns and three ragged wraps, an overclaim in the sort test's comment (the six pinned strings determine the sort, so the test adds a second literal rather than a claim the transcription cannot make), and a clause crediting the test with what the library's placement refusal guarantees. Its notes are recorded here rather than acted on: the order probe is blind to a seventh startup setting (the hash and the host's six-op count cover that), and the probe's sort half cannot fail while its per-name half passes (its teeth are for a re-declaration that moves the table).

**Gates at this commit, exit codes read directly**: `gofmt -l .` prints nothing (0), `go vet ./tune/` 0, `go test -count=1 ./tune/` 0, `make check` 0, `make datastage-check` 0 on Factorio 2.0.77 (both settings arms at `df4d6c3fc7bf854d`, both order lines, `recipe-custom`, `recipe-ignored`, `tech-custom-default`, `tech-custom` and `tech-ignored` saying their lines under the new names), `python3 -m py_compile test/check-datastage.py` 0. The package at this commit: `dist/bbbdata.wasm` 1,097,030 B, `fk_data_module.lua` 4,469,509 B and 112,258 lines, 12 stations, the data module's jump line unchanged (the span is the library's `PlanData`, and this mod's plan is not in it).

**`OrderAfter`: CLEAN.** It worked as docs/usage.md and docs/migration.md describe, the worked example being this plan; the emitted orders are what the docs' comments say they are; the refusals are unreachable from one-letter legacy orders and the review confirmed the conditions against go/settings.go; and the one thing a consumer has to know, that the prefix is captured at declaration and a second call moves only what follows it, is in the method's doc comment. Nothing to ask.

### Commit 3: the settings file is the toolchain's to write, and the packager's report is kept beside its log

**The decision: retire `tools/mod-settings.py` for `fklua modsettings write`.** FkLua c21ff07 and f80f61b give the toolchain a writer and a reader for Factorio's property tree, FkRecipes retired its own codec for them (bf95d92), and this repository's pattern for a workaround upstream has answered is to delete it (host.go, the layout check, the `tinygo.wasm` vet tag). Both callers moved: `test/check-datastage.py`'s `write_mod_settings` keeps its name and four-argument signature (the release-to-head drivers replace it by name to install a file) and builds the JSON document FkLua documents, `{"version": [2, 0, 0, 0], "startup": {...}, "runtime-global": {...}, "runtime-per-user": {}}`, handing it to `fklua modsettings write --from FILE.json --out mod-settings.dat` with the exit code and stderr read directly; `bench/run.sh` writes its setup mod's eight typed knobs the same way. `FKLUA` reaches the gate from the Makefile beside `FACTORIO_BIN` (absolute, because the speed arm's fixture is packaged in a temporary cwd, where the relative default broke it: `FileNotFoundError: [Errno 2] No such file or directory: '../FkLua/bin/fklua'`, measured the first time it was passed through). A skipped gate reads like a pass, so each caller probes the binary before the engine is asked anything: a missing file, a directory, an empty `FKLUA=`, or a binary that answers `unknown command "modsettings"` (the a1fcd04 one does, exit 2) refuses with the remedy (`an FkLua checkout at c21ff07 or later, built with cd ../FkLua && go build -o bin/fklua ./cmd/fklua, or FKLUA=<path>`) and exit 1 with no `==> Factorio` line before it.

**The two writers, compared byte for byte** (the deleted tool recovered with `git show HEAD:tools/mod-settings.py`, both driven on the gate's own documents): the `tech-custom` document is 328 bytes from each and differs at four offsets, `cmp -l` giving `215 6 2`, `217 62 0`, `223 0 111`, `224 0 100`, the count's type byte (6 against 2) and the three payload bytes where an int64 50 and a double 50.0 differ; the `recipe-custom` document, strings alone, is 198 bytes and identical (`cmp` exit 0). `fklua modsettings read` renders the new file's count as `50` and the old file's as `50.0`. The toolchain's typing is the engine's: the review staged the shipped package with the deleted writer (every number a double) and read the engine's own rewrite back on 2.0.77, `"better-belt-balancer-tech-count": 50` beside `"better-belt-balancer-tech-seconds": 20.0`, which is the correction recorded above from the other direction. So the arm tables spell each number as its setting's type, `50` for the count and `20.0` for the seconds.

**Witnesses.** `make check` 0; `make datastage-check` 0, every one of the fourteen variant arms driving its setting through the new writer and printing what it printed before, the two settings-order lines included; `make zip` 0 with `--report` accepted beside `--zip` (the echoed line carries both) and `jq -c .jumps.data dist/.fklua-mod.report.json` reading span 991,183 in `PlanData`, 12 stations, block 8,858 in `probeIn`, room 318,819 and `relayed_functions[0].widest_block_bytes` 802, every figure the scratch repackaging above gave; `make bench-setup` 0 and one cell, `bench/run.sh --mod none --scenario control-idle -n 1 -k 2 --ticks 60 --runs 1 --meter 0`, exit 0, its create log carrying the line the harness asserts against its own `WANT=`: `BENCH-SETUP scenario=control-idle n=1 k=2 tier=express item=iron-ore part=balancer-part meter=0 hitch=false rigs_built=1 surface=17x5` (the row it appended to `bench/baselines/results.tsv` was reverted, since a witness is not a baseline); `bash -n bench/run.sh` 0; `shellcheck bench/run.sh` 1 with three findings (SC2012 at two `ls` pipelines, SC2001) that HEAD's file carries at the same places, none in the new block; agents/docs-style.md's five greps over bench/README.md empty. The `idle` scenario the brief suggested is refused by the harness (`scenario 'idle' places balancer parts and needs a balancer mod`), which is why the cell is `control-idle`.

**Red-proven seven times, each observed and the restore cmp-verified** (the implementer's five, the review's two):

| injected | what fired |
|---|---|
| `FKLUA` at the a1fcd04 binary | `NOT RUN: the fklua at .../fklua-a1fcd04 has no `modsettings` command (it exited 2 saying `unknown command "modsettings"`), and every variant arm of this gate drives its setting through `fklua modsettings write`.` with the remedy, exit 1, no `==> Factorio` line |
| `FKLUA=/nonexistent` | `NOT RUN: no executable fklua at /nonexistent, ...`, the same shape |
| the document's `"startup"` key misspelled | the toolchain refuses by name before any engine run: `modsettings: the JSON names "startups", which is neither comment, nor version, nor one of startup, runtime-global, runtime-per-user`, exit 1 |
| the values moved into `"runtime-global"` (a document the writer accepts and the engine reads) | twelve of the fourteen variant arms FAIL on the defaults (`FAIL recipe-cheap: the recipe is [('iron-plate', 4), ...] and `cheap` should be [('iron-plate', 2), ('transport-belt', 1)]`, `FAIL tech-custom-default: the prerequisite is ['logistics'], not ['logistics-3']`, ...); `recipe-vanilla` and `recipe-custom-default` stay green, as the header says they carry no anti-vacuity weight |
| `FKLUA` at the a1fcd04 binary for `bench/run.sh` | `the fklua at .../fklua-a1fcd04 has no `modsettings` command` with three remedy lines, exit 1, no `creating save` |
| `make datastage-check FKLUA=/nonexistent` | refuses at the packager, since the gate depends on `mod` (`/bin/bash: /nonexistent: No such file or directory`, `make: *** [mod] Error 127`); the pass-through is shown by `make -n` and the NOT RUN line by the script's own proof above |
| every value stringified in the document (the review's) | exactly `tech-custom` (the unit reads `count 20, time 15`, the two numbers reset to their declared defaults) and `tech-ignored` (the two `number is ignored` lines gone) FAIL, every string-valued arm green: the two numbers travel through the file and their spelling is load-bearing |

**The adversarial review's three MUST-FIX and four SHOULD-FIX are taken**: two Go comments in `guest/go/obs` still named the deleted file; `bench/run.sh`'s `Env:` line and bench/README.md's requirements and environment lists did not name `FKLUA`; the working notes' present-tense claims about the old writer; a directory or an empty `FKLUA=` reached the gate's probe as a traceback rather than the NOT RUN line (`os.environ.get("FKLUA") or ...` and `os.path.isfile`); the bench comment claimed the exit code was read where only the stderr text decides (an fklua with the subcommand exits 1 on a bare `fklua modsettings` with its usage, one without exits 2 with the refusal, so the text is the discriminator, measured on both binaries); the bench probe sat after `make bench-setup`, so a cold tree refused in the packager's words; and the bench write's refusal message promised a line the directory case did not print (the guard asks for a file now). Its note stands: an unrelated executable that exits 0 passes both probes, and the arms catch it by equality on a full run, which is the structural anti-vacuity the header claims.

**`fklua modsettings write` and `read`: CLEAN.** The JSON shape and the typing rule are documented in FkLua's docs/data-stage.md and matched the engine on the first try; the writer reads its own bytes back before writing them; a refusal names the key. **Probing for the subcommand: AWKWARD, mild, FkLua's.** A script that must refuse NOT RUN on an fklua from before c21ff07 has nothing to ask but stderr: a bare `fklua modsettings` exits 1 with `fklua modsettings: usage: fklua modsettings write --from FILE.json --out mod-settings.dat` on a binary that has it and 2 with `unknown command "modsettings"` on one that does not, so both callers grep the second text. The ask: an exit code or a `fklua meta --json` field that says which subcommands the binary carries, so a harness need not parse a sentence. FkRecipes' gate parses the same sentence (its `run-ingame.sh`), so the shape is at least shared.

### The toolchain's share and the library's share of the size, separated

Round three's exact `dist/bbb.wasm` and `dist/bbbdata.wasm` (built 2026-09-08 08:56 and 08:30, before either sibling moved) were repackaged with the b88965d fklua before anything was relinked (`../FkLua/bin/fklua mod dist/bbb.wasm --persist=packed -o "$S/sync2/pack-r3guests" --report "$S/sync2/report-r3guests.json"`, from the repository root so `fklua.toml` supplies the identity, the data module and `gc = "collected"`), then the tree was relinked against both heads (`make mod`, `make zip`). Sizes by `wc -c` and `wc -l`, stations by `grep -c '::LT'` (two labels per station) and by the report's `stations`:

| | round three (a1fcd04, c7a806e) | the same two wasm files, fklua b88965d | commit 1 (33657f5, both heads) |
|---|--:|--:|--:|
| `dist/better-belt-balancer_0.3.3.zip` | 822,248 B | not built | **817,198 B** (-0.6%) |
| `fk_data_module.lua` | 4,904,124 B, 122,031 lines | **4,318,883 B, 108,919 lines** (-11.9% bytes, -10.7% lines) | 4,470,110 B, 112,279 lines (-8.8%, -8.0%) |
| `fk_module.lua` | 3,148,568 B | 3,132,893 B (-0.5%) | 3,132,893 B |
| `dist/bbbdata.wasm` | 1,085,173 B | 1,085,173 B, by construction | 1,096,842 B |
| relay stations in the data module | 13 (26 labels by grep; the report had no field) | 11 (22 labels; the report says 11) | 12 (24 labels; the report says 12) |
| `jumps.data.widest_span_bytes`, before the relay | not reported at a1fcd04 | 957,828 (146% of the 655,355-byte limit) in `(*fkrecipes.Lib).PlanData` | 991,183 (151%) |
| `jumps.data.widest_span_after_relay_bytes` | | 328,105 | 328,136 |
| `jumps.data.widest_block_function`, `widest_block_bytes`, `block_room_bytes` | | `fkrecipes.probeIn`, 8,858, **318,819** (97% of the 327,677-byte hop) | the same three |
| `jumps.control` | | widest span 303,874 (46%) in `main.flushLive`, no relay; widest block 11,033 in `main.onEvent#wasmexport`, 316,644 of room | the same |

So the toolchain's three size commits take a tenth off the data module on the same guest, the library's eight commits put a quarter of that back (151,227 B of the 585,241 taken off, 3,360 lines of the 13,112, one station), and the zip, which is compressed, moves by half a percent. The module's widest BLOCK sits in `probeIn`, a library helper, and not in `PlanData` (whose own block is 802 B, `relayed_functions[0].widest_block_bytes` in the scratchpad's `sync2/report-r3guests.json`), which is exactly the shape FkLua's docs/lua-limits.md describes and the reason b88965d reports the module's worst block rather than the widest span's. Block room is the figure to watch and it is 318,819 B; the span in `PlanData` is half again over the limit and loads because the relay carries it, as it did in round three. **The packager says so now**, under the data module's line of `make mod`'s output (kept in `dist/.fklua-mod.log`):

```
  data module dist/bbbdata.wasm (4469851 bytes of Lua)
  jump span: widest 991183 bytes in (*github.com/Techrocket9/fkrecipes/go.Lib).PlanData (151% of the 655355-byte limit), relayed through 12 station(s) in 1 function(s); widest span after the relay 328136 bytes (50% of the limit); widest block 8858 bytes in github.com/Techrocket9/fkrecipes/go.probeIn against the 327677 one hop covers, so 318819 bytes of block room (97%)
```

The 4,469,851 in that line is the unwrapped chunk; the file on disk is 259 bytes longer (the factory wrapper), as FkRecipes' sync pass measured.

### The round's close: sizes and the relay, before and after

At commit e3c0ed9, `make zip`; sizes by `ls -l` and `wc -l`, stations by `grep -c '::LT'` (two labels per station) and by the report; the `jumps` objects by `jq -c '.jumps.data, .jumps.control' dist/.fklua-mod.report.json`, which `make mod` and `make zip` write since commit 3:

| | round three (FkLua a1fcd04, FkRecipes c7a806e) | the round's close (b88965d, 9754f49) | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.3.zip` | 822,248 B | **817,626 B** | -0.6% |
| `fk_data_module.lua` | 4,904,124 B, 122,031 lines | **4,469,509 B, 112,258 lines** | -8.9% bytes, -8.0% lines |
| `fk_module.lua` | 3,148,568 B | 3,132,893 B, 88,844 lines | -0.5% |
| `dist/bbbdata.wasm` | 1,085,173 B | 1,097,030 B | +1.1% |
| `dist/bbb.wasm` | 1,302,884 B | 1,302,884 B | unmoved |
| `fk_api_gen.lua`, members pruned | 24,077 B, 56 of 4,870 | the same | unmoved (the b88965d repackaging of round three's wasm emits the same 24,077 B, and no api-generation source moved between the two FkLua heads) |
| relay stations in the data module | 13 (26 labels by grep; no report field) | 12 (24 labels; `stations` 12) | |
| `widest_span_bytes` before the relay | not reported at a1fcd04 | 991,183 B, 151% of the 655,355-byte limit, in `(*fkrecipes.Lib).PlanData` | |
| `widest_span_after_relay_bytes` | | 328,136 B (50%) | |
| `widest_block_function`, `widest_block_bytes`, `block_room_bytes` | | `fkrecipes.probeIn`, 8,858 B, **318,819 B** (97% of the 327,677-byte hop); `PlanData`'s own block 802 B with 326,875 B of room (`relayed_functions[0]`) | |
| the control module (`jumps.control`) | | widest span 303,874 B (46%) in `main.flushLive`, no relay; widest block 11,033 B in `main.onEvent#wasmexport`, 316,644 B of room | |
| data-module functions in the debug map | 170 | 172 | |

Of the data module's 434,615 B drop, the toolchain's twelve commits, three of them the size commits, account for 585,241 B (round three's exact wasm files repackaged with the new fklua before anything was relinked: 4,318,883 B, 108,919 lines, 11 stations, the table above), and the library's eight commits with this plan's two changes put 150,626 B back. The report's `outputs.data_lua_bytes` reads 4,469,250, 259 bytes under the file, which is the factory wrapper FkRecipes' sync pass measured. Block room is the figure to watch and it is 318,819 B; the span in `PlanData` is half again over the limit and loads because the relay carries it, exactly as in round three, and the packager says so on every build now.

### Every gate at its exit code, the round's close

Commit e3c0ed9, Factorio 2.0.77 (build 84539, mac-arm64, steam; `"$FACTORIO_BIN" --version` re-asked before every engine run), both sibling checkouts clean at the heads named at the top, exit codes read directly:

| gate | exit | |
|---|---|---|
| `../FkLua/bin/fklua gen-bindings --check` | 0 | the committed bindings unmoved, `go 4865 + 5, rust 4865 + 5` |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)` |
| `cd guest/go && gofmt -l .` | prints nothing | |
| `make check` | 0 | at every one of the round's commits |
| `make mod`, `make zip` | 0 | the prune tripwire green, 56 members of 4,870; `--report` written beside the log |
| `make datastage-check` | 0 | seventeen arms: both hashed mod sets at `df4d6c3fc7bf854d`, data-raw `1e1fcf4f56f5ef22` and `e7001bf98d6c6771`, the stub graph and the six orders on both, fourteen variant arms through `fklua modsettings write`, the speed arm |
| the release-to-head proof (`sync2/prefs-head2.py`, `compare-prefs.py`) | 0 | 28 pairs at commit 1 and at commit 2, recorded above |
| `python3 -m py_compile test/check-datastage.py`, `bash -n bench/run.sh` | 0 | |
| one bench cell through the new writer | 0 | `BENCH-SETUP scenario=control-idle n=1 k=2 ...` asserted by the harness |
| agents/docs-style.md's greps over bench/README.md | empty | the one human-facing file this round touched; README.md and the changelog are untouched, because nothing a player sees moved |
| `make test`, fourteen suites | **NOT RUN** | the packaged mod is pinned 2.1 and the binary is 2.0.77; `test/run.sh` refuses at its engine gate, as in every round before |
| the 2.1.16 and 2.1.17 golden rows | **NOT RUN** | each carries a `_stale` note naming every move since its capture (five on the base arm: the stub's `place_result`, the recipe customizer, the research customizer, the localised cost presets, the four generated names; four on the incumbent arm, which defines no stub) and the re-capture command for the next 2.1 session |
| the client run | **NOT REACHABLE**, owed since round three | round three's checklist stands, with the six rows reading `a`, `aab`, `b`, `bad`, `bae`, `baf` |
| the `bench/` matrix | out of scope | one cell ran as the writer's witness and its row was reverted |

### Friction, graded, the sync pass

Same grades: CLEAN worked as documented, AWKWARD worked but cost something, MISLED pointed the wrong way, BLOCKED needed a workaround or could not be done. **Nothing is BLOCKED or MISLED.**

#### `OrderAfter`: CLEAN

Recorded under commit 2. The docs' worked example is this plan, the emitted orders are what its comments say, the refusals are unreachable from one-letter legacy orders, and the one thing a consumer has to know (the prefix is captured at declaration) is in the doc comment. Round three's AWKWARD is closed.

#### An edited number under a tier says so, and a cost preset names its technology by its localised name: CLEAN

Recorded under commit 1. Both of round three's mild AWKWARDs are closed by da11cf5; the cost to this consumer was the transcription, the three-line pin and the golden, and the `Unknown key:` trade for a missing technology is documented and is this mod's ladder to order.

#### The relay is reported: CLEAN

Recorded under commit 1 and closed by FkLua 2a541a7 and b88965d; `make mod` prints it, the report carries it, and commit 3 keeps the report beside the log.

#### `fklua modsettings write` and `read`: CLEAN; probing for the subcommand: AWKWARD, mild, FkLua's

Recorded under commit 3, with the ask: an exit code or a `fklua meta --json` field that says which subcommands the binary carries, so a harness need not grep `unknown command "modsettings"` out of stderr.

#### The library's line-number citations drift with every library commit: noted, this repository's own habit

Commit 1 re-pinned the citations in `guest/go/tune` (twenty-nine added lines carrying one, 31 occurrences, 24 distinct strings); commit 1's review spent two findings on them. A function name beside a line number is what makes a stale one findable, and every citation here carries one.

#### The `jumps` object's widest block is a library helper's: noted, matches FkLua's docs

`probeIn` at 8,858 B rather than `PlanData` at 802 B, which is the shape FkLua's docs/lua-limits.md predicts (the block sits in a helper the optimizer did not inline into the section function) and the reason b88965d reports the module's worst block. Nothing to ask.

### What this pass leaves open, and what it closes

Closed: every one of round three's four AWKWARDs (the placement of a generated setting, the ignored number, the localised cost preset, the silent relay), so round three's ledger carries nothing open. Of round two's ledger, one entry is answered upstream and unrecorded until now (`PlaceResult`'s probe and the docs' "beside": FkRecipes docs/migration.md now says the entity is "extended before the `Emit` call runs"), and two stand as they were graded: `max_level` travelling with the copied unit, which docs/migration.md now documents with a workaround (a single-level source, or a written `Unit`) rather than the opt-out asked for, a documented decline; and the one-sentence ask for step 2 of the incremental path (that a recipe and the technology that unlocks it move together), still absent from docs/migration.md. Neither blocks this consumer. Corrected: round three's heading on the engine's int typing, and the count of customizer settings in the instruction that launched this pass (and in the FkRecipes follow-up brief it drew on), which said five where the plan declares four (the recipe text, the pack text, the count, the seconds). Open, unchanged from round three: the client run (the composed descriptions' line breaks, the text field's rendering, the labels as rendered, a custom recipe crafted), and the 2.1 golden rows, which only a 2.1 binary can re-capture. New and not this mod's to close: the subcommand probe above. The changelog entry for 0.3.3 is what round three wrote, because nothing a player sees moved this round: the four settings' labels and descriptions are byte for byte what they were, and their internal names have never shipped. (**DATED: that sentence is true of THIS pass and of no later one.** Fix round 2's third commit, 2026-09-14, rewrote the 0.3.3 section outright, because by then the value both feature rows announced had been withdrawn.)

## Fix round 1, 2026-09-09: the library answers the migration assessment, and this mod adopts

[`agents/migration-assessment.md`](agents/migration-assessment.md) is this mod's own adversarial assessment of the shipped shape at `d8dc79e`, twelve findings, every claim carrying its command. FkRecipes answered the library's half of it in four commits, `9754f49` to `0077f3c`, and this round is what adopting that costs the consumer. Every claim below is on Factorio 2.0.77 (build 84539, mac-arm64, steam; `"$FACTORIO_BIN" --version` re-asked before every engine run), the two sibling checkouts clean and read-only, FkLua at `b88965d` and FkRecipes at `0077f3c`.

### What moved in the library, in this mod's terms

**FkRecipes, 9754f49 to 0077f3c.** `3a9de88` is the invisible-character policy and the two reserved words folded together; this plan reaches neither. `696387f` is the one this mod is about: a LADDER THAT LANDS TWICE ON ONE NAME MERGES into the first occurrence and logs the arithmetic, where an AUTHOR's literal duplicate is refused at plan time by name. `2e5f779` stops a PLAYER'S unusable text refusing the load: the whole typed list is set aside, the author's declaration applies, and one `fkrecipes: ERROR:` line names the setting and the reason; the fallback is not total, and a declaration that cannot produce a legal result in that mod set still stops the load. `0077f3c` reshapes the composed description: an ingredient preset's `": "` separator becomes `"\n  type: "`, and a text setting's description gains two sentences.

### The baseline, the untouched tree at ca9aa36 against FkRecipes 0077f3c

| gate | |
|---|---|
| `../FkLua/bin/fklua gen-bindings --check` | **exit 0** |
| `../FkLua/bin/fklua lock --check` | **exit 0**, `fklua.lock is up to date (api 2.1.17)` |
| `make mod` | **exit 0**. The library's report claim "BBB's declarations compile unchanged" is true |
| `make check` | **exit 2**, FIVE failing tests, all in `guest/go/tune`, all encoding the OLD behaviour. The library's "nothing in BBB is required" is true of COMPILING and false of the SUITE |
| `make datastage-check` | **exit 2**, and exactly as the library predicted: both 2.0.77 arms' `mod_settings_sha256` moved `df4d6c3fc7bf854d` to `aa75497e55ce4333`, identically, and both `data_raw_sha256` are unmoved |

The five: `TestEverySettingPrototypeIsTheOneThatShipped` and `TestTheCustomResearchDefaultsAreTheFallbackUnit` are `0077f3c`'s composed-description move; `TestEveryOptionFallsAllTheWayToIronPlate` is `696387f`'s merge; `TestACustomTextTheGameCannotAnswerIsRefused` and `TestAPackTextTheGameCannotAnswerIsRefused` are `2e5f779`'s fallback, and their NAMES assert the opposite of what the library now does.

### Commit 1: the five moved contracts, and the ladder keeps every rung

**THE LADDER DECISION IS KEEP EVERY RUNG, AND IT IS MEASURED RATHER THAN ARGUED.** Assessment finding 1 is that a pack removing the item `transport-belt` breaks the DEFAULT configuration: the vanilla ladder's third rung lands on `iron-plate`, which the list already names, and the engine refuses `bbb-balancer-part` with `Duplicate item ingredients are not allowed (iron-plate exists 2 or more times).`, exit 1, no dump, no `fkrecipes:` line and no setting named, for a player who never opened the Startup tab. `696387f` merges the second landing into the first. **The engine ACCEPTS the merged recipe**: one real `--dump-data` on 2.0.77 behind a synthetic `bbbt-remover` that deletes that item and keeps the entity, exit 0, `6 iron-plate, 2 iron-gear-wheel`, and one line, `fkrecipes: bbb-balancer-part: iron-plate is in the list twice after the fallbacks, so the amounts are added: 4 plus 2 is 6`. The control run, the same fixture removing a name nothing has, is exit 0 with the vanilla list and zero `fkrecipes:` lines, which is what says the removal did the work. So the assessment's crash is closed on the ENGINE and not only in the plan.

**The two alternatives were built and costed, not reasoned about.** Dropping the terminal `iron-plate` rung wherever the list already names it at top level was applied to `recipe.go` verbatim and probed: it does NOT remove the merge (belt-express and splitter-express have no top-level `iron-plate` and still merge between two fallback ladders), it fails `TestEveryLadderTerminates` six times (`cd guest/go && go test ./tune/`, each error `vanilla item 1 ends at "iron-gear-wheel"; every ladder must end at "iron-plate"` in its own preset's wording), and it makes the recipe CHEAPER, because the ingredient is deleted rather than substituted for. Base's own recipes out of a `--dump-data` price a gear at 2 plates and a belt at 1.5 (`jq '{gear: .recipe["iron-gear-wheel"].ingredients, belt: {ings: .recipe["transport-belt"].ingredients, results: .recipe["transport-belt"].results}}'`: the gear takes 2 `iron-plate`, and the belt takes 1 `iron-plate` and 1 gear for TWO belts), so vanilla costs 11 plate-equivalent in a stock game, 10 under the merge (91%) and 8 with the rung dropped (73%); cheap is 3.5, 3 and 2. Giving the colliding rungs a terminal name that is not already in the list needs THREE distinct names as universal as `iron-plate` for belt-express alone, and there is no second such name: base 2.0.77 has 174 item names (`jq '.item | length'` over the base arm's dump reads 175, of which exactly one, `bbb-balancer-part`, is this mod's) and every candidate (`copper-plate`, `stone`, `coal`, `wood`, `electronic-circuit`) is strictly less likely to survive an overhaul than `iron-plate`. **The merge is [Item]'s substitution semantics arriving rather than an accident that happens to load**: the amount does not move down the ladder, two belts substituted for at the same amount ARE two more plates, and the addition is the only arithmetic that keeps that representable in a list the engine will accept. The highest merged amount ANY preset can reach is the sum of its own amounts, because a list's ladders can land on nothing but names already in that list: 8 for the three presets naming three ingredients and 3 for the other three. `TestEveryOptionCollapsesOntoIronPlateAndMergesTheAmounts` pins exactly those six numbers in the world that realises the bound, whose only item is `iron-plate` (`cd guest/go && go test -run TestEveryOptionCollapsesOntoIronPlateAndMergesTheAmounts ./tune/`), so the ceiling is one this suite holds rather than a figure off a throwaway probe. 8 is four orders of magnitude under the engine's 65535 item-ingredient ceiling (FkRecipes' measurement on this same build, not re-taken here).

**So `recipe.go`'s vanilla comment is rewritten and the plans are untouched.** The sentence that went said the rungs buy "a pack that removed one of them, where today's mod fails the load outright", which was measurably false in both halves: the rungs bought that pack a DIFFERENT load failure, and what they buy now is a substitution the library merges. `FallbackName`'s own doc comment in `tune.go` says only that it is the last rung of every ladder and that `TestEveryLadderTerminates` keeps it so, which is still true to the word, and it is left alone.

**The five tests are rewritten to the NEW contract rather than made green, and three are renamed.**

| test | what it asserts now |
|---|---|
| `TestEverySettingPrototypeIsTheOneThatShipped` | the three moved descriptions, TRANSCRIBED FROM THE ENGINE'S OWN DUMP rather than from the failure text (`run_arm` over the built mod at `dist/better-belt-balancer_0.3.3`, `mod-settings-dump.json`). `presetLine` ends `"\n  type: " + text`; the two text settings gain the library's two sentences through one `textFieldLines` helper, which is the one place this file derives, because those two sentences are the LIBRARY'S and identical on both fields by construction |
| `TestTheCustomResearchDefaultsAreTheFallbackUnit` | the same property through a different predicate. "Ends with the rendered fallback" was right while that was the last parameter and is not, because two library sentences follow it; `Arr[len-3]` would keep the shape and lose the meaning. It now finds every parameter beginning `"\ndefault: "`, demands EXACTLY ONE, and compares it against `FallbackUnit().Packs` rendered, which is positional in nothing and is stricter in the case that matters: a second transcription of the declared default fails always, where "ends with" caught it only if it came last |
| `TestEveryOptionCollapsesOntoIronPlateAndMergesTheAmounts` (was `TestEveryOptionFallsAllTheWayToIronPlate`) | the merge lines ASSERTED, not tolerated. The loop stays and becomes a whole-stream comparison per preset, two lines for the three-ladder presets (`4 plus 2 is 6`, `6 plus 2 is 8`) and one for the other three; the header's masking warning stays true, because a preset fallen through to the library's default plan carries a line that is not in the table. The FINAL AMOUNT is asserted too (8, 3, 8, 8, 3, 3), which the old form never checked at all |
| `TestACustomTextTheGameCannotAnswerFallsBackAndSaysSo` (was `...IsRefused`) | for each of the three texts, the emitted recipe IS this mod's declared vanilla list AND the log is exactly one `fkrecipes: ERROR:` line naming the setting, the entry and the reason. A FOURTH ROW drives the first text again in a game with no `transport-belt`, and the review is why it is there: the other three cannot tell the DECLARATION from the `vanilla` PRESET, because [Plan] declares this field's default AS `RecipePlan(RecipeVanilla)` and in a game that has everything the two are the same three pairs. Take the belt away and they part, and the row asserts the pair only a laddered declaration can give, `6 iron-plate, 2 iron-gear-wheel` and the merge line after the ERROR line |
| `TestAPackTextTheGameCannotAnswerFallsBackAndSaysSo` (was `...IsRefused`) | for each of the four texts, the unit IS `FallbackUnit` (20 x 15 s, one `automation-science-pack`) AND the stream is exactly two lines in order, the ERROR line then the cost line |

**AND THE FALLBACK'S BOUNDARY IS PINNED RATHER THAN LEFT AS A SENTENCE.** None of the seven texts above is a case the fallback cannot carry, because that fixture has every rung of every ladder. `TestAFallbackThisModsOwnPackListCannotPayForStillRefuses` is the case it cannot: the same `1 water` in a game whose only science pack is `logistic-science-pack`, where this mod's declared list has nothing to land on either and the load stops with `fkrecipes: the technology bbb-balancer has no science pack the game has; research takes at least one. The stored value of better-belt-balancer-tech-packs could not be used, so the mod's own declaration applied; correcting it under Settings > Mod settings > Startup is what a player can change here.` That second sentence is the library's own, and it is the whole difference between this refusal and the one a player who typed nothing gets in the same game.

### Two new gate arms, one per layer, and both red-proven

**The host arm, `TestAPackWithoutTransportBeltMergesRatherThanDuplicating`** (`guest/go/tune/plandata_test.go`): the everything world with `transport-belt` filtered out of `ladderVocabulary()`, six rows. vanilla is `6 iron-plate, 2 iron-gear-wheel` with one merge line, cheap is `3 iron-plate` with its own, and THE OTHER FOUR ARE THE ANTI-VACUITY HALF and are not optional: belt-fast, belt-express, splitter and splitter-express must come out at their stock-game lists and say NOTHING, which is what proves the fixture removed one name rather than emptying the vocabulary. Milliseconds, no toolchain, no engine.

**The engine arm, `check_remover`** (`test/check-datastage.py`): 1.85 s wall, measured, three consecutive runs at the same figure. It writes a `bbbt-remover` Lua fixture out at run time (no tinygo, no `fklua`, because it has to run at the DATA stage before this mod and a plain `data.lua` is the only thing that can be that cheap) and stages it through `run_arm`'s existing `extras=`. It writes NO `mod-settings.dat`, so it runs on the prototype's own `default_value`, which is the configuration of the player finding 1 is about. Four assertions, in this order: the load order, out of a new `load_order` key beside `fkrecipes_lines` (excluded from `--capture` with the probe and the log, for the same reason); `.item["iron-plate"]`, which the fixture never touches, really being PRESENT; `.item["transport-belt"]` really being `null`; and then the list AND the line, both, for the reason `RECIPE_CUSTOM_ARMS` asserts both. **The item question is asked in both directions and the positive one comes first**, because `is not None` alone is satisfied by a jq path that stopped matching anything: a stale projection would read as a removal that never happened, which is the opposite of what `check_speed`'s anti-vacuity does. **The fixture clears `minable` and `next_upgrade` TOGETHER, which is a measured trap**: clearing the first alone refuses the load with `Error while running setup for entity prototype "transport-belt": Entity must be minable when next_upgrade is set. (was fast-transport-belt)`. It keeps the entity, because deleting it breaks this mod unconditionally (`Error in assignID: entity with name 'transport-belt' does not exist`), which is a different defect.

**Red-proven four times, each injected locally in this mod's code or its gate, observed, reverted and the revert cmp-verified.** FkRecipes is read-only for this pass, so the library-side break the deep dive named is not among them. Two MORE proofs, on the two guards the review strengthened, are in the review section below; the review re-took all four of these and all four went red with the text recorded here.

| injected | what fired |
|---|---|
| `recipe.go`: vanilla's third ladder ends at `steel-plate` | the host arm, `vanilla is made of [{iron-plate 4} {iron-gear-wheel 2} {steel-plate 2}] and the gate asserts [{iron-plate 6} {iron-gear-wheel 2}]`, with `got []` where one merge line was demanded; beside the collapse test and `TestEveryLadderTerminates` |
| `recipe.go`: vanilla's third ladder ends at `iron-plate2`, packaged and run on the engine | the engine arm, both halves: `FAIL remover: with no 'transport-belt' in the game the recipe is [('iron-plate', 4), ('iron-gear-wheel', 2)]` and `the library's log lines are ['fkrecipes: bbb-balancer-part: none of transport-belt, iron-plate2 is present, so the ingredient is dropped']` |
| the fixture's `ITEM_CLASSES` with `"item"` taken out, so it removes nothing from the item table | the engine arm's THIRD anti-vacuity assertion, before either claim: `FAIL remover: 'transport-belt' is still in the item table after the fixture ran, so nothing was taken away and this arm proves nothing` |
| the fixture renamed `zzz-remover`, so it sorts after this mod | the engine arm's FIRST assertion: `FAIL remover: the data stages ran ['core', 'base', 'better-belt-balancer', 'zzz-remover'], and this arm needs zzz-remover before better-belt-balancer`. Incidentally this is the measurement that Factorio's data-stage order follows the mod NAME and not the staging order, which is the assumption `bbbt-fastbelt` already rested on and which nothing here found documented |

### The golden, and what moved in it

`test/check-datastage.py --capture` on 2.0.77, and the file moved in SEVEN lines: two hashes, the 2.0.77 `_note` and the four 2.1 `_stale` notes. **Only `mod_settings_sha256` moved, `df4d6c3fc7bf854d` to `aa75497e55ce4333`, the same on both mod sets**; both `data_raw_sha256` are unmoved to the digit (`1e1fcf4f56f5ef22` base, `e7001bf98d6c6771` incumbent) and so are both `prototype_list_checksum` values. Since that hash is taken over the whole normalised data-raw dump, its being unmoved IS the two dumps compared byte for byte. The cause is FkRecipes `0077f3c` and nothing of this plan's: the ingredient preset separator and the two sentences on each text field. The 2.1.16 and 2.1.17 rows are NOT re-captured and their `_stale` keys are kept: their mod-settings dump moves the same way and only a 2.1 binary can record it, so each of the four gains the same paragraph naming the move and the re-capture command.

### The composed tooltips, re-measured on the engine

Read out of `mod-settings-dump.json`. Four of the mod's seven settings carry a `localised_description`, which are the four the library composes.

| setting | parameters | depth | composed lines |
|---|--:|--:|--:|
| `bbb-recipe-cost` | 7 of 20 | 3 of 19 | **13** |
| `bbb-tech-cost` | 4 of 20 | 3 of 19 | 4 |
| `better-belt-balancer-recipe-ingredients` | 4 of 20 | 2 of 19 | 4 |
| `better-belt-balancer-tech-packs` | 4 of 20 | 2 of 19 | 4 |

The ceilings are FkLua's engine probe's (20 parameters, 19 nested tables; the engine's own counter prints one higher than the table count) and `maxLocalisedParams = 20` is FkRecipes' own constant recording that measurement, in both halves. **NOTHING IS NEAR EITHER CEILING**: the worst is `bbb-recipe-cost` at 7 of 20 and depth 3 of 19, and growth costs one top-level parameter per preset and no depth at all, so this mod would need 13 more recipe presets before the top level filled.

**THE LINE COUNT IS WHAT MOVED, AND 13 IS EXACT RATHER THAN APPROXIMATE.** The composition holds 12 `"\n"`, so 13 lines, line 1 being this mod's own `[mod-setting-description.bbb-recipe-cost]` entry. The client run recorded under round three saw SEVEN lines, which was `1 + 6` under the old one-line-per-preset shape; `"\n  type: "` spends two newlines where `": "` spent one, so `1 + 6 x 2 = 13`. With this mod's own locale substituted the whole tooltip is **1,163 characters** over those 13 lines, widest line **557**, which is line 1 and will wrap; the 12 library-composed lines below it are 32 to 66 characters each. (Those two totals read 788 and 182 when this commit measured them, and they are CORRECTED IN PLACE rather than annotated, because commits 2 and 3 of this same round rewrote the two dropdown entries and line 1 of each tooltip IS the consumer's entry. That is commit 3's own precedent for a figure this round made stale inside itself, where a number from an EARLIER round is annotated instead. The re-measure and its command are under the round's close.) `bbb-tech-cost` is unchanged in shape at 4 lines, and the two text fields were 2 lines before this round and are 4.

**WHAT THE ROUND DOES NOT FIX IS THE CLOSED DROPDOWN.** The client measured that widget truncating its label at roughly 37 characters, and TEN OF THIS MOD'S ELEVEN dropdown labels exceed it: `bbb-recipe-cost` has six of seven over (median overflow 15 characters, worst 28, only `Splitter: 1 splitter, 2 iron plates` at 35 fitting) and all four of `bbb-tech-cost`'s are over. That string is the CONSUMER'S label, which the library never composes and never sees. What the new shape buys is the tooltip: the copyable internal list opens its own indented line, so a truncated label no longer hides where it begins. (**CLOSED BY FIX ROUND 2'S THIRD COMMIT, 2026-09-14, AND THIS PARAGRAPH IS KEPT AS THE MEASUREMENT IT WAS.** The labels are names now and all nine fit; the eleven counted here are the eleven of the day, two of which went with the withdrawn value.)

**NOT MEASURED: WHETHER THE 13-LINE TOOLTIP RENDERS WHOLE.** No client was run in this round and none was started. The line count, the character count and the composed table are measured; rendered height, wrapping, clipping, scrolling and legibility are not. For the next session with a client at hand: hover `bbb-recipe-cost`'s info icon and check that all six `type:` lines are present and that the last of them, `  type: 1 express-splitter, 2 steel-plate`, is visible and neither clipped nor scrolled off.

### Commit 1's gates, at their exit codes

Factorio 2.0.77 (build 84539), the binary re-asked its version first, exit codes read directly.

| gate | exit | |
|---|---|---|
| `cd guest/go && go test ./tune/` | 0 | |
| `../FkLua/bin/fklua gen-bindings --check` | 0 | the committed bindings unmoved |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)` |
| `make check` | 0 | gofmt included; `gofmt -l guest/go` prints nothing |
| `make mod` | 0 | |
| `make datastage-check` | 0 | **EIGHTEEN arms**, 35.3 to 37.0 s real over three runs (`/usr/bin/time -p`): the two hashed mod sets at `aa75497e55ce4333`, data-raw `1e1fcf4f56f5ef22` and `e7001bf98d6c6771`, the stub graph and the six orders on both, fourteen variant arms, the speed arm and the new merge arm |
| agents/docs-style.md's greps over README.md | empty | the one human-facing file this commit touches |
| `make test`, fourteen suites | **NOT RUN** | the packaged mod is pinned 2.1 and the binary is 2.0.77; `test/run.sh` refuses at its engine gate, as in every round before |
| the 2.1.16 and 2.1.17 golden rows | **NOT RUN** | each `_stale` note now names this move too, with the re-capture command |
| the client run | **NOT REACHABLE**, owed since round three | the 13-line tooltip is the thing to look at; the checklist is above |

### The adversarial review, and what it changed

Commit 1 was reviewed adversarially before it was committed. **No BLOCKING finding**, all four red proofs above re-taken and red with the text recorded here, four SHOULD FIXes and six NOTEs. Everything the review asked for landed in the same change, and one thing it found is deliberately left open.

**Every FkRecipes citation in `guest/go/tune` is re-pinned against `0077f3c`**, which is the habit round three's own ledger records. Thirty-five lines carry one, 37 occurrences and 29 distinct strings; 33 of the 37 moved. Each was resolved by taking the cited line's TEXT at `9754f49`, finding that text at `0077f3c`, and then READING the new line to confirm it is the thing the sentence claims rather than a brace it happens to land on. All 33 landed on something their sentence does not name, which is what a line-number citation costs when the library moves under it; the review pointed at twelve by name, four of those inside comment blocks this commit was already editing.

| library file | occurrences | how far they moved |
|---|--:|---|
| `go/customize.go` | 15 | `:425` to `:440` at the near end and `:1004` to `:1302` at the far; `noteIgnoredText` `:880` to `:1171`, `customPrereqs` `:990` to `:1282` |
| `go/data.go` | 13 | `:235` to `:247` at the near end and `:1317` to `:1478` at the far; the three ignored-field call sites `:573` to `:575` become `:654` to `:656` |
| `go/ingredientlist.go` | 6 | `defaultWord` `:86` to `:100`, `categoryTakesItemsOnly` `:114-128` to `:145-159`, and `resolveName`'s item-then-fluid pair `:1077`/`:1080` to `:1199`/`:1202`, which is the line the `FluidExists` panic trace prints |
| `go/lib.go`, `go/world.go` | 3 | unmoved: `IngredientsSetting` is still `:378` and `UnimplementedWorld.FluidExists` still `:168` |

**Three sentences in `mod-data/locale/en/better-belt-balancer.cfg` that the same library bump falsified.** Two, one above each text field, said the composed description "ends with a second line saying `default: ...`" where it now ends with THREE lines, the declared list and the library's two sentences; both say the shape and what the entry below therefore does not repeat. The third, in the `[string-mod-setting]` header, said the recipe dropdown's tooltip is "composed from each label followed by the preset's internal names, so the two vocabularies sit side by side", which is exactly the shape `"\n  type: "` replaced. That one is corrected to the shape and to nothing else: each preset's internal list is on its own indented `type:` line under its label, and the labels-keep-their-spelling decision the paragraph argues is a later commit's to record.

**Two numbers that did not check out, in three places.** `REMOVER_LUA` is 67 lines, not thirty (`len((REMOVER_LUA % json.dumps(REMOVER_ITEM)).splitlines())`, and the engine's own log has said `Script @__bbbt-remover__/data.lua:67` on every run of the arm); `CLAUDE.md` and `test/check-datastage.py` both said thirty, and it is the number the write-it-at-run-time decision rests on. And `CLAUDE.md`'s "four orders clear of either ceiling" on the tooltip was the 65535 sentence's phrase pasted onto the parameter counts: 7 of 20 is 13 clear and depth 3 of 19 is 16.

**Two guards that could not fail for the reason they claimed, and both are now red-proven.**

| guard | what it could not catch | what it does now |
|---|---|---|
| `TestACustomTextTheGameCannotAnswerFallsBackAndSaysSo` | the DECLARED list against the `vanilla` PRESET: identical bytes in a game that has everything | a fourth row in a game with no `transport-belt`. RED-PROVEN by declaring the field's default as the flat three names instead of `asIngredients(RecipePlan(RecipeVanilla))`: `the text "3 tungsten-plate" is made of [{iron-plate 4} {iron-gear-wheel 2}] and the gate asserts [{iron-plate 6} {iron-gear-wheel 2}]`, and a DROP line where the merge line was demanded. The three rows above it stayed green, which is the finding |
| `check_remover`'s item anti-vacuity | a jq path that went stale: `.item[...]` answers `null` for everything and `is not None` passes while measuring nothing | `.item["iron-plate"]`, a name the fixture never touches, asserted PRESENT first. RED-PROVEN by misspelling the projection `.itemz[...]`: `FAIL remover: iron-plate is not in the item table either, and the fixture never touches it: the probe reads nothing`. The control is the same break with the new assertion removed, which reports `ok remover [('iron-plate', 6), ('iron-gear-wheel', 2)], and the engine loaded it` and exits 0 |

**`REMOVER_ITEM` is not the knob its comment said it was.** The comment claimed growing the fixture is editing that constant rather than editing Lua. MEASURED with `iron-gear-wheel` in its place: both data stages run and the merge line is written, and then the engine exits 1 on `Error in assignID: recipe with name 'steam-engine' does not exist. It was removed by bbbt-remover. Source: electric-network (tips-and-tricks-item).`, which `run_arm` turns into a `sys.exit` so the whole gate stops rather than one arm printing FAIL. The fixture deletes recipes without pruning the tips-and-tricks entries that name them. The comment now says that, and the fixture is not made general.

**One figure had no command behind it and now has one this repository can reproduce.** "Six presets against fifteen worlds" was a throwaway probe's shape; what replaced it is the bound and the committed test that pins it.

### What commit 1 leaves open

**A player can name the balancer part as its own ingredient, and nothing stops them.** The review constructed it: `bbb-recipe-cost = custom` with `bbb-recipe-ingredients = 2 bbb-balancer-part` is ACCEPTED, emits `[{bbb-balancer-part 2}]`, draws the ordinary "takes its ingredients from" line and no `fkrecipes: ERROR:` at all. The name resolves because the library's `planItemWorld` overlay answers for the items this plan is about to emit, which is what makes a plan able to name its own products, and this text names one of them. What the player gets is a recipe that can never be crafted, with no refusal, no fallback and no line saying so.

It is PRE-EXISTING and out of this round's scope: it arrives with the text field itself and nothing in the four library commits this round adopts touches it. It is also the player's own doing and it is recoverable without a save being lost, because the field can be edited back: the fix is the same three steps every fallback line already names, Settings > Mod settings > Startup. **Nothing gates it today**, in this mod or in the library, and no arm here drives it. Whether a self-referencing ingredient should be refused at the list, warned about, or left alone as a thing a player asked for is a question for the library rather than for this consumer, and it is written down here so a later round asks it rather than rediscovering it.

### Commit 2: the ladder is the tier list itself, and an untouched Custom moves nothing

**THE DECISION IS TO FLIP THE CUSTOM ARM'S `Position` LADDER, AND IT RESTS ON A WHOLE-DUMP MEASUREMENT.** It was `[]string{TechLogistics3, TechLogistics2, TechLogistics}` and it is `TechOptions()`, which returns `logistics, logistics-2, logistics-3` and whose head `TechDefault()` already is. **Expressing it as `TechOptions()` rather than as a transcribed slice is the second half of the decision**: the property that carries the argument is "the ladder's head is this mod's own default tier", and a slice written out beside the list it has to agree with is a copy that can drift the day the default tier moves. Written this way the Custom arm inherits the default tier's placement by construction, and `TechOptions()`' own doc comment already licenses the reading, because the strings ARE the base technology names and there is no mapping table under them.

**Measured on Factorio 2.0.77 (build 84539, mac-arm64, steam; the binary re-asked its version at the start of every run), both arms, four runs.** The instrument is the gate's own: scratchpad `fix1bbb/c2/prereq.py` imports `test/check-datastage.py` and calls its `run_arm` (private mods directory, private `write-data` user directory) and `write_mod_settings` (the stored value goes in through `fklua modsettings write`, exactly as every variant arm's does). The real Factorio user directory was never named on a command line. The tree was built with `make mod` before each pair.

| ladder | stored | `unit` | `prerequisites` | whole-dump `data_raw_sha256` |
|---|---|---|---|---|
| `logistics-3, logistics-2, logistics` (before) | nothing | 20 x 15 s, 1 automation-science-pack | `["logistics"]` | `1e1fcf4f56f5ef22` |
| `logistics-3, logistics-2, logistics` (before) | `bbb-tech-cost = custom`, nothing else | **identical** | **`["logistics-3"]`** | **`afb19d7fa0cfd0a5`** |
| `TechOptions()` (this commit) | nothing | identical | `["logistics"]` | `1e1fcf4f56f5ef22` |
| `TechOptions()` (this commit) | `bbb-tech-cost = custom`, nothing else | identical | **`["logistics"]`** | **`1e1fcf4f56f5ef22`** |

`mod_settings_sha256` is `aa75497e55ce4333` on all four, because a stored value does not move a settings prototype. **The last row is the point**: the whole normalised data-raw dump is byte-identical to the default one, so not one prototype anywhere in the table moves, which is what finally makes 0.3.3's own changelog sentence ("picking Custom changes nothing there until you edit one") true of the tree and not only of the price. On the old ladder the one field that differed was `prerequisites`, and what it cost is in the same dumps: base 2.0.77 gates `logistics-3` behind `["production-science-pack", "lubricant"]`, so an untouched Custom charged **20 red science for a technology past the production-science wall**.

**THAT MEASUREMENT'S SCOPE IS A MOD SET THAT HAS `logistics`, and the player-facing texts carry the condition rather than generalising over it.** Take the head away and the two dropdown readings PART, because a tier's ladder is `TechLadder` and the default tier's is that one name where the Custom arm has two rungs under it. Measured on the host in a game holding `logistics-2` and `logistics-3` and nothing below them (`cd guest/go && go test ./tune/ -run TestTheCustomArmTakesItsPlaceFromTheDefaultTier -v`):

| `bbb-tech-cost` | `unit` | `prerequisites` | the plan's log |
|---|---|---|---|
| `logistics`, the default | 20 x 15 s, 1 automation-science-pack | **no field at all** | `fkrecipes: bbb-balancer: no source for the logistics cost carries a unit, so the fallback cost applies and the technology has no prerequisite` |
| `custom`, nothing else stored | identical | **`["logistics-2"]`** | the `takes its research cost from` line, unchanged |

So picking Custom THERE does move the research, from nowhere to Logistics 2. That is the nearest legal place and it is what the rungs under the head are for, but it is a move, so "picking Custom does not move it" is true of a game that HAS Logistics and of no other. `mod-data/changelog.txt`, the `bbb-tech-cost` locale entry and `README.md` each say it that way and state the without-Logistics case as its own clause.

**WHAT THE ARGUMENT THAT STOOD IN `plan.go` GOT RIGHT, AND WHERE IT WENT WRONG.** Right: the tree-position rule itself, which is the section's whole point and does not move -- a cost and a place in the tree that disagree is the defect, and a blue-science price at a red-science place is the shape of it. Wrong: whose state it reasoned about. It argued from the numbers a player who picks Custom is ABOUT TO write, and a ladder decides the state they are IN; a player who has not typed yet is paying `FallbackUnit`, which is base's own `logistics` unit, and the old ladder put that price past the production-science wall. `CLAUDE.md` recorded the result as deliberate ("the mismatch this section argues against, with the sign flipped, for as long as the player has not typed"), and that paragraph is deleted rather than edited, because there is no mismatch left to record. The expensive case the old argument protected gates itself: a player who writes 300 units of four packs has stated the gate in the cost. **That last step is REASONING and not a probe**, and is marked rather than dropped because the choice between the two errors turns on it: a cost gates softly, since an unaffordable research is still reachable by playing on, where a prerequisite gates hard, since a walled one is not reachable at all until the wall is. NOT MEASURED on the engine.

**THE ONE HAZARD, AND IT IS THE MOST IMPORTANT PART OF THE COMMIT.** `test/check-datastage.py`'s `tech-custom-default` arm took its anti-vacuity FROM the prerequisite: with the ladder starting at `logistics-3`, a `mod-settings.dat` that never reached the engine and one that did gave different prerequisites, so the arm could not pass without the stored value having been read. After the flip both give `["logistics"]` with the same unit, because the three fields ship at `FallbackUnit` on purpose, so the arm's unit AND its prerequisite are exactly what a `.dat` that never arrived produces. **The property is replaced rather than noted.** The separator that survives is the library's own log line, `fkrecipes: bbb-balancer takes its research cost from better-belt-balancer-tech-packs: ...`, which is written only where the dropdown is on `custom` and which a tier does not write at all: a run of this package with no `.dat` at all, which is the base golden arm's own configuration, records ZERO `fkrecipes:` lines, measured on 2.0.77 in the same four runs. It was already asserted, and this commit **moves it to the FRONT of the arm**, which is `check_speed`'s ANTI-VACUITY FIRST discipline, and rewrites its failure text to say what it now stands for. Editing a second field instead was rejected: this arm's whole subject is the state where nothing is edited.

**Red-proven four times and controlled twice, each injected locally in this mod's code or its gate, observed, reverted and the revert cmp-verified.** FkRecipes stayed read-only. The last two rows are the review round's and pin the COUPLING rather than the placement.

| injected | what fired |
|---|---|
| `TECH_CUSTOM_ARMS`' first arm run with `startup=None`, so no `mod-settings.dat` reaches the engine at all | its NEW FIRST assertion, before either claim: `FAIL tech-custom-default: the library's log lines are []` / `and one of them has to be 'fkrecipes: bbb-balancer takes its research cost from better-belt-balancer-tech-packs: count 20, time 15, packs 1 automation-science-pack': without it the dropdown was not on custom, so the unit and the prerequisite below are the default tier's and prove nothing`, gate exit 2 |
| **the control for it**: the same `startup=None` with that assertion DELETED | `ok   tech-custom-default        20 x 15s in ['automation-science-pack'], after logistics` and `check-datastage: ok`, **gate exit 0**. The unit and the prerequisite alone cannot see a `.dat` that never arrived, which is the vacuity in the flesh |
| `Position` back to `[]string{TechLogistics3, TechLogistics2, TechLogistics}`, host layer | four tests, six failures: `custom untouched: the prerequisites are ["logistics-3"] and the cost came from "logistics"`, the same for `a written research cost` and for both arms of `TestAnUnreadableResearchNumberTakesTheDeclaredDefault`, and `TestThePositionLadderStepsUpAndThenLetsGo` on both its step row (`logistics absent: the prerequisites are ["logistics-3"] ... "logistics-2"`) and its whole-stream comparison, whose second line came back `none of logistics-3, logistics-2, logistics is present` |
| the same revert, `make mod` (exit 0) and the ENGINE layer | `FAIL tech-custom-default: the prerequisite is ['logistics-3'], not ['logistics'] -- a written cost is placed by the plan's own ladder, and nothing else places it`, and `tech-custom` with the identical text, gate exit 2 |
| `TechDefault` returning `TechOptions()[1]`, so the two ends of the coupling part (the review round) | `TestTheCustomArmTakesItsPlaceFromTheDefaultTier` by name on its first assertion: `custom untouched hangs off TechDefault(): the prerequisites are ["logistics"] and the cost came from "logistics-2"`, plus its without-`logistics` half. `go test ./tune/` exit 1 |
| **the control for it**: `TechOptions()` REORDERED to `logistics-3, logistics-2, logistics` instead | both coupling assertions stay GREEN, which is the by-construction claim measured: the menu order, the default and the Custom arm's head move together. The test's second half, which names the tier strings because it is about one particular mod set, fires as designed |

**What moved, file by file.** `guest/go/tune/plan.go`: the slice, and the "IT STARTS AT `logistics-3`" argument above it, replaced by the one that is now true and by the measurement. `guest/go/tune/tech.go`: `TechOptions`' own doc comment says it is the Custom arm's ladder as well, so a reader who reorders that list is told the order is load-bearing twice over, and `TechDefault`'s says the same thing from the other end, because a coupling annotated at one end only is half a warning. `guest/go/tune/plandata_test.go`: three `checkPrereqs` calls, the ladder test's step table, and one new test, `TestTheCustomArmTakesItsPlaceFromTheDefaultTier`, which states the property in the terms the argument uses (`TechDefault()`, not the literal `TechLogistics`) and carries the without-`logistics` pair above; `TestThePositionLadderStepsDownAndThenLetsGo` renamed `...StepsUp...`, because the ladder walks UP the tiers now and a test name asserting the opposite of what it does is what three of commit 1's five renames were for, and its step table inverts (`logistics absent` gives `logistics-2`, `only logistics-3 is left` gives `logistics-3`) along with the dropped-ladder line, whose ORDER is the ladder's. `test/check-datastage.py`: the two `TECH_CUSTOM_ARMS` expectations, the two comment blocks that say where the prerequisite comes from, the arm's own comment, and the reordered driver. `mod-data/locale/en/better-belt-balancer.cfg`, `mod-data/changelog.txt` and `README.md`: the same sentence in three voices, each now saying that picking Custom does not MOVE the research rather than only where it sits. **0.3.3 IS UNPUBLISHED, which is the whole reason its changelog entry is editable**; a shipped section is the one edit a changelog must not take, and after release this flip would have had to be a `Changes:` entry in a later version instead. `CLAUDE.md`: the ladder sentence, the behaviour sentence under Round three, the deleted mismatch paragraph, the structural-anti-vacuity sentence, and the Fix round 1 section, which gains the flip and a paragraph carrying the re-taken proofs. **Round three's red-proof row keeps round three's injection and round three's observation** and is ANNOTATED where an earlier draft rewrote it: its injection was the ladder reduced to `logistics`, which is the ladder's own head since the flip, so it reddens nothing, and the row says so and points at this round. That is this file's own habit for a superseded dated number, and a dated table is the last place a later proof belongs.

**What did NOT move, and it is a decision rather than an oversight.** Round three's and the sync pass's own sections in this note record what those rounds measured, with their commits named, exactly as `agents/migration-assessment.md` and a published changelog section do; they are annotated with the supersession where the number is one a later reader would take for head's, and not rewritten. The tier ladder `TechLadder` is untouched in both halves: it is the COST ladder and it still walks down from the tier the player asked for, which is the opposite direction on purpose and is why the two are separate functions. `tune_test.go`'s ladder pins and the tier arms' `checkPrereqs` calls do not move for the same reason.

**AN ADVERSARIAL REVIEW RAN BEFORE THE COMMIT, found nothing blocking, and changed six things.** It re-took the flip's host proof, the anti-vacuity proof and that proof's control itself, all three with the text recorded, and took a fourth of its own: the without-`logistics` probe whose table is above. What it changed. **The changelog generalised where the measurement does not**: "Where the research sits in the technology tree does not move either" was asserted flatly and then extended over the branch where it is false, and it now carries the condition and states the without-Logistics case as its own clause; the same overclaim in `CLAUDE.md` and in `plan.go` is scoped the same way, each in its own register, and the probe is a permanent test rather than a number in prose. **A dated table was rewritten rather than annotated**: round three's red-proof row had this round's proof written into it, under a header naming round three's date and round three's scratchpad, which is the one thing three other paragraphs in this same commit decline to do; the row is restored and annotated and the re-taken proof lives under Fix round 1. **The sync pass's rename-cost measurement was the one stale transcription the sweep missed** and is annotated like its two `CLAUDE.md` twins, and round three's "rung by rung down" grading with it. **The coupling was annotated at one end only** and now has a test and two proofs of its own, above. **A timing range was not reproducible from the kept logs** and is re-taken below, three consecutive runs, each logged. **"Cost gates softly where a prerequisite gates hard" read as measured and is not**; it is marked as the reasoning it is, in both places it appears, rather than deleted, because the choice between the two errors turns on it.

### Commit 2's gates, at their exit codes

Factorio 2.0.77 (build 84539, mac-arm64, steam), the binary re-asked its version first (`"$FACTORIO" --version`), exit codes read directly. Re-run whole after the review round.

| gate | exit | |
|---|---|---|
| `cd guest/go && go test ./tune/` | 0 | |
| `../FkLua/bin/fklua gen-bindings --check` | 0 | the committed bindings unmoved |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)` |
| `make check` | 0 | gofmt included; `gofmt -l guest/go` prints nothing |
| `make mod` | 0 | `test/check-changelog.py` runs inside it and passes on the edited entry: `6 version section(s), grammar ok, top section matches manifest 0.3.3` |
| `make datastage-check` | 0, 0, 0 | **EIGHTEEN arms**; three consecutive runs, `/usr/bin/time -p make datastage-check`, **real 36.25 s, 37.11 s, 36.78 s**, each run's whole output kept (scratchpad `fix1bbb/c2b-datastage-run1.log` to `run3.log`) and each exit code read directly. Both hashed mod sets unmoved, `tech-custom-default` and `tech-custom` reporting `after logistics` |
| `python3 -m py_compile test/check-datastage.py` | 0 | |
| agents/docs-style.md's greps over README.md | empty | the one human-facing file this commit touches |
| the em-dash and en-dash sweep over every added line | empty | `git diff HEAD -U0 \| grep '^+' \| perl -CSD -ne 'print if /[\x{2013}\x{2014}]/'` |
| `make test`, the 2.1.16 and 2.1.17 golden rows, the client run | **NOT RUN** | unchanged from commit 1 and from every round before |

**`test/datastage-goldens.json` IS UNTOUCHED BY THIS COMMIT, and that is a measurement rather than an absence.** The two hashed arms install no `mod-settings.dat` at all, so they run on the prototypes' own defaults and never reach the Custom arm; the flip moves a dump only where `bbb-tech-cost` is stored as `custom`, and the arms that store it are not hashed. Both `data_raw_sha256` and both `mod_settings_sha256` are what commit 1 captured.

### Commit 3: the recipe dropdown says Custom, and two effects nobody can refuse go in the changelog

**THE `bbb-recipe-cost` DESCRIPTION WAS PUBLISHED 0.2.2's, WORD FOR WORD, AND IT PROMISED THE WRONG POLICY OF THE OPTION THIS ROUND ADDED.** It read "What a balancer part costs to craft. The default is what this mod has always used. If an option names something your mods do not have, the nearest thing they do have is used instead." That sentence is true of the six presets, each of whose ingredients is a ladder ending at iron plate, and FALSE of the seventh: on Custom a name the game lacks is not swapped for a neighbour, the whole typed list is set aside and this mod's own declaration applies. One sentence covered two opposite policies, which is `agents/migration-assessment.md` finding 6, and the twin `bbb-tech-cost` had already been rewritten to name Custom in round three while this one was not touched. It now reads:

> What a balancer part costs to craft, or Custom to write the recipe yourself in the setting below. The default is what this mod has always used. Every option but Custom is safe in an overhaul pack: where one names something your mods do not have, the nearest thing they do have is used instead, and where that leaves one item named twice the two amounts are added, so what you craft can be a shorter list than the one shown here. A recipe you write yourself is the opposite on purpose, and it is taken as written: nothing is substituted for a name you typed.

557 characters against the old 182 and the twin's 622. **The twin's figure is re-measured and the earlier 520 was stale inside this very round**: `git show <rev>:mod-data/locale/en/better-belt-balancer.cfg | grep '^bbb-tech-cost=' | perl -CSD -ne 'chomp; print length($_)'` gives 520 at `ca9aa36` and at `3fd5875` and 622 since `ff647c2`, which is commit 2 of this round rewriting it. **The scope is written "every option but Custom" rather than "the options listed below"**, which the drafting pass had reached for: "below" is exact in the tooltip, where the composed description carries one label line and one indented `type:` line per PRESET and nothing at all for `custom`, and wrong in the widget, where the open dropdown the player is reading it beside holds seven values. "Every option but Custom" needs neither the tooltip's layout nor a count of presets that a seventh preset would falsify.

**AND THE SCOPED PROMISE HAD TO CARRY THE MERGE, which the adversarial review found, and which is finding 6's own shape in a sentence THIS ROUND wrote.** The draft stopped at "the nearest thing they do have is used instead", which reads as the same list with one item swapped. Commit 1 chose otherwise and deliberately: where a substitute is an item the list already names the two amounts are ADDED, so in a pack with no `transport-belt` the `vanilla` preset crafts `6 iron-plate, 2 iron-gear-wheel` while the `type:` line beside its own label still promises `4 iron-plate, 2 iron-gear-wheel, 2 transport-belt`. Three ingredients become two and the tooltip goes on showing three. One clause fixes it and the shipped sentence carries it: "and where that leaves one item named twice the two amounts are added, so what you craft can be a shorter list than the one shown here." `README.md` already spelled the merge out in commit 1, with the same worked example; what it still carried was the CIRCULAR reassurance "A recipe naming an item nobody defined would refuse to load the game, so it cannot happen", which assumes what it concludes. Every ladder terminates at `tune.go`'s `FallbackName`, `iron-plate`, so that is the actual assumption, and README now names it: "That last rung is the one thing the presets do assume: a mod set with no iron plate is outside what they cover."

**WHAT IT DELIBERATELY DOES NOT SAY IS MEASURED RATHER THAN GUESSED.** FkRecipes `0077f3c` composes two sentences onto a text setting's description, giving the vocabulary and the 2000-character limit and saying what becomes of a text the mod cannot use. Read straight out of the engine's own `mod-settings-dump.json`, they reach the two FIELDS (`better-belt-balancer-recipe-ingredients` after `"\ndefault: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt"`, and `better-belt-balancer-tech-packs` after its own) and NEITHER DROPDOWN. So the format, the limit and the fallback are the field's own small print, and the field is the row IMMEDIATELY below the dropdown rather than three rows below it, which the earlier draft said and the adversarial review corrected: `bbb-recipe-cost` carries `order = a` and `better-belt-balancer-recipe-ingredients` `order = aab` in the engine's own settings dump, and `agents/migration-assessment.md`'s client run recorded the Startup tab rendering six rows as `a`, `aab`, `b`, `bad`, `bae`, `baf`. (That assessment's own "two rows below", in its finding 6, is wrong in the other direction and is left where it is: it is a dated record with its commit named.) Repeating those lines on the dropdown would put them in front of every player who only ever picks a preset. The six preset lists are not repeated either, because the six composed `type:` lines already carry them. `Custom` is named in the twin's shape and in the twin's position, first sentence, singular where the twin is plural because there is one field and not three.

**THE WORDING OF EVERY DESCRIPTION IN THIS FILE IS UNGATED, AND THAT IS THE FINDING THE ITEM CAME WITH.** `--dump-data` does not read locale, so the settings dump holds the KEY (`["mod-setting-description.bbb-recipe-cost"]`) and never the string. The drafting pass measured that on a draft of this entry (installed, `make mod` re-run, `mod_settings_sha256` back at `aa75497e55ce4333`, byte-identical to the run before the edit), and THIS COMMIT'S OWN `make datastage-check` re-measures it on the shipped text: exit 0 with `test/datastage-goldens.json` untouched, over a commit whose only prototype-visible change would have been the description if the dump carried one. `test/datastage-goldens.json` cannot see a syllable of the .cfg. `guest/go/tune/locale_test.go`'s `TestEverySettingThisPlanDeclaresIsDescribed` checks `strings.TrimSpace(...) != ""` and nothing else. **This commit closes the one property worth closing rather than the whole gap**, with a fourth test of this mod's own, `TestEveryCustomDropdownDescriptionNamesTheCustomOption`: each dropdown that declares a Custom value must name it IN ITS FIRST SENTENCE, and the word demanded is the one the MENU shows for that value, read out of `[string-mod-setting]` and cut at its colon, so the two entries are tied to each other the way `TestTheGrandfatherMessageQuotesTheRealMenuLabel` ties its pair. It reads no further into the sentence on purpose: what the entry says ABOUT Custom is a human's to judge, and pinning it would be a transcription that fails on every rewording.

**THE FIRST-SENTENCE NARROWING IS THE ADVERSARIAL REVIEW'S, AND IT IS A RED PROOF RATHER THAN A PREFERENCE.** The draft demanded the word ANYWHERE in the entry, and the review broke that in one edit: delete `, or Custom to write the recipe yourself in the setting below`, which is the only clause that tells a player the field exists, and the entry still says "Custom" three sentences later in "Every option but Custom is safe in an overhaul pack", so the check passed on a description that had lost the exact route it claims to guard. A second probe was worse: relabel the Custom value `The default: ...` and the demanded head was already in the entry for an unrelated reason ("The default is what this mod has always used"), so a label that no longer says Custom at all passed vacuously. The check now cuts the entry at its first full stop and demands the word there, which is where the twin `bbb-tech-cost` names it and where a player who reads one line of a tooltip will see it. **A rewording that moves the word out of the opening sentence therefore FIRES, and is meant to**: it is a standard this file holds itself to, not a claim that a later mention would be useless, and the test's own comment says so and says what it still does not gate.

**Red-proven four times on the narrowed form, each injection made in this mod's own .cfg, observed, reverted and the revert cmp-verified.** The first two are the original pair re-taken; the last two are the review's probes, both of which the draft had passed.

| injected | what fired |
|---|---|
| the clause `, or Custom to write the recipe yourself in the setting below` deleted, the rest of the entry untouched | `[mod-setting-description] bbb-recipe-cost does not say "Custom" in its FIRST SENTENCE, and "bbb-recipe-cost-custom" is what the menu calls the value that switches the text field under it on: a player who reads the opening line of this tooltip is never told the field is there. The first sentence reads "What a balancer part costs to craft"`, `go test -count=1 ./tune/` exit 1. **The draft passed this, exit 0** |
| the entry restored to the published 0.2.2 text | the same message, exit 1. **In the same run `TestEverySettingThisPlanDeclaresIsDescribed` PASSED**, which is the gap in the flesh: the old test reads that exact defect as green |
| `bbb-recipe-cost-custom`'s label renamed to `Your own recipe: ...`, the entry left as this commit writes it | the same test, `does not say "Your own recipe" in its FIRST SENTENCE`, exit 1. The check reads the menu rather than a literal, so the two entries cannot drift apart |
| `bbb-recipe-cost-custom`'s label renamed to `The default: ...` | `does not say "The default" in its FIRST SENTENCE`, exit 1. **The draft passed this too**, off the unrelated second sentence |

**THIS DECISION IS REVERSED BY FIX ROUND 2'S THIRD COMMIT, 2026-09-14, AND IT AND THE TABLE UNDER IT ARE KEPT AS THE DATED RECORD THEY ARE.** All nine labels are NAMES now (`Default`, `Cheap`, `Fast belts`, ...), which is neither of the two spellings this section weighed. What it measured stands: the counts below are exact for the eleven labels of the day, and the internal-name rewrite it priced really would have lengthened three of the six. What changed is the QUESTION, not the arithmetic, and the third commit's subsection says why. Note also that the eleven counted here are ten past the truncation because two of them, `bbb-recipe-cost-custom` and `bbb-tech-cost-custom`, were withdrawn with the value in the adopt commit.

**THE SIX PRESET LABELS KEEP THEIR DISPLAY-NAME SPELLING. That is the assessment's item (h), which FkRecipes carries back as item 4 of "What BetterBeltBalancer must do to adopt this" ("leave the preset labels alone, or rewrite them, but decide it"), and the answer is KEEP, on a measurement.** The client measured the CLOSED dropdown truncating at roughly 37 characters (`Default: 4 iron plates, 2 gears, 2 tra`, 38 shown, one widget at one UI scale, and that is the whole of the evidence for the width, so a label at exactly 38 sits inside that widget's noise). Measured over this file: **ten of this mod's eleven dropdown labels are already at or past it**. The six presets run 35, 38, 50, 52, 58 and 65 characters, only `splitter` at 35 is clearly under, `custom` is 52, and all four of `bbb-tech-cost`'s are over at 43, 45, 46 and 54.

**THE CLAIM THAT INTERNAL NAMES ARE LONGER FOR ALL SIX WAS FALSE, THE ADVERSARIAL REVIEW MEASURED IT, AND THE DECISION SURVIVES ITS OWN COUNTER-EVIDENCE.** The earlier draft of this section, of `CLAUDE.md` and of the `.cfg` header all said "internal names are LONGER than display names for every one of the six", and rested "a rewritten label clips further" on it. Re-taken: each of the six labels was rebuilt in the same shape (the display head, a colon, and that preset's internal list read out of the engine's own `--dump-data` settings dump), and measured.

| preset | display | rewritten in internal names | delta |
|---|---|---|---|
| `vanilla` | 50 | `Default: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt` 58 | +8 |
| `cheap` | 38 | `Cheap: 2 iron-plate, 1 transport-belt` **37** | **-1** |
| `belt-fast` | 58 | 66 | +8 |
| `belt-express` | 65 | 73 | +8 |
| `splitter` | 35 | `Splitter: 1 splitter, 2 iron-plate` **34** | **-1** |
| `splitter-express` | 52 | `Express splitter: 1 express-splitter, 2 steel-plate` **51** | **-1** |

Only `iron-gear-wheel` (15) is longer than the display name it replaces, `gears` (5); every plural loses a character (`iron plates` to `iron-plate`, `steel plates` to `steel-plate`, `transport belts` to `transport-belt` is even). So the three presets that name a gear, which are also the only three with three ingredients, grow by eight, and **the other three SHRINK BY ONE**. Note what that does and does not buy. It is not the three longest labels that grow: `splitter-express` at 52 is longer than `vanilla` at 50 and shrinks. `splitter` fits before and after; `splitter-express` at 51 is still far over; and `cheap` at 38 to 37 is the ONE label whose clipping the rewrite could change at all, and only if the width is exactly 37 rather than the 38 the client was actually seen rendering. Against that: the three that grow go to 58, 66 and 73, past every reading of the width, and they clip MID-NAME, unreadable and uncopyable at once; dropping the leading display name to buy the characters back costs the closed widget the one thing it can still show, which is which option this is; and a label in internal names would make the tooltip read the same list twice, once in the label and once on the `type:` line under it. One character on one label, against three labels made worse and a doubled tooltip, is why the decision is still KEEP. **What changed is the ARGUMENT, not the labels.** The `[string-mod-setting]` header used to rest the decision on "a readable dropdown against a copyable one"; since `0077f3c` put each preset's internal list on its own indented `type:` line, a display-name label is no longer a TRAP with the copyable half hidden behind a second colon on the same line, which is what finding 5 measured, but merely the half a player cannot paste, with the half they can beside it. The rewritten `.cfg` paragraph says all of that, carries the table above and names the counter-evidence in it.

**THE TWO CHANGELOG ENTRIES GO UNDER `Info:`, and the category is the file's own precedent** rather than a guess: 0.3.0's only entry is an `Info:` one saying how that release relates to another, which is exactly what both of these are. Neither is a **Bugfix** (nothing was fixed; both effects are the engine's and neither can be refused), a **Feature** (nothing was added by them) or a **Change** (this release changes neither path). `Compatibility` was considered and rejected: in Factorio's vocabulary it means other mods, and both of these are about this mod's own versions and this mod's own recipe. **FkRecipes asked for two changelog sentences and this commit writes one of them**, item 5 of "What BetterBeltBalancer must do to adopt this": the silent reset is the first, and the second ("a Custom cost arm moves the technology's prerequisite to the ladder's first existing entry even when all three numbers are untouched") was answered by commit 2 rather than written down, because the flip means an untouched Custom moves nothing at all in a game that has Logistics, and 0.3.3's research `Features:` entry already carries the without-Logistics case as its own clause.

> Custom is a new option in both cost settings, and no release of this mod before 0.3.3 knows it. Launching 0.2.1, 0.2.2, 0.3.1 or 0.3.2 once resets whichever setting was on Custom back to its default and writes that reset back, with nothing said in the log, so the choice is gone rather than waiting for you when you come back to 0.3.3. The older 0.1.0, 0.2.0 and 0.3.0 have no cost settings at all and leave your choice where it was. What you had typed in the settings beside it survives untouched, which is what makes it look as though nothing was lost. Check both cost settings after rolling back to an older version, after a modpack that pins one, and on a second machine playing the same save.

> Changing what a balancer part costs to craft edits the one recipe this mod has rather than adding a second one, so an assembling machine already set to make balancer parts loses whatever is in its input slots that the new list does not use. Those items are destroyed rather than dropped on the ground, with nothing said in the log. What the machine had already finished stays in its output slot, and the chests and belts around it keep everything they were holding. Empty the machine before you change the setting if it is holding something you would miss.

Every clause is one the assessment measured: findings 3 and 8. The silent reset, the write-back that persists it and the surviving typed text are the round trip head to authentic published 0.2.2 to head; the destruction, the empty `world.item-on-ground` in all 37 loads, the surviving output slot and the untouched chest are finding 8's.

**THE FIRST ENTRY NAMES FOUR RELEASES RATHER THAN "ANY OLDER RELEASE", WHICH THE ADVERSARIAL REVIEW SCOPED OFF THIS REPOSITORY'S OWN PORTAL TABLE.** "No release before 0.3.3 knows the value" is true of all seven published releases; "launching an older one resets it" is true only of the four that DECLARE the cost settings. `agents/migration-assessment.md`'s portal table has 0.1.0 declaring none, 0.2.0 declaring only the runtime bool, and 0.3.0 declaring nothing on 2.1, and its save matrix measured that a release leaves a setting it does not declare ALONE ("the four settings 0.2.2 does not declare survived untouched"). A player who rolls back that far keeps their Custom choice, so the entry names 0.2.1, 0.2.2, 0.3.1 and 0.3.2, and adds one clause for the three that are safe. **Only 0.2.2 was measured on an authentic portal artifact**; 0.2.1, 0.3.1 and 0.3.2 come by the byte equivalence the assessment states, which belongs in this note rather than in a changelog a player reads. **AND NEITHER ENTRY CLAIMS THE SCREEN SAYS NOTHING**, which was the review's other scoping: finding 3's silence and finding 8's were both measured on the LOG, headless, and the client run measured the settings SCREEN only for finding 4. Both entries now say "with nothing said in the log", and `README.md`'s sentence with them.

**What is NOT in them is the useful sentence the drafting pass wanted and could not back**: the client's `Sync mods with save` route was measured restoring PRESET values from a save's embedded startup record, never a `custom` one, so it is left out rather than promised. It is one arm for whoever next has a client.

**`README.md` TAKES THE ASSEMBLER SENTENCE AND NOT THE RESET ONE, and the split is `agents/docs-style.md`'s own rule read against both.** The assembler loss is a permanent property of the setting that a player needs before they change it, and the "Cost and research" section already carries this class of warning; it goes in at the end of that section, in that section's voice. The reset is a statement about moving between versions of this mod and is true only of releases older than 0.3.3, so it ages out, and docs-style bans exactly that in a human-facing document: "No change-history narrative ... State the current behaviour." A changelog is where a statement about versions belongs, and it is already there. **AND THE PORTAL DESCRIPTION IS NOT `README.md`**, which the item's framing invited: `README.md` is the GitHub readme, and what the portal shows is `fklua.toml`'s `description` key, one sentence about what the mod does for a megabase, which becomes `info.json`'s `description`. The portal's long description is not in this repository at all. Nothing this commit writes needs to reach either, and neither is touched. **The README sentence opens on the EFFECT and not on one setting**, which the adversarial review asked for: it said "Changing the recipe setting", and in a section that bolds **Balancer part recipe** and **Custom balancer part recipe** as two settings that reads as the dropdown alone, while editing the Custom text destroys the same stock. It now opens "Changing what a balancer part costs to craft, whether by picking another option or by editing a custom list", which is the changelog entry's own coverage.

**LEFT OPEN, NOT THIS COMMIT'S:** `bbb-tech-cost` carries no overhaul-pack sentence at all, so after this commit the two dropdowns promise different things about their presets. The recipe one says its presets are safe and substituted and merged; the research one says only that the cost is copied from that technology. The tier fallback (a missing technology, a trigger technology with no cost, none of the three) is in `README.md` and in neither tooltip. Whoever owns the twin next should decide whether it wants the same clause.

**THREE OF THE REVIEW'S NOTES WERE TAKEN AND LEFT ALONE, WHICH IS ALSO A DECISION.** (1) `localePath` hard-codes `mod-data/locale/en`, so all four of these tests see English and nothing else. Only `en` ships, so nothing is wrong today, and a note about a language file that does not exist would be a guess about how a second one would be checked; whoever adds one adds the loop with it. (2) `strings.Cut(word, ":")` takes everything before the menu label's FIRST colon rather than a word in any dictionary sense, so a future label whose readable half itself held a colon would demand only the part before it, and the failure would read as a description defect. It fails loudly rather than silently, so the fix is the comment saying what the code does, which the test now carries, and not a parser. (3) The round-one preamble at this file's line 22 is annotated to four tests while round two's commit 4 section still says "the three assertions". That asymmetry is this file's stated habit and not an oversight: line 22 is present-tense preamble and describes head, and a dated section describes the day it was written.

### Commit 3's gates, at their exit codes

Factorio 2.0.77 (build 84539, mac-arm64, steam), the binary re-asked its version first (`"$FACTORIO" --version`), exit codes read directly. Every row below was re-run in full AFTER the adversarial review's changes landed, and `test/datastage-goldens.json` hashes `66aabbff2cd5dcec5e556012d16ba06a2b640f3d5960d426fb1f86825a083f72` before and after them.

| gate | exit | |
|---|---|---|
| `cd guest/go && go test ./tune/` | 0 | run with `-count=1`, the .cfg being an input the cache tracks |
| `../FkLua/bin/fklua gen-bindings --check` | 0 | the committed bindings unmoved |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)` |
| `make check` | 0 | gofmt included; `gofmt -l guest/go` prints nothing |
| `make mod` | 0 | `test/check-changelog.py` runs inside it and passes on the new `Info:` block |
| `python3 test/check-changelog.py mod-data/changelog.txt 0.3.3` | 0 | `6 version section(s), grammar ok, top section matches manifest 0.3.3`, run standalone as well |
| `make datastage-check` | 0 | eighteen arms |
| agents/docs-style.md's greps over README.md and mod-data/changelog.txt | empty | the two human-facing files this commit touches |
| the em-dash and en-dash sweep over every added line | empty | |
| `make test`, the 2.1.16 and 2.1.17 golden rows, the client run | **NOT RUN** | unchanged from commits 1 and 2 and from every round before |

**`test/datastage-goldens.json` IS UNTOUCHED, AND FOR THIS COMMIT THAT IS THE POINT.** Everything it changes is locale text, a changelog and a Go test, and the settings dump holds locale KEYS rather than strings. A moved `mod_settings_sha256` here would have meant something in the commit was not a locale change, and the golden would have been the wrong thing to re-capture.

### Commit 4: the 2.0 release arm is honest, and the sentence about it was not

Item 5 of the fix round, which is [`agents/migration-assessment.md`](agents/migration-assessment.md)'s finding 11: `Makefile`'s OBS_TAGS block said "trunk and the release/2.0 recut carry identical source", and `git merge-base master release/2.0` is `cf5a78e`, the 0.3.2 trunk commit. The finding reads that as the BRANCH being the defect. Re-taken blob by blob, it is not.

**THE BLOB-PROVENANCE TABLE, re-taken 2026-09-09 at `master` 760d618 and `release/2.0` 4b9f597.** For each of the 14 paths `git diff --name-only cf5a78e 4b9f597` reports, the question asked was whether the blob at `4b9f597` is byte-identical to that path's blob at SOME commit reachable from `master`:

    for p in $(git diff --name-only cf5a78e 4b9f597); do
      git rev-list master -- "$p" | sed "s|\$|:$p|" \
        | git cat-file --batch-check='%(objectname)' | grep -c "^$(git rev-parse "4b9f597:$p")$"
    done

| path | on master? | commits touching it |
|---|---|---|
| `FKLUA-GAPS.md` | **`cb89891`** | 15 |
| `guest/go/data/legacy.go` | **`cb89891`** | 2 |
| `guest/go/obs/bb2data/main.go` | **`cb89891`** | 2 |
| `guest/go/obs/harness/harness.go` | **`cb89891`** | 5 |
| `guest/go/obs/mig/main.go` | **`cb89891`** | 2 |
| `test/assert-mig.py` | **`cb89891`** | 10 |
| `test/check-datastage.py` | **`cb89891`** | 14 |
| `test/datastage-goldens.json` | **`cb89891`** | 10 |
| `fklua.toml` | no match | 14 |
| `fklua.lock` | no match | 11 |
| `guest/go/fkapi/fkapi.go` | no match | 9 |
| `mod-data/changelog.txt` | no match | 9 |
| `CLAUDE.md` | no match | 68 |
| `README.md` | no match | 22 |

`cb89891` is TRUNK'S OWN freeze-fix commit. The four in the middle are exactly the documented four-file recut stamp. The last two are the working notes, and they cannot match by construction: a BACKPORT carries trunk's paragraph into an older surrounding document, so the blob differs while not one sentence is new, which the diff of either file against the cut point shows directly. **So the branch is trunk 0.3.2's source with a trunk fix backported file for file, which is what a 2.0 hotfix arm should be, and nothing a 2.0 player would receive was written on the branch.**

**WHAT IS WRONG IS THE CLAIM, in two places.** `Makefile`'s sentence, and `4b9f597`'s own commit message, which says "the guest source is trunk 0.3.3's to the byte ... so the cherry-pick touched only the version and the changelog". Both halves are false, measured: `cb89891` IS the first trunk commit whose `fklua.toml` says 0.3.3 (walked forward from `cf5a78e`), and `git diff --name-only cb89891 4b9f597` excluding the four stamp files and the two notes is TWENTY FILES, which is the whole customizer, `guest/go/tune`, the data guest and `go.mod`. Against `50c0db3`, the trunk head of the day the branch was cut and also 0.3.3, `git diff --stat 4b9f597 50c0db3 -- guest/go ':(exclude)guest/go/fkapi/fkapi.go' mod-data` is 20 files, 2880 insertions and 726 deletions. What the branch carries is trunk 0.3.2's guest source with `cb89891`'s EIGHT freeze-fix files backported onto it, which is exactly what the provenance table above shows -- eight rows, not seven -- and is not what that sentence says. Re-counted: `git show --name-only --format= cb89891` is twelve paths and all eight matched rows (`FKLUA-GAPS.md`, `guest/go/data/legacy.go`, the three `guest/go/obs` mains, `test/assert-mig.py`, `test/check-datastage.py`, `test/datastage-goldens.json`) are among them. History here is linear and is not rewritten, so that correction lives in this note naming the commit; the changelog `Info:` line the branch ships, "The Factorio 2.0 release, carrying the same fix 0.3.3 brings to Factorio 2.1", is TRUE as written and is not a claim to carry trunk 0.3.3's source.

**THE GATE IS `test/check-release-arm.sh` AND ITS INVARIANT IS PROVENANCE, NOT IDENTITY.** Identity is true for the few hours around a release and false the moment trunk lands a commit, so a recut refreshes the misstatement instead of removing it and a gate on it would be red between releases, which is how a real failure gets ignored. What holds continuously is that every path the branch changed since its cut point carries a blob some commit reachable from `master` also carries. A recut satisfies that; so does a backport file for file; a line typed on the branch does not. It is git plumbing over two REFS and never reads the working tree.

**WHAT THAT PROVES IS PER-FILE, AND THE SCRIPT'S HEADER NOW SAYS SO RATHER THAN LEAVING THE OK LINE TO BE READ AS A TREE CLAIM.** The unit is the FILE, so no LINE on the branch is a line trunk never had; it is not a claim that the matched blobs ever COEXISTED on trunk, that the commit each came from was not later REVERTED (`rev-list` walks everything reachable and a reverted commit stays reachable), or that the branch holds trunk's LATEST value for a path rather than an older one. None of the three is reachable on today's branch, where all eight land on the single commit `cb89891`, which is strictly stronger than the gate asks, and each would still be a backport somebody made rather than a line typed here. The invariant also rests on the house rule that nothing is ever MERGED into trunk: a merged branch's own commits become reachable from `master` and every file it wrote would then match itself.

**IT ADDS ONLY `git` TO WHAT `make check` ALREADY NEEDS, and that is checked rather than assumed.** `make check` needs a Go toolchain, `gofmt` and the sibling `fklua` (`FKLUA ?= ../FkLua/bin/fklua`), and no Factorio and no network. The script runs `git rev-parse`, `git merge-base`, `git show`, `git rev-list`, `git diff` and `git cat-file`, plus the shell builtins `printf`, `sed` and `case`; there is no other executable in it. What was written here at first, "a git repository has `git` by construction", is the wrong direction and is dropped: the failure that matters is a missing BINARY, not a missing repository, and with `git` off PATH the script used to print the EXPORT's skip line at exit 0 -- an internal error wearing a skip's clothes, with the wrong diagnosis. `command -v git` is now the first probe and its absence is a FAILURE at exit 1.

**THE EXCLUSIONS ARE NAMED IN THE SCRIPT WITH THEIR REASONS AND THERE IS NO SILENT SKIP LIST.** The four stamp files are excluded because by construction they hold the 2.0 values trunk does not have, and because each has a CONTENT check of its own on the branch's own gates: `fklua gen-bindings --check` and `fklua lock --check` for the bindings and the lock, `check-changelog.py` for the changelog, and the manifest is three of the keys those read (`git show cf5a78e:Makefile` carries all three lines, so the branch runs them). `CLAUDE.md` and `README.md` are the WEAKER statement the item asked to be decided: they are REPORTED on every run and not gated, because they are not source (`make mod` packages `mod-data/` and the built wasm, and neither file reaches the package) and because failing them would be a permanently red gate on a legitimate hotfix arm. The justification is that a backport NEED NOT make them match, not that it cannot: the script said "cannot" and its own counterexample sat two lines below, which is `FKLUA-GAPS.md`, a human-facing document too, deliberately NOT excluded, whose blob on the branch IS trunk's. That is the measurement saying a document can survive a backport untouched rather than an assumption that it will, and it is exactly why the pair is a decision about REPORTING rather than a claim about what a backport can do. One more thing the next recut will meet: `test/datastage-goldens.json` is not excluded either, so a 2.0 golden re-captured ON THE BRANCH fails by name -- the right answer, since the row is trunk's emission, but a surprise better met in this note than in a red gate. The list names files rather than a directory, because an exception that names files is auditable and one that names a directory grows without anybody deciding to.

**GREEN FIRST, on the branch as it stands**, because a check that has never been green proves nothing:

    release-arm: ok -- release/2.0 0.2.3 carries no line `master` never had.
    release-arm:   14 path(s) differ from the cut point cf5a78e (trunk 0.3.2); master is 21 ahead, which is normal between releases.
    release-arm:   8 carry a blob some commit on `master` carries byte for byte.
    release-arm:   4 are the recut's stamp: fklua.toml fklua.lock guest/go/fkapi/fkapi.go mod-data/changelog.txt
    release-arm:   2 are the recut's own working notes, reported and not gated: CLAUDE.md README.md
    exit=0

The `8` is the anti-vacuity: a gate that checked nothing would say `0`.

**COST: 0.23 s**, `/usr/bin/time -p`, ten consecutive runs spanning 0.22 to 0.24, and 0.22 to 0.25 over five runs in a fresh `--no-hardlinks` clone. A bare "0.20 s" was what three sites said before; it was the mode in the repository it was taken in rather than a ceiling, and the guards have since added about 0.02 s to it, so all three now carry the band. The bound is two git processes per path that differs, over that path's own commits and no others, and what the 0.2 s IS is process spawn rather than history: everything before the loop is twelve git spawns and 0.09 s, the widest walk of a CHECKED path (`FKLUA-GAPS.md`, 15 commits) is under 0.01 s, and 16 bare `git rev-parse HEAD` calls on this machine are 0.10 s, so 28 spawns for 8 checked paths is the number. The 8 run over 2 to 15 commits each. Batching the whole loop into three processes would buy about 0.13 s and cost the per-path diagnosis that names the offending file, which against a `make check` running six Go test packages and three wasm vets is not where that trade goes. It is linear in the number of paths a recut moved, which is four when the branch is healthy.

**A MISSING BRANCH IS A SKIP AT EXIT 0, PROVED AGAINST A REAL CLONE rather than reasoned about.** `git clone --no-hardlinks --single-branch --branch master` of this repository, which has only `master` and `origin/master`:

    release-arm: SKIP -- no `release/2.0` in this clone, so the two arms cannot be compared.
    release-arm:   This is a single-branch clone, not drift.
    release-arm:   `git fetch origin release/2.0:release/2.0` to gate it.
    exit=0

and the remedy line was then run VERBATIM in that clone, which fetched the branch and turned the same invocation green, so the instruction is measured and not plausible. An EXPORT with no repository at all (`git archive master | tar -x`) skips the same way, exit 0, naming what it is.

**IT NEVER READS THE WORKING TREE, proved three ways in the clone, each byte-identical to the run on `master`**: from `release/2.0` itself; from a detached worktree (`git rev-parse --abbrev-ref HEAD` is `HEAD`); and with `guest/go/data/legacy.go` deliberately dirty in the working tree with a line that exists on no commit anywhere, which the gate does not see because it compares refs.

**THREE RED PROOFS AND ONE CONTROL, all in a `git clone --no-hardlinks` under $SCRATCH so no object was ever written into the real repository, and nothing in the real repository was reverted because nothing there was broken.** Each injection was confirmed present (a `git diff --stat` and a moved branch tip) before the gate was asked.

- **A LINE TYPED STRAIGHT ONTO THE BRANCH.** `package main` in `guest/go/data/legacy.go` given a trailing comment, committed on `release/2.0`:

      release-arm: FAIL -- `release/2.0` 0.2.3 carries source no commit on `master` ever had.
      release-arm:   written on the branch, or cherry-picked from somewhere that is not trunk:
      release-arm:     guest/go/data/legacy.go
      release-arm:   The 2.0 arm is trunk's source with a four-file stamp on it, so every other
      release-arm:   line it carries has to exist on trunk. Two ways out: land the change on
      release-arm:   trunk and recut the branch from the trunk commit that carries it, or take
      release-arm:   it off the branch. See agents/single-edge.md, "Packaging: one tree, two
      release-arm:   releases", and the Makefile's OBS_TAGS block.
      exit=1

- **A CHERRY-PICK FROM A BRANCH THAT IS NOT TRUNK.** `git checkout verify/0.3.3-2.0 -- test/assert-edge.py guest/go/edgemode` onto `release/2.0`, three files moved. The gate names the three whose content matches no master commit, and `guest/go/edgemode/edgemode_test.go`, which that branch carries at trunk's own blob, is correctly NOT among them:

      release-arm:   written on the branch, or cherry-picked from somewhere that is not trunk:
      release-arm:     guest/go/edgemode/curve_test.go
      release-arm:     guest/go/edgemode/edgemode.go
      release-arm:     test/assert-edge.py
      exit=1

- **A FILE TAKEN OFF THE BRANCH.** `git rm test/check-sprites.py` on `release/2.0`, which is the arm a recut cannot produce, because a recut copies trunk's tree:

      release-arm:   deleted on the branch, and the cut point cf5a78e has it:
      release-arm:     test/check-sprites.py
      exit=1

- **THE CONTROL, which proves the exclusion is live and deliberate rather than an accident.** The same kind of edit appended to `CLAUDE.md` on the branch and committed leaves the gate at the green line above, exit 0. That is the designed behaviour and it is what the "reported and not gated" wording in the ok line is for.

**AND THE SECOND RED PROOF FOUND A REAL DEFECT IN THE GATE, which is the reason this note names it rather than the reason it hides it.** The first draft read the branch's own blob and trunk's candidates out of ONE `git cat-file` batch and separated them by stripping up to the first newline. That is a no-op when there is no newline, and a file trunk NEVER HAD has no `rev-list` output at all, so its one-line batch left the branch's blob standing in for trunk's candidates and it compared equal to itself. The cherry-pick of three files was reported as two, and the one that got through was `guest/go/edgemode/curve_test.go`, the file trunk does not have. The guard is explicit now (`case $seen in *newline*) rest=...;; *) rest= ;; esac`) and the type is what tells a blob from a missing path, because length alone would not: `release/2.0:` plus a 20-character path plus ` missing` is also 40 characters. Re-taken, all three reds and the control read as above.

**THE CLAIMS REWRITTEN, all of them re-verified with `git grep -n "release/2\.0"` at HEAD rather than taken from the deep dive's line numbers.**

| where | was | is |
|---|---|---|
| `Makefile`, the OBS_TAGS block | "trunk and the release/2.0 recut carry identical source" | the tag reasoning is unchanged and correct; the recut is trunk's source with a four-file stamp, the branch is BEHIND trunk between releases and that is normal, and `test/check-release-arm.sh` is what refuses a branch carrying something trunk never had. Points at `agents/single-edge.md`, "Packaging: one tree, two releases" |
| `agents/single-edge.md`, "Packaging: one tree, two releases" | "Kept rebased on master per house rule" | recut from master AT RELEASE TIME and otherwise behind; the mod-data tree is identical AT A RECUT; a commit written on the branch is a gate failure, and which gate. The correction is dated in place |
| `agents/single-edge.md`, the status block at the top | the setting-flip legs and the multi-edge regression run "belong on the `release/2.0` branch" | they need a 2.0 BINARY and phase 9 ran them on one, from THIS tree, with the manifest flipped as a local uncommitted state and `test/run.sh` stamping every staged copy. Dated in place. Found by the adversarial review, and it is the same route misdescription as the four below. Phase 9's own dated section keeps its own wording, which is this file's habit: present-tense preamble describes head, a dated section describes its day |
| `fklua.toml` | "The mod-data tree is IDENTICAL on both branches" | identical AT A RECUT, and behind trunk between releases, with the gate named |
| `guest/go/data/settings.go` | "the `release/2.0` recut carries these two identically" | carries these two NAMES unchanged, because it carries this file unchanged; what it offers under them is whatever trunk release it was cut from offered |
| `guest/go/engine/engine.go` | the 2.0 golden "is deferred to the `release/2.0` recut" | deferred to wherever a 2.0 BINARY is |
| `guest/go/engine/engine_test.go` | "have no dump golden until the release/2.0 recut takes one" | until a 2.0 BINARY takes one, from this tree through the cross-series stamped path |
| `test/check-datastage.py`, three sites | the stamped path is "for the `release/2.0` recut"; the 2.0 flavour "is captured wherever a 2.0 binary is, which is the `release/2.0` recut"; `DEFERRED_OTHER_FLAVOUR`'s "Wherever a Factorio 2.0 binary is -- which is the `release/2.0` recut" | all three say the ROUTE: this 2.1-pinned tree, dumped by a 2.0 binary, with the staged info.json clamped down. That route is trunk's and it is how every re-capture since the first was taken |
| `Makefile`'s check target | the same "deferred to the release/2.0 recut" | deferred to wherever a 2.0 BINARY is |

`Makefile`'s two other mentions are untouched and were re-read to be sure: the `OBS_BASE_DEP` block, where a hard-coded `base >= 2.1.0` is a mod Factorio 2.0 refuses at the loader, makes no identity claim and is a measurement.

**THE 2.0 GOLDEN'S `_note` AND ITS TWIN IN `CLAUDE.md`, which the deep dive found stale and which are load-bearing.** Both opened "Captured on Factorio 2.0.77 from the release/2.0 recut at 0.2.1, which is the only place it can be taken". Only the FIRST of that row's five hashes was the recut's; the `_note`'s own chain, `196275f867f7f8b5 -> 258d44de08b43c9e -> 0bea93fa1594052e -> e5e7d159e88f4a6b -> df4d6c3fc7bf854d -> aa75497e55ce4333`, records FIVE re-captures since, every one of them TRUNK's through the cross-series stamped path, because the branch never carried the customizer at all. Five is measured rather than counted by eye, and the same five is what the golden's `_note` and `CLAUDE.md` now say: five arrows in the chain, five `RE-CAPTURED` markers, and walking the file's own history -- `git rev-list --reverse --full-history master -- ':(top)test/datastage-goldens.json'`, reading `2.0.77.base.mod_settings_sha256` out of each commit -- gives the first capture at `ca9d448` (the recut at 0.2.1) and the five moves at `50c0db3`, `6ec3937`, `33657f5`, `f3029b6` and `3fd5875`, all trunk. Both now say what is true: the row is taken wherever a 2.0 BINARY is, from this tree, because `guest/go/data/engine.go` keys on the RUNNING engine and not on the manifest; it was FIRST taken on the recut at 0.2.1; and THIS ROW IS TRUNK'S 2.0-FLAVOUR EMISSION rather than the branch's, which carries this file as TRUNK had it at `cb89891`, frozen there at 0.2.1's `196275f867f7f8b5`, with nothing comparing the two. Not "its own copy": the branch's blob for this file IS a trunk blob, which is the provenance point, and what is frozen is which trunk commit it came from. **NO HASH WAS RE-CAPTURED**: the non-`_note` content of `test/datastage-goldens.json` is identical before and after, compared key by key.

**WHAT THIS COMMIT DOES NOT DO.** It creates, deletes, moves, resets and recuts no branch, commits nothing on `release/2.0` and pushes nothing. The branch's own 0.2.3 changelog entry is the one player-facing document that cannot be fixed without touching the branch, and it stands as it is until the next recut; it is TRUE as written, and the gate is what makes the claim under it checkable. Nothing here says what the MOD PORTAL serves for Factorio 2.0: `origin/release/2.0` is `f8db95f` (0.2.2) and the local 0.2.3 is unpushed, but the portal is not this git remote. NOT MEASURED.

**AND FINDING 11 IS A DATED RECORD WITH ITS COMMIT NAMED, so it is corrected here and not in it**, which is this file's own habit (round one's superseded table, round three's red-proof row, and finding 6's "two rows below" under commit 3). Three corrections: the branch is honest and the claim is the defect; `git rev-list --count release/2.0..master` is 21 today where the finding says 16 (it was 17 at the assessment commit `d8dc79e` itself, so the finding's own number was taken one commit early); and `4b9f597`'s commit message carries a false sentence. Item (i) of the assessment's "what to do" list, "whichever is meant to be true, they must agree before a 2.0 release goes out", is answered by making the claim the true one and gating it, rather than by recutting a branch onto a head whose release nobody is publishing.

### The adversarial review of commit 4, and the vacuous green it found in the gate

**THE REVIEW FOUND ONE BLOCKING ITEM AND IT IS THE FAILURE THIS ESTATE FORBIDS BY NAME: THE GATE ITSELF COULD PRINT ITS OK BANNER HAVING CHECKED NOTHING.** `BASE=$(git merge-base "$TRUNK" "$ARM")` had no error check under `set -uo pipefail` with no `-e`, so a `merge-base` that answered nothing left `BASE` empty, `CHANGED` empty, every counter at 0, and the LAST thing printed was `ok -- ... 0 path(s) differ`, at exit 0, with git's two `fatal:` lines buried above it on stderr. Two live routes were measured, both re-taken here after the fix in a throwaway `git clone` under $SCRATCH:

| route | before | after |
|---|---|---|
| `git clone --depth 1 --no-single-branch`, then `git branch release/2.0 origin/release/2.0` | two `fatal:` lines, then `ok ... 0 path(s) differ from the cut point  (trunk 0.3.3)`, **exit 0** | `SKIP -- this is a shallow clone, so the cut point is not in this history.` with `git fetch --unshallow` as the remedy, **exit 0** |
| an ORPHAN `release/2.0` in a normal clone, no common ancestor | the same banner, `master is 108 ahead`, **exit 0** | `FAIL -- master and release/2.0 have no common ancestor, so release/2.0 was not cut from trunk at all`, **exit 1** |
| `version = "9.9.9-INDEX"` staged in the shallow clone | `(trunk 9.9.9-INDEX)`, read out of the INDEX | the shallow SKIP, and in a full clone the same staging changes not one word of the green line |

**AND THE SAME LINE READ THE INDEX**, which falsified the script's own headline that the working tree is never read: with `BASE` empty, `git show "$BASE:fklua.toml"` is `git show ":fklua.toml"`, which is git's spelling for stage 0 of the index. That is the sibling of the defect the implementer's own second red proof had already found in this same script -- a comparison that passed by comparing a value with itself, reaching the identical `ok` wording -- and it is why the anti-vacuity sentence in this note ("the `8` is the anti-vacuity; a gate that checked nothing would say `0`") was describing the failure mode rather than excluding it.

**THE FIX IS THREE THINGS AND THEN AN AUDIT.** A shallow clone SKIPs at exit 0, probed with `git rev-parse --is-shallow-repository` BEFORE anything is asked of history, because a truncated history would also make the per-path walk report a legitimate backport as drift; no common ancestor FAILs at exit 1, because two branches with no shared history is the drift this gate exists for and not an environmental difference; and every command substitution whose emptiness could pass for an answer is guarded, six of them by an explicit non-empty check that names which answer git could not give.

**THE AUDIT FOUND A THIRD INSTANCE OF THE SAME SHAPE, one level down, and it did not need a broken `merge-base` to reach.** `CHANGED=$(git diff --name-only "$BASE" "$ARM")` was unguarded too: with a perfectly good `BASE`, a `git diff` that failed for any reason would give the identical `ok ... 0 path(s) differ` at exit 0. It is guarded on status now AND asserted on: the two TREES are compared, and `0 path(s) differ` with the arm's tree differing from the cut point's is an internal failure rather than a pass. Two more calls inside the loop were unchecked in the weaker way -- a failing `git rev-list` (in a process substitution, whose status is unreadable, so it is a command substitution now) or `git cat-file` would have failed LOUDLY with the WRONG reason, naming the path as written-on-the-branch or deleted; both are guarded, and the loop's one genuinely-clean empty answer (a path trunk never had, which is the drift itself) says so at its own line.

**RE-PROVEN AFTER THE FIX, all in `$SCRATCH/rv4b`, four throwaway clones, nothing written into the real repository**: the ordinary green byte for byte as before (14 / 21 / 8 / 4 / 2); the line typed onto `guest/go/data/legacy.go` still red by name at exit 1; `git rm test/check-sprites.py` still red on the deletion arm; the single-branch SKIP still exit 0 with its remedy, which fetched the branch and turned the same invocation green; the export SKIP; and two NEW red proofs of the guards themselves, by breaking the guarded code and observing the designed message -- `CHANGED` forced empty gives `FAIL -- release/2.0's tree differs from the cut point cf5a78e and yet no path came back as differing, which cannot happen. This gate checked NOTHING`, and `BASE_VER` forced empty gives `FAIL -- git could not answer the cut point's version out of fklua.toml`. Both restored, and the script `cmp`s equal to the repository's.

**THE FOUR SHOULD-FIX ITEMS ARE TAKEN.** (1) Three sites of this commit disagreed on the number of golden re-captures -- "the last four", "the four re-captures recorded below", "five" -- and the honest number is FIVE, re-measured here and written into all three with the command that produces it. (2) "SEVEN freeze-fix files" contradicted this commit's own eight-row provenance table; it is EIGHT, and all eight are among `cb89891`'s twelve changed files. (3) `git diff --name-only` is config-sensitive: with `diff.relative=true` set and the gate run from a subdirectory it reported three files as branch deletions, a FALSE FAIL, measured. `--no-relative` fixes it and `-- ':(top)'` does NOT, which the review offered as an alternative and which was tested and rejected: `diff.relative` scopes the diff to the cwd as well as relativising the output, so the pathspec still comes back with three cwd-relative paths. `core.quotePath=false` went in beside it, so a non-ASCII path comes back raw rather than C-quoted. (4) With `git` off PATH the script printed the EXPORT's skip line at exit 0 -- an internal error wearing a skip's clothes, with the wrong diagnosis -- and `command -v git` is the first probe now, failing at exit 1.

**THE NOTES TAKEN.** The ok line's "carries no line `master` never had" reads as a TREE claim; the limit is stated rather than the wording narrowed, in the script's header, in `CLAUDE.md` and above, because the per-LINE claim is exactly what per-file blob provenance proves and the things it does not prove (blobs that never coexisted, a blob from a reverted commit, an older trunk value) are worth naming rather than hiding. The script said a backport "cannot" make `CLAUDE.md` and `README.md` match while its own `FKLUA-GAPS.md` counterexample sat two lines below; it is "NEED NOT" now, in all three places. The next recut's surprise (`test/datastage-goldens.json` is not excluded, so a golden re-captured ON the branch fails by name) is written down. The cost was a bare "0.20 s" in three places, which was the mode in the repository it was taken in rather than a ceiling; all three carry the band now and the guards' own 0.02 s with it. `CLAUDE.md`'s dangling "See Verification" has its target and its full stop.

**ONE NOTE IS RECORDED AND NOT ACTED ON.** `agents/single-edge.md:31` says the setting-flip legs and the multi-edge regression run "belong on the `release/2.0` branch", which is the same route misdescription this commit corrected in three other files, and phase 9's own record two thousand lines below says that work was done on TRUNK with the manifest flipped as a local uncommitted state. It is corrected in place, in the same dated form as that file's other correction from this commit. What is NOT touched is `test/run.sh:130`, "release/2.0 carries the same suites against a 2.0 binary", which the review checked and found correct: `git diff --name-only release/2.0 master -- test/` is two files, `check-datastage.py` and `datastage-goldens.json`, neither of them a suite.

**WHAT THE REVIEW GOT WRONG, measured.** Its suggested alternative fix for the `diff.relative` finding, appending `-- ':(top)'`, does not work: run from `test/` with `diff.relative=true` it still returns the same three cwd-relative paths, because `diff.relative` limits the diff to the current directory as well as relativising it. `--no-relative` is the only one of the two that answers, and is what went in.

### Commit 4's gates, at their exit codes

Exit codes read directly, logs under $SCRATCH.

| gate | exit | |
|---|---|---|
| `bash -n test/check-release-arm.sh` | 0 | |
| `shellcheck test/check-release-arm.sh` | 0 | present at `/opt/homebrew/bin/shellcheck`, no findings |
| `test/check-release-arm.sh` | 0 | the green line above |
| `cd guest/go && go test ./tune/` | 0 | |
| `make check` | 0 | the new gate included |
| `make mod` | 0 | |
| `make datastage-check` | 0 | eighteen arms |
| `../FkLua/bin/fklua gen-bindings --check` | 0 | |
| `../FkLua/bin/fklua lock --check` | 0 | |
| the em-dash and en-dash sweep over every added line | empty | |
| `make test`, the 2.1.16 and 2.1.17 golden rows, the client run | **NOT RUN** | unchanged from commits 1, 2 and 3 |

### The round's close: the package, the proof re-run, and every gate consolidated

Both measurements below were taken AFTER the round's last code commit, on the tree at `4a1b233`, which is what makes them the ROUND's figures rather than any one commit's. Factorio 2.0.77 (`"$FACTORIO_BIN" --version` re-asked and answering `Version: 2.0.77 (build 84539, mac-arm64, steam)`), exit codes read directly, every log kept under $SCRATCH.

**THE PACKAGE.** `make mod` and `make zip` both green (scratchpad `fix1bbb/f-mod.log` and `f-zip.log`). Neither log carries an `exit=` line, so what stands for the exit code is that each ends on its target's LAST recipe line, which make prints only when nothing before it failed: `mod ready: dist/better-belt-balancer_0.3.3` for the one and the `ls -l` row for the other (`Makefile:342`). `make mod` was then re-run at `4a1b233` inside `make datastage-check` with **exit 0 read directly**, and the zip on disk is unmoved by it, because only the zip target writes the archive. `ls -l dist/better-belt-balancer_0.3.3.zip` is **836,663 B** against the sync pass's close of 817,626, so **+19,037 B, +2.33%**. THREE package members moved and every other one is byte for byte what the sync pass shipped, which is what makes the growth attributable rather than a number to shrug at:

| member | the sync pass's close (e3c0ed9) | `4a1b233` | |
|---|--:|--:|---|
| `fk_data_module.lua` | 4,469,509 B, 112,258 lines | **4,676,454 B, 118,402 lines** | +206,945 B, +6,144 lines |
| `locale/en/better-belt-balancer.cfg` | 17,608 B | 24,797 B | +7,189 B |
| `changelog.txt` | 5,584 B | 7,129 B | +1,545 B |
| `fk_module.lua` | 3,132,893 B, 88,844 lines | the same | unmoved to the byte |
| `fk_api_gen.lua`, `control.lua`, `fk_abi.lua`, the graphics and the thumbnail | | the same | unmoved |
| `dist/bbb.wasm` | 1,302,884 B | the same | unmoved |
| `dist/bbbdata.wasm` | 1,097,030 B | 1,164,294 B | +67,264 B; not a package member, the data module is compiled out of it |

Sizes by `ls -l` and `wc -l` over `dist/better-belt-balancer_0.3.3/`; the two text members by `git show ca9aa36:<path> | wc -c` against `git show 4a1b233:<path> | wc -c`, and `git diff --stat ca9aa36 4a1b233 -- mod-data` is those two files and nothing else.

**AND THE BIG MEMBER'S GROWTH IS NOT THIS ROUND'S, which the packaging report says rather than an inference.** `make mod`'s data-module line reads `data module dist/bbbdata.wasm (4676195 bytes of Lua)`, `relayed through 14 station(s)`, widest span `1018692 bytes`, at the UNTOUCHED baseline `ca9aa36` (scratchpad `fix1bbb/a0-mod.log`, taken before commit 1) and then identically at all four commits and at `4a1b233`. So the whole +206,945 B arrived the moment FkRecipes `0077f3c` was relinked, and THE ROUND'S OWN contribution to the package is the two text members, +8,734 B between them, which is commit 2's and commit 3's locale and changelog work. `grep -c '::LT'` over the data module reads 28 labels, so **14 relay stations** against the sync pass's 12; the widest span goes 991,183 B (151% of the 655,355-byte limit) to **1,018,692 B (155%)** and 328,136 to 328,253 after the relay. **The number that actually bounds the module does not move at all**: the widest block is 8,858 B in `fkrecipes.probeIn` with **318,819 B of block room (97% of the 327,677 one hop covers)** on both sides, `PlanData`'s own block 802 B, and `jumps.control` is unchanged in every field (widest span 303,874 B in `main.flushLive`, no relay; widest block 11,033 B in `main.onEvent#wasmexport`, 316,644 B of room). Read with `jq -c '.outputs, .jumps.data, .jumps.control' dist/.fklua-mod.report.json`, whose `data_lua_bytes` is 4,676,195, the 259-byte factory wrapper under the file as always.

**WHAT IS NOT ATTRIBUTED IS THE COMPRESSED SPLIT, and the reason is measured rather than shrugged at.** The zip's own +19,037 B cannot be divided between the three moved members, because the sync pass's zip was not kept and this repository's zip writer is Go's `compress/flate`, which no Python `zlib` level reproduces: `unzip -lv` gives `changelog.txt` 2,593 compressed where zlib's levels 5 and 8 give 2,630 and 2,621, and `fk_data_module.lua` 359,754 where levels 5 and 6 give 373,928 and 348,050. What IS measured is which members moved and by how much, above, and what they weigh in the archive today: the data module is 359,754 of the 833,353 compressed bytes the eighteen members sum to, the two text members 11,757 between them, and the zip's own headers are the remaining 3,310 B of the 836,663.

**THE RELEASE-TO-HEAD PROOF, RE-RUN OVER ITS 28 PAIRS AT `4a1b233`.** Driver scratchpad `sync2/prefs-head2.py`, comparison scratchpad `r3/compare-prefs.py`, log scratchpad `fix1bbb/f-prefs.log`, extracts under scratchpad `fix1bbb/head-fix1`: fourteen configurations times two mod sets, `ls head-fix1/*/*-owned.json | wc -l` reads **28**, and the driver's own log carries 28 configuration rows. The nine release `mod-settings.dat` files are installed byte for byte and the five head-only configurations go through `fklua modsettings write`, exactly as in round three and the sync pass. TWO comparisons were run, and the second is the more useful of the two.

**AGAINST THE 0.3.2 RELEASE** (scratchpad `baseline/rel032`, log scratchpad `fix1bbb/f-compare-rel032.log`, exit 1, which is what the script does whenever a prototype differs). **TWENTY pairs, because the four custom configurations exist only under head**: no release could store a `custom` dropdown value, so the release baseline has ten configurations and not fourteen. **Ten prototype differences in total and every one of them is the same field**: `grep '    PROTO '` gives ten lines and `sort | uniq -c` collapses them to one, `/item/balancer-part/place_result: "bbb-balancer-part" -> "balancer-part"`, one per configuration on the BASE arm alone; all ten incumbent arms read `IDENTICAL`. That is `cb89891`'s freeze fix and 0.3.3's changelog already discloses it. **The settings dump moves in EIGHT places per pair, twenty pairs each, 160 lines in all, and every one of them is additive**: `grep '    SET '` grouped by kind and path gives exactly eight rows at twenty each, the two dropdowns each gaining `custom` in `allowed_values` (`["vanilla", ..., "splitter-express"] -> [..., "custom"]`) and each gaining a composed `localised_description`, `better-belt-balancer-recipe-ingredients` and `better-belt-balancer-tech-packs` appearing as string settings, and the whole `int-setting` and `double-setting` tables appearing, which the release declares none of.

**AGAINST THE PREVIOUS HEAD EXTRACT** (scratchpad `sync2/head-c`, the sync pass's own, at the tree this round started from; log scratchpad `fix1bbb/f-compare-headc.log`, exit 1). **THIS IS THE ROUND'S OWN FINGERPRINT.**

- **FOUR prototype differences in total, all one field on two configurations.** `grep '    PROTO '` gives four lines collapsing to one, `/technology/bbb-balancer/prerequisites: ["logistics-3"] -> ["logistics"]`, on `tech-custom` and `tech-custom-default`, both mod sets. That is commit 2's flip and nothing else: the other twelve configurations report `IDENTICAL` on both arms, so every other prototype this mod owns is byte-identical across all 28 pairs.
- **The settings dump differs in exactly THREE places per pair, 28 pairs each, 84 lines, and all three are `localised_description`**: `bbb-recipe-cost` (the library's `": "` separator becoming `"\n  type: "`) and the two text fields (the library's two new sentences, which arrive as two further parameters after the `default:` line). **`bbb-tech-cost`'s composed description did NOT move**, which is `grep -c 'SET .*bbb-tech-cost'` reading **0** on that log: the cost preset line is `<label>: cost of <technology>` on both sides, and it stays as the sync pass left it.

**SO, of the eight settings-dump places that separate a published release from head, THREE moved in this round and all three are the LIBRARY's composition; the other five did not move at all.** The five that stood still are the two `allowed_values` lists, `bbb-tech-cost`'s composed description, the int setting and the double setting. Nothing this round wrote reaches a settings PROTOTYPE, which is commit 3's own point restated from the other end: a locale file holds strings and the dump holds keys.

### Every gate of the round, at its exit code

Consolidated over the four commits and re-run whole at `4a1b233`. Factorio 2.0.77 (build 84539, mac-arm64, steam), the binary re-asked before every engine run, exit codes read directly except in the two rows that say what stands in for one instead of claiming it.

| gate | exit | |
|---|---|---|
| `gofmt -l guest/go` | prints nothing | inside `make check` at every commit, and recorded standalone by commits 1, 2 and 3 |
| `cd guest/go && go test ./tune/` | 0 | at every commit; with `-count=1` from commit 3, the .cfg being an input the cache tracks |
| `../FkLua/bin/fklua gen-bindings --check` | 0 | the committed bindings unmoved, `every one of them is bound or deferred: go 4865 + 5, rust 4865 + 5` |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)`, the api pin unmoved all round |
| `make check` | 0 | at every commit, and again at `4a1b233` after the round closed (scratchpad `fix1bbb/c5-check.log`), with `test/check-release-arm.sh` inside it since commit 4 |
| `make mod` | 0 | at every commit, and at `4a1b233` inside `make datastage-check`, whose exit code is read directly |
| `make zip` | green | at `4a1b233`; its log carries no `exit=` line, and what stands for one is the `ls -l` row that is the target's LAST recipe line (`Makefile:342`). The figures are above |
| `make datastage-check` | 0 | **EIGHTEEN arms** from commit 1 on (the merge arm is the new one), 35.3 to 37.1 s real over the runs that were timed; green again at `4a1b233` (scratchpad `fix1bbb/c5-datastage.log`), both hashed mod sets at `aa75497e55ce4333`, data-raw `1e1fcf4f56f5ef22` and `e7001bf98d6c6771` |
| `python3 test/check-changelog.py mod-data/changelog.txt 0.3.3` | 0 | `6 version section(s), grammar ok, top section matches manifest 0.3.3`; it also runs inside `make mod` |
| `python3 -m py_compile test/check-datastage.py` | 0 | commits 2 and 3, which are the two that edited it after commit 1 wrote the merge arm |
| `bash -n test/check-release-arm.sh`, `shellcheck test/check-release-arm.sh` | 0 | commit 4; shellcheck at `/opt/homebrew/bin/shellcheck`, no findings |
| `test/check-release-arm.sh` | 0 | the green line, `14 / 21 / 8 / 4 / 2`; five red proofs and a control, all in throwaway clones |
| the release-to-head proof, `prefs-head2.py` | ran to completion | 28 pairs at `4a1b233`. Its exit code is not in the log either; what stands for it is the 28 owned extracts on disk and the `wrote .../head-fix1/manifest.json` its last line names |
| its two comparisons, `compare-prefs.py` | 1, 1 | by construction: the script exits 1 whenever a prototype differs, and it found 10 against the release and 4 against the previous head, both enumerated above |
| agents/docs-style.md's greps over README.md and mod-data/changelog.txt | empty but for one pre-existing hit | the two human-facing files the round touched. The paths grep matches `README.md:141`, a table row naming `CLAUDE.md` and `agents/`, which is unchanged from HEAD and was there before this round |
| the em-dash and en-dash sweep over every added line | empty | `git diff HEAD -U0 \| grep '^+' \| perl -CSD -ne 'print if /[\x{2013}\x{2014}]/'`, at commits 2, 3 and 4 and here; commit 1's own table records the docs-style greps in its place |
| `make test`, fourteen suites | **NOT RUN** | the packaged mod is pinned 2.1 and the binary is 2.0.77, so `test/run.sh` refuses at its engine gate. Unchanged from round one, and it is not this round's to fix |
| the 2.1.16 and 2.1.17 golden rows | **NOT RUN** | same reason; commit 1 extended each `_stale` note with this round's move and the re-capture command, and no 2.1 binary is on this machine |
| the client run | **NOT REACHABLE**, owed since round three | the 13-line tooltip is the thing to look at; the checklist is below |

### What this round leaves open

Consolidated here from the four commits' own paragraphs rather than restated in each, plus what the round's own closing measurements turned up.

**THE 13-LINE TOOLTIP HAS NEVER BEEN RENDERED BY A CLIENT, AND ITS CHARACTER FIGURES ARE RE-MEASURED HERE BECAUSE COMMIT 1's WERE STALE BY TWO COMMITS OF THIS SAME ROUND.** Commit 1 recorded the recipe tooltip as 788 characters with a widest line of 182; that was measured against the .cfg as it stood before commit 2 rewrote `bbb-tech-cost`'s entry and commit 3 rewrote `bbb-recipe-cost`'s, and **line 1 of each tooltip IS the consumer's own entry**, so both totals moved with them. The two sites that carried them, commit 1's paragraph above and its twin in `CLAUDE.md`, are CORRECTED IN PLACE and say so, which is commit 3's precedent for a number this round made stale inside itself; a number from an EARLIER round is annotated instead, which is what commit 2 did to round three's red-proof row. Re-taken at `4a1b233` by rendering the engine's own composed tables with this mod's locale substituted (scratchpad `fix1bbb/c5-tooltip.py`, commit 1's `tt-render.py` pointed at `head-fix1/default/base-settings.json`; the .cfg lengths cross-checked with `grep '^bbb-recipe-cost=' mod-data/locale/en/better-belt-balancer.cfg | perl -CSD -ne 'chomp; s/^[^=]*=//; print length($_)'`):

| setting | lines | total characters | widest line |
|---|--:|--:|--:|
| `bbb-recipe-cost` | **13** | 1,163 | **557**, which is line 1 |
| `bbb-tech-cost` | 4 | 820 | **622**, which is line 1 and the tallest wrap in the mod |
| `better-belt-balancer-recipe-ingredients` | 4 | 708 | 420, line 1 |
| `better-belt-balancer-tech-packs` | 4 | 673 | 409, line 1 |

**The 13 is unchanged and exact** (12 `"\n"` in the composition, line 1 being this mod's own entry), and so is commit 1's parameter table, because a rewritten .cfg entry is one parameter however long it is. What moved is the character count, and the direction is the one that matters for a client: **the twelve composed lines below line 1 of the recipe tooltip run 32 to 66 characters and line 1 alone is 557**, so what a client has to be asked is not whether thirteen lines fit but whether a 557-character wrap plus twelve short lines fits. For the next session with a client: hover `bbb-recipe-cost`'s info icon and check that all six `type:` lines are present and that the last of them, `  type: 1 express-splitter, 2 steel-plate`, is visible and neither clipped nor scrolled off; then hover `bbb-tech-cost`'s, whose 622-character first line is the single tallest wrap this mod can produce.

**TEN OF ELEVEN DROPDOWN LABELS EXCEED THE CLIENT'S MEASURED TRUNCATION AND THIS ROUND DID NOT TOUCH THEM.** The closed widget was seen truncating at roughly 37 characters, one widget at one UI scale, and the same renderer script reports the eleven labels at 35, 38, 43, 45, 46, 50, 52, 52, 54, 58 and 65 characters, only `Splitter: 1 splitter, 2 iron plates` at 35 clearly under. Commit 3 decided KEEP on a measurement and rewrote the argument rather than the labels; what is still open is the widget itself, which no string this repository owns can fix. (**NO LONGER OPEN AS WRITTEN.** Fix round 2's third commit, 2026-09-14, rewrote all nine surviving labels as names and every one of them is under the measured width; the widget still truncates and nothing here changed that, but no label of this mod's reaches it. The eleven counted in this paragraph are the eleven of the day.)

**A PLAYER CAN NAME THE BALANCER PART AS ITS OWN INGREDIENT, AND NOTHING STOPS THEM.** `bbb-recipe-cost = custom` with `better-belt-balancer-recipe-ingredients = 2 bbb-balancer-part` is ACCEPTED (commit 1's paragraph spells that setting with its pre-rename `bbb-` prefix; the emitted name has been the generated one since the sync pass), emits `[{bbb-balancer-part 2}]`, draws the ordinary "takes its ingredients from" line and no `fkrecipes: ERROR:` at all, so the player gets a recipe that can never be crafted with nothing said. Pre-existing, arriving with the text field itself, untouched by the four library commits this round adopts, recoverable by editing the field back, and gated nowhere. Whether a self-referencing ingredient should be refused, warned about, or left alone is a question for the LIBRARY rather than for this consumer. Commit 1 has the long form.

**`bbb-tech-cost` CARRIES NO OVERHAUL-PACK SENTENCE, so the two dropdowns now promise different things about their presets.** The recipe one says its presets are safe and substituted and merged; the research one says only that the cost is copied from that technology. The tier fallback (a missing technology, a trigger technology with no cost, none of the three) is in `README.md` and in neither tooltip. Whoever owns the twin next should decide whether it wants the same clause. Commit 3's.

**THE `Sync mods with save` ROUTE IS NOT PROMISED ANYWHERE, DELIBERATELY.** The client measured it restoring PRESET values out of a save's embedded startup record and never a `custom` one, so it stayed out of the changelog rather than going in unbacked. It is one arm for whoever next has a client. Commit 3's.

**"A COST GATES SOFTLY WHERE A PREREQUISITE GATES HARD" IS REASONING AND NOT A PROBE.** It is the step the choice between the two errors turns on, so commit 2 marked it rather than dropping it, in both places it appears. NOT MEASURED on the engine.

**THE 2.0 BRANCH'S OWN 0.2.3 CHANGELOG ENTRY IS THE ONE PLAYER-FACING DOCUMENT COMMIT 4 COULD NOT FIX**, because fixing it means committing on `release/2.0`, and it stands until the next recut; it is TRUE as written, and the gate is what makes the claim under it checkable. What the MOD PORTAL serves for Factorio 2.0 is NOT MEASURED: `origin/release/2.0` is `f8db95f` (0.2.2) and the local 0.2.3 is unpushed, but the portal is not this git remote. And one surprise is written down for the next recut rather than left to be met in a red gate: `test/datastage-goldens.json` is not on `check-release-arm.sh`'s exclusion list, so a 2.0 golden re-captured ON the branch fails by name, which is the right answer since the row is trunk's emission.

**THREE OF COMMIT 3's REVIEW NOTES WERE TAKEN AND LEFT ALONE, WHICH IS ALSO A DECISION**, and they stay open in the sense that a later round may want them: `localePath` hard-codes `mod-data/locale/en`, correct while only `en` ships; `strings.Cut(word, ":")` is brittle in a direction that fails loudly, which the test's comment now states; and this file's dated-record annotations are asymmetric on purpose, present-tense preamble describing head and a dated section describing its day.

### Two claims in the assessment that this round measured false

[`agents/migration-assessment.md`](agents/migration-assessment.md) is a DATED RECORD with its commit named, so it is not rewritten; this file is where its corrections live, which is the habit round one's superseded table, round three's red-proof row and commit 3's note on finding 6 already follow. Two of its claims are now measurably wrong, and one of the two is not recorded anywhere else.

**FINDING 7's "the settings screen does not" IS FALSE, AND IT WAS FALSE WHEN IT WAS WRITTEN.** The finding says of the Custom arm moving `prerequisites` from `["logistics"]` to `["logistics-3"]`: "The changelog does disclose it in the following sentence; the settings screen does not." `[mod-setting-description] bbb-tech-cost` has carried the disclosure since `6ec3937`, the commit that ADDED the Custom arm to that dropdown, and `6ec3937` is an ancestor of the assessment's own commit `d8dc79e` (`git merge-base --is-ancestor 6ec3937 d8dc79e`). At `d8dc79e` the entry reads, verbatim, "On Custom it sits after Logistics 3, or after the highest of Logistics 2 and Logistics that your mods have, and has no prerequisite if none of the three exists." (`git show d8dc79e:mod-data/locale/en/better-belt-balancer.cfg`). And that sentence is not buried: the engine's own settings dump has `["mod-setting-description.bbb-tech-cost"]` as parameter 1 of the composed table, with every following parameter opening `"\n"`, so the entry IS the first line the tooltip renders, which the render above reproduces. **What finding 7 still fairly owed was BURIAL AND PASSIVITY**, a disclosure a player meets only by hovering an info icon, describing a move they did not ask for. Commit 2 answered that by removing the move altogether rather than by wording it better, and the entry now says the opposite thing: picking Custom does not move the research in a game that has Logistics.

**FINDING 11's READING OF `release/2.0` IS THE OTHER, AND IT IS RECORDED ONCE, UNDER COMMIT 4.** The finding's title, "`release/2.0` is not head's 2.0 arm", is TRUE; what is wrong is reading that as the branch carrying 0.3.2's mod under a new number with the branch as the defect. Measured blob by blob, the branch is trunk 0.3.2's source with trunk's own freeze fix `cb89891` backported file for file plus the documented four-file stamp, so the branch is HONEST and the Makefile's sentence was the defect; the count is 21 today where the finding says 16, and `4b9f597`'s commit message carries a false sentence of its own. Commit 4's section above has the provenance table, the gate and the three corrections; they are not repeated here.

## Re-adoption leg, 2026-09-13: FkRecipes round 1b, and the suite stayed green

FkRecipes moved four commits past `0077f3c`, to `21f5d89`, and this leg is what adopting that costs. Factorio 2.0.77 (build 84539, mac-arm64, steam; `"$FACTORIO_BIN" --version` re-asked before every engine run), both sibling checkouts clean and read-only, FkLua unmoved at `b88965d`. One commit.

### What moved in the library, in this mod's terms

`1f8e363` is the only one of the four that changes behaviour: a recipe whose RESOLVED ingredient list names its own product is emitted as it resolved and logs one line, `fkrecipes: <recipe>: <product> is in the list and is also what this recipe makes, so nothing can craft the first one unless something else produces it`. It is not a refusal and carries no `ERROR` prefix, because nothing is set aside. `642fec4`, `f671ea5` and `21f5d89` are comments, prose, a re-measured NUL boundary and the new `docs/migration.md` section; none of the three touches a line of library code this mod reaches, and a comment-only change does not move a GO guest's packaged module (FkRecipes measured that in both directions, and this mod's own figures below agree).

**THAT LINE CLOSES ONE OF THIS MOD'S OWN ROUND-1 OPEN ITEMS, and it closes it as a signal rather than as a refusal.** "A player can name the balancer part as its own ingredient, and nothing stops them" is the third of the EIGHT bullets under fix round 1's "What this round leaves open", written down there as a question for the LIBRARY rather than for this consumer; the other seven are untouched and are listed again at the end of this section. The library answered it: the recipe still loads and still emits `[{bbb-balancer-part 2}]`, and the load now says out loud that nothing can craft the first one. The reason it is not refused is base's own: `kovarex-enrichment-process` takes 40 `uranium-235` and gives back 41, so a recipe naming its own product is a shape the engine ships.

### Would `docs/migration.md`'s new section have saved the previous adoption its five red tests?

**IT WOULD HAVE PREDICTED ALL FIVE AND THE RED `make datastage-check` BESIDE THEM, AND IT WOULD NOT HAVE SAVED ONE LINE OF THE WORK.** The distinction is the whole of the answer and is worth keeping separate.

The section, "Upgrading a mod that has already adopted the library", leads with "What compiles is not what passes" and lists five behaviours as one matchable line each. Mapped onto this mod's five failures at `ca9aa36`:

| the section's behaviour | this mod's red test |
|---|---|
| a stored text the library cannot use falls back instead of refusing the load | `TestACustomTextTheGameCannotAnswerIsRefused`, `TestAPackTextTheGameCannotAnswerIsRefused` |
| a ladder that lands twice on one name merges | `TestEveryOptionFallsAllTheWayToIronPlate` |
| a composed description carries more than the entry you wrote | `TestEverySettingPrototypeIsTheOneThatShipped`, `TestTheCustomResearchDefaultsAreTheFallbackUnit` |
| nothing is deleted from a text | this plan reaches it nowhere |
| the reserved words match without regard to ASCII case | this plan reaches it nowhere |

Three behaviours produced five tests, two produced none, and no test of this mod's was left unexplained: the section's coverage of what actually broke here is complete. It also calls the engine half by name, twice and in two places: its composed-description row says a test that transcribes a setting prototype moves "and so does any hash you keep over the engine's settings dump; a hash over `data.raw` does not move", and its closing paragraph says to re-take every hash over a settings dump on every engine a row is kept for. That is `aa75497e55ce4333` arriving and both `data_raw_sha256` standing still, exactly as measured. And its closing instruction, treat the suite rather than the compiler as the gate, is the round-1 baseline table's own finding said in advance.

**WHAT IT COULD NOT HAVE SAVED IS THE REWRITING**, and the section does not claim to. Every one of the five tests had to be re-derived against the new contract and three had to be RENAMED, because a name is an assertion too; reading the section first would have changed the order of the work and the surprise, not its size. The one thing it would have saved outright is the diagnosis: five failures in one package with no statement anywhere of what the library had decided, against five lines that name each decision and link the page that states it.

**AND IT WOULD NOT HAVE COVERED THIS LEG AT ALL WERE IT NOT FOR ITS SIXTH LINE.** The five behaviours are `0077f3c` and earlier; the line this leg adopts is `1f8e363`'s, which the section carries as a paragraph under its bulleted list rather than as a sixth bullet; the table above is this file's own. A consumer reading the five and stopping would have missed the only behaviour that moved here. That is a shape rather than a defect in the section, and it is the shape FkRecipes' own notes name: the list is kept by hand and no gate of theirs fails when a line is missing.

### The suite did not go red, and that is the measurement

`make check` is **exit 0** on the untouched tree at `894220b` against FkRecipes `21f5d89`, `go test ./tune/` inside it green over 54 tests, and `make datastage-check` is **exit 0 there too, with no golden moved**, fourteen variant arms and 45.61 s real. Both were taken with the two code files of this commit checked out, which is the state a consumer who had done nothing would be in. So this leg has the shape the section calls the ordinary case: a consumer pays for the changes its own plan reaches, and **this mod's own declarations never reach the new line**. That is checked rather than assumed, in three places:

- **THE DECLARATION.** `guest/go/tune/plan.go` declares one recipe, `LegacyRecipe(part, PartName, ...)`, whose product is `PartName`. Every preset's list comes from `recipeChoices()` over `RecipePlan`, and `guest/go/tune/recipe.go`, which holds every rung of every ladder, names `PartName` ZERO times (`grep -c PartName guest/go/tune/recipe.go`); `FallbackName`, the terminal rung of all of them, is `iron-plate`. `TestEveryRecipeOptionIsTheListTheGateAsserts` pins all six lists entry by entry AND demands no log line at all in a game that has everything, which is the same claim asserted rather than grepped.
- **A DECLARED LADDER OF THIS MOD'S COULD NOT REACH IT EITHER, AND THAT IS A READING OF THE LIBRARY RATHER THAN A MEASUREMENT.** FkRecipes hands the REAL World to `resolveIngredients` on the declared path and the `planItemWorld` overlay only to the text path (`go/customize.go`: "TWO WORLDS, AND THE DIFFERENCE IS DELIBERATE ... the DECLARED ladders are walked against the real World"), so a rung naming `bbb-balancer-part` is simply absent and draws the ordinary drop line. **THE GENERAL CLAIM IS NARROWER THAN THAT AND THE LIBRARY SAYS SO**: round 1b names three routes into the line, an `IngredientOf` handle to the recipe's own result item, a text the player typed, and a ladder landing on the EXISTING item a `ResultNamed` recipe makes, which is somebody else's item rather than one of yours. It is the THIRD that a declared ladder can take, and this mod's recipe produces an item its own plan emits, so that route is closed here and the first is not taken. `selfProductLine`'s own comment in `go/data.go` still says a declared ladder can land on the product, which is true of the `ResultNamed` shape and not of this one. So the only route into the new line here is a text a PLAYER typed.
- **THE ENGINE AGREES ON EVERY CONFIGURATION THIS REPOSITORY DRIVES.** Across the 28 (configuration, mod set) pairs of the proof below, `fkrecipes_lines` is `[]` or the configuration's own expected line on all 28, and the new sentence appears in none of them.

### Two new arms, one per layer, both red-proven

**The host arm, `TestATextNamingTheBalancerPartItselfEmitsItAndSaysSo`** (`guest/go/tune/plandata_test.go`): the everything world with `bbb-recipe-cost = custom` and `2 bbb-balancer-part` in the text. It asserts the emitted list, `[{bbb-balancer-part 2}]`, and the WHOLE log stream in order, two lines: the "takes its ingredients from" line, written as the text is read, and then the self-product line, which the library evaluates on the final list. Milliseconds, no toolchain, no engine.

**The engine arm, `recipe-self-product`** (`test/check-datastage.py`, a fourth entry in `RECIPE_CUSTOM_ARMS`): the same two settings through a real `mod-settings.dat` and a real `--dump-data`. It is the half no host test can make, because what it asserts beside the list and the lines is that **the load still completes**: `run_arm` exits the process on a non-zero engine return, so the arm reaching its assertions at all is the exit code read. The arm block's runner now takes either ONE line the stream must carry or the WHOLE stream as a tuple, and the choice is on the TYPE rather than on the length, so a tuple that ever shrank to one line would keep the stricter rule. The two older arms that name a sentence keep their single string and their tolerance of whatever else the plan said; `recipe-custom-default` names none at all; this one names both of its lines and is compared whole, the way `check_remover` and `tech-ignored` already are.

`recipe-self-product` is also the sharpest ANTI-VACUITY row the customizer block has. A `.dat` that was ignored or malformed leaves every setting at its declared default, and `bbb-balancer-part` is a name no preset and no ladder of this mod can put in an ingredient list at all, so this arm cannot be satisfied by a file the engine did not read.

**Red-proven seven times, each break made on purpose, the message observed, the break reverted.** THE ORDER-SWAP ROW WAS TAKEN TWICE, once before the adversarial review and once after it changed the runner from a filtered ordered comparison to plain equality; the wording recorded is the second. The one-element-tuple row is the review's own and exists only after that change. The `.dat` row fails on the ingredient list rather than on the stream, so the runner change never touched it.

| injected | what fired |
|---|---|
| `plan.go`: the `Custom` arm taken off `IngredientsBy` | the host arm, `the data plan was refused, so the data stage would raise this at load: fkrecipes: the setting bbb-recipe-cost offers custom, and the recipe bbb-balancer-part names no Custom arm for it` |
| `plan.go`: `PartName` renamed to `bbb-balancer-piece` | the host arm, the whole stream against the whole stream, both sentences moved: `got ["fkrecipes: bbb-balancer-piece takes its ingredients from ... 2 bbb-balancer-piece" "fkrecipes: bbb-balancer-piece: bbb-balancer-piece is in the list and is also what this recipe makes, ..."]`. Which is what says both literals are load-bearing rather than derived |
| the host arm's `want` cut to the first line alone, which is the expectation a suite written before `1f8e363` carried | the host arm, `got` two lines against a `want` of one. This is the proof that the line is really in the stream, taken from the direction a consumer meets it |
| the host arm's two `want` lines swapped | the host arm, same two lines in the other order, which is the order claim standing on its own |
| the engine arm's two expected lines swapped | `FAIL recipe-self-product: the library's log lines are [...] and they have to be, whole and in order, [...]`, the whole-stream comparison firing on the engine's own stream. RE-TAKEN after the review changed the runner from a filtered ordered comparison to plain equality, and red again with the wording above |
| the engine arm's tuple cut to its FIRST line alone, still a tuple | `FAIL recipe-self-product: ... and they have to be, whole and in order, ['fkrecipes: bbb-balancer-part takes its ingredients from ...']`, the engine's own second line failing a one-element tuple. This is the proof of the review's own fix: the rule keys on the TYPE, so a tuple that shrinks to one line keeps whole-stream equality instead of silently becoming a membership test |
| the engine arm's `.dat` stripped of the text setting, leaving `{bbb-recipe-cost: custom}` | `FAIL recipe-self-product: the recipe is [('iron-plate', 4), ('iron-gear-wheel', 2), ('transport-belt', 2)] and {'bbb-recipe-cost': 'custom'} should be [('bbb-balancer-part', 2)]`, which is the anti-vacuity half: the stored text is what drives the arm |

**NO CHANGELOG ROW, AND THE DECISION IS RECORDED RATHER THAN SKIPPED.** Round three's rule for this file is that an effect nobody can refuse takes a row. This one is refusable twice over: a player reaches it only by choosing `custom` AND typing the mod's own part into the field, and what arrives is a line in the log rather than anything on screen. 0.3.3's Features row already says the field takes item names and the tooltip already carries the format; a row saying that one particular thing you can type is now reported in a file you would have to open is not what that section is for.

### Sizes, beside the previous ones

`make mod` and `make zip` both **exit 0**, read directly. Against fix round 1's close at `4a1b233`, and every figure by `ls -l`, `wc -l` and `jq -c '.outputs, .jumps.data, .jumps.control' dist/.fklua-mod.report.json` over the build that shipped:

| | `4a1b233` | this leg | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.3.zip` | 836,663 B | **837,958 B** | +1,295 B, +0.15% |
| `fk_data_module.lua` | 4,676,454 B, 118,402 lines | **4,685,165 B, 118,630 lines** | +8,711 B, **+228 lines** |
| `fk_module.lua` | 3,132,893 B, 88,844 lines | the same | unmoved to the byte |
| `control.lua`, `fk_abi.lua`, `fk_api_gen.lua`, the locale file, the changelog, the graphics | | the same | unmoved |
| `dist/bbbdata.wasm` | 1,164,294 B | 1,168,783 B | +4,489 B; not a package member |
| `dist/bbb.wasm` | 1,302,884 B | 1,303,099 B | +215 B; see below |

**THE +228 LINES ARE THE WHOLE ATTRIBUTION AND THEY ARE NOT A TRANSCRIPTION.** FkRecipes measured its own four packaged guests across `1f8e363` and found the same +228 lines per GO guest and +174 per Rust one, on both the guest that declares text settings and the one that declares none, "which is what one unconditional string compiled into every guest looks like". This mod is a Go guest and its packaged module grew by exactly 228 lines, which is that figure met independently on a THIRD Go guest nobody there built, FkRecipes having exactly two of its own (`go/examples/datastage` and `go/examples/notext`). The byte figure is this mod's own, 8,711 B.

**`dist/bbb.wasm` MOVED AND NOTHING IT PRODUCES DID, WHICH IS THE TOOLCHAIN PATH AND NOT THIS CHANGE.** The control guest imports nothing of FkRecipes, so its 215 bytes cannot be the library's; and `fk_module.lua`, which is compiled OUT of that wasm, is byte-identical to `4a1b233`'s. The cause is the one FkRecipes recorded on 2026-09-13 and it is PROBED HERE RATHER THAN INHERITED WHOLE: the host's Go is now go1.27.1, TinyGo 0.41.1 refuses it outright (`requires go version 1.19 through 1.26, got go1.27`, the build's own stderr, `make mod` **exit 2** without the remedy), so every build here runs under `GOTOOLCHAIN=go1.26.6`, which roots under the module cache rather than under the Homebrew cellar. That root really is written into the wasm: `/Users/<home>/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.darwin-arm64` appears **5 times** in `dist/bbb.wasm` and `Cellar/go` appears **zero** times. The arithmetic is the right sign and does not close the gap on its own: that root is 71 bytes where a cellar path for the same version is 38, so five occurrences account for 165 of the 215 and the rest is section-length encoding, not chased further. **WHAT IS MEASURED IS THAT THE LIBRARY IS NOT THE CAUSE**, which the byte-identical `fk_module.lua` says on its own; the path arithmetic is the mechanism, partly measured. **THE REMEDY IS NOT IN THE MAKEFILE AND WAS NOT PUT THERE.** It is an environment fact with a loud failure in front of it, and a Makefile that pinned a Go version would be this repository deciding a toolchain question for whoever builds it next.

**THE RELAY, AND THE ONE FIGURE THAT WENT THE OTHER WAY.** Stations **14**, unmoved. The widest span is the library's `PlanData` at **1,013,809 B**, 155% of the 655,355-byte limit, against `4a1b233`'s 1,018,692 B, and 328,261 B after the relay against 328,253. So the module grew 8,711 B while its widest span SHRANK 4,883, which reads wrong until it is checked against the library's own guest: FkRecipes measured `PlanData` going 1,018,671 to 1,013,790 across `1f8e363`, a drop of 4,881, on a guest nobody here built. Two independent measurements of the same compiler decision, TWO BYTES APART in the delta, which is what says the inversion is the library's and not this mod's. **The number that bounds the module does not move at all**: the widest block is 8,858 B in `fkrecipes.probeIn` with **318,819 B of block room (97%)** on both sides, `PlanData`'s own block 802 B, and `jumps.control` is unchanged in every field (widest span 303,874 B in `main.flushLive`, no relay; widest block 11,033 B in `main.onEvent#wasmexport`, 316,644 B of room).

### The release-to-head proof, re-run over its 28 pairs, and nothing moved

**THE DRIVER AND THE COMPARER WERE REBUILT RATHER THAN RE-RUN**, and that is stated because it changes what the run proves. Round one's `prefs-head2.py` and `compare-prefs.py` are gone from the scratchpad along with the nine 0.3.2 release `mod-settings.dat` files, so the 0.3.2 comparison could not be re-taken at all and is not claimed here. What was re-taken is the HEAD-TO-HEAD half, which is the one this leg is about, and its inputs are the PREVIOUS EXTRACT'S OWN settled read-backs: each configuration's effective values come out of `<config>/base-settled.json`, the file the engine itself wrote at `4a1b233`, so what is driven is the same fourteen configurations rather than a fresh guess at them. Where a release `.dat` left the five head-only settings absent and the engine settled them to their declared defaults, the rebuilt driver writes those same defaults explicitly, which is the same configuration reached by the other door.

**THE REBUILD WAS VALIDATED BEFORE IT WAS TRUSTED.** The two filters that decide what counts as this mod's (prototype names starting `bbb-` plus the legacy `balancer-part`; setting names starting `bbb-` or `better-belt-balancer-`) were run over the PREVIOUS extract's own kept dumps and reproduce that extract's `-owned.json` documents exactly, key sets and values both, 22 owned prototypes on the base arm and 21 on the incumbent. A filter that had drifted would have shown up there rather than as a quietly smaller set of differences.

**THE RESULT IS THE ONLY ONE THIS LEG MAY HAVE.** 28 pairs, 56 halves, **every one IDENTICAL**, comparer exit 0: not one prototype of this mod's and not one settings-dump place moved between `4a1b233` and this head. And the stronger statement is available beside it, because the driver records the same hashes the gate does: the **whole normalised `data-raw` dump and the whole normalised settings dump are byte-identical on all 28 pairs** to fix round 1's close, `diff` over the two logs' hash columns clean, with `mod_settings_sha256` still `aa75497e55ce4333` on every row. A new log line moves neither, which is the section's own point about reading the log stream and not only the prototypes: a consumer whose gates only hash prototypes would see nothing at all here.

### Every gate at its exit code

Factorio 2.0.77 (build 84539, mac-arm64, steam), the binary re-asked its version first, exit codes read directly and never through a pipe. Every build and test row runs under `GOTOOLCHAIN=go1.26.6` for the reason above.

| gate | exit | |
|---|---|---|
| `../FkLua/bin/fklua gen-bindings --check` | 0 | `guest/go/fkapi/fkapi.go is up to date (4865 members bound, 5 deferred)` |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)` |
| `make check` | 0 | on the untouched tree AND with the new arms; `go test ./tune/` green inside it, `gofmt -l guest/go` prints nothing, `test/check-release-arm.sh` green on `14 / 23 / 8 / 4 / 2`, the 23 being the ahead-count BEFORE this commit lands, which is the shape fix round 1's own row has |
| `make mod` | 0 | and **exit 2** without `GOTOOLCHAIN=go1.26.6`, which is the remedy above named by the build's own stderr |
| `make zip` | 0 | 837,958 B |
| `make datastage-check` | 0 | ON THE UNTOUCHED TREE first (eighteen arms, 45.6 s) and then with the new one: **NINETEEN arms**, 41.3 and 47.0 s real over two runs (`/usr/bin/time -p`; the spread is the machine's, the gate does the same work either way): the two hashed mod sets at `aa75497e55ce4333` and data-raw `1e1fcf4f56f5ef22` and `e7001bf98d6c6771`, all unmoved, the stub graph and the six orders on both, **fifteen** variant arms, the speed arm and the merge arm. No golden moved and `test/datastage-goldens.json` is not dirty |
| `python3 -m py_compile test/check-datastage.py` | 0 | |
| the release-to-head proof, 28 pairs | 0 | comparer exit 0, 56 halves IDENTICAL |
| agents/docs-style.md's greps over the files this commit touches | empty | including the dash sweep, `perl -CSD -ne 'print "$ARGV:$.: $_" if /[\x{2013}\x{2014}]/'` |
| `make test`, fourteen suites | **NOT RUN** | observed rather than assumed: **exit 2**, `test/run.sh` refusing at its engine gate with `the built mod targets Factorio 2.1 and the binary is 2.0.` The packaged mod is pinned 2.1 and the binary is 2.0.77, as in every round before |
| the 2.1.16 and 2.1.17 golden rows | **NOT RUN** | no 2.1 binary here; their `_stale` notes stand unchanged, because nothing this leg does moves either dump |
| the client run | **NOT REACHABLE**, owed since round three | unchanged by this leg: the two hovers are still `bbb-recipe-cost`'s six `type:` lines and `bbb-tech-cost`'s 622-character first line |

### What this leg closes and what it leaves open

**CLOSED: "a player can name the balancer part as its own ingredient, and nothing stops them."** The library answers it with a line, this mod pins the line at both layers, and the question that bullet handed to the library has come back answered. What is still true of it is that the recipe LOADS: a player who types it gets an uncraftable recipe and a sentence in the log, not a refusal, and the reason is base's own `kovarex-enrichment-process`. Whether this mod should say something about it in the settings tooltip, where a player would actually meet it, is a question for this mod rather than for the library, and it is new.

**ONE FREE STRENGTHENING NOT TAKEN, AND IT IS THE REVIEW'S.** Under the runner's new type rule, `recipe-custom-default` could carry an empty tuple instead of `None` and would then ASSERT the silence its own comment records as measured, where today it tolerates any stream at all, an `fkrecipes: ERROR: ` line included. It costs one character and it would need its own red proof and its own engine run, so it is left for whoever next opens that block rather than folded into a leg that is meant to be one commit.

**STILL OPEN, UNCHANGED BY THIS LEG**: the client run and its two hovers; the ten of eleven dropdown labels over the client's measured truncation (**closed by fix round 2's third commit, 2026-09-14**: two of the eleven went with the withdrawn value and the nine that remain were rewritten as names, all of them under the width); `bbb-tech-cost` carrying no overhaul-pack sentence where its twin does; the `Sync mods with save` route being unpromised; "a cost gates softly where a prerequisite gates hard" being reasoning rather than a probe; and the 2.0 branch's own 0.2.3 changelog entry, which stands until the next recut.

**AND THE LIBRARY STILL OWES ONE THING THIS LEG IS THE SECOND WITNESS TO.** FkRecipes' own notes close round 1b with it: nothing in that repository's gates sees what a consumer's suite sees, and what answers it is a document kept by hand. This leg is the case where the document was right and cost nothing, because the behaviour that moved is one this plan does not reach; round one was the case where there was no document at all. Neither is the case that would settle it, which is a behavioural change this plan DOES reach arriving with the line missing from the list.

## Fix round 2, 2026-09-14: the library withdraws the Custom value and this mod adopts the text as the switch

[`agents/migration-assessment-2.md`](agents/migration-assessment-2.md) is this mod's SECOND adversarial assessment, run at `1caaf08` on 2026-09-13 against FkRecipes `21f5d89`: eighteen findings, six of them new, and the question it asked that no round before it had is what an update from a PUBLISHED release costs a player. FkRecipes answered the library's half in five commits, `21f5d89` to `221ff9c`, and wrote a scope for the answer first: [`../FkRecipes/agents/threat-model.md`](../FkRecipes/agents/threat-model.md), whose second paragraph is the sentence the whole round turns on, **THE LOG IS NOT WHERE A PLAYER LOOKS**. A disclosure that exists only in a log does not count as one, so a decision that can only be logged is a decision that has to be made again.

**THIS SECTION IS THE ROUND'S FIRST OF FOUR COMMITS AND IT WILL GROW.** This one is the ADOPT commit: the compile break, the two plan-time refusals, the suite, the locale test turned round, the golden and the gate's arms. Three more are owed and each will append a subsection of its own: the locale prose rewrite, the dropdown labels and the changelog, and the remover fixture. What each still owes is at the end.

Every claim below is on Factorio 2.0.77 (build 84539, mac-arm64, steam) or on the host, the two sibling checkouts clean and READ-ONLY, FkRecipes at `221ff9c` and FkLua's checkout at `01d640a`. **FkLua is unmoved as a PIN and moved as a CHECKOUT, and the distinction is FkRecipes' own finding of this round**: `fklua.lock` is not touched, `../FkLua/bin/fklua gen-bindings --check` and `../FkLua/bin/fklua lock --check` both exit 0 against the committed bindings, and the api pin is still 2.1.17.

### What moved in the library, in this mod's terms

**FkRecipes, `21f5d89` to `221ff9c`, five commits.**

| commit | what it is, in this mod's terms |
|---|---|
| `6518971` | **decision A, and the one this mod is about.** The state "the player's own list is in force" moves out of a dropdown VALUE and into the TEXT FIELD. `IngredientsFrom` may sit beside `IngredientsBy` and applies whenever its stored text is not the word `default`; `CostFrom` may sit beside `CostBy` and each of its three fields is a switch of its own. `CustomCost.Seconds` becomes an `IntSettingRef`, `CustomCost.Position` is deleted outright, and a research count or seconds beside a dropdown must declare default 0 and minimum 0. It also folds in decision C: a recipe or technology whose stored value fell back carries a trailing line in its OWN `localised_description` |
| `67b5090` | **decision B**, degrade rather than refuse where the check asks the WORLD anything. A copied research unit's own science packs go through `ToolExists` now, which is finding 13 closed and is one of the two findings this round made about this mod's fixture, below. The packless refusal gains `, and none of <names> is a science pack here`, and a refusal reached after a stored value fell back keeps its FACT and loses the screen it used to name |
| `e4604d4` | **decision D**, every locale key a composed description references goes out as `{"?", {"<section>.<key>"}, "<raw>"}` instead of the bare key table. That is finding 15 closed from the other end, and it is why remedy (f) of this mod's own assessment is now actively WRONG |
| `641cb50` | **decision E**, the word `none` is disclosed beside `default` on an INGREDIENT text setting and deliberately not on a pack one, and the engine's wrap is disclosed in every composed shape that renders a typeable list. Findings 18 and 5 |
| `221ff9c` | the threat model itself, plus the four commits' house trailer. No code |

**WHAT THE WITHDRAWAL IS FOR, AND IT IS FINDING 3 MEASURED RATHER THAN ARGUED.** Factorio RESETS a stored dropdown value the running release's `allowed_values` does not list and PERSISTS the reset with no line in the log, while a setting a release does not declare at all survives verbatim. So a `custom` row is destroyed by one launch of an older release while the typed text beside it lives, which is a field still showing the player's recipe under a dropdown that has quietly gone back to a preset. Keeping the state in the TEXT makes adopting a customizer on a dropdown a mod already ships an IDENTITY: the option list does not move, no stored choice is reset or changes meaning, and a rollback loses nothing. This mod's six presets and three tiers are what they have always been, and the assessment's most severe finding has no subject left here.

### The compile break, which is four declarations in one function

All four are in `tune.Plan()` in `guest/go/tune/plan.go`, which is exactly where FkRecipes' adoption list said they would be (items 1 to 4 of its fifteen). **The compiler output observed on the untouched tree before a line was changed**, which is the whole of what the build had to say:

```
tune/plan.go:337:4: unknown field Custom in struct literal of type fkrecipes.IngredientChoices
tune/plan.go:481:4: unknown field Custom in struct literal of type fkrecipes.CostChoices
tune/plan.go:484:15: cannot use techSeconds (variable of struct type fkrecipes.DoubleSettingRef) as fkrecipes.IntSettingRef value in struct literal
tune/plan.go:485:5: unknown field Position in struct literal of type fkrecipes.CustomCost
```

What each became, read off the diff rather than off the list:

- `IngredientChoices.Custom: recipeIngredients` moves one level out to `RecipeSpec.IngredientsFrom`, beside an `IngredientsBy` whose `Choices` are unchanged.
- `CostChoices.Custom: &fkrecipes.CustomCost{...}` moves one level out to `TechSpec.CostFrom`, beside a `CostBy` that keeps its three choices and its `Fallback`.
- `CustomCost.Position` has no replacement and needs none, which is the deletion this mod gets the most out of. A written cost is no longer a dropdown value with no source technology under it, so there is no arm in which this technology has a cost and nothing to hang off; every value of `bbb-tech-cost` names a source and that source is the prerequisite whatever the three settings say. `TechOptions()` is the menu order and the choice table and nothing else now, and the coupling `TechDefault()` used to carry for the Custom arm's placement is gone with the arm.
- `lib.DoubleSetting(TechSecondsName, 15, ...)` becomes `lib.IntSetting(TechSecondsName, 0, fkrecipes.Between(0, 3600))`. `DoubleSetting` itself stays in the library and stays the right constructor for a crafting time, which this plan does not declare.

**THE CONSTRUCTOR CHANGE IS A PROTOTYPE TYPE CHANGE UNDER A NAME, AND THE LIBRARY MEASURED WHAT THAT COSTS.** `better-belt-balancer-tech-seconds` was a `double-setting` and is an `int-setting`. Measured by FkRecipes headless on 2.0.77: a stored non-integral double under a name redeclared as an int is truncated TOWARD ZERO and then range checked, in range kept and out of range replaced by the DECLARED DEFAULT rather than clamped, with nothing logged in any path and `--dump-data` exit 0 in all of them. **It costs no player anything here**: the four generated settings have never shipped in a public release, so no stored double exists to truncate. It is written down because it is the one break in this round that a consumer who HAD shipped the setting could not have been warned about from inside a guest.

### Two plan-time refusals, which are load failures and arrive after the compiler is happy

**THE CHOICE LIST COVERS THE DROPDOWN'S `allowed_values` EXACTLY.** The library adds no value of its own to a dropdown any more, so nothing is subtracted from the comparison: `RecipeValues()` was `append(RecipeOptions(), RecipeCustom)` and `TechValues()` was `append(TechOptions(), TechCustom)`, seven against six and four against three, and both sides now read one list. The refusal the library raises otherwise is `the recipe bbb-balancer-part offers nothing for the value <v> that the setting bbb-recipe-cost allows` and its technology twin (`../FkRecipes/go/data.go`, the allowed-value walk).

**`RecipeValues`, `TechValues`, `RecipeCustom` and `TechCustom` ARE DELETED OUTRIGHT RATHER THAN REDUCED TO ALIASES**, and that is a decision. The whole reason an Options/Values split existed was the seventh value and the fourth; a `RecipeValues()` that returned `RecipeOptions()` would be a second name for one list, and a reader would have to go and find out which of the two the dropdown is declared with. There is one list per dropdown now, `RecipeOptions` and `TechOptions`, read by the declaration, by `recipeChoices`/`techChoices` and by every test that walks the presets, and `RecipeDefault()`/`TechDefault()` are its head for a reason that is now the only reason: `allowed_values` and `default_value` are separate prototype fields and a default outside the list is a load error.

**A RESEARCH COUNT OR SECONDS BESIDE A `CostBy` DECLARES DEFAULT 0 AND MINIMUM 0.** This mod declared 20 with minimum 1 and 15 with minimum 1, which were `FallbackUnit()`'s own numbers. Both are 0 now, and 0 is what the word `default` looks like on a number: while a field is 0 the tier decides it. The refusal if it is not is the library's, word for word, `fkrecipes: the setting tech-count backs a research count beside a research dropdown, so its declared default and its minimum must both be 0 (0 means the dropdown decides)`. **THE MAXIMA ARE STILL THIS MOD'S AND THE MINIMA ARE NOT**, which is the whole shape of the change: the library refuses a research number with no maximum of at least 1, so the ceiling is the one bound left to choose and 1,000,000 and 3,600 are unchanged.

**AND WHAT STOPS A 0 REACHING THE ENGINE IS THE COST BEHIND THE FIELD AND NOT THE FENCE.** The engine refuses a unit whose count is 0 and one whose time is 0. What this plan emits for a field left at 0 is the number the chosen TIER carries, or, where no source in that ladder carries a unit at all, the number `CostChoices.Fallback` declares, which is `FallbackUnit()`'s 20 and 15 and is what `TestNoLogisticsAtAllIsTheFallbackAndNoPrerequisite` walks. Either way something non-zero answers, so no arm of this plan writes a 0 into a unit, and `plan.go` says so where the declarations are rather than leaving it to be read off the fence.

### The suite: fifty-two tests, thirteen gone and eleven new

`cd guest/go && go test ./tune/ -count=1` is **exit 0** and `go test ./tune/ -count=1 -v | grep -c '^=== RUN   Test'` reads **52**, where the re-adoption leg's was 54. Thirteen `func Test` came out and eleven went in (`git diff -U0 -- guest/go/tune/ | grep '^[-+]func Test'`).

**FOUR OF THE THIRTEEN HAD NOTHING LEFT TO ASSERT AT ALL, and they are two different kinds of nothing.**

- `TestAnEditedTextUnderAPresetIsIgnoredAndTheLogSaysSo` and `TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo` pinned `... is edited, but ... is not on custom, so the text is ignored` and its number twin. Those sentences exist nowhere in the library now and nothing replaces them: every non-default value is live, so there is no state left for such a line to be about. What replaces the tests is the opposite claim -- `TestATextUnderAPresetTakesTheRecipeOverAndNamesWhatItSetAside` and `TestOneFieldMovedTakesTheOtherTwoFromTheTier` -- and the gate's `recipe-text-over-preset` arm is the same sentence at the engine.
- `TestThePositionLadderStepsUpAndThenLetsGo` and `TestTheCustomArmTakesItsPlaceFromTheDefaultTier` were about `CustomCost.Position`, and there is no API behind them any more. **NOTHING REPLACES THEM EITHER, and that is the point rather than a gap**: the property they held was that a written cost lands somewhere legal in the tree, and a cost that cannot exist without a tier cannot land anywhere else. `TestTheCostAndThePrerequisiteComeFromOneSource` and `TestALadderStepsDownAndTakesThePrerequisiteWithIt` were already the tier's own version of it and are untouched.

**THE OTHER NINE ARE THE SAME PROPERTY RE-DERIVED AGAINST THE NEW CONTRACT, AND EIGHT OF THEM ARE RENAMED, BECAUSE A NAME IS AN ASSERTION TOO.** `TestTheCustomValueTakesItsIngredientsFromTheText` became `TestATypedTextTakesTheRecipeOverFromTheDropdown`; `TestTheDefaultWordUnderCustomIsTheVanillaLadder` became `TestTheDefaultWordLetsTheDropdownDecide`, which is a different claim and not a reworded one, because the word now hands the decision back to a dropdown that was never consulted before; `TestAnUnreadableCustomTextTakesTheDeclaredList` became `TestAnUnreadableTextLetsTheDropdownDecide` for the same reason; `TestACustomTextTheGameCannotAnswerFallsBackAndSaysSo` lost one word and kept its four rows; `TestTheCustomResearchCostUntouchedIsTheFallbackUnit` became `TestTheThreeFieldsUntouchedLeaveTheTierDeciding`, which is the sharpest rename of the set, since what an untouched research field means moved from "this mod's own 20 and 15" to "whatever the tier charges"; `TestTheCustomResearchCostIsTheThreeSettings` became `TestTheThreeFieldsOverrideTheTier`; `TestAnUnreadableResearchNumberTakesTheDeclaredDefault` became `TestAnUnreadableResearchNumberLeavesTheTierDeciding`; and `TestTheCustomResearchDefaultsAreTheFallbackUnit` became `TestTheThreeResearchDeclarationsAreWhatThisFileClaims`, because two of the three declarations are no longer that unit's and only the pack list still is.

**`TestEveryCustomDropdownDescriptionNamesTheCustomOption` IS THE ONE DELETION THAT COST SOMETHING, and what it cost is recorded rather than replaced.** Fix round 1 built it and red-proved it four times; it read `[string-mod-setting] bbb-recipe-cost-custom` and `bbb-tech-cost-custom`, took everything before the first colon as the word, and demanded that word in the FIRST SENTENCE of the dropdown's own description. Both entries are deleted in this commit, so the test has nothing to read. The property it guarded -- a dropdown whose text field is invisible until the description mentions it -- has no subject either, because the text field is live from the moment it stops saying `default` and no dropdown value switches it on. What is LOST is that it was the only thing anywhere that read a WORD of a description, and the ten entries that still say Custom are now unguarded by anything at all; that is the next commit's work and it is named at the end of this section.

### Two findings this round made about this mod's own test fixture

Both are about `guest/go/tune/world_test.go` and neither is the library's, which is why they are findings rather than adoption items.

**THE FIXTURE `World` STORED THE PRE-ROUND NUMBERS, SO EVERY "UNTOUCHED GAME" TEST WOULD HAVE BEEN DRIVING AN OVERRIDDEN ONE.** `everythingWorld()`'s `startupRaw` answered `fkrecipes.Num(20)` and `fkrecipes.Num(15)` for the count and the seconds, which were this mod's declared defaults and were therefore a player who had touched nothing. Under the new declarations 0 is what an untouched field holds, so 20 and 15 are a player who DRAGGED BOTH SLIDERS -- and the two numbers they dragged them to look exactly like an untouched field and are not one. A suite left that way would have had every tier test silently overriding both numbers while reading as the default configuration. Both are `Num(0)` now and the comment says why.

**`TestAnUnreachedFallbacksPackIsNeverProbed` WAS SILENTLY WRONG, AND IT IS DECISION B THAT MADE IT SO.** It used to take EVERY tool away, on the reading that a source carrying a unit ends the pack question; `67b5090` filters a COPIED tier unit's own packs through `ToolExists`, so that world loses the tier's pack as well, lands on the declared fallback after all, and reaches a refusal -- which is the opposite of what the test is named for. The world is narrower now and the claim is sharper for it: the tier is `logistics-2`, priced in one `logistic-science-pack`, in a game whose only tool is that pack, so `FallbackUnit()`'s `automation-science-pack` is a name nothing in this world answers for and NOTHING ASKS. What says the source arm ran is the prerequisite (`logistics-2`), the unit (the tier's own, which is not `FallbackUnit()`'s numbers) and the empty log stream. The property the old world was reaching by accident has its own test, `TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused`.

### The locale test is turned round: the advisories are logged and the assertion is a rename

`TestEveryCustomDropdownDescriptionNamesTheCustomOption` is replaced in place by `TestTheLocaleFileRenamesNoTechnologyOfTheGames`, which is the library's item 7 obeyed and this mod's remedy (f) reversed in one test.

**`CheckLocaleAdvisories` IS DELIBERATELY NOT WIRED INTO ANY FAILING TEST, AND THE LIBRARY'S ITEM 7 IS A PROHIBITION RATHER THAN A SUGGESTION.** It returns a note for every out-of-prefix key the plan composes, whether or not the file defines it, it never goes away, and what it asks for is that nothing be done. A suite that called `t.Error` on it would be permanently red over a thing the library says not to fix, which is precisely why FkRecipes took it out of `CheckLocale`'s return. `TestTheLocaleFileSatisfiesThePlan` is where "empty is clean" still means what it says, and it is untouched. The three advisories are `t.Log`ged, one per key, read back with `cd guest/go && go test -v ./tune/ -run TestTheLocaleFileRenamesNoTechnologyOfTheGames`:

```
note: the dropdown setting bbb-tech-cost composes the game's own key technology-name.logistics, which this plan does not own; where the game does not define it the tooltip shows logistics instead, and defining it here would rename it for every mod
note: the dropdown setting bbb-tech-cost composes the game's own key technology-name.logistics-2, which this plan does not own; where the game does not define it the tooltip shows logistics-2 instead, and defining it here would rename it for every mod
note: the dropdown setting bbb-tech-cost composes the game's own key technology-name.logistics-3, which this plan does not own; where the game does not define it the tooltip shows logistics-3 instead, and defining it here would rename it for every mod
```

**WHAT IS ASSERTED IS THE THING THIS MOD CAN GET WRONG**: that the `.cfg` defines no `[technology-name]` entry but its own, `TechName`, read in FILE ORDER out of `fkrecipes.LocaleEntries` rather than out of a map, because Go randomises a map's order per run and a failure naming a different entry each time is a failure nobody can act on. **REMEDY (f) OF THIS MOD'S OWN SECOND ASSESSMENT IS NOT DONE AND IS NOW ACTIVELY WRONG.** It said to define `technology-name.logistics-2` and `-3` here so the tooltip would not lose itself to an undefined key; `e4604d4` closed that from the library's end, so an undefined key degrades to the technology's raw internal name and the tooltip survives whole, and carrying out (f) today would set the displayed name of BASE's technologies for every mod in the game. The test is what keeps it undone.

### The golden, and the property this repository checked before re-capturing

`test/check-datastage.py --capture` on 2.0.77. **Only `mod_settings_sha256` moved, `aa75497e55ce4333` to `b6e583e6163249ba`, the same on both mod sets**; both `data_raw_sha256` are unmoved to the digit (`1e1fcf4f56f5ef22` base, `e7001bf98d6c6771` incumbent) and so are both `prototype_list_checksum` values. Since the data-raw hash is taken over the WHOLE normalised dump, its being unmoved IS those dumps compared byte for byte, which is what says a withdrawal of a dropdown value and a rewrite of five composed descriptions reach the SETTINGS stage and nothing else while every setting is at its default. That is the property FkRecipes' item 13 predicted and said to treat a move in as a finding rather than a re-capture.

**FIVE THINGS MOVED IN THE SETTINGS DUMP AND ALL FIVE ARE READ OUT OF THE ENGINE'S OWN `mod-settings-dump.json`**, not argued from the diff:

1. `bbb-recipe-cost`'s `allowed_values` is **six** where it was seven and `bbb-tech-cost`'s is **three** where it was four, both having lost `custom`.
2. `better-belt-balancer-tech-seconds` is an `int-setting` where it was a `double-setting`, under the same name, and BOTH research numbers read `default_value` 0 and `minimum_value` 0 where they read 20 with minimum 1 and 15 with minimum 1.
3. Both dropdowns' composed description gains a closing switch line, `\nThe setting below applies instead while it does not say default.`, and the INGREDIENT one gains the wrap line before it; both text settings gain the wrap line and a switch line of their own, and the ingredient one's format line gains the `none` clause (`The word none empties the list, so the recipe costs nothing to craft.`), which is finding 18 closed and remedy (g) with it.
4. The two research numbers carry a composed `localised_description` where they carried NONE AT ALL: `\nA whole number from 0 to 1000000. While it is 0 the option chosen above decides.` and its 3600 twin. That is half of finding 9 closed -- the two numeric ranges are now in words where a player looks -- and the half that is left is this mod's `[mod-setting-description]` entry saying what the number is FOR, which the library requires and this file already ships.
5. **Every locale key any composed description references is now `{"?", {"<section>.<key>"}, "<raw>"}` where it was the bare `{"<section>.<key>"}`**, so the head line of all five compositions, the nine preset labels and the three `technology-name` references each gained two elements. That is `e4604d4`, and it closes finding 15: the research-cost setting had NO TOOLTIP AT ALL on a stock install, because no locale file anywhere defines `technology-name.logistics-2`, and an undefined key inside a `localised_description` costs the whole thing on the client while the dump goes on carrying the string. No gate in either repository can see that defect, which is why the fix had to come from the shape rather than from a check.

The 2.1.16 and 2.1.17 rows are NOT re-captured and keep their `_stale` keys, each gaining the same paragraph naming the move and the re-capture command, which is the sixth time that arm's settings dump has moved unrecorded.

### The composed tooltips, re-measured on the engine, and the ceilings corrected

Read out of the engine's own settings dump with this mod's own `.cfg` resolved into it, the `{"?", ...}` groups rendered the way the engine renders them (the first alternative the file defines, else the raw fallback). Six settings carry a `localised_description` now where four did.

| setting | parameters | depth | lines | longest line | characters |
|---|--:|--:|--:|--:|--:|
| `bbb-recipe-cost` | 9 of 20 | 4 of 20 | 15 | **557** | 1,323 |
| `bbb-tech-cost` | 5 of 20 | 4 of 20 | 5 | **622** | 885 |
| `better-belt-balancer-recipe-ingredients` | 6 of 20 | 3 of 20 | 6 | 420 | 967 |
| `better-belt-balancer-tech-packs` | 6 of 20 | 3 of 20 | 6 | 409 | 862 |
| `better-belt-balancer-tech-count` | 2 of 20 | 3 of 20 | 2 | 105 | 186 |
| `better-belt-balancer-tech-seconds` | 2 of 20 | 3 of 20 | 2 | 117 | 195 |

**THE CEILINGS IN THAT TABLE ARE CORRECTED AND FIX ROUND 1'S WERE WRONG IN ONE HALF.** That round wrote "of 19" for the nesting and read the parameter ceiling as a per-STRING budget. FkRecipes re-measured both on 2.0.77 twice independently this cycle, one `--dump-data` per row: **20 PARAMETERS PER TABLE**, not per string, the 21st refusing the load with `Too many parameters for localised string: 21 > 20 (limit).`; **20 LEVELS OF NESTING DEPTH**, the 21st refusing with `Too deep recursion for localised string: 21 > 20 (limit).`; and no global table budget at all, a description holding 421 tables at depth 3 loading clean. What changes in the reading is the headroom rather than the numbers: `e4604d4` spends one level of DEPTH per wrapped key and no parameters, because a wrapper occupies the slot the bare key table occupied, and depth is the budget with 20 levels in it. **NOTHING IS NEAR EITHER CEILING**: the worst is `bbb-recipe-cost` at 9 of 20 and 4 of 20, and growth costs one parameter per preset.

**THE 557 AND THE 622 ARE THIS MOD'S OWN LOCALE ENTRIES AND A LATER COMMIT CUTS THEM.** Line 1 of each dropdown tooltip IS the consumer's `[mod-setting-description]`, and both are the entries that still offer a `Custom` option the dropdown no longer has, so the number and the sentence move together in the same commit. **NOT MEASURED: whether any of the six renders whole.** No client was run and none was started; the parameters, the depth, the lines and the characters are measured and the rendering is not, which is the client run still owed since round three and now one line larger on four of the six.

The command, kept because the arithmetic is not obvious: the composed description is read out of `mod-settings-dump.json`, `{"?", key, raw}` groups are resolved against `mod-data/locale/en/better-belt-balancer.cfg`, `""`-keyed tables are concatenated, and parameters are counted per table rather than per string.

### The gate's arms, and the three tables that were named for a value

`test/check-datastage.py`'s three customizer tables are renamed with the state they drive, because each was named for a dropdown value that does not exist: `RECIPE_CUSTOM_ARMS` is `RECIPE_TEXT_ARMS`, `TECH_CUSTOM_ARMS` is `TECH_COST_ARMS`, and `TECH_IGNORED_ARMS` is `TECH_TIER_ARMS`. **EIGHT CUSTOMIZER ARMS WHERE THERE WERE SEVEN**, and five of the seven names are gone:

| arm | what it drives |
|---|---|
| `recipe-default-word` | the reserved word stored under `cheap`, so the expected list is cheap's and the library says NOTHING. The preset is `cheap` rather than `vanilla` on purpose: a planner that ignored the dropdown while the text said the word would come out with the vanilla list, which is byte for byte what a `.dat` that never arrived produces |
| `recipe-text-alone` | a recipe typed with the dropdown untouched, `3 iron-plate, 1 splitter`, which is an ORDER and a set of amounts no preset of this mod can produce |
| `recipe-text-over-preset` | the round in one arm: `1 iron-plate` under `cheap`, the same item at a different amount, so a planner that still preferred the preset fails on the LIST and not only on the line |
| `recipe-self-product` | the re-adoption leg's arm, unchanged in what it drives and carrying its two lines |
| `tech-cost-whole` | all three fields written under `logistics-2`, 50 units at 20 seconds of a list neither the tier's nor this mod's declared default |
| `tech-tier-word` | the three ways of saying "the row above decides" all at once, the word and two zeros STORED, and no line at all |
| `tech-count-alone` | one field moved, so the expected unit is the tier's with 50 written into `count` alone |
| `tech-packs-alone` | the count arm turned round, the typed list over the tier's own count and time |

**EVERY ONE OF THE EIGHT COMPARES THE WHOLE `fkrecipes:` STREAM, IN ORDER.** The old table carried two rules, one line the stream must CARRY against the whole stream compared in order, because two of its arms could not state what else the plan would say; every arm here can, so the weaker rule is gone with the state that needed it and an extra line is a failure. `OUR_SETTINGS_ORDERS` does NOT move: `a`, `aab`, `b`, `bad`, `bae`, `baf`, six pairs, exactly as the library predicted, because withdrawing a value moves neither a name nor an order.

**THE GATE IS TWENTY ARMS**, counted from the script rather than by eye: the two hashed mod sets, sixteen variant arms (`len(RECIPE_VARIANTS) + len(TECH_VARIANTS) + len(RECIPE_TEXT_ARMS) + len(TECH_COST_ARMS) + len(TECH_TIER_ARMS) + 1` = 5 + 2 + 4 + 1 + 3 + 1), the speed arm and the merge arm.

### Red proofs

Each break made on purpose in this mod's own code, its fixture or its locale file, the failure observed, the break reverted. FkRecipes and FkLua were read-only for the whole commit, so no proof below is injected in a library.

| injected | what fired |
|---|---|
| `IngredientsBy` dropped from the `RecipeSpec` while `IngredientsFrom` stays | **TWELVE tests**, including all four rows of `TestATextTheGameCannotAnswerFallsBackAndSaysSo`: `the text "3 tungsten-plate" is made of [{iron-plate 4} {iron-gear-wheel 2} {transport-belt 2}] and the gate asserts [{iron-plate 2} {transport-belt 1}]`. Which is what says the two fields are both load-bearing and that the dropdown is still consulted when the text stands down |
| `IngredientsFrom` removed, and separately `CostFrom` | **38 each**, on the library's own binding rule: `fkrecipes: the setting <name> is declared and nothing reads it; a text setting must be bound to one recipe or technology` |
| the `Count` and `Seconds` handles swapped in the `CustomCost` literal | exactly `TestTheThreeFieldsOverrideTheTier` and `TestOneFieldMovedTakesTheOtherTwoFromTheTier`, which is the narrowest pair in the suite and is what says the two settings are told apart by their slot and not by their bounds |
| the fixture `World`'s stored `20` and `15` put back for the two research numbers | **21 tests**, which is the finding above measured: a fixture holding this mod's old declared defaults is a player who moved both sliders, and 21 tests would have been reading an overridden game as an untouched one |
| `IntSetting(TechCountName, 20, Between(1, 1000000))` put back | **three**, on the library's designed refusal: `fkrecipes: the setting tech-count backs a research count beside a research dropdown, so its declared default and its minimum must both be 0 (0 means the dropdown decides)` |
| `[technology-name] logistics-2=Belt logistics II` added to `mod-data/locale/en/better-belt-balancer.cfg` | the new `TestTheLocaleFileRenamesNoTechnologyOfTheGames`, which is the only thing in either repository that could have seen it: the advisories do not fail, the checker reads keys this plan owns, and `--dump-data` never reads a locale file at all |
| `checkFallbackNote`'s `recipe` argument flipped, at either call site | exactly one test each, which is decision C's scoping rule held from this side: a recipe's INGREDIENT text earns the assembling-machine sentence and a pack text does not |
| `textFieldLines`'s `ingredients` argument flipped, at either call site | exactly one test each, the `none` clause landing on the pack field or going missing from the ingredient one |

### What is owed upstream, with its evidence

**THE PACKLESS REFUSAL NAMES ONE RUNG TWICE WHEN THE TIER'S COPIED PACK AND THE DECLARED `Fallback`'s ONE RUNG ARE THE SAME NAME.** `67b5090` gave that sentence a rung list, and this mod's declared fallback pack is `automation-science-pack` with no ladder under it, which is also the pack `logistics` is priced in, so the two probes are the same name asked about twice and the sentence reads `... research takes at least one, and none of automation-science-pack, automation-science-pack is a science pack here`. Two tests here reach it and pin it as it is, `TestAFallbackThisModsOwnPackListCannotPayForStillRefuses` and `TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused`; `TestAReachedFallbackWithNoPackInTheGameIsRefused` reaches the single-name form, which is the same sentence with only the declared rung tried. It is cosmetic, it is in a message a player reads in an error dialog, and the fix is the library's: de-duplicate the rung list in walk order. **RECORDED HERE AND NOT FIXED HERE**, because the sentence belongs to FkRecipes and this repository pins it rather than builds it.

### Gates, at their exit codes

Exit codes read directly and never through a pipe. Every wasm build runs under `GOTOOLCHAIN=go1.26.6`, which is the re-adoption leg's own environment fact unchanged: the host's Go is go1.27.1, `tinygo version` reads `0.41.1 darwin/arm64 (using go version go1.27.1 ...)`, and TinyGo 0.41.1 refuses that toolchain outright with `requires go version 1.19 through 1.26, got go1.27`. The remedy is not in the Makefile and was not put there, for the reason that leg gives: it fails loudly with its own remedy, and a pinned Go version would be this repository deciding a toolchain question for whoever builds it next.

| gate | exit | |
|---|---|---|
| `cd guest/go && go test ./tune/ -count=1` | 0 | 52 `func Test` |
| `../FkLua/bin/fklua gen-bindings --check` | 0 | `guest/go/fkapi/fkapi.go is up to date (4865 members bound, 5 deferred)` |
| `../FkLua/bin/fklua lock --check` | 0 | `fklua.lock is up to date (api 2.1.17)`; neither the lock nor the bindings is dirty |
| `make check` | 0 | gofmt included; `test/check-release-arm.sh` green inside it on `14 / 25 / 8 / 4 / 2`, the 25 being the ahead-count BEFORE this commit lands, which is the shape the re-adoption leg's own row has |
| `GOTOOLCHAIN=go1.26.6 make mod` | 0 | |
| `make datastage-check` | 0 | after the re-capture; twenty arms, the two hashed mod sets at `b6e583e6163249ba` for the settings dump and `1e1fcf4f56f5ef22` and `e7001bf98d6c6771` for data-raw, both unmoved |
| `make test`, fourteen suites | **NOT RUN** | the packaged mod is pinned 2.1 and the binary is 2.0.77, as in every round before |
| the 2.1.16 and 2.1.17 golden rows | **NOT RUN** | no 2.1 binary here; their `_stale` notes stand, one paragraph longer |
| the client run | **NOT REACHABLE**, owed since round three | and one line larger: the two hovers are still `bbb-recipe-cost`'s and `bbb-tech-cost`'s first lines, now 557 and 622 characters, and four of the six tooltips gained a line this commit |
| the release-to-head proof, 28 pairs | **NOT TAKEN AT THIS COMMIT** | it belongs to the round's close, where the four commits are in and the settings dump has stopped moving |
| package sizes and the relay's jump figures | **NOT TAKEN AT THIS COMMIT** | same reason; the round's close is where a figure is comparable against the re-adoption leg's |

**WHICH OF THOSE ROWS WERE RE-TAKEN WHEN THIS SECTION WAS WRITTEN, said rather than left to be assumed.** `go test ./tune/`, `make check`, `gen-bindings --check` and `lock --check` were re-run against the working tree while these notes were being written and agree with the figures above, including the 52. `make mod` and `make datastage-check` are the commit's own runs and were not re-taken here, because both write into `dist/` and the gate stages mods; what stands in their place as evidence that the golden claim is real is the COMMITTED file, whose 2.0.77 row reads `b6e583e6163249ba` for the settings dump on both mod sets and `1e1fcf4f56f5ef22` and `e7001bf98d6c6771` for data-raw, unmoved, and whose `_note` carries the five moves read out of the engine's own dump.

### What this commit leaves the round, and what each remaining commit owes

**THE LOCALE PROSE, WHICH IS THE ONE THING NO GATE IN EITHER REPOSITORY CAN SEE.** `--dump-data` does not read a locale file, the checker reads KEYS and never the strings beside them, and the one test that read a word of a description went with the value it was about. **TEN ENTRIES IN `mod-data/locale/en/better-belt-balancer.cfg` GO ON INSTRUCTING THE PLAYER TO PICK AN OPTION THE DROPDOWN NO LONGER OFFERS, and nothing anywhere is red** (`grep -n Custom mod-data/locale/en/better-belt-balancer.cfg`): SIX DESCRIPTIONS -- `bbb-recipe-cost` offering "or Custom to write the recipe yourself in the setting below" and later "Every option but Custom is safe in an overhaul pack", `bbb-tech-cost` offering "or Custom to write the cost yourself in the three settings below", and the four generated fields each scoping themselves to "while the recipe/research setting above is set to Custom" -- AND FOUR NAMES, `Custom balancer part recipe` and three `Custom balancer research: ...`. **THIS IS THE MISLED SHAPE THE WHOLE ROUND EXISTS TO CLOSE ARRIVING THROUGH THE ROUND'S OWN MIGRATION**, and the second assessment moved finding 6 to CLEAN partly on the strength of one of those very sentences. The rule is one rule applied ten times: the text field applies whenever it does not say `default`, and the dropdown supplies the rest.

**STILL OWED TO THE ROUND, one commit each.**

- *The locale prose.* The ten entries above, plus the `bbb-tech-cost` entry's placement clause, which described a Custom research hanging off a ladder of its own and describes nothing now.
- *The dropdown labels and the changelog.* The 557 and the 622 are the two consumer entries and are the tallest wraps this mod can produce; the ten of eleven dropdown labels over the client's measured ~37-character truncation are unchanged and unmeasured since fix round 1. And remedy (h) of the second assessment is unwritten: the changelog discloses that changing the recipe empties an assembler and does not disclose the NEW way to change it that this round creates, which is that a typo in the text field falls back instead of refusing, so the recipe can move without the player touching the dropdown.
- *The remover fixture.* Remedy (i): `check_remover`'s fixture breaks Space Age before this mod can be judged on a stock install, so the arm it guards is only honest with the expansions disabled, and the fixture should sweep at `data-final-fixes` the way the synthetic one built for the assessment does.
- *The round's close.* The package sizes, the relay's jump figures, the release-to-head proof over its 28 pairs, and every gate consolidated at its exit code.

**TWO OF THOSE FOUR HAVE SINCE LANDED AND THE LEDGER IS KEPT AS THE COMMIT WROTE IT**, because a per-commit ledger rewritten to what is left would have the adopt commit claiming knowledge it did not have. The locale prose is the second commit, below; the labels and the changelog are the third, below that, and remedy (h) is answered there. What is left of this list is the remover fixture and the round's close.

**AND ONE THING THIS COMMIT CLOSES OUTRIGHT.** Remedy (f) is struck rather than done, and this file says so in the one place a future reader would go looking: `TestTheLocaleFileRenamesNoTechnologyOfTheGames` is what stops somebody carrying it out.

### The round's second commit: ten entries rewritten under one rule, seven comment blocks with them, and one test so it cannot go false silently

**TWO FILES, AND ONE OF THEM HAD NEVER BEEN READ BY ANYTHING.** `mod-data/locale/en/better-belt-balancer.cfg` and `guest/go/tune/locale_test.go`, `git diff --stat` reading `367 insertions(+), 88 deletions(-)`. This is FkRecipes' adoption item 9, the one that commit's own subsection above ends by naming, and it is the item that exists because the library's adversarial review of decision E found it as must-fix 2: the adoption list would have had the consumer ship six false tooltips and stop one sentence short of saying so, because item 8's "no entry becomes an ORPHAN" is true about KEYS and reads as a statement about the file.

**NOTHING WAS RED ANYWHERE, AND THE THREE REASONS ARE THREE DIFFERENT BLINDNESSES.** FkRecipes' locale checker reads KEYS -- every declared value has an entry, no entry names a value the plan does not declare -- and never the strings beside them. `--dump-data` carries the locale key and not the string, so `make datastage-check` cannot see a syllable of the file. And the one test that read a WORD of a description, `TestEveryCustomDropdownDescriptionNamesTheCustomOption`, was deleted in the adopt commit along with the `[string-mod-setting]` entries it read the word out of. Every key in the file stayed valid across the withdrawal; ten strings went false under them.

#### The ten entries, before and after, in full

Taken out of `git diff` rather than retyped. Line numbers are HEAD's for the OLD and the working tree's for the NEW, and they are the ten lines FkRecipes' item 9 names.

**FOUR NAMES, each of which named the field for an option instead of for what it holds.** A name is the half a player reads without hovering anything, and it is where four of the ten hid.

```
OLD :130  better-belt-balancer-recipe-ingredients=Custom balancer part recipe
NEW :133  better-belt-balancer-recipe-ingredients=Balancer part ingredients
```

False twice over: `Custom` is no value of `bbb-recipe-cost`, and `recipe` made the row a second copy of the dropdown above it rather than the list that goes into it. The new name says what the field holds.

```
OLD :143  better-belt-balancer-tech-packs=Custom balancer research: science packs
NEW :147  better-belt-balancer-tech-packs=Balancer research: science packs
```

```
OLD :144  better-belt-balancer-tech-count=Custom balancer research: units
NEW :148  better-belt-balancer-tech-count=Balancer research: units
```

```
OLD :145  better-belt-balancer-tech-seconds=Custom balancer research: seconds per unit
NEW :149  better-belt-balancer-tech-seconds=Balancer research: seconds per unit
```

All three carried the same false word in the same first position, and all three lose it and nothing else. The three-row shape is the engine's rather than a choice: a text field takes the packs, and a count and a time are a number each.

**SIX DESCRIPTIONS.**

```
OLD :197  bbb-recipe-cost=What a balancer part costs to craft, or Custom to write the recipe yourself in the setting below. The default is what this mod has always used. Every option but Custom is safe in an overhaul pack: where one names something your mods do not have, the nearest thing they do have is used instead, and where that leaves one item named twice the two amounts are added, so what you craft can be a shorter list than the one shown here. A recipe you write yourself is the opposite on purpose, and it is taken as written: nothing is substituted for a name you typed.
NEW :203  bbb-recipe-cost=What a balancer part costs to craft. The default is what this mod has always used. Every option is safe in an overhaul pack: where one names something your mods do not have, the nearest thing they do have is used instead, and where that leaves one item named twice the two amounts are added, so what you craft can be a shorter list than the one shown. A list you write yourself is taken as written: nothing is substituted for a name you typed.
```

FALSE IN TWO PLACES AND THE SECOND ONE IS THE ONE THE SECOND ASSESSMENT GRADED. The offer, "or Custom to write the recipe yourself in the setting below", sends the player to a row of the menu that is not there. And "Every option but Custom is safe in an overhaul pack" was the scoping that moved finding 6 from MISLED to CLEAN one round earlier: with the seventh value gone, six of six options substitute, the exception has no subject, and the promise is general again. The contrast a player choosing between the two rows still needs is kept in the last sentence, now about a LIST rather than about an option.

```
OLD :218  better-belt-balancer-recipe-ingredients=What a balancer part is made of while the recipe setting above is set to Custom. Write the amount, then the item's internal name, and separate ingredients with commas: 2 iron-plate, 1 splitter. Internal names are the ones the game uses in its own data and in rich text, not the names shown on screen. The word default means this mod's own recipe, and the tooltip on the setting above lists every preset written this way.
NEW :233  better-belt-balancer-recipe-ingredients=What a balancer part is made of. Write the amount, then the item's internal name, and separate ingredients with commas: 2 iron-plate, 1 splitter. Internal names are the ones the game uses in its own data and in rich text, not the names shown on screen. The tooltip on the setting above lists every preset written this way, which is where to copy one from.
```

The opening scoped the field to a state that cannot be reached. The closing half-sentence about `default` went with it for a different reason, below.

```
OLD :219  bbb-tech-cost=The balancer is always unlocked by its own research; this picks which logistics tier that research is priced like, or Custom to write the cost yourself in the three settings below. For a tier the cost is copied from that technology, so it follows whatever your mods charge for it, and the balancer's research moves to sit just after it in the technology tree. On Custom it sits after Logistics, the same place the setting's default puts it, so picking Custom does not move it; without Logistics it sits after the lowest of Logistics 2 and Logistics 3 your mods have, and it has no prerequisite if none of the three exists.
NEW :265  bbb-tech-cost=Which logistics tier the balancer's own research is priced like. The cost is copied from that technology, so it follows whatever your mods charge, and the research sits just after it in the technology tree. A tier your mods leave unpriced steps down to the next belt tier, and where that runs out the research takes 20 units of automation science at 15 seconds each with no prerequisite. The three settings below override that tier one field at a time: the science packs while that field does not say default, the unit count and the seconds while either is not 0.
```

FALSE IN THE OFFER AND FALSE IN A WHOLE PLACEMENT CLAUSE. "or Custom to write the cost yourself in the three settings below" is the offer; the entire third sentence described where a Custom research hangs, and `CustomCost.Position` is deleted outright, so it described nothing. What replaces it is the behaviour that is actually there: `TechLadder` in `guest/go/tune/tech.go` steps `logistics-3` down through 2 to 1 and `logistics-2` down to 1, and `FallbackUnit()` in `guest/go/tune/plan.go` is 20 automation science at 15 seconds with no prerequisite where no rung carries a unit. That is the same 20 and 15 this mod used to declare as the two numeric defaults, arriving now from the other side of the change.

```
OLD :236  better-belt-balancer-tech-packs=What the balancer research is paid in while the research setting above is set to Custom. Write the amount, then the pack's internal name, and separate packs with commas: 1 automation-science-pack, 1 logistic-science-pack. Only science packs are accepted, and internal names are the ones the game uses in its own data and in rich text, not the names shown on screen. The word default means this mod's own list.
NEW :284  better-belt-balancer-tech-packs=What the balancer research is paid in. Write the amount, then the pack's internal name, and separate packs with commas: 1 automation-science-pack, 1 logistic-science-pack. Only science packs are accepted, and internal names are the ones the game uses in its own data and in rich text, not the names shown on screen.
```

```
OLD :241  better-belt-balancer-tech-count=How many units of science the balancer research costs, while the research setting above is set to Custom.
NEW :294  better-belt-balancer-tech-count=How many units of science the balancer research costs.
```

```
OLD :242  better-belt-balancer-tech-seconds=How many seconds one unit of the balancer research takes in a lab, while the research setting above is set to Custom.
NEW :295  better-belt-balancer-tech-seconds=How many seconds one unit of the balancer research takes in a lab.
```

The two numbers were one sentence each and the second half of each sentence was the false one. The first assessment's own finding 4 quotes `How many units of science the balancer research costs, while the research setting above is set to Custom.` as the thing that states the RULE and cannot state the STATE; the rule it stated is now not a rule at all.

#### One rule, applied ten times

**THE TEXT FIELD APPLIES WHENEVER IT DOES NOT SAY `default`, AND THE DROPDOWN SUPPLIES THE REST.** That is FkRecipes' item 9 in its own words and it is the whole of the editorial policy here. Three shapes come out of it and each of the ten is one of the three: a dropdown's description drops the option from its list; a field scoped "while the setting above is set to Custom" is scoped to nothing instead and simply says what it is; and a field NAMED for the option is named for what it holds.

**FOR THE RESEARCH THE RULE IS PER FIELD, AND TWO OF THE THREE FIELDS SWITCH ON 0 RATHER THAN ON A WORD.** `better-belt-balancer-tech-packs` switches on the word `default`; `-tech-count` and `-tech-seconds` switch on 0, which is what the adopt commit's `IntSetting(TechCountName, 0, Between(0, 1000000))` and its 3600 twin mean. `bbb-tech-cost`'s new last sentence is the only entry a player reads that names all three, and it names them in exactly that split.

#### Two entries got shorter by dropping a sentence the library composes, and one of the two omissions was wrong

**THE RECIPE FIELD'S `default` SENTENCE IS THE LIBRARY'S NOW AND WAS RIGHT TO DROP.** `The word default means this mod's own recipe` was removed from `better-belt-balancer-recipe-ingredients` and `The word default means this mod's own list.` from `-tech-packs`, because FkRecipes composes `textSwitchLine` under each of them (`go/customize.go:724`): `While this says default the option chosen above applies; anything else applies instead of it.` Saying it twice in one tooltip is the thing the whole comment block above that entry exists to refuse.

**AND `bbb-tech-cost`'s THREE FIELDS WERE DROPPED ON A THEORY THAT DOES NOT HOLD, WHICH AN ADVERSARIAL REVIEW CAUGHT AND NO GATE COULD HAVE.** The first rewrite of that entry said nothing about the three settings below it, on the theory that the library says so. What the library actually composes onto that dropdown is ONE sentence, `dropdownSwitchLine` at `go/customize.go:733`, reaching the cost dropdown through `go/customize.go:680`:

```
The setting below applies instead while it does not say default.
```

It is SINGULAR ("the setting"), it is built from `CustomCost.Packs` and from nothing else (`dropdownSwitchLine(l.relativeOrder(i, c.Packs.index-1))`, where `c` is `t.spec.CostFrom`), and its switch is the word `default`. Three rows sit below that dropdown and two of them switch on 0, so the composed line cannot be read as covering them, and an entry leaving the three to it would have told a player about one field and left the other two invisible. **THAT IS THE MISLED SHAPE ARRIVING A SECOND TIME INSIDE THE COMMIT THAT EXISTS TO CLOSE IT**, and it is worth saying plainly: it was caught by a read of the library's source, not by anything that can go red. The entry enumerates all three itself now, and the overlap on the pack text is deliberate rather than a repetition, because the composed line names no field and the whole difficulty is that "the setting below" is three settings here.

#### The two lengths, and they are the two longest strings this mod puts on a settings screen

```
git show HEAD:mod-data/locale/en/better-belt-balancer.cfg | awk 'NR==197||NR==219{sub(/^[^=]*=/,""); print NR": "length}'
awk 'NR==203||NR==265{sub(/^[^=]*=/,""); print NR": "length}' mod-data/locale/en/better-belt-balancer.cfg
```

| entry | before | after |
|---|---|---|
| `bbb-recipe-cost` | 557 | **443** |
| `bbb-tech-cost` | 622 | **563** |

The 557 is the one the second assessment measured wrapping to ten visual lines inside a 24-line tooltip that fit on screen, and the adopt commit's own tooltip table names both as the two hovers the owed client run has to take. **THE 687 IS NOT REPRODUCIBLE FROM THIS TREE AND IS RECORDED AS REPORTED RATHER THAN MEASURED**: the tech entry is said to have gone to 687 characters when the three fields went back in after the review and to have been re-cut to the 563 above, and that intermediate draft exists in no commit, so no command here produces it. What the tree can say is that the final entry is 59 characters shorter than the one it replaces while carrying a clause the old one did not have.

#### The new test, and it reads one word per entry where the entries carry paragraphs

`TestNoSettingDescriptionNamesAnOptionNoDropdownOffers` in `guest/go/tune/locale_test.go` is the fifth of this mod's own locale tests, and the header sentence of that block moves from "as four tests of this mod's own" to "as five".

**WHAT IT PINS.** For each dropdown the plan declares it builds a vocabulary out of the plan rather than out of a literal: that dropdown's own allowed values, split on `-` so `belt-express` contributes `belt` and `express`, plus every capitalised word of each value's `[string-mod-setting]` label. To that it adds this mod's own DISPLAY vocabulary, read out of the same file's `[entity-name]`, `[item-name]`, `[recipe-name]` and `[technology-name]` sections. Then every capitalised word of the `[mod-setting-name]` and `[mod-setting-description]` entries of that dropdown AND of the settings bound beside it has to come from one of the two. `Logistics` and `Default` pass because they are option labels (`bbb-tech-cost-logistics=Default: Logistics, alongside transport belts`, which is the label of the day and was shortened by the third commit the day after; the test reads whatever the file holds, so the example moved and the property did not); `Balancer` passes because `[item-name] bbb-balancer-part=Balancer part` is what this mod calls its machine; `Custom` passes nothing, because no value of either dropdown is spelled that way and no prototype is named that.

**THE RELATIONSHIP IS WHAT IS PINNED AND THE WORD IS NOT.** Nothing in the test writes `Custom` down. Add a value to `RecipeOptions()` or `TechOptions()` with a label and every entry in that group may name it; withdraw one and every entry naming it fails. `capitalisedWords` is the reading underneath: a word is what whitespace separates, stripped of surrounding punctuation and backticks, counted when its first rune is upper case.

**WHAT IT DELIBERATELY DOES NOT CATCH, which its own comment lists.** It reads WORDS and not SENTENCES, so a description of a behaviour this mod no longer has, written in ordinary lower case, passes untouched. It SKIPS THE FIRST WORD OF EVERY SENTENCE in a description, because a sentence opens with a capital whatever it opens with, so a description beginning with a withdrawn option word is missed; it does not skip the first word of a `[mod-setting-name]` entry, since a label is not a sentence, and that is exactly where four of the ten hid. Its sentence boundary is `.`, `:`, `;`, `!` or `?`, which is over-generous on the colon and the semicolon and over-generous in the safe direction, since the cost is a missed word rather than a false failure. And it scans only settings bound into a dropdown group, so the hand-rolled `bbb-multi-edge-parts` is not read at all, which is why the word `Factorio` in its name is nobody's business here.

**AND ONE FALSE POSITIVE, MEASURED RATHER THAN IMAGINED.** A bound setting's entries may not use a capitalised display or product name mid-sentence. Replacing the ingredients entry's `not the names shown on screen.` with ``not the names shown on screen (say `iron-plate`, not "Iron plate").`` and running `go test ./tune/ -run TestNoSettingDescriptionNamesAnOptionNoDropdownOffers -count=1` exits **1** on:

```
[mod-setting-description] better-belt-balancer-recipe-ingredients names "Iron", and bbb-recipe-cost offers no value spelled that way: it allows vanilla, cheap, belt-fast, belt-express, splitter, splitter-express and nothing else. ...
```

`README.md:65` already makes exactly that point in exactly that phrasing (`` Internal names are the ones the game uses in its own data and in rich text (`iron-plate`, not "Iron plate") ``), so an author moving the sentence into the tooltip meets this. THE REMEDY IS NOT TO WEAKEN THE TEST, because the word it would have to start admitting is the word it exists to refuse: lower-case the display name, or give the prototype a name in one of this mod's own `[entity-name]`, `[item-name]`, `[recipe-name]` or `[technology-name]` entries, which the test already reads. `Factorio`, `Factoriopedia` and `Startup` are the same shape.

#### Its group membership is derived from the plan as well, and that is the layer out

The table of groups is a literal, and a literal that fell behind the plan would leave a seventh bound setting silently unscanned, which is the failure mode the test exists to close arriving one layer out. `TestEverySettingThisPlanDeclaresIsDescribed` (`locale_test.go:214`) holds exactly such a literal list of the six, and the new test deliberately does NOT compare against it. **THE PLAN IS ASKED INSTEAD.** `*fkrecipes.Lib` exports no accessor for the settings it holds, so the test runs `Plan().CheckLocaleWith(ModName, "", HandRolledSettings())` against an EMPTY locale file and reads the one `the setting <name> has no [mod-setting-name] entry` finding per declared setting; the findings ARE the enumeration, in declaration order. A hand-rolled name is not among them, because the hand-rolled list only suppresses orphans and never creates an obligation. If the library ever changes that sentence the loop finds nothing and the length assertion fails loudly rather than passing over an empty reading, which is the safe direction for a check whose input is a message.

#### Red proofs

Three, each break made in this mod's own file, the designed failure observed at its exit code, the break reverted. `go test ./tune/ -count=1` back at **exit 0** and `git diff --stat` back at `367 insertions(+), 88 deletions(-)` after each. FkRecipes and FkLua were read-only throughout.

| injected | what fired |
|---|---|
| `[mod-setting-name] better-belt-balancer-recipe-ingredients` put back to `Custom balancer part recipe` | exit **1**, `[mod-setting-name] better-belt-balancer-recipe-ingredients names "Custom", and bbb-recipe-cost offers no value spelled that way: it allows vanilla, cheap, belt-fast, belt-express, splitter, splitter-express and nothing else.` This is the NAME half, and it is the half the sentence-opener skip does not cover |
| `, or Custom to write the cost yourself in the three settings below` put back into `bbb-tech-cost`'s description | exit **1**, `[mod-setting-description] bbb-tech-cost names "Custom", and bbb-tech-cost offers no value spelled that way: it allows logistics, logistics-2, logistics-3 and nothing else.` This is the DESCRIPTION half, mid-sentence, which is where the other six were |
| `SettingTechSeconds` dropped from the `bbb-tech-cost` group's `beside` list | exit **1** on BOTH coverage assertions: `this plan declares the setting better-belt-balancer-tech-seconds and no dropdown group above scans it ...` and `the groups above scan 5 settings (...) and this plan declares 6 (...)`. Which is what says the enumeration is read off the plan and not off the table |

**AND ONE WHOLE-FILE READING, WHICH IS THE TEST RUN AGAINST THE DEFECT ITSELF.** HEAD's entire `.cfg` restored into the working tree and the test run five times reports **13 errors every time**, byte-identical across the five (`for i in 1 2 3 4 5; do go test ./tune/ -run TestNoSettingDescriptionNamesAnOptionNoDropdownOffers -count=1 2>&1 | grep -c 'locale_test.go:471'; done` prints `13` five times). The 13 cover all ten entries, with `bbb-recipe-cost`'s description contributing two occurrences and `bbb-tech-cost`'s three; `Logistics`, `Default` and `Balancer` are in that same file and none of them is reported, which is the vocabulary derivation doing its half of the work.

#### What this closes, and what it does not

**FINDING 18 AND REMEDY (g) ARE ANSWERED BY NOT WRITING THE WORD.** The adopt commit's golden already records the library composing `The word none empties the list, so the recipe costs nothing to craft.` onto the ingredient field's format line (`ingredientNoneClause`, `go/customize.go:969`) and deliberately not onto the pack field's, whose parser turns the word down with `research takes at least one science pack` (`go/ingredientlist.go:282`). So an entry saying it here would duplicate the library on one field and CONTRADICT it on the other. The `.cfg` says that in a comment block above the ingredients entry rather than in the entry, which is the correct place for a decision not to write something.

**HALF OF FINDING 9 IS CLOSED BY THE LIBRARY'S COMPOSED RANGE LINES AND THE OTHER HALF IS WHAT THESE TWO ENTRIES NOW ARE.** `A whole number from 0 to 1000000. While it is 0 the option chosen above decides.` and its 3600 twin (`go/customize.go:820`) put the ranges in words where a player looks, which is the half the adopt commit's golden moved. The half that was always this mod's is the entry saying what the number is FOR, and each of the two is now exactly that one sentence and nothing else: a player who sets a number and sees nothing change is answered by the composed line under it rather than by a clause naming a state that does not exist.

**FINDING 4 IS CLOSED BY THE ROUND RATHER THAN BY THIS COMMIT, AND NOTHING IN THESE NOTES HAD SAID SO.** `grep -n 'finding 4' agents/fkrecipes-migration.md` has exactly one hit in the whole file, `:854`, which is round one's changelog paragraph scoping where the silence was measured; this round's section did not name it once before this paragraph. Finding 4 is MISLED: the settings screen presents ignored values as if they were live, photographed on a client, with the engine's own settings dump byte-identical whatever the player stored, so nothing on the screen could ever have said otherwise. ITS WHOLE MECHANISM WAS A STORED TEXT THE MOD READ AND THEN IGNORED. With the text as the switch there is no such state: a field holding something other than `default`, or a number other than 0, IS in force, and a field the player left alone is a field whose row the dropdown decides. The finding has no subject left, and the four entries it quoted as stating a rule the screen could not state are four of the ten rewritten above.

**FINDING 15 WAS CLOSED IN THE ADOPT COMMIT**, from the library's end, by `e4604d4`'s `{"?", {"<section>.<key>"}, "<raw>"}` wrapper; this commit adds nothing to it and takes nothing from it.

**AND NOTHING HERE IS SEEN BY ANY GATE BUT THE NEW TEST.** `--dump-data` still carries the key and not the string; the library's checker still reads keys; `make check` reads Go and not prose. What `go test ./tune/` reads of this file is one capitalised word at a time, against two vocabularies, in six entries of a file whose settings sections run to paragraphs. Every other sentence rewritten in this commit is read by a human or it is not read.

#### A prohibition kept, which is item 3 of the round

`[technology-name]` defines `bbb-balancer=Belt balancer` and nothing else (`awk '/^\[technology-name\]/{f=1;next} /^\[/{f=0} f&&NF' mod-data/locale/en/better-belt-balancer.cfg`), so remedy (f) stays undone and `TestTheLocaleFileRenamesNoTechnologyOfTheGames` goes on being what stops somebody carrying it out. This commit rewrote the file around that section and did not touch it.

#### Five stale sentences this commit deliberately did not touch

**ALL FIVE ARE REWRITTEN BY THE THIRD COMMIT, BELOW**, together with the labels they count. All five are in the `[string-mod-setting]` label-census comment block, and they belong to the LABELS commit rather than to this one: the labels themselves do not move here, and a census rewritten without the measurement that moves with it would be two commits' work reported as one. Re-measured against the working tree with

```
awk '/^bbb-(recipe|tech)-cost-/{k=$0; sub(/^[^=]*=/,""); split(k,a,"="); print a[1], length}' mod-data/locale/en/better-belt-balancer.cfg
```

which prints `50 38 58 65 35 52 45 43 46` for the nine labels in file order:

1. `:328`, "the one place a player looks before switching to Custom". There is no switching to Custom; what the two vocabularies sit one under the other FOR is a player about to type in the field.
2. `:343`, "TEN OF THIS MOD'S ELEVEN dropdown labels are at or past it". There are NINE labels now, and EIGHT of the nine are at or past the ~37-character truncation the assessment measured; only `bbb-recipe-cost-splitter` at 35 is clearly under. The six preset figures the block quotes, 35, 38, 50, 52, 58 and 65, are all still exact.
3. `:346`, "`custom` is 52". That label was deleted with the value in the adopt commit and no `-custom` key remains in the section.
4. `:346`, "all four of `bbb-tech-cost`'s are over at 43, 45, 46 and 54". THREE survive, at 43, 45 and 46; the 54 was the custom label.
5. `:319`, the cross-reference calling the label question "item 4 of `What BetterBeltBalancer must do to adopt this`". Item 4 of that list is now `CustomCost.Seconds` taking an `IntSettingRef`; the label question is not numbered in it at all any more. (**THIS ENTRY IS WRONG AND THE THIRD COMMIT CORRECTS IT.** FkRecipes has written that heading twice, and the cross-reference named the FIRST list, where item 4 IS the label question. The sentence was not stale; only the sentences around it were.)

#### What is still owed, and is now scheduled

**`README.md` IS STALE IN AT LEAST FIVE PLACES AND NO GATE READS IT EITHER** (`grep -n 'Custom' README.md`): `:49` "each has a Custom option with settings beside it where you write the value yourself"; `:63`, the option table row `| Custom | whatever you write in the Custom balancer part recipe setting below it |`; `:65`, "**Custom balancer part recipe** is read only while the recipe setting is on Custom"; `:67`, "Logistics (the default), Logistics 2, Logistics 3, or Custom"; and `:69`, the whole Custom paragraph, whose "Left alone they are 20 automation science packs at 15 seconds each" is now 0 and 0 with the tier deciding, and whose closing placement clause has the same `CustomCost.Position` behind it that `bbb-tech-cost`'s did. The four the brief named are `:63`, `:65`, `:67` and `:69`; `:49` is a fifth.

**`mod-data/changelog.txt`'s UNRELEASED 0.3.3 ROWS ANNOUNCE THE OPTION ON BOTH SETTINGS AND NEED A REWRITE RATHER THAN AN ADDED SENTENCE.** Three rows name it (`grep -n Custom mod-data/changelog.txt`): `:7` the recipe feature row, `:8` the research feature row (which also carries the 20 and the 15 as "left alone", and a placement paragraph), and `:10` the Info row warning that launching an older release resets a setting left on Custom. All three are about a value that never shipped, so remedy (h)'s ask, one sentence disclosing that a typo in the text field is now a way to change the recipe without touching the dropdown, lands inside a rewrite rather than beside the existing rows.

**AND THE TWO THE ROUND ALREADY HAD.** The nine dropdown labels with their census block, which is the five sentences above plus the KEEP decision re-argued at nine labels rather than eleven; and the remover fixture, remedy (i).

**THREE OF THOSE FOUR ARE THE THIRD COMMIT AND ARE DONE**, and this schedule is kept as the schedule it was: `README.md`'s five places, the three 0.3.3 rows, and the nine labels with their census block. What is left of it is the remover fixture. The KEEP decision was not re-argued at nine labels; it was REVERSED, and the subsection under this one is why.

### The round's third commit: the nine labels become names, and the changelog and the README stop announcing a value that never shipped

The round's first commit withdrew `custom` from both dropdowns, its second rewrote the ten locale entries that still told a player to pick it. What was left was every OTHER player-facing string: nine option labels whose shape rested on a decision fix round 1 took and this round's library reverses, and two documents, one of which announces the withdrawn value as a feature.

#### The nine labels, before and after, and the three composed lines with them

Counted off the file with the command the second commit's subsection already uses,

```
awk '/^bbb-(recipe|tech)-cost-/{k=$0; sub(/^[^=]*=/,""); split(k,a,"="); print a[1], length}' mod-data/locale/en/better-belt-balancer.cfg
```

run at `e67a5d0` and again on the working tree:

| key | before | chars | after | chars |
|---|---|---|---|---|
| `bbb-recipe-cost-vanilla` | `Default: 4 iron plates, 2 gears, 2 transport belts` | 50 | `Default` | 7 |
| `bbb-recipe-cost-cheap` | `Cheap: 2 iron plates, 1 transport belt` | 38 | `Cheap` | 5 |
| `bbb-recipe-cost-belt-fast` | `Fast belts: 4 iron plates, 2 gears, 2 fast transport belts` | 58 | `Fast belts` | 10 |
| `bbb-recipe-cost-belt-express` | `Express belts: 4 steel plates, 2 gears, 2 express transport belts` | 65 | `Express belts` | 13 |
| `bbb-recipe-cost-splitter` | `Splitter: 1 splitter, 2 iron plates` | 35 | `Splitter` | 8 |
| `bbb-recipe-cost-splitter-express` | `Express splitter: 1 express splitter, 2 steel plates` | 52 | `Express splitter` | 16 |
| `bbb-tech-cost-logistics` | `Default: Logistics, alongside transport belts` | 45 | `Logistics (default)` | 19 |
| `bbb-tech-cost-logistics-2` | `Logistics 2, alongside fast transport belts` | 43 | `Logistics 2` | 11 |
| `bbb-tech-cost-logistics-3` | `Logistics 3, alongside express transport belts` | 46 | `Logistics 3` | 11 |

**EIGHT OF THE NINE WERE AT OR PAST THE CLIENT'S MEASURED TRUNCATION AND NONE OF THE NINE IS NOW.** The closed dropdown cuts by pixel width near 37 characters, which is two client readings of two different labels at one UI scale each and is the whole of the evidence for the width: 36 characters of a 52-character label in the second assessment's finding 5, 38 of another in the first. Only `splitter` at 35 was clearly under it. The widest label left is 19, exactly half of the wider of those two readings.

**THE THREE COST LABELS ARE THE HALF THAT COMPOSES**, because FkRecipes extends a cost preset's own line with `": cost of ", {"technology-name.<first source>"}` rather than opening a second line under it, as an ingredient preset's `  type:` line does. The technology is its LOCALISED name and not its internal one, which is `da11cf5` adopted in the sync pass, so what the label costs is paid on the same line a player reads the technology's name on. Resolved out of the engine's own settings dump against base's locale file and this mod's, the three lines go

```
64  Default: Logistics, alongside transport belts: cost of Logistics    ->  38  Logistics (default): cost of Logistics
64  Logistics 2, alongside fast transport belts: cost of logistics-2    ->  32  Logistics 2: cost of logistics-2
67  Logistics 3, alongside express transport belts: cost of logistics-3 ->  32  Logistics 3: cost of logistics-3
```

against a tooltip wrap threshold FkRecipes measured near 57 to 60 characters on 2.0.77: all three were over it and all three are now well under. `Logistics` is base's own `[technology-name] logistics=Logistics` and the two upgrade tiers render RAW, because base defines neither (`awk '/^\[technology-name\]/{f=1;next} /^\[/{f=0} f&&/^logistics/' <Factorio>/data/base/locale/en/base.cfg` prints exactly one line, `logistics=Logistics`). That is finding 15 surviving as legible text through `e4604d4`'s `{"?", {"<section>.<key>"}, "<raw>"}` wrapper, and it is why the label is the only readable name IN THIS TOOLTIP that a player gets for `logistics-2` and `logistics-3`.

#### The decision is REVERSED, and what moved is the question rather than the arithmetic

Fix round 1's commit 3 answered the assessment's item (h) with KEEP, and it answered it honestly: it built the alternative it was offered, measured it, and published its own counter-evidence. That alternative was **internal names** -- each label rebuilt as the display head, a colon, and the preset's internal list -- and the measurement stands today: `vanilla` 50 to 58, `belt-fast` 58 to 66, `belt-express` 65 to 73, against `cheap` 38 to 37, `splitter` 35 to 34 and `splitter-express` 52 to 51. Three of six grow, two of the three go past every reading of the width, and they clip mid-name. Nothing in that is wrong and none of it is retracted here. Its table is kept where it was written.

**WHAT THE ROUND'S LIBRARY ADDED IS A THIRD OPTION THAT ROUND HAD NOT BEEN OFFERED, AND A RULE FOR CHOOSING IT.** `docs/usage.md` now states it for an author outright: *"Give a dropdown option a short name, not a sentence."* Its reason is arithmetic rather than taste: the library composes the preset's typeable list beside the label, the tooltip wraps near 57 to 60 characters, "so every character of the label is a character the list has to give back", and a wrapped line's continuation starts at the left margin, so a player copying what looks like a whole line loses the end of the list. Rewriting the six presets as NAMES makes all six shorter, so the length argument that decided round 1 decides nothing here, and what settles it is the duplication: `Default: 4 iron plates, 2 gears, 2 transport belts` sat directly above `  type: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt`, which is the same recipe in the only spelling the field beside it accepts. Two lines for one fact, and the first of them the half a player cannot paste.

**AND THE ONE-LINE SHAPE THAT WOULD HAVE MADE A LONG LABEL CHEAP IS ALREADY REJECTED, ON ARITHMETIC, IN THE LIBRARY'S OWN RECORD.** `agents/implementation-notes.md`'s decision E takes the second assessment's remedy for finding 5, which was to put the label and the typeable list back on one line as `<label>: <typeable list>`, and refuses it: on the assessment's own example that line is 71 characters where today's indented `  type:` line is 66, against the same 57-to-60 threshold, and it has MORE comma boundaries for a wrap to land on -- which matters because a truncated copy is only silently wrong when the cut lands on a comma, and anywhere else the language refuses it loudly. So the two lines stay two lines, the label's whole job is to name the option, and a label that is a sentence is a label spending characters the list needs.

**WHICH ADOPTION LIST, BECAUSE THE SECOND COMMIT GOT THIS WRONG AND THIS ONE CORRECTS IT.** That commit recorded the `.cfg`'s cross-reference to "item 4 of `What BetterBeltBalancer must do to adopt this`" as a stale sentence on the ground that item 4 is now `CustomCost.Seconds`. FkRecipes has written that heading TWICE (`grep -n 'What BetterBeltBalancer must do to adopt this' ../FkRecipes/agents/implementation-notes.md` prints `:595` and `:1536`), and the citation named the FIRST: fix round 1's item 4 is `(h) Leave the preset labels alone, or rewrite them, but decide it`, and fix round 2's item 4 is the `IntSettingRef`. The cross-reference was correct and the stale-sentence entry naming it was not; the `.cfg` now distinguishes the two lists by name and the second commit's list is annotated above.

#### Two findings this rewrite produced, and neither was flagged anywhere

**`Logistics 2, alongside fast transport belts` ASSERTED WHAT BASE UNLOCKS, INSIDE A MOD WHOSE WHOLE DESIGN IS TO READ A TIER'S PRICE OUT OF THE GAME RATHER THAN WRITE IT DOWN.** The reading is confirmed against the engine's own data-raw dump: base's `logistics-2` unlocks `fast-transport-belt`, `fast-underground-belt` and `fast-splitter`, and its `logistics-3` unlocks the three express ones (`jq -r '.technology["logistics-2"].effects[].recipe' base-data-raw.json`, over the same `--dump-data` capture the gate takes). So the clause was true of base and was a claim about base, written down in a locale file, with no gate anywhere able to read a syllable of it. In a pack that repurposes that technology -- the exact case the three tiers exist to follow, since the cost is copied rather than declared -- the clause is simply false and nothing can notice. The names survive because they must: `logistics-2` and `logistics-3` have no localised name in base, so the label is the only readable one the tooltip has. The claim does not survive.

**AND THE DEFAULT TIER'S CLAUSE WAS ALREADY FALSE OF BASE, WHICH IS A SECOND FINDING AND IS NOT RECORDED ANYWHERE ELSE.** `Default: Logistics, alongside transport belts` says the default tier comes alongside transport belts. Base's `logistics` unlocks `underground-belt` and `splitter` and nothing else, and `transport-belt`'s recipe is enabled from the start and is unlocked by no technology at all (`jq -r '.technology.logistics.effects[].recipe' base-data-raw.json` prints `underground-belt` and `splitter`; `jq '[.technology[]|.effects//[]|.[]|select(.recipe=="transport-belt")]|length' base-data-raw.json` prints `0`; and `jq '.recipe["transport-belt"].enabled' base-data-raw.json` prints `null`, which is the engine's default of true). A player who read that label and went looking for the technology that gives them belts would not find one. It is gone with the rest of the clause rather than fixed, which is the same answer for the same reason.

#### The composed tooltips, re-measured, and what the rewrite does not fix

Six settings, measured the way the adopt commit measured them: the localised-string structure read out of the engine's own `--dump-data` settings dump, its parameters and depth taken with the `jq` in the library's decision E, and the CHARACTERS resolved by walking that structure against base's `locale/en/base.cfg` and this mod's `.cfg` -- concatenating every `""` group and taking the first defined key of every `{"?", ...}` alternatives group, which is the engine's own rule. Before is `e67a5d0`, after is the working tree; the dump itself has not moved since the adopt commit, because a label is a locale string and no prototype changed.

| setting | params | depth | lines | longest | chars before | chars after |
|---|---|---|---|---|---|---|
| `bbb-recipe-cost` | 9 of 20 | 4 of 20 | 15 | 443 | 1,209 | **970** |
| `bbb-tech-cost` | 5 of 20 | 4 of 20 | 5 | 563 | 826 | **733** |
| `better-belt-balancer-recipe-ingredients` | 6 of 20 | 3 of 20 | 6 | 355 | 902 | 902 |
| `better-belt-balancer-tech-packs` | 6 of 20 | 3 of 20 | 6 | 315 | 768 | 768 |
| `better-belt-balancer-tech-count` | 2 of 20 | 3 of 20 | 2 | 80 | 135 | 135 |
| `better-belt-balancer-tech-seconds` | 2 of 20 | 3 of 20 | 2 | 77 | 144 | 144 |

**FOUR OF THE SIX DO NOT MOVE AT ALL AND THAT IS THE SHAPE OF THE CHANGE.** A label is composed into the two DROPDOWN descriptions and nowhere else, so the four generated settings' tooltips are byte for byte what the second commit left. Neither the parameter count, the depth nor the LINE COUNT moves on the two that do: a label is one line however long it is, so what the rewrite buys is width and not height. The longest line of each is unmoved too, because on both dropdowns the longest line is line 1, which is this mod's own `[mod-setting-description]` entry and is the second commit's work. Against the library's re-measured ceilings of **20 parameters per table** and **20 levels of depth**, with no global table budget, the widest is 9 of 20 and the deepest 4 of 20, which is where the adopt commit left them.

**WHAT THE REWRITE DOES NOT FIX IS THE WRAP, AND IT IS NOT THIS MOD'S TO FIX.** The six `  type:` lines are the LIBRARY's rendering of the lists this mod declares in `guest/go/tune/recipe.go`, and they do not move with the labels: measured in the same dump they are 57, 38, 62, 66, 32 and 41 characters, so at least the two longest are past the 57-to-60 threshold and their continuation starts at the left margin. That is finding 5's residue, the wrapping is the engine's, and the library discloses it in the same tooltip (`A list too long for one line continues on the next; the continuation is part of the same list.`). Shortening a label cannot reach it, because the label is on a different line.

**NOT MEASURED: WHETHER ANY OF THE SIX RENDERS WHOLE, OR WHETHER ANY OF THE NINE LABELS RENDERS WHOLE.** No client was run in this commit and none was started. Every number above is a character count; the cut in the closed dropdown is by pixel width and the wrap in the tooltip is by pixel width, and the whole of the evidence for either is a handful of client readings from earlier rounds. This is the client run owed since round three, and it has two more things to look at than it did: the two CLOSED dropdowns, whose truncation is the whole reason the nine labels moved and which no number in this file has seen render one.

#### The changelog: both feature rows rewritten, two rows added, one row DELETED WHOLE, and remedy (h) answered

0.3.3 is unreleased, which is the whole reason its section is editable at all; a published section is the one edit a changelog must not take. At `e67a5d0` the section was one `Bugfixes:` row, two `Features:` rows and two `Info:` rows. It is now one `Bugfixes:` row, two rewritten `Features:` rows, a new `Changes:` row and two `Info:` rows of which one is new and one is extended.

- **The recipe `Features:` row** announced "a Custom option, and a new setting beside it". It now announces the second setting alone, says it applies whenever it does not say `default`, says that while it says `default` the option picked above is what a player gets, and says all six options work as they did and a stored choice is kept. It gains the word `none`, which is FkRecipes' decision E composing `The word none empties the list, so the recipe costs nothing to craft.` onto that field and onto no other, **and it gains the consequence the library's own sentence stops short of**: the recycling recipe the engine derives from this one goes with the ingredients, so where a player has recycling a balancer part can no longer be recycled. Finding 18 is answered in the locale file by NOT writing the word and here by writing it, which is not a contradiction: one is a tooltip the library has already written on and the other is a release note the library never sees.
- **The research `Features:` row** announced the Custom option, the "left alone they are 20 automation science packs at 15 seconds each" pair, and a placement paragraph built on the deleted `CostChoices.Position` ladder. It now announces three fields that reprice one at a time, the pack field switching on the word and each number on 0, a field left alone decided by the tier picked above, and the prerequisite never moving: the tier picked, or the next belt tier down where a mod set leaves that one unpriced, and none at all where no tier below it can be read either.
- **A `Changes:` row is added**, which the section did not have. It names the nine labels in both dropdowns, says what each option is made of or priced like is in the tooltip instead (the recipe lists in the internal names the new field takes, the research tiers by the technology's own name), says the old labels were long enough that the closed dropdown cut most of them short, and says that nothing any option DOES changes and a stored choice is kept. `Changes:` rather than `Features:` because nothing was added and nothing was fixed, which is the same reading of the file's categories that put fix round 1's two entries under `Info:`.
- **AN `Info:` ROW IS ADDED FOR `max_level`, AND IT IS THE FIRST TIME THAT BEHAVIOUR HAS BEEN SAID TO A PLAYER AT ALL.** Round two graded it AWKWARD and asked the library for an opt-out (the graded entry `` `max_level` travels with the copied unit: AWKWARD``, above): `CostBy` copies the source technology's `max_level` beside its unit, where the hand-rolled stage this replaced read three fields of the unit and nothing else, so in a pack that gave a logistics tier several levels this mod's research gains them and only the first level unlocks anything. The ask was declined with a documented workaround and the behaviour shipped in 0.3.3; nothing a player reads had said so. The row says it, and says the stock-game half in the same breath, which is what keeps it from reading as a warning about the ordinary case: none of Logistics, Logistics 2 or Logistics 3 carries a level cap with or without the expansions.
- **The rollback-hazard `Info:` row is DELETED WHOLE and not rewritten**, and that is a decision rather than a tidy-up. It warned that launching 0.2.1, 0.2.2, 0.3.1 or 0.3.2 once resets a setting left on Custom and persists the reset with nothing in the log. The effect is real and the measurement behind it stands; what has gone is the subject. `custom` never shipped in a public release, so there is no player who can have stored it, and a release note that tells a player to check for something they could never have lost is worse than silence. **NO ROW MAY CLAIM ANYTHING WAS REMOVED FROM A PLAYER**, because nothing was: the six recipe values and the three tiers are the same nine values under the same nine names, and the adopt commit's golden is what says the withdrawal reaches the settings stage and nothing else.
- **The assembler `Info:` row carries remedy (h)**, which is the one sentence the second assessment asked for and the one place it fits. The row already disclosed that changing what a balancer part costs rewrites the one recipe this mod has, so an assembling machine set to make balancer parts loses whatever its input slots hold that the new list does not use, destroyed rather than dropped and with nothing in the log -- finding 8, 37 loads with `world.item-on-ground` empty in all of them. What this round ADDS is a second way in: a typo introduced into a list that was working sets the whole list aside and the option picked above applies instead, so it is a recipe change nobody asked for, and correcting the typo is a second one. That is finding 8's cost reached by a route the player did not know they had taken, and it is disclosed where a player looks rather than in the log.

#### README: every paragraph that moved, and three things that were false independently of the withdrawal

Seven paragraphs of "Cost and research" move, which is the five places the second commit's subsection named plus two it did not.

- **`:49`, the section's opening.** "each has a Custom option with settings beside it where you write the value yourself" becomes the fields themselves: one ingredient list under the recipe setting, three research fields under the cost setting.
- **`:63`, the option table.** The `| Custom | whatever you write in the Custom balancer part recipe setting below it |` row is deleted. Six rows, the six values the dropdown allows, in the order it allows them, and they stay in DISPLAY names: this table is now the only place on a player's screen or in their documentation where a preset is spelled the way the game shows it, which is the one thing the label rewrite costs and the `.cfg` says so.
- **`:65`, the ingredients field.** The bold lead moves from **Custom balancer part recipe** to **Balancer part ingredients**, which is the name the second commit gave the setting, and "is read only while the recipe setting is on Custom" becomes "applies whenever it does not say `default`". It gains the word `none` with the recycling consequence beside it, and a pointer to the assembler paragraph at the end of the section, because the typo route is a recipe change and this is the paragraph that creates it.
- **`:67`, the research dropdown.** Rewritten, and this is the first of the three things that were false on their own terms. It said the setting "picks which technology unlocks the balancer". It does not and never did: what unlocks the balancer is this mod's own technology, `TechName = "bbb-balancer"` in `guest/go/tune/plan.go:589`, whose `Unlocks` list carries the recipe. What the dropdown picks is the technology the research is PRICED LIKE, which is also its prerequisite. The sentence now says that.
- **`:69`, the three research fields.** The old paragraph was the `CostChoices.Position` ladder in the player's words -- "A custom research sits after Logistics, which is where the setting's default already puts it ... without Logistics it sits after the lowest of Logistics 2 and Logistics 3 your mods have, and has no prerequisite if none of the three exists" -- and that is the second independently false thing, because the ladder it described was deleted by the adopt commit and there is no option it is now true of. Every value names a source technology and that source is the prerequisite, whatever the three fields say. The paragraph now says the three fields reprice one at a time, that repricing never moves the research in the tree, and that setting the unit count drops a `count_formula` the picked tier may carry, because the engine prices a research holding both by the formula and the typed number would be read by nobody.
- **`:71`, the overhaul-pack paragraph.** "A custom list is the opposite on purpose" becomes "A list you write yourself", and what an unusable list falls back to is named as the option chosen above rather than as this mod's own list, which is what the adopt commit actually made true. It also picks up the `max_level` disclosure in the same words as the changelog row. And it carries the third independently false thing, in its old form at `:69`: "in a mod set that has no automation science pack, Custom refuses to load ... **whether you typed anything into the pack field or not**". The refusal is narrower than that on both shapes. On the OLD shape a player who typed a pack list the game HAS was priced in it and loaded, so the clause was wrong about its own release; on the new one an unusable pack text is set aside for the TIER's own packs first, and the load stops only where the fallback cost is reached and the game has no automation science pack either -- by either of its two routes, no tier readable at all, or a tier read whose every pack is missing. That is what `TestAFallbackThisModsOwnPackListCannotPayForStillRefuses` pins, and the paragraph now spells out both routes rather than attaching the refusal to an option. (The claim about the old shape is a reading of a declaration this repository no longer compiles, not a re-run: the API it rested on is deleted. It is written down because the new sentence had to be narrower than the old one either way.)
- **`:73`, the assembler paragraph.** Gains the typo route, in the same words the changelog's `Info:` row carries it.

#### What is still owed

- **The remover fixture, remedy (i).** `check_remover`'s fixture breaks Space Age before this mod can be judged on a stock install, so the arm it guards is only honest with the expansions disabled, and the fixture should sweep at `data-final-fixes` the way the synthetic one built for the assessment does.
- **The round's close.** The package sizes with the growth attributed rather than assumed, the relay's jump figures out of `dist/.fklua-mod.report.json`, the release-to-head proof re-run over its 28 pairs, and every gate consolidated at its exit code. None of it is taken at this commit, because the settings dump moves once more if the fixture does.
- **The client run**, which is not this round's to close and is one hover larger than it was: the two tooltip hovers round one asked for, and now the closed dropdown itself, whose truncation is the whole reason the nine labels moved and which no number in this file has seen render.

### The round's fourth commit: the fixture deletes at the data stage and sweeps at data-final-fixes, and the arm about a neighbour finally runs with one installed

Remedy (i) is the last of the four the round set out with and the only one whose subject is a TEST rather than something a player reads. `test/check-datastage.py`'s synthetic `bbbt-remover` deleted `transport-belt` AND swept every recipe, technology effect and entity reference that deletion dangled, all at the data stage, which kills space-age before this mod can be judged; so the one arm that exists to answer the assessment's finding 1 was honest only in the configuration the gate actually runs, `DLC` all False. That is still what it runs (`python3 -c "...; print(m.DLC)"` over the script's own constants prints `{'elevated-rails': False, 'quality': False, 'space-age': False}`), and it is deliberate: this gate is about what a NEIGHBOURING pack does to this mod's ladder, not about the expansions. What remedy (i) asks is that the fixture be a pack a stock Space Age install survives, and the assessment's own paragraph measured the failure and worked around it with a two-stage fixture built for the occasion. This commit makes the shipped fixture that one.

#### What moved, what stayed, and why each is where it is

- **THE ITEM DELETION STAYS AT THE DATA STAGE, and it is the arm's whole premise.** This mod's ingredient ladder asks the item table at ITS OWN data stage, and the fixture's name sorts first, so a deletion any later is a deletion the ladder never sees. The arm asserts the ordering rather than trusting it, off the engine's own `Loading mod <name> <version> (data.lua)` lines: `['core', 'base', 'bbbt-remover', 'belt-balancer-2', 'better-belt-balancer']` on the incumbent mod set.
- **THE RECIPE SWEEP MUST MOVE, AND THE ENGINE NAMES THE FILE AND THE LINE.** A mod's `data-updates` is entitled to find base's own prototypes whole, and space-age helps itself to one this fixture had taken away. Reproduced here without an engine, on the installed 2.0.77 data: `awk 'NR==241' "$D/space-age/base-data-updates.lua"` prints `data.raw.recipe["transport-belt"].category = "pressing"`. The engine resolves a prototype REFERENCE only after the last stage has run, so a sweep is safe as late as `data-final-fixes` and unsafe any earlier.
- **THE TECHNOLOGY-EFFECT SWEEP MOVES BECAUSE IT IS COMPUTED FROM THE RECIPE SWEEP, AND IT IS NOT A FORMALITY.** It drops the `unlock-recipe` effects naming a recipe the sweep killed, so it cannot run before that set exists, and on both mod sets it really drops some. Reproduced off base's own prototype files rather than off a dump: SIX base recipes consume the belt (`python3` over `base/prototypes/recipe.lua`, walking back from each `name = "transport-belt"` ingredient line to its recipe's own `name`, prints `logistic-science-pack`, `lab`, `splitter`, `underground-belt`, `loader`, `fast-transport-belt`), and of those six plus the belt's own recipe, FIVE are unlocked by a technology (`python3` over `base/prototypes/technology.lua` for `recipe = "<name>"`, prints `electronics/lab`, `logistic-science-pack/logistic-science-pack`, `logistics/splitter`, `logistics/underground-belt`, `logistics-2/fast-transport-belt`; `loader` and `transport-belt` are unlocked by NO technology in base, the belt being enabled from the start). A SIXTH goes on the incumbent mod set, `belt-balancer-1/belt-balancer-normal-belt`, which is a NEIGHBOUR's technology losing a NEIGHBOUR's recipe and is the whole class of thing the arm could not see while it only ever ran on `base`.
- **THE ENTITY-REFERENCE SWEEP IS FREE IN EITHER STAGE AND MOVES FOR ONE RULE RATHER THAN OUT OF NECESSITY.** The adversarial review rebuilt the fixture with that half alone left behind at the data stage, all three expansions on, and got exit 0 on both mod sets with every assertion green. So `minable`, `next_upgrade`, `place_result` and `placeable_by` are not what broke space-age and moving them is not what fixes it; they move because they dangle for the same reason the recipes do and are resolved at the same moment, so ONE rule covers the whole fixture: the deletion is at the data stage and everything the deletion dangles is swept at `data-final-fixes`. Splitting the two halves across two stages would be a second rule bought with nothing.
- **AND NO ORDERING IS ASSERTED FOR THE SWEEP, for a reason narrower than "it needs none".** Everything the sweep has to come after is in an EARLIER stage in every case that matters, quality's recycling recipes at `data-updates` and the stand-in's own recipe at the data stage, so the stage alone settles those. But `data-final-fixes` is a stage with an order inside it, measured on the incumbent mod set as `['bbbt-remover', 'better-belt-balancer']`, and THIS MOD EMITS ITS LEGACY STUB THERE, after the sweep has run. That is green because the stub is an item and an entity naming nothing the fixture removes, and not because the sweep is ordering-free; an assertion there would pin the alphabet for a dependency this fixture does not have.

#### The measurement that closes remedy (i), taken by hand and deliberately NOT made an arm

Both halves on Factorio 2.0.77 build 84539 with `elevated-rails`, `quality` and `space-age` all enabled, which is the stock install the assessment measured on and not the configuration this gate runs.

- **BEFORE**, the one-file fixture: `Failed to load mod "space-age": __space-age__/base-data-updates.lua:241: attempt to index field 'transport-belt' (a nil value)`, exit 1 and no dump, with this mod's own data stage already finished and the merge line already written. The mod under test had done everything right and the run could not say so.
- **AFTER**, the two-file fixture: exit 0 on BOTH mod sets, with the engine's own log carrying `Script @__bbbt-remover__/data.lua:17: bbbt-remover: removed 1 item name(s) at the data stage`, then `Loading mod space-age ... (data-updates.lua)`, then `Script @__bbbt-remover__/data-final-fixes.lua:65: bbbt-remover: swept 12 recipe(s) at data-final-fixes`, 14 on the incumbent arm.

**THE TWO LINE NUMBERS ARE THE FIXTURE'S OWN LENGTHS AND ARE REPRODUCED HERE**, which is how the script's comment block measures them: `build_remover` written into a scratch directory produces a `data.lua` of 17 lines whose `log(` call is line 17 and a `data-final-fixes.lua` of 65 whose `log(` call is line 65, so the engine's two `Script @__bbbt-remover__/...` prefixes are the files' own last lines and a file that grew would say so.

**AND IT IS A HAND MEASUREMENT ON PURPOSE.** Nothing in this commit turns the expansions on: `DLC` stays all False, because what the gate holds is the fixture's SHAPE, and an arm that needed three expansion mods installed would be an arm most machines skip. What stops the shape regressing is a question the arm asks of the RECIPE table with no expansion in sight: the belt's own recipe must be GONE and `iron-gear-wheel`'s must be there, the second being what tells a swept recipe from a `jq` path that stopped matching anything, and a fixture whose sweep went back to the data stage or stopped happening fails on `the transport-belt recipe survived the data-final-fixes sweep, so the fixture's second stage did not run and this arm is a one-stage fixture again`. The ITEM pair, `transport-belt` gone and `iron-plate` present, says the data stage ran; the RECIPE pair says the sweep did.

#### The second finding, which nobody asked for, and which makes remedy (i) visible with the expansions OFF

**THE OLD FIXTURE WAS BROKEN ON THE INCUMBENT MOD SET TOO, AND NO EXPANSION IS NEEDED TO SHOW IT.** The stand-in declares its own recipe naming the removed item (`guest/go/obs/bb2data/main.go:195` names it `belt-balancer-normal-belt` and `:119` gives it `ingredient("transport-belt", 5)`, with `:166` to `:170` hanging it off the `belt-balancer-1` technology), and it declares it AFTER the fixture's data stage has run, `bbbt-remover` sorting before `belt-balancer-2`. So put the sweep back at the data stage and the run refuses with `Error in assignID: item with name 'transport-belt' does not exist. It was removed by bbbt-remover. Source: belt-balancer-normal-belt (recipe).` on the DLC set this gate actually runs.

**WHY NOTHING SAW IT IS THE SHARPER HALF.** `run_arm` looked `ARMS` up by the arm's NAME, and `ARMS` has no `remover` row, so `ARMS.get(arm, [])` handed back the empty list and the arm staged the BASE mod set under whatever name it was called. The arm that is ABOUT a neighbouring pack was the one arm in the gate that never ran with a neighbour installed, every assertion it made passed, and all of them were about the wrong game. The label and the mod set are separate arguments now (`run_arm(..., mod_set=ms)`, with a caller that passes no `mod_set` keeping the old behaviour exactly, the name IS the mod set), `check_remover` runs the fixture over every row of `ARMS`, and the first thing each arm asserts is the mod list this script STAGED, read back out of each staged `info.json`, against the row the arm is named after (`this arm ran on a different game from the one it is named after`), and then every one of those names against the engine's own `load_order`, because a mod staged is not yet a mod loaded and `mods` is this script's reading rather than the engine's. Running both mod sets is not what remedy (i) asked for; it is what taking remedy (i) seriously turned up.

#### `transport-belt-recycling` never exists, and that is the control

**THE SWEEP IS A RE-SCAN AND NOT A RELOCATION, which is the half a pure move would have got wrong.** At `data-final-fixes` there are recipes that DID NOT EXIST when the item was deleted: quality builds a recycling recipe out of a recipe at its own `data-updates`, and a recycling recipe RETURNS what the original consumed, so `fast-transport-belt-recycling`, `lab-recycling`, `loader-recycling`, `splitter-recycling` and `underground-belt-recycling` each name the deleted item among their RESULTS and not one of them existed when `data.lua` ran. A sweep replaying a list `data.lua` had made would leave every one of them standing.

**AND THE CONTROL IS THE ONE THAT IS IN NEITHER RUN.** `transport-belt-recycling` never exists at all, because quality builds recycling recipes at `data-updates` and the ITEM was already gone when it looked. So the five are not the belt's own recycling recipe relocated; they are five recipes the data stage could not have known about, which is what makes "re-scan, not relocation" measured rather than argued.

**THE COUNTS, off the fixture's own `swept N recipe(s)` line**: 7 on `base` and 8 on `incumbent` with the expansions off, 12 and 14 with them on. The base seven are the six consumers reproduced above plus the belt's own recipe; the incumbent's eighth is the stand-in's `belt-balancer-normal-belt`; the five added by the expansions are exactly the five recycling recipes above, and the incumbent's fourteenth is `balancer-part-recycling`, the stand-in's own part under quality.

#### What the recipe's `localised_description` carries, which is NOTHING, and that is the assertion

The arm asks this mod's own emitted recipe two questions now, and the second is an assertion that something is ABSENT. **MEASURED on 2.0.77: `.recipe["bbb-balancer-part"]` is SEVEN KEYS** -- `enabled`, `energy_required`, `ingredients`, `name`, `order`, `results`, `type` -- with no `localised_description` and no `localised_name`, this mod declaring no `Description` on its recipe.

**AND IT IS RIGHT THAT IT IS NOTHING, ON BOTH OF THE LIBRARY'S TWO REASONS FOR WRITING ONE.** FkRecipes writes a trailing sentence into an emitted prototype's own `localised_description` for a stored setting value that FELL BACK (its decision C) and for a merged amount CLAMPED at the item or fluid ceiling (its decision B). This arm earns neither: there is no `mod-settings.dat` in the run at all, because the player it is about never opened the Startup tab, so no stored value exists to fall back; and the merge is `4 plus 2 is 6` against an item ceiling of 65535, which the second assessment records as unreachable through this mod, every declared amount in the six presets being single-digit. **THE LADDER MERGE ITSELF CARRIES NO NOTE BY DESIGN**, which is decision B in the library's own words (`../FkRecipes/agents/implementation-notes.md`, **THE LADDER GETS NO NOTE, DELIBERATELY**): a resolve-or-drop ingredient ladder is the library's advertised contract and the dropdown's own composed description already discloses it, whereas a clamped amount is arithmetic a player can check nowhere. So a note appearing HERE would be the library reclassifying its contract as a degradation, on the load of a player who never touched a setting, and that is worth failing over in either direction.

**THE ASSERTION IS THE KEY'S PRESENCE AND NOT ITS VALUE**, because the library emits the key or does not emit it at all, and a key present holding null is a different prototype from a key that is not there. It is asked of a prototype fetched ONCE and asked both questions in Python rather than through a second `jq` path, which is what makes a stale path unable to answer the absence question the right way for the wrong reason: a stale path yields `null`, the INGREDIENT assertion fails first and by name, and the absence question is never reached over a prototype the probe did not find.

#### The red proofs, and why two of them are taken against a recorded dump

Every break reverted, and each catches something a different one does not.

- The sweep put back at the data stage reddens `remover-incumbent` with the expansions OFF, on `Error in assignID ... Source: belt-balancer-normal-belt (recipe).`, and reddens `remover-base` with them ON, on space-age's `base-data-updates.lua:241`. Two mod sets, two DLC configurations, one binary, one defect.
- No sweep anywhere reddens on `Source: transport-belt (recipe).`, which is the deletion's own dangling reference and proves the sweep is doing work at all.
- The second mod set dropped from the comprehension reddens on `the arms that ran are ['base'] and 'ARMS' has ['base', 'incumbent']`. That line is a TRIPWIRE rather than an assertion and the file says so: on the code as written it cannot fire, because the runs and the heading's count both come from `ARMS`; what it is for is the EDIT that narrows the comprehension back to one mod set, which nothing else here would notice, there being no FAIL an arm that never ran can produce.
- `run_arm` ignoring `mod_set` reddens on `this arm ran on a different game from the one it is named after`, which is the defect this commit found, red-proved against the fix.
- A note injected into a copy of the dump reddens the tooltip assertion, and **the same dump with the tooltip check deleted goes GREEN**, which is what says the new check and nothing else caught it.
- A stale probe path reddens on the ingredient line rather than passing quietly, which is the ordering the whole probe was reshaped for.

**THE TWO TOOLTIP PROOFS ARE TAKEN AGAINST A RECORDED DUMP RATHER THAN AGAINST THE ENGINE**, which is the precedent `OUR_SETTINGS_ORDERS`' block in the same file already sets, having read its five moves off a kept `mod-settings-dump.json`: every probe in this file is a pure function of a dump PATH, so a doctored copy is a legitimate input to the same assertion. Here it is the ONLY input, and the reason is the next paragraph.

#### One thing this commit found and does NOT write up, and the next commit carries it

**A RECIPE-BOUND FkRecipes FALLBACK NOTE CANNOT BE EMITTED ON 2.0.77 AT ALL.** Taking the positive control for the assertion above -- a recipe whose stored value really did fall back, so a note really is earned -- refuses the load outright: `Error while loading recipe prototype "bbb-balancer-part" (recipe): Localised string key is too large: 247 > 200 (limit).` That is a BLOCKED-grade defect in the LIBRARY and not in this gate, it is why the absence this arm asserts has no engine-side positive control to sit beside it, and it is why an arm asserting the positive is an arm nothing could make green. **Its arithmetic and its evidence belong to the round's next commit**, which documents it in full; nothing else about it is written here.

#### What this commit moves, and what was re-taken while these notes were written

**ONE FILE MOVES AND IT IS A TEST**, so no prototype, no locale key and no golden moves with it: `test/datastage-goldens.json` is untouched, the two hashed arms are the arms they were, and the only thing about the gate that changed is its ARM COUNT, twenty-one where it was twenty, `2 golden + 16 variant + 1 speed + 2 merge`. **WHAT WAS RE-TAKEN HERE, said rather than left to be assumed.** The arm count and the DLC row were re-read off the script's own constants while this subsection was being written (`python3` importing `test/check-datastage.py` and printing `len(ARMS)`, the five variant tables and `DLC`); the fixture's two Lua files were regenerated through `build_remover` into a scratch directory, which is where the 17 and the 65 come from; space-age's line 241, the six base recipes that consume the belt and the five base technologies that unlock them were re-read off the installed 2.0.77 data files; and the stand-in's recipe, its ingredient and its technology were re-read off `guest/go/obs/bb2data/main.go`. **EVERYTHING WITH AN ENGINE IN IT IS THE COMMIT'S OWN RUN AND WAS NOT RE-TAKEN**: the two dumps behind the before-and-after, the swept counts, the seven keys and every red proof. They are this round's measurements, re-taken once by its adversarial review, and this subsection reports them rather than reproducing them.

#### What is still owed

- **The library defect above**, which is the round's next commit and is the first thing in this round that this mod cannot fix in its own tree.
- **The round's close.** The package sizes with the growth attributed rather than assumed, the relay's jump figures out of `dist/.fklua-mod.report.json`, the release-to-head proof re-run over its 28 pairs, and every gate consolidated at its exit code.
- **`make datastage-check`'s WALL TIME at twenty-one arms is NOT RE-TAKEN in this subsection**, and it is owed to the close. The last figure anywhere is the re-adoption leg's nineteen-arm pair, real 41.30 and 46.96 on 2026-09-13; two more engine runs are two more of the gate's roughly three seconds each, but that is arithmetic and not a measurement and is not written down as one.
- **The client run**, which is not this round's to close and which this commit does not touch.

## The FkLua baseline

The migration was measured against a freshly rebuilt fklua so that a packaging difference could not be mistaken for a library effect.

| | |
|---|---|
| fklua rebuilt from | FkLua head `32eb628776d21e0d602ace70d173f0380b169238`, `vcs.time=2026-09-01T19:46:15Z`, **`vcs.modified=false` confirmed** |
| `fklua gen-bindings --check` | up to date, committed `guest/go/fkapi/fkapi.go` unmoved |
| `fklua lock --check` | up to date, **api pin stays 2.1.17** |
| `make check` | exit 0, six pure packages ok, three `go vet` passes clean, gofmt clean |
| `make mod` / `make zip` | exit 0; zip **583,756 B** |
| `make datastage-check` | exit 0, **all eleven arms ok** on Factorio 2.0.77 |
| working tree after | clean, including `--untracked-files=all` |

**The only packaging change the fklua rebuild made is the shim.** `fk_data.lua` went 26,502 to 28,169 B (+1,667), and the diff is **purely additive**: 53 lines, 30 added and 2 removed, in two hunks. A header comment retitled from `THE SEVEN IMPORTS` to `THE EIGHT IMPORTS` with a six-line description of the new one, and a new entry appended to the import table:

```lua
raise = function(ptr, len) fail(guest_string(ptr, len)) end,
```

That import is `fkdata.raise`, and it exists because of FkRecipes: its own notes grade "a data guest cannot raise its own message" as an ecosystem gap and record it as resolved upstream and adopted. No existing import's body, order or semantics moved and none was removed. **Every other generated stage file is byte-identical** across the two fklua builds: `settings.lua` (216 B), `data.lua` (208 B), `data-final-fixes.lua` (232 B), `control.lua` (122,071 B) and `fk_data_module.lua` (1,763,788 B). `fk_module.lua` moved 3,146,816 to 3,148,568 B (+1,752).

**Two new files appear in the package** and are the packager's, not the mod's: `fk_module.map.json` (31,185 B) and `fk_data_module.map.json` (12,188 B), reported as `debug map: fk_module.map.json (143 function(s), 133 with source lines), fk_data_module.map.json (58 function(s), 55 with source lines)`.

Two observations recorded so a later pass does not read them as migration effects. `fk_api_gen.lua` came out 23,843 B against the preserved 0.3.1 package's 73,476, and that is **not attributable to the fklua rebuild alone**: the preserved package predates head commit `cf5a78e`, which regenerated the bindings and restored the event-table prune, so two variables moved between the two packages. And `gen-bindings --check` reports `4865 members bound, 5 deferred` where CLAUDE.md records 4,859 at the same pin; both checks nonetheless say the committed bindings and this fklua agree, so the prose figure is simply older than the regeneration.

## The acceptance criterion

`docs/migration.md` states it and this mod adopts it verbatim: "the acceptance criterion for the migration is BetterBeltBalancer's own `mod_settings_sha256` golden: the settings prototypes have to hash to what they hashed to before." BBB holds the `data_raw_sha256` golden to the same standard, because a settings migration that moved a prototype would be a settings migration that did something else as well.

Measured in experiment C, the settings-only shape, on Factorio 2.0.77:

```
  ok   base data_raw_sha256 f4fbcaa603abc93b
  ok   base mod_settings_sha256 196275f867f7f8b5
  ok   incumbent data_raw_sha256 e7001bf98d6c6771
  ok   incumbent mod_settings_sha256 196275f867f7f8b5
check-datastage: ok
```

**Both hashes unmoved on both mod sets, and the settings dump byte-identical after `jq -S`.** All **eleven arms** are green: the two hashed mod sets, the eight variant arms (six recipe values including `recipe-vanilla`, two technology values), and the speed arm, which still derives `all four hidden prototypes at 0.5, from an underground belt`. Every variant arm reports the ingredient list or the research unit it always did, including `recipe-cheap [('iron-plate', 2), ('transport-belt', 1)]` and `tech-logistics-3 300 x 15s, after logistics-3`. The anti-vacuity property those arms have always had is unchanged and is what makes them evidence here: each drives the real setting through a `mod-settings.dat` the gate writes, so a settings prototype that had quietly stopped being read would answer with the default and fail the equality.

The pure-host half, experiment E: `PlanSettings` against a stub World produces exactly **2 OpExtend**, and their fields are what the shipped mod's settings dump carries.

```
{type="string-setting", name="bbb-recipe-cost", setting_type="startup", default_value="vanilla", order="a", allowed_values=[vanilla, cheap, belt-fast, belt-express, splitter, splitter-express]}
{type="string-setting", name="bbb-tech-cost", setting_type="startup", default_value="logistics", order="b", allowed_values=[logistics, logistics-2, logistics-3]}
```

And the locale check: `CheckLocaleWith("better-belt-balancer", cfg, ["bbb-multi-edge-parts"])` returns **0 findings** against the `.cfg` this mod already ships. **Red-proven** by deleting one `[string-mod-setting]` entry, which reports it by name:

```
the dropdown setting bbb-recipe-cost has no [string-mod-setting] entry for its value cheap
```

That is the one direction no gate in this repository could previously cover: `--dump-data` does not read locale and no suite opens a menu, so the per-value entries a dropdown needs and has no fallback for were checked only by `guest/go/tune`'s own hand-written test. The library's check is strictly wider (it also catches orphans and duplicate keys), and BBB keeps its own assertions beside it rather than instead of it.

## What it costs

**These are the scratch-clone figures**, measured on the `exp-c` branch of a clone at `scratchpad/exp/x/BetterBeltBalancer`, Factorio 2.0.77, fklua at head `32eb628`, TinyGo 0.41.1. They are what the settings-only shape cost in that clone; **the real-tree figures are in the table after it**, recorded by the implementing pass against the shipping tree from a clean rebuild on both sides, same machine, same fklua.

| | A, clone baseline | C, settings-only | |
|---|--:|--:|---|
| `dist/bbbdata.wasm` | 174,305 | **547,401** | **+214.0%** |
| `fk_data_module.lua` | 1,763,788 | **3,216,406** | **+82.4%** |
| the zip | 583,995 | **666,315** | **+14.1%** |
| `dist/bbb.wasm` (the control guest) | 1,304,631 | 1,304,631 | **unmoved** |
| `fk_module.lua` | 3,148,568 | 3,148,568 | **unmoved** |
| `fk_data.lua` (the shim) | 28,169 | 28,169 | unmoved |
| `settings.lua` / `data.lua` / `data-final-fixes.lua` | 216 / 208 / 232 | 216 / 208 / 232 | unmoved |

The control guest being unmoved to the byte is the expected shape and is worth stating: this is a load-time change and nothing outside `guest/go/data` and `guest/go/tune` is touched, so `fk_module.lua` comes out the same file, exactly as the data-stage port and the 0.3.1 cost feature both did.

The packager's own figures moved with it: the data module went `1763529 bytes of Lua` to `3216147 bytes of Lua`, its debug map from `58 function(s), 55 with source lines` to `85 function(s), 72 with source lines`, and the NaN-sensitive operation count from 64 to 80, the new ones naming `(*github.com/Techrocket9/fkrecipes/go.Lib).PlanData` (33 `f64.load`, 14 `f64.store`) and `(*github.com/Techrocket9/fkrecipes/go.resolution).readDropdown`, all `reached from export "fk_settings"`.

**The real tree, measured either side of the migration from `make clean`** (Factorio 2.0.77, fklua at `32eb628`, TinyGo 0.41.1, 2026-09-01):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.2.zip` | 583,756 | **665,986** | **+14.1%** |
| `fk_data_module.lua` | 1,763,788 | **3,216,828** | **+82.4%** |
| `dist/bbbdata.wasm` | 173,335 | **545,651** | **+214.8%** |
| `dist/bbb.wasm` | 1,302,885 | 1,302,885 | byte-identical |
| `fk_module.lua` | 3,148,568 | 3,148,568 | byte-identical |
| members / events / defines | 56 / 24 / 4 | 56 / 24 / 4 | unmoved |
| data-module functions in the debug map | 58 | 85 | |
| longest function, emitted Lua lines | `main.hidden` 3,413 | `(*fkrecipes.Lib).PlanData` **21,047** | |
| `main.settings` | 770 | **5,335** | the Emit call and what inlined into it |

No jump-span advisory, no size advisory, no `control structure too long`, no pruning complaint in the packager report. The packager's NaN report is what says the data planner is linked: `f64.load in "(*github.com/Techrocket9/fkrecipes/go.Lib).PlanData" (33 times), reached from export "fk_settings"`.

**Stage timing**, five interleaved `--dump-data` runs per arm over packages differing only in `fk_data_module.lua`, medians of the engine's own log timestamps:

| stage | before | after | delta |
|---|--:|--:|--:|
| `settings.lua` | 0.036 s | 0.068 s | +0.032 |
| `data.lua` | 0.052 s | 0.086 s | +0.034 |
| `data-final-fixes.lua` | 0.063 s | 0.100 s | +0.037 |
| settings start to prototype checksum | 0.297 s | 0.366 s | +0.069 |

The module is `require`d once per stage and there are three, so an 82% bigger module is parsed three times per game load: about a tenth of a second, load-time only. Nothing on any tick moved and nothing in the control guest moved by a byte.

**The one number worth reading twice is `PlanData`: 21,046 emitted Lua lines in the scratch clone and 21,047 in the real tree, 29% of the whole data module.** For scale, this mod's own largest function in that module is `main.hidden` at **3,413 lines**, and the function that once made the parser refuse the file, at `-opt=2` with the `//go:noinline` marks removed, was **28,139 lines** and died on `control structure too long near 'trap_unreachable'`. `PlanData` is three quarters of the way to that wall in one function, and **the consumer cannot break it up**: it is the library's function, the `//go:noinline` remedy this repository already relies on is a mark on somebody else's source, and the whole planner is reachable from `fk_settings` because `Emit` reads the stage at runtime rather than at compile time. No jump-span advisory fired in this build and the module parsed and loaded, so this is a margin note rather than a defect. It is the reason `guest/go/data/main.go`'s `//go:noinline` block is not a curiosity.

Stage timings from the kept dump log (experiment C, `C-dump.log`): settings 0.044 to 0.101, data 0.194 to 0.271, final-fixes 0.278 to 0.363. **The settings stage's log carries no `fkrecipes:` line and no `fklua:` line**, which is the quiet outcome: the library refuses loudly and says nothing when it has nothing to say.

## The full-library shape, measured and refused

Experiments D and D2 moved the item, the recipe and the technology through the library as well, which is step 2 of `docs/migration.md`'s incremental path. Both were built and run; the shape is refused on measurement rather than on taste.

**D packaged cleanly and would not load.** `make mod` exited 0. `--dump-data` exited 1:

```
   0.466 Error Util.cpp:81: Error in assignID: item with name 'bbb-balancer-part' does not exist.

Source: bbb-balancer-part (simple-entity-with-force).
```

The library had emitted the item as `better-belt-balancer-bbb-balancer-part`, and this mod's `bbb-balancer-part` **entity** still names the unprefixed item in the fields that bind an entity to the thing that places and mines it. The entity is not the library's to rename, so the two halves of one machine came apart at the prefix.

**Four references were repointed and a fifth was tried and put back, and the put-back is a measurement of its own.** The four name the ITEM: `entity.go`'s `minable.result` and `placeable_by.item`, and the same two on `legacy.go`'s stub entity. The fifth, the legacy stub ITEM's `place_result`, names the ENTITY, whose name the library never touched, and moving it produced the engine's other refusal, from a `--dump-data` run in the measuring pass (the line is in that pass's transcript rather than in a kept log):

```
Error in assignID: entity with name 'better-belt-balancer-bbb-balancer-part' does not exist.

Source: balancer-part (item).
```

So the prefix has to be applied to exactly the references that name a library-emitted prototype and to no other, by hand, across files the library never sees, which is the shape of a rename done half by a library and half by its consumer.

**D2 repointed those entity references at the prefixed name and it loads.** That is what makes the field-by-field comparison possible, and it is where the refusal is decided. `mod_settings_sha256` is **unmoved** on both arms, at `196275f867f7f8b5`, which is the good news and is the whole reason the settings half shipped. `data_raw_sha256` **moves on both**:

```
FAIL base data_raw_sha256:
  golden f4fbcaa603abc93b96c025f7af5b280d685395462d040c77cece6192cc8ff12a
  got    55977bc9d77ec12638600ea3f1d8c60d4f61d962e7b07ded79b78824be831a8a
  note base: prototype list checksum moved 790230733 -> 3444812904 (a prototype appeared or vanished)
FAIL incumbent data_raw_sha256:
  golden e7001bf98d6c6771b75be53361fda6d52587cac10cd9a7ba035a2a8261de4edf
  got    ce081d0ee07e8d7c96d698c35a4e20f6fd41d71f576e5d80a8d096b4c5297a66
  note incumbent: prototype list checksum moved 223071962 -> 2744125292 (a prototype appeared or vanished)
```

The prototype-list checksum moving is the sharpest single statement available: a prototype really did appear or vanish, because three of them are under new names.

| | shipped | through the library |
|---|---|---|
| item | `bbb-balancer-part` | `better-belt-balancer-bbb-balancer-part` |
| recipe | `bbb-balancer-part` | `better-belt-balancer-bbb-balancer-part` |
| technology | `bbb-balancer` | `better-belt-balancer-bbb-balancer` |

**Field by field, read out of the D2 dump, four things are DROPPED and nothing is added**: the item's `order` (`c[splitter]-y[bbb-balancer]`), the item's `place_result` (`bbb-balancer-part`), the recipe's `order` (`c[splitter]-y[bbb-balancer]`) and the technology's `order` (`a-b-bbb`). Everything else crosses intact: the `unit` is byte-equal (count 20, time 15, `automation-science-pack` 1), the ingredients are byte-equal (`iron-plate` 4, `iron-gear-wheel` 2, `transport-belt` 2), the prerequisites are equal (`["logistics"]`), and `enabled` is false, `energy_required` 1, `stack_size` 50, `subgroup` belt, the icon and `icon_size` 64 all unchanged.

**The dropped `place_result` is the half that cannot be repaired by repointing anything.** An item with no `place_result` does not place the balancer part, so even with the entity's own references corrected the player is left holding an item that builds nothing. There is no `PlaceResult` on `ItemSpec` to set, so the only route back is to keep the item hand-rolled, which then means keeping the recipe hand-rolled too: `Recipe` takes an `ItemRef` and can only produce an item the same plan declares.

**And the repository's own gate cannot survive the rename.** Every variant arm looks its prototypes up under the shipped names, so with them gone `check-datastage.py` crashes rather than failing:

```
  File ".../test/check-datastage.py", line 519, in ingredients_of
    return [(i["name"], i["amount"]) for i in got]
TypeError: 'NoneType' object is not iterable
```

**None of this is the library being wrong, and the honest half of the finding is that its two hard verbs both work.** Driven by hand against the renamed prototypes, `IngredientsBy` and `CostBy` produce exactly what this mod's own resolver produces: `cheap` gives `[iron-plate 2, transport-belt 1]`, and `logistics-3` gives a prerequisite of `[logistics-3]` and a `unit` equal to base's own `logistics-3` unit in the same dump. The two behaviours 0.3.1 was built for are in the library and they are correct. What is missing is a way to reach them without renaming three prototypes and dropping four fields.

D's package sizes, for the record: `bbbdata.wasm` 536,620, `fk_data_module.lua` 2,953,403, zip 652,767. Slightly smaller than the settings-only C build, because the hand-rolled recipe and technology code left the guest as the library's arrived.

## FkRecipes friction, graded

Each entry says what was tried, what happened verbatim, and the grade: CLEAN (worked as documented), AWKWARD (worked but cost something), MISLED (pointed the wrong way), BLOCKED (needed a workaround or could not be done).

### Prototype names are always prefixed, with no Legacy form: BLOCKED

`docs/migration.md` gives four `Legacy` constructors for **settings** and there is no counterpart for items, recipes or technologies. `go/data.go` computes `prefix := modName + "-"` once and applies it to `itemProto`, `recipeProto` and `techProto` unconditionally. Measured through a real engine, that turns `bbb-balancer-part` into `better-belt-balancer-bbb-balancer-part` and `bbb-balancer` into `better-belt-balancer-bbb-balancer`, moves `data_raw_sha256` on both mod sets, moves the engine's own prototype-list checksum, and breaks this mod's entity, which names the item it is placed by. The README states the design in one line under "What it does not do": "No unprefixed names, and no prefix parameter to get wrong." That is a good rule for a new mod and it is the wall for an old one. **Prototype names are as unrenameable as setting names, for a different reason**: a blueprint, a logistic request, a recipe in a player's crafting queue and another mod's compatibility patch all name a prototype by string, and unlike a startup setting there is no single dump that says whether anything is holding one.

### No `order` slot on `ItemSpec`, `RecipeSpec` or `TechSpec`: BLOCKED

The three spec structs carry `Icon`, `IconSize`, `StackSize`, `Subgroup`, `DisplayName`, `Description`, `Name`, `CraftTime`, `Ingredients`, `ResultCount`, `Category`, `CostOf`, `Unit`, `CostBy`, `After`, `AfterTech`, `Before`, `Unlocks` and `EnabledBy`, and not `Order`. Measured in D2, three `order` strings are simply absent from the dump: `c[splitter]-y[bbb-balancer]` on the item, the same on the recipe, and `a-b-bbb` on the technology. The library derives setting order from declaration order and takes an explicit order for a legacy setting, so the reasoning that produced the legacy setting parameter ("a generated order would reshuffle the settings screen for existing players") applies unchanged to a prototype: an item's `order` is its position in the crafting menu and a technology's is its position in the research tree, and this mod chose both deliberately, `c[splitter]-y[...]` so a balancer part sits with the splitters.

### No `place_result` slot on `ItemSpec`: BLOCKED

Same struct, same absence. In D2 the emitted item carries `icon`, `icon_size`, `name`, `stack_size`, `subgroup` and `type` and nothing else, so the item does not place the entity. This one has no workaround at all short of a second `data:extend` patching the library's own prototype, which is a shape the library refuses on principle elsewhere. For a mod whose item exists **only** to place an entity, this is the field that decides whether the item can be declared here.

### `Recipe` needs an `ItemRef` from the same plan: BLOCKED, and it is the same wall

`func (l *Lib) Recipe(result ItemRef, spec RecipeSpec) RecipeRef` takes a handle, and `docs/usage.md` says why: "`IngredientOf` (`Ingredient::of`) names an item your own plan declares. It always resolves, because the same plan emits the prototype." That is right for the ingredient direction and it is what closes the door on the result direction: a recipe cannot be declared here unless its item is, and the item cannot be declared here because of `place_result`, so the recipe cannot be declared here, so `IngredientsBy` (which is what this mod would have adopted the library for) is out of reach. **The three refusals above are one refusal seen from three sides.**

### The emit layer's build tag hides `Emit` from a standard-toolchain `go vet`: BLOCKED, with a one-flag workaround

`go/guest.go` opens with `//go:build tinygo.wasm`, for a stated and correct reason: "the only code in this library that touches fkdata, gated so the pure half stays host-testable (a `//go:wasmimport` is rejected off-target)". The consumer's cost is that `Emit` does not exist under any host toolchain, and `make check` vets the data guest with the ordinary Go tool under `GOOS=wasip1 GOARCH=wasm`, which does **not** set that tag:

```
cd guest/go && GOOS=wasip1 GOARCH=wasm go vet ./data/
# github.com/Techrocket9/BetterBeltBalancer/guest/go/data
# [github.com/Techrocket9/BetterBeltBalancer/guest/go/data]
vet: data/settings.go:122:16: recipesPlan().Emit undefined (type *fkrecipes.Lib has no field or method Emit)
make: *** [check] Error 1
```

`make check` exits 2 on that. The fix is one flag, `go vet -tags tinygo.wasm ./data/`, which exits 0, and it is adopted. It is graded BLOCKED rather than AWKWARD because the failure is a **compile error naming a method the consumer can see in the documentation**, in the one gate that exists to catch compile errors, on a guest that builds perfectly under TinyGo: a reader's first reading is that they mistyped the call or pinned the wrong version. Two lines in the README's Go quickstart ("`Emit` lives behind `//go:build tinygo.wasm`; a host-toolchain vet of your data guest needs `-tags tinygo.wasm`") would remove that entirely. The build tag itself is not the complaint; the silence about it is.

### `PlanData` is linked into a settings-only guest and lowers to a 21,047-line Lua function: AWKWARD

`Emit` reads the stage at runtime and dispatches to `PlanSettings` or `PlanData`, so a plan that declares only settings still carries the whole data planner. The packager sees it and says so, naming an export this mod's settings stage really does have:

```
  f64.load in "(*github.com/Techrocket9/fkrecipes/go.Lib).PlanData" (33 times), reached from export "fk_settings":
  f64.store in "(*github.com/Techrocket9/fkrecipes/go.Lib).PlanData" (14 times), reached from export "fk_settings":
```

Lowered, `PlanData` is **21,047 lines of Lua in the real tree (21,046 in the scratch clone), 29% of the whole data module**, against this mod's own largest function at 3,413 lines and against the 28,139-line function that once made Lua 5.2's parser refuse the file with `control structure too long near 'trap_unreachable'`. Nothing failed here (no jump-span advisory, the module parsed, the goldens are unmoved), and the consumer has no remedy available: the `//go:noinline` discipline `guest/go/data/main.go` carries cannot be applied to somebody else's package, and the dead half cannot be dead-code-eliminated because the dispatch is a runtime branch. Two shapes would fix it and both are the library's to choose: split the entry point so a consumer can call `EmitSettings` and `EmitData` directly (which the stage-refusal machinery already knows how to talk about), or mark the planner's own long functions. Graded AWKWARD rather than BLOCKED because it costs bytes and margin, not correctness.

### The flash-cost figure understates what a Go consumer ships: MISLED

The README says: "A minimal guest that declares one item and one recipe through this library is 370,859 bytes of wasm; the same guest built against `fkdata` alone is 42,285 bytes. The library therefore adds about 320 KiB." Measured here on a real consumer, `dist/bbbdata.wasm` went 174,305 to 547,401 B, **+373,096 B (364 KiB)**, which is in family with that figure and slightly above it. The number a player actually pays is the Lua, and it went **+1,452,618 B**, from 1,763,788 to 3,216,406, an 82.4% increase in the data module for two settings. The README already carries the correction two paragraphs later, under "Wasm size is not a proxy for what a player downloads", and it is measured there on Go against Rust rather than on library against no library. Graded MISLED because "about 320 KiB" is the sentence a consumer budgets against and the number they pay is four times it. **One line saying what the library costs in packaged Lua, for a Go consumer, would be the honest headline**, and this mod's figures are offered as the first data point.

### `docs/migration.md`'s incremental path promises a step 2 that a prototype cannot take: MISLED

Two sentences. The document's own framing: "This document is about the surfaces that let an existing mod adopt the library without changing anything a player can observe: setting names it keeps verbatim, ingredients a dropdown chooses, and a research cost a dropdown chooses." And step 2 of "Migrating incrementally": "Move the recipes and technologies across, using `IngredientsBy` and `CostBy` where a setting was driving a hand-rolled branch."

Both are true of the **settings** and false of the **prototypes**, and the document does not distinguish. Moving the recipes and technologies across is exactly what experiment D did, and it changed three prototype names, dropped four fields, moved `data_raw_sha256` on both mod sets, moved the engine's prototype-list checksum, crashed this mod's own gate, and (in D, before the entity references were repointed) failed to load at all. The first sentence's promise is kept by the four Legacy constructors and by nothing else; the prefix is the observable change, and a prototype name is more observable than a setting name, not less. **Both sentences need the qualification the worked example gets right**, which is that this mod's names "cannot be regenerated": that is true of `bbb-balancer-part` as much as of `bbb-recipe-cost`, and the worked example says it only about the settings.

### `go mod tidy` deletes an unimported require, and the FkLua require must move to v0.2.0: AWKWARD

Two things in one dependency step, both worked, both cost a detour. Adding `require github.com/Techrocket9/fkrecipes/go v0.1.0` with a directory `replace` beside it and running `go mod tidy` **deletes the require** while nothing imports it:

```
-// EXPERIMENT ONLY, dev-only: FkRecipes is a sibling checkout with no remote and
-// no tag yet. v0.1.0 is the eventual tag the README names; the replace is what
-// actually resolves it, exactly as the fklua replace above does.
-require github.com/Techrocket9/fkrecipes/go v0.1.0
```

FkRecipes' own notes record the same window from the other side ("Do not run tidy in the placeholder window"), so this is a known shape rather than a surprise, and it closes the moment the plan file imports the library. The second half is mandatory: once imported, minimum version selection bumps `github.com/Techrocket9/fklua/guest/go` from v0.0.0 to **v0.2.0**, because that is what FkRecipes requires for `fkdata.Raise`, and `go vet` refuses to run until `go mod tidy` has written it. There is no `go.sum` anywhere in this arrangement, both dependencies being directory replaces. The README's require block already shows `guest/go v0.2.0`; a consumer reading only the quickstart snippet gets it right, and a consumer who already had a working `go.mod` gets a vet refusal first.

### The settings-only plan shape is undocumented: AWKWARD

`docs/usage.md` is emphatic about routing: "**Route each plan's `Emit` into `fk_settings` and exactly one data-family hook.**" The quickstart, the "shape of a mod" section and both language examples all show both hooks. A plan that declares only settings has no data-family hook to route into, and this mod's does not have one. **It works and it is silent**: `make mod` packages it with no advisory, the load carries no `fkrecipes:` or `fklua:` line at any stage, and all eleven dump arms pass. But the reader of that sentence has to decide for themselves whether "exactly one" means "at least one", and the surrounding text ("the settings stage needs the recipes in order to know which double setting backs a crafting time, and the data stage needs the settings in order to read them back") reads as though both halves are always required. One sentence saying a settings-only plan routes `fk_settings` alone would close it. This is the shape a migration reaches **first**, by step 1 of the incremental path, so it is the first thing an adopting mod does and the one shape the document does not show.

### `CheckLocaleWith`, the Legacy constructors, and `IngredientsBy`/`CostBy`: CLEAN

All three worked exactly as documented, first try, and they are the reason this migration happened at all.

`LegacyDropdownSettingNeedingLocale` emitted both names and both orders verbatim: `PlanSettings` against a stub World produces two `OpExtend` carrying `bbb-recipe-cost` at order `a` and `bbb-tech-cost` at order `b` with their six and three allowed values, and the engine agrees, `mod_settings_sha256` coming back `196275f867f7f8b5` on both mod sets and the settings dump byte-identical after `jq -S`. The constructor name is long and it is long for a reason a consumer meets immediately: it is the one setting type with a locale obligation the library cannot meet, and the name says so before the documentation does.

`CheckLocaleWith` returned 0 findings against the `.cfg` this mod already ships, and reported the injected defect by name and by value. The migration document's advice to prefer it over `CheckLocale` for this mod is correct for the reason it gives, measured here: every name this mod ships carries `bbb-` and none carries `better-belt-balancer-`, so the prefix-shaped orphan scan would have seen nothing.

`IngredientsBy` and `CostBy` produced the right answers when driven by hand in D2: `cheap` gives `[iron-plate 2, transport-belt 1]`, and `logistics-3` gives a prerequisite of `[logistics-3]` and a `unit` byte-equal to base's own `logistics-3` unit read out of the same dump. **The verbs are right and the naming is what keeps them out of reach**, which is worth separating: the asks at the end of this note are about names and fields, not about behaviour.

### The library exports no locale reader: AWKWARD, minor

`CheckLocale` and `CheckLocaleWith` are the only exported locale surface and both return `[]string` findings. A consumer with assertions of its own has to parse the `.cfg` again: `guest/go/tune/locale_test.go` carries a `localeSections` INI parser (sections, `key=value`, comments, duplicate detection) for three checks the library deliberately does not make (non-blank descriptions, the hand-rolled bool's own name entry, and one of this mod's message strings being a prefix of a setting's label). Exporting the parsed map, or a `ParseLocale(cfg) map[string]map[string]string`, would let a consumer's extra assertions run on the same reading the checker used. Minor, and the duplicate parser is thirty lines.

### The recipe-fallback log line differs from this mod's, and it is moot: noted

The two fallback sentences differ. The library writes the first; this mod writes the second, with the chosen option name in backticks:

```
fkrecipes: <recipe>: the <chosen> ingredients name nothing this game has, so the <default> ingredients apply
[BBB] the `<option>` balancer-part recipe names items this game does not have; falling back to the default recipe
```

The recipe stayed hand-rolled, so both lines cannot be in one build, and **no gate in this repository greps either one** (checked: zero hits in `test/check-datastage.py` and `test/run.sh`, and none anywhere under `test/`). Recorded so that a later pass which does move the recipe across knows the line changes and knows nothing is watching it, which is itself worth a second look: a fallback that fires and is seen by nobody is the shape "Measure before believing" is about.

### `World` is ten methods and `PlanSettings` asks one of them: AWKWARD, minor

A settings-only host test has to implement the whole `fkrecipes.World` interface to call `PlanSettings`, whose own header says it asks only `ModName`. `guest/go/tune/plan_test.go`'s `planWorld` is nine stub methods that are never called, kept only so the type satisfies the interface. A narrower `SettingsWorld` (`ModName` alone, embedded by `World`) would make the stub one line and would change nothing in the emit layer.

### Nothing in the Go docs names the `tinygo.wasm` build tag: MISLED, mild

The first `go vet` after wiring `Emit` fails with `vet: data/settings.go:...: ....Emit undefined (type *fkrecipes.Lib has no field or method Emit)`, which reads as a wrong method name rather than a missing build tag; the README says only "Everything except the emit layer is host-testable with plain `go test`", which is about testing rather than about type-checking a consumer. One sentence in `guest.go`'s package comment or in `docs/usage.md` saying host tooling needs `-tags tinygo.wasm` would save every consumer the same ten minutes.

### `CheckLocaleWith`'s mod name is invisible to a fully-legacy plan, and the docs say it is loud: MISLED, mild

`locale.go`'s header says a wrong `modName` "is a wrong prefix for every key at once: the result is every setting reported missing and every entry reported orphaned, which is loud rather than subtle". Measured by the review with `tune.ModName` set to `better-belt-balancer-x`: `CheckLocaleWith` returned **zero findings**, because every setting this mod ships is either legacy (the name crosses verbatim, so the prefix is never used to build a key) or hand-rolled (matched verbatim from the list), and the complete-list reading removes the prefix from the orphan scan too. For a plan like this one the parameter reaches nothing, and the only thing that catches a wrong name is BBB's own `TestModNameIsTheManifestName`, which reads `fklua.toml`. The claim is true of a plan with generated settings and silently untrue of a fully-migrated one; one sentence saying so would stop a consumer trusting the wrong test.

### The ask that would halve the cost: a static stage entry point

`Emit` dispatches on `fkdata.Stage()` at run time, so a settings-only consumer links `PlanData` and everything under it (21,047 emitted Lua lines here, 29% of the packaged data module, about a tenth of a second of parse per load across three stages) for a function that can never run. `StageKindOf` is already a pure exported function, so the seam exists; what is missing is an entry point that takes the stage the consumer already knows from which hook it is in (`EmitSettings()` and `EmitData()`, or `Emit(kind StageKind)`), which would let TinyGo drop the unreachable half. Recorded as an ask rather than a design.

## Red proofs, in the real tree

Each injected, observed, restored; `git diff --quiet -- mod-data` exit 0 afterwards and `plan.go` byte-identical to the committed version.

| injected | what fired |
|---|---|
| `bbb-recipe-cost-cheap=` deleted from the cfg | `go test ./tune/` exit 1: `the dropdown setting bbb-recipe-cost has no [string-mod-setting] entry for its value cheap` |
| `bbb-renamed-away=x` added under `[mod-setting-name]` | exit 1: `the [mod-setting-name] entry bbb-renamed-away matches no setting this plan declares`. In the same run a temporary probe showed plain `CheckLocale` returning 0 findings and `CheckLocaleWith` 1: the direction the prefix rule cannot see, measured |
| the quoted label inside `single-edge-grandfathered` changed | exit 1: `[bbb] single-edge-grandfathered tells the player to turn off "Multiple belts per part" and the row in the menu is called "Allow multiple belts per balancer part (Factorio 2.0 only)": the message names an entry that is not there` |
| the recipe setting's order `"a"` to `"c"` in plan.go | `plan_test.go: bbb-recipe-cost's order is "c" and shipped as "a"`, AND `make datastage-check` exit 2 with `FAIL base mod_settings_sha256` and `FAIL incumbent mod_settings_sha256` (golden `196275f8...`, got `7b69b3d0...`) while both `data_raw_sha256` arms stayed ok. That pair is what proves the golden watches the library-emitted settings |
| the tech default `"logistics"` to `"logistics-2"` in plan.go | `plan_test.go: bbb-tech-cost's default_value is "logistics-2" and shipped as "logistics"` |

## What is NOT RUN on this machine

| gate | status | reason |
|---|---|---|
| `make test`, all fourteen suites | **NOT RUN** | The packaged mod targets Factorio 2.1 with api pin 2.1.17 and the installed binary is 2.0.77. `test/run.sh` gates the packaged mod against the binary and refuses it before starting a run |
| the 2.1.16 and 2.1.17 golden rows of `datastage-check` | **NOT RUN** | The same. A golden whose engine does not match the binary is a SKIP with a message, by design |
| the `bench/` matrix | **NOT RUN** | Out of scope, and it needs exclusive use of Factorio |

**AND SINCE ROUND TWO THE 2.1 ROWS CARRY A NAMED EXPECTATION, because the copy got wider.** The library copies a source technology's WHOLE unit plus its `max_level` where the hand-rolled code copied `count`, `time` and `ingredients`. On 2.0.77 that is a distinction without a difference and it is verified rather than assumed: base's `logistics` unit is exactly those three fields with no `max_level` (`data/base/prototypes/technology.lua` and the dump agree), and the only unit keys anywhere in the whole 2.0.77 base dump are `count`, `count_formula`, `ingredients` and `time`. `LuaTechnologyPrototype`'s attribute set is identical between the pinned 2.0.77 and 2.1.17 runtime descriptions.

**What could NOT be verified here is base 2.1.17's own logistics prototypes**: there is no prototype-api cache for 2.1.17 under FkLua's api directory and no 2.1 binary on this machine. So: **if the 2.1.17 `data_raw` golden moves on the next 2.1 session, a source-technology field the verbatim copy carries is the first thing to check** -- `logistics` having gained a `max_level`, or a unit key base did not have on 2.0.77 -- before anything is recaptured. A golden recaptured over that would bake somebody else's field into this mod's technology.

The refusal is the designed gate and not a defect. Verbatim, from `test/run.sh m1`:

```
==> Factorio 2.0; staged mods are stamped for it
the built mod targets Factorio 2.1 and the binary is 2.0.
That is not a manifest token: the guest's bindings are pinned to one API and
the ABI marshals event payloads BY NAME, so a field the other series added is
written as mandatory and read as nil here.
```

**Every verdict in this note therefore rests on the 2.0.77 golden row**, which is the row this repository has always been able to take on this machine and the row `docs/migration.md` names as the acceptance criterion. Two things follow and both are owed to a session with a 2.1 binary: the fourteen suites over a package carrying the library, and the 2.1.16 and 2.1.17 hashed arms. Neither can move the settings prototypes (a settings stage does not branch on the engine for these two, unlike `bbb-multi-edge-parts`), so the expected result is unmoved hashes and unmoved suite numbers, which is exactly the claim that has to be measured rather than assumed.

## What the library would need for the prototypes to follow

**ALL FOUR OF THESE ARE ANSWERED AND THE PROTOTYPES FOLLOWED, 2026-09-01, round two.** `LegacyItem`, `LegacyRecipe` and `LegacyTechnology` are the first; `Order` on all three specs is the second; `PlaceResult` on `ItemSpec` is the third; `ResultNamed` is the fourth, and it is the one this mod turned out not to need, its recipe producing an item the same plan declares. The list is kept as it was written, because what it asked for is exactly what arrived and that is the useful record.

Asks, not designs. Each one is what would have let experiment D2 keep this mod's `data_raw_sha256` golden, and each is stated as a slot rather than as a mechanism, because the mechanism is FkRecipes' to choose.

- **A legacy-name form for prototypes**, matching the four that exist for settings: a way to declare an item, a recipe or a technology under a full name that is emitted verbatim with no prefix. The argument is the settings argument with a wider blast radius, since a prototype name is held by blueprints, logistic requests, crafting queues and other mods' patches, and no single dump says whether anything is holding one.
- **An `Order` field on `ItemSpec`, `RecipeSpec` and `TechSpec`.** Three `order` strings are dropped today, and the library already takes an explicit order for a legacy setting for the same reason.
- **A `PlaceResult` field on `ItemSpec`.** For a mod whose item exists only to place an entity, this is the field that decides whether the item can be declared through the library at all.
- **A way for a `Recipe` to produce an item the plan does not declare**, by name with a presence check, in the shape `IngredientNamed` already has for the ingredient direction. That is what would let a mod keep a hand-rolled item and still reach `IngredientsBy`, which is the verb this mod wanted and could not have.

None of these asks anything of the planner's behaviour, which is measured correct here in both of the places this mod would use it.
