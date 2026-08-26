// Command bbb-marathon-test measures what a NET-ZERO world operation costs the
// guest heap, forever.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-marathon-test/control.lua` was, leg for leg, tick for tick and
// log line for log line, with its little data stage compiled too (obs/mardata).
// See agents/estate-port.md.
//
// # What it measures
//
// Under `-gc=leaking` every transient guest allocation is permanent -- it is in
// the linear memory, in every save, and in every multiplayer join, for the life
// of the session. So the question a 300-hour game asks is not "does it leak"
// (everything does) but "what is the SLOPE": how many bytes does one complete
// place-and-remove cycle add that never come back, and is that slope flat.
//
// The instrument is the MOD'S OWN heap probe. `[BBB] heap <what> sys=... alloc=...`
// is written at every audit (guest/go/gc.go); under `-gc=leaking` `alloc` is
// TinyGo's bump allocator reporting every byte it has ever handed out, which is
// exactly the permanent heap. This observer drives a leg, places a `bbb-audit`
// marker, and lets the mod print the number. test/assert-marathon.py fits the
// slope.
//
// # Why the schedule is the thing to be careful with
//
// EVERY NUMBER THIS SUITE REPORTS IS A FUNCTION OF WHEN THINGS HAPPENED. The
// legs below run on a plan computed from a table of (iterations, period), and a
// tick's drift anywhere in it moves a slope -- so the port keeps the plan
// arithmetic, the phase numbers and the order of the placements inside each
// phase exactly as the Lua had them. THIS OBSERVER'S OWN HEAP IS IRRELEVANT: it
// is a separate wasm module with a separate linear memory, and what the probe
// reports is the mod's.
//
// # The legs, each chosen so that ONE term dominates it
//
//	cal   ten audits with the world untouched. The audit is not free -- it
//	      re-classifies every cluster in the save -- and every other leg pays
//	      exactly one of them per iteration, so this is the constant that gets
//	      subtracted from all of them.
//	A     the headline cycle: place a 4-part balancer and its 12 belts in one
//	      tick, run it under load, remove all 16 entities, audit. Net zero.
//	B     lay a belt two tiles from a finished balancer and pick it up again.
//	      The cluster is queued and re-classified; the fingerprint says nothing
//	      moved and NOTHING is rebuilt. This is the common case -- a player
//	      building near a balancer -- and its slope is the one that multiplies
//	      by the biggest number in a real game.
//	C     remove one of a balancer's input belts and put it back: an edge really
//	      moves, so this is two full teardown-and-rebuilds per iteration. A
//	      rotation costs the same, by construction: the fingerprint covers
//	      direction.
//	D     lay a belt eighteen tiles from anything and pick it up again. No
//	      cluster is within the two-tile gate, so no compile happens at all and
//	      what is left is the raw per-EVENT cost of being entered. On a
//	      multiplayer server this is the term with the largest multiplier of all
//	      -- every belt anyone lays anywhere on the map pays it.
//	E     six entities placed in one tick and removed in one tick: a small
//	      blueprint paste and its undo.
//	G     the same edit as C but on a 4x4 -- sixteen parts, four in, four out, a
//	      32-entity network. C and G together are what says the compile term
//	      SCALES with the network rather than with the number of edits, which is
//	      what a projection out to three hundred hours needs.
//	F     a saturated balancer grown by one part and taken apart again, with
//	      every item on the surface counted every cycle. This is both a slope leg
//	      and the conservation kill-test: 100 cycles of add-part /
//	      remove-everything on a network that is FULL, and the count may never
//	      rise and may not fall by more than the documented spill loss.
//
// Every leg ends with the world in the state the calibration measured, so all
// the audits cost the same and the subtraction is honest.
//
// # The rule the rigs are built to
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART, so
// a row's input and its output stand against two different tiles and every
// column of parts is two columns wide (agents/single-edge.md). The `big` rig did
// not move at all: a 4x4 block whose west column carries the inputs and whose
// east column carries the outputs was already single-edge, and always had been.
//
// It ASSERTS NOTHING. test/assert-marathon.py decides.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const (
	part = "bbb-balancer-part"
	belt = "express-transport-belt"
	// loader is this observer's own data stage's prototype (obs/mardata), and
	// the name is written down in both places for the reason sedge's is: the
	// data guest imports fkdata and this one imports fkapi, so no package can be
	// shared between them without dragging one import into the other's module.
	loader   = "bbbt-loader"
	flowItem = "iron-plate"
	surfName = "bbb-mar"

	// beltType is what a `type` filter has to say to find a belt and not the
	// loader or the part beside it.
	beltType = "transport-belt"
)

