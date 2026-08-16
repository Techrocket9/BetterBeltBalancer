package carry

// The claim store: who mined what, for the one tick between the event that
// reported the mine and the flush that settles the pool it drained.
//
// It lives here rather than in package main because it is nothing but data and
// two decisions over it -- which claim a network's pool answers to, and what a
// force merge does to the ones already recorded -- and both are the kind of
// thing this repo has twice got wrong by writing out twice. The pool side of
// each decision is `Region`; this is the other side, so the two sit in one
// package and are proved by one `go test ./carry/`.

// Claim is one player's claim on a network they just made SMALLER, whether by
// mining a part of it -- a shrink or a dissolve alike -- or by mining something
// standing BESIDE it, which takes a port off the machine without taking a part.
//
// THE GROUND IN IT IS ALWAYS THE NETWORK'S, NEVER THE REMOVED ENTITY'S. For a
// part the two are the same tile and the distinction never came up; for a belt
// at the cluster's edge they are not, because a belt adjacent to a cluster is by
// construction one tile OUTSIDE its bounding box. A claim written down where the
// belt stood is a claim no pool can answer -- see beside_test.go, which is where
// that costs nothing to say and everything to get wrong.
//
// SCALARS AND A ONE-TILE REGION, like everything this guest keeps across a
// dispatch: the claim is recorded inside the event that reported the mine and
// consumed by the flush a tick later, and no LuaEntity or LuaPlayer survives
// that gap. The force is part of the key because clusters are per force and two
// forces' boxes are adjacent by construction; see Region.
type Claim struct {
	Where  Region
	Player uint32
}

// Claims is one flush's worth of them, IN EVENT ORDER.
//
// The order is load-bearing rather than incidental: two players mining into one
// dissolve are resolved by taking the first claim the network's box contains,
// and Factorio's event order is deterministic, so every client picks the same
// player. Nothing here sorts, dedupes or re-orders.
type Claims []Claim

// Add records that `player` shrank the network standing on `where`.
//
// Called from the mine event with nothing in hand but what the event carried
// and what the registry already knew -- no host call, so the guest's hottest
// path is untouched -- and whether the claim is ever USED is decided a tick
// later. That is why this is a note rather than a decision.
//
// EXACT DUPLICATES ARE DROPPED, AND THAT IS A BOUND RATHER THAN A TIDY-UP. The
// part path calls this once per removal and could never repeat itself; the
// NEIGHBOUR path calls it once per registered part tile its 5x5 gate found, so a
// player deconstructing a belt line beside a balancer -- fifty belts in one
// tick, which is one drag of a deconstruction planner -- walks the same handful
// of part tiles fifty times over. Deduped, the store is bounded by the MACHINE
// (its parts) instead of by the sweep, which is what makes "one entry per part a
// player mined in one tick" still true of a path that is not about parts at all.
//
// On the whole key, and nothing is re-ordered: two players mining either side of
// one part are two claims, and a duplicate arriving later may not promote its
// player past somebody already on the list. Event order is what resolves two
// miners into one teardown identically on every client.
func (c *Claims) Add(where Region, player uint32) {
	if player == 0 {
		return
	}
	for i := range *c {
		if (*c)[i].Player == player && (*c)[i].Where == where {
			return
		}
	}
	*c = append(*c, Claim{Where: where, Player: player})
}

// BeneficiaryFor is the claim test: the player of the first claim standing on a
// tile of `net`, or 0 for nobody.
//
// ONE PREDICATE WITH THE SUCCESSOR TEST, which is the 2026-08-02 fix rather
// than an arrangement -- this was written out by hand once and compared the
// surface and the box and NOT THE FORCE, while the successor test over the same
// pool always compared all three.
func (c Claims) BeneficiaryFor(net Region) uint32 {
	for i := range c {
		if net.Overlaps(c[i].Where) {
			return c[i].Player
		}
	}
	return 0
}

// FollowMerge follows `game.merge_forces(src, dst)` into the claims already
// recorded.
//
// THE SIBLING OF THE POOL REMAP, and it exists because the pool remap did not
// have one. A merge tick can contain a mine: the part was of the source force,
// so the claim names a force the merge is about to destroy, while the pool it
// belongs to is remapped to the survivor. Without this the two stop matching
// and the miner's items reach the floor instead of the pocket -- conservation
// intact, the feature wrong, which is the same shape as the force check the
// claim predicate itself was missing.
func (c Claims) FollowMerge(src, dst uint32) {
	for i := range c {
		c[i].Where.FollowMerge(src, dst)
	}
}

// Reset ends the tick. A claim outlives the event that made it by one flush and
// no longer: the flush that could have used it has run.
func (c *Claims) Reset() { *c = (*c)[:0] }
