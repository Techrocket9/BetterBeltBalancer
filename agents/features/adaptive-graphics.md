# M5 is done

**Adaptive themed graphics over merged shapes, and the exit criterion is met: a balancer of any shape reads as one structure.** Connected parts share a continuous plated surface with no border between them, trim appears only along the real outline, and the outline turns cleanly around holes and concave corners. It costs one byte per part in the guest heap, two host calls per part whose picture actually changed, and nothing at all per tick.

## The mechanism, in one paragraph

`bbb-balancer-part` is a `simple-entity-with-force`, which inherits `pictures` (a variation set) and the runtime attribute `LuaEntity.graphics_variation` from `simple-entity-with-owner`. The prototype ships **47 pictures** in one 512×384 sheet; the guest sets a part's variation whenever its cluster's SHAPE changes. There is no second entity, no rendering object per part, no `on_tick`, and nothing in `storage` but a byte. **Verified empirically before anything was built on it** (a five-minute headless probe): the attribute is accepted on this prototype, reads back, survives a fresh handle, is **1-based**, and a value above the count silently WRAPS modulo it — 48 draws cell 1 and 255 draws cell 20 — which is why `skin_test.go` asserts that every mask lands in 1..47 rather than trusting an error to appear.

## The bitmask, and why 47 rather than 16

```
bits: N=1 E=2 S=4 W=8   NE=16 SE=32 SW=64 NW=128
```

A bit is set when that neighbour tile holds a part **of the same force** — two forces' parts touching are two balancers and must not fuse into one picture, the same rule the flood fill and the compiler already obey.

The four side bits alone give 16 pictures and already put the trim only on the outside. What they cannot express is the INSIDE of a corner. A part whose north and east neighbours are both present while the north-east diagonal is empty owns the point where those two neighbours' trims meet, and if it draws nothing there the outline dies in a notch. **That is the entire visual difference between "tiles that agree about their borders" and "one machine".** A diagonal bit is meaningful only when both sides touching it are set — if a side is missing, that side's own trim already draws the corner — and canonicalising on that rule leaves the classic **47** of the 256:

| sides | configs | × meaningful corners | |
|---|--:|--:|--:|
| none | 1 | 1 | 1 |
| one | 4 | 1 | 4 |
| two, opposite | 2 | 1 | 2 |
| two, adjacent | 4 | 2 | 8 |
| three | 4 | 4 | 16 |
| four | 1 | 16 | 16 |
| | | | **47** |

`guest/go/skin` and `tools/make-graphics.py` both enumerate masks 0..255 ascending, keep the canonical ones and number them from 1. **Neither stores a table**; the rule is six lines on each side and three anchors are asserted in both, so a change made on one side fails on both. `Canon` is proved idempotent and total over all 256 masks by `go test ./skin/`.

## How the art is generated, and what an artist would replace

`tools/make-graphics.py` computes every pixel (nothing is copied from the base game or any mod) and writes three committed PNGs:

| file | what |
|---|---|
| `graphics/entity/balancer-part-variants.png` | 512×384, 8×6 cells of 64 px, 47 used |
| `graphics/icons/balancer-part.png` | 64×64, the lone part with its conduits lit |
| `graphics/entity/io-arrows.png` | 256×32, eight 32 px arrow cells |

**Nothing in it draws "an edge piece" or "a corner piece", and that is what made 47 affordable rather than three times the work of 16.** The nine cells of the local neighbourhood become a signed distance field — for each pixel, the distance to the nearest EMPTY cell — and the trim is a band at a fixed distance from that boundary. All 47 cells fall out of the same six lines. Concave corners come out rounded, because the distance to an empty cell's corner is radial, and that rounding IS the fillet that makes a blob look fused; convex corners come out sharp, which is what an armour plate should look like.

The theme on top of that: a dark slate plated body with a vertical gradient, brushed-metal hash noise, plate seams and rivets on a 16/32 px lattice **offset so they line up across a tile boundary** (a 2×2 block is one continuous plated surface, not four squares); a steel-blue edge trim with fake lighting that brightens north-facing edges and darkens south-facing ones; and a lit core in every part with a conduit running to each CONNECTED side, so two neighbours show one unbroken lit vein between their cores. Re-theming is the palette block at the top of the generator plus one `make graphics`.

