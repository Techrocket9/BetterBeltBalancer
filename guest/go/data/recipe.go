package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// ONE recipe and ONE technology.
//
// The incumbent ships three tiers whose only difference is the recipe cost --
// the part behaves identically at every tier. Copying that would be copying a
// wishlist item the forum record says was never delivered (meaningful tier
// differences), so this ships the single tier and the question stays open.
//
// Cheap on purpose: this is infrastructure, not a reward. Iron plates, gears
// and belts, all available the moment `logistics` is researched.
//
// THE COST IS A CANDIDATE FOR A STARTUP SETTING and is deliberately not one
// yet. `fkdata.StartupSetting` reads a mod's own startup settings at the DATA
// stages (never at the settings stage, where they are still being declared), so
// making the ingredient amounts configurable is a settings.go declaration and a
// read here -- no new primitive and no new stage. That is a FEATURE, and this
// pass is a behaviour-preserving port; the shape is left where it lands
// naturally rather than half-built.
//
//go:noinline
func recipe() {
	fkdata.Extend(obj(
		f("type", str("recipe")),
		f("name", str("bbb-balancer-part")),
		f("enabled", no),
		f("energy_required", num(1)),
		f("ingredients", list(
			ingredient("iron-plate", 4),
			ingredient("iron-gear-wheel", 2),
			ingredient("transport-belt", 2),
		)),
		f("results", list(ingredient("bbb-balancer-part", 1))),
		f("order", str("c[splitter]-y[bbb-balancer]")),
	))
}

// ingredient is the 2.0 ItemIngredient/ItemProduct form -- the one that names
// its `type`, rather than the 1.1 positional pair. Both an ingredient and a
// result are the same three fields here, so they are the same function.
//
//go:noinline
func ingredient(name string, amount float64) fkdata.V {
	return obj(
		f("type", str("item")),
		f("name", str(name)),
		f("amount", num(amount)),
	)
}
