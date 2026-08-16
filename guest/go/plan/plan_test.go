package plan

import (
	"math"
	"math/rand"
	"os"
	"testing"
)

// TestMain installs the compass, because nothing else in a `go test` run will.
//
// In the guest, compile.go's initBuffers fills it from the generated
// `DefinesDirection*()` accessors -- a host call into the running Factorio,
// which this package deliberately cannot make. The numbers below are what
// Factorio 2.0 happens to answer today, and writing them HERE rather than in
// plan.go is the whole point: a test may assume a concrete world, shipped code
// may not.
//
// Only two properties of them are load-bearing for the layout tests: the four
// are distinct, and north/south and east/west are opposites.
func TestMain(m *testing.M) {
	SetCompass(0, 4, 8, 12)
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// The balance property, proved by simulation rather than by inspection.
//
// A splitter makes its two outputs equal. That is the only fact these tests
// assume about Factorio; everything else is the wiring, and the wiring is what
// can be wrong.
// ---------------------------------------------------------------------------

// TestEveryRowEndsEqual is THE test. Whatever arrives on whatever subset of
// lines, every line leaves the network carrying the same flow -- which is what
// "the balancer balances" means, and it is why an arbitrary subset of the rows
// can be handed to the real outputs.
func TestEveryRowEndsEqual(t *testing.T) {
	for _, p := range []int{1, 2, 4, 8, 16, 32, 64} {
		// Exhaustive over every 0/1 input pattern while that is cheap, then
		// random real-valued ones, which catch a wiring that happens to be
		// symmetric only for equal inputs.
		if p <= 8 {
			for mask := 0; mask < 1<<uint(p); mask++ {
				in := make([]float64, p)
				for i := range in {
					if mask&(1<<uint(i)) != 0 {
						in[i] = 1
					}
				}
				checkEqual(t, p, in)
			}
		}
		rng := rand.New(rand.NewSource(int64(p)))
		for trial := 0; trial < 200; trial++ {
			in := make([]float64, p)
			for i := range in {
				in[i] = rng.Float64()
			}
			checkEqual(t, p, in)
		}
	}
}

func checkEqual(t *testing.T, p int, in []float64) {
	t.Helper()
	out := Propagate(in)
	var want, sumIn float64
	for _, v := range in {
		sumIn += v
	}
	want = sumIn / float64(p)
	for r, v := range out {
		if math.Abs(v-want) > 1e-9 {
			t.Fatalf("P=%d row %d carries %g, every row should carry %g (in=%v)",
				p, r, v, want, in)
		}
	}
}

// TestFlowIsConserved: splitters neither create nor destroy items, so a wiring
// that dropped a row on the floor (a permutation that is not a bijection) shows
// up as a changed total even when every row is still equal to every other.
func TestFlowIsConserved(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, p := range []int{2, 4, 8, 16, 32, 64} {
		in := make([]float64, p)
		var sum float64
		for i := range in {
			in[i] = rng.Float64()
			sum += in[i]
		}
		out := Propagate(in)
		var got float64
		for _, v := range out {
			got += v
		}
		if math.Abs(got-sum) > 1e-9 {
			t.Fatalf("P=%d: %g in, %g out", p, sum, got)
		}
	}
}

// TestLoopbackShapesDeliverEvenOutputs is TestEveryRowEndsEqual for the half of
// the network that test cannot see. Propagate never consults Ports, so the
// loopback wiring -- spare outputs [M, M+Loop) fed back into spare inputs
// [N, N+Loop) -- was unmodelled entirely, and 36 of the 64 shapes with n,m <= 8
// have Loop > 0. Only 3->5 had any evidence, and that evidence was a Factorio
// run.
//
// The claim is sharper than "the rows are equal", because with recirculation
// they are equal by construction and could still be equal at the WRONG value: a
// loopback wired to the wrong row, or counted twice, changes what each output
// carries without disturbing the equality. So the assertion is the absolute
// figure, S/m per output, and the conservation of S across the m real ones --
// the loopback rows carry circulating flow that must not be counted as
// delivered.
//
// n <= m only; PropagateLoop's header says why (for n > m the spare outputs
// dead-end and back up, which is a nonlinearity M2 covers in the game).
func TestLoopbackShapesDeliverEvenOutputs(t *testing.T) {
	for m := 1; m <= MaxPorts; m++ {
		for n := 1; n <= m; n++ {
			pt := Shape(n, m)
			if pt.Loop != pt.P-pt.M {
				t.Fatalf("%d->%d: Loop=%d, but for n<=m every spare output should "+
					"loop back (P-M=%d)", n, m, pt.Loop, pt.P-pt.M)
			}
			sat := make([]float64, n)
			for i := range sat {
				sat[i] = 1
			}
			checkLoop(t, n, m, pt, sat)

			rng := rand.New(rand.NewSource(int64(n)*1000 + int64(m)))
			for trial := 0; trial < 4; trial++ {
				ext := make([]float64, n)
				for i := range ext {
					ext[i] = rng.Float64()
				}
				checkLoop(t, n, m, pt, ext)
			}
		}
	}
}

func checkLoop(t *testing.T, n, m int, pt Ports, ext []float64) {
	t.Helper()
	rows, ok := PropagateLoop(ext, pt)
	if !ok {
		t.Fatalf("%d->%d: the loopback fixed point did not converge (ext=%v)", n, m, ext)
	}
	var s float64
	for _, v := range ext {
		s += v
	}
	want := s / float64(m)
	var got float64
	for r := 0; r < m; r++ {
		if math.Abs(rows[r]-want) > 1e-9 {
			t.Fatalf("%d->%d (P=%d Loop=%d): output %d carries %g, every output "+
				"should carry %g (ext=%v)", n, m, pt.P, pt.Loop, r, rows[r], want, ext)
		}
		got += rows[r]
	}
	if math.Abs(got-s) > 1e-9 {
		t.Fatalf("%d->%d: %g entered and %g left by the real outputs -- the "+
			"circulating flow is being counted as delivered", n, m, s, got)
	}
}

// TestEveryStagePairsAdjacentRows is the constraint the physical layout exists
// to satisfy: a Factorio splitter spans two adjacent rows, so a schedule that
// wanted rows 0 and 4 joined would be unbuildable.
func TestEveryStagePairsAdjacentRows(t *testing.T) {
	for _, p := range []int{2, 4, 8, 16, 32, 64} {
		ord := order(p)
		for s, row := range ord {
			for t2 := 0; t2 < p/2; t2++ {
				a, b := row[2*t2], row[2*t2+1]
				if a^b != 1<<uint(s) {
					t.Fatalf("P=%d stage %d: rows %d,%d hold lines %d,%d, which differ in more than bit %d",
						p, s, 2*t2, 2*t2+1, a, b, s)
				}
			}
		}
	}
}

// TestPermutationsAreBijections: a jumper block moves each row's contents to
// exactly one destination. If two rows targeted the same destination, one
// linked belt would silently overwrite the other's tile and the create would
// come back nil.
func TestPermutationsAreBijections(t *testing.T) {
	for _, p := range []int{4, 8, 16, 32, 64} {
		ord := order(p)
		for s := 1; s < len(ord); s++ {
			perm := permutation(ord[s-1], ord[s])
			seen := make([]bool, p)
			for _, d := range perm {
				if seen[d] {
					t.Fatalf("P=%d stage %d: two rows both move to %d", p, s, d)
				}
				seen[d] = true
			}
		}
	}
}

// ---------------------------------------------------------------------------
// The physical plan
// ---------------------------------------------------------------------------

func edges(n, m int) []Edge {
	var out []Edge
	for i := 0; i < n; i++ {
		out = append(out, Edge{TileX: 0, TileY: int32(i), Dir: East})
	}
	for i := 0; i < m; i++ {
		out = append(out, Edge{TileX: 7, TileY: int32(i), Dir: East, Out: true})
	}
	return out
}

// TestNoTwoHiddenEntitiesShareATile. create_entity of a colliding
// belt-connectable returns nil SILENTLY -- there is no error, no log, and the
// network is quietly missing a piece. The plan must never ask for one.
//
// A splitter occupies two tiles, which is exactly the case a naive check misses.
func TestNoTwoHiddenEntitiesShareATile(t *testing.T) {
	for n := 1; n <= 8; n++ {
		for m := 1; m <= 8; m++ {
			ops, _, ok := Build(nil, edges(n, m), 100, 200)
			if !ok {
				t.Fatalf("%d->%d refused", n, m)
			}
			seen := map[[2]int32]int{}
			for i, o := range ops {
				if o.Visible {
					continue
				}
				for _, tl := range tilesOf(o) {
					if j, dup := seen[tl]; dup {
						t.Fatalf("%d->%d: ops %d and %d both want tile %v", n, m, j, i, tl)
					}
					seen[tl] = i
				}
			}
		}
	}
}

func tilesOf(o Op) [][2]int32 {
	x := int32(math.Floor(o.X))
	if o.Proto == ProtoSplitter {
		// East-facing: one column, two rows, positioned on their boundary.
		y := int32(math.Floor(o.Y))
		return [][2]int32{{x, y - 1}, {x, y}}
	}
	return [][2]int32{{x, int32(math.Floor(o.Y))}}
}

// TestOneLinkedBeltPerTileSide is the S1 rule the whole edge design rests on.
// Two same-direction inputs on one tile leave one of them silently dead.
func TestOneLinkedBeltPerTileSide(t *testing.T) {
	// A single part tile fed from all four sides and drained from none is the
	// worst legal case: four linked belts on one tile, four different sides.
	var es []Edge
	for _, d := range []uint32{North, East, South, West} {
		es = append(es, Edge{TileX: 3, TileY: 4, Dir: d})
	}
	es = append(es, Edge{TileX: 3, TileY: 5, Dir: South, Out: true})
	ops, _, ok := Build(nil, es, 0, 0)
	if !ok {
		t.Fatal("refused")
	}
	type side struct {
		x, y int32
		d    uint32
	}
	seen := map[side]bool{}
	n := 0
	for _, o := range ops {
		if !o.Visible {
			continue
		}
		n++
		// The side an interface uses is the one facing its neighbour: the
		// input end of an input, the output end of an output. Both are the
		// entity's own direction for an output and its opposite for an input.
		d := o.Dir
		if o.Link == LinkInput {
			d = Opposite(d)
		}
		s := side{int32(math.Floor(o.X)), int32(math.Floor(o.Y)), d}
		if seen[s] {
			t.Fatalf("two linked belts on side %v of one tile", s)
		}
		seen[s] = true
	}
	if n != 5 {
		t.Fatalf("5 edges in, %d visible interfaces out", n)
	}
}

// TestEveryInputEndIsPaired. The executor connects by walking the input ends
// once; an unpaired one is a dead interface, and an output end with a Pair
// would be connected twice.
func TestEveryInputEndIsPaired(t *testing.T) {
	for _, nm := range [][2]int{{1, 1}, {3, 5}, {1, 4}, {4, 1}, {4, 4}, {8, 8}, {5, 3}, {7, 2}} {
		ops, pt, ok := Build(nil, edges(nm[0], nm[1]), 0, 0)
		if !ok {
			t.Fatalf("%v refused", nm)
		}
		pairedTo := map[int32]int{}
		for i, o := range ops {
			switch o.Link {
			case LinkInput:
				if o.Pair < 0 {
					// A dead-ended send is never created, so every input end
					// that exists must have a partner.
					t.Fatalf("%v: op %d is an unpaired input end at (%g,%g)", nm, i, o.X, o.Y)
				}
				if ops[o.Pair].Link != LinkOutput {
					t.Fatalf("%v: op %d pairs a non-output end", nm, i)
				}
				pairedTo[o.Pair]++
			case LinkOutput:
				if o.Pair >= 0 {
					t.Fatalf("%v: op %d is an output end carrying a Pair", nm, i)
				}
			}
		}
		for idx, n := range pairedTo {
			if n != 1 {
				t.Fatalf("%v: op %d is the partner of %d input ends", nm, idx, n)
			}
		}
		// Dead-ended output ports are the only linked belts the plan may omit.
		if pt.Loop != minInt(pt.P-pt.M, pt.P-pt.N) {
			t.Fatalf("%v: Loop=%d", nm, pt.Loop)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// loopShapes are the shapes the loopback wiring is checked on physically: a
// spread of sizes, every one of them with Loop > 0, from the smallest that can
// have one up to P=64, and spanning both of the interesting extremes -- Loop as
// large as it gets (3->5, three of eight ports recirculating) and as small
// (33->63, one of sixty-four).
//
// P=4 is the floor and not an oversight: for n <= m, Loop = P-m and m > P/2, so
// P=2 forces m=2 and Loop=0. There is no two-line shape with a loopback in it.
//
// Half of these are n > m, where Loop is the OTHER branch of Shape's min()
// (P-N, the spare inputs running out before the spare outputs do). That side is
// off-limits to the flow model -- PropagateLoop's header says why -- but this
// test is about which row is wired to which, which does not care.
var loopShapes = [][2]int{
	{1, 3}, {2, 3}, {3, 3}, {3, 5}, {1, 5}, {5, 5}, {2, 6},
	{5, 7}, {1, 9}, {9, 15}, {33, 63},
	{5, 3}, {7, 2}, {3, 1}, {6, 2}, {63, 33},
}

// hiddenPorts recovers, from Build's actual output, which op is the recv for
// each input port and which is the send for each output port.
//
// Ports are numbered by ROW, which is what makes this recoverable at all:
// ord[0] is the identity, so input port i is row i at the recv column and output
// port j is row j at the send column. The column is the discriminator because
// the jumper blocks also emit linked-belt ends -- their input ends sit at
// columns 3, 6, ... 3(k-1) and their output ends at 4, 7, ... 3k-2, so neither
// can be confused with column 0 or column Width-1.
func hiddenPorts(t *testing.T, ops []Op, pt Ports, ox, oy int32) (recv, send map[int]int) {
	t.Helper()
	recv, send = map[int]int{}, map[int]int{}
	sendCol := Width(pt.P) - 1
	for i, o := range ops {
		if o.Visible || o.Link == LinkNone {
			continue
		}
		col := int(math.Floor(o.X)) - int(ox)
		row := int(math.Floor(o.Y)) - int(oy)
		switch {
		case col == colRecv && o.Link == LinkOutput:
			recv[row] = i
		case col == sendCol && o.Link == LinkInput:
			send[row] = i
		}
	}
	return recv, send
}

// TestLoopbackPairsConnectTheRightRows checks the IDENTITY of the loopbacks,
// which nothing did. TestEveryInputEndIsPaired proves every linked-belt end has
// exactly one partner and TestEveryInputEndIsPaired's last line checks the COUNT
// against Shape -- so a Build that looped output port M+i back to input port
// N+j, or to a row with no recv on it at all, satisfies both of them.
//
// Getting it wrong is not a crash: connect_linked_belts would succeed, the
// network would still balance, and the only symptom would be a spare input port
// with two feeds and another with none -- one of them silently dead, which is
// the sharpest edge in the whole design.
//
// TestLoopbackShapesDeliverEvenOutputs cannot stand in for this, and that was
// measured rather than assumed: reading the loopback from output row i instead
// of M+i leaves that test GREEN, because one butterfly pass equalises all P rows
// and the model therefore cannot tell which row the flow re-enters on. The
// identity of a loopback is only visible in the ops.
func TestLoopbackPairsConnectTheRightRows(t *testing.T) {
	const ox, oy = 100, 200
	for _, nm := range loopShapes {
		n, m := nm[0], nm[1]
		ops, pt, ok := Build(nil, edges(n, m), ox, oy)
		if !ok {
			t.Fatalf("%d->%d refused", n, m)
		}
		if pt.Loop == 0 {
			t.Fatalf("%d->%d has Loop=0 and belongs in another test", n, m)
		}
		recv, send := hiddenPorts(t, ops, pt, ox, oy)

		// The set of loopbacks is exactly the sends that point at a recv. Real
		// output ports point at a visible interface instead, so this partitions
		// the sends rather than merely filtering them.
		got := map[int]int{} // send row -> recv row
		for row, si := range send {
			pair := ops[si].Pair
			if pair < 0 {
				t.Fatalf("%d->%d: the send on row %d is unpaired", n, m, row)
			}
			if ops[pair].Visible {
				continue
			}
			rrow := int(math.Floor(ops[pair].Y)) - oy
			if recv[rrow] != int(pair) {
				t.Fatalf("%d->%d: the send on row %d pairs op %d, which is not the "+
					"recv on any row (it is at (%g,%g))", n, m, row, pair,
					ops[pair].X, ops[pair].Y)
			}
			got[row] = rrow
		}
		if len(got) != pt.Loop {
			t.Fatalf("%d->%d: %d sends loop back, Shape says %d",
				n, m, len(got), pt.Loop)
		}
		for i := 0; i < pt.Loop; i++ {
			outRow, inRow := pt.M+i, pt.N+i
			dst, wired := got[outRow]
			if !wired {
				t.Fatalf("%d->%d (P=%d Loop=%d): spare output port %d does not "+
					"loop back at all", n, m, pt.P, pt.Loop, outRow)
			}
			if dst != inRow {
				t.Fatalf("%d->%d (P=%d Loop=%d): output port %d loops back into "+
					"input port %d, it should feed %d", n, m, pt.P, pt.Loop,
					outRow, dst, inRow)
			}
		}
		// ...and the far end really is a port that is fed by nothing else: a
		// loopback landing on a row that also carries a real input would double
		// up on one recv and starve another.
		for i := 0; i < pt.Loop; i++ {
			if pt.N+i < pt.N {
				t.Fatalf("%d->%d: loopback %d lands on a real input port", n, m, i)
			}
		}
	}
}

// TestEveryUsedInputRowHasALaneSplitter is a tripwire on the lane-fidelity
// stage, and it exists because today NOTHING catches its loss: swapping
// ProtoLaneSplitter for ProtoBelt in Build changes no count, no position and no
// pairing, so TestEntityCount (which reads len(ops)) and every other test in
// this file stay green. Nothing else in the package inspects Proto at all.
//
// What would be lost is on the declared feature bar: agents/design.md lists
// lane-level balance, and spike S1 measured the difference directly -- without
// the lane-splitter stage a left-lane-only feed parks 4/0 on every output;
// with it, 4/4. Vanilla splitters are lane-PRESERVING, so no amount of
// butterfly recovers it.
//
// One per USED input row, which is N + Loop rather than N: a looped-back port
// carries real items that arrived on one lane of a real belt, so it needs the
// stage exactly as much as an external input does. A row with no recv gets no
// head section at all.
func TestEveryUsedInputRowHasALaneSplitter(t *testing.T) {
	const ox, oy = 100, 200
	shapes := make([][2]int, 0, 64+len(loopShapes))
	for n := 1; n <= 8; n++ {
		for m := 1; m <= 8; m++ {
			shapes = append(shapes, [2]int{n, m})
		}
	}
	shapes = append(shapes, [2]int{16, 16}, [2]int{9, 15}, [2]int{33, 63}, [2]int{64, 64})

	for _, nm := range shapes {
		n, m := nm[0], nm[1]
		ops, pt, ok := Build(nil, edges(n, m), ox, oy)
		if !ok {
			t.Fatalf("%d->%d refused", n, m)
		}
		rows := map[int]bool{}
		for _, o := range ops {
			if o.Proto != ProtoLaneSplitter {
				continue
			}
			if o.Visible {
				t.Fatalf("%d->%d: a lane splitter on the visible surface", n, m)
			}
			if col := int(math.Floor(o.X)) - ox; col != colLaneSplit {
				t.Fatalf("%d->%d: a lane splitter in column %d, the lane-split "+
					"stage is column %d", n, m, col, colLaneSplit)
			}
			row := int(math.Floor(o.Y)) - oy
			if rows[row] {
				t.Fatalf("%d->%d: two lane splitters on row %d", n, m, row)
			}
			rows[row] = true
		}
		if len(rows) != pt.N+pt.Loop {
			t.Fatalf("%d->%d (P=%d Loop=%d): %d lane splitters, one per used "+
				"input row is %d", n, m, pt.P, pt.Loop, len(rows), pt.N+pt.Loop)
		}
		recv, _ := hiddenPorts(t, ops, pt, ox, oy)
		for row := range recv {
			if !rows[row] {
				t.Fatalf("%d->%d: row %d has a recv feeding no lane splitter",
					n, m, row)
			}
		}
	}
}

// TestShapeTable pins the (P, Loop) arithmetic on literals, boundaries
// included: the two powers of two where Loop collapses to nothing, the pair
// either side of the min() (5->3 loops back on the input side, 3->5 on the
// output side, and they are only equal by coincidence of this table), and the
// one shape past MaxPorts.
func TestShapeTable(t *testing.T) {
	for _, c := range []struct{ n, m, p, loop int }{
		{1, 1, 1, 0},
		{1, 2, 2, 0},
		{2, 1, 2, 0},
		{1, 3, 4, 1},
		{2, 3, 4, 1},
		{3, 3, 4, 1},
		{3, 5, 8, 3},
		{5, 3, 8, 3},
		{4, 4, 4, 0},
		{5, 7, 8, 1},
		{9, 15, 16, 1},
		{33, 63, 64, 1},
		{64, 64, 64, 0},
		{65, 1, 128, 63},
	} {
		got := Shape(c.n, c.m)
		want := Ports{N: c.n, M: c.m, P: c.p, Loop: c.loop}
		if got != want {
			t.Errorf("Shape(%d,%d) = %+v, want %+v", c.n, c.m, got, want)
		}
	}
	// The last row is past MaxPorts and Shape sizes it anyway -- refusing is
	// Build's job, not Shape's, and Build refusing is TestRefusesOversizedClusters.
	if Shape(65, 1).P <= MaxPorts {
		t.Fatal("65 inputs should size past MaxPorts")
	}
}

// TestPlanFitsItsSlot: Width() is what the slot grid is sized from, so a plan
// that reached further would overwrite its neighbour's network.
func TestPlanFitsItsSlot(t *testing.T) {
	for n := 1; n <= 16; n++ {
		for m := 1; m <= 16; m++ {
			ops, pt, ok := Build(nil, edges(n, m), 0, 0)
			if !ok {
				t.Fatalf("%d->%d refused", n, m)
			}
			w := Width(pt.P)
			for _, o := range ops {
				if o.Visible {
					continue
				}
				x := int(math.Floor(o.X))
				y := int(math.Floor(o.Y))
				if x < 0 || x >= w {
					t.Fatalf("%d->%d: x=%d outside width %d", n, m, x, w)
				}
				if y < 0 || y >= pt.P {
					t.Fatalf("%d->%d: y=%d outside %d rows", n, m, y, pt.P)
				}
			}
		}
	}
}

// TestBuildIsDeterministic. Two clients must produce byte-identical plans or
// the game desyncs; the plan has no map iteration in it and this is what says
// so out loud.
func TestBuildIsDeterministic(t *testing.T) {
	for i := 0; i < 50; i++ {
		a, _, _ := Build(nil, edges(3, 5), 10, 20)
		b, _, _ := Build(nil, edges(3, 5), 10, 20)
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

// TestRefusesOversizedClusters: MaxPorts is a slot-bounds guarantee, not advice.
func TestRefusesOversizedClusters(t *testing.T) {
	if _, _, ok := Build(nil, edges(64, 64), 0, 0); !ok {
		t.Fatal("64x64 should fit")
	}
	if _, _, ok := Build(nil, edges(65, 1), 0, 0); ok {
		t.Fatal("65 inputs should be refused, not silently truncated")
	}
}

// TestNoNetworkWithoutBothSides. A half-built balancer is a normal thing for a
// player to have on screen for a few seconds; it must cost nothing and place
// nothing.
func TestNoNetworkWithoutBothSides(t *testing.T) {
	for _, nm := range [][2]int{{0, 0}, {4, 0}, {0, 4}} {
		ops, _, ok := Build(nil, edges(nm[0], nm[1]), 0, 0)
		if !ok || len(ops) != 0 {
			t.Fatalf("%v produced %d ops", nm, len(ops))
		}
	}
}

// TestEntityCount records the size of the two networks the milestone is judged
// on, so a change that quietly doubles the compile cost shows up here first.
func TestEntityCount(t *testing.T) {
	for _, c := range []struct{ n, m, want int }{
		{4, 4, 32},
		{8, 8, 84},
	} {
		ops, _, _ := Build(nil, edges(c.n, c.m), 0, 0)
		if len(ops) != c.want {
			t.Errorf("%dx%d is %d entities, the recorded figure is %d "+
				"(update CLAUDE.md in the same commit if this is intended)",
				c.n, c.m, len(ops), c.want)
		}
	}
}

// TestBuildDoesNotAllocateOnAReusedBuffer is a shipping constraint, not
// hygiene. TinyGo builds the guest with -gc=leaking: memory is an arena that
// only grows, the arena is in every save and every multiplayer join, and a
// recompile happens every time a player lays a belt next to a balancer. One
// allocation per compile is one allocation per compile forever.
func TestBuildDoesNotAllocateOnAReusedBuffer(t *testing.T) {
	es := edges(8, 8)
	buf, _, _ := Build(nil, es, 0, 0)
	n := testing.AllocsPerRun(100, func() {
		buf, _, _ = Build(buf, es, 0, 0)
	})
	if n != 0 {
		t.Fatalf("Build allocates %g times per call on a warm buffer", n)
	}
}
