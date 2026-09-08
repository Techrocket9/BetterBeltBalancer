package tune

// THE RESEARCH COST, as WHICH TECHNOLOGY THIS ONE COPIES.
//
// The same portal ask as the recipe, one level up: a player who has re-costed
// the recipe to express belts does not want the research to still be twenty red
// science. And the answer is the same shape it has always been here -- the cost
// is READ off a base technology rather than written down -- because a number
// written down is wrong in every modpack. What the setting picks is which
// technology to read.
//
// SO THE PREREQUISITE MOVES WITH THE UNIT, and that is the half a "cost"
// setting makes easy to forget. Charging `logistics-3`'s science while still
// hanging off `logistics` puts a machine that costs blue science at a place in
// the tree a player reaches with red -- researchable long before it is
// affordable, and out of order in Factoriopedia. One technology is named once,
// and both fields come from it.

// The allowed values of `bbb-tech-cost`, in menu order.
const (
	TechLogistics  = "logistics"
	TechLogistics2 = "logistics-2"
	TechLogistics3 = "logistics-3"
)

// TechOptions is every allowed value of `bbb-tech-cost`, cheapest first. The
// strings ARE the base technology names, which is why there is no mapping table
// under this: the option a player picks is the technology they get.
func TechOptions() []string {
	return []string{TechLogistics, TechLogistics2, TechLogistics3}
}

// TechDefault is what the setting defaults to -- today's behaviour, which is
// `logistics`, the same tier the incumbent's first one hangs off.
func TechDefault() string { return TechOptions()[0] }

// TechLadder is the technologies to try for one option, most preferred first.
//
// It walks DOWN the belt tiers rather than stopping, for the reason every
// ladder in this package exists: a pack that removed `logistics-3` -- or, far
// likelier on 2.0 and after, one that turned it into a TRIGGER technology with
// no `unit` at all -- should give a player who asked for it the nearest thing
// the game still has, not a broken load and not a silently vanilla cost.
//
// THE DEFAULT ARM IS A SHAPE GUARD AND NOT A LIVE FALLBACK, exactly as
// [RecipePlan]'s is and for the same reason: [Plan] builds one `CostChoice` per
// allowed value, so an unknown string reaches the library's dropdown read
// rather than this switch, and since FkRecipes c7a806e it is REFUSED BY NAME
// where it used to get an empty ladder and land on [FallbackUnit] with no
// prerequisite:
//
//	fkrecipes: bbb-tech-cost holds "not-an-option", which is not one of its values
//
// [TestAnUnofferedStoredValueIsRefusedByName] pins that sentence for this
// dropdown and for the recipe's. What makes it unreachable is the engine
// RESETTING an unknown value to the default before the data stage runs,
// silently and with no log line; the quotation and the loud wrong-type case are
// in [RecipePlan]'s header.
//
// WHAT WALKS IT IS FkRecipes' `CostBy` since round two, as one `CostChoice` per
// option built by [Plan]. This file is the ladder DATA and the option list; the
// unit copy, the trigger-technology skip and the prerequisite moving with the
// unit are the library's, and [FallbackUnit] is what applies when no rung of the
// chosen ladder carries a unit at all.
func TechLadder(option string) []string {
	switch option {
	case TechLogistics3:
		return []string{TechLogistics3, TechLogistics2, TechLogistics}
	case TechLogistics2:
		return []string{TechLogistics2, TechLogistics}
	}
	return []string{TechLogistics}
}
