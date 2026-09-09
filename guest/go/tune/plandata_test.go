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
//
// SIX AND NOT THE DROPDOWN'S SEVEN. `custom` is a value with no plan behind it,
// so there is nothing here for it to be compared against; the gate's own recipe
// loop stops at the same six for the same reason, and the customizer's arms are
// their own, in both places.
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

// mergeLine is the library's merge sentence, whose only moving part is the
// arithmetic it ends with.
//
// A LITERAL AND NOT [PartName] OR [FallbackName], which is this file's rule for
// a pinned sentence: a renamed constant must FAIL a sentence a player reads in
// their log, not be carried through it. Only the arithmetic is a parameter,
// because it is the one part that differs between the rows below and the one a
// wrong merge would get wrong.
func mergeLine(sum string) string {
	return "fkrecipes: bbb-balancer-part: iron-plate is in the list twice " +
		"after the fallbacks, so the amounts are added: " + sum
}

// TestEveryOptionCollapsesOntoIronPlateAndMergesTheAmounts is the worst world
// every ladder was written for: a game whose only item is the last rung.
//
// THE LIBRARY IS THE ONE KEEPING THAT PROMISE NOW. It used to be [Resolve], and
// the property is the same one -- no name reaches a prototype unless the game
// has it -- so this is the same assertion asked one layer out.
//
// THE LOG STREAM IS PART OF THE ASSERTION AND DISCARDING IT MASKED THE TEST.
// The library has a fallback of its own: a chosen plan that resolves to NOTHING
// falls back to the DEFAULT option's plan, with a line saying so. So an option
// whose ladders did not end at iron plate would drop every ingredient, take
// vanilla's, and emit three iron plates -- passing every assertion below while
// measuring the fallback rather than the ladder. Measured: with belt-express's
// three ladders ending at steel-plate instead, this test passed and only
// [TestEveryLadderTerminates] fired.
//
// SO THE LINES ARE COMPARED RATHER THAN COUNTED, and the warning above stays
// true. "Every line is an error" was the old form and is wrong since FkRecipes
// 696387f: in a game whose only item IS the last rung, every ladder lands on
// `iron-plate` and the library MERGES the second landing into the first with a
// line naming the arithmetic. That line is the correct outcome, so tolerating
// the whole stream (deleting the loop) would give back exactly the masking this
// header records -- a belt-express fallen through to vanilla's plan carries
// `the belt-express ingredients name nothing this game has`, which is not in
// the table below, and a `reflect.DeepEqual` over the whole stream rejects it.
//
// AND THE AMOUNT IS ASSERTED, WHICH THE OLD FORM DID NOT DO AT ALL. It checked
// that every emitted name was `iron-plate` and never what the player pays. The
// merge is where an arithmetic defect would live, so the number is the
// assertion with teeth: 8 for the three-ladder presets and 3 for the
// two-ladder ones.
func TestEveryOptionCollapsesOntoIronPlateAndMergesTheAmounts(t *testing.T) {
	// TRANSCRIBED FROM THE PLANS BY HAND, in the order [RecipeOptions] returns
	// them, and not derived: a table built by walking [RecipePlan] and adding
	// the amounts up would agree with a defect in [RecipePlan] and say so
	// cheerfully. vanilla is 4 + 2 + 2, belt-fast and belt-express the same
	// three amounts, and the other three presets are two entries each.
	for _, tc := range []struct {
		option string
		amount float64
		logs   []string
	}{
		{RecipeVanilla, 8, []string{mergeLine("4 plus 2 is 6"), mergeLine("6 plus 2 is 8")}},
		{RecipeCheap, 3, []string{mergeLine("2 plus 1 is 3")}},
		{RecipeBeltFast, 8, []string{mergeLine("4 plus 2 is 6"), mergeLine("6 plus 2 is 8")}},
		{RecipeBeltExpress, 8, []string{mergeLine("4 plus 2 is 6"), mergeLine("6 plus 2 is 8")}},
		{RecipeSplitter, 3, []string{mergeLine("1 plus 2 is 3")}},
		{RecipeSplitterExpress, 3, []string{mergeLine("1 plus 2 is 3")}},
	} {
		// ONE ITEM, WHICH IS WHAT THE HEADER SAYS AND IS NOW LITERALLY TRUE.
		// The science pack used to be stocked here too, because the library
		// probed a `CostBy` fallback's pack whether or not the fallback was
		// reached and that probe was an ItemExists. It is a ToolExists now,
		// answered by the fixture's own `tools`, which `withItems` does not
		// touch -- so nothing in this world's item vocabulary is about
		// research any more.
		w := everythingWorld().
			withStartup(SettingRecipeCost, tc.option).
			withItems(FallbackName)
		protos, logs := extendsOf(t, dataOps(t, w))
		if !reflect.DeepEqual(logs, tc.logs) {
			t.Errorf("%s said\n got  %q\n want %q", tc.option, logs, tc.logs)
		}
		checkIngredients(t, tc.option, protoOf(t, protos, "recipe", PartName),
			[]ingredientPair{{FallbackName, tc.amount}})
	}
}

