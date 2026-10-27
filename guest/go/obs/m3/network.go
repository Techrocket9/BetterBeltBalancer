package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
)

const networkSurface = "bbb-m3-network"
const networkLink = "bbb-linked-belt"

func interfaceAt(s fkapi.LuaSurface, x, y int) fkapi.LuaEntity {
	o, ok := harness.FindAt(s, x, y, networkLink, "")
	if !ok {
		return fkapi.LuaEntity{}
	}
	return fkapi.LuaEntity{Object: o}
}

func interfaceUnit(s fkapi.LuaSurface, x, y int) uint64 {
	e := interfaceAt(s, x, y)
	if !e.Object.Valid() {
		return 0
	}
	u, err := e.UnitNumber()
	if err != nil || u == nil {
		return 0
	}
	return *u
}

func pairedAt(s fkapi.LuaSurface, x, y int) bool {
	e := interfaceAt(s, x, y)
	if !e.Object.Valid() {
		return false
	}
	nb, err := e.LinkedBeltNeighbour()
	if err != nil || nb == nil || !nb.Valid() {
		return false
	}
	back, err := (fkapi.LuaEntity{Object: *nb}).LinkedBeltNeighbour()
	if err != nil || back == nil || !back.Valid() {
		return false
	}
	u, err := (fkapi.LuaEntity{Object: *back}).UnitNumber()
	return err == nil && u != nil && *u == interfaceUnit(s, x, y)
}

