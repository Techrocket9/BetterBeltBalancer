// Command bbb-lane-test builds LANE-KEEPING balancers and measures what each
// lane of every output carries.
//
// THE FEED IS LANE-SPLIT, and every rig gets the same one: an express belt
// side-loaded from the north with iron plate and from the south with copper
// plate, so iron rides its LEFT lane and copper its RIGHT. A lane-keeping
// balancer must hand every output iron on the left and copper on the right; an
// ordinary one mixes them, and the `o12` control is that ordinary one on the
// same feed, which is what makes a clean lane rig mean something.
//
//	o12    ORDINARY 1->2, the control. Its outputs must carry copper on the
//	       left: the feed really was split and an ordinary balancer really
//	       mixes it.
//	l12    lane-keeping 1->2. At tick 600 an output belt is laid on its
//	       edgeless part while it runs full, which is a recompile under load:
//	       the carry must put each drained item back on the lane it came off
//	       (guest/go/carry.go, laneOf), and the window after it is where a
//	       mistake would show.
//	lprio  lane-keeping 1->2 with a PRIORITY output, flagged at tick 5 through
//	       the remote interface. The priority network's concentrator splitters
//	       are the only splitters in this mod that are not plain, and whether
//	       they keep lanes was never measured before this rig.
//	l22    lane-keeping 2->2: two inputs' left lanes balanced together over
//	       both outputs' left lanes.
//	ladj   an ORDINARY part and a lane-keeping part side by side, a belt into
//	       the first and a belt out of the second. They must stay two
//	       balancers -- one with only an input, one with only an output -- so
//	       nothing reaches the sink.
//	oadj   the same with two ordinary parts, the control: one balancer, and
//	       the sink fills.
//	ljam   lane-keeping 1->2 whose two outputs DEAD-END, so the whole network
//	       fills and jams. At tick 600 a third output with a sink is laid on
//	       its edgeless part: the recompile drains a FULL network and the
//	       carry must put every item back on its own lane. l12 drains freely
//	       and carries a dozen items; this one carries the machine.
//	frep   an ORDINARY 1->1 whose two parts a script fast-replaces with lane
//	       parts at tick 300 and back at tick 900. The engine raises nothing
//	       for a part it replaces, so the guest has to notice a part of the
//	       other kind on a registered tile (main.go); the sink must keep
//	       filling after each swap.
//	lfor   two lane-keeping 1->1s on ADJACENT tiles, one on the player force
//	       (iron left, copper right) and one on a second force (steel left,
//	       plastic right). The kind bit sits beside the force, so the two must
//	       stay two balancers, each delivering only its own pair. At tick 1200
//	       the second force is merged into the player's: the two become one
//	       lane-keeping 2->2, whose left lanes carry iron and steel and whose
//	       right lanes copper and plastic, never crossed. A merge that lost the
//	       kind bit would compile an ordinary network and cross them, and a carry
//	       pool that missed the remap would spill rather than reinsert.
//	lorph  a lane-keeping 1->1 compiled in on_init and then destroyed with no
//	       event, which leaves its hidden network standing and unclaimed. The
//	       benchmark phase loads the save as an UPGRADE (`bump_build`), so the
//	       rebuild must find that slot and sweep it. A lane-keeping network has
//	       no lane splitter, which is what the sweep used to look for.
//
// It ASSERTS NOTHING. test/assert-lane.py decides.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const (
	part     = "bbb-balancer-part"
	lanePart = "bbb-lane-balancer-part"
	belt     = "express-transport-belt"
	leftItem = "iron-plate"
	rightItm = "copper-plate"
	left2    = "steel-plate"
	right2   = "plastic-bar"
	other    = "bbb-lane-other"
	mergeAt  = 1200
	surfName = "bbb-lane"

	modName    = "better-belt-balancer"
	prioMethod = "set-part-priority"
)

var out = harness.Line{Tag: "[BBB-LANE] "}

var north, east, south, west uint32

// Band bases, fourteen rows apart so that no rig's pieces are inside another
// rig's two-tile neighbour gate.
const (
	o12Y   = 0
	l12Y   = 14
	lprioY = 28
	l22Y   = 42
	ladjY  = 58
	oadjY  = 62
	lorphY = 66
	ljamY  = 74
	frepY  = 82
	lforY  = 90
	rows   = lforY + 12
)

