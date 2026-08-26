// Command bbb-sedge-test builds every balancer to Factorio 2.1's rule, ONE BELT
// PER BALANCER PART, and drives the three ways an edit can break it.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-sedge-test/control.lua` was, rig for rig and log line for log
// line, with its little data stage compiled too (obs/sedgedata). See
// agents/estate-port.md.
//
// # Why the rule
//
// Every edge of a cluster is an interface linked belt standing on the cluster's
// own tile, so a part carrying an input on one side and an output on another
// carried TWO belt-connectables on one tile. 2.1 closed the collision-mask
// loophole that permitted it, and the engine's own reason is that belt-to-belt
// connections are re-derived at load rather than saved -- one belt-connectable
// per tile is what makes that unambiguous. See agents/single-edge.md and
// guest/go/sedge.go.
//
// The consequence for a rig is geometry: a 4-in/4-out balancer is EIGHT parts (a
// 4x2 block, four west parts carrying the inputs and four east parts carrying
// the outputs), not four. The smallest balancer is two.
//
//	ctrl   a bare express belt, the yardstick -- so "full throughput" is a
//	       comparison against the engine rather than arithmetic on a wiki
//	s11    1 -> 1 over TWO parts. P = 1, no butterfly stages at all
//	s22    2 -> 2 over FOUR parts (a 2x2 block). P = 2
//	s44    4 -> 4 over EIGHT parts (a 4x2 block). P = 4
//	s35    3 -> 5 over TEN parts (a 5x2 block, two west parts carrying nothing).
//	       P = 8 with loopbacks -- the asymmetric shape, where a wrong edge list
//	       reads as a rate rather than as a crash
//
// And the three refusals, each a working balancer that an edit asks for a second
// belt on one of its parts:
//
//	sbld   a second belt BUILT against an occupied part, by script. No player
//	       index, so the force.print arm fires and nothing is handed back --
//	       which is also the standing negative for the whole run
//	srot   a belt ROTATED onto an occupied part. `entity.direction = ...` raises
//	       nothing at all, so the audit is what finds it -- and rotating it back
//	       must be a SKIP, because the fingerprint is the one the netInfo never
//	       lost
//	smrg   a part BRIDGING two working balancers into one whose bridging tile
//	       would carry two belts. The teardowns belong to AddPart and are queued
//	       before the compiler ever sees the cluster they make, so the merge
//	       pre-pass is what has to refuse it -- with both standing networks
//	       untouched
//
// It ASSERTS NOTHING. test/assert-sedge.py decides; an observer that computed
// the expected answer would be a second implementation of the thing under test.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const (
	part     = "bbb-balancer-part"
	belt     = "express-transport-belt"
	flowItem = "iron-plate"
	surfName = "bbb-sedge"
)

var out = harness.Line{Tag: "[BBB-SEDGE] "}

// Band bases. Far enough apart that no rig's belts are inside another rig's
// two-tile neighbour gate.
const (
	ctrlY = 0
	s11Y  = 6
	s22Y  = 12 // rows 12..13
	s44Y  = 20 // rows 20..23
	s35Y  = 30 // rows 30..34
	sbldY = 42 // rows 42..43, and the refused belt at row 41
	srotY = 50 // rows 50..51, and the rotated belt at row 49
	smrgY = 58 // 58..59 = A, 60 = the gap, 61..62 = B
	rows  = 72
)

// The two directions this suite ever needs. Items flow EAST out of every source,
// so an input belt is one facing east on the west side of a part and an output
// belt is one facing east on its east side; nothing here needs a second
// direction except the two refusal belts, which point SOUTH into a part's north
// face.
//
// Read through the GENERATED accessors, which resolve `defines.direction.east`
// BY NAME against the running game: a define's number is Factorio's own, is not
// in the API description at all, and nothing in this repository writes one down.
// Each accessor is called directly, because FkLua prunes the define table by
// scanning for the constant ids that reach it and an id it cannot prove ships
// all 1137 of them.
var (
	east  uint32
	south uint32
)

