# Spike S1 results — edge interfaces and the hidden network (2026-07-31)

Empirical, on Factorio 2.0.77, headless. Spike artifacts are scratchpad-only (not committed); this file is the durable record. Everything below was observed, not inferred. **The architecture is viable; the edge-interface problem has a clean native solution; measured ~15×+ vs the incumbent, before tuning.**

## The collision-mask door (Q1)

Two (or four) linked-belts share one tile with zero per-tick script, but only through exactly one prototype shape:

```lua
collision_mask = { layers = { floor=true, meltable=true, object=true,
                              transport_belt=true, water_tile=true },   -- the type DEFAULT
                   not_colliding_with_itself = true }
```

The engine validates any belt-connectable whose `layers` differ from the type default (demanding collision with `transport-belt`'s mask and with itself — every deviation fails at load, 14 masks probed), **but skips validation entirely when `layers` equals the default, while still honouring `not_colliding_with_itself` at runtime.** This is a loophole, not an API: pin the game version in testing and keep the inserter+ proxy-container fallback documented in case a patch tightens it.

Facts the compiler must respect:

- Belt-connection resolution with multiple linked-belts on a tile is unambiguous, deterministic, creation-order-independent, and survives save/load: an inbound visible belt connects only to the input-type linked belt, outbound only to the output-type. Four linked-belts (3 in + 1 out) on one tile all ran at full rate.
- **Never two same-direction inputs on one tile** — one silently gets everything, the other nothing. At most one linked belt per tile side.
- `create_entity` of a colliding belt-connectable returns **nil silently** — always nil-check. Conversely `create_entity` ignores ordinary building collision (use `can_place_entity` for player-facing checks). A vanilla belt cannot be placed on a tile holding a hidden linked belt, and vice versa.
- The visible part keeps an ordinary building mask; hidden belts stack on/around it fine in either creation order.

## Linked-belt mechanics (Q2, Q5)

- Cross-surface item flow is native, full-rate, and survives save/load. Hops cost **zero throughput** (measured identical through 0/1/4 hops) — only pipeline latency (~32 items of fill per hop at express).
- `linked_belt_type` is changeable only while disconnected, and **changing it flips `direction` 180°** — set direction after the type flip.
- `connect_linked_belts` requires opposite types, not same force. Create both ends independently, connect after; no pre-declaration.
- Destroying one end is safe (survivor valid, neighbour nil, reconnectable); items on the destroyed end are lost — drain or accept. `delete_surface` with live pairs doesn't crash. Treat `linked_belt_neighbour` as the source of truth, never a cached LuaEntity.
- Chunk generation on the hidden surface is optional: entities on ungenerated chunks work at full rate (tile stays out-of-map). Real allocation-time saving; re-verify before shipping.

## The network balances exactly (Q3)

Hand-built 4×4 Beneš (two express-splitter columns, row-swap wired with a linked-belt jumper pair — cleaner than underground crossings, equally native):

| condition | outputs |
|---|---|
| saturated | 3780/3780/3780/3780 (exact) |
| **starved** (1 input at yellow rate) | 314/314/314/316 — the case that kills chest designs |
| one output blocked | remaining three exact; inputs drained evenly |
| left-lane-only feed | balanced across outputs after a `lane-splitter` edge stage |

- Vanilla splitters are lane-**preserving**, not lane-balancing: without the `lane-splitter` stage, a left-lane-only feed parks 4/0 on every output; with it, 4/4. The edge lane-splitter stage is confirmed necessary for the lane-balance feature bar.
- Stacked items (Space Age) pass through linked belts, hops, and splitters untouched at 4× throughput — but stacking requires `force.belt_stack_size_bonus` in addition to loader prototype fields; prototype alone silently does nothing.
- Undergrounds and splitters connect natively to the edge tiles — I/O feature bar satisfied by the engine.
- Byte-identical item counts across benchmark runs → deterministic.
- Splitters need no power; `lane-splitter` is 1×1 and script-placeable.

## Indicative cost (Q4)

50 rigs, 3600 ticks, isolated user dir (see below), <1% spread:

| state | per 4×4 rig per tick |
|---|---|
| running | **1.47 µs** (incl. visible feed belts/loaders/chests — 154 entities) |
| stalled | **0.15 µs** — belt sleeping recovers 90% |

Incumbent comparison (not like-for-like; bb2 not in this run): its 8 edge parts alone ≈ 21.6 µs/tick at the community 2.7 µs/part figure; our bench-measured bb2 4×4 express figure is 16.9 µs. **≳11–15× indicated, and our idle floor beats bb2's idle floor (0.15 vs 1.40 µs) by ~9×.**

## Operational

Concurrent Factorio instances fight over the user-dir `.lock`; run with `-c` pointing `write-data` at a private dir. The bench harness should adopt this if agents ever race.
