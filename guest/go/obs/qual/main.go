// Command bbb-qual-test builds balancers whose parts are UNCOMMON quality and
// drives every path where the mod under test asks the world for one of its own
// entities by name.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-qual-test/control.lua` was, rig for rig and log line for log
// line, with its little data stage compiled too (obs/qualdata). See
// agents/estate-port.md.
//
// # The defect class, which is invisible at normal quality
//
// `LuaSurface.find_entity` resolves a bare name as NORMAL QUALITY ONLY (the
// pinned API: "Normal quality will be used"), so a guest call site that looks a
// part up that way works perfectly on every normal-quality save and silently
// fails on a part a player built from a quality-rolled item. The mig suite's
// fidelity rig found the first such site (guest/go/legacy.go); this suite drives
// the other four (guest/go/findpart.go is the fix stated once):
//
//	qblk   skin.go's restyle: a 2x2 BLOCK of uncommon parts, saturated 2->2. The
//	       mod's own `[BBB] skin` line is the assertion surface -- the block's
//	       four variations are m1's known literals -- and the rig also answers the
//	       question nothing anywhere had asked: does an uncommon balancer BALANCE
//	       at all. (It does; every other lookup in the guest was already
//	       quality-blind.)
//	qcol   fastreplace.go's TRUE POSITIVE: a 1->1 column of four uncommon parts.
//	       A belt is fast-replaced onto an interior part mid-run; the part really
//	       is gone, so the guest must unregister it -- at any quality. This half
//	       passes on the unfixed guest too (nil for the wrong reason); the
//	       tripwire is qlone.
//	qlone  fastreplace.go's TRIPWIRE: one lone uncommon part, and a script builds
//	       a COLLIDING belt on its very tile (create_entity permits that; a player
//	       cannot do it). The part is still standing, so the registry must not be
//	       edited -- and the unfixed guest, asking at normal quality only, reads
//	       the part as gone and unregisters it.
//	qlim   limit.go's forceOfCluster: the over-limit block (64 input parts, one
//	       output part, one edgeless part -- the edge suite's `lim`, uncommon)
//	       given its sixty-fifth belt mid-run. The refusal must still be
//	       DELIVERED: the `told force` line is the arm a headless run can reach,
//	       and on the unfixed guest the lookup fails and the refusal is delivered
//	       to nobody. (revertOne shares the same lookup and the same fix; its
//	       observable needs a player, which no headless run has.)
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
// Three of the four already were -- `qblk` is a 2x2 whose west column carries
// the inputs and whose east column carries the outputs, `qcol` is a column whose
// two INTERIOR parts carry nothing (which is what makes the fast replace legal
// at all), and `qlone` has no belts. Only `qlim` moved: it was thirty-two parts
// with a belt on BOTH sides of each, which is the shape the rule forbids, and it
// is sixty-six parts now. See agents/single-edge.md.
//
// It ASSERTS NOTHING. test/assert-qual.py decides.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const (
	part = "bbb-balancer-part"
	belt = "express-transport-belt"
	// loader is this observer's own data stage's prototype (obs/qualdata); the
	// name is written down in both places because the two modules cannot share a
	// package.
	loader   = "bbbqual-loader"
	flowItem = "iron-plate"
	surfName = "bbb-qual"

	// uncommon is the whole suite in one string. One helper puts it on every
	// part, so a rig cannot forget.
	uncommon = "uncommon"
)

var out = harness.Line{Tag: "[BBB-QUAL] "}

// The three directions this suite needs, read through the GENERATED accessors,
// which resolve `defines.direction.*` BY NAME against the running game: a
// define's number is Factorio's own, is not in the API description at all, and
// nothing in this repository writes one down.
var north, east, west uint32

// Band bases. Far enough apart that no rig's probe belt is inside another rig's
// two-tile neighbour gate.
const (
	ctrlY  = 0
	qblkY  = 12
	qcolY  = 26
	qloneY = 40
	qlimY  = 52 // rows qlimY-4 (sink chest) .. qlimY+33 (the 65th belt)
	rows   = qlimY + 44
)

// ---------------------------------------------------------------------------
// the rig registry
// ---------------------------------------------------------------------------

// rig is one measured balancer and the chests its outputs drain into.
//
// IT HOLDS TILE POSITIONS AND NOT ENTITIES, which is the shipped guest's own
// rule reached from the other side: `fk_on_init` runs during `--create` and the
// samples are taken during `--benchmark`, so anything held here crosses a save.
type rig struct {
	Name   string
	Chests []harness.XY
}

// rigs is the registry the Lua kept in `storage.order` and `storage.rigs`, in
// registration order -- which IS the order the sample line reports in, and the
// order test/assert-qual.py parses.
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

