// Command bbb-qual-test-data is the `bbb-qual-test` observer's DATA STAGE: a
// second wasm module packaged beside it by `fklua mod --data-module`, exactly as
// the shipped mod's own data guest is (guest/go/data).
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
// refuses one that does otherwise.
//
// THE NAME IS WRITTEN DOWN HERE AND IN obs/qual, and the duplication is forced:
// the two modules cannot share a package, because this one may not import fkapi
// and that one may not import fkdata.
package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// express is `express-transport-belt`'s speed. Written out rather than read:
// base's own value, and a loader that silently followed a modded belt would
// change what the yardstick means.
const express = 0.09375

//go:wasmexport fk_data
//go:noinline
func onData() {
	fkdata.Clone("loader-1x1", "loader-1x1", "bbbqual-loader")
	fkdata.Set(fkdata.Num(express), "loader-1x1", "bbbqual-loader", "speed")
	// Nil DELETES the key rather than writing false: these three have to be
	// ABSENT, not present and empty.
	fkdata.Set(fkdata.Nil(), "loader-1x1", "bbbqual-loader", "minable")
	fkdata.Set(fkdata.Nil(), "loader-1x1", "bbbqual-loader", "next_upgrade")
	fkdata.Set(fkdata.Nil(), "loader-1x1", "bbbqual-loader", "fast_replaceable_group")
}

func main() {}
