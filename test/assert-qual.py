#!/usr/bin/env python3
"""Assert that a part at UNCOMMON quality is a part everywhere the guest asks.

`LuaSurface.find_entity` resolves a bare name as NORMAL QUALITY ONLY, so any
call site that used it worked on every normal-quality save and silently failed
on a quality-rolled part. The mig suite's fidelity rig found the first such
site; this suite drives the other four through `findOnTile` (findpart.go):

  * restyle (skin.go)          -- the qblk block's `[BBB] skin` line must say
                                  every variation was SET, once, with m1's own
                                  known literals for the shape; and there must
                                  be exactly ONE such line, because the unfixed
                                  guest retries the parts it can never find on
                                  every flush that touches the cluster.
  * reapFastReplaced           -- the TRIPWIRE: a scripted colliding belt on a
    (fastreplace.go)              standing uncommon part must NOT unregister it
                                  (the unfixed guest reads it as gone). The
                                  TRUE replace on qcol must still unregister,
                                  exactly once.
  * forceOfCluster (limit.go)  -- the over-limit refusal on the uncommon qlim
                                  column must still be DELIVERED: the `told
                                  force` line is the only arm a headless run
                                  can reach, and the unfixed guest's lookup
                                  fails and tells nobody.
  * revertOne (limit.go)       -- needs a player, which no headless run has;
                                  it shares the lookup and the fix with the
                                  three above, and what is asserted here is
                                  the standing negative: ZERO hand-backs.

Plus the question nothing anywhere had asked: an uncommon 2->2 must BALANCE,
against a bare express belt in the same save.

EVERY RIG IS BUILT TO FACTORIO 2.1'S ONE-BELT-PER-PART RULE, and three of the
four always were: `qblk`'s west column carries its inputs and its east column
its outputs, `qcol`'s two INTERIOR parts carry nothing (which is what makes the
fast replace legal), and `qlone` has no belts at all. `qlim` moved -- it was
thirty-two parts with a belt on both sides of each -- and is sixty-six now, one
output part, a 2x32 input block and an EDGELESS part for the sixty-fifth belt
to land on, without which the gesture would ask the OTHER bound and would stop
being a test of forceOfCluster.

    python3 test/assert-qual.py create.log run.log
"""

import re
import sys

QUALITY = re.compile(r"\[BBB-QUAL\] quality rig=(\S+) value=(\S+)")
# `vars` is matched with \S* rather than [\d,]* so that the guest's own
# TRUNCATION is part of what is asserted. logSkin caps the list at 32
# variations and writes a literal "..." after it (guest/go/skin.go), which no
# cluster in any other suite is big enough to reach -- qlim's sixty-six parts
# are, and a cap that moved would otherwise pass silently.
SKIN = re.compile(r"\[BBB\] skin cluster=\d+ parts=(\d+) set=(\d+) vars=(\S*)")
COLLIDE = re.compile(r"\[BBB-QUAL\] collide created=(\S+) part-standing=(\d+)")
FREPCAN = re.compile(r"\[BBB-QUAL\] frep-can value=(\S+)")
FREP = re.compile(r"\[BBB-QUAL\] frep created=(\S+) parts-left-on-tile=(\d+)")
REAP = re.compile(r"\[BBB\] a belt-connectable fast-replaced the part at (-?\d+),(-?\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
    r"(?: refused=(\d+))?"
)
AUDITED = re.compile(r"\[BBB-QUAL\] audited tag=(\S+)")
SEDGE_REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt")
OVERLIMIT = re.compile(
    r"\[BBB\] alert: cluster (\d+) would need (\d+) ports for (\d+) inputs and "
    r"(\d+) outputs, over the limit of (\d+)"
)
TOLDFORCE = re.compile(
    r"\[BBB\] told force (\d+) that cluster (\d+) is over the port limit(.*)$"
)
# EITHER ARM OF THE HAND-BACK, matched on the shape both of them share rather
# than on one sentence. This is a NEGATIVE assertion -- a headless --create has
# no players, so `revertOne` returns before it mines anything -- and an exact
# regex over a negative is the one shape a rename in the guest can make
# VACUOUS: the line stops matching, the assertion stops being able to fail, and
# nothing says so. "piece at x,y" is what `handed the refused piece at 4,7 (over
# the port limit) back to player 1` and its could-not-be-handed-back twin have
# in common, and nothing else in this guest's vocabulary produces it.
HANDEDBACK = re.compile(r"\[BBB\].*\bpiece at -?\d+,-?\d+")
SAMPLE = re.compile(r"\[BBB-QUAL\] sample tick=(\d+) (.*)")

