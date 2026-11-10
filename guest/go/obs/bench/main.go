// bbb-bench-setup: builds N belt-balancer test rigs into a fresh save, meters
// what they deliver, and profiles the one recompile a megabase cell exists for.
//
// It is the `bench/` harness's setup mod and it is NOT a suite's observer.
// Everything else under guest/go/obs reports into a `test/assert-*.py`; this one
// reports into `bench/run.sh`, which turns the numbers into a row of
// bench/baselines/results.tsv. Read bench/README.md for the method -- in-session
// controls, interleaving, and why absolute milliseconds from two sessions are
// not comparable.
//
// Everything happens in `fk_on_init`, which Factorio runs while `--create`
// builds the map, so the resulting save already contains the whole rig array and
// can be fed straight to `--benchmark`.
//
// Rig layout (one row per input/output belt, K rows, x grows east):
//
//	col 0        source infinity-chest (spawns `item`; empty in the idle scenarios)
//	col 1..2     source loader, type "output"  (chest -> belt)
//	col 3..5     input transport belts, facing east
//	col 6..6+K-1 K balancer-part columns  (plain belts in the control scenarios)
//	             the part prototype is cfg.partName -- belt-balancer-2 and -3
//	             both call theirs "balancer-part", ours is "bbb-balancer-part"
//	col 6+K..+2  output transport belts, facing east
//	col +3..+4   sink loader, type "input"  (belt -> chest)
//	col +5       sink steel-chest, drained by the meter
//
// The K x K part block is one balancer per rig: the K belts on the west face are
// its inputs, the K on the east face its outputs.
//
// ---------------------------------------------------------------------------
// THREE THINGS THIS COSTS THAT THE LUA IT REPLACES DID NOT
// ---------------------------------------------------------------------------
//
// This mod is present in EVERY cell of every matrix, including the no-mod
// controls, so anything it costs is a constant that cancels out of every delta
// the harness publishes -- which is the argument bench/README.md has always made
// about the meter. What moves is the ABSOLUTE milliseconds, and all three of
// these are measured in agents/estate-port.md's phase-7 record rather than
// asserted to be small.
//
//  1. THERE IS NO on_nth_tick BINDING AND THERE CANNOT BE ONE.
//     `LuaBootstrap::on_nth_tick` takes a Lua FUNCTION, which a guest has no way
//     to hand over, and FkLua's documented answer -- a self-re-arming
//     `fk.Defer()` -- has exactly the same cost, because a one-shot timer that
//     must reach tick 600 re-arms 600 times. So the meter is a per-tick dispatch
//     that returns immediately on 599 ticks out of 600 where the Lua's was an
//     engine-side modulo. `tick()` below is that dispatch and it is deliberately
//     the cheapest thing in this file.
//
//  2. A HOST CALL IS ~12.6 us AND A LUA CALL INTO C++ IS NOT.
//     The meter reads and clears every sink chest, so its cost is host calls per
//     chest per sample: two on a chest that has something in it and ONE on a
//     chest that does not, which is why the idle scenarios pay almost nothing.
//     `LuaEntity.GetItemCount` and `LuaEntity.ClearItemsInside` are used rather
//     than `get_inventory` and then the two calls on the inventory, because that
//     is three calls where these are two.
//
//  3. THE SINK HANDLES ARE RETAINED, WHICH THE REST OF THE ESTATE DOES NOT DO.
//     Every other observer holds TILES and re-finds an entity on the tile it was
//     built on -- the shipped guest's own no-entity-references rule, kept as a
//     habit. Here that would be a THIRD host call per chest per sample on the
//     harness's dominant cost, so the sinks are `Object.Retain()`ed instead.
//     FkLua's persistent handle space is `storage.fk_handles` and Factorio
//     serializes the reference, adopted on load under the same-build gate the
//     guest heap uses -- and `--create` and `--benchmark` are the same build by
//     construction. See FkLua's agents/abi.md, "What a retained handle means
//     after a load".
//
// ---------------------------------------------------------------------------
// AND ONE THING IT DOES NOT DO AT ALL
// ---------------------------------------------------------------------------
//
// The Lua set `game.map_settings.pollution.enabled = false` and the same for
// enemy evolution and expansion. `LuaGameScript::map_settings` is a read-only
// attribute returning a concept BY VALUE, so the generated binding hands over a
// copy and there is nothing to write back; in Lua the returned table was a live
// proxy. There is no other route -- `--map-settings` takes a COMPLETE settings
// tree and refuses a partial one, measured.
//
// It is belt-and-braces over a world that already has nothing to pollute or
// evolve: the bench surface is created peaceful with no entity autoplace, every
// enemy on every surface is destroyed at init, and no rig contains anything that
// produces pollution. The phase-7 comparability gate is the evidence rather than
// this paragraph.
package main

