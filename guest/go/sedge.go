package main

// ONE BELT PER BALANCER PART -- the rule Factorio 2.1 forces, and the refusal
// that carries it to the player.
//
// WHY THERE IS A RULE AT ALL. Every edge of a cluster is an interface linked
// belt standing ON the cluster's own tile, and a 1x1 part carrying an input on
// one side and an output on another therefore carried TWO of them on one tile.
// That was legal on 2.0 only through the collision-mask loophole spike S1 found
// (prototypes/hidden.lua's header is the long form), and 2.1 closed it: the
// validator now demands every belt-connectable collide with itself, probed
// exhaustively on 2.1.14 with no mask design passing and no runtime bypass.
//
// It is not an oversight upstream might undo. boskid's answer to the interface
// request (forums t=135830) names the invariant the check protects:
// belt-to-belt connections are NOT SAVED -- they are re-derived at load -- and
// one belt-connectable per tile is what makes that re-derivation unambiguous.
// S1's own "never two same-direction inputs on one tile" was the observed
// shadow of exactly that ambiguity. So the port is a RULE change and not an
// interface redesign: at most one edge per cluster tile, everything else about
// classification unchanged. agents/single-edge.md is the whole design.
//
// ---------------------------------------------------------------------------
// THE TWO QUESTIONS, WHICH THE OLD DESIGN CONFLATED
// ---------------------------------------------------------------------------
//
//	CAN the engine stack?   a fact about the Factorio version. The data stage
//	                        emits `not_colliding_with_itself` on 2.0.x and never
//	                        on 2.1.x, and defines the marker prototype
//	                        `bbb-can-stack` in the SAME `if` -- so the guest's
//	                        belief cannot drift from the prototype's actual
//	                        capability, because there is one branch and not two.
//	MAY the compiler use it? a per-save policy, which is phase 2's runtime-global
//	                        setting. See multiEdgeAllowed's seam.
//
// The effective rule is the AND of the two. On 2.1 the marker is absent and the
// rule always enforces; on 2.0 with the policy off, the compiler refuses
// multi-edge exactly as 2.1 does while the prototype capability sits unused --
// which is what makes a 2.0 save that never used it bit-compatible with a fresh
// single-edge world.
//
// ---------------------------------------------------------------------------
// WHAT IT COSTS, WHICH IS NOTHING ON ANY PATH THAT IS NOT ALREADY WALKING
// ---------------------------------------------------------------------------
//
// The per-tile count falls out of `classifyEdges`' own walk: it already visits
// every tile and every side, so counting the sides that answered is one integer
// per tile and no allocation and no host call. The capability is a cached point
// query -- two host calls, once per heap. The merge arm classifies at most the
// tiles a part was ADDED to this tick, and only when a merge would otherwise
// demolish a standing network.

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// StackMarkerName is the prototype that means "this engine's linked belt
// carries `not_colliding_with_itself`". See prototypes/hidden.lua; nothing ever
// places one.
const StackMarkerName = "bbb-can-stack"

// The locale keys, in mod-data/locale/en/better-belt-balancer.cfg -- the same
// pair shape the port limit has, and for the same reason: a player whose piece
// comes back and a robot whose piece stands are told different things.
const (
	msgSingleEdge      = "bbb.single-edge-refused"
	msgSingleEdgeStand = "bbb.single-edge-refused-unconnected"
)

// The engine capability, resolved once per heap.
//
// It is a tri-state and not a bool because the zero value has to mean "not
// asked yet": a fresh heap knows nothing, and a heap that came back from a save
// was written by a session whose engine this one is not required to be.
const (
	edgeModeUnchecked = iota
	edgeModeSingle    // the marker is absent: one belt per part, always
	edgeModeMulti     // the marker is present: this engine can stack
)

var edgeMode uint8

// edgeModeRecheck throws the cached capability away.
//
// Called from the three load-time hooks and from nowhere else -- a new save, a
// rebuilt guest, and the mod set (or the game version) moving -- which are the
// only moments the answer CAN change: prototypes are fixed for the life of a
// session. It is exactly legacy.go's `legacyRecheck` discipline and it is here
// for the same reason.
func edgeModeRecheck() { edgeMode = edgeModeUnchecked }

// multiEdgeAllowed reports whether the compiler may put two edges on one tile.
//
// TWO HOST CALLS, ONCE PER HEAP, and one integer compare on every call after
// that. `prototypes.entity` is a LuaCustomTable, so the raw handle plus its
// index operator is a POINT query -- against the materialising `Entity()`
// attribute, which would build a Go slice of every entity prototype in the game
// to answer a yes/no question. A missing key arrives as TagNil rather than as
// an error. This is `legacyStubPresent` with a different name in it.
//
// PHASE 2'S SEAM IS HERE AND NOWHERE ELSE. The runtime-global setting
// `bbb-multi-edge-parts` is the second half of the AND, so it belongs on the
// return of this function -- and the mode byte it is compared against is the
// reconciliation anchor for the flip handler, which is why the cache above is
// already a tri-state rather than a bool. Nothing else in the guest asks the
// question, so nothing else has to change when it lands.
func multiEdgeAllowed() bool {
	if edgeMode == edgeModeUnchecked {
		edgeMode = edgeModeSingle
		if stackMarkerPresent() {
			edgeMode = edgeModeMulti
		}
	}
	return edgeMode == edgeModeMulti
}

