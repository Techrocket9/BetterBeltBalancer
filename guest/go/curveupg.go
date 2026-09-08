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
// THE ANSWER, WHICH IS THE GRANDFATHER PASS WITH THE SIGN REVERSED
// ---------------------------------------------------------------------------
//
// sedge.go's grandfather takes a save that predates a rule and writes the
// setting that keeps it working. This is the same shape: the first load that can
// SEE the old rule in the standing networks writes `bbb-curved-exits = false`
// for that save, so an existing factory keeps working and the player opts in.
//
// THE SIGNATURE IS AN ADOPTION FAILURE OF ONE PARTICULAR SHAPE, and it is exact
// rather than heuristic. rebuildFromWorld compares the edge list it re-derives
// against the interfaces actually standing (lifecycle.go, inspectNetwork). A
// cluster built under the old rule fails that comparison by EXACTLY the edges the
// curve arm produced and by nothing else -- so removing them and comparing again
// is the whole test. It is not "there is a curve-shaped belt near a balancer",
// which would fire on a save that always had one; it is "the network standing in
// this world is the one the classifier would have built WITHOUT the curve arm",
// which nothing but an old-rule save can be.
//
// WHAT IT CANNOT TELL APART, said plainly: a player who laid one of these belts
// while the mod was uninstalled, meaning it as an output, and updated in the same
// step. That world is byte for byte an old-rule world and gets curves turned off;
// they turn them back on. It is the benign direction and it is the only one
// available -- nothing in a save records what a belt was FOR.
//
// THE LATCH IS THE SIGNATURE ITSELF and needs no anchor. Once the setting is off
// the classifier and the standing networks agree, so the next rebuild adopts and
// finds nothing; and if the player turns curves back ON, the flip handler
// requeues, every cluster recompiles WITH the curve edges, and from then on the
// standing network matches the curve-on reading exactly. Either way the
// comparison that fires this can never fire twice. A save created fresh on 0.3.3
// is covered by the same sentence from the other end: it has no adopted network
// at all, so there is nothing for an adoption to fail at.
//
// WHERE THE DECISION IS TAKEN IS THE DESIGN. It is taken per cluster INSIDE the
// adoption comparison, before rebuildFromWorld's own flush, because that flush is
// what does the damage: by the time anything at the tail of it could speak, the
// 1 -> 2 has been built and the condemned network is on the ground. Adopting on
// the curve-free reading is not a special case bolted on -- it is what the
// rebuild is for, which is to keep what is standing.
//
// AND THE WRITE AND THE MESSAGE RUN WHERE settleEdgeMode's DO: flush(), after
// endCarry, and never from inside the rebuild. The write raises
// on_runtime_mod_setting_changed synchronously, so it re-enters this guest; and
// the rebuild may not address a player at all (limit.go, refuseAdmit).
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/edgemode"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

// curveSettingNow is the stored value as the fold wants it: an explicit Off is
// what says an earlier load already decided.
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

// curveLegacy is the roots this load found standing to the old rule. It is what
// the informed flush speaks about, and it is package level rather than returned
// because the rebuild that fills it may not speak.
var curveLegacy []annNote

// curveWriteOwed is set with the first note and cleared by the flush that acts.
var curveWriteOwed bool

// curveForcedOff is the decision this load took, held between the scan that took
// it and the write that records it.
//
// A SECOND FLAG RATHER THAN THE CACHE, AND THAT IS MEASURED. `curveScanDone`
// primed `curveCache` and the cache does not survive the dispatch: `fk_migrate`
// and `fk_on_configuration_changed` BOTH fire on the load this pass runs on, and
// the second of them calls `curveRecheck` -- so between the scan and the flush
// that writes, the classifier went back to reading a setting that still says
// true. Observed on 2.0.77 as the flip handler announcing a flip nobody made and
// re-queueing the whole save, on the very run this suite passed.
//
// It is read by `curvedExitsAllowed` and it is false in every save this pass is
// not about, which is every save built on 0.3.3 or later.
var curveForcedOff bool

// noteCurveLegacy records one cluster adopted on its curve-free reading.
//
// IT DOES NOT TOUCH THE CLASSIFIER, and that is the whole of the count. The
// first cut moved the cache here, which reads as obvious -- the decision is
// taken, so honour it from now on -- and it makes the answer wrong by
// construction: with the rule off the curve arm produces no edge, so the SECOND
// old-rule cluster in the same rebuild adopts on the first comparison, is never
// noted, and a save with two of them tells the player about one and pings one.
// The scan has to read the whole save under the classifier the save was BUILT
// with, which is what curveScanDone is for.
func noteCurveLegacy(root uint32) {
	curveWriteOwed = true
	for i := range curveLegacy {
		if curveLegacy[i].root == root {
			return
		}
	}
	curveLegacy = append(curveLegacy, annNote{root: root})
}

