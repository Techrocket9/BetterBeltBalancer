// Command bbb-m3-test drives every lifecycle path that can change what the
// compiler compiled from, and reports what happened.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-m3-test/control.lua` was, phase for phase, tick for tick and
// log line for log line, with its little data stage compiled too (obs/m3data).
// See agents/estate-port.md.
//
// It ASSERTS NOTHING. The guest's own `[BBB]` lines are the assertion surface
// and test/assert-m3.py decides whether they are right; an observer that
// computed the expected answer would be a second implementation of the thing
// under test. Where the guest cannot see the answer -- what a blueprint
// captured, how many items are on a surface -- the numbers are logged raw under
// `[BBB-M3]` and judged there too.
//
// # The rule every rig is built to
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
// Every edge of a cluster is an interface linked belt standing on the cluster's
// own tile, so a part carrying an input on its west side and an output on its
// east carried TWO belt-connectables on one tile, which 2.1's collision
// validator forbids. See agents/single-edge.md and guest/go/sedge.go.
//
// What that costs this suite is GEOMETRY AND NOTHING ELSE. Every column of parts
// becomes TWO columns -- a west part carrying the row's input and an east part
// carrying its output -- so a 2-in/2-out rig is four parts and `live` is eight.
// The lifecycle path each rig exists to drive is untouched; what moved is where
// the belts stand. Per row:
//
//	x=-6 source chest   -5 loader   -4..-1 belts   0 WEST PART   1 EAST PART
//	x=2..4 belts        5 sink loader              6 chest
//
// TWO PLACES NEEDED MORE THAN A RE-LAY, and both are the same shape: an edit
// that used to land on a working balancer's free face has no free face to land
// on any more. `phaseSilentNotice` lays its belt DIAGONALLY from the cluster
// instead -- inside the two-tile neighbour gate, so the cluster is
// re-classified, and adjacent to nothing, so no tile gains a second belt.
// `died` kills the EAST part of the second row rather than the west one, which
// is what keeps that row's output orphaned and exactly frozen.
//
// AND THE STRESS CHURN AVOIDS THE REFUSAL RATHER THAN EMBRACING IT. Its six
// randomised edits are aimed so that no tile can ever carry two belts: the two
// belt edits are the row's own single input and its own single output, and the
// part edit adds and removes an EDGELESS part below the west column. That is a
// decision and assert-m3.py asserts the negative -- zero one-belt-per-part
// refusals over the whole run. This suite's subject is the twelve lifecycle
// paths and its sharpest assertion is `drift=0 unbuilt=0` after 600 ticks of
// churn; a churn that generated refusals would make its compile, build and
// teardown counters a function of the rule rather than of the path under test,
// and would leave clusters standing refused at the final audit. The refusal has
// its own suite (`sedge`), which drives all three ways of reaching it.
//
// # The rigs, one per y band on a flat scratch surface
//
//	ctrl    a bare express belt, chest to chest: the throughput yardstick
//	live    4 in / 4 out over EIGHT parts, saturated, and NOTHING is ever done to
//	        it. It is the witness: every phase below happens around it, including
//	        deleting the surface its hidden network lives on, and it has to still
//	        be delivering four belts at the end
//	clone   2 in / 2 out over four parts; the source of clone_area/clone_brush
//	died    2 in / 2 out; the EAST part of the second row is killed with die()
//	        while the network is full, which orphans that row's output
//	bdie    2 in / 2 out; an input BELT is killed with die()
//	noev    2 in / 2 out; an input belt is destroy()ed with NO EVENT AT ALL --
//	        the incumbent's killer -- and then put back at the same tile, which
//	        is also the regression test for the removal window leaking
//	swap    2 in / 2 out; a belt's direction is changed with no event (what an
//	        undone rotation does), found by re-validation, then the same belt is
//	        fast-replaced with a slower tier
//	forceA  2 in / 2 out on force "player", with forceB immediately below it on
//	forceB  force "bbb-other". They touch and must NOT become one cluster
//	bp      2 in / 2 out; the area a blueprint is taken of
//	paste   the same blueprint pasted as real entities, all in one tick
//	ghost   ghosts of the same four parts, then revived
//	bots    a roboport with construction robots builds ghosts of four parts
//
// Plus three surfaces that are not rigs: bbb-m3-b (clone destination),
// bbb-m3-doomed (deleted mid-run), bbb-m3-s (the stress surface, whose items are
// counted against what was put into it after 600 ticks of randomised churn and a
// full teardown).
package main

import (
	"math"

	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

var out = harness.Line{Tag: "[BBB-M3] "}

const (
	part   = "bbb-balancer-part"
	belt   = "express-transport-belt"
	belt2  = "fast-transport-belt"
	loader = protos.M3Loader

	surfA      = "bbb-m3-a"
	surfB      = "bbb-m3-b"
	surfDoomed = "bbb-m3-doomed"
	surfStress = "bbb-m3-s"
	hidden     = "bbb-hidden"

	pitch      = 14
	otherForce = "bbb-other"

	stressStock = 2000

	// sinkX is the sink loader's column, one tile east of the last output belt.
	// Under the one-belt rule the output belts start at x=2 rather than x=1,
	// because x=1 is the east part.
	sinkX = 5
)

var stressRows = [4]int{0, 14, 28, 42}

func stressArea() fkapi.BoundingBox { return harness.Box(-30, -20, 30, 80) }

// hiddenBox is every slot the compiler can have put a network in.
func hiddenBox() fkapi.BoundingBox { return harness.Box(0, 0, 2048, 720) }

var dirE, dirW uint32

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirE = fkapi.DefinesDirectionEast()
	dirW = fkapi.DefinesDirectionWest()
}

