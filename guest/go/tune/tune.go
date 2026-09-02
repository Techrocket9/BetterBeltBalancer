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
// unless the caller's predicate has PROVEN it present in data.raw first, and
// [Resolve] is written so that it cannot: the only strings it can return are
// ones `present` answered true for.
//
// The ladders all terminate at `iron-plate`, which is the cheapest thing any
// game with belts in it has; and if even that is absent, an ingredient is
// DROPPED rather than guessed. A recipe with fewer ingredients is a gameplay
// change in a game that has no iron at all. A recipe with an unproven one is a
// mod that does not load.
//
// # Determinism
//
// Everything here is a function of the option string and the predicate. No map
// is ranged over, nothing is sorted at run time, and the plans are slices in
// source order -- so two clients running the same mod set emit byte-identical
// ingredient lists, which is what keeps a prototype checksum from turning a
// player away at the join.
package tune

// The two startup settings this package is the fold behind.
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
// Both are defined on BOTH ENGINES, unlike `bbb-multi-edge-parts`, which exists
// on 2.0 alone because there is nothing for it to say on 2.1. A recipe cost
// means the same thing on either.
const (
	SettingRecipeCost = "bbb-recipe-cost"
	SettingTechCost   = "bbb-tech-cost"
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

// Ingredient is one resolved entry of a recipe: a name the predicate proved
// present, and how many of it.
type Ingredient struct {
	Name   string
	Amount float64
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

// Resolve turns a plan into the ingredients to emit.
//
// Each item walks its own ladder and takes the FIRST name `present` answers
// true for. An item whose whole ladder is absent is dropped. The result is
// therefore a subset of the names the predicate approved and can never contain
// anything else -- which is the one property this file exists to have, and
// [TestResolveNeverEmitsAnUnprovenName] is where it is pinned.
//
// A nil `present` is treated as "nothing exists", so a caller that forgot to
// pass one emits an empty recipe rather than an unchecked one.
func Resolve(plan []Item, present func(string) bool) []Ingredient {
	out := make([]Ingredient, 0, len(plan))
	for _, it := range plan {
		for _, name := range it.Ladder {
			if present != nil && present(name) {
				out = append(out, Ingredient{Name: name, Amount: it.Amount})
				break
			}
		}
	}
	return out
}
