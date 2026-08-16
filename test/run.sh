#!/usr/bin/env bash
# Headless verification. Nine suites, all real Factorio runs, not models:
#
#   M1  do balancer parts merge and split correctly?
#   M2  does the compiled hidden network actually balance?
#   M3  does it survive every way the world can change under it?
#   upg   does a MOD UPGRADE re-derive the registry and adopt what is standing?
#   plat  the SPACE AGE suite: a balancer on a SPACE PLATFORM surface, and
#         BELT STACKING -- does a recompile hand a stacked network back STACKED?
#         (belt stacking is a Space Age feature at the prototype level, so no
#         base-only suite can build a stacked belt at all). Its `smix` band is
#         STACKED SUSHI: the only rig anywhere whose hidden lines carry several
#         item KINDS and several items per POSITION at once, which is the pair
#         of conditions compile.go's kindAt needs to be reached at all
#   mar   what does one NET-ZERO world operation cost the guest heap, forever?
#         (the `-gc=leaking` marathon slope -- see CLAUDE.md, "The marathon save")
#   edge  the mid-operation edges: churn, merges, splits, forces and same-tick
#         edits, each with item conservation across it
#   mix   MORE THAN ONE KIND of item through one balancer: two pure belts, a
#         sushi belt, and 48 distinct kinds at once -- past the carry pool's
#         bound, which used to DESTROY the overflow. Every count is per item NAME
#   mig   ADOPTING A BELT BALANCER 2 SAVE. The only suite whose two phases run
#         under DIFFERENT MOD SETS: an incumbent's balancers are built in phase
#         one and the incumbent is uninstalled between the phases. Four legs --
#         swapped in one edit, removed a session later, arriving one at a time
#         through build events (which is also the only save written AFTER a
#         conversion, so its second phase is a plain reload that must do
#         nothing), and a stranger who owns the same prototype name and must
#         never be touched
#
#   make test          # builds the mod first
#   test/run.sh        # against whatever dist/ already holds
#   test/run.sh m2     # one suite only
#
# Each suite stages its own throwaway mod directory, `--create`s a save whose
# on_init has already built the patterns, `--benchmark`s it, and hands the logs
# to an assertion script.
#
# THE SAVE/RELOAD BETWEEN --create AND --benchmark IS LOAD-BEARING, not
# incidental: everything the benchmark phase sees exists only if the guest heap
# survived it, so every suite is also the --persist round-trip test.
#
# Factorio locks its user directory, so a second instance -- another agent, an
# open game -- makes a run fail on the .lock rather than on anything real. Every
# run here points write-data at a private directory via -c, which is the trick
# spike S1 arrived at.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="${BBB_TEST_TMP:-$ROOT/test/tmp}"
FACTORIO="${FACTORIO_BIN:-$HOME/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio}"

MOD_NAME="$(sed -n 's/^name = "\(.*\)"$/\1/p' "$ROOT/fklua.toml")"
MOD_VERSION="$(sed -n 's/^version = "\(.*\)"$/\1/p' "$ROOT/fklua.toml")"
MOD_DIR="$ROOT/dist/${MOD_NAME}_${MOD_VERSION}"

[ -x "$FACTORIO" ] || { echo "factorio not found at: $FACTORIO (set FACTORIO_BIN)" >&2; exit 1; }
[ -d "$MOD_DIR" ]  || { echo "no built mod at $MOD_DIR; run \`make mod\` first" >&2; exit 1; }

SUITES="${*:-m1 m2 m3 upg plat mar edge mix mig}"

# A private write-data directory, so a concurrent Factorio cannot take the lock
# out from under us.
USERDIR="$TMP/userdir"
CONFIG="$USERDIR/config/config.ini"
mkdir -p "$USERDIR/config"
cat > "$CONFIG" <<INI
[path]
read-data=__PATH__system-read-data__
write-data=$USERDIR

[general]
locale=auto
INI

