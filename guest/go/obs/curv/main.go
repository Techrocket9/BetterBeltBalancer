// Command bbb-curv-test is the curve-upgrade suite's observer: a world built
// under the rule that had no curved exit, handed to the guest that has one.
//
// ---------------------------------------------------------------------------
// HOW A SAVE FROM BEFORE 0.3.3 IS BUILT BY A GUEST FROM AFTER IT
// ---------------------------------------------------------------------------
//
// Two halves, and neither works alone.
//
// The WATERMARK is test/run.sh's: the create phase runs a guest built
// `-tags prestate`, whose `fk_state_version` reports 0 the way every build up to
// 0.3.2 did by not exporting it at all, and the benchmark phase runs the shipped
// one. So the save carries the stamp a pre-curve save carries.
//
// The WORLD is this file's, and the prestate build does not help with it: its
// classifier is the shipped one, so a balancer compiled beside a curve-eligible
// belt would take that belt as an output. What is needed is a network compiled
// BEFORE those belts existed, which is `m3`'s `noev` idiom used to forge rather
// than to provoke -- `create_entity` with no `raise_built`, so the mod is never
// told and never re-classifies. So `fk_on_init` builds every balancer with no
// perpendicular belt anywhere near it, forces the compile with an audit marker,
// and only THEN lays the belts.
//
// ---------------------------------------------------------------------------
// THE RIGS
// ---------------------------------------------------------------------------
//
// Every belt in them is legal, inert and unremarkable under the OLD rule. The
// first three are the shape the pass was written for; the next three are the
// shapes an adversarial review found it could not see, each a perfectly ordinary
// thing for an old save to contain and each one that used to be recompiled under
// the new rule as though it had been built to it (guest/go/curveupg.go).
//
//	A  a 1 -> 1 with a THIRD, EDGELESS part below it, and a belt line whose head
//	   stands on that part's free south face running EAST across it. Old rule:
//	   nothing at all -- the belt is perpendicular, so classifySide falls
//	   through and the chest at the end of that line stays empty forever. New
//	   rule: the engine bends it towards an interface on that tile, so it is an
//	   OUTPUT and the balancer is 1 -> 2, at half the rate on the port the
//	   player has.
//	B  the same belt against a part that ALREADY carries its one belt -- the
//	   east output of a 1 -> 1. New rule: that tile carries two edges, which the
//	   single-edge rule forbids, so the standing network is condemned.
//	C  the control: a 1 -> 1 with no perpendicular belt anywhere near it. It
//	   must adopt exactly as it stood whatever happens to the others.
//	D  HALF-BUILT: two parts with an input and no output, so the old rule gave
//	   them no network at all, and a belt line running past. There is nothing
//	   standing for an adoption to compare, so the pass that compared could never
//	   have seen this one.
//	E  an old-rule 1 -> 1 whose OUTPUT BELT was mined while the mod was
//	   uninstalled -- destroyed with no event, like the curve belts. The standing
//	   interfaces describe a belt that is gone, so no reading of the world is a
//	   bijection with them.
//	F  an old-rule 2-in/1-out over THREE parts with one extra INPUT laid while
//	   the mod was uninstalled, and its curve belt on the tile that already
//	   carries the output. Neither reading matches; under the new rule that tile
//	   carries two edges, so the machine is condemned AND the multi-edge
//	   grandfather speaks for a save with one belt on every part.
//	G  A's shape on a SECOND FORCE, which is what says the checklist is per force
//	   rather than per save.
//	H  A's shape on a SECOND SURFACE.
//
// plus `ctrl`, a bare express belt from the same kind of source to the same
// kind of sink, which is the yardstick every rate here is read against.
//
// AND THE BAND, which is forty more of A's shape laid only to be COUNTED. The
// checklist the load hands a player is one `[gps=...]` per affected balancer
// built into a chat line, and about thirty-five of them fill one; every suite in
// the estate had at most seven, so no run had ever produced a second line -- and
// until 2026-09-15 there was no second line to produce, the list being cut at
// what fitted while the sentence above it went on naming every balancer (mod
// portal report). Forty-seven affected balancers is what makes `pings ==
// balancers` a real assertion. Nothing in the band is fed, drained or timed.
//
// NOTHING HERE ASSERTS. What is measured is the mod's own log and the chests,
// and this observer reports them.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

