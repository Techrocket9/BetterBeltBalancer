package main

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// THE SETTLE: A BURST OF BUILDS OR MINES ON A BALANCER IS COMPILED ONCE, NOT
// ONCE PER TICK.
//
// `fk.Defer()` bounds a recompile at one per cluster per TICK, and a player
// dragging parts along a row of belts places one part every few ticks, so a
// twenty-part drag was twenty teardown-and-rebuilds of a network one part
// larger each time. A rebuild of a running 4x4 is about 27 ms on 0.3.7 and a
// 16x16 about 120 ms (bench/baselines/RESULTS.md, "What one edit costs"), so
// the drag was a stutter per tile. That was a mod-portal report. A drag of
// mines taking a balancer apart, and robots building or deconstructing one from
// a blueprint, are the same shape.
//
// ONLY A GESTURE IS EVER HELD: a build or a mine by a player, a construction
// robot or a space platform (`isGesture`). A burst is held only while
// everything queued since the last flush was queued from one of those six
// dispatches; one mark from anywhere else -- a script, a clone, a rotation, a
// death, or the mod's own work inside a flush -- and the queue drains on the
// next tick as it always has. So a mod or a scenario that edits balancers by
// script keeps the next-tick contract every suite in this repository is
// written against (the first cut held scripted edits too, and the `m1` suite,
// which edits by script ten ticks apart, saw its part sprites a phase late).
//
// The rule, in the order onDeferred asks it:
//
//   - AN ISOLATED BUILD IS UNCHANGED. A deferred flush with nothing compiled in
//     the last burstWindow ticks drains on the tick after the edit, exactly as
//     before, so a belt laid beside a balancer connects on the next tick.
//   - AN EDIT THAT LANDS WITHIN burstWindow TICKS OF A FLUSH THAT DID WORK OPENS
//     A BURST, AND SO DOES A MINE WITH NO BUILD BESIDE IT. The queues are
//     kept, the one-shot is re-armed, and nothing is torn down: the standing
//     network keeps running the shape it had. A mine is not an edit whose
//     answer a player waits for -- its tile is cleared on the next tick either
//     way (below) -- and the
//     first mine of a drag that took a running balancer apart was the drag's
//     dearest tick: 89 ms of a 10->10 rebuilt with 819 items carried, measured
//     by the `curs` suite's removal drag.
//   - A BURST IS COMPILED once settleQuiet ticks pass with nothing new queued,
//     or settleCap ticks after it opened, whichever comes first. A drag is
//     therefore two rebuilds -- the first part's, and the finished shape's --
//     however long it is.
//   - ROBOTS AND PLATFORMS GET A SECOND, NOT A FIFTH OF ONE: robotWindow and
//     robotQuiet stand in for burstWindow and settleQuiet whenever a robot's
//     or a platform's build or mine is in the queue.
//   - A PART MINED INSIDE A BURST TAKES ITS OWN TILE WITH IT, on the tick after
//     the mine (cutRemoved): the visible interfaces standing on that one tile
//     are drained and destroyed, and what they held goes to whoever mined it,
//     or to the floor, exactly as a removal's remainder does. The rest of the
//     network keeps running until the burst is compiled, and its items are
//     carried into the successors then, so taking a balancer apart by hand no
//     longer spills at every shrink. Nothing stands on a tile a player has
//     just cleared, so the next thing they place there does not collide.
//   - ANY OTHER PENDING WORK DRAINS AT ONCE, burst or not: a legacy
//     conversion, an insert probe, an over-limit revert.
//
// What is held is only the two queues the per-entity events fill. Every
// synchronous flush -- the audit marker, a clone or surface reconcile, a
// priority keypress that flushes inline -- drains them as before and ends the
// burst.
//
// DETERMINISTIC BY CONSTRUCTION: the clock is `game.tick`, the counters are
// guest state, and nothing depends on wall time or on iteration order. The
// state survives a save inside the guest heap; a rebuilt guest starts with it
// zeroed and the world scan queues nothing, so there is nothing to hold. A
// held burst carries NO items across ticks: a cut tile's pool is settled in
// the tick it is cut, and everything else is still on a belt.
//
// One host call (`game.tick`) per deferred tick, and only on a tick that has
// queued work. An idle guest still registers nothing and is called for nothing.
const (
	// burstWindow is how soon after a flush that compiled something a new edit
	// counts as the same gesture. A character running while dragging lays a
	// tile every 6 or 7 ticks, a stationary cursor drag faster. It is kept
	// TIGHT on purpose: a single click is a gesture of its own and its answer
	// (a port, a refusal and its hand-back) belongs on the next tick. At 30
	// the `curs` suite's separate clicks 15 and 29 ticks after an earlier one
	// were held, and four of its samples read the world a third of a second
	// early.
	burstWindow = 12
	// settleQuiet is how long a burst waits with nothing new queued before it is
	// compiled: a fifth of a second.
	settleQuiet = 12
	// settleCap is the longest any held edit waits, whatever keeps arriving.
	settleCap = 600
	// robotWindow and robotQuiet replace both windows above when anything in
	// the queue was built or mined by a construction robot or a platform. No
	// cursor is waiting on a robot, and robots arrive in waves: measured by the
	// bench's robot phase (bench/run.sh --scenario saturated --hitch -n 4 -k 4
	// --ticks 2700), eighteen of twenty parts landed within 92 ticks and the
	// last two 37 and 48 ticks after the burst had compiled, each a rebuild of
	// the whole running network. A second is long enough for the jitter inside
	// a wave; a round trip back to a roboport is not covered and is not meant
	// to be.
	robotWindow = 60
	robotQuiet  = 60
)

