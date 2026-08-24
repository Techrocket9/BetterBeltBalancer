// Command better-belt-balancer is the Go brain of the mod: TinyGo compiles it
// to wasm, FkLua compiles that to Lua, and `fklua mod` packages the result as
// the mod's control.lua side.
//
// This file is the event surface. It knows three things:
//
//   - a `bbb-balancer-part` appeared or disappeared, so the registry in
//     cluster.go changes and the clusters it touched must be recompiled;
//   - a BELT-CONNECTABLE appeared, disappeared, rotated or flipped near a
//     cluster, so that cluster's edge list may have changed;
//   - nothing relevant happened, which is almost every event, and the job is to
//     work that out for as few host calls as possible.
//
// Everything else -- the network itself -- is compile.go, and it runs only from
// here. There is no on_tick handler and there must never be one: the whole
// architecture is that a running balancer costs zero script.
//
//	tinygo build -target=wasm-unknown -scheduler=none -gc=leaking -opt=2 \
//	  -o dist/bbb.wasm ./guest/go
//	fklua mod dist/bbb.wasm --name better-belt-balancer --version 0.1.0 \
//	  --persist=packed
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

// PartName is the visible 1x1 entity, defined in mod-data/prototypes/entity.lua.
//
// The eleven per-entity events are SUBSCRIBED WITH FILTERS now (FKLUA-GAPS.md
// item 3), so the engine decides in C++ whether this guest is entered at all.
// The name is still read once the guest IS entered, because the filter says
// "one of the things this mod cares about" and the handler still has to know
// WHICH -- a part, the audit marker, a piece of our own network, or a belt.
const PartName = "bbb-balancer-part"

// AuditName is a marker entity, defined in mod-data/prototypes/hidden.lua,
// that nothing but a script can place. Building one asks the guest to check
// every cluster's stored fingerprint against a from-scratch classification of
// the world, report the result, and repair anything that drifted; the marker is
// then destroyed. It is how the M3 stress test asks "is the state you are
// holding still the state the world is in?" without a second implementation of
// the compiler in Lua, and it is a usable field diagnostic for the same reason.
const AuditName = "bbb-audit"

// isOurs reports whether a prototype name is one the compiler places.
//
// A build or death event for one of these did NOT come from us -- the compiler
// never raises -- so it is something else copying or destroying a piece of a
// network, and the cluster that owns it has to be rebuilt rather than merely
// re-classified. The edge fingerprint would not notice: it is a statement about
// the belts AROUND a cluster, and nothing about the interfaces on it.
func isOurs(name string) bool {
	return name == nameLinkedBelt || name == nameBelt ||
		name == nameSplitter || name == nameLaneSplit
}

// NOTHING IN THIS GUEST IS READ AT A HAND-DERIVED OFFSET ANY MORE, and
// `test/check-layout.py` -- which existed to keep the last two honest -- is
// deleted along with its `make mod` hook.
//
// `fklua gen-bindings` emits a struct and a `Read<Event>(ptr)` per event for all
// 218 events (FKLUA-GAPS.md item 4). The two hold-outs were
// `on_undo_applied` and `on_redo_applied`, where the generated reader was
// CORRECT AND EXPENSIVE: it decodes `actions` -- an array of tier-2 dynamic
// Values, BlueprintEntities and all -- into fresh Go memory that `-gc=leaking`
// never gives back, and the host had already deep-copied the same thing into
// the scratch buffer before the guest was entered at all (item 14).
//
// `fk.subscribe` carries a FIELD MASK now, so the field is simply never
// marshalled: see the SubscribeMasked calls in init below. Upstream measured the
// host half at 7.49 ms -> 2.7 us for a 200-action undo, and what it costs here
// is a decode of an array the host wrote as empty.

// What an event does to the world, as far as this mod cares.
const (
	evAppear = iota
	evVanish
	evModify
)

// entityFilter is the filter list every per-entity subscription carries.
//
// THREE TERMS, OR-ed by the engine, and between them they are the whole of what
// this mod can care about:
//
//   - `transport-belt-connectable`, a category the filter grammar has built in.
//     It is exactly `classifySide`'s five types plus linked belts and lane
//     splitters -- which is to say every belt that can be an edge, AND all four
//     of our own hidden prototypes, since each is a clone of a base
//     belt-connectable. One term instead of nine names and five types.
//   - the part, which is a `simple-entity-with-force` and therefore not
//     belt-connectable.
//   - the audit marker and the insert probe, both `simple-entity`, for the same
//     reason.
//   - `balancer-part`, the INCUMBENT'S part and the name this mod's data stage
//     keeps alive after the incumbent is uninstalled. A ghost of one revived out
//     of a migrating player's old blueprint book arrives through the build
//     events and is swapped for a real part; see legacy.go. A name filter for a
//     prototype that does not exist is ACCEPTED by `script.on_event` and matches
//     nothing -- measured against 2.0.77 rather than assumed, because the
//     alternative was a subscription list that had to branch on the mod set.
//
// What this removes is everything else on the map: an assembler placed, a tree
// mined, a biter killed, a wall built. Each of those used to enter the guest and
// pay a dispatch plus three host calls (`name`, `position`, `surface_index`)
// to be rejected. `on_entity_died` is the one that mattered most -- on a map
// under attack it is the highest-frequency event there is, and none of it was
// ever ours.
//
// Built once, at _initialize. `fkapi.NameFilter` allocates a small slice per
// call and this is the only allocation of the kind the guest makes; the filter
// list is decoded host-side once, at subscribe time, and never again.
var entityFilter = append(
	[]fkapi.Value{fkapi.OfMap(fkapi.KeyValue{
		Key: fkapi.OfString("filter"),
		Val: fkapi.OfString("transport-belt-connectable"),
	})},
	fkapi.NameFilter(PartName, AuditName, ProbeName, LegacyPartName)...,
)