var out = harness.Line{Tag: "[BBB-CURV] "}

const (
	part   = "bbb-balancer-part"
	belt   = "express-transport-belt"
	loader = protos.CurvLoader
	surf   = "bbb-curv"
	// The second surface, which exists so that the rebuild's walk has to reach
	// past the one every other rig is on.
	surfB = "bbb-curv-b"
	// The second force, for the same reason one level across: the checklist is
	// one message per owning force and a one-force save cannot say so.
	forceB = "bbb-curv-force"
)

var dirE, dirS uint32

// The chests, in the order they are reported. A rig registry holds TILES rather
// than handles: what it keeps has to go on being true across the save between
// `--create` and `--benchmark`.
var (
	chests    []harness.XY
	chestName []string
	chestSurf []string
)

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirE = fkapi.DefinesDirectionEast()
	dirS = fkapi.DefinesDirectionSouth()
}

func flat() fkapi.LuaSurface {
	// Wide and tall enough for the band below `ctrl`: 256 is -128..128 and the
	// band's last row is at y=142.
	return harness.Flat{
		Name: surf, MapWidth: 512, MapHeight: 512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: 32},
		ChunkRadius: 5,
		X0:          -45, Y0: -4, X1: 45, Y1: 175,
		Tile: "grass-1",
	}.Make()
}

func flatB() fkapi.LuaSurface {
	return harness.Flat{
		Name: surfB, MapWidth: 256, MapHeight: 256,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: 0},
		ChunkRadius: 2,
		X0:          -10, Y0: -4, X1: 12, Y1: 8,
		Tile: "grass-1",
	}.Make()
}

// put is one piece the mod is TOLD about.
func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true, Force: rigForce})
}

// silent is one piece the mod is NOT told about: `create_entity` with no
// `raise_built`, which raises no event of any kind.
//
// IT IS THE WHOLE OF WHAT MAKES THIS SUITE POSSIBLE, and it is `m3`'s `noev`
// idiom rather than a new one: the world changes and the registry does not hear,
// so the standing network goes on describing the world as it was. Here that is
// not a defect being provoked but a save being FORGED -- a network compiled
// before these belts existed is byte for byte a network compiled before the rule
// existed. `E`'s mined output belt is the same act with the sign reversed.
func silent(s fkapi.LuaSurface, name string, x, y int, dir *uint32) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Force: rigForce})
}

// rigForce is which force the rig currently being laid belongs to. Every piece
// of a rig has to be on one force or the belts are not edges of the parts: the
// engine filters the edge query by the CLUSTER's force, so a player-force belt
// beside a second force's part is invisible to it by design.
var rigForce string

func beltsX(s fkapi.LuaSurface, from, to, y int, raise bool) {
	step := 1
	if to < from {
		step = -1
	}
	for i := from; ; i += step {
		if raise {
			put(s, belt, i, y, &dirE, "")
		} else {
			silent(s, belt, i, y, &dirE)
		}
		if i == to {
			break
		}
	}
}

func source(s fkapi.LuaSurface, x, y int) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y, Force: rigForce})
	harness.InfinityFilter(c, "iron-plate", "", 1000)
	put(s, loader, x+1, y, &dirE, "output")
}

func sink(s fkapi.LuaSurface, sname, name string, x, y int) {
	put(s, loader, x, y, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y, Force: rigForce})
	chests = append(chests, harness.XY{X: x + 1, Y: y})
	chestName = append(chestName, name)
	chestSurf = append(chestSurf, sname)
}

// The straight 1 -> 1 every rig is built around: a source feeding part (0,base)
// from the west and part (1,base) draining east into a chest.
func spine(s fkapi.LuaSurface, sname, name string, base int) {
	put(s, part, 0, base, nil, "")
	put(s, part, 1, base, nil, "")
	source(s, -5, base)
	beltsX(s, -3, -1, base, true)
	beltsX(s, 2, 4, base, true)
	sink(s, sname, name, 5, base)
}

