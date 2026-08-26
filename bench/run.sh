#!/usr/bin/env bash
# Headless benchmark harness for belt balancers.
#
# One invocation = one matrix cell. It stages a throwaway mod directory, creates
# a save whose on_init has already built N test rigs, benchmarks that save, and
# appends a machine-readable row to bench/baselines/results.tsv.
#
#   bench/run.sh --mod bb2 --scenario saturated -n 50 -k 4 --tier express
#
# Nothing third-party is ever copied into the repo: the balancer mod is taken
# from a zip found under $BB_MODS_SRC and staged into bench/tmp/.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BENCH="$ROOT/bench"
TMP="${BENCH_TMP:-$BENCH/tmp}"

FACTORIO="${FACTORIO_BIN:-$HOME/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio}"
# Where third-party balancer mod zips live. Never inside the repo.
MODS_SRC="${BB_MODS_SRC:-/Users/$USER/Library/Application Support/factorio/mods}"

MOD=none          # none | bb2 | bb3 | bbb | <path to a mod zip>
SCENARIO=saturated # saturated | idle | control | control-idle | mega[-idle] | mega-control[-idle]
HITCH=0           # mega only: run the 64x64 recompile-hitch schedule
N=1
K=4
TIER=express
TICKS=3600
RUNS=2
METER=600
ITEM=iron-ore
PART_NAME=""      # empty -> derived from the mod (see below)
VERBOSE_TIMINGS="${BENCH_VERBOSE_TIMINGS:-}"
KEEP_SAVE=0
NOTE=""

usage() {
  sed -n '2,11p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
  cat <<'EOF'

Options:
  --mod X          none | bb2 | bb3 | bbb | /path/to/mod.zip  (default: none)
                   bbb is THIS repo: `make zip` runs first if dist/ is stale
  --scenario X     saturated | idle | control | control-idle
                   mega | mega-idle | mega-control | mega-control-idle
                                                        (default: saturated)
  --hitch          mega only: remove and restore one input belt of the 64x64
                   three times during the run and profile the recompile
  -n N             number of rigs (mega: number of BLOCKS)  (default: 1)
  -k K             balancer size (K in, KxK parts, K out) (default: 4)
  --tier X         normal | fast | express               (default: express)
  --ticks N        --benchmark-ticks                     (default: 3600)
  --runs N         --benchmark-runs                      (default: 2)
  --meter N        ticks between throughput samples, 0=off (default: 600)
  --item NAME      item to push through                  (default: iron-ore)
  --part-name NAME the balancer mod's part prototype     (default: per --mod)
  --note TEXT      free-text note recorded in the TSV
  --keep-save      do not delete the generated save
Env: FACTORIO_BIN, BB_MODS_SRC, BENCH_TMP, BENCH_VERBOSE_TIMINGS, BENCH_NO_BUILD
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --mod) MOD="$2"; shift 2 ;;
    --scenario) SCENARIO="$2"; shift 2 ;;
    -n) N="$2"; shift 2 ;;
    -k) K="$2"; shift 2 ;;
    --tier) TIER="$2"; shift 2 ;;
    --ticks) TICKS="$2"; shift 2 ;;
    --runs) RUNS="$2"; shift 2 ;;
    --meter) METER="$2"; shift 2 ;;
    --item) ITEM="$2"; shift 2 ;;
    --part-name) PART_NAME="$2"; shift 2 ;;
    --note) NOTE="$2"; shift 2 ;;
    --hitch) HITCH=1; shift ;;
    --keep-save) KEEP_SAVE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[ -x "$FACTORIO" ] || { echo "factorio not found at: $FACTORIO (set FACTORIO_BIN)" >&2; exit 1; }

