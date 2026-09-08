package tune

import (
	"sort"

	fkrecipes "github.com/Techrocket9/fkrecipes/go"
)

// THE FIXTURE GAME, and it is the whole reason the plan is checkable at all.
//
// [fkrecipes.World] is the thirteen questions the planner is allowed to ask
// about the game outside its own plan, and the emit layer answers them out of
// data.raw. A host test answers them out of this, which is what lets `go test`
// say what the data stage will emit with no Factorio in the room and no wasm
// toolchain: the fields of every prototype, the ingredient every ladder picks
// in a game that is missing something, and the refusal a probe produces.
//
// IT EMBEDS [fkrecipes.UnimplementedWorld], WHICH IS WHAT THE LIBRARY BUILT
// FOR THE DAY THAT INTERFACE GROWS -- and it has now grown twice. Round two
// added `EntityExists` and this fixture stopped compiling with a Go error
// naming a method this mod had never heard of; round three added `FluidExists`
// and `ToolExists` and did it again (measured 2026-09-07 against FkRecipes
// c7a806e: `fixtureWorld does not implement fkrecipes.World (missing method
// FluidExists)`, once per PlanData call site: three then, four in the tree
// this commit leaves). The embed answers every question this
// fixture does not with a panic naming itself, so the NEXT question costs
// nothing until a plan of this mod's actually asks it.
//
// FluidExists WAS LEFT TO THAT PANIC AND THE CUSTOMIZER TOOK IT AWAY. Nothing
// this mod DECLARES is a fluid -- [RecipePlan]'s six plans name items and only
// items -- so while every ingredient came from those six the library never
// asked. `bbb-recipe-ingredients` is a text field a player writes anything
// into, and the language looks a bare name up among the ITEMS and then among
// the FLUIDS (FkRecipes go/ingredientlist.go:1074 and :1077), so every refusal
// path a typed name can take asks this question.
//
// THE OLD COMMENT'S PREDICTION HELD, AND IT WAS RE-MEASURED RATHER THAN
// ASSUMED. With this method deleted again,
// [TestACustomTextTheGameCannotAnswerIsRefused] answers
// `panic: fkrecipes: World.FluidExists is not implemented by this fixture`
// (go/world.go:168), through `resolveName` at go/ingredientlist.go:1077 --
// which is better than the `false` an unwritten stub would have returned.
//
// THE LIST STOCKS `water`, AND THE REFUSAL IT BUYS IS THE LIBRARY'S OWN. This
// mod's recipe declares no `Category`, so it is `crafting`, and
// `categoryTakesItemsOnly` (FkRecipes go/ingredientlist.go:110-127) is the
// engine's rule written down on the library's side of the boundary: a fluid
// under that category is refused at the list, by category name
// (go/ingredientlist.go:705), before any prototype is built. MEASURED on
// Factorio 2.0.77 with the packaged mod at `bbb-recipe-cost = custom` and
// `bbb-recipe-ingredients = "1 water"`, the load fails with `fkrecipes:
// bbb-recipe-ingredients, entry 1 ("1 water"): water is a fluid, and a recipe
// in the crafting category takes items only` -- the library's sentence, naming
// the setting and the entry, and not the engine's prototype error.
//
// SO THE FLUID IS STOCKED RATHER THAN WITHHELD, because every real game has
// `water` and a player who reaches for it is likelier than one who invents a
// name no mod defines. It is what lets
// [TestACustomTextTheGameCannotAnswerIsRefused] pin that sentence; with the
// list empty the name would fall out of `resolveName` as `no item or fluid is
// named water` and the category rule would be a branch no test here reaches.
//
// THE PACK FIELD HAS ITS OWN FLUID RULE AND THE SAME STOCKED NAME PINS IT.
// A pack list is looked up among the tools, then the items, then the fluids,
// and a fluid there is refused with `water is a fluid, and research takes
// science packs only` (the library's sentence, not the category one), which
// [TestAPackTextTheGameCannotAnswerIsRefused] pins; un-stock `water` and that
// row degrades to `no science pack is named water`, which is what says the
// fluid is load-bearing for it too.
//
// EVERY ANSWER IS A LIST RATHER THAN A MAP, and `TechNames` sorts a copy. The
// World contract says that method returns SORTED names and the library's cycle
// walk rests on it; a Go map's iteration order is randomised per run, so a
// fixture built on one would make a determinism claim it could not keep.
//
// THE FIXTURE IS DELIBERATELY UNGENEROUS. It answers absent for everything it
// was not told about, so a test that forgot to stock an ingredient sees the
// ladder step past it rather than a convenient yes.
type fixtureWorld struct {
	fkrecipes.UnimplementedWorld

	modName string

	// The names this game has, by family. `entities` is what `PlaceResult` is
	// probed against, `items` what every ingredient is, and `tools` what every
	// SCIENCE PACK is.
	//
	// A PACK IS ASKED OF ToolExists AND NOT OF ItemExists, since FkRecipes
	// c7a806e. A research unit takes tool-type items and nothing else, which
	// the library measured on the engine as `Invalid research unit
	// (iron-plate). Research unit(s) can only be tool type items at the
	// moment`, so its pack ladder asks that question rather than ItemExists --
	// and go/world.go warns in as many words that "a fixture that names a
	// science pack only in its items will see every hand-rolled research cost
	// lose its packs". The two packs therefore live in `tools` and NOT in
	// `items`: no ladder in this package names a science pack as an
	// ingredient, and listing them in both would leave `withItems` reading as
	// though it still drove the pack question.
	items    []string
	entities []string
	recipes  []string
	tools    []string
	// fluids is what a name the items do not have is looked up in next, and
	// only a name a PLAYER typed ever gets that far. See the header for why
	// `water` is stocked: a recipe in the `crafting` category takes items
	// only, and the refusal that says so is the library's.
	fluids []string

	// The technologies, with the unit each one carries. A technology with a
	// nil unit is present and unit-less, which is the research_trigger shape
	// the ladder has to step past.
	techs []fixtureTech

	// The startup settings this game answers with. A name that is not here is
	// UNREADABLE, which the planner degrades to the declared default plus a
	// log line -- so a test that means to drive a dropdown must stock it.
	startup map[string]string

	// startupRaw answers a name with a Value of any kind, and it is consulted
	// FIRST. `startup` can only produce a string, so without this the library's
	// "present but not a string" arm is a branch no fixture can reach -- and
	// that arm is not the same as an absent setting: it is what a mod that
	// redefined this mod's setting as an int would hand the planner.
	//
	// IT DOES NOT MODEL THE ENGINE'S BOUNDS. A real game RESETS a stored number
	// outside its setting's declared range to the default; this map answers
	// whatever a test put in it, so a test driving 50 units cannot see a
	// maximum lowered below 50. The declared bounds are pinned as fields by
	// [TestEverySettingPrototypeIsTheOneThatShipped], and the engine enforcing
	// them is the gate's `tech-custom` arm's business.
	startupRaw map[string]fkrecipes.Value
}

