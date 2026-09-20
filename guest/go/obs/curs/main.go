// Command bbb-curs-test drives, from a real player's cursor, every gesture
// CLAUDE.md has recorded as interactive-only since the miner's pocket shipped.
//
// It is the estate's SECOND suite with no `--create` phase, and for `mig21`'s
// reason turned round: that one's world cannot be built by the binary on this
// machine, and this one's cannot be built by a MAP GENERATOR at all. A
// `game.players` entry exists only where somebody once connected, so the world
// starts from a save a graphical client made -- test/fixtures-player/, cut down
// by guest/go/obs/fixplayer -- and this mod adds its own rigs to it at
// `fk_on_init`, which fires because the fixture has never seen this mod.
//
// # What the fixture can hold, and what it therefore cannot
//
// The cutter's only route to a file is `--start-server` and `game.server_save`:
// `--benchmark` never writes a save at all. A server load DISCONNECTS every
// player and DESTROYS their character, and a disconnected player cannot be given
// one back -- `player.character = <a character just created>` is refused by name,
// "User isn't connected; can't write character". Measured, and then measured
// again from the other side: `set_controller{type = character, character = ...}`
// returns cleanly, leaves `player.character` nil, and `build_from_cursor` after
// it raises `on_pre_build` and places nothing.
//
// # So the controller is GOD, and that is a measurement rather than a shortcut
//
// A disconnected player CAN hold the god controller, and in it `build_from_cursor`
// and `mine_entity` work. What that costs was measured against the connected
// character it stands in for, on the same gesture -- a balancer part
// fast-replaced onto a loaded belt -- and the answer is nothing:
//
//	                        character (connected)   god (disconnected)
//	on_pre_build                   yes                    yes
//	on_pre_player_mined_item       yes                    yes
//	on_player_mined_entity   buffer{iron-plate x6,        identical
//	                                transport-belt x1}
//	on_player_mined_item           x2                     x2
//	on_built_entity           bbb-balancer-part           identical
//	player_index on all five        1                      1
//	belt / plates / cursor    7->8, 87->93, 5->4->5       identical
//	the mod's four lines      formed, refused,            identical
//	                          dissolved, handed back
//
// The one thing god mode changes is `build_distance`, which is unbounded there
// and 10 for a character. This mod TELEPORTS the player next to every target
// anyway: the reach is not what is under test, and a rig that only works from
// across the map would be a rig a person could not reproduce.
//
// # What it reports, and what it does not
//
// An observer asserts nothing. Every gesture writes a `pre` sample and a `post`
// sample -- the target tile, the player's inventory and cursor, the ground -- and
// test/assert-curs.py decides. What the MOD says about each one is its own
// `[BBB]` lines, which that script reads beside these.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const (
	surfName = "bbb-curs"
	part     = "bbb-balancer-part"
	belt     = "express-transport-belt"
	loader   = protos.CursLoader
	plate    = "iron-plate"

	underground = "express-underground-belt"

	// This mod's own visible edge interface. Bands (h) and (i) read it off a part
	// tile to say whether that face was classified, and band (i) places a pair of
	// its own to stand in for another mod's placeable linked belt.
	nameIface = "bbb-linked-belt"

	// A steel chest holds 48 stacks, so an insert past that is clamped by the
	// engine. `edge` uses the same number for the same reason: what the source
	// actually holds is read back rather than assumed.
	stock = 5000
)

// The bands, in tile rows. Every one is far enough from the next that no rig's
// belts can reach another's parts: the neighbour gate this mod uses is two tiles
// wide, and a gesture that recompiled a rig it was not aimed at would move a
// count nobody was watching.
const (
	// (a) A dead-ended five-belt line and no balancer at all. The part goes on
	// the MIDDLE belt, which is the only tile in the estate where a fast replace
	// takes a belt that is CARRYING something.
	lineA = 0

	// (b) A 2->2 over four parts, and a belt line that ENDS on the tile the part
	// is dropped onto. Ends rather than passes: a part dropped mid-line takes the
	// belt behind it as an input and the belt ahead as an output, which is two
	// belts on one part and a different refusal.
	repEnd = 8

	// (c) A lone part, and a five-part column with a 1->1 at each end. The
	// column's three middle parts carry nothing, so the belt that replaces the
	// centre one is an edge of the part above it and of the part below it and
	// neither is asked for a second.
	lone = 16
	col  = 20

	// (g) A 1->1 row whose two parts each already carry their one belt. The
	// second belt goes on the input part's free NORTH face.
	second = 30

	// (d) The miner's pocket, both field-report shapes. Both are DEAD-ENDED so
	// their networks fill and stay full: a pocket that offered nothing would
	// satisfy every count in this suite and prove none of them.
	pock = 36
	bmin = 44

	// (e) THE PORT LIMIT. A 2x32 block carrying sixty-four input belts, one part
	// below it carrying the single output, and one SPARE part above carrying
	// nothing -- which is where the sixty-fifth belt lands, because every part
	// that has a belt already has its one.
	lim     = 52
	limRows = 32
	limFed  = 2

	// (f) THE MERGE THE MOD CANNOT HONOUR. Two 2x16 blocks of thirty-two input
	// belts each with an output part apiece, one tile apart, and one input belt
	// beside that tile from t=0 -- which is what takes the merged cluster from
	// sixty-four edges to sixty-five.
	brdg     = lim + limRows + 14
	brdgHalf = 16
	brdgFed  = 2

	// (h) THE CURVED EXIT, and the mod portal report it answers: "you have to be
	// very careful not to place a straight belt adjacent to the balancer, as it
	// immediately curves it".
	//
	// THE SPARE ROW IS THE RIG AND NOT PADDING. Under the one-belt-per-part rule
	// a working balancer has NO FREE FACE -- the west part of a row carries its
	// input and the east part its output -- so a belt laid on any of the four
	// working parts is a second belt and is refused before the curve arm is ever
	// asked. The two spare parts carry nothing, so their north faces are the only
	// tiles in the band where a curve can be classified at all. `curveFace` is
	// the row above them, which is where every gesture in the band lands.
	//
	// DEAD-ENDED, like the two pocket rigs, so the machine fills and stays full:
	// a grabbed face belt then shows as plates arriving on a line the player
	// meant to run PAST the balancer, which is the report's own symptom, and a
	// released one has something to spill.
	curveBal   = brdg + 2*brdgHalf + 16
	curveSpare = curveBal - 1
	curveFace  = curveBal - 2

	// (i) A LINKED BELT AT THE REAR. The same rig with a `bbb-linked-belt` output
	// end standing west of the face tile and pointing into it, which is a feeder
	// the engine counts and the probe could not see: `feedsTile` walked the six
	// EDGE types, and linked-belt is deliberately not one of them.
	//
	// This mod's own interfaces can never stand on a probe tile -- an interface
	// stands on a part tile and a part tile is refused two tests earlier -- so
	// what the rig stands in for is somebody else's placeable one.
	lbeltBal   = curveBal + 12
	lbeltSpare = lbeltBal - 1
	lbeltFace  = lbeltBal - 2

	// The engine-only control for band (i): the same two shapes with no balancer
	// within ten tiles, so what they report is Factorio's reading and not ours.
	lbEng = lbeltBal + 12

	// (j) AN UNDERGROUND PAIR TURNED FROM ITS FAR END, the mod portal report of
	// 2026-09-19. A two-part row whose west part carries an ordinary input and
	// whose east part has an underground pair beside it running WEST, so the
	// near half is a second input and the row has no output and no network. The
	// player rotates the FAR half, three tiles out and outside the neighbour
	// gate; the engine swaps both ends, the near half becomes the row's output,
	// and the event names only the far one.
	urot = lbEng + 12

	rows = urot + 12
)

