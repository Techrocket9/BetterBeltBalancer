# Build

```sh
make guest    # TinyGo -> dist/bbb.wasm.  QUIET=1 drops the [BBB] log lines
              # The default is GC=collected, FkLua's paced collector, and it
              # is the SHIPPED build; GC=leaking builds the other arm, which
              # stays green on all seventeen suites. See "The third decision"
              #
              # AND dist/bbbdata.wasm, the DATA GUEST, whose flags are its own
              # and do NOT move with GC=: -gc=leaking always, because a data
              # module runs once at load and dies with the Lua state that built
              # it. It takes no GC stamp, so both arms share one data module
make mod      # fklua mod: identity, deps, mod-data/ and the DATA MODULE all
              # from fklua.toml. The stage files (settings.lua, data.lua,
              # data-final-fixes.lua) are GENERATED from the data guest's
              # exports, one per hook it has
make zip      # the same, as dist/<name>_<version>.zip -- a complete
              # installable mod, both guests included
make install  # into $MODS_DIR (defaults to the Factorio user mods dir)
make test     # headless verification, SEVENTEEN suites, and the DEFAULT is all
              # of them. WHICH FACTORIO IS ON THE MACHINE IS AN INPUT: this mod
              # ships on two engine arms out of one tree, test/run.sh reads the
              # series off the binary and stamps every staged mod's info.json
              # for it, and three suites answer differently on each -- `mig21`
              # and `mig` INVERT (see their sections) and `flip` runs on 2.0
              # only and prints a SKIP on 2.1 rather than passing. The packaged
              # mod is GATED rather than stamped: its bindings are pinned to one
              # API and the ABI marshals event payloads BY NAME, so a mismatch
              # with the binary is a defect and is reported as one.
              # (m1 m2 m3 upg curv prio plat mar edge mix mig qual sedge
              # mig21 flip iact curs
              # -- plat is
              # the only one needing Space Age, and carries the platform rig,
              # the belt-stacking leg and the stacked-sushi band; mar and edge
              # are the marathon pair; mig is the only one whose two phases
              # run under DIFFERENT MOD SETS, and is seven legs plus two
              # create-only name probes; qual runs base plus the quality
              # mod, every part in it uncommon; sedge is Factorio 2.1's
              # one-belt-per-part rule and the three ways of breaking it; and
              # mig21 is the only one with NO --create phase at all, because
              # its worlds were built by a 2.0.77 binary that is gone and the
              # committed fixture IS phase one; and iact is the only one that
              # is not about this mod's behaviour but about the INTERACTIVE
              # checklist's staged world, which it gates with a single
              # --create; and flip drives `bbb-multi-edge-parts` through all
              # four of its transitions, which only Factorio 2.0 has; and
              # curv is the only suite whose two phases are two BUILDS rather
              # than one build wearing two stamps -- see `make prestate`;
              # and prio is OUTPUT PRIORITY, sixteen rate rigs at the loads
              # that separate the two tiers, the toggle, the four refusals
              # and the spill guard from both sides, in two legs of which the
              # second runs under `bump_build` because on a fresh heap the
              # only place a part's flag survives is its own
              # `graphics_variation`; and curs is the SECOND suite with no
              # --create, and the only world in the estate a map generator
              # could not have made: what it needs is a PLAYER, and a
              # game.players entry exists only where somebody once connected,
              # so its phase one is a committed save a graphical client made
              # -- test/fixtures-player/, cut by `make player-fixture`)
make prestate # THE PRE-STATE FIXTURE, and the one thing in this build that is
              # a second full guest. The same source with `-tags prestate`, so
              # that `fk_state_version` reports 0 the way every build up to
              # 0.3.2 did by not exporting it at all, packaged under
              # dist/prestate/. `curv`'s create phase stages it, which is what
              # makes the save it writes one that predates the curve rule.
              # `make test` depends on it; `make mod` and `make zip` do not,
              # and it is never installed. About eight seconds
make check    # the SIX pure packages' unit tests (plan, skin, carry, tune,
              # edgemode, engine) -- ./tune is where BOTH FkRecipes PLANS are
              # pinned field by field on the host, the settings one against a
              # one-method Named stub and the data one against a fixture World,
              # and where the locale file is checked by FkRecipes' own
              # `CheckLocaleWith` plus BBB's three extra assertions (a
              # description on all six settings, a name on the hand-rolled
              # bool, and the grandfather message quoting a menu label that
              # exists); `go vet` over the data guest, which is the one package
              # no `go test` can reach (//go:wasmimport is rejected outside
              # GOARCH=wasm) -- and it needs NO build tag since round two, the
              # library shipping host stubs for its three emit entry points;
              # bindings and lock current; gofmt
make datastage-check
              # THE DATA STAGE'S OWN GATE, and deliberately not part of
              # `make test`: Factorio's `--dump-data` runs the settings and data
              # stages and STOPS BEFORE control.lua, so it answers a question no
              # suite can ask. TWENTY-FOUR ARMS SINCE THE COLLISION-DEFAULT
              # FIX, two mod sets HASHED against goldens, sixteen
              # VARIANT arms (one per
              # non-default value of each cost setting plus the eight
              # customizer states, driven through a mod-settings.dat the gate
              # writes)
              # asserting the exact ingredient list, the exact research unit
              # and prerequisite, or a line of the library's own log, one
              # SPEED arm that builds a Go fixture
              # mod defining a belt faster than the network's floor -- because
              # no mod set this machine can install has one -- one
              # COLLISION-DEFAULT arm behind a Lua fixture that rewrites
              # `default_collision_masks` ahead of this mod, which is the shape
              # that stopped the mod loading beside Cerys-Moon-of-Fulgora, and
              # TWO MERGE
              # arms, one per mod set, behind a Lua fixture that DELETES
              # `transport-belt` ahead of this mod at the data stage and sweeps
              # what that dangles at data-final-fixes, for the same reason
              # turned around, and TWO NOTE arms, one per channel, on the
              # trailing sentence the library composes when a stored value
              # falls back: a recipe's and a technology's, each asserting the
              # note's elements against the engine's 200-byte ceiling. See
              # Verification
make graphics # regenerate the sprite sheet, the icon and the I/O arrows
make observers
              # the TEST OBSERVERS: one wasm and one packaged mod per PORTED
              # suite, into dist/obs. `make test` depends on it. They are built
              # -gc=leaking (an observer has no steady state to pace a collector
              # against) and packaged from a working directory with no
              # fklua.toml in it, so every identity is a flag -- including
              # --api, without which the packager falls back to FkLua's own
              # default pin and refuses the guest. agents/estate-port.md
              #
              # ALL FOURTEEN ARE TINYGO AND THERE IS NO SECOND TOOLCHAIN IN
              # THIS BUILD. mig21's was cargo for one day -- phase 8's parity
              # exercise, 2026-08-25 -- and the arm was reverted 2026-08-26;
              # `make observers` from clean is 28 s and invokes no cargo, which
              # is grepped for rather than assumed. wasm-opt stays a hard
              # dependency of the TinyGo build itself. See "Pure Go"
```

