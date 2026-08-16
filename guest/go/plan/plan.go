// Package plan turns a cluster's belt edges into a list of entities to place.
//
// It is the compiler's middle end and it is DELIBERATELY PURE: no fkapi, no
// wasm imports, no host calls. That is what lets `go test ./guest/go/plan` run
// it under a normal Go toolchain and prove the balance property by simulation
// rather than by staring at a Factorio screenshot. The thin execution layer in
// the guest walks the Op list calling create_entity; it makes no decisions.
//
// # The network
//
// Given N input edges and M output edges, the balancer runs over
// P = next_pow2(max(N, M)) lines. Lines are physical ROWS on the hidden
// surface, all flowing east, and the network is a butterfly: log2(P) stages,
// each of P/2 splitters, where stage s joins the lines whose indices differ in
// bit s.
//
// Why that balances, exactly, under every load: before stage s every aligned
// block of 2^s lines carries equal flow (trivially true for s = 0). Stage s
// joins each line of block A with its partner in the adjacent block B, and a
// splitter makes its two outputs equal, so every line of A u B leaves stage s
// carrying (a+b)/2. By induction all P lines are equal after stage log2(P)-1.
// TestEveryRowEndsEqual checks it over exhaustive and random inputs.
//
// # Why the rows have to move
//
// A Factorio splitter spans two ADJACENT rows, and stage s wants to join rows r
// and r^2^s, which are adjacent only for s = 0. So before each stage the rows
// are permuted so that stage s's pairs land on rows (0,1), (2,3), ...  A row
// that has to move does so through a linked-belt jumper pair -- spike S1 found
// those cleaner than underground crossings and equally native (a hop costs zero
// throughput, only pipeline latency).
//
// # There are no belts between the pieces
//
// Every element sits directly against the next: recv, lane splitter, splitter,
// splitter, jumper, splitter, send, with no transport belt anywhere except the
// two tiles a jumper block needs for the rows that are NOT moving. Belts would
// only add buffer, and every entity is ~200 bytes of permanently leaked guest
// heap (see FKLUA-GAPS.md) plus a host call. Dropping them takes a 4x4 from 50
// entities to 32 and an 8x8 from 132 to 84. S1 established that linked belts,
// splitters and undergrounds connect to each other natively, which is the only
// fact this relies on.
//
// # Unused ports
//
// P is a power of two and N, M generally are not. Spare OUTPUT ports are looped
// back into spare INPUT ports where there are enough of them; the rest are
// dead-ended, which is correct rather than merely tolerable -- spike S1
// measured a 4x4 with one output blocked and the remaining three stayed exact.
// A spare INPUT port with nothing feeding it needs no head section at all.
//
// # Allocation
//
// Build appends into a caller-owned slice and every working array is a package
// level fixed buffer. TinyGo builds this guest with -gc=leaking, so an
// allocation per compile is an allocation per compile FOREVER, and the guest
// heap is in every save and every multiplayer join.
package plan

// The four cardinals of Factorio's `defines.direction`.
//
// VARIABLES, AND NOT WRITTEN DOWN ANYWHERE. A define's number is Factorio's own,
// is not stable across versions and is not in `runtime-api.json` at all, so
// there is nothing to bake and a compiled-in `4` is a guess that happens to be
// right today. They used to be `const`s here with a `test/check-layout.py` that
// re-derived them from the pinned API description's `order` field -- which was
// the same guess, checked against itself. `fklua gen-bindings` emits a
// `DefinesDirection*()` accessor per path now (FKLUA-GAPS.md item 11), the value
// is resolved by NAME against the running game at load, and compile.go's
// initBuffers installs the four through SetCompass.
//
// This package stays free of `fkapi` on purpose: it has no wasm imports, which
// is what lets `go test ./plan/` prove the balance property under an ordinary
// toolchain. So the compass is pushed in rather than pulled.
var North, East, South, West uint32

