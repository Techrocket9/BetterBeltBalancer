package main

// THE 64-PORT CAP, MET FROM THE PLAYER'S SIDE.
//
// `plan.MaxPorts` is 64 and refusing past it is correct -- agents/maxports.md
// §1-§3 is why the number is what it is and what raising it would really cost.
// This file is about everything AROUND the refusal, which until 2026-08-04 was
// hostile to the one person it ever happens to.
//
// WHAT IT USED TO DO. compile() classified the edges, saw the fingerprint move,
// called `teardownForRebuild(root)`, and only THEN asked plan.Build whether the
// shape fit. So the sixty-fifth belt laid against a working 64-port balancer
//
//	(a) demolished the running network,
//	(b) built nothing in its place,
//	(c) spilled the entire contents beside the cluster, and
//	(d) said so only in the log file.
//
// Every one of those is a consequence of asking the question one statement too
// late. The refusal itself was never wrong.
//
// WHAT IT DOES NOW, in the three pieces below:
//
//  1. THE CHECK MOVES IN FRONT OF THE TEARDOWN (compile.go). The edge counts are
//     in hand the moment classifyEdges returns, and `plan.Shape` needs nothing
//     else, so the answer is available before anything has been touched. The
//     standing network is left exactly as it was and keeps running; the belt
//     that could not be joined simply stands there unconnected, which is the
//     same inert degradation the failure envelope already documents for a belt
//     the guest never heard about (CLAUDE.md, "The failure envelope").
//
//  2. THE PLAYER IS TOLD. A build event carries a `player_index`, and this file
//     writes down the scalars -- surface, tile, player -- for every addition
//     that lands on or beside a cluster, exactly the way carry.go's claim store
//     writes down a mine. At refusal time the player is resolved FRESH from
//     `game.get_player`, gets a flying text at the balancer and the standard
//     `utility/cannot_build`, and a robot or script build (no player to tell)
//     gets a force-wide message instead.
//
//  3. THE PIECE IS HANDED BACK. Factorio has no cancelable build event, so the
//     standard emulation is revert-on-the-next-tick: re-find the entity at the
//     tile it was recorded on and `mine_entity` it into the placing player's
//     inventory. That happens AFTER the flush's queue drain, never during it --
//     see revertOverLimit, which is the sharpest edge in the file.
//
//  4. AND THE MERGE IS REFUSED IN FRONT OF THE TEARDOWNS TOO. A part that
//     BRIDGES two clusters into an over-limit one is queued by `AddPart` as two
//     DEAD roots and one live one, so piece 1 above is useless on its own: both
//     predecessors' networks are already down by the time flushLive reaches the
//     merged cluster. `spareOverLimitMerges` runs at the top of flushDead, finds
//     the merge products before anything has been touched, and pulls their
//     predecessors back off the teardown queue. See "The merge" below for the
//     free arithmetic that keeps it off the hot path and for what the registry
//     then believes.
//
// SINCE THE 2.1 PORT THERE ARE TWO BOUNDS AND ONE REFUSAL. The port limit is
// this file's; the one-belt-per-part rule Factorio 2.1 forces is sedge.go's.
// They share everything below the sentence: `refuseAdmit`'s three-way
// admission (the wake-race guard and the once-per-edge-state gate),
// `tellRefusal`'s three arms, `limPending`'s hand-back, and the merge pre-pass,
// which asks both questions of a candidate before it spares anything. What
// differs is the predicate and the two locale keys, which is the whole of what
// a bound is.

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

// The locale keys, in mod-data/locale/en/better-belt-balancer.cfg. A
// LocalisedString rather than a sentence built here: the guest has no business
// knowing what language anybody is playing in, and a flying text is the one
// thing this mod puts in front of a player in words.
const (
	msgOverLimit      = "bbb.over-port-limit"
	msgOverLimitStand = "bbb.over-port-limit-unconnected"
	soundCannotBuild  = "utility/cannot_build"
)

// ---------------------------------------------------------------------------
// What a player just built
// ---------------------------------------------------------------------------

// buildNote is one addition a PLAYER made, for the one tick between the build
// event that reported it and the flush that decides whether it fits.
//
// SCALARS AND NOTHING ELSE, for the reason every persistent thing in this guest
// is scalars: the build arrives in one dispatch, the compile happens in the
// deferred flush in the next, and no `LuaEntity` or `LuaPlayer` survives that
// gap. The tile is the ENTITY'S OWN -- which is the opposite of a carry claim,
// whose tile is always the NETWORK'S -- because the thing this note is for is
// handing that entity back, and a tile is how it gets re-found.
type buildNote struct {
	s      uint32
	x, y   int32
	force  uint32
	player uint32
	// isPart distinguishes the two shapes without a stored name: a part stands
	// ON a registry tile and a belt at the edge stands one tile OUTSIDE the
	// cluster's box by construction, so the flag and the registry cross-check
	// each other for free at revert time.
	isPart bool
}

