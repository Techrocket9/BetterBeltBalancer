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
//
// AND SINCE ROUND THREE THERE ARE THREE FIELDS BESIDE THE DROPDOWN, WHICH DID
// NOT COST THIS LIST A ROW. `better-belt-balancer-tech-packs`,
// `better-belt-balancer-tech-count` and `better-belt-balancer-tech-seconds`
// overwrite the chosen tier's unit ONE FIELD AT A TIME: a pack text on the word
// `default` and a number at 0 each leave that field to the tier. So a player
// who wants a price none of the three tiers charges writes one without leaving
// the tier they are on, and the tier still decides where the research hangs.
//
// THE THREE VALUES BELOW ARE THEREFORE THE WHOLE OF WHAT `bbb-tech-cost`
// ALLOWS, exactly as they were before the fields arrived. [TechOptions] is the
// one list, read by the declaration and by [techChoices] alike, and a player
// who had stored `logistics-3` still reads `logistics-3`.

// The tier values of `bbb-tech-cost`, in menu order.
const (
	TechLogistics  = "logistics"
	TechLogistics2 = "logistics-2"
	TechLogistics3 = "logistics-3"
)

// TechOptions is every allowed value of `bbb-tech-cost`, cheapest first. The
// strings ARE the base technology names, which is why there is no mapping table
// under this: the option a player picks is the technology they get.
//
// THE ORDER HERE IS THE MENU ORDER AND THAT IS ALL IT IS SINCE FIX ROUND 2.
// This list was also the placement ladder of a `custom` value that named no
// source technology; there is no such value any more, because every value of
// this dropdown names a source and the source is the prerequisite whatever the
// three settings beside it say. So nothing walks this list but the menu and the
// choice table, and [TechDefault] is still its head.
func TechOptions() []string {
	return []string{TechLogistics, TechLogistics2, TechLogistics3}
}

// TechDefault is what the setting defaults to -- today's behaviour, which is
// `logistics`, the same tier the incumbent's first one hangs off.
//
// ONE LIST AND THEREFORE ONE HEAD, exactly as [RecipeDefault] is the head of
// [RecipeOptions]: `allowed_values` and `default_value` are separate prototype
// fields and a default outside the list is a load error, so both come from one
// place here.
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
// value the dropdown allows, and [TechOptions] is that list, so no string a
// player can store reaches this switch. An unknown one reaches the library's
// dropdown read, and since FkRecipes c7a806e it is REFUSED BY NAME where it
// used to get an empty ladder and land on [FallbackUnit] with no prerequisite:
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
