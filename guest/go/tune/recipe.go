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
// EMITTED. That is not a hope: [TestVanillaIsTodaysRecipe] compares it against
// a literal copy of the ingredient list that shipped, and the data-stage dump
// golden compares the whole prototype table against a hash captured before this
// setting existed. Every recorded number in CLAUDE.md was measured on a save
// built from the vanilla plan, so the default moving is the one change here
// that would invalidate the estate.

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
// An option this build does not know falls back to vanilla rather than to
// nothing. Factorio validates `allowed_values` itself, so an unknown string can
// only arrive from a mod-settings.dat written by a NEWER build of this mod that
// was then downgraded -- and the right answer to that is the recipe the mod has
// always had, not an empty one.
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

// ResolveRecipe is the whole decision: the chosen plan, resolved against the
// game, falling back to VANILLA when nothing in the chosen plan survived.
//
// "Nothing survived" is the identity case -- an option asked for a machine this
// pack does not have at all -- and vanilla is the last resort because it is the
// plan whose every rung is a thing a game with belts in it has. It is resolved
// through the same predicate, so the last resort cannot emit an unproven name
// either.
//
// The second result says whether the fallback was taken, so the caller can put
// a line in the log naming the option that could not be honoured. A player who
// picked `splitter-express` in a pack with no splitters should be able to find
// out why their recipe is plates and gears.
func ResolveRecipe(option string, present func(string) bool) (ing []Ingredient, fellBack bool) {
	ing = Resolve(RecipePlan(option), present)
	if len(ing) > 0 {
		return ing, false
	}
	if option == RecipeVanilla {
		// Vanilla resolving to nothing is a game with no iron plate in it. The
		// recipe comes out with no ingredients, which is a strange machine to
		// craft and a load that completes; the alternative is naming a
		// prototype nobody defined, which is not.
		return ing, false
	}
	return Resolve(RecipePlan(RecipeVanilla), present), true
}
