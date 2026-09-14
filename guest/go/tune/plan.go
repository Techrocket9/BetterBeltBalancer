package tune

import (
	fkrecipes "github.com/Techrocket9/fkrecipes/go"
)

// THE PLAN, and it is the whole of what this mod hands to FkRecipes.
//
// FkRecipes is the shared data-stage library: a consumer declares what it wants
// in pure Go, the library plans it into an ordered stream of ops, and the emit
// layer executes that stream against fkdata. This mod's two startup dropdowns
// have been declared here and nowhere else since 2026-09-01; its item, its
// recipe and its technology joined them in round two, the recipe customizer's
// text setting and the research customizer's three fields in round three, and
// everything else this mod emits at a data stage is still hand-rolled in
// guest/go/data.
//
// ---------------------------------------------------------------------------
// WHY THE `Legacy` CONSTRUCTORS, WHICH IS THE FIRST DECISION IN THIS FILE
// ---------------------------------------------------------------------------
//
// Factorio persists a player's startup choices in mod-settings.dat keyed by the
// setting's NAME, and the engine has no rename mechanism: a setting that comes
// back under a different name is a NEW setting, and every player who had chosen
// a value silently gets the default instead. FkRecipes' ordinary constructors
// prefix every name with the mod's own -- `better-belt-balancer-...` -- which
// for a mod that has already shipped these two would discard what its players
// chose. So `bbb-recipe-cost` and `bbb-tech-cost` cross VERBATIM through the
// Legacy constructors, which is what those constructors exist for, and the
// order strings "a" and "b" are this mod's own for the same reason: a generated
// order would move the two rows in the menu and, worse, move the `order` field
// in the settings dump.
//
// THAT IS NOT AN ASSERTION, IT IS THE ACCEPTANCE CRITERION. The migration was
// gated on `test/check-datastage.py`, whose `mod_settings_sha256` hashes the
// whole normalised mod-settings dump: name, type, default, allowed values and
// order, on both mod sets. The hash is UNMOVED, so what the settings stage
// emits after the migration is byte-identical to what it emitted before it, and
// a save whose player had chosen `belt-express` still reads `belt-express`.
//
// ---------------------------------------------------------------------------
// WHY THE PROTOTYPES ARE HERE TOO, SINCE ROUND TWO
// ---------------------------------------------------------------------------
//
// Round one declared the settings alone and MEASURED the prototypes out
// (agents/fkrecipes-migration.md carries that run). Three things stopped them,
// and the library has answered all three:
//
//	Every item, recipe and technology name it emitted carried the mod prefix,
//	so `bbb-balancer-part` became `better-belt-balancer-bbb-balancer-part` and
//	the hand-rolled entity's `minable.result` named an item that no longer
//	existed -- a load failure, seen. [fkrecipes.Lib.LegacyItem],
//	[fkrecipes.Lib.LegacyRecipe] and [fkrecipes.Lib.LegacyTechnology] cross a
//	shipped name VERBATIM, which is the same argument the Legacy settings make
//	and a wider blast radius: a prototype name is held by blueprints, logistic
//	requests, crafting queues and other mods' patches.
//
//	`ItemSpec`, `RecipeSpec` and `TechSpec` carried no `order`, so this mod's
//	`c[splitter]-y[bbb-balancer]` and `a-b-bbb` were dropped. All three carry
//	one now.
//
//	`ItemSpec` carried no `place_result`, which is the one field that makes
//	this mod's item place anything at all. It carries one now, and it is
//	PRESENCE PROBED against the entity types -- which is what fixes the order
//	of guest/go/data/main.go's `fk_data` hook: `entity()` defines
//	`bbb-balancer-part` and `EmitData` runs after it, or the probe refuses the
//	plan by name.
//
// So the item, the recipe and the technology are declared here, and
// guest/go/data/item.go, recipe.go and technology.go are gone. What is left in
// this package is the LADDERS -- [RecipePlan] and [TechLadder] -- which are the
// data the two dropdown-driven choices are built from, and the transcriptions
// that keep the default honest.
//
// ---------------------------------------------------------------------------
// WHY THE CUSTOMIZERS' FOUR SETTINGS ARE GENERATED AND PLACED BY `OrderAfter`,
// WHICH IS THE SECOND DECISION AND THE ONE THAT MOVED IN THE SYNC PASS
// ---------------------------------------------------------------------------
//
// `recipe-ingredients`, `tech-packs`, `tech-count` and `tech-seconds` have
// never shipped, so no stored value forces their hand, and they are handed to
// [fkrecipes.Lib.IngredientsSetting], [fkrecipes.Lib.PacksSetting] and
// [fkrecipes.Lib.IntSetting] TWICE, as BARE names. Each prefixes the name with
// the mod's own and derives the order from the declaration index, so what the
// settings stage emits is
// `better-belt-balancer-recipe-ingredients` and three more of that shape --
// which is the path FkRecipes' docs/usage.md describes and its
// docs/migration.md works through, in a worked example that is literally this
// plan.
//
// ROUND THREE DECLARED ALL FOUR `Legacy`, AND THE MEASUREMENT IT RESTED ON IS
// KEPT HERE BECAUSE IT WAS A MEASUREMENT. `orderString(i)` (FkRecipes
// go/settings.go) derives a generated setting's order from its DECLARATION
// INDEX as two base-26 letters, `'a'+i/26` and `'a'+i%26`: index 0 is "aa",
// index 1 "ab", index 2 "ac". THE INDEX WAS THE ONLY THING THAT DECIDED WHERE
// ONE LANDED among this mod's two legacy orders, "a" and "b" -- indices 0
// through 25 ("aa" through "az") sort between them, and from index 26 ("ba") on
// they sort past "b", neither of which the consumer chose. Factorio sorts a
// mod's settings by `order` and then by name, and this plan declares nowhere
// near twenty-six, so every generated setting it could hold landed between the
// recipe dropdown and the research dropdown. For the recipe's text field that
// is where it belongs; for the research's three it is wrong, because the menu
// would read recipe, ingredients, packs, count, seconds, research, with THE
// RESEARCH DROPDOWN BELOW ITS OWN CUSTOMIZER FIELDS -- three rows a player reads
// before the row that decides whether any of them is live. No ordering of the
// declarations could fix it, because the letters come from the index rather
// than from where the line is written, so all four took a Legacy constructor,
// an explicit order ("aa", "ba", "bb", "bc") and a hand-written `bbb-` name.
//
// FkRecipes b2e47b6 ANSWERED THAT ASK EXACTLY, so the deviation has nothing
// left to rest on. [fkrecipes.Lib.OrderAfter] gives every generated setting
// declared after the call the named order followed by its own two letters: the
// placement becomes the consumer's and the name stays generated. Four things
// decide it, in the order they weigh:
//
//	The deviation rested on the placement ALONE, and the placement is answered.
//	Nothing else about the Legacy form was ever wanted for these four.
//
//	NOTHING HAS SHIPPED THESE NAMES, and the day 0.3.3 ships they are frozen
//	forever: mod-settings.dat is keyed by name and the engine has no rename.
//	This is the only moment the generated names cost nothing.
//
//	THIS MOD IS THE PILOT AND `OrderAfter` WAS BUILT FOR THIS PLAN. The engine
//	has never yet loaded a three-letter order from the library, and this mod's
//	dump gate is where it first meets one.
//
//	THE SEAM A READER SEES -- four rows spelled `better-belt-balancer-` beside
//	two spelled `bbb-` -- is invisible in the settings screen, which renders the
//	localised label. It appears in the library's log lines and in
//	mod-settings.dat, where every migrated mod that grows a setting will carry
//	the same seam.
//
// THE COST, STATED RATHER THAN HIDDEN. Round three's four customizer-row
// extracts in the scratchpad were dumped under the `bbb-` names and are reset
// by construction, and anybody holding an unshipped 0.3.3 build with a value
// stored in one of those four rows loses it. Nobody outside this machine does;
// round three's staging was reverted.
//
// THREE THINGS THE SETTINGS STAGE REFUSES IN A MIXED PLAN, AND NONE OF THEM IS
// REACHABLE FROM THIS PLAN'S CONSTANTS: an EMPTY order given to `OrderAfter`; a
// generated order EQUAL to a legacy one, whether or not `OrderAfter` was called;
// and a placed order that would sort PAST a legacy order extending the one it
// was placed after. Both of this mod's legacy orders are ONE LETTER, so no
// three-letter generated order can equal either, and neither "a" nor "b" is
// extended by any other legacy order of this plan's, so nothing can sort past
// one. [TestEverySettingPrototypeIsTheOneThatShipped] pins the six order
// strings and [TestTheSixOrdersSortIntoTheDeclarationOrder] the sort they make.
//
// ---------------------------------------------------------------------------
// TWO HOOKS, AND EACH NAMES THE HALF IT RUNS
// ---------------------------------------------------------------------------
//
// `Emit` reads the running stage and dispatches, which means a guest that
// calls it links BOTH planners: round one measured `PlanData` as a 21,047-line
// Lua function in a module whose plan could never reach it. [fkrecipes.Lib.EmitSettings]
// and [fkrecipes.Lib.EmitData] name the half, so the linker can drop the other
// -- and each raises if it is routed from the wrong stage, so a misroute is a
// sentence naming the hook rather than a probe failure later.
//
// guest/go/data/main.go therefore calls `EmitSettings` from `fk_settings` and
// `EmitData` from `fk_data`, and nothing calls `Emit`.
//
// ---------------------------------------------------------------------------
// WHY A FUNCTION RATHER THAN A PACKAGE-LEVEL VALUE
// ---------------------------------------------------------------------------
//
// The wasm module is instantiated FRESH per stage -- Factorio's settings stage
// and its data stages are separate Lua states with nothing carried across -- so
// there is no plan to reuse and a package-level one would only move the
// declarations from a call into an initialiser. Building it per call also keeps
// the host tests honest: [Plan] is the same function the guest calls, so a test
// that holds its output up to the light is looking at what ships.
func Plan() *fkrecipes.Lib {
	lib := fkrecipes.New()

	// Declaration order agrees with menu order here, and since the sync pass it
	// is HALF of what decides it. The two dropdowns carry the explicit orders
	// they have always shipped, "a" and "b"; the four customizer fields carry
	// orders the library derives, and where those land is decided by the
	// `OrderAfter` call in force at each declaration -- so the sort is "a",
	// "aab", "b", "bad", "bae", "baf" and the lines below are written to match
	// it. All six are STARTUP, which is not a choice made here: FkRecipes emits
	// `setting_type = "startup"` for every setting it declares, because what
	// this library exists to decide are prototypes and a prototype is built
	// before a map exists. The one setting of this mod's that is runtime-global
	// is hand-rolled in guest/go/data/settings.go for exactly that reason.
	recipeCost := lib.LegacyDropdownSettingNeedingLocale(
		SettingRecipeCost, RecipeDefault(), RecipeOptions(), "a")

	// EVERYTHING GENERATED FROM HERE SORTS UNDER THE RECIPE DROPDOWN. The call
	// names an ORDER, "a", and not a setting: what follows carries "a" and then
	// its own two letters, so it sorts after the row at "a" and before every
	// legacy order that sorts after "a" -- which is where the text field
	// belongs and, without this line, where it would happen to land anyway.
	// It is written out for the research's sake as much as its own: the two
	// calls are one decision and a plan with only the second reads like an
	// accident.
	//
	// THE CALL SITS ABOVE THE DECLARATION BECAUSE THE PREFIX IS CAPTURED AT
	// DECLARATION, not read when the plan is emitted (FkRecipes go/lib.go, the
	// OrderAfter comment). A setting declared above this line keeps the bare
	// two letters; the second call below moves what follows IT and leaves this
	// one where it is.
	lib.OrderAfter("a")

	// THE TEXT THE PLAYER WRITES, and its declared default IS the vanilla plan.
	//
	// The library never emits that list as the setting's `default_value`: a text
	// setting ships holding the word `default`, and the word keeps meaning "this
	// mod's own list, with its ladders" across releases, because the engine
	// stores every setting's current value including untouched defaults -- a
	// rendered default would freeze a silent player's recipe at the day they
	// installed the mod (FkRecipes go/lib.go, IngredientsSetting). What the
	// declaration buys is two things: the word resolves to THIS list, ladders
	// and all, and the list is written out in the setting's description so the
	// player can copy it and edit it.
	//
	// BUILT THROUGH THE SAME CONVERSION [recipeChoices] USES, from
	// `RecipePlan(RecipeVanilla)`, so `default` in the field and `vanilla` in
	// the dropdown are one list rather than two transcriptions that can drift.
	//
	// THE NAME IS BARE AND THE ORDER IS THE LIBRARY'S: this is declaration index
	// 1, so the two letters are "ab" and the prefix in force is "a", which makes
	// `better-belt-balancer-recipe-ingredients` at order "aab".
	recipeIngredients := lib.IngredientsSetting(
		RecipeIngredientsName, asIngredients(RecipePlan(RecipeVanilla)))

	techCost := lib.LegacyDropdownSettingNeedingLocale(
		SettingTechCost, TechDefault(), TechOptions(), "b")

	// AND EVERYTHING GENERATED FROM HERE SORTS UNDER THE RESEARCH DROPDOWN,
	// which is the call this whole decision was waiting for. Without it the
	// three fields below would carry the bare "ad", "ae" and "af" -- still under
	// "a", so a player would read the research dropdown AFTER the three rows it
	// switches on, and no ordering of the declarations could move them, because
	// the two letters count declaration slots rather than lines.
	//
	// A SECOND CALL MOVES ONLY WHAT FOLLOWS IT. The text field above keeps the
	// "a" it was declared under, because the prefix was captured there.
	lib.OrderAfter("b")

	// THE RESEARCH COST THE PLAYER WRITES, as the three fields a Factorio unit
	// actually has: what it is paid in, how many, and how long one takes.
	//
	// EACH FIELD IS A SWITCH OF ITS OWN AND THE TIER DECIDES THE REST. A pack
	// text on the word `default` and a number at 0 are the three ways of saying
	// "the row above decides", so a player who touches nothing is charged the
	// tier they picked, BYTE FOR BYTE, and a player who moves one slider moves
	// one field of it. That is why the two numbers ship at 0 rather than at
	// [FallbackUnit]'s 20 and 15: a declared 20 beside a tier charging 200 is
	// not a default, it is an override nobody asked for, and the library
	// refuses the declaration outright ("its declared default and its minimum
	// must both be 0 (0 means the dropdown decides)").
	//
	// THE PACK TEXT'S DECLARED DEFAULT IS STILL [FallbackUnit]'s PACK, built
	// FROM `FallbackUnit().Packs` rather than written out a second time, and
	// [TestTheThreeResearchDeclarationsAreWhatThisFileClaims] is what says the
	// three declarations are what this file claims: 0, 0 and that one pack
	// rendered into the tooltip a player copies from.
	//
	// THE PACK LIST CARRIES NO `Fallbacks` LADDER, AND THAT IS A DECISION.
	// Every other ladder in this package ends at a name every game with belts
	// has -- `iron-plate` for an ingredient, the logistics tiers for a cost --
	// and THERE IS NO SCIENCE PACK WITH THAT PROPERTY. `automation-science-pack`
	// is base's own name and an overhaul is free to remove it; no other name is
	// likelier to be present, so a second rung would be a guess dressed as a
	// ladder. What happens without one is the library's and is pinned by
	// [TestAReachedFallbackWithNoPackInTheGameIsRefused] and by
	// [TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused]: the ladder is
	// walked through `ToolExists`, a pack no rung answers for is DROPPED with a
	// line, and a unit that loses every pack is refused by name rather than
	// emitted free. THE LADDER IS REACHED THROUGH THE `Fallback`, not through
	// the text field: beside a tier an untouched pack text takes the TIER's own
	// science packs, and this list is what prices the research when the tier
	// has none left.
	//
	// WHY A MAXIMUM IS DECLARED AT ALL, since neither bound is a balance
	// opinion. The engine RESETS a stored number outside its own bounds to the
	// default rather than clamping it (measured by the library, FkRecipes
	// go/customize.go), so the declared range is exactly the set of values the
	// data stage can ever read. Beside a dropdown the MINIMUM is not this mod's
	// to choose -- it is 0, because 0 is the word `default` of a number -- and
	// the library refuses a research number with no maximum of at least 1, so
	// the ceiling is the one bound left and it is this mod's: a player who
	// types 10^12 units has not written a research, and a number the engine
	// would hand back as a reset is better than one nothing can pay.
	//
	// WHAT STOPS A 0 REACHING THE ENGINE IS THE COST BEHIND THE FIELD AND NOT
	// THE FENCE. The engine refuses a unit whose count is 0 and one whose time
	// is 0, and what the library emits for a field left at 0 is the number the
	// TIER carries or, where no source in the chosen ladder carries a unit at
	// all, the number `CostChoices.Fallback` declares
	// ([TestNoLogisticsAtAllIsTheFallbackAndNoPrerequisite] is that game, and
	// [FallbackUnit] is 20 and 15). Either way something non-zero answers, so
	// there is no arm of this plan where a 0 is written into a unit.
	//
	// THREE BARE NAMES AT DECLARATION INDICES 3, 4 AND 5, whose two letters are
	// "ad", "ae" and "af" and whose prefix is the "b" in force: the settings
	// stage emits `better-belt-balancer-tech-packs` at "bad",
	// `better-belt-balancer-tech-count` at "bae" and
	// `better-belt-balancer-tech-seconds` at "baf", each under the dropdown at
	// "b" and in the order written here.
	techPacks := lib.PacksSetting(TechPacksName, FallbackUnit().Packs)
	techCount := lib.IntSetting(TechCountName, 0, fkrecipes.Between(0, 1000000))
	techSeconds := lib.IntSetting(TechSecondsName, 0, fkrecipes.Between(0, 3600))

	// THE ITEM, and every field of it is transcribed from what shipped.
	//
	// `StackSize` is written out rather than left to the library's own default
	// of 50, which happens to be the same number: this mod's 50 is a decision
	// its dump golden pins, and a library default is a decision somebody else
	// may revisit. The two agreeing today is not a reason to stop saying which
	// one is ours.
	//
	// `PlaceResult` is the field round one could not have and the reason the
	// item could not move: this item exists only to build the balancer part,
	// and one that places nothing is not this mod's item. The library PRESENCE
	// PROBES it against the entity types, so `entity()` has to have run before
	// `EmitData` does -- see the header, and guest/go/data/main.go's `fk_data`.
	part := lib.LegacyItem(PartName, fkrecipes.ItemSpec{
		Icon:        PartIcon,
		IconSize:    64,
		StackSize:   50,
		Subgroup:    "belt",
		Order:       PartOrder,
		PlaceResult: PartName,
	})

	// THE RECIPE, whose ingredients the player chooses.
	//
	// `IngredientsBy` is the verb this mod wanted the library for and could
	// not reach in round one: one plan per dropdown value, every name a
	// LADDER, and the first rung the game actually has is what is emitted. It
	// is the same guarantee [RecipePlan]'s ladders were written to have and it
	// is the library's to keep now -- an ingredient naming a prototype nobody
	// defined is a HARD LOAD FAILURE with this mod's name on it, in somebody
	// else's overhaul pack, before a prototype of theirs is read.
	//
	// `CraftTime: 1` rather than `CraftTimeFrom`: what a balancer part costs
	// is a choice of INGREDIENTS here and one second is what shipped. The
	// library emits it as `energy_required` and omits the field entirely at
	// zero, so the 1 has to be said.
	//
	// `IngredientsFrom` SITS BESIDE `IngredientsBy` AND THE TEXT IS THE SWITCH.
	// The dropdown offers the SIX values it has always offered and the library
	// adds none of its own to it; what decides between the two rows is the text
	// field itself, which applies whenever it does not say `default` and hands
	// the choice back to the dropdown whenever it does. So the ingredient
	// override is TOTAL when it is live -- the whole list is the player's and
	// the preset is set aside with one line saying so -- and invisible when it
	// is not.
	//
	// THAT SHAPE IS WHY THE DROPDOWN'S OPTION LIST NEVER MOVED. Factorio resets
	// a stored value the running release's `allowed_values` does not list, and
	// persists the reset with no line in the log (measured by the library on
	// 2.0.77), so a `custom` row would be destroyed by one launch of an older
	// release while the typed text survived -- a field still showing the
	// player's recipe beside a dropdown that had quietly gone back to a preset.
	// Keeping the state in the TEXT makes this whole feature an identity for a
	// player who never opens the Startup tab and a no-op for a rollback.
	recipe := lib.LegacyRecipe(part, PartName, fkrecipes.RecipeSpec{
		CraftTime: 1,
		Order:     PartOrder,
		IngredientsBy: &fkrecipes.IngredientChoices{
			Setting: recipeCost,
			Choices: recipeChoices(),
		},
		IngredientsFrom: recipeIngredients,
	})

	// THE TECHNOLOGY, whose cost the player chooses.
	//
	// `CostBy` copies the chosen source's whole unit VERBATIM and makes that
	// same source the technology's sole prerequisite. THE PREREQUISITE MOVING
	// WITH THE UNIT is the rule this mod wrote by hand and the reason it is
	// worth handing over: charging `logistics-3`'s science while still hanging
	// off `logistics` would put a machine that costs blue science at a place
	// in the tree a player reaches with red -- researchable long before it is
	// affordable, and out of order in Factoriopedia.
	//
	// THE FALLBACK IS TODAY'S BEHAVIOUR, LITERALLY: base's own logistics unit,
	// 20 automation science over 15 seconds, which is what this technology has
	// cost in every save this mod has ever been in. It applies where no source
	// in the chosen ladder carries a unit -- a pack that removed the logistics
	// chain, or, far likelier since 2.0, one that turned it into a TRIGGER
	// technology -- and a technology that falls back has no prerequisite at
	// all, which is what stops `prerequisites = {"logistics"}` naming a
	// technology nobody defined.
	//
	// ONE BEHAVIOUR CHANGE IS STILL OPEN, WRITTEN DOWN RATHER THAN DISCOVERED,
	// and it is graded in agents/fkrecipes-migration.md; a second was graded
	// there and is CLOSED, which is the paragraph below.
	//
	// THE FALLBACK'S SCIENCE PACK USED TO BE PROBED WHETHER OR NOT THE FALLBACK
	// WAS REACHED, so a pack with a perfectly good `logistics` and no
	// `automation-science-pack` in it was refused at plan time where the
	// hand-rolled version loaded and copied logistics' own unit. FkRecipes
	// c7a806e answered the ask: "THE FALLBACK IS RESOLVED ONLY HERE, which is
	// the point: its packs are probed when the fallback is what applies, and
	// never when a source answered" (go/data.go, the CostBy arm), so a game
	// whose tier is priced in a pack it HAS loads with this mod's own fallback
	// pack absent, and [TestAnUnreachedFallbacksPackIsNeverProbed] is that
	// half. The fallback's NUMBERS are still checked eagerly, in the plan walk
	// and with no question asked of the game (go/customize.go,
	// validateCostChoices), which is the right split: a count of zero is this
	// mod's own mistake. What the lazy resolve MOVED rather than removed is the
	// other half -- where the fallback IS the price its packs are walked as
	// ladders, and a unit that loses every one of them is refused with
	// `fkrecipes: the technology bbb-balancer has no science pack the game has;
	// research takes at least one, and none of automation-science-pack is a
	// science pack here`, which [TestAReachedFallbackWithNoPackInTheGameIsRefused]
	// pins word for word. That refusal is the right answer and not a
	// regression: a research with no packs is not a cheap one, it is free.
	//
	// `max_level` TRAVELS WITH THE UNIT. It lives on the TECHNOLOGY rather than
	// in the unit, and `CostBy` copies the source's, where the hand-rolled
	// `researchUnit` read three fields of the unit and nothing else. In a pack
	// whose chosen source is a multi-level technology this mod's research
	// becomes multi-level too: the unlock fires at level one and the rest are
	// no-ops a player pays for. Measured on the engine with `logistics-3` given
	// a max_level of 3.
	//
	// The other direction of the same copy is an IMPROVEMENT and is why neither
	// is simply worse: a source priced by `count_formula` and no `count` used to
	// produce a unit with neither and abort the load with this mod's name on it,
	// and a verbatim copy loads.
	//
	// `CostFrom` SITS BESIDE `CostBy` AND EACH FIELD IS ITS OWN SWITCH. The
	// dropdown offers the THREE tiers it has always offered, and the three
	// settings overwrite the chosen tier's unit ONE FIELD AT A TIME: a pack
	// text on the word `default`, a count at 0 and a seconds at 0 each mean
	// "the tier decides this one". So a player who touches nothing is charged
	// the tier byte for byte, a player who types a pack list keeps the tier's
	// count and time, and nothing is ever read and then ignored -- the
	// `is edited, but ... is not on custom` line this arm used to draw has no
	// state left to be about.
	//
	// THE OPTION LIST NEVER MOVED, for the recipe dropdown's reason one level
	// up: Factorio resets a stored value the running release does not offer and
	// persists the reset silently, so a `custom` row would be destroyed by any
	// rollback while the three fields beside it survived. The state lives in
	// the fields instead, and adopting the customizer on this dropdown costs a
	// stored `logistics-3` nothing.
	//
	// AND THE TIER STILL PLACES THE TECHNOLOGY, WHICH IS WHY THERE IS NO
	// PLACEMENT FIELD HERE AT ALL. `CostBy` names a source technology on every
	// value the dropdown has, and that source is the prerequisite whatever the
	// three settings say -- so repricing the research does not move it in the
	// tree, and the cost and the place cannot come apart. The ladder this arm
	// used to carry for a value that named no source went with the value: there
	// is no arm left in which this technology has a cost and no source to hang
	// off, so there is nothing for such a ladder to decide.
	//
	// WHAT A PLAYER WHO REPRICES STILL CHANGES, said rather than hidden: the
	// count and the seconds are whole numbers, and a tier priced by a
	// `count_formula` loses that formula the moment the count setting is not 0,
	// because the engine refuses a unit carrying both (measured by the library
	// on 2.0.77). None of the three logistics tiers is priced by a formula in
	// base, so this reaches a modpack that repriced one and nothing else; the
	// library says so in the log and in the technology's own tooltip.
	lib.LegacyTechnology(TechName, fkrecipes.TechSpec{
		Icon:     PartIcon,
		IconSize: 64,
		Order:    TechOrder,
		Unlocks:  []fkrecipes.RecipeRef{recipe},
		CostBy: &fkrecipes.CostChoices{
			Setting:  techCost,
			Choices:  techChoices(),
			Fallback: FallbackUnit(),
		},
		CostFrom: &fkrecipes.CustomCost{
			Packs:   techPacks,
			Count:   techCount,
			Seconds: techSeconds,
		},
	})

	return lib
}

