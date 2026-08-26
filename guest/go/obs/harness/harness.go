// Package harness is the shared kit every ported test observer builds its world
// out of: a flat scratch surface, tile-centred placement, tile lookups, chest
// totals, the audit marker and a tick schedule.
//
// # Why it exists
//
// The estate's test mods were fourteen hand-written control.lua files and every
// one of them opened with the same two hundred lines: a MapGenSettings with
// water off, cliffs off and one autoplace tile; a chunk request and a forced
// generate; a sweep that destroys everything in a box; a decorative purge; and a
// nested loop that paves the box with grass. Then `P(x, y)` for a tile centre,
// `put()` for a create_entity that errors on nil, a chest-total helper, and a
// `bbb-audit` marker placed with `raise_built = true`. Six-way duplication,
// measured across the estate before this package was written; the two pilots
// share every item on that list.
//
// So the port is not fourteen transcriptions. It is this package plus fourteen
// thin `main`s, and what a suite's own file holds afterwards is its RIGS and its
// LOG LINES -- which is all a test mod ever meant to say.
//
// # What an observer is, and what it is not
//
// An observer ASSERTS NOTHING. The mod under test writes `[BBB] ` lines and
// `test/assert-*.py` decides; an observer that computed the expected answer
// would be a second implementation of the thing under test. What it does is
// build a world, drive it on a schedule, and report what it SEES -- chest
// totals, what stands on a tile, which phase it is in.
//
// # Two rules an observer does not inherit from the shipped guest
//
//   - `fk_on_tick` IS ALLOWED HERE. The no-tick rule is the product's: a
//     finished balancer must cost zero script, which is the whole architecture.
//     An observer's job is a schedule, and a schedule is a tick handler.
//   - THE HEAP DOES NOT MATTER HERE. `logline.go`'s diet, the `-gc` decision and
//     the marathon slopes are all about a guest that lives in a player's save
//     forever. An observer runs for seconds in a throwaway world.
//
// What an observer DOES inherit is the entity-reference rule, and it inherits it
// as a convenience rather than as doctrine: nothing here retains a LuaObject
// across a dispatch. Positions are what a rig registry holds, and a chest is
// re-found on the tile it was built on. `Object.Retain` exists and works; this
// package does not need it, and a suite that does should reach for it knowingly.
package harness

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

// PlayerForce is the force every rig in the estate is built on unless it says
// otherwise. `m3` and `edge` build on a second one; that is what Piece.Force is
// for.
const PlayerForce = "player"

// AuditMarker is the shipped mod's synchronous "drain the deferred queue and
// re-classify now" trigger (guest/go/main.go's AuditName). Building one raises
// `script_raised_built`, which the engine dispatches BEFORE create_entity
// returns -- so a marker placed here reaches the mod under test inside this same
// dispatch, across two separate wasm instances.
//
// THAT SYNCHRONY IS LOAD-BEARING FOR EVERY SUITE. A `--create` never reaches a
// tick, so without a marker at the end of `on_init` every network in every
// suite's save would compile on the first tick of the BENCHMARK phase instead of
// into the save.
const AuditMarker = "bbb-audit"

// fatal is the observer estate's own error level, and `test/run.sh`'s guest_gate
// fails a run on it.
//
// It exists because a guest cannot call Lua's `error()`. The Lua these observers
// replace aborted `on_init` outright when a create_entity came back nil, which
// is the correct severity: a rig that did not land makes every number after it
// meaningless, and a suite that carried on would report a plausible wrong
// answer. A guest has no way to unwind -- there are no coroutines under the wasm
// frame -- so the equivalent is a line the runner refuses to see.
//
// A SEPARATE TAG FROM ANY OBSERVER'S. `[BBB-OBS] error:` is one thing for
// run.sh to grep across all fourteen, and it cannot be confused with the mod's
// own `[BBB] error:`, which means something narrower (a compile produced no
// network) and is gated separately.
var fatal = Line{Tag: "[BBB-OBS] "}

// Fatal reports that the harness could not do what a rig asked, and names the
// host's own reason for it.
//
// `fk.LastError()` is what makes this better than the `error()` it replaces: the
// Lua said "could not place X at (x,y)" and stopped there, where this says why
// the engine refused.
func Fatal(what, detail string) {
	fatal.Open("error: ").S(what).S(": ").S(detail).End()
}

func fatalCall(what string) { Fatal(what, fk.LastError()) }

// ---------------------------------------------------------------------------
// geometry
// ---------------------------------------------------------------------------

// XY is a TILE, and it is what an observer's rig registry holds instead of an
// entity.
//
// A rig is built in `fk_on_init` during `--create` and sampled during
// `--benchmark`, so anything a registry keeps crosses a save. A retained
// LuaObject would survive that -- FkLua's persistent handle space lives in
// `storage` and Factorio serializes the reference -- and a tile needs nothing to
// go on being true, which is the shipped guest's own rule reached from the other
// side.
type XY struct{ X, Y int }

// Center is the centre of tile (x, y), which is where a 1x1 entity goes. The
// estate's Lua called this `P`.
func Center(x, y int) fkapi.MapPosition {
	return fkapi.MapPosition{X: float64(x) + 0.5, Y: float64(y) + 0.5}
}

// TileBox is the whole of tile (x, y): [x, x+1] x [y, y+1].
//
// `find_entities_filtered` returns everything whose bounding box TOUCHES the
// area, and a 1x1 entity on tile n occupies [n+0.15, n+0.85] or thereabouts --
// so this box finds what is on the tile and cannot reach the neighbour, whose
// nearest edge is past n+1.
func TileBox(x, y int) fkapi.BoundingBox {
	return fkapi.BoundingBox{
		LeftTop:     fkapi.MapPosition{X: float64(x), Y: float64(y)},
		RightBottom: fkapi.MapPosition{X: float64(x + 1), Y: float64(y + 1)},
	}
}

// InnerBox is tile (x, y) inset by a tenth on every side.
//
// The estate's Lua used both this and TileBox and the difference never mattered
// to an answer; it is kept because the tighter one is the honest question when
// what is being asked is "what is standing HERE", and because a query that
// cannot reach a neighbouring tile cannot be made wrong by a rig moving one
// tile.
func InnerBox(x, y int) fkapi.BoundingBox {
	return fkapi.BoundingBox{
		LeftTop:     fkapi.MapPosition{X: float64(x) + 0.1, Y: float64(y) + 0.1},
		RightBottom: fkapi.MapPosition{X: float64(x) + 0.9, Y: float64(y) + 0.9},
	}
}

