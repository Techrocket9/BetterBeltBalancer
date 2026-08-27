# Balancer benchmark harness

`bench/` measures what a belt balancer costs the Factorio simulation, in milliseconds per tick, under a reproducible saturated load. Everything runs headless against the real game binary; nothing here is a model of Factorio.

```sh
bench/run.sh --mod bb2 --scenario saturated -n 50 -k 4 --tier express
```

One invocation is one matrix cell and appends one row to [`baselines/results.tsv`](baselines/results.tsv). The incumbents' baseline is analysed in [`baselines/BASELINE.md`](baselines/BASELINE.md) and the head-to-head against BetterBeltBalancer in [`baselines/RESULTS.md`](baselines/RESULTS.md).

## Requirements

- A Factorio install. `run.sh` looks for the binary at the default Steam location on macOS; set `FACTORIO_BIN` otherwise. Which series it is is an input rather than an assumption: see "Which Factorio is running" below.
- The setup mod, which is a compiled guest. `run.sh` runs `make bench-setup` before every cell unless `BENCH_NO_BUILD` is set; that target builds one package and relinks nothing else.
- For `--mod bbb`, a built BetterBeltBalancer: `run.sh` runs `make zip` first (see the top-level [README](../README.md) for the build prerequisites), unless `BENCH_NO_BUILD` is set.
- For `--mod bb2` and `--mod bb3`, the incumbents' zips (`belt-balancer-2_2.0.9.zip`, `belt-balancer-3_*.zip`) in a directory named by `BB_MODS_SRC`. Third-party zips are never copied into the repository; they are staged into `bench/tmp/`, which is ignored by git.

## How a cell runs

Factorio's `--create` runs a mod's `on_init` while the map is generated, so a setup mod can place the entire benchmark into the save. For each cell `run.sh`:

1. stages a throwaway mod directory under `$BENCH_TMP/<cell>/mods/`: the built `bbb-bench-setup` package, a `mod-settings.dat` carrying that cell's configuration, the balancer mod's zip, and a `mod-list.json`;
2. `--create`s a save with a private write directory (`-c config.ini`), during which the setup mod's `on_init` builds `n` rigs;
3. `--benchmark`s that save with `--benchmark-ticks` and `--benchmark-runs`, uninstrumented, for the headline `avg_ms`;
4. re-runs it more briefly with `--benchmark-verbose` for the per-system breakdown, where the engine supports it (see "The per-system breakdown" below);
5. checks that items moved and that the outputs are balanced, and appends the row.

Space Age, quality and elevated-rails are disabled explicitly in `mod-list.json`; they ship inside the Factorio install and `--mod-directory` alone does not hide them. A base-only baseline keeps prototype counts and update costs comparable, and belt-balancer-2 lists Space Age only as an optional dependency.

### How a cell is configured

The setup mod's eight knobs are Factorio **startup settings**, which its own settings stage defines and `run.sh` writes into the staged `mod-settings.dat` before creating the save. `tools/mod-settings.py` is the writer; the keys are `bbb-bench-scenario`, `-n`, `-k`, `-tier`, `-item`, `-part-name`, `-meter` and `-hitch`, and every one of them is echoed back on the `BENCH-SETUP` line of the create log, so a misconfigured cell is visible in its own output.

Startup rather than map settings because a cell is two Factorio processes: `--create` and `--benchmark` read the same file directly, with no state carried in the save between them.

### Which Factorio is running

A mod whose `info.json` names a different series is refused at the loader before an entity is placed, so `run.sh` reads `Major.Minor` off `$FACTORIO --version` and **stamps** the staged setup mod for it: `factorio_version` unconditionally, and `base >= X.Y.Z` clamped down when it names a series newer than this engine. Both rewrites are no-ops when the manifest already agrees.

The balancer mod is not stamped. `--mod bbb` is built from this repository against a pinned API whose ABI marshals event payloads by name, so a disagreement with the binary is a real defect rather than a manifest token; the third-party zips are somebody else's and are used as they are.

### The per-system breakdown

`--benchmark-verbose` **crashes Factorio 2.1.16**: it emits the first tick row and then takes a SIGSEGV inside the engine's own benchmark loop, with any counter list, on any save. Measured on a vanilla no-mod save as the control, so it is the engine's defect and not a mod's. The crash also leaves a reporter process holding the run's write-directory lock, which would fail the next cell of a matrix on something unrelated to it.

**It works on Factorio 2.0.77**, which is what makes the sentence above a statement about one engine rather than about this harness. Re-checked there on a no-mod control cell: the pass ran to completion and every column came back, `whole 176.59us belts 50.90us entities 1.83us script 10.13us` over the steady-state half of 600 ticks. So the breakdown is available on the 2.0 arm and not on 2.1, and the columns being empty in a row says which engine measured it.

