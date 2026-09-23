# The standardization pass — what a reader from `fklua init` would find odd

**BBB is meant to read as what a regular FkLua mod looks like, and it is five milestones and six upstream fix rounds older than the scaffold that now defines that.** This section is the audit: every place this mod differs from what `fklua init <name> --lang go` writes today, or from the two refined `fklua-ports` guests, is either **standardized** or **kept with the measurement that justifies it**. Nothing is left as "that is just how it grew".

Run 2026-08-03 against FkLua at `1f1f576` (six rounds newer than the last integration), the live scaffold output, and `ports/qol-research` + `ports/resource-marker`.

## The stale pair, which had to be fixed before anything could be assessed

`fklua gen-bindings` + `fklua lock`, then a clean rebuild, **before** the audit proper — because the committed bindings predated rounds B1a/B1b/B2 (members 4,191 → 4,250) and **member ids moved**. Verified rather than assumed: `LuaControl.Insert` went from **member id 316 to 317**. Ids are dense indices over the generated set and are only ever meaningful against the `fk_api_gen.lua` emitted from the same generation, so a stale `fkapi.go` packaged by today's `fklua mod` calls a *different function*, silently, with no status and no error. That is the miner's-pocket insert, among others.

**Fallout: one real failure, and it was upstream's semantics rather than a bad id.** The `mar` suite began reporting `deadlines=3` against a documented zero — traced to the collector's new root-scan *reserve*, fixed by re-deriving one constant, and written up in "The floor upstream built, and the check it retired". Every other suite passed unchanged in both arms.

## The table

| what | verdict | why |
|---|---|---|
| `guest/go/` layout (scaffold writes `guest/`) | **KEEP** | `fklua gen-bindings` hard-codes `guest/go/fkapi/fkapi.go` and `fklua lock` hashes that exact path, so the module must contain that directory. **Both refined ports deviate the same way and for the same reason** (known upstream as G5/Q6); conforming to the scaffold would put the bindings at `guest/fkapi` where the lock cannot find them. `guest/go/go.mod`'s header already says this |
| `gc.go` — `SetThreshold(256 KiB)` | **KEEP** | The ports omit it and take the default. BBB installs the same value it arms against, which the 2026-08-03 latch made meaningful: an upstream bump of `defaultThreshold` moves the collector, `init` moves it back, and `gcArmIfNeeded`'s constant still agrees. Without it the two split silently. One call at load |
| `gc.go` — `SetBudget` | **KEEP, RE-DERIVED** | Not redundant, and measured: default = 66 paced steps / **7 deadlines**; `4096 + reserve` = 5 steps / **0**. The upstream floor made the default *correct* rather than *good* — it terminates on 64 granules of real work per step. `eff == budget` is upstream's own test for the dirty-rate cause, which is the one this knob fixes. Full table in "The floor upstream built" |
| `gc.go` — `gcCheckRoots` | **DELETED** | Upstream's `EffectiveBudget()` floors the budget and the collector logs its own `fkgc:` line naming the cause. A second copy of a warning can only drift from the authority that owns the condition. Replaced as a *gate* by one grep in `test/run.sh` |
| `gc.go` — `gcArmIfNeeded` / `requestFlush` | **KEEP — and it is now the convention** | The scaffold's `fk_on_tick` call site is unavailable here: the zero-script steady state is the headline measurement and there must never be an `on_tick`. **Both refined ports carry this exact shape and name BBB as its origin.** Upstream documents the anti-pattern it fixes (a conditional flush starving the pacer) with BBB as the named example |
| `gc.go` — `logHeap` gained `budget=` / `eff=` / `roots=` | **STANDARDIZED** | The first three numbers upstream asks for when `deadlines` is non-zero. This guest used to make somebody derive them |
| `logline.go` | **KEEP** | The heap diet: log lines built with `+` were, measured, the *entire* guest heap — 64 MiB of linear memory, an 18.1 ms idle worst tick. Now under 16 MiB and 2.26 ms. **Both ports copied this file**, so it is house style rather than a deviation. Header already carries the numbers |
| `fklua.toml` — no `gc` key | **STANDARDIZED** | Added `gc = "collected"`. The scaffold writes it, all seven ports carry it, and `fklua mod` refuses a manifest/build mismatch. The Makefile now passes `--gc` only for the *other* arm |
| `fklua.toml` — no `persist` key | **KEEP** | There is no such manifest key in today's fklua; `--persist` is CLI-only. Recorded in the manifest so it reads as a fact about the tool |
| `Makefile` — `GC`/`PERSIST` stamps | **KEEP** | The ports encode the arm in the wasm filename instead; both solve the same make-3.81 problem (whole-second mtimes). BBB needs two axes, not one, because it also carries a `--persist` decision the ports do not have. Left alone rather than churned |
| `Makefile` — `QUIET=1` | **KEEP** | `verboseLog` is a constant, so every line below error level is *eliminated* rather than skipped. The default is verbose because the guest's log lines are the assertion surface for all seven suites |
| `Makefile` — sed'd identity, no identity flags | **ALREADY STANDARD** | Matches the ports exactly |
| dead build steps (`patch-abi-calls.py`, `check-layout.py`, the `make mod` overlay) | **ALREADY GONE** | All three deleted in earlier passes; verified no Makefile hook survives |
| `SubscribeMasked` + `ReadOnUndoApplied` | **ALREADY ADOPTED** | No hand-derived offset remains anywhere in the guest; the only matches for "offset" are historical comments |
| `Game.Surfaces()` — a materialising dictionary read | **KEEP** | B2 added `SurfacesRaw()` + `Get(k)` for *point* queries. This call **iterates** every surface (and finds the hidden one by name inside the walk it already needs), which is exactly the case the ports' rule leaves with the materialising read |
| `fk_migrate` yes / `fk_migrate_adopt` never | **ALREADY STANDARD** | Both ports state the same rule verbatim |
| subscriptions in `init()`, event ids as literal constants | **ALREADY STANDARD** | Verified: `init()` at main.go, one literal id per `Subscribe` call |
| console commands (the B2 seam) | **ADOPTED** | `commands.go`. The audit was reachable only by placing a hidden script-only entity — i.e. not by a player at all. See "The audit has TWO DOORS" |
| `bbb-audit` / `bbb-insert-probe` prototypes | **KEEP** | Test infrastructure that must work headlessly, where no command can be issued. The command is a second door, not a replacement |
| no `red_on.go` / `red_off.go` pair | **NOT ADOPTED** | The ports carry a build-tagged injected defect and a `make red` that requires the suite to fail. BBB red-proves *per assertion, at the time it is written* instead (the belt-stacking leg, the `bmin` tripwire, the operator seam here) and records the result. A permanent red seam is a larger change than this pass; noted as the one convention deliberately left open |

