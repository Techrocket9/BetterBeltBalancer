//go:build factorio20

// The Factorio 2.0 half of the shim. Read starterpack.go's header first: it
// carries the whole reason this pair exists, and this file is the two-line
// other arm of it.
package main

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// applyStarterPack is `sp.ApplyStarterPack()` on whichever engine this is.
func applyStarterPack(sp fkapi.LuaSpacePlatform) error {
	_, err := sp.ApplyStarterPack()
	return err
}
