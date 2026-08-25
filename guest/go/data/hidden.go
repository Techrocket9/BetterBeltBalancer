package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// The hidden network's prototypes: everything the compiler places, and nothing
// a player can ever see, hold, blueprint or clone.
//
// Four prototypes, all clones of base ones with the speed raised and the
// player-facing surface removed:
//
//	bbb-linked-belt    the edge interface AND the network's row-crossing jumper
//	bbb-belt           the wire between stages
//	bbb-splitter       the balancing element (2 tiles tall)
//	bbb-lane-splitter  the 1x1 lane-fidelity stage at the inputs
//
// ---------------------------------------------------------------------------
// THE COLLISION MASK ON bbb-linked-belt WAS THE WHOLE ARCHITECTURE, AND ON
// FACTORIO 2.1 THE DOOR IS SHUT.
//
// Spike S1 probed 14 masks on 2.0.77. The engine VALIDATES any belt-connectable
// whose `layers` differ from the type default -- and the validation demands the
// mask collide with transport-belt's and with itself, so every deviation fails
// at load. But when `layers` is EXACTLY the type default the validation was
// skipped entirely, while `not_colliding_with_itself` was still honoured at
// runtime. That was the one and only door to putting two (or four)
// belt-connectables on one tile, which is what let a 1x1 balancer part carry an
// input interface on one side and an output interface on another.
//
// It was a loophole and it is closed. 2.1 fixed the equals-compare that skipped
// the validation, so the check now runs on every belt-connectable and demands
// the mask collide with itself; probed exhaustively on 2.1.14, no mask design
// passes and no runtime bypass exists (`create_entity` nils, `teleport` returns
// false). boskid's answer to the interface request (forums t=135830) explains
// the invariant it protects: belt-to-belt connections are NOT SAVED, they are
// re-derived at load, and one belt-connectable per tile is what makes that
// re-derivation unambiguous.
//
// SO THE FLAG IS EMITTED ON 2.0.x AND NEVER ON 2.1.x, and with it the marker
// prototype `bbb-can-stack` that tells the guest which world it is in. The rule
// the guest enforces when the marker is absent is at most ONE BELT PER BALANCER
// PART, because one part tile may carry at most one interface linked belt.
// agents/single-edge.md is the whole port; guest/go/sedge.go is the runtime half
// of this file.
//
// THE VERSION GUARD IS WHAT KEEPS A MISPACKAGED ZIP FROM BRICKING A 2.1 LOAD,
// and it fails SAFE: anything guest/go/engine cannot read as 2.0.x is treated as
// 2.1, because emitting the flag on 2.1 refuses the mod at load while not
// emitting it on 2.0 merely costs the multi-edge geometry the guest would then
// refuse to build anyway.
//
// Do not "tidy" the layer list. On 2.0 adding or removing one entry moves this
// prototype from the unvalidated path to the validated one, and the mod stops
// loading. It is `beltLayers()` in entity.go for exactly that reason -- one
// list, three callers, no way for one copy to drift.
// ---------------------------------------------------------------------------
//
// SPEED = 0.25 and that number is a ceiling, not a preference. A transport line
// accepts at most one item per tick per lane, so 0.25 tiles/tick -- one item
// width per tick -- is the fastest a belt can still run COMPRESSED: 60
// items/s/lane, 120/s/belt, 2.67x express and 1.33x turbo. Going higher buys
// nothing (the line cannot be fed any faster) and starts to open gaps.
//
// 0.25 is enough by construction: the network spreads N input belts over
// P >= N lines, so no hidden line ever carries more than one visible belt's
// rate. A modded belt faster than 0.25 would bottleneck here; nothing in base or
// Space Age is.
//
// THE SPEED IS A CANDIDATE FOR DERIVATION rather than a constant -- the fastest
// belt in the game is one `fkdata.Keys("transport-belt")` walk away, and that is
// the sorted-enumeration primitive's whole purpose. Deliberately not done here:
// this pass is a behaviour-preserving port and a derived speed is a different
// number in a modded game, which is a FEATURE with its own gate.
const hiddenSpeed = 0.25

