// Command bbb-mig-test-data is the `bbb-mig-test` observer's DATA STAGE: a
// second wasm module packaged beside it by `fklua mod --data-module`, exactly as
// the shipped mod's own data guest is (guest/go/data).
//
// It is the harness's own kit, and there is one thing in it -- the same one
// thing every other suite's kit has, for the same reason. See obs/obsdata.
//
// NOTHING HERE TOUCHES `balancer-part` OR ANY PROTOTYPE OF THIS MOD'S. This file
// has to load in the phase where Better Belt Balancer is not installed at all,
// which is the whole shape of the added-as-removed leg.
//
// A data guest has NO RUNTIME API -- there is no `game`, no `script` and no
// entities at this stage -- so it imports fkdata and never fkapi, and packaging
// refuses one that does otherwise.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.MigLoader) }

func main() {}
