package tune

import (
	"sort"
	"strings"
	"testing"

	fkrecipes "github.com/Techrocket9/fkrecipes/go"
)

// WHAT THE SETTINGS STAGE EMITS, SAID ON THE HOST WITH NO ENGINE IN THE ROOM.
//
// `test/check-datastage.py` hashes the whole normalised mod-settings dump and
// is the gate that says nothing moved. It is also about three seconds of
// Factorio per arm, it needs a binary whose series matches the packaged mod,
// and when it fails it reports a hash rather than a field. This is the same
// six fields per setting, compared one at a time, in a `go test` that runs
// anywhere: a hash for "nothing changed", a field comparison for "this is what
// it is". The two are not redundant, they answer different questions.
//
// THE EXPECTED VALUES BELOW ARE TRANSCRIBED, NOT DERIVED. Building them from
// [RecipeOptions] and [TechOptions] would be a gate that computes its answer
// from the same source as the thing under test, so it would agree with a defect
// in that source and say so cheerfully. They are written out by hand from the
// pre-migration dump (scratchpad reference `ref-mod-settings-base.json` at the
// time of the migration), which is the only reading that can disagree.
//
// KEYS ARE COMPARED ORDER-INSENSITIVELY AND `allowed_values` IS NOT. The
// library builds a prototype's pairs in a fixed declaration order and fkdata
// sorts them on the way out, so pair order is not something this mod may hold
// the library to. The ORDER OF A DROPDOWN'S VALUES is the opposite: it is the
// order the rows appear in the settings menu, it is in the dump, and it is a
// player-visible property of this mod.

// planWorld is the least thing [fkrecipes.Lib.PlanSettings] can be asked
// anything through: ONE METHOD.
//
// It used to be nine stubs that were never called, because the settings planner
// took the whole eleven-method World for the sake of its mod name. The library
// splits [fkrecipes.Named] out now and PlanSettings takes that, so a consumer
// holding their settings plan up to the light implements what it actually asks
// and nothing else. The data half's fixture is world_test.go.
type planWorld struct{}

func (planWorld) ModName() string { return ModName }

// wantSetting is one expected prototype: every field, transcribed.
type wantSetting struct {
	settingType  string
	name         string
	kind         string
	defaultValue string
	order        string
	allowed      []string
}

func wantSettings() []wantSetting {
	return []wantSetting{
		{
			kind:         "string-setting",
			name:         "bbb-recipe-cost",
			settingType:  "startup",
			defaultValue: "vanilla",
			order:        "a",
			allowed: []string{
				"vanilla", "cheap", "belt-fast", "belt-express",
				"splitter", "splitter-express",
			},
		},
		{
			kind:         "string-setting",
			name:         "bbb-tech-cost",
			settingType:  "startup",
			defaultValue: "logistics",
			order:        "b",
			allowed:      []string{"logistics", "logistics-2", "logistics-3"},
		},
	}
}

// planOps runs the plan and insists it produced ops at all, so every test below
// is reading a real stream rather than an empty one.
func planOps(t *testing.T) []fkrecipes.Op {
	t.Helper()
	ops, err := Plan().PlanSettings(planWorld{})
	if err != nil {
		t.Fatalf("the settings plan was refused, so the settings stage would "+
			"raise this at load: %v", err)
	}
	return ops
}

func TestTheSettingsPlanIsTwoExtendsAndNothingElse(t *testing.T) {
	ops := planOps(t)
	if len(ops) != len(wantSettings()) {
		t.Fatalf("the settings stage emits %d op(s) and this mod has %d settings "+
			"to declare through the library", len(ops), len(wantSettings()))
	}
	for i, op := range ops {
		// An OpSet would be a write into somebody else's prototype and an OpLog
		// a degradation notice; neither belongs in a plan that declares two
		// dropdowns, and either would mean the library was asked for something
		// this mod did not ask for.
		if op.Kind != fkrecipes.OpExtend {
			t.Errorf("op %d is kind %d and every op of a settings plan is an "+
				"OpExtend (%d)", i, op.Kind, fkrecipes.OpExtend)
		}
	}
}

func TestEverySettingPrototypeIsTheOneThatShipped(t *testing.T) {
	ops := planOps(t)
	want := wantSettings()
	if len(ops) != len(want) {
		t.Fatalf("%d op(s) against %d expected setting(s); "+
			"TestTheSettingsPlanIsTwoExtendsAndNothingElse says which", len(ops), len(want))
	}
	for i, w := range want {
		got := fieldsOf(t, ops[i].Proto)
		checkStr(t, w.name, got, "type", w.kind)
		checkStr(t, w.name, got, "name", w.name)
		checkStr(t, w.name, got, "setting_type", w.settingType)
		checkStr(t, w.name, got, "default_value", w.defaultValue)
		checkStr(t, w.name, got, "order", w.order)
		checkAllowed(t, w.name, got, w.allowed)

		// SIX FIELDS AND NO SEVENTH. A field this mod did not ask for is a
		// field in the dump, so it moves the golden hash and it reaches the
		// player's settings menu; the failure names it rather than saying a
		// count is wrong.
		for _, key := range sortedKeys(got) {
			switch key {
			case "type", "name", "setting_type", "default_value", "order", "allowed_values":
			default:
				t.Errorf("%s carries an unexpected field %q", w.name, key)
			}
		}
	}
}

