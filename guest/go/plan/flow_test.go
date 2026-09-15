package plan

import (
	"math"
	"testing"
)

// The flow model, anchored against the one the shipped network was measured
// with.
//
// Simulate walks the ops Build emits and solves for capacity and back-pressure;
// PropagateLoop is linear and reads the schedule. They answer the same question
// for every shape the linear one is valid on, and where they part company is
// exactly where the linear one says it stops (n > m, the dead-ended spare
// ports). A model that agreed with nothing would be a second implementation of
// the thing under test.

// The two loads every test here builds: a free output belt on every port, and
// the same rate on every input.
func freeOutputs(m int) []float64 {
	out := make([]float64, m)
	for i := range out {
		out[i] = 1
	}
	return out
}

func evenInputs(n int, load float64) []float64 {
	in := make([]float64, n)
	for i := range in {
		in[i] = load
	}
	return in
}

// TestPlainShapesMatchTheLinearModel. For n <= m nothing dead-ends, the network
// free flows, and every output carries S/max(N, M) -- which is what
// TestLoopbackShapesDeliverEvenOutputs asserts of PropagateLoop and what M2
// measures in the game (3->5 saturated delivers 782 items against a bare belt's
// 1306, which is three fifths).
func TestPlainShapesMatchTheLinearModel(t *testing.T) {
	for m := 1; m <= 8; m++ {
		for n := 1; n <= m; n++ {
			ops, pt, ok := Build(nil, edges(n, m), 100, 200)
			if !ok {
				t.Fatalf("%d->%d refused", n, m)
			}
			for _, load := range []float64{0.125, 0.5, 1.0} {
				del, tak, conv := Simulate(ops, evenInputs(n, load), freeOutputs(m))
				if !conv {
					t.Fatalf("%d->%d load %g: the flow model did not converge", n, m, load)
				}
				var s float64
				for _, v := range tak {
					s += v
				}
				if want := float64(n) * load; math.Abs(s-want) > 1e-9 {
					t.Fatalf("%d->%d load %g: the network took %g of the %g on offer, "+
						"and nothing should throttle it for n <= m", n, m, load, s, want)
				}
				rows, lin := PropagateLoop(evenInputs(n, load), pt)
				if !lin {
					t.Fatalf("%d->%d: PropagateLoop did not converge", n, m)
				}
				for j := 0; j < m; j++ {
					if math.Abs(del[j]-rows[j]) > 1e-9 {
						t.Fatalf("%d->%d load %g: output %d carries %g and the linear "+
							"model says %g", n, m, load, j, del[j], rows[j])
					}
				}
			}
		}
	}
}

// TestFlowConservesWhatItTakes. A network that delivered more than it took, or
// swallowed the difference, would make every exactness figure in prio_test.go a
// measurement of nothing.
func TestFlowConservesWhatItTakes(t *testing.T) {
	for _, nm := range [][2]int{{4, 4}, {3, 5}, {5, 3}, {8, 8}, {7, 2}, {1, 8}, {9, 15}} {
		n, m := nm[0], nm[1]
		ops, _, _ := Build(nil, edges(n, m), 0, 0)
		for _, load := range []float64{0.1, 0.5, 1.0} {
			del, tak, ok := Simulate(ops, evenInputs(n, load), freeOutputs(m))
			if !ok {
				t.Fatalf("%d->%d load %g: no convergence", n, m, load)
			}
			var in, out float64
			for _, v := range tak {
				in += v
			}
			for _, v := range del {
				out += v
			}
			if math.Abs(in-out) > 1e-9 {
				t.Fatalf("%d->%d load %g: %g in, %g out", n, m, load, in, out)
			}
		}
	}
}

// TestSaturatedDeadEndShapesDeliverFullBelts is the half of n > m the game has
// measured: M2's a5to3 rig is five inputs into three outputs with two spare
// ports dead-ended, and it delivers 1304 items to each of the three against a
// bare belt's 1306, at 0.00% spread.
func TestSaturatedDeadEndShapesDeliverFullBelts(t *testing.T) {
	for _, nm := range [][2]int{{5, 3}, {4, 1}, {7, 5}, {8, 3}, {3, 2}} {
		n, m := nm[0], nm[1]
		ops, _, _ := Build(nil, edges(n, m), 0, 0)
		del, _, ok := Simulate(ops, evenInputs(n, 1), freeOutputs(m))
		if !ok {
			t.Fatalf("%d->%d: no convergence", n, m)
		}
		for j, v := range del {
			if math.Abs(v-1) > 1e-9 {
				t.Fatalf("%d->%d saturated: output %d carries %g and every one of "+
					"them should carry a full belt", n, m, j, v)
			}
		}
	}
}