var (
	// buildNotes is one tick's worth, IN EVENT ORDER, truncated by every flush.
	// High-water like the recompile queues, so a blueprint paste sizes it once
	// and every paste after that reuses the same backing array.
	buildNotes []buildNote

	// limPending is what the flush decided to hand back, filled during the
	// drain and consumed after it. Separate from buildNotes because the two
	// have different lifetimes by one function call, and that call is the whole
	// reentrancy argument in revertOverLimit.
	limPending []buildNote

	// overLimit remembers, per cluster root, the edge-list fingerprint that was
	// last refused.
	//
	// IT IS A FEEDBACK GATE AND NOT STATE THE COMPILER READS. A refused cluster
	// is re-queued by every event within two tiles of it and by every audit --
	// and its fingerprint can never match its netInfo's, so it reaches this
	// refusal every single time. Without the memory a player who leaves the
	// sixty-fifth belt standing gets a flying text and a `cannot_build` on every
	// audit for the rest of the session.
	//
	// POINT-QUERIED ONLY, never ranged and never iterated, so it carries no
	// iteration order into anything host-visible. Nil until the first refusal:
	// a lookup and a `delete` on a nil map are both legal and free, so a save
	// that never hits the cap never allocates it.
	overLimit map[uint32]uint64

	// rebuildingFromWorld is up for exactly the span of rebuildFromWorld,
	// including the flush it runs itself. A refusal issued under it logs and is
	// buffered in rebuildRefused instead of speaking or arming the memo -- the
	// rebuild judges with the worst information a refusal will ever have, and
	// both 2026-08-05 field reports were messages sent from this window (see
	// refuseOverLimit). rebuildRefused is truncated by the requeue that
	// consumes it; nil forever in a session that never rebuilds over a refused
	// cluster, which is every session of ordinary play.
	rebuildingFromWorld bool
	rebuildRefused      []uint32
)

// noteBuiltByPlayer records that `player` built something at (x, y).
//
// Called from the two build paths -- a balancer part, and a belt-connectable
// inside the two-tile neighbour gate -- and from nowhere else. It makes NO HOST
// CALL and returns on the zero player before it touches anything, which is what
// keeps it off the guest's highest-multiplier path: `player` is zero for a
// robot, for a script build and for every event in every headless suite, so the
// marathon suite's per-operation slopes cannot move (CLAUDE.md, "The marathon
// save").
//
// Exact duplicates are dropped for the same reason `carry.Claims.Add` drops
// them: the neighbour gate walks a 5x5 neighbourhood and could otherwise record
// one belt several times over.
func noteBuiltByPlayer(s uint32, x, y int32, force, player uint32, isPart bool) {
	if player == 0 {
		return
	}
	n := buildNote{s: s, x: x, y: y, force: force, player: player, isPart: isPart}
	for i := range buildNotes {
		if buildNotes[i] == n {
			return
		}
	}
	buildNotes = append(buildNotes, n)
}

// forgetBuildNotes ends the tick. A note outlives the event that made it by one
// flush and no longer -- the flush that could have used it has run.
func forgetBuildNotes() { buildNotes = buildNotes[:0] }

// ---------------------------------------------------------------------------
// The refusal
// ---------------------------------------------------------------------------

// overLimitShape is the check compile() makes BEFORE it touches anything.
//
// It mirrors `plan.Build`'s own two tests exactly, and the mirroring is the
// point: a cluster with no inputs or no outputs is a legitimate half-built
// state that Build accepts at any size, so refusing one here would refuse
// every single-sided cluster in the world. Build keeps its `!fits` branch as an
// unreachable backstop, and reaching it is an `error:` because the two have
// disagreed.
func overLimitShape(edges []plan.Edge) (plan.Ports, bool) {
	n, m := 0, 0
	for i := range edges {
		if edges[i].Out {
			m++
		} else {
			n++
		}
	}
	pt := plan.Shape(n, m)
	return pt, pt.N > 0 && pt.M > 0 && pt.P > plan.MaxPorts
}

// forgetOverLimit clears the feedback gate for a root.
//
// Called from every compile that did NOT refuse -- a success, a skip, a cluster
// with nothing to balance -- and from teardownNet, because a network coming
// down means the cluster's state is being rewritten wholesale.
//
// WHAT IS LEFT UNCLEARED IS ONE ENTRY AND IT CANNOT MISLEAD. A cluster that
// dissolves while refused never compiles again, so its entry survives until the
// node id is reused. A false suppression would then need the reusing cluster to
// be over limit AND to hash to the same fingerprint -- and the fingerprint is
// FNV-1a over the edges' absolute TILE COORDINATES, so that means the same
// geometry at the same place, which is the same balancer rebuilt and the same
// message already delivered.
func forgetOverLimit(root uint32) {
	if len(overLimit) != 0 {
		delete(overLimit, root)
	}
}

// logRefusedOverLimit is the refusal's log line, shared by the ordinary path
// and the silent rebuild path. `alert:` and NOT `error:`: an error in this
// guest means one thing -- a compile did not produce a network that should
// have -- and `test/run.sh` fails a run on one. A player asking for a balancer
// bigger than the mod builds is an expected condition with a defined outcome,
// and the edge suite asserts this very line.
func logRefusedOverLimit(root uint32, pt plan.Ports) {
	logAlertStart("cluster ")
	logU(root)
	logS(" would need ")
	logU(uint32(pt.P))
	logS(" ports for ")
	logU(uint32(pt.N))
	logS(" inputs and ")
	logU(uint32(pt.M))
	logS(" outputs, over the limit of ")
	logU(plan.MaxPorts)
	logS("; refused BEFORE the teardown, so the standing network is untouched")
	logEnd()
}

