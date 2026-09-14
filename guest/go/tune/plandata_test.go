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
// SIX AND EXACTLY THE DROPDOWN'S SIX. The library adds no value of its own to a
// dropdown, so what the option list holds and what this loop drives are one
// list; the text field beside it is not a row here, and the customizer's arms
// are their own, in this file and in the gate alike.
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
// source answered" (go/data.go, the CostBy arm) -- and the design record calls it the
// answer to this mod's ask by name, in a row of FkRecipes'
// agents/customizer-design.md whose decision ends: the Fallback is resolved
// only when used. The fallback's NUMBERS are still checked eagerly, in the plan
// walk with no World in hand (go/customize.go, validateCostChoices), which is the right split: a
// count of zero is this mod's mistake and is knowable without asking the game
// anything.
//
// SO A GAME THAT HAS NOT GOT THIS MOD'S OWN SCIENCE PACK LOADS, as long as the
// tier it is on is priced in a pack it HAS. The game here is `logistics-2` --
// which the fixture prices in one `logistic-science-pack` -- in a game whose
// only tool is that pack, so [FallbackUnit]'s `automation-science-pack` is a
// name nothing in this world answers for and NOTHING ASKS.
//
// THE WORLD IS NARROWER THAN IT WAS AND THE REASON IS A SECOND PROBE THIS TEST
// DID NOT USED TO MEET. It used to take every tool away, on the reading that a
// source carrying a unit ends the question; the library filters a COPIED unit's
// own packs through `ToolExists` now, so a game with no tool at all loses the
// tier's pack as well and lands on the fallback after all. That is a different
// property, and [TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused] is
// where it is pinned. What is left here is the narrow claim the lazy resolve
// was asked for, and it is sharper for it: the name in the FALLBACK is absent
// from this game and the load is clean.
//
// WHAT SAYS THE SOURCE ARM RAN is the prerequisite (exactly `logistics-2`), the
// unit (the tier's own, which is not [FallbackUnit]'s numbers) and the empty log
// stream.
func TestAnUnreachedFallbacksPackIsNeverProbed(t *testing.T) {
	base := everythingWorld()
	l2, _ := base.tech(TechLogistics2)

	w := base.withStartup(SettingTechCost, TechLogistics2).
		withTools("logistic-science-pack")
	protos, logs := extendsOf(t, dataOps(t, w))
	for _, line := range logs {
		t.Errorf("a game whose logistics-2 is priced in a pack it has degraded "+
			"over a fallback nothing reached: %s", line)
	}
	got := protoOf(t, protos, "technology", TechName)
	checkPrereqs(t, "an unreached fallback", got, TechLogistics2)
	if !reflect.DeepEqual(got["unit"], *l2.unit) {
		t.Errorf("the unit is %s and logistics-2 charges %s",
			showValue(got["unit"]), showValue(*l2.unit))
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
	// THE SENTENCE NAMES THE RUNGS IT TRIED, which it did not before this
	// library round: one name here, because this mod declares its fallback
	// pack with no ladder under it.
	want := "fkrecipes: the technology bbb-balancer has no science pack the " +
		"game has; research takes at least one, and none of " +
		"automation-science-pack is a science pack here"
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
// THE CUSTOMIZER: the text field beside the dropdown, and the text is the
// switch.
//
// `better-belt-balancer-recipe-ingredients` arrived beside `bbb-recipe-cost`
// and the dropdown's six values did not move, so a player who wants a recipe
// none of the six presets is writes one and a player who does not never sees a
// difference. WHAT DECIDES IS THE TEXT ITSELF: while it says `default` the
// dropdown applies exactly as it always did, and anything else is what the
// recipe is made of, with the choice it set aside named in the log.
//
// EVERY SENTENCE A PLAYER CAN BE SHOWN IS COMPARED WORD FOR WORD rather than by
// substring: a refusal that stops the load is the only thing they are given to
// fix it with, so its wording is as much this mod's surface as the recipe is.
// ---------------------------------------------------------------------------

// theDefaultWord is FkRecipes' reserved word for "the row beside me decides",
// and where there is no row beside it, "the mod's own declared list with its
// ladders". The library keeps it unexported (go/ingredientlist.go, `defaultWord`)
// and documents it in docs/ingredient-list.md, so a consumer writes it out; it
// is written out ONCE here, because the fixture, these tests and the engine
// gate all have to say the same seven letters.
const theDefaultWord = "default"

// TestTheDefaultWordLetsTheDropdownDecide is the state every player who never
// opens the Startup tab is in, and the whole reason the text is the switch.
//
// THE WORD IS NOT A LIST AND THAT IS THE WHOLE DESIGN.
// `better-belt-balancer-recipe-ingredients` ships holding `default`, and the
// library reads the word as "the row beside me decides" rather than as a
// rendering of anything -- so the dropdown answers, in this release and in
// every later one, and adopting the customizer cost a player on `cheap`
// nothing at all.
//
// THE DROPDOWN IS ON `cheap` AND NOT ON ITS OWN DEFAULT, which is what makes
// this test able to fail. On `vanilla` the preset's list and this field's
// declared list are the same three pairs, so a planner that ignored the
// dropdown and rendered the declaration would pass; `cheap` is two pairs and
// neither of them is vanilla's.
//
// AND IT DRAWS NO LOG LINE, which is the library's stated rule for this arm:
// nothing was overridden, so there is nothing to narrate. The assertion is the
// EMPTY log stream rather than the absence of one particular sentence, because
// a line here would be the library narrating an untouched field on every load
// of every game.
//
// THE THREE SPELLINGS ARE THE LANGUAGE'S, not this mod's guesses. Surrounding
// space is trimmed and one trailing comma is tolerated, so " default " and
// "default," are the word too -- and a spelling read as a LIST instead would
// hand the player a recipe made of one ingredient called `default`, or take the
// load down.
func TestTheDefaultWordLetsTheDropdownDecide(t *testing.T) {
	for _, text := range []string{theDefaultWord, " default ", "default,"} {
		w := everythingWorld().
			withStartup(SettingRecipeCost, RecipeCheap).
			withStartup(SettingRecipeIngredients, text)
		protos, logs := extendsOf(t, dataOps(t, w))
		for _, line := range logs {
			t.Errorf("the text %q is the default word and the plan said "+
				"something about it: %s", text, line)
		}
		checkIngredients(t, "cheap under "+strconv.Quote(text),
			protoOf(t, protos, "recipe", PartName),
			[]ingredientPair{{"iron-plate", 2}, {"transport-belt", 1}})
	}
}

// TestAnUnreadableTextLetsTheDropdownDecide is the OTHER untouched path, and it
// is not the same one.
//
// A setting the planner cannot read AT ALL is a hand-edited mod-settings.dat
// with the row missing: the engine writes every setting's current value into
// that file, untouched defaults included, so no player produces this. The
// library behaves exactly as it does for the word -- the dropdown decides --
// and SAYS SO, with the sentence it uses for every unreadable setting. That
// line is the whole difference between this arm and the word above. One is a
// player's choice; the other is a file somebody edited by hand, and it should
// not pass in silence.
//
// THE DROPDOWN IS ON `cheap` FOR [TestTheDefaultWordLetsTheDropdownDecide]'s
// REASON: on `vanilla` a planner that fell back to this field's own declared
// list would be indistinguishable from one that asked the dropdown.
//
// THE FIXTURE HAS TO ANSWER THE EMITTED NAME FOR THIS TO MEAN ANYTHING.
// `better-belt-balancer-recipe-ingredients` is generated, so the name the
// library asks `StartupSetting` for is the PREFIXED one a player's file carries.
// A fixture keyed on anything else would answer absent for every arm of the
// customizer, and this is the one test that would still pass.
func TestAnUnreadableTextLetsTheDropdownDecide(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeCost, RecipeCheap).
		withoutStartup(SettingRecipeIngredients)
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "an unreadable text",
		protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{"iron-plate", 2}, {"transport-belt", 1}})
	// THE SENTENCE IS A LITERAL, not the constants spliced together: this
	// section compares every line a player is shown word for word, and a
	// renamed constant has to fail this test rather than travel through it.
	checkExactlyOneLog(t, logs,
		"fkrecipes: the setting better-belt-balancer-recipe-ingredients was not "+
			"readable, so its default applies")
}

