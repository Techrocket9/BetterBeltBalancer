#!/usr/bin/env python3
"""Assert what a balancer with a priority port delivers, and what it refuses.

The promise is two formulas (agents/priority.md, "The promise"): with S belts
arriving and q of the M outputs flagged, a flagged port carries min(S/q, 1) and
every other one carries min(max(S-q, 0)/(M-q), 1). EVERY NUMBER IN THE TABLE
BELOW IS THAT ARITHMETIC, and every one of them is read against `ctrl`, a bare
express belt in the same save.

    python3 test/assert-prio.py create.log run.log
    python3 test/assert-prio.py --leg upgrade create.log run.log

TWO LEGS. The first builds the world and runs it. The second is `upg`'s
question asked of a priority save: the guest heap is discarded between the
phases, so the registry is re-derived from the world with `pprio` zero for every
node -- and the only place the flag survives is the part's own
`graphics_variation`. `recoverPriority` is what reads it back, and if it did not,
every priority network would be ADOPTED anyway (the interfaces are in the same
places) and the loss would be silent until the first audit found the fingerprint
moved. So the upgrade leg asserts the rebuild adopted everything, the audit after
it reads drift=0, and the rates are the same ones. All three fail together when
the flag is lost, and none of them alone would.
"""

import argparse
import re
import sys

# --- the mod's own lines -----------------------------------------------------
TOGGLE = re.compile(r"\[BBB\] priority part=(-?\d+),(-?\d+) (on|off)")
PRIO_REFUSED = re.compile(
    r"\[BBB\] alert: priority refused for cluster (\d+) at part (-?\d+),(-?\d+): (.*)$")
BUILD_REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) (?:cannot be built with (\d+) priority outputs"
    r"|asks for (\d+) priority inputs) over (\d+)->(\d+) ports")
SKIN = re.compile(r"\[BBB\] skin cluster=(\d+) parts=(\d+) set=(\d+) vars=(\S+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)")
TEARDOWN = re.compile(r"\[BBB\] torn down cluster")
COMPILED = re.compile(r"\[BBB\] compiled cluster")
SPILL = re.compile(r"\[BBB\] spilled (\d+) items beside cluster")
REBUILT = re.compile(
    r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters "
    r"\((\d+) networks adopted, (\d+) rebuilt\)")
# The one line that must never appear: the guest refusing a toggle because the
# machine is over the port cap is a different bound with a different sentence,
# and so is the one-belt-per-part rule.
OVER_LIMIT = re.compile(r"\[BBB\] alert: cluster \d+ would need")
SINGLE_EDGE = re.compile(r"\[BBB\] (?:alert: )?single-edge:")

# --- the observer's lines ----------------------------------------------------
SAMPLE = re.compile(r"\[BBB-PRIO\] t=(\S+) rig=(\S+) out=\[([^\]]*)\]")
DRAW = re.compile(r"\[BBB-PRIO\] draw t=(\S+) rig=p32 in=\[([^\]]*)\]")
FLAG = re.compile(
    r"\[BBB-PRIO\] flag t=(\S+) rig=(\S+) at=(-?\d+),(-?\d+) want=(\S+) accepted=(\S+)")
VAR = re.compile(r"\[BBB-PRIO\] var t=(\S+) rig=(\S+) at=(-?\d+),(-?\d+) v=(-?\d+)")
ITEMS = re.compile(
    r"\[BBB-PRIO\] items t=(\S+) rig=(\S+) ground (\d+)->(\d+) lines (\d+)->(\d+) "
    r"total (\d+)->(\d+)")
LANE = re.compile(r"\[BBB-PRIO\] lane t=(\d+) out=(\d+) left=(\d+) right=(\d+)")
BELTS = re.compile(r"\[BBB-PRIO\] belts rig=(\S+) items=(\d+)")
DRAINED = re.compile(r"\[BBB-PRIO\] drained rig=(\S+) rows=(\d+) chests=(\d+)")
BOUNDARY = re.compile(r"\[BBB-PRIO\] boundary try=(\d+) tick=(\d+) accepted=(\S+)")

# THE WORLD, WRITTEN DOWN HERE RATHER THAN READ OFF THE GUEST. `mar`'s own red
# proof is why: an injected defect that halved a rig passed every assertion that
# suite had, because every number it checked came from the classification it had
# broken. Twenty clusters over one hundred and seventy-nine parts.
CLUSTERS, PARTS = 20, 179

# The main rate window. Both ends are reported and the delta between them is
# what a rig delivered; a balancer has a pipeline several linked-belt hops long,
# so the items standing in it at the first sample would otherwise read as
# throughput it never produced.
T0, T1 = "t1", "t2"

# How many parts the create phase flags. Written down rather than counted off the
# log, because "every flag was accepted" is vacuous over a create that asked for
# none.
CREATE_FLAGS = 17

# The skin cells: 1..47 are the shapes and 48..94 are the same shapes badged
# (guest/go/skin/skin.go). A part's variation is where the flag lives.
BADGE = 47
CELLS = 94

