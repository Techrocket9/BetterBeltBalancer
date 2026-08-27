// Command bbb-flip-test drives `bbb-multi-edge-parts` -- the runtime-global
// setting that exists on Factorio 2.0 only -- through every transition it has.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-flip-test/control.lua` was, rig for rig and log line for log
// line, with its data stage compiled too (obs/flipdata). It is the LAST suite in
// the estate to be ported, and porting it is what takes the count of
// hand-written Lua files in this repository to zero. See agents/estate-port.md,
// phase 9.
//
// THIS SUITE CANNOT RUN ON FACTORIO 2.1 AND THAT IS THE POINT. The setting is a
// runtime-global bool that guest/go/data/settings.go defines on 2.0.x and never
// on 2.1.x, so on trunk's own engine there is nothing to flip: the marker
// prototype is absent, the AND short-circuits, and the whole flip half of
// guest/go/sedge.go is unreachable. `guest/go/edgemode` proves the FOLD under
// `go test`; this is the only place anything drives the WORLD it decides about.
//
// It is also why this port waited for a 2.0 binary rather than following the
// other thirteen: a phase's FIRST gate is a golden log, `test/run.sh` prints a
// SKIP for this suite on 2.1, and there was no run on this machine to take a
// golden from until one arrived.
//
// ---------------------------------------------------------------------------
// WHAT THE FLIP-OFF ACTUALLY DOES, WHICH IS NOT WHAT THE DESIGN TEXT SAID
// ---------------------------------------------------------------------------
//
// The design called a flip OFF with multi-edge balancers standing a SWEEP: tear
// them down, spill, and tell the player. It is not. Turning the setting off puts
// `settings.global` at false with those clusters still standing, and the very
// next thing `settleEdgeMode` asks is `edgemode.GrandfatherNeeded(marker=true,
// setting=Off, n>0)` -- which is TRUE, so the grandfather pass writes the
// setting straight back ON and tells each owning force why. The flip is VETOED,
// and the sweep can never stick: the two conditions are the same condition.
//
// That is the behaviour a player reported from a live 2.0.77 session and it is
// the behaviour this suite pins. See agents/single-edge.md.
//
// ---------------------------------------------------------------------------
// THE RIGS
// ---------------------------------------------------------------------------
//
//	ctrl   a bare express belt, the yardstick
//	sok    2 -> 2 over FOUR parts, a 2x2 block laid ONE BELT PER PART. It is
//	       legal under both modes and must not be touched by any flip -- the
//	       single-edge neighbour, and the control on every window
//	me1    2 -> 2 over TWO parts, the incumbent's idiom: each part carries an
//	       input on its west face and an output on its east. Built at fk_on_init,
//	       so it is REFUSED at the false default; it compiles when the setting
//	       goes on, and it drains freely, so its delivery is what says a vetoed
//	       flip left it running
//	me2    the same shape, DEAD-ENDED, and BUILT AFTER the setting went on --
//	       which is the field report's own rig. It fills and stays full, so
//	       whatever a flip-off does to a standing multi-edge network's items
//	       shows up here as a number rather than as a rounding error
//
// IT ASSERTS NOTHING. test/assert-flip.py decides.
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
	surfName = "bbb-flip"

	// setting is the runtime-global this whole suite is about, and valueKey is
	// the one key of the ModSetting table it reads back through.
	setting  = "bbb-multi-edge-parts"
	valueKey = "value"

	// modName and flipMethod are the door the mod opens for this suite. See
	// writeSetting below for why there has to be one.
	modName    = "better-belt-balancer"
	flipMethod = "set-multi-edge-parts"

	hiddenSurface = "bbb-hidden"
)

// ours is everything THE COMPILER places, hidden surface and visible alike. It
// crosses as a `find_entities_filtered` name filter holding four names, which
// the generated `EntitySearchFilters.Name` takes as an array Value.
var ours = []string{
	"bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter",
}

var out = harness.Line{Tag: "[FLIP] "}

// The two directions this suite needs, read through the GENERATED accessors,
// which resolve `defines.direction.*` BY NAME against the running game: a
// define's number is Factorio's own, is not in the API description at all, and
// nothing in this repository writes one down.
var east, south uint32

// Band bases, and the row count the surface is paved to.
const (
	ctrlY = 0
	sokY  = 6  // rows 6..7, x = 0..1
	me1Y  = 14 // rows 14..15, x = 0
	me2Y  = 22 // rows 22..23, x = 0
	rows  = 32
)

