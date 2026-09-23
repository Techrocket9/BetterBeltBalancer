# Repository layout

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
                         subscriptions, logging and the load hooks -- including
                         `fk_state_version`, the one-integer WATERMARK a load
                         compares itself against to know which rules the save
                         it is reading was written under (see stateversion.go
                         below); a rule that changes what a standing world
                         MEANS bumps it and adds a rung rather than adding a
                         setting, cluster.go is the registry,
                         compile.go is the network compiler, and
                         classifySide is where a cluster's edge list comes
                         from -- a straight half (classifyStraight, the six
                         belt-connectable types read off direction and end
                         type) and a CURVED one (curvesFromCluster), which is
                         the belt running ACROSS a face that the engine bends
                         towards us once we feed it. The curve arm is four
                         tests over two tiles that are NOT adjacent to the
                         cluster, it declines a side-load on purpose because a
                         half-lane port breaks exact balance, and it allocates
                         nothing -- read its header before touching either
                         half ("A belt that turns as it leaves"). SINCE
                         2026-09-07 THE CURVE HALF IS BEHIND A SETTING, and
                         classifySide asks the policy before it probes, so a
                         save that turned it off makes neither host call,
                         curve.go is that setting: `bbb-curved-exits`, a
                         runtime-global bool defaulting to TRUE and defined on
                         BOTH engines, its per-heap tri-state cache, and the
                         flip handler, which re-queues EVERY cluster and lets
                         each one skip on the fingerprint it never lost. A
                         balancer whose only outputs were curves ends with
                         inputs and nothing else, which is a half-built state
                         rather than a refusal, and its network's contents
                         reach the ground because a machine that no longer
                         exists is a removal ("The curve rule is a setting"),
                         curveupg.go is what an UPGRADE has to do about that
                         setting: the first load of a save built before the
                         rule existed turns it OFF for that save, so a factory
                         somebody already had keeps the reading it was built
                         to. THE TRIGGER IS THE SAVE'S OWN STATE VERSION --
                         `fk_state_version`, stamped by FkLua and handed back
                         to fk_migrate, 0 for every build up to 0.3.2 --
                         rather than a shape recognised in the world, which
                         is the 2026-09-07 redesign: the shape it replaced
                         was an adoption failure of one exact list, and three
                         ordinary saves failed it and were recompiled under
                         the new rule as though they had been built to it.
                         While a load is UNDECIDED the signature is one curve
                         edge anywhere, the decision is taken at the first one
                         inside classifyEdges and applied by dropping them,
                         and there is no anchor and no second setting: the
                         save is stamped at this rung the next time it is
                         written ("A save from before the curve rule"),
                         stateversion.go is the rung table that watermark
                         reads against, and stateversion_prestate.go is the
                         `-tags prestate` fixture the `curv` suite creates
                         its save with,
                         globalsetting.go is `settings.global` read and
                         written, the ten lines both runtime-global bools
                         share -- sedge.go keeps the capability gate on its
                         write and the tri-state its fold wants,
                         priority.go is the PLAYER'S SIDE OF A PRIORITY PORT:
                         three doors (a keybind, a remote method and a settings
                         paste) onto one function, `setPartPriority`, which is
                         the only thing that moves `pprio`. The flag is in the
                         compile fingerprint, so a toggle is a RECOMPILE and
                         never a removal, and the file's two pre-teardown checks
                         are why: `plan.ShapeEdges` asked with the flag
                         speculatively flipped, so a shape that could not be
                         built leaves the balancer alone, and
                         `prioFitsWhatIsStanding`, which refuses a change that
                         would build a network smaller than what the machine is
                         carrying rather than putting the difference on the
                         ground. Read its header before adding a fourth door or
                         a fifth refusal ("Priorities: a port that is fed
                         first"),
                         lifecycle.go is
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
                         it existed -- and TWO SETTING METHODS that are the
                         only script route to this mod's own runtime-global
                         settings, because Factorio lets nobody but the
                         defining mod write `settings.global`:
                         `set-multi-edge-parts` since 2026-08-24, inert on 2.1
                         where that setting does not exist, and
                         `set-curved-exits` since 2026-09-07, which is the
                         first of them a suite can drive on EITHER engine and
                         is what the `m2` suite flips -- and
                         `set-part-priority`, which is not a setting at all and
                         is there for the sharpest version of the same
                         argument: the flag's own door is a KEYBIND, a keypress
                         cannot be issued from a script, and a headless run has
                         no player to press one,
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
                         laid on a part -- and a SCRIPT's fast replace raises no
                         event at all for the part it destroys, so without the
                         check there the registry keeps a tile it calls a part
                         which is holding somebody's belt. A PLAYER'S belt over a
                         part DOES raise the part's own mine first, measured
                         2026-09-14, and takes the ordinary removal path. The
                         file's second half is the FORWARD direction's one sharp
                         edge: a part dropped into the middle of a belt line is
                         refused on 2.1 and handed back, and the belt the engine
                         destroyed to make room for it is put back on its tile
                         and charged to the player, which needs `on_pre_build` to
                         tie the mine and the build together ("Fast replace").
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
                         parks 4/0 on every output without it.
                         SINCE THE PRIORITIES FEATURE IT BUILDS TWO SHAPES.
                         `Edge.Prio`, `Ports.QIn/QOut` and `Op.OutPrio` are the
                         seam; `ShapeEdges` is the ONE pre-teardown check and is
                         safe to ask from a keypress (it allocates nothing and
                         reads counts); `buildPrio` is the two-butterfly
                         construction behind a priority output; and `Capacity`
                         says how many item positions a shape's network holds,
                         which is what lets the guest refuse a toggle that would
                         spill rather than spilling. INPUT priority is refused
                         at every size -- the construction for it is not built
                         and an approximation is the one outcome this repo's own
                         rule forbids. agents/priority.md is the whole of it
guest/go/skin/           the sprite-variant mapping: a neighbour mask in, one
                         of 47 pictures out -- or one of 94, because the second
                         half of the sheet is the same 47 shapes with a PRIORITY
                         BADGE on them and `skin.Cells` is what the prototype's
                         `variation_count` and the committed PNG's cell count
                         both are. `Variation` and `IsPriority` are the two ends
                         of that: the flag rides in `graphics_variation`, which
                         is per-entity state the ENGINE persists, so it survives
                         a blueprint and a rebuilt guest where the heap does
                         not. Pure Go for the same reason, and `make check` runs
                         its five named shapes and the sheet's own header
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
                         the tiles that were already there -- which the CURVED
                         EXIT made a longer proof and did not break, since a
                         part landing on a candidate belt's rear or far tile
                         is a part tile there and the curve arm declines on
                         exactly that. Since phase 2 it
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
                         family beside them -- PLUS priority.go, which is the
                         one custom-input prototype this mod defines
                         (`bbb-toggle-priority`, ALT + P) and therefore the one
                         thing that makes the control guest's `SubscribeNamed`
                         succeed: a name that does not match is a keybind that
                         silently never fires, and the name is written twice
                         because neither guest may import the other's packages
                         -- and MINUS item.go, which is gone
                         since round two of the FkRecipes migration: the item is
                         a `LegacyItem` declaration in guest/go/tune's plan and
                         main.go's fk_data hook calls `EmitData` after entity(),
                         because the item's `place_result` names that entity and
                         the library PRESENCE PROBES it. It may NOT import fkapi and
                         `fklua mod` refuses a data module that does: there is
                         no game, no script and no runtime API at these stages.
                         Read "The shipped mod holds no Lua" before touching it,
                         and main.go's //go:noinline block before ADDING to it --
                         Lua 5.2 cannot compile an arbitrarily long function and
                         this stage is exactly the shape LLVM inlines into one.
                         THREE OF ITS DECISIONS ARE NOT CONSTANTS ANY MORE, and
                         two of them are not this package's any more either:
                         what a balancer part costs to build and to research is
                         a startup setting, DECLARED AND RESOLVED BY FkRecipES
                         out of guest/go/tune's plan since round two, so
                         recipe.go and technology.go are deleted with item.go;
                         and hidden.go's `deriveHiddenSpeed` runs at final-fixes
                         and gives the four hidden prototypes the speed of the
                         fastest belt any mod loaded, which is still this
                         package's own. "Cost, research and belt speed"
guest/go/tune/           WHAT THE DATA STAGE DECIDES that is not fixed: the
                         recipe's ingredients, the research's cost, and the
                         hidden network's belt speed. PURE Go, the SIXTH package
                         to earn that, for a reason only `engine` shares -- THE
                         INTERESTING STATES ARE STATES OF SOMEBODY ELSE'S MOD
                         SET. A fallback rung is taken only in a game with no
                         `express-transport-belt` in it and no mod set this
                         machine can install has that shape, so inside
                         guest/go/data those arms would be branches nothing
                         could execute. SINCE ROUND TWO IT KEEPS THE LADDERS AND
                         STOPPED WALKING THEM: `Resolve` and `ResolveRecipe` are
                         deleted and FkRecipes' own `IngredientNamed` ladder does
                         it, with the same safety argument one layer out -- an
                         ingredient naming a prototype nobody defined is a HARD
                         LOAD FAILURE with this mod's name on it, in somebody's
                         overhaul pack, so no name reaches data:extend that the
                         game was not asked about first.
                         plan.go is THE FkRecipes PLAN, and since
                         2026-09-01 it is where this mod's six startup settings
                         are declared: two `LegacyDropdownSettingNeedingLocale`
                         calls, which is the constructor that carries a shipped
                         mod's own names and order strings across VERBATIM,
                         because Factorio keys mod-settings.dat by name and has
                         no rename mechanism, and since the sync pass the four
                         CUSTOMIZER fields as GENERATED names placed under their
                         dropdowns by `OrderAfter`, because nothing has shipped
                         them. Since fix round 2 those four fields are the
                         SWITCH: a text that does not say `default` and a
                         number that is not 0 decide, and neither dropdown
                         carries a value for them. SINCE ROUND TWO IT DECLARES THE
                         ITEM, THE RECIPE AND THE TECHNOLOGY TOO, through
                         `LegacyItem`, `LegacyRecipe` and `LegacyTechnology`,
                         which is the same
                         argument with a wider blast radius: a prototype name is
                         held by blueprints, logistic requests, crafting queues
                         and this mod's own hand-rolled entity, whose
                         `minable.result` a prefixed item killed the load on --
                         measured, not feared. plan.go also carries the NAMES
                         AND THE TWO SORT KEYS three files have to agree about
                         (PartName, TechName, PartIcon, PartOrder, TechOrder),
                         so entity.go and legacy.go name one constant rather
                         than four literals
                         ("The settings are declared through FkRecipes").
                         It also carries the two checks nothing else in
                         this repo could make. The LOCALE one is FkRecipes'
                         own `CheckLocaleWith` now, run against this plan and
                         told the one setting declared outside it, which is
                         strictly more than the hand-rolled pair of walks it
                         replaced -- it polices name and description ORPHANS
                         against the mod's whole set of settings rather than
                         against the mod prefix, so an entry left behind by a
                         rename is caught (measured: the plain `CheckLocale`
                         finds it 0 times and `CheckLocaleWith` once) -- plus
                         three assertions of this mod's own the library
                         deliberately declines to make, which read the file
                         through `fkrecipes.LocaleEntries` since round two, so
                         the checker and the extra assertions are one reading of
                         one file rather than two parsers over one grammar. The
                         other is the PLAN, every field of every prototype it
                         declares against transcribed literals, on the host with
                         no engine: the settings against a one-method `Named`
                         stub and the item, recipe and technology against a
                         fixture World
guest/go/engine/         WHICH FACTORIO THIS IS, and the FIFTH pure package. One
                         function: is `mods["base"]` 2.0.x. Both data-stage
                         hooks call it -- the collision flag and the marker on
                         one side, the runtime setting on the other -- which is
                         what RETIRED mod-data/engine.lua rather than porting it:
                         two exports of one compiled module have no second copy
                         to drift, where two Lua states had to share a required
                         file. Pure for edgemode's reason exactly: the `true` arm
                         emits prototypes only a 2.0 binary can be shown, so its
                         dump golden was DEFERRED until a 2.0 binary existed
                         and is CAPTURED since 2026-08-26; `make check` proves
                         the DECISION regardless, on either engine. It is also the only shape a test
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
                         and obs/qual are PHASE 2 the same day -- and obs/mig21
                         is the only observer with NO fk_on_init at all, since
                         it builds no world and its whole "before" is
                         fk_on_configuration_changed, AND THE ONLY ONE WHOSE
                         PACKAGING IS A CORRECTNESS SURFACE: it must not depend
                         on better-belt-balancer, because mod load order is what
                         decides whether its sample is taken before the
                         migration or after it, so the Makefile's --dependency
                         list is red-proven rather than merely written down. And
                         obs/mix,
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
                         obs/curv and obs/curvdata are the estate's youngest
                         and are not a port at all: they are the `curv`
                         suite, whose whole trick is that the observer
                         forces the compile with an audit marker and lays
                         the curve belts AFTERWARDS with no `raise_built`,
                         so the save carries networks built to a rule the
                         guest that built them no longer has.
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
                         obs/bench and obs/benchdata are PHASE 7, the bench
                         harness's setup mod, and the only package here whose
                         consumer is bench/run.sh rather than a
                         test/assert-*.py. obs/mig21 is here with the other
                         thirteen and it is the one that LEFT AND CAME BACK: it
                         was re-ported to RUST on 2026-08-25 as phase 8's parity
                         exercise and RESTORED HERE, verbatim out of git
                         history, on 2026-08-26 -- see "Pure Go" below. They are
                         thin `main`s
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
                         It is the ONLY thing in here since the sync pass of
                         2026-09-08. mod-settings.py, this repository's own
                         transcription of Factorio's mod-settings.dat (the
                         only way to ask a settings stage a question from
                         outside), is DELETED for the toolchain's writer,
                         `fklua modsettings write` (FkLua c21ff07 or later):
                         both callers moved, test/check-datastage.py, which
                         drives the shipped mod's cost settings, and
                         bench/run.sh, which CONFIGURES the bench harness's
                         setup mod, a compiled guest that cannot require a
                         Lua config. THERE IS A THIRD SINCE 0.3.3's curve
                         rule, test/run.sh's curve_setting_off, which writes
                         the one map setting the curv suite's second leg
                         needs. All three probe the binary and refuse NOT
                         RUN, naming the remedy, when it lacks the command;
                         the first two probe before any engine run and the
                         third before its own leg's, because it is the only
                         leg of the only suite that asks for the command
