package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

const topologySurface = "bbb-m3-topology"

var topologySouth uint32
var topologyBefore [6]uint64
var topologyUnits [4]uint64
var topologyItems int64
var topologyCase string
var topologyRow int
var topologySlots [6]fkapi.BoundingBox

func topologyStats() (stats [6]uint64) {
	v, st := fkapi.RemoteCall("better-belt-balancer", "work-stats")
	if st != fkapi.StatusOK || v.Tag != fkapi.TagArray || len(v.Array) != len(stats) {
		harness.Fatal("topology work-stats", "expected six counters")
		return
	}
	for i, n := range v.Array {
		if n.Tag != fkapi.TagNumber || n.Number < 0 {
			harness.Fatal("topology work-stats", "invalid counter")
			return
		}
		stats[i] = uint64(n.Number)
	}
	return
}

// Mark placement blocks so Python can verify actual node/root IDs, rather than
// assuming the free list hands IDs out in any particular order.
func topologyLine(s fkapi.LuaSurface, label string, row, lo, hi int) {
	out.Open("topology setup=").S(label).S(" begin").End()
	for x := lo; x <= hi; x++ {
		put(s, part, x, row, nil, "")
	}
	out.Open("topology setup=").S(label).S(" end").End()
}

func topologyPorts(s fkapi.LuaSurface, row int) {
	put(s, belt, -100, row-1, &topologySouth, "")
	put(s, belt, -99, row+1, &topologySouth, "")
}

func topologySetup() {
	topologySouth = fkapi.DefinesDirectionSouth()
	s := harness.Flat{Name: topologySurface, MapWidth: 512, MapHeight: 256,
		ChunkRadius: 7, X0: -180, Y0: -5, X1: 180, Y1: 100, Tile: "grass-1"}.Make()
	// Absorb IDs freed by the synchronous init probe before the root fixtures.
	// Python still checks the observed root arrangements below.
	topologyLine(s, "reserve", 0, -150, 149)
	topologyLine(s, "nonroot-low", 10, 149, 149)
	topologyLine(s, "nonroot-chain", 10, -150, 147)
	topologyPorts(s, 10)
	topologyLine(s, "root-chain", 20, -150, 149)
	topologyPorts(s, 20)
	topologyLine(s, "zero-left", 40, -150, -1)
	topologyLine(s, "zero-right", 40, 1, 149)
	topologyLine(s, "active-left", 50, -150, -1)
	topologyPorts(s, 50)
	topologyLine(s, "active-right", 50, 1, 149)
	topologyLine(s, "lower-right", 60, 1, 149)
	topologyLine(s, "lower-left", 60, -150, -1)
	topologyPorts(s, 60)
	// Retire this singleton only after all earlier reuse probes have finished.
	// Its ID must be below BOTH predecessors, not merely below the incumbent.
	topologyLine(s, "rollback-dummy", 77, 170, 170)
	topologyLine(s, "rollback-left", 70, -150, -1)
	topologyPorts(s, 70)
	topologyLine(s, "rollback-right", 70, 1, 149)
	topologyLine(s, "changed-left", 80, -150, -1)
	topologyPorts(s, 80)
	topologyLine(s, "changed-right", 80, 1, 149)
}

func topologyIdentity(s fkapi.LuaSurface, row int) (units [4]uint64, paired bool) {
	paired = true
	for i, x := range []int{-100, -99} {
		units[i*2] = interfaceUnit(s, x, row)
		e := interfaceAt(s, x, row)
		if !e.Object.Valid() {
			paired = false
			continue
		}
		nb, err := e.LinkedBeltNeighbour()
		if err != nil || nb == nil || !nb.Valid() {
			paired = false
			continue
		}
		u, err := (fkapi.LuaEntity{Object: *nb}).UnitNumber()
		if err == nil && u != nil {
			units[i*2+1] = *u
		}
		paired = paired && pairedAt(s, x, row)
	}
	return
}

