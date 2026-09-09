// Package tune is what the DATA STAGE decides that is not fixed: the recipe's
// ingredients, the technology's research cost, and the speed of the hidden
// network's belts.
//
// It is PURE GO -- no fkdata, no fkapi, no wasm imports -- which is the fifth
// time this repo has split a decision out of a stage so a host toolchain can
// prove it (plan, skin, carry, edgemode, engine). The reason here is the one
// `engine` has and the other three do not: THE INTERESTING STATES ARE STATES OF
// SOMEBODY ELSE'S MOD SET. A ladder falls through only in a game where
// `express-transport-belt` is absent, and no mod set this repo can install has
// that shape -- so written inside `guest/go/data` the fallback arms would be
// branches nothing could execute and nothing could check.
//
// # Why a ladder at all, and what it is protecting against
//
// A data stage that names an ingredient nobody defined is a HARD LOAD FAILURE
// with this mod's name on it, in somebody else's overhaul pack, before a single
// prototype of theirs is read. That is the entire risk of making the cost
// configurable: today's recipe names three prototypes base has always had, and
// every option below names one it might not. So no name reaches `data:extend`
// unless something has PROVEN it present in data.raw first.
//
// The ladders all terminate at `iron-plate`, which is the cheapest thing any
// game with belts in it has; and if even that is absent, an ingredient is
// DROPPED rather than guessed. A recipe with fewer ingredients is a gameplay
// change in a game that has no iron at all. A recipe with an unproven one is a
// mod that does not load.
//
// # THIS PACKAGE KEEPS THE LADDERS AND STOPPED WALKING THEM
//
// `Resolve` and `ResolveRecipe` walked them here until round two of the
// FkRecipes migration and are DELETED: what walks them now is the library's
// `IngredientNamed` ladder, reached through the `IngredientsBy` in [Plan], and
// its presence probe is `fkdata.DerivedTypes("item")` where this package's
// caller wrote the item types out by hand. The property is the same one and it
// is proved one layer out, against a fixture World in plandata_test.go, so the
// two statements that could have drifted apart are one statement again.
//
// What is left is the DATA: [RecipePlan]'s six plans, [TechLadder]'s three, the
// option lists whose head is each default, [FallbackUnit], and the belt-speed
// derivation, which is not a library concern at all.
//
// # Determinism
//
// Everything here is a function of the option string. No map is ranged over,
// nothing is sorted at run time, and the plans are slices in source order -- so
// two clients running the same mod set emit byte-identical ingredient lists,
// which is what keeps a prototype checksum from turning a player away at the
// join.
package tune

// The six startup settings this package is the fold behind.
//
// NAMED HERE RATHER THAN IN THE DATA GUEST because three things have to agree
// about them and only one of the three can be compiled: the prototype the
// settings stage emits, the read at the data stage, and the LOCALE FILE, which
// is hand-edited text. Since the settings moved onto FkRecipes the first of the
// three is [Plan], in this same package, so the names, the defaults, the
// allowed values and the locale check are all reachable by one `go test`; the
// check itself is the library's own `CheckLocaleWith`, run from
// [TestTheLocaleFileSatisfiesThePlan].
//
// All six are defined on BOTH ENGINES, unlike `bbb-multi-edge-parts`, which
// exists on 2.0 alone because there is nothing for it to say on 2.1. A recipe
// cost means the same thing on either.
//
// TWO OF THE SIX ARE WRITTEN OUT AND FOUR ARE BUILT FROM [ModName], which is
// the whole of what the sync pass changed here. The two dropdowns have shipped
// under their historical names since long before this library existed and cross
// VERBATIM through the Legacy constructors, because mod-settings.dat is keyed by
// name with no rename mechanism. The four customizer fields have never shipped,
// so nothing is stored under any name of theirs, and they go through the
// library's ORDINARY constructors: it prefixes the bare name and derives the
// order, and [fkrecipes.Lib.OrderAfter] is what places each of them under the
// dropdown it belongs to.
//
// ROUND THREE DECLARED ALL FOUR Legacy AND THE REASON IS GONE. A generated
// setting's order came from its DECLARATION INDEX alone, so the three research
// fields sorted ABOVE the dropdown that switches them on and no ordering of the
// declarations could fix it; FkRecipes b2e47b6 answered exactly that ask.
// [Plan]'s header carries the measurement and the decision in full.
//
// THE SEAM IS REAL AND IT IS NOT IN THE SETTINGS SCREEN. Four rows spelled
// `better-belt-balancer-` beside two spelled `bbb-` is visible in the library's
// log lines and in `mod-settings.dat`, and nowhere a player looks: the menu
// renders the localised label, which is this mod's own text on all six. Every
// migrated mod that grows a setting carries the same seam.
const (
	SettingRecipeCost        = "bbb-recipe-cost"
	SettingRecipeIngredients = ModName + "-" + RecipeIngredientsName
	SettingTechCost          = "bbb-tech-cost"
	SettingTechPacks         = ModName + "-" + TechPacksName
	SettingTechCount         = ModName + "-" + TechCountName
	SettingTechSeconds       = ModName + "-" + TechSecondsName
)

