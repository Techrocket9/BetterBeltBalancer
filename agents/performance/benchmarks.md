# Benchmarks

`bench/` is a headless harness: it builds N saturated balancer rigs into a save via a setup mod's `on_init`, `--benchmark`s it, verifies items actually moved and the outputs are balanced, and appends a row to `bench/baselines/results.tsv`. Usage and method: [`bench/README.md`](../../bench/README.md). `bench/matrix.sh` regenerates the whole matrix (~3 min). Third-party mod zips are read from `$BB_MODS_SRC`, never committed. **`MEGA=1 bench/matrix.sh` runs a different matrix**: one heterogeneous save of 404 balancers over ten shape classes, including the only 16x16, 32x32 and 64x64 this project has ever built in a real game. See "The megabase cell" at the end of this section.

**EVERY NUMBER IN THIS SECTION WAS MEASURED ON FACTORIO 2.0.77 AND THIS MACHINE IS ON 2.1.17, so the whole section is HISTORY until somebody re-runs the matrix.** Two things found on 2026-08-25, when phase 7 of the estate port went to take a comparability baseline and could not:

- **The harness did not run at all on 2.1.** The setup mod's `info.json` said `factorio_version: 2.0` and 2.1 refuses such a mod at the LOADER, exactly as it refused every suite before `test/run.sh` learned to stamp. `bench/run.sh` stamps the staged setup mod now, the same perl over the same two fields.
- **`--benchmark-verbose` SIGSEGVs on 2.1.16**, after emitting the `t0` row, with any counter list — measured on a VANILLA NO-MOD SAVE as the control, so it is the engine's defect and not ours. So `whole_us`, `belts_us`, `entity_us` and `script_us` cannot be obtained on this engine at all, and neither can the per-tick `wholeUpdate` column this file sends every worst-tick question to. `run.sh` skips the pass on 2.1 with the reason printed rather than crashing Factorio once per cell (the crash also leaves a reporter process holding the run's own `.lock`, which fails the NEXT cell on something unrelated); `BENCH_VPROF_FORCE=1` is how a future engine gets re-tested.

**AND THE SETUP MOD IS A COMPILED GUEST since 2026-08-25** (`guest/go/obs/bench`), which moves the harness's own absolute milliseconds and no delta it publishes. What that cost, measured interleaved against the Lua it replaces, is in [`agents/estate-port.md`](../estate-port.md)'s phase-7 section.

Baseline to beat — Factorio 2.0.77, base only, M3 Pro (full tables: [`bench/baselines/BASELINE.md`](../../bench/baselines/BASELINE.md)). **The bar is belt-balancer-3 v1.0.1** (the live successor; user decision 2026-08-01), but bb2 v2.0.9 is *faster* than bb3 on every cell — bb3's `lane.valid` crash guards are extra boundary crossings — so bb2's numbers are the operative target and bb3 falls with it. Correctness bar: both deliver exactly full belt throughput with an exact per-output split — match that first.

**M4 is done and the bar is cleared by 22–67× on every saturated cell.** Full head-to-head, method and caveats: [`bench/baselines/RESULTS.md`](../../bench/baselines/RESULTS.md). Marginal cost of one 4×4 balancer, per tick, all four columns measured back to back in one session (`MODS="bb2 bb3 bbb" bench/matrix.sh`):

| per saturated 4×4 balancer, per tick | bb2 | bb3 | **bbb** | vs bb2 |
|---|---|---|---|---|
| express, whole tick | 21.9 µs | 23.1 µs | **0.49 µs** | **45×** |
| express, mod Lua only | 19.1 µs | 21.0 µs | **0 (= the control)** | — |
| normal, whole tick | 7.55 µs | 7.67 µs | **0.35 µs** | 22× |
| idle express | 1.70 µs | 2.91 µs | **0.16 µs** | 10.6× |

200 express 4×4 balancers: **0.64 ms/tick against bb2's 4.92** — 4% of the 16.67 ms 60-UPS budget instead of 30%. 500 of them cost 2.05 ms/tick and still deliver the control's exact item rate. **`scriptUpdate` is the no-mod control's in every cell** and there are zero `[BBB]` log lines inside any benchmark window: all compiling happens in `--create`, which is the whole architecture in one measurement.

Two things the numbers say that are not wins:

