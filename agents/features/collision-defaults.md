# A pack that rewrote the game's belt collision layers -- the mod that would not load

**On Factorio 2.0 this mod refused to load beside Cerys-Moon-of-Fulgora, and the message named nobody but us.** Two portal reports, toynbeeidea's and entitytiger's, each a 120-mod pack; the engine's whole account of it is

    entity prototype "bbb-linked-belt" (linked-belt) collision_mask(Modifications:
    Better Belt Balancer) must collide with entity prototype "bbb-linked-belt"
    (linked-belt) collision_mask(Modifications: Better Belt Balancer).

exit 1, no dump, no `[BBB]` line, nothing to act on. Bisected from the reporter's 120 mods to nine and then to one.

**The mechanism is an equality this mod does not own either side of.** `bbb-linked-belt` is the only prototype here that WRITES a `collision_mask`, and it writes one because on 2.0 it has to carry `not_colliding_with_itself` -- a mask is a whole value, so there is no way to add that sibling without also stating `layers`. Those layers are `beltLayers()`, the five names base gives a belt-connectable. Every other belt-connectable in the game, this file's own `bbb-belt`, `bbb-splitter` and `bbb-lane-splitter` included, states NO mask and inherits `data.raw["utility-constants"].default.default_collision_masks[<type>]` (measured: all three, and base's `express-transport-belt` and `linked-belt` too, come out of `--dump-data` with `collision_mask` null).

**A mod is entitled to rewrite that table, and Cerys 4.24.2 does**, in `prototypes/override-final/entity.lua` at its own `data-final-fixes`: it declares a `collision-layer` of its own and adds it to a deep copy of every default mask carrying `water_tile`, so that nothing can be built on its water. Afterwards every inheriting belt-connectable resolves to six layers and this one still states five, and **the 2.0 self-collision validation -- the check `hidden.go`'s header records 2.0 as SKIPPING while a belt-connectable's layers are exactly the type default -- runs**, which is exactly what `not_colliding_with_itself` cannot pass. **So the door the whole multi-edge architecture stands in is held open by an equality with a table another mod may move, and the literal was only the right answer until somebody moved it.**

**Factorio 2.1 is unaffected and the reason is the same one sentence.** The flag is never emitted there, and that validator demands every belt-connectable collide with itself, which the literal satisfies whatever the defaults say. What the fix buys on 2.1 is narrower and is worth having anyway: the interface collides with whatever a mod decided belts collide with, rather than being placeable on one more mod's water.

## The fix is in two parts and neither works alone

**`followCollisionDefaults`, in `guest/go/data/hidden.go`, called from `fk_data_final_fixes`.** It re-reads `default_collision_masks["linked-belt"].layers` at the last stage there is and writes it onto `bbb-linked-belt` when it differs from the literal. **Only `layers` is written, never the mask**, because `not_colliding_with_itself` is a sibling of it and is `canStack()`'s decision rather than the default's. **Nothing is written when nothing moved**, which is `deriveHiddenSpeed`'s own idiom on the same hook and is here for the same reason: a stock game's prototype table stays byte-identical to the one every number in this repository was measured on, and the goldens say so below.

**And an OPTIONAL DEPENDENCY on `Cerys-Moon-of-Fulgora` in `fklua.toml`**, which is a load order and nothing else: Factorio puts an installed optional dependency ahead of the mod declaring it, and the `?` form is inert in every game without it. Without the entry the two final-fixes ran this mod at 0.823 and Cerys at 0.984; with it, 0.854 and 0.892 on the nine-mod pack and 7.021 and 7.210 on the reporter's 120.

**THE LIMIT IS STATED RATHER THAN PAPERED OVER, and it is measured both ways.** `data-final-fixes` is the last stage Factorio has, so a mod that rewrites the defaults from a `data-final-fixes` sorting AFTER this one is past anything this mod can read, and there is no later stage to take an answer in. Measured on 2.0.77 with the same fixture staged both ways against the fixed guest: ahead of this mod the load completes and the mask comes out with six layers; given a dependency on `better-belt-balancer` so the engine runs it second (final-fixes at 0.475 against 0.595) the load exits 1 on the message above. The dependency entry is what puts the one known rewriter in front; a general answer does not exist.