// TestAPackWithoutTransportBeltMergesRatherThanDuplicating is the assessment's
// own trigger, at the plan layer: a game that has everything this package's
// ladders name EXCEPT `transport-belt`.
//
// WHAT IT IS FOR. The vanilla list names `iron-plate` at the top and ends its
// third ladder on `iron-plate`, so removing `transport-belt` makes the fallback
// land on a name the list already carries. That used to reach the engine as two
// `iron-plate` entries and refuse the load outright, with no `fkrecipes:` line,
// no setting named and the missing item unnamed -- on the DEFAULT preset, for a
// player who never opened the Startup tab (agents/migration-assessment.md,
// finding 1). Since FkRecipes 696387f the second landing is added into the
// first and the library says so.
//
// IT IS NOT THE SAME WORLD AS THE TEST ABOVE. That one removes everything and
// asks what a ladder does when it exhausts; this one removes ONE NAME and asks
// what a substitution does when it collides. The first is the ladder's floor,
// this is the merge's arithmetic against a real pack's shape.
//
// THE LAST FOUR ROWS ARE THE ANTI-VACUITY HALF AND ARE NOT OPTIONAL. A fixture
// that emptied the vocabulary rather than removing one name would collapse
// every preset onto iron plate and pass the first two rows; the four presets
// that name no `transport-belt` at top level must come out UNCHANGED and say
// NOTHING, which is what proves exactly one name went.
//
// AND THE ENGINE AGREES, WHICH NO HOST TEST CAN SAY. `test/check-datastage.py`'s
// `check_remover` drives this same removal through a real `--dump-data` behind a
// synthetic `bbbt-remover` and reads the merged list back out of the dump.
func TestAPackWithoutTransportBeltMergesRatherThanDuplicating(t *testing.T) {
	// The whole vocabulary less the one name, so this world is the everything
	// world minus `transport-belt` and nothing else.
	var stocked []string
	for _, name := range ladderVocabulary() {
		if name != "transport-belt" {
			stocked = append(stocked, name)
		}
	}
	if len(stocked) != len(ladderVocabulary())-1 {
		t.Fatalf("the fixture removed %d name(s) and this test removes one",
			len(ladderVocabulary())-len(stocked))
	}

	for _, tc := range []struct {
		option string
		want   []ingredientPair
		logs   []string
	}{
		// The two presets whose lists name `transport-belt` at top level AND
		// carry `iron-plate` there too, which is what makes the collision.
		{RecipeVanilla,
			[]ingredientPair{{"iron-plate", 6}, {"iron-gear-wheel", 2}},
			[]string{mergeLine("4 plus 2 is 6")}},
		{RecipeCheap,
			[]ingredientPair{{"iron-plate", 3}},
			[]string{mergeLine("2 plus 1 is 3")}},
		// ANTI-VACUITY: four presets that name a belt this game still has, or
		// no belt at all. Every one of them is its stock-game list, and every
		// one of them is silent.
		{RecipeBeltFast, []ingredientPair{
			{"iron-plate", 4}, {"iron-gear-wheel", 2}, {"fast-transport-belt", 2}}, nil},
		{RecipeBeltExpress, []ingredientPair{
			{"steel-plate", 4}, {"iron-gear-wheel", 2}, {"express-transport-belt", 2}}, nil},
		{RecipeSplitter, []ingredientPair{
			{"splitter", 1}, {"iron-plate", 2}}, nil},
		{RecipeSplitterExpress, []ingredientPair{
			{"express-splitter", 1}, {"steel-plate", 2}}, nil},
	} {
		w := everythingWorld().
			withStartup(SettingRecipeCost, tc.option).
			withItems(stocked...)
		protos, logs := extendsOf(t, dataOps(t, w))
		if !reflect.DeepEqual(logs, tc.logs) {
			t.Errorf("%s in a game with no transport belt said\n got  %q\n want %q",
				tc.option, logs, tc.logs)
		}
		checkIngredients(t, tc.option, protoOf(t, protos, "recipe", PartName), tc.want)
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
// NO ITEMS AT ALL, AND THE RESEARCH IS STILL PAID FOR. The science pack used
// to be stocked here to keep the plan from being refused over an unreachable
// fallback; it is a TOOL now and `withItems` does not touch the tools, so this
// world is what its name says. See
// [TestAnUnreachedFallbacksPackIsNeverProbed], which is where that finding was
// closed.
func TestAGameWithNoIngredientsIsAnEmptyRecipeRatherThanAnInventedOne(t *testing.T) {
	for _, option := range RecipeOptions() {
		w := everythingWorld().
			withStartup(SettingRecipeCost, option).
			withItems()
		protos, _ := extendsOf(t, dataOps(t, w))
		if got := ingredientsOf(t, protoOf(t, protos, "recipe", PartName)); len(got) != 0 {
			t.Errorf("%s invented %v in a game with no ingredients at all", option, got)
		}
	}
}

// TestAnUnreachedFallbacksPackIsNeverProbed is round two's finding CLOSED, and
// the assertion is the inverse of the one that recorded it.
//
// WHAT ROUND TWO PINNED. `TestTheFallbackPackIsProbedEvenWhenUnreachable` said
// the library validated `CostChoices.Fallback` before it resolved anything, so
// the fallback's science pack was presence probed whether or not the fallback
// was ever reached: a pack with a perfectly good `logistics` and no
// `automation-science-pack` in it was REFUSED at plan time, where the
// hand-rolled technology this replaced loaded and copied logistics' own unit.
// It was graded AWKWARD in agents/fkrecipes-migration.md and its header ended
// "the day the library probes lazily this test says so".
//
// THAT DAY IS 2026-09-07. FkRecipes c7a806e resolves the fallback at the one
// point it applies -- "THE FALLBACK IS RESOLVED ONLY HERE, which is the point:
// its packs are probed when the fallback is what applies, and never when a
// source answered" (go/data.go:696) -- and the design record calls it the
// answer to this mod's ask by name, in a row of FkRecipes'
// agents/customizer-design.md whose decision ends: the Fallback is resolved
// only when used. The fallback's NUMBERS are still checked eagerly, in the plan
// walk with no World in hand (go/data.go:348), which is the right split: a
// count of zero is this mod's mistake and is knowable without asking the game
// anything.
//
// SO A GAME WITH NO SCIENCE PACK AT ALL LOADS, as long as something carries a
// unit. What says the SOURCE arm ran, rather than a fallback that happened to
// survive, is the prerequisite (exactly `logistics`) and the empty log stream:
// [FallbackUnit] is logistics' own numbers written out, so a surviving fallback
// would carry the same unit, and the unit comparison is kept as the half that
// says the research charges what logistics charges. Measured by the review:
// with every ladder emptied the fallback survives, the unit compares equal, and
// what fires is the missing prerequisite and the "fallback cost applies" line.
func TestAnUnreachedFallbacksPackIsNeverProbed(t *testing.T) {
	base := everythingWorld()
	l1, _ := base.tech(TechLogistics)

	protos, logs := extendsOf(t, dataOps(t, base.withTools()))
	for _, line := range logs {
		t.Errorf("a game whose logistics carries a unit degraded over a "+
			"fallback nothing reached: %s", line)
	}
	got := protoOf(t, protos, "technology", TechName)
	checkPrereqs(t, "an unreached fallback", got, TechLogistics)
	if !reflect.DeepEqual(got["unit"], *l1.unit) {
		t.Errorf("the unit is %s and logistics charges %s",
			showValue(got["unit"]), showValue(*l1.unit))
	}
}

// TestAReachedFallbackWithNoPackInTheGameIsRefused is the other half, and it is
// what the lazy probe MOVED rather than removed.
//
// The pack question did not go away: it moved to the one game that has to
// answer it, which is the game where the fallback is what prices the research.
// There the pack is walked as a ladder and dropped if nothing on it is a tool
// (FkRecipes c7a806e gave [fkrecipes.Pack] a `Fallbacks` list for exactly
// that), and a unit that loses every pack is refused rather than emitted --
// because a research with no packs is not a cheap research, it is a free one,
// which the engine loads without complaint.
//
// THIS MOD DECLARES ITS FALLBACK PACK WITH NO LADDER, one rung named
// `automation-science-pack`, so this is the whole of what a game without red
// science does to it: a refusal naming the technology, at plan time, which
// `EmitData` routes through `fkdata.Raise`. That is a load this mod fails and
// it is the right answer -- the alternative is a technology a player finishes
// by opening the research screen.
func TestAReachedFallbackWithNoPackInTheGameIsRefused(t *testing.T) {
	// No technology at all, so no rung of any ladder carries a unit and the
	// fallback IS the price; and no tool, so the one pack it names resolves to
	// nothing.
	_, err := Plan().PlanData(everythingWorld().withTechs().withTools())
	if err == nil {
		t.Fatal("a game with no science pack priced this mod's research " +
			"anyway: a research that costs nothing is one a player finishes " +
			"by opening the screen")
	}
	want := "fkrecipes: the technology bbb-balancer has no science pack the " +
		"game has; research takes at least one"
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

// TestAnUnofferedStoredValueIsRefusedByName is the OTHER SIDE OF THE ENGINE'S
// GUARD, re-pinned in the library's new words.
//
// WHAT IT USED TO SAY. Round two's TestAnUnknownOptionIsWhatTheLibraryDoesToday
// recorded that a stored value no dropdown offers reached the library's own
// choice lookup, got nothing back, and came out as a recipe made of NOTHING
// beside a fallback-priced technology -- with one log line, the technology's,
// and the recipe half silent. It was a recording rather than a requirement and
// it said so: if the library ever changed its answer, that test was what would
// say the answer moved.
//
// IT MOVED ON 2026-09-07. FkRecipes c7a806e refuses such a value by name, for
// `IngredientsBy` and `CostBy` alike, and names this mod as the reason: "What
// it used to do was worse than a refusal: the choice lookup found no plan, the
// recipe came out made of nothing, and no line said so. This is the pilot's own
// finding, closed." (go/data.go). A refusal is the right answer -- a balancer
// part craftable out of thin air is a worse outcome than a mod that does not
// load, and the state is not one a player can reach by accident.
//
// WHAT MAKES THE ARM UNREACHABLE IS STILL FACTORIO AND THAT HAS NOT CHANGED.
// An unknown value in `mod-settings.dat` is silently RESET to the default
// before the data stage runs, measured on the engine, with no log line; a wrong
// TYPE is refused loudly by the engine itself. So no player produces this state
// and neither can the dump gate: it is reachable only through a hand-edited
// mod-settings.dat, which is why it is pinned on the host and not in
// test/check-datastage.py.
//
// THE NAME IN THE SENTENCE IS THE EMITTED ONE, which for this mod's two Legacy
// settings is the unprefixed name a player's mod-settings.dat actually carries.
//
// ONE DROPDOWN AT A TIME, because the library reports the FIRST refusal its
// resolution walk found: a world that broke both would only ever prove the
// recipe's.
func TestAnUnofferedStoredValueIsRefusedByName(t *testing.T) {
	const unoffered = "not-an-option"
	for _, setting := range []string{SettingRecipeCost, SettingTechCost} {
		_, err := Plan().PlanData(everythingWorld().withStartup(setting, unoffered))
		if err == nil {
			t.Errorf("%s holding %q planned cleanly: the library has gone back "+
				"to answering an unoffered value with no plan at all, which is "+
				"a recipe made of nothing or a fallback-priced technology with "+
				"no prerequisite, and nothing said about the recipe half",
				setting, unoffered)
			continue
		}
		want := "fkrecipes: " + setting + ` holds "` + unoffered +
			`", which is not one of its values`
		if err.Error() != want {
			t.Errorf("%s: the refusal reads\n got  %q\n want %q",
				setting, err.Error(), want)
		}
	}
}

// TestTheCopiedUnitCarriesTheSourceMaxLevel is A BEHAVIOUR CHANGE ON THE
// RECORD, not a wish.
//
// `CostBy` copies the source technology's whole unit AND its `max_level`, which
// lives on the technology rather than in the unit; the hand-rolled
// `researchUnit` this replaced read three fields of the unit and nothing else.
// So in a pack whose chosen source is a multi-level technology, this mod's
// research becomes multi-level too: the unlock fires at level one and the other
// levels are no-ops a player pays for. Measured on the engine with a scratch mod
// setting `logistics-3`'s max_level to 3 -- the hand-rolled stage emitted no
// max_level and the library emits `"max_level":3`.
//
// The counterpart is already covered and is not duplicated here:
// [TestTheTechnologyIsTheOneThatShipped]'s `checkOnly` names every field the
// technology may carry, and `max_level` is not among them, so a stock game
// emitting one fails there.
func TestTheCopiedUnitCarriesTheSourceMaxLevel(t *testing.T) {
	base := everythingWorld()
	l1, _ := base.tech(TechLogistics)
	l2, _ := base.tech(TechLogistics2)
	l3, _ := base.tech(TechLogistics3)
	level := fkrecipes.Num(3)
	l3.maxLevel = &level

	w := base.withStartup(SettingTechCost, TechLogistics3).withTechs(l1, l2, l3)
	protos, _ := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	v, ok := got["max_level"]
	if !ok {
		t.Fatal("the source technology carries max_level 3 and the emitted one " +
			"carries none: the library has stopped copying it, which is better " +
			"and is not what this mod is written against")
	}
	if !reflect.DeepEqual(v, level) {
		t.Errorf("the emitted max_level is %s and the source's is %s",
			showValue(v), showValue(level))
	}
	// And it travels WITH the unit, from the same source, so the pair is what
	// is asserted rather than the field alone.
	if !reflect.DeepEqual(got["unit"], *l3.unit) {
		t.Errorf("the unit is %s and logistics-3 charges %s",
			showValue(got["unit"]), showValue(*l3.unit))
	}
	checkPrereqs(t, "max-level source", got, TechLogistics3)
}

// TestACountFormulaUnitSurvivesTheCopy is the IMPROVEMENT round two brought in,
// measured rather than claimed.
//
// The hand-rolled `researchUnit` read `count`, `time` and `ingredients` out of
// the source's unit and wrote those three. A source priced by `count_formula`
// -- the multi-level shape, which carries a formula and NO count -- therefore
// produced a unit with neither, and the engine refused the load with this mod's
// name on it:
//
//	Error while loading technology prototype "bbb-balancer" (technology):
//	Key "count_formula" not found in property tree at
//	ROOT.technology.bbb-balancer.unit
//
// The library copies the unit VERBATIM, so the same pack loads. Measured on the
// engine either side: rc=1 before, rc=0 after with `"unit":{"count_formula":"100",...}`.
func TestACountFormulaUnitSurvivesTheCopy(t *testing.T) {
	base := everythingWorld()
	l1, _ := base.tech(TechLogistics)
	formula := fkrecipes.Obj(
		fkrecipes.Pair("count_formula", fkrecipes.Str("100")),
		fkrecipes.Pair("time", fkrecipes.Num(30)),
		fkrecipes.Pair("ingredients", fkrecipes.Arr(
			fkrecipes.Arr(fkrecipes.Str("automation-science-pack"), fkrecipes.Num(1)))),
	)
	l2 := fixtureTech{name: TechLogistics2, prereqs: []string{TechLogistics}, unit: &formula}

	w := base.withStartup(SettingTechCost, TechLogistics2).withTechs(l1, l2)
	protos, logs := extendsOf(t, dataOps(t, w))
	for _, line := range logs {
		t.Errorf("a count_formula source degraded: %s", line)
	}
	got := protoOf(t, protos, "technology", TechName)
	if !reflect.DeepEqual(got["unit"], formula) {
		t.Errorf("the emitted unit is %s and the source's is %s",
			showValue(got["unit"]), showValue(formula))
	}
	// THE DEEP COMPARE ABOVE CANNOT FAIL BY MOVING THE FIXTURE, both sides
	// being the same value, which is right for a verbatim-copy claim and is
	// also why it is not the whole assertion. These two are TRANSCRIBED, so a
	// library that went back to naming three fields of the unit fires here:
	// `count_formula` is the key the old copy dropped, and an invented `count`
	// beside it is the other half of the same defect.
	unit := fieldsOf(t, got["unit"])
	checkStr(t, "the copied unit", unit, "count_formula", "100")
	if v, ok := unit["count"]; ok {
		t.Errorf("the copied unit carries a count of %s, which the source does "+
			"not: a count beside a count_formula is a unit the source never had",
			showValue(v))
	}
	checkPrereqs(t, "count_formula source", got, TechLogistics2)
}

// TestARungWhoseUnitIsNotADictionaryIsSteppedPast reaches a library guard the
// fixture could not reach until it could carry a raw unit.
//
// The library's ladder walk has TWO distinct tests on a rung: the absent FLAG,
// and whether what came back is a dictionary this library can copy faithfully.
// `unitOf` always builds a map and an absent unit answers false, so the second
// was unexercised. A technology whose `unit` is present and is not a table is a
// real shape -- a mod that assigned a string to it, or a value fkdata could not
// carry across the boundary -- and copying it would be a technology researchable
// for free.
//
// THE PREREQUISITE FOLLOWS, which is the half worth asserting: stepping past a
// rung has to move the tree position as well as the cost, or the research hangs
// off a technology whose price it is not charging.
func TestARungWhoseUnitIsNotADictionaryIsSteppedPast(t *testing.T) {
	base := everythingWorld()
	l1, _ := base.tech(TechLogistics)
	l2, _ := base.tech(TechLogistics2)
	broken := fixtureTech{
		name:    TechLogistics3,
		prereqs: []string{TechLogistics2},
		unit:    rawUnit(fkrecipes.Arr(fkrecipes.Str("not"), fkrecipes.Str("a-unit"))),
	}

	w := base.withStartup(SettingTechCost, TechLogistics3).withTechs(l1, l2, broken)
	protos, _ := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkPrereqs(t, "a non-dictionary unit", got, TechLogistics2)
	if !reflect.DeepEqual(got["unit"], *l2.unit) {
		t.Errorf("the unit is %s and logistics-2 charges %s",
			showValue(got["unit"]), showValue(*l2.unit))
	}
}

// TestAnUnreadableSettingTakesTheDeclaredDefault is the other guard the fixture
// could not reach, and it is not the same as an ABSENT setting.
//
// A setting that answers PRESENT with something that is not a string is what a
// mod redefining this mod's setting as an int would hand the planner. The
// library degrades to the declared default and says so, which is the right
// answer -- a data stage that refused there would be this mod failing to load
// over somebody else's edit -- and the line is what makes it visible.
func TestAnUnreadableSettingTakesTheDeclaredDefault(t *testing.T) {
	w := everythingWorld().withRawStartup(SettingRecipeCost, fkrecipes.Num(1))
	protos, logs := extendsOf(t, dataOps(t, w))

	// The DEFAULT recipe, from the declaration rather than from the read.
	checkIngredients(t, "an unreadable setting", protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{"iron-plate", 4}, {"iron-gear-wheel", 2}, {"transport-belt", 2}})

	want := "fkrecipes: the setting " + SettingRecipeCost +
		" was not readable, so its default applies"
	found := false
	for _, line := range logs {
		found = found || line == want
	}
	if !found {
		t.Errorf("the unreadable setting was not reported; the log stream is %v\n"+
			" want it to carry %q", logs, want)
	}
}

// ---------------------------------------------------------------------------
// THE CUSTOMIZER: the seventh value, and the text field behind it.
//
// `bbb-recipe-cost` gained one value and
// `better-belt-balancer-recipe-ingredients` arrived beside it, so a player who
// wants a recipe none of the six presets is can write one. Five behaviours are
// pinned below, and every sentence a PLAYER can be shown is compared word for
// word rather than by substring: a refusal that
// stops the load is the only thing they are given to fix it with, so its
// wording is as much this mod's surface as the recipe is.
// ---------------------------------------------------------------------------

// theDefaultWord is FkRecipes' reserved word for "the mod's own declared list,
// with its ladders". The library keeps it unexported (go/ingredientlist.go:100,
// `defaultWord`) and documents it in docs/ingredient-list.md, so a consumer
// writes it out; it is written out ONCE here, because the fixture, these tests
// and the engine gate all have to say the same seven letters.
const theDefaultWord = "default"

// vanillaLadderList is what the vanilla plan resolves to in a game that has
// every rung, transcribed rather than read off [RecipePlan] for the reason
// every expectation in this file is transcribed. It is what BOTH untouched
// paths of the customizer must produce: the word `default`, and a text the
// planner could not read at all.
func vanillaLadderList() []ingredientPair {
	return []ingredientPair{
		{"iron-plate", 4}, {"iron-gear-wheel", 2}, {"transport-belt", 2}}
}

// TestTheDefaultWordUnderCustomIsTheVanillaLadder is the state a player reaches
// by picking `custom` and typing nothing.
//
// THE WORD IS NOT A LIST AND THAT IS THE WHOLE DESIGN.
// `better-belt-balancer-recipe-ingredients` ships holding `default`, and the
// library resolves that to the DECLARED list with its ladders rather than to a
// rendering of it -- so a player who switches
// to `custom` and never edits gets exactly what `vanilla` gives them, in this
// release and in every later one, and in a modpack missing a rung
// (FkRecipes go/customize.go:1138, the isDefault arm of resolveIngredientsFrom).
//
// AND IT DRAWS NO LOG LINE, which is the library's stated rule for this arm:
// "the word default: the AUTHOR's declared list with its ladders, which is the
// pre-existing resolution path and gets no line of its own"
// (go/customize.go:1084). So the assertion is the EMPTY log stream rather than
// the absence of one particular sentence: a line here would be the library
// narrating an untouched field on every load of every game.
//
// THE THREE SPELLINGS ARE THE LANGUAGE'S, not this mod's guesses. Surrounding
// space is trimmed and one trailing comma is tolerated, so " default " and
// "default," are the marker too -- which matters beyond tidiness, because
// [TestAnEditedTextUnderAPresetIsIgnoredAndTheLogSaysSo] asks the SAME function
// what "edited" means: a spelling read as an edit would tell a player their
// untouched field was ignored.
func TestTheDefaultWordUnderCustomIsTheVanillaLadder(t *testing.T) {
	for _, text := range []string{theDefaultWord, " default ", "default,"} {
		w := everythingWorld().
			withStartup(SettingRecipeCost, RecipeCustom).
			withStartup(SettingRecipeIngredients, text)
		protos, logs := extendsOf(t, dataOps(t, w))
		for _, line := range logs {
			t.Errorf("the text %q is the default marker and the plan said "+
				"something about it: %s", text, line)
		}
		checkIngredients(t, "custom on "+strconv.Quote(text),
			protoOf(t, protos, "recipe", PartName), vanillaLadderList())
	}
}

// TestAnUnreadableCustomTextTakesTheDeclaredList is the OTHER untouched path,
// and it is not the same one.
//
// A setting the planner cannot read AT ALL is a hand-edited mod-settings.dat
// with the row missing: the engine writes every setting's current value into
// that file, untouched defaults included, so no player produces this. The
// library degrades to the declared list and SAYS SO, with the sentence it uses
// for every unreadable setting -- and that line is the whole difference between
// this arm and the word above. One is a player's choice; the other is a file
// somebody edited by hand, and it should not pass in silence.
//
// THE FIXTURE HAS TO ANSWER THE EMITTED NAME FOR THIS TO MEAN ANYTHING.
// `better-belt-balancer-recipe-ingredients` is generated, so the name the
// library asks `StartupSetting` for is the PREFIXED one a player's file carries.
// A fixture keyed on anything else would answer absent for every arm of the
// customizer, and this is the one test that would still pass.
func TestAnUnreadableCustomTextTakesTheDeclaredList(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeCost, RecipeCustom).
		withoutStartup(SettingRecipeIngredients)
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "an unreadable text",
		protoOf(t, protos, "recipe", PartName), vanillaLadderList())
	// THE SENTENCE IS A LITERAL, not the constants spliced together: this
	// section compares every line a player is shown word for word, and a
	// renamed constant has to fail this test rather than travel through it.
	checkExactlyOneLog(t, logs,
		"fkrecipes: the setting better-belt-balancer-recipe-ingredients was not "+
			"readable, so its default applies")
}