// ---------------------------------------------------------------------------
// the rig registry
// ---------------------------------------------------------------------------

// rig is one measured balancer and the tiles its output chests stand on.
//
// IT HOLDS TILE POSITIONS AND NOT ENTITIES, which is the shipped guest's own
// rule reached from the other side: `fk_on_init` runs during `--create` and
// every sample is taken during `--benchmark`, so anything held here crosses a
// save.
//
// A DEAD-ENDED ROW STILL GETS A TILE, and that is what reproduces the Lua's own
// answer rather than working around it. `me2` drains into nothing, so its
// `storage.rigs` entries were `false` and `chest_count(false)` returned -1;
// here the tile where a chest WOULD have stood is recorded, no chest is ever
// built on it, and `harness.ChestCount` returns the same -1 for the same reason
// the estate's convention exists -- a number no real count can produce.
type rig struct {
	Name   string
	Chests []harness.XY
}

// rigs is the registry the Lua kept in `storage.order` and `storage.rigs`, in
// registration order -- which IS the order the sample line reports in, and the
// order test/assert-flip.py parses.
var rigs []rig

func register(name string, chests ...harness.XY) {
	rigs = append(rigs, rig{Name: name, Chests: chests})
}

// setChests replaces one rig's chest list, which `me2` needs: it is registered
// empty at init and built at t=250.
func setChests(name string, chests ...harness.XY) {
	for i := range rigs {
		if rigs[i].Name == name {
			rigs[i].Chests = chests
			return
		}
	}
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
//
// The chest itself is placed WITHOUT `raise_built`, exactly as the Lua had it:
// `put` raised and the two chest helpers did not, and a build event for a chest
// is one more dispatch through the mod under test for nothing.
func source(s fkapi.LuaSurface, x, y int) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	harness.InfinityFilter(c, flowItem, "", 1000)
	put(s, protos.FlipLoader, x+1, y, &east, "output")
}

