package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

// Classification is independent of construction: a valid answer can be empty,
// refused, or waiting for a failed build to be retried. No Lua handles live here.
type classification struct {
	valid          bool
	surf, force    uint32
	x0, y0, x1, y1 int32
	edges          []plan.Edge
	curved         []bool
	fp             uint64
	ports          plan.Ports
	fits           bool
	counts         [5]uint32
	head, tail     uint32
}

type sideDelta struct {
	id    uint32
	state uint16 // four nibbles: exists, output, curved, priority
}

var (
	classes        []classification
	partSides      []uint16
	priorityKnown  []bool
	dirtyNext      []uint32
	dirtyPart      []bool
	changedPart    []bool
	dirtyTiles     []key
	sideDeltas     []sideDelta
	deltaEdges     []plan.Edge
	deltaCurved    []bool
	mergedEdges    []plan.Edge
	mergedCurved   []bool
	freeClassEdges []classification
	oracleClass    classification
	oracleRoot     uint32
	// Sticky within one read. Fresh callers must check this immediately after
	// classifyEdges; a failed query is never an authoritative absence.
	classificationReadOK                                 bool
	preflightBlocked                                     []uint32
	workFlooded, workClassified, workRestyled, workEdges uint64
	workBuilds, workTeardowns                            uint64
)

func resetClassifications() {
	for r := range classes {
		discardClassification(uint32(r))
	}
	classes = classes[:0]
	partSides, dirtyNext = partSides[:0], dirtyNext[:0]
	dirtyPart, changedPart = dirtyPart[:0], changedPart[:0]
	priorityKnown = priorityKnown[:0]
	workQueue = workQueue[:0]
	retryRoots = retryRoots[:0]
	retiredCandidates = retiredCandidates[:0]
	preflightBlocked = preflightBlocked[:0]
	appendClassNode()
}

func appendClassNode() {
	workQueue = append(workQueue, queuedWork{})
	classes = append(classes, classification{})
	partSides = append(partSides, 0)
	priorityKnown = append(priorityKnown, false)
	dirtyNext = append(dirtyNext, 0)
	dirtyPart = append(dirtyPart, false)
	changedPart = append(changedPart, false)
}

func clearClassDirty(r uint32) {
	c := &classes[r]
	for id := c.head; id != 0; {
		next := dirtyNext[id]
		dirtyPart[id], dirtyNext[id] = false, 0
		id = next
	}
	c.head, c.tail = 0, 0
}

func invalidateClassification(r uint32) {
	if r != 0 && int(r) < len(classes) {
		classes[r].valid = false
	}
}

// Called before a node is freed, so no pending list can retain a reused id.
func discardClassification(r uint32) {
	clearClassDirty(r)
	c := &classes[r]
	if cap(c.edges) != 0 {
		freeClassEdges = append(freeClassEdges, classification{edges: c.edges[:0], curved: c.curved[:0]})
	}
	classes[r] = classification{}
}

// Call for EVERY affected old root before any reparenting. It clears the dirty
// owner lists as well as invalidating snapshots; node-local known priority stays.
func resetAffectedClassification(r uint32) { discardClassification(r) }

func acquireClassEdges(c *classification, need int) {
	if cap(c.edges) != 0 || need == 0 || len(freeClassEdges) == 0 {
		return
	}
	best := 0
	for i := 1; i < len(freeClassEdges); i++ {
		a, b := cap(freeClassEdges[i].edges), cap(freeClassEdges[best].edges)
		if (a >= need && (b < need || a < b)) || (a < need && b < need && a > b) {
			best = i
		}
	}
	c.edges, c.curved = freeClassEdges[best].edges, freeClassEdges[best].curved
	last := len(freeClassEdges) - 1
	freeClassEdges[best] = freeClassEdges[last]
	freeClassEdges[last] = classification{}
	freeClassEdges = freeClassEdges[:last]
}

func moveClassification(from, to uint32) {
	if from == to {
		return
	}
	discardClassification(to)
	classes[to] = classes[from]
	classes[from] = classification{}
	if fp, refused := overLimit[from]; refused {
		overLimit[to] = fp
		forgetOverLimit(from)
	}
}

// At most one donor has edges. Preserve its sorted snapshot and concatenate
// dirty work before releasing the other snapshot's storage.
func joinClassification(a, b uint32) {
	if len(classes[b].edges) != 0 {
		classes[a], classes[b] = classes[b], classes[a]
	}
	c, donor := &classes[a], &classes[b]
	c.include(key{c.surf, donor.x0, donor.y0})
	c.include(key{c.surf, donor.x1, donor.y1})
	for i := range c.counts {
		c.counts[i] += donor.counts[i]
	}
	if donor.head != 0 {
		if c.tail != 0 {
			dirtyNext[c.tail] = donor.head
		} else {
			c.head = donor.head
		}
		c.tail = donor.tail
		donor.head, donor.tail = 0, 0
	}
	discardClassification(b)
	forgetOverLimit(b)
}