**The other three hidden prototypes need nothing and must keep needing nothing.** They inherit, so a rewritten default reaches them by construction. Giving any of them an explicit mask would put it in `followCollisionDefaults`' position, and the gate arm asserts their absence rather than leaving it to the hash, which holds a field still without ever saying it has to be absent.

## The gate arm, and the red proof

**`check_cerys` in `test/check-datastage.py`, the gate's twenty-fourth arm**, behind `bbbt-cerys`: a Lua fixture written at run time whose `data.lua` declares a collision layer and whose `data-final-fixes.lua` is **Cerys's own loop transcribed character for character**, tabs included, with the layer name substituted and nothing else touched. Lua rather than a `build_fixture` guest for `bbbt-remover`'s reason word for word -- no tinygo, no `fklua mod`, no compile, for a fixture whose whole content is one `data:extend` and one loop -- and because a rephrasing would be this repository's model of what that mod does rather than what it does. The name sorts ahead of `better-belt-balancer`, which is how it comes to run first, and the arm ASSERTS that off the engine's own `Loading mod` lines rather than resting on the alphabet.

What it asserts past the ordering: that the fixture's rewrite LANDED (so a stale jq path fails first), that `bbb-linked-belt`'s layers EQUAL the default's layer for layer rather than merely containing the new one, that the mask's keys are `layers` alone on 2.1 and `layers` plus a true `not_colliding_with_itself` on 2.0, that the other three state no mask, and that the guest SAID it followed -- which is the only thing in the run that separates the fix from a coincidence, because a dump records a mask and not who wrote it.

**RED-PROVEN on 2.0.77 by disabling the call and rebuilding.** The engine refuses the load with the reported message verbatim, and the arm reports it the way `run_arm` reports every refused load:

    ------------- Error -------------
    entity prototype "bbb-linked-belt" (linked-belt) collision_mask(Modifications:
    Better Belt Balancer) must collide with entity prototype "bbb-linked-belt"
    (linked-belt) collision_mask(Modifications: Better Belt Balancer).
    ---------------------------------
    ==> the default collision masks, 1 arm with a pack that rewrites them
    [cerys] --dump-data exited 1

exit 1. **That is the right shape for this arm rather than a FAIL line**: what was reported was a game that would not start, and an arm whose claim is a load has no dump to read a wrong value out of.

## Measured

Factorio 2.0.77 (build 84539, mac-arm64, steam), the `release/2.0` arm at 0.2.4, shipped config.

| | |
|---|---|
| the reporter's whole pack, 120 mods, before | exit 1, no dump |
| ...and with this package | **exit 0**, and `bbb-linked-belt` carries `cerys_water_tile` beside the five, with `not_colliding_with_itself` still true |
| the nine-mod bisect, before | exit 1, no dump |
| ...and with this package | **exit 0**, same mask |
| `bbb-linked-belt`'s layers against `default_collision_masks["linked-belt"]` in both | **equal, layer for layer** |
| `make datastage-check`, 24 arms | **green**, and the 2.0.77 goldens UNMOVED: `base data_raw_sha256 414aa4796d21afaf`, `base mod_settings_sha256 2270617e8f807ea0`, `incumbent data_raw_sha256 f67c4bf8544a9010`, `incumbent mod_settings_sha256 2270617e8f807ea0` |

**The goldens not moving is the acceptance criterion and not a footnote**: the fix writes nothing in a game nobody rewrote the defaults in, so every recorded number in this repository is still a number about this prototype table.

**What it costs**, the same tree either side, `make zip`: the zip **918,022 -> 919,862 B** (+1,840, +0.20%) and `dist/bbbdata.wasm` **1,266,429 -> 1,268,924 B** (+2,495). Nothing outside `guest/go/data` moved, so the control guest is untouched and there is no runtime path here at all -- a data module runs once at load and dies with the Lua state that built it.

**WHAT IS NOT RUN: Factorio 2.1.** The binary here is 2.0.77, so the 2.1 flavour of this arm and the 2.1 golden rows are a SKIP by construction, exactly as they are for every other pass taken on this machine. The branch the fix rests on is the one `go test ./data/` already proves both arms of; what 2.1 would add is that the same re-read leaves a load that never needed it alone.
