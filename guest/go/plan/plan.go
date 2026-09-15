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
	// OutPrio and InPrio are a SPLITTER's output and input priority: 0 is
	// none, -1 is "left" and +1 is "right" in the engine's own sense, which is
	// relative to the direction the splitter faces. Every hidden splitter
	// faces East, so "left" is North, the smaller y of the pair, and "right"
	// is South, the larger y. Zero on every op of a plain butterfly, and the
	// executor makes no host call for a zero.
	OutPrio, InPrio int8
}

// Priority values for Op.OutPrio and Op.InPrio. Every hidden splitter faces
// East, so the engine's "left" is North, which is the smaller y of the two
// tiles a splitter spans, and "right" is South.
const (
	PrioNone  int8 = 0
	PrioLeft  int8 = -1
	PrioRight int8 = 1
)

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
	// Prio marks a PRIORITY port: an output that is fed before the others, or
	// an input that is drained before the others, two-tier and boolean. It is
	// a property of the part the belt stands against (cluster.go's pprio), so
	// on an engine that lets a part carry two belts both of them read it. A
	// network with no Prio edge at all is the plain butterfly, byte for byte.
	Prio bool
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

// stageCols is how many tile columns k butterfly stages occupy: one per stage,
// plus a two-column jumper block in front of every stage but the first.
func stageCols(p int) int {
	k := stages(p)
	if k == 0 {
		return 0
	}
	return k + (k-1)*jumperCols
}

// Width is how many tile columns a P-line network occupies: the recv column,
// the lane-splitter column, the stages and the send column. It is also the
// width of any plain butterfly BLOCK, which is what the priority construction
// builds its two sub-balancers out of.
func Width(p int) int {
	return colFirstStage + stageCols(p) + 1
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
	// QIn and QOut count the Prio inputs and outputs among N and M. Both zero
	// is the plain butterfly.
	QIn, QOut int
}

// SlotWidth and SlotHeight are the hidden-surface slot a network must fit in,
// in tiles, and they are the PLANNER'S bound rather than compile.go's: a slot's
// box is what a teardown sweeps, so an entity outside it survives its own
// rebuild as a ghost nothing owns and is destroyed by its neighbour's. Every
// shape ShapeEdges says fits must place every op inside [0,SlotWidth) x
// [0,SlotHeight), and plan_test.go is what holds that.
const (
	SlotWidth  = 32
	SlotHeight = 72
)

// ShapeEdges sizes a network for an edge list and says whether it can be built
// at all -- the port cap and, for a shape with priority ports, the slot. It is
// the ONE check compile() makes before it touches anything and the one Build
// repeats, so the two cannot disagree; a cluster with no inputs or no outputs
// is a legitimate half-built state and fits at any size.
//
// EVERY OUTPUT MARKED PRIORITY IS THE PLAIN BALANCER, and it is reported as
// QOut = 0 rather than as QOut = M. Two tiers where the second tier is empty is
// one tier: the priority ports share equally among themselves and there is
// nobody to share the residual with, which is what the plain butterfly already
// does. Collapsing it here rather than in Build is what makes a player who
// ticks every output of a 64-port balancer get a network instead of a refusal.
// The same goes for the inputs.
func ShapeEdges(edges []Edge) (pt Ports, fits bool) {
	n, m, qi, qo := 0, 0, 0, 0
	for i := range edges {
		if edges[i].Out {
			m++
			if edges[i].Prio {
				qo++
			}
		} else {
			n++
			if edges[i].Prio {
				qi++
			}
		}
	}
	pt = Shape(n, m)
	if qo == m {
		qo = 0
	}
	if qi == n {
		qi = 0
	}
	pt.QIn, pt.QOut = qi, qo
	if pt.N == 0 || pt.M == 0 {
		return pt, true
	}
	if pt.P > MaxPorts {
		return pt, false
	}
	if qo == 0 && qi == 0 {
		return pt, true
	}
	// A priority network is TWO butterflies deep and carries its tiers in a
	// second band of rows, so the slot is what binds it rather than the port
	// cap.
	if qi > 0 {
		// Input priority is not built. buildPrio has no de-concentrator in it,
		// and an approximation of a bound this repository states in terms of
		// exactness is worse than a refusal. agents/priority.md, "What input
		// priority does".
		return pt, false
	}
	w, h := prioExtent(pt)
	return pt, w <= SlotWidth && h <= SlotHeight
}

