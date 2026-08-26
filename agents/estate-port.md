# The test estate, in Go

**There is to be no hand-written Lua in this repository, anywhere.** The shipped
mod got there in two rounds -- `fklua mod` has always generated the control
stage, and the 2026-08-25 round replaced ten hand-written data-stage files with a
second compiled guest ([`FKLUA-GAPS.md`](../FKLUA-GAPS.md) item 25). What is left
is the TEST ESTATE, and after the pilot it is **8,524 committed lines** of it:
7,323 across twenty-one files under `test/mods`, 462 in the interactive
checklist's staging mod, 739 in the `bench/` harness's setup mod. Lua that
nothing in `make check` can reach, that no toolchain type-checks, and that only a
Factorio run can execute at all.

This file is the programme for removing it, and the record of what the PILOT
measured. The pilot is `m1` and `sedge`, done 2026-08-25.

---

## What an observer is

A suite's mod is not a test. It **builds a world, drives it on a schedule, and
reports what it sees** -- and then `test/assert-*.py` decides. That separation is
the estate's oldest rule and the port does not touch it:

> The mod under test's own `[BBB]` log lines are the assertion surface. An
> observer that computed the expected answer would be a second implementation of
> the thing under test.

So an observer's own log lines are a **contract with the assertion script**, and
they are the whole risk of this port. `test/assert-*.py` keys every regex on what
follows the `[BBB-...]` tag and never on the `Script @__mod__/control.lua:N:`
prefix the engine stamps in front of it -- which is what makes a guest's
`fk.Log` line satisfy a script written for a Lua mod's `log()` line, PROVIDED
THE TEXT IS BYTE-IDENTICAL. Everything below exists to make that provable rather
than hoped for.

## Two rules an observer does not inherit from the shipped guest

- **`fk_on_tick` is allowed.** The no-tick rule is the product's, and it is the
  whole architecture there: a finished balancer must cost zero script. An
  observer IS a schedule.
- **The heap does not matter.** `logline.go`'s diet, the `-gc` decision and the
  `mar` slopes are all about a guest that lives in a player's save forever. An
  observer runs for seconds in a world that is thrown away.

What an observer DOES keep is the no-entity-references habit, and it keeps it as
a convenience rather than as doctrine: a rig registry holds TILES, and a chest is
re-found on the tile it was built on. `Object.Retain` exists and works -- FkLua's
persistent handle space is in `storage` and Factorio serializes the reference --
so a suite that genuinely wants a handle across a tick can have one, knowingly.

---

## The layout: one Go module, N thin mains

```
guest/go/obs/harness/    the shared kit: surface, placement, lookups, chests,
                         the audit marker, the schedule, the line builder
guest/go/obs/m1/         package main -- one observer, one wasm
guest/go/obs/sedge/      package main
guest/go/obs/sedgedata/  package main -- sedge's DATA STAGE, a second module
```

**Inside the mod's own Go module (`guest/go`), not a module of their own.** That
is what lets the observers import `guest/go/fkapi` -- the generated bindings this
project already commits and `fklua lock` already hashes at that exact path --
instead of vendoring a second copy that could drift from the pin the mod under
test was built against. It is the same decision `guest/go/data` made: two `main`
packages in one module.

**The objection to answer first was whether an observer can bloat the shipped
mod.** It cannot, and that is measured rather than argued: pruning is per WASM
MODULE at package time, so what an observer calls never reaches the mod's member
table. Forcing a full re-package of the mod with `guest/go/obs` present:

| | before obs/ existed | after |
|---|---|---|
| `fk_api_gen.lua` sha256 | `5c5613b6fd5c3b07…` | **identical** |
| members / events / defines | 54 of 4859, 23, 4 | **54 of 4859, 23, 4** |

The Makefile's `GUEST_SRC` excludes `guest/go/obs` for the same reason it
excludes `guest/go/data`: a change to an observer must not relink the mod, and a
change to the mod must not re-package fourteen observers.

## The build recipe

```make
$(DIST)/obs-%.wasm:      tinygo build -gc=leaking -opt=2 ... ./obs/$*
$(OBS_M1_DIR):           cd $(OBS_DIST) && fklua mod ... -o .
```

Three things about the package step, and each of them was learned the hard way by
something in this repository:

