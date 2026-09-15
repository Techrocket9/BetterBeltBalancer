package plan

import (
	"math"
	"testing"
)

// The priority network, proved the way the plain one is: by simulating it.
//
// What a priority output promises is one sentence -- the ports a player ticked
// are fed first and share equally, the rest share what is left, equally -- and
// the whole of the work is making it EXACT at every load rather than close at
// most of them. So the assertion everywhere below is an equality against
//
//	priority: min(S/q, 1)          normal: min(max(S-q, 0)/(M-q), 1)
//
// where S is what the network actually took in, which is not what it was
// offered once it saturates.
//
// Simulate is a model of a splitter and not a splitter; the in-game suite is
// what says the engine agrees. flowmodel_test.go's header is the long form.

// prioEdges is edges() with the first q output edges ticked.
func prioEdges(n, m, q int) []Edge {
	out := make([]Edge, 0, n+m)
	for i := 0; i < n; i++ {
		out = append(out, Edge{TileX: 0, TileY: int32(i), Dir: East})
	}
	for i := 0; i < m; i++ {
		out = append(out, Edge{TileX: 7, TileY: int32(i), Dir: East, Out: true, Prio: i < q})
	}
	return out
}

// wantTiers is the promise, in the player's own terms.
func wantTiers(s float64, m, q int) (prio, normal float64) {
	prio = math.Min(s/float64(q), 1)
	if m > q {
		normal = math.Min(math.Max(s-float64(q), 0)/float64(m-q), 1)
	}
	return prio, normal
}

// checkTiers runs one load and returns the total the network took.
func checkTiers(t *testing.T, ops []Op, n, m, q int, in, out []float64) float64 {
	t.Helper()
	del, tak, ok := Simulate(ops, in, out)
	if !ok {
		t.Fatalf("%d->%d q=%d in=%v: the flow model did not converge", n, m, q, in)
	}
	var s float64
	for _, v := range tak {
		s += v
	}
	prio, normal := wantTiers(s, m, q)
	for i, got := range del {
		want := normal
		which := "normal"
		if i < q {
			want, which = prio, "priority"
		}
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("%d->%d q=%d in=%v: %s output %d carries %g and should carry %g "+
				"(the network took %g in total, and delivered %v)",
				n, m, q, in, which, i, got, want, s, del)
		}
	}
	return s
}

// TestEveryTierIsExactAtEveryLoad is THE test. Every shape the band the pure
// model and the M2 rigs both work over can have, every legal count of priority
// ports in it, and seven loads from a trickle to saturation.
func TestEveryTierIsExactAtEveryLoad(t *testing.T) {
	loads := []float64{0.05, 0.125, 0.25, 0.5, 0.75, 0.9, 1.0}
	for n := 1; n <= 8; n++ {
		for m := 2; m <= 8; m++ {
			for q := 1; q < m; q++ {
				es := prioEdges(n, m, q)
				pt, fits := ShapeEdges(es)
				if !fits {
					t.Fatalf("%d->%d q=%d (P=%d) is refused and every shape this "+
						"small fits", n, m, q, pt.P)
				}
				ops, _, ok := Build(nil, es, 100, 200)
				if !ok {
					t.Fatalf("%d->%d q=%d refused by Build", n, m, q)
				}
				for _, load := range loads {
					checkTiers(t, ops, n, m, q, evenInputs(n, load), freeOutputs(m))
				}
			}
		}
	}
}

// bigShapes are the sizes past P=8, where the construction's two bands stop
// being obviously roomy: the two powers of two it still fits in, the extremes
// of q at each, and both sides of N against M.
var bigShapes = [][3]int{
	{16, 16, 1}, {16, 16, 2}, {16, 16, 5}, {16, 16, 15},
	{9, 16, 3}, {16, 9, 3}, {12, 20, 7}, {20, 12, 7},
	{32, 32, 1}, {32, 32, 2}, {32, 32, 16}, {32, 32, 31},
	{17, 32, 4}, {32, 17, 4},
}

func TestTheBigShapesAreExactToo(t *testing.T) {
	for _, c := range bigShapes {
		n, m, q := c[0], c[1], c[2]
		es := prioEdges(n, m, q)
		if _, fits := ShapeEdges(es); !fits {
			t.Fatalf("%d->%d q=%d refused", n, m, q)
		}
		ops, _, _ := Build(nil, es, 0, 0)
		for _, load := range []float64{0.1, 0.4, 0.7, 1.0} {
			checkTiers(t, ops, n, m, q, evenInputs(n, load), freeOutputs(m))
		}
	}
}

