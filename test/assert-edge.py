#!/usr/bin/env python3
"""Assert what happens when an edit lands on a network that is FULL AND MOVING.

Two kinds of evidence, the same split the M3 suite uses:

  * the GUEST's own [BBB] lines -- what it decided, what the audit found, what
    the forces handler remapped, and in what ORDER the deferred flush compiled
    a scrambled paste;
  * [BBB-EDGE] lines for the one thing it cannot see -- how many items exist.
    Every source is a finite chest and the count covers the visible surface AND
    the hidden one, so the total is a conserved quantity and any fall in it is a
    real loss.

AND, since 2026-08-02, WHERE those items are. Conservation was never the defect;
placement was. A recompile used to drain the hidden network and spill it beside
the cluster, so every edge edit on an operating balancer rained items onto the
floor. `ground=` is on every count line and a recompile must leave it at zero:
the drained items belong INSIDE the network that was just built.

AND, since the tan-streak pass, WHERE the compiler's own entities are. The only
thing it may put on a surface a player looks at is an edge interface, and an
interface stands on a tile of the cluster itself -- under a balancer part's
sprite, which is one opaque tile. `[BBB-EDGE] place` samples that invariant over
every entity of our four hidden prototypes on the visible surface, including
around the field report's own shape: a 2x2 balancer with one corner missing,
saturated and flowing for the whole run.

AND, since 2026-08-04, WHAT HAPPENS TO A BALANCER THAT IS ALREADY AS BIG AS THIS
MOD BUILDS. `lim` is sixty-four inputs and one output -- P = plan.MaxPorts
exactly -- and a sixty-fifth belt is laid on it while it runs. The refusal is
correct and was always correct; what it used to do first was tear the working
network down. The leg asserts that it does not any more, and the numbers the
unfixed guest produced are in the LIM section so that "it was demolished" is a
measurement rather than a story.

AND, since 2026-08-05, THE OTHER SHAPE OF THAT REFUSAL. `brdg` is two WORKING
balancers with a one-tile gap and a part in the gap. A merge's teardowns are
queued by `AddPart`, not by compile(), so the check that saved `lim` could not
reach it: both halves were demolished before anything discovered that what they
became could not be built. The BRDG section carries the unfixed guest's numbers
for the same reason.

Every rig is built to Factorio 2.1's rule, one belt per balancer part, and three
of them are a redesign rather than a re-lay because the EDIT they take is what
the rule changed. `aout` and `ain` carry a spare row of parts holding nothing,
and `bmin` an attached edgeless fifth part, because a belt laid on a working
balancer has no free face to land on any more; `lim` is sixty-four belts over
sixty-six parts with a spare part of its own; `brdg`'s gap tile is flanked by ONE
belt rather than two, so the merge reaches the PORT limit rather than the
one-belt bound; and `frepa`'s belt line ENDS on the tile the part is dropped
onto. THE ONE-BELT BOUND ITSELF IS ASSERTED HERE ONLY AS A NEGATIVE -- not one
refusal for it over the whole run -- because every edit in this suite is meant to
be one the mod can honour, and the three ways of reaching that refusal are the
`sedge` suite's subject.

    python3 test/assert-edge.py create.log run.log
"""

import re
import sys

MARK = re.compile(r"\[BBB-EDGE\] mark tag=(\S+)")
COUNT = re.compile(r"\[BBB-EDGE\] count tag=(\S+) total=(\d+) ground=(\d+)")
PLAN = re.compile(r"\[BBB-EDGE\] plan chn_end=(\d+) end_tick=(\d+) stock=(\d+)")
DETB = re.compile(r"\[BBB-EDGE\] det-(begin|end|flushed) tag=(\S+)")
RIG = re.compile(r"\[BBB-EDGE\] t=(\S+) rig=(\w+) out=\[([-\d ]*)\]")
DONE = re.compile(r"\[BBB-EDGE\] done")

AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) unbuilt=(\d+) "
    r"refused=(\d+)"
)
COMPILED = re.compile(r"\[BBB\] compiled cluster (\d+) (\d+)->(\d+)")
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
TOOKBACK = re.compile(r"\[BBB\] cluster (\d+) took back (\d+) items")
SPILLED = re.compile(r"\[BBB\] spilled (\d+) items beside cluster (\d+)")
REFUSED = re.compile(r"\[BBB\] refused a part on the hidden surface")
MERGED = re.compile(
    r"\[BBB\] forces merged: (\d+) -> (\d+), (\d+) parts remapped, (\d+) clusters"
)
PLACE = re.compile(
    r"\[BBB-EDGE\] place tag=(\S+) ours=(\d+) onpart=(\d+) offpart=(\d+) parts=(\d+)"
)
STRAY = re.compile(r"\[BBB-EDGE\] stray tag=(\S+) (\S+)")
POCKETED = re.compile(r"\[BBB\] cluster (\d+) was mined by player (\d+); pocketed (\d+)")
MINERAISE = re.compile(r"\[BBB-EDGE\] player-mine-raise ok=(\S+) err=(.*)")
MINERESOLVE = re.compile(r"\[BBB-EDGE\] player-resolve p1=(\S+) players=(\d+)")
CMDREG = re.compile(r"\[BBB-EDGE\] command registered=(\S+)")
IFACE = re.compile(r"\[BBB-EDGE\] remote iface methods=(\S*)")
RCALL = re.compile(r"\[BBB-EDGE\] remote audit ok=(\S+) clusters=(\S+)")
PROBE = re.compile(
    r"\[BBB\] insert probe (\S+) (\S+) asked=(\d+) took=(\d+) held=(\d+)"
)
PROBELUA = re.compile(r"\[BBB-EDGE\] insert-probe-lua (\S+) want=(\d+) held=(\d+)")
LIM = re.compile(r"\[BBB-EDGE\] lim tag=(\S+) tick=(\d+) delivered=(\d+)")
BRDG = re.compile(r"\[BBB-EDGE\] brdg tag=(\S+) tick=(\d+) a=(\d+) b=(\d+)")
# The direct evidence that the teardowns were taken off the queue rather than
# merely regretted afterwards. Written by limit.go's spareMerge, at the top of
# flushDead, before anything has been touched.
# The line NAMES NO BOUND: there are two of them, and which one a merge would
# have broken is on the `alert:` the refusal itself writes a moment later.
SPARED = re.compile(
    r"\[BBB\] cluster (\d+) would merge into a cluster this mod cannot build; "
    r"left (\d+) standing network\(s\) alone"
)
# The one-belt-per-part refusal, which must never fire here. See SEDGE below.
SEDGEREFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt")
OVERLIMIT = re.compile(
    r"\[BBB\] alert: cluster (\d+) would need (\d+) ports for (\d+) inputs and "
    r"(\d+) outputs, over the limit of (\d+)"
)
# The two halves of the hand-back, which cannot fire in any headless suite: a
# --create has no players, so `game.get_player` resolves to nothing and
# `revertOne` returns before it mines anything. Asserted at zero for the same
# reason the pocketed lines are -- a revert leaking onto a path that has no
# player behind it would be the feature firing for a robot.
# EITHER ARM OF THE HAND-BACK, matched on the shape both of them share rather
# than on one sentence. This is a NEGATIVE assertion -- a headless --create has
# no players, so `revertOne` returns before it mines anything -- and an exact
# regex over a negative is the one shape a rename in the guest can make
# VACUOUS: the line stops matching, the assertion stops being able to fail, and
# nothing says so. "piece at x,y" is what `handed the refused piece at 4,7 (over
# the port limit) back to player 1` and its could-not-be-handed-back twin have
# in common, and nothing else in this guest's vocabulary produces it.
HANDEDBACK = re.compile(r"\[BBB\].*\bpiece at -?\d+,-?\d+")
# The other side of the same fork: nobody built it, so the force is told instead.
# This IS reachable headlessly -- every build in every suite is a script build --
# and it is the only part of the feedback that is.
TOLDFORCE = re.compile(
    r"\[BBB\] told force (\d+) that cluster (\d+) is over the port limit"
    r"(.*)$"
)
HANDBACKFAIL = re.compile(r"\[BBB\] alert: player (\d+) could not be handed back")
# FAST REPLACE. The `frep-can` lines are the ENGINE's own answer to the question
# a player's cursor asks; the rest is what actually happened to the world.
FREPCAN = re.compile(r"\[BBB-EDGE\] frep-can what=(\S+) value=(\S+)")
FREPFWD = re.compile(
    r"\[BBB-EDGE\] frep-fwd created=(\S+) belt-left=(\S+) part-there=(\S+)")
FREPEDGE = re.compile(
    r"\[BBB-EDGE\] frep-edge created=(\S+) part-survived=(\S+)")
FREPREV = re.compile(
    r"\[BBB-EDGE\] frep-rev created=(\S+) part-left=(\S+) belt-there=(\S+)")
FREPSPILL = re.compile(
    r"\[BBB-EDGE\] frep-spill tag=(\S+) handed-back=\[([^\]]*)\] "
    r"machine-removed=(\d+) where=(.*)$")
# The guest's own statement that it noticed. It can only be written when the
# part was ALREADY GONE by the time the belt's build event arrived, so its
# presence is also the measurement that the engine raised no removal event for
# the part it destroyed.
REAPED = re.compile(
    r"\[BBB\] a belt-connectable fast-replaced the part at (-?\d+),(-?\d+)")

# What the insert probe offers a steel chest, in the order probe.go offers it.
# The counts are distinct, none of them 1, and none a multiple of another: a
# count arriving as 1 (an ItemStackDefinition whose `count` never reached the
# engine), as some other leg's number, or as a stack size is a different wrong
# answer and the failure says which.
PROBE_WANT = [
    ("iron-gear-wheel", 50), ("iron-plate", 37),
    ("copper-cable", 23), ("steel-chest", 7),
]

# What taking a balancer apart BY HAND must put somewhere other than back in the
# machine. Nine shrinks into ever-smaller networks, so the reinsertion runs out
# of room and the overflow falls through -- to the miner where there is one, and
# to the floor here, because a headless run has no player. The bound is a FLOOR
# rather than a ceiling: it is the assertion that the quantity the miner's pocket
# redirects is real, on a leg that would otherwise pass vacuously.
HAND_OVERFLOW_MIN = 10

# The placement probe's samples. Every one is taken in a tick where the world is
# settled, and `flowing` is taken while every rig in the save is saturated --
# which is the condition the tan-streak field report was made under.
PLACE_TAGS = ("init", "post-merge", "post-add-out", "flowing", "final", "brdg",
              "frep")

# A floor, so that a probe which found nothing cannot pass. Eleven clusters with
# two or more edges each is well over this; the number exists only to make an
# empty sample a failure instead of a silent success.
PLACE_MIN = 20