// ctrl is the yardstick: the same source and the same sink with a plain belt run
// between them and no balancer at all.
func ctrl(s fkapi.LuaSurface, base int) {
	source(s, -5, base)
	beltsX(s, -3, 4, base, true)
	sink(s, surf, "ctrl", 5, base)
}

// THE BAND: how many, and where each one goes. Five columns twenty tiles apart
// and rows six apart, which leaves every rig's own three tiles clear of its
// neighbours and -- what the curve rule actually needs -- leaves the two tiles
// `curvesFromCluster` probes empty: the head's REAR, one west of it, and the
// tile beyond it on the far side.
const bandN = 40

func bandXY(i int) (int, int) { return -40 + (i%5)*20, 100 + (i/5)*6 }

// band lays one rig: a 1 -> 1 with an edgeless spare part under its input, fed
// and drained by one belt each so that it really compiles a network and is
// really adopted. No source, no sink and no chest -- what it is for is one more
// row on the checklist.
func band(s fkapi.LuaSurface) {
	for i := 0; i < bandN; i++ {
		ox, oy := bandXY(i)
		put(s, part, ox, oy, nil, "")
		put(s, part, ox+1, oy, nil, "")
		put(s, part, ox, oy+1, nil, "")
		put(s, belt, ox-1, oy, &dirE, "")
		put(s, belt, ox+2, oy, &dirE, "")
	}
}

const (
	baseA    = 0
	baseB    = 8
	baseC    = 16
	baseD    = 24
	baseE    = 32
	baseF    = 40
	baseG    = 48
	baseCtrl = 56
	baseH    = 0 // on surfB, which holds nothing else
)

// spines lays everything the mod is told about. It runs BEFORE the audit, so the
// networks compiled into the save know nothing of the belts below.
func spines(s, b fkapi.LuaSurface) {
	rigForce = harness.PlayerForce
	spine(s, surf, "A-main", baseA)
	// A's third part, edgeless: the tile the curve belt will stand against.
	put(s, part, 0, baseA+1, nil, "")
	spine(s, surf, "B-main", baseB)
	spine(s, surf, "C-main", baseC)

	// D, the HALF-BUILT one: an input and no output at all, so the old rule
	// compiled no network and there is nothing standing for anything to compare.
	put(s, part, 0, baseD, nil, "")
	put(s, part, 1, baseD, nil, "")
	source(s, -5, baseD)
	beltsX(s, -3, -1, baseD, true)

	// E, whose output belt is mined below. Ordinary until then.
	spine(s, surf, "E-main", baseE)

	// F, three parts, one in and one out, with the middle part left edgeless so
	// that the extra input laid below has somewhere legal to land.
	put(s, part, 0, baseF, nil, "")
	put(s, part, 1, baseF, nil, "")
	put(s, part, 2, baseF, nil, "")
	source(s, -5, baseF)
	beltsX(s, -3, -1, baseF, true)
	beltsX(s, 3, 5, baseF, true)
	sink(s, surf, "F-main", 6, baseF)

	// G, A's shape on a second force. Every piece of it, the source and the sink
	// included -- see rigForce.
	harness.CreateForce(forceB)
	rigForce = forceB
	spine(s, surf, "G-main", baseG)
	put(s, part, 0, baseG+1, nil, "")
	rigForce = harness.PlayerForce

	ctrl(s, baseCtrl)
	band(s)

	// H, A's shape on a second surface.
	spine(b, surfB, "H-main", baseH)
	put(b, part, 0, baseH+1, nil, "")
}

