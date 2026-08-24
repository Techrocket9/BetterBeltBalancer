#!/usr/bin/env python3
"""Assert what a balancer does with MORE THAN ONE KIND of item.

Every other suite ran iron plates through everything, so the multi-kind half of
`guest/go/carry.go` -- the pool's (name, quality, stack size) key, the per-kind
split, insertRemainder's walk over several groups -- had never been exercised at
all. This one runs pure belts, then sushi belts, through balancers and counts
every item in the world BY NAME either side of a forced recompile.

WHAT THIS SUITE CANNOT REACH, because it is base only: `compile.go`'s
`detailedTally` and `kindAt`. Below the stacking gate the drain takes the flat
totals and that code is never called at all, however many kinds are in flight, so
multi-kind AND STACKED is a Space Age question and lives in the `plat` suite's
`smix` band. See CLAUDE.md, "Stacked sushi".

Per name and not as one total, which is the whole point: a teardown that dropped
one kind and reinserted the rest conserves nothing, and a single total would have
to lose the same number of items twice in opposite directions to hide it.

EVERY RIG IS BUILT TO FACTORIO 2.1'S ONE-BELT-PER-PART RULE: two columns of
parts per row, plus one EDGELESS part below each west column, because under
that rule a working balancer has no free face and the belt each conservation
check lays would otherwise be REFUSED rather than compiled. N, M and the kinds
in flight are properties of the BELTS, which did not move, so every contract
below is the one it was.

    python3 test/assert-mix.py create.log run.log
"""

import re
import sys

COUNT = re.compile(
    r"\[BBB-MIX\] count tag=(\S+) total=(\d+) ground=(\d+) kinds=(\d+) "
    r"hidden=(\d+) hkinds=(\d+)"
)
KIND = re.compile(r"\[BBB-MIX\] kind tag=(\S+) name=(\S+) count=(\d+)")
SAMPLE = re.compile(r"\[BBB-MIX\] t=(\d+) rig=(\w+) out=\[([-\d ]*)\]")
OUTKINDS = re.compile(r"\[BBB-MIX\] t=(\d+) rig=(\w+) out(\d+) kinds=(\S*)")
RETURNED = re.compile(r"\[BBB\] torn down cluster \d+, returned (\d+) items")

# The guest's own statement that it met more item kinds than a carry pool holds.
#
# `alert:`, NOT `error:`, AND THAT IS WHY THIS SUITE CAN ASSERT IT AT ALL:
# test/run.sh fails any run in which a `[BBB] error:` line appears anywhere, so
# the level this line is written at is the difference between an assertable
# condition and a run that dies before a single number is read. See
# guest/go/carry.go's closePool for the reasoning at the call site.
OVERFLOW = re.compile(
    r"\[BBB\] alert: cluster (\d+) carried more than (\d+) item kinds; "
    r"(\d+) items of the kinds past that bound went to the ground beside it"
)
LISTS = re.compile(r"\[BBB-MIX\] item lists ok: many distinct=(\d+) mixfull=(\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
    r"(?: refused=(\d+))?"
)
SEDGE_REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt")

# WHAT THE RIGS BUILD, written down here rather than read off the guest's own
# report. Four clusters -- `duo`, `quad`, `mixfull` and `many` -- laid to
# Factorio 2.1's one-belt-per-part rule: two columns of parts per row, plus one
# EDGELESS part below each west column for the conservation check's belt to
# land on. duo 2 rows -> 5 parts, quad 4 -> 9, mixfull 2 -> 5, many 4 -> 9.
# `ctrl` and `probe` are bare belts and are not clusters at all.
#
# It is a statement about the SAVE and not about the compiler, which is the
# whole reason it is a constant: a rig that quietly lost a row, or that was
# rebuilt one column wide in the old multi-edge idiom, moves this number and
# nothing the guest reports about itself could say so.
WANT_CLUSTERS = 4
WANT_PARTS = 28

# How many distinct item names the `many` rig's four sushi sources cover between
# them. Asserted rather than trusted: a rig that lost half its list to a rename
# would stop overflowing and would then pass every conservation check vacuously.
MANY_KINDS = 48

T0, T1 = 1400, 3140

