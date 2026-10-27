package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

// These are world identities, never transient Lua handles. A replacement or
// clone at the same position is not the entity the old network owns.
type netInterface struct {
	x, y float64
	unit uint64
}

var (
	freeInterfaces [][]netInterface
	interfaceEnts  []fkapi.Object
	checkEnts      []fkapi.Object
	checkUnits     []uint64
)

func takeInterfaces() []netInterface {
	if n := len(freeInterfaces); n != 0 {
		refs := freeInterfaces[n-1]
		freeInterfaces[n-1] = nil
		freeInterfaces = freeInterfaces[:n-1]
		return refs[:0]
	}
	return nil
}

func recycleInterfaces(refs []netInterface) {
	if cap(refs) != 0 {
		freeInterfaces = append(freeInterfaces, refs[:0])
	}
}

func rememberInterfaces(ops []plan.Op, ents []fkapi.Object) ([]netInterface, bool) {
	refs := takeInterfaces()
	for i := range ops {
		if !ops[i].Visible {
			continue
		}
		u, err := (fkapi.LuaEntity{Object: ents[i]}).UnitNumber()
		if err != nil || u == nil || *u == 0 {
			recycleInterfaces(refs)
			return nil, false
		}
		refs = append(refs, netInterface{x: ops[i].X, y: ops[i].Y, unit: *u})
	}
	return refs, true
}

func resolveInterface(s fkapi.LuaSurface, ref netInterface) (fkapi.LuaEntity, bool) {
	name := fkapi.OfString(nameLinkedBelt)
	pos := fkapi.MapPosition{X: ref.x, Y: ref.y}
	ents, err := s.FindEntitiesFilteredInto(interfaceEnts,
		fkapi.EntitySearchFilters{Name: &name, Position: &pos})
	interfaceEnts = ents
	if err == nil {
		for _, o := range ents {
			e := fkapi.LuaEntity{Object: o}
			if u, err := e.UnitNumber(); err == nil && u != nil && *u == ref.unit {
				return e, true
			}
		}
	}
	return fkapi.LuaEntity{}, false
}

func ownsInterface(si uint32, e fkapi.LuaEntity) bool {
	u, err := e.UnitNumber()
	if err != nil || u == nil {
		return false
	}
	// Include non-root keys: a deferred or refused merge can still own a net.
	for r := uint32(1); r < uint32(len(parent)); r++ {
		ni, has := nets[r]
		if !has || ni.surf != si {
			continue
		}
		for _, ref := range ni.interfaces {
			if ref.unit == *u {
				return true
			}
		}
	}
	return false
}

func networkRoots(dst []uint32, si uint32) []uint32 {
	dst = dst[:0]
	for r := uint32(1); r < uint32(len(parent)); r++ {
		if ni, has := nets[r]; has && (si == 0 || ni.surf == si) {
			dst = append(dst, r)
		}
	}
	return dst
}

// Only an entirely readable, empty orphan may be deleted or reused.
func emptyOrphanSlot(s fkapi.LuaSurface, x, y int32) bool {
	setSearchBox(x, y, x+slotW-1, y+slotH-1)
	for _, name := range sweepNames {
		nameFilter = fkapi.OfString(name)
		ents, err := s.FindEntitiesFiltered(findByName)
		if err != nil {
			return false
		}
		// Exposed lines do not prove a splitter's internal buffer is empty.
		if len(ents) > 0 && (name == nameSplitter || name == nameLaneSplit) {
			return false
		}
		for _, o := range ents {
			e := fkapi.LuaEntity{Object: o}
			n, err := e.GetMaxTransportLineIndex()
			if err != nil || n == 0 || n > maxLinesToProbe {
				return false
			}
			for i := uint32(1); i <= n; i++ {
				line, err := e.GetTransportLine(i)
				if err != nil {
					return false
				}
				count, err := (fkapi.LuaTransportLine{Object: line}).GetItemCount(nil)
				if err != nil || count != 0 {
					return false
				}
			}
		}
	}
	return true
}

// networkIntact checks the actual construction, not merely the belts around it.
// It is deliberately an audit/adoption cost, not an ordinary fingerprint cost.
// Both ends of every pair are checked, including internal hidden jumpers.
func networkIntact(ni netInfo, edges []plan.Edge) bool {
	if ni.slot == 0 || hiddenIdx == 0 {
		return false
	}
	vis, ok := surfaceByIndex(ni.surf)
	if !ok {
		return false
	}
	hid, ok := surfaceByIndex(hiddenIdx)
	if !ok {
		return false
	}
	ox, oy := slotOrigin(ni.slot)
	ops, _, fits := plan.Build(opBuf, edges, ox, oy)
	opBuf = ops
	if !fits || len(ops) == 0 {
		return false
	}
	checkEnts, checkUnits = checkEnts[:0], checkUnits[:0]
	visible := 0
	for _, op := range ops {
		s := hid
		if op.Visible {
			s = vis
			visible++
		}
		name := fkapi.OfString(protoName(op.Proto))
		pos := fkapi.MapPosition{X: op.X, Y: op.Y}
		ents, err := s.FindEntitiesFilteredInto(interfaceEnts,
			fkapi.EntitySearchFilters{Name: &name, Position: &pos})
		interfaceEnts = ents
		if err != nil {
			return false
		}
		var found fkapi.Object
		var unit uint64
		for _, o := range ents {
			e := fkapi.LuaEntity{Object: o}
			p, err := e.Position()
			if err != nil || p.X != op.X || p.Y != op.Y {
				continue
			}
			d, err := e.Direction()
			if err != nil || d != op.Dir {
				continue
			}
			if op.Link != plan.LinkNone {
				if is, err := e.LinkedBeltTypeIs(linkName(op.Link)); err != nil || !is {
					continue
				}
			}
			f, ok := forceOf(o)
			if !ok || f != ni.force {
				continue
			}
			u, err := e.UnitNumber()
			if err != nil || u == nil || *u == 0 || unit != 0 {
				return false
			}
			found, unit = o, *u
		}
		if unit == 0 {
			return false
		}
		if op.Visible {
			owned := false
			for _, ref := range ni.interfaces {
				if ref.unit == unit && ref.x == op.X && ref.y == op.Y {
					owned = true
					break
				}
			}
			if !owned {
				return false
			}
		}
		checkEnts = append(checkEnts, found)
		checkUnits = append(checkUnits, unit)
	}
	if visible != len(ni.interfaces) {
		return false
	}
	for i, op := range ops {
		if op.Link != plan.LinkInput || op.Pair < 0 {
			continue
		}
		for side := 0; side < 2; side++ {
			a, b := i, int(op.Pair)
			if side != 0 {
				a, b = b, a
			}
			nb, err := (fkapi.LuaEntity{Object: checkEnts[a]}).LinkedBeltNeighbour()
			if err != nil || nb == nil || !nb.Valid() {
				return false
			}
			u, err := (fkapi.LuaEntity{Object: *nb}).UnitNumber()
			if err != nil || u == nil || *u != checkUnits[b] {
				return false
			}
		}
	}
	return true
}