// ---------------------------------------------------------------------------
// AND THE VISUALS ARE BLANKED, WHICH IS NOT TIDYING. A clone of a base belt
// keeps every picture the base prototype had, and `bbb-linked-belt` is the one
// hidden prototype that stands on a surface a PLAYER LOOKS AT: the compiler puts
// one on a balancer part's own tile for every edge of the cluster.
//
// The part sprite is 64 px at scale 0.5 -- exactly one tile, opaque in all 47
// cells (checked pixel by pixel) -- and it draws at render layer `object`, so it
// covers everything the interface draws ON ITS OWN TILE. What it cannot cover is
// what the interface draws OUTSIDE that tile, and base's linked belt draws a lot
// of it:
//
//	structure           192x192 at scale 0.5 = THREE TILES BY THREE, at
//	                    `structure_render_layer = "object"` -- the same layer the
//	                    part is on, spilling onto all eight neighbours
//	belt_animation_set  the running belt, plus the starting/ending patches, which
//	                    are drawn past the tile edge by design
//
// On a solid rectangle of parts the neighbours' own sprites hide most of that.
// On a shape with a NOTCH -- a 2x2 with one corner missing, which is what the
// field report was about -- the empty tile is covered by nothing, and every
// interface around it paints into it. Blanking is the fix, and it is free:
// nothing about the hidden network is meant to be looked at.
//
// Every replacement is "a valid sprite that happens to be one transparent pixel
// of __core__/graphics/empty.png" rather than nil. Both are legal -- every field
// blanked below is optional in the prototype API except one, noted where it is
// -- but headless Factorio never opens a sprite file and test/check-sprites.py
// only checks that the ones we NAME exist, so the GRAPHICAL client is the first
// thing that would notice a shape the engine did not like. A drawable-but-empty
// sprite is the shape this repo has already proved; an absent one asks the
// renderer to take a branch nothing here has exercised.
//
// WHAT THIS DOES NOT FIX: the ITEMS. There is no prototype field anywhere that
// suppresses the drawing of items on a belt-connectable, and no linked-belt
// equivalent of `LoaderPrototype::belt_length` to shorten the stretch they are
// drawn over. See CLAUDE.md, "The tan streak".
// ---------------------------------------------------------------------------

// blankAnimation is one transparent pixel, as an Animation. Valid wherever an
// Animation, Animation4Way, Sprite or Sprite4Way is wanted.
//
//go:noinline
func blankAnimation() fkdata.V {
	return obj(
		f("filename", str(emptyPNG)),
		f("priority", str("extra-high")),
		f("width", num(1)),
		f("height", num(1)),
		f("frame_count", num(1)),
		f("line_length", num(1)),
	)
}

// blankBeltAnimationSet is a TransportBeltAnimationSet (and, with the same
// fields, the WithCorners form: its extra indices are 5..12 and are covered by
// the same 20 directions) whose every direction and every patch is that one
// pixel. `animation_set` is the one non-optional member of the set, which is why
// the set is replaced whole rather than emptied field by field.
//
// `direction_count = 20` is the one number here that is not arbitrary: a belt
// animation set is indexed by direction and by patch, and the defaults run up to
// `ending_east_index = 20`. Base's own belt sets are `direction_count = 20` for
// exactly that reason. empty.png is 64x64, so 20 rows of one pixel fit inside it
// with room to spare.
//
//go:noinline
func blankBeltAnimationSet() fkdata.V {
	return obj(f("animation_set", obj(
		f("filename", str(emptyPNG)),
		f("priority", str("extra-high")),
		f("width", num(1)),
		f("height", num(1)),
		f("frame_count", num(1)),
		f("line_length", num(1)),
		f("direction_count", num(20)),
	)))
}

