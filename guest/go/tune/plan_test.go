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
// settings of three shapes carry three different field lists, so `fields` names
// what each one may have rather than one list covering all of them, and it is
// derived from the transcription below rather than from the plan: a dropdown
// has `allowed_values`, a text setting has `auto_trim`, a numeric one has
// `minimum_value` and `maximum_value`, and a `localised_description` appears on
// ALL SIX since fix round 2, which is what moved the two numbers out of the
// no-description column they used to sit in. The emission order is FkRecipes
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
	// stage can ever read -- and beside a research dropdown the MINIMUM is not
	// this mod's to choose at all, because 0 is what a number says instead of
	// the reserved word `default` and the library refuses any other floor by
	// name.
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
			// SIX VALUES, THE SIX THAT SHIPPED, IN THEIR ORDER. Factorio keys a
			// stored startup choice by this string and RESETS one the running
			// release does not offer, so a value that moved, changed spelling or
			// was added and later withdrawn is a player's preference silently
			// discarded. This list has not moved since 0.2.2 and the customizer
			// did not move it either: the text field beside the dropdown is what
			// switches, not a seventh row.
			allowed: []string{
				"vanilla", "cheap", "belt-fast", "belt-express",
				"splitter", "splitter-express",
			},
			// THE COMPOSED DESCRIPTION, transcribed rung by rung: this mod's own
			// description key, then one line per PRESET -- the value's own
			// locale entry followed by that preset's list in the language the
			// text field takes -- and then the library's two closing sentences.
			// Each preset shows the FIRST rung of every ladder, because the
			// settings stage has no data.raw to walk one with.
			//
			// THE WRAP LINE AND THE LADDER LINE ARE ON THIS DROPDOWN AND ON
			// NEITHER RESEARCH ROW, and that asymmetry is the library's own rule
			// rather than an omission here: both go wherever a TYPEABLE LIST is
			// rendered, and a research preset renders a localised technology
			// name with nothing in it to copy. THE SWITCH LINE IS LAST on both.
			description: fkrecipes.Arr(
				fkrecipes.Str(""),
				localeRef("mod-setting-description", "bbb-recipe-cost", "bbb-recipe-cost"),
				presetLine("bbb-recipe-cost", "vanilla", "4 iron-plate, 2 iron-gear-wheel, 2 transport-belt"),
				presetLine("bbb-recipe-cost", "cheap", "2 iron-plate, 1 transport-belt"),
				presetLine("bbb-recipe-cost", "belt-fast", "4 iron-plate, 2 iron-gear-wheel, 2 fast-transport-belt"),
				presetLine("bbb-recipe-cost", "belt-express", "4 steel-plate, 2 iron-gear-wheel, 2 express-transport-belt"),
				presetLine("bbb-recipe-cost", "splitter", "1 splitter, 2 iron-plate"),
				presetLine("bbb-recipe-cost", "splitter-express", "1 express-splitter, 2 steel-plate"),
				fkrecipes.Str(wrapSentence),
				fkrecipes.Str(dropdownLadderSentence),
				fkrecipes.Str(dropdownSwitchSentence),
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
			// EIGHT PARAMETERS, and only the first two are this mod's: its own
			// description key, then the declared list rendered in the language
			// the field takes, then the library's own FIVE sentences. See
			// [textFieldLines], and `true` because this is the INGREDIENT field:
			// its format line names the word `none` and its ladder line names
			// what a player crafts where the pack field's names a research.
			description: fkrecipes.Arr(append([]fkrecipes.Value{
				fkrecipes.Str(""),
				localeRef("mod-setting-description",
					"better-belt-balancer-recipe-ingredients",
					"better-belt-balancer-recipe-ingredients"),
				fkrecipes.Str("\ndefault: 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt"),
			}, textFieldLines(true)...)...),
		},
		{
			kind:         "string-setting",
			name:         "bbb-tech-cost",
			settingType:  "startup",
			defaultValue: fkrecipes.Str("logistics"),
			order:        "b",
			// THREE VALUES, THE THREE THAT SHIPPED, IN THEIR ORDER, which is the
			// recipe dropdown's statement made a second time and for the same
			// reason: a stored `logistics-3` still reads `logistics-3`, because
			// the three fields beside this row switch themselves and the row
			// never grew one.
			allowed: []string{"logistics", "logistics-2", "logistics-3"},
			// THE COMPOSED DESCRIPTION, and its lines say something different
			// from the recipe dropdown's. A research preset has no list to
			// render -- what it costs is whatever the game charges for that
			// technology, which the settings stage has no data.raw to read -- so
			// the library writes the SOURCE it would copy from, and names it by
			// the technology's LOCALISED name rather than by its internal one:
			// the tail is `": cost of "` followed by that technology's own
			// `[technology-name]` key in the alternatives form (FkRecipes
			// go/customize.go, costPresetTail). Each source is the FIRST rung
			// of that option's ladder in [TechLadder] -- `logistics`,
			// `logistics-2` and `logistics-3` are each their own tier's head --
			// which is the tier the player asked for; where a ladder steps down
			// is a fact about their mod set and is not knowable here.
			//
			// AND THE KEY IS THE GAME'S, WHICH COSTS THIS MOD NOTHING EITHER
			// WAY. `technology-name.<source>` names a technology base or some
			// other mod declared, so nothing new is owed to the locale checker
			// and nothing may be offered to it either: defining one of those
			// keys here would rename base's technology for every mod in the
			// game, which is [TestTheLocaleFileRenamesNoTechnologyOfTheGames]'s
			// subject. Where the technology or its entry is missing the
			// alternatives form degrades to the raw internal name and the
			// tooltip survives whole, where the bare key it replaced used to
			// delete the entire tooltip in silence.
			description: fkrecipes.Arr(
				fkrecipes.Str(""),
				localeRef("mod-setting-description", "bbb-tech-cost", "bbb-tech-cost"),
				costPresetLine("bbb-tech-cost", "logistics", "logistics"),
				costPresetLine("bbb-tech-cost", "logistics-2", "logistics-2"),
				costPresetLine("bbb-tech-cost", "logistics-3", "logistics-3"),
				fkrecipes.Str(dropdownSwitchSentence),
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
			// The recipe text field's eight parameters again, with this mod's
			// own rendered default in the middle and the library's same five
			// sentences after it -- LESS THE WORD `none` and IN THE PACK
			// VOCABULARY, which is why [textFieldLines] takes the kind. A pack
			// list refuses that word ("research takes at least one science
			// pack"), so the library does not offer it here, and its ladder
			// line is about a research taking fewer packs rather than about a
			// craft costing less.
			description: fkrecipes.Arr(append([]fkrecipes.Value{
				fkrecipes.Str(""),
				localeRef("mod-setting-description",
					"better-belt-balancer-tech-packs",
					"better-belt-balancer-tech-packs"),
				fkrecipes.Str("\ndefault: 1 automation-science-pack"),
			}, textFieldLines(false)...)...),
		},
		{
			// THE UNIT COUNT. An int setting, so `default_value` is a NUMBER,
			// and it is 0 BECAUSE THE ROW ABOVE DECIDES: beside a research
			// dropdown the library demands a declared default and a minimum of
			// 0 and refuses anything else by name, because 0 is what a number
			// says instead of the reserved word `default`. The ceiling is this
			// mod's own and is the one bound it chooses.
			kind:         "int-setting",
			name:         "better-belt-balancer-tech-count",
			settingType:  "startup",
			defaultValue: fkrecipes.Num(0),
			order:        "bae",
			bounds:       []float64{0, 1000000},
			description:  numberDescription("better-belt-balancer-tech-count", "1000000"),
		},
		{
			// THE SECONDS PER UNIT. An INT setting since fix round 2, where it
			// was a double: the field carries a research time in whole seconds,
			// and the library's `CustomCost.Seconds` takes an `IntSettingRef`
			// and nothing else. Its bounds are the count's for the count's
			// reason.
			//
			// THE TYPE CHANGE IS THE ONE THING HERE A PLAYER COULD FEEL, and it
			// is measured rather than assumed: a stored non-integral double
			// under a setting redeclared as an int is truncated toward zero and
			// then range checked, with the declared default applying out of
			// range and nothing logged in any path. Nobody outside this machine
			// has stored one, because this name has never shipped.
			kind:         "int-setting",
			name:         "better-belt-balancer-tech-seconds",
			settingType:  "startup",
			defaultValue: fkrecipes.Num(0),
			order:        "baf",
			bounds:       []float64{0, 3600},
			description:  numberDescription("better-belt-balancer-tech-seconds", "3600"),
		},
	}
}

