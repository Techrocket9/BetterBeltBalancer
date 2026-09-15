#!/usr/bin/env python3
"""Assert the gestures that need a PLAYER, driven from a real player's cursor.

Every section of CLAUDE.md that ends "only the TRIGGER is unverifiable
headlessly" is about this suite. A `game.players` entry exists only where
somebody once connected, so fifteen of the sixteen suites have none and
`game.get_player(1)` is nil in all of them; this one loads a committed save a
graphical client made (test/fixtures-player/) and drives seven gestures through
`build_from_cursor` and `mine_entity`:

  (a) restore     a part fast-replaced onto the MIDDLE of a loaded belt line.
                  Refused by the one-belt-per-part rule, the part handed back --
                  and the BELT PUT BACK, empty, paid for out of the player's own
                  inventory. That last clause is the fix; before it the tile
                  stayed empty and the player kept a belt the engine refunded.
  (b) end         a part onto the END of a belt line beside a working 2->2.
                  Accepted: the belt and its cargo to the player, the machine
                  3->2 and still delivering.
  (c) lone, col   a belt over a part, twice: over one standing alone, and over
                  the middle of a five-part column, which SPLITS it.
  (g) second      a second belt against a part that already has its one.
                  Refused and handed back.
  (f) brdg        a part in the one-tile gap between two 32-port balancers.
                  The merge is over the limit: refused, nothing torn down, both
                  halves still delivering, the part handed back.
  (e) lim,        the sixty-fifth belt on a 64-port balancer, twice: once with
      limfull     room to hand it back and once with the inventory FULL, where
                  it must stay standing and the player must be told.
  (d) bmin, pock  the miner's pocket, both field reports. An output belt mined
                  off a running balancer whose P halves, and a saturated
                  balancer taken apart one part per tick.
  (h) curve       THE CURVED EXIT, and the mod portal report about it. One belt
                  clicked onto a free face is an output, because the head of a
                  line has nothing behind it; the same line laid from three
                  tiles out never is; and giving the grabbed belt a belt behind
                  it gives it back. The rule is unchanged and this is what it
                  does, stated in gestures.
  (i) lb          a linked belt at that belt's REAR, which the engine counts as
                  a feeder and the probe could not see. Plus the engine's own
                  reading of the same two shapes with no balancer near them.

THE PLAYER IS IN THE GOD CONTROLLER. A save written by a headless server holds
every player DISCONNECTED and with no character, and a disconnected player
cannot be given one -- "User isn't connected; can't write character". God mode
is the controller such a player CAN hold, and it produces the same event trace,
the same mine buffer and the same inventory arithmetic as a connected character:
guest/go/obs/curs's header is the measured table.

WHAT IS NOT COVERED IS THE PIXELS. The flying text's position and colour, the
build preview over the cursor and the cannot-build sound are a human's still,
and test/interactive/README.md is where they live.

    python3 test/assert-curs.py run.log
"""

import re
import sys

# ---------------------------------------------------------------------------
# what the mod says
# ---------------------------------------------------------------------------

SEDGE_REFUSED = re.compile(
    r"\[BBB\] alert: cluster (\d+) has (\d+) parts? carrying more than one belt, "
    r"worst (\d+)")
OVERLIMIT = re.compile(
    r"\[BBB\] alert: cluster (\d+) would need (\d+) ports for (\d+) inputs and "
    r"(\d+) outputs, over the limit of (\d+)")
HANDEDBACK = re.compile(
    r"\[BBB\] handed the refused piece at (-?\d+),(-?\d+) \(([^)]+)\) back to "
    r"player (\d+)")
NOROOM = re.compile(
    r"\[BBB\] alert: player (\d+) could not be handed back the refused piece at "
    r"(-?\d+),(-?\d+) \(([^)]+)\) -- no room in the inventory")

# THE BELT PUT BACK, which is the one line in this file that did not exist when
# the suite was written. A refused fast replace destroyed a player's belt: the
# engine mines it as part of the gesture, the mod then refuses the part and hands
# it back, and the tile the belt was on stayed EMPTY with the refunded belt item
# in the player's pocket. The fix creates the belt again -- same direction, same
# force, same prototype, empty -- and charges the player one belt item for it, so
# the count goes back to what it was before the click.
RESTORED = re.compile(
    r"\[BBB\] put the replaced belt at (-?\d+),(-?\d+) back for player (\d+) "
    r"\(([^)]+)\), taking one from their inventory")
# Its two failure arms, both at alert level. Neither may fire in this suite: the
# rig gives the player belts and leaves the tile clear.
RESTORE_UNPAID = re.compile(
    r"\[BBB\] alert: player (\d+) had no (\S+) to pay for the replaced belt at "
    r"(-?\d+),(-?\d+); the gap stays")
RESTORE_FAILED = re.compile(
    r"\[BBB\] alert: could not put the replaced belt at (-?\d+),(-?\d+) back "
    r"\(([^)]+)\); the item was returned to player (\d+)")

OFFERED = re.compile(
    r"\[BBB\] cluster (\d+) offered (\d+) items to player (\d+) before the floor")
POCKETED = re.compile(
    r"\[BBB\] cluster (\d+) was mined by player (\d+); pocketed (\d+) items")
