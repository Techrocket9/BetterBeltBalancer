package tune

import (
	"math"
	"reflect"
	"testing"
)

// everything is the predicate for a vanilla-plus-Space-Age game: every name any
// ladder in this package can reach exists.
func everything(string) bool { return true }

// only is a predicate over a fixed set, which is how a modpack that is missing
// something is expressed here.
func only(names ...string) func(string) bool {
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	return func(s string) bool { return set[s] }
}

// ---------------------------------------------------------------------------
// (a) RESOLVE CAN NEVER RETURN A NAME THE PREDICATE REJECTED.
//
// This is the one property the whole ladder design exists to have: an
// ingredient naming a prototype nobody defined is a HARD LOAD FAILURE with this
// mod's name on it, inside somebody else's pack. Everything else in this file is
// a convenience; this is the safety.
// ---------------------------------------------------------------------------

func TestResolveNeverEmitsAnUnprovenName(t *testing.T) {
	// Every subset of the vocabulary is too many, so this walks every option
	// against every ONE-NAME-PRESENT world and every ONE-NAME-MISSING world,
	// which between them exercise both ends of every ladder.
	vocab := ladderVocabulary()

	check := func(what string, ing []Ingredient, present func(string) bool) {
		t.Helper()
		for _, in := range ing {
			if !present(in.Name) {
				t.Errorf("%s emitted %q, which the predicate rejected: "+
					"that is a load failure in somebody's modpack", what, in.Name)
			}
			if in.Amount <= 0 {
				t.Errorf("%s emitted %q at amount %v", what, in.Name, in.Amount)
			}
		}
	}

	for _, opt := range append(RecipeOptions(), "a-value-from-a-newer-build") {
		check(opt+" @ everything", mustResolve(t, opt, everything), everything)
		check(opt+" @ nothing", mustResolve(t, opt, only()), only())

		for _, keep := range vocab {
			p := only(keep)
			check(opt+" @ only "+keep, mustResolve(t, opt, p), p)

			missing := map[string]bool{}
			for _, n := range vocab {
				missing[n] = n != keep
			}
			q := func(s string) bool { return missing[s] }
			check(opt+" @ all but "+keep, mustResolve(t, opt, q), q)
		}
	}
}

func mustResolve(t *testing.T, opt string, present func(string) bool) []Ingredient {
	t.Helper()
	ing, _ := ResolveRecipe(opt, present)
	return ing
}

func TestResolveWithNoPredicateEmitsNothing(t *testing.T) {
	// A caller that forgot the predicate must emit an UNCHECKED recipe over
	// nobody's dead body: nil means nothing exists.
	if got := Resolve(RecipePlan(RecipeVanilla), nil); len(got) != 0 {
		t.Fatalf("a nil predicate resolved to %v; it must resolve to nothing", got)
	}
}

// ---------------------------------------------------------------------------
// (b) EVERY LADDER TERMINATES, AND TERMINATES AT THE SAME PLACE.
// ---------------------------------------------------------------------------

func TestEveryLadderTerminates(t *testing.T) {
	for _, opt := range RecipeOptions() {
		for i, item := range RecipePlan(opt) {
			if len(item.Ladder) == 0 {
				t.Fatalf("%s item %d has an empty ladder", opt, i)
			}
			last := item.Ladder[len(item.Ladder)-1]
			if last != FallbackName {
				t.Errorf("%s item %d ends at %q; every ladder must end at %q",
					opt, i, last, FallbackName)
			}
			seen := map[string]bool{}
			for _, n := range item.Ladder {
				if seen[n] {
					t.Errorf("%s item %d names %q twice", opt, i, n)
				}
				seen[n] = true
			}
		}
	}
}

func TestEveryLadderResolvesInTheWorstWorldThereIs(t *testing.T) {
	// A game whose ONLY item is iron plate. Every option must still produce a
	// recipe, and every ingredient in it must be iron plate.
	p := only(FallbackName)
	for _, opt := range RecipeOptions() {
		ing, _ := ResolveRecipe(opt, p)
		if len(ing) == 0 {
			t.Errorf("%s resolved to nothing in a game that has iron plate", opt)
		}
		for _, in := range ing {
			if in.Name != FallbackName {
				t.Errorf("%s emitted %q where only %q exists", opt, in.Name, FallbackName)
			}
		}
	}
}