// ---------------------------------------------------------------------------
// deterministic randomness
// ---------------------------------------------------------------------------

// seed is the churn's LCG state, carried in the guest heap and therefore across
// the save between `--create` and `--benchmark`, exactly as the Lua carried it
// in `storage`. `math.random` would not do: the churn schedule has to be
// identical on every run for the assertions to mean anything.
var seed float64

// rnd is the estate's own LCG, AND IT IS DELIBERATELY WRITTEN IN FLOATING POINT.
//
// THE OBVIOUS TRANSCRIPTION IS WRONG AND IS WRONG FROM THE FIRST VALUE. The Lua
// is `seed = (seed * 1103515245 + 12345) % 2147483648`, which reads as a
// textbook 31-bit LCG and is not one: Factorio's Lua is DOUBLES-ONLY, the
// product of a 31-bit seed and 1103515245 reaches 2.4e18, and a double carries
// only 53 bits of mantissa -- about 9e15. So the low nine bits of every product
// are ROUNDED AWAY before the modulus ever sees them, and every seed this
// generator produces is a multiple of 512.
//
// A Go transcription using uint64 would compute the arithmetic EXACTLY, which is
// a different generator: measured against the real Lua 5.2.1 (../FkLua/bin/lua52f,
// which is what this repository keeps for exactly this kind of question), the
// integer version diverges at the very first value and never re-converges. Every
// one of the churn's hundred steps would then pick a different row and a
// different action, and the suite's cumulative compile, build, teardown and
// create_entity counters -- which assert-m3.py checks against constants -- would
// all move.
//
// So this mirrors Lua's arithmetic rather than the LCG the Lua looks like:
//
//   - the multiply and the add are SEPARATE double operations, which is what
//     Lua's two bytecodes do. The explicit float64() conversion is what forbids
//     Go from contracting them into a fused multiply-add, which would be MORE
//     accurate and therefore wrong. (Wasm has no FMA instruction, so this is
//     belt and braces -- but the belt is free.)
//   - the modulus is `a - floor(a/b)*b`, which is how Lua 5.2's C source defines
//     `%` for numbers (`luai_nummod`), rather than fmod. Here they agree,
//     because b is 2^31 and division and multiplication by a power of two are
//     exact -- but the faithful form is the one that needs no such argument.
//
// Verified over 600 calls against lua52f: identical, value for value.
func rnd(n int) int {
	x := float64(seed*1103515245) + 12345
	seed = x - math.Floor(x/2147483648)*2147483648
	return int(math.Floor(seed/65536))%n + 1
}

// ---------------------------------------------------------------------------
// surfaces and pieces
// ---------------------------------------------------------------------------

func makeSurface(name string, rows int) fkapi.LuaSurface {
	return harness.Flat{
		Name:        name,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		ChunkRadius: uint32((rows+31)/32) + 3,
		X0:          -18,
		Y0:          -12,
		X1:          22,
		Y1:          rows + 12,
		Tile:        "grass-1",
	}.Make()
}

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, force string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Force: force, Raise: true,
	})
}

// putSoft is put for the churn, where a collision is a legitimate outcome and
// must not end the run.
func putSoft(s fkapi.LuaSurface, name string, x, y int, dir *uint32) bool {
	_, ok := harness.PlaceSoft(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Raise: true})
	return ok
}

// source is the estate's own: an infinity chest when the stock is unbounded, a
// steel chest holding a COUNTED stock when it is not, and the loader that drains
// it. A finite source is what makes the stress phase's conservation check a
// statement rather than an estimate.
func source(s fkapi.LuaSurface, x, y int, force string, count uint32, finite bool) {
	if finite {
		c := harness.Place(s, harness.Piece{Name: "steel-chest", X: x, Y: y, Force: force})
		harness.InsertInto(c, "iron-plate", count)
	} else {
		c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y, Force: force})
		harness.InfinityFilter(c, "iron-plate", "", 1000)
	}
	harness.Place(s, harness.Piece{
		Name: loader, X: x + 1, Y: y, Dir: &dirE, Type: "output", Force: force, Raise: true,
	})
}

func sink(s fkapi.LuaSurface, x, y int, force string) harness.XY {
	harness.Place(s, harness.Piece{
		Name: loader, X: x, Y: y, Dir: &dirE, Type: "input", Force: force, Raise: true,
	})
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y, Force: force})
	return harness.XY{X: x + 1, Y: y}
}

// ---------------------------------------------------------------------------
// rigs
// ---------------------------------------------------------------------------

// rig is what the estate kept in `storage.rigs`, with TILES where it kept
// entities: everything here is written in `fk_on_init` during `--create` and
// read during `--benchmark`, so it crosses a save.
type rig struct {
	name string
	base int
	outs int
	out  []outSlot
}

type outSlot struct {
	xy  harness.XY
	has bool
}

// rigOrder is the RIGS table, and the bases are (i-1)*pitch.
var rigOrder = []struct {
	name            string
	rows, ins, outs int
}{
	{name: "ctrl"},
	{name: "live", rows: 4, ins: 4, outs: 4},
	{name: "clone", rows: 2, ins: 2, outs: 2},
	{name: "died", rows: 2, ins: 2, outs: 2},
	{name: "bdie", rows: 2, ins: 2, outs: 2},
	{name: "noev", rows: 2, ins: 2, outs: 2},
	{name: "swap", rows: 2, ins: 2, outs: 2},
	{name: "forceA", rows: 2, ins: 2, outs: 2},
	{name: "bp", rows: 2, ins: 2, outs: 2},
	{name: "paste", outs: 2},
	{name: "ghost", outs: 2},
	{name: "bots", outs: 2},
}