// The three measuring windows: before the recompile under load, the one it
// happens in, and after.
const (
	carryTick = 600
	win1      = 600
	win2      = 1200
	endTick   = 2390
)

// ---------------------------------------------------------------------------
// what is measured
// ---------------------------------------------------------------------------

// probe is one belt tile whose two lanes are read every eight ticks. An item
// crosses an express tile in under eleven, so every item that passes is seen.
type probe struct {
	Name string
	X, Y int
	// counts[w][lane][item]: lane 0 left, 1 right; item 0 iron, 1 copper,
	// 2 steel, 3 plastic. The first two are the split feed every rig carries;
	// the last two only the second force's half of `lfor`.
	counts [3][2][4]int64
}

var probes []probe

// sinks are the chests reported at the end, by rig.
type sinkAt struct {
	Name string
	At   harness.XY
}

var sinks []sinkAt

func addProbe(name string, x, y int) { probes = append(probes, probe{Name: name, X: x, Y: y}) }

func window(tick uint64) int {
	switch {
	case tick < win1:
		return 0
	case tick < win2:
		return 1
	}
	return 2
}

func sample(tick uint64) {
	s := surf()
	w := window(tick)
	left := fkapi.DefinesTransportLineLeftLine()
	right := fkapi.DefinesTransportLineRightLine()
	for i := range probes {
		p := &probes[i]
		o, ok := harness.FindOnTile(s, belt, p.X, p.Y)
		if !ok {
			continue
		}
		e := fkapi.LuaEntity{Object: o}
		for lane, idx := range [2]uint32{left, right} {
			l, err := e.GetTransportLine(idx)
			if err != nil {
				continue
			}
			c, err := fkapi.LuaTransportLine{Object: l}.GetContents()
			if err != nil {
				continue
			}
			for _, it := range c {
				switch it.Name {
				case leftItem:
					p.counts[w][lane][0] += int64(it.Count)
				case rightItm:
					p.counts[w][lane][1] += int64(it.Count)
				case left2:
					p.counts[w][lane][2] += int64(it.Count)
				case right2:
					p.counts[w][lane][3] += int64(it.Count)
				}
			}
		}
	}
}

func reportLanes(w int) {
	for i := range probes {
		p := &probes[i]
		c := p.counts[w]
		out.Open("lanes w=").I(int64(w)).S(" probe=").S(p.Name).
			S(" L=").I(c[0][0]).S("/").I(c[0][1]).
			S(" R=").I(c[1][0]).S("/").I(c[1][1]).
			S(" L2=").I(c[0][2]).S("/").I(c[0][3]).
			S(" R2=").I(c[1][2]).S("/").I(c[1][3]).End()
	}
}

func reportSinks() {
	s := surf()
	for _, k := range sinks {
		iron, copper, steel, plastic := int64(0), int64(0), int64(0), int64(0)
		if o, ok := harness.FindOnTile(s, "steel-chest", k.At.X, k.At.Y); ok {
			iron = harness.ItemCountIn(o, leftItem)
			copper = harness.ItemCountIn(o, rightItm)
			steel = harness.ItemCountIn(o, left2)
			plastic = harness.ItemCountIn(o, right2)
		}
		out.Open("sink rig=").S(k.Name).S(" iron=").I(iron).S(" copper=").I(copper).
			S(" steel=").I(steel).S(" plastic=").I(plastic).End()
	}
}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true})
}

func chest(s fkapi.LuaSurface, item string, x, y int) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	harness.InfinityFilter(c, item, "", 1000)
}

// laneFeed is the lane-split feed into a balancer whose input part's west face
// is at (0, y): an east belt from x = -9 to -1, side-loaded at column `col`
// from the north (iron, the LEFT lane of an east belt) and from the south
// (copper, the right). The belt at -9 is there so that the side-loaded tile
// has a belt behind it and stays straight rather than curving.
func laneFeed(s fkapi.LuaSurface, y, col int) {
	chest(s, leftItem, col, y-3)
	put(s, protos.LaneLoader, col, y-2, &south, "output")
	put(s, belt, col, y-1, &south, "")
	chest(s, rightItm, col, y+3)
	put(s, protos.LaneLoader, col, y+2, &north, "output")
	put(s, belt, col, y+1, &north, "")
	for x := -9; x <= -1; x++ {
		put(s, belt, x, y, &east, "")
	}
}

