// Package obsdata is the one thing every observer DATA STAGE in the estate
// does, written down once.
//
// Six suites bring a data stage and all six of them define the same prototype:
// a 1x1 loader running at express speed. Base Factorio's only 1x1 loader is
// `loader-1x1`, a hidden 0.03125 prototype -- a third of a yellow belt -- and a
// source that slow could not saturate anything, while every other piece in every
// rig is a stock express belt, splitter or chest. So each suite clones it and
// raises the speed, and until phase 3 each suite wrote the same five calls out.
//
// It imports fkdata and never fkapi, because a data guest has no runtime API at
// all -- there is no `game`, no `script` and no entities at this stage -- and
// packaging refuses a data module holding fkapi.
//
// The NAMES live in the sibling `protos` package, which imports nothing, because
// an observer has to place what its data stage defined and an observer may not
// import fkdata.
package obsdata

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// ExpressLoader clones base's 1x1 loader under `name` and makes it express.
//
// `Clone` is the engine's own deep copy, made on the guest's instruction, and it
// registers the copy under the new name immediately -- so what follows patches
// the copy rather than building one. Under a clone the untouched leaves are
// literally the bytes base shipped, which is the fidelity a Get-then-Extend
// could not promise.
//
// The three Nils DELETE their keys rather than writing false, which is what
// stripping a cloned prototype needs: they have to be ABSENT, not present and
// empty. Nothing places one of these by hand, nothing mines it, and nothing
// upgrades into or out of it.
func ExpressLoader(name string) {
	fkdata.Clone(protos.BaseLoader, protos.BaseLoader, name)
	fkdata.Set(fkdata.Num(protos.ExpressSpeed), protos.BaseLoader, name, "speed")
	fkdata.Set(fkdata.Nil(), protos.BaseLoader, name, "minable")
	fkdata.Set(fkdata.Nil(), protos.BaseLoader, name, "next_upgrade")
	fkdata.Set(fkdata.Nil(), protos.BaseLoader, name, "fast_replaceable_group")
}
