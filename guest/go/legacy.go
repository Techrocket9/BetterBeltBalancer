package main

// ADOPTING A BELT BALANCER 2 / 3 SAVE.
//
// The rule, in one sentence: if an incumbent balancer mod is INSTALLED this
// guest never touches its entities, and once it is GONE this guest converts
// every `balancer-part` left standing into one of its own, once per save.
//
// THE TWO HALVES, AND NEITHER WORKS ALONE.
//
//   - The DATA half is `mod-data/prototypes/legacy.lua`, run from
//     `data-final-fixes.lua`. When a mod is removed, Factorio deletes every
//     entity whose prototype went with it, at load, before any script runs -- so
//     without a prototype of the same name and a compatible type there is
//     nothing left for this file to find. That file defines one only when
//     nobody else has, which is how "leave it alone while it is installed" is
//     enforced by the engine's own load order rather than by a list of names.
//   - The RUNTIME half is here. A prototype existing says nothing about WHOSE it
//     is, so the decision to convert is taken from `script.active_mods` and from
//     the `bbb-legacy-stub` marker prototype, which the data half defines if and
//     only if it defined the stub.
//
// WHY NOT A migrations/*.json RENAME, which is the obvious answer: a migration
// file is applied ONCE PER SAVE PER FILE and the engine remembers by file name,
// so a rename shipped here would be recorded as applied on the first load after
// this mod is installed -- which for the player this is for is a load where the
// incumbent is still present -- and could never fire again on the later load
// where it is gone. It also cannot express "do nothing while that mod is
// installed". The long form is in the data half's header.
//
// ---------------------------------------------------------------------------
// THE STATE MACHINE
// ---------------------------------------------------------------------------
//
// `legacyPhase` lives in the guest heap and therefore in the save. Three states,
// and the zero value is the one a fresh heap must start in:
//
//	legacyUnchecked  nothing has been decided on this heap. The gate every
//	                 event passes is one compare against this.
//	legacyDone       this game's `balancer-part` is ours (or there is none), and
//	                 the scan has run. Nothing more happens until the mod set
//	                 changes.
//	legacyBlocked    an incumbent is active, or a stranger owns `balancer-part`,
//	                 or the mod list could not be read. Nothing is converted and
//	                 nothing is touched -- including by the build path, which is
//	                 why a stranger lands here rather than in Done.
//
// The transitions, and which trigger drives each:
//
//	a new save (fk_on_init)                       -> Unchecked, then decide
//	a rebuilt guest (fk_migrate)                  -> Unchecked, then decide
//	the MOD SET moved (fk_on_configuration_changed)-> Unchecked, then decide
//	a fresh heap reaching an event first          -> Unchecked, then decide
//	an incumbent is active                        -> Blocked
//	a STRANGER owns `balancer-part`               -> Blocked
//	the other mod went away (the marker appeared)  Blocked -> Unchecked -> Done
//	nothing to convert, or converted               -> Done
//
// THE RE-ARM IS THE WHOLE REASON THIS IS A MACHINE AND NOT A BOOL. A player can
// install this mod first and the incumbent second, build fifty balancers with
// it, and remove it a month later; a save that had latched "scan done" would
// never look again. `fk_on_configuration_changed` is the trigger that case
// deserves and gets -- it is exactly the event that reports a neighbour being
// uninstalled, it is replicated, and it runs before the first tick, so the
// conversion is already inside the state a joining client downloads.
//
// The DEFERRED-FLUSH re-arm below is the belt and braces behind it, and it is
// cheap enough to keep: the test is the marker prototype rather than
// `script.active_mods`, because `prototypes.entity` is a point query returning
// an Object or nil -- two host calls and NO ALLOCATION, where reading the mod
// list allocates a Go string per mod every time.
//
// WHAT IT COSTS WHEN THERE IS NOTHING TO DO, which is every save this mod has
// ever been benchmarked on: one integer compare per event and one per deferred
// flush. The first check of a fresh heap pays one `script.active_mods`, two
// prototype lookups and one `find_entities_filtered` per surface, once, and then
// the phase is Done and no host call is ever made from this file again.
//
// ---------------------------------------------------------------------------
// WHAT SURVIVES AND WHAT DOES NOT
// ---------------------------------------------------------------------------
//
// Survives: every item ON THE BELTS around the balancer, because those belts are
// vanilla and nothing here touches them; the parts themselves, as this mod's
// parts, at the same tiles, on the same force, at the same quality and the same
// health; every input and output, because the compiler re-derives the edge list
// from the world exactly as it does after any other edit; a stack of the
// incumbent's item in a chest or a hand, which places this mod's parts; and a
// blueprint book full of the incumbent's balancers, whose ghosts revive into
// stubs the build path below swaps one at a time, a tick after they land.
//
// Does NOT survive: the items the incumbent was holding in its own Lua buffer.
// Belt Balancer 2 and 3 keep the items in flight through a balancer in a FIFO in
// their mod's `storage`, and Factorio deletes a removed mod's `storage` with the
// mod, before any script of ours could read it. There is no mechanism that
// could recover them and this file does not pretend otherwise; it is a fraction
// of a belt's worth per balancer, and the README says so.

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// LegacyPartName is the prototype every fork of the incumbent uses, and the
// name the data half's stub answers to.
//
// `belt-balancer` (the original) and `belt-balancer-performance` called it that
// too since Belt Balancer 2.1.0's own migration renamed `belt-balancer` ->
// `balancer-part`, and all four declare `!` conflicts against each other, so at
// most one of them is ever active.
const LegacyPartName = "balancer-part"

