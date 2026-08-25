# The interactive checklist

Seven things a headless Factorio cannot check, and the scenes the mod portal's animations are captured from. Five of the seven need a real player: `game.get_player` resolves to nothing during `--create`, the player events cannot be raised from script, and a script build is not a cursor. The headless suites pin the arithmetic, the quantities and the negative cases; this checklist pins the triggers. The `bbb-interactive-setup` mod in this directory stages a rig for gestures A to E beside spawn on a fresh world, so each of those costs about thirty seconds, and stages the five demo scenes in a second column east of them. It stages and asserts nothing; the assertions are what you see and the `[BBB]` log lines listed below.

Gestures F and G need no staged rig and no player. They need your own save and your own graphical client, which is what the suites do not have.

Every rig here is built to Factorio 2.1's rule: **a balancer part connects to one belt**. A 4-in/4-out balancer is eight parts, a west column carrying the inputs and an east column carrying the outputs, and the smallest balancer is two parts. That rule is also why two of the gestures work the way they do: a part that already has its belt cannot be given another, so a belt that is meant to change a balancer's port count has to land on a part with nothing against it. Both rigs that need one stage a spare part for exactly that purpose. The design behind the rule is in [`agents/single-edge.md`](../../agents/single-edge.md).

## Setup

```sh
make install                # the mod itself, into your Factorio mods directory
make interactive-install    # the rig-staging mod beside it
```

Enable both mods and start a new freeplay world with any settings (the setup mod disables the crash site and the intro). You spawn beside the five labelled gesture rigs with 50 express belts and 10 balancer parts in your inventory, and map tags mark every rig and every demo scene. When done, disable `bbb-interactive-setup`; nothing it stages survives into ordinary play.

Afterwards, grep `factorio-current.log` in your Factorio user directory for the lines named under each gesture. They are verbose-level `[BBB]` lines, so the mod must be the default build rather than `QUIET=1`.

The staged world itself is verified headlessly by `test/run.sh iact`, which fails on a rig that did not land, a rig that compiled to a shape other than the one intended, or any refusal at all. The refusals below are the gestures' doing; a rig that arrives already refused would waste your session.

## A. Mining a balancer part by part (y = -24)

A saturated 4-in/4-out balancer over eight parts, two columns of four, with its outputs dead-ended so it stays full. Mine it one part at a time. At every step, not only at the last part, the drained items must land in your inventory rather than on the ground. A full inventory must still spill the remainder, which is vanilla's rule too.

Log line: `offered ... items to player ... before the floor`.

## B. The belt at the edge (y = -12)

A saturated 2-in/2-out balancer over four parts, with a fifth part attached below it that carries no belt at all. Three gestures on the one rig.

**Grow it.** Lay a south-facing belt at (20, -9), on the spare part's south face. The machine goes from two ports to four.

**Shrink it again.** Mine that belt. The machine halves back to two ports, so the network it rebuilds cannot hold everything the teardown drained, and the overflow must reach your inventory rather than the floor.

Log line for both: `offered ... items to player ... before the floor`, from the belt's removal.

**Then ask a part for a second belt.** Lay a south-facing belt at (20, -13), against the north face of a part that already has its input belt on the west. Expected:

- red flying text at the refused belt saying a balancer part connects to one belt;
- the standard cannot-build sound;
- the belt back in your inventory a tick later;
- the balancer never stops, and nothing hits the ground.

Log lines: `carrying more than one belt` for the refusal, and `handed the refused piece` for the belt coming back. The hand-back line names the bound that fired in parentheses, which here is `(past the one-belt-per-part rule)`.

## C. The sixty-fifth belt (y = 0)

Sixty-four parts carrying one input belt each, one part below them carrying the single output, and one spare part above them at (20, -1) with nothing against it. P = 64, the limit exactly, and visibly running. Most of the input belts are bare single-tile stubs and only eight rows have sources behind them; that is deliberate, not clutter. Each stub is one of the sixty-four ports the gesture is about (the port count comes from the belts standing against the machine, fed or not), and sixty-four sources would bury the rig in chests for no extra evidence. The same goes for band D's two blocks. Lay one more belt against the spare part at (20, -2), facing south. Expected, all at once:

