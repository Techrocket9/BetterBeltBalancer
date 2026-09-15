// Command bbb-prio-test is the output-priority suite's observer: balancers with
// a port that is fed before the others, and the four ways asking for one is
// refused.
//
// ---------------------------------------------------------------------------
// WHAT THE RIGS ARE FOR
// ---------------------------------------------------------------------------
//
// The promise is two formulas (agents/priority.md, "The promise"): with S belts
// arriving and q of the M outputs flagged, a flagged port carries min(S/q, 1)
// and every other one carries min(max(S-q, 0)/(M-q), 1). It is exact at every
// load rather than close at most of them, so the rigs are chosen to separate the
// three regimes -- the priority tier cannot fill, it is full and the rest share
// what is left, everything is saturated -- and the load is set with belt TIERS,
// which is `m2`'s `tslow` technique: a normal-tier belt is exactly a third of an
// express one, so a row fed through one carries a third of a belt.
//
// EVERY RATE IS READ AGAINST `ctrl`, a bare express belt in the same save fed
// and drained the same way, so "a full belt" is a comparison with the engine
// rather than with arithmetic on a wiki number.
//
// ---------------------------------------------------------------------------
// WHICH PART CARRIES THE FLAG
// ---------------------------------------------------------------------------
//
// The flag is the PART's (guest/go/cluster.go's `pprio`), and `classifyEdges`
// stamps it onto every edge that tile carries. Under the one-belt-per-part rule
// a part carries one, so flagging the EAST part of a row flags that row's
// OUTPUT and nothing else. Which physical belt the priority tier feeds is
// therefore decided here, by which part is flagged, and the port numbering
// inside the planner never has to be known: the visible interfaces are appended
// in edge order, so the chest at the end of the flagged row's belt is the
// priority port's chest.
//
// NOTHING HERE ASSERTS. The mod's own `[BBB]` lines and the chests are the
// evidence and test/assert-prio.py decides.
//
// ---------------------------------------------------------------------------
// THE REMOTE METHOD, AND WHY THE SUITE IS DRIVEN THROUGH IT
// ---------------------------------------------------------------------------
//
// The player's door onto a part's priority is ALT + P, and a keypress is a path
// a script cannot take: `script.raise_event` refuses a custom input outright and
// a headless run has no player to press one. So the mod opens a second door
// (`set-part-priority`, guest/go/commands.go) that reaches the same
// `setPartPriority` with no branch below it that can tell the two apart, which
// is the same argument `bbb-audit` and `set-curved-exits` already have.
//
// THE STATUS IS READ RATHER THAN THE CALL MADE BARE, for `flip`'s reason: a bare
// `remote.call` RAISES on a missing interface or method and would abort the
// schedule from inside a tick handler, where `fkapi.RemoteCall` hands the same
// refusal back as a Status and the run reaches the assertion that says so.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

var out = harness.Line{Tag: "[BBB-PRIO] "}

const (
	part     = "bbb-balancer-part"
	belt     = "express-transport-belt"
	slowBelt = "transport-belt" // normal tier: exactly 1/3 of express
	loader   = protos.PrioLoader
	surf     = "bbb-prio"
	hidden   = "bbb-hidden"

	modName    = "better-belt-balancer"
	prioMethod = "set-part-priority"

	pitch = 12 // rows between ordinary rigs
	halfX = 14 // how far either side of x=0 the scratch surface is cleared

	// p32's three sources are FINITE, because "each input is drawn equally" is a
	// statement about what LEFT them and an infinity chest never goes down. The
	// stock is sized so that no source can empty inside the run: three inputs of
	// a saturated 3 -> 2 draw two thirds of a belt each, which over the whole
	// benchmark is well under this.
	p32Stock = 6000
)

var dirN, dirE, dirS uint32

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirN = fkapi.DefinesDirectionNorth()
	dirE = fkapi.DefinesDirectionEast()
	dirS = fkapi.DefinesDirectionSouth()
	layOutBands()
}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true,
	})
}

func beltsX(s fkapi.LuaSurface, name string, dir *uint32, from, to, fixedY int) {
	step := 1
	if to < from {
		step = -1
	}
	for i := from; ; i += step {
		put(s, name, i, fixedY, dir, "")
		if i == to {
			break
		}
	}
}

func beltsY(s fkapi.LuaSurface, name string, dir *uint32, from, to, fixedX int) {
	step := 1
	if to < from {
		step = -1
	}
	for i := from; ; i += step {
		put(s, name, fixedX, i, dir, "")
		if i == to {
			break
		}
	}
}

// source is an infinity chest of iron plate and the loader that drains it. The
// chest is placed WITHOUT `raise_built` and the loader with it, as the estate
// has always had them: the mod under test has to see the loader, which is an
// edge candidate standing next to a part, and has no interest in a chest.
func source(s fkapi.LuaSurface, x, y int, dir uint32) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	harness.InfinityFilter(c, "iron-plate", "", 1000)
	if dir == dirS {
		put(s, loader, x, y+1, &dirS, "output")
	} else {
		put(s, loader, x+1, y, &dirE, "output")
	}
}

func sourceE(s fkapi.LuaSurface, x, y int) { source(s, x, y, dirE) }

// finiteSource is sourceE with a steel chest and a known stock, so that what a
// rig DREW can be read off the chest afterwards.
func finiteSource(s fkapi.LuaSurface, x, y int) harness.XY {
	c := harness.Place(s, harness.Piece{Name: "steel-chest", X: x, Y: y})
	harness.InsertInto(c, "iron-plate", p32Stock)
	put(s, loader, x+1, y, &dirE, "output")
	return harness.XY{X: x, Y: y}
}

