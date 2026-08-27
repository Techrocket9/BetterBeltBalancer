//go:build !factorio20

// THE ONE CALL IN THE ESTATE WHOSE GENERATED SIGNATURE DIFFERS BETWEEN THE TWO
// ENGINE ARMS THIS MOD SHIPS ON, and therefore the one place an observer needs a
// build tag.
//
// `LuaSpacePlatform::apply_starter_pack` gained an OPTIONAL `silent` parameter
// between 2.0.77 and 2.1.x. `fklua api check` against the 2.0.77 arm has
// reported it against this observer since phase 3 of the estate port, and the
// answer recorded there is about the WIRE: the parameter is optional, this
// observer passes it absent, and `fk_abi.lua`'s `M.call` trims the argument list
// to the last argument actually present -- so the call reaches either engine as
// `apply_starter_pack()`. That is still true and it is not the whole story.
//
// The GO SIGNATURE is generated per pin, and an absent optional is still a
// parameter: every 2.1.x pin emits `ApplyStarterPack(silent *bool)` and 2.0.77 emits
// `ApplyStarterPack()`. One source cannot call both, so `sp.ApplyStarterPack(nil)`
// is a compile error on the release/2.0 recut -- found by `make check` the first
// time that recut was cut with a COMPILED estate under it, the 0.2.0 arm having
// predated the port entirely.
//
// The tag is derived rather than typed: the Makefile passes `-tags factorio20`
// when fklua.toml's `factorio_version` says 2.0, so the arm follows the pin that
// generated the bindings and cannot be set independently of it. Trunk takes this
// file; the recut takes its sibling. Nothing hand-written differs between the two
// branches -- both files are on both -- which is the property the recut rests on.
//
// One call site, so one shim and no harness entry: one consumer is no
// duplication to remove.
package main

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// applyStarterPack is `sp.ApplyStarterPack()` on whichever engine this is.
func applyStarterPack(sp fkapi.LuaSpacePlatform) error {
	_, err := sp.ApplyStarterPack(nil)
	return err
}