// prioExtent is the bounding box of the priority construction, in tiles: the
// two bands buildPrio lays, measured rather than asserted by
// TestEveryPriorityShapeStaysInsideItsSlot.
//
// Band 0 is the head, the balancing butterfly, the concentrator and the tap
// column, over P rows. Band 1 is the two tier balancers side by side, under it.
func prioExtent(pt Ports) (w, h int) {
	band0 := colFirstStage + 2*stageCols(pt.P) + 1
	band1 := Width(NextPow2(pt.QOut)) + Width(NextPow2(pt.M-pt.QOut))
	h1 := NextPow2(pt.QOut)
	if t := NextPow2(pt.M - pt.QOut); t > h1 {
		h1 = t
	}
	w = band0
	if band1 > w {
		w = band1
	}
	return w, pt.P + h1
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

	// The priority path's own. rankBuf is one byte a row and the others are the
	// same [MaxPorts] the rest of this block is: about 1 KB of extra globals,
	// which is what a package-level buffer costs here rather than the
	// ~50 KB agents/maxports.md warns a raised MaxPorts would (the conservative
	// collector re-scans every global at every paced step, and test/run.sh
	// fails a run on its root-set warning).
	rankBuf  [MaxPorts]uint8
	headOp   [MaxPorts]int32
	tapOp    [MaxPorts]int32
	subOp    [2 * MaxPorts]int32
	portSend [MaxPorts]int32
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

// stagesAt lays the k butterfly stages of a p-row block: the splitter columns
// and the jumper block in front of every stage but the first. It starts at
// column `col` of the slot at (ox, oy) and returns the column after the last
// stage, so that blocks can be laid end to end in one band of rows.
//
// outPrio goes on every splitter it places. PrioNone is the plain butterfly;
// PrioLeft makes the same schedule a CONCENTRATOR, because `order` puts the
// line whose bit s is clear on the smaller y of every stage-s pair, so
// prioritising the smaller y prioritises the same half at every stage and the
// winners meet each other at the next one. See "The sorter" below.
func stagesAt(ops []Op, p, col int, ox, oy int32, outPrio int8) ([]Op, int) {
	k := stages(p)
	ord := order(p)
	fxBase := float64(ox) + 0.5
	fyBase := float64(oy) + 0.5
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
				Y: float64(oy+int32(2*t)) + 1.0, Dir: East, Pair: -1, OutPrio: outPrio})
		}
		col++
	}
	return ops, col
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
	pt, fits := ShapeEdges(edges)
	if pt.N == 0 || pt.M == 0 {
		// Nothing to balance: a cluster with no inputs or no outputs is a
		// legitimate half-built state, not an error.
		return ops, pt, true
	}
	if !fits {
		return ops, pt, false
	}
	if pt.QOut > 0 {
		return buildPrio(ops, edges, pt, ox, oy)
	}

	p := pt.P

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
	ops, col := stagesAt(ops, p, colFirstStage, ox, oy, PrioNone)

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
// Priority outputs
//
// The mod's promise for a priority output is the one a player can state: the
// ports they ticked are fed FIRST and share what they get equally, and the rest
// share whatever is left, equally. So with S belts arriving and q of the M
// outputs ticked, a priority output carries min(S/q, 1) and a normal one
// carries max(S-q, 0)/(M-q) -- exact at every load rather than close at most of
// them, which is the whole reason this mod exists.
//
// It is one network, compiled once, and it runs no script. Everything below is
// vanilla splitters with `output_priority` set at create time.
//
// # Why one priority butterfly is not the answer
//
// Setting output priority on every splitter of the plain butterfly DESTROYS the
// remainder rather than distributing it: two belts into a 4x4 come out
// [1, 0, 1, 0], because a priority splitter is a MERGE and a tree of merges is
// a concentrator, not a balancer. That is the fact the construction is built
// around rather than the one it fails on.
//
// # The construction
//
//	band 0, P rows:  head -> BALANCE -> CONCENTRATE -> one tap per rank
//	band 1:          PRIO balancer (q ports)  |  RESID balancer (M-q ports)
//
//   - BALANCE is the plain butterfly over P rows with every row an output, so
//     nothing loops back and nothing dead-ends. Every row leaves it carrying
//     exactly S/P.
//   - CONCENTRATE is the SAME schedule with every splitter's output priority on
//     the smaller y. Over a uniform input it sorts: the row of rank r carries
//     clamp(S - r, 0, 1), so ranks 0..q-1 hold min(S, q) between them and the
//     ranks after them hold the rest. `order` puts the line whose bit s is
//     clear on the smaller y of every stage-s pair, so the same half wins at
//     every stage and the winners meet each other at the next one; the rank of
//     a row is then the bit reversal of the line standing on it, which is what
//     sorterRanks computes and TestTheConcentratorSorts measures.
//   - the tap column sends ranks 0..q-1 to the PRIO balancer and ranks
//     q..M-1 to the RESID one. Ranks M and past get NO tap and dead-end, which
//     is correct rather than tolerated: a balancer with M outputs cannot
//     deliver more than M belts, and a priority splitter whose overflow side is
//     blocked backs up without disturbing its priority side.
//   - each tier balancer is a plain square butterfly, shape (t, t), so its
//     every spare port loops back and it has no dead end in it. That is what
//     makes each tier exact at every load and not only at saturation: a
//     dead-ended port re-routes flow asymmetrically under partial load, which
//     TestDeadEndedSpareOutputsAreNotExactUnderPartialLoad records against the
//     plain network.
//
// BALANCE is not an optimisation and cannot be dropped for q = 1 even though
// rank 0 carries min(S, 1) whatever the input looks like. It is what makes the
// network draw EQUALLY from its inputs: the concentrator's tree is deliberately
// asymmetric, and behind it the input a row happens to sit on would decide how
// much of it is taken when the balancer is full.
//
// # What it costs
//
// Two butterflies where there was one, plus the tier balancers, plus a linked
// belt pair per tap: about 2.3x the entities of the plain network at 4x4 and
// 2.5x at 8x8, and a recompile is linear in entities. agents/priority.md has
// the table and what a toggle costs in items.
// ---------------------------------------------------------------------------