import (
	"strconv"

	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

// out has NO TAG, which is the one place this file differs from every other
// observer's line builder and is not an oversight.
//
// `bench/run.sh` greps `BENCH-SETUP`, `BENCH-METER`, `BENCH-SHAPE`,
// `BENCH-MEGA` and `BENCH-MEGA-SHAPE` at the head of a line, exactly as the Lua
// wrote them. A `[BBB-BENCH] ` prefix would be a change to the harness's own
// parsing surface for no gain.
var out = harness.Line{}

const surfaceName = "bbb-bench"

// PAD is the gap between rigs, in tiles.
const PAD = 3

// megaMinSplitters is the floor the mega scenario exists to clear. A save that
// does not carry a megabase's worth of hidden network is not a megabase
// measurement, and a silently-shrunk population would look like a performance
// win.
const megaMinSplitters = 3000

// ---------------------------------------------------------------------------
// configuration
// ---------------------------------------------------------------------------

// cfg is what `config.lua` was. See protos' own header for why it is startup
// settings now and how bench/run.sh fills them.
type config struct {
	scenario string
	n        int
	k        int
	tier     string
	item     string
	partName string
	meter    int
	hitch    bool
}

var cfg config

// readConfig pulls the eight keys out of `settings.startup`.
//
// The RAW handle plus one index read per key, which is the shipped guest's own
// idiom for a LuaCustomTable (guest/go/sedge.go's settingMultiEdge): the
// materialising attribute would build a Go slice of every startup setting every
// mod in the game defines to answer eight questions. Each value arrives as a
// one-key `{value = ...}` map, which is what the engine's own mod-settings.dat
// holds and what `settings.startup[name].value` unwraps.
//
// A MISSING KEY IS FATAL rather than defaulted. The defaults live in the
// settings stage, so a key that is not there means the settings stage did not
// run or the name drifted between the two wasm modules of this one mod -- which
// is precisely what `protos` exists to prevent and precisely the failure a
// silent default would hide.
func readConfig() {
	raw, err := fkapi.Settings.StartupRaw()
	if err != nil {
		harness.Fatal("reading settings.startup", "no table")
		return
	}
	t := fkapi.LuaCustomTable{Object: raw}
	cfg = config{
		scenario: settingStr(t, protos.BenchScenario),
		n:        int(settingNum(t, protos.BenchN)),
		k:        int(settingNum(t, protos.BenchK)),
		tier:     settingStr(t, protos.BenchTier),
		item:     settingStr(t, protos.BenchItem),
		partName: settingStr(t, protos.BenchPartName),
		meter:    int(settingNum(t, protos.BenchMeter)),
		hitch:    settingBool(t, protos.BenchHitch),
	}
}

// settingValue is the `.value` of one startup setting, or an absent value.
func settingValue(t fkapi.LuaCustomTable, name string) (fkapi.Value, bool) {
	v, err := t.Get(fkapi.OfString(name))
	if err != nil || v.Tag != fkapi.TagMap {
		harness.Fatal("no startup setting", name)
		return fkapi.Value{}, false
	}
	for i := range v.Map {
		if v.Map[i].Key.Tag == fkapi.TagString && v.Map[i].Key.Str == "value" {
			return v.Map[i].Val, true
		}
	}
	harness.Fatal("startup setting carries no value", name)
	return fkapi.Value{}, false
}

func settingStr(t fkapi.LuaCustomTable, name string) string {
	v, ok := settingValue(t, name)
	if !ok || v.Tag != fkapi.TagString {
		return ""
	}
	return v.Str
}

func settingNum(t fkapi.LuaCustomTable, name string) float64 {
	v, ok := settingValue(t, name)
	if !ok || v.Tag != fkapi.TagNumber {
		return 0
	}
	return v.Number
}

func settingBool(t fkapi.LuaCustomTable, name string) bool {
	v, ok := settingValue(t, name)
	if !ok || v.Tag != fkapi.TagBool {
		return false
	}
	return v.Bool
}

// ---------------------------------------------------------------------------
// tiers and scenarios
// ---------------------------------------------------------------------------

type tier struct{ belt, loader string }

// tiers is the Lua's TIERS table. A slice and a linear scan over three entries,
// which is the estate's rule for anything whose order could otherwise come from
// a hash.
var tiers = []struct {
	name string
	t    tier
}{
	{"normal", tier{"transport-belt", "loader"}},
	{"fast", tier{"fast-transport-belt", "fast-loader"}},
	{"express", tier{"express-transport-belt", "express-loader"}},
}

func tierOf(name string) (tier, bool) {
	for _, e := range tiers {
		if e.name == name {
			return e.t, true
		}
	}
	return tier{}, false
}

// scenario is what a rig CONTAINS: two independent switches -- whether the
// balancer parts exist, and whether items flow -- plus which builder runs.
type scenario struct{ parts, items, mega bool }

var scenarios = []struct {
	name string
	s    scenario
}{
	{"saturated", scenario{parts: true, items: true}},
	{"idle", scenario{parts: true, items: false}},
	{"control", scenario{parts: false, items: true}},
	{"control-idle", scenario{parts: false, items: false}},
	{"mega", scenario{parts: true, items: true, mega: true}},
	{"mega-idle", scenario{parts: true, items: false, mega: true}},
	{"mega-control", scenario{parts: false, items: true, mega: true}},
	{"mega-control-idle", scenario{parts: false, items: false, mega: true}},
}

func scenarioOf(name string) (scenario, bool) {
	for _, e := range scenarios {
		if e.name == name {
			return e.s, true
		}
	}
	return scenario{}, false
}

var (
	sc   scenario
	tr   tier
	east uint32
)

// init runs from `_initialize`, which Factorio calls on EVERY LOAD -- once when
// `--create` builds the save and again when `--benchmark` opens it.
//
// THAT IS WHERE A SUBSCRIPTION HAS TO BE MADE and it is not a style choice: an
// event registration does not survive a save, so a `fk.subscribe` made from
// `fk_on_init` would be live during `--create`, which never reaches a tick, and
// gone during `--benchmark`, which is the only phase that has any. The first cut
// of this file subscribed from `fk_on_init` and the meter never fired once.
//
// The compass is read here for the same reason it is in every other observer:
// re-resolving one host call per load is cheaper to reason about than trusting a
// cached define across a save.
func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	east = fkapi.DefinesDirectionEast()
}

// ---------------------------------------------------------------------------
// state that crosses the save
// ---------------------------------------------------------------------------

// rig is one balancer and its sinks, in the shape the meter reads.
//
// `sinks` are RETAINED handles -- see this file's header for why this one
// observer holds objects where the other thirteen hold tiles.
type rig struct {
	class  string
	sinks  []fkapi.Object
	totals []int64
	// deliver says whether the worst-balance search may look at this rig. The
	// over-limit cluster is REFUSED, so it has no network and delivers nothing;
	// and in the CONTROL scenarios there is no balancer at all -- a 3->5's rows
	// 3 and 4 have no source and cannot be fed by a straight belt -- so only the
	// square shapes are asked to be even there.
	deliver bool
}