// TestTheCustomValueTakesItsIngredientsFromTheText is the feature, stated as
// the recipe a player gets.
//
// THE ORDER IS THE TYPED ORDER, which is why the comparison is a slice: the
// list a player writes is the list they see in the crafting tooltip, and a
// planner that sorted it would be quietly rewriting their recipe.
//
// THE LOG LINE IS PINNED WORD FOR WORD, AND IT IS THE CANONICAL RENDERING
// RATHER THAN THE TEXT AS TYPED: a player who wrote `iron-plate x3` reads back
// `3 iron-plate` and learns the form the library would have written
// (FkRecipes go/customize.go:1152). It names the RECIPE and the SETTING by their
// EMITTED names -- the recipe unprefixed because it is Legacy, the setting
// prefixed because it is generated -- so the line points at the row in the menu
// a player would go and edit.
func TestTheCustomValueTakesItsIngredientsFromTheText(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeCost, RecipeCustom).
		withStartup(SettingRecipeIngredients, "3 iron-plate, 1 splitter")
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "a written recipe", protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{"iron-plate", 3}, {"splitter", 1}})
	checkExactlyOneLog(t, logs,
		"fkrecipes: bbb-balancer-part takes its ingredients from "+
			"better-belt-balancer-recipe-ingredients: 3 iron-plate, 1 splitter")
}