func init() {
	// Subscribing from an initialiser is what a guest does: this runs during
	// _initialize, before any event can fire. `fk.subscribe` has a generated Go
	// wrapper now, so the hand-declared `//go:wasmimport fk subscribe` this file
	// used to carry is gone (FKLUA-GAPS.md item 3).
	//
	// EVERY EVENT ID BELOW IS A LITERAL CONSTANT AT ITS CALL SITE, and it has to
	// stay that way: FkLua's constant scan prunes 218 event descriptors down to
	// the ones a guest names, and an id it cannot prove ships all of them.
	// Looping over a slice of ids would compile and would silently add several
	// hundred KB to the mod.
	//
	// Appearance. The engine raises a different one of these for each way a
	// part can come into being, and a mod that watches only on_built_entity is
	// the classic source of "my blueprint/robot/clone did not register". All six
	// take filters.
	fkapi.SubscribeFiltered(fkapi.EventOnBuiltEntity, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventOnRobotBuiltEntity, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventScriptRaisedBuilt, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventScriptRaisedRevive, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventOnEntityCloned, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventOnSpacePlatformBuiltEntity, entityFilter...)

	// Disappearance. All five take filters too.
	fkapi.SubscribeFiltered(fkapi.EventOnPlayerMinedEntity, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventOnRobotMinedEntity, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventOnEntityDied, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventScriptRaisedDestroy, entityFilter...)
	fkapi.SubscribeFiltered(fkapi.EventOnSpacePlatformMinedEntity, entityFilter...)

	// Change in place. A belt that turns around at a cluster's edge swaps an
	// input for an output without any entity being created or destroyed, and a
	// balancer that did not notice would keep pushing items into a belt that
	// now points back at it.
	//
	// THESE TWO TAKE NO FILTERS -- `runtime-api.json` gives a `filter` concept
	// to 30 events and neither of these is one of them -- so they arrive for
	// every rotation anywhere on the map and are rejected in-guest, by POSITION,
	// which is a registry lookup with no host call behind it. They are a
	// keypress, not a tick.
	fkapi.Subscribe(fkapi.EventOnPlayerRotatedEntity)
	fkapi.Subscribe(fkapi.EventOnPlayerFlippedEntity)

	// M3. Everything above reports ONE entity. These report a region, a
	// surface, or nothing but the player who pressed a key, and each of them is
	// a way for the world to change under a compiled network without any of the
	// per-entity events firing. None of them takes a filter either, and none of
	// them needs one: they are all rare.
	//
	// Undo and redo: the actions they replay mostly arrive as ordinary build
	// and mine events, but not all of them do -- an undone rotation is the one
	// that gets away -- so the handler re-validates instead of trusting them.
	//
	// MASKED, because `actions` is the one unbounded field this mod subscribes
	// to and it reads none of it. The mask is resolved once, at subscribe time,
	// like a filter; a masked container arrives as (ptr, count) = (0, 0), which
	// is a reading every generated decoder already produces, so the layout does
	// not move and `player_index` keeps the offset the guest was compiled
	// against. Upstream's own measurement of the same subscription, 200 actions
	// of 2 entities each: 7.49 ms per dispatch unmasked, 2.7 us masked.
	//
	// `Skip…` constants exist only for maskable fields -- optionals and
	// containers -- so being wrong about one costs a value that reads absent,
	// never a zero indistinguishable from a real one.
	fkapi.SubscribeMasked(fkapi.EventOnUndoApplied, fkapi.SkipOnUndoAppliedActions)
	fkapi.SubscribeMasked(fkapi.EventOnRedoApplied, fkapi.SkipOnRedoAppliedActions)

	// Cloning. `on_entity_cloned` (above) is per entity and does not fire for
	// every clone path; these two report the region, which is what a
	// reconcile needs -- including for parts that `clear_destination_entities`
	// destroyed without an event of any kind.
	fkapi.Subscribe(fkapi.EventOnAreaCloned)
	fkapi.Subscribe(fkapi.EventOnBrushCloned)

	// Surfaces. The PRE events are where the work happens: the parts and the
	// hidden entities are still valid there, so a slot can be freed and the
	// items in a network can be handed back. The POST events exist for the one
	// case the pre event cannot finish -- the surface that just went is the
	// HIDDEN one, and every network has to be rebuilt on its replacement.
	fkapi.Subscribe(fkapi.EventOnPreSurfaceDeleted)
	fkapi.Subscribe(fkapi.EventOnSurfaceDeleted)
	fkapi.Subscribe(fkapi.EventOnPreSurfaceCleared)
	fkapi.Subscribe(fkapi.EventOnSurfaceCleared)

	// Forces. `game.merge_forces` moves every entity of one force onto another
	// and raises NO per-entity event for any of them, so without this the
	// registry keeps a force index that no longer exists -- and clusters are per
	// force, so two balancers that are now one force's would stay two clusters
	// and the next compile would ask the engine to filter by a dead index.
	//
	// The MERGED half, not the merging one: `on_forces_merged` carries
	// `source_index` as a number, which is the only thing the registry needs,
	// and it fires after the transfer, when the world already agrees with what
	// the handler is about to write down. See lifecycle.go.
	fkapi.Subscribe(fkapi.EventOnForcesMerged)

	// THE ONE SETTING THIS MOD HAS, and the only subscription here that cannot
	// fire on the engine trunk targets: `bbb-multi-edge-parts` is defined by the
	// settings stage on Factorio 2.0.x and never on 2.1.x, so on 2.1 this is a
	// subscription to a change that cannot happen. It is subscribed
	// unconditionally all the same, because the alternative is a subscription list
	// that branches on the engine -- and the mod-data tree is deliberately
	// identical on both release branches (fklua.toml).
	//
	// NO FILTER EXISTS FOR IT. `runtime-api.json` gives a `filter` concept to 30
	// events and this is not one of them, so it arrives whenever ANY mod's runtime
	// setting changes and the handler's first act is to compare the name against
	// its own. That is a keypress, not a tick.
	//
	// MASKED over `player_index`, which is its only maskable field and one this
	// guest has no use for: who flipped a MAP setting cannot change what the flip
	// obliges, because every client is obliged identically or the game desyncs.
	// See sedge.go.
	fkapi.SubscribeMasked(fkapi.EventOnRuntimeModSettingChanged,
		fkapi.SkipOnRuntimeModSettingChangedPlayerIndex)
}

