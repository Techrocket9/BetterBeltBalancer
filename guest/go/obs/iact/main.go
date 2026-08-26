// Command bbb-interactive-setup stages the gestures a headless Factorio cannot
// make, and the scenes the mod portal is captured from.
//
// A COMPILED GO OBSERVER, not a Lua staging mod. It is the same program the
// hand-written `test/interactive/bbb-interactive-setup/control.lua` was, rig for
// rig and log line for log line; what changed is that there is no hand-written
// Lua left in this repository, on either side of the boundary.
// agents/estate-port.md is the programme.
//
// IT IS NOT A SUITE, and it is the only thing under guest/go/obs that is not.
// Every other observer builds a world so that an assertion script can read the
// mod's own log lines back; this one builds a world so that a HUMAN can walk
// test/interactive/README.md. Every trigger in the gesture bands needs a PLAYER
// -- game.get_player resolves to nothing in a --create, and
// on_player_mined_entity/on_built_entity with a player_index cannot be raised
// from script -- so the suites pin the arithmetic and the quantities, and a
// human pins the trigger. This mod exists to make that human check cost thirty
// seconds per gesture instead of ten minutes of setup: it stages the rigs beside
// spawn on a fresh world, hands the player the pieces, and prints where to walk.
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
// A part carries at most one interface linked belt, so a part serving an input
// on one side and an output on another is what 2.1's collision validator now
// refuses. The consequence for a rig is geometry -- a 4-in/4-out balancer is
// EIGHT parts, a west column carrying the inputs and an east column carrying
// the outputs -- and it is why two of these bands are redesigns rather than
// re-lays: under this rule a player's belt can only change a balancer's port
// count by landing on a part that has no belt yet. See agents/single-edge.md.
//
// It stages and ASSERTS NOTHING, deliberately: the assertions are the player's
// eyes and the [BBB] log lines the README says to grep for. What IS asserted,
// headlessly, is that everything below is LEGAL -- test/assert-interactive.py
// over a --create of this world requires no failed placement, no compile error
// and no refusal at all, because the gestures are what create the refusals and
// the staging must not. `test/run.sh iact` is that check.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

const (
	part   = "bbb-balancer-part"
	belt   = "express-transport-belt"
	loader = protos.IactLoader
)

var out = harness.Line{Tag: "[BBB-INTERACTIVE] "}

// The four cardinals, read from the running game at load rather than written
// down: a define's number is Factorio's own and is not stable across versions.
// They are package variables because harness.Piece.Dir is a POINTER -- north is
// 0, and "facing north" is not the same statement as "no direction given".
var dirN, dirE, dirS, dirW uint32

// Everything sits in this box, east of spawn: the gesture bands in a column
// around x = 20 and the demo scenes in a second column around x = 56, far
// enough apart that no scene's belts are inside a gesture rig's neighbour gate.
const (
	x0, x1, y0, y1 = 8, 76, -34, 104
	col            = 20 // the gesture column
	dcol           = 56 // the demo column
)

// One tile along a cardinal direction.
func step(dir uint32) (int, int) {
	switch dir {
	case dirN:
		return 0, -1
	case dirS:
		return 0, 1
	case dirE:
		return 1, 0
	}
	return -1, 0
}

// EVERY PLACEMENT GOES THROUGH HERE so that a rig which failed to land says so
// in the log rather than only in the player's confusion. The headless staging
// check fails a run on one of these lines.
//
// NOT harness.Place, AND THAT IS THE ONE THING TO KNOW ABOUT THIS FILE'S ERROR
// HANDLING. Place is Fatal -- it writes `[BBB-OBS] error:`, which test/run.sh
// refuses to see -- and that is right for a suite, where a rig that did not land
// makes every number after it a measurement of a different world. Here the gate
// is test/assert-interactive.py, which reads this line by name and reports which
// piece missed which tile. So the placement is soft and the report is ours.
func put(s fkapi.LuaSurface, p harness.Piece) {
	p.Raise = true
	if _, ok := harness.PlaceSoft(s, p); !ok {
		placeFailed(p.Name, p.X, p.Y)
	}
}