test/                    headless verification, SEVENTEEN suites (see below).
                         EVERY SUITE'S OBSERVER IS A COMPILED GUEST under
                         guest/go/obs since 2026-08-25, phase 4 having taken
                         the last three and biggest -- `m2`, `m3` and `edge` --
                         and phase 5 the interactive staging mod, which was
                         never a suite, phase 6 the two data-stage-only
                         stand-ins and phase 7 the bench harness's setup mod
                         (agents/estate-port.md). PHASE 9 TOOK `flip` ON
                         2026-08-26, on a Factorio 2.0 binary, and it is the
                         one that waited for a BINARY rather than for a phase:
                         its suite SKIPS on 2.1, and a phase's first gate is a
                         golden log there was no run here to produce. ZERO LUA
                         FILES AND ZERO LINES are left in the whole repository,
                         from twenty-four and 8,524, and `test/mods/` does not
                         exist.
                         run.sh's `copy_testmod` is the seam, and it is a
                         packaged observer out of dist/obs and nothing else
                         now: the Lua fallback beside it existed to know which
                         half a suite was in, there is no other half, and a
                         directory that does not exist is not a safety net.
                         What replaced it names the stale-package hazard phase
                         8 found -- a package does not depend on the recipe
                         that produced it, so an identity-only change with no
                         source change beside it re-packages nothing.
                         run.sh STAMPS every staged mod's info.json for the
                         engine it read off the binary, because this mod ships
                         on two arms out of one tree. Twelve answer the same on
                         either; `mig21` and `mig` INVERT and take --engine,
                         and `flip` -- which drives the multi-edge setting --
                         exists on 2.0 alone and prints a SKIP on 2.1 rather
                         than passing. `mig21` does not BUILD a multi-edge
                         world at all: it LOADS one out of fixtures-2.0/, which
                         is the only way a 2.1 binary can ever be shown one.
                         `curs` is the SECOND suite with no --create and is
                         fixtures-player/'s: what a save has to carry for it is
                         a PLAYER, which nothing a script can do creates and no
                         map generator has ever made, so its phase one is a save
                         a graphical CLIENT made, reduced to one player on nine
                         chunks by guest/go/obs/fixplayer and committed. It is
                         the suite every "the trigger needs a player" statement
                         in this file now points at. And
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
                         fixtures-player/ is the OTHER committed save and is
                         not a world at all: one player, one nine-chunk nauvis,
                         no entities and no second surface, recorded against
                         BASE ALONE. `curs` loads it and then builds its own
                         rigs into it, which is what every --create suite does
                         to a map generator's world. `make player-fixture
                         SAVE=<a save a client made>` re-cuts it from any client
                         save, through test/make-player-fixture.sh and
                         guest/go/obs/fixplayer; the source save is copied
                         before anything reads it and is never committed. See
                         "A committed save with a player in it".
                         fixtures/fastbelt is not a test mod at all but a whole
                         FACTORIO MOD WRITTEN IN GO, built by the dump gate and
                         thrown away: it defines a belt at 0.4 and an UNDERGROUND
                         at 0.5, which is the only way to show this machine a
                         belt faster than the hidden network's own floor. The
                         underground is the faster on purpose -- the 0.5 the gate
                         asserts can only come from a scan that walks more than
                         one belt family. Its `inert` package is an empty control
                         guest and exists solely because `fklua mod` cannot
                         package a data-module-only mod (FKLUA-GAPS.md item 26).
                         check-release-arm.sh is the odd one out and asks
                         nothing of Factorio: it is git plumbing over two REFS,
                         `master` and `release/2.0`, and it says the 2.0 arm
                         carries no line trunk never had. `make check` runs it
                         in 0.23 s; a clone without the branch SKIPs at exit 0
                         and prints the fetch command, and a SHALLOW one SKIPs
                         too. See "The release arm" under Verification.
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
