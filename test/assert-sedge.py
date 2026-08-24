#!/usr/bin/env python3
"""Assert Factorio 2.1's rule: one belt per balancer part.

Every edge of a cluster is an interface linked belt standing on the cluster's
own tile, so a part carrying an input on one side and an output on another
carried two belt-connectables on one tile. 2.1 closed the collision-mask
loophole that permitted it (agents/single-edge.md), so the port is a RULE
change: at most one edge per cluster tile, refused through exactly the
sixty-fifth belt's machinery when an edit asks for more.

Two halves, and the second is the one with teeth:

  THE RULE WORKS.  Four single-edge shapes -- 1->1, 2->2, 4->4 and an
                   asymmetric 3->5 -- deliver against a bare express belt in
                   the same save, with the PORT COUNTS asserted before any rate
                   is looked at. A wrong edge list reads as a rate; a wrong
                   SHAPE reads as a port count, and only one of the two can be
                   seen without knowing what the answer should be.
  THE REFUSAL WORKS.  A second belt built against an occupied part, a belt
                   ROTATED onto one (which raises no event at all, so the audit
                   is what finds it), and a part BRIDGING two working balancers
                   into one whose bridging tile would carry two belts. In all
                   three the standing network must be untouched and still
                   delivering, the audit must say `drift=1 unbuilt=0`, exactly
                   one refusal must be issued per distinct edge state, and
                   nothing may be handed back -- a headless run has no players,
                   so a revert firing here would be a revert firing for a
                   script build.

    python3 test/assert-sedge.py create.log run.log
"""

import re
import sys

COMPILED = re.compile(
    r"\[BBB\] compiled cluster (\d+) (\d+)->(\d+) over (\d+) ports, (\d+) entities"
)
REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt, "
    r"worst (\d+)"
)
OVERLIMIT = re.compile(r"\[BBB\] alert: cluster (\d+) would need (\d+) ports")
TOLDFORCE = re.compile(
    r"\[BBB\] told force (\d+) that cluster (\d+) is past the one-belt-per-part "
    r"rule(.*)$"
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
SPARED = re.compile(
    r"\[BBB\] cluster (\d+) would merge into a cluster this mod cannot build; "
    r"left (\d+) standing network\(s\) alone"
)
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
TOOKBACK = re.compile(r"\[BBB\] cluster (\d+) took back (\d+) items into its new network")
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+)"
)
AUDITED = re.compile(r"\[BBB-SEDGE\] audited tag=(\S+)")
SAMPLE = re.compile(r"\[BBB-SEDGE\] sample tag=(\S+) tick=(\d+) (.*)")
TILE = re.compile(r"\[BBB-SEDGE\] tile tag=(\S+) at=(-?\d+),(-?\d+) holds=\[(.*)\]")

# The eight shapes the save's `on_init` builds, as an exact multiset of
# (inputs, outputs, ports). Asserted BEFORE any rate is read, because a rig
# whose belts landed somewhere other than where the geometry intended can still
# deliver a plausible-looking number: three 1->1s (s11 and the two halves of
# smrg), three 2->2s (s22, sbld, srot), one 4->4 and one 3->5.
#
# The port counts are the planner's own `P = next_pow2(max(N, M))` and are the
# thing a single-edge rebuild of an old multi-edge rig would get wrong: the same
# balancer over twice as many parts must be the same MACHINE.
SHAPE_EXPECT = sorted([
    ("1", "1", "1"), ("1", "1", "1"), ("1", "1", "1"),
    ("2", "2", "2"), ("2", "2", "2"), ("2", "2", "2"),
    ("4", "4", "4"),
    ("3", "5", "8"),
])

# tag -> (clusters, parts, nets, drift, unbuilt).
#
# `t0`'s audit runs inside on_init's marker dispatch and reports BEFORE the
# drain it forces compiles anything, so it sees eight compilable clusters as
# unbuilt and no networks at all.
#
# A refused cluster still HAS its network and knows its edge list has moved past
# what the mod can build, which is exactly what drift means -- so `drift=1
# unbuilt=0` and never the other way round. `drift=0 unbuilt=1` there would be
# the signature of a refusal that demolished first and asked afterwards.
AUDIT_EXPECT = {
    "t0":             (8, 36, 0, 0, 8),
    "built":          (8, 36, 8, 0, 0),
    # sbld's second belt: one cluster's edge list has moved.
    "post-sbld":      (8, 36, 8, 1, 0),
    # ... and srot's rotation, which raised nothing, so THIS audit is what finds
    # it. Two refused clusters standing at once.
    "post-rot":       (8, 36, 8, 2, 0),
    # Rotated back: the edge list is the one the netInfo never lost.
    "post-rot-back":  (8, 36, 8, 1, 0),
    # The bridge: two clusters became one, and the merge was refused with both
    # standing networks left alone -- so `nets` still counts two of them, under
    # keys that are no longer roots.
    "post-merge":     (7, 37, 8, 2, 0),
    "post-unmerge":   (8, 36, 8, 1, 0),
    "final":          (8, 36, 8, 1, 0),
}