// The tile each gesture is aimed at, so that the observer, the log and the
// assertion script name one thing rather than three.
var (
	lineTarget   = harness.XY{X: -2, Y: lineA}
	repEndTarget = harness.XY{X: 0, Y: repEnd + 2}
	loneTarget   = harness.XY{X: 0, Y: lone}
	colTarget    = harness.XY{X: 0, Y: col + 2}
	secondTarget = harness.XY{X: 0, Y: second - 1}
	bminTarget   = harness.XY{X: 0, Y: bmin + 3}
	limTarget    = harness.XY{X: 0, Y: lim + limRows + 1}
	brdgTarget   = harness.XY{X: 0, Y: brdg + brdgHalf}
	curveTarget  = harness.XY{X: 0, Y: curveFace}
	curveRear    = harness.XY{X: -1, Y: curveFace}
	lbeltTarget  = harness.XY{X: 0, Y: lbeltFace}
	urotNear     = harness.XY{X: 2, Y: urot}
	urotFar      = harness.XY{X: 5, Y: urot}
)

// curveLine is the forward line of band (h): laid west to east, one belt per
// click, starting three tiles before the balancer. Every belt but the first has
// the one behind it in its rear, and the first is not beside a part at all.
var curveLine = []int{-3, -2, -1, 0, 1, 2, 3}

// pockOrder is the `pock` rig taken apart the way a player does it: one part per
// step, the spare first and then row by row, west part before east. Every prefix
// leaves a CONNECTED cluster, so no step is a split, and every step but the last
// two leaves a machine with an input and an output -- a real shrink into a
// smaller butterfly rather than a dissolve.
var pockOrder = [][2]int{{0, 2}, {0, 0}, {1, 0}, {0, 1}, {1, 1}}

var out = harness.Line{Tag: "[BBB-CURS] "}

var dirN, dirE, dirS, dirW uint32
var godController uint32

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirN = fkapi.DefinesDirectionNorth()
	dirE = fkapi.DefinesDirectionEast()
	dirS = fkapi.DefinesDirectionSouth()
	dirW = fkapi.DefinesDirectionWest()
	godController = fkapi.DefinesControllersGod()
}

// ---------------------------------------------------------------------------
// the world
// ---------------------------------------------------------------------------

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32) fkapi.Object {
	return harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Raise: true})
}

func putTyped(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true,
	})
}

func makeSurface() fkapi.LuaSurface {
	return harness.Flat{
		Name:        surfName,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		ChunkRadius: uint32((rows+31)/32) + 3,
		X0:          -18,
		Y0:          -14,
		X1:          22,
		Y1:          rows + 14,
		Tile:        "grass-1",
	}.Make()
}

// source is a full chest and a loader pushing east: the west end of a row.
func source(s fkapi.LuaSurface, y int) {
	c := harness.Place(s, harness.Piece{Name: "steel-chest", X: -6, Y: y})
	harness.InsertInto(c, plate, stock)
	putTyped(s, loader, -5, y, &dirE, "output")
}

// sink is a loader pulling east into a chest: the east end of a row.
func sink(s fkapi.LuaSurface, y int) harness.XY {
	putTyped(s, loader, 5, y, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 6, Y: y})
	return harness.XY{X: 6, Y: y}
}

// ONE BELT PER PART, so a row is TWO parts: the west one carries the input and
// the east one the output.
//
//	-6 chest   -5 loader   -4..-1 belts   0 WEST PART   1 EAST PART
//	2..4 belts             5 loader       6 chest
func feedIn(s fkapi.LuaSurface, y int) {
	source(s, y)
	for x := -4; x <= -1; x++ {
		put(s, belt, x, y, &dirE)
	}
}

func drainOut(s fkapi.LuaSurface, y int) harness.XY {
	for x := 2; x <= 4; x++ {
		put(s, belt, x, y, &dirE)
	}
	return sink(s, y)
}

func feed(s fkapi.LuaSurface, y int) harness.XY {
	feedIn(s, y)
	return drainOut(s, y)
}

// deadEnd is a row that is fed and NOT drained: the belts leaving it go nowhere,
// so the hidden network behind it backs up and stays full.
func deadEnd(s fkapi.LuaSurface, y int) {
	feedIn(s, y)
	for x := 2; x <= 4; x++ {
		put(s, belt, x, y, &dirE)
	}
}

// ---------------------------------------------------------------------------
// the player
// ---------------------------------------------------------------------------

func player() (fkapi.LuaPlayer, bool) {
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(1))
	if err != nil || o == nil {
		return fkapi.LuaPlayer{}, false
	}
	return fkapi.LuaPlayer{Object: *o}, true
}

// arm puts the player in the god controller and on this suite's surface.
//
// The controller is set FIRST and the teleport second, because setting it swaps
// which inventory `get_main_inventory` answers with: everything this suite puts
// in the player's hands has to go into the one the gestures will read back.
func arm() {
	p, ok := player()
	if !ok {
		harness.Fatal("no player 1 in the fixture", fk.LastError())
		return
	}
	if err := p.SetController(fkapi.LuaPlayerSetControllerArgs{Type: godController}); err != nil {
		harness.Fatal("set_controller(god)", fk.LastError())
		return
	}
	s := surf()
	if _, err := p.Teleport(harness.Center(-10, 0), &s.Object, nil, nil, nil); err != nil {
		harness.Fatal("teleporting the player onto "+surfName, fk.LastError())
		return
	}
	reportPlayer("armed")
}

func reportPlayer(tag string) {
	p, ok := player()
	if !ok {
		out.Open("player tag=").S(tag).S(" present=false").End()
		return
	}
	name, _ := p.Name()
	connected, _ := p.Connected()
	ch, _ := p.Character()
	controller, _ := p.ControllerType()
	dist, _ := p.BuildDistance()
	pos, _ := p.Position()
	sobj, _ := p.Surface()
	sname, _ := (fkapi.LuaSurface{Object: sobj}).Name()
	out.Open("player tag=").S(tag).S(" present=true").
		S(" name=").S(name).
		S(" connected=").B(connected).
		S(" character=").B(ch != nil).
		S(" controller=").U(uint64(controller)).
		S(" build_distance=").U(uint64(dist)).
		S(" surface=").S(sname).
		S(" x=").F1(pos.X).S(" y=").F1(pos.Y).End()
}

