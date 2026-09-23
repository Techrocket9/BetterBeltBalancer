# Stacked belts come back stacked — the gate that costs vanilla nothing

**The second half of the same defect, found by reading the first half's own caveat.** "A recompile is not a removal" fixed *where* a teardown's items go and recorded one thing it could not recover: **how they were STACKED**. The drain's instrument was `get_contents`, which returns totals per (name, quality), so `insert_at`'s `belt_stack_size` was passed absent and a Space Age player's 4-stacks came back as four times as many single items. Conserved, and wrong: a quarter of the density, a throughput dip until compression recovered — and, it turns out, **items on the floor**, because a network handed back four times as many belt positions as it had does not fit in itself.

**The kind key is (name, quality, stack size) now**, the drain reads `get_detailed_contents` — one `LuaItemStack` handle per belt POSITION — and the reinsertion passes the stack size it drained. `guest/go/carry.go`'s header is the long form and `compile.go`'s `detailedTally` is the read.

## The gate, and why the force bonus is the right one

**`LuaForce.belt_stack_size_bonus`, read once per force per carry transaction, in two host calls and no allocation** (an entity we are already holding for its force, and the force for its bonus). Zero — which is all of base Factorio and all of Space Age before the research — takes the path that was already there, byte for byte. Measured headlessly before any of it was written:

| probe | result |
|---|---|
| a loader prototype with `max_belt_stack_size` in **base** | **refused at load**: *"Belt stacking is disabled and can not be used. Belt stacking requires space-travel"*. The data-stage flag is `feature_flags["space_travel"]` — an underscore, where the error message has a hyphen |
| the same loader under Space Age, force bonus **0** | delivers **singles**. The bonus is what turns gameplay stacking on, and it only ever goes up |
| the same loader, force bonus **3** | 4-stacks, `get_item_count` 16 on a belt line that holds 4 positions |
| `insert_at(p, {name, count = 4}, 4)` on a **bonus-0** force | **places a stack of 4 anyway.** The bonus does not gate the API |
| `insert_at(p, {name, count = 9}, 4)` | places **4**, returns **true** — the call is atomic per POSITION and truncates the count to the stack size |
| `insert_at` onto an occupied position | **false**, and nothing is placed. So `ok` is the only return that has to be believed |
| `insert_at(p, ..., 255)` | accepted. The engine clamps nothing, so a drained stack is reproducible exactly; the uint8 parameter is the only ceiling |

The fourth row is what the gate does **not** cover and it is stated rather than hidden: a third-party mod that scripts stacks onto a bonus-0 force's belts gets them back unstacked. That is the same shape as the failure envelope above — conservation always holds, fidelity is best-effort, and the next audit re-reads the world.

## What the detailed read costs, and where it is not paid

Per **non-empty** transport line, above the gate:

