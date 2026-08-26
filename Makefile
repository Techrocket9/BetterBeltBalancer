# BetterBeltBalancer build pipeline.
#
#   make guest    TinyGo -> dist/bbb.wasm (GC=leaking for the other arm)
#                 AND dist/bbbdata.wasm, the data-stage guest
#   make mod      fklua mod -> dist/<name>_<version>/, both guests included
#   make zip      the same, as an installable dist/<name>_<version>.zip
#   make install  the built mod into a Factorio mods directory
#   make test     headless verification, FOURTEEN suites, and the default is all
#                 of them. WHICH FACTORIO IS INSTALLED IS AN INPUT: test/run.sh
#                 reads the series off the binary, stamps every staged mod's
#                 info.json for it, and gates the packaged mod against it --
#                 and three suites answer differently per engine (`mig21` and
#                 `mig` invert; `flip` runs on 2.0 alone and prints a SKIP on
#                 2.1). SUITES=m2 names one. See test/run.sh's SUITES block and
#                 agents/single-edge.md.
#   make check    the generated bindings and the lock are current
#   make datastage-check
#                 the data stage's own gate: Factorio's --dump-data, hashed
#
# `fklua mod` GENERATES the control stage (control.lua, info.json, fk_abi.lua,
# fk_api_gen.lua, fk_module.lua) and REPLACES its output directory each time, so
# nothing hand-written can live there.
#
# THERE IS NO HAND-WRITTEN LUA IN THIS MOD AT ALL. The settings and data stages
# were ten Lua files under mod-data/ and are a second compiled guest now
# (guest/go/data -> dist/bbbdata.wasm, declared in fklua.toml as `data_module`);
# `fklua mod` writes settings.lua, data.lua and data-final-fixes.lua for it, one
# per hook the module exports. What is left in mod-data/ is assets -- graphics,
# locale, changelog, thumbnail -- merged into the package before the directory
# writer AND the zip writer, so both carry the same bytes.
# Never edit anything under dist/.

SHELL := /bin/bash

FKLUA        ?= ../FkLua/bin/fklua
FACTORIO_BIN ?= $(HOME)/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio
MODS_DIR     ?= $(HOME)/Library/Application Support/factorio/mods

# Single source of truth: fklua.toml, and `fklua mod` READS IT -- identity,
# dependencies, the asset directory and the data module all come from the
# manifest, so the package command below types none of them. Every one of them
# HAS a flag form (--name, --version, --dependency, --data-module and the rest);
# they exist so one checkout can package several mods from one manifest, and a
# flag overrides the key, which is what `make GC=leaking` does. Passing them
# here would put a second statement of this mod's identity in the build.
#
# The two below are needed only to NAME the output that command produces.
MOD_NAME    := $(shell sed -n 's/^name = "\(.*\)"$$/\1/p' fklua.toml)
MOD_VERSION := $(shell sed -n 's/^version = "\(.*\)"$$/\1/p' fklua.toml)
# ...and these two are read for the TEST OBSERVERS, which are packaged from
# their own flags rather than from this manifest (see the observers block near
# the bottom). They are the two identity facts an observer MUST share with the
# mod under test: the API description its generated bindings came from, and the
# engine series its info.json declares.
MOD_API     := $(shell sed -n 's/^api = "\(.*\)"$$/\1/p' fklua.toml)
MOD_SERIES  := $(shell sed -n 's/^factorio_version = "\(.*\)"$$/\1/p' fklua.toml)

DIST      := dist
WASM      := $(DIST)/bbb.wasm
# The data-stage guest. Its path is fklua.toml's `data_module` and `fklua mod`
# reads it from there, so this variable exists only to BUILD the file and to
# depend on it; the package command is passed no flag for it.
DATA_WASM := $(DIST)/bbbdata.wasm
MOD_DIR   := $(DIST)/$(MOD_NAME)_$(MOD_VERSION)

