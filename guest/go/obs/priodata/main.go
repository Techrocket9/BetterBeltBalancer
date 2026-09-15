// Command bbb-prio-test-data is the priority suite's data stage. One express
// loader, which is what every rig in the estate feeds and drains through: base's
// only 1x1 loader runs at a third of a yellow belt.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.PrioLoader) }

var _ = fkdata.Nil

func main() {}
