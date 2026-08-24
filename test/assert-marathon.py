#!/usr/bin/env python3
"""Assert the guest's permanent-heap SLOPE per net-zero world operation.

Under `-gc=leaking` -- which is what this mod ships -- nothing the guest
allocates ever comes back. So the interesting number is not the heap at any one
moment, it is the slope: how many bytes one complete place-and-remove cycle adds
that are then in the save, in every multiplayer join, and in Lua's collector's
walk for the rest of the session.

The instrument is the guest's own probe. `[BBB] heap post-audit sys=… alloc=…`
is `runtime.ReadMemStats` through TinyGo's bump allocator, so `alloc` is every
byte ever handed out. The test mod drives a leg, marks it, and asks for an
audit; this script pairs each `[BBB-MAR] leg=… i=…` marker with the heap line
that follows it and fits a slope.

EVERY LEG PAYS EXACTLY ONE AUDIT, and an audit is not free -- it re-classifies
every cluster in the save. `cal`, `calA` and `calZ` are that audit with the
world untouched, so subtracting the calibration slope is what turns a leg's
slope into the cost of the WORLD OPERATION rather than of measuring it. The
three calibration legs are taken before, in the middle of and after the run and
must agree, which is also the check that the audit's cost really is constant.

    python3 test/assert-marathon.py create.log run.log
"""

import re
import sys

MARK = re.compile(r"\[BBB-MAR\] leg=(\w+) i=(\d+)")
HEAP = re.compile(r"\[BBB\] heap post-audit sys=(\d+) alloc=(\d+) gc=(\d)")
COLLECTED = re.compile(
    r"\[BBB\] heap post-audit sys=(\d+) alloc=\d+ gc=1 heap=\d+ live=(\d+) "
    r"free=\d+ since=\d+ cycles=(\d+) grows=(\d+) steps=(\d+) deadlines=(\d+)")
CHURN = re.compile(r"\[BBB-MAR\] churn i=(\d+) total=(\d+)")
PLAN = re.compile(r"\[BBB-MAR\] plan legs=(\d+) end_tick=(\d+) stock=(\d+)")
DONE = re.compile(r"\[BBB-MAR\] done end_tick=(\d+) churn_final=(\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
)

# What each leg's iteration IS, and how many of the primitive underneath it one
# iteration contains. The divisor turns a per-iteration slope into a per-thing
# one; the text is what the report calls it.
LEGS = [
    ("A", 1, "a whole 4-part balancer: 16 entities placed in one tick, run under "
             "load, all 16 removed in one tick"),
    ("B", 2, "a belt laid inside the neighbour gate and picked up again -- "
             "re-classified, fingerprint matched, NOTHING rebuilt"),
    ("C", 2, "an input belt removed and put back -- the edge really moves, so a "
             "full teardown and rebuild each way"),
    ("D", 2, "a belt laid 18 tiles from anything and picked up again -- the "
             "guest is entered and rejects it, no compile at all"),
    ("E", 1, "six entities in one tick and six out in one tick: a small paste "
             "and its undo"),
    ("G", 2, "the same edit as C but on a 4x4 -- sixteen parts, a 32-entity "
             "network -- so C:G is how the compile term scales"),
    ("F", 1, "a saturated balancer grown by an edgeless part, taken apart "
             "entirely and rebuilt, with every item counted"),
]

CALS = ("cal", "calA", "calZ")

# THE WORLD EVERY PROBE IS TAKEN OVER, written down here rather than read off
# the guest's own audit, for the reason the whole suite exists: an audit is only
# a constant that can be subtracted if the world it re-classifies is the same
# world every time. `(clusters, parts, networks)` for the three permanent rigs
# -- KEEP and CHURN, each a 2->2 over four parts, and BIG, a 4x4 over sixteen --
# and every leg is written so that its probe fires with the world back in
# exactly that state.
#
# Leg F is the one exception and it is a designed one: its probe fires between
# the dissolve and the rebuild, so the churn rig is not there. It has its own
# constant rather than an exemption, because "the world is missing the four
# parts F took out" is a statement and "F may report anything" is not.
#
# THIS IS ALSO THE SUITE'S ANTI-VACUITY LINE, and it is here because it was
# missing: with leg F's rebuild crippled so that only half the churn rig came
# back, every existing assertion still passed -- the count never rose, nothing
# drifted, no cluster read `unbuilt` (a cluster with inputs and no outputs is a
# legitimate half-built state), and the calibration spread stayed at 0.0%. What
# actually happened was visible in one place only: `calZ` re-classifying a world
# with two fewer parts and one fewer network than `cal` did.
WORLD = (3, 24, 3)
WORLD_F = (2, 20, 2)