// The event hook takes a REGISTERED part tile, not the edited belt's tile.
// The dirty bit covers classification and sprite work without forgetting the
// known priority encoded by pvar. Repeated neighbourhood hits cost no allocation.
func dirtyEdgesAt(k key) {
	id, ok := index[k]
	if !ok {
		return
	}
	r := find(id)
	if !dirtyPart[id] {
		c := &classes[r]
		if c.tail != 0 {
			dirtyNext[c.tail] = id
		} else {
			c.head = id
		}
		c.tail = id
		dirtyPart[id] = true
	}
	queueLive(r)
}

func dirtyPartsNear(k key) {
	for y := int32(-2); y <= 2; y++ {
		for x := int32(-2); x <= 2; x++ {
			dirtyEdgesAt(key{k.s, k.x + x, k.y + y})
		}
	}
}

func (c *classification) include(k key) {
	if k.x < c.x0 {
		c.x0 = k.x
	}
	if k.x > c.x1 {
		c.x1 = k.x
	}
	if k.y < c.y0 {
		c.y0 = k.y
	}
	if k.y > c.y1 {
		c.y1 = k.y
	}
}

func classifyTile(s fkapi.LuaSurface, k key) uint16 {
	workClassified++
	state := uint16(0)
	if !priorityKnown[index[k]] {
		classificationReadOK = false
	}
	prio := pprio[index[k]] != 0
	for d := range dirs {
		nx, ny := k.x+dirs[d][0], k.y+dirs[d][1]
		if _, interior := index[key{k.s, nx, ny}]; interior {
			continue
		}
		out, found, curved := classifySide(s, k.s, nx, ny, dirOf[d])
		if !found {
			continue
		}
		v := uint16(1)
		if out {
			v |= 2
		}
		if curved {
			v |= 4
		}
		if prio {
			v |= 8
		}
		state |= v << (4 * d)
	}
	return state
}

func stateEdges(k key, state uint16, edges []plan.Edge, curves []bool) ([]plan.Edge, []bool) {
	for d := range dirs {
		v := (state >> (4 * d)) & 15
		if v&1 == 0 {
			continue
		}
		dir, out := dirOf[d], v&2 != 0
		if !out {
			dir = plan.Opposite(dir)
		}
		edges = append(edges, plan.Edge{TileX: k.x, TileY: k.y, Dir: dir, Out: out, Prio: v&8 != 0})
		curves = append(curves, v&4 != 0)
	}
	return edges, curves
}

func sideCount(state uint16) int {
	n := 0
	for d := range dirs {
		if state&(1<<(4*d)) != 0 {
			n++
		}
	}
	return n
}

// In-place heapsort keeps the parallel curve flags aligned, with bounded stack
// and O(E log E) work even on a refused cluster with arbitrarily many edges.
func sortEdges(edges []plan.Edge, curves []bool) {
	sift := func(root, end int) {
		for child := root*2 + 1; child < end; child = root*2 + 1 {
			workEdges++
			if child+1 < end && edgeLess(edges[child], edges[child+1]) {
				child++
			}
			if !edgeLess(edges[root], edges[child]) {
				break
			}
			edges[root], edges[child] = edges[child], edges[root]
			curves[root], curves[child] = curves[child], curves[root]
			root = child
		}
	}
	for i := len(edges)/2 - 1; i >= 0; i-- {
		sift(i, len(edges))
	}
	for end := len(edges) - 1; end > 0; end-- {
		workEdges++
		edges[0], edges[end] = edges[end], edges[0]
		curves[0], curves[end] = curves[end], curves[0]
		sift(0, end)
	}
}

func (c *classification) summarize() {
	c.fp = fingerprint(c.edges)
	c.ports, c.fits = plan.ShapeEdges(c.edges)
}