- **The packager runs from `$(DIST)/obs`, which has no `fklua.toml` in it.**
  `fklua mod` reads the manifest in its WORKING DIRECTORY for every identity it
  was not given a flag for, so packaging an observer from the repository root
  merges this mod's `data = "mod-data"` asset tree and its `gc = "collected"`
  into a test mod. `test/check-datastage.py`'s fixture builder hit this first;
  same trick, same reason.
- **Every identity is a flag**, including `--dependency`, which REPLACES the
  manifest's list rather than adding to it. The two observers carry exactly the
  dependency lists their `info.json` files used to.
- **`--api=$(MOD_API)` is not optional.** With no manifest in the working
  directory the packager falls back to FkLua's own default pin (2.0.77 today),
  and a guest built against 2.1.16 bindings packaged against a 2.0.77 member
  table calls DIFFERENT MEMBERS -- ids are dense sorted indices over one
  description's set. `fklua mod` refuses it outright, which is how this was
  found; the flag is what makes the refusal unnecessary.

`--gc=leaking` (an observer has no steady state to pace a collector against) and
`--persist=$(PERSIST)` (a rig registry is written in `fk_on_init` during
`--create` and read during `--benchmark`, so it crosses the save exactly as the
mod's own heap does).

## The staging

`test/run.sh` grew one helper and one gate, and nothing else in it moved.

**`copy_testmod <name> <dest>`** stages a PACKAGED observer out of
`dist/obs/<name>_*/` when one exists and a Lua directory out of
`test/mods/<name>/` when it does not, into `$work/mods/<name>` either way. Every
existing path keeps working, `stamp_engine` keeps working (an `info.json` is an
`info.json`), and porting a suite is a Makefile recipe plus a `git rm` rather
than an edit to the runner. The version is globbed because it is the observer's
own and has nothing to do with this mod's.

**`guest_gate` fails a run on `[BBB-OBS] error:`**, which is the thing the port
took away. A hand-written test mod called Lua's `error()` when a rig failed to
land, which aborts `on_init` outright -- the correct severity, because a rig that
is not there makes every number after it a measurement of a different world. A
GUEST CANNOT DO THAT: there are no coroutines under a wasm frame, so nothing can
unwind. The honest equivalent is a line the runner refuses to see, and one tag
serves all fourteen observers. `harness.Fatal` also names the host's own reason
through `fk.LastError()`, which the `error()` it replaces never could.

### Stamping a guest, which is new and which is now measured

`test/run.sh`'s header draws a hard line: a test mod is **stamped** for the
running engine series and the packaged mod is **gated** against it, because the
mod is a guest compiled against a pinned API whose ABI marshals event payloads BY
NAME. A ported observer is now also a guest compiled against a pinned API, so the
asymmetry has to be defended rather than assumed. `fklua api check`, over the
gap between the two engine arms this repository ships on:

| guest | surface | 2.1.16 -> 2.0.77 |
|---|---|---|
| `obs-m1.wasm` | 12 members, 1 event, 10 concepts | **clean**, 0 findings, exit 0 |
| `obs-sedge.wasm` | 17 members, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |
| `bbb.wasm` (the mod) | 54 members, 23 events, 14 concepts | **impacted**, exit 1 |

and the mod's two findings are exactly the shape the header describes:
`LuaRendering::draw_sprite` loses a `light_mode` parameter and
`on_player_rotated_entity` loses a `previous_mirroring` field. **An observer
subscribes to one event and calls a dozen members**, all of them ancient, which
is what makes stamping it safe -- and that is now a number rather than an
argument. **Every phase must re-run this check for the observers it ports and
record the verdicts**; a suite whose observer comes back `impacted` may not be
stamped, and needs its own answer.

---

## The gates a phase must clear

In this order, and the first one is the one that cannot be recovered later.

1. **GOLDEN LOGS FIRST.** Run the suites on the unmodified tree and keep both
   phases' logs. Do not port a line before they exist.
2. `make check` -- `gofmt` covers `guest/go/obs` for free, and `go vet` over
   `./obs/...` is what type-checks a `main` package full of `//go:wasmexport`
   that no `go test` can reach.
3. The suites green with the ported observers.
4. **THE GOLDEN-LOG DIFF**, line for line, with only genuinely nondeterministic
   values masked and every mask justified. An unexplained diff is a port defect.
5. **A RED PROOF with teeth**: perturb one ported log line's format and show the
   assertion script fails. This is what says the assertion surface is really
   being exercised against the guest's own lines.
6. The **whole estate**, both `-gc` arms, one invocation each. The other suites
   are what say the `run.sh` changes broke nothing.

### What the pilot's golden diff came to

Normalised on exactly two things, both of them justified below, and then diffed:

| masked | why |
|---|---|
| the elapsed-seconds column Factorio stamps on every line | wall clock |
| `Script @__mod__/control.lua:N:` -> `Script @__mod__/control.lua:` | WHERE in a mod's `control.lua` the `log()` call sits. A guest's lines all come out of one generated line; a Lua mod's come from wherever its author put them. Nothing anywhere asserts on it -- `test/assert-*.py` selects on the `[BBB-...]` tag |

**Every one of the 320 `[BBB]`, `[BBB-TEST]` and `[BBB-SEDGE]` lines across both
suites and both phases is byte-identical.** 44 + 106 in `m1`, 100 + 70 in
`sedge`. Not "equivalent": identical, in order.

What is left over is six lines per log and not one of them is behaviour:

| difference | what it is |
|---|---|
| the run's start timestamp, and the free-disk figure | wall clock |
| `Checksum for script __<observer>__/control.lua` | the observer's control stage is a different file. That IS the port |
| `Checksum of bbb-sedge-test` | its data stage is a different file too. `m1` has no data stage and its mod checksum is unmoved at 0 |
| `Loading script.dat: 223,462 -> 453,863` (m1), `261,003 -> 634,750` (sedge) | the observer's guest heap is in the save now, where a Lua mod's `storage` tables were |
| the benchmark's `checksum:` and its ms figures | Factorio's state checksum covers every mod's `storage`, and an observer's `storage` is a guest heap now. The WORLD is what the 320 identical lines describe |

**`Checksum of better-belt-balancer` and `Checksum for script
__better-belt-balancer__/control.lua` are identical in every log**, which is the
same statement the member-table hash makes, from the other end.

### The pilot's red proofs

Two, one per suite, each firing in its own family and nowhere else:

| injected | what fired |
|---|---|
| `m1`: the token `phase=` renamed to `phaseno=` | *"phase 1 never ran"* through *"phase 9 never ran"* -- that token is what groups the entire log into phases. `run.sh` exit 1 |
| `sedge`: one space inserted, `tick=` -> `tick =` | the three rate windows, *"the settled window was not sampled at both ends (e, f)"* and its two siblings -- that token is what identifies a sample. `run.sh` exit 1 |

---

## The harness surface

`guest/go/obs/harness`, and what is in it is what the two pilots actually shared.
It is deliberately not a framework: every one of these was a copy-pasted block in
each of fourteen `control.lua` files, and the estate audit counted six-way
duplication of the first three before a line was written.

| | |
|---|---|
| `Flat{...}.Make()` | the scratch surface: bounded, always-day, peaceful, no water, no cliffs, one autoplace tile; chunks requested and FORCED; a box swept of everything but a character, de-decorated, and paved. One box for the sweep and the pave, because the Lua wrote the same numbers twice |
| `Surface(name)`, `Tick()` | lookups |
| `Center`, `TileBox`, `InnerBox`, `Box`, `XY` | tile geometry, and the type a rig registry holds |
| `Piece` / `Place` | one `create_entity` at a TILE, with `Dir`, `Type`, `Force` and `Raise`; Fatal when nothing comes back |
| `Audit(s, x, y)` | the `bbb-audit` marker, and the ONE place a nil return is expected -- see below |
| `FindOnTile`, `FindExactlyOne`, `NamesOnTile`, `SortStrings` | lookups, and the sorted name list a `holds=[...]` line needs |
| `ChestCount(s, name, x, y)` | a chest total, `-1` when there is no chest -- the estate's own convention, so a missing sink cannot read as "nothing was delivered" |
| `Step` / `Run` | the tick schedule, a slice and a linear scan |
| `Line` / `Fatal` | the log-line builder, and the observer error level |

### The line builder, and why it is not `fmt`

`harness.Line` is the shipped guest's `logline.go` with the heap argument taken
away and the real reason left standing. **The line text is the assertion
surface**, so a builder with one `S`/`U`/`I` per field is the shape in which a
transcription from Lua can be read against the Lua it came from, field for field.
`fmt` would hide exactly that behind a format string, and would link TinyGo's
reflection into a guest with no other use for it.

### Two deviations the pilot made, both recorded rather than hidden

- **`cliff_settings` is not sent.** The Lua passed `cliff_settings = { richness =
  0 }`, leaving the rest of the concept to Factorio's defaults. The generated
  `CliffPlacementSettings` is a struct whose `name` and `control` are
  non-optional Go strings, so writing one would send `name = ""` -- a cliff
  prototype that does not exist -- rather than omitting the field.
  `property_expression_names = { cliffiness = "0" }` is what actually produces no
  cliffs, it was already in every one of those settings blocks, and the sweep
  destroys anything that reached the box regardless. (`water = 0` is not sent
  either, and never was read: it is not a member of the MapGenSettings concept in
  2.x.)
- **Two lookups use a tighter box.** `sedge`'s Lua found its rotate belt and its
  bridging part through `{{-0.4, y+0.1}, {0.4, y+0.9}}`, an x-range that would
  also have matched tile -1. `harness.FindOnTile` uses the tile's own box. The
  result is identical -- nothing stands on tile -1 in those rows, and the
  `tile tag=` lines report what is on the tile in question at every sample -- and
  the tighter question is the honest one.

### The one thing the port found that the Lua had silently relied on

**The `bbb-audit` marker destroys itself from inside the `script_raised_built`
that `raise_built = true` dispatches**, so by the time `create_entity` returns
there is no entity to hand over: measured here, the call comes back with no
object and no error at all. The Lua never looked at the return; the first cut of
`harness.Place` did -- as it must for every other piece, because a rig that did
not land makes every number after it a measurement of a different world -- and
all eight of `sedge`'s markers came back as `[BBB-OBS] error:`. `harness.Audit`
deliberately ignores it, and says why.

**And R3 -- the audit drain crossing two wasm instances -- WORKS.** `sedge`
places eight markers, and the mod's own `[BBB] audit clusters=... nets=...` line
lands between the observer's two lines every time, with all eight tuples
identical to the golden. Two separate FkLua modules, two separate linear
memories, one Lua state: the observer's guest calls `create_entity`, the engine
raises the event before returning, Factorio calls the mod's `control.lua`, which
calls the mod's guest, which drains its deferred queue and logs -- and the
observer's own host call has not returned yet. It was the pilot's largest
unknown; it is now a measurement.

---

## What the pilot cost

Build, on this machine, `-opt=2`:

| | |
|---|---|
| first observer wasm, cold Go build cache | 3.9 s |
| each additional observer, warm | 1.8 s |
| the data-stage wasm | 0.3 s |
| `make observers` from clean, both packages | **4.1 s** |
| ...with the harness touched, which is the realistic edit | **4.3 s** |
| ...with nothing to do | 0.03 s |

The harness row is the one that matters and it is the argument for keeping the
observers thin: touching the shared package rebuilds every observer that imports
it, so the marginal cost of phase N is roughly 1.8 s of TinyGo plus 0.4 s of
packaging per suite, on top of whatever the harness itself costs to relink.

Package, and this is the number a phase should watch:

| | Lua | Go |
|---|--:|--:|
| `bbb-m1-test` source | 252 lines / 9,329 B | 286 lines + a shared 604-line harness |
| `bbb-m1-test` staged | 2 files, ~9.7 KB | 5 files, **516 KB** |
| `bbb-sedge-test` source | 392 + 20 lines / 15,243 B | 417 + 56 lines |
| `bbb-sedge-test` staged | 3 files, ~15.7 KB | 8 files, **884 KB** |

**Half a megabyte of generated Lua per observer is the price**, and it is paid by
a headless run that loads it once. It is not paid by anything a player installs.
The suites' wall-clock did not move measurably (`m1` + `sedge` together: 6.7 s
before, 7.1 s after), because what grew is a parse, not a tick.