// TestDeadEndedSpareOutputsSpreadUnderPartialLoad is a RECORD OF WHAT THE MODEL
// SAYS about the shipped plain network, and it is not a measurement of
// Factorio: no rig anywhere drives an n > m shape at partial load.
//
// Under saturation such a shape is exact, which the test above and M2's a5to3
// rig both say. Under partial load the model has the three outputs of a 5->3
// carrying 0.75, 0.75 and 1.0 of a belt rather than 0.8333 each, because two of
// its spare ports dead-end: a splitter whose one output is blocked gives the
// whole of its input to the other, and that re-routing is not symmetric.
//
// It is here because the priority construction is held to exactness at every
// load and the plain network is not, so the difference should be written down
// rather than discovered. agents/priority.md, "What the model found about the
// plain network".
func TestDeadEndedSpareOutputsSpreadUnderPartialLoad(t *testing.T) {
	ops, _, _ := Build(nil, edges(5, 3), 0, 0)
	del, tak, ok := Simulate(ops, evenInputs(5, 0.5), freeOutputs(3))
	if !ok {
		t.Fatal("5->3 at half load: no convergence")
	}
	var s float64
	for _, v := range tak {
		s += v
	}
	if math.Abs(s-2.5) > 1e-9 {
		t.Fatalf("5->3 at half load took %g of the 2.5 on offer", s)
	}
	want := []float64{0.75, 0.75, 1}
	for j, v := range del {
		if math.Abs(v-want[j]) > 1e-9 {
			t.Fatalf("5->3 at half load delivers %v; the model has recorded %v, and a "+
				"balancer would deliver %g to each", del, want, s/3)
		}
	}
}

// TestABlockedOutputIsNotRebalancedUnderPartialLoad is the SECOND record of
// what the model says about the shipped plain network, and it is the sharper of
// the two: block one output of a 4x4 and feed it a quarter of a belt on each
// input, and the port that PAIRS with the blocked one at the last stage takes
// double while the other two take their quarter.
//
// It is the butterfly rather than a defect. The last stage's splitter has two
// output rows and one of them is blocked, so it gives the whole of its input to
// the other -- and its input is half the network's flow, not a quarter. Under
// SATURATION the same rig is exact, which is what M2's `block` rig measures in
// the game: 4 out with the fourth blocked delivers 1306 1304 1306.
//
// No rig anywhere drives a blocked output at partial load, so this is a model
// result and not a measurement. The priority network does not share it --
// TestABlockedOutputIsRebalancedByTheTiers -- because a tier balancer is square
// and recirculates where the plain network dead-ends.
func TestABlockedOutputIsNotRebalancedUnderPartialLoad(t *testing.T) {
	ops, _, _ := Build(nil, edges(4, 4), 0, 0)
	out := freeOutputs(4)
	out[2] = 0
	del, tak, ok := Simulate(ops, evenInputs(4, 0.25), out)
	if !ok {
		t.Fatal("no convergence")
	}
	var s float64
	for _, v := range tak {
		s += v
	}
	if math.Abs(s-1) > 1e-9 {
		t.Fatalf("took %g of the 1 belt on offer", s)
	}
	want := []float64{0.25, 0.25, 0, 0.5}
	for j, v := range del {
		if math.Abs(v-want[j]) > 1e-9 {
			t.Fatalf("a 4x4 at quarter load with output 2 blocked delivers %v; the "+
				"model has recorded %v", del, want)
		}
	}
	// ...and saturated it is exact, which is the half the game has measured.
	del, _, _ = Simulate(ops, evenInputs(4, 1), out)
	for j, v := range del {
		want := 1.0
		if j == 2 {
			want = 0
		}
		if math.Abs(v-want) > 1e-9 {
			t.Fatalf("saturated with output 2 blocked: output %d carries %g, want %g",
				j, v, want)
		}
	}
}