// LegacyMarkerName is the prototype that means "the `balancer-part` in this
// game is OUR stub". See the data half; nothing ever places one.
const LegacyMarkerName = "bbb-legacy-stub"

// TechName is this mod's own technology, granted to a force whose balancers
// were adopted -- the incumbent's three technologies die with it, and a player
// who had fifty balancers must not be left unable to craft a replacement part.
const TechName = "bbb-balancer"

// legacyIncumbents is walked in THIS order rather than in the order
// `script.active_mods` happens to arrive in. That return is a `pairs()` walk
// over a Lua hash table, so its order is not something a lockstep game may
// depend on; nothing below is host-visible, but a log line that named a
// different mod on two clients would be a bug that only ever showed up in a
// bug report.
var legacyIncumbents = [4]string{
	"belt-balancer-2",
	"belt-balancer-3",
	"belt-balancer",
	"belt-balancer-performance",
}

const (
	legacyUnchecked = iota
	legacyDone
	legacyBlocked
)

// legacyPhase is guest state and therefore save state. Its zero value is
// legacyUnchecked, which is what a fresh heap must mean.
var legacyPhase uint8

// Which entry point drove the decision. Reported on the summary line so an
// assertion can tell the load-time hook apart from the fallback behind it --
// the same thing `upg` asserts about which trigger drove a rebuild-from-world,
// and for the same reason: a feature whose fallback silently does the work of
// its primary trigger passes every test and ships broken.
const (
	legTrigInit = iota
	legTrigMigrate
	legTrigConfig
	legTrigDispatch
	legTrigDeferred
	legTrigAudit
)

var legacyTrigger uint8

func legacyTriggerName() string {
	switch legacyTrigger {
	case legTrigInit:
		return "init"
	case legTrigMigrate:
		return "migrate"
	case legTrigConfig:
		return "configuration_changed"
	case legTrigDispatch:
		return "first-dispatch"
	case legTrigDeferred:
		return "deferred"
	}
	return "audit"
}

// legacyGate is the HOT-PATH form: one integer compare, entered once per heap.
//
// It is called from onEventBody, immediately after ensureRegistry, and it is
// deliberately the cheap half -- it will not re-test a Blocked state, because
// doing so would put two host calls on the path every belt anyone lays anywhere
// on the map takes.
func legacyGate(trigger uint8) {
	if legacyPhase == legacyUnchecked {
		legacyTrigger = trigger
		legacyCheck()
	}
}

// legacyRearm re-tests a Blocked state, and is for places that run once per tick
// at most: the tail of the deferred flush, and the audit.
//
// It is the belt and braces behind `fk_on_configuration_changed`, not the
// primary trigger: the hook fires before the first tick, this fires whenever the
// player next does something. It stays because it costs one compare when the
// phase is not Blocked, and because it is what makes `/bbb-audit` a door onto
// the conversion for a save that somehow got past the hook.
func legacyRearm(trigger uint8) {
	if legacyPhase == legacyBlocked && legacyStubPresent() {
		legacyPhase = legacyUnchecked
	}
	legacyGate(trigger)
}

// legacyRecheck throws away whatever was decided and decides again.
//
// For the load-time hooks only: a new save, a rebuilt guest, and the mod set
// changing. Those are the only three moments at which the answer CAN change --
// prototypes and the mod list are both fixed for the life of a session -- so
// this is where a full re-decision belongs and nowhere else.
func legacyRecheck(trigger uint8) {
	legacyPhase = legacyUnchecked
	legacyGate(trigger)
}

