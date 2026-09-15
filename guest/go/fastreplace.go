package main

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// FAST REPLACE: a balancer part goes over a belt, and a belt goes over a
// balancer part.
//
// `bbb-balancer-part` carries `fast_replaceable_group = "transport-belt"`
// (guest/go/data/entity.go), which is base's own group for every
// transport belt, underground belt, splitter and lane splitter. Holding a part
// over a belt now replaces it the way holding a splitter over one does: the
// belt and whatever it was carrying go to the player, and the part takes the
// tile. That is the whole feature and it is a DATA-STAGE line: the engine mines
// the belt before it creates the part and the compiler re-reads the world a tick
// later (compile.go, "THE REMOVAL WINDOW IS GONE").
//
// THE FORWARD DIRECTION NEEDED NO GUEST CODE UNTIL 2026-09-14, and the one case
// that changed it is a REFUSAL. On Factorio 2.1 a part dropped into the middle
// of a belt line takes the belt behind it as an input and the belt ahead as an
// output, which is two belts on one part, so sedge.go refuses the cluster and
// limit.go hands the part back -- and the belt the engine mined on the way in
// was gone for good, leaving a hole in the line the player never asked for. The
// second half of this file is what puts it back; see "The belt a refused fast
// replace destroyed".
//
// THE REVERSE DIRECTION IS WHAT THE FIRST HALF OF THIS FILE IS FOR, and it is a
// SCRIPT'S path rather than a player's. A fast-replaceable group is symmetric:
// the same line that lets a part replace a belt lets a BELT REPLACE A PART.
// Measured on 2.0.77 with a script standing in for the gesture --
// `create_entity{name = "express-transport-belt", fast_replace = true}` on a
// part's tile:
//
//	the part is destroyed, the part ITEM is spilled, the belt is created,
//	and NOT ONE EVENT IS RAISED FOR THE PART. No on_player_mined_entity, no
//	script_raised_destroy, no on_entity_died. The only event in the whole
//	dispatch is the BUILD event for the belt.
//
// A REAL PLAYER'S BELT-OVER-PART IS NOT THAT, and it is measured too, on 2.0.77
// on 2026-09-14 by driving `build_from_cursor` against a save with a connected
// player in it: the engine raises `on_pre_player_mined_item` and
// `on_player_mined_entity` for the PART, with the part item in the buffer,
// BEFORE `on_built_entity` for the belt. So a player's gesture is handled by the
// ordinary removal path -- `cluster N dissolved, mined by player P` -- and this
// file's check finds the tile already unregistered and returns. The "mine event
// FIRST" ordering listed below is the one a player takes; it is not a
// hypothetical.
//
// So the check below is what stops a SCRIPT leaving a PHANTOM part: a tile the
// registry calls a balancer part which holds somebody's belt. Measured on the
// guest before this file existed -- three parts became two in the world and the
// audit went on reporting `parts=3 drift=0 unbuilt=0`, because the audit
// compares edge fingerprints and a phantom tile is INTERIOR, so the belt
// standing on it is never classified and the fingerprint never moves. The
// cluster is then wrong for the rest of the session: the belt does nothing at
// all, and the tile is inside the box every teardown of that cluster sweeps.
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
// IT IS CORRECT UNDER EVERY EVENT ORDERING, and two of the three are now known
// rather than enumerated for safety:
//
//	no mine event, which is what a SCRIPT does -- the check finds the part
//	gone and removes it, crediting the builder;
//	the mine event FIRST, which is what a PLAYER does (measured 2026-09-14) --
//	`onPart` has already removed the part, so the tile lookup misses and this
//	returns without a host call;
//	the build event first, then the mine -- nothing observed produces this,
//	and it costs nothing to be right about: this removes the part, and
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
// EVERY PART IS REPLACEABLE, INCLUDING ONE CARRYING AN EDGE INTERFACE, and that
// is what the group on `bbb-linked-belt` bought (guest/go/data/hidden.go). A part
// with an edge on it holds TWO colliding entities and the engine wants both of
// them in the group; with only the part in it `can_fast_replace` was false there,
// so the one gesture a player would try on a balancer they were actually using
// was the one that did not work. The replace destroys both, and this file's check
// is unchanged by it: it keys on the PART having gone, and the interface goes
// with the part rather than instead of it.
//
// TWO CONSEQUENCES, BOTH ON THE PLAYER'S SIDE AND BOTH STATED RATHER THAN
// HIDDEN.
//
// A belt laid where an edge part was is an edge of whatever part is next to it
// ALONG THE SAME AXIS -- the removed part's tile is orthogonally adjacent to it,
// which is what makes this different from mining a part, where the belts simply
// stop being anybody's. On Factorio 2.1 that neighbour may already have its one
// belt, and then the cluster the removal leaves cannot be built. The refusal is
// correct and the TEARDOWN IS NOT THE COMPILE'S: `removePart` marks the old root
// dead, so `flushDead` has brought the network down before `flushLive` discovers
// what is left, and the machine's contents spill. That is the shape "The merge
// that would be over the limit" describes for a MERGE, met one door along, and
// it is not sparable the way a merge is -- the interface on the replaced tile is
// already gone, so the standing network is damaged whatever this guest does.
// The `edge` suite's frepd arm is the measurement.
//
// And a balancer is no longer immune to a belt DRAG. Every part a drag crosses
// is replaced, exactly as a drag across a row of splitters replaces those. The
// parts come back as items and the machine recompiles around what is left.

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
	// `findOnTile`, not `find_entity`, and HERE the difference edits the
	// registry: a bare-name `find_entity` reads NORMAL QUALITY ONLY
	// (findpart.go), so an uncommon part that is still standing came back nil
	// and was unregistered as replaced -- right for the wrong reason on a real
	// fast replace (the part really is gone, at any quality), and wrong for a
	// script that builds a colliding belt-connectable on top of one.
	if _, standing, err := findOnTile(s, PartName, tx, ty); err != nil || standing {
		// Still standing, so this is not a replace: a script built a colliding
		// belt-connectable on top of a part (which `create_entity` permits and
		// a player cannot do), or a query that failed. Either way the registry
		// is right and must not be edited on a guess.
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

// ---------------------------------------------------------------------------
// The belt a refused fast replace destroyed
// ---------------------------------------------------------------------------

// A part dropped into the MIDDLE of a belt line is refused, and the belt it
// replaced was gone for good until 2026-09-14. That is the mod author's own
// field report and it is the forward direction's one sharp edge.
//
// WHAT THE PLAYER DID. Hold a part over a belt with a belt behind it and a belt
// ahead of it, and click. The engine fast-replaces -- it has already decided,
// from `fast_replaceable_group` at the data stage, and nothing at runtime can
// stop it. The part's tile now carries two edges, the belt behind pointing in
// and the belt ahead pointing away, so on Factorio 2.1 sedge.go refuses the
// cluster, limit.go's `revertOne` mines the part back into the inventory, and
// the player is left holding a part, a belt and the belt's cargo, with a HOLE in
// a line that was working a tick ago. Every one of those steps is correct on its
// own; what was missing is the last one.
//
// WHAT IT DOES NOW. The refusal puts the belt back where it was, and CHARGES the
// player the one belt item the engine's own mine refunded them a tick earlier.
// The charge is not bookkeeping for its own sake: without it the gesture prints
// belts, since the engine refunds one every time and the restore would hand one
// back for free. `RemoveItem` returning 0 -- a full inventory, a belt already
// spent, an item spilled to the ground -- means there is nothing to pay with and
// the gap stays, with an alert saying so.
//
// THE BELT COMES BACK EMPTY, and where the cargo went is measured rather than
// reasoned: the mine's own buffer carries it into the player before any of this
// guest's code runs. Six iron plates on the replaced belt came out as six iron
// plates in the player's inventory, 97 tracked before and 97 after, none on the
// ground (2.0.77, 2026-09-14). So the items are conserved and they are in the
// player's hands, which is where a mined belt's cargo belongs; reinserting them
// would mean taking them out of an inventory this mod never put them in.
//
// HOW A FAST REPLACE IS RECOGNISED, AND THE ORDERING IT RESTS ON, MEASURED.
// This guest sees the mine and the build; what it cannot see from either is that
// they are one gesture, because a mine and a build at one tile in one tick by
// one player is otherwise indistinguishable from a player mining a belt and then
// building on the tile. `on_pre_build` is what ties them together: it is
// subscribed, its (tick, player, tile) is written into one slot, and a mine that
// matches that slot is a fast replace.
//
// MEASURED ON 2.0.77 ON 2026-09-14, five identical runs, by loading a save with
// a connected player in it under `--benchmark` and driving
// `player.build_from_cursor{position = ...}` -- a genuine cursor build, which is
// what the estate has never had. In ONE tick, in this order, all before
// `build_from_cursor` returns:
//
//	on_pre_build            (position 33.5,-45.5, no entity, the cursor's
//	                         direction, player 1)
//	on_pre_player_mined_item (transport-belt, unit 1018)
//	on_player_mined_entity  (the same belt, still valid, buffer holding one
//	                         transport-belt plus whatever it was carrying)
//	on_player_mined_item
//	on_built_entity         (bbb-balancer-part, unit 1021)
//
// `on_object_destroyed` for the belt arrives AFTER the build event, which is why
// it is useless as the signal. And a build the engine REFUSES raises nothing at
// all, `on_pre_build` included, so a rejected click cannot leave a stale slot.
//
// IT FAILS SAFE ANYWAY, which is worth keeping now that the ordering is known:
// a slot that has not been written, or written after the mine, simply does not
// match, `noteReplacedBelt` is never called, and the outcome is what shipped
// before this file learned any of it -- the part comes back and the gap stays.
//
// WHAT IT COSTS. On a headless run with no player, nothing at all: `on_pre_build`
// carries a player index and there is none, so the slot is never written and
// every match fails on the zero player before it compares anything.
//
// On a real server it is one payload-only dispatch per tile a player's build
// gesture touches -- no host call, no allocation, one struct assignment -- plus
// the read below on every tile where that gesture REPLACED something. That last
// is the one that multiplies, because a belt-over-belt upgrade drag is a fast
// replace too and this file cannot know until the build event lands that no part
// is coming: three host calls per tile after the first of a given prototype, and
// no allocation at all, which is what the memo is for.

// qualityNormal is the quality a bare name means. A belt at it needs no
// `quality` key at all, so the common case copies no string out of the host.
const qualityNormal = "normal"

// beltSpec is everything needed to put one 1x1 belt-connectable back on the tile
// it was replaced on.
//
// SCALARS AND STRINGS, no handle, for buildNote's own reason: the mine happens
// in one dispatch and the restore in the flush a tick later, and nothing the
// host lends survives that gap -- the quality is read as a NAME rather than kept
// as the prototype handle legacy.go passes, because that file's create runs in
// the same call as its destroy and this one does not.
//
// COMPARABLE, because buildNote carries one and `noteBuiltByPlayer` dedupes with
// `==`.
//
// Three shapes reach it and no others: a transport belt, an underground belt end
// and a lane splitter are the only things `can_fast_replace` answers true for
// under a part (a splitter is two tiles wide and a loader is in another group),
// and all three are 1x1, so the tile centre is the position and a direction is
// the whole of the orientation -- plus, for an underground, which end it was.
type beltSpec struct {
	name    string
	item    string // what places it, so the restore can be charged for
	quality string // "" for normal
	dir     uint32
	under   bool // an underground-belt end, which needs a `type` as well
	outEnd  bool
}

func (b beltSpec) known() bool { return b.name != "" }

// replacedBelt is one belt a player's own build gesture destroyed this tick,
// waiting for the build event that says what replaced it.
type replacedBelt struct {
	tick   uint64
	player uint32
	s      uint32
	x, y   int32
	belt   beltSpec
}

// preBuildSlot is the last `on_pre_build` this guest was told about.
//
// ONE SLOT AND NOT A QUEUE: the engine raises `on_pre_build` once per placed
// entity and the mine that a fast replace produces follows it immediately, so a
// drag across a belt line is pre_build, mine, built, pre_build, mine, built --
// each pair adjacent. The slot is matched on the tile as well as on the tick and
// the player, so a stale one cannot answer for a different tile.
type preBuildSlot struct {
	tick   uint64
	player uint32
	x, y   int32
}

var (
	preBuild preBuildSlot

	// replacedBelts is this tick's worth, truncated by every flush exactly as
	// buildNotes is and for the same reason -- a record outlives the event that
	// made it by one flush, which is the flush that could have used it.
	replacedBelts []replacedBelt

	// The reusable destination for `items_to_place_this`, so asking what places
	// a belt allocates nothing after the first probe in a save.
	itemsToPlace []fkapi.ItemToPlace

	// ONE PROTOTYPE'S WORTH OF MEMO, and it is what keeps a DRAG affordable.
	// A player upgrading a belt line with the cursor fast-replaces a belt with a
	// belt, tile after tile, and every one of those is a mine this file has to
	// read because it cannot know until the build event lands that no part is
	// coming. What it reads per entity is a direction and a quality; what it
	// reads per PROTOTYPE -- the item that places it, and whether it is an
	// underground end -- is the same answer for every tile of one drag, and
	// asking again would be two host calls and a fresh Go string each time,
	// which under `-gc=leaking` is a slope on a path a player can repeat a
	// thousand times in a gesture.
	memoName, memoItem string
	memoUnder          bool
)

// notePreBuild records the click. Payload only: no host call, no allocation.
func notePreBuild(ev fkapi.OnPreBuild) {
	preBuild.tick = ev.Tick
	preBuild.player = ev.PlayerIndex
	preBuild.x = floorTile(ev.Position.X)
	preBuild.y = floorTile(ev.Position.Y)
}

// isFastReplace reports whether a mine at (x, y) is the engine clearing a tile
// for the build the same player just asked for.
func isFastReplace(tick uint64, player uint32, x, y int32) bool {
	return player != 0 && preBuild.player == player && preBuild.tick == tick &&
		preBuild.x == x && preBuild.y == y
}

// noteReplacedBelt reads the belt about to be destroyed.
//
// EVERYTHING IS READ HERE BECAUSE THERE IS NOWHERE ELSE TO READ IT. The entity
// is valid for this dispatch and no longer; the flush that decides whether the
// part is refused runs a tick later, against a tile the belt has left.
//
// Nothing is recorded unless every question is answered, and the sharpest of
// them is the ITEM: a belt that cannot be paid for must not be restored, so a
// prototype with no `items_to_place_this` records nothing rather than recording
// something the restore would have to give away.
func noteReplacedBelt(s uint32, x, y int32, player uint32, tick uint64,
	name string, ent fkapi.LuaEntity) {
	// A RECORD FROM AN EARLIER TICK IS DROPPED HERE, NOT ONLY BY THE FLUSH. The
	// flush truncates this slice, but a flush is requested only by an edit near
	// a cluster, and the common producer of these records is nowhere near one:
	// a belt-over-belt upgrade drag across an empty base is a fast replace per
	// tile, each of which lands here and then reaches onNeighbour, which finds
	// nothing and asks for no flush. Without this the slice would grow by one
	// record per dragged tile until somebody edited a balancer. A record can
	// only ever be read by a build in its own tick, so anything older is dead.
	if len(replacedBelts) > 0 && replacedBelts[0].tick != tick {
		replacedBelts = replacedBelts[:0]
	}
	b := beltSpec{name: name}
	d, err := ent.Direction()
	if err != nil {
		return
	}
	b.dir = d
	// NO FORCE IS READ. The belt comes back on the force of the PART that
	// replaced it, which is the note's own -- a player cannot fast-replace an
	// entity of another force, so the two are the same, and taking it from the
	// note costs no host call at all.
	//
	// `name_is` first and `name` only on a miss: a quality prototype's name is a
	// host string and the answer is "normal" on every belt in a base game.
	if q, qErr := ent.Quality(); qErr == nil {
		qp := fkapi.LuaQualityPrototype{Object: q}
		if normal, nErr := qp.NameIs(qualityNormal); nErr == nil && !normal {
			qn, nameErr := qp.Name()
			if nameErr != nil {
				return
			}
			b.quality = qn
		}
	}
	if name != memoName {
		// `type_is` rather than `type`, for the reason classifySide reads the
		// curve arm's tiles that way: a predicate answers on the host and copies
		// no string.
		under, uErr := ent.TypeIs("underground-belt")
		if uErr != nil {
			return
		}
		p, pErr := ent.Prototype()
		if pErr != nil {
			return
		}
		items, iErr := fkapi.LuaEntityPrototype{Object: p}.ItemsToPlaceThisInto(itemsToPlace)
		if iErr != nil {
			return
		}
		itemsToPlace = items
		if len(items) == 0 {
			// Nothing places this, so nothing can pay for putting it back. The
			// memo is deliberately not armed: a name that answers this way costs
			// two host calls every time and there is no answer worth keeping.
			return
		}
		memoName, memoItem, memoUnder = name, items[0].Name, under
	}
	b.item, b.under = memoItem, memoUnder
	if b.under {
		out, tErr := ent.BeltToGroundTypeIs("output")
		if tErr != nil {
			return
		}
		b.outEnd = out
	}
	replacedBelts = append(replacedBelts, replacedBelt{
		tick: tick, player: player, s: s, x: x, y: y, belt: b})
}

// takeReplacedBelt answers the build event: what, if anything, the engine
// cleared off this tile for this player this tick.
//
// The record is left in place rather than removed -- the slice is truncated by
// the flush either way, and a build event that arrives twice for one entity (a
// mod raising built after the engine already did) would otherwise attach the
// belt to the first note and nothing to the second, which is the one shape
// `noteBuiltByPlayer`'s dedupe cannot then collapse.
func takeReplacedBelt(s uint32, x, y int32, player uint32, tick uint64) beltSpec {
	for i := range replacedBelts {
		r := &replacedBelts[i]
		if r.tick == tick && r.player == player && r.s == s && r.x == x && r.y == y {
			return r.belt
		}
	}
	return beltSpec{}
}

// forgetReplacedBelts ends the tick, beside forgetBuildNotes and for the same
// reason.
func forgetReplacedBelts() { replacedBelts = replacedBelts[:0] }

// backKV is the `create_entity` table the restore builds, package level so that
// putting a belt back allocates nothing. compile.go's `createArgs` is the hot
// path's own and has neither a direction nor a quality; this one runs at most
// once per refused fast replace.
var (
	backKV  [7]fkapi.KeyValue
	backPos [2]fkapi.Value
)

func backArgs(b beltSpec, force uint32, x, y int32) fkapi.Value {
	backPos[0] = fkapi.OfNumber(float64(x) + 0.5)
	backPos[1] = fkapi.OfNumber(float64(y) + 0.5)
	backKV[0] = fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(b.name)}
	backKV[1] = fkapi.KeyValue{Key: fkapi.OfString("position"),
		Val: fkapi.Value{Tag: fkapi.TagArray, Array: backPos[:]}}
	backKV[2] = fkapi.KeyValue{Key: fkapi.OfString("force"),
		Val: fkapi.OfNumber(float64(force))}
	backKV[3] = fkapi.KeyValue{Key: fkapi.OfString("direction"),
		Val: fkapi.OfNumber(float64(b.dir))}
	// `raise_built` so that a mod which watched the belt go sees it come back.
	// It re-enters this guest as an ordinary appearance, which is what queues
	// the neighbouring clusters the restored belt is an edge of -- and it is
	// safe for the same reason revertOne's own `mine_entity` is: this runs
	// AFTER the drain, so there is no drain to re-enter.
	backKV[4] = fkapi.KeyValue{Key: fkapi.OfString("raise_built"), Val: fkapi.OfBool(true)}
	n := 5
	if b.quality != "" {
		backKV[n] = fkapi.KeyValue{Key: fkapi.OfString("quality"),
			Val: fkapi.OfString(b.quality)}
		n++
	}
	if b.under {
		t := "input"
		if b.outEnd {
			t = "output"
		}
		backKV[n] = fkapi.KeyValue{Key: fkapi.OfString("type"), Val: fkapi.OfString(t)}
		n++
	}
	return fkapi.Value{Tag: fkapi.TagMap, Map: backKV[:n]}
}