//go:wasmexport fk_on_init
func onInit() {
	fk.Log("[BBB] guest init: cluster tracking armed for " + PartName)
	// A NEW MAP, OR THIS MOD BEING ADDED TO A SAVE THAT ALREADY EXISTS -- and
	// the second is the common half of the migration this feature is for: a
	// player swaps Belt Balancer 2 out and this mod in, in one edit of the mod
	// list, and `on_init` is the first moment a guest exists at all. See
	// legacy.go. It does NOT call ensureRegistry: the scan registers what it
	// converts through AddPart directly, and the world scan the first event
	// pays for adopts the networks this one just built rather than rebuilding
	// them.
	edgeModeRecheck()
	// A NEW SAVE'S REGISTRY IS EMPTY AND THEREFORE IN AGREEMENT WITH ANY RULE, so
	// this is where the anchor starts. Without it a first flip of the setting
	// would arrive on ModeUnknown and act -- which on an empty registry is a
	// no-op, so this is tidiness rather than a fix, but an anchor that says what
	// it means from the first tick is worth one assignment. See sedge.go.
	edgeAnchorSettle()
	legacyRecheck(legTrigInit)
}

// fk_on_configuration_changed is the MOD SET moving: a neighbour added, removed,
// or moved to another version.
//
// It is what makes "this mod installed first, the incumbent removed a month
// later" convert AT LOAD rather than at whatever event happens to arrive next.
// Factorio raises `on_configuration_changed` for exactly that edit and for
// nothing else a guest can observe; until upstream wired this export the event
// was consumed by the glue's own rebuild check and told a guest nothing
// (FKLUA-GAPS.md item 22, fixed upstream).
//
// REPLICATED, and therefore allowed to write guest state: it runs on the peer
// that LOADED the save, before the first tick, so its effects are already inside
// the state a joining client downloads. That is the opposite side of the rule
// from `fk_after_load`, which is armed from `script.on_load` -- the thing a
// joiner runs and nobody else -- and which this mod must never export.
//
// It fires AFTER `fk_migrate` on a load that is both, so the heap is settled
// before the guest is told the world around it moved. And it fires on the load
// that ADDS this mod, right after fk_on_init -- a newly added mod is itself a
// mod-set change -- so that load reaches legacyRecheck twice; the second pass
// finds nothing left to convert and says nothing, at the price of one by-name
// scan per surface, once.
//
//go:wasmexport fk_on_configuration_changed
func onConfigurationChanged() {
	// AND IT IS WHERE THE ENGINE CAN CHANGE UNDER A SAVE. Factorio raises this
	// for a game-version change as well as for a mod-set one, so it is the hook
	// that catches a 2.0 heap waking up on 2.1 -- where `bbb-can-stack` is gone
	// and every multi-edge cluster in the save has to be refused rather than
	// compiled. Free on every other load: sedge.go's answer is one point query.
	edgeModeRecheck()
	ensureRegistry()
	legacyRecheck(legTrigConfig)
}

// fk_migrate is the REBUILT-GUEST SIGNAL, and exporting it is now safe.
//
// It used to mean "adopt the previous build's entire linear memory", rodata
// included, which no Go guest could survive (FKLUA-GAPS.md item 13) -- so this
// mod exported nothing and inferred the same fact from `registryReady` being
// false in a freshly initialised heap. Upstream split the two acts: `fk_migrate`
// is a NOTIFICATION on a fresh heap and `fk_migrate_adopt` is the opt-in that
// really hands the bytes over. THIS MOD MUST NEVER EXPORT `fk_migrate_adopt`;
// its state is Go maps and slices reachable only through package-level roots,
// which is precisely the shape adoption cannot carry across a rebuild.
//
// What the export buys is that the rebuild happens at a NAMED POINT --
// on_configuration_changed, after on_load and before the first tick, where
// `game` is fully available -- rather than inside whichever event happened to
// arrive first. The inference stays as well, because it covers what this hook
// does not: a mod added to a save that already contains parts, and any other
// way a heap can come back empty.
//
//go:wasmexport fk_migrate
func onMigrate(oldVersion uint32) {
	logStart("the mod was rebuilt (guest state version ")
	logU(oldVersion)
	logS("); the heap is fresh and the registry comes from the world")
	logEnd()
	edgeModeRecheck()
	ensureRegistry()
	// A fresh heap knows nothing about a migration it may already have done, so
	// the decision is taken again from the world. It is cheap and it is correct:
	// on a save that has been through it, the scan finds no `balancer-part` and
	// says nothing.
	legacyRecheck(legTrigMigrate)
}

// fk_on_deferred drains one tick's worth of accumulated recompiles.
//
// Every per-entity event does two things now: it updates the registry, which
// must happen inside the event because the entity is only valid there, and it
// puts the clusters it touched on the queue and asks for a flush. `fk.Defer()`
// registers a ONE-SHOT on_tick and tears it down again from inside this
// function, so an idle guest still pays zero registrations and zero per-tick
// calls -- the invariant M4 measured, and the reason this could be adopted at
// all.
//
// The flush lands on the FOLLOWING tick. That is not a compromise here, it is
// the thing that made the removal window unnecessary (compile.go): by the time
// this runs, an entity that was mined during the previous tick is gone from the
// world, so a classification pass simply does not find it.
//
//go:wasmexport fk_on_deferred
func onDeferred() {
	// A save taken between the defer and the flush comes back with the flag
	// re-armed, and if the heap did NOT come back with it -- a rebuilt guest --
	// the queues are empty and this is where the world scan belongs anyway.
	ensureRegistry()
	flush()
	// The belt and braces behind fk_on_configuration_changed: a Blocked legacy
	// phase is re-tested against the marker prototype, which is two host calls
	// and no allocation, and only while it IS Blocked. One integer compare on
	// every other tick this guest ever runs. See legacy.go.
	legacyRearm(legTrigDeferred)
	// The one place a collection may START, and only under `--gc=collected` --
	// this is a no-op call that inlines away entirely under `-gc=leaking`. It is
	// here rather than in an `fk_on_tick` this mod does not have, and rather
	// than in the audit path, for the reasons in gc.go.
	gcCollectIfNeeded()
}