// TestATypedTextTakesTheRecipeOverFromTheDropdown is the feature, stated as the
// recipe a player gets.
//
// THE ORDER IS THE TYPED ORDER, which is why the comparison is a slice: the
// list a player writes is the list they see in the crafting tooltip, and a
// planner that sorted it would be quietly rewriting their recipe.
//
// THE OVERRIDE IS TOTAL, which is the ingredient field's rule and not the
// research fields': the WHOLE list is the player's, and the preset contributes
// nothing to it. That is why the line ends `is set aside` where the research
// line ends `supplies what the settings leave at default`.
//
// THE LOG LINE IS PINNED WORD FOR WORD, AND IT IS THE CANONICAL RENDERING
// RATHER THAN THE TEXT AS TYPED: a player who wrote `iron-plate x3` reads back
// `3 iron-plate` and learns the form the library would have written. It names
// the RECIPE, the SETTING and the DROPDOWN by their EMITTED names -- the recipe
// and the dropdown unprefixed because they are Legacy, the text setting
// prefixed because it is generated -- so the line points at the two rows in the
// menu a player would go and look at.
//
// THE DROPDOWN IS LEFT ON ITS OWN DEFAULT HERE, so the clause names `vanilla`,
// which is what a player who typed into the field and touched nothing else
// reads. [TestATextUnderAPresetTakesTheRecipeOverAndNamesWhatItSetAside] is the
// same rule with a preset under it and a list that can tell the two apart.
func TestATypedTextTakesTheRecipeOverFromTheDropdown(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeIngredients, "3 iron-plate, 1 splitter")
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "a written recipe", protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{"iron-plate", 3}, {"splitter", 1}})
	checkExactlyOneLog(t, logs,
		"fkrecipes: bbb-balancer-part takes its ingredients from "+
			"better-belt-balancer-recipe-ingredients: 3 iron-plate, 1 splitter"+
			"; the bbb-recipe-cost choice vanilla is set aside")
}

