# The FkRecipes migration, and what the library cost to adopt

BBB's two startup dropdowns (`bbb-recipe-cost` and `bbb-tech-cost`) are declared through [FkRecipes](https://github.com/Techrocket9/FkRecipes) since 2026-09-01, and its PROTOTYPES followed in **round two** the same day, on a library that had answered every refusal round one measured. Everything else this mod emits at the data stage stays hand-rolled, and this note records why, with the measurement behind each decision. **Read "Round two" for the current state**; everything before it is round one's record and is kept as it was measured, because the refusals it names are what the library's new surface is answering. It is also the dogfood report for the library: a graded list of every piece of friction met adopting it, in the shape FkRecipes' own `agents/implementation-notes.md` uses, so the asks land somewhere the library's maintainer can act on them.

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

**The locale tripwire is `guest/go/tune/locale_test.go`**, and it is the library's check plus three of this mod's own that the library does not make:

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

in a game whose `logistics` is present and carries a unit, so the fallback is unreachable by construction. The hand-rolled technology loaded there. It is not a new failure class -- a fallback that DID fire in such a pack emitted a unit naming a missing item, which is the engine's own assignID abort -- and there is no way to declare a `CostBy` with no fallback, a zero `UnitSpec` being refused for its count. `TestTheFallbackPackIsProbedEvenWhenUnreachable` pins the sentence, so the day the library probes lazily this repo is told.

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

#### The `CostBy` fallback is validated whether or not it is reachable: CLEAN, with a migration-doc ask

`docs/usage.md` says so plainly: "a science pack that does not exist is refused before anything is emitted". Measured here, in a game whose `logistics` is present and carries a unit, so the fallback cannot be reached:

```
fkrecipes: the technology bbb-balancer prices itself in automation-science-pack, which does not exist
```

It is the documented behaviour and it is the right one for a field that is the cost of last resort. **For a MIGRATING mod it is a load that used to succeed and now refuses**, which is the one thing `docs/migration.md` exists to warn about, and it warns about names rather than about this. A sentence under `CostBy` in that document would close it. Graded CLEAN because the library does what it says.

#### `LocaleEntries`: CLEAN, and it closes round one's AWKWARD

Round one asked for a parsed reading a consumer's own assertions could use, and got one. `guest/go/tune/locale_test.go`'s thirty-line INI reader is deleted; the checker and this mod's three extra assertions are one reading of one file now.

#### `Extra` and `ResultNamed` were not needed, which is worth saying

Every field of all three prototypes has a named slot. Nothing went through `Extra` and nothing was left unplaced, so round one's field-by-field DROPPED list is empty this round. `ResultNamed` is not used either: this recipe produces an item the same plan declares, so it takes the handle.

#### Consuming a library from a working tree is a build-graph hazard: not the library's, recorded anyway

`DATA_GUEST_SRC` listed this repository's own files, and FkRecipes resolves through a directory `replace`, so `make mod` did not relink the data guest when the library moved. The round-two baseline's `make datastage-check` was green over a package built against a library six commits older than the one on disk. The Makefile tracks `../FkRecipes/go` now. It is BBB's defect and it is written down here because it is a consequence of the arrangement the library's own README prescribes for an unpublished module, so the next consumer will meet it.

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
