#!/usr/bin/env python3
"""Assert what `bbb-multi-edge-parts` does to a running world when it moves.

FACTORIO 2.0 ONLY, and the suite that runs this refuses to pretend otherwise: on
2.1 the setting is not defined at all, so there is nothing to flip and
test/run.sh prints a SKIP rather than a pass. What covers the FOLD on every
engine is `go test ./edgemode/`, which proves all eighteen of its states; what
nothing else anywhere covers is the WORLD those states decide about.

FOUR TRANSITIONS, and the third is the one a live session found:

  THE FALSE DEFAULT REFUSES.  `me1` is laid the incumbent's way -- an input on
                one face of a part and an output on the other -- and the save
                opens with the setting off, so it is refused and has no network.
  ON COMPILES WHAT WAS REFUSED.  A refused cluster stores no matching
                fingerprint, so the re-queue the flip handler makes is what
                finally builds it; and a multi-edge balancer BUILT while the
                setting is on compiles straight away.
  OFF IS VETOED WHILE THEY STAND, AND THE VETO IS A NO-OP ON THE WORLD.  Turning
                the setting off with multi-edge balancers standing puts
                `settings.global` at false, and the grandfather pass -- whose
                condition is exactly "the marker is here, the setting is off, and
                there are multi-edge clusters" -- writes it straight back on and
                says why. Not one item may move, not one network may come down,
                and the balancers must go on delivering at their old rate across
                the edit. That last part is the field report: the shipped guest
                CONDEMNED them first, so a full balancer's contents reached the
                floor a tick before the mod decided to keep it working.
  OFF STICKS ONCE NOTHING IS LEFT TO VETO.  With the multi-edge clusters gone the
                same keypress is honoured, the setting stays false, and a second
                belt on a working part is refused again.

AND THE PINGS. The veto names N balancers and has to point at them, with the same
`[gps=x,y,surface]` flash the 2.1 migration summary carries. The count is
asserted against the number of multi-edge clusters and the FIRST PING is asserted
to name a tile one of them really stands on -- a count on its own cannot tell a
list of real pings from a list of plausible ones.

    python3 test/assert-flip.py create.log run.log
"""

import re
import sys

# --- the guest's own lines ---------------------------------------------------
REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt, "
    r"worst (\d+)"
)
FLIPPED = re.compile(
    r"\[BBB\] single-edge: multiple belts per part turned (ON|OFF); (\d+) "
    r"(?:clusters re-queued|standing balancers were built that way)(.*)"
)
GRANDFATHER = re.compile(
    r"\[BBB\] single-edge: kept multiple belts per part enabled for this save -- "
    r"(\d+) balancers use it; settings\.global (\S+) = true"
)
GFFAILED = re.compile(r"could not be written")
REQUEUED = re.compile(
    r"\[BBB\] single-edge: (\d+) clusters re-queued after grandfathering"
)
TOLD = re.compile(
    r"\[BBB\] single-edge: told force (\d+) about (\d+) balancers built to the "
    r"multi-edge rule, (\d+) pings(?:, first \[gps=(-?\d+),(-?\d+),(\S+?)\])?"
    r", charted (\d+)(?: from (-?\d+),(-?\d+) to (-?\d+),(-?\d+))?(.*)"
)
# THE CHART TRIPWIRE. `is_chunk_charted` answers false for everything on a
# headless run -- a force with no players has no chart to write into -- so this
# reports zero before and zero after, with nauvis's own origin chunk as the
# control. See the test mod's chart_state for the measurements behind that.
CHART = re.compile(
    r"\[FLIP\] chart tag=(\S+) force=(\d+) charted=(\d+) of=(\d+) "
    r"nauvis_origin=(\S+) players=(\d+)"
)
REMOTE = re.compile(r"\[BBB\] remote set-multi-edge-parts=(true|false)(.*)")
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")
# EITHER ARM OF THE HAND-BACK, matched on the shape both share rather than on one
# sentence. A NEGATIVE assertion -- a headless run has no players -- and an exact
# regex over a negative is the one shape a rename in the guest can make VACUOUS.
HANDEDBACK = re.compile(r"\[BBB\].*\bpiece at -?\d+,-?\d+")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)"
)