# --gc. `make GC=collected` builds the guest with FkLua's paced conservative
# collector (../FkLua/agents/gc.md) instead of the mandatory-by-default leaking
# arena. TWO flags move together and neither works alone: TinyGo needs
# `-gc=custom` so the collector's seven //go:linkname hooks are the allocator,
# and `fklua mod` needs `--gc=collected` so the emitter keeps the inlined 8-byte
# store out of line (the one store shape that would otherwise write MEM without
# marking its page, which under a collector is a use-after-free rather than a
# stale save). guest/go/gc.go is the import and the one call site.
#
# THE DEFAULT IS `collected` AND IT IS THE SHIPPED BUILD, decided THREE times.
# The full history and every table is in CLAUDE.md, "The collected-mode
# postscript"; the short form is that the decision turned twice on what the
# measurement was of.
#
#   2026-08-01  leaking. Collected cost 152 ticks of collector script after
#               every load and bounded a heap that a fresh save did not need
#               bounding. Both halves of the rule failed.
#   2026-08-02  leaking, re-affirmed, with a recommendation to re-measure: the
#               marathon suite found the heap a 300-hour multiplayer save
#               actually reaches, and the 152 ticks turned out to be structural
#               to a mass-BUILDER rather than to play.
#   2026-08-02  collected, RE-MEASURED on today's sharded pin -- which is what
#               the previous decision asked for and did not do.
#
# What the re-measurement found, both arms interleaved, n=200 k=4 express,
# 5 reps: the steady state cannot tell them apart (saturated avg_ms 0.797
# leaking against 0.757 collected over a 0.594 control; scriptUpdate 1.5-2.9 us
# in every arm INCLUDING the no-mod control), the post-load transient is
# 152 ticks -> 71 and 105 ms -> 68 (and -> 36 ticks / 54 ms since the root-scan
# budget fix of 2026-08-03), and the 4x4 recompile hitch is 12.59 ms
# leaking against 7.06 collected. What is left is size: +32.4% of fk_module.lua,
# +13.7% of the zip, +2.4 ms on the first tick after a load, and 73,112 B of
# collector metadata (not the 163 KiB the first decision was priced on --
# upstream's sharding stage C deleted the static reservation).
#
# AND WHAT IT BUYS IS A DEFECT CLASS, MEASURED RATHER THAN PROJECTED. Under
# leaking every transient allocation is permanent, TinyGo's growHeap DOUBLES,
# and `mem_grow` writes a zero into every new word. Driven up the ladder with
# 3,400 4x4 teardown-and-rebuilds, the leaking arm's grow ticks came out 48.7 ms
# at 2->4 MiB, 120.3 at 4->8, 226.1 at 8->16 and **782.4 ms at 16->32** -- one
# tick, every client at once, and nothing downstream can bound it because it is
# a fact about the growth law. The collected arm ended the same 3,400 operations
# on 0.52 MiB of linear memory against 31.9, with a worst tick of 71 ms.
#
# `make GC=leaking` stays buildable and green, and all ten suites are run in
# both arms on every pass that touches this decision.
#
# THE SHIPPED ARM IS IN fklua.toml (`gc = "collected"`) AND NOT IN A FLAG HERE,
# which is the standard shape: `fklua mod` reads the manifest, so the default
# build types no --gc at all. What this block does is build the OTHER arm, by
# passing the flag that overrides the manifest -- so there is exactly one place
# that says what ships, and the override is visible at the point of use.
GC ?= collected
ifeq ($(GC),collected)
TINYGO_GC := -gc=custom
FKLUA_GC  :=
else ifeq ($(GC),leaking)
TINYGO_GC := -gc=leaking
FKLUA_GC  := --gc=leaking
else
$(error GC must be leaking or collected, got '$(GC)')
endif

# Every flag is load-bearing; see ../FkLua/agents/guests.md. -opt=z (TinyGo's
# default) optimises for size, which is the one cost this target does not have.
TINYGO_FLAGS := -target=wasm-unknown -scheduler=none $(TINYGO_GC) -opt=2

# The [BBB] logging switch. `make QUIET=1` compiles guest/go/log_quiet.go
# instead of log_verbose.go, and every line below the error level is eliminated
# rather than merely skipped -- verboseLog is a constant and every call site is
# guarded by it. Errors are never switched off.
#
# The DEFAULT IS VERBOSE and `make test` depends on it: the guest's own log lines
# are the assertion surface for both suites.
ifeq ($(QUIET),1)
TINYGO_FLAGS += -tags bbbquiet
endif

# --persist=packed. M1 chose packed, M2 changed it to table on a measurement, and
# this is the THIRD decision, taken after upstream replaced packed's min/max
# BYTE-RANGE dirty watermark with a dirty PAGE SET (CLAUDE.md, full tables).
#
# The span pathology is what made packed unshippable: one host call touched the
# static scratch region and the heap, so a flush repacked everything in between
# and a 200-compile build took 447 s. With the page set that build is 45.8 s
# against table's 31.0 s -- 1.48x, not 30x -- and the whole 4x4 recompile hitch
# is 5.90 ms against table's 4.11, still a third of a tick.
#
# What packed buys is the save: at n=200 the bench save is 3.6 MB against
# table's 49.4 MB, and the load (every join, every reload) is 8.2 s against
# 21.6 s. What it does NOT buy is the idle GC spike -- the LIVE linear memory is
# a Lua word table in BOTH modes, only `storage`'s copy differs, so the
# collector walks the same thing either way. Measured: unchanged.
#
# `none` is still not an option: it would rebuild from the world on every load,
# in each client's own order, which is a multiplayer desync (CLAUDE.md).
PERSIST := packed