# rig -> its total delivery as a multiple of one saturated express belt.
RATE_EXPECT = {
    "s11": 1.0,
    "s22": 2.0,
    "s44": 4.0,
    "s35": 3.0,
    "sbld": 2.0,
    "srot": 2.0,
    "smrg": 2.0,
}

SPREAD = 0.01
RATE_TOL = 0.02


def parse_sample(text):
    out = {}
    for chunk in text.split():
        name, _, csv = chunk.partition("=")
        out[name] = [int(v) for v in csv.split(",")]
    return out


def window(lines, start, end):
    """The lines strictly between two markers, both matched as substrings."""
    out, on = [], False
    for line in lines:
        if start in line:
            on = True
            continue
        if on:
            if end in line:
                return out
            out.append(line)
    return out


def main():
    def read(path):
        with open(path, errors="replace") as f:
            return list(f)

    create = read(sys.argv[1])
    bench = read(sys.argv[2])
    lines = create + bench

    fail = []

    # ---- the shapes, before any rate ---------------------------------------
    #
    # OVER THE CREATE LOG ALONE, because that is where the save's own geometry
    # is: `--create` never reaches a tick, so the audit marker at the end of
    # on_init is what compiles all eight rigs into the save. Anything the
    # BENCHMARK phase compiles is an edit's doing and is asserted where that
    # edit is.
    compiles = [m.groups() for m in (COMPILED.search(l) for l in create) if m]
    shapes = sorted((n, m_, p) for _, n, m_, p, _e in compiles)
    for n, m_, p in shapes:
        print("  compiled %s->%s over %s ports" % (n, m_, p))
    if shapes != SHAPE_EXPECT:
        fail.append("the save compiled %r, expected %r. A single-edge rig whose "
                    "belts did not land where the geometry intended still "
                    "delivers a number; the port count is what says the machine "
                    "is the one that was asked for"
                    % (shapes, SHAPE_EXPECT))

    # ---- the two refusal tiles really hold what the legs are about ---------
    tiles = {m.group(1): (m.group(2), m.group(3), m.group(4))
             for m in (TILE.search(l) for l in lines) if m}
    for tag, want in (("init", ""),
                      ("post-sbld", "express-transport-belt"),
                      ("post-rot", "express-transport-belt"),
                      ("post-merge", "bbb-balancer-part")):
        got = tiles.get(tag)
        print("  tile %-11s %s" % (tag, got))
        if got is None:
            fail.append("no tile report for tag=%s" % tag)
        elif want and want not in got[2]:
            fail.append("the %s tile holds [%s] and must hold a %s: the edit "
                        "this leg is about did not happen, and every assertion "
                        "below it would pass vacuously" % (tag, got[2], want))
        elif not want and got[2] != "":
            fail.append("the merge gap tile already holds [%s] before the leg "
                        "runs" % got[2])

    # ---- the refusals: one per distinct edge state, and no others ----------
    refused = [m for m in (REFUSED.search(l) for l in lines) if m]
    for m in refused:
        print("  refused: %s" % m.group(0).strip()[:96])
    if len(refused) != 3:
        fail.append("the guest refused %d compile(s) for the one-belt-per-part "
                    "rule, expected exactly three -- one per leg, and once per "
                    "distinct edge state rather than once per audit: %r"
                    % (len(refused), [m.group(0).strip()[:70] for m in refused]))
    else:
        # Every one of the three is a single part with exactly two belts on it.
        for m in refused:
            if m.group(2) != "1" or m.group(3) != "2":
                fail.append("a refusal reports %s tiles at worst %s belts; every "
                            "leg here puts a SECOND belt on ONE part, so the "
                            "numbers are 1 and 2: %s"
                            % (m.group(2), m.group(3), m.group(0).strip()))
    if any(OVERLIMIT.search(l) for l in lines):
        fail.append("something in this save was refused for the PORT limit; no "
                    "rig here is anywhere near sixty-four belts, so the two "
                    "bounds are being confused for each other")

    # ---- and that somebody was told, on the one arm a headless run reaches --
    told = [m for m in (TOLDFORCE.search(l) for l in lines) if m]
    for m in told:
        print("  told: %s" % m.group(0).strip()[:88])
    if len(told) != 3 or any("FAILED" in m.group(3) for m in told):
        fail.append("the force was told about %d refusal(s)%s, expected three "
                    "clean reports. Every build here is a script build, so the "
                    "fork always takes force.print -- which is what says the "
                    "LocalisedString crossed and the LuaForce resolved from a "
                    "force INDEX"
                    % (len(told),
                       " (print FAILED)" if any("FAILED" in m.group(3)
                                                for m in told) else ""))
    elif {m.group(2) for m in told} != {m.group(1) for m in refused}:
        fail.append("the clusters told (%r) are not the clusters refused (%r)"
                    % (sorted(m.group(2) for m in told),
                       sorted(m.group(1) for m in refused)))

    # THE STANDING NEGATIVE. A headless --create has no players at all, so
    # `game.get_player` resolves to nothing and revertOne returns before it
    # mines anything. A hand-back here would be a revert firing for a script
    # build, which is the one thing the feature must never do.
    if any(HANDEDBACK.search(l) for l in lines):
        fail.append("a refused piece was handed back in a run with no players; "
                    "revertOne fired for a script build")
    else:
        print("  hand-backs over the whole run: 0")

    # ---- the merge: both standing networks untouched -----------------------
    spared = [m for m in (SPARED.search(l) for l in lines) if m]
    print("  spared: %s" % ([int(m.group(2)) for m in spared] or "NONE"))
    if len(spared) != 1 or int(spared[0].group(2)) != 2:
        fail.append("the guest spared %r networks (expected exactly one report "
                    "of 2): a merge the compiler will refuse must take BOTH "
                    "predecessors' teardowns off the queue, and must do it once"
                    % [int(m.group(2)) for m in spared])

    merge_win = window(lines, "merge-add begin", "merge-add end")
    if not merge_win and not any("merge-add begin" in l for l in lines):
        fail.append("the merge leg did not run")
    tore = [m.group(1) for m in (TORNDOWN.search(l) for l in merge_win) if m]
    built = [m.group(1) for m in (COMPILED.search(l) for l in merge_win) if m]
    spills = [m.group(1) for m in (SPILLED.search(l) for l in merge_win) if m]
    print("  across the merge: teardowns=%d builds=%d spills=%d"
          % (len(tore), len(built), len(spills)))
    if tore or built or spills:
        fail.append("the merge tore down %r, built %r and spilled %r. Both "
                    "halves' fingerprints are the ones their netInfos already "
                    "hold, so the right number of each is zero: a refused merge "
                    "must leave two running balancers exactly as it found them"
                    % (tore, built, spills))

    # ---- and mining the bridge back out ------------------------------------
    #
    # ONE HALF SKIPS AND THE OTHER IS RECOMPILED, and which is which is decided
    # by node ids rather than by geometry -- so what is asserted is the pair,
    # not one of them. `removePart` marks the OLD ROOT dead unconditionally, and
    # the old root of an un-merging cluster is whichever of the three nodes
    # union-find kept: the bridging part's own brand-new node (whose network
    # never existed, so the teardown is a no-op and BOTH halves skip) or one of
    # the predecessors' roots (whose network is real, so that half is torn down
    # and rebuilt). This save reaches the second case, because nothing in it
    # frees a node id before the bridging part is placed and `newNode` therefore
    # hands it the highest one. CLAUDE.md's "The merge that would be over the
    # limit" records that the key is not fixed and must not be assumed; what it
    # also says -- that mining the bridge back out costs zero teardowns -- is
    # true only of the first case.
    #
    # What matters is the same either way and is what the numbers below pin: the
    # recompile PUTS BACK everything it drained, nothing spills, and the other
    # half is not touched at all.
    un_win = window(lines, "merge-remove begin", "mark tag=post-unmerge")
    if not un_win:
        un_win = window(lines, "merge-remove begin", "audited tag=post-unmerge")
    if not un_win:
        fail.append("the un-merge window is empty: the leg did not run")
    un_tore = [(m.group(1), int(m.group(2)))
               for m in (TORNDOWN.search(l) for l in un_win) if m]
    un_built = [m.group(1) for m in (COMPILED.search(l) for l in un_win) if m]
    un_took = [(m.group(1), int(m.group(2)))
               for m in (TOOKBACK.search(l) for l in un_win) if m]
    un_spill = [m.group(1) for m in (SPILLED.search(l) for l in un_win) if m]
    print("  across the un-merge: teardowns=%r builds=%r took-back=%r spills=%d"
          % (un_tore, un_built, un_took, len(un_spill)))
    if len(un_tore) != 1 or len(un_built) != 1 or un_spill:
        fail.append("the un-merge tore down %r, built %r and spilled %r: the "
                    "half whose root survived the merge is recompiled and the "
                    "other is a SKIP, so the right numbers are one, one and none"
                    % (un_tore, un_built, un_spill))
    elif un_built[0] != un_tore[0][0]:
        fail.append("the un-merge tore down cluster %s and rebuilt cluster %s"
                    % (un_tore[0][0], un_built[0]))
    elif un_took != un_tore:
        fail.append("the un-merge drained %r and put back %r: a recompile is "
                    "not a removal, so every item the teardown took has to go "
                    "back inside the network it rebuilds" % (un_tore, un_took))

    # NOTHING IN THE WHOLE RUN MAY SPILL. Every refusal here happens in front of
    # its teardown, so no network ever comes down for one -- and the un-merge's
    # recompile, whichever form it takes, must put back what it drained.
    allspills = [m.group(0).strip() for m in (SPILLED.search(l) for l in lines) if m]
    if allspills:
        fail.append("%d spill(s) over the whole run: %r. A refusal that reaches "
                    "the spill path is a refusal that demolished something"
                    % (len(allspills), allspills[:4]))
    else:
        print("  spills over the whole run: 0")

    # ---- the audits --------------------------------------------------------
    audits, last = {}, None
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
        print("  audit %-14s %s" % (tag, (got,)))
        if got is None:
            fail.append("no audit for tag=%s" % tag)
        elif got != want:
            fail.append("audit %s is clusters=%d parts=%d nets=%d drift=%d "
                        "unbuilt=%d, expected %r" % ((tag,) + got + (want,)))

    # ---- the rates ---------------------------------------------------------
    samples = {m.group(1): parse_sample(m.group(3))
               for m in (SAMPLE.search(l) for l in lines) if m}
    for a, b, what in (("e", "f", "the settled window"),
                       ("a", "b", "before the merge"),
                       ("c", "d", "while the merge stands refused")):
        if a not in samples or b not in samples:
            fail.append("%s was not sampled at both ends (%s, %s)" % (what, a, b))
            continue
        pre, post = samples[a], samples[b]
        delta = {k: [y - x for x, y in zip(pre[k], post[k])] for k in post if k in pre}
        ctrl = delta["ctrl"][0]
        print("\n  %s: one saturated express belt delivered %d items" % (what, ctrl))
        if ctrl < 100:
            fail.append("the control belt delivered %d items over %s; nothing "
                        "in this save was flowing" % (ctrl, what))
            continue
        rigs = RATE_EXPECT if a == "e" else {"smrg": 2.0}
        for name, want in sorted(rigs.items()):
            per = delta.get(name, [])
            total = sum(per)
            ratio = total / ctrl if ctrl else 0.0
            spread = ((max(per) - min(per)) / (total / len(per))) if total else 1.0
            print("    %-5s %s  total %.3fx one belt, spread %.2f%%"
                  % (name, " ".join(str(v) for v in per), ratio, spread * 100))
            if abs(ratio - want) > RATE_TOL * want:
                fail.append("%s delivered %.3fx one belt over %s and should "
                            "deliver %.3fx" % (name, ratio, what, want))
            if spread > SPREAD:
                fail.append("%s outputs spread %.2f%% over %s, over the %.0f%% "
                            "bound" % (name, spread * 100, what, SPREAD * 100))

    if fail:
        print("\nSINGLE-EDGE ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\none belt per balancer part: the rule builds, and every way of "
          "breaking it is refused with the standing network left alone")


if __name__ == "__main__":
    main()