// rigs is the built registry. It is a SLICE and not a map, and `report` sorts it
// by name: the estate's Lua kept a table and sorted `pairs` before emitting,
// which is the same statement with the determinism argued rather than built in.
var rigs []rig

func rigbase(name string) int {
	for i := range rigOrder {
		if rigOrder[i].name == name {
			return i * pitch
		}
	}
	return 0
}

func rigOf(name string) *rig {
	for i := range rigs {
		if rigs[i].name == name {
			return &rigs[i]
		}
	}
	return nil
}

func buildRig(s fkapi.LuaSurface, base, rows, ins, outs int, force string, stock uint32, finite bool) rig {
	r := rig{base: base, outs: outs, out: make([]outSlot, outs)}
	// TWO PER ROW: the west one carries the row's input and the east one its
	// output. Parts first, belts after, so the belt-adjacency trigger is on the
	// critical path of every rig.
	for i := 0; i < rows; i++ {
		put(s, part, 0, base+i, nil, force)
		put(s, part, 1, base+i, nil, force)
	}
	for i := 0; i < ins; i++ {
		source(s, -6, base+i, force, stock, finite)
		for x := -4; x <= -1; x++ {
			put(s, belt, x, base+i, &dirE, force)
		}
	}
	for i := 0; i < outs; i++ {
		for x := 2; x <= 4; x++ {
			put(s, belt, x, base+i, &dirE, force)
		}
		r.out[i] = outSlot{xy: sink(s, sinkX, base+i, force), has: true}
	}
	return r
}

// feedAndDrain gives a rig that was built some other way (a paste, a revive, a
// robot) the belts and chests that make it measurable.
func feedAndDrain(name string, rows int, belts bool) {
	s := harness.Surface(surfA)
	r := rigOf(name)
	if r == nil {
		return
	}
	for i := 0; i < rows; i++ {
		source(s, -6, r.base+i, "", 0, false)
		if belts {
			for x := -4; x <= -1; x++ {
				put(s, belt, x, r.base+i, &dirE, "")
			}
			for x := 2; x <= 4; x++ {
				put(s, belt, x, r.base+i, &dirE, "")
			}
		} else {
			// The paste brought its own belts from x=-2 to x=4 (create_blueprint
			// takes every entity whose box INTERSECTS the area, so the belt one
			// tile outside each end came too); only the run-up from the loader
			// is missing.
			for x := -4; x <= -3; x++ {
				put(s, belt, x, r.base+i, &dirE, "")
			}
		}
		if i < len(r.out) {
			r.out[i] = outSlot{xy: sink(s, sinkX, r.base+i, ""), has: true}
		}
	}
}

// ---------------------------------------------------------------------------
// counting
// ---------------------------------------------------------------------------

// countSurface totals every item standing on one surface inside a box: what an
// item-entity is holding, what is in a transport line, and what is in a chest.
//
// A surface that is not there counts ZERO rather than being an error, which is
// the Lua's `if not (s and s.valid)` -- two phases here delete a surface on
// purpose and the counts either side of them have to keep working.
func countSurface(s fkapi.LuaSurface, area fkapi.BoundingBox, ok bool) int64 {
	if !ok {
		return 0
	}
	var total int64
	for _, e := range harness.EntitiesIn(s, area, "") {
		if harness.EntityTypeIs(e, "item-entity") {
			if _, n, got := harness.GroundStack(e); got {
				total += n
			}
			continue
		}
		total += harness.TransportLineItems(e)
		for _, item := range harness.ChestContents(e) {
			total += int64(item.Count)
		}
	}
	return total
}

func countOn(name string, area fkapi.BoundingBox) int64 {
	s, ok := harness.SurfaceIfAny(name)
	return countSurface(s, area, ok)
}

// countEverywhere is the visible box PLUS the whole hidden surface, and the
// second half is not belt and braces.
//
// Killing ONE part of a two-part balancer leaves a cluster behind, so it is a
// recompile and not a removal: the drained items go back INSIDE the network the
// flush rebuilds (2026-08-02, guest/go/carry.go) instead of onto the floor
// beside it. A visible-only count reads that correct behaviour as a loss -- it
// did, by two items, which is what found this line. What is asserted is
// conservation across the pair.
func countEverywhere(area fkapi.BoundingBox) int64 {
	return countOn(surfA, area) + countOn(hidden, hiddenBox())
}

// ---------------------------------------------------------------------------
// phases
// ---------------------------------------------------------------------------

func mark(n int, what string) {
	out.Open("phase=").I(int64(n)).S(" ").S(what).End()
}

func auditMarker(y int) { harness.Audit(harness.Surface(surfA), 16, y) }

// bp is the blueprint's entity list, held from phase 1 to phase 3 -- across two
// ticks, in the guest heap, where the Lua held it in `storage`.
var bp []fkapi.Value

// vget reads one key out of a dynamic map value. A BlueprintEntity crosses the
// boundary as a Value rather than as a generated struct, because a blueprint's
// contents are whatever the entities in it were.
func vget(v fkapi.Value, key string) (fkapi.Value, bool) {
	if v.Tag != fkapi.TagMap {
		return fkapi.Value{}, false
	}
	for _, kv := range v.Map {
		if kv.Key.Tag == fkapi.TagString && kv.Key.Str == key {
			return kv.Val, true
		}
	}
	return fkapi.Value{}, false
}

