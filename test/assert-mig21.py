#!/usr/bin/env python3
"""Assert what happens when a Factorio 2.0 multi-edge save is opened.

ON WHICH ENGINE IS THE WHOLE QUESTION, and the two answers are opposites. The
suite is one pair of committed saves and one observer; `--engine` says which
Factorio is running and therefore which of the two arms below is the correct
outcome.

  2.1  THE MIGRATION. The engine deletes all but one belt-connectable per tile
       at load, silently; the standing networks cannot be adopted; every balancer
       is refused, its remnant torn down, and what it held spilled beside it, and
       each owning force gets one checklist with a ping per balancer.
  2.0  THE GRANDFATHER. Nothing is pruned, every interface is still standing, and
       the adoption comparison therefore matches exactly -- so all of them are
       ADOPTED, nothing comes down, nothing is spilled, nothing is refused, and
       the mod writes its own setting ON to keep the save working and says so.

Those are not two readings of one behaviour: they are the two arms of one `if`
over the capability marker (guest/go/sedge.go), and until a 2.0 binary was
available again only the first had ever been executed.

The worlds these run on were built by a Factorio 2.0.77 binary and a 2.1
Factorio cannot rebuild them at any price, so the saves are committed
(test/fixtures-2.0/README.md) and each one IS phase one. There is no `--create`
in this suite and there cannot be.

WHAT THE LOAD ARRIVES INTO IS NOT WHAT THE SAVE HELD. The engine gets there
first: opening one of these under 2.1.14 silently deletes all but one
belt-connectable per tile, with no log line of any kind, and leaves the hidden
networks fully intact. So the guest wakes into balancers whose standing networks
are missing most of their interfaces and whose surviving ports are a lottery.
The first thing checked here is that the world really did arrive in that state,
because every number after it is about what the mod then does to it.

FOUR CLAIMS, and the third is the one that needed new code:

  THE HEAP WAS DECLINED.  A leg that silently adopted the saved guest heap would
                  never run the migration at all, and would pass every count
                  below by measuring nothing. The rebuild-from-world line is
                  required rather than assumed.
  THE ENGINE PRUNED IT.  One interface per part tile and zero stacked tiles
                  before any script ran, with the hidden networks still whole.
  THE REMNANTS CAME DOWN, ITEMS AND ALL.  A refusal normally leaves the standing
                  network alone, because the machine is fine and only the
                  requested edit is not. Here the machine is the thing that
                  cannot exist, so it is torn down FIRST and refused afterwards:
                  every interface and every hidden entity gone, everything they
                  held recovered exactly, and all of it spilled -- a refused
                  compile claims nothing back.
  THE PLAYER WAS TOLD, ONCE PER FORCE.  Not once per balancer, and not with the
                  ordinary "the extra piece was left in place" sentence, which
                  would be a lie: nobody placed anything.

AND ONE NEGATIVE, WHICH IS THE HALF THIS ENGINE EXISTS TO PIN. The grandfather
pass -- the 2.0 arm, which keeps a multi-edge save working by writing this mod's
own setting -- must NEVER be attempted here. Writing a `settings.global` key the
engine does not define raises, and a 2.0 save opened on 2.1 is full of exactly
the clusters that pass looks for. So a run that wrote the setting, or tried to,
fails. guest/go/edgemode proves the fold that decides it under `go test`; this
proves the fold is what the guest actually asks.

    python3 test/assert-mig21.py --fixture m2 run.log
"""

import argparse
import re
import sys