- red flying text at the refused belt naming the 64-belt limit (at the belt, not over the machine's centre, which on a 34-row column is off screen);
- the standard cannot-build sound;
- the belt back in your inventory a tick later;
- the balancer never stops and nothing hits the ground.

Repeat with a full inventory: the message still appears and the belt stays standing, unconnected. A construction robot placing the belt from a ghost gets a force-wide chat message instead and the belt stands; that arm is asserted headlessly, so seeing it here is optional.

Log lines: `handed the refused piece` (the normal case), `could not be handed back` (the full-inventory case). Both name the bound that fired in parentheses, `(over the port limit)` here and `(past the one-belt-per-part rule)` in gesture B, so a returned piece says which rule sent it back.

## D. Bridging two balancers over the limit (y = 61, the gap)

Two 32-input balancers, both running, one tile apart, with one more input belt already standing beside the gap. Place one balancer part in the gap at (20, 61). The merged machine would need 65 inputs, over the limit, so expected:

- the same refusal feedback as C;
- the part back in your inventory;
- both balancers keep running, untouched, throughout (watch their output chests keep filling);
- nothing on the ground.

Log lines: the over-limit alert naming the merge, `would merge into a cluster this mod cannot build`, and no `[BBB] error:` anywhere.

At any point, `/bbb-audit` in the console prints cluster and network counts. While a refused merge stands, `drift=1 unbuilt=0 refused=1` is correct; `drift=0 unbuilt=1` would mean the merge tore the machines down.

## E. Fast replace, both ways (y = 90)

A four-part balancer with a belt line running east into the tile below its south-west corner, and below that a five-part column fed on its top and bottom parts only.

**A part over a belt.** Hold a balancer part over the belt line's last tile at (20, 92). The cursor must show the fast-replace preview rather than a red block. Place it. Expected: the belt vanishes, the part takes the tile and joins the balancer above, the belt and whatever it was carrying arrive in your inventory, and the balancer is now three in and two out. Nothing on the ground.

The line stops there on purpose. A part dropped into the middle of a running line would take the belt behind it as an input and the belt ahead of it as an output, which is two belts on one part, and that is refused. Dropping a part into a line is a gesture that only works at the line's end now.

**A belt over a part.** Hold an express belt over (20, 98), the middle part of the column. The cursor must show the fast-replace preview. Place it. Expected: the part vanishes, the belt takes the tile, the part arrives in your inventory, and the column becomes two balancers with the new belt an output of the upper one and an input of the lower one. Run `/bbb-audit`: `drift=0 unbuilt=0`, one more cluster and one fewer part than before. A tile still counted as a part with your belt standing on it is the defect this gesture exists to catch.

**And the refusal.** Hold the same belt over (20, 96) or (20, 100), the two parts at the ends of the column. Those carry the balancer's edge interfaces, and an interface is a belt-connectable of its own, so the build must be refused and nothing may happen. Only parts with no belt against any of their free faces can be replaced this way.

The two parts either side of the middle, (20, 97) and (20, 99), can also be replaced, and they leave one of the halves asking a part for two belts, so that half is refused. The middle tile is the one that splits cleanly.

Log lines: `a belt-connectable fast-replaced the part at 20,98` for the reverse gesture, and `compiled cluster ... 3->2` for the forward one. Neither direction may produce a `[BBB] error:` or a spill line.

## F. Adopting a Belt Balancer 2 or 3 save

No staged rig for this one, and no player needed either: the headless suite drives the whole conversion, against a stand-in and once by hand against the real Belt Balancer 2. What it cannot do is look at the result or hold a blueprint book.

Take a save of your own that uses Belt Balancer, Belt Balancer 2, Belt Balancer 3 or Belt Balancer Performance, with balancers running and items on the belts. Uninstall that mod, install this one, and load. Expected:

- one `[BBB] legacy: adopted N parts from M surfaces into K clusters` line, with `trigger=configuration_changed` if this mod was already installed and `trigger=init` if it arrived in the same edit;
- every balancer where it was, drawing this mod's plating rather than a hole;
- the belts around them still carrying what they were carrying;
- a stack of the old part in a chest still there, and placing this mod's parts;
- the balancer technology researched, so the recipe is available;
- no `[BBB] error:` anywhere.

On Factorio 2.1 those adopted balancers then stop, and that is the correct outcome rather than a fault: an incumbent's balancer connects several belts to one part by construction, which 2.1 does not allow, so each one is converted and then refused with the chat summary and the pings gesture G describes. What you get is a rebuild checklist, not a working machine. Rebuild each one with one belt per part and it runs.

Then place one of your old blueprints of a balancer. Its ghosts ask for this mod's part item, and each part a robot revives becomes one of this mod's parts a tick later. The swap itself is covered headlessly; what is not is the construction network and the blueprint that reach it.

Nothing at all should happen if you load with both mods still installed. The log says so once, naming the mod and its version.

## G. Opening a Factorio 2.0 save on Factorio 2.1

Also no staged rig, and also no player. The headless suite loads two committed 2.0 saves under 2.1 and asserts everything the mod does to them; what it cannot do is tell you whether the message reads well or whether the pings go where they say.

Take a save of your own built on Factorio 2.0 with balancers whose parts connect to more than one belt, which is every balancer built before this rule existed. Open it on Factorio 2.1 with this mod installed. Factorio itself gets there first: it deletes all but one belt-connectable per tile at load, silently and before any script runs, so the balancers arrive already crippled. Expected, once the save is open:

- one chat message per force that owns an affected balancer, saying how many need rebuilding and that each part now connects to one belt;
- a `[gps=...]` ping per balancer in that message. Click them: each must centre the map on a real balancer of yours, on the right surface;
- those balancers stopped, with what they were holding on the ground beside them;
- every part still standing and still yours, with nothing of the mod's own machinery left around them;
- `/bbb-audit` reporting `drift=0 unbuilt=0` with `refused=` equal to the number of balancers the message named.

Rebuild one of them one belt per part and it compiles and runs, which is the other half of the check.

Log lines: `[BBB] single-edge: N balancers were built with several belts per part`, one `[BBB] single-edge: told force ... about N balancers` per affected force, and a `torn down cluster` and `spilled ... items beside cluster` pair per balancer. No `[BBB] error:` anywhere.

## The demo scenes (x = 56)

Five saturated scenes in a second column east of the gesture rigs, staged so that the mod portal's animations can be captured again from a known world instead of from a save nobody kept. From north to south:

| scene | shape | parts |
|---|---|---|
| cross | 1 in, 3 out | 5, a plus whose four arms carry one belt each and whose centre carries none |
| compact column | 8 in, 8 out | 16, a 2 by 8 block, the smallest 8-to-8 the rule allows |
| c-shape | 8 in, 8 out | 18, a ten-part spine with two arms |
| c-shape | 8 in, 9 out | 19, the same with one more part on the top arm, which is a 16-port butterfly with loopbacks |
| long run | 8 in, 8 out | 16, a single row taking its inputs from the north and giving its outputs to the south, alternately |

Each one runs saturated from the moment the world is created. Capture with alt-mode off for the plain view and on for the input and output arrows. The 1-to-3 fan-out on a single part that earlier animations showed is not buildable under this rule, because one part cannot carry four belts; the cross is the same read.

## A false alarm to recognise

If a balancer ever compiles on top of ground items (only possible after something else has already littered the tiles), the engine relocates the items during entity placement: items lying on the part tiles are absorbed into the new interfaces' transport lines and ride through the machine to the outputs, conserved, and whatever exceeds the lines' capacity is nudged to the nearest free ground, which is the edge of the existing litter. That reads as fresh spill appearing at the litter's frontier, and it is not: the mod placed nothing on the ground and the guest log carries no spill line in that window. Measured by diffing two autosaves that bracket such a recompile: 204 plates left the part tiles, 89 landed on the frontier ring, all item names conserved.
