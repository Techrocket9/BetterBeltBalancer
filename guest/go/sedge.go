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

import (
	"unsafe"

	"github.com/Techrocket9/BetterBeltBalancer/guest/go/edgemode"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
)

// StackMarkerName is the prototype that means "this engine's linked belt
// carries `not_colliding_with_itself`". See prototypes/hidden.lua; nothing ever
// places one.
const StackMarkerName = "bbb-can-stack"

// MultiEdgeSetting is the runtime-global bool that is the POLICY half of the
// rule. mod-data/settings.lua defines it on 2.0.x and never on 2.1.x, so on the
// engine this port is about it is not merely off -- it does not exist, and a
// read of it is nil rather than an error.
//
// Runtime-global rather than startup because the grandfather pass below WRITES
// it, and a script can write `settings.global` and can never write a startup
// setting (measured: `settings.startup` answers `LuaCustomTable is read only`).
// settings.lua's header is the rest of the argument.
const MultiEdgeSetting = "bbb-multi-edge-parts"

// A ModSetting is a table with one entry, so the policy is read as
// `settings.global[name].value` and written as `settings.global[name] =
// {value = v}` -- the whole table, never the field. There is no index-assign for
// a field of a value inside a custom table and there does not need to be.
const settingValueKey = "value"

// The locale keys, in mod-data/locale/en/better-belt-balancer.cfg -- the same
// pair shape the port limit has, and for the same reason: a player whose piece
// comes back and a robot whose piece stands are told different things.
const (
	msgSingleEdge      = "bbb.single-edge-refused"
	msgSingleEdgeStand = "bbb.single-edge-refused-unconnected"
)

// The two SUMMARY keys, which are a different act from the pair above: those
// answer a piece somebody just placed, these answer a whole SAVE that was built
// to a rule that has since changed. One is spoken per affected force, once, and
// neither mentions a hand-back because nothing was placed.
const (
	msgMigrated      = "bbb.single-edge-migrated"
	msgGrandfathered = "bbb.single-edge-grandfathered"
)

// ---------------------------------------------------------------------------
// The two halves of the rule, and the anchor that remembers which one won
// ---------------------------------------------------------------------------

// The engine CAPABILITY, resolved once per heap.
//
// Tri-state and not a bool because the zero value has to mean "not asked yet": a
// fresh heap knows nothing, and a heap that came back from a save was written by
// a session whose engine this one is not required to be -- which is precisely
// the 2.0-save-opened-on-2.1 case the migration below exists for.
const (
	edgeCapUnchecked = iota
	edgeCapSingle    // the marker is absent: this engine cannot stack, ever
	edgeCapMulti     // the marker is present: this engine can
)

// The EFFECTIVE rule -- the AND of the capability and the policy -- cached the
// same way and for the same reason. Same three values, kept in their own
// variable because the two questions have different answers and two callers
// need the capability alone.
const (
	edgeModeUnchecked = iota
	edgeModeSingle    // one belt per part
	edgeModeMulti     // two are allowed here
)

var (
	edgeCap  uint8
	edgeMode uint8

	// edgeAnchor is GUEST STATE AND THEREFORE SAVE STATE: the mode the registry
	// was last reconciled under.
	//
	// It is not a cache of the setting, and the difference is the whole reason it
	// exists. The setting is what the player wants; this is what the networks
	// STANDING IN THE WORLD were built to. A flip is a change only when the two
	// disagree, which is what makes the handler idempotent against Factorio
	// raising `on_runtime_mod_setting_changed` for a write of the value already
	// there (measured), and against the grandfather pass's own write.
	edgeAnchor edgemode.Mode
)

// edgeModeRecheck throws the cached answers away.
//
// Called from the three load-time hooks -- a new save, a rebuilt guest, and the
// mod set or the game version moving -- and from the setting-changed handler,
// which is the fourth and only other moment either answer CAN move: prototypes
// are fixed for the life of a session, and a runtime-global setting changes only
// through that event or through a load. It is exactly legacy.go's
// `legacyRecheck` discipline.
func edgeModeRecheck() { edgeCap, edgeMode = edgeCapUnchecked, edgeModeUnchecked }