// ---------------------------------------------------------------------------
// the rig registry
// ---------------------------------------------------------------------------

// rig is one measured balancer and the chests its outputs drain into.
//
// IT HOLDS TILE POSITIONS AND NOT ENTITIES, which is the shipped guest's own
// rule reached from the other side: `fk_on_init` runs during `--create` and the
// samples are taken during `--benchmark`, so anything held here crosses a save.
// A retained LuaObject would survive that (FkLua's persistent handle space is in
// `storage` and Factorio serializes the reference), and a position needs nothing
// to be true.
type rig struct {
	Name   string
	Chests []harness.XY
}

// rigs is the registry the Lua kept in `storage.order` and `storage.rigs`, in
// registration order -- which IS the order the sample line reports in, and the
// order test/assert-sedge.py parses.
//
// IT IS GUEST STATE AND THEREFORE SAVE STATE. Built in fk_on_init during
// `--create`, read on ticks 700..3400 of `--benchmark`, so an empty registry in
// the benchmark phase is this observer's guest heap having failed to survive the
// save -- which shows up as a sample line with no rigs on it and fails the suite
// rather than passing quietly.
var rigs []rig

func register(name string, chests ...harness.XY) {
	rigs = append(rigs, rig{Name: name, Chests: chests})
}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) {
	harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true,
	})
}

// source is an infinity chest of iron plate feeding a loader that faces east.
func source(s fkapi.LuaSurface, x, y int) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	count := uint32(1000)
	index := uint32(1)
	mode := "at-least"
	if err := (fkapi.LuaEntity{Object: c}).SetInfinityContainerFilters(
		[]fkapi.InfinityInventoryFilter{{
			Index: &index,
			Name:  fkapi.OfString(flowItem),
			Count: &count,
			Mode:  &mode,
		}}); err != nil {
		harness.Fatal("setting the infinity filter", fk.LastError())
	}
	put(s, protos.SedgeLoader, x+1, y, &east, "output")
}