// sink is the loader that fills a steel chest, and it returns the CHEST TILE
// rather than the chest: a rig registry holds tiles, so what it keeps goes on
// being true across the save between `--create` and `--benchmark`.
func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	put(s, loader, x, y, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

// ---------------------------------------------------------------------------
// the rigs
// ---------------------------------------------------------------------------

// tier is what an input row is fed through, in thirds of an express belt.
type tier int

const (
	none    tier = iota // an input belt with no source at all
	slow                // a normal-tier run: exactly a third of a belt
	express             // a full belt
)

type rigCfg struct {
	name string

	// rows, ins and outs drive the uniform builder; rows == 0 means the control
	// or a custom builder.
	rows, ins, outs int

	// span overrides pitch, for the two column rigs.
	span int

	// feed is one entry per input row. A shorter list leaves the rest at `none`,
	// which is what `p4lo` and `pq2` want.
	feed []tier

	// prio lists the 1-based OUTPUT rows whose east part is flagged in
	// `fk_on_init`. A rig whose flag is set at run time instead leaves it empty.
	prio []int

	// blocked lists the 1-based output rows with nowhere to go.
	blocked []int

	// deadEnd blocks EVERY output, which is what the spill guard needs: a
	// balancer that is moving anything has room, and the guard is about one that
	// has none.
	deadEnd bool

	// build names a custom builder.
	build func(s fkapi.LuaSurface, base int) []harness.XY

	base int
}

// The rig table. Every rate row states the load in belts fed and what the two
// formulas make of it; test/assert-prio.py holds the same numbers and is what
// compares them with the chests.
//
//	p4lo    4 -> 4, q=1, S = 1/3   the priority tier cannot fill: 1/3 and nothing
//	p4mid   4 -> 4, q=1, S = 2     it is full and three share one belt: 1 and 1/3
//	p4sat   4 -> 4, q=1, S = 4     everything saturated: a belt each
//	p2lo    2 -> 2, q=1, S = 1/3   the same three regimes at the smallest size
//	p2mid   2 -> 2, q=1, S = 4/3   1 and 1/3
//	p2sat   2 -> 2, q=1, S = 2     1 and 1
//	pq2     4 -> 4, q=2, S = 1     TWO priority ports sharing: 1/2 each, 0 and 0
//	p35     3 -> 5, q=1, S = 3     N != M, so the head has a loopback in it
//	p32     3 -> 2, q=1, saturated both ports full AND each input drawn equally,
//	                               which is the property the balancing butterfly
//	                               in front of the concentrator exists for
//	pblk    4 -> 4, q=1, S = 1     the PRIORITY port dead-ended: the tiers absorb
//	                               it and the three normal ports take 1/3 each
//	plane   2 -> 2, q=1, S = 1     both inputs SIDE-LOADED, half a belt on one
//	                               lane each, so the priority port has to carry a
//	                               full belt on BOTH lanes out of two half-lanes
//	ptog    2 -> 2, S = 4/3        the toggle rig, built PLAIN: 2/3 and 2/3 until
//	                               the flag goes on and 1 and 1/3 while it is
//	pfull   4 -> 4, q=1, S = 4     every output dead-ended, so it fills to a stop:
//	                               the spill guard's rig
//	pin     2 -> 2, S = 2          the input refusal's rig, built plain
//	pbig    33 -> 2                P = 64, so a priority port does not fit: the
//	                               toggle refusal's rig
//	pgrow   32 -> 2, q=1           P = 32 and fits, until a 33rd input belt lands
//	                               on its spare part and makes it P = 64: the
//	                               BUILD refusal's rig
var rigs = []rigCfg{
	{name: "ctrl"},
	{name: "p4lo", rows: 4, ins: 4, outs: 4, feed: []tier{slow}, prio: []int{1}},
	{name: "p4mid", rows: 4, ins: 4, outs: 4, feed: []tier{express, express}, prio: []int{1}},
	{name: "p4sat", rows: 4, ins: 4, outs: 4,
		feed: []tier{express, express, express, express}, prio: []int{1}},
	{name: "p2lo", rows: 2, ins: 2, outs: 2, feed: []tier{slow}, prio: []int{1}},
	{name: "p2mid", rows: 2, ins: 2, outs: 2, feed: []tier{express, slow}, prio: []int{1}},
	{name: "p2sat", rows: 2, ins: 2, outs: 2, feed: []tier{express, express}, prio: []int{1}},
	{name: "pq2", rows: 4, ins: 4, outs: 4, feed: []tier{express}, prio: []int{1, 2}},
	{name: "p35", rows: 5, ins: 3, outs: 5,
		feed: []tier{express, express, express}, prio: []int{1}},
	{name: "p32", rows: 3, ins: 3, outs: 2, build: buildP32},
	{name: "pblk", rows: 4, ins: 4, outs: 4, feed: []tier{express},
		prio: []int{1}, blocked: []int{1}},
	{name: "plane", outs: 2, build: buildLane},
	{name: "ptog", rows: 2, ins: 2, outs: 2, feed: []tier{express, slow}},
	{name: "pfull", rows: 4, ins: 4, outs: 4,
		feed: []tier{express, express, express, express}, prio: []int{1}, deadEnd: true},
	{name: "pin", rows: 2, ins: 2, outs: 2, feed: []tier{express, express}},

	// THE BOUNDARY RIG. pfull above is the guard refusing outright; this one is
	// the guard LETTING GO. It fills the same way, is then unblocked with its
	// feed cut, and the clear is asked over and over as it drains -- so the tick
	// it is accepted on is the tick the machine crossed under the successor's
	// capacity, which is the one place the guard's arithmetic is actually
	// exercised rather than merely satisfied. Nothing here writes that capacity
	// down: the mod's own answer is what says when the boundary was crossed.
	{name: "pbnd", rows: 2, ins: 2, outs: 2,
		feed: []tier{express, express}, prio: []int{1}, deadEnd: true},

	// THE THREE MEASUREMENT RIGS, which assert nothing. Each is dead-ended and
	// fed from tick 0, so each is stationary and full by the time it is read,
	// and what it was HOLDING is measured by opening it with the feed cut and
	// counting what lands in the chests, less what its own visible belts were
	// carrying. That is the fill fraction the spill guard's bound is derived
	// from and the one number agents/priority.md's capacity table has never had
	// from a game.
	{name: "mp22", rows: 2, ins: 2, outs: 2,
		feed: []tier{express, express}, deadEnd: true},
	{name: "mq22", rows: 2, ins: 2, outs: 2,
		feed: []tier{express, express}, prio: []int{1}, deadEnd: true},
	{name: "mp44", rows: 4, ins: 4, outs: 4,
		feed: []tier{express, express, express, express}, deadEnd: true},

	{name: "pbig", outs: 2, span: 44, build: buildBig},
	{name: "pgrow", outs: 3, span: 46, build: buildGrow},
}

func (c *rigCfg) feedAt(i int) tier {
	if i-1 < len(c.feed) {
		return c.feed[i-1]
	}
	return none
}

func (c *rigCfg) isBlocked(i int) bool {
	if c.deadEnd {
		return true
	}
	for _, b := range c.blocked {
		if b == i {
			return true
		}
	}
	return false
}

func (c *rigCfg) isPrio(i int) bool {
	for _, p := range c.prio {
		if p == i {
			return true
		}
	}
	return false
}

func (c *rigCfg) nOut() int {
	if c.outs == 0 {
		return 1
	}
	return c.outs
}

// rigState is what a rig keeps once it is built: the tile of every reported
// output chest, and the tiles the schedule has to reach back into.
type rigState struct {
	outs []outSlot
	// flags is every part flagged in `fk_on_init`, and flag is the one a
	// run-time toggle names. Both are TILES computed by whichever builder laid
	// the rig, because the uniform layout's row arithmetic is wrong for three of
	// the rigs and a second copy of it here would be wrong silently.
	flags []harness.XY
	flag  harness.XY
}

type outSlot struct {
	xy  harness.XY
	has bool
}

var (
	built    []rigState
	laneBase int
	rows     int
)

func layOutBands() {
	base := 0
	for i := range rigs {
		rigs[i].base = base
		span := rigs[i].span
		if span == 0 {
			span = pitch
		}
		base += span
	}
	rows = base + pitch
}

func makeSurface() fkapi.LuaSurface {
	return harness.Flat{
		Name:        surf,
		MapWidth:    1024,
		MapHeight:   1024,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		ChunkRadius: uint32((rows+31)/32) + 4,
		X0:          -halfX, Y0: -8, X1: halfX, Y1: rows + 8,
		Tile: "grass-1",
	}.Make()
}

func buildInputRow(s fkapi.LuaSurface, y int, t tier) {
	name := belt
	if t == slow {
		name = slowBelt
	}
	if t != none {
		sourceE(s, -5, y)
	}
	beltsX(s, name, &dirE, -3, -1, y)
}

func buildOutputRow(s fkapi.LuaSurface, y int, blocked bool) outSlot {
	beltsX(s, belt, &dirE, 2, 4, y)
	if blocked {
		return outSlot{}
	}
	return outSlot{xy: sink(s, 5, y), has: true}
}

func buildRig(c *rigCfg) rigState {
	s := harness.Surface(surf)
	st := rigState{outs: make([]outSlot, c.nOut())}

	if c.build != nil {
		for i, xy := range c.build(s, c.base) {
			if i < len(st.outs) {
				st.outs[i] = outSlot{xy: xy, has: true}
			}
		}
		st.flags, st.flag = customFlags(c)
		return st
	}

	if c.rows == 0 { // the control: one uninterrupted belt
		sourceE(s, -5, c.base)
		beltsX(s, belt, &dirE, -3, 4, c.base)
		st.outs[0] = outSlot{xy: sink(s, 5, c.base), has: true}
		return st
	}

	// Parts first and belts after, so the belt-adjacency trigger is on the
	// critical path of every rig. TWO PER ROW: the west one carries the row's
	// input and the east one its output, because one tile carries one belt.
	for i := 0; i < c.rows; i++ {
		put(s, part, 0, c.base+i, nil, "")
		put(s, part, 1, c.base+i, nil, "")
	}
	for i := 1; i <= c.ins; i++ {
		buildInputRow(s, c.base+i-1, c.feedAt(i))
	}
	for i := 1; i <= c.outs; i++ {
		st.outs[i-1] = buildOutputRow(s, c.base+i-1, c.isBlocked(i))
	}
	for _, row := range c.prio {
		st.flags = append(st.flags, harness.XY{X: 1, Y: c.base + row - 1})
	}
	st.flag = harness.XY{X: 1, Y: c.base}
	if c.name == "pin" {
		// The WEST part, which carries an INPUT. Flagging it is what the planner
		// refuses at every size.
		st.flag = harness.XY{X: 0, Y: c.base}
	}
	return st
}

// customFlags is the same two answers for the rigs the uniform row arithmetic
// does not describe: two columns whose flagged part is the head output above
// them, and a lane rig whose parts start three rows into its band.
func customFlags(c *rigCfg) (flags []harness.XY, flag harness.XY) {
	switch c.name {
	case "pbig":
		// Nothing at create: this rig's whole subject is the toggle being
		// REFUSED on a standing network, which needs the network first.
		return nil, harness.XY{X: 0, Y: c.base - 1}
	case "pgrow":
		// TWO, and the toggle names the head one: the second flag is what makes
		// taking the first off a change that still does not fit, which is the
		// case the guard has to allow.
		flag = harness.XY{X: 0, Y: c.base - 1}
		return []harness.XY{flag, {X: 0, Y: c.base + 32}}, flag
	case "plane":
		flag = harness.XY{X: 1, Y: laneBase}
		return []harness.XY{flag}, flag
	}
	flag = harness.XY{X: 1, Y: c.base}
	return []harness.XY{flag}, flag
}

// ---------------------------------------------------------------------------
// the custom builders
// ---------------------------------------------------------------------------

// p32: 3 in, 2 out, saturated, with FINITE sources so that what each input was
// drawn of can be read afterwards. The third row's east part carries nothing,
// which is what makes it 3 -> 2 rather than 3 -> 3.
func buildP32(s fkapi.LuaSurface, base int) []harness.XY {
	p32Src = p32Src[:0]
	outs := make([]harness.XY, 0, 2)
	for i := 0; i <= 2; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 0; i <= 2; i++ {
		p32Src = append(p32Src, finiteSource(s, -5, base+i))
		beltsX(s, belt, &dirE, -3, -1, base+i)
	}
	for i := 0; i <= 1; i++ {
		beltsX(s, belt, &dirE, 2, 4, base+i)
		outs = append(outs, sink(s, 5, base+i))
	}
	return outs
}

var p32Src []harness.XY

// plane: `m2`'s lane rig with a priority port on it. Both inputs are
// SIDE-LOADED -- a belt joining a straight run from the north -- so each carries
// half a belt on ONE lane, and S = 1 with q = 1 means the priority port must
// come out at a FULL belt on BOTH lanes. Nothing but a per-lane sample can see
// that: a lane-preserving network delivers the same chest total.
func buildLane(s fkapi.LuaSurface, base int) []harness.XY {
	pb := base + 3
	laneBase = pb
	outs := make([]harness.XY, 0, 2)
	for i := 0; i <= 1; i++ {
		put(s, part, 0, pb+i, nil, "")
		put(s, part, 1, pb+i, nil, "")
	}
	// row pb: (-3,pb) is fed by nothing and exists only to keep (-2,pb) straight.
	source(s, -2, pb-3, dirS)
	beltsY(s, belt, &dirS, pb-1, pb-1, -2)
	beltsX(s, belt, &dirE, -3, -1, pb)
	// row pb+1: the same shape one column further out, so its feed column is
	// clear of row pb's chain.
	source(s, -4, pb-3, dirS)
	beltsY(s, belt, &dirS, pb-1, pb, -4)
	beltsX(s, belt, &dirE, -5, -1, pb+1)

	for i := 0; i <= 1; i++ {
		beltsX(s, belt, &dirE, 2, 4, pb+i)
		outs = append(outs, sink(s, 5, pb+i))
	}
	return outs
}

// The two column rigs. A single column of input parts, each carrying one
// east-facing belt on its west face, with an output part at each end: the
// cheapest shape that reaches a given P with two outputs, which is what a
// priority refusal needs -- flagging the ONLY output of an n -> 1 balancer is
// every output flagged, which collapses to the plain network and builds.
//
// THREE OF THE INPUTS ARE FED and the rest are inert belts. The shape is what
// decides P, and the rate only has to be non-zero for "still delivering across
// the attempt" to mean anything.
const bigFed = 3

func buildColumn(s fkapi.LuaSurface, base, ins int, spare bool) []harness.XY {
	outs := make([]harness.XY, 0, 3)
	// The head output: a part above the column with a north-facing belt off it.
	put(s, part, 0, base-1, nil, "")
	beltsY(s, belt, &dirN, base-2, base-4, 0)
	put(s, loader, 0, base-5, &dirN, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: base - 6})
	outs = append(outs, harness.XY{X: 0, Y: base - 6})

	for i := 0; i < ins; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, belt, -1, base+i, &dirE, "")
	}
	for i := 0; i < bigFed; i++ {
		sourceE(s, -3, base+i)
	}

	// The foot output: a part below the column with an EAST-facing belt off it,
	// so that the tile below stays free for the spare part.
	foot := base + ins
	put(s, part, 0, foot, nil, "")
	beltsX(s, belt, &dirE, 1, 3, foot)
	put(s, loader, 4, foot, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 5, Y: foot})
	outs = append(outs, harness.XY{X: 5, Y: foot})

	if spare {
		// A THIRD OUTPUT, because pgrow carries TWO flags: with M = 2 and both
		// of them ticked, `ShapeEdges` collapses the shape to the plain
		// butterfly (every output flagged is one tier, which is the network this
		// mod already builds) and there would be no priority network to refuse.
		put(s, part, 0, foot+1, nil, "")
		beltsX(s, belt, &dirE, 1, 3, foot+1)
		put(s, loader, 4, foot+1, &dirE, "input")
		harness.Place(s, harness.Piece{Name: "steel-chest", X: 5, Y: foot + 1})
		outs = append(outs, harness.XY{X: 5, Y: foot + 1})

		// EDGELESS, and it is the only tile in this column with a free face: the
		// belt that makes the balancer too big has to land somewhere legal, or
		// the one-belt-per-part rule would refuse it first and this would be a
		// test of a different bound.
		put(s, part, 0, foot+2, nil, "")
	}
	return outs
}

