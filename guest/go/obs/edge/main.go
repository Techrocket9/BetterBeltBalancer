// Command bbb-edge-test drives the edits that arrive while a network is FULL AND
// MOVING.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-edge-test/control.lua` was, rig for rig, tick for tick and log
// line for log line, with its little data stage compiled too (obs/edgedata). See
// agents/estate-port.md.
//
// The M2 suite proves item conservation across ONE recompile of a saturated 2x2.
// This suite asks the questions that only a long multiplayer game asks: does it
// hold on the hundredth cycle, does it hold when two full networks are torn down
// in one flush, does it hold when a part is placed and removed inside the same
// tick, and does it hold when two forces become one.
//
// # The measurement is global and it is atomic
//
// `countAll` totals every item on the visible surface AND on the hidden surface
// -- on the ground, on a belt, in a splitter's transport lines, in a chest -- so
// nothing this mod does can move an item out of the count. Every source is a
// FINITE steel chest, so the total is a conserved quantity and any fall in it is
// a real loss. And every count is taken immediately after a `bbb-audit` marker in
// the SAME tick: the marker is the shipped synchronous "drain the queue and
// compile now" trigger, and `create_entity{raise_built=true}` dispatches it
// before it returns, so the count and the teardown it is about are one sample.
//
// # The rigs, one per 30-row band
//
//	chn   a 2-part balancer whose OUTPUTS ARE DEAD-ENDED, so its hidden network
//	      fills completely and stays that way, grown by a third part and shrunk
//	      again ONE HUNDRED TIMES with the count taken every cycle. Blocking the
//	      outputs is what makes this a hundred teardowns of a FULL network rather
//	      than of an empty one; the M2 suite proves the single case and this one
//	      proves it does not drift.
//	same  a part placed and removed inside one tick (the paste-then-undo shape),
//	      and then a part placed on the tick a deferred flush from the previous
//	      tick is pending.
//	mrg   two saturated 2-part balancers with a one-tile gap. A part in the gap
//	      BRIDGES them: two full networks come down in one flush and one comes up.
//	      Removing it again is the undo, and the split path under load.
//	rot   an edge belt turned around mid-flow, twice: once silently (which raises
//	      nothing at all and is what an undone rotation does to the world) and
//	      once through the event path.
//	frc   two forces' balancers touching. Same-tick edits interleaved between
//	      them, and then `game.merge_forces` while both networks are full.
//	det1  four clusters of one, two, three and four parts, pasted in ONE tick
//	det2  in a deliberately scrambled order. The deferred flush must compile them
//	      in the order their first part arrived, and det2 must produce exactly the
//	      sequence det1 did -- which is the determinism a lockstep game needs from
//	      the queue.
//	aout  a SATURATED 4x4, four in and four out, with a fifth OUTPUT belt (and its
//	      sink) added while it runs. This is the field report: an edge edit on an
//	      operating balancer. Nothing may reach the ground, and the 4->5 network it
//	      becomes has to balance over the window after it.
//	ain   the same shape, with a fifth INPUT belt added while it runs.
//	shrk  the same shape, with one OUTPUT BELT REMOVED while it runs. Four outputs
//	      to three LOOKS like the case where reinsertion runs out of room and it is
//	      not: P = next_pow2(max(N, M)) is 4 either side of that edit, so the
//	      butterfly is the same size and everything fits. Reading it as evidence
//	      about shrinks is what let the second field report through; `bmin` is the
//	      one that actually shrinks.
//	bmin  THE SECOND FIELD REPORT: a saturated 2-part balancer, two in and two out
//	      and DEAD-ENDED, with a third OUTPUT BELT added while it runs and then
//	      MINED AGAIN. Two outputs to three and back crosses the boundary: P goes
//	      2 -> 4 -> 2, the machine doubles and then halves, and the reinsertion
//	      into the half genuinely overflows. What the overflow is FOR is the
//	      miner's pocket, and this leg is what says the quantity is real.
//	lim   THE BIGGEST BALANCER THIS MOD BUILDS, one belt short of refusing: a 2x32
//	      block with a belt on each of its sixty-four parts, which is P = 64 =
//	      plan.MaxPorts exactly, plus one output part and one spare. A sixty-fifth
//	      input belt is laid on it while it runs and then mined off again. The only
//	      leg here that is not about items: what it asserts is that the refusal
//	      happens BEFORE the teardown, so a working network survives an edit the
//	      mod cannot honour instead of being demolished for nothing.
//	brdg  THE OTHER SHAPE OF THE SAME REFUSAL, and the one `lim`'s fix could not
//	      reach. Two WORKING balancers of thirty-three parts each with a one-tile
//	      gap that already carries one input belt. A merge's teardowns belong to
//	      `AddPart`, not to compile(), so they are queued before the flush even
//	      starts.
//	frepa FAST REPLACE, forward: a PART fast-replaced onto a belt line that ENDS
//	      on the target tile, while everything runs.
//	frepb FAST REPLACE, reverse, and the half that needs guest code: a BELT laid on
//	      a part, for which the engine raises NO EVENT AT ALL.
//	ntch  THE FIELD REPORT'S SHAPE: a 2x2 balancer with one corner missing,
//	      saturated and flowing. The missing corner is the only tile in this save
//	      enclosed by parts and not one, and it is where a visual artifact would
//	      land.
//
// AND WHERE THE COMPILER PUT ITS VISIBLE ENTITIES. `probePlacement` enumerates
// every entity of one of the mod's four hidden prototypes standing on the VISIBLE
// surface and asks one question of each: is there a registered balancer part on
// that exact tile? The compiler's contract is that the only thing it ever puts
// where a player can see it is an edge interface, and an edge interface sits on
// the cluster's own tile -- so the answer must be yes for every one of them, on
// every sample. It is a structural guarantee rather than a pixel one.
//
// GROUND ITEMS ARE COUNTED SEPARATELY FROM THE TOTAL, and that is the point of
// the 2026-08-02 pass. Conservation was always exact; what was wrong was the
// PLACEMENT. Every count line carries `ground=`, and for a recompile it must be
// zero.
//
// ASSERTS NOTHING. test/assert-edge.py decides.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

var out = harness.Line{Tag: "[BBB-EDGE] "}

const (
	part   = "bbb-balancer-part"
	belt   = "express-transport-belt"
	loader = protos.EdgeLoader

	surfName   = "bbb-edge"
	hidden     = "bbb-hidden"
	otherForce = "bbb-other"

	pitch = 30
	stock = 6000
)

// The band bases. Every one of them is a constant of the save's geometry.
const (
	chn  = 0
	same = pitch
	mrg  = 2 * pitch
	rot  = 3 * pitch
	frc  = 4 * pitch
	det1 = 5 * pitch
	det2 = 5*pitch + 15

	// The three recompile-under-load rigs. Each is four in and four out over
	// EIGHT parts, saturated, plus a FIFTH ROW OF PARTS CARRYING NOTHING --
	// which is the one thing the one-belt rule forced on this suite. Two of the
	// three exist to take a fifth belt while they run, and under the rule every
	// part of a working balancer already has its belt, so a belt laid on any of
	// them would be REFUSED rather than compiled.
	aout = 6 * pitch
	ain  = 7 * pitch
	shrk = 8 * pitch

	// The field report's own shape, in its own band because it is the only rig
	// with a hole in it.
	ntch = 9 * pitch

	// The port-boundary rig: a 2->2 over four parts plus an attached EDGELESS
	// fifth, and the only rig whose edge count crosses a power of two in both
	// directions.
	bmin = 10 * pitch

	// THE PORT LIMIT. A 2x32 block carrying SIXTY-FOUR input belts -- one on
	// each part, which is the rule -- plus one part below it carrying the single
	// output and one spare part above it carrying nothing.
	//
	// THE SPARE PART IS FORCED RATHER THAN FREE. Every one of the sixty-five
	// parts that carry a belt already has its one belt, so a sixty-fifth INPUT
	// has nowhere legal to land: laid against any of them it would reach the
	// one-belt-per-part bound instead, which is a different refusal in a
	// different suite.
	lim     = 11 * pitch
	limRows = 32
	limFed  = 3

	// THE MERGE THE MOD CANNOT HONOUR. Two 2x16 blocks carrying thirty-two input
	// belts each with an output part of their own, so each half is thirty-three
	// parts at P = 32 and a real network in its own right. ONE TILE between them,
	// with ONE input belt standing beside that tile from t=0.
	brdg     = lim + limRows + 12
	brdgHalf = 16
	brdgFed  = 3

	// FAST REPLACE, two rigs in one band, and the only rigs in this suite that
	// are BUILT MID-RUN. Their SOURCE CHESTS are created in on_init all the same,
	// because `countAll` is a conserved quantity and inserting twenty-four
	// thousand items into it mid-run would read as this mod minting matter.
	frep  = brdg + 2*brdgHalf + 12
	frepB = frep + 8
	rows  = frepB + 12
)

