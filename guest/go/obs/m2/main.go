// Command bbb-m2-test drives the network compiler and reports what came out.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-m2-test/control.lua` was, rig for rig, tick for tick and log
// line for log line, with its little data stage compiled too (obs/m2data). See
// agents/estate-port.md.
//
// It ASSERTS NOTHING. It builds rigs, samples the output chests at two ticks and
// logs the numbers; test/assert-m2.py decides whether they are right. An
// observer that computed the expected balance would be a second implementation
// of the thing under test.
//
// # The rule every rig is built to
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
// Every edge of a cluster is an interface linked belt standing on the cluster's
// own tile, so a part carrying an input on its west side and an output on its
// east carried TWO belt-connectables on one tile -- which is what 2.1's
// collision validator forbids. See agents/single-edge.md and guest/go/sedge.go.
//
// What that costs a rig is GEOMETRY AND NOTHING ELSE. Every column of parts
// becomes TWO columns: a west part carrying the input and an east part carrying
// the output, so `sat4` is eight parts rather than four and `lio` is two rather
// than one. The MACHINE does not change -- same N, same M, same
// `P = next_pow2(max(N, M))`, same butterfly, same rate -- which is exactly what
// assert-m2.py's port-count block is there to say before it looks at a number.
//
// The layout every uniform rig uses, per row:
//
//	x=-5 source chest   -4 loader   -3..-1 belts   0 WEST PART   1 EAST PART
//	x=2..4 belts        5 sink loader              6 chest
//
// # The rigs, one per y band on a flat scratch surface
//
//	ctrl     a bare express belt, chest to chest. The yardstick: whatever this
//	         delivers in the sample window is what one saturated belt is worth,
//	         so "full throughput" is a comparison against the engine rather than
//	         against a number someone worked out on paper.
//	sat4     4 belts in, 4 belts out over EIGHT parts (a 4x2 block), saturated
//	sat8     the same at 8 -- sixteen parts -- which needs three butterfly
//	         stages and two jumper blocks rather than two and one
//	a3to5    3 in, 5 out: N != M, and P=8 with loopbacks on the spare ports
//	a4to1    4 in, 1 out: the other asymmetry, where spare OUTPUT ports have to
//	         dead-end because there are no spare input ports to loop them into
//	starve   4 in, 4 out, but only ONE input has a source. This is the case that
//	         kills every chest-based design (Techrocket9 measured one output
//	         draining >9,000 items while its peers got ~80)
//	block    4 in, 4 out, but the fourth output has nowhere to go
//	regrow   3 in, 4 out; a fourth input belt is added at tick 900, under load
//	xsurf    sat4 again on a SECOND surface, because the network lives on a third
//	         one and linked belts are the only thing joining them
//
// The SHAPE band, added because the six shapes above are six of the sixty-four
// (n, m) pairs with n, m <= 8 and the pure-Go fixed-point model
// (plan.PropagateLoop) is exact only for n <= m -- everything with dead-ended
// spare outputs can only be covered in a real Factorio:
//
//	sq3      3 in, 3 out. P=4, Loop=1: the smallest square shape that is not a
//	         power of two, and probably the most common balancer anyone builds
//	a2to3    2 in, 3 out. P=4, Loop=1; each output gets 2/3 of a belt
//	a5to3    5 in, 3 out. P=8, Loop=3 and TWO DEAD-ENDED output ports -- the
//	         blocking regime no linear model can express
//	n9m9     9 in, 9 out. P=16: the first FOUR-stage butterfly, three jumper
//	         blocks, ever built in a real game
//	fdbk     a literal feedback loop: a third output belt curls round through the
//	         world and comes back into the cluster's SOUTH face, so the machine
//	         sees 3 in / 3 out and one of each is itself
//	tslow    4 in, 4 out, but one OUTPUT ROW is a normal-tier belt. A
//	         rate-LIMITED port rather than a fully blocked one
//	lane     the lane-fidelity rig. Both inputs are SIDE-LOADED, so each is half
//	         a belt on ONE lane; chest totals cannot see the difference, so this
//	         one is asserted on per-lane occupancy at the outputs
//
// The EDGE-TYPE band. classifySide keys on the entity's `type` and names six of
// them; until these rigs existed, only "transport-belt" had ever been run:
//
//	uio      2->2 through UNDERGROUND ends placed directly against the parts,
//	         both arms of the belt_to_ground_type branch
//	spio     2->2 fed and drained by vanilla express SPLITTERS whose faces span
//	         both parts, so each half is its own edge
//	lio      1->1 through LOADERS directly against the parts -- and the first
//	         1->1 (P=1, five-entity) flow rig in any suite. TWO parts under the
//	         rule, which is the smallest balancer that can exist at all
//	lsio     2->2 through LANE SPLITTERS against the parts. Base ships the
//	         `lane-splitter` TYPE and not one buildable entity of it, so the data
//	         stage clones the mod's own hidden one to have anything to place
//	pass     the NEGATIVE: a belt line running PAST the cluster's north face,
//	         perpendicular to it, which must not be classified as an edge and
//	         must not have anything stolen from it. Under the one-belt rule it
//	         has teeth it did not have before: both top parts already carry their
//	         one belt, so a classifier that read the passing line as an edge
//	         would take them to TWO and the whole cluster would be REFUSED
//
// Plus two measurements that are not rigs:
//   - a profiler around a forced full recompile of sat4 and sat8
//   - an item-conservation check around a recompile of a network that is FULL:
//     sample, recompile, sample again, all inside one tick so nothing moves for
//     any other reason.
//
// # Forcing the flush
//
// The guest batches: a build or mine event updates its registry inside the event
// and defers the recompile to the next tick (`fk.defer`), so a measurement taken
// in the tick that laid the belt would see nothing at all. `bbb-audit` -- a
// shipped marker prototype whose whole purpose is "re-classify and repair
// everything, now" -- is the synchronous escape hatch, and it is what both
// measurements below use. That is also why on_init ends with one: `--create`
// never reaches a tick, so without it every network in the save would be
// compiled on the first tick of the BENCHMARK instead.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