// Run before the ordinary rigs so the dummy really owns the lowest node id.
// The whole probe is synchronous: identities and carry cannot be hidden by a
// tick of belt flow or an unrelated deferred repair.
func networkRegression() {
	s := makeSurface(networkSurface, 24)
	dummy := put(s, part, 15, 15, nil, "")
	// A U surrounds an independent, same-force two-part balancer.
	for x := 0; x <= 8; x++ {
		put(s, part, x, 0, nil, "")
	}
	for y := 1; y <= 8; y++ {
		put(s, part, 0, y, nil, "")
		put(s, part, 8, y, nil, "")
	}
	put(s, belt, -1, 0, &dirE, "")
	put(s, belt, 9, 0, &dirE, "")
	put(s, part, 3, 4, nil, "")
	put(s, part, 4, 4, nil, "")
	put(s, belt, 2, 4, &dirE, "")
	put(s, belt, 5, 4, &dirE, "")
	harness.Audit(s, 20, 20)
	outer, inner := interfaceUnit(s, 0, 0), interfaceUnit(s, 3, 4)
	put(s, part, 0, 9, nil, "")
	harness.Audit(s, 20, 20)
	out.Open("network case=extension before=").U(outer).
		S(" after=").U(interfaceUnit(s, 0, 0)).End()

	harness.Destroy(dummy, true)
	harness.Audit(s, 20, 20)
	extension := put(s, part, 0, 10, nil, "")
	harness.Audit(s, 20, 20)
	out.Open("network case=lower-root before=").U(outer).
		S(" after=").U(interfaceUnit(s, 0, 0)).End()

	harness.Destroy(extension, true)
	harness.Audit(s, 20, 20)
	out.Open("network case=nested before=").U(inner).
		S(" after=").U(interfaceUnit(s, 3, 4)).End()
	// Reserve the just-freed lowest id for the refused bridge below. Neither
	// predecessor may inherit this id while the intervening merge rig is built.
	reserved := put(s, part, 15, 15, nil, "")

	// Refusal must keep the old fingerprint, even across repeated audits, so
	// reverting the extra belt leaves the original network standing.
	out.Open("network refusal-begin").End()
	dirS := fkapi.DefinesDirectionSouth()
	extra := put(s, belt, 3, 5, &dirS, "")
	harness.Audit(s, 20, 20)
	harness.Audit(s, 20, 20)
	during := interfaceUnit(s, 3, 4)
	harness.Destroy(extra, true)
	harness.Audit(s, 20, 20)
	out.Open("network case=refusal-audit-revert before=").U(inner).
		S(" during=").U(during).S(" after=").U(interfaceUnit(s, 3, 4)).
		S(" paired=").B(pairedAt(s, 3, 4)).End()
	out.Open("network refusal-end").End()

	// Neither silent damage changes a single belt in the edge fingerprint.
	harness.Destroy(interfaceAt(s, 3, 4).Object, false)
	harness.Audit(s, 20, 20)
	out.Open("network case=missing paired=").B(pairedAt(s, 3, 4)).End()
	if err := interfaceAt(s, 4, 4).DisconnectLinkedBelts(); err != nil {
		harness.Fatal("disconnect interface", "API call failed")
	}
	harness.Audit(s, 20, 20)
	out.Open("network case=disconnected paired=").B(pairedAt(s, 4, 4)).End()
	if nb, err := interfaceAt(s, 3, 4).LinkedBeltNeighbour(); err == nil && nb != nil {
		harness.Destroy(*nb, false)
	} else {
		harness.Fatal("destroy hidden partner", "no partner")
	}
	harness.Audit(s, 20, 20)
	out.Open("network case=hidden-partner paired=").B(pairedAt(s, 3, 4)).End()
	// Valid, reciprocal connections can still belong to the WRONG network.
	a, b := interfaceAt(s, 0, 0), interfaceAt(s, 3, 4)
	an, ae := a.LinkedBeltNeighbour()
	bn, be := b.LinkedBeltNeighbour()
	outer, inner = interfaceUnit(s, 0, 0), interfaceUnit(s, 3, 4)
	if ae != nil || be != nil || an == nil || bn == nil {
		harness.Fatal("cross-wire networks", "missing partner")
	} else if a.DisconnectLinkedBelts() != nil || b.DisconnectLinkedBelts() != nil ||
		a.ConnectLinkedBelts(bn) != nil || b.ConnectLinkedBelts(an) != nil {
		harness.Fatal("cross-wire networks", "API call failed")
	}
	harness.Audit(s, 20, 20)
	out.Open("network case=wrong-partner outer-before=").U(outer).
		S(" outer-after=").U(interfaceUnit(s, 0, 0)).
		S(" inner-before=").U(inner).S(" inner-after=").U(interfaceUnit(s, 3, 4)).
		S(" paired=").B(pairedAt(s, 0, 0) && pairedAt(s, 3, 4)).End()
	inner = interfaceUnit(s, 3, 4)
	harness.Audit(s, 20, 20)
	out.Open("network case=idempotent before=").U(inner).
		S(" after=").U(interfaceUnit(s, 3, 4)).End()

	// Seed both predecessors, then merge and split. Items must stay inside
	// networks; retaining just one predecessor would strand the other's pool.
	for _, p := range []harness.XY{{X: 0, Y: 0}, {X: 3, Y: 4}} {
		e := interfaceAt(s, p.X, p.Y)
		line, err := e.GetTransportLine(1)
		if err != nil {
			harness.Fatal("seed network", "no line")
			continue
		}
		items := fkapi.OfMap(fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString("iron-plate")},
			fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(1)})
		if ok, err := (fkapi.LuaTransportLine{Object: line}).InsertAtBack(items, nil); err != nil || !ok {
			harness.Fatal("seed network", "insert refused")
		}
	}
	outer, inner = interfaceUnit(s, 0, 0), interfaceUnit(s, 3, 4)
	put(s, part, 3, 3, nil, "")
	bridge := put(s, part, 3, 2, nil, "")
	put(s, part, 3, 1, nil, "")
	harness.Audit(s, 20, 20)
	out.Open("network case=merge outer-before=").U(outer).
		S(" outer-after=").U(interfaceUnit(s, 0, 0)).
		S(" inner-before=").U(inner).S(" inner-after=").U(interfaceUnit(s, 3, 4)).
		S(" hidden-items=").I(countOn(hidden, hiddenBox())).
		S(" visible-items=").I(countSurface(s, harness.Box(-10, -10, 24, 24), true)).End()
	harness.Destroy(bridge, true)
	harness.Audit(s, 20, 20)
	out.Open("network case=split outer=").B(pairedAt(s, 0, 0)).
		S(" inner=").B(pairedAt(s, 3, 4)).
		S(" hidden-items=").I(countOn(hidden, hiddenBox())).
		S(" visible-items=").I(countSurface(s, harness.Box(-10, -10, 24, 24), true)).End()

	// The gap is between the two surviving components. Two inputs on its tile
	// refuse the merge; deleting the low-id bridge must preserve BOTH originals.
	harness.Destroy(reserved, true)
	harness.Audit(s, 20, 20)
	put(s, belt, 2, 2, &dirE, "")
	put(s, belt, 4, 2, &dirW, "")
	harness.Audit(s, 20, 20)
	outer, inner = interfaceUnit(s, 0, 0), interfaceUnit(s, 3, 4)
	out.Open("network refusal-begin").End()
	bridge = put(s, part, 3, 2, nil, "")
	harness.Audit(s, 20, 20)
	harness.Audit(s, 20, 20)
	outerDuring, innerDuring := interfaceUnit(s, 0, 0), interfaceUnit(s, 3, 4)
	harness.Destroy(bridge, true)
	harness.Audit(s, 20, 20)
	out.Open("network case=refused-bridge outer-before=").U(outer).
		S(" outer-during=").U(outerDuring).S(" outer-after=").U(interfaceUnit(s, 0, 0)).
		S(" inner-before=").U(inner).S(" inner-during=").U(innerDuring).
		S(" inner-after=").U(interfaceUnit(s, 3, 4)).
		S(" paired=").B(pairedAt(s, 0, 0) && pairedAt(s, 3, 4)).End()
	out.Open("network refusal-end").End()
	// delete_surface during on_init waits until the first tick. Empty the
	// registry and networks now, before the ordinary M3 fixtures are built.
	out.Open("network cleanup-begin").End()
	box := harness.Box(-10, -10, 24, 24)
	for _, o := range harness.EntitiesIn(s, box, part) {
		harness.Destroy(o, true)
	}
	harness.Audit(s, 20, 20)
	// The teardown can spill the probe's seeded items. Remove those and the
	// external belts too, then flush any neighbour events before deletion.
	for _, o := range harness.EntitiesIn(s, box, "") {
		harness.Destroy(o, true)
	}
	harness.Audit(s, 20, 20)
	out.Open("network case=cleanup visible=").I(int64(len(harness.EntitiesIn(s, box, "")))).
		S(" hidden=").I(int64(len(harness.EntitiesIn(harness.Surface(hidden), hiddenBox(), "")))).End()
	deleteSurface(networkSurface)
}
