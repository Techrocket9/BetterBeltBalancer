#!/usr/bin/env python3
"""Assert what a save's own state version decides about the curved exit.

The `curv` suite creates a world with one guest and loads it with another, the
way `upg` does, and adds the two things that suite has no reason to. The create
phase runs the PRE-STATE build, whose `fk_state_version` reports 0 the way every
build up to 0.3.2 did by not exporting it at all, so the save carries the
watermark of a world that predates the rule. And the observer lays the belts the
new rule would read as outputs with `create_entity` and NO `raise_built`, after
the audit marker has compiled every network into the save, so what it carries is
balancers the classifier would have built with the curve arm switched off.

What the load must do with it: read every one of them under the rule the save
was written to, tear down only what the world itself has changed, write
`bbb-curved-exits = false`, and tell each owning force once with a ping per
balancer (CLAUDE.md, "A save from before the curve rule keeps the reading it was
built to").

    python3 test/assert-curve.py create.log run.log
    python3 test/assert-curve.py --leg off    create.log run.log
    python3 test/assert-curve.py --leg state1 create.log run.log

THREE LEGS, AND EACH ONE MOVES ONE VARIABLE.

`kept` is the pass. `off` is the same world on a save whose setting was already
false before the map was made (test/run.sh's `curve_setting_off`): the curve arm
produces no edge there, so the load has nothing to decide about and must say
nothing -- and must not announce a flip nobody made on the way past.

`state1` is the same world created by the SHIPPED guest, so the save is stamped
1 and the load is not undecided at all. Its curve belts become outputs, its
balancers are rebuilt around them, and nothing is kept. That is the correct
answer for a 0.3.3 save and it is the leg that says the trigger is the WATERMARK
and not the shape of the world: `kept` and `off` see `fk_migrate(0)` and this one
sees `fk_migrate(1)` over a world that is otherwise identical.

AND THE FRESH-WORLD NEGATIVE IS THE CREATE LOG, at no extra Factorio run. The
create phase builds a world from nothing, so a create that logged a curve-kept
line would be one firing where no world was built to the old rule at all.
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
PING = re.compile(r"first \[gps=(-?\d+),(-?\d+),(\S+?)\]")
# HOW MANY CHAT LINES THE CHECKLIST TOOK. Written only when it took more than
# one, so a message that fits is byte for byte the line it has always been.
LINES = re.compile(r" in (\d+) lines")
# ...and the one ping a message may still lose: a cluster whose surface name
# could not be read. Every surface in this world resolves, so it must not appear.
TRUNCATED = re.compile(r"\(list truncated\)")
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
# THE OTHER RULE'S MIGRATION, WHICH MUST NEVER SPEAK HERE. A tile whose second
# edge is a curve is not a tile carrying two belts in a save that is keeping the
# old reading -- but a load that read the curve first saw one, condemned the
# machine, and then had `settleEdgeMode` write `bbb-multi-edge-parts = true` and
# tell the force it had kept multiple belts per part working, for a world with
# one belt on every part. Rig F is that shape and this is the assertion that
# catches it.
SEDGE_ANY = re.compile(r"\[BBB\] (?:alert: )?single-edge:")

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
# it had broken. Eight named clusters over twenty parts, on two surfaces and two
# forces: A and G are three parts (a 1 -> 1 and the edgeless spare the curve
# belt stands against), F is three (a 2 -> 1 whose middle part is edgeless), H
# is three on the second surface, and B, C, D and E are two each.
#
# Plus the BAND: forty more of A's shape at three parts each, on the player
# force, laid so that one force's checklist is longer than a chat line holds.
BAND = 40
CLUSTERS, PARTS = 8 + BAND, 20 + 3 * BAND

# Every rig with a belt across one of its faces: A, B, D, E, F, G, H and all
# forty of the band. C has none and must be read the same way whatever happens
# to the others.
LEGACY = 7 + BAND
# ...split by owning force. G is the second force's and everything else is the
# player's. The values are what each force is told about; the KEYS are not, and
# are not asserted -- a force INDEX is the engine's to hand out.
LEGACY_PER_FORCE = [6 + BAND, 1]

# WHAT THE WORLD ITSELF CHANGED, AND ONLY THAT. Five of the eight clusters are
# exactly as the old rule compiled them and must be adopted whole. Three are not,
# and none of them is the curve's doing:
#   D  was never compiled at all -- an input and no output is a half-built state
#   E  had its output belt mined with no event, so what stands describes a belt
#      that is gone
#   F  had an extra input laid with no event
# So three rebuilds, and the two of them that HAD a network pay a teardown: E's
# successor is input-only and cannot take its items back, so those spill; F's is
# a working 2 -> 1 and takes them.
WANT_ADOPTED, WANT_REBUILT = 5 + BAND, 3
WANT_TEARDOWNS, WANT_SPILLS = 2, 1

# The final audit. `nets` is short of `clusters` by D and E, which end the run
# with an input and no output -- a legitimate half-built state that is never
# counted `unbuilt`, which is exactly why it is written down beside it.
FINAL_AUDIT = (CLUSTERS, PARTS, 6 + BAND, 0, 0, 0)
# ...and at create, where the audit reports the registry as its own dispatch
# finds it and that dispatch is what compiles: everything but D is still to be
# built when it looks.
CREATE_AUDIT = (CLUSTERS, PARTS, 0, 0, 7 + BAND, 0)

# What a balancer delivers against a bare express belt fed the same way. The M2
# suite records 0.998x for a 1 -> 1; the bound is loose on purpose, because a
# rate that has moved for the reason this suite is about does not move by 2%.
RATE_LO, RATE_HI = 0.95, 1.02
MAINS = ("A-main", "B-main", "C-main", "F-main", "G-main", "H-main")
# Every belt the old rule read as nothing, and must go on reading as nothing.
CURVES = ("A-curve", "B-curve", "D-curve", "E-curve", "F-curve", "G-curve",
          "H-curve")


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
    """The value at a named moment, or None if the line is not there at all.

    A CHECK THAT SKIPS IS A CHECK THAT PASSED, which this repository has met
    twice and which is why every caller below tests for None separately from
    testing the value. `if ground:` read an absent line as a clean zero.
    """
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
    ap.add_argument("--leg", default="kept", choices=("kept", "off", "state1"))
    ap.add_argument("logs", nargs=2)
    args = ap.parse_args()
    create, run = parse(args.logs)
    leg = args.leg
    fail = []

    # ---------------------------------------------------------------- the world
    seeded = next((int(m.group(1)) for m in (SEEDED.search(l) for l in create) if m), 0)
    if seeded < 20:
        fail.append(
            "only %d items were seeded into the compiler's own entities; every "
            "'nothing was spilled' below would be a vacuous zero" % seeded)
    held = scalar(create, INSIDE, "create", 2)
    if held is None:
        fail.append("no inside-the-networks line in the create phase")
    elif held != seeded:
        fail.append(
            "%d items were seeded and %d are standing in the compiler's entities "
            "at the end of the create. A `--create` never reaches a tick, so "
            "nothing can have moved between the two" % (seeded, held))
    print("  seeded %d items into the networks, and %s are standing there when "
          "the save is written" % (seeded, held))

    got = [tuple(int(g) for g in m.groups())
           for m in (AUDIT.search(l) for l in create) if m]
    if not got:
        fail.append("no audit in the create phase")
    elif got[0] != CREATE_AUDIT:
        fail.append("the create audit is %s and the rigs build %s"
                    % (got[0], CREATE_AUDIT))

    # THE FRESH-WORLD NEGATIVE. The create phase builds a world from nothing, so
    # nothing in it was built to a rule it did not have.
    if [l for l in create if ANY_CURVE.search(l)]:
        fail.append("the create phase spoke about curved exits, and it builds a "
                    "world from nothing: there was no earlier rule for anything "
                    "in it to have been built to")
    want_create = "false" if leg == "off" else "true"
    if setting_at(create, "create") != want_create:
        fail.append("the setting read %r in the create phase and this leg leaves "
                    "it %s" % (setting_at(create, "create"), want_create))
    print("  create: no curved-exit line, and the setting reads %s" % want_create)

    # ------------------------------------------------------------- the rebuild
    stamp = [int(m.group(1)) for m in (TOLD_MIGRATE.search(l) for l in run) if m]
    if not stamp:
        fail.append("fk_migrate was never called: either the build stamp bump "
                    "did not take, or the guest stopped exporting the hook")
    else:
        # THE WHOLE TRIGGER, IN ONE NUMBER. 0 is what FkLua stores for a guest
        # that does not export fk_state_version, which is every build up to
        # 0.3.2 and which the pre-state fixture reproduces; 1 is this build.
        want_stamp = 1 if leg == "state1" else 0
        print("  fk_migrate was handed state version %d" % stamp[0])
        if stamp[0] != want_stamp:
            fail.append(
                "the save this leg loaded was stamped state version %d and it "
                "has to be %d. %s" % (stamp[0], want_stamp,
                                      "The create phase ran the shipped guest "
                                      "rather than the pre-state one"
                                      if want_stamp == 0 else
                                      "The create phase ran the pre-state guest "
                                      "rather than the shipped one"))

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

    if leg == "state1":
        state1(create, run, fail)
    else:
        kept_leg(leg, run, adopted, rebuilt, fail)

    if fail:
        print("\nCURVE UPGRADE ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\ncurve upgrade assertions passed")


def kept_leg(leg, run, adopted, rebuilt, fail):
    """The two legs whose save predates the rule: one to decide, one already decided."""
    if (adopted, rebuilt) != (WANT_ADOPTED, WANT_REBUILT):
        fail.append(
            "%d networks were adopted and %d rebuilt, and the rigs are laid so "
            "that %d must be adopted and %d rebuilt. A cluster rebuilt here is "
            "one whose standing network was thrown away and re-derived under a "
            "rule its world was not built to, which is exactly what a player "
            "must not find -- and the three that ARE rebuilt are rebuilt because "
            "the WORLD changed under them, not because the rule did"
            % (adopted, rebuilt, WANT_ADOPTED, WANT_REBUILT))

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
    if [l for l in run if SEDGE_ANY.search(l)]:
        fail.append("the multi-edge migration spoke. Every part in this world "
                    "carries one belt, and the only way a tile here reads as "
                    "two is a curve edge the load should never have kept")

    if leg == "kept":
        if len(kept) != 1:
            fail.append("the guest kept the old reading %d times and it must do "
                        "it once, on the load that decides" % len(kept))
        elif int(kept[0].group(1)) != LEGACY:
            fail.append("the guest kept the old reading for %s balancers and the "
                        "rigs build %d with a belt across a face"
                        % (kept[0].group(1), LEGACY))
        per, chunked = [], 0
        for m in told:
            f, n, pings, tail = m.groups()
            ch, gps = CHARTED.search(tail), PING.search(tail)
            ln = LINES.search(tail)
            nlines = int(ln.group(1)) if ln else 1
            print("  told force %s about %s balancers, %s pings in %d chat "
                  "line(s), %s, charted %s"
                  % (f, n, pings, nlines, gps.group(0) if gps else "no ping",
                     ch.group(1) if ch else "?"))
            if TRUNCATED.search(tail):
                fail.append("force %s's ping list was truncated. The only ping a "
                            "message may lose is one whose surface name could not "
                            "be read, and both surfaces here resolve" % f)
            if n != pings or not gps:
                fail.append("force %s was told about %s balancers with %s pings: "
                            "a checklist that names a machine and does not point "
                            "at it is the scavenger hunt the pings exist to end"
                            % (f, n, pings))
            if not ch or ch.group(1) != pings:
                fail.append("%s balancers were charted and %s were pinged; a "
                            "[gps=] at uncharted ground opens on black"
                            % (ch.group(1) if ch else "no", pings))
            if nlines > 1:
                chunked += 1
            per.append(int(n))
        # THE ANTI-VACUITY, and it is what the band is for. `pings == balancers`
        # above is satisfied by any list short enough to have fitted, which is
        # every suite in the estate before this one: a checklist of forty-six
        # cannot be built into one chat line, so a run in which none of them took
        # more than one is a run in which the band did not land, and the
        # assertion above was never asked the question it exists to ask.
        if told and not chunked:
            fail.append("every checklist fitted in one chat line, and the band "
                        "puts %d balancers on the player force's. Either the band "
                        "is not in the world or the list was cut" % LEGACY_PER_FORCE[0])
        # THE ORDER IS ASSERTED AND NOT SORTED AWAY. Which force is told first,
        # and which of its balancers the ping list names, come out of the
        # rebuild's own cluster order -- and that order reaches every client, so
        # a run in which it moved would be a run in which two peers were handed
        # two different checklists.
        if per != LEGACY_PER_FORCE:
            fail.append("the forces were told about %s balancers, in that order, "
                        "and the rigs put %s on them" % (per, LEGACY_PER_FORCE))
        for tag in ("t1", "final"):
            if setting_at(run, tag) != "false":
                fail.append("settings.global bbb-curved-exits reads %r at %s and "
                            "the pass is supposed to have written it false"
                            % (setting_at(run, tag), tag))
    else:
        # THE SAVE WAS DECIDED BEFORE IT WAS EVER LOADED, so there is nothing for
        # this pass to find and nothing for it to say -- and the world comes out
        # exactly as it does in the leg that decides, which is the point.
        if kept or told or [l for l in run if ANY_CURVE.search(l)]:
            fail.append("a save whose setting was already off was decided again: "
                        "the curve arm cannot produce an edge there, so nothing "
                        "in that load can have been read as evidence")
        for tag in ("t1", "final"):
            if setting_at(run, tag) != "false":
                fail.append("the setting reads %r at %s and this leg staged it "
                            "off before the map was made"
                            % (setting_at(run, tag), tag))
        print("  the already-decided save: no curved-exit line anywhere, setting "
              "still off")

    # ------------------------------------------------ what moved in the world
    teardowns = sum(1 for l in run if TEARDOWN.search(l))
    spills = [int(m.group(1)) for m in (SPILL.search(l) for l in run) if m]
    compiles = sum(1 for l in run if COMPILED.search(l))
    print("  %d teardowns, %d spills (%d items), %d compiles"
          % (teardowns, len(spills), sum(spills), compiles))
    if (teardowns, len(spills)) != (WANT_TEARDOWNS, WANT_SPILLS):
        fail.append(
            "%d teardowns and %d spills, and the rigs are laid for %d and %d. "
            "Five of these eight clusters are exactly as the old rule compiled "
            "them and must not be touched at all; the two teardowns are E, whose "
            "output belt is gone, and F, which gained an input"
            % (teardowns, len(spills), WANT_TEARDOWNS, WANT_SPILLS))
    ground = scalar(run, GROUND, "final", 2)
    if ground is None:
        fail.append("no ground-items line at the end of the run")
    elif ground != sum(spills):
        fail.append("%d items are on the ground at the end of the run and %d were "
                    "spilled: the only items that may reach the floor here are "
                    "the ones E's teardown could not hand back" % (ground, sum(spills)))

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
            for name in MAINS:
                delivered = b.get(name, 0) - a.get(name, 0)
                r = delivered / ctrl
                print("  %-7s %5d items, %.3fx one belt" % (name, delivered, r))
                if not RATE_LO <= r <= RATE_HI:
                    fail.append("%s delivered %d items against the control's "
                                "%d, %.3fx one belt" % (name, delivered, ctrl, r))
            # E's OUTPUT BELT WAS MINED, so its chest is fed by nothing and stays
            # at zero. That is what says the recompile left it an INPUT-ONLY
            # cluster rather than joining it to the belt running past.
            if b.get("E-main") != 0:
                fail.append("E-main took %s items and its output belt was mined "
                            "before the save was written" % b.get("E-main"))
            for name in CURVES:
                delivered = b.get(name, 0) - a.get(name, 0)
                print("  %-7s %5d items" % (name, delivered))
                if delivered:
                    fail.append(
                        "%s took %d items. That line is a belt the old rule read "
                        "as nothing, and a save that kept the old rule must go "
                        "on reading it as nothing" % (name, delivered))

    # ---------------------------------------------------------------- the audit
    audits = [m for m in (AUDIT.search(l) for l in run) if m]
    if not audits:
        fail.append("no audit in the benchmark phase")
    else:
        got = tuple(int(g) for g in audits[-1].groups())
        print("  final audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d "
              "refused=%d" % got)
        if got != FINAL_AUDIT:
            fail.append("the final audit is %s and the rigs build %s. `nets` is "
                        "asserted beside `unbuilt` because a cluster with an "
                        "input and no output is a half-built state and is never "
                        "counted unbuilt" % (got, FINAL_AUDIT))


def state1(create, run, fail):
    """The save this build wrote: no decision, and the curve belts are outputs."""
    if [l for l in run if ANY_CURVE.search(l)]:
        fail.append("the load decided something about curved exits on a save "
                    "this build wrote. Its standing networks ARE the curve-on "
                    "reading; there is nothing in it to keep")
    for tag in ("create", "t1", "final"):
        lines = create if tag == "create" else run
        if setting_at(lines, tag) != "true":
            fail.append("the setting reads %r at %s and nothing in this leg may "
                        "write it" % (setting_at(lines, tag), tag))
    # AND THE BELTS ARE READ AS OUTPUTS, which is the half that says the leg is
    # not passing because the world is inert. The world is the same forged world
    # as the other two legs; what differs is the watermark, and under this one
    # every curve belt is a port.
    a, b = counts(run, "t1"), counts(run, "t2")
    if not a or not b:
        fail.append("no chest counts to say whether the curve belts were read as "
                    "outputs")
        return
    took = {n: b.get(n, 0) - a.get(n, 0) for n in CURVES}
    print("  the curve chests took %s" % took)
    if not any(v > 0 for v in took.values()):
        fail.append(
            "not one curve belt delivered anything. On a save stamped with this "
            "build's own state version they are ordinary outputs, so a run in "
            "which they carry nothing is one where the world was never built")


if __name__ == "__main__":
    main()
