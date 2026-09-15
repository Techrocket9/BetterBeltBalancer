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

// ingredientlessNote is the sentence the LIBRARY writes into a RECIPE's own
// tooltip when every entry that recipe declared was put to the game and every
// ladder ran out, and ingredientlessLine is the ERROR line that goes with it.
// They are [packlessNote] and [packlessLine] one prototype kind over, and
// FkRecipes 933d4d3 composed both.
//
// LITERALS RATHER THAN [PartName], for the reason [packlessNote] is one: a
// renamed constant must FAIL a sentence a player reads rather than be carried
// through it.
//
// THE NOTE ALWAYS CARRIES THE DESTRUCTION TAIL AND THE LINE NEVER DOES, and
// that is the library's rule rather than this pair drifting. FkRecipes
// docs/migration.md:417 states it with no condition on it: the tooltip carries
// the sentence "followed by the sentence about emptied input slots", and the
// ERROR line it spells beside it ends on `costs nothing to craft`. What moved
// here IS the ingredient list and it moved as far as a list can, which is why
// the tail is unconditional on this note where [fallbackLine] takes it only on
// the recipe channel.
//
// AND NEITHER NAMES A SETTING, because nobody typed anything. This is the mod
// set having none of the names the recipe offers, so a sentence sending the
// player to a field would be advice about a field that is not the problem.
const (
	ingredientlessNote = "This game has none of the ingredients this recipe " +
		"names, so it costs nothing to craft. The reason is in the log. " +
		"Changing a recipe empties an assembling machine's input slots of " +
		"anything the new list does not use."
	ingredientlessLine = "fkrecipes: ERROR: bbb-balancer-part: this game has " +
		"none of the ingredients this recipe names, so it is emitted with no " +
		"ingredients and costs nothing to craft"
)

