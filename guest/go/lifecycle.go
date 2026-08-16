package main

// M3: the paths that change what the compiler compiled from, other than a
// player building or mining one thing at a time.
//
// Every routine here obeys the same three rules as the rest of the guest:
//
//   - NO ENTITY REFERENCE OUTLIVES THE EVENT IT ARRIVED IN. Everything
//     persistent is a surface index, an integer tile and a force index. That is
//     the structural answer to the incumbent's dominant crash class -- a cached
//     LuaTransportLine invalidated by a removal path nobody anticipated -- and
//     it is why the routines below can be written at all: recovering from a
//     surprise is re-reading the world, not repairing a graph of handles.
//   - EVERY WALK IS OVER THE NODE ARRAY IN ID ORDER. Not over `index`, not over
//     `nets`; a map iteration order that differs between two clients of a
//     lockstep game is a desync.
//   - EVERY TEARDOWN RUNS BEFORE EVERY BUILD, which is why flush is split.

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
)

// ---------------------------------------------------------------------------
// Reading a part out of the world
// ---------------------------------------------------------------------------

// forceOf reads an entity's force INDEX.
//
// Two host calls: `force` is a LuaForce handle and `index` is a number on it.
// Paid only for our own parts -- never on the neighbour path, which is a
// position lookup against the registry and makes no host call at all.
func forceOf(o fkapi.Object) (uint32, bool) {
	f, err := fkapi.LuaEntity{Object: o}.Force()
	if err != nil {
		return 0, false
	}
	i, err := fkapi.LuaForce{Object: f}.Index()
	if err != nil {
		return 0, false
	}
	return i, true
}

// A filter with a name and no area: the whole surface. `findByName` cannot be
// reused for it -- its Area pointer is always set, and a zero bounding box is a
// query for one tile at the origin rather than for everything.
var (
	findByNameAll fkapi.EntitySearchFilters
	allNameFilter fkapi.Value
)

func init() {
	findByNameAll.Name = &allNameFilter
}

// ---------------------------------------------------------------------------
// Reconciling an area against the world
//
// The entry point for every path that can change several entities at once
// without reporting them one by one: an area clone, a brush clone, a surface
// that was cleared under us.
// ---------------------------------------------------------------------------

var (
	rcKeys  []key
	rcFound []key
	rcForce []uint32
	rcRoots []uint32
)

// reconcileArea makes the registry agree with the world inside a tile box, and
// rebuilds every cluster the box touches.
//
// The order is the whole content of the function:
//
//  1. bring down the network of every cluster near the box, giving its items
//     back. This has to happen FIRST and it has to happen properly, because
//     step 2 destroys without giving anything back.
//  2. destroy anything of ours still standing in the box. Nothing of ours
//     belongs there any more: compile() is the only thing that places our
//     prototypes and step 1 just removed everything it placed. What is left is
//     a COPY -- an area clone duplicates the visible interfaces along with the
//     belts around them -- and a copy's contents were minted by the clone, so
//     handing them back would create matter.
//  3. reconcile the registry: parts the clone created, parts the clone
//     destroyed (`clear_destination_entities` raises nothing).
//  4. rebuild.
func reconcileArea(si uint32, tx0, ty0, tx1, ty1 int32, what string) {
	s, ok := surfaceByIndex(si)
	if !ok {
		return
	}
	// The whole sequence is ONE carry transaction: step 1's teardowns drain the
	// legitimate networks, and step 4 builds the clusters that succeed them, so
	// the items belong to the rebuild rather than to the floor. A clone
	// reconcile is a recompile. (The COPIES the clone made are swept with
	// `give = false` in step 2 and never enter a pool: their contents were
	// minted by the clone and handing them back would create matter.)
	beginCarry()

	// 1. Two tiles of margin, the same margin the neighbour gate uses: a
	//    splitter's position sits on the boundary between the two tiles it
	//    covers, so a cluster just outside the box can still own an edge in it.
	rcRoots = rcRoots[:0]
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if !alive[i] {
			continue
		}
		k := ppos[i]
		if k.s != si || k.x < tx0-2 || k.x > tx1+2 || k.y < ty0-2 || k.y > ty1+2 {
			continue
		}
		markDead(find(i))
	}
	flushDead()

	// 2.
	killed := sweep(s, tx0, ty0, tx1, ty1, false)

	// 3. What the world says is there.
	rcFound = rcFound[:0]
	rcForce = rcForce[:0]
	setSearchBox(tx0, ty0, tx1, ty1)
	nameFilter = fkapi.OfString(PartName)
	if ents, err := s.FindEntitiesFiltered(findByName); err == nil {
		for i := range ents {
			p, err := fkapi.LuaEntity{Object: ents[i]}.Position()
			if err != nil {
				continue
			}
			f, ok := forceOf(ents[i])
			if !ok {
				continue
			}
			rcFound = append(rcFound, key{s: si, x: floorTile(p.X), y: floorTile(p.Y)})
			rcForce = append(rcForce, f)
		}
	}
	// What the registry says is there. Gone first, so that a tile which was
	// cleared and rebuilt in the same clone is removed before it is re-added.
	rcKeys = partsInBox(rcKeys, si, tx0, ty0, tx1, ty1)
	for i := range rcKeys {
		still := false
		for j := range rcFound {
			if rcFound[j] == rcKeys[i] {
				still = true
				break
			}
		}
		if !still {
			RemovePart(rcKeys[i])
		}
	}
	added := 0
	for i := range rcFound {
		if AddPart(rcFound[i], rcForce[i]) {
			added++
		}
	}

	// 4. Whatever the registry now holds in the box gets a network.
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if !alive[i] {
			continue
		}
		k := ppos[i]
		if k.s != si || k.x < tx0-2 || k.x > tx1+2 || k.y < ty0-2 || k.y > ty1+2 {
			continue
		}
		markLive(find(i))
	}
	flush()
	endCarry()
	if verboseLog {
		logStart(what)
		logS(": reconciled surface ")
		logU(si)
		logS(" box ")
		logI(tx0)
		logS(",")
		logI(ty0)
		logS("..")
		logI(tx1)
		logS(",")
		logI(ty1)
		logS(" -- ")
		logU(uint32(added))
		logS(" parts adopted, ")
		logU(killed)
		logS(" copied network entities destroyed")
		logEnd()
	}
}

