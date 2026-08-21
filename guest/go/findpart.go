package main

// ASKING THE WORLD FOR ONE ENTITY ON ONE TILE, BY NAME, AT ANY QUALITY.
//
// `LuaSurface.find_entity` looks like the right call for that question and is a
// trap: it takes an `EntityWithQualityID`, and the pinned 2.0.77 runtime API
// says of a bare name that "Normal quality will be used" -- so an entity
// standing at any OTHER quality is INVISIBLE to it. Measured on 2.0.77 against
// a real uncommon entity, with the probe table recorded in legacy.go, where the
// migration's build path sprang the trap first (2026-08-20, the mig suite's
// fidelity rig): `find_entity(name, p)` is nil where
// `find_entities_filtered{name = ..., position = p}` returns the entity.
//
// This file is that fix stated once, for the four sites that were left after
// legacy.go's: skin.go's restyle, limit.go's forceOfCluster and revertOne, and
// fastreplace.go's reapFastReplaced -- and legacy.go's own build path now goes
// through it too, so a fifth caller asks the same code or does not ask at all.
// The question every one of them is actually asking has no quality in it: is a
// thing of OURS with this name standing on this tile. A one-tile area query
// with a name filter is that question exactly -- the tenth-of-a-tile inset
// keeps the neighbouring tiles out (the same inset, for the same measured
// reason, as compile.go's setSearchBox), the name filter is applied by the
// engine in C++, and at most one entity of one name can stand on a tile, so
// the first hit is the answer.
//
// DEDICATED BUFFERS, not compile.go's `findByName`. One caller
// (`reapFastReplaced`) runs on the EVENT path, and the shared
// `searchArea`/`nameFilter` scratch belongs to the flush; giving this file its
// own filter struct costs a few static bytes and removes the aliasing question
// instead of answering it.
//
// WHAT IT COSTS against the `find_entity` it replaces: the generated binding
// for `find_entities_filtered` returns `make([]Object, n)`, so a HIT allocates
// a one-element slice where `find_entity` allocated one boxed Object, and a
// MISS allocates nothing on either side. The paths this sits on are refusals,
// fast-replace hits, picture changes and the migration's build swaps -- none
// is per-tick -- and the `mar` suite's slopes are the gate that says what it
// really moved (see CLAUDE.md, the quality pass).

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

var (
	otArea    fkapi.BoundingBox
	otName    fkapi.Value
	otFilters = fkapi.EntitySearchFilters{Area: &otArea, Name: &otName}
)

// findOnTile returns the entity of that name standing on tile (x, y) of s,
// whatever its quality. `found` and `err` are SEPARATE on purpose: one caller
// (reapFastReplaced) unregisters a part on the strength of a miss, and a query
// that FAILED is not a miss -- collapsing the two would turn a host error into
// a registry edit.
func findOnTile(s fkapi.LuaSurface, name string, x, y int32) (o fkapi.Object, found bool, err error) {
	otArea.LeftTop.X, otArea.LeftTop.Y = float64(x)+0.1, float64(y)+0.1
	otArea.RightBottom.X, otArea.RightBottom.Y = float64(x)+0.9, float64(y)+0.9
	otName = fkapi.OfString(name)
	ents, err := s.FindEntitiesFiltered(otFilters)
	if err != nil || len(ents) == 0 {
		return fkapi.Object{}, false, err
	}
	return ents[0], true, nil
}
