package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/tune"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// One technology, hanging off a belt tier -- `logistics` by default, which is
// the same place the incumbent's first tier hangs, so a player who knows that
// mod finds this one where they expect it.
//
// THE UNIT IS COPIED FROM THAT TECHNOLOGY rather than written out, so the mod
// follows whatever the base game (or an overhaul mod) says the tier costs
// instead of pinning a number that is wrong in every modpack. WHICH tier is the
// `bbb-tech-cost` startup setting since 0.3.1, asked for on the portal beside
// the recipe cost, and its default is `logistics` -- today's behaviour to the
// digit.
//
// # The prerequisite moves with the unit, and that is not decoration
//
// Charging `logistics-3`'s science while still hanging off `logistics` would put
// a machine that costs blue science at a place in the tree a player reaches with
// red: researchable long before it is affordable, and out of order in
// Factoriopedia. So ONE technology is named and BOTH fields come from it. The
// prerequisite is therefore a technology this file has already proved present,
// which also removes the way today's unguarded `prerequisites = {"logistics"}`
// could fail -- a prerequisite naming a technology nobody defined is a load
// error, not a fallback.
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
	source, count, time, ingredients := researchUnit()

	proto := []fkdata.KV{
		f("type", str("technology")),
		f("name", str("bbb-balancer")),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("effects", list(obj(
			f("type", str("unlock-recipe")),
			f("recipe", str("bbb-balancer-part")),
		))),
	}
	// A prerequisite only where there is a technology to name. `source` is "" on
	// the fallback path, which is a game whose whole logistics chain is missing
	// or trigger-researched -- and `prerequisites = {"logistics"}` there is a
	// load error rather than a cost.
	if source != "" {
		proto = append(proto, f("prerequisites", list(str(source))))
	}
	proto = append(proto,
		f("unit", obj(
			f("count", count),
			f("ingredients", ingredients),
			f("time", time),
		)),
		f("order", str("a-b-bbb")),
	)

	fkdata.Extend(obj(proto...))
}

// researchUnit picks the technology this one copies and reads its cost, walking
// the chosen option's ladder down the belt tiers and falling back to what base
// has always charged for `logistics`.
//
// The first result is the technology the cost came from, which is also the
// PREREQUISITE -- "" when nothing in the ladder had a unit.
//
// THE GUARD CHANGES NOTHING ON THE DEFAULT PATH, which is what makes it safe:
// where `logistics` has a `unit`, the first rung wins, every field below is that
// unit's and the dump is unmoved. What it removes is the one way this file could
// fail badly.
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
// every logistics tier gets the vanilla cost rather than a broken load, and
// nobody who has not done that can tell the difference.
//
//go:noinline
func researchUnit() (source string, count, time, ingredients fkdata.V) {
	for _, name := range tune.TechLadder(techOption()) {
		// The three fields are read from the `unit` map rather than by three
		// separate deep Gets, because the question being asked is about the unit
		// as a whole: a `unit` that is not a table is the trigger-technology
		// case, and three independent absent answers could not tell that apart
		// from a technology with a unit that happens to be missing a field.
		unit, ok := fkdata.Get("technology", name, "unit")
		if !ok || unit.Tag != fkdata.TagMap {
			continue
		}
		count, _ = unit.At("count")
		time, _ = unit.At("time")
		ingredients, _ = unit.At("ingredients")
		return name, count, time, ingredients
	}

	fkdata.Log("[BBB] no logistics technology in this game has a research unit " +
		"(removed, or trigger technologies); bbb-balancer falls back to " +
		"vanilla's own logistics cost and has no prerequisite")
	return "", num(20), num(15), list(list(str("automation-science-pack"), num(1)))
}

// techOption is which technology the player asked to hang this one off. See
// recipeOption: same read, same reasons, same treatment of an absent answer.
//
//go:noinline
func techOption() string {
	v, ok := fkdata.StartupSetting(tune.SettingTechCost)
	if !ok || v.Tag != fkdata.TagString {
		return tune.TechDefault()
	}
	return v.String()
}