# EVERY EXPECTED PER-PORT RATE, in belts, in the order the observer reports the
# rig's chests. None is a port with no chest at all (`pblk`'s dead-ended one),
# which the observer reports as -1 and which must stay -1.
#
# The arithmetic, rig by rig, is in guest/go/obs/prio/main.go's rig table; what
# is repeated here is only the result, because a script that recomputed it from
# the same formula the mod uses would agree with a defect in that formula.
EXPECT = {
    "p4lo":  [1 / 3.0, 0, 0, 0],
    "p4mid": [1, 1 / 3.0, 1 / 3.0, 1 / 3.0],
    "p4sat": [1, 1, 1, 1],
    "p2lo":  [1 / 3.0, 0],
    "p2mid": [1, 1 / 3.0],
    "p2sat": [1, 1],
    "pq2":   [0.5, 0.5, 0, 0],
    "p35":   [1, 0.5, 0.5, 0.5, 0.5],
    "p32":   [1, 1],
    "pblk":  [None, 1 / 3.0, 1 / 3.0, 1 / 3.0],
    "plane": [1, 0],
    "ptog":  [2 / 3.0, 2 / 3.0],
    "pin":   [1, 1],
    # pbig and pgrow are NOT here and the refusal window is where they are
    # checked. A P = 64 network is ten thousand item positions and this window
    # opens at tick 1800: what it would measure is a machine still filling.
    #
    # The five dead-ended rigs have no chest at all until the very end of the
    # run, and a rig that quietly grew one would be draining while it was
    # supposed to be filling -- which is the one condition the spill guard and
    # the three holds both need.
    "pbnd":  [None, None],
    "mp22":  [None, None],
    "mq22":  [None, None],
    "mp44":  [None, None, None, None],
    "pfull": [None, None, None, None],
}

# The toggle rig, either side of the flag. It is the same 2 -> 2 fed one express
# belt and one normal-tier one, so S = 4/3: plain it splits evenly and with a
# priority port it fills that port first.
EXPECT_TOG_ON = [1, 1 / 3.0]
EXPECT_TOG_OFF = [2 / 3.0, 2 / 3.0]

# The refusal window, which spans all three refusals. Every one of them has to
# leave the balancer it was asked about running exactly as it was.
EXPECT_REF = {
    "pin":   [1, 1],
    "pbig":  [1, 1],
    "pgrow": [1, 1, 1],
}

# A RATE'S BOUND. `m2` uses 2% on a rig total and this is per PORT, where a
# pipeline that is a few items further along at one end of the window than the
# other shows up directly, so 3% of a belt.
TOL = 0.03
# A port the formulas give nothing may still carry a handful of items from the
# tick the network was built in. One percent of a belt is two orders of
# magnitude below the smallest live rate in the table (a third of a belt) and
# two above anything a settled port can dribble.
ZERO = 0.01

# The audits, as exact tuples. `steady` is every rig standing as the save was
# written; `post-refuse` and after it carry pgrow, which a build has taken past
# the port cap's priority bound and which compile() refuses IN FRONT OF its own
# teardown -- so the network is still there, the stored fingerprint no longer
# describes the world, and the cluster is counted refused rather than unbuilt.
AUDIT_STEADY = (CLUSTERS, PARTS, CLUSTERS, 0, 0, 0)
# ...and after the settings paste, which must have changed NOTHING: the flag did
# not move, so the cluster's edge list did not either.
AUDIT_PASTE = AUDIT_STEADY
AUDIT_REFUSED = (CLUSTERS, PARTS, CLUSTERS, 1, 0, 1)

# THE SUCCESSOR'S CAPACITY IS READ OUT OF THE GUEST'S OWN REFUSAL LINE rather
# than written down here, because the guard's arithmetic is what this suite is
# about and a constant repeated here would have to be kept in step with it. What
# IS written down is that the two numbers it compared are in the right order:
# `held` over `room` is the whole of the decision.


def parse(paths):
    out = []
    for p in paths:
        with open(p, errors="replace") as f:
            out.append(f.readlines())
    return out


def samples(lines):
    """tag -> rig -> [chest counts]."""
    got = {}
    for m in (SAMPLE.search(l) for l in lines):
        if m:
            got.setdefault(m.group(1), {})[m.group(2)] = [
                int(v) for v in m.group(3).split()]
    return got


def one(lines, rx, *keys):
    """The first match whose leading groups equal keys, or None."""
    for m in (rx.search(l) for l in lines):
        if m and tuple(m.groups()[:len(keys)]) == keys:
            return m
    return None


def between(lines, lo, hi, rx):
    """Every match of rx between the two named sample tags, in order.

    The observer reports a rig table at each tag, so a tag's first line is a
    position in the log and counting the mod's own lines between two of them is
    what says a leg cost one teardown rather than three.
    """
    at = []
    for i, l in enumerate(lines):
        m = SAMPLE.search(l)
        if m and m.group(1) in (lo, hi):
            at.append((m.group(1), i))
    a = next((i for t, i in at if t == lo), None)
    b = next((i for t, i in at if t == hi), None)
    if a is None or b is None:
        return None
    return [m for m in (rx.search(l) for l in lines[a:b]) if m]