// dropLine is the library's exhausted-ladder sentence, whose only moving part
// is the rung list inside it.
//
// A LITERAL WITH ONE PARAMETER, for [mergeLine]'s reason. The rungs are what
// differs between the presets below, and they are also what a ladder walked in
// the wrong order, or cut short, would get wrong.
func dropLine(rungs string) string {
	return "fkrecipes: bbb-balancer-part: none of " + rungs +
		" is present, so the ingredient is dropped"
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
// closed. The stream comparison below is what says it a second way: not one of
// the lines in it is about the technology.
//
// THE LINE AND THE NOTE WENT UNPINNED BECAUSE THIS TEST THREW THE STREAM AWAY.
// It compared the ingredient count and discarded the logs, so a library that
// emptied this recipe and said NOTHING anywhere passed it. FkRecipes 933d4d3
// writes both halves here, and the technology's identical degradation got a
// unit, a prerequisite, a composed note and an ordered whole-stream comparison
// in [TestAReachedFallbackWithNoPackInTheGameIsEmittedFreeAndDisclosed]; this
// is the recipe's half of that shape and it is asserted the same way.
//
// SIX PRESETS, ONE LINE AND ONE NOTE EACH, MEASURED. [RecipeOptions] is six and
// each row is its own load, so the ERROR line closes that row's stream once and
// never twice; the note is 213 bytes and comes out 178 + 35, which is the
// library's 180-byte budget cut after the last space inside it.
//
// THE STREAMS ARE WHERE FIVE ROWS OF SIX DIFFER, and what differs is the
// dropdown's own fallback rather than anything about this degradation: a chosen
// plan that resolves to nothing takes the DEFAULT plan's, so five presets drop
// their own rungs, say so, and then drop vanilla's three as well. Comparing the
// whole stream in order is what says the recipe went empty by exhausting its
// ladders rather than by some shorter route that happens to end in the same
// line.
func TestAGameWithNoIngredientsIsAnEmptyRecipeRatherThanAnInventedOne(t *testing.T) {
	// The three lines VANILLA's own ladders write in this world. Every row
	// writes them, because vanilla is the plan the other five fall through to,
	// and on the vanilla row they are the whole of the stream before the ERROR.
	vanillaDrops := []string{
		dropLine("iron-plate"),
		dropLine("iron-gear-wheel, iron-plate"),
		dropLine("transport-belt, iron-plate"),
	}

	// TRANSCRIBED FROM THE PLANS BY HAND, in the order [RecipeOptions] returns
	// them, for the reason every expectation in this file is: a table walked
	// off [RecipePlan] would agree with a defect in [RecipePlan]. `own` is what
	// the CHOSEN preset's ladders say before that preset gives up, and it is
	// empty on the vanilla row, which gives up into itself.
	for _, tc := range []struct {
		option string
		own    []string
	}{
		{RecipeVanilla, nil},
		{RecipeCheap, []string{
			dropLine("iron-plate"),
			dropLine("transport-belt, iron-plate")}},
		{RecipeBeltFast, []string{
			dropLine("iron-plate"),
			dropLine("iron-gear-wheel, iron-plate"),
			dropLine("fast-transport-belt, transport-belt, iron-plate")}},
		{RecipeBeltExpress, []string{
			dropLine("steel-plate, iron-plate"),
			dropLine("iron-gear-wheel, iron-plate"),
			dropLine("express-transport-belt, fast-transport-belt, " +
				"transport-belt, iron-plate")}},
		{RecipeSplitter, []string{
			dropLine("splitter, transport-belt, iron-plate"),
			dropLine("iron-plate")}},
		{RecipeSplitterExpress, []string{
			dropLine("express-splitter, fast-splitter, splitter, " +
				"transport-belt, iron-plate"),
			dropLine("steel-plate, iron-plate")}},
	} {
		w := everythingWorld().
			withStartup(SettingRecipeCost, tc.option).
			withItems()
		protos, logs := extendsOf(t, dataOps(t, w))
		got := protoOf(t, protos, "recipe", PartName)

		if list := ingredientsOf(t, got); len(list) != 0 {
			t.Errorf("%s invented %v in a game with no ingredients at all",
				tc.option, list)
		}

		// THE CUT IS DERIVED FROM THE TAIL RATHER THAN TRANSCRIBED AS A SECOND
		// LITERAL, which is [checkFallbackNote]'s own device: the head is the
		// whole sentence less exactly this, so the two chunks cannot drift
		// from the whole by a typing error.
		const tail = "anything the new list does not use."
		checkComposedNote(t, tc.option, got, "recipe", PartName,
			ingredientlessNote, []string{
				ingredientlessNote[:len(ingredientlessNote)-len(tail)], tail})

		want := append([]string{}, tc.own...)
		if len(tc.own) > 0 {
			want = append(want, "fkrecipes: bbb-balancer-part: the "+
				tc.option+" ingredients name nothing this game has, so the "+
				"vanilla ingredients apply")
		}
		want = append(want, vanillaDrops...)
		want = append(want, ingredientlessLine)
		if !reflect.DeepEqual(logs, want) {
			t.Errorf("%s in a game with no ingredients at all said\n got  %q"+
				"\n want %q", tc.option, logs, want)
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
// THAT DAY IS 2026-09-07. FkRecipes c7a806e made the fallback's packs a
// question asked where the fallback applies rather than always, and the design
// record calls it the answer to this mod's ask by name, in a row of FkRecipes'
// agents/customizer-design.md whose decision ends: the Fallback is resolved
// only when used. The fallback's NUMBERS are still checked eagerly, in the plan
// walk with no World in hand (go/customize.go, validateCostChoices), which is the right split: a
// count of zero is this mod's mistake and is knowable without asking the game
// anything.
//
// AND "WHERE THE FALLBACK APPLIES" IS TWO PLACES AND NOT ONE. This header used
// to quote the library's own comment for that sentence -- "its packs are probed
// when the fallback is what applies, and never when a source answered" -- and
// FkRecipes 61ac80c took those words out of its source as false in both halves:
// the packs are put to the game when no source in the chosen ladder carries a
// unit, AND when a source carried one whose every pack the `tool` probe then
// dropped. docs/migration.md now says in as many words what a fixture proving
// this property needs, "a source that carries a copyable unit AND keeps at
// least one pack, which a game with every `tool` taken out of it does not
// have". The world below is exactly that, which the paragraph after next
// narrowed it to be before either repository had the sentence for why.
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
// property, and
// [TestAPackTheGameHasOnlyAsAnItemIsDroppedAndTheResearchIsEmittedFree] is
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

// packlessNote is the sentence the LIBRARY writes into a technology's own
// tooltip when every science pack it named was put to the game and the game had
// none of them, and packlessLine is the ERROR line that goes with it.
//
// LITERALS RATHER THAN [TechName] AND [FallbackUnit], which is this file's rule
// for a sentence a player reads: a renamed constant must FAIL the sentence
// rather than be carried through it.
//
// THE RUNG LIST IN THE LINE IS ONE NAME IN ALL THREE WORLDS THAT REACH IT, so
// it is written into the constant rather than taken as a parameter. This mod
// declares its fallback pack as `automation-science-pack` with no ladder under
// it, and the tier whose copied pack is lost first is priced in that same name,
// which the library deduplicates: FkRecipes `f2df7ab` names each rung once
// however many times the walk asked about it, and the two tests that used to
// pin the doubled form are below.
//
// THE NOTE NAMES NO RUNG AND THE LINE NAMES THEM ALL, and that split is the
// library's own. The names are what an author reading a log can act on; a
// player hovering the research screen cannot act on a list of prototype names
// their mod set has not got.
//
// AND NEITHER NAMES A SETTING, because nobody typed anything. This is an
// ENVIRONMENTAL degradation -- the mod set is missing what the plan names --
// so a sentence telling the player to go and fix a field would be advice about
// a field that is not the problem.
const (
	packlessNote = "This game has none of the science packs this research " +
		"names, so it takes no science pack at all. The reason is in the log."
	packlessLine = "fkrecipes: ERROR: bbb-balancer: none of " +
		"automation-science-pack is a science pack this game has, so the " +
		"research is emitted with no science pack and completes for free"
)

// TestAReachedFallbackWithNoPackInTheGameIsEmittedFreeAndDisclosed is the other
// half, and it is what the lazy probe MOVED rather than removed.
//
// The pack question did not go away: it moved to the one game that has to
// answer it, which is the game where the fallback is what prices the research.
// There the pack is walked as a ladder and dropped if nothing on it is a tool
// (FkRecipes c7a806e gave [fkrecipes.Pack] a `Fallbacks` list for exactly
// that). THIS MOD DECLARES ITS FALLBACK PACK WITH NO LADDER, one rung named
// `automation-science-pack`, so a game without red science loses it and has
// nothing else to try.
//
// THE NAME SAID "IsRefused" UNTIL FkRecipes 97a4493, AND WHAT IT PINNED WAS
// REAL: a unit that lost every pack refused the load by name, and this header
// argued the refusal was the right answer because the alternative is a
// technology a player finishes by opening the research screen. The library
// weighed the two costs against each other and took the other one. A refusal
// here is a mod set that does not load, from an `Error loading mods` dialog
// with no route to the Mod Settings screen; the escape that dialog does have
// was measured on a client and is Reset mod settings TOGETHER WITH Disable
// listed mods, which costs every startup preference in the file and leaves the
// mod off -- six steps, then re-enable and restart
// (agents/migration-assessment.md:93-108, which walked the five buttons and
// the checkbox). An empty research unit is a FREE research rather than a stuck
// one, measured in play on 2.0.77. So the technology is EMITTED and the load
// completes.
//
// THAT IS NOT A GOOD OUTCOME AND THIS TEST DOES NOT SAY IT IS. A research this
// mod's player finishes by opening a screen is a balance change nobody chose,
// and it is one this mod would rather not ship. What the library bought is that
// it is DISCLOSED instead of silent: one ERROR line naming every rung the walk
// asked the game about, and one sentence in the technology's own tooltip, above
// the description this mod wrote for it rather than over the top of it. What
// would stop it happening here at all is a second rung under the declared pack,
// which plan.go's own block declines and which is the next commit's subject.
//
// THE UNIT KEEPS [FallbackUnit]'s NUMBERS AND LOSES ONLY THE PACKS, asserted
// because it is separable: 20 and 15 are what this technology has cost in every
// save this mod has ever been in, and a unit that lost those too would be a
// second balance change riding on the first.
//
// AND IT CARRIES NO PREREQUISITE, which is the fallback arm's own signature and
// is what says this is the fallback rather than a tier that went wrong: no
// technology in this game carries a unit, so there is nothing to hang the
// research off and nothing is invented to hang it off.
//
// THE WHOLE STREAM IN ORDER, three lines and not one. The fallback announces
// itself, the pack ladder announces the drop, and the ERROR announces what is
// left; a plan that emitted the free research with only the first two lines
// would pass a search for the third and fail this comparison.
func TestAReachedFallbackWithNoPackInTheGameIsEmittedFreeAndDisclosed(t *testing.T) {
	// No technology at all, so no rung of any ladder carries a unit and the
	// fallback IS the price; and no tool, so the one pack it names resolves to
	// nothing.
	const what = "a reached fallback with no pack in the game"
	protos, logs := extendsOf(t, dataOps(t, everythingWorld().withTechs().withTools()))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, what, got, customUnit(20, 15))
	if v, ok := got["prerequisites"]; ok {
		t.Errorf("%s: the technology carries prerequisites %s, and no "+
			"technology in this game carries a unit to be one", what,
			showValue(v))
	}
	checkComposedNote(t, what, got, "technology", TechName,
		packlessNote, []string{packlessNote})

	want := []string{
		"fkrecipes: bbb-balancer: no source for the logistics cost carries a " +
			"unit, so the fallback cost applies and the technology has no " +
			"prerequisite",
		"fkrecipes: bbb-balancer: none of automation-science-pack is present, " +
			"so the science pack is dropped",
		packlessLine,
	}
	if !reflect.DeepEqual(logs, want) {
		t.Errorf("%s: the plan's log stream is\n got  %q\n want %q",
			what, logs, want)
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

// TestARefusalRaisedAfterAFallbackSaysWhatTheStoredValueDid is the one sentence
// the library adds on behalf of a player whose typed value was set aside on the
// way to a refusal, and nothing in this repository pinned it until now.
//
// WHY IT IS OWED AT ALL. The log lines the library accumulates never reach the
// host on a refused load: they are collected through the whole resolution and
// turned into ops only after every post-resolution check has passed, so a plan
// that refuses hands the engine ONE message and nothing else. Without this
// sentence a player would read a refusal about this mod's own declaration with
// nothing anywhere to say that the field they edited had been set aside.
//
// IT IS A FACT AND NOT A ROUTE, which is what separates it from the three lines
// the library writes on a load that SUCCEEDS. Those end by sending the player
// to Settings > Mod settings > Startup; this one cannot, because the client
// cannot reach that screen from an `Error loading mods` dialog -- re-measured
// by FkRecipes on 2.0.77: the dialog offers Disable listed mods, Disable all
// mods, Manage mods, Restart, Exit and a Reset mod settings checkbox, Manage
// mods has no Mod settings button and its Back returns to the same dialog. So
// the sentence states what became of the value and sends nobody anywhere.
//
// THE REFUSAL IT RIDES ON IS THE UNOFFERED DROPDOWN VALUE, and it is the only
// one this mod's own declarations can still reach after resolution. The
// packless refusal that used to carry this sentence was deleted in FkRecipes
// 97a4493 and is a free research now; the three `already exists in data.raw`
// probes are raised BEFORE resolution and so carry nothing; and this recipe's
// crafting time is the literal 1 rather than a settings reference, so the
// crafting-time floor is unreachable from here. NOTHING IN THE FIXTURE IS BENT
// TO GET HERE: this is [TestAnUnofferedStoredValueIsRefusedByName]'s world with
// one more hand-edited row in it.
//
// AND BOTH ROWS ARE HAND-EDITED FILE STATES RATHER THAN THINGS A PLAYER DOES.
// Factorio RESETS a stored value the running release's `allowed_values` does
// not list before the data stage runs, with no line in the log, so the dropdown
// half is reachable only through an edited `mod-settings.dat`; the text half is
// something a player types every day. That the two can be true at once is what
// this refusal is for.
//
// THE SENTENCE NAMES THE TEXT FIELD AND NOT THE DROPDOWN, which is the claim
// with the most in it. Only a value that FELL BACK is named; the dropdown was
// REFUSED rather than fallen back on, and the library keeps the two in
// different places. A library that named the refused setting here would be
// telling the player their dropdown was set aside, which is the opposite of
// what happened to it.
func TestARefusalRaisedAfterAFallbackSaysWhatTheStoredValueDid(t *testing.T) {
	const unoffered = "not-an-option"
	for _, tc := range []struct{ dropdown, field, typed string }{
		// The ingredient pair, whose typed name no mod in this game defines.
		{SettingRecipeCost, SettingRecipeIngredients, "3 tungsten-plate"},
		// The research pair, whose typed name is a fluid the fixture stocks.
		{SettingTechCost, SettingTechPacks, "1 water"},
	} {
		w := everythingWorld().
			withStartup(tc.dropdown, unoffered).
			withStartup(tc.field, tc.typed)
		_, err := Plan().PlanData(w)
		if err == nil {
			t.Errorf("%s holding %q planned cleanly beside a text this game "+
				"cannot answer", tc.dropdown, unoffered)
			continue
		}
		// THE WHOLE STRING, which is what makes the added sentence part of the
		// claim rather than something a substring search would find either way.
		want := "fkrecipes: " + tc.dropdown + ` holds "` + unoffered +
			`", which is not one of its values. The stored value of ` +
			tc.field + " could not be used and was set aside, so what " +
			"applied is what that field gives when it is left alone."
		if err.Error() != want {
			t.Errorf("%s: the refusal reads\n got  %q\n want %q",
				tc.dropdown, err.Error(), want)
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
//
// THE MIDDLE CLAUSE MOVED IN FkRecipes ca4c344 AND THE ADVICE DID NOT. It used
// to say the mod loaded with its own default instead, which beside a dropdown
// is false on five presets of six: what applies is the row the player is
// standing on, not the list this mod declares for the field. What is true on
// all six, and beside a field with no dropdown, is that the field behaved as
// though it had been left alone. The route at the end is unchanged, and it is
// usable here for the reason it is unusable in a refusal: this is a LOG line on
// a load that SUCCEEDED, so the player is in the game and the Mod Settings
// screen is two clicks away.
func packFallbackLine(reason string) string {
	return "fkrecipes: ERROR: " + reason +
		". The mod loaded as though that text had been left alone; fix the " +
		"text under Settings > Mod settings > Startup, then restart."
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
// WHAT THE TEXT FALLS BACK TO CAN STILL BE DEGRADED, WHICH IS NOT THE SAME AS
// A REFUSAL AND USED TO BE. What the text falls back TO is this mod's own
// declaration -- the preset behind the row the player is on -- and a mod set
// where THAT cannot produce a legal result degrades it further. Every ladder
// resolves in all four worlds here, the fourth by merging; the recipe field's
// own worst case is an empty ingredient list, which
// [TestAGameWithNoIngredientsIsAnEmptyRecipeRatherThanAnInventedOne] pins as the
// trade this mod took, and the pack field's is a research priced in no science
// pack, which
// [TestAFallbackThisModsOwnPackListCannotPayForIsEmittedFreeBesideTheSetAsideText]
// pins as the trade FkRecipes 97a4493 took on this mod's behalf. NEITHER STOPS
// THE LOAD ANY MORE. What still does is an author's declaration the library
// cannot use at all, and after a fallback that refusal carries a sentence of
// its own: [TestARefusalRaisedAfterAFallbackSaysWhatTheStoredValueDid].
//
// THE SECOND ARM IS THE ONE THIS MOD CAUSED, AND NO LABEL OF THIS MOD'S SPELLS
// IT ANY MORE. The fold exists in FkRecipes because these dropdown labels were
// the example in its design review, where they read "Default: 4 iron plates, 2
// gears, ..."; fix round 2's third commit rewrote all nine as NAMES, and
// mod-data/locale/en/better-belt-balancer.cfg records that reversal and the
// measurement behind it. THE ROW IS NOT WHAT RESTED ON THE LABELS. Display
// prose still reaches this field, by shorter routes than a settings label: the
// option table in README.md is written that way, and the game's own item name
// for `iron-plate` is "Iron plate", a capital and a plural away from what this
// row types.
// So `2 iron plates` is still the likeliest single thing a player puts in this
// field, and the answer it gets is still the internal name rather than advice
// about commas.
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
// THE AUTHOR'S OWN DESCRIPTION IS COMPOSED ABOVE THE NOTE SINCE FkRecipes
// 121aafc, AND THAT IS FINDING 21 OF agents/migration-assessment-3.md CLOSED.
// The note used to be the WHOLE `localised_description`, so a prototype whose
// description lives in a locale file -- which is both of this mod's, the
// technology by `technology-description.bbb-balancer` and the recipe by
// `recipe-description.bbb-balancer-part` -- had that description REPLACED by
// the disclosure the moment anything fell back: the player was told why the
// research was not what they typed and lost what the research is. The library
// now writes the engine's own key for the prototype it is emitting into slot 1,
// in the ALTERNATIVES form with a trailing newline INSIDE the alternative, and
// the note follows it. A consumer that declared no such key renders the note
// alone, because a concatenation group holding an undefined key is itself a
// failed alternative and the newline dies with it -- measured by the library on
// 2.0.77 build 84539, which is what makes this shape and not the flatter one
// that leaves a dangling newline on every consumer who wrote no entry.
//
// A PROTOTYPE NOTHING FELL BACK ON CARRIES NO `localised_description` AT ALL,
// which has not changed and is what [TestTheRecipeIsTheOneThatShipped] and
// [TestTheTechnologyIsTheOneThatShipped] say by listing the fields those two
// may have. The wrapper arrives WITH the note and never on its own.
//
// THE NOTE IS CHUNKED ACROSS PARAMETERS SINCE FkRecipes `137f4aa`, AND THAT IS
// WHY THE RECIPE'S NOTE IS TWO ELEMENTS AND THE TECHNOLOGY'S ONE. The engine
// refuses a localised-string element over 200 BYTES and stops the whole load
// (measured on 2.0.77 build 84539 by both repositories), which is what this
// mod's own release block was about: the recipe sentence is 268 bytes with this
// mod's 39-byte setting name in it, so before the library chunked it a typo in
// the ingredients field could not be disclosed at all. The library fills a
// chunk to 180 bytes and cuts after the last space inside it, so the 268 comes
// out 179 + 89 and the technology's 159 stays in one piece. BOTH NUMBERS MOVED
// IN ca4c344 and they moved because the SENTENCE did: it was 247 bytes cut
// 176 + 71, and the clause that replaced "so this mod's own choice applies
// instead" is longer than it was.
//
// FOUR SEPARATELY BREAKABLE THINGS, IN THIS ORDER, because the first two are
// the properties a player can be hurt by, the third is what a player reads and
// the fourth is where the library chose to cut:
//
//   - THE AUTHOR'S OWN DESCRIPTION IS STILL THERE, at slot 1, naming the key
//     for the prototype kind and the emitted name this note is riding on. This
//     is finding 21's assertion: a library that went back to replacing the
//     description fails here rather than in a research screen.
//   - EVERY STRING THE COMPOSITION HOLDS, AT ANY DEPTH, IS AT OR UNDER THE
//     ENGINE'S CEILING. This is the one that decides whether the game loads,
//     and it is asserted here as well as in `test/check-datastage.py`'s
//     `note-recipe` arm, where a real engine is what answers. The walk
//     DESCENDS into the wrapper at slot 1 rather than stepping over it, and
//     the reason is the locale KEY inside it: a key is one element by
//     definition and cannot be chunked, the engine polices it at the same 200
//     bytes, and FkRecipes DROPS the wrapper rather than compose one that
//     would not fit -- at an emitted recipe name over 181 bytes or a
//     technology name over 177 (FkRecipes docs/usage.md:333). This walk used
//     to step over slot 1 by index, so the claim that every string element was
//     measured was false for exactly that key. `check_note_elements` in the
//     Python twin descends for this reason and says so, and the two halves
//     agree now.
//   - THE ELEMENTS CONCATENATED ARE THE SENTENCE, BYTE FOR BYTE. The engine
//     joins a localised string's parameters with nothing between them, so this
//     is what says the chunking is invisible to whoever reads the tooltip.
//   - AND THE CUT FALLS WHERE THIS LIBRARY HEAD PUTS IT. Pinned so a library
//     that re-chunks fails here by name rather than moving a tooltip silently;
//     it is the weakest of the four and the first to give way if the budget
//     ever moves for a reason.
func checkFallbackNote(t *testing.T, what string, proto map[string]fkrecipes.Value, setting string, recipe bool) {
	t.Helper()
	// The tail the library's own chunker breaks before, spelled once so the
	// expected head below is the sentence minus exactly this.
	const recipeTail = "recipe empties an assembling machine's input slots " +
		"of anything the new list does not use."
	note := "The stored value of " + setting + " could not be used, so the " +
		"game loaded as though that setting had been left alone. The reason " +
		"is in the log."
	kind, name := "technology", TechName
	// 179 bytes ending in the space after `Changing a`, then the 89-byte tail.
	chunks := []string{note}
	if recipe {
		note += " Changing a " + recipeTail
		kind, name = "recipe", PartName
		chunks = []string{note[:len(note)-len(recipeTail)], recipeTail}
	}
	checkComposedNote(t, what, proto, kind, name, note, chunks)
}

// checkElementCeiling measures EVERY STRING one localised string holds, at any
// depth, against the engine's ceiling on a single element.
//
// IT DESCENDS RATHER THAN STEPPING OVER A TABLE, and the reason is the one slot
// that cannot be chunked. A `descriptionRef` wrapper's deepest string is the
// author's own locale key; the engine polices a key slot at the same 200 bytes
// it polices a literal at, and FkRecipes answers that by DROPPING the wrapper
// rather than composing a key over the ceiling (FkRecipes docs/usage.md:333
// gives the two name lengths at which it does). A walk that skipped the wrapper
// could not see either side of that decision. `check_note_elements` in
// `test/check-datastage.py` descends for this reason over a real dump; this is
// the same claim made on the host, and the two are meant to read alike.
//
// ANYTHING THAT IS NEITHER A STRING NOR AN ARRAY IS A FAILURE, because neither
// the engine nor this library has a third thing to put in a localised string.
func checkElementCeiling(t *testing.T, what string, v fkrecipes.Value, path string) {
	t.Helper()
	// The engine's, not the library's: the library fills to 180 and this is
	// the number that refuses a load. Compared against and never computed with.
	const elementCeiling = 200

	switch v.Kind {
	case fkrecipes.KindStr:
		if len(v.Str) > elementCeiling {
			t.Errorf("%s: localised_description%s is %d bytes, over the "+
				"engine's %d-byte ceiling on one element: a load carrying "+
				"this prototype STOPS, which is the defect the library's "+
				"chunker closed\n  %q",
				what, path, len(v.Str), elementCeiling, v.Str)
		}
	case fkrecipes.KindArr:
		for i, el := range v.Arr {
			checkElementCeiling(t, what, el, path+"["+strconv.Itoa(i)+"]")
		}
	default:
		t.Errorf("%s: localised_description%s is %s, and a localised string "+
			"holds strings and tables of them and nothing else",
			what, path, showValue(v))
	}
}

// checkComposedNote is the whole of what the header above describes, asked of
// ANY note this library composes onto a prototype rather than of a fallback's
// alone.
//
// IT IS A FUNCTION OF ITS OWN BECAUSE THERE ARE TWO KINDS OF NOTE NOW AND ONE
// SHAPE. A stored value the player typed and could not be used gets
// [checkFallbackNote]'s sentence; a science pack the GAME does not have gets
// the packless one, which nobody typed and which names no field to go and fix.
// Both ride in the same wrapper over the same author key with the same chunker
// under them, so the shape is asserted once and the sentence is the caller's.
func checkComposedNote(t *testing.T, what string, proto map[string]fkrecipes.Value, kind, name, note string, chunks []string) {
	t.Helper()
	got := proto["localised_description"]
	if got.Kind != fkrecipes.KindArr || len(got.Arr) < 2 ||
		!reflect.DeepEqual(got.Arr[0], fkrecipes.Str("")) {
		t.Errorf("%s: the prototype's localised_description is %s\n want the "+
			"inline form, an array opening with an empty string parameter",
			what, showValue(got))
		return
	}

	// SLOT 1 IS THE AUTHOR'S OWN DESCRIPTION AND NOT THE NOTE. Written out here
	// rather than built by a helper shared with plan_test.go's [localeRef]:
	// this is a DIFFERENT shape, the key nested inside a concatenation group
	// with the separator newline in it, and the two are only alike enough to be
	// confused.
	wantRef := fkrecipes.Arr(
		fkrecipes.Str("?"),
		fkrecipes.Arr(
			fkrecipes.Str(""),
			fkrecipes.Arr(fkrecipes.Str(kind+"-description."+name)),
			fkrecipes.Str("\n")),
		fkrecipes.Str(""))
	if !reflect.DeepEqual(got.Arr[1], wantRef) {
		t.Errorf("%s: localised_description element 1 is %s\n want %s\n  the "+
			"note joins this mod's own locale description rather than "+
			"replacing it", what, showValue(got.Arr[1]), showValue(wantRef))
	}

	checkElementCeiling(t, what, got, "")

	joined := ""
	for i, el := range got.Arr[2:] {
		if el.Kind != fkrecipes.KindStr {
			t.Errorf("%s: localised_description element %d is %s, and every "+
				"parameter of a composed note below the author's own "+
				"description is a string", what, i+2, showValue(el))
			return
		}
		joined += el.Str
	}
	if joined != note {
		t.Errorf("%s: the note reads\n got  %q\n want %q\n  the engine joins "+
			"the parameters with nothing between them, so the concatenation "+
			"is what a player sees", what, joined, note)
	}
	params := []fkrecipes.Value{fkrecipes.Str(""), wantRef}
	for _, c := range chunks {
		params = append(params, fkrecipes.Str(c))
	}
	want := fkrecipes.Arr(params...)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: the prototype's localised_description is %s\n want %s",
			what, showValue(got), showValue(want))
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
// take that instruction from a text field. IT WILL EMIT ONE WHERE THE GAME
// LEAVES IT NO CHOICE, which is the distinction FkRecipes 97a4493 drew and not
// a contradiction: a player TYPING `none` is refused because they are asking
// for a free research, and a mod set that has none of the packs this mod names
// gets one anyway because the alternative there is a load that stops.
// [TestAReachedFallbackWithNoPackInTheGameIsEmittedFreeAndDisclosed] is that
// side.
//
// THE FOURTH IS THE FLUID, and it is the recipe field's fluid case with a
// different second clause: there the category rule rejects it, here research
// takes science packs and nothing else. Every real game has `water`, the two
// fields take one language, and a player who has used the first field is the
// player likeliest to reach for the same name in the second; the fixture stocks
// `water` for exactly this row and the recipe's.
//
// ALL FOUR ROWS LAND ON A TIER THIS GAME CAN PAY FOR, which is what keeps them
// about the TEXT.
// [TestAFallbackThisModsOwnPackListCannotPayForIsEmittedFreeBesideTheSetAsideText]
// is the game that cannot: the same `1 water` where the tier's pack and this
// mod's declared one are both absent, and the research comes out free with the
// player's line last in a stream of five.
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

// TestAFallbackThisModsOwnPackListCannotPayForIsEmittedFreeBesideTheSetAsideText
// is where a PLAYER'S fallback and the ENVIRONMENT'S degradation meet on one
// technology, and it is what the library's boundary became.
//
// IT USED TO BE A REFUSAL AND IT USED TO BE THE BOUNDARY ITSELF. The claim this
// test carried was the library's own: "A VALUE THE PLAYER TYPES NEVER
// INTRODUCES A REFUSAL A PLAYER WHO TYPED NOTHING WOULD NOT ALSO HAVE HIT; AN
// INPUT THE AUTHOR DECLARES STILL REFUSES", and this world was the second half
// of it -- the author's own declared pack had nothing to land on either, so the
// load stopped. FkRecipes 97a4493 deleted that refusal: the packless unit is
// EMITTED now, so what stands here is the first half of the claim alone, and
// the whole boundary moved to
// [TestARefusalRaisedAfterAFallbackSaysWhatTheStoredValueDid].
//
// THE WALK IS THREE STEPS AND EVERY ONE OF THEM SAYS SO. `1 water` is the same
// unusable text the first row of the test above drives, so the typed list is
// set aside; the chosen tier's COPIED pack is then dropped because this game
// has `automation-science-pack` as nothing at all; this mod's declared fallback
// is reached, one `automation-science-pack`, which has nothing to land on
// either; and what comes out is a research that costs 20 of nothing over 15
// seconds. The tier is the dropdown's own default `logistics` rather than
// `logistics-2`, which is what makes the second step happen at all:
// `logistics-2` is priced in a pack this game HAS, so driving it here would
// leave nothing for the ladder to lose.
//
// FIVE LINES IN ORDER AND THE ORDER IS THE CLAIM. The player's own ERROR line
// comes LAST rather than first, which is the opposite of the recipe field's
// (see [TestATextTheGameCannotAnswerFallsBackAndSaysSo], where it comes first),
// and it is not a detail: the pack text is read where the research cost is
// priced, so the tier walk that set the packs aside has already run and written
// its four lines by the time the typed value is reported. A stream comparison
// is what pins that; a search for each line would pass on any ordering.
//
// THE RUNG NAME APPEARS ONCE, AND IT USED TO APPEAR TWICE. This game asks about
// `automation-science-pack` for two different reasons, once as the pack the
// chosen tier's unit was copied with and once as the rung this mod's declared
// fallback names. Until FkRecipes `f2df7ab` the sentence carried both askings
// and read as a stutter, and this test was one of the two here that pinned it
// that way on purpose so a library shortening it would fail by name. It did,
// and this is the name it failed by. The property survived the refusal that
// used to carry it -- the same deduplicated list is in [packlessLine] now -- so
// it is still pinned here.
//
// ONLY ONE OF THE TWO FACTS REACHES THE TOOLTIP, AND IT IS THE ENVIRONMENT'S.
// The budget is one line per prototype and FkRecipes documents that on both
// sides separately: "One line per prototype, naming the first setting the walk
// set aside" of a player's fallback (docs/usage.md:301) and "A prototype
// carries at most one of these lines: the first the walk reaches" of an
// environmental degradation (docs/usage.md:327). NEITHER SENTENCE SAYS WHICH
// KIND WINS WHEN ONE OF EACH IS TRUE, which is exactly this technology, so that
// the ENVIRONMENT'S note takes the slot is MEASURED HERE and cited nowhere: the
// packless note is written while the cost is being resolved, and the typed pack
// list is reported after. So a player in this mod set reads "This game has none
// of the science packs this research names" on the research screen and is told
// NOTHING there about the text they typed: that fact is in the log alone, which
// is not one of the three places a player looks. It is the library's own
// still-open limitation rather than this mod's to fix.
func TestAFallbackThisModsOwnPackListCannotPayForIsEmittedFreeBesideTheSetAsideText(t *testing.T) {
	const what = "a typed pack list set aside onto a declared list nothing pays for"
	w := everythingWorld().
		withStartup(SettingTechPacks, "1 water").
		withTools("logistic-science-pack")

	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, what, got, customUnit(20, 15))
	checkPrereqs(t, what, got, TechLogistics)
	checkComposedNote(t, what, got, "technology", TechName,
		packlessNote, []string{packlessNote})

	want := []string{
		"fkrecipes: bbb-balancer: automation-science-pack is not a science " +
			"pack this game has, so it is left out of the logistics cost",
		"fkrecipes: ERROR: bbb-balancer: the logistics cost names no science " +
			"pack this game has, so this mod's own declared cost applies instead",
		"fkrecipes: bbb-balancer: none of automation-science-pack is present, " +
			"so the science pack is dropped",
		packlessLine,
		packFallbackLine(`better-belt-balancer-tech-packs, entry 1 ` +
			`("1 water"): water is a fluid, and research takes science packs only`),
	}
	if !reflect.DeepEqual(logs, want) {
		t.Errorf("%s: the plan's log stream is\n got  %q\n want %q",
			what, logs, want)
	}
}

// TestAPackTheGameHasOnlyAsAnItemIsDroppedAndTheResearchIsEmittedFree is the
// OTHER side of the tool question, and it is the test above with nobody typing
// anything.
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
// hand-rolled research cost lose its packs". It is the in-miniature form of the
// demoted-pack mod set agents/migration-assessment-2.md measured on a real
// engine as finding 13, and agents/migration-assessment-3.md re-measured
// unchanged in outcome, where a pack this mod names is a plain item after
// another mod has had its say.
//
// AND WHAT A LOST LAST PACK GETS IS A FREE RESEARCH WITH A LINE AND A NOTE,
// where until FkRecipes 97a4493 it got a refusal; the name said so and does not
// any more.
// [TestAReachedFallbackWithNoPackInTheGameIsEmittedFreeAndDisclosed] carries
// the measurement that decided it. The two games are still not the same game:
// there no technology carries a unit at all, so the only rung ever asked about
// is this mod's declared fallback pack and the technology comes out with no
// prerequisite; here the chosen tier DOES carry a unit, its copied
// `automation-science-pack` is asked about and lost first, the declared
// fallback then names the same name a second time, and the prerequisite the
// tier gave the technology STAYS. What separates the two tests is the walk
// rather than the wording, and the walk is what the stream below pins.
//
// THE DROP LINES ARE ASSERTABLE NOW, WHICH THEY WERE NOT. A refused plan hands
// back the refusal and no ops at all, so the four lines this walk writes were
// invisible to this test while it expected a refusal; it said so in as many
// words. They are here.
func TestAPackTheGameHasOnlyAsAnItemIsDroppedAndTheResearchIsEmittedFree(t *testing.T) {
	const what = "a science pack this game has only as an ordinary item"
	w := everythingWorld().
		withItems(append(ladderVocabulary(), "automation-science-pack")...).
		withTools("logistic-science-pack")

	protos, logs := extendsOf(t, dataOps(t, w))
	got := protoOf(t, protos, "technology", TechName)

	checkUnit(t, what, got, customUnit(20, 15))
	checkPrereqs(t, what, got, TechLogistics)
	checkComposedNote(t, what, got, "technology", TechName,
		packlessNote, []string{packlessNote})

	// THE TEST ABOVE'S STREAM LESS ITS LAST LINE, because nothing was typed
	// here. Written out rather than shared with it: the two streams being the
	// same four lines is the claim, and a helper composing both would assert
	// itself.
	want := []string{
		"fkrecipes: bbb-balancer: automation-science-pack is not a science " +
			"pack this game has, so it is left out of the logistics cost",
		"fkrecipes: ERROR: bbb-balancer: the logistics cost names no science " +
			"pack this game has, so this mod's own declared cost applies instead",
		"fkrecipes: bbb-balancer: none of automation-science-pack is present, " +
			"so the science pack is dropped",
		packlessLine,
	}
	if !reflect.DeepEqual(logs, want) {
		t.Errorf("%s: the plan's log stream is\n got  %q\n want %q",
			what, logs, want)
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
