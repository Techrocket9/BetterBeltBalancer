#!/usr/bin/env python3
"""The SPACE AGE suite: a balancer on a platform surface, and BELT STACKING.

Two legs, both needing the DLC and nothing else in common.

1. A cluster's parts can live on a platform surface while its network lives on
   the one global hidden surface -- so the linked belts joining them cross from a
   moving platform to a surface that is not going anywhere. Spike S1 established
   that linked belts are cross-surface; this establishes that they are cross-
   surface FROM A PLATFORM, at full rate, with exact balance. `ctrl` is an
   uninterrupted belt on the same platform, so the comparison is against the
   engine on the same surface rather than against a number.

2. A recompile hands a stacked network back STACKED. Belt stacking is a Space
   Age feature at the PROTOTYPE level -- a loader's `max_belt_stack_size` is
   refused at load without the `space_travel` feature flag -- so no base-only
   suite can build a stacked belt at all, which is why this leg is here.

   Conservation was never the defect and is not the assertion: the guest drained
   totals per (name, quality) and put every item back. What was wrong is that 128
   items came back as 128 belt positions of one instead of 32 positions of four,
   which is a quarter of the density and a real throughput dip on a saturated
   stacked belt. So the assertion is on the stack PROFILE of the hidden surface,
   sampled either side of a forced recompile inside ONE tick:

     * the items that crossed into the hidden set arrived as STACKS -- the
       single-item position count does not move at all, and the growth in
       stacked items accounts for the whole delta, exactly;
     * the `plain` band, on the same stacking force but fed by an ordinary
       loader, does the opposite exactly: its singles grow by the whole delta and
       no stack is invented;
     * `formed` first, so that a run where stacking silently did not happen fails
       as vacuous rather than passing.

3. STACKED SUSHI -- the `smix` band, and the only rig in this repo that reaches
   `kindAt`'s multi-candidate branches at all.

   `detailedTally` reads a stacked line position by position and `kindAt` says
   which (name, quality) total each position belongs to. Its three branches are
   `len(totals) == 1` (no host call), a `name_is` loop over several candidates,
   and a `quality` tiebreak between two entries of the SAME name. Only the first
   had ever run: every stacked rig above is single-kind iron plate, and every
   multi-kind rig lives in the base-only `mix` suite, where the stacking gate is
   shut and `detailedTally` is never called. Multi-kind AND stacked is Space
   Age, so the band is here.

   Three things are asserted and the first is the one without which the other
   two prove nothing:

     * ANTI-VACUITY, sampled from the world rather than assumed: at the instant
       the teardown reads them, hidden transport lines really do carry two
       distinct NAMES at once with a stacked position among them, and some line
       really does carry one name at two QUALITIES -- which is the only shape a
       `name_is` cannot settle;
     * conservation EXACT PER (NAME, QUALITY) across the recompile. A kindAt that
       misattributed a position would hand back the right number of items under
       the wrong key, which a single total cannot see and this can;
     * the same stack profile the bands above assert: no single-item position is
       invented, and every item that crossed is accounted for in the stacked
       total. A kindAt that gave up returns false, `detailedTally` falls back to
       the flat totals, and that line's items come back UNSTACKED -- so a
       fallback anywhere shows up here as singles.

EVERY BAND IS BUILT TO FACTORIO 2.1'S ONE-BELT-PER-PART RULE: two columns of
parts per row, plus one EDGELESS part on the end for each band's recompile belt
to land on, because under that rule a working balancer has no free face and the
belt would otherwise be REFUSED rather than compiled. The stack profiles, the
per-kind conservation and the rates are all properties of the BELTS, which did
not move, so every contract below is the one it was.

    python3 test/assert-plat.py create.log run.log
"""

import re
import sys

SAMPLE = re.compile(r"\[BBB-PLAT\] t=(\d+) ctrl=(\d+) out=\[(\d+) (\d+)\]")
SURFACE = re.compile(r"\[BBB-PLAT\] platform state=\S+ surface=(\S+)")

BONUS = re.compile(r"\[BBB-STK\] force \S+ belt_stack_size_bonus=(\d+)")
PROFILE = re.compile(
    r"\[BBB-STK\] (formed|full before|full after|plain before|plain after|"
    r"flow before|flow after) total=(\d+) visible=(\d+) hidden=(\d+) "
    r"hitems=(\d+) hpos=(\d+) hist=(\S*)")
FLOW = re.compile(r"\[BBB-STK\] t=(\d+) ctrl=(\d+) flow=\[([\d ]+)\]")
SPILL = re.compile(r"\[BBB\] spilled (\d+) items")
TOOKBACK = re.compile(r"\[BBB\] cluster \d+ took back (\d+) items")

