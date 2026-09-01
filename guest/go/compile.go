package main

// The compiler: a cluster's shape and its adjacent belts in, a hidden splitter
// network out.
//
// Everything here runs at EVENT TIME and never per tick. That is the whole
// architecture: once the network exists, every item that moves through it is
// moved by engine C++, and the script cost of a running balancer is zero.
//
// The v1 policy is FULL TEARDOWN AND REBUILD of a cluster's network whenever
// anything relevant changes. Entity-diff minimisation is a later milestone;
// what is here instead is a cheap no-op check -- a fingerprint over the edge
// list -- so that the common case (a player laying belts NEAR a balancer
// without touching its edges) rebuilds nothing at all.
//
// Ordering rule, and it is load-bearing: every teardown for an event runs
// before every build. A cluster that split leaves its old network's visible
// linked belts standing on tiles that now belong to a DIFFERENT cluster, and a
// build that ran first would classify them as part of the world.

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

// The prototypes, defined in mod-data/prototypes/hidden.lua.
const (
	nameBelt       = "bbb-belt"
	nameSplitter   = "bbb-splitter"
	nameLaneSplit  = "bbb-lane-splitter"
	nameLinkedBelt = "bbb-linked-belt"
	hiddenSurfName = "bbb-hidden"
	linkTypeInput  = "input"
	linkTypeOutput = "output"
	// A splitter has 8 transport lines and nothing in the network has more; the
	// cap is a sanity bound on a number that arrives from the API.
	maxLinesToProbe = 8
)

func protoName(p plan.Proto) string {
	switch p {
	case plan.ProtoSplitter:
		return nameSplitter
	case plan.ProtoLaneSplitter:
		return nameLaneSplit
	case plan.ProtoLinkedBelt:
		return nameLinkedBelt
	}
	return nameBelt
}

// ---------------------------------------------------------------------------
// The hidden surface, and the slot grid on it
// ---------------------------------------------------------------------------

// slotW is comfortably wider than plan.Width(64) = 19, and slotH taller than
// the 64 rows a MaxPorts network uses. Generous on purpose: a slot's bounds are
// what a teardown sweeps, and an entity outside them would survive a rebuild as
// a ghost network nothing owns.
const (
	slotW    = 32
	slotH    = 72
	slotCols = 64
)

var (
	// hiddenIdx is the surface index, not a handle: handles do not survive a
	// save and the guest heap does. 0 means "not created yet".
	hiddenIdx uint32
	nextSlot  uint32 = 1
	freeSlots []uint32
)

func allocSlot() uint32 {
	if n := len(freeSlots); n > 0 {
		s := freeSlots[n-1]
		freeSlots = freeSlots[:n-1]
		return s
	}
	s := nextSlot
	nextSlot++
	return s
}

// releaseSlot returns a slot to the LIFO free list. LIFO rather than lowest-id
// for the same reason cluster.go's node free list is: it is deterministic
// without needing a sort.
func releaseSlot(s uint32) { freeSlots = append(freeSlots, s) }

func slotOrigin(s uint32) (int32, int32) {
	i := s - 1
	return int32(i%slotCols) * slotW, int32(i/slotCols) * slotH
}

func surfaceByIndex(i uint32) (fkapi.LuaSurface, bool) {
	o, err := fkapi.Game.GetSurface(fkapi.OfNumber(float64(i)))
	if err != nil || o == nil {
		return fkapi.LuaSurface{}, false
	}
	return fkapi.LuaSurface{Object: *o}, true
}

// hideFromAllForces withholds the hidden surface from every force's surface
// lists -- Space Age's remote view above all, which is where a player first
// meets a surface they were never meant to see (the mod's first field report,
// github.com/Techrocket9/BetterBeltBalancer issue #1).
//
// PER FORCE, because that is the only mechanism the engine has: visibility is
// `LuaForce::set_surface_hidden`, there is no surface-level flag at all
// (checked against both pinned runtime APIs, 2.0.77 and 2.1.17), and the
// default for a new force is VISIBLE. So this runs wherever the guest first
// puts its hands on the surface -- both discovery paths of hiddenSurface(),
// and rebuildFromWorld's walk for a save that predates the fix -- and
// onForceCreated covers a force born after all of those. Setting is idempotent
// and one call per force; reading first to avoid a redundant write would
// double the calls to save the engine a no-op.
//
// Never on a per-compile path: the fast path through hiddenSurface() returns
// before reaching this, so a session pays it once, at discovery.
func hideFromAllForces(s fkapi.LuaSurface) {
	forces, err := fkapi.Game.Forces()
	if err != nil {
		return
	}
	for i := range forces {
		_ = (fkapi.LuaForce{Object: forces[i].Val}).SetSurfaceHidden(s.Object, true)
	}
	if verboseLog {
		logStart("hidden surface withheld from the surface lists of ")
		logU(uint32(len(forces)))
		logS(" forces")
		logEnd()
	}
}

// hiddenSurface returns the one global hidden surface, creating it on first
// use.
//
// It is looked up by NAME before being created, which is not belt and braces:
// the surface lives in the save and the guest heap lives in the save, but a
// migration or a `--persist` change could leave one without the other, and
// create_surface on an existing name raises.
//
// No chunks are generated. Spike S1 measured entities on ungenerated chunks
// running at full rate, and generating a chunk per slot would be the single
// most expensive thing a compile did. `generate_with_lab_tiles` covers anything
// that does generate one -- a player teleporting there in the editor.
//
// `game.create_surface(name)` IS THE GENERATED CALL NOW, and the eighty-eight
// hand-written lines that used to be `guest/go/host.go` are deleted. The member
// was bound at the previous round (FKLUA-GAPS.md item 9) and still could not
// reach the engine, because an absent trailing optional was forwarded as an
// explicit nil the engine counts and rejects (item 16). `M.call` trims the
// forwarded arity to the last argument actually PRESENT now, so passing `nil`
// for `settings` makes the argument vanish rather than arrive as a nil --
// which is `game.create_surface(name)`, the call this mod has always wanted.
// Nothing a populated MapGenSettings would have said is wanted here anyway:
// `generate_with_lab_tiles`, set below as the bound attribute it is, says it.
func hiddenSurface() (fkapi.LuaSurface, bool) {
	if hiddenIdx != 0 {
		return surfaceByIndex(hiddenIdx)
	}
	if o, err := fkapi.Game.GetSurface(fkapi.OfString(hiddenSurfName)); err == nil && o != nil {
		s := fkapi.LuaSurface{Object: *o}
		if i, err := s.Index(); err == nil {
			hiddenIdx = i
			// A surface this heap did not know about is one whose visibility
			// this heap has not vouched for. Once per recovery, so the repair
			// is free where nothing needed repairing and cheap where it did.
			hideFromAllForces(s)
			return s, true
		}
	}
	o, err := fkapi.Game.CreateSurface(hiddenSurfName, nil)
	if err != nil || !o.Valid() {
		logError("could not create the hidden surface")
		return fkapi.LuaSurface{}, false
	}
	s := fkapi.LuaSurface{Object: o}
	_ = s.SetGenerateWithLabTiles(true)
	hideFromAllForces(s)
	i, err := s.Index()
	if err != nil {
		logError("hidden surface has no index")
		return s, false
	}
	hiddenIdx = i
	if verboseLog {
		logStart("hidden surface " + hiddenSurfName + " created, index=")
		logU(i)
		logEnd()
	}
	return s, true
}

// ---------------------------------------------------------------------------
// What the guest remembers about a compiled cluster
// ---------------------------------------------------------------------------

