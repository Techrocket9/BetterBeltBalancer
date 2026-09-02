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
	lib.LegacyDropdownSettingNeedingLocale(
		SettingRecipeCost, RecipeDefault(), RecipeOptions(), "a")
	lib.LegacyDropdownSettingNeedingLocale(
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
	lib.LegacyItem(PartName, fkrecipes.ItemSpec{
		Icon:        PartIcon,
		IconSize:    64,
		StackSize:   50,
		Subgroup:    "belt",
		Order:       PartOrder,
		PlaceResult: PartName,
	})

	return lib
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