**That contract was exercised on 2026-08-19** and it held: an artist replaced all three entity PNGs plus a new 288x288 mod logo, and the only thing that had to move on this side was one alignment constant in `sprite.lua` -- which was fixing a defect of ours that the placeholder had carried since M5, not accommodating the new art. See "The I/O arrows" above.

**A future artist replaces the PNGs and keeps the cell order.** The only contract is "cell *i* is the shape whose canonical mask is the *i*-th ascending one, 64×64, laid out 8 per row"; `variant_masks()` prints the list. Nothing in the guest or the prototype needs to change.

## What it costs

- **In the guest heap**: one byte per part (`pvar`, in cluster.go's parallel slices — 29 bytes per node became 30). That matters because the idle GC tail scales with the live heap, which is why it is a byte and not a struct.
- **In host calls**: `restyle` computes every part's mask from the REGISTRY, so deciding that a 200-part balancer's pictures are all still correct costs **zero**. Only the parts whose picture changed are touched, one `find_entity` and one `graphics_variation =` each. Growing a 4-part line by one tile moves exactly two pictures, and the M1 suite asserts that number (`set=2 of parts=5`) rather than trusting it.
- **Per tick**: nothing. There is no `on_tick` handler and there must never be one.
- **In the mod**: the zip went 139,762 → **185,291 B**, of which ~40 KB is the three PNGs and ~5 KB is the guest. `fk_module.lua` grew 900,786 → 953,523 B. **73 KB of that was one `panic`**: a runtime assertion in `skin`'s package initialiser linked TinyGo's whole panic-and-print machinery into a guest that has no other use for it. It is a zero-length array declaration now, and the counting is proved by the test instead. Worth remembering — a `panic` in guest code is not free the way it is in a normal Go program. (The shipped numbers are **191,677 B** zip / **1,035,941 B** of `fk_module.lua` now; the heap diet added the difference and "The heap diet" says why that was the right way round.)
- **In FkLua**: nothing. No new gap, no workaround. `graphics_variation`, `find_entity`, `rendering.draw_sprite` and the `pictures` variation set were all reachable through the generated bindings as they stood (3,878 members bound; this mod ships 28 of them, up from 25).

## When a part changes picture, and why the fingerprint is not enough

The compiler's fingerprint is a statement about the belts AROUND a cluster and says nothing about its SHAPE, so a sprite update cannot ride on it. `restyle` is called for every root the flush is about to compile, immediately before `compile`, and decides for itself: it compares the mask of every part against `pvar` and does nothing when they agree. That is the common case — a belt laid beside a balancer queues its cluster, and this looks at it and makes no host call.

`rebuildFromWorld` calls it too, before `inspectNetwork` (which holds the tile buffer `restyle` would otherwise reuse under it). A fresh heap knows nothing about what is drawn on the parts standing in the world, so that path pays two host calls per part — once per mod upgrade — and it is what makes a save built by an older build come back drawing the right shapes.

`random_variation_on_create = false` is on the prototype. Without it the engine picks a RANDOM cell the moment a part is created, which would be the wrong shape for one tick and would flicker across a blueprint paste. Cell 1 is the lone part, which is what a freshly placed part usually is.

## The I/O arrows, and the one property that made them cheap

Every visible interface the compiler places carries one sprite saying which way items cross that edge: a green double chevron pointing inwards on an input edge, an amber one pointing outwards on an output, sitting 0.3 tiles towards the side the belt is on, **alt-mode only** (Factorio's own convention for an informational overlay, and it keeps a big balancer from being covered in arrows).

**The lifetime is the whole design.** A rendering object whose target ENTITY is destroyed is destroyed with it (2.0.77 runtime doc, `ScriptRenderTarget`), and the target here is the visible linked belt that a teardown already sweeps. So the guest stores **no rendering ids at all** — no per-cluster list in the heap, no teardown path to get wrong — and the arrows come down with the network on every recompile, every clone reconcile and every surface deletion, including the ones nobody thought about. A network adopted by `rebuildFromWorld` keeps the arrows the previous session drew, for the same reason. The M3 suite asserts 58 rendering objects against 58 standing interfaces after ~100 teardowns and two surface deletions.

Eight sprite prototypes rather than one drawn with an `orientation` and an offset, because the rotation and the shift bake into the prototype: the guest names a sprite and passes a target and a surface, with no orientation to compute from a `defines.direction` value whose number this mod deliberately never writes down, and no offset table to marshal on every draw. `dirIndex` inverts `dirOf` in four comparisons, for the same reason `plan.Opposite` is a lookup on the installed compass rather than arithmetic.

**The shift is per FAMILY, and 0.3 is the DISTANCE rather than the number.** Both the generated placeholder and the 2026-08-19 artist delivery draw each chevron flush against its TAIL edge instead of centred in its 32 px cell, which puts the glyph's centroid **6.6 px** — 0.104 tiles — behind its tip. Measured, and identical in both sheets to the tenth of a pixel, so it is a property of the convention and not of one delivery. An input points INWARDS, so that bias pushes it further out and ADDS to the shift; an output points OUTWARDS, so the same bias pulls it in and SUBTRACTS. One shift of 0.3 therefore landed the two families at **0.404 and 0.196 tiles** from the tile centre: an output stopped reading as an edge marker and sat on the machine's own hub, and a corner part carrying two outputs collapsed both of them into one illegible blob. `sprite.lua` applies `0.3 -/+ ART_BIAS` per family now and all eight land at exactly **0.300**, checked by evaluating the table under `../FkLua/bin/lua52f`. If the art is ever redrawn centred, set `ART_BIAS` to 0 rather than editing the two distances.

**The defect was ours and it shipped with M5**, which is the part worth keeping: the placeholder had the same centroid bias from the day the arrows were drawn, and the only thing the new art changed was making it visible by being bolder. Nothing headless can see it — the `m3` suite counts 58 rendering objects against 58 interfaces and is satisfied by a sprite that exists, wherever it lands, and no assertion anywhere in this repo is about WHERE a cosmetic overlay sits. Found in interactive play, which is the third defect of that shape this file records after "The tan streak" and "The wake race".

## The zero-script property still holds

Re-measured after M5, n=200 k=4 express idle, against its own in-session control (`bench/run.sh --mod none --scenario control-idle` / `--mod bbb --scenario idle`):

| idle, n=200 k=4 express | control | **bbb** | Δ per balancer |
|---|---:|---:|---:|
| `scriptUpdate` | 1.28 µs | **1.30 µs** | 0.0001 |
| `wholeUpdate` | 161.05 | **154.98** | **below the control** |
| `avg_ms` | 0.1690 | 0.1875 | 0.00009 |
| worst tick | 1.66 ms | 18.58 ms | the GC tail, unchanged *at the time* |

**That last row is now 1.42 ms against 2.26** — see "The heap diet" below, which is the pass that found what the tail actually was. The rest of the table is unchanged and is kept as measured.

**`[BBB]` log lines inside the benchmark window: 0**, counted directly in both `run.log` and `verbose.log`. The create log carries 200 `compiled cluster` lines and 200 `skin cluster` lines — every sprite decision, like every compile, happens in `--create`. Recompile cost is unmoved: a 4×4 teardown-and-rebuild is 5.79 and 5.82 ms across two runs against the 5.90 M4's persist pass measured, and an 8×8 is 11.16 and 11.31 against 11.55. The eight `draw_sprite` calls a 4×4 makes are inside the noise of ~350 host calls.

## The editor's variation picker, which is a quirk and not a defect

Opening a balancer part in the **map editor's** entity dialog shows a sprite/variation picker. That is the standard editor UI for any entity with a `pictures` variation set — trees and rocks show the same control — and it is what M5's whole mechanism buys, so it cannot be turned off: 2.0.77's prototype API has no field that suppresses it (checked exhaustively over `SimpleEntityWithOwnerPrototype` and every `editor`- or `variation`-named property in `prototype-api.json`; the only related field is `random_variation_on_create`, which this mod already sets).

It is benign, with one wrinkle worth knowing: **a variation picked by hand persists until something changes that cluster's SHAPE.** `restyle` compares each part's computed mask against `pvar`, the byte the guest remembers writing, and does nothing when they agree — it does not read the entity's current variation, because that would be a host call per part on every flush to detect something only an editor can do. So a hand-picked cell survives every belt laid nearby and every recompile, and is corrected the moment a part is added or removed (or by any mod upgrade, which resets `pvar` through `rebuildFromWorld`).