// Box is a bounding box in raw map coordinates, for the rig that needs one.
func Box(x0, y0, x1, y1 float64) fkapi.BoundingBox {
	return fkapi.BoundingBox{
		LeftTop:     fkapi.MapPosition{X: x0, Y: y0},
		RightBottom: fkapi.MapPosition{X: x1, Y: y1},
	}
}

// ---------------------------------------------------------------------------
// the flat scratch surface
// ---------------------------------------------------------------------------

// Flat describes the surface every suite in the estate builds its rigs on: a
// generated, always-day, peaceful, water-free, cliff-free plain with one tile
// type on it and nothing else standing anywhere near the rigs.
//
// EVERY FIELD OF IT WAS A COPY-PASTED LITERAL IN FOURTEEN control.lua FILES.
// What differs between suites is the name, the extents and where the chunks are
// asked for; nothing else ever did.
type Flat struct {
	Name string

	// MapWidth and MapHeight bound the surface. A bounded surface is what stops
	// a chunk request wandering, and the estate used 256 or 512.
	MapWidth, MapHeight uint32

	// ChunkCenter and ChunkRadius are the `request_to_generate_chunks` call.
	// Generation is asked for and then FORCED, because a rig built on an
	// ungenerated chunk is a rig built on nothing.
	ChunkCenter fkapi.MapPosition
	ChunkRadius uint32

	// X0..Y1 is the INCLUSIVE tile box that gets cleared, de-decorated and
	// paved. It is one box and not two on purpose: the estate's Lua wrote the
	// same numbers twice, once as a BoundingBox for the sweep and once as loop
	// bounds for the tiles, and two statements of one box is one edit away from
	// a rig standing on a tile nobody paved.
	X0, Y0, X1, Y1 int

	// Tile is what the box is paved with. "grass-1" everywhere in the estate.
	Tile string
}

// Make creates the surface and returns it. A failure at any step is Fatal: every
// rig the caller is about to build depends on this having worked.
func (f Flat) Make() fkapi.LuaSurface {
	yes, no := true, false
	mgs := fkapi.MapGenSettings{
		Width:                             &f.MapWidth,
		Height:                            &f.MapHeight,
		PeacefulMode:                      &yes,
		DefaultEnableAllAutoplaceControls: &no,
		AutoplaceSettings: []fkapi.EntryStringAutoplaceSettings{
			{Key: "tile", Val: fkapi.AutoplaceSettings{
				TreatMissingAsDefault: &no,
				Settings: []fkapi.EntryStringAutoplaceControl{
					{Key: f.Tile, Val: fkapi.AutoplaceControl{}},
				},
			}},
			{Key: "decorative", Val: fkapi.AutoplaceSettings{TreatMissingAsDefault: &no}},
			{Key: "entity", Val: fkapi.AutoplaceSettings{TreatMissingAsDefault: &no}},
		},
		// CLIFFS ARE KILLED BY THE EXPRESSION AND NOT BY cliff_settings, which
		// is where this deviates from the Lua and why.
		//
		// The Lua passed `cliff_settings = { richness = 0 }`, leaving every
		// other member of the concept to Factorio's defaults. The generated
		// CliffPlacementSettings is a STRUCT whose `name` and `control` are
		// non-optional Go strings, so writing one here would send `name = ""` --
		// a cliff prototype that does not exist -- rather than omitting the
		// field. `cliffiness = "0"` is what actually produces no cliffs, it was
		// already in every one of these settings blocks, and the sweep below
		// destroys anything that reached the box regardless.
		PropertyExpressionNames: []fkapi.EntryStringString{{Key: "cliffiness", Val: "0"}},
		StartingPoints:          []fkapi.MapPosition{},
	}
	// `water = 0` is NOT passed and was never read. It is not a member of the
	// MapGenSettings concept in 2.x -- the water control lives in
	// autoplace_controls -- so the Lua's top-level key was inert, and
	// `default_enable_all_autoplace_controls = false` with an empty entity and
	// decorative list is what actually leaves the plain bare.
	o, err := fkapi.Game.CreateSurface(f.Name, &mgs)
	if err != nil {
		fatalCall("could not create surface " + f.Name)
		return fkapi.LuaSurface{}
	}
	s := fkapi.LuaSurface{Object: o}
	if err := s.SetAlwaysDay(true); err != nil {
		fatalCall("always_day on " + f.Name)
	}
	radius := f.ChunkRadius
	if err := s.RequestToGenerateChunks(f.ChunkCenter, &radius); err != nil {
		fatalCall("request_to_generate_chunks on " + f.Name)
	}
	if err := s.ForceGenerateChunkRequests(); err != nil {
		fatalCall("force_generate_chunk_requests on " + f.Name)
	}

	area := Box(float64(f.X0), float64(f.Y0), float64(f.X1), float64(f.Y1))
	// Everything the generator left standing goes, except a character: a
	// headless run has none, and the estate's Lua guarded for it anyway.
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &area})
	if err != nil {
		fatalCall("sweeping " + f.Name)
	}
	for _, e := range found {
		ent := fkapi.LuaEntity{Object: e}
		if t, err := ent.Type(); err == nil && t == "character" {
			continue
		}
		ent.Destroy(fkapi.LuaEntityDestroyArgs{})
	}
	s.DestroyDecoratives(fkapi.LuaSurfaceDestroyDecorativesArgs{Area: &area})

	// The pave. Outer x, inner y, which is the order the estate's Lua built the
	// list in -- kept so that a diff against it reads.
	tiles := make([]fkapi.Tile, 0, (f.X1-f.X0+1)*(f.Y1-f.Y0+1))
	for x := f.X0; x <= f.X1; x++ {
		for y := f.Y0; y <= f.Y1; y++ {
			tiles = append(tiles, fkapi.Tile{
				Position: fkapi.TilePosition{X: int32(x), Y: int32(y)},
				Name:     f.Tile,
			})
		}
	}
	correct, off := true, false
	noEntities := fkapi.OfBool(false)
	if err := s.SetTiles(tiles, &correct, &noEntities, &off, &off, nil, nil); err != nil {
		fatalCall("set_tiles on " + f.Name)
	}
	return s
}

// Surface looks a surface up by name. Fatal if it is not there: every caller is
// about to build on it.
func Surface(name string) fkapi.LuaSurface {
	s, ok := SurfaceIfAny(name)
	if !ok {
		fatalCall("no surface " + name)
	}
	return s
}