// sink is a loader facing east into a steel chest, and the chest's tile is what
// the registry remembers.
func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	put(s, protos.SedgeLoader, x, y, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

// feed is a source at x = -5 running east along the row, up to but not including
// x0.
func feed(s fkapi.LuaSurface, y, x0 int) {
	source(s, -5, y)
	for x := -3; x <= x0-1; x++ {
		put(s, belt, x, y, &east, "")
	}
}

// drain is a sink whose loader is at x = 2 and whose chest is at x = 3, drained
// from x0 eastwards.
func drain(s fkapi.LuaSurface, y, x0 int) harness.XY {
	for x := x0; x <= 1; x++ {
		put(s, belt, x, y, &east, "")
	}
	return sink(s, 2, y)
}

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

func auditNow(tag string) {
	harness.Audit(surf(), -12, 0)
	out.Open("audited tag=").S(tag).End()
}

// ---------------------------------------------------------------------------
// the mid-run edits
// ---------------------------------------------------------------------------

// sbldAdd is a SECOND belt against a part that already has one. A script build,
// so `player_index` is zero, the force.print arm of the refusal fires, and
// nothing may be handed back anywhere in this run.
func sbldAdd() {
	out.Open("sbld-add begin").End()
	put(surf(), belt, 0, sbldY-1, &south, "")
	out.Open("sbld-add end").End()
}

// rot is the same second belt, arrived at by ROTATION. Assigning `direction`
// raises no event of any kind -- it is the failure envelope's own case -- so the
// audit that follows is what finds the drift, and the repair pass behind it is
// what reaches the refusal.
func rot(dir uint32, tag string) {
	o, ok := harness.FindOnTile(surf(), belt, 0, srotY-1)
	if !ok {
		harness.Fatal("the srot belt is missing", "nothing of that name on the tile")
		return
	}
	if err := (fkapi.LuaEntity{Object: o}).SetDirection(dir); err != nil {
		harness.Fatal("rotating the srot belt", fk.LastError())
	}
	out.Open("srot ").S(tag).End()
}

// mergeAdd is the bridging part, into a one-tile gap whose OWN tile already has
// a belt on each side. Both halves are running and full; their teardowns are
// AddPart's and are queued before the compiler sees the cluster they make.
func mergeAdd() {
	out.Open("merge-add begin").End()
	put(surf(), part, 0, smrgY+2, nil, "")
	out.Open("merge-add end").End()
}

func mergeRemove() {
	out.Open("merge-remove begin").End()
	if o, ok := harness.FindOnTile(surf(), part, 0, smrgY+2); ok {
		yes := true
		if _, err := (fkapi.LuaEntity{Object: o}).Destroy(
			fkapi.LuaEntityDestroyArgs{RaiseDestroy: &yes}); err != nil {
			harness.Fatal("removing the bridging part", fk.LastError())
		}
	}
	out.Open("merge-remove end").End()
}

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

func report(tag string) {
	s := surf()
	out.Open("sample tag=").S(tag).S(" tick=").U(harness.Tick())
	for _, r := range rigs {
		out.S(" ").S(r.Name).S("=")
		for i, c := range r.Chests {
			if i > 0 {
				out.S(",")
			}
			out.I(harness.ChestCount(s, "steel-chest", c.X, c.Y))
		}
	}
	out.End()
}

// reportTile is THE ANTI-VACUITY LINE. Every rig in this suite is built to the
// rule it is about, so a run in which some part quietly ended up with two belts
// against it would refuse a rig the assertions expect to be running -- and a run
// in which a refusal rig's extra belt failed to be placed would pass every rate
// check while proving nothing. This reports what is really standing on the two
// tiles the refusals are about.
func reportTile(tag string, x, y int) {
	out.Open("tile tag=").S(tag).S(" at=").I(int64(x)).S(",").I(int64(y)).S(" holds=[")
	for i, n := range harness.NamesOnTile(surf(), x, y) {
		if i > 0 {
			out.S(",")
		}
		out.S(n)
	}
	out.S("]").End()
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

var schedule = []harness.Step{
	{Tick: 60, Do: func() { auditNow("built") }},
	{Tick: 500, Do: sbldAdd},
	{Tick: 502, Do: func() { auditNow("post-sbld"); reportTile("post-sbld", 0, sbldY-1) }},
	{Tick: 600, Do: func() { rot(south, "on") }},
	{Tick: 602, Do: func() { auditNow("post-rot"); reportTile("post-rot", 0, srotY-1) }},
	{Tick: 700, Do: func() { report("a") }},
	{Tick: 1000, Do: func() { report("b") }},
	{Tick: 1200, Do: func() { rot(east, "off") }},
	{Tick: 1202, Do: func() { auditNow("post-rot-back") }},
	{Tick: 1400, Do: mergeAdd},
	{Tick: 1402, Do: func() { auditNow("post-merge"); reportTile("post-merge", 0, smrgY+2) }},
	{Tick: 1500, Do: func() { report("c") }},
	{Tick: 1800, Do: func() { report("d") }},
	{Tick: 1900, Do: mergeRemove},
	{Tick: 1902, Do: func() { auditNow("post-unmerge") }},
	{Tick: 2100, Do: func() { report("e") }},
	{Tick: 3400, Do: func() { report("f") }},
	{Tick: 3450, Do: func() { auditNow("final") }},
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	east = fkapi.DefinesDirectionEast()
	south = fkapi.DefinesDirectionSouth()
}

// ---------------------------------------------------------------------------
// the rigs
// ---------------------------------------------------------------------------

//go:wasmexport fk_on_init
func onInit() {
	s := harness.Flat{
		Name:        surfName,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: rows / 2},
		// ceil(rows/32) + 3, written out because rows is a constant and the
		// arithmetic was a `math.ceil` in the Lua.
		ChunkRadius: 6,
		X0:          -16,
		Y0:          -12,
		X1:          16,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()
	rigs = rigs[:0]

	// ctrl: the yardstick. A bare express belt from the same source to the same
	// kind of sink.
	feed(s, ctrlY, 0)
	register("ctrl", drain(s, ctrlY, 0))

	// s11: 1 -> 1 over TWO parts. The west part carries the input, the east part
	// carries the output, and neither carries both -- which is the rule.
	put(s, part, 0, s11Y, nil, "")
	put(s, part, 1, s11Y, nil, "")
	feed(s, s11Y, 0)
	register("s11", drain(s, s11Y, 2))

	// s22: 2 -> 2 over a 2x2 block. Two west parts, two east parts.
	register("s22", column(s, s22Y, 2, 2)...)

	// s44: 4 -> 4 over a 4x2 block -- EIGHT parts for the shape four used to
	// build, which is the footprint the rule costs.
	register("s44", column(s, s44Y, 4, 4)...)

	// s35: 3 -> 5 over a 5x2 block. The bottom two west parts carry NOTHING,
	// which is what makes the shape asymmetric without breaking the rule: P = 8
	// with loopbacks, and a wrong edge list reads as a rate rather than a crash.
	register("s35", column(s, s35Y, 5, 3)...)

	// sbld: a working 2 -> 2, whose north-west part gets a second belt at t=500.
	register("sbld", column(s, sbldY, 2, 2)...)

	// srot: the same shape, with a belt already standing on the north-west
	// part's north face and pointing EAST -- which is not an edge, because a
	// belt flowing past a cluster is not pointing at it. Rotating it south at
	// t=600 makes it one, silently.
	srot := column(s, srotY, 2, 2)
	put(s, belt, 0, srotY-1, &east, "")
	register("srot", srot...)

	// smrg: TWO 1 -> 1 balancers in one column with a one-tile gap between them,
	// and a belt on BOTH SIDES OF THE GAP TILE. Bridging the gap makes one
	// cluster whose bridging part would carry two belts -- so the merge must be
	// refused with both standing networks untouched.
	//
	// Neither gap belt is adjacent to either half: a belt beside the gap is
	// DIAGONAL from the nearest part, and adjacency here is four-way.
	put(s, part, 0, smrgY, nil, "")
	put(s, part, 0, smrgY+1, nil, "")
	feed(s, smrgY, 0)
	a := drain(s, smrgY+1, 1)

	put(s, part, 0, smrgY+3, nil, "")
	put(s, part, 0, smrgY+4, nil, "")
	feed(s, smrgY+3, 0)
	b := drain(s, smrgY+4, 1)

	put(s, belt, -1, smrgY+2, &east, "")
	put(s, belt, 1, smrgY+2, &east, "")
	register("smrg", a, b)

	reportTile("init", 0, smrgY+2)
	auditNow("t0")
	out.Open("init complete").End()
}

// column lays `depth` rows of a west part and an east part, feeding the first
// `fed` of them and draining all of them. It is the shape s22, s44, s35, sbld
// and srot are all made of, which is the whole geometry the one-belt rule costs:
// every column of parts is two columns now.
func column(s fkapi.LuaSurface, y0, depth, fed int) []harness.XY {
	chests := make([]harness.XY, 0, depth)
	for dy := 0; dy < depth; dy++ {
		put(s, part, 0, y0+dy, nil, "")
		put(s, part, 1, y0+dy, nil, "")
		if dy < fed {
			feed(s, y0+dy, 0)
		}
		chests = append(chests, drain(s, y0+dy, 2))
	}
	return chests
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	harness.Run(schedule, fkapi.ReadOnTick(ptr).Tick)
}

func main() {}
