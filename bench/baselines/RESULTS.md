# Head-to-head: BetterBeltBalancer against belt-balancer-2 and belt-balancer-3

What one balancer costs the simulation, per tick, measured against both existing balancer mods and against a no-balancer control on the same rigs. Method and harness: [`../README.md`](../README.md). The incumbents on their own: [`BASELINE.md`](BASELINE.md). Raw rows: [`results.tsv`](results.tsv).

```sh
BENCH_TMP=/tmp/bbb-bench MODS="bb2 bb3 bbb" bench/matrix.sh
```

Measured August 2026 on an Apple M3 Pro (11 cores), macOS 26.6, Factorio 2.0.77 headless, base only (Space Age, quality and elevated-rails disabled). Every cell is `--benchmark-ticks 3600 --benchmark-runs 2`. The rows this document quotes are the ones tagged `head2head` in `results.tsv`: control, bb2, bb3 and bbb for one geometry are run back to back before the next geometry, because absolute timings on one machine drift 25-35% between sessions (see Caveats).

## Headline

A saturated 4x4 balancer on express belts costs 0.5 µs/tick against belt-balancer-2's 21.9 µs and belt-balancer-3's 23.1 µs, a 45x improvement, and none of what is left is Lua. Two hundred of them cost 0.64 ms/tick against bb2's 4.92 ms: 4% of the 16.67 ms 60-UPS budget instead of 30%.

## Marginal cost per balancer

Cost of one additional balancer, per tick, in µs, taken as the slope between the n=50 and n=200 cells minus the control's slope over the same pair, so the fixed cost of an empty save and the per-rig cost of belts, loaders, chests and the meter both cancel.

| per saturated 4x4 balancer, per tick | bb2 | bb3 | bbb | vs bb2 | vs bb3 |
| --- | --: | --: | --: | --: | --: |
| express (nth_tick 1) | 21.9 | 23.1 | 0.49 | 45x | 47x |
| normal (nth_tick 8) | 7.55 | 7.67 | 0.35 | 22x | 22x |

The control's own slope, the rig around the balancer, is 2.17 µs/rig on express and 1.13 µs/rig on normal. That is the floor a free balancer would sit at, and bbb is within a quarter of it.

Single-cell deltas against the control at the same geometry, which do not assume linearity and cover the geometries with only one `n`:

| cell | control | bb2 | bb3 | bbb | vs bb2 | vs bb3 |
| --- | --: | --: | --: | --: | --: | --: |
| express n=50 k=4 saturated | 0.2355 ms | 21.5 | 22.6 | 0.20 | 108x | 113x |
| express n=200 k=4 saturated | 0.5610 ms | 21.8 | 23.0 | 0.42 | 52x | 55x |
| express n=50 k=8 saturated | 0.3205 ms | 42.1 | 43.3 | 0.65 | 65x | 67x |
| express n=200 k=4 idle | 0.1870 ms | 1.70 | 2.91 | 0.16 | 10.6x | 18x |
| normal n=50 k=4 saturated | 0.2095 ms | 7.01 | 7.09 | -0.01 | | |
| normal n=200 k=4 saturated | 0.3785 ms | 7.42 | 7.52 | 0.26 | 29x | 29x |
| normal n=50 k=8 saturated | 0.2650 ms | 14.6 | 15.1 | 0.31 | 47x | 49x |
| normal n=200 k=4 idle | 0.1900 ms | 0.25 | 0.41 | 0.06 | 4.2x | 6.8x |

`normal n=50 k=4` came out at -0.01 µs/balancer: 50 balancers were 0.5 µs/tick cheaper than the same map without them, which says only that at that size the difference is below the noise floor. The n=1 rows (0.1750 against 0.1700 ms on express) are dominated by the roughly 0.17 ms fixed cost of an otherwise empty save and are a smoke test, not a measurement.

bbb's advantage grows with balancer size and shrinks with belt speed, the inverse of the incumbents. bb2 walks input and output lanes every poll, so an 8x8 costs it 1.9x a 4x4 and express costs it 3x normal. A compiled network is engine state: an 8x8 is 84 hidden entities against a 4x4's 32, and they cost what the engine charges for entities that happen to be busy.

## The zero-script claim