func TestNothingAtAllIsAnEmptyRecipeRatherThanAnInventedOne(t *testing.T) {
	// The degenerate pack: no iron plate either. A recipe with no ingredients
	// is a strange machine and a load that COMPLETES, which is the trade.
	for _, opt := range RecipeOptions() {
		if ing, _ := ResolveRecipe(opt, only()); len(ing) != 0 {
			t.Errorf("%s invented %v in a game with no items at all", opt, ing)
		}
	}
}

func TestTheIdentityPlanIsTheLastResort(t *testing.T) {
	// `splitter-express` in a pack with belts and plates but no splitter of any
	// tier falls all the way to its own last rung -- so it never reaches the
	// vanilla fallback, and `fellBack` must say so.
	p := only("iron-plate", "iron-gear-wheel", "transport-belt", "steel-plate")
	ing, fellBack := ResolveRecipe(RecipeSplitterExpress, p)
	if fellBack {
		t.Errorf("splitter-express fell back to vanilla; its own ladder still resolves")
	}
	want := []Ingredient{{"transport-belt", 1}, {"steel-plate", 2}}
	if !reflect.DeepEqual(ing, want) {
		t.Errorf("splitter-express in a splitterless pack = %v, want %v", ing, want)
	}

	// And the case the fallback IS for: a pack with iron plate and nothing else
	// resolves every option to iron plate, so nothing falls back either. The
	// only way to reach the fallback is a plan that resolves to NOTHING while
	// vanilla resolves to something -- which no plan here can do, because every
	// ladder ends at the same rung. That is a property worth having and it means
	// `fellBack` is a tripwire on a future plan rather than a live path today.
	for _, opt := range RecipeOptions() {
		if _, fb := ResolveRecipe(opt, only(FallbackName)); fb {
			t.Errorf("%s fell back where its own ladder terminates", opt)
		}
	}
}