// recipeChoices turns [RecipePlan] into the library's shape, one choice per
// PRESET value IN MENU ORDER.
//
// SIX AND EXACTLY SIX. The library adds no value of its own to a dropdown, so
// the choice list has to cover the `allowed_values` EXACTLY and a plan that
// covered one fewer would be refused by name ("the recipe bbb-balancer-part
// offers nothing for the value <v> that the setting bbb-recipe-cost allows").
// [RecipeOptions] is therefore read here and at the declaration alike, and the
// library checks the two against each other.
//
// BUILT FROM THE LADDERS RATHER THAN BESIDE THEM. The library checks a choice
// list against its dropdown's allowed values and refuses a mismatch by name, so
// two hand-kept lists would be caught -- but caught at load, by the engine
// raising, rather than never written. [RecipeOptions] is the one list and
// [RecipePlan] is the one plan per value, exactly as they were when this
// package resolved them itself.
func recipeChoices() []fkrecipes.IngredientChoice {
	options := RecipeOptions()
	out := make([]fkrecipes.IngredientChoice, 0, len(options))
	for _, option := range options {
		out = append(out, fkrecipes.IngredientChoice{
			Value: option, Ingredients: asIngredients(RecipePlan(option))})
	}
	return out
}

// asIngredients is the ONE conversion from this package's [Item] to the
// library's ingredient, and it is one function because it has two callers that
// must not disagree: every preset's choice list, and the DECLARED DEFAULT of
// `better-belt-balancer-recipe-ingredients`, which is vanilla's plan.
//
// If those were two transcriptions, the word `default` in the text field and
// the value `vanilla` in the dropdown could come to mean different recipes, and
// nothing outside a dump would ever say so.
func asIngredients(plan []Item) []fkrecipes.Ingredient {
	out := make([]fkrecipes.Ingredient, 0, len(plan))
	for _, item := range plan {
		// The amount crosses through int64 because that is the width the
		// library takes, and every amount in this package is a small whole
		// number -- 1, 2 or 4. [TestEveryAmountIsAWholeNumber] is what says
		// so rather than the reader.
		out = append(out, fkrecipes.IngredientNamed(
			int64(item.Amount), item.Ladder[0], item.Ladder[1:]...))
	}
	return out
}