`scriptUpdate` from `--benchmark-verbose`, steady-state half of the window. This is all mod Lua on the map, so every column includes the harness's meter; the `control` column is the meter and nothing else.

| cell | control (meter only) | bb2 | bb3 | bbb |
| --- | --: | --: | --: | --: |
| express n=200 k=4 saturated | 2.65 µs | 3823.69 | 4209.69 | 2.19 µs |
| express n=200 k=4 idle | 1.95 µs | 331.72 | 575.91 | 1.44 µs |
| express n=50 k=8 saturated | 1.25 µs | 1751.20 | 1895.61 | 1.51 µs |
| normal n=200 k=4 saturated | 1.86 µs | 1201.49 | 1217.98 | 1.94 µs |
| normal n=200 k=4 idle | 1.36 µs | 52.28 | 82.47 | 1.48 µs |

bbb's `scriptUpdate` is the control's in every cell, in both directions: 0.5 µs below the control on the headline cell and 0.1 µs above on another. Where bb2 spends 19.1 µs of Lua per balancer per tick (`(3823.69 - 2.65) / 200`), bbb spends nothing this instrument can distinguish from zero. Three independent checks agree:

1. `[BBB]` log lines inside the benchmark window: 0 at n=1, n=200 and n=500. Every code path in the guest logs in the default build, so a compile, a teardown, a re-classification or a surface scan inside the measured ticks would be visible, and `run.sh` counts them on every cell.
2. Every compile happens during `--create`. The setup mod builds the rigs in `on_init` with `raise_built = true`, the mod's `script_raised_built` handler fires there, and the save handed to `--benchmark` already contains the finished networks.
3. There is no `on_tick` handler. The benchmark is the design claim under load: a compiled network does not need a script to keep running.

## Where the remaining 0.5 µs goes

It is the engine simulating the hidden network: 32 real entities per 4x4 balancer, no Lua involved. `--benchmark-verbose` at n=200 k=4 express saturated, one session, `luaGarbageIncremental` asked for as a fifth counter (rows tagged `gc pass`):

| | control | bbb | delta per balancer |
| --- | --: | --: | --: |
| `wholeUpdate` | 565.26 µs | 672.41 µs | 0.54 µs |
| `transportLinesUpdate` | 136.37 | 150.87 | 0.07 |
| `entityUpdate` | 303.81 | 356.56 | 0.26 |
| `scriptUpdate` | 2.01 | 2.24 | 0.001 |
| `luaGarbageIncremental` | 14.80 | 48.12 | 0.17 |

The three attributed lines add to 0.50 of the 0.54 µs. The belts and entities are the hidden splitters and linked belts doing the work; the GC line is the one that is not simulation and is the subject of the next section.

## Worst ticks

`max_ms`, the worst single tick of the run, uninstrumented, from the `head2head` rows:

| cell | control | bb2 | bb3 | bbb |
| --- | --: | --: | --: | --: |
| express n=200 k=4 saturated | 2.07 | 81.8 | 79.2 | 21.1 |
| express n=50 k=8 saturated | 1.47 | 32.5 | 32.1 | 11.5 |
| normal n=200 k=4 saturated | 2.29 | 68.1 | 64.5 | 21.3 |
| express n=200 k=4 idle | 4.43 | 7.36 | 3.01 | 27.8 |
| normal n=200 k=4 idle | 1.78 | 2.52 | 3.18 | 19.6 |

On the saturated cells bbb's worst tick was a quarter of bb2's; on the idle cells it was four times worse than either incumbent. Per-tick attribution with `luaGarbageIncremental` added, n=200 k=4 express idle:

| tick | `wholeUpdate` | `luaGarbageIncremental` | `scriptUpdate` |
| --- | --: | --: | --: |
| t0 (first tick after load) | 20.63 ms | 17.41 ms | 1.60 ms |
| t712 | 16.98 ms | 16.75 ms | 0.000 ms |
| every other tick | ~0.2 ms | ~0 | ~0.002 ms |