// SetCompass installs the running game's `defines.direction` values. Called once
// per guest lifetime, from a package initialiser, before anything below runs.
//
// Not calling it leaves all four at 0, which is a network whose every entity
// faces north and whose Opposite is the identity. That is not a subtle failure:
// M2 asserts the compiled network BALANCES, and one that does not move items at
// all fails on the first rig. The unit tests install it from TestMain for the
// same reason.
func SetCompass(n, e, s, w uint32) { North, East, South, West = n, e, s, w }

// Opposite turns a direction around: two quarter turns on the installed compass.
//
// It used to be `(d + 8) & 15`, which is the 16-way compass's arithmetic and
// therefore one more thing assumed about numbers this package no longer knows.
// Defined on the four cardinals, which is every direction this mod ever forms --
// an edge direction is one of cluster.go's `dirs`, and a belt whose direction is
// not cardinal matches neither `dir` nor `back` in classifySide and is not an
// edge at all.
func Opposite(d uint32) uint32 {
	switch d {
	case North:
		return South
	case East:
		return West
	case South:
		return North
	case West:
		return East
	}
	return d
}

// MaxPorts caps a single network. 64 lines is six stages and ~500 entities,
// which is already far past any balancer a player builds; beyond it the slot
// grid's bounds stop holding and a compile is refused loudly rather than
// silently overrunning its neighbour.
const MaxPorts = 64

// maxStages is log2(MaxPorts).
const maxStages = 6

// Proto names a prototype the compiler places. The plan carries no strings at
// all: every string that crosses the host boundary costs ~1.7 us and a plan has
// dozens of entries.
type Proto uint8

const (
	ProtoBelt Proto = iota
	ProtoSplitter
	ProtoLaneSplitter
	ProtoLinkedBelt
)

// LinkType is a linked belt's end. Items go IN at an input end and come OUT at
// the paired output end, on any surface.
type LinkType uint8

const (
	LinkNone LinkType = iota
	LinkInput
	LinkOutput
)

// Op is one entity to create.
//
// Pair is an index into the same Op slice: the partner to call
// connect_linked_belts with. The invariant the executor relies on is that
// EVERY LinkInput op carries a Pair and every LinkOutput op carries -1, so one
// pass over the list connects each pair exactly once.
type Op struct {
	X, Y  float64
	Pair  int32
	Dir   uint32
	Proto Proto
	Link  LinkType
	// Visible puts the entity on the cluster's own surface instead of the
	// hidden one. Only the edge interfaces are visible.
	Visible bool
}

// Edge is one belt touching the cluster: a tile of the cluster, and the
// direction the linked belt placed there must face.
//
// One edge is one adjacent belt-connectable, and it consumes ONE SIDE of that
// tile. Two edges may share a tile (S1 ran four on one tile at full rate) but
// never a side: two same-direction inputs on a tile leave one of them silently
// dead, which is the sharpest edge in the whole design.
type Edge struct {
	TileX, TileY int32
	Dir          uint32
	Out          bool
}

// The column layout, left to right:
//
//	0        recv    linked belt, output end -- the visible input's partner
//	1        lane splitter -- the lane-fidelity stage (S1: without it a
//	         left-lane-only feed parks 4/0 on every output; with it, 4/4)
//	2        stage 0 splitters
//	then per stage s > 0:
//	         jumper IN ends / straight belt
//	         jumper OUT ends / straight belt
//	         stage s splitters
//	last     send    linked belt, input end -- the visible output's partner
//
// The jumper block is two columns and cannot be one: a row that is moving away
// can also be the destination of another row's jump, and both ends would want
// the same tile.
const (
	colRecv       = 0
	colLaneSplit  = 1
	colFirstStage = 2

	jumperCols = 2
)

// Width is how many tile columns a P-line network occupies.
func Width(p int) int {
	k := stages(p)
	w := colFirstStage + k
	if k > 1 {
		w += (k - 1) * jumperCols
	}
	return w + 1 // the send column
}

// stages is log2(p) for a power of two.
func stages(p int) int {
	k := 0
	for 1<<uint(k) < p {
		k++
	}
	return k
}

