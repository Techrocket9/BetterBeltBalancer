package tune

import (
	"reflect"
	"sort"
	"strconv"
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
//
// THE FIELD SET IS PART OF THE TRANSCRIPTION since the customizer landed. Six
// settings of four shapes carry four different field lists, so `fields` names
// what each one may have rather than one list covering all of them, and it is
// derived from the transcription below rather than from the plan: a dropdown
// has `allowed_values`, a text setting has `auto_trim`, a numeric one has
// `minimum_value` and `maximum_value`, and a `localised_description` appears on
// the four the library composes one for. The emission order is FkRecipes
// go/settings.go's, whose own note puts the description last "because it is the
// bulkiest field and because it is composed out of everything above it"; the
// comparison is order-insensitive, so what is checked here is the SET.
type wantSetting struct {
	settingType string
	name        string
	kind        string
	// defaultValue is a Value rather than a string because two of the six are
	// NUMBERS: an int setting's `default_value` is emitted as a number, and a
	// prototype that stringified it would be a different prototype.
	defaultValue fkrecipes.Value
	order        string
	allowed      []string
	autoTrim     bool
	// bounds is {minimum_value, maximum_value} for the two numeric settings and
	// nil for every other kind. NEITHER BOUND IS A BALANCE OPINION: the engine
	// RESETS a stored number outside its own bounds to the default rather than
	// clamping it, so what is declared is exactly the set of values the data
	// stage can ever read -- and the library refuses a custom research cost
	// whose count could be below 1 or whose seconds could be 0, because the
	// engine refuses a unit with either.
	bounds []float64
	// description is the composed localised_description, or the zero Value for
	// a setting the library composes none for.
	description fkrecipes.Value
}

// fields is every key this prototype may carry, from what was transcribed.
func (w wantSetting) fields() []string {
	out := []string{"type", "name", "setting_type", "default_value", "order"}
	if w.allowed != nil {
		out = append(out, "allowed_values")
	}
	if w.autoTrim {
		out = append(out, "auto_trim")
	}
	if w.bounds != nil {
		out = append(out, "minimum_value", "maximum_value")
	}
	if w.description.Kind != fkrecipes.KindNil {
		out = append(out, "localised_description")
	}
	return out
}

func wantSettings() []wantSetting {
	return []wantSetting{
		{
			kind:         "string-setting",
			name:         "bbb-recipe-cost",
			settingType:  "startup",
			defaultValue: fkrecipes.Str("vanilla"),
			order:        "a",
			// SEVEN VALUES, THE SIX THAT SHIPPED IN THEIR ORDER AND `custom`
			// LAST. Factorio keys a stored startup choice by this string, so a
			// value that moved or changed spelling is a player's preference
			// silently discarded; the seventh is the only row nobody can have
			// stored.
			allowed: []string{
				"vanilla", "cheap", "belt-fast", "belt-express",
				"splitter", "splitter-express", "custom",
			},
			// THE COMPOSED DESCRIPTION, transcribed rung by rung: this mod's own
			// description key, then one line per PRESET -- the value's own
			// locale entry followed by that preset's list in the language the
			// text field takes. No line for `custom`, which has no preset behind
			// it. Each preset shows the FIRST rung of every ladder, because the
			// settings stage has no data.raw to walk one with.
			description: fkrecipes.Arr(
				fkrecipes.Str(""),
				localeKey("mod-setting-description.bbb-recipe-cost"),
				presetLine("bbb-recipe-cost", "vanilla", "4 iron-plate, 2 iron-gear-wheel, 2 transport-belt"),
				presetLine("bbb-recipe-cost", "cheap", "2 iron-plate, 1 transport-belt"),
				presetLine("bbb-recipe-cost", "belt-fast", "4 iron-plate, 2 iron-gear-wheel, 2 fast-transport-belt"),
				presetLine("bbb-recipe-cost", "belt-express", "4 steel-plate, 2 iron-gear-wheel, 2 express-transport-belt"),
				presetLine("bbb-recipe-cost", "splitter", "1 splitter, 2 iron-plate"),
				presetLine("bbb-recipe-cost", "splitter-express", "1 express-splitter, 2 steel-plate"),
			),
		},
		{
			// THE CUSTOMIZER'S TEXT FIELD. `default_value` is the WORD and not
			// the list: the engine stores every setting's current value,
			// untouched defaults included, so a rendered list would freeze a
			// silent player's recipe at the version they installed. The list is
			// in the description instead, after this mod's own key.
			kind:         "string-setting",
			name:         "better-belt-balancer-recipe-ingredients",
			settingType:  "startup",
			defaultValue: fkrecipes.Str("default"),
			order:        "aab",
			autoTrim:     true,
			description: fkrecipes.Arr(
				fkrecipes.Str(""),
				localeKey("mod-setting-description.better-belt-balancer-recipe-ingredients"),
				fkrecipes.Str("\ndefault: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt"),
			),
		},
		{
			kind:         "string-setting",
			name:         "bbb-tech-cost",
			settingType:  "startup",
			defaultValue: fkrecipes.Str("logistics"),
			order:        "b",
			// FOUR VALUES, THE THREE THAT SHIPPED IN THEIR ORDER AND `custom`
			// LAST, which is the recipe dropdown's migration made a second time
			// and for the same reason: a stored `logistics-3` still reads
			// `logistics-3`, and the fourth row is the only one nobody can have
			// stored.
			allowed: []string{"logistics", "logistics-2", "logistics-3", "custom"},
			// THE COMPOSED DESCRIPTION, and its lines say something different
			// from the recipe dropdown's. A research preset has no list to
			// render -- what it costs is whatever the game charges for that
			// technology, which the settings stage has no data.raw to read -- so
			// the library writes the SOURCE it would copy from, and names it by
			// the technology's LOCALISED name rather than by its internal one:
			// the tail is `": cost of "` followed by
			// `{"technology-name.<source>"}` where it used to be one string
			// ending in the bare name (FkRecipes go/customize.go:725,
			// costPresetTail). Each source is the FIRST rung of that option's
			// ladder in [TechLadder] -- `logistics`, `logistics-2` and
			// `logistics-3` are each their own tier's head -- which is the tier
			// the player asked for; where a ladder steps down is a fact about
			// their mod set and is not knowable here.
			//
			// AND THE TRADE IS THIS MOD'S LADDER TO ORDER, which the library
			// says out loud rather than hiding it. `technology-name.<source>`
			// is the GAME's entry, for a technology base or some other mod
			// declared, so nothing new is owed to the locale checker; but where
			// the technology or its entry is missing the engine renders
			// `Unknown key: "technology-name.<source>"` in the tooltip, where
			// the bare internal name used to stand. All three sources here are
			// base's own, so the marker takes a pack that removed a logistics
			// tier outright.
			description: fkrecipes.Arr(
				fkrecipes.Str(""),
				localeKey("mod-setting-description.bbb-tech-cost"),
				costPresetLine("bbb-tech-cost", "logistics", "logistics"),
				costPresetLine("bbb-tech-cost", "logistics-2", "logistics-2"),
				costPresetLine("bbb-tech-cost", "logistics-3", "logistics-3"),
			),
		},
		{
			// THE RESEARCH CUSTOMIZER'S PACK LIST, the second free-text field.
			// `default_value` is the word for the reason the recipe's is, and
			// the declared list is `1 automation-science-pack`, which is
			// [FallbackUnit]'s pack: base's own `logistics` cost.
			kind:         "string-setting",
			name:         "better-belt-balancer-tech-packs",
			settingType:  "startup",
			defaultValue: fkrecipes.Str("default"),
			order:        "bad",
			autoTrim:     true,
			description: fkrecipes.Arr(
				fkrecipes.Str(""),
				localeKey("mod-setting-description.better-belt-balancer-tech-packs"),
				fkrecipes.Str("\ndefault: 1 automation-science-pack"),
			),
		},
		{
			// THE UNIT COUNT. An int setting, so `default_value` is a NUMBER,
			// and both bounds are emitted: 1 is what the library demands of a
			// count (the engine refuses a unit count of 0) and the ceiling is
			// this mod's own.
			kind:         "int-setting",
			name:         "better-belt-balancer-tech-count",
			settingType:  "startup",
			defaultValue: fkrecipes.Num(20),
			order:        "bae",
			bounds:       []float64{1, 1000000},
		},
		{
			// THE SECONDS PER UNIT. A double setting, whose minimum has to be
			// ABOVE zero rather than at least 1 -- the engine refuses a unit
			// time of 0 -- and 1 second is this mod's floor rather than the
			// library's.
			kind:         "double-setting",
			name:         "better-belt-balancer-tech-seconds",
			settingType:  "startup",
			defaultValue: fkrecipes.Num(15),
			order:        "baf",
			bounds:       []float64{1, 3600},
		},
	}
}

// localeKey is a localised string that is nothing but a key, `{"section.key"}`,
// which is how one locale entry is referenced from inside another.
func localeKey(key string) fkrecipes.Value {
	return fkrecipes.Arr(fkrecipes.Str(key))
}

// presetLine is one preset's line of a dropdown's composed description: a
// newline, the VALUE'S OWN locale entry (the label the player sees in the
// menu, not the raw key), then what that preset means written out.
//
// THE SETTING IS A PARAMETER even though only the recipe dropdown's lines are
// built here now (the research dropdown's have a shape of their own, below),
// because the key is `<setting>-<value>` in a flat namespace: a line built
// under the wrong setting would point the player's tooltip at another row's
// label, and the argument is what makes that mistake writable and therefore
// failable.
func presetLine(setting, value, text string) fkrecipes.Value {
	return fkrecipes.Arr(
		fkrecipes.Str(""),
		fkrecipes.Str("\n"),
		localeKey("string-mod-setting."+setting+"-"+value),
		fkrecipes.Str(": "+text),
	)
}

// costPresetLine is one research tier's line of `bbb-tech-cost`'s description,
// and it is a second helper rather than a `presetLine` argument because the
// LINE'S SHAPE differs: a recipe preset's line ends in one string this mod's
// language rendered, and a cost preset's ends in TWO, the words and then the
// game's own entry for the technology. Both sit BESIDE the label at the same
// level rather than under it, so a cost line carries four parameters after its
// empty format string where a recipe preset's carries three.
//
// THE NAME IS NOT `costLine`, which two tests in plandata_test.go already use
// as a local const for a pinned sentence: a package-level function of that name
// would be shadowed inside them, which compiles and reads as a mistake.
//
// THE SOURCE IS A PARAMETER BESIDE THE VALUE even though this mod spells them
// the same. [TechOptions]' strings ARE the base technology names, so `logistics`
// names both the dropdown row and the technology, and folding the two into one
// argument would make a description that named the wrong technology under the
// right row unwritable here and therefore unfailable.
func costPresetLine(setting, value, source string) fkrecipes.Value {
	return fkrecipes.Arr(
		fkrecipes.Str(""),
		fkrecipes.Str("\n"),
		localeKey("string-mod-setting."+setting+"-"+value),
		fkrecipes.Str(": cost of "),
		localeKey("technology-name."+source),
	)
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

func TestTheSettingsPlanIsSixExtendsAndNothingElse(t *testing.T) {
	ops := planOps(t)
	if len(ops) != len(wantSettings()) {
		t.Fatalf("the settings stage emits %d op(s) and this mod has %d settings "+
			"to declare through the library", len(ops), len(wantSettings()))
	}
	for i, op := range ops {
		// An OpSet would be a write into somebody else's prototype and an OpLog
		// a degradation notice; neither belongs in a plan that declares two
		// dropdowns, two text fields and two numbers, and either would mean the
		// library was asked for something this mod did not ask for.
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
			"TestTheSettingsPlanIsSixExtendsAndNothingElse says which", len(ops), len(want))
	}
	for i, w := range want {
		got := fieldsOf(t, ops[i].Proto)
		checkStr(t, w.name, got, "type", w.kind)
		checkStr(t, w.name, got, "name", w.name)
		checkStr(t, w.name, got, "setting_type", w.settingType)
		if !reflect.DeepEqual(got["default_value"], w.defaultValue) {
			t.Errorf("%s's default_value is %s and shipped as %s",
				w.name, showValue(got["default_value"]), showValue(w.defaultValue))
		}
		checkStr(t, w.name, got, "order", w.order)
		if w.allowed != nil {
			checkAllowed(t, w.name, got, w.allowed)
		}
		if w.autoTrim {
			checkBool(t, w.name, got, "auto_trim", true)
		}
		if w.bounds != nil {
			checkNum(t, w.name, got, "minimum_value", w.bounds[0])
			checkNum(t, w.name, got, "maximum_value", w.bounds[1])
		}
		if w.description.Kind != fkrecipes.KindNil {
			if !reflect.DeepEqual(got["localised_description"], w.description) {
				t.Errorf("%s's localised_description is\n got  %s\n want %s",
					w.name, showValue(got["localised_description"]),
					showValue(w.description))
			}
		}

		// THESE FIELDS AND NO OTHER. A field this mod did not ask for is a
		// field in the dump, so it moves the golden hash and it reaches the
		// player's settings menu; the failure names it rather than saying a
		// count is wrong. What each setting may carry is transcribed with it,
		// so a text setting growing an `allowed_values` (a free-text field
		// turned into a picker) fails here rather than in a menu.
		for _, key := range sortedKeys(got) {
			if !has(w.fields(), key) {
				t.Errorf("%s carries an unexpected field %q", w.name, key)
			}
		}
	}
}

// settingProto is one emitted setting prototype BY NAME, so a test that cares
// about three of the six does not have to know which index they sit at.
func settingProto(t *testing.T, ops []fkrecipes.Op, name string) map[string]fkrecipes.Value {
	t.Helper()
	for _, op := range ops {
		got := fieldsOf(t, op.Proto)
		if v, ok := got["name"]; ok && v.Kind == fkrecipes.KindStr && v.Str == name {
			return got
		}
	}
	t.Fatalf("no setting named %q was emitted", name)
	return nil
}

// TestTheCustomResearchDefaultsAreTheFallbackUnit is the one thing the
// transcription above cannot say: that the three numbers a player who picks
// Custom and types nothing is charged are BASE'S OWN `logistics` UNIT, and not
// three numbers that happen to look like it.
//
// [FallbackUnit] is what this technology costs in a game whose logistics chain
// is gone, and it is what the custom arm defaults to as well, so the two have to
// be ONE statement rather than two transcriptions -- a settings screen holding
// two different vanilla costs is a thing only a dump would ever show.
//
// THE TWO NUMBERS ARE THE HALF WITH TEETH. They are literals in [Plan] and
// literals in [FallbackUnit], so this comparison can fail. The PACK half cannot:
// `better-belt-balancer-tech-packs` is declared FROM `FallbackUnit().Packs`, and
// what is checked there is the RENDERING -- that the declared pack reaches the
// player's tooltip as `1 automation-science-pack`, which is the text they copy
// to start from.
func TestTheCustomResearchDefaultsAreTheFallbackUnit(t *testing.T) {
	unit := FallbackUnit()
	ops := planOps(t)

	checkNum(t, SettingTechCount, settingProto(t, ops, SettingTechCount),
		"default_value", float64(unit.Count))
	checkNum(t, SettingTechSeconds, settingProto(t, ops, SettingTechSeconds),
		"default_value", unit.Seconds)

	rendered := make([]string, 0, len(unit.Packs))
	for _, p := range unit.Packs {
		rendered = append(rendered, strconv.FormatInt(p.Amount, 10)+" "+p.Name)
	}
	want := fkrecipes.Str("\ndefault: " + strings.Join(rendered, ", "))
	desc := settingProto(t, ops, SettingTechPacks)["localised_description"]
	if len(desc.Arr) == 0 || !reflect.DeepEqual(desc.Arr[len(desc.Arr)-1], want) {
		t.Errorf("%s's description ends with %s and the fallback unit's packs "+
			"render as %s", SettingTechPacks, showValue(desc), showValue(want))
	}
}

// generatedSettingNames is the four names [Plan] lets the LIBRARY build, as the
// BARE strings it hands the constructors.
//
// A LITERAL TABLE, for the reason every expectation in this file is one: the
// constants in tune.go are built from [ModName] and these same four strings, so
// a table that read them back would agree with a defect in them and say so
// cheerfully. What is written out here is what FkRecipes' docs/migration.md
// works through for this mod.
func generatedSettingNames() []string {
	return []string{"recipe-ingredients", "tech-packs", "tech-count", "tech-seconds"}
}

// TestEveryEmittedNameIsTheOneItsConstructorPromises is the naming half of both
// decisions in [Plan], stated as one assertion because the two halves are the
// same question asked of different rows.
//
// THE LEGACY HALF IS UNCHANGED AND IT IS THE ONE WITH A PLAYER BEHIND IT.
// FkRecipes prefixes every generated name with the mod's own, and a prefixed
// name on something already shipped would be a NEW setting to Factorio:
// mod-settings.dat is keyed by name with no rename mechanism, so every player
// who had chosen a value would silently get the default. The same argument with
// a wider blast radius covers the PROTOTYPES, since round two: a prototype name
// is held by blueprints, logistic requests, crafting queues, other mods'
// compatibility patches and this mod's own hand-rolled entity, and the engine's
// answer to a dangling one is not a warning but `Error in assignID: item with
// name '...' does not exist`. Round one measured exactly that when the library
// prefixed them. The failure that would catch either downstream is a moved
// golden hash, which says a hash moved and not what it means.
//
// THE GENERATED HALF IS THE SYNC PASS'S AND IT IS AN EQUALITY, not a prefix
// test. The four customizer fields have never shipped, so the library names
// them -- and what it must produce is exactly [ModName], one hyphen and the bare
// name, no more and no less. A `HasPrefix` here would pass a plan that emitted
// `better-belt-balancer-bbb-tech-packs`, which is the shape round one measured
// out of the prototypes and the one this test exists to make unwritable.
func TestEveryEmittedNameIsTheOneItsConstructorPromises(t *testing.T) {
	prefix := ModName + "-"
	settings := planOps(t)

	// EXACTLY ONE SETTING PER GENERATED NAME. Zero says the prefix or the bare
	// name moved; two would be a duplicate the engine refuses at load.
	generated := map[string]bool{}
	for _, bare := range generatedSettingNames() {
		want := prefix + bare
		generated[want] = true
		found := 0
		for _, op := range settings {
			got := fieldsOf(t, op.Proto)
			if name, ok := got["name"]; ok && name.Kind == fkrecipes.KindStr && name.Str == want {
				found++
			}
		}
		if found != 1 {
			t.Errorf("%d setting(s) are emitted as %q and exactly one has to be: "+
				"the constructor is handed the bare %q and the prefix is the "+
				"library's, derived from the packaged mod name", found, want, bare)
		}
	}

	// AND NOTHING ELSE THIS PLAN EMITS CARRIES THE PREFIX AT ALL, over the
	// settings and the data plan together.
	both := append(append([]fkrecipes.Op{}, settings...), dataOps(t, everythingWorld())...)
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
		if generated[name.Str] {
			continue
		}
		if strings.HasPrefix(name.Str, prefix) {
			t.Errorf("a prototype is emitted as %q: it went through a "+
				"prefixing constructor, and every player's saved choice or "+
				"blueprint under the old name is discarded", name.Str)
		}
	}
}

