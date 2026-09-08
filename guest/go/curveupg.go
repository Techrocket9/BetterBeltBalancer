// The curved exit, arriving on a save that did not have it.
//
// The setting is `bbb-curved-exits` (curve.go); this file reads and writes it
// through the idiom sedge.go established, and behaves as the shipped default
// does where the key is missing -- Absent reads as ON -- so that a game whose
// settings stage never defined it is classified the way the prototype says
// rather than silently the other way.
//
// ---------------------------------------------------------------------------
// WHAT THE RULE DOES TO A WORLD THAT WAS BUILT WITHOUT IT
// ---------------------------------------------------------------------------
//
// A belt running across a balancer's face was NOTHING until 0.3.3 and is an
// OUTPUT now. That is a change in the meaning of a world, not only in the
// meaning of an edit -- so the first load after the update classifies a factory
// somebody else built, under a rule that factory was not built to. Measured on
// 2.0.77, one save created by the guest before the curve arm and loaded by the
// guest after it:
//
//	a curve-eligible belt beside an EDGELESS part   the balancer is rebuilt 1 -> 2.
//	                                                 Its real output HALVES (712
//	                                                 items over a window, then 355)
//	                                                 and a line the player never
//	                                                 meant as an output takes 347.
//	the same belt beside an OCCUPIED part            two edges on one tile. The
//	                                                 standing network is CONDEMNED,
//	                                                 torn down and refused: 12 items
//	                                                 on the ground, and on 2.1 the
//	                                                 balancer is dead for good.
//
// Neither is a defect in the classifier. The classifier is right about what the
// player's belts now mean; it is the player who has not been asked.
//
// ---------------------------------------------------------------------------
// THE TRIGGER IS THE SAVE'S OWN STATE VERSION
// ---------------------------------------------------------------------------
//
// `fk_state_version` (main.go, stateversion.go) is stamped into every save
// beside the build id and handed back to `fk_migrate`. Every build up to 0.3.2
// exported none, so their saves read 0; 0.3.3 reports 1. So the question "was
// this world built before a belt could turn as it left a balancer" is one
// integer compare against a number the ENGINE carried, rather than a shape this
// guest has to recognise in the world.
//
// THAT REPLACED A SIGNATURE, AND THE SIGNATURE WAS WRONG. It was an adoption
// failure of one exact shape: re-derive the edge list, drop the edges the curve
// arm produced, and adopt if the result is in one-to-one correspondence with the
// interfaces standing. A cluster the mod itself compiled and nobody has touched
// since does answer that. Anything else does not -- and "anything else" is not
// exotic. Measured on 2.0.77, three shapes an old-rule save can perfectly well
// contain, every one of which failed BOTH readings and was therefore recompiled
// under the new rule as though it had been built to it:
//
//	a HALF-BUILT cluster (inputs, no output) with a belt running past it.
//	  inspectNetwork returns at `len(ents) == 0` before it compares anything, so
//	  no reading was ever taken: the load compiled it 1 -> 1 and 807 items went
//	  into a chest the player never connected.
//	an old-rule cluster whose OUTPUT BELT WAS MINED while the mod was
//	  uninstalled. The standing interfaces describe a belt that is gone, so
//	  neither reading is a bijection: torn down and recompiled onto the curve
//	  belt, 819 items misrouted.
//	an old-rule cluster with one extra INPUT laid while the mod was uninstalled,
//	  whose curve belt lands on the tile that already carries its output. Neither
//	  reading matches, so the curve edge survives, the tile carries two, the
//	  standing network is condemned and refused, 12 items reach the ground -- and
//	  on 2.0 `settleEdgeMode` then writes `bbb-multi-edge-parts = true` and tells
//	  the force it has kept multiple belts per part working, for a save that never
//	  used them.
//
// One clean old-rule cluster anywhere else in the save rescues all three, which
// is why the first `curv` suite was green: the decision is per SAVE, so one
// cluster that does match is enough to write the setting, and the setting is
// what the other three then get. A save whose every affected cluster is one of
// the shapes above takes no decision on any load.
//
// ---------------------------------------------------------------------------
// THE SIGNATURE THE WATERMARK LEAVES, AND WHY IT IS COMPLETE
// ---------------------------------------------------------------------------
//
// While a load is UNDECIDED the signature is "this cluster's classification
// contains at least one curve edge", whatever else is true of the cluster --
// half-built, drifted, refused, adopted, or never compiled at all. At the moment
// that save was written no such belt could have been an output, because the
// build that wrote it had no way to read one; so the reading is complete rather
// than heuristic, and it needs no comparison against what is standing.
//
// ITS ONE FALSE POSITIVE IS THE ONE THE OLD SIGNATURE HAD: a player who laid one
// of these belts while the mod was uninstalled, meaning it as an output, and
// updated in the same step. They turn the rule back on. It is the benign
// direction and it is the only one available, because nothing in a save records
// what a belt was FOR.
//
// ---------------------------------------------------------------------------
// A CLUSTER WITH NO CURVE EDGE IS IDENTICAL UNDER BOTH RULES
// ---------------------------------------------------------------------------
//
// ...which is what lets the decision be taken at the FIRST curve edge any
// classification in the load produces, wherever that happens -- the rebuild's
// inspection, a legacy conversion's flush, or an ordinary compile in the same
// dispatch. There is no ordering problem to solve: a cluster classified earlier
// in this load had no curve edge, or the decision would already have been taken,
// and a cluster with no curve edge reads the same either way. Nothing has to be
// re-done.
//
// So `curveDecide` drops the curve edges from the list it was handed and every
// later classification in the load runs with the rule off, which is what makes
// adoption, the port limit, the one-belt-per-part count and the compile itself
// all see the world the save was written to -- by construction rather than by a
// second reading passed around between them. `inspectNetwork`'s curve-free retry
// and the `multiOver` recount it needed are DELETED with it.
//
// THE PROBE STAYS ON FOR THE REST OF THE WINDOW, THOUGH, and that is the one
// place the decision and its consequence are separate. What the probe finds is
// also what the player is HANDED: a save with five affected balancers has to
// name five and ping five, so switching the probe off at the first would trade a
// checklist for two host calls per face on one load in the history of a save.
// curve.go, curvedExitsAllowed.
//
// ---------------------------------------------------------------------------
// WHERE THE WINDOW OPENS AND CLOSES
// ---------------------------------------------------------------------------
//
// It opens in `fk_migrate` when the save's stamp is older than this guest's
// rung, and in `legacyScan` when a conversion actually converted something --
// see curveUndecideForLegacy. It closes at the first flush of the load that
// settles, which is `settleCurveMode`. That flush is guaranteed:
// `rebuildFromWorld` asks for one whenever the window is open, exactly as it
// does for `sedgeAnnounce`.
//
// CLOSING IT IS AS LOAD-BEARING AS OPENING IT. A belt a player lays afterwards
// arrives as an event, in an ordinary flush, and is an OUTPUT -- which is the
// feature. A window left open would read that belt as evidence about a world
// that was built before the rule.
//
// AND THE WRITE AND THE MESSAGE RUN WHERE settleEdgeMode's DO: flush(), after
// endCarry, and never from inside the rebuild. The write raises
// on_runtime_mod_setting_changed synchronously, so it re-enters this guest; and
// the rebuild may not address a player at all (limit.go, refuseAdmit).
//
// It runs BEFORE settleEdgeMode, which reads in the direction the answers depend
// -- a tile whose second edge is a curve is not a multi-edge tile in a save that
// is keeping the old reading -- and which protects nothing on its own: the
// decision is taken in classifyEdges, upstream of both. compile.go's flush()
// carries the measurement. What keeps one rule's migration from speaking for
// another's is recountEdgesPerTile below.
package main

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/edgemode"

