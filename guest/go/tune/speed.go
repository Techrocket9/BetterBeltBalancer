package tune

// THE HIDDEN NETWORK'S BELT SPEED, DERIVED FROM THE GAME RATHER THAN WRITTEN
// DOWN.
//
// The four prototypes the compiler places (`bbb-linked-belt`, `bbb-belt`,
// `bbb-splitter`, `bbb-lane-splitter`) shipped a constant 0.25 tiles/tick from
// M2 until 0.3.1. That number is 2.67x express and 2x turbo, so nothing in base
// or Space Age can out-run it -- and a mod that adds a faster belt was
// SILENTLY THROTTLED to 120 items/s per port, which is what the portal's
// "Turbo-belts" request was actually about.
//
// The measurement the derivation rests on, from this program's own probe: a
// belt faster than 0.25 runs FULLY COMPRESSED and delivers 480 x speed items/s.
// There is no engine ceiling at 0.25 -- an earlier note in this repo claimed one
// and it was wrong, and it is purged rather than corrected. 0.25 is a FLOOR
// worth keeping (it is what every recorded number in CLAUDE.md was measured on,
// and it is comfortably above every vanilla tier), and the ceiling is whatever
// the fastest belt in the loaded game is.
//
// # No upper cap, and the argument is not "we tested it"
//
// The derived speed is a value THE ENGINE HAS ALREADY ACCEPTED, on the very
// prototype it was read from. A belt-connectable prototype's `speed` is
// validated when that mod's own prototype is loaded; mirroring it onto four
// more prototypes of the same families cannot reach a value the loader would
// refuse, because the loader already did not refuse it. A cap would be this mod
// second-guessing a number Factorio validated -- and it would re-introduce the
// silent throttle for exactly the players who installed the fast belt.
//
// # The ordering caveat, stated rather than hidden
//
// The scan runs at `data-final-fixes`, which is the LAST data stage there is,
// so it sees every belt every mod defined at data, data-updates and
// final-fixes-before-us. What it cannot see is a mod whose own
// `data-final-fixes` runs AFTER this one and raises a belt then. Factorio orders
// final-fixes by dependency and then by name, so that is possible and there is
// no later stage to move to. The cost is the old behaviour for that one player:
// a hidden network at the speed of the second-fastest belt.
//
// # Our own prototypes are in the scan, deliberately
//
// The four bbb- clones are `transport-belt`, `splitter`, `lane-splitter` and
// `linked-belt` prototypes, so [BeltFamilies] enumerates them too -- at 0.25,
// the value hidden.go's clone set an hour earlier in the same load. That is
// harmless and it is not a ratchet: `data.raw` is built fresh on every load, so
// the clone is 0.25 again next time and the maximum is a function of the
// installed mods alone. Excluding them would be a name filter that has to stay
// in step with hidden.go, to change an answer the floor already covers.

// BeltFamilies is every prototype type whose members carry a belt `speed`, in
// the order the scan walks them.
//
// All seven descend from `TransportBeltConnectablePrototype`, where `speed` is
// mandatory -- so a member of one of these types has a speed by construction
// and a type outside them does not have one to read. `linked-belt` is in the
// list because this mod's own edge interface is one, and because nothing says a
// belt mod may not add another.
//
// The order does not affect the answer (a maximum is commutative) and is fixed
// anyway, because a data stage must not branch on an iteration order and the
// cheapest way not to is never to have one.
func BeltFamilies() []string {
	return []string{
		"transport-belt",
		"underground-belt",
		"splitter",
		"lane-splitter",
		"loader",
		"loader-1x1",
		"linked-belt",
	}
}

// SpeedFloor is the speed the hidden network has always run at, and the value
// the derivation can never go below.
//
// 0.25 tiles/tick is one item width per tick, which is the fastest a belt can
// run fully compressed at vanilla item pitch: 60 items/s/lane, 120/s/belt.
// P >= N means no hidden line ever carries more than one visible belt's rate,
// so this is enough by construction for every vanilla tier and the floor exists
// to keep it enough when a pack REMOVES the fast tiers rather than adding one.
const SpeedFloor = 0.25

// HiddenSpeed is the speed to give the four hidden prototypes: the fastest belt
// in the game, but never slower than [SpeedFloor].
//
// THERE IS NO UPPER BOUND AND THERE MUST NOT BE ONE. Every candidate is a value
// the engine already accepted, on the very prototype it was read from; a cap
// here would refuse a number Factorio did not, and it would refuse it for
// exactly the player who installed the fast belt this feature exists for.
//
// A NaN is ignored, and that is a property of `>` rather than a guard: NaN
// compares false against everything, so it can never become the maximum. It is
// worth naming because the alternative reading -- that a single unreadable
// prototype poisons the answer for every other belt in the game -- is what a
// `math.Max` fold would have done.
func HiddenSpeed(speeds []float64) float64 {
	best := SpeedFloor
	for _, s := range speeds {
		if s > best {
			best = s
		}
	}
	return best
}