// vpos reads a MapPosition out of a dynamic value, which the engine may hand
// over either as {x = ..., y = ...} or as an array of two numbers.
func vpos(v fkapi.Value) (x, y float64) {
	if vx, ok := vget(v, "x"); ok {
		vy, _ := vget(v, "y")
		return vx.Number, vy.Number
	}
	if v.Tag == fkapi.TagArray && len(v.Array) >= 2 {
		return v.Array[0].Number, v.Array[1].Number
	}
	return 0, 0
}

// 1. What a blueprint of a compiled balancer captures.
//
// The hidden prototypes carry `not-blueprintable`, so the claim is that a
// blueprint over a live balancer contains the visible parts and belts and NONE
// of the network -- including the visible linked-belt interfaces, which sit on
// the cluster's own tiles and would otherwise travel with the blueprint and
// resurrect a network the compiler does not know about.
func phaseBlueprintCapture() {
	mark(1, "blueprint capture over a compiled balancer")
	s := harness.Surface(surfA)
	base := rigbase("bp")
	invO, err := fkapi.Game.CreateInventory(1, nil)
	if err != nil {
		harness.Fatal("game.create_inventory", "")
		return
	}
	inv := fkapi.LuaInventory{Object: invO}
	stackO, err := inv.Get(1)
	if err != nil {
		harness.Fatal("inventory[1]", "")
		return
	}
	stack := fkapi.LuaItemStack{Object: stackO}
	bpStack := fkapi.OfMap(fkapi.KeyValue{
		Key: fkapi.OfString("name"), Val: fkapi.OfString("blueprint")})
	if _, err := stack.SetStack(&bpStack); err != nil {
		harness.Fatal("set_stack blueprint", "")
		return
	}
	force, _ := harness.ForceByName("player")
	if _, err := stack.CreateBlueprint(fkapi.LuaItemCommonCreateBlueprintArgs{
		Surface: s.Object,
		Force:   force,
		Area:    harness.Box(-1, float64(base), 4, float64(base+2)),
	}); err != nil {
		harness.Fatal("create_blueprint", "")
		return
	}
	ents, _ := stack.GetBlueprintEntities()
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if n, ok := vget(e, "name"); ok && n.Tag == fkapi.TagString {
			names = append(names, n.Str)
		}
	}
	harness.SortStrings(names)
	out.Open("bp captured=[")
	for i, n := range names {
		if i > 0 {
			out.S(" ")
		}
		out.S(n)
	}
	out.S("]").End()
	bp = ents
	inv.Destroy()
}

// 2. Ghosts, then a revive.
//
// A ghost is what a blueprint actually produces in a real game, and a ghost is
// NOT a part: the build event carries an `entity-ghost` whose name is
// "entity-ghost", so the registry must not grow. Reviving it with raise_revive
// is the script_raised_revive path; the bots phase below is the
// on_robot_built_entity one.
func phaseGhostRevive() {
	mark(2, "ghosts placed, then revived")
	s := harness.Surface(surfA)
	base := rigbase("ghost")
	ghosts := make([]fkapi.Object, 0, 4)
	for i := 0; i <= 1; i++ {
		for x := 0; x <= 1; x++ {
			o, ok := harness.PlaceSoft(s, harness.Piece{
				Name: "entity-ghost", InnerName: part, X: x, Y: base + i, Raise: true,
			})
			if ok {
				ghosts = append(ghosts, o)
			}
		}
	}
	out.Open("ghosts placed=").I(int64(len(ghosts))).End()
	revived := 0
	yes := true
	for _, g := range ghosts {
		_, e, _, err := (fkapi.LuaEntity{Object: g}).Revive(
			fkapi.LuaEntityReviveArgs{RaiseRevive: &yes})
		if err == nil && e != nil {
			revived++
		}
	}
	out.Open("ghosts revived=").I(int64(revived)).End()
	feedAndDrain("ghost", 2, true)
}

// 3. A blueprint pasted as REAL entities, all inside one tick.
//
// What an editor paste, and any mod that builds a blueprint directly, produces.
// THREE markers, because the guest batches (fk.defer, FKLUA-GAPS.md item 12,
// fixed upstream): `paste-begin` .. `paste-end` brackets the tick the entities
// arrive in and must contain NO compile at all, and `paste-flushed` (tick 92,
// two ticks later so no handler-ordering question arises) closes the window the
// single deferred compile has to land in. assert-m3.py counts compiles in both.
func phaseInstantPaste() {
	mark(3, "blueprint pasted as real entities in one tick")
	s := harness.Surface(surfA)
	base := rigbase("paste")
	// Blueprint entity positions are relative to the blueprint's own origin,
	// which is not the source rig's origin. Anchoring on the PART column rather
	// than on the bounding box puts every piece exactly where the source rig had
	// it, so the pasted rig can be fed and drained like any other.
	px, py := math.Inf(1), math.Inf(1)
	for _, e := range bp {
		n, ok := vget(e, "name")
		if !ok || n.Str != part {
			continue
		}
		p, _ := vget(e, "position")
		x, y := vpos(p)
		px = math.Min(px, math.Floor(x))
		py = math.Min(py, math.Floor(y))
	}
	placed := 0
	out.Open("paste-begin").End()
	for _, e := range bp {
		n, _ := vget(e, "name")
		p, _ := vget(e, "position")
		x, y := vpos(p)
		var dir *uint32
		if d, ok := vget(e, "direction"); ok && d.Tag == fkapi.TagNumber {
			v := uint32(d.Number)
			dir = &v
		}
		if putSoft(s, n.Str, int(math.Floor(x)-px), int(math.Floor(y)-py)+base, dir) {
			placed++
		}
	}
	out.Open("paste-end").End()
	out.Open("paste placed=").I(int64(placed)).S(" of=").I(int64(len(bp))).End()
	feedAndDrain("paste", 2, false)
}