// TestATextNamingTheBalancerPartItselfEmitsItAndSaysSo is the round-1 open item
// closed from the library's side, pinned here as what a player now reads.
//
// THE SHAPE. `2 bbb-balancer-part` typed into the text field asks for a recipe
// whose only ingredient is the thing it makes. The library
// resolves a typed name against an overlay that holds THIS PLAN'S OWN ITEMS on
// top of the game's, so `bbb-balancer-part` is present and the list emits; a
// declared LADDER cannot reach the same place, because its rungs are probed
// against the real world where this mod's item does not exist yet.
//
// WHAT MOVED. Round 1 recorded this as accepted and emitted, with the ordinary
// "takes its ingredients from" line and NOTHING SAID ABOUT THE SELF-PRODUCT
// FACT, so the player got a recipe nothing can craft and no line about that;
// the round's own note put the question to the library rather than to this mod.
// FkRecipes 1f8e363 answers it with a second line, and this test is what holds
// the library to it here.
//
// IT IS NOT A REFUSAL AND MUST NOT BECOME ONE. Base 2.0.77 ships
// `kovarex-enrichment-process`, which takes 40 `uranium-235` and gives back 41,
// so a recipe naming its own product is a shape the game itself has. The load
// still completes, which `test/check-datastage.py`'s `recipe-self-product` arm
// says on a real engine and this test cannot.
//
// THE STREAM IS COMPARED WHOLE AND IN ORDER, which is the merge arm's rule and
// is what makes the position part of the claim: the "takes its ingredients
// from" line is written as the text is read, and the self-product line is
// evaluated on the FINAL list, so it lands after it. A slice comparison fails
// on an order swap where a membership test would not.
func TestATextNamingTheBalancerPartItselfEmitsItAndSaysSo(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeIngredients, "2 "+PartName)
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "the part as its own ingredient",
		protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{PartName, 2}})
	// LITERALS, for the reason every other sentence in this section is one: a
	// renamed constant has to fail this test rather than travel through it.
	// THE TWO LINES NAME THE RECIPE DIFFERENTLY AND THIS FIXTURE CANNOT TELL,
	// which is why it is said here. The first uses EMITTED names throughout
	// (FkRecipes go/customize.go builds it from `r.emittedName` and
	// `s.emittedName`); the second's SUBJECT is the recipe as the author
	// DECLARED it and only its product is emitted (go/data.go's `addRecipe`
	// says so in as many words). Here the recipe is Legacy and so is the ITEM
	// it produces ([PartName] for both), so all three readings land on the one
	// string and the distinction is carried by this comment rather than by the
	// assertion.
	want := []string{
		"fkrecipes: bbb-balancer-part takes its ingredients from " +
			"better-belt-balancer-recipe-ingredients: 2 bbb-balancer-part" +
			"; the bbb-recipe-cost choice vanilla is set aside",
		"fkrecipes: bbb-balancer-part: bbb-balancer-part is in the list and " +
			"is also what this recipe makes, so nothing can craft the first " +
			"one unless something else produces it",
	}
	if !reflect.DeepEqual(logs, want) {
		t.Errorf("the plan's log stream is\n got  %q\n want %q", logs, want)
	}
}

// TestATextUnderAPresetTakesTheRecipeOverAndNamesWhatItSetAside is the shape
// this round exists to reach, and it is the exact inverse of what stood here.
//
// WHAT THIS TEST USED TO SAY. Until fix round 2 the text applied only while the
// dropdown said `custom`, so a player standing on `cheap` who typed a recipe
// got `cheap` and one log line telling them their edit was ignored. That is a
// field a player edits where nothing happens, and the round's answer was not to
// narrate it better but to make it impossible: THE TEXT IS THE SWITCH, so a
// non-default text under any preset is live and the preset is what is set
// aside. The sentence this test pinned exists nowhere any more, in this mod or
// in the library, because there is no state left for it to be about.
//
// WHAT MAKES THIS NOT VACUOUS is the ingredient list beside the line. `cheap`
// and the text name the same item at DIFFERENT AMOUNTS, so a planner that still
// preferred the preset would emit two iron plates and a belt and fail the
// comparison rather than only losing a sentence in the log.
//
// AND THE CLAUSE NAMES THE CHOICE, not the setting: `cheap` is the row the
// player is standing on, and it is the thing they would go looking for when the
// recipe is not what the dropdown says. A clause naming `bbb-recipe-cost`
// instead would point at the row rather than at the value it holds.
func TestATextUnderAPresetTakesTheRecipeOverAndNamesWhatItSetAside(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingRecipeCost, RecipeCheap).
		withStartup(SettingRecipeIngredients, "1 iron-plate")
	protos, logs := extendsOf(t, dataOps(t, w))

	checkIngredients(t, "a typed text under cheap",
		protoOf(t, protos, "recipe", PartName),
		[]ingredientPair{{"iron-plate", 1}})
	// A literal for the reason the unreadable arm's is: a renamed constant
	// must fail this sentence rather than be carried by it.
	checkExactlyOneLog(t, logs,
		"fkrecipes: bbb-balancer-part takes its ingredients from "+
			"better-belt-balancer-recipe-ingredients: 1 iron-plate"+
			"; the bbb-recipe-cost choice cheap is set aside")
}

// fallbackLine is the ONE line the library logs when it sets a player's text
// aside, whose moving part is the reason inside it.
//
// LITERALS AGAIN, for the reason [mergeLine] is one. The prefix carries the
// severity in the text because Factorio's `log()` has one channel and no level,
// and the tail is the only instruction a player gets: this is the sentence that
// has to survive a refactor unchanged.
//
// AND A RECIPE'S INGREDIENT TEXT CARRIES ONE SENTENCE MORE THAN A PACK TEXT
// DOES, which is why this takes an argument. Changing a recipe empties an
// assembling machine's input slots of anything the new list does not use -- up
// to eighty items destroyed outright rather than spilled on the ground, which
// is a thing worth saying before a player acts on it -- and repricing a
// research destroys nothing, so the pack field's line stops one sentence
// earlier. It is the same constant the library writes into the RECIPE'S OWN
// tooltip, which is what keeps the two from drifting apart.
func fallbackLine(reason string) string {
	return packFallbackLine(reason) +
		" Changing a recipe empties an assembling machine's input slots of " +
		"anything the new list does not use."
}

