#!/usr/bin/env python3
"""Assert M3's lifecycle behaviour from Factorio's own log.

The test mod drives every path that can change what the compiler compiled from
and logs raw numbers; every judgement is here. Two kinds of evidence:

  * the GUEST's own [BBB] lines -- what it decided, what it rebuilt, what the
    audit found. These are the assertion surface for everything the guest can
    see, exactly as in M1 and M2.
  * [BBB-M3] lines for the things it cannot: what a blueprint captured, what a
    clone left behind, how many items came back out of a stress run.

The throughput assertions are all relative to `ctrl`, a bare express belt in the
same save under the same conditions, measured over the SAME window (t=540 to
t=1500). Every rig below was disturbed in some specific way before that window
opens; what is being asserted is that it came back.

Every rig is built to Factorio 2.1's rule, one belt per balancer part, so a
2-in/2-out rig is FOUR parts: a west column carrying the row's input and an east
column carrying its output. Not one lifecycle claim moved with the geometry --
every rate, every spread and every "exactly zero" below is the number it has
always been -- and what did move is a part count. The rule's own refusal is
asserted here only as a NEGATIVE: this suite drives twelve lifecycle paths and
600 ticks of churn, and every edge list any of them produces has to be one the
mod can build.

    python3 test/assert-m3.py create.log run.log
"""

import re
import sys

SAMPLE = re.compile(r"\[BBB-M3\] t=(\d+) rig=(\w+) out=\[([-\d ]*)\]")
CAPTURED = re.compile(r"\[BBB-M3\] bp captured=\[([^\]]*)\]")
GHOSTS = re.compile(r"\[BBB-M3\] ghosts (placed|revived)=(\d+)")
PASTED = re.compile(r"\[BBB-M3\] paste placed=(\d+) of=(\d+)")
CLONE = re.compile(r"\[BBB-M3\] clone-(area|brush) parts=(\d+) leaked=(\d+)")
BOTS = re.compile(r"\[BBB-M3\] bots built=(\d+)")
SWAPPED = re.compile(r"\[BBB-M3\] swap: belt is now (\S+)")
DIED = re.compile(r"\[BBB-M3\] died: part killed, items (\d+) -> (\d+)")
INSERTED = re.compile(r"\[BBB-M3\] stress inserted=(\d+)")
RECOVERED = re.compile(r"\[BBB-M3\] stress recovered=(\d+)")
RENDERS = re.compile(r"\[BBB-M3\] t=\d+ renders=(\d+) visible_interfaces=(\d+)")

AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+) "
    r"refused=(\d+)"
)
# The one-belt-per-part refusal. It must never fire in this suite: see
# SEDGE_REFUSED below.
SEDGEREFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt")
STATE = re.compile(r"\[BBB\] state clusters=(\d+) parts=(\d+) sizes=([\d,]+)")
COMPILED = re.compile(r"\[BBB\] compiled cluster (\d+) (\d+)->(\d+)")
RECONCILED = re.compile(r"\[BBB\] (area|brush) clone: reconciled surface (\d+) box ")
CLONE_KILL = re.compile(r"\[BBB\] destroyed a cloned network entity")
DROPPED = re.compile(r"\[BBB\] deleted: surface (\d+) gave up (\d+) parts in (\d+) clusters")
HIDDEN_ALERT = re.compile(r"\[BBB\] alert: the hidden surface \S+ is being deleted")
HIDDEN_BACK = re.compile(r"\[BBB\] hidden surface recreated; (\d+) clusters rebuilt")
SURFHIDDEN = re.compile(
    r"\[BBB-M3\] surface-hidden tag=(\w+) force=(\S+) hidden=(true|false)")
FROM_WORLD = re.compile(r"\[BBB\] rebuilt from world: (\d+) surfaces, (\d+) parts, (\d+) clusters")

T0, T1 = 540, 1500