// Containers are placed without raise_built: nothing of this mod's subscribes
// to them (the entity filter admits belt-connectables and three named
// prototypes), so raising for them would be a dispatch that decides nothing.
func box(s fkapi.LuaSurface, name string, x, y int) (fkapi.Object, bool) {
	o, ok := harness.PlaceSoft(s, harness.Piece{Name: name, X: x, Y: y})
	if !ok {
		placeFailed(name, x, y)
	}
	return o, ok
}

func placeFailed(name string, x, y int) {
	out.Open("could not place ").S(name).S(" at (").I(int64(x)).S(",").
		I(int64(y)).S(")").End()
}

func prepGround(s fkapi.LuaSurface) {
	// The capture conditions, staged rather than remembered: this surface exists
	// to be RECORDED as smooth looping GIFs, and anything that changes pixels
	// between the loop's first frame and its last breaks the seam. Three things
	// do on a stock surface -- drifting cloud shadows, the day/night lighting
	// sweep, and the daytime clock behind it -- and the original portal GIFs had
	// them turned off by hand at the console. The uniform grass below is the
	// fourth, for the same reason.
	//
	// EACH ONE IS CHECKED WHERE THE LUA IGNORED THE RETURN, which is the
	// harness's standing treatment of the same calls (Flat.Make does exactly
	// this for always_day). A capture condition that silently failed to stage
	// would be discovered by a ruined recording.
	if err := s.SetShowClouds(false); err != nil {
		harness.Fatal("show_clouds on nauvis", err.Error())
	}
	if err := s.SetAlwaysDay(true); err != nil {
		harness.Fatal("always_day on nauvis", err.Error())
	}
	if err := s.SetFreezeDaytime(true); err != nil {
		harness.Fatal("freeze_daytime on nauvis", err.Error())
	}
	if err := s.SetDaytime(0); err != nil { // noon, and frozen there
		harness.Fatal("daytime on nauvis", err.Error())
	}
	out.Open("surface prepped: clouds off, daytime frozen at noon").End()

	radius := uint32(4)
	center := fkapi.MapPosition{X: float64(x0+x1) / 2, Y: float64(y0+y1) / 2}
	if err := s.RequestToGenerateChunks(center, &radius); err != nil {
		harness.Fatal("request_to_generate_chunks on nauvis", err.Error())
	}
	if err := s.ForceGenerateChunkRequests(); err != nil {
		harness.Fatal("force_generate_chunk_requests on nauvis", err.Error())
	}

	area := harness.Box(x0-4, y0-4, x1+4, y1+4)
	for _, e := range harness.EntitiesIn(s, area, "") {
		// A character is spared exactly as the estate's Lua spares one: a
		// headless run has none, and a graphical one puts the player here.
		if harness.EntityTypeIs(e, "character") {
			continue
		}
		harness.Destroy(e, false)
	}
	s.DestroyDecoratives(fkapi.LuaSurfaceDestroyDecorativesArgs{Area: &area})
	harness.PaveBox(s, x0-4, y0-4, x1+4, y1+4, "grass-1")
}

// A chest and a loader pushing a full belt along `dir`; the belt itself is the
// caller's. Infinity chests, because the player should never see these run dry.
func source(s fkapi.LuaSurface, x, y int, dir uint32) {
	if c, ok := box(s, "infinity-chest", x, y); ok {
		harness.InfinityFilter(c, "iron-plate", "", 1000)
	}
	ox, oy := step(dir)
	put(s, harness.Piece{Name: loader, X: x + ox, Y: y + oy, Dir: &dir, Type: "output"})
}

// A type="input" loader's container side is the tile its arrow points into, so
// the chest goes ONE STEP ALONG dir from the loader -- in every direction, not
// only east. The first cut of this rig offset only x, which parked the three
// north/south outputs' chests beside their loaders where nothing could reach
// them; a player's screenshot found it.
func sink(s fkapi.LuaSurface, x, y int, dir uint32) {
	put(s, harness.Piece{Name: loader, X: x, Y: y, Dir: &dir, Type: "input"})
	ox, oy := step(dir)
	box(s, "steel-chest", x+ox, y+oy)
}

