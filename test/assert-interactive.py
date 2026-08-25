#!/usr/bin/env python3
"""Assert that the interactive checklist's staged world builds clean.

`test/interactive/bbb-interactive-setup` stages the five player-gesture rigs and
the five mod-portal demo scenes on a fresh world. Nothing headless can make the
gestures -- that is the whole reason the mod exists -- but everything it STAGES
is ordinary world-building, and a rig that no longer lands, or that is itself
refused, is a checklist that wastes a human's session.

So this asserts the staging and nothing about the gestures:

  EVERY PIECE LANDED.   The staging mod routes every create_entity through one
                        helper that logs a line when the engine returns nil, so
                        a rig knocked off its tiles by a geometry edit says so.
  EVERY RIG IS LEGAL.   Not one refusal of either bound, and not one compile
                        error. The gestures are what create the refusals; a
                        refusal at staging time means a rig was built to a shape
                        this mod cannot compile, and the human would arrive to
                        find it already stopped.
  EVERY RIG COMPILED.   The exact multiset of `compiled cluster N->M over P
                        ports` lines, so a rig whose belts landed somewhere
                        other than where the geometry intended is caught by its
                        SHAPE rather than by a count that several shapes share.
  THE WORLD IS THE ONE THE CHECKLIST DESCRIBES.  Cluster and part counts off the
                        audit, against the numbers written down here rather than
                        against the guest's own -- both audits, because the first
                        reports the registry before the compiling its own
                        dispatch goes on to do and the second reports it after.

    python3 test/assert-interactive.py create.log
"""

import re
import sys

PLACEFAIL = re.compile(r"\[BBB-INTERACTIVE\] could not place (\S+) at \(([-\d]+),([-\d]+)\)")
STAGED = re.compile(r"\[BBB-INTERACTIVE\] gestures staged: ")
PREPPED = re.compile(r"\[BBB-INTERACTIVE\] surface prepped: clouds off, daytime frozen at noon")
DEMO = re.compile(r"\[BBB-INTERACTIVE\] demo scenes staged ")
AUDITED = re.compile(r"\[BBB-INTERACTIVE\] audited (\S+)")
COMPILED = re.compile(
    r"\[BBB\] compiled cluster (\d+) (\d+)->(\d+) over (\d+) ports, (\d+) entities"
)
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)"
)
OVERLIMIT = re.compile(r"\[BBB\] alert: cluster (\d+) would need (\d+) ports")
MULTIEDGE = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt"
)
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")

# The world the checklist describes, band by band. `(N, M, P)` per cluster, and
# the total part count, both derived from the geometry in control.lua rather
# than read back off the guest -- a fill that fused two bands would move the
# guest's own numbers together and neither would say anything.
#
#   pocket        4 -> 4 over eight parts, dead-ended
#   edge          2 -> 2 over four parts, plus one edgeless part
#   limit         64 -> 1 over sixty-four input parts, one output part and one
#                 edgeless part: P = 64, the limit exactly
#   bridge        two of 32 -> 1, thirty-three parts each
#   fast replace  a 2 -> 2 over four parts and a 1 -> 1 over five
#   demo          cross 1 -> 3, compact 8 -> 8, c-shape 8 -> 8, c-shape 8 -> 9,
#                 long run 8 -> 8
SHAPES = sorted([
    (4, 4, 4),    # A, pocket
    (2, 2, 2),    # B, the belt at the edge
    (64, 1, 64),  # C, the sixty-fifth belt
    (32, 1, 32),  # D, block A
    (32, 1, 32),  # D, block B
    (2, 2, 2),    # E, fast replace forward
    (1, 1, 1),    # E, fast replace reverse
    (1, 3, 4),    # demo, cross
    (8, 8, 8),    # demo, compact column
    (8, 8, 8),    # demo, c-shape
    (8, 9, 16),   # demo, c-shape express
    (8, 8, 8),    # demo, long run
])
PARTS = 228
CLUSTERS = len(SHAPES)


