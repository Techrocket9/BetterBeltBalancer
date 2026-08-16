#!/usr/bin/env python3
"""Assert that a Belt Balancer 2 save is adopted when Belt Balancer 2 is removed.

Four legs, run by test/run.sh, each a pair of Factorio phases and three of
them under DIFFERENT MOD SETS:

    --leg added    the incumbent swapped out and this mod in, in one edit
    --leg later    this mod installed first, the incumbent removed a session on
    --leg built    no incumbent ever, and the legacy parts arrive one at a time
                   through BUILD EVENTS -- which is what an old blueprint's
                   ghosts do. This leg is also the only one whose save is
                   written AFTER the conversion, so its second phase is a plain
                   reload that must do nothing at all
    --leg foreign  a mod that is NOT any fork of the incumbent owns
                   `balancer-part`, and nothing of its may be touched

What is checked, and why each one is here:

  * THE CONVERSION HAPPENED ONCE. The summary line carries the trigger that
    drove it, so an assertion can tell the load-time hook apart from the
    fallback behind it -- a feature whose fallback silently does the work of its
    primary trigger passes every test and ships broken.
  * NOTHING WAS LEFT BEHIND. A census of both prototype names, before and after.
  * THE ITEMS ON THE BELTS SURVIVED, exactly. The witness rig runs COPPER PLATE
    and every other rig in the save runs iron, so a copper count across every
    surface is that rig's contents and nothing else -- an equality rather than
    an estimate.
  * THE ITEM STACK SURVIVED, and places this mod's parts.
  * THE FORCE CAN STILL CRAFT. The incumbent's technologies went with it.
  * THE ADOPTED BALANCERS BALANCE, against a bare express belt in the same save.
    A network adopted from the wrong edge list does not show up as a crash; it
    shows up as a rate.
  * AND THE REGISTRY AGREES WITH THE WORLD afterwards: drift=0, unbuilt=0, and
    one network per cluster.

    python3 test/assert-mig.py --leg added create.log run.log
"""

import re
import sys

ADOPTED = re.compile(
    r"\[BBB\] legacy: adopted (\d+) parts? from (\d+) surfaces into (\d+) "
    r"clusters, (\d+) forces given the balancer technology, trigger=(\S+)"
)
BLOCKED = re.compile(r"\[BBB\] legacy: (\S+) (\S+) is active; its balancers are left alone")
BUILT = re.compile(r"\[BBB\] legacy: adopted a balancer-part built at (-?\d+),(-?\d+)")
CENSUS = re.compile(r"\[BBB-MIG\] census phase=(\S+) balancer-part=(\d+) bbb-balancer-part=(\d+)")
COUNT = re.compile(r"\[BBB-MIG\] count phase=(\S+) copper-plate=(\d+)")
ITEM = re.compile(r"\[BBB-MIG\] legacy-item phase=(\S+) held=(-?\d+) place_result=(\S+)")
TECH = re.compile(r"\[BBB-MIG\] tech phase=(\S+) bbb-balancer=(\S+) belt-balancer-1=(\S+)")
SAMPLE = re.compile(r"\[BBB-MIG\] sample tick=(\d+) (.*)")
LATE = re.compile(r"\[BBB-MIG\] late-build legacy=(\d+) ours=(\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
)

# The trigger each leg must have been driven by. `added` gets on_init, because
# this mod is new to the save; `later` gets on_configuration_changed, because it
# was already there and only the mod set moved. Anything else -- and in
# particular `first-dispatch` or `deferred`, the fallbacks -- means the load-time
# hook did not fire and the conversion happened late, on a tick nobody chose.
EXPECT_TRIGGER = {"added": "init", "later": "configuration_changed"}

SPREAD = 0.01       # between live outputs of one balancer
RATE_TOL = 0.02     # against the control belt


def find_all(lines, rx):
    return [m for m in (rx.search(l) for l in lines) if m]


def parse_sample(text):
    """`ctrl=123 m4x4=1,2,3,4` -> {'ctrl': [123], 'm4x4': [1,2,3,4]}"""
    out = {}
    for chunk in text.split():
        name, _, csv = chunk.partition("=")
        out[name] = [int(v) for v in csv.split(",")]
    return out