// ---------------------------------------------------------------------------
// Surfaces coming and going
// ---------------------------------------------------------------------------

var dsKeys []key

// dropSurface forgets every part on a surface and frees what its clusters held.
//
// Called from on_pre_surface_deleted and on_pre_surface_cleared, where the
// surface and everything on it is still valid -- which is what lets the
// teardown drain the hidden halves properly and hand the slots back. Doing it
// from the POST event would find the parts gone and leak a slot per cluster,
// which is a network slot that is never reused and a hidden network that is
// never destroyed.
func dropSurface(si uint32, why string) {
	rcRoots = rootsOnSurface(rcRoots, si)
	for i := range rcRoots {
		markDead(rcRoots[i])
	}
	flushDead()

	dsKeys = partsOnSurface(dsKeys, si)
	for i := range dsKeys {
		RemovePart(dsKeys[i])
	}
	// Removal queues the survivors for a rebuild, and there are none: every
	// part of every cluster on this surface has just gone.
	flushDead()
	liveRoots = liveRoots[:0]
	if verboseLog && (len(dsKeys) > 0 || len(rcRoots) > 0) {
		logStart(why)
		logS(": surface ")
		logU(si)
		logS(" gave up ")
		logU(uint32(len(dsKeys)))
		logS(" parts in ")
		logU(uint32(len(rcRoots)))
		logS(" clusters")
		logEnd()
	}
}

// hiddenSurfaceGoing is the pathological case: something outside this mod is
// deleting or clearing the surface every network lives on.
//
// It is not a crash and it is not silent. Every network comes down first, while
// the hidden entities still exist, so the items in them are returned to the
// VISIBLE surfaces -- which survive -- rather than deleted with the surface.
func hiddenSurfaceGoing(why string) {
	logAlertStart("the hidden surface " + hiddenSurfName + " is being ")
	logS(why)
	logS(" by something outside this mod; every compiled network is coming down " +
		"with it and will be rebuilt")
	logEnd()
	rcRoots = liveRootList(rcRoots)
	for i := range rcRoots {
		markDead(rcRoots[i])
	}
	flushDead()
}