# WHICH FACTORIO IS RUNNING IS AN INPUT, and it was not one until 2026-08-25.
#
# The setup mod's manifest named a series, and a Factorio that is not that
# series refuses it at the LOADER before a rig is placed:
#
#   Error Util.cpp:81: Failed to load mod "bbb-bench-setup":
#     • Incompatible Factorio version (current: 2.1, required: 2.0)
#
# So every cell in this harness stopped running the day trunk moved to 2.1, and
# nothing said so because nobody re-ran one. The staged copy is STAMPED for the
# binary now, which is test/run.sh's rule and its exact perl: factorio_version
# unconditionally, and `base >= X.Y.Z` clamped DOWN when it names a series newer
# than this engine and otherwise left alone. Both rewrites are no-ops when the
# manifest already agrees, which is what makes this safe on either arm.
#
# The BALANCER MOD is not stamped, for the reason test/run.sh states at length:
# a setup mod is engine-agnostic and the mod under test is a guest compiled
# against a pinned API whose ABI marshals event payloads by name. `--mod bbb`
# builds from this tree, so a mismatch there is a real defect; the third-party
# zips are somebody else's and are used as they are.
ENGINE_SERIES="$("$FACTORIO" --version 2>/dev/null |
  sed -n '1s/^Version: \([0-9][0-9]*\.[0-9][0-9]*\)\..*$/\1/p')"
[ -n "$ENGINE_SERIES" ] || { echo "could not read a Major.Minor series out of \`$FACTORIO --version\`" >&2
                             "$FACTORIO" --version 2>&1 | head -3 >&2; exit 1; }
ENGINE_MAJOR="${ENGINE_SERIES%%.*}"
ENGINE_MINOR="${ENGINE_SERIES##*.}"

stamp_engine() {
  local d="$1"
  [ -f "$d/info.json" ] || { echo "no info.json in $d" >&2; exit 1; }
  ENGINE_SERIES="$ENGINE_SERIES" ENGINE_MAJOR="$ENGINE_MAJOR" \
  ENGINE_MINOR="$ENGINE_MINOR" perl -pi -e '
    s/"factorio_version":\s*"[^"]*"/"factorio_version": "$ENV{ENGINE_SERIES}"/;
    s{(base\s*(?:>=|\\u003e=)\s*)(\d+)\.(\d+)\.(\d+)}{
      my ($lead, $maj, $min, $pat) = ($1, $2, $3, $4);
      ($maj > $ENV{ENGINE_MAJOR} ||
       ($maj == $ENV{ENGINE_MAJOR} && $min > $ENV{ENGINE_MINOR}))
        ? "$lead$ENV{ENGINE_MAJOR}.$ENV{ENGINE_MINOR}.0"
        : "$lead$maj.$min.$pat";
    }ge;
  ' "$d/info.json"
  grep -q "\"factorio_version\": \"$ENGINE_SERIES\"" "$d/info.json" || {
    echo "$d/info.json was not stamped for Factorio $ENGINE_SERIES" >&2
    cat "$d/info.json" >&2; exit 1; }
}

# Resolve --mod to a zip path (or empty for the no-mod control). `bbb` is this
# repo's own mod: `make zip` produces a complete installable zip in dist/ and
# the zip is a real file target, so this rebuilds only when something moved.
MOD_ZIP=""
MOD_LABEL="$MOD"
case "$MOD" in
  none) MOD_LABEL=none ;;
  bb2)  MOD_ZIP="$MODS_SRC/belt-balancer-2_2.0.9.zip" ;;
  bb3)  MOD_ZIP="$(ls "$MODS_SRC"/belt-balancer-3_*.zip 2>/dev/null | head -1)" ;;
  bbb)
    BBB_NAME="$(sed -n 's/^name = "\(.*\)"$/\1/p' "$ROOT/fklua.toml")"
    BBB_VERSION="$(sed -n 's/^version = "\(.*\)"$/\1/p' "$ROOT/fklua.toml")"
    MOD_ZIP="$ROOT/dist/${BBB_NAME}_${BBB_VERSION}.zip"
    if [ -z "${BENCH_NO_BUILD:-}" ]; then
      echo "==> make zip (bbb)"
      make -C "$ROOT" zip >/dev/null || { echo "make zip failed" >&2; exit 1; }
    fi ;;
  *)    MOD_ZIP="$MOD" ;;
