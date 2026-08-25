package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// THE LEGACY STUB: what keeps a Belt Balancer 2/3 balancer alive long enough for
// the guest to adopt it.
//
// WHY A STUB AND NOT A migrations/*.json RENAME. Factorio's prototype-migration
// files are applied ONCE PER SAVE PER FILE and the fact that they ran is
// remembered by file name. So a rename file shipped by this mod would be
// recorded as applied on the FIRST load after this mod is installed -- which,
// for the player this feature is for, is a load on which the incumbent is still
// present and its balancers must not be touched -- and could never fire again on
// the later load where the incumbent is gone. It also has no way to express "do
// nothing while that other mod is installed", and what a rename does when BOTH
// the old and the new prototype exist is not documented. The whole feature is a
// decision taken at RUNTIME, from script.active_mods, so it belongs in the
// control guest; this exists only to stop the engine deleting the evidence
// before that guest can look at it.
//
// WHAT THE ENGINE DOES WITHOUT IT. When a mod is removed, every entity whose
// prototype went with it is deleted at load, silently, before any script runs. A
// player who swaps Belt Balancer 2 for this mod would find every balancer part
// gone and every belt around them still standing. Defining a prototype of the
// SAME NAME and a COMPATIBLE TYPE is what makes the entities survive that load
// -- it is the same mechanism a rename file uses, applied by existence rather
// than by a one-shot file.
//
// WHEN IT DEFINES ANYTHING. Only when nobody else has. THIS IS THE
// data-final-fixes HOOK AND THAT IS THE WHOLE REASON THERE IS A THIRD HOOK:
// data-final-fixes runs after every mod's data and data-updates stages, so an
// incumbent that is still installed has already defined `balancer-part` and this
// does nothing at all -- which is the "leave it alone while it is installed"
// half of the feature, enforced by the engine's own load order rather than by a
// list of mod names. The runtime half is the `bbb-legacy-stub` marker below.
//
// THE TYPE TEST IS ON `simple-entity-with-force` AND THAT IS THE RIGHT TABLE.
// Belt Balancer, Belt Balancer Performance, Belt Balancer 2 and Belt Balancer 3
// all define `balancer-part` as exactly that type, and the stub has to be that
// type anyway: an entity survives its prototype's disappearance only when the
// name comes back under a compatible type. If some other mod were to define an
// entity named `balancer-part` under a DIFFERENT entity type, Factorio refuses
// to load at all -- entity names are unique across entity types -- so that case
// is loud and immediate rather than silently wrong.
//
// # The undeclared dependency the port killed
//
// The Lua this replaces called `util.empty_sprite()` and never required `util`.
// It worked because `util` is a GLOBAL at the data stages (base sets it) and
// because prototypes/hidden.lua had required it earlier in the same state --
// two facts about load order that nothing stated and nothing checked, in a file
// whose entire job is to run after every other mod. Reordering the requires, or
// dropping hidden.lua's own `local util = ...`, would have broken this one.
// There is no `util` here and no order to get wrong: `emptySprite()` in value.go
// is the four fields core's function returns, written out.
const (
	legacyEntity = "balancer-part"
	legacyMarker = "bbb-legacy-stub"
)

//go:noinline
func legacy() {
	// Two INDEPENDENT probes, tested separately because the two can in principle
	// come apart -- a mod could define the entity and not the item, or the
	// reverse. `Get` answering false is the normal "has anybody defined this"
	// answer rather than an error, which is the one case the fkdata ABI reports
	// with a status instead of raising.
	_, entityTaken := fkdata.Get("simple-entity-with-force", legacyEntity)
	_, itemTaken := fkdata.Get("item", legacyEntity)

	if !entityTaken {
		fkdata.Extend(legacyStubEntity(), legacyStubMarker())
	}
	if !itemTaken {
		fkdata.Extend(legacyStubItem())
	}
}

