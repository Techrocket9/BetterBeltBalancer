#!/usr/bin/env python3
"""Assert what happens when a Factorio 2.0 multi-edge save is opened on 2.1.

The worlds these run on were built by a Factorio 2.0.77 binary that no longer
exists on any machine here, and a 2.1 Factorio refuses to build a multi-edge
balancer at the prototype level -- so the saves are committed
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
GRANDFATHER = re.compile(r"\[BBB\] single-edge: kept multiple belts per part enabled")
GFFAILED = re.compile(r"could not be written")
FLIPPED = re.compile(r"\[BBB\] single-edge: multiple belts per part turned (ON|OFF)")
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")
TOOKBACK = re.compile(r"\[BBB\] cluster (\d+) took back (\d+) items")
HANDEDBACK = re.compile(r"\[BBB\] handed the over-limit piece at ")
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
    r"ground=(\d+) hidden_entities=(\d+) hidden_items=(\d+) iface_items=(\d+)"
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
        "refused": 21,
        "forces": {1: 21},
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
    },
}

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
    print("\na 2.0 multi-edge save opened on 2.1: the remnants came down, their "
          "items are on the ground, every balancer is refused, and each force was "
          "told once\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--fixture", required=True, choices=sorted(FIXTURES))
    ap.add_argument("logs", nargs="+")
    args = ap.parse_args()
    want = FIXTURES[args.fixture]

    text = ""
    for path in args.logs:
        with open(path, encoding="utf-8", errors="replace") as fh:
            text += fh.read()

    print(f"\n  the {args.fixture} fixture, built by Factorio 2.0.77 and opened on 2.1")

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
    samples = {}
    for m in TOTAL.finditer(text):
        samples[m.group(1)] = dict(zip(
            ("parts", "ours", "stacked", "ground", "hidden", "hitems", "vitems"),
            (int(g) for g in m.groups()[1:])))
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
