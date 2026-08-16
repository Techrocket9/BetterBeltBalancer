// Package carry is the identity a drained network's items are keyed by, and it
// exists so that the two questions asked of that identity cannot drift apart.
//
// carry.go (package main) drains a torn-down network into a POOL and then asks
// two things about it:
//
//   - WHICH network inherits it -- "is this newly built cluster a successor of
//     the one that came down". Same surface, same force, boxes overlapping.
//   - WHO gets what nobody inherits -- "did a player SHRINK this network".
//     Same surface, same force, a tile of the network inside the box. A part
//     they mined is such a tile; so is the part that the belt they mined at the
//     cluster's edge was touching, and the belt's own tile is not.
//
// THEY WERE WRITTEN TWICE AND THE SECOND ONE FORGOT THE FORCE. Two clusters of
// different forces on one surface have adjacent bounding boxes BY CONSTRUCTION
// -- two forces' parts touching are two balancers, which is the semantics the
// registry, the flood fill and the compiler all already obey -- and an L or a
// diagonal makes those boxes overlap outright. So a player mining one force's
// balancer in the same tick another force's network came down could be handed
// the other force's items. Conservation was never at risk; the wrong pocket
// was. It is the same shape as the M3 bug collectCluster had (CLAUDE.md, "two
// bugs M3 found"): a force check that every neighbour of a predicate has and
// that one predicate does not.
//
// So there is ONE predicate here and a claim is a one-tile Region. A third
// question asks the same code or it does not ask at all.
//
// PURE GO -- no fkapi, no wasm imports -- for the same reason `plan` and `skin`
// are: the trigger needs a player and a player needs an interactive game, but
// the PREDICATE needs nothing at all, so `go test ./carry/` runs it under an
// ordinary toolchain and `make check` runs it on every build. That is the only
// part of the miner's pocket any machine in this repo can check.
package carry

// Region is the identity of a piece of ground: the surface it is on, the force
// that owns it, and an INCLUSIVE tile bounding box.
//
// Inclusive because netInfo's box is: it is the bounding box of a cluster's
// PARTS, so both ends are tiles that really hold one, and a half-open reading
// would drop the outermost row of every balancer.
type Region struct {
	Surf, Force    uint32
	X0, Y0, X1, Y1 int32
}

// Tile is the degenerate Region one tile occupies, which is what a beneficiary
// claim is: a player shrank the network standing HERE, on this surface, of this
// force. Nothing else about the mine survives the tick between the event and
// the flush that settles the pool.
//
// HERE IS A TILE OF THE NETWORK. For a mined PART that is the tile the event
// reported and the two readings are the same; for something mined BESIDE the
// cluster they are not, and the caller hands over the tile of the PART it
// touched. See claims.go, and beside_test.go for what the other reading costs.
func Tile(surf, force uint32, x, y int32) Region {
	return Region{Surf: surf, Force: force, X0: x, Y0: y, X1: x, Y1: y}
}

// FollowMerge is what `game.merge_forces(src, dst)` does to an identity: the
// force it names may have stopped existing, and everything that recorded one
// has to follow.
//
// ONE STATEMENT OF THE RULE, for the same reason there is one Overlaps. A carry
// transaction holds a force index in two places -- the drained pools and the
// claims over them -- and `onForcesMerged` tears the networks down BEFORE it
// remaps the registry, so both were written down under a force that is about to
// be destroyed. Remapping one and not the other is exactly as bad as comparing
// one and not the other: the surviving pool no longer recognises the claim, and
// a player who mined a source-force part in the merge tick watches the items go
// on the floor instead of into their pocket.
func (r *Region) FollowMerge(src, dst uint32) {
	if r.Force == src {
		r.Force = dst
	}
}

// Overlaps is the whole test: same surface, same force, boxes touching.
//
// Symmetric, and that is worth having rather than merely being true: a pool
// asks it one way (does this new cluster succeed me) and a claim the other (is
// this tile mine), so a predicate that answered differently depending on which
// side asked would be a bug only one of the two callers could ever see.
func (r Region) Overlaps(o Region) bool {
	return r.Surf == o.Surf && r.Force == o.Force &&
		o.X0 <= r.X1 && o.X1 >= r.X0 && o.Y0 <= r.Y1 && o.Y1 >= r.Y0
}