def rate_rows(fail, samp, tags, table, what):
    """Compare a rig's per-port delta over a window with the two formulas.

    THE CONTROL IS READ OVER THE SAME WINDOW, which is the only way a belt means
    the same thing in a 1,740-tick window and in a 400-tick one. The first cut of
    this divided every window by the main window's control and reported four
    exact rigs at 0.23 of a belt.
    """
    lo, hi = tags
    if "ctrl" not in samp.get(lo, {}) or "ctrl" not in samp.get(hi, {}):
        fail.append("the control belt was not reported at both ends of the %s "
                    "window" % what)
        return
    belt = samp[hi]["ctrl"][0] - samp[lo]["ctrl"][0]
    if belt <= 0:
        fail.append("the control belt delivered %d items over the %s window"
                    % (belt, what))
        return
    for rig, want in sorted(table.items()):
        if rig not in samp.get(lo, {}) or rig not in samp.get(hi, {}):
            fail.append("%s: not reported at both ends of the %s window" % (rig, what))
            continue
        a, b = samp[lo][rig], samp[hi][rig]
        if len(a) != len(want) or len(b) != len(want):
            fail.append("%s: %d chests reported and the rig has %d"
                        % (rig, len(b), len(want)))
            continue
        got = []
        for i, w in enumerate(want):
            if w is None:
                if a[i] != -1 or b[i] != -1:
                    fail.append("%s port %d has a chest and it is the DEAD-ENDED "
                                "one" % (rig, i + 1))
                got.append(None)
                continue
            got.append((b[i] - a[i]) / float(belt))
        shown = " ".join("-" if g is None else "%.3f" % g for g in got)
        print("  %-6s %-34s want %s"
              % (rig, shown, " ".join("-" if w is None else "%.3f" % w for w in want)))
        for i, w in enumerate(want):
            if w is None:
                continue
            g = got[i]
            if w == 0:
                if g > ZERO:
                    fail.append(
                        "%s port %d delivered %.3f belts over the %s window and the "
                        "two formulas give it nothing: with S belts arriving and the "
                        "priority tier not full, there is no residual to share"
                        % (rig, i + 1, g, what))
                continue
            if abs(g - w) > TOL:
                fail.append(
                    "%s port %d delivered %.3f belts over the %s window, want %.3f"
                    % (rig, i + 1, g, what, w))


def unchanged(fail, samp, before, after, rigs):
    """A rig delivering the same before and after, each against its own control.

    A refusal has to leave the machine it was asked about running exactly as it
    was, and the two windows are equal in length -- so the ratio of the two
    control-normalised totals is 1 whatever the absolute rate happens to be.
    """
    for rig in rigs:
        tot = []
        for lo, hi in (before, after):
            if rig not in samp.get(lo, {}) or rig not in samp.get(hi, {}):
                fail.append("%s: not reported at both ends of the %s window"
                            % (rig, lo))
                tot = None
                break
            belt = samp[hi]["ctrl"][0] - samp[lo]["ctrl"][0]
            live = [b - a for a, b in zip(samp[lo][rig], samp[hi][rig]) if a >= 0]
            tot.append(sum(live) / float(belt) if belt else 0)
        if not tot:
            continue
        ratio = tot[1] / tot[0] if tot[0] else 0
        print("    %-6s %.3f belts before the refusals, %.3f after (x%.3f)"
              % (rig, tot[0], tot[1], ratio))
        if abs(ratio - 1) > 0.05:
            fail.append(
                "%s delivered %.3f belts before the refusals and %.3f after. A "
                "refusal is asked in front of the teardown, so the standing "
                "network is not touched at all and the rate cannot move"
                % (rig, tot[0], tot[1]))


