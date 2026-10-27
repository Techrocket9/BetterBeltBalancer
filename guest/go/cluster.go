package main

// The cluster registry: which balancer parts exist, and which of them form one
// balancer.
//
// A cluster is a maximally connected set of `bbb-balancer-part` entities under
// 4-neighbour (orthogonal) adjacency, per surface. M2 compiles one of these
// into a hidden splitter network; M1 only has to know what they are, and to
// know it identically on every client.
//
// DETERMINISM IS THE WHOLE DESIGN CONSTRAINT. Factorio is a lockstep
// simulation, so nothing observable may depend on Go map iteration order --
// which TinyGo randomises the same way the big runtime does. The rules that
// follow from that, and which every function here obeys:
//
//   - Positions are INTEGER tile coordinates plus a surface index. The
//     incumbent keys its parts by the float position it read off the entity;
//     two paths that produce 12.0 and 11.999999 then disagree about whether a
//     part is there.
//   - Every traversal is over an ARRAY (the node table) or a fixed direction
//     list. The one map, `index`, is only ever point-queried.
//   - Merging picks the SMALLER node id as the surviving root, not the larger
//     tree. Union by size would be a tie-break on insertion order, and the
//     cluster id is user-visible in the log (and, at M2, will name a hidden
//     surface slot).
//   - The split scan is an ITERATIVE flood fill. The incumbent recurses, which
//     overflows on a large balancer; a compiled guest has the same problem
//     with a worse error message.
//
// NOTHING HERE IS AN ENTITY REFERENCE, and that is the structural fix for the
// incumbent's dominant crash class. A part is a surface index, two integer tile
// coordinates and a force index; the world is re-read at compile time. There is
// no cached LuaEntity and no cached LuaTransportLine to be invalidated by a
// removal path nobody thought of, so "stale reference" is not a bug this mod
// can have -- see agents/architecture/runtime-model.md, "The failure envelope".

// key identifies one tile: which surface, and which tile on it.
//
// Surface is part of the key because space platforms are surfaces: two parts at
// the same (x, y) on a platform and on Nauvis are not neighbours.
//
// FORCE IS NOT IN THE KEY, deliberately. Two parts cannot occupy one tile
// whatever their forces, so position alone identifies a part -- and the
// neighbour gate in main.go is a point query with a position and nothing else
// in hand. Force lives in `pforce` beside the node and is compared at every
// adjacency test instead, which is where it belongs: it decides what MERGES,
// not what EXISTS.
type key struct {
	s uint32
	x int32
	y int32
}