func buildBig(s fkapi.LuaSurface, base int) []harness.XY {
	return buildColumn(s, base, 33, false)
}

func buildGrow(s fkapi.LuaSurface, base int) []harness.XY {
	return buildColumn(s, base, 32, true)
}

// growTile is where the 33rd input belt lands, which takes pgrow from P = 32 to
// P = 64 and makes the two priority ports it is already carrying unbuildable.
func growTile() harness.XY {
	c, _ := rigByName("pgrow")
	return harness.XY{X: -1, Y: c.base + 32 + 2}
}

// ---------------------------------------------------------------------------
// the flag, through the mod's own door
// ---------------------------------------------------------------------------

func rigByName(name string) (*rigCfg, int) {
	for i := range rigs {
		if rigs[i].name == name {
			return &rigs[i], i
		}
	}
	return nil, -1
}

// setPrio asks the mod to move a part's flag and reports what it answered.
//
// The return value is the mod's own: `true` when the flag now reads what was
// asked, `false` for a refusal or a tile with no part on it. A refusal is a
// legitimate outcome that three of the rigs here exist to produce, so this
// reports rather than treating it as an error.
func setPrio(tag, rig string, k harness.XY, on bool) {
	s := harness.Surface(surf)
	idx, err := s.Index()
	if err != nil {
		harness.Fatal("reading the surface index", "for "+rig)
		return
	}
	v, st := fkapi.RemoteCall(modName, prioMethod,
		fkapi.OfNumber(float64(idx)),
		fkapi.OfNumber(float64(k.X)),
		fkapi.OfNumber(float64(k.Y)),
		fkapi.OfBool(on))
	ok := st == fkapi.StatusOK && v.Tag == fkapi.TagBool && v.Bool
	out.Open("flag t=").S(tag).S(" rig=").S(rig).
		S(" at=").I(int64(k.X)).S(",").I(int64(k.Y)).
		S(" want=").B(on).S(" accepted=").B(ok).End()
}

