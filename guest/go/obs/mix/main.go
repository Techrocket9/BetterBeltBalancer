// Command bbb-mix-test drives MORE THAN ONE KIND of item through one balancer
// and reports every count PER ITEM NAME.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-mix-test/control.lua` was, rig for rig and log line for log
// line, with its little data stage compiled too (obs/mixdata). See
// agents/estate-port.md.
//
// # What it is for
//
// Every other suite ran iron plates through everything. That is the right
// default -- it makes a throughput number mean something -- and it left the
// whole multi-kind half of `guest/go/carry.go` unexercised: the pool's
// (name, quality, stack size) key, the per-kind split, insertRemainder's walk
// over several groups, and the BOUND on how many groups one pool can carry.
//
// WHAT IT CANNOT REACH, and it is a property of the DLC rather than of the rigs:
// `compile.go`'s `detailedTally` and `kindAt`. This suite is base only, and
// below the stacking gate the drain takes the flat totals and that code is never
// called at all -- so no rig here exercises it however many kinds it runs.
// Multi-kind AND STACKED lives in the `plat` suite's `smix` band.
//
// It ASSERTS NOTHING. It builds rigs, counts items BY NAME on both surfaces at
// named ticks, and logs the numbers; test/assert-mix.py decides whether they are
// right. An observer that computed the expected answer would be a second
// implementation of the thing under test.
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
// Every column of parts is TWO columns -- a west part carrying the row's input
// and an east part carrying its output -- because an edge is an interface linked
// belt standing on the cluster's own tile, and 2.1's collision validator forbids
// two belt-connectables on one tile. See agents/single-edge.md.
//
// What that costs a rig is GEOMETRY AND NOTHING ELSE. N, M and the item kinds in
// flight are properties of the BELTS, and the belts did not move, so every
// conservation contract in this file is the one it was.
//
// ...except for one thing, and it is the same thing `m2`'s conservation rig and
// the interactive checklist's band B both ran into independently: under this
// rule a WORKING BALANCER HAS NO FREE FACE. Every part of it already carries its
// one belt, so the belt each conservation check used to lay on the cluster's
// north face would now be REFUSED and the check would measure a refusal instead
// of a recompile. So every rig with parts carries one extra EDGELESS part below
// its west column, and that is where the belt goes.
//
// # The rigs, one per y band on a flat scratch surface
//
//	ctrl     a bare express belt, chest to chest, iron plates. The yardstick,
//	         exactly as in M2: "full throughput" is a comparison against the
//	         engine rather than against arithmetic on a wiki number.
//	probe    the multi-filter source with NO balancer at all: chest to belt to
//	         chest. It is here to keep the paragraph below honest -- what one
//	         infinity chest with six filters actually puts on a belt is a
//	         measurement, and this is where it is taken.
//	duo      a 2->2 balancer fed by two PURE belts -- one iron, one copper --
//	         draining freely into two chests. Does a balancer carrying two kinds
//	         still deliver two belts' worth, and does each kind come out at the
//	         rate it went in? It is one of the two rigs here whose numbers are
//	         about a RUNNING balancer, which is why its conservation edit is the
//	         last thing the schedule does rather than the first.
//	quad     the same question one size up: a 4->4 fed by two PURE iron belts and
//	         two PURE copper ones, ALTERNATING, draining freely. It exists because
//	         `duo` alone could not tell a two-line accident from a property -- and
//	         what the two of them measure together is that under SYMMETRIC
//	         SATURATION this butterfly is a PERMUTATION: every output takes
//	         exactly its share by count and exactly one kind, at both sizes.
//	mixfull  a 2->2 fed by two SUSHI belts, outputs DEAD-ENDED so the hidden
//	         network fills and stays full. Then a forced recompile inside one
//	         tick: is every kind conserved exactly?
//	many     a 4-in 4-out balancer fed by four sushi belts drawing on 48 distinct
//	         base items between them, outputs dead-ended. Past the pool's bound of
//	         32 groups the drain cannot CARRY them all, and what it cannot carry
//	         has to reach the WORLD rather than stop existing. This is the rig
//	         that fails on the guest that shipped.
//
// # How a sushi belt is made here, and why it is not six filters in one chest
//
// The obvious rig is an infinity chest with six filters feeding one loader, and
// the `probe` band is that rig, kept precisely so this paragraph is a
// measurement rather than an assertion. It does not produce a mixed belt: a
// loader draws from the first stack it finds in the source inventory and the
// infinity chest tops that same stack straight back up.
//
// So a source here holds ONE filter at a time and ROTATES it, with
// `remove_unfiltered_items` on so the previous kind is voided rather than left
// for the loader to prefer. The result is a banded belt -- a short run of each
// kind, four ticks of belt each -- which is what a real sushi bus looks like
// anyway, and it is deterministic: the band boundaries are a function of
// `game.tick` and nothing else.
//
// # Forcing the flush
//
// The guest batches: a build event updates its registry inside the event and
// defers the recompile to the next tick (`fk.Defer`), so a measurement taken in
// the tick that laid the belt would see nothing at all. `bbb-audit` -- a shipped
// marker prototype whose whole purpose is "re-classify and repair everything,
// now" -- is the synchronous escape hatch. That is also why `fk_on_init` ends
// with one: `--create` never reaches a tick, so without it every network in the
// save would be compiled on the first tick of the BENCHMARK instead.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