// eastOut is an output belt leaving the part at (x-1, y) eastwards into a
// sink, with a probe on its third tile.
func eastOut(s fkapi.LuaSurface, rig string, x, y int) {
	put(s, protos.LaneLoader, x+4, y, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 5, Y: y})
	for bx := x; bx <= x+3; bx++ {
		put(s, belt, bx, y, &east, "")
	}
	addProbe(rig+"-out@"+itoa(y), x+2, y)
	sinks = append(sinks, sinkAt{Name: rig + "-out@" + itoa(y), At: harness.XY{X: x + 5, Y: y}})
}

func block(s fkapi.LuaSurface, name string, x0, y0, w, h int) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			put(s, name, x0+dx, y0+dy, nil, "")
		}
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// oneTwo is a 2x2 block of `name`: the lane-split feed into its north-west
// part, an output from each part of its east column, and the south-west part
// edgeless (the carry leg lays its third output there).
func oneTwo(s fkapi.LuaSurface, rig, name string, y int) {
	block(s, name, 0, y, 2, 2)
	laneFeed(s, y, -7)
	addProbe(rig+"-feed", -2, y)
	eastOut(s, rig, 2, y)
	eastOut(s, rig, 2, y+1)
}

// ---------------------------------------------------------------------------
// the scheduled gestures
// ---------------------------------------------------------------------------

func setPrio(x, y int) {
	s := surf()
	idx, err := s.Index()
	if err != nil {
		harness.Fatal("reading the surface index", "for lprio")
		return
	}
	v, st := fkapi.RemoteCall(modName, prioMethod,
		fkapi.OfNumber(float64(idx)), fkapi.OfNumber(float64(x)), fkapi.OfNumber(float64(y)),
		fkapi.OfBool(true))
	ok := st == fkapi.StatusOK && v.Tag == fkapi.TagBool && v.Bool
	out.Open("flag rig=lprio accepted=").B(ok).End()
}

// jamAdd is carryAdd for ljam.
func jamAdd() {
	s := surf()
	y := ljamY + 1
	harness.Place(s, harness.Piece{Name: "steel-chest", X: -4, Y: y})
	put(s, protos.LaneLoader, -3, y, &west, "input")
	put(s, belt, -2, y, &west, "")
	put(s, belt, -1, y, &west, "")
	addProbe("ljam-west@"+itoa(y), -2, y)
	sinks = append(sinks, sinkAt{Name: "ljam-west@" + itoa(y), At: harness.XY{X: -4, Y: y}})
	out.Open("jam-add tick=").U(harness.Tick()).End()
}

// swap fast-replaces frep's two parts with `name`, the way a script does it,
// and reports what stands on each tile afterwards and what the sink holds.
func swap(name string) {
	s := surf()
	for x := 0; x <= 1; x++ {
		_, ok := harness.PlaceSoft(s, harness.Piece{Name: name, X: x, Y: frepY, FastReplace: true, Raise: true})
		if !ok {
			out.Open("frep-swap failed to=").S(name).S(" x=").I(int64(x)).End()
		}
	}
	frepReport("swap-" + name)
}

func frepReport(tag string) {
	s := surf()
	lanes, plain := 0, 0
	for x := 0; x <= 1; x++ {
		if _, ok := harness.FindOnTile(s, lanePart, x, frepY); ok {
			lanes++
		}
		if _, ok := harness.FindOnTile(s, part, x, frepY); ok {
			plain++
		}
	}
	sink := int64(0)
	if o, ok := harness.FindOnTile(s, "steel-chest", 7, frepY); ok {
		sink = harness.ItemCountIn(o, leftItem)
	}
	out.Open("frep t=").S(tag).S(" tick=").U(harness.Tick()).S(" lane=").I(int64(lanes)).
		S(" ordinary=").I(int64(plain)).S(" sink=").I(sink).End()
}

// forcePut places one piece of `lfor` on a given force.
func forcePut(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ, force string) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Type: typ, Force: force, Raise: true})
}

func forceChest(s fkapi.LuaSurface, item string, x, y int, force string) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y, Force: force})
	harness.InfinityFilter(c, item, "", 1000)
}