var out = harness.Line{Tag: "[BBB-MAR] "}

// east is read through the GENERATED accessor, which resolves
// `defines.direction.east` BY NAME against the running game: a define's number
// is Factorio's own, is not in the API description at all, and nothing in this
// repository writes one down.
var east uint32

// Rigs are 30 rows apart, which is more than `spill_item_stack`'s 12-tile radius
// in both directions: leg F counts a band and a neighbour's spill landing in it
// would read as items minted out of nothing.
const pitch = 30

const (
	keepY  = 0
	cycleY = pitch
	churnY = 2 * pitch
	farY   = 3 * pitch
	bigY   = 4 * pitch
	rows   = 5 * pitch
)

// churnStock is leg F's finite stock. A steel chest holds 48 stacks, so this is
// also the most an insert can put in one -- big enough that 100 cycles never
// starve the rig, and a round number the count can be read against.
const churnStock = 4800

func churnBand() fkapi.BoundingBox {
	return harness.Box(-20, churnY-14, 20, churnY+16)
}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) {
	harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true,
	})
}

// putSoft is put where a collision is a legitimate outcome of the schedule
// rather than a broken test.
func putSoft(s fkapi.LuaSurface, name string, x, y int, dir *uint32) {
	harness.PlaceSoft(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Raise: true})
}

func infinityChest(s fkapi.LuaSurface, x, y int) {
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
}

// sourceInf and sourceFinite are both permanent: only the parts and the belts
// between them move during a leg.
func sourceInf(s fkapi.LuaSurface, y int) {
	infinityChest(s, -6, y)
	put(s, loader, -5, y, &east, "output")
}

// sourceFinite is what makes leg F's count a CONSERVATION statement rather than
// a reading: a finite stock cannot be topped up, so an item that goes missing
// stays missing.
func sourceFinite(s fkapi.LuaSurface, y, count int) {
	c := harness.Place(s, harness.Piece{Name: "steel-chest", X: -6, Y: y})
	inv, err := (fkapi.LuaControl{Object: c}).GetInventory(fkapi.DefinesInventoryChest())
	if err != nil || inv == nil {
		harness.Fatal("the finite source has no chest inventory", fk.LastError())
		return
	}
	if _, err := (fkapi.LuaInventory{Object: *inv}).Insert(fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(flowItem)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(float64(count))},
	)); err != nil {
		harness.Fatal("stocking the finite source", fk.LastError())
	}
	put(s, loader, -5, y, &east, "output")
}

func sink(s fkapi.LuaSurface, y int) {
	put(s, loader, 4, y, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 5, Y: y})
}

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

// countBand totals every item standing in a box: on the ground, on a belt, in a
// splitter's transport lines, in a chest.
func countBand(s fkapi.LuaSurface, box fkapi.BoundingBox) int64 {
	var total int64
	for _, o := range harness.EntitiesIn(s, box, "") {
		e := fkapi.LuaEntity{Object: o}
		if t, err := e.Type(); err == nil && t == "item-entity" {
			st, err := e.Stack()
			if err != nil {
				continue
			}
			stack := fkapi.LuaItemStack{Object: st}
			if ok, err := stack.ValidForRead(); err == nil && ok {
				if n, err := stack.Count(); err == nil {
					total += int64(n)
				}
			}
			continue
		}
		total += harness.TransportLineItems(o)
		if n := harness.InventoryTotal(o); n > 0 {
			total += n
		}
	}
	return total
}