ITEMS = re.compile(r"\[BBB-STK\] smix item list ok: names=(\d+) kinds=(\d+) rotate=(\d+)")
SMIX = re.compile(
    r"\[BBB-STK\] smix tag=(before|after) total=(\d+) visible=(\d+) hidden=(\d+) "
    r"kinds=(\d+) hitems=(\d+) hpos=(\d+) hist=(\S*)")
SMIXLINES = re.compile(
    r"\[BBB-STK\] smixlines tag=(before|after) lines=(\d+) multi=(\d+) "
    r"multistacked=(\d+) qtie=(\d+) qtiestacked=(\d+) maxnames=(\d+) "
    r"maxkinds=(\d+) maxstack=(\d+)")
SMIXKIND = re.compile(r"\[BBB-STK\] smixkind tag=(before|after) name=(\S+) count=(\d+)")
OVERFLOW = re.compile(r"\[BBB\] alert: cluster \d+ carried more than \d+ item kinds")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
    r"(?: refused=(\d+))?")
SEDGE_REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt")

# WHAT THE RIGS BUILD, written down here rather than read off the guest's own
# report. Five clusters over two surfaces: the platform's 2->2 (four parts) and
# the four stacking bands, each two columns of parts plus one EDGELESS part for
# its recompile's belt to land on -- full 4 rows -> 9 parts, flow 4 -> 9,
# plain 2 -> 5, smix 2 -> 5.
#
# It is a statement about the SAVE and not about the compiler, which is the
# whole reason it is a constant: a band that quietly lost a row, or that was
# rebuilt one column wide in the old multi-edge idiom, moves this number, and
# neither the stack profile nor a rate could say so -- both are about what the
# drain did with a network, whatever size it turned out to be.
WANT_CLUSTERS = 5
WANT_PARTS = 32

# The band is sized to stay INSIDE the carry pool's 32-group bound -- overflowing
# it is the base-only `mix` suite's business ("More than thirty-two kinds") and
# here it would spill, which is the thing this band asserts does not happen.
SMIX_KINDS = 9

T0, T1 = 600, 1500
SPREAD = 0.01

# The stacking leg's own window: `flow` is recompiled at t=800 and measured from
# 200 ticks later, because a rebuild puts every drained item back at the HEAD of
# the butterfly and the outputs are briefly starved by construction.
S0, S1 = 1000, 1500
STK_SPREAD = 0.015
STK_TOTAL = 3.90


def parse_hist(s):
    """'1:72,4:264' -> {1: 72, 4: 264}."""
    out = {}
    for part in s.split(","):
        if not part:
            continue
        k, _, v = part.partition(":")
        out[int(k)] = int(v)
    return out


def singles(h):
    return h.get(1, 0)


def stacked_items(h):
    return sum(k * n for k, n in h.items() if k > 1)