type fixtureTech struct {
	name    string
	prereqs []string
	unit    *fkrecipes.Value
	trigger bool
	// maxLevel is a technology's level cap, which lives on the TECHNOLOGY and
	// not in its unit. None of this mod's sources carries one; the field is
	// here so a test can prove that, rather than so it can be assumed.
	maxLevel *fkrecipes.Value
}

func (w fixtureWorld) ModName() string { return w.modName }

func (w fixtureWorld) StartupSetting(name string) (fkrecipes.Value, bool) {
	if v, ok := w.startupRaw[name]; ok {
		return v, true
	}
	v, ok := w.startup[name]
	if !ok {
		return fkrecipes.Nil(), false
	}
	return fkrecipes.Str(v), true
}

// TechNames is SORTED, which the World contract demands and the cycle walk's
// determinism rests on. A copy, because the caller is handed the slice.
func (w fixtureWorld) TechNames() []string {
	out := make([]string, 0, len(w.techs))
	for _, t := range w.techs {
		out = append(out, t.name)
	}
	sort.Strings(out)
	return out
}

func (w fixtureWorld) TechPrereqs(name string) []string {
	if t, ok := w.tech(name); ok {
		return t.prereqs
	}
	return nil
}

func (w fixtureWorld) TechUnit(name string) (fkrecipes.Value, bool) {
	if t, ok := w.tech(name); ok && t.unit != nil {
		return *t.unit, true
	}
	return fkrecipes.Nil(), false
}

func (w fixtureWorld) TechMaxLevel(name string) (fkrecipes.Value, bool) {
	if t, ok := w.tech(name); ok && t.maxLevel != nil {
		return *t.maxLevel, true
	}
	return fkrecipes.Nil(), false
}