// stand teleports the player next to a tile, close enough that a CHARACTER's ten
// tiles of reach would cover it. God mode does not need this; the gesture is
// meant to be one a person could make, and a rig that only works from off-screen
// is not.
func stand(xy harness.XY) {
	p, ok := player()
	if !ok {
		return
	}
	s := surf()
	if _, err := p.Teleport(harness.Center(xy.X-2, xy.Y), &s.Object, nil, nil, nil); err != nil {
		out.Open("teleport refused: ").S(fk.LastError()).End()
	}
}

func mainInv() (fkapi.LuaInventory, bool) {
	p, ok := player()
	if !ok {
		return fkapi.LuaInventory{}, false
	}
	o, err := p.GetMainInventory()
	if err != nil || o == nil {
		return fkapi.LuaInventory{}, false
	}
	return fkapi.LuaInventory{Object: *o}, true
}

func cursor() (fkapi.LuaItemStack, bool) {
	p, ok := player()
	if !ok {
		return fkapi.LuaItemStack{}, false
	}
	o, err := p.CursorStack()
	if err != nil || o == nil {
		return fkapi.LuaItemStack{}, false
	}
	return fkapi.LuaItemStack{Object: *o}, true
}

// hold puts an item in the cursor, which is what `build_from_cursor` builds.
func hold(name string, count uint32) {
	c, ok := cursor()
	if !ok {
		harness.Fatal("no cursor stack", fk.LastError())
		return
	}
	stack := fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(name)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(float64(count))},
	)
	if ok, err := c.SetStack(&stack); err != nil || !ok {
		harness.Fatal("cursor_stack.set_stack "+name, fk.LastError())
	}
}

func dropCursor() {
	c, ok := cursor()
	if !ok {
		return
	}
	if err := c.Clear(); err != nil {
		out.Open("cursor clear refused: ").S(fk.LastError()).End()
	}
}

// give puts items in the player's main inventory. They are MINTED, which is why
// this suite's conservation check is per gesture over a box rather than a global
// running total: what matters here is that a gesture moves items between the
// world and the player without creating or destroying any, and every `give` is
// outside a gesture's own window.
func give(name string, count uint32) uint32 {
	inv, ok := mainInv()
	if !ok {
		harness.Fatal("no main inventory to stock", fk.LastError())
		return 0
	}
	n, err := inv.Insert(fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(name)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(float64(count))},
	))
	if err != nil {
		harness.Fatal("inventory.insert "+name, fk.LastError())
		return 0
	}
	return n
}

func held(name string) int64 {
	inv, ok := mainInv()
	if !ok {
		return -1
	}
	v := fkapi.OfString(name)
	n, err := inv.GetItemCount(&v)
	if err != nil {
		return -1
	}
	return int64(n)
}

// cursorHeld is what the cursor is holding, as a name and a count. The hand-back
// lands HERE rather than in the main inventory whenever the cursor already holds
// the same item -- measured -- so a count that ignored it would read a successful
// hand-back as a failed one.
func cursorHeld() (string, int64) {
	c, ok := cursor()
	if !ok {
		return "none", 0
	}
	if v, err := c.ValidForRead(); err != nil || !v {
		return "none", 0
	}
	name, _ := c.Name()
	n, _ := c.Count()
	return name, int64(n)
}

// ---------------------------------------------------------------------------
// the gesture
// ---------------------------------------------------------------------------

// click is THE WHOLE POINT OF THIS SUITE: a build from the player's cursor, which
// is what a mouse click is. It is not `create_entity{fast_replace = true}` --
// handed a replace the engine would refuse, that falls back to CREATING, and the
// engine raises no mine event for what it destroyed.
//
// `build_from_cursor` RETURNS NOTHING AND RAISES NOTHING when the build cannot
// happen: out of reach, or refused by the engine. So what says it landed is the
// world afterwards, which is what every caller samples.
func click(xy harness.XY, dir *uint32) {
	p, ok := player()
	if !ok {
		return
	}
	args := fkapi.LuaPlayerBuildFromCursorArgs{Position: harness.Center(xy.X, xy.Y)}
	if dir != nil {
		args.Direction = dir
	}
	if err := p.BuildFromCursor(args); err != nil {
		out.Open("build_from_cursor refused: ").S(fk.LastError()).End()
	}
}

// mine is the player's own mining gesture, and the one the miner's pocket is
// about. `force = nil`, so a player whose inventory cannot take it does not get
// to mine it -- vanilla's rule, and the mod's own hand-back follows it.
func mine(xy harness.XY, name, typ string) bool {
	p, ok := player()
	if !ok {
		return false
	}
	e, found := harness.FindAt(surf(), xy.X, xy.Y, name, typ)
	if !found {
		out.Open("nothing to mine at ").I(int64(xy.X)).S(",").I(int64(xy.Y)).End()
		return false
	}
	took, err := p.MineEntity(e, nil)
	if err != nil {
		out.Open("mine_entity refused: ").S(fk.LastError()).End()
		return false
	}
	return took
}

// ---------------------------------------------------------------------------
// looking
// ---------------------------------------------------------------------------

// tileState is what is standing on one tile, as the assertion script reads it:
// the entity's name, its direction, its force, and how many items are on its
// transport lines. The direction and the force are here for gesture (a) alone --
// a belt this mod puts back has to come back facing the way it was and belonging
// to whom it did.
func tileState(xy harness.XY) (name string, dir int64, force string, items int64) {
	s := surf()
	all := harness.EntitiesIn(s, harness.InnerBox(xy.X, xy.Y), "")
	if len(all) == 0 {
		return "empty", -1, "none", 0
	}
	// The one entity a player can see. An edge interface stands on a part's own
	// tile, so a tile can hold two of ours; the part is what the gesture is about.
	pick := all[0]
	for _, o := range all {
		if n, err := (fkapi.LuaEntity{Object: o}).Name(); err == nil && n == part {
			pick = o
			break
		}
	}
	e := fkapi.LuaEntity{Object: pick}
	name, _ = e.Name()
	if d, err := e.Direction(); err == nil {
		dir = int64(d)
	}
	if f, err := e.Force(); err == nil {
		if fn, err := (fkapi.LuaForce{Object: f}).Name(); err == nil {
			force = fn
		}
	}
	items = harness.TransportLineItems(pick)
	return
}

// boxItems totals every item inside a band: on belts, in chests, on the ground,
// and inside whatever this mod has built on the hidden surface for the cluster
// that lives there. It is the conservation instrument for one gesture.
//
// THE HIDDEN HALF IS NOT IN IT AND THAT IS DELIBERATE. A hidden network's slot
// is addressed by cluster ROOT, which moves when a cluster merges or splits, so a
// band on the visible surface has no fixed box on the hidden one. What this
// number therefore conserves is the VISIBLE side plus the player, and a gesture
// whose recompile put items back inside the network legitimately moves it. Each
// gesture's own window says which it is.
func boxItems(y0, y1 int) (total, ground int64) {
	s := surf()
	box := harness.Box(-8, float64(y0), 8, float64(y1))
	for _, e := range harness.EntitiesIn(s, box, "") {
		if harness.EntityTypeIs(e, "item-entity") {
			if _, n, got := harness.GroundStack(e); got {
				ground += n
				total += n
			}
			continue
		}
		// `InventoryTotal` answers -1 for an entity with no chest inventory,
		// which is most of what a band holds -- belts, loaders, parts. The
		// estate's own sweeps clamp it for the same reason.
		if n := harness.InventoryTotal(e); n > 0 {
			total += n
		}
		total += harness.TransportLineItems(e)
	}
	return
}