// hiddenSurfaceGone runs after the deletion. The surface is recreated lazily by
// the first compile, so all this has to do is forget where it was and ask for
// every cluster back.
func hiddenSurfaceGone() {
	// One carry transaction over the forget-then-rebuild pair: every cluster
	// here keeps its parts and gets a network back, so the items its visible
	// interfaces were holding belong in the replacement rather than on the
	// ground beside it.
	beginCarry()
	hiddenIdx = 0
	rcRoots = liveRootList(rcRoots)
	// Anything hiddenSurfaceGoing did not reach -- a clear that raised no
	// pre-event -- still has a netInfo pointing at entities that no longer
	// exist. Forget it, taking the visible interfaces down with it: leaving
	// those standing would make every rebuild collide with its own ghost and
	// abort.
	for i := range rcRoots {
		forgetNet(rcRoots[i])
	}
	// AFTER the loop, not before: forgetNet hands each slot back, and a free
	// list rebuilt on top of those returns would hand the same slot out twice.
	nextSlot = 1
	freeSlots = freeSlots[:0]
	for i := range rcRoots {
		markLive(rcRoots[i])
	}
	flush()
	endCarry()
	if verboseLog {
		logStart("hidden surface recreated; ")
		logU(uint32(len(rcRoots)))
		logS(" clusters rebuilt")
		logEnd()
	}
}

// ---------------------------------------------------------------------------
// Two forces becoming one
//
// `game.merge_forces(source, destination)` moves every entity of the source
// force onto the destination and then DESTROYS the source force. It raises no
// per-entity event for any of it, so nothing else in this file can see it.
//
// What that leaves behind without a handler, and each of these was reachable:
//
//   - every node of the source force carries a `pforce` that names a force
//     which no longer exists. `classifyEdges` sets that index as the engine's
//     force filter and `createArgs` puts it in a `create_entity` table -- one
//     of those quietly returns nothing and the other raises;
//   - two clusters that touch and used to be two balancers BECAUSE their forces
//     differed are now one balancer, and the registry still says two. Two
//     overlapping networks on tiles that belong to one cluster is the shape M3
//     found twice (CLAUDE.md, "Two bugs M3 found");
//   - a belt of the source force beside a cluster of the destination force was
//     never an edge and now is, anywhere on the map.
//
// The last one is why the sweep below is over every cluster of the surviving
// force rather than only the ones near a merge: the edge list of a balancer a
// thousand tiles away can have changed, and the fingerprint is what decides
// whether that costs anything. A merge is an administrator's keypress, so the
// conservative answer is the affordable one.
// ---------------------------------------------------------------------------

// onForcesMerged re-derives the clusters after two forces became one.
//
// The order is the whole content of the function, and it is the same order
// reconcileArea uses for the same reason: a network has to come down while
// `nets` is still keyed by the root that owns it. A cluster absorbed by a merge
// stops being a root, `liveRootList` stops returning it, and its hidden network
// and its visible interfaces would stand for the rest of the session with
// nothing left that knows where they are.
func onForcesMerged(src, dst uint32) {
	if src == 0 || src == dst {
		return
	}
	// One carry transaction, for the same reason reconcileArea is one: every
	// cluster torn down at step 2 is rebuilt at step 5, under a root the remap
	// may have changed but over the same tiles. A merge of two forces is a
	// recompile of both, not a removal of either.
	beginCarry()

	// 1. Which networks must come down. Every cluster of the source force --
	//    its stored force index is about to be wrong -- and every cluster of the
	//    destination force that TOUCHES one, because one of the two is about to
	//    stop being a root. Both are marked; the flush deduplicates.
	moved := uint32(0)
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if !alive[i] || pforce[i] != src {
			continue
		}
		moved++
		markDead(find(i))
		k := ppos[i]
		for d := 0; d < len(dirs); d++ {
			if nb, ok := index[key{k.s, k.x + dirs[d][0], k.y + dirs[d][1]}]; ok &&
				pforce[nb] == dst {
				markDead(find(nb))
			}
		}
	}
	if moved == 0 {
		// Nothing was queued and nothing was drained, but the transaction still
		// has to close: the depth counter is what decides when a pool settles.
		endCarry()
		return
	}
	// 2. Down they come, items handed back, slots released.
	flushDead()

	// 3. The remap itself -- of the registry, and of the pools step 2 drained,
	//    which carry the force their networks were built with and would
	//    otherwise be unrecognisable to the cluster that inherits them.
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if alive[i] && pforce[i] == src {
			pforce[i] = dst
		}
	}
	remapCarryForce(src, dst)

	// 4. Re-derive every component of the surviving force. The remap can only
	//    ADD adjacencies -- two parts that were separated by a force boundary
	//    are now neighbours, and no pair that was joined can have come apart --
	//    so a flood fill per unvisited node is a complete re-derivation, and it
	//    picks the smallest id in each component as the root exactly as a split
	//    does. Node id order, so every client agrees.
	gen++
	clusters := uint32(0)
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if !alive[i] || pforce[i] != dst || mark[i] == gen {
			continue
		}
		markLive(rebuildComponent(i))
		clusters++
	}
	// 5. And rebuild. Most of these are fingerprint skips: a balancer nowhere
	//    near the merge has the same edge list it had a moment ago.
	flush()
	endCarry()

	logStart("forces merged: ")
	logU(src)
	logS(" -> ")
	logU(dst)
	logS(", ")
	logPlural(moved)
	logS(" remapped, ")
	logU(clusters)
	logS(" clusters of the surviving force re-derived")
	logEnd()
}

