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

    python3 test/assert-qual.py create.log run.log
"""

import re
import sys

QUALITY = re.compile(r"\[BBB-QUAL\] quality rig=(\S+) value=(\S+)")
SKIN = re.compile(r"\[BBB\] skin cluster=\d+ parts=(\d+) set=(\d+) vars=([\d,]*)")
COLLIDE = re.compile(r"\[BBB-QUAL\] collide created=(\S+) part-standing=(\d+)")
FREPCAN = re.compile(r"\[BBB-QUAL\] frep-can value=(\S+)")
FREP = re.compile(r"\[BBB-QUAL\] frep created=(\S+) parts-left-on-tile=(\d+)")
REAP = re.compile(r"\[BBB\] a belt-connectable fast-replaced the part at (-?\d+),(-?\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
)
AUDITED = re.compile(r"\[BBB-QUAL\] audited tag=(\S+)")
OVERLIMIT = re.compile(
    r"\[BBB\] alert: cluster (\d+) would need (\d+) ports for (\d+) inputs and "
    r"(\d+) outputs, over the limit of (\d+)"
)
TOLDFORCE = re.compile(
    r"\[BBB\] told force (\d+) that cluster (\d+) is over the port limit(.*)$"
)
HANDEDBACK = re.compile(r"\[BBB\] handed the over-limit piece at ")
SAMPLE = re.compile(r"\[BBB-QUAL\] sample tick=(\d+) (.*)")

# The rig geometry the probes aim at (must match bbb-qual-test/control.lua).
QCOL_REPLACED = (0, 27)   # QCOL + 1: the interior part the belt replaces
QLONE_TILE = (0, 40)      # the lone part the colliding belt lands on

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
SKIN_EXPECT = sorted([
    ("4", "4", "21,27,17,35"),                     # qblk
    ("4", "4", "5,6,6,2"),                         # qcol, at the first flush
    ("1", "1", "1"),                               # qlone
    ("32", "32", "5," + "6," * 30 + "2"),          # qlim
    ("1", "1", "1"),                               # qcol's top part, post-split
    ("2", "1", "5,2"),                             # qcol's lower half, post-
                                                   # split: its own top moved
                                                   # 6 -> 5, its bottom kept 2
])

# The audits, tag -> (clusters, parts, nets, drift, unbuilt). nets is 3 at t0
# because qlone has no belts at all -- a cluster with no edges is a legitimate
# half-built state and gets no network. The replace splits qcol into two
# one-sided clusters (one input, no outputs and the reverse), so its network
# comes down and neither half gets one: clusters 4 -> 5 and nets 3 -> 2. The
# sixty-fifth belt moves drift to 1 and nothing else: a refused cluster still
# HAS its network and knows its edge list has moved past what the mod can
# build, and it stays that way for the rest of the run.
AUDIT_EXPECT = {
    # t0's audit runs inside on_init's marker dispatch, and the report is
    # taken BEFORE the drain it forces compiles anything -- so it sees the
    # three compilable clusters as unbuilt and no networks at all. qlone never
    # counts as unbuilt: a cluster with no edges is a legitimate half-built
    # state at any size.
    "t0":           (4, 41, 0, 0, 3),
    "post-collide": (4, 41, 3, 0, 0),
    "post-replace": (5, 40, 2, 0, 0),
    "post-lim":     (5, 40, 2, 1, 0),
    "final":        (5, 40, 2, 1, 0),
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
            last = tuple(int(g) for g in m.groups())
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
                        "unbuilt=%d, expected %r" % ((tag,) + got + (want,)))

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