**`make mod` patches nothing.** It used to: `fk_abi.lua` passed `self` to a Factorio method (they are bound closures, so `.` not `:`) and forwarded four argument slots whatever the member declared, so **every host method call failed** with an argument-count error, and `tools/patch-abi-calls.py` rewrote the two lines on the way past. Both halves are fixed upstream ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 8) and the patch script is deleted — against a fixed `fk_abi.lua` it would corrupt what it used to repair. Nothing in `../FkLua` was ever modified.

**The last hand-written host call is gone.** `M.call` used to forward the DECLARED arity of a member, so a trailing optional nobody passed arrived as an explicit `nil` the engine counts and rejects — and `game.create_surface(name)` is exactly that shape, the call this mod's whole architecture rests on. It trims to the last argument actually PRESENT now ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 16, fixed upstream), so `fkapi.Game.CreateSurface(name, nil)` reaches the engine as `game.create_surface(name)` and the 88 lines of `guest/go/host.go` — a member id, an argument-block size and two field offsets, all hand-derived — are deleted along with the four `check-layout.py` guards over them.

**What the mod actually ships, and why 17,341 new lines of bindings cost 2.4 KB.** `gen-bindings` grew `guest/go/fkapi/fkapi.go` from 74,361 to 91,668 lines in the 2026-08-01 round — almost all of it the 1137 `defines.*` accessors — and the pruning is what decides whether that reaches a save. Measured across the round, same flags, same pin:

| | before | after |
|---|---|---|
| `dist/better-belt-balancer_0.1.0.zip` | 137,361 B | **139,762 B** (+1.7%) |
| `fk_module.lua` (the compiled guest) | 869,177 B | 900,786 B (+3.6%) |
| `fk_api_gen.lua` (the API table) | 13,585 B / 64 lines | 13,901 B / **69 lines** |
| defines shipped | — | **4 of 1137** |
| members shipped | 24 of 3,905 | 25 of 3,905 |
| `dist/bbb.wasm` | 363,051 B | 713,899 B |

**The wasm nearly doubled and none of it is code.** Its `code` section went 65,439 → 67,219 bytes; the growth is DWARF — `.debug_pubnames` 23 KB → 177 KB and `.debug_str` 28 KB → 172 KB — because TinyGo dead-code-eliminates the 1133 unused accessors out of the module but leaves their symbols in the debug sections. FkLua compiles the `code` section, so nothing reaches the mod; it is noted because `ls -l dist/bbb.wasm` is a misleading way to watch mod size, and it is the one thing the accessor design costs (see `FKLUA-GAPS.md`).

The TinyGo flags are FkLua's and each one is load-bearing (`../FkLua/agents/guests.md`): `-target=wasm-unknown -scheduler=none -opt=2`, plus the `-gc` the mode stamp moves. binaryen (`wasm-opt`) is a hard dependency of the TinyGo build, not an optional extra.

**`--gc=collected` is the shipped build since 2026-08-02, and the `-gc` mode is a decision rather than a requirement.** Upstream shipped the paced collector, this mod measured it three times, and the answer changed on the third because the measurement did: the steady state cannot tell the arms apart on today's pin, the post-load collector transient fell from 152 ticks to 71, and the thing leaking costs is a **782 ms single-tick `memory.grow` stall** at the 16→32 MiB doubling — measured, not projected. `make GC=leaking` builds the other arm and all seven suites are green in both. Read "The collected-mode postscript" and its four decision sections in order, and **"The marathon save"** for what the flip is about.

**`--persist=packed`, decided three times.** The guest heap is in every save and every multiplayer join, so the mode is a shipping cost, not a preference — and the right answer has changed twice because what it depends on changed upstream twice. The history, because each step explains the next:

| | mode | why |
|---|---|---|
| **M1** | `packed` | save size alone, for a guest that only ever wrote a few words |
| **M2** | `table` | M2 writes a lot, and `packed`'s dirty watermark was a min/max **byte range**: one host call touched the static scratch region and the heap, so a flush repacked everything in between. 41 ms per 4×4 recompile, and a 200-compile build took **447 s** |
| M2+ | `table`, re-confirmed | the marshalling arena ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 10) took a host call from 180 B of leaked heap to **zero**. It stopped the range GROWING; it could not make a span SHORT, so `packed` was still 1.7× the hitch |
| **now** | **`packed`** | upstream replaced the byte range with a dirty **page SET**. The span pathology is gone, and the two costs that are left are not the ones the M2 decision was weighing |