// netInfo is everything a teardown needs, and nothing else.
//
// The visible bounding box is stored rather than recomputed because a teardown
// often happens when the cluster it belonged to no longer exists -- the last
// part was mined -- and the visible linked belts still standing on those tiles
// have to be found somehow.
//
// fp is a fingerprint of the edge list, not the list itself. The list is a
// per-compile allocation and this guest's allocator never gives memory back
// (-gc=leaking), so a stored slice per cluster would be a leak that grows with
// play. A 64-bit collision would mean one missed rebuild, and it would be the
// SAME missed rebuild on every client, so it is not a desync.
type netInfo struct {
	fp             uint64
	slot           uint32
	surf           uint32
	force          uint32
	x0, y0, x1, y1 int32
	ents           uint32
}

// nets is point-queried only: NEVER RANGED OVER, never used to decide an order.
// Where something has to be done to every network -- the orphan sweep, the
// audit -- the walk is over cluster.go's node array in id order and this map is
// point-queried from it.
var nets map[uint32]netInfo

// ---------------------------------------------------------------------------
// Reusable buffers.
//
// -gc=leaking means an allocation per compile is an allocation per compile
// forever, and the guest heap is in every save and every multiplayer join.
// Everything below is allocated once and reused; the only growth is the first
// time a bigger cluster is compiled than any before it.
// ---------------------------------------------------------------------------

var (
	tileBuf []key
	edgeBuf []plan.Edge
	opBuf   []plan.Op
	entBuf  []fkapi.Object
	cq      []uint32

	kvBuf  [5]fkapi.KeyValue
	posBuf [2]fkapi.Value

	beltTypeVals [6]fkapi.Value
	beltTypes    fkapi.Value
	searchPos    fkapi.MapPosition
	searchArea   fkapi.BoundingBox
	nameFilter   fkapi.Value
	forceFilter  fkapi.Value

	// findByPos and findByName are prebuilt so that a query does not construct
	// a 320-byte filter struct from scratch each time.
	findByPos  fkapi.EntitySearchFilters
	findByName fkapi.EntitySearchFilters
)

func init() { initBuffers() }

// initBuffers builds, once, everything about a host call that a call cannot
// change: the compass out of the generated `defines.direction` accessors, the
// belt-type array every edge search filters on, the two reusable
// `EntitySearchFilters` and the scratch they read their arguments out of, and
// every constant key, tag and array identity in the `create_entity` table.
//
// It is a package initialiser and NOT called again. The header used to say
// ensureRegistry re-ran it on a heap another build wrote; that is initRegistry's
// job and only initRegistry's. Nothing here is state a save can carry -- a
// rebuilt guest runs its own package initialisers before any of this could be
// stale -- and re-running it would be free rather than wrong, which is why the
// claim survived unnoticed.
//
// See createArgs for what is left per call: six scalar stores.
func initBuffers() {
	// `defines.direction`, from the GENERATED accessors (FKLUA-GAPS.md item 11).
	// A define's number is Factorio's own and is not in the API description at
	// all, so nothing here writes one down: the accessor asks the running game
	// through a table that carries the dotted NAME, resolved once at load, and
	// caches on first use.
	//
	// EACH ACCESSOR IS CALLED DIRECTLY AND THAT IS THE POINT. FkLua prunes the
	// define table by scanning the guest for the constant ids that reach the
	// `fk.define` import, exactly as it prunes members and events; naming the
	// four accessors ships four paths out of 1137, and computing an id -- a
	// table of them, a loop, an offset -- would compile, would work, and would
	// silently ship all 1137. Same rule as an event id.
	//
	// Legal from a package initialiser: fk_mod.lua resolves the define table
	// when the control chunk loads, which is before _initialize runs, and
	// upstream binds the whole ABI before it "because a guest's package
	// initialisers can call the API".
	dirOf = [4]uint32{
		fkapi.DefinesDirectionNorth(),
		fkapi.DefinesDirectionEast(),
		fkapi.DefinesDirectionSouth(),
		fkapi.DefinesDirectionWest(),
	}
	// The planner is pure Go with no wasm imports -- that is what lets its
	// balance property be proved under an ordinary `go test` -- so the compass
	// is pushed into it rather than imported by it.
	plan.SetCompass(dirOf[0], dirOf[1], dirOf[2], dirOf[3])

	// THE EDGE CLASSIFIER HAS TWO GATES AND THIS IS THE FIRST ONE. The engine
	// applies this type list in C++ before `classifySide` sees anything, so a
	// family missing from it is not merely unclassified -- it is never returned,
	// and the switch below can never be reached for it. The two lists have to
	// agree, and there is nothing that makes them: adding "lane-splitter" to
	// classifySide alone changed exactly nothing, measured, on the run before
	// this line was added.
	beltTypeVals[0] = fkapi.OfString("transport-belt")
	beltTypeVals[1] = fkapi.OfString("underground-belt")
	beltTypeVals[2] = fkapi.OfString("splitter")
	beltTypeVals[3] = fkapi.OfString("lane-splitter")
	beltTypeVals[4] = fkapi.OfString("loader-1x1")
	beltTypeVals[5] = fkapi.OfString("loader")
	beltTypes = fkapi.Value{Tag: fkapi.TagArray, Array: beltTypeVals[:]}

	findByPos = fkapi.EntitySearchFilters{}
	findByPos.Position = &searchPos
	findByPos.Type = &beltTypes
	// The FORCE filter is applied by the engine in C++, so per-force edge
	// classification costs nothing: a belt belonging to another force is never
	// returned, and an interface is never placed where it could not connect.
	findByPos.Force = &forceFilter

	findByName = fkapi.EntitySearchFilters{}
	findByName.Area = &searchArea
	findByName.Name = &nameFilter

	// The create_entity argument table, everything about it that a call cannot
	// change. See createArgs: what is left per call is six scalar stores.
	kvBuf[argName].Key = fkapi.OfString("name")
	kvBuf[argName].Val.Tag = fkapi.TagString
	kvBuf[argPos].Key = fkapi.OfString("position")
	kvBuf[argPos].Val = fkapi.Value{Tag: fkapi.TagArray, Array: posBuf[:]}
	posBuf[0].Tag = fkapi.TagNumber
	posBuf[1].Tag = fkapi.TagNumber
	kvBuf[argDir].Key = fkapi.OfString("direction")
	kvBuf[argDir].Val.Tag = fkapi.TagNumber
	// A ForceID as an INDEX, not a name. The index is what the registry stores
	// (a name would be a string in every save and a string crossing on every
	// compile), and `force` accepts one.
	kvBuf[argForce].Key = fkapi.OfString("force")
	kvBuf[argForce].Val.Tag = fkapi.TagNumber
	// A linked belt's end type must be set AT CREATION. Changing it later is
	// only legal while disconnected and FLIPS THE DIRECTION 180 degrees (spike
	// S1), so the create-with-both form is the one with no order-of-operations
	// trap in it.
	kvBuf[argType].Key = fkapi.OfString("type")
	kvBuf[argType].Val.Tag = fkapi.TagString

	// The constant half of the draw_sprite table. What a call changes is the
	// sprite name, the target and the surface.
	arrowArgs = fkapi.LuaRenderingDrawSpriteArgs{}
	arrowArgs.OnlyInAltMode = &arrowAlt

	nets = map[uint32]netInfo{}
	hiddenIdx = 0
	nextSlot = 1
	freeSlots = freeSlots[:0]
	deadRoots = deadRoots[:0]
	liveRoots = liveRoots[:0]
	carryPools = carryPools[:0]
	carryItems = carryItems[:0]
	carryTake = carryTake[:0]
	carryClaims.Reset()
	buildNotes = buildNotes[:0]
	limPending = limPending[:0]
	addedTiles = addedTiles[:0]
	// The engine capability is re-derived rather than assumed: a fresh heap has
	// never asked, and the load hooks that can reach a DIFFERENT engine throw
	// the answer away again. See sedge.go.
	edgeMode = edgeModeUnchecked
	stranded = stranded[:0]
	limCands = limCands[:0]
	limCandMerge = limCandMerge[:0]
	overLimit = nil
	carryDepth = 0
	curPool = -1
	spillInit = false
	carryInit = false
	statCompiles, statSkipped, statBuilds, statTeardowns, statCreates = 0, 0, 0, 0, 0
}