esac
BAL_NAME=""
if [ -n "$MOD_ZIP" ]; then
  [ -f "$MOD_ZIP" ] || { echo "balancer mod zip not found: '$MOD_ZIP'" >&2
                         echo "put it under \$BB_MODS_SRC ($MODS_SRC)" >&2; exit 1; }
  MOD_LABEL="$(basename "$MOD_ZIP" .zip)"
  # mod name = zip basename minus the trailing _<version>
  BAL_NAME="$(sed 's/_[0-9][0-9.]*$//' <<<"$MOD_LABEL")"
fi

# Which prototype the setup mod places. Ours is namespaced; the incumbents are
# not, and they share the name because bb3 is a fork of bb2.
if [ -z "$PART_NAME" ]; then
  case "$BAL_NAME" in
    better-belt-balancer) PART_NAME="bbb-balancer-part" ;;
    *)                    PART_NAME="balancer-part" ;;
  esac
fi

case "$SCENARIO" in
  saturated|idle|mega|mega-idle)
    [ -n "$MOD_ZIP" ] || { echo "scenario '$SCENARIO' places balancer parts and needs a balancer mod" >&2; exit 2; } ;;
  control|control-idle|mega-control|mega-control-idle)
    [ -z "$MOD_ZIP" ] || { echo "scenario '$SCENARIO' places no balancer parts; use --mod none" >&2; exit 2; } ;;
  *) echo "unknown scenario: $SCENARIO" >&2; exit 2 ;;
esac
# Scenarios that must move exactly zero items, and scenarios that build the
# heterogeneous mega population. Named once here rather than string-matched
# four times below.
IS_IDLE=0; case "$SCENARIO" in idle|control-idle|mega-idle|mega-control-idle) IS_IDLE=1 ;; esac
IS_MEGA=0; case "$SCENARIO" in mega|mega-idle|mega-control|mega-control-idle) IS_MEGA=1 ;; esac
if [ "$HITCH" -eq 1 ] && [ "$IS_MEGA" -eq 0 ]; then
  echo "--hitch needs a mega scenario (the 64x64 only exists there)" >&2; exit 2
fi

CELL="${MOD_LABEL}-${SCENARIO}-n${N}-k${K}-${TIER}"
[ "$HITCH" -eq 1 ] && CELL="${CELL}-hitch"
WORK="$TMP/$CELL"
MODDIR="$WORK/mods"
SAVE="$WORK/bench.zip"
CREATE_LOG="$WORK/create.log"
RUN_LOG="$WORK/run.log"

rm -rf "$WORK"
mkdir -p "$MODDIR"

# Private write dir so concurrent Factorio instances (other agents' test runs)
# cannot fight over the shared user dir's .lock — the spike-s1 finding. The
# read-data path resolves relative to the executable inside the app bundle.
mkdir -p "$WORK/userdir"
cat > "$WORK/config.ini" <<INI
[path]
read-data=__PATH__executable__/../data
write-data=$WORK/userdir
INI
FCONF=(-c "$WORK/config.ini")

# --- stage the mod directory -------------------------------------------------
# THE SETUP MOD IS A COMPILED GO GUEST (guest/go/obs/bench), staged out of the
# package `make bench-setup` writes. There is no hand-written Lua in this
# repository, and agents/estate-port.md is the programme that got there.
#
# ONE TARGET AND NOT `observers`: a benchmark cell must not relink the other
# thirteen packages, and a matrix run is dozens of cells.
if [ -z "${BENCH_NO_BUILD:-}" ]; then
  make -C "$ROOT" bench-setup >/dev/null || { echo "make bench-setup failed" >&2; exit 1; }