// bandForces stages `lfor`. The player's half is two parts at (0,y) and (1,y)
// fed from the west and drained east; the second force's half is the two parts
// directly below, fed from the SOUTH up a lane-split column at x = 0 (on a belt
// running north the left lane is the west one) and drained south from x = 1.
func bandForces(s fkapi.LuaSurface) {
	y := lforY
	for x := 0; x <= 1; x++ {
		put(s, lanePart, x, y, nil, "")
		forcePut(s, lanePart, x, y+1, nil, "", other)
	}
	laneFeed(s, y, -7)
	eastOut(s, "lforA", 2, y)

	// The second force's feed: a north column from (0, y+10) up to (0, y+2),
	// side-loaded at row y+8 from the west (steel, the left lane) and from the
	// east (plastic, the right).
	r := y + 8
	forceChest(s, left2, -3, r, other)
	forcePut(s, protos.LaneLoader, -2, r, &east, "output", other)
	forcePut(s, belt, -1, r, &east, "", other)
	forceChest(s, right2, 3, r, other)
	forcePut(s, protos.LaneLoader, 2, r, &west, "output", other)
	forcePut(s, belt, 1, r, &west, "", other)
	for by := y + 10; by >= y+2; by-- {
		forcePut(s, belt, 0, by, &north, "", other)
	}
	// Its output: south from (1, y+2) into a sink at (1, y+7).
	for by := y + 2; by <= y+5; by++ {
		forcePut(s, belt, 1, by, &south, "", other)
	}
	forcePut(s, protos.LaneLoader, 1, y+6, &south, "input", other)
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 1, Y: y + 7, Force: other})
	addProbe("lforB-out@"+itoa(y+4), 1, y+4)
	sinks = append(sinks, sinkAt{Name: "lforB-out@" + itoa(y+7), At: harness.XY{X: 1, Y: y + 7}})
}

// mergeForces merges the second force into the player's, which is the
// `lfor` gesture.
func mergeForces() {
	src, okA := harness.ForceByName(other)
	dst, okB := harness.ForceByName(harness.PlayerForce)
	if okA && okB {
		if err := fkapi.Game.MergeForces(src, dst); err != nil {
			harness.Fatal("merge_forces", fk.LastError())
		}
	}
	out.Open("forces merged tick=").U(harness.Tick()).End()
}

// carryAdd lays a third output on l12's edgeless part while it runs full: a
// west belt from (-1, y+1) into a sink, sink first so that the belt the guest
// sees is the last piece.
func carryAdd() {
	s := surf()
	y := l12Y + 1
	harness.Place(s, harness.Piece{Name: "steel-chest", X: -4, Y: y})
	put(s, protos.LaneLoader, -3, y, &west, "input")
	put(s, belt, -2, y, &west, "")
	put(s, belt, -1, y, &west, "")
	addProbe("l12-west@"+itoa(y), -2, y)
	sinks = append(sinks, sinkAt{Name: "l12-west@" + itoa(y), At: harness.XY{X: -4, Y: y}})
	out.Open("carry-add tick=").U(harness.Tick()).End()
}

func auditNow(tag string) {
	harness.Audit(surf(), -16, 0)
	out.Open("audited tag=").S(tag).End()
}