// feedTo(s, x, y, dir, length): a source pushing along `dir` down a run of
// `length` belts that ENDS at (x, y). (x, y) is the tile touching the part, so
// the part it feeds is one step further along `dir`.
//
// The estate's Lua defaulted `length` to 2 and passed 3 once; Go has no default
// arguments, so every call site names it.
func feedTo(s fkapi.LuaSurface, x, y int, dir uint32, length int) {
	dx, dy := step(dir)
	fx, fy := x-dx*(length-1), y-dy*(length-1)
	source(s, fx-2*dx, fy-2*dy, dir)
	for i := 0; i < length; i++ {
		put(s, harness.Piece{Name: belt, X: fx + dx*i, Y: fy + dy*i, Dir: &dir})
	}
}

// drainFrom(s, x, y, dir, length): `length` belts STARTING at (x, y) and running
// along `dir` into a sink. (x, y) is the tile touching the part, so the part it
// drains is one step BACK along `dir`.
func drainFrom(s fkapi.LuaSurface, x, y int, dir uint32, length int) {
	dx, dy := step(dir)
	for i := 0; i < length; i++ {
		put(s, harness.Piece{Name: belt, X: x + dx*i, Y: y + dy*i, Dir: &dir})
	}
	sink(s, x+dx*length, y+dy*length, dir)
}

// ---------------------------------------------------------------------------
// The gesture bands. Each is one gesture from the README, staged so the gesture
// is the only thing left to do.
// ---------------------------------------------------------------------------

// A: THE MINER'S POCKET. A saturated dead-ended 4 -> 4, which under the rule is
// EIGHT parts: four west parts carrying the inputs, four east parts carrying the
// outputs. The gesture is mining it part by part and watching the items land in
// the inventory at EVERY step, not only the last (the 2026-08-02 field report).
// Eight parts rather than four means eight steps, which is more of the same
// evidence rather than a different check.
func bandPocket(s fkapi.LuaSurface) {
	b := -24
	for i := 0; i < 4; i++ {
		put(s, harness.Piece{Name: part, X: col, Y: b + i})
		put(s, harness.Piece{Name: part, X: col + 1, Y: b + i})
		feedTo(s, col-1, b+i, dirE, 2)
		// No sink: the outputs dead-end, so the network fills and stays full.
		put(s, harness.Piece{Name: belt, X: col + 2, Y: b + i, Dir: &dirE})
		put(s, harness.Piece{Name: belt, X: col + 3, Y: b + i, Dir: &dirE})
	}
}

// B: THE BELT AT THE EDGE, and it is a REDESIGN rather than a re-lay. The old
// gesture was "lay a belt on a free face of a 2-part balancer", and under the
// one-belt-per-part rule there is no such face: every part of a working
// balancer already has its belt. So the rig carries an ATTACHED EDGELESS PART
// -- a fifth part hanging off the 2x2 block with nothing against it -- and the
// belt goes there. That takes P from 2 to 4, and mining it takes P back to 2,
// which is the same power-of-two boundary crossing the `edge` suite's `bmin`
// leg pins: the machine halves, the reinsertion overflows, and the overflow is
// what must reach the miner rather than the floor.
//
// The same rig carries the SINGLE-EDGE refusal, because it is the only bound
// that can be reached here without also crossing the port limit: a belt laid
// against a part that already has one is refused, and the balancer keeps
// running.
func bandEdge(s fkapi.LuaSurface) {
	b := -12
	for i := 0; i < 2; i++ {
		put(s, harness.Piece{Name: part, X: col, Y: b + i})
		put(s, harness.Piece{Name: part, X: col + 1, Y: b + i})
		feedTo(s, col-1, b+i, dirE, 2)
		put(s, harness.Piece{Name: belt, X: col + 2, Y: b + i, Dir: &dirE})
		put(s, harness.Piece{Name: belt, X: col + 3, Y: b + i, Dir: &dirE})
	}
	// The edgeless fifth part. Its west and east neighbours are empty and the
	// input belts one row up are DIAGONAL from it, so nothing touches it until
	// the player's belt does.
	put(s, harness.Piece{Name: part, X: col, Y: b + 2})
}

