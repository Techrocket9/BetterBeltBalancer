package plan

import "math"

// The flow model: the network Build emitted, solved to its steady state with
// capacity and back-pressure in it.
//
// # Why Propagate is not enough
//
// Propagate and PropagateLoop are LINEAR. They average a pair of rows, which is
// what a splitter does while both of its outputs can take their share, and they
// have no way to say that an output is full or that a line carries at most one
// belt. That is exactly the half a priority splitter lives in: it fills its
// priority side FIRST and gives the other side the overflow, so the answer
// depends on whether the priority side is at capacity. PropagateLoop's own
// header already records the same hole from the other direction -- it is valid
// only for n <= m, because past that the spare outputs dead-end and back up.
//
// So this is a fixed point over two quantities per link: how much the
// downstream will accept (`acc`, computed backwards) and how much is actually
// moving (`flow`, computed forwards). Back-pressure is the two meeting.
//
// # It is a model of a splitter, not a splitter
//
// Four rules about Factorio go in and nothing else does:
//
//   - a splitter divides its input between its two outputs equally, and gives
//     the whole of it to one output when the other cannot take its share;
//   - with an output priority it fills the priority side as far as that side
//     will take, and the other side gets what is left;
//   - with an input priority it draws from the priority side first when it
//     cannot take everything on offer;
//   - a belt line carries at most one belt, so a splitter carries at most two.
//
// Everything else is the wiring, and the wiring is what can be wrong. What this
// cannot say is that the ENGINE agrees -- an in-game suite is the only thing
// that can, and TestPlainShapesMatchTheLinearModel is the anchor that says the
// model at least agrees with the one the M2 rigs were measured against.
//
// It lives in a test file rather than beside Propagate in plan.go, and that is
// measured rather than tidy. Propagate really is dead-code-eliminated out of
// the wasm: deleting it moves the `code` section not one byte. This is not --
// something in it, the closure buildFlow wires its links with or the map it
// indexes tiles by, survives TinyGo's reachability pass and costs 1,103 bytes
// of compiled code for a function no guest can call. A _test.go file cannot
// reach the wasm at all, which is the whole of the fix.

// flowIters caps the relaxation. The sweeps are over a graph whose longest path
// is a few dozen hops and whose only feedback is the loopback, so a converging
// network settles in the low hundreds; the cap is an order above that so that
// reaching it means the iteration does not converge at all, which the caller is
// told about rather than hidden from.
const flowIters = 4000

// flowEps is the settling tolerance and also the tolerance the callers compare
// against. Below it the sweeps are moving less than an item per ten thousand
// belts.
const flowEps = 1e-12

// A link is one belt line between two ops: a tile boundary, or a linked belt's
// teleport. from or to is -1 where the world is at that end.
type flowLink struct {
	from, to  int32
	acc, flow float64
}

type flowNode struct {
	in, out    [2]int32
	nin, nout  int8
	proto      Proto
	outP, inP  int8
	isTeleport bool
}

type flowGraph struct {
	nodes []flowNode
	links []flowLink
	src   []int32 // link per input edge, in edge order
	sink  []int32 // link per output edge, in edge order
}

// tileOf returns the tiles an op occupies and, for a splitter, in north-to-south
// order -- which is the order Op.OutPrio names with PrioLeft and PrioRight.
func tilesOfOp(o Op) (x int32, y [2]int32, n int) {
	x = int32(math.Floor(o.X))
	if o.Proto == ProtoSplitter {
		ly := int32(math.Floor(o.Y))
		return x, [2]int32{ly - 1, ly}, 2
	}
	return x, [2]int32{int32(math.Floor(o.Y)), 0}, 1
}