// SurfaceIfAny is Surface for a surface that may legitimately not exist.
//
// It is the estate's `game.surfaces[name]` read against a nil, and the surface
// it is always asked about is `bbb-hidden` -- which the mod under test creates
// LAZILY, on the first compile, and which two suites then go on to delete out
// from under it on purpose. A count that has to include the hidden side has to
// cope with there being no hidden side yet; `m2`'s conservation check reads
// `if hid then` for exactly that reason, and `m3` deletes it and watches the
// mod build another.
func SurfaceIfAny(name string) (fkapi.LuaSurface, bool) {
	o, err := fkapi.Game.GetSurface(fkapi.OfString(name))
	if err != nil || o == nil {
		return fkapi.LuaSurface{}, false
	}
	return fkapi.LuaSurface{Object: *o}, true
}

// Tick is the current game tick, for a report line that carries one.
func Tick() uint64 {
	t, err := fkapi.Game.Tick()
	if err != nil {
		fatalCall("game.tick")
	}
	return t
}

// ---------------------------------------------------------------------------
// placing
// ---------------------------------------------------------------------------

// Piece is one `create_entity`, at a TILE rather than at a position: everything
// the estate places is 1x1 and goes at a tile centre.
type Piece struct {
	Name string
	X, Y int

	// Pos overrides the tile centre with a raw map position, for the one thing
	// in the estate that is not 1x1.
	//
	// `m2`'s `spio` rig feeds two rows through ONE vanilla express splitter, and
	// a splitter is two tiles wide: facing east it straddles the boundary
	// BETWEEN its two rows, so its y is an integer where every other piece's is
	// an integer-plus-a-half, and its x is the centre of the single column it
	// stands in. The estate's Lua called this `put_at` and wrote the position
	// out; there is no tile that names it.
	//
	// One suite uses this, which is below the bar for a helper of its own -- and
	// the alternative is not a smaller surface, it is a second copy of the
	// argument table below in an observer that should be holding rigs and log
	// lines. When Pos is set, X and Y are used only to NAME the piece in a
	// Fatal.
	Pos *fkapi.MapPosition

	// Dir is `defines.direction.*`, absent when nil. A pointer rather than a
	// value because north is 0 and "facing north" is not the same statement as
	// "no direction given".
	Dir *uint32

	// Type is a linked belt's or a loader's end type, "input" or "output".
	// Empty means the key is not sent.
	Type string

	// InnerName is what an `entity-ghost` is a ghost OF, absent when empty.
	//
	// A ghost is not the thing it depicts, which is the whole of what `m3`'s
	// ghost phase is about: the build event carries an entity whose `name` is
	// "entity-ghost", so the mod under test's registry must not grow when one is
	// placed, and must grow when it is revived.
	InnerName string

	// Force defaults to PlayerForce.
	Force string

	// Quality is `quality = "..."`, absent when empty.
	//
	// THE `qual` SUITE IS ENTIRELY THIS ONE FIELD. `find_entity` resolves a bare
	// name as NORMAL QUALITY ONLY, so a guest lookup that used it worked on every
	// other suite's save and silently failed on a quality-rolled part
	// (guest/go/findpart.go). Every part that suite builds carries it.
	Quality string

	// FastReplace sends `fast_replace = true`.
	//
	// IT IS NOT THE PLAYER'S GESTURE AND MUST NOT BE READ AS ONE. Handed a
	// replace the engine would refuse, `create_entity` falls back to CREATING --
	// so a rig that wants to drive the real thing asks `can_fast_replace` first,
	// which is what `qual`'s replace probe does.
	FastReplace bool

	// Raise sends `raise_built = true`, so the engine dispatches
	// `script_raised_built` before create_entity returns. Every piece of a rig
	// that the mod under test must SEE needs it; a source or sink chest does
	// not, and the estate's Lua left it off there.
	Raise bool
}

// Place builds one piece and returns it. A create_entity that came back with
// nothing is Fatal -- the estate's Lua called `error()` here, and for the same
// reason.
func Place(s fkapi.LuaSurface, p Piece) fkapi.Object {
	o, ok := place(s, p)
	if !ok {
		fatal.Open("error: could not place ").S(p.Name).S(" at (").
			I(int64(p.X)).S(",").I(int64(p.Y)).S("): ").S(fk.LastError()).End()
	}
	return o
}

// PlaceSoft builds one piece and reports whether anything came back, WITHOUT
// being Fatal about it.
//
// It is for the one shape Place cannot serve: a placement whose failure is a
// legitimate outcome of the schedule rather than a broken rig. The estate's Lua
// called this `put_soft`, and `mar` is where it earns its keep -- every leg
// there places and removes the same tiles a hundred times over, and a collision
// with a piece the previous iteration has not finished giving up is a fact about
// the schedule, not a reason to fail the run. `qual`'s two probes use it for a
// different reason: what came back IS the measurement (`created=true`).
func PlaceSoft(s fkapi.LuaSurface, p Piece) (fkapi.Object, bool) { return place(s, p) }

func place(s fkapi.LuaSurface, p Piece) (fkapi.Object, bool) {
	force := p.Force
	if force == "" {
		force = PlayerForce
	}
	pos := Center(p.X, p.Y)
	if p.Pos != nil {
		pos = *p.Pos
	}
	kv := make([]fkapi.KeyValue, 0, 8)
	kv = append(kv,
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(p.Name)},
		fkapi.KeyValue{Key: fkapi.OfString("position"),
			Val: fkapi.OfArray(fkapi.OfNumber(pos.X), fkapi.OfNumber(pos.Y))},
		fkapi.KeyValue{Key: fkapi.OfString("force"), Val: fkapi.OfString(force)},
	)
	if p.Dir != nil {
		kv = append(kv, fkapi.KeyValue{Key: fkapi.OfString("direction"),
			Val: fkapi.OfNumber(float64(*p.Dir))})
	}
	if p.Type != "" {
		kv = append(kv, fkapi.KeyValue{Key: fkapi.OfString("type"),
			Val: fkapi.OfString(p.Type)})
	}
	if p.InnerName != "" {
		kv = append(kv, fkapi.KeyValue{Key: fkapi.OfString("inner_name"),
			Val: fkapi.OfString(p.InnerName)})
	}
	if p.Quality != "" {
		kv = append(kv, fkapi.KeyValue{Key: fkapi.OfString("quality"),
			Val: fkapi.OfString(p.Quality)})
	}
	if p.FastReplace {
		kv = append(kv, fkapi.KeyValue{Key: fkapi.OfString("fast_replace"),
			Val: fkapi.OfBool(true)})
	}
	if p.Raise {
		kv = append(kv, fkapi.KeyValue{Key: fkapi.OfString("raise_built"),
			Val: fkapi.OfBool(true)})
	}
	o, err := s.CreateEntity(fkapi.OfMap(kv...))
	if err != nil || o == nil {
		return fkapi.Object{}, false
	}
	return *o, true
}