// curveSettingNow is the stored value as the fold wants it: an explicit Off is
// what says a player, or an earlier load, has already answered.
func curveSettingNow() edgemode.Setting {
	on, present := readGlobalBool(CurvedExitsSetting)
	switch {
	case !present:
		return edgemode.SettingAbsent
	case on:
		return edgemode.SettingOn
	}
	return edgemode.SettingOff
}

// The setting itself -- its name, its cache, its recheck, its read, its write
// and its flip handler -- is guest/go/curve.go and guest/go/globalsetting.go.
// This file is only what an UPGRADE has to do about it.

const msgCurvesKept = "bbb.curved-exits-kept"

// What this pass calls itself in the log, handed to `tellAffected` rather than
// written inside it: the two summaries that were there before it are
// single-edge's, and one function saying `single-edge:` for all three would name
// the wrong rule for the one balancer a player went looking for.
const (
	curveHeading = "curved exits: "
	curveWhat    = "balancers built before a belt could turn as it left one"
)

// curveUndecided is the window: this load is reading a world written before the
// curve rule existed and has not settled what to do about it.
//
// It is NOT persistent state and there is nothing in the save that corresponds
// to it. What persists is the state version FkLua stamps and the setting this
// pass may write, and both of those are answers rather than flags -- which is
// the whole difference between this and a `bbb-curve-decided` setting nobody has
// to keep in step with anything.
//
// THE ONE THING THAT COSTS, stated: FkLua republishes the stamp in the same
// dispatch as `fk_migrate`, and the flush that settles this is the NEXT tick's.
// A save quit inside that one tick comes back stamped at this rung with the
// decision unwritten, and is then read with curves on. `fk_migrate` fires at the
// first outermost dispatch, which is inside a tick of a running game, so getting
// there means loading and quitting between two ticks; the outcome is the
// behaviour of the build before this pass, and it is not worth a mechanism.
var curveUndecided bool