def check_stacked_sushi(lines, fail):
    """The `smix` band: stacked AND multi-kind, which nothing else here is."""
    itemlist = None
    prof, mix, kinds = {}, {}, {"before": {}, "after": {}}
    overflow = 0
    for raw in lines:
        m = ITEMS.search(raw)
        if m:
            itemlist = tuple(int(v) for v in m.groups())
        m = SMIX.search(raw)
        if m:
            prof[m.group(1)] = {
                "total": int(m.group(2)), "visible": int(m.group(3)),
                "hidden": int(m.group(4)), "kinds": int(m.group(5)),
                "hitems": int(m.group(6)), "hpos": int(m.group(7)),
                "hist": parse_hist(m.group(8)),
            }
        m = SMIXLINES.search(raw)
        if m:
            mix[m.group(1)] = dict(zip(
                ("lines", "multi", "multistacked", "qtie", "qtiestacked",
                 "maxnames", "maxkinds", "maxstack"),
                (int(v) for v in m.groups()[1:])))
        m = SMIXKIND.search(raw)
        if m:
            kinds[m.group(1)][m.group(2)] = int(m.group(3))
        if OVERFLOW.search(raw):
            overflow += 1

    print("\nstacked sushi -- the multi-kind stacked drain (`smix`):")
    if itemlist is None or "before" not in prof or "after" not in prof:
        fail.append("the stacked-sushi band did not run at all")
        return
    names, want_kinds, rotate = itemlist
    print("  %d item names over %d (name, quality) kinds, rotating every %d ticks"
          % (names, want_kinds, rotate))
    if want_kinds != SMIX_KINDS:
        fail.append("the band's source list carries %d kinds, expected %d"
                    % (want_kinds, SMIX_KINDS))

    # ANTI-VACUITY FIRST. Everything below is about what the guest did with lines
    # of a particular shape, so a run whose lines were not that shape proves
    # nothing at all -- and the shape is a function of the rotation period
    # against the length of a hidden transport line, which is exactly the sort
    # of thing that stops being true when a constant moves.
    b = mix["before"]
    print("  at the moment of the teardown, of %d hidden lines carrying this "
          "band: %d carried two or more NAMES (%d of those stacked), %d carried "
          "one name at two QUALITIES (%d stacked); richest line %d kinds, "
          "biggest stack %d"
          % (b["lines"], b["multi"], b["multistacked"], b["qtie"],
             b["qtiestacked"], b["maxkinds"], b["maxstack"]))
    if b["multistacked"] < 3:
        fail.append("only %d hidden line(s) carried two names AND a stacked "
                    "position; kindAt's name_is loop was not reached and the "
                    "band proves nothing -- shorten the rotation period"
                    % b["multistacked"])
    if b["qtiestacked"] < 1:
        fail.append("no hidden line carried one name at two qualities with a "
                    "stacked position; kindAt's quality tiebreak was not "
                    "reached")
    if b["maxstack"] < 2:
        fail.append("nothing on this band is stacked (biggest position %d); "
                    "detailedTally never ran" % b["maxstack"])
    if prof["before"]["kinds"] < SMIX_KINDS:
        fail.append("only %d of %d kinds were in flight when the network was "
                    "torn down" % (prof["before"]["kinds"], SMIX_KINDS))

    # Conservation, PER (name, quality).
    lost = []
    for k in sorted(set(kinds["before"]) | set(kinds["after"])):
        before, after = kinds["before"].get(k, 0), kinds["after"].get(k, 0)
        if before != after:
            lost.append("%s %d -> %d" % (k, before, after))
    print("  conservation across the recompile, per (name, quality): %s"
          % ("EXACT over %d kinds, %d items"
             % (len(kinds["before"]), sum(kinds["before"].values()))
             if not lost else "BROKEN"))
    for k in sorted(kinds["before"]):
        print("    %-34s %6d" % (k, kinds["before"][k]))
    if lost:
        fail.append("the stacked-sushi recompile did not conserve every kind: "
                    + "; ".join(lost))
    if not kinds["before"]:
        fail.append("the stacked-sushi band held nothing when it was recompiled")

    # ...and the stack profile, exactly as the `full` and `flow` bands assert it.
    pb, pa = prof["before"], prof["after"]
    moved = pa["hidden"] - pb["hidden"]
    dsing = singles(pa["hist"]) - singles(pb["hist"])
    dstk = stacked_items(pa["hist"]) - stacked_items(pb["hist"])
    print("  %d items crossed into the network: %+d single, %+d stacked "
          "(hist %s -> %s)"
          % (moved, dsing, dstk,
             " ".join("%dx%d" % (n, k) for k, n in sorted(pb["hist"].items())),
             " ".join("%dx%d" % (n, k) for k, n in sorted(pa["hist"].items()))))
    if pb["total"] != pa["total"]:
        fail.append("smix: %d items before the recompile, %d after"
                    % (pb["total"], pa["total"]))
    if moved <= 0:
        fail.append("smix: nothing crossed into the rebuilt network (%d)" % moved)
    if dsing != 0:
        fail.append("smix: the recompile put %d more SINGLE items on the hidden "
                    "surface; a multi-kind stacked line came back unstacked, "
                    "which is what a kindAt fallback looks like" % dsing)
    if dstk != moved:
        fail.append("smix: %d items crossed into the network but only %d of "
                    "them are in stacks" % (moved, dstk))
    if overflow:
        fail.append("the stacked-sushi band overflowed the carry pool's kind "
                    "bound %d time(s); it is sized to stay inside it" % overflow)