def audit_tuples(lines):
    return [tuple(int(g) for g in m.groups())
            for m in (AUDIT.search(l) for l in lines) if m]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--leg", default="fresh", choices=("fresh", "upgrade"))
    ap.add_argument("logs", nargs=2)
    args = ap.parse_args()
    create, run = parse(args.logs)
    fail = []

    # ------------------------------------------------------------------ create
    #
    # THE FLAGS THE CREATE PHASE SET, and that they were ACCEPTED. Everything
    # below rests on the world having been built with priority ports in it, and
    # a create whose remote calls were all refused would deliver plain rates on
    # every rig and fail thirty assertions without naming the cause.
    flags = [m for m in (FLAG.search(l) for l in create) if m]
    wanted = [m for m in flags if m.group(1) == "create"]
    if len(wanted) != CREATE_FLAGS:
        fail.append("the create phase asked for %d flags and the rigs carry %d"
                    % (len(wanted), CREATE_FLAGS))
    refused = [m for m in wanted if m.group(6) != "true"]
    if refused:
        fail.append("the create phase was refused %d of its flags, first on %s: "
                    "every rate below would be a plain balancer's"
                    % (len(refused), refused[0].group(2)))
    print("  create: %d flags asked for, %d accepted" % (len(wanted), len(wanted) - len(refused)))

    # AND THAT THE ENGINE IS SHOWING THEM. The flag lives in the part's own
    # `graphics_variation`, which is what carries it across a blueprint and
    # across a rebuilt guest, so a create that set the registry and not the
    # picture would lose every one of them on the next load.
    badged = {"p4lo", "p4mid", "p4sat", "p2lo", "p2mid", "p2sat", "pq2", "p35",
              "p32", "pblk", "plane", "pfull", "pgrow", "pbnd", "mq22"}
    plain = {"ptog", "pin", "pbig", "mp22", "mp44"}
    for m in (m for m in (VAR.search(l) for l in create) if m and m.group(1) == "create"):
        rig, v = m.group(2), int(m.group(5))
        if not 1 <= v <= CELLS:
            fail.append("%s's flagged part is drawing cell %d, which is outside "
                        "1..%d" % (rig, v, CELLS))
        elif rig in badged and v <= BADGE:
            fail.append("%s's part at %s,%s is drawing cell %d, which is an "
                        "unbadged shape: the flag was set and the picture did "
                        "not follow, so the next load would lose it"
                        % (rig, m.group(3), m.group(4), v))
        elif rig in plain and v > BADGE:
            fail.append("%s's part at %s,%s is drawing cell %d, which is a badged "
                        "one, and nothing has flagged it"
                        % (rig, m.group(3), m.group(4), v))

    got = audit_tuples(create)
    if not got:
        fail.append("no audit in the create phase")

    # ----------------------------------------------------------- the rebuild
    if args.leg == "upgrade":
        hits = [m for m in (REBUILT.search(l) for l in run) if m]
        if not hits:
            print("the guest never rebuilt its registry from the world; this leg "
                  "is about a save whose guest heap was discarded and there is "
                  "nothing for it to be about")
            sys.exit(1)
        surfaces, parts, clusters, adopted, rebuilt = (int(g) for g in hits[0].groups())
        print("  rebuilt: %d surfaces, %d parts, %d clusters, %d adopted, %d rebuilt"
              % (surfaces, parts, clusters, adopted, rebuilt))
        if (clusters, parts) != (CLUSTERS, PARTS):
            fail.append("the rebuild found %d clusters of %d parts and the rigs "
                        "build %d of %d" % (clusters, parts, CLUSTERS, PARTS))
        if rebuilt:
            fail.append(
                "%d networks were rebuilt rather than adopted. Every network in "
                "this save is exactly what the world describes, priority ports "
                "included, so a rebuild here is the guest throwing away a "
                "machine it could have kept" % rebuilt)
        if adopted != CLUSTERS:
            fail.append("%d networks were adopted and the save holds %d"
                        % (adopted, CLUSTERS))
    else:
        if [m for m in (REBUILT.search(l) for l in run) if m]:
            fail.append("the registry was rebuilt from the world on a leg whose "
                        "guest heap was not touched")

    # --------------------------------------------------------------- the rates
    samp = samples(run)
    if T0 not in samp or T1 not in samp:
        print("the rig table was not reported at both ends of the window")
        sys.exit(1)
    belt = samp[T1]["ctrl"][0] - samp[T0]["ctrl"][0]
    if belt <= 0:
        print("the control belt delivered %d items -- nothing is moving" % belt)
        sys.exit(1)
    print("\none saturated express belt over the window: %d items\n" % belt)
    print("  %-6s %-34s %s" % ("rig", "per-port belts delivered", "want"))
    rate_rows(fail, samp, (T0, T1), EXPECT, "main")

    # THE LANE SAMPLE, which is the one thing a chest total cannot see.
    #
    # `plane` is fed by two SIDE-LOADED belts, so each input is half a belt on
    # ONE lane, and S = 1 with q = 1 means the priority port carries a full belt.
    # A network that preserved lanes would fill the same chest at the same rate
    # with every item on one side of the belt, so the totals above cannot tell
    # the two apart and this is what does.
    lanes = [m for m in (LANE.search(l) for l in run) if m and m.group(2) == "1"]
    if len(lanes) < 5:
        fail.append("the priority port's lanes were sampled %d times and the "
                    "schedule takes five" % len(lanes))
    both = [m for m in lanes if int(m.group(3)) > 0 and int(m.group(4)) > 0]
    if len(both) != len(lanes):
        fail.append(
            "%d of %d samples of plane's priority port have items on only ONE "
            "lane: two half-lane inputs have to come out as one full belt, and a "
            "lane-preserving network delivers the same chest total"
            % (len(lanes) - len(both), len(lanes)))
    if lanes:
        print("\n  plane's priority port, per lane: %s"
              % " ".join("%s/%s" % (m.group(3), m.group(4)) for m in lanes))

    # p32: EVERY INPUT DRAWN EQUALLY. The balancing butterfly in front of the
    # concentrator exists for exactly this -- without it the row an input
    # happened to sit on would decide how much of it was taken once the network
    # filled (agents/priority.md, "The construction").
    a = one(run, DRAW, T0)
    b = one(run, DRAW, T1)
    if not a or not b:
        fail.append("p32's finite sources were not read at both ends of the window")
    else:
        before = [int(v) for v in a.group(2).split()]
        after = [int(v) for v in b.group(2).split()]
        drawn = [x - y for x, y in zip(before, after)]
        mean = sum(drawn) / float(len(drawn)) if drawn else 0
        spread = (max(drawn) - min(drawn)) / mean if mean else 1.0
        print("\n  p32 drew %s from its three inputs, spread %.2f%%"
              % (drawn, spread * 100))
        if min(drawn) <= 0:
            fail.append("p32 drew %s: an input that was never drawn from makes "
                        "'equally' vacuous" % drawn)
        elif spread > 0.01:
            fail.append(
                "p32's inputs were drawn %s, a spread of %.2f%% over the 1%% "
                "bound. A saturated balancer takes the same from every input, "
                "and a concentrator's tree is deliberately asymmetric, so this "
                "is what says the butterfly in front of it is doing its job"
                % (drawn, spread * 100))
        if min(after) <= 0:
            fail.append("one of p32's sources emptied, so the last part of the "
                        "window measured a different rig")

    # ------------------------------------------------------------- the toggle
    #
    # ON then OFF on a RUNNING balancer, through the mod's own remote method.
    print("\n  the toggle, on a running 2 -> 2:")
    for tag, want, on in (("tog-on", EXPECT_TOG_ON, "true"),
                          ("tog-off", EXPECT_TOG_OFF, "false")):
        m = one(run, FLAG, tag)
        if not m:
            fail.append("%s: the toggle was never asked for" % tag)
            continue
        if m.group(6) != "true":
            fail.append("%s: the mod refused the toggle, and a running 2 -> 2 with "
                        "one priority port is the shape this whole feature is "
                        "about" % tag)
        t = one(run, TOGGLE, m.group(3), m.group(4), "on" if on == "true" else "off")
        if not t:
            fail.append("%s: the guest never logged the flag moving at %s,%s"
                        % (tag, m.group(3), m.group(4)))
        # ITEM CONSERVATION, INSIDE ONE TICK. The observer counts, toggles,
        # forces the flush with an audit marker and counts again, so nothing
        # anywhere in the save can have moved for any other reason.
        it = one(run, ITEMS, tag)
        if not it:
            fail.append("%s: no item count around the toggle" % tag)
        else:
            gb, ga, tb, ta = (int(it.group(3)), int(it.group(4)),
                              int(it.group(7)), int(it.group(8)))
            print("    %-8s items %d -> %d, ground %d -> %d" % (tag, tb, ta, gb, ga))
            if ta != tb:
                fail.append(
                    "%s: %d items before the toggle and %d after, inside one "
                    "tick. A toggle is a recompile and a recompile is not a "
                    "removal: the drained items go back inside the network the "
                    "same flush builds" % (tag, tb, ta))
            if ga != gb:
                fail.append("%s: %d items reached the ground. A priority change "
                            "never spills" % (tag, ga - gb))
        v = one(run, VAR, tag)
        if not v:
            fail.append("%s: the flagged part's picture was not read back" % tag)
        else:
            badged_now = int(v.group(5)) > BADGE
            if badged_now != (on == "true"):
                fail.append("%s: the part is drawing cell %s and the flag is %s"
                            % (tag, v.group(5), on))
    # ONE TEARDOWN AND ONE COMPILE PER TOGGLE, and no more. A flag is the only
    # thing that moved, so exactly one cluster's fingerprint did.
    for tag, nxt in (("tog-on", "tog-on-a"), ("tog-off", "tog-off-a")):
        for rx, want, what in ((TEARDOWN, 1, "teardown"), (COMPILED, 1, "compile")):
            got = between(run, tag, nxt, rx)
            if got is None:
                continue
            if len(got) != want:
                fail.append("%s: %d %ss where a toggle is %d"
                            % (tag, len(got), what, want))
    rate_rows(fail, samp, ("tog-on-a", "tog-on-b"),
              {"ptog": EXPECT_TOG_ON}, "flag ON")
    rate_rows(fail, samp, ("tog-off-a", "tog-off-b"),
              {"ptog": EXPECT_TOG_OFF}, "flag OFF")

    # ----------------------------------------------------------- the refusals
    print("\n  the refusals:")
    refusals(run, fail)
    rate_rows(fail, samp, ("ref-a", "ref-b"), EXPECT_REF, "refusal")
    # ...AND THE SHARPER FORM OF THE SAME CLAIM: each of the three is compared
    # with ITSELF over an equal window taken before the refusals, because "the
    # balancer keeps running" is a statement about that machine and not about a
    # number somebody worked out.
    unchanged(fail, samp, ("ref-pre", "ref-mid"), ("ref-a", "ref-b"),
              sorted(EXPECT_REF))

    # --------------------------------------------------------- the spill guard
    print("\n  the spill guard:")
    spill_guard(run, fail)

    # ------------------------------------------------- the boundary, and the holds
    print("\n  the boundary the guard lets go on:")
    boundary(run, fail)
    print("\n  what a jammed balancer was holding:")
    holds(run, fail)

    # ------------------------------------------------------------- the audits
    steady = audit_tuples(run)
    if not steady:
        fail.append("no audit in the benchmark phase")
    else:
        if steady[0] != AUDIT_STEADY:
            fail.append("the steady-state audit is %s and the rigs build %s"
                        % (steady[0], AUDIT_STEADY))
        # THE PASTE CHANGED NOTHING, which is the whole of leg (b): a flag the
        # planner refused must not reach the registry by the back door, and the
        # engine has by then already written the source's badged picture onto the
        # destination. The audit two ticks after the queueing edit is what says
        # the next COMPILE read the registry rather than that picture.
        if len(steady) > 1 and steady[1] != AUDIT_PASTE:
            fail.append(
                "the audit after the settings paste is %s and nothing about that "
                "cluster may have moved: it is %s. A paste onto a balancer no "
                "priority port fits is refused, so the flag stays down -- and a "
                "refusal that left the part WEARING the source's badge can be "
                "read back as a flag by the next restyle, which is a priority "
                "port nobody asked for on a machine that cannot have one"
                % (steady[1], AUDIT_PASTE))
        if steady[-1] != AUDIT_REFUSED:
            fail.append(
                "the final audit is %s and the rigs leave %s: pgrow's flagged "
                "output has been made unbuildable by a BUILD, which compile() "
                "refuses in front of its own teardown -- so its network is still "
                "standing, its stored fingerprint no longer describes the world, "
                "and it is counted refused and not unbuilt"
                % (steady[-1], AUDIT_REFUSED))

    # ---------------------------------------------------------- the negatives
    if [l for l in run if OVER_LIMIT.search(l)] or [l for l in create if OVER_LIMIT.search(l)]:
        fail.append("a cluster was refused for the PORT LIMIT. No rig here goes "
                    "past sixty-four belts, and a shape that broke both bounds "
                    "would take that sentence rather than a priority one")
    if [l for l in run if SINGLE_EDGE.search(l)] or [l for l in create if SINGLE_EDGE.search(l)]:
        fail.append("the one-belt-per-part rule spoke. Every rig here is laid one "
                    "belt per part, so a refusal for it is a defect in the SAVE")

    if fail:
        print("\nPRIORITY ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\npriority assertions passed (%d rate rigs, three refusals and the "
          "spill guard)" % len(EXPECT))


