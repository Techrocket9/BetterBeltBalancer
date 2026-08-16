#!/usr/bin/env bash
# Re-runs the whole baseline matrix into bench/baselines/results.tsv.
#
#   bench/matrix.sh                       # the incumbent baseline
#   MODS="bb2 bb3 bbb" bench/matrix.sh    # the M4 head-to-head
#   MODS="bb2" TIERS="express" bench/matrix.sh
#
# Every cell is 3600 ticks x 2 runs. On an M-series Mac the full matrix is a few
# minutes; the n=200 express cells dominate. The three-mod head-to-head is ~40.
#
# EVERY MOD FOR A GEOMETRY IS RUN BEFORE THE NEXT GEOMETRY, and the control with
# them, because that is the only thing that makes the columns comparable:
# absolute timings on this machine drift 25-35% between sessions with background
# load, and a slow-moving drift scales a whole cell-group together.
#
#   BENCH_TMP=/private/tmp/bbb-bench MODS="bb2 bb3 bbb" bench/matrix.sh
#
# is worth typing for a long matrix -- the saves are hundreds of MB and having
# Spotlight index them inside the repo was itself worth 25%.
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TIERS="${TIERS:-express normal}"
MODS="${MODS:-bb2}"
TICKS="${TICKS:-3600}"
RUNS="${RUNS:-2}"

# MEGA=1 runs the megabase matrix INSTEAD of the uniform one: one heterogeneous
# save per arm -- ten shape classes per block, plus a 16x16, a 32x32, a 64x64
# and a deliberately over-limit 65-input cluster -- at ~4.4k hidden splitters.
#
#   BENCH_TMP=/private/tmp/bbb-bench MEGA=1 REPS=3 bench/matrix.sh
#
# Same interleaving rule as the uniform matrix below and for the same reason:
# every arm of a geometry runs before the next geometry, so a slow-moving
# session drift scales the whole group together instead of biasing one column.
# The 64x64 recompile hitch is NOT here -- it mutates the world mid-run, so it
# is not a steady-state cell:  bench/run.sh --mod bbb --scenario mega --hitch
if [ -n "${MEGA:-}" ]; then
  MEGA_N="${MEGA_N:-40}"
  REPS="${REPS:-3}"
  for rep in $(seq 1 "$REPS"); do
    for sc in mega mega-idle; do
      ctrl=mega-control; [ "$sc" = mega-idle ] && ctrl=mega-control-idle
      # --keep-save: the save SIZE is one of the things a megabase cell is for,
      # and it is ~2 MB, not the hundreds of MB the uniform n=200 cells make.
      "$HERE/run.sh" --mod bbb --scenario "$sc" -n "$MEGA_N" --tier express \
        --ticks "$TICKS" --runs "$RUNS" --keep-save --note "mega r$rep"
      "$HERE/run.sh" --mod none --scenario "$ctrl" -n "$MEGA_N" --tier express \
        --ticks "$TICKS" --runs "$RUNS" --keep-save --note "mega r$rep"
    done
  done
  exit 0
fi

# n k scenario
CELLS="1:4:saturated 50:4:saturated 200:4:saturated 50:8:saturated 200:4:idle"

for tier in $TIERS; do
  for cell in $CELLS; do
    IFS=: read -r n k sc <<<"$cell"
    for mod in $MODS; do
      "$HERE/run.sh" --mod "$mod" --scenario "$sc" -n "$n" -k "$k" --tier "$tier" \
        --ticks "$TICKS" --runs "$RUNS"
    done
    # matching no-balancer control for the same geometry
    ctrl=control; [ "$sc" = idle ] && ctrl=control-idle
    "$HERE/run.sh" --mod none --scenario "$ctrl" -n "$n" -k "$k" --tier "$tier" \
      --ticks "$TICKS" --runs "$RUNS"
  done
done
