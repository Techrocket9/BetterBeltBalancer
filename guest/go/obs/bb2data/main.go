// Command belt-balancer-2 is A STAND-IN FOR BELT BALANCER 2, and only its data
// stage. It is one of the first two packages in this repository with NO CONTROL
// STAGE AT ALL: `fklua mod --data-module` with no control positional, which
// ships info.json, fk_abi.lua, fk_data.lua, fk_data_module.lua and the stage
// hook, and no control.lua, fk_module.lua or fk_api_gen.lua at all.
//
// WHY A STAND-IN AND NOT THE REAL MOD. The suite has to run from a clean
// checkout on any machine, and vendoring somebody else's mod into this
// repository is a distribution question this project does not need to answer.
// What the migration actually depends on is the PROTOTYPES -- a
// `simple-entity-with-force` named `balancer-part`, an `item` of the same name
// that places it, and a technology that dies with the mod -- and those are
// reproduced here exactly, under the real mod's own name and version so that
// `script.active_mods` and `mods[...]` see what they would really see. The real
// Belt Balancer 2 was run through the same flow by hand once, and the numbers
// are recorded in CLAUDE.md.
//
// IT HAS NO RUNTIME, deliberately, and that is a statement about the feature
// rather than about the port. The real mod's runtime moves items through a Lua
// FIFO in its own `storage`, and none of that survives its removal -- Factorio
// deletes a removed mod's `storage` with the mod. So a stand-in that balanced
// would be modelling the one thing the migration cannot recover. What matters is
// that the ENTITIES are standing and the BELTS around them are full, and both of
// those are the world's, not the mod's.
//
// # It is staged under FOUR names and there is one copy of it
//
// `guest/go/legacy.go` names four incumbents -- belt-balancer, belt-balancer-2,
// belt-balancer-3 and belt-balancer-performance -- and what differs between them
// is the NAME and nothing else. `test/run.sh`'s `mig_standin` copies this one
// package and rewrites `info.json`'s `name` and `version`, guarded by two greps,
// because a silently unrenamed copy would stage belt-balancer-2 under every name
// and pass every leg. That rewrite is anchored on `^(\s*)"name":` and `fklua
// mod` emits a two-space-indented `"name": "..."`, so the port did not move it.
//
// NONE OF BELT BALANCER 2'S ART IS USED. The entity draws the engine's own empty
// sprite; a headless run never opens a sprite file and the rigs are measured,
// not looked at. The one path this names is base's own splitter icon, which is
// what the real mod's parts are drawn with in Factoriopedia and in a hand.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

//go:wasmexport fk_data
//go:noinline
func onData() {
	fkdata.Extend(entity(), item(), recipe(), technology())
}

func main() {}

// The incumbent's part. Every field is the real mod's, and the one the migration
// reads back off a standing one is `max_health` -- the `fid` rig damages a part
// to 85 of this 170, so that a conversion which silently repaired it would be
// visible. (The other fidelity term, quality, is the engine's and not a
// prototype field.)
//
//go:noinline
func entity() fkdata.V {
	return obj(
		f("type", str("simple-entity-with-force")),
		f("name", str(part)),
		f("icon", str(icon)),
		f("icon_size", num(64)),
		f("flags", strs("placeable-neutral", "player-creation")),
		f("minable", obj(f("mining_time", num(0.1)), f("result", str(part)))),
		f("max_health", num(170)),
		f("corpse", str("splitter-remnants")),
		f("resistances", list(obj(f("type", str("fire")), f("percent", num(60))))),
		f("collision_box", box(-0.35, -0.35, 0.35, 0.35)),
		f("selection_box", box(-0.5, -0.5, 0.5, 0.5)),
		// The mask is what makes a part refuse to share a tile with a belt, which
		// is what every rig's geometry in this suite rests on.
		f("collision_mask", obj(f("layers", layers(
			"floor", "meltable", "object", "transport_belt", "water_tile",
		)))),
		f("picture", obsdata.EmptySprite()),
	)
}

// The item, and it is the incumbent's own rather than a convenience. The `mig`
// suite puts fifty of these in a steel chest and asserts across the swap that
// the stack survived and that its `place_result` flipped from `balancer-part` to
// `bbb-balancer-part` -- which is a statement about anything only if the stack
// size and the place_result start out as the real mod's.
//
//go:noinline
func item() fkdata.V {
	return obj(
		f("type", str("item")),
		f("name", str(part)),
		f("icon", str(icon)),
		f("icon_size", num(64)),
		f("subgroup", str("belt")),
		f("order", str("c[splitter]-x[balancer]")),
		f("place_result", str(part)),
		f("stack_size", num(50)),
	)
}