# The PURE-FEED rigs: which item each of their input belts carries, in row
# order. `duo` is a 2->2 and `quad` a 4->4 with the two kinds alternating.
PURE = {
    "duo":  ("iron-plate", "copper-plate"),
    "quad": ("iron-plate", "copper-plate", "iron-plate", "copper-plate"),
}

# WHAT IS ASSERTED ABOUT THE TYPE SPLIT, AND WHAT IS ONLY RECORDED.
#
# THE BUTTERFLY BALANCES COUNTS, NOT KINDS. Nothing in plan.Build knows what an
# item is; a splitter divides its input side between its output side by
# position, and an item's NAME never enters the arithmetic. So the mix at any
# one output is whatever the geometry happens to produce, it is NOT a designed
# guarantee, and no future planner change is obliged to preserve it.
#
# MEASURED, 2026-08-24, on Factorio 2.1.14 with the rigs laid single-edge:
# UNDER SYMMETRIC SATURATION THIS BUTTERFLY IS A PERMUTATION. `duo` delivers
# 1306 and 1306 -- exactly balanced by count, 0.00% spread -- with out1 taking
# ALL the copper and out2 ALL the iron; `quad` delivers 1306 per output at
# 0.00% spread with outputs 1 and 2 taking all the copper and 3 and 4 all the
# iron, from inputs that alternate iron/copper/iron/copper. Both sizes, every
# input saturated, every output draining freely: each output takes exactly its
# share by count and exactly one kind.
#
# THAT IS NOT WHAT THIS SUITE USED TO RECORD, and the difference is a port
# ORDER rather than a regression. Until 2026-08-24 `duo`'s window opened AFTER
# its conservation belt had already taken it from 2->2 to 3->2 over P=4 -- an
# asymmetric network with a dead-ended spare port and a loopback, where the
# flows genuinely have to cross -- and the multi-edge geometry put that belt
# FIRST in the edge list. That network mixed, 75/25. Laid single-edge the same
# belt enters LAST, the same P=4 network delivers 100/0, and both are exactly
# balanced by count. So the old 15% per-output floor was never a statement
# about the balancer: it was a statement about one asymmetric network's port
# assignment, and it is retired rather than re-tuned. The schedule now edits
# `duo` after its window, so what the rig measures is the 2->2 its own
# description names.
#
# WHAT REPLACES IT IS THE CHECK THE FLOOR WAS GROPING AT. "The two kinds are
# taking different paths through the network" is only a defect if it costs a
# kind its THROUGHPUT -- one kind backing up while the other flows. So every
# kind fed into a pure rig must come out at the rate it went in: iron on one
# belt in, one belt of iron out, summed over every output. That is true of a
# permutation, false of a network that starves a kind, and says nothing about
# a mix nothing ever promised.
KIND_TOL = 0.02


def read(paths):
    counts, kinds, samples, outkinds, lines = {}, {}, {}, {}, []
    for path in paths:
        with open(path, errors="replace") as f:
            for raw in f:
                lines.append(raw)
                m = COUNT.search(raw)
                if m:
                    counts[m.group(1)] = tuple(int(v) for v in m.groups()[1:])
                    kinds.setdefault(m.group(1), {})
                    continue
                m = KIND.search(raw)
                if m:
                    kinds.setdefault(m.group(1), {})[m.group(2)] = int(m.group(3))
                    continue
                m = SAMPLE.search(raw)
                if m:
                    samples.setdefault(m.group(2), {})[int(m.group(1))] = [
                        int(v) for v in m.group(3).split()
                    ]
                    continue
                m = OUTKINDS.search(raw)
                if m:
                    per = {}
                    for pair in m.group(4).split(","):
                        if ":" in pair:
                            n, c = pair.rsplit(":", 1)
                            per[n] = int(c)
                    outkinds.setdefault((m.group(2), int(m.group(3))), {})[
                        int(m.group(1))
                    ] = per
    return counts, kinds, samples, outkinds, lines


