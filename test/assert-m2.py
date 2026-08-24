#!/usr/bin/env python3
"""Assert the compiled network's behaviour from Factorio's own log.

The test mod logs what each output chest held at tick 1800 and tick 3540 and
nothing else; every judgement is here. Rates are measured over the WINDOW
between the two samples rather than from zero, because a balancer has a pipeline
(several linked-belt hops at ~32 items of fill each) and the items sitting in it
at tick 1800 would otherwise read as throughput the balancer never delivered.

`ctrl` is a bare express belt run under the same conditions in the same save. It
is the yardstick: "full throughput" means each output matches what one saturated
express belt delivered in the same window, as measured by the engine, not by
arithmetic on a number off a wiki.

    python3 test/assert-m2.py create.log run.log
"""

import re
import sys

SAMPLE = re.compile(r"\[BBB-M2\] t=(\d+) rig=(\w+) out=\[([-\d ]*)\]")
LOSS = re.compile(r"\[BBB-M2\] loss before=(\d+) after=(\d+) returned=(-?\d+)")
RETURNED = re.compile(r"\[BBB\] torn down cluster \d+, returned (\d+) items")
TOOKBACK = re.compile(r"\[BBB\] cluster \d+ took back (\d+) items")
COMPILED = re.compile(
    r"\[BBB\] compiled cluster (\d+) (\d+)->(\d+) over (\d+) ports, (\d+) entities"
)
TIMING = re.compile(r"\[BBB-M2\] timing (.+?) Duration: ([\d.]+)ms")
LANE = re.compile(r"\[BBB-M2\] lane t=(\d+) out=(\d+) left=(\d+) right=(\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=\d+ nets=(\d+) drift=(\d+) unbuilt=(\d+)")

T0, T1 = 1800, 3540

# rig -> (live outputs, what the total should be in units of one saturated belt,
#         how much spread between live outputs is allowed)
#
# The spread bound is a FRACTION of the mean over the window. A physical splitter
# balancer is exact up to the granularity of an item, and each window carries
# ~1300 items per output, so 1% is loose by an order of magnitude and still
# catches anything structurally wrong -- a mis-wired stage shows up as 2:1, not
# as 1.005:1.
EXPECT = [
    #  rig       live  belts-in  spread
    ("sat4", 4, 4.0, 0.01),
    ("sat8", 8, 8.0, 0.01),
    # 3 inputs spread over 5 outputs: each gets 3/5 of a belt, and the point is
    # that P=8 with loopbacks on the spare ports loses none of it.
    ("a3to5", 5, 3.0, 0.01),
    # 4 inputs into 1 output: the output is the bottleneck, so it runs full and
    # the three spare ports dead-end without disturbing it.
    ("a4to1", 1, 1.0, 0.01),
    # ONE input feeding four outputs. The case that kills chest designs --
    # Techrocket9 measured one output draining >9,000 items while its peers got
    # ~80 on the loaders-into-a-shared-chest approach.
    ("starve", 4, 1.0, 0.02),
    # Three live outputs; the fourth has nowhere to go and must neither drag the
    # others down nor unbalance them.
    ("block", 3, 3.0, 0.01),
    # Three inputs until tick 900, four after. The window is entirely after the
    # recompile, so this asks: did the rebuilt network come back at full rate?
    ("regrow", 4, 4.0, 0.01),
    # Parts on one surface, network on another, feed and drain on the first.
    ("xsurf", 4, 4.0, 0.01),

    # --- the shape band ----------------------------------------------------
    # The smallest square shape that is not a power of two: P=4 with one
    # loopback. plan.PropagateLoop puts every row at S/max(N,M) = 1 belt.
    ("sq3", 3, 3.0, 0.01),
    # 2 in, 3 out: P=4, one loopback, and every output gets 2/3 of a belt.
    ("a2to3", 3, 2.0, 0.01),
    # 5 in, 3 out: P=8 with three loopbacks and TWO DEAD-ENDED output ports.
    # No linear model reaches this one -- a dead end backs up, blocks its
    # splitter's other side and re-routes the flow, which is a saturation
    # nonlinearity. Three outputs, one full belt each: the three real ports are
    # the only way out, so they are the bottleneck and they run full.
    ("a5to3", 3, 3.0, 0.01),
    # 9 in, 9 out: P=16, four butterfly stages and THREE jumper blocks. Nothing
    # in any suite had ever gone past P=8.
    ("n9m9", 9, 9.0, 0.01),
    # A literal feedback loop: the third output belt curls through the world and
    # comes back into the north face, so the machine is 3->3 and one in and one
    # out are the same belt. In steady state the loop carries L, every output
    # carries (2+L)/3 and the loop's output is its input, so L=1 and each real
    # output is exactly one belt. A network that jams instead of settling reads
    # as a rate collapse here.
    ("fdbk", 2, 2.0, 0.01),

    # --- the edge-type band ------------------------------------------------
    # classifySide keys on the entity's `type`, and until these three rigs
    # existed only "transport-belt" had ever been exercised in a real Factorio.
    # Undergrounds: an output half against the part on the way in, an input half
    # against it on the way out -- both arms of the belt_to_ground_type branch.
    ("uio", 2, 2.0, 0.01),
    # Vanilla express splitters spanning both parts, in and out. A splitter is
    # two tiles wide and each half is its own edge; that was a comment in
    # classifySide and nothing else.
    ("spio", 2, 2.0, 0.01),
    # Loaders directly against the part -- and the first 1->1 network (P=1, no
    # stages at all, five entities) any suite has ever run items through.
    ("lio", 1, 1.0, 0.01),
    # LANE SPLITTERS against the part, and this rig is its own red proof: it was
    # written and run BEFORE the guest could see one, and it delivered **0 0**
    # while every other rig here was green and the audit reported
    # `21 clusters, 20 networks, drift=0 unbuilt=0`. That is the shape the whole
    # band is about -- a belt-connectable family the classifier does not name is
    # SILENTLY not an edge, and a fingerprint over an empty edge list matches
    # the world perfectly, so nothing but a rate can see it.
    #
    # It also took TWO changes, not the one that was obvious. See the note on
    # the audit block at the bottom.
    ("lsio", 2, 2.0, 0.01),
]