var out = harness.Line{Tag: "[BBB-M2] "}

const (
	part     = "bbb-balancer-part"
	belt     = "express-transport-belt"
	slowBelt = "transport-belt" // normal tier: exactly 1/3 of express
	under    = "express-underground-belt"
	split    = "express-splitter"
	lsplit   = protos.M2LaneSplitter // base has no buildable lane splitter
	loader   = protos.M2Loader

	surfA  = "bbb-m2-a"
	surfB  = "bbb-m2-b"
	hidden = "bbb-hidden"

	pitch = 12 // rows between rigs
	halfX = 16 // how far either side of x=0 the scratch surface is cleared
)

// The four cardinal directions, resolved once at load. A define's NUMBER is
// Factorio's own and is not stable across versions, so it is asked for by name
// rather than written down -- the shipped guest takes the same reading.
var dirN, dirE, dirS, dirW uint32

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirN = fkapi.DefinesDirectionNorth()
	dirE = fkapi.DefinesDirectionEast()
	dirS = fkapi.DefinesDirectionSouth()
	dirW = fkapi.DefinesDirectionWest()
	layOutBands()
}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

// put is the estate's `put()`: one create_entity at a tile centre, on the player
// force, with `raise_built = true` so the mod under test sees it. Fatal when
// nothing comes back, because a rig that did not land makes every number after
// it a measurement of a different world.
func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true,
	})
}

// putAt is put() for the one thing in this suite whose position is not a tile
// centre: a two-tile splitter facing east sits on the boundary BETWEEN its two
// rows, so its y is an integer and not an integer-plus-a-half.
func putAt(s fkapi.LuaSurface, name string, px, py float64, dir *uint32) fkapi.Object {
	pos := fkapi.MapPosition{X: px, Y: py}
	return harness.Place(s, harness.Piece{
		Name: name, X: int(px), Y: int(py), Pos: &pos, Dir: dir, Raise: true,
	})
}

// belts lays a run of belts, INCLUSIVE, all facing the same way, along x or y.
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

// source is an infinity chest of iron plate and the loader that drains it, and
// `dir` is the direction the ITEMS travel. East is every rig but `lane`, whose
// feeds come down from the north.
//
// THE CHEST IS PLACED WITHOUT `raise_built` and the loader with it, exactly as
// the Lua had them: the mod under test has to see the loader, which is an edge
// candidate standing next to a part, and has no interest in a chest.
func source(s fkapi.LuaSurface, x, y int, dir uint32) fkapi.Object {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	harness.InfinityFilter(c, "iron-plate", "", 1000)
	if dir == dirS {
		put(s, loader, x, y+1, &dirS, "output")
	} else {
		put(s, loader, x+1, y, &dirE, "output")
	}
	return c
}

func sourceE(s fkapi.LuaSurface, x, y int) fkapi.Object { return source(s, x, y, dirE) }

