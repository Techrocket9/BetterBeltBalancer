// Package protos is the names and numbers a suite's OBSERVER and its own DATA
// STAGE both have to agree about, written down once.
//
// # Why a package with nothing in it but constants
//
// An observer with a data stage is TWO wasm modules -- `obs/<suite>` and
// `obs/<suite>data` -- and they cannot share an ordinary package, because the
// control guest imports fkapi and the data guest imports fkdata and packaging
// refuses either one holding the other's. So the loader prototype a data stage
// DEFINES and the loader name the observer PLACES were written down twice per
// suite, and phase 2 recorded that as a deviation rather than fixing it: three
// of five data stages sharing a package is worse than none.
//
// PHASE 3 MADE IT FIVE OF FIVE (`mix`, `plat` and `mig` all bring a loader), so
// this is that package. It imports NOTHING -- not fkapi, not fkdata, not the
// standard library -- which is the whole trick: a package with no imports is one
// both halves may have.
//
// # What belongs here
//
// A fact TWO modules of one suite must agree about, and nothing else. A rig's
// geometry belongs to the observer; a prototype's other fields belong to the
// data stage. What crosses the line is the NAME the one defines and the other
// places, and the one number that has to match base's own belt.
package protos

// BaseLoader is base Factorio's only 1x1 loader, and it is both the prototype
// TYPE and the prototype NAME. Every data stage in the estate clones it.
//
// It is a hidden 0.03125 prototype, which is a third of a yellow belt: a source
// that slow could not saturate anything, which is why every suite clones it at
// ExpressSpeed instead of using it.
const BaseLoader = "loader-1x1"

// ExpressSpeed is `express-transport-belt`'s speed.
//
// WRITTEN OUT RATHER THAN READ, and that is a decision each data stage's header
// already carries: it is base's own value, and a loader that silently followed a
// modded belt would change what every yardstick in the estate means.
const ExpressSpeed = 0.09375

// The loader each suite's data stage defines and its observer places. One
// constant per suite rather than one shared name, because the names are what the
// goldens have and three of them happen to coincide only by history.
//
// IactLoader is the odd one out and is not a suite's at all: it belongs to the
// rig-staging mod a HUMAN enables to walk test/interactive/README.md. It is the
// same prototype for the same reason -- half of those gestures are about where a
// FULL machine's items go when it is taken apart, and an empty balancer would
// prove nothing -- and it keeps the name the hand-written data stage gave it.
const (
	SedgeLoader = "bbbs-loader"
	MarLoader   = "bbbt-loader"
	QualLoader  = "bbbqual-loader"
	MixLoader   = "bbbt-loader"
	PlatLoader  = "bbbt-loader"
	MigLoader   = "bbbmig-loader"
	M2Loader    = "bbbt-loader"
	M3Loader    = "bbbt-loader"
	EdgeLoader  = "bbbt-loader"
	IactLoader  = "bbbi-loader"
)

// M2LaneSplitter is the `m2` suite's `lsio` rig, and it is the one prototype in
// the estate that is a clone of THIS MOD'S OWN rather than of base's.
//
// Base ships the `lane-splitter` TYPE and not one buildable entity of it -- the
// type exists for Space Age's turbo lane splitter, and there is nothing in
// data.raw to place. Cloning the mod's own hidden one rather than writing a
// lane splitter from scratch is the point: it is a real prototype a real
// Factorio validated, so what `lsio` exercises is the engine's type and
// classifySide's reading of it, not a prototype the harness invented and might
// have got wrong. The observer depends on `better-belt-balancer`, so its data
// stage has already run by the time this clone is made.
const (
	M2LaneSplitter   = "bbbt-lane-splitter"
	BBBLaneSplitter  = "bbb-lane-splitter"
	LaneSplitterType = "lane-splitter"
)

// PlatStackLoader is the `plat` suite's second loader, and the only prototype in
// the estate that differs from the others in more than its name.
//
// `max_belt_stack_size` is refused at load without the `space_travel` feature
// flag -- "Belt stacking is disabled and can not be used" -- which is exactly
// why the stacking leg lives in the Space Age suite and not in a base-only one.
// The prototype is only half of what stacking needs: a force whose
// `belt_stack_size_bonus` is 0 receives SINGLES from this loader, and the
// observer sets the other half.
const PlatStackLoader = "bbbt-stackloader"

// PlatStackSize is that loader's `max_belt_stack_size`.
const PlatStackSize = 4

// ---------------------------------------------------------------------------
// the bench harness's configuration channel
// ---------------------------------------------------------------------------
//
// THE ONE PLACE IN THE ESTATE WHERE THIS PACKAGE CARRIES SOMETHING THAT IS NOT
// A PROTOTYPE NAME, and it is here for the reason the package exists: two wasm
// modules of one mod have to agree about a name and neither can import the
// other's half.
//
// `bench/mods/bbb-bench-setup` was configured by a `config.lua` that
// `bench/run.sh` REWROTE per matrix cell -- eight keys in a table the mod
// `require`d. A Go guest cannot require a Lua file, so the channel is
// STARTUP SETTINGS: `obs/benchdata` defines one per key, `bench/run.sh` writes
// them into a `mod-settings.dat` it composes per cell (tools/mod-settings.py),
// and `obs/bench` reads them out of `settings.startup` at `fk_on_init`.
//
// STARTUP RATHER THAN RUNTIME-GLOBAL, and the reason is the two-phase shape of
// a cell rather than a preference. A runtime-global is seeded from
// mod-settings.dat into the SAVE at creation and read from the save thereafter,
// which would work; a startup setting is read from mod-settings.dat by both
// processes directly, so `--create` and `--benchmark` are configured by one
// file with no state in between. Nothing here shapes a prototype, so the data
// stage never reads one.
const (
	BenchScenario = "bbb-bench-scenario"
	BenchN        = "bbb-bench-n"
	BenchK        = "bbb-bench-k"
	BenchTier     = "bbb-bench-tier"
	BenchItem     = "bbb-bench-item"
	BenchPartName = "bbb-bench-part-name"
	BenchMeter    = "bbb-bench-meter"
	BenchHitch    = "bbb-bench-hitch"
)

// BenchScenarios is every value BenchScenario may take, and the FIRST of them is
// the default -- which is `stringSetting`'s rule in the shipped guest's own
// settings stage, and it is what makes "the default is a value the engine will
// accept" true by construction rather than by a second list.
//
// The four `mega` scenarios build a HETEROGENEOUS population where the four
// above them build `n` copies of one shape; `bench/README.md` is the table.
var BenchScenarios = []string{
	"saturated", "idle", "control", "control-idle",
	"mega", "mega-idle", "mega-control", "mega-control-idle",
}

// BenchTiers is every value BenchTier may take. Same rule: the head is the
// default, and `express` is the head because it is the tier that matters for a
// late-game base and the worst case for the incumbent's polling divisor.
var BenchTiers = []string{"express", "fast", "normal"}
