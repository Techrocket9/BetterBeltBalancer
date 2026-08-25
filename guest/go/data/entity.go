package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/skin"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// The icon every player-facing prototype of this mod uses, and the sheet the
// part draws from. Named once because five prototypes across three files
// reference them and a stale path is a defect only the GRAPHICAL client sees --
// headless Factorio never opens a sprite file, which is how one survived every
// suite and then refused to load in a real game. `test/check-sprites.py` walks
// the packaged mod for exactly this class.
const (
	partIcon     = "__better-belt-balancer__/graphics/icons/balancer-part.png"
	partVariants = "__better-belt-balancer__/graphics/entity/balancer-part-variants.png"
)

// entity defines the visible part: a 1x1 tile that clicks together with its
// neighbours.
//
// `simple-entity-with-force` mirrors the incumbent, and the choice is not
// cosmetic. The part must be force-owned (clusters are per force), must not be
// an active entity (it costs nothing per tick, which is the whole point of the
// architecture), and must occupy a tile the way a belt does so that a player
// cannot put a belt through one.
//
// THE COLLISION MASK IS THE LOAD-BEARING LINE. Verified against the prototype
// API and base's own prototypes/collision-layers.lua: all five layer names
// exist.
//
//	floor, meltable, object, water_tile   what a normal building collides with
//	transport_belt                        what makes a belt refuse to overlap it
//
// Without `transport_belt` a player can lay a belt straight through the
// balancer, and the belt-orientation-decides-I/O rule stops making sense.
//
//go:noinline
func entity() {
	fkdata.Extend(obj(
		f("type", str("simple-entity-with-force")),
		f("name", str("bbb-balancer-part")),
		f("icon", str(partIcon)),
		f("icon_size", num(64)),

		// `get-by-unit-number` is what lets the runtime resolve a part from a
		// stored unit number instead of from a cached LuaEntity. Caching entity
		// references is the dominant crash class in every fork of the incumbent;
		// this flag is the alternative, and it has to be declared at data stage
		// or it is not available later.
		f("flags", strs("placeable-neutral", "player-creation", "get-by-unit-number")),

		f("minable", obj(f("mining_time", num(0.1)), f("result", str("bbb-balancer-part")))),
		f("placeable_by", obj(f("item", str("bbb-balancer-part")), f("count", num(1)))),

		// FAST REPLACE, and it is the one line that makes a part behave like the
		// machine it is. `"transport-belt"` is base's own group for every belt,
		// underground, splitter and lane splitter, so holding a part over a belt
		// replaces it the way holding a splitter over one does: the belt and
		// whatever it was carrying go to the player, the part takes the tile.
		//
		// IT DOES NOT WEAKEN THE COLLISION MASK ABOVE and must not be read as
		// doing so. Fast replace is an exception the engine makes for the entity
		// being REPLACED and for nothing else, so a belt still cannot be laid
		// THROUGH a balancer -- it can only be laid ON one, one tile at a time,
		// which is the reverse gesture and is handled in guest/go/fastreplace.go.
		//
		// The group is symmetric by construction, so this also buys the reverse:
		// a belt held over a part replaces the part. Only a part with NO EDGE
		// INTERFACE on its tile can be replaced that way -- `bbb-linked-belt` is
		// a belt-connectable of its own and blocks the placement, measured --
		// which means the gesture reaches interior parts and is refused on edges.
		f("fast_replaceable_group", str("transport-belt")),
		f("max_health", num(170)),
		f("corpse", str("splitter-remnants")),
		f("resistances", list(obj(f("type", str("fire")), f("percent", num(60))))),

		f("collision_box", box(-0.35, -0.35, 0.35, 0.35)),
		f("selection_box", box(-0.5, -0.5, 0.5, 0.5)),
		f("collision_mask", obj(f("layers", beltLayers()))),

		// M5: THE ADAPTIVE SPRITE. `pictures` is a variation set and the runtime
		// picks one per entity through `LuaEntity.graphics_variation`, which
		// `simple-entity-with-force` inherits from `simple-entity-with-owner`.
		// The guest sets it whenever a cluster's SHAPE changes
		// (guest/go/skin.go), so a balancer of any shape draws as one continuous
		// structure with trim only along its real outline. Zero per-tick cost, no
		// extra entities, no rendering objects, nothing stored per part but one
		// byte.
		//
		// THE CELL COUNT COMES FROM guest/go/skin AND IS NOT WRITTEN DOWN TWICE.
		// It is the number of canonical neighbour masks, and the sheet, the
		// generator and the runtime all have to agree about it -- which is
		// exactly the drift a literal 47 in a Lua file could not be checked
		// against. `skin.Count` is proved by that package's own tests to be the
		// count of what `skin.Canon` actually produces, so a re-theme that
		// changed the enumeration would move this prototype with it.
		//
		// Laid out 8 per row, in the order tools/make-graphics.py and
		// guest/go/skin both enumerate: canonical neighbour masks ascending.
		// 64 px at scale 0.5 is exactly one 32 px tile. `pictures` takes priority
		// over `picture` and `animations`, so this is the only art the entity has.
		f("pictures", obj(f("sheet", obj(
			f("filename", str(partVariants)),
			f("priority", str("high")),
			f("width", num(64)),
			f("height", num(64)),
			f("variation_count", num(skin.Count)),
			f("line_length", num(8)),
			f("scale", num(0.5)),
		)))),

		// The engine would otherwise pick a RANDOM cell for a part the moment it
		// is created, which would be the wrong shape for one tick (the guest sets
		// the right one on the next) and would flicker on every blueprint paste.
		// Cell 1 is the lone part -- the correct picture for a part with no
		// neighbours, which is what a freshly placed one usually is.
		f("random_variation_on_create", no),
	))
}

// beltLayers is the type default for a belt-connectable, verified against
// base's own prototypes/collision-layers.lua.
//
// ONE FUNCTION FOR THREE CALLERS, and the third is why it exists: the visible
// part, the legacy stub and the hidden linked belt all name the same five
// layers, and on 2.0 the linked belt's copy is the one that must be EXACTLY the
// type default or the engine moves that prototype from the unvalidated path to
// the validated one and the mod stops loading (hidden.go's header). Three
// hand-written copies of a list that has to be identical is a list that will
// stop being identical.
//
//go:noinline
func beltLayers() fkdata.V {
	return layers("floor", "meltable", "object", "transport_belt", "water_tile")
}