// fk_on_event is a wrapper around the handler, and the wrapper exists for one
// line.
//
// `onEvent` returns from a dozen places -- almost every event this guest is
// entered for is rejected early and cheaply, which is the design. The collector's
// pacing has to be told about the allocation those rejections made anyway (a name
// bought, a return block decoded), and a `defer` would cost a closure on every
// event under a guest that has no scheduler. So the handler keeps its early
// returns and the export owns the one call that must happen however it left.
//
// Under `-gc=leaking` `gcArmIfNeeded` has no body and this is the same function
// it always was.
//
//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	onEventBody(id, ptr)
	gcArmIfNeeded()
}

func onEventBody(id, ptr uint32) {
	// The registry may be empty because this is the first event after a mod
	// upgrade discarded the guest heap. Rebuilding from the world has to happen
	// before anything is decided, or the first thing decided is that a
	// perfectly good balancer has no network and needs a second one. One bool
	// test; see lifecycle.go.
	ensureRegistry()
	// The same shape, one state further on: a fresh heap has not yet decided
	// whether this game's `balancer-part` is an incumbent's or the stub this mod
	// ships, and it must decide before it reacts to one. One integer compare;
	// see legacy.go, which is emphatic about why this is the CHEAP gate and the
	// re-arm lives in fk_on_deferred.
	legacyGate(legTrigDispatch)

	var obj fkapi.Object
	what := evAppear
	// Who mined it, for the one event that says. Zero everywhere else, and it is
	// read out of the event's own decoded block, so it costs nothing: a scalar
	// that was already marshalled whether or not this guest looked at it. What it
	// is FOR is carry.go's beneficiary -- a player who mines the last part of a
	// balancer gets what the machine was holding, exactly as they would from a
	// splitter.
	minedBy := uint32(0)
	// ... and who BUILT it, for the one build event that says. Same shape and
	// the same cost -- a scalar already sitting in the decoded block whether or
	// not this guest reads it. What it is FOR is the port limit: a build that
	// would push a balancer past plan.MaxPorts is refused a tick later, and the
	// player who made it is the one to tell and to hand the piece back to. Zero
	// for a robot, a script build, a revive and a clone -- and zero in every
	// headless suite, which is why nothing else in them moved. See limit.go.
	builtBy := uint32(0)
	switch id {
	// --- events that carry no entity ---------------------------------------
	case fkapi.EventOnUndoApplied:
		onUndo(fkapi.ReadOnUndoApplied(ptr).PlayerIndex, "undo")
		return
	case fkapi.EventOnRedoApplied:
		onUndo(fkapi.ReadOnRedoApplied(ptr).PlayerIndex, "redo")
		return
	case fkapi.EventOnAreaCloned:
		onAreaCloned(fkapi.ReadOnAreaCloned(ptr))
		return
	case fkapi.EventOnBrushCloned:
		onBrushCloned(fkapi.ReadOnBrushCloned(ptr))
		return
	case fkapi.EventOnPreSurfaceDeleted:
		onSurfaceGoing(fkapi.ReadOnPreSurfaceDeleted(ptr).SurfaceIndex, "deleted")
		return
	case fkapi.EventOnPreSurfaceCleared:
		onSurfaceGoing(fkapi.ReadOnPreSurfaceCleared(ptr).SurfaceIndex, "cleared")
		return
	case fkapi.EventOnSurfaceDeleted:
		onSurfaceGone(fkapi.ReadOnSurfaceDeleted(ptr).SurfaceIndex)
		return
	case fkapi.EventOnSurfaceCleared:
		onSurfaceGone(fkapi.ReadOnSurfaceCleared(ptr).SurfaceIndex)
		return
	case fkapi.EventOnForcesMerged:
		ev := fkapi.ReadOnForcesMerged(ptr)
		if di, err := (fkapi.LuaForce{Object: ev.Destination}).Index(); err == nil {
			onForcesMerged(ev.SourceIndex, di)
		}
		return
	case fkapi.EventOnRuntimeModSettingChanged:
		// EVERY MOD'S SETTINGS ARRIVE HERE -- the event takes no filter -- so the
		// handler's first act is to compare the name, and `Setting` is a Go string
		// the decoder copied out of the host before this guest was entered. That
		// copy is the one cost, it is a keypress rather than a tick, and there is
		// no cheaper form: a name is the only thing that identifies which setting
		// moved. See sedge.go for what happens when it is ours.
		onEdgeModeSettingChanged(fkapi.ReadOnRuntimeModSettingChanged(ptr).Setting)
		return
	}

	switch id {
	// The four that build a balancer part. Hand-derived offsets until the
	// dictionary `tags` field started generating; the readers below cost one
	// extra branch each on the presence byte for `tags` and allocate only when
	// an entity actually carries some.
	case fkapi.EventOnBuiltEntity:
		ev := fkapi.ReadOnBuiltEntity(ptr)
		obj, builtBy = ev.Entity, ev.PlayerIndex
	case fkapi.EventOnRobotBuiltEntity:
		obj = fkapi.ReadOnRobotBuiltEntity(ptr).Entity
	case fkapi.EventOnSpacePlatformBuiltEntity:
		obj = fkapi.ReadOnSpacePlatformBuiltEntity(ptr).Entity
	case fkapi.EventScriptRaisedRevive:
		obj = fkapi.ReadScriptRaisedRevive(ptr).Entity

	case fkapi.EventScriptRaisedBuilt:
		obj = fkapi.ReadScriptRaisedBuilt(ptr).Entity
	case fkapi.EventOnEntityCloned:
		obj = fkapi.ReadOnEntityCloned(ptr).Destination

	case fkapi.EventOnPlayerMinedEntity:
		ev := fkapi.ReadOnPlayerMinedEntity(ptr)
		obj, what, minedBy = ev.Entity, evVanish, ev.PlayerIndex
	case fkapi.EventOnRobotMinedEntity:
		obj, what = fkapi.ReadOnRobotMinedEntity(ptr).Entity, evVanish
	case fkapi.EventOnSpacePlatformMinedEntity:
		obj, what = fkapi.ReadOnSpacePlatformMinedEntity(ptr).Entity, evVanish
	case fkapi.EventOnEntityDied:
		obj, what = fkapi.ReadOnEntityDied(ptr).Entity, evVanish
	case fkapi.EventScriptRaisedDestroy:
		obj, what = fkapi.ReadScriptRaisedDestroy(ptr).Entity, evVanish

	case fkapi.EventOnPlayerRotatedEntity:
		obj, what = fkapi.ReadOnPlayerRotatedEntity(ptr).Entity, evModify
	case fkapi.EventOnPlayerFlippedEntity:
		obj, what = fkapi.ReadOnPlayerFlippedEntity(ptr).Entity, evModify
	default:
		return
	}

	if !obj.Valid() {
		return
	}

	ent := fkapi.LuaEntity{Object: obj}
	// `position` and `surface_index` are LuaControl's and LuaEntity inherits
	// them. The generator forwards inherited members onto the subclass now
	// (FKLUA-GAPS.md item 5, fixed upstream), so this is the obvious call
	// rather than a hand-built LuaControl over the same handle.
	//
	// THE POSITION IS READ BEFORE THE NAME, AND THAT ORDER IS A HEAP DECISION.
	// A name comes back as a Go string -- `getStr` copies the host's bytes,
	// because the arena underneath them is released when the call returns -- and
	// under `-gc=leaking` that copy is permanent, in the save and in every
	// multiplayer join. Measured at 32 B for one `express-transport-belt`
	// (CLAUDE.md, "The marathon save", leg D). A position and a surface index
	// are scalars decoded out of the return block and cost nothing.
	//
	// The engine's filter admits every belt-connectable on the map, so this
	// runs for every belt anyone lays or picks up anywhere -- which on a busy
	// server is the highest-multiplier term the guest has. So: ask the cheap
	// questions first, and buy the name only when the answer can depend on it.
	pos, err := ent.Position()
	if err != nil {
		return
	}
	si, err := ent.SurfaceIndex()
	if err != nil {
		return
	}
	// WHEN THE NAME CANNOT MATTER. Something DISAPPEARING or being ROTATED can
	// only concern this mod if it is one of our parts (which is a registered
	// tile), one of our network's visible interfaces (which stands ON a
	// registered tile), one of our hidden entities (which is on the hidden
	// surface), or a belt near a cluster (which is what the gate asks). All four
	// are answered from guest memory.
	//
	// An APPEARANCE is the case that has no cheap gate and never will: a part
	// placed alone in the middle of nowhere has no neighbours to be recognised
	// by, so the name is the only thing that can identify it and it is bought
	// unconditionally.
	if what != evAppear && si != hiddenIdx && !nearCluster(si, pos.X, pos.Y) {
		return
	}
	name, err := ent.Name()
	if err != nil {
		return
	}

	if name == PartName {
		k := key{s: si, x: floorTile(pos.X), y: floorTile(pos.Y)}
		f := uint32(0)
		if what == evAppear {
			// The only place a force is read. Two host calls, on the part path
			// alone -- never on the neighbour path, which is a position lookup
			// against the registry and makes no host call at all.
			var ok bool
			if f, ok = forceOf(obj); !ok {
				return
			}
		}
		onPart(k, f, what, minedBy, builtBy)
		return
	}

	// AN INCUMBENT'S PART, APPEARING AFTER THE MIGRATION SCAN. The one that
	// matters is a ghost out of a migrating player's old blueprint book, revived
	// by a construction robot; a script from another mod building one lands here
	// too. It is swapped for one of ours in place. A DISAPPEARANCE needs nothing:
	// a stub was never in this registry. See legacy.go.
	if name == LegacyPartName {
		if what == evAppear {
			legacyBuilt(si, pos)
		}
		return
	}

	if name == AuditName {
		if what == evAppear {
			auditAll()
			_, _ = ent.Destroy(fkapi.LuaEntityDestroyArgs{})
		}
		return
	}

	// The insert probe: the miner's-pocket arithmetic asked of a container, so
	// that the half of that path which needs no player can be pinned headlessly.
	// See probe.go.
	if name == ProbeName {
		if what == evAppear {
			noteInsertProbe(si, pos)
			_, _ = ent.Destroy(fkapi.LuaEntityDestroyArgs{})
		}
		return
	}

	// One of OUR prototypes, in an event we did not cause: the compiler never
	// raises, so this is something else copying or destroying a piece of a
	// network.
	if isOurs(name) {
		switch {
		case id == fkapi.EventOnEntityCloned:
			// A copy. It belongs to no slot and it sits exactly where the
			// rebuild wants to put a real one, so it goes now. Its contents go
			// with it: the clone minted them.
			_, _ = ent.Destroy(fkapi.LuaEntityDestroyArgs{})
			if verboseLog {
				logStart("destroyed a cloned network entity (")
				logS(name)
				logS(")")
				logEnd()
			}
		case what == evVanish:
			onNetworkLoss(si, pos.X, pos.Y, name)
		}
		return
	}

	// A BELT-CONNECTABLE APPEARING ON A PART'S TILE MEANS THE PART IS GONE, and
	// nothing else says so: `fast_replaceable_group` is symmetric, so the line
	// that lets a part replace a belt lets a belt replace a part -- and the
	// engine raises no event at all for the entity it replaced. One tile lookup
	// on the appearance path, a host call only on a hit, and it runs BEFORE the
	// neighbourhood walk below so that walk sees the registry the removal left.
	// See fastreplace.go.
	if what == evAppear {
		reapFastReplaced(si, floorTile(pos.X), floorTile(pos.Y), builtBy)
	}
	onNeighbour(si, pos.X, pos.Y, minedBy, builtBy)
}