// Audit places a `bbb-audit` marker, which asks the mod under test to drain its
// deferred queue and re-classify the world, synchronously, inside this dispatch.
//
// IT DELIBERATELY DOES NOT CHECK WHAT COMES BACK, and that is not laziness --
// it is the one place a marker differs from every other piece an observer
// places. The marker DESTROYS ITSELF from inside the `script_raised_built` that
// `raise_built = true` dispatches, so by the time `create_entity` returns there
// is no entity left to hand over: measured here, the call comes back with no
// object and no error at all. The Lua this replaces never looked either, for
// exactly this reason.
//
// So a marker cannot report whether it worked, and nothing needs it to: what
// says the drain happened is the mod's own `[BBB] audit ...` line landing in the
// log between this observer's two, which is what every assertion script keys on.
func Audit(s fkapi.LuaSurface, x, y int) {
	place(s, Piece{Name: AuditMarker, X: x, Y: y, Raise: true})
}

// ---------------------------------------------------------------------------
// looking
// ---------------------------------------------------------------------------

// FindOnTile returns the one entity of the given name standing on tile (x, y).
//
// The name filter is applied by the engine in C++, and the box cannot reach a
// neighbouring tile, so "found" here means exactly one thing.
func FindOnTile(s fkapi.LuaSurface, name string, x, y int) (fkapi.Object, bool) {
	box := TileBox(x, y)
	n := fkapi.OfString(name)
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &box, Name: &n})
	if err != nil || len(found) == 0 {
		return fkapi.Object{}, false
	}
	return found[0], true
}

// FindAt is the estate's `at(s, x, y, filter)`: a POINT query at the tile
// centre, optionally filtered by name or by type (empty means the key is not
// sent).
//
// A POINT AND NOT A BOX, and the difference is load-bearing wherever a rig lays
// belts side by side. `find_entities_filtered` with an `area` returns everything
// whose bounding box TOUCHES it, and a transport belt's selection box is the
// whole of its tile -- so a box query on tile x can also reach the belt on x+1
// along the shared edge, and `[0]` of that is whichever the engine listed first.
// `mar` removes one named belt out of a run of four, a hundred times over, and
// the wrong one would be a different world with a plausible slope. A position
// query returns what CONTAINS the point, which is one entity.
//
// FindOnTile above stays the box form: it is what a lookup wants when the answer
// is "is there one of these here at all", and the pilot's rigs have nothing
// beside the tiles they ask about.
func FindAt(s fkapi.LuaSurface, x, y int, name, typ string) (fkapi.Object, bool) {
	pos := Center(x, y)
	f := fkapi.EntitySearchFilters{Position: &pos}
	if name != "" {
		v := fkapi.OfString(name)
		f.Name = &v
	}
	if typ != "" {
		v := fkapi.OfString(typ)
		f.Type = &v
	}
	found, err := s.FindEntitiesFiltered(f)
	if err != nil || len(found) == 0 {
		return fkapi.Object{}, false
	}
	return found[0], true
}

// KillAt removes what FindAt finds, the way a player mining it does as far as
// the mod under test is concerned: an event is raised and the entity is gone
// when the dispatch returns. It reports whether there was anything there.
//
// NOTHING HERE IS FATAL ABOUT A MISS, for the same reason PlaceSoft exists: a
// leg that removes what a previous iteration may not have placed is a fact about
// the schedule.
func KillAt(s fkapi.LuaSurface, x, y int, name, typ string) bool {
	o, ok := FindAt(s, x, y, name, typ)
	if !ok {
		return false
	}
	Destroy(o, true)
	return true
}

// Destroy destroys one entity, optionally raising `script_raised_destroy`.
func Destroy(o fkapi.Object, raise bool) {
	args := fkapi.LuaEntityDestroyArgs{}
	if raise {
		args.RaiseDestroy = &raise
	}
	if _, err := (fkapi.LuaEntity{Object: o}).Destroy(args); err != nil {
		fatalCall("destroying an entity")
	}
}

// EntitiesIn is every entity in a box, optionally filtered by name. It is the
// raw sweep `mar`'s conservation count and `qual`'s tile probes are built on.
func EntitiesIn(s fkapi.LuaSurface, box fkapi.BoundingBox, name string) []fkapi.Object {
	f := fkapi.EntitySearchFilters{Area: &box}
	if name != "" {
		v := fkapi.OfString(name)
		f.Name = &v
	}
	found, err := s.FindEntitiesFiltered(f)
	if err != nil {
		fatalCall("sweeping a box")
		return nil
	}
	return found
}

// EntitiesInOfType is every entity in a box of one `type`.
//
// A SEPARATE FUNCTION FROM EntitiesIn RATHER THAN A WIDER ONE, because the two
// are asked in the same breath and mean different things: the estate's
// item-conservation counts sweep a box TWICE -- once filtered to `item-entity`
// for what is lying on the ground, and once unfiltered for what is standing in
// transport lines and inventories -- and a single call with both keys would be
// neither. `m2`'s loss check and `edge`'s whole-world count are both that pair.
func EntitiesInOfType(s fkapi.LuaSurface, box fkapi.BoundingBox, typ string) []fkapi.Object {
	t := fkapi.OfString(typ)
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &box, Type: &t})
	if err != nil {
		fatalCall("sweeping a box for " + typ)
		return nil
	}
	return found
}

// FindExactlyOne is FindOnTile with the estate's own sanity check: a rig that
// found none, or more than one, has already gone wrong and every number after it
// would be about a different world.
func FindExactlyOne(s fkapi.LuaSurface, name string, x, y int) (fkapi.Object, bool) {
	box := TileBox(x, y)
	n := fkapi.OfString(name)
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &box, Name: &n})
	if err != nil {
		fatalCall("looking for " + name)
		return fkapi.Object{}, false
	}
	if len(found) != 1 {
		fatal.Open("error: expected 1 ").S(name).S(" at (").I(int64(x)).S(",").
			I(int64(y)).S("), found ").U(uint64(len(found))).End()
		return fkapi.Object{}, false
	}
	return found[0], true
}

