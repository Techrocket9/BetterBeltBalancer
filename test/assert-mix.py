#!/usr/bin/env python3
"""Assert what a balancer does with MORE THAN ONE KIND of item.

Every other suite ran iron plates through everything, so the multi-kind half of
`guest/go/carry.go` -- the pool's (name, quality, stack size) key, the per-kind
split, insertRemainder's walk over several groups -- had never been exercised at
all. This one runs two pure belts, then a sushi belt, through balancers and
counts every item in the world BY NAME either side of a forced recompile.

WHAT THIS SUITE CANNOT REACH, because it is base only: `compile.go`'s
`detailedTally` and `kindAt`. Below the stacking gate the drain takes the flat
totals and that code is never called at all, however many kinds are in flight, so
multi-kind AND STACKED is a Space Age question and lives in the `plat` suite's
`smix` band. See CLAUDE.md, "Stacked sushi".

Per name and not as one total, which is the whole point: a teardown that dropped
one kind and reinserted the rest conserves nothing, and a single total would have
to lose the same number of items twice in opposite directions to hide it.

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

# How many distinct item names the `many` rig's four sushi sources cover between
# them. Asserted rather than trusted: a rig that lost half its list to a rename
# would stop overflowing and would then pass every conservation check vacuously.
MANY_KINDS = 48

T0, T1 = 1400, 3140

# `duo` is two PURE belts -- one iron, one copper -- into a 2->2 that drains
# freely. Two belts in, two out, so two belts' worth had better come out.
DUO_TYPES = ("iron-plate", "copper-plate")

# HOW EVEN THE TYPE SPLIT HAS TO BE, AND WHY IT IS A FLOOR RATHER THAN A BAND.
#
# THE BUTTERFLY BALANCES COUNTS, NOT KINDS. Nothing in plan.Build knows what an
# item is; a splitter divides its input side between its output side by position,
# and an item's NAME never enters the arithmetic. So the type mix at an output is
# whatever the geometry happens to produce, it is NOT a designed guarantee, and
# no future planner change is obliged to preserve it.
#
# MEASURED, on this rig, 2026-08-04: a 2->2 fed one pure iron belt and one pure
# copper belt delivers 1306 and 1304 items -- exactly balanced by COUNT, 0.15%
# spread -- and the mix at each output is 75/25 in favour of the kind fed on its
# own side (out1 copper 981 / iron 325, out2 iron 977 / copper 327). Half and
# half is the naive expectation and it is not what the machine is for; the ratio
# is a consequence of the lane geometry and is recorded here as a measurement
# rather than explained, because nothing in the mod chose it.
#
# What IS load-bearing is that neither output is STARVED of a kind: an output
# seeing only iron would mean the two kinds took different paths through the
# network, which is a real defect and would read as 0%. The floor is 15%, which
# is well clear of the 25% the geometry produces and a long way from the 0% a
# defect produces.
TYPE_FLOOR = 0.15


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

    if "duo" not in samples or T0 not in samples["duo"] or T1 not in samples["duo"]:
        fail.append("duo: never reported")
    else:
        a, b = samples["duo"][T0], samples["duo"][T1]
        deltas = [b[i] - a[i] for i in range(len(b))]
        total = sum(deltas)
        ratio = total / float(belt)
        mean = total / float(len(deltas))
        spread = (max(deltas) - min(deltas)) / mean if mean else 1.0
        print("duo: two PURE belts (iron, copper) into a 2->2, per-output %s, "
              "%d items, %.3fx one belt, spread %.2f%%"
              % (deltas, total, ratio, spread * 100))
        if ratio < 1.98 or ratio > 2.02:
            fail.append(
                "duo: delivered %.3f belts, expected 2.0 -- a balancer carrying "
                "two kinds must not throttle" % ratio)
        if spread > 0.01:
            fail.append("duo: outputs spread %.2f%% (%s), over the 1%% bound"
                        % (spread * 100, deltas))

        # BOTH kinds at BOTH outputs. See TYPE_FLOOR above for why the bound is
        # a floor rather than a band.
        for i in (1, 2):
            per0 = outkinds.get(("duo", i), {}).get(T0)
            per1 = outkinds.get(("duo", i), {}).get(T1)
            if per0 is None or per1 is None:
                fail.append("duo: output %d never reported its per-kind split" % i)
                continue
            win = {n: per1.get(n, 0) - per0.get(n, 0) for n in set(per0) | set(per1)}
            got = sum(win.values()) or 1
            shown = " ".join("%s:%d" % (n, win[n]) for n in sorted(win))
            print("     out%d over the window: %s" % (i, shown))
            for name in DUO_TYPES:
                share = win.get(name, 0) / float(got)
                if share < TYPE_FLOOR:
                    fail.append(
                        "duo: output %d received %.1f%% %s, under the %.0f%% "
                        "floor -- the two kinds are taking different paths "
                        "through the network"
                        % (i, share * 100, name, TYPE_FLOOR * 100))

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

    if fail:
        print("\nMIX ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        return 1
    print("\nmix assertions passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