# The control guest is everything under guest/go EXCEPT the generated bindings
# and the data guest's own main package -- the two are compiled separately and a
# shared source list would rebuild each of them whenever the other moved.
#
# `guest/go/obs` is excluded for the same reason `guest/go/data` is: it is a set
# of separate main packages that the control guest does not import, so a change
# to a test observer must not relink the mod, and a change to the mod must not
# re-package fourteen observers.
GUEST_SRC := $(shell find guest/go -name '*.go' -not -path '*/fkapi/*' -not -path 'guest/go/data/*' -not -path 'guest/go/obs/*') guest/go/go.mod
# The data guest: its own main package, plus the PURE packages it shares with
# the control guest. `engine` is the version branch both stages ask, `skin` is
# where the sprite sheet's cell count comes from, and `tune` is what the two
# cost settings and the belt-speed derivation decide -- none of them is copied
# here, which is the whole reason the two guests are one Go module.
DATA_GUEST_SRC := $(shell find guest/go/data guest/go/engine guest/go/skin guest/go/tune -name '*.go' -not -name '*_test.go') guest/go/go.mod
DATA_SRC  := $(shell find mod-data -type f)

.PHONY: all guest mod zip install test check datastage-check clean graphics observers

all: mod

# --- guest -------------------------------------------------------------------

guest: $(WASM) $(DATA_WASM)

# The gc mode changes the WASM and touches no source file, so a file target
# would happily hand back a module built with the other allocator. Same shape as
# PERSIST_STAMP below and for the same make-3.81 reason: switching modes DELETES
# what the other mode built rather than merely post-dating it, because make
# compares whole seconds and a stamp touched in the same second as its output
# does not count as newer.
GC_STAMP := $(DIST)/.gc-$(GC)
$(GC_STAMP):
	@mkdir -p $(DIST) && rm -f $(DIST)/.gc-* $(WASM) $(MOD_DIR).zip && touch $@

$(WASM): $(GUEST_SRC) $(shell find guest/go/plan -name '*.go' -not -name '*_test.go') guest/go/fkapi/fkapi.go $(GC_STAMP)
	@mkdir -p $(DIST)
	cd guest/go && tinygo build $(TINYGO_FLAGS) -o ../../$(WASM) .
	@ls -l $(WASM)

# THE DATA GUEST'S FLAGS ARE ITS OWN AND DO NOT MOVE WITH GC=.
#
# -gc=leaking always, and no `-tags bbbquiet` either. A data module runs ONCE, at
# load, and dies with the Lua state that built it: there is no tick to pace a
# collector against, nothing to persist, and `fklua mod` packages it
# --persist=none whatever the control guest uses. Giving it the collector would
# attach 73 KB of metadata and a pacing surface to a program with no steady state
# to pace.
#
# So it takes no GC_STAMP dependency, which is what makes `make GC=leaking` and
# the default arm share one data module rather than rebuilding it twice for no
# difference. The one flag it does share is -opt=2, for the reason the control
# guest has it: -opt=z optimises for size, which is the cost this target does not
# have.
$(DATA_WASM): $(DATA_GUEST_SRC)
	@mkdir -p $(DIST)
	cd guest/go && tinygo build -target=wasm-unknown -scheduler=none -gc=leaking \
	  -opt=2 -o ../../$(DATA_WASM) ./data
	@ls -l $(DATA_WASM)

# --- mod ---------------------------------------------------------------------

# One step. Identity, dependencies and the mod-data/ tree all come out of
# fklua.toml; the only flag is the persistence mode, which is a build decision
# rather than an identity.
#
# There is no layout check here any more. `test/check-layout.py` guarded the ABI
# constants the guest derived by hand, and there are none left: the last two
# event offsets went when `fk.subscribe` gained a field mask, and the four
# `defines.direction` values went when `gen-bindings` started emitting an
# accessor per define. See CLAUDE.md, "The layout check is gone".
mod: $(WASM) $(DATA_WASM) $(DATA_SRC) fklua.toml
	$(FKLUA) mod $(WASM) --persist=$(PERSIST) $(FKLUA_GC) -o $(DIST)
	python3 test/check-sprites.py $(MOD_DIR)
	python3 test/check-changelog.py $(MOD_DIR)/changelog.txt $(MOD_VERSION)
	@echo "mod ready: $(MOD_DIR)"

# The distributable form. It carries the data stage too, which it could not
# before: `--zip` wrote from the same five-entry map the directory writer used.
#
# The zip is a REAL FILE TARGET, not just a phony recipe, because `bench/run.sh
# --mod bbb` calls `make zip` before every matrix cell: a phony rule would
# re-package (and re-check) a dozen times for one matrix run. The phony `zip`
# is kept as the name people type.
zip: $(MOD_DIR).zip