def refusals(run, fail):
    """The three shapes a priority port is refused, each with its own sentence."""
    # A PRIORITY INPUT, refused at every size, because the construction for one
    # is not built (agents/priority.md, "What input priority does").
    m = one(run, FLAG, "ref-in")
    if not m:
        fail.append("the input refusal was never asked for")
    else:
        if m.group(6) != "false":
            fail.append("flagging an INPUT part was ACCEPTED. The planner refuses "
                        "a priority input at every size and an approximation is "
                        "the one outcome this repository's rule forbids")
        line = [l for l in run
                if PRIO_REFUSED.search(l) and "is a priority input" in l]
        if not line:
            fail.append("no `is a priority input` refusal in the log")
        else:
            print("    input:  %s" % line[0].split("alert: ")[1].strip())
        var = one(run, VAR, "ref-in")
        if var and int(var.group(5)) > BADGE:
            fail.append("pin's part is drawing a badged cell after a refusal: a "
                        "toggle is refused BEFORE the flag moves")

    # TOO BIG. pbig is 33 -> 2, which is P = 64, and a priority network is two
    # butterflies wide: 35 columns of a 32-column slot.
    m = one(run, FLAG, "ref-big")
    if not m:
        fail.append("the too-big refusal was never asked for")
    else:
        if m.group(6) != "false":
            fail.append("flagging an output of a P=64 balancer was ACCEPTED, and "
                        "the priority construction does not fit the slot there")
        line = [l for l in run if PRIO_REFUSED.search(l) and "does not fit" in l]
        if not line:
            fail.append("no `does not fit` refusal in the log")
        else:
            print("    too big: %s" % line[0].split("alert: ")[1].strip())
        var = one(run, VAR, "ref-big")
        if var and int(var.group(5)) > BADGE:
            fail.append("pbig's part is drawing a badged cell after a refusal")

    # A SETTINGS PASTE ONTO A BALANCER NO PRIORITY PORT FITS. The third door onto
    # the flag, and the only one that can put one on a part the player never
    # pointed at. The handler asks the same pre-teardown check a keypress asks and
    # is refused -- and by then the ENGINE has already copied the source's badged
    # `graphics_variation` onto the destination, so what must not happen is that
    # picture being read back as a flag by the next restyle.
    v = one(run, VAR, "paste")
    if not v:
        fail.append("the paste destination's picture was never read back")
    else:
        print("    paste:  pbig's part at %s,%s is drawing cell %s"
              % (v.group(3), v.group(4), v.group(5)))
        if int(v.group(5)) > BADGE:
            fail.append(
                "pbig's part is drawing cell %s after a REFUSED settings paste, "
                "and a cell over %d is a priority badge. The flag did not move, "
                "so the picture has to come back to the shape the part actually "
                "has -- a badge left standing there is read back as a flag by the "
                "next restyle, on a balancer that cannot carry one"
                % (v.group(5), BADGE))

    # A BUILD that makes a standing priority network unbuildable, and then a flag
    # taken OFF it. pgrow is a 32 -> 3 carrying two priority ports; a 33rd input
    # belt takes it to P = 64, and neither the shape it has nor the shape one
    # fewer flag would give can be built there.
    got = [m for m in (BUILD_REFUSED.search(l) for l in run) if m]
    outs = [m for m in got if m.group(2)]
    want = [("2", "33", "3"), ("1", "33", "3")]
    shown = [(m.group(2), m.group(4), m.group(5)) for m in outs]
    if shown != want:
        fail.append(
            "the compile-time priority refusals are %s and the rigs produce %s: "
            "the grow gives pgrow two priority outputs over 33->3 ports, and "
            "taking one flag off gives it one over the same ports. A refusal "
            "naming any other shape is a cluster this leg did not touch"
            % (shown, want))
    for m in outs:
        print("    build:  cluster %s cannot be built with %s priority outputs "
              "over %s->%s ports" % (m.group(1), m.group(2), m.group(4), m.group(5)))

    # ...AND TAKING THE FLAG OFF IS ALLOWED. The shape does not fit either way,
    # so a guard that asked only "does the result fit" would refuse it -- and a
    # player whose balancer grew past the bound would be holding a machine they
    # could not un-break, with the only way out being to take the BELT away.
    m = one(run, FLAG, "unflag")
    if not m:
        fail.append("the un-flag was never asked for")
    else:
        if m.group(6) != "true":
            fail.append(
                "taking a flag OFF pgrow was REFUSED. The balancer does not fit a "
                "priority port with two flags on it and does not fit one with a "
                "single flag either, so this change cannot make anything worse -- "
                "and refusing it is what leaves the machine unrecoverable")
        v = one(run, VAR, "unflag")
        if v:
            print("    unflag: accepted, and the part is drawing cell %s" % v.group(5))
            if int(v.group(5)) > BADGE and m.group(6) == "true":
                fail.append("pgrow's part is still drawing a badged cell after its "
                            "flag was taken off")