// TestAnEditedTextUnderAPresetIsIgnoredAndTheLogSaysSo is the shape a pair of
// settings must not have: a field the player edits where nothing happens.
//
// The two rows are not both live. The text applies only while the dropdown says
// `custom`, and on any preset the preset wins -- so the library says out loud
// that it read the text and is not using it, which is the entire reason it
// parses a text it has no intention of applying.
//
// WHAT MAKES THIS NOT VACUOUS is the ingredient list beside the line. `cheap`
// and the text name the same item at different amounts, so an implementation
// that quietly took the text would emit one iron plate and fail the comparison
// rather than only losing a sentence in the log.
func TestAnEditedTextUnderAPresetIsIgnoredAndTheLogSaysSo(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeCost, RecipeCheap).
		withStartup(SettingRecipeIngredients, "1 iron-plate")
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "an edited text under cheap",
		protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{"iron-plate", 2}, {"transport-belt", 1}})
	// A literal for the reason the unreadable arm's is: a renamed constant
	// must fail this sentence rather than be carried by it.
	checkExactlyOneLog(t, logs,
		"fkrecipes: better-belt-balancer-recipe-ingredients is edited, but "+
			"bbb-recipe-cost is not on custom, so the text is ignored")
}

// fallbackLine is the ONE line the library logs when it sets a player's text
// aside, whose moving part is the reason inside it.
//
// LITERALS AGAIN, for the reason [mergeLine] is one. The prefix carries the
// severity in the text because Factorio's `log()` has one channel and no level,
// and the tail is the only instruction a player gets: this is the sentence that
// has to survive a refactor unchanged.
func fallbackLine(reason string) string {
	return "fkrecipes: ERROR: " + reason +
		". The mod loaded with its own default instead; fix the text under " +
		"Settings > Mod settings > Startup, then restart."
}