// sorterRanks fills rankBuf with the concentration rank of each physical row at
// the concentrator's exit: rank 0 is the row that fills first.
//
// A line wins stage s when its bit s is clear, so after k stages the line's
// value is clamp(S - rev(line), 0, 1) where rev reverses its k bits -- the
// first stage decides the LAST bit of the rank. The row that line ends on is
// `order`'s last stage read backwards.
func sorterRanks(p int) {
	k := stages(p)
	if k == 0 {
		rankBuf[0] = 0
		return
	}
	last := order(p)[k-1]
	for r := 0; r < p; r++ {
		v, rev := last[r], 0
		for i := 0; i < k; i++ {
			rev = rev<<1 | (v>>uint(i))&1
		}
		rankBuf[r] = uint8(rev)
	}
}

// subAt lays one tier's balancer: a plain butterfly of shape (t, t) over
// NextPow2(t) rows at (cx, cy), with its spare ports looped back into its spare
// rows. Square because a tier's lines and its ports are the same set.
//
// entry and exit come back in subOp, packed as the entry op of port i followed
// by the exit op of port i, so that one [MaxPorts] buffer serves both and the
// caller can wire the taps and the visible interfaces without a second pass.
// Only the t real ports are reported; the loopbacks are wired here.
func subAt(ops []Op, t int, cx, cy int32) []Op {
	pp := NextPow2(t)
	fx := float64(cx) + 0.5
	fy := float64(cy) + 0.5
	for r := 0; r < pp; r++ {
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fx + colRecv, Y: fy + float64(r),
			Dir: East, Link: LinkOutput, Pair: -1})
		recvOp[r] = int32(len(ops) - 1)
		// Every tier line gets the lane stage. The concentrator's splitters are
		// the first in this mod that are not plain, nothing has measured
		// whether an output priority keeps a vanilla splitter's lane fidelity,
		// and one entity a line is what makes the question not matter.
		ops = append(ops, Op{Proto: ProtoLaneSplitter, X: fx + colLaneSplit,
			Y: fy + float64(r), Dir: East, Pair: -1})
	}
	ops, col := stagesAt(ops, pp, colFirstStage, cx, cy, PrioNone)
	for r := 0; r < pp; r++ {
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fx + float64(col), Y: fy + float64(r),
			Dir: East, Link: LinkInput, Pair: -1})
		sendOp[r] = int32(len(ops) - 1)
	}
	for i := t; i < pp; i++ {
		ops[sendOp[i]].Pair = recvOp[i]
	}
	for i := 0; i < t; i++ {
		subOp[2*i], subOp[2*i+1] = recvOp[i], sendOp[i]
	}
	return ops
}

