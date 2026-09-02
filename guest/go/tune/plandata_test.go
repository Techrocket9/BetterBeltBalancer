package tune

import (
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

// protoNamed finds one prototype of the stream by its `name`, which is what
// lets the tests below stop depending on the order the library emits in. The
// ORDER is `PlanData`'s own documented one (items, then recipes, then
// technologies) and this mod has one of each, so nothing here needs it.
func protoNamed(t *testing.T, protos []fkrecipes.Value, name string) map[string]fkrecipes.Value {
	t.Helper()
	for _, p := range protos {
		fields := fieldsOf(t, p)
		if n, ok := fields["name"]; ok && n.Kind == fkrecipes.KindStr && n.Str == name {
			return fields
		}
	}
	t.Fatalf("no prototype named %q was emitted", name)
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
const dataPrototypeCount = 1

// TestTheItemIsTheOneThatShipped is the transcription, field by field.
//
// EIGHT FIELDS AND NO NINTH. A field this mod did not ask for is a field in the
// dump and therefore a moved golden hash, and one it stopped asking for is a
// silently different item -- an absent `place_result` is an item that builds
// nothing, which is the exact half round one could not have.
func TestTheItemIsTheOneThatShipped(t *testing.T) {
	protos, _ := extendsOf(t, dataOps(t, everythingWorld()))
	got := protoNamed(t, protos, PartName)

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