// TestTheSixOrdersSortIntoTheDeclarationOrder is what `OrderAfter` was called
// for, stated as the thing a player sees.
//
// FACTORIO SORTS A MOD'S SETTINGS BY `order` AND THEN BY NAME, so the menu is a
// STRING SORT over the six and not the order the declarations are written in.
// [TestEverySettingPrototypeIsTheOneThatShipped] compares each order string
// field by field, and six pinned strings determine the sort, so this test adds
// no claim that one cannot make: what it adds is a SECOND literal, written in
// declaration order, that fails by naming the row that landed elsewhere.
//
// THE EXPECTED LIST IS A LITERAL and it is in DECLARATION order on purpose:
// what is asserted is that the two agree. A plan whose generated orders landed
// somewhere else would still pass a per-field comparison written to match them.
//
// AND EACH GENERATED ORDER CARRIES ITS DROPDOWN'S OWN ORDER AS A PREFIX, which
// is the claim the transcription does not make and the reason the second table
// exists. Sorting after `b` is what `bad` and `zzz` have in common; being
// spelled `b` and then two letters is what says the field sits DIRECTLY under
// the row that switches it on. What keeps a legacy order from coming between
// them is the library's own refusal (FkRecipes go/settings.go, the placement
// check: a legacy order that extends the prefix and sorts before the generated
// one is refused by name), which this plan's one-letter orders never reach.
func TestTheSixOrdersSortIntoTheDeclarationOrder(t *testing.T) {
	// The six as [Plan] declares them, transcribed: the emitted name and the
	// order string beside it.
	want := []settingOrder{
		{"bbb-recipe-cost", "a"},
		{"better-belt-balancer-recipe-ingredients", "aab"},
		{"bbb-tech-cost", "b"},
		{"better-belt-balancer-tech-packs", "bad"},
		{"better-belt-balancer-tech-count", "bae"},
		{"better-belt-balancer-tech-seconds", "baf"},
	}

	got := settingOrdersOf(t, planOps(t))
	sort.Slice(got, func(i, j int) bool {
		if got[i].order != got[j].order {
			return got[i].order < got[j].order
		}
		return got[i].name < got[j].name
	})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the settings sort by (order, name) into\n got  %v\n want the "+
			"declaration order, %v", got, want)
	}

	// WHICH DROPDOWN EACH FIELD SITS UNDER, transcribed: the recipe's text field
	// under the recipe dropdown's `a`, the research's three under the research
	// dropdown's `b`.
	for _, tc := range []settingOrder{
		{"better-belt-balancer-recipe-ingredients", "a"},
		{"better-belt-balancer-tech-packs", "b"},
		{"better-belt-balancer-tech-count", "b"},
		{"better-belt-balancer-tech-seconds", "b"},
	} {
		order, emitted := "", false
		for _, s := range got {
			if s.name == tc.name {
				order, emitted = s.order, true
			}
		}
		if !emitted {
			// The comparison above already reports the list it did get; this
			// says which name it went looking for and could not find.
			t.Errorf("no setting named %s was emitted at all, so nothing here "+
				"says whether the row under %q is placed", tc.name, tc.order)
			continue
		}
		if !strings.HasPrefix(order, tc.order) || len(order) <= len(tc.order) {
			t.Errorf("%s carries the order %q and the dropdown it belongs to "+
				"carries %q: a generated order that does not extend its "+
				"dropdown's own is somewhere after it rather than under it",
				tc.name, order, tc.order)
		}
	}
}

// settingOrder is one emitted setting's name and order, which is the whole of
// what decides where its row lands.
type settingOrder struct {
	name  string
	order string
}

// settingOrdersOf reads the pair off every emitted setting prototype, in
// emission order.
func settingOrdersOf(t *testing.T, ops []fkrecipes.Op) []settingOrder {
	t.Helper()
	out := make([]settingOrder, 0, len(ops))
	for _, op := range ops {
		got := fieldsOf(t, op.Proto)
		name, order := got["name"], got["order"]
		if name.Kind != fkrecipes.KindStr || order.Kind != fkrecipes.KindStr {
			t.Fatalf("a setting prototype has no string name and order: %s",
				showValue(op.Proto))
		}
		out = append(out, settingOrder{name.Str, order.Str})
	}
	return out
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
