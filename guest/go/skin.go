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
// WHY THIS IS NOT DONE IN THE BUILD EVENT, where the entity is already in hand:
// a part appearing changes its NEIGHBOURS' pictures too, and their handles are
// not. So this follows the guest's standing rule (CLAUDE.md): work that reads
// only the event's own payload happens in the event, work that reads the world
// happens in the flush. The picture is therefore correct one tick after the
// part is placed, which is 17 ms and which nobody can see.
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
	skinSort []key
	skinPos  fkapi.MapPosition
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
		v := skin.Variation(maskAt(tiles[i], f))
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
	set := uint32(0)
	for i := range skinTile {
		// A part sits at its tile's centre. `find_entity` by name is one call
		// and returns the one thing that can be on that tile under that name;
		// the alternative -- one area query for the whole cluster and a position
		// read per entity -- costs O(parts) whatever changed.
		skinPos.X = float64(skinTile[i].x) + 0.5
		skinPos.Y = float64(skinTile[i].y) + 0.5
		o, err := surf.FindEntity(fkapi.OfString(PartName), skinPos)
		if err != nil || o == nil {
			continue
		}
		if err := (fkapi.LuaEntity{Object: *o}).SetGraphicsVariation(skinWant[i]); err != nil {
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
