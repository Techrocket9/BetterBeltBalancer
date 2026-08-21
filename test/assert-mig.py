#!/usr/bin/env python3
"""Assert that a Belt Balancer save is adopted when the incumbent is removed.

Seven legs and a name probe, run by test/run.sh, each a pair of Factorio phases
(the probe is one phase) and most of them under DIFFERENT MOD SETS. The legs
cover two axes, and neither is a subset of the other: WHICH MOD owns
`balancer-part`, and WHICH TRANSITION of guest/go/legacy.go's state machine the
load makes.

    --leg added    the incumbent swapped out and this mod in, in one edit
    --leg later    this mod installed first, the incumbent removed a session on
    --leg bb3      the same shape with BELT BALANCER 3, the live successor. It
                   is the coexistence shape rather than the swap shape because a
                   misspelled name degrades into the STRANGER path -- Blocked
                   with no log line -- so the conversion would still happen and
                   only the NAMED blocked line pins the row of the name list
    --leg built    no incumbent ever, and the legacy parts arrive one at a time
                   through BUILD EVENTS -- which is what an old blueprint's
                   ghosts do. This leg is also the only one whose save is
                   written AFTER the conversion, so its second phase is a plain
                   reload that must do nothing at all
    --leg readd    DONE -> BLOCKED: this mod first, an incumbent INSTALLED beside
                   it afterwards, on a save that has already converted. The
                   standing balancers are ours and must stay ours and keep
                   running; nothing may be converted; and the late build, which
                   now places the INCUMBENT'S `balancer-part`, must be left
                   exactly where it is
    --leg foreign  a mod that is NOT any fork of the incumbent owns
                   `balancer-part`, and nothing of its may be touched
    --leg fgone    ...and then THAT mod is uninstalled. `legacyCheck` promises
                   the stranger the same thing it promises the incumbents: the
                   load on which they leave is the load on which our stub
                   appears and their balancers become ours
    --leg probe    one create phase, and the only question is whether the guest
                   recognised this incumbent BY NAME

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
  * THE ITEM STACK SURVIVED, and places whatever the mod set says it should.
  * THE FORCE CAN STILL CRAFT. The incumbent's technologies went with it.
  * THE ADOPTED BALANCERS BALANCE, against a bare express belt in the same save.
    A network adopted from the wrong edge list does not show up as a crash; it
    shows up as a rate.
  * AND THE REGISTRY AGREES WITH THE WORLD afterwards: drift=0, unbuilt=0, and
    one network per cluster.

    python3 test/assert-mig.py --leg added create.log run.log
    python3 test/assert-mig.py --leg probe --incumbent belt-balancer \
        --version 3.4.4 create.log
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
# this mod is new to the save; the rest get on_configuration_changed, because it
# was already there and only the mod set moved. Anything else -- and in
# particular `first-dispatch` or `deferred`, the fallbacks -- means the load-time
# hook did not fire and the conversion happened late, on a tick nobody chose.
EXPECT_TRIGGER = {
    "added": "init",
    "later": "configuration_changed",
    "bb3": "configuration_changed",
    "fgone": "configuration_changed",
}

# The legs whose phase one has this mod and an INCUMBENT installed side by side,
# which is where the "is active; its balancers are left alone" line is written.
COEXIST = ("later", "bb3")

# Which incumbent the blocked line must NAME, and at which version, for the legs
# whose whole point is the name. `later` is deliberately not in this table: it
# predates the parameterized stand-in and its assertions are left as they were.
EXPECT_BLOCKED = {
    "bb3": ("belt-balancer-3", "1.0.1"),
    "readd": ("belt-balancer-2", "2.0.9"),
}

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


# --------------------------------------------------------------------------
# The checks several legs share. Each one is exactly the body it was written as
# inside the added/later path; they are functions so that a leg which needs the
# same statement does not get a copy that can drift away from it.
# --------------------------------------------------------------------------

def check_witness(counts, fail, phases=("t1", "post-audit", "final")):
    """The conservation witness: exact, across every surface, including the
    hidden one that did not exist in phase one."""
    base = counts.get("create")
    if base is None:
        fail.append("no copper count was taken in phase one")
        return
    if base < 16:
        fail.append("the witness rig held only %d copper plates; a conservation "
                    "claim over that is vacuous" % base)
        return
    print("  witness: %d copper plates in phase one" % base)
    for phase in phases:
        got = counts.get(phase)
        if got is None:
            continue
        if got != base:
            fail.append("copper %d -> %d at phase=%s: the swap, or the first "
                        "compile after it, lost items off the belts"
                        % (base, got, phase))
        else:
            print("  witness: %d at phase=%s" % (got, phase))


def check_item(items, fail, expect_place):
    """The item half: a stack of `balancer-part` in a chest survives the mod set
    moving, and places whatever the surviving prototype says it places."""
    if "create" not in items or "t1" not in items:
        return
    held0, place0 = items["create"]
    held1, place1 = items["t1"]
    print("  legacy item: %d held, place_result %s -> %s" % (held1, place0, place1))
    if held1 != held0:
        fail.append("the stack of the incumbent's item went %d -> %d across the "
                    "swap; a removed prototype takes its items with it and the "
                    "stub is what stops that" % (held0, held1))
    if place1 != expect_place:
        fail.append("a surviving legacy stack places %r and should place %r"
                    % (place1, expect_place))


def check_throughput(run, fail):
    """Throughput over the window, against the control belt in the same save."""
    samples = {int(m.group(1)): parse_sample(m.group(2)) for m in find_all(run, SAMPLE)}
    if 1800 not in samples or 3540 not in samples:
        fail.append("the throughput window was not sampled at both ends")
        return
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


def check_audit(audits, fail, expect_parts, expect_clusters):
    """And the registry agrees with the world."""
    if not audits:
        fail.append("no audit ran after the migration")
        return
    c, p, nets, drift, unbuilt = (int(g) for g in audits[-1].groups())
    print("  final audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d"
          % (c, p, nets, drift, unbuilt))
    if drift or unbuilt:
        fail.append("the audit found drift=%d unbuilt=%d after the migration"
                    % (drift, unbuilt))
    if nets != c:
        fail.append("%d clusters and %d networks: a cluster the classifier did "
                    "not see" % (c, nets))
    if p != expect_parts:
        fail.append("the audit counted %d parts and %d were adopted"
                    % (p, expect_parts))
    if expect_clusters is not None and c != expect_clusters:
        fail.append("the migration reported %d clusters and the audit finds %d"
                    % (expect_clusters, c))


def check_blocked_name(blocked, leg, fail):
    """The blocked line NAMES the mod, and that is the only thing in this suite
    that can see a row of `legacyIncumbents`.

    A name the guest does not recognise does not fail loudly: it takes the
    stranger path, which is Blocked and SILENT, so every conversion assertion
    would go on passing while a real mod stopped being recognised."""
    want = EXPECT_BLOCKED.get(leg)
    if want is None or not blocked:
        return
    got = (blocked[0].group(1), blocked[0].group(2))
    if got != want:
        fail.append("the guest said %s %s is active and this leg installed %s %s: "
                    "an incumbent it does not recognise by name takes the SILENT "
                    "stranger path instead" % (got[0], got[1], want[0], want[1]))


# --------------------------------------------------------------------------

def probe(create, name, version):
    """A NAME PROBE: one create phase, and the only question is whether the
    guest recognised this incumbent BY NAME.

    Cheap on purpose. `legacyIncumbents` has four rows and two of them have no
    leg of their own, because what a second full leg would add over the first is
    nothing -- the conversion side of the feature is identical whichever name
    blocked it. What is NOT identical is the row of the name list, and a typo
    there degrades SILENTLY: an unrecognised mod owning `balancer-part` takes
    the stranger path, which is Blocked with no log line at all. The named
    blocked line is the only observable that pins the row, so it is the only
    thing this probe asserts -- over a world that really does contain balancers
    the guest declined to touch, which is the anti-vacuity half."""
    fail = []
    census = {m.group(1): (int(m.group(2)), int(m.group(3)))
              for m in find_all(create, CENSUS)}
    blocked = find_all(create, BLOCKED)
    adopted = find_all(create, ADOPTED)

    if "create" not in census:
        print("no census at all: the observer mod did not run")
        sys.exit(1)
    legacy, ours = census["create"]
    print("  %s %s installed beside this mod: %d %s standing, %d of ours"
          % (name, version, legacy, "balancer-part", ours))
    if legacy < 8:
        fail.append("only %d incumbent parts were built; a 'nothing was "
                    "converted' claim over that is vacuous" % legacy)
    if ours != 0:
        fail.append("%d of this mod's parts already exist; the probe is not "
                    "looking at an untouched incumbent world" % ours)

    if len(blocked) != 1:
        fail.append("the guest said it was leaving the incumbent alone %d times "
                    "and it is once per decision -- and ZERO means it did not "
                    "recognise %s at all, which is the silent stranger path"
                    % (len(blocked), name))
    else:
        got = (blocked[0].group(1), blocked[0].group(2))
        print("  %s" % blocked[0].group(0).strip())
        if got != (name, version):
            fail.append("the guest named %s %s and this probe installed %s %s"
                        % (got[0], got[1], name, version))
    if adopted:
        fail.append("something was converted while %s was still installed, which "
                    "is the one thing this feature promises not to do" % name)

    report(fail, "%s is recognised by name and its balancers were left alone" % name)


def readd(create, run, census, counts, items, techs, blocked, adopted, audits,
          legacy_before, fail):
    """DONE -> BLOCKED: an incumbent INSTALLED after this mod, on a save this mod
    has already converted.

    This is the transition `fk_on_configuration_changed` exists for that no leg
    drove until it existed. Phase one is the `built` leg's: no incumbent, so
    `balancer-part` is our own stub, the observer's parts arrive through build
    events, and the save is written with the phase Done. Then belt-balancer-2
    arrives.

    THE PROBE WITH TEETH IS THE LATE BUILD. It places `balancer-part`, which is
    now the INCUMBENT'S prototype, and `legacyBuilt` is gated on the phase being
    Done. A gate reading the wrong phase does not crash and loses no items: it
    silently swaps a working mod's freshly built entity out from under it, and
    the only thing that can see it is this probe."""
    built = find_all(create, BUILT)
    print("  phase one: %d parts swapped after arriving through a build event"
          % len(built))
    if len(built) != legacy_before:
        fail.append("%d parts were placed in phase one and %d were swapped "
                    "through the build path" % (legacy_before, len(built)))
    if adopted:
        fail.append("the whole-world scan converted something (%s). In phase one "
                    "there is nothing for it to find, and in phase two an "
                    "incumbent is installed and nothing may be touched at all"
                    % adopted[0].group(0).strip())
    if find_all(create, BLOCKED):
        fail.append("phase one had no incumbent installed and the guest said one "
                    "was active")
    if len(blocked) != 1:
        fail.append("the incumbent arrived and the guest said it was leaving it "
                    "alone %d times; it is once per decision, and the audits in "
                    "phase two are each a door onto the re-arm" % len(blocked))
    else:
        print("  the incumbent arrives: %s" % blocked[0].group(0).strip())
    check_blocked_name(blocked, "readd", fail)
    if find_all(run, BUILT):
        fail.append("a `balancer-part` was converted in phase two, with an "
                    "incumbent installed")

    # THE WORLD IS OURS AND STAYS OURS. Phase one converted every part, so the
    # incumbent arriving must not move a single entity in either direction.
    for phase in ("t1", "post-audit", "final"):
        after = census.get(phase)
        if after is None:
            fail.append("no census for phase=%s" % phase)
            continue
        print("  phase=%-10s balancer-part=%d bbb-balancer-part=%d"
              % (phase, after[0], after[1]))
        if after != (0, legacy_before):
            fail.append("the world moved at phase=%s: balancer-part 0 -> %d and "
                        "ours %d -> %d" % (phase, after[0], legacy_before, after[1]))

    check_witness(counts, fail)

    # THE ITEM GOES THE OTHER WAY IN THIS LEG, and that is the observation
    # rather than an accident: phase one's stack is our own stub's, and once
    # belt-balancer-2 is installed the prototype it places is the incumbent's
    # again. A stack that still placed `bbb-balancer-part` would mean our stub
    # had won a name the incumbent owns.
    check_item(items, fail, "balancer-part")

    # THE TECHNOLOGY HALF IS THIS LEG'S OWN. Everywhere else `belt-balancer-1`
    # must be ABSENT, because the incumbent was removed; here it must be PRESENT
    # and unresearched, because the incumbent just arrived -- and `bbb-balancer`
    # must still be researched from the phase-one conversion, because nothing
    # about an incumbent arriving takes a technology away.
    for phase, want in (("create", ("false", "absent")), ("t1", ("true", "false"))):
        got = techs.get(phase)
        if got is None:
            fail.append("no technology line for phase=%s" % phase)
            continue
        print("  technology at phase=%-6s bbb-balancer=%s belt-balancer-1=%s"
              % (phase, got[0], got[1]))
        if got != want:
            fail.append("technology at phase=%s is bbb-balancer=%s "
                        "belt-balancer-1=%s and should be %s / %s"
                        % (phase, got[0], got[1], want[0], want[1]))

    # THE STANDING NETWORKS KEEP RUNNING across the incumbent's arrival.
    check_throughput(run, fail)
    check_audit(audits, fail, legacy_before, None)

    # AND THE ONE WITH TEETH.
    late = find_all(run, LATE)
    if not late:
        fail.append("the late build never ran; the one probe that can see the "
                    "build path's phase gate did not happen")
    else:
        lg, ou = int(late[-1].group(1)), int(late[-1].group(2))
        print("  the INCUMBENT'S balancer-part BUILT in phase two: legacy=%d ours=%d"
              % (lg, ou))
        if (lg, ou) != (1, 0):
            fail.append("a `balancer-part` built while belt-balancer-2 is "
                        "installed came out legacy=%d ours=%d: the build path is "
                        "gated on the phase being Done and it swapped a working "
                        "mod's entity out from under it" % (lg, ou))

    report(fail, "the incumbent arrived, nothing was converted, and the "
                 "balancers this mod already owns keep running")


def main():
    args = sys.argv[1:]
    leg = "added"
    inc_name = inc_version = None
    while args and args[0].startswith("--"):
        if args[0] == "--leg":
            leg, args = args[1], args[2:]
        elif args[0] == "--incumbent":
            inc_name, args = args[1], args[2:]
        elif args[0] == "--version":
            inc_version, args = args[1], args[2:]
        else:
            print("unknown option %r" % args[0])
            sys.exit(2)

    create, run = [], []
    for path, into in zip(args[:2], (create, run)):
        with open(path, errors="replace") as f:
            into.extend(f)
    both = create + run

    if leg == "probe":
        if not (inc_name and inc_version):
            print("--leg probe needs --incumbent NAME --version VER")
            sys.exit(2)
        probe(create, inc_name, inc_version)
        return

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

    # ------------------------------------------------------------------ readd
    if leg == "readd":
        readd(create, run, census, counts, items, techs, blocked, adopted, audits,
              legacy_before, fail)
        return

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

    # ------------------------------------------------ added, later, bb3, fgone
    if leg in COEXIST:
        # EXACTLY ONE, and that is the latch rather than tidiness: the create
        # phase places an audit marker, which is a door onto the re-arm, and a
        # Blocked state that re-decided on every audit would say so again here.
        if len(blocked) != 1:
            fail.append("phase one had both mods installed and the guest said it "
                        "was leaving the incumbent alone %d times; it is once per "
                        "decision" % len(blocked))
        else:
            print("  coexistence: %s" % blocked[0].group(0).strip())
        check_blocked_name(blocked, leg, fail)
        if find_all(create, ADOPTED):
            fail.append("something was converted while the incumbent was still "
                        "installed, which is the one thing this feature promises "
                        "not to do")

    if leg == "fgone":
        # THE STRANGER'S PHASE ONE IS SILENT, and that is the whole difference
        # between it and an incumbent's. `legacyCheck` writes a named line for a
        # mod it recognises and nothing at all for one it does not, so a blocked
        # line here would mean the guest had decided bbb-mig-foreign is a Belt
        # Balancer.
        if find_all(create, BLOCKED):
            fail.append("the guest reported an incumbent as active in phase one; "
                        "bbb-mig-foreign is not one and the stranger branch is "
                        "silent by design")
        if find_all(create, ADOPTED):
            fail.append("a stranger's `balancer-part` entities were converted "
                        "while that stranger was still installed")
        if find_all(create, BUILT):
            fail.append("a `balancer-part` was swapped through the build path "
                        "while the stranger owned the prototype")
        # AND STOP HERE IF ANY OF THOSE FIRED. This leg is the only one that has
        # this mod and the stranger installed side by side while the observer is
        # BUILDING the stranger's entities -- the `foreign` leg installs this mod
        # only in phase two, so its phase one has no guest at all -- and a guest
        # that converted them in phase one leaves phase two with nothing left to
        # adopt. That reaches the "nothing was adopted at all" diagnostic below,
        # which is about the data stage and would send the next reader to the
        # wrong file.
        if fail:
            report(fail, "")

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

    check_witness(counts, fail)
    check_item(items, fail, "bbb-balancer-part")

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

    if leg == "fgone":
        # `belt-balancer-1` is absent in BOTH phases of this leg -- the stranger
        # never defined one -- so the check above says nothing here and would
        # pass on a guest that granted nothing. What IS a statement is that the
        # technology was granted BY the conversion: unresearched while the
        # stranger stood, researched once its balancers became ours.
        before = techs.get("create")
        if before is None:
            fail.append("no technology line for phase=create")
        else:
            print("  technology before the stranger left: bbb-balancer=%s "
                  "belt-balancer-1=%s" % (before[0], before[1]))
            if before != ("false", "absent"):
                fail.append("technology at phase=create is bbb-balancer=%s "
                            "belt-balancer-1=%s and should be false / absent -- "
                            "this mod is installed and has converted nothing yet"
                            % (before[0], before[1]))

    check_throughput(run, fail)

    # The BUILD path, in phase two, long after any scan. In these legs the
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

    check_audit(audits, fail, legacy_before, clusters)

    for m in find_all(run, BLOCKED) + (find_all(run, ADOPTED) if leg == "built" else []):
        fail.append("the reload decided the migration again: %s"
                    % m.group(0).strip()[-90:])

    if leg == "built":
        report(fail, "every part was swapped after its own build event, and the "
                     "reload decided nothing again")
    elif leg == "fgone":
        report(fail, "the stranger was uninstalled and its balancers became ours")
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
