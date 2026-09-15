# Priority ports -- what the planner builds, what it refuses, and what a toggle costs

The planner's half of the input/output priorities feature: `guest/go/plan`. The guest's half -- the per-part flag, the UI, the persistence and the refusal wiring -- is somebody else's, and the seam between them is `plan.Edge.Prio`, `plan.Ports.QIn/QOut`, `plan.ShapeEdges` and `plan.Op.OutPrio/InPrio`.

**OUTPUT priority is built and verified. INPUT priority is refused**, and the last section says why and what building it would take.

Written for a session that has not seen the design conversation. Read `guest/go/plan/plan.go`'s package header first if you have not: the plain butterfly, the jumper blocks and the loopback are all assumed here.

---

## The promise

A player ticks some of a balancer's output ports. What they then get, with S belts arriving and q of the M outputs ticked:

| | per port |
|---|---|
| a ticked (priority) port | `min(S/q, 1)` |
| every other port | `min(max(S - q, 0) / (M - q), 1)` |

That is: the ticked ports are fed FIRST and share what they get equally; the rest share what is left, equally. Exact at every load, not close at most of them -- which is the whole reason this mod exists rather than an approximation of it.

Two more properties the construction is held to, because a balancer promises them too:

- **equal offers draw equal intake.** A network that cannot take everything on offer takes the same from every input.
- **a blocked port is absorbed.** An output belt that cannot take anything leaves the rest of its tier sharing what it would have had.

And the rule above all of them: **zero per-tick, per-item script.** Priority is expressed entirely in the compiled network, through vanilla splitters with `output_priority` set at create time, and a shape that cannot be expressed that way is REFUSED rather than emulated.

## Why one priority butterfly is not the answer

Setting `output_priority` on every splitter of the plain butterfly **destroys the remainder rather than sharing it**: two belts into a 4x4 come out `[1, 0, 1, 0]`.

That is not a bug to be fixed, it is what a priority splitter IS. A splitter that fills one side first is a MERGE, and a tree of merges is a CONCENTRATOR. The plain butterfly's whole job is the opposite. So the construction uses both, in that order.

## The construction

```
band 0, P rows      col 0   recv (a linked belt's output end), one per used input row
                    col 1   lane splitter
                            BALANCE      k stage columns + k-1 jumper blocks
                            CONCENTRATE  the same again, output priority on the smaller y
                    last    one tap (a linked belt's input end) per rank below M+Loop

band 1, under it    PRIO balancer, shape (q, q)   |   RESID balancer, shape (M-q, M-q)
```

- **BALANCE** is the plain butterfly over P rows with every row an output, so `Loop` is zero, nothing dead-ends, and every row leaves it carrying exactly `S/P`. It is not an optimisation and cannot be dropped even for `q = 1`, where the tap would be right without it: it is what makes the network draw EQUALLY from its inputs, because the concentrator's tree is deliberately asymmetric and behind it the row an input happened to sit on would decide how much of it was taken once the network filled.

- **CONCENTRATE** is BALANCE's own schedule with every splitter's output priority on the smaller y. `order` puts the line whose bit s is clear on the smaller y of every stage-s pair, so the same half wins at every stage and the winners meet each other at the next one. Over the equal rows BALANCE produced, that sorts: **the row of rank r carries `clamp(S - r, 0, 1)`**. A line's value is the composition of `min(2x, 1)` for a win and `max(2x-1, 0)` for a loss, which telescopes to `clamp(S - rev(line), 0, 1)` where `rev` reverses the line index's k bits. `sorterRanks` is that: the rank of a physical row is the bit reversal of the line standing on it after the last stage.

- **The tap column** sends ranks `0..q-1` to the PRIO tier, ranks `q..M-1` to the RESID tier, ranks `M..M+Loop-1` back to the head's spare rows, and gives ranks past that no tap at all. Ranks `0..q-1` hold `min(S, q)` between them, so the priority tier receives exactly what the promise says it should; the rest is the residual.

- **Each tier is a SQUARE plain butterfly**, shape `(t, t)` over `NextPow2(t)` rows, so every spare port loops back into a spare row and it has no dead end in it. That is what makes a tier exact at every load and not only at saturation.