// ours is everything the compiler is allowed to put on the visible surface. All
// four are named rather than just the linked belt: the assertion is about the
// CONTRACT, so a hidden splitter appearing out there has to fail rather than go
// uncounted.
var ours = []string{"bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter"}

// The hidden surface's slot grid is 32x72 per slot, 64 columns. Twenty columns
// and four rows is eighty slots, far more than this save can use and far less
// than a query over the whole address space.
func hiddenArea() fkapi.BoundingBox { return harness.Box(0, 0, 640, 288) }
func visArea() fkapi.BoundingBox    { return harness.Box(-24, -18, 28, rows+18) }

// The determinism rigs: {x of the west part column, rows}, and the order their
// first parts are placed. Each is a 2-wide block with its input belt at x-1 and
// its output at x+2, and the columns are five apart so that no rig's output belt
// is adjacent to the next rig's input.
var detShapes = [4][2]int{{8, 1}, {13, 2}, {18, 3}, {23, 4}}
var detOrder = [4]int{3, 1, 4, 2}

var dirN, dirE, dirS, dirW uint32

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	dirN = fkapi.DefinesDirectionNorth()
	dirE = fkapi.DefinesDirectionEast()
	dirS = fkapi.DefinesDirectionSouth()
	dirW = fkapi.DefinesDirectionWest()
}

// ---------------------------------------------------------------------------
// world helpers
// ---------------------------------------------------------------------------

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, force string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Force: force, Raise: true,
	})
}

func putTyped(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ, force string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Force: force, Raise: true,
	})
}

func putSoft(s fkapi.LuaSurface, name string, x, y int, dir *uint32, force string) bool {
	_, ok := harness.PlaceSoft(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Force: force, Raise: true,
	})
	return ok
}

func makeSurface() fkapi.LuaSurface {
	return harness.Flat{
		Name:        surfName,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		ChunkRadius: uint32((rows+31)/32) + 3,
		X0:          -24,
		Y0:          -18,
		X1:          28,
		Y1:          rows + 18,
		Tile:        "grass-1",
	}.Make()
}

func source(s fkapi.LuaSurface, y int, force string) {
	c := harness.Place(s, harness.Piece{Name: "steel-chest", X: -6, Y: y, Force: force})
	harness.InsertInto(c, "iron-plate", stock)
	putTyped(s, loader, -5, y, &dirE, "output", force)
}

func sink(s fkapi.LuaSurface, y int, force string) harness.XY {
	putTyped(s, loader, 5, y, &dirE, "input", force)
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 6, Y: y, Force: force})
	return harness.XY{X: 6, Y: y}
}

// ONE BELT PER PART, so a row is TWO parts: the west one carries the input and
// the east one the output.
//
//	x=-6 source chest   -5 loader   -4..-1 belts   0 WEST PART   1 EAST PART
//	x=2..4 belts        5 sink loader              6 chest
func feed(s fkapi.LuaSurface, y int, force string) harness.XY {
	feedIn(s, y, force)
	return drainOut(s, y, force)
}

// feedIn is the input side alone, for a row whose output is somewhere other than
// due east.
func feedIn(s fkapi.LuaSurface, y int, force string) {
	source(s, y, force)
	for x := -4; x <= -1; x++ {
		put(s, belt, x, y, &dirE, force)
	}
}

// ... and drainOut is the output side alone.
func drainOut(s fkapi.LuaSurface, y int, force string) harness.XY {
	for x := 2; x <= 4; x++ {
		put(s, belt, x, y, &dirE, force)
	}
	return sink(s, y, force)
}

// ---------------------------------------------------------------------------
// counting
//
// Everything, on both surfaces. An item this mod can lose is an item that left
// this total, and there is nowhere else for one to be.
// ---------------------------------------------------------------------------

// countArea returns TWO numbers: everything, and the part of it that is lying on
// the ground. The second is what the recompile policy is about --
// `spill_item_stack` puts an item on a belt when there is one under it and on the
// floor otherwise, so a spill that lands on the output belts is invisible to the
// total and very visible to a player.
func countArea(name string, area fkapi.BoundingBox) (total, ground int64) {
	s, ok := harness.SurfaceIfAny(name)
	if !ok {
		return 0, 0
	}
	for _, e := range harness.EntitiesIn(s, area, "") {
		if harness.EntityTypeIs(e, "item-entity") {
			if _, n, got := harness.GroundStack(e); got {
				total += n
				ground += n
			}
			continue
		}
		total += harness.TransportLineItems(e)
		for _, item := range harness.ChestContents(e) {
			total += int64(item.Count)
		}
	}
	return total, ground
}

func countAll() (total, ground int64) {
	a, ag := countArea(surfName, visArea())
	b, bg := countArea(hidden, hiddenArea())
	return a + b, ag + bg
}

// auditAndCount is the atomic sample: the marker drains the queue and
// re-classifies inside this very dispatch, so the count that follows describes
// the world the audit just reported on.
func auditAndCount(tag string) {
	out.Open("mark tag=").S(tag).End()
	harness.Audit(surf(), 24, 4)
	total, ground := countAll()
	out.Open("count tag=").S(tag).S(" total=").I(total).S(" ground=").I(ground).End()
}

// ---------------------------------------------------------------------------
// WHERE the compiler put its visible entities
//
// The visual half of the mod rests on one structural fact: the only thing the
// compiler ever creates on a surface a player looks at is an edge interface, and
// an edge interface stands on a tile of the cluster itself -- under a balancer
// part's own sprite, which is one opaque tile. Anything of ours on a bare tile is
// a picture with nothing over it.
//
// The tile is taken with `floor`, which is exact for the 1x1 entities involved
// (both sit at a tile centre) and would round a 2-tile splitter's centre onto one
// of its tiles -- generous in the direction that could only hide a failure, never
// invent one.
// ---------------------------------------------------------------------------

func tileOf(o fkapi.Object) (harness.XY, bool) {
	p, err := (fkapi.LuaEntity{Object: o}).Position()
	if err != nil {
		return harness.XY{}, false
	}
	return harness.XY{X: floorI(p.X), Y: floorI(p.Y)}, true
}

// floorI is math.Floor for a value that is about to become a tile index.
func floorI(v float64) int {
	i := int(v)
	if float64(i) > v {
		i--
	}
	return i
}

func probePlacement(tag string) {
	s := surf()
	parts := make(map[harness.XY]bool)
	nparts := 0
	for _, e := range harness.EntitiesIn(s, visArea(), part) {
		if xy, ok := tileOf(e); ok {
			parts[xy] = true
			nparts++
		}
	}
	names := make([]fkapi.Value, 0, len(ours))
	for _, n := range ours {
		names = append(names, fkapi.OfString(n))
	}
	list := fkapi.OfArray(names...)
	box := visArea()
	found, _ := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &box, Name: &list})
	total, off := 0, 0
	strays := make([]string, 0, 8)
	for _, e := range found {
		total++
		xy, ok := tileOf(e)
		if ok && parts[xy] {
			continue
		}
		off++
		n, _ := (fkapi.LuaEntity{Object: e}).Name()
		strays = append(strays, n+"@"+itoa(xy.X)+","+itoa(xy.Y))
	}
	harness.SortStrings(strays)
	out.Open("place tag=").S(tag).S(" ours=").I(int64(total)).
		S(" onpart=").I(int64(total - off)).S(" offpart=").I(int64(off)).
		S(" parts=").I(int64(nparts)).End()
	for i := 0; i < len(strays) && i < 8; i++ {
		out.Open("stray tag=").S(tag).S(" ").S(strays[i]).End()
	}
}

// itoa is a signed decimal INSIDE a string rather than appended to a line, which
// the stray list needs because it is sorted before it is emitted. `strconv` is a
// package a guest otherwise never links.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
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

// ---------------------------------------------------------------------------
// the output registry
// ---------------------------------------------------------------------------

// outRig is one named entry of the estate's `storage.out`: a rig and the chest
// TILES its outputs drain into.
type outRig struct {
	name   string
	chests []harness.XY
}

var outs []outRig

func outAdd(name string, chests []harness.XY) {
	outs = append(outs, outRig{name: name, chests: chests})
}

func outAppend(name string, xy harness.XY) {
	for i := range outs {
		if outs[i].name == name {
			outs[i].chests = append(outs[i].chests, xy)
			return
		}
	}
}

// chestTotal is a chest's contents, or 0 when there is no chest there.
//
// ZERO AND NOT -1, which is where this differs from harness.ChestCount and why
// it is written out: the estate's `report` opened with `local n = 0` and only
// added to it `if c and c.valid`, so a rig whose chest has gone reports 0. The -1
// convention belongs to the suites that use it to say "there is no sink here".
func chestTotal(s fkapi.LuaSurface, xy harness.XY) int64 {
	if n := harness.ChestCount(s, "steel-chest", xy.X, xy.Y); n > 0 {
		return n
	}
	return 0
}