// TestACustomTextTheGameCannotAnswerFallsBackAndSaysSo pins the three texts a
// player is likeliest to get wrong, and what the mod does with each: it applies
// ITS OWN DECLARED LIST and writes one line naming the setting and the reason.
// A FOURTH ROW drives the first of those texts a second time in a game with no
// `transport-belt`, and it is there because the other three cannot tell the
// DECLARATION from the `vanilla` PRESET: in a game that has everything the two
// are the same three pairs. Its own comment says how they part.
//
// THE NAME USED TO SAY "IsRefused" AND THE LIBRARY STOPPED REFUSING. FkRecipes
// 2e5f779 made a value the PLAYER controls fall back instead, and the reason is
// measured rather than stylistic: a refusal on a field the player types into is
// a LOCK-OUT. The error dialog has no route to the Mod Settings screen, a
// disabled mod's settings are not shown there either, and the engine rewrites
// mod-settings.dat on a successful load and on no failed one -- so the value
// that caused the refusal cannot be edited from inside the refusal. The one
// escape measured was Reset mod settings plus Disable listed mods: six steps
// and every startup preference in the file lost
// (agents/migration-assessment.md, finding 2). A test whose name asserted the
// opposite of what the library now does would be worse than no test, hence the
// rename.
//
// WHAT IS ASSERTED IS THE PAIR, and neither half alone would do. The LIST says
// the player got this mod's recipe rather than a guess or a hole -- vanilla,
// which is the `default_value` of the dropdown they are not on, applied because
// their text was set aside. The LINE says the mod noticed: a fallback with no
// line is a player editing a field where nothing happens and nothing explains
// it, which is the same defect the ignored-text arm exists for at the other end.
//
// NOTHING IS SUBSTITUTED FOR A TYPED NAME, and that has not changed. A ladder is
// the AUTHOR saying "any of these will do"; a typed name is a player naming one
// thing, and the library still never guesses at it. What moved is what happens
// to the WHOLE typed list when one entry cannot be used: it is set aside for the
// author's declaration rather than taking the load down with it.
//
// THE FALLBACK IS NOT TOTAL AND NO ROW HERE IS THE EXCEPTION. What the text
// falls back TO is this mod's own declaration, and a mod set where THAT cannot
// produce a legal result still stops the load. Every ladder resolves in all
// four worlds here, the fourth by merging; the recipe field's own worst case
// is not a refusal at all but an empty ingredient list, which
// [TestAGameWithNoIngredientsIsAnEmptyRecipeRatherThanAnInventedOne] pins as the
// trade this mod took. The pack field's worst case IS a refusal, and
// [TestAPackTextTheGameCannotAnswerFallsBackAndSaysSo]'s last row is it.
//
// THE SECOND ARM IS THE ONE THIS MOD CAUSED. Its dropdown labels spell DISPLAY
// names ("Default: 4 iron plates, 2 gears, ..."), so a player copying the label
// they were on into the field types `iron plates` -- and the language answers
// with the internal name rather than with advice about commas. That fold exists
// in FkRecipes because this mod's labels were the example in its design review;
// keeping the labels as they are is the decision recorded in
// mod-data/locale/en/better-belt-balancer.cfg, and this is what makes keeping
// them safe.
func TestACustomTextTheGameCannotAnswerFallsBackAndSaysSo(t *testing.T) {
	// THE WHOLE VOCABULARY LESS `transport-belt`, FOR THE LAST ROW ALONE. Built
	// and guarded exactly as
	// [TestAPackWithoutTransportBeltMergesRatherThanDuplicating] builds it,
	// because a fixture that took two names away would make that row pass for
	// the wrong reason.
	var noBelt []string
	for _, name := range ladderVocabulary() {
		if name != "transport-belt" {
			noBelt = append(noBelt, name)
		}
	}
	if len(noBelt) != len(ladderVocabulary())-1 {
		t.Fatalf("the fixture removed %d name(s) and this test removes one",
			len(ladderVocabulary())-len(noBelt))
	}

	// THE VANILLA LIST, WHICH IS THIS MOD'S DECLARATION AND NOT A GUESS.
	// Written out rather than taken from [RecipePlan] for the reason every
	// expectation in this file is written out, and it is the same literal
	// [TestVanillaIsTodaysRecipe] holds.
	vanilla := []ingredientPair{
		{"iron-plate", 4}, {"iron-gear-wheel", 2}, {"transport-belt", 2}}

	for _, tc := range []struct {
		text   string
		reason string
		// stocked is the game's item vocabulary, and it is nil on every row but
		// the last: the first three run in the everything world.
		stocked []string
		want    []ingredientPair
		// extra is the SECOND log line, and only the last row has one.
		extra string
	}{
		{
			// A name no mod in this game defines. The fixture stocks every rung
			// of every ladder and nothing else, which is what makes it absent.
			text: "3 tungsten-plate",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("3 tungsten-plate"): no item or fluid is named tungsten-plate`,
			want: vanilla,
		},
		{
			// The display name off this mod's own dropdown label.
			text: "2 iron plates",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("2 iron plates"): no item or fluid is named "iron plates"; ` +
				`did you mean iron-plate`,
			want: vanilla,
		},
		{
			// A NAME THE GAME HAS, WHICH IS THE THIRD SHAPE AND NOT A TYPO AT
			// ALL. This mod's recipe declares no `Category`, so it is
			// `crafting`, and the library rejects a fluid there at the list
			// rather than letting the engine refuse the prototype later
			// (FkRecipes go/ingredientlist.go:145-159, and the sentence at
			// :826). Every real game has `water`, so this is the reason a
			// player reaches after "no item or fluid is named": the field
			// takes an ingredient, water IS one, and what cannot be used
			// names the category rather than the spelling.
			text: "1 water",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("1 water"): water is a fluid, and a recipe in the crafting ` +
				`category takes items only`,
			want: vanilla,
		},
		{
			// THE ROW THAT TELLS THE DECLARED LIST FROM THE `vanilla` PRESET,
			// which the three above cannot. [Plan] declares this field's
			// default AS `RecipePlan(RecipeVanilla)`, so in a game that has
			// every rung the author's declaration and the preset are the same
			// three pairs and a fallback onto either would pass those rows. In
			// a game with no `transport-belt` they part: the DECLARATION
			// carries its ladders, so its third one falls to `iron-plate`,
			// merges into the first and says so. A rendered preset -- the flat
			// three names the library would have stored had it emitted the list
			// as a `default_value` -- has no third ladder, so it would come out
			// as two pairs and say nothing.
			//
			// THE TEXT IS THE FIRST ROW'S, DELIBERATELY: the only thing this
			// row moves is the game.
			text: "3 tungsten-plate",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("3 tungsten-plate"): no item or fluid is named tungsten-plate`,
			stocked: noBelt,
			want:    []ingredientPair{{"iron-plate", 6}, {"iron-gear-wheel", 2}},
			extra:   mergeLine("4 plus 2 is 6"),
		},
	} {
		w := everythingWorld().
			withStartup(SettingRecipeCost, RecipeCustom).
			withStartup(SettingRecipeIngredients, tc.text)
		if tc.stocked != nil {
			w = w.withItems(tc.stocked...)
		}
		ops, err := Plan().PlanData(w)
		if err != nil {
			t.Errorf("the text %q took the load down: a text a player typed "+
				"may not refuse a load a player who typed nothing would have "+
				"got, and this fixture has every rung of every ladder: %v",
				tc.text, err)
			continue
		}
		protos, logs := extendsOf(t, ops)
		checkIngredients(t, "the text "+strconv.Quote(tc.text),
			protoOf(t, protos, "recipe", PartName), tc.want)
		// THE WHOLE STREAM IN ORDER, which is what the fourth row needs and the
		// first three lose nothing by: the fallback line comes first, because
		// the text is read before the declaration it falls back to is walked,
		// and then that walk says whatever THIS game made it say.
		wantLogs := []string{fallbackLine(tc.reason)}
		if tc.extra != "" {
			wantLogs = append(wantLogs, tc.extra)
		}
		if !reflect.DeepEqual(logs, wantLogs) {
			t.Errorf("the text %q said\n got  %q\n want %q",
				tc.text, logs, wantLogs)
		}
	}
}

// checkExactlyOneLog is the assertion the customizer's three log arms share:
// the line is there word for word, and it is the ONLY thing the plan said.
//
// The second half is what a substring search would miss. A plan that emitted
// the right line beside a dropped ingredient, or beside a second copy of
// itself, is not the plan this mod wants -- and in a game that has everything,
// any other line is a degradation with no cause.
func checkExactlyOneLog(t *testing.T, logs []string, want string) {
	t.Helper()
	if len(logs) == 1 && logs[0] == want {
		return
	}
	t.Errorf("the plan's log stream is %q\n want exactly one line, %q", logs, want)
}