// hiddenFlags keeps a hidden entity out of every path that could copy it. The
// network is ALWAYS recompiled from visible state; an entity that survived into
// a blueprint or a clone would be a second, untracked network.
//
//go:noinline
func hiddenFlags() fkdata.V {
	return strs(
		"placeable-neutral",
		"player-creation",
		"not-blueprintable",
		"not-deconstructable",
		"not-on-map",
		"no-copy-paste",
		"not-selectable-in-game",
		"not-upgradable",
		"not-in-kill-statistics",
		"not-in-made-in",
		"hide-alt-info",
	)
}

// cloneHidden is `clone(t, base, name)` from the Lua, and the ONE structural
// difference the port makes anywhere.
//
// The Lua deep-copied a prototype into a local table, stripped and blanked THAT,
// and extended the result -- so what `data:extend` saw was already hidden. Here
// `fkdata.Clone` is the deep copy and it registers the copy immediately, so the
// strip and the blank are patches applied to a prototype that is already in
// data.raw. The END STATE is identical, and it is the end state Factorio
// validates: `data:extend` only files a table under data.raw[type][name], and
// the prototype check is C++ after every mod's stage has run. Mutating a
// registered prototype is what every data-updates stage in the game does.
//
// AND THE CLONE IS A HOST PRIMITIVE FOR A REASON THAT IS ABOUT FIDELITY, NOT
// SPEED. These four carry about 1,000 scalar leaves between them after
// stripping, every one of them base's own bytes deep-copied by the engine.
// Reading a prototype into the guest and writing it back would re-serialise all
// of them, so any field the value model cannot express, any float that does not
// round-trip and any key it drops would change the prototype SILENTLY while the
// mod still loads. Under a clone the untouched leaves are literally the bytes
// base shipped, and the patches below reach about forty of them.
//
//go:noinline
func cloneHidden(typ, base, name string) at {
	p := at{typ, name}
	fkdata.Clone(typ, base, name)
	p.set(num(hiddenSpeed), "speed")
	strip(p)
	p.set(blankBeltAnimationSet(), "belt_animation_set")
	return p
}

// strip turns a cloned prototype into a hidden one: no item, no upgrade path, no
// fast-replace group (it would offer our belt as a replacement for a real one),
// no selection box, and none of the per-tick cost of being visible.
//
//go:noinline
func strip(p at) {
	p.drop("minable")
	p.drop("placeable_by")
	p.set(yes, "hidden")
	p.set(yes, "hidden_in_factoriopedia")
	p.drop("next_upgrade")
	p.drop("fast_replaceable_group")
	p.set(hiddenFlags(), "flags")
	p.drop("selection_box")
	p.set(no, "selectable_in_game")
	p.set(num(1), "max_health")
	p.drop("corpse")
	p.drop("dying_explosion")
	p.drop("working_sound")
	p.drop("open_sound")
	p.drop("close_sound")
	p.drop("impact_category")
}