// part is the one helper every rig places its parts through, so that no rig can
// forget the quality this whole suite is about.
func placePart(s fkapi.LuaSurface, x, y int) {
	harness.Place(s, harness.Piece{
		Name: part, X: x, Y: y, Quality: uncommon, Raise: true,
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
	put(s, loader, x+1, y, &east, "output")
}

// sink is a loader facing east into a steel chest, and the chest's tile is what
// the registry remembers.
func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	put(s, loader, x, y, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

func auditNow(tag string) {
	harness.Audit(surf(), -16, 0)
	out.Open("audited tag=").S(tag).End()
}

// reportQuality is THE ANTI-VACUITY LINE: what quality one part of a rig REALLY
// carries, read back from the world. A run where the quality silently did not
// take would pass every conservation check while proving nothing, so the
// assertion script requires this to say uncommon before it believes anything.
func reportQuality(name string, x, y int) {
	out.Open("quality rig=").S(name).S(" value=")
	found := harness.EntitiesIn(surf(), harness.InnerBox(x, y), part)
	if len(found) == 0 {
		out.S("MISSING").End()
		return
	}
	q, err := (fkapi.LuaEntity{Object: found[0]}).Quality()
	if err != nil {
		out.S("MISSING").End()
		return
	}
	n, err := (fkapi.LuaQualityPrototype{Object: q}).Name()
	if err != nil {
		out.S("MISSING").End()
		return
	}
	out.S(n).End()
}

// ---------------------------------------------------------------------------
// the mid-run probes
// ---------------------------------------------------------------------------

// collideProbe is qlone. `create_entity` with no `fast_replace` places a belt on
// the part's tile regardless of the collision mask (a script can; a player
// cannot). The part is STILL STANDING afterwards, so the guest's appearance
// check must leave the registry alone -- the unfixed guest, asking for the part
// at normal quality only, read it as gone and unregistered it.
func collideProbe() {
	s := surf()
	_, made := harness.PlaceSoft(s, harness.Piece{
		Name: belt, X: 0, Y: qloneY, Dir: &east, Raise: true,
	})
	standing := len(harness.EntitiesIn(s, harness.InnerBox(0, qloneY), part))
	out.Open("collide created=").B(made).S(" part-standing=").I(int64(standing)).End()
}

// replaceProbe is the TRUE REPLACE, qcol. The engine's own answer first
// (`can_fast_replace` is the question a player's cursor asks, and quality gating
// it would be worth knowing), then the replace itself: the engine destroys the
// part with NO EVENT and raises only the belt's build event, which is the whole
// reason fastreplace.go exists.
func replaceProbe() {
	s := surf()
	can := false
	if force, ok := harness.ForceByName(harness.PlayerForce); ok {
		v, err := s.CanFastReplace(fkapi.LuaSurfaceCanFastReplaceArgs{
			Name:      fkapi.OfString(belt),
			Position:  harness.Center(0, qcolY+1),
			Direction: &east,
			Force:     &force,
		})
		if err != nil {
			harness.Fatal("asking can_fast_replace", fk.LastError())
		}
		can = v
	} else {
		harness.Fatal("resolving the player force", fk.LastError())
	}
	out.Open("frep-can value=").B(can).End()

	_, made := harness.PlaceSoft(s, harness.Piece{
		Name: belt, X: 0, Y: qcolY + 1, Dir: &east, FastReplace: true, Raise: true,
	})
	gone := len(harness.EntitiesIn(s, harness.InnerBox(0, qcolY+1), part))
	out.Open("frep created=").B(made).S(" parts-left-on-tile=").I(int64(gone)).End()
}

// pokeAdd and pokeRemove are qblk's POKE. A belt inside the block's two-tile
// neighbour gate but NOT adjacent to it: the cluster is queued, the flush runs,
// the edge list has not moved and the shape has not moved -- so a healthy
// restyle says NOTHING, and the unfixed one (which can never find the uncommon
// parts and never records a variation as set) emits one more `[BBB] skin ...
// set=0` line per flush, forever. The assertion is on the COUNT of the block's
// skin lines.
func pokeAdd() { put(surf(), belt, 0, qblkY-2, &east, "") }

func pokeRemove() {
	found := harness.EntitiesIn(surf(), harness.InnerBox(0, qblkY-2), belt)
	if len(found) > 0 {
		harness.Destroy(found[0], true)
	}
}

// limAdd is THE SIXTY-FIFTH BELT, and it lands on the EDGELESS part below the
// block.
//
// Under the one-belt rule every one of the sixty-four input parts already
// carries its belt, so a sixty-fifth belt against any of them would be refused
// for the OTHER bound -- which is the `sedge` suite's business and would never
// reach `plan.Shape` at all. The spare part is what keeps this the PORT-limit
// gesture, and therefore what keeps it a test of `forceOfCluster`.
func limAdd() { put(surf(), belt, 0, qlimY+33, &north, "") }

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

func report(tick uint64) {
	s := surf()
	out.Open("sample tick=").U(tick)
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

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

var schedule = []harness.Step{
	{Tick: 150, Do: collideProbe},
	{Tick: 152, Do: func() { auditNow("post-collide") }},
	{Tick: 200, Do: replaceProbe},
	{Tick: 202, Do: func() { auditNow("post-replace") }},
	{Tick: 300, Do: limAdd},
	{Tick: 302, Do: func() { auditNow("post-lim") }},
	{Tick: 500, Do: pokeAdd},
	{Tick: 510, Do: pokeRemove},
	{Tick: 900, Do: func() { report(900) }},
	{Tick: 2100, Do: func() { report(2100) }},
	{Tick: 2140, Do: func() { auditNow("final") }},
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	north = fkapi.DefinesDirectionNorth()
	east = fkapi.DefinesDirectionEast()
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
		// ceil(rows/32) + 3, written out because rows is a constant and the
		// arithmetic was a `math.ceil` in the Lua.
		ChunkRadius: 6,
		X0:          -20,
		Y0:          -12,
		X1:          20,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()
	rigs = rigs[:0]

	// ctrl: the yardstick.
	source(s, -5, ctrlY)
	for x := -3; x <= 3; x++ {
		put(s, belt, x, ctrlY, &east, "")
	}
	register("ctrl", sink(s, 4, ctrlY))

	// qblk: a 2x2 block, two in west and two out east, saturated.
	for dy := 0; dy <= 1; dy++ {
		for dx := 0; dx <= 1; dx++ {
			placePart(s, dx, qblkY+dy)
		}
	}
	out2 := make([]harness.XY, 0, 2)
	for dy := 0; dy <= 1; dy++ {
		source(s, -5, qblkY+dy)
		for x := -3; x <= -1; x++ {
			put(s, belt, x, qblkY+dy, &east, "")
		}
		for x := 2; x <= 4; x++ {
			put(s, belt, x, qblkY+dy, &east, "")
		}
		out2 = append(out2, sink(s, 5, qblkY+dy))
	}
	register("qblk", out2...)
	reportQuality("qblk", 0, qblkY)

	// qcol: four parts in a column, one belt in at the top and one out at the
	// bottom, so it compiles (1->1) and its two INTERIOR parts carry no interface
	// -- which is what makes the fast replace at qcolY+1 legal.
	for dy := 0; dy <= 3; dy++ {
		placePart(s, 0, qcolY+dy)
	}
	put(s, belt, -1, qcolY, &east, "")
	put(s, belt, 1, qcolY+3, &east, "")
	reportQuality("qcol", 0, qcolY+1)

	// qlone: one part, nothing against it. A cluster with no edges is a
	// legitimate half-built state; what matters is that it is REGISTERED, so the
	// collide probe's wrong answer is visible as a cluster count.
	placePart(s, 0, qloneY)
	reportQuality("qlone", 0, qloneY)

	// qlim: the edge suite's `lim` block, uncommon, and SIXTY-SIX PARTS under the
	// one-belt rule where it used to be thirty-two.
	//
	//	(0, qlimY-1)             the OUTPUT part, its belt leaving north
	//	(0..1, qlimY..qlimY+31)  the 2x32 input block: sixty-four parts, each
	//	                         carrying one belt pointing inwards, which is
	//	                         P = plan.MaxPorts exactly
	//	(0, qlimY+32)            an EDGELESS part, where the sixty-fifth belt
	//	                         lands (see limAdd above)
	//
	// Three of the sixty-four inputs are fed, which is a belt or so of flow into a
	// machine with one way out: the output runs saturated and the assertion that
	// it kept running across the refusal has something to measure.
	placePart(s, 0, qlimY-1)
	for r := 0; r <= 31; r++ {
		placePart(s, 0, qlimY+r)
		placePart(s, 1, qlimY+r)
	}
	placePart(s, 0, qlimY+32)
	for r := 0; r <= 31; r++ {
		put(s, belt, -1, qlimY+r, &east, "")
		put(s, belt, 2, qlimY+r, &west, "")
	}
	for r := 0; r <= 2; r++ {
		source(s, -5, qlimY+r)
		for x := -3; x <= -2; x++ {
			put(s, belt, x, qlimY+r, &east, "")
		}
	}
	put(s, belt, 0, qlimY-2, &north, "")
	put(s, loader, 0, qlimY-3, &north, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: qlimY - 4})
	register("qlim", harness.XY{X: 0, Y: qlimY - 4})
	reportQuality("qlim", 0, qlimY)

	auditNow("t0")
	out.Open("init complete").End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	harness.Run(schedule, fkapi.ReadOnTick(ptr).Tick)
}

func main() {}