// ---------------------------------------------------------------------------
// Coming back on a heap this build did not write
//
// A mod upgrade rebuilds the guest, and FkLua does not hand a rebuilt guest the
// old heap unless it exports `fk_migrate_adopt` -- which this mod must never do
// (FKLUA-GAPS.md item 13: the bytes handed over are a DIFFERENT build's linear
// memory, rodata and all, so every string constant and every static address in
// them belongs to another program). It DOES export `fk_migrate`, which upstream
// split off as a notification on a FRESH heap; see main.go. So the registry
// comes back empty while the world is still full of parts and still full of
// compiled networks.
//
// Nothing breaks in the meantime, and that is worth saying plainly: a compiled
// network is engine state. Splitters keep splitting and linked belts keep
// teleporting whether or not any script remembers them. What must not happen is
// the guest deciding a cluster has no network and building it a SECOND one.
//
// `fk_migrate` names the moment for the case it covers. The bool below covers
// everything else that can leave this guest holding an empty registry over a
// full world -- a mod added to a save that already has parts, a `--persist`
// mode changed under it -- so the first event of any kind still rebuilds before
// it decides anything. It is one bool test on the hot path and it stays.
// ---------------------------------------------------------------------------

// registryReady is false in a freshly initialised heap and true in a heap that
// has been through here. It is not set by fk_on_init: a mod added to an
// existing save gets on_init too, and that save may already contain parts.
var registryReady bool

func ensureRegistry() {
	if registryReady {
		return
	}
	// Set BEFORE the scan. rebuildFromWorld makes host calls, and a host call
	// can raise an event synchronously; a re-entrant scan would register every
	// part twice over.
	registryReady = true
	rebuildFromWorld()
}

var (
	// The surfaces `game.surfaces` reported, and their indices, sorted. Package
	// level and reused: this runs once per session, but under -gc=leaking once
	// per session is once per session forever.
	rfwSurfIdx []uint32
	rfwSurfObj []fkapi.Object

	rfwRoots []uint32
	// rfwClaim is slot -> root, point-queried only; rfwMaxClaim is tracked as
	// it fills rather than read back out of it, because ranging a map is a
	// desync waiting for a client whose hash order differs.
	rfwClaim    map[uint32]uint32
	rfwMaxClaim uint32
	rfwSlots    []uint32
	adoptPos    []key
	adoptDir    []uint32
	adoptSeen   []bool
)

// collectSurfaces fills rfwSurfIdx/rfwSurfObj with every surface in the game,
// SORTED BY INDEX, and sets hiddenIdx if the hidden surface is among them.
//
// ASK WHAT SURFACES EXIST. `game.surfaces` binds now (FKLUA-GAPS.md item 15): a
// dyn-keyed dictionary return comes back as an ordered pair slice, and `pairs()`
// over this one yields the NAME as the key, so the hidden surface falls out of
// the same walk rather than costing its own `get_surface` by name.
//
// What this replaces is a probe: `get_surface(1)`, `get_surface(2)`, ...
// stopping after 64 consecutive misses, which was ~65 host calls on a save with
// one surface and a guess about how sparse an index can get. It is now one call
// plus one `index` read per surface -- two, on that save.
//
// TWO CALLERS, ONE WALK, and it is a shared function rather than two copies for
// the reason the sort exists at all: both of them go on to register parts in the
// order this list gives, which decides node ids, which decide cluster roots and
// hidden-surface slots. Two walks that could disagree about an order would be a
// desync waiting for the day one of them changed. See legacy.go, which is the
// second caller.
//
// SORTED BY INDEX, by insertion, which is what the probe loop used to give for
// free. fk_abi.lua declines to promise an iteration order for a dictionary
// return -- reasonably, since it walks `pairs()`. Two or three surfaces is a
// real save and a dozen is a big one, so an insertion sort is the whole
// algorithm.
func collectSurfaces() {
	hiddenIdx = 0
	rfwSurfIdx, rfwSurfObj = rfwSurfIdx[:0], rfwSurfObj[:0]
	pairs, err := fkapi.Game.Surfaces()
	if err != nil {
		return
	}
	for i := range pairs {
		si, err := (fkapi.LuaSurface{Object: pairs[i].Val}).Index()
		if err != nil {
			continue
		}
		if pairs[i].Key.Tag == fkapi.TagString && pairs[i].Key.Str == hiddenSurfName {
			hiddenIdx = si
		}
		at := len(rfwSurfIdx)
		rfwSurfIdx = append(rfwSurfIdx, si)
		rfwSurfObj = append(rfwSurfObj, pairs[i].Val)
		for at > 0 && rfwSurfIdx[at-1] > si {
			rfwSurfIdx[at-1], rfwSurfIdx[at] = rfwSurfIdx[at], rfwSurfIdx[at-1]
			rfwSurfObj[at-1], rfwSurfObj[at] = rfwSurfObj[at], rfwSurfObj[at-1]
			at--
		}
	}
}