// C: THE SIXTY-FIFTH BELT. Sixty-four input parts in a 2x32 block, one part
// below them carrying the single output, and one EDGELESS part above them
// waiting for the player's belt -- so P = 64, the limit exactly, and the
// sixty-fifth belt has somewhere legal to land. Without that spare part the
// gesture could only ask an occupied part for a second belt, which is the other
// bound and band B's job.
func bandLimit(s fkapi.LuaSurface) {
	b := 0
	put(s, harness.Piece{Name: part, X: col, Y: b - 1}) // the edgeless part the 65th belt lands on
	for i := 0; i < 32; i++ {
		put(s, harness.Piece{Name: part, X: col, Y: b + i})
		put(s, harness.Piece{Name: part, X: col + 1, Y: b + i})
		put(s, harness.Piece{Name: belt, X: col - 1, Y: b + i, Dir: &dirE}) // west inputs
		put(s, harness.Piece{Name: belt, X: col + 2, Y: b + i, Dir: &dirW}) // east inputs
	}
	// Feed eight of the sixty-four so items visibly flow: everything funnels to
	// the one output, which therefore runs saturated.
	for _, i := range []int{4, 12, 20, 28} {
		source(s, col-4, b+i, dirE)
		put(s, harness.Piece{Name: belt, X: col - 2, Y: b + i, Dir: &dirE})
		source(s, col+5, b+i, dirW)
		put(s, harness.Piece{Name: belt, X: col + 3, Y: b + i, Dir: &dirW})
	}
	// The one output, on a part of its own below the block.
	put(s, harness.Piece{Name: part, X: col, Y: b + 32})
	drainFrom(s, col, b+33, dirS, 2)
}

// D: THE BRIDGE. Two balancers of thirty-two inputs and one output each -- a
// 2x16 block plus an output part apiece -- with a one-tile gap between them and
// ONE belt already standing against the gap tile. A part in that gap merges
// them into a machine wanting 65 inputs, which is over the limit; the gap tile
// itself carries a single belt, so the merge fails the port bound rather than
// the one-belt-per-part bound and this stays the over-limit merge gesture.
func bandBridge(s fkapi.LuaSurface) {
	block := func(top int) {
		for i := 0; i < 16; i++ {
			put(s, harness.Piece{Name: part, X: col, Y: top + i})
			put(s, harness.Piece{Name: part, X: col + 1, Y: top + i})
			put(s, harness.Piece{Name: belt, X: col - 1, Y: top + i, Dir: &dirE})
			put(s, harness.Piece{Name: belt, X: col + 2, Y: top + i, Dir: &dirW})
		}
		for _, i := range []int{4, 11} {
			source(s, col-4, top+i, dirE)
			put(s, harness.Piece{Name: belt, X: col - 2, Y: top + i, Dir: &dirE})
		}
	}

	block(45) // block A: parts y 45..60
	block(62) // block B: parts y 62..77, gap at (col, 61)

	// Each block's one output, on its own part, pointing away from the gap.
	put(s, harness.Piece{Name: part, X: col, Y: 44})
	drainFrom(s, col, 43, dirN, 2)
	put(s, harness.Piece{Name: part, X: col, Y: 78})
	drainFrom(s, col, 79, dirS, 2)

	// The gap tile's belt: inert today (it is diagonal from every part), one more
	// input the moment a part lands on (col, 61). One belt and not two, because
	// two on that tile would be the other refusal.
	put(s, harness.Piece{Name: belt, X: col - 1, Y: 61, Dir: &dirE})
}