// variation is what the ENGINE has drawn on a part, which is where the flag
// lives (guest/go/skin.go): cells 1..47 are the shapes and 48..94 the same
// shapes badged. Reading it back out of the world is a stronger statement than
// the mod's own skin line, because a refused toggle writes no skin line at all
// and the question a refusal has to answer is whether the flag moved.
func variation(tag, rig string, k harness.XY) {
	s := harness.Surface(surf)
	v := int64(-1)
	if o, ok := harness.FindOnTile(s, part, k.X, k.Y); ok {
		if got, err := (fkapi.LuaEntity{Object: o}).GraphicsVariation(); err == nil && got != nil {
			v = int64(*got)
		}
	}
	out.Open("var t=").S(tag).S(" rig=").S(rig).
		S(" at=").I(int64(k.X)).S(",").I(int64(k.Y)).S(" v=").I(v).End()
}

// ---------------------------------------------------------------------------
// counting
// ---------------------------------------------------------------------------

// countAround is every item a teardown of the rig at `base` could be holding or
// could put on the floor: the ground and every transport line in a wide box
// around it, plus the WHOLE hidden surface.
//
// The two samples it is used for are taken in ONE TICK with an audit marker
// between them, so nothing anywhere else in the save can have moved -- every
// other rig's items are a constant that cancels. That is what makes counting
// the whole hidden surface right rather than merely safe: the network coming
// down is somewhere in it and no slot arithmetic is needed to say where.
func countAround(base int) (ground, lines int64) {
	s := harness.Surface(surf)
	box := harness.Box(-20, float64(base-14), 20, float64(base+16))
	for _, e := range harness.EntitiesInOfType(s, box, "item-entity") {
		if _, n, ok := harness.GroundStack(e); ok {
			ground += n
		}
	}
	for _, e := range harness.EntitiesIn(s, box, "") {
		lines += harness.TransportLineItems(e)
	}
	if hid, ok := harness.SurfaceIfAny(hidden); ok {
		for _, e := range harness.EntitiesInOfType(hid, harness.Box(-16, -16, 2200, 400), "item-entity") {
			if _, n, ok := harness.GroundStack(e); ok {
				ground += n
			}
		}
		for _, e := range harness.EntitiesIn(hid, harness.Box(-16, -16, 2200, 400), "") {
			lines += harness.TransportLineItems(e)
		}
	}
	return ground, lines
}

