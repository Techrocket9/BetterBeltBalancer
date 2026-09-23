# The root scan that could not fit in a step — the 14× transient, attributed

**The hypothesis above was wrong, and the measurement that disproved it took one build.** The pre-policy guest was built and run: its post-load transient is 982 ticks, exactly like the shipped one, and the heap the save is written with is **byte-identical** — `sys=1420696 alloc=1323008 heap=1388544 grows=20` on both. The item-placement policy allocates nothing extra during a `--create`, which in hindsight is obvious: a create builds 200 networks and tears down none, and carry only runs when something is drained. **The 2.5× compile term is real and it never reaches this measurement.**

**What the 14× actually is: this guest's GLOBALS crossed sixteen kilobytes, and sixteen kilobytes is exactly one paced step's budget.** Every mark step ends with a termination attempt that re-scans the guest's whole globals range and charges the scan against the same budget as everything else (`../FkLua/guest/go/fkgc/collect.go`):

```go
gcm.terminations++
gcm.rootWords = 0
gcMarkReachable()
budget = charge(budget, gcm.rootWords<<2)   // granules = rootWords / 4
budget = drainGray(budget)
if budget == 0 || … { return budget }       // no termination this step
beginSweep()
```

`charge` saturates at zero, so the attempt completes only while `rootWords / 4 < budget` — i.e. **globals bytes < 16 × budget**, which at the default budget of 1024 is **4,096 words**. Above it the guest re-scans its roots, spends the entire allowance doing so, fails the `budget == 0` test, and repeats: the mark phase becomes structurally incapable of terminating, and the phase stays 1 until `markDeadline` (`4 × heapGranules/budget + 600`, ≈939 steps for this heap) forces an unbudgeted finish hundreds of ticks later.

## The bisect, and the seven granules

Each row is one `--create` of the n=200 k=4 express idle bench save plus one verbose `--benchmark`, the transient read from the per-tick `scriptUpdate` column, and the collector's own `Stats()` read from a single `bbb-audit` marker placed at tick 1250 — sparse on purpose, because an audit allocates and that is the thing under measurement:

| commit | what it added | rootWords | steps | terminations | deadlines | transient |
|---|---|--:|--:|--:|--:|--:|
| `4a25294` | pre-carry — the third decision's own guest | 4,070 | **71** | 2 | 0 | 67 ms |
| `bb608d0` | **the item-placement policy** — the named suspect | 4,070 | **71** | 2 | 0 | 66 ms |
| `76184bb` | belt stacking | 4,070 | **71** | 2 | 0 | 66 ms |
| `093b0b1` | the miner's pocket | 4,070 | **71** | 2 | 0 | 74 ms |
| **`1a3fb3e`** | **the shrink, and `probe.go`** | **4,278** | **982** | **913** | **1** | **1,005 ms** |
| `e886fc3` | a claim is a Region | 4,278 | 982 | 913 | 1 | 1,065 ms |
| `5fb61c1` | the threshold latch | 4,278 | 982 | 913 | 1 | 1,041 ms |

**`marked` is 95 objects on one side of that line and 101 on the other, the live set moves by 224 bytes, and the heap does not move at all.** Nothing about the work changed; what changed is that a step could no longer afford to ask whether it was finished. `terminations` — 2 against 913 — is the whole defect on one counter, and it is the counter nothing was printing.

**The margin on the good side was SEVEN GRANULES.** At `rootWords = 4,070` the root scan costs 1,017 of a 1,024-granule budget and leaves 7 for the rest of the attempt. This mod ran five milestones, shipped a `-gc` decision and passed seven suites in both arms **104 bytes** from a cliff nobody knew was there, and `1a3fb3e` — two package-level variables, `probe.go`'s search filter and the pocket's claim store — stepped over it.

## The fix: the budget is the other half of a contract this guest was in

`guest/go/gc.go` installs `fkgc.SetBudget` beside the `SetThreshold` it already installed, and the constant is derived from the root scan rather than from the pause: **an allowance of real work, plus a budget for this guest's own roots.**

