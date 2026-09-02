package tune

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	fkrecipes "github.com/Techrocket9/fkrecipes/go"
)

// WHAT THE DATA STAGE EMITS, SAID ON THE HOST, FIELD BY FIELD.
//
// The same division of labour plan_test.go's header sets out for the settings.
// `test/check-datastage.py` hashes the whole normalised dump and answers
// "nothing moved"; this answers "this is what it is", names the field when it
// disagrees, needs no Factorio and no wasm toolchain, and runs in milliseconds.
//
// THE EXPECTED VALUES ARE TRANSCRIBED FROM THE PRE-MIGRATION DUMP, not derived
// from the plan. A test that built its expectation out of [RecipePlan] would
// agree with a defect in [RecipePlan] and say so cheerfully. They come from the
// scratchpad references taken before a line was moved
// (`ref-item-bbb-balancer-part.json` and its two neighbours), which is the one
// reading that can disagree with this package.
//
// KEYS ARE COMPARED ORDER-INSENSITIVELY, ARRAYS ARE NOT. fkdata sorts a map's
// pairs on the way out and `jq -S` sorts them again in the gate, so pair order
// is not something this mod may hold the library to. An ingredient LIST's order
// is a different matter: it is what a player sees in the recipe tooltip, it is
// in the dump, and the golden hash pins it.

// dataOps runs the data plan against a fixture game and insists it was not
// refused. A refusal here is a mod that does not load: `EmitData` routes every
// one through `fkdata.Raise`.
func dataOps(t *testing.T, w fixtureWorld) []fkrecipes.Op {
	t.Helper()
	ops, err := Plan().PlanData(w)
	if err != nil {
		t.Fatalf("the data plan was refused, so the data stage would raise "+
			"this at load: %v", err)
	}
	return ops
}

// extendsOf is the OpExtend stream with the log lines separated out, so a test
// can index the prototypes without counting degradation notices.
func extendsOf(t *testing.T, ops []fkrecipes.Op) (protos []fkrecipes.Value, logs []string) {
	t.Helper()
	for i, op := range ops {
		switch op.Kind {
		case fkrecipes.OpExtend:
			protos = append(protos, op.Proto)
		case fkrecipes.OpLog:
			logs = append(logs, op.Line)
		default:
			// An OpSet is a write into somebody else's prototype. This plan
			// splices no prerequisite into anything, so one here would mean the
			// library was asked for something this mod did not ask for.
			t.Errorf("op %d is kind %d, and this plan produces only extends "+
				"and logs", i, op.Kind)
		}
	}
	return protos, logs
}

// protoOf finds one prototype of the stream by its TYPE AND its name, which is
// what lets the tests below stop depending on the order the library emits in.
//
// THE TYPE IS NOT DECORATION HERE. This mod's item and its recipe are BOTH
// called `bbb-balancer-part` -- Factorio keeps one namespace per prototype type
// and a recipe named after the thing it makes is the ordinary shape -- so a
// lookup by name alone would hand the item's fields to the recipe's test and
// pass on the two fields they share.
func protoOf(t *testing.T, protos []fkrecipes.Value, typ, name string) map[string]fkrecipes.Value {
	t.Helper()
	for _, p := range protos {
		fields := fieldsOf(t, p)
		gotType, hasType := fields["type"]
		gotName, hasName := fields["name"]
		if !hasType || !hasName {
			continue
		}
		if gotType.Kind == fkrecipes.KindStr && gotType.Str == typ &&
			gotName.Kind == fkrecipes.KindStr && gotName.Str == name {
			return fields
		}
	}
	t.Fatalf("no %s named %q was emitted", typ, name)
	return nil
}