// toggleCheck is the whole of what a priority change costs in items: count, move
// the flag, drain the queue synchronously, count again.
//
// THE AUDIT MARKER IS WHAT MAKES IT ONE SAMPLE. A toggle queues the cluster and
// asks for the next tick's flush, so a count taken a tick later would include
// everything every belt in the save moved in between. `bbb-audit` is the
// shipped synchronous "drain and re-classify now" trigger and it runs inside
// this dispatch, so the two counts are one atomic pair and the difference can
// only be the teardown.
func toggleCheck(tag, rig string, on bool) {
	c, i := rigByName(rig)
	if c == nil {
		return
	}
	gb, lb := countAround(c.base)
	setPrio(tag, rig, built[i].flag, on)
	harness.Audit(harness.Surface(surf), -20, c.base)
	ga, la := countAround(c.base)
	out.Open("items t=").S(tag).S(" rig=").S(rig).
		S(" ground ").I(gb).S("->").I(ga).
		S(" lines ").I(lb).S("->").I(la).
		S(" total ").I(gb + lb).S("->").I(ga + la).End()
	variation(tag, rig, built[i].flag)
}

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

func report(tag string) {
	s := harness.Surface(surf)
	for i := range rigs {
		out.Open("t=").S(tag).S(" rig=").S(rigs[i].name).S(" out=[")
		for j, slot := range built[i].outs {
			if j > 0 {
				out.S(" ")
			}
			if !slot.has {
				// A blocked output has no chest. -1 is the estate's own "there
				// is none here", so that a missing sink cannot read as nothing
				// delivered.
				out.I(-1)
				continue
			}
			out.I(harness.ChestCount(s, "steel-chest", slot.xy.X, slot.xy.Y))
		}
		out.S("]").End()
	}
	if len(p32Src) > 0 {
		out.Open("draw t=").S(tag).S(" rig=p32 in=[")
		for i, k := range p32Src {
			if i > 0 {
				out.S(" ")
			}
			out.I(harness.ChestCount(s, "steel-chest", k.X, k.Y))
		}
		out.S("]").End()
	}
}

