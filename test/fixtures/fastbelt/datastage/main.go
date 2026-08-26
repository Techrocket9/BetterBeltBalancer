// Command fastbelt-data is the fast-belt fixture's DATA STAGE: two belt
// prototypes faster than BetterBeltBalancer's hidden network, so that
// `deriveHiddenSpeed` has something to derive.
//
// ---------------------------------------------------------------------------
// WHY THE FIXTURE EXISTS
//
// The speed derivation's whole point is a game with a belt faster than 0.25
// tiles/tick in it, and no mod set this repository can install has one: vanilla
// tops out at turbo, 0.125, which is HALF the floor. So on every mod set the
// gate could otherwise reach, the correct behaviour and a derivation that did
// nothing at all are the same dump -- which is a gate that cannot fail on the
// feature it is gating.
//
// `guest/go/tune`'s unit tests prove the arithmetic and the regenerated default
// goldens prove the no-change case. This is the third leg: the engine's own
// prototype table, after a real mod really did add a faster belt.
//
// ---------------------------------------------------------------------------
// TWO FAMILIES, AND THE FASTER ONE IS NOT THE OBVIOUS ONE
//
// The belt is 0.4 and the UNDERGROUND is 0.5, so the answer the gate asserts
// (0.5) can only come from a scan that walks more than `transport-belt`. A
// derivation that read the obvious family alone would produce 0.4, which is
// still above the floor, still a change, and still wrong -- and a fixture with
// one fast belt in it could not tell the two apart.
//
// ---------------------------------------------------------------------------
// CLONED RATHER THAN WRITTEN OUT
//
// A transport belt is about five hundred scalar leaves of animation sets,
// sounds and icons, every one of them mandatory somewhere. `fkdata.Clone` is
// the engine's own deep copy, so the fixture is a real belt prototype that a
// real Factorio validated, and the two lines below are the only difference
// between it and base's own.
package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// The two speeds, in tiles per tick. Both are above BetterBeltBalancer's 0.25
// floor and the underground is the faster, which is what makes the assertion
// in test/check-datastage.py a statement about the SCAN rather than about one
// family of it.
const (
	beltSpeed        = 0.4
	undergroundSpeed = 0.5
)

//go:wasmexport fk_data
//go:noinline
func onData() {
	fkdata.Clone("transport-belt", "transport-belt", "bbbt-fast-belt")
	fkdata.Set(fkdata.Num(beltSpeed), "transport-belt", "bbbt-fast-belt", "speed")
	// The upgrade chain and the underground it points at both name base's own
	// prototypes, which would make this belt a member of somebody else's
	// upgrade planner. Dropped: the fixture is a speed, not a belt anybody
	// plays with.
	fkdata.Set(fkdata.Nil(), "transport-belt", "bbbt-fast-belt", "next_upgrade")
	fkdata.Set(fkdata.Nil(), "transport-belt", "bbbt-fast-belt", "related_underground_belt")

	fkdata.Clone("underground-belt", "underground-belt", "bbbt-fast-underground")
	fkdata.Set(fkdata.Num(undergroundSpeed), "underground-belt", "bbbt-fast-underground", "speed")
	fkdata.Set(fkdata.Nil(), "underground-belt", "bbbt-fast-underground", "next_upgrade")
	fkdata.Set(fkdata.Nil(), "underground-belt", "bbbt-fast-underground", "related_transport_belt")

	fkdata.Log("[BBBT] fast-belt fixture: a belt and an underground above the floor")
}