func report(tag string) {
	order := make([]string, 0, len(outs))
	for i := range outs {
		order = append(order, outs[i].name)
	}
	harness.SortStrings(order)
	s := surf()
	for _, name := range order {
		var chests []harness.XY
		for i := range outs {
			if outs[i].name == name {
				chests = outs[i].chests
			}
		}
		out.Open("t=").S(tag).S(" rig=").S(name).S(" out=[")
		for i, xy := range chests {
			if i > 0 {
				out.S(" ")
			}
			out.I(chestTotal(s, xy))
		}
		out.S("]").End()
	}
}

// ---------------------------------------------------------------------------
// P1: a hundred cycles of add-part / remove-part on a network that is full
// ---------------------------------------------------------------------------

// Sixteen ticks, and the number is chosen rather than convenient: a rebuild
// starts on an EMPTY network and refills it out of the backed-up input belts, so
// the removal has to be late enough that what comes down is full again.
const (
	chnIters  = 100
	chnPeriod = 16
	chnT0     = 300
	chnEnd    = chnT0 + chnIters*chnPeriod
)

func chnStep(phase, i int) {
	s := surf()
	switch phase {
	case 0:
		putSoft(s, part, 0, chn+2, nil, "")
	case 10:
		harness.KillAt(s, 0, chn+2, part, "")
	case 13:
		auditAndCount("chn" + itoa(i))
	}
}

// ---------------------------------------------------------------------------
// the one-off scenarios
// ---------------------------------------------------------------------------

// A part placed and removed inside ONE dispatch chain. The registry gains a node
// and gives it straight back; the flush on the next tick is handed a root whose
// slot has been freed and must drop it rather than compile it.
func pSameTick() {
	s := surf()
	out.Open("sametick begin").End()
	putSoft(s, part, 0, same+2, nil, "")
	harness.KillAt(s, 0, same+2, part, "")
	out.Open("sametick end").End()
}

// A part placed on the tick BEFORE, and another on the tick the deferred flush
// for it lands. Whether this observer's on_tick runs before or after the guest's
// one-shot is engine order and is deterministic either way; what must not happen
// is a network built under a root that has stopped being one.
func pPendingA() {
	putSoft(surf(), part, 0, same+2, nil, "")
	out.Open("pending first").End()
}

func pPendingB() {
	putSoft(surf(), part, 0, same+3, nil, "")
	out.Open("pending second").End()
}

func pPendingUndo() {
	s := surf()
	harness.KillAt(s, 0, same+2, part, "")
	harness.KillAt(s, 0, same+3, part, "")
}

// The bridge: one part in the gap between two SATURATED balancers. Two whole
// networks are torn down in one flush and one is built.
func pMerge() {
	out.Open("merge begin").End()
	putSoft(surf(), part, 0, mrg+2, nil, "")
}

func pUnmerge() {
	out.Open("split begin").End()
	harness.KillAt(surf(), 0, mrg+2, part, "")
}

// An edge belt turned around with `entity.direction = ...`, which raises NOTHING.
// The audit is the documented recovery, and it must SEE the drift before it
// repairs it.
func setBeltDir(x, y int, d uint32) {
	if b, ok := harness.FindAt(surf(), x, y, "", "transport-belt"); ok {
		(fkapi.LuaEntity{Object: b}).SetDirection(d)
	}
}

func pRotSilent() {
	setBeltDir(-1, rot, dirW)
	out.Open("rot silent").End()
}

func pRotRestore() {
	setBeltDir(-1, rot, dirE)
	out.Open("rot restored").End()
}

// The same edge, through the event path: the belt is mined and a belt facing the
// other way is laid on the same tile inside the same tick.
func pRotEvent() {
	s := surf()
	harness.KillAt(s, -1, rot+1, "", "transport-belt")
	putSoft(s, belt, -1, rot+1, &dirW, "")
	out.Open("rot event flipped").End()
}

func pRotEventBack() {
	s := surf()
	harness.KillAt(s, -1, rot+1, "", "transport-belt")
	putSoft(s, belt, -1, rot+1, &dirE, "")
	out.Open("rot event restored").End()
}

// Two forces editing in one tick, alternating. Their parts touch and must stay
// two balancers; each edit must reach only its own.
//
// A WHOLE ROW EACH, because a row is two parts under the one-belt rule: a west
// part taking the new input and an east part giving the new output. The old
// version added one part and put a belt on both sides of it, which is the shape
// the rule forbids.
func pForcesInterleaved() {
	s := surf()
	out.Open("forces interleaved begin").End()
	putSoft(s, part, 0, frc-1, nil, "")
	putSoft(s, part, 0, frc+4, nil, otherForce)
	putSoft(s, part, 1, frc-1, nil, "")
	putSoft(s, part, 1, frc+4, nil, otherForce)
	putSoft(s, belt, -1, frc-1, &dirE, "")
	putSoft(s, belt, -1, frc+4, &dirE, otherForce)
	putSoft(s, belt, 2, frc-1, &dirE, "")
	putSoft(s, belt, 2, frc+4, &dirE, otherForce)
	out.Open("forces interleaved end").End()
}

// And then the two forces become one, while both networks are full. No
// per-entity event is raised for any of it.
//
// What this leg CANNOT reach is the other half of the merge: a player mining a
// source-force part in the same tick, whose claim names a force the merge is
// about to destroy. A headless --create has no player, so player_index is 0 on
// every removal any suite can produce and the claim list is empty in all of them.
// That half is proved by `go test ./carry/` instead, which `make check` runs.
func pForcesMerge() {
	out.Open("forces merge begin").End()
	src, okA := harness.ForceByName(otherForce)
	dst, okB := harness.ForceByName("player")
	if okA && okB {
		if err := fkapi.Game.MergeForces(src, dst); err != nil {
			harness.Fatal("merge_forces", fk.LastError())
		}
	}
	out.Open("forces merge end").End()
}

// Four clusters in one tick, their parts interleaved in a scrambled order. The
// flush must compile them in the order their FIRST part arrived.
func detPaste(base int, tag string) {
	s := surf()
	out.Open("det-begin tag=").S(tag).End()
	for step := 1; step <= 4; step++ {
		for _, k := range detOrder {
			cx, n := detShapes[k-1][0], detShapes[k-1][1]
			if step <= n {
				putSoft(s, part, cx, base+step-1, nil, "")
				putSoft(s, part, cx+1, base+step-1, nil, "")
			}
		}
	}
	for _, k := range detOrder {
		cx, n := detShapes[k-1][0], detShapes[k-1][1]
		for r := 0; r < n; r++ {
			putSoft(s, belt, cx-1, base+r, &dirE, "")
			putSoft(s, belt, cx+2, base+r, &dirE, "")
		}
	}
	out.Open("det-end tag=").S(tag).End()
}

func detFlushed(tag string) { out.Open("det-flushed tag=").S(tag).End() }

func detClear(base int) {
	s := surf()
	for _, shape := range detShapes {
		cx, n := shape[0], shape[1]
		for r := 0; r < n; r++ {
			harness.KillAt(s, cx, base+r, part, "")
			harness.KillAt(s, cx+1, base+r, part, "")
			harness.KillAt(s, cx-1, base+r, "", "transport-belt")
			harness.KillAt(s, cx+2, base+r, "", "transport-belt")
		}
	}
}

// A balancer part built ON THE HIDDEN SURFACE, which is what an area clone or a
// careless script can do and what nothing else in the suite reaches. The guest
// must refuse to register it: a cluster there would put its bounding box inside
// the slot grid, and a teardown spills a network's items beside the CLUSTER --
// which would be a surface no player can ever reach.
func pHiddenPart() {
	h, ok := harness.SurfaceIfAny(hidden)
	if !ok {
		return
	}
	out.Open("hidden-part begin").End()
	harness.PlaceSoft(h, harness.Piece{Name: part, X: 1000, Y: 500, Raise: true})
	e, found := harness.FindAt(h, 1000, 500, part, "")
	out.Open("hidden-part placed=").B(found).End()
	if found {
		harness.Destroy(e, true)
	}
}

// ---------------------------------------------------------------------------
// The three recompile-under-load rigs
// ---------------------------------------------------------------------------

// The fifth OUTPUT, with a loader and a chest to drain it, so the rig after the
// edit is a real 4->5 balancer rather than one with a blocked port. It leaves
// SOUTH off the spare row's west part, which is the only tile of this cluster
// with a free face.
func pAddOutput() {
	s := surf()
	out.Open("add-out begin").End()
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: aout + 7})
	harness.PlaceSoft(s, harness.Piece{
		Name: loader, X: 0, Y: aout + 6, Dir: &dirS, Type: "input", Raise: true,
	})
	putSoft(s, belt, 0, aout+5, &dirS, "")
	outAppend("aout", harness.XY{X: 0, Y: aout + 7})
	out.Open("add-out end").End()
}

