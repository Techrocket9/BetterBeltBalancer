package main

// THE PLAYER'S SIDE OF INPUT AND OUTPUT PRIORITIES.
//
// A balancer part carries a priority flag (cluster.go's `pprio`), the flag
// decides what the compiler wires, and the compiled network is the whole of the
// feature: a priority port is a splitter with `splitter_output_priority` set on
// it, decided once at compile time. There is no per-tick anything and there must
// never be one. This file is only about how the flag gets SET.
//
// THREE DOORS, ONE FUNCTION. A keybind, which is what a player uses; a remote
// method, because a keypress is a path only a human can reach and this
// repository has said three times what that costs (see commands.go); and a
// settings paste, which is the gesture a player expects to carry a machine's
// configuration to the next machine. All three reach setPartPriority and
// nothing else does.
//
// A TOGGLE IS A RECOMPILE AND IS NOT A REMOVAL, which is the one thing about
// this that could lose a player's items. The flag is in the compile fingerprint
// (compile.go), so flipping it takes the cluster down the path an edge edit
// takes and no other: `compile` sees the fingerprint move, calls
// `teardownForRebuild`, and the pool that teardown opens is claimed by the
// network the same flush builds -- carry.go's first decision, matched by root
// because a cluster that had a network is its own successor. So the items go
// back INSIDE the balancer. Nothing here records a mine claim, and that is
// deliberate: a claim is what makes a removal's leftovers a player's property,
// and a toggle is not a removal -- `noteMinedByPlayer` is never called from this
// file, so `settleCarry` has nobody to offer anything to and the reinsertion is
// the only outcome. The one way anything reaches the ground is the fourth
// decision, a successor that is materially smaller than what came down, and
// that spill is `spillPool`'s: beside the VISIBLE cluster, never on the hidden
// surface, and logged.
//
// AND A SHAPE THAT WOULD NOT FIT IS REFUSED BEFORE THE FLAG MOVES, which is the
// other thing that could cost a player a working balancer. `plan.ShapeEdges` is
// the one pre-teardown check and it is asked here with the flag SPECULATIVELY
// flipped, so a toggle the compiler could not honour leaves the flag, the
// network and the items exactly as they were. Refusing after the flip would
// work too -- the check in front of the teardown would save the network -- and
// would leave the cluster standing refused until the player guessed to toggle
// back, which is a worse answer to the same question.

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/plan"
)

// InputTogglePriority is the custom-input prototype guest/go/data/priority.go
// defines and the name this guest subscribes under. The two modules cannot share
// a constant -- the control guest may not import `guest/go/tune` and the data
// guest may not import `fkapi` -- so it is written twice, which is the shape
// `PartName` already has, and each copy names the other.
const InputTogglePriority = "bbb-toggle-priority"

// The locale keys, in mod-data/locale/en/better-belt-balancer.cfg.
//
// THE TWO REFUSAL KEYS ARE SHAPED LIKE limit.go's PAIR because they go through
// the same `tellRefusal`: one sentence for a player who can be handed the piece
// back and one for a robot or script build that leaves it standing. The third
// is the TOGGLE's own, which has no piece in it at all -- nobody built anything,
// the flag simply did not move.
const (
	msgPrioOn        = "bbb.priority-on"
	msgPrioOff       = "bbb.priority-off"
	msgPrioTooBig    = "bbb.priority-too-big"
	msgPrioTooBigStd = "bbb.priority-too-big-unconnected"
	msgPrioRefused   = "bbb.priority-refused"
)

// What setPartPriority was asked for.
const (
	prioToggle = iota
	prioOff
	prioOn
)

// Its own message buffers rather than limit.go's, and the reason is the COLOUR.
// A refusal is vanilla's cannot-build red because that is what the base game has
// taught everybody a refusal looks like (limit.go, and the 2026-08-05 field
// report behind it); a confirmation must not be, or "Priority on" reads as
// something having gone wrong. limit.go's `limText` is wired to the red once, at
// init, and pointing it somewhere else per call would make two files share a
// mutable struct to save a few static bytes.
var (
	prioMsg  [2]fkapi.Value
	prioPos  fkapi.MapPosition
	prioText fkapi.LuaPlayerCreateLocalFlyingTextArgs
	// The amber the priority badge on the sprite is drawn in
	// (tools/make-graphics.py's PRIO_FILL), so the words and the mark on the
	// machine are the same colour.
	prioColR, prioColG, prioColB float32 = 0.98, 0.73, 0.33
	prioColor                            = fkapi.Color{R: &prioColR, G: &prioColG, B: &prioColB}
)