// TestTheDataPlanIsTheseExtendsAndNothingElse is the shape of the whole stream:
// one prototype per declaration and NOT ONE LOG LINE.
//
// A log line is the library saying something degraded -- an ingredient dropped,
// a setting unreadable, a cost source with no unit. In a game that has
// everything this mod's ladders can reach, every one of those is a defect in
// the plan or in the fixture, so the count is asserted at zero rather than
// tolerated.
func TestTheDataPlanIsTheseExtendsAndNothingElse(t *testing.T) {
	protos, logs := extendsOf(t, dataOps(t, everythingWorld()))
	if len(protos) != dataPrototypeCount {
		t.Errorf("the data stage emits %d prototype(s) and this mod declares %d "+
			"through the library", len(protos), dataPrototypeCount)
	}
	for _, line := range logs {
		t.Errorf("the plan degraded in a game that has everything: %s", line)
	}
}

// dataPrototypeCount is what this mod declares through the library at a data
// stage, written down rather than measured off the plan for the reason every
// expectation in this file is transcribed.
const dataPrototypeCount = 3

// TestTheItemIsTheOneThatShipped is the transcription, field by field.
//
// EIGHT FIELDS AND NO NINTH. A field this mod did not ask for is a field in the
// dump and therefore a moved golden hash, and one it stopped asking for is a
// silently different item -- an absent `place_result` is an item that builds
// nothing, which is the exact half round one could not have.
func TestTheItemIsTheOneThatShipped(t *testing.T) {
	protos, _ := extendsOf(t, dataOps(t, everythingWorld()))
	got := protoOf(t, protos, "item", PartName)

	checkStr(t, PartName, got, "type", "item")
	checkStr(t, PartName, got, "name", "bbb-balancer-part")
	checkStr(t, PartName, got, "icon",
		"__better-belt-balancer__/graphics/icons/balancer-part.png")
	checkNum(t, PartName, got, "icon_size", 64)
	checkNum(t, PartName, got, "stack_size", 50)
	checkStr(t, PartName, got, "subgroup", "belt")
	checkStr(t, PartName, got, "order", "c[splitter]-y[bbb-balancer]")
	checkStr(t, PartName, got, "place_result", "bbb-balancer-part")
	checkOnly(t, PartName, got,
		"type", "name", "icon", "icon_size", "stack_size", "subgroup",
		"order", "place_result")
}

// TestThePlaceResultProbeRefusesAnAbsentEntity is the ordering constraint of
// guest/go/data/main.go's `fk_data` hook, stated as the failure it produces.
//
// The item's `place_result` names this mod's own hand-rolled entity, and the
// library presence probes every name it emits -- because the engine's answer to
// an item naming an entity that is not there is an assignID abort naming the
// ITEM rather than the declaration. So `entity()` has to run before `EmitData`,
// and a pass that reorders those two gets this sentence at load rather than a
// mod that quietly ships an item that builds nothing.
func TestThePlaceResultProbeRefusesAnAbsentEntity(t *testing.T) {
	_, err := Plan().PlanData(everythingWorld().withoutEntities())
	if err == nil {
		t.Fatal("a game with no bbb-balancer-part entity planned cleanly: the " +
			"place_result probe is not being made, so a reordered fk_data hook " +
			"would ship an item that builds nothing")
	}
	want := "fkrecipes: the item bbb-balancer-part names a place_result " +
		"bbb-balancer-part that does not exist"
	if err.Error() != want {
		t.Errorf("the refusal reads\n got  %q\n want %q", err.Error(), want)
	}
}

// checkNum is checkStr for a number. Factorio has one number type and it is a
// double, so an integer field is compared as one.
func checkNum(t *testing.T, proto string, got map[string]fkrecipes.Value, key string, want float64) {
	t.Helper()
	v, ok := got[key]
	if !ok {
		t.Errorf("%s has no %s field", proto, key)
		return
	}
	if v.Kind != fkrecipes.KindNum {
		t.Errorf("%s's %s came out as kind %d rather than a number", proto, key, v.Kind)
		return
	}
	if v.Num != want {
		t.Errorf("%s's %s is %v and shipped as %v", proto, key, v.Num, want)
	}
}