// legacyCheck decides, and scans if the answer is yes. Only ever entered from
// legacyUnchecked, so each of its log lines is written once per decision rather
// than once per call.
func legacyCheck() {
	if name, version, active := legacyIncumbentActive(); active {
		legacyPhase = legacyBlocked
		logStart("legacy: ")
		logS(name)
		logS(" ")
		logS(version)
		logS(" is active; its balancers are left alone until it is removed")
		logEnd()
		return
	}
	if !legacyStubPresent() {
		// SOMETHING THAT IS NOT AN INCUMBENT THIS MOD KNOWS ABOUT OWNS THE NAME,
		// and that is the only way to get here: with no incumbent installed the
		// data half defines the stub AND the marker, so a missing marker means a
		// stranger got to `balancer-part` first. Converting their entities would
		// be the worst thing this file could do.
		//
		// BLOCKED AND NOT DONE, and the difference is the build path. `Done` is
		// this guest saying "the `balancer-part` in this game is MINE", which is
		// what `legacyBuilt` is gated on -- so a stranger's newly built part
		// would be swapped out from under them. Blocked also gets the marker
		// re-test, which is right: the stranger can be uninstalled too, and on
		// that load the stub appears and their balancers become ours, which is
		// the same promise the incumbents get.
		legacyPhase = legacyBlocked
		return
	}
	// SET BEFORE THE SCAN, for the same reason ensureRegistry sets its flag
	// before rebuilding: the scan makes host calls, a host call can raise an
	// event synchronously, and a re-entrant scan would convert every part twice.
	legacyPhase = legacyDone
	legacyScan()
}

// legacyIncumbentActive reads the mod list and looks for one of the four.
//
// It is the only thing here that allocates -- `ActiveMods` builds a Go string
// pair per mod -- which is why it is reached once per decision and never from a
// re-test. The marker prototype answers a narrower question for two host calls
// and no bytes; this one exists because it can NAME the mod in the log line, and
// because it is the honest test of the rule the feature states.
//
// A read that FAILS reports "active", not "clear". A guest that cannot see the
// mod list must not start converting entities on the strength of a prototype
// lookup alone.
func legacyIncumbentActive() (name, version string, active bool) {
	mods, err := fkapi.Script.ActiveMods()
	if err != nil {
		return "", "", true
	}
	for i := range legacyIncumbents {
		for j := range mods {
			if mods[j].Key == legacyIncumbents[i] {
				return mods[j].Key, mods[j].Val, true
			}
		}
	}
	return "", "", false
}

// legacyStubPresent reports whether `bbb-legacy-stub` exists, which is the data
// half saying "the `balancer-part` prototype in this game is mine".
//
// TWO HOST CALLS AND NO ALLOCATION. `prototypes.entity` is a LuaCustomTable, so
// the raw handle plus its index operator is a POINT query -- against the
// materialising `Entity()` attribute, which would build a Go slice of every
// entity prototype in the game (thousands, with a string each) to answer a
// yes/no question. A missing key is Lua's `nil`, which arrives as TagNil rather
// than as an error; measured against a real 2.0.77, `prototypes.entity[missing]`
// is nil and raises nothing.
func legacyStubPresent() bool {
	raw, err := fkapi.Prototypes.EntityRaw()
	if err != nil {
		return false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(LegacyMarkerName))
	if err != nil {
		return false
	}
	return v.Tag == fkapi.TagObject
}

// ---------------------------------------------------------------------------
// The scan
// ---------------------------------------------------------------------------

var (
	// The forces that owned at least one converted part, and a handle to each.
	// Both are appended in conversion order, which is surface order and then
	// engine order within a surface -- deterministic on every client.
	//
	// The handle is kept only for the length of this one dispatch, which is the
	// only condition under which this guest keeps an entity-derived handle at
	// all. It saves a `game.forces[...]` lookup per force and it is the same
	// shape onForcesMerged uses with the event's own `destination`.
	legacyForceIdx []uint32
	legacyForceObj []fkapi.Object

	// The tiles converted, so the summary line can say how many CLUSTERS they
	// made rather than only how many parts there were. Read after the flush,
	// when every root has settled.
	legacyKeys []key
)

