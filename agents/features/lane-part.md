# The lane-keeping part

A second balancer part, `bbb-lane-balancer-part`, whose balancer never mixes a belt's two lanes: the left lane of every input is balanced over the left lane of every output, the right over the right. Asked for on the mod portal (thread "No mixing-option", Gamer433, 2026-11-10): a belt carrying copper cable on one lane and steel on the other, split where a splitter does not fit, should arrive at every output with the same item on the same side. Built 2026-11-10.

## The decision: a second part, not a mode

Three shapes were weighed (map setting, per-part flag, separate item). The mod's author chose the separate item, priced at twice the ordinary part, and refusing to interact with the ordinary part at all. What decided it:

- **It is chosen before placement**, which is what the request asked for (a mode flipped after placing on a live belt has already mixed it). A per-part flag cannot be: ALT + P acts on the part under the cursor.
- **Lane-keeping is a property of the whole network**, not of one part. A flag per part would need a rule for a cluster with some parts flagged; two kinds that never join need none.
- **The variation byte is already full of meaning.** Priority doubled the sheet 47 -> 94. An independent flag would double it again; a second prototype has its own sheet with the same 94 cells.
- **No migration surface.** Nothing older than this release has the part, so there is no Belt Balancer 2/3 adoption arm and no pre-existing save to keep a reading for.

## What makes it lane-keeping: one column

`plan.KeepLanes` turns every `ProtoLaneSplitter` op into a `ProtoBelt` on the same tile, same direction, and changes nothing else. The lane-splitter column at `colLaneSplit` is the ONLY place a network mixes lanes: S1 measured that vanilla splitters keep lanes ("without it a left-lane-only feed parks 4/0 on every output", `agents/spike-s1.md`), and belts and linked belts keep them by construction. A belt rather than a gap keeps the layout identical (recv at column 0, first stage at column 2), so a lane-keeping network has the ordinary one's shape, ports, slot footprint and entity count. `compile` calls it right after `plan.Build` when the root's group carries `laneBit`. It applies to the priority network too: its head and every tier line lay the same column.

**Whether the priority network's concentrator splitters keep lanes had never been measured** (the comment in `subAt` said so). The `lane` suite's `lprio` rig measured it on 2.1.24: a lane-keeping 1->2 with a priority output delivers iron only on the left and copper only on the right, 837/837 to the priority output and 0 to its sibling.

**The ordinary network does not just mix a split feed, it swaps it.** Measured by the `o12` control on the same feed (iron left, copper right, saturated): both outputs carry copper on the left and iron on the right, in every window. Worth knowing before anyone writes "mixes" as if it meant "averages".

## Keeping the two kinds apart: `laneBit`

The registry's `pforce` word carries `laneBit` (bit 31) for a lane-keeping part (`cluster.go`). Forces are capped at 64, so the bit is never a real force index. Every comparison that was "same force" is now "same force and same kind" without a line of its own: `AddPart`'s merge, `removePart`'s seeds, `rebuildComponent`, `compile`'s flood, `skin`'s neighbour mask (so the sprites do not fuse across kinds), the settings paste (`onSettingsPasted` refuses a different group), the carry pool's region claim, and the miner's-pocket claim. **It fails closed**: a comparison nobody extended still keeps the kinds apart, and a group value that reaches the host unmasked is a force index the engine refuses, loudly.