func stackMarkerPresent() bool {
	raw, err := fkapi.Prototypes.EntityRaw()
	if err != nil {
		return false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(StackMarkerName))
	if err != nil {
		return false
	}
	return v.Tag == fkapi.TagObject
}

// ---------------------------------------------------------------------------
// The predicate
// ---------------------------------------------------------------------------

// What the last `classifyEdges` found, per tile. Written by that walk and read
// by the two functions below; package level for the same reason `edgeBuf` is,
// and parallel to it -- a classification produces an edge LIST and this shape
// summary, and both describe the same pass.
//
// `sedgeWorst` is the highest edge count on any tile that carried MORE THAN
// ONE, so it is 0 or at least 2 and never 1; `sedgeTiles` is how many such
// tiles there were. Neither is state a save carries: both are overwritten by
// every classification.
var (
	sedgeWorst uint32
	sedgeTiles uint32
)

// multiEdgeShape is the check compile() makes BESIDE overLimitShape and in
// front of the same teardown.
//
// It mirrors nothing in `plan.Build`, and that is the difference from the port
// limit: a multi-edge cluster is not a planner INPUT problem -- the planner
// would happily emit a network for it, and the engine would then refuse every
// second interface with a silent nil. So this is the only test there is, and
// the backstop for it lives beside overLimitShape's rather than inside plan.
func multiEdgeShape() (worst uint32, over bool) {
	if sedgeWorst < 2 || multiEdgeAllowed() {
		return sedgeWorst, false
	}
	return sedgeWorst, true
}

// logRefusedSingleEdge is the refusal's log line, shared by the ordinary path
// and the silent rebuild path.
//
// `alert:` and NOT `error:`, for the reason limit.go's twin gives: an error in
// this guest means a compile did not produce a network that should have, and
// `test/run.sh` fails a run on one. A balancer built to a rule this Factorio
// does not permit is an expected condition with a defined outcome.
func logRefusedSingleEdge(root uint32, worst, tiles uint32) {
	logAlertStart("cluster ")
	logU(root)
	logS(" has ")
	logPlural(tiles)
	logS(" carrying more than one belt, worst ")
	logU(worst)
	logS("; this Factorio allows one belt per balancer part, so the compile is")
	logS(" refused BEFORE the teardown and the standing network is untouched")
	logEnd()
}

// refuseSingleEdge is compile()'s answer when a cluster asks for two belts on
// one part. The standing network has not been touched and must not be.
//
// Every discipline here is limit.go's, reached through the same three-way
// admission: a refusal issued from inside `rebuildFromWorld` logs and requeues
// and tells nobody (the wake race -- the rebuild judges the world with the
// worst information a refusal will ever have), a repeat of an edge state
// already refused says nothing at all, and anything else logs and speaks.
func refuseSingleEdge(root uint32, fp uint64, worst uint32, tiles []key, force uint32) {
	switch refuseAdmit(root, fp) {
	case refuseSilent:
		return
	case refuseLogOnly:
		logRefusedSingleEdge(root, worst, sedgeTiles)
		return
	}
	logRefusedSingleEdge(root, worst, sedgeTiles)
	limMsg[1].Number = float64(worst)
	tellRefusal(root, tiles, force, msgSingleEdge, msgSingleEdgeStand, 1,
		"past the one-belt-per-part rule")
}

// ---------------------------------------------------------------------------
// The merge, and the theorem that keeps it off the hot path
// ---------------------------------------------------------------------------
//
// A part that BRIDGES two clusters is queued by `AddPart` as two DEAD roots and
// one live one, so the check in compile() is useless on its own: both
// predecessors' networks are down before flushLive discovers that what they
// became cannot be built. limit.go's `spareOverLimitMerges` is the pass that
// takes those teardowns back off the queue, and it needs the same answer for
// this rule that it already gets for the port limit.
//
// THE PORT LIMIT GETS ITS ANSWER FROM ARITHMETIC -- a cluster of C parts has at
// most 4C edges, so `4C <= MaxPorts` is a proof no classification could find
// enough of them. There is no such bound here: any two-part merge could break
// the rule. What replaces it is a theorem about what a merge can CHANGE.
//
//	A tile's edge count is over its EXTERIOR sides. Adding a part to the world
//	makes one side of each of its neighbours INTERIOR, which can only take edges
//	away. So the only tile whose count can have gone UP is the new part's own --
//	and this guest has the tick's new parts in hand already.
//
// SOUND, NOT COMPLETE, AND SOUNDNESS IS THE REQUIREMENT. Sparing a merge that
// then COMPILES SUCCESSFULLY would leave both predecessors' networks standing
// beside the new one -- three networks over one cluster, two of them holding
// items nothing will ever come back for. So a "yes" here must mean a refusal,
// and a bridging tile carrying two edges is one: the full classification would
// see the same two on the same tile. A "no" that is wrong costs the OLD
// behaviour -- the predecessors are torn down for a refusal, items conserved
// and spilled -- and never a network left standing where it should not be.
//
// THE DESIGN'S "ALSO SPARE WHEN EITHER PREDECESSOR IS ALREADY REFUSED" IS NOT
// TAKEN, AND THE REASON IS SOUNDNESS. It reads as free conservatism and it is
// the unsound direction: a part is fast-replaceable onto a belt
// (fastreplace.go), so the bridging part can be placed ON one of the two belts
// that made a predecessor's tile multi-edge -- taking that tile to ONE edge and
// making the merged cluster perfectly buildable. Spare it and the disaster
// above is exactly what happens. The same argument retires it for the port
// bound, where a bridging part laid over a belt likewise removes an edge.