Measured 2026-08-01, Factorio 2.0.77, both modes interleaved in one session so session drift (25–35%, see Benchmarks) cannot bias one side. Every cell is n=200 k=4 express unless it says otherwise; save sizes are the M2 map with `--map-gen-seed 12345` fixed, `stat -f%z`, against a `--persist=none` control.

| | `table` | `packed` | |
|---|--:|--:|---|
| 200-rig `--create` | 31.0 s | 45.8 s | 1.48× — was ~30× |
| 4×4 recompile hitch | 4.11 ms | 5.90 ms | +1.79 ms, ¼ tick → ⅓ tick |
| 8×8 recompile hitch | 9.45 ms | 11.55 ms | +2.10 ms |
| M2 map save delta | +141,413 B | **+14,089 B** | **10.0×** |
| n=200 bench save | 49.4 MB | **3.6 MB** | **13.8×** (control: 0.86 MB) |
| n=200 save **load** | 21.6 s | **8.2 s** | **2.6×** (control: 0.0 s) |
| idle worst tick, median of 15 | 15.00 ms | 14.16 ms | a wash |
| idle worst tick, mean / max | 15.37 / 18.12 | 15.79 / 22.00 | a wash |
| saturated `avg_ms`, mean of 6 | 0.485 ms | 0.497 ms | unchanged |
| saturated `scriptUpdate` | 1.88, 1.93 µs | 1.95, 1.83 µs | unchanged |

**Every recompile-hitch number in this section predates the 2026-08-02 item- placement policy** and is kept as measured, because it is what the `--persist` decision was taken on. A recompile of a network that is CARRYING ITEMS costs more now — it puts them back rather than dropping them — and the before/after pair is in "A recompile is not a removal". Nothing about the persist comparison moves: the extra work is host calls on both sides of it.

**The measurement that decided it is not the one that was expected to.** The pass was run to see whether the page set closed the idle GC spike ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 17) — the standing theory being that `table`'s giant word table in `storage` is what Lua's collector walks. **It did not, and the theory was wrong.** `packed` mirrors the live memory into `string.pack` pages *for the save*; the memory the guest actually runs on is a Lua word table in **both** modes, and that is what the collector walks. Only `storage`'s copy differs. So the GC spike is not a cost of the `table` decision at all and drops out of the comparison entirely — 15.0 ms median under `table` against 14.2 under `packed`, over 15 runs each, with `packed`'s tail slightly worse (22.0 vs 18.1). Against a no-mod control's 1.5 ms, both are the same regression.

What remains, and it is one-sided: `packed` costs **+1.79 ms per recompile** and **+14.8 s on a 200-rig create**, and saves **45.8 MB of save and 13.4 s of load** at the same n=200. The hitch is per edit and stays under half a tick; the create is one-off and nobody builds 200 balancers in one tick; the save and the load are paid by every player, on every load and every multiplayer join, forever. `packed` also wins the M2-map save delta 10×, so this is not only a large-map effect.

**The rule the pass set out with, and why it was overruled.** The rule was: flip if the idle worst tick materially improves AND create stays within ~2× of `table` AND the hitch stays under one tick (16.7 ms). Create passed (1.48×) and the hitch passed (5.90 ms, and 11.55 on an 8×8); **the worst tick failed**, so the rule as written says do not flip. It was overruled because the rule assumed the GC spike was a cost of the mode — that is what made it the gate — and the measurement retired that premise. Once the spike is the same on both sides it is not evidence about the mode at all, and what is left to decide on is 45.8 MB of save and 13.4 s of load against 1.79 ms of hitch and a one-off create. A 21.6 s join and a 49 MB save are shipping-quality defects of exactly the class the rule was written to protect against; they just were not the ones anyone was looking for. Anything that reopens this should re-measure rather than re-argue.

**The 447 s pathology is gone and that is worth stating as a number**: the same 200 compiles now cost 45.8 s under `packed` while building **4× the rigs** the 447 s run built. The clean, drift-proof form is the ratio to `table` — **~30× before, 1.48× now**.

