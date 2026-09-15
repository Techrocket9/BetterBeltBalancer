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
// the only outcome.
//
// AND NOTHING REACHES THE GROUND, which is the one place this file goes further
// than an ordinary edge edit does. The fourth decision spills what a materially
// smaller successor cannot hold, and a priority change really can build a
// smaller network -- clearing a balancer's last flag returns it to the plain
// butterfly, and on a 4x4 a second flag is smaller than the first. So the
// the successor is asked what it could take back and that is compared against
// what the machine is carrying BEFORE the flag moves, and a change that would
// not fit is refused with the balancer still running:
// `prioFitsWhatIsStanding`. The spill is a removal's outcome and
// a toggle is not a removal, so a toggle does not get it.
//
// WHAT THAT GUARD DOES NOT COVER, because neither is a flag changing:
//
//   - a belt MINED off a priority port shrinks the machine too, and takes the
//     pocket-then-spill rule every mine takes -- what the successor cannot hold
//     is offered to the player who mined it and only then to the floor
//     (carry.go, "The miner's pocket"). That is the contract for a removal and
//     this guard has no business overriding it.
//   - a blueprint or a paste landing a flagged part into a balancer that is
//     then too big goes through `compile`'s refusal, which is in front of its
//     own teardown, so the standing network is not touched at all.
//
// The two writers of `pprio` are `setPartPriority`, which is guarded, and
// `recoverPriority`, which raises a flag only on a part whose picture this guest
// has never written -- a revived ghost, a clone, a fresh heap. That part arrived
// with its belts, so the cluster it lands in is recompiled by the BUILD it came
// in on, and a build that shrinks a machine has taken the ordinary
// recompile-and-spill rule since long before this file existed (CLAUDE.md, "A
// mine beside a machine is a mine of that machine"). Changing that is a decision
// about builds and not about flags, and it is not taken here.
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
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/skin"
)

// InputTogglePriority is the custom-input prototype guest/go/data/priority.go
// defines and the name this guest subscribes under. The two modules cannot share
// a constant -- the control guest may not import `guest/go/tune` and the data
// guest may not import `fkapi` -- so it is written twice, which is the shape
// `PartName` already has, and each copy names the other.
const InputTogglePriority = "bbb-toggle-priority"

// The locale keys, in mod-data/locale/en/better-belt-balancer.cfg.
//
// THREE REFUSALS AND THEY ARE NOT THE SAME REFUSAL, which is what the key names
// say. A shape too big for a priority port is a statement about the machine's
// size; a priority INPUT is a statement about this version, since the planner
// refuses one at every size (agents/priority.md, "What input priority does");
// and a balancer too full to shrink is a statement about the moment, and comes
// back the second it has drained.
//
// The first two have a PAIR, shaped like limit.go's and delivered through the
// same `tellRefusal`: one sentence for a player who can be handed the piece back
// and one for a robot or script build that leaves it standing. Their singles are
// the TOGGLE's, which has no piece in it at all -- nobody built anything, the
// flag simply did not move. The third has no pair, because nothing but a toggle
// can reach it: a build cannot make a standing network smaller.
const (
	msgPrioOn        = "bbb.priority-on"
	msgPrioOff       = "bbb.priority-off"
	msgPrioTooBig    = "bbb.priority-too-big"
	msgPrioTooBigStd = "bbb.priority-too-big-unconnected"
	msgPrioRefused   = "bbb.priority-refused"
	msgPrioInput     = "bbb.priority-input"
	msgPrioInputStd  = "bbb.priority-input-unconnected"
	msgPrioInputRef  = "bbb.priority-input-refused"
	msgPrioHolding   = "bbb.priority-holding"
)

// The clauses tellRefusal names the bound with, in the force-wide line and in
// the hand-back line revertOne writes. Both complete "cluster N is ___".
const (
	whyPrioTooBig = "too big for a priority port"
	whyPrioInput  = "asking for an unsupported priority input"
)

// What setPartPriority was asked for.
const (
	prioToggle = iota
	prioOff
	prioOn
)