// sampleLanes is `m2`'s per-lane reading pointed at the priority port. A lane
// total is the one thing a chest cannot see: two half-belts arriving on one lane
// each have to come out as one belt on BOTH lanes, and a network that preserved
// lanes would fill the same chest at the same rate with every item on one side.
func sampleLanes(tick uint64) {
	if laneBase == 0 {
		return
	}
	s := harness.Surface(surf)
	for row := 0; row <= 1; row++ {
		var l1, l2 int64
		for x := 2; x <= 4; x++ {
			b, ok := harness.FindAt(s, x, laneBase+row, "", "transport-belt")
			if !ok {
				continue
			}
			e := fkapi.LuaEntity{Object: b}
			if line, err := e.GetTransportLine(fkapi.DefinesTransportLineLeftLine()); err == nil {
				if c, err := (fkapi.LuaTransportLine{Object: line}).GetItemCount(nil); err == nil {
					l1 += int64(c)
				}
			}
			if line, err := e.GetTransportLine(fkapi.DefinesTransportLineRightLine()); err == nil {
				if c, err := (fkapi.LuaTransportLine{Object: line}).GetItemCount(nil); err == nil {
					l2 += int64(c)
				}
			}
		}
		out.Open("lane t=").U(tick).S(" out=").I(int64(row + 1)).
			S(" left=").I(l1).S(" right=").I(l2).End()
	}
}

func auditNow(y int) { harness.Audit(harness.Surface(surf), -20, y) }

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

// grow lays the 33rd input belt on pgrow's spare part. The cluster is carrying a
// priority port already, so the shape the build asks for is P = 64 with a
// priority output -- which compile() refuses in front of its own teardown, and
// the network that is standing goes on running.
func grow() {
	k := growTile()
	put(harness.Surface(surf), belt, k.X, k.Y, &dirE, "")
	out.Open("grew pgrow: a 33rd input belt at ").I(int64(k.X)).S(",").I(int64(k.Y)).End()
}

// openAndCut is what makes the spill guard's second half possible: the dead
// ends become sinks and the sources are taken away, so the rig drains instead of
// settling at whatever a flowing network holds.
//
// THE SOURCE LOADER IS WHAT GOES, not the input belt. A loader at x = -4 is four
// tiles from the nearest part and outside the mod's two-tile neighbour gate, so
// cutting the feed there re-classifies nothing; destroying the belt at x = -1
// would be an edge edit and would recompile the very network this is about to
// ask a question of.
func openAndCut(rig string) {
	c, _ := rigByName(rig)
	s := harness.Surface(surf)
	for i := 0; i < c.rows; i++ {
		y := c.base + i
		if o, ok := harness.FindOnTile(s, loader, -4, y); ok {
			harness.Destroy(o, false)
		}
	}
	for i := 1; i <= c.outs; i++ {
		sink(s, 5, c.base+i-1)
	}
	out.Open("opened ").S(rig).S(": the outputs are sinks and the sources are gone").End()
}

// onBelts is what a rig's OWN visible belts are carrying, which is the term that
// has to come off a drained total before it describes the MACHINE.
//
// The interfaces standing on the part tiles are deliberately not counted: they
// are the compiler's and `plan.Capacity` counts them, so they belong on the
// machine's side of the subtraction. What is counted is the player's own run in
// and the player's own run out.
func onBelts(rig string) int64 {
	c, _ := rigByName(rig)
	s := harness.Surface(surf)
	total := int64(0)
	for i := 0; i < c.rows; i++ {
		y := c.base + i
		for x := -3; x <= 4; x++ {
			if x >= 0 && x <= 1 {
				continue // the part tiles
			}
			if b, ok := harness.FindAt(s, x, y, "", "transport-belt"); ok {
				total += harness.TransportLineItems(b)
			}
		}
	}
	return total
}