# stage <workdir> <test-mod-name> [space-age]
stage() {
  local work="$1" testmod="$2" dlc="${3:-false}"
  rm -rf "$work"
  mkdir -p "$work/mods"
  cp -R "$MOD_DIR" "$work/mods/"
  cp -R "$ROOT/test/mods/$testmod" "$work/mods/$testmod"
  # The Space Age DLC mods ship inside the Factorio install's data/ directory,
  # so they load no matter what --mod-directory says. Base only wherever
  # possible: a base-only run is one fewer variable, and only the `plat` suite
  # is about things Space Age owns -- platform surfaces and belt stacking.
  cat > "$work/mods/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": $dlc },
    { "name": "quality", "enabled": $dlc },
    { "name": "space-age", "enabled": $dlc },
    { "name": "$MOD_NAME", "enabled": true },
    { "name": "$testmod", "enabled": true }
  ]
}
JSON
}

# Called between the two phases when $BETWEEN is set. The `upg` suite uses it to
# make the guest look REBUILT to the mod glue: `control.lua` compares
# `storage.fk_build` with the module's build stamp, and a mismatch means the
# saved guest heap is not adopted -- which is exactly what a mod version bump
# does, and the only way to reach the rebuild-from-world path from a test.
#
# Changing the stamp rather than recompiling with different flags is not a
# shortcut: `fk_migrate` is a NOTIFICATION on a fresh heap (FKLUA-GAPS.md item
# 13, fixed upstream), so the old heap is discarded unread either way and the
# guest starts from its own data segments. The two are the same code path to the
# byte, and this is the only place a second guest BUILD is exercised at all.
#
# ASSERT ON OUR OWN LOG LINES, NEVER ON `fklua: ` ONES. A runtime log line is
# API surface -- upstream says so, and this repo is the reason it says so: the
# `fk_migrate` redesign moved WHEN "fklua: this mod was rebuilt" fires without
# changing a character of its text, and the assertion keyed on it failed while
# the bump had worked perfectly. assert-upgrade.py greps `[BBB] the mod was
# rebuilt`, which is this guest's own statement about its own state. If a
# `fklua: ` line ever has to be matched, match the OPENING CLAUSE only; the
# detail after it is free to change.
bump_build() {
  local old="$1/mods/${MOD_NAME}_${MOD_VERSION}"
  # Factorio insists on Major.Middle.Minor, so the bump is a real one.
  local upgv="${MOD_VERSION%.*}.$(( ${MOD_VERSION##*.} + 1 ))"
  local new="$1/mods/${MOD_NAME}_${upgv}"
  local f="$new/fk_module.lua"
  mv "$old" "$new"
  # The VERSION too, so Factorio treats it as a mod upgrade and runs
  # on_configuration_changed -- which is where the glue notices the guest is a
  # different build and re-publishes the fresh heap to `storage`. Without the
  # bump, on_load alone declines to adopt and the save would keep pointing at
  # the heap nobody is using.
  perl -pi -e 's/"version": "[^"]+"/"version": "'"$upgv"'"/' "$new/info.json"
  grep -q 'build = "' "$f" || { echo "no build stamp in $f" >&2; exit 1; }
  perl -pi -e 's/build = "[^"]+"/build = "m3-upgrade-test"/' "$f"
  echo "==> mod version and guest build stamp bumped: this is an upgrade"
}

# --- the mig suite's staging -------------------------------------------------
#
# EVERY OTHER SUITE RUNS ITS TWO PHASES UNDER ONE MOD SET. This one cannot: the
# thing under test is what happens when a NEIGHBOUR is uninstalled, so the mod
# list has to differ between `--create` and `--benchmark`, and in one leg this
# mod is not present for the create at all.
#
# A mod DIRECTORY that is present but not in mod-list.json is added back by
# Factorio as enabled, so "removed" here means the directory is deleted as well
# as the entry -- which is also what a player does.

# mig_list <workdir> <extra-mod-name-or-empty> <bbb-enabled>
mig_list() {
  local work="$1" extra="$2" bbb="$3"
  local extra_entry=""
  [ -n "$extra" ] && extra_entry="    { \"name\": \"$extra\", \"enabled\": true },"
  cat > "$work/mods/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": false },
    { "name": "quality", "enabled": false },
    { "name": "space-age", "enabled": false },
