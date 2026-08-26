# The test estate, in Go

**There is to be no hand-written Lua in this repository, anywhere.** The shipped
mod got there in two rounds -- `fklua mod` has always generated the control
stage, and the 2026-08-25 round replaced ten hand-written data-stage files with a
second compiled guest ([`FKLUA-GAPS.md`](../FKLUA-GAPS.md) item 25). What is left
is the TEST ESTATE, and after phase 7 it is **478 committed lines** of it, all of
them in `bbb-flip-test`, which waits for a Factorio 2.0 binary. Lua that nothing in `make check` can reach, that no
toolchain type-checks, and that only a Factorio run can execute at all. It was
8,524 over twenty-four files when the pilot started, and none of what is left is a
SUITE.

This file is the programme for removing it, and the record of what each phase
measured. The pilot is `m1` and `sedge`, phase 2 is `mar`, `mig21` and `qual`,
phase 3 is `mix`, `plat` and `mig`, phase 4 is `m2`, `m3` and `edge` -- the three
biggest -- phase 5 is the interactive staging mod, phase 6 the two
data-stage-only stand-ins and phase 7 the `bench/` harness's setup mod, all done
2026-08-25. **Every suite in the estate is a
Go observer now, and so is every mod either of them stages.**

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
guest/go/obs/mar/        phase 2, and its data stage in obs/mardata/
guest/go/obs/mig21/      phase 2 -- no data stage, and no fk_on_init either
guest/go/obs/qual/       phase 2, and its data stage in obs/qualdata/
guest/go/obs/bb2data/    phase 6 -- a DATA STAGE WITH NO OBSERVER above it, and
guest/go/obs/foreigndata/  the same. Not observers: the mig suite's third-party
                         half, packaged with no control module at all
test/obs-data/           the estate's non-Lua asset trees, one directory per mod,
                         carried by `fklua mod --include`. Two locale files
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
  manifest's list rather than adding to it. Every observer carries exactly the
  dependency list its `info.json` file used to -- checked field for field against
  the deleted original, and for `mig21` that list is a CORRECTNESS surface rather
  than an identity: see phase 2's red proof.
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

## Phase 2: `mar`, `mig21`, `qual` -- done 2026-08-25

1,194 lines of Lua deleted across three mods; the estate is **7,357 lines over
twenty files**, from 8,524 over twenty-four. `flip` was chartered here and is
deferred to the 2.0 session (above).

### The golden diff, which is empty

Same two masks as the pilot's and no others. **Every one of the 9,835 + 480 + 204
tagged lines is byte-identical, in order**, across five logs:

| log | tagged lines | verdict |
|---|--:|---|
| `mar` create + run | 65 + **9,835** | identical |
| `mig21` m2 | 237 | identical |
| `mig21` edge | 243 | identical |
| `qual` create + run | 172 + 32 | identical |

**`mar`'s 9,835 is the one worth reading twice**, because the tag set is
`[BBB-MAR]` AND `[BBB]` -- so it contains every `[BBB] heap ... sys=... alloc=...`
probe the MOD wrote, 681 of them, each one the number this suite exists to
measure. Identical in order means the mod's heap trajectory is the same tick for
tick under a Go observer as under the Lua one.

What is left over is the pilot's own five categories and nothing else: the run
timestamp and the free-disk figure; `Checksum for script
__<observer>__/control.lua`, which IS the port; `Checksum of <observer>` for the
two that have a data stage (`mig21` has none, and its mod checksum is unmoved at
0, exactly as `m1`'s was); `Loading script.dat` where the observer has a guest
heap in the save now (`mar` 239,912 -> 753,972 B, `qual` 441,505 -> 938,963 B --
and `mig21` is unmoved, because `--benchmark` never saves and its script.dat is
the committed fixture's); and the benchmark's own `checksum:` and ms figures,
Factorio's state checksum covering every mod's `storage`.

### The gate this phase had that the pilot did not

**The `mar` slope table, in the `-gc=leaking` arm, byte-identical.** It is the
sharpest measurement in the repository and the one a scheduling drift of a single
tick would move:

| leg | B/primitive | | leg | B/primitive |
|---|--:|---|---|--:|
| A | 1,280 | | E | 560 |
| B | 352 | | G | 3,736 |
| C | 1,209 | | F | 2,080 |
| D | 32 | | linear memory | **3.92 MiB** |

...and `diff` over the WHOLE of `assert-marathon.py`'s output -- every raw
`B/iter`, every net figure, all seven linearity ratios, the 1,136 B calibration
at 0.0% spread and all ten world tuples -- is **empty**.

### The red proof: the dependency list is load-bearing end to end

The pilot's two red proofs perturbed a log line's FORMAT. This one perturbs the
PACKAGING, because `mig21` is the first observer whose `info.json` is a
correctness surface rather than an identity.

Add `--dependency "better-belt-balancer"` to the `mig21` recipe -- one flag,
nothing else touched -- and Factorio's load order puts the observer AFTER the mod
under test, so its `on_configuration_changed` runs after the migration has
already torn the remnants down. **Five assertions fire and `run.sh` exits 1**,
and the one with the name on it is the anti-vacuity the observer's own header
promises:

> nothing was seeded into the networks before the migration ran. Either the
> observer's `on_configuration_changed` now runs AFTER this mod's -- in which case
> there was nothing left to seed -- or the fixture arrived with no networks at
> all. Every item number below would be a vacuous zero

with `seeded 0 items`, `0 interfaces on 77 part tiles` and `0 hidden entities` in
the report beneath it. That is the whole `--dependency` chain proved: a Makefile
flag, into a generated `info.json`, into Factorio's mod ordering, into which
handler sees the world first.

**And the three generated `info.json` files are field-for-field identical to the
hand-written ones they replace** -- name, version, title, author,
`factorio_version`, description and dependencies -- which is the same statement
from the other end. `--dependency ""` was considered and NOT used: it means an
EMPTY list, and these mods' lists are not empty (`mig21`'s is `base >= 2.1.0`
alone). What the suite needs is the absence of one entry, not the absence of all
of them.

### What it cost

| | Lua | Go |
|---|--:|--:|
| `bbb-marathon-test` source | 496 + 16 lines | 609 + 53 lines |
| `bbb-marathon-test` staged | 3 files, ~19 KB | 8 files, **876 KB** |
| `bbb-mig21-observer` source | 277 lines | 541 lines |
| `bbb-mig21-observer` staged | 2 files, ~11 KB | 5 files, **740 KB** |
| `bbb-qual-test` source | 357 + 20 lines | 423 + 41 lines |
| `bbb-qual-test` staged | 3 files, ~15 KB | 8 files, **916 KB** |

`make observers` for all five packages is 13.3 s warm. The three suites together
run in 39.8 s against the goldens' 32.2 s; what grew is five hundred kilobytes of
`fk_module.lua` being PARSED three times, not anything a tick does.

`fklua api check --from 2.1.16 --to 2.0.77`, which every phase owes for the
observers it ports:

| guest | surface | verdict |
|---|---|---|
| `obs-mar.wasm` | 21 members, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |
| `obs-mig21.wasm` | 20 members, 1 event, 5 concepts | **clean**, 0 findings, exit 0 |
| `obs-qual.wasm` | 19 members, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |

So all three stay STAMPED for the running engine rather than gated against it,
which is what the pilot's table established and what this one re-establishes for
three guests whose member counts are nearly double the pilot's.

### What the harness gained, and it is nine things

Everything below was in two or three of the three observers, which is the bar the
pilot set for putting something in the shared package:

| | |
|---|---|
| `PlaceSoft` | the Lua's `put_soft`: a placement whose failure is a fact about the schedule (`mar`, a hundred iterations over the same tiles) or IS the measurement (`qual`'s two probes report `created=true`) |
| `Piece.Quality` | the whole of the `qual` suite in one field |
| `Piece.FastReplace` | `qual`'s replace probe |
| `FindAt` | the Lua's `at()`, a POINT query -- see the deviation below |
| `KillAt` | `FindAt` plus a raising destroy, which is every removal `mar` makes |
| `Destroy` | the same for an object already in hand |
| `EntitiesIn` | a box sweep, for `mar`'s conservation count and `qual`'s tile probes |
| `InventoryTotal` | a chest total for an entity already in hand (`ChestCount` is now this plus a lookup) |
| `TransportLineItems` | every item on one entity's lines, with the `pcall` the Lua wrapped `get_max_transport_line_index` in becoming an `err != nil` arm |
| `ForceByName` | `game.forces[name]` as a LuaCustomTable point query, which is what `can_fast_replace` needs and the whole-table read would allocate around |
| `Line.B` | `tostring` on a boolean, which three observers' log lines carry |

### Three deviations, all recorded rather than hidden

- **`FindAt` is a POINT query where `FindOnTile` is a box**, and the difference is
  load-bearing rather than stylistic. `find_entities_filtered` with an `area`
  returns everything whose bounding box TOUCHES it, and a transport belt's
  selection box is the whole of its tile -- so a box query on tile x can also
  reach the belt on x+1 along the shared edge, and `[0]` of that is whichever the
  engine listed first. `mar` removes one named belt out of a run of four, a
  hundred times over, and the wrong one would be a different world with a
  plausible slope. The Lua used `position` and so does this. The pilot's own box
  deviation stands where it is: its rigs have nothing beside the tiles they ask
  about.
- **`#line` is `LuaTransportLine.Length()`**, which is the bound form of the same
  Lua length operator and not a substitution. `get_item_count()` was the other
  candidate and would differ on a stacked belt; nothing in these fixtures is
  stacked, and the faithful one costs nothing.
- **The loader name is written down twice per suite**, in `obs/<suite>` and in
  `obs/<suite>data`. It is forced: a data guest may not import fkapi and a control
  guest may not import fkdata, so no package can be shared between them. A
  constants-only package with no imports at all WOULD work and is deliberately not
  built yet -- phase 3 adds two more loaders (`mix` and `plat`), and a shared
  package that three of five data stages use is worse than none. Build it when
  phase 3 makes it five of five, and bring `sedge` into it then.

### What phase 3 inherits from this one

- **`fk_on_configuration_changed` works and is a plain no-argument export.**
  `mig21` is the first observer to use it and `mig` in phase 3 is the second --
  that suite's two phases run under different mod sets, which is precisely what
  the hook reports.
- **`fkapi.Log(Value)` is still untried.** Nothing here needed a `LuaProfiler`,
  so the note above stands exactly as the pilot left it: `plat` is the first
  consumer, and it should verify the line against its golden before anything is
  built on it.
- **`fkapi.TableSize` is still unused.** `mig21` needed a `#`-over-a-table twice
  (`by_tile` and `per_surface`) and both became ordinary Go slices, which is what
  every census in the estate probably becomes.
- **The package-time jump-span check has not fired.** `mar`'s `fk_on_init` builds
  four rigs including a 4x4 and is comfortably inside it; `edge`'s fifteen
  clusters over 198 parts in phase 4 is still the first place likely to meet it.

---

## Phase 3: `mix`, `plat`, `mig` -- done 2026-08-25