var (
	rigs []rig
	// totals is the uniform scenarios' per-output-INDEX accumulator, summed
	// across every rig. The mega scenarios have no single such vector and use
	// each rig's own `totals` instead.
	totals []int64
	isMega bool
	// hitchTargets are the rigs the hitch schedule removes and restores one
	// input belt of: the tile immediately west of each one's top-left part.
	// Block 0's first 4x4 and its 8x8, then the three giants that build.
	hitchTargets []hitchTarget
	hitchArmed   bool
)

type hitchTarget struct {
	name string
	tile harness.XY
}

// ---------------------------------------------------------------------------
// surface
// ---------------------------------------------------------------------------

// makeSurface is the Lua's make_surface: a blank, deterministic plain, bounded,
// generated, swept, de-decorated and paved.
//
// `harness.Flat` is the shared form of it and its header records the two things
// every one of these settings blocks deviated on -- `cliff_settings` and
// `water`, neither of which was doing anything. `freeze_daytime` is set here
// rather than in Flat because this is the only caller that asked for it.
func makeSurface(width, height int) fkapi.LuaSurface {
	cx := ceilDiv(width+64, 64) + 1
	cy := ceilDiv(height+64, 64) + 1
	radius := cx
	if cy > radius {
		radius = cy
	}
	s := harness.Flat{
		Name:      surfaceName,
		MapWidth:  uint32(width + 64),
		MapHeight: uint32(height + 64),
		ChunkCenter: fkapi.MapPosition{
			X: float64(width) / 2, Y: float64(height) / 2,
		},
		ChunkRadius: uint32(radius),
		X0:          -32, Y0: -32, X1: width + 32, Y1: height + 32,
		Tile: "grass-1",
	}.Make()
	if err := s.SetFreezeDaytime(true); err != nil {
		harness.Fatal("freeze_daytime on "+surfaceName, "refused")
	}
	return s
}

// ceilDiv is the Lua's `math.ceil(a / b)` for non-negative integers. The two
// call sites above are `math.ceil((w + 64) / 32 / 2)`, which is one division by
// 64 rather than two.
func ceilDiv(a, b int) int {
	if a <= 0 {
		return 0
	}
	return (a + b - 1) / b
}

// killEnemies destroys every enemy on every surface. Nothing here should ever be
// attacked; the three `game.map_settings` writes that went with it in the Lua
// are unreachable from a guest and this file's header says what that costs.
func killEnemies() {
	for _, s := range harness.SurfacesByIndex() {
		for _, e := range harness.EntitiesOfForce(s, "enemy") {
			harness.Destroy(e, false)
		}
	}
}

// ---------------------------------------------------------------------------
// rig construction
// ---------------------------------------------------------------------------

func rigWidth(k int) int  { return k + 12 }
func rigHeight(k int) int { return k }

// source builds one row's infinity chest and its loader, and returns nothing:
// the chest is configured and forgotten, because only the SINKS are read again.
func source(s fkapi.LuaSurface, ox, ty int) {
	src := harness.Place(s, harness.Piece{
		Name: "infinity-chest", X: ox, Y: ty, Raise: true,
	})
	if sc.items {
		harness.InfinityFilter(src, cfg.item, "", 200)
	}
	harness.RemoveUnfiltered(src, true)
	loader(s, ox+2, ty, "output")
}

// loader places one 1x2 loader.
//
// A RAW POSITION AND NOT A TILE, and that is the geometry rather than a style:
// a loader is two tiles long, so facing east its centre x sits ON the boundary
// between the two columns it occupies while its y is an ordinary tile centre.
// The Lua wrote `{ x = ox + 2, y = ty + 0.5 }` for the same reason.
func loader(s fkapi.LuaSurface, x, ty int, kind string) {
	pos := fkapi.MapPosition{X: float64(x), Y: float64(ty) + 0.5}
	harness.Place(s, harness.Piece{
		Name: tr.loader, X: x, Y: ty, Pos: &pos,
		Dir: &east, Type: kind, Raise: true,
	})
}

func belt(s fkapi.LuaSurface, x, y int) {
	harness.Place(s, harness.Piece{Name: tr.belt, X: x, Y: y, Dir: &east, Raise: true})
}

// sink builds one row's three output belts, its loader and its chest, and
// returns the chest RETAINED.
func sink(s fkapi.LuaSurface, outX0, ty int) fkapi.Object {
	for tx := outX0; tx <= outX0+2; tx++ {
		belt(s, tx, ty)
	}
	loader(s, outX0+4, ty, "input")
	chest := harness.Place(s, harness.Piece{
		Name: "steel-chest", X: outX0 + 5, Y: ty, Raise: true,
	})
	return chest.Retain()
}

// buildRig is the uniform scenarios' rig: K rows, one balancer.
func buildRig(s fkapi.LuaSurface, ox, oy int) []fkapi.Object {
	k := cfg.k
	partX0 := ox + 6
	outX0 := partX0 + k
	sinks := make([]fkapi.Object, 0, k)

	for row := 0; row < k; row++ {
		ty := oy + row
		source(s, ox, ty)

		// Belt run. With no parts the belt is continuous across the columns the
		// parts would have occupied, so items still reach the sink.
		for tx := ox + 3; tx < outX0; tx++ {
			isPartColumn := tx >= partX0 && tx < outX0
			if !sc.parts || !isPartColumn {
				belt(s, tx, ty)
			}
		}
		sinks = append(sinks, sink(s, outX0, ty))
	}

	// Parts last: belt-balancer-2 links a part to its belts at build time, and
	// linking from either side works, but building parts into finished belt rows
	// exercises the path a player actually takes.
	if sc.parts {
		for px := 0; px < k; px++ {
			for py := 0; py < k; py++ {
				harness.Place(s, harness.Piece{
					Name: cfg.partName, X: partX0 + px, Y: oy + py, Raise: true,
				})
			}
		}
	}
	return sinks
}