**`SetBudget` is a latched knob for the same reason `SetThreshold` is**, and that is not a coincidence worth passing over: `initialize()` fills only zeroed fields, so this fix would have been silently discarded by the exact defect the previous commit fixed upstream. The round that found that latch is the round this fix could first be written in.

`gcCheckRoots` was the second half, because the version of this defect that shipped was invisible — seven suites green, every item count identical, a collection quietly taking fourteen times as long. It read `fkgc.RootWords()` on the first flush after a completed collection and logged a `[BBB] error:` when the roots grew back to three quarters of the budget. **It is deleted, and what deleted it is upstream doing the same job better** — see "The floor upstream built, and the check it retired" below.

## What it costs and what it buys, measured

Two reps of each arm, interleaved `collected / leaking / regressed` in one session, n=200 k=4 express idle, no meter and no probe, everything read from the verbose pass's own per-tick columns:

| | **fixed, collected** | fixed, leaking | `5fb61c1`, collected |
|---|--:|--:|--:|
| post-load transient | **36, 36 ticks** | 0 | 982, 982 |
| its `scriptUpdate` total | **54.0, 53.5 ms** | 0 | 1,021.9, 1,061.0 ms |
| its median / worst step | 1.47 / 2.85 ms | — | 1.01 / 2.52 ms |
| `t0`, the load tick | 4.93, 5.06 ms | 2.26, 2.54 | 4.53, 4.70 |
| steady `scriptUpdate` | **0.05 µs** | 0.04 | 0.04, 0.05 |
| steady `wholeUpdate` | 224.9, 223.8 µs | 215.9, 220.4 | 207.6, 200.9 |
| steady worst tick | 1.24, 1.26 ms | 1.31, 2.09 | 1.34, 1.27 |
| shipped zip | 291,364 B | 258,320 B | 290,455 B |

**19× on the transient, and it lands below the 71 ticks / 68 ms the mode decision was priced on** — the budget halves the number of sweep steps while doubling each one, so the total work is unchanged by construction and what actually moved is that the mark terminates on its first attempt. The steady state is untouched in every column, `scriptUpdate` is 0.04–0.05 µs in all three arms, and there are zero `[BBB]` lines in any benchmark window.

**The leaking arm did not move by a byte**, which is the property `gc.go` has had since it existed: `fkgc.Enabled()` is a compile-time false, so both the `SetBudget` and the whole of the check beside it are eliminated. `fk_module.lua` is 1,760,312 B before and after and the two files differ in exactly one line — the build stamp — and the `mar` suite's seven slopes came back identical to the byte (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B, 3.92 MiB).

## The floor upstream built, and the check it retired

**Everything above was measured against a collector that no longer exists, and the round that replaced it (FkLua `48fb51b` + `7347721`, 2026-08-03) moved this guest's constant and deleted its check.** Two changes upstream, in order:

1. **`EffectiveBudget()` floors the budget** at `rootScanCost() + 64`. A guest whose globals cost more than one step no longer fails to terminate — the collector raises its own allowance and **logs one `fkgc:` line saying so**, naming the cause and stating in terms that `SetBudget` is not the fix. **The cliff this whole section is about is gone.**
2. **The scan's cost is RESERVED rather than merely afforded.** `markStep` holds `rootScanCost()` back, gives the queues what is left, and adds it again at the attempt. So a step spends at most `budget` **in total**, of which the scan is a part.

**Change 2 is why the number had to go UP rather than away**, and it is a change in what a budget MEANS rather than a regression in anything. Before it, the scan was charged after the queues had spent; now it is held back before they start. At the shipped `1024 + 1024` that took this guest's real work per step from ~2,048 granules to ~930 — and the `mar` suite, which asserts `deadlines=0`, started failing. Measured 2026-08-03, 680 world operations, everything else held:

| `SetBudget` | `budget` / `eff` | paced steps | deadlines | |
|---|--:|--:|--:|---|
| none — the default | 1024 / **1180** | 66 | **7** | the floor BINDS |
| `1024 + reserve` | 2048 / 2048 | 12 | **3** | what shipped |
| **`4096 + reserve`** | **5120 / 5120** | **5** | **0** | **shipped now** |
| `8192 + reserve` | 9216 / 9216 | 3 | 0 | |