// ---------------------------------------------------------------------------
// Counters.
//
// The one claim M3 makes that cannot be read off a balance measurement is HOW
// MANY TIMES a network was rebuilt for a given player action, so the compiler
// counts. `statSkipped` is the fingerprint doing its job: a compile that was
// asked for and found nothing to do.
// ---------------------------------------------------------------------------

var (
	statCompiles  uint32 // compile() entered
	statSkipped   uint32 // ... and the fingerprint said nothing moved
	statBuilds    uint32 // networks actually placed
	statTeardowns uint32 // networks actually removed
	statCreates   uint32 // create_entity calls that placed something
)

func logStats(what string) {
	if !verboseLog {
		return
	}
	logStart("stats ")
	logS(what)
	logS(" compiles=")
	logU(statCompiles)
	logS(" skipped=")
	logU(statSkipped)
	logS(" builds=")
	logU(statBuilds)
	logS(" teardowns=")
	logU(statTeardowns)
	logS(" creates=")
	logU(statCreates)
	logEnd()
	// The heap, on the same trigger and in the same shape under both build
	// variants. See gc.go: `sys=` is the number Factorio's worst tick is
	// 0.2 ms/MiB of, and it is the only cross-arm number there is.
	logHeap(what)
}

// ---------------------------------------------------------------------------
// THE REMOVAL WINDOW IS GONE, AND `fk.defer` IS WHY
//
// Recorded because deleting a safety mechanism deserves an argument, not a
// diff. A mined belt is STILL VALID during the event that reports it, so a
// recompile triggered from inside that event finds it and wires it straight
// back in. M2 could not defer -- deferring meant subscribing to on_tick
// forever, the one cost this architecture exists to avoid -- so `classifySide`
// carried a one-position blind spot, armed and restored by every entry into
// flushLive.
//
// `fk.Defer()` moves every per-entity recompile off the dispatch that reported
// the change and onto the next tick (FKLUA-GAPS.md item 12, fixed upstream).
// The engine destroys a mined entity when its own dispatch RETURNS, so by the
// time the flush runs there is nothing to ignore: the classifier re-reads the
// world and the belt is simply not in it. That also covers the case the window
// never could -- a mod raising `script_raised_destroy` and then NOT destroying,
// where the window would wrongly have ignored an entity that is still there.
//
// The synchronous flush paths that remain (a cloned area, a surface being
// deleted, an audit) are all events that report a REGION rather than an entity
// about to vanish, so none of them ever needed the window either.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Edge classification
// ---------------------------------------------------------------------------

// collectCluster gathers a cluster's tiles by flooding from its root.
//
// O(cluster), not O(map): scanning the node array would be correct but would
// make every recompile cost the size of the whole factory. The traversal order
// is the fixed direction list, so every client enumerates identically.
func collectCluster(root uint32) []key {
	tileBuf = tileBuf[:0]
	if root == 0 || int(root) >= len(alive) || !alive[root] {
		return tileBuf
	}
	gen++
	cq = cq[:0]
	cq = append(cq, root)
	mark[root] = gen
	// The FORCE check is not optional and its absence is invisible until two
	// forces build against each other: the flood would walk straight through the
	// boundary, and the cluster's bounding box -- which is what a teardown
	// sweeps -- would swallow the neighbour's visible interfaces. The neighbour
	// keeps working right up until it does not, because its own edge list never
	// changed and its fingerprint still matches.
	f := pforce[root]
	for i := 0; i < len(cq); i++ {
		k := ppos[cq[i]]
		tileBuf = append(tileBuf, k)
		for d := 0; d < len(dirs); d++ {
			nb, ok := index[key{k.s, k.x + dirs[d][0], k.y + dirs[d][1]}]
			if !ok || mark[nb] == gen || pforce[nb] != f {
				continue
			}
			mark[nb] = gen
			cq = append(cq, nb)
		}
	}
	return tileBuf
}

// dirOf maps the neighbour offsets in cluster.go's `dirs` to Factorio
// directions. The two lists are in the same order and must stay that way.
//
// Filled by initBuffers from the generated `defines.direction` accessors; see
// there for why it is not a table of literals.
var dirOf [4]uint32

// classifyEdges finds every belt-connectable pointing into or out of the
// cluster, one per boundary tile SIDE.
//
// The incumbent's accepted limitation is inherited by construction: a belt's
// `direction` is where it sends items, so a belt curving away at the edge is
// simply not pointing at us and is not an output.
//
// IT ALSO COUNTS THE EDGES PER TILE, into sedge.go's two package-level numbers,
// and that costs one integer per tile and nothing else: this walk already
// visits every tile and every side, so the answer to "does any part carry more
// than one belt" -- the rule Factorio 2.1 forces -- falls out of a pass that was
// happening anyway. No allocation and no extra host call, which is the gate the
// `mar` suite holds everything on this path to.
func classifyEdges(surf fkapi.LuaSurface, tiles []key, force uint32) []plan.Edge {
	edgeBuf = edgeBuf[:0]
	sedgeWorst, sedgeTiles = 0, 0
	// The engine filters by force for us. A belt of another force could never
	// have connected to an interface of ours, so classifying it as an edge
	// would place a linked belt that silently moves nothing.
	forceFilter = fkapi.OfNumber(float64(force))
	for i := range tiles {
		k := tiles[i]
		onTile := uint32(0)
		for d := 0; d < len(dirs); d++ {
			nx, ny := k.x+dirs[d][0], k.y+dirs[d][1]
			if _, ok := index[key{k.s, nx, ny}]; ok {
				continue // interior side
			}
			dir := dirOf[d]
			out, found := classifySide(surf, nx, ny, dir)
			if !found {
				continue
			}
			onTile++
			// An INPUT interface faces the incoming belt with its input side,
			// so it points the other way; an OUTPUT interface points at the
			// belt it feeds.
			ld := dir
			if !out {
				ld = plan.Opposite(dir)
			}
			edgeBuf = append(edgeBuf, plan.Edge{TileX: k.x, TileY: k.y, Dir: ld, Out: out})
		}
		if onTile > 1 {
			sedgeTiles++
			if onTile > sedgeWorst {
				sedgeWorst = onTile
			}
		}
	}
	return edgeBuf
}

// classifySide reports whether the tile at (tx, ty) holds something that feeds
// the cluster tile on its `dir` side, or is fed by it.
//
// `dir` points FROM the cluster tile TOWARDS this one, so a belt flowing
// Opposite(dir) is flowing into the cluster.
func classifySide(surf fkapi.LuaSurface, tx, ty int32, dir uint32) (out bool, found bool) {
	searchPos.X = float64(tx) + 0.5
	searchPos.Y = float64(ty) + 0.5
	ents, err := surf.FindEntitiesFiltered(findByPos)
	if err != nil {
		return false, false
	}
	back := plan.Opposite(dir)
	for i := range ents {
		e := fkapi.LuaEntity{Object: ents[i]}
		t, err := e.Type()
		if err != nil {
			continue
		}
		d, err := e.Direction()
		if err != nil {
			continue
		}
		switch t {
		case "transport-belt", "splitter", "lane-splitter":
			// All three are read the same way -- a direction and nothing else --
			// and they are together for two different reasons.
			//
			// A SPLITTER is two tiles wide and each half is its own edge; the
			// per-tile search finds it once from each of the cluster tiles it
			// touches, which is exactly the per-half behaviour wanted. (M2's
			// `spio` rig.)
			//
			// A LANE SPLITTER is 1x1, so nothing about halves applies to it. It
			// is here because it is a directional belt-connectable exactly as a
			// belt is, and because it was the one such family that could stand
			// against a balancer and be SILENTLY invisible: unnamed in this
			// switch, a cluster fed and drained entirely through them classifies
			// zero edges, compiles to nothing and delivers nothing -- while
			// reporting `drift=0 unbuilt=0`, because a fingerprint over an empty
			// edge list matches the world perfectly. Measured on the guest
			// before this line existed; M2's `lsio` rig is the red proof.
			if d == back {
				return false, true
			}
			if d == dir {
				return true, true
			}
		case "underground-belt":
			// Only the overground end exists as an entity, and which end it is
			// decides which way it can talk to us.
			bt, err := e.BeltToGroundType()
			if err != nil {
				continue
			}
			if bt == linkTypeOutput && d == back {
				return false, true
			}
			if bt == linkTypeInput && d == dir {
				return true, true
			}
		case "loader-1x1", "loader":
			lt, err := e.LoaderType()
			if err != nil {
				continue
			}
			if lt == linkTypeOutput && d == back {
				return false, true
			}
			if lt == linkTypeInput && d == dir {
				return true, true
			}
		}
	}
	return false, false
}

