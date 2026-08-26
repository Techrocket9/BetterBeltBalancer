// Command bbb-m1-test drives the M1 cluster registry through every merge and
// split shape, and marks each phase in the log so the assertions can be
// positional.
//
// A COMPILED GO OBSERVER, not a Lua test mod. It is the same program the
// hand-written `test/mods/bbb-m1-test/control.lua` was, phase for phase and log
// line for log line; what changed is that there is no hand-written Lua left in
// this repository, on either side of the boundary. agents/estate-port.md is the
// programme.
//
// It asserts NOTHING itself -- the mod under test's own `[BBB] state` lines are
// the assertion surface and `test/assert-log.py` checks them. An observer that
// computed the expected answer would be a second implementation of the thing
// under test.
//
// Phase 1 runs in fk_on_init, which Factorio runs while `--create` builds the
// map, so the save already contains the clusters. Phases 2..9 run on later ticks
// during `--benchmark`, which means the registry has to survive a save and a
// reload to get there -- that is not incidental, it is what proves
// `--persist=packed` carries the guest heap. Both guests' heaps, now: this one's
// too.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const part = "bbb-balancer-part"

var out = harness.Line{Tag: "[BBB-TEST] "}

// The three surfaces. A and B carry parts at IDENTICAL coordinates: a registry
// that forgot to key by surface merges them, and every phase after the first is
// then wrong -- which is the space-platform bug, found on a flat map.
const (
	surfA    = "bbb-m1-a"
	surfB    = "bbb-m1-b"
	surfSkin = "bbb-m1-skin"
)

// flat is the scratch surface every one of the three is made from. The estate's
// Lua wrote this block out once per test mod; it is harness.Flat now.
func flat(name string) fkapi.LuaSurface {
	return harness.Flat{
		Name:        name,
		MapWidth:    256,
		MapHeight:   256,
		ChunkCenter: fkapi.MapPosition{X: 32, Y: 8},
		ChunkRadius: 4,
		X0:          -8,
		Y0:          -8,
		X1:          72,
		Y1:          24,
		Tile:        "grass-1",
	}.Make()
}

func place(s fkapi.LuaSurface, x, y int) {
	harness.Place(s, harness.Piece{Name: part, X: x, Y: y, Raise: true})
}

// remove takes the one part standing on a tile out through
// `destroy{raise_destroy = true}`, which is the script-raised removal path.
func remove(s fkapi.LuaSurface, x, y int) {
	o, ok := harness.FindExactlyOne(s, part, x, y)
	if !ok {
		return
	}
	yes := true
	if _, err := (fkapi.LuaEntity{Object: o}).Destroy(
		fkapi.LuaEntityDestroyArgs{RaiseDestroy: &yes}); err != nil {
		harness.Fatal("destroying the part", fk.LastError())
	}
}

func phase(n uint64, what string) {
	out.Open("phase=").U(n).S(" ").S(what).End()
}

// ---------------------------------------------------------------------------
// the patterns
// ---------------------------------------------------------------------------
//
//	           x=0        x=10..13      x=20..22        x=30..34
//	  y=0      A single   A line        A L (corner)    A bridged pair
//	  y=1                               |
//	  y=2                               |
//
// Surface B carries a single at (0,0) and a pair at (10,0)-(11,0): the same
// tiles surface A uses, so nothing may join across them.

func phase1() {
	phase(1, "build: single, line, L, bridge, cross-surface")
	a := flat(surfA)
	b := flat(surfB)

	// 1. a lone part -- a cluster of one
	place(a, 0, 0)

	// 2. a line, built left to right: three successive merges
	place(a, 10, 0)
	place(a, 11, 0)
	place(a, 12, 0)
	place(a, 13, 0)

	// 3. an L: the corner merges two arms that are not in line
	place(a, 20, 0)
	place(a, 21, 0)
	place(a, 22, 0)
	place(a, 22, 1)
	place(a, 22, 2)

	// 4. two separate clusters, THEN the tile that bridges them: one placement
	//    that merges two existing clusters rather than growing one
	place(a, 30, 0)
	place(a, 31, 0)
	place(a, 33, 0)
	place(a, 34, 0)
	place(a, 32, 0)

	// 5. surface B, at coordinates surface A is already using
	place(b, 0, 0)
	place(b, 10, 0)
	place(b, 11, 0)
}

// Remove the bridge: one cluster of 5 becomes two of 2.
func phase2() {
	phase(2, "remove bridge a(32,0): expect split 2+2")
	remove(harness.Surface(surfA), 32, 0)
}

// Remove the second tile of the line: 4 becomes 1 and 2.
func phase3() {
	phase(3, "remove line middle a(11,0): expect split 1+2")
	remove(harness.Surface(surfA), 11, 0)
}

// Remove the lone part: its cluster disappears entirely.
func phase4() {
	phase(4, "remove lone a(0,0): expect dissolve")
	remove(harness.Surface(surfA), 0, 0)
}

