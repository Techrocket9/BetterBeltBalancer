#!/usr/bin/env python3
"""Assert that the lane-keeping part never mixes a belt's two lanes.

Every rig is fed one express belt carrying iron plate on its LEFT lane and
copper plate on its RIGHT, and the observer reads both lanes of a tile on every
output every eight ticks, in three windows: before tick 600, the window a
recompile under load lands in, and after. What is asserted:

  * the FEED really is split: on every feed probe the left lane carries iron
    and no copper, the right copper and no iron. Without this every clean
    output below would be vacuous.
  * the ORDINARY control `o12` does not keep lanes: copper reaches the left
    lane of its outputs in every window. (Measured: on a saturated split feed
    an ordinary balancer swaps the two lanes outright.)
  * every LANE-KEEPING output, in every window, carries no copper on its left
    lane and no iron on its right. That covers a plain 1->2, a 2->2, a 1->2
    with a priority output (the concentrator splitters), and the window after
    a recompile of a running (`l12`) and a jammed (`ljam`) network, where the
    carry has to put each item back on the lane it came off.
  * the priority output takes the whole flow and its sibling none.
  * the lane outputs balance: equal iron and copper per sink, and the 2->2's
    two outputs equal.
  * a lane-keeping part beside an ordinary one stays a separate balancer, so
    `ladj` delivers nothing; the all-ordinary `oadj` beside it delivers.
  * the two kinds replace each other: `frep`'s two ordinary parts are
    script-fast-replaced by lane parts and back, and its sink keeps filling
    after each swap. The final audit is clean.
  * the benchmark phase is an UPGRADE: the rebuild adopts every lane-keeping
    network and sweeps the one the observer orphaned (a lane-keeping 1->1 is
    three hidden entities), which a sweep that looked for lane splitters could
    not see.

Red proof, 2026-11-10: with drain()'s lane tally switched off the carry is
lane-blind, and the window after the recompile shows copper on the left lane of
both `l12-west` (3) and `ljam-west` (7). With it on, zero.
"""

import re
import sys

LINE = re.compile(r"\[BBB-LANE\] (.*)$")
LANES = re.compile(r"lanes w=(\d) probe=(\S+) L=(\d+)/(\d+) R=(\d+)/(\d+)")
SINK = re.compile(r"sink rig=(\S+) iron=(\d+) copper=(\d+)")
TOOK = re.compile(r"\[BBB\] cluster \d+ took back (\d+) items")
SWEPT = re.compile(r"\[BBB\] swept (\d+) orphaned hidden entities")
FREP = re.compile(r"frep t=(\S+) tick=\d+ lane=(\d+) ordinary=(\d+) sink=(\d+)")
AUDIT = re.compile(r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
                   r"unbuilt=(\d+)")
REBUILT = re.compile(
    r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters "
    r"\((\d+) networks adopted, (\d+) rebuilt\)")


def read(path):
    with open(path, encoding="utf-8", errors="replace") as f:
        return f.read().splitlines()


