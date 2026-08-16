# BetterBeltBalancer build pipeline.
#
#   make guest    TinyGo -> dist/bbb.wasm (GC=leaking for the other arm)
#   make mod      fklua mod -> dist/<name>_<version>/, data stage included
#   make zip      the same, as an installable dist/<name>_<version>.zip
#   make install  the built mod into a Factorio mods directory
#   make test     headless verification, eight suites: SUITES=m2 for one of
#                 (m1 m2 m3 upg plat mar edge mix)
#   make check    the generated bindings and the lock are current
#
# `fklua mod` GENERATES the control stage (control.lua, info.json, fk_abi.lua,
# fk_api_gen.lua, fk_module.lua) and REPLACES its output directory each time, so
# nothing hand-written can live there. The DATA STAGE is hand-written Lua in
# mod-data/, and `fklua mod` merges it into the package itself now -- fklua.toml
# `[mod] data` is the default for --include, and the merge happens before the
# directory writer AND the zip writer, so both carry the same bytes.
# Never edit anything under dist/.

SHELL := /bin/bash

FKLUA        ?= ../FkLua/bin/fklua
FACTORIO_BIN ?= $(HOME)/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio
MODS_DIR     ?= $(HOME)/Library/Application Support/factorio/mods

# Single source of truth: fklua.toml, and `fklua mod` READS IT -- identity,
# dependencies and the data-stage directory all come from the manifest, so the
# package command takes no identity flags at all. The two below are needed only
# to NAME the output that command produces.
MOD_NAME    := $(shell sed -n 's/^name = "\(.*\)"$$/\1/p' fklua.toml)
MOD_VERSION := $(shell sed -n 's/^version = "\(.*\)"$$/\1/p' fklua.toml)

DIST    := dist
WASM    := $(DIST)/bbb.wasm
MOD_DIR := $(DIST)/$(MOD_NAME)_$(MOD_VERSION)

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
# `make GC=leaking` stays buildable and green, and all seven suites are run in
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

GUEST_SRC := $(shell find guest/go -name '*.go' -not -path '*/fkapi/*') guest/go/go.mod
DATA_SRC  := $(shell find mod-data -type f)

.PHONY: all guest mod zip install test check clean graphics

all: mod

# --- guest -------------------------------------------------------------------

guest: $(WASM)

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
mod: $(WASM) $(DATA_SRC) fklua.toml
	$(FKLUA) mod $(WASM) --persist=$(PERSIST) $(FKLUA_GC) -o $(DIST)
	python3 test/check-sprites.py $(MOD_DIR)
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

$(MOD_DIR).zip: $(WASM) $(DATA_SRC) fklua.toml $(PERSIST_STAMP) $(GC_STAMP)
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
interactive-install:
	@mkdir -p "$(MODS_DIR)"
	rm -rf "$(MODS_DIR)/bbb-interactive-setup"
	cp -R test/interactive/bbb-interactive-setup "$(MODS_DIR)/"
	@echo "installed bbb-interactive-setup into $(MODS_DIR)"

# --- test --------------------------------------------------------------------

test: mod
	FACTORIO_BIN="$(FACTORIO_BIN)" test/run.sh $(SUITES)

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
	cd guest/go && go test ./plan/ ./skin/ ./carry/
	@# --lang comes from fklua.toml's `lang = ["go"]`; passing it was a
	@# workaround for a gen-bindings that ignored the manifest.
	$(FKLUA) gen-bindings --check
	$(FKLUA) lock --check
	@out=$$(cd guest/go && gofmt -l . | grep -v '^fkapi/' || true); \
	  if [ -n "$$out" ]; then echo "gofmt: $$out"; exit 1; fi
	@echo "check: ok"

clean:
	rm -rf $(DIST) test/tmp