var schedule = []harness.Step{
	{Tick: 5, Do: func() { setPrio(1, lprioY) }},
	{Tick: carryTick, Do: carryAdd},
	{Tick: carryTick, Do: jamAdd},
	{Tick: mergeAt, Do: mergeForces},
	{Tick: 299, Do: func() { frepReport("before") }},
	{Tick: 300, Do: func() { swap(lanePart) }},
	{Tick: 899, Do: func() { frepReport("lane") }},
	{Tick: 900, Do: func() { swap(part) }},
	{Tick: endTick, Do: func() { frepReport("end") }},
	{Tick: win1 - 1, Do: func() { reportLanes(0) }},
	{Tick: win2 - 1, Do: func() { reportLanes(1) }},
	{Tick: endTick, Do: func() { reportLanes(2); reportSinks() }},
	{Tick: endTick + 2, Do: func() { auditNow("final") }},
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	north = fkapi.DefinesDirectionNorth()
	east = fkapi.DefinesDirectionEast()
	south = fkapi.DefinesDirectionSouth()
	west = fkapi.DefinesDirectionWest()
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
		ChunkRadius: 6,
		X0:          -20,
		Y0:          -12,
		X1:          20,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()
	probes, sinks = probes[:0], sinks[:0]

	oneTwo(s, "o12", part, o12Y)
	oneTwo(s, "l12", lanePart, l12Y)
	oneTwo(s, "lprio", lanePart, lprioY)

	// l22: a column of seven, inputs on the west of its top and bottom parts,
	// outputs on the east of the second and the third from the bottom. Every
	// part carries one belt at most (Factorio 2.1's rule). The two feeds
	// side-load at different columns so that the top feed's southern loader
	// and the bottom feed's northern one do not meet.
	block(s, lanePart, 0, l22Y, 1, 7)
	laneFeed(s, l22Y, -7)
	laneFeed(s, l22Y+6, -4)
	addProbe("l22-feed@"+itoa(l22Y), -2, l22Y)
	addProbe("l22-feed@"+itoa(l22Y+6), -2, l22Y+6)
	eastOut(s, "l22", 1, l22Y+2)
	eastOut(s, "l22", 1, l22Y+4)

	// ladj and oadj: two parts side by side, a plain iron feed into the west
	// one and an output from the east one.
	for i, kind := range [2]string{lanePart, part} {
		y := ladjY
		rig := "ladj"
		if i == 1 {
			y, rig = oadjY, "oadj"
		}
		put(s, part, 0, y, nil, "")
		put(s, kind, 1, y, nil, "")
		chest(s, leftItem, -5, y)
		put(s, protos.LaneLoader, -4, y, &east, "output")
		for x := -3; x <= -1; x++ {
			put(s, belt, x, y, &east, "")
		}
		put(s, protos.LaneLoader, 6, y, &east, "input")
		harness.Place(s, harness.Piece{Name: "steel-chest", X: 7, Y: y})
		for x := 2; x <= 5; x++ {
			put(s, belt, x, y, &east, "")
		}
		sinks = append(sinks, sinkAt{Name: rig, At: harness.XY{X: 7, Y: y}})
	}

	// frep: an ordinary 1->1 with a plain iron feed and a sink, two parts.
	put(s, part, 0, frepY, nil, "")
	put(s, part, 1, frepY, nil, "")
	chest(s, leftItem, -5, frepY)
	put(s, protos.LaneLoader, -4, frepY, &east, "output")
	for x := -3; x <= -1; x++ {
		put(s, belt, x, frepY, &east, "")
	}
	put(s, protos.LaneLoader, 6, frepY, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 7, Y: frepY})
	for x := 2; x <= 5; x++ {
		put(s, belt, x, frepY, &east, "")
	}

	harness.CreateForce(other)
	bandForces(s)

	// lorph: a lane-keeping 1->1 with no feed, compiled by the audit below and
	// then destroyed with no event.
	put(s, lanePart, 0, lorphY, nil, "")
	put(s, lanePart, 1, lorphY, nil, "")
	put(s, belt, -1, lorphY, &east, "")
	put(s, belt, 2, lorphY, &east, "")

	// ljam: oneTwo's shape with dead-end outputs, each probed.
	block(s, lanePart, 0, ljamY, 2, 2)
	laneFeed(s, ljamY, -7)
	addProbe("ljam-feed", -2, ljamY)
	for dy := 0; dy <= 1; dy++ {
		for x := 2; x <= 5; x++ {
			put(s, belt, x, ljamY+dy, &east, "")
		}
		addProbe("ljam-dead@"+itoa(ljamY+dy), 4, ljamY+dy)
	}

	auditNow("t0")

	for x := 0; x <= 1; x++ {
		o, ok := harness.FindOnTile(s, lanePart, x, lorphY)
		if !ok {
			harness.Fatal("finding an lorph part", fk.LastError())
			continue
		}
		harness.Destroy(o, false)
	}
	out.Open("orphaned rig=lorph").End()
	out.Open("init complete").End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	tick := fkapi.ReadOnTick(ptr).Tick
	if tick > 0 && tick%8 == 0 && tick < endTick {
		sample(tick)
	}
	harness.Run(schedule, tick)
}

func main() {}
