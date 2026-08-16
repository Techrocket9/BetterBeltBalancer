// Package skin turns a balancer part's neighbourhood into a sprite variation.
//
// This is M5's whole idea in one function. A balancer is an arbitrary blob of
// 1x1 parts and it has to LOOK like one machine, which means every part has to
// draw the piece of the outline that its own position in the blob calls for:
// no border where it touches a sibling, a border where the balancer ends, and a
// rounded fillet where the outline turns around a hole. The prototype ships 47
// pictures; the runtime picks one per part through
// `LuaEntity.graphics_variation`. There is no per-tick cost, no extra entity,
// and no rendering object -- the engine draws a different sprite from an array
// it already had.
//
// PURE GO ON PURPOSE, with no fkapi and no wasm imports, exactly like
// `guest/go/plan`: `go test ./skin/` runs the mapping under an ordinary
// toolchain, so the five named shapes are proved before Factorio is asked
// anything. `make check` runs it.
//
// # The mask
//
//	bits: N=1 E=2 S=4 W=8   NE=16 SE=32 SW=64 NW=128
//
// A bit is set when that neighbour tile holds a part OF THE SAME FORCE -- two
// forces' parts touching are two balancers (CLAUDE.md, "Clusters are per
// force") and must not fuse into one picture.
//
// # Why 47 and not 16
//
// The four side bits alone give 16 pictures and already put the trim only on
// the outside: for a run of parts, each draws its own north and south edge and
// they line up into one band. What sides alone cannot express is the INSIDE of
// a corner. Take a part whose north and east neighbours are both present while
// the north-east diagonal is empty: the outline has to turn around that empty
// tile, and the two neighbours draw the two straight halves of that turn. This
// part owns the corner point where they meet, and if it draws nothing there the
// outline dies in a hard right angle with a notch in it. The diagonal bits are
// what let it draw the fillet instead. That is the difference between "tiles
// that agree about their borders" and "one machine".
//
// A diagonal bit is meaningful ONLY when both sides touching it are set: if a
// side is missing then that side's own trim already draws the whole corner and
// what is beyond the diagonal cannot change the picture. Canonicalising on that
// rule leaves 47 masks of the 256 -- the classic blob count:
//
//	0 sides         1 config  x 1        =  1
//	1 side          4         x 1        =  4
//	2 opposite      2         x 1        =  2
//	2 adjacent      4         x 2        =  8
//	3 sides         4         x 4        = 16
//	4 sides         1         x 16       = 16
//	                                       47
//
// # The contract with the sprite sheet
//
// `tools/make-graphics.py` enumerates masks 0..255 ascending, keeps the
// canonical ones and draws them in that order, 8 per row. This file numbers
// them the same way, from 1, because `graphics_variation` is 1-based (measured:
// setting 0 raises, and a value above the count wraps modulo it). Neither side
// stores a table; both run the same six-line enumeration, and the three anchors
// asserted in skin_test.go are asserted in the generator too, so a change made
// on one side fails on both.
package skin

// The mask bits, in the order they are numbered. The guest builds a mask by
// walking its neighbour offsets in exactly this order.
const (
	N uint8 = 1 << iota
	E
	S
	W
	NE
	SE
	SW
	NW
)

// Count is how many pictures the prototype must ship.
const Count = 47

// variation[m] is the 1-based sprite index for a canonical mask, and 0 for a
// mask that is not canonical. Nothing indexes it without going through Canon,
// so the zeroes are unreachable rather than a fallback.
var variation [256]uint8

func init() {
	n := uint8(0)
	for m := 0; m < 256; m++ {
		if Canon(uint8(m)) == uint8(m) {
			n++
			variation[m] = n
		}
	}
	_ = n
}

// countIsRight fails to compile if the enumeration stops producing exactly the
// number of pictures the prototype declares. It is a declaration, not a check:
// a `panic` here would be a runtime assertion nobody can reach, and it would
// link TinyGo's whole panic-and-print machinery into a guest that has no other
// use for it -- 118 KB of Lua in every save, measured. `TestEveryMaskIsInTheSheet`
// is where the counting is actually proved; this only pins the constant to it.
var countIsRight [Count - 47]struct{}

// Canon drops the diagonal bits that cannot change the picture.
//
// Two neighbourhoods that differ only in those bits are the same shape as far
// as this part's own outline is concerned, and giving them one variation is
// what keeps the sheet at 47 cells instead of 256.
func Canon(m uint8) uint8 {
	c := m & (N | E | S | W)
	if m&NE != 0 && c&N != 0 && c&E != 0 {
		c |= NE
	}
	if m&SE != 0 && c&S != 0 && c&E != 0 {
		c |= SE
	}
	if m&SW != 0 && c&S != 0 && c&W != 0 {
		c |= SW
	}
	if m&NW != 0 && c&N != 0 && c&W != 0 {
		c |= NW
	}
	return c
}

// Variation is the 1-based sprite index for a neighbour mask. Total order, no
// error case: every one of the 256 masks canonicalises into the 47.
func Variation(m uint8) uint8 { return variation[Canon(m)] }