func phasePasteFlushed() { out.Open("paste-flushed").End() }

// 4/5. A construction robot builds a ghost part.
func phaseBotsStart() {
	mark(4, "roboport and construction robots given ghosts to build")
	s := harness.Surface(surfA)
	base := rigbase("bots")
	port := harness.Place(s, harness.Piece{Name: "roboport", X: 12, Y: base + 1})
	if inv, err := (fkapi.LuaControl{Object: port}).GetInventory(
		fkapi.DefinesInventoryRoboportRobot()); err == nil && inv != nil {
		(fkapi.LuaInventory{Object: *inv}).Insert(fkapi.OfMap(
			fkapi.KeyValue{Key: fkapi.OfString("name"),
				Val: fkapi.OfString("construction-robot")},
			fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(6)}))
	}
	chest := harness.Place(s, harness.Piece{Name: "storage-chest", X: 9, Y: base})
	harness.InsertInto(chest, part, 8)
	portPlaced = true
	for i := 0; i <= 1; i++ {
		for x := 0; x <= 1; x++ {
			harness.PlaceSoft(s, harness.Piece{
				Name: "entity-ghost", InnerName: part, X: x, Y: base + i, Raise: true,
			})
		}
	}
}

func phaseBotsCheck() {
	mark(5, "what the robots built")
	s := harness.Surface(surfA)
	base := rigbase("bots")
	box := harness.Box(-1, float64(base), 2, float64(base+2))
	n := len(harness.EntitiesIn(s, box, part))
	out.Open("bots built=").I(int64(n)).End()
	if n > 0 {
		feedAndDrain("bots", 2, true)
	}
}

// 6/7. Cloning an area, and cloning a brush.
//
// The claims: the cloned parts form their OWN cluster with a freshly compiled
// network; the visible interfaces that the clone copied along with them are
// destroyed rather than left standing as a second, untracked network; and no
// hidden-surface entity is cloned at all.
func countLeaked(s fkapi.LuaSurface, box fkapi.BoundingBox) int64 {
	names := fkapi.OfArray(fkapi.OfString("bbb-belt"), fkapi.OfString("bbb-splitter"),
		fkapi.OfString("bbb-lane-splitter"))
	n, err := s.CountEntitiesFiltered(fkapi.EntitySearchFilters{Area: &box, Name: &names})
	if err != nil {
		return 0
	}
	return int64(n)
}

func phaseCloneArea() {
	mark(6, "clone_area of a compiled balancer onto another surface")
	a, b := harness.Surface(surfA), harness.Surface(surfB)
	base := rigbase("clone")
	yes, no := true, false
	if err := a.CloneArea(fkapi.LuaSurfaceCloneAreaArgs{
		SourceArea:               harness.Box(-7, float64(base), 7, float64(base+2)),
		DestinationArea:          harness.Box(-7, 2, 7, 4),
		DestinationSurface:       &b.Object,
		CloneEntities:            &yes,
		CloneTiles:               &no,
		ClearDestinationEntities: &yes,
	}); err != nil {
		harness.Fatal("clone_area", "")
	}
	box := harness.Box(-12, -4, 14, 12)
	out.Open("clone-area parts=").I(int64(len(harness.EntitiesIn(b, box, part)))).
		S(" leaked=").I(countLeaked(b, box)).End()
}

func phaseCloneBrush() {
	mark(7, "clone_brush of the same balancer")
	a, b := harness.Surface(surfA), harness.Surface(surfB)
	base := rigbase("clone")
	positions := make([]fkapi.TilePosition, 0, 15*2)
	for x := -7; x <= 7; x++ {
		for y := base; y <= base+1; y++ {
			positions = append(positions, fkapi.TilePosition{X: int32(x), Y: int32(y)})
		}
	}
	yes, no := true, false
	if err := a.CloneBrush(fkapi.LuaSurfaceCloneBrushArgs{
		SourceOffset:             fkapi.TilePosition{X: 0, Y: int32(base)},
		DestinationOffset:        fkapi.TilePosition{X: 0, Y: 24},
		SourcePositions:          positions,
		DestinationSurface:       &b.Object,
		CloneEntities:            &yes,
		CloneTiles:               &no,
		ClearDestinationEntities: &yes,
	}); err != nil {
		harness.Fatal("clone_brush", "")
	}
	box := harness.Box(-12, 20, 14, 34)
	out.Open("clone-brush parts=").I(int64(len(harness.EntitiesIn(b, box, part)))).
		S(" leaked=").I(countLeaked(b, box)).End()
}