One layer of one band, P = 8, with rank in brackets:

```
   row   BALANCE (plain)        CONCENTRATE (priority to the smaller y)      tap
    0  ---[ ]---.---[ ]---   ---[*]---.---[*]---.---[*]---   ------------>  rank 0  -> PRIO
    1  ---[ ]---'---[ ]---   ---[*]---'---[*]---'---[*]---   ------------>  rank 1  -> RESID
    2  ---[ ]---.---[ ]---   ---[*]---.---[*]---.---[*]---   ------------>  rank 4  -> RESID
    3  ---[ ]---'---[ ]---   ---[*]---'---[*]---'---[*]---   ------------>  rank 5  -> RESID
    4  ---[ ]---.---[ ]---   ---[*]---.---[*]---.---[*]---   ------------>  rank 2  -> RESID
    5  ---[ ]---'---[ ]---   ---[*]---'---[*]---'---[*]---   ------------>  rank 3  -> RESID
    6  ---[ ]---.---[ ]---   ---[*]---.---[*]---.---[*]---   ------------>  rank 6  -> spare
    7  ---[ ]---'---[ ]---   ---[*]---'---[*]---'---[*]---   ------------>  rank 7  -> spare

        [ ] a splitter with no priority      [*] one with output_priority = left
        the jumper blocks between stages are elided; a splitter spans the two
        rows its bracket pair covers, and left is the smaller y
```

The ranks are NOT in row order past P = 4, and that is the bit reversal rather than an error: `TestRanksAreAPermutation` says every row has exactly one and that row 0 is always rank 0.

### Ranks past the last output port

They are the plain network's spare ports, and they are wired the plain network's way: the first `Loop = min(P-M, P-N)` of them feed the head's spare rows and the rest dead-end.

**That is load-bearing and it was found by a failing test rather than by design.** Without it a saturated 7->5 drew 0.625 of a belt from four of its inputs, 0.75 from two and a whole belt from the seventh. The cause: a butterfly whose outputs are blocked UNEVENLY is not input-fair, and the back-pressure a concentrator sends back is shaped by its concentration tree. The plain network never has that shape, because `Shape`'s `Loop` fills every row of the head that a real input does not. Giving the priority network the same loopback gives it the same fairness. `TestSaturatedInputsAreDrawnEqually`.

The recirculated flow re-enters BALANCE and competes for the priority ranks again, which is harmless: a rank past `M` carries flow only when every real port is full, and at that point the priority ports are at a belt each and cannot take more.

## What the flow model is, and what it is not

`guest/go/plan/flowmodel_test.go`. `Propagate` and `PropagateLoop` are LINEAR -- they average a pair of rows, which is what a splitter does while both of its outputs can take their share -- and a priority splitter's whole behaviour is what happens when one side is at capacity. So the verification needed a model with capacity and back-pressure in it.

`Simulate` walks the ops `Build` emits, as a graph read off their POSITIONS, and solves a fixed point over two quantities per belt line: how much the downstream will accept and how much is moving. Four rules about Factorio go in:

- a splitter divides its input between its two outputs equally, and gives the whole of it to one output when the other cannot take its share;
- with an output priority it fills the priority side as far as that side will take;
- with an input priority it draws from the priority side first when it cannot take everything on offer;
- a belt line carries at most one belt, so a splitter carries at most two.

**What it proves**: that the wiring is right. It reads the real op list, so an op that moved a tile stops being connected here exactly as it would stop being connected in the game.