// What a refusal is allowed to do this time. Every refusal in this guest goes
// through refuseAdmit, whichever bound it broke -- the port limit here, the
// one-belt-per-part rule in sedge.go -- because the wake-race discipline and
// the feedback gate are properties of REFUSING, not of the bound.
const (
	refuseSilent  = iota // this exact edge state has been refused before
	refuseLogOnly        // inside rebuildFromWorld: log, requeue, tell nobody
	refuseSpeak          // log, and put it in front of somebody
)

// refuseAdmit decides which of the three a refusal is, and arms the gate when
// the answer is the third.
//
// A REBUILD-FROM-WORLD REFUSAL LOGS, REQUEUES AND TELLS NOBODY. The rebuild's
// flush runs with the worst information a refusal will ever have -- when it
// runs from the first event of a session, that event's own build note does not
// exist yet -- and anything said now is either the wrong arm or a claim about a
// state the next tick falsifies (both happened; the 2026-08-05 field reports and
// lifecycle.go's comment are the record). So: the memo is NOT armed, the cluster
// goes back on the queue, and the first ordinary flush re-judges with the notes
// in hand and delivers the one correct message. A refused cluster in a save
// nobody is editing reaches that flush with no notes and gets the piece-stands
// message, which is then true.
//
// Otherwise it is ONCE PER DISTINCT EDGE STATE, not once per flush. See
// overLimit.
func refuseAdmit(root uint32, fp uint64) int {
	if rebuildingFromWorld {
		// NOT markLive: this runs inside the rebuild's own flushLive drain,
		// whose loop has already captured its length and whose tail truncates
		// the queue to [:0] -- an append here would be silently erased.
		// rebuildFromWorld requeues these AFTER its flush returns, the same
		// after-the-drain discipline revertOverLimit follows.
		rebuildRefused = append(rebuildRefused, root)
		return refuseLogOnly
	}
	if prev, ok := overLimit[root]; ok && prev == fp {
		return refuseSilent
	}
	if overLimit == nil {
		overLimit = make(map[uint32]uint64)
	}
	overLimit[root] = fp
	return refuseSpeak
}

// refuseOverLimit is compile()'s answer when a cluster's edge list has outgrown
// plan.MaxPorts. The standing network has not been touched and must not be.
//
// Returns with the world exactly as it found it: everything below is a log
// line, a message and a queue insertion.
func refuseOverLimit(root uint32, fp uint64, pt plan.Ports, tiles []key, force uint32) {
	switch refuseAdmit(root, fp) {
	case refuseSilent:
		return
	case refuseLogOnly:
		logRefusedOverLimit(root, pt)
		return
	}
	logRefusedOverLimit(root, pt)
	limMsg[1].Number = float64(plan.MaxPorts)
	// The number the PLAYER sees is the belt count on the machine's bigger
	// side, never pt.P: a player placed one belt and has no window into the
	// compiler, so "would need 128 ports" for a 65th belt read as gibberish in
	// the field (report, 2026-08-05). The log line above keeps P -- it is for
	// whoever reads logs, and ports are its native unit.
	side := pt.N
	if pt.M > side {
		side = pt.M
	}
	limMsg[2].Number = float64(side)
	tellRefusal(root, tiles, force, msgOverLimit, msgOverLimitStand, 2,
		"over the port limit")
}

// limBuf is everything a message needs, package level so that telling a player
// their balancer is full allocates nothing. `OfArray` is variadic and would
// allocate a slice per call; a LocalisedString is an array of a key and its
// parameters, so the array is built once here and only the numbers move.
var (
	limMsg   [3]fkapi.Value
	limPos   fkapi.MapPosition
	limText  fkapi.LuaPlayerCreateLocalFlyingTextArgs
	limSound fkapi.PlaySoundSpecification
	// Vanilla's cannot-build red, near enough. The default is white, which
	// reads as information rather than refusal; the 2026-08-05 interactive
	// check expected "red flying text" because that is what the base game has
	// taught everybody a refusal looks like.
	limColR, limColG, limColB float32 = 1.0, 0.25, 0.25
	limColor                          = fkapi.Color{R: &limColR, G: &limColG, B: &limColB}
)

func init() {
	limMsg[0] = fkapi.OfString(msgOverLimit)
	limMsg[1] = fkapi.OfNumber(0)
	limMsg[2] = fkapi.OfNumber(0)
	limText.Position = &limPos
	limText.Color = &limColor
	limSound.Path = soundCannotBuild
}