// sink is a loader facing east into a steel chest, and the chest's tile is what
// the registry remembers.
func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	put(s, protos.FlipLoader, x, y, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

// feed is a source at x = -5 running east along the row, up to but not
// including x0.
func feed(s fkapi.LuaSurface, y, x0 int) {
	source(s, -5, y)
	for x := -3; x <= x0-1; x++ {
		put(s, belt, x, y, &east, "")
	}
}

// drain is a sink at x = 2 (chest at 3) fed from x0 eastwards.
//
// x0 = 2 IS A LEGITIMATE EMPTY RANGE and `sok` uses it: the loader stands
// directly against the east column's parts and there is no belt between them.
func drain(s fkapi.LuaSurface, y, x0 int) harness.XY {
	for x := x0; x <= 1; x++ {
		put(s, belt, x, y, &east, "")
	}
	return sink(s, 2, y)
}

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

// ---------------------------------------------------------------------------
// the setting, read and written from script
// ---------------------------------------------------------------------------

// settingValue is `settings.global["bbb-multi-edge-parts"].value`, as the string
// the log line carries.
//
// READING AN UNDEFINED RUNTIME SETTING RETURNS NIL AND RAISES NOTHING, so
// `absent` is the ordinary answer on an engine that does not define it -- which
// is exactly what this suite is skipped on. It is reported rather than assumed,
// because a run whose setting never existed would satisfy several counts below
// while proving none of them.
//
// The read is the shipped guest's own idiom (guest/go/sedge.go's
// `settingMultiEdge`): `settings.global` is a LuaCustomTable, so this is the raw
// handle plus one index read -- two host calls -- against a whole-dictionary
// attribute that would materialise every runtime setting in the game.
func settingValue() string {
	raw, err := fkapi.Settings.GlobalRaw()
	if err != nil {
		return "absent"
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(setting))
	if err != nil || v.Tag != fkapi.TagMap {
		return "absent"
	}
	for i := range v.Map {
		if v.Map[i].Key.Tag != fkapi.TagString || v.Map[i].Key.Str != valueKey {
			continue
		}
		if v.Map[i].Val.Tag == fkapi.TagBool && v.Map[i].Val.Bool {
			return "true"
		}
		return "false"
	}
	return "absent"
}

func reportSetting(tag string) {
	out.Open("setting tag=").S(tag).S(" value=").S(settingValue()).End()
}

// writeSetting is THE WRITE, AND WHY IT IS A `remote.call` AND NOT AN
// ASSIGNMENT.
//
// `settings.global[k] = v` from THIS mod raises, measured on 2.0.77: "Settings
// can only be changed by the owning player or the mod that made the setting." A
// runtime-global has no owning player, so the mod that DEFINED the setting is
// the only script in the game that may write it -- which means no test mod can
// ever flip it and the whole flip half of guest/go/sedge.go would be reachable
// by a human and by nothing else.
//
// So better-belt-balancer opens the door itself, beside the audit and for the
// same reason (guest/go/commands.go). What crosses is the same
// `writeMultiEdgeSetting` a player's keypress reaches, so this drives the real
// path rather than a stand-in: the write raises
// `on_runtime_mod_setting_changed` SYNCHRONOUSLY, inside the assigning
// statement, so everything the guest does about the flip has happened by the
// time the call returns -- except the deferred flush it asks for, which lands on
// the next tick.
//
// ONE STATUS CHECK WHERE THE LUA HAD NONE AT ALL. The Lua called `remote.call`
// bare, which RAISES on a missing interface or method and would have aborted the
// schedule; `fkapi.RemoteCall` hands the same refusal back as a Status, so a
// mod that did not open the door reads as `accepted=false` and the run gets to
// the assertion that says so.
func writeSetting(on bool, tag string) {
	out.Open("writing tag=").S(tag).S(" value=").B(on).End()
	v, st := fkapi.RemoteCall(modName, flipMethod, fkapi.OfBool(on))
	accepted := st == fkapi.StatusOK && v.Tag == fkapi.TagBool && v.Bool
	out.Open("wrote tag=").S(tag).S(" accepted=").B(accepted).
		S(" value=").S(settingValue()).End()
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

// reportWorld is WHERE THE ITEMS ARE, which is the whole of what the field
// report was about.
//
// `inside` is everything standing in a transport line of anything the COMPILER
// placed, hidden surface and visible alike; `ground` is every loose item on
// every surface. A vetoed flip has to move neither.
//
// Both sweeps are whole-surface: no `Area`, exactly as the Lua had it, which is
// why they call the generated filter directly rather than going through
// `harness.EntitiesIn` -- that helper takes a box because every other count in
// the estate has one.
func reportWorld(tag string) {
	var ground, inside, visible, hidden int64
	itemName := fkapi.OfString("item-on-ground")
	oursName := fkapi.OfArray(
		fkapi.OfString(ours[0]), fkapi.OfString(ours[1]),
		fkapi.OfString(ours[2]), fkapi.OfString(ours[3]),
	)
	for _, s := range harness.SurfacesByIndex() {
		name, err := s.Name()
		if err != nil {
			continue
		}
		if found, err := s.FindEntitiesFiltered(
			fkapi.EntitySearchFilters{Name: &itemName}); err == nil {
			for _, o := range found {
				if _, n, ok := harness.GroundStack(o); ok {
					ground += n
				}
			}
		}
		if found, err := s.FindEntitiesFiltered(
			fkapi.EntitySearchFilters{Name: &oursName}); err == nil {
			for _, o := range found {
				inside += harness.TransportLineItems(o)
				if name == hiddenSurface {
					hidden++
				} else {
					visible++
				}
			}
		}
	}
	out.Open("world tag=").S(tag).S(" tick=").U(harness.Tick()).
		S(" ground=").I(ground).S(" inside=").I(inside).
		S(" interfaces=").I(visible).S(" hidden=").I(hidden).End()
}

// chartState is WHETHER THE FORCE CAN SEE THE GROUND THE PINGED BALANCERS STAND
// ON, AND THE WALL THAT MAKES THAT UNANSWERABLE HERE.
//
// A `[gps=]` is a coordinate and nothing else: clicking one opens the map there
// whether or not the force has charted it, and an uncharted point is BLACK. So
// the mod charts what it pings, and the obvious check is `is_chunk_charted`
// after the message.
//
// IT ANSWERS FALSE FOR EVERYTHING ON A HEADLESS RUN. Measured on 2.0.77 and not
// assumed: with no players, `force.chart` charts nothing, `force.chart_all` over
// a fully generated surface charts nothing, a powered radar charts nothing, and
// NAUVIS'S OWN ORIGIN CHUNK -- which every real game charts at world creation --
// reads uncharted too. A force with no players has no chart to write into. That
// puts the EFFECT behind the same player wall as the flying text and the
// hand-back; what is on this side of it is the guest's own
// `charted N from x,y to x,y`, which the assertion script reads.
//
// SO THIS IS A TRIPWIRE, NOT A MEASUREMENT OF THE FIX. It reports zero before
// and zero after, together with the nauvis control that says why, and
// test/assert-flip.py asserts exactly that -- so the day a Factorio charts
// headlessly the run fails and asks for the real assertion instead of this one.
// That is the shape the `edge` suite's `player-mine-raise ok=false` probe has,
// for the same reason, and `mig21` carries the same tripwire from the other
// engine.
//
// ONE CHUNK PER MULTI-EDGE PART TILE, counted DISTINCT: `is_chunk_charted` takes
// a CHUNK position, all four of those tiles are in chunk (0, 0), and the count
// being over distinct chunks is what makes the three samples comparable.
func chartState(tag string) {
	s := surf()
	force, ok := harness.ForceByName(harness.PlayerForce)
	if !ok {
		harness.Fatal("resolving the player force", harness.PlayerForce)
		return
	}
	f := fkapi.LuaForce{Object: force}
	index, err := f.Index()
	if err != nil {
		harness.Fatal("reading the player force index", fk.LastError())
		return
	}
	players, err := fkapi.Game.Players()
	if err != nil {
		harness.Fatal("reading game.players", fk.LastError())
		return
	}

	var seen []harness.XY
	charted, distinct := 0, 0
	for _, y0 := range [2]int{me1Y, me2Y} {
		for dy := 0; dy <= 1; dy++ {
			c := harness.XY{X: floorDiv32(0), Y: floorDiv32(y0 + dy)}
			dup := false
			for _, p := range seen {
				if p == c {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			seen = append(seen, c)
			distinct++
			if ok, err := f.IsChunkCharted(s.Object,
				fkapi.ChunkPosition{X: int32(c.X), Y: int32(c.Y)}); err == nil && ok {
				charted++
			}
		}
	}

	// THE CONTROL, in the same line: nauvis's origin chunk, generated by world
	// creation and charted by nothing, for the same force.
	nauOK := false
	if nauvis, err := fkapi.Game.GetSurface(fkapi.OfString("nauvis")); err == nil && nauvis != nil {
		origin := fkapi.ChunkPosition{X: 0, Y: 0}
		if gen, err := (fkapi.LuaSurface{Object: *nauvis}).IsChunkGenerated(origin); err == nil && gen {
			if c, err := f.IsChunkCharted(*nauvis, origin); err == nil {
				nauOK = c
			}
		}
	}

	out.Open("chart tag=").S(tag).S(" force=").U(uint64(index)).
		S(" charted=").I(int64(charted)).S(" of=").I(int64(distinct)).
		S(" nauvis_origin=").B(nauOK).S(" players=").I(int64(len(players))).End()
}

// floorDiv32 is a tile coordinate's chunk coordinate. Written out because Go's
// `/` truncates towards zero and a chunk index has to floor: every tile this
// suite asks about is non-negative, and the arm that is never taken here is what
// keeps that from being a latent wrong answer for anybody who moves a band.
func floorDiv32(v int) int {
	if v < 0 {
		return -((-v + 31) / 32)
	}
	return v / 32
}

func auditNow(tag string) {
	harness.Audit(surf(), -12, 0)
	out.Open("audited tag=").S(tag).End()
}

// ---------------------------------------------------------------------------
// the mid-run edits
// ---------------------------------------------------------------------------

// buildMulti is the incumbent's idiom, in two parts: each one carries an input
// on its west face and an output on its east, which is two belts on one tile.
func buildMulti(y0 int, deadEnded bool) []harness.XY {
	s := surf()
	outs := make([]harness.XY, 0, 2)
	for dy := 0; dy <= 1; dy++ {
		put(s, part, 0, y0+dy, nil, "")
		feed(s, y0+dy, 0)
		if deadEnded {
			// One belt tile and nothing after it: the network fills and stays
			// full, so what it is holding when a flip lands is a real quantity.
			// The tile a chest WOULD have stood on is recorded all the same --
			// see the rig type's header.
			put(s, belt, 1, y0+dy, &east, "")
			outs = append(outs, harness.XY{X: 3, Y: y0 + dy})
		} else {
			outs = append(outs, drain(s, y0+dy, 1))
		}
	}
	return outs
}

func buildMe2() {
	out.Open("me2-build begin").End()
	setChests("me2", buildMulti(me2Y, true)...)
	out.Open("me2-build end").End()
}

// stripMulti takes the multi-edge clusters out of the world entirely, so that
// the flip OFF that follows has nothing to veto. A dissolve is a REMOVAL, so its
// items spill -- which is why every ground number in this suite is read as a
// DELTA over the window it belongs to rather than as an absolute.
func stripMulti() {
	out.Open("strip begin").End()
	s := surf()
	n := 0
	for _, y0 := range [2]int{me1Y, me2Y} {
		for dy := 0; dy <= 1; dy++ {
			for _, o := range harness.EntitiesIn(s, harness.InnerBox(0, y0+dy), part) {
				harness.Destroy(o, true)
				n++
			}
		}
	}
	out.Open("strip end removed=").I(int64(n)).End()
}

// secondBelt is A SECOND BELT against a part of `sok` that already has one: the
// ordinary refusal, asked once at the false default the save starts at and once
// after the second flip-off has stuck. Same gesture, same bound, two different
// reasons for the mode to be single.
func secondBelt() {
	out.Open("second-belt begin").End()
	put(surf(), belt, 0, sokY-1, &south, "")
	out.Open("second-belt end").End()
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

var schedule = []harness.Step{
	// The save opens at the false default: me1 is built the incumbent's way and
	// has to be refused.
	{Tick: 60, Do: func() {
		reportSetting("default")
		chartState("default")
		auditNow("default")
	}},

	// ON. The refused cluster's fingerprint never matched, so the requeue the
	// handler makes is what finally compiles it.
	{Tick: 200, Do: func() { writeSetting(true, "on") }},
	{Tick: 210, Do: func() { reportSetting("post-on"); auditNow("post-on") }},

	// ...and a multi-edge balancer BUILT while it is on, which is the shape the
	// field report was made with.
	{Tick: 250, Do: buildMe2},
	{Tick: 270, Do: func() { auditNow("post-me2") }},

	{Tick: 900, Do: func() { report("a"); reportWorld("a") }},
	{Tick: 1300, Do: func() { report("b") }},

	// OFF, with both multi-edge balancers standing and me2 full. THE VETO.
	{Tick: 1400, Do: func() { reportWorld("pre-veto") }},
	{Tick: 1405, Do: func() { writeSetting(false, "off-vetoed") }},
	{Tick: 1406, Do: func() { reportWorld("post-veto") }},
	{Tick: 1410, Do: func() {
		reportSetting("post-veto")
		chartState("post-veto")
		auditNow("post-veto")
	}},
	{Tick: 1420, Do: func() { reportWorld("veto-settled") }},

	// ...and the balancers are still running, which is what "a no-op on the
	// world" has to mean.
	{Tick: 1800, Do: func() { report("c") }},
	{Tick: 2600, Do: func() { report("d"); reportWorld("d") }},

	// Now take the multi-edge clusters away and flip OFF again. With nothing to
	// veto the flip STICKS, and single-edge is enforced from there.
	{Tick: 2700, Do: stripMulti},
	{Tick: 2720, Do: func() { auditNow("post-strip"); reportWorld("post-strip") }},
	{Tick: 2800, Do: func() { writeSetting(false, "off-sticks") }},
	{Tick: 2810, Do: func() { reportSetting("post-sticks"); auditNow("post-sticks") }},
	{Tick: 2820, Do: func() { reportWorld("post-sticks") }},

	// And the bound is back: a second belt on a working single-edge part is
	// refused again, now because the player turned the setting off rather than
	// because it had never been on.
	{Tick: 2900, Do: secondBelt},
	{Tick: 2910, Do: func() { auditNow("post-second-belt") }},

	{Tick: 3100, Do: func() { report("f"); reportWorld("final") }},
	{Tick: 3120, Do: func() {
		reportSetting("final")
		chartState("final")
		auditNow("final")
	}},
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
		ChunkRadius: 4,
		X0:          -16,
		Y0:          -12,
		X1:          16,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()
	rigs = rigs[:0]

	// ctrl: the yardstick.
	feed(s, ctrlY, 0)
	register("ctrl", drain(s, ctrlY, 0))

	// sok: ONE BELT PER PART, and therefore legal in both modes. Two west parts
	// carry the inputs, two east parts carry the outputs.
	sokOut := make([]harness.XY, 0, 2)
	for dy := 0; dy <= 1; dy++ {
		put(s, part, 0, sokY+dy, nil, "")
		put(s, part, 1, sokY+dy, nil, "")
		feed(s, sokY+dy, 0)
		sokOut = append(sokOut, drain(s, sokY+dy, 2))
	}
	register("sok", sokOut...)

	// me1: the incumbent's idiom, draining freely. Refused until the flip.
	register("me1", buildMulti(me1Y, false)...)

	// me2 does not exist yet: it is built at t=250, after the setting is on.
	register("me2")

	reportSetting("init")
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