At t712 the map is idle and the mod runs no Lua, and the tick still costs 17 ms: a Lua garbage-collection pause over the guest's linear memory, which is a Lua table. The pause tracked the heap (3.0 ms at n=50 idle against 27.8 ms at n=200, for a heap 4x larger), and the heap turned out to be 64 MiB of the mod's own log-line strings, permanent under a leaking allocator. Rebuilt on one reusable buffer with the same line formats, and re-measured on the same cell, median of five runs each with its own in-session control:

| n=200 k=4 express idle | before | after |
| --- | --: | --: |
| guest linear memory | 64 MiB | under 16 MiB |
| `max_ms` | 18.05 ms | 2.26 ms (control 1.42) |
| bench save | 3.63 MB | 1.50 MB |
| `--create` | 44.4 s | 11.7 s |

At n=500 idle (median of 3): 256 MiB and 49.68 ms to under 16 MiB and 5.08 ms, against a 3.10 ms control. On the saturated n=200 cell the worst tick went from 21.1 ms to 2.43 ms against the control's 1.67. The `head2head` table above is kept as the record of what a 64 MiB guest heap costs; the current build does not have one.

Two things about the tail that were true before and after: bbb's GC mean is lower than bb2's (48.12 µs/tick against 260.19 at n=200 k=4 express saturated, control 14.80), because bb2 allocates fresh tables every poll and bbb allocates nothing at steady state; and the incumbents' spikes are their own Lua on a busy map, while bbb's were a collector pause whose frequency did not depend on load. `--persist=packed` does not change the live heap and was measured not to change this tail (idle n=200 `max_ms` 15.00 ms median in table mode against 14.16 packed, over three cells of five runs per mode); packed is shipped for the save and load instead (49.4 MB to 3.6 MB at n=200; 21.6 s to 8.2 s to load).

## Scale-out: 500 balancers

One cell, bbb only, with its own control beside it. The incumbents were not run here: at 21.9 µs/balancer bb2 would be about 10.9 ms/tick of balancer on top of its rigs, roughly 12.5 ms/tick in total.

| n=500 k=4 express saturated | avg_ms | max_ms | script_us | throughput/600t | balance |
| --- | --: | --: | --: | --: | --: |
| control (no balancers) | 1.6195 | 8.40 | 4.68 | 900,000 | 1.000 |
| bbb | 2.0500 | 88.50 | 6.49 | 900,000 | 1.001 |
| bb2, projected from its slope | ~12.5 | | ~9,600 | | |

0.86 µs/balancer, above the 0.49 the n=50 to n=200 slope gives, which is what a map with 8,000 more busy entities should do. 500 balancers cost 0.43 ms/tick of balancer, 2.6% of the 60-UPS budget, and deliver exactly the control's item rate. The `max_ms` of 88.5 ms is the same pre-fix GC pause, grown with the heap. The idle n=500 cell, re-measured after the log-line fix and with batched compiles (median of three, own control), reads 5.08 ms (control 3.10) instead of 49.68 ms, a 1.96 MB save instead of 8.46 MB, and a 36.9 s create instead of 85.6 s.

## Throughput and correctness

Verified before any timing was trusted, and as a rate rather than a total: the meter drains every sink chest every 600 ticks and logs cumulative per-output counts, and what matters is the steady-state window, because a cumulative total also carries whatever is standing in the pipeline. Items delivered per 600-tick window, once filled:

| cell | control | bb2 | bb3 | bbb | belt arithmetic |
| --- | --: | --: | --: | --: | --- |
| n=200 k=4 express | 360,000 | 360,000 | 360,000 | 360,000 | 200 rigs x 4 belts x 45 items/s x 10 s |
| n=200 k=4 normal | 120,000 | 120,000 | 120,000 | 120,000 | x 15 items/s |
| n=50 k=8 express | 180,000 | 180,000 | 180,000 | 180,000 | 50 x 8 x 45 x 10 |
| n=500 k=4 express | 900,000 | | | 900,000 | 500 x 4 x 45 x 10 |

Exact in every cell, against both the control and the incumbents. The idle cells moved 0 items, which the harness enforces.

The `balance` column reads 1.001 for bbb where bb2 and bb3 read 1.000, and it is a constant, not a drift. At n=200 k=4 express the four output columns run 74,800 / 75,200 / 74,800 / 75,200 at the first sample and 434,800 / 435,200 / 434,800 / 435,200 at the last: a gap of 400 items (2 per rig) at both ends, and between t=1200 and t=3000 every output gained exactly 270,000. The rates are identical; the offset is a half-tick of belt phase established once while the hidden pipeline fills and never repaid. bb2 reads a clean 1.000 because it moves items between transport lines itself and equalises them exactly.