**What it does not prove**: that the engine agrees. Nobody has run a priority network in Factorio. The anchor that makes it worth anything is that on every plain shape with n <= m at every load it reproduces `PropagateLoop`, which is the model the M2 rigs were measured against and which matches them (3->5 saturated delivers three fifths of a belt per output, 782 items against a bare belt's 1306).

It lives in a `_test.go` file for a measured reason: `Propagate` really is dead-code-eliminated out of the wasm -- deleting it moves the `code` section not one byte -- and `Simulate` is not. Something in it (the closure `buildFlow` wires its links with, or the map it indexes tiles by) survives TinyGo's reachability pass and costs 1,103 bytes of compiled code for a function no guest can call. A test file cannot reach the wasm at all.

## What is verified

| | |
|---|---|
| every shape with n, m <= 8, every legal q, seven loads from 0.05 to saturation | **1,568 cases, delivery exact to 1e-9 and intake spread exactly 0** |
| P=16 and P=32, both sides of N against M, four loads | exact to 1.7e-13, intake spread 0 |
| uneven feeds (one input fed, the last input fed, 0.3/0.9/0.1/1.0, halving) | exact |
| a blocked port, priority or normal, at quarter load | the tier absorbs it |
| the concentrator's head and total at P = 2..32 | `rank 0 = min(S, 1)`, the rest carries `S - min(S, 1)` |
| every op inside its slot, and the extent exactly tight | `TestEveryPriorityShapeStaysInsideItsSlot` |
| no two ops on one tile | `TestNoTwoPriorityEntitiesShareATile` |
| every linked belt's input end paired exactly once | `TestEveryPriorityInputEndIsPaired` |
| only splitters carry a priority, only the concentrator's, only on the left | `TestOnlySplittersCarryAPriority` |
| determinism, and zero allocation on a warm buffer | as the plain path |

## The fit, and why the slot binds it

A priority network is two butterflies wide where the plain one is a single, and it carries its tiers in a second band of rows. Band 0 is `colFirstStage + 2*stageCols(P) + 1` columns; band 1 is `Width(NextPow2(q)) + Width(NextPow2(M-q))` wide and `max(NextPow2(q), NextPow2(M-q))` tall under it. `prioExtent` computes it and `ShapeEdges` refuses on it.

Widest shape at each P, with the slot at 32 x 72:

| P | band 0 width | worst extent | fits |
|---|--:|--:|---|
| 2 | 5 | 6 x 3 | yes |
| 4 | 11 | 11 x 8 | yes |
| 8 | 17 | 17 x 16 | yes |
| 16 | 23 | 23 x 32 | yes |
| 32 | 29 | 29 x 64 | yes |
| **64** | **35** | **35 x 128** | **no** |

So **a priority port is refused at P = 64 and allowed at every size under it**, and the boundary is not a constant anybody chose: P=64 wants 35 columns of a 32-column slot and there is no packing of two 64-row blocks that is not also 128 rows of a 72-row slot. The tests measure the extent off the ops rather than off the rule, and check it is exactly tight -- a loose bound would refuse shapes that fit.

**Ticking EVERY output is the plain balancer**, and `ShapeEdges` reports it as `QOut = 0`. Two tiers where the second is empty is one tier, and one tier is the network this mod already builds -- so a player who ticks all 64 outputs of a 64-port balancer gets a network rather than a refusal, and gets the plain one to the byte. The same collapse applies to the inputs.

The port cap still comes first: a cluster past `MaxPorts` is refused as over the port limit whether or not anything is ticked.

## What it costs

Entities, and the item positions the network can hold. A tile of belt holds eight item positions (0.25 of a tile per item along a lane, two lanes) and a splitter is two belts wide; the visible interfaces count, because a teardown drains the cluster's box as well as the slot.

| shape | plain entities | plain capacity | q | priority entities | priority capacity | growth | **residual bound** |
|---|--:|--:|--:|--:|--:|--:|--:|
| 2->2 | 11 | 112 | 1 | 18 | 192 | 1.71x | 80 |
| 4->4 | 32 | 320 | 1 | 71 | 736 | 2.30x | 416 |
| 4->4 | 32 | 320 | 2 | 58 | 608 | 1.90x | 288 |
| 3->5 | 72 | 720 | 1 | 143 | 1456 | 2.02x | 736 |
| 5->3 | 74 | 752 | 1 | 128 | 1312 | 1.74x | 560 |
| 8->8 | 84 | 832 | 1 | 199 | 2016 | 2.42x | 1184 |
| 8->8 | 84 | 832 | 2 | 203 | 2064 | 2.48x | 1232 |
| 8->8 | 84 | 832 | 4 | 176 | 1792 | 2.15x | 960 |
| 16->16 | 208 | 2048 | 1 | 515 | 5152 | 2.52x | 3104 |
| 32->32 | 496 | 4864 | 1 | 1267 | 12576 | 2.59x | 7712 |

The entity count is **not monotone in q**, and that is the tier balancers rather than an error: a 4x4 costs more with one priority port than with two, because one leaves three normal ports and a square butterfly over three ports is four rows with a loopback in it, where two and two are a pair of single-splitter blocks.

**A recompile is linear in entities**, and the measured constant is the host boundary: `agents/maxports.md` has a 1,152-entity 64x64 at 155 ms empty and ~390 ms saturated, and an 84-entity 8x8 at ~11.5 ms empty. On that constant a 32->32 with a priority port is 1,267 entities, which is the largest network this mod would build -- larger than today's largest, at a size the port cap allows without comment.

### What a toggle costs in items

Toggling a priority flag is a RECOMPILE: the network comes down, the drained items go into the carry pool, and the network that goes up takes them back in plan order (`carry.go`, "A recompile is not a removal"). Nothing is ever spilled on the hidden surface and nothing goes to a player's inventory. What the successor cannot hold takes the existing visible-surface spill beside the cluster.

So the direction matters, and it is asymmetric:

- **turning a priority port ON never spills.** The successor is bigger than the predecessor in every shape in the table, so everything the teardown drained fits.
- **turning the LAST priority port off** shrinks the network back to the plain one, and the residual is `max(0, what was standing - plain capacity)`. The bound is the last column of the table.

**The bound needs a network that is completely full, which means one whose outputs are all blocked -- and a jammed network really is near capacity.** That is the one place the capacity arithmetic above can be checked against the game, and it checks out: CLAUDE.md records a saturated dead-ended 4x4 draining **232 items**, against the 320 positions this file computes for it (73%), and M2's full 2x2 draining **72** against 112 (64%). The shortfall is the splitters' internal buffers, which no transport line reports.

So the residual is not theoretical. A jammed 8->8 whose last priority flag is cleared can put on the order of a thousand items on the ground, and **that is the one number in this design that is not small**. What keeps it rare rather than small is that it needs every output of that balancer blocked at the moment of the toggle; a balancer that is moving anything has room.

It is written down rather than fixed because the two ways to shrink it are both worse. The concentrator is a whole butterfly and cannot be made cheaper without giving up the exactness the feature is for; and a toggle that refused to run on a full balancer would be a worse answer than a spill the player can walk over.

## What the model found about the plain network

Two results about the network this mod already ships, neither of them measured in a game, because no rig anywhere drives either shape. Both are recorded as tests in `flow_test.go` so that a change in either is noticed.

- **An `n > m` shape spreads under partial load.** Five inputs at half a belt into three outputs deliver 0.75, 0.75 and 1.0 rather than 0.8333 each. Its spare ports dead-end, a splitter whose one output is blocked gives the whole of its input to the other, and that re-routing is not symmetric. Saturated the same shape is exact, which is what M2's `a5to3` rig measures in the game (1304 items to each of three, 0.00% spread).
- **A blocked output is not rebalanced under partial load.** Block one port of a 4x4 and feed it a quarter of a belt on each input: the port that shares its last splitter with the blocked one takes double (0.25, 0.25, 0, 0.5). Saturated it is exact, which is M2's `block` rig.

The priority network does not share the second one: its tiers are square butterflies with every spare port looped back, so a blocked port's flow recirculates instead of doubling up on a neighbour. `TestABlockedOutputIsRebalancedByTheTiers`.

Whether the engine agrees with either is unknown and would want a rig at partial load.

## What input priority does

**Nothing. `ShapeEdges` refuses an edge list with `QIn > 0`**, and `Build` refuses it too.

That is a decision and not an oversight. An approximation would be the one outcome this repository's own rule forbids: a network that compiled, ran, balanced, and quietly ignored the flag a player set. The refusal is delivered through the same machinery the port limit and the one-belt-per-part rule use.

**The mirror construction exists and is measured to be the same shape.** An input priority is the output one with the arrows reversed: the tiers move to the input side, the concentrator becomes a DE-concentrator (a butterfly with `input_priority` on the smaller y, which drains its low ranks first), and BALANCE stays where it is. Probed with the model on a plain butterfly with every splitter's `InPrio` set to the left:

| | intake, every input offering a full belt |
|---|---|
| 4 inputs, 1 output | `[1, 0, 0, 0]` |
| 8 inputs, 1 output | `[1, 0, 0, 0, 0, 0, 0, 0]` |
| 4 inputs, 2 outputs | `[1, 0, 1, 0]` |

The first two are exactly what a de-concentrator should do. The third is the mirror of `[1, 0, 1, 0]` on the output side and says the same thing: a de-concentrator alone over-drains a SET of inputs rather than strictly the lowest ranks, so it needs a balancing butterfly and a pair of input-side tier balancers in front of it, exactly as the output side needed them behind.

What building it would take, in order:

1. an input-rank function, the mirror of `sorterRanks`, measured against the model rather than derived;
2. a band laid `PRIOIN | NORMIN` tiers, then `DE-CONCENTRATE`, then `BALANCE`, then the outputs. Same 6k-1 columns as the output side, so the same fit table: P <= 32;
3. the semantics stated as sharply as the output side's, which is harder than it looks -- an input priority does nothing at all until the network is backed up, so its whole observable is under saturation;
4. **and the combination**, which is the part that does not fit. Input AND output priority in one network is `DE-CONCENTRATE + BALANCE + CONCENTRATE` in series: 9k-3 columns, which is 24 at P=8 and 33 at P=16 against a 32-column slot. Past P=8 it would have to be split across two bands with a linked-belt junction between them, at two more entities a row.

Until then, `Op.InPrio` is a field nothing sets. `TestOnlySplittersCarryAPriority` asserts that no op carries one, which is what makes the guest's executor safe to write against it.

## The seam, for the guest side

- **`ShapeEdges(edges) (Ports, fits)` is the ONE pre-teardown check**, and it is safe to call from a keypress: it allocates nothing, touches no package buffer and no other state, reads the edge COUNTS rather than the order, and its `fits` is exactly the answer `Build` gives a tick later for the same list. `TestShapeEdgesIsAnswerableFromAKeypress` holds all four.
- **`Op.OutPrio` and `Op.InPrio`**: 0 is none, -1 is left, +1 is right, in the engine's own sense, which is relative to the direction the splitter faces. Every hidden splitter faces East, so left is North, the smaller y of the two tiles a splitter spans. **Only `ProtoSplitter` ops ever carry a non-zero value, only `OutPrio`, and only `PrioLeft`**; no lane splitter, belt or linked belt does, and nothing sets `InPrio` at all.
- **`Ports.QOut`** is the count of ticked outputs AFTER the collapse: `M` ticked outputs reports 0. `Ports.Loop` still describes the head's loopback and is still `min(P-M, P-N)`.
- **Port order**: the priority output edges take ports `0..q-1` in edge order and the rest take `q..M-1`, but nothing outside the planner needs to know that. The visible interfaces are still appended in EDGE order, so the k-th visible end is the k-th edge, exactly as in the plain path.

### What a suite should assert

The load is in belts fed. Per output, with a bare belt in the same save as the yardstick, as the M2 rigs do it:

| rig | fed | priority port | every other port |
|---|---|---|---|
| 4->4, one ticked | 0.5 belts (one input at half) | **0.5** | **0** |
| 4->4, one ticked | 2 belts (two inputs full) | **1.0** | **0.3333** |
| 4->4, one ticked | 4 belts (saturated) | **1.0** | **1.0** |
| 2->2, one ticked | 0.5 belts | **0.5** | **0** |
| 2->2, one ticked | 1.5 belts | **1.0** | **0.5** |
| 2->2, one ticked | 2 belts (saturated) | **1.0** | **1.0** |

`TestTheRatesASuiteShouldAssert` pins the same six rows against the model. Two more worth a leg each: every input drawn equally when the rig is saturated, and a blocked priority port leaving the normal ports at exactly a third of a belt each on a 4x4 fed one belt.