// NamesOnTile is every entity name standing on tile (x, y), SORTED.
//
// Sorted because it is reported into a log line an assertion script compares
// against a literal, and `find_entities_filtered` promises no order.
func NamesOnTile(s fkapi.LuaSurface, x, y int) []string {
	box := InnerBox(x, y)
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Area: &box})
	if err != nil {
		fatalCall("reading the tile at")
		return nil
	}
	names := make([]string, 0, len(found))
	for _, e := range found {
		if n, err := (fkapi.LuaEntity{Object: e}).Name(); err == nil {
			names = append(names, n)
		}
	}
	SortStrings(names)
	return names
}

// SortStrings is an insertion sort, and it is here rather than `sort.Strings`
// for one reason: these lists are a handful of entity names, and `sort` is a
// package a guest otherwise never links.
func SortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		v := s[i]
		j := i - 1
		for j >= 0 && s[j] > v {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = v
	}
}

// ChestCount totals everything in the chest of the given name on tile (x, y).
//
// -1 when there is no such chest, which is the estate's own convention: a rig
// whose sink went missing reports a number no real count can produce, rather
// than a zero that reads as "nothing was delivered".
func ChestCount(s fkapi.LuaSurface, name string, x, y int) int64 {
	o, ok := FindOnTile(s, name, x, y)
	if !ok {
		return -1
	}
	return InventoryTotal(o)
}

// InventoryTotal is everything in one entity's CHEST inventory, or -1 if it has
// none. Split out of ChestCount because three suites total a chest they are
// already holding rather than one they have to look up.
func InventoryTotal(o fkapi.Object) int64 {
	inv, err := (fkapi.LuaControl{Object: o}).GetInventory(fkapi.DefinesInventoryChest())
	if err != nil || inv == nil {
		return -1
	}
	contents, err := (fkapi.LuaInventory{Object: *inv}).GetContents()
	if err != nil {
		return -1
	}
	var total int64
	for _, item := range contents {
		total += int64(item.Count)
	}
	return total
}

// TransportLineItems is every item standing in one entity's transport lines, or
// 0 for an entity that has none.
//
// THE `err != nil` ARM IS THE ESTATE'S `pcall` AND NOT AN ERROR PATH. Asking a
// chest for its transport lines RAISES in Factorio, so the Lua wrapped
// `get_max_transport_line_index` in a pcall and read a failure as "no lines" --
// which is what a sweep over every entity in a box needs, because most of them
// are not belts. A generated binding hands the same refusal back as a Status.
func TransportLineItems(o fkapi.Object) int64 {
	e := fkapi.LuaEntity{Object: o}
	n, err := e.GetMaxTransportLineIndex()
	if err != nil {
		return 0
	}
	var total int64
	for i := uint32(1); i <= n; i++ {
		line, err := e.GetTransportLine(i)
		if err != nil {
			continue
		}
		if c, err := (fkapi.LuaTransportLine{Object: line}).GetItemCount(nil); err == nil {
			total += int64(c)
		}
	}
	return total
}

// ForceByName is `game.forces[name]`, which is a LuaCustomTable index: the raw
// handle plus its index operator, two host calls and no allocation. The
// whole-table attribute would build a Go slice of every force in the game to
// answer a point query.
func ForceByName(name string) (fkapi.Object, bool) {
	raw, err := fkapi.Game.ForcesRaw()
	if err != nil {
		return fkapi.Object{}, false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(name))
	if err != nil || v.Tag != fkapi.TagObject {
		return fkapi.Object{}, false
	}
	return v.Object, true
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

// Step is one entry of an observer's tick schedule.
type Step struct {
	Tick uint64
	Do   func()
}

// Run fires every step whose tick this is.
//
// A SLICE AND A LINEAR SCAN, not a map. The estate's Lua used a table keyed by
// tick, which is a point query and would be fine here too -- but a schedule is a
// dozen entries read sixty times a second in a world where nothing else is
// happening, and a slice is the shape whose determinism needs no argument.
func Run(steps []Step, tick uint64) {
	for _, s := range steps {
		if s.Tick == tick {
			s.Do()
		}
	}
}

// ---------------------------------------------------------------------------
// prototypes
// ---------------------------------------------------------------------------

// ItemProtoExists, EntityProtoExists and QualityProtoExists answer whether the
// running game has a prototype of that name.
//
// TWO HOST CALLS AND NO ALLOCATION EACH, which is the shipped guest's own idiom
// (guest/go/legacy.go's legacyStubPresent): `prototypes.item` is a
// LuaCustomTable, so the RAW handle plus its index operator is a POINT query,
// where the materialising attribute would build a Go slice of every item
// prototype in the game -- thousands, with a string each -- to answer a yes/no
// question. A missing key is Lua's nil, which arrives as TagNil rather than as
// an error.
//
// TWO SUITES NEED THEM AND FOR OPPOSITE REASONS. `mix` and `plat` check their
// item lists at init so that a renamed prototype fails the CREATE log with the
// name in it, rather than leaving a rig that quietly carries fewer kinds than it
// claims -- which for `mix`'s overflow rig would be a rig that never overflows
// and passes vacuously. `mig` guards every lookup with one, because
// `find_entities_filtered{name = ...}` RAISES for a name the game does not have
// and that suite runs in a phase where `bbb-balancer-part` is not a prototype at
// all.
func ItemProtoExists(name string) bool {
	raw, err := fkapi.Prototypes.ItemRaw()
	if err != nil {
		return false
	}
	return customTableHas(raw, name)
}

func EntityProtoExists(name string) bool {
	raw, err := fkapi.Prototypes.EntityRaw()
	if err != nil {
		return false
	}
	return customTableHas(raw, name)
}

func QualityProtoExists(name string) bool {
	_, ok := QualityProto(name)
	return ok
}

// QualityProto is the LuaQualityPrototype of that name.
//
// IT IS A HANDLE AND NOT A STRING, and that is a deviation from the Lua worth
// knowing about. `InfinityInventoryFilter.quality` takes a QualityID, which the
// description spells `string or LuaQualityPrototype`, and the Lua sent the
// string; the generated struct field is a `*Object`, so a guest sends the
// prototype. The two are the same value to the engine, it is the reading the
// shipped guest already takes for the same union (guest/go/legacy.go passes the
// handle rather than copying a name into the guest heap), and it costs the one
// point query this function is.
func QualityProto(name string) (fkapi.Object, bool) {
	raw, err := fkapi.Prototypes.QualityRaw()
	if err != nil {
		return fkapi.Object{}, false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(name))
	if err != nil || v.Tag != fkapi.TagObject {
		return fkapi.Object{}, false
	}
	return v.Object, true
}

func customTableHas(raw fkapi.Object, name string) bool {
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(name))
	if err != nil {
		return false
	}
	return v.Tag == fkapi.TagObject
}