REBUILT = re.compile(
    r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters "
    r"\((\d+) networks adopted, (\d+) rebuilt\)"
)
WASREBUILT = re.compile(r"\[BBB\] the mod was rebuilt")
REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt, "
    r"worst (\d+)"
)
SUMMARY = re.compile(
    r"\[BBB\] single-edge: (\d+) balancers were built with several belts per part"
)
TOLD = re.compile(
    r"\[BBB\] single-edge: told force (\d+) about (\d+) balancers built to the "
    # No `$`: this is matched against the whole log as one string, where `$`
    # without re.MULTILINE anchors to the END OF THE FILE. `.` does not cross a
    # newline, so the trailing group already stops at the end of the line -- which
    # is what the ` -- print FAILED` suffix has to be caught by.
    r"multi-edge rule(.*)"
)
# THE PING LIST, which both arms carry. The count and the first ping are logged
# because `force.print` goes to the game's chat and no script can read it back:
# without them a suite could say a message was sent and nothing at all about
# whether it pointed anywhere.
PINGS = re.compile(
    r"multi-edge rule, (\d+) pings(, list truncated)?"
    r"(?:, first \[gps=(-?\d+),(-?\d+),(\S+?)\])?"
)
REQUEUED = re.compile(
    r"\[BBB\] single-edge: (\d+) clusters re-queued after grandfathering"
)
GRANDFATHER = re.compile(r"\[BBB\] single-edge: kept multiple belts per part enabled")
GFFAILED = re.compile(r"could not be written")
FLIPPED = re.compile(r"\[BBB\] single-edge: multiple belts per part turned (ON|OFF)")
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")
TOOKBACK = re.compile(r"\[BBB\] cluster (\d+) took back (\d+) items")
# EITHER ARM OF THE HAND-BACK, matched on the shape both of them share rather
# than on one sentence. This is a NEGATIVE assertion -- a headless --create has
# no players, so `revertOne` returns before it mines anything -- and an exact
# regex over a negative is the one shape a rename in the guest can make
# VACUOUS: the line stops matching, the assertion stops being able to fail, and
# nothing says so. "piece at x,y" is what `handed the refused piece at 4,7 (over
# the port limit) back to player 1` and its could-not-be-handed-back twin have
# in common, and nothing else in this guest's vocabulary produces it.
HANDEDBACK = re.compile(r"\[BBB\].*\bpiece at -?\d+,-?\d+")
# The ordinary per-piece refusal message. It must NOT be used for a migration:
# nothing was placed, so "the extra piece was left in place" is a sentence about
# an event that never happened.
TOLDPIECE = re.compile(r"\[BBB\] told force \d+ that cluster \d+ is past the")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)"
)
SEEDED = re.compile(r"\[MIG21\] seeded hidden=(\d+) visible=(\d+) total=(\d+)")
TOTAL = re.compile(
    r"\[MIG21\] total tag=(\S+) parts=(\d+) ours=(\d+) stacked_tiles=(\d+) "
    r"ground=(\d+) hidden_entities=(\d+) hidden_items=(\d+) iface_items=(\d+) "
    r"delivered=(\d+)"
)

# WHAT EACH COMMITTED FIXTURE HOLDS, WRITTEN DOWN HERE RATHER THAN READ OFF THE
# GUEST'S OWN SUMMARY LINE. The guest's cluster count and the world's come from
# the same flood fill, so a fill that fused two clusters would move both together
# and neither would say anything -- which is exactly the trap the `mig` suite's
# cluster count fell into and was taken out of. These numbers come from the
# fixtures' own create logs, written by the 2.0 binary that made the saves.
FIXTURES = {
    "m2": {
        "parts": 77,
        "clusters": 21,
        "surfaces": 4,
        # Every one of the 21 rigs is multi-edge, and there is no single-edge
        # balancer anywhere in this world: even a 1->1 over one part carries two
        # belts, which is the whole reason the rule forces a geometry change.
        # VERIFIED rather than assumed, from the fixture's own committed create
        # log: the 21 compiled shapes are 1->1 over one part, seven 2->2 over two,
        # and thirteen larger ones, and every one of them puts an input on one
        # face of a part and an output on another.
        "refused": 21,
        "forces": {1: 21},
        # WHICH WAY THE CHEST TOTAL MOVES WHILE THE BALANCERS RUN, which is a
        # fact about the fixture's own rigs and not about this mod. `m2` feeds
        # from INFINITY chests and drains into ordinary ones, so the total a
        # container census sees only ever rises. See `edge` for the other case.
        "chests": "rise",
    },
    "edge": {
        "parts": 95,
        "clusters": 15,
        "surfaces": 3,
        "refused": 15,
        # TWO FORCES, which is the shape this fixture adds over m2: the edge suite
        # builds on a second force, and two forces' parts touching are two
        # balancers -- so the summary is spoken twice, once to each.
        "forces": {1: 14, 4: 1},
        # ...AND THE OTHER DIRECTION, because every source in the `edge` world is a
        # FINITE steel chest -- that is what makes its conserved-total assertions
        # possible at all -- and most of its rigs are dead-ended. So a container
        # census there is dominated by the sources DRAINING, and the total falls
        # while the balancers run. Written down per fixture rather than softened
        # to "changed", because "the number moved" is satisfied by a leak.
        "chests": "fall",
    },
}