// curves lays the belts the OLD rule had nothing to say about, with no event --
// and mines the one belt E is about, the same way.
//
// Every head's two probe tiles are kept clear by the layout rather than by luck:
// `curvesFromCluster` reads the head's REAR and the tile beyond it on the far
// perpendicular side, and nothing in this world is ever built on either.
func curves(s, b fkapi.LuaSurface) {
	rigForce = harness.PlayerForce
	beltsX(s, 0, 4, baseA+2, false)
	sink(s, surf, "A-curve", 5, baseA+2)
	beltsX(s, 1, 4, baseB+1, false)
	sink(s, surf, "B-curve", 5, baseB+1)
	beltsX(s, 1, 4, baseD+1, false)
	sink(s, surf, "D-curve", 5, baseD+1)

	// E: the output belt goes, with no event, and the curve belt arrives.
	if o, ok := harness.FindOnTile(s, belt, 2, baseE); ok {
		harness.Destroy(o, false)
	} else {
		out.Open("could not find E's output belt to mine").End()
	}
	beltsX(s, 1, 4, baseE+1, false)
	sink(s, surf, "E-curve", 5, baseE+1)

	// F: an extra input on the middle part's north face, and the curve belt on
	// the tile that already carries the output.
	silent(s, belt, 1, baseF-1, &dirS)
	beltsX(s, 2, 5, baseF+1, false)
	sink(s, surf, "F-curve", 6, baseF+1)

	// The band's curve belts: one each, on the spare part's south face. One belt
	// is the whole classification -- the rest of A's line is there to carry items
	// to a chest, and nothing here counts items.
	for i := 0; i < bandN; i++ {
		ox, oy := bandXY(i)
		silent(s, belt, ox, oy+2, &dirE)
	}

	rigForce = forceB
	beltsX(s, 0, 4, baseG+2, false)
	sink(s, surf, "G-curve", 5, baseG+2)
	rigForce = harness.PlayerForce

	beltsX(b, 0, 4, baseH+2, false)
	sink(b, surfB, "H-curve", 5, baseH+2)
}

//go:wasmexport fk_on_init
func onInit() {
	s := flat()
	b := flatB()
	spines(s, b)
	// The synchronous drain: `--create` never reaches a tick, so without this
	// every network in the save would be compiled on the first tick of the
	// benchmark instead of into the save -- and this one has to be compiled
	// before the belts below exist, which is the whole point.
	harness.Audit(s, 8, 58)
	curves(s, b)
	out.Open("built rigs and then laid the curve belts with no event").End()
	// SEEDED BEFORE THE REPORT, so that the create log's `inside` line is the
	// seeding read back out of the world rather than the zero that precedes it.
	// That is what makes the count an identity a run can be held to -- a
	// `--create` never reaches a tick, so nothing has moved between the two --
	// and the assertion script compares them.
	seed()
	report("create")
}

// ours is everything this mod's compiler places: the hidden network proper and
// the edge interfaces standing on the visible part tiles. Both are drained by a
// teardown, so both are seeded and both are counted.
var ours = [...]string{"bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter"}

func oursFilter() fkapi.Value {
	vs := make([]fkapi.Value, 0, len(ours))
	for _, n := range ours {
		vs = append(vs, fkapi.OfString(n))
	}
	return fkapi.OfArray(vs...)
}

// mine is every entity of ours anywhere in the game, on every surface.
func mine() []fkapi.Object {
	all, err := fkapi.Game.Surfaces()
	if err != nil {
		return nil
	}
	f := oursFilter()
	var got []fkapi.Object
	for i := range all {
		s := fkapi.LuaSurface{Object: all[i].Val}
		found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Name: &f})
		if err != nil {
			continue
		}
		got = append(got, found...)
	}
	return got
}

// seed puts one item on every transport line of everything the compiler placed.
//
// THE SAVE IS A `--create` AND A `--create` NEVER REACHES A TICK, so without
// this every network in it is EMPTY and "the upgrade spilled nothing" would be a
// vacuous zero -- `mig21`'s own trap, met here for the same reason. Seeding makes
// it a KNOWN NUMBER instead: what is standing inside the compiler's entities
// when the benchmark opens can be compared with what was put there.
func seed() {
	stack := fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString("iron-plate")},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(1)},
	)
	n := int64(0)
	for _, o := range mine() {
		e := fkapi.LuaEntity{Object: o}
		cnt, err := e.GetMaxTransportLineIndex()
		if err != nil {
			continue
		}
		for i := uint32(1); i <= cnt; i++ {
			line, err := e.GetTransportLine(i)
			if err != nil {
				continue
			}
			if ok, err := (fkapi.LuaTransportLine{Object: line}).InsertAtBack(stack, nil); err == nil && ok {
				n++
			}
		}
	}
	out.Open("seeded ").I(n).S(" items into the compiler's own entities").End()
}