2,121 lines of Lua deleted across three mods; the estate is **5,263 lines over
fourteen files**, from 8,524 over twenty-four when the pilot started. All three
suites are byte-identical to their goldens and the whole estate is green in both
`-gc` arms.

This is the phase that consumed the LAST piece of FkLua surface the port was
waiting on -- `fkapi.Log(Value)`, the bound global `log()` -- and the phase that
made the shared-constants package five-of-five, which phase 2 deliberately
deferred.

### THE PROFILER, verified before anything was built on it

The pilot and phase 2 both left the same note: `fkapi.Log(Value)` is untried,
`plat` is the first consumer, and it should be checked against a golden line
before anything rests on it. So the first thing this phase did after taking its
goldens was build a THROWAWAY observer with nothing in it but the idiom, package
it, and run one `--create`:

```
Script =[C]:4294967295: [BBB-SPIKE] timing audit only, nothing pending Duration: 0.507333ms
```

against the golden's

```
Script @__bbb-plat-test__/control.lua:690: [BBB-STK] timing audit only, nothing pending Duration: 14.646167ms
```

**IT WORKS, and the shape is exact**: same text, same `Duration: N ms` rendering,
same unit, same position in the log. `fkapi.Log(fkapi.OfArray(fkapi.OfString(""),
fkapi.OfString(tag), fkapi.OfObject(prof)))` is the whole of it, and the empty
leading element is LocalisedString's "concatenate the rest" form. The ledger item
is [`FKLUA-GAPS.md`](../FKLUA-GAPS.md) 27.

**One thing differs and it is not the duration: the `Script <origin>:` prefix.**
`fk.Log` is a wasm IMPORT the generated `control.lua` answers with a Lua `log()`
call, so Factorio attributes it to that file; `fkapi.Log` is a HOST CALL that
`fk_abi.lua` makes through `pcall`, so Factorio attributes it to the C boundary
and writes `=[C]:4294967295:` (which is `-1` as unsigned -- "no line" -- and is
therefore a constant rather than a wall clock). Both are the engine's own note of
where a `log()` was called from; nothing anywhere reads it, and phases 1 and 2
already masked the part of it that moved for exactly this reason.

### The golden diff, which is empty

Phase 1's two masks, ONE WIDENED and ONE ADDED, and every one justified:

| masked | why |
|---|---|
| the elapsed-seconds column Factorio stamps on every line | wall clock, phase 1's |
| the whole `Script <origin>:` attribution, not just its line number | phases 1 and 2 masked `control.lua:N:` -> `control.lua:`; the profiler's line comes from the C boundary instead, so the mask has to cover the origin rather than only the digit in it. It is the same fact being hidden and the same reason: nothing reads it |
| the digits of `Duration: N ms` | the profiler's own measured duration, which is a wall clock by definition. THE TEXT AROUND IT IS NOT MASKED, which is what makes the tag, the label and the rendering itself part of the diff |

**Every one of the 1,290 tagged lines across twenty logs is byte-identical, in
order.** `mig` alone is sixteen of those logs, because its seven legs run two
phases each under DIFFERENT MOD SETS and its two name probes run one:

| log | tagged lines | | log | tagged lines |
|---|--:|---|---|--:|
| `mix` create + run | 75 + **347** | | `mig4` create + run | 104 + 35 |
| `plat` create + run | 92 + 74 | | `mig5` create + run | 104 + 32 |
| `mig1` create + run | 12 + 104 | | `mig6` create + run | 12 + 36 |
| `mig2` create + run | 21 + 92 | | `mig7` create + run | 20 + 92 |
| `mig3` create + run | 21 + 92 | | `migp1`, `migp2` create | 21 + 21 |

What is left over is the pilot's own categories and nothing else: the run
timestamp and the free-disk figure; `Checksum for script
__<observer>__/control.lua`, which IS the port; `Checksum of <observer>`, because
all three have a data stage now; `Loading script.dat`, where the observer's guest
heap is in the save where a Lua mod's `storage` tables were (`mix` 250,755 ->
799,188 B, `plat` 265,197 -> 663,553, `mig1` 1,761 -> 539,940); and the
benchmark's own `checksum:` and ms figures, Factorio's state checksum covering
every mod's `storage`.

**`Checksum of better-belt-balancer` is 3503679581 and `Checksum for script
__better-belt-balancer__/control.lua` is 2133551073 in the golden AND in the
ported run**, in every log that has them. The mod under test did not move.

### The red proof: a LocalisedString whose first element is not empty

A NEW FAMILY, and it found a hole in the runner on the way.

`log{"", "tag ", p}` renders the profiler. `log{"tag ", p}` does not: the first
element of a LocalisedString is a KEY, so the engine looks up `"[BBB-STK] timing
full recompile (audit-forced) "` in the locale, does not find it, and writes
`Unknown key: "..."` instead of the sentence. It is exactly the mistake somebody
transcribing this idiom makes, and dropping that one element is a one-token edit.

**Injected, the first run came back GREEN.** All five timing lines rendered as
`Unknown key`, `assert-plat.py` passed every assertion it has, and `run.sh`
exited 0 -- because that script does not read the timing lines (they are
informational; only `assert-m2.py` parses a `Duration:`), and `Unknown key` was
grepped in the CREATE log and not in the benchmark's. The golden diff caught it,
five lines, but nothing in the estate's own gate did.

So `test/run.sh` greps for `Unknown key` in the run phase now, in both places
that read a `run.log` -- the benchmark and `mig21`'s fixture load -- matching
what the create phase has done since the estate had a locale file. Verified safe
first: the term appears in no log of any suite in either arm, only in a comment
inside this mod's own locale file. With the gate closed, the same injection
fails by name:

```
script error during benchmark; see .../plat/run.log
65: Script =[C]:4294967295: Unknown key: "[BBB-STK] timing audit only, nothing pending "
...five lines...
RUNSH EXIT=1
```

That is the phase's red proof and also its one runner change: a broken
LocalisedString in ANY suite's run phase is now a failed run rather than a line
nobody reads.

### `fklua api check --from 2.1.16 --to 2.0.77`, and the FIRST IMPACTED OBSERVER

| guest | surface | verdict |
|---|---|---|
| `obs-mix.wasm` | 25 members, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |
| `obs-mig.wasm` | 42 members, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |
| `obs-plat.wasm` | 42 members, 1 event, 13 concepts | **impacted**, 1 finding, exit 1 |

The charter's rule says an impacted observer may not be stamped and needs its own
answer, so here is `plat`'s. The finding:

```
LuaSpacePlatform::apply_starter_pack   breaking   parameter "silent" removed
```

**It is real and this call site is unaffected, and that is checkable rather than
arguable.** `silent` is optional, this observer passes it ABSENT, and
`fk_abi.lua`'s `M.call` trims the argument list to the last argument actually
PRESENT before invoking -- read out of the packaged ABI rather than quoted:

```lua
local n = #m.sig.args
while n > 0 do
  local f = m.sig.args[n]
  if f.has == nil or io_.ld8(argp + f.has) ~= 0 then break end
  n = n - 1
end
```

With `silent` absent, `n` falls to 0 and the call reaches the engine as
`apply_starter_pack()`, which is the whole of 2.0.77's signature. What would
break is an observer that PASSED it, and none does. So `plat` stays stamped, and
the tripwire is the check itself: the day this observer sends `silent`, the
verdict is still `impacted` and the answer has to be a different one.

It is also worth recording that the check EARNED ITS PLACE here. Two phases of
clean verdicts made it look like a formality; one finding out of 766 breaking
changes, landing on the one observer that touches DLC surface, is what it is for.

### The shared constants package, and the retrofit

Phase 2 deferred this with a reason: "a shared package that three of five data
stages use is worse than none. Build it when phase 3 makes it five of five."
Phase 3 brings three more loaders, so it is SIX of six and the package is built.

**`guest/go/obs/protos` imports NOTHING** -- not fkapi, not fkdata, not the
standard library -- and that is the whole trick. A control guest may not import
fkdata and a data guest may not import fkapi, so no package holding either can be
shared between the two halves of one suite; a package with no imports at all can.
It holds `BaseLoader`, `ExpressSpeed`, one loader name per suite, and the
stacking loader's name and size.

**`guest/go/obs/obsdata` is the second half and it imports fkdata**, which every
data stage may. `ExpressLoader(name)` is the five `fkdata` calls all six data
stages made identically. The three phase-1 and phase-2 data stages were
retrofitted onto both packages in the same commit, which took them from 53, 56
and 41 lines to 26, 29 and 24; the three observers lost their duplicated `loader`
constants the same way.

### What it cost

| | Lua | Go |
|---|--:|--:|
| `bbb-mix-test` source | 609 + 21 lines / 28,743 B | 692 + 25 lines |
| `bbb-mix-test` staged | 3 files, ~28 KB | 8 files, **1.06 MB** |
| `bbb-plat-test` source | 724 + 24 lines / 34,102 B | 1,080 + 46 lines |
| `bbb-plat-test` staged | 3 files, ~34 KB | 8 files, **1.37 MB** |
| `bbb-mig-test` source | 693 + 23 lines / 30,106 B | 784 + 26 lines |
| `bbb-mig-test` staged | 3 files, ~30 KB | 8 files, **1.20 MB** |

**The packages are twice the pilot's half-megabyte**, and the reason is worth
knowing before phase 4 sizes anything: these three observers call 25 to 42
members where `m1` called 12, and the member table plus the ABI shapes each one
needs is most of what a packaged observer weighs. It is still paid by a headless
run that parses it once and never by anything a player installs.

`make observers` builds and packages all EIGHT from clean in **17.9 s** (phase 1
recorded 4.1 s for two, phase 2 13.3 s for five), which is the number to watch:
touching the shared harness relinks every observer that imports it, and phase 3
added two more shared packages below it.

The whole estate is **2m0.6s** in the collected arm and 2m11.2s leaking, against
2m2s and 2m19s after phase 2 -- three suites' worth of extra `fk_module.lua`
being parsed, and not anything a tick does.

### The `mar` slopes, unmoved

The gate phase 2 added, and phase 3 has to clear it for a different reason: this
phase touched `guest/go/obs/mardata` and `guest/go/obs/mar` in the retrofit, so
the marathon observer is not the same file it was.

| leg | B/primitive | | leg | B/primitive |
|---|--:|---|---|--:|
| A | 1,280 | | E | 560 |
| B | 352 | | G | 3,736 |
| C | 1,209 | | F | 2,080 |
| D | 32 | | linear memory | **3.92 MiB** |

...byte-identical to phase 2's, which is byte-identical to the record this
repository has carried since the single-edge port. A constants package that
changed a number would move one of these.

### Deviations, all recorded rather than hidden

- **A quality and a planet cross as HANDLES, not strings.**
  `InfinityInventoryFilter.quality` takes a QualityID, which the description
  spells `string or LuaQualityPrototype`, and
  `LuaForce.create_space_platform`'s `planet` takes a SpaceLocationID the same
  way; the Lua sent the string and the generated struct fields are `*Object` and
  `Object`, so the observers resolve the prototype through a LuaCustomTable point
  query and send that. It is the same value to the engine, it is the reading the
  SHIPPED guest already takes for the same union (`guest/go/legacy.go` passes a
  quality handle rather than copying a name into the guest heap), and it costs
  one point query. `harness.QualityProto` and `harness.SpaceLocationProto`.
