package carry

import "testing"

// A merge tick can contain a mine, and that is the whole of this file.
//
// `game.merge_forces(src, dst)` moves every entity of one force onto another
// and destroys the source force, raising no per-entity event for any of it.
// onForcesMerged tears the affected networks DOWN FIRST -- it has to, because a
// cluster absorbed by the merge stops being a root -- and only then remaps. So
// in the merge tick a pool is holding items under a force index that is about
// to be destroyed, and if a player mined a part of that force in the same tick,
// so is the claim over it.
//
// Both have to follow the merge. Remapping only the pool leaves the two naming
// different forces, the surviving network stops recognising the claim, and the
// items the player mined go on the floor. It fails closed -- conservation is
// unaffected, the pool settles into a successor, a pocket or the ground either
// way -- which is exactly why nothing in seven headless suites can see it.

// The merge under test: force 2 is absorbed into force 1, with force 2's
// balancer standing at (10,10)-(13,13) and one of its parts mined by player 7
// in the same tick.
func merged() (Region, Claims) {
	pool := Region{Surf: 1, Force: 2, X0: 10, Y0: 10, X1: 13, Y1: 13}
	var claims Claims
	claims.Add(Tile(1, 2, 12, 12), 7)
	pool.FollowMerge(2, 1)
	claims.FollowMerge(2, 1)
	return pool, claims
}

// THE ONE THIS FILE EXISTS FOR. Transcribing the shipped remap -- the pool
// alone, which is what remapCarryForce did -- fails it:
//
//	--- FAIL: TestAMergeCarriesTheClaimWithThePool
//	    the surviving network credited player 0, not 7: a claim left naming the
//	    destroyed force is a claim nobody's pool can answer
func TestAMergeCarriesTheClaimWithThePool(t *testing.T) {
	pool, claims := merged()
	if got := claims.BeneficiaryFor(pool); got != 7 {
		t.Fatalf("the surviving network credited player %d, not 7: a claim left "+
			"naming the destroyed force is a claim nobody's pool can answer", got)
	}
}

// The destroyed force must not survive anywhere, because a later claim of the
// SURVIVING force on the same ground would then be answered by whichever of the
// two the scan reached first.
func TestNothingKeepsNamingTheDestroyedForce(t *testing.T) {
	_, claims := merged()
	for i := range claims {
		if claims[i].Where.Force == 2 {
			t.Fatalf("claim %d still names force 2, which no longer exists", i)
		}
	}
}

// A merge is not a licence to credit everybody. A third force's miner, on
// ground that overlaps by construction, is untouched by a merge it was not part
// of -- and is still refused by the survivor, which is the property the claim
// predicate already had and that this must not undo.
func TestAMergeLeavesAThirdForceAlone(t *testing.T) {
	pool := Region{Surf: 1, Force: 2, X0: 10, Y0: 10, X1: 13, Y1: 13}
	var claims Claims
	claims.Add(Tile(1, 3, 12, 12), 9) // force 3 mined its own part, same tile
	pool.FollowMerge(2, 1)
	claims.FollowMerge(2, 1)

	if claims[0].Where.Force != 3 {
		t.Fatalf("a merge of 2 into 1 moved force 3's claim to force %d",
			claims[0].Where.Force)
	}
	if got := claims.BeneficiaryFor(pool); got != 0 {
		t.Fatalf("the merged network credited player %d, who mined another "+
			"force's part", got)
	}
}

// The destination force's own claims are already correct, so following the
// merge must be a no-op for them -- and following it twice must be too, since
// the depth counter can put two reconcile paths' remaps in one transaction.
func TestFollowingAMergeIsIdempotent(t *testing.T) {
	var claims Claims
	claims.Add(Tile(1, 1, 5, 5), 4)
	claims.FollowMerge(2, 1)
	claims.FollowMerge(2, 1)
	if claims[0].Where.Force != 1 || claims[0].Player != 4 {
		t.Fatalf("a claim of the surviving force was moved: force %d, player %d",
			claims[0].Where.Force, claims[0].Player)
	}
}

// Event order, which is deterministic, is what decides two players mining into
// one dissolve. Nothing may sort or dedupe, and a merge may not re-order.
func TestTheFirstClaimInEventOrderWins(t *testing.T) {
	net := Region{Surf: 1, Force: 1, X0: 0, Y0: 0, X1: 3, Y1: 3}
	var claims Claims
	claims.Add(Tile(1, 9, 1, 1), 5) // another force's part, same ground
	claims.Add(Tile(1, 1, 2, 2), 6)
	claims.Add(Tile(1, 1, 3, 3), 7)
	if got := claims.BeneficiaryFor(net); got != 6 {
		t.Fatalf("beneficiary is player %d, not the first claim of this force", got)
	}
	claims.FollowMerge(9, 1) // and now the first claim IS of this force
	if got := claims.BeneficiaryFor(net); got != 5 {
		t.Fatalf("after the merge the beneficiary is player %d, not the first "+
			"claim in event order", got)
	}
}

// player_index is 0 on every removal a headless run can produce -- a robot, a
// death, a script destroy, a surface deletion -- and 0 is "nobody", not a
// player. It must never be recorded, or a dissolve with no miner would pocket
// into a player that does not exist.
func TestNobodyIsNotAPlayer(t *testing.T) {
	var claims Claims
	claims.Add(Tile(1, 1, 5, 5), 0)
	if len(claims) != 0 {
		t.Fatalf("a claim was recorded for player 0")
	}
	net := Region{Surf: 1, Force: 1, X0: 0, Y0: 0, X1: 9, Y1: 9}
	if got := claims.BeneficiaryFor(net); got != 0 {
		t.Fatalf("an empty claim list found player %d", got)
	}
}

// Reset is the end of the tick, and it must keep the backing array: the store
// is reused every flush for the life of the mod, and an allocation per flush is
// permanent under -gc=leaking and garbage the pacer walks under the collector.
func TestResetKeepsTheBuffer(t *testing.T) {
	var claims Claims
	claims.Add(Tile(1, 1, 5, 5), 4)
	before := cap(claims)
	claims.Reset()
	if len(claims) != 0 {
		t.Fatalf("Reset left %d claims", len(claims))
	}
	if cap(claims) != before {
		t.Fatalf("Reset released the buffer: cap %d -> %d", before, cap(claims))
	}
}

// A merge remaps the force and NOTHING ELSE. The tile is where the part stood
// and the surface is where it stood on; a merge moves neither.
func TestAMergeMovesNoGround(t *testing.T) {
	var claims Claims
	claims.Add(Tile(4, 2, -7, 11), 3)
	claims.FollowMerge(2, 1)
	want := Tile(4, 1, -7, 11)
	if claims[0].Where != want {
		t.Fatalf("the merge moved the ground under the claim: %+v", claims[0].Where)
	}
}