// ---------------------------------------------------------------------------
// The events that report a region rather than an entity
// ---------------------------------------------------------------------------

// onUndo re-validates after an undo or a redo.
//
// The `actions` array is deliberately not read, and since the subscription
// masks it the host does not marshal it either. Every action it describes is
// replayed through the ordinary build and mine paths -- an undone removal
// becomes a ghost and then a build, an undone build becomes a deconstruction
// and then a mine -- so the registry is already right for almost all of them.
// The exception is an undone ROTATION, which changes a belt's direction and
// therefore a cluster's edge list without raising on_player_rotated_entity, and
// re-classifying is both simpler and stricter than trusting a decode of a
// variant table full of BlueprintEntities.
//
// Scoped to the player's surface, which is one host call's worth of scoping and
// the difference between re-classifying one planet and re-classifying a Space
// Age save's worth of them.
func onUndo(playerIndex uint32, what string) {
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(playerIndex)))
	if err == nil && o != nil {
		p := fkapi.LuaPlayer{Object: *o}
		if si, err := p.SurfaceIndex(); err == nil {
			revalidateSurface(si, what)
			return
		}
	}
	// No player to scope by -- a headless server, or an undo raised by another
	// mod with an index that resolves to nothing. Re-validate everything: the
	// conservative answer is the correct one, and this path is a keypress, not
	// a tick.
	revalidateAll(what)
}