// legacyScan converts every stub in the world and compiles what it made.
//
// SURFACES IN INDEX ORDER, through the same `collectSurfaces` rebuildFromWorld
// uses. Registration order decides node ids, node ids decide cluster roots, and
// roots decide hidden-surface slots -- so two clients that walked surfaces in
// two orders would place one network in two different slots, which is a desync
// and not a cosmetic difference.
//
// IT FLUSHES BEFORE IT RETURNS. The ordinary build path defers its compiles to
// the next tick, and the triggers that matter here cannot wait for one: a
// `--create` never reaches a tick at all, and `fk_on_init` on a save this mod
// was just added to is exactly that case. So this ends where the audit marker
// ends -- with the networks built inside the same dispatch.
func legacyScan() {
	collectSurfaces()
	legacyForceIdx = legacyForceIdx[:0]
	legacyForceObj = legacyForceObj[:0]
	legacyKeys = legacyKeys[:0]

	surfaces, converted := uint32(0), uint32(0)
	for i := range rfwSurfIdx {
		si := rfwSurfIdx[i]
		if si == hiddenIdx {
			continue
		}
		surfaces++
		converted += legacyConvertOn(fkapi.LuaSurface{Object: rfwSurfObj[i]}, si)
	}

	// Nothing found is the overwhelmingly common answer and it is not worth a
	// line: a save that never had an incumbent would otherwise carry one of
	// these for every new map anybody ever makes.
	if converted == 0 {
		return
	}

	// The technology, per force, before the flush -- a force that owned
	// balancers must be able to craft a replacement part, and the incumbent's
	// own three technologies went with it.
	researched := uint32(0)
	for i := range legacyForceObj {
		if legacyGrantTech(legacyForceObj[i]) {
			researched++
		}
	}
	legacyForceObj = legacyForceObj[:0]

	flush()

	// How many balancers those parts turned out to be, counted after the flush
	// so every merge and every re-root has settled. Node ids, not a map walk.
	clusters := uint32(0)
	gen++
	for i := range legacyKeys {
		id, ok := index[legacyKeys[i]]
		if !ok {
			continue
		}
		r := find(id)
		if mark[r] == gen {
			continue
		}
		mark[r] = gen
		clusters++
	}
	legacyKeys = legacyKeys[:0]

	logStart("legacy: adopted ")
	logPlural(converted)
	logS(" from ")
	logU(surfaces)
	logS(" surfaces into ")
	logU(clusters)
	logS(" clusters, ")
	logU(researched)
	logS(" forces given the balancer technology, trigger=")
	logS(legacyTriggerName())
	logEnd()
	logState()
}

// legacyConvertOn converts every stub on one surface.
//
// `findByNameAll` is the whole-surface name filter registerPartsOn uses; the two
// never interleave.
func legacyConvertOn(s fkapi.LuaSurface, si uint32) uint32 {
	allNameFilter = fkapi.OfString(LegacyPartName)
	ents, err := s.FindEntitiesFiltered(findByNameAll)
	if err != nil {
		return 0
	}
	n := uint32(0)
	for i := range ents {
		if legacyConvertOne(s, si, ents[i]) {
			n++
		}
	}
	return n
}