So on 2.1 the pass is skipped with the reason printed, and `whole_us`, `belts_us`, `entity_us` and `script_us` are empty in the row. `BENCH_VPROF_FORCE=1` runs it anyway, which is how a future engine gets re-tested. Where the pass does run, it averages `wholeUpdate`, `transportLinesUpdate`, `entityUpdate` and `scriptUpdate` over the steady-state second half.

## The rig

Each rig is one balancer plus everything needed to keep it permanently saturated. For a `K` in, `K` out balancer, `K` rows, with x growing east:

```
col 0        infinity-chest spawning `item`   (empty in the idle scenarios)
col 1..2     loader, type "output"            chest -> belt
col 3..5     input transport belts, facing east
col 6..6+K-1 the K x K block of balancer parts (plain belts in the controls)
col 6+K..+2  output transport belts, facing east
col +3..+4   loader, type "input"             belt -> chest
col +5       steel-chest, drained by the meter
```

The `K` belts on the west face of the part block are the inputs and the `K` on the east face the outputs; belt orientation is what decides, for belt-balancer-2 and for BetterBeltBalancer alike. Entities are created with `raise_built = true`, so the balancer mod's `script_raised_built` handler registers them exactly as if a player had placed them, and every compile happens during `--create`. The benchmark window therefore contains no compiles, which `run.sh` checks for BetterBeltBalancer by counting `[BBB]` log lines inside the benchmark (the guest logs on every code path it has, so a non-zero count means script ran during the measurement).

Rigs are laid out in a `ceil(sqrt(n))`-wide grid on a dedicated `bbb-bench` surface: no water, cliffs, resources, decoratives or enemies, paved flat, always day and frozen there. The surface is created peaceful with no entity autoplace, and any enemy on any surface is destroyed at init, so nothing in a bench save has anything to pollute or evolve.

The meter drains every sink chest every `--meter` ticks and logs cumulative per-output-column totals. It is what keeps the rigs saturated, and it is identical in every scenario, so it cancels out of any delta.

What it costs is two host calls per chest that has something in it and one per chest that does not, plus one guest dispatch per tick. The setup mod is a compiled guest, and there is no binding for Factorio's `on_nth_tick` because that call takes a Lua function: a mod that wants work every N ticks is entered on all of them and counts for itself. Both costs are the same in every cell including the no-mod controls, which is what makes them cancel; what they move is the absolute milliseconds a row reports.

### Uniform scenarios

| scenario | rigs contain | what it measures |
| --- | --- | --- |
| `saturated` | parts, chests spawning items | the real cost: every input belt compressed, every output drawing at full rate |
| `idle` | parts, chests empty | the polling floor: what the balancer costs with nothing to move |
| `control` | no parts, belts run straight through | the belts, loaders, chests and meter alone; subtract it from `saturated` |
| `control-idle` | no parts, chests empty | the same for `idle` |

The two `control*` scenarios require `--mod none` (they place no balancer part); the other two require a balancer mod.

### Megabase scenarios

The uniform scenarios build `n` copies of one shape at `k <= 8`. A megabase is a mix of sizes, most small and a few large, and the `mega` family builds that mix.

| scenario | rigs contain | pairs with |
| --- | --- | --- |
| `mega` | parts, chests spawning items | `mega-control` |
| `mega-idle` | parts, chests empty | `mega-control-idle` |
| `mega-control` | no parts, belts run through | (`--mod none`) |
| `mega-control-idle` | no parts, chests empty | (`--mod none`) |