fi
SETUP_PKG="$(ls -d "$ROOT"/dist/obs/bbb-bench-setup_*/ 2>/dev/null | head -1)"
[ -n "$SETUP_PKG" ] || { echo "no bbb-bench-setup package under dist/obs; run \`make bench-setup\`" >&2; exit 1; }
cp -R "$SETUP_PKG" "$MODDIR/bbb-bench-setup"
stamp_engine "$MODDIR/bbb-bench-setup"

# THE CONFIGURATION CHANNEL. This used to be a `config.lua` written into the
# staged copy of the mod and `require`d by it; a Go guest cannot require a Lua
# file, so the eight keys are STARTUP SETTINGS the mod's own settings stage
# defines and this writes into a mod-settings.dat. tools/mod-settings.py is the
# PropertyTree writer and its header carries the format; guest/go/obs/protos
# holds the key names, which are the one thing the mod's two wasm modules have
# to agree about.
#
# JSON carries the three types the tree needs without the shell declaring them,
# which is what makes an int arrive as a number and --hitch as a bool.
python3 - "$MODDIR/mod-settings.dat" "$ENGINE_SERIES.0" <<PY || { echo "could not write mod-settings.dat" >&2; exit 1; }
import json, subprocess, sys
doc = {"startup": {
  "bbb-bench-scenario":  "$SCENARIO",
  "bbb-bench-n":         $N,
  "bbb-bench-k":         $K,
  "bbb-bench-tier":      "$TIER",
  "bbb-bench-item":      "$ITEM",
  "bbb-bench-part-name": "$PART_NAME",
  "bbb-bench-meter":     $METER,
  "bbb-bench-hitch":     $([ "$HITCH" -eq 1 ] && echo True || echo False),
}}
subprocess.run([sys.executable, "$ROOT/tools/mod-settings.py",
                "--out", sys.argv[1], "--factorio-version", sys.argv[2]],
               input=json.dumps(doc), text=True, check=True)
PY