$extra_entry
    { "name": "$MOD_NAME", "enabled": $bbb },
    { "name": "bbb-mig-test", "enabled": true }
  ]
}
JSON
}

# stage_mig <workdir> <extra-mod-name> <bbb-in-phase-one>
stage_mig() {
  local work="$1" extra="$2" bbb="$3"
  rm -rf "$work"
  mkdir -p "$work/mods"
  cp -R "$ROOT/test/mods/bbb-mig-test" "$work/mods/bbb-mig-test"
  [ -n "$extra" ] && cp -R "$ROOT/test/mods/$extra" "$work/mods/$extra"
  [ "$bbb" = true ] && cp -R "$MOD_DIR" "$work/mods/"
  mig_list "$work" "$extra" "$bbb"
}

# The incumbent goes, this mod arrives, in one edit of the mod list -- which is
# what a player does when they read "use this instead".
mig_swap_in() {
  rm -rf "$1/mods/belt-balancer-2"
  cp -R "$MOD_DIR" "$1/mods/"
  mig_list "$1" "" true
  echo "==> belt-balancer-2 uninstalled and $MOD_NAME installed: this is the swap"
}

# This mod was already there; the incumbent leaves a session later.
mig_drop_incumbent() {
  rm -rf "$1/mods/belt-balancer-2"
  mig_list "$1" "" true
  echo "==> belt-balancer-2 uninstalled; $MOD_NAME was already installed"
}

# The stranger STAYS. This mod arrives beside it and must leave it alone.
mig_add_bbb_beside_foreign() {
  cp -R "$MOD_DIR" "$1/mods/"
  mig_list "$1" "bbb-mig-foreign" true
  echo "==> $MOD_NAME installed beside bbb-mig-foreign, which still owns balancer-part"
}

# run <workdir> <ticks>
run() {
  local work="$1" ticks="$2"
  local save="$work/map.zip"
  echo "==> creating save (phase 1 runs in on_init)"
  if ! "$FACTORIO" -c "$CONFIG" --mod-directory "$work/mods" --create "$save" \
        --map-gen-seed 12345 --disable-audio >"$work/create.log" 2>&1; then
    echo "map creation failed; see $work/create.log" >&2
    tail -40 "$work/create.log" >&2; exit 1
  fi
  if grep -qiE "^ *[0-9.]+ Error|stack traceback|Unknown key" "$work/create.log"; then
    echo "errors during map creation; see $work/create.log" >&2
    grep -inE "error|traceback" "$work/create.log" | head -20 >&2; exit 1
  fi

  [ -n "${BETWEEN:-}" ] && "$BETWEEN" "$work"

  echo "==> benchmarking ${ticks}t"
  # One run: each --benchmark-runs pass reloads the save and replays the same
  # ticks, which would duplicate every phase in the log.
  if ! "$FACTORIO" -c "$CONFIG" --mod-directory "$work/mods" --benchmark "$save" \
        --benchmark-ticks "$ticks" --benchmark-runs 1 --disable-audio \
        >"$work/run.log" 2>&1; then
    echo "benchmark failed; see $work/run.log" >&2; tail -40 "$work/run.log" >&2; exit 1
  fi
  if grep -qE "stack traceback|Error while running" "$work/run.log"; then
    echo "script error during benchmark; see $work/run.log" >&2
    grep -nE "Error|traceback" "$work/run.log" | head >&2; exit 1
  fi
  # A compile that failed leaves a loud line and no network. It is never
  # acceptable, in any suite.
  if grep -qE "\[BBB\] error:" "$work/create.log" "$work/run.log"; then
    echo "the guest reported a compile error:" >&2
    grep -hE "\[BBB\] error:" "$work/create.log" "$work/run.log" | head -20 >&2; exit 1
  fi
  # THE COLLECTOR'S OWN ROOT-SET LINE, and this assertion is what replaced a
  # hand-rolled check in guest/go/gc.go.
  #
  # `fkgc` logs this once, the first time EffectiveBudget() has to floor the
  # budget because this guest's globals cost more than one step -- which means
  # gcBudget is no longer covering the root re-scan it is built to cover, and
  # the pause is the collector's choice rather than ours. It is not fatal to the
  # game (that is the whole point of the floor upstream added) and it IS fatal
  # to a run here, because gc.go's constant is derived from a number that has
  # just moved and nothing else would say so.
  #
  # Only this line. The other `fkgc:` lines -- an outrun, a refused grow -- are
  # conditions about the allocation rate that a stress suite may legitimately
  # provoke, and the `mar` suite asserts `deadlines=0` over them directly.
  if grep -qE "fkgc: this guest's ROOT SET is larger" "$work/create.log" "$work/run.log"; then
    echo "the collector had to raise this guest's step budget to cover its own" >&2
    echo "globals re-scan: gcRootGranules in guest/go/gc.go is now too small." >&2
    echo "Compare 'roots=' against 'budget='/'eff=' on the [BBB] heap line." >&2
    exit 1
  fi
}