// NextPow2 rounds up to a power of two, with 0 -> 0 and 1 -> 1.
func NextPow2(n int) int {
	if n <= 1 {
		return n
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// Ports is the shape of the network the edges imply.
type Ports struct {
	N, M, P int
	// Loop is how many spare output ports are wired back into spare input
	// ports. Output ports [M, M+Loop) feed input ports [N, N+Loop).
	Loop int
}

// Shape sizes a network for n inputs and m outputs.
func Shape(n, m int) Ports {
	hi := n
	if m > hi {
		hi = m
	}
	p := NextPow2(hi)
	loop := p - m
	if p-n < loop {
		loop = p - n
	}
	if loop < 0 {
		loop = 0
	}
	return Ports{N: n, M: m, P: p, Loop: loop}
}

// ---------------------------------------------------------------------------
// Working buffers. Package level and fixed size, because -gc=leaking makes any
// per-call allocation permanent. Nothing here is re-entrant, and nothing has to
// be: a guest call is a single thread inside one Factorio tick.
// ---------------------------------------------------------------------------

var (
	ordBuf   [maxStages][MaxPorts]int
	ordRows  [maxStages][]int
	permBuf  [MaxPorts]int
	whereBuf [MaxPorts]int
	inUsed   [MaxPorts]bool
	outUsed  [MaxPorts]bool
	recvOp   [MaxPorts]int32
	sendOp   [MaxPorts]int32
	jinBuf   [MaxPorts]int32
)

// order returns, for each stage, the logical line sitting at each physical row
// just before that stage runs.
//
// Stage s must join lines i and i|2^s, so its pairs are laid on rows (0,1),
// (2,3), ... in ascending order of i. ord[0] is the identity by construction,
// which is why the input ports can be numbered by row.
//
// The result aliases a package buffer and is valid until the next call.
func order(p int) [][]int {
	k := stages(p)
	for s := 0; s < k; s++ {
		row := ordBuf[s][:p]
		t := 0
		for i := 0; i < p; i++ {
			if i&(1<<uint(s)) != 0 {
				continue
			}
			row[2*t] = i
			row[2*t+1] = i | 1<<uint(s)
			t++
		}
		ordRows[s] = row
	}
	return ordRows[:k]
}

// permutation maps each physical row to where its contents must be before the
// next stage. The result aliases a package buffer.
func permutation(from, to []int) []int {
	p := len(from)
	where := whereBuf[:p]
	for r, line := range to {
		where[line] = r
	}
	out := permBuf[:p]
	for r, line := range from {
		out[r] = where[line]
	}
	return out
}

// Build produces the entity list for a cluster, appending into dst.
//
// ox, oy are the slot origin on the hidden surface, in tiles. The ops come back
// in creation order and reference each other only by index, never by position,
// so the executor never has to search for anything.
//
// ok is false when the cluster is beyond MaxPorts; the caller must refuse the
// compile rather than build a network that overruns its slot.
func Build(dst []Op, edges []Edge, ox, oy int32) (ops []Op, pt Ports, ok bool) {
	ops = dst[:0]
	n, m := 0, 0
	for _, e := range edges {
		if e.Out {
			m++
		} else {
			n++
		}
	}
	pt = Shape(n, m)
	if pt.N == 0 || pt.M == 0 {
		// Nothing to balance: a cluster with no inputs or no outputs is a
		// legitimate half-built state, not an error.
		return ops, pt, true
	}
	if pt.P > MaxPorts {
		return ops, pt, false
	}

	p := pt.P
	k := stages(p)
	ord := order(p)

	for i := 0; i < p; i++ {
		inUsed[i] = i < pt.N
		outUsed[i] = i < pt.M
		recvOp[i], sendOp[i] = -1, -1
	}
	for i := 0; i < pt.Loop; i++ {
		inUsed[pt.N+i] = true
		outUsed[pt.M+i] = true
	}

	// No closures anywhere below. A closure that mutates `ops` forces the slice
	// header onto the heap, and a heap allocation per compile is a heap
	// allocation per compile forever under -gc=leaking.
	fxBase := float64(ox) + 0.5
	fyBase := float64(oy) + 0.5

	// --- head: recv, lane splitter -----------------------------------------
	for r := 0; r < p; r++ {
		if !inUsed[r] {
			continue
		}
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fxBase + colRecv, Y: fyBase + float64(r),
			Dir: East, Link: LinkOutput, Pair: -1})
		recvOp[r] = int32(len(ops) - 1)
		ops = append(ops, Op{Proto: ProtoLaneSplitter, X: fxBase + colLaneSplit, Y: fyBase + float64(r),
			Dir: East, Pair: -1})
	}

	// --- stages -------------------------------------------------------------
	col := colFirstStage
	for s := 0; s < k; s++ {
		if s > 0 {
			perm := permutation(ord[s-1], ord[s])
			jin := jinBuf[:0]
			for r := 0; r < p; r++ {
				if perm[r] == r {
					ops = append(ops, Op{Proto: ProtoBelt, X: fxBase + float64(col),
						Y: fyBase + float64(r), Dir: East, Pair: -1})
					ops = append(ops, Op{Proto: ProtoBelt, X: fxBase + float64(col+1),
						Y: fyBase + float64(r), Dir: East, Pair: -1})
					continue
				}
				ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fxBase + float64(col),
					Y: fyBase + float64(r), Dir: East, Link: LinkInput, Pair: -1})
				jin = append(jin, int32(len(ops)-1))
			}
			j := 0
			for r := 0; r < p; r++ {
				if perm[r] == r {
					continue
				}
				ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fxBase + float64(col+1),
					Y: fyBase + float64(perm[r]), Dir: East, Link: LinkOutput, Pair: -1})
				ops[jin[j]].Pair = int32(len(ops) - 1)
				j++
			}
			col += jumperCols
		}
		for t := 0; t < p/2; t++ {
			// An east-facing splitter's position is on the boundary between the
			// two rows it spans, not on either tile's centre.
			ops = append(ops, Op{Proto: ProtoSplitter, X: fxBase + float64(col),
				Y: float64(oy+int32(2*t)) + 1.0, Dir: East, Pair: -1})
		}
		col++
	}

	// --- tail: send ---------------------------------------------------------
	for r := 0; r < p; r++ {
		if !outUsed[r] {
			continue
		}
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fxBase + float64(col), Y: fyBase + float64(r),
			Dir: East, Link: LinkInput, Pair: -1})
		sendOp[r] = int32(len(ops) - 1)
	}

	// --- loopbacks ----------------------------------------------------------
	for i := 0; i < pt.Loop; i++ {
		ops[sendOp[pt.M+i]].Pair = recvOp[pt.N+i]
	}

	// --- the visible edge interfaces ---------------------------------------
	//
	// An INPUT edge is a linked belt whose input end faces the incoming belt;
	// its partner is the hidden recv, which is an output end. An OUTPUT edge is
	// the mirror image. `connect_linked_belts` wants opposite ends, which is
	// exactly what this produces.
	nin, nout := 0, 0
	for _, e := range edges {
		if e.Out {
			ops = append(ops, Op{Proto: ProtoLinkedBelt, X: float64(e.TileX) + 0.5,
				Y: float64(e.TileY) + 0.5, Dir: e.Dir, Link: LinkOutput, Visible: true, Pair: -1})
			ops[sendOp[nout]].Pair = int32(len(ops) - 1)
			nout++
		} else {
			ops = append(ops, Op{Proto: ProtoLinkedBelt, X: float64(e.TileX) + 0.5,
				Y: float64(e.TileY) + 0.5, Dir: e.Dir, Link: LinkInput, Visible: true,
				Pair: recvOp[nin]})
			nin++
		}
	}
	return ops, pt, true
}