// TestNoEmittedNameCarriesTheGeneratedPrefix is the whole point of the Legacy
// constructors, stated as an assertion.
//
// FkRecipes prefixes every generated name with the mod's own, and a prefixed
// setting name would be a NEW setting to Factorio: mod-settings.dat is keyed by
// name with no rename mechanism, so every player who had chosen a value would
// silently get the default. The failure that would catch it downstream is a
// moved golden hash, which says a hash moved and not what it means.
//
// AND IT COVERS THE PROTOTYPES TOO SINCE ROUND TWO, where the argument is the
// same one with a wider blast radius. A prototype name is held by blueprints,
// logistic requests, crafting queues, other mods' compatibility patches and
// this mod's own hand-rolled entity, and the engine's answer to a dangling one
// is not a warning but `Error in assignID: item with name '...' does not
// exist`. Round one measured exactly that when the library prefixed them.
func TestNoEmittedNameCarriesTheGeneratedPrefix(t *testing.T) {
	prefix := ModName + "-"
	both := append(planOps(t), dataOps(t, everythingWorld())...)
	for _, op := range both {
		if op.Kind != fkrecipes.OpExtend {
			continue
		}
		got := fieldsOf(t, op.Proto)
		name, ok := got["name"]
		if !ok || name.Kind != fkrecipes.KindStr {
			t.Error("a prototype has no string `name` field")
			continue
		}
		if strings.HasPrefix(name.Str, prefix) {
			t.Errorf("a prototype is emitted as %q: it went through a "+
				"prefixing constructor, and every player's saved choice or "+
				"blueprint under the old name is discarded", name.Str)
		}
	}
}

// fieldsOf flattens a prototype's pairs into a map, which is what makes the
// comparison above insensitive to the order the library builds them in. A
// duplicate key is reported rather than silently overwritten: fkdata would keep
// one of the two and nothing else would ever say which.
func fieldsOf(t *testing.T, v fkrecipes.Value) map[string]fkrecipes.Value {
	t.Helper()
	if v.Kind != fkrecipes.KindMap {
		t.Fatalf("a setting prototype came out as kind %d rather than a map", v.Kind)
	}
	out := map[string]fkrecipes.Value{}
	for _, pair := range v.Map {
		if _, dup := out[pair.Key]; dup {
			t.Errorf("the prototype carries %q twice", pair.Key)
		}
		out[pair.Key] = pair.Val
	}
	return out
}

func sortedKeys(m map[string]fkrecipes.Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func checkStr(t *testing.T, setting string, got map[string]fkrecipes.Value, key, want string) {
	t.Helper()
	v, ok := got[key]
	if !ok {
		t.Errorf("%s has no %s field", setting, key)
		return
	}
	if v.Kind != fkrecipes.KindStr {
		t.Errorf("%s's %s came out as kind %d rather than a string", setting, key, v.Kind)
		return
	}
	if v.Str != want {
		t.Errorf("%s's %s is %q and shipped as %q", setting, key, v.Str, want)
	}
}

// checkAllowed compares the dropdown's values IN ORDER, which is the one thing
// about a prototype's contents this test is order-sensitive about: it is the
// order of the rows in the settings menu.
func checkAllowed(t *testing.T, setting string, got map[string]fkrecipes.Value, want []string) {
	t.Helper()
	v, ok := got["allowed_values"]
	if !ok {
		t.Errorf("%s has no allowed_values, so it is not a dropdown at all", setting)
		return
	}
	if v.Kind != fkrecipes.KindArr {
		t.Errorf("%s's allowed_values came out as kind %d rather than an array", setting, v.Kind)
		return
	}
	if len(v.Arr) != len(want) {
		t.Errorf("%s offers %d value(s) and shipped %d: %s against %s",
			setting, len(v.Arr), len(want), showArr(v), strings.Join(want, ", "))
		return
	}
	for i, w := range want {
		if v.Arr[i].Kind != fkrecipes.KindStr {
			t.Errorf("%s's allowed value %d is kind %d rather than a string",
				setting, i, v.Arr[i].Kind)
			continue
		}
		if v.Arr[i].Str != w {
			t.Errorf("%s's allowed value %d is %q and shipped as %q",
				setting, i, v.Arr[i].Str, w)
		}
	}
}

func showArr(v fkrecipes.Value) string {
	parts := make([]string, 0, len(v.Arr))
	for _, e := range v.Arr {
		parts = append(parts, e.Str)
	}
	return strings.Join(parts, ", ")
}
