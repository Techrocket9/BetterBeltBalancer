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
// AND SINCE ROUND THREE THERE IS A FOURTH VALUE WITH NO TECHNOLOGY UNDER IT.
// `custom` hands the research cost to three settings the player fills in --
// `better-belt-balancer-tech-packs`, `better-belt-balancer-tech-count` and
// `better-belt-balancer-tech-seconds` -- so a player who wants a price none of
// the three tiers charges can write one. It is a value of the dropdown and NOT
// an option of this file: [TechLadder] and [TechOptions]
// answer for the three tiers, [TechValues] is the four the setting allows, and
// the library refuses the two lists disagreeing in either direction.

// The tier values of `bbb-tech-cost`, in menu order.
const (
	TechLogistics  = "logistics"
	TechLogistics2 = "logistics-2"
	TechLogistics3 = "logistics-3"
)

// TechOptions is every allowed value of `bbb-tech-cost` that names a
// technology, cheapest first. The strings ARE the base technology names, which
// is why there is no mapping table under this: the option a player picks is the
// technology they get.
func TechOptions() []string {
	return []string{TechLogistics, TechLogistics2, TechLogistics3}
}

// TechCustom is the fourth value of `bbb-tech-cost`, and the only one that
// names no technology: the packs, the unit count and the seconds per unit come
// from the three settings beside it.
//
// IT IS DELIBERATELY NOT IN [TechOptions], exactly as [RecipeCustom] is not in
// [RecipeOptions] and for the same library rule: a value covered by a preset
// choice AND by a Custom arm is refused by name -- "gives custom a preset as
// well as a Custom arm; name the arm's value with CustomValue"
// (FkRecipes go/customize.go:425, `customArmValues`). So [TechLadder],
// [techChoices] and every test that iterates the TIERS keep reading the three,
// and only the dropdown's declaration reads [TechValues].
//
// The spelling is the library's own default for the arm, which is why
// `CostChoices.CustomValue` is left empty in [Plan]: that field exists for a mod
// whose dropdown ALREADY ships a preset called `custom`, and this one does not.
const TechCustom = "custom"

// TechValues is every value `bbb-tech-cost` allows, in the order the dropdown
// shows them: the three tiers, then the custom arm.
//
// THE THREE KEEP THEIR SPELLING AND THEIR POSITIONS AND THE FOURTH IS LAST,
// which is the whole migration, and it is the same statement [RecipeValues]
// makes for the recipe. Factorio keys a stored startup choice by its VALUE
// STRING in mod-settings.dat, so every player who had chosen `logistics-3`
// still reads `logistics-3` after the update; the only row that is new is the
// one nobody has stored.
func TechValues() []string {
	return append(TechOptions(), TechCustom)
}

// TechDefault is what the setting defaults to -- today's behaviour, which is
// `logistics`, the same tier the incumbent's first one hangs off.
//
// THE HEAD OF THE TIERS AND NOT OF [TechValues]: the default has to be a value
// a ladder answers for, and the custom arm is the one value that is not.
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
// TIER value and hands `custom` to the Custom arm instead, so no string a player
// can store reaches this switch. An unknown one reaches the library's dropdown
// read, and since FkRecipes c7a806e it is REFUSED BY NAME where it used to get
// an empty ladder and land on [FallbackUnit] with no prerequisite:
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