// buildPrio is Build for an edge list with priority outputs in it. The header
// above is the construction; this is the wiring.
func buildPrio(ops []Op, edges []Edge, pt Ports, ox, oy int32) ([]Op, Ports, bool) {
	p, q := pt.P, pt.QOut
	mn := pt.M - q
	fx := float64(ox) + 0.5
	fy := float64(oy) + 0.5

	// --- band 0: the head, over the N real input rows. Nothing loops back into
	// this band: every tier balancer recirculates its own spare ports, so the
	// flow that reaches one has already lost or won the priority competition
	// and must not be offered it again.
	for r := 0; r < pt.N+pt.Loop; r++ {
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fx + colRecv, Y: fy + float64(r),
			Dir: East, Link: LinkOutput, Pair: -1})
		headOp[r] = int32(len(ops) - 1)
		ops = append(ops, Op{Proto: ProtoLaneSplitter, X: fx + colLaneSplit,
			Y: fy + float64(r), Dir: East, Pair: -1})
	}
	ops, col := stagesAt(ops, p, colFirstStage, ox, oy, PrioNone)
	ops, col = stagesAt(ops, p, col, ox, oy, PrioLeft)

	// --- the tap column. A rank past the last output port gets no tap at all.
	sorterRanks(p)
	taps := pt.M + pt.Loop
	for i := 0; i < taps; i++ {
		tapOp[i] = -1
	}
	for r := 0; r < p; r++ {
		rank := int(rankBuf[r])
		if rank >= taps {
			continue
		}
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: fx + float64(col), Y: fy + float64(r),
			Dir: East, Link: LinkInput, Pair: -1})
		tapOp[rank] = int32(len(ops) - 1)
	}

	// --- band 1: the two tiers, side by side under band 0. portSend[i] is the
	// op a visible output pairs with, numbered priority ports first.
	by := oy + int32(p)
	ops = subAt(ops, q, ox, by)
	for i := 0; i < q; i++ {
		ops[tapOp[i]].Pair = subOp[2*i]
		portSend[i] = subOp[2*i+1]
	}
	ops = subAt(ops, mn, ox+int32(Width(NextPow2(q))), by)
	for j := 0; j < mn; j++ {
		ops[tapOp[q+j]].Pair = subOp[2*j]
		portSend[q+j] = subOp[2*j+1]
	}
	// The ranks past the last output port are the plain network's spare ports
	// and they are wired the plain network's way: the first Loop of them feed
	// the head's spare rows and the rest dead-end. That is not tidiness. A
	// head row with nothing on it makes the network draw UNEVENLY from its
	// inputs once it saturates -- 7 belts into a 7->5 came out 0.625 four ways,
	// 0.75 twice and a whole belt once -- because the back-pressure the
	// concentrator sends back is shaped by the concentration tree, and a
	// butterfly whose outputs are blocked unevenly is not input-fair.
	// TestSaturatedInputsAreDrawnEqually is what holds it.
	for i := 0; i < pt.Loop; i++ {
		ops[tapOp[pt.M+i]].Pair = headOp[pt.N+i]
	}

	// --- the visible edge interfaces, in edge order, so that the k-th visible
	// end is still the k-th edge whatever port the plan gave it.
	nin, nprio, nnorm := 0, 0, 0
	for _, e := range edges {
		if e.Out {
			port := q + nnorm
			if e.Prio {
				port = nprio
				nprio++
			} else {
				nnorm++
			}
			ops = append(ops, Op{Proto: ProtoLinkedBelt, X: float64(e.TileX) + 0.5,
				Y: float64(e.TileY) + 0.5, Dir: e.Dir, Link: LinkOutput, Visible: true, Pair: -1})
			ops[portSend[port]].Pair = int32(len(ops) - 1)
			continue
		}
		ops = append(ops, Op{Proto: ProtoLinkedBelt, X: float64(e.TileX) + 0.5,
			Y: float64(e.TileY) + 0.5, Dir: e.Dir, Link: LinkInput, Visible: true,
			Pair: headOp[nin]})
		nin++
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

// ---------------------------------------------------------------------------
// How much a network can be given back
// ---------------------------------------------------------------------------

// Reinsertable is a LOWER BOUND on how many items the network for a shape can
// be given back.
//
// A teardown drains what is STANDING in a network and the flush hands it to
// whatever cluster succeeds it; what the successor cannot hold spills on the
// visible surface beside the cluster (carry.go, "A recompile is not a
// removal"). That is the right answer for a machine a player took apart and the
// wrong one for a flag they ticked, so the guest asks this before it moves a
// priority flag and refuses a change the successor could not swallow. A bound
// that were too generous would pass a toggle and spill the difference, which is
// the one outcome the guard exists to prevent, so it is deliberately under what
// the engine has been measured giving back.
//
// WHAT THE ENGINE GIVES BACK IS NOT THE TILE ARITHMETIC, and the gap is
// measured twice. A tile of belt holds eight item positions (0.25 of a tile per
// item along a lane, two lanes), a splitter is two tiles and a lane splitter is
// one, which makes a plain 2->2 96 positions and a plain 4->4 288. A JAMMED
// network of those two shapes drains 72 and 232 -- CLAUDE.md's M2 conservation
// check and the `hand` leg's first shrink -- which is 75.0% and 80.6% of the
// arithmetic. The shortfall is the splitter family: eight linked belts alone
// account for all but 8 of the 2->2's 72, so one splitter and two lane
// splitters gave back 8 positions between them where the arithmetic claims 32.
//
// SO THE DISCOUNT IS FLAT AND IT IS TWO THIRDS. Two measurements cannot fit a
// per-proto model -- read per tile they imply 2 positions a splitter-family tile
// on the 2->2 and 3.33 on the 4->4 -- and a model fitted to one of them would be
// a guess dressed as arithmetic on the other. Two thirds is under the lower of
// the two FRACTIONS, 75.0%, with eight points to spare.
//
// AND IT IS UNDER THE PESSIMISTIC READING ON SHAPES NOBODY HAS MEASURED, which
// is what carries it past the two plain rows: take the worse of the two --
// belts and linked belts exact at eight a tile, the splitter family at two --
// and the flat two thirds is under it on every shape this planner builds, by
// 2.48 points at the tightest (1->4 at q=2). That matters because a priority network is MORE
// splitter and not less: 40% of a priority 2->2's tiles are splitter family
// against 33% of the plain one's, so a bound that leaned on the plain shapes'
// composition would lean the wrong way. TestTheBoundHoldsOnShapesNobodyMeasured.
//
// BEING TOO LOW COSTS A REFUSAL A PLAYER DID NOT NEED, which is the side to be
// wrong on -- the other side is items on the ground -- and the refusal names the
// remedy, which is to let the balancer empty.
//
// IT BUILDS THE SHAPE RATHER THAN COUNTING IT. The op count of a priority
// network is the head, two butterflies, a tap per rank and two tier balancers,
// and an arithmetic version of that would be a second copy of the layout that
// could drift from the one buildPrio lays. Positions are a function of the
// SHAPE alone -- where an edge sits decides where a visible interface goes and
// never how many there are -- so a synthetic edge list of N inputs and M
// outputs with QOut of them flagged builds the same entity count the real one
// does.
//
// It is for a keypress and not for a compile: it allocates on its first call at
// a size no earlier call reached, it clobbers this package's working buffers the
// way Build does, and it costs a whole Build. The guest calls it from
// setPartPriority, which is an outermost dispatch, and from nowhere else.
//
// A shape with no inputs or no outputs, and one that does not fit, take nothing
// back: neither has a network.
func Reinsertable(pt Ports) int {
	if pt.N <= 0 || pt.M <= 0 {
		return 0
	}
	roomEdges = roomEdges[:0]
	for i := 0; i < pt.N; i++ {
		roomEdges = append(roomEdges, Edge{Dir: East})
	}
	for i := 0; i < pt.M; i++ {
		roomEdges = append(roomEdges, Edge{Dir: East, Out: true, Prio: i < pt.QOut})
	}
	ops, _, ok := Build(roomOps[:0], roomEdges, 0, 0)
	roomOps = ops
	if !ok {
		return 0
	}
	return reinsertableOf(ops)
}

// roomEdges and roomOps are Reinsertable's own, so that a capacity question
// cannot overwrite an op list a caller is still holding. Slices rather than
// arrays for the reason the buffer block above gives: an array of MaxPorts ops
// would be tens of kilobytes of GLOBALS, which the conservative collector
// re-scans at every paced step, where a slice header is three words and its
// backing array is ordinary heap.
var (
	roomEdges []Edge
	roomOps   []Op
)

// reinsertableOf is the items an op list can be given back.
//
// The tile arithmetic first: eight positions a tile, a splitter over two tiles
// and everything else over one. A LANE SPLITTER IS ONE TILE, which is what
// `tilesOf` in plan_test.go has always said and what the layout proves -- the
// ops lay one per row at colLaneSplit, so a two-tile one would collide with the
// row below. The VISIBLE interfaces count, because a teardown drains the
// cluster's box as well as the slot.
//
// Then the two thirds, for the reason Reinsertable's header gives at length.
func reinsertableOf(ops []Op) int {
	const perTile = 8
	tiles := 0
	for i := range ops {
		if ops[i].Proto == ProtoSplitter {
			tiles += 2
			continue
		}
		tiles++
	}
	return tiles * perTile * 2 / 3
}
