// Command bbb-plat-test-data is the `bbb-plat-test` observer's DATA STAGE: a
// second wasm module packaged beside it by `fklua mod --data-module`, exactly as
// the shipped mod's own data guest is (guest/go/data).
//
// It is the only data stage in the estate with TWO prototypes in it, and the
// second is the whole of what makes the belt-stacking leg possible.
//
// A data guest has NO RUNTIME API -- there is no `game`, no `script` and no
// entities at this stage -- so it imports fkdata and never fkapi, and packaging
// refuses one that does otherwise.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

//go:wasmexport fk_data
//go:noinline
func onData() {
	obsdata.ExpressLoader(protos.PlatLoader)

	// THE STACKING SOURCE, and it is a clone of the express loader rather than of
	// base's: the Lua deep-copied its own already-patched `p`, so the stacking
	// loader inherits the express speed and the three stripped keys and adds one
	// field. Cloning base again and re-patching would be the same prototype by a
	// longer road and one edit away from the two drifting apart.
	fkdata.Clone(protos.BaseLoader, protos.PlatLoader, protos.PlatStackLoader)

	// `max_belt_stack_size` is REFUSED AT LOAD without the `space_travel` feature
	// flag -- "Belt stacking is disabled and can not be used" -- which is exactly
	// why the stacking leg lives in this suite and not in a base-only one. The
	// flag's key is `space_travel`; the engine's error message spells it with a
	// hyphen and the table does not.
	//
	// The prototype is only half of it: a force whose `belt_stack_size_bonus` is 0
	// receives SINGLES from this loader. Both are needed, and the observer sets
	// the other half.
	if fkdata.FeatureFlag("space_travel") {
		fkdata.Set(fkdata.Num(protos.PlatStackSize),
			protos.BaseLoader, protos.PlatStackLoader, "max_belt_stack_size")
	}
}

func main() {}
