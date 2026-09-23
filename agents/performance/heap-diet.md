# The heap diet — the idle GC tail, closed

**The idle GC spike was never a persistence cost, an architecture cost or a compiler cost. It was the guest's own log lines, built with `+`.** Fixing that took the idle worst tick at n=200 from **18.1 ms to 2.26 ms** and at n=500 from **49.7 ms to 5.08 ms**, and the bench save from 3.6 MB to 1.5 MB. It is the last performance defect this repo had a name for ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 17), and it is closed.

## What made it findable

Upstream wrote down the budget. `../FkLua/agents/guests.md`, "the guest heap budget": **0.2 ms of worst tick per MiB of guest linear memory**, flat from 8 MiB to 128, because Lua 5.2 walks a table in one `propagatemark` it cannot split across ticks — and it is the memory's SIZE, not the part in use, because `mem_grow` zeroes every new word and TinyGo's `growHeap` **doubles**. So a guest is always on a ladder, and `fk_mod.lua` now logs which rung it is on:

```
fklua: this guest's linear memory is now 64 MiB, which is about 12.8 ms of
worst tick -- ...
```

Three of those lines in one `--create` (16 → 32 → 64 MiB) is what turned a two-milestone attribution argument into a ten-minute measurement.

## The measurement that found it

One experiment, and it was decisive. Build the same guest with `QUIET=1`, which eliminates every `[BBB]` line below the error level, and create the same n=200 k=4 express bench save:

| n=200 k=4 express idle | doublings logged | idle worst tick | `--create` |
|---|---|--:|--:|
| verbose, string concatenation | 16 → 32 → **64 MiB** | 19.9 ms | 44.4 s |
| `QUIET=1`, no lines at all | **none (under 16 MiB)** | 2.28 ms | 9.3 s |

**The entire guest heap was log lines**, and so was three quarters of the create time. Everything the compile path does — the plan on its warm buffer, the edge list, the `create_entity` tables, the arena — was already frugal enough that without the logging the heap never reached the first rung that is visible at all. The M2 zero-allocation claim held through M3, M4 and M5; nobody had checked the instrumentation.

## Why a log line was 9 KB

`-gc=leaking` was mandatory then, so every intermediate string of every concatenation was permanent — in the heap, in the save, in every multiplayer join. And `+` in a loop is quadratic. `logState` is the one that mattered:

```go
s := "[BBB] state clusters=" + u32(uint32(len(snap))) + " parts=" + u32(nParts) + " sizes="
for i, n := range snap {
    if i > 0 { s += "," }
    s += u32(n)        // 64 of these, each allocating len(s)+len(n)
}
```

Up to 64 cluster sizes appended one at a time is ~8.8 KB of dead strings, plus a `strconv.FormatUint` allocation per number. **It runs once per part placed**, and a n=200 k=4 create places 3,200 parts: ~26 MB, which is the 32 → 64 MiB rung.

## The fix

`guest/go/logline.go`: one package-level `[512]byte`, an index, `copy` to append, and `unsafe.String` to hand the host a string that borrows the buffer rather than one that copies it. Every log line in the guest goes through it; `strconv` is gone from the module, and so are `u32`, `i32` and `f2s`. **The line formats are byte-for-byte what they were** — they are the assertion surface for every suite, and all of them pass unchanged.

Two things it is worth knowing before touching that file, both measured:

- **`copy` on a fixed array, not `append` on a slice.** `append` carries a growth branch that LLVM inlines into every one of the ~200 call sites: wasm `code` 98,373 B against 81,457, which is 1,339,988 B of generated Lua against 1,035,941. A line that overran would be truncated instead of grown; the worst line this guest can write is 427 bytes, and both unbounded ones already cap themselves (`state` at 64 sizes, `skin` at 32 variations).
- **`//go:noinline` on the writers was tried and rejected.** It is worth 17 KB of wasm `code` and 160 KB of generated Lua — the mod would ship *smaller* than before the file existed — and it costs **2.1 ms on every 4×4 recompile**. Measured interleaved against the same base, four reps each, medians minus each run's own control: 5.50 ms without it, 7.59 ms with. The profiled window writes two log lines, so this is not the log path getting slower; it is what not inlining these does to the code around the ~350 host calls a recompile makes. A per-edit millisecond is worth more than a per-load kilobyte.

## What it bought

Five reps per cell, pre-fix and post-fix builds of the same commit, same machine, same session (`bench/run.sh --mod bbb --scenario idle`, `--keep-save`):

| | before | **after** | |
|---|--:|--:|---|
| n=200 linear memory | **64 MiB** | **under 16 MiB** | no doubling logged |
| n=200 idle worst tick, median | 18.05 ms | **2.26 ms** | **8.0×** |
| n=200 idle worst tick, range | 17.0–19.9 (5 reps) | **2.19–2.37** (8 reps) | |
| n=200 bench save | 3.38–3.73 MB | **1.34–1.51 MB** | 2.4× |
| n=200 `--create` | 44.4 s | **11.2 s** | 4.0× |
| n=500 linear memory | **256 MiB** | **under 16 MiB** | |
| n=500 idle worst tick, median of 3 | 49.68 ms | **5.08 ms** | **9.8×** |
| n=500 bench save | 8.46 MB | **1.96 MB** | 4.3× |
| n=500 `--create` | 85.6 s | **36.9 s** | 2.3× |

**The recompile hitch is unchanged**, which was the thing this pass was not allowed to break. Interleaved against a build of the previous commit, three reps each, medians minus each run's own `idle tick pair` control: a 4x4 teardown-and-rebuild is **5.54 ms against 5.61**, an 8x8 **10.94 against 10.85**.