def spill_guard(run, fail):
    """A priority change never puts anything on the ground.

    pfull is a dead-ended 4 -> 4 with one priority port, fed hard from tick 0, so
    by the time it is asked it is stationary and carrying more than the plain
    network it would become could hold. The guard reads what is standing, finds
    it over the successor's capacity, and refuses the change with the balancer
    still running (guest/go/priority.go, `prioFitsWhatIsStanding`).
    """
    m = one(run, FLAG, "spill-refused")
    if not m:
        fail.append("the spill guard was never asked")
        return
    if m.group(6) != "false":
        fail.append(
            "clearing pfull's flag was ACCEPTED while it was full. The plain "
            "network it would build holds fewer item positions than a jammed "
            "priority 4 -> 4 is carrying, so the difference would have gone on "
            "the ground")
    hold = [l for l in run if PRIO_REFUSED.search(l) and "the balancer holds" in l]
    if not hold:
        fail.append("no `the balancer holds` refusal in the log")
    else:
        got = re.search(r"holds (\d+) items and the network this would build "
                        r"holds (\d+) item positions", hold[0])
        if not got:
            fail.append("the holding refusal did not carry its two numbers")
        else:
            held, room = int(got.group(1)), int(got.group(2))
            print("    refused: the balancer holds %d items, the successor holds "
                  "%d positions" % (held, room))
            if room <= 0:
                fail.append("the successor was reported as holding %d item "
                            "positions, which is not a network at all" % room)
            if held <= room:
                fail.append(
                    "the balancer held %d items against %d positions, so the "
                    "guard refused a change that would have FITTED. This leg's "
                    "whole subject is a network too full to shrink and it was "
                    "not full enough" % (held, room))
    it = one(run, ITEMS, "spill-refused")
    if it and int(it.group(4)) != int(it.group(3)):
        fail.append("%d items reached the ground on a REFUSED toggle, which tears "
                    "nothing down at all" % (int(it.group(4)) - int(it.group(3))))
    var = one(run, VAR, "spill-refused")
    if var and int(var.group(5)) <= BADGE:
        fail.append("pfull's part stopped drawing a badged cell after a refused "
                    "toggle: the flag moved and the change was refused, which is "
                    "the worst of both")

    # ...and then the same toggle once it has drained.
    m = one(run, FLAG, "spill-cleared")
    if not m:
        fail.append("the drained toggle was never asked")
        return
    if m.group(6) != "true":
        fail.append("clearing pfull's flag was refused after it had drained. The "
                    "guard is about the moment and not about the shape, so it has "
                    "to come back the second the machine is empty")
    it = one(run, ITEMS, "spill-cleared")
    if not it:
        fail.append("no item count around the drained toggle")
    else:
        gb, ga, tb, ta = (int(it.group(3)), int(it.group(4)),
                          int(it.group(7)), int(it.group(8)))
        print("    cleared: items %d -> %d, ground %d -> %d" % (tb, ta, gb, ga))
        if ga != gb:
            fail.append("%d items reached the ground clearing a drained "
                        "balancer's flag" % (ga - gb))
        if ta != tb:
            fail.append("%d items before the drained toggle and %d after"
                        % (tb, ta))
    for rx, what in ((TEARDOWN, "teardown"), (COMPILED, "compile")):
        got = between(run, "spill-cleared", "final", rx)
        if got is not None and len(got) != 1:
            fail.append("the drained toggle cost %d %ss and a toggle is one"
                        % (len(got), what))
    spilled = [m for m in (SPILL.search(l) for l in run) if m]
    if spilled:
        fail.append("the guest spilled %d times in this run, %d items the first "
                    "time. Nothing here is a REMOVAL: every teardown in this save "
                    "is a recompile, and a recompile puts its items back inside "
                    "the network it builds"
                    % (len(spilled), int(spilled[0].group(1))))
    var = one(run, VAR, "spill-cleared")
    if var and int(var.group(5)) > BADGE:
        fail.append("pfull's part is still drawing a badged cell after the flag "
                    "was cleared")


