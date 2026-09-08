#!/usr/bin/env python3
"""Assert that a save built before the curved exit keeps the reading it was built to.

The `curv` suite creates a world with one guest and loads it with another, the
way `upg` does -- and its observer does the one thing `upg` has no reason to:
after the audit marker has compiled every network into the save, it lays the
belts the new rule would read as outputs with `create_entity` and NO
`raise_built`, so the mod is never told. What the save carries is therefore
three networks the classifier would have built with the curve arm switched off,
in a world the curve arm has plenty to say about, which is what a save from
before 0.3.3 is (CLAUDE.md, "A save from before the curve rule keeps the reading
it was built to").

What the load must do with it: adopt every one of those networks on their
curve-free reading, tear nothing down, spill nothing, write
`bbb-curved-exits = false` for that save, and tell each owning force once with a
ping per balancer.

    python3 test/assert-curve.py create.log run.log
    python3 test/assert-curve.py --leg off create.log run.log

THE `off` LEG IS THE SAME WORLD ON A SAVE THAT WAS ALREADY DECIDED, staged by
test/run.sh writing a mod-settings.dat. With the setting off the curve arm
produces no edge at all, so the adoption comparison matches on its FIRST reading
and this pass never runs -- which is the negative that says it cannot fire twice.

AND THE FRESH-WORLD NEGATIVE IS THE CREATE LOG, at no extra Factorio run. The
create phase is this same guest building a world from nothing: `bump_build`
moves a version and a build stamp and not a line of code, so a create that
logged a curve-kept line would be one firing on a save with no adopted network
at all.
"""

import argparse
import re
import sys

REBUILT = re.compile(
    r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters "
    r"\((\d+) networks adopted, (\d+) rebuilt\)"
)
TOLD_MIGRATE = re.compile(r"\[BBB\] the mod was rebuilt \(guest state version (\d+)\)")

KEPT = re.compile(
    r"\[BBB\] curved exits: kept the old reading for this save -- (\d+) balancers"
)
KEPT_FAILED = re.compile(r"\[BBB\] alert: curved exits: (\d+) balancers were built")
TOLD = re.compile(
    r"\[BBB\] curved exits: told force (\d+) about (\d+) balancers built before "
    r"a belt could turn as it left one, (\d+) pings(.*)$"
)
CHARTED = re.compile(r"charted (\d+)")
# Any line at all from the pass, for the two negatives.
ANY_CURVE = re.compile(r"\[BBB\] (?:alert: )?curved exits:")
# THE FLIP HANDLER'S OWN LINE, which must not appear at all. The write this pass
# makes raises `on_runtime_mod_setting_changed` synchronously, and a handler that
# announced it would be telling the player they had changed something they had
# not -- and re-queueing the whole save from inside the settling flush to do it.
# It is what the anchor in settleCurveMode exists to prevent, and it fired on the
# first passing run of this suite (CLAUDE.md, "A save from before the curve
# rule").
FLIPPED = re.compile(r"\[BBB\] curved exits: a belt across a balancer's face is")

TEARDOWN = re.compile(r"\[BBB\] torn down cluster")
SPILL = re.compile(r"\[BBB\] spilled (\d+) items beside cluster")
COMPILED = re.compile(r"\[BBB\] compiled cluster")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)"
)

SETTING = re.compile(r"\[BBB-CURV\] setting t=(\S+) value=(\S+)")
CHEST = re.compile(r"\[BBB-CURV\] chest t=(\S+) (\S+) n=(-?\d+)")
GROUND = re.compile(r"\[BBB-CURV\] ground t=(\S+) items=(-?\d+) stacks=(-?\d+)")
INSIDE = re.compile(r"\[BBB-CURV\] inside t=(\S+) items=(-?\d+) ents=(-?\d+)")
SEEDED = re.compile(r"\[BBB-CURV\] seeded (\d+) items")

# THE WORLD, WRITTEN DOWN HERE RATHER THAN READ OFF THE GUEST. `mar`'s own red
# proof is why: an injected defect that halved a rig passed every assertion the
# suite had, because every number it checked came from the same classification
# it had broken. Three clusters -- A is three parts (a 1 -> 1 and the edgeless
# spare the curve belt stands against), B and C are two each.
CLUSTERS, PARTS = 3, 7
# A and B. C has no perpendicular belt anywhere near it and must adopt on the
# first reading like any ordinary save.
LEGACY = 2

# What a 1 -> 1 delivers against a bare express belt fed the same way. The M2
# suite records 0.998x for this shape; the bound is loose on purpose, because a
# rate that has moved for the reason this suite is about does not move by 2%.
RATE_LO, RATE_HI = 0.95, 1.02