const (
	part     = "bbb-balancer-part"
	belt     = "express-transport-belt"
	surfName = "bbb-mix-a"
	hidden   = "bbb-hidden"
)

var out = harness.Line{Tag: "[BBB-MIX] "}

// east is read through the GENERATED accessor, which resolves
// `defines.direction.east` BY NAME against the running game: a define's number
// is Factorio's own, is not in the API description at all, and nothing in this
// repository writes one down.
var east uint32

// ---------------------------------------------------------------------------
// the item lists
//
// Every name is checked against `prototypes.item` at init and a missing one is
// FATAL, so a rename in a future Factorio fails the run in the CREATE log with
// the name in it rather than producing a rig that quietly carries fewer kinds
// than it claims.
// ---------------------------------------------------------------------------

// Two sushi bands of six, so `mixfull` can hold up to twelve distinct kinds --
// comfortably inside the pool's bound of 32, which is the point: this rig
// exercises the multi-kind tally, split and reinsertion paths WITHOUT
// overflowing anything.
var mixfullItems = [][]string{
	{"iron-plate", "copper-plate", "steel-plate", "iron-gear-wheel",
		"copper-cable", "electronic-circuit"},
	{"stone", "stone-brick", "coal", "wood", "pipe", "plastic-bar"},
}

// Four bands of twelve = 48 distinct kinds, comfortably PAST the pool's bound of
// 32 groups. That is the whole purpose of the `many` rig, and 48 rather than 33
// because the bound is on GROUPS: a base-only run has one quality and no belt
// stacking, so a group is exactly a name here, but a rig sized at the bound plus
// one would stop overflowing the day either of those changed.
//
// No two bands share a name, so the count below is the count of distinct kinds
// and `checkItems` says so out loud rather than leaving it to be counted from
// the source. Every name is a base item.
var manyItems = [][]string{
	{"iron-plate", "copper-plate", "steel-plate", "iron-ore", "copper-ore",
		"coal", "stone", "stone-brick", "wood", "iron-gear-wheel", "copper-cable",
		"iron-stick"},
	{"electronic-circuit", "advanced-circuit", "processing-unit", "plastic-bar",
		"sulfur", "battery", "explosives", "engine-unit", "electric-engine-unit",
		"flying-robot-frame", "low-density-structure", "rocket-fuel"},
	{"pipe", "pipe-to-ground", "transport-belt", "fast-transport-belt",
		"express-transport-belt", "underground-belt", "splitter", "inserter",
		"fast-inserter", "long-handed-inserter", "small-electric-pole",
		"medium-electric-pole"},
	{"automation-science-pack", "logistic-science-pack", "military-science-pack",
		"chemical-science-pack", "production-science-pack", "utility-science-pack",
		"speed-module", "efficiency-module", "productivity-module", "solid-fuel",
		"uranium-ore", "barrel"},
}

// checkItems is the ANTI-VACUITY gate, and it has two halves. A missing name is
// FATAL and names itself, in the create log, rather than leaving a rig that
// quietly carries fewer kinds than it claims -- which for `many` would mean an
// overflow rig that never overflows and passes vacuously. And `distinct` is
// LOGGED for the same reason: assert-mix.py reads it and requires the number it
// was promised, because a shorter list might not overflow at all.
func checkItems() {
	var missing []string
	var seen []string
	for _, band := range mixfullItems {
		for _, name := range band {
			if !harness.ItemProtoExists(name) {
				missing = append(missing, name)
			}
		}
	}
	for _, band := range manyItems {
		for _, name := range band {
			if !harness.ItemProtoExists(name) {
				missing = append(missing, name)
			}
			known := false
			for _, s := range seen {
				if s == name {
					known = true
					break
				}
			}
			if !known {
				seen = append(seen, name)
			}
		}
	}
	if len(missing) > 0 {
		l := harness.Line{Tag: "[BBB-OBS] "}
		l.Open("error: no such item prototype: ")
		for i, m := range missing {
			if i > 0 {
				l.S(", ")
			}
			l.S(m)
		}
		l.End()
		return
	}
	out.Open("item lists ok: many distinct=").U(uint64(len(seen))).
		S(" mixfull=").U(uint64(len(mixfullItems[0]) + len(mixfullItems[1]))).End()
}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func surf() fkapi.LuaSurface { return harness.Surface(surfName) }

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) {
	harness.Place(s, harness.Piece{Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true})
}