func init() {
	// SUBSCRIBED BY NAME, because that is how Factorio delivers a keybind: a
	// custom input has no `defines.events` entry at all, so the numeric id says
	// what the payload IS and the name says what to register under
	// (fkapi.SubscribeNamed). A name no prototype in this game has comes back as
	// a status and one log line rather than a mod that will not load.
	//
	// MASKED over all three of CustomInputEvent's maskable fields. This handler
	// reads `input_name` and `player_index` and nothing else -- the cursor
	// position is not where the part is (the player's mouse is over it, which is
	// what `player.selected` answers), and the selected-prototype struct and the
	// GUI element handle are two tier-2 values encoded on every keypress for
	// nothing.
	fkapi.SubscribeNamedMasked(fkapi.EventCustomInputEvent,
		fkapi.SkipCustomInputEventCursorDirection|
			fkapi.SkipCustomInputEventSelectedPrototype|
			fkapi.SkipCustomInputEventElement, InputTogglePriority)

	// A SETTINGS PASTE, which is shift-click copy between two entities.
	//
	// NO FILTER EXISTS FOR IT -- `runtime-api.json` gives a `filter` concept to
	// 30 events and this is not one of them, on either pin -- so it arrives for
	// every paste anywhere on the map and the handler's first act is one
	// `name_is` on the destination. That is the same shape the two rotation
	// events already have, and it is a keypress rather than a tick.
	fkapi.Subscribe(fkapi.EventOnEntitySettingsPasted)

	prioMsg[0] = fkapi.OfString(msgPrioOn)
	prioMsg[1] = fkapi.OfNumber(0)
	prioText.Position = &prioPos
	prioText.Color = &prioColor
}

// onTogglePriority is the keybind.
//
// It asks the player what they are POINTING AT rather than taking the cursor
// position out of the payload, because those are different questions: the
// payload's `cursor_position` is where the mouse is in the world, and
// `player.selected` is the entity the engine has decided is under it, which is
// the one the player sees highlighted. Six host calls on a keypress and none on
// any other path.
func onTogglePriority(e *fkapi.CustomInputEvent) {
	if e.InputName != InputTogglePriority || e.PlayerIndex == 0 {
		return
	}
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(e.PlayerIndex)))
	if err != nil || o == nil {
		return
	}
	p := fkapi.LuaPlayer{Object: *o}
	sel, err := p.Selected()
	if err != nil || sel == nil {
		// Nothing under the cursor. Vanilla says nothing when a key does not
		// apply and neither does this.
		return
	}
	ent := fkapi.LuaEntity{Object: *sel}
	// `name_is` rather than `name`: the comparison happens on the host and no
	// string is copied into the guest heap, which under -gc=leaking is the
	// difference between a keypress costing nothing and costing 32 bytes
	// forever. It is also the answer to "is this one of ours" for every
	// prototype at once.
	if ours, err := ent.NameIs(PartName); err != nil || !ours {
		return
	}
	k, ok := tileOfEntity(ent)
	if !ok {
		return
	}
	id, ok := index[k]
	if !ok {
		// A part the world has and the registry does not. Nothing here can
		// repair that -- `/bbb-audit` is what re-derives the registry from the
		// world -- and flagging a tile nobody is tracking would do nothing.
		return
	}
	// ANOTHER FORCE'S BALANCER IS NOT YOURS TO CONFIGURE. Clusters are per force
	// and two forces' parts touch all the time, so a player standing in front of
	// somebody else's machine can hover a part of it. `pforce` is registry state
	// and the player's own index is one host call.
	if fi, err := p.ForceIndex(); err != nil || fi != pforce[id] {
		return
	}
	setPartPriority(k, prioToggle, e.PlayerIndex)
}

// onSettingsPasted carries the flag from one part to another.
//
// THE FLAG COMES OUT OF THE REGISTRY AND NOT OUT OF THE PASTED VARIATION, even
// though the engine writes that variation itself: measured on 2.0.77, a script
// `copy_settings` between two of these entities moves `graphics_variation` from
// the source to the destination, so the picture arrives whether this handler
// exists or not. Reading the flag back out of it would work today and would rest
// on an engine behaviour nobody promised; the registry is the authority
// everywhere else in this guest and is the authority here.
//
// `pvar` IS CLEARED FOR THE DESTINATION, which is the other half. The engine has
// written a variation behind the guest's back, so what the guest remembers being
// drawn there is wrong -- and if the flag happened not to change, restyle would
// compare its own answer against that stale memory, find them equal, and leave
// the source's SHAPE on a part whose neighbourhood is not the source's. Zeroing
// it means restyle looks, which is what "we have not set one" has always meant.
func onSettingsPasted(p uint32) {
	e := fkapi.ReadOnEntitySettingsPasted(p)
	dst := fkapi.LuaEntity{Object: e.Destination}
	if ours, err := dst.NameIs(PartName); err != nil || !ours {
		return
	}
	dk, ok := tileOfEntity(dst)
	if !ok {
		return
	}
	did, ok := index[dk]
	if !ok {
		return
	}
	// The source's tile, and no second `name_is` for it: the registry holds
	// nothing but balancer parts, so a tile that is in it IS one.
	sk, ok := tileOfEntity(fkapi.LuaEntity{Object: e.Source})
	if !ok {
		return
	}
	sid, ok := index[sk]
	if !ok || pforce[sid] != pforce[did] {
		return
	}
	pvar[did] = 0
	if pprio[sid] == pprio[did] {
		// Same flag, and the picture is repaired by the flush the paste does not
		// otherwise need. Queue it anyway, because the variation the engine just
		// wrote is the source's shape.
		markLive(find(did))
		requestFlush()
		return
	}
	on := prioOff
	if pprio[sid] != 0 {
		on = prioOn
	}
	setPartPriority(dk, on, e.PlayerIndex)
}