# The ceilings. Each is the measured value with headroom, and each exists so
# that an allocation regression in the path underneath it fails the suite rather
# than being discovered by a player three hundred hours in. Bytes per PRIMITIVE
# (i.e. after the divisor above), net of the audit.
CEILING = {
    "A": 12000.0,
    "B": 1200.0,
    "C": 6000.0,
    "D": 200.0,
    "E": 6000.0,
    "G": 12000.0,
    "F": 12000.0,
}

# Superlinearity. The second half of a leg may not cost materially more per
# iteration than the first half: a slope that grows with the number of
# iterations already run is the shape that makes a 300-hour save unplayable, and
# it is the whole reason this suite exists.
SUPERLINEAR = 1.35
SUPERLINEAR_FLOOR = 96.0  # bytes; below this the ratio is noise, not a shape

# Conservation, leg F. 100 cycles, each of which tears a FULL network down twice
# (a merge and a dissolve). M3 measures 0.89% lost over ~100 teardowns, to
# fractional item positions and whatever a splitter holds outside its transport
# lines.
CHURN_LOSS = 0.06


def read(paths):
    out = []
    for p in paths:
        with open(p, encoding="utf-8", errors="replace") as fh:
            out.extend(fh.readlines())
    return out


def samples(lines):
    """leg -> [(iteration, sys, alloc)], in log order.

    A marker claims the FIRST heap line after it, which is the `post-audit` line
    the marker's own audit wrote.
    """
    by_leg = {}
    pending = None
    for line in lines:
        m = MARK.search(line)
        if m:
            pending = (m.group(1), int(m.group(2)))
            continue
        h = HEAP.search(line)
        if h and pending:
            leg, i = pending
            by_leg.setdefault(leg, []).append(
                (i, int(h.group(1)), int(h.group(2)), int(h.group(3))))
            pending = None
    return by_leg


# Under `--gc=collected` the heap is RECLAIMED, so `alloc` is not a permanent
# total and its slope is not a leak rate -- the number that matters there is the
# linear memory, because that is what never shrinks and what Factorio bills.
# `make GC=collected` has to stay green, so this suite reports the right thing in
# both arms rather than only in the shipped one.
COLLECTED_SYS_CEILING = 4 * 1024 * 1024
COLLECTED_LIVE_CEILING = 256 * 1024


def slope(pts):
    """Bytes of permanent heap per iteration, over the whole leg."""
    if len(pts) < 2:
        return 0.0
    return (pts[-1][2] - pts[0][2]) / (pts[-1][0] - pts[0][0])


def halves(pts):
    mid = len(pts) // 2
    return slope(pts[:mid + 1]), slope(pts[mid:])


def fail(msg):
    print("FAIL: " + msg)
    sys.exit(1)


def collected_arm(lines, by_leg):
    """The same run under `--gc=collected`, where the claim is different.

    `alloc` is reclaimed there, so it is not a permanent total and the slope
    table above would be arithmetic on noise. What is asserted instead is the
    thing the collector exists to bound and the thing Factorio actually bills:
    LINEAR MEMORY, which never shrinks in either arm.
    """
    pts = [p for name, _, _ in LEGS for p in by_leg.get(name, [])]
    pts += [p for n in CALS for p in by_leg.get(n, [])]
    curve = sorted({p[1] for p in pts})
    stats = [tuple(int(g) for g in m.groups())
             for m in (COLLECTED.search(l) for l in lines) if m]
    if not stats:
        fail("the guest reported gc=1 with no collector statistics")
    last = stats[-1]
    print("--gc=collected: the slope table does not apply, because the heap is "
          "given back. What is measured instead is what never shrinks.")
    print()
    print("  linear memory over %d operations: %s"
          % (len(pts), " -> ".join("%.2f MiB" % (s / 1048576.0) for s in curve)))
    print("  live set at the end: %d B" % last[1])
    print("  collections %d, grows %d, paced steps %d, forward-progress "
          "deadlines %d" % (last[2], last[3], last[4], last[5]))
    if curve[-1] > COLLECTED_SYS_CEILING:
        fail("the collected arm reached %.2f MiB of linear memory, over the "
             "%.0f MiB ceiling: the collector is being outrun"
             % (curve[-1] / 1048576.0, COLLECTED_SYS_CEILING / 1048576.0))
    if last[1] > COLLECTED_LIVE_CEILING:
        fail("the live set is %d B, over the %d B ceiling: something is being "
             "RETAINED, which is a real leak in either arm"
             % (last[1], COLLECTED_LIVE_CEILING))
    # A non-zero count of something documented to be zero forever is a defect
    # report, not a statistic. This suite carried deadlines=6 for a day while the
    # root-scan starvation (FKLUA-GAPS item 21) went unnoticed; this assertion
    # would have caught it on the commit that caused it. If a legitimate reason
    # for a deadline ever exists, the fix is to document it HERE with a number,
    # not to delete the check.
    if last[5] != 0:
        fail("forward-progress deadlines = %d; the collected arm is documented "
             "to run at zero, and a non-zero count means the collector is being "
             "starved (see FKLUA-GAPS item 21 for the last cause)" % last[5])
    conservation(lines)
    drift(lines)
    print()
    print("marathon assertions passed (collected arm)")
    return