// legacyConvertOne replaces one stub with one of this mod's parts.
//
// EVERYTHING IS READ BEFORE ANYTHING IS DESTROYED, because after the destroy
// there is nothing left to ask. The order is forced the other way round by the
// collision box: both prototypes collide on `object` and `transport_belt`, so
// the new part cannot be created until the old one is out of the tile.
//
// A create that fails here is a REAL failure -- the tile was occupied by exactly
// one thing and that thing has just gone -- so it is an `error:` line, which
// fails a test run, rather than a silent skip.
func legacyConvertOne(s fkapi.LuaSurface, si uint32, o fkapi.Object) bool {
	if hiddenIdx != 0 && si == hiddenIdx {
		return false
	}
	ent := fkapi.LuaEntity{Object: o}
	pos, err := ent.Position()
	if err != nil {
		return false
	}
	fo, err := ent.Force()
	if err != nil {
		return false
	}
	fi, err := (fkapi.LuaForce{Object: fo}).Index()
	if err != nil {
		return false
	}
	// The quality is passed straight through as the prototype HANDLE rather than
	// as its name: `QualityID` is `string or LuaQualityPrototype`, and a handle
	// costs one host call and copies no string into the guest heap. A quality
	// prototype is a global object and is not invalidated by the destroy below.
	q, qErr := ent.Quality()
	// Health, so a damaged balancer stays damaged. Both prototypes carry the
	// same max_health of 170, so this is a straight copy; the alternative is
	// silently repairing a building the player did not ask us to touch.
	h, hErr := ent.Health()

	// NO raise_destroy. A conversion is not a removal and nothing outside this
	// mod should be told one happened -- least of all this guest's own event
	// handler, which is subscribed to `balancer-part` for the build path below.
	if _, err := ent.Destroy(fkapi.LuaEntityDestroyArgs{}); err != nil {
		return false
	}

	np, err := s.CreateEntity(legacyCreateArgs(pos, fi, q, qErr == nil))
	if err != nil || np == nil {
		logErrStart("legacy: create_entity returned nil replacing a ")
		logS(LegacyPartName)
		logS(" at ")
		logF1(pos.X)
		logS(",")
		logF1(pos.Y)
		logEnd()
		return false
	}
	if hErr == nil && h != nil {
		_ = (fkapi.LuaEntity{Object: *np}).SetHealth(*h)
	}

	// Straight into the registry, exactly as a build event does -- so the
	// clusters form, merge and queue themselves through the one code path, and
	// the flush at the end of the scan restyles and compiles them the way it
	// would after a blueprint paste.
	k := key{s: si, x: floorTile(pos.X), y: floorTile(pos.Y)}
	AddPart(k, fi)
	legacyKeys = append(legacyKeys, k)

	for i := range legacyForceIdx {
		if legacyForceIdx[i] == fi {
			return true
		}
	}
	legacyForceIdx = append(legacyForceIdx, fi)
	legacyForceObj = append(legacyForceObj, fo)
	return true
}

// legacyCreateArgs is this file's own `create_entity` table.
//
// Separate from compile.go's `createArgs` and deliberately so: that one is on
// the hot path, is five fixed keys written once by initBuffers, and has no
// `quality`. This one runs at most once per part on a once-per-save path, so it
// is written the obvious way.
var (
	legacyKV  [4]fkapi.KeyValue
	legacyPos [2]fkapi.Value
)

func legacyCreateArgs(pos fkapi.MapPosition, force uint32, quality fkapi.Object, haveQuality bool) fkapi.Value {
	legacyPos[0] = fkapi.OfNumber(pos.X)
	legacyPos[1] = fkapi.OfNumber(pos.Y)
	legacyKV[0] = fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(PartName)}
	legacyKV[1] = fkapi.KeyValue{Key: fkapi.OfString("position"), Val: fkapi.OfArray(legacyPos[0], legacyPos[1])}
	legacyKV[2] = fkapi.KeyValue{Key: fkapi.OfString("force"), Val: fkapi.OfNumber(float64(force))}
	n := 3
	if haveQuality {
		legacyKV[3] = fkapi.KeyValue{Key: fkapi.OfString("quality"), Val: fkapi.OfObject(quality)}
		n++
	}
	return fkapi.Value{Tag: fkapi.TagMap, Map: legacyKV[:n]}
}