def conservation(rig, counts, kinds, fail, ground_must_grow=False):
    """Compare the before/after samples of one rig's forced recompile, per name."""
    b, a = rig + "-before", rig + "-after"
    if b not in counts or a not in counts:
        fail.append("%s: the conservation check never ran" % rig)
        return None
    (bt, bg, bk, bh, bhk) = counts[b]
    (at, ag, ak, ah, ahk) = counts[a]
    print("%-8s before: total=%-6d ground=%-4d kinds=%-3d hidden=%-5d hkinds=%d"
          % (rig, bt, bg, bk, bh, bhk))
    print("%-8s after:  total=%-6d ground=%-4d kinds=%-3d hidden=%-5d hkinds=%d"
          % ("", at, ag, ak, ah, ahk))

    lost = []
    for name in sorted(set(kinds[b]) | set(kinds[a])):
        before, after = kinds[b].get(name, 0), kinds[a].get(name, 0)
        if before != after:
            lost.append((name, before, after))
    if lost:
        for name, before, after in lost[:12]:
            print("    %-28s %6d -> %6d  (%+d)" % (name, before, after, after - before))
        fail.append(
            "%s: %d item KINDS did not survive a recompile (%d items net). A "
            "teardown reads every transport line and then destroys the entity, "
            "so a kind the pool declines to carry is a kind that ceases to "
            "exist unless something puts it on the ground."
            % (rig, len(lost), at - bt)
        )
    if ground_must_grow:
        # THE ONE WINDOW IN THIS SUITE WHERE THE GROUND IS ALLOWED TO GROW, and
        # it is required to. Past maxCarryKinds groups the pool cannot carry a
        # kind, so it goes to the world instead of ceasing to exist -- and a run
        # where nothing reached the ground is a run where the rig never
        # overflowed, which would pass every other check here while proving
        # nothing at all.
        #
        # It is a FLOOR and not an equality because `spill_item_stack` allows
        # belts: a spilled item that lands on one of the rig's own belts is
        # conserved and counted, but it is not on the ground. The `edge` suite
        # measures the same split on its dissolve leg (118 spilled, 90 of them
        # on the ground).
        if ag <= bg:
            fail.append(
                "%s: nothing reached the ground (%d -> %d). This rig exists to "
                "overflow the carry pool, and an overflow that spills nothing "
                "spilled nothing because there was nothing to spill -- the rig "
                "is vacuous." % (rig, bg, ag))
    elif ag != bg:
        fail.append(
            "%s: %d items reached the GROUND across a recompile (%d -> %d). A "
            "recompile is not a removal: the network that goes up in the same "
            "flush has room and must take them back inside it."
            % (rig, ag - bg, bg, ag))
    return (bt, bg, bk, bh, bhk), (at, ag, ak, ah, ahk)