# --persist changes the package and touches no source file, so a file target
# would happily hand back a zip built in the other mode. The mode is a stamp
# the zip depends on, and switching modes DELETES the zip rather than merely
# post-dating it: make 3.81 (what macOS ships) compares whole seconds, so a
# stamp touched in the same second as the zip does not count as newer.
PERSIST_STAMP := $(DIST)/.persist-$(PERSIST)
$(PERSIST_STAMP):
	@mkdir -p $(DIST) && rm -f $(DIST)/.persist-* $(MOD_DIR).zip && touch $@

$(MOD_DIR).zip: $(WASM) $(DATA_WASM) $(DATA_SRC) fklua.toml $(PERSIST_STAMP) $(GC_STAMP)
	$(FKLUA) mod $(WASM) --persist=$(PERSIST) $(FKLUA_GC) --zip -o $(DIST)
	@ls -l $(MOD_DIR).zip

# The 47-variant sprite sheet and the icon, both computed rather than drawn.
# Outputs are committed, so neither `make mod` nor a contributor without Python
# needs this; it is here so that a re-theme is one edit and one command. The cell
# ORDER is a contract with guest/go/skin -- see the header of either file.
graphics:
	python3 tools/make-graphics.py

# --- install -----------------------------------------------------------------

install: mod
	@mkdir -p "$(MODS_DIR)"
	rm -rf "$(MODS_DIR)/$(MOD_NAME)_$(MOD_VERSION)"
	cp -R $(MOD_DIR) "$(MODS_DIR)/"
	@echo "installed into $(MODS_DIR)"

# The rig-staging mod for the five player-gesture checks no headless run can
# make; test/interactive/README.md is the checklist.
#
# IT IS A COMPILED OBSERVER NOW, so this builds it rather than copying a
# directory of Lua -- the prerequisite is the one package and not `observers`,
# so installing the rigs does not relink the other eleven, and it touches
# nothing the shipped mod owns. The destination keeps the BARE NAME: Factorio
# accepts a mod directory without a version suffix, it is what this target has
# always written, and it is what makes the rm -rf above replace the previous
# install instead of accumulating beside it.
interactive-install: $(OBS_IACT_DIR)
	@mkdir -p "$(MODS_DIR)"
	rm -rf "$(MODS_DIR)/bbb-interactive-setup"
	cp -R $(OBS_IACT_DIR) "$(MODS_DIR)/bbb-interactive-setup"
	@echo "installed bbb-interactive-setup into $(MODS_DIR)"

# --- test --------------------------------------------------------------------

test: mod observers
	FACTORIO_BIN="$(FACTORIO_BIN)" test/run.sh $(SUITES)

# --- the test observers ------------------------------------------------------
#
# THERE IS NO HAND-WRITTEN LUA ANYWHERE IN THIS REPOSITORY, AND THAT INCLUDES
# THE TEST ESTATE. The suites' observer mods -- the mods that build the rigs,
# drive the schedule and report what they see -- were fourteen control.lua files
# and are becoming compiled Go guests, one phase at a time.
# agents/estate-port.md is the programme; `m1` and `sedge` are the pilot.
#
# ONE GO MODULE, N THIN MAINS. The observers live under guest/go/obs inside the
# mod's OWN module, which is what lets them share one generated bindings tree
# (guest/go/fkapi) and one harness package instead of vendoring either. Pruning
# is PER WASM MODULE at package time, so what an observer calls cannot reach the
# shipped mod's member table -- measured, not assumed: the packaged mod's
# fk_api_gen.lua is byte-identical either side of this directory existing.
#
# EVERY IDENTITY COMES FROM A FLAG AND THE PACKAGER RUNS FROM $(OBS_DIST).
# `fklua mod` reads the manifest in its WORKING DIRECTORY for anything it was
# not given a flag for, so packaging an observer from the repository root would
# merge this mod's `data = "mod-data"` asset tree and its `gc = "collected"` into
# a test mod. A directory with no fklua.toml in it is a package built from its
# flags alone. (test/check-datastage.py's fixture builder learned this first;
# same trick, same reason.)
#
# --gc=leaking, not the mod's collected: an observer runs for seconds in a world
# that is thrown away, so a collector would be 73 KB of metadata and a pacing
# surface for nothing. --persist=$(PERSIST), because an observer's rig registry
# is written in `fk_on_init` during `--create` and read during `--benchmark`, so
# it crosses the save exactly as the mod's own heap does.
OBS_SRC  := $(shell find guest/go/obs -name '*.go') guest/go/go.mod
OBS_DIST := $(DIST)/obs