// 8/9. A belt whose direction changes with NO EVENT, and the re-validation.
//
// `entity.direction = x` raises nothing at all. That is exactly what an undone
// ROTATION does to the world, and the reason the undo handler re-validates
// rather than trusting the actions array it is handed.
//
// The undo EVENT itself is out of reach here and the harness does not pretend
// otherwise: a headless --create has no player to scope by, LuaUndoRedoStack can
// be read and edited but never APPLIED, and script.raise_event refuses
// ("on_undo_applied can't be raised through script"). What is tested is the
// machinery the handler calls -- a pass that re-classifies every cluster from
// the world -- reached here through the audit marker, which runs the same
// classify-and-repair and additionally reports what it found.
func phaseSilentRotate() {
	mark(8, "belt direction changed with no event")
	s := harness.Surface(surfA)
	if b, ok := harness.FindAt(s, -1, rigbase("swap"), "", "transport-belt"); ok {
		(fkapi.LuaEntity{Object: b}).SetDirection(dirW)
	}
	out.Open("swap: input belt turned around silently").End()
}

func phaseRevalidate() {
	mark(9, "re-validation finds the silent rotation")
	auditMarker(0)
}

// 10. Fast-replacing a belt with a different tier.
//
// create_entity{fast_replace} destroys the belt that was there and raises no
// mine event for it, so the BUILD event alone has to land the recompile. It is
// the same shape as a construction robot performing an upgrade order, which is
// why the bot-upgrade path needs no handler of its own.
func phaseFastReplace() {
	mark(10, "input belt fast-replaced with a slower tier")
	s := harness.Surface(surfA)
	base := rigbase("swap")
	if b, ok := harness.FindAt(s, -1, base, "", "transport-belt"); ok {
		(fkapi.LuaEntity{Object: b}).SetDirection(dirE)
	}
	harness.PlaceSoft(s, harness.Piece{
		Name: belt2, X: -1, Y: base, Dir: &dirE, FastReplace: true, Raise: true,
	})
	name := "gone"
	if now, ok := harness.FindAt(s, -1, base, "", "transport-belt"); ok {
		if n, err := (fkapi.LuaEntity{Object: now}).Name(); err == nil {
			name = n
		}
	}
	out.Open("swap: belt is now ").S(name).End()
}

// 11/12/13. A belt destroyed with NO EVENT WHATSOEVER: the incumbent's killer.
//
// destroy{} with no raise_destroy is what a badly-behaved mod does, and no
// amount of event handling can see it. The claim is structural: the guest holds
// no reference to that belt so nothing goes stale, the network keeps running
// (its linked belts do not care that a visible belt vanished -- items simply
// stop arriving on that port), and the next event that touches the cluster
// re-derives the edge list from the world and puts it right.
//
// Phase 13 then puts a belt back on the SAME TILE. If the removal window ever
// leaked -- if a position armed by an earlier removal were still armed -- that
// belt would be classified as absent and this rig would never come back to full
// rate. It is the M3 regression test for FKLUA item 9.
func phaseSilentDestroy() {
	mark(11, "input belt destroyed with destroy{} -- no event of any kind")
	s := harness.Surface(surfA)
	if b, ok := harness.FindAt(s, -1, rigbase("noev"), "", "transport-belt"); ok {
		harness.Destroy(b, false)
	}
	out.Open("noev: belt gone, no event raised").End()
}

// The placement is DIAGONAL from the nearest part and that is the whole of what
// the one-belt rule changed here. It used to be a south-facing belt on the top
// west part's north face, which was an extra input -- and under the rule that
// part already carries its own input on its west side, so the same belt would
// now be REFUSED and this phase would measure a refusal instead of a
// re-classification. A belt at (-1, base-1) is inside the two-tile neighbour
// gate, so the cluster is queued and re-derived from the world; it is
// orthogonally adjacent to no part at all, so no tile gains a second belt. The
// fingerprint moves anyway, because the belt phase 11 destroyed silently is
// missing from the classification this placement provokes -- which is the thing
// under test.
func phaseSilentNotice() {
	mark(12, "an unrelated placement inside the cluster's neighbour gate")
	put(harness.Surface(surfA), belt, -1, rigbase("noev")-1, &dirE, "")
	out.Open("noev: unrelated placement made, cluster re-classified").End()
}

func phaseSilentRecover() {
	mark(13, "the destroyed belt is put back on the same tile")
	put(harness.Surface(surfA), belt, -1, rigbase("noev"), &dirE, "")
	out.Open("noev: belt replaced").End()
}

// 14/15. Death.
//
// The teardown that takes the hidden network's items back out is DEFERRED to the
// next tick like every other recompile, so an audit marker forces the flush
// inside this tick and the two counts stay one atomic sample apart. (Counting a
// tick later would work for the assertion and would stop being a measurement of
// the teardown: items would have moved for other reasons in between.)
func phasePartDied() {
	mark(14, "a balancer part killed with die() while its network is full")
	s := harness.Surface(surfA)
	base := rigbase("died")
	box := harness.Box(-20, float64(base-6), 20, float64(base+10))
	before := countEverywhere(box)
	// THE EAST PART OF THE SECOND ROW, not the west one. Under the one-belt rule
	// a row's output stands against its east part, so killing that part is what
	// takes the row's OUTPUT off the machine and leaves its chest orphaned -- the
	// property this rig has always been about. Killing the west part would take
	// an INPUT off instead and leave both outputs live at half a belt each, which
	// is a different measurement. Three parts survive and stay one cluster: the
	// row above is intact and the east part below it is still attached to it.
	if p, ok := harness.FindAt(s, 1, base+1, part, ""); ok {
		(fkapi.LuaEntity{Object: p}).Die(nil, nil)
	}
	auditMarker(1)
	out.Open("died: part killed, items ").I(before).S(" -> ").I(countEverywhere(box)).End()
}