## The megabase cell

The uniform matrix is 200 copies of one 4x4. The `mega` scenario builds a mix (see [`../README.md`](../README.md)): 404 rigs of ten shape classes plus a 16x16, a 32x32, a 64x64 and a 65-input cluster the mod must refuse; 5,938 parts, 4,376 hidden splitters, on a 152x408 surface. Measured with the shipped configuration (`--persist=packed --gc=collected`), three reps of each of the four cells interleaved bbb/control per scenario, per-tick columns read from the verbose pass with the post-load transient excluded and reported separately. Rows tagged `mega r1..r3`.

| n=40 blocks, express, steady state | control | bbb |
| --- | --: | --: |
| saturated `scriptUpdate`, median tick | 0.21 µs | 0.29 µs |
| saturated `wholeUpdate`, median tick | 774.56 µs | 909.60 µs |
| saturated worst tick | 3.279 ms | 3.361 ms |
| idle `scriptUpdate`, median tick | 0.17 µs | 0.21 µs |
| idle `wholeUpdate`, median tick | 164.25 µs | 167.96 µs |
| idle worst tick | 2.726 ms | 1.976 ms |
| saturated `avg_ms` (transient included) | 0.8145 | 0.9485 |
| idle `avg_ms` (transient included) | 0.1895 | 0.2240 |
| `[BBB]` lines in any benchmark window | | 0 |

Read the medians, not the TSV's `script_us`: that column is a mean, and this save's meter drains 4,404 sink chests six times a run at about 1 ms a time, which averages to about 3.4 µs of every tick on both sides and swamps a median tick of 0.2 µs. The marginal whole-tick cost is `(0.9485 - 0.8145) / 404`, or 0.33 µs per balancer, lower than the uniform 4x4's 0.49 because most of a block is 2-to-2s and 3-to-3s.

The post-load transient: a `--create` compiles all 404 networks inside one dispatch where no paced collector step can run, so the collection lands on the first ticks after the load. `scriptUpdate` returns to the control's after 39 ticks in all six bbb passes, with 74.1 ms of script in total saturated (72.6-76.3) and 69.6 ms idle (67.6-75.3), steps of 1.65 ms median and 3.66 ms worst; the load tick `t0` is 8.19 ms saturated and 7.74 idle against the control's 3.10 and 2.73. The save costs 2,261,190 B saturated against the control's 1,376,909, and loads in 0.174 s against 0.010 s. The create takes 39.6 s against 1.60 s, which is where the whole architecture's bill lands.

Delivery per output over the same 3,000-tick window, against the same save's own bare express belts (one uninterrupted express belt delivered 2,162 items per output):

| class | rigs | outputs | bbb total | vs control | per-output min / max | balance |
| --- | --: | --: | --: | --: | --- | --: |
| 2-to-2 | 120 | 2 | 523,680 | 100.2% | 261,840 / 261,840 | 1.0000 |
| 3-to-3 | 80 | 3 | 520,800 | 100.0% | 173,600 / 173,600 | 1.0000 |
| 4x4 | 80 | 4 | 696,000 | 100.6% | 173,920 / 174,080 | 1.0009 |
| 8x8 | 40 | 8 | 693,600 | 100.3% | 86,640 / 86,720 | 1.0009 |
| 3-to-5 | 40 | 5 | 258,519 | 99.6% | 51,680 / 51,759 | 1.0015 |
| 5-to-3 | 40 | 3 | 258,718 | 99.7% | 86,160 / 86,320 | 1.0019 |
| 16x16 | 1 | 16 | 34,560 | 99.9% | 2,158 / 2,162 | 1.0019 |
| 32x32 | 1 | 32 | 68,880 | 99.6% | 2,150 / 2,154 | 1.0019 |
| 64x64 | 1 | 64 | 137,280 | 99.2% | 2,142 / 2,148 | 1.0028 |
| 65-to-1 | 1 | 1 | 0 | | refused | n/a |

