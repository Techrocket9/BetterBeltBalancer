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
const (
	SedgeLoader = "bbbs-loader"
	MarLoader   = "bbbt-loader"
	QualLoader  = "bbbqual-loader"
	MixLoader   = "bbbt-loader"
	PlatLoader  = "bbbt-loader"
	MigLoader   = "bbbmig-loader"
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
