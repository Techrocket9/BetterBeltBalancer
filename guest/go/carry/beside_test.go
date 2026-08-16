package carry

import "testing"

// A removal a player caused is not only a PART they mined, and that is the
// whole of this file.
//
// The field report: place a downward-facing output belt on a running balancer,
// then mine it again. The network the recompile builds is SMALLER than the one
// it drained -- an output crossing back over a power-of-two port boundary halves
// the butterfly -- so the reinsertion overflows and the overflow reaches the
// floor. Nothing was lost and nothing was in the wrong place; there was simply
// nobody recorded to offer it to, because the claim was written down only when
// the removal took a PART away.
//
// The policy never said that. "A removal a PLAYER caused offers what no network
// could take to that player before the floor" is indifferent to which entity
// they removed: mining the belt off the side of a machine shrinks the machine
// exactly as mining a part of it does, and vanilla would hand the difference
// over either way.
//
// WHAT IS DIFFERENT ABOUT THE BELT IS WHERE IT STOOD, and that is the one
// decision this file pins. A mined PART's tile is a tile of the network by
// construction, so `Tile(surf, force, mined)` answers to its own pool. A mined
// BELT's tile is one tile OUTSIDE the box by construction -- a belt adjacent to
// a cluster is adjacent to it -- so the same call, made with the tile the event
// handed over, records a claim no pool can ever answer. The tile that has to be
// written down is the tile of the PART the belt was touching, which the
// neighbour gate is already holding: it is the registry key it just looked up.
//
// The gate is the guest's hottest path and makes no host call, and none of this
// changes that: the tile, the force and the player are all already in hand.

// beside is the field report's shape. A two-part balancer at x=0, rows 4 and 5,
// with the mined belt one tile south of the bottom part.
func beside() (net Region, part, belt Region) {
	net = Region{Surf: 1, Force: 1, X0: 0, Y0: 4, X1: 0, Y1: 5}
	return net, Tile(1, 1, 0, 5), Tile(1, 1, 0, 6)
}

// THE RULE THIS FILE EXISTS FOR, and it is a GREEN test rather than a red one:
// the store has always been able to answer this question and the shipped guest
// never asked it. That is the honest shape of the defect, and it is why the two
// tests that DO fail against the shipped code are further down -- what a pure
// package can prove about the miner's pocket is the identity, never the
// trigger. The quantity flowing through here is pinned by the `edge` suite's
// `bmin` leg instead; see test/mods/bbb-edge-test/control.lua.
func TestAMineBesideANetworkIsTheNetworksClaim(t *testing.T) {
	net, part, _ := beside()
	var c Claims
	c.Add(part, 7)
	if got := c.BeneficiaryFor(net); got != 7 {
		t.Fatalf("the network credited player %d, not 7: mining the belt off "+
			"the side of a machine shrinks the machine, and the overflow is "+
			"the miner's", got)
	}
}

// ...AND THE OBVIOUS IMPLEMENTATION OF IT IS THE ONE THAT DOES NOTHING. The
// part path records the tile the EVENT reported, because for a part that tile
// is the network's. Reusing that call on the neighbour path writes down the
// belt's tile, which is one outside the box by construction and answers to
// nobody -- a fix that compiles, runs, costs a claim per mine and changes not a
// single item's destination.
func TestTheMinedBeltsOwnTileIsNobodysClaim(t *testing.T) {
	net, _, belt := beside()
	var c Claims
	c.Add(belt, 7)
	if got := c.BeneficiaryFor(net); got != 0 {
		t.Fatalf("a claim written down where the BELT stood was answered by "+
			"player %d: the box is the cluster's parts and the belt is beside "+
			"them, so this can only pass by accident", got)
	}
}

// The tile is the key rather than the root, and this is why. A tick can contain
// several removals: a part mined elsewhere in the same cluster re-roots the
// survivors at the smallest surviving node id, and the flush re-resolves every
// queued root through `find` before it compiles. A root written down at event
// time is a number that may no longer be one; the GROUND does not move.
func TestTheClaimSurvivesTheClusterBeingReRooted(t *testing.T) {
	net, part, _ := beside()
	var c Claims
	c.Add(part, 7)
	// Whatever the flush re-resolves the root to, the pool is opened against the
	// bounding box the network was BUILT with -- which still contains the tile.
	if got := c.BeneficiaryFor(net); got != 7 {
		t.Fatalf("the claim stopped answering: player %d", got)
	}
}

// A belt between two balancers is mined once and shrinks BOTH, so both pools
// must credit the miner. The gate walks a 5x5 neighbourhood and finds a part of
// each; each part's own tile is what goes on the list.
func TestOneBeltBetweenTwoClustersClaimsBoth(t *testing.T) {
	left := Region{Surf: 1, Force: 1, X0: 0, Y0: 0, X1: 1, Y1: 1}
	right := Region{Surf: 1, Force: 1, X0: 3, Y0: 0, X1: 4, Y1: 1}
	var c Claims
	c.Add(Tile(1, 1, 1, 0), 7) // the part on the left of the mined belt
	c.Add(Tile(1, 1, 3, 0), 7) // ... and the one on its right
	if got := c.BeneficiaryFor(left); got != 7 {
		t.Fatalf("the left network credited player %d", got)
	}
	if got := c.BeneficiaryFor(right); got != 7 {
		t.Fatalf("the right network credited player %d", got)
	}
}