A 64-port balancer splits 64 ways at 1.0028 and delivers 99.2% of a full belt on every output. The 3-to-5 spreads three belts over five outputs where the control leaves two of them at zero, and the 5-to-3 runs its three outputs at a full belt each. Worst per-rig balance anywhere in the save is 1.003 on all three saturated reps. The 65-input cluster is refused exactly once per create with no `[BBB] error:` in any log of the session.

The 64x64 is 1,152 hidden entities. Its first compile, by the audit subtraction described in the harness README, is 137.98-161.50 ms across three creates. Its recompile (`--hitch`: one input belt removed and restored, profiler opened in the tick that mutates and closed in the tick that flushes, median of three, minus that run's own idle tick pair):

| one 64x64 teardown-and-rebuild | idle tick pair | minus one input | full |
| --- | --: | --: | --: |
| empty (`mega-idle`) | 0.371 ms | 158.84 ms | 154.64 ms |
| saturated, run A | 1.646 ms | 366.51 ms | 393.73 ms |
| saturated, run B | 1.732 ms | 375.33 ms | 385.07 ms |

An empty 64x64 recompile is about 155 ms and a full one about 380 ms, against the 8x8's 25.7 ms saturated. The saturated arm rises within a run (306.8, 368.2, 410.4 ms; then 313.7, 377.1, 426.7) because the network is still filling and the reinsertion is proportional to items in flight, so the ceiling is the last rep's 410-427 ms: a 25-tick freeze on every client, triggered by one belt. It is the main input to any decision about raising the 64-port limit; [`../../agents/maxports.md`](../../agents/maxports.md) has the arithmetic.

## Targets

The design targets were at least 10x at scale, about 100x saturated, and idle equal to the control.

| target | result |
| --- | --- |
| at least 10x at scale | Met with margin: 45x (express 4x4) and 22x (normal 4x4) by slope; 29-67x by single-cell delta on every saturated geometry; 10.6x on idle express against bb2 and 18x against bb3. The weakest cell is idle normal at 4.2x against bb2, where bb2's own cost is 0.25 µs/balancer. |
| about 100x saturated | Not met on whole-tick cost, and not reachable there: 45x is what remains once the hidden network's own engine cost is counted, and that is simulation work rather than overhead. Only `express n=50 k=4` touches 100x (108x), too close to the noise floor to quote alone. On mod Lua alone it is met without bound: 19.1 µs/balancer for bb2 against nothing measurable. |
| idle equals control | Met. At n=200 k=4 express idle, `wholeUpdate` is 176.51 µs against the control's 176.58 and `scriptUpdate` 1.44 against 1.95; the worst tick is 2.26 ms against a 1.42 ms control after the log-line fix. |

## Caveats

- Session-to-session drift is 25-35% on this machine and is the largest error term. The `control` row for n=200 k=4 express reads 0.4735 ms in the original baseline session, 0.5940 in one re-measure and 0.5610 in the head-to-head; bb2 at the same cell reads 3.8125 and 4.9245. Spotlight indexing `bench/tmp` accounted for most of it. Every ratio in this document comes from cells measured back to back with their own control, and the rows from a first contaminated pass are annotated as superseded in `results.tsv`. The absolute milliseconds should not be compared with `BASELINE.md`'s.
- Two `confirm` rows repeat the headline cells uninstrumented: saturated n=200 express 0.6605 / 0.6605 / 0.6995 / 0.6445 across four measurements (within 5% of the mean), idle 0.2130 / 0.2070 / 0.2115 / 0.2190 (within 3%).
- bbb's marginal cost is a difference between two numbers within 15% of each other, so its relative precision is much worse than bb2's: 0.49 by slope, 0.42 by single-cell delta at n=200, 0.86 at n=500. Read it as "0.4-0.9 µs, and the ratio is tens", not as three significant figures.
- `avg_ms` includes the few hundred ticks of belt filling at the start of each run and any GC pauses; the `*_us` columns average over the calm half.
- The `control` rows carry `K` extra belt tiles per row where the balancer rows have parts, so they slightly overstate the non-balancer floor, which makes every delta here a slight underestimate of the balancer's cost.
- Single machine, base only, one benchmark process at a time.