SPILLED = re.compile(r"\[BBB\] .*spilled (\d+) items")
TORNDOWN = re.compile(r"\[BBB\] torn down cluster (\d+), returned (\d+) items")
COMPILED = re.compile(
    r"\[BBB\] compiled cluster (\d+) (\d+)->(\d+) over (\d+) ports, (\d+) entities")
DISSOLVED = re.compile(r"\[BBB\] cluster (\d+) dissolved, mined by player (\d+)")
AUDIT = re.compile(
    r"\[BBB\] audit clusters=(\d+) parts=(\d+) nets=(\d+) drift=(\d+) "
    r"unbuilt=(\d+) refused=(\d+)")

# ---------------------------------------------------------------------------
# what the observer says
# ---------------------------------------------------------------------------

PLAYER = re.compile(r"\[BBB-CURS\] player tag=(\S+) present=(\S+)(.*)$")
SAMPLE = re.compile(r"\[BBB-CURS\] sample tag=(\S+) (.*)$")
AUDITED = re.compile(r"\[BBB-CURS\] audited tag=(\S+)")
RATE = re.compile(r"\[BBB-CURS\] rate tag=(\S+) delivered=(\d+)")
LOADED = re.compile(r"\[BBB-CURS\] line loaded placed=(\d+)")
FILLED = re.compile(r"\[BBB-CURS\] inventory filled=(\d+) spare=(\d+) is_full=(\S+)")
POCKMINE = re.compile(
    r"\[BBB-CURS\] pock-mine step=(\d+) x=(-?\d+) y=(-?\d+) took=(\S+)")
BMINMINE = re.compile(r"\[BBB-CURS\] bmin-mine took=(\S+)")

# Bands (h) and (i) report on a line of their own rather than through `sample`,
# because the release in (h3) spills on purpose and every `sample` tag in this
# file is asserted to have found nothing on the ground.
CURVE = re.compile(r"\[BBB-CURS\] curve tag=(\S+) (.*)$")
ENGINE = re.compile(
    r"\[BBB-CURS\] engine tag=(\S+) y=(-?\d+) shape=(\S+) rear=(\S+)")

BELT = "express-transport-belt"
PART = "bbb-balancer-part"
PLATE = "iron-plate"

# `defines.controllers.god` and two `defines.direction` values, as the engine
# numbers them. The observer reports whatever its own accessors gave it, so these
# are transcriptions of what a run produced rather than constants a guest wrote
# down -- which is the same standing every define in this repository has.
GOD = 2
WEST = 12
SOUTH = 8

# The tile each gesture is aimed at, from guest/go/obs/curs/main.go. They are
# written down here rather than parsed out of the log so that a rig that moved
# fails by name instead of quietly asserting a different tile.
LINE_TILE = (-2, 0)

# The audits, tag -> (clusters, parts, nets, drift, unbuilt, refused).
#
# THE TUPLE IS THE ASSERTION AND `unbuilt=0` ALONE WOULD NOT BE: a cluster with
# no inputs or no outputs is a legitimate half-built state and is never counted,
# so a rig that lost half its belts reads `unbuilt=0` while delivering nothing.
#
# `armed` is the world as the rigs build it: eleven clusters over 168 parts, ten
# of them with a network -- the lone part at band (c) has no belts at all, and a
# cluster with no edges gets none.
#
# The last nine carry `drift=1 refused=1`, and that is the full-inventory
# negative's own residue: the sixty-fifth belt is still standing, unconnected,
# because the mod could not hand it back. A refused cluster is a STABLE state --
# it still has its network and knows its edge list has moved past what the mod
# can build -- and it stays that way for the rest of the run.
#
# THE WHOLE TABLE MOVED BY +2 CLUSTERS AND +12 PARTS when bands (h) and (i)
# arrived, which is the two curve rigs: four working parts and two spare ones
# each. Nothing else in it moved, which is what says those bands are forty tiles
# clear of every gesture above them.
AUDIT_EXPECT = {
    "armed":            (11, 168, 10, 0, 0, 0),
    "post-restore":     (11, 168, 10, 0, 0, 0),
    "post-end":         (11, 169, 10, 0, 0, 0),
    "post-lone":        (10, 168, 10, 0, 0, 0),
    "post-col":         (11, 167, 11, 0, 0, 0),
    "post-second":      (11, 167, 11, 0, 0, 0),
    "post-brdg":        (11, 167, 11, 0, 0, 0),
    "post-lim":         (11, 167, 11, 0, 0, 0),
    "post-limfull":     (11, 167, 11, 1, 0, 1),
    "post-bmin-add":    (11, 167, 11, 1, 0, 1),
    "post-curve-line":  (11, 167, 11, 1, 0, 1),
    "post-curve-face":  (11, 167, 11, 1, 0, 1),
    "post-curve":       (11, 167, 11, 1, 0, 1),
    "post-lb":          (11, 167, 11, 1, 0, 1),
    "post-bmin-mine":   (11, 167, 11, 1, 0, 1),
    "post-pock":        (10, 162, 10, 1, 0, 1),
    "final":            (10, 162, 10, 1, 0, 1),
}