// tellRefusal puts a refusal in front of somebody.
//
// Three outcomes, and which one happens is decided by whether a PLAYER built
// the thing that broke the machine:
//
//	a player we can still resolve  ->  flying text at the balancer, the standard
//	                                   cannot-build sound, and the piece back
//	a player who has since left    ->  the log line above and nothing else, which
//	                                   is the ordinary case rather than an error
//	a robot or a script            ->  a force-wide message; there is nobody to
//	                                   hand a belt to at flush time, so it stands
//
// `game.get_player` is called FRESH here and never held: the build arrived one
// tick ago and a `LuaPlayer` does not survive that gap. This is the miner's
// pocket's own pattern (carry.go) and for the same reason.
//
// NOTHING BELOW DEPENDS ON WHICH BOUND BROKE, which is why it takes the two
// sentences and their parameter count rather than a shape: the caller has
// already written its numbers into limMsg[1..] and says how many of them there
// are, and `why` is the clause the force-wide log line names the rule with. The
// port limit and the one-belt-per-part rule are the two callers today.
func tellRefusal(root uint32, tiles []key, force uint32,
	msgKey, standKey string, nparam int, why string) {
	x0, y0, x1, y1 := clusterBox(tiles)
	limPos.X = float64(x0+x1)/2 + 0.5
	limPos.Y = float64(y0+y1)/2 + 0.5

	// The first note in EVENT ORDER whose entity stands on or beside this
	// cluster. Event order is deterministic, so two players building into one
	// refusal in the same tick resolve to the same one on every client -- the
	// same rule `carry.Claims.BeneficiaryFor` follows for a mine.
	told := uint32(0)
	surf := uint32(0)
	if len(tiles) > 0 {
		surf = tiles[0].s
	}
	for i := range buildNotes {
		n := buildNotes[i]
		if n.s != surf || !besideCluster(n, tiles) {
			continue
		}
		if told == 0 {
			told = n.player
			// The text goes AT THE REFUSED PIECE, not at the box centre set
			// above. The centre reads as "at the balancer" and is where the
			// first cut put it -- and on the one machine tall enough for the
			// two to differ it is wrong: a player laying the sixty-fifth belt
			// at the top of a 32-part column got the sound and never saw the
			// text, because it spawned seventeen tiles south of their screen
			// (field report, 2026-08-05). The piece the player just placed is
			// the one tile their eyes are guaranteed to be on.
			limPos.X = float64(n.x) + 0.5
			limPos.Y = float64(n.y) + 0.5
		}
		// EVERY addition this tick made to this cluster goes back, not only the
		// one that happened to tip it over. Nothing here can tell which of a
		// pasted handful was the sixty-fifth edge, and none of them is connected
		// to anything -- the network was never rebuilt -- so handing them all
		// back settles it in one step. The alternative converges too, one belt
		// and one flying text per tick, which is worse to be on the end of.
		limPending = append(limPending, n)
	}

	if told != 0 {
		limMsg[0] = fkapi.OfString(msgKey)
		msg := fkapi.Value{Tag: fkapi.TagArray, Array: limMsg[:1+nparam]}
		o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(told)))
		if err != nil || o == nil {
			// Left between the build and the flush. Ordinary, not an error --
			// and the revert below will find no player either and leave the
			// piece standing, which is the robot outcome by another route.
			return
		}
		p := fkapi.LuaPlayer{Object: *o}
		limText.Text = msg
		_ = p.CreateLocalFlyingText(limText)
		_ = p.PlaySound(limSound)
		return
	}

	// Nobody to hand anything back to: a robot revived a ghost, or a script
	// built it. The belt stands where it is, unconnected, and the force is told
	// why -- a different sentence, because "you got it back" would be a lie.
	limMsg[0] = fkapi.OfString(standKey)
	msg := fkapi.Value{Tag: fkapi.TagArray, Array: limMsg[:1+nparam]}
	f, ok := forceOfCluster(tiles, force)
	if !ok {
		return
	}
	err := f.Print(msg, nil)
	// THE ONE HALF OF THE FEEDBACK A HEADLESS RUN CAN REACH, and therefore the
	// one half worth a line. `force.print` writes to the game's chat, which no
	// script can read back and which `--benchmark` does not put in the log -- so
	// without this the `edge` suite could say the refusal happened and nothing
	// at all about whether anybody was told. The two things that could realistically
	// fail here are resolving a LuaForce from a force INDEX (the registry keeps
	// no handle, so it comes off a part standing on the cluster) and the
	// LocalisedString crossing the boundary; both are behind this line.
	if verboseLog {
		logStart("told force ")
		logU(force)
		logS(" that cluster ")
		logU(root)
		logS(" is ")
		logS(why)
		logS("; nobody built it, so the piece stands")
		if err != nil {
			logS(" -- print FAILED")
		}
		logEnd()
	}
}

// besideCluster is the same question the neighbour gate asks, asked of a note:
// is any tile of this cluster within two tiles of it?
//
// Two rather than one for the reason nearCluster gives -- a splitter is two
// tiles wide and reports the boundary between its halves -- and over the
// cluster's own tile list rather than over `index`, because the note has to be
// attributed to THIS cluster and not merely to some cluster.
func besideCluster(n buildNote, tiles []key) bool {
	for i := range tiles {
		k := tiles[i]
		dx, dy := k.x-n.x, k.y-n.y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx <= 2 && dy <= 2 {
			return true
		}
	}
	return false
}