// sample is one line per gesture side: what the target tile holds, what the
// player holds, and what is loose in the band.
func sample(tag string, xy harness.XY, y0, y1 int, items ...string) {
	name, dir, force, onTile := tileState(xy)
	total, ground := boxItems(y0, y1)
	cname, ccount := cursorHeld()
	out.Open("sample tag=").S(tag).
		S(" tile=").S(name).
		S(" dir=").I(dir).
		S(" force=").S(force).
		S(" tile_items=").I(onTile).
		S(" box=").I(total).
		S(" ground=").I(ground).
		S(" cursor=").S(cname).S(":").I(ccount)
	for _, it := range items {
		out.S(" ").S(it).S("=").I(held(it))
	}
	out.End()
}

func auditAt(tag string, x, y int) {
	harness.Audit(surf(), x, y)
	out.Open("audited tag=").S(tag).End()
}

// ---------------------------------------------------------------------------
// (a) a part fast-replaced onto the middle of a loaded belt line
// ---------------------------------------------------------------------------

// lineLoad fills one belt tile to saturation: four items per lane at the belt's
// own 0.25-tile item pitch.
//
// IT RUNS IN THE SAME TICK AS THE GESTURE, and that is what makes the numbers
// exact rather than nearly right. The line's other four belts are EMPTY, so an
// item given a tick to move leaves the middle belt and the buffer the mine
// produces is smaller than what was placed -- measured on the spike this rig came
// from, eight placed and six mined three ticks later. Loaded and replaced in one
// dispatch, nothing has moved and the buffer is the eight.
func lineLoad(xy harness.XY) int64 {
	e, found := harness.FindAt(surf(), xy.X, xy.Y, "", "transport-belt")
	if !found {
		harness.Fatal("no belt to load at the line target", fk.LastError())
		return 0
	}
	stack := fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(plate)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(1)},
	)
	var placed int64
	harness.EachLine(e, func(line fkapi.LuaTransportLine) {
		for _, at := range [4]float32{0.1, 0.35, 0.6, 0.85} {
			if ok, err := line.InsertAt(at, stack, nil); err == nil && ok {
				placed++
			}
		}
	})
	return placed
}

func gRestore() {
	out.Open("gesture begin name=restore").End()
	placed := lineLoad(lineTarget)
	out.Open("line loaded placed=").I(placed).End()
	stand(lineTarget)
	hold(part, 5)
	sample("restore-pre", lineTarget, lineA-3, lineA+3, plate, belt, part)
	click(lineTarget, nil)
	sample("restore-click", lineTarget, lineA-3, lineA+3, plate, belt, part)
	out.Open("gesture end name=restore").End()
}

func gRestoreRead(tag string) {
	sample(tag, lineTarget, lineA-3, lineA+3, plate, belt, part)
}

// ---------------------------------------------------------------------------
// (b) a part onto the END of a belt line beside a working balancer
// ---------------------------------------------------------------------------

func gEndOfLine() {
	out.Open("gesture begin name=end-of-line").End()
	stand(repEndTarget)
	hold(part, 5)
	sample("end-pre", repEndTarget, repEnd-3, repEnd+6, plate, belt, part)
	click(repEndTarget, nil)
	sample("end-click", repEndTarget, repEnd-3, repEnd+6, plate, belt, part)
	out.Open("gesture end name=end-of-line").End()
}

// ---------------------------------------------------------------------------
// (c) a belt over a part
// ---------------------------------------------------------------------------

func gLone() {
	out.Open("gesture begin name=lone").End()
	stand(loneTarget)
	hold(belt, 5)
	sample("lone-pre", loneTarget, lone-3, lone+3, plate, belt, part)
	click(loneTarget, &dirS)
	sample("lone-click", loneTarget, lone-3, lone+3, plate, belt, part)
	out.Open("gesture end name=lone").End()
}

func gColumn() {
	out.Open("gesture begin name=column").End()
	stand(colTarget)
	hold(belt, 5)
	sample("col-pre", colTarget, col-3, col+7, plate, belt, part)
	click(colTarget, &dirS)
	sample("col-click", colTarget, col-3, col+7, plate, belt, part)
	out.Open("gesture end name=column").End()
}

// ---------------------------------------------------------------------------
// (g) a second belt on a part that already has one
// ---------------------------------------------------------------------------

func gSecondBelt() {
	out.Open("gesture begin name=second-belt").End()
	stand(secondTarget)
	hold(belt, 5)
	sample("second-pre", secondTarget, second-4, second+3, plate, belt, part)
	click(secondTarget, &dirS)
	sample("second-click", secondTarget, second-4, second+3, plate, belt, part)
	out.Open("gesture end name=second-belt").End()
}

// ---------------------------------------------------------------------------
// (d) the miner's pocket
// ---------------------------------------------------------------------------

// gPockMine is the field report's own gesture: a saturated balancer taken apart
// ONE PART PER TICK, by the player, with `mine_entity`.
//
// Each of the first four steps is a SHRINK -- the machine recompiles into a
// smaller butterfly and hands back less than it drained -- and only the fifth is
// the dissolve. That is the whole of what the first report was about: the pocket
// shipped crediting the miner on the dissolve alone, by which time there is
// almost nothing left in the machine.
func gPockMine(i int) func() {
	return func() {
		xy := pockOrder[i-1]
		target := harness.XY{X: xy[0], Y: pock + xy[1]}
		stand(target)
		took := mine(target, part, "")
		out.Open("pock-mine step=").I(int64(i)).
			S(" x=").I(int64(target.X)).S(" y=").I(int64(target.Y)).
			S(" took=").B(took).End()
		sample("pock-"+step(i), target, pock-3, pock+6, plate, belt, part)
	}
}

func step(i int) string {
	return [...]string{"1", "2", "3", "4", "5"}[i-1]
}

// gBminAdd is the SECOND field report: an output belt laid by cursor on the spare
// edgeless part of a running balancer. It takes the machine from two ports to
// three, so P goes 2 -> 4 and the butterfly doubles.
func gBminAdd() {
	out.Open("gesture begin name=bmin-add").End()
	stand(bminTarget)
	hold(belt, 5)
	sample("bmin-pre", bminTarget, bmin-3, bmin+6, plate, belt, part)
	click(bminTarget, &dirS)
	sample("bmin-add", bminTarget, bmin-3, bmin+6, plate, belt, part)
	out.Open("gesture end name=bmin-add").End()
}

// ... and mined off again, which is the report. P goes back 4 -> 2, the machine
// HALVES, the reinsertion legitimately runs out of room, and what will not fit is
// offered to the player who mined it before it reaches the floor.
func gBminMine() {
	out.Open("gesture begin name=bmin-mine").End()
	stand(bminTarget)
	took := mine(bminTarget, "", "transport-belt")
	out.Open("bmin-mine took=").B(took).End()
	sample("bmin-mine", bminTarget, bmin-3, bmin+6, plate, belt, part)
	out.Open("gesture end name=bmin-mine").End()
}