//go:noinline
func hidden() {
	// The edge interface and the row-crossing jumper.
	linked := cloneHidden("linked-belt", "linked-belt", "bbb-linked-belt")
	linked.set(obj(f("layers", beltLayers())), "collision_mask")
	if canStack() {
		// The door, on the engine that still has one. Without it a second linked
		// belt cannot share the tile, and without that a 1x1 part cannot be both
		// an input and an output -- which is the rule guest/go/sedge.go enforces
		// everywhere else.
		//
		// Set as its own key rather than folded into the mask above, because the
		// mask's `layers` must stay EXACTLY the type default (see the header)
		// and this is a sibling of `layers`, not a member of it.
		linked.set(yes, "collision_mask", "not_colliding_with_itself")
	}
	// A linked belt remembers its partner in a blueprint and re-links on paste,
	// and can be carried through a clone the same way. Both would resurrect a
	// network the compiler does not know about, so both are refused at the
	// prototype.
	linked.set(no, "allow_blueprint_connection")
	linked.set(no, "allow_clone_connection")
	// THE THREE-BY-THREE SPRITE, gone. Every field of LinkedBeltStructure is
	// optional and every one of them is replaced rather than dropped. This is the
	// one that was actually being seen: it is the only prototype here that stands
	// on the visible surface.
	linked.set(obj(
		f("direction_in", emptySprite()),
		f("direction_out", emptySprite()),
		f("direction_in_side_loading", emptySprite()),
		f("direction_out_side_loading", emptySprite()),
		f("back_patch", emptySprite()),
		f("front_patch", emptySprite()),
	), "structure")

	belt := cloneHidden("transport-belt", "express-transport-belt", "bbb-belt")
	belt.drop("related_underground_belt")

	splitter := cloneHidden("splitter", "express-splitter", "bbb-splitter")
	splitter.set(str("bbb-belt"), "related_transport_belt")
	// Belt-and-braces: these three never leave the hidden surface, and a surface
	// with no generated chunks is not a place anything is looked at. Blanked
	// anyway so that the rule is "nothing the compiler places draws anything"
	// rather than "nothing the compiler places draws anything WHERE IT MATTERS",
	// which is the kind of qualification the tan streak got in through.
	splitter.set(blankAnimation(), "structure")
	splitter.set(blankAnimation(), "structure_patch")
	splitter.set(emptySprite(), "frozen_patch")

	lane := cloneHidden("lane-splitter", "lane-splitter", "bbb-lane-splitter")
	// LaneSplitterPrototype::structure is the one MANDATORY picture in this file
	// (`optional = false` in the prototype API), so it is replaced, never
	// dropped.
	lane.set(blankAnimation(), "structure")
	lane.set(blankAnimation(), "structure_patch")

	// The audit marker and the insert probe, which are not clones of anything.
	// One Extend, as the Lua's one `data:extend` was.
	fkdata.Extend(auditMarker(), insertProbe())

	// THE CAPABILITY MARKER, and it is defined here rather than unconditionally
	// because its whole meaning is "the linked belt above carries
	// `not_colliding_with_itself`, so this engine can stack two of them on one
	// tile". The guest cannot read a prototype's collision mask and must not
	// carry a version number of its own; a point lookup of this name against
	// `prototypes.entity` answers the question in two host calls and no
	// allocation, and the guest's belief cannot drift from the prototype's actual
	// capability because the two are one `if`.
	//
	// It is the `bbb-legacy-stub` idiom exactly (legacy.go), for the same reason
	// and with the same shape: a prototype that exists to be looked up and that
	// nothing ever places. See guest/go/sedge.go, `multiEdgeAllowed`.
	if canStack() {
		fkdata.Extend(obj(
			f("type", str("simple-entity")),
			f("name", str("bbb-can-stack")),
			f("hidden", yes),
			f("hidden_in_factoriopedia", yes),
			// NOT hiddenFlags(): that list carries `placeable-neutral` and
			// `player-creation`, and a placeable simple-entity must declare an
			// `icon` -- the 2.0 loader refuses the mod without one ("Key
			// \"icon\" not found"), which the first graphical 2.0 session found,
			// because this prototype exists only on 2.0.x and nothing on 2.1
			// ever loads the branch. lookupOnlyFlags is the load-tested shape
			// for a prototype that exists to be looked up and never placed.
			f("flags", lookupOnlyFlags()),
			f("max_health", num(1)),
			f("collision_mask", obj(f("layers", obj()))),
			f("collision_box", box(0, 0, 0, 0)),
			f("selectable_in_game", no),
			// The graphical client validates every sprite path at load and
			// headless does not, which is how a stale filename here once
			// survived every suite and then refused to load in the real game.
			// The engine's own empty sprite has no file dependency of ours at
			// all.
			f("picture", emptySprite()),
		))
	}
}

// lookupOnlyFlags is the flag list for a prototype that exists to be LOOKED UP
// and never placed: `bbb-can-stack` here and `bbb-legacy-stub` in legacy.go.
//
// It is hiddenFlags() without `placeable-neutral` and `player-creation`, and
// that difference is load-bearing rather than tidy -- a placeable simple-entity
// must declare an `icon` or the 2.0 loader refuses the mod outright. Both
// markers declare no icon, so neither may be placeable.
//
//go:noinline
func lookupOnlyFlags() fkdata.V {
	return strs("not-blueprintable", "not-deconstructable", "not-on-map",
		"no-copy-paste", "not-selectable-in-game", "not-upgradable",
		"not-in-kill-statistics", "not-in-made-in", "hide-alt-info")
}