# What a SATURATED express belt tile holds: four items per lane at the belt's own
# 0.25-tile item pitch. It is the cargo gesture (a) takes off the belt it
# destroys and the number the player must end up holding.
SATURATED = 8


def parse_sample(text):
    """`k=v k=v` into a dict, with ints where the value is one."""
    out = {}
    for chunk in text.split():
        name, _, val = chunk.partition("=")
        try:
            out[name] = int(val)
        except ValueError:
            out[name] = val
    return out


def fmt(s):
    return ("tile=%s dir=%s items=%s box=%s ground=%s cursor=%s plate=%s "
            "belt=%s part=%s" % (s.get("tile"), s.get("dir"), s.get("tile_items"),
                                 s.get("box"), s.get("ground"), s.get("cursor"),
                                 s.get(PLATE), s.get(BELT), s.get(PART)))


def main():
    lines = []
    for path in sys.argv[1:]:
        with open(path, errors="replace") as f:
            lines.extend(f)

    fail = []
    samples, order = {}, []
    for line in lines:
        m = SAMPLE.search(line)
        if m:
            samples[m.group(1)] = parse_sample(m.group(2))
            order.append(m.group(1))

    def need(tag):
        s = samples.get(tag)
        if s is None:
            fail.append("no sample for tag=%s: the observer did not reach that "
                        "gesture, and every assertion about it below is vacuous"
                        % tag)
        return s

    # A window is the slice of the log between two observer tags, which is how a
    # mod line is attributed to the gesture that caused it.
    def window_span(start, end):
        i = j = None
        for n, line in enumerate(lines):
            if i is None and start in line:
                i = n
            elif i is not None and end in line:
                j = n
                break
        if i is None or j is None:
            fail.append("no window from %r to %r" % (start, end))
            return None
        return i, j

    def window(start, end):
        span = window_span(start, end)
        return [] if span is None else lines[span[0]:span[1] + 1]

    def find(rx, where):
        return [m.groups() for m in (rx.search(l) for l in where) if m]

    # ---- the player, which is the whole suite's anti-vacuity ---------------
    players = {m.group(1): (m.group(2), m.group(3)) for m in
               (PLAYER.search(l) for l in lines) if m}
    for tag in ("init", "armed", "final"):
        got = players.get(tag)
        print("  player %-6s %s" % (tag, (got[0] + got[1]) if got else "MISSING"))
        if not got or got[0] != "true":
            fail.append("no player at tag=%s. This suite drives a CURSOR: with "
                        "no player every gesture below does nothing at all and "
                        "every sample it takes is of a world nobody touched. "
                        "The fixture under test/fixtures-player/ is what carries "
                        "one; `make player-fixture SAVE=<a client save>` cuts a "
                        "new one." % tag)
    armed = players.get("armed", ("", ""))[1]
    if "controller=%d" % GOD not in armed:
        fail.append("the player is not in the god controller at tag=armed:%s. "
                    "A disconnected player cannot hold a character -- the engine "
                    "refuses `player.character =` by name -- and in the character "
                    "controller `build_from_cursor` places nothing." % armed)
    if "surface=bbb-curs" not in armed:
        fail.append("the player was not teleported onto this suite's own "
                    "surface:%s" % armed)

    # ---- the audits --------------------------------------------------------
    #
    # The observer's `audited tag=` line is written AFTER the marker it placed,
    # so the mod's own audit line for that tag is the last one before it.
    audited = [m.group(1) for m in (AUDITED.search(l) for l in lines) if m]
    tagged, last = {}, None
    for line in lines:
        m = AUDIT.search(line)
        if m:
            last = tuple(int(v) for v in m.groups())
            continue
        m = AUDITED.search(line)
        if m and last:
            tagged[m.group(1)] = last
    for tag in AUDIT_EXPECT:
        want, got = AUDIT_EXPECT[tag], tagged.get(tag)
        print("  audit %-15s %s" % (tag, got))
        if got is None:
            fail.append("no audit for tag=%s" % tag)
        elif got != want:
            fail.append("audit %s is %s and the rigs make %s" % (tag, got, want))
    if len(audited) != len(AUDIT_EXPECT):
        fail.append("the run took %d audits and this suite schedules %d"
                    % (len(audited), len(AUDIT_EXPECT)))

    # ---- (a) the restore ---------------------------------------------------
    placed = find(LOADED, lines)
    if not placed or int(placed[0][0]) != SATURATED:
        fail.append("the belt at band (a) was loaded with %s items and a "
                    "saturated express belt holds %d -- the gesture's whole "
                    "cargo number comes from this"
                    % (placed[0][0] if placed else "no", SATURATED))

    pre, click = need("restore-pre"), need("restore-click")
    if pre and click:
        print("  restore pre  %s" % fmt(pre))
        print("  restore post %s" % fmt(click))
        if pre["tile"] != BELT or pre["dir"] != WEST:
            fail.append("band (a)'s target is %s facing %s and the rig lays a "
                        "west-facing %s there" % (pre["tile"], pre["dir"], BELT))
        if pre["tile_items"] != SATURATED:
            fail.append("band (a)'s belt carries %d items at the moment of the "
                        "click and a saturated one holds %d"
                        % (pre["tile_items"], SATURATED))
        if click["tile"] != PART:
            fail.append("the part did not land on band (a)'s belt: the tile "
                        "holds %s. `build_from_cursor` raises nothing at all "
                        "when the build cannot happen, so this is what says the "
                        "gesture reached the engine" % click["tile"])
        if click[PLATE] - pre[PLATE] != SATURATED:
            fail.append("the replaced belt's cargo did not reach the player: "
                        "%d plates against %d before, and the belt carried %d"
                        % (click[PLATE], pre[PLATE], SATURATED))

    win = window("gesture begin name=restore", "audited tag=post-restore")
    refusals, handed = find(SEDGE_REFUSED, win), find(HANDEDBACK, win)
    restored = find(RESTORED, win)
    print("  restore refusals=%d handed=%d restored=%d"
          % (len(refusals), len(handed), len(restored)))
    if len(refusals) != 1:
        fail.append("band (a) produced %d one-belt-per-part refusals and the "
                    "gesture makes exactly one" % len(refusals))
    elif refusals[0][2] != "2":
        fail.append("band (a)'s refusal reports worst=%s and a part between two "
                    "belts of one line carries two" % refusals[0][2])
    if len(handed) != 1:
        fail.append("the refused part was handed back %d times and the gesture "
                    "makes exactly one" % len(handed))
    elif (int(handed[0][0]), int(handed[0][1])) != LINE_TILE:
        fail.append("the hand-back names %s,%s and band (a)'s target is %d,%d"
                    % (handed[0][0], handed[0][1], LINE_TILE[0], LINE_TILE[1]))

    # THE RED PROOF, and it is the reason this suite was written before the fix
    # was: on the guest that shipped through 0.3.3's first cut the four
    # assertions below fail together -- no restore line, the tile empty forever,
    # and the player one belt richer than they started.
    if len(restored) != 1:
        fail.append(
            "the belt the fast replace destroyed was not put back (%d restore "
            "lines). A player clicked a part onto a belt, the engine mined that "
            "belt as part of the gesture, and the mod refused the part -- so "
            "without this the tile is left EMPTY and the player keeps a belt "
            "item they did not ask for." % len(restored))
    elif (int(restored[0][0]), int(restored[0][1])) != LINE_TILE:
        fail.append("the restore names %s,%s and band (a)'s target is %d,%d"
                    % (restored[0][0], restored[0][1],
                       LINE_TILE[0], LINE_TILE[1]))
    for rx, what in ((RESTORE_UNPAID, "the player could not pay for it"),
                     (RESTORE_FAILED, "the belt could not be created")):
        if find(rx, win):
            fail.append("band (a)'s restore took its failure arm (%s), and the "
                        "rig gives the player belts and leaves the tile clear"
                        % what)

    for tag in ("restore-1", "restore-2", "restore-8"):
        s = need(tag)
        if not s:
            continue
        print("  %-10s %s" % (tag, fmt(s)))
        if s["tile"] != BELT:
            fail.append("%s: the tile holds %s and the belt the gesture "
                        "destroyed has to be back on it" % (tag, s["tile"]))
        else:
            if s["dir"] != WEST or s["force"] != "player":
                fail.append("%s: the restored belt faces %s for force %s, and "
                            "the one it replaced faced %d for player"
                            % (tag, s["dir"], s["force"], WEST))
            if s["tile_items"] != 0:
                fail.append("%s: the restored belt carries %d items. Its cargo "
                            "went to the player with the mine; a belt that came "
                            "back holding it would be this mod minting matter"
                            % (tag, s["tile_items"]))
        if pre and s[BELT] != pre[BELT]:
            fail.append("%s: the player holds %d %s and held %d before the "
                        "click. The engine refunded one when it mined the belt "
                        "and the restore charges one back, so the count does "
                        "not move." % (tag, s[BELT], BELT, pre[BELT]))
        if pre and s[PLATE] != pre[PLATE] + SATURATED:
            fail.append("%s: the player holds %d plates and the belt's cargo "
                        "was %d" % (tag, s[PLATE], SATURATED))

    # ---- (b) the end of a line --------------------------------------------
    pre, click, after = need("end-pre"), need("end-click"), need("end-1")
    if pre and click and after:
        print("  end pre  %s" % fmt(pre))
        print("  end post %s" % fmt(after))
        if pre["tile"] != BELT:
            fail.append("band (b)'s target holds %s and the rig ends a belt "
                        "line there" % pre["tile"])
        if click["tile"] != PART or after["tile"] != PART:
            fail.append("band (b)'s part did not land and stay: %s then %s. "
                        "This is the ACCEPTED arm -- the belt behind it is an "
                        "input and there is nothing ahead, so the part carries "
                        "one belt" % (click["tile"], after["tile"]))
        if click[PLATE] - pre[PLATE] != pre["tile_items"]:
            fail.append("band (b): the replaced belt carried %d items and the "
                        "player gained %d"
                        % (pre["tile_items"], click[PLATE] - pre[PLATE]))
        if click[BELT] - pre[BELT] != 1:
            fail.append("band (b): the engine refunds the belt it mined, so the "
                        "player's belt count rises by one; it moved by %d"
                        % (click[BELT] - pre[BELT]))
    win = window("gesture begin name=end-of-line", "audited tag=post-end")
    shapes = [(g[1], g[2], g[3]) for g in find(COMPILED, win)]
    print("  end compiled %s" % shapes)
    if ("3", "2", "4") not in shapes:
        fail.append("band (b) did not become a 3->2 over four ports: compiled "
                    "%s. A part on the end of that line takes the belt behind "
                    "it as a third input" % shapes)
    if find(SEDGE_REFUSED, win) or find(OVERLIMIT, win):
        fail.append("band (b) was refused, and its part carries one belt")

    rates = {m.group(1): int(m.group(2)) for m in
             (RATE.search(l) for l in lines) if m}
    print("  rates %s" % rates)
    if rates.get("end-a", 0) <= 0:
        fail.append("the 3->2 band (b) became delivered nothing: a machine that "
                    "stopped satisfies every count above")

    # ---- (c) a belt over a part -------------------------------------------
    for name, begin, end in (("lone", "lone", "audited tag=post-lone"),
                             ("col", "column", "audited tag=post-col")):
        pre, after = need(name + "-pre"), need(name + "-1")
        if not (pre and after):
            continue
        print("  %-4s pre  %s" % (name, fmt(pre)))
        print("  %-4s post %s" % (name, fmt(after)))
        if pre["tile"] != PART:
            fail.append("band (c)/%s's target holds %s and the rig puts a part "
                        "there" % (name, pre["tile"]))
        if after["tile"] != BELT or after["dir"] != SOUTH:
            fail.append("band (c)/%s: the belt did not replace the part -- the "
                        "tile holds %s facing %s"
                        % (name, after["tile"], after["dir"]))
        if after[PART] - pre[PART] != 1:
            fail.append("band (c)/%s: the replaced part did not reach the "
                        "player (%d held, %d before)"
                        % (name, after[PART], pre[PART]))
        w = window("gesture begin name=" + begin, end)
        if name == "lone" and not find(DISSOLVED, w):
            fail.append("band (c)/lone: no `dissolved, mined by player` line. "
                        "The engine raises the ordinary mine pair for a part a "
                        "PLAYER fast-replaces -- only a scripted "
                        "`create_entity{fast_replace}` raises nothing -- so the "
                        "removal reaches the registry through it")

    # ---- (g) a second belt -------------------------------------------------
    pre, click, after = need("second-pre"), need("second-click"), need("second-1")
    if pre and click and after:
        print("  second pre  %s" % fmt(pre))
        print("  second post %s" % fmt(after))
        if pre["tile"] != "empty":
            fail.append("band (g)'s target is not empty: %s" % pre["tile"])
        if click["tile"] != BELT:
            fail.append("band (g)'s belt did not land: %s" % click["tile"])
        if after["tile"] != "empty":
            fail.append("band (g): the refused belt is still standing (%s). A "
                        "second belt on a part that has one is refused and "
                        "handed back" % after["tile"])
        if after[BELT] != pre[BELT] or after["cursor"] != pre["cursor"]:
            fail.append("band (g): the player is not where they started -- "
                        "%d/%s against %d/%s. The hand-back lands in the CURSOR "
                        "whenever it holds the same item"
                        % (after[BELT], after["cursor"], pre[BELT], pre["cursor"]))
    win = window("gesture begin name=second-belt", "audited tag=post-second")
    if len(find(SEDGE_REFUSED, win)) != 1:
        fail.append("band (g) produced %d one-belt-per-part refusals and the "
                    "gesture makes exactly one" % len(find(SEDGE_REFUSED, win)))
    if len(find(HANDEDBACK, win)) != 1:
        fail.append("band (g)'s belt was handed back %d times"
                    % len(find(HANDEDBACK, win)))

    # ---- (f) the bridge ----------------------------------------------------
    pre, click, after = need("brdg-pre"), need("brdg-click"), need("brdg-1")
    if pre and click and after:
        print("  brdg pre  %s" % fmt(pre))
        print("  brdg post %s" % fmt(after))
        if click["tile"] != PART:
            fail.append("band (f)'s part did not land: %s" % click["tile"])
        if after["tile"] != "empty":
            fail.append("band (f): the bridging part is still standing (%s)"
                        % after["tile"])
        if after[PART] != pre[PART]:
            fail.append("band (f): the player holds %d parts and held %d -- a "
                        "refused merge costs them nothing"
                        % (after[PART], pre[PART]))
    win = window("gesture begin name=bridge", "audited tag=post-brdg")
    over = find(OVERLIMIT, win)
    print("  brdg over-limit %s" % over)
    if len(over) != 1:
        fail.append("band (f) produced %d over-limit refusals and the merge "
                    "makes exactly one" % len(over))
    if find(TORNDOWN, win):
        fail.append("band (f) tore something down: %s. The merge's teardowns "
                    "belong to AddPart and are queued before the compiler sees "
                    "the cluster they make, so the pre-pass takes them back off "
                    "the queue -- a refused merge must cost NOTHING"
                    % find(TORNDOWN, win))
    if rates.get("brdg-b", 0) <= rates.get("brdg-a", -1):
        fail.append("band (f)'s two halves stopped delivering across the "
                    "refused merge: %s then %s"
                    % (rates.get("brdg-a"), rates.get("brdg-b")))

    # ---- (e) the sixty-fifth belt -----------------------------------------
    pre, click, after = need("lim-pre"), need("lim-click"), need("lim-1")
    if pre and click and after:
        print("  lim pre  %s" % fmt(pre))
        print("  lim post %s" % fmt(after))
        if click["tile"] != BELT:
            fail.append("band (e)'s belt did not land: %s" % click["tile"])
        if after["tile"] != "empty":
            fail.append("band (e): the sixty-fifth belt is still standing (%s)"
                        % after["tile"])
        if after[BELT] != pre[BELT] or after["cursor"] != pre["cursor"]:
            fail.append("band (e): the player is not where they started -- "
                        "%d/%s against %d/%s"
                        % (after[BELT], after["cursor"], pre[BELT], pre["cursor"]))
    win = window("gesture begin name=lim", "audited tag=post-lim")
    over = find(OVERLIMIT, win)
    print("  lim over-limit %s" % over)
    if len(over) != 1 or over[0][2] != "65" or over[0][4] != "64":
        fail.append("band (e)'s refusal is %s and the rig lays a sixty-fifth "
                    "input on a balancer at the limit of 64" % over)
    if len(find(HANDEDBACK, win)) != 1:
        fail.append("band (e)'s belt was handed back %d times"
                    % len(find(HANDEDBACK, win)))
    if rates.get("lim-b", 0) <= rates.get("lim-a", -1):
        fail.append("band (e)'s balancer stopped delivering across the refusal: "
                    "%s then %s. The refusal happens BEFORE the teardown, so "
                    "the standing network is untouched"
                    % (rates.get("lim-a"), rates.get("lim-b")))

    # ---- (e) ... with nowhere to put it back ------------------------------
    filled = find(FILLED, lines)
    print("  limfull filler %s" % (str(filled[0]) if filled else None))
    if not filled or filled[0][1] != "0":
        fail.append("the inventory was not filled before the full-inventory "
                    "negative (%s): with room in it the piece comes back and "
                    "the gesture measures the ordinary hand-back a second time"
                    % (str(filled[0]) if filled else "no filler line"))
    after = need("limfull-1")
    if after:
        print("  limfull post %s" % fmt(after))
        if after["tile"] != BELT:
            fail.append("the full-inventory negative: the refused belt is not "
                        "standing (%s). `revertOne` mines with force = nil, "
                        "which is vanilla's rule -- a player whose inventory "
                        "cannot take it does not get to mine it" % after["tile"])
    win = window("gesture begin name=lim-full", "audited tag=post-limfull")
    noroom = find(NOROOM, win)
    print("  limfull no-room %s" % noroom)
    if len(noroom) != 1:
        fail.append("the full-inventory negative produced %d `could not be "
                    "handed back` alerts and it makes exactly one" % len(noroom))
    if find(HANDEDBACK, win):
        fail.append("the full-inventory negative handed the piece back after "
                    "all: %s" % find(HANDEDBACK, win))

    # ---- (d) the miner's pocket -------------------------------------------
    mined = find(BMINMINE, lines)
    if not mined or mined[0][0] != "true":
        fail.append("band (d2)'s belt was not mined by the player")
    win = window("gesture begin name=bmin-mine", "audited tag=post-bmin-mine")
    offered = [int(g[1]) for g in find(OFFERED, win)]
    pocketed = [int(g[2]) for g in find(POCKETED, win)]
    print("  bmin offered=%s pocketed=%s" % (offered, pocketed))
    if offered != pocketed or len(offered) != 1:
        fail.append("band (d2) offered %s and pocketed %s: the port boundary "
                    "this rig crosses (P 4 -> 2) is the second field report, "
                    "and what will not fit in the halved machine goes to the "
                    "player who mined it" % (offered, pocketed))
    # THE BASELINE IS THE MINE'S OWN SAMPLE AND NOT THE ONE BEFORE THE GESTURE.
    # `mine_entity` hands the player the mined belt's OWN CARGO inside the
    # dispatch -- eight items off a saturated express belt -- and the pocket
    # arrives a tick later out of the deferred flush. Measuring from before the
    # click counts both and reads a correct pocket as eight too many.
    mid, after = need("bmin-mine"), need("bmin-1")
    if mid and after and pocketed:
        if after[PLATE] - mid[PLATE] != pocketed[0]:
            fail.append("band (d2): the player gained %d plates across the "
                        "flush and the mod says it pocketed %d"
                        % (after[PLATE] - mid[PLATE], pocketed[0]))

    steps = find(POCKMINE, lines)
    print("  pock steps=%d" % len(steps))
    if len(steps) != 5 or any(s[3] != "true" for s in steps):
        fail.append("band (d1) mined %d parts and the rig is five: %s"
                    % (len(steps), steps))
    win = window("sample tag=pock-pre", "audited tag=post-pock")
    offered = [int(g[1]) for g in find(OFFERED, win)]
    pocketed = [int(g[2]) for g in find(POCKETED, win)]
    print("  pock offered=%s pocketed=%s" % (offered, pocketed))
    if offered != pocketed:
        fail.append("band (d1) offered %s and pocketed %s" % (offered, pocketed))
    if sum(offered) <= 0:
        fail.append("band (d1) pocketed nothing at all. THAT IS THE FIRST FIELD "
                    "REPORT: the pocket shipped crediting the miner on the "
                    "DISSOLVE alone, by which time a balancer taken apart part "
                    "by part is almost empty. A rig that was not saturated "
                    "satisfies every other count here")
    pre, after = need("pock-pre"), need("pock-5")
    if pre and after:
        if after[PLATE] - pre[PLATE] != sum(offered):
            fail.append("band (d1): the player gained %d plates over the "
                        "teardown and the mod says it offered %d"
                        % (after[PLATE] - pre[PLATE], sum(offered)))
        if after[PART] - pre[PART] != 5:
            fail.append("band (d1): the player holds %d more parts than they "
                        "did and mined five" % (after[PART] - pre[PART]))

    # ---- (h) the curved exit ----------------------------------------------
    #
    # The mod portal report is "you have to be very careful not to place a
    # straight belt adjacent to the balancer, as it immediately curves it", and
    # these three gestures are what that is: the FIRST belt of a line has nothing
    # behind it, which is the same world state as a deliberate corner, and the
    # rule cannot tell them apart because there is nothing to tell apart.
    curves, corder = {}, []
    for line in lines:
        m = CURVE.search(line)
        if m:
            curves[m.group(1)] = parse_sample(m.group(2))
            corder.append(m.group(1))

    def curve(tag):
        c = curves.get(tag)
        if c is None:
            fail.append("no curve sample for tag=%s" % tag)
        return c

    def grabbed(c):
        return c["iface_w"] == "true" or c["iface_e"] == "true"

    # (h2) the line laid from three tiles out, one belt per click. It passes two
    # free faces and is never taken, because every belt that reaches one has the
    # belt behind it in its rear already.
    laid = [t for t in corder if t.startswith("curve-line")]
    print("  curve line tags=%d" % len(laid))
    if len(laid) != 8:
        fail.append("band (h2) took %d samples and the line is seven clicks plus "
                    "a settled read" % len(laid))
    for tag in laid:
        c = curves[tag]
        if grabbed(c):
            fail.append("band (h2): the line was grabbed at %s (%s). A belt laid "
                        "past a balancer with a belt behind it is a SIDE-LOAD, "
                        "which this rule declines on purpose -- half a lane on a "
                        "port backs the butterfly up" % (tag, c))
        if c["line"] != 0:
            fail.append("band (h2): %d items reached the passing line at %s. "
                        "Nothing of the machine's may, and the rig is dead-ended "
                        "so anything there came out of it" % (c["line"], tag))
    win = window("gesture begin name=curve-line", "gesture end name=curve-line")
    if find(COMPILED, win):
        fail.append("band (h2) recompiled the balancer: %s. Seven belts laid past "
                    "it move no edge at all" % find(COMPILED, win))

    # (h1) ONE belt on a free face, which is the report.
    face = curve("curve-face-2")
    if face:
        print("  curve face %s" % face)
        if face["iface_w"] != "true":
            fail.append("band (h1): the free face was NOT classified (%s). A belt "
                        "clicked onto it has an empty rear by construction -- it "
                        "is the head of a line that does not exist yet -- so the "
                        "engine curves it and this mod gives it a port" % face)
        if face["shape"] != "right":
            fail.append("band (h1): the engine reads the face belt as %r and it "
                        "curves towards the interface below it, which is `right`"
                        % face["shape"])
    win = window("gesture begin name=curve-face", "audited tag=post-curve-face")
    got = find(COMPILED, win)
    print("  curve face compiled %s" % [(g[1], g[2], g[3]) for g in got])
    if len(got) != 1 or (got[0][1], got[0][2], got[0][3]) != ("2", "3", "4"):
        fail.append("band (h1) compiled %s and one belt on a free face takes a "
                    "2->2 over two ports to a 2->3 over four"
                    % [(g[1], g[2], g[3]) for g in got])

    # ... and what it costs the player, which is the half the report noticed. The
    # rig is dead-ended, so anything on that line came out of the machine through
    # a port the player did not mean to open. The assertion is a FLOOR: the count
    # is a rate over the ticks the port stood open, and 7 is what this schedule
    # produced.
    filled = curve("curve-face-filled")
    if filled:
        print("  curve face filled line=%s" % filled["line"])
        if filled["line"] < 1:
            fail.append("band (h1) pushed nothing onto the player's line, so the "
                        "grab cost nothing visible and the gesture proves less "
                        "than it looks like it does")

    # (h3) the remedy: give that belt a belt behind it and the port goes away.
    rel = curve("curve-rel-2")
    if rel:
        print("  curve release %s" % rel)
        if grabbed(rel):
            fail.append("band (h3): the port is still there after a belt was laid "
                        "behind the face belt (%s). A fed rear is a side-load and "
                        "the rule declines one" % rel)
        if rel["shape"] != "straight":
            fail.append("band (h3): the engine still reads the face belt as %r "
                        "with a belt feeding its rear" % rel["shape"])
    relwin = window_span("gesture begin name=curve-release", "audited tag=post-curve")
    win = [] if relwin is None else lines[relwin[0]:relwin[1] + 1]
    got = find(COMPILED, win)
    print("  curve release compiled %s" % [(g[1], g[2], g[3]) for g in got])
    if len(got) != 1 or (got[0][1], got[0][2], got[0][3]) != ("2", "2", "2"):
        fail.append("band (h3) compiled %s and releasing the port takes the "
                    "machine back to a 2->2 over two ports"
                    % [(g[1], g[2], g[3]) for g in got])
    spilled = [int(g[0]) for g in find(SPILLED, win)]
    print("  curve release spilled %s" % spilled)
    if len(spilled) != 1 or spilled[0] < 1:
        fail.append("band (h3) spilled %s. Releasing a port HALVES the butterfly "
                    "-- P goes 4 -> 2 -- so a machine that was full has more than "
                    "the successor can hold, and what will not fit reaches the "
                    "ground. That is the ordinary teardown policy and it is the "
                    "price of every release in this band" % spilled)

    # ---- (i) a linked belt at the rear ------------------------------------
    #
    # `feedsTile` walked the six EDGE types, and linked-belt is deliberately not
    # one of them: every visible interface this mod places is a linked belt on a
    # part tile, and one in the edge query would make a cluster's own output an
    # edge of itself. The probe inherited the omission, so a player-placeable
    # linked belt from another mod was a feeder the engine saw and the mod did
    # not -- and the belt got a port that side-loads half a lane.
    for tag in ("lb-pre", "lb-click", "lb-2", "lb-10"):
        c = curve(tag)
        if not c:
            continue
        if grabbed(c):
            fail.append("band (i): the face belt was grabbed at %s (%s) with a "
                        "linked belt OUTPUT end feeding its rear. Revert the "
                        "seventh type out of `probeTypes` and this is what comes "
                        "back" % (tag, c))
    click = curve("lb-click")
    if click and click["tile"] != BELT:
        fail.append("band (i): the click did not land -- the face tile holds %s. "
                    "Every `iface=false` above is vacuous without a belt there"
                    % click["tile"])
    win = window("gesture begin name=linked-rear", "audited tag=post-lb")
    if find(COMPILED, win):
        fail.append("band (i) recompiled the balancer: %s" % find(COMPILED, win))

    # The engine's own reading, with no balancer within ten tiles. This is what
    # says the rig is really curve-eligible and the linked belt is really a
    # feeder, rather than the mod having declined for a reason of its own.
    eng = {m.group(1): (m.group(3), m.group(4))
           for m in (ENGINE.search(l) for l in lines) if m}
    print("  engine control %s" % eng)
    if eng.get("empty-rear", (None,))[0] != "right":
        fail.append("the engine control with an EMPTY rear reads %s and a belt "
                    "with one perpendicular feeder and nothing behind it curves "
                    "towards that feeder. Without this row band (i)'s rig might "
                    "simply not be curve-eligible and its every `false` would "
                    "mean nothing" % (eng.get("empty-rear"),))
    if eng.get("linked-rear", (None,))[0] != "straight":
        fail.append("the engine control with a linked belt output end at the rear "
                    "reads %s and the engine counts one as a feeder, which is why "
                    "the probe must see it too" % (eng.get("linked-rear"),))

    # ---- the whole run -----------------------------------------------------
    #
    # NOTHING REACHES THE FLOOR IN THIS SUITE, and that is a headline rather than
    # a formality. Every removal here was made by a PLAYER, so everything a
    # machine could not take back is offered to that player first -- where the
    # `edge` suite's twin of band (d2) puts 124 of the same 128 items on the
    # ground, because a headless --create has nobody to offer them to.
    ground = {tag: samples[tag]["ground"] for tag in order
              if samples[tag].get("ground")}
    print("  ground non-zero at %s" % (ground or "no tag"))
    if ground:
        fail.append("items reached the ground at %s. Every removal in this "
                    "suite was made by a player, and a player is offered what "
                    "no network could take before the floor" % ground)
    # ... AND THE MOD SPILLS IN EXACTLY ONE WINDOW, which is band (h3)'s release.
    # That one is a machine getting SMALLER -- a port taken off a full balancer,
    # P 4 -> 2 -- which is the ordinary teardown policy and not a gesture anybody
    # is being credited for. Everywhere else a spill is a defect, and the window
    # is named rather than the check weakened: `edge` scopes its own four the
    # same way.
    stray = [(n, m.group(1)) for n, m in
             ((n, SPILLED.search(l)) for n, l in enumerate(lines)) if m
             and not (relwin and relwin[0] <= n <= relwin[1])]
    if stray:
        fail.append("the mod spilled outside band (h3)'s release: %s" % stray)

    for f in fail:
        print("FAIL: " + f)
    if fail:
        sys.exit(1)
    print("the player gestures all behave: the refused fast replace puts the "
          "belt back, every hand-back lands, the pocket is paid, and a belt on a "
          "free face is a port that a belt behind it gives back")


if __name__ == "__main__":
    main()