// techChoices is the same for [TechLadder]: one choice per TIER value, whose
// sources are that option's ladder, most preferred first.
//
// THREE AND EXACTLY THREE, for the reason [recipeChoices] is six and exactly
// six: the choice list covers the dropdown's `allowed_values` with nothing
// added and nothing subtracted, so [TechOptions] is read here and at the
// declaration alike.
func techChoices() []fkrecipes.CostChoice {
	options := TechOptions()
	out := make([]fkrecipes.CostChoice, 0, len(options))
	for _, option := range options {
		out = append(out, fkrecipes.CostChoice{Value: option, Sources: TechLadder(option)})
	}
	return out
}

// FallbackUnit is what this technology costs in a game whose whole logistics
// chain is missing or trigger-researched: base's own `logistics` unit, written
// out.
//
// IT IS TODAY'S BEHAVIOUR AND NOT A NEW NUMBER. 20 automation science over 15
// seconds is what `logistics` has charged since 1.0 and therefore what this
// technology has cost in every save this mod has ever been in, so a pack that
// removed the chain gets the vanilla cost rather than a broken load, and
// nobody who has not removed it can tell the difference.
//
// IT IS ALSO WHAT `better-belt-balancer-tech-packs` DECLARES, since round
// three: the pack list a player sees written out in that field's tooltip, and
// the list the word `default` in it stands for, is taken from here rather than
// written a second time. The two NUMBERS beside it declare 0, which beside a
// tier is the word `default` of a number; see the declarations in [Plan].
//
// Exported so a test can compare it against the fixture's `logistics` without
// either of them being derived from the other.
func FallbackUnit() fkrecipes.UnitSpec {
	return fkrecipes.UnitSpec{
		Count:   20,
		Seconds: 15,
		Packs:   []fkrecipes.Pack{{Name: "automation-science-pack", Amount: 1}},
	}
}

