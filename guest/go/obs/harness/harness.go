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
	o, err := fkapi.Game.GetSurface(fkapi.OfString(name))
	if err != nil || o == nil {
		fatalCall("no surface " + name)
		return fkapi.LuaSurface{}
	}
	return fkapi.LuaSurface{Object: *o}
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

	// Dir is `defines.direction.*`, absent when nil. A pointer rather than a
	// value because north is 0 and "facing north" is not the same statement as
	// "no direction given".
	Dir *uint32

	// Type is a linked belt's or a loader's end type, "input" or "output".
	// Empty means the key is not sent.
	Type string

	// Force defaults to PlayerForce.
	Force string

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

func place(s fkapi.LuaSurface, p Piece) (fkapi.Object, bool) {
	force := p.Force
	if force == "" {
		force = PlayerForce
	}
	pos := Center(p.X, p.Y)
	kv := make([]fkapi.KeyValue, 0, 6)
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