// fingerprint is FNV-1a over the edge list. See netInfo.fp for why a hash and
// not the list.
func fingerprint(edges []plan.Edge) uint64 {
	h := uint64(14695981039346656037)
	mix := func(v uint64) {
		for i := 0; i < 8; i++ {
			h ^= v & 0xff
			h *= 1099511628211
			v >>= 8
		}
	}
	for i := range edges {
		e := edges[i]
		mix(uint64(uint32(e.TileX)))
		mix(uint64(uint32(e.TileY)))
		b := uint64(e.Dir) << 1
		if e.Out {
			b |= 1
		}
		mix(b)
	}
	// The count, so that a prefix of the same list is a different fingerprint.
	mix(uint64(len(edges)))
	return h
}

// ---------------------------------------------------------------------------
// Building
// ---------------------------------------------------------------------------

// Fixed slots in kvBuf. The order is the order the table crosses in, and it is
// fixed so that every constant part of the table can be written ONCE, by
// initBuffers, and never again.
const (
	argName = iota
	argPos
	argDir
	argForce
	argType // last, because it is the only optional one
)

// createArgs builds the table for create_entity.
//
// It allocates nothing, and since the constant parts moved into initBuffers it
// also WRITES almost nothing: a `fkapi.Value` is a ~64-byte tagged union and a
// `KeyValue` is two of them, so the obvious spelling -- five composite literals
// per call -- copied ~640 bytes into a buffer whose keys and tags never differ
// between calls. What is left below is six scalar stores.
//
// The measured cost of a create_entity is elsewhere and this does not move it:
// a 4x4 recompile is ~350 host calls at ~12 us each, and that 12 us is the
// tier-2 encode on the LUA side (`read_dyn` walking the table this describes).
// See CLAUDE.md.
func createArgs(name string, x, y float64, dir uint32, ltype string, force uint32) fkapi.Value {
	posBuf[0].Number = x
	posBuf[1].Number = y
	kvBuf[argName].Val.Str = name
	kvBuf[argDir].Val.Number = float64(dir)
	kvBuf[argForce].Val.Number = float64(force)
	n := argType
	if ltype != "" {
		kvBuf[argType].Val.Str = ltype
		n++
	}
	return fkapi.Value{Tag: fkapi.TagMap, Map: kvBuf[:n]}
}

func linkName(l plan.LinkType) string {
	switch l {
	case plan.LinkInput:
		return linkTypeInput
	case plan.LinkOutput:
		return linkTypeOutput
	}
	return ""
}

// compile (re)builds one cluster's network.
//
// Returns false only for a real failure, which is always logged loudly: a
// half-built network moves items into a hole, so the failure path destroys
// everything it placed and leaves the cluster with no network at all.
func compile(root uint32) bool {
	statCompiles++
	tiles := collectCluster(root)
	if len(tiles) == 0 {
		return true
	}
	force := pforce[root]
	surf, ok := surfaceByIndex(tiles[0].s)
	if !ok {
		return false
	}
	edges := classifyEdges(surf, tiles, force)
	fp := fingerprint(edges)

	ni, hadNet := nets[root]
	if hadNet && ni.fp == fp && ni.force == force {
		statSkipped++
		forgetOverLimit(root)
		return true // nothing that matters changed
	}

	// THE ORDERING CARVE-OUT, AND IT IS THE ONE PLACE A REFUSAL DEMOLISHES
	// ANYTHING. Every refusal below this line leaves the standing network alone,
	// because the machine is fine and only the requested EDIT is not -- that is
	// the sixty-fifth belt's whole fix, and the `sedge` suite's `sbld` rig asserts
	// it. A CONDEMNED cluster is the opposite case: what is standing was built
	// with two belts on one part on an engine that no longer permits it, so the
	// machine itself is the thing that cannot exist, and leaving it up is a latent
	// engine risk on every load rather than a degraded balancer. It comes down
	// first, and the refusal that follows claims nothing -- so the pool the
	// teardown opened settles onto the ground, which is what this mod does with a
	// REMOVAL's items, and a machine that cannot exist any more is a removal.
	//
	// Only rebuildFromWorld and the setting-flip sweep condemn, so this is a scan
	// of an empty slice in every save that was built under the rule it is running
	// under. BOTH of them INVERT the stored fingerprint before they condemn, and
	// that is what carries a condemned cluster past the skip above: flipping the
	// setting moves nothing in the world, so nothing else would.
	//
	// `hadNet` goes false with it, for two reasons: the teardown further down must
	// not run a second time, and a compile that SUCCEEDS after this (the setting
	// flipped back on) has to draw from the `owned` pool this just opened, which
	// takeCarry matches by root rather than by geometry. See sedge.go.
	if hadNet && takeCondemned(root) {
		teardownForRebuild(root)
		hadNet = false
	}

	// THE PORT LIMIT IS ASKED HERE, IN FRONT OF THE TEARDOWN, AND THAT ORDER IS
	// THE WHOLE OF THE 2026-08-04 FIX. `plan.Shape` needs the edge COUNTS and
	// nothing else, and they are in hand the moment classifyEdges returns -- so
	// the answer has been available at this point all along and was being asked
	// twenty lines further down, where `plan.Build` reports it. By then
	// `teardownForRebuild` had already demolished a working network, and the
	// refusal that followed built nothing to put the items back into: the
	// sixty-fifth belt against a 64-port balancer emptied the machine onto the
	// floor and mentioned it only in the log.
	//
	// Refusing here leaves the standing network standing. The belt that could
	// not be joined is inert -- the same degradation as a belt this guest never
	// heard about (CLAUDE.md, "The failure envelope") -- and limit.go tells the
	// player and offers the piece back. See agents/maxports.md §4.
	if pt, over := overLimitShape(edges); over {
		refuseOverLimit(root, fp, pt, tiles, force)
		return false
	}
	// AND THE ONE-BELT-PER-PART RULE IS ASKED IN THE SAME PLACE, for the same
	// reason and through the same machinery (sedge.go, limit.go). The port limit
	// goes first because its answer does not depend on the mode -- a cluster
	// past sixty-four ports is past it whichever geometry built it -- so a
	// player whose balancer breaks both bounds is told about the one that is
	// true on every engine.
	//
	// The counts come out of classifyEdges above and nothing between here and
	// there re-classifies.
	if worst, over := multiEdgeShape(); over {
		refuseSingleEdge(root, fp, worst, tiles, force)
		return false
	}
	forgetOverLimit(root)

	if hadNet {
		// The recompile proper, and its drain is this cluster's OWN. It opens a
		// pool nothing else may claim (carry.go, `owned`), which is also why
		// claimCarry skipped this root: a cluster that had a network is its own
		// successor and nobody else's.
		teardownForRebuild(root)
	}
	if len(edges) == 0 {
		return true
	}

	// The visible bounding box, for the teardown that will one day have to find
	// these interfaces without the cluster to guide it -- and, since 2026-08-02,
	// for deciding which drained items this network inherits.
	x0, y0, x1, y1 := clusterBox(tiles)

	slot := allocSlot()
	ox, oy := slotOrigin(slot)
	ops, pt, fits := plan.Build(opBuf, edges, ox, oy)
	opBuf = ops
	// UNREACHABLE SINCE THE CHECK MOVED IN FRONT OF THE TEARDOWN, and kept as an
	// `error:` precisely because of that: `overLimitShape` above mirrors
	// plan.Build's own two tests, so getting here means the mirror has cracked
	// -- and by then a working network really has been demolished for nothing.
	// `test/run.sh` fails any run that produces one of these.
	if !fits {
		logErrStart("cluster ")
		logU(root)
		logS(" needs ")
		logU(uint32(pt.P))
		logS(" ports, over the limit of ")
		logU(plan.MaxPorts)
		logS("; not compiled, and the early refusal did not catch it")
		logEnd()
		releaseSlot(slot)
		return false
	}
	// THE OTHER UNREACHABLE BACKSTOP, and the twin of the one above. plan.Build
	// has no test to mirror for the one-belt-per-part rule -- it would happily
	// emit a network for a multi-edge cluster and the engine would refuse every
	// second interface with a silent nil -- so the mirror is of the
	// CLASSIFICATION rather than of the planner, and it is asked here, where a
	// working network has already been demolished, for the same reason: getting
	// here means the check in front of the teardown did not agree with this one.
	// See sedge.go.
	if singleEdgeBackstop(root) {
		releaseSlot(slot)
		return false
	}
	if len(ops) == 0 {
		// Inputs but no outputs, or the other way round. Legitimate.
		releaseSlot(slot)
		return true
	}

	hid, ok := hiddenSurface()
	if !ok {
		releaseSlot(slot)
		return false
	}

	if !execute(ops, surf, hid, force) {
		releaseSlot(slot)
		return false
	}

	statBuilds++
	nets[root] = netInfo{fp: fp, slot: slot, surf: tiles[0].s, force: force,
		x0: x0, y0: y0, x1: x1, y1: y1, ents: uint32(len(ops))}
	// The network is standing and empty. Whatever the teardowns of this flush
	// drained out of the network(s) this one succeeds goes back INSIDE it --
	// entBuf is parallel to ops, so the fill order is plan order and needs no
	// second look at the world. See carry.go.
	takeCarry(root, tiles[0].s, force, x0, y0, x1, y1, hadNet, ops, entBuf)
	if verboseLog {
		logStart("compiled cluster ")
		logU(root)
		logS(" ")
		logU(uint32(pt.N))
		logS("->")
		logU(uint32(pt.M))
		logS(" over ")
		logU(uint32(pt.P))
		logS(" ports, ")
		logU(uint32(len(ops)))
		logS(" entities, slot ")
		logU(slot)
		logEnd()
	}
	return true
}