// TestAnUnevenFeedIsStillExact. Every load above is the same rate on every
// input, which is the one case a wiring that mixed up two input ports would
// survive. These are not.
func TestAnUnevenFeedIsStillExact(t *testing.T) {
	feeds := [][]float64{
		{1, 0, 0, 0}, {0, 0, 0, 1}, {1, 1, 0, 0},
		{0.3, 0.9, 0.1, 1.0}, {1, 0.5, 0.25, 0.125},
	}
	for _, feed := range feeds {
		for _, c := range [][3]int{{4, 4, 1}, {4, 4, 2}, {4, 4, 3}, {4, 6, 2}, {4, 3, 1}, {4, 8, 3}, {4, 5, 4}} {
			n, m, q := c[0], c[1], c[2]
			ops, _, _ := Build(nil, prioEdges(n, m, q), 100, 200)
			checkTiers(t, ops, n, m, q, append([]float64(nil), feed[:n]...), freeOutputs(m))
		}
	}
}

// TestSaturatedInputsAreDrawnEqually is the other half of what a balancer
// promises and the one the construction got wrong first: a network that cannot
// take everything on offer must take the same from every input.
//
// It is the reason the tap column keeps the plain network's loopback. Without
// it a saturated 7->5 drew 0.625 from four of its inputs, 0.75 from two and a
// whole belt from the seventh, because the ranks past the last output port
// dead-ended and a butterfly whose outputs are blocked unevenly is not
// input-fair. The plain network never has that shape: Shape's Loop fills every
// row of the head that a real input does not.
func TestSaturatedInputsAreDrawnEqually(t *testing.T) {
	for n := 1; n <= 8; n++ {
		for m := 2; m <= 8; m++ {
			for q := 1; q < m; q++ {
				ops, _, _ := Build(nil, prioEdges(n, m, q), 0, 0)
				for _, load := range []float64{0.75, 0.9, 1.0} {
					_, tak, ok := Simulate(ops, evenInputs(n, load), freeOutputs(m))
					if !ok {
						t.Fatalf("%d->%d q=%d load %g: no convergence", n, m, q, load)
					}
					for i := range tak {
						if math.Abs(tak[i]-tak[0]) > 1e-9 {
							t.Fatalf("%d->%d q=%d load %g: the network drew %v from "+
								"its inputs, which are all offering the same",
								n, m, q, load, tak)
						}
					}
				}
			}
		}
	}
}

// TestTheConcentratorSorts is the one claim the whole construction rests on and
// the one that is not obvious: the second butterfly, with every splitter's
// output priority on the smaller y, turns P equal rows into a staircase, and
// the row holding rank r carries clamp(S - r, 0, 1).
//
// It is measured on the real thing rather than on a copy of the schedule: the
// network is built with EVERY output a separate port, so that port r is rank r
// and what the model reports is what the concentrator put there.
func TestTheConcentratorSorts(t *testing.T) {
	for _, p := range []int{2, 4, 8, 16, 32} {
		// n = m = P puts one output on every rank and makes Loop zero, so
		// nothing recirculates and the staircase is the whole answer.
		ops, _, ok := Build(nil, prioEdges(p, p, 1), 0, 0)
		if !ok {
			t.Fatalf("P=%d refused", p)
		}
		for _, s := range []float64{0.5, 1, 1.5, float64(p) / 2, float64(p) - 0.25, float64(p)} {
			del, tak, conv := Simulate(ops, evenInputs(p, s/float64(p)), freeOutputs(p))
			if !conv {
				t.Fatalf("P=%d S=%g: no convergence", p, s)
			}
			var got float64
			for _, v := range tak {
				got += v
			}
			if math.Abs(got-s) > 1e-9 {
				t.Fatalf("P=%d: offered %g and the network took %g", p, s, got)
			}
			// Port 0 is rank 0 and it is the only port the concentrator feeds
			// directly; the rest go through the residual balancer, which
			// equalises. So what the staircase is checkable through here is its
			// HEAD and its TOTAL, which is what the construction uses it for.
			if want := math.Min(s, 1); math.Abs(del[0]-want) > 1e-9 {
				t.Fatalf("P=%d S=%g: rank 0 carries %g, a concentrator's first row "+
					"carries min(S, 1) = %g", p, s, del[0], want)
			}
			var rest float64
			for _, v := range del[1:] {
				rest += v
			}
			if want := s - math.Min(s, 1); math.Abs(rest-want) > 1e-9 {
				t.Fatalf("P=%d S=%g: the ranks past the first carry %g between them "+
					"and should carry %g", p, s, rest, want)
			}
		}
	}
}

