# CLAUDE.md — working context for BetterBeltBalancer

A Factorio mod: automated belt balancers that click together into arbitrary shapes and balance items across the belts feeding and draining them (belt orientation decides input vs output). Two goals, in order:

1. **Demonstrate FkLua is fit for a serious mod** — the mod's brain is written in Go (TinyGo → wasm → Lua via [`../FkLua`](../FkLua)), not hand-written Lua.
2. **Significantly outperform belt-balancer-2**, the incumbent, which manipulates transport lines from Lua every tick.

Conventions are inherited from FkLua — read [`../FkLua/CLAUDE.md`](../FkLua/CLAUDE.md) before working here. The rules below repeat only what is load-bearing daily.

---

> **TRUNK TARGETS FACTORIO 2.1 SINCE 2026-08-24, AND ITS RULE IS ONE BELT PER BALANCER PART.** 2.1 closed the collision-mask loophole that let two interface linked belts share a part's tile, so a part carries at most one edge and a cluster that asks for more is refused. The design is [`agents/single-edge.md`](agents/single-edge.md) and the rule is `guest/go/sedge.go`; "One belt per balancer part" below is the working note. **ALL FOURTEEN SUITES ARE IN THE DEFAULT AND THIRTEEN OF THEM RUN ON EITHER ENGINE**, as of 2026-08-24: `m2`, `mar` and `upg` had their rigs REBUILT single-edge -- which is also what restored the heap-slope gate the port's first three phases shipped without -- then `mix`, `plat`, `qual`, `m3` and `edge`, and finally `mig`, which is the one suite whose rigs were deliberately NOT re-laid: its world is the INCUMBENT'S, so on 2.1 an incumbent balancer converts and is then refused, and what was added is a band laid one belt per part that converts into a working network. **AND THE RELEASE/2.0 ARM IS VERIFIED SINCE 2026-08-24**: the whole estate is green on Factorio 2.0.77 in both `-gc` arms with `mar`'s slopes byte-identical, `mig21` and `mig` INVERT there (nothing is pruned, so a multi-edge save is adopted whole and a converted Belt Balancer save is grandfathered and RUNS at 3.997x and 2.996x one belt), and the fourteenth suite -- `flip`, which drives the setting that exists on 2.0 only -- is how the flip-off turned out to be a VETO rather than the sweep the design described, and how the veto's own spill was found and fixed. See [`agents/single-edge.md`](agents/single-edge.md), phase 9.

## Critical rules

- **Rebase, never merge.** Trunk is `master`, history is linear, `git merge --ff-only` is the guard.
- **Determinism is a correctness property.** Factorio is lockstep multiplayer: no entropy, no wall clock, no iteration-order dependence in anything host-visible. Lua `pairs` order over tables keyed by anything but dense integers is a desync.
- **Documentation drift is a gate failure, not a follow-up.** Update this file in the same commit as the change it describes. When a topic outgrows this file, move it to `agents/<topic>.md` and leave an index entry here. Create `agents/` files on demand only.
- **Human-facing documents follow the house style in [`agents/docs-style.md`](agents/docs-style.md).** `README.md`, `FKLUA-GAPS.md`, the bench and test READMEs are written for a public GitHub reader: no local paths, no change-history narrative, no milestone or round codes as if known, no CLAUDE.md section citations, no agent or process attribution, and no em-dashes or en-dashes anywhere. This file and `agents/` are working notes and are exempt. Read that file before editing any of them, and run its grep check before committing.
- **Measure before believing.** Every performance claim in this repo carries the command that produced it. A variant that computes a different answer is not a faster variant — benchmarks compare outcomes (item counts delivered per output) before timings are trusted.
- **Never test against `/opt/homebrew/bin/lua`** (it is 5.5). Factorio is doubles-only Lua 5.2.1; use `../FkLua/bin/lua52f` for host-side Lua testing.

## Repository layout

```
fklua.toml, fklua.lock   the mod project, written by `fklua init`. fklua.toml is
                         the ONLY place the identity, the dependencies, the
                         binding language, the asset directory, the DATA MODULE
                         and the SHIPPED GC ARM live, and
                         `fklua mod`/`gen-bindings` READ IT -- so the build
                         types none of them. EVERY ONE OF THEM ALSO HAS A FLAG
                         FORM (--name, --version, --title, --author,
                         --description, --dependency, --data-module, --gc,
                         --api, --factorio-version), and that is upstream's
                         deliberate shape rather than an oversight: one checkout
                         packaging several mods drives them from one Makefile
                         with one manifest describing the shipped one and flags
                         describing the rest. A flag OVERRIDES the key, which is
                         how `make GC=leaking` builds the other arm. There is no
                         `persist` key and that is not an omission: --persist has
                         no manifest form, so the Makefile passes it on every
                         invocation
guest/go/                the Go guest, its own module (//go:wasmimport is
                         rejected outside GOARCH=wasm). main.go is events,
                         subscriptions and logging, cluster.go is the registry,
                         compile.go is the network compiler, lifecycle.go is
                         everything that changes several things at once (clones,
                         surface deletion, rebuild-from-world, the audit),
                         carry.go is what happens to the ITEMS a teardown
                         drains -- a recompile puts them back inside the
                         network it just built, only a real removal spills
                         them ("A recompile is not a removal"), and it puts
                         them back STACKED as it found them, behind a
                         belt_stack_size_bonus gate that leaves every
                         base-only path byte for byte what it was
                         ("Stacked belts come back stacked"). A removal a
                         PLAYER caused offers what no network could take to
                         that player before the floor, like mining a vanilla
                         machine -- for EVERY part they mine and not only
                         the one that empties the cluster, and for the BELT
                         they mine at the machine's edge and not only for
                         parts, which are the two 2026-08-02 field reports
                         ("The miner's pocket").
                         commands.go is the OPERATOR SURFACE: `/bbb-audit` and
                         a `better-belt-balancer` remote interface, both
                         registered from `init` through FkLua's callback seam
                         and both reaching `auditAll`. It is what gives a PLAYER
                         a door onto the diagnostic that was script-only until
                         it existed -- and, since 2026-08-24, a
                         `set-multi-edge-parts` method that is the ONLY script
                         route to this mod's own runtime setting, because
                         Factorio lets nobody but the defining mod write
                         `settings.global`,
                         probe.go is the `bbb-insert-probe` marker: it asks
                         a CHEST the question the pocket asks a player, so
                         the one half of that path that never needed a
                         player is pinned headlessly. Read its header before
                         declaring anything else unverifiable,
                         log_*.go is the [BBB] logging switch (`make QUIET=1`)
                         and logline.go is the zero-allocation line builder
                         every log line goes through -- read its header before
                         adding a log line, because building one with `+` was,
                         measured, the entire guest heap ("The heap diet").
                         gc.go is the `--gc=collected` seam and the `[BBB] heap`
                         line. --gc=collected IS THE SHIPPED BUILD since
                         2026-08-02 ("The third decision"); `make GC=leaking`
                         builds the other arm and both stay green. The seam is
                         an import that is an EMPTY PACKAGE under -gc=leaking,
                         one CollectIfNeeded at the end of fk_on_deferred, and
                         one gcArmIfNeeded at the end of fk_on_event -- which
                         does NOT collect, it asks for the deferred flush that
                         does, because an event handler is not guaranteed to be
                         an outermost dispatch. Read its header before adding a
                         call site. It also installs the two collector knobs,
                         and the BUDGET is a measured deviation rather than a
                         preference: at the default this guest cannot hold
                         `deadlines=0` ("The floor upstream built, and the check
                         it retired"). The arming shape is not a BBB quirk any
                         more -- both refined fklua-ports guests copy it by name
                         for guests that have no fk_on_tick and must not grow
                         one.
                         limit.go is what happens when a cluster asks for MORE
                         THAN 64 PORTS: the check moved in front of the
                         teardown, so the standing network survives an edit
                         the mod cannot honour instead of being demolished for
                         nothing ("The sixty-fifth belt"), and the SECOND check
                         in front of a MERGE's teardowns, which are AddPart's
                         and are queued before the compiler ever sees the
                         cluster they make ("The merge that would be over the
                         limit"). It carries the
                         build-note store -- the miner's-pocket pattern with
                         the tile pointing the other way, at the ENTITY rather
                         than at the network -- the once-per-edge-state
                         feedback gate, the revert, which runs from flush()
                         AFTER endCarry() and may not run any earlier, and the
                         STRANDED list, the one place in this guest where
                         `nets` holds a network under a key that is no longer a
                         root. Read its header before moving either call.
                         fastreplace.go is the OTHER HALF of the one data-stage
                         line that lets a part be placed over a belt: a
                         fast-replaceable group is symmetric, so a BELT can be
                         laid on a part -- and the engine raises no event at all
                         for the part it destroys, so without the check there
                         the registry keeps a tile it calls a part which is
                         holding somebody's belt ("Fast replace").
                         host.go is DELETED: it wrote out the one host call the
                         generated binding could not make, and the binding can
                         make it now (FKLUA-GAPS.md item 16)
guest/go/plan/           the network planner: edges in, entities out. PURE Go,
                         no fkapi, no wasm imports -- so `go test ./plan`
                         runs it under a normal toolchain and proves the
                         balance property by simulation. `make check` runs it.
                         `Propagate` proves the BUTTERFLY equalises P lines
                         and never consults Ports, so the LOOPBACK wiring --
                         spare outputs [M, M+Loop) fed back into spare
                         inputs [N, N+Loop) -- went unmodelled through five
                         milestones: 36 of the 64 shapes with n,m <= 8 have
                         Loop > 0 and only 3->5 had evidence, from a
                         Factorio run. `PropagateLoop` iterates that
                         recirculation to its free-flow fixed point, and
                         every n <= m shape up to 64 lines delivers
                         S/max(N,M) per output, conserved. It stops at
                         n <= m on purpose: past it the spare outputs
                         DEAD-END and back up, which is a saturation no
                         linear model can express and which M2's `a4to1`
                         and `starve` rigs cover in the game instead. The
                         model also cannot tell WHICH row a loopback
                         re-enters on -- one butterfly pass equalises every
                         row, measured -- so the wiring IDENTITY is pinned
                         separately against Build's own ops, which is what
                         the pre-existing pairing test's COUNT check could
                         not see. And the lane-splitter stage has a
                         tripwire, because swapping it for a plain belt
                         moved no count and failed no test, while spike S1
                         measured what it costs: a left-lane-only feed
                         parks 4/0 on every output without it
guest/go/skin/           the sprite-variant mapping: a neighbour mask in, one
                         of 47 pictures out. Pure Go for the same reason, and
                         `make check` runs its five named shapes too
guest/go/carry/          the carry-pool IDENTITY: `Region` (surface, force,
                         inclusive tile box) and the ONE predicate over it.
                         Pure Go for the third time, and it exists because
                         the two questions asked of that identity -- which
                         network inherits a drained pool, and which miner may
                         pocket what nobody inherits -- were written out
                         separately and one of them lost the force ("A claim
                         is a Region"). The CLAIM STORE lives here too --
                         `Claims`, and `Region.FollowMerge`, the one rule a
                         force merge applies to both sides -- because the
                         sibling omission was the remap that followed the
                         merge into the pools and not the claims. And
                         `beside_test.go` is the third omission of the same
                         shape: a belt mined at a cluster's EDGE shrinks the
                         machine too, so it records a claim -- on a tile of
                         the NETWORK, never on the mined belt's own tile,
                         which is one outside the box by construction ("A
                         mine beside a machine is a mine of that machine").
                         The trigger needs a player; the predicate, the
                         merge and the tile need nothing, so `make check`
                         proves all three
                         sedge.go is FACTORIO 2.1'S RULE, ONE BELT PER
                         BALANCER PART: the engine CAPABILITY (a cached point
                         query for the `bbb-can-stack` marker, which the data
                         stage defines in the same `if` that emits
                         `not_colliding_with_itself`, so the guest's belief
                         cannot drift from the prototype's), the per-tile edge
                         count that falls out of classifyEdges' own walk, the
                         refusal -- which is limit.go's, shared down to the
                         wake-race guard and the hand-back -- and the
                         BRIDGING-TILE THEOREM the merge pre-pass needs,
                         because adding a part can only take edges AWAY from
                         the tiles that were already there. Since phase 2 it
                         also carries the PER-SAVE POLICY -- the
                         `bbb-multi-edge-parts` runtime-global setting, its
                         read and its one write, the heap ANCHOR that says what
                         the registry was last reconciled under, the
                         setting-changed handler and its sweep -- and A SAVE
                         BUILT TO THE OTHER RULE: the condemnation that lets a
                         refusal demolish a remnant that cannot exist, and the
                         two summaries, one per affected force, that the
                         informed flush speaks. Read its header before touching
                         either, and agents/single-edge.md before touching
                         anything about the port
guest/go/edgemode/       the mode DECISION: capability AND policy, what a flip
                         obliges, and whether to grandfather. PURE Go, the
                         fourth package to earn that -- and for a reason the
                         other three do not have: THE ENGINE ITS INTERESTING
                         STATES LIVE ON IS ONE THIS MACHINE CANNOT RUN.
                         Multi-edge is Factorio 2.0 only, so the setting
                         reading true, a player flipping it and the grandfather
                         pass writing it are unreachable from a 2.1 headless
                         run; inside `main` that fold would be four branches
                         nothing could execute, and here `make check` proves
                         all eighteen of its states -- including the one that
                         matters most on 2.1, that the write is never attempted
                         where the settings key does not exist
guest/go/data/           THE DATA GUEST: this mod's SETTINGS and DATA stages, and
                         a SECOND `main` package compiled to its own wasm
                         (dist/bbbdata.wasm, named by fklua.toml's
                         `data_module`). Three exports, one per stage file
                         `fklua mod` then generates -- fk_settings, fk_data and
                         fk_data_final_fixes -- and no fk_data_updates, because
                         nothing here wants one and a hook that is not exported
                         gets no file. main.go is the hooks, value.go the
                         prototype-table shorthands, and one file per prototype
                         family beside them. It may NOT import fkapi and
                         `fklua mod` refuses a data module that does: there is
                         no game, no script and no runtime API at these stages.
                         Read "The shipped mod holds no Lua" before touching it,
                         and main.go's //go:noinline block before ADDING to it --
                         Lua 5.2 cannot compile an arbitrarily long function and
                         this stage is exactly the shape LLVM inlines into one.
                         THREE OF ITS DECISIONS ARE NOT CONSTANTS ANY MORE:
                         recipe.go and technology.go read a startup setting for
                         what a balancer part costs to build and to research, and
                         hidden.go's `deriveHiddenSpeed` runs at final-fixes and
                         gives the four hidden prototypes the speed of the
                         fastest belt any mod loaded. All three folds are
                         guest/go/tune; what is left here is the two things only
                         a data stage can do, ask the game whether a name exists
                         and emit. "Cost, research and belt speed"
guest/go/tune/           WHAT THE DATA STAGE DECIDES that is not fixed: the
                         recipe's ingredients, the research's cost, and the
                         hidden network's belt speed. PURE Go, the SIXTH package
                         to earn that, for a reason only `engine` shares -- THE
                         INTERESTING STATES ARE STATES OF SOMEBODY ELSE'S MOD
                         SET. A fallback rung is taken only in a game with no
                         `express-transport-belt` in it and no mod set this
                         machine can install has that shape, so inside
                         guest/go/data those arms would be branches nothing
                         could execute. `Resolve` cannot return a name the
                         caller's predicate rejected, which is the whole safety
                         argument: an ingredient naming a prototype nobody
                         defined is a HARD LOAD FAILURE with this mod's name on
                         it, in somebody's overhaul pack. It also carries the one
                         check nothing else in this repo could make -- every
                         allowed value of both settings has its own
                         [string-mod-setting] locale entry, in both directions,
                         because --dump-data does not read locale and no suite
                         opens a menu
guest/go/engine/         WHICH FACTORIO THIS IS, and the FIFTH pure package. One
                         function: is `mods["base"]` 2.0.x. Both data-stage
                         hooks call it -- the collision flag and the marker on
                         one side, the runtime setting on the other -- which is
                         what RETIRED mod-data/engine.lua rather than porting it:
                         two exports of one compiled module have no second copy
                         to drift, where two Lua states had to share a required
                         file. Pure for edgemode's reason exactly: the `true` arm
                         emits prototypes only a 2.0 binary can be shown, so its
                         dump golden is deferred and `make check` proves the
                         DECISION regardless. It is also the only shape a test
                         can reach -- a package that imports fkdata cannot be
                         built by a host toolchain at all
guest/go/fkapi/          GENERATED by `fklua gen-bindings`, committed, hashed by
                         fklua.lock. Never hand-edited. The path is fixed:
                         `fklua lock` looks for exactly guest/go/fkapi/fkapi.go
guest/go/obs/            THE TEST OBSERVERS, and the second half of "no
                         hand-written Lua anywhere". A suite's mod builds a
                         world, drives it on a schedule and reports what it
                         sees; fourteen of them were control.lua files and are
                         becoming compiled guests one phase at a time
                         (agents/estate-port.md). obs/harness is the shared kit
                         -- the flat scratch surface, tile-centred placement,
                         tile lookups, chest totals, the audit marker, the tick
                         schedule and the log-line builder -- which every one of
                         those files used to carry its own copy of. obs/m1 and
                         obs/sedge are the PILOT, 2026-08-25, and obs/sedgedata
                         is the first observer DATA STAGE; obs/mar, obs/mig21
                         and obs/qual are PHASE 2 the same day, and obs/mix,
                         obs/plat and obs/mig are PHASE 3, which is the one that
                         consumed the last piece of FkLua surface the port was
                         waiting on -- obs/plat times a recompile through
                         fkapi.Log(Value), the bound global log(), which is the
                         only way anything can read a LuaProfiler's duration
                         (FKLUA-GAPS.md item 27). obs/m2, obs/m3 and
                         obs/edge are PHASE 4 and the last of the suites: the
                         three biggest, 3,350 lines of Lua between them, and
                         the phase that pinned three things nothing else could
                         -- that a profiler spanning a tick boundary needs
                         Object.Retain, that `m3`'s churn LCG has to be
                         transcribed in FLOATING POINT because Factorio's Lua
                         rounds the low nine bits off every product (a uint64
                         version diverges at the first value), and that
                         fkapi's Event* constants are FkLua's SUBSCRIBE
                         indices rather than Factorio's event ids, so
                         script.raise_event has to be handed
                         script.get_event_id(name) (FKLUA-GAPS.md item 28).
                         obs/iact and obs/iactdata are PHASE 5 and are NOT A
                         SUITE: they are the rig-staging mod a HUMAN enables to
                         walk test/interactive/README.md, which `iact` gates
                         headlessly and `make interactive-install` installs by
                         hand. It is the first consumer of fkapi.RemoteCall --
                         verified against a spike before anything rested on it,
                         because the calls it makes (freeplay's crash site and
                         intro) are SILENT and no golden diff could have seen a
                         failure -- and the only observer with a player event
                         handler, every line of which is behind the same wall
                         the miner's pocket's trigger is.
                         obs/protos and obs/obsdata are
                         phase 3's other half: the loader NAME a suite's data
                         stage defines and its observer places was written down
                         twice per suite, forced, because a control guest may
                         not import fkdata and a data guest may not import
                         fkapi -- so protos imports NOTHING AT ALL, which is
                         what lets both halves have it, and obsdata holds the
                         five fkdata calls all six data stages made identically.
                         mig21 is the
                         first with no fk_on_init at all -- it builds no world,
                         and its whole "before" is fk_on_configuration_changed.
                         IT IS ALSO THE ONE WHOSE PACKAGING IS A CORRECTNESS
                         SURFACE: it must NOT depend on better-belt-balancer,
                         because mod load order is what decides whether its
                         sample is taken before the migration or after it, so
                         the Makefile's --dependency list is red-proven rather
                         than merely written down. They are thin `main`s
                         in the MOD'S OWN module, so they share one generated
                         bindings tree with it; pruning is per WASM MODULE, so
                         what an observer calls cannot reach the mod's member
                         table -- measured, `fk_api_gen.lua` byte-identical
                         either side of this directory existing. `make
                         observers` packages them into dist/obs and test/run.sh
                         stages one from there when it exists
mod-data/                the mod's ASSETS, and SINCE 2026-08-25 NOT ONE LINE OF
                         LUA: graphics/, locale/, changelog.txt, thumbnail.png.
                         `fklua mod` merges it into the package itself;
                         fklua.toml's `[mod] data` names it. It held ten
                         hand-written Lua files until the data stage became a
                         second GUEST -- see guest/go/data below, and "The
                         shipped mod holds no Lua". The two keys cannot overlap:
                         an included file named data.lua, settings.lua or
                         data-final-fixes.lua COLLIDES with the stage file the
                         data module generates, and that is an error at package
                         time rather than a silent winner. changelog.txt is the portal and
                         in-game changelog, written to FACTORIO'S OWN GRAMMAR
                         (lua-api.factorio.com/latest/auxiliary/changelog-format.html:
                         99-dash separators, `Version: X.Y.Z` immediately after
                         each, 2-space categories ending in a colon, `    - `
                         entries, 6-space continuations, no tabs, no trailing
                         whitespace) -- the engine drops a malformed changelog
                         WHOLE and silently, and headless never reads it, so
                         `make mod` runs test/check-changelog.py over the
                         packaged copy: the grammar, plus the tripwire that
                         fklua.toml's version must be the TOP section, because
                         a release bumped without its changelog section is the
                         drift no suite can see. Red-proven on eight injected
                         defects the day it was written (2026-08-24)
tools/                   make-graphics.py -- the 47-cell adaptive sprite
                         sheet, the icon and the I/O arrows, all COMPUTED
                         rather than drawn, and their committed PNGs.
                         mod-settings.py is the OTHER thing in here and it is
                         not about art at all: Factorio's own
                         mod-settings.dat, written from scratch, which is the
                         only way to ask a settings stage a question from
                         outside. Two callers with two different questions --
                         test/check-datastage.py drives the shipped mod's cost
                         settings, and bench/run.sh CONFIGURES the bench
                         harness's setup mod, which is a compiled guest and
                         cannot require a Lua config
test/                    headless verification, FOURTEEN suites (see below).
                         EVERY SUITE'S OBSERVER IS A COMPILED GUEST under
                         guest/go/obs since 2026-08-25, phase 4 having taken
                         the last three and biggest -- `m2`, `m3` and `edge` --
                         and phase 5 the interactive staging mod, which was
                         never a suite, phase 6 the two data-stage-only
                         stand-ins and phase 7 the bench harness's setup mod
                         (agents/estate-port.md). TWO Lua files and 478 lines
                         left in the whole repository, from twenty-four and
                         8,524, and both are `bbb-flip-test`, which cannot be
                         ported on this machine because its suite SKIPS on
                         Factorio 2.1 and a phase's first gate is a run.
                         run.sh's `copy_testmod` is the seam -- a packaged
                         observer out of dist/obs when there is one, a Lua
                         directory out of test/mods when there is not, staged
                         identically either way -- so nothing else in the
                         runner knows which half a suite is in.
                         run.sh STAMPS every staged mod's info.json for the
                         engine it read off the binary, because this mod ships
                         on two arms out of one tree. Eleven answer the same on
                         either; `mig21` and `mig` INVERT and take --engine,
                         and `flip` -- which drives the multi-edge setting --
                         exists on 2.0 alone and prints a SKIP on 2.1 rather
                         than passing. `mig21` does not BUILD a multi-edge
                         world at all: it LOADS one out of fixtures-2.0/, which
                         is the only way a 2.1 binary can ever be shown one. And
                         interactive/ -- which is THE CHECKLIST ALONE since
                         phase 5 of the estate port. The rig-staging mod it
                         describes is guest/go/obs/iact now, packaged into
                         dist/obs and installed by `make interactive-install`;
                         it stages the five PLAYER gestures no headless run can
                         make, and the checklist adds TWO that need a real save
                         and a graphical client rather than a player -- adopting
                         a Belt Balancer save, and opening a 2.0 multi-edge one
                         on 2.1 -- plus the settings-menu gesture the `flip`
                         suite drives from script. THAT MOD ALSO STAGES THE
                         FIVE MOD-PORTAL DEMO SCENES, which is what makes the
                         GIF captures reproducible instead of living in a save
                         nobody kept. Every rig and every scene in it is
                         single-edge since 2026-08-24 and `iact` is what says
                         so. The checklist is what a guided playtest walks, and
                         its own "false alarm" note is a measurement ("The wake
                         race").
                         `mar` measures what one NET-ZERO world operation
                         costs the guest heap forever, `edge` drives every
                         edit that lands while a network is full and moving,
                         and TWO of them run MORE THAN ONE KIND of item
                         through a balancer: `mix`, which is how the carry
                         pool's 32-group bound turned out to be an item sink
                         ("More than thirty-two kinds"), and `plat`'s `smix`
                         band, the only rig anywhere that is multi-kind AND
                         STACKED at once -- the pair of conditions `kindAt`
                         needs to be reached at all ("Stacked sushi").
                         `mig` is the only suite whose two phases run under
                         DIFFERENT MOD SETS, and its incumbent stand-in
                         (guest/go/obs/bb2data) is a DATA-STAGE-ONLY package --
                         no control module at all, one of two here -- staged
                         under ALL FOUR of `legacyIncumbents`' names: one
                         package, its info.json rewritten at staging time,
                         because what differs between the four is the NAME and
                         nothing else. The stranger who owns the same prototype
                         name is guest/go/obs/foreigndata, the second.
                         Its observer builds a DAMAGED part, an UNCOMMON one, a
                         column of four parts across TWO FORCES and a SECOND
                         SURFACE, because health, quality, the per-force
                         technology grant and the every-surface walk are four
                         things legacy.go claims and nothing measured -- and
                         the first run of the first of them found a defect.
                         fixtures/fastbelt is not a test mod at all but a whole
                         FACTORIO MOD WRITTEN IN GO, built by the dump gate and
                         thrown away: it defines a belt at 0.4 and an UNDERGROUND
                         at 0.5, which is the only way to show this machine a
                         belt faster than the hidden network's own floor. The
                         underground is the faster on purpose -- the 0.5 the gate
                         asserts can only come from a scan that walks more than
                         one belt family. Its `inert` package is an empty control
                         guest and exists solely because `fklua mod` cannot
                         package a data-module-only mod (FKLUA-GAPS.md item 26)
bench/                   the benchmark harness (separate concern, do not disturb).
                         ITS SETUP MOD IS A COMPILED GUEST TOO since
                         2026-08-25 (guest/go/obs/bench and obs/benchdata),
                         which is phase 7 of the estate port and the last one
                         this machine can do. Three things about it that no
                         suite's observer has to think about: there is no
                         on_nth_tick binding, so the meter is a per-tick
                         dispatch; the sink chests are the estate's only
                         RETAINED handles, because a tile lookup would be a
                         third host call on the harness's dominant cost; and
                         it is the one package built --persist=table rather
                         than packed, because packed repacks every dirty page
                         after every guest call and this guest is dispatched
                         sixty times a second. Its gate is not a golden log
                         but a COMPARABILITY run -- see agents/estate-port.md,
                         phase 7
dist/                    build output, gitignored. NOTHING here is hand-written
README.md                the outward-facing page: what the mod is, how it
                         works, the headline numbers, how to build it, and
                         what is not done. This file is the working context;
                         that one is for somebody who has not read this one
```

## The shipped mod holds no Lua

**Every byte of Lua in the packaged mod is generated, and `mod-data/` is graphics, locale, a changelog and a thumbnail.** The settings and data stages were ten hand-written files there until 2026-08-25 and are `guest/go/data` now — a SECOND wasm module beside the control guest, from which `fklua mod` writes `settings.lua`, `data.lua` and `data-final-fixes.lua`, one per hook the module exports.

**The old reason for the exception was wrong, and it is worth saying which half.** It read: *Factorio's data stage is declarative with no runtime and no state, so there is nothing there for a Go guest to be the brain of.* The first clause is true of the STAGE and false of this mod's use of it. That "declarative" stage branched on the engine version to decide whether two belt-connectables may share a tile, defined a runtime setting only on the versions that have one, deep-copied four base belt prototypes and patched their speed and their sprites, computed eight arrow sprites from an offset constant, derived a technology's cost from whichever tier base says it hangs off, and defined a stub entity only when no incumbent had claimed the name. None of it could be reached by `go test`, and the version branch had to be a file two Lua states each `require`d because they share nothing else.

**Two modules and not more exports on one**, which is FkLua's own D1 and is measured rather than tidy: `require` re-executes at every stage, so a control guest hooked into the data family is parsed once per stage — **+150 ms per game load** for a 3.1 MB program the stage never calls — and its package initialisers would run against a runtime API that is not there. A data module is compiled `-gc=leaking` and packaged `--persist=none` whatever the control guest's arms are, because it runs once and dies with the Lua state that built it. `fklua mod` REFUSES a data module that imports `fkapi` at all.

**One Go module, though**, and that is what paid for the pass: `guest/go/data` and the control guest share `guest/go/engine` (the version branch, which both stage hooks now call) and `guest/go/skin` (the sprite sheet's cell count, which the prototype and the runtime had each written down separately). The shared-file problem is retired rather than re-solved.

**What must not be tidied:** the `//go:noinline` marks in `guest/go/data`. Lua 5.2 keeps a jump offset in an 18-bit field, this stage is exactly the straight-line-through-small-helpers shape LLVM inlines hardest, and at `-opt=2` all six sections folded into one wasm function that lowered to a 28,139-line Lua function the parser refused (`control structure too long near 'trap_unreachable'`, measured on the first packaged build). Marked, the largest is 3,414 lines and the emitted module is 27% smaller as well. `main.go`'s header is the long form.

**The gate is `test/check-datastage.py`**, and it is a different question from `make test` — see Verification.

**`fklua mod` still `rm -rf`s its output directory** before writing, so nothing hand-written can survive there; the tree named by `[mod] data` is merged in before the directory writer AND the zip writer, so both carry the same bytes.

### What it cost, measured either side of the port

Both packages built from clean on 2026-08-25, shipped config (`--persist=packed --gc=collected`), same FkLua, same pin, same machine:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.0.zip` | 453,975 B | **546,004 B** | +20.3% |
| `fk_module.lua` (the control guest) | 3,136,956 B | **3,136,956 B** | **byte-identical** |
| `fk_data_module.lua` (the data guest) | — | **1,498,245 B** | new |
| `fk_data.lua` (the shim, verbatim) | — | 24,106 B | new |
| `dist/bbbdata.wasm` | — | 141,799 B | new |
| hand-written Lua in the package | **45,037 B** | **0** | eleven files |
| members bound into the mod | 54 | **54** | of 4,859, none added |

**The control guest is byte-identical, and that is the shape of the whole change**: nothing in `guest/go` outside the new `data/` and `engine/` packages was touched, so `fk_module.lua` comes out the same file. What the zip pays is a second compiled guest for a stage that used to be 45 KB of source — 92 KB of download, once, against a data stage `go test` can reach and that cannot drift from the runtime consuming it.

**Nothing on any hot path moved, measured rather than argued.** A data module runs at load and is not in the game at all afterwards, so the standing gate is that the CONTROL guest's per-operation heap slopes do not move — and the `mar` suite's leaking arm came back **identical to the byte** against a build of the pre-port commit run in the same session: **1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B** per primitive over **3.92 MiB** of linear memory, 1,136 B of calibration at 0.0% spread, 0 items lost over 200 teardowns, 681 audits at drift=0. That is the set this file has recorded since the single-edge port, and it is what says a second guest costs the first one nothing.

**All fourteen suites are green in BOTH arms** over the ported package (1m49s collected, 1m59s leaking, one invocation each), and **no suite's numbers moved at all** — which, for a change that touches only the load-time half, is the only result available, and is the second validator behind the dump.

## Cost, research and belt speed -- the three things the data stage stopped writing down

**The mod portal's first feature request, in three parts, shipped as 0.3.1.** All three are data-stage decisions and none of them is a constant any more: what a balancer part costs to craft, which technology unlocks it and what that costs, and how fast the hidden network's belts run. **The DEFAULT of all three is what shipped before**, and that is the property everything else in this repository rests on -- every rate, every heap slope and both dump goldens were measured on it.

**The three folds live in `guest/go/tune`, which is PURE GO and is the sixth package to earn that.** The reason is `engine`'s and no other package's: **the interesting states are states of somebody else's mod set.** A fallback rung is taken only in a game with no `express-transport-belt` in it, and the speed derivation only does anything in a game with a belt faster than 0.25 -- neither is a mod set this machine can install, so inside `guest/go/data` those arms would be branches nothing could execute and nothing could check. What is left in the data guest is the two things only a data stage can do: ask the game whether a name exists, and emit.

### The recipe, and the ladder that is the whole risk

`bbb-recipe-cost`, a STARTUP string setting, defined on both engines (unlike `bbb-multi-edge-parts`, which exists on 2.0 alone). Startup is forced rather than preferred: a recipe is a PROTOTYPE, so what it costs has to be readable at the data stage, and `fkdata.StartupSetting` is the only kind that is.

| option | ingredients |
|---|---|
| **`vanilla`** (DEFAULT) | 4 iron-plate, 2 iron-gear-wheel, 2 transport-belt -- **byte-identical to 0.3.0** |
| `cheap` | 2 iron-plate, 1 transport-belt |
| `belt-fast` | 4 iron-plate, 2 iron-gear-wheel, 2 fast-transport-belt |
| `belt-express` | 4 steel-plate, 2 iron-gear-wheel, 2 express-transport-belt |
| `splitter` | 1 splitter, 2 iron-plate |
| `splitter-express` | 1 express-splitter, 2 steel-plate |

**AN INGREDIENT NAMING A PROTOTYPE NOBODY DEFINED IS A HARD LOAD FAILURE WITH THIS MOD'S NAME ON IT**, in somebody else's overhaul pack, before a prototype of theirs is read. That is the entire risk of the feature and it is measured rather than feared: with the guard removed and a plan pointing at a Space-Age-only belt, a base-only `--dump-data` dies on `Error in assignID: item with name 'turbo-transport-belt' does not exist.`

So every ingredient is a LADDER of candidate names and `tune.Resolve` takes the first one the caller's predicate proved present. Three properties, all of them `go test ./tune/`:

- **Resolve can never return a name the predicate rejected.** Walked over every option against every one-name-present and every one-name-missing world the ladders can reach. A nil predicate means nothing exists, so a caller who forgot one emits an empty recipe rather than an unchecked one.
- **Every ladder terminates, and at the same rung**: `iron-plate`, the cheapest thing any game with belts in it has. A game whose only item is iron plate gets every option as iron plate; a game with no iron plate at all gets a recipe with NO ingredients, which is a strange machine and a load that COMPLETES. That is the trade, stated: never break the load, always say so in the log.
- **The vanilla plan is byte-equal to a literal copy of 0.3.0's ingredient list**, written out a second time in the test rather than restated from the plan.

**The predicate is `itemExists`, and it probes the prototype's `name` LEAF rather than the prototype.** `Get("item", "iron-plate")` marshals a whole prototype across the wasm boundary to answer yes or no; `name` is mandatory on everything in data.raw, so its presence and the prototype's are the same fact and the read is one string. It walks the twenty-one prototype types an ITEM can be, because a modpack is entitled to make an ingredient a `capsule` or a `module`, and being wrong in that list is cheap in one direction only: a type left out steps a ladder past a name that did exist, and a type that does not exist simply answers absent.

**"The identity plan is the last resort" is implemented and is a tripwire rather than a live path.** `ResolveRecipe` falls back to vanilla when the chosen plan resolves to nothing -- and no plan here can do that while vanilla resolves to something, because every ladder ends at the same rung. It exists for the plan somebody adds later.

### The technology, where the prerequisite moves with the unit

`bbb-tech-cost`, the same shape: `logistics` (DEFAULT, today's behaviour), `logistics-2`, `logistics-3`. The strings ARE the base technology names, so there is no mapping table under them.

**The unit was already READ rather than written down** -- that is 0.1.0's decision and the reason this mod follows whatever an overhaul charges for a tier. What the setting picks is which technology to read, and **the PREREQUISITE comes from the same read**. Charging `logistics-3`'s science while still hanging off `logistics` would put a machine that costs blue science at a place in the tree a player reaches with red: researchable long before it is affordable, and out of order in Factoriopedia. One technology is named once and both fields come from it.

The ladder is `logistics-3` -> `logistics-2` -> `logistics` -> the vanilla fallback (20 automation science over 15 s), and a rung is taken only when that technology exists AND carries a `unit` -- the trigger-technology case 2.0 introduced and base itself uses. **The fallback emits NO prerequisite**, which is strictly better than what 0.3.0 did: `prerequisites = {"logistics"}` naming a technology nobody defined is a load error, not a cost.

Measured, base only: `logistics-2` is 200 units over 30 s and `logistics-3` is 300 over 15 s, and the gate asserts each against that technology's OWN unit in the same dump rather than against a figure transcribed here -- which is the only comparison that stays true the day base re-costs a tier.

### The belt speed, and the claim that was wrong for five milestones

**`hiddenSpeed` was 0.25 from M2 to 0.3.0 and this file called it a CEILING.** It is not. The measurement: a belt faster than 0.25 runs FULLY COMPRESSED and delivers **480 x speed items/s**, so a modded belt above 120 items/s was silently throttled to 120 at every port of every balancer -- which is what the portal's "Turbo-belts" ask was about. The false ceiling claim is purged from `hidden.go` rather than corrected in place.

0.25 is a **FLOOR** now, and `deriveHiddenSpeed` at `data-final-fixes` raises the four hidden prototypes to the fastest belt any loaded mod defines. The scan is `fkdata.Keys` per belt family and one leaf `Get` per member -- the sorted-enumeration primitive's whole purpose, because a `pairs()` walk over `data.raw["transport-belt"]` is insertion order and a data stage that branched on one would be a join refusal nobody can reproduce. Seven families (`transport-belt`, `underground-belt`, `splitter`, `lane-splitter`, `loader`, `loader-1x1`, `linked-belt`), all descending from `TransportBeltConnectablePrototype` where `speed` is mandatory. About thirty host calls, once per game load, and nothing on any tick.

**THERE IS NO UPPER CAP AND THERE MUST NOT BE ONE.** Every candidate is a value the engine ALREADY ACCEPTED, on the very prototype it was read from: a belt-connectable's `speed` is validated when that mod's prototype loads, so mirroring it onto four more prototypes of the same families cannot reach a value the loader would refuse. A cap would be this mod second-guessing a number Factorio validated, and it would re-introduce the throttle for exactly the player who installed the fast belt. A NaN is ignored, and that is a property of `>` rather than a guard.

**The ordering caveat, stated rather than hidden**: `data-final-fixes` is the LAST stage there is, so the scan sees every belt every mod defined at data, data-updates and final-fixes-before-us. A mod whose own `data-final-fixes` runs AFTER this one and raises a belt then is missed, and there is nowhere later to move to. The cost for that one player is the old behaviour: the network at the second-fastest belt's speed.

**Our own four clones are IN the scan, at the floor, and that is deliberate.** They are `transport-belt`, `splitter`, `lane-splitter` and `linked-belt` prototypes, so the families enumerate them. It is not a ratchet -- `data.raw` is rebuilt on every load, so the clone is at the floor again next time and the maximum is a function of the installed mods alone. Excluding them would be a name filter that has to stay in step with `hidden.go`, to change an answer the floor already covers. **The floor is not made redundant by that**, and the red proof below is why: lowering `SpeedFloor` lowers the creation value too, so the whole thing follows it down.

### What it costs

Both packages from clean, 2026-08-25, shipped config (`--persist=packed --gc=collected`), same FkLua (b185900), same pin, same machine:

| | before (0.3.0) | after (0.3.1) | |
|---|--:|--:|---|
| `dist/better-belt-balancer_*.zip` | 546,003 B | **560,909 B** | +2.73% |
| `fk_module.lua` (the control guest) | 3,136,956 B | **3,136,956 B** | **byte-identical** |
| `fk_data_module.lua` (the data guest) | 1,498,245 B | **1,763,788 B** | +17.7% |
| `dist/bbb.wasm` | 1,293,635 B | 1,293,683 B | +48 B, the build stamp |
| `dist/bbbdata.wasm` | 141,793 B | 173,028 B | +22.0% |
| members bound into the mod | 54 | **54** | of 4,859, none added |
| events subscribed / defines read | 23 / 4 | 23 / 4 | unmoved |

**The control guest is byte-identical and that is the shape of the whole change**: this is a load-time feature and nothing in `guest/go` outside `data/` and the new `tune/` was touched. **The `mar` slopes came back identical to the byte** in the leaking arm -- **1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B** per primitive over **3.92 MiB** of linear memory, 1,136 B of calibration at 0.0% spread, 0 items lost over 200 teardowns, 681 audits at drift=0 -- which is the standing gate for any pass that claims to touch only the load path.

### Red-proven three times, and the three catch different things

| injected defect | what fired |
|---|---|
| **the existence guard deleted** from `tune.Resolve` (with a plan pointing at a Space-Age-only belt, so a base-only game lacks it) | the unit test names every unproven ingredient in every option, and the `recipe-belt-express` gate arm dies on the ENGINE's own `Error in assignID: item with name 'turbo-transport-belt' does not exist.` -- `--dump-data exited 1`. **With the guard IN PLACE and the same plan the arm is green**, stepping past the missing rung to `express-transport-belt`: that control is what makes this a proof of the guard rather than of the plan |
| **the floor dropped** from 0.25 to 0.125 in `tune.SpeedFloor` | three unit-test failures, and **BOTH golden `data_raw_sha256` arms move** -- the four hidden prototypes come out at 0.125, half the speed the whole estate was measured on. The SPEED arm stays green, correctly: 0.5 beats either floor. That division is the point -- the goldens are the no-change proof and the speed arm is the change proof |
| **the vanilla default drifted** (iron-plate 4 -> 3) | `TestVanillaIsTodaysRecipe` names both lists, BOTH golden `data_raw_sha256` arms move, and the `recipe-vanilla` gate arm prints the drifted list against the expected one. Three independent detectors for the one change that would invalidate every recorded number in this repository |

## Build

```sh
make guest    # TinyGo -> dist/bbb.wasm.  QUIET=1 drops the [BBB] log lines
              # The default is GC=collected, FkLua's paced collector, and it
              # is the SHIPPED build; GC=leaking builds the other arm, which
              # stays green on all fourteen suites. See "The third decision"
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
make test     # headless verification, FOURTEEN suites, and the DEFAULT is all
              # of them. WHICH FACTORIO IS ON THE MACHINE IS AN INPUT: this mod
              # ships on two engine arms out of one tree, test/run.sh reads the
              # series off the binary and stamps every staged mod's info.json
              # for it, and three suites answer differently on each -- `mig21`
              # and `mig` INVERT (see their sections) and `flip` runs on 2.0
              # only and prints a SKIP on 2.1 rather than passing. The packaged
              # mod is GATED rather than stamped: its bindings are pinned to one
              # API and the ABI marshals event payloads BY NAME, so a mismatch
              # with the binary is a defect and is reported as one.
              # (m1 m2 m3 upg plat mar edge mix mig qual sedge mig21 flip iact
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
              # four of its transitions, which only Factorio 2.0 has)
make check    # the SIX pure packages' unit tests (plan, skin, carry, tune,
              # edgemode, engine); `go vet` over the data guest, which is the
              # one package no `go test` can reach (//go:wasmimport is rejected
              # outside GOARCH=wasm); bindings and lock current; gofmt
make datastage-check
              # THE DATA STAGE'S OWN GATE, and deliberately not part of
              # `make test`: Factorio's `--dump-data` runs the settings and data
              # stages and STOPS BEFORE control.lua, so it answers a question no
              # suite can ask. ELEVEN ARMS SINCE 0.3.1, ~27 s: two mod sets
              # HASHED against goldens, seven VARIANT arms (one per non-default
              # value of each cost setting, driven through a mod-settings.dat
              # the gate writes) asserting the exact ingredient list or the
              # exact research unit, and one SPEED arm that builds a Go fixture
              # mod defining a belt faster than the network's floor -- because
              # no mod set this machine can install has one. See Verification
make graphics # regenerate the sprite sheet, the icon and the I/O arrows
make observers
              # the TEST OBSERVERS: one wasm and one packaged mod per PORTED
              # suite, into dist/obs. `make test` depends on it. They are built
              # -gc=leaking (an observer has no steady state to pace a collector
              # against) and packaged from a working directory with no
              # fklua.toml in it, so every identity is a flag -- including
              # --api, without which the packager falls back to FkLua's own
              # default pin and refuses the guest. agents/estate-port.md
```

**`make mod` patches nothing.** It used to: `fk_abi.lua` passed `self` to a Factorio method (they are bound closures, so `.` not `:`) and forwarded four argument slots whatever the member declared, so **every host method call failed** with an argument-count error, and `tools/patch-abi-calls.py` rewrote the two lines on the way past. Both halves are fixed upstream ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 8) and the patch script is deleted — against a fixed `fk_abi.lua` it would corrupt what it used to repair. Nothing in `../FkLua` was ever modified.

**The last hand-written host call is gone.** `M.call` used to forward the DECLARED arity of a member, so a trailing optional nobody passed arrived as an explicit `nil` the engine counts and rejects — and `game.create_surface(name)` is exactly that shape, the call this mod's whole architecture rests on. It trims to the last argument actually PRESENT now ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 16, fixed upstream), so `fkapi.Game.CreateSurface(name, nil)` reaches the engine as `game.create_surface(name)` and the 88 lines of `guest/go/host.go` — a member id, an argument-block size and two field offsets, all hand-derived — are deleted along with the four `check-layout.py` guards over them.

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
| M2+ | `table`, re-confirmed | the marshalling arena ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 10) took a host call from 180 B of leaked heap to **zero**. It stopped the range GROWING; it could not make a span SHORT, so `packed` was still 1.7× the hitch |
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

**The measurement that decided it is not the one that was expected to.** The pass was run to see whether the page set closed the idle GC spike ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 17) — the standing theory being that `table`'s giant word table in `storage` is what Lua's collector walks. **It did not, and the theory was wrong.** `packed` mirrors the live memory into `string.pack` pages *for the save*; the memory the guest actually runs on is a Lua word table in **both** modes, and that is what the collector walks. Only `storage`'s copy differs. So the GC spike is not a cost of the `table` decision at all and drops out of the comparison entirely — 15.0 ms median under `table` against 14.2 under `packed`, over 15 runs each, with `packed`'s tail slightly worse (22.0 vs 18.1). Against a no-mod control's 1.5 ms, both are the same regression.

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

## Verification

`make test` is the gate and it is a real Factorio run, not a model. `test/run.sh` stages a throwaway mod directory per suite, `--create`s a save whose `on_init` has already built the patterns, `--benchmark`s it, and hands both logs to an assertion script. Every run points `write-data` at a private directory via `-c`, because Factorio locks its user dir and a second instance -- another agent, an open game -- would otherwise fail the run on the `.lock` rather than on anything real.

**`make datastage-check` IS A SECOND GATE AND A DIFFERENT QUESTION, and it is not part of `make test`.** All fourteen suites are about the RUNTIME: they build a world, run it and read the guest's own log lines, and by the time any of them is looking the data stage has been over for a whole load. Not one of them can see a prototype FIELD. `test/check-datastage.py` runs Factorio's own `--dump-data`, which executes the settings and data stages and **stops before `control.lua`**, normalises the result with `jq -S` and hashes it against a golden in `test/datastage-goldens.json`. About three seconds per arm, against the suites' minutes. **ELEVEN ARMS SINCE 0.3.1 and about 27 seconds**: the two hashed mod sets, seven VARIANT arms and one SPEED arm, which are not hashed and are described below.

**It exists because the data stage became a guest** ("The shipped mod holds no Lua") and a behaviour-preserving port needs an instrument that can say so. The goldens were captured from the hand-written Lua **before** a line of Go was written, and the Go stage reproduced them **byte for byte on both arms, first run**. Four things about how it is built:

- **The WHOLE dump is hashed, not this mod's prototypes.** A data stage can reach anything: `technology.go` reads base's `logistics` unit, and the Lua it replaced held that unit BY REFERENCE, so an edit to our copy would silently have edited base's technology. A subset hash is blind to the entire class of defect where a data stage damages somebody else's prototypes.
- **`jq -S` and not the engine's own `Prototype list checksum`.** Key order in the dump is INSERTION order, so re-ordering six `data:extend` calls -- which is exactly what a port from six files to one hook does -- makes byte-different dumps of an identical game; `-S` sorts that away and preserves a real field change. The engine's checksum is order-insensitive, which is the tempting shortcut, and it is measured BLIND TO FIELD VALUES (unmoved when a `stack_size` went 1 -> 42). A gate that cannot fail on the defect class a port most likely produces is not a gate. It is recorded as a smoke test and labelled as one.
- **TWO MOD SETS, because `legacy.go` has two arms.** `base` is this mod alone, where nobody owns `balancer-part` and the stub IS defined; `incumbent` stages the `mig` suite's Belt Balancer stand-in beside it, which owns the name, so the branch takes its other arm and emits nothing. A one-armed gate would never have looked at the half that matters to a migrating player.
- **A golden is per ENGINE and per MOD SET** and the file says so. The dump carries every prototype every mod defined, so a machine with different DLC produces a different hash for a mod that is perfectly fine; a golden whose engine does not match the binary is a SKIP with a message, never a failure.

**Red-proven twice, and the two proofs catch different things — which is the result rather than a formality.** Both injected, observed and reverted on 2026-08-25:

| injected defect | what fired |
|---|---|
| **one field value moved**: the part item's `stack_size` 50 -> 49 | **BOTH arms' `data_raw` hashes**, and the engine's own **prototype list checksum did not move at all** -- no prototype appeared or vanished, so the smoke test is silent on it. That is the disqualifying blindness quoted above, met in this repo rather than taken from upstream's note |
| **the legacy stub defined unconditionally**, so it lands even when an incumbent owns `balancer-part` | **the `incumbent` arm ALONE**, with `base` green to the digit -- because where nobody owns the name the stub is defined either way, so that arm cannot see it. The checksum DID move here (2044872442 -> 3917129794) and the gate reported it as a note. **This is the whole justification for the second mod set**: a one-armed gate is green on a mod that eats a still-installed neighbour's prototypes |

**THE VARIANT AND SPEED ARMS ARE NOT HASHED, AND THAT IS A DIFFERENT INSTRUMENT FOR A DIFFERENT QUESTION.** 0.3.1 made three data-stage decisions depend on things a golden cannot hold still ("Cost, research and belt speed"). A hash is right for *nothing moved*; it is wrong for *this moved to exactly that*, because a hash that changed says nothing about WHAT changed and has to be re-captured by whoever moved it -- which is the one thing a golden must never make easy. So the DEFAULT settings stay hashed, and:

- **one VARIANT arm per non-default value of each cost setting, one variable at a time.** Seven arms where three would do: the two settings decide two different prototypes and could be driven together, and an arm that moved two things is an arm whose failure does not say which. Each asserts the exact ingredient list, or the exact research unit AND prerequisite -- and the unit is compared against the SOURCE TECHNOLOGY'S OWN unit in the same dump, which is a statement no transcribed figure could make and which stays true the day base re-costs a tier. **`recipe-vanilla` is an arm too**, and it is not redundant with the golden: the golden arm writes no settings file at all, so it proves the prototype's `default_value`, while this one drives the writer.
- **the SPEED arm builds a Factorio mod out of Go**, `test/fixtures/fastbelt`, because vanilla's fastest belt is turbo at 0.125 -- HALF this mod's floor -- so on every other arm a correct derivation and one that does nothing are the same dump. Its BELT is 0.4 and its UNDERGROUND is 0.5, so the 0.5 the arm asserts can only come from a scan that walks more than `transport-belt`. It is built at gate time rather than committed, and it checks the FIXTURE'S OWN belts before believing anything about ours: a fixture that failed to load leaves the hidden prototypes at the floor, which is also what a broken derivation leaves them at.

**A VARIANT ARM DRIVES THE REAL SETTING, through a `mod-settings.dat` the gate WRITES.** That is the only way to ask a settings stage a question from outside, there is no flag for it, and there is no Lua in this repository to do it with. The format is Factorio's own binary property tree -- eight bytes of version, a bool, and a three-key dictionary of `{value = ...}` wrappers -- and the writer was verified by ROUND-TRIPPING the engine's own `mod-settings.dat` before it was used to write one. **Its anti-vacuity is structural**: if the file were ignored or unreadable, every variant arm would read back the DEFAULT recipe, and the assertion is an equality against the variant's own list. There is no way for these arms to pass while measuring nothing.

**A golden failure no longer stops them.** A drifted default moves the hash AND every arm downstream of it, and the hash alone does not say which prototype -- so stopping there would throw away the eight lines that name it. One report, one exit code.

**What it CANNOT cover here is the 2.0 flavour, and that is a property of the branch rather than of the gate.** `guest/go/engine` keys on the RUNNING engine, so a 2.1 binary produces the 2.1 flavour whatever the manifest says: no `not_colliding_with_itself`, no `bbb-can-stack`, and a mod-settings dump carrying the two startup cost settings and no `runtime-global` entry. (It was an EMPTY dump until 0.3.1, when the two cost settings became the first thing this mod's settings stage emits on 2.1; the 2.0 golden, when it is captured, will carry those two plus the bool.) The other flavour's golden is captured wherever a 2.0 binary is, which is the `release/2.0` recut, and the script prints the command. **The DECISION does not wait for it**: `go test ./engine/` proves 2.0.x true, 2.1.x false and false-safe for anything unreadable, which is the same split `edgemode` already makes one stage later.

**WHICH FACTORIO IS RUNNING IS AN INPUT TO THE HARNESS, NOT AN ACCIDENT OF THE MACHINE.** This mod ships on two engine arms out of one tree ([`agents/single-edge.md`](agents/single-edge.md), "Packaging: one tree, two releases"), and a mod whose `info.json` names the other series is refused at the LOADER before an entity is placed. So `test/run.sh` reads `Major.Minor` off `$FACTORIO --version` and STAMPS every staged copy -- every test mod, observer and stand-in -- with it: `factorio_version` unconditionally, and `base >= X.Y.Z` clamped DOWN when it names a series newer than this engine and otherwise left alone, which is what makes the whole thing a no-op on the newer one. **The packaged mod is GATED rather than stamped**, and that asymmetry is the point: a test mod's calls are ancient and few, while the mod under test is a guest compiled against a pinned API whose ABI marshals event payloads BY NAME -- so stamping it would let a field the other series added load as mandatory and read as nil, silently. A disagreement between `fklua.toml` and the binary is a real defect and is reported as one.

**THAT ASYMMETRY USED TO REST ON "a test mod is engine-agnostic Lua", AND SINCE 2026-08-25 IT RESTS ON A MEASUREMENT INSTEAD.** The estate port makes an observer a guest compiled against a pinned API too, so the sentence had to be replaced rather than reworded. `fklua api check` over the gap between the two arms this mod ships on:

| guest | its API surface | 2.1.16 -> 2.0.77 |
|---|---|---|
| `obs-m1.wasm` | 12 members, 1 event, 10 concepts | **clean**, no findings, exit 0 |
| `obs-sedge.wasm` | 17 members, 1 event, 12 concepts | **clean**, no findings, exit 0 |
| `bbb.wasm`, the mod | 54 members, 23 events, 14 concepts | **impacted**, exit 1 |

and the mod's two findings are exactly the shape the paragraph above describes: `LuaRendering::draw_sprite` loses a `light_mode` parameter and `on_player_rotated_entity` loses a `previous_mirroring` field. An observer subscribes to ONE event and calls a dozen members, all of them old, which is what makes stamping it safe. **Every phase of the port re-runs that check for the observers it ports and records the verdicts** ([`agents/estate-port.md`](agents/estate-port.md)); an observer that came back `impacted` may not be stamped and needs its own answer.

**PHASE 3 PRODUCED THE FIRST ONE, and it is what the check is for.** `obs-plat.wasm` is the only observer that touches Space Age surface, and `LuaSpacePlatform::apply_starter_pack` loses its `silent` parameter going back to 2.0.77 -- one finding out of 766 breaking changes. Its answer is checkable rather than arguable and it is in the port's phase-3 record: the parameter is OPTIONAL, that observer passes it ABSENT, and `fk_abi.lua`'s `M.call` trims the argument list to the last argument actually present before invoking, so the call reaches the engine as `apply_starter_pack()` -- which is the whole of 2.0.77's signature. It stays stamped, and the tripwire is the check itself.

**Eleven of the fourteen answer the same on both engines. Three do not**, and that is what the release/2.0 verification of 2026-08-24 was for: `mig21` and `mig` INVERT (a multi-edge save is pruned and refused on 2.1, and adopted and kept on 2.0) and take `--engine` from the series `run.sh` read, with no default -- a script that guessed would assert the wrong half and be green for the wrong reason on one of them. `flip` exists on 2.0 alone and prints a SKIP on 2.1 rather than passing.

**TEN OF THE THIRTEEN SUITES RUN ON FACTORIO 2.1.** The wall is in two layers, measured 2026-08-24 rather than assumed: 2.1 refuses a mod whose `info.json` says `factorio_version: 2.0` outright, so every suite fails at the loader before an entity is placed; and past that, a rig that puts two belts on one part is refused by the compiler. `m1` needed nothing but the one token and is green -- it is belt-free, so the rule cannot touch it -- `sedge` is the rule's own suite, `iact` is the interactive checklist's own world staged and checked headlessly, and `mig21` gets past both layers by not BUILDING a multi-edge world at all: it LOADS one out of `test/fixtures-2.0/`, which is the only way a 2.1 binary can ever be shown one.

**`m2`, `mar` and `upg` were the FIRST TRANCHE of the rebuilt estate, 2026-08-24, and `mix`, `plat` and `qual` the second the same day.** Every rig in them was re-laid one belt per part -- which in practice means every column of parts became two, a west column carrying the row's inputs and an east column its outputs -- and every table below that describes what one of them asserts has been re-recorded on 2.1.14. What is unmoved is the point: `m2` delivers the same rates against the same control belt, the same item conservation across a recompile, and the same port count for every shape, because **the same balancer over twice as many parts is the same MACHINE**; `mar`'s per-operation heap slopes are unmoved on every term the 300-hour projection uses; `upg` still adopts every network it finds rather than rebuilding one; and **`plat`'s whole stacking leg came back identical to the item** -- 1,128 items over 336 positions, +0/+128, +16/+0, +0/+48, `smix` exact over nine kinds -- because a hidden network is a function of the BELTS.

**One thing every rebuilt suite had to solve, and five of them arrived at it independently.** A conservation check forces a recompile by laying a belt on the cluster, and under this rule **a working balancer has no free face**: every part of it already carries its one belt, so that belt is now REFUSED and the check would measure a refusal instead of a teardown. Every such rig carries one extra EDGELESS part -- `m2`'s conservation rig, the interactive checklist's band B, every band of `mix` and `plat`, `m3`'s re-aimed notice placement, and `edge`'s `aout`, `ain`, `bmin` and `lim` -- and the belt goes there. Five independent passes needing the same trick is as strong a signal as this file has that it is the shape of the rule rather than a workaround.

**THE WHOLE ESTATE IS REBUILT AND EVERY NUMBER IN THIS SECTION IS A 2.1.14 GATE**, as of 2026-08-24. `mig` was the last and the only one whose ANSWER the rule moved rather than its geometry; [`agents/single-edge.md`](agents/single-edge.md)'s phase-7 section is that pass.

**Ten of the thirteen suites run both phases under ONE mod set. `mig` is the exception**, and it has to be: what it tests is what happens when a NEIGHBOUR is installed or uninstalled, so its `BETWEEN` hooks rewrite `mod-list.json` AND add or delete a mod directory between the phases (a directory that is present but not listed is added back by Factorio as enabled, so "removed" has to mean both, and Factorio requires a mod directory to be named for the mod it holds, so a renamed stand-in has to be staged under a matching directory name). Its own section is "Adopting a Belt Balancer 2 or 3 save". The other two run ONE phase each and each has its own reason: `mig21` has no `--create` at all, because its worlds were built by a binary that is gone, and `iact` has no `--benchmark`, because what it asks is answered by what the guest logged at load.

**`--benchmark` NEVER SAVES, so a leg is exactly two phases and a third is impossible.** That is the shape every leg is designed inside, and it is also why `mig`'s two name probes stop after `--create`: `create_only` runs the first phase and the same `guest_gate` over its log alone, because what those probes ask is answered by what the guest decided at load and a benchmark phase would cost a whole Factorio run to add nothing.

**The audit has TWO DOORS and they are not interchangeable.** `commands.go` registers `/bbb-audit` and `remote.call('better-belt-balancer', 'audit')`, which is how a player or another mod asks; the `bbb-audit` MARKER is the synchronous trigger a test mod's `on_init` uses, and no command can be issued from `on_init` or from a headless run at all. The command is asserted against Factorio's own `commands.commands` registry -- 2.0.77 has no `commands.run_command`, so no suite can type it -- and the remote leg drives the same handler end to end, which is evidence about the command leg because ONE id-dispatched export serves both with no branch that can tell them apart.

**The interface has a SECOND method since 2026-08-24 and its whole justification is the same one.** `remote.call('better-belt-balancer', 'set-multi-edge-parts', true|false)` is the only script route to `bbb-multi-edge-parts`, because Factorio refuses `settings.global[k] = v` from anybody but the mod that DEFINED the setting -- *"Settings can only be changed by the owning player or the mod that made the setting"*, measured, and a runtime-global has no owning player. Without it every transition of the flip handler was reachable by a human and by nothing else, which is the condition this file keeps recording as "a path whose bugs a player finds". It reaches the same `writeMultiEdgeSetting` a keypress does, so the `flip` suite drives the real path; and it is inert on Factorio 2.1 by construction, where the write is gated on a marker that is absent. `edge` asserts the method list as an exact SET, because a method here is public API another mod can come to depend on.

**Two shipped prototypes exist for the harness and are worth knowing about before adding a third.** `bbb-audit` asks the guest to re-classify the world and report; `bbb-insert-probe` asks it to run the miner's-pocket insert against whatever container or character is on the marker's own tile and report what the engine took (`guest/go/probe.go`). Both are hidden, script-placeable only, and both destroy themselves. They ship rather than living in `test/` because the alternative is a second implementation of the thing under test in Lua — and because both answer a question a player with a misbehaving save would want to ask.

**Every test mod's `on_init` ends with a `bbb-audit` marker, and that is not boilerplate.** The guest defers its recompiles to the next tick and `--create` never reaches a tick, so without it every network in every suite's save would be compiled on the first tick of the `--benchmark` phase instead of into the save. The marker is the shipped synchronous "drain and re-classify now" trigger; the same call is in the bench harness's setup mod (`guest/go/obs/bench`), guarded by the prototype's existence so the bb2, bb3 and no-mod cells are untouched.

**The save/reload between `--create` and `--benchmark` is load-bearing** in both suites: everything the benchmark phase sees exists only if the guest heap survived it, so this is also the `--persist` round-trip test.

**The guest's own `[BBB]` log lines are the assertion surface.** The test mods assert nothing; a test mod that computed the expected answer would be a second implementation of the thing under test.

**AND SINCE 2026-08-25 THE TEST MODS ARE COMPILED GUESTS TOO, one phase at a time.** **ALL ELEVEN SUITES THAT CAN BE PORTED ARE PORTED** as of phase 4 -- `m1`, `sedge`, `mar`, `mig21`, `qual`, `mix`, `plat`, `mig`, and then `m2`, `m3` and `edge`, the three biggest -- **AND SO IS THE INTERACTIVE STAGING MOD**, which phase 5 took the same day and which was never a suite at all; all of their Lua is deleted. The bench harness and two data-stage-only stand-ins follow in [`agents/estate-port.md`](agents/estate-port.md)'s phases, and `flip` waits for a 2.0 binary because its suite SKIPS on 2.1 and a phase's first gate is a golden log. Nothing about the sentence above moves -- an observer still asserts nothing -- and what the port makes load-bearing is the sentence AFTER it. **An observer's own log lines are a contract with the assertion script**: `test/assert-*.py` keys every regex on what follows the `[BBB-...]` tag and never on the `Script @__mod__/control.lua:N:` prefix the engine stamps in front of it, which is what lets a guest's `fk.Log` line satisfy a script written for a Lua mod's `log()` -- PROVIDED THE TEXT IS BYTE-IDENTICAL. So a phase is gated on a golden-log diff taken before a line is ported, and on a red proof that perturbing one token of one line fails the suite. The pilot's: **all 320 `[BBB]`, `[BBB-TEST]` and `[BBB-SEDGE]` lines across both suites and both phases byte-identical**, with only the elapsed-seconds column and the control.lua line number masked; and renaming `phase=` to `phaseno=` reports all nine of `m1`'s phases as never having run, while one space inserted into `sedge`'s `tick=` empties all three of its rate windows.

**One thing the port found that the Lua had silently relied on, and every later phase inherits it.** The `bbb-audit` marker DESTROYS ITSELF from inside the `script_raised_built` that `raise_built = true` dispatches, so by the time `create_entity` returns there is no entity to hand back: measured, the call comes back with no object and no error at all. The Lua never looked at the return; a harness that checked it -- as it must for every other piece, because a rig that did not land makes every number after it a measurement of a different world -- turned all eight of `sedge`'s audits into failures. `harness.Audit` ignores it and says why. **And the audit drain really does cross two wasm instances**: the observer's guest calls `create_entity`, the engine raises before returning, Factorio calls the mod's `control.lua`, which calls the MOD's guest, which drains its deferred queue and logs -- all before the observer's own host call returns. Eight markers, eight audit tuples, identical to the golden.

### M1 -- `test/assert-log.py`

Cluster count and every cluster size, sorted, after each of six phases. Patterns: a lone part, a line built end to end, an L (whose corner is the shape a neighbour-count heuristic gets wrong), two clusters bridged by one placement, three split shapes, a dissolve, an `on_entity_died` removal, and two surfaces carrying parts at identical coordinates so a registry that forgot to key by surface fails immediately.

**Three more phases assert the M5 sprite** off the guest's own `[BBB] skin cluster=… parts=… set=… vars=…` line — the variation it put on every part of a cluster, in (y, x) order. Phase 7 builds the five named shapes in one tick and compares all five against the numbers `guest/go/skin/ skin_test.go` derives in pure Go; phase 8 grows the line by one tile and asserts `set=2 of parts=5`, which is the incremental claim rather than the correctness one; phase 9 takes the plus apart at its centre and asserts four lone parts back at variation 1. See "M5 is done" below.

### M2 -- `test/assert-m2.py`

Twenty-one rigs on a flat scratch surface, run for 3,600 ticks, sampled at ticks 1800 and 3540. **Rates are measured over that window, not from zero**: a balancer has a pipeline (several linked-belt hops at ~32 items of fill each) and the items standing in it at the first sample would otherwise read as throughput it never delivered. `ctrl` is a bare express belt in the same save under the same conditions -- the yardstick, so "full throughput" is a comparison against the engine rather than against arithmetic on a wiki number.

**EVERY RIG WAS RE-LAID SINGLE-EDGE ON 2026-08-24 AND THE TABLES BELOW ARE THE RE-RECORDING.** Under Factorio 2.1's rule a part carries at most one belt, so a row's input and its output cannot stand against the same tile: every column of parts became two, a west column carrying the inputs and an east column carrying the outputs. The save went from **77 parts to 156** over the same twenty-one clusters. What did NOT change is anything the suite asserts -- every rate, every spread, every port count and the item-conservation figures are the ones this table has carried since 2026-08-05, because **the same balancer over twice as many parts is the same machine**: N, M and `P = next_pow2(max(N, M))` are properties of the BELTS, and the belts did not move.

Measured 2026-08-24, base only, Factorio 2.1.14 (the rows are unchanged from the 2026-08-01/08-05 recordings on 2.0.77 except where a cell says so). One saturated express belt delivered **1,306 items** over the window:

| rig | what it is | parts | per-output | total vs one belt | spread |
|---|---|--:|---|---|---|
| `sat4` | 4 in, 4 out, saturated | 8 | 1305 1306 1304 1306 | 3.998x | 0.15% |
| `sat8` | 8 in, 8 out, saturated | 16 | 1304-1306 | 7.994x | 0.15% |
| `a3to5` | 3 in, 5 out (P=8, loopbacks) | 10 | 782 784 782 783 782 | 2.996x | 0.26% |
| `a4to1` | 4 in, 1 out (spare ports dead-ended) | 8 | 1306 | 1.000x | 0.00% |
| `starve` | 4 in but only ONE fed | 8 | 326 326 328 326 | 1.000x | 0.61% |
| `block` | 4 out, the fourth blocked | 8 | 1306 1304 1306 | 2.998x | 0.15% |
| `regrow` | 3 in until tick 900, then 4 | 8 | 1306 1304 1306 1306 | 3.998x | 0.15% |
| `xsurf` | parts on a second surface | 8 | 1306 1304 1306 1304 | 3.997x | 0.15% |

Spread is the gap between the best and worst live output as a fraction of the mean; the bound the assertions enforce is 1% (2% for starvation), which is loose by an order of magnitude -- a mis-wired stage shows up as 2:1, not as 1.005:1, and the whole point of a loose bound is that it is a STRUCTURAL check rather than a tuned one. `starve` is the case that kills every chest-based design: Techrocket9 measured one output draining >9,000 items while its peers got ~80.

**The SHAPE band, and why six rigs were not enough.** The eight above are eight of the sixty-four (n, m) pairs with n, m <= 8, and `plan.PropagateLoop` proves the loopback wiring in pure Go only for **n <= m** -- past it the spare outputs DEAD-END and back up, which is a saturation no linear model can express. Seven more rigs, against the same 1,306-item control in the same save:

| rig | what it is | parts | per-output | total vs one belt | spread |
|---|---|--:|---|---|---|
| `sq3` | 3 in, 3 out (P=4, Loop=1) | 6 | 1304 1304 1304 | 2.995x | 0.00% |
| `a2to3` | 2 in, 3 out (2/3 of a belt each) | 6 | 870 870 870 | 1.998x | 0.00% |
| `a5to3` | 5 in, 3 out (P=8, Loop=3, TWO dead ends) | 10 | 1304 1304 1304 | 2.995x | 0.00% |
| `n9m9` | 9 in, 9 out -- **P=16**, four stages | 18 | 1304-1306 | 8.992x | 0.15% |
| `fdbk` | a literal FEEDBACK LOOP, 3->3 | 6 | 1304 1304 | 1.997x | 0.00% |
| `tslow` | 4 in, 4 out, one output row normal-tier | 8 | 1306 1304 1306 / **436** | 3.333x | 0.15% |
| `lane` | two SIDE-LOADED inputs, half a belt each | 4 | 653 653 | 1.000x | 0.00% |

Four of them are worth more than a row each:

- **`a5to3` is the shape the model cannot reach, and it comes out exact.** Two spare output ports dead-end, back up, block their splitters' other sides and re-route the flow -- and the three real ports, being the only way out, run at a full belt each with **0.00% spread**. Nothing was loosened to get that; the bound is the same 1% every other rig carries.
- **`n9m9` is the first four-stage butterfly ever built in a real game.** P=16, three jumper blocks, **194 entities over 16 ports** -- `Width(16)` is 13 columns against the slot's 32, so the compile is not refused and the assertion checks the port count before it looks at a rate.
- **`fdbk` closes a loop through the WORLD.** The third output belt curls round and comes back into the cluster's SOUTH face, so the machine sees 3 in and 3 out and one of each is the same belt. In steady state the loop carries L, every output carries (2+L)/3 and the loop's output IS its input, so L=1 and the two real outputs deliver exactly the two belts that went in. A network that jammed instead of settling would read as a rate collapse; it settles. **The face it returns on is FORCED by the one-belt rule rather than chosen**: every other tile of the cluster already carries its one belt, so the loop row's west part is the only tile with a free face left, and the return run therefore goes UNDER the block instead of over it. Its westward run passes directly beneath the east column, and a west-facing belt on a part's south face is neither `dir` nor `back` from that side -- the same fall-through `pass` relies on, met from the other direction.
- **`tslow` is a rate-LIMITED port, which is not `block`'s dead one.** One output row is a normal-tier belt -- exactly a third of express, with an express sink loader behind it so the belt is the only limiter. It delivers **0.334x** while the three express rows stay at a full belt each: the balancer does not throttle itself to its slowest customer.

**`lane` is the one rig chest totals cannot judge, and it is the only assertion in this suite that does not read a chest.** Both inputs are fed by SIDE-LOADING alone -- a belt joining a straight belt from the north -- so each carries half a belt on ONE lane. A vanilla splitter is lane-PRESERVING (spike S1), so a network built without the lane-splitter stage delivers all of it on one lane of every output, at the same rate, into the same chests. What separates them is where the items are STANDING, so the test mod logs both transport lines of both output rows at five ticks of steady flow and the assertion reads those. **Red-proven**: swap `ProtoLaneSplitter` for `ProtoBelt` in `plan.Build`'s head -- one token, same op count, same positions -- and both outputs go from **left=30 right=30 to left=60 right=0**, with those two assertions the only thing in the whole suite that fails and the chests reading **653 653, 1.000x, in both arms to the item**. That is the S1 measurement (`a left-lane-only feed parks 4/0 on every output`) met in a real Factorio.

**The EDGE-TYPE band, and it is the older gap of the two.** `classifySide` keys on the entity's `type`, and until these five rigs existed **only `transport-belt` had ever been run in a real Factorio** -- every edge of every rig in every suite was a plain express belt. Same save, same 1,306-item control:

| rig | what it is | parts | per-output | total vs one belt | spread |
|---|---|--:|---|---|---|
| `uio` | 2->2 entirely through UNDERGROUND ends | 4 | 1306 1306 | 2.000x | 0.00% |
| `spio` | 2->2 through vanilla express SPLITTERS | 4 | 1306 1306 | 2.000x | 0.00% |
| `lio` | 1->1 through LOADERS -- **P=1, five entities** | 2 | 1304 | 0.998x | 0.00% |
| `lsio` | 2->2 through LANE SPLITTERS | 4 | 1306 1306 | 2.000x | 0.00% |
| `pass` | the NEGATIVE: a belt line going PAST the cluster | 4 | 1306 1306 / 1306 | 2.000x | 0.00% |

- **`uio` exercises both arms of the `belt_to_ground_type` branch** in one rig: the OUTPUT half of a pair sits against the part on the way in, the INPUT half against it on the way out. The halves are created west to east so each pair takes the nearest partner; created out of order a pair would span the part and link the wrong two ends.
- **`spio` turns a comment into an assertion.** `classifySide`'s splitter case says "a splitter is two tiles wide and each half is its own edge; the per-tile search finds it once from each of the cluster tiles it touches" -- and nothing had ever checked it. One express splitter feeds both parts and a second drains both, so all four edges of the network are splitter halves.
- **`lio` is the smallest network this compiler can build**, and the first 1->1 any suite has run items through: P=1, no butterfly stages at all, **five entities**. It is also the loader arm of `classifySide`, which `agents/design.md` lists as *wishlist* interop -- it is implemented, and it works. The 0.998x is one item pair of pipeline, not a bottleneck. Under the one-belt rule it is **two parts**, and that is the floor for the whole mod: one part to carry the input loader and one to carry the output, so a one-part balancer no longer exists at any shape.
- **`lsio` is the one that found a defect**, and it is written up on its own below. Base ships the `lane-splitter` TYPE and not one buildable entity of it (the type exists for Space Age's turbo lane splitter), so the test mod clones the mod's own hidden `bbb-lane-splitter` into a placeable `bbbt-lane-splitter` -- a real prototype a real Factorio validated, rather than one the harness invented.
- **`pass` is the negative and it is the one that could be silently wrong.** A belt line runs east along the top parts' NORTH faces: from that face `dir` is north and `back` is south, so an east-facing belt is neither and falls through. That is the incumbent's accepted limitation (*a belt curving away is not an output*) met from the other side, and until this rig nothing asserted it. Both halves are checked -- the balancer delivers its own two belts exactly, AND the passing line's own chest gets a full belt, so nothing was stolen from it.

  **Under the one-belt rule it acquired teeth it did not have**, and this is where the whole re-lay pays for itself. The line now runs over TWO parts, and both already carry their one belt -- the west part its input, the east part its output -- so a classifier that read the passing line as an edge does not merely deliver an odd rate: it takes two tiles to two belts each and the cluster is REFUSED outright. **Red-proven 2026-08-24** by turning that one line SOUTH, which is one token in the rig: `pass` goes to **0 0, 0.000x**, the passing line's own chest gets **0**, and six assertions fire -- the three `pass` rate checks, `nets != clusters`, the audit's `refused=1`, and the named refusal (`cluster 147 has 2 parts carrying more than one belt`). On the pre-port geometry the same injection would have cost a rate and nothing else.

**The final audit is part of both bands.** After the last sample, on a world nothing has touched since tick 900: **21 clusters, 156 parts, 21 networks, drift=0, unbuilt=0, refused=0**. A classifier that read `pass`'s line as an edge, or `fdbk`'s return belt wrongly, would have a fingerprint the world does not match and would say so there rather than merely delivering an odd rate. Every shape's PORT COUNT is asserted too, before its rate is looked at: `3->3` and `2->3` over 4, `5->3` over 8, `9->9` over **16**, `1->1` over 1. **`nets` against `clusters` is asserted as well**, and that is not decoration — it is the only line in the suite that can name a cluster the classifier simply did not see.

**And since the re-lay the CLUSTER AND PART COUNTS are asserted against a constant in `assert-m2.py` rather than read off the guest**, exactly as the `mig` suite's cluster count is and for the same reason: `(21, 156)` is a statement about the SAVE, and a rig that quietly lost a row -- or that was rebuilt one column wide in the old idiom -- moves it while every rate it still has stays plausible. Two more lines say the same thing from the other side: **the audit's `refused=` column must be zero**, and **no single-edge refusal may be issued anywhere in the run**. Every rig here is built to the rule, so a refusal is a defect in the SAVE and not in the guest.

### The lane splitter the classifier could not see, and the SECOND gate

**A balancer fed and drained entirely through lane splitters compiled to nothing, delivered nothing, and reported `drift=0 unbuilt=0` while it did.** Found by writing the rig before the code, which is why the numbers below are measurements of the defect rather than a description of it.

A lane splitter is a 1x1 directional belt-connectable — the same reading as a transport belt, `d == back` in and `d == dir` out — and `classifySide`'s switch did not name the type. The consequence is not "an edge classified wrongly", it is **no edge at all**: a cluster with an empty edge list is a legitimate half-built state, `plan.Build` returns early, and **a fingerprint over an empty edge list matches the world perfectly**, so the audit is genuinely satisfied. The rig read `0 0` while the other twenty were green, and the audit read **21 clusters, 20 networks, drift=0, unbuilt=0** — one whole balancer inert, and one number anywhere in the suite that could see it.

**It took two changes and only one of them was the obvious one.** Adding `"lane-splitter"` to `classifySide`'s switch moved **nothing at all** — same `0 0`, same 21/20 — because the edge classifier has **two gates** and the switch is the second. The first is the `type` array carried by the `find_entities_filtered` behind every edge query, applied by the engine in C++ before the guest sees anything, and a family missing from it is never returned to be switched on. Both lists live in `guest/go/compile.go` (`beltTypeVals` in `initBuffers`, and the switch in `classifySide`), **nothing makes them agree**, and until `lsio` nothing would have noticed. Both carry a comment saying so now.

With both gates open: **1306 1306, 2.000x, 0.00% spread**, and the audit at **21 clusters, 21 networks**.

**Blast radius: none, measured rather than assumed.** The change widens a filter and a switch, so the risk is a family that starts being classified where it was not. `m3` and `edge` between them assert classification harder than anything else in the repo and contain no lane splitter at all; both were run against the pre-change and post-change guests in the same session and every number is identical — `m3`'s 15,856-of-16,000 stress recovery, its twelve rig rates and its `drift=0 unbuilt=0`; `edge`'s 128-item `bmin` spill, its 118-item by-hand teardown, its 225 teardowns at a median of 72 handed back, and its visible-surface probe at 60/60/65/66/49 with **0 off a part tile** on every sample.

**Item conservation across a recompile** is checked exactly. A 2->2 network is fed with nothing draining it until every belt and splitter in it is full, and then, inside a single tick with no other movement possible: count every item on both surfaces, force a recompile, count again. **2,680 before, 2,680 after**, the teardown having drained 72 items out of the hidden network and the rebuild having **put all 72 back INSIDE it** — which is the second half of the assertion since 2026-08-02: a recompile that reached the spill path fails the suite. See "A recompile is not a removal". Every one of those four numbers survived the single-edge re-lay to the item, which is the sharpest statement in this section that the machine did not change.

**What the re-lay DID change is how the edit that provokes the recompile is made**, and it is worth knowing before anyone edits that rig. The old check laid a belt on the cluster's one free face -- the north side of the top part -- and under the one-belt rule that part already carries its input, so the same gesture would now be REFUSED and the check would measure a refusal instead of a recompile. So the block is three rows tall and **the bottom row's two parts carry nothing**: the belt goes against the edgeless west part, which is a third input and takes P from 2 to 4. Same network coming down, a bigger one going up, and the reinsertion therefore fits with room to spare -- which is the correct shape for a conservation check and not the one for a shrink (see "A mine beside a machine is a mine of that machine" for what reading a non-shrinking shrink as evidence once cost).

**"Inside a single tick" is now something the test has to arrange**, and that is the one place batching changed a suite rather than a number. The recompile is deferred to the next tick, so the check lays its belt and then places a `bbb-audit` marker, which drains the queue synchronously inside the same dispatch. Counting a tick later would have worked for the assertion and would have stopped being a measurement of the teardown: items move for other reasons in between.

**The recompile hitch this suite reports, re-recorded on the single-edge rigs.** Same method as always -- the probe opens in the mutating tick and closes in the flushing one, so the window contains one whole engine tick as well as the recompile, and each cell is the median of three runs minus that run's own `idle tick pair, nothing pending` control (0.31–0.36 ms). Measured 2026-08-24, Factorio **2.1.14**, base only, shipped configuration:

| forced teardown-and-rebuild of a SATURATED rig | ms |
|---|--:|
| `sat4` (4->4 over eight parts, 32 entities), one input removed | **10.75** |
| `sat4`, full | **10.43** |
| `sat8` (8->8 over sixteen parts, 84 entities), one input removed | **24.03** |
| `sat8`, full | **22.47** |

**These are NOT comparable with the 11.55 / 25.72 ms in "A recompile is not a removal"**, and the reason is two variables at once rather than one: that table was measured on Factorio 2.0.77 with four-part rigs, and this one on 2.1.14 with eight-part rigs. Nothing here is claimed as a win or a regression -- it is what the suite reports today, and the number to compare a future guest against.

### M3 -- `test/assert-m3.py`

Twelve rigs on a flat scratch surface plus three more surfaces, run for 1,560 ticks. Every rig is damaged in one specific way before the measurement window (t=540 to t=1500) opens, and what is asserted is **what that damage should leave**, measured against `ctrl` -- an uninterrupted express belt in the same save. Re-measured 2026-08-24 on Factorio 2.1.14 with every rig REBUILT SINGLE-EDGE, base only; one saturated express belt delivered 720 items over the window, and **not one rate, spread or "exactly zero" in this section moved**:

| rig | what was done to it | per-output | total | expected |
|---|---|---|---|---|
| `live` | nothing -- the witness, including the hidden surface being deleted under it | 720 x4 | 4.000x | 4.000x |
| `clone` | an area clone AND a brush clone taken of it | 720 720 | 2.000x | 2.000x |
| `bp` | a blueprint taken of it | 720 720 | 2.000x | 2.000x |
| `forceA` | a second force's parts built directly against it | 720 720 | 2.000x | 2.000x |
| `forceB` | ... and it, against the first force's | 720 720 | 2.000x | 2.000x |
| `ghost` | built by reviving ghosts (`script_raised_revive`) | 720 720 | 2.000x | 2.000x |
| `paste` | a blueprint's 14 entities placed as real entities in one tick | 720 720 | 2.000x | 2.000x |
| `bots` | built by construction robots from ghosts | 690 690 | 1.917x | pipeline still filling |
| `bdie` | an input belt killed with `die()` | 360 360 | 1.000x | 1.000x |
| `noev` | an input belt `destroy()`ed with NO EVENT, then put back on the same tile | 720 720 | 2.000x | 2.000x |
| `swap` | an express input belt fast-replaced with a FAST one (2/3 the speed) | 600 600 | 1.667x | 1.667x |
| `died` | the EAST part of a row killed with `die()` while the network was full | 720 | 1.000x | 1.000x |

Spread between live outputs is **0.00% on every rig**. `swap` is the sharpest of these: missing the recompile entirely reads as 2.000x and classifying the new belt as absent reads as 1.000x, so 1.667x is only reachable by getting it right. `died`'s orphaned row is asserted to have received **exactly** zero further items. What else the suite checks, from the guest's own log:

- a blueprint over a live balancer captures 14 entities and, of ours, **only `bbb-balancer-part`** -- no hidden prototype and no visible interface -- and exactly **4** of them, which is what a 2-in/2-out rig is under the rule;
- for a 14-entity paste in one tick: **4 parts registered inside that tick, 0 network builds inside it, and exactly 1 by the deferred flush on the next** (see below -- all three are assertions, and the middle one is what says the work was deferred rather than merely cheap);
- both clone paths reconciled, **8 copied interfaces destroyed**, **0** hidden entities cloned onto the destination surface, and **4 parts** on each destination;
- deleting a balancer's surface unregisters its 4 parts in 1 cluster; deleting the **hidden** surface logs an alert, recreates it and rebuilds all 18 clusters, and `live` does not lose a single item to it;
- the audit finds `drift=1` immediately after a belt is turned around with no event at all, and the final one finds a world of **14 clusters, 59 parts and 14 NETWORKS** at `drift=0 unbuilt=0 refused=0` after 600 ticks of randomised churn. `nets` is asserted against a constant beside `clusters` because **`unbuilt=0` is weak evidence**: a cluster with inputs and no outputs is a legitimate half-built state and is never counted, so a rig that quietly lost its network satisfies it. Every cluster in this save has both, so every one must have a network;
- **not one one-belt-per-part refusal, anywhere.** That is a decision about the churn as much as an assertion about the rigs: its six randomised edits are aimed at a row's own single input, its own single output and an EDGELESS part below the west column, so no tile can ever be asked for a second belt. A churn that generated refusals would make the compile, build and teardown counters below a function of the RULE rather than of the lifecycle path under test, and would leave clusters standing refused at the final audit. The three ways of reaching that refusal are the `sedge` suite's;
- **the I/O arrows have not leaked**: 58 rendering objects against 58 standing visible interfaces, after ~100 teardowns, two surface deletions (one of them the hidden surface under every network at once), cloned interfaces and interfaces killed from outside. The guest stores no rendering id, so this is the assertion that the engine's "a rendering object dies with its target entity" really is what takes them down;
- final cluster sizes are one 4 and thirteen 2s -- two forces' parts touching never merged;
- **item conservation under churn**: 16,000 items in, 15,856 recovered after every stress cluster is torn down (re-measured 2026-08-05 from clean in both arms; the 15,857 this file carried was one item of drift whose commit of origin nobody identified). 0.90% lost over ~100 teardowns, to fractional item positions and whatever a splitter holds outside its transport lines. The assertion is `recovered <= inserted` (nothing may be minted) and `lost <= 2%`. This one also ends with an audit marker, for the same reason the recompile check does.

**Two rigs needed more than a re-lay, and both are the same shape**: an edit that used to land on a working balancer's free face has no free face to land on any more. `phase_silent_notice`'s "unrelated placement" is DIAGONAL from the cluster now -- inside the two-tile neighbour gate, so the cluster is re-classified, and orthogonally adjacent to nothing, so no tile gains a second belt -- and `died` kills the EAST part of the second row rather than the west one, which is what takes that row's OUTPUT off the machine and leaves its chest orphaned. Killing the west part would take an INPUT off instead and leave both outputs live at half a belt each, which is a different measurement.

**Red-proven**: put the notice belt back where it used to be -- a south-facing belt on the top west part's north face -- and four assertions fire. `noev` delivers **0 0, 0.000x**, one cluster is refused for the one-belt-per-part rule (twice, once per audit that reaches it), the final audit reads `refused=1`, and the world tuple comes out **(14, 59, 13)** against the (14, 59, 14) the rigs build. **`unbuilt` stayed 0 through all of it**, which is exactly why the `nets` column had to exist.

The 600-tick stress phase drives four clusters with a `storage`-carried LCG, adding and removing belts and parts every six ticks. Cumulative counters at the final audit, **before batching → after**: 288 → **139** compiles, 121 → **89** builds, 107 → **75** teardowns, 1,251 → **950** `create_entity` calls. No script errors and no `[BBB] error:` in either. Roughly half the work for the same outcome, on a workload that was never a blueprint paste -- one mutation every six ticks is the case batching should help *least*, and it still halves it, because a single removal queues both the root that died and the roots that survived and they now share one drain.

### What M3 implements and does NOT verify

An unverified path is not a tested path, and the list is short but real:

| path | state |
|---|---|
| `on_undo_applied` / `on_redo_applied` | **Implemented, unverifiable headlessly.** Three separate walls: a headless `--create` has no player, so `player_index` resolves to nothing; `LuaUndoRedoStack` can be read and edited but never APPLIED, so no script can trigger an undo; and `script.raise_event` refuses outright -- *"on_undo_applied (ID 191) can't be raised through script"*. What IS verified is the machinery the handler calls: the M3 `swap` rig turns a belt around with no event at all (which is exactly what an undone rotation does to the world) and a re-classification pass finds it -- `drift=1`, repaired. The handler adds only the decision to run that pass, scoped to the player's surface where there is one and over everything where there is not. Both subscriptions are **masked** over `actions`, so the field the handler ignores is never marshalled -- see "Which FIELDS reach the guest". |
| `on_player_rotated_entity`, `on_player_flipped_entity` | **Implemented, event not exercised** -- both require a player. They are also the only two of this mod's per-entity subscriptions that carry **no filter**: `runtime-api.json` gives a `filter` concept to 30 events and neither is one of them, so both arrive for every rotation on the map and are rejected by the in-guest position gate. The world change they report (a direction change; an underground's ends swapping) is exercised directly, and lands in a recompile. Both are read through the generated `fkapi.ReadOnPlayerRotatedEntity`/`...Flipped...`, so their layout moves with the pin. |
| `on_space_platform_built_entity` / `_mined_entity` | **Implemented, event not exercised.** The `plat` suite verifies the substantive claim end-to-end -- a balancer whose parts are on a platform surface and whose network is on the hidden surface, at exact rate -- but it builds them with `create_entity{raise_built=true}`, which raises `script_raised_built`. Reaching the platform events needs a player on the platform or the platform's own construction robots, and `plat.hub` is nil immediately after `apply_starter_pack()`, so the robots never appeared. Both are read through generated payload structs -- `on_space_platform_built_entity` was one of the four that deferred over a dictionary `tags` field. |
| `on_pre_surface_cleared` / `on_surface_cleared` | **Implemented, not separately exercised.** They share `dropSurface` and `hiddenSurfaceGone` with the delete pair, which are exercised. |
| **the miner's pocket** — a player mining a balancer keeps what the network was holding | **Implemented; only the TRIGGER is unverifiable headlessly, and that is narrower than it was.** The wall is measured rather than quoted: a headless `--create` has no players, so `game.get_player(1)` is nil (`players=0`), and `on_player_mined_entity` is not raiseable — `script.raise_event` refuses it outright, *"on_player_mined_entity (ID 74) can't be raised through script"*. **The `edge` suite asserts both refusals**, so the day either falls the run fails and asks for the real test. Since the 2026-08-02 field report the suite ALSO pins the two halves that never needed a player: the **insert arithmetic**, asked of a steel chest through the same `insertOne` from inside the same deferred flush (`bbb-insert-probe`, four legs, guest and Lua numbers cross-checked), and the **quantity** — a saturated balancer taken apart one part per tick, 118 items that a player now receives instead of the floor, and, since the second field report, an output belt laid on a running balancer and mined again, 128 more. Since the force correction it also pins the **claim predicate**: which network a claim belongs to is pure guest logic, so it lives in `guest/go/carry` and `make check` proves it — including the two-force overlapping-box case the shipped code got wrong, and the tile a mine BESIDE a network is claimed under, which is the second report's own trap. Plus the fallback and the negative as before. **What is behind the wall is one line of policy and no arithmetic**, which is exactly what the two reports were about: both were the list of removals that get a beneficiary being narrower than the sentence describing it. See "The miner's pocket" and "A mine beside a machine is a mine of that machine". |
| **the over-limit feedback** — a player told their balancer is full, and handed the belt back | **Implemented; only the TRIGGER is unverifiable headlessly, and it is the same wall as the pocket's.** A headless `--create` has no players, so `game.get_player` resolves to nothing and `revertOne` returns before it mines anything — which means the flying text, the `utility/cannot_build` sound and the hand-back cannot fire in any suite. Everything ELSE about the pass is pinned by the `edge` suite's `lim` leg: the refusal happening before the teardown (0 items on the ground where the unfixed guest put 1,690), the standing network still delivering at its old rate across the edit, the audit's `drift=1 unbuilt=0`, the feedback gate firing **exactly once** per distinct edge state rather than once per audit, and the ROBOT arm of the feedback end to end — every headless build is a script build, so the fork always takes `force.print`, and the suite asserts that the LocalisedString crossed and the `LuaForce` resolved. Plus the negative, which is the half with teeth: **zero pieces handed back over the whole run**, so a revert firing for a script build fails. Since 2026-08-05 the same holds for the MERGE form -- the `brdg` leg refuses a bridge between two working balancers and the same trigger is the same one line behind the same wall. **And for the WAKE RACE**, which needs a player AND a same-version reinstall -- a build that lands in the one dispatch where `rebuildFromWorld` has run and the event's own build note does not exist yet. No suite can reach it (every suite's two phases run one mod set built once), and what they pin is the same negative: `player_index` is zero, so neither head is ever entered. See "The sixty-fifth belt", "The merge that would be over the limit" and "The wake race". |
| **the FAST-REPLACE gesture** — a part placed over a belt, and a belt placed over a part | **Implemented; only the CURSOR is unverifiable headlessly, and that is a narrower wall than the pocket's.** `create_entity{fast_replace = true}` is not a player: handed a replace the engine would refuse it falls back to CREATING, so the `edge` suite's `frepa`/`frepb` rigs ask `can_fast_replace` first and drive only what a cursor could. Everything that is not the cursor is pinned there — the engine's own `can_fast_replace` answers in both directions and on an interface-carrying part, the belt really being gone and the part really being registered, the 3→2 network that results balancing to 0.00%, the reverse SPLIT reaching the registry (`15 clusters / 195 parts, drift=0 unbuilt=0`), and the guest's own removal line firing exactly once. What a human still has to see is the preview over the cursor and the replaced piece arriving in an INVENTORY rather than on the ground; a script build has no player, so every headless number here is the spill arm. See "Fast replace". |
| ~~A real second guest BUILD~~ | **Now verified.** This used to read "the same code path to the byte, because this mod exports no `fk_migrate`". It exports one now, so the `upg` suite's build-stamp bump reaches `on_configuration_changed`, the hook, and a rebuild driven by it -- and `assert-upgrade.py` asserts the guest was told *and* that the notification, not the first-event fallback, is what ran the scan. What is still not exercised is `fk_migrate_adopt`, which this mod must never export. |

### `upg` -- `test/assert-upgrade.py`, then M2's own assertions again

A mod upgrade, done properly: the M2 map is created by one guest and loaded by another (version bumped, build stamp changed), so the saved heap is declined and the registry is empty while the world is full of running networks. Measured 2026-08-24 on Factorio 2.1.14, over the re-laid single-edge rigs:

    after the upgrade: 4 surfaces scanned, 156 parts, 21 clusters
    21 networks adopted as they stood, 0 rebuilt

and then **M2's entire assertion set is run over the result** -- 3.998x on `sat4`, 7.994x on `sat8`, exact item conservation across a forced recompile, all of it. A network adopted with the wrong slot, or adopted when it should have been rebuilt, does not balance. The whole rebuild-and-adopt cost ~19 ms on top of the recompile it happened to land inside.

**It grew from 9 clusters to 21 when M2 gained the shape and edge-type bands**, and the twelve it gained are the interesting ones: adoption re-derives every edge list from the world through the same `classifyEdges`, so this is also where a P=16 butterfly, a feedback loop closed through the world, and edges made of undergrounds, splitters, loaders and lane splitters are adopted rather than merely compiled. The single-edge re-lay doubled the PARTS it walks -- 77 to 156 -- and moved nothing else: the twenty-one networks it re-derives are the same twenty-one networks, so all twenty-one still adopt.

**THE BUILD STAMP IS WHAT DECLINES THE HEAP, NOT THE VERSION, and `bump_build` moves both because only one of them does the work.** That is `agents/single-edge.md`'s S2 result 6 stated as a property of this suite, and it is red-proven here as of 2026-08-24: leave the `perl` that rewrites `build = "…"` out of `bump_build` so the mod version alone moves, and the saved heap is ADOPTED, `fk_migrate` never fires, `rebuildFromWorld` never runs, and `assert-upgrade.py` fails with *"the guest never rebuilt its registry from the world after the upgrade -- every later placement would have built a SECOND network"*. A leg that bumped only `info.json` would test nothing at all and would say so nowhere.

### `plat` -- `test/assert-plat.py`, the only suite that needs Space Age

Three legs, together because the DLC is all they have in common and a second DLC-only suite would cost a second Factorio run for one rig.

**The platform surface.** A 2->2 balancer built on a **space platform surface** (`force.create_space_platform` + `apply_starter_pack`), with the hidden network on the global hidden surface as always -- so its linked belts cross from a moving platform to a surface that is not going anywhere. Against an uninterrupted belt on the same platform: **676 and 676 against 676, 2.000x, 0.00% spread.**

**Belt stacking.** Four bands on a scratch surface, on their own force with `belt_stack_size_bonus = 3` -- so the same save carries both arms of the guest's stacking gate, the platform rigs being on a force whose bonus stays 0. What is asserted is the stack PROFILE of the hidden surface either side of a forced recompile: **0 extra single-item positions** on a stacked network, the whole crossing accounted for by stacked items, **+16 singles and +0 stacked** on the unstacked band, and 0 spills. Full table and method: "Stacked belts come back stacked" below.

**Stacked sushi** -- a fifth band, `smix`, on the same stacking force, and the only rig in this repo whose hidden transport lines carry more than one item KIND and more than one item per POSITION at the same time. That pair of conditions is what `kindAt` needs to be entered at all, and until 2026-08-05 nothing anywhere met it. **Conservation exact per (name, quality) over nine kinds, +0 single and +64 stacked across the recompile, 0 spills**, with the anti-vacuity sample showing 14 of 24 hidden lines carrying two names and 6 carrying one name at two qualities at the moment the teardown read them. Full table, both red proofs and the rotation constant: "Stacked sushi" below.

**The five bands share a save and do not share a measurement.** `smix` runs seven item names that no other band in this suite touches -- every other rig here is iron plate -- and every count and every profile in `control.lua` filters by name, so the four bands above keep the figures they were recorded with to the item while the fifth is measured independently. One Factorio run, two answers.

**The geometry doubled on 2026-08-24 and not one number moved.** Every band is two columns of parts under Factorio 2.1's rule, plus one EDGELESS part on the end for its recompile's belt to land on -- a working balancer has no free face any more, so the belt each `recompile` used to put on the block's north face would be REFUSED rather than compiled. Re-measured that day on 2.1.14, every figure in this section and in the two below came back **identical to the item**: 676/676 on the platform, 1,128 items over 336 positions at `formed`, +0/+128, +16/+0, +0/+48, 3.971x at 0.54% spread, and `smix` exact over nine kinds at +0/+64. That is the expected result rather than a lucky one -- the hidden network is a function of the BELTS, and the belts did not move -- and it is what makes the re-lay checkable at all. The save is **five clusters over thirty-two parts** across two surfaces and the final audit is asserted against those constants, with `nets == clusters` and `drift=0 unbuilt=0 refused=0`.

### `mar` -- `test/assert-marathon.py`, the permanent-heap slope

Under `-gc=leaking` every transient guest allocation is permanent, so the interesting number is not the heap at any one moment but the SLOPE: what one complete place-and-remove cycle adds that never comes back. This suite drives 680 net-zero world operations over 4,600 ticks and reads the guest's own `[BBB] heap post-audit sys=… alloc=…` probe after each one. Full method, numbers and the projection they support: **"The marathon save"** below.

Two properties make the table attributable rather than a list:

- **Every leg pays exactly one audit**, and `cal`/`calA`/`calZ` measure that audit with the world untouched, before, in the middle of and after the run. It came out **1,136 B all three times, 0.0% spread**, and is subtracted from every other leg. A drifting calibration would fail the suite: nothing below it could then be attributed.
- **Every leg leaves the world exactly as it found it**, so every audit sees the same three clusters and costs the same.

The suite is green in both `-gc` arms and asserts a different thing in each. Under `collected` the heap is given back, so `alloc` is not a permanent total and its slope would be arithmetic on noise; the script detects `gc=1` and asserts what never shrinks in either arm instead — linear memory, the live set, and that the collector is not being outrun.

**THE SECOND OF THOSE TWO PROPERTIES WAS A COMMENT AND IS NOW AN ASSERTION, and it took the single-edge re-lay to notice.** `assert-marathon.py` now pins the `(clusters, parts, networks)` tuple every leg's probe was taken over -- `(3, 24, 3)` for the three permanent rigs, and `(2, 20, 2)` for leg F, whose probe deliberately fires between the dissolve and the rebuild -- against constants written down in the script rather than read off the guest, and it requires each leg's audits to collapse to exactly ONE tuple. It exists because the injected defect meant as this suite's red proof **passed every assertion the suite had**: with leg F's rebuild crippled so that only half the churn rig came back, the item count never rose, nothing drifted, no cluster read `unbuilt` (a cluster with inputs and no outputs is a legitimate half-built state, so it is not "unbuilt" -- it is half-built), and the calibration spread stayed at **0.0%**. The only visible trace was `calZ` re-classifying a world with two fewer parts and one fewer network than `cal` did, and nothing was looking. It fails by name now: *"leg calZ audited a world of (3, 22, 2) and the rigs build (3, 24, 3)"*.

### `edge` -- `test/assert-edge.py`, the edits that land mid-operation

M2 proves item conservation across ONE recompile of a saturated 2x2. This suite asks what only a long multiplayer game asks. Every source is a **finite** steel chest and `count_all` totals every item on the visible surface AND the hidden one — ground, belts, splitter transport lines, chests — so the total is a conserved quantity and any fall in it is a real loss. Every count is taken in the same tick as a `bbb-audit` marker, which `create_entity{raise_built=true}` dispatches before it returns, so the count and the teardown it describes are one sample.

**Since 2026-08-02 every count line also carries `ground=`**, and that column is the one the policy pass added: conservation was never the defect, placement was. Re-recorded 2026-08-24 on Factorio 2.1.14 with every rig rebuilt to the one-belt rule; the baseline is **fifteen clusters over one hundred and ninety-eight parts** — the original seven, three saturated 4×4s over eight parts each that exist to take an edge edit while they run (two of them carrying a fifth ROW that holds nothing), `ntch`, the first field report's own shape (a C of five parts around a hole, saturated end to end and never edited; see "The tan streak"), `bmin`, the second's (a saturated dead-ended 2→2 over four parts plus an attached edgeless fifth, which crosses a power-of-two port boundary in both directions), `lim`, a 2×32 block carrying **sixty-four input belts** with an output part below it and a spare above — P = `plan.MaxPorts` exactly, 1,026 hidden entities, the biggest network this mod builds, sixty-six parts — and `brdg`, TWO balancers of thirty-three parts each (thirty-two inputs and one output apiece, P = 32) with one tile between them, which is the merge shape "The sixty-fifth belt" left uncovered. Neither of the last two is about items.

**THE SPARE PARTS ARE THE ONE-BELT RULE SHOWING THROUGH AND ARE NOT PADDING**, and they are what makes four of these rigs a redesign rather than a re-lay. `aout`, `ain`, `bmin` and `lim` all exist to take a belt ON A WORKING BALANCER, and under the rule a working balancer has no free face: every part that carries a belt has its one belt. An attached part with nothing against it is the only place a player's belt can still change a machine's port count — which is the conclusion m2's conservation rig and the interactive checklist's band B reached independently, from the other two sides. `brdg`'s gap tile is flanked by ONE belt rather than two for the mirror reason, and `frepa`'s belt line ENDS on the tile the part is dropped onto, because a part dropped mid-line takes the belt behind it as an input and the belt ahead as an output.

**And the baseline did not move when fast replace added two more rigs**, which is worth a sentence because every rig before them cost the whole table a re-record. `frepa` and `frepb` are BUILT MID-RUN, after the last of the assertions above has been made — only their source chests exist from `on_init`, so the conserved total never rises — and every count, every audit and every placement sample before them is the number it has always been. Anything added here later should do the same unless it has a reason not to.

**EVERY TAGGED AUDIT NOW ASSERTS `nets` BESIDE `clusters` AND `parts`**, against a constant derived from the rigs. `unbuilt=0` is weak evidence on its own: a cluster with inputs and no outputs is a legitimate half-built state and is never counted, so a save in which a rig quietly lost its network satisfies it. The one place `nets` is legitimately short is the ninth step of the by-hand teardown, where one part is left standing and one part carries one belt — so the survivor has an input or an output and never both.

| what is done to a full network | what came out |
|---|---|
| **100 add-part / remove-part cycles** on a balancer whose outputs are DEAD-ENDED, so its hidden network is full and stays full | 200 teardowns, median **72 items handed back each**, **0 lost**, **0 on the ground**, the count never rose, `drift=0 unbuilt=0` on all 100 audits, fifteen clusters of one hundred and ninety-eight parts back every cycle |
| a part placed and removed **inside one dispatch chain** | no change; the flush drops a root whose slot is already on the free list |
| a part placed on the tick a **deferred flush from the previous tick is pending** | one cluster of six parts, drift=0 |
| a part **bridging two saturated clusters** — two full networks down in one flush, one up | 15 clusters → 14, conserved, **0 on the ground** |
| **the undo of that merge** — the split path, still under load | 14 → 15, conserved, **0 on the ground** |
| an edge belt turned around **silently** (`entity.direction = …`, which raises nothing), twice | the audit **reports `drift=1`** before repairing, both times |
| the same edge through the **event path** | `drift=0` — the recompile already happened |
| **two forces editing in one tick**, alternating — A WHOLE ROW EACH, because a row is two parts and one part cannot take both the new input and the new output | still two clusters over the touching parts |
| **`game.merge_forces` while both networks are full** | 14 clusters over 202 parts, **6 parts remapped**, `drift=0`, both halves still delivering, **0 on the ground** |
| a part built **on the hidden surface** | refused |
| **two identical scrambled pastes** of four clusters (1, 2, 3 and 4 ROWS, so two, four, six and eight parts, entities interleaved so the arrival order is 3, 1, 4, 2) | **0 builds inside the paste tick, 4 on the flush, order 3 1 4 2 — both times** |
| **an OUTPUT BELT added to a saturated 4×4 while it runs** — the field report. It lands on the SPARE ROW, which is the only tile of that rig with a free face | 4→5 over eight ports, **0 on the ground**, and the network it became delivers **300 300 300 300 300** over the next 500 ticks, **0.00% spread** |
| **an INPUT BELT added** to another saturated 4×4 while it runs, on its own spare row | 5→4 over eight ports, **0 on the ground** |
| **an OUTPUT BELT mined** off a third — the SHRINK THAT IS NOT ONE | 4→3, **0 on the ground**. `P = next_pow2(max(N, M))` is 4 either side, so the butterfly it rebuilds is the SAME SIZE and the capacity fallback is never reached. This row used to be quoted as evidence that a shrink's reinsertion fits with room to spare; it is not, and reading it that way is what let the second field report through |
| **an OUTPUT BELT laid against a saturated 2→2's attached EDGELESS part and then MINED AGAIN** — the second field report, and the shrink that really shrinks | P 2 → 4 on the placement, **0 on the ground**; P 4 → 2 on the removal, **128 items spilled and 124 of them on the ground**. The machine halves, the reinsertion overflows, and until 2026-08-02 nobody was credited because the miner of a BELT recorded no claim. The assertion is a FLOOR (40): headless has no player, so what is pinned is the quantity. The 128 is the same 128 the 2.0.77 rig produced, which is a coincidence of geometry worth noticing rather than a check. See "A mine beside a machine is a mine of that machine" |
| **every part of that rig mined** — the cluster DISSOLVES, which is a removal | **118 items spilled back to the world**, 90 of them on the ground. This is the other half of the policy and it is asserted in the same suite so the two cannot drift apart. It reaches the dissolve by a `script_raised_destroy`, so it is also the **fallback** assertion for the miner's pocket: no player, no beneficiary, today's spill |
| **a saturated TEN-PART rig taken apart ONE PART PER TICK — the way a PLAYER does it**, nine shrinks and a dissolve | **110 items on the ground from the nine shrinks and 0 from the dissolve**, out of 168 spilled. The order is the spare row and then row by row, west part then east part: every prefix leaves a CONNECTED cluster, and eight of the nine shrinks leave a machine with at least one input and one output, so P really does come down 4 → 2 → 1 rather than the machine simply dying. This is the 2026-08-02 field report's own gesture and the quantity the miner's pocket redirects; the assertion is a FLOOR, so a leg where every shrink happened to fit fails rather than passing vacuously. With a player all 110 go to the miner. See "The shrink was the whole feature" |
| **the miner's-pocket INSERT, asked of a steel chest** — `insert` is a `LuaControl` member and a chest is a `LuaControl`, so the pocket's own call needs no player | four legs through the same `insertOne` from inside the same deferred flush: **50/50/50, 37/37/37, 23/23/23, 7/7/7** (asked/took/held), each cross-checked from Lua. A count arriving as **1** — the signature of an `ItemStackDefinition` whose `count` never reached the engine — fails by name |
| **the operator seam** -- the console command and the remote interface | `/bbb-audit` is in Factorio's own `commands.commands`; the interface exposes exactly `audit`; `remote.call` returns **14 clusters**, equal to the count of the audit that same call ran. Red-proven: renaming the command fails the suite |
| **a ONE-BELT-PER-PART refusal anywhere in the run** | **0.** Every rig here is laid so that no tile ever carries two belts and every edit is aimed at a spare part, at an edge the rig already has, or at a tile with nothing on it. A refusal for the other bound would mean a rig or an edit had quietly stopped being the thing it is named for, and every count above would go on passing. The three ways of REACHING that refusal are the `sedge` suite's |
| **the two walls that stop the pocket's TRIGGER being tested here**, probed in the same tick | `script.raise_event(on_player_mined_entity)` **refused** — *"can't be raised through script"* — and `game.get_player(1)` nil over `players=0`. Both asserted, so the day either falls the suite says to write the real test |
| **a pocket anywhere in the run** | **0**, and `m3` asserts the same over its own dozen removal paths. Every removal in either suite is a script destroy, a death, a robot, a merge or a surface event, so the beneficiary must never be consulted |
| **every entity of ours standing on the VISIBLE surface**, sampled seven times including once with every rig saturated and once with a refused merge standing | **180–197 of them, and every single one on a registered part tile — 0 off one, every sample.** The structural half of the tan-streak fix, and the `ntch` rig delivering **376 376** over the same window is what says it was measured on a balancer that was actually running |
| **A SIXTY-FIFTH INPUT BELT laid on a balancer already at `plan.MaxPorts`** — `lim`, 64 in and 1 out over sixty-six parts, running. It goes against the SPARE PART: every other part of that rig has its one belt, so a sixty-fifth belt laid anywhere else would reach the one-belt bound instead and this leg would be measuring the other refusal | the compile is **refused before the teardown**: **0 items on the ground**, the conserved total unmoved, the standing network delivering **184 items over 246 ticks before the edit and 185 after**, and the audit reporting `drift=1 unbuilt=0 refused=1` — a cluster that still HAS its network and knows its edge list has moved past what the mod can build. **Exactly one** `alert:` for the edit, not one per audit |
| **and that belt mined off again** | the edge list is back to sixty-four, which is the fingerprint the `netInfo` already holds, so the compile is a **SKIP**: `drift=0`, and nothing was rebuilt because nothing was ever torn down |
| **A PART BRIDGING TWO WORKING BALANCERS into one that is over the limit** — `brdg`, 32+32 inputs plus the gap tile's ONE flanking belt, so 65 over 2. One flanking belt and not two: a second would have to stand on the bridging part's own east face, taking that tile to two belts as well, and compile() asks the port bound first and returns — so the refusal would still read as a port refusal over a shape that is illegal twice, and the input count in the assertion is what says so | the merge is refused with **0 teardowns, 0 builds and 0 spills in the dispatch**: **0 items on the ground**, the conserved total unmoved, and both halves still delivering **184 and 184 items over 246 ticks against 186 and 185 before**. The audit reports `clusters=11 nets=12 drift=1 unbuilt=0 refused=1` — every standing network COUNTED, including the two now keyed by roots that are not roots — and says the same thing at all four samples while the refusal stands. **Exactly one** `alert:` |
| **and that part mined off again** | both halves' fingerprints are the ones their `netInfo`s never lost, so both compiles are a **SKIP**: `drift=0`, **0 teardowns and 0 builds**, and nothing reached the ground on the way back either |
| **A PART FAST-REPLACED ONTO THE LAST TILE OF A BELT LINE** running past a saturated balancer — `frepa`. The line ENDS there, because a part dropped mid-line would take the belt behind it as an input and the belt ahead as an output | the belt is gone, the part is registered, the cluster is 4 → 5 parts, and the **3→2 network it became delivers 262 262 over 350 ticks, 0.00% spread**. The belt and its eight items are handed back to the ground (no player), and nothing else moves |
| **A BELT FAST-REPLACED ONTO THE MIDDLE OF A NECK** — `frepb`, and the half the engine tells nobody about. The rig is a 1→1 row, a three-part neck carrying nothing, and another 1→1 row: the belt becomes an edge of the part above it AND of the part below it, so both of those have to be otherwise edgeless or one half would be refused instead of built | the nine-part column SPLITS: 14 clusters / 196 parts → **15 / 195, drift=0 unbuilt=0 refused=0**, the guest logging its own removal **exactly once**, the part handed back onto the belt that replaced it, and both halves still delivering **[262, 262] → [132, 264]**, 76% of the column. Without `fastreplace.go` it is **14 / 196** — a tile the registry calls a part with a player's belt standing on it |
| **and a belt over a part that CARRIES an interface** | refused: `can_fast_replace` is false, because `bbb-linked-belt` is a belt-connectable on that same tile |

**Zero spills over the whole run except THREE WINDOWS, and the suite fails on a fourth.** The windows are the dissolve (1 spill, 118 items), the by-hand teardown's shrinks (4 spills, 168 items) and `bmin`'s port-boundary removal (1 spill, 128 items) — all three of them removals a player would be credited for, and all three counted rather than treated as leaks. 230 recompiles reinserted something, median 72 items, and any recompile outside those three windows that reaches the spill path fails the run.

The scrambled-paste row is the determinism claim the deferred queue has to make. Events arrive in engine order, which is deterministic; what the queue adds is that it drains in the order roots were first inserted, dedupes without reordering, and re-resolves each root through `find` before compiling. Two identical pastes producing two different orders would be a desync, not a cosmetic difference.

The dead-ended outputs on the churn rig are load-bearing and the suite asserts that they worked: it requires a median teardown to have returned at least ten items, so a leg of empty teardowns fails rather than passing vacuously. Since 2026-08-02 it requires the same of the REINSERTION, so a policy that quietly stopped putting anything back would fail the same way.

### `mix` -- `test/assert-mix.py`, more than one kind of item

**Every other suite ran iron plates through everything.** That is the right default for a throughput number, and it left the whole multi-kind half of `guest/go/carry.go` -- the pool's (name, quality, stack size) key, the per-kind split, `insertRemainder`'s walk over several groups, and the BOUND on how many groups one pool may carry -- having never once run with more than one group in hand. **This suite is base only, and that bounds what it can reach**: below the stacking gate `drain` takes the flat totals and `compile.go`'s `detailedTally` is never called, so no rig here can exercise `kindAt`, however many kinds it runs. Multi-kind AND STACKED is Space Age and lives in the `plat` suite's `smix` band ("Stacked sushi"). Base only, five bands on a flat scratch surface plus a `ctrl` express belt, 3,220 ticks. **Every count is per item NAME**, on both surfaces, inside one tick: a teardown that dropped one kind and reinserted the rest conserves nothing, and a single total would have to lose the same number of items twice in opposite directions to hide it. Re-measured 2026-08-24 with the rigs REBUILT SINGLE-EDGE, Factorio 2.1.14; one saturated express belt delivered **1,306 items** over the window:

| rig | what it is | what came out |
|---|---|---|
| `duo` | a 2->2 fed by two PURE belts, one iron and one copper, draining freely | **1306 1306**, 2.000x one belt, spread 0.00%. Conservation across a forced recompile **exact per name** over 48 names and 20,076 items, **0 on the ground** |
| `quad` | the same one size up: a 4x4 fed by two iron and two copper belts, ALTERNATING | **1306 1304 1306 1304**, 3.997x one belt, spread 0.15% |
| `mixfull` | a 2->2 fed by two SUSHI belts, outputs DEAD-ENDED so it stays full | conservation **exact per name**, 6,616 items, **0 on the ground** |
| `many` | a dead-ended 4x4 fed by four sushi belts covering **48 distinct base items** | **7,936 items in, 7,936 out, every name exact**, 18 of them on the ground, and the guest's own overflow alert fires: **64 items past the 32-group bound put on the world** |
| `probe` | one infinity chest with SIX filters and no balancer at all | a measurement, not an assertion -- see below |

**The geometry doubled and not one contract moved.** Every column of parts is two under Factorio 2.1's rule, plus one EDGELESS part below each west column -- because a working balancer has no free face any more and the belt each conservation check lays on one would be REFUSED rather than compiled, which is the same conclusion `m2`'s conservation rig and the interactive checklist's band B reached independently. The save is **four clusters over twenty-eight parts** and the final audit is asserted against those constants: `nets == clusters`, `drift=0 unbuilt=0 refused=0`. `unbuilt=0` alone would not do it -- a cluster with no inputs or no outputs is a legitimate half-built state and never counts as unbuilt, so a rig rebuilt one column wide in the old idiom would be refused, deliver nothing, and still read `unbuilt=0`.

**Two things the rigs measure rather than assume, and the first is why the suite is built the way it is.** The obvious sushi source is an infinity chest with six filters feeding a loader, and it **does not make a mixed belt**: the `probe` band is exactly that rig and it delivered **2,292 items in 1 of 6 kinds, electronic-circuit 100%**, because a loader draws from the first stack it finds and the chest tops that same stack straight back up. So a source here holds ONE filter at a time and rotates it every four ticks with `remove_unfiltered_items` on, which gives a banded belt -- what a real sushi bus looks like anyway -- and is deterministic, the band boundaries being a function of `game.tick` and nothing else. The `probe` band stays in the save so that this paragraph keeps being a measurement.

And **the butterfly balances COUNTS, not KINDS.** Nothing in `plan.Build` knows what an item is; a splitter divides its input side between its output side by position and a name never enters the arithmetic. **Under symmetric saturation it is a PERMUTATION**, measured at two sizes on 2026-08-24: `duo` splits exactly by count and gives out1 ALL the copper and out2 ALL the iron; `quad`, fed iron/copper/iron/copper, gives outputs 1 and 2 all the copper and 3 and 4 all the iron, at 1306 apiece. Every input saturated, every output draining freely, each output takes exactly its share by count and exactly one kind.

**That is not what this suite used to record, and the difference is a PORT ORDER rather than a regression.** Until 2026-08-24 `duo`'s window opened AFTER its conservation belt had already taken it from 2->2 to 3->2 over P=4 -- an asymmetric network with a dead-ended spare port and a loopback, where the flows genuinely have to cross -- and the multi-edge geometry put that belt FIRST in the edge list. That network mixed, **75/25**. Laid single-edge the same belt enters LAST, the same P=4 network delivers **100/0**, and both are exactly balanced by count. So the old per-output **floor at 15%** was never a statement about the balancer: it was a statement about one asymmetric network's port assignment, and it is **retired rather than re-tuned**. `duo`'s conservation edit moved to the END of the schedule, so what its window measures is the 2->2 its own description names.

**What replaces it is the check the floor was groping at.** "The two kinds are taking different paths through the network" is only a defect if it costs a kind its THROUGHPUT -- one kind backing up while the other flows. So every kind fed into a pure rig must come out at the rate it went in, summed over every output: `duo` **1.000 belt of each of two kinds**, `quad` **1.998 belts of each**, at a 2% tolerance. True of a permutation, false of a network that starves a kind, and silent about a mix nothing ever promised. `quad` exists because `duo` alone could not tell a two-line accident from a property.

**Anti-vacuity is the whole risk in this suite and there are three guards.** The 48 item names are checked against `prototypes.item` at init, so a rename fails the CREATE log with the name in it; the count of DISTINCT names is logged and the assertion script requires the 48 it was promised, because a shorter list might not overflow at all and would then pass every conservation check while proving nothing; and **the guest's own overflow alert must be present**, which is the only direct evidence that the past-the-bound path ran at all.

**Red-proven in the new geometry, which is the only thing that makes `many` evidence.** Reverting the `tally` fix -- `addOverflow` and the spill accounting replaced by the pre-2026-08-04 `return` -- and running the same rigs: `many` loses **16 item kinds and 64 items**, 7,936 in and 7,872 out, with **nothing at all on the ground** and no alert anywhere in the run, and the per-name table names every one of them (chemical-science-pack, coal, express-transport-belt, iron-gear-wheel, iron-stick, low-density-structure, military-science-pack, pipe-to-ground, production-science-pack, productivity-module, speed-module, stone and five more). **Three assertions fire** and they are three different statements: the kinds did not survive, nothing reached the ground on a rig that exists to overflow, and the guest never said so. The fixed guest puts exactly those 64 items on the world.

### `mig` -- `test/assert-mig.py`, a save that used to be somebody else's

The ninth suite, **seven legs and two name probes**, and **the only one whose two phases run under different mod sets** -- a mod is installed or uninstalled between `--create` and `--benchmark`. Its rigs, its numbers, its seventeen red proofs and the run against the real Belt Balancer 2 are in "Adopting a Belt Balancer 2 or 3 save", which is where the whole feature lives; what is worth knowing here is that `guest/go/obs/bb2data` is a DATA-STAGE-ONLY stand-in carrying the real mod's own name and version, so `script.active_mods` sees what it would really see -- and that it is staged under **all four** of `legacyIncumbents`' names, by copying the one package and rewriting its `info.json` at staging time.

**The legs cover TWO AXES and neither is a subset of the other**: WHICH MOD owns `balancer-part` (four incumbent names and a stranger who is none of them), and WHICH TRANSITION of `legacy.go`'s state machine the load makes. The second axis is the one that was empty: until 2026-08-20 only `Blocked -> Done` by removal was ever driven, so the `Done -> Blocked` recheck that `fk_on_configuration_changed` exists for had no test at all, and neither did the promise `legacyCheck` makes to a stranger in as many words. `readd` and `fgone` are those two. **Both axes survived the single-edge port untouched**, because the state machine knows nothing about belts.

**AND SINCE 2026-08-24 THE OUTCOME IS SPLIT, WHICH IS THE PORT.** This is the one suite whose rigs were deliberately NOT re-laid one belt per part: its world is the INCUMBENT'S, and Belt Balancer's idiom is a single column of parts with a belt on every free face, so re-laying them would have been re-laying the thing under test. They convert exactly as they always did and are then REFUSED, and what was added is the **`sok` band** -- the same balancer laid two columns wide, which is a shape one of their users could genuinely have -- which converts into a network that runs. Nine clusters over thirty-one parts, and one audit line carries the whole answer, identically in all five conversion legs:

    clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7

Measured 2026-08-24, Factorio 2.1.14, base plus the quality mod. One saturated express belt delivered **1,306 items** over t=1800..3540:

| rig | laid | what it is on 2.1 |
|---|---|---|
| `m4x4` | the incumbent's way, 4 parts | refused. **0 0 0 0** |
| `m3to5` | the incumbent's way, 5 parts | refused. **0 0 0 0 0** |
| **`sok2`** | **two columns, 4 parts** | **1306 1306 -- 2.000x one belt, spread 0.00%** |
| **`sok4`** | **two columns, 8 parts, P=4** | **1304 1306 1306 1304 -- 3.997x, spread 0.15%** |

**The zeros are asserted as zeros rather than as a loosened bound**, which is the discipline the rest of the estate kept: a refused cluster has no network at all, so an item in one of those chests would be a balancer that got built when the rule says it cannot be. And **nothing is torn down and nothing is spilled**, which is where this suite asserts the opposite of `mig21`'s: there the balancers were STANDING when the save opened, so the remnant had to come down and everything it held reached the ground; here the clusters are seconds old, `hadNet` is false, and there is no teardown for the refusal to be in front of. The items are where they always were -- on the player's own belts, which is what the copper witness measures from the other side, at **48 at every one of four samples**.

**The refusal SHAPE is a multiset**, `[2, 2, 2, 2, 2, 3, 4]` parts carrying more than one belt, and `m3to5` is the row that makes it a statement about a classification rather than about a constant: three inputs and five outputs over five parts means three of its parts carry two belts and two carry one. Nothing else in the world has a count that is neither zero nor its whole size.

**And the `added` leg's rebuild-from-world ADOPTS the two working clusters beside the seven it refuses** -- `9 clusters, 2 adopted, 7 rebuilt` -- which closes a gap [`agents/single-edge.md`](agents/single-edge.md)'s phase-2 section records against `mig21` and could not close there: neither committed fixture has a single-edge cluster in it.

**The one defect this pass found and did not fix is FIXED, 2026-08-24.** A converted-and-refused balancer used to be announced with the ORDINARY per-piece message -- the one that says the extra piece was left in place unconnected, when nobody placed anything -- and the migration summary with its GPS checklist arrived only in the `added` leg, where a rebuild-from-world happens to follow the conversion. **A legacy conversion is the THIRD PRODUCER of that summary now**, so every conversion shape speaks the checklist once, per force, and no leg speaks the per-piece copy at all: `told per cluster: 0` and one summary in all six. `EXPECT_SUMMARY` moved from a measurement of the defect to a statement of the rule, and `EXPECT_TOLDPIECE = 0` is the half with teeth. The whole pass -- including the second false sentence it found on the way, and the 2.0 grandfather arm it settled -- is [`agents/single-edge.md`](agents/single-edge.md)'s phase-8 section.

### `qual` -- `test/assert-qual.py`, a part at uncommon quality is a part

The tenth suite, base plus the official `quality` mod, and **every part in every rig is UNCOMMON**, because the defect class it exists for is invisible at normal: `find_entity` resolves a bare name as normal quality only, so a lookup that used it worked on every other suite's save and silently failed on a quality-rolled part. Four rigs drive the four fixed call sites -- `qblk` the sprite write, `qlone` and `qcol` the two directions of the fast-replace reap, `qlim` the over-limit refusal's delivery -- and the suite's numbers, its red proof and the one thing still behind the player wall are in "A part at uncommon quality is a part", where the whole pass lives. It is also the only place anything asserts that **an uncommon balancer balances at all**: 2.000x one belt at 0.00% spread.

**Three of its four rigs were already built to Factorio 2.1's one-belt-per-part rule and did not move**, which is worth knowing because it is the only suite in the repository that can say so: `qblk`'s west column carries its inputs and its east column its outputs, `qcol`'s two INTERIOR parts carry nothing (which is what made the fast replace legal in the first place), and `qlone` has no belts at all. Only `qlim` moved -- it was thirty-two parts with a belt on both sides of each -- and it is **sixty-six** now: one output part, a 2x32 input block, and an EDGELESS part for the sixty-fifth belt to land on, without which the gesture would ask the single-edge bound instead and stop being a test of `forceOfCluster`. The save is **four clusters over seventy-five parts** and every audit is asserted as an exact tuple.

### `sedge` -- `test/assert-sedge.py`, one belt per balancer part

The eleventh suite, **the only one built to Factorio 2.1's rule and the only one that runs on it**. Eight clusters over thirty-six parts on a flat scratch surface, 3,500 ticks, base only: four single-edge shapes measured against a bare express belt in the same save with their **port counts asserted as an exact multiset before any rate is read**, and the three ways an edit can ask a part for a second belt -- a script BUILD, a silent ROTATION that raises no event at all, and a MERGE whose teardowns belong to `AddPart` and are queued before the compiler sees the cluster they make. Every one is refused in front of its teardown with the standing network still delivering, the audit reads `drift=1 unbuilt=0` and never the reverse, exactly one refusal is issued per distinct edge state, and **zero pieces are handed back and zero items spilled over the whole run**. The rig table, the red proof (fifteen assertions, ten `[BBB] error:` lines and 64 items on the floor with the predicate disabled) and the two design deviations are in [`agents/single-edge.md`](agents/single-edge.md)'s implementation-status section, where the whole port lives.

### `mig21` -- `test/assert-mig21.py`, a Factorio 2.0 multi-edge save opened

**The twelfth suite, and the only one with no `--create` phase at all.** Its worlds were built by a Factorio 2.0.77 binary that no longer exists here and that a 2.1 binary cannot reproduce at any price, so they are committed under `test/fixtures-2.0/` and **the fixture IS phase one**. Two of them: the m2 save (21 rigs, 77 parts, 4 surfaces) and the edge save (15 clusters, 95 parts, `lim` at 64 belts over 32 parts, and a SECOND FORCE).

**What the load arrives into is not what the save held, and that is measured rather than assumed.** Opening one under 2.1.14 does not crash: the ENGINE silently deletes all but one belt-connectable per tile, with no log line of any kind, before any script of any mod runs -- 77 interfaces over 77 part tiles where about 140 were built -- and leaves the hidden networks fully intact. What the deleted interfaces held went with them, at most eight items each, and no mod can recover it.

| | m2 | edge |
|---|---|---|
| the heap | **declined**, asserted rather than assumed: a leg that adopted it would measure nothing | the same |
| the rebuild | 77 parts, 21 clusters, **0 adopted, 21 rebuilt** | 95 parts, 15 clusters, **0 adopted, 15 rebuilt** |
| what the engine had already done | **0 stacked tiles**, 652 hidden entities standing | **0 stacked**, 1,898 standing |
| seeded into those networks before the migration | 2,320 items | 6,540 |
| recovered by the teardowns | **2,320, exactly** | **6,540, exactly** |
| spilled | **2,320** -- a refused compile claims nothing back | **6,540** |
| put back inside a network | **0** | **0** |
| on the ground afterwards, stable over 300 ticks | 1,006 (the rest landed on the player's own belts) | 5,645 |
| the compiler's entities afterwards | **0 visible, 0 hidden**; the player's 77 parts untouched | **0 and 0**; 95 parts untouched |
| the summary | **one** `force.print`, naming 21 balancers | **two**, force 1 about 14 and force 4 about 1 |
| the audit, twice and identical | `nets=0 drift=0 unbuilt=0 refused=21` | `nets=0 drift=0 unbuilt=0 refused=15` |

**THE FIXTURES ARE TICK-0 SAVES, SO THE NETWORKS WERE EMPTY, AND THAT WOULD HAVE MADE EVERY ITEM NUMBER A VACUOUS ZERO.** A `--create` never reaches a tick. So the observer SEEDS -- one item into every transport line of everything the compiler placed, in its own `on_configuration_changed`, which is the only "before" any script can reach because the migration runs from `fk_migrate` before tick 0. That is better than a stand-in for a running balancer's contents, because it is a KNOWN NUMBER: what the teardown recovered is asserted as an equality against it rather than as a floor. **And it cannot pass vacuously** -- the observer runs first because it sorts before this mod and deliberately declares no dependency on it, and if that order ever flipped there would be nothing to seed, the count would be zero, and the suite fails on a zero.

**Two refusal lines per cluster is the designed shape**, not a wart: the rebuild refuses with the worst information a refusal will ever have and is forbidden to speak, so it logs and re-queues, and the informed flush a tick later refuses again and delivers the one message.

**Both arms' ping lists CHART what they point at**, and the suite carries the tripwire that says why that cannot be checked directly: `is_chunk_charted` reads false for everything on a headless run, nauvis's origin chunk included, because a force with no players has no chart. What is asserted is the guest's own `charted N` against the ping count, and zero-before-zero-after on the engine's side.

**And the negative is the half THAT engine exists to pin**: the grandfather write must never be attempted where the settings key does not exist, so the 2.1 arm fails on any grandfather line, any failed-write alert and any setting-changed line at all. Red-proven three times, each catching something different -- the condemnation disabled (eleven assertions, 652 hidden entities left standing over 2,320 stranded items, audit `nets=21 drift=21`), the summary suppressed (exactly two, everything else unmoved), and the announce check removed (exactly one, a migration announced with the ordinary "the extra piece was left in place" sentence). Full tables in [`agents/single-edge.md`](agents/single-edge.md)'s phase-2 status section.

**ON FACTORIO 2.0 EVERY LINE OF THAT TABLE INVERTS, AND THE SUITE TAKES `--engine` FOR IT.** Nothing is pruned there -- that engine is the one that CAN stack, which is what multi-edge is -- so the m2 fixture arrives with **145 interfaces over 77 part tiles, 67 of them carrying two**, the standing networks match the edge lists re-derived from the world exactly, and all **21 clusters are ADOPTED with 0 rebuilt**. Nothing is condemned, torn down, spilled or refused; the ground total is **0** at every sample; and the grandfather pass writes this mod's own setting ON, re-queues every cluster (where each one skips on the fingerprint it never lost) and tells force 1 about 21 balancers with **21 pings**. The audit reads `clusters=21 parts=77 nets=21 drift=0 unbuilt=0 refused=0`, twice and identically, and the chests take **10,850 items over 300 ticks** -- because "adopted" has to mean the balancers still WORK, and a save that is merely frozen satisfies every other number here. The `edge` fixture is the same story over 15 clusters and two forces (14 pings and 1). There is no default for `--engine`: the outcomes are opposite, so a script that guessed would be green for the wrong reason on one of them.

### `flip` -- `test/assert-flip.py`, the multi-edge setting turned on and off

**The fourteenth suite, and the only one that cannot run on Factorio 2.1**: `bbb-multi-edge-parts` is defined on 2.0.x and never on 2.1.x, so there is nothing there to flip and `test/run.sh` prints a SKIP rather than passing -- with the assertion script failing on a setting that reads `absent`, which is this file's own "a check that skips is a check that passed" applied to a whole suite. What covers the FOLD on every engine is `go test ./edgemode/`; what nothing else covers is the WORLD it decides about.

Four rigs on a scratch surface: a control belt, `sok` (2 -> 2 over four parts, ONE BELT PER PART, legal in both modes and the control on every window), `me1` (2 -> 2 over two parts, the incumbent's idiom, draining freely) and `me2` (the same shape, DEAD-ENDED and built after the setting went on -- the field report's own rig, full and static). Four transitions:

| | measured |
|---|---|
| the false default | `me1` refused, `sok` built. Audit `(2, 6, 1, 0, 0, 1)` |
| turned ON | the handler re-queues, `me1` compiles, and a multi-edge balancer BUILT while it is on compiles straight away. `(3, 8, 3, 0, 0, 0)` |
| turned OFF with both standing and `me2` full | **VETOED**: the setting goes straight back on, one message per force with a ping per balancer, and **ground 0 -> 0 -> 0, items inside the networks 120 -> 120 -> 120, 12 interfaces and 21 hidden entities unmoved**. `sok` and `me1` both deliver **2.000x one belt at 0.00% spread over a window spanning the flip** |
| turned OFF once the multi-edge rigs are gone | it STICKS: the setting stays false, and a second belt on a working part is refused again. `(1, 4, 1, 1, 0, 1)` |

**THE FLIP-OFF IS A VETO AND THE SWEEP THE DESIGN DESCRIBED IS UNREACHABLE**, which is the finding: reaching `edgemode.ActSweep` means the capability marker is present, and the very next thing `settleEdgeMode` asks is `GrandfatherNeeded(marker, Off, n)` -- which is `n > 0`, the same condition under which a sweep finds anything. **Red-proven** with the destructive sweep put back: six assertions, `0 -> 88 -> 64` items on the ground, the networks emptied 120 -> 24, and eight interfaces torn down and rebuilt. See [`agents/single-edge.md`](agents/single-edge.md)'s phase-9 section, which is where the whole 2.0 arm lives.

**And `settings.global` is writable only by the mod that DEFINED the setting** (measured: *"Settings can only be changed by the owning player or the mod that made the setting"*, and a runtime-global has no owning player), so the suite drives the flip through `remote.call('better-belt-balancer', 'set-multi-edge-parts', ...)` -- a second method beside `audit`, reaching the same `writeMultiEdgeSetting` a player's keypress does and inert on 2.1 where the write is gated on the absent marker. Same argument as `bbb-audit`'s: a path only a human can reach is a path whose bugs a player finds.

### `iact` -- `test/assert-interactive.py`, the interactive checklist's own world

The thirteenth suite, and **the only one that is not about this mod's behaviour at all**: it is about the world `guest/go/obs/iact` stages, which is what a human enables before walking the checklist ([`test/interactive/README.md`](test/interactive/README.md)) and where the mod portal's demo scenes live. It stages the SAME PACKAGE `make interactive-install` installs, out of `dist/obs`, which is the point: a second copy anywhere would be a second world that could drift from the one the checklist describes. Nothing headless can make the gestures, which is the whole reason the checklist exists; but everything that mod STAGES is ordinary world-building, and a rig that stopped landing, or one this mod refuses, costs a human a whole session to discover and costs a single `--create` to catch.

**Twelve balancers over 228 parts** on nauvis: five gesture bands in a column at x = 20 and five demo scenes at x = 56, all of them single-edge since 2026-08-24. What it asserts:

- **every piece landed.** The staging mod routes every `create_entity` through one helper that logs when the engine returns nil.
- **nothing was refused.** Not one over-limit line and not one one-belt-per-part line. **The gestures are what create the refusals** and the staging must not: a refused rig is one a player finds already stopped.
- **every rig compiled to the SHAPE its geometry intended**, as an exact multiset over the create log -- `1->1/P1, 1->3/P4, 2->2/P2, 2->2/P2, 4->4/P4, 8->8/P8, 8->8/P8, 8->8/P8, 8->9/P16, 32->1/P32, 32->1/P32, 64->1/P64`. A rig whose belts landed elsewhere still compiles to something plausible; only the shape says which.
- **both audits.** `(12, 228, 0, 0, 12, 0)` and then `(12, 228, 12, 0, 0, 0)`: an audit reports the registry as its own dispatch finds it and that dispatch is also what drains the queue, so the first marker sees every cluster unbuilt and the second, placed behind it, sees them built. A `--create` never reaches a tick, so there is no third way to look.

**One `--create` and no benchmark**, because the whole question is answered by what the guest logged at load. That makes it the estate's ONLY create-only suite, which is worth knowing before anything is added to it: nothing is ever loaded, so no state checksum is taken, no benchmark milliseconds are printed, and the guest heap the observer writes into the save is never read back. **Red-proven** by staging a second belt against an occupied part: three assertions fire, one per family (a one-belt-per-part refusal, a compiled multiset missing that rig, and the second audit at `nets=11 refused=1`) -- re-run against the ported observer 2026-08-25 and unchanged.

**And the placement check has teeth, which the estate port had to prove separately**, because `put` is the one thing in the observer that is deliberately NOT `harness.Place`: a placement that did not land is reported by this observer's own `[BBB-INTERACTIVE] could not place <name> at (x,y)` line rather than by the harness's Fatal, so the assertion script names the piece and the tile. Give the observer a loader name its own DATA STAGE does not define -- the port's own new hazard, and what `obs/protos` exists to prevent -- and 97 placements fail by name. Two things that will NOT provoke it, measured on the way: `create_entity` places straight through a collision (a chest dropped on an occupied belt tile lands happily), and a duplicate part on an occupied tile is created too, with the registry deduping it by tile and no count moving.

**And the geometry behind every gesture is measured rather than asserted**, by a throwaway probe that drove each one by script: the numbers are in [`agents/single-edge.md`](agents/single-edge.md)'s phase-3 status section, including the two that matter most -- band B's shrink drains 200 items and **128 will not fit**, the same boundary crossing the `edge` suite's `bmin` leg records, and band A's eight-part teardown spills 18 items on each of three shrinks and 178 on the dissolve. Both are quantities a player's pocket redirects, so a rig where everything happened to fit would be a checklist that proves nothing.

### The layout check is gone

`make mod` used to run `test/check-layout.py`, which re-derived from the `fk_api_gen.lua` that `fklua mod` had just emitted **exactly the constants the guest derived by hand**, and failed the build when the two disagreed. The file is **deleted**, along with its Makefile hook, because that list is now empty. It went 25 event offsets plus two nested layouts → six offsets plus `host.go`'s four `create_surface` wire constants → two offsets plus four `defines.direction` values → **nothing**:

| what left | when |
|---|---|
| 19 event offsets and two nested layouts | `gen-bindings` began emitting a struct and a `Read<Event>(ptr)` per event |
| the four build events' offsets | a dictionary field inside a struct started generating, so all 218 events had readers |
| `host.go`'s four `create_surface` constants | `M.call` learned to trim an absent trailing optional; the file is deleted |
| `on_undo_applied` / `on_redo_applied` → `player_index` | `fk.subscribe` gained a **field mask**, so the expensive field is not marshalled and the generated reader is simply used ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 14) |
| the four `defines.direction` values | `gen-bindings` began emitting a `DefinesDirection*()` accessor per define path ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 11) |

**Deleting it is the right call and not merely a tidy-up, because the last two rows were the ones it was worst at.** The offsets it could check honestly: it re-derived them from the generated table, which is the authority. The four directions it could not — it compared `plan.go`'s literals against the `order` field of the pinned `runtime-api.json`, and a define's `order` is *not* its value; it is a sort key that happens to coincide today. That check was the same guess, marked against itself, and it would have passed on a Factorio where the numbers had moved. The accessor asks the running game by name, so there is nothing left to check and nothing left that could be checked well.

Everything the guest reads — every event field, every nested `BoundingBox` and `TilePosition`, `on_brush_cloned`'s tile array and its stride, and now every define — comes from generated code and moves with the pin by construction.

## A recompile is not a removal — where a teardown's items go

**The defect interactive play found that seven headless suites did not.** Every edge edit on an operating balancer — adding one output belt — emptied the hidden network onto the ground beside it. Every suite passed while it did, because every suite asked the wrong question: *were the items conserved?* They always were. The question nobody had asked is *where did they end up*, and the answer was "on the floor", on every recompile, forever.

**The policy, in one line: a recompile reinserts, a removal spills — and since 2026-08-02 a removal a PLAYER caused pockets first.** The implementation is `guest/go/carry.go` and its file header is the long form. The short form:

| what happened | what the drained items do |
|---|---|
| an **edge edit** on a standing cluster (a belt added, mined, rotated, retiered) | go back inside the network the same flush rebuilds |
| a **merge** — two clusters bridged | the survivor takes both halves' pools |
| a **split** — a cluster broken in two | both successors take even shares |
| a cluster **shrunk** (a part mined, parts remain) | the survivor takes them |
| a **clone reconcile**, a **force merge**, the **hidden surface** coming back | all recompiles; all reinsert |
| anything **MINED BY A PLAYER** that makes the machine smaller — a part, whether the cluster dissolves or merely shrinks, or a **belt at the cluster's edge**, which takes a port off it | whatever the successor network takes goes back inside it as always; **everything left over goes into that player's inventory**, and only the remainder of *that* to the ground — vanilla's rule for a mined machine. This row was narrower than the sentence above it twice: restricted to the dissolve it emptied a balancer onto the floor shrink by shrink, and restricted to PARTS it emptied one onto the floor when a player mined the output belt back off. Both were field reports on 2026-08-02. See "The miner's pocket" |
| a cluster **DISSOLVED** any other way, a surface **deleted**, a network **forgotten** | spill beside the cluster, exactly as before — nobody to credit |
| what a **materially smaller** network cannot hold | spills; this is the only way a recompile can still put anything on the floor |

### The four decisions, and why each is the one it is

1. **Which network inherits a pool.** The pool remembers the surface, the force and the visible bounding box of the cluster it came out of, and a cluster built in the same flush claims it when all three agree — boxes *overlapping* is the test. That single predicate covers all three shapes above: a plain recompile matches itself, a merged cluster's box contains both old ones, and a split's successors are both inside the old one. The **force** check is not decoration (two forces' parts touching are two balancers whose boxes are adjacent by construction) and it is the one thing `on_forces_merged` has to patch: it tears the networks down *before* it remaps the registry, so `remapCarryForce` follows the merge into the pools already drained. Without it the surviving cluster failed to recognise the absorbed half's pool and 52 items went on the floor — measured, in exactly that shape. **It follows the merge into the CLAIMS too since 2026-08-02**, which was the same omission one level down: a claim carries a force since it became a one-tile `carry.Region`, a player mining a source-force part in the merge tick wrote one down under the force about to be destroyed, and the survivor's remapped pool then failed to match it — the pocket becoming the floor. Both are `carry.Region.FollowMerge` now, and the claim store moved into the pure package with the predicate so `make check` covers it. See "The miner's pocket".

2. **How a pool divides between several claimants** (the split): even shares per item kind, the earlier claimant taking the odd one — `ceil(remaining / claimants-left)`, which is exact for one and splits 24 as 12/12. The counting has to happen **before the first build**, which is why `claimCarry` is a separate pass over the flush's queue: the first successor cannot know to take half unless something has already found the second. It is a flood fill per queued cluster and **not one host call**, and it runs at all only when a teardown left something behind.

3. **Where in the new network they go.** In **plan order** — `plan.Build` emits input side to output side — filling each transport line before the next, and **interior lines (splitters, lane splitters, straight belts) before any linked belt**. The second half is the one that matters: every stage still in front of a reinserted item rebalances it, whereas an item put on an edge line reaches a player's belt without passing through the butterfly at all. Measured on the 4→5 network a recompile built under load: **300 300 300 300 300 over the next 500 ticks, 0.00% spread.** `entBuf` is parallel to `ops`, so the order needs no second look at the world and no sort.

4. **`insert_at`, not `insert_at_back`, and that is the difference between 32 items and all of them.** The back of a line is ONE position, so `insert_at_back` succeeds once per line per tick and then refuses until the belt has moved. The first cut of this file used it and recovered exactly 32 items into a network with 32 transport lines, dropping the other 40 of a 72-item drain straight through to the spill. This is the same property `agents/design.md` records as an incumbent sin — *"script `insert_at_back` cannot produce compressed belts"* — met from the other side. `insert_at(position)` places at a named point, `line_length` says how far that goes, and walking a line at the belt's own **0.25-tile item pitch** fills it the way the drain found it.

**The transaction, and why it is a counter.** Pools live for one flush: `flush()` opens one around teardown-then-build and settles what nobody claimed when it closes. The three reconcile paths that split their own flush in half — `reconcileArea`, `hiddenSurfaceGone`, `onForcesMerged` — open one around the whole sequence, and the nesting counter absorbs the inner `flush()`. **Every path that opens one must close it**: the items are in guest memory until it does, and a pool that outlived its dispatch would be items in no transport line, no inventory and no ground stack. `dropSurface` and `hiddenSurfaceGoing` deliberately do *not* open one — there is nothing left to rebuild onto — so they spill immediately, which is what they always did.

### What it costs, measured

**The recompile hitch went up and this is the trade.** Interleaved `old / new / old / new / old / new` in one session, `test/run.sh m2`, medians of three, each minus that run's own `idle tick pair, nothing pending` control (0.36 old / 0.48 new), shipped `--persist=packed --gc=collected`:

| forced teardown-and-rebuild of a SATURATED rig | before | **after** | |
|---|--:|--:|---|
| 4×4, full recompile | 7.69 ms | **11.55 ms** | 1.50× |
| 4×4, one input removed | 9.40 ms | **11.76 ms** | 1.25× |
| 8×8, full recompile | 14.38 ms | **25.72 ms** | 1.79× |
| 8×8, one input removed | 18.10 ms | **26.92 ms** | 1.49× |

**It is boundary-bound like everything else here and it is proportional to the items actually in flight.** A 4×4 recompile hands back ~80 items and an 8×8 ~230, and the reinsertion is about 1.4 host calls per item recovered plus two per transport line touched, at the same ~12.6 µs per call the rest of the compile pays. The line walk is **lazy** for exactly this reason: it asks an entity how many lines it has, and a line how long it is, only as far as the items reach, so a recompile that recovered twenty items touches two entities and pays for two. An empty or lightly loaded network pays almost nothing, and **steady state is still zero** — there is no `on_tick` handler and there must never be one.

An 8×8 saturated recompile is now over one engine tick. That is stated rather than hidden, and it is the right side to be wrong on: the alternative is several hundred items on the floor every time a player touches a belt near a big balancer.

**The heap, and the one thing this pass gave back.** Under `-gc=leaking` a transient is permanent, so the `mar` suite's per-operation slopes move. Measured 2026-08-02, same suite, same method as "The marathon save":

| one operation | before | after | |
|---|--:|--:|---|
| teardown-and-rebuild of a **2→2** | 681 B | **1,180 B** | 1.73× |
| teardown-and-rebuild of a **4×4** | 1,517 B | **3,736 B** | 2.46× |
| a balancer grown by a part, dissolved and rebuilt | 1,921 B | **1,712 B** | **0.89×** |
| a whole 4-part balancer in and out | 1,216 B | 1,216 B | — |
| a belt inside the gate, laid and picked up | 352 B | 352 B | — |
| a belt 18 tiles from anything | 32 B | 32 B | — |
| linear memory over the 680-operation run | 1.92 MiB | **3.92 MiB** | one more rung |

Every leg is still **linear** (second half against first, ×1.00–×1.07; the suite fails at ×1.35), which is the property that makes a 300-hour projection multiplication rather than a curve. Re-running the edit-rate model with the new 4×4 term gives **~34.9 KiB per player-hour against 21.9**, so the busy four-player 300-hour figure moves 25.7 MiB → ~41 MiB and the linear-memory rung 32 → 64 MiB. **In the shipped `--gc=collected` build none of it persists**: the same 680 operations end on 0.77 MiB of linear memory against 0.46 before, and the **live set — what a save actually carries — is 8,960 B against 8,736**. This is the defect class "The third decision" flipped the mode for, arriving on schedule.

**And one term went the other way.** `drain` reads a line's contents with `GetContentsInto` on a package-level buffer now instead of `GetContents`, which was `make([]ItemWithQualityCount, n)` per non-empty line. That mattered more once a recompile started reinserting — a network handed its items back is *full* the next time it comes down, so far more lines take that branch — and it is worth ~2.2 KB per 4×4 recompile: the 4×4 leg was **6,517 B** before the buffer and is 3,736 B after, and the whole-balancer leg came out *below* its pre-policy figure. It is the only allocation on that path BBB owns; the rest is generated-binding return values ("What is left, quantified").

### What is still lost, honestly

Fractional item positions, and whatever a splitter holds outside its transport lines. **The stack sizes are no longer on this list** — see the section below, which is the pass that took them off it.

One known imprecision, recorded rather than fixed: a cluster that claims a pool geometrically but is then **not built** (it has no adjacent belts at all, or its compile fails) leaves its share to spill at the end of the flush. It needs two same-force clusters with overlapping boxes recompiling in the same tick, one of them beltless; conservation is unaffected and the fallback is the correct one. **Since the beneficiary pass that share is offered to the miner before the floor**, if a player is what caused the dissolve, so the fallback got slightly better without the imprecision going away.

### More than thirty-two kinds — the bound that was an item sink

**A drained pool holds `maxCarryKinds` = 32 (name, quality, stack) groups, and the thirty-third was logged and DROPPED.** `drain()` had already read that group off a transport line and `sweep()` destroys the entity a few statements later, so those items ceased to exist — the one thing this mod is not allowed to do, in the one file whose header says so twice. Found by review on 2026-08-04, not by play, and then reproduced in a headless run before a line was changed.

**What a player would see, and why it is not exotic.** Thirty-three distinct groups through one balancer is a **sushi belt or a mall bus** — a mixed line feeding a balancer is one of the things people build balancers for. Under Space Age it is nearer eight item kinds than thirty-three, because a group is a (name, quality, stack size) and belt stacking multiplies the first by the last. The symptom is silent: the balancer keeps working, the counts nobody is taking are smaller than they were, and the only trace is a `[BBB] error:` line in a log nobody reads.

**Why eight suites were green while it happened**, and it is the same shape as the two field reports above: **all of them run iron plates.** One kind, one quality, no stacking, so the pool never held more than a handful of groups and the branch was unreachable from every rig in the repo. `mix` is the suite that can reach it, and it is red-proven against the guest that shipped: **16 kinds and 72 items destroyed** on one recompile of a saturated 4×4 carrying 48 kinds.

**The fix is that the overflow spills, and WHEN it spills is the part that was measured twice.** `tally` appends what it cannot carry to a small buffer and `closePool` puts it on the ground at the centre of the cluster's box — the same place, the same call and the same arithmetic as the `spillPool` every removal path has always used. The first cut spilled at the moment of the decision, inside `tally`, which is appealing because there is then nothing to remember; it lost **one stone-brick of 4,336**, because `teardownNet` sweeps the hidden half FIRST and the visible cluster box SECOND, `spill_item_stack` allows belts, and the visible sweep re-drained what had just been put in the middle of it into a pool already too full to take it. The control that proved it: raise the bound to 64 so that nothing overflows and the same rig conserves exactly. Buffering to the end of the teardown, after both sweeps, gives **4,336 in and 4,336 out**.

**What it gives up is placement, not conservation**, which is the doctrine `compile.go`'s `detailedTally` already states in as many words: an overflow group is not carried into the successor network and is not offered to a miner's pocket, because both of those are what the POOL is for and a buffer general enough to reach either would be the pool with no bound at all. Past thirty-two groups this guest stops promising WHERE a teardown's items land and goes on promising THAT they land. The bound itself stays: it bounds package-level memory that is never given back.

**The line is an `alert:`, and the level is load-bearing rather than a matter of taste.** `test/run.sh` fails any run in which a `[BBB] error:` line appears at all, because an error from this guest means one thing — a compile did not produce a network. An overflow is not that, and at `error:` the `mix` suite could not have asserted it: the runner would have killed the run before a single number was read. `alert:` is this guest's existing level for "something outside the ordinary happened and the mod coped", it is never switched off by `QUIET=1`, and the suite **requires** it in the overflow window.

**What it costs.** Nothing on any path base single-kind play reaches: the branch is past thirty-two groups, `carryOverflow` is empty on every other tick, and the common path gains one slice truncation in `openPool` and one `len()` test in `closePool`. Measured rather than asserted — the `mar` suite's seven per-operation heap slopes came back **identical to the byte** in the `-gc=leaking` arm (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B, 3.92 MiB of linear memory), and a pre-fix/post-fix run of the other six suites differs in **no asserted number at all** — only in profiler milliseconds and in which function `fklua mod`'s NaN report attributes a hoisted `f64` to. Package built 2026-08-04, shipped config:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 298,874 B | **300,422 B** | +0.52% |
| `fk_module.lua` | 2,282,383 B | **2,316,009 B** | +1.47% |
| members bound into the mod | 38 | **38** | of 4,257 |

## The miner's pocket — a player who mines a balancer keeps what was in it

**Mining a vanilla splitter puts what it was holding in your pocket.** Mining the last part of a balancer emptied a hidden network onto the ground beside it — conserved, placed correctly by the policy above (a dissolve *is* a removal, and a removal spills), and still not what the game does. This pass is the rest of that sentence: a removal a PLAYER caused offers its items to that player first, and only the remainder reaches the floor.

**The pool carries a `player_index` and nothing else.** A scalar, because the no-entity-references rule is absolute here and the gap is a whole tick wide: the mine arrives in one dispatch, the network is not drained until the deferred flush in the next, and a `LuaPlayer` handle does not survive that. `game.get_player` is called **fresh, at settle time**, and returning nothing is the ordinary case rather than an error — a player who left between the mine and the flush simply gets today's spill.

### Which removals get a beneficiary

| removal | beneficiary |
|---|---|
| `on_player_mined_entity` on **any part**, whether the removal dissolves the cluster or merely **shrinks** it | that player. See "The shrink was the whole feature" below — restricting this to the dissolve was the first cut and made the pocket almost inert in play |
| `on_player_mined_entity` on a **belt-connectable at a cluster's edge** — the belt beside the machine rather than a part of it | that player, **since 2026-08-02**. Removing an output takes a *port* off the machine, so this is a shrink like any other. See "A mine beside a machine is a mine of that machine" below — this row said *none* for one commit and a player found it |
| a **robot** deconstruction | none, deliberately, for a belt at the edge exactly as for a part. See below |
| `on_entity_died`, `script_raised_destroy`, a **surface** deleted or cleared, a network **forgotten** | none — nobody did those |

**The robot case is a decision, not an omission.** Vanilla sends a deconstruction robot's haul to a logistic storage chest, which needs either the robot itself — an entity reference the deferred flush cannot hold, by the rule this whole guest is built on — or a network-and-chest search the guest would have to do from scratch at settle time. Spilling is what every other non-player removal here already does and it is recoverable. Revisit only with the robot's own inventory in hand, which means doing the work **inside** the event, which means putting back the removal window `fk.Defer()` deleted.

### The shrink was the whole feature — the 2026-08-02 field report

**The pocket shipped crediting the miner only on the DISSOLVE, and a player reported that mining a full balancer in the map editor put ONE item in their inventory and the rest on the floor.** The report was accurate and the cause was not where anybody looked: not the boundary, not `insert`, not the count. It was the word *dissolves* in the table above.

**A player does not mine a balancer in one tick.** They mine a part, the machine recompiles **smaller** — fewer ports, a smaller butterfly, less transport line — they mine the next one, and so on down to the last. Every one of those steps is a SHRINK, every shrink hands back less than it drained, and the difference falls through to the spill by the fourth decision of "A recompile is not a removal". Only the final dissolve recorded a miner, and by then there was almost nothing left in the machine. Measured headlessly on a saturated four-part 4×4 with dead-ended outputs, mined one part per tick:

| step | what the guest did | cumulative on the ground |
|---|---|--:|
| mine part 0 | shrink: 232 drained, 224 reinserted, **8 spilled** | 7 |
| mine part 1 | shrink: 224 drained, 72 reinserted, **152 spilled** | 156 |
| mine part 2 | shrink: 72 drained, 26 reinserted, **46 spilled** | 200 |
| mine part 3 | **dissolve**: 26 drained — the only step the pocket ever saw | 224 |

**206 of 232 items were already on the floor before the beneficiary was consulted at all**, and a smaller or less loaded balancer leaves the dissolve holding a single item — which is exactly the report. **Editor mode is where it looks worst rather than where it is different**: the editor controller ships `mining_speed = 6` and `instant_deconstruction = true`, so a player rips the balancer apart fast enough that the whole machine appears on the ground at once.

**The fix is one call site**: `removePart` records the miner for every part a player mines, not only for the one that empties the cluster. Nothing else moves, and the precedence is what makes that safe — a claimed pool is never pocketed, so the survivor still claims the shrink's pool and `takeCarry` still reinserts everything that fits. The beneficiary is consulted by `settleCarry` over the **remainder alone**, so a shrink that fits entirely — the ordinary one, and every edge edit in the `edge` suite — reaches none of this and costs nothing.

**What it cost to find, and the lesson.** The first hypothesis was that the count on the `ItemStackDefinition` was not reaching the engine — `ItemStackDefinition.count` defaults to `1` when it is absent, so "exactly one item pocketed" is that defect's exact signature, and the ABI was the obvious suspect. It was wrong, and disproving it is what `probe.go` now is: a shipped diagnostic that asks a **chest** the question the pocket asks a player. That took the boundary off the table in one headless run and left the guest's own policy as the only place the defect could be. **The lesson is the one this file already records in another form: a path that no suite can reach is a path whose bugs a player finds.** The pocket was declared unverifiable headlessly and the declaration was half true — the trigger needs a player, the *arithmetic* and the *quantity* never did, and both are pinned now.

### A mine beside a machine is a mine of that machine — the second field report

**Place a downward-facing output belt on a running balancer, then mine it again: a small pile of items on the floor, with an inventory that had room for them.** Same reporter, same day, and the same shape as the shrink report one level out — the policy sentence already covered it and the implementation asked a narrower question.

**Nothing was lost and nothing was placed wrongly.** Removing an output takes a *port* off the machine, and `P = next_pow2(max(N, M))` is a step function: two outputs to three took P from 2 to 4 when the belt was laid, and mining it takes P back to 2. The butterfly halves, the network the recompile builds cannot hold what the one it drained was holding, and the difference falls through to the spill by decision 4 of "A recompile is not a removal". That part is correct and stays. What was wrong is where the difference went: `removePart` records a claim and `onNeighbour` did not, so a **belt** mined at the edge had no beneficiary and the overflow went to the floor rather than to the miner.

**Why seven suites were green while it happened, and it is not "no suite drove that edit".** The `edge` suite drives exactly that edit — `shrk`, an output belt mined off a saturated 4×4 — and CLAUDE.md quoted it as evidence for the old policy row: *"the `edge` suite's `shrk` leg measures that reinsertion fitting with room to spare"*. **The measurement was true and the generalisation was not.** `shrk` goes four outputs to three, and `next_pow2(max(4, 3))` is 4 exactly as `next_pow2(max(4, 4))` is: the machine it rebuilds is the *same size*, so of course everything fits. The suite's own bound (40 items) has never been approached because nothing was ever overflowing. **A shrink that does not shrink the butterfly is not evidence about shrinks**, and reading it as such is what let the report through.

**The fix is one call site and one tile.** `onNeighbour` records the miner too, and the claim goes on **the part tile the gate just looked up** — not on the mined belt's tile, which is one outside the network's box by construction and would answer to no pool at all. That is the trap the obvious fix falls into: the part path passes the tile the *event* reported, because for a part that tile is the network's, and reusing the same call on the neighbour path compiles, runs, records a claim per mine and changes nothing. Tile-keyed rather than root-keyed for the reason "The three decisions" already gives — `flushLive` re-resolves every queued root a tick later and a part mined elsewhere in the same tick re-roots the survivors — so the two paths agree by construction instead of by argument.

**It costs nothing on the guest's hottest path**, which is the one property that was never negotiable: the gate is entered for every belt anyone lays or picks up anywhere on the map, and the tile, the force (`pforce[id]`) and the player index are all already in hand. There is **no host call**, and the note is skipped outright unless a `player_index` arrived — which is every event but one. `carry.Claims.Add` also dedupes exact repeats now, because the gate calls it once per part tile in a 5×5 neighbourhood and a deconstruction planner dragged along a belt line beside a balancer would otherwise grow the store with the *sweep* instead of with the *machine*.

**The `bmin` rig, and it is a tripwire rather than a proof.** The `edge` suite gained the shape the report was made with: a saturated two-part balancer, two in and two out, dead-ended so it stays full, with a third output belt added while it runs and then mined again — P going 2 → 4 → 2, which is the boundary crossing `shrk` does not make. Measured 2026-08-02:

| | measured |
|---|---|
| the belt **laid** (P 2 → 4, the machine grows) | **0 items on the ground**, like every other edge edit |
| the belt **mined** (P 4 → 2, the machine halves) | **128 items on the ground** — the field report, reproduced |
| the same run on the guest **before** the fix | **128**, byte for byte |

That last row is the honest split and it is why the assertion is a **floor** (40) rather than an equality: `player_index` is 0 on every removal a headless run can produce, so the redirection is invisible here and only the *quantity* is not. The suite now has **three** spill windows and no more — the dissolve, the by-hand teardown's shrinks, and this — and a spill outside any of them still fails the run. What the fix does with those 128 items is checked by `go test ./carry/` for the identity, by the chest probe for the arithmetic, and by a player for the trigger.

### The three decisions

1. **Precedence: a claimed pool is never pocketed.** The four decisions of "A recompile is not a removal" run first and unchanged. If any cluster built in the same flush succeeds the network geometrically — a merge, a split, a shrink, a plain recompile — the items go **into it**, and the beneficiary is never consulted. `settleCarry` is the only caller, and it sits between the claim and the ground. So the ordering is: **network, then miner, then floor.**

2. **A claim is keyed by TILE, not by cluster root**, and that is the one thing the obvious implementation gets wrong. A decon planner mining a four-part balancer in a single tick removes three parts that merely *shrink* it — each re-rooting the survivors at the smallest surviving node id — and then a fourth that dissolves, so the root at the moment of the dissolve is **not** the root the `netInfo` is filed under, and a root-keyed claim would silently miss. The mined part's **tile** is inside the visible bounding box of the network coming down whichever root owns it, so a claim is `(surface, force, tile, player)` and `openPool` takes the first claim standing on a tile of the network it was handed. First in event order, which the engine makes deterministic, so two players mining into one dissolve resolve identically on every client. **The `force` term is the 2026-08-02 correction below and it was missing for two commits.**

   **And the tile is the NETWORK's, which is not the same as the mined entity's.** For a part the two readings coincide and the distinction never came up; for a **belt at the cluster's edge** they do not, because a belt adjacent to a cluster is by construction one tile *outside* its box — so the neighbour path passes the tile of the **part it was touching**, which is the registry key the gate just looked up. Handing over the tile the event reported would record a claim no pool can ever answer, silently, which is the whole of `carry/beside_test.go`.

3. **`LuaControl.insert`, plain counts, one host call per item kind.** The count may span many inventory stacks, so a 72-item pool of one kind is one call and it returns how many were actually taken. The **belt** stack size is dropped here on purpose: an inventory has no notion of one, and vanilla mining loses it too. "Stacked belts come back stacked" recovers stack density for the *reinsertion* path, which is the only path where it means anything.

**Why not `event.buffer`, which is how vanilla does it.** `on_player_mined_entity` carries a `LuaInventory` the engine then empties into the player, and it is the right answer for the entity being mined — which is the **part**, a `simple-entity-with-force` holding nothing at all. The items are in a hidden network on another surface, they are not read until the flush a tick later, and the buffer is valid only inside its own dispatch. Reaching it would mean draining inside the event, which is exactly the removal window `fk.Defer()` retired, for a cosmetic difference in where the items land.

### What it costs

One `uint32` on `carryPool`, one `(surface, force, tile, player)` entry per cluster a player shrank or dissolved in one tick — truncated by every settle, high-water like the recompile queues, and **bounded by the machine rather than by the gesture** since `Claims.Add` dedupes exact repeats (a belt line deconstructed beside a balancer walks the same handful of part tiles once per belt) — and **one bound member**, `LuaControl.insert`, taking the mod from 36 to 37 of 4,187. `game.get_player` was already bound for `on_undo_applied`. Nothing on any hot path: the claim list is empty on every tick nobody mined a balancer, `beneficiaryFor` is a scan of an empty slice, and the pocket itself costs one `get_player` plus one `insert` per item kind on the one dispatch where a machine was removed. Package built 2026-08-02, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 278,309 B | **281,920 B** | +1.30% |
| `fk_module.lua` | 2,112,476 B | **2,159,726 B** | +2.24% |
| members bound into the mod | 36 | **37** | of 4,187 |

**The 2026-08-02 correction added no runtime cost and one prototype.** The fix itself is a call site moved four lines up; what is not free is `probe.go`, which is diagnostic-and-test machinery that ships for the same reason `bbb-audit` does. Measured over the same round:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 281,920 B | **287,334 B** | +1.92% |
| `fk_module.lua` | 2,159,726 B | **2,222,176 B** | +2.89% |
| members bound into the mod | 37 | **38** | of 4,187 (`LuaEntity.type`) |
| shipped prototypes | 5 | **6** | `bbb-insert-probe`, hidden, script-only |

Nothing on any hot path moves: the probe's queue is empty on every tick nobody placed a marker, so `flush` gains one length test on a nil slice, and the entity filter gains a third name term that the engine decodes once at subscribe time.

### What is verified, and what the user has to check by hand

**Only the TRIGGER is interactive now.** The wall is real and unchanged — nothing headless can make this guest resolve a `LuaPlayer` — but the two things that were behind it with the trigger are not behind it any more, and the field report is why. Both were added 2026-08-02 and both live in the `edge` suite.

**The insert arithmetic, asked of a steel chest.** `insert` is a member of `LuaControl`, and a chest is a `LuaControl`, and so is a character: `LuaEntity.insert`, `LuaPlayer.insert` and `LuaControl.insert` are **one member id, one signature and one tier-2 encode of one table**. So the exact call the pocket makes to a player can be made to a chest with no player anywhere. The `bbb-insert-probe` marker (`guest/go/probe.go`, and a fifth shipped prototype alongside `bbb-audit`) is placed on a container, **deferred exactly as the pocket is** so the call happens inside `fk_on_deferred` rather than inside a build event, and offers it four legs through the very same `insertOne` the pocket uses:

    [BBB] insert probe container iron-gear-wheel asked=50 took=50 held=50
    [BBB] insert probe container iron-plate      asked=37 took=37 held=37
    [BBB] insert probe container copper-cable    asked=23 took=23 held=23
    [BBB] insert probe container steel-chest     asked=7  took=7  held=7

Four questions, one per leg, and the counts are distinct and none of them is 1 so that a wrong answer says which: a two-key `{name, count}` map; a three-key one with a **quality**, proving the optional third key does not displace the second; a two-key one **after** the three-key one, which is the shared-`carryKV`-buffer question; and one whose item name was **read out of the world** rather than written down in the guest, so it is a heap-allocated string out of `getStr` rather than a pointer into `.rodata`. The suite reads all four back from Lua as well, so the guest's own numbers are checked against something that did not cross the boundary. A count arriving as 1 — which is what an `ItemStackDefinition` whose `count` never reached the engine produces — fails the suite by name.

**The quantity, measured by taking a balancer apart the way a player does.** The `edge` suite's `hand` leg mines a saturated **ten-part** rig one part per tick and counts the ground at every step. Ten parts because a 4×4 is eight under the one-belt rule and this rig carries a spare row as well, and the order is the spare row and then row by row, west part then east part -- so every prefix leaves a CONNECTED cluster and eight of the nine shrinks leave a machine with at least one input and one output. On the shipped build:

    taking a saturated balancer apart ONE PART PER TICK:
      cumulative on the ground: pre-hand=0 ... hand-6=78 hand-7=85 hand-8=95
                                hand-9=110 hand-10=110
      the nine SHRINKS put 110 items there and the dissolve 0

Headless has no player, so those 110 items still land on the floor and every other number in the suite is what it was. What is pinned is that **the overflow is a real quantity**: the assertion is a floor rather than a ceiling, because a leg where every shrink happened to fit would satisfy every other check in the suite and would say nothing at all about the thing that was fixed. With a player, all 110 go to the miner before the ground. The dissolve gets nothing here and that is the rule showing through: the ninth step leaves ONE part, one part carries one belt, so the survivor has an input or an output and never both and `plan.Build` gives it no network at all to drain.

**...and the same quantity for the BELT at the edge**, which is the second field report's `bmin` leg. An output belt laid on a saturated two-part balancer and then mined again takes P from 2 to 4 and back:

    an OUTPUT BELT placed on a running balancer and then mined again:
      P went 2 -> 4 on the placement and 4 -> 2 on the removal
      items on the ground from the removal: 124 (floor 40), of 128 spilled

**0 on the placement, 128 on the removal, and 128 on the pre-fix guest too** — the fix is invisible to a headless run for the reason above, so what the leg is is a tripwire on the quantity and not on the redirection. It is also the leg the suite lacked: `shrk` mines an output belt as well, but four outputs to three leaves `next_pow2(max(N, M))` at 4, so nothing overflows and its bound has never been approached. See "A mine beside a machine is a mine of that machine".

**What is left, and it is only the trigger.** The `edge` suite carries a probe that runs in the same tick as the removal leg:

    [BBB-EDGE] player-mine-raise ok=false
        err=on_player_mined_entity (ID 76) (76) can't be raised through script.
    [BBB-EDGE] player-resolve p1=false players=0

Two walls, and the suite **asserts both**, so if either ever falls the run fails and says to write the real test instead of documenting it: a headless `--create` has no players, so `game.get_player(1)` is nil; and `on_player_mined_entity` is not one of the events `script.raise_event` will raise — `LuaBootstrap` carries a `raise_*` helper for each of the eleven that can be, and there is none for this one. This is the same shape as the undo/redo entry in the table below.

What the suites **do** pin:

- **the fallback**, which is the entire existing dissolve assertion: the `edge` suite's removal leg reaches the spill by a `script_raised_destroy`, records no beneficiary, and comes out at exactly the numbers it always did — **118 items spilled, 90 of them on the ground**, and every recompile still at `ground=0`. There are **three** spill windows in the suite now and no more: that removal, the `hand` leg's shrinks, and `bmin`'s port-boundary removal. A spill outside any of them fails the run;
- **the negative**, which is the half with teeth. `edge` and `m3` between them drive every removal path there is except a player mining — `die()`, `destroy()` with and without an event, a fast-replace, a shrink, a split, a force merge, a clone reconcile, a surface deleted, the hidden surface deleted, ~100 randomised stress teardowns and 200 churn teardowns — and both assert that **zero** teardowns credited a player. A claim leaking across a removal path it does not belong to fails the suite. **That assertion still holds after the shrink and then the neighbour path began recording claims**, and it is not a weaker statement: `player_index` is zero on every removal a headless run can produce, and `noteMinedByPlayer` returns on zero before it appends, so the suites' numbers did not move by a single item;
- **the log line**, `cluster N dissolved, mined by player P`, which is the only evidence a headless run could ever give that `player_index` reached the registry. `assert-log.py` counts dissolves on the word rather than on the end of the line so that the suffix cannot silently break a counter that has been green since M1. Its sibling, `cluster N offered X items to player P before the floor`, is written by `handBack` at the exact point the decision is taken and is what an interactive check greps for — it cannot fire in any suite, for the same reason;
- **the identity**, by `go test ./carry/`, which is the only machine in this repo that can check any of this. `carry/beside_test.go` is the neighbour case: that the claim goes on a tile of the NETWORK and that the mined belt's own tile answers to nobody, that one belt between two clusters credits both, that a force boundary is respected, and that a decon sweep does not grow the store.

**What the user tests interactively**, and it is still one thing: that a player mining is what reaches this code. Two gestures, one per field report. Mine a balancer that is carrying items **part by part** and check that the items arrive in the inventory rather than on the ground at *every* step, not only the last; and **lay a belt on the spare edgeless part attached to a running balancer and mine it again**, which is the gesture that halves the machine — those items must arrive in the inventory too. (Under the one-belt-per-part rule a working balancer has no free face for a belt at all, so the checklist's band B stages a spare part to receive one; measured on that rig, the mine drains 200 items and 128 will not fit.) Grep for `offered … before the floor` either way. A full inventory must still spill the remainder, and an edge edit that does not shrink the machine must still put everything back inside the network and pocket nothing.

### A claim is a Region — the force the claim did not carry

**The claim test compared the surface and the box and not the FORCE, while the successor test over the same pool — three hundred lines away in the same file — compared all three.** Found by review, not by play, and it is the third time this repo has met the same shape: a force check that every neighbour of a predicate has and that one predicate does not (see "Two bugs M3 found", where `collectCluster` was the one).

What it could do, and it is narrow but real. Clusters are per force, so **two forces' parts touching are two balancers whose bounding boxes are adjacent by construction** — and around an L or a diagonal they overlap outright. Two such networks coming down in one tick, with a player mining one of them, and the pool of the *other* force's network could find that claim inside its own box and credit that player. Conservation was never at risk — the pool is settled either way, into a successor or a pocket or the floor — and no suite could see it, both because no headless run has a player and because the wrong player is not a wrong count. **The wrong pocket is the whole feature being wrong.**

**The fix is not a comparison, it is deleting the second predicate.** A claim is the one-tile `carry.Region` it always was — surface, force, and a box with both corners on the mined tile — and `beneficiaryFor` and `carryPool.matches` are both `Region.Overlaps` now. There is no second place left to keep in step, and a third question about the same identity asks the same code or does not ask at all.

**The first failing test this repo could write for the miner's pocket.** The trigger needs a player and always will; the *predicate* needs nothing at all, so it moved to `guest/go/carry` — pure Go, no fkapi, the third package to earn that treatment after `plan` and `skin` — and `make check` runs it. Transcribing the shipped guest's two predicates into one function and running the new tests against it fails on three of them, and all three are the same defect seen from a different side:

    --- FAIL: TestAClaimOfAnotherForceIsNotThisNetworksClaim
        network of force 1 accepted a claim on force 2's part at (12,12):
        one force's miner would pocket the other's items
    --- FAIL: TestAClaimIsJustADegenerateRegion
    --- FAIL: TestOverlapsIsSymmetric
        Overlaps(1,2) is not symmetric

The last one is worth keeping for its own sake: a pool asks the predicate one way (*does this new cluster succeed me*) and a claim the other (*is this tile mine*), so **asymmetry is exactly what "written out twice" looks like** from the outside, and it is a property a future third caller would break again.

**It costs nothing.** The force is `pforce[id]` at the mine event — registry state already in hand at the call site, read before the node is freed — so `noteMinedByPlayer` still makes no host call and cannot fail; the claim carries one more `uint32` on a slice that is empty on every tick nobody mined a balancer; and `Region` is a struct of scalars that TinyGo inlines away. Measured over the round, shipped config: the zip went **287,334 → 287,422 B (+88)** and `fk_module.lua` **2,222,176 → 2,224,002 B (+1,826)**, and no member, define or prototype was added. All seven suites are green and **their numbers did not move by a single item**, which is the expected result rather than a weak one: `player_index` is zero on every removal a headless run can produce, so `beneficiaryFor` scans an empty slice in every one of them.

### ...and a force in the key is a force that can be DESTROYED

**The same round, one level down: `remapCarryForce` followed the merge into the pools and not into the claims.** A claim carries a force index only since the section above made it a one-tile `Region`, and `game.merge_forces` is the one event that can destroy the force something wrote down. `onForcesMerged` tears the affected networks down *before* it remaps the registry — it has to, because a cluster absorbed by a merge stops being a root — so in that single tick the drained pool and any claim over it both name a force on its way out. The pool was remapped; the claim was not; the survivor's `Overlaps` then found no claim at all, and a player who mined a source-force part in the merge tick watched the items go on the floor.

**It fails closed, which is the whole reason a review found it and nothing else could.** Conservation is untouched — the pool settles into a successor, a pocket or the ground either way — and the wrong answer is the *absence* of a pocket, which no count anywhere can see. It is the narrowest window in the feature (a merge tick, a mine, the source force) and it is exactly the window a multiplayer administrator's keypress opens.

**The claim store moved to `guest/go/carry` with the predicate**, which is what makes the merge machine-checkable at all: `carry.Claims` is `Add`, `BeneficiaryFor`, `FollowMerge` and `Reset` over pure data, and `carry.Region.FollowMerge` is the one statement of the merge rule that both `remapCarryForce` loops now call. Transcribing the shipped remap — the pools alone — fails two of the eight new tests:

    --- FAIL: TestAMergeCarriesTheClaimWithThePool
        the surviving network credited player 0, not 7: a claim left naming
        the destroyed force is a claim nobody's pool can answer
    --- FAIL: TestNothingKeepsNamingTheDestroyedForce
        claim 0 still names force 2, which no longer exists

The `edge` suite's `merge_forces` leg is unchanged and could not have caught this: a headless `--create` has no player, so `player_index` is 0 on every removal any suite can produce and the claim list is empty in all seven. The leg carries a comment saying so and pointing at the test that does cover it — `go test ./carry/`, which `make check` runs.

Zip **287,422 → 287,782 B (+360)**, `fk_module.lua` **2,224,002 → 2,228,940 B**; all seven suites green from clean in the shipped config and green again in the `GC=leaking` arm, with every number byte-identical for the reason above.

**And what the second field report cost, over the same round**: zip **287,782 → 288,684 B (+902, +0.31%)**, `fk_module.lua` **2,228,940 → 2,246,630 B (+0.79%)**. No member, define or prototype was added — the whole change is a call site in the neighbour gate, a dedupe loop in `Claims.Add` and one log line.

## Stacked belts come back stacked — the gate that costs vanilla nothing

**The second half of the same defect, found by reading the first half's own caveat.** "A recompile is not a removal" fixed *where* a teardown's items go and recorded one thing it could not recover: **how they were STACKED**. The drain's instrument was `get_contents`, which returns totals per (name, quality), so `insert_at`'s `belt_stack_size` was passed absent and a Space Age player's 4-stacks came back as four times as many single items. Conserved, and wrong: a quarter of the density, a throughput dip until compression recovered — and, it turns out, **items on the floor**, because a network handed back four times as many belt positions as it had does not fit in itself.

**The kind key is (name, quality, stack size) now**, the drain reads `get_detailed_contents` — one `LuaItemStack` handle per belt POSITION — and the reinsertion passes the stack size it drained. `guest/go/carry.go`'s header is the long form and `compile.go`'s `detailedTally` is the read.

### The gate, and why the force bonus is the right one

**`LuaForce.belt_stack_size_bonus`, read once per force per carry transaction, in two host calls and no allocation** (an entity we are already holding for its force, and the force for its bonus). Zero — which is all of base Factorio and all of Space Age before the research — takes the path that was already there, byte for byte. Measured headlessly before any of it was written:

| probe | result |
|---|---|
| a loader prototype with `max_belt_stack_size` in **base** | **refused at load**: *"Belt stacking is disabled and can not be used. Belt stacking requires space-travel"*. The data-stage flag is `feature_flags["space_travel"]` — an underscore, where the error message has a hyphen |
| the same loader under Space Age, force bonus **0** | delivers **singles**. The bonus is what turns gameplay stacking on, and it only ever goes up |
| the same loader, force bonus **3** | 4-stacks, `get_item_count` 16 on a belt line that holds 4 positions |
| `insert_at(p, {name, count = 4}, 4)` on a **bonus-0** force | **places a stack of 4 anyway.** The bonus does not gate the API |
| `insert_at(p, {name, count = 9}, 4)` | places **4**, returns **true** — the call is atomic per POSITION and truncates the count to the stack size |
| `insert_at` onto an occupied position | **false**, and nothing is placed. So `ok` is the only return that has to be believed |
| `insert_at(p, ..., 255)` | accepted. The engine clamps nothing, so a drained stack is reproducible exactly; the uint8 parameter is the only ceiling |

The fourth row is what the gate does **not** cover and it is stated rather than hidden: a third-party mod that scripts stacks onto a bonus-0 force's belts gets them back unstacked. That is the same shape as the failure envelope above — conservation always holds, fidelity is best-effort, and the next audit re-reads the world.

### What the detailed read costs, and where it is not paid

Per **non-empty** transport line, above the gate:

| | host calls |
|---|---|
| below the gate (all of base) | `get_item_count` + `get_contents` — **exactly as before** |
| above it, line **not** stacked | + one `get_detailed_contents`, and **nothing per position**: `len(detailed) == get_item_count` means every position holds one item, so the flat totals are already the right answer and are taken |
| above it, line stacked, **one** item kind on it | + one `count` per position. The name and the quality come from the totals, so no string crosses the boundary and nothing is allocated |
| above it, line stacked, several kinds | + `name_is` per distinct name until one answers (a bool return over a string the guest already holds — never `name`, which would copy the host's bytes into the guest heap), and `quality` only to break a tie between two entries of the same name |

**And the reinsertion got CHEAPER, which is the opposite of what the pass expected.** `insert_at` is atomic per position, so a group of 24 items in 4-stacks costs six calls where the old code made twenty-four. Measured on the `plat` suite's own profiler — the audit-forced recompile, medians of three interleaved runs, each minus that run's own `audit only, nothing pending` control (7.56 / 7.37 ms):

| recompile of a **stacked** rig | stack-aware | flat (the gate forced off) | |
|---|--:|--:|---|
| `full`, a dead-ended 4×4 holding 928 items | **19.63 ms** | 33.59 ms | **1.71× faster** |
| `flow`, a running 4×4 | **11.77 ms** | 19.02 ms | 1.62× faster |
| `plain`, 2 parts, **unstacked lines above an open gate** | 6.58 ms | 6.30 ms | **+0.28 ms** — the one cost |

The last row is the whole price of the gate being open on a line that turns out not to be stacked: one `get_detailed_contents` per non-empty line. Everything else in the table is a win, and the flat arm also **spilled 320 items onto the ground** in the same run, because 928 items reinserted as 928 positions do not fit in a network that held them in 232.

**That table was measured on a FOUR-band save and the suite has five bands now**, so it is kept as measured and is not comparable with anything taken since. The profiler window is around the AUDIT rather than around a tick pair — it has to be, because the sample either side has to be atomic — so it carries a whole-save re-classification, and adding a band makes every cell bigger including the control's. Measured 2026-08-05, medians of three, same method: the `audit only, nothing pending` control is **10.85 ms** against 7.56, and net of it `full` is **26.92 ms**, `flow` **14.35**, `plain` **7.66** and the new `smix` **12.46**. Same ratio to the control, one more band under it.

**Nothing base-only moved, and that is measured rather than asserted.** The `mar` suite's seven slopes under `-gc=leaking` came out **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — and the M2 recompile timings, the `edge` suite's medians and every rate in every suite are unchanged.

### The leg, and why it is in `plat`

**Belt stacking is a Space Age feature at the PROTOTYPE level**, so no base-only suite can build a stacked belt at all — the first probe above is the load error that says so. The leg therefore lives in the `plat` suite, which is now the **Space Age suite** rather than the platform one: two legs with the DLC in common and nothing else, because a second DLC-only suite would cost a second Factorio run for one rig.

It builds on its own flat scratch surface on its own force (`bbb-stack`, `belt_stack_size_bonus = 3`), so **one save exercises both arms of the gate** — the platform leg's rigs are on `player`, whose bonus stays 0. Four bands: a stacked control belt, a dead-ended stacked 4×4 (`full`), a running stacked 4×4 (`flow`), and `plain` — the same stacking force fed by an ORDINARY loader, which is the branch that opens the gate and then finds nothing stacked.

**Every band is two columns of parts since the 2026-08-24 single-edge re-lay, plus one EDGELESS part on the end**, and that last part is what the recompile's belt lands against: under Factorio 2.1's rule a working balancer has no free face, so the belt that used to arrive on the block's north face would be refused rather than compiled and the band would measure a refusal. Every number in the table below is unchanged by that, to the item, which is the expected result rather than a lucky one — a hidden network is a function of the BELTS and the belts did not move.

The assertion is on the stack PROFILE of the hidden surface, sampled either side of a forced recompile inside one tick (`assert-plat.py`). Conservation was never the defect and is not the headline:

| | measured |
|---|---|
| stacks really formed first | 1,128 items over **336 belt positions — 72 of size 1, 264 of size 4**. A run where stacking silently did not happen fails as vacuous |
| `full`, a stacked 4×4 recompiled | 10,952 items before and after; the single-item positions moved by **exactly 0** and the 128 items that crossed into the network are **all** in the stacked total |
| `plain`, unstacked under an open gate | the exact opposite, exactly: **+16 singles, +0 stacked**, and no stack invented |
| `flow`, recompiled while stacked items are moving | +0 singles, **+48 all stacked**, and over the 500 ticks starting 200 after its own recompile it delivers **1496 1488 1492 1496 against one saturated stacked belt's 1504 — 3.971×, spread 0.54%** |
| spills | **0** across all three recompiles |

**The leg was verified to fail on the unfixed guest**, which is the only thing that makes it evidence: with the gate forced off, six assertions fire — 572 more single positions on `full`, 304 more on `flow`, 320 items on the ground, and the throughput falls to **3.884×** with the spread at 1.23%. **Re-run 2026-08-05 with the fifth band in the save it is nine assertions and 392 items on the ground**, the extra three and the extra 72 being `smix`'s; every one of the original six fires with the number it was recorded with. **Re-run again 2026-08-24 against the single-edge bands it is the same nine and the same 392**, with 572 and 304 to the position and `smix`'s 56×4 histogram becoming 196×1 — and per-kind conservation staying EXACT, because the gate-off path is still conservation-correct and only unstacks.

Two things about the numbers. The `plat` timings carry a whole-save re-classification as well as the recompile (the audit marker is what makes an atomic before/after sample possible at all), so they are comparable **only** with each other and with the same measurement from another build — never with M2's tick-pair recompile figures. And the stacked window is measured from 200 ticks after the recompile rather than from the recompile itself, because a rebuild puts every drained item back at the HEAD of the butterfly and the outputs are briefly starved by construction — the same shape the `edge` suite measures the same way.

**What it costs to ship**, same flags, same pin, measured either side of the change: `fk_module.lua` **2,046,296 → 2,112,476 B (+3.2%)**, the zip **273,917 → 278,309 B (+1.6%)**, `dist/bbb.wasm` 942,882 → 957,817 B. That is a per-load handful of kilobytes for a path most players never enter, and it is the only cost on this side of the scale.

The mod ships **36 bound members** now, up from 28: `get_detailed_contents`, `LuaItemStack`'s `count` / `name_is` / `quality`, `LuaQualityPrototype.name_is`, `LuaEntity.force` and `LuaForce.belt_stack_size_bonus`. No new FkLua gap; every one of them was reachable through the generated bindings as they stood.

### Stacked sushi — the three branches `kindAt` has, and the two nobody had run

**`detailedTally` reads a stacked line position by position and `kindAt` says which (name, quality) total each position belongs to. It has three branches and exactly one of them had ever executed, in any run, on any machine.** Closed 2026-08-05 by the `plat` suite's fifth band; no guest line changed, which is the point — this is the measurement the path never had.

| `kindAt` branch | what it needs | before |
|---|---|---|
| `len(totals) == 1` — no host call at all | a stacked line carrying **one** kind | every stacked rig in the repo |
| the `name_is` loop over candidates | a stacked line carrying **two names** | **never run** |
| the `quality` tiebreak, plus `askedAlready` | a stacked line carrying **one name at two qualities** | **never run** |

**Why eight suites could not reach it, and it is a two-sided gap rather than a missing rig.** Multi-kind rigs live in `mix`, which is base-only — and below the stacking gate `drain` uses the flat totals and **`detailedTally` is never called at all**, so `mix` cannot reach `kindAt` however many kinds it runs. Every rig above the gate is Space Age and single-kind iron plate, so it takes the cheap branch. The two conditions have to be met by ONE line at ONE moment, and only Space Age can do it.

**The rig.** `smix` — a dead-ended 2→2 on the stacking force, fed by two **stacked sushi** sources: `bbbt-stackloader` behind an infinity chest holding ONE filter at a time and rewriting it every four ticks with `remove_unfiltered_items` on. That is `mix`'s own technique and it is a measurement rather than a preference: the naive rig, one chest carrying six filters at once, delivers a single kind (2,292 items, electronic-circuit 100%), because a loader draws from the first stack it finds and the chest tops that same stack straight back up.

**Six names none of the other bands touch**, so the four iron-plate bands keep their recorded figures to the item while this one is measured independently — and one of them, `plastic-bar`, is on the rotation **three times over, at normal, uncommon and rare**, consecutively, so two qualities of one name land on one line. `quality` is an optional key of `InfinityInventoryFilter` in the pinned 2.0.77 runtime API and the `quality` mod is already loaded here, so the tiebreak costs no new dependency.

**The rotation period is a measured constant, not a preference.** A hidden belt tile holds four item positions per lane and a stacking loader at express rate emits ~3 items/tick over two lanes, so four ticks is ~1.5 stacked positions per lane — comfortably shorter than a line, which is the whole condition. A period longer than a line gives single-kind lines and the band would pass every assertion while exercising nothing, so the suite **samples the world and fails if the shape did not land** rather than assuming the arithmetic.

Measured 2026-08-05, Factorio 2.0.77, shipped config:

| | measured |
|---|---|
| **anti-vacuity, at the instant the teardown read them** | of 24 hidden lines carrying this band, **14 carried two or more NAMES and all 14 had a stacked position**; **6 carried one name at two QUALITIES**, all 6 stacked; biggest stack 4. Both `kindAt` branches reached, by measurement |
| kinds in flight | **9** of 9 (name, quality) pairs, 704 items — inside the carry pool's 32-group bound by design, since overflowing it is `mix`'s job and here it would spill |
| conservation across the recompile | **EXACT per (name, quality)**, all nine: copper-cable 120, copper-plate 80, electronic-circuit 112, iron-gear-wheel 80, plastic-bar **normal 40 / uncommon 80 / rare 32**, steel-plate 40, stone-brick 120 |
| the stack profile | **64 items crossed, +0 single and +64 stacked** — 56 positions of 4 became 72 of 4, and not one position of 1 was invented |
| spills, and pool overflows | **0** and **0** |
| the network | a 2→2 recompiled to 3→2 over 4 ports, 28 entities, **288 items handed back** |

**Red-proven twice, and the two proofs catch different things — which is the result rather than a formality.**

| injected defect | what fired |
|---|---|
| **the stacking gate forced off** (`stacksPossible` returns false), the established proof for this leg | the three PROFILE assertions: **+196 single positions** where the fixed guest adds none, the 56×4 histogram becoming 196×1, and the crossing going negative. **Per-kind conservation stayed EXACT** — the gate-off path is still conservation-correct, it just unstacks |
| **`kindAt` always answers candidate 0**, so every position is attributed to the line's first total | **only per-kind conservation**, and it named seven of the nine kinds: copper-cable 120→88, copper-plate 80→104, electronic-circuit 112→96, plastic-bar normal 40→64, **rare 32→16, uncommon 80→64**, steel-plate 40→24, stone-brick 120→168. The stack profile was **byte-identical to the healthy run** (+0/+64), the item TOTAL was unmoved at 704, and every other band in the suite passed |

The second row is why the counting is per (name, quality) and not per name and not a total. A misattributing `kindAt` hands back the right number of items under the wrong key: a single total sees nothing, a per-NAME total would still have missed the plastic-bar rows, and the stack profile — the leg's existing instrument — cannot see it at all. The two proofs together say the band's assertions are not redundant with each other.

**It costs one Factorio run of nothing.** No guest line moved, so the `mar` slopes came back identical to the byte (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) and the shipped zip did not change. Inside the suite the save gains a band: the four bands above are unmoved to the item, `plat`'s profiler control rises 7.56 → 10.85 ms for the reason the timing note above gives, and the recompile count the suite requires goes 3 → 4.

## The sixty-fifth belt — refusing without demolishing

**`plan.MaxPorts` is 64 and refusing past it was always correct. What the refusal did FIRST was tear down the working balancer it was refusing to replace.** `compile()` classified the edges, saw the fingerprint move, called `teardownForRebuild(root)`, and only then asked `plan.Build` whether the shape fit — so one belt too many against a 64-port balancer demolished it, built nothing, spilled the whole machine on the floor and said so only in the log file. Found by reading `agents/maxports.md` §4, which had it written down as a queued task; closed 2026-08-04.

**Everything below is one statement moving twenty lines up, and everything around it.** `plan.Shape(n, m)` needs the edge COUNTS and nothing else, and they are in hand the moment `classifyEdges` returns.

### Measured, both sides of the change

The `edge` suite's new `lim` rig: a column of thirty-two parts with a belt on both sides of every one of them pointing inwards — **sixty-four inputs and one output, which is P = `next_pow2(64)` = `MaxPorts` exactly**, 1,026 hidden entities, the biggest network this mod builds. A sixty-fifth input belt is laid on it while it runs and then mined off again. Same rig, same schedule, the only difference being where the check sits:

| | check AFTER the teardown | **check BEFORE it** |
|---|--:|--:|
| the standing network | **torn down, 1,876 items drained** | untouched |
| items on the ground | **1,690** (`ground` 336 → 2,026) | **0** |
| delivery over 246 ticks, before → after the edit | 184 → **10** | 184 → **185** |
| the audit afterwards | `nets=9 drift=0 unbuilt=1` | `nets=10 drift=1 unbuilt=0` |
| the log | `[BBB] error:` **three times** for one belt | one `[BBB] alert:` |
| `test/run.sh` | **killed the run** on the first error line | green |

The ten items in the "after" column of the failing run are the output belt draining; the machine had stopped. And **the item total was conserved to the item throughout**, which is exactly why eight suites were green over this for five milestones: the defect was never a count, it was where the items were and whether the machine still existed.

### The four pieces, and the one that is sharp

`guest/go/limit.go`, whose header is the long form.

1. **The check moves in front of the teardown** (`compile.go`). `overLimitShape(edges)` mirrors `plan.Build`'s own two tests — a cluster with no inputs or no outputs is a legitimate half-built state at ANY size, so refusing one here would refuse every single-sided cluster in the world. `plan.Build` keeps its `!fits` branch as an unreachable backstop and it is still an `error:`, so a crack between the two mirrors fails a run.
2. **The player is told.** `noteBuiltByPlayer` writes down `(surface, tile, force, player_index, isPart)` for every addition a PLAYER makes on or beside a cluster — the miner's-pocket pattern exactly, scalars only, **no host call**, and nothing at all when `player_index` is zero. `on_built_entity` is the only build event that carries one, so a robot, a script build, a revive and a clone all record nothing, and so does every event in every headless suite. At refusal time the player is resolved FRESH with `game.get_player` and gets `create_local_flying_text` at the refused piece — the box centre until the 2026-08-05 interactive check found that on a 32-part column it spawned seventeen tiles off the placing player's screen — in **vanilla's cannot-build red** rather than the default white, plus `utility/cannot_build`; a robot or script build gets `force.print` with a **different** locale key, because "you got it back" would be a lie. The sentence itself counts BELTS PER SIDE and not ports, and deliberately does not narrate the hand-back: both are the same playtest's, and all four are written up in "The wake race" below.
3. **The piece is handed back, and WHERE that runs is the whole design.** `revertOverLimit` is called from `flush()` **after `endCarry()`** — after every teardown, every build and the settling of the carry transaction — and may not run any earlier, for three reasons that are each sufficient: `mine_entity` dispatches `on_player_mined_entity` SYNCHRONOUSLY, so it re-enters the registry, frees nodes, re-roots components and refills the very queues `flushLive` is iterating; the compile buffers (`tileBuf`, `edgeBuf`, `opBuf`, `entBuf`) are package level and not re-entrant; and a claim recorded before `endCarry` would be settled against pools the miner had nothing to do with. Recorded after, it lives exactly one flush and is answered by the teardown the revert itself provoked. The mine takes the fingerprint back to what the `netInfo` already holds, so the next flush is a **SKIP**: at no point in the whole sequence is anything torn down or rebuilt. `force = nil` on the mine, which is vanilla's rule — a full inventory declines and the piece stays standing rather than evaporating.
4. **The feedback gate, which is not optional.** A refused cluster is re-queued by every audit and by every event within two tiles of it, and its fingerprint can never match its `netInfo`'s, so it reaches the refusal EVERY TIME — three times for one belt in the measurement above. `overLimit` is a root → refused-fingerprint map, **point-queried only, never ranged**, nil until the first refusal (a lookup and a `delete` on a nil map are both free, so a save that never hits the cap never allocates it), and it fires the message once per distinct edge state. The suite asserts **exactly one**. **A refusal issued from inside `rebuildFromWorld` does not arm it**, and that is not a detail: the memo is what silenced the one refusal that had never been delivered, the first time a player laid the sixty-fifth belt as the first event of a session. See "The wake race" below.

### What is verified, and the one thing that is not

The `edge` suite's LIM section asserts the refusal line and its numbers (128 ports for 65 inputs against a limit of 64), one refusal per edit, zero ground items, an unmoved conserved total, the audit's `drift=1 unbuilt=0`, and delivery holding across the edit as a ratio between two equal-length windows — with a `before` window that must be non-zero, so a dead control fails rather than passing vacuously.

**The ROBOT arm of the feedback is verified and the PLAYER arm is not**, and the split is exactly where the wall is. Every build in every suite is a script build, so `player_index` is zero and the fork always takes `force.print` — which means the LocalisedString really does cross the boundary and the `LuaForce` really is resolvable from a force INDEX, which is the part that could realistically fail (the registry keeps no force handle; it comes off a part standing on the cluster). Neither is readable back from Lua — `force.print` goes to the game's chat and `--benchmark` does not log it — so the guest writes one line saying it happened and the suite asserts that line, **once, with no error**.

**The flying text, the sound and the hand-back need a player, and join the miner's pocket on the interactive checklist.** Same wall, measured the same way: a headless `--create` has no players, so `game.get_player` resolves to nothing and `revertOne` returns before it mines anything. What the suite asserts there is the NEGATIVE — **zero pieces handed back over the whole run** — exactly as it asserts zero pockets, so a revert firing for a script build fails the run.

**What the user tests interactively**, and it is one gesture: build a balancer with sixty-four belts against it, leave one part of it with no belt, and lay a sixty-fifth belt against THAT part. (Under the one-belt-per-part rule a sixty-fifth belt laid anywhere else is the other refusal, so the spare part is what makes this the port-limit gesture; the checklist's band C stages it.) Expected — the flying text naming the limit, the standard cannot-build sound, the belt back in the inventory, and the balancer still running exactly as it was. Then do it again with the inventory full: the message still appears and the belt stays standing unconnected, with a `[BBB] alert: player … could not be handed back` in the log. Then have a construction robot place it from a ghost: a force-wide chat message instead, and the belt stands. Grep for `handed the refused piece` for the first case; the second and third are the two lines either side of it. Both hand-back lines NAME THE BOUND THAT FIRED -- `(over the port limit)` here, `(past the one-belt-per-part rule)` for the belt gesture -- because `tellRefusal` and `revertOne` are shared by both bounds and the sentence used to name only this one.

### The shape this left uncovered, and the pass that covered it

**A part bridging two clusters into an over-limit one demolished both**, and it shipped that way for a day. `AddPart` marks the two predecessors DEAD, so `flushDead` had already brought their networks down by the time `flushLive` reached the merged cluster and the check above had nothing left to protect. It is closed — see "The merge that would be over the limit" below, and `agents/maxports.md` §5.

### What it costs

Package built 2026-08-04, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 300,459 B | **310,630 B** | +3.38% |
| `fk_module.lua` | 2,319,175 B | **2,432,188 B** | +4.87% |
| `dist/bbb.wasm` | 1,039,715 B | 1,076,918 B | |
| members bound into the mod | 38 | **42** | of 4,257 |

The four new members are `LuaPlayer.create_local_flying_text`, `LuaPlayer.play_sound`, `LuaControl.mine_entity` and `LuaForce.print`. No new FkLua gap: every one was reachable through the generated bindings as they stood, and `LuaSurface.find_entity` and `game.get_player` were already bound.

**Nothing on any hot path moves, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — which is the check that says a build note recorded on the guest's highest-multiplier path costs nothing where no player builds. `flush()` gains one length test on a nil slice; `compile()` gains a loop over an edge list it has just walked anyway, and a `delete` on an empty map.

## The merge that would be over the limit — the other half of the same refusal

**"The sixty-fifth belt" moved the check in front of the teardown `compile()` does to ITSELF, and shipped with a note saying that a MERGE past the limit still demolished both balancers.** That note was accurate for a day. Closed 2026-08-05; `guest/go/limit.go`'s "The merge" and [`agents/maxports.md`](agents/maxports.md) §5 are the long form.

**Whose teardown it is, is the whole difference.** An edge edit's teardown belongs to `compile()`, which is why one statement moving twenty lines up fixed it. A merge's teardowns belong to `AddPart`, which marks BOTH predecessors' roots dead the moment the bridging part arrives — so `flushDead` had demolished two working balancers before `flushLive` so much as looked at the cluster they became. The items were conserved (the carry transaction settled them, unclaimed, onto the ground) and the revert still handed the part back, so both balancers returned on the following tick **empty**, with everything they had been holding in a heap on the floor between them. The alert said *"refused BEFORE the teardown, so the standing network is untouched"* while it happened: true of the cluster being refused, which had no network of its own, and a lie about the two that did.

**`spareOverLimitMerges` runs at the top of `flushDead`** — the one choke point every teardown in this guest goes through — and takes the merge's teardowns back off the queue. `compile()` then reaches the merged cluster and refuses it through the path that was already there.

### Why it is free, which is what the old note got wrong

The note said covering it *"means classifying the merged cluster's edges before `flushDead` runs, which is a second classification pass on every flush, on the guest's hot path."* True of the obvious implementation. Two things make the pass cost nothing:

- **A dead root that is NO LONGER A ROOT has been absorbed**, and what absorbed it is the cluster the flush is about to build. Exact for a merge, one `find` per queued root, and it rejects every other queue filler for free: a shrink, a split, an audit repair, a clone reconcile and a surface deletion all queue roots that are still roots. Only a candidate that had something absorbed into it may have teardowns taken away, which is also what keeps the pass out of `dropSurface`'s way.
- **A cluster of C parts has at most 4C exterior sides and therefore at most 4C edges**, so `4*csize[r] <= MaxPorts` is a PROOF that no classification could find enough of them. Sixteen parts is the largest cluster provable that way, and every balancer any suite in this repo merges is smaller — the `mar` suite's merge leg is five parts.

Being wrong about the bound costs exactly the old behaviour: the merge falls through to the refusal in `compile()`, where it always was.

### What the registry believes while a refused merge stands

**One cluster, two networks, and at least one of them keyed by a node that is no longer a root.** It is the only state in this guest where `nets` holds something `liveRootList` can never reach, so it is REPORTED rather than left to be discovered: `strandedNets` tells `auditAll` how many networks a refused merge left behind, the audit counts them in `nets=`, and the merged cluster comes out `drift=1 unbuilt=0` — an edge list past what the mod can build, and a guest that knows it. **`drift=0 unbuilt=1` there is the signature of the defect.**

Which key the survivor's network is under is not fixed and must not be assumed: `newNode` reuses freed ids, so the merged root can be either predecessor's root OR the bridging part's own brand-new node with no network at all. The measured run is the second case, which is exactly why the audit asks limit.go instead of reading `nets[root]` and drawing a conclusion.

**The state is stable and the suite asserts it**: four audits while the refusal stands, the same report from all four, exactly one `alert:` for the whole edit, and no teardown, build or spill in between. Three ways out:

| what happens | what the pass does |
|---|---|
| the edge list shrinks back under the limit | every stranded network is queued dead, INCLUDING the merged root's own — a cluster that tears itself down in `compile()` opens an `owned` pool and then declines every geometric one (carry.go, `takeCarry`), so the other predecessor's items would spill. Down in `flushDead` instead, both pools are unowned and the one network that goes up claims them both |
| the bridging part is mined — the revert, or a player | the cluster splits back into what it was, each component re-roots at its smallest node id (which is the root it already had), and each half's fingerprint is the one its `netInfo` never lost. **Both compiles are a SKIP; nothing is torn down and nothing is rebuilt** |
| the stranded node is FREED — a dissolve in one event, a surface deleted | nothing can ever reclaim that network and its id is on the free list, so `sweepStranded` brings it down on the next flush before the id can be reused under it. A dissolve is a removal, so it spills, which is correct |

### Measured, both sides of the change

The `edge` suite's new `brdg` rig, and the only difference between the columns is the one call at the top of `flushDead`:

| | no pre-pass | **the pre-pass** |
|---|--:|--:|
| the two standing networks | **both torn down, 1,044 items each** | untouched |
| items on the ground | **+1,814** (`ground` 336 → 2,150) | **0** |
| delivery over 246 ticks, before → after the edit | 186 → **8** and 185 → **8** | 186 → **184** and 185 → **184** |
| the audit while the refusal stands | `clusters=11 nets=10 drift=0 unbuilt=1` | `clusters=11 nets=12 drift=1 unbuilt=0` |
| the bridging part mined off again | both halves rebuilt from scratch | **0 teardowns, 0 builds** — a SKIP |
| visible interfaces of ours standing | 114 | **180** |
| the conserved total | unmoved | unmoved |

**The item total was conserved to the item in both arms**, which is why this was invisible for a milestone: what moved is where the items were and whether the machines still existed.

### What it costs

**Nothing on any hot path, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — and leg F *is* the merge leg, a balancer grown by a part and taken apart again a hundred times. Six parts is under the sixteen the bound proves safe, so the pass never classifies and never allocates. Package built 2026-08-05, shipped config:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 310,628 B | **316,749 B** | +1.97% |
| `fk_module.lua` | 2,432,188 B | **2,516,161 B** | +3.45% |
| `dist/bbb.wasm` | 1,076,928 B | 1,098,032 B | |
| members bound into the mod | 42 | **42** | of 4,257 — none added |

### The merge shapes that are STILL not covered

Two, and they are the same one seen twice: **a clone reconcile (`reconcileArea`) and `game.merge_forces` both call `flushDead()` themselves, before the merging happens.** Their sequence is *bring every affected network down → reconcile the registry → rebuild*, so by the time the merged cluster exists its predecessors are already gone and there is nothing to spare. Items are conserved and both balancers come back empty, exactly as the bridging part used to do. Neither is a gesture a player makes — a clone big enough to bridge two 33-port balancers is a mod or the map editor, and `merge_forces` is an administrator's keypress — and covering them means the same decision in the middle of two paths whose whole design is a wholesale rewrite. Written down rather than done; `agents/maxports.md` §5 carries it too.

## The wake race — what a guided playtest found that nine suites could not

**A guided interactive playtest on 2026-08-05 walked the staged gestures against a real graphical client, and every defect it found is one no headless run can reach.** Nine suites were green throughout and always would have been: a `--create` has no players at all, so `player_index` is zero on every build and every mine any suite can produce, and the whole feedback half of "The sixty-fifth belt" — the flying text, the sound, the hand-back — sits behind the same wall as the miner's pocket's trigger. Everything it found is the table at the end. The first item needs a section of its own, because it is not about words: it is about the one dispatch in which this guest is awake and knows nothing.

### The wake, and the gesture that opens it

**`make install` over a mod of the same version is a silent guest swap**, and it is what an author does twenty times an afternoon. Factorio raises no `on_configuration_changed` for it — nothing was added, removed or re-versioned — so neither `fk_migrate` nor `fk_on_configuration_changed` fires. The packaged guest is a different BUILD, though, so the saved heap is declined and the mod comes up on a fresh one: an empty registry over a world full of parts and full of running networks. That is exactly the state "Coming back on a heap this build did not write" is about, and `registryReady` is the fallback that covers it — false in a freshly initialised heap, so the FIRST EVENT of the session rebuilds the registry from the world before it decides anything (`ensureRegistry` at the top of `onEventBody`, main.go).

**What nobody had asked is what happens when that first event is a PLAYER BUILD that takes a balancer past the limit.** Then the rebuild and the build are ONE dispatch, in that order, and the rebuild goes first with information the event has not handed over yet:

> `rebuildFromWorld` runs its own `flush()` — it has to: a rebuild that merely queued would leave every cluster it could not adopt uncompiled until a tick arrived, and in a `--create` none ever does. That flush compiles the cluster the player's new piece just made over-limit, and `buildNotes` is still EMPTY, because `noteBuiltByPlayer` is called from `onPart` and `onNeighbour`, and neither has run.

lifecycle.go's own comment is the shortest statement of the problem: the rebuild "judges the world with the worst information a refusal will ever have".

### The first head — the rebuild spoke, and it was wrong two ways

Both symptoms are one window seen twice, and lifecycle.go records them in the order they were reported:

| | what the player saw | what did it |
|---|---|---|
| **first** | nothing at all: the piece stayed, no text, no sound, no hand-back | the rebuild's refusal armed the feedback memo, `overLimit[root] = fp`. The informed flush a tick later found `prev == fp` and returned before it said or handed back anything — the gate doing exactly its job, to the one refusal that had not yet been delivered |
| **then** | a chat line saying the extra piece was **left in place, unconnected** — one tick before the piece arrived in their inventory | with no build note beside the cluster, `tellOverLimit` takes the `told == 0` fork, which is the ROBOT sentence and a `force.print`. The informed retry then did the player thing and handed the piece back, so the player was told one thing and given another |

**One branch cures both, and it is the first thing `refuseOverLimit` does.** `rebuildingFromWorld` is up for exactly the span of `rebuildFromWorld`, including the flush it runs itself (lifecycle.go sets it before `collectSurfaces` and clears it after `flush()` returns). A refusal issued under it **logs, requeues and tells nobody**:

- it **logs**, through the same `logRefusedOverLimit` the ordinary path uses, so the log stays a complete record of every refusal and the `edge` suite's line is the line it always was. The gate is on the MESSAGE, not on the log;
- it **does not arm the memo**, so nothing suppresses the informed retry;
- it **does not speak**, so nothing claims a final state one more tick can falsify;
- it appends the root to `rebuildRefused` — **not `markLive`**, and that is the same after-the-drain discipline `revertOverLimit` follows for its mine: this runs inside the rebuild's own `flushLive` drain, whose loop has captured its length and whose tail truncates the queue to `[:0]`, so an append there would be silently erased. `rebuildFromWorld` requeues them AFTER its flush returns and asks for another one.

The next ordinary flush then re-judges with whatever notes the rest of the dispatch recorded and delivers the one correct message. **A refused cluster in a save nobody is editing reaches that flush with no notes and gets the piece-stands message, which is then true.**

### The second head — a build event the registry has already seen

Found on the staged bridge gesture (main.go cites it as *the bridge gesture*), and it is what makes the requeue above worth anything.

**The engine places the entity before it raises the build event**, so when that build event is the first of the session the rebuild at the top of the same dispatch scanned a world that already contained the part — and registered it. `AddPart` therefore reports **no change**, and `onPart` used to return right there: no build note, no queue insertion. The cluster the rebuild had requeued would then be re-judged with nothing beside it, take the `told == 0` fork again, and the over-limit piece would stand with nothing but a chat line.

`onPart`'s `else if id, dup := index[k]; dup && builtBy != 0` branch is the fix: record the note, `markLive(find(id))`, `logState()`, ask for a flush. Three things make it safe outside the window it was written for, and all three are the code's own:

- **outside the wake race a duplicate PLAYER build event is a mod raising `built` twice for one entity**, and what it costs then is a note plus a flush that skips on the fingerprint;
- **`builtBy` is 0 for scripts and robots**, so the branch is not entered for them at all — and their wake-race outcome (the piece stands, the force is told) is the designed one already;
- `noteBuiltByPlayer` drops exact duplicates, so a note recorded twice is one note.

**The belt at the edge needs no second head**, and the asymmetry is worth knowing before anyone tidies it: `onNeighbour`'s gate is PROXIMITY, not novelty. It walks the 5×5 neighbourhood, queues every cluster it touches and records the note whenever `builtBy` is non-zero, whether or not anything about the registry changed. Only the part path gates on `AddPart` having done something.

### The principle: the rebuild never speaks, only informed flushes do

That is the durable half of this pass, and it is worth stating on its own because the next thing that wants to talk to a player will be written somewhere the same argument applies. `rebuildFromWorld` reconstructs a whole session's registry from a world it has never seen, inside one dispatch, with none of that dispatch's own events yet delivered. Any verdict it reaches about a PLAYER is provisional by construction. So it may log, it may queue, and it may not address anybody — the flush that has the notes is the one that speaks, and there is always one, because the rebuild requeues what it refused.

### What is verified, and what is behind the wall

**The wake race itself needs a player AND a same-version reinstall**, which is two things a headless run does not have and one of them is not even about players: every suite's two phases run one mod set built once, and the `mig` suite — the only one that changes a mod set at all — changes a NEIGHBOUR's, which is precisely the case that DOES raise `on_configuration_changed` and therefore never reaches this window. So the trigger is on the interactive side of the wall with the flying text, the sound and the hand-back it is about. `test/interactive/README.md`'s gestures C and D are the two gestures it happens to; the wake race is either of them done as the FIRST thing in a session whose guest was swapped under it, which no staged rig can set up for anybody.

**What the suites pin is the negative, and it is the negative they already had.** `player_index` is zero on every build any suite can produce, so `noteBuiltByPlayer` returns before it appends and NEITHER HEAD CAN BE ENTERED: the `edge` suite's **zero pieces handed back over the whole run** still holds, a revert firing for a script build still fails the run, and the `lim` and `brdg` legs' "exactly one `alert:` per edit" still means what it always meant. What would show up in a suite if the rebuild path ever did start speaking is a second `alert:` line, since the rebuild path logs ungated by the memo — one refusal from the rebuild and one from the informed retry is the expected shape of a wake-race refusal in a log, and it is one refusal as far as the player is concerned.

### The rest of what the playtest found, and where each fix lives

| what it found | where the fix is |
|---|---|
| **The flying text was at the box centre.** A player laying the sixty-fifth belt at the top of the 32-part `lim` column "got the sound and never saw the text, because it spawned seventeen tiles south of their screen" | `tellOverLimit` (limit.go) moves it to the refused piece's own tile — "the one tile their eyes are guaranteed to be on" — taken from the first build note beside the cluster in EVENT ORDER, which is the same tie-break `carry.Claims.BeneficiaryFor` uses for a mine. The box centre is still computed above the loop, and the only branch that reads a position is the one a note has already moved |
| **The text was WHITE**, which is the default and "reads as information rather than refusal"; the check "expected red flying text, because that is what the base game has taught everybody a refusal looks like" | `limColor` (limit.go), 1.0 / 0.25 / 0.25 — "vanilla's cannot-build red, near enough" |
| **"would need 128 ports" read as gibberish.** A player placed one belt and has no window into the compiler | the message speaks in **BELTS PER SIDE**: `over-port-limit` takes the limit as `__1__` and `max(pt.N, pt.M)` as `__2__`. The LOG line keeps P — "it is for whoever reads logs, and ports are its native unit" — so the `edge` suite's numbers (128 ports for 65 inputs) are untouched |
| **The message narrated the hand-back**, and the round-trip is invisible at game speed: to the player this is a placement that simply failed, and a sentence about an inventory transaction they never saw "reads as a transaction to go looking for" | it deliberately does not say so any more. State the rule, like vanilla does, and stop (locale comments on `over-port-limit`) |
| **`Unknown key: "entity-name.bbb-linked-belt"`** in the engine's own *X is in the way*, which fires once per colliding entity and therefore names the invisible edge interfaces standing on a part's tile | `[entity-name]` entries for all four hidden prototypes, all reading **`Balancer`** — "the player sees one machine, so anything of ours in their way IS the balancer as far as they know". They have no player-facing surface of their own and this is the one place the ENGINE names them anyway |

### The one report that was not a defect

**Fresh spill appearing beside a balancer that had just recompiled**, and the mod placed nothing on the ground. If a balancer compiles on top of ground items — only possible where something else has already littered those tiles — the engine relocates them during entity placement: items on the part tiles are absorbed into the new interfaces' transport lines and ride through the machine to the outputs, conserved, and whatever exceeds the lines' capacity is nudged to the nearest free ground, which is the edge of the existing litter. That reads as a spill at the litter's frontier and it is not one.

**Measured by diffing the two autosaves that bracket the recompile**, which is the method the whole playtest used to keep an observation from becoming a report: **204 plates left the part tiles, 89 landed on the frontier ring, every item name conserved**, and the guest's log carries no spill line in that window. `test/interactive/README.md` keeps it under "A false alarm to recognise" so the next person to see it does not go looking for a leak.

### What it costs

**Nothing on any path an ordinary session takes, and it is structural rather than measured.** `rebuildingFromWorld` is one bool, tested only inside a refusal — which is a path a save that never hits the cap never reaches. `rebuildRefused` is "nil forever in a session that never rebuilds over a refused cluster, which is every session of ordinary play", and the requeue behind it is a length test on a nil slice once per rebuild, which is once per session. The second head costs one `index` probe on the branch where `AddPart` reported no change — a duplicate build, which is not a thing that happens on a hot path — and no host call anywhere. Nothing was added to the compile path, the neighbour gate or the flush proper, and no member, define or prototype was bound for any of it.

<!-- ============================ FAST REPLACE ============================ -->

## Fast replace — a part goes over a belt, and a belt goes over a part

**A balancer part can be placed over a piece of belt now, the way a vanilla splitter can.** It is one data-stage line and one guest check, and the two halves are not the same size at all: the line is the whole feature, and the check is the whole risk.

```lua
fast_replaceable_group = "transport-belt",   -- guest/go/data/entity.go
```

That string is base's own group for **every transport belt, underground belt, splitter and lane splitter** (`data/base/prototypes/entity/transport-belts.lua`; loaders are `"loader"` and linked belts are `"linked-belts"`, so neither is touched). `fast_replaceable_group` is an `EntityPrototype` property, so a `simple-entity-with-force` may carry one — checked against the pinned `prototype-api.json` rather than assumed.

**THE COLLISION MASK DID NOT MOVE AND MUST NOT.** `transport_belt` is still in it, which is what stops a player laying a belt THROUGH a balancer; fast replace is an exception the engine makes for the entity being REPLACED and for nothing else. `can_place_entity` for a part over a belt is still **false** after the change, and `can_fast_replace` is what went true. Measured, both before and after.

### What the engine actually does, measured

Probed on Factorio 2.0.77, base only, with a scratch mod. `can_fast_replace` is the question a player's cursor asks, so it is the one the table is about.

| the cursor holds | over | before the line | after |
|---|---|---|---|
| a balancer part | a transport belt | false | **true** |
| a balancer part | an underground belt end | false | **true** |
| a balancer part | a lane splitter | false | **true** |
| a balancer part | a splitter | false | false (two tiles wide) |
| a balancer part | a loader | false | false (another group) |
| a belt / underground / lane splitter | a part with **no** interface on its tile | false | **true** |
| a splitter or a loader | a part | false | false |
| any of them | a part that **carries an edge interface** | false | **false** |

The last row is the one that shapes everything below. `bbb-linked-belt` is a belt-connectable standing on the cluster's own tile, and a vanilla belt collides with it — so a part with any edge against it cannot be belt-replaced, and the reverse gesture reaches interior parts only. That is a consequence rather than a design, and it is written down instead of fought.

**And the four hidden prototypes really do stay out of it, though not for the reason `guest/go/data/hidden.go` gives.** `strip()` sets `fast_replaceable_group = nil`, and the engine does not read that as "no group": it defaults an entity with no declared group to a **singleton group named after itself**, which is what the runtime reports for all four (`bbb-linked-belt` → `bbb-linked-belt`, and so on) and what it reported for the part before this change. Either reading gives the same answer — none of them shares a group with a belt or with the part — and `can_fast_replace` is false for every one of them, which is the authority.

**The upgrade planner cannot pair a part with a belt.** Two entities are upgrades of each other only with the same group, the same collision BOX and the same collision MASK. The masks are identical (`floor+meltable+object+transport_belt+water_tile`, measured); the boxes are not — the part is ±0.35 and every belt is ±0.4 — so the box is the only thing keeping them apart, and it is worth knowing that it is what is doing the work.

**`create_entity{fast_replace = true}` is not the gesture and must not be used as one.** Handed a replace the engine would refuse, it falls back to CREATING, and a `simple-entity-with-force` is created whatever it collides with — so a script gets a part and a belt on one tile. Worse in the other direction: over a part that carries an interface it **mines the part and then fails to place the belt**, returning nil and leaving neither. The `edge` suite's rigs therefore ask `can_fast_replace` first and build only when the answer is yes, and the one leg that deliberately drives the refusal puts the part back afterwards.

### The forward direction needs no guest code, and that is a property rather than luck

The engine mines the belt before it creates the part, so the build event arrives in a world the belt has already left; `AddPart` registers the part inside the event and the compiler re-reads the world from the deferred flush a tick later ("THE REMOVAL WINDOW IS GONE", compile.go). Whether a PLAYER's fast replace also raises a mine event for the belt is unknown and does not matter: a mine on that tile reaches `onNeighbour`, which records a claim and queues the cluster, and both orderings end in the same recompile.

### The reverse direction is what `guest/go/fastreplace.go` is for

**A fast-replaceable group is symmetric.** The same line that lets a part replace a belt lets a belt replace a part, and the engine raises **no event at all** for the part it destroys. Measured, with a script standing in for the gesture: the only event in the whole dispatch is the BUILD event for the belt.

So without a check the registry keeps a **phantom** — a tile it calls a balancer part which is holding somebody's belt. Measured on the guest before the file existed: three parts became two in the world and the audit went on reporting `parts=3 drift=0 unbuilt=0`, because a phantom tile is INTERIOR, the belt standing on it is never classified, and the fingerprint therefore never moves. The cluster is then wrong for the rest of the session: **the belt a player laid does nothing at all**, and the tile is inside the box every teardown of that cluster sweeps.

**The check is one map probe on the appearance path and a host call only on a hit.** Is the appearing belt-connectable's own tile a registered part tile — the centre of the 5×5 neighbourhood `onNeighbour` walks anyway — and if so, is that part still standing? In ordinary play the first answer is always no, because the part's collision mask carries `transport_belt` and the only way a belt reaches a part's tile is by replacing the part.

**It is correct under every event ordering**, which matters because a headless run cannot say what a PLAYER's fast replace raises (there is no player in a `--create`, and the `edge` suite's existing probe asserts both walls). No mine event — the check removes the part. The mine event FIRST — `onPart` has already removed it, the tile lookup misses, and no host call is made. The build first and the mine after — the check removes it, and `removePart` on an unregistered tile is a no-op. All three end in the same registry.

**Who is credited is `builtBy`**, the player who placed the belt: a fast replace hands the replaced entity to the player doing the replacing, so if the shrinking machine cannot take back everything it was holding, that player is the one the overflow belongs to. It is the miner's pocket's rule (carry.go) reached through the other door, and `RemovePartMinedBy` is literally the same call `onPart` makes for a mine.

### The consequence a player will meet, stated rather than hidden

**Dragging a belt line across a balancer replaces the parts it can.** A drag is a sequence of builds and each one is subject to the same check, so a belt dragged over a balancer takes out every part with no belt against any of its free faces and is refused on the rest. That is exactly what vanilla does when a belt is dragged over a splitter, it is the price of the group being one string in both directions, and it is recoverable: the parts arrive in the inventory as items and the machine recompiles around what is left. A balancer whose every part carries an edge is immune by construction.

### Verified, and the two red proofs

Two rigs in the `edge` suite, `frepa` and `frepb`, and they are the only rigs in that suite **built mid-run** — after the last of its existing assertions has been made, so that not one baseline in it moved. Their source chests and their stock are created in `on_init` all the same, because `count_all` is a conserved quantity and inserting twenty-four thousand items into it halfway through would read as this mod minting matter.

| | measured |
|---|---|
| **forward**: a part dropped onto a belt of a live line beside a saturated 2-part balancer | `can_fast_replace` **true**, the belt gone, the part registered, the cluster 2 → 3 parts |
| ...and what it handed back | `express-transport-belt` ×1 and the belt's eight iron plates, on the ground; the conserved total unmoved once the engine's own machine item is taken out of it |
| ...and the balancer it became | **3 → 2 over four ports, 262 262 over 350 ticks, 0.00% spread**. The line ENDS on the replaced tile under the one-belt rule, so the part it becomes takes an input and gives no output |
| **reverse**: a belt laid on the middle of a nine-part column's three-part NECK | `can_fast_replace` **true**, the part gone, the belt on the tile, the part item handed back **onto the belt that replaced it** |
| ...and the registry | 14 clusters / 196 parts → **15 clusters / 195 parts, drift=0 unbuilt=0**: the column SPLIT, and the new belt is an output of the upper half and an input of the lower one. The neck is what makes the split possible under the one-belt rule -- the target's two vertical neighbours must be edgeless, or the belt would be a second belt for one of them |
| ...and the guest said so | `a belt-connectable fast-replaced the part at 0,418`, **exactly once** |
| ...and the column kept running | `[262, 262]` → `[132, 264]`, 76% — see below |
| **the refusal**: a belt over a part carrying an interface | `can_fast_replace` **false**; `create_entity` ignoring that returns nil and mines the part anyway, which the rig repairs |
| spills, ground items and `[BBB] error:` across the whole leg | **0**, **0** and **0** |

**The column delivers less after the split and that is the correct answer.** Before: two independent belts in and two out, 2.0 belts. After: the lower cluster has TWO inputs — its own belt and the one coming down from upstairs — and one output, and a balancer equalises its inputs, so it draws half a belt from each and delivers 1.0, while the upper splits its one belt between its own output and the belt feeding downstairs and delivers 0.5. 1.5 belts of 2.0. The bound is set at 70% knowing that; what may not happen is a half that STOPS.

**RED PROOF 1, the data-stage line.** Remove `fast_replaceable_group` and rebuild: all three `can_fast_replace` answers come back **false**, the rigs correctly do nothing, and the suite fails with

    FAIL: post-frep-fwd: the guest saw 14 clusters of 195 parts, expected 14 of 196

**RED PROOF 2, the guest check.** Put the line back and make `reapFastReplaced` return immediately: the belt is created, the part is gone from the world, and

    FAIL: post-frep-rev: the guest saw 14 clusters of 196 parts, expected 15 of 195

with `frepb` going on delivering its pre-split figure unchanged, because nothing was rebuilt and the belt is inert. That is the phantom, and it is what a player would be left with.

### What is interactive, and it is one gesture in two directions

The trigger is behind the same wall as the miner's pocket and the over-limit hand-back: a `--create` has no player, so nothing headless can make a CURSOR do this. `test/interactive/` stages gesture **E** for it (`make interactive-install`; [`test/interactive/README.md`](test/interactive/README.md) is the checklist), and the rig's tiles are verified headlessly so the coordinates in it are not a guess:

    part-over-belt(20,92)=true    at=[express-transport-belt]
    belt-over-part(20,96)=false   at=[bbb-balancer-part,bbb-linked-belt]
    belt-over-part(20,98)=true    at=[bbb-balancer-part]
    belt-over-part(20,100)=false  at=[bbb-balancer-part,bbb-linked-belt]

**Those tiles moved when the band was rebuilt single-edge and one of them is a different gesture now**, so re-derive them from `control.lua` rather than from memory. A part dropped into the MIDDLE of a running line takes the belt behind it as an input and the belt ahead as an output, which is two belts on one part and is refused -- so the forward half's line ENDS on the target tile and the balancer it joins becomes **3->2**, not 3->3. The reverse half's column is FIVE parts rather than four, because the belt that splits it is an edge of the part above it and of the part below it and both of those must be otherwise edgeless. An end part now carries ONE interface rather than two, which is the same refusal for the same reason.

What a human has to see: the fast-replace preview rather than a red block over a belt; the belt and its cargo arriving in the INVENTORY rather than on the ground (a script build has no player, so every headless number here is the spill arm); the reverse gesture handing the PART back the same way; and the refusal over the parts that carry interfaces. Grep for `a belt-connectable fast-replaced the part at` and for `compiled cluster … 3->2`.

### What it costs

Package built 2026-08-16, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 327,613 B | **330,778 B** | +0.97% |
| `fk_module.lua` | 2,534,887 B | **2,559,428 B** | +0.97% |
| `dist/bbb.wasm` | 1,106,413 B | 1,114,847 B | |
| members bound into the mod | 42 | **42** | of 4,257 — none added |

`LuaSurface.find_entity` was already bound (limit.go's revert uses it), so nothing new crosses the boundary and `make check` needed no re-pin.

**Nothing on any hot path moves, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB of linear memory — which is the gate a check that runs for every belt built anywhere on the map has to clear. Leg D (a belt laid 18 tiles from anything) is 32 B and leg B (a belt laid inside the neighbour gate) is 352 B, both unmoved: a `map[key]uint32` point query allocates nothing.

The `edge` suite is 4,650 → 5,850 ticks and still runs in about fifteen seconds. Its placement probe is **191 / 191 / 196 / 197 / 180 / 180, 0 off a part tile**, byte-identical to the run before this pass, plus one new `frep` sample at 192 — which is what building the rigs late buys.

<!-- ========================== END FAST REPLACE ========================== -->

<!-- BEGIN: the 2.1 single-edge rule (2026-08-24) -->

## One belt per balancer part — Factorio 2.1's rule

**The whole design is [`agents/single-edge.md`](agents/single-edge.md) and the whole implementation is `guest/go/sedge.go`; this is the working note that says where the seams are.** Phases 1, 2 and 3 shipped 2026-08-24: the rule, its refusal and its merge arm; then the runtime-global setting, the grandfather pass, the migration and the `sedge` and `mig21` suites; then the interactive and demo worlds rebuilt single-edge, with the `iact` suite gating them. **Phase 4 followed the same day with the first tranche of the rebuilt test estate -- `m2`, `mar` and `upg`** -- which is what turned this section's "nothing on any hot path moves" from a structural claim into a measured one. **Phases 5, 6 and 7 finished the estate the same day** (`mix`, `plat` and `qual`; `m3` and `edge`; `mig`), so all thirteen suites run on 2.1 in both `-gc` arms. **What is left is the GIF re-capture**, which needs a graphical client and is the user's to do.

**Why there is a rule.** Every edge of a cluster is an interface linked belt standing ON the cluster's own tile, so a 1x1 part carrying an input on one side and an output on another carried TWO belt-connectables on one tile. That was legal on 2.0 only through the collision-mask loophole spike S1 found, and 2.1 closed it: the validator now demands every belt-connectable collide with itself, probed exhaustively on 2.1.14 with no mask design passing and no runtime bypass (`create_entity` nils, `teleport` returns false). It is not an oversight upstream might undo -- boskid's answer names the invariant it protects: **belt-to-belt connections are not saved, they are re-derived at load, and one belt-connectable per tile is what makes that unambiguous.** S1's own "never two same-direction inputs on one tile" was the observed shadow of exactly that ambiguity.

**So the port is a RULE change and not an interface redesign.** A cluster tile may carry at most one edge; `classifySide`, the six types, the direction reading, the planner, the hidden network, carry, skin, the registry and the audit are all unchanged. What changes is the geometry a player builds: **a 4-in/4-out balancer is eight parts, not four**, and the smallest balancer is two.

### The two questions, and the one `if` that answers the first

| question | channel |
|---|---|
| **CAN the engine stack?** a fact about the Factorio version | the data stage emits `not_colliding_with_itself` on 2.0.x and NEVER on 2.1.x, and defines the marker prototype `bbb-can-stack` in the SAME branch. The guest reads the marker, not a version -- so its belief cannot drift from the prototype's actual capability, because there is one `if` and not two. `guest/go/data/hidden.go`, over the branch in `guest/go/engine` |
| **MAY the compiler use it?** a per-save policy | `settings.global["bbb-multi-edge-parts"]`, a **runtime-global** bool defaulting to false, which `guest/go/data/settings.go` defines on 2.0.x and never on 2.1.x, over the SAME `guest/go/engine` branch the data stage asks -- one Go function called by two exports of one module, where two Lua states had to share a required file. Read behind the same cache; the seam is the return of `multiEdgeAllowed()` and nowhere else |

**The effective rule is the AND, and the marker is the OUTER term** -- so on 2.1 the setting is never read at all, which is exactly right because it is not defined there. **The fold is `guest/go/edgemode`, a pure package**, and it is pure for a reason `plan`, `skin` and `carry` do not have: **the engine its interesting states live on is one this machine cannot run.** Multi-edge exists on 2.0 only, so the setting reading true, a player flipping it and the grandfather pass writing it are unreachable from a 2.1 headless run; written inside `main` that fold would be four branches nothing could ever execute, and as a package `make check` proves all eighteen of its states.

**Runtime-global is forced rather than preferred.** The grandfather pass has THE MOD flip the setting, and a script can write `settings.global` and can never write a startup setting (measured: `settings.startup` answers `LuaCustomTable is read only`). It also buys a flip with no restart and an ordinary replicated event instead of a whole load cycle.

**The version guard fails SAFE.** Anything `guest/go/engine` cannot read as 2.0.x is treated as 2.1, because emitting the flag on 2.1 refuses the mod at load while not emitting it on 2.0 merely costs the multi-edge geometry the guest would then refuse to build anyway.

**The capability is a cached point query** -- `prototypes.entity["bbb-can-stack"]`, two host calls once per heap and one integer compare after that, which is `legacyStubPresent` with a different name in it. The cache is a TRI-STATE and not a bool because the zero value has to mean "not asked yet", and it is thrown away by the three load hooks (`fk_on_init`, `fk_migrate`, `fk_on_configuration_changed`). The third is the one that matters: Factorio raises `on_configuration_changed` for a GAME-VERSION change as well as a mod-set one, so it is the hook that catches a 2.0 heap waking up on 2.1.

### What it costs, which is nothing on any path that was not already walking

**The per-tile count falls out of `classifyEdges`' own walk.** That walk already visits every tile and every side, so counting the sides that answered is one integer per tile: no allocation, no extra host call, which is the gate the `mar` suite holds everything on that path to. The refusal is asked in `compile()` in front of the teardown, beside `overLimitShape`'s and after it -- the port limit goes first because its answer does not depend on the mode, so a player whose balancer breaks both bounds is told about the one that is true on every engine.

**The refusal IS the sixty-fifth belt's, shared rather than copied.** `refuseAdmit` is the three-way admission both bounds go through -- the wake-race guard (a refusal issued from inside `rebuildFromWorld` logs, requeues and tells nobody), the once-per-edge-state feedback gate, and everything else -- and `tellRefusal` is the three arms (a player gets flying text and the piece back, a robot or script gets `force.print`, a player who left gets the log line). What differs is the predicate and two locale keys, which is the whole of what a bound is.

### The merge, and the bridging-tile theorem

A part that BRIDGES two clusters is queued by `AddPart` as two DEAD roots and one live one, so the check in `compile()` is useless on its own -- limit.go's `spareOverLimitMerges` is what takes those teardowns back off the queue, and it needs the same answer for this rule that it already gets for the port limit. **The port limit gets its answer from arithmetic** (`4C <= MaxPorts` is a proof no classification could find enough edges); there is no such bound here, because any two-part merge could break the rule. What replaces it:

> **A tile's edge count is over its EXTERIOR sides, and adding a part makes one side of each of its neighbours INTERIOR -- which can only take edges away. So the only tile whose count can have gone UP is the new part's own.**

`AddPart` writes that tile down (`noteAddedPart`, one append to a high-water slice truncated by every flush -- the `buildNotes` shape exactly), and the pre-pass classifies at most those tiles, at most four host calls each and usually two, because a bridging part has at least two interior sides by construction. **Nothing at all is recorded on an engine that can stack or inside a `rebuildFromWorld`**, and the classification is skipped outright unless a merge would really strand a standing network -- which is what keeps a blueprint paste and a whole-world rebuild off it.

**SOUND, NOT COMPLETE, AND SOUNDNESS IS THE REQUIREMENT.** Sparing a merge that then compiles successfully would leave both predecessors' networks standing beside the new one -- three networks over one cluster, two of them holding items nothing will ever come back for. A bridging tile carrying two edges is a refusal for certain, so "yes" is sound; a "no" that is wrong costs the OLD behaviour and never a network left standing where it should not be.

**The design's "also spare when either predecessor is already refused" is NOT taken, and that is a correction rather than an omission.** It reads as free conservatism and it is the unsound direction: a part is fast-replaceable onto a belt (`fastreplace.go`), so a bridging part can be placed ON one of the two belts that made a predecessor's tile multi-edge -- taking that tile to one edge and the merged cluster to perfectly buildable. The same argument retires it for the port bound. What IS implemented is a fall-back to the full classification when the candidate has STRANDED networks under it, where the theorem's premise (the predecessors compiled, so they were valid) is the thing that does not hold.

### A save built to the other rule — the grandfather pass and the migration

Everything above is about an EDIT: somebody asks a part for a second belt and is refused before anything is touched. This is about a whole SAVE arriving where the rule is not the one it was built under, which is a different act. **Two triggers, one mechanism, and which outcome it reaches is chosen by the capability marker.**

**One scan.** `rebuildFromWorld` classifies every cluster it inspects anyway, so "does any tile carry two belts" is a fold over a pass that was already happening -- zero extra host calls. `inspectNetwork` RETURNS it rather than leaving `sedgeWorst` for the caller to read, and that is not style: it has three early returns that never classify (an empty cluster, a surface that has gone, and the ordinary "nothing standing, this is a clean build"), and on any of them the global still holds the PREVIOUS cluster's answer.

**Two outcomes.**

| the marker says | what happens |
|---|---|
| **CAN stack** (2.0) | every multi-edge cluster is ADOPTED, because all its interfaces are still standing and the adoption comparison matches exactly. The pass then writes the anchor, writes `settings.global["bbb-multi-edge-parts"] = true`, and tells each owning force ONCE that this save keeps working, that new balancers are one belt per part, and that these will stop working on 2.1. A save with no multi-edge cluster stays at the false default and hears nothing |
| **CANNOT stack** (2.1) | the engine has already deleted all but one interface per tile, so the comparison cannot match. The remnant is CONDEMNED, torn down, and the compile refuses -- and each owning force is told once, with a clickable `[gps=x,y,surface]` per balancer |

**THE GRANDFATHER LATCH IS ONCE PER SAVE BY CONSTRUCTION, not by a flag**: the pass writes the setting it tests, so the next load reads it true and does nothing. A player who later rebuilds single-edge is never flipped back DOWN -- a silent downgrade under somebody relying on multi-edge is a trap -- and turning it off is one click of a Map setting.

**...AND TURNING IT OFF WHILE MULTI-EDGE BALANCERS ARE STILL STANDING IS VETOED, WHICH IS NOT WHAT THE DESIGN SAID.** It called that a SWEEP -- condemn every multi-edge network, spill what it held, tell the player -- and the sweep has never been able to stick. Reaching `edgemode.ActSweep` means the capability marker is present (the anchor said Multi, which requires it), so the very next thing `settleEdgeMode` asks is `GrandfatherNeeded(marker, Off, n)`, which on that path is `n > 0` -- **the same condition under which a sweep finds anything**. So the setting goes straight back on and the player gets the grandfather warning naming what to rebuild. A flip-off with nothing multi-edge standing finds `n == 0`, says nothing, and sticks, which is the only way the setting ever goes off and is the right one.

**And what the sweep did on the way was put the world on the floor.** Reported from a live 2.0.77 session on 2026-08-24 and reproduced headlessly: the condemnation was taken by the flush, the network torn down into an `owned` pool, the compile refused (the setting is still off at that instant), and `closePool` spilled everything -- and only THEN did the grandfather write the setting back and re-queue, so the networks came back EMPTY. **A vetoed flip has to be a no-op on the world**, so the scan counts and announces and touches nothing (`scanStackedMultiEdge`), and the grandfather's own re-queue puts the clusters back where each one skips on the fingerprint it never lost. **Which leaves `condemnStanding` with ONE producer, `rebuildFromWorld`** -- the 2.1 migration, where the machine really cannot exist and the engine has already pruned it. The `flip` suite is the measurement, in both directions; see [`agents/single-edge.md`](agents/single-edge.md), phase 9.

**The warning carries a `[gps=]` per balancer since the same day**, which reverses this design's own reasoning that the 2.0 message should carry none "because the player is not sent on a tour of balancers that are running": the sentence has always ended in *rebuild them one belt per part*, and naming N machines without saying where is the scavenger hunt the pings exist to end. Same cap discipline as the migration summary -- the list stops at what one readable chat line holds and the count in the sentence stays exact -- with the guest's `told force` line carrying the ping count, a truncation note and the FIRST ping verbatim, which is what makes "the pings name real cluster tiles" a measurement rather than an inference from a count.

**AND A PING LIST CHARTS WHAT IT POINTS AT.** A `[gps=]` opens the map at a coordinate whether or not the force has ever seen it, and an uncharted coordinate is BLACK -- reported from the same live session, where nineteen right answers across two surfaces all opened on fog. `tellAffected` is where all three producers arrive (the veto, the load-time grandfather and the 2.1 migration summary), so `chartAffected` hangs off the same loop and the SAME condition the ping list uses: every ping a player can click is charted, and one the cap dropped is not. What is charted is the CLUSTER'S box plus eight tiles rather than the ping's tile, because `lim` is sixty-six parts and spans more than one chunk. **The EFFECT is behind the player wall** and that is measured rather than assumed: on a headless run `force.chart`, `force.chart_all` over a fully generated surface, a radar and even nauvis's own origin chunk all leave `is_chunk_charted` false, because a force with no players has no chart to write into. So the guest logs `charted N from x,y to x,y`, three suites assert that against the ping count and the rig's own geometry, and `flip` and `mig21` carry the zero-before-zero-after tripwire that fails the day a Factorio charts headlessly. See [`agents/single-edge.md`](agents/single-edge.md), phase 10.

**THE WRITE IS GATED ON THE MARKER AS A CORRECTNESS MATTER.** Writing a `settings.global` key this engine does not define RAISES (measured on 2.1.14, `LuaCustomTable doesn't contain key`), and a 2.0 save opened on 2.1 is full of exactly the clusters the pass looks for -- so a fold that forgot the marker would raise inside the load of every save the migration exists for. `TestGrandfatherNeverWritesWhereTheKeyDoesNotExist` and the `mig21` suite's negative are the two places that is pinned.

**The anchor is not a cache of the setting.** `edgeAnchor` is guest state and therefore save state: the mode the REGISTRY was last reconciled under. The setting is what the player wants; this is what the networks standing in the world were built to, and a flip is a change only when the two disagree. That comparison is load-bearing rather than defensive -- Factorio raises `on_runtime_mod_setting_changed` for a write of the value already there, and SYNCHRONOUSLY, inside the assigning statement. So the grandfather pass writes the anchor FIRST and the setting second, and its own re-entrant handler finds agreement and does nothing. No self-write flag exists and none is needed.

**The write runs where `revertOverLimit`'s mine runs**: `flush()`, after `endCarry()`. The same three reasons -- a synchronous re-entry into the queues the drain is iterating, the package-level compile buffers, and a carry transaction that has to be closed before anything can file against it.

**THE ORDERING CARVE-OUT IS THE ONE PLACE A REFUSAL DEMOLISHES ANYTHING**, and it is deliberately not the over-limit standing-state idiom. An over-limit refusal leaves the working machine alone because the machine is fine and only the requested edit is not; here the machine itself is the thing that cannot exist, and a stacked linked belt standing in a 2.1 world is a latent engine risk on every load rather than a degraded balancer (boskid, above). So a CONDEMNED cluster is torn down first and refused afterwards. **The items need no new code**: the teardown opens an owned pool, the refused compile claims nothing, and the pool settles onto the ground, which is what this mod does with a REMOVAL's items and a machine that cannot exist any more is a removal. Only `inspectNetwork` and the 2.0 flip sweep condemn, and both INVERT the stored fingerprint first -- which is what carries a condemned cluster past the skip, since flipping a setting moves nothing in the world.

**Considered and rejected: partial service** -- compiling a degraded network that keeps one deterministically chosen edge per tile so items keep moving. Any pick is functionally arbitrary: a four-part 4x4 keeps four of its eight belts, and which four decides whether it deadlocks, starves or silently delivers wrong ratios. This mod's one promise is exact balance, and a stopped balancer that says why beats a running one that lies.

**`refused=` joined the audit line**, and it had to. A migration tears the remnant down on purpose, so those clusters end with no network -- and `drift=0 unbuilt=1` is, in this file, the SIGNATURE of a refusal that demolished first and asked afterwards. `auditAll` compares each cluster's fresh fingerprint against the one the feedback gate remembers refusing on and counts the matches; a refused cluster is never counted `unbuilt`, because `unbuilt` is this guest saying it should have built something and did not. The column is APPENDED, because this line is the assertion surface for every suite here and several match it with an unanchored pattern; the `sedge` suite's recorded tuples are unchanged to the digit, and its own refusals read `refused=1` beside `drift=1 unbuilt=0`.

### The interactive and demo worlds, which the rule changed as much as it changed the rigs

**`guest/go/obs/iact` stages both the player-gesture rigs and the mod portal's demo scenes, and every one of them is single-edge since 2026-08-24.** Two of the five gesture bands are REDESIGNS rather than re-lays, and the reason is one sentence: **under this rule a working balancer has no free face**, so a player's belt can only change its port count by landing on a part that has no belt yet.

- the shrink band stages an **attached edgeless part** for the belt to land on. Measured on the saturated rig: the belt takes P from 2 to 4 with 72 items handed back and none spilled, and mining it drains 200 with **128 that will not fit** -- the `edge` suite's `bmin` boundary, reached by a gesture rather than by a script;
- the sixty-fifth-belt band needs a **sixty-sixth part** for the same reason, or the gesture would only ever reach the single-edge bound instead of the port limit;
- the bridge band's gap tile carries **one** flanking belt and not two, for that reason inverted: a second belt would have to stand on the bridging part's own far face, so the merged shape would be illegal twice over. Measured on the `edge` suite's `brdg` rig: `compile()` asks the port bound first and returns, so the refusal still reads as a port refusal and the one-belt break is never reported -- which makes the input count in the assertion the only thing that can see it;
- the fast-replace band's line **ends** on its target tile, because a part dropped mid-line takes the belt behind it as an input and the belt ahead as an output. Its reverse column is **five** parts, because the splitting belt is an edge of the part above it and of the part below it and both must be otherwise edgeless;
- **`single-part-1-to-3-fanout` is unrepresentable** and retires: one part cannot carry four belts. The cross form is the same 1->3 read and was already a scene, so the portal goes from six scenes to five.

**The `iact` suite is what stops that world rotting**, and its assertion with teeth is that **nothing is refused at staging time**: the gestures create the refusals, and a rig that arrives already refused is a session wasted. Its section is under Verification above; the numbers, the deviations and the per-gesture probe readings are in [`agents/single-edge.md`](agents/single-edge.md)'s phase-3 status section. **The GIF re-capture is the one thing outstanding** and it needs a graphical client, so the pre-port captures stay in `docs/media/` until it lands rather than leaving the portal description with broken links.

### What runs, and what this leaves owed

`m1`, `sedge`, `mig21`, `iact`, and since 2026-08-24 `m2`, `mar` and `upg` -- the first tranche of the rebuilt estate -- and `mix`, `plat`, `qual`, `m3` and `edge`, the second the same day. **That tranche is what paid off the debt this section opened with.** Phases 1 and 2 could not run `mar`, so every "nothing on any hot path moves" claim above was structural rather than measured; the `mar` suite runs now, over rigs re-laid one belt per part, and the claims hold as measurements: the four terms that touch a compile came back at **1,209 / 3,736 / 1,280 / 2,080 B** against 1,180 / 3,736 / 1,216 / 1,712 before the port, with **the 4×4 term identical to the byte** because leg G's rig was already single-edge and did not have to move -- which makes it the control on the whole re-lay. The three terms that touch only the EVENT path -- a far belt built, a far belt mined, a belt laid inside the neighbour gate -- are byte-identical too, at 32 / 0 / 352 B, and those are the paths `noteAddedPart` and the per-tile count were added to. Linear memory over 680 operations is **3.92 MiB**, the number this repo has recorded since 2026-08-02.

**`mig` was the last and is the one suite whose rigs were deliberately NOT re-laid**: its world is the INCUMBENT'S, so re-laying it would have been re-laying the thing under test. It changes OUTCOME rather than geometry -- an incumbent balancer converts and is then refused -- and what phase 7 added is a band laid one belt per part, which converts into a network that runs at 2.000x and 3.997x one belt. Its audit is `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7`, and the one defect it found is a MESSAGE: a converted-and-refused balancer gets the ordinary per-piece sentence rather than the migration summary, unless a rebuild-from-world happens to follow the conversion. See [`agents/single-edge.md`](agents/single-edge.md)'s phase-7 section. The second tranche cost the guest nothing and moved no number it did not have to: `plat`'s stacking leg came back identical to the item, `qual` moved one rig of four, `mix` retired one assertion -- a per-output type floor that turned out to be a statement about one asymmetric network's port order rather than about the balancer -- and `m3`'s twelve rig rates and `edge`'s delivery ratios, spill quantities and placement probe are the numbers they have always been. `edge` was the one the design chartered as needing real work rather than a re-lay: four of its rigs carry a spare part or an ended belt line so that the EDIT they exist to take is still legal, and the two over-limit refusals are told apart by their output count rather than their input count now that both are sixty-five inputs. See [`agents/single-edge.md`](agents/single-edge.md)'s phase-5 status sections.

**And the 2.0 arm of phase 2 is owed with the same estate.** The grandfather write actually landing, both arms of the flip handler and `sweepStackedInterfaces` are unreachable on 2.1 by construction -- the setting is not defined, so nothing can flip it and the write is refused before it is attempted. What is pinned instead is the FOLD, exhaustively, by `go test ./edgemode/`, and the NEGATIVE, by `mig21`, which fails on any grandfather line, any failed-write alert and any setting-changed line at all. The positive legs need a 2.0 binary and belong on the `release/2.0` branch.

<!-- ========================= END SINGLE EDGE ========================= -->

<!-- BEGIN: adopting an incumbent's save (2026-08-16) -->

## Adopting a Belt Balancer 2 or 3 save — the migration

**The rule, in one sentence: while an incumbent is INSTALLED this mod never touches its entities, and once it is GONE this mod converts every `balancer-part` left standing into one of its own, once per save.** Requested 2026-08-16 in exactly those terms; `guest/go/legacy.go` and `guest/go/data/legacy.go` are the two halves and their headers are the long form.

**The incumbents are FOUR NAMES and one prototype.** `belt-balancer` (the original), `belt-balancer-performance`, `belt-balancer-2` and `belt-balancer-3` all define a `simple-entity-with-force` called **`balancer-part`** and an `item` of the same name that places it -- the original was renamed into that name by Belt Balancer 2.1.0's own `migrations/2020-02-28_Belt-balancer_2.1.0.json`. All four declare `!` conflicts against each other, so **at most one is ever active**, and their I/O model is this mod's: adjacent belts, orientation decides in from out. That is what makes adoption possible at all -- the world already says what each balancer's ports are, and `classifyEdges` re-derives them the way it does after any other edit.

### Why NOT a `migrations/*.json` rename, which is the obvious answer

**A prototype-migration file is applied ONCE PER SAVE PER FILE and the engine remembers by FILE NAME** (<https://lua-api.factorio.com/latest/auxiliary/migrations.html>). So a rename shipped by this mod would be recorded as applied on the **first** load after this mod is installed -- which, for the player this feature is for, is a load where the incumbent is still present and its balancers must not be touched -- and could never fire again on the later load where it is gone. It also has no way to express "do nothing while that other mod is installed", and what a rename does when BOTH the old and the new prototype exist is undocumented. The decision is taken at RUNTIME, from `script.active_mods` and a marker prototype, so it belongs in the guest.

### The two halves, and neither works alone

**The DATA half stops the engine deleting the evidence.** When a mod is removed, Factorio deletes every entity whose prototype went with it, **at load, before any script runs** -- so without a prototype of the same name and a compatible type there is nothing left for a guest to find. The data guest exports `fk_data_final_fixes`, and `guest/go/data/legacy.go` is that hook: it defines a stub `balancer-part` entity, a stub `balancer-part` item and a marker prototype **only when nobody else has**. `data-final-fixes` is the stage that can see that: it runs after every mod's data and data-updates stages, so an incumbent that is still installed has already defined its own and this file does nothing at all. **The "leave it alone while it is installed" half is enforced by the engine's own load order rather than by a list of mod names.**

**The RUNTIME half decides WHOSE the prototype is**, because a prototype existing says nothing about that. `guest/go/legacy.go` reads `script.active_mods` for the four names, and `prototypes.entity["bbb-legacy-stub"]` for the marker the data half defines **if and only if** it defined the stub. The marker is the guard with the blast radius: without it a mod nobody has heard of that happens to define `balancer-part` would have its entities eaten. See the red proof below, which is exactly that.

**Two things about the stub prototype are decisions rather than copies:**

- **`placeable_by = { item = "bbb-balancer-part" }` and `minable.result = "bbb-balancer-part"`**, so a player who mines a stub and a robot that revives a ghost of one both end up holding THIS mod's item. That is what makes a migrating player's **blueprint book** keep working: every blueprint they took names `balancer-part`, its ghosts ask for our item, and the part the robot builds is queued by `legacyBuilt` and swapped by the next flush.
- **`not-blueprintable` is deliberately ABSENT.** A player cannot build a stub and no flag is what stops them: nothing places that prototype, because the stub item's `place_result` is `bbb-balancer-part`, there is no recipe and no technology. That is structural and stronger than a flag -- and the flag would break the one path above in exchange for refusing a capture of a prototype that exists for at most one load. `not-upgradable` IS set: there is no upgrade path onto or off it and there must not be one.

**And the item is not renamed, which is a decision too.** A stack of the incumbent's part in a chest, a hand, a logistic request or a bot survives because the prototype survives, and its `place_result` is this mod's part -- so a legacy stack simply places this mod's balancers. Walking every inventory in the game to rewrite stacks would be a scan of the whole world for a cosmetic difference in what a stack is called.

**The locale entry for `balancer-part` is the INCUMBENT'S OWN ENGLISH TEXT, verbatim** ("Balancer Part"), and only in `en`. This mod loads after both Belt Balancer 2 and 3, so a different string would rename a working mod's entity in its own player's UI; identical text makes the override a no-op, and shipping no non-English file leaves every translation they have winning by fallback.

### The state machine, and the re-arm is why it is a machine and not a bool

`legacyPhase` lives in the guest heap and therefore in the save. Its zero value is `legacyUnchecked`, which is what a fresh heap must mean.

| state | what it means |
|---|---|
| `legacyUnchecked` | nothing decided on this heap. The gate every event passes is one compare against this |
| `legacyDone` | this game's `balancer-part` is ours (or there is none) and the scan has run |
| `legacyBlocked` | an incumbent is active, or a STRANGER owns `balancer-part`, or the mod list could not be read. Nothing is converted and nothing is touched, **including by the build path** |

| trigger | what it does | reported as |
|---|---|---|
| `fk_on_init` | a new save, **or this mod added to one that already exists** -- the common half of the swap | `trigger=init` |
| **`fk_on_configuration_changed`** | the MOD SET moved: a neighbour added, **removed**, or re-versioned. **It also fires on the load that ADDS this mod, right after `fk_on_init`** (a newly added mod is itself a mod-set change; measured in the `mig` suite's `added` leg, where the guest's `rebuilt from world` line lands before tick 0), so that load decides twice: `init` converts, and the second decision finds nothing and says nothing -- one extra by-name scan per surface, once | `trigger=configuration_changed` |
| `fk_migrate` | a rebuilt guest, on a fresh heap | `trigger=migrate` |
| `onEventBody`, after `ensureRegistry` | a fresh heap that reached an event before any hook | `trigger=first-dispatch` |
| the tail of `fk_on_deferred` | re-tests a **Blocked** state only | `trigger=deferred` |
| the top of `flush()` | not a decision at all: it swaps the stubs a BUILD EVENT queued | one line per stub |
| `auditAll` | the same, and it is what makes `/bbb-audit` a door onto the conversion | `trigger=audit` |

**The re-arm is the case a bool cannot serve.** A player can install this mod first, install the incumbent second, build fifty balancers with it, and remove it a month later; a save that had latched "scan done" would never look again. `fk_on_configuration_changed` is the trigger that case deserves and it is upstream's, landed 2026-08-16 for this shape ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 22): it is exactly the event that reports a neighbour being uninstalled, it is REPLICATED (it runs on the peer that loaded, before the first tick, so the conversion is already inside the state a joining client downloads), and it fires AFTER `fk_migrate` on a load that is both. It is the opposite side of the peer-local rule from `fk_after_load`, which this mod must never export.

**The deferred-flush re-arm behind it is belt and braces and is kept because it is free.** The test there is the MARKER PROTOTYPE and not `script.active_mods`: `prototypes.entity` is a `LuaCustomTable`, so the raw handle plus its index operator is a POINT query returning an Object or nil -- **two host calls and zero allocation**, where reading the mod list allocates a Go string per mod. `script.active_mods` is reached once per decision, never from a re-test, and it earns its place by NAMING the mod in the log line.

**Both gates were measured before either was written**, on 2.0.77 rather than assumed, because the design turned on them:

| probe | result |
|---|---|
| `script.on_event(id, f, {{filter="name", name=<a prototype that does not exist>}})` | **accepted**, no error. So `balancer-part` is in this mod's ordinary subscription filter unconditionally, and there is no branch on the mod set at subscribe time |
| `prototypes.entity[<a name that does not exist>]` | **nil**, and raises nothing. So the marker test is a value read rather than a guarded call |
| `find_entities_filtered{name=<a name that does not exist>}` | **RAISES**. Which is why the test mod guards its census and why the scan runs only when the marker says the prototype is ours |
| `mods[...]` at runtime | **nil** -- it is a data-stage global. `script.active_mods` is the runtime form and it carries versions |
| `find_entities_filtered` and `technologies[...].researched = true` from `on_init` | both legal |

### What the conversion does per part

Read position, force index, **quality** (as the prototype HANDLE, since `QualityID` is `string or LuaQualityPrototype` -- one host call and no string copied into the guest heap) and **health**; `destroy` with **no** `raise_destroy`; `create_entity` with the same position, force and quality; copy the health back; and `AddPart` straight into the registry, which is the same call a build event makes. So the clusters form, merge and queue themselves through the one code path and the flush restyles and compiles them exactly as it would after a blueprint paste.

**Read everything before destroying anything**, because after the destroy there is nothing left to ask -- and the order is forced the other way round by the collision box, since both prototypes collide on `object` and `transport_belt`.

**Health is preserved rather than reset**, at two host calls per part on a once-per-save path, because both prototypes carry `max_health = 170` and the alternative is silently repairing a building the player did not ask us to touch.

**Surfaces in INDEX ORDER**, through the same `collectSurfaces` `rebuildFromWorld` uses -- factored out of it for this, which is the one shared edit this pass makes to `lifecycle.go`. Registration order decides node ids, node ids decide cluster roots, roots decide hidden-surface slots: two clients walking surfaces in two orders would place one network in two different slots, which is a desync. `legacyIncumbents` is walked in ITS OWN order too, because `ActiveMods` is a `pairs()` walk over a Lua hash table.

**And the scan FLUSHES before it returns.** The ordinary build path defers to the next tick and the triggers that matter here cannot wait for one: a `--create` never reaches a tick at all, and `fk_on_init` on a save this mod was just added to is exactly that case. So it ends where the audit marker ends, with the networks built inside the same dispatch.

**The technology, per force that owned a converted part.** The incumbent's three technologies die with it, and a player left holding fifty balancers and no way to craft a replacement part would have been given a worse save than they started with. A point query again, and deliberately silent about a force that already has it and about a game where an overhaul mod removed it.

### What survives and what does not

| | |
|---|---|
| the balancers | as this mod's, at the same tiles, the same force, the same quality and the same health, compiled into networks on the load that adopts them |
| every item ON THE BELTS | untouched: those belts are vanilla and nothing here reads or writes them |
| every input and output | re-derived from the world by `classifyEdges`, so the ports are the ports the belts already implied |
| a stack of the incumbent's ITEM | survives, keeps its name, and places this mod's parts |
| a BLUEPRINT BOOK of the incumbent's balancers | keeps working: ghosts ask for this mod's item, and each part a robot builds is swapped by the next flush |
| the ability to craft | any force that owned a balancer is given `bbb-balancer` |
| **the incumbent's own item BUFFER** | **gone, and there is no mechanism that could get it back** |

**The buffer, precisely, because "some items are lost" is not good enough.** `objects/balancer.lua` in Belt Balancer 2 takes items OFF the belts with `lane.remove_item(item)` and holds them in `balancer.buffer`, a plain Lua array in the mod's own `storage`, refilled each tick to `output_lane_count * 2`. So a 4-output balancer holds up to **16 items** that are not in the world at all, and Factorio deletes a removed mod's `storage` **with the mod**, before any script of ours could read it. A handful of items per balancer, stated in the README rather than glossed.

### The `mig` suite — seven legs, two name probes, and two axes

The **ninth** suite, and the only one whose two phases run under **different mod sets**: `test/run.sh`'s `stage_mig` and its `BETWEEN` hooks rewrite `mod-list.json` and add or delete a mod DIRECTORY between `--create` and `--benchmark` (a directory that is present but not listed is added back by Factorio as enabled, so "removed" has to mean both).

**The suite covers two axes and neither is a subset of the other**: WHICH MOD owns `balancer-part`, and WHICH TRANSITION of the state machine above the load makes. Until 2026-08-20 the first axis was one name of five and the second was one transition of several — only `belt-balancer-2` had ever been in front of the guest, and only `Blocked -> Done` by removal was ever driven.

| | the legs |
|---|---|
| **which mod** | `belt-balancer-2` (legs 1, 2, 5), `belt-balancer-3` (leg 3), `belt-balancer` and `belt-balancer-performance` (the two probes), a STRANGER who is none of them (legs 6, 7), and nobody at all (leg 4) |
| **which transition** | `Unchecked -> Done` on a new save (1), `Blocked -> Done` when an incumbent is removed (2, 3), Done with nothing to find and the build path doing the work (4), **`Done -> Blocked` when an incumbent ARRIVES (5)**, Blocked and staying Blocked (6), **Blocked -> Done when the STRANGER is removed (7)** |

`guest/go/obs/bb2data` is a **DATA-STAGE-ONLY stand-in** under the real mod's own name and version, so `script.active_mods` sees what it would really see. It has no control stage AT ALL -- not an inert one, none: `fklua mod --data-module` with no control positional, which is what it always wanted and could not have until 2026-08-25 ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 26). That is deliberate rather than incidental: the real mod's runtime is the one thing the migration cannot recover, so a stand-in that balanced would be modelling it. None of its art is used.

**IT IS STAGED UNDER ALL FOUR INCUMBENT NAMES AND THERE IS ONE COPY OF IT.** What differs between the four rows of `legacyIncumbents` is the NAME and nothing else — the prototypes are the same prototypes — so `mig_standin` copies the one directory and rewrites `info.json`'s `name` and `version` at staging time, into a directory named for the target mod (Factorio requires that). The rewrite is checked by two greps, because a silently unrenamed copy would stage belt-balancer-2 under every name and pass every leg.

`guest/go/obs/mig` -- a compiled observer since 2026-08-25, `test/mods/bbb-mig-test/control.lua` before that -- is present in BOTH phases and builds everything in `fk_on_init` with `balancer-part`, which in phase one is the only balancer prototype in the game. Six rigs, and the last three are about what the conversion CARRIES rather than about a rate:

| rig | what it is |
|---|---|
| `ctrl` | a bare express belt, the yardstick |
| `m4x4` | 4 parts, 4 in, 4 out, saturated -- the shape a migrating player is most likely to have |
| `m3to5` | P=8 with loopbacks. Adoption re-derives the edge list, and an asymmetric shape is where a wrong one reads as a rate rather than as a crash |
| `wit` | **the conservation witness**: 2 parts, 2 in and 2 out, **no source and no sink at all**, its belts hand-loaded with **COPPER PLATE** while every other rig runs iron. A copper count across every surface is therefore exactly that rig's contents, before the swap and after it, which is what makes "the items on the belts survived" an equality rather than an estimate |
| `fid` | **the fidelity pair**: one part DAMAGED to 85 of 170 and one built at UNCOMMON quality. Those are the only two properties `legacyConvertOne` reads off the old entity and writes onto the new one, and both are invisible on an undamaged normal-quality part |
| `frc` | **the force column**: four parts in one column, the top two on the player force and the bottom two on a second force, TOUCHING. Two forces' parts touching are two balancers |
| **`sok2`, `sok4`** | **THE SINGLE-EDGE BAND, added 2026-08-24**: the same balancer laid TWO COLUMNS wide -- inputs down the west column, outputs down the east, one belt per part -- at 2 in / 2 out over four parts and 4 in / 4 out over eight. A shape a Belt Balancer user could genuinely have, and the only rigs in this suite that Factorio 2.1 can build |

Plus **a second surface**, `bbb-mig-b`, carrying two more parts and their belts; and a steel chest holding 50 of the incumbent's item.

**Every rig this suite adds carries a belt in and a belt out per part, fed or not**, and that is a constraint rather than decoration: a cluster with no inputs or no outputs compiles to nothing, which is a legitimate half-built state, and a cluster the classifier never saw would then be invisible.

**EVERY RIG BUT THE `sok` BAND IS LAID THE INCUMBENT'S WAY AND THAT IS DELIBERATE.** One column of parts with a belt on both free faces of each row is two belts per part, which is what Belt Balancer builds and what Factorio 2.1 forbids -- so on 2.1 all seven of those clusters convert and are then REFUSED. This is the one suite in the estate whose rigs were NOT re-laid for the port, because its world is somebody else's and re-laying it would have been re-laying the thing under test. See the `mig` entry under Verification for the split outcome and its numbers.

**Thirty-one parts, nine clusters, three surfaces, two forces**, and the whole suite is measured against that world.

**The three conversion legs, measured** — re-measured 2026-08-24 against the eight-rig single-edge-band world on Factorio **2.1.14**, base plus the `quality` mod, shipped configuration. **Every conversion row below is what it always was and only the OUTCOME rows moved**, which is the whole finding of the port: `legacy.go` was not touched: 

| | leg 1 `added` | leg 2 `later` | leg 3 `bb3` |
|---|---|---|---|
| the incumbent | `belt-balancer-2 2.0.9` | the same | **`belt-balancer-3 1.0.1`** |
| phase one | the incumbent alone, this mod absent | the incumbent AND this mod | the incumbent AND this mod |
| between the phases | incumbent out, this mod in | incumbent out | incumbent out |
| coexistence in phase one | n/a | **exactly one** `belt-balancer-2 2.0.9 is active; its balancers are left alone`, **0 converted** | **exactly one**, and it NAMES `belt-balancer-3 1.0.1`, **0 converted** |
| parts adopted | **31 from 3 surfaces into 9 clusters** | the same | the same |
| trigger | **`init`** | **`configuration_changed`** | **`configuration_changed`** |
| forces given the technology | **2** | 2 | 2 |
| census, before -> after | 31 / 0 -> **0 / 31** | the same | the same |
| the witness's copper | **48 -> 48 -> 48 -> 48** over four samples | the same | the same |
| the item stack | **50 held**, `place_result` `balancer-part` -> `bbb-balancer-part` | the same | the same |
| technology after | `bbb-balancer=true`, `belt-balancer-1=absent` | the same | the same |
| `ctrl` over t=1800..3540 | 1306 items | 1306 | 1306 |
| **`sok2`** | **1306 1306, 2.000x**, spread **0.00%** | identical | identical |
| **`sok4`** | **1304 1306 1306 1304, 3.997x**, spread **0.15%** | identical | identical |
| `m4x4`, `m3to5` | **0 0 0 0 and 0 0 0 0 0** -- refused, and a refused cluster has no network | identical | identical |
| refused | **7 clusters**, parts-carrying-two-belts **[2, 2, 2, 2, 2, 3, 4]** | identical | identical |
| the fidelity pair | **85.0 of 170.0 health** and quality **uncommon**, both on a `bbb-balancer-part` at t1 and again at final | the same | the same |
| parts per force | **player=29, bbb-mig-force-b=2**, and the second force's `bbb-balancer` **researched** | the same | the same |
| where they are | `nauvis:0/0 bbb-mig-a:0/29 bbb-hidden:0/0 bbb-mig-b:0/2` -- nothing left as the incumbent's anywhere, on either surface that had any | the same | the same |
| the late build | `legacy=0 ours=1` | the same | the same |
| final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` | the same | the same |

**The trigger word on the summary line is an assertion, not decoration.** Leg 1 must be driven by `init` and legs 2 and 3 by `configuration_changed`; a leg that came out `first-dispatch` or `deferred` fails, because **a feature whose fallback silently does its primary trigger's work passes every test and ships broken.** That is the same thing `upg` asserts about which trigger drove a rebuild-from-world, and it is asserted here for the same reason. **Leg 2's ONE blocked line is the latch**: its create phase places an audit marker, which is a door onto the re-arm, and a Blocked state that re-decided every time it was asked would say so twice.

**THE RATES ARE THE `sok` BAND'S NOW, AND THE OTHERS ARE ZERO.** An adopted balancer laid one belt per part delivers exactly what a built one does, against a control belt in the same save; one laid the incumbent's way is refused and delivers nothing at all, which is the honest outcome rather than a regression. And in leg 1 `rebuildFromWorld` runs AFTER the conversion and **adopts the two working clusters beside the seven it refuses** -- `9 clusters, 2 adopted, 7 rebuilt` -- which is the only place in the estate a single-edge cluster is adopted next to refused ones.

**Leg 3, `bb3`, is the LIVE SUCCESSOR, and it is the coexistence shape rather than the swap shape on purpose.** Belt Balancer 3 is the one a player is most likely to be migrating off today and the one name in `legacyIncumbents` a typo would be most expensive in — and the reason the leg has to have both mods installed in phase one is that **a name this guest does not recognise does not fail loudly.** It falls through `legacyIncumbentActive` to the STRANGER branch, which is Blocked and **silent**, and then the removal load converts everything exactly as it should. Every conversion number in the column above is what a misspelled `belt-balancer-3` produces too. The NAMED blocked line from phase one is the only observable in this suite that can see a row of that list, which is what the leg is for and all it adds over leg 2.

**Leg 4, `built`, is the BUILD path and the PLAIN RELOAD.** No incumbent is ever installed, so `balancer-part` is this mod's own stub from the first byte and the observer's nineteen parts arrive one at a time through build events -- which is the path an old blueprint's ghosts take, minus the robot. Measured: **the whole-world scan runs and finds nothing** (it is silent, as it is on every other save this mod has ever been benchmarked on), **31 parts swapped after their own build events**, the `sok` band delivering 2.000x and 3.997x while the seven incumbent-idiom clusters are refused and deliver nothing, and the witness at 48 throughout. **This is the leg that found the quality defect below**: all of them is what it reports now and one short is what it reported the day the rig was written. It is also **the only leg whose save is written AFTER a conversion** and whose second phase changes no mod at all, so it is the only place the once-per-save flag can be seen surviving a save: on the reload the guest **says nothing about the migration at all**, which is asserted as the absence of any `[BBB] legacy:` line in the benchmark log.

**Leg 5, `readd`, is `Done -> Blocked` — the transition the hook exists for and the one nothing drove.** Its phase one is leg 4's: no incumbent, our own stub, thirty-one parts swapped through the build path, and the save written with the phase **Done**. Then `belt-balancer-2 2.0.9` is INSTALLED beside us. This is not an exotic order — a player installs this mod, uses it, and later installs a Belt Balancer to compare — and a save that had latched "scan done" would go on converting. Measured 2026-08-20:

| | measured |
|---|---|
| phase one | **31 parts swapped through the build path**, no scan summary line, no blocked line |
| the incumbent arrives | **exactly one** `belt-balancer-2 2.0.9 is active; its balancers are left alone`, and **0 converted** in phase two — no scan line and no build-path line |
| the world | `balancer-part=0 bbb-balancer-part=31` at **t1, post-audit and final**: the balancers this mod already owns do not move in either direction |
| the witness's copper | **48 -> 48 -> 48 -> 48** |
| the item stack | **50 held**, `place_result` **`bbb-balancer-part` -> `balancer-part`** — the other way round from every other leg, and correct: our stub owned the name while nobody else did, and the incumbent owns it again now |
| the technology | `bbb-balancer` **false -> true** (granted by phase one's own conversion, and an incumbent arriving takes nothing away) and `belt-balancer-1` **absent -> false** — PRESENT and unresearched, where every other leg requires it absent |
| the standing networks | `ctrl` 1306, **`sok2` 1306 1306 at 2.000x / 0.00% and `sok4` 1306 1304 1306 1304 at 3.997x / 0.15%** — unmoved across the incumbent's arrival. The seven refused clusters stay refused and deliver nothing, which is also unmoved |
| the fidelity pair | **85.0 of 170.0** and **uncommon**, on a `bbb-balancer-part` — here they crossed a SAVE as well as a swap, the conversion having happened in phase one |
| final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` |
| **the late build, which is the one with teeth** | it places the **INCUMBENT'S** `balancer-part` now, and comes out **`legacy=1 ours=0`** |

**That last row is the whole reason the leg exists.** `legacyBuilt` is gated on the phase being Done, and a gate reading the wrong phase does not crash, does not lose an item and does not move a rate: it **silently swaps a working mod's freshly built entity out from under it**. Nothing else in this suite can see it.

**Leg 6, `foreign`, is the stranger.** `guest/go/obs/foreigndata` defines `balancer-part` exactly as the incumbents do under a name this mod has never heard of, and it STAYS installed while this mod arrives beside it. Measured: **0 converted**, census `31 / 0` unmoved at every sample, the stranger's item still placing the stranger's entity, and the audit at **`clusters=0 parts=0 nets=0 drift=0 unbuilt=0 refused=0`** -- this mod owns nothing at all in that save, which since the port includes REFUSING nothing: a refusal there would mean a stranger's balancer had been converted and then declined. Its damaged part is still at **85.0 of 170.0**, still **uncommon**, and still a `balancer-part`; the second force's `bbb-balancer` is **false**, because a grant there would mean something of the stranger's had been converted.

**Leg 7, `fgone`, is the stranger UNINSTALLED**, which `legacyCheck` promises in as many words — *"the stranger can be uninstalled too, and on that load the stub appears and their balancers become ours, which is the same promise the incumbents get"* — and which nothing tested until it existed. It is not leg 6 with a different hook, and the difference is **when this mod is installed**: leg 6 has no guest at all in phase one, so the only thing it can watch a stranger's entities do is stand still. Here both mods are installed from the first byte, so the observer BUILDS nineteen of the stranger's `balancer-part` entities with a guest watching — and the guest must not touch one of them. Measured 2026-08-20:

| | measured |
|---|---|
| phase one, with the stranger installed | **ZERO blocked lines** (the stranger branch is silent by design, and a line here would mean the guest had decided bbb-mig-foreign is a Belt Balancer), **0 converted by the scan** and **0 swapped through the build path** over thirty-one build events |
| phase two, the stranger gone | **31 parts from 3 surfaces into 9 clusters, 2 forces, trigger=`configuration_changed`** |
| census | 31 / 0 -> **0 / 31** |
| the witness's copper | **48 -> 48 -> 48 -> 48** |
| the item stack | **50 held**, `place_result` `balancer-part` -> `bbb-balancer-part` |
| the technology | `bbb-balancer` **false -> true**, `belt-balancer-1` **absent in both phases** |
| rates | `ctrl` 1306, **`sok2` 2.000x** at 0.00% and **`sok4` 3.997x** at 0.15%; the seven refused clusters deliver nothing |
| the late build | `legacy=0 ours=1` |
| final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` |

**The technology check in this leg is its own, and that is not tidiness.** `belt-balancer-1` is absent in BOTH phases here — the stranger never defined one — so the assertion every other conversion leg makes (*the incumbent's technology is gone*) says nothing at all and would pass on a guest that granted nothing. What IS a statement is `bbb-balancer` going **false -> true**: unresearched while the stranger stood, researched by the conversion that followed it out.

**The two NAME PROBES, `belt-balancer` and `belt-balancer-performance`.** One `--create` each and nothing else. What a full leg would add over leg 2 is nothing — the conversion side of the feature is identical whichever name blocked it — and what it would cost is a benchmark phase, so the probe asserts the one thing that is not identical: the named blocked line, over a world that really does contain nineteen balancers the guest declined to touch. Measured 2026-08-24: **`belt-balancer 3.4.4`** and **`belt-balancer-performance 1.0.5`**, one blocked line each naming exactly that, **31 standing and 0 of ours**, **0 converted** and **0 refused**. The two versions are the harness's and are plausible rather than real; what they pin is that the guest read them back out of `script.active_mods`, not that any release carries them.

**And every leg ends with a LATE BUILD**, which is the probe that found the one defect review missed. One `balancer-part` is placed by script in phase two, well clear of every rig and after the final audit so nothing else moves, and what it separates is "this game's `balancer-part` is MINE" from "somebody else's": the four conversion legs come out **`legacy=0 ours=1`** and the two legs where somebody else owns the name -- the stranger, and the incumbent that arrived after us -- come out **`legacy=1 ours=0`**. The scan alone could not tell them apart, because the scan is gated on the marker either way; the BUILD path is gated on the PHASE, and the first cut of `legacyCheck` put the stranger case in `Done` -- which is the guest saying the name is its own. It says `Blocked` now, and Blocked also gets the marker re-test, which is what leg 7 is about.

**What the suite costs**: 12.7 s for the original four legs, 27.0 s for seven legs and two probes, 30.6 s once the world grew to nineteen parts on three surfaces, and **32.0 s** for the thirty-one-part world the single-edge band made, all on the same machine. The probes are ~1.5 s each because they stop after `--create`.

Re-measured on the review gate, back to back on one machine and against the SAME `dist/`, which is the only comparison worth having: **12.4 s** for master's four legs and **29.1 s** for the seven and two here. Master's four are green against this branch's guest, which is the expected result and is worth stating -- the quality fix can only help a leg that never built a quality part.

**And it is the one base suite that is not base-only.** `mig_list` enables the **`quality`** mod, in BOTH phases of every leg, because `legacyConvertOne` passes the old entity's quality through to the new one and a game with only `normal` in it cannot tell a guest that carries it apart from one that drops the key. A constant across the two phases is not a mod-set change, so no leg's trigger moves; `quality` depends on `base` alone, so this stays base-plus-one rather than becoming a second Space Age run.

### What the conversion CARRIES, and where the parts are

Three things `legacy.go` claims in as many words and nothing measured until 2026-08-20: that a converted part keeps its **health** and its **quality**, that the technology is granted **per force**, and that the scan walks **every surface**. Every part in every leg was an undamaged, normal-quality, player-force part on one surface, so all four claims were satisfied by a guest that did none of them.

| what | how it is measured | measured |
|---|---|---|
| **health** | the `fid` rig damages one part to 85 of 170 in `on_init` and the observer reports the value it reads back, in phase one and again after the swap | **85.0 -> 85.0**, on a `bbb-balancer-part`, at t1 and at final, in every conversion leg |
| **quality** | the other `fid` part is CREATED at `uncommon`, which needs the quality mod, which is why `mig_list` enables it | **uncommon -> uncommon** |
| **per-force technology** | the `frc` rig puts two parts on a second force; the summary line's own force count, plus a per-force technology line for that force | **2 forces given the balancer technology**, and `bbb-mig-force-b`'s `bbb-balancer` **researched** |
| **two forces' parts touching are two balancers** | the audit's cluster count, against **the number written down in `assert-mig.py`** rather than against the guest's own summary | **9 clusters** out of 31 parts; a fusion reads as 8 |
| **every surface was scanned** | the summary line's surface count against the number of non-hidden surfaces the observer can see | **3 and 3** |
| **parts on more than one surface were really converted** | the per-surface census | `bbb-mig-a:0/29 bbb-mig-b:0/2` -- nothing left as the incumbent's on either |

**The anti-vacuity guards are the point of three of those rows.** A part that was never damaged sits at `max_health`, and an equality across the swap is then satisfied by a guest that copies nothing -- so phase one must report a value BELOW the maximum or the leg fails as vacuous. A game with only `normal` quality in it cannot tell a carried quality from a dropped key -- so phase one must report `uncommon`. And a fusion check over one force says nothing -- so the second force must own exactly the two parts the rig builds, at every sample.

### A CHECK THAT SKIPS IS A CHECK THAT PASSED -- the review gate, 2026-08-20

**Three of this suite's shared checks read `if got is None: continue`, and a phase the observer stopped reporting therefore took its whole assertion with it, silently, while the leg went on printing the phases that were still there and exiting zero.** Found by reviewing the coverage pass rather than by a run, and closed the same day; two of the three are the shape the original inline code had, and the third arrived with the fidelity rig.

**It is a measurement, not a worry.** Delete three lines from the observer's tick-1 handler -- `report_counts("t1")`, `report_item("t1")`, `report_fidelity("t1")` -- and the pre-fix script prints

    witness: 48 copper plates in phase one
    witness: 48 at phase=post-audit
    witness: 48 at phase=final
    fidelity in phase one: 85.0 of 170.0 health, quality uncommon
    fidelity at phase=final  bbb-balancer-part at 85.0 of 170.0 health
    ...
    the incumbent's balancers were adopted and they balance

and **exits 0**, on a run in which the copper count at the one sample taken straight after the load was never made, the item stack was never compared with itself at all, and the health and the quality were never checked across the swap. Every one of those lines is written unconditionally by the observer at every phase named, so an absent one is a broken harness rather than a legitimate shape, and the run that says so is worth more than the run that does not. All three fail now, by name and by phase, and so does the `foreign` leg's own inline item check.

**And the same review found the probe's success message telling a lie.** The two name probes asserted the SCAN had converted nothing (`if adopted`) and said nothing about the BUILD path -- and those two names are in front of the guest nowhere else in this repo, while the observer builds nineteen of the incumbent's entities with the guest listening. With `legacyBuilt`'s phase gate removed the probe printed *"belt-balancer is recognised by name and its balancers were left alone"* and exited 0 over a create log carrying **nineteen** `legacy: adopted a balancer-part built at` lines. One line of assertion, and the more expensive of the two failures is watched on the whole name list rather than on half of it.

**The pattern to carry forward**, because this suite is entirely built of log lines and every future check here will be too: a missing line and a wrong line are not the same failure, and only one of them fails on its own. Assert the presence, then assert the value.

**THE CLUSTER COUNT IS THE ONE NUMBER THAT HAD TO STOP COMING FROM THE GUEST.** `check_audit` compared the audit's cluster count against the count on the guest's own summary line, and both come out of the same flood fill -- so a fill that fused two forces' touching parts moves them together and neither says anything, while every census, every copper count and every rate stays exactly what it was. The expected count is a constant in `assert-mig.py` now, derived from the rigs the observer builds, and the summary line is checked against it as well.

**AND THE SUMMARY LINE'S SURFACE NUMBER COUNTS SURFACES SCANNED, NOT SURFACES THAT HAD PARTS.** `legacyScan` increments it once per non-hidden surface before it looks at one, so `adopted 31 parts from 3 surfaces` on a save whose parts are all on two of them is the literal truth and reads as something else. It is left as it is and the reading is written down here rather than changed, because the line is the assertion surface for this whole suite and its format is read by `assert-mig.py` and quoted a dozen times in this file. What the pass added instead is the observer's own **per-surface census**, which is the only thing that can see a scan that visited a surface and converted nothing on it -- red-proven below, and the two statements fail differently.

### The quality nobody had ever built — what the fidelity rig found

**A `balancer-part` standing at any quality but `normal` was invisible to the migration's build path, and stood there unconverted, unregistered and unlogged for the rest of the save.** Found on the first run of the `fid` rig, 2026-08-20, before a line of the assertion had been written: leg 4 swapped **18 of 19** parts and the one it missed was the uncommon one.

**`LuaSurface.find_entity` takes an `EntityWithQualityID`, and the pinned runtime API says of a bare name that "Normal quality will be used".** So `find_entity("balancer-part", p)` is a query for a NORMAL-quality part at `p` and nothing else. Measured on 2.0.77 with a scratch mod, against a real uncommon entity at a known position:

| | result |
|---|---|
| `find_entity("iron-chest", p)`, normal chest at p | the object |
| `find_entity("iron-chest", p)`, **uncommon** chest at p | **nil** |
| `find_entity({name = "iron-chest", quality = "uncommon"}, p)` | the object |
| `find_entities_filtered{name = "iron-chest", position = p}` | **1**, whatever the quality |

**The whole-world SCAN never had it**, which is why six of the nine legs were green over it: `legacyConvertOn` filters on the name alone through `findByNameAll`, and the engine returns every quality. Only `legacyRunBuilds` -- the path that re-finds a stub one tick after its build event -- asked the quality-scoped question, and that path is **the blueprint book's**: a migrating player's ghosts revive as stubs and are swapped a tick later. An uncommon balancer-part in an old blueprint would have revived and stayed a stub.

The fix is that call, and it is `setSearchBox` + `findByName` -- the idiom `registerPartsIn` already uses. One slice per stub built, on a once-per-save path.

**FOUR MORE CALL SITES PASS A BARE NAME TO `find_entity` AND ARE NOT FIXED HERE**, because none of them is this pass's subject, none can be red-proven by this suite, and the cheap repair for the two that matter is not cheap:

| call site | what a non-normal-quality part does to it |
|---|---|
| `skin.go`, `restyle` | the part is never found, so `graphics_variation` is never set and an uncommon balancer draws **cell 1, the lone-part picture, forever** -- and because `pvar` is written only after the engine takes the value, that part is RETRIED on every restyle of its cluster, which is one host call per flush that touches it rather than the zero the M5 budget is priced at. The mechanism's whole budget is *one byte per part*, and the two repairs are a second byte (the quality) or an allocating area query on the flush path -- and the `mar` suite asserts those slopes to the byte. **A design question, not a one-line fix**. The `mig` save has carried one such part since the `fid` rig existed, so the cost is being paid in this repo today |
| `limit.go`, `forceOfCluster` | the cluster's force cannot be resolved, so an over-limit refusal on a balancer of uncommon parts is logged and **nobody is told** |
| `limit.go`, `revertOne` | the over-limit piece is not found, so it is **not handed back** -- the negative the `edge` suite asserts (zero hand-backs) is unmoved, because a headless run has no player |
| `fastreplace.go`, `reapFastReplaced` | it reads a standing uncommon part as GONE and **unregisters a part that is still there**. It needs a foreign belt-connectable to APPEAR on a part's tile, which in ordinary play only a fast replace does -- and a fast replace really did remove it, so the answer is right for the wrong reason on the gesture, and wrong for a script that builds a colliding belt |

None of the four is reachable by any rig in this repo, because nothing outside `mig` builds a quality part at all. Fixing them is a pass of its own and it starts with a rig, not with a call site.

> **ALL FOUR ARE FIXED SINCE 2026-08-20**, by exactly that pass: `findOnTile` (guest/go/findpart.go) is the fix stated once, the `qual` suite is the rig it starts with, and "A part at uncommon quality is a part" below is the write-up -- including the answer to the `restyle` design question, which turned out to cost nothing the `mar` suite can see. The table above is kept as the record of what each site did while it stood.

### The build path DEFERS, and that is a correction the harness forced

The obvious version of `legacyBuilt` converts in place, inside the build event. It works, and **`create_entity` then returns NIL to whoever placed the entity**, because the entity was destroyed during the event it raised. Measured within minutes of writing it: the harness's own `create_entity{name = "balancer-part", raise_built = true}` came back nil and the test mod raised on it -- which is exactly what another mod scripting a `balancer-part` into the world would do.

So the event half writes down **one tile** and asks for a flush, and `legacyRunBuilds` at the top of `flush()` re-finds the stub and swaps it. The stub stands for one tick, which is invisible (it draws this mod's own lone-part picture) and is the same latency every ordinary part placement already has. It is also where this guest does everything else that reads the world, and it means the `AddPart` lands in the drain that is about to compile it -- including the synchronous drain a `bbb-audit` marker forces, which is the only one a `--create` ever reaches.

### Red-proven seventeen times, and every proof catches a different thing

The first three are 2026-08-16 and are about the feature; the next four are 2026-08-20 and are about the three legs and two probes added that day; the next seven are the same day's fidelity pass; the last three are the same day's REVIEW gate, and they are the odd ones out -- the defect they inject is in the HARNESS rather than in the mod, because what they prove is that three checks which had never been able to fail now can. Every one is an injected defect, built, run, and reverted.

**The first seven rows were measured against the FOUR-RIG, ELEVEN-PART world and are kept as measured, and the rest against the NINETEEN-PART one.** The suite builds thirty-one parts on three surfaces now, and it runs on Factorio 2.1 where seven of its nine clusters are refused -- so the counts inside every row below are the world of the day they were taken and not today's. Which assertion fires is the claim; the count beside it is the evidence that was taken for it. **None of the seventeen retires under the port**, because every one of them is about the CONVERSION and the port did not touch `legacy.go`; five were re-derived against the reworked suite on 2026-08-24 and they are in [`agents/single-edge.md`](agents/single-edge.md)'s phase-7 section.

| injected defect | what came out |
|---|---|
| **`data-final-fixes.lua`'s require commented out** -- no stub prototype | phase two's census is `balancer-part=0 bbb-balancer-part=0`: **all 11 entities deleted by the engine at load**, and the 50-item stack with them (`held=0 place_result=nil`). The suite fails on "nothing was adopted at all". **The witness's 48 copper plates survive**, which is the honest detail: the belts are vanilla, so what the missing stub loses is the machine and not the goods |
| **`legacyStubPresent()` removed from `legacyCheck`** -- the marker guard | the foreign leg converts **11 of a stranger's entities** (`balancer-part=0 bbb-balancer-part=11`, `trigger=init`, audit `clusters=3 parts=11`) and **four assertions fire**. The other legs stay green, which is the whole point: the guard is invisible to every leg that is entitled to convert |
| **the stranger case landing in `Done` rather than `Blocked`** | the SCAN still leaves the stranger alone -- `balancer-part=11 bbb-balancer-part=0` at every census, audit `clusters=0` -- and the **BUILD PATH does not**: one part built beside the stranger in phase two comes out `legacy=0 ours=1`. Exactly one assertion fires, and it is the late-build probe, which exists because of it |
| **`"belt-balancer-3"` misspelled in `legacyIncumbents`** | leg 3 fails on **exactly one assertion** -- *"the guest said it was leaving the incumbent alone 0 times"* -- and **every other number in that leg is byte-identical to the green run**: 11 parts, 2 surfaces, 3 clusters, 1 force, `trigger=configuration_changed`, witness 48 at four samples, 3.997x at 0.15% and 2.995x at 0.13%, late build `legacy=0 ours=1`, audit `clusters=3 parts=11 nets=3 drift=0 unbuilt=0`. Leg 2 stays green with `belt-balancer-2 2.0.9` still named. **That is the proof, not a caveat**: an unrecognised name takes the silent stranger path, the removal converts everything anyway, and the blocked line is the only thing that ever knew |
| **`"belt-balancer"` and `"belt-balancer-performance"` misspelled**, the same way | legs 1--7 all stay green and **each probe fails on its own line** -- *"...0 times and it is once per decision -- and ZERO means it did not recognise `belt-balancer` at all, which is the silent stranger path"*, and the same for `belt-balancer-performance`. Both probes still report **11 standing, 0 of ours**, so the failure is recognition and not staging |
| **`legacyBuilt`'s phase gate removed** -- the build path stops asking whether this game's `balancer-part` is ours | leg 5 fires **exactly the two assertions that are about it**: *"a `balancer-part` was converted in phase two, with an incumbent installed"* and *"a `balancer-part` built while belt-balancer-2 is installed came out legacy=0 ours=1"*, with `[BBB] legacy: adopted a balancer-part built at 12,0` in the log to say so. Everything else in the leg is unmoved -- 11 swapped in phase one, one blocked line, census `0 / 11` throughout, witness 48, 3.997x/2.996x, audit `3 / 11 / 3 / 0 / 0`. **The injection is caught EARLIER in the suite too**, by leg 2, whose phase one has an incumbent installed and whose eleven live entities the ungated build path converts -- so leg 5 was re-run in isolation to see its own assertions fire |
| **`legacyStubPresent()` removed from `legacyCheck`**, again -- but read by leg 7 rather than by leg 6 | leg 7 fires **exactly one assertion**, *"a `balancer-part` was swapped through the build path while the stranger owned the prototype"*, over **11** `adopted a balancer-part built at` lines in the CREATE log. **This is a moment leg 6 cannot see at all**: it has no guest in phase one, so the only thing it can watch a stranger's entities do is stand still, where leg 7 watches eleven of them being BUILT with the guest listening |
| **the bare-name `find_entity` put back in `legacyRunBuilds`** -- the defect as it shipped | legs 1--3 stay **green**, which is the honest half: the whole-world scan is quality-agnostic and never had the bug. **Leg 4 fails**, on *"19 parts were placed and 18 were swapped through the build path"*, *"1 incumbent parts are still standing"*, *"19 parts went in and 18 of ours came out"*, *"the audit counted 18 parts"*, and -- naming the tile -- *"the quality tile holds a balancer-part at phase=t1 and should hold a bbb-balancer-part"*, twice |
| **the `SetHealth` copy skipped in `legacyConvertOne`** | leg 1 fires **exactly the two health assertions**, one per phase: *"the damaged part is at 170.0 health at phase=t1 and was at 85.0 before the swap"*. Nothing else in the leg moves -- the part is converted, registered, compiled and balancing, and **silently repaired to full**, which is a building this mod was not asked to touch |
| **the `quality` key dropped from `legacyCreateArgs`** | leg 1 fires **exactly the two quality assertions**: *"the uncommon part came back at quality 'normal' at phase=t1"*. Every other number is unmoved: a quality is not a rate and not a count |
| **only the first force granted the technology** in `legacyScan` | leg 1 fires **exactly three**: *"1 force(s) were given the balancer technology and the force rig puts parts on 2"*, and the second force's `bbb-balancer` being false at t1 and at final. The parts, the clusters, the items and the rates are all unmoved -- a force that cannot craft a spare part is not a number any of them carry |
| **the force check removed from `AddPart`'s adjacency loop** -- two forces' touching parts fuse | leg 1 fires **exactly two, and both are the cluster count**: *"the conversion made 6 clusters out of the observer's 7"* and *"the audit finds 6 clusters and the observer built 7"*. **19 parts, 48 copper, 3.997x, 2.995x, drift=0, unbuilt=0 -- every other number in the leg is byte-identical to the green run.** That is the whole reason the expected count stopped being the guest's own |
| **`legacyScan` breaks out after the first surface it converts anything on** | leg 1 fires the surface cross-check by name -- *"the scan reports 2 surfaces and the world has 3 that are not the hidden one (nauvis, bbb-mig-a, bbb-mig-b)"* -- plus a cascade (17 adopted of 19, 6 clusters, and *"the migration ran 2 times in one save"*, because the `added` leg decides twice and the second decision finishes the job the first abandoned) |
| **`legacyScan` VISITS every surface and converts on only the first one that had parts** | the same leg, and now the per-surface census is the assertion with the name on it: *"this mod's parts are standing on 1 surface(s) after the conversion and were built on 2"*. The surface COUNT is correct here -- 3 scanned, 3 in the world -- so the cross-check above says nothing and the census is the only thing that can see it. **The two surface statements fail on different defects, which is why there are two** |
| **the observer's `report_counts("t1")`, `report_item("t1")` and `report_fidelity("t1")` deleted** -- the harness stops reporting a phase rather than the guest doing something wrong | the fixed script fires **exactly three**, by name and by phase: *"no copper count for phase=t1"*, *"no legacy-item line for phase=t1"*, *"no health line for phase=t1"*. **The same logs against the pre-fix script exit 0** and print the cheerful *"the incumbent's balancers were adopted and they balance"*, having skipped the copper count at the one sample straight after the load, the item comparison entirely, and the health and quality across the swap. That is the proof: not that a defect was caught, but that three checks which had never been able to fail now can |
| **the observer reports the quality line only in phase one** (health untouched) | the fixed script fires **exactly two**: *"no quality line for phase=t1"* and *"...for phase=final"*. Pre-fix, **exit 0** again. The finer injection is needed because both lines come out of one function, so deleting the call reaches the health branch first -- what this one covers is the quality line's FORMAT drifting out from under its regex while the health line's does not |
| **`legacyBuilt`'s phase gate removed**, read by the two NAME PROBES rather than by leg 5 | each probe fires **exactly one assertion**, the new one: *"19 `balancer-part` entities were swapped through the BUILD path while belt-balancer was still installed"*, over 19 `adopted a balancer-part built at` lines in the create log. **Pre-fix the same log exits 0 and prints "belt-balancer is recognised by name and its balancers were left alone"** -- a success message that was false in its own second clause. The probes were run in isolation, as leg 5's proof was and for the same reason |

**The fourth and fifth rows are the ones worth reading twice**, because they are the shape this suite exists to catch and the shape that is hardest to catch: the defect changes **nothing about what the mod does**. A misspelled incumbent name converts the same eleven parts into the same three clusters at the same rates on the same trigger. What it changes is that the guest stopped recognising a real mod by name — and the day that mod's balancers are standing in a save while it is still installed, it would convert them.

### The real Belt Balancer 2, once, by hand

The stand-in is a stand-in, so the added-as-removed flow was run once against the **actual cloned Belt Balancer 2 source** (github IThundxr/belt-balancer-2, MIT, info.json version corrected to 2.0.9), with its real control stage running during phase one. Not committed and not a suite dependency.

**It does not load as cloned**, and that is worth recording: `objects/part.lua` has a `goto continue` at line 266 with **no `::continue::` label**, which is a load-time Lua error (*"no visible label 'continue' for <goto> at line 266"*). One label inserted at the end of that loop body and it loads. The released 2.0.9 presumably carries the fix; the repository at HEAD (`456da9d`) does not.

With that one line added, the same `assert-mig.py --leg added` passes against the real mod with **numbers identical to the stand-in's, to the item** -- measured 2026-08-16 against the four-rig world, and not re-run since: 11 parts adopted from 2 surfaces into 3 clusters, 1 force researched, `trigger=init`, witness 48/48/48/48, 50 held with `place_result` flipped, `bbb-balancer=true`, control 1306, `m4x4` 3.997x at 0.15% and `m3to5` 2.995x at 0.13%, final audit `clusters=3 parts=11 nets=3 drift=0 unbuilt=0`. That is what makes the stand-in evidence rather than a model.

### What it costs

**Nothing on any hot path, and that is measured rather than argued.** The `mar` suite's seven per-operation slopes under `-gc=leaking` came back **identical to the byte** -- 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory -- which is the gate a pass that puts a gate on `onEventBody` and another on `fk_on_deferred` has to clear. Both are one integer compare in every save that has never seen an incumbent, and the marker re-test costs two host calls and no allocation in the only state that reaches it. The collected arm ended on 0.46 MiB with a 9,184 B live set, 9 collections in 5 paced steps and **0 forward-progress deadlines**.

Package built 2026-08-16, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 327,615 B | **347,170 B** | +5.97% |
| `fk_module.lua` | 2,534,887 B | **2,732,022 B** | +7.78% |
| `dist/bbb.wasm` | 1,106,373 B | 1,153,588 B | |
| members bound into the mod | 42 | **51** | of 4,257 |
| prototypes added | — | **3** | a `balancer-part` entity, a `balancer-part` item and the `bbb-legacy-stub` marker. ALL THREE CONDITIONAL: none is defined when another mod owns the name |
| sprite references checked | 7 | **10** | |

The nine new members are `LuaBootstrap.active_mods`, `LuaPrototypes.entity` (the handle-returning form) and `LuaCustomTable`'s index operator, `LuaEntity.quality`, `LuaEntity.health` and its setter, `LuaForce.technologies` (the handle-returning form) and `LuaTechnology.researched` and its setter. **No new FkLua gap and no re-pin**: every one was reachable through the generated bindings as they stood, `fklua gen-bindings --check` and `fklua lock --check` are unmoved, and the API pin does not move. The one thing that DID need upstream is the hook, and it landed in the same round.

### Verification, and the one gesture that is interactive

All nine suites green in **both arms** from clean, 2026-08-16, and **no other suite's numbers moved at all** -- M2's control is 1,306 with every rig rate unchanged, `edge`'s baseline is fifteen clusters over ninety-five parts with 0 lost over 200 teardowns, `mix` still reports its 72-item overflow, `plat` still reports its stacked-sushi band. That is the expected result rather than a weak one: no save in any other suite contains a `balancer-part` prototype, so the gate is a compare and the scan is never entered.

**What a headless run does NOT reach is a ROBOT reviving a ghost**, and the look of the thing. The code path a revive takes IS exercised -- the `built` leg drives `legacyBuilt` nineteen times through `script_raised_built`, and the late-build probe drives it once more in every leg -- so what is behind the wall is the construction network and the blueprint, not the swap. Both they and the pixels are on the interactive checklist as gesture F ([`test/interactive/README.md`](test/interactive/README.md)): take a real save with a real incumbent, swap the mods, and check the summary line, the plating, the belts, the chest, the technology -- then place one of the old blueprints and watch each revived ghost become one of this mod's parts.

**And verified again 2026-08-20, after the COVERAGE PASS that took the suite from four legs to seven plus two probes.** No guest line changed -- `dist/bbb.wasm` and `fk_module.lua` (2,745,246 B) are the bytes the merge left, `make check` is green with bindings and lock unmoved, and the sprite checker is green at 10 references. **All nine suites green in BOTH arms**, the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0), and **no other suite's numbers moved at all**, which is the only result a test-only pass may have. Inside `mig`, the four pre-existing legs report exactly what they always did -- 11 parts, 3 clusters, `trigger=init` and `trigger=configuration_changed`, witness 48 at every sample, 3.997x and 2.995x, audit `clusters=3 parts=11 nets=3 drift=0 unbuilt=0` -- and what the pass adds is three legs, two probes and four red proofs. The suite went 12.7 s to 27.0 s. **Those eleven-part counts are that pass's world**; the fidelity pass below took it to nineteen parts on three surfaces, and every current number in this section is the later one.

**And verified again the same day, after the FIDELITY PASS.** The suite claimed four things `legacyConvertOne` and `legacyScan` do -- health, quality, per-force technology and every surface -- and measured none of them, because every part in every leg was an undamaged normal-quality player-force part on one surface. Three rigs and a second surface later it measures all four, and **the first run of the first rig found a defect**: a legacy part at any quality but `normal` was invisible to the build path's `find_entity`, so leg 4 swapped 18 of 19. Fixed at that call site; the four other bare-name `find_entity` calls in the guest are written up above and were NOT fixed here -- they got their own pass and their own suite the same week ("A part at uncommon quality is a part" below).

`make check` green with bindings and lock unmoved, sprite checker green at 10 references, and **all nine suites green in BOTH arms**. The leaking arm's seven slopes came back **identical to the byte** -- 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0 -- which is the gate a guest change has to clear, and it clears it for the structural reason: the call that moved is on a once-per-save path that no other suite reaches. **No other suite's numbers moved at all.** Shipped config, forced clean rebuild either side of the fix:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 413,460 B | **413,608 B** | +0.04% |
| `fk_module.lua` | 2,745,246 B | **2,746,726 B** | +0.05% |
| `dist/bbb.wasm` | 1,162,033 B | 1,162,312 B | |
| members bound into the mod | 51 | **51** | of 4,257 -- none added; `find_entities_filtered` was already bound |

**What the pass closed, in one sentence each.** `belt-balancer-3` was never in front of the guest and neither were the other two names, so three of the four rows of `legacyIncumbents` were unpinned and a typo in any of them would have converted that mod's balancers out from under it while it was still installed. `Done -> Blocked` was never driven, so the build path's phase gate -- the one thing standing between an incumbent that arrives late and having its freshly built entities swapped -- had no test. And `legacyCheck`'s promise to a stranger, that uninstalling gets them the same adoption an incumbent gets, was a comment.

**And verified once more the same day, after the REVIEW GATE over both passes above.** Nothing in the guest, the observer or `run.sh` moved -- the whole change is four assertions in `test/assert-mig.py` that could not previously fail, plus two stale prose numbers in this file -- so `dist/` is byte-for-byte what the fidelity pass left (zip 413,608 B, `fk_module.lua` 2,746,726 B, 51 members) and no rebuild was needed to reach it. `make check` green with bindings and lock unmoved; **all nine suites green in BOTH arms**, each arm one invocation; the leaking arm's seven slopes back **identical to the byte** -- 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0; **no other suite's numbers moved**, which for an assertion-only change is the only result available. Three red proofs, all of them injections into the HARNESS, are the last three rows of the table above.

**What the gate found, and it is one shape seen twice.** Three shared checks and one probe could pass while measuring nothing -- a missing log line was a skipped assertion rather than a failed one, and the probe's own success sentence (*"...and its balancers were left alone"*) printed over a create log carrying nineteen conversions of a still-installed mod's entities. Both are written up in "A CHECK THAT SKIPS IS A CHECK THAT PASSED". What it did NOT find: nothing was weakened by either pass, the original four legs assert everything they always did and five things more, `bench/` was untouched, and every claim in the two passes' tables re-ran to the number recorded.

### And on Factorio 2.0 the outcome splits the other way -- the conversion is grandfathered

**A player who uninstalls Belt Balancer on 2.0 keeps their base, and until 2026-08-24 nothing had ever run that branch.** The conversion is byte-identical on both engines -- `legacy.go` knows nothing about the rule -- and so is the first flush after it: the setting defaults to false, so all seven incumbent-idiom clusters are refused with the same seven alert lines and the same shape multiset `[2, 2, 2, 2, 2, 3, 4]`. What happens NEXT is the whole difference. The capability marker is present, so `settleEdgeMode` asks `edgemode.GrandfatherNeeded(marker, Off, 7)`, gets true, writes `bbb-multi-edge-parts` ON, re-queues every cluster and tells each owning force -- and the very next flush compiles all seven.

| | 2.1 | **2.0** |
|---|---|---|
| the final audit | `clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7` | **`nets=9 ... refused=0`** |
| `sok2` / `sok4` | 2.000x / 3.997x | **2.000x / 3.997x** |
| `m4x4` / `m3to5` | **0 0 0 0** and **0 0 0 0 0** | **3.997x at 0.15% and 2.996x at 0.26%** -- the pre-port records to the digit |
| what the player is told | the migration checklist | **the grandfather warning**, both once per force: 6 and 1 |
| teardowns, spills | 0, 0 | **0, 0** |

**Two constants had to be SPLIT rather than switched**, and both splits are the kind one engine cannot see. The number of refusal LINES is 7 on both; the audit's own `refused=` column is 7 on 2.1 and **0** on 2.0, because the grandfather compiles all seven a flush later and a successful compile clears the feedback memo. And the `added` leg's rebuild-from-world reports **2 adopted / 7 rebuilt on BOTH** -- it runs in the dispatch after the conversion's flush, before the grandfather's re-queue has been flushed -- so that is its own constant and not the audit's. **The per-force log line is SHARED by both messages** (`tellAffected` writes it whichever sentence it is delivering), so what it pins is asserted on both engines; only the summary SENTENCE is 2.1-only, and on 2.0 it would be false of every balancer a flush later.

### And verified again 2026-08-24, on Factorio 2.1, where the outcome splits

**The single-edge port did not touch this feature and changed its answer completely.** `legacy.go` is byte-identical: 31 parts still convert from 3 surfaces into 9 clusters at their health and their quality, 2 forces are still granted the technology, the item stack still survives and still flips its `place_result`, the witness's 48 copper plates are still 48 at every sample, and the state machine's two axes are untouched because it knows nothing about belts. What moved is what the compiler then does with the clusters: **seven of the nine are laid the incumbent's way -- one column of parts with a belt on both free faces, which is two belts per part -- and Factorio 2.1 refuses them.**

**This is the one suite whose rigs were deliberately NOT re-laid.** Every other suite in the estate obeyed the rule because its rigs are ours; this one's world is somebody else's, and re-laying it would have been re-laying the thing under test. What was added instead is the **`sok` band**: the same balancer two columns wide, one belt per part, which is a shape a Belt Balancer user could genuinely have and which converts into a network that runs. One world, both outcomes, which is the portal story rather than a hypothetical.

    clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7

identical in all five conversion legs, with `sok2` at **1306 1306 -- 2.000x one belt, 0.00% spread**, `sok4` at **1304 1306 1306 1304 -- 3.997x, 0.15%**, and `m4x4` and `m3to5` at **exactly zero**, asserted as zeros rather than as a loosened bound. **Nothing is torn down and nothing is spilled** -- a cluster the conversion just created never had a network, so the refusal in front of the teardown has no teardown to be in front of -- which is where this suite asserts the opposite of `mig21`'s for the same refusal, and correctly. And leg 1's rebuild-from-world **adopts the two working clusters beside the seven it refuses**, which closes a gap the port's phase-2 section records against `mig21` and could not close there.

`make check` green with bindings and lock unmoved (no guest line changed and no package was rebuilt); **all thirteen suites green in BOTH arms**, each arm one invocation, with `mig` back in the default; the leaking arm's slopes **identical to the byte** to the port's phase-4 record; and **no other suite's numbers moved**. Five red proofs, two about the new outcome and three regressions across three families of the conversion, are in [`agents/single-edge.md`](agents/single-edge.md)'s phase-7 section, along with the one defect this pass found and did not fix -- a converted-and-refused balancer gets the ORDINARY per-piece message rather than the migration summary, unless a rebuild-from-world happens to follow the conversion.

<!-- END: adopting an incumbent's save -->

## A part at uncommon quality is a part — the quality-blind lookups, closed

**The four bare-name `find_entity` call sites the migration pass wrote up as NOT fixed are fixed, and the tenth suite exists because none of them was reachable by any rig this repo had.** The defect class, once more in one sentence: `find_entity` takes an `EntityWithQualityID` and resolves a bare name as **normal quality only** (the probe table is in the migration section), so a lookup that used it worked on every normal-quality save this repo has ever run and silently failed on a part a player built from a quality-rolled item. Closed 2026-08-20; `guest/go/findpart.go` is the fix and its header is the long form.

**`findOnTile` is the fix stated once** — a one-tile area query with a name filter, which is the question every caller was actually asking (is a thing of OURS with this name standing on this tile), with the quality left out of it the way `setSearchBox` + `findByName` already leaves it out everywhere else. All five sites go through it now, `legacyRunBuilds` included, so a sixth question about the same identity asks the same code or does not ask at all. Two details are decisions rather than plumbing:

- **Dedicated buffers, not `findByName`'s.** One caller (`reapFastReplaced`) runs on the EVENT path and the shared `searchArea`/`nameFilter` scratch belongs to the flush; a private filter struct costs a few static bytes and removes the aliasing question instead of answering it.
- **`found` and `err` are separate returns, on purpose.** `reapFastReplaced` unregisters a part on the strength of a MISS, and the old code deliberately refused to edit the registry on a failed query. A helper that collapsed the two would have turned a host error into a registry edit — which is exactly the class of quiet semantic change a mechanical refactor ships.

### What each site did while it stood, measured rather than recited

The migration section's table said what a non-normal part would do to each site; the `qual` suite's red proof ran the pre-fix guest and watched all of it happen at once (below). The sharpest two: `reapFastReplaced` really does unregister a standing part when a script builds a colliding belt on it — the audit went `clusters=4` → `3` with the part still in the world — and `restyle` really is an eternal retry, not a one-time miss: **24 skin lines instead of 6** over one 2,160-tick run, every one `set=0`, because a part it can never find is re-queried by every flush that touches its cluster.

### The `qual` suite — the tenth, base plus the quality mod

Every part in every rig is uncommon (`[BBB-QUAL] quality rig=… value=uncommon` is asserted per rig, so a run where the quality silently failed to apply fails as vacuous). One 2,160-tick run, four rigs plus a control belt:

| rig | what it is | what came out |
|---|---|---|
| `qblk` | a 2x2 BLOCK, two in and two out, saturated | skin line `parts=4 set=4 vars=21,27,17,35` (m1's own literals for the shape) exactly ONCE — a mid-run poke inside the neighbour gate provokes the extra flush that the unfixed guest turns into another `set=0` line — and **900 900 against the control's 900: 2.000x one belt, 0.00% spread**, which is the first evidence anywhere that an uncommon balancer BALANCES |
| `qcol` | a 1→1 column of four, the fast-replace TRUE POSITIVE | `can_fast_replace` **true** over an uncommon interior part (the engine does not gate the gesture on quality either), the replace really removes it, the guest unregisters it exactly once, and the column splits 1 cluster → 2 with the halves' skin lines carrying the right variations (`5,2` with `set=1` — the bottom part's picture was already right and is not re-set) |
| `qlone` | one lone part, the TRIPWIRE | a script builds a COLLIDING belt on its very tile; the part is still standing, and the registry does not move — `clusters=4 parts=75` before and after, and **zero** `fast-replaced the part` lines for its tile |
| `qlim` | the edge suite's 64-input block, uncommon, given its sixty-fifth belt | **exactly one** refusal alert (128 ports for 65 inputs over the limit of 64) and **exactly one clean `told force 1` line** — the refusal is DELIVERED, which is the whole point: `forceOfCluster` reads the force off a part of the cluster, and every part is uncommon. Delivery holds across the edit (900 items over the window) and the audit stands at `drift=1 unbuilt=0 refused=1` for the rest of the run |

**Three of the four rigs were single-edge already, which is why this suite cost the 2.1 port so little.** `qblk`'s west column carries the inputs and its east column the outputs, `qcol`'s two interior parts carry nothing, `qlone` has no belts. `qlim` was the exception -- a belt on both sides of every part is exactly what 2.1 forbids -- and it is **sixty-six parts** now: one output part above, a 2x32 input block, and one EDGELESS part below for the sixty-fifth belt. That spare part is forced rather than free, and for the reason the interactive checklist's band C records: under the rule all sixty-four input parts already carry their belt, so a sixty-fifth belt against any of them would ask the SINGLE-EDGE bound instead and this would stop being a test of `forceOfCluster` at all.

The audit walk is asserted at every step — `(4, 75, nets=0, 0, unbuilt=3, refused=0)` at t0 (the audit inside `on_init`'s marker dispatch reports BEFORE the drain it forces compiles anything, and qlone's edgeless cluster is never "unbuilt"), `(4, 75, 3, 0, 0, 0)` post-collide, `(5, 74, 2, 0, 0, 0)` post-replace, and `(5, 74, 2, 1, 0, 1)` from the refusal to the end. **The tuple is the assertion and `unbuilt=0` alone would not be**: a cluster with no inputs or no outputs is a legitimate half-built state and never counts as unbuilt, so a rig that lost half its belts would read `unbuilt=0` while delivering nothing. `nets` is written down per tag rather than compared against `clusters` here, because three of this save's five clusters are legitimately network-free by the end. The skin assertion is an **exact multiset of six lines** for the whole run, which is what makes both halves of the restyle claim — found once, and never retried — one comparison; qlim's line is **truncated by the guest at 32 variations** with a literal `...`, which no cluster in any other suite is big enough to reach, so the truncation is part of what is matched rather than parsed away.

### Red-proven, one pre-fix build, every family firing at once

The whole fix stashed, the guest rebuilt, the same suite run. Three rigs, three sites, three failure families, each naming itself:

| site | what fired on the pre-fix guest |
|---|---|
| `restyle` | **24 skin lines instead of 6, every one `set=0`, every variation 0** — the retry-forever half and the never-found half in one number |
| `reapFastReplaced` | *"the guest unregistered the lone part at (0, 40) under a COLLIDING belt"*, plus the audit walk failing at every post-collide sample (`clusters=3 parts=74` where the world holds 4 and 75) |
| `forceOfCluster` | *"the force was told about the refusal 0 time(s)"* — the alert fired and nobody was told |

**Re-run 2026-08-24 against the single-edge rigs**, with `findOnTile`'s filter given `Quality = "normal"` -- which is the pre-fix `find_entity` semantics stated exactly -- and all three families fire again with the same shapes: **24 skin lines instead of 6, every one `set=0`**, the lone part unregistered under the colliding belt with the audit walk failing at all four post-collide tags, and the force told **0 times**. The true positives stay true in that arm too.

And the true positives stayed true on BOTH arms, which is what makes them controls rather than assertions of the fix: the colliding belt was created, the real fast replace removed its part and was reaped exactly once, and the uncommon block delivered 2.000x at 0.00% in the pre-fix run too — conservation and throughput were never the defect.

### What is still behind the player wall

`revertOne`'s isPart arm shares the lookup and the fix, and its observable — the over-limit part arriving back in the inventory — needs a player, which no headless run has. Same wall as always ("The sixty-fifth belt"); the suite asserts the standing negative (**zero hand-backs**), and the interactive gesture, if anyone wants it, is the over-limit gesture from the checklist done with a quality part in hand.

### What it costs

Package built 2026-08-20, shipped config (`--persist=packed --gc=collected`):

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.1.0.zip` | 413,608 B | **414,060 B** | +0.11% |
| `fk_module.lua` | 2,746,726 B | **2,747,461 B** | +0.03% |
| `dist/bbb.wasm` | 1,162,312 B | 1,163,064 B | |
| members bound into the mod | 51 | **51** | none added — `find_entities_filtered` was always bound and `find_entity` stays bound for nothing (the binding is generated either way) |

**Nothing on any hot path moves, and the `restyle` design question dissolved under measurement.** The migration write-up priced the repair as "a second byte or an allocating query on the flush path" against `mar` slopes asserted to the byte — and the seven slopes came back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B, 3.92 MiB of linear memory), because the one-element slice `find_entities_filtered` returns lands in the same TinyGo allocation size class as the boxed Object `find_entity` returned. The retry cost the old code was PAYING — one host call per unfound part per flush, forever, on any save with a quality part — is gone with the defect. `reapFastReplaced`'s ordinary-play cost is unchanged: the point-query miss returns before any host call, and `mar`'s leg B (352 B) and leg D (32 B) say so. `make test` went 1m27.8s → 1m34.9s for the tenth suite.

## Status

Design fixed 2026-07-31: **compile, don't interpret** -- balancer clusters compile at build time into hidden-surface networks of real splitter entities stitched in with linked-belts; steady-state script cost is zero. The Go/FkLua guest is the compiler. Read [`agents/design.md`](agents/design.md) before any implementation work, and [`agents/spike-s1.md`](agents/spike-s1.md) before touching the edge interfaces -- every gotcha in it is load-bearing.

**All five milestones are done, and so is the one performance defect that outlived them.** S1, M1, M2, M3, M4 and M5; the sections below are in that order, and "M5 is done" is the last of them. After M5 came one more pass — **"The heap diet"**, which closed the idle GC spike that had been open since M4 — and then the wrap-up: [`README.md`](README.md), this section, and a final verification from clean.

**Where the project stands.** Both goals in the header are met and both are measured rather than asserted:

| | |
|---|---|
| **The brain is Go** | 4,600 lines of it, plus 92k of generated bindings. No hand-written Lua in the control stage at all — `fklua mod` writes every byte of it. `guest/go/host.go` and `test/check-layout.py`, the two files that existed because a binding could not do something, are both deleted |
| **...and so is the DATA STAGE, so the mod contains no hand-written Lua at all** | Since 2026-08-25 the settings and data stages are `guest/go/data`, a second compiled guest, and `mod-data/` is graphics, locale, a changelog and a thumbnail. Eleven Lua files and 45,037 bytes gone; the version branch two Lua states had to share a file for is one Go function two exports call; the technology's aliasing of base's own unit table died with the reference. Proved BYTE-IDENTICAL against goldens taken from the Lua before the port, on both mod sets, by hashing Factorio's own `--dump-data`, with the control guest's `fk_module.lua` and the `mar` heap slopes unmoved to the byte. "The shipped mod holds no Lua" |
| **What it costs to build and to research is the player's** | Two STARTUP string settings on both engines: six recipe plans and three technology tiers, both defaulting to what shipped, both resolved through existence-guarded fallback ladders so that no ingredient and no prerequisite ever names a prototype the game does not have. Proven by the eleven-arm dump gate, one arm per non-default value, driven through a `mod-settings.dat` the gate writes; red-proven by deleting the guard, which kills a base-only load on the engine's own `Error in assignID`. "Cost, research and belt speed" |
| **...and the hidden network is as fast as the fastest belt installed** | 0.25 tiles/tick was called a ceiling for five milestones and is a FLOOR: a belt above it runs fully compressed at 480 x speed items/s, so a modded belt over 120 items/s was silently throttled at every port. `data-final-fixes` now scans all seven belt-connectable families and gives the four hidden prototypes the maximum, with NO upper cap, because every candidate is a value the engine already accepted on the prototype it came from. Vanilla and Space Age are unchanged to the byte (turbo is 0.125, half the floor), which the dump goldens prove; a Go fixture mod defining a 0.5 underground proves it derives |
| **It beats the incumbent** | 45×/47× against belt-balancer-2 and -3 on a saturated express 4×4, `scriptUpdate` at the no-mod control's in every cell, exact balance under every load condition. [`bench/baselines/RESULTS.md`](bench/baselines/RESULTS.md) |
| **It is correct under the whole 2.0 lifecycle** | Eight headless suites in a real Factorio, and the stale-reference crash class is impossible by construction rather than by handling |
| **A teardown never destroys an item, whatever is on the belt** | A recompile reinserts what it drained; only a removal spills — and past the carry pool's 32-group bound, which used to DROP the overflow after reading it off a line about to be destroyed, it spills that too. 48 kinds through one saturated 4x4: 4,336 items in, 4,336 out, every name exact. "More than thirty-two kinds" |
| **An edit mid-operation keeps its items IN the machine** | A recompile reinserts what it drained; only a removal spills. Zero ground items across 200 teardowns of a saturated rig, a bridge-merge, a split under load and every shape of edge edit — and the 4→5 network a recompile built balances to 0.00% over the next 500 ticks. "A recompile is not a removal" |
| **...and it keeps them STACKED** | A Space Age recompile hands a stacked network back stacked: 0 extra single-item belt positions, every crossing item accounted for in the stacked total, 0 spills, and the recompile itself is **1.7× faster** than the flat path it replaced. Base Factorio pays nothing — the gate is one force attribute and the `mar` slopes are identical to the byte. "Stacked belts come back stacked" |
| **...even when the stack is not all one thing** | A stacked SUSHI network — 14 of 24 hidden lines carrying two item names and 6 carrying one name at two qualities, measured at the instant the teardown read them — comes back **exact per (name, quality) over nine kinds, +0 single positions, 0 spills**. That is the pair of conditions `kindAt`'s two multi-candidate branches need, and until 2026-08-05 no rig anywhere met it. "Stacked sushi" |
| **...and a player who mines one keeps them** | Any removal a player caused — a part mined, a shrink as much as the dissolve, and since the second field report **a belt mined at the machine's edge** — offers whatever no network could take to that player before the ground, like mining a vanilla machine. Only the TRIGGER needs a player; the insert arithmetic, the claim identity and both quantities (118 items taking a balancer apart by hand, 128 mining an output belt off one) are pinned headlessly by `go test ./carry/` and the `edge` suite. "The miner's pocket" |
| **...and one it cannot build is refused without demolishing the one that works** | A 65th belt against a 64-port balancer is refused BEFORE the teardown: 0 items on the ground where the unfixed guest put 1,690, the standing network delivering 184 items over 246 ticks before the edit and 185 after, and the player told, with the belt back in their inventory. "The sixty-fifth belt" |
| **...or the TWO that work** | A part bridging two working balancers into one that is over the limit is refused before their teardowns too, which are `AddPart`'s and not the compiler's: 0 items on the ground where the unfixed guest put 1,814, both halves still delivering 184 and 184 against 186 and 185 across the edit, and mining the part back out costing 0 teardowns and 0 builds. "The merge that would be over the limit" |
| **...and a part clicks over a belt like a splitter does** | `fast_replaceable_group = "transport-belt"` on the part, which is base's own group: a balancer can be dropped straight into a belt line you already have, and the belt goes to the player. The group is symmetric, so a belt laid on a part replaces it too — and the engine raises NO event for the part it destroys, which is the whole of `guest/go/fastreplace.go`. "Fast replace" |
| **A Belt Balancer 2 or 3 save becomes one of ours** | Uninstall the incumbent and every `balancer-part` it left standing becomes one of this mod's, at load, once per save: 31 parts across 3 surfaces and 2 forces into 9 clusters, **at the health and the quality they were standing at**, of which the ones laid ONE BELT PER PART deliver 2.000x and 3.997x one belt and the ones laid the incumbent's way are refused on Factorio 2.1 -- their geometry cannot function there under any design, so what a player gets is their parts, their items and a rebuild checklist, with the items on the belts conserved exactly (48 copper before and after), the item stacks surviving and placing our parts, and the technology granted. Nothing at all happens while the incumbent is installed, or while any other mod owns the name -- **including an incumbent that arrives AFTER this mod, on a save this mod has already converted**, where the balancers we own keep running and a `balancer-part` the newcomer places stays theirs. All four incumbent names are exercised, and so is the stranger being uninstalled in his turn. Proved against the real Belt Balancer 2 as well as the harness stand-in. "Adopting a Belt Balancer 2 or 3 save" |
| **A part at any quality is a part** | Every place the guest asks the world for one of its own entities by name is quality-blind since 2026-08-20 -- `findOnTile`, one helper for all five sites, after the migration pass found `find_entity` resolves a bare name as normal quality only. An uncommon balancer draws its shape, balances at 2.000x with 0.00% spread, is refused past the port limit WITH the refusal delivered, and survives a scripted colliding belt without losing a registry entry. The tenth suite (`qual`) is every part of that, red-proven against the pre-fix guest in one run. "A part at uncommon quality is a part" |
| **Nothing the compiler places draws anything** | The hidden prototypes are clones of base belts and kept base's pictures — including a three-by-three linked-belt `structure` on the one prototype that stands where a player looks. All four are blanked, and the `edge` suite asserts the structural half: 180–197 visible-surface entities of ours, **every one on a registered part tile, 0 off one**, across six samples. "The tan streak" |
| **Its long-game cost is measured, not assumed** | Every net-zero world operation's permanent-heap slope, flat over hundreds of iterations, a 300-hour projection built on it, and — since 2026-08-02 — the one stall that projection predicted, measured at **782 ms** and then removed by shipping `--gc=collected`. See "The marathon save" and "The third decision" |
| **The art is drawn, not computed** | All four assets are an artist's, delivered 2026-08-19: the 47-cell sheet, the icon, the I/O arrows and the mod logo. They dropped in with no code change but one alignment constant, because the cell order and the eight arrow cells were a contract the spec stated and the delivery met. `tools/make-graphics.py` still generates the placeholders and still DEFINES that contract; it is the fallback and the specification, not the shipped pixels |
| **One belt per balancer part, on Factorio 2.1** | 2.1 closed the collision-mask loophole two interface linked belts shared a tile through, so trunk targets 2.1 and a cluster tile carries at most one edge. The rule enforces from a per-tile count that falls out of the classification walk, the refusal is the sixty-fifth belt's own machinery down to the wake-race guard, and a merge that would break it is spared by the bridging-tile theorem: measured on the `sedge` suite at 1.002x / 2.000x / 4.004x / 3.001x one belt, three refusals with the standing networks still delivering, and zero spills and zero hand-backs. Red-proven: with the predicate off the same run dies on ten `[BBB] error:` lines and puts 64 items on the floor. **Phases 1 through 9 are done -- all fourteen suites are green in both `-gc` arms on Factorio 2.1.14 AND on 2.0.77**, which is the other engine arm and which restored the heap-slope gate on both. "One belt per balancer part" and [`agents/single-edge.md`](agents/single-edge.md) |
| **...and on Factorio 2.0 the mode is a setting, which the mod defends rather than obeys** | Multi-edge survives there behind `bbb-multi-edge-parts`, default off, and turning it OFF while multi-edge balancers are standing is **VETOED**: the setting goes straight back on and the player is told what to rebuild, with a ping per balancer -- and the map is CHARTED around each one, because a ping at uncharted ground opens on black. It was a SWEEP until 2026-08-24 -- tear them down and spill -- which could never stick (the condition that makes a sweep find something is the condition that makes the grandfather write the setting back) and which put a full balancer's contents on the floor on the way. Measured either side of the fix: ground `0 -> 88 -> 64` against `0 -> 0 -> 0`, and the networks' own contents `120 -> 24 -> 76` against `120` throughout. The `flip` suite drives all four transitions, on the only engine that has them |
| **A 2.0 multi-edge save opened on 2.1 stops honestly, and says where the items went** | The engine has already deleted all but one belt-connectable per tile before any script runs, silently, so the guest wakes into crippled machines. It tears every remnant down -- the one place a refusal demolishes anything, because a stacked linked belt in a 2.1 world is an engine risk rather than a degraded balancer -- recovers what they were holding exactly, spills all of it, refuses every cluster and tells each owning force once with a clickable ping per balancer. Measured on both committed fixtures: 2,320 and 6,540 items recovered and spilled to the item, 0 put back, 0 of the compiler's entities left standing, the player's 77 and 95 parts untouched, and the audit stable at `nets=0 drift=0 unbuilt=0 refused=N`. On Factorio 2.0 the same scan reaches the other outcome: the balancers are kept working, the mod writes its own setting to say so, and the force is warned once. "A save built to the other rule" and [`agents/single-edge.md`](agents/single-edge.md) |
| **...and the TEST ESTATE has followed the mod out of Lua** | An observer mod builds a world, drives it on a schedule and reports what it sees; fourteen of them were `control.lua` files. **EVERY SUITE THAT CAN BE PORTED IS A COMPILED GUEST since 2026-08-25** -- the pilot's `m1` and `sedge`, then `mar`, `mig21`, `qual`, then `mix`, `plat`, `mig`, and in phase 4 the three biggest, `m2`, `m3` and `edge` -- sharing one harness package and the mod's own generated bindings under `guest/go/obs`. **Phase 5 took the INTERACTIVE STAGING MOD**, which was never a suite: the world a human walks and the mod portal's demo scenes are staged by a compiled guest now, and `make interactive-install` installs the very package `iact` gates. **Phase 6 took the two DATA-STAGE-ONLY STAND-INS**, the first packages here with no control stage at all, and **PHASE 7 THE `bench/` HARNESS'S SETUP MOD**, which is the last one this machine can do. The estate is **478 lines of Lua from 8,524, and all of it is `bbb-flip-test`**, which waits for a 2.0 binary. Every one of their log lines came back byte-identical against goldens taken before the port -- 320 in phase 1, 10,519 in phase 2, 1,290 in phase 3, 4,220 in phase 4, 498 in phase 5 and 819 in phase 6, in order, with only a timestamp, a source attribution and a profiler's own measured milliseconds masked; a red proof per phase says the assertion surface is really exercised; and the mod's own heap slopes and member table have not moved by a byte through any of it. **Phase 7's gate is not a golden log but a COMPARABILITY run**, because `bench/` is measurement infrastructure: the same cells, both setup mods, interleaved in one session, with throughput identical to the item in every one and the marginal cost per balancer agreeing to 0.002 µs; `agents/estate-port.md` |
| **It is not shipped** | No mod-portal release, and no play-testing beyond the suites and one guided pass. Licensed MIT (`LICENSE`, 2026-08-15) |

Final verification, from `make clean`, **2026-08-02**, Factorio 2.0.77, in the SHIPPED configuration (`--persist=packed --gc=collected`): `make check` green; `test/run.sh` green on all seven suites (M1 6+3 phases, M2 8 rigs, M3 12 rigs, `upg` plus M2's whole assertion set again, `plat`, `mar`'s slope legs and `edge`'s hundred-cycle churn) — **and green again on all seven in the `GC=leaking` arm**, which is the bar every pass that touches the mode decision keeps. Re-run from clean in both arms after the item-placement policy ("A recompile is not a removal"), again in the shipped arm after the prototype-visual pass ("The tan streak"), and **again in BOTH arms after the belt-stacking pass** ("Stacked belts come back stacked") — the `mar` slopes came back identical to the byte in the leaking arm, which is the measurement that says the gate really is closed for base-only play. **Green in both arms again after the miner's-pocket correction** ("The shrink was the whole feature"), which added two legs to the `edge` suite rather than changing a number in any other one: `player_index` is zero on every removal a headless run can produce, so the fix is invisible to the suites and what they gained is the measurement of what it redirects. **Green on all seven in the shipped arm again after the beneficiary's force check** ("A claim is a Region"); every suite number is byte-identical for the same reason, and the evidence that pass added is a `go test` — the first failing test this repo has ever been able to write for the miner's pocket. **And green in both arms again after the second field report** ("A mine beside a machine is a mine of that machine"), which is the last change this file records. That one DID move the suite numbers, because it added a rig rather than only a call site: the `edge` baseline is twelve clusters over thirty-one parts, and its new `bmin` leg reproduces the report at **128 items on the ground** — the same 128 on the pre-fix guest, which is what makes it a tripwire on the quantity rather than on the fix. **The `mar` slopes came back identical to the byte in the leaking arm** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B per primitive and 3.92 MiB of linear memory — which is the measurement that says a claim recorded on the guest's hottest path costs nothing where no player mines. n=200 k=4 express saturated **0.7565 ms/tick against the control's 0.5940** (median of 5 interleaved reps; the earlier 0.4510/0.4760 pair is a different session and not comparable), 1,740,000 items at balance 1.001, zero `[BBB]` lines in the benchmark window.

**Re-verified from clean 2026-08-03**, Factorio 2.0.77, after FkLua's 2026-08-03 fix round (the `fkgc` pre-init threshold latch, and F2's optional attributes): `make check` green, bindings regenerated and locked, sprite checker green, and **all seven suites green in BOTH arms** — the leaking arm's seven slopes back identical to the byte (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) and the `edge` baseline back at twelve clusters over thirty-one parts with 0 lost over 200 teardowns. Two things this round found and neither is upstream's: the `gc.go` comment claiming the arming threshold "cannot drift apart" was describing a call the collector had been discarding all along ("The threshold this guest installed and the collector never read" — it cost this mod nothing, by an accident of taste), and **the collected arm's post-load transient is 985 ticks / ~906 ms against the 71 / 68 ms "The third decision" was taken on** ("The transient that grew while nobody was looking"). The zero-script steady state is unaffected and re-measured at the control's in both scenarios; the transient is the one open number this repo now has.

**Closed the same day, and it was not what the round named.** The item-placement policy is exonerated by measurement — the pre-policy guest reproduces the 982-tick transient on a byte-identical heap — and the cause is this guest's **globals crossing 16 KiB, which is exactly one paced collector step's budget**, after which the mark phase can never afford its own termination attempt. One knob (`fkgc.SetBudget`, beside the `SetThreshold` this guest already installed) and one check that has been verified to fire. **The transient is 36 ticks and 54 ms** — better than the 71 / 68 the mode decision was priced on — and the `mar` suite, which is the half that is about play rather than about a benchmark, runs 8 collections in **14 paced steps with 0 forward-progress deadlines** against 6 in 779 with 6. Final verification from clean on **2026-08-03**: `make check` green, sprite checker green, **all seven suites green in BOTH arms**, the leaking arm's `fk_module.lua` byte-identical apart from its build stamp and its seven slopes identical to the byte. Shipped zip 290,455 → **291,364 B**. See "The root scan that could not fit in a step", and [`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 21, which is the upstream ask.

**And re-verified again from clean 2026-08-03 after the STANDARDIZATION PASS**. That pass regenerated the bindings against a FkLua six rounds newer (B1a/B1b/B2, members 4,191 → 4,250, and **member ids moved** — `LuaControl.Insert` 316 → 317, so the committed pair was one that would have called a different function silently), re-derived the collector budget the same round's root-scan RESERVE had moved out from under (the `mar` suite's `deadlines` went 0 → 3 and back to **0**), deleted the check upstream now does better, moved the shipped GC arm into `fklua.toml`, and gave the audit a door a player can open. `make check` green, sprite checker green, **all seven suites green in BOTH arms**; `mar` runs 9 collections in **5 paced steps with 0 forward-progress deadlines**, and the leaking arm's linear memory came back at 3.92 MiB, identical to the record. The post-load transient is **17 ticks / 44.2 ms**, better than the 36 / 54 the previous section records, and steady `scriptUpdate` is a median of **0.33 µs**. Shipped zip 291,364 → **298,784 B** (+2.5%, almost all of it the command seam's `fk_on_call` trampoline and the audit's remote form). The keep/standardize table is "The standardization pass" at the end of this file.

**And verified 2026-08-04 after the EIGHTH SUITE** — `mix`, the first suite ever to run more than one KIND of item through a balancer, and the item sink it found on its first run ("More than thirty-two kinds"). `make check` green, sprite checker green, **all eight suites green in BOTH arms**. The leaking arm's seven slopes came back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB), and a pre-fix/post-fix run of the other six suites differs in **no asserted number at all** — which is the gate this pass had to clear, because every pre-existing rig is single-kind and a fix visible to any of them would have been a fix on the wrong path. What the pass adds is the red proof: the guest that shipped destroys **16 item kinds and 72 items** on one recompile of a saturated 4×4 carrying 48 kinds, and the fixed one puts exactly those 72 on the world. Shipped zip 298,874 → **300,422 B** (+0.52%), `fk_module.lua` 2,282,383 → **2,316,009 B** (+1.47%).

**And verified 2026-08-04 after the OVER-LIMIT PASS** — "The sixty-fifth belt", the one item on [`agents/maxports.md`](agents/maxports.md)'s queued list that was a defect rather than a direction. `make check` green, sprite checker green, **all eight suites green in BOTH arms**, and the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) — which is the gate a pass that records something on the guest's highest-multiplier build path has to clear. The `edge` suite gained a rig, so its baseline moved to thirteen clusters over sixty-three parts and its placement-probe counts to 114–131; **no other suite's numbers moved at all**, which is expected rather than weak, since nothing else in this repo reaches 64 ports. The red proof is in `assert-edge.py`'s LIM section and it is the reason the leg exists: with the check back where it was, the same belt tore down a 1,876-item network, put 1,690 items on the floor, stopped delivery dead, and made `test/run.sh` kill the run on an `[BBB] error:` before a single assertion was read. Shipped zip 300,459 → **310,630 B** (+3.38%), `fk_module.lua` 2,319,175 → **2,432,188 B** (+4.87%), members 38 → **42** of 4,257.

**And verified 2026-08-05 after the STACKED-SUSHI BAND** — `plat`'s fifth band, the first rig anywhere to reach `kindAt`'s two multi-candidate branches ("Stacked sushi"). `make check` green, sprite checker green, **all eight suites green in BOTH arms**, and the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) — which is the expected result rather than a weak one, because **not one guest line changed**: this is a suite-only pass and the shipped zip is the byte-for-byte one the over-limit pass left. No other suite's numbers moved either, and inside `plat` the four pre-existing bands are unmoved to the item (1,128 items over 336 positions at `formed`; +0/+128, +16/+0, +0/+48; 3.971× at 0.54% spread; 676 676 against 676 on the platform), because `smix` runs six item names none of them touches and every count in the test mod filters by name. What moved is additive: the suite now requires four recompiles to have handed something back rather than three, and the profiler's own `audit only, nothing pending` control rises 7.56 → **10.85 ms** because the window carries a whole-save re-classification and the save gained a band. Two red proofs, and they catch different things: the gate forced off fires the three profile assertions (**+196 single positions**, 56×4 → 196×1) and leaves per-kind conservation exact, while a `kindAt` that always answers candidate 0 fires **only** per-kind conservation, naming seven of nine kinds, with the stack profile byte-identical to the healthy run and the item total unmoved at 704.


**And verified 2026-08-05 after the OVER-LIMIT MERGE PASS** — "The merge that would be over the limit", the shape the pass above shipped with a note saying it did not cover. `make check` green, sprite checker green, **all eight suites green in BOTH arms**, and the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) — which is the gate this pass had to clear above all others, because leg F IS the merge leg and a pre-pass that classified every merge would have moved it. It did not: six parts is under the sixteen the 4C bound proves safe, so the pass never reaches a host call there. The `edge` suite gained a two-cluster rig, so its baseline moved to **fifteen clusters over ninety-five parts** and its placement-probe counts to **180–197**; **no other suite's numbers moved at all**. The red proof is in `assert-edge.py`'s BRDG section: with the call at the top of `flushDead` removed, one part placed in a one-tile gap drained 1,044 items out of each of two working balancers, spilled every one of them, took delivery from 186 and 185 items per 246 ticks to eight each, and left the audit reporting `nets=10 drift=0 unbuilt=1` — with the item total conserved to the item throughout, which is why eight suites were green over it. Shipped zip 310,628 → **316,749 B** (+1.97%), `fk_module.lua` 2,432,188 → **2,516,161 B** (+3.45%), members **42, none added**.

**And verified 2026-08-16 after the FAST-REPLACE PASS** ("Fast replace"). `make check` green, sprite checker green, **all eight suites green in BOTH arms from clean**, and the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) — which is the gate this pass had above all others, because the check it adds runs for every belt-connectable built anywhere on the map. It is a `map[key]uint32` point query and allocates nothing, and leg D (a belt laid 18 tiles from anything, 32 B) and leg B (a belt inside the neighbour gate, 352 B) say so.

**No other suite's numbers moved, and neither did the `edge` suite's**, which is the unusual part and was the point of building `frepa` and `frepb` mid-run: its baseline is still fifteen clusters over ninety-five parts and its placement probe is still 191 / 191 / 196 / 197 / 180 / 180 with 0 off a part tile, plus one new `frep` sample at 192. The suite went 4,650 → 5,850 ticks and still runs in about fifteen seconds. Two red proofs, and they fail on different tags: without the prototype line `post-frep-fwd` reports 14 clusters of 95 parts against 14 of 96, and with the line but without `reapFastReplaced` `post-frep-rev` reports 14 of 96 against 15 of 95 — the phantom, with the column still delivering `[264, 264]` because the belt a player laid is inert. Shipped zip 327,613 → **330,778 B** (+0.97%), `fk_module.lua` 2,534,887 → **2,559,428 B** (+0.97%), members **42, none added**.

**And verified 2026-08-16 after the MIGRATION PASS** — "Adopting a Belt Balancer 2 or 3 save", and the ninth suite. `make check` green (bindings and lock unmoved: nine members were bound and not one of them needed a re-pin), sprite checker green at 10 references, **all nine suites green in BOTH arms**, and the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB) — which is the gate a pass that puts a gate on `onEventBody` AND another on `fk_on_deferred` has to clear above all others. **No other suite's numbers moved at all**, and that is expected rather than weak: no save in any other suite contains a `balancer-part` prototype, so both gates are an integer compare and the scan is never entered. The suite is the only one whose two phases run under different mod sets, and it is red-proven three times — without the data-stage stub the engine deletes all 11 entities and the 50-item stack at load; without the marker guard a stranger's 11 entities are converted; and with the stranger case landing in `Done` rather than `Blocked` the scan still leaves them alone while the BUILD path swaps one out from under them, which is the defect the late-build probe was added to catch and did. Proved once against the real Belt Balancer 2 source as well, with numbers identical to the stand-in's to the item. Shipped zip 327,615 → **347,170 B** (+5.97%), `fk_module.lua` 2,534,887 → **2,732,022 B** (+7.78%), members 42 → **51** of 4,257, and three conditional prototypes added. The one thing it needed from upstream is `fk_on_configuration_changed` ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 22), which landed in the same round and is what makes the removed-later case convert at load rather than at the next thing the player happens to do.

**And verified 2026-08-16 once more with BOTH of that day's passes merged**. The two were built on sibling branches from the same base and rebased into one line — fast replace first, the migration on top — and the merged tree was gated from clean rather than trusted from its halves: `make check` green (bindings and lock unmoved), sprite checker green at 10 references, **all nine suites green in BOTH arms** (`make test` 1m10s, `make GC=leaking test` 1m14s, one invocation each), and the leaking arm's seven slopes back **identical to the byte** (1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and 3.92 MiB). The `mig` suite's four legs report exactly what each half recorded alone — 11 parts, 2 surfaces, 3 clusters, `trigger=init` and `trigger=configuration_changed`, 48 copper before and after, 3.997× and 2.995× — and the `edge` suite's fast-replace legs still read 14/95 → 14/96 forward and 14/96 → 15/95 reverse. Merged package, shipped config: zip **349,822 B**, `fk_module.lua` **2,745,246 B**, members **51** of 4,257, 22 events subscribed, 4 defines read; the two halves' own before/after rows above were each measured against a 327,613 B base and add up to this within the two bytes of zip timestamp noise the harness carries. `fk_on_configuration_changed` is FkLua master `6e3eb28`, and `bin/fklua` was rebuilt from it before either arm ran.

**And verified 2026-08-20 after the QUALITY PASS** — "A part at uncommon quality is a part", the pass that closed the four bare-name `find_entity` call sites the migration round wrote up, and the TENTH suite. (The migration round's own three verifications from earlier the same week are recorded in its section rather than here.) `make check` green (bindings and lock unmoved: nothing new crosses the boundary, members **51, none added**), sprite checker green at 10 references, **all ten suites green in BOTH arms** (`make test` 1m34.9s, `make GC=leaking test` one invocation), and the leaking arm's seven slopes back **identical to the byte** — 1,216 / 352 / 1,180 / 32 / 736 / 3,736 / 1,712 B and **3.92 MiB** of linear memory — which is the gate the `restyle` half of the fix was priced against, and it clears it exactly: the one-element slice the area query returns lands in the same allocation size class as the boxed return it replaced. **No other suite's numbers moved.** The red proof is one pre-fix build firing all three headless families at once: 24 `set=0` skin lines where the fixed guest writes 6 real ones, a standing part unregistered under a scripted colliding belt, and an over-limit refusal delivered to nobody. Shipped zip 413,608 → **414,060 B** (+0.11%), `fk_module.lua` 2,746,726 → **2,747,461 B**.

**And verified 2026-08-24 after PHASE 1 OF THE 2.1 PORT** — "One belt per balancer part", the rule Factorio 2.1 forces, and the ELEVENTH suite. `make check` green (bindings and lock unmoved, and `fk_api_gen.lua` **byte-identical** to master's: the capability query reuses the two members the migration's marker probe already bound), sprite checker green at 10 references, both `-gc` arms BUILD, and `m1` and `sedge` green on **2.1.14**. Shipped zip 414,060 → **421,520 B** (+1.80%), `fk_module.lua` 2,747,461 → **2,837,744 B** (+3.29%), members **50, none added**, one conditional prototype added (`bbb-can-stack`, never defined on 2.1). The red proof is one line -- `multiEdgeAllowed()` forced true, which is the pre-port guest -- and it fires fifteen assertions over a run `test/run.sh` had already killed on ten `[BBB] error: create_entity returned nil for bbb-linked-belt` lines, with 64 items on the floor and two balancers delivering 0.000x.

**And verified 2026-08-24 after PHASE 2 OF THE 2.1 PORT** — the runtime-global setting, the grandfather pass, the migration, and the TWELFTH suite. `make check` green (bindings and lock unmoved), sprite checker green at 10 references, and `m1`, `sedge` and `mig21` green on **2.1.14 IN BOTH `-gc` ARMS** -- which phase 1 could not manage and did not claim; the leaking arm was built there and not run — the `sedge` suite's rig rates and its eight audit tuples byte-identical to phase 1's record, which is what says the `refused=` column really was appended rather than inserted. Shipped zip 422,640 → **439,832 B** (+4.07%), `fk_module.lua` 2,837,744 → **3,018,845 B** (+6.38%), members 50 → **53** of 4,259, events subscribed 22 → **23**, no prototype added and one CONDITIONAL setting (never defined on 2.1). The three members are `LuaSettings.global` in its handle-returning form, `LuaCustomTable`'s INDEX-ASSIGN operator — [`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 23, asked for by this feature and landed upstream before it was written — and `LuaSurface.name`, for the `[gps=]` pings.

The `mig21` numbers are in its section above and its red proofs are three, each catching something the other two do not: the condemnation disabled leaves 652 hidden entities standing over 2,320 stranded items and fires eleven assertions; the summary suppressed fires exactly two and moves no other number; and the announce check removed fires exactly one, the one that catches a migration announced with a sentence about an extra piece nobody placed.

**THIS VERIFICATION IS NARROWER THAN EVERY ONE ABOVE IT AND THE GAP IS THE POINT.** The other nine suites cannot run -- 2.1 refuses their manifests, and their rigs are multi-edge -- so **the `mar` slopes were not measured**. That is owed and belongs with the test-estate rebuild rather than with either pass; the "nothing on any hot path moves" claims in "One belt per balancer part" are structural until then. **And the 2.0 arm of phase 2 is owed with the same estate**: the grandfather write landing, both arms of the flip handler and `sweepStackedInterfaces` are unreachable on 2.1 by construction, so what stands in for them is `go test ./edgemode/` over all eighteen states of the fold and `mig21`'s negative over the one thing that must never happen here.

**And verified 2026-08-24 after THE FIRST TRANCHE OF THE REBUILT ESTATE** — `m2`, `mar` and `upg`, whose rigs were re-laid one belt per part. This is the pass the paragraph above says is owed, for the half of the debt it covers. No guest line changed and no package was rebuilt for it: the whole change is three test mods' `control.lua`, three `info.json` version tokens, three assertion scripts and `test/run.sh`'s default. `make check` green; **all six suites green in BOTH arms** (`make test`, and `make GC=leaking test` as one invocation), and the leaking arm's slope table is the first this repo has recorded on 2.1.14: **1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B** per primitive over **3.92 MiB** of linear memory, with the 4×4 term and the linear memory identical to the byte and the three event-path terms unmoved. The collected arm runs 9 collections in 6 paced steps with **0 forward-progress deadlines** on 0.46 MiB and a 10,192 B live set.

**And verified 2026-08-24 after THE SECOND TRANCHE** — `m3` and `edge`, rebuilt single-edge, plus one guest fix the rebuild asked for: the two hand-back lines name the BOUND that refused the piece, because `tellRefusal` and `revertOne` are shared by both bounds and the sentence used to name only the port limit ("The sixty-fifth belt"'s own defect, one level down from `spareMerge`'s line). `make check` green; **all nine suites green in BOTH arms** (`make test`, and `make GC=leaking test` as one invocation each), and the leaking arm's slopes came back **identical to the byte** — 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive, 1,136 B of calibration at 0.0% spread, **3.92 MiB** of linear memory — which is the gate a change to `limPending` has to clear and clears structurally: that slice is empty on every tick nobody's build was refused, and a headless run never appends to it at all. **No suite outside `m3` and `edge` moved by a number.**

Inside them, the property worth stating is what did NOT move: `m3`'s twelve rig rates, spreads and its "exactly zero" are the numbers they have always been, and so are `edge`'s `lim` and `brdg` delivery ratios, its 118-item dissolve, its 128-item port-boundary spill, its 300×5 at 0.00% spread and its `ntch` at 376 376. What moved is geometry — 14 clusters over 59 parts in `m3`, 15 over 198 in `edge` — and four `edge` rigs that are a redesign rather than a re-lay, each carrying a spare part or an ended belt line so that the EDIT it exists to take is still legal. Both suites gained a `nets` assertion beside `unbuilt`, for the reason the `mar` suite gained its world tuple: a cluster with inputs and no outputs is a legitimate half-built state and is never counted unbuilt, so `unbuilt=0` is satisfied by a save in which a rig quietly lost its network. **`m3`'s red proof is exactly that** — put its "unrelated placement" belt back on a working part's face and `unbuilt` stays 0 while `nets` falls to 13 and `noev` delivers 0.000x. `edge`'s is its `brdg` gap tile given a second flanking belt, which measured something new: `compile()` asks the PORT bound first and returns, so a shape that is illegal twice still reads as a port refusal and only the input count in the assertion can see it.

`m2` is unmoved on everything it asserts -- 1,306-item control, 3.998x / 7.994x / 2.996x, 0.15% spread, 2,680 items before and after a forced recompile with all 72 handed back inside -- over **156 parts instead of 77**, which is the statement the whole re-lay exists to make. `upg` adopts all twenty-one networks and rebuilds none. Three red proofs, one per suite, and two of them found something rather than confirming something: turning `pass`'s passing line SOUTH fires six assertions where the pre-port geometry would have cost a rate alone; crippling leg F's rebuild **passed every assertion `mar` had**, which is why `assert-marathon.py` now pins the `(clusters, parts, networks)` tuple each leg audits; and bumping only the mod VERSION in `bump_build` leaves the guest heap adopted and `assert-upgrade.py` failing on a rebuild that never happened, which is `agents/single-edge.md`'s S2 result 6 red-proven in this repo.

**And verified 2026-08-25 after THE FEATURE PASS that is release 0.3.1** -- the portal's first feature request, in three parts: the recipe's cost, the research's cost and the hidden network's belt speed ("Cost, research and belt speed"). `make check` green, with the sixth pure package (`tune`) in it and bindings and lock unmoved -- **no member, define, event or prototype was added**, because everything the three folds need was already bound. `make mod`'s sprite and changelog checks green at 0.3.1. **The eleven-arm dump gate green**: the two goldens, seven variant arms and the speed arm, 26.5 s. **All fourteen suites green in BOTH arms** (2m13s collected, 2m06s leaking, one invocation each), and **no suite's number moved at all** -- which for a load-time-only feature is the only result available, and the standing gate is the `mar` slopes, back **identical to the byte**: 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive, 1,136 B of calibration at 0.0% spread, **3.92 MiB** of linear memory, 0 items lost over 200 teardowns, 681 audits at drift=0.

**THE GOLDEN DIFF IS EXACTLY TWO LINES AND THAT IS THE HEADLINE.** `mod_settings_sha256` moved in both arms, `{}` -> the two `string-setting` prototypes (`ca3d163b...` -> `c1c8c44b...`), and **`data_raw_sha256` and `prototype_list_checksum` are byte-unchanged in both** -- 957bfa81... and 918808251 on `base`, 4ffb0e3b... and 2044872442 on `incumbent`. The recipe, the technology and all four hidden prototypes came out identical to the prototype table this repo has been measured on all along, which is what the defaults are for. A setting PROTOTYPE lives in the settings stage's own Lua state, so it reaches `mod-settings-dump.json` and never `data-raw-dump.json`; that is why one hash moved and the other could not. Three red proofs, in the table at the end of "Cost, research and belt speed".

**And verified 2026-08-25 after PHASE 1 OF THE ESTATE PORT** -- `m1` and `sedge`'s observer mods out of Lua and into compiled guests, with the shared harness, the packaging recipe and the staging seam the other thirteen will reuse ([`agents/estate-port.md`](agents/estate-port.md)). `make check` green, with `go vet ./obs/...` added to it and bindings and lock unmoved -- **no member, define, event or prototype was added to the mod**, and its `fk_api_gen.lua` is **byte-identical** across a forced re-package with `guest/go/obs` present, at 54 members of 4,859, which is the measurement behind "pruning is per wasm module". **All fourteen suites green in BOTH arms** (1m52s collected, 1m57s leaking, one invocation each), and **no suite's number moved at all** -- the standing gate is the `mar` slopes, back **identical to the byte**: 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive, 1,136 B of calibration at 0.0% spread, **3.92 MiB** of linear memory; the collected arm 9 collections in 6 paced steps with 0 forward-progress deadlines on a 10,192 B live set.

**THE GOLDEN-LOG DIFF IS THE HEADLINE AND IT IS EMPTY.** Both suites' create and benchmark logs were captured before a line was ported, normalised on the elapsed-seconds column and the `control.lua:N:` line number and nothing else, and diffed: **all 320 `[BBB]`, `[BBB-TEST]` and `[BBB-SEDGE]` lines are byte-identical, in order** -- 44 + 106 in `m1`, 100 + 70 in `sedge`. What is left over is six lines per log and none of them is behaviour: the run's start timestamp and the free-disk figure; the observer's own control.lua and (for `sedge`) data-stage checksums, which IS the port; `script.dat` growing 223 KB -> 454 KB and 261 KB -> 635 KB, because the observer's guest heap is in the save where its Lua `storage` tables were; and Factorio's own state checksum, which covers every mod's `storage`. **`Checksum of better-belt-balancer` and `Checksum for script __better-belt-balancer__/control.lua` are identical in every log**, which is the member-table hash said from the other end. Two red proofs, one per suite and each firing in its own family: `phase=` renamed to `phaseno=` reports all nine of `m1`'s phases as never having run, and one space inserted into `sedge`'s `tick=` empties all three of its rate windows -- `run.sh` exit 1 both times, and green again on revert.

**And verified 2026-08-25 after PHASE 2 OF THE ESTATE PORT** -- `mar`, `mig21`'s observer and `qual`, which are the first observers with real per-tick STATE and arithmetic ([`agents/estate-port.md`](agents/estate-port.md)). 1,194 more lines of Lua deleted; the estate is **7,357 lines over twenty files**, from 8,524 over twenty-four. `make check` green with bindings and lock unmoved and **no member, define, event or prototype added to the mod**; `fklua api check --from 2.1.16 --to 2.0.77` **clean with 0 findings on all three** new guests (21, 20 and 19 members, one event each), which is what keeps them STAMPED for the running engine rather than gated against it. **All fourteen suites green in BOTH arms** (2m2s collected, 2m19s leaking, one invocation each) and **no suite's number moved at all** -- the `mar` slopes back **identical to the byte**, 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive over **3.92 MiB**, with all seven linearity ratios, the 1,136 B calibration at 0.0% spread and all ten world tuples unmoved, and the collected arm again 9 collections in 6 paced steps with 0 deadlines on a 10,192 B live set.

**THE GOLDEN DIFF IS AGAIN EMPTY, AND `mar`'S IS THE ONE THAT MATTERS.** Five logs, the pilot's two masks and no others: **9,835 + 480 + 204 tagged lines byte-identical, in order** -- 65 + 9,835 in `mar`, 237 and 243 in `mig21`'s two fixtures, 172 + 32 in `qual`. `mar`'s tag set is `[BBB-MAR]` AND `[BBB]`, so those 9,835 include **all 681 `[BBB] heap sys=... alloc=...` probes the MOD wrote** -- the number this suite exists to measure, identical tick for tick under a Go observer. The leftovers are the pilot's own five categories: the timestamp and free-disk figure, the observer's control.lua checksum (which IS the port), the data-stage checksum for the two that have one (`mig21` has none and its mod checksum is unmoved at 0, exactly as `m1`'s was), `script.dat` growing where the observer's guest heap is now in the save (`mar` 240 KB -> 754 KB, `qual` 442 KB -> 939 KB; `mig21` unmoved, because `--benchmark` never saves and its `script.dat` is the committed fixture's), and Factorio's own state checksum.

**The red proof is the PACKAGING this time, not a log line's format.** `mig21` is the first observer whose `info.json` is a correctness surface: it samples the world from its own `on_configuration_changed`, which is the only "before" any script can reach, and whether that runs before this mod's is decided by MOD LOAD ORDER -- so it must not declare a dependency on `better-belt-balancer`, and that is now one `--dependency` flag in the Makefile rather than a hand-written file. Add the dependency back and **five assertions fire with `run.sh` exit 1**, led by the one the observer's own header promises: *"nothing was seeded into the networks before the migration ran ... every item number below would be a vacuous zero"*, over `seeded 0 items` and `0 interfaces on 77 part tiles`. That is the whole chain proved end to end -- a Makefile flag, into a generated `info.json`, into Factorio's ordering, into which handler sees the world first. And all three generated `info.json` files are **field-for-field identical** to the hand-written ones they replace, which is the same statement from the other end.

**One gap found and filed**: `fklua mod` cannot package a mod that has only a data stage (FKLUA-GAPS.md item 26). The speed arm's fixture needs a data stage and nothing else; the workaround is an empty `main` with no exported hook as its control module, confined to `test/fixtures/fastbelt` and reaching nothing that ships.

**What is deferred**: the `release/2.0` golden. The two cost settings are defined on BOTH engines, so when that arm is captured its mod-settings dump will carry these two plus `bbb-multi-edge-parts` -- three settings where this file's older note says one.

**What a future session should pick up first**, in order:

**And verified 2026-08-25 after PHASE 3 OF THE ESTATE PORT** -- `mix`, `plat` and `mig`, which is the phase that consumed the LAST piece of FkLua surface the port was waiting on ([`agents/estate-port.md`](agents/estate-port.md)). 2,121 more lines of Lua deleted; the estate is **5,263 lines over fourteen files**, from 8,524 over twenty-four.

**`fkapi.Log(Value)` -- the bound global `log()`, and the only way anything can read a LuaProfiler's duration -- was verified against a golden line before a rig was written**, which is what the pilot and phase 2 both left as a note for whoever needed it first. A throwaway observer with nothing in it but the idiom rendered `Duration: 0.507333ms` in the same shape, the same unit and the same position in the log as the Lua it replaces; the ledger item is [`FKLUA-GAPS.md`](FKLUA-GAPS.md) 27. The one thing that differs is Factorio's own note of WHERE the `log()` came from -- a host call is made through `pcall` inside the ABI and is attributed to the C boundary, where the plain-string import is answered by a Lua `log()` in the generated control stage -- and nothing anywhere reads that prefix.

`make check` green with bindings and lock unmoved and **no member, define, event or prototype added to the mod**. **Every one of the 1,290 tagged lines across TWENTY logs is byte-identical, in order** -- `mig` alone is sixteen of them, because its seven legs run two phases each under different mod sets and its two name probes run one. **All fourteen suites green in BOTH arms** (2m0.6s collected, 2m11.2s leaking, one invocation each) and **no suite's number moved at all** -- the `mar` slopes back **identical to the byte**, 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive over **3.92 MiB**, with all seven linearity ratios unmoved, which is the gate a phase that retrofitted `obs/mar` and `obs/mardata` onto a shared constants package had to clear.

**The red proof is a new family and it found a hole in the runner on the way.** `log{"", "tag ", p}` renders a profiler; `log{"tag ", p}` looks the tag up as a locale KEY and writes `Unknown key: "..."` instead. Injected, the first run came back GREEN on all five timing lines -- `assert-plat.py` does not read them, and `Unknown key` was grepped in the CREATE log and not in the benchmark's. `test/run.sh` greps for it in the run phase now, in both places that read a `run.log`, and with the gate closed the same injection fails by name and exits 1. Verified safe first: the term appears in no log of any suite in either arm.

**And verified 2026-08-25 after PHASE 4 OF THE ESTATE PORT** -- `m2`, `m3` and `edge`, the three biggest observers in the estate and the LAST of the suites ([`agents/estate-port.md`](agents/estate-port.md)). 3,350 more lines of Lua deleted; the estate is **1,827 lines over eight files**, from 8,524 over twenty-four, and **not one of what is left is a suite**. `make check` green with bindings and lock unmoved and **no member, define, event or prototype added to the mod**; `fklua api check --from 2.1.16 --to 2.0.77` **clean with 0 findings on all three** (25, **48** and 39 members, one event each -- `m3`'s is the widest surface any observer has, covering blueprints, clones, ghosts, robots, surface deletion and rendering), so all three stay STAMPED. **All fourteen suites green in BOTH arms** (2m33s collected, 2m45s leaking, one invocation each) and **no suite's number moved at all** -- the `mar` slopes back **identical to the byte**, 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive over **3.92 MiB**.

**THE GOLDEN DIFF IS EMPTY OVER 4,220 TAGGED LINES, AND THE MASK LIST STOPPED BEING AN ARGUMENT.** This phase took the goldens TWICE -- two runs of the unmodified tree -- and diffed them against each other first: under exactly the three established masks the two are identical across all 4,220 lines, which is a measurement that those three are the ONLY nondeterministic things in these logs and that any later diff is a port defect rather than a candidate for a fourth mask. Then, ported: **364 + 89 in `m2`, 158 + 457 in `m3`, 438 + 2,714 in `edge`, byte-identical, in order** -- and every number the three assertion scripts print is identical too. The leftovers are the pilot's own categories and nothing else.

**And verified 2026-08-25 after PHASE 5 OF THE ESTATE PORT** -- the INTERACTIVE STAGING MOD, which is the first thing the programme has ported that was never a suite ([`agents/estate-port.md`](agents/estate-port.md)). 462 more lines of Lua deleted; the estate is **1,365 lines over six files**, and what is left is the bench harness, two data-stage-only stand-ins and `bbb-flip-test`. `make check` green with bindings and lock unmoved and **no member, define, event or prototype added to the mod**; `fklua api check --from 2.1.16 --to 2.0.77` **clean with 0 findings** on `obs-iact.wasm` (22 members, 1 event, 10 concepts) and on its data module (an empty surface, which is the expected answer for a guest that imports fkdata and never fkapi), so it stays STAMPED -- which matters more here than for a suite, because this is the one package a person installs by hand. **All fourteen suites green in BOTH arms** (2m59s collected, 3m11s leaking, one invocation each) and **no suite's number moved at all** -- the `mar` slopes back **identical to the byte**, 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive over **3.92 MiB**, with 1,136 B of calibration at 0.0% spread and the collected arm at 9 collections in 6 paced steps with 0 forward-progress deadlines.

**The golden diff is empty over 498 tagged lines and the whole masked diff is FOUR LINES**, which is the smallest any phase has produced: the run timestamp, the free-disk figure, and the observer's own two checksums, which ARE the port. `Checksum of better-belt-balancer` (3503679581) and `Checksum for script __better-belt-balancer__/control.lua` (507607469) are identical either side. There is no `Loading script.dat` row at all, because `iact` is the estate's only create-only suite -- nothing is loaded, so no state checksum and no benchmark milliseconds exist to move. The masks were proved by self-diff first, as phase 4 established.

**And this phase consumed the last piece of FkLua surface the port had never used: `fkapi.RemoteCall`.** The charter has carried a note since phase 2 that whoever reached it first should verify it against a golden line, and this mod is the consumer -- it asks `freeplay` to disable the crash site and the intro, which are SILENT calls no golden diff could see fail. A spike, before the port was trusted: `remote.call('better-belt-balancer', 'audit')` comes back `StatusOK` with a `TagNumber` value and the mod's own `[BBB] audit` line in the log beside it, so arguments cross out and a value crosses back; `freeplay` resolves even in a headless `--create`, so these calls really fire; and a missing METHOD is `StatusCallFailed` rather than a raise, which is what lets the Lua's two guards (`if remote.interfaces[...] then pcall(...)`) collapse into one status check. `LuaForce.Chart` and `LuaForce.AddChartTag` were spiked in the same run, because the player block uses both and nothing headless can enter it.

**Three things phase 4 pinned that nothing else could.** A profiler that spans a TICK BOUNDARY needs `Object.Retain` -- `m2` times a recompile across one by construction, because the mod batches, and a handle that is not retained is valid only inside its own dispatch (five `[BBB-OBS] error:` lines, exactly one per tick-crossing window). `m3`'s churn LCG has to be transcribed in FLOATING POINT: `seed * 1103515245` reaches 2.4e18 against a double's 53-bit mantissa, so Factorio's Lua rounds the low nine bits off every product and every seed it produces is a multiple of 512 -- a `uint64` transcription is a DIFFERENT generator that diverges at the first value, checked against `../FkLua/bin/lua52f` over 600 calls. And `fkapi`'s `Event*` constants are FkLua's SUBSCRIBE indices, not Factorio's event ids, so `script.raise_event` has to be handed `script.get_event_id(name)` (FKLUA-GAPS.md item 28, which is also where `fk.LastError()` is verified: `edge` asserts the engine's refusal of `on_player_mined_entity` verbatim, and it comes back byte-identical).

**TWO RED PROOFS, AND BOTH SAY THE SAME THING ABOUT WHERE THE TEETH ARE.** Perturbing an `m2` profiler LABEL came back GREEN, because `assert-m2.py` read the timing lines only to PRINT them -- phase 3's `Unknown key` hole one level out, and a whole block that could not fail. `WANT_TIMINGS` closes it and the same injection then fails by name. And swapping `m3`'s float64 LCG for the textbook `uint64` one moves the churn completely -- compiles 141 -> 153, teardowns 75 -> 87, `create_entity` 921 -> 1,001, `stress recovered` 15,856 -> 15,848, **425 of 457 run-log lines differing** -- while `assert-m3.py` PASSES, as it should, because CLAUDE.md tracks those counters as measurements of the guest that legitimately move when its batching changes. **For a port the golden diff is the instrument, and this is the clearest evidence in four phases that it belongs in the gate list rather than beside it.**

**And the package-time jump-span note is RETIRED.** Every phase since the pilot recorded it as "not fired yet, and `edge` is the first place likely to meet it". It does not meet it: `edge`'s `fk_on_init` builds fifteen clusters over 198 parts and packages with no `//go:noinline` at all, against a threshold of 655,355 bytes of jump span whose own documentation puts the widest span across all of FkLua's guests at 248,861. The one performance finding is `edge`'s own guest heap crossing 16 MiB, which cost a 725 ms `memory.grow` stall -- benign by the charter's rule (an observer runs for seconds in a world that is thrown away) and traced, after one wrong hypothesis that was measured and discarded, entirely to `EntityType` copying a host string per entity per sweep. `type_is` compares on the host: 0 rungs, 350 ms. What did not move is the wall clock, and that is the honest residual -- 40 s against the Lua's 18 s, which is the boundary rather than the heap.

**The standing programme is [`agents/estate-port.md`](agents/estate-port.md): the test estate out of Lua, one phase at a time.** Phases 1 through 4 all landed 2026-08-25 -- `m1` and `sedge` with the harness, the packaging recipe and the `copy_testmod` staging seam every later phase reuses; then `mar`, `mig21`'s observer and `qual`; then `mix`, `plat` and `mig`, which consumed the last piece of FkLua surface the port was waiting on and built the shared constants package; and then `m2`, `m3` and `edge`, the three biggest, which is **every suite that can be ported on this machine**. **Phase 5 landed the same day and took the interactive staging mod**, which was never a suite: `make interactive-install` builds and installs the very package `iact` stages, and `fkapi.RemoteCall` -- the last piece of FkLua surface the port had never used -- is verified. What is left is **phase 6's two data-stage-only stand-ins** (whose first problem is that they have no golden of their own: they are data stages inside `mig`'s and `mig21`'s runs, so those are the goldens to take), phase 7's bench harness -- which needs a comparability gate the suites do not, because every published performance figure in this repository was measured with those mods -- and phase 8's Rust parity exercise. `flip` waits for a 2.0 binary: its suite SKIPS on 2.1, and a phase's first gate is a golden log there is no run here to produce. **Read the "gates a phase must clear" section before writing a line**, and in particular capture the goldens first -- TWICE, which is what phase 4 found turns the mask list from an argument into a measurement -- because they are the one thing that cannot be recovered afterwards.

0. **The rest of the 2.1 port**, which is now the biggest thing outstanding and is chartered in [`agents/single-edge.md`](agents/single-edge.md). Phases 1, 2 and 3 are done -- the rule, its refusal, the setting, the grandfather pass, the migration, the `sedge` and `mig21` suites and the interactive and demo worlds -- and so are phases 4 and 5, the estate's two tranches: **`m2`, `mar`, `upg`, `m3` and `edge` are rebuilt single-edge and green in both arms**, which restored the `mar` slopes and the leaking arm as gates and paid off the "structural rather than measured" half of the earlier phases' claims. What is left is **`plat`, `mix`, `mig` and `qual` rebuilt the same way** -- `mig` is the one that needs real design work rather than a re-lay, because every adopted incumbent balancer is multi-edge by construction and so converts and is then refused -- then the re-captured portal GIFs, and the `release/2.0` branch. **On that branch and nowhere else** live the 2.0-only legs phase 2 could not run: the grandfather write actually landing, both arms of the flip handler, and `sweepStackedInterfaces` against a standing multi-edge world. Each phase's own status section is the record of what is done and of where its design needed correcting.
1. **A licence.** Nothing can be released without one, and FkLua is in the same position — the two want the same answer.
2. **The icon's hard edge, if it is ever worth a round trip.** The art is in (see the Status table). The one measured deviation left is that the icon carries exactly two alpha values, 0 and 255, where all 18 base-game building icons sampled carry 5-26% partial alpha on their silhouette. Its COVERAGE is in family -- 93.4% opaque against a transport belt's 85.6% -- so the ask, if made, is antialiasing alone and not a redraw. It costs a slightly cut-out edge at native 64 px and nothing once the GUI scales it down. Polish, not a defect.
3. **The module size, which is now the whole cost of the `-gc` decision.** `fk_module.lua` was 1,899,407 B under the shipped `--gc=collected` when that decision was taken and is **2,265,404 B** as of 2026-08-03 (the zip, **291,364 B**); it is +28.7% on the leaking arm's 1,760,312 B, and "The third decision" says that is the only thing left on collected's side of the scale. Nothing in this repo moves it — it is the collector's own code, emitted — but it is what a mod-portal release will be judged on, and it is the first thing to re-measure if upstream's emitter changes.
4. **The six unverifiable paths** in "What M3 implements and does NOT verify" — undo/redo application, the two player-rotation events, the space-platform build/mine events, the miner's pocket's TRIGGER, the over-limit feedback's and, since 2026-08-16, the FAST-REPLACE cursor. Each needs a player or a platform hub that a headless `--create` does not have; none is a code gap. The pocket is the one a user can check in thirty seconds of interactive play and it is worth doing before a release. Two gestures, one per field report: mine a balancer that is carrying items **part by part** and watch where they go at every step, and **lay a belt on the spare edgeless part attached to a running balancer and mine it off again**, which is the edit that halves the machine (under the one-belt-per-part rule a working balancer has no free face, so the spare part is what the rig stages for it). The over-limit one is a third gesture in the same session: lay a sixty-fifth belt against the spare part of a sixty-four-belt balancer and check that the text, the sound and the returned belt all arrive and that the balancer never stops ("The sixty-fifth belt"). A fourth, for the merge shape: put a part in the one-tile gap between two balancers whose belts add up to more than sixty-four and check that BOTH of them keep running and that the part comes back to your inventory ("The merge that would be over the limit"). A fifth, for FAST REPLACE, in both directions: drop a balancer part onto the END of a belt line and check that the belt and its cargo arrive in your inventory, then lay a belt on the MIDDLE part of a five-part column and check that the PART does — and that the same belt is refused over a part carrying an edge interface ("Fast replace"). **All five gestures are pre-staged now, and all five rigs are single-edge since 2026-08-24**: `make interactive-install` puts a rig-staging mod beside the real one, a fresh world spawns you next to the five rigs with the pieces in hand, and [`test/interactive/README.md`](test/interactive/README.md) is the checklist with the log line each gesture must produce. That mod also stages the FIVE MOD-PORTAL DEMO SCENES, so a capture is reproducible; the `iact` suite gates the whole world with one `--create`. **A sixth gesture is the SINGLE-EDGE refusal**, which shares band B's rig: lay a belt against a part that already has one and check the same text, sound and hand-back the sixty-fifth belt gets. **A seventh gesture joined that checklist on 2026-08-16 and it needs no player at all**: swap a real incumbent out of a real save and look at the result, and place one of that save's old blueprints so a robot revives a legacy ghost. Everything else about the migration is headless, including a run against the real Belt Balancer 2; what is behind the wall there is a construction network and a graphical client, not `game.get_player`. **An eighth is opening a real Factorio 2.0 multi-edge save on 2.1**, which also needs no player: check the chat summary, click the `[gps=]` pings, and look at the stopped balancers and the items on the ground beside them. The `mig21` suite pins every number over two committed fixtures; what is behind the wall is whether the message reads well and whether a ping lands where it says. **A ninth is the WAKE RACE, and it is the one gesture no staged rig can set up**, because what it needs besides a player is a `make install` over the same version on a save that already has the balancer: the guest then wakes on a fresh heap with the world already built, and the gesture is to make the sixty-fifth belt (or the bridging part) the FIRST thing done in that session. Expected — one message, the right one, and the piece back. See "The wake race", which is also where the rest of the 2026-08-05 playtest's findings are. **Those reports are also the standing warning about this list**: the pocket sat on it for one commit, a player found the defect, and two of the three things "unverifiable" was covering turned out never to have needed a player at all — then the same thing happened a second time to the belt at the edge, which was not on the list at all because nobody had noticed it was the same removal. Before adding anything here, ask which half of it is actually behind the wall. **And the playtest is the other half of the same warning**: most of what it found was not a path nobody had written, it was a path nobody could SEE — a message in the wrong words, in the wrong colour, seventeen tiles off the screen, or sent one tick before it became untrue. No assertion this repo can write is about any of that.

**M1 is done.** The mod skeleton, the visible `bbb-balancer-part` entity and the cluster registry in the Go guest, verified headlessly.

**M2 is done.** Clusters compile into hidden splitter networks and the exit criterion is met: saturated 4x4 and 8x8 rigs balance to 0.15% spread at 99.9% of one belt per output, and starvation, blocked outputs, asymmetry, recompile under load and cross-surface all hold (table above). What M3, M4 and M5 inherit:

- **The network is a butterfly over P = next_pow2(max(N, M)) lines**, log2(P) stages of P/2 splitters, rows permuted between stages so each stage's pairs land on adjacent rows, and a row that has to move does so through a linked-belt jumper pair. There are **no belts between the pieces** -- every element sits directly against the next -- which took a 4x4 from 50 entities to 32 and an 8x8 from 132 to 84. Spare output ports loop back into spare input ports where there are enough of them and dead-end otherwise.

| network | entities | of which visible |
|---|---|---|
| 1->1 | 5 | 2 |
| 2->2 | 11 | 4 |
| 4->4 | 32 | 8 |
| 8->8 | 84 | 16 |
| 3->5 | 72 | 8 |

- **Invariant: at most one linked belt per tile SIDE.** Two edges may share a tile -- S1 ran four on one at full rate -- but two same-direction inputs on a tile leave one of them silently dead. `TestOneLinkedBeltPerTileSide` pins it.
- **Invariant: every teardown for an event runs before any build.** A cluster that split leaves the old network's visible interfaces standing on tiles that now belong to a different cluster; a build that ran first would classify them as part of the world.
- **Retired invariant: "a mined entity is still valid during the event that reports it".** M2 and M3 compiled inside the event that reported a removal, so the classifier carried a one-position blind spot -- the belt on its way out -- armed and restored by every flush. **`fk.Defer()` retired the whole mechanism** and the reasoning is worth keeping: the engine destroys a mined entity when its own dispatch RETURNS, so a flush that happens on the next tick re-reads a world the belt is simply not in. There is nothing to ignore, and the case the window could never handle -- a mod raising `script_raised_destroy` and then not destroying -- now comes out right instead of wrong. See `compile.go`.
- **Recompile is full teardown and rebuild**, plus a fingerprint over the edge list so that a change which does not move an edge rebuilds nothing. Entity-diff minimisation is M3+.
- **Recompile cost is boundary-bound, not algorithm-bound.** A 4x4 rebuild is 4.4 ms against 0.13 ms for the same 32 `create_entity` calls made straight from Lua; an 8x8 is 9.6 ms. (Of an EMPTY network. One carrying items also pays to put them back — see "A recompile is not a removal", which is the same boundary at the same ~12.6 µs per call.) That is ~350 host calls at ~12.6 us each, and that 12.6 us is the tier-2 encode on the Lua side -- see the `--persist` section for what does and does not move it. The planner itself is microseconds and allocates nothing on a warm buffer.
- **Items are never deleted, and since 2026-08-02 they are not put on the floor either.** Teardown reads every transport line in the slot; a RECOMPILE puts the total back inside the network it rebuilds and only a REMOVAL spills it beside the cluster (`spill_item_stack`, item by item -- `drop_full_stack = true` was measured placing nothing at all). See "A recompile is not a removal" below. What is not recovered either way: fractional item positions, anything a splitter holds outside its transport lines, and how the items were STACKED.
- The hidden surface is created lazily by name, generates no chunks (S1: entities on ungenerated chunks run at full rate; re-verified here), and is carved into 32x72 slots keyed by cluster root id. `MaxPorts` is 64; beyond it a compile is refused loudly rather than overrunning its neighbour's slot -- and the refusal happens BEFORE the teardown, so the balancer that was already there keeps running ("The sixty-fifth belt").
- The hidden belt tier runs at **speed 0.25**, which M2 recorded as a ceiling and which is a FLOOR since 0.3.1: a belt above it runs fully compressed and delivers 480 x speed items/s, so there is no engine ceiling there at all and a modded belt faster than 0.25 was being throttled. It is enough by construction for a vanilla game -- P >= N means no hidden line ever carries more than one visible belt's rate, and nothing in base or Space Age reaches 0.25 -- and `deriveHiddenSpeed` raises it to the fastest belt installed. See "Cost, research and belt speed".

**M3 is done.** Every 2.0 lifecycle path that can change what the compiler compiled from is handled and kill-tested (table above), and the exit criterion is met: **the stale-reference crash class is impossible by construction, not by handling.** The rules that make it so, and which nothing here may break:

- **No entity reference outlives the event it arrived in.** Persistent state is a surface index, two integer tile coordinates, a force index and a slot number. Nothing dereferences a `LuaEntity` or a `LuaTransportLine` from a previous tick, because none is kept. That is not a discipline applied to a design that could go either way -- it is why every recovery path below can be written at all: recovering from a surprise is *re-reading the world*, never repairing a graph of handles. (`get-by-unit-number` is declared on the part prototype and is still unused; there has been no need.)
- **Clusters are per force**, and adjacency, the compiler's flood fill and the edge search all agree about it. Two forces' parts touching are two balancers; a belt of another force is never an edge, because `find_entities_filtered` applies the force filter in C++ and never returns it. This is the semantics the part prototype always claimed and the code did not have.
- **Every recompile is from the visible world.** The registry says which tiles are parts; everything else -- which belts, facing where, of what force -- is re-read at compile time.

### The failure envelope

The honest statement of what happens when nothing tells us. Another mod calling `entity.destroy()` on a belt beside a balancer raises **no event of any kind**, and no amount of subscribing can change that. What follows:

| | |
|---|---|
| **What breaks** | Nothing, immediately. The network is engine state: its linked belts do not care that a visible belt vanished, so items simply stop arriving on that port. The other ports keep balancing exactly. |
| **What is wrong** | Only the guest's fingerprint, which still describes an edge that is no longer there. A rebuild triggered for another reason would place an interface facing nothing, which is inert. |
| **What recovers it** | Any event that touches the cluster -- a belt laid within two tiles, a part added or removed, a rotation, a clone, an undo, a surface event. The edge list is re-derived from the world every time, so the first such event is correct. Also: any `bbb-audit` marker, which re-classifies every cluster and reports the drift before repairing it. M3's `noev` rig is exactly this sequence and comes back to 1.978x. |
| **What never happens** | A crash, a desync, or a duplicated network. There is no reference to go stale and no state that can disagree with itself. |

The same envelope covers a belt whose `direction` is assigned directly (what an undone rotation does to the world), and a mod that swaps a belt for another tier without raising. **A bot upgrade needs no handler at all** for the same reason: whatever the removal path was, the *build* event for the replacement fires, and that recompile reads the world as it now is. This was checked rather than assumed -- M3's `swap` rig fast-replaces an express belt with a fast one, which raises a build event and no mine event, and lands at exactly 1.667x.

`on_object_destroyed` is deliberately **not** used, though `agents/design.md` listed it. It requires registering every belt of interest by unit number, which is the per-entity bookkeeping this design exists to avoid, and it fires *after* the fact carrying only a registration number -- so it could not even say where to recompile without a unit-number-to-position table, i.e. a cache.

### Coming back on a heap this build did not write

The guest heap is discarded whenever the mod is rebuilt, and this mod deliberately keeps it that way. **It exports `fk_migrate` and must never export `fk_migrate_adopt`.** Those used to be one hook: exporting `fk_migrate` meant adopting the previous build's **entire linear memory**, `.rodata` and all, so a guest that exported it was reading its own string constants out of another program's image. Upstream split them ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 13, fixed) -- `fk_migrate(old_version)` is a **notification on a fresh heap** and `fk_migrate_adopt` is the opt-in that really hands the bytes over. This guest's state is Go maps and slices reachable only through package-level roots, which is exactly the shape adoption cannot carry across a rebuild, so the adopt half is permanently off the table.

So a mod update starts with an empty registry and a world full of parts and running networks. Nothing breaks in the meantime, and that is the point: **a compiled network does not need a script to keep running.** What must not happen is the guest deciding a cluster has no network and building it a second one.

Two mechanisms, and the second is not redundant:

- **`fk_migrate` names the moment.** It fires from `on_configuration_changed` -- after `on_load`, before the first tick, with `game` fully available -- so the rebuild happens at a deterministic point rather than inside whichever event happened to arrive first. The `upg` suite asserts the hook ran *and* that it, rather than the fallback, drove the rebuild.
- **`registryReady` stays** as the fallback: a plain bool that is false in a freshly initialised heap, so the first event of any session rebuilds before it decides anything. It covers what the hook does not -- a mod added to a save that already contains parts, a `--persist` mode changed underneath it. One bool test on the hot path, and it is not going away.

The rebuild **adopts** rather than rebuilds wherever the evidence is complete. For each cluster it finds the visible interfaces standing in the cluster's box, follows one of them to its hidden partner (whose position *is* a slot number), and compares the set of (tile, direction) it found with the edge list it just re-derived. An exact match means the network is correct -- no tick has passed since it was built -- and the cluster is adopted for about thirty host calls instead of a teardown and ~350. Anything less falls back to a rebuild. Slots no cluster claims are then swept, which is also where a hidden entity orphaned by a prototype rename would go: our four prototypes can only be renamed by a version of this mod, which is a rebuilt guest, which is exactly this path.

**`game.surfaces` binds now** ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 15, fixed upstream): a dictionary return whose key is a dyn value comes back as an ordered pair slice, and `pairs()` over this one yields the surface NAME, so the hidden surface falls out of the same walk instead of costing its own `get_surface` by name. The scan is **one host call plus one `index` read per surface** -- two on a base save. It used to probe `get_surface(1), get_surface(2), …` and stop after 64 consecutive misses: ~65 host calls on that same save, and a guess about how sparse an index can get.

**The list is sorted by index before anything walks it**, which the probe gave for free and this does not. `fk_abi.lua` explicitly declines to promise an iteration order for a dictionary return -- it walks `pairs()` -- and surface order decides the order parts are registered, which decides node ids, which decide cluster roots and slot claims. Two clients of a lockstep game disagreeing about that is a desync. It is an insertion sort because a big save has a dozen surfaces.

### How many recompiles a batch costs, and why it is now one

**`fk.Defer()`, and the bound is one build per affected CLUSTER per tick** -- not per entity, not per part. `fk_on_deferred` is a one-shot `on_tick` the host registers when the guest asks and tears down again from inside the flush, so an idle guest still pays zero registrations and zero per-tick calls; that is the M4 measurement this could not be allowed to break, and it did not ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 12, fixed upstream).

**The split is: the registry updates inside the event, the compile does not.** `AddPart`/`RemovePart` read nothing from the world -- the tile and the force are already in hand from the event -- so they cannot be deferred and are not; an entity is valid only inside its own dispatch. What is queued is a cluster id, deduplicated, with no host call behind it.

Measured, not claimed (`test/assert-m3.py` counts `compiled cluster` lines between markers, and asserts both halves):

| | before | after |
|---|---|---|
| 12-entity blueprint paste, 2-part balancer | 2 builds | **1** (0 inside the paste tick, 1 by the flush on the next) |
| `--create` of the n=200 k=4 bench save | 800 compiles | **200** |
| 100 belts pasted along a finished balancer's edge | up to 100 classifications | **1** |

The last row is where the win actually lives, and it is why `onNeighbour` matters as much as `onPart`: a belt laid next to a cluster used to cost a full edge re-classification (~16 `find_entities_filtered` for a 4×4) that the fingerprint then threw away 99 times out of 100. The paste row is the one that is easy to under-sell -- 2 → 1 looks small because a 2-part balancer is small.

Two things that did not change:

- Built parts-first (the M2 rigs), a cluster still becomes buildable partway through and the bound is per tick, so a rig built across several ticks costs one build per tick that moved an edge.
- A cluster whose edge list did not move rebuilds nothing at all -- the fingerprint still skips, and skips are still most of a churn run.

**`markDead` and `markLive` are O(N²) in the queue, deliberately, and the bound is worth writing down rather than removing.** Both dedupe by scanning the queue linearly, so queuing N clusters in one tick costs N²/2 `uint32` comparisons. N is not the number of clusters in the save, it is **the clusters touched in ONE tick**, and the queue is truncated to `[:0]` by every flush. The worst case this mod has a name for is the n=200 bench `--create`: 200 clusters in one dispatch is 20,000 comparisons, which is a handful of microseconds against the 45.8 s that create takes — three orders of magnitude below the compile it is deduplicating for. A map would allocate per tick on the guest's hottest path to save that, which is the trade "The heap diet" spent a milestone learning to refuse. **Left as it is; revisit only if something starts queuing thousands of clusters in one tick, and re-measure rather than re-argue.**

**The one-tick latency is real and is stated rather than hidden.** A network appears on the tick after its last part, and anything that has to measure the result in the tick it caused it -- which is every item-conservation check in `test/` -- has to force the flush. The shipped `bbb-audit` marker is the synchronous escape hatch and it is what the test mods use; it is also what their `on_init` uses, because `--create` never reaches a tick at all and the compiling would otherwise land on the first tick of the benchmark.

### Which events reach the guest at all

`script.on_event(id, handler, filters)` applies its filter list **in C++ before the handler runs**, and `fk.subscribe` carries one now ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 3, fixed upstream). All eleven per-entity subscriptions use it, with the same five-term list:

| term | covers |
|---|---|
| `{filter = "transport-belt-connectable"}` | every belt that can be an edge — `classifySide`'s six types — **and all four of our own hidden prototypes**, each being a clone of a base belt-connectable |
| `{filter = "name", name = "bbb-balancer-part"}` | the part, a `simple-entity-with-force` and therefore not belt-connectable |
| `{filter = "name", name = "bbb-audit"}` | the marker, a `simple-entity`, same reason |
| `{filter = "name", name = "bbb-insert-probe"}` | the insert probe, the same |
| `{filter = "name", name = "balancer-part"}` | the INCUMBENT'S part, which this mod's data stage keeps alive as a stub once the incumbent is gone ("Adopting a Belt Balancer 2 or 3 save"). A name filter for a prototype that does not exist is accepted and matches nothing, measured, so the list does not branch on the mod set |

**Five terms, not the dozen names and types the obvious version needs.** `transport-belt-connectable` is a category the filter grammar has built in, and finding it is the difference between a filter list that has to be revisited every time a prototype is added and one that does not. Verified against the pinned `runtime-api.json`: all eleven of this mod's per-entity events accept it, and `on_player_rotated_entity`/`on_player_flipped_entity` accept no filters at all and keep the in-guest gate.

What this removes is everything else on the map — an assembler placed, a tree mined, **a biter killed**. Each of those used to enter the guest and pay a dispatch plus three host calls (`name`, `position`, `surface_index`) to be rejected. `on_entity_died` is the one that mattered: on a map under attack it is the highest-frequency event there is, and none of it was ever ours.

**The in-guest name read stays, and that is not a half-measure.** The filter says "one of the things this mod cares about"; the handler still has to know *which* — a part, the marker, a piece of our own network being copied or destroyed, or a belt near a cluster. What changed is how often that is paid.

**And WHEN it is paid moved once more, for a reason that is about the heap rather than about the clock.** `onEvent` reads `position` and `surface_index` first now and buys the `name` last, because a name comes back as a Go string — `getStr` copies the host's bytes, since the arena under them is released when the call returns — and under `-gc=leaking` that copy is permanent, 32 B for one `express-transport-belt`, in the save and in every multiplayer join. A DISAPPEARANCE or a ROTATION can only concern this mod if it is on a registered tile, inside the two-tile neighbour gate, or on the hidden surface, and all three are answered from guest memory — so those events do not buy a name at all. An APPEARANCE still buys it unconditionally and always will: a part placed alone in the middle of nowhere has no neighbours to be recognised by. Measured at half of the largest-multiplier term this guest has; see "The marathon save".

### `on_forces_merged` — the one event with no per-entity form at all

`game.merge_forces(source, destination)` moves every entity of one force onto another and then destroys the source force, and it raises **no per-entity event for any of it**. Nothing else in this guest can see it, and clusters are per force, so without a handler:

- every node of the source force keeps a `pforce` naming a force that no longer exists — and that index is what `classifyEdges` hands the engine as a filter and what `createArgs` puts in a `create_entity` table;
- two clusters that touch and were two balancers only BECAUSE their forces differed are now one balancer, and the registry still says two. Two networks over tiles that belong to one cluster is the shape "Two bugs M3 found" is about, twice;
- a belt of the source force beside a balancer of the destination force was never an edge and now is — anywhere on the map.

`lifecycle.go`'s handler takes the **MERGED** half rather than the merging one: `on_forces_merged` carries `source_index` as a number, which is the only thing the registry needs, and it fires after the transfer, when the world already agrees with what the handler is about to write down. The order inside it is the same order `reconcileArea` uses and for the same reason — **a network has to come down while `nets` is still keyed by the root that owns it**, because a cluster absorbed by a merge stops being a root, `liveRootList` stops returning it, and its hidden network and its visible interfaces would stand for the rest of the session with nothing left that knows where they are.

It then remaps, re-derives every component of the surviving force by flood fill in node id order (the remap can only ADD adjacencies, so a fill per unvisited node is complete, and the smallest id in each component is the root exactly as a split does it), and rebuilds. **The rebuild sweep is over every cluster of that force rather than only the ones near the merge**, because of the third bullet above; most of them are fingerprint skips, and a merge is an administrator's keypress. The `edge` suite drives it with both forces' networks full: six clusters over sixteen parts afterwards, `drift=0`, both halves still delivering, and the items conserved.

Two caveats worth carrying:

- **The filter list is a load-time cost, not a per-event one.** It crosses as a tier-2 value and is decoded once, at subscribe time, inside `_initialize`.
- **The event ids must stay literal constants at the call site.** FkLua's constant scan prunes 218 event descriptors to the 22 this guest names; an id it cannot prove ships all of them, which is a bigger mod and nothing tells you. A loop over a slice of ids would compile and would be wrong.

### Which FIELDS reach the guest — the undo mask

A filter decides whether the guest is entered. Once it is, the encode is **eager and complete**: every field of the payload is marshalled before the handler runs. That is the right trade for a flat event — the mean one has under five scalar fields and a host call per field would cost more — and it is the wrong one for exactly one event this mod subscribes to.

`on_undo_applied` carries `actions`, an unbounded array of tier-2 dynamic values: an undo of a 200-entity blueprint deep-copied every `BlueprintEntity` in it across the boundary so that this guest could read one `uint32`. Upstream measured that dispatch at **7.49 ms**.

`fk.subscribe` takes a **field mask** now, resolved once at subscribe time exactly as a filter is, and the two subscriptions carry it:

```go
fkapi.SubscribeMasked(fkapi.EventOnUndoApplied, fkapi.SkipOnUndoAppliedActions)
fkapi.SubscribeMasked(fkapi.EventOnRedoApplied, fkapi.SkipOnRedoAppliedActions)
```

**7.49 ms → 2.7 µs on the same subscription** ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 14, fixed upstream). Three properties make it safe to use rather than merely fast, and they are why the hand-derived offset could be deleted instead of kept as a belt-and-braces:

- **The layout does not move.** A masked container is written as `(ptr, count) = (0, 0)`, which is a reading every generated decoder already produces, so `player_index` keeps the offset the guest was compiled against and `fkapi.ReadOnUndoApplied(ptr).PlayerIndex` is now the obvious call.
- **A masked field is written EMPTY, not skipped.** The scratch buffer is reused across dispatches; leaving the bytes alone would show the guest the *previous* event's data.
- **Only optionals and containers are maskable**, and a `Skip…` constant exists only for those. Being wrong about a mask costs a value that reads absent, never a zero indistinguishable from a real one.

### `defines.*` — asked for by name, never written down

`gen-bindings` emits a `Defines<Path>()` accessor per define value now ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 11, fixed upstream), and the four cardinal directions are read through it:

```go
dirOf = [4]uint32{
    fkapi.DefinesDirectionNorth(), fkapi.DefinesDirectionEast(),
    fkapi.DefinesDirectionSouth(), fkapi.DefinesDirectionWest(),
}
plan.SetCompass(dirOf[0], dirOf[1], dirOf[2], dirOf[3])
```

**There is no constant to generate and that is the design.** A define's number is Factorio's own and is not stable across versions, and it is not in `runtime-api.json` at all — the description carries names and an order, not values. So the generated table carries the dotted PATH, `control.lua` resolves it against the running game once at load, and the guest holds a per-build id. The accessor caches on first use: one host call for the life of the mod.

**Each accessor is called directly, and computing an id would silently ship all 1137.** The pruning machinery is the same constant scan that prunes members and events — a scan for a literal reaching the `fk.define` import — so a table of ids, a loop, or an offset would compile, would work, and would add ~45 KB of paths to every save. Named directly, `fklua mod` reports **`API: 4 defines read, of 1137`**, and the packaged `fk_api_gen.lua` carries four lines.

`guest/go/plan` has no `fkapi` import, which is what lets `go test ./plan/` prove the balance property under an ordinary toolchain, so the compass is pushed into it by `initBuffers` rather than pulled. `plan.Opposite` was `(d + 8) & 15` and is now a lookup on the installed compass: the arithmetic was one more thing assumed about numbers this package no longer knows.

### Semantics fixed at M3

- **Clusters are per force.** Documented above; the hidden network is built with the cluster's force index, and the edge search is filtered by it.
- **An area or brush clone always recompiles the clusters in its destination box.** The reconcile brings them down (returning their items), destroys anything of ours the clone copied (whose contents were minted by the clone and must NOT be returned), reconciles the registry against the world -- including parts that `clear_destination_entities` destroyed without an event -- and rebuilds. It is a superset of what the per-entity `on_entity_cloned` handler already did, and it is the layer that catches what that handler does not see.
- **A surface being deleted or cleared is handled in the PRE event**, where the parts and the hidden entities are still valid, so slots are freed and items handed back. The POST event exists for one case: the surface that just went was the hidden one, and every network has to be rebuilt on its replacement.
- **Items are lost only to fractional positions and splitter internals.** Measured at 0.90% over ~100 teardowns under M3's randomised churn — and at **0.00% over 200 teardowns of a network deliberately kept FULL** by the `edge` suite, which is the sharper version of the same claim: what M3 loses is the fractional positions of a network in motion, not a fixed tax per teardown. **Where they go was fixed later** — see "A recompile is not a removal".
- **Nothing on the hidden surface is ever a balancer.** `AddPart` refuses a part whose surface index is the hidden one, whatever put it there. The guard is in the registry rather than in a caller because the consequence is item loss a player cannot see: `teardownNet` spills a network's contents beside the CLUSTER, so a cluster registered there would hand its items back to a surface nobody can reach — and it would give that cluster a bounding box inside the slot grid, which a teardown sweeps. Exercised by the `edge` suite.

### Two bugs M3 found, both silent, both from the same shape

Recorded because the shape will recur: **a cluster that keeps working while being wrong, because its own edge list never changed.**

1. `collectCluster` -- the compiler's flood fill -- had no force check when the registry's flood fills got one. A second force's cluster therefore had a bounding box that swallowed the first's, and the first's visible interfaces were destroyed by the second's very next teardown. The victim's fingerprint still matched, so it never rebuilt: it delivered **nothing at all** for 450 ticks until an unrelated event happened to rebuild it.
2. `find_entities_filtered` returns everything whose bounding box *touches* the area, and a 1x1 entity on tile n occupies exactly `[n, n+1]` -- so the obvious sweep box (`right_bottom = x1+1, y1+1`) also returns everything on tiles `x1+1` and `y1+1`. `setSearchBox` insets by a tenth of a tile. Clusters are adjacent whenever two forces build against each other and diagonally whenever anyone builds an L.

The lesson for M4 and beyond: **the fingerprint is a statement about the belts AROUND a cluster and says nothing about the interfaces ON it.** Anything that can remove an interface must put its cluster on the teardown queue, not just the build queue. That is why a build or death event naming one of our own prototypes -- which the compiler never raises, so it is always somebody else -- forces a rebuild rather than a re-classification.

### What the fingerprint covers, and the one thing it deliberately does not

FNV-1a over the edge list: for each edge, the CLUSTER TILE it sits on, the direction the interface must face, and whether it is an input or an output; then the count. A cluster tile plus a side is in bijection with the neighbouring belt tile, so position is covered; direction and in/out-ness are covered directly, so a rotation, a flip, an underground's ends being swapped and a belt appearing or vanishing all move it.

**The belt's prototype NAME is not in it, and that is the right answer rather than an omission.** The interface the compiler places sits on the cluster's own tile, not on the belt's; its whole contract with the belt is *a belt-connectable is there, facing that way, on that force*. A belt swapped for another tier at the same position and direction needs no change to the network at all -- and `P >= N` means the hidden lines have capacity for a faster one.

M3's `swap` rig is the evidence. Fast-replacing an express input belt with a fast one raises a build event, the cluster is re-classified, the fingerprint matches, and **nothing is rebuilt** -- and throughput moves from 2.000x to exactly 1.667x, which is (1 + 0.0625/0.09375) belts, purely because the belt itself is slower. The network did the right thing by not changing.

The one case this gives up is a belt whose *name* changes in a way that changes its CLASSIFICATION without changing its direction -- and there is none: `classifySide` keys on the entity's `type` and, for undergrounds and loaders, on which end it is, and every one of those changes the direction or the in/out flag too.

**M4 is done.** The head-to-head against both incumbents, the table above and [`bench/baselines/RESULTS.md`](bench/baselines/RESULTS.md) in full. The exit criterion (≥10× at scale) is met with margin on every saturated cell; the ~100× stretch target is not met on whole-tick cost and never could have been, because what is left is the hidden network's own engine cost rather than overhead — on the axis that target was about, mod Lua, there is nothing left to measure. What M5 and anything after it inherit from it:

- **The idle claim held literally**: `scriptUpdate` came out AT the control's, not near it, and `[BBB]` lines inside a benchmark window are zero. There is no `on_tick` handler and there must never be one.
- **The verbose default cost nothing to benchmark against.** `make QUIET=1` exists and benchmark builds were expected to want it; they do not, because the guest runs no code at steady state and so logs nothing. Keeping the verbose build is what makes "zero `[BBB]` lines" an assertion rather than a tautology.
- **The one regression M4 found is the GC tail** (above). It is a property of the LIVE heap's size — which is the guest's linear memory as a Lua word table, and is the same object in every persisting mode — so anything that grows what the registry stores per part makes it worse, and that is a performance consideration and not only a save-size one. M4 attributed it to `--persist=table` specifically; the persist re-measurement showed `packed` has the same tail, so the attribution was to the wrong half of the mode. **It was not the registry either** — see "The heap diet": the heap was log lines, and the rule that survives all three attributions is the general one, *anything permanently allocated per event is a worst tick*. The per-part discipline stays; it was simply never the biggest term.
- **`--create` cost is real and is where the whole architecture's bill lands**, and batching took a **4× bite out of it**: 4 compiles per 4×4 rig as the part block grew, 800 of them for the n=200 save — now **200**, one per cluster, because the whole `on_init` is one tick. Nobody builds 200 balancers in one tick, but a large blueprint paste is the same shape and gets the same reduction.
- **`bbb-audit`** is a shipped prototype, and it earned a second job. A script placing one asks for a full re-classification and gets `[BBB] audit clusters=… drift=… unbuilt=…` plus `[BBB] stats … compiles=… skipped=… builds=… teardowns=… creates=…`. It is also the only **synchronous** "drain the deferred queue and compile now" trigger there is, which is what every test mod's `on_init` uses to keep the compiling inside `--create` where `--benchmark` cannot see it. M4 did not need it; M4's successor could not have been measured without it.

### What M5 inherited, and what it did with it

Kept because the four notes below are what shaped the answer, and the last two were written before there was one:

- **Event-time cost went DOWN, in two ways.** Filters mean the guest is not entered at all for anything that is not a belt-connectable or one of our two named prototypes, and `fk.Defer()` means a build or mine event does a registry update and a queue insertion and nothing else -- no classification, no host call on the neighbour path.
- **A benchmark that builds its rigs in `on_init` pays a one-off surface scan** at the first event. `fk_after_load` exists upstream now and is **deliberately not adopted**: it fires only after a LOAD, never on a new map, and the two loads that matter are already covered — an adopted heap needs nothing, and a rebuilt guest gets `fk_migrate`.
- **Work that reads only the event's own payload happens in the event; work that reads the world happens in the flush.** M5 obeyed it: the mask is computed from the registry (no host call, so it could have gone either way) and the entities are touched in the flush, which is what makes the sprite correct one tick after the part is placed rather than inside the event.
- **The merged tile set is free**: `collectCluster(root)` produces it, in a deterministic order, with no host call, and that is what `restyle` walks.

## M5 is done

**Adaptive themed graphics over merged shapes, and the exit criterion is met: a balancer of any shape reads as one structure.** Connected parts share a continuous plated surface with no border between them, trim appears only along the real outline, and the outline turns cleanly around holes and concave corners. It costs one byte per part in the guest heap, two host calls per part whose picture actually changed, and nothing at all per tick.

### The mechanism, in one paragraph

`bbb-balancer-part` is a `simple-entity-with-force`, which inherits `pictures` (a variation set) and the runtime attribute `LuaEntity.graphics_variation` from `simple-entity-with-owner`. The prototype ships **47 pictures** in one 512×384 sheet; the guest sets a part's variation whenever its cluster's SHAPE changes. There is no second entity, no rendering object per part, no `on_tick`, and nothing in `storage` but a byte. **Verified empirically before anything was built on it** (a five-minute headless probe): the attribute is accepted on this prototype, reads back, survives a fresh handle, is **1-based**, and a value above the count silently WRAPS modulo it — 48 draws cell 1 and 255 draws cell 20 — which is why `skin_test.go` asserts that every mask lands in 1..47 rather than trusting an error to appear.

### The bitmask, and why 47 rather than 16

```
bits: N=1 E=2 S=4 W=8   NE=16 SE=32 SW=64 NW=128
```

A bit is set when that neighbour tile holds a part **of the same force** — two forces' parts touching are two balancers and must not fuse into one picture, the same rule the flood fill and the compiler already obey.

The four side bits alone give 16 pictures and already put the trim only on the outside. What they cannot express is the INSIDE of a corner. A part whose north and east neighbours are both present while the north-east diagonal is empty owns the point where those two neighbours' trims meet, and if it draws nothing there the outline dies in a notch. **That is the entire visual difference between "tiles that agree about their borders" and "one machine".** A diagonal bit is meaningful only when both sides touching it are set — if a side is missing, that side's own trim already draws the corner — and canonicalising on that rule leaves the classic **47** of the 256:

| sides | configs | × meaningful corners | |
|---|--:|--:|--:|
| none | 1 | 1 | 1 |
| one | 4 | 1 | 4 |
| two, opposite | 2 | 1 | 2 |
| two, adjacent | 4 | 2 | 8 |
| three | 4 | 4 | 16 |
| four | 1 | 16 | 16 |
| | | | **47** |

`guest/go/skin` and `tools/make-graphics.py` both enumerate masks 0..255 ascending, keep the canonical ones and number them from 1. **Neither stores a table**; the rule is six lines on each side and three anchors are asserted in both, so a change made on one side fails on both. `Canon` is proved idempotent and total over all 256 masks by `go test ./skin/`.

### How the art is generated, and what an artist would replace

`tools/make-graphics.py` computes every pixel (nothing is copied from the base game or any mod) and writes three committed PNGs:

| file | what |
|---|---|
| `graphics/entity/balancer-part-variants.png` | 512×384, 8×6 cells of 64 px, 47 used |
| `graphics/icons/balancer-part.png` | 64×64, the lone part with its conduits lit |
| `graphics/entity/io-arrows.png` | 256×32, eight 32 px arrow cells |

**Nothing in it draws "an edge piece" or "a corner piece", and that is what made 47 affordable rather than three times the work of 16.** The nine cells of the local neighbourhood become a signed distance field — for each pixel, the distance to the nearest EMPTY cell — and the trim is a band at a fixed distance from that boundary. All 47 cells fall out of the same six lines. Concave corners come out rounded, because the distance to an empty cell's corner is radial, and that rounding IS the fillet that makes a blob look fused; convex corners come out sharp, which is what an armour plate should look like.

The theme on top of that: a dark slate plated body with a vertical gradient, brushed-metal hash noise, plate seams and rivets on a 16/32 px lattice **offset so they line up across a tile boundary** (a 2×2 block is one continuous plated surface, not four squares); a steel-blue edge trim with fake lighting that brightens north-facing edges and darkens south-facing ones; and a lit core in every part with a conduit running to each CONNECTED side, so two neighbours show one unbroken lit vein between their cores. Re-theming is the palette block at the top of the generator plus one `make graphics`.

**That contract was exercised on 2026-08-19** and it held: an artist replaced all three entity PNGs plus a new 288x288 mod logo, and the only thing that had to move on this side was one alignment constant in `sprite.lua` -- which was fixing a defect of ours that the placeholder had carried since M5, not accommodating the new art. See "The I/O arrows" above.

**A future artist replaces the PNGs and keeps the cell order.** The only contract is "cell *i* is the shape whose canonical mask is the *i*-th ascending one, 64×64, laid out 8 per row"; `variant_masks()` prints the list. Nothing in the guest or the prototype needs to change.

### What it costs

- **In the guest heap**: one byte per part (`pvar`, in cluster.go's parallel slices — 29 bytes per node became 30). That matters because the idle GC tail scales with the live heap, which is why it is a byte and not a struct.
- **In host calls**: `restyle` computes every part's mask from the REGISTRY, so deciding that a 200-part balancer's pictures are all still correct costs **zero**. Only the parts whose picture changed are touched, one `find_entity` and one `graphics_variation =` each. Growing a 4-part line by one tile moves exactly two pictures, and the M1 suite asserts that number (`set=2 of parts=5`) rather than trusting it.
- **Per tick**: nothing. There is no `on_tick` handler and there must never be one.
- **In the mod**: the zip went 139,762 → **185,291 B**, of which ~40 KB is the three PNGs and ~5 KB is the guest. `fk_module.lua` grew 900,786 → 953,523 B. **73 KB of that was one `panic`**: a runtime assertion in `skin`'s package initialiser linked TinyGo's whole panic-and-print machinery into a guest that has no other use for it. It is a zero-length array declaration now, and the counting is proved by the test instead. Worth remembering — a `panic` in guest code is not free the way it is in a normal Go program. (The shipped numbers are **191,677 B** zip / **1,035,941 B** of `fk_module.lua` now; the heap diet added the difference and "The heap diet" says why that was the right way round.)
- **In FkLua**: nothing. No new gap, no workaround. `graphics_variation`, `find_entity`, `rendering.draw_sprite` and the `pictures` variation set were all reachable through the generated bindings as they stood (3,878 members bound; this mod ships 28 of them, up from 25).

### When a part changes picture, and why the fingerprint is not enough

The compiler's fingerprint is a statement about the belts AROUND a cluster and says nothing about its SHAPE, so a sprite update cannot ride on it. `restyle` is called for every root the flush is about to compile, immediately before `compile`, and decides for itself: it compares the mask of every part against `pvar` and does nothing when they agree. That is the common case — a belt laid beside a balancer queues its cluster, and this looks at it and makes no host call.

`rebuildFromWorld` calls it too, before `inspectNetwork` (which holds the tile buffer `restyle` would otherwise reuse under it). A fresh heap knows nothing about what is drawn on the parts standing in the world, so that path pays two host calls per part — once per mod upgrade — and it is what makes a save built by an older build come back drawing the right shapes.

`random_variation_on_create = false` is on the prototype. Without it the engine picks a RANDOM cell the moment a part is created, which would be the wrong shape for one tick and would flicker across a blueprint paste. Cell 1 is the lone part, which is what a freshly placed part usually is.

### The I/O arrows, and the one property that made them cheap

Every visible interface the compiler places carries one sprite saying which way items cross that edge: a green double chevron pointing inwards on an input edge, an amber one pointing outwards on an output, sitting 0.3 tiles towards the side the belt is on, **alt-mode only** (Factorio's own convention for an informational overlay, and it keeps a big balancer from being covered in arrows).

**The lifetime is the whole design.** A rendering object whose target ENTITY is destroyed is destroyed with it (2.0.77 runtime doc, `ScriptRenderTarget`), and the target here is the visible linked belt that a teardown already sweeps. So the guest stores **no rendering ids at all** — no per-cluster list in the heap, no teardown path to get wrong — and the arrows come down with the network on every recompile, every clone reconcile and every surface deletion, including the ones nobody thought about. A network adopted by `rebuildFromWorld` keeps the arrows the previous session drew, for the same reason. The M3 suite asserts 58 rendering objects against 58 standing interfaces after ~100 teardowns and two surface deletions.

Eight sprite prototypes rather than one drawn with an `orientation` and an offset, because the rotation and the shift bake into the prototype: the guest names a sprite and passes a target and a surface, with no orientation to compute from a `defines.direction` value whose number this mod deliberately never writes down, and no offset table to marshal on every draw. `dirIndex` inverts `dirOf` in four comparisons, for the same reason `plan.Opposite` is a lookup on the installed compass rather than arithmetic.

**The shift is per FAMILY, and 0.3 is the DISTANCE rather than the number.** Both the generated placeholder and the 2026-08-19 artist delivery draw each chevron flush against its TAIL edge instead of centred in its 32 px cell, which puts the glyph's centroid **6.6 px** — 0.104 tiles — behind its tip. Measured, and identical in both sheets to the tenth of a pixel, so it is a property of the convention and not of one delivery. An input points INWARDS, so that bias pushes it further out and ADDS to the shift; an output points OUTWARDS, so the same bias pulls it in and SUBTRACTS. One shift of 0.3 therefore landed the two families at **0.404 and 0.196 tiles** from the tile centre: an output stopped reading as an edge marker and sat on the machine's own hub, and a corner part carrying two outputs collapsed both of them into one illegible blob. `sprite.lua` applies `0.3 -/+ ART_BIAS` per family now and all eight land at exactly **0.300**, checked by evaluating the table under `../FkLua/bin/lua52f`. If the art is ever redrawn centred, set `ART_BIAS` to 0 rather than editing the two distances.

**The defect was ours and it shipped with M5**, which is the part worth keeping: the placeholder had the same centroid bias from the day the arrows were drawn, and the only thing the new art changed was making it visible by being bolder. Nothing headless can see it — the `m3` suite counts 58 rendering objects against 58 interfaces and is satisfied by a sprite that exists, wherever it lands, and no assertion anywhere in this repo is about WHERE a cosmetic overlay sits. Found in interactive play, which is the third defect of that shape this file records after "The tan streak" and "The wake race".

### The zero-script property still holds

Re-measured after M5, n=200 k=4 express idle, against its own in-session control (`bench/run.sh --mod none --scenario control-idle` / `--mod bbb --scenario idle`):

| idle, n=200 k=4 express | control | **bbb** | Δ per balancer |
|---|---:|---:|---:|
| `scriptUpdate` | 1.28 µs | **1.30 µs** | 0.0001 |
| `wholeUpdate` | 161.05 | **154.98** | **below the control** |
| `avg_ms` | 0.1690 | 0.1875 | 0.00009 |
| worst tick | 1.66 ms | 18.58 ms | the GC tail, unchanged *at the time* |

**That last row is now 1.42 ms against 2.26** — see "The heap diet" below, which is the pass that found what the tail actually was. The rest of the table is unchanged and is kept as measured.

**`[BBB]` log lines inside the benchmark window: 0**, counted directly in both `run.log` and `verbose.log`. The create log carries 200 `compiled cluster` lines and 200 `skin cluster` lines — every sprite decision, like every compile, happens in `--create`. Recompile cost is unmoved: a 4×4 teardown-and-rebuild is 5.79 and 5.82 ms across two runs against the 5.90 M4's persist pass measured, and an 8×8 is 11.16 and 11.31 against 11.55. The eight `draw_sprite` calls a 4×4 makes are inside the noise of ~350 host calls.

### The editor's variation picker, which is a quirk and not a defect

Opening a balancer part in the **map editor's** entity dialog shows a sprite/variation picker. That is the standard editor UI for any entity with a `pictures` variation set — trees and rocks show the same control — and it is what M5's whole mechanism buys, so it cannot be turned off: 2.0.77's prototype API has no field that suppresses it (checked exhaustively over `SimpleEntityWithOwnerPrototype` and every `editor`- or `variation`-named property in `prototype-api.json`; the only related field is `random_variation_on_create`, which this mod already sets).

It is benign, with one wrinkle worth knowing: **a variation picked by hand persists until something changes that cluster's SHAPE.** `restyle` compares each part's computed mask against `pvar`, the byte the guest remembers writing, and does nothing when they agree — it does not read the entity's current variation, because that would be a host call per part on every flush to detect something only an editor can do. So a hand-picked cell survives every belt laid nearby and every recompile, and is corrected the moment a part is added or removed (or by any mod upgrade, which resets `pvar` through `rebuildFromWorld`).

## The tan streak — a clone of a belt keeps the belt's pictures

**A field report from interactive play: a tan streak between adjacent balancer parts on a 2×2 with one corner missing, visible only while items were flowing.** Seven headless suites were green while it happened, and they always would have been: not one of them can see a pixel.

`bbb-linked-belt` is `util.table.deepcopy` of base's `linked-belt` with the speed raised and the player-facing surface stripped — and "stripped" covered the selection box, the sounds and the item, not the **art**. It kept:

| what it kept | how big | at which layer |
|---|---|---|
| `structure` | 192×192 at scale 0.5 = **three tiles by three** | `object` — the part's own layer |
| `belt_animation_set` | the running belt plus its starting/ending patches, drawn past the tile edge by design | `transport-belt`, `transport-belt-endings` |

The part sprite is 64 px at scale 0.5 — **exactly one tile, opaque in all 47 cells** (decoded and checked pixel by pixel, minimum alpha 255 on every edge of every cell) — at render layer `object`. So an interface is completely covered **on its own tile** and completely uncovered everywhere else, and base's linked belt paints eight neighbouring tiles. On a solid rectangle the neighbours' own sprites hide most of it. On a shape with a **notch** — which is exactly what the field report was — the empty tile is covered by nothing at all.

### What was blanked, and why every replacement is a sprite rather than a `nil`

`guest/go/data/hidden.go`. All four hidden prototypes, because the rule worth having is "nothing the compiler places draws anything", not "nothing the compiler places draws anything where it matters" — which is the qualification this defect came in through. Only `bbb-linked-belt` stands where a player looks; the other three are belt-and-braces on the hidden surface.

| prototype | field | replaced with |
|---|---|---|
| all four | `belt_animation_set` | a set whose `animation_set` is one transparent pixel, `direction_count = 20` |
| `bbb-linked-belt` | `structure` — all six `Sprite4Way` members | `util.empty_sprite()` each |
| `bbb-splitter` | `structure`, `structure_patch`, `frozen_patch` | blank animation / blank animation / `util.empty_sprite()` |
| `bbb-lane-splitter` | `structure`, `structure_patch` | blank animations |

**Validity, against the pinned `prototype-api.json` rather than against a passing run:** `TransportBeltConnectablePrototype::belt_animation_set` is optional (default null) and so is every member of `LinkedBeltStructure` and every splitter structure — *except* `LaneSplitterPrototype::structure`, which is `optional = false` and is therefore **replaced, never dropped**. Inside a belt animation set `animation_set` is itself non-optional, which is why the set is swapped whole instead of emptied field by field.

**Everything is a drawable empty sprite, not an absent one, and that is deliberate.** Both are legal; only one is a shape this repo has already proved. Headless Factorio never opens a sprite file and `test/check-sprites.py` only checks that the paths we name exist, so the **graphical** client is the first thing that would notice a shape the engine dislikes — which is precisely how a stale filename once shipped (commit e42e07d). `util.empty_sprite()` is core's own idiom and is already what the audit marker uses.

**`direction_count = 20` is the one number that is not arbitrary.** A belt animation set is indexed by direction and by patch and its defaults run up to `ending_east_index = 20`; base's own belt sets are `direction_count = 20` for that reason. `__core__/graphics/empty.png` is 64×64, so twenty rows of one pixel fit inside it with room to spare. The `WithCorners` form used by `bbb-belt` adds indices 5..12, which the same twenty cover.

### The items: 2.0.77 offers nothing, and this is the honest residual

**There is no prototype field in 2.0.77 that suppresses the drawing of items on a belt-connectable, and no linked-belt equivalent of `belt_length`.** Checked exhaustively rather than guessed, against `doc-html/prototype-api.json` (application_version 2.0.77, api_version 6):

- `LinkedBeltPrototype` has exactly five properties of its own — `allow_blueprint_connection`, `allow_clone_connection`, `allow_side_loading`, `structure`, `structure_render_layer`. None of them is item-shaped.
- `TransportBeltConnectablePrototype` has six — `animation_speed_coefficient`, `belt_animation_set`, `collision_box`, `flags`, `selection_priority`, `speed`.
- A sweep of **every** property on **every** prototype and type whose name matches `item|render|draw|hide|visib|belt_length` returns, for the belt family, only `draw_circuit_wires` / `draw_copper_wires` (splitter, belt, loader), `structure_render_layer` (linked belt, loader) and **`LoaderPrototype::belt_length`** — which exists on loaders alone. There is no such field on `LinkedBeltPrototype`, so the visible travel is one tile and cannot be shortened.
- Raising `speed` does not help either: a transport line's item DENSITY is fixed by item size, not by speed, so a saturated interface holds the same number of items however fast it runs. (The 0.25 this paragraph was written against is a FLOOR since 0.3.1, not the ceiling it says; the point stands either way, because the drawn overhang is a function of item size and not of speed. `guest/go/data/hidden.go` is the header, `hidden.lua` having been the file before the data stage became a guest.)

**So what a player should still expect:** items on a visible interface are drawn at render layer `item`, which is *below* `object`, so the part's own sprite hides them on the part's tile. What it cannot hide is the ~0.16-tile **overhang** of an item sprite whose centre has reached the far edge of the interface belt — and for an INPUT edge that far edge points *into* the cluster, so on a shape with a notch the overhang lands in the notch. That is a thin band at the tile boundary, flow-dependent, and it is what is left after the blanking. Nothing in this repo can remove it; it needs a prototype field the engine does not have.

### The structural half, which is the durable one

Blanking is a statement about one prototype. The `edge` suite now asserts the statement that outlives it: **the only thing the compiler ever creates on a surface a player looks at is an edge interface, and an interface stands on a tile of the cluster itself** — under a part's opaque one-tile sprite. `probe_placement` enumerates every entity of all four hidden prototypes on the visible surface and requires a registered part on that exact tile, and it is sampled five times, including once with every rig in the save saturated:

| sample | ours | on a part tile | off one |
|---|--:|--:|--:|
| `init` | 125 | 125 | **0** |
| `post-merge` | 125 | 125 | **0** |
| `post-add-out` | 130 | 130 | **0** |
| `flowing` | 131 | 131 | **0** |
| `final` | 114 | 114 | **0** |

(Re-measured 2026-08-04 after the `lim` rig joined the suite; the counts have now moved twice — 56/56/61/61/54 originally, then 60/60/65/66/49, now these, because each new rig adds its own edge interfaces. The assertion column has never moved: **0 off a part tile, every sample, every recording.**)

All four prototype names are probed rather than just the linked belt, so a hidden splitter appearing out there fails rather than going uncounted; and a sample of fewer than twenty fails, so a probe that found nothing cannot pass.

The rig it is really about is `ntch`: parts at (0,b), (1,b) and (0,b+1), **the corner at (1,b+1) deliberately empty**, two inputs from the west, one output east off the top-right part and one SOUTH off the bottom-left one — which is what keeps the notch a notch instead of filling it with an output belt. It is saturated from `on_init` to the last tick and delivers **376 376** over the same 500-tick window the `aout` balance check uses, which is what makes the `flowing` sample a measurement of a balancer that was actually running.

## The heap diet — the idle GC tail, closed

**The idle GC spike was never a persistence cost, an architecture cost or a compiler cost. It was the guest's own log lines, built with `+`.** Fixing that took the idle worst tick at n=200 from **18.1 ms to 2.26 ms** and at n=500 from **49.7 ms to 5.08 ms**, and the bench save from 3.6 MB to 1.5 MB. It is the last performance defect this repo had a name for ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 17), and it is closed.

### What made it findable

Upstream wrote down the budget. `../FkLua/agents/guests.md`, "the guest heap budget": **0.2 ms of worst tick per MiB of guest linear memory**, flat from 8 MiB to 128, because Lua 5.2 walks a table in one `propagatemark` it cannot split across ticks — and it is the memory's SIZE, not the part in use, because `mem_grow` zeroes every new word and TinyGo's `growHeap` **doubles**. So a guest is always on a ladder, and `fk_mod.lua` now logs which rung it is on:

```
fklua: this guest's linear memory is now 64 MiB, which is about 12.8 ms of
worst tick -- ...
```

Three of those lines in one `--create` (16 → 32 → 64 MiB) is what turned a two-milestone attribution argument into a ten-minute measurement.

### The measurement that found it

One experiment, and it was decisive. Build the same guest with `QUIET=1`, which eliminates every `[BBB]` line below the error level, and create the same n=200 k=4 express bench save:

| n=200 k=4 express idle | doublings logged | idle worst tick | `--create` |
|---|---|--:|--:|
| verbose, string concatenation | 16 → 32 → **64 MiB** | 19.9 ms | 44.4 s |
| `QUIET=1`, no lines at all | **none (under 16 MiB)** | 2.28 ms | 9.3 s |

**The entire guest heap was log lines**, and so was three quarters of the create time. Everything the compile path does — the plan on its warm buffer, the edge list, the `create_entity` tables, the arena — was already frugal enough that without the logging the heap never reached the first rung that is visible at all. The M2 zero-allocation claim held through M3, M4 and M5; nobody had checked the instrumentation.

### Why a log line was 9 KB

`-gc=leaking` was mandatory then, so every intermediate string of every concatenation was permanent — in the heap, in the save, in every multiplayer join. And `+` in a loop is quadratic. `logState` is the one that mattered:

```go
s := "[BBB] state clusters=" + u32(uint32(len(snap))) + " parts=" + u32(nParts) + " sizes="
for i, n := range snap {
    if i > 0 { s += "," }
    s += u32(n)        // 64 of these, each allocating len(s)+len(n)
}
```

Up to 64 cluster sizes appended one at a time is ~8.8 KB of dead strings, plus a `strconv.FormatUint` allocation per number. **It runs once per part placed**, and a n=200 k=4 create places 3,200 parts: ~26 MB, which is the 32 → 64 MiB rung.

### The fix

`guest/go/logline.go`: one package-level `[512]byte`, an index, `copy` to append, and `unsafe.String` to hand the host a string that borrows the buffer rather than one that copies it. Every log line in the guest goes through it; `strconv` is gone from the module, and so are `u32`, `i32` and `f2s`. **The line formats are byte-for-byte what they were** — they are the assertion surface for every suite, and all of them pass unchanged.

Two things it is worth knowing before touching that file, both measured:

- **`copy` on a fixed array, not `append` on a slice.** `append` carries a growth branch that LLVM inlines into every one of the ~200 call sites: wasm `code` 98,373 B against 81,457, which is 1,339,988 B of generated Lua against 1,035,941. A line that overran would be truncated instead of grown; the worst line this guest can write is 427 bytes, and both unbounded ones already cap themselves (`state` at 64 sizes, `skin` at 32 variations).
- **`//go:noinline` on the writers was tried and rejected.** It is worth 17 KB of wasm `code` and 160 KB of generated Lua — the mod would ship *smaller* than before the file existed — and it costs **2.1 ms on every 4×4 recompile**. Measured interleaved against the same base, four reps each, medians minus each run's own control: 5.50 ms without it, 7.59 ms with. The profiled window writes two log lines, so this is not the log path getting slower; it is what not inlining these does to the code around the ~350 host calls a recompile makes. A per-edit millisecond is worth more than a per-load kilobyte.

### What it bought

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

### What is left, quantified

The heap probe (the leaking allocator's own bump pointer, read at the audit) puts the residual at:

| | per rig (16 parts) | of which |
|---|--:|---|
| registering parts and their events | ~4.6 KB | ~290 B/part: the registry's parallel slices and `index` map, each carrying its doubling-growth history, plus one `name` string per build event |
| **one compile** | **~1.3 KB** | **all of it in generated binding RETURN values** |

Totals: 1.28 MB of heap at n=200 and 2.74 MB at n=500, against the 16 MiB rung.

**The ~1.3 KB per compile is not controllable from here**, and that is the honest part. A `find_entities_filtered` return is `out := make([]Object, n)` in `fkapi.go`; an entity return is `return &v`; a `type` read is `return string(b)`. A 4×4 compile makes 16 boundary queries, ~16 type reads, 32 `create_entity` calls, 16 `find_entity` calls and 8 `draw_sprite` calls, and every one of them leaves a Go value on a heap that never gives anything back. The marshalling arena (item 10) fixed the ABI's own side; this is the caller's side, and upstream's own note says so — "the 48 bytes that remain are the caller's, not the ABI's" (`../FkLua/agents/abi.md`).

At 1.3 KB per recompile a save needs **~800 balancer edits** to add one MiB of heap, not the ~12,000 this paragraph claimed until 2026-08-02: 1,048,576 over 1,300 is 806, and the error survived two rounds because nobody divided. The measured per-operation figures are in "The marathon save", which replaces this estimate with a table and a projection. [`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 17 stays open with those numbers on it rather than being closed as fixed. Nothing downstream can move it without changing how the classifier reads the world, which is a correctness surface and not a place to economise.

### The cost, stated plainly

The mod got bigger. `fk_module.lua` 953,523 → **1,035,941 B** (+8.6%) and the zip 186,267 → **191,677 B** (+2.9%), because ~200 `copy` call sites inline more code than ~200 string concatenations did. That is a per-load cost of about five kilobytes, against 45 MB of worst-tick pause and 2.1 MB of save at n=200. It is also the second time this repo has traded module size for a runtime property and been right to (the first was `-opt=2` over `-opt=z`, upstream's decision).

### The collected-mode postscript — the diet made the collector unnecessary HERE

**Upstream built the thing this section is a workaround for, and this mod measured it three times.** FkLua's `--gc=collected` ([`../FkLua/agents/gc.md`](../FkLua/agents/gc.md)) is a paced conservative mark-sweep over the guest's own heap: `-gc=custom` on the TinyGo side, one import, and one call. It is wired here — `make GC=collected` builds it, all seven suites are green under it, and it stays that way.

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

### Where collected WINS, which is why it stays buildable

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

**7.0×, checksum-identical, with a heap that plateaus against one that only climbs.** That is the whole case for the feature, on the workload it was designed for, and BBB does not have that workload — 5,400 balancer edits is not a session, it is a fuzzer. But an allocation regression in the compile path *would* look exactly like it, and today nothing but the discipline in [`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 17 stops one.

**No `fkgc:` line was ever logged, in any run.** The OOM valve never fired and `Stats().Deadlines` was **0** everywhere — including the 54× churn, which is the first real-guest evidence for either. `markDeadline`'s slack and floor were unvalidated against a real guest before this pass; they held.

### The first decision (2026-08-01) — leaking, and what would reverse it

**BetterBeltBalancer shipped `-gc=leaking`.** The rule this pass set out with was: flip if collected costs nothing the steady state can see AND bounds a heap that is actually growing. **Neither half passed.** The steady state gained 152 ticks of script per load on a mod whose headline is that it has none, and the heap it would bound is 1.9 MiB that the diet already made invisible to Lua's collector. The costs it does have — +11.9% zip, +25.9% of `fk_module.lua`, +2.3 ms per load, 163 KiB of permanent linear memory — are paid by every player on every join, forever, which is the same class of cost the `--persist` decision turned on.

`make GC=collected` stays green and stays tested for the case that reverses it:

- **an allocation regression in the compile path.** The 54× churn table is what one looks like, and the collected arm is the only one that survives it. If "What is left, quantified" ever stops being ~1.3 KB per compile, re-measure rather than re-argue.
- **a guest that gains a real `fk_on_tick`.** Every cost above traces to the create being one dispatch and the collection therefore landing after a load. A guest that ticks would pace as designed, and the 152-tick lump would not exist.
- **a contributor who should not have to know about the diet.** That is the feature's real pitch and it is a fair one; it is just not worth 25% of the module today, on a guest where the discipline is already written down and already enforced by a measurement.

### The second decision (2026-08-02) — re-taken on marathon numbers, and not flipped

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

### The third decision (2026-08-02) — re-measured, and this mod ships `--gc=collected`

**Every number the leaking decision rested on was re-taken on today's pin, both arms interleaved, and the two that were supposed to be disqualifying are not.** This is the pass the section above asked for and did not do. Method as before: `bench/run.sh`, n=200 k=4 express, `--mod` pointed at the two arms' own zips so no TinyGo rebuild lands between cells, **five reps of six cells each in `leaking / collected / control` order** so session drift (25–35%, see Benchmarks) cannot bias one side; the `mar`/`m2` cells three reps the same way. Worst ticks and the post-load transient are read from the verbose pass's **per-tick** `wholeUpdate` and `scriptUpdate` columns, not from `max_ms`, which conflates the load tick ([`bench/README.md`](bench/README.md)). Rows tagged `gcpass2` in [`bench/baselines/results.tsv`](bench/baselines/results.tsv).

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

#### What it buys, MEASURED rather than projected — the doubling stall

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

#### The rule, and what it decided

The rule this pass set out with, from the section above: **flip if the re-measured steady-state costs are within noise of leaking, AND the post-load transient is materially reduced or bounded and explained, AND the size penalty is the only remaining cost** — weighed against removing the marathon stall class. All three passed, and the thing on the other side of the scale got bigger rather than smaller. **BetterBeltBalancer ships `--gc=collected`.**

`make GC=leaking` stays buildable, stays green on all seven suites, and is what a future pass re-measures against. What would reverse THIS decision:

- **a `fk_module.lua` that a mod portal or a load time cannot carry.** 1.9 MB is the cost, and it is the whole cost.
- **a guest whose live set stops being ~9 KB.** Every argument above rests on the collector having almost nothing to retain, which is what makes its mark cheap and its steps invisible. A guest that kept state per network rather than a 64-bit fingerprint would be a different measurement.
- **a single-player-only audience.** The quiet column of the projection never reaches a doubling worth the name, and for that player the collected arm is 1.9 MB of module for nothing.

#### The pacer, and the call site that was missing

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

#### The threshold this guest installed and the collector never read

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

#### The transient that grew while nobody was looking

**The zero-script steady state holds, exactly, in both arms — and the collected arm's post-load transient is 14× what "The third decision" recorded.** Found on 2026-08-03 by re-running the two headline cells against their own in-session controls, which is the check that decision asked for and which nothing since had done. Method as always: `bench/run.sh`, n=200 k=4 express, three interleaved reps of `bbb saturated / none control / bbb idle / none control-idle`, per-tick `scriptUpdate` read from the verbose pass rather than from `max_ms`. Rows tagged `gcround6*` in [`bench/baselines/results.tsv`](bench/baselines/results.tsv).

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

## The root scan that could not fit in a step — the 14× transient, attributed

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

### The bisect, and the seven granules

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

### The fix: the budget is the other half of a contract this guest was in

`guest/go/gc.go` installs `fkgc.SetBudget` beside the `SetThreshold` it already installed, and the constant is derived from the root scan rather than from the pause: **an allowance of real work, plus a budget for this guest's own roots.**

**`SetBudget` is a latched knob for the same reason `SetThreshold` is**, and that is not a coincidence worth passing over: `initialize()` fills only zeroed fields, so this fix would have been silently discarded by the exact defect the previous commit fixed upstream. The round that found that latch is the round this fix could first be written in.

`gcCheckRoots` was the second half, because the version of this defect that shipped was invisible — seven suites green, every item count identical, a collection quietly taking fourteen times as long. It read `fkgc.RootWords()` on the first flush after a completed collection and logged a `[BBB] error:` when the roots grew back to three quarters of the budget. **It is deleted, and what deleted it is upstream doing the same job better** — see "The floor upstream built, and the check it retired" below.

### What it costs and what it buys, measured

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

### The floor upstream built, and the check it retired

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

### The half that is not about a benchmark, and the re-attribution it forces

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

### What was NOT done, and why

**A synchronous `fkgc.Collect()` at the end of the audit's flush** — so a `--create` writes its save with a swept heap and the post-load transient collects nothing — was the obvious candidate and is measured-unnecessary. It would take 36 ticks to 0, and it would do it by putting a full unpaced collection inside the one path a test harness controls, which is instrumenting the benchmark rather than fixing the mod. The budget fix reaches the same gate from the other side and fixes every collection a *player* ever causes, which the audit-path collect would not have touched at all. `gc.go`'s standing argument for why a collection does not belong in the audit path is unchanged and still correct.

## The marathon save — 300 hours, four players, and where the bill lands

**The heap diet asked what the guest allocates. This asks what it allocates PER EDIT, forever.** Under `-gc=leaking` a transient allocation is permanent: it is in the linear memory, in the save, in every multiplayer join, and in Lua's collector's walk for the rest of the session. So the question a marathon multiplayer game asks is not "does it leak" — everything does, by construction — but what the SLOPE is, whether any of it is superlinear, how much of it is ours, and what a 300-hour save therefore looks like.

Measured 2026-08-02 on Factorio 2.0.77 and re-recorded 2026-08-24 on **2.1.14** after the rigs were re-laid single-edge; base only, `-gc=leaking`, `--persist=packed`, `test/run.sh mar`: 680 net-zero world operations over 4,600 ticks, each followed by a `bbb-audit` marker so the guest prints its own `[BBB] heap post-audit sys=… alloc=…`. Under leaking `alloc` is TinyGo's bump allocator reporting every byte it has ever handed out, which IS the permanent heap.

### The slopes

**RE-RECORDED 2026-08-24 ON THE SINGLE-EDGE RIGS**, Factorio 2.1.14, `make GC=leaking test`. Every rig in this suite was re-laid one belt per part, which in three of the five legs that touch a balancer makes the CLUSTER bigger (two columns of parts instead of one) while leaving the NETWORK the same size, and in one makes the network smaller. The old figures are kept beside the new ones because the pairs are what say which term moved and why. Net of the audit (1,136 B, measured three times with the world untouched, **0.0% spread**, unchanged); each leg is shaped so that one term dominates it:

| one operation | 2.0.77, multi-edge | **2.1.14, single-edge** | what moved |
|---|--:|--:|---|
| a belt-connectable **mined or rotated** anywhere, no cluster within two tiles | 0 B | **0 B** | nothing; the guest rejects it from guest memory |
| a belt-connectable **built** anywhere, same | 32 B | **32 B** | nothing; it is the `name` string and nothing else |
| a belt laid **inside the two-tile gate** and picked up again | 352 B | **352 B** | nothing; the fingerprint still throws the re-classification away |
| one **teardown-and-rebuild of a 2→2** (11 entities) | 1,180 B | **1,209 B** | +29 B: the same network, over a four-part cluster instead of a two-part one |
| one **teardown-and-rebuild of a 4×4** (32 entities) | 3,736 B | **3,736 B** | **nothing at all** -- see below |
| a **whole 4-part balancer** in and out, run under load | 1,216 B | **1,280 B** | 16 entities each way instead of 18, over a 2x2 block instead of a 1x4 column |
| a **six-entity paste and its undo** | 736 B | **560 B** | the six entities now make a 1→1 (P=1, 5 entities) where they made a 2→2 (P=2, 11) |
| a full balancer **grown by a part, dissolved and rebuilt** | 1,712 B | **2,080 B** | five parts torn down and four put back, where it was three and two |
| one **`bbb-audit`** over the save | 1,136 B | **1,136 B** | nothing |
| linear memory over the 680-operation run | 3.92 MiB | **3.92 MiB** | nothing |

**THE 4×4 LEG IS THE CONTROL ON THE WHOLE RE-LAY AND IT CAME BACK IDENTICAL TO THE BYTE.** That is not luck: a 4x4 built the wide way -- inputs on the west column, outputs on the east, two interior columns carrying nothing -- was ALREADY single-edge and never had a tile with two belts on it, so leg G's rig is the one thing in this suite that did not move. 3,736 B before the port and 3,736 B after is the measurement that says the guest's compile path is untouched and every other row of this table is geometry.

**And every term the 300-hour projection uses is in that unmoved set.** The edit-rate model below multiplies exactly three of these -- the far-belt build (32 B), the belt inside the gate (176 B per primitive) and the 4×4 recompile (3,736 B) -- and all three are byte-identical across the port. **The projection stands exactly as recorded**, and it is worth saying out loud that this was checked rather than assumed.

**The 2.0.77 column itself carries the 2026-08-02 item-placement policy** — a recompile reinserts its drained items instead of spilling them ("A recompile is not a removal"), which is more host calls on the two recompile legs and fewer allocations on every leg that drains. Its own before/after pair (681 → 1,180 for the 2→2, 1,517 → 3,736 for the 4×4, 1,921 → 1,712 for the grown balancer) is in that section. **Under the shipped `--gc=collected` none of it persists**: on 2.1.14 the same 680 operations end on **0.46 MiB of linear memory with a 10,192 B live set**, 9 collections in 6 paced steps and **0 forward-progress deadlines** (2026-08-02 on 2.0.77: 0.46 MiB and 8,736 B; 2026-08-03: 0.71 MiB and 16,448 B). All of them are orders inside the suite's 4 MiB and 256 KiB ceilings.

**Nothing is superlinear, and that is the finding rather than the arithmetic.** Every leg's second hundred iterations cost what its first hundred did, to within 7% — the suite fails on 1.35×. Measured over the re-laid rigs: A ×1.00, B ×1.00, C ×1.03, D ×1.00, E ×1.00, G ×1.07, F ×1.00. That is the classic 300-hour killer ruled out directly rather than argued: it means a place/remove cycle that nets zero entities really does cost a constant, and three hundred hours is multiplication rather than a curve.

**Why it is flat, checked shape by shape.** Every monotonic-growth candidate in the guest was audited and every one of them is bounded by a HIGH-WATER MARK rather than by operations ever performed:

| shape | verdict |
|---|---|
| registry node ids | **reused.** `freeNode` pushes the slot onto `freeID` and `newNode` pops it, LIFO. Ids do not grow with parts ever placed, and the seven parallel slices (`parent`, `csize`, `ppos`, `pforce`, `alive`, `mark`, `pvar`) grow only to the most parts the map has ever held at once |
| the `pvar` byte | in those slices, so the same answer. A reused slot is reset to 0, which is "we do not know what is drawn there" |
| `index`, `nets`, `rfwClaim` | Go maps, all three point-queried only. `delete` is called on every removal path; the bucket array is high-water, and under leaking each doubling's predecessor stays — so ~2× high-water, not ~1× operations |
| slot grid | **reused.** `releaseSlot` pushes onto `freeSlots`, `allocSlot` prefers it. `nextSlot` rises only to the most networks standing at once, and `sweepOrphanSlots` rebuilds the allocator around what is actually claimed |
| the defer queue | `deadRoots`/`liveRoots` are truncated to `[:0]` by every flush; high-water is the clusters touched in one tick |
| edge lists, plans, tile buffers, the `create_entity` table | package-level and reused; `netInfo` stores a 64-bit FINGERPRINT rather than the edge list, exactly so that a stored slice per cluster is not a leak that grows with play |
| a map only ever written | there is none |

The linearity check is what proves this rather than the reading: a registry whose ids grew monotonically would show up as a slope that climbs as the slices double, and it does not.

### Whose bytes they are

**Every byte in the compile terms is a generated binding's return value, and none of it is BBB's own.** A `find_entities_filtered` return is `out := make([]Object, n)`; an entity return is `return &v`; a `type` or `name` read is `string(b)` out of `getStr`, which must copy because the marshalling arena underneath is released when the call returns. A 2→2 teardown-and-rebuild makes about thirty such calls, which is the 681 B. The arena (FKLUA-GAPS.md item 10) fixed the ABI's own side; this is the caller's, and upstream's own note says so — "the 48 bytes that remain are the caller's, not the ABI's".

**The one term that WAS ours is taken.** `onEvent` used to read `name`, `position` and `surface_index` in that order for every belt-connectable event on the map, and the engine's filter admits every belt there is — so every belt anyone lays or picks up anywhere paid 32 B of permanent heap for a string the guest usually threw away. Position and surface index are scalars decoded out of the return block and cost nothing, so they go first now, and a **disappearance or a rotation** that is not on a registered tile, not inside the two-tile gate and not on the hidden surface returns without buying the name at all. All four questions are answered from guest memory (`nearCluster`, 25 point queries, no host call).

An **appearance** still buys it unconditionally and always will: a part placed alone in the middle of nowhere has no neighbours to be recognised by, so the name is the only thing that can identify it. Measured, same suite, before → after:

| | before | after |
|---|--:|--:|
| a belt laid and picked up 18 tiles from anything | 64 B | **32 B** |
| a whole 4-part balancer placed and removed (18 entities each way) | 1,664 B | **1,216 B** |
| a six-entity paste and its undo | 864 B | **736 B** |

The 448 B the second row saved is exactly the 14 belts' mine events at 32 B each, which is the arithmetic working.

### The projection

**The edit-rate model, stated so it can be argued with.** Per player-hour, for a player who is actively building rather than idling:

| | events/player-hour | why |
|---|--:|---|
| belt-connectables **built** anywhere | 250 | a busy hour lays a few hundred belt pieces; every one enters the guest |
| belt-connectables **mined** anywhere | 150 | free now |
| belts laid **inside a balancer's gate**, no edge moved | 10 | balancers are a small fraction of a base's tiles |
| edits that **move an edge** on a 4×4 | 6 | |
| whole balancers built or removed | 1 | ~3.6 KB each at 4×4 size |

= 8,000 + 0 + 1,760 + 9,102 + 3,600 ≈ **21.9 KiB per player-hour**.

**Re-run 2026-08-02 with the item-placement policy's 4×4 term (3,736 B): ~34.9 KiB per player-hour, 1.60×.** The table below is the 21.9 KiB model as measured; multiply the heap rows by 1.6 for today's guest, which moves the busy column to ~41 MiB of permanent heap and one rung up (32 → 64 MiB) of linear memory. Both are `-gc=leaking` numbers and the shipped build is `collected`, where the live set did not move at all.

**The single-edge port did not move any of this, and that was checked rather than assumed.** All three multiplied terms -- 32 B for a far belt built, 176 B for a belt inside the gate, 3,736 B for a 4×4 recompile -- came back byte-identical on 2.1.14 over the re-laid rigs (see "The slopes" above). The projection stands as recorded.

| after 300 hours | quiet (1 player, ¼ rate) | **busy (4 players)** | extreme (8 players, 2× rate) |
|---|--:|--:|--:|
| permanent guest heap | 1.6 MiB | **25.7 MiB** | 103 MiB |
| linear memory (TinyGo's `growHeap` DOUBLES) | 2 MiB | **32 MiB** | 128 MiB |
| save size added (`packed`, 113 KiB/MiB) | 0.2 MB | **3.5 MB** | 14 MB |
| host RAM, per client (5.00 MiB/MiB) | 10 MiB | **160 MiB** | 640 MiB |
| join / load time added (~26 ms/MiB, flat) | 0.05 s | **0.8 s** | 3.3 s |
| Lua GC total per cycle (0.202 ms/MiB) | 0.3 ms | **5.2 ms** | 21 ms |
| Lua GC **worst tick** (sharded) | ~0.5 ms | **~0.5 ms** | ~0.5 ms |
| worst **`memory.grow`** tick, the last doubling | ~0 ms | **~420 ms** | **~1.8 s** |

Rates from `../FkLua/agents/guests.md`, "the guest heap budget". The `packed` save number is the same 0.44 B/word the heap slope implies: **the heap slope IS the save-size slope**, because packed pages track the heap's high-water mark.

**And the row that matters is the last one, which is not the row anybody expected.** Sharding made Lua's collector's worst tick FLAT at ~0.5 ms whatever the heap size — the 0.2 ms/MiB penalty this repo spent two milestones attributing is a per-cycle TOTAL now and not a pause. The save is 3.5 MB. The join is under a second. All three are documented bounds a mod can ship with.

What is not is `memory.grow`. TinyGo's `growHeap` grows by exactly the current size, so a leaking guest is always on the ladder, and `mem_grow` writes a zero into every new word at ~107 ns a word in Factorio's Lua. The step from 16 MiB to 32 writes 4,194,304 fresh words — **~450 ms in one tick**, less the ~25 ms the runtime's paced pre-build can have ready. Upstream measured the same shape directly: **491 ms worst grow tick for a leaking guest at a 16 MiB target and 974 ms at 40 MiB**, against a collected guest's flat 23–25 ms. The pre-build is capped at 1 MiB deliberately — a lookahead of "one grow" would be unbounded for a doubling guest — so it removes a fixed ~25 ms and 2.5% of a 32 MiB doubling.

**Nothing downstream can bound that.** It is a fact about the growth LAW, not about collecting; guests.md says so in as many words. A busy 300-hour multiplayer server under `-gc=leaking` pays about eight of these stalls over its life, the last two being roughly 225 ms and 450 ms — single-tick freezes every client feels at once.

> **MEASURED SINCE, AND WORSE THAN THIS.** The rung above was arithmetic; driven up the ladder for real on 2026-08-02 it came out **48.7 ms at 2→4 MiB, 120.3 at 4→8, 226.1 at 8→16 and 782.4 ms at 16→32** — the two middle rungs on the 107 ns/word model almost exactly, and the last one 1.74× over it, because a 16 MiB grow also pays eight new shards' last array-part reallocation (`../FkLua/agents/sharding.md` §15). **This is the row that flipped the `-gc` decision**; the mod ships `--gc=collected` since that pass, and the same 3,400 operations end on 0.52 MiB of linear memory instead of 31.9 with a worst tick of 71 ms instead of 782. See "The third decision" above. Everything else in this section is a statement about the OTHER arm, which stays buildable and stays measured.

### What this does NOT say

- **It is not a leak in the ordinary sense.** Nothing is retained: the collected arm of the same run reports a **live set of 8,736 B** at the end of 680 operations (16,448 B on the 2026-08-03 re-measurement). Every byte above that is reclaimable and simply is not reclaimed.
- **A single-player game is unaffected.** The quiet column costs 0.2 MB of save and no stall at all, which is what the mod has been measured at all along.
- **The compile slope is not reachable from here.** It is generated binding return values, [`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 17, and moving it means changing how the classifier reads the world — a correctness surface, not a place to economise.

## Benchmarks

`bench/` is a headless harness: it builds N saturated balancer rigs into a save via a setup mod's `on_init`, `--benchmark`s it, verifies items actually moved and the outputs are balanced, and appends a row to `bench/baselines/results.tsv`. Usage and method: [`bench/README.md`](bench/README.md). `bench/matrix.sh` regenerates the whole matrix (~3 min). Third-party mod zips are read from `$BB_MODS_SRC`, never committed. **`MEGA=1 bench/matrix.sh` runs a different matrix**: one heterogeneous save of 404 balancers over ten shape classes, including the only 16x16, 32x32 and 64x64 this project has ever built in a real game. See "The megabase cell" at the end of this section.

**EVERY NUMBER IN THIS SECTION WAS MEASURED ON FACTORIO 2.0.77 AND THIS MACHINE HAS 2.1.16, so the whole section is HISTORY until somebody re-runs the matrix.** Two things found on 2026-08-25, when phase 7 of the estate port went to take a comparability baseline and could not:

- **The harness did not run at all on 2.1.** The setup mod's `info.json` said `factorio_version: 2.0` and 2.1 refuses such a mod at the LOADER, exactly as it refused every suite before `test/run.sh` learned to stamp. `bench/run.sh` stamps the staged setup mod now, the same perl over the same two fields.
- **`--benchmark-verbose` SIGSEGVs on 2.1.16**, after emitting the `t0` row, with any counter list — measured on a VANILLA NO-MOD SAVE as the control, so it is the engine's defect and not ours. So `whole_us`, `belts_us`, `entity_us` and `script_us` cannot be obtained on this engine at all, and neither can the per-tick `wholeUpdate` column this file sends every worst-tick question to. `run.sh` skips the pass on 2.1 with the reason printed rather than crashing Factorio once per cell (the crash also leaves a reporter process holding the run's own `.lock`, which fails the NEXT cell on something unrelated); `BENCH_VPROF_FORCE=1` is how a future engine gets re-tested.

**AND THE SETUP MOD IS A COMPILED GUEST since 2026-08-25** (`guest/go/obs/bench`), which moves the harness's own absolute milliseconds and no delta it publishes. What that cost, measured interleaved against the Lua it replaces, is in [`agents/estate-port.md`](agents/estate-port.md)'s phase-7 section.

Baseline to beat — Factorio 2.0.77, base only, M3 Pro (full tables: [`bench/baselines/BASELINE.md`](bench/baselines/BASELINE.md)). **The bar is belt-balancer-3 v1.0.1** (the live successor; user decision 2026-08-01), but bb2 v2.0.9 is *faster* than bb3 on every cell — bb3's `lane.valid` crash guards are extra boundary crossings — so bb2's numbers are the operative target and bb3 falls with it. Correctness bar: both deliver exactly full belt throughput with an exact per-output split — match that first.

**M4 is done and the bar is cleared by 22–67× on every saturated cell.** Full head-to-head, method and caveats: [`bench/baselines/RESULTS.md`](bench/baselines/RESULTS.md). Marginal cost of one 4×4 balancer, per tick, all four columns measured back to back in one session (`MODS="bb2 bb3 bbb" bench/matrix.sh`):

| per saturated 4×4 balancer, per tick | bb2 | bb3 | **bbb** | vs bb2 |
|---|---|---|---|---|
| express, whole tick | 21.9 µs | 23.1 µs | **0.49 µs** | **45×** |
| express, mod Lua only | 19.1 µs | 21.0 µs | **0 (= the control)** | — |
| normal, whole tick | 7.55 µs | 7.67 µs | **0.35 µs** | 22× |
| idle express | 1.70 µs | 2.91 µs | **0.16 µs** | 10.6× |

200 express 4×4 balancers: **0.64 ms/tick against bb2's 4.92** — 4% of the 16.67 ms 60-UPS budget instead of 30%. 500 of them cost 2.05 ms/tick and still deliver the control's exact item rate. **`scriptUpdate` is the no-mod control's in every cell** and there are zero `[BBB]` log lines inside any benchmark window: all compiling happens in `--create`, which is the whole architecture in one measurement.

Two things the numbers say that are not wins:

- **`avg_ms` timings on this machine drift 25–35% between sessions** (Spotlight indexing `bench/tmp` was worth most of that). Only cells measured back to back with their own control may be compared; `BASELINE.md`'s absolute milliseconds and `RESULTS.md`'s are from different sessions and are not interchangeable.
- **~~We have a GC tail the incumbents do not~~ — FIXED, and it was ours.** ([`FKLUA-GAPS.md`](FKLUA-GAPS.md) item 17, and see "The heap diet" above for the whole pass.) M4 measured a 27.8 ms worst tick at n=200 idle express against bb2's 7.4 and the control's 4.4, with 16.75 of a 17.0 ms tick in `luaGarbageIncremental` and `scriptUpdate` at exactly zero. Two milestones then went into attributing it: M4 blamed `--persist=table`, the persist pass proved the mode was irrelevant (`packed` had the same tail), and the round-5 answer is that **the pause is 0.2 ms per MiB of guest linear memory and our linear memory was 64 MiB of dead log-line strings.** It is now under 16 MiB and the worst tick is **2.26 ms against the control's 1.42** — below bb2's 7.4 and bb3's 3.0, on the one axis where this mod used to lose. The GC *mean* was already 5× cheaper than bb2 (48 µs/tick vs 260) and still is. What remains open is a residual of ~1.3 KB per recompile in generated binding return values; it is quantified in the heap-diet section and it is upstream's, not ours.

### The round-2 re-check: did filters and batching cost anything?

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

### The round-4 re-check: the mask, the defines and `game.surfaces`

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

### The round-5 re-check: the heap diet, and the last row that was still red

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

### The round-6 re-check: the upstream gc round, and the row nobody had re-read

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

### The megabase cell — 4,376 hidden splitters, and the first 64x64 anyone built

**Every benchmark above this line is 200 copies of one 4x4.** A megabase is not that, and the two things this repo had never measured are the two a megabase is made of: a MIX of shapes, and shapes past `P = 8`. The `mega` scenario builds both — ten shape classes per block, plus a 16x16, a 32x32, a 64x64 (which is `plan.MaxPorts` exactly) and a deliberately over-limit 65-input cluster. [`bench/README.md`](bench/README.md) documents the population and the two log families it adds; this section is what it measured.

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

**Read the medians above, not the TSV's `script_us`.** That column is a MEAN over the verbose pass's steady half, and this save's meter drains 4,404 sink chests six times a run at ~1 ms a time — which averages to ~3.4 µs of every tick and is where the TSV's `script_us` of 3.44 (control) and 3.60 (bbb) comes from. It is the harness's own cost, it is the same on both sides, and it swamps a median tick of 0.2 µs. The rows are in [`bench/baselines/results.tsv`](bench/baselines/results.tsv) tagged `mega r1..r3`.

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

#### Correctness first, and it is unanimous

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

**The 65-input cluster is refused, once, and the world is untouched.** Exactly one `[BBB] alert: cluster … would need 128 ports for 65 inputs and 1 outputs, over the limit of 64; refused BEFORE the teardown` in every bbb create, and **zero `[BBB] error:` in any log of the whole session** — which is the other half of the assertion, because the failure mode this replaced was three errors and a demolished network per belt ([`agents/maxports.md`](agents/maxports.md) §4). `run.sh` fails a mega cell that does not carry the alert.

#### The 64x64: what it costs to build and what it costs to touch

`[BBB] compiled cluster 5683 64->64 over 64 ports, **1152 entities**, slot 403` — against the ~1,150 `agents/maxports.md` §1 derives by arithmetic, which is the first time that arithmetic has been checked against a real one.

**The first compile**, measured at create by the audit subtraction described in `bench/README.md` — an audit with nothing pending, then the 64x64 built, then the audit that compiles it. Three creates in this session:

| | audit only | audit + the 64x64 | **difference** |
|---|--:|--:|--:|
| saturated create A | 1,710.97 ms | 1,858.15 ms | **147.18 ms** |
| idle create | 1,830.32 ms | 1,968.30 ms | **137.98 ms** |
| saturated create B | 1,796.50 ms | 1,958.00 ms | **161.50 ms** |

(The audit itself is 1.7–1.8 s for 403 clusters. It is a whole-save re-classification, it happens once inside `--create`, and it is why the measurement has to be a subtraction.)

**The recompile hitch — the number [`agents/maxports.md`](agents/maxports.md) §3 was waiting for.** M2's tick-pair pattern: one input belt of the 64x64 removed and put back, the profiler opened in the tick that mutates and closed in the tick that flushes, median of three, minus that run's own `idle tick pair, nothing pending` control.

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

## Environment

- Factorio **2.1.14** (Steam, Space Age, mac-arm64) at `~/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/` — binary at `MacOS/factorio`, supports `--benchmark`, `--benchmark-ticks/-runs`, `--mod-directory`, headless. `--benchmark` never saves. **There is no 2.0 binary on this machine any more**, which is what makes the nine multi-edge suites unrunnable rather than merely red, and what the `test/fixtures-2.0/` saves exist for: they are the last 2.0.77 suite runs' worlds, preserved 2026-08-22, and cannot be regenerated. **2.1 also removed `LuaGameScript::create_profiler`** — `helpers.create_profiler` is the replacement, every test mod and the bench harness carry the old call, and the guest calls no profiler at all. Every number in this file that predates 2026-08-24 was measured on **2.0.77**.
- User dir: `~/Library/Application Support/factorio/` (mods/, saves/).
- FkLua toolchain: `../FkLua/bin/fklua`; guest build needs TinyGo 0.41.1 + binaryen. Guest workflow: `tinygo build -target=wasm-unknown -scheduler=none -gc=custom -opt=2` then `fklua mod <wasm> --persist=packed` -- the collected arm needs no `--gc` because `fklua.toml` carries `gc = "collected"`. `-gc=leaking` + `--gc=leaking` is the other arm, and the two flags move together behind the Makefile's `GC` stamp.
- Reference mods under scratchpad `existing-mods/` (not committed).
- Where FkLua does not do what this mod needs, the gap is written down in [`FKLUA-GAPS.md`](FKLUA-GAPS.md) and worked around here. **Do not fork or patch FkLua for it** — that project wants the feedback, not a divergent copy.

## The standardization pass — what a reader from `fklua init` would find odd

**BBB is meant to read as what a regular FkLua mod looks like, and it is five milestones and six upstream fix rounds older than the scaffold that now defines that.** This section is the audit: every place this mod differs from what `fklua init <name> --lang go` writes today, or from the two refined `fklua-ports` guests, is either **standardized** or **kept with the measurement that justifies it**. Nothing is left as "that is just how it grew".

Run 2026-08-03 against FkLua at `1f1f576` (six rounds newer than the last integration), the live scaffold output, and `ports/qol-research` + `ports/resource-marker`.

### The stale pair, which had to be fixed before anything could be assessed

`fklua gen-bindings` + `fklua lock`, then a clean rebuild, **before** the audit proper — because the committed bindings predated rounds B1a/B1b/B2 (members 4,191 → 4,250) and **member ids moved**. Verified rather than assumed: `LuaControl.Insert` went from **member id 316 to 317**. Ids are dense indices over the generated set and are only ever meaningful against the `fk_api_gen.lua` emitted from the same generation, so a stale `fkapi.go` packaged by today's `fklua mod` calls a *different function*, silently, with no status and no error. That is the miner's-pocket insert, among others.

**Fallout: one real failure, and it was upstream's semantics rather than a bad id.** The `mar` suite began reporting `deadlines=3` against a documented zero — traced to the collector's new root-scan *reserve*, fixed by re-deriving one constant, and written up in "The floor upstream built, and the check it retired". Every other suite passed unchanged in both arms.

### The table

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

### The verification, and the two gates the pass was not allowed to move

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

### What did NOT change, and why that is the finding

**The architecture is not a deviation.** Every quirk this pass examined turned out to be either load-bearing with a number behind it, or already the convention — in two cases (`gcArmIfNeeded`, `logline.go`) because the campaign copied it *from here*. The one thing that was genuinely stale was the bindings, and the one thing that was genuinely wrong was a constant that upstream's own round had moved out from under.

## Index of `agents/`

| File | Covers |
|---|---|
| [`agents/design.md`](agents/design.md) | **Read before implementing anything.** The compiled-network architecture, why the incumbent loses, the edge-interface problem, milestones, feature bar |
| [`agents/spike-s1.md`](agents/spike-s1.md) | The empirical record behind the edge interfaces: the collision-mask door, linked-belt mechanics, why the lane-splitter stage is not optional |
| [`agents/docs-style.md`](agents/docs-style.md) | **Read before creating or editing any human-facing document** (`README.md`, `FKLUA-GAPS.md`, the bench and test READMEs). The public-audience house style: what must not appear, the formatting rules, the structure per document type, the licence wording, and the grep check to run before committing |
| [`agents/single-edge.md`](agents/single-edge.md) | **The Factorio 2.1 port.** Why one belt per balancer part is a rule change rather than an interface redesign, the two questions the capability marker and the runtime setting answer, the refusal semantics, the grandfather pass, what a 2.0 multi-edge save meets on a 2.1 load (the engine prunes the interfaces silently, measured), the packaging plan, the test estate, and the implementation-status section for what phase 1 actually shipped -- with the `sedge` suite's numbers, its red proof and the two places the implementation had to deviate from the design |
| [`agents/estate-port.md`](agents/estate-port.md) | **The test estate, in Go.** The programme for removing the last hand-written Lua in the repository: fourteen suites' observer mods, the interactive staging mod, two data-stage-only stand-ins and the bench harness, in eight phases. What an observer is and the two shipped-guest rules it does not inherit, the one-module/N-mains layout and the measurement that says an observer cannot bloat the mod, the packaging recipe (a neutral working directory, every identity a flag, `--api` not optional), the `copy_testmod` staging seam, the `[BBB-OBS] error:` gate that replaces Lua's `error()`, the six gates every phase must clear, the harness surface, and a record per phase -- the PILOT (`m1`, `sedge`), phase 2 (`mar`, `mig21`, `qual`), phase 3 (`mix`, `plat`, `mig`), phase 4 (`m2`, `m3`, `edge`) phase 5 (the interactive staging mod, which was never a suite), phase 6 (the two data-stage-only stand-ins, the first packages here with NO CONTROL STAGE) and PHASE 7 (the `bench/` harness's setup mod, the last one this machine can do), all 2026-08-25: every tagged log line byte-identical in order, a red proof each, the api-check verdicts that turned the stamped-vs-gated asymmetry into a number and produced the first `impacted` observer, phase 3's verification of `fkapi.Log(Value)` against a golden profiler line and phase 5's of `fkapi.RemoteCall` -- both before anything was built on them. **Phase 7's gate is not a golden log**: `bench/` is MEASUREMENT INFRASTRUCTURE, so what it owes is a COMPARABILITY run -- the same cells, both setup mods, interleaved in one session against one dist/ -- and it is where the `--persist=table` decision, the missing `on_nth_tick` binding and two engine findings about Factorio 2.1.16 are written up |
| [`agents/maxports.md`](agents/maxports.md) | The 64-port cap: where it comes from (our slot geometry, not the engine), the three constraints an uncap must clear (size-class slots, heap buffers vs the root-scan gate, the per-edit hitch bound) — and §4, DONE 2026-08-04, what hitting the cap does now, with the before/after measurement and the one shape (a merge into an over-limit cluster) still not covered |
| [`agents/single-edge.md`](agents/single-edge.md) | **DESIGN, not implemented.** The Factorio 2.1 port: 2.1's fixed collision-mask validation closes the `not_colliding_with_itself` door the edge interfaces stand on (boskid's report and answer, 2026-08-23/24), so the rule becomes one belt per part — the multi-edge setting (runtime-global, 2.0 only, default off, script-grandfathered UP on the first load of a save updating from the first GA release: dirty saves keep multi-edge and are warned once, clean saves land on the new default), the limit.go-style refusal, the migration of multi-edge saves, the interactive/GIF world rework, packaging as two releases from one tree. The two gating S2 probes are MEASURED: a 2.1 load of a 2.0 save silently deletes all but one belt-connectable per tile (no crash, no log line, hidden network intact, crippled delivery), and a fresh single-edge network on 2.1.14 runs at full rate. The 2.0.77 fixture saves are committed under `test/fixtures-2.0/` and cannot be regenerated without a 2.0 binary |