// ---------------------------------------------------------------------------
// The reference model. Not shipped logic -- this is what the unit tests check
// the wiring against, and it lives here rather than in the test file so that
// the schedule and the layout are read from the same source. Nothing in the
// guest reaches it, so it is dead-code-eliminated out of the wasm.
// ---------------------------------------------------------------------------

// Propagate runs a flow vector through the network the physical layout builds:
// permute rows, then average adjacent pairs, once per stage.
func Propagate(in []float64) []float64 {
	p := len(in)
	k := stages(p)
	ord := order(p)
	f := append([]float64(nil), in...)
	next := make([]float64, p)
	for s := 0; s < k; s++ {
		if s > 0 {
			perm := permutation(ord[s-1], ord[s])
			for r := 0; r < p; r++ {
				next[perm[r]] = f[r]
			}
			copy(f, next)
		}
		for t := 0; t < p/2; t++ {
			mid := (f[2*t] + f[2*t+1]) / 2
			f[2*t], f[2*t+1] = mid, mid
		}
	}
	return f
}

// loopIters caps the fixed-point search in PropagateLoop.
//
// The iteration is a contraction with ratio Loop/P, and Loop/P < 1/2 for every
// shape PropagateLoop is valid on (see below: P is next_pow2(m), so m > P/2 and
// Loop = P-m < P/2), which reaches 1e-12 in about fifty passes. The cap is two
// orders above that so it can only ever be reached by a wiring that does not
// converge at all -- which the caller is told about rather than hidden from.
const loopIters = 500