// Its own message buffers rather than limit.go's, and the reason is the COLOUR.
// A confirmation must not be vanilla's cannot-build red, or "Priority on" reads
// as something having gone wrong; a refusal must be, because that is what the
// base game has taught everybody a refusal looks like (limit.go, and the
// 2026-08-05 field report behind it). So the amber below is the DEFAULT this
// struct is wired to and `flyRefusal` borrows `limColor` for the three sentences
// that are refusals. Sharing the struct with limit.go instead would mean one
// mutable struct serving two files whose messages go to different places: this
// one is always at a part a player is pointing at, limit.go's is at a piece they
// just built.
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
	if setPartPriority(dk, on, e.PlayerIndex) {
		return
	}
	// A REFUSED PASTE HAS TO PUT THE PICTURE BACK ITSELF, and that is the one
	// thing zeroing `pvar` above costs. `setPartPriority` restores the flag on
	// every refusal, but the DESTINATION is still wearing the variation the
	// engine copied off the source -- badged, if the source was flagged -- and
	// with `pvar` at 0 the next restyle hands it to `recoverPriority`, which
	// reads a badge as a flag and raises one with no guard and no message. That
	// is a hole straight through the spill guard: the player is told the change
	// did not happen and the flag arrives a flush later anyway, on a balancer
	// the compiler will then refuse.
	//
	// So the shape the registry says this part should be drawing is written
	// here, in the dispatch that caused the mess. One host call on a keypress,
	// on the entity handle the event already carried, and `recoverPriority` has
	// nothing left to find.
	v := skin.Variation(maskAt(dk, pforce[did]), pprio[did] != 0)
	if err := dst.SetGraphicsVariation(v); err != nil {
		return
	}
	pvar[did] = v
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
//
// A TOGGLE IS REFUSED BEFORE THE FLAG MOVES: `plan.ShapeEdges` is asked with the
// flag speculatively flipped, so a shape the compiler could not build leaves the
// flag, the network and the items exactly as they were, rather than leaving the
// cluster standing refused until the player guessed to toggle back.
//
// EXCEPT THAT TAKING A FLAG OFF IS ALWAYS ALLOWED WHERE THE OLD SHAPE DID NOT
// FIT EITHER, which is the one direction that rule was wrong in. The fit bound
// is about the SHAPE and not about the change, so at P = 64 every q >= 1 is
// refused -- and a balancer that arrived there with two flags on it, by growing
// past P = 32 with them already set, could then not have either one taken off:
// each single un-flag lands on q = 1, which is refused, so the machine is stuck
// refused forever while README, agents/priority.md and limit.go's own hand-back
// sentence all tell the player that taking the flag off is the way out. The
// pre-toggle shape costs nothing to ask -- `ShapeEdges` reads the edge counts
// and `shapeWithTileFlipped` flips a field in a list already in hand, so there
// is no host call on either -- and a change that cannot make things worse than
// they already are is not a change to refuse.
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
	edges := classifyEdges(surf, tiles, force)
	pt, fits := plan.ShapeEdges(edges)
	if !fits {
		_, wasFits := shapeWithTileFlipped(edges, k)
		if on || wasFits {
			pprio[id] = prev
			logPriorityRefused(root, k, pt)
			tellPriorityRefused(k, pt, player)
			return false
		}
		// THE CLUSTER COULD NOT BE BUILT BEFORE THIS FLAG MOVED EITHER AND THE
		// FLAG IS COMING OFF, so it is allowed through: see the header. The
		// spill guard is skipped and that is not an omission -- a shape the
		// compiler refuses is refused in FRONT of its own teardown, so nothing
		// comes down and there is nothing to spill.
	} else if !prioFitsWhatIsStanding(root, k, pt, edges, player) {
		pprio[id] = prev
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

// prioFitsWhatIsStanding is the second pre-teardown check: a toggle that would
// make the balancer SMALLER than the items it is carrying is refused instead of
// spilling them.
//
// The fourth decision of "A recompile is not a removal" (carry.go) is that a
// successor which cannot hold what came down puts the remainder on the ground
// beside the cluster. That is the right answer for a machine a player mined --
// they asked for it to stop existing -- and the wrong one for a flag they
// ticked, which is a configuration change on a machine they want to keep. So
// the spill is closed at the source rather than handled downstream: nothing is
// torn down, nothing moves, and the player is told to let the balancer drain.
//
// THE BOUND IS `plan.Reinsertable` AND IT IS A LOWER ONE, deliberately under
// what a jammed network of either measured shape gave back in a game. A bound
// taken off the tile arithmetic instead passes a toggle on a 2->2 holding
// anything from 65 to 96 items and spills the difference, which is the defect
// this guard exists to prevent, met inside the guard. The two measurements and
// the discount they fix are in plan.Reinsertable's header.
//
// IT COMPARES TWO OF THEM RATHER THAN ASKING WHICH WAY THE FLAG WENT, and
// that is not caution. A 4x4's SECOND priority port is smaller than its first --
// one priority port leaves three normal ones and a square butterfly over three
// ports is four rows with a loopback in it, where two and two are a pair of
// single-splitter blocks -- so a toggle ON shrinks the network there.
// plan.TestAPriorityToggleCanShrinkTheNetworkInEitherDirection is that row.
//
// A cluster with no standing network has nothing to drain and skips the whole
// thing, which is every first toggle on a half-built balancer.
//
// AND IT IS ONLY ASKED WHEN THE POST-TOGGLE SHAPE FITS. A toggle onto a shape
// the compiler refuses tears nothing down -- the refusal is in front of the
// teardown -- so there is no successor for the items to fail to fit into.
func prioFitsWhatIsStanding(root uint32, k key, pt plan.Ports, edges []plan.Edge, player uint32) bool {
	ni, standing := nets[root]
	if !standing {
		return true
	}
	room := plan.Reinsertable(pt)
	was, wasFits := shapeWithTileFlipped(edges, k)
	// A PRE-TOGGLE SHAPE THAT DOES NOT FIT IS NOT A CAPACITY TO COMPARE
	// AGAINST, and the answer is to allow rather than to count. `Reinsertable`
	// gives such a shape 0, so a comparison would refuse every toggle on a
	// cluster whose edges have outgrown the bound -- which is the one gesture
	// that gets a refused balancer working again, and the trap the header's
	// coming-off rule is about, met one function along. What is standing there
	// was built by neither shape: the compiler refuses in front of the
	// teardown, so a cluster carrying a network it can no longer build is
	// carrying one from before the edges moved, and the toggle is not what
	// makes it wrong.
	if !wasFits || room >= plan.Reinsertable(was) {
		return true
	}
	held := heldItems(ni)
	if held <= uint32(room) {
		return true
	}
	logPriorityHolding(root, k, held, room)
	tellPriorityHolding(k, player)
	return false
}

// shapeWithTileFlipped is the shape this edge list had before one tile's flag
// moved: the tile's own edges flipped back, counted, and flipped again.
//
// `plan.ShapeEdges` reads the edge COUNTS and nothing else, so asking it twice
// about one list costs no host call and no allocation -- it is the same property
// that makes it safe to ask from a keypress at all. Flipping the tile rather
// than re-classifying the cluster is what keeps it that way: a second
// `classifyEdges` would be a position query per side of every part.
func shapeWithTileFlipped(edges []plan.Edge, k key) (plan.Ports, bool) {
	flipTilePrio(edges, k)
	pt, fits := plan.ShapeEdges(edges)
	flipTilePrio(edges, k)
	return pt, fits
}

func flipTilePrio(edges []plan.Edge, k key) {
	for i := range edges {
		if edges[i].TileX == k.x && edges[i].TileY == k.y {
			edges[i].Prio = !edges[i].Prio
		}
	}
}

// heldItems is the drain's own reading of a standing network, made without
// draining it: every entity a teardown would sweep, and the item count of every
// transport line on it.
//
// IT ENUMERATES EXACTLY WHAT `teardownNet` ENUMERATES -- the hidden slot's box
// and the visible cluster's box, through the same four names and the same
// prebuilt filter -- because a number that counted a different set would be a
// bound on the wrong thing. What it does not do is `drain`'s second half: no
// contents cross the boundary, so there is one host call per transport line and
// no allocation per line.
//
// IT IS PROPORTIONAL TO THE NETWORK, which on the largest shape the fit rule
// allows is a few thousand host calls: 1,267 entities, one line-count call each
// and up to eight line reads. At the ~12.6 us this repo measures for a tier-2
// call that is the same order as the recompile it is standing in front of, and
// it is paid once, on a keypress, only when the successor could take back less
// than the predecessor and only when a network is standing. Nothing has measured the product, and the cost of getting
// it wrong is the spill this exists to prevent.
//
// IT COUNTS ITEMS AND THE BOUND IT IS COMPARED AGAINST COUNTS BELT POSITIONS,
// and under belt stacking those are not the same unit: a stacked position holds
// up to four items, so the count can exceed the positions occupied. The
// comparison is therefore CONSERVATIVE on a stacking force -- it can refuse a
// toggle that would in fact have fitted -- and that is the side to be wrong on,
// because the other side is items on the ground. It is conservative on every
// force in any case, `plan.Reinsertable` being under what the engine has been
// measured giving back.
//
// A HALF THAT CANNOT BE READ CONTRIBUTES NOTHING, which is the honest reading
// rather than a fallback: the hidden surface being gone means the network's
// hidden half is gone with it, and a visible surface that cannot be resolved
// holds no interfaces of ours either.
func heldItems(ni netInfo) uint32 {
	total := uint32(0)
	// `hiddenIdx` rather than `hiddenSurface()`, which CREATES the surface when
	// it is missing. A keypress has no business making one, and a network that
	// is standing was compiled onto a surface that already exists.
	if hiddenIdx != 0 && ni.slot != 0 {
		if hid, ok := surfaceByIndex(hiddenIdx); ok {
			ox, oy := slotOrigin(ni.slot)
			total += countItemsIn(hid, ox, oy, ox+slotW-1, oy+slotH-1)
		}
	}
	if vis, ok := surfaceByIndex(ni.surf); ok {
		total += countItemsIn(vis, ni.x0, ni.y0, ni.x1, ni.y1)
	}
	return total
}

// heldEnts is the only buffer this reading needs; it grows to the biggest box
// any toggle has looked at and is reused. The filter and the search box are
// `sweep`'s own, which is the point: this is sweep's query without the sweep.
var heldEnts []fkapi.Object

func countItemsIn(s fkapi.LuaSurface, x0, y0, x1, y1 int32) uint32 {
	setSearchBox(x0, y0, x1, y1)
	total := uint32(0)
	for n := range sweepNames {
		nameFilter = fkapi.OfString(sweepNames[n])
		ents, err := s.FindEntitiesFilteredInto(heldEnts, findByName)
		if err != nil {
			continue
		}
		heldEnts = ents
		for i := range ents {
			e := fkapi.LuaEntity{Object: ents[i]}
			// The line count is ASKED FOR rather than assumed per prototype, for
			// `drain`'s reason: a constant that was wrong by one would quietly
			// under-count here, which is a spill this guard promised to prevent.
			lines, err := e.GetMaxTransportLineIndex()
			if err != nil || lines == 0 || lines > maxLinesToProbe {
				continue
			}
			for j := uint32(1); j <= lines; j++ {
				l, err := e.GetTransportLine(j)
				if err != nil {
					break
				}
				c, err := fkapi.LuaTransportLine{Object: l}.GetItemCount(nil)
				if err == nil {
					total += c
				}
			}
		}
	}
	return total
}

// refusePriority is compile()'s answer when a shape with a priority port on it
// cannot be built, which is the same answer refuseOverLimit gives for the port
// cap and goes through the same three-way admission: a refusal issued from
// inside `rebuildFromWorld` logs and requeues and tells nobody, a repeat of an
// edge state already refused says nothing, and anything else logs and speaks.
//
// It is reached when something OTHER than a toggle produced the shape -- a belt
// laid against a flagged part, a merge, a blueprint pasting a flagged balancer
// into a bigger one -- because a toggle is refused before the flag moves and
// never gets here.
//
// TWO SENTENCES, ONE PREDICATE. A shape carrying a priority INPUT is refused at
// every size, because the construction for it is not built; anything else that
// reaches here is refused for its size. A cluster can be both, and the input
// sentence wins for `refuseShape`'s reason: taking the flag off is the fix in
// either case, and it is the flag the player last touched.
func refusePriority(root uint32, fp uint64, pt plan.Ports, tiles []key, force uint32) {
	found := refusalFound(root, tiles, force)
	input := pt.QIn > 0
	switch refuseAdmit(root, fp) {
	case refuseSilent:
		return
	case refuseLogOnly:
		logRefusedPriority(root, pt, found, input)
		return
	}
	logRefusedPriority(root, pt, found, input)
	if input {
		tellRefusal(root, tiles, force, msgPrioInput, msgPrioInputStd, 0, whyPrioInput)
		return
	}
	side := pt.N
	if pt.M > side {
		side = pt.M
	}
	limMsg[1].Number = float64(side)
	tellRefusal(root, tiles, force, msgPrioTooBig, msgPrioTooBigStd, 1, whyPrioTooBig)
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
//
// An input priority takes no number with it. Where the bound falls for a SIZE is
// something a player can act on; that this version does not build an input
// priority at all is true at every size, and a number beside it would suggest a
// smaller balancer were the way out.
func tellPriorityRefused(k key, pt plan.Ports, player uint32) {
	if player == 0 {
		return
	}
	if pt.QIn > 0 {
		prioMsg[0] = fkapi.OfString(msgPrioInputRef)
		flyRefusal(k, prioMsg[:1], player)
		return
	}
	side := pt.N
	if pt.M > side {
		side = pt.M
	}
	prioMsg[0] = fkapi.OfString(msgPrioRefused)
	prioMsg[1].Number = float64(side)
	flyRefusal(k, prioMsg[:2], player)
}

// tellPriorityHolding says the balancer is too full to be rebuilt smaller. No
// number: the count and the room are in the log line, and what a player can act
// on is the machine emptying, which they can see.
func tellPriorityHolding(k key, player uint32) {
	if player == 0 {
		return
	}
	prioMsg[0] = fkapi.OfString(msgPrioHolding)
	flyRefusal(k, prioMsg[:1], player)
}

// flyRefusal is the toggle's refusal in front of the player who pressed the key:
// the text in vanilla's cannot-build red and the standard cannot-build sound,
// which is what the base game has taught everybody a refusal looks and sounds
// like (limit.go, and the 2026-08-05 field report behind it). A confirmation
// goes through flyAtTile instead and keeps the badge's amber, or "Priority on"
// reads as something having gone wrong.
func flyRefusal(k key, msg []fkapi.Value, player uint32) {
	prioText.Color = &limColor
	flyAtTile(k, fkapi.Value{Tag: fkapi.TagArray, Array: msg}, player)
	prioText.Color = &prioColor
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

// logPriorityRefused is the toggle's own refusal, and it ends in the reason so
// that the two are one line apart for a suite to key on: a shape that does not
// fit is about the machine's size and an input priority is about this version.
//
// `alert:` and not `error:` for the reason limit.go gives: a player asking for a
// balancer the mod does not build is an expected condition with a defined
// outcome, and `test/run.sh` fails a run on an `error:`.
func logPriorityRefused(root uint32, k key, pt plan.Ports) {
	logPriorityHead(root, k)
	logU(uint32(pt.N))
	logS(" inputs and ")
	logU(uint32(pt.M))
	logS(" outputs with ")
	logU(uint32(pt.QIn))
	logS(" priority inputs and ")
	logU(uint32(pt.QOut))
	logS(" priority outputs ")
	if pt.QIn > 0 {
		logS("is a priority input, which this version does not build")
	} else {
		logS("does not fit")
	}
	logS("; the flag was not set")
	logEnd()
}

// logPriorityHolding is the spill guard's refusal: the two numbers it compared,
// in the units it compared them in, because that comparison is the whole of the
// decision and neither number is visible anywhere else.
func logPriorityHolding(root uint32, k key, held uint32, room int) {
	logPriorityHead(root, k)
	logS("the balancer holds ")
	logU(held)
	logS(" items and the network this would build can take back ")
	logU(uint32(room))
	logS("; the flag was not set")
	logEnd()
}

func logPriorityHead(root uint32, k key) {
	logAlertStart("priority refused for cluster ")
	logU(root)
	logS(" at part ")
	logI(k.x)
	logS(",")
	logI(k.y)
	logS(": ")
}

// logRefusedPriority is compile()'s refusal, shared by the ordinary path and the
// silent rebuild path -- the twin of logRefusedOverLimit, and the edge suite
// reads this line.
func logRefusedPriority(root uint32, pt plan.Ports, found int, input bool) {
	logAlertStart("cluster ")
	logU(root)
	if input {
		logS(" asks for ")
		logU(uint32(pt.QIn))
		logS(" priority inputs over ")
		logU(uint32(pt.N))
		logS("->")
		logU(uint32(pt.M))
		logS(" ports, which this version does not build; refused")
		logRefusalFound(found)
		logEnd()
		return
	}
	logS(" cannot be built with ")
	logU(uint32(pt.QOut))
	logS(" priority outputs over ")
	logU(uint32(pt.N))
	logS("->")
	logU(uint32(pt.M))
	logS(" ports; refused")
	logRefusalFound(found)
	logEnd()
}