# The rig geometry the probes aim at (must match bbb-qual-test/control.lua).
QCOL_REPLACED = (0, 27)   # QCOL + 1: the interior part the belt replaces
QLONE_TILE = (0, 40)      # the lone part the colliding belt lands on

# WHAT THE RIGS BUILD, written down here rather than read off the guest's own
# report. Four clusters: qblk's 2x2, qcol's 4-column, qlone's single part, and
# qlim's sixty-six -- an output part, a 2x32 input block and the edgeless part
# the sixty-fifth belt lands on. Under Factorio 2.1's one-belt-per-part rule
# qlim is the only one of the four that moved (it was thirty-two parts with a
# belt on both sides of each), and it moved because the rule forbids the shape
# it had. A rig that quietly lost a row still delivers a plausible number on
# the outputs it still has; this is the line that sees that.
WANT_CLUSTERS = 4
WANT_PARTS = 75

# The skin lines the whole run may produce, as an exact multiset. The
# variation numbers are m1's known literals (assert-log.py, SKIN_EXPECT) for
# the same shapes; recomputing them here would be a second copy of the thing
# under test.
#
#   at the first flush: qblk (2x2 block), qcol (4-column), qlone (lone part),
#   qlim (32-column, whose 30 interior parts are all variation 11);
#   after the fast replace splits qcol: its top part alone again (set=1,
#   variation back to 1), and its lower half a 2-column whose top part moved
#   to 3 while the bottom kept its 9 -- so parts=2, set=1.
#
# The unfixed guest fails this two ways at once: `set=0` on every line (it can
# never find an uncommon part to touch), and MORE lines than these, because a
# part it never touched is retried by every flush that queues its cluster --
# which is what the qblk poke at t=500/510 exists to provoke.
#
# qlim's string is what a one-part-wide stalk on each end of a two-wide block
# looks like: the output part alone at the top (5), the block's first row (22,
# 27) and then its body (25, 43) -- and the guest stops there, because the list
# is capped at 32. The full sequence continues "25,43" to the block's last row
# (18, 35) and ends with the edgeless part alone at the bottom (2). Derived
# from `guest/go/skin`'s own pure-Go `Variation` over the tile set the rig
# builds -- the same package `make check` proves and the same one m1's literals
# come from -- rather than read back off a run of the thing under test.
SKIN_EXPECT = sorted([
    ("4", "4", "21,27,17,35"),                     # qblk
    ("4", "4", "5,6,6,2"),                         # qcol, at the first flush
    ("1", "1", "1"),                               # qlone
    ("66", "66", "5,22,27," + "25,43," * 14 + "25,..."),    # qlim, truncated
    ("1", "1", "1"),                               # qcol's top part, post-split
    ("2", "1", "5,2"),                             # qcol's lower half, post-
                                                   # split: its own top moved
                                                   # 6 -> 5, its bottom kept 2
])

# The audits, tag -> (clusters, parts, nets, drift, unbuilt, refused). nets is
# 3 at t0 because qlone has no belts at all -- a cluster with no edges is a
# legitimate half-built state and gets no network. The replace splits qcol into
# two one-sided clusters (one input, no outputs and the reverse), so its
# network comes down and neither half gets one: clusters 4 -> 5 and nets 3 -> 2.
# The sixty-fifth belt moves drift to 1 and refused to 1 and nothing else: a
# refused cluster still HAS its network and knows its edge list has moved past
# what the mod can build, and it stays that way for the rest of the run.
#
# THE TUPLE IS THE ASSERTION AND `unbuilt=0` ALONE WOULD NOT BE. A cluster with
# no inputs or no outputs is a legitimate half-built state and never counts as
# unbuilt, so a rig that lost half its belts reads `unbuilt=0` while delivering
# nothing -- which is why the cluster and part counts are pinned against
# WANT_CLUSTERS/WANT_PARTS as well, and why `nets` is written down per tag
# rather than compared against `clusters` (three of this save's five clusters
# are legitimately network-free by the end).
AUDIT_EXPECT = {
    # t0's audit runs inside on_init's marker dispatch, and the report is
    # taken BEFORE the drain it forces compiles anything -- so it sees the
    # three compilable clusters as unbuilt and no networks at all. qlone never
    # counts as unbuilt: a cluster with no edges is a legitimate half-built
    # state at any size.
    "t0":           (4, 75, 0, 0, 3, 0),
    "post-collide": (4, 75, 3, 0, 0, 0),
    "post-replace": (5, 74, 2, 0, 0, 0),
    "post-lim":     (5, 74, 2, 1, 0, 1),
    "final":        (5, 74, 2, 1, 0, 1),
}

SPREAD = 0.01
RATE_TOL = 0.02