// The 4-neighbourhood, in a fixed order. Every traversal visits neighbours in
// this order, so two clients enumerate a component identically.
var dirs = [4][2]int32{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// The node table. Index 0 is a dummy so that a node id is never 0 and reads as
// a cluster number a human can hold in their head.
//
// Parallel slices rather than a slice of structs: TinyGo's -gc=leaking never
// returns memory, and this whole structure lives in every save, so the layout
// that allocates least wins.
var (
	parent []uint32 // union-find parent; parent[r] == r at a root
	// Active roots only, for verbose event snapshots without a node-table scan.
	rootPrev, rootNext []uint32
	rootHead           uint32
	csize              []uint32 // cluster size, meaningful only at a root
	ppos               []key    // where this node's part is
	pforce             []uint32 // which force built it; parts of two forces never merge
	alive              []bool   // false for a slot on the free list
	mark               []uint32 // flood-fill generation stamp
	// pvar is the sprite variation currently ON the entity, or 0 for "we have
	// not set one" -- a freshly built part, or a part registered by a
	// rebuild-from-world that has no idea what the last build put there. It is
	// one byte per part and it exists so that restyle() makes a host call only
	// for the parts whose PICTURE actually changed: a part added to the edge of
	// a 200-part balancer costs a handful of calls, not four hundred. See
	// skin.go.
	pvar []uint8
	// pprio is the part's INPUT/OUTPUT PRIORITY flag, 1 for on and 0 for off,
	// and it is a property of the PART rather than of the belt: every edge the
	// tile carries reads it, which on an engine that lets a part carry two
	// belts is both of them. It decides what plan.Build wires -- a priority
	// output is fed before the others and a priority input is drained before
	// them -- so it is in the compile fingerprint, and toggling it is a
	// recompile like any other edge edit.
	//
	// ONE BYTE, NOT A BOOL, because it is written into the sprite variation
	// beside the shape (skin.go) and read back out of it, and because a bool
	// slice costs the same byte. Zero for a fresh node, like every other slice
	// here: a part nobody has flagged is not a priority part.
	pprio  []uint8
	freeID []uint32 // reusable slots, LIFO -- deterministic
	// gen is the current flood-fill generation: every fill increments it and
	// stamps `mark`, so a fill needs no clearing pass over a slice that is as
	// long as the registry.
	//
	// IT WRAPS AT 2^32, and the wrap is deterministic rather than safe. A node
	// whose `mark` still holds the value `gen` had 4,294,967,296 fills ago reads
	// as already-visited and is dropped from its own cluster. Nothing here
	// defends against it because nothing can reach it: `initRegistry` resets it
	// to 0, a fill happens per queued cluster per flush, and a player editing
	// one balancer every tick for a 300-hour session (agents/performance/marathon-save.md, "The marathon
	// save", which models six edge-moving edits per player-hour) spends four
	// orders of magnitude less than that. It is written down because a future
	// caller that filled per PART rather than per cluster would move it two
	// orders closer, and because every client wraps on the same tick -- so the
	// failure would be a wrong answer, not a desync.
	gen uint32

	// index is the only map, and it is only ever point-queried: never ranged
	// over, never used to decide an order.
	index map[key]uint32

	nParts uint32

	// q is the flood fill's queue, kept alive so a split does not allocate.
	q []uint32
)

func init() { initRegistry() }

// initRegistry puts the registry back to empty.
//
// A named function rather than a bare `init` body because it is called twice:
// once at _initialize, and once from ensureRegistry when the guest comes back
// on a heap it cannot trust (a mod upgrade -- see rebuildFromWorld). Anything
// added here that only `init` did would be a half-reset registry, which is
// worse than either state.
func initRegistry() {
	resetClassifications()
	parent = parent[:0]
	rootPrev, rootNext = rootPrev[:0], rootNext[:0]
	rootHead = 0
	csize = csize[:0]
	ppos = ppos[:0]
	pforce = pforce[:0]
	alive = alive[:0]
	mark = mark[:0]
	pvar = pvar[:0]
	pprio = pprio[:0]
	freeID = freeID[:0]
	q = q[:0]
	gen = 0
	nParts = 0
	index = map[key]uint32{}

	// The dummy slot, so ids start at 1.
	parent = append(parent, 0)
	rootPrev, rootNext = append(rootPrev, 0), append(rootNext, 0)
	csize = append(csize, 0)
	ppos = append(ppos, key{})
	pforce = append(pforce, 0)
	alive = append(alive, false)
	mark = append(mark, 0)
	pvar = append(pvar, 0)
	pprio = append(pprio, 0)
}

func newNode(k key, f uint32) uint32 {
	if n := len(freeID); n > 0 {
		id := freeID[n-1]
		freeID = freeID[:n-1]
		parent[id], csize[id], ppos[id], alive[id], mark[id] = id, 1, k, true, 0
		pforce[id] = f
		// A reused slot knows nothing about the entity now standing on it.
		pvar[id], pprio[id] = 0, 0
		discardClassification(id)
		partSides[id], dirtyNext[id] = 0, 0
		dirtyPart[id], changedPart[id] = false, false
		priorityKnown[id] = false
		forgetOverLimit(id)
		workQueue[id] = queuedWork{}
		addRoot(id)
		return id
	}
	id := uint32(len(parent))
	parent = append(parent, id)
	rootPrev, rootNext = append(rootPrev, 0), append(rootNext, 0)
	csize = append(csize, 1)
	ppos = append(ppos, k)
	pforce = append(pforce, f)
	alive = append(alive, true)
	mark = append(mark, 0)
	pvar = append(pvar, 0)
	pprio = append(pprio, 0)
	appendClassNode()
	addRoot(id)
	return id
}

func addRoot(r uint32) {
	rootPrev[r], rootNext[r] = 0, rootHead
	if rootHead != 0 {
		rootPrev[rootHead] = r
	}
	rootHead = r
}

func removeRoot(r uint32) {
	prev, next := rootPrev[r], rootNext[r]
	if prev == 0 {
		rootHead = next
	} else {
		rootNext[prev] = next
	}
	if next != 0 {
		rootPrev[next] = prev
	}
	rootPrev[r], rootNext[r] = 0, 0
}

func freeNode(id uint32) {
	r := find(id)
	if ni, has := nets[r]; has && ni.mergeOwner == id {
		// A retry may outlive this node's retirement. Never let rollback
		// provenance refer to a later part that reuses the same id.
		ni.mergeOwner = 0
		nets[r] = ni
	}
	alive[id] = false
	csize[id] = 0
	workQueue[id].retired = true
	queueRetiredNode(id)
}

// find is the union-find lookup with path compression, iterative because the
// generated Lua inherits this call depth and Factorio's stack is not ours.
func find(a uint32) uint32 {
	r := a
	for parent[r] != r {
		r = parent[r]
	}
	for a != r {
		next := parent[a]
		parent[a] = r
		a = next
	}
	return r
}

// AddPart records a part at k, belonging to force f, and merges it with
// whatever it touches OF THE SAME FORCE.
//
// Returns false when the tile was already occupied, which is not an error: the
// same placement arrives twice whenever a script builds with raise_built (the
// engine event and the script event both fire) and the second one has to be a
// no-op rather than a second node. It is also how every reconcile path here
// stays idempotent -- an area clone and the per-entity clone event report the
// same part, and so do a revive and the robot build behind it.
func AddPart(k key, f uint32) bool {
	// NOTHING ON THE HIDDEN SURFACE IS EVER A BALANCER, whatever put it there --
	// a mod cloning an area onto it, a script building on it. The compiler is
	// the only thing that may place anything there, and this is the guard rather
	// than one in each caller because the consequence is item loss a player
	// cannot even see: `teardownNet` spills a network's contents beside the
	// CLUSTER, so a cluster registered on the hidden surface would return its
	// items to a surface no player can reach. It would also give that cluster a
	// bounding box inside the slot grid, and a teardown sweeps its box.
	if hiddenIdx != 0 && k.s == hiddenIdx {
		if verboseLog {
			logStart("refused a part on the hidden surface at ")
			logI(k.x)
			logS(",")
			logI(k.y)
			logS(" -- nothing there is ours but the compiler's own network")
			logEnd()
		}
		return false
	}
	if _, ok := index[k]; ok {
		return false
	}
	id := newNode(k, f)
	index[k] = id
	nParts++
	// Decide before the first union. A low-id bridge must not temporarily
	// inherit its first neighbour's net: a refused merge would then leave that
	// incumbent owned by the bridge, which is freed when the edit is reverted.
	neighborRoot := uint32(0)
	extension := true
	for i := 0; i < len(dirs); i++ {
		nb, ok := index[key{k.s, k.x + dirs[i][0], k.y + dirs[i][1]}]
		if !ok || pforce[nb] != f {
			continue
		}
		r := find(nb)
		if neighborRoot != 0 && neighborRoot != r {
			extension = false
			break
		}
		neighborRoot = r
	}
	merged := false
	safeExtension := extension && workQueue[id].pending == 0 &&
		(neighborRoot == 0 || workQueue[neighborRoot].pending == 0)
	// A real merge can also retain its sole populated predecessor. Zero-port
	// snapshots contribute bounds, counts and dirty work, not a second machine.
	safeMerge := !extension && !curveUndecided
	populated, networks := 0, 0
	var neighbors [4]uint32
	nneighbors := 0
	for i := 0; safeMerge && i < len(dirs); i++ {
		nb, ok := index[key{k.s, k.x + dirs[i][0], k.y + dirs[i][1]}]
		if !ok || pforce[nb] != f {
			continue
		}
		r := find(nb)
		seen := false
		for _, prev := range neighbors[:nneighbors] {
			seen = seen || prev == r
		}
		if seen {
			continue
		}
		neighbors[nneighbors] = r
		nneighbors++
		c := &classes[r]
		if !c.valid || !c.fits || workQueue[r].pending != 0 {
			safeMerge = false
			break
		}
		if len(c.edges) != 0 {
			populated++
		}
		if ni, has := nets[r]; has {
			networks++
			if ni.condemned || ni.fp != c.fp {
				safeMerge = false
			}
		}
		safeMerge = safeMerge && populated <= 1 && networks <= 1
	}
	if safeMerge {
		classes[id] = classification{valid: true, surf: k.s, force: f,
			x0: k.x, y0: k.y, x1: k.x, y1: k.y}
		classes[id].summarize()
	}
	// A lone incumbent can survive an extension until compile compares edges.
	// Merges with multiple populated predecessors and batches already requiring
	// teardown keep the all-down-before-build path and its unowned carry pools.
	for i := 0; i < len(dirs); i++ {
		nb, ok := index[key{k.s, k.x + dirs[i][0], k.y + dirs[i][1]}]
		if !ok || pforce[nb] != f {
			continue
		}
		ra, rb := find(id), find(nb)
		if ra == rb {
			continue
		}
		// Smaller id survives: the cluster id a player sees does not jump
		// around because of which side happened to be bigger.
		if rb < ra {
			ra, rb = rb, ra
		}
		if safeMerge {
			joinClassification(ra, rb)
		} else if safeExtension {
			moveClassification(neighborRoot, ra)
		} else {
			discardClassification(ra)
			discardClassification(rb)
		}
		workQueue[ra].pending += workQueue[rb].pending
		workQueue[ra].strandedN += workQueue[rb].strandedN
		workQueue[rb].pending = 0
		workQueue[rb].strandedN = 0
		parent[rb] = ra
		removeRoot(rb)
		csize[ra] += csize[rb]
		_, hasA := nets[ra]
		netB, hasB := nets[rb]
		if (safeExtension || safeMerge) && !(hasA && hasB) {
			if safeMerge && hasA {
				ni := nets[ra]
				if ni.mergeOwner == 0 {
					ni.mergeOwner = ra
				}
				nets[ra] = ni
			}
			if hasB {
				if safeMerge && netB.mergeOwner == 0 {
					netB.mergeOwner = rb
				}
				nets[ra] = netB
				delete(nets, rb)
				forgetOverLimit(rb)
			}
		} else {
			markDead(ra)
			markDead(rb)
		}
		logMerge(ra, rb)
		merged = true
	}
	if !merged {
		logFormed(id)
	}
	r := find(id)
	if classes[r].valid {
		classes[r].include(k)
		classes[r].counts[0]++
	}
	dirtyPartsNear(k)
	queueLive(r)
	return true
}

// RemovePart drops the part at k and re-establishes the clusters left behind.
//
// The cluster the part belonged to is REBUILT rather than patched: union-find
// has no delete, so every surviving node reachable from one of the removed
// part's neighbours is re-pointed at the smallest id in its own component. That
// is O(cluster), never O(map), and it is the only place a split can be
// discovered.
func RemovePart(k key) bool { return removePart(k, 0) }

// RemovePartMinedBy is RemovePart for the one removal that has somebody to
// credit: a PLAYER mining the part, `player` being the event's `player_index`.
//
// It records the miner for EVERY part a player mines, not only for the one that
// dissolves the cluster -- see the note at the call site below, which is the
// field-reported defect. Every other removal path calls RemovePart, passes zero,
// and is byte-for-byte what it was.
func RemovePartMinedBy(k key, player uint32) bool { return removePart(k, player) }

func removePart(k key, player uint32) bool {
	id, ok := index[k]
	if !ok {
		return false
	}
	oldRoot := find(id)
	// Cached zero sides are insufficient after a refusal: the incumbent may
	// still own an interface on this tile. Require agreement with its snapshot.
	leaf := !curveUndecided && classes[oldRoot].valid && partSides[id] == 0 &&
		workQueue[oldRoot].pending == 0
	if ni, has := nets[oldRoot]; has && (ni.condemned || ni.fp != classes[oldRoot].fp) {
		leaf = false
	}
	neighbor, degree := uint32(0), 0
	for _, d := range dirs {
		if nb, has := index[key{k.s, k.x + d[0], k.y + d[1]}]; has && pforce[nb] == pforce[id] {
			neighbor, degree = nb, degree+1
		}
	}
	if leaf && degree == 1 {
		// Even a geometric leaf can have union-find children. Flatten the
		// surviving component before its id can be recycled.
		c := &classes[oldRoot]
		prev := uint32(0)
		for n := c.head; n != 0; n = dirtyNext[n] {
			if n == id {
				if prev == 0 {
					c.head = dirtyNext[n]
				} else {
					dirtyNext[prev] = dirtyNext[n]
				}
				if c.tail == n {
					c.tail = prev
				}
				break
			}
			prev = n
		}
		dirtyPart[id], dirtyNext[id] = false, 0
		c.counts[0]--
		removeRoot(oldRoot)
		delete(index, k)
		freeNode(id)
		nParts--
		gen++
		r := rebuildComponent(neighbor)
		// rebuildComponent's queue is already the complete membership list;
		// use it for exact bounds without another flood or any host calls.
		c.x0, c.x1 = ppos[r].x, ppos[r].x
		c.y0, c.y1 = ppos[r].y, ppos[r].y
		for _, n := range q {
			c.include(ppos[n])
		}
		moveClassification(oldRoot, r)
		if r != oldRoot {
			if ni, has := nets[oldRoot]; has {
				nets[r] = ni
				delete(nets, oldRoot)
			}
		}
		noteMinedByPlayer(k, pforce[id], player)
		logShrunk(r)
		dirtyPartsNear(k)
		queueLive(r)
		return true
	}
	removeRoot(oldRoot)
	discardClassification(oldRoot)
	f := pforce[id]
	delete(index, k)
	freeNode(id)
	nParts--

	// Seeds in the fixed direction order, so components are discovered in the
	// same order everywhere. Only same-force neighbours: a part of another
	// force was never in this cluster and its network must not be disturbed.
	var seeds [4]uint32
	nseeds := 0
	for i := 0; i < len(dirs); i++ {
		if nb, ok := index[key{k.s, k.x + dirs[i][0], k.y + dirs[i][1]}]; ok && pforce[nb] == f {
			seeds[nseeds] = nb
			nseeds++
		}
	}
	// The old root's network comes down unconditionally: its visible interfaces
	// stand on tiles that may now belong to a different cluster, or to none.
	markDead(oldRoot)
	// THE MINER IS RECORDED FOR EVERY PART A PLAYER MINES, AND NOT ONLY FOR THE
	// ONE THAT DISSOLVES THE CLUSTER. That distinction was the miner's pocket's
	// first cut and it made the feature almost inert in play, which is how a
	// player found it and seven suites did not.
	//
	// Taking a balancer apart by hand is a sequence of SHRINKS ending in one
	// dissolve, and a shrink makes the machine SMALLER: the network that goes
	// back up has fewer ports and less line, so what the survivor cannot take
	// falls through to the spill -- carry.go's fourth decision, "what does not
	// fit". Measured on a saturated four-part 4x4 mined one part per tick: 8,
	// 152 and 46 items on the floor across the three shrinks, and 26 left for
	// the dissolve to hand over. The pocket got the dregs and the floor got the
	// machine.
	//
	// PRECEDENCE IS UNCHANGED AND IS WHAT MAKES THIS SAFE: a claimed pool is
	// never pocketed. The survivor still claims the shrink's pool and takeCarry
	// still puts back everything that fits; the beneficiary is consulted by
	// settleCarry over the remainder alone. So a shrink that fits entirely --
	// which is the ordinary one -- reaches none of this and costs nothing.
	//
	// Recorded rather than acted on: carry.go decides at settle time, and a
	// neighbouring cluster that claims the pool geometrically still wins.
	//
	// THE FORCE GOES WITH IT, and `f` above is why this is free: it is
	// `pforce[id]`, read before the node was freed, registry state rather than a
	// host call. A neighbouring cluster of ANOTHER force has an adjacent -- and
	// diagonally an overlapping -- bounding box by construction, so without it a
	// claim could be answered by the wrong balancer's pool. See carry.go's
	// beneficiary section.
	noteMinedByPlayer(k, f, player)
	if nseeds == 0 {
		logDissolvedBy(oldRoot, player)
		dirtyPartsNear(k)
		return true
	}

	gen++
	var roots [4]uint32
	nroots := 0
	for i := 0; i < nseeds; i++ {
		if mark[seeds[i]] == gen {
			continue
		}
		r := rebuildComponent(seeds[i])
		roots[nroots] = r
		nroots++
		markLive(r)
	}
	// Ascending, so the log reads the same whichever neighbour the flood
	// happened to start from. Four elements at most.
	for i := 1; i < nroots; i++ {
		v := roots[i]
		j := i - 1
		for j >= 0 && roots[j] > v {
			roots[j+1] = roots[j]
			j--
		}
		roots[j+1] = v
	}
	if nroots == 1 {
		logShrunk(roots[0])
	} else {
		logSplit(oldRoot, roots[:nroots])
	}
	dirtyPartsNear(k)
	return true
}

// rebuildComponent floods the component containing start, points every node in
// it at the smallest id it contains, and returns that id.
func rebuildComponent(start uint32) uint32 {
	q = q[:0]
	q = append(q, start)
	mark[start] = gen
	root := start
	f := pforce[start]
	for i := 0; i < len(q); i++ {
		workFlooded++
		n := q[i]
		k := ppos[n]
		for d := 0; d < len(dirs); d++ {
			nb, ok := index[key{k.s, k.x + dirs[d][0], k.y + dirs[d][1]}]
			if !ok || mark[nb] == gen || pforce[nb] != f {
				continue
			}
			mark[nb] = gen
			if nb < root {
				root = nb
			}
			q = append(q, nb)
		}
	}
	workQueue[root].pending = 0
	workQueue[root].strandedN = 0
	for i := 0; i < len(q); i++ {
		parent[q[i]] = root
		workQueue[root].pending += ownPending(q[i])
		if workQueue[q[i]].stranded {
			workQueue[root].strandedN++
		}
	}
	csize[root] = uint32(len(q))
	addRoot(root)
	return root
}

// ---------------------------------------------------------------------------
// Walks over the registry.
//
// Every one of these is driven by the NODE ARRAY in id order, never by `index`.
// That is not a style preference: a Go map's iteration order is randomised, and
// a lockstep simulation in which one client tears clusters down in a different
// order from another is a desync, not a cosmetic difference.
//
// They are all O(parts on the map). Every caller is a rare event -- a surface
// being deleted, an area being cloned, a mod being upgraded -- and never the
// build/mine path, which stays a point query.
// ---------------------------------------------------------------------------

// liveRootList appends every live cluster root, in ascending node id.
func liveRootList(dst []uint32) []uint32 {
	dst = dst[:0]
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if alive[i] && parent[i] == i {
			dst = append(dst, i)
		}
	}
	return dst
}