// ---------------------------------------------------------------------------
// MEGA: a heterogeneous population at megabase scale
// ---------------------------------------------------------------------------
//
// Everything above builds `n` copies of one shape. The mega scenarios build a
// MIX, because a megabase is a mix: square balancers, non-power-of-two ones
// whose spare ports loop back, dead-ended ones whose spare outputs back up, and
// -- for the first time in a real game rather than in `plan`'s simulator -- the
// three sizes past P=8, up to the P=64 that IS `plan.MaxPorts`.
//
// A rig here is `n` inputs on the west face and `m` outputs on the east face of
// a part block `cols` wide and `max(n, m)` tall. Rows past `n` have no source
// and rows past `m` no sink, which is the whole of what makes a 3->5 a 3->5.
// Otherwise the column layout is the uniform rig's, tile for tile.
//
// `-n` is the number of BLOCKS. Each block is the ten small rigs below; the four
// giants are built once regardless, because one 64x64 is a measurement and forty
// of them is a different benchmark.

type shape struct {
	name       string
	n, m, cols int
	// defer_ on the 64x64 is what lets the create time its very first compile
	// against an otherwise-identical audit; see megaInit.
	defer_ bool
	// overLimit is the 65-input cluster: P would be 128, the guest must refuse
	// it BEFORE tearing anything down, and it must deliver nothing.
	overLimit bool
}

var megaBlock = []shape{
	{name: "2->2", n: 2, m: 2, cols: 2},
	{name: "2->2", n: 2, m: 2, cols: 2},
	{name: "2->2", n: 2, m: 2, cols: 2},
	{name: "3->3", n: 3, m: 3, cols: 3},
	{name: "3->3", n: 3, m: 3, cols: 3},
	{name: "4x4", n: 4, m: 4, cols: 4},
	{name: "4x4", n: 4, m: 4, cols: 4},
	{name: "3->5", n: 3, m: 5, cols: 4}, // loopback: spare outputs feed spare inputs
	{name: "5->3", n: 5, m: 3, cols: 4}, // dead end: spare outputs back up
	{name: "8x8", n: 8, m: 8, cols: 4},
}

var megaGiants = []shape{
	{name: "16x16", n: 16, m: 16, cols: 4},
	{name: "32x32", n: 32, m: 32, cols: 4},
	{name: "64x64", n: 64, m: 64, cols: 4, defer_: true},
	{name: "65->1", n: 65, m: 1, cols: 2, overLimit: true},
}

func shapeHeight(s shape) int {
	if s.n > s.m {
		return s.n
	}
	return s.m
}
func shapeWidth(s shape) int { return s.cols + 12 }