## The verification, and the two gates the pass was not allowed to move

All seven suites green from clean in **both** arms, twice over (once mid-pass, once final), plus `make check`, the sprite checker and `go test ./plan ./skin ./carry`. The leaking arm's `mar` slopes came back at **3.92 MiB** of linear memory, identical to the record — which is the measurement that says a collector-side change really did cost the other arm nothing.

**The two headline bench cells, interleaved with their own in-session controls** (n=200 k=4 express, two reps each, `bbb / none / bbb / none`), reading the steady half of the per-tick verbose pass:

| n=200 k=4 express | control | **bbb** |
|---|--:|--:|
| idle `scriptUpdate` | 1.65 / 1.40 µs | **1.53 / 2.06 µs** |
| saturated `scriptUpdate` | 1.98 µs | **2.67 / 2.75 µs** |
| saturated `wholeUpdate` | 541.70 µs | 711.35 / 758.79 µs |
| balance | 1.000 | **1.001** |

`scriptUpdate` is 1.4–2.8 µs in every cell **including the no-mod control**, one of the idle reps landing below it — the zero-script steady state, unchanged.

**The post-load transient did not regress; it improved.** Measured per tick on the idle save, the same way "The root scan that could not fit in a step" measured it: **17 ticks and 44.2 ms**, against the 36 and 54 that section records and the 71 / 68 the mode decision was priced on. The larger budget does fewer, bigger steps, so the total work is unchanged by construction and the tail is shorter. Steady-state `scriptUpdate` after it is a **median of 0.33 µs** over 1,199 ticks.

Two isolated ~0.85 ms ticks at t=600 and t=1200 are **the harness's, not ours**, and that is checked rather than assumed: the no-mod control save shows the same two spikes at the same two ticks (0.70 and 1.24 ms). An idle balancer allocates nothing and starts no collection.

## What did NOT change, and why that is the finding

**The architecture is not a deviation.** Every quirk this pass examined turned out to be either load-bearing with a number behind it, or already the convention — in two cases (`gcArmIfNeeded`, `logline.go`) because the campaign copied it *from here*. The one thing that was genuinely stale was the bindings, and the one thing that was genuinely wrong was a constant that upstream's own round had moved out from under.
