// Command bbb-m2-test-data is the `bbb-m2-test` observer's DATA STAGE: a second
// wasm module packaged beside it by `fklua mod --data-module`, exactly as the
// shipped mod's own data guest is (guest/go/data).
//
// TWO PROTOTYPES, and both exist for the same reason: BASE FACTORIO HAS NO
// BUILDABLE VERSION of the thing the rig needs to put against a balancer part.
//
//   - the express loader, which every suite in the estate clones. See
//     obs/obsdata, which is where that one thing lives now that seven suites
//     need it, and obs/protos, which is where the NAME lives so that this module
//     and obs/m2 can both have it without sharing an import.
//   - a placeable LANE SPLITTER, which is this suite's alone. Base ships the
//     `lane-splitter` TYPE and not one entity of it.
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
	obsdata.ExpressLoader(protos.M2Loader)

	// THE MOD'S OWN HIDDEN LANE SPLITTER, MADE PLACEABLE.
	//
	// Its inherited speed (0.25, the hidden network's ceiling) is left alone on
	// purpose: the rig is about CLASSIFICATION, and an edge slower than express
	// would cap it and turn a throughput assertion into a measurement of this
	// prototype.
	//
	// The three Nils DELETE their keys rather than writing false, which is what
	// stripping a cloned prototype needs: they have to be ABSENT, not present
	// and empty.
	fkdata.Clone(protos.LaneSplitterType, protos.BBBLaneSplitter, protos.M2LaneSplitter)
	fkdata.Set(fkdata.Nil(), protos.LaneSplitterType, protos.M2LaneSplitter, "minable")
	fkdata.Set(fkdata.Nil(), protos.LaneSplitterType, protos.M2LaneSplitter, "next_upgrade")
	fkdata.Set(fkdata.Nil(), protos.LaneSplitterType, protos.M2LaneSplitter, "fast_replaceable_group")
}

func main() {}