**The default row is the one that answers "is the manual budget redundant now". It is not.** The floor makes the default *correct* — it can terminate — and it does it by leaving 64 granules of real work per step, which is why 66 steps and 7 deadlines. What the manual budget buys is not correctness any more; it is enough real work per step that the DIRTY RATE cannot outrun the mark.

**And `eff == budget` on every other row is what says which cause this is.** Upstream states the diagnostic rule and this guest now logs both halves of it — `logHeap` prints `budget=`, `eff=` and `roots=`:

> if Deadlines rises, compare `EffectiveBudget()` against `Budget()` FIRST. Equal means [the dirty rate] and this is the knob. Larger means [the root set], the collector has already applied the floor.

4096 granules of real work is also **upstream's own documented remedy for this exact symptom** on another mod (`agents/guests.md`: nixie-tubes, "the default gave 15 outruns and 3 mark-termination deadlines, and `fkgc.SetBudget(4096)` gave a clean plateau with neither").

**`gcCheckRoots` is deleted, and the reason is a principle rather than a tidy-up.** It existed because the condition was invisible and nothing reported it. Upstream now reports it, from the only place that can — it is the only component holding both `rootWords` and `Budget()` — and states the rule this guest was violating: *a condition only one component can observe is that component's obligation to report*. Keeping a second implementation of the same warning would be a copy that can drift from the authority. What replaces it as a GATE is one grep in `test/run.sh`: the run fails on the collector's own root-set line, so if this guest's globals ever outgrow `gcRootGranules` again the suite says so rather than a benchmark noticing months later. The other `fkgc:` lines are deliberately not fatal — an outrun is a statement about the allocation rate that a stress suite may legitimately provoke, and `mar` asserts `deadlines=0` over them directly.

## The half that is not about a benchmark, and the re-attribution it forces

**The bench transient is the symptom that got measured; the disease was in ordinary play, and the `mar` suite had been reporting it as a number for a day.** Same suite, same 680 world operations over 4,600 ticks, collected arm:

| | `5fb61c1` | **fixed** |
|---|--:|--:|
| collections | 6 | **8** |
| paced steps | **779** | **14** |
| forward-progress deadlines | **6** | **0** |
| linear memory at the end | 0.71 MiB | **0.59 MiB** |
| live set at the end | 16,448 B | **9,216 B** |

Eight collections finishing in fourteen paced steps, against six that burned 779 and were ended by the deadline six times. The live set nearly halves because a mark forced to terminate by a deadline retains what it had not got to; the linear memory follows it.

**And this retires the write-rate attribution above.** The pacer section explains the `mar` suite's deadlines as legs C and G outrunning the collector with their write rate, and closes with "no budget knob is set here, because setting one on this evidence would be tuning against a fuzzer". The evidence was misread: the deadlines were not a write rate at all, they were the same root scan, and the budget knob was not a tuning parameter but the missing half of the guest's configuration. The reasoning was sound and rested on a counter (`Deadlines`) whose only documented cause is a mutator that outruns the collector — which is what `SetBudget`'s own header says, and it is wrong in exactly this case. **The general lesson is the one this file keeps relearning: a non-zero count of a thing that is documented to be zero forever is a defect report, and this repo had six of them in a suite it runs on every pass.**

## What was NOT done, and why

**A synchronous `fkgc.Collect()` at the end of the audit's flush** — so a `--create` writes its save with a swept heap and the post-load transient collects nothing — was the obvious candidate and is measured-unnecessary. It would take 36 ticks to 0, and it would do it by putting a full unpaced collection inside the one path a test harness controls, which is instrumenting the benchmark rather than fixing the mod. The budget fix reaches the same gate from the other side and fixes every collection a *player* ever causes, which the audit-path collect would not have touched at all. `gc.go`'s standing argument for why a collection does not belong in the audit path is unchanged and still correct.