// E: FAST REPLACE, both directions.
//
// The forward half is a belt line that ENDS one tile below a running 2 -> 2, and
// that ending is the rule's doing: a part dropped into the MIDDLE of a line
// takes the belt behind it as an input and the belt ahead of it as an output,
// which is two belts on one part and is refused. Replacing the line's last tile
// gives the new part one input, and the balancer becomes three in and two out.
//
// The reverse half is a five-part column with edges on its two ends only, so the
// three middle parts carry no interface and a belt can be laid on them. Only the
// MIDDLE one splits cleanly: the new belt is an output of the half above and an
// input of the half below, and each half's other part already carries its own
// belt, so the two parts either side of the new belt must be the ones with
// nothing on them.
func bandFastReplace(s fkapi.LuaSurface) {
	b := 90
	for i := 0; i < 2; i++ {
		put(s, harness.Piece{Name: part, X: col, Y: b + i})
		put(s, harness.Piece{Name: part, X: col + 1, Y: b + i})
		feedTo(s, col-1, b+i, dirE, 2)
		drainFrom(s, col+2, b+i, dirE, 2)
	}
	// The line, ending on (col, b+2). An east-facing belt against a part's SOUTH
	// face is neither `dir` nor `back` from that side, so it is not an edge of the
	// balancer above it -- the `pass` rig's rule, met from the other side.
	source(s, col-5, b+2, dirE)
	for x := col - 3; x <= col; x++ {
		put(s, harness.Piece{Name: belt, X: x, Y: b + 2, Dir: &dirE})
	}

	c := b + 6
	for i := 0; i < 5; i++ {
		put(s, harness.Piece{Name: part, X: col, Y: c + i})
	}
	feedTo(s, col-1, c, dirE, 2)
	drainFrom(s, col+1, c+4, dirE, 2)
}

// ---------------------------------------------------------------------------
// The demo band: the mod portal's capture scenes, versioned here so a capture
// is reproducible instead of living in a save nobody kept.
//
// All five are single-edge, which changed four of them and retired the fifth:
// `single-part-1-to-3-fanout` asked one part to carry four belts and cannot
// exist under this rule, so the cross form -- a plus of five parts whose four
// arms carry one belt each and whose centre carries none -- is the 1 -> 3 read
// now.
// ---------------------------------------------------------------------------

// cross 1 -> 3. The centre part touches four parts and no belt at all, which is
// the shape of the rule in one picture.
func demoCross(s fkapi.LuaSurface) {
	cy := -24
	put(s, harness.Piece{Name: part, X: dcol, Y: cy})
	put(s, harness.Piece{Name: part, X: dcol, Y: cy - 1})
	put(s, harness.Piece{Name: part, X: dcol + 1, Y: cy})
	put(s, harness.Piece{Name: part, X: dcol, Y: cy + 1})
	put(s, harness.Piece{Name: part, X: dcol - 1, Y: cy})
	feedTo(s, dcol-2, cy, dirE, 2)
	drainFrom(s, dcol, cy-2, dirN, 2)
	drainFrom(s, dcol+2, cy, dirE, 2)
	drainFrom(s, dcol, cy+2, dirS, 2)
}

// compact-column 8 -> 8: sixteen parts in a 2x8 block, the smallest 8 -> 8 the
// rule allows.
func demoCompact(s fkapi.LuaSurface) {
	b := -10
	for i := 0; i < 8; i++ {
		put(s, harness.Piece{Name: part, X: dcol, Y: b + i})
		put(s, harness.Piece{Name: part, X: dcol + 1, Y: b + i})
		feedTo(s, dcol-1, b+i, dirE, 2)
		drainFrom(s, dcol+2, b+i, dirE, 2)
	}
}

// The C: a ten-part spine with two arms, eight inputs down the spine's west
// face and the outputs off the arms' free faces. `toparm` is how many parts the
// top arm has, which is the only difference between the 8 -> 8 and the 8 -> 9:
// four gives eight outputs, five gives nine and a P = 16 butterfly with
// loopbacks. The spine's two corner parts carry nothing -- the arms turn there.
func demoCShape(s fkapi.LuaSurface, b, toparm int) {
	for i := 0; i <= 9; i++ {
		put(s, harness.Piece{Name: part, X: dcol, Y: b + i})
	}
	for x := dcol + 1; x <= dcol+toparm; x++ {
		put(s, harness.Piece{Name: part, X: x, Y: b})
	}
	for x := dcol + 1; x <= dcol+4; x++ {
		put(s, harness.Piece{Name: part, X: x, Y: b + 9})
	}
	for i := 1; i <= 8; i++ {
		feedTo(s, dcol-1, b+i, dirE, 2)
	}
	for x := dcol + 1; x <= dcol+toparm; x++ {
		drainFrom(s, x, b-1, dirN, 2)
	}
	for x := dcol + 1; x <= dcol+4; x++ {
		drainFrom(s, x, b+10, dirS, 2)
	}
}

