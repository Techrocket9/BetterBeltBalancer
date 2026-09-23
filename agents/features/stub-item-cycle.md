# The two-element cycle that froze a game — the stub item's `place_result`

**A mod-portal report, 2026-09-07: Space Exploration plus a shelf of quality-of-life mods on Factorio 2.0.77 with this mod at 0.2.2, and the game freezes permanently -- no crash, no error dialog, nothing in the log.** Three gestures did it and two did not, and the split is the whole diagnosis:

| what the player did | froze |
|---|---|
| enabled the personal roboport | **yes** |
| hovered or added the balancer part in the QUICKBAR, unresearched | **yes** |
| placed one part | **yes** |
| the same save in `/editor` | no |
| a fresh save with the same mod set | no |

**Every one of those is explained by one handler in one neighbour.** HandyHandsRefactored 2.0.12 auto-crafts what a player's quickbar asks for, from `on_nth_tick`, in a branch guarded on `player.character.allow_dispatching_robots` -- the personal roboport -- and on the controller being a character. That is why the editor is exempt (a different controller) and why the roboport toggle is a trigger on its own. The rest of `items` is quickbar slots, the cursor, ghosts inside the roboport's construction area, upgrades and requests, which is why hovering the item and placing a part are the other two triggers, and why a fresh save with nothing built and nothing in the bar never enters the loop.

**What it does, verbatim, `smarts.lua` lines 252-262:**

```lua
for item, data in pairs(items) do
    local entity_prototype = prototypes.entity[item]
    if entity_prototype and entity_prototype.items_to_place_this then
        for _, v in pairs(entity_prototype.items_to_place_this) do
            update_item(v.name, 0)
            if v.name ~= item then items[item] = nil end
        end
    end
end
```

`update_item` INSERTS a new key into `items` when `v.name` is not already there, and `items` is the table being walked. That is a mutation-during-`pairs` which Lua does not promise anything about, and it is fine as long as the item-to-entity-to-items graph has no cycle in it.

**This mod's data stage had one, and it was invisible to every gate in the repository.** `prototypes.entity[e].items_to_place_this` is not a field any data stage WRITES: the engine derives it, from every item whose `place_result` is `e` PLUS the item named by `e`'s own `placeable_by`. So `legacyStubItem`'s `place_result = "bbb-balancer-part"` filed the stub item under our part, while `legacyStubEntity`'s `placeable_by` filed our item under the stub -- two prototypes, each in the other's list, from two lines that never mention each other. As the engine reported them:

    prototypes.entity["bbb-balancer-part"].items_to_place_this = {bbb-balancer-part, balancer-part}
    prototypes.entity["balancer-part"].items_to_place_this     = {bbb-balancer-part}

**Reproduced headlessly with a probe that runs that loop verbatim**, on the reporter's own save under his own 55-mod set: **NO TERMINATION after 100,000 iterations** for a quickbar seeded `{bbb-balancer-part}`, for one seeded `{balancer-part}`, and for five slots with ours among them; the same seeding without our part terminates in **4 iterations**. A native stack sample of the hung process is `LuaGameScript::runNthTickHandler` -> `luaV_execute` -> `LuaEntityPrototype::luaReadItemsToPlaceThis`, at 97-99% of a core, forever.

## The four shapes, measured in the engine before anything was changed

A scratch data mod defined four candidate pairs and the probe ran the same loop over each. `A` is this mod's part and `B` the stub:

| | the stub item places | the stub entity's `placeable_by` | the engine reports | the loop |
|---|---|---|---|---|
| **S0**, what shipped | **our part** | our item | `A=[A,B]  B=[A]` | **NO TERMINATION**, on all three seedings |
| S1 | the stub | our item AND the stub item | `A=[A]  B=[A,B]` | terminates, 1 / 1 / 4 iterations |
| **S2**, taken | **the stub** | our item alone | `A=[A]  B=[A,B]` | terminates, 1 / 1 / 4 iterations |
| S3 | nothing | our item | `A=[A]  B=[A]` | terminates, 1 / 2 / 5 iterations |

**S1 and S2 produce the same graph and S2 is the smaller change**: the engine appends an item whose `place_result` is an entity to that entity's list whether or not `placeable_by` also names it, so S1's second term buys nothing. Both leave `bbb-balancer-part` placed by exactly one item, which is the acyclic property, and both leave a legacy stack building something.

**S3 was rejected because it costs the feature.** An item with no `place_result` is inert: a migrating player's fifty `balancer-part` in a chest become fifty things that cannot be placed, which is most of what "your stacks keep working" was for. S2 keeps them placeable and costs one tick -- the stub they place is queued by `legacyBuilt` and swapped by the next flush, which is the latency every ordinary part placement already has, and the stub draws this mod's own lone-part picture while it stands. The blueprint path is untouched: a ghost still asks for `bbb-balancer-part` through `placeable_by`, a robot still builds the stub, and the same flush still swaps it.

## What now watches it

**`test/check-datastage.py` asserts the graph on both golden arms, and does not hash it.** A hash says nothing moved and cannot say which shape it is holding still. The projection scans every prototype of every type for `place_result == "bbb-balancer-part"` -- any mod's item is entitled to break this, which is the same argument the whole-dump hash makes one level up -- and the assertions are: exactly one item places our part, in EITHER arm; and on `base`, the marker prototype is present and the `balancer-part` item places `balancer-part`. On `incumbent` the marker must be ABSENT, which is the stub branch not firing; there is no equivalent signal for the stub ITEM and none is needed, because a second `item` of one name is a duplicate-name load failure and that arm would die on `--dump-data exited 1`.