# THE WORLD THE FINAL AUDIT MUST FIND, written down here rather than read off the
# guest's own report. `(clusters, parts, networks)`:
#
#   bbb-m3-a  live 8, clone 4, died 3 (one part killed), bdie 4, noev 4, swap 4,
#             forceA 4, forceB 4, bp 4, paste 4, ghost 4, bots 4
#             = 12 clusters over 51 parts
#   bbb-m3-b  the area clone's 4 and the brush clone's 4 = 2 clusters, 8 parts
#   bbb-m3-s  nothing: the stress phase mines every part it built
#   bbb-m3-doomed  deleted at tick 420
#
# `nets == clusters` IS THE HALF `unbuilt` CANNOT MAKE. A cluster with inputs and
# no outputs is a legitimate half-built state and is never counted unbuilt, so
# `unbuilt=0` is satisfied by a world in which a rig quietly lost its network --
# which is exactly what a mis-classified edge, or a refusal nobody expected,
# would leave behind. Every cluster in this save has both an input and an output
# by construction, so every one of them must HAVE a network.
FINAL_WORLD = (14, 59, 14)

# The cluster sizes at the end of `--create`, as an exact multiset: thirteen
# four-part rigs (clone, died, bdie, noev, swap, forceA, forceB, bp, the doomed
# surface's rig and the four stress rigs) and `live`'s eight.
#
# EXACT RATHER THAN A PREDICATE, because the predicate this replaces --
# "one 4 and the rest 2" -- cannot be written under the one-belt rule: `live` is
# legitimately eight parts, so "no cluster of eight" is no longer available as
# the statement that forceA and forceB did not fuse. A fusion moves the multiset
# (twelve 4s and two 8s), and so does any other rig gaining or losing a part.
CREATE_SIZES = sorted([4] * 13 + [8])

# rig -> (live outputs, what the total should be in units of one saturated belt,
#         allowed shortfall, allowed excess, allowed spread) -- all fractions.
#
# The nominal is NOT "full throughput" for most of these: each rig was damaged
# in a specific way and the nominal is what that damage should leave behind.
EXPECT = [
    # The witness. Nothing is ever done to it, INCLUDING having the surface its
    # hidden network lives on deleted under it at tick 450.
    ("live", 4, 4.0, 0.01, 0.01, 0.01),
    # An area clone and a brush clone were taken of it: the source must be
    # untouched by having been copied.
    ("clone", 2, 2.0, 0.01, 0.01, 0.01),
    # A blueprint was taken of it.
    ("bp", 2, 2.0, 0.01, 0.01, 0.01),
    # Two forces, touching. Neither may be disturbed by the other's compiles --
    # and before the force check reached the COMPILER's flood fill (it was in
    # the registry's but not there), forceA's interfaces were swept away by
    # forceB's first teardown and it delivered nothing at all for 450 ticks.
    ("forceA", 2, 2.0, 0.01, 0.01, 0.01),
    ("forceB", 2, 2.0, 0.01, 0.01, 0.01),
    # Built by reviving ghosts (script_raised_revive).
    ("ghost", 2, 2.0, 0.01, 0.01, 0.01),
    # Built by pasting a blueprint's entities as real entities in one tick.
    ("paste", 2, 2.0, 0.01, 0.01, 0.01),
    # Built by construction robots (on_robot_built_entity). Its belts were only
    # laid at tick 480, so the window opens with the pipeline still filling --
    # several linked-belt hops at ~32 items each.
    ("bots", 2, 2.0, 0.06, 0.01, 0.01),
    # An input belt was killed by die(): one belt in, two out.
    ("bdie", 2, 1.0, 0.01, 0.01, 0.01),
    # An input belt was destroyed with NO EVENT, the cluster was re-classified
    # by an unrelated placement, and then the belt was put back ON THE SAME
    # TILE. If the removal window could leak, that last placement would be
    # classified as absent and this would sit at half.
    ("noev", 2, 2.0, 0.03, 0.01, 0.01),
    # An express input belt was fast-replaced with a FAST one, which is 2/3 the
    # speed: 1 + 2/3 belts in. Missing the recompile entirely reads as 2.0;
    # classifying the new belt as absent reads as 1.0.
    ("swap", 2, 1 + 0.0625 / 0.09375, 0.02, 0.02, 0.01),
    # A part was killed by die() with the network full. The row it served is
    # orphaned -- checked separately, it must be exactly frozen -- and the row
    # that is left runs at one full belt.
    ("died", 1, 1.0, 0.01, 0.01, 0.01),
]


