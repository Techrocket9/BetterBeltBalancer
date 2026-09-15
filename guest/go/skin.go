package main

// M5: the adaptive sprite, on the entities.
//
// `guest/go/skin` decides WHICH picture a part should draw from its
// neighbourhood; this file is the part that touches the world. It runs from the
// deferred flush, alongside the compiler, and its whole job is to make as few
// host calls as it possibly can:
//
//   - the mask for every part of a cluster is computed from the REGISTRY, which
//     is in guest memory. Working out that a 200-part balancer's pictures are
//     all still correct costs zero host calls;
//   - `pvar` remembers what was last put on each part, so only the parts whose
//     picture actually changed are touched. Placing one part against a big
//     balancer changes at most nine pictures, whatever its size;
//   - each of those costs one `find_entity` and one `graphics_variation =`.
//     There is no per-tick anything, no rendering object, and no second entity.
//
// AND THE VARIATION CARRIES THE PRIORITY FLAG AS WELL AS THE SHAPE, which makes
// this file the place a part's priority is PERSISTED rather than merely drawn:
// `skin.Variation` puts an ordinary part in cells 1..47 and a priority one in
// 48..94, the engine keeps that byte, and `recoverPriority` below reads it back
// off a world this guest did not write. A flag kept only in the guest heap would
// not survive a mod update, because the heap is declined on every rebuilt guest.
//
// WHY THIS IS NOT DONE IN THE BUILD EVENT, where the entity is already in hand:
// a part appearing changes its NEIGHBOURS' pictures too, and their handles are
// not. So this follows the guest's standing rule (CLAUDE.md): work that reads
// only the event's own payload happens in the event, work that reads the world
// happens in the flush. The picture is therefore correct one tick after the
// part is placed, which is 17 ms and which nobody can see.
//
// THE EDITOR'S VARIATION PICKER, which is the one place this can be driven from
// outside. `simple-entity-with-force` has a `pictures` set, so the map editor's
// entity dialog offers a variation picker -- that is the standard editor UI for
// any entity with one and 2.0.77 has no prototype field that suppresses it. A
// hand-picked cell survives until something changes the cluster's SHAPE, because
// restyle compares against `pvar` rather than reading the entity back. SINCE
// PRIORITIES THAT IS ONE STEP LOUDER THAN IT WAS: a cell above 47 picked by hand
// is a priority cell, so the next pass that does not know what is drawn there --
// a mod update's rebuild-from-world, which is the only one -- reads it back as
// the flag and the balancer really does gain a priority port. An editor pick
// below 47 on a priority part loses the flag the same way. Neither is worth a
// host call per part per flush to defend against.
//
// THE FORCE CHECK IN maskAt IS NOT COSMETIC. Two forces' parts touching are two
// balancers -- they never merge, they compile separately -- so they must not
// fuse into one shape either. Without it, an enemy force building against a
// player balancer would make both of them draw as one machine.

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/skin"
)

// The eight neighbour offsets, in the bit order `skin` numbers them:
// N, E, S, W, NE, SE, SW, NW. The first four are cluster.go's `dirs` and must
// stay in step with it.
var skinOff = [8][2]int32{
	{0, -1}, {1, 0}, {0, 1}, {-1, 0},
	{1, -1}, {1, 1}, {-1, 1}, {-1, -1},
}

var (
	skinTile []key
	skinWant []uint8
	skinNode []uint32
	// skinEnt is the entity each candidate resolved to, parallel to the three
	// above once restyle has compacted away the parts the world does not have.
	// Held rather than re-found because the priority read below wants every
	// handle at once.
	skinEnt  []fkapi.Object
	skinRead []fkapi.BulkOptUint8
	skinSort []key
)

// maskAt builds the neighbour mask for one tile. No host call: the registry
// already knows what is on every tile, which is the whole reason this is cheap.
func maskAt(k key, f uint32) uint8 {
	m := uint8(0)
	for i := 0; i < len(skinOff); i++ {
		nb, ok := index[key{k.s, k.x + skinOff[i][0], k.y + skinOff[i][1]}]
		if !ok || pforce[nb] != f {
			continue
		}
		m |= 1 << uint(i)
	}
	return m
}