// ---------------------------------------------------------------------------
// what is reported
// ---------------------------------------------------------------------------

const (
	curveSetting    = "bbb-curved-exits"
	settingValueKey = "value"
)

// settingValue is `settings.global["bbb-curved-exits"].value` as the string the
// log line carries.
//
// REPORTED RATHER THAN ASSUMED, because every assertion about this pass rests on
// the write having landed: a run in which it silently did nothing would satisfy
// "the curve chests stayed empty" for the simplest of other reasons, namely that
// the belts were never read as outputs at all. It is the shipped guest's own
// idiom -- the raw LuaCustomTable handle plus one index read, two host calls,
// against a whole-dictionary attribute that would materialise every runtime
// setting in the game.
func settingValue() string {
	raw, err := fkapi.Settings.GlobalRaw()
	if err != nil {
		return "absent"
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(curveSetting))
	if err != nil || v.Tag != fkapi.TagMap {
		return "absent"
	}
	for i := range v.Map {
		if v.Map[i].Key.Tag != fkapi.TagString || v.Map[i].Key.Str != settingValueKey {
			continue
		}
		if v.Map[i].Val.Tag == fkapi.TagBool && v.Map[i].Val.Bool {
			return "true"
		}
		return "false"
	}
	return "absent"
}

// inside is what is standing in the compiler's entities right now, and how many
// of them there are.
func inside() (int64, int64) {
	ents := mine()
	total := int64(0)
	for _, o := range ents {
		total += harness.TransportLineItems(o)
	}
	return total, int64(len(ents))
}

// report is one line per chest plus the ground total, the setting and what is
// standing inside the networks, at a named moment.
//
// THE GROUND IS COUNTED OVER BOTH SURFACES, which is what makes its total a
// conserved quantity again: a rig on the second surface that spilled would
// otherwise be counted nowhere.
func report(tag string) {
	out.Open("setting t=").S(tag).S(" value=").S(settingValue()).End()
	s, okA := harness.SurfaceIfAny(surf)
	b, okB := harness.SurfaceIfAny(surfB)
	if !okA || !okB {
		out.Open("no surface at ").S(tag).End()
		return
	}
	for i := range chests {
		on := s
		if chestSurf[i] == surfB {
			on = b
		}
		n := harness.ChestCount(on, "steel-chest", chests[i].X, chests[i].Y)
		out.Open("chest t=").S(tag).S(" ").S(chestName[i]).S(" n=").I(n).End()
	}
	ground, stacks := int64(0), int64(0)
	// The box covers the band as well as the named rigs: a spill out there would
	// otherwise be counted nowhere and the total would stop being conserved.
	for _, o := range harness.EntitiesInOfType(s, harness.Box(-45, -6, 45, 150), "item-entity") {
		if _, c, ok := harness.GroundStack(o); ok {
			ground += c
			stacks++
		}
	}
	for _, o := range harness.EntitiesInOfType(b, harness.Box(-12, -6, 14, 10), "item-entity") {
		if _, c, ok := harness.GroundStack(o); ok {
			ground += c
			stacks++
		}
	}
	out.Open("ground t=").S(tag).S(" items=").I(ground).S(" stacks=").I(stacks).End()
	held, ents := inside()
	out.Open("inside t=").S(tag).S(" items=").I(held).S(" ents=").I(ents).End()
}

var schedule = []harness.Step{
	// After the rebuild and its deferred flush, which is where the decision is
	// written and the message spoken.
	{Tick: 30, Do: func() { report("early") }},
	{Tick: 300, Do: func() { report("t1") }},
	{Tick: 1100, Do: func() { report("t2") }},
	{Tick: 1140, Do: func() {
		if s, ok := harness.SurfaceIfAny(surf); ok {
			out.Open("final audit follows").End()
			harness.Audit(s, 8, 58)
		}
	}},
	{Tick: 1180, Do: func() { report("final") }},
}

//go:wasmexport fk_on_event
func onEvent(_ uint32, ptr uint32) {
	harness.Run(schedule, fkapi.ReadOnTick(ptr).Tick)
}

var _ = fk.Log

func main() {}
