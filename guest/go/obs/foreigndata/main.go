// Command bbb-mig-foreign is A STRANGER WHO OWNS THE NAME, and only a data
// stage. It is one of the first two packages in this repository with no control
// stage at all -- see the sibling `obs/bb2data` for what that packaging is.
//
// This mod is not Belt Balancer, not Belt Balancer 2, not Belt Balancer 3 and
// not Belt Balancer Performance. It defines `balancer-part` anyway, exactly as
// they do, and its entities must never be converted by this mod's migration.
//
// WHAT IT PROVES, and it is the guard with the real blast radius. The runtime
// decision cannot be taken from the prototype's existence, because a prototype
// says nothing about whose it is; and it cannot be taken from
// `script.active_mods` alone, because that is a list of four names and this mod
// is not on it. It is taken from `bbb-legacy-stub`, a marker prototype the
// shipped mod's data guest defines IF AND ONLY IF it also defined the stub --
// which it does not here, because this mod got there first.
//
// The `mig` suite runs it two ways. `foreign` installs the real mod beside it
// and asserts that nothing at all happened; `fgone` uninstalls the STRANGER,
// which is the promise `legacyCheck` makes in as many words -- their balancers
// become ours on that load, the same promise the incumbents get.
//
// # Why this is a separate package from the incumbent stand-in
//
// The two define almost the same prototypes: the item is identical and the
// entity differs by one field. They are not shared, and the reason is what this
// mod IS. A constructor with a `resistances bool` on it would model the stranger
// as a variant of the incumbent, which is precisely the reading the whole
// negative exists to refuse -- and it would put the one difference between them
// behind an argument instead of in front of a reader.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

//go:wasmexport fk_data
//go:noinline
func onData() {
	fkdata.Extend(entity(), item())
}

func main() {}

// The stranger's part, which is an incumbent's minus its fire resistance -- and
// there is NO recipe and NO technology, which is the other difference and the
// one a leg reads. `fgone` asserts `belt-balancer-1` absent in BOTH of its
// phases, so the only technology statement that leg makes is about
// `bbb-balancer` going false -> true across the stranger's departure.
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
		f("collision_box", box(-0.35, -0.35, 0.35, 0.35)),
		f("selection_box", box(-0.5, -0.5, 0.5, 0.5)),
		f("collision_mask", obj(f("layers", layers(
			"floor", "meltable", "object", "transport_belt", "water_tile",
		)))),
		f("picture", obsdata.EmptySprite()),
	)
}

// The item exists because the `foreign` leg asserts that the stranger's stack
// still places the STRANGER'S entity after this mod arrives beside it. Without
// one there would be no stack to ask about.
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

const (
	part = "balancer-part"
	icon = "__base__/graphics/icons/splitter.png"
)

// The prototype-table shorthands, forwarded from obsdata. See obs/bb2data for
// why they are functions rather than function values.
func obj(kvs ...fkdata.KV) fkdata.V      { return obsdata.Obj(kvs...) }
func f(key string, v fkdata.V) fkdata.KV { return obsdata.F(key, v) }
func str(s string) fkdata.V              { return obsdata.Str(s) }
func num(x float64) fkdata.V             { return obsdata.Num(x) }
func strs(ss ...string) fkdata.V         { return obsdata.Strs(ss...) }
func box(a, b, c, d float64) fkdata.V    { return obsdata.Box(a, b, c, d) }
func layers(names ...string) fkdata.V    { return obsdata.Layers(names...) }