- **`avg_ms` timings on this machine drift 25–35% between sessions** (Spotlight indexing `bench/tmp` was worth most of that). Only cells measured back to back with their own control may be compared; `BASELINE.md`'s absolute milliseconds and `RESULTS.md`'s are from different sessions and are not interchangeable.
- **~~We have a GC tail the incumbents do not~~ — FIXED, and it was ours.** ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 17, and see "The heap diet" above for the whole pass.) M4 measured a 27.8 ms worst tick at n=200 idle express against bb2's 7.4 and the control's 4.4, with 16.75 of a 17.0 ms tick in `luaGarbageIncremental` and `scriptUpdate` at exactly zero. Two milestones then went into attributing it: M4 blamed `--persist=table`, the persist pass proved the mode was irrelevant (`packed` had the same tail), and the round-5 answer is that **the pause is 0.2 ms per MiB of guest linear memory and our linear memory was 64 MiB of dead log-line strings.** It is now under 16 MiB and the worst tick is **2.26 ms against the control's 1.42** — below bb2's 7.4 and bb3's 3.0, on the one axis where this mod used to lose. The GC *mean* was already 5× cheaper than bb2 (48 µs/tick vs 260) and still is. What remains open is a residual of ~1.3 KB per recompile in generated binding return values; it is quantified in the heap-diet section and it is upstream's, not ours.

## The round-2 re-check: did filters and batching cost anything?

They could have. Both changes touch the event path, and the M4 claim that must survive is not a millisecond — it is **zero script in the steady state**. Two cells re-run, each against its own control **in the same session**, n=200 k=4 express (`bench/run.sh --mod bbb` / `--mod none`):

| | control | **bbb** | Δ per balancer |
|---|---:|---:|---:|
| saturated, `scriptUpdate`, run 1 | 1.72 µs | **1.94 µs** | 0.001 |
| saturated, `scriptUpdate`, run 2 | 1.84 | **2.25** | 0.002 |
| idle, `scriptUpdate` | 1.54 | **1.53** | **below the control** |
| saturated, `wholeUpdate`, run 1 | 506.18 | 615.39 | 0.55 |
| saturated, `wholeUpdate`, run 2 | 508.44 | 633.07 | 0.62 |
| idle, `wholeUpdate` | 181.36 | 194.20 | 0.06 |

**`[BBB]` log lines inside every benchmark window: 0**, on every cell, as before — and the create log confirms why: 200 `compiled cluster` lines, all of them during `--create`.