def boundary(run, fail):
    """The guard letting go, which is the half `pfull` cannot show.

    pbnd is the same dead-ended priority 2 -> 2, opened with its feed cut, and the
    clear is asked every forty ticks while it drains. The tick it is accepted on
    IS the tick the machine crossed under the successor's capacity, so the
    reinsertion that follows is the tightest one this feature can produce -- the
    network is as full as it can be and still fit. What must come of it is
    nothing on the ground.
    """
    tries = [m for m in (BOUNDARY.search(l) for l in run) if m]
    if not tries:
        fail.append("the boundary was never polled")
        return
    ok = [m for m in tries if m.group(3) == "true"]
    print("    %d attempts, accepted on try %s"
          % (len(tries), ok[0].group(1) if ok else "none"))
    if not ok:
        fail.append(
            "pbnd's flag was refused on all %d attempts. It was opened with its "
            "feed cut, so it drains to nothing: a guard that never lets go is one "
            "a player cannot get past by waiting, which is the remedy its own "
            "message tells them to use" % len(tries))
        return
    if len(ok) > 1:
        fail.append("the boundary poll was accepted %d times and it stops at the "
                    "first" % len(ok))
    it = one(run, ITEMS, "boundary")
    if not it:
        fail.append("no item count around the accepted boundary toggle")
        return
    gb, ga, tb, ta = (int(it.group(3)), int(it.group(4)),
                      int(it.group(7)), int(it.group(8)))
    print("    items %d -> %d, ground %d -> %d" % (tb, ta, gb, ga))
    if ga != gb:
        fail.append(
            "%d items reached the ground on the toggle the guard ACCEPTED. That "
            "is the guard's arithmetic being wrong in the direction it exists to "
            "prevent: it said the successor could hold what was standing and the "
            "successor could not" % (ga - gb))
    if ta != tb:
        fail.append("%d items before the boundary toggle and %d after, inside one "
                    "tick" % (tb, ta))
    v = one(run, VAR, "boundary")
    if v and int(v.group(5)) > BADGE:
        fail.append("pbnd's part is still drawing a badged cell after the flag "
                    "was cleared")


