// Command bbbdata is BetterBeltBalancer's SETTINGS and DATA stages: the mod's
// declarative half, compiled from Go instead of hand-written in Lua.
//
// It is a SECOND WASM MODULE beside the control guest, packaged by `fklua mod
// --data-module` (this mod declares it in fklua.toml). The two are separate
// modules for a measured reason and three architectural ones, all of them
// FkLua's agents/datastage.md D1: sharing the control module would parse and
// instantiate a 3 MB program at every data-family stage that hooks it -- +150 ms
// per game load, per stage, for a program the stage never calls -- and would run
// the control guest's package initialisers against a runtime API that does not
// exist at these stages.
//
// They are two `main` packages in ONE Go module, which is what lets both import
// guest/go/engine and guest/go/skin. That is not a convenience: it is what
// RETIRED the problem mod-data/engine.lua existed to solve. Factorio's settings
// stage is its own Lua state and nothing carries across a stage boundary, so
// when this was Lua the version branch could only be shared by being a required
// file, and two copies of it would have been one edit away from a guest that
// believes it may stack over a prototype that cannot. One compiled module with
// two exports has no second copy to drift.
//
// # The hooks, and why there are three of them
//
//	fk_settings          -> settings.lua           one runtime-global setting
//	fk_data              -> data.lua               everything a player sees
//	fk_data_final_fixes  -> data-final-fixes.lua   the legacy stub, and only it
//
// `fklua mod` writes a stage file for each hook that is exported and for no
// others, so there is no empty `data-updates.lua` in the package -- and nothing
// here wants one. The split between the second and the third is load-bearing
// and is the migration's: `data-final-fixes` is the only pass that runs after
// every OTHER mod's data and data-updates stages, so it is the only place that
// can see whether an incumbent Belt Balancer already defined `balancer-part`.
// See legacy.go.
//
// # No state survives a stage
//
// This module is instantiated FRESH for each stage it hooks -- Factorio's
// settings stage is a separate Lua state, and `require` re-executes a file at
// every stage anyway -- so a package-level variable set in fk_data is back at
// its zero value in fk_data_final_fixes. Nothing here relies on one; the place
// to keep something between stages is data.raw, which is what legacy.go's
// "has anybody defined this" probe reads.
//
// # Determinism
//
// A data stage runs on every client and a divergent prototype set is a JOIN
// REFUSAL, so nothing here may branch on an iteration order. It does not have
// to try: fkdata sorts every dictionary in both directions and Keys is sorted,
// and there is no map ranged over in this package at all. The one thing that IS
// a guest-side hazard is floating point, and sprite.go is where it lands.
//
// # Build
//
//	tinygo build -target=wasm-unknown -scheduler=none -gc=leaking -opt=2 \
//	    -o dist/bbbdata.wasm ./data
//
// -gc=leaking and --persist=none whatever the control guest uses, because this
// module runs once and dies with the Lua state that built it. The Makefile's
// GC= stamp does not reach it.
//
// # Verification
//
// `test/check-datastage.py` is the gate: it runs Factorio's own `--dump-data`,
// which executes exactly these three hooks and stops before control.lua, and
// hashes the normalised result against a golden captured from the Lua this
// replaced. Two mod sets, because legacy.go has two arms.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/engine"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// ---------------------------------------------------------------------------
// //go:noinline IS LOAD-BEARING ON EVERY SECTION FUNCTION BELOW, AND IT IS NOT
// ABOUT SPEED.
//
// LUA 5.2 CANNOT COMPILE AN ARBITRARILY LONG FUNCTION. A jump's offset lives in
// an 18-bit signed field of one instruction, so a control structure whose body
// out-runs that range is refused by the PARSER -- `control structure too long`
// -- and the mod does not load at all.
//
// This whole stage is straight-line table building through small helpers, which
// is the shape LLVM inlines most eagerly: at -opt=2 it folded all six data-stage
// sections and every builder under them into ONE wasm function, which the
// emitter lowered to ONE Lua function of 28,139 lines. Measured, on the first
// packaged build:
//
//	Failed to load mod "better-belt-balancer":
//	  fk_data_module.lua:32937: control structure too long near 'trap_unreachable'
//
// Marking the sections noinline puts the source's own boundaries back, so one Go
// function is one wasm function is one Lua function, each far inside the limit.
// It costs one call per section on a path that runs ONCE PER GAME LOAD, which is
// not a budget anything here is spending.
//
// It is a property of the emitted Lua rather than of this mod, so anything added
// to this package wants the same treatment -- and a builder that grows into a
// few hundred prototypes wants it too. The symptom is unmistakable and the
// message names neither the cause nor the file, so it is written down here.
// ---------------------------------------------------------------------------

//go:noinline
//go:wasmexport fk_settings
func onSettings() { settings() }

//go:wasmexport fk_data
//go:noinline
func onData() {
	// The order the six required files ran in, kept. It does not change the
	// game -- `data:extend` order is insertion order in the dump and the gate
	// normalises it away, and Factorio's prototype checksum is order-insensitive
	// outright -- but it keeps this file readable beside the mod-data/data.lua
	// it replaces, and it keeps hidden.go's clones before the sprite table that
	// nothing depends on either way.
	entity()
	hidden()
	sprites()
	item()
	recipe()
	technology()
}

//go:wasmexport fk_data_final_fixes
//go:noinline
func onFinalFixes() { legacy() }

// canStack is WHICH ENGINE THIS IS: can this Factorio put two belt-connectables
// on one tile, which decides `not_colliding_with_itself`, the `bbb-can-stack`
// marker and whether the multi-edge setting exists at all.
//
// BOTH STAGES ASK IT THROUGH THIS ONE FUNCTION, which is the whole of what
// mod-data/engine.lua used to buy at the cost of being a separate file two
// separate Lua states each parsed. The argument, the fails-safe-towards-2.1
// reasoning and the tests are in guest/go/engine.
func canStack() bool { return engine.Is2_0(fkdata.ModVersion("base")) }