// PropagateLoop is Propagate with the LOOPBACK wiring modelled: the fixed point
// of the network Build actually places, rather than of the butterfly alone.
//
// Build feeds spare output ports [M, M+Loop) back into spare input ports
// [N, N+Loop), and Propagate knows nothing about it -- it takes a P-vector and
// returns a P-vector, never consulting Ports. So the recirculating half of every
// shape with Loop > 0 (36 of the 64 shapes with n,m <= 8) was unmodelled, and
// only 3->5 had any evidence at all, from the in-game M2 suite.
//
// The model: rows [0, N) receive the external flow, rows [N, N+Loop) receive
// last pass's output on rows [M, M+Loop), and rows [N+Loop, P) receive nothing
// -- a spare input port with nothing feeding it gets no recv section at all.
// Iterate to the fixed point. Since one Propagate pass equalises all P rows, the
// steady state is arithmetic: total entering T = S + Loop*T/P, so every row
// carries T/P = S/(P-Loop) = S/max(N, M). For 3->5 saturated that is 3/5 of a
// belt per output, which is what M2 measures in the game (782 items against a
// bare belt's 1306).
//
// # Where this is exact, and where it is not
//
// ONLY FOR N <= M, which is why the tests over it stop there. For n <= m,
// Loop = P-M, so every port that is not a real output is looped back, nothing
// dead-ends, and the load on a line is S/m <= n/m <= one belt: the network free
// flows and a linear flow model is the whole truth. For n > m the loopbacks run
// out (Loop = P-N < P-M) and the remaining spare outputs are DEAD-ENDED, which
// in the game backs up, blocks its splitter's other output and re-routes the
// flow -- a saturation nonlinearity that no linear model can express, and which
// this one would silently get wrong. M2's `a4to1` and `starve` rigs cover that
// side in a real Factorio, which is the only place it can be covered.
//
// ok is false if the shape is inconsistent with ext, or if the iteration did not
// converge. Failing by return rather than by panic is the house rule: a panic
// links ~73 KB of TinyGo's print machinery into a guest that has no other use
// for it. Nothing in the guest reaches this function, so it costs the wasm
// nothing either way, but the style is worth keeping consistent.
func PropagateLoop(ext []float64, pt Ports) (rows []float64, ok bool) {
	p := pt.P
	if p <= 0 || len(ext) != pt.N || pt.N+pt.Loop > p || pt.M+pt.Loop > p {
		return nil, false
	}
	f := make([]float64, p)
	in := make([]float64, p)
	for it := 0; it < loopIters; it++ {
		for r := range in {
			in[r] = 0
		}
		copy(in, ext)
		for i := 0; i < pt.Loop; i++ {
			in[pt.N+i] = f[pt.M+i]
		}
		next := Propagate(in)
		var d float64
		for r, v := range next {
			e := v - f[r]
			if e < 0 {
				e = -e
			}
			if e > d {
				d = e
			}
		}
		copy(f, next)
		if d < 1e-12 {
			return f, true
		}
	}
	return f, false
}