# THE SAVE HAS FIFTEEN CLUSTERS AND ONE HUNDRED AND NINETY-EIGHT PARTS, and every
# one of the counts below is derived from the rigs rather than copied from a
# passing run. Under the one-belt rule a row of a balancer is TWO parts -- a west
# one carrying the row's input and an east one carrying its output -- so a
# 2-in/2-out rig is four parts and a 4x4 is eight:
#
#   chn   4   a dead-ended 2->2 that the churn leg grows and shrinks
#   same  4   the same-tick and pending-flush rig
#   mrg   4+4 two of them with a one-tile gap, bridged and un-bridged
#   rot   4   the silently rotated edge
#   frcA  4   two forces' balancers, touching
#   frcB  4
#   aout 10   a saturated 4x4 (eight parts) PLUS a fifth row carrying nothing
#   ain  10   the same
#   shrk  8   a saturated 4x4; its edit takes a belt away, so it needs no spare
#   ntch  5   a C of parts around a hole, two in and two out
#   bmin  5   a dead-ended 2->2 plus an attached EDGELESS fifth part
#   lim  66   a 2x32 block carrying 64 input belts, an output part and a spare
#   brdg 33+33  two of the same shape at half the size, one tile apart
#
# THE SPARE PARTS ARE THE RULE SHOWING THROUGH and are not padding. `aout`,
# `ain`, `bmin` and `lim` all exist to take a belt ON A WORKING BALANCER, and
# under the rule a working balancer has no free face: every part that carries a
# belt has its one belt. An attached part carrying nothing is the only place a
# player's belt can still change a machine's port count, which is the same
# conclusion m2's conservation rig and the interactive checklist's band B reached
# independently.
#
# brdg, ntch, bmin and lim are never edited in a way that moves a part, so like
# the recompile-under-load rigs they widen every baseline below by a constant and
# move nothing else.
#
# tag -> (clusters, parts, nets, drift, unbuilt)
#
# `nets` IS ASSERTED AND `unbuilt` IS NOT ENOUGH ON ITS OWN. A cluster with
# inputs and no outputs is a legitimate half-built state and is never counted
# unbuilt, so `unbuilt=0` is satisfied by a save in which a rig quietly lost its
# network -- which is exactly what a mis-classified edge or an unexpected refusal
# leaves behind. Every cluster here has both an input and an output except where
# a leg has deliberately taken one away, and those are the two rows below that
# say so.
BASELINE = (15, 198)
EXPECT = {
    "pre-sametick":            (15, 198, 15, 0, 0),
    # A part placed and removed inside one dispatch chain leaves nothing behind:
    # the node is allocated and freed, and the flush on the next tick is handed
    # a root whose slot is on the free list and must DROP it rather than compile
    # a cluster that no longer exists.
    "post-sametick":           (15, 198, 15, 0, 0),
    # A part placed the tick before and another placed on the tick its deferred
    # flush lands. Both are EDGELESS parts hanging off `same`'s west column, so
    # the cluster gains two parts and no edges at all.
    "post-pending":            (15, 200, 15, 0, 0),
    "post-pending-undo":       (15, 198, 15, 0, 0),
    "pre-merge":               (15, 198, 15, 0, 0),
    # The bridge. Two SATURATED networks come down in one flush and one comes
    # up: fifteen clusters become fourteen and the mrg pair becomes one of nine
    # parts.
    "post-merge":              (14, 199, 14, 0, 0),
    # And the undo of that merge: the split path, still under load.
    "post-split":              (15, 198, 15, 0, 0),
    "pre-rot":                 (15, 198, 15, 0, 0),
    # `entity.direction = ...` raises nothing at all, so the audit is the only
    # thing that can find it -- and it must REPORT the drift before repairing
    # it, which is what makes this an assertion rather than a tautology.
    "post-rot-silent":         (15, 198, 15, 1, 0),
    "post-rot-restored":       (15, 198, 15, 1, 0),
    # The same edge through the event path: the recompile already happened, so
    # there is nothing left for the audit to find.
    "post-rot-event":          (15, 198, 15, 0, 0),
    "post-rot-event-back":     (15, 198, 15, 0, 0),
    "pre-forces":              (15, 198, 15, 0, 0),
    # Two forces edited in one tick, alternating: A WHOLE ROW EACH, because a row
    # is two parts and one part cannot take both the new input and the new
    # output. Their parts touch and they must STILL be two clusters, so the
    # cluster count does not move at all and the part count goes up by four.
    "post-forces-interleaved": (15, 202, 15, 0, 0),
    # And then they are one force. The two touching clusters become one, and
    # nothing drifted.
    "post-forces-merge":       (14, 202, 14, 0, 0),
    # A part built on the HIDDEN surface. It must not become a cluster.
    "post-hidden-part":        (14, 202, 14, 0, 0),
    # The three recompile-under-load rigs, in order. Every one of these is an
    # edge edit on a saturated 4x4 and none of them moves a part or a cluster --
    # the belt each of them adds lands on the SPARE ROW those rigs carry.
    "pre-add-out":             (14, 202, 14, 0, 0),
    "post-add-out":            (14, 202, 14, 0, 0),
    # The port-boundary rig's GROWING half: a third output belt laid against the
    # attached edgeless part, which takes P from 2 to 4. Same shape as add-out
    # and the same requirement -- nothing on the ground.
    "pre-bmin":                (14, 202, 14, 0, 0),
    "post-bmin-add":           (14, 202, 14, 0, 0),
    "pre-add-in":              (14, 202, 14, 0, 0),
    "post-add-in":             (14, 202, 14, 0, 0),
    "pre-shrink":              (14, 202, 14, 0, 0),
    "post-shrink":             (14, 202, 14, 0, 0),
    # And then every part of the shrk rig is mined -- eight of them now. The
    # cluster DISSOLVES, which is a removal and not a recompile, and the items
    # come back to the world rather than into a network that no longer exists.
    "post-remove":             (13, 194, 13, 0, 0),
    # The port-boundary rig's SHRINKING half: that same third output belt mined
    # again, P back from 4 to 2. Still an edge edit, so the cluster and part
    # counts do not move -- what moves is the ground.
    "pre-bmin-remove":         (13, 194, 13, 0, 0),
    "post-bmin-remove":        (13, 194, 13, 0, 0),
    # The `aout` rig taken apart ONE PART PER TICK, which is what a player does
    # by hand: TEN parts, so ten steps. The spare row goes first and then the
    # block row by row, west part then east part, so every prefix leaves a
    # CONNECTED cluster and eight of the nine shrinks leave a machine with at
    # least one input and one output.
    "pre-hand":                (13, 194, 13, 0, 0),
    "hand-1":                  (13, 193, 13, 0, 0),
    "hand-2":                  (13, 192, 13, 0, 0),
    "hand-3":                  (13, 191, 13, 0, 0),
    "hand-4":                  (13, 190, 13, 0, 0),
    "hand-5":                  (13, 189, 13, 0, 0),
    "hand-6":                  (13, 188, 13, 0, 0),
    "hand-7":                  (13, 187, 13, 0, 0),
    "hand-8":                  (13, 186, 13, 0, 0),
    # THE NINTH STEP LEAVES ONE PART, AND ONE PART CARRIES ONE BELT. So the last
    # survivor has an input or an output and never both, which `plan.Build`
    # reads as a legitimate half-built cluster: no network, and NOT `unbuilt`.
    # It is the one place in this suite where `nets` is legitimately short of
    # `clusters`, and it is a consequence of the rule rather than of the leg.
    "hand-9":                  (13, 185, 12, 0, 0),
    "hand-10":                 (12, 184, 12, 0, 0),
    "final":                   (12, 184, 12, 0, 0),
    # THE PORT LIMIT. `lim` is sixty-four inputs and one output, which is
    # P = plan.MaxPorts exactly, and it has been running untouched since t=0.
    "pre-lim":                 (12, 184, 12, 0, 0),
    # ... and then a sixty-fifth input belt, against the SPARE PART -- the only
    # tile of that rig with a free face, and the only way to reach the port bound
    # rather than the one-belt one. P would have to be 128 and the compile is
    # REFUSED, before the teardown, so the network is still there and `nets`
    # still holds it. The audit therefore reports a cluster WITH a network
    # (unbuilt stays 0) whose stored fingerprint no longer describes the world
    # (drift becomes 1), which is the exact truth.
    #
    # A `drift=0 unbuilt=1` here would mean the network came down -- the defect.
    "post-lim":                (12, 184, 12, 1, 0),
    "post-lim-window":         (12, 184, 12, 1, 0),
    # The belt mined off again. The edge list is back to sixty-four, which is
    # the fingerprint the netInfo already holds, so the compile is a SKIP and
    # the drift is gone without anything having been rebuilt.
    "post-lim-back":           (12, 184, 12, 0, 0),
    # THE MERGE THAT WOULD BE OVER THE LIMIT. Two working thirty-three-part
    # balancers with a one-tile gap; ONE input belt stands beside the gap, so a
    # part in it makes one cluster of 65 inputs and 2 outputs and P would have to
    # be 128. A second flanking belt would put two belts on the bridging part
    # itself and the merge would be refused for the OTHER bound.
    "pre-brdg":                (12, 184, 12, 0, 0),
    # Twelve clusters become eleven and the two halves become one cluster of
    # sixty-seven parts -- the registry took the part, as it takes every part.
    # What must NOT have happened is the teardown: BOTH networks are still
    # standing, keyed by roots that are not roots any more, so the audit counts
    # them -- `nets` stays at twelve while `clusters` falls to eleven -- and
    # reports the cluster they now belong to as drifted.
    #
    # `drift=0 unbuilt=1` here is the defect: it means flushDead demolished two
    # working balancers before flushLive found out that what they became could
    # not be built. That is exactly what the unfixed guest reports, measured.
    "post-brdg":               (11, 185, 12, 1, 0),
    # ... and it STAYS that way. No second refusal, nothing torn down, nothing
    # rebuilt, and the same report every time it is asked.
    "brdg-hold-1":             (11, 185, 12, 1, 0),
    "brdg-hold-2":             (11, 185, 12, 1, 0),
    "post-brdg-window":        (11, 185, 12, 1, 0),
    # The bridging part mined off. The cluster splits back into the two it was,
    # each re-roots at its smallest node id -- which is the root it already had
    # -- and each half's fingerprint is the one its netInfo never lost, so both
    # compiles are a SKIP.
    "post-brdg-back":          (12, 184, 12, 0, 0),
    "post-brdg-final":         (12, 184, 12, 0, 0),
    # FAST REPLACE. Its two rigs are the only ones in this suite that are built
    # MID-RUN, and that is why every number above this line is a statement about
    # the same world it has always been about. Building them adds two clusters
    # and eleven parts -- frepa's four and frepb's seven.
    "frep-built":              (14, 195, 14, 0, 0),
    "pre-frep":                (14, 195, 14, 0, 0),
    # A PART fast-replaced onto the LAST TILE of the belt line running past
    # frepa. The belt is gone, the part joined the cluster, and the balancer that
    # was two in and two out is three in and two out. The line ends there because
    # a part dropped MID-line would take the belt behind it as an input and the
    # belt ahead as an output, which is two belts on one tile.
    "post-frep-fwd":           (14, 196, 14, 0, 0),
    # A belt refused over an EDGE part -- one carrying an interface. Nothing
    # moves: `can_fast_replace` is false there and the rig puts back what
    # `create_entity` mined behind the engine's own check.
    "post-frep-edge":          (14, 196, 14, 0, 0),
    # And a belt laid on the middle of frepb's NECK, which is the half nothing
    # tells the guest about. The column SPLITS into two three-part clusters, and
    # both of them are buildable because the target's two vertical neighbours
    # carry nothing: the new belt is the upper half's second output and the lower
    # half's second input.
    #
    # `(14, 196)` here is the defect, and it is what the guest without
    # guest/go/fastreplace.go reports -- a tile it calls a balancer part which is
    # holding somebody's belt, for the rest of the session.
    "post-frep-rev":           (15, 195, 15, 0, 0),
    "frep-final":              (15, 195, 15, 0, 0),
}

