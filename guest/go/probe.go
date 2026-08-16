package main

// The insert probe: the miner's-pocket arithmetic, asked of a container.
//
// WHY IT EXISTS. carry.go's pocket path hands a drained network's contents to
// the player who mined the machine, and when it landed the whole of it was
// declared UNVERIFIABLE HEADLESSLY: a `--create` has no player,
// `game.get_player(1)` is nil, and `on_player_mined_entity` is not one of the
// events `script.raise_event` will raise. Then a player reported that mining a
// full balancer in the map editor put ONE item in their pocket and the rest on
// the floor, and the first question anybody could ask was "is the count even
// reaching the engine?" -- which nothing in seven suites could answer.
//
// THE WALL WAS IN THE WRONG PLACE. `insert` is a member of LuaControl, and a
// CHEST is a LuaControl, and so is a CHARACTER: `LuaEntity.insert`,
// `LuaPlayer.insert` and `LuaControl.insert` are one member id, one signature
// and one tier-2 encode of one table. So the arithmetic is reproducible in a
// headless run with no player anywhere -- offer a chest a known count through
// the very function the pocket uses, and read back what it holds.
//
// It answered the question in one run: 50 asked, 50 taken, 50 held. The defect
// was not here at all -- it was that only a DISSOLVE recorded a beneficiary, so
// taking a balancer apart by hand spilled it shrink by shrink before the pocket
// was ever consulted. See cluster.go's removePart and carry.go's header. This
// file stays because the question will be asked again, and because a shipped
// `insert` whose count is not pinned by anything is how the question got
// expensive the first time.
//
// WHAT IT PINS, precisely: that `insertOne` puts `count` items into a real
// inventory and reports `count` -- for a plain kind, for a kind with a quality,
// for several kinds in a row through the shared `carryKV` builder, for a name
// read out of the world rather than written down here, against a container and
// against a character, and from inside the DEFERRED FLUSH where the pocket
// actually runs. What it does not pin is the TRIGGER: that a player mining a
// part is what reaches this code needs a player, and stays interactive. See
// CLAUDE.md's table of what M3 implements and does not verify.
//
// IT IS A FIELD DIAGNOSTIC TOO, and that is why it ships rather than living in
// the test harness, exactly as `bbb-audit` does. Placing one on a chest in a
// real save answers "is the boundary handing this engine the count I gave it?"
// in one log line.

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// probeKinds is what the probe offers, in order, through one shared builder.
//
// Three legs, each asking a different question of the same call:
//
//  1. a plain (name, count) with NO quality -- the two-key map, which is what
//     base Factorio's contents always produce;
//  2. a (name, count, quality) -- the three-key map, and the one that proves
//     the optional third key does not displace the second;
//  3. a plain one again, AFTER the three-key one, which is the shared-buffer
//     question: `carryStack` writes into a package-level [3]KeyValue and hands
//     back a slice of it, so a leg that shrinks from three keys to two must not
//     inherit the previous leg's third.
//
// The counts are distinct, none of them 1, and none of them a multiple of
// another: a count that arrives as 1, as a stack size, or as some other leg's
// number is a different wrong answer and the log says which.
var probeKinds = [3]struct {
	name    string
	quality string
	count   uint32
}{
	{"iron-gear-wheel", "", 50},
	{"iron-plate", "normal", 37},
	{"copper-cable", "", 23},
}

// ProbeName is the marker that asks for it. Like the audit marker, only a
// script can place one: no item, no recipe, no technology yields it.
const ProbeName = "bbb-insert-probe"

// probeReq is one pending request: where the marker stood.
//
// IT IS DEFERRED, AND THAT IS THE POINT RATHER THAN A CONVENIENCE. The pocket
// runs from `settleCarry`, which runs from `endCarry`, which runs at the end of
// `flush` -- inside `fk_on_deferred`, a tick after the event that caused it, on
// a fresh marshalling arena and with no event payload underneath it. A probe
// that ran inside its own build event would be testing `insert` from a place
// the pocket never calls it, so it queues a tile and a surface -- the same two
// scalars every other deferred thing in this guest keeps -- and runs where the
// real call runs.
type probeReq struct {
	surf uint32
	pos  fkapi.MapPosition
}