// ---------------------------------------------------------------------------
// (e) the sixty-fifth belt
// ---------------------------------------------------------------------------

func gLimAdd() {
	out.Open("gesture begin name=lim").End()
	stand(limTarget)
	hold(belt, 5)
	sample("lim-pre", limTarget, lim-5, lim+limRows+4, plate, belt, part)
	click(limTarget, &dirN)
	sample("lim-click", limTarget, lim-5, lim+limRows+4, plate, belt, part)
	out.Open("gesture end name=lim").End()
}

// gLimFull is the NEGATIVE, and the one gesture in this suite whose expected
// outcome is that the refused piece STAYS STANDING. The mod mines a hand-back
// with `force = nil`, which is vanilla's rule: a player whose inventory cannot
// take it does not get to mine it.
//
// THE ORDER IS EMPTY, HOLD ONE, FILL, CLICK -- and the first run got it wrong in
// a way worth writing down, because nothing about it is obvious. Factorio REFILLS
// AN EMPTIED CURSOR FROM THE MAIN INVENTORY: place the last belt from a cursor
// while the inventory holds more and the engine moves them into the cursor,
// which FREES the slot they were in and gives the hand-back somewhere to land.
// Measured: main went two belts to zero across the click and the piece came back
// into main anyway. So the inventory must hold NO belt of its own before the
// cursor is loaded with exactly one, and the filler goes in after that.
func gLimFull() {
	out.Open("gesture begin name=lim-full").End()
	stand(limTarget)
	emptyInventory()
	hold(belt, 1)
	fillInventory()
	sample("limfull-pre", limTarget, lim-5, lim+limRows+4, plate, belt, part)
	click(limTarget, &dirN)
	sample("limfull-click", limTarget, lim-5, lim+limRows+4, plate, belt, part)
	out.Open("gesture end name=lim-full").End()
}

// fillInventory stuffs the main inventory with steel chests until the engine
// stops taking any, and reports both what it took and whether ONE more would fit.
//
// THAT SECOND NUMBER IS THE SIGNAL AND `is_full` IS NOT. Measured: an inventory
// whose eighty slots were all occupied answered `is_full = false`, because a
// partly-filled stack still has room in it -- which is a true answer to a
// different question from the one this gesture asks. What the hand-back needs is
// that ONE steel chest cannot be inserted, and that is what `spare` says.
func fillInventory() {
	var total int64
	for i := 0; i < 16; i++ {
		n := give("steel-chest", 4000)
		total += int64(n)
		if n == 0 {
			break
		}
	}
	spare := give("steel-chest", 1)
	full := false
	if inv, ok := mainInv(); ok {
		full, _ = inv.IsFull()
	}
	out.Open("inventory filled=").I(total).S(" spare=").U(uint64(spare)).
		S(" is_full=").B(full).End()
}

func emptyInventory() {
	inv, ok := mainInv()
	if !ok {
		return
	}
	if err := inv.Clear(); err != nil {
		out.Open("inventory clear refused: ").S(fk.LastError()).End()
	}
	dropCursor()
}

// ---------------------------------------------------------------------------
// (f) the bridge
// ---------------------------------------------------------------------------

func gBrdg() {
	out.Open("gesture begin name=bridge").End()
	stand(brdgTarget)
	hold(part, 5)
	sample("brdg-pre", brdgTarget, brdg-5, brdg+2*brdgHalf+5, plate, belt, part)
	click(brdgTarget, nil)
	sample("brdg-click", brdgTarget, brdg-5, brdg+2*brdgHalf+5, plate, belt, part)
	out.Open("gesture end name=bridge").End()
}

// ---------------------------------------------------------------------------
// (h) and (i) the curved exit
// ---------------------------------------------------------------------------

// curveSample is band (h) and (i)'s own reporting line, and it is deliberately
// NOT `sample`: the release in (h3) spills on purpose, and every `sample` tag in
// this suite is asserted to have found nothing on the ground.
//
// `iface_w` and `iface_e` are the DIRECT statement of a grab. An edge interface
// stands on the cluster's own tile, so one on a spare part's tile IS that part's
// north face having been classified; a compile line says the same thing from the
// mod's side and the two are asserted together.
func curveSample(tag string, face, spare int) {
	s := surf()
	name, shape := "empty", "none"
	if o, ok := harness.FindAt(s, 0, face, "", "transport-belt"); ok {
		name, _ = (fkapi.LuaEntity{Object: o}).Name()
		shape = shapeOf(o)
	}
	_, w := harness.FindOnTile(s, nameIface, 0, spare)
	_, e := harness.FindOnTile(s, nameIface, 1, spare)
	var line, ground int64
	for x := -3; x <= 4; x++ {
		if o, ok := harness.FindAt(s, x, face, "", "transport-belt"); ok {
			line += harness.TransportLineItems(o)
		}
	}
	for _, o := range harness.EntitiesIn(s, harness.Box(-8, float64(face-2), 8, float64(face+5)), "") {
		if harness.EntityTypeIs(o, "item-entity") {
			if _, n, got := harness.GroundStack(o); got {
				ground += n
			}
		}
	}
	out.Open("curve tag=").S(tag).S(" face=").I(int64(face)).
		S(" tile=").S(name).S(" shape=").S(shape).
		S(" iface_w=").B(w).S(" iface_e=").B(e).
		S(" line=").I(line).S(" ground=").I(ground).End()
}

// shapeOf asks the ENGINE what a belt's rendered shape is. A curve is the whole
// question the band is about and `belt_shape` is the only thing that states it
// directly.
func shapeOf(o fkapi.Object) string {
	e := fkapi.LuaEntity{Object: o}
	for _, want := range [3]string{"straight", "left", "right"} {
		if is, err := e.BeltShapeIs(want); err == nil && is {
			return want
		}
	}
	return "?"
}

// clearFace destroys the belts a gesture put on a face row, raising the event so
// the mod recompiles around what is left.
func clearFace(face int) {
	s := surf()
	for x := -3; x <= 4; x++ {
		if o, ok := harness.FindAt(s, x, face, "", "transport-belt"); ok {
			harness.Destroy(o, true)
		}
	}
}

// (h1) THE REPORT: one belt clicked onto a free face, direction along the face,
// nothing else anywhere near it. Its rear is empty by construction -- it is the
// head of a line that does not exist yet -- so the engine curves it towards the
// interface and the mod classifies it as an output.
func gCurveFace() {
	out.Open("gesture begin name=curve-face").End()
	stand(curveTarget)
	hold(belt, 20)
	curveSample("curve-face-pre", curveFace, curveSpare)
	click(curveTarget, &dirE)
	curveSample("curve-face-click", curveFace, curveSpare)
	out.Open("gesture end name=curve-face").End()
}

// (h2) THE LINE THE REPORT MEANT TO LAY: the same row built west to east, one
// belt per click, starting three tiles clear of the balancer. Every belt that
// reaches a free face has the one behind it in its rear already, so none of them
// is ever a curve.
func gCurveLay(i int) func() {
	return func() {
		x := curveLine[i]
		tag := "curve-line-" + digit(i)
		stand(harness.XY{X: x, Y: curveFace})
		hold(belt, 20)
		click(harness.XY{X: x, Y: curveFace}, &dirE)
		curveSample(tag, curveFace, curveSpare)
	}
}