// checkOnly names every field a prototype is allowed to carry, so an extra one
// is reported BY NAME rather than as a count that does not match.
func checkOnly(t *testing.T, proto string, got map[string]fkrecipes.Value, allowed ...string) {
	t.Helper()
	for _, key := range sortedKeys(got) {
		if !has(allowed, key) {
			t.Errorf("%s carries an unexpected field %q", proto, key)
		}
	}
}

// ---------------------------------------------------------------------------
// THE RECIPE, and the six ingredient lists a player can choose between.
// ---------------------------------------------------------------------------

// TestTheRecipeIsTheOneThatShipped is the transcription of everything about it
// that is not the ingredients: the ingredients get their own table below,
// because they are what the setting moves.
//
// `enabled` IS THE FIELD THE TWO PROTOTYPES ARE COUPLED THROUGH, and it is why
// the recipe and the technology could not cross one at a time. The library
// emits it as "no technology in THIS PLAN unlocks me", so a recipe declared
// beside a hand-rolled technology comes out `enabled = true` -- craftable from
// the first minute, and a moved data hash.
func TestTheRecipeIsTheOneThatShipped(t *testing.T) {
	protos, _ := extendsOf(t, dataOps(t, everythingWorld()))
	got := protoOf(t, protos, "recipe", PartName)

	checkStr(t, "the recipe", got, "type", "recipe")
	checkStr(t, "the recipe", got, "name", "bbb-balancer-part")
	checkBool(t, "the recipe", got, "enabled", false)
	checkNum(t, "the recipe", got, "energy_required", 1)
	checkStr(t, "the recipe", got, "order", "c[splitter]-y[bbb-balancer]")
	checkResults(t, got, "bbb-balancer-part", 1)
	checkOnly(t, "the recipe", got,
		"type", "name", "enabled", "energy_required", "ingredients", "results", "order")
}

// TestEveryRecipeOptionIsTheListTheGateAsserts drives the real dropdown through
// the fixture, one value at a time, and compares against the lists
// `test/check-datastage.py` carries.
//
// TRANSCRIBED FROM THAT SCRIPT rather than derived from [RecipePlan], which is
// what makes this a second statement of the same fact rather than a restatement
// of the thing under test. The gate drives the same six values through a real
// `mod-settings.dat` and a real engine; this drives them through a fixture in
// milliseconds and says WHICH ingredient moved.
func TestEveryRecipeOptionIsTheListTheGateAsserts(t *testing.T) {
	for _, tc := range []struct {
		option string
		want   []ingredientPair
	}{
		{RecipeVanilla, []ingredientPair{
			{"iron-plate", 4}, {"iron-gear-wheel", 2}, {"transport-belt", 2}}},
		{RecipeCheap, []ingredientPair{
			{"iron-plate", 2}, {"transport-belt", 1}}},
		{RecipeBeltFast, []ingredientPair{
			{"iron-plate", 4}, {"iron-gear-wheel", 2}, {"fast-transport-belt", 2}}},
		{RecipeBeltExpress, []ingredientPair{
			{"steel-plate", 4}, {"iron-gear-wheel", 2}, {"express-transport-belt", 2}}},
		{RecipeSplitter, []ingredientPair{
			{"splitter", 1}, {"iron-plate", 2}}},
		{RecipeSplitterExpress, []ingredientPair{
			{"express-splitter", 1}, {"steel-plate", 2}}},
	} {
		w := everythingWorld().withStartup(SettingRecipeCost, tc.option)
		protos, logs := extendsOf(t, dataOps(t, w))
		for _, line := range logs {
			t.Errorf("%s degraded in a game that has everything: %s", tc.option, line)
		}
		checkIngredients(t, tc.option, protoOf(t, protos, "recipe", PartName), tc.want)
	}
}

