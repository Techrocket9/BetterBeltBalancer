# The tan streak — a clone of a belt keeps the belt's pictures

**A field report from interactive play: a tan streak between adjacent balancer parts on a 2×2 with one corner missing, visible only while items were flowing.** Seven headless suites were green while it happened, and they always would have been: not one of them can see a pixel.

`bbb-linked-belt` is `util.table.deepcopy` of base's `linked-belt` with the speed raised and the player-facing surface stripped — and "stripped" covered the selection box, the sounds and the item, not the **art**. It kept:

| what it kept | how big | at which layer |
|---|---|---|
| `structure` | 192×192 at scale 0.5 = **three tiles by three** | `object` — the part's own layer |
| `belt_animation_set` | the running belt plus its starting/ending patches, drawn past the tile edge by design | `transport-belt`, `transport-belt-endings` |

The part sprite is 64 px at scale 0.5 — **exactly one tile, opaque in all 47 cells** (decoded and checked pixel by pixel, minimum alpha 255 on every edge of every cell) — at render layer `object`. So an interface is completely covered **on its own tile** and completely uncovered everywhere else, and base's linked belt paints eight neighbouring tiles. On a solid rectangle the neighbours' own sprites hide most of it. On a shape with a **notch** — which is exactly what the field report was — the empty tile is covered by nothing at all.

## What was blanked, and why every replacement is a sprite rather than a `nil`

`guest/go/data/hidden.go`. All four hidden prototypes, because the rule worth having is "nothing the compiler places draws anything", not "nothing the compiler places draws anything where it matters" — which is the qualification this defect came in through. Only `bbb-linked-belt` stands where a player looks; the other three are belt-and-braces on the hidden surface.

| prototype | field | replaced with |
|---|---|---|
| all four | `belt_animation_set` | a set whose `animation_set` is one transparent pixel, `direction_count = 20` |
| `bbb-linked-belt` | `structure` — all six `Sprite4Way` members | `util.empty_sprite()` each |
| `bbb-splitter` | `structure`, `structure_patch`, `frozen_patch` | blank animation / blank animation / `util.empty_sprite()` |
| `bbb-lane-splitter` | `structure`, `structure_patch` | blank animations |

**Validity, against the pinned `prototype-api.json` rather than against a passing run:** `TransportBeltConnectablePrototype::belt_animation_set` is optional (default null) and so is every member of `LinkedBeltStructure` and every splitter structure — *except* `LaneSplitterPrototype::structure`, which is `optional = false` and is therefore **replaced, never dropped**. Inside a belt animation set `animation_set` is itself non-optional, which is why the set is swapped whole instead of emptied field by field.

**Everything is a drawable empty sprite, not an absent one, and that is deliberate.** Both are legal; only one is a shape this repo has already proved. Headless Factorio never opens a sprite file and `test/check-sprites.py` only checks that the paths we name exist, so the **graphical** client is the first thing that would notice a shape the engine dislikes — which is precisely how a stale filename once shipped (commit e42e07d). `util.empty_sprite()` is core's own idiom and is already what the audit marker uses.

**`direction_count = 20` is the one number that is not arbitrary.** A belt animation set is indexed by direction and by patch and its defaults run up to `ending_east_index = 20`; base's own belt sets are `direction_count = 20` for that reason. `__core__/graphics/empty.png` is 64×64, so twenty rows of one pixel fit inside it with room to spare. The `WithCorners` form used by `bbb-belt` adds indices 5..12, which the same twenty cover.

## The items: 2.0.77 offers nothing, and this is the honest residual

**There is no prototype field in 2.0.77 that suppresses the drawing of items on a belt-connectable, and no linked-belt equivalent of `belt_length`.** Checked exhaustively rather than guessed, against `doc-html/prototype-api.json` (application_version 2.0.77, api_version 6):

- `LinkedBeltPrototype` has exactly five properties of its own — `allow_blueprint_connection`, `allow_clone_connection`, `allow_side_loading`, `structure`, `structure_render_layer`. None of them is item-shaped.
- `TransportBeltConnectablePrototype` has six — `animation_speed_coefficient`, `belt_animation_set`, `collision_box`, `flags`, `selection_priority`, `speed`.
- A sweep of **every** property on **every** prototype and type whose name matches `item|render|draw|hide|visib|belt_length` returns, for the belt family, only `draw_circuit_wires` / `draw_copper_wires` (splitter, belt, loader), `structure_render_layer` (linked belt, loader) and **`LoaderPrototype::belt_length`** — which exists on loaders alone. There is no such field on `LinkedBeltPrototype`, so the visible travel is one tile and cannot be shortened.
- Raising `speed` does not help either: a transport line's item DENSITY is fixed by item size, not by speed, so a saturated interface holds the same number of items however fast it runs. (The 0.25 this paragraph was written against is a FLOOR since 0.3.1, not the ceiling it says; the point stands either way, because the drawn overhang is a function of item size and not of speed. `guest/go/data/hidden.go` is the header, `hidden.lua` having been the file before the data stage became a guest.)

**So what a player should still expect:** items on a visible interface are drawn at render layer `item`, which is *below* `object`, so the part's own sprite hides them on the part's tile. What it cannot hide is the ~0.16-tile **overhang** of an item sprite whose centre has reached the far edge of the interface belt — and for an INPUT edge that far edge points *into* the cluster, so on a shape with a notch the overhang lands in the notch. That is a thin band at the tile boundary, flow-dependent, and it is what is left after the blanking. Nothing in this repo can remove it; it needs a prototype field the engine does not have.

## The structural half, which is the durable one

Blanking is a statement about one prototype. The `edge` suite now asserts the statement that outlives it: **the only thing the compiler ever creates on a surface a player looks at is an edge interface, and an interface stands on a tile of the cluster itself** — under a part's opaque one-tile sprite. `probe_placement` enumerates every entity of all four hidden prototypes on the visible surface and requires a registered part on that exact tile, and it is sampled five times, including once with every rig in the save saturated:

| sample | ours | on a part tile | off one |
|---|--:|--:|--:|
| `init` | 125 | 125 | **0** |
| `post-merge` | 125 | 125 | **0** |
| `post-add-out` | 130 | 130 | **0** |
| `flowing` | 131 | 131 | **0** |
| `final` | 114 | 114 | **0** |

(Re-measured 2026-08-04 after the `lim` rig joined the suite; the counts have now moved twice — 56/56/61/61/54 originally, then 60/60/65/66/49, now these, because each new rig adds its own edge interfaces. The assertion column has never moved: **0 off a part tile, every sample, every recording.**)

All four prototype names are probed rather than just the linked belt, so a hidden splitter appearing out there fails rather than going uncounted; and a sample of fewer than twenty fails, so a probe that found nothing cannot pass.

The rig it is really about is `ntch`: parts at (0,b), (1,b) and (0,b+1), **the corner at (1,b+1) deliberately empty**, two inputs from the west, one output east off the top-right part and one SOUTH off the bottom-left one — which is what keeps the notch a notch instead of filling it with an output belt. It is saturated from `on_init` to the last tick and delivers **376 376** over the same 500-tick window the `aout` balance check uses, which is what makes the `flowing` sample a measurement of a balancer that was actually running.