// The four BARE names, which is what a generated constructor takes: the prefix
// is the library's to add, derived from `fkdata.ModName()` at the settings
// stage and from [ModName] in a host test, and there is no prefix parameter
// anywhere in the library.
//
// EXPORTED BECAUSE THE TWO HALVES HAVE DIFFERENT CALLERS. [Plan] passes the
// bare name to [fkrecipes.Lib.IngredientsSetting] and the three beside it; the
// transcription in plan_test.go, the locale file and test/check-datastage.py
// all name the EMITTED one, because that is what the prototype, the dump and a
// player's mod-settings.dat carry. The const block above is the one place the
// two are tied together.
//
// THE NAMES ARE THE ONES FkRecipes' docs/migration.md WORKS THROUGH for this
// mod, which is not a coincidence: the worked example is this plan.
const (
	RecipeIngredientsName = "recipe-ingredients"
	TechPacksName         = "tech-packs"
	TechCountName         = "tech-count"
	TechSecondsName       = "tech-seconds"
)

// SettingMultiEdgeParts is the 2.0-only runtime-global bool, which this package
// does NOT decide anything about and names for one reason: it is a setting this
// mod declares OUTSIDE FkRecipes, and the library's locale checker has to be
// told so.
//
// `CheckLocaleWith` polices `[mod-setting-name]` and `[mod-setting-description]`
// against the complete set of the mod's setting names rather than against the
// mod prefix, which is the stronger reading and the one that catches a renamed
// setting's leftover entry. It can only be complete if it is handed what the
// mod declares elsewhere, and this is the whole of that list. Its own
// definition is guest/go/data/settings.go, behind the `bbb-can-stack` gate.
const SettingMultiEdgeParts = "bbb-multi-edge-parts"

// ModName is the name fklua.toml packages this mod under, and it is the prefix
// FkRecipes derives every generated name from.
//
// WRITTEN DOWN HERE BECAUSE A HOST TEST HAS NO fkdata TO ASK. At the stage the
// library reads it off the World, which reads it off fklua; a `go test` has
// neither, so the locale check has to be handed one. A wrong value would be a
// wrong prefix for every key at once, which is loud rather than subtle, and
// [TestModNameIsTheManifestName] compares it against fklua.toml.
const ModName = "better-belt-balancer"

// HandRolledSettings is every setting this mod declares outside FkRecipes, for
// `CheckLocaleWith`.
//
// THE LIST SUPPRESSES ORPHANS AND CREATES NO OBLIGATIONS, which is the
// library's rule and the reason the locale test still makes three assertions of
// its own: being told a name exists tells the checker nothing about whether it
// is a dropdown, a bool or a runtime-global, so it never demands an entry for
// one. What it stops is `bbb-multi-edge-parts`' own `[mod-setting-name]` line
// being reported as an entry matching no declared setting.
func HandRolledSettings() []string {
	return []string{SettingMultiEdgeParts}
}

// Item is one entry of a PLAN: a ladder of candidate names, most preferred
// first, and the amount to ask for.
//
// The amount does not move down the ladder. A fallback is a substitute for a
// thing the game does not have, not a re-costing of the recipe -- and a
// substitute priced differently would make the same option mean two different
// things in two mod sets, which is worse than either.
type Item struct {
	Ladder []string
	Amount float64
}

// FallbackName is the last rung of every ladder in this package, and
// [TestEveryLadderTerminates] is what keeps it that way.
const FallbackName = "iron-plate"