// curveLegacy is the roots this load found standing to the old rule. It is what
// the informed flush speaks about, and it is package level rather than returned
// because the rebuild that fills it may not speak.
var curveLegacy []annNote

// curveWriteOwed is set with the first note and cleared by the flush that acts.
var curveWriteOwed bool

// curveForcedOff is the decision this load took, held between the classification
// that took it and the write that records it.
//
// A SEPARATE FLAG RATHER THAN THE CACHE, AND THAT IS MEASURED. Priming
// `curveCache` was the first cut, and the cache does not survive the dispatch:
// `fk_migrate` and `fk_on_configuration_changed` BOTH fire on the load this pass
// runs on, and the second of them calls `curveRecheck` -- so between the decision
// and the flush that writes, the classifier went back to reading a setting that
// still says true. Observed on 2.0.77 as the flip handler announcing a flip
// nobody made and re-queueing the whole save, on the very run this suite passed.
//
// It is false in every save this pass is not about, which is every save built on
// 0.3.3 or later.
var curveForcedOff bool

// curveVersionArrived opens the window when the save is older than this guest's
// rung. `fk_migrate` is the only caller; see stateversion.go for the rungs.
func curveVersionArrived(oldVersion uint32) {
	if oldVersion < stateVersion {
		curveUndecided = true
	}
}

// curveUndecideForLegacy opens the same window for a Belt Balancer world this
// mod has just converted.
//
// THERE IS NO WATERMARK TO READ ON THAT PATH AND THERE CANNOT BE. A mod being
// ADDED to an existing save reaches `fk_on_init`, and Factorio raises no
// configuration change for a guest that was not there before -- so no
// `fk_migrate` fires and no `oldVersion` exists. What decides instead is what
// was converted: the incumbent shared this mod's own old limitation, a belt
// curving away from one of its parts was not an output either, and the balancers
// this scan has just taken over were laid to that reading.
//
// It is called only when something was actually converted, so a save with no
// incumbent in its history never opens the window -- and a conversion whose
// world has no curve belt in it decides nothing, because the window is opened
// and the decision is not the same act.
//
// AND IT IS RIGHT ON A 0.3.3 SAVE TOO, which is why it does not ask the
// watermark even where one exists: what those parts were built to is a fact
// about the INCUMBENT and not about which of our builds wrote the save.
func curveUndecideForLegacy() { curveUndecided = true }

// noteCurveLegacy records one cluster whose classification produced a curve edge
// while the load was undecided.
func noteCurveLegacy(root uint32) {
	curveWriteOwed = true
	for i := range curveLegacy {
		if curveLegacy[i].root == root {
			return
		}
	}
	curveLegacy = append(curveLegacy, annNote{root: root})
}

// curveDecide is the whole of the reading, applied to one classification.
//
// Called from classifyEdges and from nowhere else, which is what makes it
// unmissable: every path that asks the world what a cluster's edges are goes
// through that one function, so there is no call site to forget on the day
// somebody adds a fifth.
//
// IT TAKES THE DECISION AND APPLIES IT IN ONE ACT, which is why the strip is
// here rather than at the caller. The caller is holding `edgeBuf` and is about
// to fingerprint it, compare it against what is standing, count its edges per
// tile or build from it; handing any of them the reading with the curve edges
// still in would be handing them the world under the other rule.
func curveDecide(tiles []key) {
	found := false
	for i := range edgeCurved {
		if edgeCurved[i] {
			found = true
			break
		}
	}
	if !found {
		return
	}
	curveForcedOff = true
	if id, ok := index[tiles[0]]; ok {
		noteCurveLegacy(find(id))
	} else {
		// A classification of tiles the registry does not hold cannot happen --
		// every caller got them from collectCluster -- but the write is what the
		// player's save depends on and it may not be lost to a lookup.
		curveWriteOwed = true
	}
	n := 0
	for i := range edgeBuf {
		if edgeCurved[i] {
			continue
		}
		edgeBuf[n], edgeCurved[n] = edgeBuf[i], false
		n++
	}
	edgeBuf, edgeCurved = edgeBuf[:n], edgeCurved[:n]
	recountEdgesPerTile()
}

