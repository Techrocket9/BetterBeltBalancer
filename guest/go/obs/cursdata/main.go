// Command bbb-curs-test-data is the `bbb-curs-test` observer's DATA STAGE: a
// second wasm module packaged beside it by `fklua mod --data-module`, exactly as
// every other suite's is.
//
// One prototype, and it is the estate's own: a 1x1 loader fast enough to
// saturate an express belt, which base has no buildable form of. See obs/obsdata
// for the definition and obs/protos for where the NAME lives so that this module
// and obs/curs can both have it without sharing an import.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.CursLoader) }

func main() {}
