#!/usr/bin/env python3
"""Assert that a mod upgrade re-derives the registry and ADOPTS what is standing.

A rebuilt guest gets a fresh heap, and it is now TOLD SO: this mod exports
`fk_migrate`, which upstream split off as a notification on a fresh heap rather
than the adopt-the-old-linear-memory hook it used to be (CLAUDE.md, "Coming back
on a heap this build did not write"). So the guest wakes up with an empty
registry and a world full of parts and running networks, and puts that right at
a named point -- on_configuration_changed, before the first tick -- rather than
inside whichever event happened to arrive first.

It has to do so WITHOUT tearing down networks that are already correct: no tick
has passed since they were built, so they are.

Two things are checked here: that the guest was told, and that the rebuild line
says what it should. Whether the adoption was CORRECT is decided immediately
afterwards by test/assert-m2.py run over the same logs -- a network adopted with
the wrong slot, or adopted when it should have been rebuilt, does not balance.

**This is also the only place a real second guest BUILD is exercised.** Until
`fk_migrate` was exported, the version bump and the build-stamp bump took the
same path to the byte, because the old heap was declined either way.

    python3 test/assert-upgrade.py create.log run.log
"""

import re
import sys

REBUILT = re.compile(
    r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters "
    r"\((\d+) networks adopted, (\d+) rebuilt\)"
)
TOLD = re.compile(r"\[BBB\] the mod was rebuilt \(guest state version (\d+)\)")
SWEPT = re.compile(r"\[BBB\] swept (\d+) orphaned hidden entities")


def main():
    create, run = [], []
    for path, into in zip(sys.argv[1:3], (create, run)):
        with open(path, errors="replace") as f:
            into.extend(f)

    fail = []
    told = next((i for i, l in enumerate(run) if TOLD.search(l)), None)
    if told is None:
        fail.append("fk_migrate was never called: either the build stamp bump did "
                    "not take, or the guest stopped exporting the hook")

    hits = [m for m in (REBUILT.search(l) for l in run) if m]
    if not hits:
        print("the guest never rebuilt its registry from the world after the "
              "upgrade -- every later placement would have built a SECOND network")
        sys.exit(1)
    if len(hits) > 1:
        fail.append("the registry was rebuilt %d times in one session; it must "
                    "happen once, at the first event" % len(hits))

    rebuilt_at = next(i for i, l in enumerate(run) if REBUILT.search(l))
    if told is not None and told > rebuilt_at:
        fail.append("the registry was rebuilt from the world BEFORE fk_migrate ran, "
                    "so it was the first-event fallback that did it and the hook is "
                    "not driving the upgrade path")

    surfaces, parts, clusters, adopted, rebuilt = (int(g) for g in hits[0].groups())
    print("  after the upgrade: %d surfaces scanned, %d parts, %d clusters"
          % (surfaces, parts, clusters))
    print("  %d networks adopted as they stood, %d rebuilt" % (adopted, rebuilt))
    for m in (SWEPT.search(l) for l in run):
        if m:
            print("  %s orphaned hidden entities swept" % m.group(1))

    if parts < 30:
        fail.append("only %d parts were found; the M2 map has far more" % parts)
    if clusters < 8:
        fail.append("only %d clusters were rebuilt" % clusters)
    if adopted < clusters - 1:
        fail.append(
            "only %d of %d networks were adopted. Adoption is what makes an "
            "upgrade cheap and lossless -- a rebuild is ~350 host calls per "
            "cluster and spills every item in the network on the floor"
            % (adopted, clusters))

    if fail:
        print("\nUPGRADE ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\nupgrade assertions passed; M2's own numbers follow, over the "
          "adopted networks\n")


if __name__ == "__main__":
    main()