MOD_ENTRIES=""
if [ -n "$MOD_ZIP" ]; then
  cp "$MOD_ZIP" "$MODDIR/"
  MOD_ENTRIES=",
    { \"name\": \"$BAL_NAME\", \"enabled\": true }"
fi

# The Space Age DLC mods ship inside the Factorio install's data/ directory, so
# they are visible no matter what --mod-directory says. Disable them explicitly:
# a base-only baseline is cleaner, and belt-balancer-2 lists space-age as an
# optional dependency, so it does not need them.
cat > "$MODDIR/mod-list.json" <<JSON
{
  "mods": [
    { "name": "base", "enabled": true },
    { "name": "elevated-rails", "enabled": false },
    { "name": "quality", "enabled": false },
    { "name": "space-age", "enabled": false },
    { "name": "bbb-bench-setup", "enabled": true }$MOD_ENTRIES
  ]
}
JSON

# --- create the save (on_init builds the rigs) -------------------------------
echo "==> [$CELL] creating save"
if ! "$FACTORIO" "${FCONF[@]}" --mod-directory "$MODDIR" --create "$SAVE" --disable-audio \
      >"$CREATE_LOG" 2>&1; then
  echo "map creation failed; see $CREATE_LOG" >&2; tail -40 "$CREATE_LOG" >&2; exit 1
fi
grep -E "BENCH-SETUP" "$CREATE_LOG" || { echo "setup mod never reported; see $CREATE_LOG" >&2
                                         grep -iE "error|failed" "$CREATE_LOG" | head -20 >&2; exit 1; }
# ...AND THE CELL IS THE CELL THAT WAS ASKED FOR. The setup mod echoes all eight
# configuration keys, and this compares them against what was written into
# mod-settings.dat above.
#
# IT IS THE ANTI-VACUITY GUARD FOR THE WHOLE CONFIGURATION CHANNEL. A settings
# stage defines a DEFAULT for every key, so a key misspelled on either side of
# that channel is not absent at runtime: it reads back as the default, and the
# cell quietly builds a plausible rig for a configuration nobody asked for, with
# a throughput and a balance that look perfectly reasonable. This is the only
# place that can be seen.
WANT="BENCH-SETUP scenario=$SCENARIO n=$N k=$K tier=$TIER item=$ITEM part=$PART_NAME meter=$METER hitch=$([ "$HITCH" -eq 1 ] && echo true || echo false) "
grep -qF "$WANT" "$CREATE_LOG" || {
  echo "the setup mod did not build the cell that was asked for." >&2
  echo "  asked:  $WANT" >&2
  echo "  got:    $(grep -o 'BENCH-SETUP.*' "$CREATE_LOG" | head -1)" >&2
  echo "A settings key that does not reach the guest reads back as its own default," >&2
  echo "so this is what stands between a typo and a plausible wrong measurement." >&2
  exit 1; }
if grep -qiE "^ *[0-9.]+ Error|Unknown key|stack traceback" "$CREATE_LOG"; then
  echo "errors during map creation; see $CREATE_LOG" >&2; grep -inE "error" "$CREATE_LOG" | head >&2; exit 1
fi
# BetterBeltBalancer reports its own failures on a log line rather than by
# throwing, so they would otherwise pass the check above.
if grep -q "\[BBB\] error:" "$CREATE_LOG"; then
  echo "the balancer mod logged errors during map creation; see $CREATE_LOG" >&2
  grep -n "\[BBB\] error:" "$CREATE_LOG" | head >&2; exit 1
fi
# ...AND THE SETUP MOD'S OWN ERROR LEVEL, which is the thing the estate port
# took away. The Lua called `error()` when a piece failed to land, which aborts
# on_init outright -- the correct severity, because a rig that is not there makes
# every number after it a measurement of a different world. A GUEST CANNOT DO
# THAT: there are no coroutines under a wasm frame, so nothing can unwind. The
# honest equivalent is a line the runner refuses to see, and it is the same tag
# test/run.sh's guest_gate greps across the whole estate.
if grep -q "\[BBB-OBS\] error:" "$CREATE_LOG"; then
  echo "the setup mod could not build the rigs; see $CREATE_LOG" >&2
  grep -n "\[BBB-OBS\] error:" "$CREATE_LOG" | head >&2; exit 1
fi

# The mega scenario reports its own population and, when the balancer mod is
# ours, the number of hidden splitters it compiled -- the thing that makes the
# save a megabase rather than a big empty map. It also carries a deliberately
# OVER-LIMIT cluster (65 inputs, P would be 128), which the guest must refuse
# with an `alert:` and never with an `error:` -- the check above is the other
# half of that assertion.
if [ "$IS_MEGA" -eq 1 ]; then
  grep -E "BENCH-MEGA(-SHAPE)? " "$CREATE_LOG" | sed 's/^.*BENCH-MEGA/    BENCH-MEGA/'
  grep -E "\[BENCH-MEGA\] timing" "$CREATE_LOG" | sed 's/^.*\[BENCH-MEGA\]/    [BENCH-MEGA]/' || true
  if [ -n "$MOD_ZIP" ] && [ "$BAL_NAME" = better-belt-balancer ]; then
    OVER="$(grep -c "\[BBB\] alert:.*over the limit" "$CREATE_LOG" || true)"
    echo "    over-limit refusals (65-input cluster): $OVER"
    [ "$OVER" -ge 1 ] || { echo "the 65-input cluster was NOT refused; the cap did not fire" >&2; exit 1; }
  fi
fi

# --- benchmark ---------------------------------------------------------------
echo "==> [$CELL] benchmarking ${TICKS}t x ${RUNS}"
VERBOSE_ARGS=()
[ -n "$VERBOSE_TIMINGS" ] && VERBOSE_ARGS=(--benchmark-verbose "$VERBOSE_TIMINGS")
if ! "$FACTORIO" "${FCONF[@]}" --mod-directory "$MODDIR" --benchmark "$SAVE" \
      --benchmark-ticks "$TICKS" --benchmark-runs "$RUNS" \
      "${VERBOSE_ARGS[@]}" --disable-audio >"$RUN_LOG" 2>&1; then
  echo "benchmark failed; see $RUN_LOG" >&2; tail -40 "$RUN_LOG" >&2; exit 1
fi
if grep -qE "stack traceback|Error while running" "$RUN_LOG"; then
  echo "script error during benchmark; see $RUN_LOG" >&2; grep -nE "Error" "$RUN_LOG" | head >&2; exit 1
fi

# BetterBeltBalancer compiles at BUILD time and has no on_tick handler, so a
# benchmark of a finished save should not run a single line of its Lua. Every
# guest code path logs, so counting [BBB] lines in the benchmark window is a
# direct check of that: a compile, a teardown or a surface scan inside the
# measured ticks would be visible here. (Zero is also expected for bb2/bb3,
# which do not use the prefix -- this line only says something about us.)
BBB_LINES="$(grep -c "\[BBB\]" "$RUN_LOG" || true)"
[ "$BBB_LINES" -eq 0 ] || echo "    [BBB] log lines inside the benchmark: $BBB_LINES"

# --hitch deliberately mutates the world mid-run, so this cell is NOT a
# steady-state measurement and its avg_ms is not comparable with one. Each probe
# spans two ticks -- opened in the tick that mutates, closed in the tick that
# flushes -- so subtract the `idle tick pair` line from the two beside it.
if [ "$HITCH" -eq 1 ]; then
  grep "\[BENCH-MEGA\] hitch" "$RUN_LOG" | sed 's/^.*\[BENCH-MEGA\]/    [BENCH-MEGA]/'
fi

# --- sanity: did anything actually move? -------------------------------------
# A rig where no items flow times nothing. The last meter line of the first run
# carries the cumulative per-output counts.
THROUGHPUT=0
BALANCE=""
if [ "$METER" -gt 0 ]; then
  # Only the first run: each --benchmark-runs pass reloads the save and replays
  # the same ticks, so run 1 is representative and run boundaries are the
  # "Performed N updates" lines.
  LAST_METER="$(awk '/Performed .* updates/{exit} /BENCH-METER tick=[1-9]/{l=$0} END{print l}' "$RUN_LOG")"
  if [ -z "$LAST_METER" ]; then
    echo "no BENCH-METER line past tick 0: --ticks ($TICKS) must exceed --meter ($METER)" >&2
    exit 1
  fi
  THROUGHPUT="$(sed -E 's/.*cumulative=([0-9]+).*/\1/' <<<"$LAST_METER")"
  PER_OUTPUT="$(sed -E 's/.*per_output=([0-9,]+).*/\1/' <<<"$LAST_METER")"
  echo "    $LAST_METER"
  # The per-shape aggregates that go with that sample. Correctness before
  # timings: a class that stopped delivering, or a 64x64 that does not split 64
  # ways, is visible here and outranks every number below it.
  if [ "$IS_MEGA" -eq 1 ]; then
    awk '/Performed .* updates/ { exit }
         /BENCH-SHAPE tick=[1-9]/ {
           line = $0; sub(/^.*BENCH-SHAPE/, "BENCH-SHAPE", line)
           match(line, /class=[^ ]+/); last[substr(line, RSTART, RLENGTH)] = line
         }
         END { for (c in last) print "    " last[c] }' "$RUN_LOG" | sort
  fi
  if [ "$IS_IDLE" -eq 1 ]; then
    [ "$THROUGHPUT" -eq 0 ] || { echo "idle scenario moved $THROUGHPUT items; it must move none" >&2; exit 1; }
    BALANCE="n/a"
  else
    [ "$THROUGHPUT" -gt 0 ] || { echo "NOTHING MOVED: rig is broken, timings are meaningless" >&2; exit 1; }
    # max/min across the K output columns; a working balancer keeps this near 1.
    BALANCE="$(awk -F, -v OFS= '{mn=$1;mx=$1;for(i=1;i<=NF;i++){if($i<mn)mn=$i;if($i>mx)mx=$i}
                                 printf "%.3f", (mn>0? mx/mn : 999)}' <<<"$PER_OUTPUT")"
    echo "    balance max/min = $BALANCE"
    # In the mega scenarios `per_output` is the WORST-balanced rig in the save,
    # so this one number is the worst per-rig balance anywhere -- across every
    # shape class including the 16x16, the 32x32 and the 64x64. A balancer that
    # does not balance is a correctness finding and outranks every timing in the
    # row, so it fails the cell rather than being recorded and passed over.
    if [ "$IS_MEGA" -eq 1 ]; then
      awk -v b="$BALANCE" 'BEGIN{exit !(b+0 > 1.25)}' \
        && { echo "MEGA BALANCE FAILED: worst rig max/min = $BALANCE" >&2
             echo "  a balancer that does not balance outranks every timing here" >&2; exit 1; }
      awk -v b="$BALANCE" 'BEGIN{exit !(b+0 > 1.02)}' \
        && echo "    WARNING: worst per-rig balance $BALANCE is above 1.02"
    fi
  fi