// sink is the loader that fills a steel chest, and it returns the CHEST TILE
// rather than the chest: a rig registry holds tiles, so that what it keeps goes
// on being true across the save between `--create` and `--benchmark`.
func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	put(s, loader, x, y, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

// ---------------------------------------------------------------------------
// the rigs
// ---------------------------------------------------------------------------

// rigCfg is one row of the estate's `RIGS` table.
//
// Every rig gets a y band of `span` rows, defaulting to pitch, and the bands are
// laid out in table order. The first nine all take the default, so their bases
// are the (i-1)*pitch they have always been; a rig that needs more room than one
// pitch says so rather than silently overlapping its neighbour.
type rigCfg struct {
	name string

	// rows, ins and outs drive the uniform builder. rows == 0 means either the
	// control (one uninterrupted belt) or a custom `build`.
	rows, ins, outs int

	// span overrides pitch. Only `n9m9` needs one.
	span int

	// fed lists the 1-BASED input rows that get a source; nil means all of them.
	// `starve` is the only rig that names any.
	fed []int

	// blocked lists the 1-based output rows with nowhere to go.
	blocked []int

	// growTo is the input row `regrow` adds at tick 900.
	growTo int

	// otherSurface puts the rig on surface b.
	otherSurface bool

	// build names a custom builder for the shapes the uniform one cannot express
	// -- a feedback loop, side-loaded inputs, undergrounds, splitters, loaders.
	build func(s fkapi.LuaSurface, base int) []harness.XY

	// base is filled in by layOutBands.
	base int
}

func (c *rigCfg) isFed(i int) bool {
	if c.fed == nil {
		return true
	}
	for _, f := range c.fed {
		if f == i {
			return true
		}
	}
	return false
}

func (c *rigCfg) isBlocked(i int) bool {
	for _, b := range c.blocked {
		if b == i {
			return true
		}
	}
	return false
}

// nOut is how many output slots this rig REPORTS, which is `cfg.outs or 1`:
// `block` leaves a hole and the slot has to be explicit rather than short.
func (c *rigCfg) nOut() int {
	if c.outs == 0 {
		return 1
	}
	return c.outs
}

var rigs = []rigCfg{
	{name: "ctrl"},
	{name: "sat4", rows: 4, ins: 4, outs: 4},
	{name: "sat8", rows: 8, ins: 8, outs: 8},
	{name: "a3to5", rows: 5, ins: 3, outs: 5},
	{name: "a4to1", rows: 4, ins: 4, outs: 1},
	{name: "starve", rows: 4, ins: 4, outs: 4, fed: []int{1}},
	{name: "block", rows: 4, ins: 4, outs: 4, blocked: []int{4}},
	{name: "regrow", rows: 4, ins: 3, outs: 4, growTo: 4},
	{name: "xsurf", rows: 4, ins: 4, outs: 4, otherSurface: true},

	// the shape band
	{name: "sq3", rows: 3, ins: 3, outs: 3},
	{name: "a2to3", rows: 3, ins: 2, outs: 3},
	{name: "a5to3", rows: 5, ins: 5, outs: 3},
	{name: "n9m9", rows: 9, ins: 9, outs: 9, span: 16},
	{name: "tslow", outs: 4, build: buildTslow},
	{name: "fdbk", outs: 2, build: buildFdbk},
	{name: "lane", outs: 2, build: buildLane},

	// the edge-type band
	{name: "uio", outs: 2, build: buildUio},
	{name: "spio", outs: 2, build: buildSpio},
	{name: "lio", outs: 1, build: buildLio},
	{name: "lsio", outs: 2, build: buildLsio},
	{name: "pass", outs: 3, build: buildPass},
}

// The state a rig keeps once it is built. TILES and not entities: everything
// here is written in `fk_on_init` during `--create` and read during
// `--benchmark`, so it crosses a save.
type rigState struct {
	// outs is one entry per REPORTED output slot, in the order assert-m2.py
	// reads them. A slot with has == false is `block`'s hole and reports -1.
	outs []outSlot
}

type outSlot struct {
	xy  harness.XY
	has bool
}

var (
	built    []rigState // parallel to rigs
	lossBase int
	laneBase int
	rowsA    int
	rowsB    int
)

// layOutBands assigns every rig its base and works out how tall each surface has
// to be. It runs in `init` rather than in `fk_on_init` because nothing in it
// touches the world and two later readers -- the schedule and the report -- want
// the numbers.
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
	// The loss rig sits two pitches clear of the last band. That gap is not
	// decoration: lossArea reaches 14 rows ABOVE lossBase, and a rig inside it
	// would put its own items into a count that is supposed to describe one
	// teardown.
	lossBase = base + 2*pitch
	rowsA = lossBase + 2*pitch
	// Surface b carries `xsurf` alone and does not need the other rigs' rows.
	for i := range rigs {
		if rigs[i].otherSurface {
			rowsB = rigs[i].base + 2*pitch
		}
	}
}

// chunkRadius is the Lua's `math.ceil(rows / 32) + 3`.
func chunkRadius(rows int) uint32 { return uint32((rows+31)/32) + 3 }