// TestRanksAreAPermutation. sorterRanks is read as an index into the tap
// column, so two rows sharing a rank would silently drop a port and leave
// another one unfed.
func TestRanksAreAPermutation(t *testing.T) {
	for _, p := range []int{1, 2, 4, 8, 16, 32, 64} {
		sorterRanks(p)
		seen := make([]bool, p)
		for r := 0; r < p; r++ {
			v := int(rankBuf[r])
			if v < 0 || v >= p {
				t.Fatalf("P=%d: row %d has rank %d", p, r, v)
			}
			if seen[v] {
				t.Fatalf("P=%d: two rows carry rank %d", p, v)
			}
			seen[v] = true
		}
		if rankBuf[0] != 0 {
			t.Fatalf("P=%d: rank 0 is on row %d, and `order` puts the line that "+
				"wins every stage on row 0", p, rankBuf[0])
		}
	}
}

// ---------------------------------------------------------------------------
// The slot
// ---------------------------------------------------------------------------

// fittingPrioShapes is every (n, m, q) worth building: the whole n, m <= 8 band
// and the big shapes, plus the sizes that must be refused.
func fittingPrioShapes() [][3]int {
	out := make([][3]int, 0, 400)
	for n := 1; n <= 8; n++ {
		for m := 2; m <= 8; m++ {
			for q := 1; q < m; q++ {
				out = append(out, [3]int{n, m, q})
			}
		}
	}
	return append(out, bigShapes...)
}