---

## The phases

Each phase is: goldens, port, the six gates, `git rm` the Lua, and a section in
this file recording what it measured and what it deviated on.

| phase | suites | what is new about it |
|---|---|---|
| **1 (done)** | `m1`, `sedge` | the harness, the build recipe, the staging seam. `sedge` brings the first observer DATA STAGE |
| **2** | `mar`, `mig21`'s observer, `qual`, `flip` | the first suites whose observers carry real per-tick STATE and arithmetic. `mar` reads the mod's `[BBB] heap` probe and drives 680 world operations from a schedule; `flip` drives `remote.call`, which is a member no observer has bound yet |
| **3** | `mix`, `plat`, `mig` | `plat` needs Space Age surfaces and `helpers.create_profiler`; `mix` needs infinity-chest filter rotation over 48 item names; `mig` is the only suite whose two phases run under different mod sets, and its observer is the one that reports a census |
| **4** | `m2`, `m3`, `edge` | the big ones. `m3` carries an LCG and 600 ticks of randomised churn; `edge` counts every item on two surfaces inside one tick. These are where the harness will earn or fail to earn its keep |
| **5** | `test/interactive/bbb-interactive-setup` | not a suite -- the world a HUMAN walks, and where the mod portal's demo scenes live. `iact` gates it, so the port has a headless check already |
| **6** | `test/mods/belt-balancer-2`, `test/mods/bbb-mig-foreign` | data-stage-only stand-ins, and **blocked on [`FKLUA-GAPS.md`](../FKLUA-GAPS.md) item 26**: `fklua mod` cannot package a mod with no control module. The `test/fixtures/fastbelt` workaround (an inert empty `main`) works and costs ~113 KB of generated Lua that is required and never called; whether to spend that on two stand-ins or wait for the gap is a phase-6 decision |
| **7** | `bench/mods/*` | LAST, and alone. Every published performance figure in this repository was measured with those setup mods, so the port has to carry a comparability gate the suites do not need: the same matrix cell, INTERLEAVED in one session, old and new setup mods against the same `dist/`, with the no-mod control in the same session. Session drift on this machine is 25-35% |
| **8** | `mig21`'s observer again, in RUST | the parity exercise. One observer written twice against one ABI is the strongest statement this repository can make about FkLua's second backend, and `mig21` is the right one: no `--create` phase, so the whole thing is one load and one set of assertions |