// TestEveryOptionFallsAllTheWayToIronPlate is the worst world every ladder was
// written for: a game whose only item is the last rung.
//
// THE LIBRARY IS THE ONE KEEPING THAT PROMISE NOW. It used to be [Resolve], and
// the property is the same one -- no name reaches a prototype unless the game
// has it -- so this is the same assertion asked one layer out.
func TestEveryOptionFallsAllTheWayToIronPlate(t *testing.T) {
	for _, option := range RecipeOptions() {
		w := everythingWorld().
			withStartup(SettingRecipeCost, option).
			withItems(FallbackName, "automation-science-pack")
		protos, _ := extendsOf(t, dataOps(t, w))
		got := ingredientsOf(t, protoOf(t, protos, "recipe", PartName))
		if len(got) == 0 {
			t.Errorf("%s resolved to nothing in a game that has iron plate", option)
		}
		for _, ing := range got {
			if ing.name != FallbackName {
				t.Errorf("%s emitted %q where only %q exists",
					option, ing.name, FallbackName)
			}
		}
	}
}

// TestAGameWithNoIngredientsIsAnEmptyRecipeRatherThanAnInventedOne is the
// degenerate pack: no iron plate either.
//
// A recipe with no ingredients is a strange machine to craft and a load that
// COMPLETES, which is the trade this mod took when the ladders were written and
// which the library takes the same way. The alternative is naming a prototype
// nobody defined, which is not a load that completes.
//
// THE SCIENCE PACK IS STILL IN THIS GAME AND IT HAS TO BE. See
// TestTheFallbackPackIsProbedEvenWhenUnreachable, which is the finding.
func TestAGameWithNoIngredientsIsAnEmptyRecipeRatherThanAnInventedOne(t *testing.T) {
	for _, option := range RecipeOptions() {
		w := everythingWorld().
			withStartup(SettingRecipeCost, option).
			withItems("automation-science-pack")
		protos, _ := extendsOf(t, dataOps(t, w))
		if got := ingredientsOf(t, protoOf(t, protos, "recipe", PartName)); len(got) != 0 {
			t.Errorf("%s invented %v in a game with no ingredients at all", option, got)
		}
	}
}

// TestTheFallbackPackIsProbedEvenWhenUnreachable pins a BEHAVIOUR CHANGE round
// two brought in, so that it is a decision on the record rather than a surprise.
//
// The library validates `CostChoices.Fallback` before it resolves anything, so
// the fallback's science pack is presence probed whether or not the fallback is
// ever reached. A pack with a perfectly good `logistics` and no
// `automation-science-pack` in it is therefore REFUSED at plan time, where the
// hand-rolled technology this replaced would have loaded and used logistics'
// own unit.
//
// It is written down rather than worked around: there is no way to declare a
// CostBy with no fallback (a zero UnitSpec is refused for its count), the
// failure it replaces is narrower rather than absent (a fallback that DID fire
// in such a pack emitted a unit naming a missing item, which is the engine's
// own assignID abort), and the day the library probes lazily this test says so.
func TestTheFallbackPackIsProbedEvenWhenUnreachable(t *testing.T) {
	// `logistics` is present and carries a unit, so the fallback is
	// unreachable by construction; only the pack is missing.
	_, err := Plan().PlanData(everythingWorld().withItems(FallbackName))
	if err == nil {
		t.Fatal("a game with no automation-science-pack planned cleanly: the " +
			"library has stopped validating an unreachable fallback, which is " +
			"better and is not what this mod is written against")
	}
	want := "fkrecipes: the technology bbb-balancer prices itself in " +
		"automation-science-pack, which does not exist"
	if err.Error() != want {
		t.Errorf("the refusal reads\n got  %q\n want %q", err.Error(), want)
	}
}

// ---------------------------------------------------------------------------
// THE TECHNOLOGY, and the three costs a player can choose between.
// ---------------------------------------------------------------------------