// legacyGrantTech marks this mod's technology researched on one force, and
// reports whether it had to.
//
// A point query again, for the reason legacyStubPresent gives: `technologies` is
// a LuaCustomTable, and the whole-table attribute would build a Go slice of
// every technology in the game to reach one of them.
//
// It is deliberately silent about a force that already had it, and about a game
// where the technology does not exist at all -- an overhaul mod may have removed
// it, and a migration is not the place to argue.
func legacyGrantTech(f fkapi.Object) bool {
	raw, err := (fkapi.LuaForce{Object: f}).TechnologiesRaw()
	if err != nil {
		return false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(TechName))
	if err != nil || v.Tag != fkapi.TagObject {
		return false
	}
	t := fkapi.LuaTechnology{Object: v.Object}
	if done, err := t.Researched(); err != nil || done {
		return false
	}
	if err := t.SetResearched(true); err != nil {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// A stub that appears AFTER the scan
// ---------------------------------------------------------------------------

// legacyBuilt notes one stub that arrived through a build event.
//
// WHERE THESE COME FROM. The scan handles a world that already has them; this
// handles the ones that appear afterwards, and the case that makes it worth
// having is a migrating player's BLUEPRINT BOOK. Every blueprint they took while
// the incumbent was installed names `balancer-part`, and the data half's
// `placeable_by` makes a ghost of one ask a construction robot for a
// `bbb-balancer-part` item -- so the book keeps working, and every balancer it
// places becomes a real part. A script from another mod building one lands here
// too.
//
// It is reached only when the phase is Done, which is the guest saying the
// `balancer-part` prototype in this game is the data half's stub. While an
// incumbent is installed this returns without a host call, which is what makes
// having `balancer-part` in the subscription filter affordable at all.
//
// THAT GATE IS WHY A STRANGER OWNING THE NAME LANDS IN BLOCKED AND NOT IN DONE
// (legacyCheck). The scan is gated on the marker and would leave a stranger
// alone either way; this is gated on the PHASE, so `Done` there would have swapped
// somebody else's newly built entity out from under them. Red-proven: the mig
// suite's late-build probe comes out `legacy=0 ours=1` beside the stranger with
// the phase set wrong, and `legacy=1 ours=0` with it set right.
//
// IT DEFERS, AND THAT IS A CORRECTION RATHER THAN A PREFERENCE. The obvious
// version converts in place, inside the event -- and `create_entity` RETURNS NIL
// to whoever placed the entity when that entity is destroyed during the build
// event it raised. Measured immediately: the harness's own
// `create_entity{... raise_built = true}` came back nil and the test mod raised
// on it. A mod scripting a `balancer-part` into the world is entitled to a
// handle back, so the stub is left standing for one tick and swapped from the
// flush -- which is where this guest does everything else that reads the world.
func legacyBuilt(si uint32, pos fkapi.MapPosition) {
	if legacyPhase != legacyDone {
		return
	}
	legacyBuildQueue = append(legacyBuildQueue, key{
		s: si, x: floorTile(pos.X), y: floorTile(pos.Y),
	})
	requestFlush()
}

// legacyBuildQueue is one tile per stub waiting to be swapped. Scalars only, the
// same shape probe.go's queue has: nothing in it is an entity reference, so
// nothing in it can be stale a tick later.
var legacyBuildQueue []key

// legacyRunBuilds is the flush half, called from the top of flush() so that the
// parts it registers are compiled by the same drain -- including the synchronous
// drain a `bbb-audit` marker forces, which is the only one a `--create` ever
// reaches.
//
// One length test on an empty slice on every flush this mod will ever do in a
// game that never had an incumbent.
func legacyRunBuilds() {
	if len(legacyBuildQueue) == 0 {
		return
	}
	legacyForceIdx = legacyForceIdx[:0]
	legacyForceObj = legacyForceObj[:0]
	legacyKeys = legacyKeys[:0]
	for i := range legacyBuildQueue {
		k := legacyBuildQueue[i]
		s, ok := surfaceByIndex(k.s)
		if !ok {
			continue
		}
		// Re-found from the world, because a tick has passed and whatever placed
		// it may have taken it away again.
		//
		// AN AREA QUERY RATHER THAN `find_entity`, AND THAT IS A DEFECT THE
		// FIDELITY RIG FOUND rather than a matter of taste. `find_entity` takes
		// an `EntityWithQualityID`, and the pinned runtime API says of a bare
		// name that "Normal quality will be used" -- so a `balancer-part`
		// standing at any other quality is INVISIBLE to it. Measured on 2.0.77
		// against a real uncommon entity: `find_entity(name, p)` is nil where
		// `find_entities_filtered{name = ..., position = p}` returns it, and
		// `find_entity({name = ..., quality = ...}, p)` returns it too.
		//
		// This is the BLUEPRINT BOOK'S path -- a robot reviving a ghost of an
		// uncommon balancer-part -- so the stub stood there for the rest of the
		// save, unconverted, unregistered and unlogged. The whole-world scan
		// above never had the bug: `findByNameAll` filters on the name alone and
		// the engine returns every quality.
		//
		// The slice this allocates is once per stub built on a once-per-save
		// path, which is where the obvious thing is affordable and the wrong
		// answer was not. `findOnTile` (findpart.go) is this fix stated once --
		// the same trap stood at four more call sites when it was found here,
		// and a sixth question about the same identity should ask the same
		// code or not ask at all.
		o, found, ferr := findOnTile(s, LegacyPartName, k.x, k.y)
		if ferr != nil || !found {
			continue
		}
		if !legacyConvertOne(s, k.s, o) {
			continue
		}
		if verboseLog {
			logStart("legacy: adopted a ")
			logS(LegacyPartName)
			logS(" built at ")
			logI(k.x)
			logS(",")
			logI(k.y)
			logEnd()
		}
	}
	legacyBuildQueue = legacyBuildQueue[:0]
	for i := range legacyForceObj {
		legacyGrantTech(legacyForceObj[i])
	}
	legacyForceObj = legacyForceObj[:0]
	legacyKeys = legacyKeys[:0]
	logState()
}