# THE TAGS AT WHICH A REFUSED MERGE IS STANDING, and what the audit's `nets=`
# must read there. It is the assertion that limit.go's stranded networks are
# COUNTED: they are keyed by roots `liveRootList` can never return, so without
# `strandedNets` the audit would quietly report two fewer networks than the save
# contains -- and, when the merged cluster's root is a node that never had a
# network of its own, would call a cluster whose two balancers are running
# perfectly well `unbuilt`.
BRDG_REFUSED = ("post-brdg", "brdg-hold-1", "brdg-hold-2", "post-brdg-window")
BRDG_NETS = 12

# The tags at which a RECOMPILE has just happened and nothing may be on the
# ground because of it: the field report's three shapes, the port-boundary rig's
# growing half, plus the merge and the split of two saturated networks.
#
# `ground` is CUMULATIVE over the run, so this list can only hold tags that
# happen before the first leg that deliberately spills. The `brdg` leg runs last
# of all and is checked as a DELTA in its own section instead -- the same
# assertion, phrased the only way it can be that late in the schedule.
NO_GROUND = (
    "post-add-out", "post-add-in", "post-bmin-add", "post-merge", "post-split",
)

# The shrink that is not one. Four inputs and three outputs is a smaller network
# than four and four in every sense a player can see -- but P = next_pow2(max(N,
# M)) is 4 either way, so the butterfly `shrk` rebuilds is the SAME SIZE and
# everything fits with room to spare. That is why this bound has never been
# approached, and it is why this suite was silent while a player watched an
# output-belt removal empty a balancer onto the floor: the only shrink it drove
# did not shrink anything. The `bmin` leg below is the one that crosses the
# boundary. The bound stays because a regression that made the interior smaller
# would show here first.
SHRINK_GROUND_BOUND = 40

# THE PORT-BOUNDARY SHRINK, and this one is a FLOOR. Two outputs to three and
# back takes P from 2 to 4 and down to 2 again, so the network the recompile
# builds really is half the one it drained and the reinsertion really does run
# out of room. That overflow is what the miner's pocket redirects; headless has
# no player, so it lands on the ground and is counted here.
#
# A floor rather than a ceiling for the same reason the by-hand leg's is: a leg
# where the shrink happened to fit would satisfy every other assertion in this
# suite and would say nothing at all about the quantity. Measured at 128 items
# on 2026-08-02, on the guest before AND after the fix -- player_index is 0 on
# every removal a headless run can produce, so the redirection is invisible here
# and only the quantity is not. The floor is set well under it so that ordinary
# variation in how full the belts are does not fail the run.
BMIN_OVERFLOW_MIN = 40

# The scrambled paste places the four clusters' first parts in this order, so
# this is the order the deferred flush must compile them in -- and it must be
# the same order both times.
DET_ORDER = [3, 1, 4, 2]

# 200 teardowns of a full network in the chn leg alone, plus the one-offs. M3
# measures 0.89% lost over ~100 teardowns, to fractional item positions and
# whatever a splitter holds outside its transport lines.
LOSS_BOUND = 0.08


def read(paths):
    out = []
    for p in paths:
        with open(p, encoding="utf-8", errors="replace") as fh:
            out.extend(fh.readlines())
    return out