// ---------------------------------------------------------------------------
// the probe
//
// The audit marker is the shipped synchronous "drain the queue and re-classify
// now" trigger, and it is also what makes the mod print its heap. The [BBB-MAR]
// line goes FIRST so the assertion script can attribute the `post-audit` line
// that follows it to this leg and this iteration.
// ---------------------------------------------------------------------------

func probe(leg string, i int) {
	out.Open("leg=").S(leg).S(" i=").I(int64(i)).End()
	harness.Audit(surf(), 20, 4)
}

// ---------------------------------------------------------------------------
// the legs
//
// Each is a function of (phase, i): `phase` counts ticks within the iteration,
// `i` counts iterations. Every one of them leaves the world exactly as it found
// it, which is what makes the audits comparable.
// ---------------------------------------------------------------------------

// legCal is nothing but the probe: the constant every other leg pays once.
func legCal(name string) func(phase, i int) {
	return func(phase, i int) {
		if phase == 0 {
			probe(name, i)
		}
	}
}

// aRows is leg A's balancer: a 2->2 over FOUR parts -- two rows, a west part
// carrying each row's input and an east part carrying its output -- plus 8 input
// belts and 4 output belts, all in one tick each way, so one deferred flush
// builds it and one takes it down. That is the batching the mod is designed
// around.
const aRows = 2

func legA(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		for r := 0; r < aRows; r++ {
			putSoft(s, part, 0, cycleY+r, nil)
			putSoft(s, part, 1, cycleY+r, nil)
			for x := -4; x <= -1; x++ {
				putSoft(s, belt, x, cycleY+r, &east)
			}
			for x := 2; x <= 3; x++ {
				putSoft(s, belt, x, cycleY+r, &east)
			}
		}
	case 9:
		for r := 0; r < aRows; r++ {
			harness.KillAt(s, 0, cycleY+r, part, "")
			harness.KillAt(s, 1, cycleY+r, part, "")
			for x := -4; x <= -1; x++ {
				harness.KillAt(s, x, cycleY+r, "", beltType)
			}
			for x := 2; x <= 3; x++ {
				harness.KillAt(s, x, cycleY+r, "", beltType)
			}
		}
	case 11:
		probe("A", i)
	}
}

// legB is a belt two tiles above the keep balancer's top part. Inside the mod's
// two-tile neighbour gate, so the cluster is queued and re-classified; outside
// the edge, so the fingerprint matches and nothing is rebuilt.
func legB(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		putSoft(s, belt, 0, keepY-2, &east)
	case 1:
		harness.KillAt(s, 0, keepY-2, "", beltType)
	case 3:
		probe("B", i)
	}
}

// legC is one of keep's input belts, removed and put back. The edge list really
// moves both times, so an iteration is TWO full teardown-and-rebuilds.
func legC(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		harness.KillAt(s, -1, keepY, "", beltType)
	case 1:
		putSoft(s, belt, -1, keepY, &east)
	case 3:
		probe("C", i)
	}
}

// legD is a belt eighteen tiles from the nearest balancer part. The mod's guest
// is entered -- the engine's filter admits every belt-connectable on the map --
// and the in-guest position gate rejects it without a compile. What is left is
// the cost of being entered at all, which is the term a busy server multiplies
// by the largest number.
func legD(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		putSoft(s, belt, 18, farY, &east)
	case 1:
		harness.KillAt(s, 18, farY, "", beltType)
		probe("D", i)
	}
}