func digit(i int) string { return [...]string{"0", "1", "2", "3", "4", "5", "6"}[i] }

// (h3) ... AND THE REMEDY. The grabbed belt is given a belt behind it, which is
// what the player wanted in the first place. Its rear is fed, the curve arm
// declines a side-load, the port goes away and the machine recompiles 2->2.
func gCurveRelease() {
	out.Open("gesture begin name=curve-release").End()
	stand(curveRear)
	hold(belt, 20)
	curveSample("curve-rel-pre", curveFace, curveSpare)
	click(curveRear, &dirE)
	curveSample("curve-rel-click", curveFace, curveSpare)
	out.Open("gesture end name=curve-release").End()
}

// (i) the linked belt at the rear.
func gLinkedRear() {
	out.Open("gesture begin name=linked-rear").End()
	stand(lbeltTarget)
	hold(belt, 20)
	curveSample("lb-pre", lbeltFace, lbeltSpare)
	click(lbeltTarget, &dirE)
	curveSample("lb-click", lbeltFace, lbeltSpare)
	out.Open("gesture end name=linked-rear").End()
}

// engSample is the engine-only control: what Factorio makes of the same two
// shapes with no balancer within ten tiles. It is what says the rig is really
// curve-eligible and the linked belt is really a feeder, rather than the mod
// having declined for some reason of its own.
func engSample(tag string, y int) {
	s := surf()
	shape, rear := "no-belt", "empty"
	if o, ok := harness.FindAt(s, 0, y, "", "transport-belt"); ok {
		shape = shapeOf(o)
	}
	for _, o := range harness.EntitiesIn(s, harness.InnerBox(-1, y), "") {
		if n, err := (fkapi.LuaEntity{Object: o}).Name(); err == nil {
			rear = n
		}
	}
	out.Open("engine tag=").S(tag).S(" y=").I(int64(y)).
		S(" shape=").S(shape).S(" rear=").S(rear).End()
}

// ---------------------------------------------------------------------------
// the world, built once
// ---------------------------------------------------------------------------

// curveRig is bands (h) and (i): a dead-ended 2->2 over four parts, a spare row
// of two EDGELESS parts above it whose north faces are the gesture row, and
// nothing on the gesture row at all.
func curveRig(s fkapi.LuaSurface, bal int) {
	for r := 0; r <= 1; r++ {
		put(s, part, 0, bal+r, nil)
		put(s, part, 1, bal+r, nil)
		deadEnd(s, bal+r)
	}
	put(s, part, 0, bal-1, nil)
	put(s, part, 1, bal-1, nil)
}

// linkedPair places a `bbb-linked-belt` output end at (x, y) facing `dir` and
// its input partner five tiles west, and connects them.
//
// CONNECTING IT IS NOT WHAT MAKES IT A FEEDER and the control row says so: an
// UNCONNECTED output end keeps a belt straight too, measured on 2.0.77. The pair
// is connected anyway so that the rig is a feeder in the ordinary sense as well
// as in the engine's shape reading.
func linkedPair(s fkapi.LuaSurface, x, y int, dir *uint32) {
	o := putTyped(s, nameIface, x, y, dir, "output")
	i := putTyped(s, nameIface, x-5, y, dir, "input")
	if err := (fkapi.LuaEntity{Object: o}).ConnectLinkedBelts(&i); err != nil {
		harness.Fatal("connect_linked_belts", fk.LastError())
	}
}

func buildCurve(s fkapi.LuaSurface) {
	curveRig(s, curveBal)

	curveRig(s, lbeltBal)
	linkedPair(s, -1, lbeltFace, &dirE)

	// The control, twice: the same belt-and-feeder with a linked belt output end
	// at its rear, and with nothing there.
	put(s, belt, 0, lbEng, &dirE)
	put(s, belt, 0, lbEng+1, &dirN)
	linkedPair(s, -1, lbEng, &dirE)

	put(s, belt, 0, lbEng+4, &dirE)
	put(s, belt, 0, lbEng+5, &dirN)
}

// buildUrot is band (j)'s row. The far half is at x=5 and the near one at x=2,
// so the rotated entity is four tiles from the nearest part and the two-tile
// gate cannot see it. The input half is placed first so the output half pairs
// with it rather than with nothing.
func buildUrot(s fkapi.LuaSurface) {
	put(s, part, 0, urot, nil)
	put(s, part, 1, urot, nil)
	feedIn(s, urot)
	putTyped(s, underground, urotFar.X, urot, &dirW, "input")
	putTyped(s, underground, urotNear.X, urot, &dirW, "output")
	putTyped(s, loader, 6, urot, &dirE, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 7, Y: urot})
}

// urotSample says which end the NEAR half is, which is the thing the gesture
// moves without touching it.
func urotSample(tag string) {
	s := surf()
	line := out.Open("urot tag=").S(tag)
	if e, found := harness.FindOnTile(s, underground, urotNear.X, urotNear.Y); found {
		isOut, _ := (fkapi.LuaEntity{Object: e}).BeltToGroundTypeIs("output")
		line = line.S(" near-output=").B(isOut)
	} else {
		line = line.S(" near-output=absent")
	}
	_, iface := harness.FindOnTile(s, nameIface, 1, urot)
	line.S(" iface=").B(iface).End()
}

func gUrot() {
	out.Open("gesture begin name=urot").End()
	stand(urotFar)
	urotSample("urot-pre")
	p, ok := player()
	e, found := harness.FindOnTile(surf(), underground, urotFar.X, urotFar.Y)
	turned := false
	if ok && found {
		turned, _ = (fkapi.LuaEntity{Object: e}).Rotate(fkapi.LuaEntityRotateArgs{ByPlayer: &p.Object})
	}
	out.Open("urot-rotate turned=").B(turned).End()
	urotSample("urot-turned")
}

func buildLine(s fkapi.LuaSurface) {
	// A WEST-facing dead-ended line: the head is x=-4 and nothing feeds it, so
	// what `lineLoad` puts on the middle belt has nowhere to go and what the mod
	// puts back there stays empty. Five belts, so the target has two behind it
	// and two in front.
	for x := -4; x <= 0; x++ {
		put(s, belt, x, lineA, &dirW)
	}
}