// The fifth INPUT: one south-facing belt, and NOTHING FEEDING IT. The edit under
// test is the edge list going 4->5 inputs and the network being rebuilt over
// eight ports; a source chest would put 4,800 new items into a total whose whole
// job is to be conserved, and the classifier does not care whether an input belt
// is carrying anything.
func pAddInput() {
	out.Open("add-in begin").End()
	putSoft(surf(), belt, 0, ain+5, &dirN, "")
	out.Open("add-in end").End()
}

// One OUTPUT BELT mined off a running 4x4. Four in, three out afterwards.
func pRemoveOutput() {
	out.Open("shrink begin").End()
	harness.KillAt(surf(), 2, shrk, "", "transport-belt")
	out.Open("shrink end").End()
}

// THE OTHER HALF OF THE POLICY, in the same suite so the two cannot drift apart:
// every part of the shrk balancer mined, which DISSOLVES the cluster. There is no
// successor network for the drained items to go into and the machine has
// genuinely been removed, so they must come back to the world -- onto the belts
// under them where there is room, and onto the ground where there is not, which
// is what a mined machine does in vanilla.
func pRemoveCluster() {
	s := surf()
	out.Open("remove begin").End()
	for r := 0; r <= 3; r++ {
		harness.KillAt(s, 0, shrk+r, part, "")
		harness.KillAt(s, 1, shrk+r, part, "")
	}
	out.Open("remove end").End()
}

// TAKING A BALANCER APART BY HAND: ONE PART PER TICK, WHICH IS THE FIELD REPORT.
//
// `pRemoveCluster` above mines every part in ONE tick, so it is one teardown of
// one network and one removal. A PLAYER cannot do that: they mine a part, the
// machine recompiles SMALLER, they mine the next one, and so on down to the
// dissolve. Each of those shrinks is a recompile into a network with fewer ports
// and less line, so each one hands back less than it drained.
//
// THE ORDER IS THE SPARE ROW AND THEN ROW BY ROW, WEST PART THEN EAST PART, and
// it is chosen rather than convenient. Every prefix of it leaves a CONNECTED
// cluster, so no step is a split. And every step but the last two leaves a
// machine with at least one input and one output, so it is a real shrink into a
// smaller butterfly.
//
// THE NINTH STEP HAS NO NETWORK AND THAT IS STRUCTURAL. One part can carry one
// belt, so the last part standing has an input or an output and never both --
// which `plan.Build` reads as a legitimate half-built cluster, not an unbuilt one.
var handOrder = func() [][2]int {
	o := [][2]int{{0, 4}, {1, 4}}
	for r := 0; r <= 3; r++ {
		o = append(o, [2]int{0, r}, [2]int{1, r})
	}
	return o
}()

func pHandMine(i int) func() {
	return func() {
		xy := handOrder[i-1]
		out.Open("hand mine ").I(int64(i)).End()
		harness.KillAt(surf(), xy[0], aout+xy[1], part, "")
	}
}

// THE SECOND FIELD REPORT: AN OUTPUT BELT PLACED ON A RUNNING BALANCER AND THEN
// MINED AGAIN.
//
// Adding a SOUTH-facing output belt off the bottom part takes `bmin` to two in
// and three out, and P = next_pow2(max(N, M)) goes 2 -> 4. Mining that belt again
// takes it back to P = 2, and THAT is the edit nothing else in this suite
// performs -- the machine halves, the reinsertion legitimately runs out of room,
// and carry.go's fourth decision sends the difference somewhere.
func pBminAdd() {
	out.Open("bmin-add begin").End()
	putSoft(surf(), belt, 0, bmin+3, &dirS, "")
	out.Open("bmin-add end").End()
}

func pBminRemove() {
	out.Open("bmin-remove begin").End()
	harness.KillAt(surf(), 0, bmin+3, "", "transport-belt")
	out.Open("bmin-remove end").End()
}

// ---------------------------------------------------------------------------
// THE SIXTY-FIFTH BELT, and the bridge that would be over the limit
// ---------------------------------------------------------------------------

// The sixty-fifth input: one belt below the spare part pointing NORTH, into it.
// Nothing feeds it, exactly as `ain`'s fifth input is unfed -- the classifier does
// not care whether an input belt is carrying anything, and the edit under test is
// the edge COUNT going 64 -> 65.
func pLimAdd() {
	out.Open("lim-add begin").End()
	putSoft(surf(), belt, 0, lim+limRows+1, &dirN, "")
	out.Open("lim-add end").End()
}

// ... and mined again. The edge list goes back to sixty-four, which is the
// fingerprint the guest's netInfo already holds, so the compile is a SKIP: the
// network that was never torn down is never rebuilt either.
func pLimRemove() {
	out.Open("lim-remove begin").End()
	harness.KillAt(surf(), 0, lim+limRows+1, "", "transport-belt")
	out.Open("lim-remove end").End()
}

// The bridging part. The one input belt beside its tile has been standing since
// t=0, so this one placement takes the merged cluster from 64 edges to 65 -- the
// smallest step over the limit this rig can make.
func pBrdgAdd() {
	out.Open("brdg-add begin").End()
	putSoft(surf(), part, 0, brdg+brdgHalf, nil, "")
	out.Open("brdg-add end").End()
}

func pBrdgRemove() {
	out.Open("brdg-remove begin").End()
	harness.KillAt(surf(), 0, brdg+brdgHalf, part, "")
	out.Open("brdg-remove end").End()
}

var brdgA, brdgB, limChest harness.XY

func brdgReport(tag string) {
	s := surf()
	out.Open("brdg tag=").S(tag).S(" tick=").U(harness.Tick()).
		S(" a=").I(chestTotal(s, brdgA)).S(" b=").I(chestTotal(s, brdgB)).End()
}

func limReport(tag string) {
	out.Open("lim tag=").S(tag).S(" tick=").U(harness.Tick()).
		S(" delivered=").I(chestTotal(surf(), limChest)).End()
}

// ---------------------------------------------------------------------------
// the probes
// ---------------------------------------------------------------------------

// THE INSERT PROBE: the miner's-pocket arithmetic, asked of a steel chest.
//
// `insert` is a member of LuaControl and a chest is a LuaControl, so the call the
// pocket makes to a player can be made to a chest -- same member id, same
// signature, same tier-2 encode of the same table -- in a headless run with no
// player anywhere. The guest reports what it asked for, what the engine said it
// took and what the entity holds afterwards; this reads the same three numbers
// back so the guest's own arithmetic is checked against something that did not
// come through the boundary.
//
// The marker is deferred by the guest and runs inside the next flush, which is
// where the pocket runs, so the report is one tick behind the placement.
var probeAt = harness.XY{X: 24, Y: 10}

var probeWant = []struct {
	name  string
	count int64
}{
	{"iron-gear-wheel", 50}, {"iron-plate", 37},
	{"copper-cable", 23}, {"steel-chest", 7},
}

func pInsertProbe() {
	s := surf()
	out.Open("insert-probe begin").End()
	harness.Place(s, harness.Piece{Name: "steel-chest", X: probeAt.X, Y: probeAt.Y})
	harness.PlaceSoft(s, harness.Piece{
		Name: "bbb-insert-probe", X: probeAt.X, Y: probeAt.Y, Raise: true,
	})
}

func pInsertProbeRead() {
	s := surf()
	c, found := harness.FindOnTile(s, "steel-chest", probeAt.X, probeAt.Y)
	for _, w := range probeWant {
		held := int64(-1)
		if found {
			held = harness.ItemCountIn(c, w.name)
		}
		out.Open("insert-probe-lua ").S(w.name).S(" want=").I(w.count).
			S(" held=").I(held).End()
	}
	// AND THEN THE CHEST GOES, CONTENTS AND ALL. The probe's items are MINTED by
	// the probe -- they came from nowhere, which is the point of asking for a
	// known count -- and this suite's whole instrument is a global total that is
	// conserved. Leaving 117 invented items inside the counted area would read as
	// the mod creating them.
	if found {
		harness.Destroy(c, false)
	}
}