func phaseBeltDied() {
	mark(15, "an input belt killed with die()")
	s := harness.Surface(surfA)
	if b, ok := harness.FindAt(s, -1, rigbase("bdie"), "", "transport-belt"); ok {
		(fkapi.LuaEntity{Object: b}).Die(nil, nil)
	}
	out.Open("bdie: input belt killed").End()
}

// 16/17. Surfaces.
func phaseDeleteSurface() {
	mark(16, "the surface a balancer stands on is deleted")
	deleteSurface(surfDoomed)
}

func phaseDeleteHidden() {
	mark(17, "the HIDDEN surface is deleted by another mod")
	deleteSurface(hidden)
}

func deleteSurface(name string) {
	if s, ok := harness.SurfaceIfAny(name); ok {
		if _, err := fkapi.Game.DeleteSurface(s.Object); err != nil {
			harness.Fatal("delete_surface "+name, "")
		}
	}
}

// 18/19. The stress: 600 ticks of randomised churn around four clusters, then
// everything comes down so that every item in every hidden network is handed
// back and the total can be compared with what was put in.
func phaseStressStart() {
	mark(18, "stress begins")
	out.Open("stress inserted=").I(int64(len(stressRows) * 2 * stressStock)).End()
}

// EVERY ONE OF THE SIX IS AIMED SO THAT NO TILE CAN CARRY TWO BELTS. The two
// belt edits are the row's own single input (west of the west part) and its own
// single output (east of the east part), so a tile that gains a belt had none;
// the part edit adds and removes an EDGELESS part below the west column, whose
// three free faces are bare ground. See the header for why this churn avoids the
// one-belt-per-part refusal rather than embracing it.
func stressStep() {
	s := harness.Surface(surfStress)
	row := stressRows[rnd(len(stressRows))-1]
	switch rnd(6) {
	case 1:
		if b, ok := harness.FindAt(s, -1, row, "", "transport-belt"); ok {
			harness.Destroy(b, true)
		}
	case 2:
		if _, ok := harness.FindAt(s, -1, row, "", "transport-belt"); !ok {
			putSoft(s, belt, -1, row, &dirE)
		}
	case 3:
		if b, ok := harness.FindAt(s, 2, row+1, "", "transport-belt"); ok {
			harness.Destroy(b, true)
		}
	case 4:
		if _, ok := harness.FindAt(s, 2, row+1, "", "transport-belt"); !ok {
			putSoft(s, belt, 2, row+1, &dirE)
		}
	case 5:
		if _, ok := harness.FindAt(s, 0, row+2, part, ""); !ok {
			putSoft(s, part, 0, row+2, nil)
		}
	default:
		if p, ok := harness.FindAt(s, 0, row+2, part, ""); ok {
			harness.Destroy(p, true)
		}
	}
}

// Every teardown here is deferred, so the audit marker drains the queue before
// the items are counted. Without it this would count the surface a tick before
// the networks came down and report the run as having lost nearly everything.
func phaseStressEnd() {
	mark(19, "stress ends: every part removed, so every network is torn down")
	s := harness.Surface(surfStress)
	n := fkapi.OfString(part)
	found, _ := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Name: &n})
	for _, p := range found {
		harness.Destroy(p, true)
	}
	auditMarker(2)
	out.Open("stress recovered=").I(countOn(surfStress, stressArea())).End()
}

func phaseAudit() {
	mark(20, "final audit")
	auditMarker(4)
}

// M5: the I/O arrows must not leak.
//
// Every visible interface the compiler places carries exactly one rendering
// object, and the guest stores no id for it -- it relies on the engine
// destroying a rendering object whose TARGET ENTITY is destroyed. This suite is
// where that claim is worth testing: it has taken ~100 networks down, deleted a
// surface with balancers on it, deleted the HIDDEN surface under every network
// at once, cloned interfaces and killed some from outside. If any of those paths
// left a rendering object behind, the count is higher than the number of
// interfaces standing.
//
// Counting entities and counting rendering objects is an independent
// observation, not a second implementation of anything the guest does.
func renderCheck(tick uint64) {
	mod := "better-belt-balancer"
	objs, _ := fkapi.Rendering.GetAllObjects(&mod)
	var ifaces int64
	for _, s := range harness.SurfacesByIndex() {
		if n, err := s.Name(); err == nil && n == hidden {
			continue
		}
		ifaces += int64(len(harness.EntitiesIn(s, harness.Box(-2048, -2048, 2048, 2048),
			"bbb-linked-belt")))
	}
	out.Open("t=").U(tick).S(" renders=").I(int64(len(objs))).
		S(" visible_interfaces=").I(ifaces).End()
}

