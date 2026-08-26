// Command bbb-sedge-test-data is the `bbb-sedge-test` observer's DATA STAGE: a
// second wasm module packaged beside it by `fklua mod --data-module`, exactly as
// the shipped mod's own data guest is (guest/go/data).
//
// It is the harness's own kit, and there is one thing in it -- the same one
// thing every other suite's kit has, for the same reason.
//
// Base Factorio's only 1x1 loader is `loader-1x1`, a hidden 0.03125 prototype,
// and a source that slow could not saturate anything. Every other piece in these
// rigs is a stock express belt or chest, so what the suite measures is the rate
// a real one delivers.
//
// # Why it is a module and not four lines of Lua
//
// It was four lines of Lua, and that is the whole point of the estate port: the
// repository holds no hand-written Lua on either side of the boundary, so a
// test mod with a data stage gets a compiled data stage. It costs one more wasm
// build and one more `--data-module` flag.
//
// A data guest has NO RUNTIME API -- there is no `game`, no `script` and no
// entities at this stage -- so it imports fkdata and never fkapi, and packaging
// refuses one that does otherwise. It is built `-gc=leaking` and packaged
// `--persist=none` whatever the control guest uses, because it runs once and
// dies with the Lua state that built it.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/obsdata"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

//go:wasmexport fk_data
//go:noinline
func onData() { obsdata.ExpressLoader(protos.SedgeLoader) }

func main() {}