def fail(msg):
    print("FAIL: " + msg)
    sys.exit(1)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(2)
    lines = read(sys.argv[1:])

    plan = None
    for line in lines:
        m = PLAN.search(line)
        if m:
            plan = m
    if not plan:
        fail("the test mod never logged its plan; the run did not start")
    stock = int(plan.group(3))
    if not any(DONE.search(l) for l in lines):
        fail("the run never reached its last phase; raise BBB_EDGE_TICKS")

    # ---- pair every tagged marker with the audit it triggered --------------
    audits, counts, ground, pending = {}, {}, {}, None
    for line in lines:
        m = MARK.search(line)
        if m:
            pending = m.group(1)
            continue
        a = AUDIT.search(line)
        if a and pending:
            audits[pending] = tuple(int(g) for g in a.groups())
            continue
        c = COUNT.search(line)
        if c:
            counts[c.group(1)] = int(c.group(2))
            ground[c.group(1)] = int(c.group(3))
            pending = None

    # ---- the hundred cycles, and "no drift" is the claim -------------------
    chn = [(int(t[3:]), counts[t]) for t in counts if t.startswith("chn")]
    chn.sort()
    if len(chn) < 100:
        fail("the churn leg logged %d cycles, expected 100" % len(chn))
    first, last = chn[0][1], chn[-1][1]
    rises = [i for (i, v), (_, prev) in zip(chn[1:], chn[:-1]) if v > prev]
    drifted = [i for i, _ in chn if audits["chn%d" % i][3] != 0]
    unbuilt = [i for i, _ in chn if audits["chn%d" % i][4] != 0]
    shapes = {audits["chn%d" % i][:2] for i, _ in chn}
    print("a hundred add-part/remove-part cycles on a SATURATED balancer:")
    print("  items %d -> %d over %d cycles and %d teardowns: %d lost (%.2f%%)"
          % (first, last, len(chn), 2 * len(chn), first - last,
             100.0 * (first - last) / max(first, 1)))
    print("  the count never rose: %s" % ("yes" if not rises else "NO"))
    print("  audits reporting drift: %d;  unbuilt: %d" % (len(drifted), len(unbuilt)))
    print("  every cycle's audit saw the same world: %s" % sorted(shapes))
    # WHERE the items are, which is what the 2026-08-02 policy change is about.
    # Two hundred recompiles of a network whose outputs are DEAD-ENDED: the
    # spill has nowhere to put an item except the floor, so under the old
    # drain-and-spill policy this number climbed all run. A recompile now
    # reinserts, so it must stay at zero -- and it is cumulative, so one leaked
    # spill anywhere in the leg shows up in every cycle after it.
    chn_ground = [(i, ground["chn%d" % i]) for i, _ in chn]
    worst = max(g for _, g in chn_ground)
    print("  items on the GROUND over the leg: first %d, last %d, worst %d"
          % (chn_ground[0][1], chn_ground[-1][1], worst))
    if worst > 0:
        leaked = [i for i, g in chn_ground if g > 0]
        fail("the churn leg put %d items on the ground (first at cycle %d): a "
             "recompile is not a removal, and the drained items belong back "
             "inside the network that was just built" % (worst, leaked[0]))
    # And the networks really were carrying items. Only chn is torn down during
    # this leg, so every `returned` line in it is one of the two hundred, and a
    # leg of empty teardowns would prove nothing about conservation.
    returned = sorted(int(m.group(2))
                      for m in (TORNDOWN.search(l) for l in lines) if m)
    if len(returned) < 150:
        fail("only %d teardowns returned any items at all; the churn leg was "
             "tearing down EMPTY networks and proves nothing" % len(returned))
    print("  items handed back per teardown: min %d, median %d, max %d over %d "
          "teardowns that carried something"
          % (returned[0], returned[len(returned) // 2], returned[-1], len(returned)))
    if returned[len(returned) // 2] < 10:
        fail("the median teardown returned %d items: the networks were not full"
             % returned[len(returned) // 2])
    if rises:
        fail("the item count ROSE at cycles %r: the churn minted matter" % rises[:5])
    if drifted:
        fail("cycles %r left the guest disagreeing with the world" % drifted[:5])
    if unbuilt:
        fail("cycles %r left a cluster with inputs and outputs and no network"
             % unbuilt[:5])
    if shapes != {BASELINE}:
        fail("the churn leg did not return the world to %d clusters of %d parts "
             "every cycle: %r" % (BASELINE[0], BASELINE[1], sorted(shapes)))
    if first - last > LOSS_BOUND * first:
        fail("the churn leg lost %.2f%% of its items, over the %.0f%% bound"
             % (100.0 * (first - last) / first, 100 * LOSS_BOUND))
    print()

    # ---- the one-off edges -------------------------------------------------
    print("%-26s %9s %7s %6s %7s %9s %8s"
          % ("edge", "clusters", "parts", "nets", "drift", "items", "ground"))
    for tag, (c, p, n, d, u) in EXPECT.items():
        if tag not in audits:
            fail("the %s phase never ran" % tag)
        got = audits[tag]
        print("%-26s %9d %7d %6d %7d %9d %8d"
              % (tag, got[0], got[1], got[2], got[3], counts[tag], ground[tag]))
        if (got[0], got[1]) != (c, p):
            fail("%s: the guest saw %d clusters of %d parts, expected %d of %d"
                 % (tag, got[0], got[1], c, p))
        # `nets` IS THE HALF `unbuilt` CANNOT MAKE. A cluster with inputs and no
        # outputs is a legitimate half-built state and is never counted unbuilt,
        # so `unbuilt=0` is satisfied by a save in which a rig quietly lost its
        # network. Every count here is written down beside its reason.
        if got[2] != n:
            fail("%s: the audit counted %d standing networks over %d clusters "
                 "and the rigs leave %d. %s"
                 % (tag, got[2], got[0], n,
                    "A network short of a cluster is a rig that stopped "
                    "balancing and said nothing -- `unbuilt` cannot see it, "
                    "because a cluster with no outputs is a legitimate "
                    "half-built state" if got[2] < n else
                    "More networks than the save should hold means something "
                    "was left standing under a key nothing will reclaim"))
        if got[3] != d:
            fail("%s: drift=%d, expected %d" % (tag, got[3], d))
        if got[4] != u:
            fail("%s: unbuilt=%d, expected %d" % (tag, got[4], u))
    print()

    # AND NOT ONE ONE-BELT-PER-PART REFUSAL, ANYWHERE IN THE RUN. Every rig here
    # is laid so that no tile ever carries two belts, and every edit is aimed at
    # a spare part, at an edge the rig already has, or at a tile with nothing on
    # it. This is the assertion that says so: a refusal for the other bound would
    # mean a rig or an edit had quietly stopped being the thing it is named for,
    # and every count above it would go on passing. The three ways of REACHING
    # that refusal are the `sedge` suite's subject.
    sedge = [m for m in (SEDGEREFUSED.search(l) for l in lines) if m]
    if sedge:
        fail("%d cluster(s) were refused for the one-belt-per-part rule: %r. No "
             "rig in this suite may ask a part for a second belt"
             % (len(sedge), [m.group(0).strip()[:72] for m in sedge[:3]]))
    print("one-belt-per-part refusals over the whole run: 0, as every rig and "
          "every edit is laid to avoid one")
    print()

    # ---- conservation across every one of them -----------------------------
    ordered = [(t, counts[t]) for t in EXPECT if t in counts]
    peak = max(v for _, v in ordered)
    floor_ = min(v for _, v in ordered)
    print("items across the one-off edges: %d high, %d low, %d put in"
          % (peak, floor_, stock))
    if peak > first:
        fail("the one-off edges MINTED items: %d against the churn leg's opening "
             "%d" % (peak, first))
    if first - floor_ > LOSS_BOUND * first:
        fail("the one-off edges lost %.2f%% of the items, over the %.0f%% bound"
             % (100.0 * (first - floor_) / first, 100 * LOSS_BOUND))
    print()

    # ---- a recompile is not a removal --------------------------------------
    #
    # The field report. Adding an output belt to an operating balancer used to
    # rain its hidden network onto the ground; the drained items belong back
    # inside the network the recompile just built.
    print("what a recompile under load leaves on the ground:")
    for tag in NO_GROUND:
        print("  %-24s %d" % (tag, ground[tag]))
        if ground[tag] != 0:
            fail("%s put %d items on the ground: a recompile (an edge edit, a "
                 "merge or a split) must reinsert into the new network, not "
                 "spill beside the cluster" % (tag, ground[tag]))
    # The shrink that turns out not to be one. Four outputs to three LOOKS like
    # the capacity fallback's case and is not: P = next_pow2(max(N, M)) is 4 on
    # both sides of the edit, so the butterfly is the same size and everything
    # fits. Kept as a bound because a regression that shrank the interior would
    # show up here; it has never been approached, and reading it as evidence
    # that "a shrink fits with room to spare" is what let the second field
    # report through. The `bmin` leg below is the shrink that shrinks.
    shrink = ground["post-shrink"] - ground["pre-shrink"]
    print("  %-24s %d (bound %d -- P stays 4, so nothing should overflow)"
          % ("post-shrink", shrink, SHRINK_GROUND_BOUND))
    if shrink > SHRINK_GROUND_BOUND:
        fail("the shrink spilled %d items, over the %d bound: reinsertion should "
             "have taken all but the overflow" % (shrink, SHRINK_GROUND_BOUND))

    # And the reinsertion really happened -- a leg where every teardown found an
    # empty network would satisfy every assertion above and prove nothing.
    took = sorted(int(m.group(2))
                  for m in (TOOKBACK.search(l) for l in lines) if m)
    spilled = [int(m.group(1)) for m in (SPILLED.search(l) for l in lines) if m]
    if len(took) < 150:
        fail("only %d recompiles reinserted anything at all; the drained items "
             "were not going back into the networks" % len(took))
    print("  reinserted per recompile: min %d, median %d, max %d over %d "
          "recompiles that carried something"
          % (took[0], took[len(took) // 2], took[-1], len(took)))
    if took[len(took) // 2] < 10:
        fail("the median recompile reinserted %d items: the networks were not "
             "full and this proves nothing" % took[len(took) // 2])
    print()

    # ---- ... and a REMOVAL still spills, which is the other half ------------
    #
    # Every part of the shrk balancer mined at once. Nothing succeeds it, so the
    # items must come back to the world. Without this the whole policy could be
    # "never spill", which would be a different bug with a quieter symptom.
    #
    # TWO WINDOWS MAY SPILL AND NOTHING ELSE MAY. The first is the removal above.
    # The second is the by-hand teardown below it, whose SHRINKS overflow into a
    # smaller machine on purpose -- that overflow is the miner's where there is a
    # miner, and this suite has none, so it lands on the floor and is counted
    # here rather than treated as a leak.
    windows = {"remove begin": "removal", "hand mine": "hand",
               "bmin-remove begin": "bmin"}
    spills = {"removal": [], "hand": [], "bmin": []}
    phase = None
    for line in lines:
        opened = next((v for k, v in windows.items()
                       if "[BBB-EDGE] " + k in line), None)
        if opened:
            phase = opened
            continue
        m = MARK.search(line)
        if m:
            phase = None
            continue
        s = SPILLED.search(line)
        if s and phase:
            spills[phase].append(int(s.group(1)))
    removal, hand, bmin = spills["removal"], spills["hand"], spills["bmin"]
    inside = removal + hand + bmin
    print("a cluster DISSOLVED, which is a removal and not a recompile:")
    print("  spills in the removal window:   %d, totalling %d items"
          % (len(removal), sum(removal)))
    print("  spills in the by-hand window:    %d, totalling %d items"
          % (len(hand), sum(hand)))
    print("  spills in the port-boundary window: %d, totalling %d items"
          % (len(bmin), sum(bmin)))
    print("  spills over the whole run: %d, totalling %d items"
          % (len(spilled), sum(spilled)))
    if not removal or sum(removal) < 10:
        fail("mining every part of a saturated balancer returned %d items to "
             "the world: a dissolved cluster has no successor network and its "
             "contents must come back, not vanish into the reinsertion path"
             % sum(removal))
    if len(spilled) != len(inside):
        fail("%d spills happened outside the three windows a player caused and "
             "%d inside: a recompile nobody mined into must not reach the spill "
             "path at all here" % (len(spilled) - len(inside), len(inside)))
    print()

    # ---- the port-boundary shrink -------------------------------------------
    #
    # THE SECOND FIELD REPORT: place an output belt on a running balancer and
    # then mine it again, and a small pile of items appears on the floor beside
    # it -- with an inventory that had room for them.
    #
    # Nothing was lost and nothing was placed wrongly by the recompile: the
    # machine got SMALLER (two outputs to three and back takes P from 2 to 4 and
    # down again, so the butterfly halves), the reinsertion overflowed by
    # carry.go's fourth decision, and the overflow had nowhere to go but the
    # ground because a belt mined BESIDE a cluster recorded no beneficiary. The
    # policy always said a removal a player caused offers its overflow to that
    # player first; the implementation asked only about mined PARTS.
    #
    # The quantity is what this leg pins, because headless has no player and the
    # redirection itself is unreachable -- the same wall as the trigger, measured
    # by the probe below. The `shrk` leg cannot stand in for it: P is 4 on both
    # sides of that edit, so its "shrink" rebuilds the same-size network and its
    # bound has never been approached.
    for tag in ("pre-bmin-remove", "post-bmin-remove"):
        if tag not in ground:
            fail("the port-boundary leg did not run: %s is missing" % tag)
    overflow = ground["post-bmin-remove"] - ground["pre-bmin-remove"]
    print("an OUTPUT BELT placed on a running balancer and then mined again:")
    print("  P went 2 -> 4 on the placement and 4 -> 2 on the removal")
    print("  items on the ground from the removal: %d (floor %d)"
          % (overflow, BMIN_OVERFLOW_MIN))
    print("  (with a player, all %d go to the miner before the floor)" % overflow)
    if overflow < BMIN_OVERFLOW_MIN:
        fail("mining the output belt off a saturated balancer overflowed by %d "
             "items, under the %d floor: this rig did not cross the port "
             "boundary full, so it says nothing about where a shrink's overflow "
             "goes -- which is exactly the hole the `shrk` leg left"
             % (overflow, BMIN_OVERFLOW_MIN))
    if not bmin:
        fail("the port-boundary removal reached no spill at all: either the rig "
             "was empty or the overflow went somewhere this suite cannot see")
    print()

    # ---- taking a balancer apart BY HAND -------------------------------------
    #
    # THE FIELD REPORT, AND THE QUANTITY THE MINER'S POCKET IS ABOUT. A player
    # does not mine a balancer in one tick; they mine a part, the machine
    # recompiles SMALLER, and they mine the next one. Each shrink hands back less
    # than it drained, and the difference falls through to the spill -- which
    # until 2026-08-02 was the ONLY thing that happened to it, because the miner
    # was recorded on the dissolve alone. The dissolve then got the dregs.
    #
    # Headless has no player, so the overflow still lands on the floor here. What
    # is pinned is that it is a REAL quantity: a leg where every shrink happened
    # to fit would satisfy every other assertion in this suite and would say
    # nothing at all about the thing that was fixed.
    hand_tags = ["pre-hand"] + ["hand-%d" % i for i in range(1, 11)]
    if any(t not in ground for t in hand_tags):
        fail("the by-hand teardown leg did not run: %s"
             % ", ".join(t for t in hand_tags if t not in ground))
    steps = [ground[t] - ground["pre-hand"] for t in hand_tags]
    print("taking a saturated balancer apart ONE PART PER TICK:")
    print("  cumulative on the ground: %s"
          % " ".join("%s=%d" % (t, g) for t, g in zip(hand_tags, steps)))
    shrink_overflow = ground["hand-9"] - ground["pre-hand"]
    dissolve = ground["hand-10"] - ground["hand-9"]
    print("  the nine SHRINKS put %d items there and the dissolve %d"
          % (shrink_overflow, dissolve))
    print("  (with a player, all %d go to the miner before the floor)"
          % (shrink_overflow + dissolve))
    if shrink_overflow < HAND_OVERFLOW_MIN:
        fail("the nine shrinks overflowed by %d items, under the %d floor: "
             "this rig was not full enough for the by-hand teardown to say "
             "anything about where a shrink's overflow goes"
             % (shrink_overflow, HAND_OVERFLOW_MIN))
    print()

    # ---- the insert arithmetic, pinned against a real inventory --------------
    #
    # `insert` is a member of LuaControl and a chest is a LuaControl, so the call
    # the miner's pocket makes to a player can be made to a chest -- same member
    # id, same signature, same tier-2 encode of the same table -- with no player
    # anywhere. This is what used to be behind the "unverifiable" wall along with
    # the trigger, and it is the half a field report asked about first.
    probes = [m for m in (PROBE.search(l) for l in lines) if m]
    lua = [m for m in (PROBELUA.search(l) for l in lines) if m]
    if len(probes) != len(PROBE_WANT) or len(lua) != len(PROBE_WANT):
        fail("the insert probe produced %d guest lines and %d lua lines, "
             "expected %d of each: the pocket's arithmetic is unpinned again"
             % (len(probes), len(lua), len(PROBE_WANT)))
    print("the miner's pocket, asked of a steel chest:")
    for m, l, (name, want) in zip(probes, lua, PROBE_WANT):
        what, item, asked, took, held = (m.group(1), m.group(2),
                                         int(m.group(3)), int(m.group(4)),
                                         int(m.group(5)))
        print("  %-16s asked=%-3d took=%-3d held=%-3d lua=%s"
              % (item, asked, took, held, l.group(3)))
        if item != name or asked != want:
            fail("insert probe leg %s asked for %d of %s, expected %d of %s: "
                 "probe.go and this assertion have drifted"
                 % (item, asked, item, want, name))
        if what != "container":
            fail("the insert probe found a %s rather than a container" % what)
        if took != want or held != want or int(l.group(3)) != want:
            fail("insert of %d %s put %d in (engine said %d, lua sees %s): the "
                 "count on the ItemStackDefinition is not reaching the engine, "
                 "which is the miner pocketing one item and spilling the rest"
                 % (want, item, held, took, l.group(3)))
    print()

    # ---- the beneficiary: what this suite CAN say about it ------------------
    #
    # A removal started by a PLAYER hands what no network could take to that
    # player before anything reaches the ground (carry.go, "the beneficiary").
    # The TRIGGER is unverifiable headlessly and stays in CLAUDE.md's table; the
    # arithmetic above is not, and neither is the quantity above that.
    #
    # First, the walls, measured rather than quoted -- if either ever falls, this
    # assertion fails and the real test becomes writable.
    raised = MINERAISE.search("\n".join(lines))
    resolved = MINERESOLVE.search("\n".join(lines))
    if not raised or not resolved:
        fail("the player-mine probe did not run: without it nothing in this "
             "suite says WHY the pocket is untested")
    print("why the pocket cannot be tested headlessly:")
    print("  script.raise_event(on_player_mined_entity): ok=%s -- %s"
          % (raised.group(1), raised.group(2).strip()))
    print("  game.get_player(1) resolves: %s, #game.players=%s"
          % (resolved.group(1), resolved.group(2)))
    if raised.group(1) != "false":
        fail("script.raise_event accepted on_player_mined_entity: the mine "
             "event can be synthesised now, so the beneficiary path is "
             "testable and should be tested rather than documented")
    if resolved.group(1) != "false" or resolved.group(2) != "0":
        fail("a headless --create has a player now (%s, #players=%s): the "
             "pocket is reachable and should be asserted end to end"
             % (resolved.group(1), resolved.group(2)))

    # THE OPERATOR SEAM: the console command and the remote interface.
    #
    # A console command cannot be issued from script (2.0.77 has no
    # `commands.run_command`), so the command leg is asserted against Factorio's
    # OWN registry -- `commands.commands`, the engine's table rather than the
    # mod's claim about itself. The remote leg drives the same handler end to
    # end, and it is evidence about the command leg because one id-dispatched
    # export serves both with no branch that can tell them apart.
    creg = CMDREG.search("\n".join(lines))
    iface = IFACE.search("\n".join(lines))
    rcall = RCALL.search("\n".join(lines))
    if not creg or not iface or not rcall:
        fail("the operator probe did not run: the command and remote seam in "
             "guest/go/commands.go is unasserted")
    print("the operator seam:")
    print("  /bbb-audit in commands.commands: %s" % creg.group(1))
    print("  remote interface 'better-belt-balancer' methods: %s" % iface.group(1))
    print("  remote.call(...,'audit'): ok=%s clusters=%s"
          % (rcall.group(1), rcall.group(2)))
    if creg.group(1) != "true":
        fail("the /bbb-audit console command did not reach Factorio's command "
             "registry: fkapi.AddCommand in guest/go/commands.go's init did not "
             "take effect")
    if iface.group(1) != "audit":
        fail("the remote interface exposes %r, expected exactly 'audit'"
             % iface.group(1))
    if rcall.group(1) != "true":
        fail("remote.call('better-belt-balancer', 'audit') failed: fk_on_call "
             "did not dispatch")
    # A handler that ran but wrote no result comes back as nil, which is the
    # exact signature of a `retp` that WriteDyn never touched -- so it is worth
    # naming separately from a wrong number.
    if rcall.group(2) == "nil":
        fail("remote.call returned nil: fk_on_call dispatched but wrote no "
             "result through WriteDyn(retp, ...)")
    # And the number is checked against the audit THAT CALL ITSELF PROVOKED
    # rather than against a baseline written down here: the cluster count moves
    # legitimately through this suite (a merge, a split, a dissolve), so a
    # constant would be a second thing to maintain and would fail for the wrong
    # reason. The guest logs `command bbb-audit:` on entry and `audit
    # clusters=N` from inside `auditAll`, in that order, in one dispatch.
    joined = "\n".join(lines)
    at = joined.find("[BBB] command bbb-audit:")
    if at < 0:
        fail("the guest never logged its own `command bbb-audit:` line, so "
             "fk_on_call did not reach the handler in commands.go")
    after = AUDIT.search(joined[at:])
    if not after:
        fail("no `[BBB] audit clusters=` line followed the command dispatch: "
             "the handler was entered but auditAll did not run")
    if int(rcall.group(2)) != int(after.group(1)):
        fail("the remote audit RETURNED %s clusters but the audit it ran "
             "reported %s: the value crossing the boundary is not the value "
             "auditAll computed" % (rcall.group(2), after.group(1)))
    print("  ...and that count matches the audit the same call ran (%s)"
          % after.group(1))

    # Second, the negative, which is the half that has teeth. Every removal in
    # this suite is a script destroy, a force merge or a surface event -- not one
    # of them is a player mining -- so the beneficiary must never be consulted.
    # A pool that reached a player here would mean the claim leaked across a
    # removal path it does not belong to.
    pocketed = [m for m in (POCKETED.search(l) for l in lines) if m]
    print("  pocketed lines over the whole run: %d (must be 0)" % len(pocketed))
    if pocketed:
        fail("%d teardowns credited a player in a suite where no player mines "
             "anything: the beneficiary is firing for a script destroy, a "
             "robot or a shrink" % len(pocketed))
    print()

    # ---- and the network it became still balances ---------------------------
    #
    # Four inputs and five outputs, measured as a RATE over the window after the
    # edit rather than as a cumulative total: the four original ports carry
    # thousands of items from before the recompile and the fifth carries none.
    windows_out = {}
    for line in lines:
        r = RIG.search(line)
        if r:
            windows_out.setdefault(r.group(1), {})[r.group(2)] = \
                [int(v) for v in r.group(3).split()]
    a, b = windows_out.get("aout-a"), windows_out.get("aout-b")
    if not a or not b or "aout" not in a or "aout" not in b:
        fail("the aout balance window did not run")
    delta = [y - x for x, y in zip(a["aout"], b["aout"])]
    if len(delta) != 5:
        fail("the aout rig has %d outputs after the edit, expected 5" % len(delta))
    mean = sum(delta) / 5.0
    spread = (max(delta) - min(delta)) / mean if mean else 1.0
    print("the 4->5 balancer the recompile built, over the 500-tick window "
          "after it:")
    print("  per-output %r, spread %.2f%%" % (delta, 100 * spread))
    if mean < 100:
        fail("the aout rig delivered %.0f items per output over the window: it "
             "stopped running after the recompile" % mean)
    if spread > 0.02:
        fail("the 4->5 network the recompile built is %.2f%% out of balance; the "
             "reinserted items must not be observable as an imbalance"
             % (100 * spread))

    # ---- nothing on the hidden surface is ever a balancer ------------------
    if not any(REFUSED.search(l) for l in lines):
        fail("a bbb-balancer-part was built on the HIDDEN surface and the guest "
             "registered it: a cluster there gets a bounding box inside the slot "
             "grid, and its teardown spills the network's items onto a surface "
             "no player can reach")
    print("a part built on the hidden surface was refused, and the world stayed "
          "at fourteen clusters of ninety-seven parts")
    print()

    # ---- two forces becoming one ------------------------------------------
    merged = [m for m in (MERGED.search(l) for l in lines) if m]
    if not merged:
        fail("game.merge_forces raised on_forces_merged and the guest said "
             "nothing: the registry is still holding a force index that no "
             "longer exists")
    m = merged[-1]
    print("forces merged: %s -> %s, %s parts remapped, %s clusters of the "
          "surviving force re-derived" % m.groups())
    # SIX, not three: the other force's rig is four parts under the one-belt rule
    # and the interleaved leg gave it a whole row of two more.
    if int(m.group(3)) != 6:
        fail("the merge remapped %s parts, expected 6" % m.group(3))
    # Every cluster of the surviving force, which is the whole save less the two
    # the other force never had: the merge re-derives them all rather than only
    # the ones near it, because a belt of the source force beside a balancer of
    # the destination force was never an edge and now is.
    if int(m.group(4)) != 14:
        fail("the merge left %s clusters, expected 14" % m.group(4))
    # The merged balancer has to still be a balancer.
    finals = {r.group(2): [int(v) for v in r.group(3).split()]
              for r in (RIG.search(l) for l in lines) if r}
    for name in ("frcA", "frcB"):
        outs = finals.get(name, [])
        if not outs or min(outs) <= 0:
            fail("after the merge, rig %s delivered %r -- the merged balancer "
                 "stopped balancing" % (name, outs))
    print("  and both halves of the merged balancer are still delivering: %r"
          % {n: finals[n] for n in ("frcA", "frcB")})
    print()

    # ---- the deferred queue is deterministic -------------------------------
    windows, order = {}, []
    for line in lines:
        d = DETB.search(line)
        if d:
            order.append((d.group(1), d.group(2), len(order)))
            windows.setdefault(d.group(2), {})[d.group(1)] = len(order)
            continue
    seqs = {}
    for tag in ("1", "2"):
        w = windows.get(tag)
        if not w or set(w) != {"begin", "end", "flushed"}:
            fail("the determinism paste tag=%s did not run" % tag)
        inside, after, phase = [], [], None
        for line in lines:
            d = DETB.search(line)
            if d and d.group(2) == tag:
                phase = d.group(1)
                continue
            if d:
                continue
            c = COMPILED.search(line)
            if c and phase == "begin":
                inside.append(int(c.group(2)))
            elif c and phase == "end":
                after.append(int(c.group(2)))
        if inside:
            fail("paste %s compiled %d networks INSIDE the tick the entities "
                 "arrived in; the whole point of the deferred flush is that it "
                 "compiles none" % (tag, len(inside)))
        seqs[tag] = after
        print("paste %s: 0 builds inside the paste tick, %d on the flush, in "
              "port order %r" % (tag, len(after), after))
    if seqs["1"] != DET_ORDER:
        fail("the flush compiled the scrambled paste in the order %r, not the "
             "order its clusters' first parts arrived (%r)"
             % (seqs["1"], DET_ORDER))
    if seqs["1"] != seqs["2"]:
        fail("two identical scrambled pastes compiled in DIFFERENT orders, %r "
             "and %r: the deferred queue is not deterministic, which in a "
             "lockstep game is a desync" % (seqs["1"], seqs["2"]))
    print("  both pastes produced the same order, and it is the arrival order")
    print()

    # ---- the sixty-fifth belt -----------------------------------------------
    #
    # THE REFUSAL HAPPENS BEFORE THE TEARDOWN, WHICH IS THE WHOLE LEG. `lim` is
    # a column of thirty-two parts carrying sixty-four input belts and one
    # output, so P = next_pow2(max(64, 1)) = 64 = plan.MaxPorts exactly and it is
    # the biggest network this mod builds -- 1,026 entities. A sixty-fifth input
    # belt would need P = 128, and there is no honest way to give it one.
    #
    # WHAT THE UNFIXED GUEST DID, MEASURED (2026-08-04, this rig, this schedule,
    # against the same guest with compile.go's check disabled so the refusal
    # falls back to where it used to be -- inside plan.Build, after the
    # teardown). Verbatim from the run:
    #
    #   [BBB] torn down cluster 32, returned 1876 items
    #   [BBB] error: cluster 32 needs 128 ports, over the limit of 64
    #   [BBB] spilled 1876 items beside cluster 32
    #   [BBB] audit clusters=10 parts=57 nets=9 drift=0 unbuilt=1
    #
    # The working sixty-four-port network was demolished, nothing was built in
    # its place, and 1,876 items came back to the world -- 1,690 of them onto the
    # floor, `ground` going 336 -> 2026 across one belt being laid. Delivery over
    # the two windows below went from 184 items in 246 ticks to TEN, and the ten
    # are the output belt draining. The first assertion to fire is `post-lim:
    # drift=0, expected 1`: the audit's honest report that there is no network
    # left to have drifted.
    #
    # Three more things about that run are worth keeping. The `error:` line
    # appeared THREE times for one belt, because a refused cluster is re-queued
    # by every audit and by every event near it -- which is what the once-per-
    # edge-state assertion below is really about. `test/run.sh` killed the run on
    # the first of them, before a single assertion here was read. And the item
    # total was conserved to the item throughout, which is exactly why eight
    # suites were green over this for five milestones.
    #
    # Four things are asserted, and the first is the one with teeth.
    #
    # TWO refusals happen in this suite and they are told apart by the OUTPUT
    # count: both are 65 inputs now -- `lim` gains its sixty-fifth belt on a
    # spare part and `brdg`'s gap tile carries one flanking belt rather than two
    # -- so `lim` is 65 in and 1 out and `brdg` is 65 in and 2 out. Splitting
    # them here rather than counting the total is what keeps each leg's
    # once-per-edge-state assertion sharp: a second refusal for one edit has to
    # fail on that edit's own count.
    over_all = [m for m in (OVERLIMIT.search(l) for l in lines) if m]
    if len(over_all) != 2:
        fail("the guest refused %d times over the whole run, expected exactly 2 "
             "-- one for the sixty-fifth belt and one for the over-limit merge: %r"
             % (len(over_all), [m.group(0) for m in over_all]))
    over = [m for m in over_all if m.group(4) == "1"]
    over_brdg = [m for m in over_all if m.group(4) == "2"]
    told_all = [m for m in (TOLDFORCE.search(l) for l in lines) if m]
    for tag in ("pre-lim", "post-lim", "post-lim-window", "post-lim-back"):
        if tag not in ground:
            fail("the port-limit leg did not run: %s is missing" % tag)
    lim_ground = ground["post-lim"] - ground["pre-lim"]
    lim_items = counts["pre-lim"] - counts["post-lim"]
    print("a SIXTY-FIFTH input belt laid on a balancer that is already at "
          "plan.MaxPorts:")
    if not over:
        fail("the guest never logged an over-limit refusal: either the rig is "
             "not at 64 ports any more (check LIM_PARTS and the belt on each "
             "side of every part) or the check in compile.go did not fire")
    for m in over:
        print("  refused: cluster %s would need %s ports for %s inputs and %s "
              "outputs, limit %s" % m.groups())
    if len(over) != 1:
        fail("the guest refused %d times for one edit. The refusal is reached "
             "by every audit and by every event within two tiles of the "
             "cluster, and its fingerprint can never match, so the feedback "
             "gate in limit.go must fire ONCE per distinct edge state -- "
             "otherwise a player who leaves the belt standing gets a flying "
             "text and a cannot-build sound for the rest of the session"
             % len(over))
    if int(over[0].group(2)) != 128 or int(over[0].group(5)) != 64:
        fail("the refusal names %s ports against a limit of %s, expected 128 "
             "against 64: the rig is not the shape this leg is about"
             % (over[0].group(2), over[0].group(5)))
    if int(over[0].group(3)) != 65:
        fail("the refused shape has %s inputs, expected 65" % over[0].group(3))

    # 2. NOTHING WAS DEMOLISHED. The network is still in `nets`, so the audit
    #    sees a cluster WITH a network whose fingerprint no longer matches the
    #    world -- drift=1, unbuilt=0 -- which the EXPECT table above asserts
    #    exactly. Here: nothing was drained, so nothing reached the ground and
    #    the conserved total did not move.
    print("  items on the ground from the refusal: %d (must be 0)" % lim_ground)
    print("  the conserved total moved by:          %d" % lim_items)
    if lim_ground != 0:
        fail("the refused edit put %d items on the ground: the check is behind "
             "the teardown again, so a working balancer was demolished to "
             "discover that its replacement would not fit" % lim_ground)
    if lim_items > 0:
        fail("%d items went missing across the refused edit: a refusal must not "
             "touch the network at all" % lim_items)

    # 3. AND IT KEPT RUNNING. Two windows of the same length either side of the
    #    edit, compared as a RATIO rather than against a constant: sixty-three
    #    of this rig's output ports dead-end, so the share of the input that
    #    reaches the live one climbs all run as they back-fill, and a fixed
    #    expected number would be a figure copied from a passing run.
    lim = {m.group(1): (int(m.group(2)), int(m.group(3))) for m in
           (LIM.search(l) for l in lines) if m}
    need = ("before-open", "before-close", "after-open", "after-close")
    if any(t not in lim for t in need):
        fail("the port-limit delivery windows did not run: %s"
             % ", ".join(t for t in need if t not in lim))
    before = lim["before-close"][1] - lim["before-open"][1]
    after = lim["after-close"][1] - lim["after-open"][1]
    bt = lim["before-close"][0] - lim["before-open"][0]
    at_ = lim["after-close"][0] - lim["after-open"][0]
    print("  the standing 64-port network, either side of the refusal:")
    print("    %d items over %d ticks BEFORE, %d over %d AFTER"
          % (before, bt, after, at_))
    if before <= 0:
        fail("the 64-port balancer delivered nothing over the window BEFORE the "
             "edit: this leg's control is dead and it proves nothing about the "
             "refusal")
    rate_b, rate_a = before / float(bt), after / float(at_)
    print("    %.3f items/tick before, %.3f after (%.0f%%)"
          % (rate_b, rate_a, 100 * rate_a / rate_b))
    if rate_a < 0.5 * rate_b:
        fail("delivery fell from %.3f to %.3f items/tick across the refused "
             "edit: the standing network was disturbed by an edit that was "
             "supposed to leave it entirely alone" % (rate_b, rate_a))

    # 4. SOMEBODY WAS TOLD, and the half of that which a headless run can reach.
    #    Every build in every suite is a script build, so `player_index` is zero
    #    and the fork always takes its `force.print` arm. That message goes to
    #    the game's chat, which no script can read back and which --benchmark
    #    does not log -- so the guest logs the fact instead, and the two things
    #    behind that line are the ones that could realistically fail: resolving
    #    a LuaForce from a force INDEX (the registry keeps no handle, so it comes
    #    off a part standing on the cluster) and the LocalisedString crossing the
    #    boundary at all.
    told = [m for m in told_all if m.group(2) == over[0].group(1)]
    print("  the force was told: %d time(s)%s"
          % (len(told), (" -- " + told[0].group(3).strip()) if told else ""))
    if len(told) != 1:
        fail("the force was told %d times, expected exactly 1: a headless build "
             "has no player_index, so every refusal here must take the "
             "force.print arm and the feedback gate must fire once" % len(told))
    if "FAILED" in told[0].group(3):
        fail("force.print returned an error: the LocalisedString or the "
             "LuaForce this mod resolves from a force index is not reaching the "
             "engine, so nobody is being told anything")

    # 5. AND THE NEGATIVE, which is the same shape as the pocket's. A headless
    #    --create has no players, so `game.get_player` resolves to nothing and
    #    the hand-back returns before it mines anything -- exactly as the
    #    beneficiary is never consulted anywhere in this suite. A revert firing
    #    here would mean it is firing for a build nobody made.
    handed = [l for l in lines if HANDEDBACK.search(l) or HANDBACKFAIL.search(l)]
    print("  pieces handed back over the whole run: %d (must be 0 -- no player "
          "exists to hand one to)" % len(handed))
    if handed:
        fail("%d over-limit pieces were handed back in a suite with no players: "
             "the revert is running for a script build, which has nobody to "
             "give a belt to" % len(handed))
    print()

    # ---- the merge that would be over the limit ------------------------------
    #
    # THE OTHER SHAPE OF THE SAME REFUSAL, and the one `lim`'s fix could not
    # reach. compile()'s check sits in front of the teardown compile() does to
    # ITSELF, which is the whole answer for an edge edit and no answer at all for
    # a merge: `AddPart` marks BOTH predecessors' roots dead, so flushDead has
    # already demolished them by the time flushLive looks at what they became.
    #
    # WHAT THE UNFIXED GUEST DID, MEASURED (2026-08-05, this rig, this schedule,
    # against the same guest with the call at the top of flushDead removed).
    # Verbatim:
    #
    #   [BBB] merge 18+64->18 (17 parts)
    #   [BBB] merge 18+80->18 (33 parts)
    #   [BBB] torn down cluster 64, returned 1044 items
    #   [BBB] torn down cluster 80, returned 1044 items
    #   [BBB] alert: cluster 18 would need 128 ports for 66 inputs and 2 outputs
    #   [BBB] spilled 1044 items beside cluster 64
    #   [BBB] spilled 1044 items beside cluster 80
    #   [BBB] audit clusters=11 parts=90 nets=10 drift=0 unbuilt=1
    #
    # Two working balancers, 2,088 items drained, both spilled, nothing built:
    # `ground` went 336 -> 2150 across one part being placed, and delivery over
    # the two windows below fell from 186 and 185 items in 246 ticks to EIGHT
    # each, which is the output belts draining. The item total was conserved to
    # the item throughout -- which is exactly why this suite was green over it
    # for a milestone.
    #
    # Note what the alert said while it happened: "refused BEFORE the teardown,
    # so the standing network is untouched". True of the cluster being refused,
    # which had no network, and a lie about the two that did.
    brdg = {m.group(1): (int(m.group(2)), int(m.group(3)), int(m.group(4)))
            for m in (BRDG.search(l) for l in lines) if m}
    need = ("before-open", "before-close", "after-open", "after-close",
            "back-open", "back-close")
    if any(t not in brdg for t in need):
        fail("the over-limit merge leg did not run: %s"
             % ", ".join(t for t in need if t not in brdg))
    print("a part BRIDGING two working balancers into one that is over the "
          "limit:")

    # 1. THE REFUSAL, and it names the merged shape rather than either half's.
    if not over_brdg:
        fail("the guest never refused the merge: either the rig's two halves no "
             "longer add up to 66 inputs (check BRDG_HALF and the two belts on "
             "the gap tile) or the merged cluster was never classified")
    for m in over_brdg:
        print("  refused: cluster %s would need %s ports for %s inputs and %s "
              "outputs, limit %s" % m.groups())
    if len(over_brdg) != 1:
        fail("the guest refused the merge %d times for one part. Every audit and "
             "every event within two tiles re-queues the merged cluster and its "
             "fingerprint can never match, so limit.go's feedback gate must fire "
             "ONCE per distinct edge state" % len(over_brdg))
    if int(over_brdg[0].group(2)) != 128 or int(over_brdg[0].group(3)) != 65:
        fail("the refused merge names %s ports for %s inputs, expected 128 for "
             "65: the rig is not the shape this leg is about"
             % (over_brdg[0].group(2), over_brdg[0].group(3)))

    # 2. AND THE TEARDOWNS WERE TAKEN OFF THE QUEUE, which is the fix itself.
    #    The guest says how many networks it left alone; it must be BOTH, and it
    #    must say so exactly once. One would mean only the survivor's own was
    #    spared -- the absorbed half demolished -- which conserves items, leaves
    #    the ground clean, and empties half the machine.
    spared = [m for m in (SPARED.search(l) for l in lines) if m]
    print("  standing networks the refusal left alone: %s"
          % ([int(m.group(2)) for m in spared] or "NONE"))
    if len(spared) != 1 or int(spared[0].group(2)) != 2:
        fail("the guest spared %r networks (expected exactly one report of 2): "
             "a merge past the limit must take BOTH predecessors' teardowns off "
             "the queue, and must do it once"
             % [int(m.group(2)) for m in spared])

    # 3. NOTHING WAS TOUCHED AT ALL. Not the ground, not the conserved total,
    #    and -- the sharp one -- not a single teardown or build in either
    #    dispatch. `lim` gets this for free because there is nothing queued to
    #    tear down; here there were two, and they had to be cancelled.
    for what, a, b in (("the merge", "brdg-add begin", "mark tag=post-brdg"),
                       ("the un-merge", "brdg-remove begin",
                        "mark tag=post-brdg-back")):
        win, on = [], False
        for line in lines:
            if a in line:
                on = True
                continue
            if on:
                if b in line:
                    break
                win.append(line)
        if not win:
            fail("the %s window is empty: the leg did not run" % what)
        tore = [m.group(1) for m in (TORNDOWN.search(l) for l in win) if m]
        built = [m.group(1) for m in (COMPILED.search(l) for l in win) if m]
        spills = [m.group(1) for m in (SPILLED.search(l) for l in win) if m]
        print("  across %-12s teardowns=%d builds=%d spills=%d"
              % (what, len(tore), len(built), len(spills)))
        if tore or built or spills:
            fail("%s tore down %r, built %r and spilled %r. Both halves' "
                 "fingerprints are the ones their netInfos already hold, so the "
                 "right number of each is zero: a refused merge must leave two "
                 "running balancers exactly as it found them, and mining the "
                 "bridge back out must be a SKIP"
                 % (what, tore, built, spills))
    # `ground` is cumulative by this point in the schedule -- three earlier legs
    # spilled on purpose -- so both halves of this are DELTAS.
    for what, a, b in (("the merge", "pre-brdg", "post-brdg"),
                       ("the un-merge", "post-brdg-window", "post-brdg-back")):
        dg, di = ground[b] - ground[a], counts[a] - counts[b]
        print("  %-14s put %d items on the ground and moved the total by %d "
              "(both must be 0)" % (what, dg, di))
        if dg != 0 or di != 0:
            fail("%s put %d items on the ground and moved the conserved total "
                 "by %d: the two standing balancers were demolished to discover "
                 "that what they became would not fit" % (what, dg, di))

    # 4. THE AUDIT COUNTS WHAT IS STANDING. A spared network is keyed by a root
    #    that is no longer a root, so `liveRootList` can never reach it and
    #    limit.go's `strandedNets` is the only thing that knows it exists. The
    #    EXPECT table above already asserts drift=1 and unbuilt=0 at these tags;
    #    this is the `nets=` column, which is what says the two balancers a
    #    player is looking at are accounted for rather than merely uncounted.
    for tag in BRDG_REFUSED:
        if audits[tag][2] != BRDG_NETS:
            fail("%s: the audit reported nets=%d with a refused merge standing, "
                 "expected %d. Two networks are keyed by roots the audit's own "
                 "walk cannot reach, and if it does not ask limit.go for them it "
                 "under-reports the save" % (tag, audits[tag][2], BRDG_NETS))
    print("  the audit counted %d networks at all four refused samples, "
          "including the two under keys that are no longer roots" % BRDG_NETS)

    # 5. AND BOTH HALVES KEPT RUNNING. Three windows of the same length: before
    #    the merge, across the refusal, and after the bridge is mined out again.
    #    Ratios rather than constants, for `lim`'s reason -- thirty-one of each
    #    half's thirty-two output ports dead-end, so both back-fill all run.
    for tag, (t0, t1) in (("across the refusal", ("before", "after")),
                          ("after the un-merge", ("before", "back"))):
        b_t = brdg[t0 + "-close"][0] - brdg[t0 + "-open"][0]
        a_t = brdg[t1 + "-close"][0] - brdg[t1 + "-open"][0]
        for half in (1, 2):
            b = brdg[t0 + "-close"][half] - brdg[t0 + "-open"][half]
            a = brdg[t1 + "-close"][half] - brdg[t1 + "-open"][half]
            print("    half %s %-20s %d items over %d ticks -> %d over %d (%.0f%%)"
                  % ("AB"[half - 1], tag, b, b_t, a, a_t, 100.0 * a / max(b, 1)))
            if b <= 0:
                fail("half %s delivered nothing over the window BEFORE the "
                     "merge: this leg's control is dead" % "AB"[half - 1])
            if a < 0.5 * b:
                fail("half %s delivered %d items over %d ticks %s against %d "
                     "over %d before: the balancer was demolished by an edit "
                     "that was supposed to leave it entirely alone"
                     % ("AB"[half - 1], a, a_t, tag, b, b_t))

    # 6. SOMEBODY WAS TOLD, once, on the arm a headless run can reach.
    told_b = [m for m in told_all if m.group(2) == over_brdg[0].group(1)]
    if len(told_b) != 1 or "FAILED" in told_b[0].group(3):
        fail("the force was told about the refused merge %d time(s)%s, expected "
             "exactly 1 clean report"
             % (len(told_b), " with a print FAILURE" if told_b and
                "FAILED" in told_b[0].group(3) else ""))
    print("  the force was told once, and the LocalisedString reached the engine")
    print()

    # ---- fast replace ------------------------------------------------------
    #
    # `bbb-balancer-part` carries `fast_replaceable_group = "transport-belt"`,
    # base's own group for every belt, underground, splitter and lane splitter.
    # A part held over a belt replaces it the way a splitter does.
    #
    # THE GROUP IS SYMMETRIC, which is what makes the second half of this leg
    # about guest code rather than about a data-stage line. A belt laid on a part
    # destroys that part and the engine raises NO EVENT for it -- measured on
    # 2.0.77: the only event in the whole dispatch is the BUILD event for the
    # belt. Without guest/go/fastreplace.go the registry keeps a phantom: a tile
    # it calls a balancer part which is holding a player's belt. The audit cannot
    # see it either, because a phantom tile is INTERIOR, so the belt standing on
    # it is never classified and the fingerprint never moves.
    #
    # WHAT THE UNFIXED GUEST DOES, MEASURED (this rig, this schedule, against
    # the same guest with the check in reapFastReplaced disabled):
    #
    #   post-frep-rev: the guest saw 14 clusters of 96 parts, expected 15 of 95
    #
    # and the frepb column goes on reporting four parts in one cluster, with the
    # belt inert, for the rest of the run.
    cans = {m.group(1): m.group(2) for m in (FREPCAN.search(l) for l in lines) if m}
    spills_fr = {m.group(1): (m.group(2), int(m.group(3)), m.group(4).strip())
                 for m in (FREPSPILL.search(l) for l in lines) if m}
    need = ("part-over-belt", "belt-over-edge-part", "belt-over-interior-part")
    if any(t not in cans for t in need):
        fail("the fast-replace leg did not run: %s"
             % ", ".join(t for t in need if t not in cans))
    print("what the engine says a player may fast-replace:")
    for what in need:
        print("  %-24s %s" % (what, cans[what]))

    # 1. THE FORWARD GESTURE, which is the feature. `can_fast_replace` is FALSE
    #    without the prototype line -- that is this leg's data-stage red proof --
    #    and the belt has to actually be gone afterwards, because
    #    `create_entity{fast_replace = true}` creates a part on top of a belt it
    #    could not replace rather than refusing.
    if cans["part-over-belt"] != "true":
        fail("the engine refuses to fast-replace a belt with a balancer part: "
             "`fast_replaceable_group` is missing from bbb-balancer-part in "
             "mod-data/prototypes/entity.lua, or it is not base's own "
             "\"transport-belt\"")
    fwd = FREPFWD.search("\n".join(lines))
    if not fwd:
        fail("the forward fast-replace leg did not report")
    print("a PART fast-replaced onto a belt of a running line:")
    print("  created=%s  belt still there=%s  part there=%s"
          % (fwd.group(1), fwd.group(2), fwd.group(3)))
    if fwd.group(1) != "true" or fwd.group(3) != "true":
        fail("the part was not created over the belt")
    if fwd.group(2) != "false":
        fail("the belt is STILL THERE under the new part: create_entity fell "
             "back to creating rather than replacing, so this leg measured two "
             "entities on one tile instead of a fast replace")
    if "fwd" not in spills_fr:
        fail("the forward leg logged no ground sample")
    handed, machine, where = spills_fr["fwd"]
    print("  what the replace handed back: [%s], %s (%d of them the engine's "
          "own machine item, removed so the conserved total stays conserved)"
          % (handed, where, machine))
    if "express-transport-belt" not in handed:
        fail("the belt itself was not handed back: with no player the engine "
             "spills it, and nothing else in this window can produce one")
    if machine != 1:
        fail("the forward replace put %d machine items on the ground, expected "
             "exactly 1 (the belt)" % machine)

    # 2. AND THE BALANCER IT BECAME BALANCES. Three in and TWO out over four
    #    ports -- the line the part was dropped into ends on that tile, so it
    #    brings an input and no output -- measured as a rate over the window
    #    after the edit rather than as a cumulative total.
    fa, fb = windows_out.get("frep-after-open"), windows_out.get("frep-after-close")
    ba, bb = windows_out.get("frep-before-open"), windows_out.get("frep-before-close")
    for w, n in ((fa, "frep-after-open"), (fb, "frep-after-close"),
                 (ba, "frep-before-open"), (bb, "frep-before-close")):
        if not w or "frepa" not in w or "frepb" not in w:
            fail("the fast-replace window %s did not run" % n)
    da = [y - x for x, y in zip(fa["frepa"], fb["frepa"])]
    if len(da) != 2:
        fail("the frepa rig has %d outputs after the edit, expected 2" % len(da))
    mean = sum(da) / float(len(da))
    spread = (max(da) - min(da)) / mean if mean else 1.0
    print("  the 3->2 balancer the replace built, over the window after it:")
    print("    per-output %r, spread %.2f%%" % (da, 100 * spread))
    if mean < 100:
        fail("the frepa rig delivered %.0f items per output: it stopped running "
             "after the part was dropped into the line" % mean)
    if spread > 0.02:
        fail("the 3->2 network is %.2f%% out of balance: a part dropped into a "
             "live belt line must produce a balancer like any other"
             % (100 * spread))

    # 3. THE REFUSAL. A part that carries an edge interface cannot be
    #    belt-replaced: `bbb-linked-belt` is a belt-connectable of its own
    #    standing on that same tile, so the engine's own check says no. That is
    #    what a player's cursor gets, and it is the only thing that keeps the
    #    reverse gesture off the edges of a machine.
    edge = FREPEDGE.search("\n".join(lines))
    if not edge:
        fail("the fast-replace refusal leg did not report")
    print("a BELT over a part that carries an edge interface:")
    print("  can_fast_replace=%s  create_entity created=%s  part survived=%s"
          % (cans["belt-over-edge-part"], edge.group(1), edge.group(2)))
    if cans["belt-over-edge-part"] != "false":
        fail("the engine would let a player lay a belt on a part that carries "
             "an edge interface. Two belt-connectables on one tile is the whole "
             "of spike S1's loophole and it is not something to rely on the "
             "other way round: re-measure before believing this")
    if edge.group(1) != "false":
        fail("create_entity placed a belt on a part carrying an interface: it "
             "returned an entity where it has always returned nil")

    # 4. THE REVERSE, and the guest noticing it. `can_fast_replace` is true for
    #    the middle of frepb's NECK, the part goes, the belt takes the tile --
    #    and the ONLY thing that tells the guest is the belt's own build event.
    rev = FREPREV.search("\n".join(lines))
    if not rev:
        fail("the reverse fast-replace leg did not report")
    print("a BELT fast-replaced onto an INTERIOR part:")
    print("  can_fast_replace=%s  created=%s  part still there=%s  belt there=%s"
          % (cans["belt-over-interior-part"], rev.group(1), rev.group(2),
             rev.group(3)))
    if cans["belt-over-interior-part"] != "true":
        fail("the engine refuses a belt over an interior part, so the reverse "
             "gesture this leg is about cannot happen and the guard it proves "
             "is unreachable")
    if rev.group(1) != "true" or rev.group(3) != "true":
        fail("the belt was not created over the part")
    if rev.group(2) != "false":
        fail("the part is still standing under the belt: create_entity fell "
             "back to creating rather than replacing")
    reaped = [m for m in (REAPED.search(l) for l in lines) if m]
    print("  the guest unregistered it: %d time(s) %r"
          % (len(reaped), [(m.group(1), m.group(2)) for m in reaped]))
    if len(reaped) != 1:
        fail("the guest logged %d fast-replace removals, expected exactly 1. "
             "None at all means guest/go/fastreplace.go never fired and the "
             "registry is holding a tile that a player's belt is standing on; "
             "more than one means it is firing for something that is not a "
             "replace" % len(reaped))
    # That line can only be written when the part was ALREADY GONE by the time
    # the belt's build event arrived, so it is also this suite's measurement of
    # the thing it cannot ask directly: the engine raised no removal event for
    # the entity it replaced. What a PLAYER's fast replace raises is behind the
    # same wall as the miner's pocket -- there is no player in a --create -- and
    # the guard is written to be correct whichever way that falls (see
    # guest/go/fastreplace.go).
    print("  ...which is only reachable if no removal event preceded it, so the "
          "engine raised none")
    if "rev" not in spills_fr:
        fail("the reverse leg logged no ground sample")
    handed, machine, where = spills_fr["rev"]
    print("  what the replace handed back: [%s], %s (%d machine items removed)"
          % (handed, where, machine))
    if "bbb-balancer-part" not in handed:
        fail("the PART itself was not handed back: with no player the engine "
             "puts it on the belt it just created, or on the floor if there is "
             "none, exactly as a mined machine goes")
    if machine != 1:
        fail("the reverse replace put %d machine items on the ground, expected "
             "exactly 1 (the part)" % machine)

    # 5. AND BOTH HALVES OF THE SPLIT KEEP RUNNING, AND THE NEW BELT IS AN EDGE
    #    OF BOTH OF THEM. The column became a two-part cluster above the belt and
    #    a one-part cluster below it, and that one belt is an OUTPUT of the first
    #    and an INPUT of the second -- two networks in series through a tile that
    #    used to be inside one of them. Windows of the same length either side.
    #
    #    THE COLUMN DELIVERS LESS AFTERWARDS AND THAT IS THE CORRECT ANSWER
    #    rather than a regression, so the bound is set knowing why. Before: two
    #    independent belts in and two out, 2.0 belts delivered. After: the lower
    #    cluster has TWO inputs -- its own belt and the one coming down from
    #    upstairs -- and one output, and a balancer equalises its inputs, so it
    #    draws half a belt from each and delivers 1.0; the upper splits its one
    #    belt between its own output and the belt feeding downstairs and delivers
    #    0.5. 1.5 belts of 2.0, measured at 76%: [262, 262] -> [132, 264]. What
    #    must not happen is a half that STOPS.
    #
    #    THE NECK IS WHAT MAKES THIS SHAPE POSSIBLE AT ALL under the one-belt
    #    rule. The belt that splits the column becomes an edge of the part above
    #    it AND of the part below it, so both of those must be otherwise
    #    edgeless -- and a second column of parts beside the target would keep
    #    the cluster connected around the belt and there would be no split.
    b_tot = [y - x for x, y in zip(ba["frepb"], bb["frepb"])]
    a_tot = [y - x for x, y in zip(fa["frepb"], fb["frepb"])]
    print("  frepb either side of the split: %r -> %r (%.0f%% of the column)"
          % (b_tot, a_tot, 100.0 * sum(a_tot) / max(sum(b_tot), 1)))
    if min(b_tot) <= 0:
        fail("frepb delivered nothing BEFORE the reverse replace: this leg's "
             "control is dead and it proves nothing")
    if min(a_tot) <= 0:
        fail("frepb delivered %r after the split: one half of the column stopped "
             "balancing, which is what a phantom part -- or an interface placed "
             "on a tile that now holds a belt -- looks like" % a_tot)
    if sum(a_tot) < 0.7 * sum(b_tot):
        fail("frepb delivered %d items after the split against %d before: the "
             "cascade costs half a belt and no more"
             % (sum(a_tot), sum(b_tot)))
    # The LOWER half GAINED an input, and it is fed through the new belt only if
    # the guest noticed the replace and re-derived the cluster at all.
    if a_tot[1] < b_tot[1]:
        fail("the lower half of frepb delivered %d after the split against %d "
             "before: it gained an input and cannot have got slower"
             % (a_tot[1], b_tot[1]))
    print()

    # ---- and nothing of ours stands where nothing covers it ----------------
    #
    # The structural half of the tan-streak fix. Blanking `bbb-linked-belt`'s
    # pictures stops the interface from drawing anything ITSELF; this is the
    # other half of the same claim, and it is the durable one -- that an
    # interface is only ever placed on a tile that a balancer part's own opaque
    # sprite already covers. The two together are what make "an edge interface
    # is invisible" a property of the compiler rather than of one prototype.
    places = {}
    for line in lines:
        m = PLACE.search(line)
        if m:
            places[m.group(1)] = tuple(int(g) for g in m.groups()[1:])
    strays = {}
    for line in lines:
        m = STRAY.search(line)
        if m:
            strays.setdefault(m.group(1), []).append(m.group(2))
    print("every visible-surface entity the compiler created, against the tiles "
          "the registry calls parts:")
    print("  %-14s %6s %8s %9s %7s" % ("sample", "ours", "on a part", "off one",
                                       "parts"))
    for tag in PLACE_TAGS:
        if tag not in places:
            fail("the placement probe never ran at %s" % tag)
        ours, onpart, offpart, nparts = places[tag]
        print("  %-14s %6d %8d %9d %7d" % (tag, ours, onpart, offpart, nparts))
        if ours < PLACE_MIN:
            fail("the placement probe found %d entities of ours on the visible "
                 "surface at %s: it was looking at an empty world and proves "
                 "nothing" % (ours, tag))
        if offpart:
            fail("%d of our entities stand on a tile that is not a registered "
                 "balancer part at %s: %r. A part's sprite is exactly one opaque "
                 "tile, so anything of ours outside one is drawn over bare "
                 "ground -- which is the whole tan-streak class"
                 % (offpart, tag, strays.get(tag, [])[:8]))

    # ... including the field report's own shape, saturated the whole time. The
    # notch at (1, b+1) is the one tile in this save enclosed by parts and not
    # one, so it is where a stray would land; the probe above covers it and this
    # asserts the rig really was flowing while it did.
    n_a = (windows_out.get("aout-a") or {}).get("ntch")
    n_b = (windows_out.get("aout-b") or {}).get("ntch")
    if not n_a or not n_b or len(n_b) != 2:
        fail("the 2x2-minus-a-corner rig did not report; the probe's headline "
             "sample was taken over a rig that may not have been running")
    flow = [y - x for x, y in zip(n_a, n_b)]
    print("  the 2x2-minus-a-corner rig, over the same 500-tick window: "
          "per-output %r" % flow)
    if min(flow) <= 0:
        fail("the notch rig delivered %r over the window: it was not flowing, "
             "and the field report's artifact only appears when it is" % flow)

    print()
    print("edge assertions passed")


if __name__ == "__main__":
    main()