func onAreaCloned(ev fkapi.OnAreaCloned) {
	if !ev.DestinationSurface.Valid() {
		return
	}
	si, err := fkapi.LuaSurface{Object: ev.DestinationSurface}.Index()
	if err != nil {
		return
	}
	a := ev.DestinationArea
	reconcileArea(si,
		floorTile(a.LeftTop.X), floorTile(a.LeftTop.Y),
		floorTile(a.RightBottom.X), floorTile(a.RightBottom.Y), "area clone")
}

// onBrushCloned reconciles the bounding box of the destination positions.
//
// A brush is a scattered set of tiles rather than a rectangle, and reconciling
// their bounding box is a superset of reconciling exactly them: the extra tiles
// hold nothing the clone touched, so the registry agrees with the world there
// already and the reconcile finds nothing to do.
//
// `ReadOnBrushCloned` copies `source_positions` into a fresh Go slice, which
// `-gc=leaking` keeps forever. It is bounded by the brush size and it is a
// player keypress rather than a tick, which is why this one is read through the
// generated struct and `on_undo_applied`'s unbounded `actions` array is not.
func onBrushCloned(ev fkapi.OnBrushCloned) {
	if !ev.DestinationSurface.Valid() {
		return
	}
	si, err := fkapi.LuaSurface{Object: ev.DestinationSurface}.Index()
	if err != nil {
		return
	}
	dx := ev.DestinationOffset.X - ev.SourceOffset.X
	dy := ev.DestinationOffset.Y - ev.SourceOffset.Y
	if len(ev.SourcePositions) == 0 {
		return
	}
	var x0, y0, x1, y1 int32
	for i := range ev.SourcePositions {
		x, y := ev.SourcePositions[i].X+dx, ev.SourcePositions[i].Y+dy
		if i == 0 {
			x0, y0, x1, y1 = x, y, x, y
			continue
		}
		if x < x0 {
			x0 = x
		}
		if x > x1 {
			x1 = x
		}
		if y < y0 {
			y0 = y
		}
		if y > y1 {
			y1 = y
		}
	}
	reconcileArea(si, x0, y0, x1, y1, "brush clone")
}

// onSurfaceGoing runs while the surface and everything on it is still valid.
func onSurfaceGoing(si uint32, why string) {
	if si == 0 {
		return
	}
	if si == hiddenIdx && hiddenIdx != 0 {
		hiddenSurfaceGoing(why)
		return
	}
	dropSurface(si, why)
}

// onSurfaceGone runs after. There is only one thing left that can need doing.
func onSurfaceGone(si uint32) {
	if si != 0 && si == hiddenIdx {
		hiddenSurfaceGone()
	}
}

// onNetworkLoss handles one of our own entities being destroyed by something
// that is not the compiler.
//
// The edge fingerprint cannot see this -- it describes the belts around a
// cluster, not the interfaces on it -- so the owning cluster is put on the
// TEARDOWN queue as well as the build queue, which is what forces a rebuild
// rather than a re-classification.
func onNetworkLoss(si uint32, x, y float64, name string) {
	if si == hiddenIdx && hiddenIdx != 0 {
		// A hidden entity. Which slot it was in says which cluster owns it.
		if x < 0 || y < 0 {
			return
		}
		col, row := uint32(x)/slotW, uint32(y)/slotH
		if col >= slotCols {
			return
		}
		slot := row*slotCols + col + 1
		auditRoots = liveRootList(auditRoots)
		for i := range auditRoots {
			if ni, ok := nets[auditRoots[i]]; ok && ni.slot == slot {
				markDead(auditRoots[i])
				markLive(auditRoots[i])
			}
		}
		requestFlush()
		if verboseLog {
			logStart("a hidden ")
			logS(name)
			logS(" was destroyed from outside; slot ")
			logU(slot)
			logS(" queued for a rebuild")
			logEnd()
		}
		return
	}
	tx, ty := floorTile(x), floorTile(y)
	hit := false
	for dy := int32(-2); dy <= 2; dy++ {
		for dx := int32(-2); dx <= 2; dx++ {
			id, ok := index[key{si, tx + dx, ty + dy}]
			if !ok {
				continue
			}
			markDead(find(id))
			markLive(find(id))
			hit = true
		}
	}
	if !hit {
		return
	}
	requestFlush()
	if verboseLog {
		logStart("a visible ")
		logS(name)
		logS(" was destroyed from outside; its cluster was queued for a rebuild")
		logEnd()
	}
}