//go:noinline
func legacyStubEntity() fkdata.V {
	return obj(
		f("type", str("simple-entity-with-force")),
		f("name", str(legacyEntity)),

		// WHAT IS COPIED FROM THE INCUMBENT AND WHY. The type and the collision
		// box are what make a standing entity survive the load; the collision
		// mask is what keeps it refusing to share a tile with a belt for the one
		// load it is still standing; the mining result and `placeable_by` are
		// what make a player who mines one, or a construction robot that revives
		// an old blueprint's ghost of one, end up holding this mod's item
		// instead of nothing.
		f("collision_box", box(-0.35, -0.35, 0.35, 0.35)),
		f("selection_box", box(-0.5, -0.5, 0.5, 0.5)),
		f("collision_mask", obj(f("layers", beltLayers()))),
		f("max_health", num(170)),
		f("corpse", str("splitter-remnants")),
		f("resistances", list(obj(f("type", str("fire")), f("percent", num(60))))),
		f("minable", obj(f("mining_time", num(0.1)), f("result", str("bbb-balancer-part")))),
		f("placeable_by", obj(f("item", str("bbb-balancer-part")), f("count", num(1)))),

		// A PLAYER CANNOT BUILD ONE, AND NO FLAG IS WHAT STOPS THEM. Nothing
		// places this prototype: the stub item below has
		// `place_result = "bbb-balancer-part"`, there is no recipe and no
		// technology, and the only item whose `placeable_by` names it is this
		// mod's own part item, which places this mod's own part. That is
		// structural and stronger than any flag.
		//
		// `not-blueprintable` is DELIBERATELY ABSENT, and the reason is the one
		// case this whole file exists for. A migrating player's blueprint book is
		// full of `balancer-part`, and those books keep working: a ghost of this
		// prototype asks for a `bbb-balancer-part` item through `placeable_by`, a
		// robot builds the stub, and the guest swaps it for a real part inside
		// the build event (guest/go/legacy.go). Refusing the blueprint would
		// break the one path that makes an old book useful, in exchange for
		// stopping a capture of a prototype that exists for at most one load.
		//
		// `not-upgradable` is present because there is no upgrade path onto or
		// off this prototype and there must not be one; the guest's conversion is
		// the only route out.
		f("flags", strs("placeable-neutral", "player-creation", "not-upgradable")),
		f("hidden", yes),
		f("hidden_in_factoriopedia", yes),

		// THIS MOD'S OWN LONE-PART PICTURE, cell (0,0) of the variant sheet, so
		// that the moment of conversion is not a visible change. A stub and the
		// part that replaces it draw the same pixels.
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("picture", obj(
			f("filename", str(partVariants)),
			f("priority", str("high")),
			f("width", num(64)),
			f("height", num(64)),
			f("x", num(0)),
			f("y", num(0)),
			f("scale", num(0.5)),
		)),
	)
}

// legacyStubMarker is defined WITH the entity and never on its own, because its
// whole meaning is "the `balancer-part` prototype in this game is the stub
// above". The control guest cannot tell an incumbent's prototype from ours by
// looking at `balancer-part` itself, and it must not convert some unknown fifth
// mod's entities; a point lookup of this name against `prototypes.entity`
// answers that in two host calls and no allocation. See guest/go/legacy.go,
// `legacyStubPresent`.
//
// Nothing ever places one. It is a prototype that exists to be looked up, which
// is why it takes lookupOnlyFlags() rather than the placeable list -- see
// hidden.go, where the other marker of this shape lives and where the 2.0 loader
// found out what happens to a placeable simple-entity with no icon.
//
//go:noinline
func legacyStubMarker() fkdata.V {
	return obj(
		f("type", str("simple-entity")),
		f("name", str(legacyMarker)),
		f("hidden", yes),
		f("hidden_in_factoriopedia", yes),
		f("flags", lookupOnlyFlags()),
		f("max_health", num(1)),
		f("collision_mask", obj(f("layers", obj()))),
		f("collision_box", box(0, 0, 0, 0)),
		f("selectable_in_game", no),
		// The graphical client validates every sprite path at load and headless
		// does not, which is how a stale filename here once survived every suite
		// and then refused to load in the real game. The engine's own empty
		// sprite has no file dependency of ours at all.
		f("picture", emptySprite()),
	)
}

// legacyStubItem is tested separately from the entity because the two can in
// principle come apart.
//
// A stack of the incumbent's parts in a chest, in a player's inventory, in a
// logistic request or on a belt is deleted with its prototype exactly as an
// entity is, and there is no runtime pass that could get them back -- so the
// item has to survive the same load. It does not need renaming and is
// deliberately not renamed: `place_result` points at this mod's part, so a
// legacy stack simply places this mod's balancers. Walking every inventory in
// the game to rewrite stacks would be a scan of the whole world for a cosmetic
// difference in what the stack is called.
//
// The stack size is the incumbent's 50, so a full stack stays a full stack.
//
//go:noinline
func legacyStubItem() fkdata.V {
	return obj(
		f("type", str("item")),
		f("name", str(legacyEntity)),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),
		f("subgroup", str("belt")),
		f("order", str("c[splitter]-y[bbb-balancer]-z[legacy]")),
		f("place_result", str("bbb-balancer-part")),
		f("stack_size", num(50)),
		f("hidden", yes),
		f("hidden_in_factoriopedia", yes),
	)
}