// And a belt mined between two FORCES' balancers credits its miner in exactly
// one of them. Two forces' parts touching are two balancers whose boxes are
// adjacent by construction, so this is not a hypothetical arrangement -- it is
// what the gate finds every time anyone builds against a neighbour.
func TestABeltMinedBetweenTwoForcesCreditsOnlyOne(t *testing.T) {
	ours := Region{Surf: 1, Force: 1, X0: 0, Y0: 0, X1: 1, Y1: 1}
	theirs := Region{Surf: 1, Force: 2, X0: 0, Y0: 2, X1: 1, Y1: 3}
	var c Claims
	// The gate found a part of each; only ours is of the force that will answer.
	c.Add(Tile(1, 1, 1, 1), 7)
	c.Add(Tile(1, 2, 1, 2), 7)
	if got := c.BeneficiaryFor(ours); got != 7 {
		t.Fatalf("our own network credited player %d", got)
	}
	// The other force's network was shrunk by the same keypress and is credited
	// too -- the same player mined the belt off both. What must NOT happen is
	// one force's claim answering the other's pool, which is what the force term
	// in Overlaps is for; here the two claims are distinct and each is answered
	// by its own.
	if theirs.Overlaps(c[0].Where) {
		t.Fatalf("force 2's network was answered by force 1's claim")
	}
	if !theirs.Overlaps(c[1].Where) {
		t.Fatalf("force 2's network was not answered by its own claim")
	}
}

// THE BOUND, and it is why Add dedupes. The gate records one claim per
// registered part tile it found, so a player deconstructing a belt LINE beside
// a balancer -- fifty belts in one tick, which a deconstruction planner does --
// walks the same handful of part tiles fifty times over. Without the dedupe the
// store grows with the sweep rather than with the machine, and under -gc=leaking
// that growth is permanent.
//
// Against the shipped Add, which appends unconditionally, this fails:
//
//	--- FAIL: TestADeconSweepDoesNotGrowTheClaimStore
//	    50 belts mined beside a 4-part balancer left 200 claims, not 4
func TestADeconSweepDoesNotGrowTheClaimStore(t *testing.T) {
	var c Claims
	for belt := 0; belt < 50; belt++ {
		// Every belt in the line sees the same four parts through the gate.
		for part := int32(0); part < 4; part++ {
			c.Add(Tile(1, 1, 0, part), 7)
		}
	}
	if len(c) != 4 {
		t.Fatalf("50 belts mined beside a 4-part balancer left %d claims, not 4",
			len(c))
	}
}

// The dedupe is on the whole key and not on the ground. Two players mining on
// opposite sides of one part in the same tick both claim that part's tile, and
// collapsing them would silently discard one player's evidence -- which matters
// the moment the first of them leaves the game between the mine and the flush.
func TestTwoPlayersOnOneTileAreTwoClaims(t *testing.T) {
	var c Claims
	c.Add(Tile(1, 1, 0, 5), 7)
	c.Add(Tile(1, 1, 0, 5), 9)
	if len(c) != 2 {
		t.Fatalf("two players' claims on one tile collapsed to %d", len(c))
	}
}

// And it must not re-order. Event order is what decides two players mining into
// one teardown, so a duplicate arriving later may not promote its player past
// somebody who was already on the list.
func TestTheDedupeKeepsEventOrder(t *testing.T) {
	net := Region{Surf: 1, Force: 1, X0: 0, Y0: 0, X1: 3, Y1: 3}
	var c Claims
	c.Add(Tile(1, 1, 1, 1), 7)
	c.Add(Tile(1, 1, 2, 2), 9)
	c.Add(Tile(1, 1, 1, 1), 7) // the second belt of player 7's sweep
	if len(c) != 2 {
		t.Fatalf("the duplicate was recorded: %d claims", len(c))
	}
	if got := c.BeneficiaryFor(net); got != 7 {
		t.Fatalf("beneficiary is player %d, not the first claim in event order",
			got)
	}
	if c[1].Player != 9 {
		t.Fatalf("the dedupe re-ordered the store: %+v", c)
	}
}

// Nobody is still not a player on this path either. Every removal a headless run
// can produce carries player_index 0, and the neighbour gate is entered for
// every belt anyone lays or picks up anywhere on the map -- so a store that
// recorded zeroes would grow on the guest's hottest path and would then hand a
// dissolve a beneficiary that does not exist.
func TestABeltMinedByNobodyRecordsNothing(t *testing.T) {
	_, part, _ := beside()
	var c Claims
	for i := 0; i < 100; i++ {
		c.Add(part, 0)
	}
	if len(c) != 0 {
		t.Fatalf("%d claims recorded for player 0", len(c))
	}
}