def holds(run, fail):
    """What a jammed balancer was carrying, in items.

    NOT AN ASSERTION ABOUT A NUMBER. The capacity table in agents/priority.md is
    item POSITIONS, computed off the op list, and what fraction of them a jammed
    network really holds is the term the spill guard's bound rests on and the one
    thing no measurement anywhere had. Each rig is opened with its feed cut and
    drained into its own chests; what its own visible belts were carrying comes
    off, and what is left is the machine's.
    """
    belts = {m.group(1): int(m.group(2))
             for m in (BELTS.search(l) for l in run) if m}
    drained = {m.group(1): int(m.group(3))
               for m in (DRAINED.search(l) for l in run) if m}
    for rig in ("mp22", "mq22", "mp44", "pbnd"):
        if rig not in belts or rig not in drained:
            fail.append("%s was not measured at both ends of the drain" % rig)
            continue
        held = drained[rig] - belts[rig]
        print("    %-5s drained %5d, its own belts held %4d, so the machine held %5d"
              % (rig, drained[rig], belts[rig], held))
        if held <= 0:
            fail.append(
                "%s's machine is measured as holding %d items. It was dead-ended "
                "and fed from tick 0, so a jammed balancer holding nothing is a "
                "rig that never filled and every bound derived from this number "
                "would be derived from noise" % (rig, held))


if __name__ == "__main__":
    sys.exit(main() or 0)