// ---------------------------------------------------------------------------
// THE RESEARCH CUSTOMIZER: the fourth value, and the three fields behind it.
//
// `bbb-tech-cost` gained `custom` and `better-belt-balancer-tech-packs`,
// `better-belt-balancer-tech-count` and `better-belt-balancer-tech-seconds`
// arrived beside it, so a player who wants a research cost none of the three
// tiers charges can write one. The same rule the recipe's
// section states holds here: every sentence a PLAYER can be shown is compared
// word for word, because a refusal that stops the load is all they are given to
// fix it with.
//
// WHAT IS DIFFERENT FROM THE RECIPE'S HALF, and it is why these are not the
// same tests twice. A written recipe replaces one field of one prototype; a
// written research cost replaces the unit AND decides where the technology
// hangs, because the prerequisite that used to move with the copied unit has no
// source to move from. So every arm below asserts the PAIR.
// ---------------------------------------------------------------------------

// customUnit is the unit shape the library emits for a written cost: the
// engine's SHORT TUPLE form, `{count, time, ingredients={{name, amount}}}`,
// which is the same shape [unitOf] builds for a fixture technology and the same
// one the tier arms copy verbatim.
func customUnit(count, seconds float64, packs ...ingredientPair) fkrecipes.Value {
	tuples := make([]fkrecipes.Value, 0, len(packs))
	for _, p := range packs {
		tuples = append(tuples, fkrecipes.Arr(fkrecipes.Str(p.name), fkrecipes.Num(p.amount)))
	}
	return fkrecipes.Obj(
		fkrecipes.Pair("count", fkrecipes.Num(count)),
		fkrecipes.Pair("time", fkrecipes.Num(seconds)),
		fkrecipes.Pair("ingredients", fkrecipes.Arr(tuples...)),
	)
}

// checkUnit is the whole-unit comparison the four arms below share.
func checkUnit(t *testing.T, what string, got map[string]fkrecipes.Value, want fkrecipes.Value) {
	t.Helper()
	if !reflect.DeepEqual(got["unit"], want) {
		t.Errorf("%s: the unit is %s\n want %s", what, showValue(got["unit"]), showValue(want))
	}
}

// TestTheCustomResearchCostUntouchedIsTheFallbackUnit is the state a player
// reaches by picking Custom and typing nothing, and it is deliberately the same
// cost `logistics` charges in a stock game.
//
// THE THREE FIELDS DEFAULT TO [FallbackUnit] so that switching to Custom is a
// no-op until the player edits something. That is the opposite of a hidden
// change: the row they just picked prices the research exactly as the row above
// it did, and every edit from there is theirs.
//
// THE PACK TEXT IS UNTOUCHED AND THE TWO NUMBERS ARE STILL READ, which is the
// library's rule and not a shortcut: "THE COUNT AND THE SECONDS ARE ALWAYS
// READ, on both text paths. They are separate settings and the player may have
// moved them whether or not they touched the pack list" (FkRecipes
// go/customize.go:1217). So the log line reports all three even here, where
// nothing was written -- one line, and it is the only thing the plan says.
//
// AND THE PREREQUISITE IS `logistics-3`, which is the [Plan] `Position`
// ladder's first rung: a written cost has no source technology to move with, so
// the arm carries its own placement.
func TestTheCustomResearchCostUntouchedIsTheFallbackUnit(t *testing.T) {
	w := everythingWorld().withStartup(SettingTechCost, TechCustom)
	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, "custom untouched", got,
		customUnit(20, 15, ingredientPair{"automation-science-pack", 1}))
	checkPrereqs(t, "custom untouched", got, TechLogistics3)
	checkExactlyOneLog(t, logs,
		"fkrecipes: bbb-balancer takes its research cost from "+
			"better-belt-balancer-tech-packs: count 20, time 15, "+
			"packs 1 automation-science-pack")

	// NO max_level, AND IT IS AN ASSERTION RATHER THAN AN ABSENCE NOBODY
	// LOOKED AT. Under a tier the library copies the source technology's level
	// cap along with its unit ([TestTheCopiedUnitCarriesTheSourceMaxLevel]);
	// under Custom there is no source, so a cap arriving here could only have
	// come from a technology this cost has nothing to do with.
	checkOnly(t, TechName, got,
		"type", "name", "icon", "icon_size", "prerequisites", "unit", "effects", "order")
}

// TestTheCustomResearchCostIsTheThreeSettings is the feature, stated as the
// research a player gets.
//
// ALL THREE FIELDS ARE DRIVEN AT ONCE, which is the one place in this file that
// moves more than one variable, and it is deliberate: the three are one cost.
// What makes it readable is that no two of them can be confused -- 50 units, 20
// seconds and a two-pack list, against defaults of 20, 15 and one pack -- so a
// planner that dropped any one of them fails on that field by name.
//
// THE PACK ORDER IS THE TYPED ORDER, for the reason a written recipe's is: the
// list a player writes is the list the research screen shows them back.
func TestTheCustomResearchCostIsTheThreeSettings(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingTechCost, TechCustom).
		withStartup(SettingTechPacks, "1 automation-science-pack, 1 logistic-science-pack").
		withNumberStartup(SettingTechCount, 50).
		withNumberStartup(SettingTechSeconds, 20)
	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, "a written research cost", got, customUnit(50, 20,
		ingredientPair{"automation-science-pack", 1},
		ingredientPair{"logistic-science-pack", 1}))
	checkPrereqs(t, "a written research cost", got, TechLogistics3)
	// A LITERAL, not the constants spliced together: this section compares
	// every line a player is shown word for word, and a renamed constant has to
	// fail this test rather than travel through it.
	checkExactlyOneLog(t, logs,
		"fkrecipes: bbb-balancer takes its research cost from "+
			"better-belt-balancer-tech-packs: count 50, time 20, "+
			"packs 1 automation-science-pack, 1 logistic-science-pack")
}

// TestThePositionLadderStepsDownAndThenLetsGo is the placement half of the
// custom arm, driven through every rung.
//
// THE COST DOES NOT MOVE WITH IT, which is the difference from
// [TestALadderStepsDownAndTakesThePrerequisiteWithIt] and the reason both
// exist. A tier's ladder steps down to a technology whose UNIT is then copied,
// so the pair moves together; this ladder decides a place in the tree only, and
// the price is the player's in all three worlds.
//
// THE LAST CASE IS THE ONE WITH NO ROWS LEFT. A prerequisite naming a
// technology nobody defined is a load error rather than a cost, so the library
// emits none and says so -- and this mod's research still exists, still costs
// what the player wrote, and simply hangs off nothing.
func TestThePositionLadderStepsDownAndThenLetsGo(t *testing.T) {
	base := everythingWorld().withStartup(SettingTechCost, TechCustom)
	l1, _ := base.tech(TechLogistics)
	l2, _ := base.tech(TechLogistics2)
	want := customUnit(20, 15, ingredientPair{"automation-science-pack", 1})
	const costLine = "fkrecipes: bbb-balancer takes its research cost from " +
		"better-belt-balancer-tech-packs: count 20, time 15, " +
		"packs 1 automation-science-pack"

	for _, tc := range []struct {
		name  string
		world fixtureWorld
		after string
	}{
		{"logistics-3 absent", base.withTechs(l1, l2), TechLogistics2},
		{"only logistics is left", base.withTechs(l1), TechLogistics},
	} {
		protos, logs := extendsOf(t, dataOps(t, tc.world))
		got := protoOf(t, protos, "technology", TechName)
		checkPrereqs(t, tc.name, got, tc.after)
		checkUnit(t, tc.name, got, want)
		checkExactlyOneLog(t, logs, costLine)
	}

	// NO LOGISTICS CHAIN AT ALL. A tier would land on [FallbackUnit] here; the
	// custom arm does not, because its cost never came from a technology.
	protos, logs := extendsOf(t, dataOps(t, base.withTechs()))
	got := protoOf(t, protos, "technology", TechName)
	checkUnit(t, "no logistics at all", got, want)
	if v, ok := got["prerequisites"]; ok {
		t.Errorf("the technology carries prerequisites %s in a game with no "+
			"logistics technology at all: that names something nobody defined",
			showValue(v))
	}
	// THE ORDER IS THE COST AND THEN THE PLACEMENT, which is the library's own
	// ("what it costs, then where it hangs", FkRecipes go/data.go:632) and is
	// asserted rather than tolerated: a transcript a maintainer reads top to
	// bottom is the only place these two lines are ever seen together.
	if !reflect.DeepEqual(logs, []string{
		costLine,
		"fkrecipes: bbb-balancer: none of logistics-3, logistics-2, logistics " +
			"is present, so the technology has no prerequisite",
	}) {
		t.Errorf("the plan's log stream is %q\n want the cost and then the "+
			"dropped ladder", logs)
	}
}