| | host calls |
|---|---|
| below the gate (all of base) | `get_item_count` + `get_contents` — **exactly as before** |
| above it, line **not** stacked | + one `get_detailed_contents`, and **nothing per position**: `len(detailed) == get_item_count` means every position holds one item, so the flat totals are already the right answer and are taken |
| above it, line stacked, **one** item kind on it | + one `count` per position. The name and the quality come from the totals, so no string crosses the boundary and nothing is allocated |
| above it, line stacked, several kinds | + `name_is` per distinct name until one answers (a bool return over a string the guest already holds — never `name`, which would copy the host's bytes into the guest heap), and `quality` only to break a tie between two entries of the same name |

**And the reinsertion got CHEAPER, which is the opposite of what the pass expected.** `insert_at` is atomic per position, so a group of 24 items in 4-stacks costs six calls where the old code made twenty-four. Measured on the `plat` suite's own profiler — the audit-forced recompile, medians of three interleaved runs, each minus that run's own `audit only, nothing pending` control (7.56 / 7.37 ms):

| recompile of a **stacked** rig | stack-aware | flat (the gate forced off) | |
|---|--:|--:|---|
| `full`, a dead-ended 4×4 holding 928 items | **19.63 ms** | 33.59 ms | **1.71× faster** |
| `flow`, a running 4×4 | **11.77 ms** | 19.02 ms | 1.62× faster |
| `plain`, 2 parts, **unstacked lines above an open gate** | 6.58 ms | 6.30 ms | **+0.28 ms** — the one cost |

The last row is the whole price of the gate being open on a line that turns out not to be stacked: one `get_detailed_contents` per non-empty line. Everything else in the table is a win, and the flat arm also **spilled 320 items onto the ground** in the same run, because 928 items reinserted as 928 positions do not fit in a network that held them in 232.

**That table was measured on a FOUR-band save and the suite has five bands now**, so it is kept as measured and is not comparable with anything taken since. The profiler window is around the AUDIT rather than around a tick pair — it has to be, because the sample either side has to be atomic — so it carries a whole-save re-classification, and adding a band makes every cell bigger including the control's. Measured 2026-08-05, medians of three, same method: the `audit only, nothing pending` control is **10.85 ms** against 7.56, and net of it `full` is **26.92 ms**, `flow` **14.35**, `plain` **7.66** and the new `smix` **12.46**. Same ratio to the control, one more band under it.

**Nothing base-only moved, and that is measured rather than asserted.** The `mar` suite's seven slopes under `-gc=leaking` came out **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — and the M2 recompile timings, the `edge` suite's medians and every rate in every suite are unchanged.

## The leg, and why it is in `plat`

**Belt stacking is a Space Age feature at the PROTOTYPE level**, so no base-only suite can build a stacked belt at all — the first probe above is the load error that says so. The leg therefore lives in the `plat` suite, which is now the **Space Age suite** rather than the platform one: two legs with the DLC in common and nothing else, because a second DLC-only suite would cost a second Factorio run for one rig.

It builds on its own flat scratch surface on its own force (`bbb-stack`, `belt_stack_size_bonus = 3`), so **one save exercises both arms of the gate** — the platform leg's rigs are on `player`, whose bonus stays 0. Four bands: a stacked control belt, a dead-ended stacked 4×4 (`full`), a running stacked 4×4 (`flow`), and `plain` — the same stacking force fed by an ORDINARY loader, which is the branch that opens the gate and then finds nothing stacked.

**Every band is two columns of parts since the 2026-08-24 single-edge re-lay, plus one EDGELESS part on the end**, and that last part is what the recompile's belt lands against: under Factorio 2.1's rule a working balancer has no free face, so the belt that used to arrive on the block's north face would be refused rather than compiled and the band would measure a refusal. Every number in the table below is unchanged by that, to the item, which is the expected result rather than a lucky one — a hidden network is a function of the BELTS and the belts did not move.

The assertion is on the stack PROFILE of the hidden surface, sampled either side of a forced recompile inside one tick (`assert-plat.py`). Conservation was never the defect and is not the headline:

| | measured |
|---|---|
| stacks really formed first | 1,128 items over **336 belt positions — 72 of size 1, 264 of size 4**. A run where stacking silently did not happen fails as vacuous |
| `full`, a stacked 4×4 recompiled | 10,952 items before and after; the single-item positions moved by **exactly 0** and the 128 items that crossed into the network are **all** in the stacked total |
| `plain`, unstacked under an open gate | the exact opposite, exactly: **+16 singles, +0 stacked**, and no stack invented |
| `flow`, recompiled while stacked items are moving | +0 singles, **+48 all stacked**, and over the 500 ticks starting 200 after its own recompile it delivers **1496 1488 1492 1496 against one saturated stacked belt's 1504 — 3.971×, spread 0.54%** |
| spills | **0** across all three recompiles |

**The leg was verified to fail on the unfixed guest**, which is the only thing that makes it evidence: with the gate forced off, six assertions fire — 572 more single positions on `full`, 304 more on `flow`, 320 items on the ground, and the throughput falls to **3.884×** with the spread at 1.23%. **Re-run 2026-08-05 with the fifth band in the save it is nine assertions and 392 items on the ground**, the extra three and the extra 72 being `smix`'s; every one of the original six fires with the number it was recorded with. **Re-run again 2026-08-24 against the single-edge bands it is the same nine and the same 392**, with 572 and 304 to the position and `smix`'s 56×4 histogram becoming 196×1 — and per-kind conservation staying EXACT, because the gate-off path is still conservation-correct and only unstacks.

Two things about the numbers. The `plat` timings carry a whole-save re-classification as well as the recompile (the audit marker is what makes an atomic before/after sample possible at all), so they are comparable **only** with each other and with the same measurement from another build — never with M2's tick-pair recompile figures. And the stacked window is measured from 200 ticks after the recompile rather than from the recompile itself, because a rebuild puts every drained item back at the HEAD of the butterfly and the outputs are briefly starved by construction — the same shape the `edge` suite measures the same way.

**What it costs to ship**, same flags, same pin, measured either side of the change: `fk_module.lua` **2,046,296 → 2,112,476 B (+3.2%)**, the zip **273,917 → 278,309 B (+1.6%)**, `dist/bbb.wasm` 942,882 → 957,817 B. That is a per-load handful of kilobytes for a path most players never enter, and it is the only cost on this side of the scale.

The mod ships **36 bound members** now, up from 28: `get_detailed_contents`, `LuaItemStack`'s `count` / `name_is` / `quality`, `LuaQualityPrototype.name_is`, `LuaEntity.force` and `LuaForce.belt_stack_size_bonus`. No new FkLua gap; every one of them was reachable through the generated bindings as they stood.

## Stacked sushi — the three branches `kindAt` has, and the two nobody had run

**`detailedTally` reads a stacked line position by position and `kindAt` says which (name, quality) total each position belongs to. It has three branches and exactly one of them had ever executed, in any run, on any machine.** Closed 2026-08-05 by the `plat` suite's fifth band; no guest line changed, which is the point — this is the measurement the path never had.

| `kindAt` branch | what it needs | before |
|---|---|---|
| `len(totals) == 1` — no host call at all | a stacked line carrying **one** kind | every stacked rig in the repo |
| the `name_is` loop over candidates | a stacked line carrying **two names** | **never run** |
| the `quality` tiebreak, plus `askedAlready` | a stacked line carrying **one name at two qualities** | **never run** |

**Why eight suites could not reach it, and it is a two-sided gap rather than a missing rig.** Multi-kind rigs live in `mix`, which is base-only — and below the stacking gate `drain` uses the flat totals and **`detailedTally` is never called at all**, so `mix` cannot reach `kindAt` however many kinds it runs. Every rig above the gate is Space Age and single-kind iron plate, so it takes the cheap branch. The two conditions have to be met by ONE line at ONE moment, and only Space Age can do it.

**The rig.** `smix` — a dead-ended 2→2 on the stacking force, fed by two **stacked sushi** sources: `bbbt-stackloader` behind an infinity chest holding ONE filter at a time and rewriting it every four ticks with `remove_unfiltered_items` on. That is `mix`'s own technique and it is a measurement rather than a preference: the naive rig, one chest carrying six filters at once, delivers a single kind (2,292 items, electronic-circuit 100%), because a loader draws from the first stack it finds and the chest tops that same stack straight back up.

**Six names none of the other bands touch**, so the four iron-plate bands keep their recorded figures to the item while this one is measured independently — and one of them, `plastic-bar`, is on the rotation **three times over, at normal, uncommon and rare**, consecutively, so two qualities of one name land on one line. `quality` is an optional key of `InfinityInventoryFilter` in the pinned 2.0.77 runtime API and the `quality` mod is already loaded here, so the tiebreak costs no new dependency.

**The rotation period is a measured constant, not a preference.** A hidden belt tile holds four item positions per lane and a stacking loader at express rate emits ~3 items/tick over two lanes, so four ticks is ~1.5 stacked positions per lane — comfortably shorter than a line, which is the whole condition. A period longer than a line gives single-kind lines and the band would pass every assertion while exercising nothing, so the suite **samples the world and fails if the shape did not land** rather than assuming the arithmetic.

Measured 2026-08-05, Factorio 2.0.77, shipped config:

| | measured |
|---|---|
| **anti-vacuity, at the instant the teardown read them** | of 24 hidden lines carrying this band, **14 carried two or more NAMES and all 14 had a stacked position**; **6 carried one name at two QUALITIES**, all 6 stacked; biggest stack 4. Both `kindAt` branches reached, by measurement |
| kinds in flight | **9** of 9 (name, quality) pairs, 704 items — inside the carry pool's 32-group bound by design, since overflowing it is `mix`'s job and here it would spill |
| conservation across the recompile | **EXACT per (name, quality)**, all nine: copper-cable 120, copper-plate 80, electronic-circuit 112, iron-gear-wheel 80, plastic-bar **normal 40 / uncommon 80 / rare 32**, steel-plate 40, stone-brick 120 |
| the stack profile | **64 items crossed, +0 single and +64 stacked** — 56 positions of 4 became 72 of 4, and not one position of 1 was invented |
| spills, and pool overflows | **0** and **0** |
| the network | a 2→2 recompiled to 3→2 over 4 ports, 28 entities, **288 items handed back** |

**Red-proven twice, and the two proofs catch different things — which is the result rather than a formality.**

| injected defect | what fired |
|---|---|
| **the stacking gate forced off** (`stacksPossible` returns false), the established proof for this leg | the three PROFILE assertions: **+196 single positions** where the fixed guest adds none, the 56×4 histogram becoming 196×1, and the crossing going negative. **Per-kind conservation stayed EXACT** — the gate-off path is still conservation-correct, it just unstacks |
| **`kindAt` always answers candidate 0**, so every position is attributed to the line's first total | **only per-kind conservation**, and it named seven of the nine kinds: copper-cable 120→88, copper-plate 80→104, electronic-circuit 112→96, plastic-bar normal 40→64, **rare 32→16, uncommon 80→64**, steel-plate 40→24, stone-brick 120→168. The stack profile was **byte-identical to the healthy run** (+0/+64), the item TOTAL was unmoved at 704, and every other band in the suite passed |

The second row is why the counting is per (name, quality) and not per name and not a total. A misattributing `kindAt` hands back the right number of items under the wrong key: a single total sees nothing, a per-NAME total would still have missed the plastic-bar rows, and the stack profile — the leg's existing instrument — cannot see it at all. The two proofs together say the band's assertions are not redundant with each other.

**It costs one Factorio run of nothing.** No guest line moved, so the `mar` slopes came back identical to the byte (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) and the shipped zip did not change. Inside the suite the save gains a band: the four bands above are unmoved to the item, `plat`'s profiler control rises 7.56 → 10.85 ms for the reason the timing note above gives, and the recompile count the suite requires goes 3 → 4.