// auditNow asks the guest to drain its deferred queue and re-classify every
// cluster, synchronously, inside this call. See "forcing the flush" above.
func auditNow() { harness.Audit(surf(), -30, 0) }

// chest places an infinity chest WITHOUT raising, exactly as the Lua did: a
// source is not something the mod under test needs to see.
func chest(s fkapi.LuaSurface, x, y int) fkapi.Object {
	return harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
}

// source is a PURE source: one item, one filter, never rotated.
func source(s fkapi.LuaSurface, x, y int, item string) {
	harness.InfinityFilter(chest(s, x, y), item, "", 1000)
	put(s, protos.MixLoader, x+1, y, &east, "output")
}

// multiSource is a chest carrying EVERY filter at once. Only the probe band uses
// it, and what it is for is the measurement in the header.
func multiSource(s fkapi.LuaSurface, x, y int, items []string) {
	harness.MultiFilter(chest(s, x, y), items, 1000)
	put(s, protos.MixLoader, x+1, y, &east, "output")
}

// rotate is how often a sushi source rewrites its single filter.
const rotate = 4

// sushi is one rotating source. It holds the chest's TILE and not the chest:
// `fk_on_init` runs during `--create` and the rotation runs during
// `--benchmark`, so anything held here crosses a save.
type sushi struct {
	X, Y   int
	Items  []string
	Offset int
}

var sushis []sushi

// sushiSource is the same chest and loader as `source`, but its single filter is
// rewritten every `rotate` ticks and `remove_unfiltered_items` voids what the
// last band left behind. `offset` staggers the bands so that four of these do
// not all switch to their first item on the same tick.
func sushiSource(s fkapi.LuaSurface, x, y int, items []string, offset int) {
	c := chest(s, x, y)
	harness.RemoveUnfiltered(c, true)
	harness.InfinityFilter(c, items[0], "", 1000)
	put(s, protos.MixLoader, x+1, y, &east, "output")
	sushis = append(sushis, sushi{X: x, Y: y, Items: items, Offset: offset})
}

func rotateSushi(tick uint64) {
	if tick%rotate != 0 {
		return
	}
	step := int(tick / rotate)
	s := surf()
	for _, sr := range sushis {
		c, ok := harness.FindAt(s, sr.X, sr.Y, "infinity-chest", "")
		if !ok {
			continue
		}
		harness.InfinityFilter(c, sr.Items[(step+sr.Offset)%len(sr.Items)], "", 1000)
	}
}