# --- the test mod's ----------------------------------------------------------
SETTING = re.compile(r"\[FLIP\] setting tag=(\S+) value=(\S+)")
WROTE = re.compile(r"\[FLIP\] wrote tag=(\S+) accepted=(\S+) value=(\S+)")
SAMPLE = re.compile(r"\[FLIP\] sample tag=(\S+) tick=(\d+) (.*)")
WORLD = re.compile(
    r"\[FLIP\] world tag=(\S+) tick=(\d+) ground=(\d+) inside=(\d+) "
    r"interfaces=(\d+) hidden=(\d+)"
)

# THE WORLD THE TEST MOD BUILDS, written down here rather than read off anything
# the guest said. A cluster count taken from the guest's own flood fill cannot
# see a fill that fused two rigs, which is the trap the `mig` suite fell into and
# was taken out of.
#
#   ctrl  a bare belt          sok  2 -> 2 over four parts, ONE BELT PER PART
#   me1   2 -> 2 over two parts, the incumbent's idiom, draining freely
#   me2   the same, DEAD-ENDED, and built at t=250 after the setting went on
EXPECT_MULTI = 2
# Every tile a multi-edge rig stands on. A ping names the ROOT's tile and which
# node is the root depends on registration order, so this is a membership test
# rather than an equality -- what it rules out is a ping that points at empty
# ground, which is the scavenger hunt the pings were added to end.
ME_TILES = {(0, 14), (0, 15), (0, 22), (0, 23)}
SURFACE = "bbb-flip"

# WHERE THE FIRST CHARTED BOX HAS TO BE. `me1` is two parts at (0, 14) and
# (0, 15), so its tile box is x 0..0 and y 14..15; a tile covers [x, x+1], so the
# far corner is one past the last tile, and then `chartMargin` = 8 on every side.
EXPECT_FIRST_BOX = (0 - 8, 14 - 8, 0 + 1 + 8, 15 + 1 + 8)

# The rate window SPANS the veto: `b` is 105 ticks before the flip-off and `d` is
# 1,195 after it. A snapshot either side would say the balancers were running
# before and after; this says they never stopped.
WINDOW = ("b", "d")
RATE_TOL = 0.02

# `(clusters, parts, nets, drift, unbuilt, refused)` at each marker, in the order
# the schedule places them. Tuples rather than bounds, because every one of them
# is a fact about a world this script knows the shape of -- and `nets` is
# asserted at every step because `unbuilt=0` is weak evidence: a cluster with
# inputs and no outputs is a legitimate half-built state and is never counted, so
# a rig that quietly lost its network satisfies it.
EXPECT_WALK = [
    (2, 6, 0, 0, 2, 0),   # t0, inside on_init: reports BEFORE the drain it forces
    (2, 6, 1, 0, 0, 1),   # default: sok built, me1 refused at the false default
    (2, 6, 2, 0, 0, 0),   # post-on: the flip re-queued me1 and it compiled
    (3, 8, 3, 0, 0, 0),   # post-me2: a multi-edge balancer built while ON
    (3, 8, 3, 0, 0, 0),   # post-veto: the veto changed NOTHING
    (1, 4, 1, 0, 0, 0),   # post-strip: only sok is left
    (1, 4, 1, 0, 0, 0),   # post-sticks: the honoured flip-off changed nothing
    (1, 4, 1, 1, 0, 1),   # post-second-belt: refused again, network untouched
    (1, 4, 1, 1, 0, 1),   # final
]

fails = []


def check(cond, msg):
    if not cond:
        fails.append(msg)
    return cond


def report():
    if fails:
        print()
        for f in fails:
            print(f"FAIL: {f}", file=sys.stderr)
        print(f"\n{len(fails)} assertion(s) failed", file=sys.stderr)
        sys.exit(1)
    print("\nthe multi-edge setting: refused at the default, compiled when turned "
          "on, VETOED when turned off with balancers standing -- not one item "
          "moved and every balancer still at rate -- and honoured once nothing "
          "was left to veto\n")


def parse_sample(text):
    """`ctrl=123 sok=4,5` -> {'ctrl': [123], 'sok': [4, 5]}"""
    out = {}
    for chunk in text.split():
        name, _, csv = chunk.partition("=")
        out[name] = [int(v) for v in csv.split(",")]
    return out