// long-run 8 -> 8: sixteen parts in a single row, taking their inputs from the
// north and giving their outputs to the south, alternately. Belt orientation
// alone decides which is which, and here that is the whole picture.
func demoLongRun(s fkapi.LuaSurface) {
	y := 60
	for i := 0; i < 16; i++ {
		put(s, harness.Piece{Name: part, X: dcol + i, Y: y})
		if i%2 == 0 {
			feedTo(s, dcol+i, y-1, dirS, 3)
		} else {
			drainFrom(s, dcol+i, y+1, dirS, 2)
		}
	}
}

// ---------------------------------------------------------------------------

// audit places the shipped `bbb-audit` marker, which asks the mod under test to
// drain its deferred queue and re-classify the world synchronously.
//
// harness.Audit and NOT put, and that is not an oversight. The audit marker is a
// shipped prototype whose whole contract is that it destroys itself inside the
// dispatch its own placement raises, so `create_entity` hands back nil for it
// every time -- which put would report as a rig that failed to land.
func audit(s fkapi.LuaSurface, tag string) {
	harness.Audit(s, col-10, -30)
	out.Open("audited ").S(tag).End()
}

func init() {
	// The player block below is the only reason this observer has an event
	// handler at all. The id is a literal constant at the call site, which is
	// not optional: FkLua prunes 218 event descriptors down to the ones a guest
	// names, and an id it cannot prove ships all of them.
	fkapi.Subscribe(fkapi.EventOnPlayerCreated)
	dirN = fkapi.DefinesDirectionNorth()
	dirE = fkapi.DefinesDirectionEast()
	dirS = fkapi.DefinesDirectionSouth()
	dirW = fkapi.DefinesDirectionWest()
}

//go:wasmexport fk_on_init
func onInit() {
	// No crash site and no intro: the rigs are the scenario.
	//
	// ONE STATUS CHECK WHERE THE LUA HAD TWO GUARDS. It wrote
	// `if remote.interfaces["freeplay"] then pcall(remote.call, ...) end` --
	// an existence test and then a protected call, because remote.call RAISES
	// on a missing interface or method. fkapi.RemoteCall returns
	// StatusCallFailed instead, deliberately: another mod not being installed
	// is an ordinary thing for a guest to have an opinion about. So the two
	// guards collapse into the return value, and both spellings mean the same
	// thing here -- do it if freeplay is there, carry on if it is not. A
	// headless --create loads no scenario script, so it is not there and both
	// calls are no-ops; a player's freeplay world is where they land.
	fkapi.RemoteCall("freeplay", "set_disable_crashsite", fkapi.OfBool(true))
	fkapi.RemoteCall("freeplay", "set_skip_intro", fkapi.OfBool(true))

	s := harness.Surface("nauvis")
	prepGround(s)

	bandPocket(s)
	bandEdge(s)
	bandLimit(s)
	bandBridge(s)
	bandFastReplace(s)

	demoCross(s)
	demoCompact(s)
	demoCShape(s, 6, 4)  // c-shape 8 -> 8
	demoCShape(s, 30, 5) // c-shape 8 -> 9
	demoLongRun(s)

	// TWO MARKERS, AND THE SECOND IS THE ONE WORTH READING. The audit reports the
	// registry as it stands when its own dispatch begins, and the compiling is
	// what that dispatch then does -- so the first marker reports every cluster
	// unbuilt and the second, placed after the first has drained the queue,
	// reports them built. A --create never reaches a tick, so there is no other
	// way to see the compiled state from here.
	audit(s, "staged")
	audit(s, "compiled")

	out.Open("gestures staged: pocket y-24, edge y-12, limit y0 " +
		"(spare part at (20,-1)), bridge y45 (gap at (20,61)), fast replace y90").End()
	out.Open("demo scenes staged at x56: cross y-24, compact y-10, " +
		"c-shape 8->8 y6, c-shape 8->9 y30, long run y60").End()
}

