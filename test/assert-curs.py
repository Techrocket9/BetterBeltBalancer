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
# `armed` is the world as the rigs build it: nine clusters over 156 parts, eight
# of them with a network -- the lone part at band (c) has no belts at all, and a
# cluster with no edges gets none.
#
# The last five carry `drift=1 refused=1`, and that is the full-inventory
# negative's own residue: the sixty-fifth belt is still standing, unconnected,
# because the mod could not hand it back. A refused cluster is a STABLE state --
# it still has its network and knows its edge list has moved past what the mod
# can build -- and it stays that way for the rest of the run.
AUDIT_EXPECT = {
    "armed":          (9, 156, 8, 0, 0, 0),
    "post-restore":   (9, 156, 8, 0, 0, 0),
    "post-end":       (9, 157, 8, 0, 0, 0),
    "post-lone":      (8, 156, 8, 0, 0, 0),
    "post-col":       (9, 155, 9, 0, 0, 0),
    "post-second":    (9, 155, 9, 0, 0, 0),
    "post-brdg":      (9, 155, 9, 0, 0, 0),
    "post-lim":       (9, 155, 9, 0, 0, 0),
    "post-limfull":   (9, 155, 9, 1, 0, 1),
    "post-bmin-add":  (9, 155, 9, 1, 0, 1),
    "post-bmin-mine": (9, 155, 9, 1, 0, 1),
    "post-pock":      (8, 150, 8, 1, 0, 1),
    "final":          (8, 150, 8, 1, 0, 1),
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
    def window(start, end):
        i = j = None
        for n, line in enumerate(lines):
            if i is None and start in line:
                i = n
            elif i is not None and end in line:
                j = n
                break
        if i is None or j is None:
            fail.append("no window from %r to %r" % (start, end))
            return []
        return lines[i:j + 1]

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
    if find(SPILLED, lines):
        fail.append("the mod spilled: %s" % find(SPILLED, lines))

    for f in fail:
        print("FAIL: " + f)
    if fail:
        sys.exit(1)
    print("the player gestures all behave: the refused fast replace puts the "
          "belt back, every hand-back lands, and the pocket is paid")


if __name__ == "__main__":
    main()