Here `-n` is the number of blocks, not rigs. One block is ten balancers: three 2-to-2, two 3-to-3, two 4x4, one 3-to-5, one 5-to-3 and one 8x8, so that one save carries square shapes, a shape whose spare outputs loop back (3-to-5) and one whose spare outputs dead-end (5-to-3). Four more rigs are built once whatever `-n` says: a 16x16, a 32x32, a 64x64 (the mod's `MaxPorts` exactly), and a 65-input cluster the guest must refuse. `run.sh` requires the create log to carry the over-limit `[BBB] alert:` line for that cluster and no `[BBB] error:` line anywhere. A rig here has `n` inputs on the west face and `m` outputs on the east face of a part block `cols` wide and `max(n, m)` tall; rows past `n` have no source and rows past `m` no sink.

The setup mod counts the hidden splitter family (`bbb-splitter` plus `bbb-lane-splitter`) after compiling and fails the create below 3,000, so a silently shrunk population cannot read as a performance win. The default `-n 40` builds 404 rigs and 4,376 hidden splitters on a 152x408 surface.

Two extra log families appear: `BENCH-MEGA` and `BENCH-MEGA-SHAPE` at create time (population, splitter count and two timing probes), and `BENCH-SHAPE` at every meter sample, one line per shape class with that class's own min, max and balance. `run.sh` prints the last of each. Because shapes have different output counts, `per_output` on the `BENCH-METER` line is the vector of the worst-balanced rig in the save, so the harness's max/min is the worst per-rig balance anywhere; a mega cell fails above 1.25 and warns above 1.02. The refused 65-input cluster is excluded (it has no network), and in the control scenarios only the square shapes are asked to be even, because a 3-to-5's rows 3 and 4 have no source.

The first compile of the 64x64 is timed at create as a subtraction: the setup mod flushes everything else, times an audit with nothing pending, builds the 64x64, and times the audit that compiles it.

```
[BENCH-MEGA] timing audit only, nothing pending         Duration: 1710.97ms
[BENCH-MEGA] timing audit + FIRST COMPILE of the 64x64  Duration: 1858.15ms
```

`--hitch` (mega only) profiles the 64x64 recompile during the run: three times (ticks 1210, 1510 and 1810, chosen not to coincide with the 600-tick meter) it removes one input belt of the 64x64 and puts it back, opening a profiler in the tick that mutates and closing it in the tick that flushes, with an `idle tick pair, nothing pending` probe beside each to subtract. That cell mutates the world mid-run, so its `avg_ms` is not a steady-state number and must not be compared with one; its `max_ms` is the hitch itself.

```sh
bench/run.sh --mod bbb --scenario mega --hitch -n 40 --ticks 2400 --runs 1
```

### Belt tier

belt-balancer-2 picks its `on_nth_tick` divisor as `0.25 / belt_speed` over the output belts and falls back to every tick when that is not an integer:

| tier | belt | speed | `0.25/speed` | nth_tick |
| --- | --- | --- | --- | --- |
| `normal` | transport-belt | 0.03125 | 8 | 8 |
| `fast` | fast-transport-belt | 0.0625 | 4 | 4 |
| `express` | express-transport-belt | 0.09375 | 2.667 | 1 |

Express belts are the worst case for it by construction (8x the polling rate of normal belts) and are the tier that matters for a late-game base.

## Running the whole matrix

`bench/matrix.sh` runs the uniform matrix (`n=1,50,200 k=4`, `n=50 k=8` saturated and `n=200 k=4` idle, at each tier) with a matching control after every geometry. `MODS`, `TIERS`, `TICKS` and `RUNS` select the arms.

```sh
bench/matrix.sh                                       # the incumbent baseline
MODS="bb2 bb3 bbb" bench/matrix.sh                    # the head-to-head
BENCH_TMP=/tmp/bbb-bench MODS="bb2 bb3 bbb" bench/matrix.sh
```

Every mod for a geometry runs before the next geometry, and the control with them, because absolute timings drift 25-35% between sessions on one machine with background load, and a slow-moving drift scales a whole cell group together. Point `BENCH_TMP` outside the repository for a long matrix: the saves are hundreds of MB, and Spotlight indexing them inside the tree was measured to account for about 25% of the drift on its own.

`MEGA=1` runs the megabase matrix instead: `mega`, `mega-idle` and their controls, `REPS` times, with `--keep-save` because the save size is one of the things a megabase cell is for.

A mega save is about 35 MB, of which most is the setup mod's own guest heap: it places about twenty thousand entities inside one `on_init`, and a guest that allocates during a dispatch has no tick in which to collect. That heap is in `script.dat` and therefore in the save, at 64 MiB of linear memory for `-n 40`. It is not the hundreds of megabytes the uniform `n=200` cells produce, and it is charged identically to every arm of a mega geometry, so it cancels out of the deltas those cells are compared on.

```sh
BENCH_TMP=/tmp/bbb-mega BENCH_VPROF_TICKS=3600 MEGA=1 REPS=3 bench/matrix.sh
```

`BENCH_VPROF_TICKS=3600` matters there: a collected guest's post-load collector transient has to be excluded from a steady-state median and measured on its own, and both need the per-tick verbose column over a window longer than the default 1200 ticks.

## Sanity checks

`run.sh` fails the cell rather than recording a suspect row when:

- the setup mod does not log `BENCH-SETUP ... rigs_built=n` during `--create`;
- map creation produces a Lua error, or the balancer mod logs `[BBB] error:`;
- the benchmark logs `stack traceback` or `Error while running`;
- a `saturated` or `control` cell moves zero items, or an idle cell moves any (a rig where nothing moves is an idle map, not a benchmark);
- a mega cell's worst per-rig balance exceeds 1.25.

`balance` in the TSV is max/min across the `K` output columns; a working balancer sits at 1.000.

## Options

```
--mod X          none | bb2 | bb3 | bbb | /path/to/mod.zip  (default: none)
--scenario X     saturated | idle | control | control-idle
                 mega | mega-idle | mega-control | mega-control-idle
                                                       (default: saturated)
--hitch          mega only: profile three 64x64 recompiles during the run
-n N             number of rigs (mega: number of blocks)  (default: 1)
-k K             balancer size (K in, KxK parts, K out)   (default: 4)
--tier X         normal | fast | express                  (default: express)
--ticks N        --benchmark-ticks                        (default: 3600)
--runs N         --benchmark-runs                         (default: 2)
--meter N        ticks between throughput samples, 0=off  (default: 600)
--item NAME      item to push through                     (default: iron-ore)
--part-name NAME the balancer mod's part prototype        (default: per --mod)
--note TEXT      free-text note recorded in the TSV
--keep-save      do not delete the generated save
```

Environment: `FACTORIO_BIN`, `BB_MODS_SRC` (where the third-party zips live; the default is your Factorio `mods` directory), `BENCH_TMP` (default `bench/tmp`), `BENCH_VPROF_TICKS` (length of the verbose pass; 0 disables it), `BENCH_VPROF_EXTRA` (extra `--benchmark-verbose` counters for that pass, for example `luaGarbageIncremental`), `BENCH_VPROF_FORCE` (run the verbose pass on an engine where it is known to crash), `BENCH_NO_BUILD` (skip the builds that `--mod bbb` and the setup mod run).

`--mod bb2` resolves to `$BB_MODS_SRC/belt-balancer-2_2.0.9.zip` and `--mod bb3` to `$BB_MODS_SRC/belt-balancer-3_*.zip`. `--mod bbb` is this repository: it runs `make zip` (a real file target, so it rebuilds only when something changed) and uses `dist/<name>_<version>.zip`, named from `fklua.toml`. Any other value is a path to a mod zip.

The setup mod places whichever entity the `bbb-bench-part-name` setting names. belt-balancer-2 and belt-balancer-3 both call theirs `balancer-part` (bb3 is a fork of bb2); BetterBeltBalancer's is `bbb-balancer-part`. `run.sh` derives it from the mod name inside the zip and `--part-name` overrides. A scenario that places parts against a mod without that prototype fails in the setup mod's `on_init` with a clear message.

## Reading `results.tsv`

| column | meaning |
| --- | --- |
| `avg_ms`, `min_ms`, `max_ms` | mean over `runs` of Factorio's own per-run average; min of mins; max of maxes. Uninstrumented. |
| `whole_us` | `wholeUpdate`, mean over the steady-state half of the verbose pass, in µs |
| `belts_us` | `transportLinesUpdate`, the engine's own belt cost |
| `entity_us` | `entityUpdate` |
| `script_us` | `scriptUpdate`: all mod Lua, the balancer plus this harness's own |
| `throughput` | cumulative items delivered by the end of run 1 |
| `balance` | max/min across the output columns; 1.000 is perfect |
| `factorio` | the engine version that produced the row |
| `note` | free text; the head-to-head rows are tagged `head2head`, the megabase rows `mega r1..r3` |

`avg_ms` includes the couple of hundred ticks it takes to fill empty belts at the start of a run; the `*_us` columns do not.

The file is an append-only record, so it holds rows from more than one engine version and more than one session. **Only rows from one session are comparable with each other**, which is what the `factorio` and `date` columns are for and why `matrix.sh` interleaves every arm of a geometry: absolute timings on one machine drift 25-35% between sessions, and a whole Factorio version is a larger step than that.

`max_ms` is not a worst tick. Every `--benchmark` run begins by loading the save, and for a mod holding a guest heap that load is tens of milliseconds inside the measured window. Where a worst tick is really the question, read the per-tick `wholeUpdate` column out of the verbose pass's own log (`$BENCH_TMP/<cell>/verbose.log`) and drop the first hundred ticks. Factorio emits verbose columns in its own canonical order rather than the order asked for, which is why the parser reads the header. On an engine where the verbose pass crashes there is no such column and no worst tick can be read at all.

## Caveats

- `control` has `K` extra transport-belt tiles per row where the balancer scenario has `K` part columns, because items have to cross that gap. Belts are cheaper to update than parts but not free, so the control slightly overstates the non-balancer floor. The effect is small next to the numbers compared.
- `script_us` is total mod Lua, including the setup mod's per-tick dispatch and its meter. Compare it against the `control` row at the same `n` and `k` to isolate the balancer.
- Each rig is an independent balancer, so the numbers are per-balancer marginal costs taken from the slope across `n`, not from one absolute reading: a fresh save costs about 0.09 ms/tick before any rig exists.
- Cells compared with each other must be measured in the same session, for the drift reason above. Re-measure the control alongside whatever it is being subtracted from rather than reusing an old row.
- `luaGarbageIncremental` is not part of `script_us`; Factorio counts it separately and both live inside `wholeUpdate`. A mod that allocates is charged there, so run a pass with `BENCH_VPROF_EXTRA=luaGarbageIncremental` whenever `max_ms` looks worse than `script_us` can explain.
