# Priority ports -- what the planner builds, what it refuses, and what a toggle costs

The whole of the input/output priorities feature: the planner (`guest/go/plan`), which decides what a priority network IS, and the guest around it (`guest/go/priority.go`), which is where the flag lives, how a player moves it and what happens when it cannot be honoured. The seam between them is `plan.Edge.Prio`, `plan.Ports.QIn/QOut`, `plan.ShapeEdges`, `plan.Reinsertable` and `plan.Op.OutPrio/InPrio`.

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

The port cap comes first: a cluster past `MaxPorts` is refused as over the port limit whether or not anything is ticked, and that is true of the sentence a TOGGLE gets as well as of the one a build gets. It is the only one of the three bounds a flag cannot fix, so naming a priority bound instead would send the player to take a flag off and leave them refused for a reason nobody mentioned.

## What it costs

Entities, and how many items the network can be given back. The second column is `plan.Reinsertable` and it is a LOWER bound rather than a count of belt positions: the tile arithmetic (eight positions a tile, a splitter over two tiles, a lane splitter over one, the visible interfaces included because a teardown drains the cluster's box as well as the slot) is 25% over what a jammed 2->2 gave a real teardown and 19% over what a jammed 4->4 did, so the bound is the arithmetic less a third. `plan.Reinsertable`'s header is that measurement and the discount it fixes.

| shape | plain entities | plain takes back | q | priority entities | priority takes back | growth | **residual bound** |
|---|--:|--:|--:|--:|--:|--:|--:|
| 2->2 | 11 | 64 | 1 | 18 | 106 | 1.71x | 42 |
| 4->4 | 32 | 192 | 1 | 71 | 442 | 2.30x | 250 |
| 4->4 | 32 | 192 | 2 | 58 | 362 | 1.90x | 170 |
| 3->5 | 72 | 448 | 1 | 143 | 912 | 2.02x | 464 |
| 5->3 | 74 | 458 | 1 | 128 | 816 | 1.74x | 358 |
| 8->8 | 84 | 512 | 1 | 199 | 1253 | 2.42x | 741 |
| 8->8 | 84 | 512 | 2 | 203 | 1280 | 2.48x | 768 |
| 8->8 | 84 | 512 | 4 | 176 | 1109 | 2.15x | 597 |
| 16->16 | 208 | 1280 | 1 | 515 | 3258 | 2.52x | 1978 |
| 32->32 | 496 | 3072 | 1 | 1267 | 8037 | 2.59x | 4965 |

The entity count is **not monotone in q**, and that is the tier balancers rather than an error: a 4x4 costs more with one priority port than with two, because one leaves three normal ports and a square butterfly over three ports is four rows with a loopback in it, where two and two are a pair of single-splitter blocks.

**A recompile is linear in entities**, and the measured constant is the host boundary: `agents/maxports.md` has a 1,152-entity 64x64 at 155 ms empty and ~390 ms saturated, and an 84-entity 8x8 at ~11.5 ms empty. On that constant a 32->32 with a priority port is 1,267 entities, which is the largest network this mod would build -- larger than today's largest, at a size the port cap allows without comment.

### What a toggle costs in items

Toggling a priority flag is a RECOMPILE: the network comes down, the drained items go into the carry pool, and the network that goes up takes them back in plan order (`carry.go`, "A recompile is not a removal"). Nothing is ever spilled on the hidden surface and nothing goes to a player's inventory, because a toggle records no mine claim and a claim is what makes a removal's leftovers somebody's property.

**IT NEVER REACHES THE GROUND, because a change that would not fit is refused instead.** That is the guest's `prioFitsWhatIsStanding`, and it is a decision this design took the other way first: the residual was written down as a cost a player could walk over rather than a thing to prevent. The reason it moved is the size of it. A jammed 8->8 whose last flag is cleared has the last column of the table to put on the floor, on the order of a thousand items, and none of it is the player's doing in the sense a mine is. A mine says take this machine apart; a flag says configure it.

So before the flag moves, the guest asks `plan.Reinsertable` what the successor could take back and what the predecessor could. If the successor is smaller it counts what the standing network is holding, and refuses the change when the count exceeds what the successor could take. Nothing is torn down and nothing is said in chat: the balancer is still running and the player is told to let it empty.

**THE DIRECTION IS NOT THE QUESTION, AND THE FIRST CUT OF THIS SECTION HAD THAT WRONG.** It read "turning a priority port ON never spills", which is true of the plain network against a priority one and false of q against q+1: the table's own 4x4 rows are 736 positions at q=1 and 608 at q=2, for the tier-balancer reason the paragraph above the table gives. So a toggle ON shrinks that network. The guard compares two capacities and never asks which way the flag went, and `TestAPriorityToggleCanShrinkTheNetworkInEitherDirection` is the row that says why.

**The count is items and the bound counts belt positions**, and under belt stacking those are not the same unit: a stacked position holds up to four items, so the count can exceed the positions occupied and the guard can refuse a change that would have fitted. That is the side to be wrong on. On every force that cannot stack, which is all of base Factorio, one item is one position and the comparison is exact.

**THE TILE ARITHMETIC IS NOT THE BOUND, AND ASSUMING IT WAS IS WHAT THE FIRST CUT OF THIS GUARD GOT WRONG.** The two places this repository has drained a jammed network of a shape the arithmetic can be computed for: CLAUDE.md's M2 conservation check puts **72 items** in a full plain 2->2, where the tiles say 96, and the `hand` leg's first shrink puts **232** in a saturated dead-ended plain 4->4, where they say 288. A guard comparing what the machine holds against the tile arithmetic therefore passes a toggle on a 2->2 holding anything from 65 to 96 items and spills the difference -- with the refusal message, which tells the player to let the balancer empty, walking them into the band rather than out of it. The shortfall is the splitter family: eight linked belts alone account for all but 8 of the 2->2's 72, so one splitter and two lane splitters gave back 8 where the arithmetic claims 32.

`plan.Reinsertable` is therefore the arithmetic less a third, which is under both measured fractions (75.0% and 80.6%) and under the pessimistic per-tile reading of them on every shape this planner builds, tightest at 1->4 with q=2 and 2.48 points to spare. It is a bound and not a measurement; being under costs a refusal a player did not need, and being over costs items on the floor.

That is also what says the guard is reachable rather than theoretical. A priority 2->2 takes back 106 and the plain 2->2 it becomes takes back 64; a jammed priority 2->2 is carrying around 120 items, well over the 64. So the suite's rig is a dead-ended 2->2: fill it, clear the flag, and the refusal fires.

### What the guard does not cover

Two shrinks, and neither is a flag changing:

- **a belt MINED off a priority port** shrinks the machine too, and takes the rule every mine takes: what the successor cannot hold is offered to the player who mined it and only then to the floor (CLAUDE.md, "The miner's pocket"). That is the contract for a removal and this guard has no business overriding it.
- **a blueprint or a paste that lands a flagged part into a balancer that is then too big** is refused by `compile`, in front of its own teardown, so there is nothing standing to lose.

And one comparison the guard declines to make: a pre-toggle shape that does not fit has no capacity, so `Reinsertable` gives it 0 and the guard allows rather than counting. Refusing there would refuse the one gesture that gets a refused balancer working again, and what is standing under such a cluster was built by neither shape -- the compiler refuses in front of the teardown, so a network it can no longer build is one from before the edges moved.

And one comparison it declines to make: a pre-toggle shape that does not fit has no capacity, so `Reinsertable` gives it 0 and the guard allows rather than counting. Refusing there would refuse the one gesture that gets a refused balancer working again, and what is standing under such a cluster was built by neither shape -- the compiler refuses in front of the teardown, so a network it can no longer build is one from before the edges moved.

The two writers of the flag are the toggle, which is guarded, and `recoverPriority`, which raises one only on a part whose picture the guest has never written. That part arrived with its belts, so the cluster it lands in is recompiled by the BUILD it came in on, and a build that shrinks a machine has taken the ordinary recompile-and-spill rule since long before priorities existed.

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

## The guest's half

`guest/go/priority.go`, and one byte per part.

### The flag

`pprio` is one byte a node in `cluster.go`'s parallel slices, beside `pvar`, and it is the PART's rather than the belt's: `classifyEdges` reads it once per tile and stamps it onto every edge that tile carries, which on an engine that lets a part hold two belts is both of them. A player flags a part.

It is in the compile FINGERPRINT (`compile.go`), which is what makes a toggle a recompile: the flag is the only thing that moves, nothing in the world does, and a hash blind to it would make the gesture a silent no-op -- the shape `m3`'s `swap` rig is about. It costs the `Dir` field one more bit of shift and no more mixing.

### Where it survives

In `graphics_variation`, which is the one piece of per-entity state this mod already writes and the engine already persists. Cells 1..47 are the shapes and 48..94 are the same shapes badged, so the byte is the in-world indicator at the same time. The guest heap is declined on every rebuilt guest, so a flag kept only there would be lost on every release of this mod and a player's factory would quietly go back to balancing evenly.

Measured on 2.0.77 against a prototype declaring `variation_count = 94`:

| | |
|---|---|
| 94 | reads back 94 |
| 95 | reads back 1 |
| 255 | reads back 67 |
| 0 | refused, "allowed values are from 1 to 256" |

so the engine wraps modulo the count. A blueprint over a `simple-entity-with-force` with a `placeable_by` carries `variation = 50` and a revived ghost comes back at 50, so a blueprint round-trip keeps the flag with nothing asked of the guest.

`recoverPriority` (`skin.go`) is the other half: a part whose picture the guest has never written -- a revived ghost, a clone, or a whole world on a fresh heap -- has its variation read back, and ONLY the flag is taken, because a pasted part's neighbourhood is not its source's. One host call for the whole cluster through the bulk getter, skipped entirely when every candidate's picture is already known.

Driven end to end on 2.0.77 through the remote method on a 2x2: the part at 10,10 flips and restyle writes variation 21 -> 68, which is 21 + 47; off again writes 68 -> 21, one teardown and one rebuild each way. A blueprint taken over the result carries `variation = 68` for the flagged part and 17, 27, 35 for the other three; pasted elsewhere and revived, the cluster comes up `skin cluster=5 parts=4 set=0 vars=68,27,17,35`, which is `recoverPriority` finding the world already showing what it wanted and making no write at all. With it stubbed out the same paste comes up `set=4 vars=21,...` and the flag is gone.

### The three doors

A keybind (`bbb-toggle-priority`, ALT + P over the part the player is pointing at), a remote method (`set-part-priority`), and a settings paste. All three reach `setPartPriority` and nothing else does.

The remote method is not a convenience. A keypress cannot be issued from a script -- `script.raise_event` refuses a custom input outright -- and a headless run has no player to press one, so without it the whole feature is reachable by a human and by nothing else. It reaches the same function with no branch below it that can tell them apart except the player there is to tell, which is what makes a suite driving it evidence about the keybind.

ALT + P is free: a `--dump-data` of base, quality, elevated-rails and space-age on 2.0.77 has 14 custom inputs, twelve of them `ALT + <letter>` over A B C D E F G L R T U Y, one on TAB and one unbound. The engine's own compiled-in bindings are not prototypes and no dump lists them.

The settings paste takes the flag from the REGISTRY and not from the pasted variation, even though the engine writes that variation itself: measured on 2.0.77, a script `copy_settings` between two of these entities moves `graphics_variation` from the source to the destination. Reading the flag back out of it would work today and would rest on an engine behaviour nobody promised. `pvar` is cleared for the destination in the same breath, because the engine has written a picture behind the guest's back and restyle must look rather than compare against a memory that is now wrong.

**AND A REFUSED PASTE PUTS THE PICTURE BACK ITSELF**, which is the price of that clearing. `setPartPriority` restores the flag on every refusal, and the destination is still wearing the source's variation -- badged, if the source was flagged -- with `pvar` at 0; the next restyle hands it to `recoverPriority`, which reads a badge as a flag and raises one with no guard and no message. That is a hole straight through the spill guard and through the fit check: the player is told the change did not happen, and the flag lands a flush later on a cluster the compiler then refuses. So the handler writes the shape the registry says that part should draw and sets `pvar` to it, one host call on the entity the event already carried, and `recoverPriority` finds nothing.

### The four refusals

`plan.ShapeEdges` answers one question, can this be built, and three bounds sit behind it; the fourth is the guest's own and is about the moment rather than the shape. Each has one sentence for a player and one log line.

The order is the port cap, then a priority input, then a size, and it is the same order on both sides.

| | when | toggle | build |
|---|---|---|---|
| **over the port limit** | more than `MaxPorts` belts, whatever is flagged | `over-port-limit`, for a flag going ON; a flag coming off is allowed through | `over-port-limit` |
| **too big for a priority port** | the priority construction does not fit the slot, which is P = 64 | `priority-refused` | `priority-too-big` |
| **a priority input** | `QIn > 0` after the collapse, at any size | `priority-input-refused` | `priority-input` |
| **too full to shrink** | the successor could take back less than the balancer is carrying | `priority-holding` | unreachable: a build cannot make a standing network smaller |

Each build row has a second key with `-unconnected` on it, for the robot or script build that leaves the piece standing rather than handing it back, and `refuseShape` (`limit.go`) is the one place a refused `Ports` chooses. A cluster can break more than one bound at once. The port cap is named first because it is the only one taking a flag off does not fix; among the priority bounds a priority INPUT wins over a size, because there taking the flag off is the fix either way and it is the flag the player last touched.

**A TOGGLE IS REFUSED BEFORE THE FLAG MOVES.** `ShapeEdges` is asked with the flag speculatively flipped, so a shape the compiler could not build leaves the flag, the network and the items exactly as they were. Refusing after the flip would save the network too -- the check is in front of the teardown -- and would leave the cluster standing refused until the player guessed to toggle back.

**EXCEPT THAT TAKING A FLAG OFF IS ALWAYS ALLOWED WHERE THE OLD SHAPE DID NOT FIT EITHER.** The fit bound is about the SHAPE and not about the change, so at P = 64 every q >= 1 is refused -- and a balancer that arrived there with two flags on it, by growing past P = 32 with them already set, could then not have either one taken off: each single un-flag lands on q = 1, which is refused, so the machine stays refused forever while this file, README and `limit.go`'s own hand-back sentence all say that taking the flag off is the way out. The pre-toggle shape is free to ask -- `ShapeEdges` reads the edge counts and `shapeWithTileFlipped` flips a field in a list already in hand, so neither makes a host call -- and a change that cannot make things worse than they already are is not one to refuse. The spill guard is skipped on that path for the same reason the refusal leaves the network alone: nothing is torn down, so nothing can spill.

**Every-port-flagged collapses rather than refusing.** `ShapeEdges` reports `QOut = 0` for M flagged outputs and `QIn = 0` for N flagged inputs, so a player who ticks every port of a 64-port balancer gets the plain network to the byte instead of a refusal, and a balancer whose every INPUT is flagged is not an input-priority refusal at all. Two tiers where the second is empty is one tier.

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

The guest's half is log lines, and the suite drives every one of them through `remote.call("better-belt-balancer", "set-part-priority", surface, x, y, on)`, because the keybind needs a player:

| what | the line |
|---|---|
| the flag moved | `priority part=X,Y on` / `priority part=X,Y off` |
| the picture followed | `skin cluster=...` with the flagged part's cell 47 above its unflagged one |
| a shape too big | `alert: priority refused for cluster N at part X,Y: ... does not fit; the flag was not set` |
| a priority input | `alert: priority refused for cluster N at part X,Y: ... is a priority input, which this version does not build; the flag was not set` |
| too full to shrink | `alert: priority refused for cluster N at part X,Y: the balancer holds H items and the network this would build can take back C; the flag was not set` |
| a build reaching the same bounds | `alert: cluster N cannot be built with Q priority outputs over n->m ports; refused` and `alert: cluster N asks for Q priority inputs over n->m ports, which this version does not build; refused` |

**The spill guard's rig is a DEAD-ENDED 2->2 with one output flagged**, four parts under the one-belt-per-part rule, fed until it stops taking anything. `plan.Reinsertable` is 106 flagged and 64 plain, both pinned by `TestPriorityCapacityIsRecorded`, and the guard's comparison is `held <= 64`. So the leg is: fill it, clear the flag, assert the refusal line with `H` over 64, assert the network still standing and still delivering, and assert **zero items on the ground** over the whole window. Then unblock the outputs, let it drain past the boundary, clear the flag again and assert one teardown, one rebuild and no spill. The rig has to be dead-ended rather than merely saturated: a balancer that is moving anything has room, which is what makes the refusal rare in play and what makes it reachable here.