// Restyle FIRST: a newly revived/registered tile may carry a persisted priority.
// Commit side states only after every read succeeded, so retries cannot lose a
// delta. The unchanged arm never touches the sparse edges or scans the bounds.
func prepareClassification(r uint32, surf fkapi.LuaSurface) bool {
	classificationReadOK = true
	c := &classes[r]
	if !c.valid || curveUndecided {
		c.valid = false
		tiles := collectCluster(r)
		if !restyleTiles(r, tiles) {
			return false
		}
		edges := classifyEdges(surf, tiles, pforce[r])
		if !classificationReadOK {
			return false
		}
		acquireClassEdges(c, len(edges))
		c.surf, c.force = ppos[r].s, pforce[r]
		c.edges = append(c.edges[:0], edges...)
		c.curved = append(c.curved[:0], edgeCurved...)
		c.x0, c.y0, c.x1, c.y1 = clusterBox(tiles)
		c.counts = [5]uint32{}
		for _, k := range tiles {
			partSides[index[k]] = 0
		}
		for i, e := range edges {
			workEdges++
			id := index[key{ppos[r].s, e.TileX, e.TileY}]
			dir := e.Dir
			if !e.Out {
				dir = plan.Opposite(dir)
			}
			d, _ := dirIndex(dir)
			v := uint16(1)
			if e.Out {
				v |= 2
			}
			if edgeCurved[i] {
				v |= 4
			}
			if e.Prio {
				v |= 8
			}
			partSides[id] |= v << (4 * d)
		}
		for _, k := range tiles {
			c.counts[sideCount(partSides[index[k]])]++
		}
		c.summarize()
		c.valid = !curveUndecided
		clearClassDirty(r)
		return true
	}
	if c.head == 0 {
		return true
	}
	dirtyTiles = dirtyTiles[:0]
	for id := c.head; id != 0; id = dirtyNext[id] {
		dirtyTiles = append(dirtyTiles, ppos[id])
	}
	if !restyleTiles(r, dirtyTiles) {
		return false
	}
	classificationReadOK = true
	forceFilter = fkapi.OfNumber(float64(pforce[r]))
	sideDeltas = sideDeltas[:0]
	for id := c.head; id != 0; id = dirtyNext[id] {
		state := classifyTile(surf, ppos[id])
		if state != partSides[id] {
			sideDeltas = append(sideDeltas, sideDelta{id, state})
		}
	}
	if !classificationReadOK {
		return false
	}
	if len(sideDeltas) == 0 {
		clearClassDirty(r)
		return true
	}
	deltaEdges, deltaCurved = deltaEdges[:0], deltaCurved[:0]
	for _, d := range sideDeltas {
		old := sideCount(partSides[d.id])
		c.counts[old]--
		c.counts[sideCount(d.state)]++
		partSides[d.id], changedPart[d.id] = d.state, true
		deltaEdges, deltaCurved = stateEdges(ppos[d.id], d.state, deltaEdges, deltaCurved)
	}
	sortEdges(deltaEdges, deltaCurved)
	mergedEdges, mergedCurved = mergedEdges[:0], mergedCurved[:0]
	j := 0
	for i, e := range c.edges {
		workEdges++
		if changedPart[index[key{ppos[r].s, e.TileX, e.TileY}]] {
			continue
		}
		for j < len(deltaEdges) && edgeLess(deltaEdges[j], e) {
			workEdges++
			mergedEdges = append(mergedEdges, deltaEdges[j])
			mergedCurved = append(mergedCurved, deltaCurved[j])
			j++
		}
		mergedEdges = append(mergedEdges, e)
		mergedCurved = append(mergedCurved, c.curved[i])
	}
	mergedEdges = append(mergedEdges, deltaEdges[j:]...)
	mergedCurved = append(mergedCurved, deltaCurved[j:]...)
	acquireClassEdges(c, len(mergedEdges))
	c.edges = append(c.edges[:0], mergedEdges...)
	c.curved = append(c.curved[:0], mergedCurved...)
	for _, d := range sideDeltas {
		changedPart[d.id] = false
	}
	c.summarize()
	clearClassDirty(r)
	return true
}

// Metadata belongs to the last complete fresh oracle call, not to an arbitrary
// edge slice or to speculative answers subsequently edited by priority probes.
func recordOracleClass(tiles []key, force uint32) {
	oracleClass, oracleRoot = classification{}, 0
	if !classificationReadOK || len(tiles) == 0 {
		return
	}
	o := &oracleClass
	o.surf, o.force = tiles[0].s, force
	oracleRoot = find(index[tiles[0]])
	o.x0, o.y0, o.x1, o.y1 = clusterBox(tiles)
	o.counts[0] = uint32(len(tiles))
	for i := 0; i < len(edgeBuf); {
		j := i + 1
		workEdges++
		for j < len(edgeBuf) && edgeBuf[j].TileX == edgeBuf[i].TileX && edgeBuf[j].TileY == edgeBuf[i].TileY {
			workEdges++
			j++
		}
		o.counts[j-i]++
		o.counts[0]--
		i = j
	}
	o.fp = fingerprint(edgeBuf)
	o.ports, o.fits = plan.ShapeEdges(edgeBuf)
}

// Fresh oracle results never alias the snapshot. Audits can use this before
// repairing drift; it deliberately does not adopt the oracle's answer.
func classificationMatches(r uint32, edges []plan.Edge) bool {
	c := &classes[r]
	o := &oracleClass
	if len(edges) != len(edgeBuf) || (len(edges) != 0 && &edges[0] != &edgeBuf[0]) {
		return false
	}
	if !classificationReadOK || !c.valid || c.head != 0 || oracleRoot != r ||
		c.surf != o.surf || c.force != o.force || c.surf != ppos[r].s || c.force != pforce[r] ||
		c.x0 != o.x0 || c.y0 != o.y0 || c.x1 != o.x1 || c.y1 != o.y1 ||
		c.counts != o.counts || c.ports != o.ports || c.fits != o.fits || c.fp != o.fp ||
		len(c.edges) != len(edges) || len(c.curved) != len(edgeCurved) {
		return false
	}
	tiles := uint32(0)
	for _, n := range o.counts {
		tiles += n
	}
	if tiles != csize[r] {
		return false
	}
	for i := range edges {
		workEdges++
		if c.edges[i] != edges[i] || c.curved[i] != edgeCurved[i] {
			return false
		}
	}
	return true
}