// rebuildFromWorld re-derives the whole registry, and ADOPTS the networks it
// finds rather than rebuilding them.
//
// Adoption is what makes a mod upgrade cheap and lossless. A rebuild of every
// balancer on a large map is hundreds of milliseconds of teardown and thousands
// of create_entity calls, and every item standing in every hidden network would
// be spilled on the floor. Adoption costs about thirty host calls per cluster
// and touches nothing: the network is already correct, because no tick has
// passed since it was built.
//
// It is only taken when the evidence is complete -- the visible interfaces
// found in the cluster's box match the edge list re-derived from the world,
// position for position and direction for direction. Anything less falls back
// to a rebuild, which is always correct.
func rebuildFromWorld() {
	statCompiles, statSkipped, statBuilds, statTeardowns, statCreates = 0, 0, 0, 0, 0
	// A refusal issued while this flag is up logs, requeues and says NOTHING
	// to anybody -- see refuseOverLimit. The rebuild judges the world with the
	// worst information a refusal will ever have (when it runs from the first
	// event of a session, that event's own build note cannot exist yet), and a
	// message sent now claims a final state one more tick may falsify: the
	// 2026-08-05 field reports were first a hand-back suppressed entirely, and
	// then a chat line saying "the piece was left in place" one tick before
	// the informed retry handed the piece back. The first real flush re-judges
	// with the notes recorded and delivers the one correct message.
	rebuildingFromWorld = true

	collectSurfaces()

	surfaces, found := uint32(0), uint32(0)
	for i := range rfwSurfIdx {
		si := rfwSurfIdx[i]
		surfaces++
		if si == hiddenIdx {
			continue
		}
		found += registerPartsOn(fkapi.LuaSurface{Object: rfwSurfObj[i]}, si)
	}

	// Adopt or rebuild, cluster by cluster, in node id order.
	if rfwClaim == nil {
		rfwClaim = map[uint32]uint32{}
	}
	for k := range rfwClaim {
		delete(rfwClaim, k)
	}
	rfwRoots = liveRootList(rfwRoots)
	adopted, rebuilt := uint32(0), uint32(0)
	rfwMaxClaim = 0
	for i := range rfwRoots {
		r := rfwRoots[i]
		// The pictures first, and BEFORE inspectNetwork, which holds the tile
		// buffer restyle() would otherwise reuse under it. A fresh heap knows
		// nothing about what is drawn on the parts standing in the world (pvar
		// is empty), so this is the one path that pays two host calls per part
		// -- once per mod upgrade, and it is what makes a save built by an older
		// build of this mod come back drawing the shapes it should.
		restyle(r)
		slot, exact := inspectNetwork(r)
		if slot > 0 {
			if _, taken := rfwClaim[slot]; taken {
				// Two clusters cannot own one slot. Neither is trustworthy.
				exact = false
			} else {
				rfwClaim[slot] = r
				if slot > rfwMaxClaim {
					rfwMaxClaim = slot
				}
			}
		}
		if exact {
			adopted++
			continue
		}
		// Not adopted. inspectNetwork has already recorded where the standing
		// interfaces are (and the slot, if it could be read); inverting the
		// fingerprint is what makes the next compile TEAR THAT DOWN -- items
		// and all -- before it builds. Leaving the interfaces standing would
		// not collide: a linked belt does not collide with itself, so the
		// rebuild would quietly put a second one on the same tile side and one
		// of the two would move nothing (spike S1).
		if ni, had := nets[r]; had {
			ni.fp = ^ni.fp
			nets[r] = ni
		}
		rebuilt++
		markLive(r)
	}

	sweepOrphanSlots()
	flush()
	rebuildingFromWorld = false
	// The silent refusals above, requeued now that the flush's drain is over
	// (an append during the drain would have been truncated with the queue).
	// The next ordinary flush re-judges them with whatever build notes the
	// rest of this dispatch records, and speaks then.
	if len(rebuildRefused) > 0 {
		for i := range rebuildRefused {
			markLive(rebuildRefused[i])
		}
		rebuildRefused = rebuildRefused[:0]
		requestFlush()
	}
	registryReady = true

	logStart("rebuilt from world: ")
	logU(surfaces)
	logS(" surfaces, ")
	logU(found)
	logS(" parts, ")
	logU(uint32(len(rfwRoots)))
	logS(" clusters (")
	logU(adopted)
	logS(" networks adopted, ")
	logU(rebuilt)
	logS(" rebuilt)")
	logEnd()
	logStats("after-rebuild")
}

