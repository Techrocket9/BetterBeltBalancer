// Command bbb-marathon-test-data is the `bbb-marathon-test` observer's DATA
// STAGE: a second wasm module packaged beside it by `fklua mod --data-module`,
// exactly as the shipped mod's own data guest is (guest/go/data).
//
// It is the harness's own kit, and there is one thing in it -- the same one
// thing every other suite's kit has, for the same reason.
//
// Base Factorio's only 1x1 loader is `loader-1x1`, a hidden 0.03125 prototype,
// and a source that slow could not saturate anything. Every other piece in these
// rigs is a stock express belt, splitter or chest, so what the suite measures is
// the rate a real one delivers.
//
// A data guest has NO RUNTIME API -- there is no `game`, no `script` and no
// entities at this stage -- so it imports fkdata and never fkapi, and packaging
// refuses one that does otherwise. It is built `-gc=leaking` and packaged
// `--persist=none` whatever the control guest uses, because it runs once and
// dies with the Lua state that built it.
//
// THE NAME AND THE SPEED LIVE IN obs/protos, which imports nothing at all --
// which is what lets both this module and obs/mar have them. The five calls that
// clone the loader live in obs/obsdata, which every data stage in the estate
// shares. See agents/estate-port.md, phase 3.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.MarLoader) }

func main() {}
