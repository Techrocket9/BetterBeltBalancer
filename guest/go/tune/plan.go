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
// text setting in round three, and everything else this mod emits at a data
// stage is still hand-rolled in guest/go/data.
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
// WHY THE CUSTOMIZER'S TEXT SETTING IS `Legacy` TOO, WHICH IS THE SECOND
// DECISION AND THE ONE THAT DEVIATES FROM THE LIBRARY'S OWN DOCUMENTATION
// ---------------------------------------------------------------------------
//
// `bbb-recipe-ingredients` has never shipped, so no stored value forces its
// hand: [fkrecipes.Lib.IngredientsSetting] would take a bare name, prefix it,
// and derive its order from the declaration index -- which is exactly what
// FkRecipes' docs/migration.md writes in the worked example for THIS MOD
// ("balancer-part-ingredients"), and what
// [fkrecipes.Lib.LegacyIngredientsSetting]'s own comment says the Legacy form is
// not for ("a name this mod ALREADY SHIPS"). It is declared Legacy anyway, and
// the reason is MEASURED IN THE LIBRARY rather than argued.
//
// `orderString(i)` (FkRecipes go/settings.go) derives a generated setting's
// order from its DECLARATION INDEX as two base-26 letters, `'a'+i/26` and
// `'a'+i%26`: index 0 is "aa", index 1 "ab", index 2 "ac". THE INDEX IS THE
// ONLY THING THAT DECIDES WHERE ONE LANDS among this mod's two legacy orders,
// "a" and "b" -- indices 0 through 25 ("aa" through "az") sort between them,
// and from index 26 ("ba") on they sort past "b", neither of which the
// consumer chose. Factorio sorts a mod's settings by `order` and then by name,
// and this plan declares nowhere near twenty-six, so every generated setting
// it could hold lands between the recipe dropdown and the research dropdown.
//
// FOR THIS ONE SETTING THAT HAPPENS TO BE WHERE IT BELONGS, and that is not the
// point. The research customizer is three more settings -- packs, count,
// seconds -- and generated orders put all three there too: recipe, ingredients,
// packs, count, seconds, research. The research dropdown would sit BELOW its own
// custom fields, and no ordering of the declarations can fix it, because the
// letters come from the index rather than from where the line is written. The
// Legacy constructor takes an EXPLICIT order, "aa" sorts between "a" and "b",
// and [TestEverySettingPrototypeIsTheOneThatShipped] is what pins the three
// order strings.
//
// THE NAME IS THE COST AND IT IS PAID DELIBERATELY. A generated one would read
// `better-belt-balancer-recipe-ingredients`, beside two rows spelled `bbb-`;
// one settings namespace with a seam in it is a thing a player can see, and
// nothing is lost by choosing the historical prefix for a name before anybody
// has stored it. The friction is the library's and is filed as such: a consumer
// holding legacy orders cannot CHOOSE where a generated setting lands relative
// to them, because the declaration index picks the letters.
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

	// Declaration order agrees with menu order here, but it is not what
	// DECIDES it: all three of these carry an explicit order string, so "a",
	// "aa" and "b" are the sort and the lines below are only written to match.
	// "a" and "b" are what the two dropdowns have always shipped; "aa" is the
	// customizer's, argued in the header. All three are STARTUP, which is not a
	// choice made here: FkRecipes emits `setting_type = "startup"` for every
	// setting it declares, because what this library exists to decide are
	// prototypes and a prototype is built before a map exists. The one setting
	// of this mod's that is runtime-global is hand-rolled in
	// guest/go/data/settings.go for exactly that reason.
	recipeCost := lib.LegacyDropdownSettingNeedingLocale(
		SettingRecipeCost, RecipeDefault(), RecipeValues(), "a")

	// THE TEXT THE PLAYER WRITES, and its declared default IS the vanilla plan.
	//
	// The library never emits that list as the setting's `default_value`: a text
	// setting ships holding the word `default`, and the word keeps meaning "this
	// mod's own list, with its ladders" across releases, because the engine
	// stores every setting's current value including untouched defaults -- a
	// rendered default would freeze a silent player's recipe at the day they
	// installed the mod (FkRecipes go/lib.go:230, IngredientsSetting). What the
	// declaration buys is two things: the word resolves to THIS list, ladders
	// and all, and the list is written out in the setting's description so the
	// player can copy it and edit it.
	//
	// BUILT THROUGH THE SAME CONVERSION [recipeChoices] USES, from
	// `RecipePlan(RecipeVanilla)`, so `default` in the field and `vanilla` in
	// the dropdown are one list rather than two transcriptions that can drift.
	recipeIngredients := lib.LegacyIngredientsSetting(
		SettingRecipeIngredients, asIngredients(RecipePlan(RecipeVanilla)), "aa")

	techCost := lib.LegacyDropdownSettingNeedingLocale(
		SettingTechCost, TechDefault(), TechOptions(), "b")

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
	// `Custom` IS THE SEVENTH VALUE AND `CustomValue` IS LEFT EMPTY, which the
	// library reads as the word `custom` -- the field exists for a mod whose
	// dropdown already ships a preset by that name, and none of the six is one.
	// On any preset the text is not read: it is parsed only to decide whether
	// the player EDITED it, and an edit under a preset draws one line saying so
	// rather than changing anything (FkRecipes go/customize.go:730,
	// noteIgnoredText).
	recipe := lib.LegacyRecipe(part, PartName, fkrecipes.RecipeSpec{
		CraftTime: 1,
		Order:     PartOrder,
		IngredientsBy: &fkrecipes.IngredientChoices{
			Setting: recipeCost,
			Choices: recipeChoices(),
			Custom:  recipeIngredients,
		},
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
	// never when a source answered" (go/data.go:598), so a game with no science
	// pack at all loads as long as a source carries a unit, and
	// [TestAnUnreachedFallbacksPackIsNeverProbed] is that half. The fallback's
	// NUMBERS are still checked eagerly, in the plan walk and with no question
	// asked of the game (go/data.go:330), which is the right split: a count of
	// zero is this mod's own mistake. What the lazy resolve MOVED rather than
	// removed is the other half -- where the fallback IS the price its packs
	// are walked as ladders, and a unit that loses every one of them is refused
	// with `fkrecipes: the technology bbb-balancer has no science pack the game
	// has; research takes at least one`, which
	// [TestAReachedFallbackWithNoPackInTheGameIsRefused] pins. That refusal is
	// the right answer and not a regression: a research with no packs is not a
	// cheap one, it is free.
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
	})

	return lib
}

// recipeChoices turns [RecipePlan] into the library's shape, one choice per
// PRESET value IN MENU ORDER.
//
// SIX AND NOT SEVEN. `custom` is a value of the dropdown with no plan behind
// it, and the library takes it as the `Custom` arm instead; a choice covering
// it as well would be refused by name ("gives custom a preset as well as a
// Custom arm"). So this walks [RecipeOptions] and the dropdown is declared
// with [RecipeValues], and the library checks the two against each other.
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
// `bbb-recipe-ingredients`, which is vanilla's plan.
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

// techChoices is the same for [TechLadder]: one choice per allowed value, whose
// sources are that option's ladder, most preferred first.
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
