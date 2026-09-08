// Command bbb-curv-test is the curve-upgrade suite's observer: a world built
// under the rule that had no curved exit, handed to the guest that has one.
//
// ---------------------------------------------------------------------------
// HOW A SAVE FROM BEFORE 0.3.3 IS BUILT BY A GUEST FROM AFTER IT
// ---------------------------------------------------------------------------
//
// The `upg` suite already creates a save with one guest and loads it with
// another (test/run.sh, bump_build), and that is half of what this needs. The
// other half is a world whose STANDING NETWORKS were compiled under the old
// reading, and the same source cannot build one under the new rule -- except by
// the one route `m3`'s `noev` rig established: place the curve belts with
// `create_entity` and NO `raise_built`, so the mod is never told and never
// re-classifies.
//
// So `fk_on_init` builds three balancers with no perpendicular belt anywhere
// near them, forces the compile with an audit marker, and only THEN lays the
// belts that the new rule would read as outputs. What is written into the save
// is three networks the classifier would have built with the curve arm switched
// off, in a world the curve arm has plenty to say about -- which is exactly a
// save made before the feature existed, and is what `guest/go/curveupg.go` is
// looking for.
//
// ---------------------------------------------------------------------------
// THE RIGS
// ---------------------------------------------------------------------------
//
// Every belt in them is legal, inert and unremarkable under the OLD rule:
//
//	A  a 1 -> 1 with a THIRD, EDGELESS part below it, and a belt line whose head
//	   stands on that part's free south face running EAST across it. Old rule:
//	   nothing at all -- the belt is perpendicular, so classifySide falls
//	   through and the chest at the end of that line stays empty forever. New
//	   rule: the engine bends it towards an interface on that tile, so it is an
//	   OUTPUT and the balancer is 1 -> 2, at half the rate on the port the
//	   player has.
//	B  the same belt against a part that ALREADY carries its one belt -- the
//	   east output of a 1 -> 1. Old rule: inert again. New rule: that tile
//	   carries two edges, which the single-edge rule forbids, so the standing
//	   network is condemned and the balancer stops.
//	C  the control: a 1 -> 1 with no perpendicular belt anywhere near it. It
//	   must adopt exactly as it stood whatever happens to the other two.
//
// plus `ctrl`, a bare express belt from the same kind of source to the same
// kind of sink, which is the yardstick every rate here is read against.
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
)

var dirE uint32

// The chests, in the order they are reported. A rig registry holds TILES rather
// than handles: what it keeps has to go on being true across the save between
// `--create` and `--benchmark`.
var (
	chests    []harness.XY
	chestName []string
)

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirE = fkapi.DefinesDirectionEast()
}

func flat() fkapi.LuaSurface {
	return harness.Flat{
		Name: surf, MapWidth: 256, MapHeight: 256,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: 24},
		ChunkRadius: 4,
		X0:          -10, Y0: -4, X1: 12, Y1: 44,
		Tile: "grass-1",
	}.Make()
}

// put is one piece the mod is TOLD about.
func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true})
}

// silent is one piece the mod is NOT told about: `create_entity` with no
// `raise_built`, which raises no event of any kind.
//
// IT IS THE WHOLE OF WHAT MAKES THIS SUITE POSSIBLE, and it is `m3`'s `noev`
// idiom rather than a new one: the world changes and the registry does not hear,
// so the standing network goes on describing the world as it was. Here that is
// not a defect being provoked but a save being FORGED -- a network compiled
// before these belts existed is byte for byte a network compiled before the rule
// existed.
func silent(s fkapi.LuaSurface, name string, x, y int, dir *uint32) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir})
}

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
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	harness.InfinityFilter(c, "iron-plate", "", 1000)
	put(s, loader, x+1, y, &dirE, "output")
}

func sink(s fkapi.LuaSurface, name string, x, y int) {
	put(s, loader, x, y, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	chests = append(chests, harness.XY{X: x + 1, Y: y})
	chestName = append(chestName, name)
}

// The straight 1 -> 1 every rig is built around: a source feeding part (0,base)
// from the west and part (1,base) draining east into a chest.
func spine(s fkapi.LuaSurface, name string, base int) {
	put(s, part, 0, base, nil, "")
	put(s, part, 1, base, nil, "")
	source(s, -5, base)
	beltsX(s, -3, -1, base, true)
	beltsX(s, 2, 4, base, true)
	sink(s, name, 5, base)
}

// ctrl is the yardstick: the same source and the same sink with a plain belt run
// between them and no balancer at all.
func ctrl(s fkapi.LuaSurface, base int) {
	source(s, -5, base)
	beltsX(s, -3, 4, base, true)
	sink(s, "ctrl", 5, base)
}

const (
	baseA    = 0
	baseB    = 12
	baseC    = 24
	baseCtrl = 32
)

// spines lays everything the mod is told about. It runs BEFORE the audit, so the
// networks compiled into the save know nothing of the belts below.
func spines(s fkapi.LuaSurface) {
	spine(s, "A-main", baseA)
	// A's third part, edgeless: the tile the curve belt will stand against.
	put(s, part, 0, baseA+1, nil, "")
	spine(s, "B-main", baseB)
	spine(s, "C-main", baseC)
	ctrl(s, baseCtrl)
}

// curves lays the belts the OLD rule had nothing to say about, with no event.
//
// A's head stands on (0, baseA+2), the south face of the spare part at
// (0, baseA+1). B's stands on (1, baseB+1), the south face of the part whose
// east face already holds the spine's output. Both tiles the curve rule reads
// are kept clear by the layout rather than by luck: the REAR of each head and
// the far perpendicular tile are never built on.
func curves(s fkapi.LuaSurface) {
	beltsX(s, 0, 4, baseA+2, false)
	sink(s, "A-curve", 5, baseA+2)
	beltsX(s, 1, 4, baseB+1, false)
	sink(s, "B-curve", 5, baseB+1)
}

//go:wasmexport fk_on_init
func onInit() {
	s := flat()
	spines(s)
	// The synchronous drain: `--create` never reaches a tick, so without this
	// every network in the save would be compiled on the first tick of the
	// benchmark instead of into the save -- and this one has to be compiled
	// before the belts below exist, which is the whole point.
	harness.Audit(s, 8, 40)
	curves(s)
	out.Open("built rigs and then laid the curve belts with no event").End()
	report("create")
	seed()
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
func report(tag string) {
	out.Open("setting t=").S(tag).S(" value=").S(settingValue()).End()
	s, ok := harness.SurfaceIfAny(surf)
	if !ok {
		out.Open("no surface at ").S(tag).End()
		return
	}
	for i := range chests {
		n := harness.ChestCount(s, "steel-chest", chests[i].X, chests[i].Y)
		out.Open("chest t=").S(tag).S(" ").S(chestName[i]).S(" n=").I(n).End()
	}
	ground := int64(0)
	stacks := 0
	for _, o := range harness.EntitiesInOfType(s, harness.Box(-12, -6, 14, 46), "item-entity") {
		if _, c, ok := harness.GroundStack(o); ok {
			ground += c
			stacks++
		}
	}
	out.Open("ground t=").S(tag).S(" items=").I(ground).S(" stacks=").I(int64(stacks)).End()
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
			harness.Audit(s, 8, 40)
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