fi

# --- parse timings -----------------------------------------------------------
# Factorio prints one "avg: X ms, min: Y ms, max: Z ms" per run.
read -r AVG MIN MAX < <(grep -oE "avg: [0-9.]+ ms, min: [0-9.]+ ms, max: [0-9.]+ ms" "$RUN_LOG" \
  | sed -E 's/avg: ([0-9.]+) ms, min: ([0-9.]+) ms, max: ([0-9.]+) ms/\1 \2 \3/' \
  | awk 'NF==3{s+=$1; n++; if(n==1||$2<mn)mn=$2; if(n==1||$3>mx)mx=$3}
         END{if(n>0) printf "%.4f %.4f %.4f\n", s/n, mn, mx}') || true
[ -n "${AVG:-}" ] || { echo "could not parse benchmark output; see $RUN_LOG" >&2; tail -20 "$RUN_LOG" >&2; exit 1; }

# --- per-system breakdown ----------------------------------------------------
# A second, shorter pass with --benchmark-verbose. Kept separate so the headline
# avg_ms above is measured with no instrumentation at all. Only the steady-state
# half is averaged: the first few hundred ticks are spent filling empty belts.
VPROF_TICKS="${BENCH_VPROF_TICKS:-1200}"
# `--benchmark-verbose` CRASHES FACTORIO 2.1.16, and it is the engine's defect
# rather than this harness's or any mod's: it emits the `t0` row and then takes
# a SIGSEGV inside Benchmark.cpp, with any counter list, on any save -- measured
# on a completely vanilla no-mod save as the control. The crash also leaves a
# reporter process holding the run's own `.lock`, which would then fail the NEXT
# cell of a matrix on something that has nothing to do with it.
#
# So the pass is skipped on 2.1 with the reason stated, rather than costing every
# cell a crash and a log upload. `BENCH_VPROF_FORCE=1` runs it anyway, for
# whoever wants to re-test the engine: the day it is fixed, that is how it is
# found out. What is lost while it stands is whole_us/belts_us/entity_us/
# script_us -- the per-system breakdown -- and the per-tick column
# bench/README.md sends a reader to for a real worst tick.
if [ "$ENGINE_MAJOR.$ENGINE_MINOR" = "2.1" ] && [ -z "${BENCH_VPROF_FORCE:-}" ]; then
  [ "$VPROF_TICKS" -gt 0 ] && echo "    (no per-system breakdown: --benchmark-verbose SIGSEGVs on Factorio $ENGINE_SERIES; BENCH_VPROF_FORCE=1 to try it)"
  VPROF_TICKS=0
fi
# BENCH_VPROF_EXTRA appends counters (e.g. luaGarbageIncremental) for a
# diagnostic run. Factorio emits the columns in ITS OWN canonical order, not the
# order asked for, so the parser below reads the header rather than counting
# columns — asking for a fifth counter otherwise silently relabels the fourth.
VPROF_COUNTERS="wholeUpdate,transportLinesUpdate,entityUpdate,scriptUpdate${BENCH_VPROF_EXTRA:+,$BENCH_VPROF_EXTRA}"
WHOLE_US=; BELTS_US=; ENTITY_US=; SCRIPT_US=
if [ "$VPROF_TICKS" -gt 0 ]; then
  VLOG="$WORK/verbose.log"
  echo "==> [$CELL] per-system breakdown (${VPROF_TICKS}t)"
  if "$FACTORIO" "${FCONF[@]}" --mod-directory "$MODDIR" --benchmark "$SAVE" \
        --benchmark-ticks "$VPROF_TICKS" --benchmark-runs 1 \
        --benchmark-verbose "$VPROF_COUNTERS" \
        --disable-audio >"$VLOG" 2>&1; then
    read -r WHOLE_US BELTS_US ENTITY_US SCRIPT_US GC_US < <(
      awk -F, -v half="$(( VPROF_TICKS / 2 ))" '
        /^tick,/ { for (i = 2; i <= NF; i++) if ($i != "") col[$i] = i; next }
        /^t[0-9]+,/ {
          t = substr($1,2)+0; if (t < half) next
          w += $col["wholeUpdate"]; b += $col["transportLinesUpdate"]
          e += $col["entityUpdate"]; s += $col["scriptUpdate"]
          if ("luaGarbageIncremental" in col) g += $col["luaGarbageIncremental"]
          n++
        }
        END { if (n>0) printf "%.2f %.2f %.2f %.2f %.2f\n",
                w/n/1000, b/n/1000, e/n/1000, s/n/1000, g/n/1000 }
      ' "$VLOG") || true
    echo "    whole ${WHOLE_US}us  belts ${BELTS_US}us  entities ${ENTITY_US}us  script ${SCRIPT_US}us${BENCH_VPROF_EXTRA:+  gc ${GC_US}us}"
  else
    echo "    (verbose pass failed, see $VLOG)" >&2
  fi
fi

FVER="$(grep -oE "Factorio [0-9]+\.[0-9]+\.[0-9]+" "$RUN_LOG" | head -1 | awk '{print $2}')"
DATE="$(date -u +%Y-%m-%d)"

RESULTS="$BENCH/baselines/results.tsv"
mkdir -p "$(dirname "$RESULTS")"
if [ ! -f "$RESULTS" ]; then
  printf 'scenario\tmod\tn\tk\ttier\tticks\truns\tavg_ms\tmin_ms\tmax_ms\twhole_us\tbelts_us\tentity_us\tscript_us\tthroughput\tbalance\tfactorio\tdate\tnote\n' > "$RESULTS"
fi
printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
  "$SCENARIO" "$MOD_LABEL" "$N" "$K" "$TIER" "$TICKS" "$RUNS" \
  "$AVG" "$MIN" "$MAX" \
  "${WHOLE_US:-}" "${BELTS_US:-}" "${ENTITY_US:-}" "${SCRIPT_US:-}" \
  "$THROUGHPUT" "${BALANCE:-}" "${FVER:-unknown}" "$DATE" "$NOTE" \
  >> "$RESULTS"

echo "    avg ${AVG} ms/tick  min ${MIN}  max ${MAX}  -> $RESULTS"
[ "$KEEP_SAVE" -eq 1 ] || rm -f "$SAVE"