func ladderVocabulary() []string {
	seen := map[string]bool{}
	var out []string
	for _, opt := range RecipeOptions() {
		for _, item := range RecipePlan(opt) {
			for _, n := range item.Ladder {
				if !seen[n] {
					seen[n] = true
					out = append(out, n)
				}
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// (c) THE VANILLA PLAN IS BYTE-EQUAL TO WHAT THIS MOD HAS ALWAYS EMITTED.
//
// THE TRIPWIRE THAT PROTECTS EVERY RECORDED NUMBER IN THE REPO. Every rate,
// every heap slope and every dump golden in CLAUDE.md was measured on a save
// whose balancer parts cost this. A default that drifted would move the
// data-stage golden and nothing else, and the golden would be re-captured by
// whoever moved it.
// ---------------------------------------------------------------------------

func TestVanillaIsTodaysRecipe(t *testing.T) {
	// A LITERAL COPY of the ingredient list that shipped in 0.3.0's
	// guest/go/data/recipe.go, written out here so that the comparison is
	// against a second statement of it rather than against the plan restated.
	want := []Ingredient{
		{Name: "iron-plate", Amount: 4},
		{Name: "iron-gear-wheel", Amount: 2},
		{Name: "transport-belt", Amount: 2},
	}
	got, fellBack := ResolveRecipe(RecipeVanilla, everything)
	if fellBack {
		t.Fatal("the vanilla plan fell back to itself")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the DEFAULT recipe moved:\n got  %v\n want %v\n"+
			"every recorded number in this repo was measured on the want", got, want)
	}
	if RecipeDefault() != RecipeVanilla {
		t.Fatalf("the default option is %q, not %q", RecipeDefault(), RecipeVanilla)
	}
}

func TestTheDefaultTechnologyIsTodays(t *testing.T) {
	if TechDefault() != TechLogistics {
		t.Fatalf("the default technology is %q, not %q", TechDefault(), TechLogistics)
	}
	if got := TechLadder(TechDefault()); !reflect.DeepEqual(got, []string{"logistics"}) {
		t.Fatalf("the default ladder is %v, not [logistics]", got)
	}
}

// ---------------------------------------------------------------------------
// The technology ladder.
// ---------------------------------------------------------------------------

func TestTechLaddersWalkDown(t *testing.T) {
	cases := map[string][]string{
		TechLogistics:                {"logistics"},
		TechLogistics2:               {"logistics-2", "logistics"},
		TechLogistics3:               {"logistics-3", "logistics-2", "logistics"},
		"a-value-from-a-newer-build": {"logistics"},
	}
	for opt, want := range cases {
		if got := TechLadder(opt); !reflect.DeepEqual(got, want) {
			t.Errorf("TechLadder(%q) = %v, want %v", opt, got, want)
		}
	}
	// Every ladder ends at the tier this mod has always used, so the last
	// resort is today's behaviour rather than an absent technology.
	for _, opt := range TechOptions() {
		l := TechLadder(opt)
		if l[len(l)-1] != TechLogistics {
			t.Errorf("TechLadder(%q) ends at %q, not %q", opt, l[len(l)-1], TechLogistics)
		}
		if l[0] != opt {
			t.Errorf("TechLadder(%q) starts at %q; an option must ask for itself first",
				opt, l[0])
		}
	}
}

// ---------------------------------------------------------------------------
// The belt-speed derivation.
// ---------------------------------------------------------------------------

func TestHiddenSpeedHoldsTheFloor(t *testing.T) {
	// VANILLA AND SPACE AGE, every belt tier that exists, plus the four
	// prototypes this mod's own clone set to the floor an hour earlier in the
	// same load. Turbo is 0.125, half the floor -- so on a stock game the
	// derivation must change NOTHING, which is what the regenerated dump golden
	// proves from the other side.
	vanilla := []float64{
		0.03125, 0.0625, 0.09375, // yellow, red, blue transport belts
		0.03125, 0.0625, 0.09375, // undergrounds
		0.03125, 0.0625, 0.09375, // splitters
		0.125, 0.125, 0.125, // turbo belt, underground, splitter (Space Age)
		0.125,                  // turbo lane splitter
		0.25, 0.25, 0.25, 0.25, // our own four, already at the floor
	}
	if got := HiddenSpeed(vanilla); got != SpeedFloor {
		t.Fatalf("a stock game derived %v; it must derive the floor %v", got, SpeedFloor)
	}
	if got := HiddenSpeed(nil); got != SpeedFloor {
		t.Fatalf("an empty game derived %v, want the floor %v", got, SpeedFloor)
	}
}

func TestHiddenSpeedTakesTheFastestBelt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		speeds []float64
		want   float64
	}{
		{"one modded belt above the floor", []float64{0.09375, 0.25, 0.5}, 0.5},
		{"the fastest of several", []float64{0.5, 0.3, 0.75, 0.25}, 0.75},
		{"exactly the floor", []float64{0.25}, 0.25},
		{"a hair over the floor", []float64{0.2500001}, 0.2500001},
		{"a hair under it", []float64{0.2499999}, 0.25},
		{"negative speeds cannot win", []float64{-1, 0.09375}, 0.25},
	} {
		if got := HiddenSpeed(tc.speeds); got != tc.want {
			t.Errorf("%s: HiddenSpeed(%v) = %v, want %v", tc.name, tc.speeds, got, tc.want)
		}
	}
}

func TestHiddenSpeedIgnoresANaN(t *testing.T) {
	// A NaN cannot become the maximum, because `>` is false against it -- so one
	// unreadable prototype cannot poison the answer for every other belt in the
	// game. A math.Max fold would have propagated it.
	got := HiddenSpeed([]float64{0.09375, math.NaN(), 0.5})
	if got != 0.5 {
		t.Fatalf("a NaN in the scan gave %v, want 0.5", got)
	}
	if got := HiddenSpeed([]float64{math.NaN()}); got != SpeedFloor {
		t.Fatalf("a NaN alone gave %v, want the floor %v", got, SpeedFloor)
	}
}

func TestTheBeltFamiliesAreTheBeltConnectableTypes(t *testing.T) {
	// All seven descend from TransportBeltConnectablePrototype, where `speed` is
	// mandatory. Pinned as a list because the scan is only as complete as this
	// is: a family left out is a belt family whose speed is silently ignored,
	// which is the defect the whole feature exists to remove.
	want := []string{
		"transport-belt", "underground-belt", "splitter", "lane-splitter",
		"loader", "loader-1x1", "linked-belt",
	}
	if got := BeltFamilies(); !reflect.DeepEqual(got, want) {
		t.Fatalf("BeltFamilies() = %v, want %v", got, want)
	}
	// The four prototypes this mod clones are members of four of them, which is
	// what makes "our own participate in the scan" true rather than assumed.
	for _, ours := range []string{"linked-belt", "transport-belt", "splitter", "lane-splitter"} {
		found := false
		for _, f := range want {
			found = found || f == ours
		}
		if !found {
			t.Errorf("this mod clones a %q and the scan does not walk that family", ours)
		}
	}
}