func report(tick uint64) {
	order := make([]string, 0, len(rigs))
	for i := range rigs {
		order = append(order, rigs[i].name)
	}
	harness.SortStrings(order)
	s := harness.Surface(surfA)
	for _, name := range order {
		r := rigOf(name)
		n := r.outs
		if n == 0 {
			n = 1
		}
		out.Open("t=").U(tick).S(" rig=").S(name).S(" out=[")
		for i := 0; i < n; i++ {
			if i > 0 {
				out.S(" ")
			}
			if i < len(r.out) && r.out[i].has {
				out.I(harness.ChestCount(s, "steel-chest", r.out[i].xy.X, r.out[i].xy.Y))
			} else {
				out.I(-1)
			}
		}
		out.S("]").End()
	}
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

// portPlaced is the Lua's `storage.port`, as a flag rather than as an entity.
//
// The roboport is re-found on its tile every tick it is topped up, which is the
// estate's own no-entity-references habit: a handle kept across a tick would
// have to be Retained (harness.Profiler.Retain is where that is written up), and
// the tile is a fact that goes on being true for nothing.
var portPlaced bool

var schedule = []harness.Step{
	{Tick: 30, Do: phaseBlueprintCapture},
	{Tick: 60, Do: phaseGhostRevive},
	{Tick: 90, Do: phaseInstantPaste},
	{Tick: 92, Do: phasePasteFlushed},
	{Tick: 120, Do: phaseBotsStart},
	{Tick: 150, Do: phaseCloneArea},
	{Tick: 180, Do: phaseCloneBrush},
	{Tick: 210, Do: phaseSilentRotate},
	{Tick: 225, Do: phaseRevalidate},
	{Tick: 240, Do: phaseFastReplace},
	{Tick: 270, Do: phaseSilentDestroy},
	{Tick: 300, Do: phaseSilentNotice},
	{Tick: 330, Do: phaseSilentRecover},
	{Tick: 360, Do: phasePartDied},
	{Tick: 390, Do: phaseBeltDied},
	{Tick: 420, Do: phaseDeleteSurface},
	{Tick: 450, Do: phaseDeleteHidden},
	{Tick: 480, Do: phaseBotsCheck},
	{Tick: 540, Do: func() { report(540) }},
	{Tick: 546, Do: phaseStressStart},
	{Tick: 1200, Do: phaseStressEnd},
	{Tick: 1260, Do: phaseAudit},
	{Tick: 1500, Do: func() { report(1500); renderCheck(1500) }},
}

//go:wasmexport fk_on_init
func onInit() {
	seed = 20260801
	rows := len(rigOrder)*pitch + 8
	a := makeSurface(surfA, rows)
	makeSurface(surfB, 48)
	doomed := makeSurface(surfDoomed, 16)
	stress := makeSurface(surfStress, 60)

	harness.CreateForce(otherForce)

	rigs = make([]rig, 0, len(rigOrder)+1)
	for i := range rigOrder {
		cfg := &rigOrder[i]
		base := i * pitch
		switch {
		case cfg.name == "ctrl":
			source(a, -6, base, "", 0, false)
			for x := -4; x <= 4; x++ {
				put(a, belt, x, base, &dirE, "")
			}
			rigs = append(rigs, rig{name: "ctrl", base: base, outs: 1,
				out: []outSlot{{xy: sink(a, sinkX, base, ""), has: true}}})
		case cfg.rows > 0:
			r := buildRig(a, base, cfg.rows, cfg.ins, cfg.outs, "player", 0, false)
			r.name = cfg.name
			rigs = append(rigs, r)
		default:
			rigs = append(rigs, rig{name: cfg.name, base: base, outs: cfg.outs,
				out: make([]outSlot, cfg.outs)})
		}
	}

	// The second force, directly below forceA's parts so the two touch. They
	// must not merge, and each must get its own network out of its own belts.
	fb := rigbase("forceA") + 2
	for i := 0; i <= 1; i++ {
		put(a, part, 0, fb+i, nil, otherForce)
		put(a, part, 1, fb+i, nil, otherForce)
	}
	rb := rig{name: "forceB", base: fb, outs: 2, out: make([]outSlot, 2)}
	for i := 0; i <= 1; i++ {
		source(a, -6, fb+i, otherForce, 0, false)
		for x := -4; x <= -1; x++ {
			put(a, belt, x, fb+i, &dirE, otherForce)
		}
		for x := 2; x <= 4; x++ {
			put(a, belt, x, fb+i, &dirE, otherForce)
		}
		rb.out[i] = outSlot{xy: sink(a, sinkX, fb+i, otherForce), has: true}
	}
	rigs = append(rigs, rb)

	// A rig on the surface that gets deleted.
	buildRig(doomed, 0, 2, 2, 2, "player", 0, false)

	// The stress rigs: finite sources, so the items can be counted.
	for _, row := range stressRows {
		for i := 0; i <= 1; i++ {
			put(stress, part, 0, row+i, nil, "")
			put(stress, part, 1, row+i, nil, "")
		}
		for i := 0; i <= 1; i++ {
			source(stress, -6, row+i, "player", stressStock, true)
			for x := -4; x <= -1; x++ {
				put(stress, belt, x, row+i, &dirE, "")
			}
			for x := 2; x <= 4; x++ {
				put(stress, belt, x, row+i, &dirE, "")
			}
			sink(stress, sinkX, row+i, "")
		}
	}

	// `--create` never reaches a tick, so the deferred flush the build events
	// armed would otherwise run on the first tick of the BENCHMARK. The audit
	// marker drains it here, which is also what keeps `--create` doing the
	// compiling exactly as it did before the guest learned to batch.
	auditMarker(3)
	players, _ := fkapi.Game.Players()
	out.Open("init complete: ").I(int64(len(rigOrder))).S(" rigs, players=").
		I(int64(len(players))).End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	t := fkapi.ReadOnTick(ptr).Tick
	harness.Run(schedule, t)

	// The roboport has no electric network out here, so it is topped up directly
	// for as long as the bots have work to do.
	if portPlaced && t <= 480 {
		s := harness.Surface(surfA)
		if p, ok := harness.FindAt(s, 12, rigbase("bots")+1, "roboport", ""); ok {
			e := fkapi.LuaEntity{Object: p}
			if size, err := e.ElectricBufferSize(); err == nil && size != nil {
				e.SetEnergy(*size)
			}
		}
	}
	if t > 546 && t <= 1146 && t%6 == 0 {
		stressStep()
	}
}

func main() {}