# -opt=2 for the reason the mod has it, and -gc=leaking for the reason above.
# No GC_STAMP dependency: the observers do not move with `make GC=`.
OBS_TINYGO := -target=wasm-unknown -scheduler=none -gc=leaking -opt=2

$(DIST)/obs-%.wasm: $(OBS_SRC) guest/go/fkapi/fkapi.go
	@mkdir -p $(DIST)
	cd guest/go && tinygo build $(OBS_TINYGO) -o ../../$@ ./obs/$*

OBS_COMMON := --persist=$(PERSIST) --gc=leaking --api=$(MOD_API) \
              --factorio-version $(MOD_SERIES) --author BetterBeltBalancer

OBS_M1_DIR    := $(OBS_DIST)/bbb-m1-test_0.1.0
OBS_SEDGE_DIR := $(OBS_DIST)/bbb-sedge-test_0.1.0
OBS_MAR_DIR   := $(OBS_DIST)/bbb-marathon-test_0.1.0
OBS_MIG21_DIR := $(OBS_DIST)/bbb-mig21-observer_0.1.0
OBS_QUAL_DIR  := $(OBS_DIST)/bbb-qual-test_0.1.0
OBS_MIX_DIR   := $(OBS_DIST)/bbb-mix-test_0.1.0
OBS_PLAT_DIR  := $(OBS_DIST)/bbb-plat-test_0.1.0
OBS_MIG_DIR   := $(OBS_DIST)/bbb-mig-test_0.1.0
OBS_M2_DIR    := $(OBS_DIST)/bbb-m2-test_0.1.0
OBS_M3_DIR    := $(OBS_DIST)/bbb-m3-test_0.1.0
OBS_EDGE_DIR  := $(OBS_DIST)/bbb-edge-test_0.1.0
# THE ONE PACKAGE HERE THAT IS NOT A SUITE'S, and the one whose version is not
# 0.1.0: it is the rig-staging mod a HUMAN installs beside the real one to walk
# test/interactive/README.md, and 0.2.0 is the version its hand-written
# info.json carried.
OBS_IACT_DIR  := $(OBS_DIST)/bbb-interactive-setup_0.2.0

observers: $(OBS_M1_DIR) $(OBS_SEDGE_DIR) $(OBS_MAR_DIR) $(OBS_MIG21_DIR) \
           $(OBS_QUAL_DIR) $(OBS_MIX_DIR) $(OBS_PLAT_DIR) $(OBS_MIG_DIR) \
           $(OBS_M2_DIR) $(OBS_M3_DIR) $(OBS_EDGE_DIR) $(OBS_IACT_DIR)

$(OBS_M1_DIR): $(DIST)/obs-m1.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-m1.wasm) \
	  $(OBS_COMMON) --name bbb-m1-test --version 0.1.0 \
	  --title "BBB M1 Test" \
	  --description "Places and removes bbb-balancer-part in known patterns so the M1 cluster registry can be asserted from the log. Not a gameplay mod." \
	  --dependency "base >= 2.0.0" --dependency "better-belt-balancer" \
	  -o .

# The one observer so far with a DATA STAGE of its own: a 1x1 loader fast enough
# to saturate an express belt, which base has no buildable form of. A second
# wasm module, exactly as the mod's own data guest is.
$(OBS_SEDGE_DIR): $(DIST)/obs-sedge.wasm $(DIST)/obs-sedgedata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-sedge.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-sedgedata.wasm) \
	  --name bbb-sedge-test --version 0.1.0 \
	  --title "BBB single-edge verification" \
	  --description "Builds balancers to Factorio 2.1's one-belt-per-part rule and drives the three ways an edit can break it. Asserts nothing itself." \
	  --dependency "base >= 2.1.0" --dependency "better-belt-balancer" \
	  -o .

# The marathon suite. Its data stage is the same 1x1 express loader sedge's is,
# under its own name: the two guests cannot share a package, because a data guest
# may not import fkapi and a control guest may not import fkdata.
$(OBS_MAR_DIR): $(DIST)/obs-mar.wasm $(DIST)/obs-mardata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-mar.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-mardata.wasm) \
	  --name bbb-marathon-test --version 0.1.0 \
	  --title "BBB marathon: the permanent-heap slope per world operation" \
	  --description "Runs hundreds of NET-ZERO world cycles -- place a balancer and remove it, lay a belt beside one and pick it up, rotate an edge, paste and undo -- and reads the guest's own heap probe after each one. Under -gc=leaking the slope of that number IS the marathon-save cost. Asserts nothing itself; test/assert-marathon.py does." \
	  --dependency "base >= 2.0.0" --dependency "better-belt-balancer" \
	  -o .

