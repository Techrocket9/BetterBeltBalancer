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
// An option this build does not know gets the default's ladder, for the reason
// [RecipePlan] gives: `allowed_values` is validated by the engine, so an unknown
// string means a downgrade from a newer build of this mod.
func TechLadder(option string) []string {
	switch option {
	case TechLogistics3:
		return []string{TechLogistics3, TechLogistics2, TechLogistics}
	case TechLogistics2:
		return []string{TechLogistics2, TechLogistics}
	}
	return []string{TechLogistics}
}