def conservation(lines):
    totals = [(int(m.group(1)), int(m.group(2)))
              for m in (CHURN.search(l) for l in lines) if m]
    if len(totals) < 50:
        fail("leg F logged %d item counts, expected 100" % len(totals))
    first, last = totals[0][1], totals[-1][1]
    rises = [(i, t) for (i, t), (_, prev) in zip(totals[1:], totals[:-1]) if t > prev]
    lost = first - last
    print()
    print("item conservation under 100 add-part/remove-everything cycles on a "
          "network that is full:")
    print("  first count %d, last %d, lost %d (%.2f%%) over %d cycles, %d "
          "teardowns" % (first, last, lost, 100.0 * lost / max(first, 1),
                         len(totals), 2 * len(totals)))
    if rises:
        fail("the count rose at %d cycles (first at i=%d): items were minted"
             % (len(rises), rises[0][0]))
    if lost > CHURN_LOSS * first:
        fail("leg F lost %.2f%% of its items, over the %.0f%% bound"
             % (100.0 * lost / first, 100 * CHURN_LOSS))
    print("  the count never rose, in any of %d cycles" % len(totals))


def drift(lines):
    # Every audit is claimed by the marker that asked for it. The `init` one is
    # exempt and only that one: it runs inside on_init, where the audit REPORTS
    # BEFORE IT REPAIRS and no cluster has been compiled yet, so `unbuilt` is
    # legitimately the whole save.
    audits, leg = [], None
    for line in lines:
        m = MARK.search(line)
        if m:
            leg = m.group(1)
            continue
        a = AUDIT.search(line)
        if a and leg is not None:
            audits.append((leg, tuple(int(g) for g in a.groups())))
    drifted = [(l, a) for l, a in audits if l != "init" and (a[3] != 0 or a[4] != 0)]
    print()
    print("%d audits over the run; drift=0 unbuilt=0 in %d of them (the on_init "
          "one is exempt: it reports before it repairs and nothing is compiled "
          "yet)" % (len(audits), len(audits) - len(drifted) - 1))
    if drifted:
        fail("%d audits reported drift or an unbuilt cluster, first %r"
             % (len(drifted), drifted[0]))

    # EVERY LEG LEAVES THE WORLD AS IT FOUND IT, which is what makes one audit
    # the same price as the next and the calibration subtractable at all. Each
    # leg's audits must therefore collapse to ONE (clusters, parts, networks)
    # tuple, and that tuple must be the one the rigs build.
    shapes = {}
    for leg, a in audits:
        if leg == "init":
            continue
        shapes.setdefault(leg, set()).add(a[:3])
    print("the world every probe was taken over:")
    for leg in sorted(shapes):
        seen = sorted(shapes[leg])
        want = WORLD_F if leg == "F" else WORLD
        print("  %-4s %s%s" % (leg, " and ".join(repr(s) for s in seen),
                               "" if seen == [want] else "   <-- expected %r" % (want,)))
        if len(seen) > 1:
            fail("leg %s audited %d different worlds (%s): a leg that does not "
                 "leave the world as it found it makes its own audit -- and "
                 "every calibration after it -- a different measurement"
                 % (leg, len(seen), ", ".join(repr(s) for s in seen)))
        if seen[0] != want:
            fail("leg %s audited a world of %r and the rigs build %r: either the "
                 "leg is not net-zero or a rig is not the shape this suite "
                 "thinks it is" % (leg, seen[0], want))

    if not any(DONE.search(l) for l in lines):
        fail("the run never reached its end tick; raise BBB_MAR_TICKS")


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(2)
    lines = read(sys.argv[1:])
    by_leg = samples(lines)
    collected = any(p[3] == 1 for pts in by_leg.values() for p in pts)

    plan = PLAN.search("\n".join(l for l in lines if "[BBB-MAR] plan" in l))
    if not plan:
        fail("the test mod never logged its plan; the run did not start")
    stock = int(plan.group(3))

    if collected:
        return collected_arm(lines, by_leg)

    # ---- the audit, measured three times ----------------------------------
    cal = {}
    for name in CALS:
        pts = by_leg.get(name)
        if not pts or len(pts) < 5:
            fail("calibration leg %s produced %d samples, expected 10"
                 % (name, len(pts or [])))
        cal[name] = slope(pts)
    audit_cost = sum(cal.values()) / len(cal)
    spread = (max(cal.values()) - min(cal.values())) / max(audit_cost, 1.0)
    print("one bbb-audit, world untouched: %s B of permanent heap"
          % " / ".join("%.0f" % cal[n] for n in CALS))
    print("  mean %.0f B, spread %.1f%% -- this is subtracted from every leg below"
          % (audit_cost, 100 * spread))
    if spread > 0.30:
        fail("the audit's own cost is not constant (%.1f%% spread over the run); "
             "no leg below can be attributed" % (100 * spread))
    print()

    # ---- the legs ----------------------------------------------------------
    print("%-4s %6s %12s %12s %12s  %s"
          % ("leg", "iters", "B/iter raw", "B/iter net", "B/primitive", "what one iteration is"))
    results = {}
    for name, divisor, what in LEGS:
        pts = by_leg.get(name)
        if not pts or len(pts) < 10:
            fail("leg %s produced %d samples" % (name, len(pts or [])))
        raw = slope(pts)
        net = raw - audit_cost
        per = net / divisor
        results[name] = (raw, net, per, pts)
        print("%-4s %6d %12.0f %12.0f %12.0f  %s"
              % (name, len(pts), raw, net, per, what.split(" -- ")[0]))
    print()

    # ---- is any of it superlinear? -----------------------------------------
    print("linearity -- the second half of a leg against its first:")
    for name, _, _ in LEGS:
        pts = results[name][3]
        a, b = halves(pts)
        a_net, b_net = a - audit_cost, b - audit_cost
        ratio = b_net / a_net if a_net > SUPERLINEAR_FLOOR else 1.0
        print("  %-4s first half %8.0f B/iter, second half %8.0f B/iter  (x%.2f)"
              % (name, a_net, b_net, ratio))
        if a_net > SUPERLINEAR_FLOOR and ratio > SUPERLINEAR:
            fail("leg %s costs %.2fx more per iteration in its second half than "
                 "its first: the slope GROWS with the number of operations "
                 "already done, which is the 300-hour killer" % (name, ratio))
    print()

    # ---- the ceilings ------------------------------------------------------
    for name, _, what in LEGS:
        per = results[name][2]
        if per > CEILING[name]:
            fail("leg %s costs %.0f B per %s, over the documented ceiling of "
                 "%.0f B" % (name, per, what.split(" --")[0], CEILING[name]))
    print("every leg is inside its documented ceiling")

    # ---- linear memory, which is what Factorio actually bills --------------
    all_pts = [p for name, _, _ in LEGS for p in results[name][3]]
    all_pts += [p for n in CALS for p in by_leg.get(n, [])]
    sys_seen = sorted({p[1] for p in all_pts})
    print("linear memory over the run (`sys`, the number Factorio's collector "
          "walks): %s" % " -> ".join("%.2f MiB" % (s / 1048576.0) for s in sys_seen))
    print()

    # ---- conservation, and the guest agreeing with the world --------------
    conservation(lines)
    print("  stock put in: %d" % stock)
    drift(lines)
    print()
    print("marathon assertions passed")


if __name__ == "__main__":
    main()
