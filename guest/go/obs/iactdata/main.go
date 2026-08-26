// Command bbb-interactive-setup-data is the `bbb-interactive-setup` staging
// mod's DATA STAGE: a second wasm module packaged beside it by
// `fklua mod --data-module`, exactly as the shipped mod's own data guest is
// (guest/go/data).
//
// One prototype, and it is the same one every suite in the estate defines.
//
// Base Factorio's only 1x1 loader is `loader-1x1`, a hidden 0.03125 prototype --
// a third of a yellow belt -- and a source that slow could not fill anything. A
// player walking the checklist has to see the gesture rigs SATURATED, because
// what half of those gestures are about is where a full machine's items go when
// it is taken apart; an empty balancer would pocket nothing and prove nothing.
//
// Nothing here ships. This mod is enabled for ten minutes and disabled again.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.IactLoader) }

func main() {}