func (w fixtureWorld) TechHasResearchTrigger(name string) bool {
	t, ok := w.tech(name)
	return ok && t.trigger
}

func (w fixtureWorld) TechExists(name string) bool {
	_, ok := w.tech(name)
	return ok
}

func (w fixtureWorld) ItemExists(name string) bool   { return has(w.items, name) }
func (w fixtureWorld) EntityExists(name string) bool { return has(w.entities, name) }
func (w fixtureWorld) RecipeExists(name string) bool { return has(w.recipes, name) }

// ToolExists is the science-pack question, and THREE things in this mod's plan
// ask it now: a `CostBy` fallback's pack ladder, the custom arm's declared pack
// ladder when the text is untouched, and every name a player types into
// `bbb-tech-packs`. All three ask this and not [fixtureWorld.ItemExists], which
// is what [TestAPackTheGameHasOnlyAsAnItemIsDroppedAndThenRefused] pins from the
// outside.
func (w fixtureWorld) ToolExists(name string) bool { return has(w.tools, name) }

// FluidExists is asked only about a name a PLAYER typed into
// `bbb-recipe-ingredients` or `bbb-tech-packs`, and only after the items (or,
// in a pack list, the tools and then the items) answered no. See the header for
// what the one stocked fluid is for.
func (w fixtureWorld) FluidExists(name string) bool { return has(w.fluids, name) }

func (w fixtureWorld) tech(name string) (fixtureTech, bool) {
	for _, t := range w.techs {
		if t.name == name {
			return t, true
		}
	}
	return fixtureTech{}, false
}

func has(list []string, name string) bool {
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}

// unitOf is the shape Factorio's own technology unit has: a dictionary of
// count, time and the SHORT TUPLE ingredient form. The library copies whatever
// it reads VERBATIM, so a fixture that wrote the long dict form would be
// checking a copy of something no engine produces.
// rawUnit lets a fixture technology carry a unit that is PRESENT and is not a
// dictionary at all -- an array, a string, a number.
//
// The library steps past such a rung rather than copying it, on a guard whose
// own header calls it distinct from the absent-flag one, and neither `unitOf`
// nor an absent unit can reach it: one always builds a map and the other
// answers false. A technology whose `unit` is not a table is a real shape (a
// mod that assigned a string to it, or a value fkdata could not carry across
// the boundary faithfully), and copying it would be a technology researchable
// for free.
func rawUnit(v fkrecipes.Value) *fkrecipes.Value { return &v }

func unitOf(count, seconds float64, pack string, amount float64) *fkrecipes.Value {
	v := fkrecipes.Obj(
		fkrecipes.Pair("count", fkrecipes.Num(count)),
		fkrecipes.Pair("time", fkrecipes.Num(seconds)),
		fkrecipes.Pair("ingredients", fkrecipes.Arr(
			fkrecipes.Arr(fkrecipes.Str(pack), fkrecipes.Num(amount)))),
	)
	return &v
}

// everythingWorld is a game that has every name any ladder in this package can
// reach, the two science packs its research can be priced in, `water` for the
// one refusal a fluid earns, the three logistics technologies with DISTINCT
// units, the balancer part entity, and all six settings answering their
// declared defaults.
//
// THE THREE UNITS DIFFER ON PURPOSE. What `CostBy` promises is that the unit
// comes from the source the setting names, and three identical units would be
// satisfied by a planner that always copied the first.
func everythingWorld() fixtureWorld {
	return fixtureWorld{
		modName:  ModName,
		items:    ladderVocabulary(),
		tools:    []string{"automation-science-pack", "logistic-science-pack"},
		fluids:   []string{"water"},
		entities: []string{PartName},
		techs: []fixtureTech{
			{name: TechLogistics, unit: unitOf(20, 15, "automation-science-pack", 1)},
			{name: TechLogistics2, prereqs: []string{TechLogistics},
				unit: unitOf(200, 30, "logistic-science-pack", 1)},
			{name: TechLogistics3, prereqs: []string{TechLogistics2},
				unit: unitOf(300, 15, "logistic-science-pack", 2)},
		},
		// ALL SIX SETTINGS ANSWER, THE TWO TEXT ONES WITH THE RESERVED WORD,
		// which is what a real game hands the planner: Factorio stores every
		// setting's current value in mod-settings.dat, untouched defaults
		// included (measured by the library, FkRecipes go/lib.go:230), so a
		// player who never opened the settings screen still answers `default`
		// here. A fixture that left it absent would model the hand-edited file
		// instead, which is [TestAnUnreadableCustomTextTakesTheDeclaredList]'s
		// world and is reached through `withoutStartup`.
		startup: map[string]string{
			SettingRecipeCost:        RecipeDefault(),
			SettingRecipeIngredients: theDefaultWord,
			SettingTechCost:          TechDefault(),
			SettingTechPacks:         theDefaultWord,
		},
		// THE TWO NUMBERS CANNOT GO IN THE MAP ABOVE, which is why they are
		// here rather than beside their siblings: `startup` produces a string
		// and the library reads a research count through `KindNum`
		// (FkRecipes go/customize.go:825, readNumber), so a string would be
		// UNREADABLE and every custom arm would carry a degradation line and
		// the declared default. These are the declared defaults said in the
		// kind the engine stores them in.
		startupRaw: map[string]fkrecipes.Value{
			SettingTechCount:   fkrecipes.Num(20),
			SettingTechSeconds: fkrecipes.Num(15),
		},
	}
}