// localeRef is how the library references one locale entry from inside
// another: the engine's ALTERNATIVES form, `{"?", {"section.key"}, "raw"}`,
// where it used to write the bare `{"section.key"}`.
//
// THE THIRD ELEMENT IS WHAT THE PLAYER READS WHERE THE GAME HAS NO ENTRY, and
// it is a parameter here because it differs per key: a setting's description
// falls back to the setting's own emitted name, a dropdown value's label to the
// raw value, and a technology's name to its internal name. The library measured
// what the bare form cost -- an undefined key inside a composed description
// deletes the WHOLE tooltip on the settings screen and the whole description
// block on a recipe, with exit 0, no engine warning and the dump still carrying
// the string -- so this shape is the defect's cure and not decoration.
//
// THE RAW FALLBACK IS LAST AND THAT IS PART OF THE TRANSCRIPTION. A plain
// string alternative always resolves, so a raw string anywhere but the end
// short-circuits every alternative after it and the key would never be read.
func localeRef(section, key, raw string) fkrecipes.Value {
	return fkrecipes.Arr(
		fkrecipes.Str("?"),
		fkrecipes.Arr(fkrecipes.Str(section+"."+key)),
		fkrecipes.Str(raw),
	)
}

// The library's own sentences, transcribed. They are constants rather than
// literals at each site for the reason [textFieldLines] is one helper: they are
// the LIBRARY'S text, identical wherever it composes them, so a second
// hand-written spelling would be the drift the library's own corpus exists to
// prevent. What is this mod's -- the rendered default line, the preset lists --
// is written out at every site instead.
//
// TRANSCRIBED FROM WHAT THE PLAN EMITS AND CHECKED AGAINST THE LIBRARY'S OWN
// CONSTANTS, one by one: `listWrapLine`, `textFormatLine` with
// `ingredientNoneClause`, `textSwitchLine`, `textFallbackLine`,
// `dropdownSwitchLine` and `researchRangeLine` in FkRecipes go/customize.go.
const (
	wrapSentence = "\nA list too long for one line continues on the next; " +
		"the continuation is part of the same list."
	formatSentence = "\nWrite internal names, as the default line above does, " +
		"in at most 2000 characters."
	// THE WORD `none` IS AN INGREDIENT CLAUSE AND NOT A SENTENCE, appended to
	// the format line above on an ingredient list and on nothing else.
	noneClause = " The word none empties the list, so the recipe costs " +
		"nothing to craft."
	// ABOVE, on both text fields: `bbb-recipe-cost` sorts at "a" and its text
	// field at "aab", `bbb-tech-cost` at "b" and its pack field at "bad". The
	// library compares the EMITTED ORDER STRINGS rather than the declaration
	// order, so this word is a fact about where the rows land in the menu.
	textSwitchSentence = "\nWhile this says default the option chosen above " +
		"applies; anything else applies instead of it."
	// SET ASIDE AND THEN NOTHING ELSE HAPPENS, which is the wording FkRecipes
	// ca4c344 arrived at and this file's fourth spelling of the same fact. It
	// used to say "that default applies instead", which is false beside a
	// dropdown: what applies is the PRESET the player is standing on, and this
	// mod measured all six of them under one byte-identical sentence. What is
	// true on all six, and beside a field with no dropdown at all, is that the
	// field behaves as though it held the word `default`.
	textFallbackSentence = "\nA text this mod cannot use is set aside and the " +
		"field behaves as though it said default; the reason is in the log, " +
		"or in the load error if the load stops anyway."
	// And BELOW on both dropdowns, which is the same two pairs read from the
	// other end.
	dropdownSwitchSentence = "\nThe setting below applies instead while it " +
		"does not say default."

	// THE LADDER LINES, WHICH ARE THIS MOD'S OWN FINDING COMPOSED BY THE
	// LIBRARY. Every list this plan renders into a tooltip is a list of
	// LADDERS, and what the game builds from it can be shorter than what the
	// tooltip shows: a rung the mod set has takes an absent one's place, an
	// entry no rung answers for is left out, and two entries landing on one
	// name have their amounts added. None of that was disclosed on any
	// prototype, and FkRecipes justified the silence by citing a disclosure
	// "the dropdown's own composed description already discloses it" -- which
	// was THIS MOD'S locale entry for `mod-setting-description.bbb-recipe-cost`
	// and nothing the library wrote. That was finding 22 of
	// agents/migration-assessment-3.md; FkRecipes 933d4d3 composes the sentence
	// itself, so a consumer who never wrote that locale line gets it too.
	//
	// THREE SENTENCES AND NOT ONE, because the vocabulary is the list's. An
	// ingredient list shortens what a player CRAFTS and a pack list shortens
	// what a research TAKES, so the two text fields get different words for one
	// rule; and the ingredient DROPDOWN says "an option" rather than "a list
	// this mod chose", because the lists it is about are the presets the player
	// is choosing between.
	ingredientLadderSentence = "\nWhere a list this mod chose names something " +
		"your mods do not have, the next name it offers is used instead; an " +
		"entry it offers nothing for is left out, and two that land on one " +
		"name have their amounts added, so what you craft can be a shorter " +
		"list than the one shown."
	packsLadderSentence = "\nWhere a list this mod chose names a science pack " +
		"your mods do not have, the next name it offers is used instead; a " +
		"pack it offers nothing for is left out, and two that land on one " +
		"pack have their amounts added, so the research can take fewer packs " +
		"than the list shows."
	dropdownLadderSentence = "\nWhere an option names something your mods do " +
		"not have, the next name it offers is used instead; an entry it " +
		"offers nothing for is left out, and two that land on one name have " +
		"their amounts added, so what you craft can be a shorter list than " +
		"the one shown."
)