// measured is the four dead-ended rigs, in the order they are reported.
var measured = [...]string{"mp22", "mq22", "mp44", "pbnd"}

func measureBelts() {
	for _, r := range measured {
		out.Open("belts rig=").S(r).S(" items=").I(onBelts(r)).End()
	}
}

func measureDrained() {
	s := harness.Surface(surf)
	for _, r := range measured {
		c, i := rigByName(r)
		total := int64(0)
		for _, slot := range built[i].outs {
			if slot.has {
				total += harness.ChestCount(s, "steel-chest", slot.xy.X, slot.xy.Y)
			}
		}
		out.Open("drained rig=").S(r).S(" rows=").I(int64(c.rows)).
			S(" chests=").I(total).End()
	}
}

// pasteSettings is shift-click copy between two parts, which is the third door
// onto the flag (guest/go/priority.go, `onSettingsPasted`) and the only one that
// can put a flag on a part the player never pointed at.
//
// THE DESTINATION IS A BALANCER THAT CANNOT CARRY ONE. pbig is 33 -> 2, which is
// P = 64, and the priority construction wants 35 columns of a 32-column slot --
// so the handler asks the same pre-teardown check a keypress asks, is refused,
// and the flag must not move. What the ENGINE has already done by then is the
// interesting half: `copy_settings` moves `graphics_variation` itself, so the
// destination is wearing the source's badged picture while its flag is clear.
func pasteSettings(srcRig, dstRig string) {
	s := harness.Surface(surf)
	_, si := rigByName(srcRig)
	src, ok := harness.FindOnTile(s, part, built[si].flag.X, built[si].flag.Y)
	if !ok {
		harness.Fatal("finding the paste source", srcRig)
		return
	}
	dk := pasteTile()
	dst, ok := harness.FindOnTile(s, part, dk.X, dk.Y)
	if !ok {
		harness.Fatal("finding the paste destination", dstRig)
		return
	}
	if _, err := (fkapi.LuaEntity{Object: dst}).CopySettings(src, nil); err != nil {
		harness.Fatal("copy_settings", dstRig)
		return
	}
	out.Open("pasted ").S(srcRig).S("'s settings onto ").S(dstRig).
		S(" at ").I(int64(dk.X)).S(",").I(int64(dk.Y)).End()
}

// pasteTile is pbig's FOOT output part, which is a part of a balancer no
// priority port fits.
func pasteTile() harness.XY {
	c, _ := rigByName("pbig")
	return harness.XY{X: 0, Y: c.base + 33}
}

// nudge queues a cluster without changing its edge list: a belt TWO tiles from a
// part is inside the mod's neighbour gate and adjacent to nothing, so the
// cluster is re-classified and its fingerprint does not move.
//
// It is what makes the paste leg a question about the next COMPILE rather than
// about the tick the paste landed in: a refused paste queues nothing, so without
// this the stale picture the engine wrote would sit there until an audit
// happened to look.
func nudge() {
	c, _ := rigByName("pbig")
	put(harness.Surface(surf), belt, -2, c.base+10, &dirE, "")
	out.Open("nudged pbig: a belt two tiles from a part, which is an edge of nothing").End()
}

// ---------------------------------------------------------------------------
// the boundary poll
// ---------------------------------------------------------------------------
//
// pbnd's clear is asked over and over while the rig drains, because the tick the
// guard lets go on is the tick the machine crossed under the successor's
// capacity -- and that capacity is the mod's own arithmetic, which nothing here
// writes down. Polling is what turns "just under" into a question the guest
// answers rather than a constant this file would have to keep in step.
var (
	bndFrom uint64
	bndDone bool
	bndTry  int
)

func pollBoundary(tick uint64) {
	if bndDone || bndFrom == 0 || tick < bndFrom || tick > bndFrom+400 {
		return
	}
	if (tick-bndFrom)%40 != 0 {
		return
	}
	bndTry++
	c, i := rigByName("pbnd")
	gb, lb := countAround(c.base)
	s := harness.Surface(surf)
	idx, err := s.Index()
	if err != nil {
		return
	}
	v, st := fkapi.RemoteCall(modName, prioMethod,
		fkapi.OfNumber(float64(idx)),
		fkapi.OfNumber(float64(built[i].flag.X)),
		fkapi.OfNumber(float64(built[i].flag.Y)),
		fkapi.OfBool(false))
	ok := st == fkapi.StatusOK && v.Tag == fkapi.TagBool && v.Bool
	out.Open("boundary try=").I(int64(bndTry)).S(" tick=").U(tick).
		S(" accepted=").B(ok).End()
	if !ok {
		return
	}
	// Accepted, so the flush that follows is the recompile -- and the count
	// either side of the marker is the one atomic pair that can say whether it
	// spilled.
	bndDone = true
	harness.Audit(s, -20, c.base)
	ga, la := countAround(c.base)
	out.Open("items t=boundary rig=pbnd ground ").I(gb).S("->").I(ga).
		S(" lines ").I(lb).S("->").I(la).
		S(" total ").I(gb + lb).S("->").I(ga + la).End()
	variation("boundary", "pbnd", built[i].flag)
}