// registerPartsOn puts every part on one surface into the registry.
func registerPartsOn(s fkapi.LuaSurface, si uint32) uint32 {
	allNameFilter = fkapi.OfString(PartName)
	ents, err := s.FindEntitiesFiltered(findByNameAll)
	if err != nil {
		return 0
	}
	n := uint32(0)
	for i := range ents {
		p, err := fkapi.LuaEntity{Object: ents[i]}.Position()
		if err != nil {
			continue
		}
		f, ok := forceOf(ents[i])
		if !ok {
			continue
		}
		if AddPart(key{s: si, x: floorTile(p.X), y: floorTile(p.Y)}, f) {
			n++
		}
	}
	// Nothing is compiled yet: the caller decides per cluster whether to adopt
	// the network already standing or to build a new one.
	deadRoots = deadRoots[:0]
	liveRoots = liveRoots[:0]
	return n
}

// inspectNetwork looks at what is standing around a cluster: which hidden slot
// it uses, and whether it matches what the world now implies.
//
// IT RECORDS WHAT IT FINDS EITHER WAY. A cluster with interfaces standing gets
// a netInfo with the real bounding box and the real slot even when the match
// fails, because the caller's fallback is to rebuild -- and a rebuild that did
// not know where the old interfaces were would leave them there. `slot` is 0
// when the interfaces exist but their hidden partners could not be followed,
// which is a netInfo teardown handles by sweeping only the visible half.
func inspectNetwork(root uint32) (slot uint32, exact bool) {
	tiles := collectCluster(root)
	if len(tiles) == 0 {
		return 0, false
	}
	si := tiles[0].s
	s, ok := surfaceByIndex(si)
	if !ok {
		return 0, false
	}
	x0, y0, x1, y1 := tiles[0].x, tiles[0].y, tiles[0].x, tiles[0].y
	for i := range tiles {
		if tiles[i].x < x0 {
			x0 = tiles[i].x
		}
		if tiles[i].x > x1 {
			x1 = tiles[i].x
		}
		if tiles[i].y < y0 {
			y0 = tiles[i].y
		}
		if tiles[i].y > y1 {
			y1 = tiles[i].y
		}
	}

	setSearchBox(x0, y0, x1, y1)
	nameFilter = fkapi.OfString(nameLinkedBelt)
	ents, err := s.FindEntitiesFiltered(findByName)
	if err != nil || len(ents) == 0 {
		return 0, false // nothing standing: a clean build, with nothing to remove
	}

	adoptPos = adoptPos[:0]
	adoptDir = adoptDir[:0]
	ok = true
	for i := range ents {
		p, err := fkapi.LuaEntity{Object: ents[i]}.Position()
		if err != nil {
			ok = false
			break
		}
		d, err := fkapi.LuaEntity{Object: ents[i]}.Direction()
		if err != nil {
			ok = false
			break
		}
		adoptPos = append(adoptPos, key{s: si, x: floorTile(p.X), y: floorTile(p.Y)})
		adoptDir = append(adoptDir, d)
	}

	// The slot, from where the first interface's partner sits. The hidden
	// surface is carved on a fixed grid, so a position IS a slot number.
	// `linked_belt_neighbour` is an OPTIONAL attribute, so an absent one comes
	// back as `(nil, nil)` rather than as an error -- upstream's F2, 2026-08-03.
	// An unconnected linked belt is the ordinary case here (it is what a
	// half-built network looks like), so the nil test is the reading, not a
	// guard: before F2 it arrived as ERR_NO_MEMBER and was indistinguishable
	// from a Factorio that does not have the member at all.
	if nb, err := (fkapi.LuaEntity{Object: ents[0]}).LinkedBeltNeighbour(); err == nil && nb != nil && nb.Valid() {
		if np, err := (fkapi.LuaEntity{Object: *nb}).Position(); err == nil &&
			np.X >= 0 && np.Y >= 0 {
			col, row := uint32(np.X)/slotW, uint32(np.Y)/slotH
			if col < slotCols {
				slot = row*slotCols + col + 1
			}
		}
	}

	// Does the standing network match the world? The edge list is
	// (tile, direction) and so is an interface, one to one.
	force := pforce[root]
	edges := classifyEdges(s, tiles, force)

	// Record it before deciding, so that the caller's fallback -- invert the
	// fingerprint and let compile() tear this down -- has a box to sweep.
	nets[root] = netInfo{fp: fingerprint(edges), slot: slot, surf: si, force: force,
		x0: x0, y0: y0, x1: x1, y1: y1, ents: uint32(len(edges))}

	if !ok || slot == 0 || len(edges) != len(adoptPos) {
		return slot, false
	}
	for len(adoptSeen) < len(edges) {
		adoptSeen = append(adoptSeen, false)
	}
	for i := range edges {
		adoptSeen[i] = false
	}
	for i := range adoptPos {
		hit := false
		for j := range edges {
			if adoptSeen[j] {
				continue
			}
			if edges[j].TileX == adoptPos[i].x && edges[j].TileY == adoptPos[i].y &&
				edges[j].Dir == adoptDir[i] {
				adoptSeen[j], hit = true, true
				break
			}
		}
		if !hit {
			return slot, false
		}
	}
	return slot, true
}