func buildWorld(s fkapi.LuaSurface) {
	buildLine(s)

	// (b)
	for r := 0; r <= 1; r++ {
		put(s, part, 0, repEnd+r, nil)
		put(s, part, 1, repEnd+r, nil)
		feed(s, repEnd+r)
	}
	harness.InsertInto(harness.Place(s,
		harness.Piece{Name: "steel-chest", X: -6, Y: repEnd + 2}), plate, stock)
	putTyped(s, loader, -5, repEnd+2, &dirE, "output")
	for x := -4; x <= 0; x++ {
		put(s, belt, x, repEnd+2, &dirE)
	}

	// (c1) one part, alone.
	put(s, part, loneTarget.X, loneTarget.Y, nil)

	// (c2) a five-part column with a 1->1 at each end. The three middle parts
	// carry nothing, which is what lets the belt that replaces the centre one be
	// an edge of the part above it AND of the part below it without either being
	// asked for a second.
	for r := 0; r <= 4; r++ {
		put(s, part, 0, col+r, nil)
	}
	put(s, part, 1, col, nil)
	put(s, part, 1, col+4, nil)
	feed(s, col)
	feed(s, col+4)

	// (g) a 1->1 row, both parts carrying their one belt.
	put(s, part, 0, second, nil)
	put(s, part, 1, second, nil)
	feed(s, second)

	// (d1) a 2->2 over four parts, DEAD-ENDED so it fills and stays full, plus a
	// spare part with no belt at all -- which is what `pockOrder` mines first and
	// what makes every later step a shrink rather than a split.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, pock+r, nil)
		put(s, part, 1, pock+r, nil)
		deadEnd(s, pock+r)
	}
	put(s, part, 0, pock+2, nil)

	// (d2) the same shape again, and the spare part here is where the third
	// output belt lands: under the one-belt rule the four working parts have no
	// free face between them.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, bmin+r, nil)
		put(s, part, 1, bmin+r, nil)
		deadEnd(s, bmin+r)
	}
	put(s, part, 0, bmin+2, nil)
}

func buildLim(s fkapi.LuaSurface) {
	// THE BIGGEST BALANCER THIS MOD BUILDS, one belt short of refusing: sixty-four
	// input belts over a 2x32 block, one output below, one SPARE part above.
	put(s, part, 0, lim-1, nil)
	put(s, belt, 0, lim-2, &dirN)
	putTyped(s, loader, 0, lim-3, &dirN, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: lim - 4})
	for r := 0; r < limRows; r++ {
		put(s, part, 0, lim+r, nil)
		put(s, part, 1, lim+r, nil)
		put(s, belt, -1, lim+r, &dirE)
		put(s, belt, 2, lim+r, &dirW)
	}
	put(s, part, 0, lim+limRows, nil)
	for r := 0; r < limFed; r++ {
		source(s, lim+r)
		for x := -4; x <= -2; x++ {
			put(s, belt, x, lim+r, &dirE)
		}
	}
}