var schedule = []harness.Step{
	{Tick: 1800, Do: func() { report("t1") }},
	{Tick: 1900, Do: func() { sampleLanes(1900) }},
	{Tick: 2300, Do: func() { sampleLanes(2300) }},
	{Tick: 2700, Do: func() { sampleLanes(2700) }},
	{Tick: 3100, Do: func() { sampleLanes(3100) }},
	{Tick: 3500, Do: func() { sampleLanes(3500) }},
	{Tick: 3540, Do: func() { report("t2") }},
	// The steady-state audit: every rig has been standing untouched since the
	// save was written, so the world and the registry must agree exactly.
	{Tick: 3560, Do: func() { auditNow(0) }},

	// --- the toggle, on a running balancer ----------------------------------
	//
	// AFTER EVERY RATE ABOVE HAS REPORTED, which is `m2`'s rule for its own
	// setting band: no figure in the main window is taken over a save whose
	// wiring has moved.
	{Tick: 3600, Do: func() { toggleCheck("tog-on", "ptog", true) }},
	{Tick: 3620, Do: func() { report("tog-on-a") }},
	{Tick: 4020, Do: func() { report("tog-on-b") }},
	{Tick: 4040, Do: func() { toggleCheck("tog-off", "ptog", false) }},
	{Tick: 4060, Do: func() { report("tog-off-a") }},
	{Tick: 4460, Do: func() { report("tog-off-b") }},

	// --- the refusals -------------------------------------------------------
	{Tick: 4500, Do: func() { report("ref-pre") }},
	{Tick: 4520, Do: func() {
		_, i := rigByName("pin")
		setPrio("ref-in", "pin", built[i].flag, true)
		variation("ref-in", "pin", built[i].flag)
	}},
	{Tick: 4530, Do: func() {
		_, i := rigByName("pbig")
		setPrio("ref-big", "pbig", built[i].flag, true)
		variation("ref-big", "pbig", built[i].flag)
	}},
	// The paste, then a queueing edit, then the audit: three ticks apart so that
	// the compile the paste did NOT ask for has somewhere to happen before the
	// audit looks.
	{Tick: 4540, Do: func() { pasteSettings("p2sat", "pbig") }},
	{Tick: 4550, Do: nudge},
	{Tick: 4560, Do: func() { auditNow(0) }},
	{Tick: 4570, Do: func() { variation("paste", "pbig", pasteTile()) }},
	{Tick: 4580, Do: grow},
	{Tick: 4600, Do: func() { auditNow(0) }},
	{Tick: 4620, Do: func() { report("ref-a") }},
	{Tick: 5020, Do: func() { report("ref-b") }},
	// TAKING A FLAG OFF A BALANCER THAT ALREADY DOES NOT FIT. pgrow is carrying
	// two priority ports over a shape that a build has just taken to P = 64, so
	// neither the shape it has nor the shape one fewer flag would give can be
	// built -- and the change has to be allowed anyway, because refusing it
	// leaves the player holding a machine they cannot un-break.
	{Tick: 5040, Do: func() {
		_, i := rigByName("pgrow")
		setPrio("unflag", "pgrow", built[i].flag, false)
		variation("unflag", "pgrow", built[i].flag)
	}},
	{Tick: 5060, Do: func() { auditNow(0) }},

	// --- the spill guard ----------------------------------------------------
	//
	// pfull has been dead-ended and fed hard since tick 0, so by now every belt
	// and every splitter in it is stationary and it is carrying more than the
	// plain network it would become could hold.
	{Tick: 5100, Do: func() { toggleCheck("spill-refused", "pfull", false) }},
	{Tick: 5120, Do: func() { auditNow(0) }},
	{Tick: 5140, Do: func() { openAndCut("pfull") }},
	// Long enough for four express belts to empty a stopped network into four
	// chests with nothing left feeding it.
	{Tick: 5660, Do: func() { toggleCheck("spill-cleared", "pfull", false) }},
	{Tick: 5680, Do: func() { report("spill-after") }},

	// --- the boundary, and the three holds ----------------------------------
	{Tick: 5700, Do: measureBelts},
	{Tick: 5720, Do: func() {
		for _, r := range measured {
			openAndCut(r)
		}
		bndFrom = 5760
	}},
	{Tick: 6300, Do: measureDrained},
	{Tick: 6340, Do: func() { auditNow(0) }},
	{Tick: 6360, Do: func() { report("final") }},
}

//go:wasmexport fk_on_init
func onInit() {
	makeSurface()
	built = make([]rigState, len(rigs))
	for i := range rigs {
		built[i] = buildRig(&rigs[i])
	}
	// THE FLAGS BEFORE THE AUDIT, so that every network in the save is compiled
	// ONCE, as the priority network it is meant to be. A toggle only queues the
	// cluster and asks for a flush; a `--create` never reaches a tick, so the
	// marker below is the only drain there is and it does the compiling too.
	for i := range rigs {
		for _, k := range built[i].flags {
			setPrio("create", rigs[i].name, k, true)
		}
	}
	auditNow(0)
	// EVERY rig's flagged part is read back out of the world, and so are the
	// three whose flag must still be down: a variation is where the flag lives,
	// so this is the create phase saying what it built rather than what it asked
	// for.
	for i := range rigs {
		if rigs[i].name == "ctrl" {
			continue
		}
		variation("create", rigs[i].name, built[i].flag)
	}
	out.Open("init complete: ").I(int64(len(rigs))).S(" rigs").End()
	report("create")
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	tick := fkapi.ReadOnTick(ptr).Tick
	harness.Run(schedule, tick)
	pollBoundary(tick)
}

func main() {}