# THE ONE OBSERVER THAT MUST NOT DEPEND ON THE MOD UNDER TEST, and the dependency
# list is the whole of why.
#
# It samples the world from its own `on_configuration_changed`, which is the only
# "before" any script can reach: the migration runs from `fk_migrate` before tick
# 0, so a sample taken from a tick handler is a sample taken afterwards. Handlers
# run in MOD LOAD ORDER, `bbb-mig21-observer` sorts before `better-belt-balancer`
# by name, and a dependency on it would put this one AFTER -- at which point the
# seeding finds nothing to seed, reports zero, and test/assert-mig21.py fails on
# the zero. So the list is `base` alone, exactly as its hand-written info.json
# said, and `--dependency` REPLACES rather than adds.
#
# No data stage: it builds no rig and needs no prototype.
$(OBS_MIG21_DIR): $(DIST)/obs-mig21.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-mig21.wasm) \
	  $(OBS_COMMON) --name bbb-mig21-observer --version 0.1.0 \
	  --title "BBB 2.0-save-on-2.1 observer" \
	  --description "Reports what a Factorio 2.0 balancer save looks like when it is opened on 2.1, before and after this mod's migration runs on it. Builds nothing and asserts nothing." \
	  --dependency "base >= 2.1.0" \
	  -o .

# The quality suite. `quality` rather than `better-belt-balancer` in the second
# slot, exactly as its info.json had it: the mod under test is pulled in by the
# mod-list, and what this one cannot load without is the quality prototype tree
# every part in every rig is rolled from.
$(OBS_QUAL_DIR): $(DIST)/obs-qual.wasm $(DIST)/obs-qualdata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-qual.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-qualdata.wasm) \
	  --name bbb-qual-test --version 0.1.0 \
	  --title "BBB quality verification" \
	  --description "Builds balancers whose parts are UNCOMMON quality and drives every path where the guest asks the world for a part by name. Asserts nothing itself." \
	  --dependency "base >= 2.0.0" --dependency "quality" \
	  -o .

$(OBS_MIX_DIR): $(DIST)/obs-mix.wasm $(DIST)/obs-mixdata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-mix.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-mixdata.wasm) \
	  --name bbb-mix-test --version 0.1.0 \
	  --title "BBB mixed-item verification" \
	  --description "Runs several distinct item types through one balancer -- two pure belts, a sushi belt, and enough kinds at once to overflow the carry pool -- and reports every count per item name. Asserts nothing itself." \
	  --dependency "base >= 2.0.0" --dependency "better-belt-balancer" \
	  -o .

# THE ONLY SPACE AGE OBSERVER, and `space-age` in its dependency list is the
# reason it is the only one: without it the DLC's own prototype tree is not
# guaranteed loaded when this mod's data stage runs, and `bbbt-stackloader`'s
# `max_belt_stack_size` is refused at load outright without the `space_travel`
# feature flag the DLC brings.
$(OBS_PLAT_DIR): $(DIST)/obs-plat.wasm $(DIST)/obs-platdata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-plat.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-platdata.wasm) \
	  --name bbb-plat-test --version 0.1.0 \
	  --title "BBB space-platform verification" \
	  --description "Builds a balancer on a SPACE PLATFORM surface and reports what its outputs received. Asserts nothing itself." \
	  --dependency "base >= 2.0.0" --dependency "space-age" \
	  --dependency "better-belt-balancer" \
	  -o .

# THE OBSERVER WITH SIX OPTIONAL DEPENDENCIES AND NOT ONE REQUIRED ONE BEYOND
# base, and every one of the six is load-bearing rather than tidy.
#
# This mod is present in BOTH phases of every leg and BETTER BELT BALANCER IS
# NOT -- that is the shape of the suite. A hard dependency on it would refuse the
# load in the phase where the incumbent owns `balancer-part`, which is exactly
# the phase `fk_on_init` runs in. And an OPTIONAL dependency is not a no-op: it
# is what puts this mod AFTER whichever of them is installed in Factorio's load
# order, so `prototypes.entity["balancer-part"]` resolves to whoever really owns
# the name rather than to a race. `--dependency` REPLACES the manifest's list
# rather than adding to it, so this list is the whole of it.
$(OBS_MIG_DIR): $(DIST)/obs-mig.wasm $(DIST)/obs-migdata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-mig.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-migdata.wasm) \
	  --name bbb-mig-test --version 0.1.0 \
	  --title "BBB migration verification" \
	  --description "Builds Belt Balancer 2 shaped balancers in phase one and reports, in phase two, what survived the swap and what the adopted balancers deliver. Asserts nothing itself." \
	  --dependency "base >= 2.1.0" \
	  --dependency "(?) better-belt-balancer" \
	  --dependency "(?) belt-balancer" \
	  --dependency "(?) belt-balancer-2" \
	  --dependency "(?) belt-balancer-3" \
	  --dependency "(?) belt-balancer-performance" \
	  --dependency "(?) bbb-mig-foreign" \
	  -o .

