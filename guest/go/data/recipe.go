package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/tune"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// ONE recipe and ONE technology.
//
// The incumbent ships three tiers whose only difference is the recipe cost --
// the part behaves identically at every tier. Copying that would be copying a
// wishlist item the forum record says was never delivered (meaningful tier
// differences), so this ships the single tier and the question stays open.
//
// WHAT IT COSTS IS A STARTUP SETTING SINCE 0.3.1, asked for on the mod portal,
// and the DEFAULT is the recipe this mod has always emitted to the byte: 4 iron
// plates, 2 gears, 2 transport belts, one second. Every rate, every heap slope
// and every dump golden this repo records was measured on a save built from
// that, so the default moving is the one change here that would invalidate the
// estate -- and `tune.TestVanillaIsTodaysRecipe` compares it against a literal
// copy of the list that shipped rather than against the plan restated.
//
// THE PLANS AND THE FALLBACK LADDERS ARE guest/go/tune, WHICH IS PURE GO, and
// that split is the whole safety argument. A data stage that names an
// ingredient nobody defined is a hard load failure with this mod's name on it,
// inside somebody else's overhaul pack, before a prototype of theirs is read --
// and the arms where that happens are unreachable from any mod set this
// repository can install, so they belong somewhere `make check` can execute
// them. What is left here is the two things only a data stage can do: ask the
// game whether a name exists, and emit.
//
//go:noinline
func recipe() {
	option := recipeOption()
	ingredients, fellBack := tune.ResolveRecipe(option, itemExists)
	if fellBack {
		fkdata.Log("[BBB] the `" + option + "` balancer-part recipe names items " +
			"this game does not have; falling back to the default recipe")
	}
	if len(ingredients) == 0 {
		// A game with no iron plate in it. The part is craftable from nothing,
		// which is a strange machine and a load that COMPLETES -- the
		// alternative is naming a prototype nobody defined, which is not.
		fkdata.Log("[BBB] no balancer-part ingredient could be resolved against " +
			"this game's items; the recipe is emitted with none")
	}

	items := make([]fkdata.V, len(ingredients))
	for i, in := range ingredients {
		items[i] = ingredient(in.Name, in.Amount)
	}

	fkdata.Extend(obj(
		f("type", str("recipe")),
		f("name", str("bbb-balancer-part")),
		f("enabled", no),
		f("energy_required", num(1)),
		f("ingredients", list(items...)),
		f("results", list(ingredient("bbb-balancer-part", 1))),
		f("order", str("c[splitter]-y[bbb-balancer]")),
	))
}

// recipeOption is which plan the player asked for.
//
// A startup setting is readable at every DATA stage and at none of the settings
// stage, which is where it is being declared -- so this is only ever called from
// `fk_data`. An absent or non-string answer is the default, which covers a mod
// set where somebody stripped the setting and a `--dump-data` run of a build
// whose settings stage did not run.
//
//go:noinline
func recipeOption() string {
	v, ok := fkdata.StartupSetting(tune.SettingRecipeCost)
	if !ok || v.Tag != fkdata.TagString {
		return tune.RecipeDefault()
	}
	return v.String()
}

// itemExists is the PREDICATE every ingredient name is checked against, and the
// reason the ladders can be trusted: `tune.Resolve` can only return names this
// answered true for.
//
// It probes the prototype's own `name` field rather than the prototype, and
// that is a real difference at this boundary. `Get("item", "iron-plate")`
// marshals the WHOLE prototype across the wasm boundary -- every icon, every
// flag, every localised-name table -- to answer a yes/no question; a leaf is one
// string. `name` is mandatory on every prototype in data.raw, so its presence
// and the prototype's are the same fact.
//
// THE TYPE LIST IS WHAT MAKES THIS ANSWER "AN ITEM", not "a prototype". An
// ingredient names an item, and Factorio's item types are a family rather than
// one type: `iron-plate` is an `item`, `express-splitter`'s item is an `item`,
// but a modpack is perfectly entitled to make a balancer ingredient a `capsule`
// or a `module`. A name found in any of them is a name `data:extend` will
// accept in an ingredient list.
//
//go:noinline
func itemExists(name string) bool {
	if name == "" {
		return false
	}
	for _, typ := range itemTypes() {
		if _, ok := fkdata.Get(typ, name, "name"); ok {
			return true
		}
	}
	return false
}

// itemTypes is every prototype type an ITEM can be, as of the 2.1 prototype
// API: everything descending from `ItemPrototype`.
//
// Written out rather than derived because there is nothing to derive it from --
// the description that lists prototype inheritance is prototype-api.json, which
// no guest reads -- and because being wrong in the SAFE direction is cheap: a
// type left out means a ladder steps past a name that did exist and lands on the
// next rung, which is a slightly different recipe. Being wrong the other way is
// not possible: a type that does not exist in data.raw simply answers absent.
//
//go:noinline
func itemTypes() []string {
	return []string{
		// The ones an ingredient realistically is, first, so that the common
		// case is one host call.
		"item",
		"module",
		"tool",
		"ammo",
		"capsule",
		"gun",
		"armor",
		"repair-tool",
		"item-with-entity-data",
		"rail-planner",
		"space-platform-starter-pack",
		"item-with-label",
		"item-with-inventory",
		"blueprint-book",
		"item-with-tags",
		"selection-tool",
		"blueprint",
		"copy-paste-tool",
		"deconstruction-item",
		"upgrade-item",
		"spidertron-remote",
	}
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
