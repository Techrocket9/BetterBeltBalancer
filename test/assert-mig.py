#!/usr/bin/env python3
"""Assert what a Belt Balancer save becomes when the incumbent is removed.

ON FACTORIO 2.1 THIS SUITE ASSERTS A DIFFERENT OUTCOME THAN IT USED TO, AND
NOTHING ABOUT THE CONVERSION CHANGED TO MAKE IT SO. `legacy.go` is untouched by
the single-edge port: parts still convert at their health and their quality, the
technology is still granted per force, item stacks still survive and still place
this mod's parts, and the items on the players' own belts are conserved exactly.
What changed is the engine underneath. 2.1 allows ONE BELT PER BALANCER PART
(agents/single-edge.md), and Belt Balancer's own idiom is a single column of
parts with a belt on every free face -- two belts per part, by construction. So
an incumbent balancer converts and is then REFUSED, and this suite's throughput
assertions invert: the rigs laid the incumbent's way must deliver NOTHING, and
the `sok` band -- laid two columns wide, which a Belt Balancer user could
genuinely have -- must deliver at rate. That is the honest portal story in one
world, and it is asserted as plainly as the old rates were rather than by
loosening a bound.

Seven legs and two name probes, run by test/run.sh, each a pair of Factorio
phases (the probes are one phase) and most of them under DIFFERENT MOD SETS. The
legs cover two axes, and neither is a subset of the other: WHICH MOD owns
`balancer-part`, and WHICH TRANSITION of guest/go/legacy.go's state machine the
load makes. Both axes survive the port untouched -- the state machine knows
nothing about belts.

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
  * THE ONES THIS ENGINE CAN BUILD BALANCE, against a bare express belt in the
    same save. A network adopted from the wrong edge list does not show up as a
    crash; it shows up as a rate. Only the `sok` band can reach this now, and
    that is what it is for.
  * AND THE ONES IT CANNOT ARE REFUSED, DELIVER EXACTLY ZERO, AND SAY SO. One
    refusal per cluster, naming how many of its parts carry more than one belt
    -- a count that is neither zero nor the whole cluster on `m3to5`, which is
    what separates a real classification from a constant. Nothing is torn down
    and nothing is spilled: a freshly converted cluster never had a network, so
    the refusal in front of the teardown has no teardown to be in front of.
  * WHAT THE CONVERSION CARRIED. `legacyConvertOne` reads a HEALTH and a QUALITY
    off the entity it destroys and writes both onto the one it creates, and both
    are invisible on an undamaged normal-quality part -- which is what every
    part in this suite was until the fidelity rig existed.
  * WHOSE THE BALANCERS ARE. Two forces' parts TOUCHING are two balancers, and
    the technology is granted per force. A fusion moves no item count anywhere.
  * AND WHERE THEY ARE. The scan walks every surface; the summary line's own
    surface number counts surfaces SCANNED rather than surfaces that had parts
    on them, so the per-surface census is what can see a scan that stopped
    early.
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
TECHF = re.compile(r"\[BBB-MIG\] tech-force phase=(\S+) force=(\S+) bbb-balancer=(\S+)")
HEALTH = re.compile(
    r"\[BBB-MIG\] health phase=(\S+) name=(\S+) value=(\S+) max=(\S+)")
QUALITY = re.compile(r"\[BBB-MIG\] quality phase=(\S+) name=(\S+) value=(\S+)")
SURFACES = re.compile(r"\[BBB-MIG\] surfaces phase=(\S+) (.*)")
FORCEPARTS = re.compile(r"\[BBB-MIG\] force-parts phase=(\S+) (.*)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)"
)
REBUILT = re.compile(
    r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters "
    r"\((\d+) networks adopted, (\d+) rebuilt\)"
)

# THE REFUSAL, and everything that is said about it.
#
# `alert:` and never `error:` -- test/run.sh fails a run on an error line, and a
# balancer built to a rule this Factorio does not permit is an expected condition
# with a defined outcome (guest/go/sedge.go, logRefusedSingleEdge).
REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt, "
    r"worst (\d+)"
)
# The ORDINARY per-cluster refusal message: what a robot or a script that placed
# a piece is told. See EXPECT_SUMMARY below for why a migration reaches it.
TOLDPIECE = re.compile(
    r"\[BBB\] told force (\d+) that cluster (\d+) is past the one-belt-per-part"
)
# The MIGRATION summary, which is a different act: one line per affected force,
# with a `[gps=]` ping per balancer, for a whole save that was built to a rule
# that has since changed.
SUMMARY = re.compile(
    r"\[BBB\] single-edge: (\d+) balancers were built with several belts per part"
)
SUMTOLD = re.compile(
    r"\[BBB\] single-edge: told force (\d+) about (\d+) balancers built to the "
    # No `$`: this is matched per line, and the trailing group is what catches
    # the ` -- print FAILED` suffix.
    r"multi-edge rule(.*)"
)
# THE GRANDFATHER PASS, which is the 2.0 arm and must never be attempted here:
# `bbb-multi-edge-parts` is not defined on 2.1 and writing an undefined
# settings.global key RAISES.
GRANDFATHER = re.compile(r"\[BBB\] single-edge: kept multiple belts per part enabled")
FLIPPED = re.compile(r"\[BBB\] single-edge: multiple belts per part turned (ON|OFF)")
# EITHER ARM OF THE HAND-BACK, matched on the shape both of them share rather
# than on one sentence. A NEGATIVE assertion, and an exact regex over a negative
# is the one shape a rename in the guest can make VACUOUS -- the line stops
# matching, the assertion stops being able to fail, and nothing says so.
HANDEDBACK = re.compile(r"\[BBB\].*\bpiece at -?\d+,-?\d+")
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")

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

# THE WORLD THE OBSERVER BUILDS, written down HERE and not derived from anything
# the guest said. The cluster count is the one number in this suite that a
# defect can move without moving a single item: two forces' parts touching are
# two balancers, and a flood fill that lost its force check fuses them into one
# while every census, every copper count and every rate stays exactly what it
# was. Cross-checking the audit against the guest's own summary line -- which is
# what this script did until the force rig existed -- cannot see that at all,
# because both numbers come from the same fill.
#
#   m4x4 4 parts / m3to5 5 / sok2 4 / sok4 8 / wit 2 / fid 2 /
#   frc 2+2 on two forces / surface B 2
EXPECT_PARTS = 31
EXPECT_CLUSTERS = 9
EXPECT_FORCES = 2                    # forces that owned a converted part
FORCE_B = "bbb-mig-force-b"
EXPECT_FORCE_B_PARTS = 2
EXPECT_PART_SURFACES = 2             # surfaces that actually carry parts
HIDDEN_SURFACE = "bbb-hidden"        # this mod's own, and never scanned

FID_HEALTH = 85.0
FID_QUALITY = "uncommon"

# WHAT THE ENGINE CAN AND CANNOT BUILD OUT OF THAT WORLD, which is the whole of
# what the single-edge port changed about this suite.
#
# Seven of the nine clusters are laid the incumbent's way -- one column of parts
# with a belt on both free faces of each row -- so every one of their parts
# carries two belts and every one of them is refused. Two are the `sok` band,
# laid two columns wide, and they compile and run.
EXPECT_NETS = 2                      # sok2 and sok4, and nothing else
EXPECT_REFUSED = 7

# HOW MANY PARTS OF EACH REFUSED CLUSTER CARRY MORE THAN ONE BELT, sorted. This
# is the assertion that says the classification is real rather than a constant:
# `m3to5` has three inputs and five outputs over five parts, so three of its
# parts carry two belts and two carry one, and nothing else in the world has a
# count that is neither zero nor its whole size.
#
#   m4x4 4 / m3to5 3 / wit 2 / fid 2 / frc 2 / frc 2 / surface B 2
EXPECT_REFUSED_SHAPE = [2, 2, 2, 2, 2, 3, 4]

# The rigs the conversion must leave RUNNING, and what each owes against the
# control belt. Two of them rather than one because one rig delivering its rate
# could be an accident of a network with no stages in it: `sok4` is P=4, a real
# butterfly, re-derived from a converted world.
WORKING = (("sok2", 2.0), ("sok4", 4.0))

# ...and the rigs that must deliver EXACTLY NOTHING. Asserted as zero rather
# than as a loosened bound: a refused cluster has no network at all, so an item
# arriving in one of these chests is not a rate that drifted, it is a balancer
# that was built when it should not have been.
STOPPED = ("m4x4", "m3to5")

# WHICH LEGS SPEAK THE MIGRATION SUMMARY: ALL OF THEM, ONCE EACH, and none of
# them reaches the ordinary per-piece message at all.
#
# THIS TABLE USED TO SAY SOMETHING ELSE, AND WHAT IT SAID WAS A MEASUREMENT OF A
# DEFECT. `sedgeAnnounce` -- the roots whose refusal belongs in the summary
# rather than in the per-piece message -- had two producers and both of them
# were `rebuildFromWorld`; a LEGACY CONVERSION is neither, so only `added` spoke
# the summary. There, and only there, a rebuild follows the conversion in the
# same load: this mod is new to the save, so the `fk_on_configuration_changed`
# after `fk_on_init` finds registryReady false and rebuilds. The other five legs
# handed the player seven copies of a sentence about an extra piece being left in
# place unconnected, when nobody had placed anything -- the defect class
# `mig21`'s third red proof exists for, reached through the other door -- and
# `added` gave them both.
#
# The conversion is the THIRD PRODUCER since 2026-08-24 (guest/go/sedge.go's
# refuseSingleEdge, over guest/go/legacy.go's legacyConvertedRoot), so every leg
# that converts speaks the checklist and no leg speaks the per-piece copy.
# `added` speaks it ONCE rather than twice: the rebuild that follows the
# conversion finds the refusal already delivered and does not re-announce it
# (guest/go/limit.go, refusalDelivered).
#
# BOTH DIRECTIONS ARE STILL PINNED. A leg that stops speaking fails, a leg that
# speaks twice fails, and a single per-piece line anywhere fails -- which is the
# assertion the fix had to be able to move.
EXPECT_SUMMARY = {
    "added": True,
    "later": True,
    "bb3": True,
    "built": True,
    "readd": True,
    "fgone": True,
}

# ...AND HOW MANY ORDINARY PER-PIECE MESSAGES A CONVERSION MAY PRODUCE, which is
# the other half of the same statement and is its own constant so that a
# regression in either direction has a number to move. Zero: nobody placed a
# piece, so no cluster in this suite may be told one was left in place.
EXPECT_TOLDPIECE = 0


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
#
# A MISSING LINE IS A FAILURE AND NOT A SKIP, in every one of them. Three of
# these used to step over an absent phase -- `continue` in the witness and the
# fidelity loops, an early `return` in the item check -- so a phase the observer
# stopped reporting took its whole assertion with it, silently, and the leg went
# on printing the phases that were still there and exiting zero. Every one of
# these lines is written unconditionally by the observer at every phase named
# here, so an absent one is a broken harness rather than a legitimate shape, and
# the run that says so is worth more than the run that does not.
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
            fail.append("no copper count for phase=%s: the conservation claim is "
                        "made of these lines, so a missing one is the claim not "
                        "being made" % phase)
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
    missing = [p for p in ("create", "t1") if p not in items]
    if missing:
        fail.append("no legacy-item line for phase=%s; the stack is only ever "
                    "compared against itself, so a missing end is a check that "
                    "did not happen" % ", ".join(missing))
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
    """Throughput over the window, against the control belt in the same save.

    TWO STATEMENTS NOW, and the second is as sharp as the first. The `sok` band
    was laid one belt per part, so this engine can build it and it must deliver
    at rate; the incumbent-idiom rigs put two belts on every part, so this engine
    refuses them and they must deliver NOTHING AT ALL. Asserting the second as a
    zero rather than as a loosened bound is the point: a refused cluster has no
    network, so an item arriving in one of those chests would be a balancer that
    got built when the rule says it cannot be."""
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
    for name, expect in WORKING:
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
            fail.append("%s delivered %.3fx one belt and should deliver %.3fx: it "
                        "is laid one belt per part, so this engine can build it "
                        "and the conversion has to leave it RUNNING -- this is "
                        "the only rig band in the suite that still proves an "
                        "adopted balancer balances at all"
                        % (name, ratio, expect))
        if spread > SPREAD:
            fail.append("%s outputs spread %.2f%%, over the %.0f%% bound"
                        % (name, spread * 100, SPREAD * 100))
    for name in STOPPED:
        per = delta.get(name)
        if per is None:
            fail.append("no samples for rig %s" % name)
            continue
        print("    %-6s %s  -- refused, and a refused cluster has no network"
              % (name, " ".join(str(v) for v in per)))
        if any(v != 0 for v in per):
            fail.append("%s delivered %s items and it is built the incumbent's "
                        "way -- two belts on one part, which this Factorio "
                        "refuses. Nothing should have been compiled for it"
                        % (name, per))


def check_audit(audits, fail, expect_parts, summary_clusters):
    """And the registry agrees with the world.

    THE WHOLE TUPLE IS WRITTEN DOWN HERE rather than cross-checked against
    anything the guest said, which is the estate's own convention and is why the
    cluster count was taken off the summary line in the first place: a flood fill
    that fused two forces' touching parts moves the audit and the summary
    together and neither says anything.

    `nets` IS NO LONGER `clusters` AND THAT IS THE PORT, not a regression. Seven
    of the nine clusters are refused and a refused cluster never gets a network,
    so the two numbers are 2 and 9 and `nets == clusters` would now be the
    failing state. `refused` is what tells a declined cluster apart from one the
    classifier never saw -- `unbuilt` is this guest saying it should have built
    something and did not, and it stays zero throughout."""
    if not audits:
        fail.append("no audit ran after the migration")
        return
    got = tuple(int(g) for g in audits[-1].groups())
    want = (EXPECT_CLUSTERS, expect_parts, EXPECT_NETS, 0, 0, EXPECT_REFUSED)
    print("  final audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d "
          "refused=%d" % got)
    if got != want:
        fail.append("the audit reads %s and should read %s: %d clusters over %d "
                    "parts, of which the %d laid one belt per part carry a "
                    "network and the %d laid the incumbent's way are refused, "
                    "with nothing drifting and nothing left unbuilt"
                    % (got, want, EXPECT_CLUSTERS, expect_parts, EXPECT_NETS,
                       EXPECT_REFUSED))
    if summary_clusters is not None and got[0] != summary_clusters:
        fail.append("the migration reported %d clusters and the audit finds %d"
                    % (summary_clusters, got[0]))


def check_refusals(lines, leg, fail):
    """THE REFUSALS, AND WHAT WAS SAID ABOUT THEM.

    Seven clusters in this world are built the incumbent's way, and every one of
    them has to be refused BY NAME with a count of its offending parts. The count
    is what makes this a statement about a classification rather than about a
    constant: `m3to5` is three inputs and five outputs over five parts, so three
    of them carry two belts and two carry one.

    AND NOTHING IS TORN DOWN AND NOTHING IS SPILLED. That is the difference
    between this suite and `mig21`, which asserts the opposite for the same
    refusal: there the balancers were STANDING when the save opened, so the
    remnant had to come down and everything it held reached the ground. Here the
    clusters are seconds old -- `legacyScan` created the parts and the very next
    flush refused them -- so there is no network to tear down, no carry pool to
    open and nothing to spill. The items are where they always were, on the
    player's own belts, which is what the copper witness measures from the other
    side."""
    seen = {}
    for m in find_all(lines, REFUSED):
        root = int(m.group(1))
        if root not in seen:
            seen[root] = (int(m.group(2)), int(m.group(3)))
    shape = sorted(v[0] for v in seen.values())
    print("  refused: %d clusters, parts-carrying-two-belts %s"
          % (len(seen), shape))
    if len(seen) != EXPECT_REFUSED:
        fail.append("%d clusters were refused and %d of this world's %d are laid "
                    "the incumbent's way, two belts on every part"
                    % (len(seen), EXPECT_REFUSED, EXPECT_CLUSTERS))
    if shape != EXPECT_REFUSED_SHAPE:
        fail.append("the refusals name %s parts carrying more than one belt and "
                    "the rigs imply %s: m3to5 is the one whose count is neither "
                    "zero nor its whole size, so a constant cannot produce this "
                    "list" % (shape, EXPECT_REFUSED_SHAPE))
    worst = sorted({v[1] for v in seen.values()})
    if worst not in ([], [2]):
        fail.append("a refusal reported a part carrying %s belts; the incumbent's "
                    "idiom puts an input on one face and an output on the other, "
                    "which is two" % worst)

    # NOT ONE ORDINARY PER-PIECE MESSAGE, which is the half of this that a player
    # reported. See EXPECT_SUMMARY: a converted balancer is a MIGRATION and gets
    # the checklist; "the extra piece was left in place, unconnected" is a
    # sentence about an event that never happened, because nobody placed anything.
    told = {int(m.group(2)) for m in find_all(lines, TOLDPIECE)}
    print("  told per cluster: %d" % len(told))
    if len(told) != EXPECT_TOLDPIECE:
        fail.append("the ORDINARY per-piece refusal message went to clusters %s "
                    "and a conversion must produce %d of them: nobody placed a "
                    "piece, so nothing may be told one was left in place "
                    "unconnected" % (sorted(told), EXPECT_TOLDPIECE))

    # THE MIGRATION SUMMARY, in both directions.
    summ = find_all(lines, SUMMARY)
    sumtold = find_all(lines, SUMTOLD)
    want_summary = EXPECT_SUMMARY.get(leg, False)
    if want_summary:
        if len(summ) != 1:
            fail.append("the migration summary was spoken %d times and a "
                        "conversion speaks it exactly once: the flush that "
                        "refuses the converted clusters is the flush that "
                        "summarises them, and a rebuild-from-world following in "
                        "the same load must find the refusal already delivered "
                        "and stay quiet" % len(summ))
        elif int(summ[0].group(1)) != EXPECT_REFUSED:
            fail.append("the migration summary named %s balancers and %d were "
                        "refused" % (summ[0].group(1), EXPECT_REFUSED))
        per_force = {int(m.group(1)): int(m.group(2)) for m in sumtold}
        print("  migration summary told: %s" % per_force)
        if sum(per_force.values()) != EXPECT_REFUSED:
            fail.append("the summary reached forces %s, adding up to %d of %d "
                        "refused balancers -- it is once per FORCE, never once "
                        "per balancer and never once per save"
                        % (per_force, sum(per_force.values()), EXPECT_REFUSED))
        if len(per_force) != EXPECT_FORCES:
            fail.append("the summary reached %d force(s) and the force rig puts "
                        "refused balancers on %d" % (len(per_force), EXPECT_FORCES))
        for m in sumtold:
            if "FAILED" in m.group(3):
                fail.append("the summary to force %s did not reach it"
                            % m.group(1))
        # AND WHICH OF THE TWO MIGRATION SENTENCES IT USED. A conversion tears
        # nothing down -- the clusters are seconds old and never had a network --
        # so the summary must NOT be the one that says their contents are on the
        # ground. That claim is true of `mig21`'s fixtures, where the networks
        # were standing when the save opened, and false here; getting it wrong
        # would be the same defect this whole assertion block is about, moved one
        # sentence to the left.
        for m in summ:
            if "on the ground beside them" in m.group(0):
                fail.append("the summary says the refused balancers' contents "
                            "are on the ground, and a conversion spills nothing: "
                            "these clusters never had a network to tear down, "
                            "which is what the 0 teardowns and 0 spills below say")
    elif summ or sumtold:
        fail.append("the migration summary was spoken in a leg that should not "
                    "speak it (%s)" % (summ or sumtold)[0].group(0).strip()[-80:])

    # NOTHING WAS STANDING, SO NOTHING CAME DOWN.
    torn = find_all(lines, TORNDOWN)
    spilled = find_all(lines, SPILLED)
    if torn or spilled:
        fail.append("%d teardowns and %d spills: a cluster the conversion just "
                    "created has never had a network, so the refusal in front of "
                    "the teardown has no teardown to be in front of and nothing "
                    "of the player's can reach the ground"
                    % (len(torn), len(spilled)))

    # AND THE TWO NEGATIVES THIS ENGINE EXISTS TO PIN.
    if find_all(lines, HANDEDBACK):
        fail.append("a piece was handed back to a player: there is no player in "
                    "a headless run, and in a migration nobody placed anything")
    if find_all(lines, GRANDFATHER) or find_all(lines, FLIPPED):
        fail.append("the guest touched `bbb-multi-edge-parts`. It is not defined "
                    "on Factorio 2.1 and writing an undefined settings.global "
                    "key RAISES, so the capability gate has to hold here")


def parse_pairs(text, sep):
    """`a:1/2 b:3/4` -> {'a': '1/2', ...}, and `a=1 b=2` the same way."""
    out = {}
    for chunk in text.split():
        name, _, rest = chunk.partition(sep)
        out[name] = rest
    return out


def check_fidelity(healths, qualities, fail, expect_name, phases=("t1", "final")):
    """WHAT THE CONVERSION CARRIED: the health and the quality.

    `legacyConvertOne` reads both off the entity it is about to destroy and
    writes them onto the one it creates -- two calls, on a path that until the
    fidelity rig existed had only ever been handed undamaged normal-quality
    parts. Both lines could have been absent and every other assertion in this
    suite would have passed unchanged.

    ANTI-VACUITY IS THE WHOLE RISK HERE and it is checked in phase one: a part
    that was never damaged is at max_health, and an equality across the swap is
    then satisfied by a guest that copies nothing at all."""
    base = healths.get("create")
    if base is None:
        fail.append("no health was reported in phase one")
        return
    # `part_at` reports `none` for a tile that holds neither prototype, which is
    # the fidelity rig not having been built at all. Named rather than allowed to
    # reach `float()` and come out as a traceback.
    if base[1] == "none" or base[2] == "none":
        fail.append("the fidelity rig's damaged tile held nothing in phase one "
                    "(name=%s): there is no part to carry a health across the "
                    "swap" % base[0])
        return
    _, value0, max0 = base[0], float(base[1]), float(base[2])
    if value0 >= max0:
        fail.append("the fidelity part was at %.1f of %.1f in phase one: it was "
                    "never damaged, so an equality across the swap is vacuous"
                    % (value0, max0))
    elif value0 != FID_HEALTH:
        fail.append("the fidelity part was damaged to %.1f and the rig asks for "
                    "%.1f" % (value0, FID_HEALTH))
    q0 = qualities.get("create")
    if q0 is None:
        fail.append("no quality was reported in phase one")
        return
    if q0[1] != FID_QUALITY:
        fail.append("the fidelity part was built at quality %r and the rig asks "
                    "for %r -- the quality mod is enabled for both phases of "
                    "every leg of this suite, and a game with only `normal` in "
                    "it cannot tell a guest that carries the quality apart from "
                    "one that drops the key" % (q0[1], FID_QUALITY))
    print("  fidelity in phase one: %.1f of %.1f health, quality %s"
          % (value0, max0, q0[1]))

    for phase in phases:
        got = healths.get(phase)
        if got is None:
            fail.append("no health line for phase=%s: the equality across the "
                        "swap IS this check, so a phase that stopped reporting "
                        "takes the whole of it away" % phase)
            continue
        if got[1] == "none":
            fail.append("the damaged tile held nothing at phase=%s (name=%s): "
                        "the part the conversion was supposed to carry is gone"
                        % (phase, got[0]))
            continue
        name, value, mx = got[0], float(got[1]), float(got[2])
        print("  fidelity at phase=%-6s %s at %.1f of %.1f health"
              % (phase, name, value, mx))
        if name != expect_name:
            fail.append("the damaged tile holds a %s at phase=%s and should "
                        "hold a %s" % (name, phase, expect_name))
        if value != value0:
            fail.append("the damaged part is at %.1f health at phase=%s and was "
                        "at %.1f before the swap: legacyConvertOne reads the "
                        "health off the old entity and writes it onto the new "
                        "one, and a part silently repaired to full is a building "
                        "this mod was not asked to touch"
                        % (value, phase, value0))
        gotq = qualities.get(phase)
        if gotq is None:
            fail.append("no quality line for phase=%s, and the quality is the "
                        "half of this rig that found a defect" % phase)
            continue
        print("  fidelity at phase=%-6s %s at quality %s"
              % (phase, gotq[0], gotq[1]))
        if gotq[0] != expect_name:
            fail.append("the quality tile holds a %s at phase=%s and should hold "
                        "a %s" % (gotq[0], phase, expect_name))
        if gotq[1] != q0[1]:
            fail.append("the %s part came back at quality %r at phase=%s: the "
                        "quality is passed to `create_entity` as the prototype "
                        "HANDLE and a dropped key reads as `normal`"
                        % (q0[1], gotq[1], phase))


def check_forces(techf, forceparts, fail, expect_researched, phases=("t1", "final")):
    """WHOSE THE BALANCERS ARE. The technology is granted per force that owned a
    converted part, and a force left without it is a player holding balancers
    they cannot craft a spare for.

    The parts-per-force line is the anti-vacuity half: a run in which the second
    force's parts were never built would satisfy every other statement here by
    having only one force in it."""
    for phase in ("create",) + tuple(phases):
        got = forceparts.get(phase)
        if got is None:
            fail.append("no parts-per-force line for phase=%s" % phase)
            continue
        n = int(got.get(FORCE_B, -1))
        if n != EXPECT_FORCE_B_PARTS:
            fail.append("the second force owns %d parts at phase=%s and the "
                        "force rig builds %d; a fusion check over one force is "
                        "vacuous" % (n, phase, EXPECT_FORCE_B_PARTS))
    print("  parts per force at phase=t1: %s"
          % " ".join("%s=%s" % kv for kv in sorted(forceparts.get("t1", {}).items())))
    for phase in phases:
        got = techf.get(phase)
        if got is None:
            fail.append("no second-force technology line for phase=%s" % phase)
            continue
        if got != expect_researched:
            fail.append("the second force's bbb-balancer is %s at phase=%s and "
                        "should be %s: the grant is per force that owned a "
                        "converted part" % (got, phase, expect_researched))
        else:
            print("  the second force's technology at phase=%-6s bbb-balancer=%s"
                  % (phase, got))


def check_surfaces(surfs, fail, scanned, expect_converted):
    """AND WHERE THEY ARE.

    THE SUMMARY LINE'S OWN SURFACE NUMBER COUNTS SURFACES SCANNED, not surfaces
    that had anything on them -- `legacyScan` increments it once per non-hidden
    surface before it looks -- so it reads the same whether the parts were on
    one surface or on five, and a scan that stopped after the first surface it
    converted something on would report exactly what it reports now. Two
    statements, therefore, and they are not the same one:

      * every surface in the game was scanned: the number on the summary line
        equals the number of non-hidden surfaces the observer can see;
      * parts on MORE THAN ONE surface were really converted, which only the
        per-surface census can say."""
    base = surfs.get("create")
    if base is None:
        fail.append("no per-surface census in phase one")
        return
    with_parts = [s for s, v in base.items() if int(v.split("/")[0]) > 0]
    if len(with_parts) < EXPECT_PART_SURFACES:
        fail.append("phase one put parts on %d surface(s) and the rigs cover %d; "
                    "a multi-surface claim over one surface is vacuous"
                    % (len(with_parts), EXPECT_PART_SURFACES))
    print("  phase one: parts on %s" % ", ".join(sorted(with_parts)))

    after = surfs.get("t1")
    if after is None:
        fail.append("no per-surface census for phase=t1")
        return
    print("  phase=t1  %s" % " ".join("%s:%s" % kv for kv in after.items()))
    visible = [s for s in after if s != HIDDEN_SURFACE]
    if scanned is not None and len(visible) != scanned:
        fail.append("the scan reports %d surfaces and the world has %d that are "
                    "not the hidden one (%s): that number counts surfaces "
                    "SCANNED, so a scan that skipped one would say so here and "
                    "nowhere else" % (scanned, len(visible), ", ".join(visible)))

    if not expect_converted:
        return
    ours = [s for s, v in after.items() if int(v.split("/")[1]) > 0]
    if len(ours) != EXPECT_PART_SURFACES:
        fail.append("this mod's parts are standing on %d surface(s) after the "
                    "conversion and were built on %d: a scan that stopped after "
                    "the first surface it converted anything on leaves the rest "
                    "as the incumbent's, forever, with nothing logged"
                    % (len(ours), EXPECT_PART_SURFACES))


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
    the guest declined to touch, which is the anti-vacuity half.

    IT WATCHES BOTH DOORS ALL THE SAME, and that is one line rather than a
    second axis. `legacyIncumbents` gates the SCAN through the phase, and the
    phase is also what gates the BUILD path -- and these two names are in front
    of the guest nowhere else in this repo, while the observer builds nineteen
    of the incumbent's entities with the guest listening. Asserting only the
    scan would leave the more expensive of the two failures unwatched on half
    the name list."""
    fail = []
    census = {m.group(1): (int(m.group(2)), int(m.group(3)))
              for m in find_all(create, CENSUS)}
    blocked = find_all(create, BLOCKED)
    adopted = find_all(create, ADOPTED)
    built = find_all(create, BUILT)

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
    if built:
        fail.append("%d `balancer-part` entities were swapped through the BUILD "
                    "path while %s was still installed: `legacyBuilt` is gated on "
                    "the phase being Done, and a gate that let these through "
                    "swaps a working mod's freshly built entity out from under it"
                    % (len(built), name))
    # AND NOTHING WAS REFUSED, which is the same statement one step further on.
    # A refusal names a cluster of OURS, so one here would mean the incumbent's
    # entities had been converted and then declined -- the loudest possible form
    # of touching a working mod's world.
    for m in find_all(create, REFUSED):
        fail.append("a cluster was refused for the one-belt-per-part rule while "
                    "%s was installed (%s): this mod owns no cluster in that save"
                    % (name, m.group(0).strip()[-70:]))

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

    # THE STANDING NETWORKS KEEP RUNNING across the incumbent's arrival, and the
    # refused ones stay refused. Both halves of that are check_throughput's now.
    check_throughput(run, fail)
    check_refusals(create + run, "readd", fail)
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
    techf = {m.group(1): m.group(3) for m in find_all(both, TECHF)}
    healths = {m.group(1): (m.group(2), m.group(3), m.group(4))
               for m in find_all(both, HEALTH)}
    qualities = {m.group(1): (m.group(2), m.group(3)) for m in find_all(both, QUALITY)}
    surfs = {m.group(1): parse_pairs(m.group(2), ":") for m in find_all(both, SURFACES)}
    forceparts = {m.group(1): parse_pairs(m.group(2), "=")
                  for m in find_all(both, FORCEPARTS)}
    adopted = find_all(both, ADOPTED)
    blocked = find_all(both, BLOCKED)
    audits = find_all(run, AUDIT)

    for phase in ("create", "t1"):
        if phase not in census:
            print("no census for phase=%s: the observer mod did not run" % phase)
            sys.exit(1)

    legacy_before, ours_before = census["create"]
    print("  phase one: %d %s standing, %d of ours" % (legacy_before, "balancer-part", ours_before))

    if legacy_before != EXPECT_PARTS:
        fail.append("%d incumbent parts were built and the rigs make %d; every "
                    "count below is against that number, so a world that did not "
                    "go up whole is measured rather than reported"
                    % (legacy_before, EXPECT_PARTS))
    if ours_before != 0:
        fail.append("phase one already had %d of this mod's parts; the leg is not "
                    "measuring a migration" % ours_before)

    # ------------------------------------------------------------------ readd
    if leg == "readd":
        # The conversion happened in PHASE ONE here, through the build path, so
        # the fidelity pair crossed a save as well as a swap and the force grant
        # was legacyRunBuilds' rather than legacyScan's. There is no summary
        # line to read a scanned-surface count off, hence `scanned=None`.
        check_fidelity(healths, qualities, fail, "bbb-balancer-part")
        check_forces(techf, forceparts, fail, "true")
        check_surfaces(surfs, fail, None, True)
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
        if "t1" not in items:
            fail.append("no legacy-item line for phase=t1; the stranger's stack "
                        "is not being looked at")
        elif items["t1"][1] != "balancer-part":
            fail.append("the stranger's item now places %r; this mod rewrote a "
                        "prototype it does not own" % items["t1"][1])
        # THE STRANGER'S DAMAGED PART IS STILL DAMAGED AND STILL UNCOMMON AND
        # STILL THEIRS. `check_fidelity`'s equality is the same statement it
        # makes in every other leg; what differs is which prototype must be
        # standing on the tile at the end of it.
        check_fidelity(healths, qualities, fail, "balancer-part")
        # The technology exists in phase two (this mod is installed beside the
        # stranger) and must be UNRESEARCHED: a grant here would mean something
        # of the stranger's had been converted.
        check_forces(techf, forceparts, fail, "false")
        check_surfaces(surfs, fail, None, False)
        if not audits:
            fail.append("no audit ran; the guest was never asked to look at this world")
        else:
            got = tuple(int(g) for g in audits[-1].groups())
            print("  audit: clusters=%d parts=%d nets=%d drift=%d unbuilt=%d "
                  "refused=%d" % got)
            if got != (0, 0, 0, 0, 0, 0):
                fail.append("the audit reads %s; this mod should own nothing at "
                            "all in this save, which includes refusing nothing "
                            "-- a refusal here would mean a stranger's balancer "
                            "had been converted and then declined" % (got,))
        # NOTHING WAS REFUSED, because nothing was converted. `check_refusals`
        # would be wrong here (it expects the seven), so the negative is made
        # directly: any refusal at all is a stranger's entity in this mod's hands.
        for m in find_all(both, REFUSED) + find_all(both, SUMMARY):
            fail.append("a cluster was refused for the one-belt-per-part rule "
                        "(%s); this mod owns no cluster in this save"
                        % m.group(0).strip()[-70:])
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

    scanned = None
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
        n, scanned, clusters, forces = int(n), int(surfaces), int(clusters), int(forces)
        print("  adopted %d parts from %s surfaces into %d clusters, %d forces "
              "researched, trigger=%s" % (n, surfaces, clusters, forces, trigger))
        if n != legacy_before:
            fail.append("%d parts were standing and %d were adopted"
                        % (legacy_before, n))
        if forces != EXPECT_FORCES:
            fail.append("%d force(s) were given the balancer technology and the "
                        "force rig puts parts on %d: the grant is per force that "
                        "owned a converted part, and a player left holding "
                        "balancers with no recipe is worse off than before"
                        % (forces, EXPECT_FORCES))
        if clusters != EXPECT_CLUSTERS:
            fail.append("the conversion made %d clusters out of the observer's "
                        "%d: two forces' parts touching are two balancers"
                        % (clusters, EXPECT_CLUSTERS))
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
    check_fidelity(healths, qualities, fail, "bbb-balancer-part")
    check_forces(techf, forceparts, fail, "true")
    check_surfaces(surfs, fail, scanned, True)

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
    check_refusals(both, leg, fail)

    # THE REBUILD THAT FOLLOWS THE CONVERSION, and only the `added` leg has one.
    #
    # It is the only place in the estate where a single-edge cluster is ADOPTED
    # beside refused ones -- agents/single-edge.md records that gap explicitly
    # for the `mig21` suite, whose two fixtures are multi-edge to the last
    # cluster. Here the rebuild finds nine, adopts the two the `sok` band
    # compiled a moment earlier because their standing interfaces match the edge
    # list exactly, and rebuilds the seven that have no network to adopt.
    if leg == "added":
        reb = find_all(run, REBUILT)
        if not reb:
            fail.append("this leg adds the mod to an existing save, so the "
                        "on_configuration_changed after on_init must rebuild "
                        "from world; it did not")
        else:
            surfaces, parts, clusters_r, adopted_n, rebuilt_n = (
                int(g) for g in reb[-1].groups())
            print("  the rebuild after the conversion: %d clusters, %d adopted, "
                  "%d rebuilt" % (clusters_r, adopted_n, rebuilt_n))
            if (clusters_r, adopted_n, rebuilt_n) != (
                    EXPECT_CLUSTERS, EXPECT_NETS, EXPECT_REFUSED):
                fail.append("the rebuild found %d clusters and adopted %d of "
                            "them, rebuilding %d; it should adopt the %d the "
                            "`sok` band just compiled -- their interfaces match "
                            "the edge list exactly -- and rebuild the %d that "
                            "have no network to adopt"
                            % (clusters_r, adopted_n, rebuilt_n, EXPECT_NETS,
                               EXPECT_REFUSED))

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
        report(fail, "the incumbent's balancers were adopted: the ones laid one "
                     "belt per part balance, and the rest are refused")


def report(fail, ok_message):
    if fail:
        print("\nMIGRATION ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\n" + ok_message)


if __name__ == "__main__":
    main()
