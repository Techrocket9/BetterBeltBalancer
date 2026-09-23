# The wake race — what a guided playtest found that nine suites could not

**A guided interactive playtest on 2026-08-05 walked the staged gestures against a real graphical client, and every defect it found is one no headless run can reach.** Nine suites were green throughout and always would have been: a `--create` has no players at all, so `player_index` is zero on every build and every mine any suite can produce, and the whole feedback half of "The sixty-fifth belt" — the flying text, the sound, the hand-back — sits behind the same wall as the miner's pocket's trigger. Everything it found is the table at the end. The first item needs a section of its own, because it is not about words: it is about the one dispatch in which this guest is awake and knows nothing.

## The wake, and the gesture that opens it

**`make install` over a mod of the same version is a silent guest swap**, and it is what an author does twenty times an afternoon. Factorio raises no `on_configuration_changed` for it — nothing was added, removed or re-versioned — so neither `fk_migrate` nor `fk_on_configuration_changed` fires. The packaged guest is a different BUILD, though, so the saved heap is declined and the mod comes up on a fresh one: an empty registry over a world full of parts and full of running networks. That is exactly the state "Coming back on a heap this build did not write" is about, and `registryReady` is the fallback that covers it — false in a freshly initialised heap, so the FIRST EVENT of the session rebuilds the registry from the world before it decides anything (`ensureRegistry` at the top of `onEventBody`, main.go).

**What nobody had asked is what happens when that first event is a PLAYER BUILD that takes a balancer past the limit.** Then the rebuild and the build are ONE dispatch, in that order, and the rebuild goes first with information the event has not handed over yet:

> `rebuildFromWorld` runs its own `flush()` — it has to: a rebuild that merely queued would leave every cluster it could not adopt uncompiled until a tick arrived, and in a `--create` none ever does. That flush compiles the cluster the player's new piece just made over-limit, and `buildNotes` is still EMPTY, because `noteBuiltByPlayer` is called from `onPart` and `onNeighbour`, and neither has run.

lifecycle.go's own comment is the shortest statement of the problem: the rebuild "judges the world with the worst information a refusal will ever have".

## The first head — the rebuild spoke, and it was wrong two ways

Both symptoms are one window seen twice, and lifecycle.go records them in the order they were reported:

| | what the player saw | what did it |
|---|---|---|
| **first** | nothing at all: the piece stayed, no text, no sound, no hand-back | the rebuild's refusal armed the feedback memo, `overLimit[root] = fp`. The informed flush a tick later found `prev == fp` and returned before it said or handed back anything — the gate doing exactly its job, to the one refusal that had not yet been delivered |
| **then** | a chat line saying the extra piece was **left in place, unconnected** — one tick before the piece arrived in their inventory | with no build note beside the cluster, `tellOverLimit` takes the `told == 0` fork, which is the ROBOT sentence and a `force.print`. The informed retry then did the player thing and handed the piece back, so the player was told one thing and given another |

**One branch cures both, and it is the first thing `refuseOverLimit` does.** `rebuildingFromWorld` is up for exactly the span of `rebuildFromWorld`, including the flush it runs itself (lifecycle.go sets it before `collectSurfaces` and clears it after `flush()` returns). A refusal issued under it **logs, requeues and tells nobody**:

- it **logs**, through the same `logRefusedOverLimit` the ordinary path uses, so the log stays a complete record of every refusal and the `edge` suite's line is the line it always was. The gate is on the MESSAGE, not on the log;
- it **does not arm the memo**, so nothing suppresses the informed retry;
- it **does not speak**, so nothing claims a final state one more tick can falsify;
- it appends the root to `rebuildRefused` — **not `markLive`**, and that is the same after-the-drain discipline `revertOverLimit` follows for its mine: this runs inside the rebuild's own `flushLive` drain, whose loop has captured its length and whose tail truncates the queue to `[:0]`, so an append there would be silently erased. `rebuildFromWorld` requeues them AFTER its flush returns and asks for another one.

The next ordinary flush then re-judges with whatever notes the rest of the dispatch recorded and delivers the one correct message. **A refused cluster in a save nobody is editing reaches that flush with no notes and gets the piece-stands message, which is then true.**

## The second head — a build event the registry has already seen

Found on the staged bridge gesture (main.go cites it as *the bridge gesture*), and it is what makes the requeue above worth anything.

**The engine places the entity before it raises the build event**, so when that build event is the first of the session the rebuild at the top of the same dispatch scanned a world that already contained the part — and registered it. `AddPart` therefore reports **no change**, and `onPart` used to return right there: no build note, no queue insertion. The cluster the rebuild had requeued would then be re-judged with nothing beside it, take the `told == 0` fork again, and the over-limit piece would stand with nothing but a chat line.

`onPart`'s `else if id, dup := index[k]; dup && builtBy != 0` branch is the fix: record the note, `markLive(find(id))`, `logState()`, ask for a flush. Three things make it safe outside the window it was written for, and all three are the code's own:

- **outside the wake race a duplicate PLAYER build event is a mod raising `built` twice for one entity**, and what it costs then is a note plus a flush that skips on the fingerprint;
- **`builtBy` is 0 for scripts and robots**, so the branch is not entered for them at all — and their wake-race outcome (the piece stands, the force is told) is the designed one already;
- `noteBuiltByPlayer` drops exact duplicates, so a note recorded twice is one note.

**The belt at the edge needs no second head**, and the asymmetry is worth knowing before anyone tidies it: `onNeighbour`'s gate is PROXIMITY, not novelty. It walks the 5×5 neighbourhood, queues every cluster it touches and records the note whenever `builtBy` is non-zero, whether or not anything about the registry changed. Only the part path gates on `AddPart` having done something.