func topologySlot(partner fkapi.Object) fkapi.BoundingBox {
	pos, err := (fkapi.LuaEntity{Object: partner}).Position()
	if err != nil {
		harness.Fatal("topology slot", "missing partner position")
		return fkapi.BoundingBox{}
	}
	x := int(pos.X) / plan.SlotWidth * plan.SlotWidth
	y := int(pos.Y) / plan.SlotHeight * plan.SlotHeight
	return harness.Box(float64(x)+0.01, float64(y)+0.01,
		float64(x+plan.SlotWidth)-0.01, float64(y+plan.SlotHeight)-0.01)
}

func topologySeed() {
	s := harness.Surface(topologySurface)
	for i, row := range []int{10, 20, 50, 60, 70, 80} {
		// Seed the hidden output link, bypassing splitter-internal buffers that
		// transport-line item counts cannot observe between engine ticks.
		e := interfaceAt(s, -99, row)
		nb, err := e.LinkedBeltNeighbour()
		if err != nil || nb == nil {
			harness.Fatal("topology seed", "missing hidden partner")
			continue
		}
		topologySlots[i] = topologySlot(*nb)
		line, err := (fkapi.LuaEntity{Object: *nb}).GetTransportLine(1)
		items := fkapi.OfMap(fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString("copper-plate")},
			fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(1)})
		if err != nil {
			harness.Fatal("topology seed", "missing line")
		} else if ok, err := (fkapi.LuaTransportLine{Object: line}).InsertAtBack(items, nil); err != nil || !ok {
			harness.Fatal("topology seed", "insert refused")
		}
	}
}

func topologyCount(row int) int64 {
	total := countOn(topologySurface, harness.Box(-180, float64(row-3), 180, float64(row+4)))
	for i, r := range []int{10, 20, 50, 60, 70, 80} {
		if row == r {
			if row == 80 {
				// A changed port must replace the network. Follow its actual
				// output partner, not an assumed allocator reuse of the old slot.
				e := interfaceAt(harness.Surface(topologySurface), -99, row)
				if e.Object.Valid() {
					if nb, err := e.LinkedBeltNeighbour(); err == nil && nb != nil && nb.Valid() {
						topologySlots[i] = topologySlot(*nb)
					}
				}
			}
			total += countOn(hidden, topologySlots[i])
		}
	}
	return total
}

func topologyBegin(label string, row int) {
	topologyCase, topologyRow = label, row
	topologyBefore = topologyStats()
	topologyUnits, _ = topologyIdentity(harness.Surface(topologySurface), row)
	topologyItems = topologyCount(row)
	out.Open("topology case=").S(label).S(" begin").End()
}

func topologySample() {
	s := harness.Surface(topologySurface)
	stats := topologyStats()
	units, paired := topologyIdentity(s, topologyRow)
	if topologyCase == "changed-port" {
		paired = paired && pairedAt(s, 0, topologyRow)
	}
	out.Open("topology case=").S(topologyCase).S(" work=[")
	for i := range stats {
		if i > 0 {
			out.S(" ")
		}
		out.I(int64(stats[i]) - int64(topologyBefore[i]))
	}
	out.S("] before=[")
	for i := range units {
		if i > 0 {
			out.S(" ")
		}
		out.U(topologyUnits[i])
	}
	out.S("] after=[")
	for i := range units {
		if i > 0 {
			out.S(" ")
		}
		out.U(units[i])
	}
	out.S("] paired=").B(paired).S(" items-before=").I(topologyItems).
		S(" items-after=").I(topologyCount(topologyRow)).
		S(" parts=").I(int64(len(harness.EntitiesIn(s,
		harness.Box(-180, float64(topologyRow-3), 180, float64(topologyRow+4)), part)))).
		S(" interfaces=").I(int64(len(harness.EntitiesIn(s,
		harness.Box(-180, float64(topologyRow-3), 180, float64(topologyRow+4)), networkLink)))).End()
}

func topologyRemove(x, row int) {
	s := harness.Surface(topologySurface)
	if p, ok := harness.FindAt(s, x, row, part, ""); ok {
		harness.Destroy(p, true)
	} else {
		harness.Fatal("topology remove", "missing part")
	}
}

func topologyTransient(x, row int) {
	b := put(harness.Surface(topologySurface), belt, x, row-1, &topologySouth, "")
	harness.Destroy(b, true)
}