def parse(paths):
    out = []
    for p in paths:
        with open(p, errors="replace") as f:
            out.append(f.readlines())
    return out


def counts(lines, tag):
    """chest name -> count at a named moment."""
    got = {}
    for m in (CHEST.search(l) for l in lines):
        if m and m.group(1) == tag:
            got[m.group(2)] = int(m.group(3))
    return got


def scalar(lines, rx, tag, group):
    for m in (rx.search(l) for l in lines):
        if m and m.group(1) == tag:
            return int(m.group(group))
    return None


def setting_at(lines, tag):
    for m in (SETTING.search(l) for l in lines):
        if m and m.group(1) == tag:
            return m.group(2)
    return None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--leg", default="kept", choices=("kept", "off"))
    ap.add_argument("logs", nargs=2)
    args = ap.parse_args()
    create, run = parse(args.logs)
    fail = []

    # ---------------------------------------------------------------- the world
    seeded = next((int(m.group(1)) for m in (SEEDED.search(l) for l in create) if m), 0)
    if seeded < 20:
        fail.append(
            "only %d items were seeded into the compiler's own entities; every "
            "'nothing was spilled' below would be a vacuous zero" % seeded)
    print("  seeded %d items into the networks before the save was written" % seeded)

    # THE FRESH-WORLD NEGATIVE. The create phase is this guest building a world
    # from nothing, so nothing can have been adopted and nothing may be kept.
    if [l for l in create if ANY_CURVE.search(l)]:
        fail.append("the create phase spoke about curved exits, and it builds a "
                    "world from nothing: there is no adopted network for an "
                    "adoption to have failed at")
    want_create = "true" if args.leg == "kept" else "false"
    if setting_at(create, "create") != want_create:
        fail.append("the setting read %r in the create phase and this leg leaves "
                    "it %s" % (setting_at(create, "create"), want_create))
    print("  create: no curved-exit line, and the setting reads %s" % want_create)

    # ------------------------------------------------------------- the rebuild
    if not any(TOLD_MIGRATE.search(l) for l in run):
        fail.append("fk_migrate was never called: either the build stamp bump "
                    "did not take, or the guest stopped exporting the hook")
    hits = [m for m in (REBUILT.search(l) for l in run) if m]
    if not hits:
        print("the guest never rebuilt its registry from the world; there is "
              "nothing for this suite to be about")
        sys.exit(1)
    if len(hits) > 1:
        fail.append("the registry was rebuilt %d times in one session" % len(hits))
    surfaces, parts, clusters, adopted, rebuilt = (int(g) for g in hits[0].groups())
    print("  rebuilt: %d surfaces, %d parts, %d clusters, %d adopted, %d rebuilt"
          % (surfaces, parts, clusters, adopted, rebuilt))
    if (clusters, parts) != (CLUSTERS, PARTS):
        fail.append("the rebuild found %d clusters of %d parts and the rigs "
                    "build %d of %d" % (clusters, parts, CLUSTERS, PARTS))
    if rebuilt or adopted != clusters:
        fail.append(
            "%d of %d networks were adopted and %d rebuilt. Adopting is the "
            "whole of this pass: a rebuilt cluster is one whose standing network "
            "was thrown away and re-derived under the new rule, which is exactly "
            "what a player must not find" % (adopted, clusters, rebuilt))

    # ------------------------------------------------------------- the decision
    kept = [m for m in (KEPT.search(l) for l in run) if m]
    told = [m for m in (TOLD.search(l) for l in run) if m]
    if any(KEPT_FAILED.search(l) for l in run):
        fail.append("the setting could not be written; the guest said so and the "
                    "next load would classify these balancers with curves again")

    if [l for l in run if FLIPPED.search(l)]:
        fail.append("the flip handler announced a change nobody made. The only "
                    "write in this run is the guest's own, and a re-entrant "
                    "handler that acts on it re-queues the whole save from "
                    "inside the flush that is settling it")

    if args.leg == "kept":
        if len(kept) != 1:
            fail.append("the guest kept the old reading %d times and it must do "
                        "it once, on the load that decides" % len(kept))
        elif int(kept[0].group(1)) != LEGACY:
            fail.append("the guest kept the old reading for %s balancers and the "
                        "rigs build %d that were laid to it"
                        % (kept[0].group(1), LEGACY))
        if len(told) != 1:
            fail.append("%d forces were told and one force owns every rig here"
                        % len(told))
        else:
            f, n, pings, tail = told[0].groups()
            ch = CHARTED.search(tail)
            print("  told force %s about %s balancers, %s pings, charted %s"
                  % (f, n, pings, ch.group(1) if ch else "?"))
            if int(n) != LEGACY or int(pings) != LEGACY:
                fail.append("force %s was told about %s balancers with %s pings "
                            "and the rigs build %d: a checklist that names a "
                            "machine and does not point at it is the scavenger "
                            "hunt the pings exist to end" % (f, n, pings, LEGACY))
            if not ch or int(ch.group(1)) != LEGACY:
                fail.append("%s balancers were charted and %d were pinged; a "
                            "[gps=] at uncharted ground opens on black"
                            % (ch.group(1) if ch else "no", LEGACY))
        for tag in ("t1", "final"):
            if setting_at(run, tag) != "false":
                fail.append("settings.global bbb-curved-exits reads %r at %s and "
                            "the pass is supposed to have written it false"
                            % (setting_at(run, tag), tag))
    else:
        # THE SAVE WAS DECIDED BEFORE IT WAS EVER LOADED, so there is nothing for
        # this pass to find and nothing for it to say.
        if kept or told or [l for l in run if ANY_CURVE.search(l)]:
            fail.append("a save whose setting was already off was decided again: "
                        "the curve arm cannot produce an edge there, so the "
                        "adoption comparison should have matched on its first "
                        "reading and this pass should never have run")
        for tag in ("t1", "final"):
            if setting_at(run, tag) != "false":
                fail.append("the setting reads %r at %s and this leg staged it "
                            "off before the map was made"
                            % (setting_at(run, tag), tag))
        print("  the already-decided save: no curved-exit line anywhere, setting "
              "still off")

    # ------------------------------------------------ nothing moved in the world
    teardowns = sum(1 for l in run if TEARDOWN.search(l))
    spills = [int(m.group(1)) for m in (SPILL.search(l) for l in run) if m]
    compiles = sum(1 for l in run if COMPILED.search(l))
    print("  %d teardowns, %d spills (%d items), %d compiles"
          % (teardowns, len(spills), sum(spills), compiles))
    if teardowns or spills:
        fail.append("%d teardowns and %d spills of %d items. Adopting is what "
                    "this pass is for: a save that keeps its own reading has "
                    "nothing to tear down and nothing to put on the floor"
                    % (teardowns, len(spills), sum(spills)))
    ground = scalar(run, GROUND, "final", 2)
    if ground:
        fail.append("%d items are on the ground at the end of the run" % ground)

    # ------------------------------------------------------------- the balancers
    for tag in ("t1", "t2"):
        if not counts(run, tag):
            fail.append("no chest counts for t=%s" % tag)
    a, b = counts(run, "t1"), counts(run, "t2")
    if a and b:
        ctrl = b.get("ctrl", 0) - a.get("ctrl", 0)
        if ctrl < 500:
            fail.append("the control belt delivered %d items over the window; "
                        "every ratio below would be measured against nothing"
                        % ctrl)
        else:
            for name in ("A-main", "B-main", "C-main"):
                got = b.get(name, 0) - a.get(name, 0)
                r = got / ctrl
                print("  %-7s %5d items, %.3fx one belt" % (name, got, r))
                if not RATE_LO <= r <= RATE_HI:
                    fail.append("%s delivered %d items against the control's "
                                "%d, %.3fx one belt" % (name, got, ctrl, r))
            for name in ("A-curve", "B-curve"):
                got = b.get(name, 0) - a.get(name, 0)
                print("  %-7s %5d items" % (name, got))
                if got:
                    fail.append(
                        "%s took %d items. That line is a belt the old rule read "
                        "as nothing, and a save that kept the old rule must go "
                        "on reading it as nothing" % (name, got))

    # ---------------------------------------------------------------- the audit
    audits = [m for m in (AUDIT.search(l) for l in run) if m]
    if not audits:
        fail.append("no audit in the benchmark phase")
    else:
        got = tuple(int(g) for g in audits[-1].groups())
        want = (CLUSTERS, PARTS, CLUSTERS, 0, 0, 0)
        print("  final audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d "
              "refused=%d" % got)
        if got != want:
            fail.append("the final audit is %s and the rigs build %s. `nets` is "
                        "asserted beside `unbuilt` because a cluster with no "
                        "outputs is a half-built state and is never counted "
                        "unbuilt" % (got, want))

    if fail:
        print("\nCURVE UPGRADE ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\ncurve upgrade assertions passed")


if __name__ == "__main__":
    main()