// curveScanDone moves the classifier, once, at the end of the rebuild's
// inspection loop.
//
// THE MODE MOVES HERE AND THE SETTING WAITS FOR THE FLUSH, and the two halves
// have different reasons. The MODE has to move before rebuildFromWorld's own
// flush, because that flush is what compiles every cluster the scan could not
// adopt and it must compile them under the rule this save is keeping. The WRITE
// may not happen here at all: it raises `on_runtime_mod_setting_changed`
// synchronously, and a rebuild's write would land that handler inside the
// rebuild's own drain.
//
// IT SETS A FLAG AND NOT THE CACHE, and that is the one thing about this that
// was measured rather than reasoned. See curveForcedOff.
func curveScanDone() {
	if curveWriteOwed {
		curveForcedOff = true
	}
}

// settleCurveMode writes the setting and says so, once, from the informed flush.
//
// It is beside settleEdgeMode in flush() and for the same three reasons: the
// write re-enters this guest synchronously, the drain is over, and the carry
// transaction has closed. One bool test on every flush of every save this is not
// about, which is every save built on 0.3.3 or later.
func settleCurveMode() {
	if !curveWriteOwed || rebuildingFromWorld {
		return
	}
	curveWriteOwed = false
	n, _ := gatherAnnounced(curveLegacy)
	curveLegacy = curveLegacy[:0]
	// n IS ZERO ONLY IF EVERY NOTED CLUSTER WAS DESTROYED IN THE ONE TICK
	// between the rebuild and this flush, and then the classifier stays off for
	// the rest of the session with nothing written. That is the direction that
	// cannot surprise a standing machine -- there is none left to surprise --
	// and the next load reads the setting fresh, finds nothing to detect, and
	// comes up with the rule on.
	// THE FOLD IS ASKED EVEN THOUGH THE CACHE HAS ALREADY MOVED, and the two are
	// not the same question: `curveCache` is what the classifier used for the
	// rest of the rebuild, and this is whether the SAVE has to be told.
	//
	// ITS SettingOff ARM IS A SHAPE GUARD AND NOT THE LATCH, which is worth
	// saying because it reads like the latch. A note is recorded only for an
	// edge the CURVE ARM produced, and the curve arm is behind
	// `curvedExitsAllowed` -- so a save whose setting is already Off produces no
	// curve edges, no failed adoption and no note, and `n` is zero before this
	// fold is reached at all. The latch is one level up, in the classifier gate,
	// and the fold agreeing with it is what keeps the two from drifting. Every
	// state of it is proved by go test ./edgemode/.
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

// ---------------------------------------------------------------------------
// The curve-free reading
// ---------------------------------------------------------------------------

// edgeBase is the edge list with every curve-produced edge taken out.
//
// A SECOND BUFFER RATHER THAN A FILTER IN PLACE: `edgeBuf` is what the caller is
// still holding, and the comparison below needs both readings at once. High
// water like every other buffer here, so it costs nothing after the first load
// that uses it and nothing at all in a save that never does.
var edgeBase []plan.Edge

// curveFreeEdges is the edge list without the edges the curve arm produced, and
// how many were taken out.
func curveFreeEdges(edges []plan.Edge) ([]plan.Edge, int) {
	edgeBase = edgeBase[:0]
	removed := 0
	for i := range edges {
		if i < len(edgeCurved) && edgeCurved[i] {
			removed++
			continue
		}
		edgeBase = append(edgeBase, edges[i])
	}
	return edgeBase, removed
}

// multiOver is "does any tile in this edge list carry two edges", asked of a
// list rather than read out of classifyEdges' own counters.
//
// It has to be recomputed rather than reused: the counters describe the reading
// WITH the curve edges in it, and a cluster adopted on the curve-free reading
// would otherwise be announced as multi-edge when the second belt on that tile is
// exactly the one this pass has just declined to see. The loop is quadratic in an
// edge list bounded by plan.MaxPorts and it runs once per adopted cluster on one
// load in the history of a save.
func multiOver(edges []plan.Edge) bool {
	for i := range edges {
		n := 0
		for j := range edges {
			if edges[j].TileX == edges[i].TileX && edges[j].TileY == edges[i].TileY {
				n++
			}
		}
		if n > 1 {
			return true
		}
	}
	return false
}