### What later phases inherit that the pilot did not need

Four pieces of FkLua surface landed for this port and the pilot consumed one of
them. They belong to the phases that use them, and **no new gap is filed for any
of them**:

- **`fk.LastError()`** -- used, by `harness.Fatal`, which names the engine's own
  refusal where the Lua's `error()` could only name the tile.
- **`fkapi.Log(Value)`**, the bound global `log()` -- **unused in the pilot and
  it is what unblocks phases 3, 4 and 7**. A `LuaProfiler`'s duration cannot be
  read by any member it has; the engine renders it only when the profiler is an
  ELEMENT of a `LocalisedString`, and `fk.Log` takes a plain string. `m2`, `plat`
  and `bench` all publish `helpers.create_profiler` timings by regexing exactly
  that line out of `factorio-current.log`, so those three suites were
  unportable until the global bound. The shape is
  `fkapi.Log(fkapi.OfArray(fkapi.OfString(""), fkapi.OfString("tag "),
  fkapi.OfObject(prof)))`. **Untried here** -- the first phase that needs it
  should verify it against a golden line before building anything on it.
- **`fkapi.TableSize(Value)`** -- unused so far; `#` and `table_size` appear in
  the unported observers' census and chest-scan code.
- **The package-time jump-span check.** An over-long emitted function is refused
  at package time now with a `//go:noinline` remedy named. A data stage is the
  shape most likely to reach it (item 25's own postscript); an observer's
  `on_init` is the second, and phase 4's `edge` -- one function that builds
  fifteen clusters over 198 parts -- is the first place in the estate likely to
  meet it. The remedy is one pragma per rig-building section.

### A note for phase 3, and for `mig21` in phase 2

`test/run.sh`'s `stage_fixture` OVERWRITES a staged test mod's `control.lua` with
a comment, to neutralize it while keeping its data stage. That works unchanged on
a ported observer -- the file is generated, and replacing it leaves a mod that
loads its data stage and does nothing -- so no runner change is owed. The other
generated files sit there unread.