// auditMarker is the audit request. Not part of the network: a one-shot ask.
//
// Placing one (only a script can -- there is no item, no recipe and no
// technology that yields it) asks the guest to compare every cluster's stored
// edge fingerprint against a fresh classification of the world, log the result
// and repair whatever drifted. It destroys itself on the way out.
//
// It exists because the alternative is a Lua reimplementation of the compiler
// inside the test harness, which would assert that two implementations agree
// rather than that one is right. It ships rather than living in the test mod so
// that the same question can be asked of a real save that is behaving oddly.
//
// Collision mask is EMPTY on purpose: the marker has to be placeable on a tile
// that already holds a belt or a balancer part.
//
//go:noinline
func auditMarker() fkdata.V {
	return obj(
		f("type", str("simple-entity")),
		f("name", str("bbb-audit")),
		f("hidden", yes),
		f("hidden_in_factoriopedia", yes),
		f("flags", hiddenFlags()),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("max_health", num(1)),
		f("collision_mask", obj(f("layers", obj()))),
		f("collision_box", box(0, 0, 0, 0)),
		f("selectable_in_game", no),
		// Never rendered (selectable_in_game = false, zero-size boxes), but the
		// GRAPHICAL client still validates every sprite path at load -- headless
		// does not, which is how a stale filename here survived every suite and
		// benchmark and then refused to load in the real game. An invisible
		// marker gets the engine's empty sprite and no file dependency of ours
		// at all.
		f("picture", emptySprite()),
	)
}

// insertProbe is the same shape as the audit marker and for the same reason: a
// one-shot request a script places, which the guest answers in the log and then
// destroys.
//
// What it asks is "does the boundary hand this engine the item COUNT I gave
// it?" -- placed on a container, the guest offers that container a known number
// of a known item through the very function the miner's pocket uses, and reports
// what came back. It exists because that arithmetic was unverifiable headlessly
// (a --create has no player) and a player found a defect in it that seven suites
// could not; `insert` is a LuaControl member, and a chest is a LuaControl, so a
// chest can be asked the question a player's pockets could not be.
//
// UNLIKE THE AUDIT MARKER IT IS DEFERRED: the answer arrives on the NEXT tick,
// because the pocket runs inside the deferred flush and a probe that answered
// inside its own build event would be testing the call from a place the pocket
// never makes it. A --create never reaches a tick, so a marker placed in on_init
// reports on the first tick of the run rather than into the save.
//
// Empty collision mask, like the audit marker: it has to be placeable on a tile
// that already holds the container it is about to fill.
//
// THE COLLISION BOX IS A TILE, AND UNLIKE THE AUDIT MARKER'S THAT MATTERS.
// Factorio snaps a placed entity to the grid its bounding box implies, so the
// audit marker's zero-size box snaps to a tile CORNER: one created at
// {x + 0.5, y + 0.5} reports its position as {x + 1, y + 1}. The audit does not
// care where it landed, and this does -- it has to find the container under
// itself. A tile-sized box snaps to the tile CENTRE, exactly as the balancer
// part does, so the probe's position is the tile it was asked for.
//
//go:noinline
func insertProbe() fkdata.V {
	return obj(
		f("type", str("simple-entity")),
		f("name", str("bbb-insert-probe")),
		f("hidden", yes),
		f("hidden_in_factoriopedia", yes),
		f("flags", hiddenFlags()),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("max_health", num(1)),
		f("collision_mask", obj(f("layers", obj()))),
		f("collision_box", box(-0.35, -0.35, 0.35, 0.35)),
		f("selectable_in_game", no),
		f("picture", emptySprite()),
	)
}