def read(paths):
    lines = []
    for path in paths:
        with open(path, errors="replace") as f:
            lines.extend(f)
    return lines


def first(rx, lines):
    for raw in lines:
        m = rx.search(raw)
        if m:
            return m
    return None


def all_of(rx, lines):
    return [m for m in (rx.search(raw) for raw in lines) if m]


def main():
    create = read(sys.argv[1:2])
    run = read(sys.argv[2:3])
    fail = []

    def need(cond, msg):
        if not cond:
            fail.append(msg)

    # ---------------------------------------------------------------- registry
    # The guest heap is discarded whenever the mod is rebuilt (this mod exports
    # fk_migrate, which is a NOTIFICATION on a fresh heap, and never
    # fk_migrate_adopt -- CLAUDE.md, "Coming back on a heap this build did not
    # write"), so the registry is re-derived from the world. On a fresh map like
    # this one there is no rebuild to be told about and the fallback does it:
    # `registryReady` is false, so the first event of the session scans. What is
    # asserted here is that the mechanism runs at all, unconditionally.
    m = first(FROM_WORLD, create)
    need(m is not None, "the guest never rebuilt its registry from the world")
    if m:
        print("  registry rebuilt from the world at the first event: %s surfaces, "
              "%s parts" % (m.group(1), m.group(2)))

    # -------------------------------------------------------------- blueprints
    m = first(CAPTURED, run)
    need(m is not None, "no blueprint was captured")
    if m:
        names = m.group(1).split()
        ours = sorted({n for n in names if n.startswith("bbb-")})
        need(ours == ["bbb-balancer-part"],
             "a blueprint over a compiled balancer captured %s; the hidden "
             "prototypes must never enter one" % ours)
        need(names.count("bbb-balancer-part") == 4,
             "the blueprint captured %d parts, expected 4 (two rows of two "
             "under the one-belt rule)" % names.count("bbb-balancer-part"))
        need(any(n.endswith("transport-belt") for n in names),
             "the blueprint captured no belts -- was it taken of nothing?")
        print("  blueprint captured %d entities, and of ours only "
              "bbb-balancer-part" % len(names))

    ghosts = {m.group(1): int(m.group(2)) for m in all_of(GHOSTS, run)}
    need(ghosts.get("placed") == 4, "expected 4 ghosts, got %s" % ghosts.get("placed"))
    need(ghosts.get("revived") == 4, "expected 4 revives, got %s" % ghosts.get("revived"))

    m = first(BOTS, run)
    need(m is not None and int(m.group(1)) == 4,
         "construction robots built %s parts, expected 4" % (m and m.group(1)))

    # ------------------------------------------------- compiles under a paste
    # The guest batches (`fk.defer`): every event in a tick updates the registry
    # in place and queues its cluster, and one flush on the following tick drains
    # the queue. So there are two numbers here and both are assertions.
    #
    #   in_tick   compiles between paste-begin and paste-end -- MUST be 0, which
    #             is what says the work was deferred rather than merely cheap;
    #   by_flush  compiles between paste-begin and paste-flushed (tick 92) --
    #             MUST be exactly 1, one build for the one cluster the paste
    #             made, however many entities arrived.
    #
    # Before batching this was 2, one per PART. The parts are still registered
    # inside the paste tick, and `[BBB] state` says so: that half cannot be
    # deferred, because the entity is only valid inside its own event.
    in_tick, by_flush, registered = 0, 0, 0
    where = None
    for raw in run:
        if "[BBB-M3] paste-begin" in raw:
            where = "tick"
        elif "[BBB-M3] paste-end" in raw:
            where = "flush"
        elif "[BBB-M3] paste-flushed" in raw:
            where = None
        elif where and COMPILED.search(raw):
            by_flush += 1
            if where == "tick":
                in_tick += 1
        elif where == "tick" and STATE.search(raw):
            registered += 1
    m = first(PASTED, run)
    need(m is not None and int(m.group(1)) == int(m.group(2)),
         "the paste placed %s of %s entities" % (m and m.group(1), m and m.group(2)))
    need(registered == 4,
         "the paste registered %d parts inside the tick it happened in, expected "
         "4; the registry update is the half that cannot be deferred" % registered)
    need(in_tick == 0,
         "pasting a 2-part balancer built %d networks INSIDE the paste tick; the "
         "compile is supposed to be deferred to the next tick" % in_tick)
    need(by_flush == 1,
         "the deferred flush after the paste built %d networks; the paste makes "
         "exactly one cluster and must cost exactly one build" % by_flush)
    if m:
        print("  blueprint paste: %s entities in one tick -> 4 parts registered, "
              "0 builds in the tick, %d build by the flush on the next"
              % (m.group(2), by_flush))

    # ------------------------------------------------------------------ clones
    clones = {m.group(1): (int(m.group(2)), int(m.group(3))) for m in all_of(CLONE, run)}
    for kind in ("area", "brush"):
        got = clones.get(kind)
        need(got is not None, "no clone-%s result" % kind)
        if got:
            need(got[0] == 4, "clone-%s put %d parts on the destination, expected 4"
                 % (kind, got[0]))
            need(got[1] == 0, "clone-%s leaked %d hidden network entities onto the "
                 "destination surface" % (kind, got[1]))
    need(len(all_of(RECONCILED, run)) == 2,
         "the guest did not reconcile both cloned regions")
    kills = len(all_of(CLONE_KILL, run))
    need(kills >= 4,
         "the guest destroyed only %d cloned interfaces; an area clone copies "
         "the visible linked belts along with the parts" % kills)
    print("  clones: area and brush both reconciled, %d copied interfaces "
          "destroyed, 0 hidden entities cloned" % kills)

    # ---------------------------------------------------------------- surfaces
    m = first(DROPPED, run)
    need(m is not None and int(m.group(2)) == 4 and int(m.group(3)) == 1,
         "deleting a surface did not unregister its 4 parts in 1 cluster (got %r)"
         % (m.groups() if m else None,))
    need(first(HIDDEN_ALERT, run) is not None,
         "deleting the HIDDEN surface did not produce a loud alert")
    m = first(HIDDEN_BACK, run)
    need(m is not None and int(m.group(1)) >= 15,
         "the hidden surface was not recreated with every cluster rebuilt (got %s)"
         % (m and m.group(1)))
    if m:
        print("  surfaces: a balancer's surface deleted cleanly; the hidden surface "
              "deleted, recreated and %s clusters rebuilt" % m.group(1))

    # ----------------------------------------------- the hidden surface HIDES
    # Surface visibility is a PER-FORCE flag whose default is VISIBLE, and an
    # unhidden bbb-hidden sits in Space Age's remote-view surface list (issue
    # #1, the mod's first field report). The observer reads
    # `LuaForce::get_surface_hidden` back for every force at three moments, and
    # every line must say hidden=true. Three tags, three of the mod's paths:
    #
    #   initial    the surface CREATED, inside on_init's audit flush, with all
    #              four forces already standing -- hide-at-creation;
    #   recreated  the surface built AGAIN after its deletion at tick 450 --
    #              the lazy-recreation path must hide too;
    #   late       a force created at tick 500, AFTER the surface existed, so
    #              no hide-at-creation can have covered it. That one is the
    #              mod's on_force_created handler or nothing.
    vis = {}
    for m in all_of(SURFHIDDEN, run):
        vis.setdefault(m.group(1), {})[m.group(2)] = m.group(3) == "true"
    for tag, must in (("initial", "bbb-other"), ("recreated", "bbb-other"),
                      ("late", "bbb-late")):
        forces = vis.get(tag, {})
        need(must in forces,
             "the %s surface-hidden probe never reported force %s -- the probe "
             "did not run, or the force was never created" % (tag, must))
        exposed = sorted(n for n, h in forces.items() if not h)
        need(not exposed,
             "the hidden surface is VISIBLE to force(s) %s at tag=%s -- it would "
             "appear in their remote-view surface list" % (", ".join(exposed), tag))
    if all(vis.get(t) for t in ("initial", "recreated", "late")):
        print("  the hidden surface is withheld from every force's surface lists: "
              "%d/%d/%d forces at creation / after recreation / with a force "
              "created late" % (len(vis["initial"]), len(vis["recreated"]),
                                len(vis["late"])))

    # ------------------------------------------------------------------ audits
    # Four now, not two: two are the audits under test (the re-validation after a
    # silent rotation, and the final one), and two are there to force the guest's
    # deferred queue to drain inside the tick that a measurement is taken in --
    # `phase_part_died` and `phase_stress_end`. Only the first and the last are
    # judged.
    audits = all_of(AUDIT, run)
    need(len(audits) == 4, "expected 4 audits, got %d" % len(audits))
    if len(audits) >= 2:
        audits = [audits[0], audits[-1]]
        need(int(audits[0].group(4)) >= 1,
             "the first audit found no drift, but a belt had just been turned "
             "around with no event at all -- re-validation is not seeing the world")
        final = tuple(int(g) for g in audits[1].groups())
        need(final[3] == 0,
             "the final audit found %d clusters whose stored fingerprint does not "
             "match a from-scratch classification of the world" % final[3])
        need(final[4] == 0,
             "the final audit found %d clusters with belts on both sides and no "
             "network" % final[4])
        # THE WORLD, against a constant rather than against the guest's own idea
        # of it. `unbuilt=0` alone is weak evidence: a cluster with inputs and no
        # outputs is a legitimate half-built state and is never counted, so a rig
        # that quietly lost its network satisfies it. Every cluster in this save
        # has an input and an output, so `nets` must equal `clusters` and both
        # must be the number the rigs build.
        need(final[:3] == FINAL_WORLD,
             "the final audit found a world of (clusters=%d, parts=%d, nets=%d) "
             "and the rigs leave (%d, %d, %d): %s"
             % (final[:3] + FINAL_WORLD +
                ("a network short of a cluster is a rig that stopped balancing "
                 "and said nothing"
                 if final[2] != final[0] else
                 "the churn did not put the world back as it found it",)))
        need(final[5] == 0,
             "the final audit reports refused=%d: a cluster in this suite asked "
             "for more than the mod can build" % final[5])
        print("  audit: drift=%s after a silent rotation; drift=0 unbuilt=0 "
              "refused=0 after 600 ticks of churn, over a world of %d clusters, "
              "%d parts and %d networks"
              % ((audits[0].group(4),) + final[:3]))

    # AND NOT ONE ONE-BELT-PER-PART REFUSAL, ANYWHERE. This is a decision about
    # the churn as much as an assertion about the rigs: the six randomised edits
    # are aimed at a row's own single input, its own single output and an
    # EDGELESS part below the column, so no tile in this save can ever be asked
    # for a second belt. A refusal here would make the compile, build and
    # teardown counters above a function of the RULE rather than of the lifecycle
    # path under test, and would leave a cluster standing refused at the final
    # audit. The rule's three trigger shapes are the `sedge` suite's subject.
    sedge = all_of(SEDGEREFUSED, run) + all_of(SEDGEREFUSED, create)
    need(not sedge,
         "%d cluster(s) were refused for the one-belt-per-part rule: %r. Every "
         "rig and every churn edit in this suite is laid so that no tile ever "
         "carries two belts"
         % (len(sedge), [m.group(0).strip()[:70] for m in sedge[:3]]))
    print("  one-belt-per-part refusals over the whole run: 0, as every rig and "
          "every churn edit is laid to avoid one")

    # ------------------------------------------------------------------ forces
    # The last registry state of the build phase. Two forces built against each
    # other: had they merged there would be a second cluster of 4.
    states = all_of(STATE, create)
    need(bool(states), "no registry state was ever logged")
    if states:
        sizes = sorted(int(v) for v in states[-1].group(3).split(","))
        need(sizes == CREATE_SIZES,
             "cluster sizes at the end of --create are %s, expected %s. Two "
             "forces' parts touching must not merge -- a fusion reads as one "
             "fewer four and one more eight -- and nothing else may have gained "
             "or lost a part either" % (sizes, CREATE_SIZES))
        print("  forces: %d clusters, sizes %s -- the two forces did not merge"
              % (len(sizes), sizes))

    # ------------------------------------------------------------------- items
    m = first(SWAPPED, run)
    need(m is not None and m.group(1) == "fast-transport-belt",
         "the fast-replace left %s" % (m and m.group(1)))

    # Killing ONE part of a two-part balancer leaves a cluster standing, so this
    # is a RECOMPILE: the drained items go back inside the network the flush
    # rebuilds rather than onto the floor. The count therefore covers both
    # surfaces and what is asserted is conservation across the pair -- a
    # visible-only count read the correct behaviour as a two-item loss, which is
    # how this assertion was found. What is still allowed to go is the fractional
    # item positions and whatever a splitter holds outside its transport lines.
    m = first(DIED, run)
    need(m is not None, "the died phase reported no item counts")
    if m:
        was, now = int(m.group(1)), int(m.group(2))
        need(now <= was, "killing a part MINTED items (%d -> %d)" % (was, now))
        need(was - now <= 0.02 * was,
             "killing a part with a full network lost %d of %d items (%.2f%%), "
             "over the 2%% bound" % (was - now, was, 100.0 * (was - now) / was))
        print("  a part killed with die() conserved %d of %d items across the "
              "recompile, both surfaces counted" % (now, was))

    ins, rec = first(INSERTED, run), first(RECOVERED, run)
    need(ins is not None and rec is not None, "the stress run reported no item counts")
    if ins and rec:
        put, back = int(ins.group(1)), int(rec.group(1))
        lost = put - back
        need(back <= put,
             "the stress run ended with MORE items than it started with (%d -> %d): "
             "a teardown is minting matter" % (put, back))
        # The documented spill bound: a teardown recovers every item its
        # transport lines hold and cannot recover fractional positions or
        # whatever a splitter holds outside them. Over ~100 teardowns that is
        # about 1%.
        need(lost <= put // 50,
             "the stress run lost %d of %d items (%.1f%%), over the 2%% bound"
             % (lost, put, 100.0 * lost / put))
        print("  stress: %d items in, %d recovered after a full teardown (%.2f%% "
              "lost to fractional positions and splitter buffers)"
              % (put, back, 100.0 * lost / put))

    # The beneficiary must never fire here. This suite drives every removal path
    # there is EXCEPT a player mining -- die(), destroy() with and without an
    # event, a fast-replace, a surface deleted, the hidden surface deleted, a
    # clone reconcile, ~100 randomised stress teardowns -- and none of them has
    # anybody to credit. A pocket line in this log would mean carry.go's claim
    # list is leaking across removal paths it does not belong to.
    pocket = [l for l in create + run if "pocketed" in l and "[BBB]" in l]
    need(not pocket,
         "%d teardowns credited a player in a suite where nothing is player-mined"
         % len(pocket))
    print("  the mined-by-player beneficiary fired 0 times, as it must: no "
          "removal in this suite is a player mining")

    # ------------------------------------------------------------ M5: the arrows
    # One rendering object per visible interface, always -- no more, no fewer.
    #
    # The guest stores no rendering ids at all: the arrows are drawn ON the
    # interface entities and the engine destroys a rendering object whose target
    # entity is destroyed. This suite is the place that claim is worth checking,
    # because by t=1500 it has torn ~100 networks down, deleted a surface with
    # balancers on it, deleted the HIDDEN surface under every network at once,
    # cloned interfaces and killed some from outside. A leak on any of those
    # paths shows up as renders > interfaces, and a teardown that took an arrow
    # off a network still standing shows up as renders < interfaces.
    m = first(RENDERS, run)
    need(m is not None, "the suite never counted the I/O arrows")
    if m:
        renders, ifaces = int(m.group(1)), int(m.group(2))
        need(ifaces > 0, "there were no visible interfaces left to carry arrows")
        need(renders == ifaces,
             "%d rendering objects against %d visible interfaces: the I/O arrows "
             "are leaking or being lost" % (renders, ifaces))
        print("  I/O arrows: %d rendering objects, %d visible interfaces, after "
              "~100 teardowns and two surface deletions" % (renders, ifaces))

    # -------------------------------------------------------------- throughput
    samples = {}
    for m in all_of(SAMPLE, run):
        samples.setdefault(m.group(2), {})[int(m.group(1))] = [
            int(v) for v in m.group(3).split()
        ]
    need("ctrl" in samples, "the control belt was never sampled")
    if "ctrl" in samples and T0 in samples["ctrl"] and T1 in samples["ctrl"]:
        belt = samples["ctrl"][T1][0] - samples["ctrl"][T0][0]
        need(belt > 500, "the control belt only moved %d items; the rigs cannot be "
                         "judged against it" % belt)
        print("\n  one saturated express belt delivered %d items between t=%d and "
              "t=%d\n" % (belt, T0, T1))
        print("  %-8s %-4s %-26s %8s %9s %7s"
              % ("rig", "outs", "delivered", "total", "expected", "spread"))
        for name, live, nominal, under, over, spread_max in EXPECT:
            s = samples.get(name)
            if not s or T0 not in s or T1 not in s:
                fail.append("rig %s was never sampled" % name)
                continue
            got = [b - a for a, b in zip(s[T0], s[T1])][:live]
            total = sum(got) / float(belt)
            mean = sum(got) / float(len(got))
            spread = (max(got) - min(got)) / mean if mean else 0.0
            print("  %-8s %-4d %-26s %7.3fx %8.3fx %6.2f%%"
                  % (name, live, " ".join(str(v) for v in got), total, nominal,
                     100 * spread))
            if total < nominal * (1 - under) or total > nominal * (1 + over):
                fail.append("%s delivered %.3f belts, expected %.3f"
                            % (name, total, nominal))
            if len(got) > 1 and spread > spread_max:
                fail.append("%s outputs spread %.2f%%, over %.2f%%"
                            % (name, 100 * spread, 100 * spread_max))
        # The orphaned row of `died` must be EXACTLY frozen: its belts no longer
        # touch a part, so nothing may reach them at all.
        s = samples.get("died")
        if s and T0 in s and T1 in s and len(s[T0]) > 1:
            if s[T1][1] != s[T0][1]:
                fail.append("the row orphaned by the part that died still received "
                            "%d items" % (s[T1][1] - s[T0][1]))
            else:
                print("\n  the row orphaned by the dead part received exactly 0 "
                      "further items")

    if fail:
        print("\nM3 LIFECYCLE ASSERTIONS FAILED:")
        for f in fail:
            print("  " + f)
        sys.exit(1)
    print("\nM3 lifecycle assertions passed (%d rigs)" % len(EXPECT))


if __name__ == "__main__":
    main()