def read(paths):
    samples, lines = {}, []
    for path in paths:
        with open(path, errors="replace") as f:
            for raw in f:
                lines.append(raw)
                m = SAMPLE.search(raw)
                if m:
                    counts = [int(v) for v in m.group(3).split()]
                    samples.setdefault(m.group(2), {})[int(m.group(1))] = counts
    return samples, lines


def main():
    samples, lines = read(sys.argv[1:])
    fail = []

    if "ctrl" not in samples or T0 not in samples["ctrl"] or T1 not in samples["ctrl"]:
        print("the control belt never reported -- did the test mod run at all?")
        return 1

    belt = samples["ctrl"][T1][0] - samples["ctrl"][T0][0]
    if belt <= 0:
        print("the control belt delivered %d items -- nothing is moving" % belt)
        return 1
    print("one saturated express belt over t=%d..%d: %d items\n" % (T0, T1, belt))

    hdr = "%-8s %-5s %-32s %7s %8s %7s" % (
        "rig", "live", "per-output delta", "total", "vs belt", "spread")
    print(hdr)
    print("-" * len(hdr))

    for rig, live, mult, tol in EXPECT:
        if rig not in samples or T0 not in samples[rig] or T1 not in samples[rig]:
            fail.append("%s: never reported" % rig)
            continue
        a, b = samples[rig][T0], samples[rig][T1]
        deltas = [b[i] - a[i] for i in range(len(b)) if a[i] >= 0 and b[i] >= 0]
        if len(deltas) != live:
            fail.append("%s: %d live outputs, expected %d" % (rig, len(deltas), live))
            continue
        total = sum(deltas)
        mean = total / float(live)
        spread = (max(deltas) - min(deltas)) / mean if mean else 1.0
        ratio = total / float(belt)
        shown = " ".join(str(d) for d in deltas)
        if len(shown) > 32:
            shown = shown[:29] + "..."
        print("%-8s %-5d %-32s %7d %7.3fx %6.2f%%"
              % (rig, live, shown, total, ratio, spread * 100))

        if spread > tol:
            fail.append(
                "%s: outputs spread %.2f%% (%s), over the %.0f%% bound -- the "
                "network is not balancing" % (rig, spread * 100, deltas, tol * 100)
            )
        # Throughput. A balancer that balances perfectly at half rate is still
        # broken, and that is the shape a hidden bottleneck takes.
        if ratio < mult * 0.98:
            fail.append(
                "%s: delivered %.3f belts, expected %.1f -- something in the "
                "network is throttling" % (rig, ratio, mult)
            )
        if ratio > mult * 1.02:
            fail.append(
                "%s: delivered %.3f belts, expected %.1f -- more came out than "
                "went in" % (rig, ratio, mult))

    # --- the rigs whose outputs are not uniform ------------------------------
    #
    # EXPECT's tuple says "every live output should be the same"; these three
    # are interesting precisely because they should not be, so each states its
    # own arithmetic. NOTHING here is allowed to be vague: a bound that would
    # pass on a network that did the wrong thing is worse than no bound.

    def deltas(rig):
        if rig not in samples or T0 not in samples[rig] or T1 not in samples[rig]:
            fail.append("%s: never reported" % rig)
            return None
        a, b = samples[rig][T0], samples[rig][T1]
        return [b[i] - a[i] for i in range(len(b))]

    # tslow: 4 in, 4 out, but the fourth OUTPUT ROW is a normal-tier belt --
    # exactly a third of express, and the sink loader behind it is still
    # express, so the belt is the only limiter. This is a RATE-LIMITED port
    # rather than a dead one, and the claim is that it neither drags the other
    # three down nor unbalances them: a balancer that equalised on the slow
    # port's rate would deliver 4/3 belts in total instead of 10/3.
    d = deltas("tslow")
    if d is not None:
        fast, slow = d[:3], d[3]
        fmean = sum(fast) / 3.0
        fspread = (max(fast) - min(fast)) / fmean if fmean else 1.0
        print("tslow    3 express outputs %s (%.3fx belt each, spread %.2f%%), "
              "1 normal-tier output %d (%.3fx belt)"
              % (fast, fmean / belt, fspread * 100, slow, slow / float(belt)))
        if fspread > 0.01:
            fail.append("tslow: the three express outputs spread %.2f%% (%s) -- a "
                        "rate-limited fourth port must not unbalance the rest"
                        % (fspread * 100, fast))
        if not 0.98 <= fmean / belt <= 1.02:
            fail.append("tslow: each express output delivered %.3f belts, expected "
                        "1.0 -- the balancer throttled itself to the slow port"
                        % (fmean / belt))
        # A normal belt is 0.03125 tiles/tick against express's 0.09375.
        if not 0.98 / 3 <= slow / float(belt) <= 1.02 / 3:
            fail.append("tslow: the normal-tier output delivered %.3f belts, "
                        "expected 1/3 -- that port is not running at its own "
                        "belt's rate" % (slow / float(belt)))

    # pass: a working 2->2 with a belt line running east along the top part's
    # NORTH face. From that face `dir` is north and `back` is south, so an
    # east-facing belt is neither and falls through classifySide: not an edge.
    # The two claims are that the balancer is unaffected and that the passing
    # line loses nothing to it.
    d = deltas("pass")
    if d is not None:
        bal, thru = d[:2], d[2]
        bmean = sum(bal) / 2.0
        bspread = (max(bal) - min(bal)) / bmean if bmean else 1.0
        print("pass     balancer outputs %s (%.3fx belt total, spread %.2f%%), "
              "the line going PAST it %d (%.3fx belt)"
              % (bal, sum(bal) / float(belt), bspread * 100, thru, thru / float(belt)))
        if bspread > 0.01:
            fail.append("pass: balancer outputs spread %.2f%% (%s)"
                        % (bspread * 100, bal))
        if not 1.96 <= sum(bal) / float(belt) <= 2.04:
            fail.append("pass: the balancer delivered %.3f belts, expected 2.0 -- "
                        "a belt merely going past its north face changed it"
                        % (sum(bal) / float(belt)))
        if not 0.98 <= thru / float(belt) <= 1.02:
            fail.append("pass: the line running PAST the cluster delivered %.3f "
                        "belts, expected 1.0 -- the balancer classified it as an "
                        "edge and is taking items out of it" % (thru / float(belt)))

    # lane: THE ONE RIG CHEST TOTALS CANNOT JUDGE.
    #
    # Both inputs are side-loaded, so each is half a belt on ONE lane. A vanilla
    # splitter is lane-PRESERVING, so a network built without the lane-splitter
    # stage delivers all of it on one lane of every output -- at exactly the
    # same rate, into exactly the same chests. What separates the two is where
    # the items are STANDING, which is why the test mod logs per-lane occupancy.
    #
    # RED-PROVEN, and the numbers are the point. With plan.go's head section
    # building a ProtoBelt where it builds a ProtoLaneSplitter -- one token,
    # same op count, same positions -- both outputs go from left=30 right=30
    # over five samples to left=60 right=0, and these two assertions are the
    # ONLY thing in the whole suite that fails. The chests are untouched:
    # 653 653 and 1.000x belt in both arms, to the item. That is the claim.
    d = deltas("lane")
    if d is not None:
        total = sum(d)
        print("lane     side-loaded inputs (half a belt each, one lane): outputs "
              "%s, %.3fx belt total" % (d, total / float(belt)))
        if not 0.98 <= total / float(belt) <= 1.02:
            fail.append("lane: delivered %.3f belts, expected 1.0 (two inputs of "
                        "half a belt each) -- the side-loading is not producing "
                        "the feed this rig is about" % (total / float(belt)))
    lanes = {}
    for raw in lines:
        m = LANE.search(raw)
        if m:
            lanes.setdefault(int(m.group(2)), []).append(
                (int(m.group(1)), int(m.group(3)), int(m.group(4))))
    if not lanes:
        fail.append("lane: no per-lane sample was ever logged, so the lane-"
                    "fidelity claim is untested -- chest totals cannot see it")
    for out in sorted(lanes):
        rows = lanes[out]
        left = sum(l for _, l, _ in rows)
        right = sum(r for _, _, r in rows)
        both = sum(1 for _, l, r in rows if l > 0 and r > 0)
        print("         output %d over %d samples: left=%d right=%d, both lanes "
              "occupied on %d of them" % (out, len(rows), left, right, both))
        if left == 0 or right == 0:
            fail.append(
                "lane: output %d had %d items on its left lane and %d on its "
                "right over %d samples. A one-lane feed came out on one lane: "
                "the lane-splitter stage is not doing its job, and no chest "
                "total in this suite can see it" % (out, left, right, len(rows)))
            continue
        if both < len(rows) - 1:
            fail.append(
                "lane: output %d had both lanes occupied on only %d of %d "
                "samples -- the lanes are not being kept fed" % (out, both, len(rows)))
        minority = min(left, right) / float(left + right)
        if minority < 0.2:
            fail.append(
                "lane: output %d is %.1f%% on one lane (left=%d right=%d). The "
                "lane splitter should halve a one-lane feed, not merely leak "
                "onto the other lane" % (out, 100 * (1 - minority), left, right))

    # --- item conservation across a recompile -------------------------------
    # The teardown that matters is the one the check itself provoked, which is
    # the last one logged before the check reported. Other rigs tear down during
    # the same run and their numbers are not this rig's.
    loss, returned, last = None, None, None
    took, lasttook = None, None
    for raw in lines:
        m = RETURNED.search(raw)
        if m:
            last = int(m.group(1))
            continue
        m = TOOKBACK.search(raw)
        if m:
            lasttook = int(m.group(1))
            continue
        m = LOSS.search(raw)
        if m:
            loss = (int(m.group(1)), int(m.group(2)), int(m.group(3)))
            returned, took = last, lasttook
    print()
    if loss is None:
        fail.append("the item-conservation check never ran")
    else:
        before, after, delta = loss
        print("recompile of a FULL network, both surfaces counted, no tick "
              "elapsed: %d items before, %d after (%+d); the teardown drained %s "
              "out of the hidden network and the rebuild put %s back INSIDE it"
              % (before, after, delta,
                 returned if returned is not None else "nothing",
                 took if took is not None else "nothing"))
        if returned in (None, 0):
            fail.append("the teardown drained nothing, so this proves nothing")
        elif took in (None, 0):
            # A recompile is not a removal (guest/go/carry.go). The items came
            # out of a network whose cluster is still standing, so they belong
            # in the one that replaced it -- not beside it on the floor.
            fail.append(
                "the teardown drained %d items out of a full network and the "
                "rebuild took none of them back: a recompile must reinsert"
                % returned)
        elif delta != 0:
            fail.append(
                "%d items went missing across a recompile of a full network "
                "(%d before, %d after, both surfaces counted, no tick elapsed). "
                "The guest drained %d out of the network; whatever it could not "
                "put back is a player-facing bug the incumbent does not have."
                % (-delta, before, after, returned)
            )

    # --- the networks that were built ---------------------------------------
    built = {}
    for raw in lines:
        m = COMPILED.search(raw)
        if m:
            built["%s->%s" % (m.group(2), m.group(3))] = (int(m.group(4)), int(m.group(5)))
    for shape, want_ports in (("4->4", 4), ("8->8", 8), ("3->5", 8), ("4->1", 4),
                              # P = next_pow2(max(N, M)) for each new shape, so
                              # a rig that quietly compiled to the wrong size --
                              # an edge missed, an edge invented -- fails here
                              # before its throughput is even looked at.
                              ("3->3", 4), ("2->3", 4), ("5->3", 8),
                              ("9->9", 16), ("1->1", 1)):
        if shape not in built:
            fail.append("no %s network was ever compiled" % shape)
        elif built[shape][0] != want_ports:
            fail.append("%s was compiled over %d ports, expected %d"
                        % (shape, built[shape][0], want_ports))
    if built:
        print("\nnetworks compiled:")
        for k in sorted(built):
            print("  %-6s %3d entities over %2d ports" % (k, built[k][1], built[k][0]))

    # --- the final audit ------------------------------------------------------
    # Nothing has touched the world since tick 900, so the registry and the
    # world must agree exactly. This is where `pass` is really decided: a
    # classifier that read the belt going past a cluster's north face as an edge
    # would have a fingerprint the world does not match, and would say so here.
    audits = [tuple(int(g) for g in m.groups())
              for m in (AUDIT.search(l) for l in lines) if m]
    if not audits:
        fail.append("no audit was ever logged, so nothing checked the registry "
                    "against the world")
    else:
        clusters, nets, drift, unbuilt = audits[-1]
        print("\nfinal audit: %d clusters, %d networks, drift=%d, unbuilt=%d"
              % (clusters, nets, drift, unbuilt))
        if drift or unbuilt:
            fail.append("the final audit found drift=%d unbuilt=%d over %d "
                        "clusters, on a world nothing has touched since tick 900"
                        % (drift, unbuilt, clusters))
        # Every cluster in this save has belts on both sides of it, so every one
        # of them must have compiled to a network. A cluster with no network is
        # a cluster whose edges the classifier did not recognise -- which is
        # exactly what `lsio` was before the lane-splitter case existed, and
        # this is the ONLY line in the suite that would have said so without a
        # throughput number: 21 clusters, 20 networks, drift=0, unbuilt=0.
        #
        # THE EDGE CLASSIFIER HAS TWO GATES AND ONLY ONE OF THEM IS OBVIOUS.
        # `classifySide`'s switch is the second; the first is the `type` array
        # `find_entities_filtered` carries, applied by the engine in C++ before
        # the guest sees anything. Adding "lane-splitter" to the switch alone
        # moved NOTHING -- lsio still read 0 0 and the audit still read 21/20 --
        # because the entity was never returned to be switched on. Both lists
        # are in guest/go/compile.go and nothing but this rig makes them agree.
        if nets != clusters:
            fail.append("%d of %d clusters have no network at all: something "
                        "adjacent to a balancer is not being classified as an "
                        "edge" % (clusters - nets, clusters))

    times = [(m.group(1), float(m.group(2)))
             for m in (TIMING.search(l) for l in lines) if m]
    if times:
        print("\ncompile cost, wall clock (helpers.create_profiler):")
        for what, ms in times:
            print("  %-52s %8.3f ms" % (what, ms))

    if fail:
        print("\nM2 NETWORK ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        return 1
    print("\nM2 network assertions passed (%d uniform-output rigs, plus tslow, "
          "pass and lane)" % len(EXPECT))
    return 0


if __name__ == "__main__":
    sys.exit(main())