// ---------------------------------------------------------------------------
// forces and surfaces
// ---------------------------------------------------------------------------

// CreateForce is `game.create_force`, and a failure is Fatal: every rig the
// caller is about to build belongs to it.
func CreateForce(name string) fkapi.LuaForce {
	o, err := fkapi.Game.CreateForce(name)
	if err != nil {
		fatalCall("could not create force " + name)
		return fkapi.LuaForce{}
	}
	return fkapi.LuaForce{Object: o}
}

// SurfacesByIndex is every surface in the game, in INDEX order.
//
// SORTED RATHER THAN LEFT TO THE WALK, and `mig` is what needs it: it reports a
// per-surface census into a line an assertion reads, and `game.surfaces` is a
// Lua hash whose iteration order is not a promise. It is the same habit
// `collectSurfaces` follows in the shipped guest, and there for a much harder
// reason -- surface order decides node ids, which decide slots, which is a
// desync.
//
// An insertion sort, for the reason SortStrings is one: a big save has a dozen
// surfaces and `sort` is a package a guest otherwise never links.
func SurfacesByIndex() []fkapi.LuaSurface {
	pairs, err := fkapi.Game.Surfaces()
	if err != nil {
		fatalCall("reading game.surfaces")
		return nil
	}
	out := make([]fkapi.LuaSurface, 0, len(pairs))
	idx := make([]uint32, 0, len(pairs))
	for _, p := range pairs {
		s := fkapi.LuaSurface{Object: p.Val}
		n, err := s.Index()
		if err != nil {
			continue
		}
		j := len(out)
		out = append(out, s)
		idx = append(idx, n)
		for j > 0 && idx[j-1] > idx[j] {
			idx[j-1], idx[j] = idx[j], idx[j-1]
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// infinity chests
// ---------------------------------------------------------------------------

// InfinityFilter sets an infinity chest's ONE filter, optionally at a quality.
//
// One filter and not a list, because that is what every source in the estate
// wants and because of a measurement: the `mix` suite's `probe` band is an
// infinity chest carrying SIX filters at once and it delivers ONE KIND -- a
// loader draws from the first stack it finds and the chest tops that same stack
// straight back up. A sushi belt is made by ROTATING a single filter instead.
// MultiFilter below is that probe, kept precisely so the paragraph stays a
// measurement.
func InfinityFilter(o fkapi.Object, item, quality string, count uint32) {
	index := uint32(1)
	mode := "at-least"
	f := fkapi.InfinityInventoryFilter{
		Index: &index,
		Name:  fkapi.OfString(item),
		Count: &count,
		Mode:  &mode,
	}
	if quality != "" {
		q, ok := QualityProto(quality)
		if !ok {
			Fatal("no quality prototype", quality)
			return
		}
		f.Quality = &q
	}
	if err := (fkapi.LuaEntity{Object: o}).SetInfinityContainerFilters(
		[]fkapi.InfinityInventoryFilter{f}); err != nil {
		fatalCall("setting an infinity filter to " + item)
	}
}

// MultiFilter sets EVERY name as a filter at once. Only the `mix` suite's
// `probe` band uses it, and what it is for is the measurement above.
func MultiFilter(o fkapi.Object, items []string, count uint32) {
	mode := "at-least"
	fs := make([]fkapi.InfinityInventoryFilter, 0, len(items))
	for i, name := range items {
		index := uint32(i + 1)
		c := count
		fs = append(fs, fkapi.InfinityInventoryFilter{
			Index: &index,
			Name:  fkapi.OfString(name),
			Count: &c,
			Mode:  &mode,
		})
	}
	if err := (fkapi.LuaEntity{Object: o}).SetInfinityContainerFilters(fs); err != nil {
		fatalCall("setting a multi-filter chest")
	}
}

// RemoveUnfiltered turns on an infinity chest's `remove_unfiltered_items`, which
// is what makes a rotating filter a SUSHI source rather than a mixed one: the
// previous band's kind is voided rather than left in the chest for the loader to
// go on preferring.
func RemoveUnfiltered(o fkapi.Object, on bool) {
	if err := (fkapi.LuaEntity{Object: o}).SetRemoveUnfilteredItems(on); err != nil {
		fatalCall("setting remove_unfiltered_items")
	}
}

// ---------------------------------------------------------------------------
// counting items
// ---------------------------------------------------------------------------

// Tally is a per-key item count that keeps INSERTION ORDER.
//
// A SLICE AND A LINEAR SCAN RATHER THAN A MAP, and the reason is the estate's
// oldest rule read one level out: the counts are emitted into log lines an
// assertion script compares against literals, so the ORDER has to be a
// property of the program rather than of a hash. The suites sort before they
// emit -- `Names` does it -- so a map would in fact be safe here; a slice is the
// shape whose determinism needs no argument, and forty-eight kinds is a scan
// nobody can measure.
type Tally struct {
	keys   []string
	counts []int64
}

// Add adds n to key, which may be negative: `mix` subtracts an infinity chest's
// contents back out of a count it has already taken.
func (t *Tally) Add(key string, n int64) {
	for i, k := range t.keys {
		if k == key {
			t.counts[i] += n
			return
		}
	}
	t.keys = append(t.keys, key)
	t.counts = append(t.counts, n)
}

// Get is the count under key, 0 for a key never added.
func (t *Tally) Get(key string) int64 {
	for i, k := range t.keys {
		if k == key {
			return t.counts[i]
		}
	}
	return 0
}

// Names is every key whose count is NON-ZERO, sorted.
//
// Non-zero rather than every key, because that is what the Lua did: a name whose
// pluses and minuses cancelled -- an item that was only ever inside an infinity
// chest -- is not a kind that is present, and emitting a `count=0` line for it
// would put a line in the log the golden does not have.
func (t *Tally) Names() []string {
	out := make([]string, 0, len(t.keys))
	for i, k := range t.keys {
		if t.counts[i] != 0 {
			out = append(out, k)
		}
	}
	SortStrings(out)
	return out
}

// Total is the sum of every non-zero count.
func (t *Tally) Total() int64 {
	var n int64
	for _, c := range t.counts {
		if c != 0 {
			n += c
		}
	}
	return n
}

// Reset empties the tally, keeping the capacity.
func (t *Tally) Reset() { t.keys, t.counts = t.keys[:0], t.counts[:0] }

// EachLine calls fn with every transport line of one entity.
//
// THE err != nil ARM IS THE ESTATE'S pcall AND NOT AN ERROR PATH. Asking a chest
// for its transport lines RAISES in Factorio, so the Lua wrapped
// `get_max_transport_line_index` in a pcall and read a failure as "no lines" --
// which is what a sweep over every entity in a box needs, because most of them
// are not belts. A generated binding hands the same refusal back as a Status.
func EachLine(o fkapi.Object, fn func(line fkapi.LuaTransportLine)) {
	e := fkapi.LuaEntity{Object: o}
	n, err := e.GetMaxTransportLineIndex()
	if err != nil {
		return
	}
	for i := uint32(1); i <= n; i++ {
		l, err := e.GetTransportLine(i)
		if err != nil {
			continue
		}
		fn(fkapi.LuaTransportLine{Object: l})
	}
}

// ChestContents is one entity's CHEST inventory, per (name, quality), or nil for
// an entity that has none.
func ChestContents(o fkapi.Object) []fkapi.ItemWithQualityCount {
	inv, err := (fkapi.LuaControl{Object: o}).GetInventory(fkapi.DefinesInventoryChest())
	if err != nil || inv == nil {
		return nil
	}
	c, err := (fkapi.LuaInventory{Object: *inv}).GetContents()
	if err != nil {
		return nil
	}
	return c
}

// GroundStack is what an `item-entity` is holding: its name, its count, and
// whether there was anything to read.
//
// `valid_for_read` is asked FIRST and separately, exactly as the Lua did: an
// item entity whose stack is empty answers false, and reading `name` off it
// raises.
func GroundStack(o fkapi.Object) (string, int64, bool) {
	st, err := (fkapi.LuaEntity{Object: o}).Stack()
	if err != nil {
		return "", 0, false
	}
	s := fkapi.LuaItemStack{Object: st}
	if ok, err := s.ValidForRead(); err != nil || !ok {
		return "", 0, false
	}
	name, err := s.Name()
	if err != nil {
		return "", 0, false
	}
	n, err := s.Count()
	if err != nil {
		return "", 0, false
	}
	return name, int64(n), true
}

// EntityType is one entity's `type`, empty when it cannot be read. It is the
// filter every count in the estate applies by hand: an `item-entity` is counted
// through its stack and everything else through its lines and its inventory,
// and an `infinity-container` is not a conserved quantity and is counted out.
func EntityType(o fkapi.Object) string {
	t, err := (fkapi.LuaEntity{Object: o}).Type()
	if err != nil {
		return ""
	}
	return t
}

// EntityTypeIs asks whether one entity's `type` is a given string WITHOUT
// bringing the string across.
//
// IT IS EntityType's ANSWER WITHOUT EntityType's ALLOCATION, and on a sweep that
// is the whole cost. `Type()` returns a Go string, which means `getStr` COPIES
// the host's bytes into the guest heap -- the arena underneath them is released
// when the call returns, so it has to. That is nothing on a handful of entities
// and it is the dominant term in `edge`, which asks the question of every entity
// on two whole surfaces after each of a hundred churn cycles: roughly nine
// hundred thousand short-lived strings that `-gc=leaking` never gives back.
//
// `type_is` compares on the HOST and returns a bool, so nothing crosses but the
// answer. It is the same reading the shipped guest already takes for the same
// reason -- guest/go/carry.go uses `name_is` over `name` and its header says why.
func EntityTypeIs(o fkapi.Object, want string) bool {
	ok, err := (fkapi.LuaEntity{Object: o}).TypeIs(want)
	return err == nil && ok
}

// SpaceLocationProto is the LuaSpaceLocationPrototype of that name.
//
// A HANDLE AND NOT A STRING, the same deviation QualityProto records and for the
// same reason: `LuaForce.create_space_platform`'s `planet` takes a
// SpaceLocationID, which the description spells `string or
// LuaSpaceLocationPrototype`, and the generated struct field is an `Object`.
func SpaceLocationProto(name string) (fkapi.Object, bool) {
	raw, err := fkapi.Prototypes.SpaceLocationRaw()
	if err != nil {
		return fkapi.Object{}, false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(name))
	if err != nil || v.Tag != fkapi.TagObject {
		return fkapi.Object{}, false
	}
	return v.Object, true
}

// PaveBox lays one tile name over an INCLUSIVE tile box.
//
// It is Flat.Make's own pave with the surface handed in, for the one caller that
// pavés a surface it did not create: `plat` lays space-platform foundation over
// a platform Factorio made for it. The four flags are the Lua's
// `set_tiles(tiles, true, false, false, false)` -- correct the edges, and touch
// neither entities nor decoratives nor raise an event.
func PaveBox(s fkapi.LuaSurface, x0, y0, x1, y1 int, tile string) {
	tiles := make([]fkapi.Tile, 0, (x1-x0+1)*(y1-y0+1))
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			tiles = append(tiles, fkapi.Tile{
				Position: fkapi.TilePosition{X: int32(x), Y: int32(y)},
				Name:     tile,
			})
		}
	}
	correct, off := true, false
	noEntities := fkapi.OfBool(false)
	if err := s.SetTiles(tiles, &correct, &noEntities, &off, &off, nil, nil); err != nil {
		fatalCall("paving with " + tile)
	}
}

// ---------------------------------------------------------------------------
// timing
// ---------------------------------------------------------------------------

// Profiler is `helpers.create_profiler`, and reading one is the whole reason
// fkapi has a binding for Factorio's GLOBAL log().
//
// A LuaProfiler HAS NO ACCESSOR RETURNING ITS DURATION -- its complete member
// set is add, divide, reset, restart, stop, object_name, object_name_is and
// valid, and not one of them returns the number. The engine renders it only when
// the profiler is an ELEMENT OF A LocalisedString, so `log{"", "took ", p}` is
// the whole idiom, and a guest that cannot call the global log cannot time
// anything and read the answer. Three suites publish timings by regexing exactly
// that line out of factorio-current.log.
//
// VERIFIED AGAINST THE GOLDEN BEFORE ANYTHING WAS BUILT ON IT (phase 3): what
// lands in the log is `... Duration: 0.507333ms`, the same shape and the same
// unit the Lua produced. The one thing that differs is Factorio's own note of
// WHERE the log() was called from -- `=[C]:...` rather than
// `@__mod__/control.lua:N:` -- because this is a host call fk_abi.lua makes
// through pcall where fk.Log is a wasm import the generated control.lua answers
// with a Lua log(). Nothing reads that prefix; see agents/estate-port.md.
type Profiler struct{ o fkapi.Object }

// StartProfiler creates a RUNNING profiler. `helpers.create_profiler(stopped)`
// defaults to started, which is what every call site in the estate wants.
func StartProfiler() Profiler {
	o, err := fkapi.Helpers.CreateProfiler(nil)
	if err != nil {
		fatalCall("helpers.create_profiler")
		return Profiler{}
	}
	return Profiler{o: o}
}

// Retain makes a profiler survive past the dispatch that created it, and
// Release gives the handle back.
//
// THIS IS THE ONE PLACE IN THE ESTATE THAT NEEDS A HANDLE ACROSS A TICK, and it
// needs one by construction rather than by convenience. Everything else here
// re-finds an entity on the tile it was built on, which is the shipped guest's
// own rule and costs a point query; a profiler is not on a tile and has no
// identity to re-find it by. `m2` times a recompile ACROSS THE TICK BOUNDARY --
// the mod under test batches, so the belt is destroyed in one tick and the
// network rebuilt when the deferred queue drains in the next, and a window that
// closed inside the first tick would measure a registry update and nothing at
// all of the compile. So the clock has to start in one dispatch and stop in
// another.
//
// A handle that is not retained is VALID ONLY DURING ITS OWN DISPATCH, and the
// failure is loud rather than silent: the port's first run came back with five
// `[BBB-OBS] error: stopping a profiler` lines, one for each of `m2`'s five
// tick-crossing windows, and none at all for the three that open and close
// inside one tick. The Lua kept its profiler in an upvalue and never had to
// think about it.
//
// Release is paired on every path: timedEnd releases whether or not it logs.
func (p Profiler) Retain() Profiler { return Profiler{o: p.o.Retain()} }

// Release hands a retained handle back. It is a no-op on a handle that was
// never retained, so a caller that pairs it unconditionally is correct.
func (p Profiler) Release() { p.o.Release() }

// Stop stops the clock. Nothing reads the duration until Log does.
func (p Profiler) Stop() {
	if err := (fkapi.LuaProfiler{Object: p.o}).Stop(); err != nil {
		fatalCall("stopping a profiler")
	}
}

// Log writes `<text>Duration: N ms`, where text carries its own tag and its own
// trailing space exactly as the Lua's second array element did.
//
// The empty first element is LocalisedString's "concatenate the rest" form.
func (p Profiler) Log(text string) {
	if err := fkapi.Log(fkapi.OfArray(
		fkapi.OfString(""), fkapi.OfString(text), fkapi.OfObject(p.o),
	)); err != nil {
		fatalCall("logging a profiler")
	}
}

// ---------------------------------------------------------------------------
// counting one named item, and finding by force
// ---------------------------------------------------------------------------

// EntitiesOfForce is every entity on a whole surface belonging to one force.
//
// NO AREA, which is what the `mig` suite needs and what makes it different from
// EntitiesIn: that suite counts its witness item over EVERY surface, including
// the hidden one this mod's compiler works on -- and that surface does not exist
// at all in the phase where the incumbent is installed, which is exactly why the
// count has to be written this way rather than against a named pair of boxes.
func EntitiesOfForce(s fkapi.LuaSurface, force string) []fkapi.Object {
	f := fkapi.OfString(force)
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Force: &f})
	if err != nil {
		return nil
	}
	return found
}