def main():
    args = sys.argv[1:]
    leg = "added"
    if args and args[0] == "--leg":
        leg, args = args[1], args[2:]

    create, run = [], []
    for path, into in zip(args[:2], (create, run)):
        with open(path, errors="replace") as f:
            into.extend(f)
    both = create + run

    fail = []

    census = {m.group(1): (int(m.group(2)), int(m.group(3))) for m in find_all(both, CENSUS)}
    counts = {m.group(1): int(m.group(2)) for m in find_all(both, COUNT)}
    items = {m.group(1): (int(m.group(2)), m.group(3)) for m in find_all(both, ITEM)}
    techs = {m.group(1): (m.group(2), m.group(3)) for m in find_all(both, TECH)}
    adopted = find_all(both, ADOPTED)
    blocked = find_all(both, BLOCKED)
    audits = find_all(run, AUDIT)

    for phase in ("create", "t1"):
        if phase not in census:
            print("no census for phase=%s: the observer mod did not run" % phase)
            sys.exit(1)

    legacy_before, ours_before = census["create"]
    print("  phase one: %d %s standing, %d of ours" % (legacy_before, "balancer-part", ours_before))

    if legacy_before < 8:
        fail.append("only %d incumbent parts were built; the rigs did not go up"
                    % legacy_before)
    if ours_before != 0:
        fail.append("phase one already had %d of this mod's parts; the leg is not "
                    "measuring a migration" % ours_before)

    # ------------------------------------------------------------------ built
    # No incumbent was ever installed here, so `balancer-part` is this mod's own
    # stub from the first byte and every part the observer places arrives through
    # a BUILD EVENT rather than through the scan. That is the path an old
    # blueprint's ghosts take, minus the robot.
    if leg == "built":
        if find_all(both, ADOPTED):
            fail.append("the whole-world scan ran; in this leg there is nothing "
                        "for it to find, and every part should have been swapped "
                        "by the flush that followed its own build event")
        built = find_all(create, BUILT)
        print("  %d parts swapped after arriving through a build event" % len(built))
        if len(built) != legacy_before:
            fail.append("%d parts were placed and %d were swapped through the build "
                        "path" % (legacy_before, len(built)))
        # THE PLAIN RELOAD. This is the only leg whose save is written AFTER a
        # conversion and whose second phase changes no mod at all, so it is the
        # only place the once-per-save flag can be seen surviving a save.
        #
        # WHAT IS ASSERTED IS WHAT A LOG CAN SHOW: the reload takes no DECISION
        # (no blocked line) and reports no SCAN (no summary line). It cannot
        # distinguish "did not scan" from "scanned an already-converted world and
        # found nothing", because an empty scan is silent by design -- a save
        # that never had an incumbent would otherwise carry a line for every new
        # map anybody makes. What says the flag really survived is the late build
        # below: it is swapped, so the phase is still Done rather than having
        # been decided again into something else.

    # ---------------------------------------------------------------- foreign
    if leg == "foreign":
        if adopted:
            fail.append("a stranger's `balancer-part` entities were CONVERTED (%s). "
                        "The marker prototype is the only thing standing between "
                        "this mod and every other mod's entities"
                        % adopted[0].group(0).strip())
        if blocked:
            fail.append("the guest reported an incumbent as active; bbb-mig-foreign "
                        "is not one and the log line is wrong")
        for phase in ("t1", "post-audit"):
            after = census.get(phase)
            if after is None:
                fail.append("no census for phase=%s" % phase)
                continue
            print("  phase=%-10s balancer-part=%d bbb-balancer-part=%d"
                  % (phase, after[0], after[1]))
            if after != (legacy_before, 0):
                fail.append("the world moved at phase=%s: balancer-part %d -> %d "
                            "and ours 0 -> %d"
                            % (phase, legacy_before, after[0], after[1]))
        base = counts.get("create")
        for phase in ("t1", "post-audit"):
            if phase in counts and counts[phase] != base:
                fail.append("the witness rig's copper went %d -> %d at phase=%s"
                            % (base, counts[phase], phase))
        late = find_all(run, LATE)
        if not late:
            fail.append("the late build never ran; the one probe that separates "
                        "'this balancer-part is mine' from 'somebody else's' did "
                        "not happen")
        else:
            lg, ou = int(late[-1].group(1)), int(late[-1].group(2))
            print("  a balancer-part BUILT in phase two: legacy=%d ours=%d" % (lg, ou))
            if (lg, ou) != (1, 0):
                fail.append("a `balancer-part` built beside the stranger came out "
                            "legacy=%d ours=%d: the build path converted a "
                            "prototype this mod does not own" % (lg, ou))
        if "t1" in items and items["t1"][1] != "balancer-part":
            fail.append("the stranger's item now places %r; this mod rewrote a "
                        "prototype it does not own" % items["t1"][1])
        if not audits:
            fail.append("no audit ran; the guest was never asked to look at this world")
        else:
            c, p, nets, drift, unbuilt = (int(g) for g in audits[-1].groups())
            print("  audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d"
                  % (c, p, nets, drift, unbuilt))
            if c or p or drift or unbuilt:
                fail.append("the audit found clusters=%d parts=%d drift=%d "
                            "unbuilt=%d; this mod should own nothing at all in "
                            "this save" % (c, p, drift, unbuilt))
        report(fail, "the stranger's balancers were left alone")
        return

    # ------------------------------------------------------ added and later
    if leg == "later":
        # EXACTLY ONE, and that is the latch rather than tidiness: the create
        # phase places an audit marker, which is a door onto the re-arm, and a
        # Blocked state that re-decided on every audit would say so again here.
        if len(blocked) != 1:
            fail.append("phase one had both mods installed and the guest said it "
                        "was leaving the incumbent alone %d times; it is once per "
                        "decision" % len(blocked))
        else:
            print("  coexistence: %s" % blocked[0].group(0).strip())
        if find_all(create, ADOPTED):
            fail.append("something was converted while the incumbent was still "
                        "installed, which is the one thing this feature promises "
                        "not to do")

    clusters = None
    if leg != "built" and not adopted:
        print("nothing was adopted at all. Either the data stage did not keep the "
              "entities alive across the load -- in which case the census below is "
              "zero and the world is empty -- or the guest never looked.")
        for phase, (legacy, ours) in sorted(census.items()):
            print("    census phase=%-10s balancer-part=%d bbb-balancer-part=%d"
                  % (phase, legacy, ours))
        sys.exit(1)
    if len(adopted) > 1:
        fail.append("the migration ran %d times in one save; it is once per save, "
                    "and the phase flag that says so lives in the guest heap"
                    % len(adopted))

    if adopted:
        n, surfaces, clusters, forces, trigger = adopted[0].groups()
        n, clusters, forces = int(n), int(clusters), int(forces)
        print("  adopted %d parts from %s surfaces into %d clusters, %d forces "
              "researched, trigger=%s" % (n, surfaces, clusters, forces, trigger))
        if n != legacy_before:
            fail.append("%d parts were standing and %d were adopted"
                        % (legacy_before, n))
        if forces < 1:
            fail.append("no force was given the balancer technology; a player with "
                        "fifty balancers and no recipe is worse off than before")
        want = EXPECT_TRIGGER.get(leg)
        if want is not None and trigger != want:
            fail.append("the conversion was driven by trigger=%s and this leg must "
                        "be driven by %s -- a fallback doing the primary trigger's "
                        "work passes every test and ships broken" % (trigger, want))

    after_legacy, after_ours = census["t1"]
    if after_legacy != 0:
        fail.append("%d incumbent parts are still standing after the conversion"
                    % after_legacy)
    if after_ours != legacy_before:
        fail.append("%d parts went in and %d of ours came out"
                    % (legacy_before, after_ours))

    # The witness: exact, across every surface, including the hidden one that did
    # not exist in phase one.
    base = counts.get("create")
    if base is None:
        fail.append("no copper count was taken in phase one")
    elif base < 16:
        fail.append("the witness rig held only %d copper plates; a conservation "
                    "claim over that is vacuous" % base)
    else:
        print("  witness: %d copper plates in phase one" % base)
        for phase in ("t1", "post-audit", "final"):
            got = counts.get(phase)
            if got is None:
                continue
            if got != base:
                fail.append("copper %d -> %d at phase=%s: the swap, or the first "
                            "compile after it, lost items off the belts"
                            % (base, got, phase))
            else:
                print("  witness: %d at phase=%s" % (got, phase))

    # The item half.
    if "create" in items and "t1" in items:
        held0, place0 = items["create"]
        held1, place1 = items["t1"]
        print("  legacy item: %d held, place_result %s -> %s" % (held1, place0, place1))
        if held1 != held0:
            fail.append("the stack of the incumbent's item went %d -> %d across the "
                        "swap; a removed prototype takes its items with it and the "
                        "stub is what stops that" % (held0, held1))
        if place1 != "bbb-balancer-part":
            fail.append("a surviving legacy stack places %r, so it is present and "
                        "useless" % place1)

    # The technology half.
    if "t1" in techs:
        ours, theirs = techs["t1"]
        print("  technology after the swap: bbb-balancer=%s belt-balancer-1=%s"
              % (ours, theirs))
        if ours != "true":
            fail.append("the force cannot craft a balancer part after the migration "
                        "(bbb-balancer=%s)" % ours)
        if theirs != "absent":
            fail.append("the incumbent's technology is still present (%s); the "
                        "incumbent was not really removed" % theirs)

    # Throughput over the window, against the control belt in the same save.
    samples = {int(m.group(1)): parse_sample(m.group(2)) for m in find_all(run, SAMPLE)}
    if 1800 not in samples or 3540 not in samples:
        fail.append("the throughput window was not sampled at both ends")
    else:
        a, b = samples[1800], samples[3540]
        delta = {k: [y - x for x, y in zip(a[k], b[k])] for k in b if k in a}
        ctrl = delta["ctrl"][0]
        print("  one saturated express belt delivered %d items over the window" % ctrl)
        if ctrl < 500:
            fail.append("the control belt delivered %d items; nothing in this save "
                        "was flowing" % ctrl)
        for name, expect in (("m4x4", 4.0), ("m3to5", 3.0)):
            per = delta.get(name)
            if not per:
                fail.append("no samples for rig %s" % name)
                continue
            total = sum(per)
            ratio = total / ctrl if ctrl else 0.0
            spread = (max(per) - min(per)) / (sum(per) / len(per)) if total else 1.0
            print("    %-6s %s  total %.3fx one belt, spread %.2f%%"
                  % (name, " ".join(str(v) for v in per), ratio, spread * 100))
            if abs(ratio - expect) > RATE_TOL * expect:
                fail.append("%s delivered %.3fx one belt and should deliver %.3fx: "
                            "the adopted network is not the shape the world implies"
                            % (name, ratio, expect))
            if spread > SPREAD:
                fail.append("%s outputs spread %.2f%%, over the %.0f%% bound"
                            % (name, spread * 100, SPREAD * 100))

    # The BUILD path, in phase two, long after any scan. In these three legs the
    # prototype is this mod's own stub, so one placed by a script (or revived
    # from a ghost by a robot) must become one of ours.
    late = find_all(run, LATE)
    if not late:
        fail.append("the late build never ran")
    else:
        lg, ou = int(late[-1].group(1)), int(late[-1].group(2))
        print("  a balancer-part BUILT in phase two: legacy=%d ours=%d" % (lg, ou))
        if (lg, ou) != (0, 1):
            fail.append("a `balancer-part` built after the migration came out "
                        "legacy=%d ours=%d; the build path did not swap it"
                        % (lg, ou))

    # And the registry agrees with the world.
    if not audits:
        fail.append("no audit ran after the migration")
    else:
        c, p, nets, drift, unbuilt = (int(g) for g in audits[-1].groups())
        print("  final audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d"
              % (c, p, nets, drift, unbuilt))
        if drift or unbuilt:
            fail.append("the audit found drift=%d unbuilt=%d after the migration"
                        % (drift, unbuilt))
        if nets != c:
            fail.append("%d clusters and %d networks: a cluster the classifier did "
                        "not see" % (c, nets))
        if p != legacy_before:
            fail.append("the audit counted %d parts and %d were adopted"
                        % (p, legacy_before))
        if clusters is not None and c != clusters:
            fail.append("the migration reported %d clusters and the audit finds %d"
                        % (clusters, c))

    for m in find_all(run, BLOCKED) + (find_all(run, ADOPTED) if leg == "built" else []):
        fail.append("the reload decided the migration again: %s"
                    % m.group(0).strip()[-90:])

    if leg == "built":
        report(fail, "every part was swapped after its own build event, and the "
                     "reload decided nothing again")
    else:
        report(fail, "the incumbent's balancers were adopted and they balance")


def report(fail, ok_message):
    if fail:
        print("\nMIGRATION ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\n" + ok_message)


if __name__ == "__main__":
    main()