- **`harness.Tally` is a slice and a linear scan where the Lua used a table.**
  Both are safe -- every emitter here sorts before it writes -- and a slice is
  the shape whose determinism needs no argument. Forty-eight kinds is a scan
  nobody can measure. The same choice is made for `plat`'s stack-size histogram,
  which is kept in ascending key order by an insertion sort rather than by
  `table.sort` over `pairs`.
- **`harness.Line.F1` rounds half-away-from-zero where C's `printf` rounds
  half-to-even**, and the difference is unreachable in the one place it is used:
  `mig`'s fidelity rig reports a health that was SET to an integer and a
  `max_health` that comes off a prototype as one, so every value is exact at one
  decimal. Anything that starts logging a real measurement should re-read that
  comment first.
- **A sushi source is re-found on its TILE every rotation** where the Lua kept
  the chest handle in `storage`. It is the estate's own no-entity-references
  habit and it costs one point query per source per four ticks; `Object.Retain`
  would work and is deliberately not reached for.
- **`checkItems` cannot abort.** The Lua called `error()`, which aborts `on_init`
  outright; a guest has no way to unwind, so a missing prototype writes
  `[BBB-OBS] error:` naming every one of them and returns, and `run.sh`'s
  `guest_gate` fails the run. That is the harness's standing answer and phases 1
  and 2 took it too; it is repeated here because `mix` and `plat` are the two
  suites where the anti-vacuity check is the whole point of the line.

### What phase 4 inherits

- **`fkapi.Log(Value)` is proved and `harness.Profiler` wraps it.** `m2` is the
  second consumer and its `[BBB-M2] timing ... Duration: N ms` line IS parsed --
  `assert-m2.py` carries `re.compile(r"\\[BBB-M2\\] timing (.+?) Duration:
  ([\\d.]+)ms")`, which is the regex `plat`'s equivalent does not have. So `m2`'s
  red proof can be the one the charter originally wanted for `plat`: perturb the
  label and watch the parse fail.
- **`fkapi.TableSize` is STILL unused.** Three phases in, every `#` and
  `table_size` in the estate has become an ordinary Go slice length. Phase 4
  should expect the same rather than looking for a use.
- **The package-time jump-span check has still not fired.** `plat`'s
  `buildStacking` is the largest `fk_on_init` in the estate so far -- five bands,
  two loaders, a force and a platform -- and it is comfortably inside. `edge`'s
  fifteen clusters over 198 parts remains the first place likely to meet it, and
  the remedy is one `//go:noinline` per rig-building section.
- **`harness` is 1,229 + 130 lines now** and carries prototype point queries, a
  per-name tally, per-line contents, ground stacks, force and surface lookups,
  infinity-chest filters, item counts, technology reads and the profiler. `m2`,
  `m3` and `edge` are the suites the harness was built for; if it earns its keep
  anywhere it is there.
- **One runner change to know about**: `Unknown key` now fails a run phase. A
  suite that legitimately logs one would have to say so.

---

---

## Phase 4: `m2`, `m3`, `edge` -- done 2026-08-25

3,350 lines of Lua deleted across three mods -- the three biggest in the estate --
and what is left of it is **1,827 lines over eight files**: the interactive
staging mod, the bench harness's setup mod, and the three data-stage-only
stand-ins phases 5 and 6 own. From 8,524 over twenty-four when the pilot started.

All three suites are byte-identical to their goldens, the whole estate is green
in both `-gc` arms, and the `mar` slopes are unmoved.

### The masks stopped being an argument and became a measurement

Phases 1 to 3 justified each mask in prose. This phase took the goldens TWICE --
two runs of the unmodified tree, back to back -- and diffed them against each
other first. Under exactly the three established masks the two golden runs are
**identical across all 4,220 tagged lines**, which says those three are the only
nondeterministic things in these logs and that any diff after the port is a port
defect rather than a candidate for a fourth mask. It costs one extra run per
phase and it is worth doing again.

| masked | why |
|---|---|
| the elapsed-seconds column | wall clock |
| the whole `Script <origin>:` attribution | phase 3's, widened for the profiler's C-boundary origin |
| the digits of `Duration: N ms` | the profiler's own measured duration. THE TEXT AROUND IT IS NOT MASKED |

**Every one of the 4,220 tagged lines is byte-identical, in order**, and so is
every number the three assertion scripts print:

| log | tagged lines | | log | tagged lines |
|---|--:|---|---|--:|
| `m2` create + run | 364 + 89 | | `edge` create + run | 438 + **2,714** |
| `m3` create + run | 158 + 457 | | | |

What is left over is the pilot's own categories and nothing else: the run
timestamp and the free-disk figure; `Checksum for script
__<observer>__/control.lua`, which IS the port; `Checksum of <observer>`, all
three having a data stage now; and `Loading script.dat`, where the observer's
guest heap is in the save where a Lua mod's `storage` tables were (`m2` 343,071
-> 2,073,601 B, `m3` 273,941 -> 1,308,951, `edge` 528,280 -> 2,480,592). The mod
under test is unmoved in every log.

`Checksum of bbb-m3-test` and `Checksum of bbb-edge-test` come out EQUAL, which
is not a coincidence to be explained away: both data stages are now the same four
calls into `obsdata.ExpressLoader`, so they are the same bytes.

### The LCG, and why the obvious transcription is wrong from the first value

`m3`'s churn carries its own generator in the save, because the schedule has to be
identical on every run for the assertions to mean anything:

```lua
storage.seed = (storage.seed * 1103515245 + 12345) % 2147483648
```

That reads as a textbook 31-bit LCG and IS NOT ONE. Factorio's Lua is
doubles-only; the product of a 31-bit seed and 1103515245 reaches 2.4e18; a double
carries 53 bits of mantissa, about 9e15. **The low nine bits of every product are
rounded away before the modulus sees them**, and every seed this generator
produces is a multiple of 512. A Go transcription in `uint64` computes the
arithmetic EXACTLY, which is a different generator.

Checked rather than reasoned about, against `../FkLua/bin/lua52f` -- the
doubles-only Lua 5.2.1 this repository keeps for exactly this kind of question --
over 300 churn steps and 600 `rnd` calls:

| candidate | against the real Lua |
|---|---|
| `uint64`, the textbook LCG | **differs at the very first value** and never re-converges |
| `float64`, mirroring Lua's own arithmetic | **identical, value for value, all 600** |

So `rnd` is written in floating point: the multiply and the add are separate
double operations, with an explicit `float64()` conversion between them to forbid
Go contracting them into a fused multiply-add that would be MORE accurate and
therefore wrong (wasm has no FMA instruction, so this is belt and braces -- but
the belt is free); and the modulus is `a - floor(a/b)*b`, which is how Lua 5.2's
C source defines `%` for numbers rather than `fmod`.

### The red proofs, and what they say about where the teeth are

**`m2`: the profiler label -- and the hole it found first.** The charter chartered
this one because `assert-m2.py` is the only script in the estate that PARSES a
`Duration:` line. Perturbing one label to `teardown+rebuild(complete)` came back
**GREEN**: the script read the timings only to PRINT them, so a drifted label
printed a different word and passed, and a suite that stopped emitting the lines
altogether would have printed nothing and passed too. That is phase 3's `Unknown
key` finding again one level out -- a check that skips is a check that passed,
applied to a whole block. `WANT_TIMINGS` closes it, and with it closed the same
injection fails by name:

```
  the profiler published 8 timing line(s) and these 2 were not among them:
  'sat4 teardown+rebuild(full)', 'sat8 teardown+rebuild(full)'. A label is what
  says WHICH window was measured, and a recompile figure is only meaningful
  against the `idle tick pair` control it is subtracted from
```

The labels are load-bearing rather than decorative: every recompile figure in
CLAUDE.md is one of these windows minus the `idle tick pair` control, and a run
that mislabelled one would publish a number attributed to the wrong measurement.

**`m3`: the LCG, and the golden diff earning its place as a first-class gate.**
Swapping the float64 generator for the textbook `uint64` one -- the transcription
anybody would write -- and running the suite:

| | ported (== golden) | textbook `uint64` |
|---|--:|--:|
| `[BBB] stats audit` | compiles=127 skipped=34 builds=89 teardowns=71 creates=921 | 139 / 34 / 101 / 83 / **1001** |
| final counters | 141 / 48 / 89 / 75 / 921 | 153 / 48 / 101 / 87 / **1001** |
| `stress recovered` | 15,856 | **15,848** |
| tagged run-log lines differing | 0 of 457 | **425 of 457** |
| `assert-m3.py` | passed | **passed** |

A completely different churn -- a different row and a different action at every
one of a hundred steps -- and the suite's own assertions cannot see it. They are
not supposed to: CLAUDE.md tracks those counters as measurements of the GUEST that
legitimately move when its batching changes (288 -> 139 compiles across the
`fk.defer` round), so pinning them in the assertion script would fail every honest
guest change. **For a PORT the golden diff is the right instrument, and this is
the clearest evidence in four phases that it belongs in the gate list rather than
beside it.**

### `fk.LastError()`, verified -- and the trap next door

`edge` is the estate's first real consumer of the last-error import, and it is the
sharp case: it asserts the engine's own refusal of
`script.raise_event(on_player_mined_entity)`, because that refusal is the whole
reason the miner's-pocket trigger is documented rather than tested. The verdict is
that it works exactly, golden against ported:

```
err=on_player_mined_entity (ID 76) (76) can't be raised through script.
```

byte-identical, including the whitespace collapse the Lua's `tostring(err):gsub`
applied. [`FKLUA-GAPS.md`](../FKLUA-GAPS.md) item 28 is the ledger entry.

**What went wrong on the way is not the import and is worth more than it.** The
first run produced

```
err=on_player_pipette (ID 114) (114) can't be raised through script.
```

`fkapi.EventOnPlayerMinedEntity` is 114, and that is FkLua's own dense index over
the description's event set -- what `fk.subscribe` takes, which the generated
control stage maps to `defines.events[name]` at load. `script.raise_event` takes
FACTORIO'S number, 76 here, which is not knowable at build time. Handing it 114 did
not fail; it proved a different event unraiseable. **`assert-edge.py` passed it**,
because that assertion checks only `ok=false` -- as it should, since what it is
guarding is "the day this stops being refused, write the real test". The golden
diff caught it, one line out of 3,152.

FkLua excludes `defines.events` from its generated define accessors deliberately
(*"offering a guest both spellings of `on_tick` would be a trap dressed as a
convenience"*), and `script.get_event_id(name)` is the door it leaves open: the
NAME travels and the engine resolves the number. No gap; a hazard, and the next
guest that calls `raise_event` should know it.

### The jump-span check DOES NOT FIRE, and that closes a three-phase note

Every phase since the pilot has recorded the package-time jump-span check as "not
fired yet, and `edge`'s fifteen clusters over 198 parts is the first place likely
to meet it". **It does not meet it.** `edge`'s `fk_on_init` packages with no
`//go:noinline` at all and its five rig-building sections free to be inlined into
one function. The threshold is 655,355 bytes of jump span, and FkLua's own
documentation puts the widest span across all of its guests at 248,861; this
observer is not close. The sections stay separate functions for readability and
carry no pragma, because a pragma with a reason that is not true is worse than no
pragma. **The note can be retired: nothing in this estate is going to reach it.**

