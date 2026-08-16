package skin

import (
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The three anchors tools/make-graphics.py asserts about its own enumeration.
// They are here so that a change made on one side fails on both, which is the
// only thing keeping the sheet's cell order and this file's numbering together.
func TestAnchors(t *testing.T) {
	if Variation(0) != 1 {
		t.Fatalf("the lone part is cell 1, got %d", Variation(0))
	}
	if Variation(N|E|S|W) != 16 {
		t.Fatalf("the fully enclosed part with no diagonals is cell 16, got %d",
			Variation(N|E|S|W))
	}
	if Variation(0xFF) != Count {
		t.Fatalf("the deep interior is the last cell (%d), got %d", Count, Variation(0xFF))
	}
}

// Every mask must land inside the sheet, and the canonical ones must be exactly
// Count of them -- a variation above the count silently WRAPS in the engine
// (measured: 48 draws cell 1, 255 draws cell 20), so an off-by-one here is a
// wrong picture rather than an error anyone would see.
func TestEveryMaskIsInTheSheet(t *testing.T) {
	seen := map[uint8]bool{}
	for m := 0; m < 256; m++ {
		v := Variation(uint8(m))
		if v < 1 || v > Count {
			t.Fatalf("mask %d -> variation %d, outside 1..%d", m, v, Count)
		}
		if Canon(uint8(m)) == uint8(m) {
			seen[v] = true
		}
		// Canonicalising is idempotent, or the guest and the generator could
		// disagree about which of two masks names a cell.
		if Canon(Canon(uint8(m))) != Canon(uint8(m)) {
			t.Fatalf("Canon is not idempotent at %d", m)
		}
	}
	if len(seen) != Count {
		t.Fatalf("canonical masks reach %d cells, want %d", len(seen), Count)
	}
}

// A diagonal may never change the picture when one of the sides it touches is
// missing: that side's own trim already draws the corner. This is the whole
// justification for 47 cells instead of 256, and it is what stops a part from
// changing its sprite because something appeared diagonally across a gap.
func TestDiagonalsOnlyMatterBetweenTwoConnectedSides(t *testing.T) {
	for m := 0; m < 256; m++ {
		for _, d := range []uint8{NE, SE, SW, NW} {
			var a, b uint8
			switch d {
			case NE:
				a, b = N, E
			case SE:
				a, b = S, E
			case SW:
				a, b = S, W
			case NW:
				a, b = N, W
			}
			if uint8(m)&a != 0 && uint8(m)&b != 0 {
				continue
			}
			if Variation(uint8(m)) != Variation(uint8(m)|d) {
				t.Fatalf("mask %d changes picture when %d is added, "+
					"but a touching side is missing", m, d)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// The five named shapes.
//
// These are the same five the headless suite builds in a real Factorio and
// reads back out of the guest's own log line, in the same (y, x) order, so a
// failure here and a failure there are the same failure told twice: this one
// says the mapping is wrong, that one says the mapping never reached an entity.
// ---------------------------------------------------------------------------

type tile struct{ x, y int }

func vars(tiles []tile) string {
	in := map[tile]bool{}
	for _, t := range tiles {
		in[t] = true
	}
	ord := append([]tile(nil), tiles...)
	sort.Slice(ord, func(i, j int) bool {
		if ord[i].y != ord[j].y {
			return ord[i].y < ord[j].y
		}
		return ord[i].x < ord[j].x
	})
	out := make([]string, 0, len(ord))
	for _, t := range ord {
		var m uint8
		for _, n := range []struct {
			bit    uint8
			dx, dy int
		}{
			{N, 0, -1}, {E, 1, 0}, {S, 0, 1}, {W, -1, 0},
			{NE, 1, -1}, {SE, 1, 1}, {SW, -1, 1}, {NW, -1, -1},
		} {
			if in[tile{t.x + n.dx, t.y + n.dy}] {
				m |= n.bit
			}
		}
		out = append(out, strconv.Itoa(int(Variation(m))))
	}
	return strings.Join(out, ",")
}

func TestNamedShapes(t *testing.T) {
	cases := []struct {
		name  string
		tiles []tile
		want  string
	}{
		// A run: two ends and two middles, and the two middles are the SAME
		// cell, which is what makes a long balancer one unbroken bar.
		{"line", []tile{{0, 0}, {1, 0}, {2, 0}, {3, 0}}, "3,11,11,9"},
		// The corner is cell 4 (north and east, no diagonal): the shape a
		// neighbour-COUNT heuristic gets wrong, exactly as in the M1 suite.
		{"L", []tile{{0, 0}, {0, 1}, {0, 2}, {1, 2}, {2, 2}}, "5,6,4,11,9"},
		// The centre is the only part in these five with all four sides and no
		// diagonals: cell 16, the last of the side-only block.
		{"plus", []tile{{1, 0}, {0, 1}, {1, 1}, {2, 1}, {1, 2}}, "5,3,16,9,2"},
		// EVERY CELL HERE IS ABOVE 16, which is the point of the test: a 2x2 is
		// four corner parts that each see their diagonal, and a 16-variant
		// scheme would draw them identically to an L's corner and leave a notch
		// in the middle of the block.
		{"2x2 block", []tile{{0, 0}, {1, 0}, {0, 1}, {1, 1}}, "21,27,17,35"},
		// The donut: a 4x4 ring around a 1x1... a 2x2 hole. The four corners of
		// the ring are ordinary corners; the four straight runs are cells 6 and
		// 11; and the four parts touching the hole diagonally are NOT the same
		// cell as the ones that do not, which is the inner outline being drawn.
		{"donut", ring4(), "7,11,11,13,6,6,6,6,4,11,11,10"},
	}
	for _, c := range cases {
		if got := vars(c.tiles); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func ring4() []tile {
	var out []tile
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x >= 1 && x <= 2 && y >= 1 && y <= 2 {
				continue
			}
			out = append(out, tile{x, y})
		}
	}
	return out
}