var (
	// queueTouches counts every markDead/markLive call, deduplicated or not, so
	// a deferred tick can tell whether anything was queued since the last one
	// without remembering what.
	queueTouches uint32
	seenTouches  uint32
	lastTouch    uint64

	// The tick of the last deferred flush that drained one of the two queues.
	lastWorkFlush uint64
	haveWorkFlush bool

	holding   bool
	holdStart uint64

	// cutQueue is every part removed since the last deferred tick: its tile,
	// the force it was built by, and the root its cluster had. Read by
	// cutRemoved when the tick is held, and emptied by every deferred tick and
	// every flush either way.
	cutQueue []cutTile

	// gestureDispatch is true for the length of a gesture's dispatch (main.go's
	// onEvent); flushDepth for the length of a flush. foreignTouch records that
	// something was queued outside the one or inside the other since the last
	// flush.
	gestureDispatch bool
	flushDepth      int
	foreignTouch    bool

	// buildDispatch is true for the length of a gesture BUILD's dispatch, and
	// buildTouch records that one queued something since the last flush. A
	// mine with a build beside it -- a fast replace, a belt laid where a part
	// was -- is answered like a build.
	buildDispatch bool
	buildTouch    bool

	// robotDispatch and robotTouch are the same pair for a robot's or a
	// platform's build or mine.
	robotDispatch bool
	robotTouch    bool

	// settleTick is game.tick as read by the deferred tick that is running,
	// valid only when settleRead is set.
	settleTick uint64
	settleRead bool
)

type cutTile struct {
	k           key
	force, root uint32
}

// isGesture is the six dispatches whose queued work a burst may hold, whether
// the one in hand is a build, and whether a robot or a platform made it.
func isGesture(id uint32) (gesture, build, robot bool) {
	switch id {
	case fkapi.EventOnBuiltEntity:
		return true, true, false
	case fkapi.EventOnRobotBuiltEntity, fkapi.EventOnSpacePlatformBuiltEntity:
		return true, true, true
	case fkapi.EventOnPlayerMinedEntity:
		return true, false, false
	case fkapi.EventOnRobotMinedEntity, fkapi.EventOnSpacePlatformMinedEntity:
		return true, false, true
	}
	return false, false, false
}

// noteQueued is called by markDead and markLive.
func noteQueued() {
	queueTouches++
	if !gestureDispatch || flushDepth > 0 {
		foreignTouch = true
	} else {
		if buildDispatch {
			buildTouch = true
		}
		if robotDispatch {
			robotTouch = true
		}
	}
}

// noteRemoved is called by removePart for every part that leaves the registry.
// Recorded, not acted on: whether its tile is cut on its own depends on whether
// the next deferred tick is held.
func noteRemoved(k key, force, root uint32) {
	cutQueue = append(cutQueue, cutTile{k: k, force: force, root: root})
}

// cutRemoved takes down the visible interfaces on every tile a part was
// removed from since the last deferred tick, when that tick is held.
//
// OUTSIDE A CARRY TRANSACTION, on purpose: the pool closePool settles here is
// a removal's -- the miner's pocket first (the claim noteMinedByPlayer made on
// this very tile), then the floor -- and settling it does NOT reset the claims,
// which the flush that ends the burst still needs for the rest of the machine.
// A tile with nothing of ours on it costs one find per interface name and no
// pool: closePool pops an empty one.
func cutRemoved() {
	for i := range cutQueue {
		c := cutQueue[i]
		vis, ok := surfaceByIndex(c.k.s)
		if !ok {
			continue
		}
		openPool(c.root, netInfo{surf: c.k.s, force: c.force,
			x0: c.k.x, y0: c.k.y, x1: c.k.x, y1: c.k.y}, false)
		sweep(vis, c.k.x, c.k.y, c.k.x, c.k.y, true)
		closePool()
	}
	cutQueue = cutQueue[:0]
}

// holdFlush decides, inside fk_on_deferred, whether this tick keeps the queue.
func holdFlush() bool {
	settleRead = false
	if len(deadRoots) == 0 && len(liveRoots) == 0 {
		return false
	}
	now, err := fkapi.Game.Tick()
	if err != nil {
		return false
	}
	settleTick, settleRead = now, true
	if queueTouches != seenTouches {
		seenTouches = queueTouches
		lastTouch = now
	}
	if foreignTouch || len(legacyBuildQueue) != 0 || len(probeQueue) != 0 || len(limPending) != 0 {
		return false
	}
	window, quiet := uint64(burstWindow), uint64(settleQuiet)
	if robotTouch {
		window, quiet = robotWindow, robotQuiet
	}
	if !holding {
		// A mine opens a burst on its own: its tile is cut on this tick
		// whatever happens, so holding the rest costs the player nothing they
		// can see, and the first mine of a drag is otherwise its dearest
		// rebuild -- the whole running machine, carried. Not with a build
		// beside it: a belt fast-replaced onto a part is a port the player is
		// waiting for, and the `curs` suite's column gesture says so.
		mineOnly := len(cutQueue) != 0 && !buildTouch
		if !mineOnly && (!haveWorkFlush || now-lastWorkFlush > window) {
			return false
		}
		holding = true
		holdStart = now
	}
	if now-lastTouch < quiet && now-holdStart < settleCap {
		cutRemoved()
		return true
	}
	return false
}

// settled records a deferred flush that drained one of the two queues. It is
// called only when holdFlush read the tick, which is exactly when there was
// something to drain.
func settled() {
	if settleRead {
		lastWorkFlush, haveWorkFlush = settleTick, true
	}
}

// endBurst is the part of every flush, synchronous or deferred, that the
// settle cares about: whatever was held has now been compiled.
func endBurst() {
	holding = false
	foreignTouch = false
	buildTouch = false
	robotTouch = false
	cutQueue = cutQueue[:0]
	seenTouches = queueTouches
}