// restyle brings one cluster's pictures up to date.
//
// Called for every root the flush is about to compile, and for every cluster a
// rebuild-from-world adopts. It is a no-op -- not one host call -- for a cluster
// whose shape did not change, which is what makes it safe to call from the same
// places the compiler is called from: a belt laid beside a balancer queues its
// cluster, and this looks at it and finds nothing to do.
//
// It uses `tileBuf` through collectCluster, so it must not run in the middle of
// anything else that is holding that buffer. Every caller runs it to completion
// first.
func restyle(root uint32) {
	tiles := collectCluster(root)
	if len(tiles) == 0 {
		return
	}
	f := pforce[root]

	skinTile, skinWant, skinNode = skinTile[:0], skinWant[:0], skinNode[:0]
	for i := range tiles {
		id, ok := index[tiles[i]]
		if !ok {
			continue
		}
		v := skin.Variation(maskAt(tiles[i], f), pprio[id] != 0)
		if pvar[id] == v {
			continue
		}
		skinTile = append(skinTile, tiles[i])
		skinWant = append(skinWant, v)
		skinNode = append(skinNode, id)
	}
	if len(skinTile) == 0 {
		return
	}

	surf, ok := surfaceByIndex(tiles[0].s)
	if !ok {
		return
	}

	// One query per part whose picture changed, never per part; the alternative
	// -- one area query for the whole cluster and a position read per entity --
	// costs O(parts) whatever changed. `findOnTile` rather than `find_entity`,
	// because a part at any quality but normal is invisible to a bare-name
	// `find_entity` (findpart.go) -- and this loop was the WORST home that trap
	// had: a part it can never find is left at pvar 0 and re-queried on every
	// flush that touches its cluster, forever, so an uncommon balancer drew the
	// lone-part picture on every tile AND paid a host call per part per flush
	// for it.
	//
	// A part the world does not have is COMPACTED AWAY rather than skipped
	// later, so that skinEnt stays parallel to the other three and the bulk read
	// below can hand the host one contiguous run of handles.
	skinEnt = skinEnt[:0]
	unknown, n := 0, 0
	for i := range skinTile {
		o, found, ferr := findOnTile(surf, PartName, skinTile[i].x, skinTile[i].y)
		if ferr != nil || !found {
			continue
		}
		skinTile[n], skinWant[n], skinNode[n] = skinTile[i], skinWant[i], skinNode[i]
		skinEnt = append(skinEnt, o)
		if pvar[skinNode[n]] == 0 {
			unknown++
		}
		n++
	}
	skinTile, skinWant, skinNode = skinTile[:n], skinWant[:n], skinNode[:n]
	if unknown > 0 {
		recoverPriority(f)
	}

	set := uint32(0)
	for i := range skinTile {
		// recoverPriority may have found the world already showing what we
		// want, which is what a whole balancer pasted from a blueprint looks
		// like: same shapes, same flags, nothing to write.
		if pvar[skinNode[i]] == skinWant[i] {
			continue
		}
		if err := (fkapi.LuaEntity{Object: skinEnt[i]}).SetGraphicsVariation(skinWant[i]); err != nil {
			continue
		}
		// Only after the engine took it. A part the world does not have (a
		// registry that drifted, a surface being torn down under us) keeps its 0
		// and is tried again next time rather than being remembered wrongly.
		pvar[skinNode[i]] = skinWant[i]
		set++
	}
	logSkin(root, tiles, set)
}