// packFallbackLine is the same line without the recipe's second sentence, which
// is what a science-pack text gets.
func packFallbackLine(reason string) string {
	return "fkrecipes: ERROR: " + reason +
		". The mod loaded with its own default instead; fix the text under " +
		"Settings > Mod settings > Startup, then restart."
}

// TestATextTheGameCannotAnswerFallsBackAndSaysSo pins the three texts a player
// is likeliest to get wrong, and what the mod does with each: it applies THE
// PRESET THE DROPDOWN IS ON, writes one line naming the setting and the reason,
// and says so in the recipe's own tooltip. A FOURTH ROW drives the first of
// those texts a second time in a game with no `transport-belt`, and it is there
// because the other three cannot tell a preset that carries its ladders from a
// flat list: in a game that has everything the two are the same two pairs. Its
// own comment says how they part.
//
// EVERY ROW STANDS ON `cheap` AND NOT ON THE DROPDOWN'S OWN DEFAULT, and that
// is what makes the headline claim above falsifiable at all; the `cheap` local
// below carries the reason.
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
// WHAT IS ASSERTED IS THE TRIPLE, and no one of the three would do. The LIST
// says the player got this mod's recipe rather than a guess or a hole --
// `cheap`, the preset the dropdown is on, applied because their text was set
// aside. The LINE says the mod noticed. And the TOOLTIP is the only one of the
// three a player ever sees: the settings screen still holds the text they
// typed, and nothing but that trailing line tells them the game is not using
// it.
//
// NOTHING IS SUBSTITUTED FOR A TYPED NAME, and that has not changed. A ladder is
// the AUTHOR saying "any of these will do"; a typed name is a player naming one
// thing, and the library still never guesses at it. What moved is what happens
// to the WHOLE typed list when one entry cannot be used: it is set aside for the
// author's declaration rather than taking the load down with it.
//
// THE FALLBACK IS NOT TOTAL AND NO ROW HERE IS THE EXCEPTION. What the text
// falls back TO is this mod's own declaration -- the preset behind the row the
// player is on -- and a mod set where THAT cannot produce a legal result still
// stops the load. Every ladder resolves in all
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
func TestATextTheGameCannotAnswerFallsBackAndSaysSo(t *testing.T) {
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

	// THE `cheap` LIST, WHICH IS THE PRESET THE DROPDOWN IS ON AND NOT A GUESS.
	// Written out rather than taken from [RecipePlan] for the reason every
	// expectation in this file is written out.
	//
	// `cheap` AND NOT `vanilla`, WHICH IS WHAT MAKES THIS TEST ABLE TO FAIL ON
	// ITS OWN HEADLINE. [Plan] declares the text field's default AS
	// `RecipePlan(RecipeVanilla)`, so on the dropdown's own default value the
	// preset and the field's declared list are the same three ladders: a
	// planner that fell back to the FIELD rather than to the DROPDOWN would
	// emit identical bytes and every row here would pass. Driving `cheap` is
	// the same precaution [TestTheDefaultWordLetsTheDropdownDecide] and
	// [TestAnUnreadableTextLetsTheDropdownDecide] take, and the research twin
	// [TestAPackTextTheGameCannotAnswerFallsBackAndSaysSo] takes by driving
	// `logistics-2`.
	cheap := []ingredientPair{{"iron-plate", 2}, {"transport-belt", 1}}

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
			want: cheap,
		},
		{
			// The display name off this mod's own dropdown label.
			text: "2 iron plates",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("2 iron plates"): no item or fluid is named "iron plates"; ` +
				`did you mean iron-plate`,
			want: cheap,
		},
		{
			// A NAME THE GAME HAS, WHICH IS THE THIRD SHAPE AND NOT A TYPO AT
			// ALL. This mod's recipe declares no `Category`, so it is
			// `crafting`, and the library rejects a fluid there at the list
			// rather than letting the engine refuse the prototype later
			// (FkRecipes go/ingredientlist.go:145-159, and the sentence
			// `categoryTakesItemsOnly` guards). Every real game has `water`, so
			// this is the reason a
			// player reaches after "no item or fluid is named": the field
			// takes an ingredient, water IS one, and what cannot be used
			// names the category rather than the spelling.
			text: "1 water",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("1 water"): water is a fluid, and a recipe in the crafting ` +
				`category takes items only`,
			want: cheap,
		},
		{
			// THE ROW THAT SAYS THE FALLBACK CARRIES ITS LADDERS, which the
			// three above cannot. In a game that has every rung the chosen
			// preset resolves to two flat names and a planner that had lost the
			// ladders would pass those rows. In a game with no
			// `transport-belt` the two part: `cheap`'s second entry is a
			// LADDER, so it falls to `iron-plate`, merges into the first and
			// says so, where a flat list would come out as one name and one
			// missing one and say nothing.
			//
			// WHAT THE TEXT FALLS BACK TO IS THE DROPDOWN, AND THAT IS THE
			// ROUND'S OWN CHANGE. A text this mod cannot use behaves exactly as
			// the word `default` does, so the row the player is standing on
			// decides -- `cheap` on every row here, which is not the list this
			// field declares as its own.
			//
			// THE TEXT IS THE FIRST ROW'S, DELIBERATELY: the only thing this
			// row moves is the game.
			text: "3 tungsten-plate",
			reason: `better-belt-balancer-recipe-ingredients, entry 1 ` +
				`("3 tungsten-plate"): no item or fluid is named tungsten-plate`,
			stocked: noBelt,
			want:    []ingredientPair{{"iron-plate", 3}},
			extra:   mergeLine("2 plus 1 is 3"),
		},
	} {
		w := everythingWorld().
			withStartup(SettingRecipeCost, RecipeCheap).
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
		recipe := protoOf(t, protos, "recipe", PartName)
		checkIngredients(t, "the text "+strconv.Quote(tc.text), recipe, tc.want)
		checkFallbackNote(t, "the text "+strconv.Quote(tc.text), recipe,
			SettingRecipeIngredients, true)
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

// checkFallbackNote is the sentence a PLAYER reads without opening the log,
// which is the half of a fallback that used to be nowhere at all.
//
// WHY IT IS ASSERTED AND NOT LEFT TO THE LOG LINE BESIDE IT. A refused text is
// a state the settings screen cannot show: the field still holds what the
// player typed and the game is not using it, and the log is not where a player
// looks. FkRecipes writes one trailing line into the prototype's own
// `localised_description` for exactly that, so the recipe tooltip in the
// crafting menu and the technology's in the research screen say why they are
// not what was typed.
//
// THE RECIPE CARRIES ONE SENTENCE MORE THAN THE TECHNOLOGY, joined by a space
// on the one line: changing a recipe empties an assembling machine's input
// slots of anything the new list does not use, and repricing a research
// destroys nothing. It is the same constant the library appends to a recipe's
// ERROR line, which is what keeps the tooltip and the log from drifting apart,
// and [fallbackLine] is this file's other reader of it.
//
// THE SHAPE IS TWO PARAMETERS AND NOT THREE because neither this mod's recipe
// nor its technology declares a `Description`: an author's own description
// would sit at slot 1 with the note after it. A prototype nothing fell back on
// carries NO `localised_description` at all, which [TestTheRecipeIsTheOneThatShipped]
// and [TestTheTechnologyIsTheOneThatShipped] say by listing the fields those
// two may have.
func checkFallbackNote(t *testing.T, what string, proto map[string]fkrecipes.Value, setting string, recipe bool) {
	t.Helper()
	note := "The stored value of " + setting + " could not be used, so this " +
		"mod's own choice applies instead. The reason is in the log."
	if recipe {
		note += " Changing a recipe empties an assembling machine's input " +
			"slots of anything the new list does not use."
	}
	want := fkrecipes.Arr(fkrecipes.Str(""), fkrecipes.Str(note))
	if !reflect.DeepEqual(proto["localised_description"], want) {
		t.Errorf("%s: the prototype's localised_description is %s\n want %s",
			what, showValue(proto["localised_description"]), showValue(want))
	}
}

// checkExactlyOneLog is the shorthand for a whole-stream comparison against ONE
// sentence: the line is there word for word, and it is the only thing the plan
// said. It is not the only way that claim is made in this file -- an arm
// expecting more than one line compares the slice with [reflect.DeepEqual]
// instead, and so do some that expect exactly one -- so a test using this
// helper and a test comparing a one-element slice are saying the same thing.
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
// THE RESEARCH CUSTOMIZER: three fields beside the dropdown, each its own
// switch.
//
// `better-belt-balancer-tech-packs`, `better-belt-balancer-tech-count` and
// `better-belt-balancer-tech-seconds` arrived beside `bbb-tech-cost` and the
// dropdown's three values did not move. WHAT DECIDES IS EACH FIELD ITSELF, one
// field at a time: a pack text on the word `default` and a number at 0 leave
// that field to the chosen tier, and anything else overwrites it. So a player
// who touches nothing is charged the tier byte for byte, and a player who drags
// one slider changes one number.
//
// WHAT IS DIFFERENT FROM THE RECIPE'S HALF, and it is why these are not the
// same tests twice. A written recipe replaces the WHOLE list, and its line ends
// `is set aside`; a written research cost replaces the fields it names and the
// tier supplies the rest, so its line ends `supplies what the settings leave at
// default`. The PLACE IN THE TREE is the tier's either way -- every value of
// this dropdown names a source technology and that source is the prerequisite
// however the cost was written -- which is why no arm below asserts a placement
// that moves.
//
// The same rule the recipe's section states holds here: every sentence a PLAYER
// can be shown is compared word for word, because a refusal that stops the load
// is all they are given to fix it with.
// ---------------------------------------------------------------------------

// customUnit is the unit shape the library emits for a written cost: the
// engine's SHORT TUPLE form, `{count, time, ingredients={{name, amount}}}`,
// which is the same shape [unitOf] builds for a fixture technology and the same
// one the tier arms copy verbatim.
//
// THE FIELD ORDER IS THE TIER'S, which is why this still matches a unit the
// library built by OVERWRITING one rather than by constructing one: the library
// replaces a field in the place it already had and appends only a field the
// unit did not carry, so a tier built by [unitOf] and then written over comes
// out in this order.
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

// checkUnit is the whole-unit comparison the arms below share.
func checkUnit(t *testing.T, what string, got map[string]fkrecipes.Value, want fkrecipes.Value) {
	t.Helper()
	if !reflect.DeepEqual(got["unit"], want) {
		t.Errorf("%s: the unit is %s\n want %s", what, showValue(got["unit"]), showValue(want))
	}
}

// TestTheThreeFieldsUntouchedLeaveTheTierDeciding is the state every player who
// never opens the Startup tab is in, and it is fix round 2's central property
// stated as the load they get.
//
// THE TIER BYTE FOR BYTE, AND NOT A COST THAT HAPPENS TO MATCH IT. The
// comparison is against the FIXTURE'S OWN `logistics-2` unit rather than against
// three numbers written out here, which is what makes the claim "whatever this
// game charges for that technology" rather than "200 logistic science": a
// planner that rebuilt the unit out of the three settings' declared defaults
// would produce 0, 0 and this mod's declared pack, and fail on every field.
//
// AND NOT ONE LOG LINE, WHICH IS THE HALF WITH THE DESIGN IN IT. The library
// answers "the custom cost does not apply at all" when all three fields are at
// their declared defaults beside a tier, so there is nothing to narrate; a line
// here would be the library reporting an override on every load of every game.
//
// `logistics-2` AND NOT THE DEFAULT TIER, so a planner that ignored the dropdown
// and copied the first rung would fail: the three fixture units differ from each
// other on purpose.
func TestTheThreeFieldsUntouchedLeaveTheTierDeciding(t *testing.T) {
	base := everythingWorld()
	w := base.withStartup(SettingTechCost, TechLogistics2)
	l2, ok := base.tech(TechLogistics2)
	if !ok || l2.unit == nil {
		t.Fatal("the fixture has no logistics-2 unit to be compared against")
	}

	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, "three untouched fields under logistics-2", got, *l2.unit)
	checkPrereqs(t, "three untouched fields under logistics-2", got, TechLogistics2)
	for _, line := range logs {
		t.Errorf("nothing was overridden and the plan said something: %s", line)
	}
}