func TestTheTechnologyIsTheOneThatShipped(t *testing.T) {
	protos, _ := extendsOf(t, dataOps(t, everythingWorld()))
	got := protoOf(t, protos, "technology", TechName)

	checkStr(t, TechName, got, "type", "technology")
	checkStr(t, TechName, got, "name", "bbb-balancer")
	checkStr(t, TechName, got, "icon",
		"__better-belt-balancer__/graphics/icons/balancer-part.png")
	checkNum(t, TechName, got, "icon_size", 64)
	checkStr(t, TechName, got, "order", "a-b-bbb")
	checkEffects(t, got, "bbb-balancer-part")
	checkOnly(t, TechName, got,
		"type", "name", "icon", "icon_size", "prerequisites", "unit", "effects", "order")
}

// TestTheCostAndThePrerequisiteComeFromOneSource is the rule `CostBy` exists to
// make easy, driven through all three values.
//
// THE UNIT IS COMPARED AGAINST THE FIXTURE'S OWN, not against a transcription.
// The claim is not "200 logistic science" -- it is "whatever this game charges
// for that technology", which is the whole reason the cost is read rather than
// pinned, and it is the same statement `test/check-datastage.py` makes by
// comparing against the SOURCE TECHNOLOGY'S unit in the same dump. The three
// fixture units differ from each other so a planner that always copied the
// first would fail here.
func TestTheCostAndThePrerequisiteComeFromOneSource(t *testing.T) {
	for _, option := range TechOptions() {
		base := everythingWorld()
		w := base.withStartup(SettingTechCost, option)
		protos, logs := extendsOf(t, dataOps(t, w))
		for _, line := range logs {
			t.Errorf("%s degraded in a game that has all three tiers: %s", option, line)
		}
		got := protoOf(t, protos, "technology", TechName)
		checkPrereqs(t, option, got, option)

		src, ok := base.tech(option)
		if !ok || src.unit == nil {
			t.Fatalf("the fixture has no unit for %s to be compared against", option)
		}
		if !reflect.DeepEqual(got["unit"], *src.unit) {
			t.Errorf("%s emitted the unit %s and %s charges %s",
				option, showValue(got["unit"]), option, showValue(*src.unit))
		}
	}
}

// TestALadderStepsDownAndTakesThePrerequisiteWithIt is the case the ladders
// exist for: a pack that removed a tier, or turned it into a trigger technology.
//
// The prerequisite has to move WITH the unit, so what is asserted is the pair.
func TestALadderStepsDownAndTakesThePrerequisiteWithIt(t *testing.T) {
	base := everythingWorld()
	l1, _ := base.tech(TechLogistics)
	l2, _ := base.tech(TechLogistics2)

	for _, tc := range []struct {
		name       string
		world      fixtureWorld
		wantSource string
		wantUnit   *fkrecipes.Value
	}{
		{
			// logistics-3 removed outright.
			name:       "logistics-3 absent",
			world:      base.withStartup(SettingTechCost, TechLogistics3).withTechs(l1, l2),
			wantSource: TechLogistics2,
			wantUnit:   l2.unit,
		},
		{
			// logistics-3 present and TRIGGER researched, which is the shape
			// 2.0 introduced and base itself uses elsewhere. It carries no
			// unit, so the ladder steps past it.
			name: "logistics-3 is a trigger technology",
			world: base.withStartup(SettingTechCost, TechLogistics3).
				withTechs(l1, l2, fixtureTech{name: TechLogistics3, trigger: true}),
			wantSource: TechLogistics2,
			wantUnit:   l2.unit,
		},
		{
			// Both upper tiers gone: the ladder ends at the tier this mod has
			// always used, so the last resort is today's behaviour.
			name:       "only logistics is left",
			world:      base.withStartup(SettingTechCost, TechLogistics3).withTechs(l1),
			wantSource: TechLogistics,
			wantUnit:   l1.unit,
		},
	} {
		protos, _ := extendsOf(t, dataOps(t, tc.world))
		got := protoOf(t, protos, "technology", TechName)
		checkPrereqs(t, tc.name, got, tc.wantSource)
		if !reflect.DeepEqual(got["unit"], *tc.wantUnit) {
			t.Errorf("%s: the unit is %s and %s charges %s",
				tc.name, showValue(got["unit"]), tc.wantSource, showValue(*tc.wantUnit))
		}
	}
}