// stackCapable reports whether this Factorio can put two belt-connectables on
// one tile at all.
//
// TWO HOST CALLS, ONCE PER HEAP, and one integer compare after that.
// `prototypes.entity` is a LuaCustomTable, so the raw handle plus its index
// operator is a POINT query -- against the materialising `Entity()` attribute,
// which would build a Go slice of every entity prototype in the game to answer a
// yes/no question. A missing key arrives as TagNil rather than as an error. This
// is `legacyStubPresent` with a different name in it.
func stackCapable() bool {
	if edgeCap == edgeCapUnchecked {
		edgeCap = edgeCapSingle
		if stackMarkerPresent() {
			edgeCap = edgeCapMulti
		}
	}
	return edgeCap == edgeCapMulti
}

// multiEdgeAllowed reports whether the compiler may put two edges on one tile.
//
// THE AND, AND THE MARKER IS THE OUTER TERM -- so on 2.1 the setting is never
// read at all, which is exactly right because it is not defined there. The fold
// itself is guest/go/edgemode, where all eighteen of its states are proved under
// `go test`: every interesting one of them is unreachable from a 2.1 headless
// run, which is the only Factorio this repository has.
//
// One integer compare on every call after the first of each heap, which is what
// keeps it affordable on `noteAddedPart`'s path -- that one runs for every part
// anybody places anywhere.
func multiEdgeAllowed() bool {
	if edgeMode == edgeModeUnchecked {
		edgeMode = edgeModeSingle
		if stackCapable() && settingMultiEdge() == edgemode.SettingOn {
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

// settingMultiEdge reads the policy.
//
// NIL IS THE ORDINARY ANSWER AND NOT AN ERROR. Reading an undefined runtime
// setting returns nil and raises nothing (measured on 2.1.14), so this needs no
// version gate of its own: Absent IS "this engine does not define it". Only the
// WRITE has to be gated, and it is gated on the capability marker rather than on
// this, because writing an undefined key raises.
//
// `settings.global` is a LuaCustomTable, so this is the `prototypes.entity`
// idiom again: the raw handle plus one index read, two host calls, against a
// whole-dictionary attribute that would materialise every runtime setting in the
// game. The handle is taken fresh every time -- no reference outlives its
// dispatch, which is this guest's oldest rule -- and the whole call happens once
// per heap behind multiEdgeAllowed's cache.
func settingMultiEdge() edgemode.Setting {
	raw, err := fkapi.Settings.GlobalRaw()
	if err != nil {
		return edgemode.SettingAbsent
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(MultiEdgeSetting))
	if err != nil || v.Tag != fkapi.TagMap {
		return edgemode.SettingAbsent
	}
	for i := range v.Map {
		if v.Map[i].Key.Tag != fkapi.TagString || v.Map[i].Key.Str != settingValueKey {
			continue
		}
		if v.Map[i].Val.Tag == fkapi.TagBool && v.Map[i].Val.Bool {
			return edgemode.SettingOn
		}
		return edgemode.SettingOff
	}
	return edgemode.SettingAbsent
}

// settingKV is the one-entry ModSetting table the write hands over, package
// level so that flipping the setting allocates nothing.
var settingKV [1]fkapi.KeyValue

// writeMultiEdgeSetting is `settings.global["bbb-multi-edge-parts"] = {value =
// on}`, and it is the only write this mod makes to anything outside its own
// entities.
//
// THE CAPABILITY GATE IS A CORRECTNESS GATE. Writing a `settings.global` key
// that this engine does not define RAISES -- measured on 2.1.14,
// `LuaCustomTable doesn't contain key ...` -- and a 2.0 save opened on 2.1 is
// full of exactly the clusters the only caller looks for. The caller checks it
// too (edgemode.GrandfatherNeeded takes the marker as its outer term); the
// belt-and-braces here costs one cached compare and buys the guarantee that no
// future caller can reach the raise by forgetting.
//
// It is expressible at all only since FkLua grew an index-assign member kind
// (FKLUA-GAPS.md item 23): the runtime API declares no write side on
// `LuaCustomTable`'s index operator, so the binding is emitted from an allowlist
// over what the description says in prose.
func writeMultiEdgeSetting(on bool) bool {
	if !stackCapable() {
		return false
	}
	raw, err := fkapi.Settings.GlobalRaw()
	if err != nil {
		return false
	}
	settingKV[0] = fkapi.KeyValue{Key: fkapi.OfString(settingValueKey), Val: fkapi.OfBool(on)}
	err = fkapi.LuaCustomTable{Object: raw}.Set(fkapi.OfString(MultiEdgeSetting),
		fkapi.Value{Tag: fkapi.TagMap, Map: settingKV[:]})
	return err == nil
}

// edgeAnchorSettle records what the registry has just been reconciled to.
//
// Called from every path that leaves the registry in agreement with the rule: a
// rebuild from world, a new save, and the two arms of the flip handler. It is
// the only writer other than the grandfather pass, which sets it BEFORE its own
// write so that the synchronous event that write raises finds agreement and does
// nothing.
func edgeAnchorSettle() { edgeAnchor = edgemode.ModeOf(multiEdgeAllowed()) }

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
		// A REFUSAL THE REBUILD ISSUED IS A BALANCER NOBODY JUST BUILT. The
		// world was like this when the save opened; the informed flush a tick
		// later will refuse it again and speak, and what it should say then is
		// the SUMMARY rather than the ordinary "the extra piece was left in
		// place" -- there is no extra piece. Noting the root here is what carries
		// that across the one tick between the two. It is a union with what the
		// rebuild's own fold found, because a multi-edge cluster with no standing
		// network is never classified by inspectNetwork and would otherwise reach
		// the informed flush unannounced.
		noteAnnounce(root)
		return
	}
	logRefusedSingleEdge(root, worst, sedgeTiles)
	if announcedRefusal(root) {
		// Into the summary, and nothing said here: settleEdgeMode speaks once per
		// affected force at the end of this flush, with a ping per balancer.
		return
	}
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

// ---------------------------------------------------------------------------
// A SAVE BUILT TO THE OTHER RULE
// ---------------------------------------------------------------------------
//
// Everything above this line is about an EDIT: somebody asks a part for a second
// belt and is refused before anything is touched. Everything below is about a
// whole SAVE arriving where the rule is not the one it was built under, which is
// a different act and needs different machinery in three places.
//
// TWO TRIGGERS, ONE MECHANISM.
//
//	a 2.0 save opened on 2.1 (or any load where the marker is gone): the heap is
//	declined, rebuildFromWorld runs, and every multi-edge cluster it classifies
//	is found there.
//	the setting flipped OFF on 2.0, same build: the heap survives and nothing
//	rebuilds, so `sweepStackedInterfaces` goes and looks.
//
// WHAT HAPPENS TO AN OFFENDING NETWORK IS A MANDATORY TEARDOWN AND THEN A
// REFUSAL, and it is deliberately NOT the over-limit standing-state idiom. An
// over-limit refusal leaves the working machine alone because the machine is
// fine and only the requested EDIT is not. Here the machine itself is the thing
// that cannot exist: stacked linked belts standing in a world whose engine
// forbids them are a latent engine risk on every load rather than merely an
// unsupported shape (boskid, forums t=135830 -- belt-to-belt connections are not
// saved, they are re-derived at load, and one belt-connectable per tile is what
// makes that re-derivation unambiguous). They come down at the first
// opportunity.
//
// AND ON 2.1 THE ENGINE HAS ALREADY HALF-DONE IT, SILENTLY. Loading a 2.0
// multi-edge save under 2.1.14 does not crash: the engine deletes all but one
// belt-connectable per tile with no log line of any kind, and leaves the hidden
// network fully intact -- measured, m2's 21 rigs came back with 77 interfaces of
// the ~140 they were built with and every hidden splitter still standing. So the
// rebuild wakes into clusters whose standing networks are missing most of their
// interfaces and whose remaining ports are a lottery. The teardown recovers
// everything still in the hidden half; what the engine deleted went with the
// interfaces it deleted, which is at most eight items each and is not ours to
// recover.
//
// THE ITEMS NEED NO NEW CODE. The teardown opens an `owned` carry pool, the
// refused compile claims nothing, and `closePool`/`settleCarry` spill it beside
// the cluster -- which is what this mod has done with a REMOVAL's items since
// "A recompile is not a removal", and a machine that cannot exist any more is a
// removal.
//
// CONSIDERED AND REJECTED: PARTIAL SERVICE -- compiling a degraded network that
// keeps one deterministically chosen edge per tile so items keep moving. Any
// pick is functionally arbitrary: a four-part 4x4 keeps four of its eight belts,
// and which four decides whether it deadlocks, starves, or silently delivers
// wrong ratios. This mod's one promise is exact balance, and a stopped balancer
// that says why beats a running one that lies. Also rejected: mining the
// player's excess belts to make the cluster legal (destroying their property),
// and auto-placing chests for the drained items (inventing entities they did not
// build).

var (
	// sedgeCondemned is the roots whose STANDING network was built to a rule this
	// engine no longer allows, and which must therefore come down even though the
	// compile that follows will refuse.
	//
	// IT IS WHAT KEEPS THE ORDINARY REFUSAL ORDINARY. compile() asks the rule in
	// front of its own teardown, and that is the whole of the sixty-fifth belt's
	// fix -- a player who lays a second belt on a WORKING single-edge balancer
	// must find it still running. The difference here is not the new edge list, it
	// is that the network already standing is itself invalid, and only the two
	// producers below can tell the two apart. Empty in every save that was built
	// under the rule it is running under, which is every save anybody makes today.
	sedgeCondemned []uint32

	// sedgeAnnounce is the roots whose refusal belongs in the SUMMARY rather than
	// in the ordinary "the extra piece was left in place" message.
	//
	// Two producers, and their union is what makes it complete: rebuildFromWorld's
	// fold, which sees every cluster it classifies including the ones it adopts
	// (that is the 2.0 grandfather case, where nothing is ever refused), and
	// refuseSingleEdge's rebuild arm, which catches a multi-edge cluster that had
	// no standing network for the fold to inspect.
	sedgeAnnounce []uint32
)

func condemnStanding(root uint32) {
	for i := range sedgeCondemned {
		if sedgeCondemned[i] == root {
			return
		}
	}
	sedgeCondemned = append(sedgeCondemned, root)
}

// takeCondemned reports whether this root's standing network has to come down,
// and consumes the answer: the teardown happens once.
func takeCondemned(root uint32) bool {
	for i := range sedgeCondemned {
		if sedgeCondemned[i] != root {
			continue
		}
		sedgeCondemned[i] = sedgeCondemned[len(sedgeCondemned)-1]
		sedgeCondemned = sedgeCondemned[:len(sedgeCondemned)-1]
		return true
	}
	return false
}

// forgetCondemned drops whatever the flush did not reach -- a condemned cluster
// that dissolved before it could be compiled. Its network went down with it.
func forgetCondemned() { sedgeCondemned = sedgeCondemned[:0] }

func noteAnnounce(root uint32) {
	for i := range sedgeAnnounce {
		if sedgeAnnounce[i] == root {
			return
		}
	}
	sedgeAnnounce = append(sedgeAnnounce, root)
}

// announcedRefusal is a membership TEST and not a take: settleEdgeMode walks the
// whole list at the end of the flush to build the summary, so an entry consumed
// here would be a balancer missing from its own checklist.
func announcedRefusal(root uint32) bool {
	for i := range sedgeAnnounce {
		if sedgeAnnounce[i] == root {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// The flip
// ---------------------------------------------------------------------------

// onEdgeModeSettingChanged is `on_runtime_mod_setting_changed`, and it can only
// ever fire on 2.0: 2.1 does not define the setting.
//
// THE COMPARISON AGAINST THE ANCHOR IS LOAD-BEARING AND NOT DEFENSIVE. Factorio
// raises this for a write of the value already there (measured on 2.1.14), and
// it raises it SYNCHRONOUSLY, inside the assigning statement -- so the
// grandfather pass's own write re-enters this guest before `Set` returns. That
// pass writes the anchor first, so the re-entrant call lands on agreement and
// does nothing at all, which is why no self-write flag is needed.
//
// NOTHING HERE FLUSHES. The write that raises it runs from flush() itself, after
// endCarry, so a flush here would be a flush inside a flush. It queues and asks
// for the next tick's, exactly as an ordinary event does.
func onEdgeModeSettingChanged(name string) {
	if name != MultiEdgeSetting {
		return
	}
	// The cached answers were resolved before the flip and are now wrong. This is
	// the fourth and last caller of the recheck, and the only one that is not a
	// load hook.
	edgeModeRecheck()
	want, act := edgemode.Reconcile(stackCapable(), settingMultiEdge(), edgeAnchor)
	edgeAnchor = want
	switch act {
	case edgemode.ActNone:
		return
	case edgemode.ActSweep:
		n := sweepStackedInterfaces()
		logStart("single-edge: multiple belts per part turned OFF; ")
		logU(n)
		logS(" standing balancers were built that way and are coming down")
		logEnd()
	default:
		// Turned back ON. Nothing standing is wrong -- a refused cluster never got
		// a network -- so every cluster is simply re-queued: the refused ones
		// compile, because their stored fingerprint never matched the world, and
		// every other one skips on the fingerprint it never lost.
		auditRoots = liveRootList(auditRoots)
		for i := range auditRoots {
			markLive(auditRoots[i])
		}
		logStart("single-edge: multiple belts per part turned ON; ")
		logU(uint32(len(auditRoots)))
		logS(" clusters re-queued")
		logEnd()
	}
	requestFlush()
}

// sweepStackedInterfaces finds every standing network that was built with two
// belts on one part and marks it for demolition.
//
// A WHOLE-SAVE RE-CLASSIFICATION, which is what an audit is, and it is affordable
// for the same reason: this is a keypress. It cannot be reached at all on an
// engine that has no setting to flip.
//
// THE FINGERPRINT HAS TO BE INVERTED and that is not a trick borrowed for
// convenience -- it is the same statement rebuildFromWorld makes for the same
// reason. Nothing about the WORLD changed when the setting moved, so a plain
// re-queue would find the stored fingerprint matching and skip; inverting it is
// how this guest says "whatever you have recorded, it is not what should be
// standing there".
func sweepStackedInterfaces() uint32 {
	auditRoots = liveRootList(auditRoots)
	found := uint32(0)
	for i := range auditRoots {
		r := auditRoots[i]
		ni, had := nets[r]
		if !had {
			continue // nothing standing to be wrong
		}
		tiles := collectCluster(r)
		if len(tiles) == 0 {
			continue
		}
		s, ok := surfaceByIndex(tiles[0].s)
		if !ok {
			continue
		}
		classifyEdges(s, tiles, pforce[r])
		if sedgeWorst < 2 {
			continue
		}
		found++
		condemnStanding(r)
		noteAnnounce(r)
		ni.fp = ^ni.fp
		nets[r] = ni
		markLive(r)
	}
	return found
}

// ---------------------------------------------------------------------------
// What the player is told, once, at the end of the informed flush
// ---------------------------------------------------------------------------

// settleEdgeMode is the one place either summary is spoken and the one place the
// setting is written.
//
// WHERE IT RUNS IS THE DESIGN, and it is the same argument revertOverLimit
// makes: `flush()`, AFTER `endCarry()`. The write raises
// `on_runtime_mod_setting_changed` synchronously, so it re-enters this guest --
// and a re-entrant handler that appended to the queues this flush is draining, or
// that touched the package-level compile buffers, would be the exact hazard
// `mine_entity` posed. After the drain there is no drain to re-enter.
//
// AND IT NEVER SPEAKS FROM INSIDE THE REBUILD. That is the wake-race principle
// (limit.go, lifecycle.go): rebuildFromWorld reconstructs a whole session's
// registry inside one dispatch with none of that dispatch's own events delivered,
// so any verdict it reaches about a PLAYER is provisional. It may log, it may
// queue, and it may not address anybody -- so the rebuild's own flush finds this
// gated shut, and the flush the rebuild then asks for is the one that speaks.
//
// ONE LENGTH TEST ON AN EMPTY SLICE on every flush of every save that was built
// under the rule it is running under.
func settleEdgeMode() {
	if len(sedgeAnnounce) == 0 || rebuildingFromWorld {
		return
	}
	n := gatherAffected()
	sedgeAnnounce = sedgeAnnounce[:0]
	if n == 0 {
		return
	}
	if edgemode.GrandfatherNeeded(stackCapable(), settingMultiEdge(), n) {
		grandfatherMultiEdge(n)
		return
	}
	if stackCapable() {
		// 2.0, and the setting is already on: these balancers are working and were
		// grandfathered on an earlier load. Nothing to say -- the once-per-state
		// gate, not a nag on every load.
		return
	}
	tellMigrated(n)
}

// grandfatherMultiEdge keeps a save that predates the rule working, and says so.
//
// THE ANCHOR IS WRITTEN BEFORE THE SETTING, and that ordering is the whole
// re-entrancy argument: `Set` dispatches the setting-changed event before it
// returns, the handler compares the setting against the anchor, and an anchor
// that already says Multi makes that call a no-op. Writing them the other way
// round would have the handler decide the mode had just changed and re-queue
// every cluster in the save for nothing.
//
// NOTHING IS RECOMPILED. The clusters this is about were ADOPTED by the rebuild
// -- on 2.0 all their interfaces are still standing, so the adoption comparison
// matched exactly -- and they are running. The flip is so that the NEXT edit to
// one of them is compiled rather than refused.
func grandfatherMultiEdge(n uint32) {
	edgeAnchor = edgemode.ModeMulti
	if !writeMultiEdgeSetting(true) {
		// The write is the feature. If it did not land the save is single-edge
		// from here, which is survivable and must not be silent.
		logAlertStart("single-edge: ")
		logU(n)
		logS(" balancers use several belts per part and the setting that keeps")
		logS(" them working could not be written; they will be refused")
		logEnd()
		edgeAnchor = edgemode.ModeSingle
		return
	}
	edgeModeRecheck()
	logStart("single-edge: kept multiple belts per part enabled for this save -- ")
	logU(n)
	logS(" balancers use it; settings.global ")
	logS(MultiEdgeSetting)
	logS(" = true")
	logEnd()
	tellAffected(msgGrandfathered, false)
}

// tellMigrated is the 2.1 half: the balancers cannot be kept, so what is offered
// instead is a checklist.
func tellMigrated(n uint32) {
	logStart("single-edge: ")
	logU(n)
	logS(" balancers were built with several belts per part; this Factorio")
	logS(" cannot stack belt-connectables, so they are refused and their")
	logS(" contents are on the ground beside them")
	logEnd()
	tellAffected(msgMigrated, true)
}

// ---------------------------------------------------------------------------
// Gathering the affected clusters, and pointing at them
// ---------------------------------------------------------------------------

// affCluster is one balancer on the checklist: everything a ping needs, and no
// entity reference, because these are read a tick after the rebuild that found
// them.
type affCluster struct {
	force uint32
	surf  uint32
	x, y  int32
}

var (
	affected  []affCluster
	affForces []uint32
	affTile   [1]key
)

// gatherAffected resolves the announced roots into clusters, deduplicating by
// the root each one resolves to NOW -- a cluster can have been re-rooted between
// the rebuild that announced it and this flush, and two announcements that
// resolve to one cluster are one balancer on the checklist.
//
// Node ids in append order, which is the rebuild's node-id order and then event
// order: deterministic on every client, which matters because what comes out of
// it reaches `force.print`.
func gatherAffected() uint32 {
	affected = affected[:0]
	affForces = affForces[:0]
	gen++
	for i := range sedgeAnnounce {
		r := sedgeAnnounce[i]
		if int(r) >= len(alive) || !alive[r] {
			continue
		}
		r = find(r)
		if mark[r] == gen {
			continue
		}
		mark[r] = gen
		k := ppos[r]
		f := pforce[r]
		affected = append(affected, affCluster{force: f, surf: k.s, x: k.x, y: k.y})
		seen := false
		for j := range affForces {
			if affForces[j] == f {
				seen = true
				break
			}
		}
		if !seen {
			affForces = append(affForces, f)
		}
	}
	return uint32(len(affected))
}

// tellAffected says one thing to each force that owns an affected balancer.
//
// `withPings` is what separates the two messages rather than a second function:
// the 2.1 one is a checklist of machines to rebuild and every one of them is
// somewhere the player has to go, so it carries a `[gps=...]` per cluster; the
// 2.0 one is a warning about a save that still works, so it carries none and the
// player is not sent on a tour of balancers that are running.
func tellAffected(msgKey string, withPings bool) {
	for i := range affForces {
		f := affForces[i]
		count := uint32(0)
		gpsReset()
		for j := range affected {
			if affected[j].force != f {
				continue
			}
			count++
			if withPings {
				gpsAdd(&affected[j])
			}
			if affTile[0] == (key{}) || count == 1 {
				affTile[0] = key{s: affected[j].surf, x: affected[j].x, y: affected[j].y}
			}
		}
		if count == 0 {
			continue
		}
		lf, ok := forceOfCluster(affTile[:], f)
		if !ok {
			// No part of ours left standing on that tile to read a force off, or the
			// index moved. There is nobody to address and nothing to be done about
			// it; the log line above is the record either way.
			continue
		}
		limMsg[0] = fkapi.OfString(msgKey)
		limMsg[1] = fkapi.OfNumber(float64(count))
		nparam := 1
		if withPings {
			limMsg[2] = fkapi.OfString(gpsString())
			nparam = 2
		}
		msg := fkapi.Value{Tag: fkapi.TagArray, Array: limMsg[:1+nparam]}
		err := lf.Print(msg, nil)
		// The one half of this a headless run can see. `force.print` writes to the
		// game's chat, which no script can read back and which `--benchmark` does
		// not log -- so without this line a suite could say the migration happened
		// and nothing at all about whether anybody was told. What could
		// realistically fail behind it is resolving a LuaForce from a force INDEX
		// and the LocalisedString crossing the boundary, and both are on this side
		// of it.
		logStart("single-edge: told force ")
		logU(f)
		logS(" about ")
		logU(count)
		logS(" balancers built to the multi-edge rule")
		if err != nil {
			logS(" -- print FAILED")
		}
		logEnd()
	}
	// limMsg[1] and [2] are the refusal's parameters too, and it writes its own
	// numbers before every use; [0] is rewritten by every caller. Leaving a string
	// in [2] would keep whatever the last summary borrowed alive, so it is put
	// back to a number.
	limMsg[2] = fkapi.OfNumber(0)
}

// ---------------------------------------------------------------------------
// The ping list
// ---------------------------------------------------------------------------
//
// `[gps=x,y,surface]` is Factorio's own rich text for a clickable map ping, and
// a LocalisedString parameter renders it. Building the list is the one place
// this guest assembles a string that is not a log line, so it is assembled the
// way log lines are (logline.go): ONE package-level fixed array, `copy` rather
// than `append`, and `unsafe.String` to hand over a borrow rather than a copy.
// A separate buffer from logline's, because a log line written between building
// this and sending it would otherwise overwrite it.
//
// TRUNCATED RATHER THAN GROWN, and the truncation is a policy here rather than a
// backstop: a base with two hundred affected balancers would produce a chat line
// nobody can read, so the list stops at what fits and the count in the sentence
// -- which is exact -- says how many there really are.

var (
	gpsBuf    [900]byte
	gpsLen    int
	gpsDigits [12]byte
	// One-entry surface-name memo. Consecutive clusters are almost always on the
	// same surface, and a name is a host call and a string copy.
	gpsSurfIdx  uint32
	gpsSurfName string
	gpsSurfOK   bool
)

func gpsReset() {
	gpsLen = 0
	gpsSurfOK = false
}

func gpsS(s string) { gpsLen += copy(gpsBuf[gpsLen:], s) }

func gpsI(v int32) {
	if v < 0 {
		gpsS("-")
		v = -v
	}
	u := uint32(v)
	i := len(gpsDigits)
	for {
		i--
		gpsDigits[i] = byte('0' + u%10)
		u /= 10
		if u == 0 {
			break
		}
	}
	gpsLen += copy(gpsBuf[gpsLen:], gpsDigits[i:])
}

// gpsAdd appends one ping, and stops appending once the buffer is nearly full.
// The reserve is comfortably more than the longest ping a real surface name can
// produce, so a ping is never written half-way.
func gpsAdd(c *affCluster) {
	if gpsLen > len(gpsBuf)-128 {
		return
	}
	name, ok := gpsSurfaceName(c.surf)
	if !ok {
		return
	}
	if gpsLen > 0 {
		gpsS(" ")
	}
	gpsS("[gps=")
	gpsI(c.x)
	gpsS(",")
	gpsI(c.y)
	gpsS(",")
	gpsS(name)
	gpsS("]")
}

func gpsSurfaceName(si uint32) (string, bool) {
	if gpsSurfOK && gpsSurfIdx == si {
		return gpsSurfName, true
	}
	s, ok := surfaceByIndex(si)
	if !ok {
		return "", false
	}
	n, err := s.Name()
	if err != nil {
		return "", false
	}
	gpsSurfIdx, gpsSurfName, gpsSurfOK = si, n, true
	return n, true
}

// gpsString borrows the buffer. The host copies the bytes into a Lua string
// during the call it is passed to, and nothing writes the buffer in between.
func gpsString() string {
	if gpsLen == 0 {
		return ""
	}
	return unsafe.String(&gpsBuf[0], gpsLen)
}
