package carry

import "testing"

// The two networks the whole package is about: same surface, DIFFERENT forces,
// boxes that overlap. That is not a contrived arrangement -- two forces' parts
// touching are two balancers by construction (CLAUDE.md, "clusters are per
// force"), so their bounding boxes are adjacent, and an L or a diagonal makes
// them overlap outright.
var (
	netA = Region{Surf: 1, Force: 1, X0: 10, Y0: 10, X1: 13, Y1: 13}
	netB = Region{Surf: 1, Force: 2, X0: 12, Y0: 12, X1: 15, Y1: 15}
)

// A claim is one tile, so it is a Region like everything else -- and it must go
// through the same predicate, or the two answers drift. They did.
func TestAClaimOfAnotherForceIsNotThisNetworksClaim(t *testing.T) {
	// Tile (12,12) is inside BOTH boxes. It is force 2's part, so force 1's
	// network must not credit force 2's miner with what it drained.
	claim := Tile(1, 2, 12, 12)
	if netA.Overlaps(claim) {
		t.Fatalf("network of force %d accepted a claim on force %d's part at "+
			"(12,12): one force's miner would pocket the other's items",
			netA.Force, claim.Force)
	}
	if !netB.Overlaps(claim) {
		t.Fatalf("network of force %d rejected a claim on its own part", netB.Force)
	}
}

// The claim test and the successor test are one predicate, which is the
// property that makes the drift above impossible rather than merely fixed.
func TestAClaimIsJustADegenerateRegion(t *testing.T) {
	for _, c := range []struct {
		surf, force uint32
		x, y        int32
	}{
		{1, 1, 10, 10}, {1, 1, 13, 13}, {1, 1, 12, 11}, // inside
		{1, 1, 9, 10}, {1, 1, 14, 13}, {1, 1, 12, 14}, // outside
		{1, 2, 11, 11}, {2, 1, 11, 11}, // wrong force, wrong surface
	} {
		tile := Tile(c.surf, c.force, c.x, c.y)
		want := netA.Overlaps(tile)
		got := netA.Surf == c.surf && netA.Force == c.force &&
			c.x >= netA.X0 && c.x <= netA.X1 && c.y >= netA.Y0 && c.y <= netA.Y1
		if got != want {
			t.Fatalf("Tile(%d,%d,%d,%d): the one-tile region and the written-out "+
				"membership test disagree (%v vs %v)", c.surf, c.force, c.x, c.y, want, got)
		}
	}
}

// A claim ON a network's own part is the ordinary case and must still work: the
// box is the bounding box of the cluster's PARTS, so the mined part's tile is
// one of them by construction, corners included.
func TestAMinersOwnTileIsAlwaysInside(t *testing.T) {
	for _, p := range [][2]int32{{10, 10}, {13, 10}, {10, 13}, {13, 13}, {11, 12}} {
		if !netA.Overlaps(Tile(1, 1, p[0], p[1])) {
			t.Fatalf("a part of the cluster at (%d,%d) is outside its own box", p[0], p[1])
		}
	}
}

// The three shapes carryPool.matches exists for, unchanged by the claim sharing
// its code: a recompile matches itself, a merged cluster contains both halves,
// and both of a split's successors are inside the old box.
func TestTheSuccessorShapes(t *testing.T) {
	old := Region{Surf: 3, Force: 7, X0: 0, Y0: 0, X1: 7, Y1: 1}
	same := old
	merged := Region{Surf: 3, Force: 7, X0: 0, Y0: 0, X1: 11, Y1: 1}
	left := Region{Surf: 3, Force: 7, X0: 0, Y0: 0, X1: 2, Y1: 1}
	right := Region{Surf: 3, Force: 7, X0: 5, Y0: 0, X1: 7, Y1: 1}
	for i, r := range []Region{same, merged, left, right} {
		if !old.Overlaps(r) {
			t.Fatalf("successor %d did not claim the pool it descends from", i)
		}
	}
	// ... and neither surface nor force is negotiable for any of them.
	for i, r := range []Region{same, merged, left, right} {
		wrongForce, wrongSurf := r, r
		wrongForce.Force++
		wrongSurf.Surf++
		if old.Overlaps(wrongForce) || old.Overlaps(wrongSurf) {
			t.Fatalf("successor %d claimed across a force or a surface", i)
		}
	}
}

// Overlap is a statement about two boxes and says the same thing whichever is
// asked. A pool asks it one way (does this new cluster succeed me) and a claim
// the other (is this tile mine), so an asymmetric predicate would be a bug that
// only one of the two callers could see.
func TestOverlapsIsSymmetric(t *testing.T) {
	boxes := []Region{netA, netB,
		{Surf: 1, Force: 1, X0: 13, Y0: 13, X1: 13, Y1: 13},
		{Surf: 1, Force: 1, X0: 14, Y0: 14, X1: 20, Y1: 20},
		{Surf: 1, Force: 1, X0: -4, Y0: -4, X1: 11, Y1: 11},
	}
	for i, a := range boxes {
		for j, b := range boxes {
			if a.Overlaps(b) != b.Overlaps(a) {
				t.Fatalf("Overlaps(%d,%d) is not symmetric", i, j)
			}
		}
	}
}

// Inclusive on every bound, because netInfo's box is inclusive tile
// coordinates: two boxes that share an edge tile really are touching, and a
// half-open reading would drop the outermost row of every balancer.
func TestBoundsAreInclusive(t *testing.T) {
	a := Region{Surf: 1, Force: 1, X0: 0, Y0: 0, X1: 4, Y1: 4}
	touch := Region{Surf: 1, Force: 1, X0: 4, Y0: 4, X1: 9, Y1: 9}
	clear := Region{Surf: 1, Force: 1, X0: 5, Y0: 5, X1: 9, Y1: 9}
	if !a.Overlaps(touch) {
		t.Fatal("boxes sharing their corner tile did not overlap")
	}
	if a.Overlaps(clear) {
		t.Fatal("boxes one tile apart overlapped")
	}
}

// Negative tile coordinates are ordinary in Factorio -- the origin is the
// middle of the map -- so nothing here may assume a sign.
func TestNegativeCoordinates(t *testing.T) {
	r := Region{Surf: 1, Force: 1, X0: -9, Y0: -9, X1: -5, Y1: -5}
	if !r.Overlaps(Tile(1, 1, -7, -9)) {
		t.Fatal("a tile inside a negative box was rejected")
	}
	if r.Overlaps(Tile(1, 1, -4, -7)) {
		t.Fatal("a tile outside a negative box was accepted")
	}
}
