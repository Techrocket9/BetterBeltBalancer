# Design — the compiled-network balancer

Established 2026-07-31 from three research passes: the forum record (incl.
[t=100141](https://forums.factorio.com/viewtopic.php?t=100141)), the belt-balancer-2
2.0.9 source, and the local 2.0.77 `runtime-api.json`/`prototype-api.json`. Facts below
marked **verified** were checked against local API JSON or the mod source; everything
else is design intent until a spike confirms it.

---

## Why the incumbent loses

belt-balancer-2 models each balancer cluster as a `simple-entity-with-force` grid and
moves every item through Lua: pull off input `LuaTransportLine`s into a `storage`
FIFO, round-robin `insert_at_back` onto outputs, on `on_nth_tick(gcd(0.25/belt_speed))`
— which degrades to **every tick** whenever the divisor is non-integral, and express
belts (0.25/0.09375 = 2.67) make it non-integral. Cost is ~13–14 Lua↔C++ boundary
crossings per item moved plus an I+O API-call floor per activation even when idle;
community-measured ~2.7 µs/part/tick, 8,000 parts = 16 ms/tick. therax's verdict
stands: per-item Lua movement is unfixable. Rseding91: the API calls run the same C++
logic as belt flow — the cost IS the boundary.

Additional incumbent sins to avoid: cached `LuaTransportLine`s crash when belts vanish
via unexpected paths (the dominant crash class across all forks); items vanish into an
invisible Lua buffer (stats/circuits can't see them, head-of-line blocking); script
`insert_at_back` cannot produce compressed belts, so it also de-optimizes downstream
belt segments.

## The architecture

**Compile, don't interpret.** A balancer cluster is compiled — at build/mine/rotate
time, never per tick — into a network of *real* splitter entities on a hidden surface,
stitched to the visible tiles by `linked-belt` pairs. All steady-state item movement is
engine C++ (belts sleep when stalled, compressed segments update ~O(1), splitters are
native). Steady-state script cost: **zero**. This is a genuine leapfrog: no mod on the
portal does it (verified by the research pass).

Building blocks, all shipped hidden in base 2.0 (**verified** in
`data/base/prototypes/entity/transport-belts.lua`):

- `linked-belt` — belt-connectable that teleports items to its paired end,
  **cross-surface capable**, native. Runtime: `connect_linked_belts`,
  `disconnect_linked_belts`, `linked_belt_neighbour`, `linked_belt_type`.
- `lane-splitter` — 1x1 splitter over a single belt's two lanes (new in 2.0); the
  primitive for lane-accurate balance.
- `loader-1x1`, `linked-container`, `proxy-container` — the escape-hatch kit (see edge
  problem below).

**The visible entity** stays a 1x1 auto-joining part (the signature UX of the
incumbent): belt orientation alone decides input vs output; clusters merge on adjacent
placement and split on removal (flood fill — iterative, integer-keyed; the incumbent's
is recursive and float-keyed, both bugs).

**The network shape**: for a cluster with N input and M output lanes, build a
Beneš/butterfly-style balancer over P = next-pow2(max(N,M)) ports, unused ports closed
with loopbacks — ~P·log₂P/2 splitters; a 64-lane monster is ~192 hidden splitters,
negligible. Hidden network uses a max-speed custom belt tier so it never bottlenecks
(or 2× width overprovision). Lane fidelity via a `lane-splitter` stage at the edges.
Balance semantics equal a physical splitter balancer under every load condition —
which script approaches only approximate.

**What happens to the items in flight when a network comes down.** A hidden network
is real engine state carrying real items, and a recompile destroys it. The v1 policy
was: drain every transport line in the slot and `spill_item_stack` the total beside the
cluster. Conservation held — nothing was ever lost, and the `edge` suite measured 0
items lost over 200 teardowns of a network deliberately kept full — but the PLACEMENT
was wrong, and interactive play found it before a suite did: **adding one output belt to
a running balancer emptied its hidden network onto the ground.**

**The policy since 2026-08-02 (`guest/go/carry.go`) is: a recompile reinserts, a removal
spills.** A recompile is not a removal — the cluster is still standing and the network
that goes up a moment later is empty and has room — so the drain carries. Items come out
into a pool; the flush tears down and builds; each network built in that flush takes the
pool of the network(s) it succeeds, matched on surface, force and overlapping bounding
box, which is one predicate covering a plain recompile, a merge (the survivor takes both
halves) and a split (both successors take even shares). They go back in **in plan order,
interior lines before linked belts**, at the belt's own 0.25-tile item pitch via
`insert_at` — plan order runs input side to output side, so every stage still in front
of a reinserted item rebalances it, and the observable balance is unchanged.
`insert_at_back` cannot do this: the back of a line is one position, so it takes one item
per line per tick — the same property the incumbent's per-tick mover suffers from, met
from the other side. What the reinsertion cannot take (the new network is materially
smaller) falls back to the spill, and so does everything a true removal drains: a cluster
dissolved, a surface deleted, a network forgotten. Ground-and-belt return is the right
answer for a machine that has been mined and the wrong one for a machine that is still
there.

**The hidden surface**: one global `game.create_surface` with bounded width/height,
lab tiles, `no_enemies_mode`, chunks force-generated, allocated in a grid of network
slots. Hidden entities carry `not-blueprintable, not-deconstructable,
not-selectable-in-game, not-on-map, no-copy-paste, hidden` so blueprints/undo/clone can
never capture them; the network is always **recompiled from visible state**, never
trusted from cloned/undone state.

## The one hard problem — edge interfaces — RESOLVED by spike S1

**Two-to-four linked-belts share one tile with zero per-tick script**, via the one
prototype shape the engine permits: collision mask layers exactly equal to the
belt-connectable type default plus `not_colliding_with_itself = true` (any deviation
from the default layer set fails prototype validation; the default set skips
validation and the flag is honoured at runtime). Belt-connection resolution is
deterministic and order-independent; at most one linked belt per tile side, never two
same-direction inputs on a tile. Full findings, gotchas, and measured numbers:
[`agents/spike-s1.md`](agents/spike-s1.md) — read it before writing the compiler.

Fallback if a future Factorio patch closes the loophole: inserter-assisted second
interface (hidden instant inserters + `proxy-container`, miniloader-1.x technique).

S1 also validated the network end-to-end: exact balance under saturation, starvation,
and blocked outputs; lane fidelity needs the `lane-splitter` edge stage (vanilla
splitters are lane-preserving); stacked items native (needs
`force.belt_stack_size_bonus`); measured **1.47 µs/tick per running 4×4 rig, 0.15 µs
stalled** — ≳11–15× the incumbent before tuning.

Known-bad design, do not revisit: loaders into one shared chest. Fair only under
surplus; under starvation one output drained >9,000 items while peers got ~80
(Techrocket9's measurement in t=100141). A chest also erases lane identity.

## Where FkLua fits (goal 1)

The mod's brain — cluster topology (merge/split flood fill), belt-edge classification,
Beneš network construction, hidden-surface slot allocation, entity-diff planning
(minimal create/destroy on recompile) — is a Go guest compiled by FkLua. This is
integer/graph code, FkLua's sweet spot, and it runs at event time so guest-side speed
is ample. The Lua control shim stays as thin as `fklua mod` makes it.

Guest constraints inherited from FkLua (see `../FkLua/CLAUDE.md`): keep the guest heap
small (it lives in `storage` and every save/join carries it — prefer `--persist=packed`
or recompute-from-world on load), determinism absolute, one call must finish in a tick.
Recompiles are incremental per cluster, so worst case is bounded by cluster size, not
map size.

## Space Age / 2.0 obligations

- Stacked belts: native splitters pass stacks untouched — free with architecture A.
- Space platforms: visible parts on platform surface, network on the global hidden
  surface; linked-belts are cross-surface (**verified** in prototype doc). Handle
  `on_space_platform_built_entity`/mined, platform deletion (`on_pre_surface_deleted`).
- 2.0 lifecycle events: `on_undo_applied`/`on_redo_applied`,
  `on_player_flipped_entity`, `on_entity_cloned`/`on_area_cloned`,
  `script_raised_*`, `on_object_destroyed` for belts destroyed without events.
- Quality: parts have quality tiers like any entity; network is quality-agnostic.

## Feature bar (from the forum record)

Must match the incumbent: 1x1 auto-joining parts, orientation-driven I/O, lane-level
balance, belts/undergrounds/splitters as I/O, no power, tier variants. Known accepted
limitation: a belt curving away at the edge is not an output. Wishlist the incumbent
never delivered (candidates for later): meaningful tier differences, filters/priority,
loader interop. Graphics: adaptive unified sprite over the merged shape (incumbent has
a single repeated tile — no prior art to reuse). **Delivered at M5**: 47-variant blob
tiling, unified at any shape, art computed rather than drawn and replaceable a PNG at a
time (`CLAUDE.md`, "M5 is done").

**And the bar the forum record does not state, because no incumbent had to clear
it: a player's EXISTING save.** Every fork of the incumbent defines the same
`simple-entity-with-force` called `balancer-part` with the same I/O model, so a
save full of them already says what each balancer's ports are — which is what
makes adoption possible rather than merely desirable. Delivered 2026-08-16:
uninstall the incumbent and every part it left standing becomes one of this
mod's, at load, once per save, with the belts and the item stacks untouched and
the technology granted. Nothing happens while the incumbent is installed, and
nothing happens to a `balancer-part` some OTHER mod owns. `CLAUDE.md`, "Adopting
a Belt Balancer 2 or 3 save"; `guest/go/legacy.go` and
`mod-data/prototypes/legacy.lua` are the two halves.

## Milestones

| # | Scope | Exit criterion |
|---|---|---|
| S1 | **Spikes** (throwaway, scratchpad): dual belt-connectables per tile; linked-belt cross-surface item flow; hand-built hidden 4×4 network end-to-end; measured cost of that rig vs belt-balancer-2 same rig | Edge-interface variant chosen with evidence; hidden-network cost measured |
| M1 | Mod skeleton via `fklua init`; visible part entity; cluster tracking (merge/split) in the Go guest; placeholder graphics | Parts merge/split correctly under bench harness scrutiny |
| M2 | **DONE.** Network compiler v1: butterfly build over next-pow2 ports, hidden surface management, linked-belt stitching, belt-edge classification | **Met**: saturated 4×4/8×8 balance to 0.15% spread at 99.9% of a belt per output; starvation, blocked output, 3→5, 4→1, recompile-under-load and cross-surface all hold. Numbers in `CLAUDE.md` |
| M3 | **DONE.** Lifecycle hardening: undo/redo/clone/blueprint/platforms/rotation/flip/script-destroyed belts; recompile-from-visible-state everywhere | **Met**: twelve kill-test rigs, each damaged a different way, all back to their expected rate at 0.00% spread; clones, surface deletion (including the hidden surface), two forces, ghosts, robots and 600 ticks of churn all hold; nothing persistent is an entity reference, so there is no stale-reference class to have. Numbers and the failure envelope in `CLAUDE.md` |
| M4 | **DONE.** Benchmarks vs belt-balancer-2/-3 across the matrix; idle-cost proof (zero script in steady state) | **Met on ≥10×, not on the ~100× stretch.** Marginal cost of a saturated 4×4: **0.49 µs/tick vs bb2's 21.9 and bb3's 23.1 — 45×** on express, 22× on normal, 65× at 8×8; 200 of them cost 0.64 ms/tick against bb2's 4.92. **`scriptUpdate` is the no-mod control's in every cell** and zero `[BBB]` lines run inside any benchmark window — every compile is in `--create`. Idle: `wholeUpdate` 176.51 µs against the control's 176.58. ~100× is unreachable on whole-tick cost because what remains is the hidden network's own engine cost, not overhead; on mod Lua there is nothing left to measure. One regression: a 17 ms Lua GC pause from the guest heap, worse than either incumbent's idle tail. M4 read that as a `--persist=table` cost; the 2026-08-01 persist re-measurement found `packed` has the same tail (the live memory is a Lua word table in both modes) and flipped the shipped mode to `packed` on save size and load time instead. Numbers, ratios and caveats in [`../bench/baselines/RESULTS.md`](../bench/baselines/RESULTS.md) |
| M5 | **DONE.** Adaptive themed graphics over merged shapes | **Met.** A 47-variant blob tile set on the part prototype's `pictures`, chosen per part through `LuaEntity.graphics_variation` at cluster-shape-change time: connected parts share a continuous plated surface with no border between them, trim runs only along the real outline, and the outline turns around holes and concave corners with a rounded fillet. 16 side-only variants would have left a notch at every inner corner, which is exactly what stops a blob looking fused; the 47 cost nothing extra to generate because the art is a distance field over the local 3x3, not a hand-drawn edge-and-corner kit. Zero per-tick cost, no second entity, one byte per part in the guest heap, and host calls only for the parts whose picture actually moved. I/O arrows on the edges too, alt-mode only, drawn ON the interface entities so the engine destroys them with the network and the guest stores no rendering ids. Verified: the five named shapes in pure Go (`go test ./skin/`) AND end to end in a real Factorio (M1 phases 7-9), no arrow leak after ~100 teardowns and two surface deletions (M3), and `scriptUpdate` still at the control's. Scheme, bitmask and how to re-theme: `CLAUDE.md`, "M5 is done" |

## Benchmark discipline

`bench/` harness (`bench/README.md`) measures scenario × mod × scale on headless
2.0.77 `--benchmark`; outcomes are verified (items moved, outputs near-equal)
before timings are trusted. Baselines and the M4 head-to-head are in
`bench/baselines/`. Two rules that came out of running it and are not optional:
**compare only cells measured in the same session** (drift is 25–35%), and
**verify throughput as a per-window RATE, not a cumulative total** — a
cumulative total also counts whatever is standing in the pipeline, which is how
a correct balancer with a deeper pipeline reads as a slower one.