var probeQueue []probeReq

// noteInsertProbe is the event half: a scalar and a position, no host call.
func noteInsertProbe(si uint32, pos fkapi.MapPosition) {
	probeQueue = append(probeQueue, probeReq{surf: si, pos: pos})
	requestFlush()
}

// runInsertProbes is the flush half, called at the end of every flush. It costs
// one branch on an empty slice, which is every flush this mod will ever do in a
// real game.
func runInsertProbes() {
	if len(probeQueue) == 0 {
		return
	}
	for i := range probeQueue {
		insertProbe(probeQueue[i].surf, probeQueue[i].pos)
	}
	probeQueue = probeQueue[:0]
}

var (
	probeCountArg fkapi.Value
	// A CHARACTER counts, and not only a chest. A character is the closest thing
	// a headless run has to a player's pockets: a real LuaControl with a slotted
	// main inventory, a cursor stack and a trash inventory, where a chest is a
	// single flat slot list. The pocket path resolves a LuaPlayer, and the two
	// differ in exactly the ways that could make `insert` behave differently, so
	// the suite pins both.
	probeTypeArg = fkapi.OfArray(fkapi.OfString("container"), fkapi.OfString("character"))
	probeFind    fkapi.EntitySearchFilters
	probePos     fkapi.MapPosition
	probeLimit   = uint32(1)
	probeInit    bool
	probeFound   [1]fkapi.Object
)

// insertProbe runs the three legs against whatever inventory holder is on the
// marker's own tile, and logs one line per leg.
//
// The holder is found by TYPE rather than by name, so a suite may use any chest
// it likes and a player diagnosing a real save may use the one that is already
// there. A marker on a tile with nothing to fill says so and does nothing: this
// is a request, not an assertion, and the assertion lives in the suite.
func insertProbe(si uint32, pos fkapi.MapPosition) {
	surf, ok := surfaceByIndex(si)
	if !ok {
		return
	}
	if !probeInit {
		probeFind.Position = &probePos
		probeFind.Type = &probeTypeArg
		probeFind.Limit = &probeLimit
		probeInit = true
	}
	probePos = pos
	found, err := surf.FindEntitiesFilteredInto(probeFound[:0], probeFind)
	if err != nil || len(found) == 0 {
		logStart("insert probe found nothing to insert into")
		logEnd()
		return
	}
	o := &found[0]
	what, err := (fkapi.LuaEntity{Object: *o}).Type()
	if err != nil {
		what = "?"
	}
	for i := range probeKinds {
		k := &probeKinds[i]
		took := insertOne(*o, k.name, k.quality, k.count)
		probeCountArg = fkapi.OfString(k.name)
		held, err := (fkapi.LuaControl{Object: *o}).GetItemCount(&probeCountArg)
		if err != nil {
			held = 0
		}
		logStart("insert probe ")
		logS(what)
		logS(" ")
		logS(k.name)
		logS(" asked=")
		logU(k.count)
		logS(" took=")
		logU(took)
		logS(" held=")
		logU(held)
		logEnd()
	}
	// The fourth leg, and it is not about the engine: the item name is READ OUT
	// OF THE WORLD rather than written down here, so it is a heap-allocated Go
	// string out of `getStr` and not a pointer into .rodata -- which is what
	// every name in a carry pool is. A chest's own prototype name is also an
	// item name, so this costs one host call and asks the one question the three
	// legs above cannot.
	if nm, err := (fkapi.LuaEntity{Object: *o}).Name(); err == nil {
		took := insertOne(*o, nm, "", 7)
		probeCountArg = fkapi.OfString(nm)
		held, err := (fkapi.LuaControl{Object: *o}).GetItemCount(&probeCountArg)
		if err != nil {
			held = 0
		}
		logStart("insert probe ")
		logS(what)
		logS(" ")
		logS(nm)
		logS(" asked=7 took=")
		logU(took)
		logS(" held=")
		logU(held)
		logEnd()
	}
}
