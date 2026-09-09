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

`costPresetText` (go/customize.go:575) renders a tier's line as `cost of <first source>`, so the research dropdown's tooltip ends `Default: Logistics, alongside transport belts: cost of logistics` and `... : cost of logistics-2`. For the recipe dropdown the internal-name rendering is right, because the player types that vocabulary into the field beside it; for a cost dropdown the player never types a technology name, so the internal name buys nothing and reads as a raw key beside a localised label. The ask: render the source as its localised name (`{"technology-name.<name>"}`), or let a consumer opt out of the composition for a cost dropdown. The composition itself, and the checker demanding the description that anchors it, are correct.

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

Closed: every one of round three's four AWKWARDs (the placement of a generated setting, the ignored number, the localised cost preset, the silent relay), so round three's ledger carries nothing open. Of round two's ledger, one entry is answered upstream and unrecorded until now (`PlaceResult`'s probe and the docs' "beside": FkRecipes docs/migration.md now says the entity is "extended before the `Emit` call runs"), and two stand as they were graded: `max_level` travelling with the copied unit, which docs/migration.md now documents with a workaround (a single-level source, or a written `Unit`) rather than the opt-out asked for, a documented decline; and the one-sentence ask for step 2 of the incremental path (that a recipe and the technology that unlocks it move together), still absent from docs/migration.md. Neither blocks this consumer. Corrected: round three's heading on the engine's int typing, and the count of customizer settings in the instruction that launched this pass (and in the FkRecipes follow-up brief it drew on), which said five where the plan declares four (the recipe text, the pack text, the count, the seconds). Open, unchanged from round three: the client run (the composed descriptions' line breaks, the text field's rendering, the labels as rendered, a custom recipe crafted), and the 2.1 golden rows, which only a 2.1 binary can re-capture. New and not this mod's to close: the subcommand probe above. The changelog entry for 0.3.3 is what round three wrote, because nothing a player sees moved this round: the four settings' labels and descriptions are byte for byte what they were, and their internal names have never shipped.

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

**WHAT THE ROUND DOES NOT FIX IS THE CLOSED DROPDOWN.** The client measured that widget truncating its label at roughly 37 characters, and TEN OF THIS MOD'S ELEVEN dropdown labels exceed it: `bbb-recipe-cost` has six of seven over (median overflow 15 characters, worst 28, only `Splitter: 1 splitter, 2 iron plates` at 35 fitting) and all four of `bbb-tech-cost`'s are over. That string is the CONSUMER'S label, which the library never composes and never sees. What the new shape buys is the tooltip: the copyable internal list opens its own indented line, so a truncated label no longer hides where it begins.

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

**TEN OF ELEVEN DROPDOWN LABELS EXCEED THE CLIENT'S MEASURED TRUNCATION AND THIS ROUND DID NOT TOUCH THEM.** The closed widget was seen truncating at roughly 37 characters, one widget at one UI scale, and the same renderer script reports the eleven labels at 35, 38, 43, 45, 46, 50, 52, 52, 54, 58 and 65 characters, only `Splitter: 1 splitter, 2 iron plates` at 35 clearly under. Commit 3 decided KEEP on a measurement and rewrote the argument rather than the labels; what is still open is the widget itself, which no string this repository owns can fix.

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