def main(create_log, run_log):
    create, run = read(create_log), read(run_log)
    fail = []

    obs_create = [m.group(1) for l in create for m in [LINE.search(l)] if m]
    if "init complete" not in obs_create:
        fail.append("the observer never finished building its world")
    if "orphaned rig=lorph" not in obs_create:
        fail.append("lorph was never orphaned, so the sweep below proves nothing")

    lanes, sinks, flag = {}, {}, None
    for l in run:
        m = LANES.search(l)
        if m:
            w, probe = int(m.group(1)), m.group(2)
            lanes[(w, probe)] = tuple(int(m.group(i)) for i in range(3, 7))
            continue
        m = SINK.search(l)
        if m:
            sinks[m.group(1)] = (int(m.group(2)), int(m.group(3)))
            continue
        if "flag rig=lprio" in l:
            flag = l.endswith("accepted=true")

    probes = sorted({p for _, p in lanes})
    if not probes:
        fail.append("no lane sample was reported")

    def windows(probe):
        return [w for w in range(3) if (w, probe) in lanes]

    # The feed.
    for probe in [p for p in probes if "-feed" in p]:
        for w in windows(probe):
            li, lc, ri, rc = lanes[(w, probe)]
            if lc or ri or not li or not rc:
                fail.append("%s w=%d: the feed is not split (L iron/copper %d/%d, "
                            "R %d/%d)" % (probe, w, li, lc, ri, rc))

    # The ordinary control mixes.
    for w in range(3):
        mixed = sum(lanes.get((w, p), (0, 0, 0, 0))[1] for p in probes
                    if p.startswith("o12-out"))
        if not mixed:
            fail.append("o12 w=%d: the ordinary control kept its lanes, so the "
                        "lane rigs prove nothing" % w)

    # The lane-keeping outputs are clean, and the ones that should flow do.
    lane_rigs = ("l12-", "lprio-", "l22-", "ljam-")
    quiet = {"lprio-out@29"}
    for probe in [p for p in probes if p.startswith(lane_rigs) and "-feed" not in p]:
        ws = windows(probe)
        if not ws:
            fail.append("%s: never sampled" % probe)
        for w in ws:
            li, lc, ri, rc = lanes[(w, probe)]
            if lc or ri:
                fail.append("%s w=%d: lanes mixed -- copper on the left %d, iron "
                            "on the right %d" % (probe, w, lc, ri))
            if probe not in quiet and (not li or not rc):
                fail.append("%s w=%d: nothing flowed (L iron %d, R copper %d)"
                            % (probe, w, li, rc))
    for need in ("l12-west@15", "ljam-west@75"):
        if 1 not in windows(need):
            fail.append("%s: the recompile window was never sampled" % need)

    # Priority.
    if flag is not True:
        fail.append("the priority flag on lprio was not accepted")
    hi, lo = sinks.get("lprio-out@28", (0, 0)), sinks.get("lprio-out@29", (-1, -1))
    if not hi[0] or not hi[1] or lo != (0, 0):
        fail.append("lprio: the priority output got %s and its sibling %s" % (hi, lo))

    # Balance.
    for name, (iron, copper) in sorted(sinks.items()):
        if name.startswith(lane_rigs) and abs(iron - copper) > 4:
            fail.append("%s: %d iron against %d copper from one split feed"
                        % (name, iron, copper))
    a, b = sinks.get("l22-out@44", (0, 0)), sinks.get("l22-out@46", (0, 0))
    if not a[0] or abs(a[0] - b[0]) > max(4, a[0] // 50):
        fail.append("l22: the two outputs delivered %s and %s" % (a, b))
    a, b = sinks.get("l12-out@14", (0, 0)), sinks.get("l12-out@15", (0, 0))
    if not a[0] or abs(a[0] - b[0]) > max(4, a[0] // 50):
        fail.append("l12: the two outputs delivered %s and %s" % (a, b))

    # Two kinds never join.
    if sinks.get("ladj") != (0, 0):
        fail.append("ladj: a lane part beside an ordinary one delivered %s, so "
                    "they joined" % (sinks.get("ladj"),))
    if not sinks.get("oadj", (0, 0))[0]:
        fail.append("oadj: two ordinary parts delivered nothing, so ladj's "
                    "silence proves nothing")

    # The carry moved something on both recompiles, and a jammed network's
    # worth on one of them.
    took = sorted(int(m.group(1)) for l in run for m in [TOOK.search(l)] if m)
    if len(took) < 2 or took[-1] < 40:
        fail.append("the recompiles carried %s items; ljam's jammed network "
                    "should carry forty or more" % took)

    # The two kinds replace each other.
    frep = {m.group(1): tuple(int(m.group(i)) for i in range(2, 5))
            for l in run for m in [FREP.search(l)] if m}
    want = {"before": (0, 2), "swap-bbb-lane-balancer-part": (2, 0), "lane": (2, 0),
            "swap-bbb-balancer-part": (0, 2), "end": (0, 2)}
    for tag, kinds in want.items():
        if tag not in frep:
            fail.append("frep: no %s report" % tag)
        elif frep[tag][:2] != kinds:
            fail.append("frep %s: %d lane and %d ordinary parts standing, want %s"
                        % (tag, frep[tag][0], frep[tag][1], kinds))
    if all(t in frep for t in want):
        a, b, c, d = (frep[t][2] for t in ("before", "swap-bbb-lane-balancer-part",
                                          "lane", "end"))
        if not (0 < a and c > b + 100 and d > frep["swap-bbb-balancer-part"][2] + 100):
            fail.append("frep: the sink read %d, %d, %d, %d across the swaps; it must "
                        "keep filling after each" % (a, b, c, d))
    audits = [m for l in run for m in [AUDIT.search(l)] if m]
    if not audits:
        fail.append("no final audit")
    elif audits[-1].group(4) != "0" or audits[-1].group(5) != "0":
        fail.append("the final audit reads drift=%s unbuilt=%s"
                    % (audits[-1].group(4), audits[-1].group(5)))

    # The upgrade.
    swept = [int(m.group(1)) for l in run for m in [SWEPT.search(l)] if m]
    if swept != [3]:
        fail.append("the rebuild swept %s orphaned entities; lorph's lane-keeping "
                    "1->1 is three" % swept)
    rb = [m for l in run for m in [REBUILT.search(l)] if m]
    if len(rb) != 1:
        fail.append("expected one rebuild-from-world in the benchmark, saw %d" % len(rb))
    else:
        adopted, rebuilt = int(rb[0].group(4)), int(rb[0].group(5))
        # Seven standing networks (o12, l12, lprio, l22, oadj, ljam, frep) and
        # ladj's two one-sided clusters, which have none to adopt.
        if (adopted, rebuilt) != (7, 2):
            fail.append("the rebuild adopted %d and rebuilt %d; seven networks "
                        "stand and two clusters have none" % (adopted, rebuilt))

    if fail:
        print("lane: FAIL")
        for f in fail:
            print("  " + f)
        return 1
    print("lane: ok -- %d probes clean across three windows, carry %s, "
          "orphan swept, %d sinks balanced" % (len(probes), took, len(sinks)))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1], sys.argv[2]))
