# Baseline: belt-balancer-2 and belt-balancer-3 on Factorio 2.0.77

The cost of the two existing balancer mods, belt-balancer-2 v2.0.9 and belt-balancer-3 v1.0.1, measured with the harness in [`../README.md`](../README.md). These are the numbers BetterBeltBalancer was built to beat; the head-to-head that re-measures both beside it is [`RESULTS.md`](RESULTS.md), and the raw rows are in [`results.tsv`](results.tsv).

```sh
bench/matrix.sh                       # regenerates the belt-balancer-2 rows
MODS="bb3" bench/matrix.sh            # the belt-balancer-3 rows
```

Measured August 2026 on an Apple M3 Pro (11 cores), macOS 26.6, Factorio 2.0.77 headless, base only (Space Age, quality and elevated-rails disabled). Every cell is `--benchmark-ticks 3600 --benchmark-runs 2`, and every saturated cell was verified to move items at the belts' full rate with `balance = 1.000` across all outputs.

## Headline

A saturated 4x4 belt-balancer-2 balancer on express belts costs about 17 µs of simulation time every tick. Two hundred of them cost 3.3 ms/tick, 20% of the 16.67 ms budget for 60 UPS, and about 90% of that is Lua. belt-balancer-3 is 25-40% more expensive saturated and about twice as expensive idle, so beating belt-balancer-2 subsumes it.

## Marginal cost per balancer

Cost of one additional balancer, per tick, taken as the slope between the n=50 and n=200 cells so that the fixed cost of an empty save (about 0.09 ms/tick) and the per-rig cost of belts, loaders, chests and the meter both cancel. Single-cell deltas against the matching control row agree to within about 10%.

| geometry | tier | nth_tick | saturated, whole tick | saturated, Lua only | idle |
| --- | --- | --- | --- | --- | --- |
| 4x4 | express | 1 | 16.9 µs | 15.1 µs | 1.40 µs |
| 4x4 | normal | 8 | 5.9 µs | 5.0 µs | 0.17 µs |
| 8x8 | express | 1 | 31.5 µs | 27.9 µs | |
| 8x8 | normal | 8 | 10.6 µs | 8.5 µs | |

"Lua only" is `scriptUpdate` from `--benchmark-verbose` minus the control's `scriptUpdate` (the meter, about 0.008 µs/rig). "Whole tick" is the `wholeUpdate` delta and includes the engine-side cost of the balancer's transport-line writes.

The rig around the balancer (6 belts, 2 loaders, 2 chests per row plus the meter) costs 1.7 µs/rig on express and 1.1 µs/rig on normal at k=4. That is the floor a zero-cost balancer would sit at.

Three things the table shows:

1. Express belts cost 3x normal, not 8x. belt-balancer-2 registers `on_nth_tick(0.25 / belt_speed)` and falls back to every tick when the ratio is not an integer. Express belts (0.09375 tiles/tick, ratio 2.667) trip the fallback, so an express balancer polls 8x as often as a normal one; it is only 3x as expensive because each poll also moves 3x fewer items. The per-invocation overhead dominates.
2. Idle balancers are not free, and the 8x is visible there. With empty belts, express costs 1.40 µs/tick and normal 0.17 µs, exactly the polling ratio with no item work to dilute it. 200 idle express balancers burn 0.28 ms/tick doing nothing.
3. Cost scales with belt count, not part count. An 8x8 has 4x the parts of a 4x4 but only 2x the belts, and costs 1.9x as much: the hot path walks input and output lanes (two per belt) and never touches the interior parts.

## Full matrix, belt-balancer-2

`avg_ms` is the whole-tick average over the run; the delta is against the matching control row at the same geometry.

Express (nth_tick = 1):

| n | k | scenario | belt-balancer-2 | control | delta ms | delta µs/balancer |
| --: | --: | --- | --: | --: | --: | --: |
| 1 | 4 | saturated | 0.1845 | 0.1605 | 0.024 | 24.0 |
| 50 | 4 | saturated | 1.0320 | 0.2235 | 0.809 | 16.2 |
| 200 | 4 | saturated | 3.8125 | 0.4735 | 3.339 | 16.7 |
| 50 | 8 | saturated | 1.8730 | 0.2995 | 1.574 | 31.5 |
| 200 | 4 | idle | 0.4505 | 0.1715 | 0.279 | 1.4 |