### The one performance finding, and the hypothesis it killed

`edge`'s benchmark phase went 18 s to 42 s and its worst tick to 725 ms, with the
observer's own guest heap crossing a rung:

```
fklua: this guest's linear memory is now 16 MiB...
```

Benign by the charter's own rule -- an observer runs for seconds in a world that
is thrown away -- and still forty seconds of somebody's afternoon on the estate's
longest suite. The first hypothesis was the `[]Object` a sweep allocates per call,
and `FindEntitiesFilteredInto` exists for exactly that. **It was worth nothing**:
buffered, the rung was still crossed and the run was 41.9 s against 42.2.

The whole term was `EntityType`. It returns a Go string, so `getStr` COPIES the
host's bytes into the guest heap -- it has to, the arena under them is released
when the call returns -- and `edge` asks it of every entity on two whole surfaces
after each of a hundred churn cycles and at every one of its sixty-odd samples:
roughly nine hundred thousand short-lived strings that `-gc=leaking` never gives
back. `type_is` compares on the host and returns a bool. Measured alone, with the
buffers reverted:

| | rungs crossed | worst tick |
|---|--:|--:|
| `EntityType` | 1 (16 MiB) | 725 ms |
| `EntityTypeIs` | **0** | 350 ms |

So `harness.EntityTypeIs` ships and `harness.EntitiesInto` was DELETED rather than
kept behind a comment claiming a win it did not produce. It is the same reading
the shipped guest already takes for the same reason (`carry.go` uses `name_is`
over `name`, and its header says why).

**What did not move is the wall clock**, and that is the honest residual: 40 s
against the Lua's 18 s. It is the boundary, not the heap -- a sweep asks three or
four host calls of every entity it visits, at the ~12.6 us this repository has
measured for years, where Lua's own call into C++ is nearly free. Nothing about
the port can remove it and no number this suite asserts is affected by it.

### The other things this phase found

- **A profiler that spans a tick boundary needs `Object.Retain`.** `m2` times a
  recompile ACROSS the tick boundary by construction -- the mod batches, so the
  belt is destroyed in one tick and the network rebuilt when the deferred queue
  drains in the next -- and a handle that is not retained is valid only inside its
  own dispatch. The failure is loud rather than silent: five `[BBB-OBS] error:
  stopping a profiler` lines, exactly one per tick-crossing window, and none for
  the three that open and close inside one tick. The Lua kept its profiler in an
  upvalue and never had to think about it. `harness.Profiler.Retain`/`Release`,
  and it is the first use of the retain space in the estate.
- **`m2` needed a raw-position placement** for `spio`'s two-tile splitter, which
  straddles the boundary between its rows. `harness.Piece.Pos`.
- **`m3` needed `InnerName`** for a ghost, which is what a blueprint produces in a
  real game and is deliberately NOT a part.

### What it cost

| | Lua | Go |
|---|--:|--:|
| `bbb-m2-test` source | 879 + 48 lines / 39 KB | 1,086 + 47 lines |
| `bbb-m2-test` staged | 3 files, ~41 KB | 8 files, **1.0 MB** |
| `bbb-m3-test` source | 839 + 19 lines / 35 KB | 1,105 + 24 lines |
| `bbb-m3-test` staged | 3 files, ~36 KB | 8 files, **1.4 MB** |
| `bbb-edge-test` source | 1,632 + 18 lines / 78 KB | 1,711 + 24 lines |
| `bbb-edge-test` staged | 3 files, ~79 KB | 8 files, **1.6 MB** |

`harness` is 1,342 + 130 lines. `make observers` builds and packages all ELEVEN in
**56 s** from clean (phase 3 recorded 17.9 s for eight), and the three new
observers are the three largest in it.

The whole estate is **2m33s** in the collected arm and **2m45s** leaking, against
2m0.6s and 2m11.2s after phase 3 -- almost all of it `edge`'s boundary cost above.

`fklua api check --from 2.1.16 --to 2.0.77`:

| guest | surface | verdict |
|---|---|---|
| `obs-m2.wasm` | 25 members, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |
| `obs-m3.wasm` | **48 members**, 1 event, 12 concepts | **clean**, 0 findings, exit 0 |
| `obs-edge.wasm` | 39 members, 1 event, 11 concepts | **clean**, 0 findings, exit 0 |

`m3` is the widest surface any observer has -- blueprints, clones, ghosts, robots,
surface deletion and rendering -- and it comes back clean, so all three stay
STAMPED for the running engine rather than gated against it.

### The `mar` slopes, unmoved

The gate phase 2 added, and this phase touched `harness` in three places:

| leg | B/primitive | | leg | B/primitive |
|---|--:|---|---|--:|
| A | 1,280 | | E | 560 |
| B | 352 | | G | 3,736 |
| C | 1,209 | | F | 2,080 |
| D | 32 | | linear memory | **3.92 MiB** |

Byte-identical to phase 3's, which is byte-identical to the record this repository
has carried since the single-edge port.

### What phase 5 inherits

- **The estate's remaining Lua is 1,827 lines over eight files**, and none of it
  is a suite: `test/interactive/bbb-interactive-setup` (phase 5),
  `bench/mods/*` (phase 7), and the two data-stage-only stand-ins plus
  `bbb-flip-test` that phases 6 and the 2.0 session own.
- **`test/interactive` is gated by `iact`**, which is a `--create` and an assertion
  script, so phase 5 has a headless check already -- but its README checklist
  quotes TILE COORDINATES and file paths, and the port moves the file. Re-derive
  them from the Go rather than copying them across.
- **Take the goldens twice.** The A/B of two unmodified runs is what turns the
  mask list from an argument into a measurement, and it is cheap.
- **`fkapi.TableSize` is STILL unused**, four phases in. Every `#` and
  `table_size` in the estate has become an ordinary Go slice length.
- **The jump-span note is retired** (above). Nothing here will reach it.

---

## Phase 5: the interactive staging mod -- done 2026-08-25

462 lines of Lua deleted from one mod, and it is the first thing this programme
has ported that is NOT A SUITE. The estate is **1,365 lines over six files**,
from 8,524 over twenty-four when the pilot started: `bench/mods/*` (phase 7),
the two data-stage-only stand-ins (phase 6) and `bbb-flip-test` (the 2.0
session). Nothing else is left.

`guest/go/obs/iact` and `guest/go/obs/iactdata` are the observer and its data
stage. What makes it different from the eleven above it:

- **Its consumer is a HUMAN, not an assertion script.** It stages the five
  player-gesture rigs and the five mod-portal demo scenes beside spawn on a
  fresh world, hands the player the pieces and prints where to walk;
  `test/interactive/README.md` is the checklist that world exists for. The
  `iact` suite is a headless `--create` over the same package, and what it
  asserts is that the staging is LEGAL -- every piece landed, nothing refused,
  the exact compiled-shape multiset, both audit tuples.
- **`make interactive-install` installs the very package the suite stages.**
  That target used to copy a directory of Lua and now builds and copies
  `dist/obs/bbb-interactive-setup_0.2.0`, under the bare name Factorio also
  accepts. Its prerequisite is the one package rather than `observers`, so
  installing the rigs relinks nothing else and touches nothing the shipped mod
  owns.
- **It is the only package here whose version is not 0.1.0.** 0.2.0 is what its
  hand-written `info.json` carried, and `copy_testmod` globs the version for
  exactly this reason.
- **It has a player event handler, and nothing headless can reach it.**
  `on_player_created` teleports the arriving player, hands over 50 belts and 10
  parts, charts the map, tags every rig and prints the greeting. A `--create`
  has no players, so the `iact` suite sees none of it -- the same wall the
  miner's pocket's trigger is behind, and the same wall the Lua was behind.

### The masks, proved by self-diff before a line was ported

Phase 4's method, and it cost one extra run. Two golden runs of the unmodified
tree, back to back, normalised on the three established masks and diffed against
each other: **identical across all 498 tagged lines**, with the only differences
anywhere in the file being the run's start timestamp and the free-disk figure.
So those three masks are the only nondeterminism in this log, and any diff after
the port is a port defect rather than a candidate for a fourth mask.

### The golden diff, which is empty -- and the smallest one yet

**All 498 `[BBB]` and `[BBB-INTERACTIVE]` lines are byte-identical, in order**,
and so is every number `test/assert-interactive.py` prints: the twelve-shape
multiset, both audit tuples `(12, 228, 0, 0, 12, 0)` and `(12, 228, 12, 0, 0, 0)`,
and zero on all three of placements refused, refusals and spills.

The whole masked diff is **four lines**:

| difference | what it is |
|---|---|
| the run's start timestamp, and the free-disk figure | wall clock, and both moved between the two goldens too |
| `Checksum of bbb-interactive-setup` | its data stage is a different file. That IS the port |
| `Checksum for script __bbb-interactive-setup__/control.lua` | its control stage is a different file. That IS the port |

`Checksum of better-belt-balancer` is 3503679581 and `Checksum for script
__better-belt-balancer__/control.lua` is 507607469 in the golden AND in the
ported run. The mod under test did not move.

**There is no `Loading script.dat` row here, and its absence is why this diff is
smaller than every previous phase's**: `iact` is the estate's only create-only
suite, so nothing is ever loaded, no state checksum is taken and no benchmark
milliseconds are printed. The guest heap this observer writes into the save is
never read back by anything.

### `fkapi.RemoteCall`, verified before anything rested on it

The charter has carried a standing note since phase 2: `RemoteCall` is the
outbound half of mod-to-mod interop, no observer had used it, and whoever
reached it first should treat it the way phase 3 was told to treat
`fkapi.Log(Value)`. This mod is that consumer -- it asks `freeplay` to disable
the crash site and skip the intro -- and the calls it makes are SILENT, so the
golden diff could not have seen a failure. A throwaway spike, packaged and run
through one `--create` before the port was trusted:

```
[BBB-SPIKE] remote audit status=0 tag=2 number=0
[BBB-SPIKE] remote freeplay status=0
[BBB-SPIKE] remote nomethod status=5
```

**It works in both directions.** The first line calls the mod under test's own
`better-belt-balancer` interface -- `audit`, which every suite in the estate
already drives through a marker -- and gets `StatusOK` back with a `TagNumber`
value: arguments crossed out, a return value crossed back, and the mod's own
`[BBB] audit` line landed in the log where the call was made. The second says
`freeplay` is present even in a headless `--create`, so these calls really do
fire rather than being inert. The third is the one that licenses the deviation
below: a missing METHOD is `StatusCallFailed` and not a raise.