// THE PROBE THAT SAYS WHY THE REMOVAL ABOVE IS THE ONLY HALF THAT CAN BE TESTED.
//
// A dissolve started by a PLAYER hands the drained network to that player's
// inventory before anything reaches the ground (carry.go, "the beneficiary"), and
// this suite cannot reach that path. Two walls, and this probe measures both
// rather than asserting them from the documentation:
//
//   - a headless `--create` has no players at all, so `game.get_player(1)` is nil
//     and the beneficiary would fall back to the spill even if one were set;
//   - `on_player_mined_entity` is not one of the events `script.raise_event` will
//     raise -- LuaBootstrap carries a `raise_*` helper for each of the eleven that
//     can be, and there is none for this one.
//
// THE ERROR TEXT IS ASSERTED VERBATIM, and `fk.LastError()` is what makes that
// possible from a guest. A generated binding returns a Status, which can only say
// "the Factorio API raised"; LastError is the SENTENCE it raised with, and its own
// documentation names this as the honest exception to "log it, do not branch on
// it" -- an engine that stopped refusing this should fail a suite rather than
// quietly widen it.
//
// The entity handed to the raise is a belt FOUR tiles from the nearest part, so
// the probe is inert whichever way it goes: a vanish event outside the two-tile
// neighbour gate is rejected in-guest before anything is looked up, and the raise
// does not destroy anything either way.
func pProbePlayerMine() {
	s := surf()
	e, _ := harness.FindAt(s, -4, ntch, "", "transport-belt")

	// THE EVENT IS NAMED BY ITS FACTORIO ID AND NOT BY fkapi's CONSTANT, and
	// that distinction cost this port a run to find.
	//
	// `fkapi.EventOnPlayerMinedEntity` is 114, and it is FkLua's own dense index
	// over the API description's event set -- what `fk.subscribe` takes, which
	// the generated control.lua maps to `defines.events[name]` at load.
	// `script.raise_event` takes FACTORIO'S number, which is 76 on this engine
	// and is not knowable at build time. Handing it 114 does not fail: 114 is
	// `on_player_pipette` to the engine, so the probe cheerfully proved that a
	// DIFFERENT event cannot be raised, and `assert-edge.py` -- which only checks
	// `ok=false` -- passed it. The golden-log diff is what caught it.
	//
	// FkLua excludes `defines.events` from its generated define accessors on
	// purpose ("offering a guest both spellings of on_tick would be a trap
	// dressed as a convenience", internal/factorio/gen.go), and
	// `script.get_event_id` is the door it leaves open: the NAME travels, the
	// engine resolves the number.
	id, err := fkapi.Script.GetEventId(fkapi.OfString("on_player_mined_entity"))
	if err != nil {
		harness.Fatal("script.get_event_id(on_player_mined_entity)", fk.LastError())
		return
	}
	err = fkapi.Script.RaiseEvent(
		fkapi.OfNumber(float64(id)),
		fkapi.OfMap(
			fkapi.KeyValue{Key: fkapi.OfString("player_index"), Val: fkapi.OfNumber(1)},
			fkapi.KeyValue{Key: fkapi.OfString("entity"), Val: fkapi.OfObject(e)},
		))
	msg := ""
	if err != nil {
		msg = collapseSpace(fk.LastError())
	}
	out.Open("player-mine-raise ok=").B(err == nil).S(" err=").S(msg).End()

	p, _ := fkapi.Game.GetPlayer(fkapi.OfNumber(1))
	players, _ := fkapi.Game.Players()
	out.Open("player-resolve p1=").B(p != nil).
		S(" players=").I(int64(len(players))).End()
}

// collapseSpace is the Lua's `tostring(err):gsub("%s+", " ")`: every run of
// whitespace becomes one space, so a message the engine wrapped over two lines
// reads as one.
func collapseSpace(s string) string {
	b := make([]byte, 0, len(s))
	sp := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f' {
			sp = true
			continue
		}
		if sp && len(b) > 0 {
			b = append(b, ' ')
		} else if sp && len(b) == 0 {
			b = append(b, ' ')
		}
		sp = false
		b = append(b, c)
	}
	if sp {
		b = append(b, ' ')
	}
	return string(b)
}

// THE OPERATOR SEAM: the console command and the remote interface the guest
// registers in `init` (guest/go/commands.go).
//
// A console command CANNOT be triggered from script -- 2.0.77 has no
// `commands.run_command` -- so the command leg is asserted as far as Factorio's
// OWN registry and no further: `commands.commands` is the engine's table, not the
// mod's claim about itself, so a name in it is the engine confirming the
// registration reached it.
//
// The remote leg is what drives the handler end to end, and it is evidence about
// the command leg because ONE export serves both: `fk_on_call` is id-dispatched
// and has no branch that can tell which door a call came through.
func pProbeOperators() {
	cmds, _ := fkapi.Commands.Commands()
	registered := false
	for _, c := range cmds {
		if c.Key == "bbb-audit" {
			registered = true
		}
	}
	out.Open("command registered=").B(registered).End()

	ifaces, _ := fkapi.Remote.Interfaces()
	methods := make([]string, 0, 4)
	for _, iface := range ifaces {
		if iface.Key != "better-belt-balancer" {
			continue
		}
		for _, m := range iface.Val {
			methods = append(methods, m.Key)
		}
	}
	harness.SortStrings(methods)
	out.Open("remote iface methods=")
	for i, m := range methods {
		if i > 0 {
			out.S(",")
		}
		out.S(m)
	}
	out.End()

	got, st := fkapi.RemoteCall("better-belt-balancer", "audit")
	clusters := "nil"
	if st == 0 && got.Tag == fkapi.TagNumber {
		clusters = itoa(int(got.Number))
	}
	out.Open("remote audit ok=").B(st == 0).S(" clusters=").S(clusters).End()
}

// ---------------------------------------------------------------------------
// FAST REPLACE
// ---------------------------------------------------------------------------

// frepIn is the input side minus the chest: the frep chests are made in on_init
// so their stock is inside the conserved total from t=0, and `feedIn` would try
// to build a second one on the same tile.
func frepIn(s fkapi.LuaSurface, y int) {
	putTyped(s, loader, -5, y, &dirE, "output", "")
	for x := -4; x <= -1; x++ {
		put(s, belt, x, y, &dirE, "")
	}
}

func frepFeed(s fkapi.LuaSurface, y int) harness.XY {
	frepIn(s, y)
	return drainOut(s, y, "")
}

func pFrepBuild() {
	s := surf()
	out.Open("frep-build begin").End()
	for r := 0; r <= 1; r++ {
		put(s, part, 0, frep+r, nil, "")
		put(s, part, 1, frep+r, nil, "")
	}
	outAdd("frepa", []harness.XY{frepFeed(s, frep), frepFeed(s, frep+1)})
	// The belt line, x = -4 through 0, ENDING on the tile the part is dropped
	// onto. It is an edge of nothing until then: an east-facing belt on a part's
	// SOUTH face is neither `dir` nor `back` and falls through classifySide.
	putTyped(s, loader, -5, frep+2, &dirE, "output", "")
	for x := -4; x <= 0; x++ {
		put(s, belt, x, frep+2, &dirE, "")
	}

	// frepb: a 1->1 row, a three-part NECK carrying nothing, and another 1->1
	// row. The target is the middle of the neck; its two vertical neighbours must
	// be edgeless, or the belt that lands on it would be a second belt for one of
	// them and that half would be refused rather than built.
	for r := 0; r <= 4; r++ {
		put(s, part, 0, frepB+r, nil, "")
	}
	put(s, part, 1, frepB, nil, "")
	put(s, part, 1, frepB+4, nil, "")
	outAdd("frepb", []harness.XY{frepFeed(s, frepB), frepFeed(s, frepB+4)})
	out.Open("frep-build end").End()
}

// A fast replace hands the replaced ENTITY back as an item, and with no player to
// hand it to the engine spills it. That item is new matter -- the engine minted
// it, exactly as the insert probe mints its own -- and `countAll` is a conserved
// quantity, so the machine items are logged and then removed. What the belt was
// CARRYING is left exactly where it fell: that is a real spill and it belongs in
// the count.
var machineItems = []string{belt, part}

// frepSweep looks in both places the engine hands a replaced entity back: onto a
// belt if there is one under it and onto the floor if there is not. The reverse
// gesture is exactly the case where the item lands on a BELT -- the one that was
// just created on that tile.
func frepSweep(tag string, y0, y1 int) {
	s := surf()
	var tally harness.Tally
	took := int64(0)
	where := "ground"
	area := harness.Box(-8, float64(y0), 8, float64(y1))

	for _, e := range harness.EntitiesInOfType(s, area, "item-entity") {
		if n, c, ok := harness.GroundStack(e); ok {
			tally.Add(n, c)
			for _, m := range machineItems {
				if n == m {
					took += c
					harness.Destroy(e, false)
					break
				}
			}
		}
	}

	types := fkapi.OfArray(
		fkapi.OfString("transport-belt"), fkapi.OfString("underground-belt"),
		fkapi.OfString("splitter"), fkapi.OfString("lane-splitter"),
		fkapi.OfString("loader-1x1"), fkapi.OfString("loader"))
	found, _ := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &area, Type: &types})
	for _, e := range found {
		harness.EachLine(e, func(l fkapi.LuaTransportLine) {
			for _, m := range machineItems {
				v := fkapi.OfString(m)
				n, err := l.GetItemCount(&v)
				if err != nil || n == 0 {
					continue
				}
				tally.Add(m, int64(n))
				took += int64(n)
				where = "on a belt"
				l.RemoveItem(fkapi.OfMap(
					fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(m)},
					fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(float64(n))}))
			}
		})
	}

	out.Open("frep-spill tag=").S(tag).S(" handed-back=[")
	for i, n := range tally.Names() {
		if i > 0 {
			out.S(",")
		}
		out.S(n).S("x").I(tally.Get(n))
	}
	out.S("] machine-removed=").I(took).S(" where=").S(where).End()
}