// execute walks the plan. Two passes: create everything, then connect the
// linked belts, because a pair cannot be connected until both ends exist.
//
// create_entity of a COLLIDING belt-connectable returns nil silently -- no
// error, no log -- so every create is checked. There is no "mostly built"
// state: the first nil unwinds the whole network.
func execute(ops []plan.Op, vis, hid fkapi.LuaSurface, force uint32) bool {
	entBuf = entBuf[:0]
	for i := range ops {
		o := &ops[i]
		s := hid
		if o.Visible {
			s = vis
		}
		h, err := s.CreateEntity(createArgs(protoName(o.Proto), o.X, o.Y, o.Dir, linkName(o.Link), force))
		if err != nil || h == nil {
			logErrStart("create_entity returned nil for ")
			logS(protoName(o.Proto))
			logS(" at ")
			logF1(o.X)
			logS(",")
			logF1(o.Y)
			logS(" (op ")
			logU(uint32(i))
			logS(" of ")
			logU(uint32(len(ops)))
			logS(") -- aborting this cluster")
			logEnd()
			unwind()
			return false
		}
		statCreates++
		entBuf = append(entBuf, *h)
		if o.Visible {
			drawArrow(vis, *h, o)
		}
	}
	for i := range ops {
		if ops[i].Link != plan.LinkInput || ops[i].Pair < 0 {
			continue
		}
		e := fkapi.LuaEntity{Object: entBuf[i]}
		nb := entBuf[ops[i].Pair]
		if err := e.ConnectLinkedBelts(&nb); err != nil {
			logErrStart("connect_linked_belts failed at op ")
			logU(uint32(i))
			logS(" -- aborting this cluster")
			logEnd()
			unwind()
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// The I/O arrows
//
// One sprite per visible interface, drawn ON that interface entity, saying
// which way items move across that edge. Alt-mode only, which is Factorio's own
// convention for an informational overlay.
//
// THE ONLY REASON THIS IS AFFORDABLE IS THE TARGET. A rendering object whose
// target ENTITY is destroyed is destroyed with it (2.0.77 runtime doc,
// `ScriptRenderTarget`), and the entity here is the visible linked belt the
// teardown already sweeps. So there is no id to store, no per-cluster list in
// the guest heap, and no teardown path to get wrong -- the arrows come down
// with the network by construction, on every recompile, every clone reconcile
// and every surface deletion, including the ones nobody thought about.
//
// A network adopted by rebuildFromWorld keeps the arrows the previous session
// drew, for the same reason: the interfaces they hang on were not touched.
// ---------------------------------------------------------------------------

var (
	// By the SIDE the belt is on, in cluster.go's `dirs` order (N E S W). The
	// prototypes carry the rotation and the shift; see mod-data/prototypes/
	// sprite.lua for why that is eight prototypes and not one.
	arrowIn  = [4]string{"bbb-arrow-in-n", "bbb-arrow-in-e", "bbb-arrow-in-s", "bbb-arrow-in-w"}
	arrowOut = [4]string{"bbb-arrow-out-n", "bbb-arrow-out-e", "bbb-arrow-out-s", "bbb-arrow-out-w"}

	arrowArgs fkapi.LuaRenderingDrawSpriteArgs
	arrowAlt  = true
)

// dirIndex inverts dirOf: a Factorio direction back to its slot in `dirs`.
//
// Four comparisons, and it is what lets this file pick a sprite without knowing
// what number `defines.direction.north` has -- the same reason plan.Opposite is
// a lookup on the installed compass rather than arithmetic.
func dirIndex(d uint32) (int, bool) {
	for i := 0; i < 4; i++ {
		if dirOf[i] == d {
			return i, true
		}
	}
	return 0, false
}

// drawArrow puts the I/O indicator on one visible interface.
//
// The op's direction is the direction ITEMS MOVE, so an output interface points
// at the belt it feeds (the side the belt is on) and an input interface points
// away from the belt that feeds it (the opposite side).
func drawArrow(vis fkapi.LuaSurface, e fkapi.Object, o *plan.Op) {
	i, ok := dirIndex(o.Dir)
	if !ok {
		return
	}
	switch o.Link {
	case plan.LinkOutput:
		arrowArgs.Sprite = arrowOut[i]
	case plan.LinkInput:
		arrowArgs.Sprite = arrowIn[(i+2)&3]
	default:
		return
	}
	arrowArgs.Target = fkapi.OfObject(e)
	arrowArgs.Surface = vis.Object
	// The return is a LuaRenderObject and it is deliberately dropped: keeping it
	// would be state to persist, invalidate and sweep, and the target is what
	// owns its lifetime.
	if _, err := fkapi.Rendering.DrawSprite(arrowArgs); err != nil {
		logErrStart("could not draw the I/O arrow for ")
		logS(arrowArgs.Sprite)
		logEnd()
	}
}

// unwind destroys everything execute placed. Handles are transient -- the host
// drops them when this dispatch returns -- so they are still live here.
func unwind() {
	for i := range entBuf {
		e := fkapi.LuaEntity{Object: entBuf[i]}
		_, _ = e.Destroy(fkapi.LuaEntityDestroyArgs{})
	}
	entBuf = entBuf[:0]
}

// ---------------------------------------------------------------------------
// Teardown
// ---------------------------------------------------------------------------

// teardown removes a cluster's whole network and takes its items out of it.
//
// ITEMS IN THE NETWORK ARE NOT DELETED. The incumbent has no equivalent bug to
// forgive us with: it never had a hidden network to lose items in. Every
// transport line in the slot is read before the entity holding it is destroyed,
// and the total goes into a carry pool.
//
// WHERE THAT POOL ENDS UP IS carry.go's DECISION AND IT IS NOT "THE GROUND".
// Inside a flush -- which is every recompile, every merge and every split -- the
// network that goes up in the same flush takes the items back; only a real
// removal spills beside the cluster. What is NOT recovered either way is the
// fractional item positions, the items inside a splitter's own internal buffer
// beyond its transport lines, and how the items were STACKED on the line.
func teardown(root uint32) { teardownNet(root, true, false) }

// teardownForRebuild is the same teardown, marked as compile()'s own: the pool
// it opens belongs to `root` and no geometric successor may claim it.
func teardownForRebuild(root uint32) { teardownNet(root, true, true) }

// forgetNet is teardown for a network whose HIDDEN half no longer exists --
// the surface under it was deleted or cleared by another mod. The visible
// interfaces are still standing and still have to come down, items and all,
// or the rebuild collides with them and aborts.
func forgetNet(root uint32) { teardownNet(root, false, false) }

func teardownNet(root uint32, hiddenHalf, owned bool) {
	ni, ok := nets[root]
	if !ok {
		return
	}
	delete(nets, root)
	// The over-limit feedback gate goes with the network: a cluster whose
	// network has come down is having its state rewritten wholesale, and the
	// fingerprint it was last refused on describes a machine that no longer
	// exists. See limit.go.
	forgetOverLimit(root)
	openPool(root, ni, owned)
	statTeardowns++

	// slot 0 means "we know the interfaces are there but not which slot they
	// lead to" -- rebuildFromWorld's fallback, when a linked belt's partner
	// could not be followed. The visible half still has to come down; the
	// hidden half becomes an orphan for sweepOrphanSlots to find.
	sweptHidden := false
	if hid, ok := hiddenSurface(); ok && hiddenHalf && ni.slot != 0 {
		ox, oy := slotOrigin(ni.slot)
		sweep(hid, ox, oy, ox+slotW-1, oy+slotH-1, true)
		sweptHidden = true
	}
	vis, haveVis := surfaceByIndex(ni.surf)
	if haveVis {
		sweep(vis, ni.x0, ni.y0, ni.x1, ni.y1, true)
	}
	// THE SLOT GOES BACK ON THE FREE LIST ONLY ONCE WHAT WAS STANDING IN IT IS
	// GONE. `hiddenSurface()` can fail -- create_surface refused, or the surface
	// was deleted under us between one call and the next -- and the old
	// unconditional release handed the next compile a slot with a live lane
	// splitter still in it. That is not a cosmetic collision: the rebuild would
	// place its own interfaces on top, `sweepOrphanSlots` would see one slot
	// claimed twice, and a later teardown would spill one network's items for
	// the other. `forgetNet` is the case where there is nothing to destroy --
	// the hidden surface itself is gone, and with it everything on it -- so it
	// releases as before.
	//
	// The failure mode of NOT releasing is a leaked slot, which
	// `sweepOrphanSlots` rebuilds the allocator around on the next
	// rebuild-from-world, and which costs 32x72 tiles of a surface nobody can
	// reach. That is the right side to be wrong on.
	if ni.slot != 0 && (sweptHidden || !hiddenHalf) {
		releaseSlot(ni.slot)
	}

	// The line says what came OUT of the network, which is what it always said;
	// where it goes is decided afterwards, by the flush this teardown is part of.
	if total := closePool(); total > 0 && verboseLog {
		logStart("torn down cluster ")
		logU(root)
		logS(", returned ")
		logU(total)
		logS(" items")
		logEnd()
	}
}

// sweep drains and destroys every one of our entities in a box.
//
// Four queries by NAME rather than one unfiltered one, because the alternative
// is reading `entity.name` per entity -- a string return, the most expensive
// thing in the ABI -- and because knowing the prototype is what says how many
// transport lines to look for.
// `give` is false for the one case where the items must NOT come back: an area
// clone that copied our interfaces copied their contents too, so handing those
// back would mint matter out of nothing. Everywhere else it is true.
var sweepNames = [4]string{nameBelt, nameLinkedBelt, nameLaneSplit, nameSplitter}

// setSearchBox turns INCLUSIVE tile bounds into the search box for them, inset
// by a tenth of a tile on every side.
//
// The inset is not cosmetic and it cost a full M3 test run to find.
// `find_entities_filtered` returns everything whose bounding box TOUCHES the
// area, and a 1x1 entity on tile n occupies exactly [n, n+1] -- so the obvious
// box, left_top = (x0, y0) and right_bottom = (x1+1, y1+1), also returns
// everything on tiles x1+1 and y1+1. A teardown of one cluster then destroys
// the visible interfaces of the cluster on the very next tile, which keeps
// working right up until it does not: the victim's edge list has not changed,
// so its fingerprint still matches and it never rebuilds.
//
// Two clusters ARE adjacent whenever two forces build against each other (parts
// of different forces never merge), and diagonally whenever anyone builds an L
// around a corner. 0.1 is comfortably inside any tile and comfortably outside
// the neighbouring one.
func setSearchBox(x0, y0, x1, y1 int32) {
	searchArea.LeftTop.X, searchArea.LeftTop.Y = float64(x0)+0.1, float64(y0)+0.1
	searchArea.RightBottom.X, searchArea.RightBottom.Y = float64(x1)+0.9, float64(y1)+0.9
}

func sweep(s fkapi.LuaSurface, x0, y0, x1, y1 int32, give bool) uint32 {
	setSearchBox(x0, y0, x1, y1)
	killed := uint32(0)
	for n := range sweepNames {
		nameFilter = fkapi.OfString(sweepNames[n])
		ents, err := s.FindEntitiesFiltered(findByName)
		if err != nil {
			continue
		}
		for i := range ents {
			e := fkapi.LuaEntity{Object: ents[i]}
			if give {
				drain(e)
			}
			_, _ = e.Destroy(fkapi.LuaEntityDestroyArgs{})
			killed++
		}
	}
	return killed
}

// drain reads every transport line an entity has into the spill tally.
//
// The line count is ASKED FOR rather than hardcoded per prototype: a belt has
// 2, a splitter 8, a lane splitter its own number, and a constant that was
// wrong by one would silently delete items on every recompile. It is also
// cheaper than probing until the index raises -- a raise inside the host's pcall
// costs an error object and a formatted message.
func drain(e fkapi.LuaEntity) {
	n, err := e.GetMaxTransportLineIndex()
	if err != nil || n == 0 || n > maxLinesToProbe {
		return
	}
	// The stacking gate, once per force per carry transaction (carry.go). False
	// for every removal path and for every force that cannot stack, which is all
	// of base Factorio, and then this loop is what it always was.
	stacked := stacksPossible(e)
	for i := uint32(1); i <= n; i++ {
		l, err := e.GetTransportLine(i)
		if err != nil {
			return
		}
		line := fkapi.LuaTransportLine{Object: l}
		// The cheap numeric question first: most lines in an idle network are
		// empty, and get_contents allocates.
		total, err := line.GetItemCount(nil)
		if err != nil || total == 0 {
			continue
		}
		// ...and the `Into` form for the ones that are not, which is the only
		// allocation on this path that BBB owns. `GetContents` is
		// `make([]ItemWithQualityCount, n)` per line; a full 4x4 has ~40
		// non-empty lines and that was ~1.6 KB of transient per teardown. It
		// mattered more once a recompile started REINSERTING, because a network
		// that was handed its items back is full again the next time it comes
		// down, so far more lines take this branch than used to.
		//
		// ALWAYS USE THE RETURN VALUE: it aliases the buffer when it fit and is a
		// fresh allocation when it did not, and dropping it would leak the growth
		// on every call.
		items, err := line.GetContentsInto(drainBuf)
		if err != nil {
			continue
		}
		drainBuf = items
		if stacked && detailedTally(line, total, items) {
			continue
		}
		for j := range items {
			tally(items[j].Name, items[j].Quality, 1, items[j].Count)
		}
	}
}

// detailedTally reads one line POSITION BY POSITION so that the stack sizes
// survive the recompile, and reports whether it accounted for every item on it.
//
// FALSE MEANS "THE FLAT TOTALS ARE STILL UNTALLIED", and drain falls back to
// them. That is what makes this safe to be clever in: conservation never depends
// on any of the reasoning below, because the cheap path is exact and is the
// fallback for every case this one cannot resolve. The worst a mistake here can
// do is hand the items back unstacked, which is what the whole mod did until
// today.
//
// The host calls, in the order they are worth avoiding:
//
//   - `get_detailed_contents` is one call per non-empty line, and it is the
//     whole cost on a line that turns out NOT to be stacked -- which is most of
//     them even on a stacking force, because only the belts downstream of a
//     stacking loader carry stacks. len(det) == total means every position holds
//     exactly one item; the flat path already has the right answer and is
//     cheaper, so it takes it.
//   - a position's `count` is then one call each, and it is the only one that is
//     unavoidable: the stack size is the thing being recovered.
//   - a position's NAME is not read at all when the line carries one kind, which
//     is the ordinary case and the reason this is arranged around the totals.
//     Reading it would also be the only ALLOCATION on the path -- `getStr`
//     copies the host's bytes -- so the multi-kind branch identifies a position
//     with `name_is`, a bool return over a string this guest already holds,
//     rather than with `name`.
//   - a position's QUALITY is read only when one name on the line carries more
//     than one of them, which is rarer still.
func detailedTally(line fkapi.LuaTransportLine, total uint32, totals []fkapi.ItemWithQualityCount) bool {
	det, err := line.GetDetailedContentsInto(detailBuf)
	if err != nil {
		return false
	}
	detailBuf = det
	if len(det) == 0 || len(totals) == 0 || uint32(len(det)) == total {
		return false
	}
	lineGroups = lineGroups[:0]
	got := uint32(0)
	for i := range det {
		st := fkapi.LuaItemStack{Object: det[i].Stack}
		c, err := st.Count()
		// A stack of more than 255 cannot be handed to `insert_at`, whose
		// `belt_stack_size` is a uint8. Nothing in the game makes one; a mod
		// could, and the honest answer is to give the whole line back flat.
		if err != nil || c == 0 || c > 255 {
			return false
		}
		k, ok := kindAt(st, totals)
		if !ok {
			return false
		}
		addLineGroup(totals[k].Name, totals[k].Quality, uint8(c), c)
		got += c
	}
	// The self-check that makes the fallback meaningful: the positions have to
	// add up to what get_item_count said before anything is tallied.
	if got != total {
		return false
	}
	for i := range lineGroups {
		tally(lineGroups[i].name, lineGroups[i].quality, lineGroups[i].stack, lineGroups[i].count)
	}
	return true
}

// kindAt says which of the line's (name, quality) totals a position belongs to.
//
// One candidate means no host call at all. Otherwise it is `name_is` per
// candidate until one answers, and `quality` only to break a tie between two
// entries of the SAME name -- which is the only thing the name cannot settle.
func kindAt(st fkapi.LuaItemStack, totals []fkapi.ItemWithQualityCount) (int, bool) {
	if len(totals) == 1 {
		return 0, true
	}
	first := -1
	for i := range totals {
		if askedAlready(totals, i) {
			continue
		}
		ok, err := st.NameIs(totals[i].Name)
		if err != nil {
			return 0, false
		}
		if ok {
			first = i
			break
		}
	}
	if first < 0 {
		return 0, false
	}
	n := 0
	for i := range totals {
		if totals[i].Name == totals[first].Name {
			n++
		}
	}
	if n == 1 {
		return first, true
	}
	q, err := st.Quality()
	if err != nil {
		return 0, false
	}
	qp := fkapi.LuaQualityPrototype{Object: q}
	for i := first; i < len(totals); i++ {
		if totals[i].Name != totals[first].Name {
			continue
		}
		ok, err := qp.NameIs(totals[i].Quality)
		if err != nil {
			return 0, false
		}
		if ok {
			return i, true
		}
	}
	return 0, false
}

// askedAlready reports whether an earlier entry carries the same NAME, so that
// two qualities of one item cost one `name_is` between them rather than two.
func askedAlready(totals []fkapi.ItemWithQualityCount, i int) bool {
	for j := 0; j < i; j++ {
		if totals[j].Name == totals[i].Name {
			return true
		}
	}
	return false
}

// addLineGroup merges one position into the line's own group list. It is
// separate from tally() because nothing may reach the pool until the whole line
// has added up -- see detailedTally.
func addLineGroup(name, quality string, stack uint8, count uint32) {
	for i := range lineGroups {
		if lineGroups[i].name == name && lineGroups[i].quality == quality &&
			lineGroups[i].stack == stack {
			lineGroups[i].count += count
			return
		}
	}
	lineGroups = append(lineGroups, carryItem{name: name, quality: quality, count: count, stack: stack})
}

// drainBuf is drain()'s reusable contents buffer. One transport line's worth;
// the only growth is the first time a fuller line is read than any before it.
// detailBuf and lineGroups are the stacking path's equivalents, and they are
// never touched at all below the gate.
var (
	drainBuf   []fkapi.ItemWithQualityCount
	detailBuf  []fkapi.DetailedItemOnLine
	lineGroups []carryItem
)

var (
	spillKV     [3]fkapi.KeyValue
	spillArgs   fkapi.LuaSurfaceSpillItemStackArgs
	spillRad    = 12.0
	spillLooted = false
	// Item by item, not whole stacks. `drop_full_stack = true` was measured
	// putting NOTHING on the ground at all for a 72-item spill; false places
	// each item and lets the engine find room for it.
	spillFull     = false
	spillFallback = true
	spillInit     bool
)

func spill(s fkapi.LuaSurface, x, y float64, name, quality string, count uint32) {
	if !spillInit {
		spillArgs.MaxRadius = &spillRad
		spillArgs.EnableLooted = &spillLooted
		spillArgs.DropFullStack = &spillFull
		spillArgs.UseStartPositionOnFailure = &spillFallback
		spillInit = true
	}
	spillKV[0] = fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(name)}
	spillKV[1] = fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(float64(count))}
	n := 2
	if quality != "" {
		spillKV[2] = fkapi.KeyValue{Key: fkapi.OfString("quality"), Val: fkapi.OfString(quality)}
		n = 3
	}
	spillArgs.Position.X, spillArgs.Position.Y = x, y
	spillArgs.Stack = fkapi.Value{Tag: fkapi.TagMap, Map: spillKV[:n]}
	if _, err := s.SpillItemStack(spillArgs); err != nil {
		logErrStart("could not return ")
		logU(count)
		logS(" ")
		logS(name)
		logS(" to the world")
		logEnd()
	}
}

// ---------------------------------------------------------------------------
// The recompile queue
//
// One event can touch several clusters -- a split makes up to four -- and every
// teardown has to happen before every build. These two lists are what enforces
// that.
//
// SINCE `fk.defer` THEY ARE DRAINED ONCE PER TICK rather than at the end of the
// event that filled them, so the ordering guarantee is now tick-wide instead of
// event-wide: every teardown for everything that happened in a tick runs before
// any build for any of it. A blueprint paste is P separate dispatches, and the
// whole point of batching is that P of them cost one drain.
//
// The list of events that still drain SYNCHRONOUSLY is short and each is there
// for a reason the deferral cannot satisfy: `on_pre_surface_deleted` and
// `on_pre_surface_cleared` (the entities are valid only inside that event, and
// the items in them have to be handed back before the surface goes), the two
// clone events (the copies the clone made must not stand for a tick), the audit
// marker (it reports what it found, so it has to have done the work), and
// rebuildFromWorld.
// ---------------------------------------------------------------------------

var (
	deadRoots []uint32
	liveRoots []uint32
)

// requestFlush queues the drain for the next tick.
//
// Idempotent within a tick by construction -- the host registers one one-shot
// on_tick handler however many times this is called and tears it down again
// from inside the flush -- so an idle guest still pays zero registrations and
// zero per-tick calls, which is the M4 measurement this must not break.
func requestFlush() { fk.Defer() }

func markDead(r uint32) {
	for i := range deadRoots {
		if deadRoots[i] == r {
			return
		}
	}
	deadRoots = append(deadRoots, r)
}

func markLive(r uint32) {
	for i := range liveRoots {
		if liveRoots[i] == r {
			return
		}
	}
	liveRoots = append(liveRoots, r)
}

// flushDead runs every queued teardown and nothing else.
//
// Separated from the build half because the reconcile paths (a cloned area, a
// deleted surface) have work to do BETWEEN the two: an area clone has to bring
// the legitimate networks down -- giving their items back -- before it destroys
// the interfaces the clone duplicated, and only then may anything be rebuilt.
// AND IT IS WHERE A MERGE PAST THE PORT LIMIT IS REFUSED. compile()'s own check
// is in front of its own teardown and that is no help at all here: a bridging
// part queues BOTH predecessors' roots dead, so without this pass two working
// balancers are demolished before flushLive discovers that what they became
// cannot be built. `spareOverLimitMerges` takes those teardowns back off the
// queue -- for a merge and never for anything else -- and costs a `find` per
// queued root and no host call at all below seventeen parts. See limit.go.
func flushDead() {
	spareOverLimitMerges()
	for i := range deadRoots {
		teardown(deadRoots[i])
	}
	deadRoots = deadRoots[:0]
}

// flushLive builds every queued cluster.
//
// EVERY QUEUED ROOT IS RE-RESOLVED FIRST, and that is not tidiness -- it is what
// makes a deferred drain correct. `markLive` is given the root a cluster had at
// the moment of the event, and a tick can contain several events: two parts
// placed next to each other merge, and the root queued by the first is a node
// that is no longer a root by the time the flush runs. `collectCluster` floods
// from whatever node it is given, so compiling a stale root would build the
// merged cluster's network under the wrong key -- and then compiling the real
// root would find no netInfo and build a SECOND one on top of it.
//
// A node that has been freed since (its part was mined in the same tick) is
// dropped: whatever cluster its removal left behind queued itself.
func flushLive() {
	n := 0
	for i := range liveRoots {
		r := liveRoots[i]
		if int(r) >= len(alive) || !alive[r] {
			continue
		}
		r = find(r)
		dup := false
		for j := 0; j < n; j++ {
			if liveRoots[j] == r {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		liveRoots[n] = r
		n++
	}
	liveRoots = liveRoots[:n]
	// Before the first build, and only when a teardown left something behind:
	// how many of these clusters descend from each pending pool. A split cannot
	// divide its predecessor's items fairly unless the first successor already
	// knows the second exists. No host call; see carry.go.
	claimCarry(liveRoots)
	for i := range liveRoots {
		// The sprite before the network, and both from the same re-resolved
		// root. restyle() is free -- not one host call -- for a cluster whose
		// SHAPE did not change, which is every cluster queued by a belt being
		// laid nearby, so this is not a second cost on the common path. See
		// skin.go.
		restyle(liveRoots[i])
		compile(liveRoots[i])
	}
	liveRoots = liveRoots[:0]
}

// flush is the only place a network is created or destroyed.
//
// Called from fk_on_deferred for everything the per-entity events queued during
// a tick, and directly from the handful of paths that cannot wait (see the
// queue's comment above).
//
// THE CARRY TRANSACTION SPANS BOTH HALVES, and that is what makes a recompile a
// recompile: the items a teardown drains are held until the builds have run and
// then go back into the networks that succeeded the ones they came out of.
// Whatever nobody claimed spills when the transaction closes. The three
// reconcile paths that split their own flush in half open the transaction
// themselves and the nesting counter absorbs this one.
func flush() {
	// AN INCUMBENT'S PART THAT ARRIVED THROUGH A BUILD EVENT, swapped for one of
	// ours before anything else happens -- so the AddPart it makes is compiled by
	// this same drain rather than by the next one, including on the synchronous
	// drain a `bbb-audit` marker forces, which is the only one a `--create` ever
	// reaches. One length test on an empty slice in every save that never had a
	// Belt Balancer 2 in it. See legacy.go, which is emphatic about why this is
	// deferred rather than done inside the event.
	legacyRunBuilds()
	beginCarry()
	flushDead()
	flushLive()
	endCarry()
	// AND THE OVER-LIMIT REVERT RUNS HERE, AFTER THE DRAIN AND AFTER THE
	// TRANSACTION HAS CLOSED, WHICH IS THE ONLY PLACE IT MAY. `mine_entity`
	// dispatches on_player_mined_entity synchronously, so it re-enters this
	// guest through the registry -- freeing nodes, re-rooting components and
	// filling the queues this flush has just emptied. Doing that inside the
	// drain would be mutating the queue being iterated; doing it before
	// endCarry would file the mine's claim against pools it has nothing to do
	// with. limit.go's header spells out all three reasons. One branch on an
	// empty slice when nobody built anything over the limit, which is every
	// tick of every game that never reaches 64 belts.
	revertOverLimit()
	// The insert probe runs HERE, after the transaction has settled, because
	// that is where the miner's pocket runs -- see probe.go. One branch on an
	// empty slice otherwise.
	runInsertProbes()
	// AND THE MIGRATION SUMMARY RUNS HERE, FOR THE SAME REASON THE REVERT DOES.
	// On Factorio 2.0 it WRITES a runtime-global setting, and that write raises
	// `on_runtime_mod_setting_changed` synchronously -- inside the assigning
	// statement, measured -- so it re-enters this guest exactly as `mine_entity`
	// does. After the drain there is no drain to re-enter and no transaction to
	// file a claim against. One length test on an empty slice in every save that
	// was built under the rule it is running under. See sedge.go.
	settleEdgeMode()
	// One tick's worth of "who built what" is spent. The notes were filled by
	// the events of the PREVIOUS tick and read by the drain above; anything
	// after this belongs to the next one. The tick's NEW PART TILES go with them
	// and for the same reason: the merge pre-pass that reads them has run. So do
	// the condemnations, whose teardowns the drain has either performed or made
	// moot -- a condemned cluster that dissolved before it was compiled took its
	// network down with it. And so does the conversion's own root list, whose
	// whole span is exactly this: the migration made those clusters and this drain
	// is what refused them, so past here they are ordinary balancers like any
	// other and an edit to one is an edit (legacy.go, legacyRoots).
	forgetBuildNotes()
	forgetAddedParts()
	forgetCondemned()
	forgetLegacyConverted()
}

// The log helpers -- logError, logAlert, and the builder every line is
// assembled in -- live in logline.go. `f2s` is gone with them: a coordinate is
// written straight into the line by logF1 rather than built out of three
// strings that are then thrown away and never reclaimed.