// TestAPackTextTheGameCannotAnswerFallsBackAndSaysSo is the recipe field's
// fallback contract asked of the OTHER text field: four texts a player is
// likeliest to get wrong, and for each one the research this mod's own
// declaration prices, plus the line that says the typed value was set aside.
//
// RENAMED FOR THE REASON THE RECIPE FIELD'S TWIN WAS. FkRecipes 2e5f779 made
// these four fall back rather than refuse, and the header there carries the
// lock-out measurement that decided it.
//
// WHAT THE DECLARATION IS HERE. [FallbackUnit] -- 20 units of 15 seconds paid
// in one `automation-science-pack`, which is base's own `logistics` cost -- and
// it is the same three numbers `better-belt-balancer-tech-count`,
// `-tech-seconds` and `-tech-packs` declare as their defaults. So the assertion
// is exactly what [TestTheCustomResearchCostUntouchedIsTheFallbackUnit] gets
// for a player who typed NOTHING: a text this mod cannot use prices the
// research the way an empty field does.
//
// TWO LINES AND NOT ONE, IN THIS ORDER, and both are asserted. The library
// reports the fallback first and the cost it went on to charge second, which is
// the transcript a maintainer reads top to bottom: what went wrong, then what
// the player got. The stream is compared WHOLE for
// [TestThePositionLadderStepsDownAndThenLetsGo]'s reason -- that is the only
// place two lines are ever seen together -- and it is what keeps the pair from
// being satisfied by a plan that logged the error and then charged something
// else.
//
// THE FIRST IS THE ONE THAT SEPARATES THIS FIELD FROM THE RECIPE'S. Both take
// the same language and the same names, and `1 iron-plate` is a perfectly good
// entry in one and unusable in the other -- because the engine takes tool-type
// items in a research unit and nothing else ("Invalid research unit
// (iron-plate). Research unit(s) can only be tool type items at the moment",
// measured by the library). The sentence says which of the two fields the
// player is standing in rather than repeating the engine's.
//
// THE SECOND IS THE MISTAKE THIS MOD'S OWN LABELS INVITE, exactly as the recipe
// field's display-name case is: `automation science pack` is what the research
// screen calls it, and the language answers with the internal name.
//
// THE THIRD IS THE WORD THE RECIPE FIELD ACCEPTS AND THIS ONE DOES NOT. `none`
// is a recipe with no ingredients, which is a thing; a research with no science
// pack is one a player finishes by opening the screen, and the library will not
// emit it on this mod's behalf.
//
// THE FOURTH IS THE FLUID, and it is the recipe field's fluid case with a
// different second clause: there the category rule rejects it, here research
// takes science packs and nothing else. Every real game has `water`, the two
// fields take one language, and a player who has used the first field is the
// player likeliest to reach for the same name in the second; the fixture stocks
// `water` for exactly this row and the recipe's.
//
// THE FALLBACK IS NOT TOTAL AND NONE OF THESE FOUR ROWS IS THE EXCEPTION,
// because this game has the science pack the declaration names.
// [TestAFallbackThisModsOwnPackListCannotPayForStillRefuses] is the game that
// does not.
func TestAPackTextTheGameCannotAnswerFallsBackAndSaysSo(t *testing.T) {
	// The cost the fallback lands on, said once and not derived: it is the same
	// sentence [TestTheCustomResearchCostUntouchedIsTheFallbackUnit] pins as
	// the only thing an untouched Custom arm says.
	const costLine = "fkrecipes: bbb-balancer takes its research cost from " +
		"better-belt-balancer-tech-packs: count 20, time 15, " +
		"packs 1 automation-science-pack"

	for _, tc := range []struct {
		text   string
		reason string
	}{
		{
			text: "1 water",
			reason: `better-belt-balancer-tech-packs, entry 1 ("1 water"): ` +
				`water is a fluid, and research takes science packs only`,
		},
		{
			text: "1 iron-plate",
			reason: `better-belt-balancer-tech-packs, entry 1 ` +
				`("1 iron-plate"): iron-plate is an item, not a science pack`,
		},
		{
			text: "1 automation science pack",
			reason: `better-belt-balancer-tech-packs, entry 1 ` +
				`("1 automation science pack"): no science pack is named ` +
				`"automation science pack"; did you mean automation-science-pack`,
		},
		{
			text: "none",
			reason: "better-belt-balancer-tech-packs: research takes at " +
				"least one science pack",
		},
	} {
		w := everythingWorld().
			withStartup(SettingTechCost, TechCustom).
			withStartup(SettingTechPacks, tc.text)
		ops, err := Plan().PlanData(w)
		if err != nil {
			t.Errorf("the pack text %q took the load down in a game that has "+
				"this mod's own science pack: a text a player typed may not "+
				"refuse a load a player who typed nothing would have got: %v",
				tc.text, err)
			continue
		}
		protos, logs := extendsOf(t, ops)
		checkUnit(t, "the pack text "+strconv.Quote(tc.text),
			protoOf(t, protos, "technology", TechName),
			customUnit(20, 15, ingredientPair{"automation-science-pack", 1}))
		if want := []string{fallbackLine(tc.reason), costLine}; !reflect.DeepEqual(logs, want) {
			t.Errorf("the pack text %q said\n got  %q\n want %q",
				tc.text, logs, want)
		}
	}
}

// TestAFallbackThisModsOwnPackListCannotPayForStillRefuses is the BOUNDARY of
// the contract the two tests above pin, and it is the library's own claim
// rather than this mod's inference: "A VALUE THE PLAYER TYPES NEVER INTRODUCES
// A REFUSAL A PLAYER WHO TYPED NOTHING WOULD NOT ALSO HAVE HIT; AN INPUT THE
// AUTHOR DECLARES STILL REFUSES" (FkRecipes go/customize.go:857, playerFallback).
//
// THE WORLD IS THE ONE THAT MAKES IT SHARP. `1 water` is the same unusable text
// the first row of the test above drives, and the game is the same one except
// that its science packs are `logistic-science-pack` alone. So the fallback
// lands on this mod's declared list, one `automation-science-pack`, which has
// nothing to land on either, and the load stops. It stops for a player who typed
// NOTHING in that same game too, which is the whole of what makes this refusal
// legitimate where a refusal on the typed text would not be:
// [TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused] is that player, and
// README says it in their terms.
//
// AND THE REFUSAL CARRIES A SECOND SENTENCE THE UNTOUCHED ONE DOES NOT. The
// library appends what became of the stored value, so a player is not left
// reading a refusal about science packs while wondering what happened to the
// text they typed. That sentence is the whole difference between this refusal
// and the untouched arm's, and asserting the WHOLE string is what pins it: a
// library that fell back silently and then refused would produce the shorter
// sentence and fail here.
func TestAFallbackThisModsOwnPackListCannotPayForStillRefuses(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingTechCost, TechCustom).
		withStartup(SettingTechPacks, "1 water").
		withTools("logistic-science-pack")

	_, err := Plan().PlanData(w)
	if err == nil {
		t.Fatal("a game with no automation science pack in it priced this " +
			"mod's declared research anyway: the fallback landed on a pack " +
			"nobody defined, which is a load error with this mod's name on it")
	}
	const want = "fkrecipes: the technology bbb-balancer has no science pack " +
		"the game has; research takes at least one. The stored value of " +
		"better-belt-balancer-tech-packs could not be used, so the mod's own " +
		"declaration applied; correcting it under Settings > Mod settings > " +
		"Startup is what a player can change here."
	if err.Error() != want {
		t.Errorf("the refusal is\n got  %q\n want %q", err.Error(), want)
	}
}

// TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused is the OTHER side of
// the tool question, and it is not the same as the refusal above.
//
// A player who typed nothing gets the DECLARED pack list, whose ladder is
// walked through `ToolExists` and whose rungs are DROPPED rather than refused
// when the game has none of them (FkRecipes go/data.go:1167,
// resolvePackLadders) -- the same tolerance every ingredient ladder in this
// package has.
//
// THE GAME HERE HAS SCIENCE PACKS AND HAS THIS ONE AS AN ORDINARY ITEM, which
// is what makes the question sharp rather than "a game with no packs at all":
// `logistic-science-pack` is a tool, `automation-science-pack` is in the items
// and nowhere else, and a ladder asking ItemExists would sail through. That is
// exactly the shape FkRecipes' own go/world.go warns a consumer's fixture
// about: "a fixture that names a science pack only in its items will see every
// hand-rolled research cost lose its packs".
//
// AND WHAT A LOST LAST PACK GETS IS A REFUSAL, not a free research. It is the
// same sentence [TestAReachedFallbackWithNoPackInTheGameIsRefused] pins for the
// fallback, which is the point: a lost pack reads the same however the cost was
// chosen. The drop itself is a log line the library writes on the way, and it
// is not asserted here because a refused plan hands back the refusal and no
// ops at all.
func TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingTechCost, TechCustom).
		withItems(append(ladderVocabulary(), "automation-science-pack")...).
		withTools("logistic-science-pack")

	_, err := Plan().PlanData(w)
	if err == nil {
		t.Fatal("a game holding this mod's science pack as an ordinary item " +
			"priced its research anyway: the pack ladder is asking ItemExists, " +
			"and the engine takes tool-type items in a unit and nothing else")
	}
	want := "fkrecipes: the technology bbb-balancer has no science pack the " +
		"game has; research takes at least one"
	if err.Error() != want {
		t.Errorf("the refusal reads\n got  %q\n want %q", err.Error(), want)
	}
}

// TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo is the research side
// of [TestAnEditedTextUnderAPresetIsIgnoredAndTheLogSaysSo], and it pins two
// claims more than that one does: that under a tier NONE of the three fields
// enters the unit, and that each one the player moved SAYS SO. Each of the two
// numbers draws
// noteIgnoredNumber's sentence (FkRecipes go/customize.go:1204, called at
// go/data.go:654 and 655) and the pack text draws noteIgnoredText's
// (go/customize.go:1171, called at go/data.go:656), in the order the library
// calls them: the count, then the seconds, then the text. Under a tier the unit
// is copied from the source and no number of this mod's enters it, and a player
// who drags the count slider, sees the research unchanged and finds nothing in
// the log has been told their edit landed when it did not.
//
// EDITED MEANS "NOT THE DECLARED DEFAULT" FOR A NUMBER, and that is the half
// this mod is on the hook for rather than the library. A numeric setting has no
// reserved word standing for the author's answer the way a text field's
// `default` does, so the library compares the stored number against the
// DECLARED one -- and the number it compares against is the one [Plan] writes
// down, 20 for the count and 15 for the seconds (plan.go:284 and 285). A field
// sitting at its declared default is indistinguishable from an untouched one
// and draws no line, which is why the three-line arm drives 50 and 20 and not
// 20 and 15.
//
// THE STREAM IS ASSERTED WHOLE AND IN ORDER on every arm. "One of these is
// somewhere in the log" would be passed by a library that emitted the count's
// line twice, or the seconds' before the count's, and the order is the one a
// player reads down.
//
// WHAT MAKES THIS NOT VACUOUS is the unit beside the lines. Every value driven
// here is one the fixture's logistics-2 does not charge -- it charges 200 units
// of 30 seconds and one logistic science pack (world_test.go's everythingWorld;
// the real game's charges one of each, which is what the gate's arm drives
// against), against 50 or 20 units, 20 or 15 seconds and two automation packs,
// so the two declared defaults are not that tier's numbers either, and the
// arms that park a field at its default lose no teeth:
// a planner that took any one of these under a tier fails the unit comparison
// by that field, and not only loses a sentence in the log.
func TestAnEditedPackTextUnderATierIsIgnoredAndTheLogSaysSo(t *testing.T) {
	// Literals, for the reason every pinned sentence in this file is one.
	const (
		countLine = "fkrecipes: better-belt-balancer-tech-count is edited, but " +
			"bbb-tech-cost is not on custom, so the number is ignored"
		secondsLine = "fkrecipes: better-belt-balancer-tech-seconds is edited, " +
			"but bbb-tech-cost is not on custom, so the number is ignored"
		packsLine = "fkrecipes: better-belt-balancer-tech-packs is edited, but " +
			"bbb-tech-cost is not on custom, so the text is ignored"
	)
	for _, tc := range []struct {
		name           string
		count, seconds float64
		want           []string
	}{
		{"both numbers moved", 50, 20, []string{countLine, secondsLine, packsLine}},
		{"the count at its declared default", 20, 20, []string{secondsLine, packsLine}},
		{"the seconds at its declared default", 50, 15, []string{countLine, packsLine}},
	} {
		w := everythingWorld().
			withStartup(SettingTechCost, TechLogistics2).
			withStartup(SettingTechPacks, "2 automation-science-pack").
			withNumberStartup(SettingTechCount, tc.count).
			withNumberStartup(SettingTechSeconds, tc.seconds)
		l2, _ := w.tech(TechLogistics2)
		protos, logs := extendsOf(t, dataOps(t, w))
		got := protoOf(t, protos, "technology", TechName)

		checkUnit(t, tc.name+" under logistics-2", got, *l2.unit)
		checkPrereqs(t, tc.name+" under logistics-2", got, TechLogistics2)
		if !reflect.DeepEqual(logs, tc.want) {
			t.Errorf("%s: the plan's log stream is\n got  %q\n want the "+
				"ignored fields in the library's order, %q", tc.name, logs, tc.want)
		}
	}
}

// TestAnUnreadableResearchNumberTakesTheDeclaredDefault is the two numbers'
// version of [TestAnUnreadableCustomTextTakesTheDeclaredList], and there are
// two shapes of unreadable rather than one because a number has both: a row
// MISSING from a hand-edited mod-settings.dat, and a row PRESENT under the
// wrong type, which is what another mod redefining this mod's setting as text
// would hand the planner. The library reads both through one function
// (FkRecipes go/customize.go:1302, readNumber) and both take the declared
// default with one sentence, which is pinned here word for word.
//
// THE STREAM IS ASSERTED WHOLE, two lines in order: the degradation first,
// because the number is read before the cost line is written, and then the
// cost line carrying the DECLARED number -- which is the half with teeth, since
// the fixture supplies nothing and the 20 or the 15 can only have come from
// [Plan]'s own declaration.
func TestAnUnreadableResearchNumberTakesTheDeclaredDefault(t *testing.T) {
	want := customUnit(20, 15, ingredientPair{"automation-science-pack", 1})
	const costLine = "fkrecipes: bbb-balancer takes its research cost from " +
		"better-belt-balancer-tech-packs: count 20, time 15, " +
		"packs 1 automation-science-pack"
	for _, tc := range []struct {
		name  string
		world fixtureWorld
		line  string
	}{
		{
			"the count row is missing",
			everythingWorld().withStartup(SettingTechCost, TechCustom).
				withoutNumberStartup(SettingTechCount),
			"fkrecipes: the setting better-belt-balancer-tech-count was not " +
				"readable, so its default applies",
		},
		{
			"the seconds row holds text",
			everythingWorld().withStartup(SettingTechCost, TechCustom).
				withRawStartup(SettingTechSeconds, fkrecipes.Str("15")),
			"fkrecipes: the setting better-belt-balancer-tech-seconds was not " +
				"readable, so its default applies",
		},
	} {
		protos, logs := extendsOf(t, dataOps(t, tc.world))
		got := protoOf(t, protos, "technology", TechName)
		checkUnit(t, tc.name, got, want)
		checkPrereqs(t, tc.name, got, TechLogistics3)
		if !reflect.DeepEqual(logs, []string{tc.line, costLine}) {
			t.Errorf("%s: the plan's log stream is %q\n want the degradation "+
				"and then the cost, %q", tc.name, logs, []string{tc.line, costLine})
		}
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
		t.Errorf("%s: the prerequisites are %s and the cost came from %q",
			what, showValue(v), want)
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
