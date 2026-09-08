package tune

import (
	"math"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// (a) EVERY LADDER TERMINATES, AND TERMINATES AT THE SAME PLACE.
//
// THE PROPERTY THE WHOLE LADDER DESIGN RESTS ON, and the one this package still
// owns after round two of the FkRecipes migration handed the WALK to the
// library. What the library guarantees is that no name it emits is one the game
// does not have; what nothing outside this file can guarantee is that a ladder
// ends somewhere a game with belts in it always has, so that the guarantee is
// not vacuously satisfied by an empty recipe. plandata_test.go asserts the
// consequence against a fixture World; this asserts the shape.
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

// ladderVocabulary is every name any ladder in this package can reach, which is
// what stocks the fixture game in world_test.go. Built from the plans so that a
// rung added there is a rung the fixture has, rather than a rung the fixture
// silently answers absent for.
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
// (b) THE VANILLA PLAN IS BYTE-EQUAL TO WHAT THIS MOD HAS ALWAYS EMITTED.
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
	want := []ingredientPair{
		{"iron-plate", 4},
		{"iron-gear-wheel", 2},
		{"transport-belt", 2},
	}

	// THROUGH THE PLAN SINCE ROUND TWO, not through a resolver of this
	// package's own: `ResolveRecipe` is deleted and what turns a plan into
	// ingredients is the library. The literal above did not move, which is the
	// point -- this test compares the SAME statement against a different
	// machine.
	protos, logs := extendsOf(t, dataOps(t, everythingWorld()))
	for _, line := range logs {
		t.Errorf("the default recipe degraded in a game that has everything: %s", line)
	}
	got := ingredientsOf(t, protoOf(t, protos, "recipe", PartName))
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
	// THE UNKNOWN-OPTION CASE IS GONE FROM HERE, and its deletion is the
	// finding rather than a tidy-up. It asserted `TechLadder`'s default arm,
	// which NO SHIPPED PATH CONSULTS: [Plan] builds one CostChoice per allowed
	// value, so an unknown string reaches the library's own lookup and never
	// this switch. What the real path does with one since FkRecipes c7a806e is
	// REFUSE the load by name, and
	// [TestAnUnofferedStoredValueIsRefusedByName] is where that is pinned.
	cases := map[string][]string{
		TechLogistics:  {"logistics"},
		TechLogistics2: {"logistics-2", "logistics"},
		TechLogistics3: {"logistics-3", "logistics-2", "logistics"},
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