`LuaForce.Chart` and `LuaForce.AddChartTag` were spiked in the same run
(`chart err=false`, `chart_tag err=false got=true`), because the player block
uses both and nothing headless can enter it.

### Two red proofs, in two families

**The suite's own documented one, re-run against the PORTED observer.** Stage a
second belt against a part that already carries one -- one line added to band B
-- and three assertions fire, one per family, with `run.sh` exiting 1:

```
1 cluster(s) were refused for carrying more than one belt per part while STAGING
the staged world compiled [... '2->2/P2' missing ...] and the checklist's geometry is [...]
the second audit reads clusters=12 parts=228 nets=11 drift=0 unbuilt=0 refused=1
```

**And one for the seam this port actually moved.** `put` used to be a Lua
`create_entity` whose nil return was reported by `log(string.format(...))`; it is
`harness.PlaceSoft` plus this observer's own `Line` now, and if that line's format
drifted a genuinely missing rig would go unnoticed while every other number
stayed green. The injection is the port's own new hazard: **the observer and its
data stage are two separate wasm modules**, which is what `obs/protos` exists to
keep in step, so give the observer a loader name the data stage does not define.
97 placements fail, each named with its piece and its tile:

```
bbbi-loader-x did not land at (17,-24)
...and 87 more placements that did not land
```

Two things this proof settled on the way, both worth knowing before writing a
third: `create_entity` places THROUGH a collision -- a chest dropped on an
occupied belt tile lands happily -- and a duplicate part on an occupied tile is
created too, with the registry deduping it by tile and no count moving. Neither
is a way to make a placement fail. A prototype that does not exist is.

### `fklua api check --from 2.1.16 --to 2.0.77`

| guest | surface | verdict |
|---|---|---|
| `obs-iact.wasm` | 22 members, 1 event, 10 concepts | **clean**, 0 findings, exit 0 |
| `obs-iactdata.wasm` | 0 members, 0 events, 0 concepts | **clean**, 0 findings, exit 0 |

So it stays STAMPED for the running engine rather than gated against it, which
matters more here than for a suite: this is the one package a person installs by
hand into their own Factorio. The data module's empty surface is the expected
answer and is recorded rather than assumed -- a data guest imports fkdata and
never fkapi, so it has no runtime API surface to break.

**And the generated `info.json` is field-for-field identical to the hand-written
one it replaces** -- name, version, title, author, `factorio_version`,
description and both dependencies -- which is the same statement phase 2 made
about its three.

### What it cost

| | Lua | Go |
|---|--:|--:|
| `bbb-interactive-setup` source | 446 + 16 lines / 20,294 B | 609 + 26 lines |
| `bbb-interactive-setup` staged | 3 files, ~21 KB | 8 files, **880 KB** |

`make observers` builds and packages all TWELVE. The `iact` suite itself is
unmoved at about fifteen seconds, because a `--create` parses the package once
and never ticks.

### Deviations, all recorded rather than hidden

- **Two guards collapse into one status.** The Lua wrote
  `if remote.interfaces["freeplay"] then pcall(remote.call, ...) end` -- an
  existence test and then a protected call, because `remote.call` RAISES on a
  missing interface or method. `fkapi.RemoteCall` returns `StatusCallFailed`
  instead, deliberately, so the two guards are the return value and both
  spellings mean the same thing: do it if freeplay is there, carry on if it is
  not. Measured above rather than assumed.
- **The surface crosses as a HANDLE where the Lua passed a name.** `p.teleport`
  takes a SurfaceID, which the description spells `string or LuaSurface or uint`
  and whose generated parameter is an `Object`. It is the same deviation
  `harness.QualityProto` and `harness.SpaceLocationProto` record for their own
  unions, and the same reading the shipped guest already takes.
- **`put` is `PlaceSoft` and NOT `harness.Place`.** Every other observer in the
  estate is Fatal about a placement that did not land, because a rig that is not
  there makes every number after it a measurement of a different world. Here the
  gate is `test/assert-interactive.py`, which reads
  `[BBB-INTERACTIVE] could not place <name> at (x,y)` by name and reports which
  piece missed which tile -- so the placement is soft and the report is this
  observer's own. Red-proven above.
- **The capture conditions are CHECKED where the Lua ignored the return.**
  `show_clouds`, `always_day`, `freeze_daytime` and `daytime` are what make a
  portal GIF loop seamlessly, and a setter that silently failed would be
  discovered by a ruined recording. They are `harness.Fatal` now, which is what
  `Flat.Make` already does with the same `always_day` call.
- **The player block is transcribed and cannot be executed here.** Teleport,
  insert, chart, chart tags and the seven printed lines are behind the same wall
  as the miner's pocket's trigger, in the Go exactly as in the Lua. `go vet`
  type-checks it, the spike above proved the two force members it leans on, and
  the rest is a human's first thirty seconds in the staged world.
- **The prep of an EXISTING surface stays in the observer.** `harness.Flat`
  makes a scratch surface; this mod preps nauvis, which nothing else does, so the
  only piece that moved into the shared package is the pave (`harness.PaveBox`,
  which already existed for `plat`). One caller is below the bar for a helper.

### What phase 6 inherits

- **The estate's remaining Lua is 1,365 lines over six files**: `bench/mods/*`
  (722 + 17, phase 7), `test/mods/belt-balancer-2/data.lua` (94) and
  `test/mods/bbb-mig-foreign/data.lua` (54), which are phase 6, and
  `bbb-flip-test` (458 + 20), which waits for a 2.0 binary.
- **The two stand-ins have no golden of their own, and that is the phase's first
  problem.** Every phase so far took goldens of the suite it was porting; these
  two are DATA STAGES inside other suites' runs, so what phase 6 has to diff is
  `mig`'s seven legs and two name probes, and `foreign`'s leg within them. Take
  those goldens before touching anything.
- **`mig_standin`'s `info.json` REWRITE SURVIVES THE PORT UNCHANGED, and that
  is measured rather than assumed.** It stages ONE directory under all four of
  `legacyIncumbents`' names by rewriting `name` and `version` with `perl -pi`,
  guarded by two greps -- because a silently unrenamed copy would stage
  belt-balancer-2 under every name and pass every leg. Its substitutions are
  anchored `^(\s*)"name":\s*"[^"]*"`, and `fklua mod` emits a
  two-space-indented `"name": "..."` that matches: run over a generated
  `info.json` from phase 5's own package, both rewrites land and both greps
  pass. So that half of phase 6 is already answered; keep the greps.
- **A data-stage-only package still needs a control module** (item 26). The
  `test/fixtures/fastbelt` workaround is an inert empty `main` and costs about
  113 KB of generated Lua that is required and never called. An upstream fix for
  data-module-only packaging is being built in parallel; if it lands, prefer it,
  and note that these two stand-ins would then be the first packages in the
  repository with no control stage at all -- which `stage_fixture`'s
  `control.lua` overwrite, and `copy_testmod` itself, have never had to face.
  (**It landed and was preferred**; `copy_testmod` needed nothing, and
  `stage_fixture` is documented as unreachable from either stand-in rather than
  guarded. Phase 6 below.)
- **`copy_testmod` handles a non-0.1.0 version and a bare-name destination
  without any change to the runner**, which phase 5 is the proof of.
- **`fkapi.TableSize` is STILL unused**, five phases in.

---

## Phase 6: the two data-stage-only stand-ins -- done 2026-08-25

148 lines of Lua deleted across two mods, and **the first packages in this
repository with no control stage at all**. The estate is **1,217 lines over four
files**, from 8,524 over twenty-four when the pilot started: `bench/mods/*` (739,
phase 7) and `bbb-flip-test` (478, the 2.0 session). Nothing else is left.

`guest/go/obs/bb2data` is the `mig` suite's Belt Balancer stand-in and
`guest/go/obs/foreigndata` is the stranger who owns its prototype name. What
makes them different from the twelve observers above them:

- **They are not observers.** They observe nothing, drive nothing and log
  nothing. Each is a `fk_data` export and a `main() {}`, and the whole of what
  they are is prototypes -- which is exactly the shape
  [`FKLUA-GAPS.md`](../FKLUA-GAPS.md) item 26 was open for.
- **`fklua mod` takes NO control positional for them.** Each package is
  `info.json`, `fk_abi.lua`, `fk_data.lua`, `fk_data_module.lua`, `data.lua` and
  a locale file: six files, no `control.lua`, no `fk_module.lua`, no
  `fk_api_gen.lua`. `fk_abi.lua` stays because `fk_data.lua` requires it for the
  codec.
- **`--persist`, `--gc` and `--api` are gone from their flag set.** The first two
  are REFUSED rather than ignored, because each describes how a control guest is
  compiled and there is none; `--api` goes for a plainer reason, which is that a
  package with no member table has no description to pick. `OBS_DATA_ONLY` is
  what is left, and it is identity alone.
- **They are the estate's first `--include`.** A Factorio locale is `.cfg` and no
  data-stage API can emit one, so `test/obs-data/<mod>/` holds each mod's tree
  and `--include` merges its CONTENTS into the package -- the observers' form of
  the `data = "mod-data"` the shipped mod's manifest carries. See
  [`test/obs-data/README.md`](../test/obs-data/README.md).

### The masks, proved by self-diff before a line was ported

Phase 4's method, and the goldens here are somebody else's: these two have no
suite, so what had to be captured is `test/run.sh mig`'s whole log set -- sixteen
logs across seven legs and two name probes, the legs' two phases running under
DIFFERENT MOD SETS. Two runs of the unmodified tree, back to back into the same
tmp path so that no path mask is needed, normalised on the three established
masks and diffed against each other: **identical across all 819 tagged lines**,
with the only differences anywhere being the run's start timestamp, the free-disk
figure, and the benchmark's own `Performed N updates` / `avg, min, max`
millisecond lines. So those three masks are the only nondeterminism in this log
set, and any diff after the port is a port defect.

### The golden diff, which is empty

**All 819 `[BBB]` and `[BBB-MIG]` lines across the sixteen logs are
byte-identical, in order**, and so is every number `test/assert-mig.py` prints in
every leg: 31 parts from 3 surfaces into 9 clusters, 2 forces, 85.0 of 170.0
health, quality uncommon, 50 held with `place_result` flipped, the witness's 48
copper at four samples, `sok2` at 2.000x and `sok4` at 3.997x, the refusal
multiset `[2, 2, 2, 2, 2, 3, 4]`, and `clusters=9 parts=31 nets=2 drift=0
unbuilt=0 refused=7`.

What is left over is **four categories, and one of them is new to this phase**:

| difference | what it is |
|---|---|
| the run's start timestamp, and the free-disk figure | wall clock; both moved between the two goldens too |
| the benchmark's ms figures | the same |
| `Checksum of belt-balancer-2` (and of `belt-balancer`, `-3`, `-performance`, and of `bbb-mig-foreign`) | the mod's files are different files. That IS the port -- and it is the SAME value under all four incumbent names, which is `mig_standin` staging one package four ways |
| the save's own size, and the benchmark's `checksum:` | **new here, and measured rather than inferred** -- see below |