// tileOfEntity is the tile and surface an entity stands on, in the registry's
// own key. Two host calls and no allocation; `surface_index` rather than
// `surface` so that no object crosses to be indexed on the other side.
func tileOfEntity(e fkapi.LuaEntity) (key, bool) {
	pos, err := e.Position()
	if err != nil {
		return key{}, false
	}
	si, err := e.SurfaceIndex()
	if err != nil {
		return key{}, false
	}
	return key{si, int32(floorF(pos.X)), int32(floorF(pos.Y))}, true
}

// floorF is math.Floor without importing math, which for one call would link
// more of the runtime than the arithmetic is worth. A part's centre is always
// tile + 0.5 and always positive-or-negative, so the truncation has to round
// towards minus infinity rather than towards zero.
func floorF(v float64) float64 {
	t := float64(int64(v))
	if v < 0 && t != v {
		t--
	}
	return t
}

// setPartPriority is the one place the flag moves.
//
// `player` is the player to tell, or 0 for nobody -- a remote caller, or a
// player who has since left. Returns whether the flag now reads what was asked,
// which is what the remote method hands back.
//
// IT RUNS FROM AN EVENT OR A REMOTE DISPATCH AND NEVER FROM INSIDE A FLUSH, and
// that is what makes the classification below legal: `collectCluster` and
// `classifyEdges` write the flush's own `tileBuf` and `edgeBuf`, which are
// package level and not re-entrant. A keypress, a paste and a remote call are
// all outermost dispatches. Nothing in this file may be called from `flush`.
func setPartPriority(k key, want int, player uint32) bool {
	id, ok := index[k]
	if !ok {
		return false
	}
	on := pprio[id] == 0
	switch want {
	case prioOff:
		on = false
	case prioOn:
		on = true
	}
	if (pprio[id] != 0) == on {
		return true // already what was asked
	}
	root := find(id)
	surf, ok := surfaceByIndex(k.s)
	if !ok {
		return false
	}
	force := pforce[root]
	tiles := collectCluster(root)
	if len(tiles) == 0 {
		return false
	}

	prev := pprio[id]
	pprio[id] = 0
	if on {
		pprio[id] = 1
	}
	// THE ONE PRE-TEARDOWN CHECK, ASKED WITH THE FLAG ALREADY FLIPPED. A
	// priority network is wider than the plain butterfly, so a balancer that
	// fits its slot without one may not fit with one -- and the flag is the only
	// thing that moved, so this is the only place that can tell.
	if pt, fits := plan.ShapeEdges(classifyEdges(surf, tiles, force)); !fits {
		pprio[id] = prev
		logPriorityRefused(root, k, pt)
		tellPriorityRefused(k, pt, player)
		return false
	}

	// The picture and the network both follow from the flush: restyle writes the
	// badged cell and compile sees the fingerprint move. Nothing is torn down
	// here, and nothing is spoken by a path that is not the one holding the
	// answer.
	markLive(root)
	requestFlush()
	logPriorityToggle(k, on)
	tellPriorityToggled(k, on, player)
	return true
}

// refusePriority is compile()'s answer when the shape a PRIORITY port asks for
// does not fit, which is the same answer refuseOverLimit gives for the port cap
// and goes through the same three-way admission: a refusal issued from inside
// `rebuildFromWorld` logs and requeues and tells nobody, a repeat of an edge
// state already refused says nothing, and anything else logs and speaks.
//
// It is reached when something OTHER than a toggle produced the shape -- a belt
// laid against a flagged part, a merge, a blueprint pasting a flagged balancer
// into a bigger one -- because a toggle is refused before the flag moves and
// never gets here.
func refusePriority(root uint32, fp uint64, pt plan.Ports, tiles []key, force uint32) {
	found := refusalFound(root, tiles, force)
	switch refuseAdmit(root, fp) {
	case refuseSilent:
		return
	case refuseLogOnly:
		logRefusedPriority(root, pt, found)
		return
	}
	logRefusedPriority(root, pt, found)
	side := pt.N
	if pt.M > side {
		side = pt.M
	}
	limMsg[1].Number = float64(side)
	tellRefusal(root, tiles, force, msgPrioTooBig, msgPrioTooBigStd, 1,
		"too big for a priority port")
}