// CountNamedOn is how many entities of one prototype name stand on one surface,
// optionally of one force.
//
// GUARDED ON THE PROTOTYPE BY THE CALLER, and that is not tidiness:
// `find_entities_filtered{name = ...}` RAISES for a name the game does not have,
// and `mig` runs in a phase where `bbb-balancer-part` is not a prototype at all.
func CountNamedOn(s fkapi.LuaSurface, name, force string) int {
	n := fkapi.OfString(name)
	f := fkapi.EntitySearchFilters{Name: &n}
	if force != "" {
		v := fkapi.OfString(force)
		f.Force = &v
	}
	found, err := s.FindEntitiesFiltered(f)
	if err != nil {
		return 0
	}
	return len(found)
}

// ItemCountIn is how many of one named item an entity holds across its transport
// lines and its chest inventory. The `pcall` arm is EachLine's.
func ItemCountIn(o fkapi.Object, item string) int64 {
	v := fkapi.OfString(item)
	var total int64
	EachLine(o, func(l fkapi.LuaTransportLine) {
		if c, err := l.GetItemCount(&v); err == nil {
			total += int64(c)
		}
	})
	inv, err := (fkapi.LuaControl{Object: o}).GetInventory(fkapi.DefinesInventoryChest())
	if err == nil && inv != nil {
		if c, err := (fkapi.LuaInventory{Object: *inv}).GetItemCount(&v); err == nil {
			total += int64(c)
		}
	}
	return total
}