// withNumberStartup is `withStartup` for the two numeric settings: the
// everything game with one of them answering another number.
//
// It goes through `startupRaw` because that is the only map that can hold a
// number, and `StartupSetting` consults it FIRST -- so a test that drives a
// number here and one that drives a dropdown through `withStartup` do not have
// to know about each other.
func (w fixtureWorld) withNumberStartup(name string, value float64) fixtureWorld {
	return w.withRawStartup(name, fkrecipes.Num(value))
}

// withoutNumberStartup is `withoutStartup` for the two numeric settings: the
// everything game with one of them answering ABSENT. It has to exist beside
// `withoutStartup` because the two numbers live in `startupRaw`, which that
// helper does not touch -- so without this, "the count row is missing from a
// hand-edited mod-settings.dat" would be a state no fixture could reach.
func (w fixtureWorld) withoutNumberStartup(name string) fixtureWorld {
	next := map[string]fkrecipes.Value{}
	for k, v := range w.startupRaw {
		if k != name {
			next[k] = v
		}
	}
	w.startupRaw = next
	return w
}

// withStartup is the one-variable-at-a-time driver: the everything game with
// one dropdown answering something else.
func (w fixtureWorld) withStartup(name, value string) fixtureWorld {
	next := map[string]string{}
	for k, v := range w.startup {
		next[k] = v
	}
	next[name] = value
	w.startup = next
	return w
}

// withoutStartup makes one setting answer ABSENT, which is what a hand-edited
// mod-settings.dat missing a row hands the planner and the one thing
// `withRawStartup` cannot express: a present value of the wrong type is refused
// for a text setting, where an absent one degrades to the declared default with
// a line saying so.
func (w fixtureWorld) withoutStartup(name string) fixtureWorld {
	next := map[string]string{}
	for k, v := range w.startup {
		if k != name {
			next[k] = v
		}
	}
	w.startup = next
	return w
}

// withItems replaces the game's whole item vocabulary, which is how a modpack
// that is missing something is expressed here.
func (w fixtureWorld) withItems(items ...string) fixtureWorld {
	w.items = items
	return w
}

// withTools replaces the game's whole SCIENCE PACK vocabulary, which is how a
// pack that renamed or removed the science packs is expressed here. Called with
// nothing it is a game with no science pack at all, which is the game a
// `CostBy` fallback cannot be paid for in.
func (w fixtureWorld) withTools(tools ...string) fixtureWorld {
	w.tools = tools
	return w
}

// withTechs replaces the game's whole technology set.
func (w fixtureWorld) withTechs(techs ...fixtureTech) fixtureWorld {
	w.techs = techs
	return w
}

// withRawStartup makes one setting answer PRESENT with a value of any kind,
// which is how "the setting is there and is not a string" is expressed here.
func (w fixtureWorld) withRawStartup(name string, v fkrecipes.Value) fixtureWorld {
	next := map[string]fkrecipes.Value{}
	for k, val := range w.startupRaw {
		next[k] = val
	}
	next[name] = v
	w.startupRaw = next
	return w
}

// withoutEntities is the game where this mod's own entity has not been defined
// yet, which is what a `fk_data` hook that ran EmitData before entity() would
// hand the planner.
func (w fixtureWorld) withoutEntities() fixtureWorld {
	w.entities = nil
	return w
}