Save sizes are one `--create` per mode compared with `stat -f%z`. Recompile costs are `helpers.create_profiler` around a forced full teardown-and-rebuild, reported by `make test`, **median of three runs, minus that run's own `idle tick pair, nothing pending` control** (0.30–0.49 ms). `--create` times are the Factorio process's own elapsed clock at `Goodbye`; load times are from `Loading script.dat` to the first tick's script line. Worst ticks are `max_ms` from `bench/run.sh --scenario idle`, three cells of five runs per mode, interleaved `table`/`packed`/`table`/…. No other Factorio was running.

**What `+1.79 ms` is made of.** A 4×4 recompile is ~350 host calls, so the delta is **5.1 µs per host call**. Upstream charges ~40 µs per page *actually written* per call, which puts the whole recompile at ~45 dirtied pages — about one new page per eight host calls. That is the marshalling arena doing its second job: consecutive calls allocate into the same page, so the page set has almost nothing to add. Without the arena every call would dirty a fresh page and this delta would be ~14 ms.

**All seven suites pass under `packed`**, which is the round-trip test — the save/reload between `--create` and `--benchmark` is what every suite's benchmark phase depends on. Item throughput and balance are identical between modes at n=200 saturated (1,740,000 items, 1.001), which is what makes the timings comparable at all.

**How that profiler works changed when the guest learned to batch, and the numbers did not.** The compile no longer happens inside the tick that lays the belt, so the probe opens in the mutating tick and closes in the flushing one, with an `idle tick pair, nothing pending` control measured the same way to subtract (0.30–0.49 ms, one engine tick of the M2 save). Post-subtraction, in the `table` mode of the day: **4.39 ms for a 4×4 and 9.63 ms for an 8×8**, against 4.4 and 9.6 before. (Re-measured under `table` in the persist pass above: 4.11 and 9.45 — the same numbers a session apart. The shipped mode is `packed` and pays 5.90 and 11.55.) The alternative — forcing the flush with a `bbb-audit` marker, which is what the item-conservation check has to do because it needs an atomic sample — was measured and rejected *for timing*: the audit re-classifies every cluster in the save, 16 ms of its own against a 5 ms recompile.

**A 4×4 recompile is ~350 host calls and 4.4 ms of them is 12.6 µs per call** (under `packed`, 5.90 ms and 16.9 µs — the same 12.6 plus the 5.1 the page flush costs, above). That is the tier-2 encode, and upstream measures the same shape at up to 14.3 µs through a real guest — `read_dyn` walking the `create_entity` table on the LUA side. Nothing on the Go side moves it: `createArgs` allocates nothing, and hoisting every constant part of that table into `initBuffers` (the keys, the tags, the position array's identity — ~640 bytes of struct copying per call removed) changed the measured recompile by **nothing at all**, 4.0–4.6 ms before and after. The hoist is kept because it is strictly less work; it is not a win, and the win is not downstream.

**`none` became mechanically possible at M3 and is still the wrong choice.** M1 and M2 ruled it out because FkLua has no guest `on_load` hook, and noticing a load seemed to require subscribing to `on_tick` forever. M3 found the way round that — `registryReady` is false in a freshly initialised heap, so the FIRST EVENT of any session rebuilds the registry from the world and adopts the networks already standing ("Coming back on a heap this build did not write" below). That is exactly what `none` would need, every load, and it is measured: 21 clusters and 77 parts re-derived and adopted in ~19 ms.

What rules it out now is **multiplayer**. A client joining mid-game would run that rebuild independently, assign its own node ids in its own order, and hand out its own hidden-surface slots — and the next compile would place a network in a different slot on that client than on every other. That is a desync, not a cosmetic difference. Under `table` the joiner adopts the same heap everyone else has, so every client's ids agree by construction.