# THE BIG THREE. Every one of them declares `better-belt-balancer` outright, and
# for `m2` that is a TIMING surface rather than an identity: its recompile
# profiler opens in the tick that mutates and closes in the tick that flushes,
# which is only a measurement of the recompile if the mod's own deferred flush
# has already run when the observer's tick handler is entered. The dependency is
# what fixes the load order, which fixes the handler order.
$(OBS_M2_DIR): $(DIST)/obs-m2.wasm $(DIST)/obs-m2data.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-m2.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-m2data.wasm) \
	  --name bbb-m2-test --version 0.1.0 \
	  --title "BBB M2 network verification" \
	  --description "Builds saturated, starved, blocked, asymmetric and cross-surface balancer rigs and reports what each output actually received. Asserts nothing itself." \
	  --dependency "base >= 2.0.0" --dependency "better-belt-balancer" \
	  -o .

$(OBS_M3_DIR): $(DIST)/obs-m3.wasm $(DIST)/obs-m3data.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-m3.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-m3data.wasm) \
	  --name bbb-m3-test --version 0.1.0 \
	  --title "BBB M3 lifecycle verification" \
	  --description "Drives every 2.0 lifecycle path that can change what the compiler compiled from -- blueprints, ghosts, robots, clones, surface deletion, rotation, silent script destruction, part death, two forces, and 600 ticks of randomised churn -- and reports what happened. Asserts nothing itself." \
	  --dependency "base >= 2.0.0" --dependency "better-belt-balancer" \
	  -o .

$(OBS_EDGE_DIR): $(DIST)/obs-edge.wasm $(DIST)/obs-edgedata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-edge.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-edgedata.wasm) \
	  --name bbb-edge-test --version 0.1.0 \
	  --title "BBB edges: every edit that lands MID-OPERATION" \
	  --description "Drives the edits that arrive while a network is full and moving: a hundred add-part/remove-part cycles on a saturated balancer, a merge and its undo across two full networks, a same-tick place-and-remove, two forces editing in one tick and then being merged, two identical scrambled pastes whose compile order must match, and an output belt, an input belt and an output-belt removal on three saturated 4x4s. Counts every item on both surfaces across every one of them, and how many of them are lying on the ground. Asserts nothing itself." \
	  --dependency "base >= 2.0.0" --dependency "better-belt-balancer" \
	  -o .

# THE RIG-STAGING MOD, WHICH IS NOT A SUITE. It is what a human enables beside
# the real mod to walk test/interactive/README.md, and `make interactive-install`
# below puts THIS package into a Factorio mods directory -- so unlike every
# recipe above it, what this builds is something a person runs by hand.
#
# `test/run.sh iact` stages it out of dist/obs like any other observer and
# --creates one save over it, which is what keeps the checklist's world from
# rotting: a rig that stopped landing, or one this mod refuses, costs a human a
# session to discover and costs a single --create to catch.
$(OBS_IACT_DIR): $(DIST)/obs-iact.wasm $(DIST)/obs-iactdata.wasm
	@mkdir -p $(OBS_DIST)
	rm -rf $@
	cd $(OBS_DIST) && $(abspath $(FKLUA)) mod $(abspath $(DIST)/obs-iact.wasm) \
	  $(OBS_COMMON) --data-module $(abspath $(DIST)/obs-iactdata.wasm) \
	  --name bbb-interactive-setup --version 0.2.0 \
	  --title "BBB interactive checklist rigs" \
	  --description "Pre-stages the five player-gesture rigs the headless suites cannot drive (the miner's pocket, the belt at the edge, the 65th belt, the over-limit bridge, fast replace both ways) and the five mod-portal demo scenes, beside spawn on a fresh world. Every rig is built to Factorio 2.1's one-belt-per-part rule. Enable together with better-belt-balancer, start a new world, follow test/interactive/README.md, then disable." \
	  --dependency "base >= 2.1.0" --dependency "better-belt-balancer" \
	  -o .

# --- housekeeping ------------------------------------------------------------