// buildMegaRig returns the rig's sinks and the x of its first part column.
func buildMegaRig(s fkapi.LuaSurface, ox, oy int, sh shape) ([]fkapi.Object, int) {
	h := shapeHeight(sh)
	partX0 := ox + 6
	outX0 := partX0 + sh.cols
	sinks := make([]fkapi.Object, 0, sh.m)

	for row := 0; row < h; row++ {
		ty := oy + row

		if row < sh.n {
			source(s, ox, ty)
			for tx := ox + 3; tx < partX0; tx++ {
				belt(s, tx, ty)
			}
		}

		// The part columns carry belts in the control scenarios, so a row that
		// has both a source and a sink still delivers. A row with a source and
		// no sink backs up -- which is what the dead-ended shapes do in the
		// balancer arm too, so the two arms agree about that row rather than
		// differing.
		if !sc.parts {
			for tx := partX0; tx < outX0; tx++ {
				belt(s, tx, ty)
			}
		}

		if row < sh.m {
			sinks = append(sinks, sink(s, outX0, ty))
		}
	}

	if sc.parts {
		for px := 0; px < sh.cols; px++ {
			for py := 0; py < h; py++ {
				harness.Place(s, harness.Piece{
					Name: cfg.partName, X: partX0 + px, Y: oy + py, Raise: true,
				})
			}
		}
	}
	return sinks, partX0
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

// checkConfig is the Lua's three asserts. A guest cannot call `error()`, so an
// unknown tier or scenario is `[BBB-OBS] error:` and bench/run.sh fails the cell
// on it -- which is the same severity by a different route.
func checkConfig() bool {
	var ok bool
	if tr, ok = tierOf(cfg.tier); !ok {
		harness.Fatal("bbb-bench: unknown tier", cfg.tier)
		return false
	}
	if sc, ok = scenarioOf(cfg.scenario); !ok {
		harness.Fatal("bbb-bench: unknown scenario", cfg.scenario)
		return false
	}
	// A scenario that places parts against a mod that does not define that
	// prototype fails here rather than at create_entity, 16 x n times over.
	if sc.parts && !harness.EntityProtoExists(cfg.partName) {
		harness.Fatal("bbb-bench: no such entity prototype",
			cfg.partName+" (is the balancer mod enabled?)")
		return false
	}
	return true
}

func benchInit() {
	w, h := rigWidth(cfg.k), rigHeight(cfg.k)
	pitchX, pitchY := w+PAD, h+PAD
	cols := isqrtCeil(cfg.n)
	rows := ceilDiv(cfg.n, cols)

	// With --hitch a uniform cell also carries the DRAG rig below the grid.
	sw, sh := cols*pitchX, rows*pitchY
	if cfg.hitch && sc.parts {
		if w := dragCols + 22 + PAD; w > sw {
			sw = w
		}
		sh += dragRows + 2 + PAD
	}
	s := makeSurface(sw, sh)
	killEnemies()

	totals = make([]int64, cfg.k)
	rigs = make([]rig, 0, cfg.n)
	for i := 0; i < cfg.n; i++ {
		ox := (i % cols) * pitchX
		oy := (i / cols) * pitchY
		rigs = append(rigs, rig{sinks: buildRig(s, ox, oy)})
	}

	// BetterBeltBalancer batches its recompiles onto the following tick
	// (`fk.defer`), and `--create` never reaches a tick -- so without this every
	// network in the save would be compiled on the FIRST TICK OF THE BENCHMARK,
	// which is the one measurement this harness exists to keep clean.
	// `bbb-audit` is that mod's own synchronous "drain and re-classify now"
	// marker; it destroys itself. Guarded by the prototype's existence, so the
	// bb2, bb3 and no-mod cells are untouched.
	if harness.EntityProtoExists(harness.AuditMarker) {
		harness.Audit(s, -41, -41)
		// A second audit with NOTHING pending re-classifies every cluster twice
		// over (its own pass, then the flush it ends with, which is the whole of
		// what an undo or a redo runs) and compiles nothing. It is the only
		// stand-in for an undo there is: on_undo_applied cannot be raised from a
		// script (agents/verification/suites/m3.md).
		p := harness.StartProfiler()
		harness.Audit(s, -41, -41)
		p.Stop()
		p.Log("[BENCH] timing audit only, nothing pending ")
	}

	if cfg.hitch && sc.parts {
		buildDragRig(s, 0, rows*pitchY)
	}

	reportSetup(len(rigs), sw, sh)
}

// isqrtCeil is `math.ceil(math.sqrt(n))` without a square root: the smallest c
// whose square reaches n. n is a rig count, so the loop is under 32 iterations
// for anything anybody has ever run.
func isqrtCeil(n int) int {
	if n <= 1 {
		return 1
	}
	c := 1
	for c*c < n {
		c++
	}
	return c
}

func megaInit() {
	nblocks := cfg.n
	if nblocks < 1 {
		harness.Fatal("bbb-bench: mega needs at least one block", "n < 1")
		return
	}

	// Block metrics: the ten small shapes stacked vertically, PAD between them.
	blockH, blockW := 0, 0
	for _, sh := range megaBlock {
		blockH += shapeHeight(sh) + PAD
		if w := shapeWidth(sh); w > blockW {
			blockW = w
		}
	}
	pitchX := blockW + PAD
	bcols := isqrtCeil(nblocks)
	brows := ceilDiv(nblocks, bcols)

	// The giants get their own column to the right of the block grid: they are
	// 16 to 65 rows tall and would otherwise dictate every cell's height.
	giantH, giantW := 0, 0
	for _, sh := range megaGiants {
		giantH += shapeHeight(sh) + PAD
		if w := shapeWidth(sh); w > giantW {
			giantW = w
		}
	}

	width := bcols*pitchX + giantW + PAD
	height := brows * blockH
	if giantH > height {
		height = giantH
	}

	s := makeSurface(width, height)
	killEnemies()
	isMega = true

	var counts []int
	var order []string
	add := func(sh shape, ox, oy int) (int, int) {
		sinks, partX0 := buildMegaRig(s, ox, oy, sh)
		rigs = append(rigs, rig{
			class:   sh.name,
			sinks:   sinks,
			totals:  make([]int64, len(sinks)),
			deliver: !sh.overLimit && (sc.parts || sh.n == sh.m),
		})
		found := false
		for i, name := range order {
			if name == sh.name {
				counts[i]++
				found = true
				break
			}
		}
		if !found {
			order = append(order, sh.name)
			counts = append(counts, 1)
		}
		return len(rigs) - 1, partX0
	}

	var small []hitchTarget
	for b := 0; b < nblocks; b++ {
		ox := (b % bcols) * pitchX
		y := (b / bcols) * blockH
		for _, sh := range megaBlock {
			_, px := add(sh, ox, y)
			if b == 0 && (sh.name == "4x4" || sh.name == "8x8") && !hasTarget(small, sh.name) {
				small = append(small, hitchTarget{sh.name, harness.XY{X: px - 1, Y: y}})
			}
			y += shapeHeight(sh) + PAD
		}
	}
	hitchTargets = append(hitchTargets, small...)

	gx, gy := bcols*pitchX, 0
	var deferred shape
	deferredY := 0
	for _, sh := range megaGiants {
		if sh.defer_ {
			deferred, deferredY = sh, gy
		} else {
			_, px := add(sh, gx, gy)
			if !sh.overLimit {
				hitchTargets = append(hitchTargets, hitchTarget{sh.name, harness.XY{X: px - 1, Y: gy}})
			}
		}
		gy += shapeHeight(sh) + PAD
	}

	// Everything except the 64x64 is standing. Flush it, then time an audit with
	// NOTHING pending, then build the 64x64 and time the audit that compiles it.
	// The difference is the first compile of a P=64 network, measured the same
	// way M2 measures a recompile: two probes and a subtraction, never one
	// number on its own. (The audit re-classifies every cluster in the save, so
	// the control is not small and cannot be skipped.)
	const mx, my = -40, -40
	audited := harness.EntityProtoExists(harness.AuditMarker)
	if audited {
		harness.Audit(s, mx, my)
		p := harness.StartProfiler()
		harness.Audit(s, mx, my)
		p.Stop()
		p.Log("[BENCH-MEGA] timing audit only, nothing pending ")
	}

	_, gpx := add(deferred, gx, deferredY)
	if audited {
		q := harness.StartProfiler()
		harness.Audit(s, mx, my)
		q.Stop()
		q.Log("[BENCH-MEGA] timing audit + FIRST COMPILE of the 64x64 ")
	}

	hitchTargets = append(hitchTargets, hitchTarget{deferred.name, harness.XY{X: gpx - 1, Y: deferredY}})
	hitchArmed = true

	for i, name := range order {
		out.Open("BENCH-MEGA-SHAPE class=").S(name).S(" rigs=").I(int64(counts[i])).End()
	}

	// The floor. A whole-surface name query is what we want: the hidden surface
	// holds nothing else.
	nsplit, nlane := 0, 0
	hid, haveHidden := harness.SurfaceIfAny("bbb-hidden")
	if haveHidden {
		nsplit = harness.CountNamedOn(hid, "bbb-splitter", "")
		nlane = harness.CountNamedOn(hid, "bbb-lane-splitter", "")
	}
	out.Open("BENCH-MEGA blocks=").I(int64(nblocks)).
		S(" rigs=").I(int64(len(rigs))).
		S(" hidden_splitters=").I(int64(nsplit + nlane)).
		S(" (bbb-splitter=").I(int64(nsplit)).
		S(" bbb-lane-splitter=").I(int64(nlane)).S(")").End()
	if sc.parts && haveHidden && nsplit+nlane < megaMinSplitters {
		harness.Fatal("bbb-bench: mega built too few hidden splitters",
			"floor is 3000 -- raise --n")
	}

	reportSetup(len(rigs), width, height)
}

// reportSetup writes the line bench/run.sh requires of every cell. A cell whose
// create log does not carry it is failed rather than recorded.
//
// IT ECHOES ALL EIGHT CONFIGURATION KEYS AND THAT IS THE ANTI-VACUITY GUARD,
// not decoration. A settings stage defines a DEFAULT for every key, so a key
// that `bench/run.sh` misspells when it writes mod-settings.dat is not absent at
// runtime -- it reads back as the default, and the cell would build a plausible
// rig for a configuration nobody asked for. `run.sh` compares this line against
// what it asked for, field for field, which is the only place that can be seen.
func reportSetup(built, w, h int) {
	out.Open("BENCH-SETUP scenario=").S(cfg.scenario).
		S(" n=").I(int64(cfg.n)).
		S(" k=").I(int64(cfg.k)).
		S(" tier=").S(cfg.tier).
		S(" item=").S(cfg.item).
		S(" part=").S(cfg.partName).
		S(" meter=").I(int64(cfg.meter)).
		S(" hitch=").B(cfg.hitch).
		S(" rigs_built=").I(int64(built)).
		S(" surface=").I(int64(w)).S("x").I(int64(h)).End()
}

// ---------------------------------------------------------------------------
// metering: throughput + balance sanity check
// ---------------------------------------------------------------------------

// drain reads one chest and empties it, and is the whole of what the meter
// costs. ONE host call on an empty chest and two on a full one -- see this
// file's header.
func drain(o fkapi.Object) int64 {
	e := fkapi.LuaEntity{Object: o}
	n, err := e.GetItemCount(nil)
	if err != nil || n == 0 {
		return 0
	}
	if err := e.ClearItemsInside(); err != nil {
		return 0
	}
	return int64(n)
}

// meter drains every sink chest and accumulates per-output-index totals. Runs
// rarely (default every 600 ticks) and is identical across scenarios, so it
// cancels in any delta. It also keeps the sinks from backing up, which is what
// keeps the rigs saturated.
func meter(tick uint64) {
	var window int64
	for i := range rigs {
		for idx, chest := range rigs[i].sinks {
			if n := drain(chest); n > 0 {
				totals[idx] += n
				window += n
			}
		}
	}

	var cumulative int64
	for _, v := range totals {
		cumulative += v
	}
	out.Open("BENCH-METER tick=").U(tick).
		S(" window=").I(window).
		S(" cumulative=").I(cumulative).
		S(" per_output=")
	for i, v := range totals {
		if i > 0 {
			out.S(",")
		}
		out.I(v)
	}
	out.End()
}

// megaMeter is the mega population's meter. There is no single per-output-index
// vector to report, because the rigs have different output counts -- so what
// goes on the BENCH-METER line instead is the per-output vector of the
// WORST-BALANCED rig in the save, which makes run.sh's own max/min the worst
// per-rig balance anywhere. A strictly sharper gate than the uniform case's, and
// it needs no change over there. Per-class aggregates go on their own
// BENCH-SHAPE lines for the write-up.
func megaMeter(tick uint64) {
	var window, grand int64
	var order []string
	var classOuts [][]int64
	var classRigs []int64
	var classTotal []int64

	worstRatio := 0.0
	var worstVec []int64
	worstClass := ""

	for i := range rigs {
		r := &rigs[i]
		for idx, chest := range r.sinks {
			if n := drain(chest); n > 0 {
				r.totals[idx] += n
				window += n
			}
		}

		ci := -1
		for j, name := range order {
			if name == r.class {
				ci = j
				break
			}
		}
		if ci < 0 {
			order = append(order, r.class)
			classOuts = append(classOuts, nil)
			classRigs = append(classRigs, 0)
			classTotal = append(classTotal, 0)
			ci = len(order) - 1
		}
		classRigs[ci]++

		var mn, mx int64 = -1, 0
		for idx, v := range r.totals {
			for len(classOuts[ci]) <= idx {
				classOuts[ci] = append(classOuts[ci], 0)
			}
			classOuts[ci][idx] += v
			classTotal[ci] += v
			grand += v
			if mn < 0 || v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		if r.deliver && mn >= 0 {
			ratio := 999.0
			if mn > 0 {
				ratio = float64(mx) / float64(mn)
			}
			if ratio > worstRatio {
				worstRatio, worstVec, worstClass = ratio, r.totals, r.class
			}
		}
	}

	for i, name := range order {
		var mn, mx int64 = -1, 0
		for _, v := range classOuts[i] {
			if mn < 0 || v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		if mn < 0 {
			mn = 0
		}
		balance := 0.0
		if mn > 0 {
			balance = float64(mx) / float64(mn)
		}
		out.Open("BENCH-SHAPE tick=").U(tick).
			S(" class=").S(name).
			S(" rigs=").I(classRigs[i]).
			S(" outputs=").I(int64(len(classOuts[i]))).
			S(" total=").I(classTotal[i]).
			S(" min=").I(mn).
			S(" max=").I(mx).
			S(" balance=").F4(balance).End()
	}

	// No rig has delivered anything yet (the idle scenarios, or the first sample
	// of a saturated run): report a single zero rather than an empty field,
	// which would make run.sh's balance parser read 999.
	out.Open("BENCH-METER tick=").U(tick).
		S(" window=").I(window).
		S(" cumulative=").I(grand).
		S(" per_output=")
	if worstVec != nil && worstRatio > 0 {
		for i, v := range worstVec {
			if i > 0 {
				out.S(",")
			}
			out.I(v)
		}
	} else {
		out.S("0")
	}
	out.S(" worst_class=")
	if worstClass == "" {
		out.S("none")
	} else {
		out.S(worstClass)
	}
	out.End()
}

// ---------------------------------------------------------------------------
// the recompile hitch (--hitch only)
// ---------------------------------------------------------------------------
//
// M2's tick-pair pattern, verbatim in shape: the guest defers its recompile to
// the following tick, so the profiler is opened in the tick that MUTATES and
// closed in the tick that FLUSHES. Each window therefore contains one whole
// engine tick as well as the recompile, and the `idle tick pair` probe measures
// exactly that and nothing else. SUBTRACT IT.
//
// Five targets (4x4, 8x8, 16x16, 32x32, 64x64), three reps each, ROUND-ROBIN so
// a saturated network that is still filling moves every target's reps alike
// rather than the last target's alone. The windows start at 1210 and are 40
// ticks apart, ending at 1774: none of them reaches a multiple of the 600-tick
// meter interval, and a meter sample inside a window would be measured as
// recompile. The 64x64 is the same 1,152 entities one deep and two deep, which
// makes it the control when two builds are compared.

const (
	hitchFirst = 1210
	hitchStep  = 40
	hitchReps  = 3
)

func hasTarget(ts []hitchTarget, name string) bool {
	for i := range ts {
		if ts[i].name == name {
			return true
		}
	}
	return false
}

// hitchOpen is RETAINED for the reason harness.Profiler.Retain exists: every one
// of these windows spans a tick boundary by construction, and a handle that is
// not retained is valid only inside the dispatch that made it. The Lua kept its
// profiler in a module-level upvalue and never had to think about it.
var (
	hitchOpen  harness.Profiler
	hitchLive  bool
	hitchLabel string
)

func hitchStart(label string) {
	hitchOpen = harness.StartProfiler().Retain()
	hitchLabel = label
	hitchLive = true
}

func hitchStop() {
	if !hitchLive {
		return
	}
	hitchOpen.Stop()
	hitchOpen.Log("[BENCH-MEGA] hitch " + hitchLabel + " ")
	hitchOpen.Release()
	hitchLive = false
}

func hitchTick(t uint64) {
	if !hitchArmed || len(hitchTargets) == 0 || t < hitchFirst-2 {
		return
	}
	slots := uint64(len(hitchTargets) * hitchReps)
	i := (t - (hitchFirst - 2)) / hitchStep
	if i >= slots {
		return
	}
	base := hitchFirst + i*hitchStep
	tg := hitchTargets[i%uint64(len(hitchTargets))]
	s := harness.Surface(surfaceName)
	switch t {
	case base - 2:
		hitchStart("idle tick pair, nothing pending")
	case base, base + 2, base + 4:
		hitchStop()
		if t == base {
			if !harness.KillAt(s, tg.tile.X, tg.tile.Y, "", "transport-belt") {
				out.Open("[BENCH-MEGA] hitch: no input belt at the ").S(tg.name).End()
				return
			}
			hitchStart(tg.name + " teardown+rebuild(-1 input)")
		} else if t == base+2 {
			hitchStart(tg.name + " teardown+rebuild(full)")
			belt(s, tg.tile.X, tg.tile.Y)
		}
	}
}

// ---------------------------------------------------------------------------
// the drag (--hitch on a uniform cell)
// ---------------------------------------------------------------------------
//
// A player dragging balancer parts into place beside belts that are already
// running: dragRows input belts end at a part column, dragRows output belts
// start one column further on, and nothing stands in either column at create.
// From dragStart one part is placed every dragGap ticks, the input column top
// to bottom and then the output column, which is a 10->10 balancer of twenty
// parts built the way the mod-portal report describes. Every tick from just
// before the first part to well after the last is profiled on its own, so the
// log carries the drag's whole cost tick by tick, and an idle stretch earlier
// in the run gives the same windows with nothing happening, to subtract.
//
// The sinks are not metered: a steel chest takes more than one belt delivers
// in the length of a run, and the uniform meter's per-index vector is the
// grid's.
//
// THEN THE SAME BALANCER BY ROBOTS, in a run long enough to reach it
// (--ticks 2700). A powered roboport beside the rig holds thirty construction
// robots and a storage chest twenty parts. At botMineAt every part is marked
// for deconstruction and the robots take the balancer apart; at botBuildAt a
// ghost goes on every tile and they build it again. Each phase is profiled tick
// by tick for botSpan ticks, as `botmine` and `botbuild`, and logged as a
// phase marker so the compiles inside it can be counted. Robot arrivals are
// whatever the engine makes of thirty robots and twenty jobs.

const (
	dragRows   = 10
	dragCols   = 2
	dragGap    = 6
	dragStart  = 1210
	dragTail   = 120
	idleFrom   = 1050
	idleTo     = 1150
	botMineAt  = 1460
	botBuildAt = 2060
	botSpan    = 560
)

var dragParts []harness.XY

func buildDragRig(s fkapi.LuaSurface, ox, oy int) {
	partX0 := ox + 6
	outX0 := partX0 + dragCols
	for row := 0; row < dragRows; row++ {
		ty := oy + row
		source(s, ox, ty)
		for tx := ox + 3; tx < partX0; tx++ {
			belt(s, tx, ty)
		}
		sink(s, outX0, ty)
	}
	for c := 0; c < dragCols; c++ {
		for row := 0; row < dragRows; row++ {
			dragParts = append(dragParts, harness.XY{X: partX0 + c, Y: oy + row})
		}
	}
	// The robots' corner, east of the sinks. A roboport is 4x4 and a
	// substation 2x2, so both are placed by their centre on a tile corner.
	bx := float64(ox + 18)
	by := float64(oy + dragRows/2)
	port := harness.Place(s, harness.Piece{Name: "roboport", X: ox + 18, Y: oy,
		Pos: &fkapi.MapPosition{X: bx, Y: by}})
	harness.Place(s, harness.Piece{Name: "substation", X: ox + 21, Y: oy,
		Pos: &fkapi.MapPosition{X: bx + 3, Y: by}})
	harness.Place(s, harness.Piece{Name: "electric-energy-interface", X: ox + 21, Y: oy,
		Pos: &fkapi.MapPosition{X: bx + 3, Y: by + 2}})
	if inv, err := (fkapi.LuaControl{Object: port}).GetInventory(
		fkapi.DefinesInventoryRoboportRobot()); err == nil && inv != nil {
		harness.InsertInto(*inv, "construction-robot", 30)
	}
	chest := harness.Place(s, harness.Piece{Name: "storage-chest", X: ox + 16, Y: oy + dragRows + 1})
	harness.InsertInto(chest, cfg.partName, uint32(len(dragParts)))
}

// botPhase starts the robot phase that begins at t, if any.
func botPhase(s fkapi.LuaSurface, t uint64) {
	switch t {
	case botMineAt:
		out.Open("[BENCH] phase=botmine").End()
		for _, xy := range dragParts {
			if e, ok := harness.FindAt(s, xy.X, xy.Y, cfg.partName, ""); ok {
				ent := fkapi.LuaEntity{Object: e}
				if f, err := ent.Force(); err == nil {
					_, _ = ent.OrderDeconstruction(f, nil, nil)
				}
			}
		}
	case botBuildAt:
		out.Open("[BENCH] botmine left=").I(int64(partsStanding(s))).End()
		out.Open("[BENCH] phase=botbuild").End()
		for _, xy := range dragParts {
			harness.PlaceSoft(s, harness.Piece{Name: "entity-ghost", InnerName: cfg.partName,
				X: xy.X, Y: xy.Y, Raise: true})
		}
	case botBuildAt + botSpan:
		out.Open("[BENCH] botbuild built=").I(int64(partsStanding(s))).End()
		out.Open("[BENCH] phase=end").End()
	}
}

func partsStanding(s fkapi.LuaSurface) int {
	n := 0
	for _, xy := range dragParts {
		if _, ok := harness.FindAt(s, xy.X, xy.Y, cfg.partName, ""); ok {
			n++
		}
	}
	return n
}

var (
	dragOpen     harness.Profiler
	dragLive     bool
	dragTickOpen uint64
)

func dragTick(t uint64) {
	if len(dragParts) == 0 {
		return
	}
	end := uint64(dragStart + dragGap*(len(dragParts)-1) + dragTail)
	inIdle := t >= idleFrom && t <= idleTo
	inDrag := t >= dragStart-1 && t <= end
	inBotMine := t >= botMineAt-1 && t < botMineAt+botSpan
	inBotBuild := t >= botBuildAt-1 && t < botBuildAt+botSpan
	if dragLive {
		dragOpen.Stop()
		label := "[BENCH] drag tick="
		switch {
		case dragTickOpen < dragStart:
			label = "[BENCH] idle tick="
		case dragTickOpen >= botBuildAt-1:
			label = "[BENCH] botbuild tick="
		case dragTickOpen >= botMineAt-1:
			label = "[BENCH] botmine tick="
		}
		dragOpen.Log(label + strconv.FormatUint(dragTickOpen+1, 10) + " ")
		dragOpen.Release()
		dragLive = false
	}
	if t == dragStart {
		out.Open("[BENCH] phase=drag").End()
	}
	if t >= dragStart && (t-dragStart)%dragGap == 0 {
		if i := int((t - dragStart) / dragGap); i < len(dragParts) {
			s := harness.Surface(surfaceName)
			harness.Place(s, harness.Piece{Name: cfg.partName, X: dragParts[i].X, Y: dragParts[i].Y, Raise: true})
		}
	}
	if t == end {
		out.Open("[BENCH] phase=end").End()
	}
	if t >= botMineAt {
		botPhase(harness.Surface(surfaceName), t)
	}
	if (inIdle && t < idleTo) || (inDrag && t < end) || inBotMine || inBotBuild {
		dragOpen = harness.StartProfiler().Retain()
		dragTickOpen = t
		dragLive = true
	}
}

// ---------------------------------------------------------------------------
// exports
// ---------------------------------------------------------------------------

//go:wasmexport fk_on_init
func onInit() {
	readConfig()
	if !checkConfig() {
		return
	}
	if sc.mega {
		megaInit()
	} else {
		benchInit()
	}
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	tick(fkapi.ReadOnTick(ptr).Tick)
}

// tick is the whole per-tick cost of this mod and is deliberately the cheapest
// thing in the file: two integer tests on the ticks that do nothing.
//
// The Lua registered `script.on_nth_tick(interval)` and, with --hitch, an
// `on_tick`. There is no on_nth_tick binding and there cannot be one; the
// header says why and what it costs.
func tick(t uint64) {
	if cfg.meter > 0 && t > 0 && t%uint64(cfg.meter) == 0 {
		if isMega {
			megaMeter(t)
		} else {
			meter(t)
		}
	}
	if cfg.hitch {
		if isMega {
			hitchTick(t)
		} else {
			dragTick(t)
		}
	}
}

func main() {}