def parse_sample(text):
    out = {}
    for chunk in text.split():
        name, _, csv = chunk.partition("=")
        out[name] = [int(v) for v in csv.split(",")]
    return out


def main():
    lines = []
    for path in sys.argv[1:3]:
        with open(path, errors="replace") as f:
            lines.extend(f)

    fail = []

    # ---- anti-vacuity: the parts really are uncommon -----------------------
    quality = {m.group(1): m.group(2)
               for m in (QUALITY.search(l) for l in lines) if m}
    for rig in ("qblk", "qcol", "qlone", "qlim"):
        got = quality.get(rig)
        print("  %-5s quality=%s" % (rig, got))
        if got != "uncommon":
            fail.append("rig %s reports quality=%s; every part in this suite "
                        "must be uncommon or the whole suite is vacuous"
                        % (rig, got))

    # ---- restyle: the skin lines, as an exact multiset ---------------------
    skins = sorted(m.groups() for m in (SKIN.search(l) for l in lines) if m)
    for parts, set_, vars_ in skins:
        print("  skin parts=%s set=%s vars=%s"
              % (parts, set_, vars_ if len(vars_) < 40 else vars_[:37] + "..."))
    if skins != SKIN_EXPECT:
        fail.append("the run's skin lines are not the expected six: got %d "
                    "line(s) %r, expected %r. set=0 means restyle could not "
                    "FIND the uncommon parts; extra lines mean it is retrying "
                    "them on every flush" %
                    (len(skins),
                     [(p, s, v[:20]) for p, s, v in skins],
                     [(p, s, v[:20]) for p, s, v in SKIN_EXPECT]))

    # ---- the collide probe: a standing part must stay registered -----------
    collide = [m for m in (COLLIDE.search(l) for l in lines) if m]
    if len(collide) != 1 or collide[0].group(1) != "true":
        fail.append("the colliding belt was not created (%r); the probe never "
                    "asked its question" % [m.group(0) for m in collide])
    elif collide[0].group(2) != "1":
        fail.append("the lone part is not standing after the colliding belt "
                    "(part-standing=%s); the probe world is not the one the "
                    "assertion is about" % collide[0].group(2))
    else:
        print("  collide: belt created, part still standing")

    reaps = [(int(m.group(1)), int(m.group(2)))
             for m in (REAP.search(l) for l in lines) if m]
    if QLONE_TILE in reaps:
        fail.append("the guest unregistered the lone part at %r under a "
                    "COLLIDING belt: it asked for the part at normal quality, "
                    "got nil, and edited the registry on it. The part is still "
                    "standing" % (QLONE_TILE,))
    if reaps != [QCOL_REPLACED]:
        fail.append("fast-replace unregistrations over the whole run are %r, "
                    "expected exactly one at %r (the true replace on qcol)"
                    % (reaps, QCOL_REPLACED))
    else:
        print("  reap: exactly one unregistration, at the replaced part")

    # ---- the true replace: quality does not gate the ENGINE either ---------
    can = [m for m in (FREPCAN.search(l) for l in lines) if m]
    if not can or can[-1].group(1) != "true":
        fail.append("can_fast_replace over the uncommon interior part is %r; "
                    "the engine gates fast replace somewhere this suite "
                    "believed it does not"
                    % [m.group(1) for m in can])
    frep = [m for m in (FREP.search(l) for l in lines) if m]
    if not frep or frep[-1].group(1) != "true" or frep[-1].group(2) != "0":
        fail.append("the true fast replace did not happen as staged: %r"
                    % [m.group(0) for m in frep])
    else:
        print("  frep: engine says true, part really gone, belt created")

    # ---- the audits, paired with the tag logged just after each ------------
    audits = {}
    last = None
    for l in lines:
        m = AUDIT.search(l)
        if m:
            last = tuple(0 if g is None else int(g) for g in m.groups())
            continue
        m = AUDITED.search(l)
        if m and last is not None:
            audits[m.group(1)] = last
            last = None
    for tag, want in AUDIT_EXPECT.items():
        got = audits.get(tag)
        print("  audit %-13s %s" % (tag, (got,)))
        if got is None:
            fail.append("no audit for tag=%s" % tag)
        elif got != want:
            fail.append("audit %s is clusters=%d parts=%d nets=%d drift=%d "
                        "unbuilt=%d refused=%d, expected %r"
                        % ((tag,) + got + (want,)))

    # THE GEOMETRY THE SAVE WAS SUPPOSED TO BUILD, asserted where the world is
    # still as on_init left it. The tuples above pin this too, but only as one
    # of six numbers each; saying it once by name is what makes a rig that came
    # out the wrong shape report as a rig rather than as an audit.
    t0 = audits.get("t0")
    if t0 and (t0[0], t0[1]) != (WANT_CLUSTERS, WANT_PARTS):
        fail.append("the save holds %d clusters over %d parts and the rigs "
                    "build %d over %d -- a rig is not the shape this suite "
                    "thinks it is" % (t0[0], t0[1], WANT_CLUSTERS, WANT_PARTS))

    # NOTHING HERE MAY BE REFUSED FOR THE ONE-BELT-PER-PART RULE. Every rig in
    # this save is built to it -- qlim's sixty-fifth belt lands on an EDGELESS
    # part precisely so that it asks the PORT bound and not this one -- so a
    # single-edge refusal is a statement about the save rather than about the
    # guest, and it would take the `told force` assertion below with it by
    # answering it for the wrong reason.
    sedge = [m for m in (SEDGE_REFUSED.search(l) for l in lines) if m]
    if sedge:
        fail.append("%d single-edge refusal(s) were issued in a save whose "
                    "rigs are all built to the rule; the first was cluster %s "
                    "with %s part(s) carrying more than one belt"
                    % (len(sedge), sedge[0].group(1), sedge[0].group(2)))

    # ---- the refusal, and that somebody was TOLD ---------------------------
    over = [m for m in (OVERLIMIT.search(l) for l in lines) if m]
    if len(over) != 1:
        fail.append("the guest refused %d times, expected exactly one -- the "
                    "sixty-fifth belt on qlim: %r"
                    % (len(over), [m.group(0)[:90] for m in over]))
    elif (over[0].group(2), over[0].group(3), over[0].group(4),
          over[0].group(5)) != ("128", "65", "1", "64"):
        fail.append("the refusal is not qlim's: %s" % over[0].group(0).strip())
    else:
        print("  refusal: %s..." % over[0].group(0).strip()[:70])

    told = [m for m in (TOLDFORCE.search(l) for l in lines) if m]
    if len(told) != 1 or "FAILED" in told[0].group(3):
        fail.append("the force was told about the refusal %d time(s)%s, "
                    "expected exactly one clean report. On an UNCOMMON column "
                    "the unfixed guest cannot find a part to read the force "
                    "off, and the refusal is delivered to nobody"
                    % (len(told),
                       " (print FAILED)" if told and "FAILED" in told[0].group(3)
                       else ""))
    elif over and told[0].group(2) != over[0].group(1):
        fail.append("the force was told about cluster %s and the refusal was "
                    "cluster %s" % (told[0].group(2), over[0].group(1)))
    else:
        print("  told: %s" % told[0].group(0).strip()[:70])

    if any(HANDEDBACK.search(l) for l in lines):
        fail.append("an over-limit piece was handed back in a run with no "
                    "players; revertOne fired for a script build")

    # ---- an uncommon balancer balances -------------------------------------
    samples = {int(m.group(1)): parse_sample(m.group(2))
               for m in (SAMPLE.search(l) for l in lines) if m}
    if 900 not in samples or 2100 not in samples:
        fail.append("the throughput window was not sampled at both ends")
    else:
        a, b = samples[900], samples[2100]
        delta = {k: [y - x for x, y in zip(a[k], b[k])] for k in b if k in a}
        ctrl = delta["ctrl"][0]
        print("  one saturated express belt delivered %d items over the window"
              % ctrl)
        if ctrl < 500:
            fail.append("the control belt delivered %d items; nothing in this "
                        "save was flowing" % ctrl)
        per = delta.get("qblk", [])
        total = sum(per)
        ratio = total / ctrl if ctrl else 0.0
        spread = (max(per) - min(per)) / (sum(per) / len(per)) if total else 1.0
        print("  qblk %s  total %.3fx one belt, spread %.2f%%"
              % (" ".join(str(v) for v in per), ratio, spread * 100))
        if abs(ratio - 2.0) > RATE_TOL * 2.0:
            fail.append("the uncommon 2->2 delivered %.3fx one belt and should "
                        "deliver 2.000x" % ratio)
        if spread > SPREAD:
            fail.append("qblk outputs spread %.2f%%, over the %.0f%% bound"
                        % (spread * 100, SPREAD * 100))
        lim = sum(delta.get("qlim", [0]))
        print("  qlim delivered %d items across the refusal" % lim)
        if lim < 100:
            fail.append("qlim delivered %d items over the window; the refused "
                        "edit stopped a standing network (or nothing was ever "
                        "flowing, which fails the same way on purpose)" % lim)

    if fail:
        print("\nQUALITY ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\nan uncommon part is a part everywhere the guest asks")


if __name__ == "__main__":
    main()