// TestNoLogisticsAtAllIsTheFallbackAndNoPrerequisite is the arm the hand-rolled
// technology carried a guard for and no mod set this machine can install could
// reach.
//
// TWO THINGS, AND THE SECOND IS THE ONE WITH TEETH. The unit is [FallbackUnit],
// which is base's own logistics cost written out; and there is NO
// `prerequisites` FIELD AT ALL, because a prerequisite naming a technology
// nobody defined is a load error rather than a cost.
func TestNoLogisticsAtAllIsTheFallbackAndNoPrerequisite(t *testing.T) {
	w := everythingWorld().withTechs()
	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	if _, ok := got["prerequisites"]; ok {
		t.Errorf("the technology carries prerequisites %v in a game with no "+
			"logistics technology at all: that names something nobody defined",
			got["prerequisites"])
	}
	want := fkrecipes.Obj(
		fkrecipes.Pair("count", fkrecipes.Num(20)),
		fkrecipes.Pair("time", fkrecipes.Num(15)),
		fkrecipes.Pair("ingredients", fkrecipes.Arr(
			fkrecipes.Arr(fkrecipes.Str("automation-science-pack"), fkrecipes.Num(1)))),
	)
	if !reflect.DeepEqual(got["unit"], want) {
		t.Errorf("the fallback unit is %s, want %s",
			showValue(got["unit"]), showValue(want))
	}

	// The library says so out loud, which is the line that replaces this mod's
	// own `[BBB] no logistics technology ...`.
	found := false
	for _, line := range logs {
		found = found || strings.Contains(line, "no source for the")
	}
	if !found {
		t.Errorf("the fallback fired and nothing said so; the log lines were %v", logs)
	}
}