// Remove the L's corner: the two arms come apart.
func phase5() {
	phase(5, "remove L corner a(22,0): expect split 2+2")
	remove(harness.Surface(surfA), 22, 0)
}

// A DIFFERENT removal path. Everything above goes through
// `destroy{raise_destroy=true}`, so nothing has yet proved the guest is
// listening to on_entity_died -- and a part that dies to a biter or a nuke and
// stays in the registry is a cluster the compiler will build a network for
// twice.
func phase6() {
	phase(6, "kill b(11,0) via die(): expect shrink, on_entity_died path")
	o, ok := harness.FindExactlyOne(harness.Surface(surfB), part, 11, 0)
	if !ok {
		return
	}
	if _, err := (fkapi.LuaEntity{Object: o}).Die(nil, nil); err != nil {
		harness.Fatal("die() on b(11,0)", fk.LastError())
	}
}

// ---------------------------------------------------------------------------
// M5: the adaptive sprite
// ---------------------------------------------------------------------------
//
// The five named shapes, on their own surface so nothing above is disturbed.
// What is asserted is the mod's `[BBB] skin ... vars=` line -- the variation it
// put on every part of a cluster, in (y, x) order -- against the numbers
// guest/go/skin/skin_test.go proves in pure Go for the same five shapes. The Go
// test says the mapping is right; this says the mapping reached the entities in
// a real Factorio.
//
// ALL OF PHASE 7 IS ONE TICK, deliberately: the mod defers its work to the next
// tick, so thirty placements produce five skin lines, one per finished shape,
// rather than a line per intermediate shape.
//
//	x=0..3   line       x=10..12  L        x=20..22  plus
//	x=30..31 2x2 block  x=40..43  donut (4x4 ring around a 2x2 hole)

func phase7() {
	phase(7, "build the five named shapes: line, L, plus, 2x2, donut")
	s := flat(surfSkin)
	place(s, 0, 0)
	place(s, 1, 0)
	place(s, 2, 0)
	place(s, 3, 0)

	place(s, 10, 0)
	place(s, 10, 1)
	place(s, 10, 2)
	place(s, 11, 2)
	place(s, 12, 2)

	place(s, 21, 0)
	place(s, 20, 1)
	place(s, 21, 1)
	place(s, 22, 1)
	place(s, 21, 2)

	place(s, 30, 0)
	place(s, 31, 0)
	place(s, 30, 1)
	place(s, 31, 1)

	for y := 0; y <= 3; y++ {
		for x := 40; x <= 43; x++ {
			if x >= 41 && x <= 42 && y >= 1 && y <= 2 {
				continue
			}
			place(s, x, y)
		}
	}
}

// Grow the line by one tile at its east end. The new part and the part that USED
// to be the end change picture; the other three do not, and the mod must say
// so -- `set=2` out of `parts=5` is the claim that a part added to a balancer
// costs host calls for the pictures that moved and not for the ones that did
// not. That is what keeps this affordable on a 200-part balancer.
func phase8() {
	phase(8, "grow the line east: expect set=2 of parts=5")
	place(harness.Surface(surfSkin), 4, 0)
}

// Take the plus apart at its centre. Four lone parts, each of which must go back
// to the lone-part picture (variation 1): the update path in the other
// direction, and through RemovePart rather than AddPart.
func phase9() {
	phase(9, "remove the plus centre: expect four lone parts at variation 1")
	remove(harness.Surface(surfSkin), 21, 1)
}

// ---------------------------------------------------------------------------
// schedule
// ---------------------------------------------------------------------------

// Phases 2..9 land on separate ticks so the log is unambiguously ordered, and so
// that each one is a fresh event dispatch rather than a continuation of the one
// before.
var schedule = []harness.Step{
	{Tick: 10, Do: phase2},
	{Tick: 20, Do: phase3},
	{Tick: 30, Do: phase4},
	{Tick: 40, Do: phase5},
	{Tick: 50, Do: phase6},
	{Tick: 60, Do: phase7},
	{Tick: 70, Do: phase8},
	{Tick: 80, Do: func() {
		phase9()
		out.Open("phases complete").End()
	}},
}

func init() {
	// AN OBSERVER MAY HAVE A TICK HANDLER. The no-on_tick rule is the shipped
	// guest's, and it is the whole architecture there: a finished balancer must
	// cost zero script. What this mod is IS a schedule.
	//
	// The id is a literal constant at the call site, which is not optional:
	// FkLua prunes 218 event descriptors down to the ones a guest names, and an
	// id it cannot prove ships all of them.
	fkapi.Subscribe(fkapi.EventOnTick)
}

//go:wasmexport fk_on_init
func onInit() {
	phase1()
	out.Open("init complete").End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	harness.Run(schedule, fkapi.ReadOnTick(ptr).Tick)
}

func main() {}