// buildFlow turns the op list into the graph. It reads POSITIONS, which is the
// point: an op that moved a tile away from its neighbour stops being connected
// here exactly as it would stop being connected in the game.
func buildFlow(ops []Op) *flowGraph {
	g := &flowGraph{nodes: make([]flowNode, len(ops))}
	owner := map[[2]int32]int32{} // tile -> op<<1 | slot
	for i := range ops {
		o := ops[i]
		g.nodes[i].proto = o.Proto
		g.nodes[i].outP, g.nodes[i].inP = o.OutPrio, o.InPrio
		g.nodes[i].isTeleport = o.Link == LinkInput
		g.nodes[i].in = [2]int32{-1, -1}
		g.nodes[i].out = [2]int32{-1, -1}
		if o.Visible {
			continue
		}
		x, ys, n := tilesOfOp(o)
		for s := 0; s < n; s++ {
			owner[[2]int32{x, ys[s]}] = int32(i)<<1 | int32(s)
		}
	}

	add := func(from, to int32, toSlot int) int32 {
		id := int32(len(g.links))
		g.links = append(g.links, flowLink{from: from, to: to})
		if from >= 0 {
			nd := &g.nodes[from]
			nd.out[nd.nout] = id
			nd.nout++
		}
		if to >= 0 {
			nd := &g.nodes[to]
			nd.in[toSlot] = id
			if int8(toSlot)+1 > nd.nin {
				nd.nin = int8(toSlot) + 1
			}
		}
		return id
	}

	// One outgoing link per output the op has: the teleport for a linked belt's
	// input end, and otherwise the tile east of each tile it occupies. An op
	// with nothing east of it gets a link to the world, which for a visible
	// output end is the player's belt and everywhere else is a dead end.
	for i := range ops {
		o := ops[i]
		if o.Link == LinkInput {
			if o.Pair < 0 {
				continue
			}
			add(int32(i), o.Pair, 0)
			continue
		}
		if o.Visible {
			// A visible output end drains into the world.
			g.sink = append(g.sink, add(int32(i), -1, 0))
			continue
		}
		x, ys, n := tilesOfOp(o)
		for s := 0; s < n; s++ {
			w, ok := owner[[2]int32{x + 1, ys[s]}]
			if !ok {
				add(int32(i), -1, 0)
				continue
			}
			add(int32(i), w>>1, int(w&1))
		}
	}
	// A splitter whose west neighbour is missing on one tile has one input, and
	// it must be the right one: the model's in[0] is the north tile whatever is
	// standing there. Slots are filled by the adding side above, so the only
	// thing left is to make nin cover a used slot 1 with an empty slot 0.
	for i := range g.nodes {
		if g.nodes[i].in[1] >= 0 && g.nodes[i].nin < 2 {
			g.nodes[i].nin = 2
		}
	}
	// The sources, in edge order. Build appends the visible interfaces last and
	// in the order of the edge list, so the k-th visible input end is the k-th
	// input edge whatever port the plan gave it.
	for i := range ops {
		if ops[i].Visible && ops[i].Link == LinkInput {
			g.src = append(g.src, add(-1, int32(i), 0))
		}
	}
	return g
}

// splitOut is the output rule: a belts arriving, c0 and c1 acceptable on the
// two sides.
func splitOut(a, c0, c1 float64, prio int8) (float64, float64) {
	switch prio {
	case PrioLeft:
		o0 := math.Min(a, c0)
		return o0, math.Min(a-o0, c1)
	case PrioRight:
		o1 := math.Min(a, c1)
		return math.Min(a-o1, c0), o1
	}
	return math.Min(c0, math.Max(a/2, a-c1)), math.Min(c1, math.Max(a/2, a-c0))
}

// shareIn is the same rule seen from the other side: the node can take c in
// total, the two inputs are currently pushing f0 and f1, and the answer is how
// much each may push. Without a priority an input may always push its fair
// half, and more when the other one is not using its own.
func shareIn(c, f0, f1 float64, prio int8) (float64, float64) {
	switch prio {
	case PrioLeft:
		return math.Min(1, c), math.Min(1, math.Max(0, c-f0))
	case PrioRight:
		return math.Min(1, math.Max(0, c-f1)), math.Min(1, c)
	}
	return math.Min(1, math.Max(c/2, c-f1)), math.Min(1, math.Max(c/2, c-f0))
}