// forceOfCluster gets a LuaForce for a cluster whose force INDEX is all the
// registry keeps.
//
// There is no `game.forces[i]` that is cheaper than this: the dictionary read
// materialises every force on the map, and this mod holds no force handle
// anywhere. A part of the cluster is standing on `tiles[0]` by construction, so
// one find and one attribute read produce the object -- on a path that runs
// once per distinct refusal and never otherwise.
func forceOfCluster(tiles []key, force uint32) (fkapi.LuaForce, bool) {
	if len(tiles) == 0 {
		return fkapi.LuaForce{}, false
	}
	surf, ok := surfaceByIndex(tiles[0].s)
	if !ok {
		return fkapi.LuaForce{}, false
	}
	// `findOnTile` rather than `find_entity`: a part at any quality but normal
	// is invisible to a bare-name `find_entity` (findpart.go), and this lookup
	// failing quietly is a refusal delivered to NOBODY -- the balancer is
	// protected either way, and the player watches their belt do nothing with
	// no message at all.
	o, found, ferr := findOnTile(surf, PartName, tiles[0].x, tiles[0].y)
	if ferr != nil || !found {
		return fkapi.LuaForce{}, false
	}
	f, err := fkapi.LuaEntity{Object: o}.Force()
	if err != nil {
		return fkapi.LuaForce{}, false
	}
	lf := fkapi.LuaForce{Object: f}
	// The cluster's force index is registry state and the entity's is the
	// world's; they agree unless something has moved between the classification
	// pass and here, in which case there is no message worth sending.
	if i, err := lf.Index(); err != nil || i != force {
		return fkapi.LuaForce{}, false
	}
	return lf, true
}

var limPartPos fkapi.MapPosition

// ---------------------------------------------------------------------------
// The merge -- refusing in front of the TEARDOWNS as well as in front of the
// rebuild
// ---------------------------------------------------------------------------
//
// Piece 1 above moved the refusal in front of `teardownForRebuild`, which is
// the teardown compile() does to ITSELF. It is the whole answer for an EDGE
// edit and it is no answer at all for a MERGE, because a merge's teardowns
// belong to somebody else and have already happened:
//
//	AddPart marks BOTH predecessors' roots dead and the merged root live, so
//	flushDead has demolished two working balancers before flushLive so much as
//	looks at the cluster they became.
//
// The items were conserved -- the carry transaction settled them, unclaimed,
// onto the ground -- and the revert still handed the bridging part back, so
// both balancers returned on the following tick EMPTY, with their contents in a
// heap on the floor between them. That is the same defect piece 1 fixed, one
// event shape over.
//
// WHAT SPAREOVERLIMITMERGES DOES. It runs at the top of `flushDead`, which is
// the one choke point every teardown in this guest goes through, and it asks
// the same question `overLimitShape` asks -- but of the cluster the flush is
// ABOUT to build rather than of the one it has just demolished. When the answer
// is yes it pulls the merge's predecessors back off `deadRoots`, so their
// networks are never touched, and remembers them in `stranded`. compile() then
// reaches the merged cluster, refuses it through the ordinary path in piece 1,
// and returns without a teardown of its own.
//
// WHAT IT COSTS ON THE HOT PATH, WHICH IS NOTHING, AND WHY. Classifying every
// merge product's edges would be a second `find_entities_filtered` per boundary
// tile on every flush that merged anything -- which is the reason this was
// written down as uncovered rather than done. It is avoided by an arithmetic
// bound that needs no host call at all: A CLUSTER OF C PARTS HAS AT MOST 4C
// EXTERIOR SIDES AND THEREFORE AT MOST 4C EDGES, so `4*csize[r] <= MaxPorts`
// is a proof that no classification could find enough of them. Sixteen parts is
// the largest cluster that can be proved safe, and every balancer any suite in
// this repo merges is smaller than that -- the `mar` suite's merge leg is five
// parts -- so the pass is a handful of integer comparisons and a `find` per
// queued root, with no host call and no allocation, on every flush that is not
// about a balancer of seventeen parts or more.
//
// The bound is CONSERVATIVE, and being wrong about it costs exactly the old
// behaviour: a cluster whose recorded shape has been overtaken by a world the
// guest was never told about (the failure envelope) is classified when it need
// not be, or -- if a mod added belts with no event at all -- is not classified
// when it should be, and the merge falls through to the refusal in compile()
// where it always used to be. Nothing about that path changed.
//
// WHAT THE REGISTRY BELIEVES WHILE A REFUSED MERGE STANDS, and it is the one
// state in this guest where `nets` holds a network under a key that is no
// longer a root:
//
//	the registry has ONE cluster, rooted wherever union-find put it, and TWO
//	standing networks -- the merged root's own (if the survivor was one of the
//	predecessors) and one per stranded predecessor.
//
// That is reported honestly rather than left to be discovered: `strandedNets`
// tells `auditAll` how many networks a refused merge left under non-root keys,
// so the audit counts them in `nets=` and reports the merged cluster as
// `drift=1` -- a cluster whose edge list has moved past what the mod can build
// and which knows it. `unbuilt` stays 0, because there is nothing unbuilt; the
// machine a player is looking at is still running. A `drift=0 unbuilt=1` there
// is the signature of the defect this section is about.
//
// AND IT IS STABLE. The state persists with no further work: a later flush that
// queues the merged root finds no dead root absorbed into it, so `compile`
// reaches the refusal directly and the feedback gate keeps it quiet. Nothing
// oscillates, and nothing is torn down or rebuilt until one of three things
// happens --
//
//	the edge list SHRINKS back under the limit (a belt mined, a rotation): the
//	pass reclaims, `markDead`s every stranded predecessor so flushDead brings
//	them down properly, and the merged cluster builds and claims both their
//	carry pools by geometry, exactly as an ordinary merge does. THE MERGED
//	ROOT'S OWN NETWORK IS STRANDED TOO, FOR THAT REASON: a cluster that tears
//	itself down in compile() opens an `owned` pool and then declines every
//	geometric one (carry.go, takeCarry), so the other predecessor's items would
//	spill. Stranded, it comes down in flushDead like the other and the rebuild
//	claims both;
//
//	the BRIDGING PART IS MINED, which is what the revert does and what a player
//	does: the cluster splits back into what it was, union-find re-roots each
//	component at its smallest node id -- which are the original roots, because
//	that is how they were chosen in the first place -- and each half's
//	fingerprint matches the netInfo it never lost, so BOTH compiles are a SKIP.
//	Nothing is torn down and nothing is rebuilt at any point in the whole
//	sequence, which is the same property piece 1 has for an edge edit;
//
//	the STRANDED NODE IS FREED -- the cluster dissolved in one event, a surface
//	went. There is then no cluster left that could ever reclaim that network, so
//	`sweepStranded` brings it down on the next flush, before the node id can be
//	reused under it.