// presetLine is one preset's line of a dropdown's composed description: a
// newline, the VALUE'S OWN locale entry (the label the player sees in the
// menu, not the raw key), then what that preset means written out.
//
// IT IS TWO LINES PER PRESET SINCE FkRecipes 0077f3c, NOT ONE. The separator
// between the label and the list was `": "` and is `"\n  type: "`, so the label
// keeps its own line and the copyable internal list is indented under it. The
// reason is the consumer's own dropdown labels: this mod's are 35 to 65
// characters and the closed widget truncates at about 37, so on the old
// one-line shape the list a player was meant to copy began past the fold.
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
		localeRef("string-mod-setting", setting+"-"+value, value),
		fkrecipes.Str("\n  type: "+text),
	)
}

// textFieldLines is the FIVE sentences the LIBRARY appends to every text
// setting it composes a description for, under this mod's own rendered default
// line: the wrap, the ladder, the format, the switch and the fallback.
//
// TRANSCRIBED FROM WHAT THE PLAN EMITS AND THEN CHECKED SENTENCE BY SENTENCE
// AGAINST THE LIBRARY'S OWN CONSTANTS. Two of the five were here before fix
// round 2, two arrived with it, and the ladder line arrived with FkRecipes
// 933d4d3; the format line grew a clause on one of the two fields.
//
// THE LADDER LINE SITS SECOND, DIRECTLY UNDER THE WRAP LINE AND OVER THE
// FORMAT LINE, and the order is the library's rule rather than a preference:
// the wrap line names a RELATIONSHIP between the rendered default line and its
// continuation, so nothing may come between the list and it, and the format
// line under both still says "as the default line above does", which three
// lines up is as true as one.
//
// ONE HELPER RATHER THAN TWO COPIES, which is the one place this file derives
// where it otherwise transcribes, and the reason is what these sentences ARE:
// the LIBRARY'S, identical on both text settings by construction. What could
// differ between the two settings is the `\ndefault: ...` line ABOVE them,
// which is this mod's own rendering and IS written out twice.
//
// EXCEPT FOR THE TWO THINGS THAT ARE NOT IDENTICAL, which is why this takes an
// argument. `none` empties an ingredient list and is a legitimate thing for a
// player to want; a pack list REFUSES the word ("research takes at least one
// science pack"), so the library does not offer it there. And the ladder line
// is a WHOLE SEPARATE SENTENCE per field rather than a clause: a research unit
// is priced in packs and a recipe is crafted from a list, so "what you craft"
// names nothing on a technology and "fewer packs" names nothing on a recipe. A
// parameter is what makes "the ingredient field says it and the pack field does
// not" a thing this file can state and therefore get wrong.
//
// THE THIRD SENTENCE IS THE SWITCH AND IT IS FIX ROUND 2'S WHOLE SUBJECT. The
// settings screen has no conditional visibility at all, so it can never show
// which of the two rows is deciding; saying the RULE in both tooltips is what
// the library can do instead, and this is the text-field half of it.
//
// THE FOURTH IS THE FALLBACK, ANNOUNCED WHERE THE PLAYER TYPES. A text this mod
// cannot use is set aside, this mod's declared list applies, and the reason
// goes to the log. The clause "or in the load error if the load stops anyway"
// is the library saying the fallback is NOT total, which is
// [TestATextTheGameCannotAnswerFallsBackAndSaysSo]'s subject.
func textFieldLines(ingredients bool) []fkrecipes.Value {
	format := formatSentence
	if ingredients {
		format += noneClause
	}
	ladder := packsLadderSentence
	if ingredients {
		ladder = ingredientLadderSentence
	}
	return []fkrecipes.Value{
		fkrecipes.Str(wrapSentence),
		fkrecipes.Str(ladder),
		fkrecipes.Str(format),
		fkrecipes.Str(textSwitchSentence),
		fkrecipes.Str(textFallbackSentence),
	}
}

