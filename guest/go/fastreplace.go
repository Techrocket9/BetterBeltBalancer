package main

// FAST REPLACE: a balancer part goes over a belt, and a belt goes over a
// balancer part.
//
// `bbb-balancer-part` carries `fast_replaceable_group = "transport-belt"`
// (mod-data/prototypes/entity.lua), which is base's own group for every
// transport belt, underground belt, splitter and lane splitter. Holding a part
// over a belt now replaces it the way holding a splitter over one does: the
// belt and whatever it was carrying go to the player, and the part takes the
// tile. That is the whole feature and it is a DATA-STAGE line -- the forward
// direction needs no guest code at all, because the engine mines the belt
// before it creates the part and the compiler re-reads the world a tick later
// (compile.go, "THE REMOVAL WINDOW IS GONE").
//
// THE REVERSE DIRECTION IS WHAT THIS FILE IS FOR, and it is not optional.
// A fast-replaceable group is symmetric: the same line that lets a part replace
// a belt lets a BELT REPLACE A PART. Measured on 2.0.77 with a script standing
// in for the gesture -- `create_entity{name = "express-transport-belt",
// fast_replace = true}` on a part's tile:
//
//	the part is destroyed, the part ITEM is spilled, the belt is created,
//	and NOT ONE EVENT IS RAISED FOR THE PART. No on_player_mined_entity, no
//	script_raised_destroy, no on_entity_died. The only event in the whole
//	dispatch is the BUILD event for the belt.
//
// So without the check below the registry keeps a PHANTOM part: a tile it calls
// a balancer part which holds a player's belt. Measured on the guest before this
// file existed -- three parts became two in the world and the audit went on
// reporting `parts=3 drift=0 unbuilt=0`, because the audit compares edge
// fingerprints and a phantom tile is INTERIOR, so the belt standing on it is
// never classified and the fingerprint never moves. The cluster is then wrong
// for the rest of the session: the belt a player laid does nothing at all, and
// the tile is inside the box every teardown of that cluster sweeps.
//
// THE CHECK: on a belt-connectable APPEARING, is its own tile a registered part
// tile, and if so is that part still standing? A tile lookup answers the first
// (guest memory, no host call, and it is the centre of the 5x5 neighbourhood
// `onNeighbour` walks anyway), and only a HIT asks the world. In ordinary play
// the answer to the first question is always no -- the part's collision mask
// carries `transport_belt`, so the only way a belt reaches a part's tile is by
// replacing the part -- which is why this costs one map probe per belt built
// anywhere on the map and nothing else.
//
// IT IS CORRECT UNDER EVERY EVENT ORDERING, which matters because a headless
// run cannot tell us what a PLAYER's fast replace raises (there is no player in
// a `--create`; see the edge suite's probe). The three possibilities:
//
//	no mine event, which is what a script does -- the check finds the part
//	gone and removes it, crediting the builder;
//	the mine event FIRST -- `onPart` has already removed the part, so the
//	tile lookup misses and this returns without a host call;
//	the build event first, then the mine -- this removes the part, and
//	`removePart` on an unregistered tile is a no-op, so the mine that
//	follows changes nothing.
//
// WHO IS CREDITED. `builtBy`, the player who placed the belt: a fast replace
// hands the replaced entity to the player doing the replacing, so if the
// shrinking machine cannot take back everything it was holding, that player is
// the one the overflow belongs to. It is the miner's pocket's own rule
// (carry.go) reached through the other door, and `RemovePartMinedBy` is
// literally the same call `onPart` makes for a mine.
//
// WHAT IS NOT COVERED, and it is script-only. A belt cannot replace a part that
// carries an EDGE INTERFACE, because `bbb-linked-belt` is a belt-connectable of
// its own standing on that same tile and it blocks the placement --
// `can_fast_replace` is false there, measured, so the engine refuses a player
// outright. A SCRIPT that ignores `can_fast_replace` and calls `create_entity`
// anyway gets the part destroyed and no belt created, with no event of any kind:
// nothing appears, so nothing below can fire. That is the failure envelope
// (CLAUDE.md) exactly as it already reads for a mod calling `entity.destroy()`
// on a part, and it is not new.

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// frPos is the one position this file passes to the host, package level so the
// check allocates nothing on the one path that reaches a host call.
var frPos fkapi.MapPosition

// reapFastReplaced removes a part that a belt-connectable has just replaced.
//
// Called from onEventBody for an APPEARANCE and from nowhere else, BEFORE
// `onNeighbour`, so that the neighbourhood walk that follows sees the registry
// the removal left rather than the one it found.
func reapFastReplaced(surf uint32, tx, ty int32, builtBy uint32) {
	k := key{s: surf, x: tx, y: ty}
	if _, ok := index[k]; !ok {
		// The whole cost in ordinary play: one point query against a map that
		// is only ever point-queried.
		return
	}
	s, ok := surfaceByIndex(surf)
	if !ok {
		return
	}
	frPos.X = float64(tx) + 0.5
	frPos.Y = float64(ty) + 0.5
	o, err := s.FindEntity(fkapi.OfString(PartName), frPos)
	if err != nil || o != nil {
		// Still standing, so this is not a replace: a script that built a
		// colliding belt-connectable on top of a part (which `create_entity`
		// permits and a player cannot do), or a query that failed. Either way
		// the registry is right and must not be edited on a guess.
		return
	}
	if !RemovePartMinedBy(k, builtBy) {
		return
	}
	if verboseLog {
		logStart("a belt-connectable fast-replaced the part at ")
		logI(tx)
		logS(",")
		logI(ty)
		logS("; unregistered it")
		if builtBy != 0 {
			logS(" for player ")
			logU(builtBy)
		}
		logEnd()
	}
	logState()
	requestFlush()
}