def main():
    text = ""
    for path in sys.argv[1:]:
        with open(path, encoding="utf-8", errors="replace") as fh:
            text += fh.read()

    print("\n  the multi-edge setting, driven through all four of its transitions")

    setting = {m.group(1): m.group(2) for m in SETTING.finditer(text)}
    wrote = {m.group(1): (m.group(2), m.group(3)) for m in WROTE.finditer(text)}
    world = {m.group(1): tuple(int(g) for g in m.groups()[1:])
             for m in WORLD.finditer(text)}
    samples = {m.group(1): parse_sample(m.group(3)) for m in SAMPLE.finditer(text)}

    # --- the setting exists at all --------------------------------------------
    # A run whose setting was never defined would satisfy several counts below
    # while proving none of them, and `absent` is exactly what a 2.1 engine
    # answers. The suite is skipped there; this is what says the skip worked.
    for tag in ("init", "default", "post-on", "post-veto", "post-sticks", "final"):
        if not check(tag in setting, f"no setting line for tag={tag}"):
            continue
        check(setting[tag] != "absent",
              f"`bbb-multi-edge-parts` reads absent at tag={tag}: this engine does "
              "not define it, so nothing in this suite was exercised at all")
    if fails:
        report()

    # --- transition 1: the false default refuses -------------------------------
    print(f"  the save opens at value={setting['default']}")
    check(setting["init"] == "false" and setting["default"] == "false",
          "the setting did not start at its false default, so the refusal below "
          "is not a refusal AT the default")
    refusals = [(int(m.group(1)), int(m.group(2)), int(m.group(3)))
                for m in REFUSED.finditer(text)]
    print(f"  one-belt-per-part refusals over the whole run: {len(refusals)}")
    check(len(refusals) == 2,
          f"{len(refusals)} clusters were refused and this run asks for exactly "
          "two: `me1` at the false default the save opens on, and `sok`'s second "
          "belt after the flip-off has stuck. A third means a rig is not the "
          "shape it is named for")
    check(all(r[2] == 2 for r in refusals),
          f"a refusal named a part carrying {[r[2] for r in refusals]} belts; both "
          "gestures here put exactly two on one tile")

    # --- transition 2: ON compiles what was refused ----------------------------
    flips = [(m.group(1), int(m.group(2)), m.group(3)) for m in FLIPPED.finditer(text)]
    kinds = [f[0] for f in flips]
    print(f"  flips the guest saw: {kinds}")
    check(kinds == ["ON", "OFF", "OFF"],
          f"the guest saw {kinds} and the schedule writes ON, OFF (vetoed) and OFF "
          "(sticks). A missing one means the write never reached the handler; an "
          "extra one means the grandfather's own re-entrant write was not absorbed "
          "by the anchor it deliberately writes first")
    for tag, want in (("on", "true"), ("off-vetoed", "false"),
                      ("off-sticks", "false")):
        if not check(tag in wrote, f"no write line for tag={tag}"):
            continue
        accepted, value = wrote[tag]
        check(accepted == "true",
              f"the write at tag={tag} was refused by the mod. `settings.global` "
              "is writable only by the mod that DEFINED the setting, which is why "
              "this goes through remote.call at all")
        # THE VALUE STRAIGHT AFTER THE WRITE, which for the vetoed flip is still
        # FALSE: the handler is synchronous but the veto is settled by the flush it
        # asks for, one tick later. `post-veto` below is where it reads true again.
        check(value == want,
              f"straight after the write at tag={tag} the setting reads {value} "
              f"and should read {want}")
    remotes = [m.group(1) for m in REMOTE.finditer(text)]
    check(remotes == ["true", "false", "false"],
          f"the remote door was called with {remotes} and the schedule calls it "
          "with true, false, false")
    check(not any("REFUSED" in m.group(2) for m in REMOTE.finditer(text)),
          "the remote door refused a write. `writeMultiEdgeSetting` is gated on "
          "the capability marker and on Factorio 2.0 that marker is present")

    # --- transition 3: OFF IS VETOED, and the veto touches nothing -------------
    off = [f for f in flips if f[0] == "OFF"]
    if check(len(off) == 2, "the two flip-offs are not both in the log"):
        veto, sticks = off
        print(f"  the vetoed flip-off found {veto[1]} standing multi-edge balancers")
        check(veto[1] == EXPECT_MULTI,
              f"the flip-off found {veto[1]} multi-edge balancers standing and the "
              f"rigs build {EXPECT_MULTI}: me1, refused at the default and compiled "
              "by the flip, and me2, built while the setting was on")
        check("VETOED" in veto[2],
              "the flip-off did not say it was vetoing. A flip-off that reaches "
              "ActSweep with anything standing is ALWAYS vetoed -- the condition "
              "that makes the scan find something is the condition that makes the "
              "grandfather pass write the setting back on -- so the sweep can "
              "never stick and must never demolish anything on the way")
        check(sticks[1] == 0,
              f"the second flip-off found {sticks[1]} multi-edge balancers standing "
              "and the rigs were stripped before it. A flip-off with nothing to "
              "veto is the only one that is honoured")

    gf = list(GRANDFATHER.finditer(text))
    print(f"  the setting was written back on {len(gf)} time(s)")
    check(len(gf) == 1,
          f"the guest wrote the setting back on {len(gf)} times and this run vetoes "
          "exactly one flip-off. Zero means the veto did not happen at all; two "
          "means the second flip-off was vetoed as well, which it must not be")
    if gf:
        check(int(gf[0].group(1)) == EXPECT_MULTI,
              f"the veto named {gf[0].group(1)} balancers and {EXPECT_MULTI} were "
              "standing")
        check(gf[0].group(2) == "bbb-multi-edge-parts",
              f"the veto wrote {gf[0].group(2)}")
    check(not GFFAILED.search(text),
          "the write back to true FAILED. On 2.0 the setting is defined and the "
          "capability gate is open, so nothing here may decline it")
    req = list(REQUEUED.finditer(text))
    check(len(req) == 1,
          f"{len(req)} re-queues after grandfathering, and a veto owes exactly one: "
          "the clusters it has just declined to demolish go back on the queue, "
          "where each one SKIPS on the fingerprint it never lost")

    check(setting["post-veto"] == "true",
          f"the setting reads {setting['post-veto']} after the vetoed flip-off, and "
          "a veto means it goes back on")
    check(setting["post-sticks"] == "false" and setting["final"] == "false",
          "the second flip-off did not stick, and with no multi-edge balancer "
          "standing there was nothing to veto it")

    # THE FIELD REPORT ITSELF: not one item may move.
    #
    # GATED ON ITS OWN INPUTS AND NEVER ON `fails`. A block that runs only when
    # nothing has failed yet is a block an unrelated failure can silence, and
    # these are the assertions the whole suite exists for -- the first version of
    # this script gated them that way and the red proof below printed three
    # failures with the ground total never once compared.
    veto_tags = ("pre-veto", "post-veto", "veto-settled", "d")
    have_world = all(check(t in world, f"no world sample for tag={t}")
                     for t in veto_tags)
    if have_world:
        pre, post, settled, later = (world[t] for t in veto_tags)
        print("  across the veto: ground {}->{}->{}->{}, inside {}->{}->{}->{}, "
              "interfaces {}->{}->{}->{}, hidden {}->{}->{}->{}".format(
                  pre[1], post[1], settled[1], later[1],
                  pre[2], post[2], settled[2], later[2],
                  pre[3], post[3], settled[3], later[3],
                  pre[4], post[4], settled[4], later[4]))
        check(pre[1] == 0,
              f"{pre[1]} items were on the ground before the veto, so nothing after "
              "it can be attributed to the veto")
        check(post[1] == 0 and settled[1] == 0 and later[1] == 0,
              f"THE VETO PUT ITEMS ON THE GROUND: {pre[1]} -> {post[1]} -> "
              f"{settled[1]} -> {later[1]}. A vetoed flip is a no-op on the world. "
              "This is the 2026-08-24 field report, and what produced it was the "
              "flip-off CONDEMNING every multi-edge network and spilling it a tick "
              "before the mod decided to keep it working")
        check(pre[2] > 0,
              "the networks held nothing at all before the veto, so a spill could "
              "not have shown up in this leg however broken the guest was")
        check(post[2] == pre[2] and settled[2] == pre[2],
              f"the items standing INSIDE the networks moved across the veto: "
              f"{pre[2]} -> {post[2]} -> {settled[2]}. `me2` is dead-ended, so its "
              "network is full and static, and nothing about a declined keypress "
              "may touch it")
        check(post[3] == pre[3] and post[4] == pre[4] and
              settled[3] == pre[3] and settled[4] == pre[4],
              f"the compiler's own entities moved across the veto: "
              f"{pre[3]}/{pre[4]} interfaces/hidden -> {post[3]}/{post[4]} -> "
              f"{settled[3]}/{settled[4]}. Nothing was torn down and nothing rebuilt")

    # --- the pings ------------------------------------------------------------
    told = list(TOLD.finditer(text))
    print("  forces told: %s" % [(int(m.group(1)), int(m.group(2)), int(m.group(3)))
                                 for m in told])
    check(len(told) == 1,
          f"{len(told)} forces were told and every rig here is on `player`: the "
          "message is once per FORCE, never once per balancer and never once per "
          "save")
    for m in told:
        check("FAILED" not in m.group(12),
              f"the message to force {m.group(1)} did not reach it")
        count, pings = int(m.group(2)), int(m.group(3))
        check(count == EXPECT_MULTI,
              f"the veto message named {count} balancers and {EXPECT_MULTI} were "
              "standing")
        check(pings == count,
              f"the message named {count} balancers and carried {pings} pings. The "
              "two are equal until a base has more affected balancers than one "
              "readable chat line can point at, and this one has two")
        check("truncated" not in m.group(12),
              "the ping list was truncated with two balancers in it, which means "
              "the cap moved or the buffer did")
        if check(m.group(4) is not None,
                 "the message carried no ping to check. The guest logs the first "
                 "one verbatim precisely so that this can be a measurement rather "
                 "than an inference from a count"):
            x, y, surf = int(m.group(4)), int(m.group(5)), m.group(6)
            print(f"  first ping: [gps={x},{y},{surf}]")
            check((x, y) in ME_TILES,
                  f"the first ping points at ({x}, {y}) and the multi-edge rigs "
                  f"stand on {sorted(ME_TILES)}. A ping at empty ground is the "
                  "scavenger hunt the pings were added to end")
            check(surf == SURFACE,
                  f"the first ping names surface {surf!r} and the rigs are on "
                  f"{SURFACE!r}")
        # AND THE MOD CHARTED WHAT IT POINTED AT. One box per ping, and the first
        # of them has to be this rig's own cluster grown by the margin -- a count
        # alone would be satisfied by charting the wrong ground N times.
        charted = int(m.group(7))
        check(charted == pings,
              f"the message carried {pings} pings and charted {charted} boxes. A "
              "`[gps=]` opens the map at a coordinate whether or not the force "
              "has ever seen it, and an uncharted coordinate is black, so every "
              "ping in a message a player can click has to be charted")
        if check(m.group(8) is not None,
                 "no charted box was logged, so the margin and the geometry "
                 "behind the pings are unasserted"):
            box = tuple(int(m.group(i)) for i in (8, 9, 10, 11))
            print(f"  first charted box: {box}")
            check(box == EXPECT_FIRST_BOX,
                  f"the first charted box is {box} and the first pinged cluster "
                  f"plus the margin is {EXPECT_FIRST_BOX}. It is the CLUSTER's "
                  "box and not the ping's tile: a point would leave the far end "
                  "of a long balancer in the dark")

    # --- the chart tripwire ---------------------------------------------------
    #
    # NOT A MEASUREMENT OF THE FIX, and it says so. `is_chunk_charted` answers
    # false for everything on a headless run: with no players a force has no
    # chart to write into, so `force.chart`, `force.chart_all` over a fully
    # generated surface and a radar all leave it false, and so does NAUVIS'S OWN
    # ORIGIN CHUNK, which every real game charts at world creation. That control
    # is on the same line, which is what makes this a statement about the engine
    # rather than about the mod.
    #
    # So what is asserted is the wall, in both directions -- and the day a
    # Factorio charts headlessly, this fails and asks for the real assertion
    # instead of going on passing. Same shape as the `edge` suite's
    # `player-mine-raise ok=false` probe, and for the same reason.
    charts = {m.group(1): tuple(int(g) if g.isdigit() else g
                                for g in m.groups()[2:]) for m in CHART.finditer(text)}
    for tag in ("default", "post-veto", "final"):
        if not check(tag in charts, f"no chart sample for tag={tag}"):
            continue
        got, of, nauvis, players = charts[tag]
        print(f"  chart at {tag}: {got} of {of} chunks, nauvis origin {nauvis}, "
              f"{players} players")
        check(players == 0,
              f"{players} players in a headless run. If that is now possible the "
              "chart assertions here can be real ones: assert the pinged chunks "
              "ARE charted after the veto, and delete this tripwire")
        check(nauvis == "false",
              "nauvis's origin chunk is charted for the player force on a "
              "headless run. That is the control this suite's chart numbers rest "
              "on -- if the engine charts anything here now, `force.chart` can be "
              "checked directly and must be")
        check(got == 0,
              f"{got} of the pinged chunks read charted at tag={tag}. Nothing "
              "headless can chart, so a non-zero here means the wall has fallen "
              "and the guest's charting can and must be asserted through "
              "`is_chunk_charted` rather than through its own log line")

    # --- the balancers kept running across it ---------------------------------
    a, b = WINDOW
    if check(a in samples and b in samples,
             f"no delivery sample for {a} or {b}"):
        sa, sb = samples[a], samples[b]
        belt = sb["ctrl"][0] - sa["ctrl"][0]
        print(f"  over t={a}..{b}, which spans the veto, one bare express belt "
              f"delivered {belt} items")
        check(belt > 0,
              "the control belt delivered nothing, so every ratio below is against "
              "zero")
        for rig, want in (("sok", 2.0), ("me1", 2.0)):
            per = [sb[rig][i] - sa[rig][i] for i in range(len(sb[rig]))]
            total = sum(per) / belt if belt else 0
            mean = sum(per) / len(per)
            spread = (max(per) - min(per)) / mean if mean else 1
            print(f"    {rig}: {per} -- {total:.3f}x one belt, spread {spread:.2%}")
            check(abs(total - want) <= RATE_TOL * want,
                  f"{rig} delivered {total:.3f}x one belt over a window that spans "
                  f"the vetoed flip-off and owes {want:.1f}x. A declined keypress "
                  "may not cost a balancer a single item")
            check(spread <= 0.02,
                  f"{rig}'s outputs spread {spread:.2%}, and a balancer balances")

    # --- and only the strip destroyed anything --------------------------------
    torn = list(TORNDOWN.finditer(text))
    spills = list(SPILLED.finditer(text))
    print(f"  teardowns over the whole run: {len(torn)}; spills: {len(spills)}")
    # THE STRIP IS THE ONLY WINDOW, and it is a REMOVAL: the multi-edge parts are
    # destroyed outright so that the second flip-off has nothing to veto, and a
    # removal spills, which is what this mod has done since "A recompile is not a
    # removal". Two clusters, so two of each and no more.
    check(len(torn) == EXPECT_MULTI,
          f"{len(torn)} networks came down over the whole run and only the strip "
          f"may take any down: {EXPECT_MULTI} clusters are destroyed there, and a "
          "teardown anywhere else is a flip that touched the world")
    check(len(spills) == EXPECT_MULTI,
          f"{len(spills)} spills, and only the strip's two removals may make one")
    check(not HANDEDBACK.search(text),
          "a piece was handed back to a player. There is no player in a headless "
          "run, so `revertOne` must return before it mines anything")

    # --- the audit walk -------------------------------------------------------
    walk = [tuple(int(g) for g in m.groups()) for m in AUDIT.finditer(text)]
    print("  audit walk:")
    for got in walk:
        print("    clusters={} parts={} nets={} drift={} unbuilt={} refused={}"
              .format(*got))
    check(walk == EXPECT_WALK,
          "the audit walk is\n    %s\n  and the rigs and the schedule imply\n"
          "    %s" % (walk, EXPECT_WALK))

    report()


if __name__ == "__main__":
    main()