// tellPriorityToggled is the flying text at the part, which is the only thing
// that happens in the same tick the key was pressed -- the badge on the sprite
// arrives with the flush, one tick later, and nobody can see 17 ms.
//
// A player who has left between the gesture and here gets the log line and
// nothing else, which is the ordinary case rather than an error. There is no
// force-wide arm: a remote caller is a script and has somewhere better to put
// the return value.
func tellPriorityToggled(k key, on bool, player uint32) {
	if player == 0 {
		return
	}
	msg := msgPrioOff
	if on {
		msg = msgPrioOn
	}
	prioMsg[0] = fkapi.OfString(msg)
	flyAtTile(k, fkapi.Value{Tag: fkapi.TagArray, Array: prioMsg[:1]}, player)
}

// tellPriorityRefused says the flag did not move. It names the rule and stops,
// for the reason limit.go's messages do: the 2026-08-05 field report was about a
// sentence that narrated a transaction the player never saw.
func tellPriorityRefused(k key, pt plan.Ports, player uint32) {
	if player == 0 {
		return
	}
	side := pt.N
	if pt.M > side {
		side = pt.M
	}
	prioMsg[0] = fkapi.OfString(msgPrioRefused)
	prioMsg[1].Number = float64(side)
	flyAtTile(k, fkapi.Value{Tag: fkapi.TagArray, Array: prioMsg[:2]}, player)
	if o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(player))); err == nil && o != nil {
		_ = fkapi.LuaPlayer{Object: *o}.PlaySound(limSound)
	}
}

// flyAtTile puts one localised string over one tile, for one player.
//
// AT THE PART AND NOT AT THE CLUSTER'S CENTRE, which is limit.go's own
// correction: a player hovering a part at the top of a 32-part column and given
// a message at the box centre never sees it. The tile they are pointing at is
// the one tile their eyes are guaranteed to be on.
func flyAtTile(k key, msg fkapi.Value, player uint32) {
	o, err := fkapi.Game.GetPlayer(fkapi.OfNumber(float64(player)))
	if err != nil || o == nil {
		return
	}
	prioPos.X, prioPos.Y = float64(k.x)+0.5, float64(k.y)+0.5
	prioText.Text = msg
	_ = fkapi.LuaPlayer{Object: *o}.CreateLocalFlyingText(prioText)
}

// logPriorityToggle is the assertion surface for the headless suite: which tile,
// and which way. `[BBB] priority part=x,y on|off`.
func logPriorityToggle(k key, on bool) {
	if !verboseLog {
		return
	}
	logStart("priority part=")
	logI(k.x)
	logS(",")
	logI(k.y)
	logS(" ")
	if on {
		logS("on")
	} else {
		logS("off")
	}
	logEnd()
}

// logPriorityRefused is the toggle's own refusal. `alert:` and not `error:` for
// the reason limit.go gives: a player asking for a balancer the mod does not
// build is an expected condition with a defined outcome, and `test/run.sh` fails
// a run on an `error:`.
func logPriorityRefused(root uint32, k key, pt plan.Ports) {
	logAlertStart("priority refused for cluster ")
	logU(root)
	logS(" at part ")
	logI(k.x)
	logS(",")
	logI(k.y)
	logS(": ")
	logU(uint32(pt.N))
	logS(" inputs and ")
	logU(uint32(pt.M))
	logS(" outputs with ")
	logU(uint32(pt.QIn))
	logS(" priority inputs and ")
	logU(uint32(pt.QOut))
	logS(" priority outputs does not fit; the flag was not set")
	logEnd()
}

// logRefusedPriority is compile()'s refusal, shared by the ordinary path and the
// silent rebuild path -- the twin of logRefusedOverLimit, and the edge suite
// reads this line.
func logRefusedPriority(root uint32, pt plan.Ports, found int) {
	logAlertStart("cluster ")
	logU(root)
	logS(" cannot be built with ")
	logU(uint32(pt.QIn))
	logS(" priority inputs and ")
	logU(uint32(pt.QOut))
	logS(" priority outputs over ")
	logU(uint32(pt.N))
	logS("->")
	logU(uint32(pt.M))
	logS(" ports; refused")
	logRefusalFound(found)
	logEnd()
}