// recoverPriority reads the PRIORITY flag back off parts whose picture this
// guest has never written, and it is the whole reason the flag lives in the
// sprite variation at all.
//
// A part arrives with a variation nobody here chose in three ways and all three
// land on `pvar == 0`, which is what that zero means: a ghost revived from a
// BLUEPRINT (measured on 2.0.77 -- a blueprint over a simple-entity-with-force
// with a placeable_by carries `variation`, and the revived entity comes back
// wearing it), an entity CLONED from another surface, and a whole world arriving
// on a FRESH HEAP, which is every load of a rebuilt guest. Without this the
// first restyle would overwrite all three with the unflagged shape and the flag
// would be gone -- silently, because the picture it would write is a perfectly
// good picture.
//
// ONLY THE FLAG IS TAKEN, NEVER THE SHAPE. A pasted part's neighbourhood is not
// its source's, so the shape is recomputed from the registry as it always is;
// `skin.IsPriority` is the only thing read out of the recovered byte.
//
// ONE HOST CALL FOR THE WHOLE CLUSTER and no allocation per part: the bulk form
// of the getter takes the run of handles skinEnt already holds and writes two
// bytes per element into a buffer this file keeps. The ordinary getter returns a
// *uint8, which under -gc=leaking is a permanent allocation per part, on a path
// a blueprint paste drives once per part pasted.
//
// A PART WHOSE PICTURE THE GUEST DOES KNOW IS READ AND IGNORED. It is in the run
// because splitting the handles into two buffers would cost more than the
// engine's own read, and ignoring it is not an oversight: a variation that moved
// under a pvar we trust is either a settings paste, which has its own event and
// its own handler, or somebody picking a cell by hand in the map editor -- and
// an editor pick silently flipping a balancer's port priority on the next flush
// is worse than an editor pick that only looks wrong.
func recoverPriority(f uint32) {
	if cap(skinRead) < len(skinEnt) {
		skinRead = make([]fkapi.BulkOptUint8, len(skinEnt))
	}
	skinRead = skinRead[:len(skinEnt)]
	if _, err := fkapi.LuaEntityGraphicsVariationBulk(skinEnt, skinRead); err != nil {
		return
	}
	for i := range skinTile {
		// An element the host could not read comes back as the zero value with
		// Has false, never as the previous crossing's number, so a dead handle
		// leaves the flag alone.
		if pvar[skinNode[i]] != 0 || !skinRead[i].Has {
			continue
		}
		// IT ONLY EVER RAISES THE FLAG, and there is no `else` because both
		// states that reach here with pprio already set are states the REGISTRY
		// is right about: a fresh node's flag is 0, and the one other way to
		// arrive at pvar 0 is the settings paste, which either moved the flag
		// from the source or, having been refused, wrote the destination's own
		// picture and its `pvar` back before leaving (priority.go). Clearing it
		// on a variation that disagreed would let the engine's own copy of a
		// picture overrule what this guest was just told.
		if skin.IsPriority(skinRead[i].V) {
			pprio[skinNode[i]] = 1
			skinWant[i] = skin.Variation(maskAt(skinTile[i], f), true)
		}
		if skinRead[i].V == skinWant[i] {
			// The world already shows what this part should show, so there is
			// nothing to write and pvar can simply start where it is.
			pvar[skinNode[i]] = skinWant[i]
		}
	}
}

// logSkin is the assertion surface for the headless suite: the whole cluster's
// pictures, in (y, x) order, which is a statement about the SHAPE that a human
// and an assertion script can both read. The five shapes `skin_test.go` proves
// in Go are built in a real Factorio and compared against this line.
//
// Sorted rather than in flood-fill order because flood-fill order depends on
// which node is the root, and the root depends on the order the parts were
// built in -- which is a detail of the test mod, not of the mod.
func logSkin(root uint32, tiles []key, set uint32) {
	if !verboseLog {
		return
	}
	skinSort = skinSort[:0]
	for i := range tiles {
		k := tiles[i]
		at := len(skinSort)
		skinSort = append(skinSort, k)
		for at > 0 && (skinSort[at-1].y > k.y ||
			(skinSort[at-1].y == k.y && skinSort[at-1].x > k.x)) {
			skinSort[at-1], skinSort[at] = skinSort[at], skinSort[at-1]
			at--
		}
	}
	logStart("skin cluster=")
	logU(root)
	logS(" parts=")
	logU(uint32(len(skinSort)))
	logS(" set=")
	logU(set)
	logS(" vars=")
	for i := range skinSort {
		if i > 0 {
			logS(",")
		}
		if i == 32 {
			logS("...")
			break
		}
		logU(uint32(pvar[index[skinSort[i]]]))
	}
	logEnd()
}