// bridgingTilesOverloaded asks the theorem above of one merge candidate.
//
// Costs NO HOST CALL at all unless a merge would really strand something, and
// then at most four per tile a part was added to this tick -- fewer in
// practice, because a bridging part has at least two INTERIOR sides by
// construction and an interior side is a map probe rather than a query.
func bridgingTilesOverloaded(r uint32) bool {
	if len(addedTiles) == 0 || multiEdgeAllowed() {
		return false
	}
	// NOTHING TO STRAND, NOTHING TO DECIDE. `spareMerge` only spares teardowns
	// of predecessors that HAVE a standing network, so a merge of clusters that
	// have none -- every merge inside a blueprint paste, and every merge
	// `rebuildFromWorld` makes while it is registering a world onto a fresh
	// heap -- has nothing this pass could protect. Answering "no" for those
	// costs nothing and saves the classification.
	if !mergeWouldStrand(r) {
		return false
	}
	force := pforce[r]
	for i := range addedTiles {
		k := addedTiles[i]
		id, ok := index[k]
		// Added and then mined again inside the same tick (the edge suite has
		// that leg), or absorbed into a different cluster: not this candidate's.
		if !ok || find(id) != r {
			continue
		}
		surf, ok := surfaceByIndex(k.s)
		if !ok {
			continue
		}
		if edgesOnTile(surf, k, force) >= 2 {
			return true
		}
	}
	return false
}

// mergeWouldStrand reports whether any predecessor queued dead into `r` has a
// standing network. Pure guest memory: a `find` and a point query per queued
// root.
func mergeWouldStrand(r uint32) bool {
	for i := range deadRoots {
		d := deadRoots[i]
		if int(d) >= len(alive) || !alive[d] || find(d) != r {
			continue
		}
		if _, has := nets[d]; has {
			return true
		}
	}
	return false
}

// edgesOnTile is `classifyEdges` restricted to one tile.
//
// The interior test is the same one classifyEdges makes and is deliberately
// force-blind for the same reason: a part of ANOTHER force standing beside this
// one still occupies the side, so no interface can be placed against it.
func edgesOnTile(surf fkapi.LuaSurface, k key, force uint32) int {
	forceFilter = fkapi.OfNumber(float64(force))
	n := 0
	for d := 0; d < len(dirs); d++ {
		nx, ny := k.x+dirs[d][0], k.y+dirs[d][1]
		if _, ok := index[key{k.s, nx, ny}]; ok {
			continue
		}
		if _, found := classifySide(surf, nx, ny, dirOf[d]); found {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// The tick's new parts
// ---------------------------------------------------------------------------

// addedTiles is every tile a part was registered on during this tick, IN EVENT
// ORDER, truncated by every flush. High-water like the recompile queues, so a
// blueprint paste sizes it once and every paste after that reuses the backing
// array.
//
// It exists for the theorem above and for nothing else, which is why both of
// the cases that would make it big skip it: a rebuild-from-world registers a
// whole save's parts and has no standing networks to protect while it does, and
// an engine that can stack never asks the question.
var addedTiles []key

func noteAddedPart(k key) {
	if rebuildingFromWorld || multiEdgeAllowed() {
		return
	}
	addedTiles = append(addedTiles, k)
}

func forgetAddedParts() { addedTiles = addedTiles[:0] }

// ---------------------------------------------------------------------------
// The unreachable backstop
// ---------------------------------------------------------------------------

// singleEdgeBackstop is `multiEdgeShape` asked a second time, after the plan
// has been built and the teardown has already happened.
//
// UNREACHABLE, and kept as an `error:` precisely because of that -- it is the
// twin of the `!fits` branch beside it, which mirrors plan.Build's own test the
// way this mirrors the classification's. Getting here means the check in front
// of the teardown did not run or did not agree, and by then a working network
// really has been demolished for nothing. `test/run.sh` fails any run that
// produces one of these.
func singleEdgeBackstop(root uint32) bool {
	worst, over := multiEdgeShape()
	if !over {
		return false
	}
	logErrStart("cluster ")
	logU(root)
	logS(" has a part carrying ")
	logU(worst)
	logS(" belts and this Factorio allows one; not compiled, and the early")
	logS(" refusal did not catch it")
	logEnd()
	return true
}