// chartTag is one map marker: where it goes and what it says.
type chartTag struct {
	x, y int
	text string
}

var tags = []chartTag{
	{col, -23, "A: mine me part by part"},
	{col, -10, "B: belt on (20,-9), mine it, then try (20,-13)"},
	{col, 16, "C: 65th belt at (20,-2), facing south"},
	{col, 61, "D: one part in this gap"},
	{col, 92, "E: part onto (20,92); belt onto (20,98)"},
	{dcol, -24, "demo: cross 1-to-3"},
	{dcol, -10, "demo: compact column 8-to-8"},
	{dcol, 10, "demo: c-shape 8-to-8"},
	{dcol, 34, "demo: c-shape 8-to-9"},
	{dcol + 7, 60, "demo: long run 8-to-8"},
}

// The lines the player reads in chat when they spawn. They are the checklist's
// own summary, kept beside it: test/interactive/README.md is the long form.
var greeting = []string{
	"[BBB] Five gesture rigs east of you and five demo scenes further east.",
	"[BBB] A: y=-24 mine the balancer part by part.",
	"[BBB] B: y=-12 lay a south-facing belt at (20,-9), mine it, then try (20,-13).",
	"[BBB] C: y=0 lay a south-facing belt at (20,-2), against the spare part.",
	"[BBB] D: y=61 place a balancer part in the one-tile gap between the two big ones.",
	"[BBB] E: y=90 drop a part onto (20,92); then lay a belt on (20,98).",
	"[BBB] Expected outcomes and the log lines to grep: test/interactive/README.md",
}

// welcome teleports the arriving player to the rigs, hands them the pieces every
// gesture needs, tags the map and prints where to walk.
//
// NOTHING IN HERE CAN RUN HEADLESSLY, and that is the whole reason this mod
// exists: a `--create` has no players, so `on_player_created` never fires and
// the `iact` suite sees none of it. It is the same wall the miner's pocket's
// trigger is behind. What the suite DOES check is everything fk_on_init staged.
func welcome(p fkapi.LuaPlayer) {
	// A RAW MAP POSITION, not a tile centre: the estate's Lua teleported to
	// `{col - 8, -16}` and a player is not a 1x1 entity on a tile.
	//
	// The surface crosses as a HANDLE where the Lua passed the name. SurfaceID
	// is `string or LuaSurface or uint` and the generated parameter is an
	// Object, which is the same deviation harness.QualityProto records for
	// QualityID and the shipped guest already takes for the same union.
	nauvis := harness.Surface("nauvis")
	p.Teleport(fkapi.MapPosition{X: col - 8, Y: -16}, &nauvis.Object, nil, nil, nil)

	// The pieces every gesture needs, so nothing has to be crafted.
	p.Insert(fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(belt)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(50)}))
	p.Insert(fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(part)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(10)}))

	fo, err := p.Force()
	if err != nil {
		return
	}
	force := fkapi.LuaForce{Object: fo}
	so, err := p.Surface()
	if err != nil {
		return
	}
	force.Chart(so, harness.Box(x0-32, y0-32, x1+32, y1+32))
	for _, t := range tags {
		text := t.text
		// The error is ignored where the Lua wrapped this in a pcall, and for
		// the same reason: a chart tag that would not place is a missing label
		// on a rig that is standing there anyway.
		force.AddChartTag(so, fkapi.ChartTagSpec{
			Position: harness.Center(t.x, t.y),
			Text:     &text,
		})
	}
	for _, line := range greeting {
		p.Print(fkapi.OfString(line), nil)
	}
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnPlayerCreated {
		return
	}
	e := fkapi.ReadOnPlayerCreated(ptr)
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(e.PlayerIndex)))
	if err != nil || o == nil {
		return
	}
	welcome(fkapi.LuaPlayer{Object: *o})
}

func main() {}