// TestTheThreeFieldsOverrideTheTier is the feature, stated as the research a
// player gets.
//
// ALL THREE FIELDS ARE DRIVEN AT ONCE, which is the one place in this file that
// moves more than one variable, and it is deliberate: the three are one cost.
// What makes it readable is that no two of them can be confused -- 50 units, 20
// seconds and a two-pack list, against a tier charging 200 units, 30 seconds
// and one logistic science pack -- so a planner that dropped any one of them
// fails on that field by name.
//
// THE PACK ORDER IS THE TYPED ORDER, for the reason a written recipe's is: the
// list a player writes is the list the research screen shows them back.
//
// AND THE PREREQUISITE IS STILL THE TIER'S SOURCE, which is the whole of what
// separates a repriced research from a moved one. A player who writes a cost
// has said what it costs and nothing about where it sits, so `logistics-2` is
// what it hangs off before and after.
//
// THE LINE ENDS IN THE CLAUSE THAT NAMES THE TIER even though the settings left
// it nothing to supply, and that is the library's shape rather than an
// oversight here: the clause states the RULE a player needs when they move one
// field, and a sentence that appeared and vanished with the number of fields
// driven would be a sentence nobody could learn.
func TestTheThreeFieldsOverrideTheTier(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingTechCost, TechLogistics2).
		withStartup(SettingTechPacks, "1 automation-science-pack, 1 logistic-science-pack").
		withNumberStartup(SettingTechCount, 50).
		withNumberStartup(SettingTechSeconds, 20)
	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, "a written research cost", got, customUnit(50, 20,
		ingredientPair{"automation-science-pack", 1},
		ingredientPair{"logistic-science-pack", 1}))
	checkPrereqs(t, "a written research cost", got, TechLogistics2)
	// A LITERAL, not the constants spliced together: this section compares
	// every line a player is shown word for word, and a renamed constant has to
	// fail this test rather than travel through it.
	checkExactlyOneLog(t, logs,
		"fkrecipes: bbb-balancer takes its research cost from "+
			"better-belt-balancer-tech-packs: count 50, time 20, "+
			"packs 1 automation-science-pack, 1 logistic-science-pack"+
			"; the bbb-tech-cost choice logistics-2 supplies what the settings "+
			"leave at default")
}

