package tune

import (
	fkrecipes "github.com/Techrocket9/fkrecipes/go"
)

// THE SETTINGS PLAN, and it is the whole of what this mod hands to FkRecipes.
//
// FkRecipes is the shared data-stage library: a consumer declares what it wants
// in pure Go, the library plans it into an ordered stream of ops, and the emit
// layer executes that stream against fkdata. This mod's two startup dropdowns
// are declared here and nowhere else since 2026-09-01.
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
// WHY THE PLAN HOLDS SETTINGS ONLY, AND THE PROTOTYPES DID NOT MOVE
// ---------------------------------------------------------------------------
//
// The obvious next step is to declare the item, the recipe and the technology
// through the library too, and it was tried and MEASURED before it was declined
// (agents/fkrecipes-migration.md carries the run). Three things stopped it, and
// the first is fatal on its own:
//
//	Every item, recipe and technology name the library emits carries the mod
//	prefix, and there are no Legacy constructors for prototypes. `bbb-balancer-
//	part` would become `better-belt-balancer-bbb-balancer-part`, and the entity
//	this repo hand-rolls names the item in `minable.result` -- so the load dies
//	on the engine's own `Error in assignID: item with name 'bbb-balancer-part'
//	does not exist. Source: bbb-balancer-part (simple-entity-with-force).`
//
//	`ItemSpec`, `RecipeSpec` and `TechSpec` carry no `order` slot, so the item's
//	and the recipe's `c[splitter]-y[bbb-balancer]` and the technology's `a-b-bbb`
//	would be dropped -- which moves this mod's row in every crafting menu and in
//	Factoriopedia.
//
//	`ItemSpec` carries no `place_result`, which is the one field that makes the
//	item place the entity at all.
//
// So guest/go/data/recipe.go and technology.go stay hand-rolled, this package's
// [RecipePlan], [TechLadder] and [Resolve] stay the resolver behind them, and
// what crossed is the settings stage alone. Their own headers are the long form
// of what they do; this note is only about why they did not move.
//
// ---------------------------------------------------------------------------
// EMIT IS ROUTED INTO fk_settings AND NOWHERE ELSE
// ---------------------------------------------------------------------------
//
// `Emit` dispatches on the running stage: at the settings stage it plans
// settings, at any data-family stage it plans everything else. This plan
// declares no item, recipe or technology, so a data-family route would plan an
// empty stream and emit nothing -- work for no result, at a stage whose cost is
// paid by every player on every load. guest/go/data/settings.go is therefore
// the only caller, and data.go and data-final-fixes.go do not mention this
// package at all.
//
// THE PRICE OF ROUTING ONLY ONE HOOK IS THAT A PROTOTYPE DECLARED HERE WOULD BE
// SILENTLY DROPPED -- compiled into the shipped data module, paid for on every
// load, and emitted by nothing -- and no gate downstream can see that: the
// golden hashes cannot, because a prototype that is never emitted is not in the
// dump. [TestThePlanDeclaresNothingNoHookEmits] is the invariant that keeps this
// decision honest, and a pass that adds a prototype here has to route a
// data-family hook into Emit as well or fail it.
//
// ONE MEASURED CONSEQUENCE, WRITTEN DOWN BECAUSE IT LOOKS LIKE A DEFECT AND IS
// NOT: `PlanData` is LINKED INTO THE MODULE ANYWAY. The dispatch is a run-time
// branch on the stage name rather than a compile-time one, so TinyGo cannot
// prove the data half unreachable from an `Emit` call site and keeps it. It is
// the single largest function in the packaged data module and it never runs.
// The numbers are in CLAUDE.md, "The settings are declared through FkRecipes".
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
	// of this mod's that is runtime-global is hand-rolled beside the Emit call
	// for exactly that reason.
	lib.LegacyDropdownSettingNeedingLocale(
		SettingRecipeCost, RecipeDefault(), RecipeOptions(), "a")
	lib.LegacyDropdownSettingNeedingLocale(
		SettingTechCost, TechDefault(), TechOptions(), "b")

	return lib
}