// sweepOrphanSlots destroys every hidden network no cluster claims, and rebuilds
// the slot allocator around the ones that are claimed.
//
// The occupied slots are found from the LANE SPLITTERS on the hidden surface:
// there is exactly one per used input port, so every network that exists has at
// least one and no network has many. One query plus one position read each,
// against one query per slot for a scan that would have to guess how far to go.
//
// This is also where a prototype rename lands. Our four prototypes are ours, so
// they can only be renamed by a version of this mod -- which is a rebuilt guest,
// which is exactly the path that arrives here.
func sweepOrphanSlots() {
	hid, ok := hiddenSurface()
	if !ok {
		nextSlot = 1
		freeSlots = freeSlots[:0]
		return
	}
	allNameFilter = fkapi.OfString(nameLaneSplit)
	ents, err := hid.FindEntitiesFiltered(findByNameAll)
	if err != nil {
		return
	}
	rfwSlots = rfwSlots[:0]
	maxSlot := uint32(0)
	for i := range ents {
		p, err := fkapi.LuaEntity{Object: ents[i]}.Position()
		if err != nil || p.X < 0 || p.Y < 0 {
			continue
		}
		col, row := uint32(p.X)/slotW, uint32(p.Y)/slotH
		if col >= slotCols {
			continue
		}
		s := row*slotCols + col + 1
		if s > maxSlot {
			maxSlot = s
		}
		seen := false
		for j := range rfwSlots {
			if rfwSlots[j] == s {
				seen = true
				break
			}
		}
		if !seen {
			rfwSlots = append(rfwSlots, s)
		}
	}
	if rfwMaxClaim > maxSlot {
		maxSlot = rfwMaxClaim
	}

	orphans := uint32(0)
	freeSlots = freeSlots[:0]
	// Descending, so the LIFO free list hands the low slots out first -- which
	// is what a fresh allocator does, and keeps a save's slot numbering stable.
	for s := maxSlot; s >= 1; s-- {
		if _, claimed := rfwClaim[s]; claimed {
			continue
		}
		freeSlots = append(freeSlots, s)
		occupied := false
		for j := range rfwSlots {
			if rfwSlots[j] == s {
				occupied = true
				break
			}
		}
		if !occupied {
			continue
		}
		ox, oy := slotOrigin(s)
		orphans += sweep(hid, ox, oy, ox+slotW-1, oy+slotH-1, false)
	}
	nextSlot = maxSlot + 1
	if verboseLog && orphans > 0 {
		logStart("swept ")
		logU(orphans)
		logS(" orphaned hidden entities from slots no cluster claims")
		logEnd()
	}
}

// ---------------------------------------------------------------------------
// The audit
//
// What the stress test asks at the end, and what a player can ask for at any
// time by having a script place a `bbb-audit` marker: does every cluster's
// stored fingerprint still equal the one a from-scratch classification of the
// world produces?
//
// It REPORTS before it repairs, so a drift that only the audit would have fixed
// still shows up in the log as a drift.
//
// TWO THINGS REACH IT: the `bbb-audit` marker, which is script-placeable and is
// what every test mod uses, and the `/bbb-audit` console command, which is how a
// PLAYER with a misbehaving save asks the same question. See commands.go -- the
// marker cannot be placed by hand, so until the command existed this whole
// diagnostic was reachable only from another mod's Lua.
// ---------------------------------------------------------------------------