// TestOneFieldMovedTakesTheOtherTwoFromTheTier is the property the whole shape
// rests on, and neither test above can make it: that the three fields are three
// switches and not one.
//
// THREE ARMS, ONE FIELD EACH, and every arm asserts the WHOLE unit. The tier is
// `logistics-2` -- 200 units of 30 seconds paid in one logistic science pack --
// and every value driven here is one that tier does not charge, so a planner
// that took all three fields whenever any of them moved would come out with
// this mod's declared 0, 0 and one automation science pack in the two slots the
// player did not touch, and fail by field.
//
// THE PACK ARM IS THE ONE WITH A SECOND CLAIM IN IT. A typed pack list replaces
// the tier's `ingredients` IN PLACE, so the count and the time keep the tier's
// numbers AND the unit keeps the tier's field order; a planner that rebuilt the
// map would pass the value comparison a `DeepEqual` over an ordered map turns
// down.
func TestOneFieldMovedTakesTheOtherTwoFromTheTier(t *testing.T) {
	const line = "fkrecipes: bbb-balancer takes its research cost from " +
		"better-belt-balancer-tech-packs: "
	const tail = "; the bbb-tech-cost choice logistics-2 supplies what the " +
		"settings leave at default"
	base := everythingWorld().withStartup(SettingTechCost, TechLogistics2)

	for _, tc := range []struct {
		name  string
		world fixtureWorld
		want  fkrecipes.Value
		log   string
	}{
		{
			"the count alone", base.withNumberStartup(SettingTechCount, 50),
			customUnit(50, 30, ingredientPair{"logistic-science-pack", 1}),
			line + "count 50, time 30, packs 1 logistic-science-pack" + tail,
		},
		{
			"the seconds alone", base.withNumberStartup(SettingTechSeconds, 20),
			customUnit(200, 20, ingredientPair{"logistic-science-pack", 1}),
			line + "count 200, time 20, packs 1 logistic-science-pack" + tail,
		},
		{
			"the pack text alone",
			base.withStartup(SettingTechPacks, "3 automation-science-pack"),
			customUnit(200, 30, ingredientPair{"automation-science-pack", 3}),
			line + "count 200, time 30, packs 3 automation-science-pack" + tail,
		},
	} {
		protos, logs := extendsOf(t, dataOps(t, tc.world))
		got := protoOf(t, protos, "technology", TechName)
		checkUnit(t, tc.name, got, tc.want)
		checkPrereqs(t, tc.name, got, TechLogistics2)
		checkExactlyOneLog(t, logs, tc.log)
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
// WHAT IT FALLS BACK TO IS THE TIER, AND THAT IS FIX ROUND 2'S CHANGE. A pack
// text this mod cannot use behaves exactly as the word `default` does, so the
// row the player is standing on decides, and with the two numbers untouched as
// well NOTHING of the custom cost applies: the technology is priced by
// `logistics-2` byte for byte, which is precisely what
// [TestTheThreeFieldsUntouchedLeaveTheTierDeciding] asserts for a player who
// typed nothing at all.
//
// ONE LINE AND NOT TWO, which is the other half of that. The library says the
// typed text was set aside and then has no override to report, so the cost line
// that used to follow is absent -- and the stream is compared WHOLE rather than
// searched, which is what makes the absence part of the claim: a plan that
// logged the error and then wrote a cost of its own would fail here.
//
// THE TIER IS `logistics-2` AND NOT THE DEFAULT ONE, so a planner that fell
// back to this mod's own declared pack list rather than to the tier would come
// out with one automation science pack where 200 units of one logistic science
// pack belong.
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
			withStartup(SettingTechCost, TechLogistics2).
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
		tech := protoOf(t, protos, "technology", TechName)
		checkUnit(t, "the pack text "+strconv.Quote(tc.text), tech,
			customUnit(200, 30, ingredientPair{"logistic-science-pack", 1}))
		checkFallbackNote(t, "the pack text "+strconv.Quote(tc.text), tech,
			SettingTechPacks, false)
		if want := []string{packFallbackLine(tc.reason)}; !reflect.DeepEqual(logs, want) {
			t.Errorf("the pack text %q said\n got  %q\n want %q",
				tc.text, logs, want)
		}
	}
}

