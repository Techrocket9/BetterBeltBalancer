package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// One technology, hanging off `logistics` -- the same place the incumbent's
// first tier hangs, so a player who knows that mod finds this one where they
// expect it.
//
// THE UNIT IS COPIED FROM `logistics` rather than written out, so the mod
// follows whatever the base game (or an overhaul mod) says that tier costs
// instead of pinning a number that is wrong in every modpack.
//
// # The aliasing hazard that died in the port
//
// The Lua this replaces read `logistics.unit.ingredients` and put THE SAME
// TABLE into this technology -- a reference, not a copy, which is what Lua's `=`
// is. Two technologies then shared one ingredients array, and any later mod (or
// any later edit here) that touched ours would silently have edited base's
// `logistics` as well. It never bit, because nothing ever wrote to it.
//
// It cannot bite now, and not because anybody is being careful: `fkdata.Get`
// marshals the value across the wasm boundary, so what arrives is guest memory
// that base's table has no relationship with. The hazard is gone BY
// CONSTRUCTION rather than by discipline, which is the only way this repo
// counts one as gone. test/check-datastage.py hashes the WHOLE dump for the
// same reason -- a data stage that damaged somebody else's prototype is exactly
// what a subset hash would be blind to.
//
//go:noinline
func technology() {
	count, time, ingredients := logisticsUnit()

	fkdata.Extend(obj(
		f("type", str("technology")),
		f("name", str("bbb-balancer")),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("effects", list(obj(
			f("type", str("unlock-recipe")),
			f("recipe", str("bbb-balancer-part")),
		))),
		f("prerequisites", list(str("logistics"))),
		f("unit", obj(
			f("count", count),
			f("ingredients", ingredients),
			f("time", time),
		)),
		f("order", str("a-b-bbb")),
	))
}

// logisticsUnit reads base's `logistics` research cost, or falls back to what
// base has always charged for it.
//
// THE GUARD IS NEW AND IT CHANGES NOTHING ON THE GOLDEN PATH, which is what
// makes it safe to add inside a behaviour-preserving port: where `logistics`
// has a `unit`, every field below is that unit's and the dump is unmoved. What
// it removes is the one way this file could fail badly.
//
// The Lua indexed `data.raw.technology["logistics"].unit.count` unguarded, so a
// modpack in which `logistics` is absent -- or, much likelier now, one in which
// it is a TRIGGER technology carrying `research_trigger` instead of `unit`, a
// form 2.0 introduced and base itself uses -- crashed the data stage with
// `attempt to index a nil value`, from inside a file the player has no reason
// to suspect.
//
// THE FALLBACK IS TODAY'S BEHAVIOUR, LITERALLY: base's own logistics unit, 20
// automation science over 15 seconds, which is what this technology has cost in
// every save this mod has ever been in. So a pack that removes or retriggers
// `logistics` gets the vanilla cost rather than a broken load, and nobody who
// has not done that can tell the difference.
//
//go:noinline
func logisticsUnit() (count, time, ingredients fkdata.V) {
	// The three fields are read from the `unit` map rather than by three
	// separate deep Gets, because the question being asked is about the unit as
	// a whole: a `unit` that is not a table is the trigger-technology case, and
	// three independent absent answers could not tell that apart from a
	// technology with a unit that happens to be missing a field.
	unit, ok := fkdata.Get("technology", "logistics", "unit")
	if !ok || unit.Tag != fkdata.TagMap {
		fkdata.Log("[BBB] base's `logistics` technology has no research unit " +
			"(removed, or a trigger technology); bbb-balancer falls back to " +
			"vanilla's own logistics cost")
		return num(20), num(15), list(list(str("automation-science-pack"), num(1)))
	}
	count, _ = unit.At("count")
	time, _ = unit.At("time")
	ingredients, _ = unit.At("ingredients")
	return count, time, ingredients
}
