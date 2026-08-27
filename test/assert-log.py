#!/usr/bin/env python3
"""Assert the M1 cluster registry from Factorio's own log.

The guest logs one `[BBB] state clusters=N parts=N sizes=...` line after every
event that changed the registry, and guest/go/obs/m1 marks each phase with
`[BBB-TEST] phase=N`. This walks the log in order and checks the state at the
end of each phase against what the pattern the test built must produce.

The state line is the whole assertion surface on purpose: cluster COUNT and
every cluster SIZE, sorted, is exactly what "parts merge and split correctly"
means. The per-transition lines (merge/split/dissolved) are checked too, but as
a second opinion -- a registry that reached the right sizes by the wrong route
is still wrong.

    python3 test/assert-log.py create.log run.log
"""

import re
import sys

# phase -> (what the state must be at the end of it, transition-line demands)
#
# The sizes are derived by hand from the patterns in the test mod's control.lua,
# which is the point: nothing in this file re-implements the flood fill.
#
#   A: single(1)  line(4)  L(5)  bridged(5)      B: single(1)  pair(2)
EXPECT = [
    (
        1,
        "clusters=6 parts=18 sizes=1,1,2,4,5,5",
        # 7 clusters are born as a lone part; 12 placements join something.
        {"formed": 7, "merge": 12, "split": 0, "dissolved": 0},
    ),
    (
        2,
        # the 5-part bridged cluster comes apart into 2 and 2
        "clusters=7 parts=17 sizes=1,1,2,2,2,4,5",
        {"formed": 0, "merge": 0, "split": 1, "dissolved": 0},
    ),
    (
        3,
        # the 4-part line loses its second tile: 1 and 2
        "clusters=8 parts=16 sizes=1,1,1,2,2,2,2,5",
        {"formed": 0, "merge": 0, "split": 1, "dissolved": 0},
    ),
    (
        4,
        # the lone part goes; its cluster stops existing
        "clusters=7 parts=15 sizes=1,1,2,2,2,2,5",
        {"formed": 0, "merge": 0, "split": 0, "dissolved": 1},
    ),
    (
        5,
        # the L loses its corner: two arms of 2. A corner is the shape a
        # neighbour-count heuristic gets wrong -- the removed tile had two
        # neighbours, as the line's middle did, but they are not collinear.
        "clusters=8 parts=14 sizes=1,1,2,2,2,2,2,2",
        {"formed": 0, "merge": 0, "split": 1, "dissolved": 0},
    ),
    (
        6,
        # surface B's pair loses one part to entity.die(): no split, just a
        # smaller cluster -- and the only phase that arrives via on_entity_died.
        "clusters=8 parts=13 sizes=1,1,1,2,2,2,2,2",
        {"formed": 0, "merge": 0, "split": 0, "dissolved": 0},
    ),
]

# M5, phases 7-9: the adaptive sprite.
#
# The guest logs `[BBB] skin cluster=R parts=N set=K vars=v,v,...` for every
# cluster whose SHAPE changed -- the variation it put on each part, in (y, x)
# order. The numbers are the ones guest/go/skin/skin_test.go derives in pure Go
# for the same five shapes, and they are written here as literals on purpose:
# a test that recomputed them would be a second copy of the thing under test.
#
# `set` is the second claim, and the cheaper one to break: it is how many parts
# the guest actually made a host call for. Growing a 4-part line by one tile
# moves exactly two pictures, and a guest that re-set all five would still draw
# the right thing while doing work that does not scale.
SKIN_EXPECT = {
    7: [
        ("4", "4", "3,11,11,9"),                                # line
        ("5", "5", "5,6,4,11,9"),                               # L
        ("5", "5", "5,3,16,9,2"),                               # plus
        ("4", "4", "21,27,17,35"),                              # 2x2 block
        ("12", "12", "7,11,11,13,6,6,6,6,4,11,11,10"),          # donut
    ],
    8: [("5", "2", "3,11,11,11,9")],
    9: [("1", "1", "1")] * 4,
}

LINE = re.compile(r"\[(BBB|BBB-TEST)\] (.*)$")
PHASE = re.compile(r"^phase=(\d+)")
STATE = re.compile(r"^state (clusters=\d+ parts=\d+ sizes=[\d,]*)$")
SKIN = re.compile(r"^skin cluster=\d+ parts=(\d+) set=(\d+) vars=([\d,]*)$")
DISSOLVED = re.compile(r"^cluster \d+ dissolved(?:, mined by player \d+)?$")


def phases(paths):
    """Group every [BBB]/[BBB-TEST] line by the phase it falls in."""
    out = {}
    cur = None
    for path in paths:
        with open(path, errors="replace") as f:
            for raw in f:
                m = LINE.search(raw.rstrip("\n"))
                if not m:
                    continue
                body = m.group(2)
                p = PHASE.match(body)
                if p:
                    cur = int(p.group(1))
                    out.setdefault(cur, [])
                    continue
                if cur is not None:
                    out[cur].append(body)
    return out


def main():
    seen = phases(sys.argv[1:])
    failures = []

    for n, want_state, want_counts in EXPECT:
        lines = seen.get(n)
        if lines is None:
            failures.append("phase %d never ran" % n)
            continue

        states = [STATE.match(l).group(1) for l in lines if STATE.match(l)]
        if not states:
            failures.append("phase %d: the guest logged no state line at all" % n)
        elif states[-1] != want_state:
            failures.append(
                "phase %d: state is %r, expected %r" % (n, states[-1], want_state)
            )

        got = {
            "formed": sum(1 for l in lines if l.startswith("cluster ") and "formed" in l),
            "merge": sum(1 for l in lines if l.startswith("merge ")),
            "split": sum(1 for l in lines if l.startswith("split ")),
            # "... dissolved", optionally followed by ", mined by player N" --
            # the suffix a PLAYER-driven dissolve carries (carry.go's
            # beneficiary). No headless run can produce one, and matching on the
            # word rather than on the end of the line is what keeps this counter
            # honest if one ever does.
            "dissolved": sum(1 for l in lines if DISSOLVED.search(l)),
        }
        for kind, want in sorted(want_counts.items()):
            if got[kind] != want:
                failures.append(
                    "phase %d: %d %s line(s), expected %d" % (n, got[kind], kind, want)
                )

    for n, want in sorted(SKIN_EXPECT.items()):
        lines = seen.get(n)
        if lines is None:
            failures.append("phase %d never ran" % n)
            continue
        got = [SKIN.match(l).groups() for l in lines if SKIN.match(l)]
        if sorted(got) != sorted(want):
            failures.append(
                "phase %d: skin lines are %r, expected %r" % (n, sorted(got), sorted(want))
            )

    for n in sorted(seen):
        states = [STATE.match(l).group(1) for l in seen[n] if STATE.match(l)]
        skins = [SKIN.match(l).group(3) for l in seen[n] if SKIN.match(l)]
        print("  phase %d: %s" % (n, states[-1] if states else "(no state line)"))
        for s in skins:
            print("            skin vars=%s" % s)

    if failures:
        print("\nM1 CLUSTER ASSERTIONS FAILED:")
        for f in failures:
            print("  " + f)
        return 1
    print("\nM1 cluster and sprite assertions passed (%d + %d phases)"
          % (len(EXPECT), len(SKIN_EXPECT)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