fails = []


def check(cond, msg):
    if not cond:
        fails.append(msg)
    return cond


FIELDS = ("parts", "ours", "stacked", "ground", "hidden", "hitems", "vitems",
          "delivered")

# The compiler's own surface, which is the one a `[gps=]` must never name.
HIDDEN_SURFACE = "bbb-hidden"


def world_samples(text):
    return {m.group(1): dict(zip(FIELDS, (int(g) for g in m.groups()[1:])))
            for m in TOTAL.finditer(text)}


def report(engine="2.1"):
    if fails:
        print()
        for f in fails:
            print(f"FAIL: {f}", file=sys.stderr)
        print(f"\n{len(fails)} assertion(s) failed", file=sys.stderr)
        sys.exit(1)
    if engine == "2.0":
        print("\na 2.0 multi-edge save opened on 2.0: every balancer was adopted "
              "whole, nothing came down, nothing reached the ground, and the mod "
              "kept the save working and said so\n")
    else:
        print("\na 2.0 multi-edge save opened on 2.1: the remnants came down, "
              "their items are on the ground, every balancer is refused, and each "
              "force was told once\n")


def grandfather_arm(text, want):
    """THE 2.0 ARM: the same save, on the engine that built it.

    Nothing about this is the 2.1 arm with different numbers. It is the other
    branch of the capability marker, and every headline assertion inverts:

      the engine PRUNES NOTHING, so the tiles that carried two belt-connectables
        still do -- which is the original architecture and the premise the whole
        multi-edge mode rests on;
      the standing interfaces therefore MATCH the re-derived edge list, so every
        cluster is ADOPTED and none is rebuilt;
      NOTHING is condemned, torn down, spilled or refused;
      and the grandfather pass writes this mod's own setting ON, re-queues every
        cluster (where each one skips on the fingerprint it never lost) and tells
        each owning force once, with a ping per balancer.

    THE POSITIVE THAT WAS UNREACHABLE. `mig21` on 2.1 asserts the NEGATIVE of all
    of this -- no grandfather line, no failed write, no setting-changed line --
    because writing a settings key the engine does not define RAISES. This is the
    other half, and until a 2.0 binary was available again `go test ./edgemode/`
    was the only machine anywhere that could execute the decision."""
    check(WASREBUILT.search(text) is not None,
          "the guest never said it had been rebuilt: the saved heap was ADOPTED, "
          "so fk_migrate never fired, rebuildFromWorld never ran, and this leg "
          "measured nothing at all")
    reb = REBUILT.search(text)
    if not check(reb is not None, "no rebuild-from-world line"):
        return report("2.0")
    surfaces, parts, clusters, adopted, rebuilt = (int(g) for g in reb.groups())
    print(f"  rebuilt from world: {surfaces} surfaces, {parts} parts, "
          f"{clusters} clusters ({adopted} adopted, {rebuilt} rebuilt)")
    check((parts, clusters, surfaces) ==
          (want["parts"], want["clusters"], want["surfaces"]),
          f"the rebuild found {parts} parts in {clusters} clusters over {surfaces} "
          f"surfaces and the fixture holds {want['parts']}/{want['clusters']}/"
          f"{want['surfaces']}")
    # THE INVERSION. On 2.1 this reads 0 adopted and N rebuilt, because the engine
    # deleted most of the interfaces before any script ran and the comparison
    # cannot match. Here every one of them is still standing.
    check((adopted, rebuilt) == (clusters, 0),
          f"{adopted} clusters were adopted and {rebuilt} rebuilt. On the engine "
          "that BUILT this save nothing has been pruned, so every standing network "
          "matches the edge list re-derived from the world and every one of them "
          "is adopted whole")

    samples = world_samples(text)
    for tag in ("cfg", "t1", "post-audit", "final"):
        check(tag in samples, f"the observer reported no sample for tag={tag}")
    if fails:
        return report("2.0")
    cfg, t1, mid, fin = (samples[t] for t in ("cfg", "t1", "post-audit", "final"))

    print(f"  before: {cfg['ours']} interfaces on {cfg['parts']} part tiles, "
          f"{cfg['stacked']} of them carrying two, {cfg['hidden']} hidden entities")
    # THE PREMISE, AND IT IS THE EXACT OPPOSITE OF THE 2.1 ARM'S. There the first
    # assertion is that no tile carries two; here it is that many do. A run where
    # this engine had started pruning would satisfy nothing below and would be a
    # far bigger finding than a failed assertion.
    check(cfg["stacked"] > 0,
          "not one tile carried two belt-connectables when the first script ran. "
          "This engine does not prune them -- that is what multi-edge IS -- so a "
          "zero here means either the save is not the multi-edge fixture or this "
          "Factorio has started behaving like 2.1")
    check(cfg["ours"] > want["parts"],
          f"{cfg['ours']} interfaces stand over {want['parts']} part tiles. A "
          "multi-edge balancer puts more than one on some of them, so an interface "
          "count equal to the part count is the pruned 2.1 shape")
    check(cfg["hidden"] > 0, "the hidden networks were not standing at all")
    check(cfg["ground"] == 0,
          f"{cfg['ground']} items were already on the ground before the load "
          "settled")

    # --- NOTHING MOVED -------------------------------------------------------
    for tag, s in (("t1", t1), ("post-audit", mid), ("final", fin)):
        check((s["ours"], s["hidden"], s["parts"]) ==
              (cfg["ours"], cfg["hidden"], cfg["parts"]),
              f"the compiler's entities moved at tag={tag}: "
              f"{cfg['ours']}/{cfg['hidden']} interfaces/hidden over "
              f"{cfg['parts']} parts became {s['ours']}/{s['hidden']}/{s['parts']}. "
              "A grandfathered save is one nothing was done to")
        check(s["ground"] == 0,
              f"{s['ground']} items are on the ground at tag={tag}. On this engine "
              "the balancers are kept, so nothing is torn down and nothing spills")
    check(not TORNDOWN.search(text) and not SPILLED.search(text),
          "a network came down or something spilled. Every cluster here was "
          "adopted whole and the grandfather pass keeps them running")
    check(not REFUSED.search(text),
          "a cluster was refused. The setting is written ON before anything is "
          "compiled, so a multi-edge cluster is exactly what this engine builds")

    # --- ...AND THEY ARE STILL RUNNING ---------------------------------------
    # `nothing moved` is satisfied by a save that is frozen. The rigs' sources are
    # infinity chests and their sinks are ordinary ones, so items arriving is what
    # separates a balancer that was kept from one that merely still stands.
    moved = fin["delivered"] - cfg["delivered"]
    print(f"  items in chests: {cfg['delivered']} -> {t1['delivered']} -> "
          f"{fin['delivered']} ({moved:+d} over 300 ticks, expected to "
          f"{want['chests']})")
    check(moved > 0 if want["chests"] == "rise" else moved < 0,
          f"the chest total moved {moved:+d} over 300 ticks and this fixture's "
          f"rigs make it {want['chests']}. Adopting a network and leaving it "
          "standing is not the claim; the claim is that it still works, and items "
          "arriving somewhere is the only thing that says so")
    check(fin["hitems"] > 0,
          "the hidden networks are carrying nothing at all, so nothing is flowing "
          "through them")

    # --- THE GRANDFATHER PASS, which is the positive this engine exists to pin -
    gf = list(GRANDFATHER.finditer(text))
    check(len(gf) == 1,
          f"the guest wrote its own setting on {len(gf)} times and this load owes "
          "exactly one: the save has multi-edge balancers, the setting defaults to "
          "false, and without the flip every one of them would be refused on the "
          "load that adopts them")
    check(not GFFAILED.search(text),
          "the grandfather write FAILED. On this engine the setting is defined and "
          "the capability gate is open")
    req = list(REQUEUED.finditer(text))
    check(len(req) == 1 and int(req[0].group(1)) == want["clusters"],
          f"the grandfather re-queued {[m.group(1) for m in req]} clusters and the "
          f"save has {want['clusters']}. Every grandfather owes the re-queue -- for "
          "an adopted cluster it is a fingerprint skip, and for one that was "
          "refused it is the only thing that will ever give it a network")
    # AND NOT the setting-changed handler's own line. The pass writes the ANCHOR
    # before the setting, so the synchronous event its own write raises lands on
    # agreement and returns without saying anything. A `turned ON` line here would
    # mean that ordering had been lost.
    check(not FLIPPED.search(text),
          "the setting-changed handler announced a flip. The grandfather pass "
          "writes the anchor BEFORE the setting precisely so that its own "
          "re-entrant event finds agreement and does nothing")

    # --- who was told, and where the pings point ------------------------------
    told = {}
    for m in TOLD.finditer(text):
        told[int(m.group(1))] = int(m.group(2))
        check("FAILED" not in m.group(3),
              f"the message to force {m.group(1)} did not reach it")
    print(f"  told: {told}")
    check(told == want["forces"],
          f"the warning went to {told} and this fixture's balancers belong to "
          f"{want['forces']} -- one message per force, never one per balancer")
    pings = list(PINGS.finditer(text))
    check(len(pings) == len(want["forces"]),
          f"{len(pings)} of the {len(want['forces'])} messages carried a ping "
          "count. The warning names N balancers and has to point at them")
    for i, m in enumerate(pings):
        n = int(m.group(1))
        force = sorted(want["forces"])[i]
        print(f"    force {force}: {n} pings, first {m.group(3)},{m.group(4)} on "
              f"{m.group(5)}")
        check(n == want["forces"][force],
              f"force {force} was told about {want['forces'][force]} balancers and "
              f"got {n} pings. They are equal until a base has more of them than "
              "one readable chat line can point at")
        check(m.group(2) is None,
              f"force {force}'s ping list was truncated with "
              f"{want['forces'][force]} balancers in it")
        if check(m.group(3) is not None,
                 f"force {force}'s message carried no ping to check; the guest "
                 "logs the first one verbatim precisely so that this can be a "
                 "measurement rather than an inference from a count"):
            # AND IT POINTS AT A SURFACE THE PLAYER CAN GO TO. This script does
            # not know the fixture's cluster tiles -- they are somebody else's
            # world -- so the tile itself is checked by the `flip` suite, which
            # builds its own. What is checkable here is the surface, and the one
            # that would be wrong is the one the compiler works on: a ping onto
            # `bbb-hidden` sends a player to a place they cannot reach.
            check(m.group(5) != HIDDEN_SURFACE,
                  f"force {force}'s first ping points at the hidden surface, "
                  "which is the compiler's own and is not somewhere a player can "
                  "go")

    check(not TOLDPIECE.search(text),
          "a grandfathered cluster was announced with the ORDINARY refusal "
          "message, which says an extra piece was left in place unconnected. "
          "Nobody placed anything and nothing was refused")
    check(not HANDEDBACK.search(text),
          "a piece was handed back to a player. There is no player in a headless "
          "run and nothing was placed in this one")

    # --- and the state is stable ---------------------------------------------
    audits = AUDIT.findall(text)
    check(len(audits) >= 2, f"{len(audits)} audits; the observer asks for two")
    want_audit = (want["clusters"], want["parts"], want["clusters"], 0, 0, 0)
    for a in audits:
        got = tuple(int(g) for g in a)
        print("  audit clusters={} parts={} nets={} drift={} unbuilt={} "
              "refused={}".format(*got))
        check(got == want_audit,
              f"the audit reads {got} and should read {want_audit}: every cluster "
              "adopted and holding its network, nothing drifting, nothing unbuilt "
              "and nothing refused")
    check(len(set(audits)) == 1, "the audits disagree with each other")

    report("2.0")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--fixture", required=True, choices=sorted(FIXTURES))
    # WHICH FACTORIO IS RUNNING, and therefore which of the two arms is right.
    # test/run.sh reads it off the binary and passes it; there is no default,
    # because a script that guessed would silently assert the wrong outcome and
    # pass on neither engine for the right reason.
    ap.add_argument("--engine", required=True, choices=("2.0", "2.1"))
    ap.add_argument("logs", nargs="+")
    args = ap.parse_args()
    want = FIXTURES[args.fixture]

    text = ""
    for path in args.logs:
        with open(path, encoding="utf-8", errors="replace") as fh:
            text += fh.read()

    print(f"\n  the {args.fixture} fixture, built by Factorio 2.0.77 and opened on "
          f"{args.engine}")
    if args.engine == "2.0":
        return grandfather_arm(text, want)

    # --- the load happened at all, and it happened on a FRESH heap -------------
    check(WASREBUILT.search(text) is not None,
          "the guest never said it had been rebuilt: the saved heap was ADOPTED, "
          "so fk_migrate never fired, rebuildFromWorld never ran, and this leg "
          "measured nothing at all")
    reb = REBUILT.search(text)
    if not check(reb is not None, "no rebuild-from-world line"):
        report()
        return
    surfaces, parts, clusters, adopted, rebuilt = (int(g) for g in reb.groups())
    print(f"  rebuilt from world: {surfaces} surfaces, {parts} parts, "
          f"{clusters} clusters ({adopted} adopted, {rebuilt} rebuilt)")
    check(parts == want["parts"],
          f"the rebuild found {parts} parts and the fixture holds {want['parts']}")
    check(clusters == want["clusters"],
          f"the rebuild made {clusters} clusters and the fixture holds "
          f"{want['clusters']}")
    check(surfaces == want["surfaces"],
          f"the rebuild walked {surfaces} surfaces and the fixture has "
          f"{want['surfaces']}")
    check(adopted + rebuilt == clusters,
          f"{adopted} adopted plus {rebuilt} rebuilt is not the {clusters} clusters "
          "that were found; one was neither")

    # --- the world the ENGINE handed over -------------------------------------
    samples = world_samples(text)
    for tag in ("cfg", "t1", "post-audit", "final"):
        check(tag in samples, f"the observer reported no sample for tag={tag}")
    if fails:
        report()
        return
    cfg, t1, mid, fin = (samples[t] for t in ("cfg", "t1", "post-audit", "final"))

    print(f"  before: {cfg['ours']} interfaces on {cfg['parts']} part tiles, "
          f"{cfg['stacked']} tiles carrying two, {cfg['hidden']} hidden entities")
    check(cfg["stacked"] == 0,
          f"{cfg['stacked']} tiles were still carrying two belt-connectables when "
          "the first script ran. 2.1 is supposed to have deleted all but one per "
          "tile before that, silently, and it is the premise this suite rests on")
    check(cfg["ours"] == want["parts"],
          f"{cfg['ours']} interfaces survived the load over {want['parts']} part "
          "tiles; the engine keeps exactly one per tile")
    check(cfg["hidden"] > 0,
          "the hidden networks were already gone before the mod ran, so there was "
          "nothing for the migration to tear down and this leg is vacuous")
    check(cfg["ground"] == 0,
          f"{cfg['ground']} items were already on the ground before the migration, "
          "so the spill below cannot be attributed to it")

    # --- the remnants came down ------------------------------------------------
    for tag, s in (("t1", t1), ("post-audit", mid), ("final", fin)):
        check(s["ours"] == 0,
              f"{s['ours']} of the compiler's entities are still standing on the "
              f"visible surfaces at tag={tag}. A network built to the multi-edge "
              "rule is a latent engine risk on every load and has to come down")
        check(s["hidden"] == 0,
              f"{s['hidden']} hidden entities are still standing at tag={tag}")
        check(s["parts"] == want["parts"],
              f"{s['parts']} parts at tag={tag}: the migration must take down what "
              "the COMPILER placed and never touch the player's own parts")

    # --- what they were holding ------------------------------------------------
    seed = SEEDED.search(text)
    if not check(seed is not None, "the observer never reported what it seeded"):
        report()
        return
    seeded = int(seed.group(3))
    check(seeded > 0,
          "nothing was seeded into the networks before the migration ran. Either "
          "the observer's on_configuration_changed now runs AFTER this mod's -- in "
          "which case there was nothing left to seed -- or the fixture arrived with "
          "no networks at all. Every item number below would be a vacuous zero")
    check(cfg["hitems"] + cfg["vitems"] == seeded,
          f"{cfg['hitems'] + cfg['vitems']} items were standing in the networks and "
          f"{seeded} were put there")

    returned = sum(int(m.group(2)) for m in TORNDOWN.finditer(text))
    teardowns = len(TORNDOWN.findall(text))
    spilled = sum(int(m.group(1)) for m in SPILLED.finditer(text))
    spills = len(SPILLED.findall(text))
    tookback = sum(int(m.group(2)) for m in TOOKBACK.finditer(text))
    print(f"  seeded {seeded} items; {teardowns} teardowns returned {returned}, "
          f"{spills} spills placed {spilled}, {tookback} went back into a network")
    check(returned == seeded,
          f"the teardowns recovered {returned} of the {seeded} items that were "
          "standing in the networks")
    check(spilled == returned,
          f"{returned} items were recovered and {spilled} were spilled. A refused "
          "compile claims nothing, so every one of them belongs on the ground")
    check(tookback == 0,
          f"{tookback} items were put back INSIDE a network. Every cluster here was "
          "refused, so there is no network for them to go into")
    check(teardowns == want["refused"],
          f"{teardowns} networks were torn down and {want['refused']} clusters in "
          "this fixture are built to the multi-edge rule")

    print(f"  on the ground afterwards: {t1['ground']} of {spilled} -- the rest "
          "landed on the player's own belts, which spill_item_stack allows")
    check(t1["ground"] > 0,
          "nothing at all reached the ground. What a stopped balancer was holding "
          "is precisely what the player is told to go and collect")
    check(t1["ground"] == mid["ground"] == fin["ground"],
          f"the ground total moved after the migration: {t1['ground']} -> "
          f"{mid['ground']} -> {fin['ground']}")

    # --- the refusals ----------------------------------------------------------
    roots = [int(m.group(1)) for m in REFUSED.finditer(text)]
    distinct = sorted(set(roots))
    print(f"  refused: {len(distinct)} clusters over {len(roots)} refusal lines")
    check(len(distinct) > 0,
          "no cluster in this fixture was refused, so nothing this suite asserts "
          "was exercised at all")
    check(len(distinct) == want["refused"],
          f"{len(distinct)} clusters were refused and {want['refused']} in this "
          "fixture are built to the multi-edge rule")
    # TWICE EACH, and that is the designed shape rather than a wart: the rebuild
    # refuses with the worst information a refusal will ever have and is forbidden
    # to speak, so it logs and re-queues; the informed flush a tick later refuses
    # again and delivers the one message. See limit.go, refuseAdmit.
    check(len(roots) == 2 * len(distinct),
          f"{len(roots)} refusal lines over {len(distinct)} clusters. The rebuild "
          "logs one and the informed retry logs one, so two each is the shape")

    # --- what the player was told ---------------------------------------------
    summ = SUMMARY.search(text)
    check(summ is not None and int(summ.group(1)) == want["refused"],
          f"the migration summary did not name {want['refused']} balancers")
    told = {}
    for m in TOLD.finditer(text):
        told[int(m.group(1))] = int(m.group(2))
        check("FAILED" not in m.group(3),
              f"the message to force {m.group(1)} did not reach it")
    print(f"  told: {told}")
    check(told == want["forces"],
          f"the summary went to {told} and this fixture's balancers belong to "
          f"{want['forces']} -- one message per force, never one per balancer")
    check(not TOLDPIECE.search(text),
          "a migrated cluster was announced with the ORDINARY refusal message, "
          "which says the extra piece was left in place unconnected. Nobody placed "
          "anything: the world was like this when the save opened")
    check(not HANDEDBACK.search(text),
          "a piece was handed back to a player. There is no player in a headless "
          "run, and nothing was placed in this one")

    # --- THE NEGATIVE: no grandfather, on an engine where the write would raise -
    check(not GRANDFATHER.search(text),
          "the guest tried to KEEP multiple belts per part enabled. "
          "`bbb-multi-edge-parts` is not defined on Factorio 2.1 and writing an "
          "undefined settings key RAISES, so this is the one thing the pass must "
          "never do here")
    check(not GFFAILED.search(text),
          "the guest attempted the grandfather write and it failed, which means the "
          "capability gate did not hold")
    check(not FLIPPED.search(text),
          "the setting-changed handler ran. Nothing can change a setting that this "
          "engine does not define")

    # --- and the state is stable ----------------------------------------------
    audits = AUDIT.findall(text)
    check(len(audits) >= 2, f"{len(audits)} audits; the observer asks for two")
    for a in audits:
        got = tuple(int(g) for g in a)
        print("  audit clusters={} parts={} nets={} drift={} unbuilt={} "
              "refused={}".format(*got))
        check(got == (want["clusters"], want["parts"], 0, 0, 0, want["refused"]),
              f"the audit reads {got} and should read "
              f"{(want['clusters'], want['parts'], 0, 0, 0, want['refused'])}: every "
              "cluster refused, none of them holding a network, and none of them "
              "`unbuilt` -- unbuilt is this guest saying it should have built "
              "something and did not")
    check(len(set(audits)) == 1,
          "the audits disagree with each other. A refused cluster is supposed to be "
          "a stable state, not one that oscillates between teardown and rebuild")

    report()


if __name__ == "__main__":
    main()