func canFastReplace(s fkapi.LuaSurface, name string, x, y int, dir uint32) bool {
	force, _ := harness.ForceByName("player")
	pos := harness.Center(x, y)
	ok, err := s.CanFastReplace(fkapi.LuaSurfaceCanFastReplaceArgs{
		Name:      fkapi.OfString(name),
		Position:  pos,
		Force:     &force,
		Direction: &dir,
	})
	return err == nil && ok
}

// FORWARD: a part onto the belt line, while everything is running.
//
// THE CREATE IS GATED ON `can_fast_replace` AND THAT IS NOT BELT AND BRACES.
// `create_entity{fast_replace = true}` does not ask that question: handed a
// gesture the engine would refuse it falls back to CREATING, and a
// simple-entity-with-force is created whatever it collides with -- so a guest
// without the prototype line would end up with a part and a belt on one tile, and
// the next compile would try to put an interface there and fail.
func pFrepForward() {
	s := surf()
	out.Open("frep-fwd begin").End()
	can := canFastReplace(s, part, 0, frep+2, dirN)
	out.Open("frep-can what=part-over-belt value=").B(can).End()
	made := false
	if can {
		_, made = harness.PlaceSoft(s, harness.Piece{
			Name: part, X: 0, Y: frep + 2, FastReplace: true, Raise: true,
		})
	}
	_, beltLeft := harness.FindAt(s, 0, frep+2, "", "transport-belt")
	_, partThere := harness.FindAt(s, 0, frep+2, part, "")
	out.Open("frep-fwd created=").B(made).S(" belt-left=").B(beltLeft).
		S(" part-there=").B(partThere).End()
	frepSweep("fwd", frep-3, frep+5)
	out.Open("frep-fwd end").End()
}

// THE REFUSAL: a part that carries an edge interface cannot be belt-replaced,
// because `bbb-linked-belt` is a belt-connectable standing on that same tile.
// `can_fast_replace` is the engine's answer to a PLAYER and it is the assertion.
//
// `create_entity` does not ask that question -- it mines the part and only then
// discovers it cannot place the belt -- so the part is put back. A player cannot
// reach that state and a phantom left standing here would be an artefact in every
// audit after it rather than the thing under test.
func pFrepEdge() {
	s := surf()
	out.Open("frep-edge begin").End()
	out.Open("frep-can what=belt-over-edge-part value=").
		B(canFastReplace(s, belt, 0, frepB, dirS)).End()
	_, made := harness.PlaceSoft(s, harness.Piece{
		Name: belt, X: 0, Y: frepB, Dir: &dirS, FastReplace: true, Raise: true,
	})
	_, still := harness.FindAt(s, 0, frepB, part, "")
	out.Open("frep-edge created=").B(made).S(" part-survived=").B(still).End()
	if !still {
		harness.PlaceSoft(s, harness.Piece{Name: part, X: 0, Y: frepB})
	}
	frepSweep("edge", frepB-3, frepB+8)
	out.Open("frep-edge end").End()
}