## The principle: the rebuild never speaks, only informed flushes do

That is the durable half of this pass, and it is worth stating on its own because the next thing that wants to talk to a player will be written somewhere the same argument applies. `rebuildFromWorld` reconstructs a whole session's registry from a world it has never seen, inside one dispatch, with none of that dispatch's own events yet delivered. Any verdict it reaches about a PLAYER is provisional by construction. So it may log, it may queue, and it may not address anybody — the flush that has the notes is the one that speaks, and there is always one, because the rebuild requeues what it refused.

## What is verified, and what is behind the wall

**The wake race itself needs a player AND a same-version reinstall**, which is two things a headless run does not have and one of them is not even about players: every suite's two phases run one mod set built once, and the `mig` suite — the only one that changes a mod set at all — changes a NEIGHBOUR's, which is precisely the case that DOES raise `on_configuration_changed` and therefore never reaches this window. So the trigger is on the interactive side of the wall with the flying text, the sound and the hand-back it is about. `test/interactive/README.md`'s gestures C and D are the two gestures it happens to; the wake race is either of them done as the FIRST thing in a session whose guest was swapped under it, which no staged rig can set up for anybody.

**What the suites pin is the negative, and it is the negative they already had.** `player_index` is zero on every build any suite can produce, so `noteBuiltByPlayer` returns before it appends and NEITHER HEAD CAN BE ENTERED: the `edge` suite's **zero pieces handed back over the whole run** still holds, a revert firing for a script build still fails the run, and the `lim` and `brdg` legs' "exactly one `alert:` per edit" still means what it always meant. What would show up in a suite if the rebuild path ever did start speaking is a second `alert:` line, since the rebuild path logs ungated by the memo — one refusal from the rebuild and one from the informed retry is the expected shape of a wake-race refusal in a log, and it is one refusal as far as the player is concerned.

## The rest of what the playtest found, and where each fix lives

| what it found | where the fix is |
|---|---|
| **The flying text was at the box centre.** A player laying the sixty-fifth belt at the top of the 32-part `lim` column "got the sound and never saw the text, because it spawned seventeen tiles south of their screen" | `tellOverLimit` (limit.go) moves it to the refused piece's own tile — "the one tile their eyes are guaranteed to be on" — taken from the first build note beside the cluster in EVENT ORDER, which is the same tie-break `carry.Claims.BeneficiaryFor` uses for a mine. The box centre is still computed above the loop, and the only branch that reads a position is the one a note has already moved |
| **The text was WHITE**, which is the default and "reads as information rather than refusal"; the check "expected red flying text, because that is what the base game has taught everybody a refusal looks like" | `limColor` (limit.go), 1.0 / 0.25 / 0.25 — "vanilla's cannot-build red, near enough" |
| **"would need 128 ports" read as gibberish.** A player placed one belt and has no window into the compiler | the message speaks in **BELTS PER SIDE**: `over-port-limit` takes the limit as `__1__` and `max(pt.N, pt.M)` as `__2__`. The LOG line keeps P — "it is for whoever reads logs, and ports are its native unit" — so the `edge` suite's numbers (128 ports for 65 inputs) are untouched |
| **The message narrated the hand-back**, and the round-trip is invisible at game speed: to the player this is a placement that simply failed, and a sentence about an inventory transaction they never saw "reads as a transaction to go looking for" | it deliberately does not say so any more. State the rule, like vanilla does, and stop (locale comments on `over-port-limit`) |
| **`Unknown key: "entity-name.bbb-linked-belt"`** in the engine's own *X is in the way*, which fires once per colliding entity and therefore names the invisible edge interfaces standing on a part's tile | `[entity-name]` entries for all four hidden prototypes, all reading **`Balancer`** — "the player sees one machine, so anything of ours in their way IS the balancer as far as they know". They have no player-facing surface of their own and this is the one place the ENGINE names them anyway |

## The one report that was not a defect

**Fresh spill appearing beside a balancer that had just recompiled**, and the mod placed nothing on the ground. If a balancer compiles on top of ground items — only possible where something else has already littered those tiles — the engine relocates them during entity placement: items on the part tiles are absorbed into the new interfaces' transport lines and ride through the machine to the outputs, conserved, and whatever exceeds the lines' capacity is nudged to the nearest free ground, which is the edge of the existing litter. That reads as a spill at the litter's frontier and it is not one.

**Measured by diffing the two autosaves that bracket the recompile**, which is the method the whole playtest used to keep an observation from becoming a report: **204 plates left the part tiles, 89 landed on the frontier ring, every item name conserved**, and the guest's log carries no spill line in that window. `test/interactive/README.md` keeps it under "A false alarm to recognise" so the next person to see it does not go looking for a leak.

## What it costs

**Nothing on any path an ordinary session takes, and it is structural rather than measured.** `rebuildingFromWorld` is one bool, tested only inside a refusal — which is a path a save that never hits the cap never reaches. `rebuildRefused` is "nil forever in a session that never rebuilds over a refused cluster, which is every session of ordinary play", and the requeue behind it is a length test on a nil slice once per rebuild, which is once per session. The second head costs one `index` probe on the branch where `AddPart` reported no change — a duplicate build, which is not a thing that happens on a hot path — and no host call anywhere. Nothing was added to the compile path, the neighbour gate or the flush proper, and no member, define or prototype was bound for any of it.

<!-- ============================ FAST REPLACE ============================ -->