// Every sample is two ticks after its edit. No audit or synchronous flush is
// allowed here: both would invalidate the very cached classifications we test.
func topologyTick(t uint64) {
	sample := false
	switch t {
	case 2:
		topologySetup()
	case 4:
		topologySeed()
	case 5:
		topologyBegin("descendant-merge", 10)
		put(harness.Surface(topologySurface), part, 148, 10, nil, "")
	case 8:
		topologyBegin("leaf-nonroot", 10)
		topologyRemove(-150, 10)
	case 11:
		topologyBegin("reuse", 10)
		put(harness.Surface(topologySurface), part, 170, 10, nil, "")
	case 14:
		topologyBegin("leaf-root", 20)
		topologyRemove(-150, 20)
	case 17:
		topologyBegin("zero-merge", 40)
		put(harness.Surface(topologySurface), part, 0, 40, nil, "")
	case 20:
		topologyBegin("active-merge", 50)
		put(harness.Surface(topologySurface), part, 0, 50, nil, "")
	case 25:
		topologyBegin("lower-root-dirty-merge", 60)
		topologyTransient(-120, 60)
		topologyTransient(120, 60)
		put(harness.Surface(topologySurface), part, 0, 60, nil, "")
	case 32:
		topologyBegin("dirty-leaves", 20)
		topologyTransient(50, 20)
		topologyTransient(100, 20)
		topologyRemove(149, 20)
		topologyRemove(-149, 20)
	case 35:
		topologyBegin("descendants-touch", 10)
		put(harness.Surface(topologySurface), part, -120, 11, nil, "")
	case 40:
		topologyRemove(170, 77)
	case 43:
		// These belts touch neither predecessor orthogonally. Let their dirty
		// neighborhoods settle before testing the cached, one-incumbent merge.
		s := harness.Surface(topologySurface)
		north := fkapi.DefinesDirectionNorth()
		put(s, belt, 0, 69, &topologySouth, "")
		put(s, belt, 0, 71, &north, "")
	case 46:
		out.Open("topology refusal-begin").End()
		topologyBegin("refused-bridge", 70)
		put(harness.Surface(topologySurface), part, 0, 70, nil, "")
	case 49:
		topologyBegin("refused-bridge-remove", 70)
		topologyRemove(0, 70)
	case 51:
		topologySample()
		out.Open("topology refusal-end").End()
	case 55:
		// Queue the predecessor neighborhoods now, so the extra input exists
		// but both snapshots have settled before the bridge is placed.
		put(harness.Surface(topologySurface), belt, 0, 79, &topologySouth, "")
	case 58:
		topologyBegin("changed-port", 80)
		put(harness.Surface(topologySurface), part, 0, 80, nil, "")
	case 65:
		// Independent fresh metadata oracle, AFTER every performance sample and
		// after the refused bridge's removal has settled. Never force a flush
		// with an audit inside the measured edit windows above.
		out.Open("topology oracle-begin").End()
		harness.Audit(harness.Surface(topologySurface), 175, 75)
		out.Open("topology oracle-end").End()
	case 7, 10, 13, 16, 19, 22, 27, 34, 37, 48, 60:
		sample = true
	case 70:
		s := harness.Surface(topologySurface)
		for _, p := range harness.EntitiesIn(s, harness.Box(-180, -5, 180, 100), part) {
			harness.Destroy(p, true)
		}
	case 73:
		s := harness.Surface(topologySurface)
		for _, e := range harness.EntitiesIn(s, harness.Box(-180, -5, 180, 100), "") {
			harness.Destroy(e, true)
		}
	case 76:
		var hiddenEntities int
		for _, box := range topologySlots {
			hiddenEntities += len(harness.EntitiesIn(harness.Surface(hidden), box, ""))
		}
		out.Open("topology case=cleanup visible=").I(int64(len(harness.EntitiesIn(
			harness.Surface(topologySurface), harness.Box(-180, -5, 180, 100), "")))).
			S(" hidden=").I(int64(hiddenEntities)).End()
		deleteSurface(topologySurface)
	}
	if sample {
		topologySample()
	}
}