// TestAFallbackThisModsOwnPackListCannotPayForStillRefuses is the BOUNDARY of
// the contract the two tests above pin, and it is the library's own claim
// rather than this mod's inference: "A VALUE THE PLAYER TYPES NEVER INTRODUCES
// A REFUSAL A PLAYER WHO TYPED NOTHING WOULD NOT ALSO HAVE HIT; AN INPUT THE
// AUTHOR DECLARES STILL REFUSES" (FkRecipes go/customize.go, playerFallback).
//
// TWO THINGS SEPARATE THIS WORLD FROM THE TEST ABOVE, AND THE SECOND IS WHAT
// DECIDES IT. The science packs are `logistic-science-pack` alone, AND the
// dropdown sits at its own default `logistics` rather than at `logistics-2`.
// The tier is what makes the difference: `logistics-2` is priced in a pack this
// game HAS, so driving it here makes the refusal disappear entirely, where
// `logistics` is priced in `automation-science-pack` and loses it.
//
// SO THE WALK IS TWO STEPS AND NOT ONE. `1 water` is the same unusable text the
// first row of the test above drives, so the typed list is set aside; the
// chosen tier's COPIED pack is then dropped because this game has it as
// nothing at all; and only THEN is this mod's declared fallback reached, one
// `automation-science-pack`, which has nothing to land on either. The load
// stops. It stops for a player who typed
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
//
// IT POINTS NOWHERE, AND THAT IS A CORRECTION RATHER THAN A LOSS. It used to
// end by telling the player to correct the setting under Settings then Mod
// settings then Startup; the `Error loading mods` dialog has no route to that
// screen (measured by the library on 2.0.77), so the advice was unusable while
// the fact in front of it was true. The fact stayed and the route went.
//
// THE NAME APPEARS TWICE IN THE LIST OF RUNGS AND THAT IS WHAT THE LIBRARY
// EMITS. It names every rung the walk asked about in the order it asked, and
// this game asks about `automation-science-pack` twice for two different
// reasons: once as the pack the chosen tier's own unit was copied with, and
// once as the rung this mod's declared fallback names. It reads as a stutter to
// anybody who does not know that, and it is transcribed here rather than
// smoothed over, because what this test pins is the sentence a player is
// actually shown.
func TestAFallbackThisModsOwnPackListCannotPayForStillRefuses(t *testing.T) {
	w := everythingWorld().
		withStartup(SettingTechPacks, "1 water").
		withTools("logistic-science-pack")

	_, err := Plan().PlanData(w)
	if err == nil {
		t.Fatal("a game with no automation science pack in it priced this " +
			"mod's declared research anyway: the fallback landed on a pack " +
			"nobody defined, which is a load error with this mod's name on it")
	}
	const want = "fkrecipes: the technology bbb-balancer has no science pack " +
		"the game has; research takes at least one, and none of " +
		"automation-science-pack, automation-science-pack is a science pack " +
		"here. The stored value of better-belt-balancer-tech-packs could not " +
		"be used, so the mod's own declaration applied."
	if err.Error() != want {
		t.Errorf("the refusal is\n got  %q\n want %q", err.Error(), want)
	}
}

// TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused is the OTHER side of
// the tool question, and it is not the same as the refusal above.
//
// A player who typed nothing gets the DECLARED pack list, whose ladder is
// walked through `ToolExists` and whose rungs are DROPPED rather than refused
// when the game has none of them (FkRecipes go/data.go,
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
// same refusal [TestAReachedFallbackWithNoPackInTheGameIsRefused] pins, WITH
// ONE RUNG MORE IN ITS LIST, and the difference is the whole of what these two
// games are: there no technology carries a unit at all, so the only rung ever
// asked about is this mod's declared fallback pack; here the chosen tier DOES
// carry a unit, its copied `automation-science-pack` is asked about and lost
// first, and the declared fallback then names the same name a second time. The
// library reports every rung the walk asked in the order it asked, so the
// sentence carries it twice. The drop itself is a log line the library writes
// on the way, and it is not asserted here because a refused plan hands back the
// refusal and no ops at all.
func TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused(t *testing.T) {
	w := everythingWorld().
		withItems(append(ladderVocabulary(), "automation-science-pack")...).
		withTools("logistic-science-pack")

	_, err := Plan().PlanData(w)
	if err == nil {
		t.Fatal("a game holding this mod's science pack as an ordinary item " +
			"priced its research anyway: the pack ladder is asking ItemExists, " +
			"and the engine takes tool-type items in a unit and nothing else")
	}
	// THE SAME SENTENCE THE FALLBACK ARM GETS, LESS THE STORED-VALUE CLAUSE,
	// because nothing here was typed. The rung list repeats the name for
	// [TestAFallbackThisModsOwnPackListCannotPayForStillRefuses]'s reason: the
	// tier's copied pack and the declared fallback's one rung are the same
	// name asked about twice.
	want := "fkrecipes: the technology bbb-balancer has no science pack the " +
		"game has; research takes at least one, and none of " +
		"automation-science-pack, automation-science-pack is a science pack here"
	if err.Error() != want {
		t.Errorf("the refusal reads\n got  %q\n want %q", err.Error(), want)
	}
}

// TestAnUnreadableResearchNumberLeavesTheTierDeciding is the two numbers'
// version of [TestAnUnreadableTextLetsTheDropdownDecide], and there are two
// shapes of unreadable rather than one because a number has both: a row MISSING
// from a hand-edited mod-settings.dat, and a row PRESENT under the wrong type,
// which is what another mod redefining this mod's setting as text would hand
// the planner. The library reads both through one function (FkRecipes
// go/customize.go, readNumber) and both take the declared default with one
// sentence, which is pinned here word for word.
//
// AND THE DECLARED DEFAULT IS 0, SO THE DEGRADATION IS A NO-OP IN THE PROTOTYPE
// AND A LINE IN THE LOG. That is the whole shape of this test since fix round 2:
// a number that cannot be read falls to 0, 0 means the tier decides, and the
// research comes out priced by `logistics-2` exactly as it does for a player
// whose file is intact. The stream is compared WHOLE, so the absence of a cost
// line is part of the claim: a planner that wrote a 0 into the unit, or that
// reported an override nothing overrode, fails here.
//
// THE OTHER FIELD IS LEFT UNTOUCHED ON EACH ARM, which is what keeps the two
// distinguishable: an implementation that treated one unreadable number as a
// reason to take BOTH from the settings would still produce the tier's numbers
// here and be caught by the log comparison rather than by the unit.
func TestAnUnreadableResearchNumberLeavesTheTierDeciding(t *testing.T) {
	base := everythingWorld().withStartup(SettingTechCost, TechLogistics2)
	l2, ok := base.tech(TechLogistics2)
	if !ok || l2.unit == nil {
		t.Fatal("the fixture has no logistics-2 unit to be compared against")
	}
	for _, tc := range []struct {
		name  string
		world fixtureWorld
		line  string
	}{
		{
			"the count row is missing",
			base.withoutNumberStartup(SettingTechCount),
			"fkrecipes: the setting better-belt-balancer-tech-count was not " +
				"readable, so its default applies",
		},
		{
			"the seconds row holds text",
			base.withRawStartup(SettingTechSeconds, fkrecipes.Str("15")),
			"fkrecipes: the setting better-belt-balancer-tech-seconds was not " +
				"readable, so its default applies",
		},
	} {
		protos, logs := extendsOf(t, dataOps(t, tc.world))
		got := protoOf(t, protos, "technology", TechName)
		checkUnit(t, tc.name, got, *l2.unit)
		checkPrereqs(t, tc.name, got, TechLogistics2)
		if !reflect.DeepEqual(logs, []string{tc.line}) {
			t.Errorf("%s: the plan's log stream is %q\n want the degradation "+
				"and nothing else, %q", tc.name, logs, []string{tc.line})
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