Against their own in-session no-mod controls, the tail is now a rounding error rather than a regression: **n=200 idle control 1.42 ms against our 2.26**, and **n=500 idle control 3.10 ms against our 5.08**. It used to be 4.43 against 27.8.

The target this pass set out with was ≤16 MiB at n=200 and ≤32 MiB at n=500, i.e. ~3 ms and ~6 ms of worst tick. **Both are met at n=500's budget**: the heap does not reach the first logged rung at either size.

## What is left, quantified

The heap probe (the leaking allocator's own bump pointer, read at the audit) puts the residual at:

| | per rig (16 parts) | of which |
|---|--:|---|
| registering parts and their events | ~4.6 KB | ~290 B/part: the registry's parallel slices and `index` map, each carrying its doubling-growth history, plus one `name` string per build event |
| **one compile** | **~1.3 KB** | **all of it in generated binding RETURN values** |

Totals: 1.28 MB of heap at n=200 and 2.74 MB at n=500, against the 16 MiB rung.

**The ~1.3 KB per compile is not controllable from here**, and that is the honest part. A `find_entities_filtered` return is `out := make([]Object, n)` in `fkapi.go`; an entity return is `return &v`; a `type` read is `return string(b)`. A 4×4 compile makes 16 boundary queries, ~16 type reads, 32 `create_entity` calls, 16 `find_entity` calls and 8 `draw_sprite` calls, and every one of them leaves a Go value on a heap that never gives anything back. The marshalling arena (item 10) fixed the ABI's own side; this is the caller's side, and upstream's own note says so — "the 48 bytes that remain are the caller's, not the ABI's" (`../FkLua/agents/abi.md`).

At 1.3 KB per recompile a save needs **~800 balancer edits** to add one MiB of heap, not the ~12,000 this paragraph claimed until 2026-08-02: 1,048,576 over 1,300 is 806, and the error survived two rounds because nobody divided. The measured per-operation figures are in "The marathon save", which replaces this estimate with a table and a projection. [`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 17 stays open with those numbers on it rather than being closed as fixed. Nothing downstream can move it without changing how the classifier reads the world, which is a correctness surface and not a place to economise.

## The cost, stated plainly

The mod got bigger. `fk_module.lua` 953,523 → **1,035,941 B** (+8.6%) and the zip 186,267 → **191,677 B** (+2.9%), because ~200 `copy` call sites inline more code than ~200 string concatenations did. That is a per-load cost of about five kilobytes, against 45 MB of worst-tick pause and 2.1 MB of save at n=200. It is also the second time this repo has traded module size for a runtime property and been right to (the first was `-opt=2` over `-opt=z`, upstream's decision).

## The collected-mode postscript — the diet made the collector unnecessary HERE

**Upstream built the thing this section is a workaround for, and this mod measured it three times.** FkLua's `--gc=collected` ([`../FkLua/agents/gc.md`](../../../FkLua/agents/gc.md)) is a paced conservative mark-sweep over the guest's own heap: `-gc=custom` on the TinyGo side, one import, and one call. It is wired here — `make GC=collected` builds it, all seven suites are green under it, and it stays that way.

> **THE SHIPPED BUILD IS `--gc=collected` SINCE 2026-08-02**, and the four sections below are the decision in the order it was taken: leaking on one benchmark, leaking re-affirmed on marathon numbers with a recommendation to re-measure, and then **"The third decision"**, which re-measured on today's sharded pin and flipped it. Read them in order; each one's premise is what the next one moved. The loser's numbers are the useful half in both directions.

This section is the 2026-08-01 pass, when the answer was leaking.

The wiring, so that re-taking the decision is a flag and not a project: `guest/go/gc.go` (the import, one `fkgc.CollectIfNeeded()` at the end of `fk_on_deferred`, and the `[BBB] heap` line both variants emit), a `GC` variable in the Makefile that moves `-gc=custom` and `--gc=collected` together behind one stamp, and nothing else. Under `-gc=leaking` `guest/go/fkgc` is an empty package of no-ops, so the leaking build is what it was.

Measured 2026-08-01, Factorio 2.0.77, n=200 k=4 express, both arms interleaved in one session with a no-mod control in the same rep — the same rule the `--persist` pass used, and this session drifted more than that one did (the control's own idle worst tick spanned 0.86–11.1 ms over five reps).

| | `-gc=leaking` | `--gc=collected` | |
|---|--:|--:|---|
| `dist/bbb.wasm` | 770,238 B | 864,065 B | |
| `fk_module.lua` | 1,048,475 B | **1,319,975 B** | **+25.9%**, every load |
| the zip | 196,699 B | **220,057 B** | **+11.9%** |
| M2 save (seeded, deterministic) | 956,564 B | 961,007 B | +4,443 B |
| n=200 bench save, median of 5 | 1.33 MB | 1.45 MB | ranges overlap |
| heap after the n=200 `--create` | 2,017,648 B | **1,736,008 B** | **−14%**, and see below |
| saturated `avg_ms`, median of 5 | 0.6830 | 0.8545 | control 0.6080 |
| idle `avg_ms`, median of 5 | 0.2050 | 0.2315 | control 0.2075 |
| idle `scriptUpdate`, steady half | 1.86 µs | 1.69 µs | control 1.50 |
| first tick after a load (`t0`) | 2.9 ms | **5.2 ms** | 5/5 reps, no overlap |
| **ticks of collector script after a load** | **0** | **152** | see below |
| 4×4 recompile, median of 3 minus own control | 8.94 ms | 6.70 ms | ±25%, a wash |
| 8×8 recompile, same | 13.27 ms | 13.68 ms | a wash |
| items / balance | 1,740,000 / 1.001 | 1,740,000 / 1.001 | identical |

**The row that decided it is the one about ticks, and it is not a millisecond.** The mod's headline is that a finished balancer runs **no script at all** — zero `[BBB]` lines and `scriptUpdate` at the no-mod control's, in every benchmark window, since M4. The collected build does not have that property, and the reason is structural rather than incidental:

> `--create` never reaches a tick, so all 200 networks are compiled inside ONE `bbb-audit` dispatch. **No paced step can run inside a dispatch**, so the create allocates 1.3 MB with `cycles=0` — and the first tick after the save is loaded runs the deferred flush, which starts the collection the create deferred.

Measured off the per-tick `scriptUpdate` column, and it is the same number in every one of ten cells: **152 collector steps over ticks 0–151, 105 ms of script in total, median 623 µs per step, p90 1.30 ms, worst 2.1 ms** (3.66 ms on `t0`, which carries the load as well). Then it stops and `scriptUpdate` returns to the control's for the rest of the run. The leaking arm's same column has exactly one tick over 50 µs and it is the harness's own meter.

**What those 105 ms buy on this guest is close to nothing, and the reason is gc.md's own success metric.** That document is emphatic that *the collector's job is to prevent `memory.grow`, not to free bytes* — because linear memory never shrinks, so a heap that has been 1.6 MiB is walked as 1.6 MiB forever. On the n=200 create the growth all happened inside the one dispatch, before any step could run: the collection reclaims the ~1.3 KB-per-compile binding residue ("What is left, quantified" above — about 0.26 MB of a 1.57 MB heap) and the heap size does not move at all.

**And the tail it would be defending against is already gone.** The attribution run — the same idle cell with `luaGarbageIncremental` added to the breakdown, three reps, both arms and the control — puts Lua's own collector at **8–21 µs of mean tick in all three arms**, the no-mod control included. At the post-diet 1.9 MiB the guest heap contributes nothing this instrument can see, which is exactly what the diet was for. There is no tail here for a guest-heap collector to shorten.

Two smaller findings, both worth keeping:

- **The `−14%` create heap is the ALLOCATOR, not the collection.** `cycles=0` on that line: nothing had been collected yet. TinyGo's `growHeap` doubles, so leaking's 1.14 MB of allocations sit in a 1.92 MiB arena; `fkgc` grows in smaller steps and reached 1.50 MiB for the same work. That is an argument for the `-gc=custom` seam that has nothing to do with collecting — and it does not survive the collector's **163 KiB of static metadata**, which is linear memory in every save and every join whether or not anything is ever collected.
- **The recompile hitch is a wash, and the collected arm's spread is wider for a reason.** The profiled window opens in the mutating tick and closes in the flushing one — which is where `fkgc.CollectIfNeeded()` lives — so a collection's initial root scan can land inside the probe. `-1 input` came out 7.55 → 8.13 ms and `full` 8.94 → 6.70 ms across the same three interleaved reps; both directions are inside this session's noise and neither is claimed.

## Where collected WINS, which is why it stays buildable

One workload separates them completely, and it is the one a future contributor would create by accident. The M3 stress phase cranked from one mutation every six ticks to **one every tick over 5,400 ticks** — 5,400 mutations against the shipped suite's 100, a scratchpad copy of the M3 observer (`test/mods/bbb-m3-test` then, `guest/go/obs/m3` now) so the shipped one keeps its calibrated numbers:

| 54× churn, 5,400 mutations | `-gc=leaking` | `--gc=collected` |
|---|--:|--:|
| linear memory at the end | **4,114,800 B** | **585,032 B** |
| the curve | 182 KB → 445 → 969 → 2,018 → **4,115 KB**, still climbing | 115 → 218 → **418 KB, flat from t≈2.4 s** |
| `memory.grow` calls | on the doubling ladder | **6, none after t≈2.4 s** |
| live set | — | ~12.9 KB, stable |
| collections / mark deadlines | — | 8 / **0** |
| items in / recovered | 16,000 / **12,075** | 16,000 / **12,075** |
| final audit | `clusters=14 parts=29 nets=14 drift=0 unbuilt=0` | **identical** |

**7.0×, checksum-identical, with a heap that plateaus against one that only climbs.** That is the whole case for the feature, on the workload it was designed for, and BBB does not have that workload — 5,400 balancer edits is not a session, it is a fuzzer. But an allocation regression in the compile path *would* look exactly like it, and today nothing but the discipline in [`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 17 stops one.

**No `fkgc:` line was ever logged, in any run.** The OOM valve never fired and `Stats().Deadlines` was **0** everywhere — including the 54× churn, which is the first real-guest evidence for either. `markDeadline`'s slack and floor were unvalidated against a real guest before this pass; they held.

## The first decision (2026-08-01) — leaking, and what would reverse it

**BetterBeltBalancer shipped `-gc=leaking`.** The rule this pass set out with was: flip if collected costs nothing the steady state can see AND bounds a heap that is actually growing. **Neither half passed.** The steady state gained 152 ticks of script per load on a mod whose headline is that it has none, and the heap it would bound is 1.9 MiB that the diet already made invisible to Lua's collector. The costs it does have — +11.9% zip, +25.9% of `fk_module.lua`, +2.3 ms per load, 163 KiB of permanent linear memory — are paid by every player on every join, forever, which is the same class of cost the `--persist` decision turned on.

`make GC=collected` stays green and stays tested for the case that reverses it:

- **an allocation regression in the compile path.** The 54× churn table is what one looks like, and the collected arm is the only one that survives it. If "What is left, quantified" ever stops being ~1.3 KB per compile, re-measure rather than re-argue.
- **a guest that gains a real `fk_on_tick`.** Every cost above traces to the create being one dispatch and the collection therefore landing after a load. A guest that ticks would pace as designed, and the 152-tick lump would not exist.
- **a contributor who should not have to know about the diet.** That is the feature's real pitch and it is a fair one; it is just not worth 25% of the module today, on a guest where the discipline is already written down and already enforced by a measurement.

## The second decision (2026-08-02) — re-taken on marathon numbers, and not flipped

**The first of those three reversal conditions has not happened and the decision should still be re-taken, because the premise under the OTHER half moved.** The 2026-08-01 rule was: flip if collected costs nothing the steady state can see AND bounds a heap that is actually growing. Both halves were judged failed. Measured again 2026-08-02 against today's pin, with "The marathon save" above supplying the second half:

| the reason leaking won | as measured then | as measured now |
|---|---|---|
| **152 ticks of collector script per load**, on a mod whose headline is that it runs none | the n=200 bench `--create`, where 200 networks compile in ONE dispatch and no paced step can run inside one | **structural to a mass-builder, not to play.** Over the `mar` suite's 680 world operations spread across 4,600 ticks, the collected arm ran **29 paced steps and 6 collections in the whole run**. A player's edits arrive one tick at a time and pace as designed |
| **163 KiB of permanent linear memory** | a static `.bss` reservation | **73,112 B**, read off `meta=`. Upstream deleted the reservation at sharding stage C: `32,116 + 40,960 × ceil(heap / 4 MiB)`, a 31.4 KiB floor plus 0.977% |
| **+25.9% of `fk_module.lua`, +11.9% of the zip** | 1,048,475 → 1,319,975 B / 196,699 → 220,057 B | **worse: 1,434,040 → 1,898,702 B (+32.4%) and 228,190 → 259,294 B (+13.6%)** |
| **+2.3 ms per load** | measured | not re-measured |
| **the heap it would bound is 1.9 MiB the diet already made invisible** | true of a fresh save, and it was the load-bearing half | **false of a 300-hour one.** 25.7 MiB on a busy four-player server, and the doubling into it is a ~450 ms single-tick stall that nothing downstream can bound |

And the head-to-head on the workload the question is actually about — the same `mar` suite, same 680 net-zero operations, both arms:

| | `-gc=leaking` | `--gc=collected` |
|---|--:|--:|
| linear memory, start → end | 0.05 → **1.92 MiB** | 0.14 → **0.46 MiB** |
| the curve | the doubling ladder, 4 rungs | 7 grows in ~0.06 MiB steps |
| live set at the end | — | **8,736 B** |
| collections / paced steps | — | 6 / **29** |
| forward-progress deadlines | — | **2** |
| `fkgc:` outrun lines | — | **1** |
| items lost over 200 teardowns of a full network | 0 | **0** |
| final audit | `drift=0 unbuilt=0` | **identical** |
| all seven suites | green | **green** |

> The collected column is 2026-08-02 and is kept as measured. Today's guest runs the same suite in **14 paced steps with 0 deadlines and no outrun line**, on 0.59 MiB and a 9,216 B live set: the 29 steps and 2 deadlines here were the earliest visible symptom of the root-scan defect, one round before it became 779 and 6. See "The root scan that could not fit in a step".

**4.2× at 680 operations, and the ratio only grows**, because one side is on a doubling ladder and the other is 8.7 KiB of live set in a 0.46 MiB arena.

**The recommendation, which is not taken here.** On these numbers the mode decision points at `--gc=collected` for a mod that expects marathon multiplayer saves, and the reason is one row: the leaking arm's ~450 ms `memory.grow` stall is the only cost in the whole projection that a player would call a defect, and it is the only one no amount of discipline in this repo can remove. Against it stand +32.4% of `fk_module.lua` and +13.6% of the zip, which are per-load kilobytes of exactly the class this repo has twice traded for a runtime property and been right to.

**What is missing before anyone flips it, and it is the repo's own standard:** an interleaved `bench/run.sh` pass on today's sharded pin, re-measuring `t0`, the post-load collector steps, idle and saturated `scriptUpdate` and the worst tick. The 2026-08-01 numbers for those predate sharding, the grow pacing and the metadata rework, and this pass did not re-take them. **Re-measure rather than re-argue** — which is what the previous decision asked for, and the two rows above that moved are exactly the ones it named.

Two things that are NOT arguments for flipping, recorded so they are not mistaken for some:

- **The outrun and the two deadlines are ours, not the collector's.** `gcCollectIfNeeded()` is called from `fk_on_deferred` and nowhere else, and `fk_on_deferred` does not run on a tick where an event allocated but queued no cluster — which is exactly the `mar` suite's leg D, a belt laid far from any balancer. So the guest allocates on a path that never steps its own collector, the pacer falls behind, and `fkgc` correctly degrades to leaking and says so. A collected build that shipped would want a second call site, and choosing one is a design decision this pass deliberately did not take.
- **The mod is not shipped**, so the module-size rows are a download and a load, not a migration.

## The third decision (2026-08-02) — re-measured, and this mod ships `--gc=collected`

**Every number the leaking decision rested on was re-taken on today's pin, both arms interleaved, and the two that were supposed to be disqualifying are not.** This is the pass the section above asked for and did not do. Method as before: `bench/run.sh`, n=200 k=4 express, `--mod` pointed at the two arms' own zips so no TinyGo rebuild lands between cells, **five reps of six cells each in `leaking / collected / control` order** so session drift (25–35%, see Benchmarks) cannot bias one side; the `mar`/`m2` cells three reps the same way. Worst ticks and the post-load transient are read from the verbose pass's **per-tick** `wholeUpdate` and `scriptUpdate` columns, not from `max_ms`, which conflates the load tick ([`bench/README.md`](../../bench/README.md)). Rows tagged `gcpass2` in [`bench/baselines/results.tsv`](../../bench/baselines/results.tsv).

| | recorded 2026-08-01 | **today, leaking** | **today, collected** |
|---|---|--:|--:|
| `fk_module.lua` | 1,048,475 → 1,319,975 (+25.9%) | 1,434,097 | **1,899,407 (+32.4%)** |
| the zip | 196,699 → 220,057 (+11.9%) | 228,212 | **259,378 (+13.7%)** |
| `dist/bbb.wasm` | 770,238 / 864,065 | 795,455 | 910,633 |
| collector metadata (`meta=`) | 163 KiB, a `.bss` reservation | — | **73,112 B** |
| M2 save (seeded, deterministic) | 956,564 / 961,007 | 956,655 | 960,920 (**+4,265 B**) |
| n=200 bench save, idle / saturated | 1.33 / 1.45 MB | 1.396 / 1.271 MB | 1.456 / 1.261 MB |
| **saturated `avg_ms`**, median of 5 | 0.6830 / 0.8545 (ctl 0.6080) | 0.7965 | **0.7565** (ctl 0.5940) |
| **idle `avg_ms`**, median of 5 | 0.2050 / 0.2315 (ctl 0.2075) | 0.2055 | 0.2255 (ctl 0.1905) |
| idle `scriptUpdate`, steady half | 1.86 / 1.69 µs (ctl 1.50) | 1.48 µs | 1.82 µs (ctl 1.57) |
| saturated `scriptUpdate`, steady half | — | 2.92 µs | **2.10 µs** (ctl 2.24) |
| steady worst tick, idle / saturated | not taken per-tick | 1.15 / 2.65 ms | 1.30 / 2.75 ms (ctl 0.94 / 1.88) |
| **first tick after a load (`t0`)** | 2.9 / 5.2 ms | 2.73 / 3.32 ms | **5.33 / 5.19 ms** |
| load, `script.dat` → first tick's script line | — | 0.119 / 0.128 s | 0.134 / 0.130 s |
| **ticks of collector script after a load** | 0 / **152** | 0 | **71** → see below |
| that transient's total `scriptUpdate` | — / 105 ms | — | **65–71 ms** → see below |
| its steps: median / p90 / worst | 623 µs / 1.30 / 2.1 ms | — | 838 µs / 1.36 / **3.19 ms** (t0, which carries the load) |
| 4×4 recompile, median of 3 minus own control | 8.94 / 6.70 | 12.59 ms | **7.06 ms** |
| 8×8 recompile, same | 13.27 / 13.68 | 13.53 ms | 13.59 ms |
| items / balance | identical | 1,740,000 / 1.001 | 1,740,000 / 1.001 |
| all seven suites, from clean | green / green | green | **green** |

**The steady state cannot tell the two arms apart, and that is the row that decided it.** Every timing cell above either overlaps or points the wrong way for leaking: collected is FASTER on saturated `avg_ms`, on saturated `scriptUpdate` and on the 4×4 recompile, and the differences in all three are inside a session whose no-mod control moved 0.559–0.631 ms on the same six cells. `scriptUpdate` is 1.5–2.9 µs in every arm **including the control that has no balancer mod loaded at all** — which is the mod's headline property, unchanged: a finished balancer still runs no script, collected or not.

**And the 152 ticks are 71.** The transient is real, it is still the honest cost, and it is still structural to a `--create` that compiles 200 networks in one dispatch where no paced step can run — but sharding stage C and the grow pacing more than halved it: ticks 0–71, 65–71 ms of `scriptUpdate` in total against 105, then it stops and the column returns to the control's for the rest of the run. The steps are fewer and individually bigger (838 µs median against 623), and the worst of them is `t0` itself, which carries the load in both arms.

> **THAT ROW IS STALE AND IT IS THE ONE ROW THIS DECISION TURNED ON.** Re-measured 2026-08-03 on the same cell, the transient is **985 ticks and ~906 ms of `scriptUpdate`**, not 71 and 68 — **14×** — and it is the same 983 ticks / 915 ms on a guest built against the FkLua commit *before* the 2026-08-03 round, so it is not that round's doing. It is still a transient and it still ends dead: at t≈985 the column drops to 0.2–0.3 µs and stays there for the rest of the run. See "The transient that grew while nobody was looking" below, which is the whole re-measurement and what it does and does not change.
>
> **AND FIXED 2026-08-03, at which point the row became better than it was when the decision was taken.** The 985 ticks were this guest's globals crossing 16 KiB — one paced step's budget — after which the collector's mark phase could not afford its own termination attempt and ran to its livelock deadline. `guest/go/gc.go` installs a budget derived from the root scan now, and the transient is **36 ticks and 54 ms**, against the 71 and 68 this table quotes. The decision stands on a gate that reads better than it did. See "The root scan that could not fit in a step".

**What is left is size, and it is the only thing left.** +32.4% of `fk_module.lua`, +13.7% of the zip, +2.4 ms on `t0`, 73,112 B of collector metadata, 4,265 B of M2 save. That is a download and a per-load handful of kilobytes — the same class of cost this repo has now traded three times for a runtime property (`-opt=2` over `-opt=z`, `packed` over `table`, and this).

### What it buys, MEASURED rather than projected — the doubling stall

The 2026-08-02 projection said the leaking arm's one player-visible defect is a `memory.grow` stall of **~450 ms** when linear memory steps from 16 MiB to 32, and that nothing downstream can bound it. That was arithmetic — 4,194,304 words × 107 ns. **It is now a measurement, and it is worse than the arithmetic said.**

The instrument is a scratchpad copy of the `mar` suite's observer (`test/mods/bbb-marathon-test` then, `guest/go/obs/mar` since the 2026-08-25 estate port) running one leg — G, the 4×4 teardown-and-rebuild, the most expensive net-zero operation the suite has — 3,400 times over 13,700 ticks, benchmarked with per-tick `--benchmark-verbose`. (Scratchpad, not committed, for the same reason the 54× churn arm was: the shipped suite keeps its calibrated schedule.) The guest's own `[BBB] heap … sys=` line marks each rung, the probe's iteration number fixes the tick it was read at, and the grow tick is the one outlier inside that window.

| rung | words filled | **leaking worst tick** | ns/word |
|---|--:|--:|--:|
| 2 → 4 MiB | 524,288 | 48.7 ms | 92.9 |
| 4 → 8 MiB | 1,048,576 | 120.3 ms | 114.8 |
| 8 → 16 MiB | 2,097,152 | 226.1 ms | 107.8 |
| **16 → 32 MiB** | **4,194,304** | **782.4 ms** | **186.5** |

The two middle rungs land on the 107 ns/word model almost exactly, which is what says the instrument is measuring what it claims. **The last one does not, and it is the one the projection was about**: 782 ms is 1.74× the predicted 450, and the excess is the shape `../FkLua/agents/sharding.md` §15 names — a 16 MiB grow creates eight new 2¹⁹-word shards and pays each one's last array-part reallocation (~17 ms apiece, flat in heap size) on top of the fill. So the projection's arithmetic was sound and its ANSWER was optimistic, and the correction goes the wrong way for leaking.

Same 3,400 operations, the other arm:

| | `-gc=leaking` | `--gc=collected` |
|---|--:|--:|
| linear memory at the end | **33,474,768 B (31.9 MiB)** | **548,248 B (0.52 MiB)** |
| the growth law | the doubling ladder, 9 rungs | 8 grows, all below 0.6 MiB |
| worst tick in the whole run | **782.4 ms** | **71.4 ms** |
| the next four worst | 226.1 / 136.9 / 120.3 / 93.8 ms | 57.8 / 46.4 / 45.4 / 41.1 ms |
| live set at the end | — | 9,456 B |

**61×, and the collected arm's non-grow tail is lower too.** That last row is worth more than the ratio: it says the collector is not buying a bounded heap at the price of a worse ordinary tick, which is the trade a paced collector is usually accused of.

### The rule, and what it decided

The rule this pass set out with, from the section above: **flip if the re-measured steady-state costs are within noise of leaking, AND the post-load transient is materially reduced or bounded and explained, AND the size penalty is the only remaining cost** — weighed against removing the marathon stall class. All three passed, and the thing on the other side of the scale got bigger rather than smaller. **BetterBeltBalancer ships `--gc=collected`.**

`make GC=leaking` stays buildable, stays green on all seven suites, and is what a future pass re-measures against. What would reverse THIS decision:

- **a `fk_module.lua` that a mod portal or a load time cannot carry.** 1.9 MB is the cost, and it is the whole cost.
- **a guest whose live set stops being ~9 KB.** Every argument above rests on the collector having almost nothing to retain, which is what makes its mark cheap and its steps invisible. A guest that kept state per network rather than a 64-bit fingerprint would be a different measurement.
- **a single-player-only audience.** The quiet column of the projection never reaches a doubling worth the name, and for that player the collected arm is 1.9 MB of module for nothing.

### The pacer, and the call site that was missing

The second decision recorded the outrun and the two forward-progress deadlines as "ours, not the collector's", and named the fix it did not take: `CollectIfNeeded` ran only from `fk_on_deferred`, which does not run on a tick where an event allocated but queued no cluster. **That is fixed here and it is a prerequisite of the flip, not a follow-up.** The shape is in `guest/go/gc.go`:

> `fk_on_event` ends in `gcArmIfNeeded()`, which asks `fkgc.Stats()` whether the pressure has reached the collector's own threshold and, if it has, calls `requestFlush()` — the same `fk.Defer()` every other path uses. **It does not collect.** The collection still starts in `fk_on_deferred`, one tick later, at the outermost dispatch the safe-point argument requires; an event handler is not guaranteed to be one. An idle guest raises no events, arms nothing and registers nothing, so the zero-steady-state property is untouched — and `fkgc.Enabled()` is a compile-time false under `-gc=leaking`, so the leaking arm's `fk_module.lua` did not move by a byte.

The threshold is `fkgc`'s own (256 KiB) and `init` installs it with `fkgc.SetThreshold`, so the two cannot drift: arming BELOW it would defer a flush that `CollectIfNeeded` then declines, on every event, forever. **That last clause was FALSE from the moment it was written until 2026-08-03 — the install was silently discarded — and it cost this mod nothing, by accident. See "The threshold this guest installed and the collector never read" below.**

**Measured, on the starved path in isolation** — a scratchpad leg that lays twenty belts eighteen tiles from anything and picks them up again, 3,000 times, 120,000 allocating events, **with no audit marker inside the run at all** because an audit IS a flush and would hand the unfixed build the call site it lacks:

| pure far-belt run, 120,000 events | collected, before | collected, **after** |
|---|--:|--:|
| collections | **0** | **5** |
| paced steps | 0 | 687 |
| linear memory at the end | 2,121,112 B | **679,320 B** |
| `memory.grow` calls | 32 | 10 |
| forward-progress deadlines | 0 | **0** |
| live set | — (nothing was ever collected) | 8,624 B |

`cycles=0` over 120,000 allocating events is the starvation, isolated: the collected build was behaving exactly like a leaking one on the guest's highest-multiplier path. **3.1× of linear memory, for one call at the end of one function.**

**And the two deadlines on the `mar` suite are unchanged by it, which is a finding rather than a failure.** They are still 2, before and after, and the `fkgc:` outrun line is still 1. Located: they fire in leg C (i=85) and leg G (i=55), the two teardown-and-rebuild legs, which are the highest WRITE rate the suite has — and `markDeadline` is a write-rate escape, not a start-rate one (`../FkLua/agents/gc.md`: it fires when mark termination stops making net progress against the mutator's dirty rate). The audit's diagnosis attributed them to the starvation and the measurement says otherwise. What they cost is bounded and small: one unbudgeted mark termination over a **8,736 B live set** plus the outstanding dirty set, on a suite that performs 680 world operations in 4,600 ticks — **an edit every 6.8 ticks, about 75× the busiest rate the projection models.** No budget knob is set here, because setting one on this evidence would be tuning against a fuzzer.

> **BOTH SENTENCES ARE WRONG AND THE FIX WAS THE BUDGET KNOB.** `markDeadline` is a forward-progress escape, not a write-rate one specifically, and what was outrunning this collector was not the mutator — it was the guest's own **globals**, re-scanned at every termination attempt and charged against the same step budget. Legs C and G are simply where a collection is most likely to be in flight. With the budget derived from that scan the `mar` suite runs 8 collections in **14 paced steps with 0 deadlines**, against 6 in 779 with 6. See "The root scan that could not fit in a step".

### The threshold this guest installed and the collector never read

**`init`'s `fkgc.SetThreshold(gcArmBytes)` was discarded, on every build, for the whole life of the pacer fix above.** `initialize()` assigned both collector knobs their defaults unconditionally, and on `-target=wasm-unknown` at TinyGo 0.41.1 it runs AFTER a guest's package initialisers whatever `runtime_wasmentry.go` reads like — so this mod's install was written and then overwritten, and `gc.go` installed nothing at all. Reported against the **Rust** arm by another mod (fklua-ports' AutoDeconstruct, which asked for 128 KiB and ran at 256 with `cycles=0` for a whole verification run), confirmed on Go, and fixed upstream 2026-08-03 by **latching** a non-zero value rather than by a call-ordering rule: `SetThreshold(0)` means "restore the default", so a non-zero field is always something a guest asked for and `initialize()` leaves it alone.

**Which comparison BBB was making, because the answer decides the blast radius.** `gcArmIfNeeded` compares `Stats().SinceGC` against its own constant `gcArmBytes`, **not** against the collector's live threshold — which is the shape upstream describes as disagreeing with the collector by construction. It did not, and the reason is one line of arithmetic: `gcArmBytes` is `256 << 10` and `fkgc.defaultThreshold` is `256 << 10`, so the value the collector was left holding after discarding the install is exactly the value the install asked for. **The arming was coherent by accident**, and the accident is that this mod deliberately chose fkgc's own default as its constant instead of a number of its own. A guest that had tuned the knob — which is the entire reason the knob exists — got the divergence instead.

**What changes now: nothing, measured twice rather than argued.** The whole guest-side delta of the upstream fix round is 39 lines in `fkgc/heap.go` and `fkgc/collect.go`, all of it the latch. Both A/Bs are the shipped `--gc=collected` guest against the `mar` suite, same save, same schedule:

| `mar`, collected arm, 680 operations | linear memory | live set | cycles / grows / steps / deadlines |
|---|--:|--:|--:|
| FkLua guest at `0966a9e` (**no latch**) | 0.71 MiB | 16,448 B | 6 / 11 / 779 / 6 |
| FkLua guest at `8328020` (**latched**) | 0.71 MiB | 16,448 B | 6 / 11 / 779 / 6 |
| ...and with the **pre-F2 bindings** on the latched guest | 0.71 MiB | 16,448 B | 6 / 11 / 779 / 6 |

Byte-identical in all three, which is the expected result and is worth having as a measurement rather than as the inference: the third row is there because F2 regenerated 5,640 lines of bindings in the same round, and "nothing moved" has to mean nothing rather than two changes cancelling. The **leaking** arm cannot see any of this — `fkgc` is an empty package there — and its seven slopes came back identical to the byte as well (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB), which is the control on the whole round.

**So the latch buys this mod a property it did not have rather than a number.** An upstream bump of `defaultThreshold` used to split the two silently — the collector moving to the new default while `gcArmBytes` stayed at 256 KiB — and now `init` moves the collector back and the two still agree. The comment in `gc.go` claiming they "cannot drift apart on an upstream bump" was describing the fix rather than the code, one round early.

**And the `mar` collected-arm figures moved anyway, which is the honest part.** Today's are the table above; the ones this file recorded on 2026-08-02 were 0.77 MiB, a **8,960 B** live set, and 6 / — / **29** / **2**. None of that movement is this round's — the three-way A/B says so — and none of it is BBB's own allocation behaviour either, because the leaking arm's slopes did not move by a byte across the same commits. What is left is upstream's own gc work between the two recordings (the escape rework and the paced grow, both 2026-08-02) and the fact that **the collected arm was never re-measured across BBB's last four commits**; the final-verification paragraph restates the LEAKING slopes and is silent about this arm. Everything is far inside the suite's ceilings — 0.71 MiB against 4 MiB, 16,448 B against 256 KiB — and the outrun line is still exactly 1.

**The re-attribution, with evidence, because the old one no longer describes the data.** The deadlines were recorded as firing in legs C and G, "the two teardown-and-rebuild legs, which are the highest WRITE rate the suite has". Today there are six and they fire in **B (i=8), C (i=33), D (i=43), G (i=24), G (i=72) and F (i=86)** — and **leg D is the cheapest leg in the suite**, a belt laid eighteen tiles from anything and picked up again, 16 B per primitive. A write-rate escape does not fire there. What fits the spread is that a deadline is attributed to whichever leg was *running* when the counter moved, and a collection started under one leg's write rate is still marking when the schedule has moved on to the next — so the leg column locates a mark termination, not the mutator that caused it. The write-rate reading is retired as unsupported rather than replaced: pinning it down needs a per-collection trace the guest does not emit. It stays a non-defect on the same grounds as before — six unbudgeted mark terminations over a 16,448 B live set, on a suite doing an edit every 6.8 ticks, about 75× the busiest rate the projection models — and **no budget knob is set here**, for the same reason.

### The transient that grew while nobody was looking

**The zero-script steady state holds, exactly, in both arms — and the collected arm's post-load transient is 14× what "The third decision" recorded.** Found on 2026-08-03 by re-running the two headline cells against their own in-session controls, which is the check that decision asked for and which nothing since had done. Method as always: `bench/run.sh`, n=200 k=4 express, three interleaved reps of `bbb saturated / none control / bbb idle / none control-idle`, per-tick `scriptUpdate` read from the verbose pass rather than from `max_ms`. Rows tagged `gcround6*` in [`bench/baselines/results.tsv`](../../bench/baselines/results.tsv).

**The steady state first, because it is the headline and it is intact.** Taken after the transient ends, per tick:

| n=200 k=4 express, steady state | control | **bbb, collected** |
|---|--:|--:|
| idle `scriptUpdate`, median | 0.21 µs | **0.2 µs** |
| saturated `scriptUpdate`, median | 0.21 µs | **0.3 µs** |
| saturated `wholeUpdate`, median | 467.9 µs | **470.8 µs** |
| `[BBB]` lines in the benchmark window | — | **0** |

A finished balancer still runs no script. That is the property, unchanged since M4, and it is measured here at the *median tick* rather than averaged over a window that contains a warm-up.

**And the leaking arm is its own control on the whole round**, which is what makes the next table attributable to the collector and to nothing else: `script 1.36 µs`, `whole 169.20 µs`, `avg 0.1795 ms/tick` — against the 1.36 and 168.98 the round-5 re-check recorded for the same cell. Byte-for-byte the historical numbers, so nothing in the guest, the bindings or the packaging moved.

**The transient, which did move:**

| n=200 idle, collected | recorded 2026-08-02 | **2026-08-03** |
|---|--:|--:|
| ticks of collector script after a load | 71 | **985** |
| that transient's total `scriptUpdate` | 65–71 ms | **~906 ms** |
| per-tick median inside it | 838 µs | ~950 µs |
| `t0` | 5.33 ms | 2.95 ms |
| what follows it | the control's | **the control's** |
| the same run on a **pre-latch** FkLua guest | — | **983 ticks, 915 ms** |

**Three A/Bs say it is not the 2026-08-03 round.** The pre-latch guest reproduces it to within noise (983/915 against 985/906); the `mar` suite's collected statistics are byte-identical across latched, unlatched and pre-F2-bindings builds; and the leaking arm matches its own historical record. What is left is BBB's own **item-placement policy** (2026-08-02, "A recompile is not a removal"), which took the 4×4 compile term from 1,517 B to 3,736 B — 2.5× the allocation inside the one `bbb-audit` dispatch that a 200-network `--create` is — and which was never bench-re-measured in this arm; the final-verification paragraph of that pass restates the LEAKING slopes and is silent about the collected bench. That is a hypothesis with a mechanism and a matching date, and it is **not** measured: confirming it means building the pre-policy guest, which this pass did not do.

**What it does and does not change.** It does not reopen the mode decision on its own: the stall class collected was chosen to remove is a **782 ms single tick**, and this is 0.9 ms per tick for sixteen seconds — diffuse warm-up against a freeze every client feels at once, and the steady state is still a wash. It is also **structural to a mass-builder**: `--create` compiles 200 networks in one dispatch where no paced step can run, which is the shape agents/gc.md names and the reason the transient exists at all; a save built by play never allocates 1.3 MB between two ticks. What it does change is that the 71-tick figure was one of the three gates the flip was decided on, and it is now a 985-tick figure that nobody re-measured for a day. **Anything that reopens this should re-measure rather than re-argue** — starting with the pre-policy guest, which is the one term this pass named and did not test.

> **ANSWERED 2026-08-03, AND THE HYPOTHESIS WAS WRONG.** The pre-policy guest was built and it reproduces the 982-tick transient exactly, on a save whose heap is byte-identical — so the item-placement policy is exonerated, and a `--create` never runs the carry path at all because it tears nothing down. The cause is this guest's **globals** crossing 16 KiB, which is exactly one paced step's budget, after which the collector's mark phase can never afford to finish. Everything above is kept as it was reasoned; the section **"The root scan that could not fit in a step"** below is the attribution, the fix and the re-priced gate. The transient is **36 ticks and 54 ms** now.