Read with the session-drift caveat above, which is the whole reason each row carries its own control. `avg_ms` marginal cost came out **0.33 and 0.73 µs/balancer** on the two saturated runs against M4's 0.42 — that spread is the GC pauses averaged over the run, not a change in the event path, and M4's own four measurements of the same cell spanned 0.6445–0.6995 ms. The idle cell moved the *right* way (0.07 µs/balancer against M4's 0.16) and its worst tick came out 18.3 ms against M4's 27.8, which is encouraging and is **not** claimed as a fix: same machine, different session, and the GC tail is a property of heap size.

The honest summary is the first three rows. `scriptUpdate` is still the control's, in both directions, on a guest that now runs *less* code per event than the one M4 measured.

## The round-4 re-check: the mask, the defines and `game.surfaces`

Same two cells, same in-session controls, n=200 k=4 express, 3600t × 2:

| | control | **bbb** | Δ per balancer |
|---|---:|---:|---:|
| saturated, `scriptUpdate` | 1.98 µs | **1.72 µs** | **below the control** |
| idle, `scriptUpdate` | 1.25 | **1.39** | 0.0007 |
| saturated, `wholeUpdate` | 476.95 | 467.25 | **below the control** |
| idle, `wholeUpdate` | 160.58 | 168.98 | 0.04 |
| saturated, `avg_ms` | 0.4885 | **0.4665** | **below the control** |
| idle, `avg_ms` | 0.1755 | 0.1865 | 0.00006 |

**`[BBB]` log lines inside every benchmark window: 0** — both `run.log` and `verbose.log`, both cells, counted directly rather than relying on the harness's "only prints when non-zero" line. Throughput 1,740,000 items at balance 1.001.

Three of six rows landing *below* the control is what a marginal cost near zero looks like when session drift is ±5%; it is not a claim that the mod makes Factorio faster. The reading that matters is the one that has held since M4: the steady state runs no Lua of ours, and nothing in this round changed that, because nothing in this round touched the steady state — the mask and the defines are event-path and load-path costs, and the surface walk happens once per session. Worst tick is still the GC tail (17.9 / 18.0 ms against the controls' 1.6 / 1.4), unchanged and unrelated.

## The round-5 re-check: the heap diet, and the last row that was still red

Same two cells, same in-session controls, n=200 k=4 express, 3600t × 2. This is the round that made the last row of the previous three tables stop being a regression:

| | control | **bbb** | Δ per balancer |
|---|---:|---:|---:|
| saturated, `scriptUpdate` | 1.72 µs | **1.61** | **below the control** |
| idle, `scriptUpdate` | 1.23 | 1.36 | 0.0007 |
| saturated, `wholeUpdate` | 506.41 | **460.70** | **below the control** |
| idle, `wholeUpdate` | 167.36 | **164.46** | **below the control** |
| saturated, `avg_ms` | 0.4760 | **0.4510** | **below the control** |
| idle, `avg_ms` | 0.1710 | **0.1707** | **below the control** |
| saturated, **worst tick** | 1.67 ms | **2.43 ms** | was 21.1 at M4 |
| idle, **worst tick**, median of 3 | 1.42 ms | **2.26 ms** | was 27.8 at M4 |

**`[BBB]` log lines inside every benchmark window: 0** — both `run.log` and `verbose.log`, both cells, counted directly. Throughput 1,740,000 items at balance 1.001, unchanged. The create log carries 200 `compiled cluster` lines and 200 `skin cluster` lines, as it always has.

Nothing in this round touched the event path, the compiler or the steady state, so the first six rows are the same statement they have been since M4 and the scatter around the control is session drift. **The last two rows are the round**: worst tick was the one column where this mod lost to both incumbents and to the control, and it now sits within a millisecond of the control on both scenarios.

## The round-6 re-check: the upstream gc round, and the row nobody had re-read

Same two cells, same in-session controls, n=200 k=4 express, three interleaved reps, 2026-08-03. Run because FkLua landed a `fkgc` fix that this mod's `gc.go` was on the wrong side of; the fix turned out to cost this mod nothing and the re-run found something else.

**Read the per-tick columns for this round, not `avg_ms`.** The collected arm carries a ~985-tick post-load transient that `avg_ms` averages into the whole window, so the harness figures below are what they are and the steady-state numbers underneath them are the claim:

| | control | **bbb, collected** | |
|---|--:|--:|---|
| saturated `avg_ms`, median of 3 | 0.4860 | 0.8180 | transient included |
| idle `avg_ms`, median of 3 | 0.1770 | 0.4125 | transient included |
| **saturated `scriptUpdate`, steady median** | 0.21 µs | **0.3 µs** | the claim |
| **idle `scriptUpdate`, steady median** | 0.21 µs | **0.2 µs** | the claim |
| **saturated `wholeUpdate`, steady median** | 467.9 µs | **470.8 µs** | |
| `[BBB]` lines in any benchmark window | — | **0** | counted directly |
| throughput / balance | 1,729,600 / 1.000 | 1,740,000 / 1.001 | |

**The leaking arm is the control on the round itself**: `script 1.36 µs`, `whole 169.20 µs`, `avg 0.1795` — the round-5 numbers for that cell to the decimal, which says nothing in the guest, the regenerated bindings or the packaging moved. The transient, why it is 985 ticks against a recorded 71, the three A/Bs that rule out this round, and what it does and does not mean for the `-gc` decision are in **"The transient that grew while nobody was looking"**.

## The megabase cell — 4,376 hidden splitters, and the first 64x64 anyone built

**Every benchmark above this line is 200 copies of one 4x4.** A megabase is not that, and the two things this repo had never measured are the two a megabase is made of: a MIX of shapes, and shapes past `P = 8`. The `mega` scenario builds both — ten shape classes per block, plus a 16x16, a 32x32, a 64x64 (which is `plan.MaxPorts` exactly) and a deliberately over-limit 65-input cluster. [`bench/README.md`](../../bench/README.md) documents the population and the two log families it adds; this section is what it measured.

Measured 2026-08-05, Factorio 2.0.77, base only, shipped configuration (`--persist=packed --gc=collected`), M3 Pro, nothing else running:

```sh
BENCH_TMP=/private/tmp/bbb-mega BENCH_VPROF_TICKS=3600 MEGA=1 REPS=3 bench/matrix.sh
```

**404 rigs, 5,938 parts, 404 clusters, 4,376 hidden splitters** (2,504 `bbb-splitter` + 1,872 `bbb-lane-splitter`) on a 152x408 surface. Three reps of each of the four cells, interleaved `bbb / control` per scenario so session drift scales the group together; per-tick `scriptUpdate` and `wholeUpdate` read from the verbose pass's own columns, the post-load transient **excluded** from every steady figure and reported on its own below, meter ticks dropped from the medians and kept in the worst tick where they cancel.

| n=40 blocks, express, steady state | control | **bbb** |
|---|--:|--:|
| saturated `scriptUpdate`, median tick | 0.21 µs | **0.29 µs** |
| saturated `wholeUpdate`, median tick | 774.56 µs | 909.60 µs |
| saturated worst tick | 3.279 ms | **3.361 ms** |
| idle `scriptUpdate`, median tick | 0.17 µs | **0.21 µs** |
| idle `wholeUpdate`, median tick | 164.25 µs | 167.96 µs |
| idle worst tick | 2.726 ms | **1.976 ms** — *below the control* |
| saturated `avg_ms` (harness, transient INCLUDED) | 0.8145 | 0.9485 |
| idle `avg_ms` (same) | 0.1895 | 0.2240 |
| `[BBB]` lines in any benchmark window | — | **0** |

**Read the medians above, not the TSV's `script_us`.** That column is a MEAN over the verbose pass's steady half, and this save's meter drains 4,404 sink chests six times a run at ~1 ms a time — which averages to ~3.4 µs of every tick and is where the TSV's `script_us` of 3.44 (control) and 3.60 (bbb) comes from. It is the harness's own cost, it is the same on both sides, and it swamps a median tick of 0.2 µs. The rows are in [`bench/baselines/results.tsv`](../../bench/baselines/results.tsv) tagged `mega r1..r3`.

**The headline claim holds at 4,376 splitters, and it holds literally.** Steady `scriptUpdate` is 0.21 against 0.29 µs saturated and 0.17 against 0.21 µs idle — both sides of both pairs sub-microsecond, against a control with **no balancer mod loaded at all**. 404 finished balancers run no script, the same as 200 did and the same as one does. The marginal whole-tick cost is `(0.9485 - 0.8145) / 404` = **0.33 µs per balancer**, against the uniform 4x4 cell's 0.49 — lower because most of a block is 2->2s and 3->3s.

**The post-load transient, which is what this create is FOR.** A `--create` compiles all 404 networks inside one `bbb-audit` dispatch where no paced step can run, so the collection lands on the first ticks after the load: this is the paced collector's documented worst case, at twice the network count the mode decision was priced on.

| | measured |
|---|---|
| ticks until `scriptUpdate` returns to the control's | **39**, in all six bbb passes — no spread at all |
| that transient's total `scriptUpdate` | **74.1 ms** saturated (72.6–76.3), **69.6 ms** idle (67.6–75.3) |
| its steps: median / worst | 1.65 ms / 3.66 ms |
| `t0`, the load tick | **8.19 ms** saturated / 7.74 idle, against the control's 3.10 / 2.73 |
| what follows it | the control's, for the remaining 3,556 ticks |

**That is better than the gate the `-gc` flip was decided on** — 71 ticks and 68 ms — on a save with twice the networks, and it is 2.3x the standardization pass's 17 ticks / 44.2 ms at n=200 uniform, which is roughly the ratio of the work. Nothing here reopens the mode decision; it says the number the root-scan fix left behind scales with the CREATE rather than with the save.

| what a 4,376-splitter save costs | control | **bbb** |
|---|--:|--:|
| `--create` (process clock to `Goodbye`) | 1.60 s | **39.6 s** |
| save size, saturated / idle (`stat -f%z`) | 1,376,909 / 1,265,236 B | **2,261,190 / 1,947,522 B** |
| load, `Loading script.dat` → the tick-0 script line | 0.010 s | **0.174 s** (median of 3) |

The 38 seconds of create is where the whole architecture's bill lands, exactly as the M4 note says: 404 compiles, one dispatch. Nobody builds 404 balancers in one tick. **+884 KB of save and +164 ms of load** is what a megabase's worth of compiled network and guest heap costs every player on every join, forever.

### Correctness first, and it is unanimous

Delivery per output over the same 3,000-tick window, against the same save's own bare express belts — the control is the yardstick, so "full throughput" is a comparison with the engine rather than arithmetic on a wiki number. One uninterrupted express belt delivered **2,162** items per output:

| class | rigs | outputs | bbb total | vs control | per-output min/max | balance |
|---|--:|--:|--:|--:|---|--:|
| `2->2` | 120 | 2 | 523,680 | 100.2% | 261,840 / 261,840 | **1.0000** |
| `3->3` | 80 | 3 | 520,800 | 100.0% | 173,600 / 173,600 | **1.0000** |
| `4x4` | 80 | 4 | 696,000 | 100.6% | 173,920 / 174,080 | 1.0009 |
| `8x8` | 40 | 8 | 693,600 | 100.3% | 86,640 / 86,720 | 1.0009 |
| `3->5` | 40 | 5 | 258,519 | 99.6% | 51,680 / 51,759 | 1.0015 |
| `5->3` | 40 | 3 | 258,718 | 99.7% | 86,160 / 86,320 | 1.0019 |
| **`16x16`** | 1 | 16 | 34,560 | 99.9% | 2,158 / 2,162 | 1.0019 |
| **`32x32`** | 1 | 32 | 68,880 | 99.6% | 2,150 / 2,154 | 1.0019 |
| **`64x64`** | 1 | **64** | 137,280 | **99.2%** | 2,142 / 2,148 | **1.0028** |
| `65->1` | 1 | 1 | **0** | — | refused, and must be | n/a |

**A 64-port balancer splits 64 ways at 1.0028 and delivers 99.2% of a full belt on every one of them.** The two dead-end/loopback shapes are the other half of the statement and the half a uniform matrix cannot make: `3->5` spreads three belts over five outputs where the control leaves two of them at **zero**, and `5->3` runs its three outputs at a full belt each from five inputs. Worst per-rig balance anywhere in the save is **1.003** on all three saturated reps.

**The 65-input cluster is refused, once, and the world is untouched.** Exactly one `[BBB] alert: cluster … would need 128 ports for 65 inputs and 1 outputs, over the limit of 64; refused` in every bbb create, and **zero `[BBB] error:` in any log of the whole session** — which is the other half of the assertion, because the failure mode this replaced was three errors and a demolished network per belt ([`agents/maxports.md`](../maxports.md) §4). `run.sh` fails a mega cell that does not carry the alert.

### The 64x64: what it costs to build and what it costs to touch

`[BBB] compiled cluster 5683 64->64 over 64 ports, **1152 entities**, slot 403` — against the ~1,150 `agents/maxports.md` §1 derives by arithmetic, which is the first time that arithmetic has been checked against a real one.

**The first compile**, measured at create by the audit subtraction described in `bench/README.md` — an audit with nothing pending, then the 64x64 built, then the audit that compiles it. Three creates in this session:

| | audit only | audit + the 64x64 | **difference** |
|---|--:|--:|--:|
| saturated create A | 1,710.97 ms | 1,858.15 ms | **147.18 ms** |
| idle create | 1,830.32 ms | 1,968.30 ms | **137.98 ms** |
| saturated create B | 1,796.50 ms | 1,958.00 ms | **161.50 ms** |

(The audit itself is 1.7–1.8 s for 403 clusters. It is a whole-save re-classification, it happens once inside `--create`, and it is why the measurement has to be a subtraction.)

**The recompile hitch — the number [`agents/maxports.md`](../maxports.md) §3 was waiting for.** M2's tick-pair pattern: one input belt of the 64x64 removed and put back, the profiler opened in the tick that mutates and closed in the tick that flushes, median of three, minus that run's own `idle tick pair, nothing pending` control.

```sh
bench/run.sh --mod bbb --scenario mega      --hitch -n 40 --ticks 2400 --runs 1
bench/run.sh --mod bbb --scenario mega-idle --hitch -n 40 --ticks 2400 --runs 1
```

| one 64x64 teardown-and-rebuild | idle tick pair | −1 input | full |
|---|--:|--:|--:|
| **EMPTY** (`mega-idle`) | 0.371 ms | **158.84 ms** | **154.64 ms** |
| **SATURATED**, run A | 1.646 ms | **366.51 ms** | **393.73 ms** |
| **SATURATED**, run B | 1.732 ms | **375.33 ms** | **385.07 ms** |

**An empty 64x64 recompile is ~155 ms and a full one is ~380 ms**, against the 8x8's 25.7 ms saturated. The empty arm is dead flat across its three reps (152.6 / 159.2 / 160.7 and 151.6 / 155.0 / 156.9); the **saturated arm rises within a run** — 306.8 → 368.2 → 410.4, reproduced as 313.7 → 377.1 → 426.7 in the second run — and that is not noise, it is the network still filling: the reinsertion is proportional to the items in flight ("A recompile is not a removal") and a saturated P=64 butterfly holds thousands of them. The medians above are therefore a mid-fill reading and the ceiling is the last rep's ~410–427 ms.

**Which is a 25-tick freeze, and it is the honest headline of this section.** It is a single tick, on every client, in lockstep, triggered by one belt. `agents/maxports.md` §3 asked for this number before anyone raises `MaxPorts`, and the answer is that 64 is already past where the hitch is comfortable — see that file for what it now says about 128 and beyond.