// legE is six entities in one tick and six out in one tick -- the event shape of
// a small blueprint paste and its undo.
//
// Two parts and four belts, which under the one-belt rule is a 1->1: the west
// part carries the input belt at x=11 and the east part the output belt at x=14,
// and the two outer belts (10 and 15) extend the runs without touching anything.
// It sits three tiles clear of leg D's tile so that neither leg can ever be
// inside the other's gate, whatever order they run in.
var (
	eParts = [...]int{12, 13}
	eBelts = [...]int{10, 11, 14, 15}
)

func legE(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		for _, x := range eParts {
			putSoft(s, part, x, farY, nil)
		}
		for _, x := range eBelts {
			putSoft(s, belt, x, farY, &east)
		}
	case 4:
		for _, x := range eParts {
			harness.KillAt(s, x, farY, part, "")
		}
		for _, x := range eBelts {
			harness.KillAt(s, x, farY, "", beltType)
		}
	case 7:
		probe("E", i)
	}
}

// legG is leg C on a 4x4 -- sixteen parts and a 32-entity network, against C's
// two parts and eleven. Two full teardown-and-rebuilds per iteration, exactly as
// C, so the ratio between them is the compile term's dependence on network size
// and nothing else.
func legG(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		harness.KillAt(s, -1, bigY, "", beltType)
	case 1:
		putSoft(s, belt, -1, bigY, &east)
	case 3:
		probe("G", i)
	}
}

// legF is the conservation leg. A saturated 2->2 over FOUR parts on a FINITE
// stock is grown by a fifth part (teardown and rebuild while the network is
// full), then taken apart entirely, then rebuilt. Every item in the band is
// counted while the network is DOWN -- which is the only moment the count is
// complete, because a teardown spills the hidden network onto the visible
// ground.
//
// The fifth part carries no belt and cannot: under the one-belt rule every part
// of the standing 2->2 already has one, so the only part that can be ADDED is an
// edgeless one below the block. The edit is still a real recompile -- the
// cluster's tile set moves, so the fingerprint does.
func legF(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		putSoft(s, part, 0, churnY+2, nil)
	case 6:
		harness.KillAt(s, 0, churnY+2, part, "")
		for r := 0; r <= 1; r++ {
			harness.KillAt(s, 0, churnY+r, part, "")
			harness.KillAt(s, 1, churnY+r, part, "")
		}
	case 7:
		probe("F", i)
	case 8:
		out.Open("churn i=").I(int64(i)).S(" total=").
			I(countBand(s, churnBand())).End()
		for r := 0; r <= 1; r++ {
			putSoft(s, part, 0, churnY+r, nil)
			putSoft(s, part, 1, churnY+r, nil)
		}
	}
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

type leg struct {
	Name   string
	Iters  int
	Period int
	Run    func(phase, i int)
}

// legs is the PLAN TABLE, and its order and its numbers are the suite. Nothing
// here may be reordered or retimed without re-recording every figure the
// assertion script prints.
var legs = []leg{
	{"cal", 10, 3, legCal("cal")},
	{"A", 100, 12, legA},
	{"calA", 10, 3, legCal("calA")},
	{"B", 100, 4, legB},
	{"C", 100, 4, legC},
	{"D", 100, 2, legD},
	{"E", 50, 8, legE},
	{"G", 100, 4, legG},
	{"F", 100, 10, legF},
	{"calZ", 10, 3, legCal("calZ")},
}

// gap is a pause between legs, so a leg's last flush cannot land inside the next
// one's first iteration.
const (
	gap   = 20
	start = 90
)

// plan is legs laid out on the tick line, and endTick is where the last one
// finishes.
//
// IT IS COMPUTED AT MODULE LOAD AND NOT IN `storage`, exactly as the Lua's was:
// the arithmetic is a fold over a constant table, so both phases of a run and
// every client of a multiplayer game reach the same answer without anything
// crossing a save.
type slot struct {
	leg
	T0 int
}

var (
	plan    []slot
	endTick uint64
)

