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
// recipe and its technology joined them in round two, and everything else this
// mod emits at a data stage is still hand-rolled in guest/go/data.
//
// ---------------------------------------------------------------------------
// WHY THE `Legacy` CONSTRUCTORS, WHICH IS THE ONE DECISION IN THIS FILE
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

	// Declaration order is menu order, and the explicit "a" and "b" are what
	// the two settings have always shipped. Both are STARTUP, which is not a
	// choice made here: FkRecipes emits `setting_type = "startup"` for every
	// setting it declares, because what this library exists to decide are
	// prototypes and a prototype is built before a map exists. The one setting
	// of this mod's that is runtime-global is hand-rolled in
	// guest/go/data/settings.go for exactly that reason.
	recipeCost := lib.LegacyDropdownSettingNeedingLocale(
		SettingRecipeCost, RecipeDefault(), RecipeOptions(), "a")
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
	recipe := lib.LegacyRecipe(part, PartName, fkrecipes.RecipeSpec{
		CraftTime:     1,
		Order:         PartOrder,
		IngredientsBy: &fkrecipes.IngredientChoices{Setting: recipeCost, Choices: recipeChoices()},
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
	// TWO BEHAVIOUR CHANGES, WRITTEN DOWN RATHER THAN DISCOVERED, and both are
	// graded in agents/fkrecipes-migration.md.
	//
	// THE FALLBACK'S SCIENCE PACK IS PROBED WHETHER OR NOT THE FALLBACK IS
	// REACHED, so a pack with a perfectly good `logistics` and no
	// `automation-science-pack` in it is refused at plan time where the
	// hand-rolled version loaded and copied logistics' own unit. A migrating
	// consumer cannot avoid it: a zero UnitSpec is refused for its count, so a
	// CostBy always carries a fallback and a fallback is always probed. Graded
	// AWKWARD -- a load that used to succeed and now refuses.
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
// allowed value IN MENU ORDER.
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
		plan := RecipePlan(option)
		ings := make([]fkrecipes.Ingredient, 0, len(plan))
		for _, item := range plan {
			// The amount crosses through int64 because that is the width the
			// library takes, and every amount in this package is a small whole
			// number -- 1, 2 or 4. [TestEveryAmountIsAWholeNumber] is what says
			// so rather than the reader.
			ings = append(ings, fkrecipes.IngredientNamed(
				int64(item.Amount), item.Ladder[0], item.Ladder[1:]...))
		}
		out = append(out, fkrecipes.IngredientChoice{Value: option, Ingredients: ings})
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