// rootsOnSurface appends every live root whose OWN part sits on surface si.
//
// A cluster is single-surface by construction (adjacency is per surface), so
// testing the root's tile is testing the cluster.
func rootsOnSurface(dst []uint32, si uint32) []uint32 {
	dst = dst[:0]
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if alive[i] && parent[i] == i && ppos[i].s == si {
			dst = append(dst, i)
		}
	}
	return dst
}

// partsOnSurface appends the tile of every live part on surface si.
func partsOnSurface(dst []key, si uint32) []key {
	dst = dst[:0]
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if alive[i] && ppos[i].s == si {
			dst = append(dst, ppos[i])
		}
	}
	return dst
}

// partsInBox appends the tile of every live part on surface si inside the
// closed tile box [x0,x1] x [y0,y1].
func partsInBox(dst []key, si uint32, x0, y0, x1, y1 int32) []key {
	dst = dst[:0]
	for i := uint32(1); i < uint32(len(parent)); i++ {
		if !alive[i] {
			continue
		}
		k := ppos[i]
		if k.s != si || k.x < x0 || k.x > x1 || k.y < y0 || k.y > y1 {
			continue
		}
		dst = append(dst, k)
	}
	return dst
}

// Snapshot returns the live cluster sizes, ascending.
//
// The active-root chain is updated with topology edits. Walking it rather than
// every node keeps verbose logging independent of a large cluster's tile count.
func Snapshot(dst []uint32) []uint32 {
	dst = dst[:0]
	for r := rootHead; r != 0; r = rootNext[r] {
		dst = append(dst, csize[r])
	}
	// Insertion sort: these lists are short, and it costs no code size.
	for i := 1; i < len(dst); i++ {
		v := dst[i]
		j := i - 1
		for j >= 0 && dst[j] > v {
			dst[j+1] = dst[j]
			j--
		}
		dst[j+1] = v
	}
	return dst
}