What crosses to the host is `realForce(g)`, at five places: `createArgs`, `backArgs`, the force filter in `classifyEdges` and in `sedge.go`'s `edgesOnTile`, and `forceOfCluster`'s compare. What finds a part by name uses `partNameOf(g)`: `restyle`, `forceOfCluster`, `revertOne`, `reapFastReplaced`. The two world scans (`reconcileArea`'s box scan and `registerPartsOn`) scan both names and OR the bit in for the lane part. A force merge compares on `realForce`, carries the bit across in the remap, and remaps the carry pools for both groups.

The keybind and the paste ask `isPartEntity`, which is two `name_is` calls.

**A part of one kind built on a tile registered as the other** removes the registered one first (`main.go`, the part branch). The two prototypes share `fast_replaceable_group = "transport-belt"`, so one can replace the other. A player's gesture raises the mine before the build and never reaches this; a script's `fast_replace` raises nothing for the replaced entity (`fastreplace.go`), and without this the new part would be refused as a duplicate and the tile left registered as the old kind. `reconcileArea` likewise treats a tile whose world group differs from its registry group as gone and re-added.

## The carry keeps lanes

A recompile drains the network into a pool and reinserts it (`carry.go`). For a lane-keeping network every group carries a `lane` (1 left, 2 right): `drain` sets `tallyLane` per transport line from `laneOf(i)`, and `takeCarry` reinserts each lane's groups through a `lineWalk` restricted to that lane's lines. An ordinary pool has lane 0 throughout and takes one pass, exactly as before.

`laneOf` resolves the left-line defines BY NAME (`left_line`, `secondary_left_line`, `left_underground_line`, `left_split_line`, `secondary_left_split_line`); anything else is a right line. No number is written down.

**Red proof, 2026-11-10, 2.1.24**: with `drain`'s lane tally forced off, the window after the recompile shows copper on the left lane of `l12-west` (3 sightings) and `ljam-west` (7), and iron on the right (1 and 3). With it on, zero in every window.

The priority toggle's capacity guard (`prioFitsWhatIsStanding`) counts a lane-keeping network as twice its fuller lane (`heldItems`), because a successor with room R takes back R/2 per lane.

## The orphan sweep looks for linked belts now

`sweepOrphanSlots` found occupied hidden slots by their lane splitters, one per used input port. A lane-keeping network has none, so an orphaned one would never have been swept, and a slot above the highest the scan saw would have been handed out again over the entities still in it. A claimed lane network was never at risk (claimed slots are skipped). The scan reads linked belts now: every network has them, about three times as many entities, on a path that runs once per rebuild; the dedup checks the last slot first. The `lane` suite's `lorph` rig is the orphan: a lane-keeping 1->1 destroyed silently after `on_init` compiled it, and the upgrade's rebuild sweeps its 3 entities (recv, belt, send).

## Data stage and art

- `data/entity.go`: `part(name, icon, sheet)` emits both prototypes; the lane part uses `graphics/entity/lane-balancer-part-variants.png`.
- `tune/plan.go`: the item and the recipe, `IngredientOf(part, 2)`. The ingredient is the ordinary part, so twice the price holds under every preset and every typed recipe. The technology unlocks both recipes.
- Locale: entity, item and recipe names, two descriptions (196 and 160 bytes, under the 200-byte element ceiling).
- The green sheet and icon are a hue rotation of the committed blue ones, `make graphics-lane` (`tools/make-graphics.py --lane`): pixels of hue 168-205 degrees moved -68 degrees with a smooth band edge, everything warm, grey or slate untouched (58,161 of 393,216 sheet pixels). See `adaptive-graphics.md`.
- **The dump goldens move**: on 2.1.24 four new prototypes (entity, item, recipe, and the quality mod's derived recycling recipe) and one new unlock each on `bbb-balancer` and `recycling`; on 2.0.77 the first three and the `bbb-balancer` unlock. `mod_settings_sha256` does not move (no setting).

## Not verified

- **The look in a graphical client.** Headless Factorio opens no sprite file. The preview at half scale was looked at; the game has not been.
- **A player's ALT + P, paste and fast-replace gestures between kinds.** Behind the player wall like the rest of priority's doors. The scripted fast replace in both directions is the `lane` suite's `frep` rig. The interactive checklist's section L (`test/interactive/README.md`, staged at x = 92 by `bbb-interactive-setup`) is where a human walks them, with the look.
- **Factorio 2.0 is verified** (2026-11-10, 2.0.77, as a recut worktree): both `make test` arms green with `lane` reading the 2.1 numbers to the item, and the 2.0.77 goldens re-captured. On 2.0 the dump has no recycling recipe for the lane part.
