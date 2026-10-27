package main

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

var buildNotesInFlush int
var buildLookupEnts []fkapi.Object
var buildIdentityEnt [1]fkapi.Object
var buildIdentityRead [1]fkapi.BulkOptUint64

// The force-merge event moves the entity without changing its unit number.
func remapBuildForce(src, dst uint32) {
	for i := range buildNotes {
		if buildNotes[i].force == src {
			buildNotes[i].force = dst
		}
	}
}

func readBuildUnit(ent fkapi.LuaEntity) (uint64, bool) {
	buildIdentityEnt[0] = ent.Object
	count, err := fkapi.LuaEntityUnitNumberBulk(buildIdentityEnt[:], buildIdentityRead[:])
	return buildIdentityRead[0].V(), err == nil && count == 1 && buildIdentityRead[0].Has && buildIdentityRead[0].V() != 0
}

// Event-layer hook: call AFTER recording the part/neighbour build note, while
// the actual event entity is still in hand. Never recover this identity later
// from a coordinate: a different player's replacement could already stand there.
func bindPlayerBuildIdentity(s uint32, x, y int32, player uint32, ent fkapi.LuaEntity) {
	if player == 0 {
		return
	}
	needed, isPart := false, false
	for _, n := range buildNotes {
		if !n.done && n.unit == 0 && n.s == s && n.x == x && n.y == y && n.player == player {
			needed = true
			isPart = n.isPart
			break
		}
	}
	if !needed {
		return
	}
	u, identified := readBuildUnit(ent)
	f, ferr := ent.ForceIndex()
	x0, y0, x1, y1, bounded := x-2, y-2, x+2, y+2, true
	if !isPart {
		pos, err := ent.Position()
		if err != nil {
			bounded = false
		} else {
			x0, y0, x1, y1, bounded = neighbourFootprint(ent, pos.X, pos.Y)
		}
	}
	for i := range buildNotes {
		n := buildNotes[i]
		if n.done || n.unit != 0 || n.s != s || n.x != x || n.y != y || n.player != player {
			continue
		}
		if !identified || ferr != nil || !bounded {
			buildNotes[i].done = true
			reportBuildFailure(n, "Could not read this build's identity or affected area. Automatic refund is disabled for it; please inspect the placed entity.")
			continue
		}
		buildNotes[i].unit, buildNotes[i].force = u, f
		buildNotes[i].x0, buildNotes[i].y0, buildNotes[i].x1, buildNotes[i].y1 = x0, y0, x1, y1
		// Duplicate notifications may arrive after the original replacement
		// record expired. Keep its already-owned belt spec and retry history.
		for j := range buildNotes {
			if j == i || buildNotes[j].done || buildNotes[j].unit != u {
				continue
			}
			if !buildNotes[j].belt.known() {
				buildNotes[j].belt = n.belt
			}
			buildNotes[i].done = true
			break
		}
	}
}

func reportBuildFailure(n buildNote, message string) {
	logAlertStart("player build at ")
	logI(n.x)
	logS(",")
	logI(n.y)
	logS(": ")
	logS(message)
	logEnd()
	if o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(n.player))); err == nil && o != nil {
		_ = (fkapi.LuaPlayer{Object: *o}).Print(fkapi.OfString("Better Belt Balancer: "+message), nil)
	}
}

func beginBuildNotes() {
	buildNotesInFlush = len(buildNotes)
	for i := range buildNotes {
		buildNotes[i].seen, buildNotes[i].retry, buildNotes[i].exhausted = false, false, false
		buildNotes[i].refused = false
		if !buildNotes[i].done && buildNotes[i].unit == 0 {
			buildNotes[i].done = true
			reportBuildFailure(buildNotes[i], "Build identity was not recorded. No automatic refund will touch an unidentified entity.")
		}
	}
}

// The bounded event footprint avoids losing context for wide belts/loaders.
func buildTouchesRoot(n buildNote, root uint32) bool {
	if n.s != ppos[root].s || n.force != pforce[root] {
		return false
	}
	for y := n.y0; y <= n.y1; y++ {
		for x := n.x0; x <= n.x1; x++ {
			if id, ok := index[key{n.s, x, y}]; ok && find(id) == root {
				return true
			}
		}
	}
	return false
}

func noteBuildResult(root uint32, result compileResult, exhausted bool) {
	for i := 0; i < buildNotesInFlush; i++ {
		n := &buildNotes[i]
		if n.done || !buildTouchesRoot(*n, root) {
			continue
		}
		n.seen = true
		if result == compileRefused {
			n.refused = true
			if rebuildingFromWorld {
				n.retry = true
			} // informed refusal follows the rebuild
		}
		if result == compileRetry {
			n.retry, n.exhausted = true, n.exhausted || exhausted
		}
	}
}

func pendingBuildFor(root uint32) bool {
	for _, n := range buildNotes {
		if !n.done && n.refundWhy == "" && buildTouchesRoot(n, root) {
			return true
		}
	}
	return false
}

// Retain only unresolved work. Reentrant events can append during refunds;
// those notes are beyond this drain's boundary and belong to the next flush.
func forgetBuildNotes() {
	kept := 0
	for i, n := range buildNotes {
		if !n.done && (i >= buildNotesInFlush || n.refundWhy != "" || (n.seen && n.retry)) {
			buildNotes[kept] = n
			kept++
		}
	}
	for i := kept; i < len(buildNotes); i++ {
		buildNotes[i] = buildNote{}
	}
	buildNotes = buildNotes[:kept]
}

// Re-find the original entity, not merely an entity at the original position.
// The two booleans keep an unreadable identity separate from an absent one.
func resolveBuildNote(surf fkapi.LuaSurface, n buildNote) (fkapi.Object, bool, bool) {
	if n.unit == 0 {
		return fkapi.Object{}, false, false
	}
	if n.isPart {
		o, found, err := findOnTile(surf, PartName, n.x, n.y)
		if err != nil || !found {
			return fkapi.Object{}, false, err == nil
		}
		return matchBuildIdentity(o, n)
	}
	pos := fkapi.MapPosition{X: float64(n.x) + 0.5, Y: float64(n.y) + 0.5}
	force := fkapi.OfNumber(float64(n.force))
	ents, err := surf.FindEntitiesFilteredInto(buildLookupEnts,
		fkapi.EntitySearchFilters{Position: &pos, Type: &beltTypes, Force: &force})
	buildLookupEnts = ents
	if err != nil {
		return fkapi.Object{}, false, false
	}
	readable := true
	for _, o := range ents {
		ent, found, ok := matchBuildIdentity(o, n)
		if found {
			return ent, true, true
		}
		readable = readable && ok
	}
	return fkapi.Object{}, false, readable
}

func matchBuildIdentity(o fkapi.Object, n buildNote) (fkapi.Object, bool, bool) {
	e := fkapi.LuaEntity{Object: o}
	u, ok := readBuildUnit(e)
	if !ok {
		return fkapi.Object{}, false, false
	}
	if u != n.unit {
		return fkapi.Object{}, false, true
	}
	f, err := e.ForceIndex()
	if err != nil {
		return fkapi.Object{}, false, false
	}
	return o, f == n.force, true
}