// numberDescription is the whole composed description of a RESEARCH NUMBER,
// which the two numeric settings carry since fix round 2 and carried nothing
// before it.
//
// THREE PARAMETERS AND NOT SEVEN. A number has no list to render, so there is
// no default line, no wrap line and no format line: what the library owes the
// player here is the RANGE, which the settings screen shows nowhere, and what 0
// means, which they could not guess. The ceiling is a string parameter because
// it is rendered by the library's own amount formatter rather than by Go's
// float printing -- 1000000 and not 1e+06 -- and this mod's two ceilings differ.
func numberDescription(setting, max string) fkrecipes.Value {
	return fkrecipes.Arr(
		fkrecipes.Str(""),
		localeRef("mod-setting-description", setting, setting),
		fkrecipes.Str("\nA whole number from 0 to "+max+
			". While it is 0 the option chosen above decides."),
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
// THE NAME AVOIDS `costLine`, WHICH IS A RULE ABOUT THE SHAPE AND NOT ABOUT
// TODAY'S CALLERS. plandata_test.go pins whole log sentences as local consts
// and reaches for names of exactly that shape when it does; a package-level
// function sharing one would be SHADOWED inside any test that declared it,
// which compiles, runs and reads as a mistake. Keeping the two vocabularies
// apart costs a word here and cannot be got wrong later.
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
		localeRef("string-mod-setting", setting+"-"+value, value),
		fkrecipes.Str(": cost of "),
		localeRef("technology-name", source, source),
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

// TestTheThreeResearchDeclarationsAreWhatThisFileClaims is the one thing the
// transcription above cannot say: WHY those three declared defaults are those
// three, which is two different reasons and not one.
//
// THE TWO NUMBERS DECLARE 0 AND 0 BECAUSE THE ROW ABOVE DECIDES. Beside a
// research dropdown the library takes 0 as the number's reserved word and
// refuses any other declared default by name, so a plan declaring [FallbackUnit]'s
// 20 and 15 here would not load at all -- and a player who had never opened the
// settings screen would be charged 20 units of 15 seconds whatever tier they
// were standing on, which is an override nobody asked for wearing a default's
// clothes.
//
// WHAT THIS TEST ADDS FOR THE TWO NUMBERS IS THE ANTI-VACUITY GUARD AND
// NOTHING ELSE, and the header says so rather than implying more. That the
// emitted `default_value` is 0 is already transcribed field by field in
// [wantSettings]; the two `checkNum` calls below restate it so this test reads
// as one statement, and the claim only this test can make is the one under
// them: that [FallbackUnit] charges a NON-ZERO count and a non-zero time, so
// "the two settings declare 0" is a statement that they are NOT that unit's
// numbers. Without that guard an editor who zeroed the fallback unit would make
// both readings agree for the wrong reason.
//
// THE PACK HALF IS THE ONE THAT IS NOT DERIVABLE FROM THE TRANSCRIPTION AT
// ALL, and it is why this test exists as a test rather than as a comment.
// [wantSettings] transcribes the rendered string `\ndefault: 1
// automation-science-pack`; what is checked here is that the string is what
// `FallbackUnit().Packs` RENDERS TO, so the settings screen cannot come to hold
// two different vanilla costs without one of the two readings moving.
//
// THE PACK TEXT IS THE OPPOSITE OF THE NUMBERS AND ITS DEFAULT IS STILL THAT
// UNIT'S. A text
// field's reserved word is `default`, so the field ships holding the word and
// the DECLARED list is what the word stands for where no tier answers; that
// list is `FallbackUnit().Packs`, and what is checked here is the RENDERING --
// that the declared pack reaches the player's tooltip as
// `1 automation-science-pack`, which is the text they copy to start from.
//
// # "ENDS WITH" WAS THE PREDICATE AND STOPPED BEING THE RIGHT ONE
//
// The rendered default was the LAST parameter of the composed description until
// FkRecipes 0077f3c. It is the THIRD of seven now, because the library appends
// four sentences of its own after it ([textFieldLines]). Reading `Arr[len-4]`
// instead would keep the shape of the old check and none of its meaning: it
// would pass on a library that appended a fifth sentence and moved this mod's
// line, and fail on one that appended a sixth while rendering the same list.
//
// SO THE PREDICATE IS "EXACTLY ONE PARAMETER IS A `\ndefault: ` LINE, AND IT IS
// THIS ONE", which is positional in nothing and is STRICTER than the old check
// in one way that matters: a description carrying two such lines -- the drift
// this test exists to catch, the declared default transcribed a second time
// somewhere -- failed "ends with" only if the second one came last, and fails
// this always.
func TestTheThreeResearchDeclarationsAreWhatThisFileClaims(t *testing.T) {
	unit := FallbackUnit()
	ops := planOps(t)

	for _, tc := range []struct{ setting, what string }{
		{SettingTechCount, "the research count"},
		{SettingTechSeconds, "the research time"},
	} {
		checkNum(t, tc.setting+" ("+tc.what+")",
			settingProto(t, ops, tc.setting), "default_value", 0)
	}
	if unit.Count == 0 || unit.Seconds == 0 {
		t.Errorf("FallbackUnit charges %d unit(s) of %v second(s), and a 0 in "+
			"either makes the two assertions above vacuous: what they say is "+
			"that the two settings do NOT ship holding this unit's numbers",
			unit.Count, unit.Seconds)
	}

	rendered := make([]string, 0, len(unit.Packs))
	for _, p := range unit.Packs {
		rendered = append(rendered, strconv.FormatInt(p.Amount, 10)+" "+p.Name)
	}
	want := fkrecipes.Str("\ndefault: " + strings.Join(rendered, ", "))
	desc := settingProto(t, ops, SettingTechPacks)["localised_description"]
	var defaults []fkrecipes.Value
	for _, param := range desc.Arr {
		if param.Kind == fkrecipes.KindStr &&
			strings.HasPrefix(param.Str, "\ndefault: ") {
			defaults = append(defaults, param)
		}
	}
	if len(defaults) != 1 || !reflect.DeepEqual(defaults[0], want) {
		t.Errorf("%s's description carries %d \"\\ndefault: \" line(s), %s, and "+
			"the fallback unit's packs render as %s; the whole description is %s",
			SettingTechPacks, len(defaults), showValues(defaults),
			showValue(want), showValue(desc))
	}
}

// showValues renders a slice of values for a failure message, which the one
// caller above needs because what it found may be none of them or two.
func showValues(vs []fkrecipes.Value) string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, showValue(v))
	}
	return "[" + strings.Join(out, ", ") + "]"
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
