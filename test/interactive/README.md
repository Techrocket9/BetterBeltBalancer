# The interactive checklist

Six things a headless Factorio cannot check. Five of them need a real player: `game.get_player` resolves to nothing during `--create`, the player events cannot be raised from script, and a script build is not a cursor. The headless suites pin the arithmetic, the quantities and the negative cases; this checklist pins the triggers. The `bbb-interactive-setup` mod in this directory stages a rig for gestures A to E beside spawn on a fresh world, so each of those costs about thirty seconds. It stages and asserts nothing; the assertions are what you see and the `[BBB]` log lines listed below.

The sixth, F, needs no player and no staged rig. It needs your own save and your own graphical client, which is what the suites do not have.

## Setup

```sh
make install                # the mod itself, into your Factorio mods directory
make interactive-install    # the rig-staging mod beside it
```

Enable both mods and start a new freeplay world with any settings (the setup mod disables the crash site and the intro). You spawn beside five labelled rigs with 50 express belts and 10 balancer parts in your inventory, and map tags mark the gesture for each rig. When done, disable `bbb-interactive-setup`; nothing it stages survives into ordinary play.

Afterwards, grep `factorio-current.log` in your Factorio user directory for the lines named under each gesture. They are verbose-level `[BBB]` lines, so the mod must be the default build rather than `QUIET=1`.

## A. Mining a balancer part by part (y = -24)

A saturated 4-part balancer with its outputs dead-ended, so it stays full. Mine it one part at a time. At every step, not only at the last part, the drained items must land in your inventory rather than on the ground. A full inventory must still spill the remainder, which is vanilla's rule too.

Log line: `offered ... items to player ... before the floor`.

## B. Mining a belt off the edge (y = -12)

A saturated 2-part balancer with a free south face. Lay a south-facing belt on that face at (20, -10), wait a second, then mine the belt. Adding it grows the machine (P from 2 to 4) and mining it halves the machine again (P from 4 to 2); the overflow the smaller machine cannot hold must reach your inventory, not the floor.

Log line: the same `before the floor` line, from the belt's removal.

## C. The sixty-fifth belt (y = 0)

Thirty-two parts carrying 64 input belts and one output: P = 64, the limit exactly, and visibly running. Lay one more belt against it at (20, -1), facing south. Expected, all at once:

- red flying text at the refused belt naming the 64-belt limit (at the belt, not over the machine's centre, which on a 32-part column is off screen);
- the standard cannot-build sound;
- the belt back in your inventory a tick later;
- the balancer never stops and nothing hits the ground.

Repeat with a full inventory: the message still appears and the belt stays standing, unconnected. A construction robot placing the belt from a ghost gets a force-wide chat message instead and the belt stands; that arm is asserted headlessly, so seeing it here is optional.

Log lines: `handed the over-limit piece` (the normal case), `could not be handed back` (the full-inventory case).

## D. Bridging two balancers over the limit (y = 56, the gap)

Two 32-input balancers, both running, one tile apart, the gap already flanked by two more input belts. Place one balancer part in the gap at (20, 56). The merged machine would need 66 inputs, over the limit, so expected:

- the same refusal feedback as C;
- the part back in your inventory;
- both balancers keep running, untouched, throughout (watch their output chests keep filling);
- nothing on the ground.

Log lines: the over-limit alert naming the merge, and no `[BBB] error:` anywhere.

At any point, `/bbb-audit` in the console prints cluster and network counts. While a refused merge stands, `drift=1 unbuilt=0` is correct; `drift=0 unbuilt=1` would mean the merge tore the machines down.

## E. Fast replace, both ways (y = 84)

A two-part balancer with a plain belt line running east one tile below it, and below that a four-part column fed only on its top and bottom rows.

**A part over a belt.** Hold a balancer part over the belt line at (20, 86). The cursor must show the fast-replace preview rather than a red block. Place it. Expected: the belt vanishes, the part takes the tile and joins the balancer above, the belt and whatever it was carrying arrive in your inventory, and the balancer is now three in and three out with all three outputs filling. Nothing on the ground.

**A belt over a part.** Hold an express belt over (20, 90) or (20, 91), the two middle parts of the column. The cursor must show the fast-replace preview. Place it. Expected: the part vanishes, the belt takes the tile, the part arrives in your inventory, and the column becomes two balancers with the new belt an output of the upper one and an input of the lower one. Run `/bbb-audit`: `drift=0 unbuilt=0`, one more cluster and one fewer part than before. A tile still counted as a part with your belt standing on it is the defect this gesture exists to catch.

**And the refusal.** Hold the same belt over (20, 89) or (20, 92), the two parts at the ends of the column. Those carry the balancer's edge interfaces, and an interface is a belt-connectable of its own, so the build must be refused and nothing may happen. Only parts with no belt against any of their free faces can be replaced this way.

Log lines: `a belt-connectable fast-replaced the part at 20,90` for the reverse gesture, and `compiled cluster ... 3->3` for the forward one. Neither direction may produce a `[BBB] error:` or a spill line.

## F. Adopting a Belt Balancer 2 or 3 save

No staged rig for this one, and no player needed either: the headless suite drives the whole conversion, against a stand-in and once by hand against the real Belt Balancer 2. What it cannot do is look at the result or hold a blueprint book.

Take a save of your own that uses Belt Balancer, Belt Balancer 2, Belt Balancer 3 or Belt Balancer Performance, with balancers running and items on the belts. Uninstall that mod, install this one, and load. Expected:

- one `[BBB] legacy: adopted N parts from M surfaces into K clusters` line, with `trigger=configuration_changed` if this mod was already installed and `trigger=init` if it arrived in the same edit;
- every balancer where it was, drawing this mod's plating rather than a hole, and running within a second;
- the belts around them still carrying what they were carrying;
- a stack of the old part in a chest still there, and placing this mod's parts;
- the balancer technology researched, so the recipe is available;
- no `[BBB] error:` anywhere.

Then place one of your old blueprints of a balancer. Its ghosts ask for this mod's part item, and each part a robot revives becomes one of this mod's parts a tick later. The swap itself is covered headlessly; what is not is the construction network and the blueprint that reach it.

Nothing at all should happen if you load with both mods still installed. The log says so once, naming the mod and its version.

## A false alarm to recognise

If a balancer ever compiles on top of ground items (only possible after something else has already littered the tiles), the engine relocates the items during entity placement: items lying on the part tiles are absorbed into the new interfaces' transport lines and ride through the machine to the outputs, conserved, and whatever exceeds the lines' capacity is nudged to the nearest free ground, which is the edge of the existing litter. That reads as fresh spill appearing at the litter's frontier, and it is not: the mod placed nothing on the ground and the guest log carries no spill line in that window. Measured by diffing two autosaves that bracket such a recompile: 204 plates left the part tiles, 89 landed on the frontier ring, all item names conserved.
