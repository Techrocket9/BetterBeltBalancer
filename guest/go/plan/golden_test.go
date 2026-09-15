package plan

import (
	"fmt"
	"math"
	"os"
	"testing"
)

// The plain butterfly, pinned op for op.
//
// The priority construction reaches into Build, and the one property no
// priority test can assert for it is that an edge list with NO Prio in it comes
// out exactly as it did before any of it was written. Every rate, every heap
// slope and every entity count this repository has recorded was measured on
// that network, and a change to it that still balances would pass the whole
// existing suite: TestEveryRowEndsEqual reads the model rather than the ops,
// TestEntityCount reads len(ops), and TestNoTwoHiddenEntitiesShareATile is
// satisfied by any layout that does not collide. A digest over the ops is the
// only thing that sees a splitter that moved one tile.
//
// The digests below were taken at fdf49b9, the commit before Build learned the
// word Prio, and they are a record rather than an expectation: a change that
// moves one is a change to the shipped network and belongs in a commit that
// says so and re-measures what CLAUDE.md records.

// opDigest is FNV-1a over every field of every op, in order. Positions go in as
// their IEEE bits rather than as text, so a tile-and-a-half is not rounded into
// agreement with a tile.
func opsDigest(ops []Op) uint64 {
	const (
		off   = 14695981039346656037
		prime = 1099511628211
	)
	h := uint64(off)
	put := func(v uint64) {
		for i := 0; i < 8; i++ {
			h ^= v & 0xff
			h *= prime
			v >>= 8
		}
	}
	put(uint64(len(ops)))
	for _, o := range ops {
		put(math.Float64bits(o.X))
		put(math.Float64bits(o.Y))
		put(uint64(uint32(o.Pair)))
		put(uint64(o.Dir))
		put(uint64(o.Proto))
		put(uint64(o.Link))
		if o.Visible {
			put(1)
		} else {
			put(0)
		}
		put(uint64(uint8(o.OutPrio)))
		put(uint64(uint8(o.InPrio)))
	}
	return h
}

// goldenShapes is every shape with n, m <= 8 -- the band M2 and the pure model
// both work over -- plus the loopback spread plan_test.go already keeps, plus
// the four sizes past P=8 that nothing else builds twice.
func goldenShapes() [][2]int {
	out := make([][2]int, 0, 64+len(loopShapes)+8)
	for n := 1; n <= 8; n++ {
		for m := 1; m <= 8; m++ {
			out = append(out, [2]int{n, m})
		}
	}
	out = append(out, loopShapes...)
	out = append(out, [2]int{16, 16}, [2]int{9, 16}, [2]int{32, 32},
		[2]int{17, 32}, [2]int{64, 64}, [2]int{33, 64}, [2]int{64, 33}, [2]int{1, 64})
	return out
}

// TestPlainShapesAreByteIdenticalToTheRecord builds every shape at a fixed slot
// origin and checks the digest. The origin is not (0,0): a plan that dropped
// the offset would be caught by TestPlanFitsItsSlot only at the top-left slot.
func TestPlainShapesAreByteIdenticalToTheRecord(t *testing.T) {
	const ox, oy = 100, 200
	if os.Getenv("BBB_PRINT_GOLDEN") != "" {
		for _, nm := range goldenShapes() {
			ops, _, ok := Build(nil, edges(nm[0], nm[1]), ox, oy)
			if !ok {
				t.Fatalf("%v refused", nm)
			}
			fmt.Printf("\t{%d, %d, 0x%016x},\n", nm[0], nm[1], opsDigest(ops))
		}
		t.Skip("printed the table")
	}
	want := goldenDigests
	if len(want) != len(goldenShapes()) {
		t.Fatalf("%d shapes and %d recorded digests", len(goldenShapes()), len(want))
	}
	for i, nm := range goldenShapes() {
		if want[i].n != nm[0] || want[i].m != nm[1] {
			t.Fatalf("row %d of the record is %d->%d and the shape list has %d->%d",
				i, want[i].n, want[i].m, nm[0], nm[1])
		}
		ops, _, ok := Build(nil, edges(nm[0], nm[1]), ox, oy)
		if !ok {
			t.Fatalf("%d->%d refused", nm[0], nm[1])
		}
		if got := opsDigest(ops); got != want[i].d {
			t.Errorf("%d->%d: %d ops digest to 0x%016x, the plain network is 0x%016x",
				nm[0], nm[1], len(ops), got, want[i].d)
		}
	}
}