var (
	// stranded is the roots whose networks a refused merge left standing.
	//
	// They are NOT roots any more -- that is the whole point of the state -- so
	// nothing that walks `liveRootList` will ever find them, and this slice is
	// the only thing that knows they exist. Empty in every save that never
	// merges two balancers past sixty-four ports, high-water like the recompile
	// queues, and in EVENT ORDER, which is `deadRoots`' order, which the engine
	// makes deterministic.
	stranded []uint32

	// The candidate list for one pass, and whether each candidate had a dead
	// root ABSORBED into it this flush. Only a candidate that did may have
	// teardowns taken away from it: everything else -- a surface being deleted,
	// an area clone reconciling, a network of ours destroyed from outside --
	// queues a root that is still a root, and those teardowns are not this
	// pass's to cancel.
	limCands     []uint32
	limCandMerge []bool
)

// spareOverLimitMerges decides, before any teardown runs, which of them belong
// to a merge this mod cannot honour.
//
// Called from the top of `flushDead` and from nowhere else.
func spareOverLimitMerges() {
	if len(deadRoots) == 0 && len(stranded) == 0 {
		return
	}
	limCands = limCands[:0]
	limCandMerge = limCandMerge[:0]

	// A dead root that is NO LONGER A ROOT has been absorbed by something, and
	// what absorbed it is the cluster this flush is about to build. That test is
	// exact for the shape this pass is about and rejects every other queue
	// filler for free: a shrink, a split, an audit repair, a clone reconcile and
	// a surface deletion all queue roots that are still roots.
	for i := range deadRoots {
		d := deadRoots[i]
		if int(d) >= len(alive) || !alive[d] {
			continue
		}
		if r := find(d); r != d && alive[r] {
			addLimCand(r, true)
		}
	}
	// And whatever now owns a network a previous flush stranded has to decide
	// again, because the edit that makes a refused merge buildable does not
	// have to be a merge -- it is usually a belt being mined.
	for i := range stranded {
		s := stranded[i]
		if int(s) >= len(alive) || !alive[s] {
			continue
		}
		addLimCand(find(s), false)
	}

	for i := range limCands {
		r := limCands[i]
		if overLimitMerge(r, limCandMerge[i]) {
			if limCandMerge[i] {
				spareMerge(r)
			}
			continue
		}
		reclaimStranded(r)
	}
	sweepStranded()
}

func addLimCand(r uint32, merge bool) {
	for i := range limCands {
		if limCands[i] == r {
			if merge {
				limCandMerge[i] = true
			}
			return
		}
	}
	limCands = append(limCands, r)
	limCandMerge = append(limCandMerge, merge)
}

// overLimitMerge asks, of a cluster that has not been compiled yet, whether
// the compiler is going to refuse it -- for EITHER bound, because a merge past
// either one demolishes the same two working balancers.
//
// `bridged` says whether this candidate had a dead root absorbed into it THIS
// flush (a real merge) or is a candidate only because a previous refusal left
// networks stranded under it. The distinction matters to the second bound and
// not to the first.
//
// IT MUST BE EXACT IN THE "YES" DIRECTION. Sparing a merge that then compiles
// successfully leaves both predecessors' networks standing beside the new one,
// which is strictly worse than the defect this pass exists to fix. Being
// conservative the other way costs the old behaviour and nothing else.
func overLimitMerge(r uint32, bridged bool) bool {
	if int(r) >= len(csize) || !alive[r] {
		return false
	}
	// THE TWO FREE TESTS FIRST, and neither makes a host call.
	//
	// The port bound: AT MOST FOUR EDGES PER PART, so `4C <= MaxPorts` is a
	// proof and not a heuristic. It is what keeps the whole pass off the hot
	// path -- see the header.
	ports := int(csize[r])*4 > plan.MaxPorts
	// The one-belt-per-part bound has no such arithmetic: any two-part merge
	// could break it. What stands in for it is the BRIDGING-TILE THEOREM
	// (sedge.go) -- adding a part can only take edges away from the tiles that
	// were already there, so the only tile that can newly violate is the new
	// part's own.
	//
	// A candidate with stranded networks under it is the one case the theorem
	// cannot serve: a stranded predecessor was REFUSED rather than compiled, so
	// "the predecessors were valid single-edge" is not true of it and the full
	// classification is the only exact answer. `stranded` is empty in every save
	// that has never refused a merge, so the test costs a length check.
	sedge := !multiEdgeAllowed()
	if sedge && bridged && len(stranded) == 0 {
		if bridgingTilesOverloaded(r) {
			return true
		}
		sedge = false
	}
	if !ports && !sedge {
		return false
	}
	tiles := collectCluster(r)
	if len(tiles) == 0 {
		return false
	}
	surf, ok := surfaceByIndex(tiles[0].s)
	if !ok {
		return false
	}
	edges := classifyEdges(surf, tiles, pforce[r])
	if ports {
		if _, over := overLimitShape(edges); over {
			return true
		}
	}
	if sedge {
		if _, over := multiEdgeShape(); over {
			return true
		}
	}
	return false
}