for suite in $SUITES; do
  case "$suite" in
    m1)
      echo "=== M1: cluster merge/split ==="
      stage "$TMP/m1" bbb-m1-test
      run "$TMP/m1" "${BBB_TEST_TICKS:-100}"
      echo "==> asserting cluster transitions"
      python3 "$ROOT/test/assert-log.py" "$TMP/m1/create.log" "$TMP/m1/run.log"
      ;;
    m2)
      echo "=== M2: compiled network balance ==="
      stage "$TMP/m2" bbb-m2-test
      run "$TMP/m2" "${BBB_M2_TICKS:-3600}"
      echo "==> asserting network behaviour"
      python3 "$ROOT/test/assert-m2.py" "$TMP/m2/create.log" "$TMP/m2/run.log"
      ;;
    m3)
      echo "=== M3: lifecycle hardening ==="
      stage "$TMP/m3" bbb-m3-test
      run "$TMP/m3" "${BBB_M3_TICKS:-1560}"
      echo "==> asserting lifecycle behaviour"
      python3 "$ROOT/test/assert-m3.py" "$TMP/m3/create.log" "$TMP/m3/run.log"
      ;;
    upg)
      # The M2 rigs, created by one guest and loaded by another. The saved heap
      # is discarded, so the first event of the benchmark phase re-derives the
      # whole registry from the world -- and ADOPTS the networks already
      # standing rather than rebuilding them. Whether it adopted correctly is
      # then decided by running M2's own assertions over the result: if a single
      # network were mis-adopted, its rig would not balance.
      echo "=== M3: a mod upgrade -- the guest heap is discarded mid-save ==="
      stage "$TMP/upg" bbb-m2-test
      BETWEEN=bump_build run "$TMP/upg" "${BBB_M2_TICKS:-3600}"
      unset BETWEEN
      echo "==> asserting the rebuild, then M2's own numbers over it"
      python3 "$ROOT/test/assert-upgrade.py" "$TMP/upg/create.log" "$TMP/upg/run.log"
      python3 "$ROOT/test/assert-m2.py" "$TMP/upg/create.log" "$TMP/upg/run.log"
      ;;
    plat)
      echo "=== Space Age: a space platform surface, and belt stacking ==="
      stage "$TMP/plat" bbb-plat-test true
      run "$TMP/plat" "${BBB_PLAT_TICKS:-1560}"
      echo "==> asserting the platform rig and the stacking leg"
      python3 "$ROOT/test/assert-plat.py" "$TMP/plat/create.log" "$TMP/plat/run.log"
      ;;
    mar)
      # The `-gc=leaking` marathon question, measured rather than argued: what
      # does ONE net-zero world cycle cost the guest heap permanently, and is
      # that slope flat. The instrument is the guest's own `[BBB] heap` probe,
      # read after each of ~500 cycles.
      echo "=== marathon: the permanent-heap slope per world operation ==="
      stage "$TMP/mar" bbb-marathon-test
      run "$TMP/mar" "${BBB_MAR_TICKS:-4600}"
      echo "==> fitting the slopes"
      python3 "$ROOT/test/assert-marathon.py" "$TMP/mar/create.log" "$TMP/mar/run.log"
      ;;
    edge)
      # Correctness under churn: every edit that lands while a network is
      # MID-OPERATION, with items in flight. Conservation across each one is the
      # assertion, because a teardown that drops items is the failure this
      # architecture can have and the incumbent cannot -- and, since 2026-08-02,
      # WHERE they end up: a recompile must put them back inside the network it
      # just built and only a real removal may spill them. Every count carries a
      # ground-item total and the recompile tags must read zero.
      echo "=== edge: mid-operation churn, merges, splits and forces ==="
      stage "$TMP/edge" bbb-edge-test
      run "$TMP/edge" "${BBB_EDGE_TICKS:-5850}"
      echo "==> asserting conservation and the edge cases"
      python3 "$ROOT/test/assert-edge.py" "$TMP/edge/create.log" "$TMP/edge/run.log"
      ;;
    mix)
      # MORE THAN ONE KIND. Every other suite runs iron plates through
      # everything, which is the right default for a throughput number and left
      # the multi-kind half of guest/go/carry.go unexercised: the pool's
      # (name, quality, stack size) key, the per-kind split, insertRemainder's
      # walk over several groups, and the BOUND on how many groups one pool can
      # carry -- which until 2026-08-04 dropped what it could not hold, after
      # the drain had already read it off a line about to be destroyed.
      # Every count in this suite is per item NAME, on both surfaces, inside one
      # tick -- because a teardown that dropped one kind and reinserted the rest
      # conserves nothing and a single total can hide it.
      echo "=== mix: several item kinds through one balancer ==="
      stage "$TMP/mix" bbb-mix-test
      run "$TMP/mix" "${BBB_MIX_TICKS:-3200}"
      echo "==> asserting per-kind conservation and the mixed-load rates"
      python3 "$ROOT/test/assert-mix.py" "$TMP/mix/create.log" "$TMP/mix/run.log"
      ;;
    mig)
      # THE ONLY SUITE WHOSE TWO PHASES RUN UNDER DIFFERENT MOD SETS. Four legs,
      # and the last is the one with the blast radius.
      echo "=== mig: adopting a Belt Balancer 2 save ==="

      echo "--- leg 1: the incumbent swapped out and this mod in, in one edit ---"
      stage_mig "$TMP/mig1" belt-balancer-2 false
      BETWEEN=mig_swap_in run "$TMP/mig1" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --leg added \
        "$TMP/mig1/create.log" "$TMP/mig1/run.log"

      echo "--- leg 2: this mod installed first, the incumbent removed later ---"
      stage_mig "$TMP/mig2" belt-balancer-2 true
      BETWEEN=mig_drop_incumbent run "$TMP/mig2" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --leg later \
        "$TMP/mig2/create.log" "$TMP/mig2/run.log"

      # No incumbent has ever been installed here, so `balancer-part` is this
      # mod's own stub from the first byte and the observer's parts arrive one at
      # a time through BUILD EVENTS -- which is the path an old blueprint's
      # ghosts take. It is also the only leg whose save is written AFTER the
      # conversion and whose second phase changes no mod at all, so it is the
      # only one that can show the once-per-save flag surviving a save and
      # costing nothing on the way back.
      echo "--- leg 3: legacy parts arriving through build events, then a plain reload ---"
      stage_mig "$TMP/mig3" "" true
      run "$TMP/mig3" "${BBB_MIG_TICKS:-3600}"
      python3 "$ROOT/test/assert-mig.py" --leg built \
        "$TMP/mig3/create.log" "$TMP/mig3/run.log"

      echo "--- leg 4: a stranger owns balancer-part and must be left alone ---"
      stage_mig "$TMP/mig4" bbb-mig-foreign false
      BETWEEN=mig_add_bbb_beside_foreign run "$TMP/mig4" "${BBB_MIG_TICKS:-3600}"
      unset BETWEEN
      python3 "$ROOT/test/assert-mig.py" --leg foreign \
        "$TMP/mig4/create.log" "$TMP/mig4/run.log"
      ;;
    *)
      echo "unknown suite: $suite (expected m1, m2, m3, upg, plat, mar, edge, mix or mig)" >&2
      exit 1
      ;;
  esac
done