// onPart updates the registry and queues whatever the change touched.
//
// THE REGISTRY UPDATE IS NOT DEFERRABLE and the recompile is. `AddPart` and
// `RemovePart` read nothing from the world -- the tile and the force are
// already in hand -- so they happen here, inside the event, and the queue they
// fill is drained once per tick from fk_on_deferred. Both the root that existed
// BEFORE and the roots that exist AFTER are queued by cluster.go; the flush runs
// every teardown before any build, because a split leaves the old network's
// visible interfaces standing on tiles that now belong to a different cluster.
// `minedBy` is the player_index for on_player_mined_entity and 0 for every
// other path -- a robot, a death, a script destroy. It reaches the registry
// only so that a removal which DISSOLVES a cluster can hand the drained network
// to the miner rather than to the ground; see carry.go.
func onPart(k key, force uint32, what int, minedBy, builtBy uint32) {
	changed := false
	switch what {
	case evAppear:
		changed = AddPart(k, force)
		// A PART A PLAYER PLACED IS A PART THAT MIGHT NOT FIT. The registry has
		// taken it either way -- the cluster is bigger now whatever the compiler
		// decides -- and the note is what lets the flush a tick later hand it
		// back if the shape it made is past plan.MaxPorts. Scalars only, no host
		// call, and nothing at all unless a player built it. See limit.go.
		if changed {
			noteBuiltByPlayer(k.s, k.x, k.y, force, builtBy, true)
		} else if id, dup := index[k]; dup && builtBy != 0 {
			// THE WAKE RACE'S SECOND HEAD (field report, 2026-08-05, the
			// bridge gesture). When this event is the FIRST of a session, the
			// rebuild-from-world at the top of this dispatch scanned a world
			// that already contained this part -- the engine places before the
			// event fires -- and registered it, so AddPart reports no change.
			// Any refusal that rebuild issued fired before this note could
			// exist (a rebuild's refusal never arms the feedback memo, for
			// exactly this reason; limit.go, refuseOverLimit), so the informed
			// retry still needs the
			// note recorded and the cluster queued, or an over-limit piece
			// stands with nothing but a chat line. Outside the wake race a
			// duplicate PLAYER build event is a mod raising built twice for
			// one entity, and this is then a note plus a flush that skips on
			// the fingerprint. builtBy is 0 for scripts and robots, whose
			// wake-race outcome (the piece stands, the force is told) is the
			// designed one already.
			noteBuiltByPlayer(k.s, k.x, k.y, force, builtBy, true)
			markLive(find(id))
			logState()
			requestFlush()
			return
		}
	case evVanish:
		changed = RemovePartMinedBy(k, minedBy)
	default:
		return // a part cannot be rotated
	}
	if !changed {
		return
	}
	logState()
	requestFlush()
}

// onNeighbour handles everything that is NOT one of our parts: the belts,
// undergrounds, splitters and loaders whose orientation is what decides a
// cluster's inputs and outputs.
//
// The gate is a TILE LOOKUP, not an API read. By the time we are here the
// position is already in hand, and asking the registry whether any cluster tile
// is within two tiles is a handful of in-guest probes against a map that is
// only ever point-queried. Only a hit costs anything more.
//
// Two tiles rather than one: a splitter is two tiles wide and its position sits
// on the boundary between them, so a splitter whose far half touches a cluster
// reports a position two tiles away from the tile that matters.
//
// THIS IS THE PATH A 100-BELT PASTE ALONG A BALANCER'S EDGE TAKES, and it is the
// reason the flush is deferred: a hundred belts used to be a hundred
// classifications of the same cluster, ninety-nine of which the fingerprint then
// threw away. Now they are a hundred queue insertions -- deduplicated, no host
// call -- and one compile.
// nearCluster reports whether any registered part tile is within two tiles of
// (x, y) on surface `surf`.
//
// Twenty-five point queries against `index`, which is guest memory, and NOT ONE
// HOST CALL. It is the same neighbourhood onNeighbour walks and for the same
// reason: a splitter is two tiles wide and its position sits on the boundary
// between the two tiles it covers, so a splitter whose far half touches a
// cluster reports a position two tiles from the tile that matters.
func nearCluster(surf uint32, x, y float64) bool {
	tx, ty := floorTile(x), floorTile(y)
	for dy := int32(-2); dy <= 2; dy++ {
		for dx := int32(-2); dx <= 2; dx++ {
			if _, ok := index[key{surf, tx + dx, ty + dy}]; ok {
				return true
			}
		}
	}
	return false
}