// REVERSE: a south-facing belt onto an INTERIOR part, which splits the column
// into a two-part cluster above and a one-part cluster below, with the new belt
// an OUTPUT of the first and an INPUT of the second.
//
// Nothing tells the guest this happened except the belt's own build event, so
// without guest/go/fastreplace.go the registry keeps four parts in one cluster,
// the belt is INTERIOR and therefore never classified, the fingerprint never
// moves, and nothing at all is rebuilt.
func pFrepReverse() {
	s := surf()
	out.Open("frep-rev begin").End()
	can := canFastReplace(s, belt, 0, frepB+2, dirS)
	out.Open("frep-can what=belt-over-interior-part value=").B(can).End()
	made := false
	if can {
		_, made = harness.PlaceSoft(s, harness.Piece{
			Name: belt, X: 0, Y: frepB + 2, Dir: &dirS, FastReplace: true, Raise: true,
		})
	}
	_, partLeft := harness.FindAt(s, 0, frepB+2, part, "")
	_, beltThere := harness.FindAt(s, 0, frepB+2, "", "transport-belt")
	out.Open("frep-rev created=").B(made).S(" part-left=").B(partLeft).
		S(" belt-there=").B(beltThere).End()
	frepSweep("rev", frepB-3, frepB+8)
	out.Open("frep-rev end").End()
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

func ac(tag string) func() { return func() { auditAndCount(tag) } }

var schedule = []harness.Step{
	{Tick: chnEnd + 20, Do: ac("pre-sametick")},
	{Tick: chnEnd + 24, Do: pSameTick},
	{Tick: chnEnd + 28, Do: ac("post-sametick")},
	{Tick: chnEnd + 32, Do: pPendingA},
	{Tick: chnEnd + 33, Do: pPendingB},
	{Tick: chnEnd + 38, Do: ac("post-pending")},
	{Tick: chnEnd + 42, Do: pPendingUndo},
	{Tick: chnEnd + 46, Do: ac("post-pending-undo")},

	{Tick: chnEnd + 70, Do: ac("pre-merge")},
	{Tick: chnEnd + 72, Do: pMerge},
	{Tick: chnEnd + 76, Do: func() { auditAndCount("post-merge"); probePlacement("post-merge") }},
	{Tick: chnEnd + 110, Do: pUnmerge},
	{Tick: chnEnd + 114, Do: ac("post-split")},

	{Tick: chnEnd + 140, Do: ac("pre-rot")},
	{Tick: chnEnd + 142, Do: pRotSilent},
	{Tick: chnEnd + 144, Do: ac("post-rot-silent")},
	{Tick: chnEnd + 148, Do: pRotRestore},
	{Tick: chnEnd + 150, Do: ac("post-rot-restored")},
	{Tick: chnEnd + 154, Do: pRotEvent},
	{Tick: chnEnd + 158, Do: ac("post-rot-event")},
	{Tick: chnEnd + 162, Do: pRotEventBack},
	{Tick: chnEnd + 166, Do: ac("post-rot-event-back")},

	{Tick: chnEnd + 200, Do: ac("pre-forces")},
	{Tick: chnEnd + 202, Do: pForcesInterleaved},
	{Tick: chnEnd + 206, Do: ac("post-forces-interleaved")},
	{Tick: chnEnd + 240, Do: pForcesMerge},
	{Tick: chnEnd + 244, Do: ac("post-forces-merge")},

	{Tick: chnEnd + 300, Do: func() { detPaste(det1, "1") }},
	{Tick: chnEnd + 303, Do: func() { detFlushed("1") }},
	{Tick: chnEnd + 310, Do: func() { detClear(det1) }},
	{Tick: chnEnd + 320, Do: func() { detPaste(det2, "2") }},
	{Tick: chnEnd + 323, Do: func() { detFlushed("2") }},
	{Tick: chnEnd + 330, Do: func() { detClear(det2) }},

	{Tick: chnEnd + 350, Do: pHiddenPart},
	{Tick: chnEnd + 354, Do: ac("post-hidden-part")},

	// The field report, and the three shapes of it. Nothing here changes the
	// number of parts or of clusters: every one of them is an EDGE edit on a
	// balancer that is running, which is exactly the case a recompile must not
	// put on the floor.
	{Tick: chnEnd + 400, Do: ac("pre-add-out")},
	{Tick: chnEnd + 404, Do: pAddOutput},
	{Tick: chnEnd + 408, Do: func() { auditAndCount("post-add-out"); probePlacement("post-add-out") }},
	// ... and then the balance property of what it became, over a 500-tick window
	// that opens well after the new port's own belt and loader have filled. It is
	// a RATE, not a total.
	{Tick: chnEnd + 560, Do: func() { report("aout-a") }},

	// bmin's GROWING half. It has to happen here rather than beside its own
	// shrinking half, because the network it grows into must be FULL before that
	// one comes down and a P=4 butterfly takes a few hundred ticks to back up out
	// of two express belts.
	{Tick: chnEnd + 596, Do: ac("pre-bmin")},
	{Tick: chnEnd + 600, Do: pBminAdd},
	{Tick: chnEnd + 604, Do: ac("post-bmin-add")},

	{Tick: chnEnd + 1060, Do: func() { report("aout-b") }},
	// The notch rig has been saturated and moving for the whole run by now, which
	// is the condition the field report was made under: the artifact was
	// invisible until items were flowing.
	{Tick: chnEnd + 1062, Do: func() { probePlacement("flowing") }},

	{Tick: chnEnd + 1080, Do: ac("pre-add-in")},
	{Tick: chnEnd + 1084, Do: pAddInput},
	{Tick: chnEnd + 1088, Do: ac("post-add-in")},

	{Tick: chnEnd + 1120, Do: ac("pre-shrink")},
	{Tick: chnEnd + 1124, Do: pRemoveOutput},
	{Tick: chnEnd + 1128, Do: ac("post-shrink")},

	{Tick: chnEnd + 1136, Do: pProbePlayerMine},
	// The operator seam. It runs BEFORE the removal legs below, because
	// `remote.call('...','audit')` really does audit -- it drains the deferred
	// queue and recompiles -- so it must not land between a leg that mines
	// something and the count that describes it.
	{Tick: chnEnd + 1138, Do: pProbeOperators},
	{Tick: chnEnd + 1140, Do: pRemoveCluster},
	{Tick: chnEnd + 1144, Do: ac("post-remove")},

	// ... and bmin's SHRINKING half, five hundred ticks after the belt was laid,
	// which is the field report itself. It is scheduled after every tag whose
	// assertion is that the ground is EMPTY -- ground is cumulative over the run,
	// and this leg deliberately puts items on it.
	{Tick: chnEnd + 1160, Do: ac("pre-bmin-remove")},
	{Tick: chnEnd + 1164, Do: pBminRemove},
	{Tick: chnEnd + 1168, Do: ac("post-bmin-remove")},

	// The field report's own gesture: the `aout` rig taken apart ONE PART PER
	// TICK. TEN STEPS, because the rig is ten parts: eight for a 4x4 under the
	// one-belt rule and two more for the spare row `pAddOutput` needed.
	{Tick: chnEnd + 1200, Do: ac("pre-hand")},
	{Tick: chnEnd + 1204, Do: pHandMine(1)},
	{Tick: chnEnd + 1208, Do: ac("hand-1")},
	{Tick: chnEnd + 1212, Do: pHandMine(2)},
	{Tick: chnEnd + 1216, Do: ac("hand-2")},
	{Tick: chnEnd + 1220, Do: pHandMine(3)},
	{Tick: chnEnd + 1224, Do: ac("hand-3")},
	{Tick: chnEnd + 1228, Do: pHandMine(4)},
	{Tick: chnEnd + 1232, Do: ac("hand-4")},
	{Tick: chnEnd + 1236, Do: pHandMine(5)},
	{Tick: chnEnd + 1240, Do: ac("hand-5")},
	{Tick: chnEnd + 1244, Do: pHandMine(6)},
	{Tick: chnEnd + 1248, Do: ac("hand-6")},
	{Tick: chnEnd + 1252, Do: pHandMine(7)},
	{Tick: chnEnd + 1256, Do: ac("hand-7")},
	{Tick: chnEnd + 1260, Do: pHandMine(8)},
	{Tick: chnEnd + 1264, Do: ac("hand-8")},
	{Tick: chnEnd + 1268, Do: pHandMine(9)},
	{Tick: chnEnd + 1272, Do: ac("hand-9")},
	{Tick: chnEnd + 1276, Do: pHandMine(10)},
	{Tick: chnEnd + 1280, Do: ac("hand-10")},

	{Tick: chnEnd + 1288, Do: pInsertProbe},
	{Tick: chnEnd + 1294, Do: pInsertProbeRead},

	{Tick: chnEnd + 1298, Do: func() {
		auditAndCount("final")
		report("final")
		probePlacement("final")
	}},

	// THE SIXTY-FIFTH BELT, last because it is the only leg whose rig is
	// thirty-two parts and because the tail of this suite is quiet.
	//
	// The two windows are the same length (246 ticks) and are compared as a
	// ratio. They have to be: sixty-three of this rig's output ports dead-end, so
	// the share of the input that reaches the live one climbs all run as they
	// fill, and a constant would be a number copied from a passing run.
	{Tick: chnEnd + 1304, Do: func() { limReport("before-open") }},
	{Tick: chnEnd + 1548, Do: ac("pre-lim")},
	{Tick: chnEnd + 1550, Do: func() { limReport("before-close") }},
	{Tick: chnEnd + 1552, Do: pLimAdd},
	{Tick: chnEnd + 1556, Do: func() { auditAndCount("post-lim"); limReport("after-open") }},
	{Tick: chnEnd + 1802, Do: func() { limReport("after-close") }},
	{Tick: chnEnd + 1806, Do: ac("post-lim-window")},
	{Tick: chnEnd + 1810, Do: pLimRemove},
	{Tick: chnEnd + 1814, Do: ac("post-lim-back")},

	// THE BRIDGE THAT WOULD BE OVER THE LIMIT, last of all and for the same
	// reason `lim` is next-to-last.
	{Tick: chnEnd + 1848, Do: func() { brdgReport("before-open") }},
	{Tick: chnEnd + 2092, Do: ac("pre-brdg")},
	{Tick: chnEnd + 2094, Do: func() { brdgReport("before-close") }},
	{Tick: chnEnd + 2096, Do: pBrdgAdd},
	{Tick: chnEnd + 2100, Do: func() {
		auditAndCount("post-brdg")
		brdgReport("after-open")
		probePlacement("brdg")
	}},
	// Two more audits while the refusal STANDS. The state a refused merge leaves
	// is the only one in this guest where `nets` holds a network under a key that
	// is no longer a root, and it has to be stable.
	{Tick: chnEnd + 2198, Do: ac("brdg-hold-1")},
	{Tick: chnEnd + 2298, Do: ac("brdg-hold-2")},
	{Tick: chnEnd + 2346, Do: func() { brdgReport("after-close") }},
	{Tick: chnEnd + 2350, Do: ac("post-brdg-window")},
	{Tick: chnEnd + 2354, Do: pBrdgRemove},
	{Tick: chnEnd + 2358, Do: ac("post-brdg-back")},
	// The recovery window opens a hundred ticks after the un-merge: the half
	// whose root the merged cluster had been using is torn down and rebuilt from
	// its own carry pool, and a rebuild puts every reinserted item back at the
	// HEAD of the butterfly, so its output is briefly starved by construction.
	{Tick: chnEnd + 2458, Do: func() { brdgReport("back-open") }},
	{Tick: chnEnd + 2704, Do: func() { brdgReport("back-close") }},
	{Tick: chnEnd + 2708, Do: ac("post-brdg-final")},

	// FAST REPLACE, after everything else, and that is deliberate: these two rigs
	// are BUILT HERE rather than in on_init so that every baseline above is a
	// statement about the same world it has always been about.
	{Tick: chnEnd + 2728, Do: pFrepBuild},
	// Three hundred and twenty ticks to fill: source chest, loader, four belts, a
	// P=2 hidden network and three more belts before the first item reaches a
	// chest.
	{Tick: chnEnd + 3048, Do: func() {
		auditAndCount("frep-built")
		report("frep-before-open")
	}},
	{Tick: chnEnd + 3398, Do: func() { report("frep-before-close") }},
	{Tick: chnEnd + 3402, Do: ac("pre-frep")},
	{Tick: chnEnd + 3406, Do: pFrepForward},
	{Tick: chnEnd + 3410, Do: ac("post-frep-fwd")},
	{Tick: chnEnd + 3414, Do: pFrepEdge},
	{Tick: chnEnd + 3418, Do: ac("post-frep-edge")},
	{Tick: chnEnd + 3422, Do: pFrepReverse},
	{Tick: chnEnd + 3426, Do: ac("post-frep-rev")},
	// The after-window opens 162 ticks past the last edit and is the same length
	// as the before-window, so the two are a ratio.
	{Tick: chnEnd + 3588, Do: func() { report("frep-after-open") }},
	{Tick: chnEnd + 3938, Do: func() { report("frep-after-close") }},
	{Tick: chnEnd + 3942, Do: func() { auditAndCount("frep-final"); probePlacement("frep") }},

	{Tick: chnEnd + 3946, Do: func() { out.Open("done").End() }},
}

// ---------------------------------------------------------------------------
// on_init
// ---------------------------------------------------------------------------

// rig builds one rig: `nrows` rows of TWO parts, of which the first `fed` are fed
// and drained. A row that is not fed carries nothing at all on either of its
// parts, which is the only place in a working balancer a player's belt can still
// land.
func rig(s fkapi.LuaSurface, name string, base, nrows int, force string, fed int) []harness.XY {
	if fed == 0 {
		fed = nrows
	}
	for r := 0; r < nrows; r++ {
		put(s, part, 0, base+r, nil, force)
		put(s, part, 1, base+r, nil, force)
	}
	chests := make([]harness.XY, 0, fed)
	for r := 0; r < fed; r++ {
		chests = append(chests, feed(s, base+r, force))
	}
	if name != "" {
		outAdd(name, chests)
	}
	return chests
}

// The five sections below are separate functions for READABILITY and for no
// other reason, and that is worth saying because the obvious other reason turned
// out not to apply.
//
// `fk_on_init` here builds fifteen clusters over one hundred and ninety-eight
// parts, which is by a wide margin the largest one in the estate, and FkLua's
// packager refuses an emitted function whose JUMP SPAN exceeds 655,355 bytes --
// naming `//go:noinline` per section as the remedy. Three phases of this port
// recorded that check as "not fired yet, and `edge` is the first place likely to
// meet it". IT DOES NOT MEET IT: measured, this observer packages with no pragmas
// at all and with these sections free to be inlined into one function, well
// inside a threshold whose own documentation puts the widest span across all of
// FkLua's guests at 248,861 bytes. So there are no pragmas here, because one
// carrying a reason that is not true is worse than none.
func buildEarlyRigs(s fkapi.LuaSurface) {
	// chn's outputs DEAD-END: belts, and nothing to take the items off them. The
	// whole hidden network backs up and stays full, so every one of the two
	// hundred teardowns below drains a network that is carrying items.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, chn+r, nil, "")
		put(s, part, 1, chn+r, nil, "")
	}
	for r := 0; r <= 1; r++ {
		source(s, chn+r, "")
		for x := -4; x <= -1; x++ {
			put(s, belt, x, chn+r, &dirE, "")
		}
		for x := 2; x <= 4; x++ {
			put(s, belt, x, chn+r, &dirE, "")
		}
	}
	rig(s, "same", same, 2, "", 0)
	// Two clusters with a one-tile gap at MRG+2. Row MRG+2 gets no belts, so the
	// bridging part brings no edge of its own -- the whole change is the merge.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, mrg+r, nil, "")
		put(s, part, 1, mrg+r, nil, "")
	}
	for r := 3; r <= 4; r++ {
		put(s, part, 0, mrg+r, nil, "")
		put(s, part, 1, mrg+r, nil, "")
	}
	mrgChests := make([]harness.XY, 0, 4)
	for _, r := range []int{0, 1, 3, 4} {
		mrgChests = append(mrgChests, feed(s, mrg+r, ""))
	}
	outAdd("mrg", mrgChests)
	rig(s, "rot", rot, 2, "", 0)
	rig(s, "frcA", frc, 2, "", 0)
	rig(s, "frcB", frc+2, 2, otherForce, 0)
}