**A save records the checksum of every mod that made it, and the state checksum
covers it.** `4087984835` is the ported stand-in's mod checksum, and it is present
as a little-endian `uint32` at offset 76 of the save's own `level-init.dat`. That
is why the map grew by one to five bytes (zlib noise on a four-byte change) in the
five legs that have a stand-in installed at `--create`, and why the benchmark's
`checksum:` moved in the six legs that have one installed at `--benchmark`.
**`mig4` is the control on both halves**: it is the one leg with no stand-in in
either phase, and its map is byte-identical at 951,901 with its state checksum
unmoved at 20,235,449. `mig5` is the other half of the same control -- no stand-in
at create, so its map does not move; one at benchmark, so its checksum does.

This retires the attribution earlier phases gave that row. Phases 1 to 4 read the
state checksum as "Factorio's state checksum covers every mod's `storage`, and an
observer's `storage` is a guest heap now", which cannot be what moved it here:
neither stand-in has a control stage in EITHER version, so neither has ever had a
`storage` at all. There is no `Loading script.dat` row in this diff for the same
reason.

### The gate that came for free, and it is the sharpest one this phase had

**`test/check-datastage.py`'s `incumbent` arm hashes Factorio's own `--dump-data`
of the WHOLE prototype table, with the stand-in's prototypes in it -- and the hash
did not move.** `4ffb0e3b68f149d8` before the port and after it. That is a
byte-level equivalence proof of the entity, the item, the recipe and the
technology, taken by the engine rather than by anything in this repository, and it
is a stronger statement than any log line could make: a log diff says the
migration behaved the same, and this says the prototypes ARE the same.

It needed one change to reach: the arm staged `test/mods/belt-balancer-2` by path,
and the stand-in is a built package now. `staged_mod` resolves it out of
`dist/obs` with the same glob `run.sh`'s `copy_testmod` uses, and
`make datastage-check` names that ONE package as a prerequisite rather than all
fourteen observers.

### The other workaround that went with it, and it was not on the list

**`test/fixtures/fastbelt/inert` is deleted.** It was the workaround item 26
described -- an empty `main` compiled to about 113 KB of Lua that was `require`d
and never called -- and the ledger's own preamble says every workaround in it is
since deleted. The fixture packages data-only now (`--data-module` alone, no
positional, no `--persist`) and the speed arm is green: `all four hidden
prototypes at 0.5, from an underground belt`. It was Go rather than Lua, so it
does not move the estate's line count; it moves the ledger's honesty.

### The red proof: a prototype field is load-bearing to a conversion assertion

Drop ONE field from the ported stand-in's entity -- `max_health = 170` -- and
rebuild. Base's default for a `simple-entity-with-force` turns out to be **10**,
so the `fid` rig's damage-to-85 clamps there, and **three assertions fire on the
first conversion leg with `run.sh` exiting 1**:

```
  the fidelity part was at 10.0 of 10.0 in phase one: it was never damaged, so
  an equality across the swap is vacuous
  the damaged part is at 170.0 health at phase=t1 and was at 10.0 before the
  swap: legacyConvertOne reads the health off the old entity and writes it onto
  the new one, and a part silently repaired to full is a building this mod was
  not asked to touch
  ...and the same at phase=final
```

The first line is the one with teeth, and it is the suite's own anti-vacuity
guard: a part that was never damaged sits at `max_health`, and an equality across
the swap is then satisfied by a guest that copies nothing. One field of a
third-party mod's prototype, reproduced in Go, is what makes the fidelity half of
this whole feature a measurement.

### `fklua api check --from 2.1.16 --to 2.0.77`

| guest | surface | verdict |
|---|---|---|
| `obs-bb2data.wasm` | 0 members, 0 events, 0 concepts | **clean**, 0 findings, exit 0 |
| `obs-foreigndata.wasm` | 0 members, 0 events, 0 concepts | **clean**, 0 findings, exit 0 |

The empty surface is the expected answer and is recorded rather than assumed,
exactly as phase 5 recorded it for `obs-iactdata`: a data guest imports fkdata and
never fkapi, so it has no runtime API surface to break. Both stay STAMPED for the
running engine.