func makeSurface(name string, rows int) fkapi.LuaSurface {
	return harness.Flat{
		Name:        name,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		ChunkRadius: chunkRadius(rows),
		X0:          -halfX,
		Y0:          -8,
		X1:          halfX,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()
}

func surfaceOf(c *rigCfg) fkapi.LuaSurface {
	if c.otherSurface {
		return harness.Surface(surfB)
	}
	return harness.Surface(surfA)
}

func buildInputRow(s fkapi.LuaSurface, y int, fed bool) {
	if fed {
		sourceE(s, -5, y)
	}
	beltsX(s, belt, &dirE, -3, -1, y)
}

func buildOutputRow(s fkapi.LuaSurface, y int, blocked bool) outSlot {
	beltsX(s, belt, &dirE, 2, 4, y)
	if blocked {
		return outSlot{}
	}
	return outSlot{xy: sink(s, 5, y), has: true}
}

func buildRig(c *rigCfg) rigState {
	s := surfaceOf(c)
	st := rigState{outs: make([]outSlot, c.nOut())}

	if c.build != nil {
		for i, xy := range c.build(s, c.base) {
			if i < len(st.outs) {
				st.outs[i] = outSlot{xy: xy, has: true}
			}
		}
		return st
	}

	if c.rows == 0 { // the control: one uninterrupted belt
		sourceE(s, -5, c.base)
		beltsX(s, belt, &dirE, -3, 4, c.base)
		st.outs[0] = outSlot{xy: sink(s, 5, c.base), has: true}
		return st
	}

	// Parts FIRST, belts after, so that the belt events are what drive the
	// compiles. Building the belts first would work too and would compile once;
	// this way the belt-adjacency trigger is on the critical path of every rig.
	//
	// TWO PER ROW: the west one carries the row's input and the east one its
	// output, because one tile may carry only one belt.
	for i := 0; i < c.rows; i++ {
		put(s, part, 0, c.base+i, nil, "")
		put(s, part, 1, c.base+i, nil, "")
	}
	for i := 1; i <= c.ins; i++ {
		buildInputRow(s, c.base+i-1, c.isFed(i))
	}
	for i := 1; i <= c.outs; i++ {
		st.outs[i-1] = buildOutputRow(s, c.base+i-1, c.isBlocked(i))
	}
	return st
}

// ---------------------------------------------------------------------------
// the custom builders
//
// Every one of them places its PARTS first and its belts after, for the reason
// buildRig gives: the belt-adjacency trigger is then on the critical path of
// every rig rather than only of the ones the uniform builder makes.
// ---------------------------------------------------------------------------

// tslow: 4 in, 4 out, all express EXCEPT the last output row, which is a
// normal-tier belt -- exactly a third of express. A RATE-LIMITED port, which is
// a different question from `block`'s fully dead one: the three express outputs
// must stay at a full belt each while the fourth trickles, and the balancer must
// not throttle itself to the slow port's rate. The sink loader stays express so
// the belt is the only limiter.
func buildTslow(s fkapi.LuaSurface, base int) []harness.XY {
	out := make([]harness.XY, 4)
	for i := 0; i <= 3; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 0; i <= 3; i++ {
		sourceE(s, -5, base+i)
		beltsX(s, belt, &dirE, -3, -1, base+i)
	}
	for i := 0; i <= 2; i++ {
		beltsX(s, belt, &dirE, 2, 4, base+i)
		out[i] = sink(s, 5, base+i)
	}
	beltsX(s, slowBelt, &dirE, 2, 4, base+3)
	out[3] = sink(s, 5, base+3)
	return out
}

// fdbk: a literal feedback loop.
//
//	row base+2  ->[W][E]->  sink               real in/out
//	row base+3  ->[W][E]->  sink               real in/out
//	row base+4    [W][E]-> -> -> -> +          the loop's own output
//	row base+5     ^ <- <- <- <- <- +          the return run
//
// The machine sees 3 in and 3 out, and one of each is itself. In steady state
// the loop carries L, every output carries (2+L)/3, and the loop's output IS its
// input, so L = 1: each real output ends up at exactly one belt and the two of
// them together at exactly the two belts that went in. The interesting part is
// that the loop is a physical belt in the world, so it fills, and a network that
// jams instead of settling shows up as a rate collapse.
//
// THE RETURN COMES IN FROM THE SOUTH, and under the one-belt rule that is forced
// rather than chosen: the loop's own west part is the only tile in the cluster
// with a free face, because every other one already carries its one belt. It
// runs UNDER the block rather than over it for the same reason -- the north
// faces of row base+2 belong to nothing and must keep belonging to nothing, and
// a return run laid there would be one tile from two parts.
//
// No tile of the return run is orthogonally adjacent to a part except
// (0, base+5), which is the intended input: the westward run along row base+5
// passes under the east column, but a WEST-facing belt on a part's south face is
// neither `dir` nor `back` from that side and falls through classifySide,
// exactly as `pass` does from the north.
func buildFdbk(s fkapi.LuaSurface, base int) []harness.XY {
	out := make([]harness.XY, 0, 2)
	for i := 2; i <= 4; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 2; i <= 3; i++ {
		sourceE(s, -5, base+i)
		beltsX(s, belt, &dirE, -3, -1, base+i)
		beltsX(s, belt, &dirE, 2, 4, base+i)
		out = append(out, sink(s, 5, base+i))
	}
	// east off the bottom row, south at column 6, west underneath the block, and
	// north into the bottom-left part's south face. Every turn is a CURVE (fed
	// from one side with nothing behind it), so the loop carries both lanes at
	// full rate and cannot be the thing that limits it.
	beltsX(s, belt, &dirE, 2, 5, base+4)
	put(s, belt, 6, base+4, &dirS, "")
	beltsX(s, belt, &dirW, 6, 1, base+5)
	put(s, belt, 0, base+5, &dirN, "")
	return out
}

// lane: the lane-fidelity rig, and the only one in any suite that chest totals
// cannot judge.
//
// Both inputs are fed by SIDE-LOADING and by nothing else -- a belt joining from
// the NORTH -- so each input row carries exactly half a belt, all of it on the
// same (left) lane. A vanilla splitter is lane-PRESERVING (spike S1), so a
// network without the lane-splitter stage would deliver every one of those items
// on one lane of every output and the chest totals would be IDENTICAL. What
// separates the two is per-lane occupancy, which sampleLanes reads.
//
// THE DEAD BELT BEHIND EACH TARGET IS LOAD-BEARING AND IS NOT SCENERY. A belt
// whose ONLY input is from a side is a CURVE, and a curve carries both lanes at
// full rate -- which would feed the rig a whole belt across both lanes and
// quietly turn the assertion into a tautology. An unfed belt in line behind it
// makes the target a STRAIGHT belt, and a perpendicular belt joining a straight
// belt fills exactly one lane. That is the whole rig.
//
//	row pb-3   [chest]        [chest]
//	row pb-2   [loader S]     [loader S]
//	row pb-1     v              v
//	row pb       v          ->  ->->[W][E]-> sink   x=-2 side-loads here
//	row pb+1   ->->->->->->->->[W][E]-> sink        x=-4 side-loads here
//	           x=-5  -4  -3  -2
//
// The side-loading happens on the PLAYER'S OWN BELTS, upstream of the parts, so
// the one-belt rule does not touch it: what doubles is the part columns, and the
// feed that reaches the west part is the same half-belt on one lane it always
// was.
func buildLane(s fkapi.LuaSurface, base int) []harness.XY {
	pb := base + 3
	laneBase = pb
	out := make([]harness.XY, 0, 2)
	for i := 0; i <= 1; i++ {
		put(s, part, 0, pb+i, nil, "")
		put(s, part, 1, pb+i, nil, "")
	}

	// row pb: (-3,pb) is fed by nothing and exists only to keep (-2,pb) straight.
	source(s, -2, pb-3, dirS)
	beltsY(s, belt, &dirS, pb-1, pb-1, -2)
	beltsX(s, belt, &dirE, -3, -1, pb)

	// row pb+1: same shape one column further out, so its feed column is clear
	// of row pb's chain. (-5,pb+1) is the dead belt here.
	source(s, -4, pb-3, dirS)
	beltsY(s, belt, &dirS, pb-1, pb, -4)
	beltsX(s, belt, &dirE, -5, -1, pb+1)

	for i := 0; i <= 1; i++ {
		beltsX(s, belt, &dirE, 2, 4, pb+i)
		out = append(out, sink(s, 5, pb+i))
	}
	return out
}

// uio: 2->2 where all four connections are UNDERGROUND ends placed directly
// against the parts -- the output half of a pair on the input side, the input
// half of a pair on the output side, which is both arms of classifySide's
// belt_to_ground_type branch.
//
// The halves are created west to east so each pair takes the nearest partner: a
// pair created out of order could span the part and link the wrong two ends.
func buildUio(s fkapi.LuaSurface, base int) []harness.XY {
	out := make([]harness.XY, 0, 2)
	for i := 0; i <= 1; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 0; i <= 1; i++ {
		y := base + i
		sourceE(s, -7, y)
		beltsX(s, belt, &dirE, -5, -4, y)
		put(s, under, -3, y, &dirE, "input")
		put(s, under, -1, y, &dirE, "output")
		put(s, under, 2, y, &dirE, "input")
		put(s, under, 4, y, &dirE, "output")
		beltsX(s, belt, &dirE, 5, 5, y)
		out = append(out, sink(s, 6, y))
	}
	return out
}

// spio: 2->2 fed by ONE vanilla express splitter whose output face spans both
// parts and drained by a second whose input face does. A splitter is two tiles
// wide and the per-tile search finds it once from each cluster tile it touches,
// so each half is its own edge -- a claim that lived only in a comment in
// classifySide until this rig.
func buildSpio(s fkapi.LuaSurface, base int) []harness.XY {
	out := make([]harness.XY, 0, 2)
	for i := 0; i <= 1; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 0; i <= 1; i++ {
		y := base + i
		sourceE(s, -7, y)
		beltsX(s, belt, &dirE, -5, -2, y)
		beltsX(s, belt, &dirE, 3, 4, y)
		out = append(out, sink(s, 5, y))
	}
	// An east-facing splitter's position is the boundary between its two rows,
	// and its x is the centre of the single COLUMN it stands in: -1 on the way
	// in, against the west parts, and 2 on the way out, against the east ones.
	putAt(s, split, -0.5, float64(base)+1.0, &dirE)
	putAt(s, split, 2.5, float64(base)+1.0, &dirE)
	return out
}

// lio: 1->1 through LOADERS directly against the parts, which is the loader arm
// of classifySide and also the smallest network this compiler can build -- P=1,
// no stages at all, five entities. TWO parts under the one-belt rule, and two is
// the fewest a balancer can have: one to carry the input and one to carry the
// output.
func buildLio(s fkapi.LuaSurface, base int) []harness.XY {
	put(s, part, 0, base, nil, "")
	put(s, part, 1, base, nil, "")
	sourceE(s, -2, base)                  // chest at -2, loader at -1, against the west part
	return []harness.XY{sink(s, 2, base)} // loader at 2, against the east part; chest at 3
}

// lsio: 2->2 fed and drained through LANE SPLITTERS placed directly against the
// parts. A lane splitter is 1x1 and directional, so it classifies exactly as a
// transport belt does -- d == back on the way in, d == dir on the way out -- and
// until `classifySide` named the type it was the one belt-connectable family
// that could stand against a balancer and be silently invisible to it. This rig
// is its own red proof: on the guest before the case existed the cluster has no
// recognised edges at all, so it compiles to nothing and both chests stay empty.
//
// The entity is the data stage's clone of the mod's own hidden one, because base
// ships the TYPE and no buildable instance of it.
func buildLsio(s fkapi.LuaSurface, base int) []harness.XY {
	out := make([]harness.XY, 0, 2)
	for i := 0; i <= 1; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 0; i <= 1; i++ {
		y := base + i
		sourceE(s, -7, y)
		beltsX(s, belt, &dirE, -5, -2, y)
		put(s, lsplit, -1, y, &dirE, "")
		put(s, lsplit, 2, y, &dirE, "")
		beltsX(s, belt, &dirE, 3, 4, y)
		out = append(out, sink(s, 5, y))
	}
	return out
}

// pass: the NEGATIVE. A working 2->2, plus a belt line running east along row
// `base` -- directly over the north face of the top part and perpendicular to
// it. classifySide keys on the belt's direction: from the part's north side
// `dir` is north and `back` is south, and an EAST-facing belt is neither, so it
// falls through and is not an edge. That is the incumbent's accepted limitation
// ("a belt curving away is not an output") met from the other side, and until
// this rig nothing asserted it.
//
// The passing line has its own source and its own chest: the balancer must
// deliver its own two belts exactly, and must not take a single item from a line
// that merely goes past.
//
// UNDER THE ONE-BELT RULE THIS RIG HAS TEETH IT DID NOT HAVE BEFORE. The line
// now runs over the north faces of BOTH top parts, and both of them already
// carry their one belt -- the west part its input, the east part its output --
// so a classifier that read the passing line as an edge would not merely deliver
// an odd rate: it would take two tiles to two belts each and the whole cluster
// would be REFUSED, delivering nothing at all.
func buildPass(s fkapi.LuaSurface, base int) []harness.XY {
	out := make([]harness.XY, 3)
	for i := 1; i <= 2; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 1; i <= 2; i++ {
		y := base + i
		sourceE(s, -5, y)
		beltsX(s, belt, &dirE, -3, -1, y)
		beltsX(s, belt, &dirE, 2, 4, y)
		out[i-1] = sink(s, 5, y)
	}
	sourceE(s, -5, base)
	beltsX(s, belt, &dirE, -3, 4, base)
	out[2] = sink(s, 5, base)
	return out
}

// ---------------------------------------------------------------------------
// the item-conservation rig
//
// A 2-in 2-out balancer fed hard with NOTHING draining it, so within a few
// hundred ticks every belt and every splitter in the hidden network is full and
// the whole thing is stationary. Then, inside a single tick: count everything
// countable, force a recompile, count again. Nothing else can have moved, so the
// difference is exactly what the teardown handed back -- and the guest logs what
// it thinks it handed back, so the two numbers have to agree.
//
// THE FIFTH AND SIXTH PARTS ARE WHAT MAKES THE EDIT POSSIBLE AT ALL. The check
// needs a real edge change on a network that is full, and under the one-belt
// rule every part of a working 2->2 already carries its one belt -- so the belt
// that used to be laid on a free face would now be REFUSED, and the check would
// measure a refusal instead of a recompile. So the block is three rows tall and
// the bottom row carries nothing: the belt goes against the EDGELESS west part,
// which is a third input and takes P from 2 to 4.
// ---------------------------------------------------------------------------

func buildLossRig(base int) {
	s := harness.Surface(surfA)
	for i := 0; i <= 2; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	for i := 0; i <= 1; i++ {
		sourceE(s, -5, base+i)
		beltsX(s, belt, &dirE, -3, -1, base+i)
		beltsX(s, belt, &dirE, 2, 4, base+i)
	}
}

// lossArea is everything a spilled item can end up in or on. Wide enough to
// contain the guest's spill radius: items that landed outside it would read as
// loss and the suite would be lying about which side the defect was on.
func lossArea() fkapi.BoundingBox {
	return harness.Box(-20, float64(lossBase-14), 20, float64(lossBase+16))
}

// countArea is the estate's own two-sweep count: what is lying on the GROUND,
// and what is standing in a transport LINE.
func countArea(s fkapi.LuaSurface, area fkapi.BoundingBox) (ground, lines int64) {
	for _, e := range harness.EntitiesInOfType(s, area, "item-entity") {
		if _, n, ok := harness.GroundStack(e); ok {
			ground += n
		}
	}
	for _, e := range harness.EntitiesIn(s, area, "") {
		lines += harness.TransportLineItems(e)
	}
	return ground, lines
}

// countVisibleItems is every item this rig can be holding, on BOTH surfaces.
//
// Counting only the visible side would not be conservation at all: the point of
// the network is that most of the items are somewhere the player cannot see, so
// a teardown that deleted them would look like a gain on the visible side. The
// whole hidden surface is counted rather than one slot, because no tick passes
// between the two samples and every other rig's network is therefore frozen.
//
// The three returns are the Lua's exactly, INCLUDING that the ground column
// reports only surface a's ground while the total carries the hidden side's too.
func countVisibleItems() (total, ground, lines int64) {
	ga, la := countArea(harness.Surface(surfA), lossArea())
	var gh, lh int64
	if hid, ok := harness.SurfaceIfAny(hidden); ok {
		gh, lh = countArea(hid, harness.Box(-16, -16, 2200, 400))
	}
	return ga + la + gh + lh, ga, la + lh
}

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

// report writes one line per rig, in TABLE order and not in map order. Nothing
// downstream depends on that -- the assertion script keys on the rig name -- but
// a log that reorders itself between runs is a diff nobody can read, and this
// file is the assertion surface.
func report(tick uint64) {
	for i := range rigs {
		out.Open("t=").U(tick).S(" rig=").S(rigs[i].name).S(" out=[")
		for j, slot := range built[i].outs {
			if j > 0 {
				out.S(" ")
			}
			if !slot.has {
				// `block` leaves a hole. -1 is the estate's own "there is no
				// chest here", so that a missing sink cannot read as "nothing
				// was delivered".
				out.I(-1)
				continue
			}
			s := harness.Surface(surfA)
			if rigs[i].otherSurface {
				s = harness.Surface(surfB)
			}
			out.I(harness.ChestCount(s, "steel-chest", slot.xy.X, slot.xy.Y))
		}
		out.S("]").End()
	}
}

// ---------------------------------------------------------------------------
// per-lane occupancy, for the `lane` rig
//
// The one thing a chest total cannot see. A lane-PRESERVING network -- which is
// what this compiler builds if the lane-splitter stage is dropped -- delivers a
// one-lane feed as a one-lane output, and the chests fill at exactly the same
// rate either way. So the assertion is on where the items are STANDING: both
// transport lines of both output rows, summed over the three visible output
// belts, at several ticks of steady flow.
// ---------------------------------------------------------------------------

func sampleLanes(tick uint64) {
	if laneBase == 0 {
		return
	}
	s := harness.Surface(surfA)
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

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

// A full recompile, timed ACROSS THE TICK BOUNDARY, because that is where the
// work is now.
//
// Removing an input belt and putting it back is two complete teardown/rebuild
// cycles of a network already at its final size -- exactly the cost a player
// pays for laying one belt at the edge of a finished balancer. The guest
// batches, so the destroy costs a registry update and a one-shot registration
// and nothing else; the compile happens when `fk_on_deferred` drains the queue
// on the FOLLOWING tick.
//
// So the profiler is opened in the tick that mutates and closed in the tick that
// flushes. This observer declares `better-belt-balancer` as a dependency, which
// fixes the load order, which fixes the handler order: the guest's flush has run
// by the time the schedule below is entered. The window therefore contains one
// whole engine tick as well as the recompile, and the `idle tick pair` line
// measures exactly that and nothing else. SUBTRACT IT.
//
// (The alternative -- forcing the flush with an audit marker, as lossCheck does
// -- was measured and rejected for timing: the audit re-classifies every cluster
// in the save, which is 16 ms of its own against a 5 ms recompile.)
var (
	timingLabel string
	timingProf  harness.Profiler
	timingOn    bool
)

// RETAINED, because this window spans a tick boundary and a handle that is not
// retained is valid only inside its own dispatch. See harness.Profiler.Retain,
// which is where the reason and the measurement live.
func timedBegin(label string) {
	timingLabel, timingProf, timingOn = label, harness.StartProfiler().Retain(), true
}

func timedEnd() {
	if !timingOn {
		return
	}
	timingProf.Stop()
	timingProf.Log("[BBB-M2] timing " + timingLabel + " ")
	timingProf.Release()
	timingOn = false
}

// rigByName is a linear scan over twenty-one entries, which is what the estate's
// `storage.rigs[name]` was and is measurable by nothing.
func rigByName(name string) (*rigCfg, bool) {
	for i := range rigs {
		if rigs[i].name == name {
			return &rigs[i], true
		}
	}
	return nil, false
}

// timeDrop destroys the belt at the west edge of a rig's first input row. That
// is a real edge change, so the fingerprint moves and the network is rebuilt.
func timeDrop(name string) {
	timedEnd()
	c, ok := rigByName(name)
	if !ok {
		return
	}
	s := surfaceOf(c)
	b, found := harness.FindAt(s, -1, c.base, "", "transport-belt")
	if !found {
		out.Open("timing ").S(name).S(": no input belt found").End()
		return
	}
	timedBegin(name + " teardown+rebuild(-1 input)")
	harness.Destroy(b, true)
}

func timeRestore(name string) {
	timedEnd()
	c, ok := rigByName(name)
	if !ok {
		return
	}
	timedBegin(name + " teardown+rebuild(full)")
	put(surfaceOf(c), belt, -1, c.base, &dirE, "")
}

func grow() {
	c, ok := rigByName("regrow")
	if !ok {
		return
	}
	buildInputRow(surfaceOf(c), c.base+c.growTo-1, true)
	out.Open("regrow: input ").I(int64(c.growTo)).S(" added at tick ").U(harness.Tick()).End()
}

func lossCheck() {
	before, gb, lb := countVisibleItems()
	s := harness.Surface(surfA)
	// A belt against the EDGELESS west part of the bottom row is a genuine new
	// edge -- a third input -- so the fingerprint moves and the network is torn
	// down and rebuilt. Every other tile of this cluster already carries its one
	// belt, and a belt against any of them would be refused rather than compiled.
	put(s, belt, -1, lossBase+2, &dirE, "")
	// Same tick, so that "before" and "after" are one atomic sample apart and
	// the difference can only be the teardown. The audit marker is what makes
	// that possible now that the recompile is deferred; it costs a full
	// re-classification of every cluster, which is why this timing line is not
	// comparable with the two above.
	p := harness.StartProfiler()
	auditNow(lossBase)
	p.Stop()
	after, ga, la := countVisibleItems()
	out.Open("loss before=").I(before).S(" after=").I(after).
		S(" returned=").I(after - before).End()
	out.Open("loss detail ground ").I(gb).S("->").I(ga).
		S(" lines(both surfaces) ").I(lb).S("->").I(la).End()
	p.Log("[BBB-M2] timing loss recompile (audit-forced, whole-save re-classification) ")
}

// rawCreateCost is what the engine itself charges for the same work, so that a
// slow compile can be attributed to the right side of the boundary.
func rawCreateCost() {
	hid, ok := harness.SurfaceIfAny(hidden)
	if !ok {
		out.Open("raw: no hidden surface").End()
		return
	}
	p := harness.StartProfiler()
	for i := 1; i <= 32; i++ {
		harness.PlaceSoft(hid, harness.Piece{Name: "bbb-belt", X: 200 + i, Y: 200, Dir: &dirE})
	}
	p.Stop()
	p.Log("[BBB-M2] timing raw 32 create_entity on the hidden surface ")
	q := harness.StartProfiler()
	found := harness.EntitiesIn(hid, harness.Box(190, 190, 250, 210), "")
	q.Stop()
	q.Log("[BBB-M2] timing raw find_entities_filtered (" + itoa(len(found)) + " hits) ")
	for _, e := range found {
		harness.Destroy(e, false)
	}
}

// itoa is the one place this observer needs a number INSIDE a string rather than
// appended to a line, because the count goes into a profiler label. `strconv` is
// a package a guest otherwise never links, and Line's own U cannot help here.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

// auditNow asks the guest to drain its deferred queue and re-classify every
// cluster, synchronously, inside this call. See "forcing the flush" above.
func auditNow(y int) { harness.Audit(harness.Surface(surfA), -30, y) }

var schedule = []harness.Step{
	{Tick: 30, Do: rawCreateCost},
	// Each probe spans two ticks; see the timing block above.
	{Tick: 598, Do: func() { timedBegin("idle tick pair, nothing pending") }},
	{Tick: 600, Do: func() { timeDrop("sat4") }},
	{Tick: 602, Do: func() { timeRestore("sat4") }},
	{Tick: 604, Do: timedEnd},
	{Tick: 660, Do: func() { timeDrop("sat8") }},
	{Tick: 662, Do: func() { timeRestore("sat8") }},
	{Tick: 664, Do: timedEnd},
	{Tick: 900, Do: grow},
	{Tick: 1200, Do: lossCheck},
	{Tick: 1800, Do: func() { report(1800) }},
	// Five lane samples spread over the measurement window. One would be a
	// snapshot of a belt that happens to have a gap in it; five is a statement
	// about where the items live.
	{Tick: 1900, Do: func() { sampleLanes(1900) }},
	{Tick: 2300, Do: func() { sampleLanes(2300) }},
	{Tick: 2700, Do: func() { sampleLanes(2700) }},
	{Tick: 3100, Do: func() { sampleLanes(3100) }},
	{Tick: 3500, Do: func() { sampleLanes(3500) }},
	{Tick: 3540, Do: func() { report(3540) }},
	// After the last sample, so it cannot disturb one. Every rig has been
	// standing untouched since tick 900, so the world and the registry must
	// agree exactly -- and `pass`'s belt line running over a cluster's north
	// face is the reason this audit is worth taking: a classifier that decided
	// it WAS an edge would have rebuilt that network and reported the drift.
	{Tick: 3560, Do: func() { auditNow(0) }},
}

//go:wasmexport fk_on_init
func onInit() {
	makeSurface(surfA, rowsA)
	makeSurface(surfB, rowsB)
	built = make([]rigState, len(rigs))
	for i := range rigs {
		built[i] = buildRig(&rigs[i])
	}
	buildLossRig(lossBase)
	// Compile everything NOW rather than on the first tick after the save is
	// loaded. See "forcing the flush" at the top of this file.
	auditNow(0)
	out.Open("init complete: ").I(int64(len(rigs))).S(" rigs").End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	harness.Run(schedule, fkapi.ReadOnTick(ptr).Tick)
}

func main() {}