// recountEdgesPerTile rebuilds sedge.go's two per-tile numbers over the edge
// list as it now stands.
//
// classifyEdges counts them in the walk that produces the edges, which costs
// nothing and is wrong the moment an edge is taken back out: a tile whose second
// edge was a curve is not a tile carrying two belts in a save that is keeping the
// old reading, and reporting it as one condemns a working machine and hands the
// player the multi-edge grandfather's sentence about a rule they never used.
//
// AND IT IS A RECOUNT RATHER THAN A CLEAR, WHICH IS WHAT KEEPS THE 2.1 MIGRATION
// HONEST. That migration's whole question is whether a tile carries two belts,
// asked of a save the engine has already pruned; a load can be undecided about
// curves and be reading such a save at the same time, and the two answers must
// not be confused. A tile whose only second edge was the curve stops counting,
// which is right -- the save is keeping the reading under which that belt is
// nothing. A tile with two STRAIGHT edges goes on counting whatever the curve
// decision was, so the remnant is still condemned and the player still told.
// `mig21` asserts the other half from the other side: neither of its fixtures
// has a belt across a face, so no load of them may decide anything at all.
//
// Quadratic in an edge list bounded by plan.MaxPorts, and it runs only for a
// cluster that had a curve edge on the one load in a save's history that decides.
func recountEdgesPerTile() {
	sedgeWorst, sedgeTiles = 0, 0
	for i := range edgeBuf {
		n, first := uint32(0), true
		for j := range edgeBuf {
			if edgeBuf[j].TileX != edgeBuf[i].TileX || edgeBuf[j].TileY != edgeBuf[i].TileY {
				continue
			}
			n++
			if j < i {
				// Counted from that tile's first edge and no other, or a tile
				// with three edges would be counted three times.
				first = false
			}
		}
		if !first || n < 2 {
			continue
		}
		sedgeTiles++
		if n > sedgeWorst {
			sedgeWorst = n
		}
	}
}

// settleCurveMode closes the window, writes the setting and says so, once, from
// the informed flush.
//
// It is beside settleEdgeMode in flush() and for the same three reasons: the
// write re-enters this guest synchronously, the drain is over, and the carry
// transaction has closed. One bool test on every flush of every save this is not
// about, which is every save built on 0.3.3 or later.
func settleCurveMode() {
	if rebuildingFromWorld {
		// The rebuild runs a flush of its own and this is not it. Closing the
		// window there would close it before the conversion, the requeue and the
		// compiles that follow in the same load have been classified at all.
		return
	}
	curveUndecided = false
	if !curveWriteOwed {
		return
	}
	curveWriteOwed = false
	n, _ := gatherAnnounced(curveLegacy)
	curveLegacy = curveLegacy[:0]
	// n IS ZERO ONLY IF EVERY NOTED CLUSTER WAS DESTROYED IN THE ONE TICK
	// between the decision and this flush, and then the classifier stays off for
	// the rest of the session with nothing written. That is the direction that
	// cannot surprise a standing machine -- there is none left to surprise --
	// and the next load reads the setting fresh, finds no curve edge to decide
	// on, and comes up with the rule on.
	//
	// THE FOLD IS ASKED EVEN THOUGH THE DECISION IS ALREADY TAKEN, and the two
	// are not the same question: `curveForcedOff` is what the classifier used for
	// the rest of the load, and this is whether the SAVE has to be told.
	//
	// ITS SettingOff ARM IS A SHAPE GUARD AND NOT A LATCH. A note is recorded
	// only for an edge the CURVE ARM produced, and the curve arm is behind the
	// setting -- so a save whose setting is already Off produces no curve edge,
	// no note, and `n` is zero before this fold is reached at all. Every state of
	// it is proved by go test ./edgemode/.
	if n == 0 || !edgemode.CurveKeepNeeded(curveSettingNow(), n) {
		return
	}
	// THE ANCHOR, WRITTEN IMMEDIATELY BEFORE THE SETTING AND NOT EARLIER, which
	// is `grandfatherMultiEdge`'s ordering exactly. `Set` dispatches
	// `on_runtime_mod_setting_changed` before it returns, and
	// `onCurvedExitsSettingChanged` acts unless the cache already agrees with
	// what it then reads -- so a cache put to Off here makes that re-entry a
	// no-op. Anywhere earlier is too early: `curveRecheck` runs from three load
	// hooks and two of them fire on this load.
	curveForcedOff = false
	curveCache = curveOff
	if !writeGlobalBool(CurvedExitsSetting, false) {
		// The write is the feature. Without it the next load classifies these
		// balancers with curves again and does the thing this pass exists to
		// avoid, so a failure may not be silent -- and the classifier stays off
		// for the rest of this session, which is the direction that agrees with
		// the networks that were just adopted.
		curveForcedOff = true
		logAlertStart("curved exits: ")
		logU(n)
		logS(" balancers were built before a belt could turn as it left one, and")
		logS(" the setting that keeps them as they are could not be written")
		logEnd()
		return
	}
	curveRecheck()
	logStart("curved exits: kept the old reading for this save -- ")
	logU(n)
	logS(" balancers have a belt across a face that was not an output when they")
	logS(" were built; settings.global ")
	logS(CurvedExitsSetting)
	logS(" = false")
	logEnd()
	tellAffected(msgCurvesKept, true, curveHeading, curveWhat)
}