// spareMerge takes every teardown that belongs to the merge into `r` back off
// the queue.
//
// `d == r` is included deliberately: the survivor's own network must be
// stranded rather than left for compile() to tear down, or the reclaim later on
// would open an `owned` pool and decline the other predecessor's items. The
// header says it at more length.
func spareMerge(r uint32) {
	n, spared := 0, uint32(0)
	for i := range deadRoots {
		d := deadRoots[i]
		if int(d) < len(alive) && alive[d] && find(d) == r {
			if _, has := nets[d]; has {
				addStranded(d)
				spared++
			}
			continue
		}
		deadRoots[n] = d
		n++
	}
	deadRoots = deadRoots[:n]
	if spared != 0 && verboseLog {
		// THE LINE NAMES NO BOUND, because there are two of them now: a merge is
		// spared for the port limit or for the one-belt-per-part rule, and which
		// one it was is on the `alert:` line the refusal itself writes a moment
		// later. It used to say "past the port limit" and that was the only
		// bound there was.
		logStart("cluster ")
		logU(r)
		logS(" would merge into a cluster this mod cannot build; left ")
		logU(spared)
		logS(" standing network(s) alone instead of demolishing them")
		logEnd()
	}
}

func addStranded(d uint32) {
	for i := range stranded {
		if stranded[i] == d {
			return
		}
	}
	stranded = append(stranded, d)
}

// reclaimStranded gives every network a refused merge into `r` spared back to
// the ordinary machinery, now that `r` fits again.
//
// TWO OUTCOMES, and which one happens is decided by whether the cluster is
// STILL MERGED:
//
//	SEVERAL stranded networks under `r` -- the merge stands and its edge list
//	has come back under the limit. Every one of them, INCLUDING `r`'s own, is
//	queued dead so flushDead brings it down: their pools are then unowned,
//	`claimCarry` counts `r` against all of them, and the one network `r` builds
//	takes the lot back. Letting compile() tear `r`'s own down instead would open
//	an `owned` pool, and an owning cluster declines every geometric one
//	(carry.go, takeCarry) -- so the other predecessor's items would spill.
//
//	`r`'s OWN AND NOTHING ELSE -- the bridging part was mined and the cluster is
//	back to being what it was. It is simply released, with no teardown at all:
//	its fingerprint is the one its netInfo never stopped holding, so compile()
//	SKIPS it. Measured: the un-merge costs nothing, and the half that was never
//	the merged root is not looked at twice.
func reclaimStranded(r uint32) {
	if len(stranded) == 0 {
		return
	}
	others := strandedNets(r)
	n := 0
	for i := range stranded {
		s := stranded[i]
		if int(s) < len(alive) && alive[s] && find(s) == r {
			if s != r || others != 0 {
				markDead(s)
			}
			continue
		}
		stranded[n] = s
		n++
	}
	stranded = stranded[:n]
}

// sweepStranded is the one that keeps the state from outliving its cluster.
//
// A stranded network whose node id has been FREED can never be reclaimed -- no
// cluster resolves to it any more -- and the id is on the free list, so the next
// part placed anywhere could be handed it and compile() would then read another
// network's box and slot out of `nets`. It comes down here instead, on the first
// flush after the node went, which is a removal and therefore spills: correct,
// because a dissolve is a removal.
//
// An entry whose network came down some other way (an area clone, a surface
// going) is simply dropped.
func sweepStranded() {
	if len(stranded) == 0 {
		return
	}
	n := 0
	for i := range stranded {
		s := stranded[i]
		if _, has := nets[s]; !has {
			continue
		}
		if int(s) < len(alive) && alive[s] {
			stranded[n] = s
			n++
			continue
		}
		markDead(s)
	}
	stranded = stranded[:n]
}