// Simulate solves the network for one load.
//
// in is what each input edge is offered, out is what each output edge's belt
// will take -- 1 for a free express belt and 0 for one that is blocked, both in
// units of a full belt. Both are in EDGE ORDER, which is the order the caller
// wrote them, not the port order the plan chose.
//
// delivered and taken come back the same way: what each output edge received
// and what each input edge was actually able to put in. They differ from `in`
// exactly when the network is backed up, and their totals are equal in the
// steady state, which is what conserved means here.
func Simulate(ops []Op, in, out []float64) (delivered, taken []float64, ok bool) {
	g := buildFlow(ops)
	if len(in) != len(g.src) || len(out) != len(g.sink) {
		return nil, nil, false
	}
	for i := range g.links {
		g.links[i].acc = 1
	}
	for i, l := range g.sink {
		g.links[l].acc = math.Min(1, out[i])
	}
	// A link into the world that is not a sink is a dead-ended port: it takes
	// nothing, which is what makes its splitter give everything to the other
	// side and, when that side is full too, back up.
	isSink := make([]bool, len(g.links))
	for _, l := range g.sink {
		isSink[l] = true
	}
	for i := range g.links {
		if g.links[i].to < 0 && !isSink[i] {
			g.links[i].acc = 0
		}
	}

	settled := false
	for it := 0; it < flowIters && !settled; it++ {
		var worst float64
		// Backwards: what each node will accept on each of its inputs.
		for i := len(g.nodes) - 1; i >= 0; i-- {
			nd := &g.nodes[i]
			if nd.nin == 0 {
				continue
			}
			var c float64
			for s := int8(0); s < nd.nout; s++ {
				if nd.out[s] >= 0 {
					c += g.links[nd.out[s]].acc
				}
			}
			if nd.nout == 0 {
				c = 0
			}
			if nd.nin < 2 {
				c = math.Min(1, c)
				if l := nd.in[0]; l >= 0 {
					worst = track(worst, &g.links[l].acc, c)
				}
				continue
			}
			c = math.Min(2, c)
			var f0, f1 float64
			if nd.in[0] >= 0 {
				f0 = g.links[nd.in[0]].flow
			}
			if nd.in[1] >= 0 {
				f1 = g.links[nd.in[1]].flow
			}
			a0, a1 := shareIn(c, f0, f1, nd.inP)
			if nd.in[0] >= 0 {
				worst = track(worst, &g.links[nd.in[0]].acc, a0)
			}
			if nd.in[1] >= 0 {
				worst = track(worst, &g.links[nd.in[1]].acc, a1)
			}
		}
		// Forwards: what each node sends.
		for i, l := range g.src {
			worst = track(worst, &g.links[l].flow, math.Min(in[i], g.links[l].acc))
		}
		for i := range g.nodes {
			nd := &g.nodes[i]
			if nd.nout == 0 {
				continue
			}
			var a float64
			for s := int8(0); s < nd.nin; s++ {
				if nd.in[s] >= 0 {
					a += g.links[nd.in[s]].flow
				}
			}
			if nd.nout < 2 {
				if l := nd.out[0]; l >= 0 {
					worst = track(worst, &g.links[l].flow, math.Min(a, g.links[l].acc))
				}
				continue
			}
			c0, c1 := g.links[nd.out[0]].acc, g.links[nd.out[1]].acc
			o0, o1 := splitOut(a, c0, c1, nd.outP)
			worst = track(worst, &g.links[nd.out[0]].flow, o0)
			worst = track(worst, &g.links[nd.out[1]].flow, o1)
		}
		settled = worst < flowEps
	}
	if !settled {
		return nil, nil, false
	}
	delivered = make([]float64, len(g.sink))
	for i, l := range g.sink {
		delivered[i] = g.links[l].flow
	}
	taken = make([]float64, len(g.src))
	for i, l := range g.src {
		taken[i] = g.links[l].flow
	}
	return delivered, taken, true
}

// track writes v into *p and returns the larger of worst and the distance it
// moved, which is the whole of the convergence test.
func track(worst float64, p *float64, v float64) float64 {
	d := math.Abs(v - *p)
	*p = v
	if d > worst {
		return d
	}
	return worst
}