// `minedBy` is the player_index for on_player_mined_entity and 0 for everything
// else -- a belt laid, a belt a robot took, a belt that died. It is here for the
// same reason it is on onPart, and the reason is the field report below.
func onNeighbour(surf uint32, x, y float64, minedBy, builtBy uint32) {
	tx, ty := floorTile(x), floorTile(y)
	found := false
	nearForce := uint32(0)
	for dy := int32(-2); dy <= 2; dy++ {
		for dx := int32(-2); dx <= 2; dx++ {
			k := key{surf, tx + dx, ty + dy}
			id, ok := index[k]
			if !ok {
				continue
			}
			markLive(find(id))
			if nearForce == 0 {
				nearForce = pforce[id]
			}
			// A BELT MINED AT A BALANCER'S EDGE SHRINKS THE BALANCER, and until
			// 2026-08-02 nobody was credited for it. Removing an output takes a
			// PORT off the machine, and a port crossing back over a power-of-two
			// boundary halves the butterfly -- so the network the recompile
			// builds is materially smaller than the one it drained, the
			// reinsertion overflows by carry.go's fourth decision, and the
			// overflow went to the floor because the beneficiary was recorded
			// only when a PART was mined. Place an output belt on a running
			// balancer and mine it again and you watch it happen; that is the
			// field report.
			//
			// The claim goes on the PART'S tile, `k`, and not on the belt's.
			// `k` is the registry key this loop just looked up, so it is a tile
			// of the network by construction -- which is what a pool's box can
			// answer -- whereas the belt is one tile outside that box by
			// construction and would answer to nobody. The force is `pforce[id]`
			// for the same reason it is on the part path: two forces' boxes are
			// adjacent by construction and a claim must not be answered by the
			// neighbour's pool.
			//
			// KEYED BY TILE RATHER THAN BY THE ROOT THIS LOOP JUST RESOLVED, and
			// that is not consistency for its own sake: `flushLive` re-resolves
			// every queued root through `find` a tick later, and a part mined
			// elsewhere in the same tick re-roots the survivors at the smallest
			// surviving node id. The root is a number that may stop being one;
			// the ground does not move.
			//
			// NO HOST CALL, which is this path's defining property -- the tile,
			// the force and the player are all already in hand, and Claims.Add
			// returns on the zero player before it touches anything. A belt laid
			// or picked up anywhere on the map is the guest's
			// highest-multiplier event and it pays one compare.
			if minedBy != 0 {
				noteMinedByPlayer(k, pforce[id], minedBy)
			}
			found = true
		}
	}
	if !found {
		return
	}
	// A BELT A PLAYER LAID BESIDE A BALANCER IS A PORT THEY MIGHT NOT BE ALLOWED
	// TO HAVE, and the sixty-fifth of them is the whole of the 2026-08-04 pass.
	// ONE note per belt rather than one per part tile the loop above walked --
	// the tile here is the BELT'S OWN, because handing it back means re-finding
	// it, which is the exact opposite of the claim above, whose tile is always
	// the network's. The force is the first cluster this belt touches, which is
	// the only one that can refuse over it. See limit.go.
	if builtBy != 0 {
		noteBuiltByPlayer(surf, tx, ty, nearForce, builtBy, false)
	}
	requestFlush()
}

// floorTile turns a map position into the tile containing it.
//
// A 1x1 entity sits at a tile's centre, so this is (n + 0.5) -> n for positive
// coordinates and (-n - 0.5) -> -n-1 for negative ones. A plain int32 cast
// truncates TOWARDS ZERO, which would fold the two tiles either side of the
// origin onto each other -- so the negative case is corrected explicitly rather
// than by importing math.Floor for one call.
func floorTile(v float64) int32 {
	i := int32(v)
	if v < 0 && float64(i) != v {
		i--
	}
	return i
}

// ---------------------------------------------------------------------------
// Logging.
//
// Every line carries `[BBB] ` so a headless run can grep the transitions out of
// Factorio's log, which is how M1's exit criterion is checked. It is one line
// per placement, which is a lot of lines during a 3,200-part `--create` -- so
// none of them allocates. Every line below is assembled in the one reusable
// buffer in logline.go, and THAT is the heap diet: building these with `+` and
// `strconv` was, measured, the entire guest heap (16 -> 64 MiB at n=200).
// ---------------------------------------------------------------------------

// logPlural writes "1 part" or "N parts".
func logPlural(n uint32) {
	logU(n)
	if n == 1 {
		logS(" part")
		return
	}
	logS(" parts")
}

func logFormed(root uint32) {
	if !verboseLog {
		return
	}
	logStart("cluster ")
	logU(root)
	logS(" formed (")
	logPlural(csize[root])
	logS(")")
	logEnd()
}

func logMerge(keep, gone uint32) {
	if !verboseLog {
		return
	}
	logStart("merge ")
	logU(keep)
	logS("+")
	logU(gone)
	logS("->")
	logU(keep)
	logS(" (")
	logPlural(csize[keep])
	logS(")")
	logEnd()
}

func logShrunk(root uint32) {
	if !verboseLog {
		return
	}
	logStart("cluster ")
	logU(root)
	logS(" shrunk (")
	logPlural(csize[root])
	logS(")")
	logEnd()
}

// logDissolvedBy is the dissolve line, plus who mined the last part where a
// player did.
//
// The suffix is the only evidence a headless run can give that `player_index`
// reached the registry at all -- the pocket itself needs a player and a
// `--create` has none (CLAUDE.md, "What M3 implements and does NOT verify"), and
// on_player_mined_entity is not one of the events script.raise_event will raise.
// The unsuffixed form is what every other removal path writes, and every suite
// asserting on it is unchanged.
func logDissolvedBy(root uint32, player uint32) {
	if !verboseLog {
		return
	}
	logStart("cluster ")
	logU(root)
	logS(" dissolved")
	if player != 0 {
		logS(", mined by player ")
		logU(player)
	}
	logEnd()
}

func logSplit(old uint32, roots []uint32) {
	if !verboseLog {
		return
	}
	logStart("split ")
	logU(old)
	logS(" -> ")
	for i, r := range roots {
		if i > 0 {
			logS(",")
		}
		logU(r)
	}
	logS(" sizes=")
	for i, r := range roots {
		if i > 0 {
			logS(",")
		}
		logU(csize[r])
	}
	logEnd()
}

// snap is reused so a state line does not allocate a fresh slice each time.
var snap []uint32

// logState is the assertable line: the whole registry, in one deterministic
// shape, after every event that changed it.
//
// THIS IS THE ONE THAT MATTERED. It runs once per part placed and lists up to
// 64 cluster sizes; written with `s += u32(n)` it left ~9 KB of dead strings
// behind per call, and `-gc=leaking` keeps all of it. 3,200 parts is ~26 MB of
// permanent heap for a line nothing but the test harness reads.
func logState() {
	if !verboseLog {
		return
	}
	snap = Snapshot(snap)
	logStart("state clusters=")
	logU(uint32(len(snap)))
	logS(" parts=")
	logU(nParts)
	logS(" sizes=")
	for i, n := range snap {
		if i > 0 {
			logS(",")
		}
		if i == 64 {
			logS("...")
			break
		}
		logU(n)
	}
	logEnd()
}

func main() {}