// The names and the two sort keys three files have to agree about: this plan,
// which emits them, and guest/go/data's entity.go and legacy.go, which name the
// item from an entity and the entity from an item.
//
// WRITTEN DOWN ONCE BECAUSE A MISMATCH IS A LOAD FAILURE AND NOT A WARNING.
// The engine's answer to an entity whose `minable.result` names an item nobody
// defined is `Error in assignID: item with name '...' does not exist`, with
// this mod's name on it, before a prototype of anybody else's is read -- which
// is exactly what round one measured when the library prefixed these names.
// They live in this package rather than in the data guest because the plan is
// the thing that emits them and this package is the one a host `go test` can
// reach.
const (
	// PartName is the item, the recipe and the entity, which all three share.
	// One string, because `place_result`, `minable.result` and
	// `placeable_by.item` are the three fields that bind them into one machine.
	PartName = "bbb-balancer-part"

	// TechName is the technology that unlocks the recipe.
	TechName = "bbb-balancer"

	// PartIcon is the icon every player-facing prototype of this mod uses. A
	// stale path here is a defect only the GRAPHICAL client sees -- headless
	// Factorio never opens a sprite file -- which is why test/check-sprites.py
	// walks the packaged mod for exactly this class.
	PartIcon = "__better-belt-balancer__/graphics/icons/balancer-part.png"

	// PartOrder puts the item and the recipe next to the splitters, which is
	// where a player looks for this.
	PartOrder = "c[splitter]-y[bbb-balancer]"

	// TechOrder is the technology's place in the research screen.
	TechOrder = "a-b-bbb"
)