**Red-proven 2026-09-07**, the one line reverted and the package rebuilt: the `base` arm fails by name on both new assertions -- ``` `bbb-balancer-part` is placed by ['balancer-part', 'bbb-balancer-part'] and must be placed by [bbb-balancer-part] alone``` and ``` the `balancer-part` item places 'bbb-balancer-part' and must place 'balancer-part' ``` -- with the `incumbent` arm green to the digit, which is the asymmetry the second mod set exists for and the third time it has paid for itself.

**The `mig` suite lost a signal and gained a better one.** `place_result` was that suite's sharpest line -- a legacy stack placed `balancer-part` under the incumbent and `bbb-balancer-part` under us, so the flip was the observation. Under S2 it reads `balancer-part` on both sides of every swap, whoever owns the name, and that is asserted as NOT MOVING rather than deleted: the item of a name places the entity of that name, so a stack that started placing `bbb-balancer-part` again would be the cycle back. What moves instead is the list the ENGINE derives, which `guest/go/obs/mig` now reports per phase through `harness.EntityPlacers`:

    [BBB-MIG] placers phase=t1 legacy=bbb-balancer-part/2 ours=bbb-balancer-part/1

`EXPECT_PLACERS` in `test/assert-mig.py` is one row per leg per phase, and the two shapes it is built out of are `("bbb-balancer-part", 2)` where our stub owns the name and `("balancer-part", 1)` where an incumbent or the stranger does. **The `1` on `ours` in every phase of every leg is the assertion**: a second item placing our part is the defect, and the seven legs put this mod beside four incumbent names, a stranger, both directions of the swap and a plain reload.

## The 2.1 base goldens are STALE and cannot be re-captured here

The 2.0.77 `base` line moved `f4fbcaa603abc93b` -> **`1e1fcf4f56f5ef22`** and was re-captured; `incumbent` is byte-unchanged at `e7001bf98d6c6771`, which is the control that says the stub branch really did take its other arm there. **The 2.1.16 and 2.1.17 `base` rows are now wrong and this machine has no 2.1 binary**, so both carry a `_stale` note naming the change and the command:

    make mod && test/check-datastage.py --capture

They are marked rather than deleted or skipped. `check-datastage.py` prints a `_stale` note ON a failure and never in place of one -- an unrecapturable golden that stopped failing would be an engine nobody is checking at all, which is this repository's own "a check that skips is a check that passed" met in the gate that was written to answer a question no suite can ask. **The two `incumbent` rows are not stale for the STUB** on either engine: another mod owns `balancer-part` there, so the stub item never existed to move. They carry `_stale` notes of their own since the customizers, four moves each against the base rows' five, every one a settings move. They are stale for 0.3.3's OTHER data-stage change as well, `bbb-linked-belt` gaining a `fast_replaceable_group`, which moves BOTH mod sets on any engine and so reaches the `incumbent` rows the stub never did.

**And the prototype list checksum moved with it**, `790230733` -> `3427049257` on the `base` arm, which is a mild surprise: this file records that checksum as blind to field values (measured, when a `stack_size` went 1 -> 42). A `place_result` is not an ordinary field value, it is a cross-reference the engine resolves into an id, and that is the reading this move supports. It is still a smoke test and is still never a proof.

## The wrong turns, measured and ruled out

Four, and each was a real candidate with a real mechanism, killed by a measurement rather than by argument:

| the theory | what killed it |
|---|---|
| **FkLua's dispatcher rotation.** A one-shot `on_tick` that unregisters and re-registers itself rotates the handler list, and two of them armed together pin the cursor at index 1 | Real, and it is not this. It TERMINATES as soon as one of them stops re-arming, so it defeats the pacing rather than hanging. Transcribed verbatim and run under `../FkLua/bin/lua52f`: two one-shots owing three steps each drained in **6 calls in one dispatch**, one always re-arming beside one owing three made **7 calls** and returned. Filed as [`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 31 |
| **the hidden surface**, which the 0.3.2 round had just touched | The freeze reproduces with no balancer built and no network compiled, on gestures that never reach a surface |
| **the belt speed at the 0.25 floor under Space Exploration**, which loads a great many belt families and whose derivation runs at `data-final-fixes` | SE's deep-space belts top out at **0.1875**, below the floor, so `deriveHiddenSpeed` leaves all four hidden prototypes where they were and the arm never executes |
| **a belt-connectable sharing the part's collision box under Squeak Through**, which rewrites collision masks wholesale | Nothing of any mod's shares that tile in the dump. Checked in the same pass: the technology unit resolves to the plain 20-pack fallback under this mod set, so the tech ladder is not in it either |

**None of the five gates in this repository could have seen it, and that is the finding rather than an excuse.** All fourteen suites are about the RUNTIME and the defect is a prototype relationship the engine derives at load; the dump gate hashed a dump that CONTAINED the cycle and called it the golden; and nothing anywhere walks a prototype graph, because nothing in this mod does. What closes it is an assertion about a SHAPE, on a gate that already had the dump in hand.