def main():
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    lines = open(sys.argv[1], encoding="utf-8", errors="replace").read().splitlines()
    fail = []

    print("the interactive checklist's staged world")

    # --- every piece landed ---------------------------------------------------
    missed = [PLACEFAIL.search(l) for l in lines]
    missed = [m for m in missed if m]
    print("  placements the engine refused: %d" % len(missed))
    for m in missed[:10]:
        fail.append("%s did not land at (%s,%s)" % (m.group(1), m.group(2), m.group(3)))
    if len(missed) > 10:
        fail.append("...and %d more placements that did not land" % (len(missed) - 10))

    # The staging mod's own bookend lines, so a run in which on_init died half
    # way through fails HERE rather than on a count that happens to add up.
    if not any(STAGED.search(l) for l in lines):
        fail.append("the staging mod never reported its gesture bands: on_init "
                    "did not finish")
    if not any(DEMO.search(l) for l in lines):
        fail.append("the staging mod never reported its demo scenes: on_init "
                    "did not finish")
    if not any(PREPPED.search(l) for l in lines):
        fail.append("the staging mod never staged the capture conditions: a "
                    "looping GIF needs clouds off and the daytime frozen, and "
                    "the prep line is missing")

    # --- every rig is legal ---------------------------------------------------
    over = [l for l in lines if OVERLIMIT.search(l)]
    multi = [l for l in lines if MULTIEDGE.search(l)]
    spills = [SPILLED.search(l) for l in lines]
    spills = [m for m in spills if m]
    print("  refusals: %d over the port limit, %d over one belt per part"
          % (len(over), len(multi)))
    print("  spills: %d" % len(spills))
    if over:
        fail.append("%d cluster(s) were refused for the port limit while STAGING; "
                    "the gestures create the refusals and the rigs must not"
                    % len(over))
    if multi:
        fail.append("%d cluster(s) were refused for carrying more than one belt "
                    "per part while STAGING; a rig is built to a shape this "
                    "Factorio cannot compile" % len(multi))
    if spills:
        fail.append("%d spill(s) during staging; nothing here is ever torn down"
                    % len(spills))

    # --- every rig compiled, and to the shape the geometry intended -----------
    got = sorted((int(m.group(2)), int(m.group(3)), int(m.group(4)))
                 for m in (COMPILED.search(l) for l in lines) if m)
    print("  compiled: %s" % ", ".join("%d->%d/P%d" % s for s in got))
    if got != SHAPES:
        fail.append("the staged world compiled %s and the checklist's geometry "
                    "is %s" % (["%d->%d/P%d" % s for s in got],
                               ["%d->%d/P%d" % s for s in SHAPES]))

    # --- the audits -----------------------------------------------------------
    #
    # TWO OF THEM, AND THEY SAY DIFFERENT THINGS. An audit reports the registry
    # as its own dispatch finds it, and that dispatch is also what drains the
    # queue -- so the first marker sees every cluster unbuilt and the second,
    # placed behind it, sees them built. A --create never reaches a tick, so
    # there is no third way to look.
    tags = [m.group(1) for m in (AUDITED.search(l) for l in lines) if m]
    audits = [tuple(int(g) for g in m.groups())
              for m in (AUDIT.search(l) for l in lines) if m]
    print("  audits: %s -> %s" % (tags, audits))
    if tags != ["staged", "compiled"]:
        fail.append("the staging mod placed audit markers %s and should place "
                    "exactly ['staged', 'compiled']" % tags)
    if len(audits) != 2:
        fail.append("%d audit line(s) in the create log, expected 2" % len(audits))
    else:
        first, second = audits
        if (first[0], first[1]) != (CLUSTERS, PARTS):
            fail.append("the first audit found %d clusters over %d parts and the "
                        "checklist stages %d over %d"
                        % (first[0], first[1], CLUSTERS, PARTS))
        if first[4] != CLUSTERS:
            fail.append("the first audit reports %d unbuilt and every one of the "
                        "%d clusters should still be waiting for the drain its "
                        "own dispatch runs" % (first[4], CLUSTERS))
        if second != (CLUSTERS, PARTS, CLUSTERS, 0, 0, 0):
            fail.append("the second audit reads clusters=%d parts=%d nets=%d "
                        "drift=%d unbuilt=%d refused=%d and should read "
                        "%d/%d/%d/0/0/0"
                        % (second + (CLUSTERS, PARTS, CLUSTERS)))

    if fail:
        print("\nSTAGING ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\nthe checklist's world stages clean: %d balancers over %d parts, "
          "every one legal under one belt per part, none refused"
          % (CLUSTERS, PARTS))


if __name__ == "__main__":
    main()
