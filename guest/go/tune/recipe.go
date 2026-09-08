package tune

// THE RECIPE COST, as six named plans behind one startup setting.
//
// Asked for on the mod portal: the vanilla recipe is iron plates, gears and
// yellow belts, which is right for a mod whose whole pitch is that a balancer
// is infrastructure -- and wrong for anybody who wants the machine to cost what
// the belts it balances cost. So the shape is a plan per opinion rather than a
// number per ingredient: six sliders would be six settings, twelve strings of
// locale and a combinatorial space nobody can test, and none of the six numbers
// would be guarded.
//
// VANILLA IS THE DEFAULT AND IT IS BYTE-IDENTICAL TO WHAT THIS MOD HAS ALWAYS
// EMITTED. That is not a hope: [TestVanillaIsTodaysRecipe] drives the plan and
// compares what it emits against a literal copy of the ingredient list that
// shipped, and the data-stage dump golden compares the whole prototype table
// against a hash captured before this setting existed. Every recorded number in
// CLAUDE.md was measured on a save built from the vanilla plan, so the default
// moving is the one change here that would invalidate the estate.
//
// WHAT THIS FILE IS SINCE ROUND TWO OF THE FkRecipes MIGRATION: the PLANS and
// nothing else. `ResolveRecipe` walked them against a caller's predicate and is
// deleted; the library walks them now, as one `IngredientChoice` per option
// built by [Plan]. See the package doc.

// The allowed values of `bbb-recipe-cost`, in menu order.
//
// THE FIRST IS THE DEFAULT, by construction rather than by a second constant --
// Factorio's `allowed_values` and `default_value` are separate fields and a
// default that is not in the list is a load error, so the two are taken from
// one place here and [RecipeDefault] is the head of it.
const (
	RecipeVanilla         = "vanilla"
	RecipeCheap           = "cheap"
	RecipeBeltFast        = "belt-fast"
	RecipeBeltExpress     = "belt-express"
	RecipeSplitter        = "splitter"
	RecipeSplitterExpress = "splitter-express"
)

// RecipeOptions is every allowed value of `bbb-recipe-cost`, in the order the
// dropdown shows them: cheapest first, then the two belt tiers, then the two
// splitter-based ones.
func RecipeOptions() []string {
	return []string{
		RecipeVanilla,
		RecipeCheap,
		RecipeBeltFast,
		RecipeBeltExpress,
		RecipeSplitter,
		RecipeSplitterExpress,
	}
}

// RecipeDefault is what the setting defaults to, and it is the head of
// [RecipeOptions] rather than a constant beside it.
func RecipeDefault() string { return RecipeOptions()[0] }

// RecipePlan is one option's ingredients, before any of them is checked against
// data.raw.
//
// # THE DEFAULT ARM IS A SHAPE GUARD AND NOT A LIVE FALLBACK
//
// It used to say that an option this build does not know "falls back to vanilla
// rather than to nothing", and that stopped being true when the library took
// over the walk. [Plan] builds ONE `IngredientChoice` per allowed value, so an
// unknown string never reaches this function at all: it reaches the library's
// own dropdown read, which since FkRecipes c7a806e REFUSES IT BY NAME rather
// than answering with no plan -- "A STORED VALUE THE DROPDOWN DOES NOT OFFER IS
// REFUSED. ... What it used to do was worse than a refusal: the choice lookup
// found no plan, the recipe came out made of nothing, and no line said so. This
// is the pilot's own finding, closed." (go/data.go:1298). The refusal names the
// setting as a player's mod-settings.dat carries it, this mod's two being
// Legacy and therefore unprefixed:
//
//	fkrecipes: bbb-recipe-cost holds "not-an-option", which is not one of its values
//
// WHAT MAKES THAT UNREACHABLE IS THE ENGINE, not this switch. Factorio
// validates `allowed_values` itself, and it does it by RESETTING rather than by
// refusing: a mod-settings.dat carrying `bbb-recipe-cost = "not-an-option"` is
// silently put back to the default before the data stage runs, with no log
// line, so the recipe comes out vanilla. A wrong TYPE is the loud case and is
// refused outright:
//
//	Error StringSetting.cpp:71: Failed to load mod mod setting
//	(bbb-recipe-cost): Value must be a string in property tree at
//	ROOT.startup.bbb-recipe-cost.value.
//
// So the arm below is what this function answers if it is ever called by
// something other than [Plan], and [TestAnUnofferedStoredValueIsRefusedByName]
// is what pins the other side of the engine's guard. Neither is a live path.
func RecipePlan(option string) []Item {
	switch option {
	case RecipeCheap:
		// The "I do not want to think about this" option: two plates and a
		// belt, craftable the moment `logistics` is.
		return []Item{
			{Ladder: []string{"iron-plate"}, Amount: 2},
			{Ladder: []string{"transport-belt", FallbackName}, Amount: 1},
		}
	case RecipeBeltFast:
		// Vanilla, with the belt tier raised. The plates and gears do not move:
		// what these two options are about is which BELT a balancer part costs.
		return []Item{
			{Ladder: []string{"iron-plate"}, Amount: 4},
			{Ladder: []string{"iron-gear-wheel", FallbackName}, Amount: 2},
			{Ladder: []string{"fast-transport-belt", "transport-belt", FallbackName}, Amount: 2},
		}
	case RecipeBeltExpress:
		return []Item{
			{Ladder: []string{"steel-plate", FallbackName}, Amount: 4},
			{Ladder: []string{"iron-gear-wheel", FallbackName}, Amount: 2},
			{Ladder: []string{"express-transport-belt", "fast-transport-belt",
				"transport-belt", FallbackName}, Amount: 2},
		}
	case RecipeSplitter:
		// The "it IS a splitter" reading: a balancer part is one splitter's
		// worth of machine, so it costs one. The plates are the plating.
		return []Item{
			{Ladder: []string{"splitter", "transport-belt", FallbackName}, Amount: 1},
			{Ladder: []string{"iron-plate"}, Amount: 2},
		}
	case RecipeSplitterExpress:
		return []Item{
			{Ladder: []string{"express-splitter", "fast-splitter", "splitter",
				"transport-belt", FallbackName}, Amount: 1},
			{Ladder: []string{"steel-plate", FallbackName}, Amount: 2},
		}
	}
	// vanilla, and every unknown string.
	//
	// The three ladders have a fallback rung each even though this is the plan
	// that must not change, and that costs nothing where the names exist: on any
	// game with `iron-gear-wheel` and `transport-belt` in it -- which is every
	// game this mod has ever been measured in -- the first rung wins and the
	// output is the literal that shipped. What the rungs buy is a pack that
	// removed one of them, where today's mod fails the load outright.
	return []Item{
		{Ladder: []string{"iron-plate"}, Amount: 4},
		{Ladder: []string{"iron-gear-wheel", FallbackName}, Amount: 2},
		{Ladder: []string{"transport-belt", FallbackName}, Amount: 2},
	}
}