def check_stacking(lines, fail):
    bonus = None
    prof = {}
    flow = {}
    spills = []
    tookback = []
    for raw in lines:
        m = BONUS.search(raw)
        if m:
            bonus = int(m.group(1))
        m = PROFILE.search(raw)
        if m:
            prof[m.group(1)] = {
                "total": int(m.group(2)), "visible": int(m.group(3)),
                "hidden": int(m.group(4)), "hitems": int(m.group(5)),
                "hpos": int(m.group(6)), "hist": parse_hist(m.group(7)),
            }
        m = FLOW.search(raw)
        if m:
            flow[int(m.group(1))] = (int(m.group(2)),
                                     [int(v) for v in m.group(3).split()])
        m = SPILL.search(raw)
        if m:
            spills.append(int(m.group(1)))
        m = TOOKBACK.search(raw)
        if m:
            tookback.append(int(m.group(1)))

    print("\nbelt stacking (Space Age):")
    if bonus is None:
        fail.append("the stacking leg did not run at all")
        return
    if bonus < 1:
        fail.append("the stacking force's belt_stack_size_bonus is %d" % bonus)
    for want in ("formed", "full before", "full after", "plain before",
                 "plain after", "flow before", "flow after"):
        if want not in prof:
            fail.append("the stacking leg never sampled '%s'" % want)
            return

    # Not vacuous: stacks really formed before anything was recompiled.
    f = prof["formed"]
    fstacked = sum(n for k, n in f["hist"].items() if k > 1)
    print("  before any recompile the hidden networks held %d items over %d belt "
          "positions, %s" % (f["hitems"], f["hpos"],
                             " ".join("%d of size %d" % (n, k)
                                      for k, n in sorted(f["hist"].items()))))
    if fstacked < 50:
        fail.append("only %d stacked belt positions formed; belt stacking did "
                    "not happen and the leg proves nothing" % fstacked)
    if stacked_items(f["hist"]) < f["hitems"] / 2:
        fail.append("under half the items on the hidden surface are stacked "
                    "(%d of %d)" % (stacked_items(f["hist"]), f["hitems"]))

    print("  %-6s %8s %8s %8s %8s   %s"
          % ("band", "total", "hidden", "singles", "stacked", "verdict"))
    for band, stacking in (("full", True), ("plain", False), ("flow", True)):
        b, a = prof[band + " before"], prof[band + " after"]
        moved = a["hidden"] - b["hidden"]
        dsing = singles(a["hist"]) - singles(b["hist"])
        dstk = stacked_items(a["hist"]) - stacked_items(b["hist"])
        if b["total"] != a["total"]:
            fail.append("%s: %d items before the recompile, %d after"
                        % (band, b["total"], a["total"]))
        if stacking:
            verdict = "all %d reinserted items arrived stacked" % moved
            if dsing != 0:
                fail.append("%s: the recompile put %d more SINGLE items on the "
                            "hidden surface; a stacked network came back "
                            "unstacked" % (band, dsing))
            if dstk != moved:
                fail.append("%s: %d items crossed into the network but only %d "
                            "of them are in stacks" % (band, moved, dstk))
        else:
            verdict = "unstacked, and no stack was invented"
            if dstk != 0:
                fail.append("%s: the recompile invented %d stacked items on a "
                            "band that has none" % (band, dstk))
            if dsing != moved:
                fail.append("%s: %d items crossed into the network but the "
                            "single-item positions grew by %d"
                            % (band, moved, dsing))
        print("  %-6s %8d %8d %+8d %+8d   %s"
              % (band, a["total"], a["hidden"], dsing, dstk, verdict))

    if spills:
        fail.append("a recompile spilled %d items onto the ground; only a "
                    "removal may spill" % sum(spills))
    print("  %d recompiles handed items back (median %d), %d items spilled"
          % (len(tookback),
             sorted(tookback)[len(tookback) // 2] if tookback else 0,
             sum(spills)))
    # Four bands are recompiled -- full, plain, flow and smix -- and every one of
    # them is meant to be holding something when it is.
    if len(tookback) < 4:
        fail.append("only %d recompile(s) handed anything back; the rigs were "
                    "not full" % len(tookback))

    if S0 not in flow or S1 not in flow:
        fail.append("the stacked flow rig was not sampled at t=%d and t=%d"
                    % (S0, S1))
        return
    ctrl = flow[S1][0] - flow[S0][0]
    outs = [flow[S1][1][i] - flow[S0][1][i] for i in range(len(flow[S1][1]))]
    mean = sum(outs) / float(len(outs))
    spread = (max(outs) - min(outs)) / mean if mean else 0.0
    total = sum(outs) / float(ctrl) if ctrl else 0.0
    print("  the 4x4 recompiled UNDER STACKED LOAD, over the 500 ticks after "
          "it: %s" % " ".join(str(v) for v in outs))
    print("    against one saturated stacked belt's %d: %.3fx, spread %.2f%%"
          % (ctrl, total, 100 * spread))
    if ctrl < 1000:
        fail.append("the stacked control belt only moved %d items; stacking is "
                    "not saturating" % ctrl)
    if total < STK_TOTAL:
        fail.append("the recompiled stacked balancer delivered %.3f belts, "
                    "under %.2f" % (total, STK_TOTAL))
    if spread > STK_SPREAD:
        fail.append("stacked outputs spread %.2f%%, over %.2f%%"
                    % (100 * spread, 100 * STK_SPREAD))


def main():
    lines = []
    for path in sys.argv[1:]:
        with open(path, errors="replace") as f:
            lines.extend(f)

    fail = []
    samples = {}
    surface = None
    for raw in lines:
        m = SURFACE.search(raw)
        if m:
            surface = m.group(1)
        m = SAMPLE.search(raw)
        if m:
            samples[int(m.group(1))] = [int(v) for v in m.groups()[1:]]

    if surface in (None, "NIL"):
        print("no space platform surface was created; nothing was verified")
        sys.exit(1)
    if T0 not in samples or T1 not in samples:
        print("the platform rig was not sampled at t=%d and t=%d" % (T0, T1))
        sys.exit(1)

    ctrl = samples[T1][0] - samples[T0][0]
    outs = [samples[T1][i] - samples[T0][i] for i in (1, 2)]
    total = sum(outs) / float(ctrl)
    mean = sum(outs) / float(len(outs))
    spread = (max(outs) - min(outs)) / mean if mean else 0.0

    print("  platform surface: %s" % surface)
    print("  one uninterrupted belt on the same platform delivered %d items "
          "between t=%d and t=%d" % (ctrl, T0, T1))
    print("  2->2 balancer on the platform: %s  total %.3fx  spread %.2f%%"
          % (" ".join(str(v) for v in outs), total, 100 * spread))

    if ctrl < 400:
        fail.append("the control belt only moved %d items" % ctrl)
    if not 1.98 <= total <= 2.02:
        fail.append("the platform balancer delivered %.3f belts, expected 2.000" % total)
    if spread > SPREAD:
        fail.append("outputs spread %.2f%%, over %.2f%%" % (100 * spread, 100 * SPREAD))

    check_stacking(lines, fail)
    check_stacked_sushi(lines, fail)

    # --- the final audit, over both surfaces ---------------------------------
    #
    # THE CLUSTER AND PART COUNTS ARE THE POINT AND `unbuilt=0` WOULD NOT BE. A
    # cluster with no inputs or no outputs is a legitimate half-built state and
    # never counts as unbuilt, so a band that came out one column wide -- the
    # old multi-edge idiom -- would be REFUSED, hold nothing, and still read
    # `unbuilt=0`. The tuple is what sees it, and `nets == clusters` is what
    # says every band's edges were recognised.
    audits = [tuple(0 if g is None else int(g) for g in m.groups())
              for m in (AUDIT.search(l) for l in lines) if m]
    if not audits:
        fail.append("no audit was ever logged, so nothing checked the registry "
                    "against the world")
    else:
        clusters, parts, nets, drift, unbuilt, refused = audits[-1]
        print("\nfinal audit: %d clusters, %d parts, %d networks, drift=%d, "
              "unbuilt=%d, refused=%d"
              % (clusters, parts, nets, drift, unbuilt, refused))
        if (clusters, parts) != (WANT_CLUSTERS, WANT_PARTS):
            fail.append("the save holds %d clusters over %d parts and the rigs "
                        "build %d over %d -- a band is not the shape this suite "
                        "thinks it is"
                        % (clusters, parts, WANT_CLUSTERS, WANT_PARTS))
        if nets != clusters:
            fail.append("%d of %d clusters have no network at all: something "
                        "adjacent to a balancer is not being classified as an "
                        "edge" % (clusters - nets, clusters))
        if drift or unbuilt or refused:
            fail.append("the final audit found drift=%d unbuilt=%d refused=%d "
                        "over %d clusters, on a world nothing has touched since "
                        "tick 900" % (drift, unbuilt, refused, clusters))

    # Every band here is built to the one-belt-per-part rule, so a refusal is a
    # statement about the SAVE and not about the guest. Asserted separately from
    # the audit's `refused=` column because a refusal can be issued, delivered
    # and then withdrawn between two audits -- and because it names the cluster,
    # which the column does not.
    sedge = [m for m in (SEDGE_REFUSED.search(l) for l in lines) if m]
    if sedge:
        fail.append("%d single-edge refusal(s) were issued in a save whose "
                    "bands are all built to the rule; the first was cluster %s "
                    "with %s part(s) carrying more than one belt"
                    % (len(sedge), sedge[0].group(1), sedge[0].group(2)))

    if fail:
        print("\nSPACE AGE ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\nspace age assertions passed (platform surface, belt stacking, "
          "stacked sushi)")


if __name__ == "__main__":
    main()