def main():
    counts, kinds, samples, outkinds, lines = read(sys.argv[1:])
    fail = []

    # --- the multi-filter probe ---------------------------------------------
    # Not an assertion about the mod: it is the measurement that justifies the
    # rotating source the two sushi rigs use. One infinity chest with six filters
    # feeding one loader does NOT make a mixed belt, and this is where that is
    # observed rather than assumed.
    probe = outkinds.get(("probe", 1), {}).get(T1)
    if probe:
        tot = sum(probe.values()) or 1
        top = max(probe.items(), key=lambda kv: (kv[1], kv[0]))
        print("the multi-filter probe (6 filters, one chest, one loader) "
              "delivered %d items in %d of 6 kinds; %s alone was %.1f%%\n"
              % (tot, len(probe), top[0], 100.0 * top[1] / tot))
    else:
        fail.append("the multi-filter probe never reported")

    # --- throughput, against the control belt --------------------------------
    if "ctrl" not in samples or T0 not in samples["ctrl"] or T1 not in samples["ctrl"]:
        print("the control belt never reported -- did the test mod run at all?")
        return 1
    belt = samples["ctrl"][T1][0] - samples["ctrl"][T0][0]
    if belt <= 0:
        print("the control belt delivered %d items -- nothing is moving" % belt)
        return 1
    print("one saturated express belt over t=%d..%d: %d items" % (T0, T1, belt))

    for rig, feeds in sorted(PURE.items()):
        n = len(feeds)
        if rig not in samples or T0 not in samples[rig] or T1 not in samples[rig]:
            fail.append("%s: never reported" % rig)
            continue
        a, b = samples[rig][T0], samples[rig][T1]
        deltas = [b[i] - a[i] for i in range(len(b))]
        total = sum(deltas)
        ratio = total / float(belt)
        mean = total / float(len(deltas))
        spread = (max(deltas) - min(deltas)) / mean if mean else 1.0
        print("%s: %d PURE belts (%s) into a %d->%d, per-output %s, %d items, "
              "%.3fx one belt, spread %.2f%%"
              % (rig, n, ", ".join(sorted(set(feeds))), n, n, deltas, total,
                 ratio, spread * 100))
        if abs(ratio - n) > 0.02 * n:
            fail.append(
                "%s: delivered %.3f belts, expected %d -- a balancer carrying "
                "two kinds must not throttle" % (rig, ratio, n))
        if spread > 0.01:
            fail.append("%s: outputs spread %.2f%% (%s), over the 1%% bound"
                        % (rig, spread * 100, deltas))

        # EVERY KIND OUT AT THE RATE IT WENT IN, summed over the outputs. See
        # KIND_TOL above: the per-output MIX is recorded and not asserted,
        # because this butterfly is a permutation under symmetric saturation
        # and nothing in the mod ever promised otherwise. What a defect looks
        # like is a kind that does not come out at all, or comes out slower
        # than it was fed, and that is what this sees.
        got = {}
        for i in range(1, n + 1):
            per0 = outkinds.get((rig, i), {}).get(T0)
            per1 = outkinds.get((rig, i), {}).get(T1)
            if per0 is None or per1 is None:
                fail.append("%s: output %d never reported its per-kind split"
                            % (rig, i))
                continue
            win = {k: per1.get(k, 0) - per0.get(k, 0)
                   for k in set(per0) | set(per1)}
            print("     out%d over the window: %s"
                  % (i, " ".join("%s:%d" % (k, win[k]) for k in sorted(win))))
            for k, v in win.items():
                got[k] = got.get(k, 0) + v
        for name in sorted(set(feeds)):
            fed = feeds.count(name)
            out = got.get(name, 0)
            print("     %-14s %d belt(s) in, %.3f belt(s) out (%d items)"
                  % (name, fed, out / float(belt), out))
            if abs(out / float(belt) - fed) > KIND_TOL * fed:
                fail.append(
                    "%s: %d belt(s) of %s went in and %.3f came out. A kind "
                    "that does not leave at the rate it arrived is a kind the "
                    "network is holding back, whichever output it leaves by"
                    % (rig, fed, name, out / float(belt)))

    # --- conservation, per item name, across a forced recompile --------------
    print("\nitem conservation across a forced recompile, counted BY NAME on "
          "both surfaces inside one tick:")
    conservation("duo", counts, kinds, fail)
    res = conservation("mixfull", counts, kinds, fail)

    # ANTI-VACUITY, AND IT IS A STATEMENT ABOUT THE WHOLE SAVE RATHER THAN ABOUT
    # ONE RIG, which is worth saying plainly because the number looks per-rig and
    # is not. `hkinds` is the distinct kinds standing on the HIDDEN surface, and
    # that surface carries every compiled network at once -- so it cannot say
    # which of them a kind was inside. What it can say is that the rotating
    # sushi source is really producing mixed belts: if it had degenerated to one
    # kind (which is exactly what the `probe` band shows the OBVIOUS source
    # doing), every conservation check in this suite would pass while proving
    # nothing at all. `many`'s 48-name list check and the overflow alert are the
    # sharp version of this; the floor here is the cheap one that covers the two
    # 2->2 rigs as well.
    if res:
        before, _ = res
        if before[4] < 6:
            fail.append(
                "only %d distinct kinds were standing in the compiled networks "
                "when mixfull was torn down. The sushi sources are not producing "
                "mixed belts, so every conservation check here is vacuous."
                % before[4])

    # --- the overflow rig -----------------------------------------------------
    #
    # 48 distinct kinds through one saturated, dead-ended 4x4, whose hidden
    # network holds ~230 items -- so nearly every one of those kinds is standing
    # on a line when the recompile drains it, and the pool's 32-group bound is
    # passed by a wide margin.
    #
    # THIS IS THE RIG THAT FAILS ON THE GUEST THAT SHIPPED. Until 2026-08-04
    # tally() logged the 33rd group and returned, and drain() had already read it
    # off a transport line that sweep() destroys a moment later -- so the items
    # were gone. The per-name check below is what sees it; a single total would
    # see it too, but not which kinds, and "which" is what says it was the bound
    # rather than the arithmetic.
    print()
    lists = None
    for line in lines:
        m = LISTS.search(line)
        if m:
            lists = (int(m.group(1)), int(m.group(2)))
    if lists is None:
        fail.append("the test mod never reported its item lists")
    elif lists[0] != MANY_KINDS:
        fail.append(
            "many: the rig covers %d distinct item names, not %d -- a shorter "
            "list may not overflow the pool at all, and this rig proves nothing "
            "unless it does" % (lists[0], MANY_KINDS))
    else:
        print("the `many` rig's four sushi sources cover %d distinct item names"
              % lists[0])

    conservation("many", counts, kinds, fail, ground_must_grow=True)

    # ANTI-VACUITY, and it is the guest's own account rather than a count of
    # ours: the alert fires exactly when tally() met a group past the bound, so
    # its presence is the only direct evidence that the overflow path RAN. A rig
    # that quietly stayed inside 32 groups would conserve everything, spill
    # nothing, and say nothing about the thing this suite was written for.
    over = [
        (int(m.group(1)), int(m.group(2)), int(m.group(3)))
        for m in (OVERFLOW.search(l) for l in lines)
        if m
    ]
    if not over:
        fail.append(
            "no [BBB] alert: ... more than N item kinds line anywhere in the "
            "run. Either the pool's bound was never reached -- in which case "
            "`many` is vacuous and its numbers mean nothing -- or the overflow "
            "no longer says so, in which case a player has no way to know why "
            "there are items on the floor.")
    else:
        spilled = sum(o[2] for o in over)
        print("the guest reported %d overflowing teardown(s), %d items past the "
              "%d-group bound put on the world" % (len(over), spilled, over[0][1]))

    drained = [int(m.group(1)) for m in (RETURNED.search(l) for l in lines) if m]
    if not drained or max(drained) == 0:
        fail.append("no teardown drained anything, so nothing here proves anything")
    else:
        print("\nthe largest single teardown handed back %d items" % max(drained))

    # --- the final audit, on a world nothing has touched since tick 1200 ------
    #
    # THE CLUSTER AND PART COUNTS ARE THE POINT AND `unbuilt=0` WOULD NOT BE. A
    # cluster with no inputs or no outputs is a legitimate half-built state and
    # never counts as unbuilt, so a rig that came out one column wide -- the old
    # multi-edge idiom -- would be REFUSED, deliver nothing, and still read
    # `unbuilt=0`. The tuple is what sees it, and `nets == clusters` is what
    # says every cluster's edges were recognised.
    audits = [tuple(0 if g is None else int(g) for g in m.groups())
              for m in (AUDIT.search(l) for l in lines) if m]
    if not audits:
        fail.append("no audit was ever logged, so nothing checked the registry "
                    "against the world")
    else:
        clusters, parts, nets, drift, unbuilt, refused = audits[-1]
        print("\nfinal audit: %d clusters, %d parts, %d networks, drift=%d, "
              "unbuilt=%d, refused=%d"
              % (clusters, parts, nets, drift, unbuilt, refused))
        if (clusters, parts) != (WANT_CLUSTERS, WANT_PARTS):
            fail.append("the save holds %d clusters over %d parts and the rigs "
                        "build %d over %d -- a rig is not the shape this suite "
                        "thinks it is"
                        % (clusters, parts, WANT_CLUSTERS, WANT_PARTS))
        if nets != clusters:
            fail.append("%d of %d clusters have no network at all: something "
                        "adjacent to a balancer is not being classified as an "
                        "edge" % (clusters - nets, clusters))
        if drift or unbuilt or refused:
            fail.append("the final audit found drift=%d unbuilt=%d refused=%d "
                        "over %d clusters, on a world nothing has touched since "
                        "tick 1200" % (drift, unbuilt, refused, clusters))

    # Every rig in this save is built to the one-belt-per-part rule, so a
    # refusal is a statement about the SAVE and not about the guest. Asserted
    # separately from the audit's `refused=` column because a refusal can be
    # issued, delivered and then withdrawn between two audits -- and because it
    # names the cluster, which the column does not.
    sedge = [m for m in (SEDGE_REFUSED.search(l) for l in lines) if m]
    if sedge:
        fail.append("%d single-edge refusal(s) were issued in a save whose rigs "
                    "are all built to the rule; the first was cluster %s with "
                    "%s part(s) carrying more than one belt"
                    % (len(sedge), sedge[0].group(1), sedge[0].group(2)))

    if fail:
        print("\nMIX ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        return 1
    print("\nmix assertions passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