Normal (nth_tick = 8):

| n | k | scenario | belt-balancer-2 | control | delta ms | delta µs/balancer |
| --: | --: | --- | --: | --: | --: | --: |
| 1 | 4 | saturated | 0.1750 | 0.1655 | 0.010 | 9.5 |
| 50 | 4 | saturated | 0.4595 | 0.1995 | 0.260 | 5.2 |
| 200 | 4 | saturated | 1.5175 | 0.3665 | 1.151 | 5.8 |
| 50 | 8 | saturated | 0.7720 | 0.2405 | 0.532 | 10.6 |
| 200 | 4 | idle | 0.2030 | 0.1705 | 0.033 | 0.2 |

The n=1 rows are dominated by the roughly 0.16 ms fixed cost of an otherwise empty save; treat them as a smoke test, not a measurement.

## Throughput and correctness

belt-balancer-2 is correct and does not throttle. At every saturated cell the rig delivered the belts' full rate: 200 rigs x 4 express belts moved 360,000 items per 600 ticks, which is 3.0 items/tick/rig, or 4 x 45 items/s. The per-output split was exact (`balance` = 1.000 in every row). Any replacement has to match that before its timings mean anything.

## The tail

The mean hides the shape. Instrumenting the n=200 k=4 express cell per tick (1800 ticks, `scriptUpdate` only):

| | |
| --- | --- |
| mean `scriptUpdate` | 2.93 ms |
| ticks over 5 ms | 667 of 1800 (37%) |
| worst tick | 41.7 ms of script, 48.0 ms whole |
| `luaGarbageIncremental` mean | 0.17 ms |

A base with 200 express balancers does not just lose 20% of its tick budget on average; it drops several frames a second. `max_ms` in `results.tsv` shows the same at every size (57.6 ms at n=200 express against 1.6 ms for the control). The engine's own `luaGarbageIncremental` counter stays near 0.17 ms, so the time is inside the mod's own Lua; its `run()` allocates a fresh `next_lanes` table per pass and does `table.remove(buffer, 1)`.

## belt-balancer-3 v1.0.1

Same machine, method and matrix (rows tagged `belt-balancer-3_1.0.1` in `results.tsv`; two `confirm` rows are a repeat pass agreeing within 1.6-3%). bb3 is a fork of bb2 whose hot-path changes are `lane.valid` guards (crash fixes) plus minor Lua micro-optimisations. The guards are extra Lua-to-C++ boundary crossings per lane per activation:

| marginal per 4x4 balancer, per tick | bb3 express | bb2 express | bb3 normal | bb2 normal |
| --- | --: | --: | --: | --: |
| saturated, whole tick | ~23 µs | 16.9 µs | 7.4 µs | 5.9 µs |
| idle | 2.8 µs | 1.40 µs | 0.55 µs | 0.17 µs |

8x8 express at n=50: 43.2 µs/balancer against bb2's 31.5. Correctness matches bb2 (full throughput, `balance = 1.000` in every saturated cell) and the spikes are worse (`max_ms` 82.4 at n=200 express against bb2's 57.6).

## Reading these numbers against RESULTS.md

`RESULTS.md` re-measures bb2 and bb3 beside BetterBeltBalancer in one session rather than reusing the rows above. Do not compare its absolute milliseconds with this document's: the same control cell reads 0.4735 ms here and 0.5610 there, because timings on one machine drift 25-35% between sessions. The shape survives re-measurement (bb2 beats bb3 on every cell, express costs about 3x normal, an 8x8 about 1.9x a 4x4), and every ratio in `RESULTS.md` is taken within a session for that reason.

## Caveats

- Single machine, single run of the matrix, no confidence intervals. Re-run `bench/matrix.sh` before trusting a difference smaller than about 10%.
- `avg_ms` includes the couple of hundred ticks it takes to fill empty belts at the start of each run; the `*_us` columns are steady-state only. This makes `avg_ms` a slight underestimate of steady-state cost.
- The `control` rows carry `K` extra belt tiles per row where the balancer rows have parts, so they slightly overstate the non-balancer floor, which makes the delta a slight underestimate too.
- The bb3 matrix may have run with other Factorio processes active on the same machine (`run.sh` isolates the write directory, so there is no lock contention, but CPU contention is possible). The two `confirm` rows agree with the originals within 1.6-3%, so the numbers are trusted at about the 5% level.