func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	put(s, protos.MixLoader, x, y, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

// ---------------------------------------------------------------------------
// the rigs
//
//	x=-5 source chest   -4 loader   -3..-1 belts   0 WEST PART   1 EAST PART
//	x=2..4 belts        5 sink loader   6 chest
//	                                   (dead-ended rigs stop after x=4)
//
// TWO COLUMNS OF PARTS, not one: one belt per part, so a row's input and its
// output cannot stand against the same tile.
//
// One extra EDGELESS part hangs below the west column of every rig that has
// parts, at (0, base + rows). It carries no belt and is therefore the one tile
// in the cluster a player's belt can still reach -- laying an east-facing belt
// at (-1, base + rows) is a real edge change (a new INPUT, so the port count
// goes UP), the fingerprint moves, and the network is torn down and rebuilt.
// That is how every conservation check here forces a recompile.
// ---------------------------------------------------------------------------

const pitch = 14

// rigCfg is one band. `Parts` is the number of ROWS and the part count is twice
// it plus one.
type rigCfg struct {
	Name    string
	Parts   int
	Feeds   []string
	Sushi   [][]string
	Multi   []string
	Drained bool
}

var rigCfgs = []rigCfg{
	{Name: "ctrl"},
	{Name: "probe", Multi: mixfullItems[0]},
	{Name: "duo", Parts: 2, Feeds: []string{"iron-plate", "copper-plate"}, Drained: true},
	// The same shape one size up, and the kinds ALTERNATE down the rows rather
	// than being grouped, so a network that merely passed row i to row i and one
	// that permuted the rows would read differently.
	{Name: "quad", Parts: 4, Drained: true,
		Feeds: []string{"iron-plate", "copper-plate", "iron-plate", "copper-plate"}},
	{Name: "mixfull", Parts: 2, Sushi: mixfullItems},
	// Four parts, four in, four out, dead-ended -- M2's `sat4` shape, which is the
	// one this repo has already measured holding ~230 items when it is full. That
	// matters: 48 kinds have to be SIMULTANEOUSLY standing on the lines for the
	// pool to overflow, and a network that only holds a dozen items could not do
	// it however many kinds went past.
	{Name: "many", Parts: 4, Sushi: manyItems},
}

// rig is one built band: where it is, how many rows, and the chests its outputs
// drain into. TILES rather than entities, for the reason `sushi` holds one.
type rig struct {
	Name   string
	Base   int
	Parts  int
	Chests []harness.XY
}

var rigs []rig

func rigNamed(name string) *rig {
	for i := range rigs {
		if rigs[i].Name == name {
			return &rigs[i]
		}
	}
	harness.Fatal("no rig", name)
	return nil
}

func buildRig(cfg rigCfg, base int) {
	s := surf()
	r := rig{Name: cfg.Name, Base: base, Parts: cfg.Parts}

	if cfg.Parts == 0 { // a bare belt, chest to chest
		if cfg.Multi != nil {
			multiSource(s, -5, base, cfg.Multi)
		} else {
			source(s, -5, base, "iron-plate")
		}
		for x := -3; x <= 3; x++ {
			put(s, belt, x, base, &east, "")
		}
		r.Chests = append(r.Chests, sink(s, 4, base))
		rigs = append(rigs, r)
		return
	}

	// Parts FIRST, belts after, so that the belt events are what drive the
	// compiles -- the same order M2 builds in and for the same reason.
	for i := 0; i < cfg.Parts; i++ {
		put(s, part, 0, base+i, nil, "")
		put(s, part, 1, base+i, nil, "")
	}
	put(s, part, 0, base+cfg.Parts, nil, "")

	for i := 1; i <= cfg.Parts; i++ {
		y := base + i - 1
		if cfg.Sushi != nil {
			sushiSource(s, -5, y, cfg.Sushi[i-1], (i-1)*3)
		} else {
			source(s, -5, y, cfg.Feeds[i-1])
		}
		for x := -3; x <= -1; x++ {
			put(s, belt, x, y, &east, "")
		}
		for x := 2; x <= 4; x++ {
			put(s, belt, x, y, &east, "")
		}
		// A dead-ended output backs up, which is what fills the hidden network and
		// keeps it full: the conservation rigs need a network that is holding as
		// much as it can hold.
		if cfg.Drained {
			r.Chests = append(r.Chests, sink(s, 5, y))
		}
	}
	rigs = append(rigs, r)
}

// ---------------------------------------------------------------------------
// counting, BY ITEM NAME
//
// Everything, on both surfaces: on the ground, on a belt, inside a splitter's
// transport lines, in a chest. An item this mod can lose is an item that left
// this total, and there is nowhere else for one to be.
//
// Counting only the visible side would not be conservation at all -- the point
// of the network is that most of the items are somewhere the player cannot see,
// so a teardown that deleted them would look like a gain on the visible side.
// The whole hidden surface is counted rather than one slot, because no tick
// passes between the two samples and every other rig's network is therefore
// frozen.
//
// PER NAME rather than as one total, which is the whole point of this suite: a
// teardown that dropped one KIND and reinserted the rest conserves nothing, and
// a single total would have to lose the same number of items twice in opposite
// directions to hide it.
// ---------------------------------------------------------------------------

// visArea is wide enough to contain every band plus the spill radius of the one
// rig that deliberately spills. It is a constant rather than a function of the
// band count because it is read in the same breath as the hidden surface's, and
// an item that landed outside it would read as LOSS -- which would put the bug
// on the wrong side of every assertion in this suite.
func visArea() fkapi.BoundingBox    { return harness.Box(-34, -20, 20, 140) }
func hiddenArea() fkapi.BoundingBox { return harness.Box(-16, -16, 2200, 400) }

// countArea adds everything countable in one box into `t` and returns how much
// of it was on the GROUND.
func countArea(s fkapi.LuaSurface, area fkapi.BoundingBox, t *harness.Tally) int64 {
	var ground int64
	for _, e := range harness.EntitiesIn(s, area, "") {
		if harness.EntityType(e) == "item-entity" {
			if name, n, ok := harness.GroundStack(e); ok {
				t.Add(name, n)
				ground += n
			}
			continue
		}
		harness.EachLine(e, func(l fkapi.LuaTransportLine) {
			c, err := l.GetContents()
			if err != nil {
				return
			}
			for _, item := range c {
				t.Add(item.Name, int64(item.Count))
			}
		})
		for _, item := range harness.ChestContents(e) {
			t.Add(item.Name, int64(item.Count))
		}
	}
	return ground
}

// countKinds is the whole-world count, and the infinity chests are DELIBERATELY
// EXCLUDED. They mint and void items every tick by design, so anything they hold
// is not a conserved quantity -- including them would make every count a
// measurement of the source rather than of the balancer. They are found by type
// and their contents subtracted back out.
//
// The HIDDEN surface is counted into its own tally as well as the total, and
// that second number is the anti-vacuity one: it is exactly the compiled
// networks' contents and nothing else, so "how many distinct kinds were really
// inside a balancer when it was torn down" is answerable without trusting the
// guest's own account of it.
func countKinds(all *harness.Tally) (ground, hiddenTotal int64, hiddenKinds int) {
	var hid harness.Tally
	s := surf()
	ground = countArea(s, visArea(), all)
	if h, err := fkapi.Game.GetSurface(fkapi.OfString(hidden)); err == nil && h != nil {
		ground += countArea(fkapi.LuaSurface{Object: *h}, hiddenArea(), &hid)
	}
	for _, name := range hid.Names() {
		n := hid.Get(name)
		all.Add(name, n)
		if n > 0 {
			hiddenTotal += n
			hiddenKinds++
		}
	}
	for _, c := range harness.EntitiesIn(s, visArea(), "") {
		if harness.EntityType(c) != "infinity-container" {
			continue
		}
		for _, item := range harness.ChestContents(c) {
			all.Add(item.Name, -int64(item.Count))
		}
	}
	return ground, hiddenTotal, hiddenKinds
}

// emit is one whole-world sample: a count line and then one line per kind,
// SORTED by name, so two samples of the same world produce the same lines in the
// same order on every machine.
func emit(tag string) {
	var t harness.Tally
	ground, hiddenTotal, hiddenKinds := countKinds(&t)
	names := t.Names()
	out.Open("count tag=").S(tag).S(" total=").I(t.Total()).
		S(" ground=").I(ground).S(" kinds=").U(uint64(len(names))).
		S(" hidden=").I(hiddenTotal).S(" hkinds=").U(uint64(hiddenKinds)).End()
	for _, name := range names {
		out.Open("kind tag=").S(tag).S(" name=").S(name).
			S(" count=").I(t.Get(name)).End()
	}
}

// ---------------------------------------------------------------------------
// the conservation check
//
// Inside a single tick, with no other movement possible: count every item by
// name on both surfaces, lay a belt on the cluster's one free face, force the
// recompile with an audit marker, count again. Nothing else can have moved, so
// the difference is exactly what the teardown did with what it drained.
// ---------------------------------------------------------------------------

func conserve(name string) {
	r := rigNamed(name)
	if r == nil {
		return
	}
	out.Open("mark rig=").S(name).S(" tick=").U(harness.Tick()).End()
	emit(name + "-before")
	// Against the EDGELESS part below the west column, which under the one-belt
	// rule is the only tile of this cluster a belt can still reach: an east-facing
	// belt there is a new INPUT edge, so the port count goes UP and the network the
	// rebuild produces is at least as big as the one it replaced. That matters: a
	// shrink would spill legitimately (carry.go, decision 4) and this check is
	// about the kinds, not about capacity.
	put(surf(), belt, -1, r.Base+r.Parts, &east, "")
	auditNow()
	emit(name + "-after")
}

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

// chestKinds is one sink's total and its per-name breakdown, sorted -- or -1 and
// an empty detail when there is no chest there, which is the estate's own
// convention: a rig whose sink went missing reports a number no real count can
// produce rather than a zero that reads as "nothing was delivered".
func chestKinds(s fkapi.LuaSurface, c harness.XY) (int64, []string, *harness.Tally) {
	o, ok := harness.FindOnTile(s, "steel-chest", c.X, c.Y)
	if !ok {
		return -1, nil, nil
	}
	t := &harness.Tally{}
	var total int64
	for _, item := range harness.ChestContents(o) {
		t.Add(item.Name, int64(item.Count))
		total += int64(item.Count)
	}
	return total, t.Names(), t
}

func report(tick uint64) {
	s := surf()
	for _, name := range []string{"ctrl", "probe", "duo", "quad"} {
		r := rigNamed(name)
		if r == nil {
			continue
		}
		totals := make([]int64, 0, len(r.Chests))
		for i, c := range r.Chests {
			total, names, t := chestKinds(s, c)
			totals = append(totals, total)
			out.Open("t=").U(tick).S(" rig=").S(name).S(" out").U(uint64(i + 1)).S(" kinds=")
			for j, n := range names {
				if j > 0 {
					out.S(",")
				}
				out.S(n).S(":").I(t.Get(n))
			}
			out.End()
		}
		out.Open("t=").U(tick).S(" rig=").S(name).S(" out=[")
		for i, n := range totals {
			if i > 0 {
				out.S(" ")
			}
			out.I(n)
		}
		out.S("]").End()
	}
}

// ---------------------------------------------------------------------------
// the schedule
//
// No edit lands INSIDE the throughput window, so every rate here is measured
// over a stretch in which nothing was touched -- the same discipline M2's
// `regrow` rig follows. The two dead-ended rigs are edited before it, because
// nothing about them is measured over time; `duo` is edited AFTER it, because it
// is the only rig whose numbers are about a running balancer.
//
// `duo`'S EDIT IS AT THE END, and it is worth saying why rather than leaving it
// as a schedule detail. Its conservation belt takes the rig from 2->2 (P=2) to
// 3->2 (P=4), and the per-kind split at an output is a function of WHICH PORT
// each belt landed on -- so with the edit first, this rig's headline measurement
// was never about the 2->2 its own description names. Under the old multi-edge
// geometry the extra belt entered the edge list FIRST and the split came out
// 75/25; under the new one it enters LAST and the same P=4 network delivers
// 100/0, exactly balanced by count either way. Neither is a defect -- the
// butterfly balances COUNTS and nothing in plan.Build knows what an item is --
// and neither was ever the question. The pristine 2->2 is, and now that is what
// the window sees.
// ---------------------------------------------------------------------------

var schedule = []harness.Step{
	{Tick: 1000, Do: func() { conserve("mixfull") }},
	// `many` last of the three, and 200 ticks after `mixfull`, because it is the
	// one whose result depends on how FULL it is: 48 kinds only overflow a
	// 32-group pool if 48 kinds are simultaneously standing on the lines, and a
	// dead-ended network freezes holding whatever was in flight when it filled.
	// Every source cycles its twelve items every 48 ticks, so by t=1200 each of
	// them has run its list twenty-five times over.
	{Tick: 1200, Do: func() { conserve("many") }},
	{Tick: 1400, Do: func() { report(1400) }},
	{Tick: 3140, Do: func() { report(3140) }},
	// After the last sample, so it cannot disturb one.
	{Tick: 3160, Do: func() { conserve("duo") }},
	// Last of all: nothing else touches the world after it, so the registry and
	// the world must agree exactly -- and the cluster and part counts are what say
	// the rigs are the shape this suite thinks they are, which no rate and no
	// per-name total can.
	{Tick: 3180, Do: auditNow},
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	east = fkapi.DefinesDirectionEast()
}

//go:wasmexport fk_on_init
func onInit() {
	checkItems()
	rows := (len(rigCfgs) + 1) * pitch
	harness.Flat{
		Name:        surfName,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		// ceil(rows/32) + 3, written out because rows is a constant and the
		// arithmetic was a `math.ceil` in the Lua. rows is 98, so ceil is 4.
		ChunkRadius: 7,
		X0:          -34,
		Y0:          -20,
		X1:          20,
		Y1:          rows + 20,
		Tile:        "grass-1",
	}.Make()

	rigs = rigs[:0]
	sushis = sushis[:0]
	for i, cfg := range rigCfgs {
		buildRig(cfg, i*pitch)
	}
	// Compile everything NOW rather than on the first tick after the save is
	// loaded. See "forcing the flush" in the header.
	auditNow()
	out.Open("init complete: ").U(uint64(len(rigCfgs))).S(" rigs").End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	tick := fkapi.ReadOnTick(ptr).Tick
	rotateSushi(tick)
	harness.Run(schedule, tick)
}

func main() {}