// TestEveryAmountIsAWholeNumber is what plan.go's int64 conversion rests on.
//
// [Item.Amount] is a float64 and the library takes an int64, so a plan with a
// fractional amount would be TRUNCATED silently on the way across. Every amount
// this package declares is 1, 2 or 4; this is what says so.
func TestEveryAmountIsAWholeNumber(t *testing.T) {
	for _, option := range RecipeOptions() {
		for i, item := range RecipePlan(option) {
			if item.Amount != float64(int64(item.Amount)) {
				t.Errorf("%s item %d asks for %v, which int64 truncates on the "+
					"way into the library", option, i, item.Amount)
			}
			if item.Amount < 1 {
				t.Errorf("%s item %d asks for %v, which the engine refuses",
					option, i, item.Amount)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// The readers the two prototypes above need.
// ---------------------------------------------------------------------------

// ingredientPair is one expected ingredient, transcribed.
type ingredientPair struct {
	name   string
	amount float64
}

// ingredientsOf reads a recipe's `ingredients` back as pairs. The LONG DICT
// form is what a 2.0 recipe uses -- `{type, name, amount}` -- and the `type` is
// checked rather than skipped, because the technology unit's SHORT TUPLE form
// is refused here by the engine and the two are easy to confuse.
func ingredientsOf(t *testing.T, proto map[string]fkrecipes.Value) []ingredientPair {
	t.Helper()
	v, ok := proto["ingredients"]
	if !ok || v.Kind != fkrecipes.KindArr {
		t.Fatal("the recipe has no ingredients array")
	}
	out := make([]ingredientPair, 0, len(v.Arr))
	for i, entry := range v.Arr {
		fields := fieldsOf(t, entry)
		checkStr(t, "ingredient", fields, "type", "item")
		name, amount := fields["name"], fields["amount"]
		if name.Kind != fkrecipes.KindStr || amount.Kind != fkrecipes.KindNum {
			t.Fatalf("ingredient %d is not a (name, amount) pair", i)
		}
		checkOnly(t, "ingredient", fields, "type", "name", "amount")
		out = append(out, ingredientPair{name: name.Str, amount: amount.Num})
	}
	return out
}

// checkIngredients compares the list IN ORDER, which is the one thing about a
// recipe's contents this file is order-sensitive about: it is what a player
// sees in the tooltip and it is in the dump.
func checkIngredients(t *testing.T, option string, proto map[string]fkrecipes.Value, want []ingredientPair) {
	t.Helper()
	got := ingredientsOf(t, proto)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s is made of %v and the gate asserts %v", option, got, want)
	}
}

// checkResults pins the one product: this mod's own item, one of it.
func checkResults(t *testing.T, proto map[string]fkrecipes.Value, name string, amount float64) {
	t.Helper()
	v, ok := proto["results"]
	if !ok || v.Kind != fkrecipes.KindArr || len(v.Arr) != 1 {
		t.Fatalf("the recipe's results are %v and it makes one thing", v)
	}
	fields := fieldsOf(t, v.Arr[0])
	checkStr(t, "the result", fields, "type", "item")
	checkStr(t, "the result", fields, "name", name)
	checkNum(t, "the result", fields, "amount", amount)
	checkOnly(t, "the result", fields, "type", "name", "amount")
}

// checkEffects pins the one effect: unlocking this mod's own recipe.
func checkEffects(t *testing.T, proto map[string]fkrecipes.Value, recipe string) {
	t.Helper()
	v, ok := proto["effects"]
	if !ok || v.Kind != fkrecipes.KindArr || len(v.Arr) != 1 {
		t.Fatalf("the technology's effects are %v and it unlocks one recipe", v)
	}
	fields := fieldsOf(t, v.Arr[0])
	checkStr(t, "the effect", fields, "type", "unlock-recipe")
	checkStr(t, "the effect", fields, "recipe", recipe)
	checkOnly(t, "the effect", fields, "type", "recipe")
}

// checkPrereqs pins the prerequisite list at exactly one name, which is what
// `CostBy` promises: the source the cost came from and nothing else.
func checkPrereqs(t *testing.T, what string, proto map[string]fkrecipes.Value, want string) {
	t.Helper()
	v, ok := proto["prerequisites"]
	if !ok || v.Kind != fkrecipes.KindArr {
		t.Errorf("%s: the technology has no prerequisites array", what)
		return
	}
	if len(v.Arr) != 1 || v.Arr[0].Kind != fkrecipes.KindStr || v.Arr[0].Str != want {
		t.Errorf("%s: the prerequisites are %v and the cost came from %q",
			what, v, want)
	}
}

func checkBool(t *testing.T, proto string, got map[string]fkrecipes.Value, key string, want bool) {
	t.Helper()
	v, ok := got[key]
	if !ok {
		t.Errorf("%s has no %s field", proto, key)
		return
	}
	if v.Kind != fkrecipes.KindBool {
		t.Errorf("%s's %s came out as kind %d rather than a bool", proto, key, v.Kind)
		return
	}
	if v.Bool != want {
		t.Errorf("%s's %s is %v and shipped as %v", proto, key, v.Bool, want)
	}
}

// showValue renders a Value the way a prototype reads, because the struct's own
// %v is six fields of which four are empty and the interesting one is nested.
// A unit that came out wrong should say so in one line a reader can compare.
func showValue(v fkrecipes.Value) string {
	switch v.Kind {
	case fkrecipes.KindBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case fkrecipes.KindNum:
		return strconv.FormatFloat(v.Num, 'g', -1, 64)
	case fkrecipes.KindStr:
		return strconv.Quote(v.Str)
	case fkrecipes.KindArr:
		parts := make([]string, 0, len(v.Arr))
		for _, item := range v.Arr {
			parts = append(parts, showValue(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case fkrecipes.KindMap:
		parts := make([]string, 0, len(v.Map))
		for _, pair := range v.Map {
			parts = append(parts, pair.Key+"="+showValue(pair.Val))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return "nil"
}