// strandedNets reports how many standing networks a refused merge into `root`
// has left under keys that are no longer roots.
//
// `root`'s own is excluded: that one is `nets[root]` and the audit counts it
// through the ordinary branch. Nothing here depends on the slice's order -- it
// is a count -- so no iteration order reaches anything host-visible.
func strandedNets(root uint32) uint32 {
	if len(stranded) == 0 {
		return 0
	}
	n := uint32(0)
	for i := range stranded {
		s := stranded[i]
		if s == root || int(s) >= len(alive) || !alive[s] {
			continue
		}
		if _, has := nets[s]; !has {
			continue
		}
		if find(s) == root {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// The revert
// ---------------------------------------------------------------------------

// revertOverLimit hands the refused additions back to the players who made
// them.
//
// WHERE IT RUNS IS THE WHOLE DESIGN, and it is called from exactly one place:
// `flush()`, AFTER `endCarry()` -- so after every teardown, every build and the
// settling of the carry transaction. It may not run any earlier, for three
// reasons that are each sufficient:
//
//  1. `mine_entity` DISPATCHES `on_player_mined_entity` SYNCHRONOUSLY. That
//     re-enters this guest through fk_on_event before mine_entity returns, and
//     what it re-enters is `onPart` or `onNeighbour` -- which free registry
//     nodes, re-root surviving components and append to `deadRoots`/`liveRoots`.
//     Running that inside `flushLive`'s loop would be mutating the very queue
//     being iterated, and freeing node ids out from under roots the loop has not
//     reached yet.
//  2. THE COMPILE BUFFERS ARE PACKAGE LEVEL AND NOT RE-ENTRANT -- `tileBuf`,
//     `edgeBuf`, `opBuf`, `entBuf`. Nothing in the nested handler touches them
//     today (`onNeighbour` is 25 map probes and no host call, and the
//     fast-replace check beside it runs on the APPEARANCE path, which a mine is
//     not -- see fastreplace.go), but "today" is not a property to build a
//     reentrancy argument on. After the drain there is no drain to re-enter.
//  3. THE CARRY TRANSACTION MUST BE CLOSED. A mine records a claim
//     (`carry.Claims`), and a claim is consumed by the settle at the END of a
//     flush. One recorded before `endCarry` would be settled against THIS
//     flush's pools -- pools belonging to networks the miner had nothing to do
//     with. Recorded after, it lives exactly one flush and is answered by the
//     teardown the revert itself provoked, which is the correct one.
//
// What the mine leaves behind is an ordinary edit: the queues it filled are
// drained by the flush on the following tick, the cluster's fingerprint has
// gone back to what its netInfo already says, and the compile is a SKIP. The
// standing network is never torn down at any point in the whole sequence.
func revertOverLimit() {
	if len(limPending) == 0 {
		return
	}
	// By index over a saved length, and by VALUE: nothing below can append to
	// limPending (only refuseOverLimit does, and only from inside the drain that
	// has already finished), but a mine that somehow did would otherwise
	// invalidate a pointer into the backing array mid-loop.
	n := len(limPending)
	for i := 0; i < n; i++ {
		revertOne(limPending[i])
	}
	limPending = limPending[:0]
}

func revertOne(n buildNote) {
	// The registry cross-check, and it costs nothing: a part stands ON a
	// registered tile and a belt at a cluster's edge stands one tile OUTSIDE
	// the cluster's box by construction. If the world has moved on -- the part
	// was mined by somebody else in the meantime, a belt was replaced by a part
	// -- the note describes something that is not there any more and the safe
	// answer is to leave it alone.
	_, registered := index[key{n.s, n.x, n.y}]
	if registered != n.isPart {
		return
	}
	surf, ok := surfaceByIndex(n.s)
	if !ok {
		return
	}
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(n.player)))
	if err != nil || o == nil {
		return // left between the build and the flush; the piece stands
	}
	p := fkapi.LuaPlayer{Object: *o}

	var ent fkapi.Object
	limPartPos.X = float64(n.x) + 0.5
	limPartPos.Y = float64(n.y) + 0.5
	if n.isPart {
		// `findOnTile`, not `find_entity`: a player can build an over-limit
		// part at any quality their cursor holds, and a bare-name lookup
		// (findpart.go) would leave a non-normal one standing instead of
		// handing it back.
		e, found, ferr := findOnTile(surf, PartName, n.x, n.y)
		if ferr != nil || !found {
			return
		}
		ent = e
	} else {
		// findByPos filters by belt-connectable TYPE and by FORCE, both in C++,
		// so what comes back is a belt of the right force on that tile and
		// nothing else. THE FORCE IS SET HERE rather than inherited from the
		// classification pass: that pass ran during the drain and left
		// `forceFilter` naming whichever cluster it happened to finish on, which
		// is not necessarily this note's.
		searchPos.X = limPartPos.X
		searchPos.Y = limPartPos.Y
		forceFilter = fkapi.OfNumber(float64(n.force))
		ents, err := surf.FindEntitiesFiltered(findByPos)
		if err != nil || len(ents) == 0 {
			return
		}
		ent = ents[0]
	}

	// `force = nil`, which is the vanilla rule: a player whose inventory cannot
	// take it does not get to mine it, and the piece stays standing rather than
	// evaporating. The bool it returns says which happened.
	took, err := p.MineEntity(ent, nil)
	if err != nil {
		return
	}
	if !took {
		// An alert rather than a verbose line, because the outcome is now the
		// robot one -- an unconnected piece standing in the world -- and the
		// player has been told it would not fit but not that they still have it.
		logAlertStart("player ")
		logU(n.player)
		logS(" could not be handed back the over-limit piece at ")
		logI(n.x)
		logS(",")
		logI(n.y)
		logS(" -- no room in the inventory; it stays where it is, unconnected")
		logEnd()
		return
	}
	// The same level and the same reason as carry.go's `offered … before the
	// floor`: this is the one piece of evidence a player checking the feature
	// interactively can grep for, and it cannot fire in any headless suite
	// because a headless --create has no players at all.
	if verboseLog {
		logStart("handed the over-limit piece at ")
		logI(n.x)
		logS(",")
		logI(n.y)
		logS(" back to player ")
		logU(n.player)
		logEnd()
	}
}