func buildLoadRigs(s fkapi.LuaSurface) {
	// The three recompile-under-load rigs. Four in and four out over eight parts,
	// saturated -- and `aout` and `ain` carry a fifth row that holds nothing, so
	// that the belt each of them is given mid-run has somewhere legal to land.
	// The part count never moves in any of the three: every edit is a belt.
	rig(s, "aout", aout, 5, "", 4)
	rig(s, "ain", ain, 5, "", 4)
	rig(s, "shrk", shrk, 4, "", 0)

	// ntch: the field report's own shape, saturated and left running for the whole
	// suite. A C of five parts around a HOLE at (1, b+1) -- the one tile in this
	// save enclosed by parts on three sides and not one itself, so it is where a
	// stray visible entity would be most visible.
	b := ntch
	put(s, part, 0, b, nil, "")
	put(s, part, 1, b, nil, "")
	put(s, part, 0, b+1, nil, "")
	put(s, part, 0, b+2, nil, "")
	put(s, part, 1, b+2, nil, "")
	for _, r := range []int{0, 1} {
		feedIn(s, b+r, "")
	}
	north := drainOut(s, b, "")
	south := drainOut(s, b+2, "")
	outAdd("ntch", []harness.XY{north, south})

	// bmin: a 2->2 over four parts, DEAD-ENDED like chn, plus an attached FIFTH
	// PART carrying nothing. That fifth part is where the third output belt goes:
	// under the one-belt rule the four working parts have no free face between
	// them. It is registered in no output list because it has no chests; what is
	// measured about it is the ground.
	for r := 0; r <= 1; r++ {
		put(s, part, 0, bmin+r, nil, "")
		put(s, part, 1, bmin+r, nil, "")
	}
	put(s, part, 0, bmin+2, nil, "")
	for r := 0; r <= 1; r++ {
		source(s, bmin+r, "")
		for x := -4; x <= -1; x++ {
			put(s, belt, x, bmin+r, &dirE, "")
		}
		for x := 2; x <= 4; x++ {
			put(s, belt, x, bmin+r, &dirE, "")
		}
	}
}

func buildLim(s fkapi.LuaSurface) {
	// lim: THE BIGGEST BALANCER THIS MOD BUILDS, one belt short of refusing.
	put(s, part, 0, lim-1, nil, "")
	put(s, belt, 0, lim-2, &dirN, "")
	putTyped(s, loader, 0, lim-3, &dirN, "input", "")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: lim - 4})
	limChest = harness.XY{X: 0, Y: lim - 4}
	for r := 0; r < limRows; r++ {
		put(s, part, 0, lim+r, nil, "")
		put(s, part, 1, lim+r, nil, "")
		put(s, belt, -1, lim+r, &dirE, "")
		put(s, belt, 2, lim+r, &dirW, "")
	}
	put(s, part, 0, lim+limRows, nil, "")
	for r := 0; r < limFed; r++ {
		source(s, lim+r, "")
		for x := -4; x <= -2; x++ {
			put(s, belt, x, lim+r, &dirE, "")
		}
	}
}

func buildBrdg(s fkapi.LuaSurface) {
	// brdg: TWO balancers with a one-tile gap, whose MERGE is over the limit. The
	// GAP TILE gets ONE input belt beside it here and keeps it for the whole run:
	// it is an edge of nothing until a part stands there, and then it is the one
	// that takes the merged cluster to sixty-five.
	gap := brdg + brdgHalf
	foot := brdg + 2*brdgHalf
	put(s, part, 0, brdg-1, nil, "")
	put(s, belt, 0, brdg-2, &dirN, "")
	putTyped(s, loader, 0, brdg-3, &dirN, "input", "")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: brdg - 4})
	brdgA = harness.XY{X: 0, Y: brdg - 4}
	for r := 0; r <= 2*brdgHalf; r++ {
		if r == brdgHalf {
			continue
		}
		put(s, part, 0, brdg+r, nil, "")
		put(s, part, 1, brdg+r, nil, "")
		put(s, belt, -1, brdg+r, &dirE, "")
		put(s, belt, 2, brdg+r, &dirW, "")
	}
	put(s, belt, -1, gap, &dirE, "")
	put(s, part, 0, foot+1, nil, "")
	put(s, belt, 0, foot+2, &dirS, "")
	putTyped(s, loader, 0, foot+3, &dirS, "input", "")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 0, Y: foot + 4})
	brdgB = harness.XY{X: 0, Y: foot + 4}
	for r := 0; r < brdgFed; r++ {
		for _, y := range []int{brdg + r, gap + 1 + r} {
			source(s, y, "")
			for x := -4; x <= -2; x++ {
				put(s, belt, x, y, &dirE, "")
			}
		}
	}
}

func buildFrepChests(s fkapi.LuaSurface) {
	// THE FAST-REPLACE RIGS' SOURCE CHESTS, AND NOTHING ELSE OF THEM. The rigs
	// themselves are built mid-run so that no baseline above moves, but their
	// stock has to be inside the conserved total from t=0: `countAll` is the
	// instrument this whole suite rests on, and thirty thousand items appearing
	// in it halfway through would read as the mod minting matter.
	for _, y := range []int{frep, frep + 1, frep + 2, frepB, frepB + 4} {
		c := harness.Place(s, harness.Piece{Name: "steel-chest", X: -6, Y: y})
		harness.InsertInto(c, "iron-plate", stock)
	}
}

//go:wasmexport fk_on_init
func onInit() {
	s := makeSurface()
	harness.CreateForce(otherForce)

	buildEarlyRigs(s)
	buildLoadRigs(s)
	buildLim(s)
	buildBrdg(s)
	buildFrepChests(s)

	// `--create` never reaches a tick, so the flush every build armed would land
	// on the first tick of the benchmark. The marker drains it here.
	out.Open("mark tag=init").End()
	harness.Audit(s, 24, 4)
	// Every network in the save now exists, so this is the widest sample the
	// placement probe ever gets.
	probePlacement("init")
	// The real number, not the requested one: a steel chest holds 48 stacks, so
	// an insert of stock stops at 4,800 whatever stock says.
	total, _ := countAll()
	out.Open("plan chn_end=").I(chnEnd).S(" end_tick=").I(chnEnd + 3950).
		S(" stock=").I(total).End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	t := fkapi.ReadOnTick(ptr).Tick
	if t >= chnT0 && t < chnEnd {
		d := int(t - chnT0)
		chnStep(d%chnPeriod, d/chnPeriod+1)
		return
	}
	harness.Run(schedule, t)
}

func main() {}