check:
	@# The network planner, the sprite-variant mapping and the carry-pool
	@# identity are pure Go with no wasm imports, so they run under a normal
	@# toolchain -- which is the point of each being its own package. The
	@# balance property is proved in ./plan by simulation before Factorio ever
	@# sees a splitter, the five named shapes in ./skin before it is asked to
	@# draw one, and ./carry pins which network a drained pool -- and the miner
	@# who may pocket it -- belongs to. That last one is the ONLY machine-checked
	@# part of the miner's pocket: the trigger needs a player, the predicate
	@# needs nothing.
	@#
	@# ./edgemode is the fourth, and it is there for a reason the other three do
	@# not have: THE ENGINE ITS INTERESTING STATES LIVE ON IS ONE THIS MACHINE
	@# CANNOT RUN. Multiple belts per balancer part exist on Factorio 2.0 only, so
	@# the setting reading true, a player flipping it and the grandfather pass
	@# writing it are all unreachable from a 2.1 headless run. Written inside
	@# `main` that fold would be four branches nothing could ever execute; here,
	@# all eighteen of its states are checked -- including the one that matters
	@# most on 2.1, which is that the write is never attempted where the key does
	@# not exist.
	@#
	@# ./tune is the sixth and its reason is ./engine's, one step out: the
	@# interesting states of a fallback ladder are states of SOMEBODY ELSE'S MOD
	@# SET. A rung falls through only in a game with no `express-transport-belt`
	@# in it, and no mod set this repository can install has that shape -- so
	@# inside guest/go/data those arms would be branches nothing could execute.
	@# It also carries the one check nothing else in this repo could make: every
	@# allowed value of both cost settings has its own [string-mod-setting]
	@# locale entry, in BOTH directions. --dump-data does not read locale and no
	@# suite opens a menu, so a missing entry loads perfectly and renders as
	@# `Unknown key: ...` in front of the player.
	@#
	@# ./engine is the fifth and it is there for the SAME reason, one stage
	@# earlier: it is the data stage's version branch -- can this Factorio put two
	@# belt-connectables on one tile -- and its `true` arm emits prototypes only a
	@# 2.0 binary can be shown. `test/check-datastage.py` hashes what the data
	@# stage EMITS and can only ever cover the flavour of the binary it runs on,
	@# so the DECISION is pinned here and the 2.0 emission's dump golden is
	@# deferred to the release/2.0 recut. It is also the only shape a test can
	@# reach at all: a package that imports fkdata cannot be built by a host
	@# toolchain, because //go:wasmimport is rejected outside GOARCH=wasm.
	cd guest/go && go test ./plan/ ./skin/ ./carry/ ./edgemode/ ./engine/ ./tune/
	@# Both guests have to COMPILE, and the data guest is the one a `go test`
	@# cannot reach. `go vet` over ./data type-checks it under the wasm build
	@# tags without asking TinyGo for a wasm module.
	cd guest/go && GOOS=wasip1 GOARCH=wasm go vet ./data/
	@# The TEST OBSERVERS, for the same reason: they are `main` packages full of
	@# //go:wasmexport that no `go test` can reach, and the harness under them is
	@# what fourteen suites will share. gofmt below already covers them, because
	@# they are inside guest/go; this is what type-checks them.
	cd guest/go && GOOS=wasip1 GOARCH=wasm go vet ./obs/...
	@# The fast-belt fixture is a WHOLE FACTORIO MOD written in Go -- built by
	@# test/check-datastage.py for the speed arm and never shipped -- and it is
	@# its OWN Go module, so neither the tests nor the vet above reach it. Same
	@# two checks, one directory over.
	cd test/fixtures/fastbelt && GOOS=wasip1 GOARCH=wasm go vet ./...
	@# --lang comes from fklua.toml's `lang = ["go"]`; passing it was a
	@# workaround for a gen-bindings that ignored the manifest.
	$(FKLUA) gen-bindings --check
	$(FKLUA) lock --check
	@out=$$(cd guest/go && gofmt -l . | grep -v '^fkapi/' || true); \
	  out="$$out $$(cd test/fixtures/fastbelt && gofmt -l . || true)"; \
	  if [ -n "$$(echo $$out)" ]; then echo "gofmt: $$out"; exit 1; fi
	@echo "check: ok"

# THE DATA STAGE'S OWN GATE, and it is not part of `make test` on purpose.
#
# `test/run.sh`'s fourteen suites are all about the RUNTIME: they --create a
# save, --benchmark it and read the guest's own log lines. Not one of them can
# see a prototype field, because by the time a suite is looking the data stage
# has been over for a whole load. This runs Factorio's `--dump-data`, which
# executes the settings and data stages and STOPS BEFORE control.lua, and hashes
# the normalised result against a golden captured from the hand-written Lua the
# data guest replaced.
#
# Separate target because it is a different question and a different cost --
# about three seconds per arm against the suites' minutes -- and because it is
# the one gate that stays meaningful when the guest under test has no runtime
# behaviour to measure at all.
datastage-check: mod
	FACTORIO_BIN="$(FACTORIO_BIN)" test/check-datastage.py

clean:
	rm -rf $(DIST) test/tmp