func buildBrdg(s fkapi.LuaSurface) {
	// TWO balancers with a one-tile gap, whose merge is over the limit. The gap
	// tile carries ONE input belt beside it from t=0 -- an edge of nothing until a
	// part stands there, and then the one that takes the merged cluster to
	// sixty-five.
	gap := brdg + brdgHalf
	foot := brdg + 2*brdgHalf
	put(s, part, 0, brdg-1, nil)
	put(s, belt, 0, brdg-2, &dirN)
	putTyped(s, loader, 0, brdg-3, &dirN, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: brdg - 4})
	for r := 0; r <= 2*brdgHalf; r++ {
		if r == brdgHalf {
			continue
		}
		put(s, part, 0, brdg+r, nil)
		put(s, part, 1, brdg+r, nil)
		put(s, belt, -1, brdg+r, &dirE)
		put(s, belt, 2, brdg+r, &dirW)
	}
	put(s, belt, -1, gap, &dirE)
	put(s, part, 0, foot+1, nil)
	put(s, belt, 0, foot+2, &dirS)
	putTyped(s, loader, 0, foot+3, &dirS, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: foot + 4})
	for r := 0; r < brdgFed; r++ {
		for _, y := range []int{brdg + r, gap + 1 + r} {
			source(s, y)
			for x := -4; x <= -2; x++ {
				put(s, belt, x, y, &dirE)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

// t0 is where the gestures start. Everything before it is the rigs filling: the
// `pock` and `bmin` networks have to be FULL before a player takes them apart, or
// the pocket would be offered nothing and every number about it would be a zero
// that looks like a pass.
const t0 = 620

var schedule = []harness.Step{
	{Tick: 1, Do: arm},
	{Tick: 2, Do: func() { auditAt("armed", 12, 0) }},

	// (a) A part onto the middle of a loaded belt line. The load and the click are
	// one dispatch; the three reads after it are what says the restored belt stays
	// as it came back.
	{Tick: t0, Do: gRestore},
	{Tick: t0 + 1, Do: func() { gRestoreRead("restore-1") }},
	{Tick: t0 + 2, Do: func() { gRestoreRead("restore-2") }},
	{Tick: t0 + 8, Do: func() { gRestoreRead("restore-8") }},
	{Tick: t0 + 10, Do: func() { auditAt("post-restore", 12, 0) }},

	// (b)
	{Tick: t0 + 40, Do: gEndOfLine},
	{Tick: t0 + 44, Do: func() { sample("end-1", repEndTarget, repEnd-3, repEnd+6, plate, belt, part) }},
	{Tick: t0 + 46, Do: func() { auditAt("post-end", 12, 0) }},
	// ... and the rate of the machine it became, over a window that opens well
	// after the new port's own belts have filled.
	{Tick: t0 + 400, Do: func() { rate("end-a", endChests...) }},

	// (c)
	{Tick: t0 + 60, Do: gLone},
	{Tick: t0 + 64, Do: func() { sample("lone-1", loneTarget, lone-3, lone+3, plate, belt, part) }},
	{Tick: t0 + 66, Do: func() { auditAt("post-lone", 12, 0) }},

	{Tick: t0 + 80, Do: gColumn},
	{Tick: t0 + 84, Do: func() { sample("col-1", colTarget, col-3, col+7, plate, belt, part) }},
	{Tick: t0 + 86, Do: func() { auditAt("post-col", 12, 0) }},

	// (g)
	{Tick: t0 + 110, Do: gSecondBelt},
	{Tick: t0 + 114, Do: func() { sample("second-1", secondTarget, second-4, second+3, plate, belt, part) }},
	{Tick: t0 + 116, Do: func() { auditAt("post-second", 12, 0) }},

	// (f) The bridge, before either pocket leg: a refused merge must cost NOTHING,
	// and the ground is cumulative over a run.
	{Tick: t0 + 140, Do: func() { rate("brdg-a", brdgChests...) }},
	{Tick: t0 + 150, Do: gBrdg},
	{Tick: t0 + 154, Do: func() { sample("brdg-1", brdgTarget, brdg-5, brdg+2*brdgHalf+5, plate, belt, part) }},
	{Tick: t0 + 156, Do: func() { auditAt("post-brdg", 12, 0) }},
	{Tick: t0 + 402, Do: func() { rate("brdg-b", brdgChests...) }},

	// (e) The sixty-fifth belt, then the same gesture with nowhere to put it back.
	{Tick: t0 + 200, Do: func() { rate("lim-a", limChest) }},
	{Tick: t0 + 210, Do: gLimAdd},
	{Tick: t0 + 214, Do: func() { sample("lim-1", limTarget, lim-5, lim+limRows+4, plate, belt, part) }},
	{Tick: t0 + 216, Do: func() { auditAt("post-lim", 12, 0) }},
	{Tick: t0 + 450, Do: func() { rate("lim-b", limChest) }},

	{Tick: t0 + 470, Do: gLimFull},
	{Tick: t0 + 474, Do: func() { sample("limfull-1", limTarget, lim-5, lim+limRows+4, plate, belt, part) }},
	{Tick: t0 + 476, Do: func() { auditAt("post-limfull", 12, 0) }},
	// The filler goes again before the pocket legs, whose whole measurement is
	// what arrives in an inventory that has room.
	{Tick: t0 + 480, Do: emptyInventory},

	// (d) The two field reports. Last, because both put items on the ground on
	// purpose and every tag above asserts that it is empty.
	{Tick: t0 + 520, Do: gBminAdd},
	{Tick: t0 + 524, Do: func() { auditAt("post-bmin-add", 12, 0) }},
	{Tick: t0 + 1020, Do: gBminMine},
	{Tick: t0 + 1024, Do: func() { sample("bmin-1", bminTarget, bmin-3, bmin+6, plate, belt, part) }},
	{Tick: t0 + 1026, Do: func() { auditAt("post-bmin-mine", 12, 0) }},

	{Tick: t0 + 1060, Do: func() { sample("pock-pre", harness.XY{X: 0, Y: pock}, pock-3, pock+6, plate, belt, part) }},
	{Tick: t0 + 1064, Do: gPockMine(1)},
	{Tick: t0 + 1068, Do: gPockMine(2)},
	{Tick: t0 + 1072, Do: gPockMine(3)},
	{Tick: t0 + 1076, Do: gPockMine(4)},
	{Tick: t0 + 1080, Do: gPockMine(5)},
	{Tick: t0 + 1084, Do: func() { auditAt("post-pock", 12, 0) }},

	// (h) The curved exit, in the gap while (d2)'s network fills. The bands are
	// forty tiles clear of every other rig, so nothing here touches a count above
	// it -- and (h3) SPILLS on purpose, which is the one window in this suite
	// where the mod may.
	{Tick: t0 + 540, Do: func() { out.Open("gesture begin name=curve-line").End() }},
	{Tick: t0 + 542, Do: gCurveLay(0)},
	{Tick: t0 + 544, Do: gCurveLay(1)},
	{Tick: t0 + 546, Do: gCurveLay(2)},
	{Tick: t0 + 548, Do: gCurveLay(3)},
	{Tick: t0 + 550, Do: gCurveLay(4)},
	{Tick: t0 + 552, Do: gCurveLay(5)},
	{Tick: t0 + 554, Do: gCurveLay(6)},
	{Tick: t0 + 570, Do: func() { curveSample("curve-line-settled", curveFace, curveSpare) }},
	{Tick: t0 + 572, Do: func() { out.Open("gesture end name=curve-line").End(); clearFace(curveFace) }},
	{Tick: t0 + 576, Do: func() { auditAt("post-curve-line", 12, 0) }},

	{Tick: t0 + 590, Do: gCurveFace},
	{Tick: t0 + 592, Do: func() { curveSample("curve-face-2", curveFace, curveSpare) }},
	{Tick: t0 + 594, Do: func() { auditAt("post-curve-face", 12, 0) }},
	// A hundred and thirty ticks of it standing as a live port, which is what
	// fills the three-port network and puts plates on a line the player meant to
	// run past the machine.
	{Tick: t0 + 720, Do: func() { curveSample("curve-face-filled", curveFace, curveSpare) }},

	{Tick: t0 + 730, Do: gCurveRelease},
	{Tick: t0 + 732, Do: func() { curveSample("curve-rel-2", curveFace, curveSpare) }},
	{Tick: t0 + 740, Do: func() { curveSample("curve-rel-10", curveFace, curveSpare) }},
	{Tick: t0 + 742, Do: func() { auditAt("post-curve", 12, 0) }},

	// (i) the linked belt at the rear, and the engine's own reading of the shape.
	{Tick: t0 + 760, Do: gLinkedRear},
	{Tick: t0 + 762, Do: func() { curveSample("lb-2", lbeltFace, lbeltSpare) }},
	{Tick: t0 + 770, Do: func() { curveSample("lb-10", lbeltFace, lbeltSpare) }},
	{Tick: t0 + 772, Do: func() { auditAt("post-lb", 12, 0) }},
	{Tick: t0 + 780, Do: func() {
		engSample("linked-rear", lbEng)
		engSample("empty-rear", lbEng+4)
	}},

	// (j) the underground pair turned from its far end, in the same gap.
	{Tick: t0 + 800, Do: gUrot},
	{Tick: t0 + 802, Do: func() { urotSample("urot-2") }},
	{Tick: t0 + 804, Do: func() { auditAt("post-urot", 12, 0) }},
	{Tick: t0 + 1000, Do: func() { rate("urot", harness.XY{X: 7, Y: urot}) }},

	{Tick: t0 + 1100, Do: func() { reportPlayer("final"); auditAt("final", 12, 0) }},
}

// rate totals named sink chests, which is what says a balancer is still
// DELIVERING across an edit rather than merely standing.
//
// THE TILES ARE PASSED IN RATHER THAN DERIVED. A row of the ordinary rigs drains
// east into a chest at x=6; `lim` and `brdg` each drain their whole block through
// ONE port, north or south into a chest at x=0, which is the shape that makes
// them a 64->1 and a 32->1 at all.
func rate(tag string, chests ...harness.XY) {
	s := surf()
	var total int64
	for _, xy := range chests {
		if c, found := harness.FindOnTile(s, "steel-chest", xy.X, xy.Y); found {
			if n := harness.InventoryTotal(c); n > 0 {
				total += n
			}
		}
	}
	out.Open("rate tag=").S(tag).S(" delivered=").I(total).End()
}

// The sinks each band is measured at.
var (
	endChests  = []harness.XY{{X: 6, Y: repEnd}, {X: 6, Y: repEnd + 1}}
	limChest   = harness.XY{X: 0, Y: lim - 4}
	brdgChests = []harness.XY{{X: 0, Y: brdg - 4}, {X: 0, Y: brdg + 2*brdgHalf + 4}}
)

//go:wasmexport fk_on_init
func onInit() {
	s := makeSurface()
	buildWorld(s)
	buildLim(s)
	buildBrdg(s)
	buildCurve(s)
	buildUrot(s)

	// `--create` never reaches a tick, and this suite has no `--create` -- but the
	// fixture's own first tick is the benchmark's, so without a marker here every
	// network in the save would compile on it. The marker drains the queue inside
	// this dispatch instead.
	harness.Audit(s, 12, 0)
	out.Open("mark tag=init").End()
	reportPlayer("init")
	out.Open("plan t0=").I(t0).S(" end_tick=").I(t0 + 1100).S(" rows=").I(rows).End()
}

// start is the first tick this guest sees, and THE SCHEDULE IS RELATIVE TO IT.
//
// Every other suite in the estate creates its own save, so its world begins at
// tick 0 and an absolute schedule is the same thing. This one's begins at
// whatever tick the client stopped playing on -- ninety thousand and some on the
// save the committed fixture was cut from -- so an absolute schedule fires
// nothing at all. Measured: 1,800 ticks, three log lines, and every gesture
// still in the future.
var start uint64

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	t := fkapi.ReadOnTick(ptr).Tick
	if start == 0 {
		start = t
	}
	harness.Run(schedule, t-start)
}

func main() {}