// The recipe exists to be what the technology unlocks. Nothing in the suite
// crafts one -- every part in every rig is created by script -- so what it is
// for is that a technology whose one effect names a recipe nobody defined is a
// load error rather than a fallback.
//
//go:noinline
func recipe() fkdata.V {
	return obj(
		f("type", str("recipe")),
		f("name", str(recipeName)),
		f("enabled", fkdata.Bool(false)),
		f("energy_required", num(3)),
		f("ingredients", list(
			ingredient("iron-gear-wheel", 20),
			ingredient("electronic-circuit", 15),
			ingredient("transport-belt", 5),
		)),
		f("results", list(ingredient(part, 1))),
		f("order", str("g[balancer]-a[balancer]")),
	)
}

// The technology is here because it is the one thing the migration has to
// REPLACE rather than preserve: it goes with the mod, and a player left holding
// fifty balancers and no way to craft a part would have been given a worse save
// than they started with. Three of the `mig` suite's legs assert
// `belt-balancer-1` present-and-unresearched before the swap and absent after
// it, and `fgone` asserts it absent in BOTH phases -- which is what makes the
// stranger's leg's technology check a statement about `bbb-balancer` alone.
//
// # The unit is read from `logistics`, and the aliasing hazard died in the port
//
// The Lua this replaces wrote `ingredients = data.raw.technology["logistics"]
// .unit.ingredients` -- a REFERENCE, not a copy, which is what Lua's `=` is. Two
// technologies then shared one ingredients array, and any later mod that touched
// this one would silently have edited base's `logistics` as well. It never bit,
// because nothing ever wrote to it. It cannot bite now, and not because anybody
// is being careful: `fkdata.Get` marshals the value across the wasm boundary, so
// what arrives is guest memory base's table has no relationship with. Same
// hazard, same removal, same reasoning as the shipped guest's own technology.go.
//
// # The unit is read ONCE, as a map, and there is deliberately no fallback
//
// Three separate deep `Get`s would be three host calls to answer one question,
// and could not tell a trigger technology -- 2.0's `research_trigger` form,
// which carries no `unit` at all -- from a unit that is merely missing a field.
// The shipped guest reads it the same way for the same reason, and then falls
// back to vanilla's own cost, which THIS MUST NOT: a stand-in more robust than
// the mod it stands in for would be modelling something false. Where `logistics`
// has no unit the three leaves are absent, `data:extend` refuses the technology
// by name, and the load stops -- which is the severity the Lua's own unguarded
// `attempt to index a nil value` had, delivered by Factorio's own validator.
//
//go:noinline
func technology() fkdata.V {
	unit, _ := fkdata.Get("technology", "logistics", "unit")
	count, _ := unit.At("count")
	ingredients, _ := unit.At("ingredients")
	time, _ := unit.At("time")

	return obj(
		f("type", str("technology")),
		f("name", str("belt-balancer-1")),
		f("icon", str(icon)),
		f("icon_size", num(64)),
		f("effects", list(obj(
			f("type", str("unlock-recipe")),
			f("recipe", str(recipeName)),
		))),
		f("prerequisites", list(str("logistics"))),
		f("unit", obj(
			f("count", count),
			f("ingredients", ingredients),
			f("time", time),
		)),
	)
}

// ingredient is the 2.0 ItemIngredient/ItemProduct form -- the one that names
// its `type` rather than the 1.1 positional pair. An ingredient and a result are
// the same three fields, so they are the same function.
//
//go:noinline
func ingredient(name string, amount float64) fkdata.V {
	return obj(f("type", str("item")), f("name", str(name)), f("amount", num(amount)))
}

const (
	// The name FOUR mods define and this one stands in for. It is also the whole
	// of what the stranger next door (obs/foreigndata) has in common with it.
	part       = "balancer-part"
	recipeName = "belt-balancer-normal-belt"
	icon       = "__base__/graphics/icons/splitter.png"
)

// The prototype-table shorthands, forwarded from obsdata under the short names
// guest/go/data/value.go uses, so that a prototype in Go reads like the
// prototype it is. One line each and inlined away; a package-level `var f =
// obsdata.F` would be a function VALUE, which is an indirect call TinyGo has no
// obligation to devirtualize.
func obj(kvs ...fkdata.KV) fkdata.V      { return obsdata.Obj(kvs...) }
func f(key string, v fkdata.V) fkdata.KV { return obsdata.F(key, v) }
func str(s string) fkdata.V              { return obsdata.Str(s) }
func num(x float64) fkdata.V             { return obsdata.Num(x) }
func list(vs ...fkdata.V) fkdata.V       { return obsdata.List(vs...) }
func strs(ss ...string) fkdata.V         { return obsdata.Strs(ss...) }
func box(a, b, c, d float64) fkdata.V    { return obsdata.Box(a, b, c, d) }
func layers(names ...string) fkdata.V    { return obsdata.Layers(names...) }