// restoreReplacedBelt puts one belt back and charges the player for it.
//
// Called from revertOne and from nowhere else, AFTER the refused part has been
// mined out of the tile -- the two entities collide, so the belt cannot be
// created until the part is gone, which is legacyConvertOne's ordering met from
// the other side.
//
// `carryStack` is borrowed for both item calls rather than re-implemented. It is
// carry.go's scratch and it is free here by construction: revertOne runs after
// `endCarry`, so the transaction that owns those bytes has closed.
func restoreReplacedBelt(surf fkapi.LuaSurface, n buildNote, p fkapi.LuaPlayer) {
	b := n.belt
	// THE CHARGE COMES FIRST. A create that succeeded and a charge that then
	// failed would be a free belt, which is the one outcome this must not have;
	// a charge that succeeded and a create that then failed is refundable, and
	// is refunded below.
	paid, err := p.RemoveItem(carryStack(b.item, b.quality, 1))
	if err != nil {
		return
	}
	if paid == 0 {
		logAlertStart("player ")
		logU(n.player)
		logS(" had no ")
		logS(b.item)
		logS(" to pay for the replaced belt at ")
		logI(n.x)
		logS(",")
		logI(n.y)
		logS("; the gap stays")
		logEnd()
		return
	}
	o, err := surf.CreateEntity(backArgs(b, n.force, n.x, n.y))
	if err != nil || o == nil {
		_ = insertOne(p.Object, b.item, b.quality, 1)
		logAlertStart("could not put the replaced belt at ")
		logI(n.x)
		logS(",")
		logI(n.y)
		logS(" back (")
		logS(b.name)
		logS("); the item was returned to player ")
		logU(n.player)
		logEnd()
		return
	}
	// The line the interactive checklist greps for, at the same level and for
	// the same reason as revertOne's `handed the refused piece`: no headless run
	// can reach it, because a --create has no players at all.
	if verboseLog {
		logStart("put the replaced belt at ")
		logI(n.x)
		logS(",")
		logI(n.y)
		logS(" back for player ")
		logU(n.player)
		logS(" (")
		logS(b.name)
		logS("), taking one from their inventory")
		logEnd()
	}
}