func buildPlan() {
	t := start
	plan = make([]slot, 0, len(legs))
	for _, l := range legs {
		plan = append(plan, slot{leg: l, T0: t})
		t += l.Iters*l.Period + gap
	}
	endTick = uint64(t)
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	east = fkapi.DefinesDirectionEast()
	buildPlan()
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
		ChunkRadius: 8,
		X0:          -22,
		Y0:          -16,
		X1:          26,
		Y1:          rows + 16,
		Tile:        "grass-1",
	}.Make()

	// keep: never structurally touched. Legs B and C happen at its edge. A 2->2
	// over four parts -- west column in, east column out.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, keepY+r, nil, "")
		put(s, part, 1, keepY+r, nil, "")
		sourceInf(s, keepY+r)
		for x := -4; x <= -1; x++ {
			put(s, belt, x, keepY+r, &east, "")
		}
		for x := 2; x <= 3; x++ {
			put(s, belt, x, keepY+r, &east, "")
		}
		sink(s, keepY+r)
	}

	// cycle: only the ends are permanent. Leg A places and removes everything
	// between them.
	for r := 0; r < aRows; r++ {
		sourceInf(s, cycleY+r)
		sink(s, cycleY+r)
	}

	// big: the M2 `sat4` shape. Four inputs on the west face of a 4x4 block of
	// parts, four outputs on the east -- a 32-entity network, against keep's
	// eleven.
	//
	// THIS RIG DID NOT MOVE FOR THE ONE-BELT RULE and never had to: only the west
	// column carries an input and only the east column an output, the two
	// interior columns carry nothing at all, and no tile has ever had two. A 4x4
	// built the wide way was single-edge before there was a rule.
	for r := 0; r <= 3; r++ {
		for c := 0; c <= 3; c++ {
			put(s, part, c, bigY+r, nil, "")
		}
	}
	for r := 0; r <= 3; r++ {
		infinityChest(s, -4, bigY+r)
		put(s, loader, -3, bigY+r, &east, "output")
		for x := -2; x <= -1; x++ {
			put(s, belt, x, bigY+r, &east, "")
		}
		put(s, belt, 4, bigY+r, &east, "")
		put(s, loader, 5, bigY+r, &east, "input")
		harness.Place(s, harness.Piece{Name: "steel-chest", X: 6, Y: bigY + r})
	}

	// churn: a finite stock, so leg F's count is a conservation statement. A 2->2
	// over four parts, like keep.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, churnY+r, nil, "")
		put(s, part, 1, churnY+r, nil, "")
		sourceFinite(s, churnY+r, churnStock)
		for x := -4; x <= -1; x++ {
			put(s, belt, x, churnY+r, &east, "")
		}
		for x := 2; x <= 3; x++ {
			put(s, belt, x, churnY+r, &east, "")
		}
		sink(s, churnY+r)
	}

	// `--create` never reaches a tick, so the flush every build event armed would
	// otherwise land on the first tick of the benchmark. The marker drains it
	// here, exactly as the other suites' on_init does.
	probe("init", 0)
	out.Open("plan legs=").I(int64(len(legs))).S(" end_tick=").U(endTick).
		S(" stock=").I(countBand(s, churnBand())).End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	t := fkapi.ReadOnTick(ptr).Tick

	// ONE LEG PER TICK AT MOST, and the early return is the Lua's own: legs do
	// not overlap, so the first slot whose span contains this tick is the only
	// one that can. A slot this tick is PAST falls through to the next.
	for _, sl := range plan {
		if t < uint64(sl.T0) {
			continue
		}
		span := uint64(sl.Iters * sl.Period)
		if t < uint64(sl.T0)+span {
			d := t - uint64(sl.T0)
			sl.Run(int(d%uint64(sl.Period)), int(d/uint64(sl.Period))+1)
			return
		}
	}
	if t == endTick {
		out.Open("done end_tick=").U(endTick).S(" churn_final=").
			I(countBand(surf(), churnBand())).End()
	}
}

func main() {}