**And the generated `info.json` files are field-for-field identical to the
hand-written ones they replace** -- name, version, title, author,
`factorio_version`, description and dependencies. The one textual difference is
that the JSON writer escapes `>` inside the dependency string, which
`stamp_engine`'s clamp already matched both spellings of before this phase existed
(*"`>` survives as a literal in a hand-written manifest and as > in one a JSON
writer produced, so both forms are matched"*), and which `mig_standin`'s two
anchored rewrites do not touch at all.

### What it cost

| | Lua | Go |
|---|--:|--:|
| `belt-balancer-2` source | 94 + 9 lines | 211 lines + a shared 102-line value kit |
| `belt-balancer-2` staged | 3 files, ~10 KB | 6 files, **707 KB** |
| `bbb-mig-foreign` source | 54 + 11 lines | 102 lines |
| `bbb-mig-foreign` staged | 3 files, ~9 KB | 6 files, **475 KB** |

**These are the two cheapest packages in the estate to build**: 0.68 s for both
from a warm cache against about 1.8 s per observer, because a data guest is 63 to
76 KB of wasm where an observer is a megabyte of one. `make observers` builds and
packages all FOURTEEN from clean in **32 s** (phase 4 recorded 56 s for eleven, on
a colder Go cache).

The `mig` suite is **36.8 s** against the goldens' 36.1 -- half a megabyte of
`fk_data_module.lua` being PARSED once per Factorio load, and nothing a tick does.
The whole estate is **2m38s** collected and **2m45s** leaking, against 2m33s and
2m45s after phase 4.

### The `mar` slopes, unmoved

The gate phase 2 added. This phase touches no observer and no harness, so it is a
control rather than a risk -- and it is recorded because a shared package gaining
a file is the shape that has moved it before:

| leg | B/primitive | | leg | B/primitive |
|---|--:|---|---|--:|
| A | 1,280 | | E | 560 |
| B | 352 | | G | 3,736 |
| C | 1,209 | | F | 2,080 |
| D | 32 | | linear memory | **3.92 MiB** |

Byte-identical to phase 4's, which is byte-identical to the record this repository
has carried since the single-edge port, with the calibration at 1,136 B and 0.0%
spread.

### Deviations, all recorded rather than hidden

- **The two stand-ins do NOT share a prototype constructor**, and they define
  almost the same thing: the item is identical and the entity differs by one
  field. A constructor with a `resistances bool` on it would model the stranger
  as a VARIANT of the incumbent, which is precisely the reading the whole negative
  exists to refuse -- the runtime decision cannot be taken from the prototype, and
  the code should not suggest otherwise. What IS shared is the value kit, which is
  about Go and not about Factorio.
- **`obsdata` gained the prototype-table shorthands, copied from the shipped
  guest's `guest/go/data/value.go` rather than shared with it.** `guest/go/data`
  is `package main` -- it has to be, an `//go:wasmexport fk_data` lives in it --
  so nothing can import it. A third package both imported would couple the mod's
  own data stage to the test estate's, which is exactly what `GUEST_SRC`
  excluding `guest/go/obs` exists to prevent.
- **Each `main` forwards the shorthands through one-line functions** rather than
  package-level `var f = obsdata.F`. A function VALUE is an indirect call TinyGo
  has no obligation to devirtualize; a one-line function is inlined away, and the
  prototype tables read like the prototypes they are.
- **`logistics`' research unit is read ONCE as a map and there is deliberately no
  fallback.** Three separate deep `Get`s would be three host calls to answer one
  question and could not tell a 2.0 trigger technology from a unit missing a
  field. The shipped guest reads it the same way and then falls back to vanilla's
  own cost; this must not, because a stand-in more robust than the mod it stands
  in for would be modelling something false. Where `logistics` has no unit the
  leaves are absent, `data:extend` refuses the technology by name and the load
  stops -- the severity the Lua's own unguarded `attempt to index a nil value`
  had, delivered by Factorio's own validator.
- **The Lua's `util` require is gone and there was no hazard in it.** Both
  stand-ins wrote `local util = require("util")` before calling
  `util.empty_sprite()`, so unlike the shipped guest's legacy stub -- which called
  it as an undeclared global and worked by load-order accident -- there was
  nothing here to kill. `obsdata.EmptySprite()` is the same four fields written
  out, for the same reason: `util` is a Lua library and a data guest has no Lua.

### What phase 7 inherits

- **The estate's remaining Lua is 1,217 lines over four files**: `bench/mods/`
  (722 + 17) and `bbb-flip-test` (458 + 20), which waits for a 2.0 binary. Phase 7
  is the last one this machine can do.
- **`bench/` is MEASUREMENT INFRASTRUCTURE and its gate is not a golden log.**
  Every published performance figure in this repository was measured with that
  setup mod, so the port owes a COMPARABILITY gate the suites never needed: the
  same matrix cell, INTERLEAVED in one session, the Lua setup mod and the Go one
  against the same `dist/`, with the no-mod control in the same session, and
  `scriptUpdate` medians and item throughput compared BEFORE any baseline row is
  written. Session drift on this machine is 25-35%, which is why interleaving is
  the method and why `bench/README.md` documents it.
- **`bench/` needs Factorio exclusivity.** `test/run.sh` is safe beside another
  instance (private write-data dirs); the bench harness is not, and a second
  instance invalidates the cells rather than failing them.
- **A data-only package is an ordinary thing to reach for now.**
  `bbb-bench-setup` has a `config.lua` and a `control.lua`, so it needs both
  halves -- but the two stand-ins and the `fastbelt` fixture are three worked
  examples of the data half on its own.
- **`fkapi.TableSize` is STILL unused**, six phases in.

---

## Phase 7: the `bench/` harness's setup mod -- done 2026-08-25

739 lines of Lua deleted from one mod, and it is the LAST one this machine can
do. **The estate is 478 lines over two files**, from 8,524 over twenty-four when
the pilot started, and both of them are `bbb-flip-test`, which waits for a
Factorio 2.0 binary. Nothing else in this repository is hand-written Lua.

`guest/go/obs/bench` and `guest/go/obs/benchdata` are the setup mod and its
settings stage. Four things make it unlike the fourteen packages above it:

- **Its consumer is `bench/run.sh`, not a `test/assert-*.py`**, so its gate is
  not a golden log. Every published performance figure in this repository was
  measured with this mod, so what the port owes is a **COMPARABILITY** run: the
  same matrix cells, both setup mods, INTERLEAVED IN ONE SESSION against one
  `dist/`, with the no-mod control in the same session, because absolute timings
  on this machine drift 25-35% between sessions.
- **It is dispatched EVERY TICK** and no other package here is.
- **It is the estate's only package built `--persist=table`**, which is a
  measurement rather than a preference.
- **It holds RETAINED handles**, which the estate's own habit says not to do.

### Two findings about Factorio 2.1.16, before a line was ported

The first gate of every phase is a baseline, and this one could not be taken:
**`bench/` had not run since trunk moved to 2.1.**

**The harness was refused at the loader.** The setup mod's `info.json` said
`factorio_version: 2.0`, and 2.1 refuses such a mod before an entity is placed --
`Incompatible Factorio version (current: 2.1, required: 2.0)` -- which is exactly
the wall `test/run.sh` grew `stamp_engine` for. `bench/run.sh` stamps the staged
setup mod now, the same perl over the same two fields; the balancer mod stays
unstamped for the reason that file states at length.

**`--benchmark-verbose` SIGSEGVs, and it is the ENGINE'S defect.** It emits the
`t0` row and then crashes inside `Benchmark.cpp`, with any counter list.
Measured on a completely **vanilla no-mod save** as the control, so it is nothing
this repository ships:

| | exit | tick rows | SIGSEGV |
|---|---|--:|---|
| `--benchmark` plain, vanilla save | 0 | -- | no |
| `--benchmark-verbose wholeUpdate`, vanilla save | 1 | **1** | **yes** |
| the same over a staged bench save, any counter list | 1 | **1** | **yes** |

So `whole_us`, `belts_us`, `entity_us` and `script_us` are **unobtainable on this
engine**, and so is the per-tick `wholeUpdate` column CLAUDE.md sends every
worst-tick question to. The crash also leaves a reporter process holding the
run's own `.lock`, which fails the NEXT cell of a matrix on something unrelated
to it -- so `run.sh` skips the pass on 2.1 with the reason printed rather than
crashing Factorio once per cell. `BENCH_VPROF_FORCE=1` is how a future engine
gets re-tested.

**The consequence is that the gate rests on `avg_ms`, throughput and balance**,
which is the right place for it: `avg_ms` is uninstrumented and is exactly where
a per-tick guest dispatch shows up, and throughput and balance are deterministic
COUNTS rather than timings.

### The configuration channel

`config.lua` was a table the harness **rewrote inside the staged copy of the
mod** and the mod `require`d. A Go guest cannot require a Lua file, so the eight
keys are **startup settings**: `obs/benchdata` defines one per key,
`bench/run.sh` composes a `mod-settings.dat` per cell, and `obs/bench` reads them
out of `settings.startup` at `fk_on_init`.

- **`tools/mod-settings.py` is the PropertyTree writer**, MOVED out of
  `test/check-datastage.py` rather than transcribed. That file has had it since
  the data-stage gate needed to drive a non-default cost setting, and it was
  verified then by round-tripping the engine's own `mod-settings.dat`. Both
  callers produce byte-identical output through one function, and
  `make datastage-check` is green over all eight of its variant arms, which is
  the engine agreeing with the refactor rather than this paragraph.
- **STARTUP rather than runtime-global**, because a cell is two Factorio
  processes: `--create` and `--benchmark` read the same file directly, with no
  state carried in the save between them.
- **`guest/go/obs/protos` holds the key names.** It is that package's first entry
  that is not a prototype name, and it is there for the package's own reason: the
  mod's two wasm modules cannot import each other's half.

**AND THE CHANNEL NEEDED AN ANTI-VACUITY GUARD THAT `config.lua` DID NOT.** A
missing key in a Lua table is `nil` and the mod's own `assert` caught it; a
settings stage defines a DEFAULT for every key, so a key misspelled on either
side of this channel is not absent at runtime -- it reads back as the default,
and the cell builds a plausible rig for a configuration nobody asked for, with a
throughput and a balance that look perfectly reasonable. So the setup mod echoes
**all eight keys** on its `BENCH-SETUP` line and `run.sh` compares that line,
field for field, against what it wrote. That is this phase's red proof and it is
below.

### The gate: comparability, interleaved, one session

Three cells, three reps of each, `lua / go` back to back within every rep so a
slow-moving drift scales both, plus a separate session for the n=200 headline
geometry. `avg_ms` is Factorio's own per-run average, median of three reps.

**Correctness first, and it is exact.** Item throughput and the per-output vector
are deterministic counts, and they are IDENTICAL under the two setup mods in
every cell:

| cell | items | per output |
|---|--:|---|
| n=1 control | 8,648 | 2162, 2162, 2162, 2162 |
| n=50 control | 432,400 | 108100 x4 |
| n=50 saturated, `--mod bbb` | 435,000 | 108700, 108800, 108700, 108800 |
| n=200 control | 1,729,600 | 432400 x4 |
| n=200 saturated, `--mod bbb` | **1,740,000** | 434800, 435200, 434800, 435200 |
| mega, 404 rigs, `--mod bbb` | **1,204,725** | worst rig 480, 482, 482, 484, 482 (`3->5`) |

The n=200 row is the headline geometry and it reproduces CLAUDE.md's own figure
for it TO THE ITEM, at balance 1.001, under both setup mods.

**Then the timings, and THE THING THE HARNESS PUBLISHES IS UNMOVED.** What a
matrix cell is FOR is a delta -- `saturated` minus its own `control`, over `n` --
and the setup mod is in both halves of it:

| n=200 k=4 express | `lua` setup | `go` setup |
|---|--:|--:|
| saturated, median of 3 | 0.4440 ms/tick | 0.4695 |
| control, median of 3 | 0.4490 | 0.4740 |
| **saturated - control** | **-0.0050** | **-0.0045** |
| **per balancer** | **-0.025 µs** | **-0.023 µs** |

Both arms put the marginal whole-tick cost of one 4x4 balancer at zero, which is
what this repository has measured since M4, and they agree with each other to
0.002 µs.

**What DOES move is the absolute milliseconds, by a constant:**

| cell | `lua` | `go` | delta |
|---|--:|--:|--:|
| n=1 control (4 sink chests) | 0.1835 | 0.1900 | +0.0065 (+3.5%) |
| n=50 control (200 chests) | 0.2225 | 0.2335 | +0.0110 (+4.9%) |
| n=50 saturated (200 chests) | 0.2270 | 0.2475 | +0.0205 (+9.0%) |
| n=200 saturated (800 chests) | 0.4440 | 0.4695 | +0.0255 (+5.7%) |
| n=200 control (800 chests) | 0.4490 | 0.4740 | +0.0250 (+5.6%) |

Every one of those spans overlaps the other arm's three reps. And it decomposes
into the two things a guest pays that a Lua mod does not, both of them named in
the observer's own header before either was measured:

| term | measured |
|---|---|
| the per-tick dispatch | **~6.5 µs/tick**, which is the n=1 row: four chests cost almost nothing and what is left is the dispatch |
| the meter, per sink chest per sample | **~14 µs**, from the n=1 to n=200 slope (796 more chests for +0.0185 ms/tick over a 600-tick interval). Two host calls at the ~12.6 µs this repository has measured for years on a chest that has something in it, and one on a chest that does not |

**IT CANCELS OUT OF EVERY DELTA AND IT DOES NOT CANCEL OUT OF A ROW.** That is
the same statement `bench/README.md` has always made about the meter, and the
README says it about both terms now.

### `--persist=table`, and the 37% it was worth

**THE FIRST GATE FAILED**, and the number was not subtle: packaged
`--persist=$(PERSIST)` like every other observer, the Go setup mod was **+33% to
+54%** on absolute `avg_ms`, with the three reps of each arm tight enough that it
could not be read as drift.

`packed` mirrors the live heap into `string.pack` pages and **repacks every dirty
page after every guest call**, at about 40 µs a page. That is the right trade for
the shipped mod, which is dispatched when a player edits something and whose save
a person keeps forever. This mod is dispatched **sixty times a second** for a save
that is deleted at the end of the cell.

Measured on the same three cells, each arm against the `lua` baseline of its own
session:

| cell | `go` packed | `go` table |
|---|--:|--:|
| n=1 control | +33.2% | **+3.5%** |
| n=50 control | +35.9% | **+4.9%** |
| n=50 saturated | +37.4% | **+9.0%** |

**The whole of that was the repack.** `table` makes `storage.fk_mem` BE the word
table the guest writes into, so a store lands in what Factorio serializes with no
sync step and the steady-state cost is nothing; what it costs instead is save
size, which for a save deleted at the end of the cell is not a cost.

It is the one package in the estate whose recipe does not take `$(PERSIST)`, and
the recipe carries these numbers.

### There is no `on_nth_tick` binding and there cannot be one

`LuaBootstrap::on_nth_tick` takes a Lua FUNCTION, which a guest has no way to
hand over, and FkLua's documented answer -- a self-re-arming `fk.Defer()` -- has
**exactly the same cost**, because a one-shot timer that must reach tick 600
re-arms 600 times. So the meter is a per-tick dispatch that returns immediately on
599 ticks out of 600 where the Lua's was an engine-side modulo. It is measured
above at ~6.5 µs/tick, and `tick()` is deliberately the cheapest function in the
file: two integer tests and no host call on the ticks that do nothing.

**And the subscription has to be made from `init()` rather than from
`fk_on_init`.** An event registration does not survive a save, so a
`fk.subscribe` made from `fk_on_init` is live during `--create`, which never
reaches a tick, and gone during `--benchmark`, which is the only phase that has
any. The first cut of this file did that and **the meter never fired once** --
the cell failed on `no BENCH-METER line past tick 0`, which is a check `run.sh`
already had.

### The sink handles are RETAINED, which is a deliberate exception

Every other observer holds TILES and re-finds an entity on the tile it was built
on. Here that would be a THIRD host call per chest per sample on the harness's
dominant cost, so the sinks are `Object.Retain()`ed -- FkLua's persistent handle
space is `storage.fk_handles`, Factorio serializes the reference, and it is
adopted on load under the same-build gate the guest heap uses. `--create` and
`--benchmark` are the same build by construction. **It works at 1,553 handles**,
which is what the mega population's sinks come to.

### The MEGA smoke: identical world, and one cost that is not

`--create` over the 404-rig heterogeneous save, both setup mods, plus the
`--hitch` legs. **Every structural number is identical:**

| | `lua` | `go` |
|---|---|---|
| rigs | 404 | **404** |
| shape classes | 120 `2->2`, 80 `3->3`, 80 `4x4`, 40 `3->5`, 40 `5->3`, 40 `8x8`, 1 each of `16x16`, `32x32`, `64x64`, `65->1` | **identical** |
| hidden splitters | 4,376 (2,504 + 1,872) | **4,376 (2,504 + 1,872)** |
| surface | 152x408 | **152x408** |
| the 65-input cluster | refused once | **refused once** |
| the 64x64's first compile | 1866.8 ms audit-only, 2074.4 ms with it | 1878.4, 2044.9 |
| throughput / worst-rig balance | 1,204,725 / 1.008 | **1,204,725 / 1.008** |
| the `--hitch` legs | 3 idle-tick pairs and 6 recompile windows, all rendering `Duration: N ms` | **the same, through `harness.Profiler`** |

The hitch figures are in family and show the same within-run rise the Lua's do
(371/340, 439/466, 479/480 ms against 376/352, 462/494, 525/520): the network is
still filling, which is what CLAUDE.md's own megabase section records.

**What is NOT identical is the save.** The Go setup mod places about twenty
thousand entities inside one `fk_on_init`, and a guest that allocates during a
dispatch has no tick in which to collect anything:

| | `lua` | `go` |
|---|--:|--:|
| guest linear memory | -- | **64 MiB** |
| `script.dat` | 3.6 MB | **305 MB** |
| the mega save | 2.25 MB | **35.7 MB** |
| mega `avg_ms`, 1800t | 1.128 | 1.755, 1.845, 2.025 |

`bench/README.md` said a mega save was about 2 MB and says 35 MB now. It is
charged identically to every arm of a mega geometry, so it cancels out of the
deltas those cells are compared on, and the population and the throughput above
are what the smoke was for.

**A COLLECTED BUILD WAS TRIED AND REVERTED, and the reason it was reverted is
worth more than the heap it saved.** `-gc=custom` plus `--gc=collected`, with a
synchronous `fkgc.Collect()` at the end of `fk_on_init` (which is the one caller
`fkgc.Collect` documents itself for) and `CollectIfNeeded` from the tick handler,
took linear memory 64 MiB -> 36 and `script.dat` 305 MB -> 178. **And it silently
dropped one rig**: the `2->2` class delivered 198,016 items against 199,680,
which is exactly one rig of 120, and that rig reported `per_output=0,0` -- so its
sinks were never drained. Three further runs of the leaking build returned
`cumulative=1204725` exactly, the Lua's own figure, so the loss belongs to the
collected arm. A benchmark harness that silently stops measuring one rig is worse
than a large save by a wide margin, and chasing a conservative collector's
interaction with `--persist=table` is not this phase's subject.

So the shipped configuration is `--persist=table --gc=leaking`, and the heap is a
recorded cost with a named cause. **The cause is in the shared harness rather
than here**: `harness.place` builds a `[]KeyValue` of up to eight entries per
placement, and a `KeyValue` is two `Value`s, which is roughly 1.8 KB of
short-lived allocation per entity placed. Twenty thousand of those in one
dispatch is the 64 MiB. Reusing a package-level buffer there would be a
harness-wide change with an estate-wide blast radius, and it belongs to a pass
with its own measurement rather than to the end of this one.

### One thing the port could not do at all

The Lua set `game.map_settings.pollution.enabled = false` and the same for enemy
evolution and expansion. `LuaGameScript::map_settings` is a read-only attribute
returning a concept BY VALUE, so the generated binding hands over a copy and there
is nothing to write back; in Lua the returned table was a live proxy.
`--map-settings` was the other candidate and it takes a COMPLETE settings tree --
`Key "diffusion_ratio" not found in property tree at ROOT.pollution`, measured --
so a three-boolean patch is not a thing it accepts.

It is belt-and-braces over a world that already has nothing to pollute or evolve:
the bench surface is created peaceful with no entity autoplace, and every enemy on
every surface is destroyed at init. **The comparability gate is the evidence
rather than that sentence**, and it is one of the reasons the gate is worth more
here than a log diff would have been.

### The red proof: the configuration channel is load-bearing end to end

Two injections into `bench/run.sh`'s settings writer, run against a control cell
of `n=8`. The second is the one with teeth.

| injected | what came out |
|---|---|
| **the VALUE is wrong**: the harness asks for `n=8` and writes `n+1` | the setup mod builds 9 rigs and `run.sh` **exits 1**, printing `asked:` and `got:` side by side |
| **the KEY is wrong**: `bbb-bench-n` written as `bbb-bench-nn` | the value never reaches the guest, **the settings stage's own default does**, and the cell builds `n=1`. Caught by the same check, by name, with the same pair printed |

The control run is `rigs_built=8` and `cumulative=11584`. That is the whole chain
proved: a shell variable, into JSON, into a PropertyTree, into `mod-settings.dat`,
into a settings stage, out of `settings.startup`, into a rig count and into an
item total. **And the second row is a failure mode a golden-log diff could never
have found**, because a defaulted setting produces a perfectly well-formed log.

### `fklua api check --from 2.1.16 --to 2.0.77`

| guest | surface | verdict |
|---|---|---|
| `obs-bench.wasm` | 24 members, 1 event, 11 concepts | **clean**, 0 findings, exit 0 |
| `obs-benchdata.wasm` | 0 members, 0 events, 0 concepts | **clean**, 0 findings, exit 0 |

So it stays STAMPED for the running engine rather than gated against it, which is
what `bench/run.sh`'s new stamp depends on. The data module's empty surface is the
expected answer and is recorded rather than assumed.

**And the generated `info.json` is field-for-field identical to the hand-written
one it replaces** -- name, version, title, author, description and all four
dependencies, including the three OPTIONAL ones, which are load-bearing in the
same way `mig`'s are: this mod is present in every cell including the no-mod
controls, so it may not REQUIRE a balancer mod.

### What it cost

| | Lua | Go |
|---|--:|--:|
| `bbb-bench-setup` source | 722 + 17 lines / 28.8 KB | 1,080 + 123 lines |
| `bbb-bench-setup` staged | 3 files, ~29 KB | 8 files, **1.30 MB** |
| `dist/obs-bench.wasm` | -- | 593,104 B |
| `dist/obs-benchdata.wasm` | -- | 62,441 B |

`make bench-setup` from clean is **2.8 s**, and `bench/run.sh` runs that one
target rather than `observers` for the reason `interactive-install` names one
package rather than all: a matrix run is dozens of cells and none of them should
relink the estate.

### The estate, both arms, and the `mar` slopes

Whole estate green in **both `-gc` arms**, one invocation each, with the ported
setup mod in the tree. `make check` green with bindings and lock unmoved;
`make datastage-check` green, which is the shared settings writer proved by the
engine's own `--dump-data`.

| leg | B/primitive | | leg | B/primitive |
|---|--:|---|---|--:|
| A | 1,280 | | E | 560 |
| B | 352 | | G | 3,736 |
| C | 1,209 | | F | 2,080 |
| D | 32 | | linear memory | **3.92 MiB** |

Byte-identical to phase 6's, which is byte-identical to the record this
repository has carried since the single-edge port, with the calibration at
1,136 B and 0.0% spread. This phase touched `harness/line.go` (one new `F4`) and
`obs/protos` (constants), so it is a real check rather than a formality.

### What phase 8 inherits

- **The estate's remaining Lua is 478 lines over two files, and both are
  `bbb-flip-test`.** It cannot be ported here: the suite drives a setting that
  exists on Factorio 2.0 only, `test/run.sh` prints a SKIP for it on 2.1, and the
  first gate of every phase is a run. It moves to the `release/2.0` session with
  the other things owed there.
- **`bench/` is the only consumer of the estate that is not `test/run.sh`**, and
  it stages its own package. `copy_testmod` was not touched and does not know
  about it.
- **The bench harness is UNVERIFIED against its own history.** Every row in
  `results.tsv` older than this phase was measured on Factorio 2.0.77 with the
  Lua setup mod, and the two engine findings above mean neither variable can be
  held while the other moves. A re-run of the matrix on 2.1 is owed and is a pass
  of its own; the README says which rows are comparable with which.
- **`harness.place` allocates ~1.8 KB per entity placed**, quantified above. It
  is the largest single number the estate port has left on the table and it
  belongs to a pass with an estate-wide gate.
- **`fkapi.TableSize` is STILL unused**, seven phases in.

## The phases

Each phase is: goldens, port, the six gates, `git rm` the Lua, and a section in
this file recording what it measured and what it deviated on.

| phase | suites | what is new about it |
|---|---|---|
| **1 (done)** | `m1`, `sedge` | the harness, the build recipe, the staging seam. `sedge` brings the first observer DATA STAGE |
| **2 (done)** | `mar`, `mig21`'s observer, `qual` | the first observers with real per-tick STATE and arithmetic. `mar` reads the mod's `[BBB] heap` probe and drives 680 world operations from a schedule; `mig21` brings the first `fk_on_configuration_changed` and the first observer whose PACKAGING is load-bearing. **`flip` was in this phase and is DEFERRED** -- see below |
| **3 (done)** | `mix`, `plat`, `mig` | `plat` needs Space Age surfaces and `helpers.create_profiler`; `mix` needs infinity-chest filter rotation over 48 item names; `mig` is the only suite whose two phases run under different mod sets, and its observer is the one that reports a census |
| **4 (done)** | `m2`, `m3`, `edge` | the big ones. `m3` carries an LCG and 600 ticks of randomised churn; `edge` counts every item on two surfaces inside one tick. These are where the harness will earn or fail to earn its keep |
| **5 (done)** | the interactive staging mod | not a suite -- the world a HUMAN walks, and where the mod portal's demo scenes live. `iact` gated it already, so this phase had a headless check from the start; it is also the first consumer of `fkapi.RemoteCall`, and the only observer with a player event handler |
| **6 (done)** | `test/mods/belt-balancer-2`, `test/mods/bbb-mig-foreign` | data-stage-only stand-ins, and the first packages here with NO CONTROL STAGE. It was blocked on [`FKLUA-GAPS.md`](../FKLUA-GAPS.md) item 26 and the fix landed: `fklua mod --data-module` with no positional. The `test/fixtures/fastbelt` workaround it would have needed is deleted rather than spent. These two have no suite of their own, so the goldens are `mig`'s whole log set |
| **7 (done)** | `bench/mods/*` | LAST, and alone. Every published performance figure in this repository was measured with those setup mods, so the port has to carry a comparability gate the suites do not need: the same matrix cell, INTERLEAVED in one session, old and new setup mods against the same `dist/`, with the no-mod control in the same session. Session drift on this machine is 25-35% |
| **8** | `mig21`'s observer again, in RUST | the parity exercise. One observer written twice against one ABI is the strongest statement this repository can make about FkLua's second backend, and `mig21` is the right one: no `--create` phase, so the whole thing is one load and one set of assertions |

### What waits for a 2.0 binary

**`flip` cannot be ported on this machine and it is not a scheduling
preference.** The suite drives `bbb-multi-edge-parts`, which
`guest/go/data/settings.go` defines on 2.0.x and never on 2.1.x, so on trunk's
own engine `test/run.sh` prints a SKIP rather than running it -- and **THE FIRST
GATE OF EVERY PHASE IS A GOLDEN LOG.** There is no run here that can produce one,
so a ported `flip` would be a transcription nothing had ever executed, sitting in
the tree looking green because the suite it belongs to skips. That is the exact
shape of "a check that skips is a check that passed", which this repository
already has a section about.

So it moves to the `release/2.0` session, with the other things owed there: the
grandfather write actually landing, both arms of the flip handler, and
`sweepStackedInterfaces` against a standing multi-edge world.

**And it carries one thing no other observer needs.** `flip` drives the setting
through `remote.call('better-belt-balancer', 'set-multi-edge-parts', ...)`,
because Factorio refuses `settings.global[k] = v` from anybody but the mod that
DEFINED the setting. That is an OUTBOUND `remote.call` -- bound in fkapi as
`RemoteCall`, and **used by no observer in the estate so far**. Whoever ports it
should treat that call the way phase 3 is told to treat `fkapi.Log(Value)`:
verify it against a golden line before building anything on it.

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