var auditRoots []uint32

// auditAll returns the number of clusters it classified, which is what the
// remote form of the command hands back to its caller. Every other caller
// ignores it and reads the log line instead.
func auditAll() uint32 {
	ensureRegistry()
	// AND IT IS A DOOR ONTO THE MIGRATION TOO. `/bbb-audit` is the one thing a
	// player can type that means "look at the world again", so a save whose
	// incumbent was removed without the load hook seeing it converts here. It is
	// one integer compare on every other audit ever run. See legacy.go.
	legacyRearm(legTrigAudit)
	auditRoots = liveRootList(auditRoots)
	nets0, drift, unbuilt := uint32(0), uint32(0), uint32(0)
	for i := range auditRoots {
		r := auditRoots[i]
		tiles := collectCluster(r)
		if len(tiles) == 0 {
			continue
		}
		s, ok := surfaceByIndex(tiles[0].s)
		if !ok {
			drift++
			continue
		}
		edges := classifyEdges(s, tiles, pforce[r])
		n, m := 0, 0
		for j := range edges {
			if edges[j].Out {
				m++
			} else {
				n++
			}
		}
		ni, had := nets[r]
		// AND WHATEVER A REFUSED MERGE LEFT STANDING UNDER A KEY THAT IS NO
		// LONGER A ROOT. This is the one state in which `nets` holds a network
		// that `liveRootList` can never reach, so without asking limit.go the
		// audit would under-report `nets=` by one per spared predecessor and --
		// where the merged cluster's own root is a node that never had a network
		// -- would call a cluster whose two balancers are running perfectly well
		// `unbuilt`. See limit.go, "The merge".
		if k := strandedNets(r); k != 0 {
			nets0 += k
			if had {
				nets0++
			}
			// Its edge list is past what this mod builds and the guest knows it:
			// what stands is not what the cluster now asks for, which is exactly
			// what drift means.
			drift++
			continue
		}
		if had {
			nets0++
			if ni.fp != fingerprint(edges) {
				drift++
			}
			continue
		}
		// No network. Legitimate only when there is nothing to balance.
		if n > 0 && m > 0 {
			unbuilt++
		}
	}
	logStart("audit clusters=")
	logU(uint32(len(auditRoots)))
	logS(" parts=")
	logU(nParts)
	logS(" nets=")
	logU(nets0)
	logS(" drift=")
	logU(drift)
	logS(" unbuilt=")
	logU(unbuilt)
	logEnd()
	logStats("audit")
	// Repair, after reporting.
	for i := range auditRoots {
		markLive(auditRoots[i])
	}
	flush()
	// AFTER the flush, which is the only heap reading that means anything on a
	// `--create`: the audit reports before it repairs, so `logStats("audit")`
	// above is taken before the compiles it is about to provoke. A bench save's
	// 200 networks are all built by this one flush (`--create` never reaches a
	// tick), so this line is the heap the save is taken on. See gc.go.
	logHeap("post-audit")
	return uint32(len(auditRoots))
}

// revalidateSurface re-classifies every cluster on one surface and rebuilds the
// ones whose edge list moved.
//
// The fingerprint makes this cheap in the case that matters: a cluster nothing
// happened to costs one classification and no entity work at all.
func revalidateSurface(si uint32, why string) {
	rcRoots = rootsOnSurface(rcRoots, si)
	before := statBuilds
	for i := range rcRoots {
		markLive(rcRoots[i])
	}
	flush()
	if verboseLog {
		logStart(why)
		logS(": revalidated ")
		logU(uint32(len(rcRoots)))
		logS(" clusters on surface ")
		logU(si)
		logS(", ")
		logU(statBuilds - before)
		logS(" rebuilt")
		logEnd()
	}
}

// revalidateAll is revalidateSurface with no surface to scope by.
func revalidateAll(why string) {
	rcRoots = liveRootList(rcRoots)
	before := statBuilds
	for i := range rcRoots {
		markLive(rcRoots[i])
	}
	flush()
	if verboseLog {
		logStart(why)
		logS(": revalidated all ")
		logU(uint32(len(rcRoots)))
		logS(" clusters, ")
		logU(statBuilds - before)
		logS(" rebuilt")
		logEnd()
	}
}