// InsertInto is `chest.insert{name = ..., count = ...}` and returns how many the
// engine took.
func InsertInto(o fkapi.Object, name string, count uint32) uint32 {
	n, err := (fkapi.LuaControl{Object: o}).Insert(fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(name)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(float64(count))},
	))
	if err != nil {
		fatalCall("inserting " + name)
		return 0
	}
	return n
}

// Researched is `force.technologies[name].researched`, as a THREE-VALUED answer:
// researched, present-but-not, or ABSENT.
//
// The third is the one that matters and it is why this does not return a plain
// bool. `mig` reports a technology's state as `true`, `false` or `absent`, and
// the difference between the last two is the whole of what says an incumbent's
// technology tree went with it.
//
// A LuaCustomTable point query, so a force with a thousand technologies costs
// two host calls rather than a Go slice of all of them.
func Researched(force fkapi.Object, name string) (done, present bool) {
	raw, err := (fkapi.LuaForce{Object: force}).TechnologiesRaw()
	if err != nil {
		return false, false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(name))
	if err != nil || v.Tag != fkapi.TagObject {
		return false, false
	}
	r, err := (fkapi.LuaTechnology{Object: v.Object}).Researched()
	if err != nil {
		return false, true
	}
	return r, true
}

// ItemPlaceResult is the entity name an item prototype places, empty when it
// places nothing.
//
// It is `mig`'s sharpest line: a stack of a removed mod's item survives only
// because a stub prototype kept the name alive, and `place_result` is what makes
// a surviving stack USEFUL rather than merely present.
func ItemPlaceResult(item string) string {
	raw, err := fkapi.Prototypes.ItemRaw()
	if err != nil {
		return ""
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(item))
	if err != nil || v.Tag != fkapi.TagObject {
		return ""
	}
	pr, err := (fkapi.LuaItemPrototype{Object: v.Object}).PlaceResult()
	if err != nil || pr == nil {
		return ""
	}
	n, err := (fkapi.LuaEntityPrototype{Object: *pr}).Name()
	if err != nil {
		return ""
	}
	return n
}