// TestEveryPriorityShapeStaysInsideItsSlot. A slot's box is what a teardown
// sweeps, so an entity outside it survives its own rebuild as a ghost nothing
// owns and is destroyed by its neighbour's teardown. prioExtent is what
// ShapeEdges refuses on, so it has to be the truth about where the ops land and
// not merely an upper bound on it.
func TestEveryPriorityShapeStaysInsideItsSlot(t *testing.T) {
	const ox, oy = 100, 200
	for _, c := range fittingPrioShapes() {
		n, m, q := c[0], c[1], c[2]
		es := prioEdges(n, m, q)
		pt, fits := ShapeEdges(es)
		if !fits {
			t.Fatalf("%d->%d q=%d refused", n, m, q)
		}
		ops, _, _ := Build(nil, es, ox, oy)
		w, h := prioExtent(pt)
		if w > SlotWidth || h > SlotHeight {
			t.Fatalf("%d->%d q=%d fits and wants %dx%d of a %dx%d slot",
				n, m, q, w, h, SlotWidth, SlotHeight)
		}
		maxX, maxY := -1, -1
		for _, o := range ops {
			if o.Visible {
				continue
			}
			for _, tl := range tilesOf(o) {
				x, y := int(tl[0])-ox, int(tl[1])-oy
				if x < 0 || y < 0 {
					t.Fatalf("%d->%d q=%d: an op at (%d,%d) is before the slot origin",
						n, m, q, x, y)
				}
				if x >= w || y >= h {
					t.Fatalf("%d->%d q=%d: an op at (%d,%d) is outside the %dx%d "+
						"extent ShapeEdges measured", n, m, q, x, y, w, h)
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
		// ...and the extent is the truth rather than a generous bound: a loose
		// one would refuse shapes that fit.
		if maxX+1 != w || maxY+1 != h {
			t.Fatalf("%d->%d q=%d: the ops reach (%d,%d) and prioExtent says %dx%d",
				n, m, q, maxX, maxY, w, h)
		}
	}
}

// TestNoTwoPriorityEntitiesShareATile. create_entity of a colliding
// belt-connectable returns nil SILENTLY, and a priority network is four blocks
// in two bands rather than one, so the packing is a new way to get it wrong.
func TestNoTwoPriorityEntitiesShareATile(t *testing.T) {
	for _, c := range fittingPrioShapes() {
		n, m, q := c[0], c[1], c[2]
		ops, _, ok := Build(nil, prioEdges(n, m, q), 100, 200)
		if !ok {
			t.Fatalf("%d->%d q=%d refused", n, m, q)
		}
		seen := map[[2]int32]int{}
		for i, o := range ops {
			if o.Visible {
				continue
			}
			for _, tl := range tilesOf(o) {
				if j, dup := seen[tl]; dup {
					t.Fatalf("%d->%d q=%d: ops %d and %d both want tile %v",
						n, m, q, j, i, tl)
				}
				seen[tl] = i
			}
		}
	}
}

// TestEveryPriorityInputEndIsPaired. The executor connects by walking the input
// ends once, so an unpaired one is a dead interface and an output end carrying
// a Pair would be connected twice. The priority path has four more kinds of
// pair in it than the plain one -- tap to tier, tier loopback, tier to visible,
// and the concentrator's spare ranks back to the head -- and this is what says
// none of them was left half wired.
func TestEveryPriorityInputEndIsPaired(t *testing.T) {
	for _, c := range fittingPrioShapes() {
		n, m, q := c[0], c[1], c[2]
		ops, _, _ := Build(nil, prioEdges(n, m, q), 0, 0)
		pairedTo := map[int32]int{}
		for i, o := range ops {
			switch o.Link {
			case LinkInput:
				if o.Pair < 0 {
					t.Fatalf("%d->%d q=%d: op %d is an unpaired input end at (%g,%g)",
						n, m, q, i, o.X, o.Y)
				}
				if ops[o.Pair].Link != LinkOutput {
					t.Fatalf("%d->%d q=%d: op %d pairs a non-output end", n, m, q, i)
				}
				pairedTo[o.Pair]++
			case LinkOutput:
				if o.Pair >= 0 {
					t.Fatalf("%d->%d q=%d: op %d is an output end carrying a Pair",
						n, m, q, i)
				}
			}
		}
		for idx, k := range pairedTo {
			if k != 1 {
				t.Fatalf("%d->%d q=%d: op %d is the partner of %d input ends",
					n, m, q, idx, k)
			}
		}
	}
}

// TestOnlySplittersCarryAPriority, and only the concentrator's. A priority on
// a belt or a linked belt is a field the executor would hand to a member that
// does not exist; one on a balancing splitter is the defect this construction
// exists to avoid, and it would show up as a rate rather than as an error.
func TestOnlySplittersCarryAPriority(t *testing.T) {
	for _, c := range fittingPrioShapes() {
		n, m, q := c[0], c[1], c[2]
		ops, _, _ := Build(nil, prioEdges(n, m, q), 0, 0)
		prio, plain := 0, 0
		for i, o := range ops {
			if o.InPrio != PrioNone {
				t.Fatalf("%d->%d q=%d: op %d carries an input priority and nothing "+
					"builds one", n, m, q, i)
			}
			if o.OutPrio == PrioNone {
				continue
			}
			if o.Proto != ProtoSplitter {
				t.Fatalf("%d->%d q=%d: op %d is a %d carrying an output priority",
					n, m, q, i, o.Proto)
			}
			if o.OutPrio != PrioLeft {
				t.Fatalf("%d->%d q=%d: op %d prioritises %d, and the concentration "+
					"tree wins on the smaller y", n, m, q, i, o.OutPrio)
			}
			prio++
		}
		for _, o := range ops {
			if o.Proto == ProtoSplitter && o.OutPrio == PrioNone {
				plain++
			}
		}
		// One concentrator, so exactly one butterfly's worth of splitters
		// carries the priority.
		pt, _ := ShapeEdges(prioEdges(n, m, q))
		if want := stages(pt.P) * pt.P / 2; prio != want {
			t.Fatalf("%d->%d q=%d: %d splitters carry a priority and the "+
				"concentrator over P=%d is %d of them", n, m, q, prio, pt.P, want)
		}
		if plain == 0 {
			t.Fatalf("%d->%d q=%d: no plain splitter anywhere, so nothing balances",
				n, m, q)
		}
	}
}

// ---------------------------------------------------------------------------
// The refusal
// ---------------------------------------------------------------------------

// TestPriorityIsRefusedExactlyPastP32. The slot is what binds a priority
// network rather than the port cap: it is two butterflies wide where the plain
// one is a single, and it carries its tiers in a second band of rows. P=64 is
// 35 columns of a 32-column slot, and there is no packing of two 64-row blocks
// that is not also 128 rows.
func TestPriorityIsRefusedExactlyPastP32(t *testing.T) {
	for _, c := range [][3]int{
		{64, 64, 1}, {33, 64, 1}, {64, 33, 1}, {64, 64, 32}, {40, 40, 3}, {64, 2, 1},
	} {
		if _, fits := ShapeEdges(prioEdges(c[0], c[1], c[2])); fits {
			t.Errorf("%d->%d q=%d has P=64 and a priority network that size does "+
				"not fit its slot", c[0], c[1], c[2])
		}
	}
	// ...and everything at P=32 and under does, at every q.
	for _, p := range []int{2, 4, 8, 16, 32} {
		for _, q := range []int{1, 2, p / 2, p - 1} {
			if q < 1 || q >= p {
				continue
			}
			if _, fits := ShapeEdges(prioEdges(p, p, q)); !fits {
				t.Errorf("%d->%d q=%d is refused and P=%d fits", p, p, q, p)
			}
		}
	}
	// The port cap still comes first, and it still reads as the port cap: a
	// cluster past MaxPorts is refused whether or not anything is ticked.
	if _, fits := ShapeEdges(prioEdges(65, 1, 0)); fits {
		t.Error("65 inputs should be refused")
	}
}

// TestEveryOutputTickedIsThePlainBalancer. Two tiers where the second is empty
// is one tier, and one tier is the network this mod already builds. A player
// who ticks every output of a 64-port balancer gets it rather than a refusal,
// and the ops are the plain network's to the byte.
func TestEveryOutputTickedIsThePlainBalancer(t *testing.T) {
	for _, nm := range [][2]int{{4, 4}, {3, 5}, {8, 8}, {64, 64}, {33, 63}} {
		n, m := nm[0], nm[1]
		all := prioEdges(n, m, m)
		pt, fits := ShapeEdges(all)
		if !fits {
			t.Fatalf("%d->%d with every output ticked is refused", n, m)
		}
		if pt.QOut != 0 {
			t.Fatalf("%d->%d with every output ticked reports QOut=%d", n, m, pt.QOut)
		}
		got, _, _ := Build(nil, all, 100, 200)
		want, _, _ := Build(nil, edges(n, m), 100, 200)
		if opsDigest(got) != opsDigest(want) {
			t.Fatalf("%d->%d with every output ticked is not the plain network", n, m)
		}
	}
}

// TestInputPriorityIsRefused. buildPrio has no de-concentrator in it, so an
// input edge marked Prio would compile to a network that ignores the mark --
// which is the one outcome this repository's own rule about approximations
// forbids. agents/priority.md, "What input priority does".
func TestInputPriorityIsRefused(t *testing.T) {
	es := prioEdges(4, 4, 1)
	es[0].Prio = true
	pt, fits := ShapeEdges(es)
	if fits {
		t.Fatal("an input marked Prio should be refused while nothing builds one")
	}
	if pt.QIn != 1 {
		t.Fatalf("QIn is %d and one input is marked", pt.QIn)
	}
	if _, _, ok := Build(nil, es, 0, 0); ok {
		t.Fatal("Build should refuse it too, and for the same reason")
	}
	// Every input ticked is no input priority at all, by the same collapse the
	// outputs get.
	every := edges(4, 4)
	for i := range every {
		if !every[i].Out {
			every[i].Prio = true
		}
	}
	if pt, fits := ShapeEdges(every); !fits || pt.QIn != 0 {
		t.Fatalf("every input ticked: fits=%v QIn=%d", fits, pt.QIn)
	}
}

// ---------------------------------------------------------------------------
// The shipping constraints the plain path already carries
// ---------------------------------------------------------------------------

func TestPriorityBuildIsDeterministic(t *testing.T) {
	es := prioEdges(5, 7, 2)
	for i := 0; i < 20; i++ {
		a, _, _ := Build(nil, es, 10, 20)
		b, _, _ := Build(nil, es, 10, 20)
		if len(a) != len(b) {
			t.Fatal("length differs between runs")
		}
		for j := range a {
			if a[j] != b[j] {
				t.Fatalf("op %d differs between runs: %+v vs %+v", j, a[j], b[j])
			}
		}
	}
}

func TestPriorityBuildDoesNotAllocateOnAReusedBuffer(t *testing.T) {
	es := prioEdges(8, 8, 3)
	buf, _, _ := Build(nil, es, 0, 0)
	n := testing.AllocsPerRun(100, func() {
		buf, _, _ = Build(buf, es, 0, 0)
	})
	if n != 0 {
		t.Fatalf("Build allocates %g times per call on a warm buffer", n)
	}
}

// TestPriorityEntityCount records what the construction costs, because a
// recompile is linear in entities and CLAUDE.md's hitch figures are measured
// per entity. agents/priority.md carries the same table with the plain
// networks beside it.
//
// It is not monotone in q and that is the tier balancers rather than an error:
// a 4x4 costs MORE with one priority output than with two, because one leaves
// three normal ports, and a square butterfly over three ports is four rows with
// a loopback in it where two and two are a pair of single-splitter blocks.
func TestPriorityEntityCount(t *testing.T) {
	for _, c := range []struct {
		n, m, q, want int
	}{
		{2, 2, 1, 18},
		{4, 4, 1, 71},
		{4, 4, 2, 58},
		{4, 4, 3, 71},
		{8, 8, 1, 199},
		{8, 8, 2, 203},
		{8, 8, 4, 176},
		{16, 16, 1, 515},
		{32, 32, 1, 1267},
	} {
		ops, _, ok := Build(nil, prioEdges(c.n, c.m, c.q), 0, 0)
		if !ok {
			t.Fatalf("%d->%d q=%d refused", c.n, c.m, c.q)
		}
		if len(ops) != c.want {
			t.Errorf("%d->%d q=%d is %d entities, the recorded figure is %d "+
				"(update agents/priority.md in the same commit if this is intended)",
				c.n, c.m, c.q, len(ops), c.want)
		}
	}
}

// TestABlockedOutputIsRebalancedByTheTiers. A blocked output is the ordinary way
// a balancer meets a full chest, and the priority network absorbs one where the
// plain network does not: its tiers are SQUARE butterflies with every spare port
// looped back, so a blocked port's flow recirculates instead of doubling up on
// whichever port shares its last splitter.
//
// The plain network's behaviour is recorded beside this one in
// TestABlockedOutputIsNotRebalancedUnderPartialLoad. Both are model results: no
// rig drives a blocked output at partial load.
func TestABlockedOutputIsRebalancedByTheTiers(t *testing.T) {
	for _, c := range []struct {
		n, m, q, blocked int
		want             []float64
	}{
		// The priority port blocked: the three normal ports share the belt.
		{4, 4, 1, 0, []float64{0, 1.0 / 3, 1.0 / 3, 1.0 / 3}},
		// A normal port blocked: the priority port fills first, as ever.
		{4, 4, 1, 2, []float64{1, 0, 0, 0}},
		// Two tiers, one priority port blocked: its partner takes the tier's
		// whole share, and the residual is still the residual.
		{8, 8, 2, 0, []float64{0, 1, 1.0 / 6, 1.0 / 6, 1.0 / 6, 1.0 / 6, 1.0 / 6, 1.0 / 6}},
	} {
		ops, _, _ := Build(nil, prioEdges(c.n, c.m, c.q), 0, 0)
		out := freeOutputs(c.m)
		out[c.blocked] = 0
		del, _, ok := Simulate(ops, evenInputs(c.n, 0.25), out)
		if !ok {
			t.Fatalf("%d->%d q=%d: no convergence", c.n, c.m, c.q)
		}
		for j, v := range del {
			if math.Abs(v-c.want[j]) > 1e-9 {
				t.Fatalf("%d->%d q=%d with output %d blocked delivers %v, want %v",
					c.n, c.m, c.q, c.blocked, del, c.want)
			}
		}
	}
}

// TestShapeEdgesIsAnswerableFromAKeypress. The guest asks ShapeEdges at TOGGLE
// time, on a speculatively flipped edge list, outside any flush, so that a
// toggle the compiler cannot honour is refused before the flag moves. Three
// things have to hold for that to be sound, and none of them is obvious from
// reading the call site:
//
//   - it allocates nothing. The guest is built -gc=leaking, so an allocation
//     per keypress is an allocation per keypress forever;
//   - it touches no package buffer and no other state, so asking it between
//     flushes cannot disturb a compile in flight;
//   - the answer is the one Build gives a tick later, for the same edge list in
//     any order. A toggle that passed here and was refused at the flush would
//     put the registry and the world one flag apart.
func TestShapeEdgesIsAnswerableFromAKeypress(t *testing.T) {
	cases := fittingPrioShapes()
	cases = append(cases, [3]int{64, 64, 1}, [3]int{33, 64, 2}, [3]int{65, 1, 0})
	for _, c := range cases {
		es := prioEdges(c[0], c[1], c[2])
		pt, fits := ShapeEdges(es)

		// Build's own refusal is ShapeEdges' -- for a shape with both sides.
		_, bpt, ok := Build(nil, es, 0, 0)
		if ok != fits {
			t.Fatalf("%v: ShapeEdges says fits=%v and Build says ok=%v", c, fits, ok)
		}
		if bpt != pt {
			t.Fatalf("%v: ShapeEdges says %+v and Build says %+v", c, pt, bpt)
		}

		// ...and it reads the counts, not the order. The guest flips a flag on
		// one edge of a list classifyEdges just produced; nothing says that
		// list arrives sorted.
		rev := make([]Edge, len(es))
		for i := range es {
			rev[i] = es[len(es)-1-i]
		}
		rpt, rfits := ShapeEdges(rev)
		if rfits != fits || rpt != pt {
			t.Fatalf("%v reversed: %+v/%v against %+v/%v", c, rpt, rfits, pt, fits)
		}

		if n := testing.AllocsPerRun(50, func() { ShapeEdges(es) }); n != 0 {
			t.Fatalf("%v: ShapeEdges allocates %g times per call", c, n)
		}
	}
	// A half-built cluster is answerable too, and cheaply: it fits at any size.
	for _, nm := range [][2]int{{0, 0}, {8, 0}, {0, 8}, {70, 0}} {
		if _, fits := ShapeEdges(edges(nm[0], nm[1])); !fits {
			t.Fatalf("%v: a cluster with one side is a legitimate half-built state", nm)
		}
	}
}

// TestTheRatesASuiteShouldAssert pins the two shapes an in-game suite is
// cheapest to build, at the three loads that separate the tiers: under-supply
// (the priority port cannot fill), mid (it is full and the rest share the
// remainder) and saturation. The load is in BELTS FED, so a rig feeds it by
// choosing how many input belts run and at what tier.
//
// These are the model's figures. A suite compares items delivered over a window
// against a bare belt in the same save, which is the ratio in the last column.
func TestTheRatesASuiteShouldAssert(t *testing.T) {
	for _, c := range []struct {
		n, m, q int
		fed     []float64
		want    []float64
	}{
		{4, 4, 1, []float64{0.5, 0, 0, 0}, []float64{0.5, 0, 0, 0}},
		{4, 4, 1, []float64{1, 1, 0, 0}, []float64{1, 1.0 / 3, 1.0 / 3, 1.0 / 3}},
		{4, 4, 1, []float64{1, 1, 1, 1}, []float64{1, 1, 1, 1}},
		{2, 2, 1, []float64{0.5, 0}, []float64{0.5, 0}},
		{2, 2, 1, []float64{1, 0.5}, []float64{1, 0.5}},
		{2, 2, 1, []float64{1, 1}, []float64{1, 1}},
	} {
		ops, _, _ := Build(nil, prioEdges(c.n, c.m, c.q), 0, 0)
		del, tak, ok := Simulate(ops, c.fed, freeOutputs(c.m))
		if !ok {
			t.Fatalf("%d->%d q=%d fed %v: no convergence", c.n, c.m, c.q, c.fed)
		}
		var s float64
		for _, v := range tak {
			s += v
		}
		for j, v := range del {
			if math.Abs(v-c.want[j]) > 1e-9 {
				t.Fatalf("%d->%d q=%d fed %v (S=%g): delivers %v, want %v",
					c.n, c.m, c.q, c.fed, s, del, c.want)
			}
		}
	}
}

// TestPriorityCapacityIsRecorded pins agents/priority.md's capacity table, both
// columns and the residual bound between them, and pins the shape-only form
// against the built one on every row.
//
// Those are two different claims. `capacityOf` counts an op list, which is what
// the table was measured with; `Capacity` takes a Ports and builds the shape
// itself, which is what the guest asks on a keypress, and the only thing that
// says the two agree is comparing them on every row.
//
// The residual bound is the difference: a toggle is a recompile, a recompile
// hands the drained items to the network that succeeds it, and what that
// successor cannot hold spills beside the cluster.
func TestPriorityCapacityIsRecorded(t *testing.T) {
	for _, c := range []struct {
		n, m, q            int
		plain, prio, resid int
	}{
		{2, 2, 1, 112, 192, 80},
		{4, 4, 1, 320, 736, 416},
		{4, 4, 2, 320, 608, 288},
		{3, 5, 1, 720, 1456, 736},
		{5, 3, 1, 752, 1312, 560},
		{8, 8, 1, 832, 2016, 1184},
		{8, 8, 2, 832, 2064, 1232},
		{8, 8, 4, 832, 1792, 960},
		{16, 16, 1, 2048, 5152, 3104},
		{32, 32, 1, 4864, 12576, 7712},
	} {
		pl, ptPl, _ := Build(nil, edges(c.n, c.m), 0, 0)
		if got := capacityOf(pl); got != c.plain {
			t.Errorf("%d->%d plain holds %d item positions, recorded %d",
				c.n, c.m, got, c.plain)
		}
		pr, ptPr, _ := Build(nil, prioEdges(c.n, c.m, c.q), 0, 0)
		if got := capacityOf(pr); got != c.prio {
			t.Errorf("%d->%d q=%d holds %d item positions, recorded %d "+
				"(update agents/priority.md in the same commit)",
				c.n, c.m, c.q, got, c.prio)
		}
		if got := c.prio - c.plain; got != c.resid {
			t.Errorf("%d->%d q=%d leaves a residual bound of %d, recorded %d",
				c.n, c.m, c.q, got, c.resid)
		}
		if got := Capacity(ptPl); got != c.plain {
			t.Errorf("%d->%d plain: Capacity(Ports) says %d and the built ops "+
				"hold %d -- the guest asks the first and the teardown fills the "+
				"second", c.n, c.m, got, c.plain)
		}
		if got := Capacity(ptPr); got != c.prio {
			t.Errorf("%d->%d q=%d: Capacity(Ports) says %d and the built ops "+
				"hold %d", c.n, c.m, c.q, got, c.prio)
		}
	}
}

// TestAPriorityToggleCanShrinkTheNetworkInEitherDirection is why the guest's
// spill guard compares two capacities rather than asking which way the flag
// moved.
//
// Turning a port's priority ON is bigger than the plain network on every row of
// the table above, which is where the design's "toggling ON never spills" came
// from -- and it is a claim about plain against priority, not about q against
// q+1. A 4x4's second priority port is SMALLER than its first: one priority
// port leaves three normal ones and a square butterfly over three ports is four
// rows with a loopback in it, where two and two are a pair of single-splitter
// blocks. So a toggle ON shrinks the network there, and a guard that only
// watched the OFF direction would let that one spill.
func TestAPriorityToggleCanShrinkTheNetworkInEitherDirection(t *testing.T) {
	one, _ := ShapeEdges(prioEdges(4, 4, 1))
	two, _ := ShapeEdges(prioEdges(4, 4, 2))
	if Capacity(two) >= Capacity(one) {
		t.Fatalf("4->4 holds %d positions at q=1 and %d at q=2; the guard's "+
			"whole reason for comparing capacities is that the second is smaller",
			Capacity(one), Capacity(two))
	}
	plain, _ := ShapeEdges(edges(4, 4))
	if Capacity(two) <= Capacity(plain) {
		t.Fatalf("4->4 q=2 holds %d positions and the plain network holds %d; "+
			"a priority network smaller than the plain one would make the very "+
			"first toggle a spill", Capacity(two), Capacity(plain))
	}
}

// TestTheInputPriorityMirrorIsTheSameShape is not a test of anything shipped:
// nothing sets InPrio, and ShapeEdges refuses an input marked Prio. It is the
// probe that says WHY the refusal is a missing construction rather than a
// missing line, and it is here so that agents/priority.md's claim about the
// mirror is pinned rather than remembered.
//
// A butterfly whose every splitter has its INPUT priority on the smaller y
// drains its low rows first. With one output to drain into, it takes a whole
// belt from the first input and nothing from the rest -- which is exactly what
// a de-concentrator should do. With TWO, it takes a belt from each of two
// inputs rather than a belt and a half from the first: the mirror of the
// [1, 0, 1, 0] a priority butterfly produces on the output side, and the same
// thing it says -- a de-concentrator alone spreads over a SET of inputs, so it
// needs a balancing butterfly and a pair of input-side tiers around it.
func TestTheInputPriorityMirrorIsTheSameShape(t *testing.T) {
	for _, c := range []struct {
		n, m int
		want []float64
	}{
		{4, 1, []float64{1, 0, 0, 0}},
		{8, 1, []float64{1, 0, 0, 0, 0, 0, 0, 0}},
		{4, 2, []float64{1, 0, 1, 0}},
	} {
		ops, _, ok := Build(nil, edges(c.n, c.m), 0, 0)
		if !ok {
			t.Fatalf("%d->%d refused", c.n, c.m)
		}
		for i := range ops {
			if ops[i].Proto == ProtoSplitter {
				ops[i].InPrio = PrioLeft
			}
		}
		_, tak, conv := Simulate(ops, evenInputs(c.n, 1), freeOutputs(c.m))
		if !conv {
			t.Fatalf("%d->%d: no convergence", c.n, c.m)
		}
		for i, v := range tak {
			if math.Abs(v-c.want[i]) > 1e-9 {
				t.Fatalf("%d->%d with every splitter drawing from the smaller y "+
					"first: intake %v, want %v", c.n, c.m, tak, c.want)
			}
		}
	}
}
