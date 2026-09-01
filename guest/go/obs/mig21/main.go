// Command bbb-mig21-observer reports what a Factorio 2.0 balancer save looks
// like when it is opened on Factorio 2.1, before and after this mod's migration
// runs on it.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-mig21-observer/control.lua` was, sample for sample and log line
// for log line. See agents/estate-port.md.
//
// It BUILDS NO RIG and it asserts nothing. Every balancer in these saves was
// built by a 2.0 binary this machine no longer has (test/fixtures-2.0/README.md),
// so the world is the fixture's; this mod says what is in it at named moments,
// and test/assert-mig21.py decides whether that is right.
//
// # The one thing it has to get right: WHEN "before" is
//
// The migration does not wait for a tick. The mod's guest heap is declined -- it
// is a different build -- so `fk_migrate` fires from `on_configuration_changed`,
// BEFORE THE FIRST TICK, and by tick 0 the condemned remnants are already down
// and their contents already on the ground. A sample taken from `on_tick` is a
// sample taken afterwards, and the only "before" any script can reach is this
// mod's own `on_configuration_changed`.
//
// WHETHER THAT ONE RUNS FIRST IS FACTORIO'S CHOICE. Handlers run in mod load
// order, and `bbb-mig21-observer` sorts before `better-belt-balancer`, which is
// why this mod deliberately declares NO DEPENDENCY on it -- a dependency would
// put it after. THAT IS A PACKAGING FACT NOW AND NOT A FILE: the Makefile passes
// `--dependency "base >= 2.1.0"` and nothing else, which is what the hand-written
// info.json used to say. Measured, and it does not have to be trusted: if the
// order ever flipped, the seeding below would find nothing to seed and report
// zero, and the assertion script fails on a zero. It cannot pass vacuously.
//
// IT IS ALREADY POST-PRUNING WHATEVER HAPPENS. The ENGINE deletes all but one
// belt-connectable per tile at LOAD, before any script of any mod runs, with no
// log line at all -- measured, m2's 77 part tiles came back with 77 interfaces of
// the ~140 the save was built with. Nothing here can see the world as the 2.0
// binary left it, and the assertions are written knowing that.
//
// # Why it seeds the networks, which is the part that looks like cheating
//
// The fixtures are `--create` saves. A `--create` never reaches a tick, so the
// rigs were built and the save was written with every belt in them EMPTY -- and a
// migration that recovers nothing, spills nothing and conserves nothing trivially
// would satisfy every count in this suite while proving none of them.
//
// So the one moment before the migration is also where the items go in: one item
// into every transport line of every entity this mod's compiler places, on every
// surface. That is not a stand-in for a running balancer's contents, it is better
// than one for this purpose -- it is a KNOWN NUMBER, so "what the teardown
// recovered" can be asserted as an equality against it rather than as a floor.
//
// # The one export the pilot did not have
//
// `fk_on_configuration_changed` is what makes this observer possible at all, and
// it is the only suite in the estate whose whole "before" is that hook. There is
// no `fk_on_init` here and no rig to build.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

const (
	hiddenSurface = "bbb-hidden"
	seedItem      = "iron-plate"
)

// ours is everything the mod's compiler places, wherever it is: the hidden
// network proper and the edge interfaces standing on the visible part tiles.
// Both are drained by a teardown, so both are seeded.
var ours = [...]string{"bbb-linked-belt", "bbb-belt", "bbb-splitter", "bbb-lane-splitter"}

var out = harness.Line{Tag: "[MIG21] "}

// oursFilter is the `name = OURS` list the Lua passed to every
// find_entities_filtered here: a name filter may be a LIST, and the engine
// applies it in C++.
func oursFilter() fkapi.Value {
	vs := make([]fkapi.Value, 0, len(ours))
	for _, n := range ours {
		vs = append(vs, fkapi.OfString(n))
	}
	return fkapi.OfArray(vs...)
}

// surfaces is `pairs(game.surfaces)`, which yields the engine's own order --
// index order in practice, and the order every sample line in this suite is
// recorded in.
func surfaces() []fkapi.EntryValueObject {
	all, err := fkapi.Game.Surfaces()
	if err != nil {
		harness.Fatal("reading game.surfaces", fk.LastError())
		return nil
	}
	return all
}

func findOurs(s fkapi.LuaSurface) []fkapi.Object {
	f := oursFilter()
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Name: &f})
	if err != nil {
		harness.Fatal("finding this mod's entities", fk.LastError())
		return nil
	}
	return found
}

// ---------------------------------------------------------------------------
// the seed
// ---------------------------------------------------------------------------

// seedEntity puts one item on every transport line of one entity.
//
// `insert_at_back` REPORTS WHETHER IT LANDED. An empty line always takes one, so
// on these saves this is one per line and the count is exact; a line that somehow
// already held something at the back is skipped and not counted, which keeps the
// number honest rather than optimistic.
func seedEntity(o fkapi.Object) int64 {
	e := fkapi.LuaEntity{Object: o}
	count, err := e.GetMaxTransportLineIndex()
	if err != nil {
		return 0
	}
	stack := fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(seedItem)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(1)},
	)
	var n int64
	for i := uint32(1); i <= count; i++ {
		line, err := e.GetTransportLine(i)
		if err != nil {
			continue
		}
		ok, err := (fkapi.LuaTransportLine{Object: line}).InsertAtBack(stack, nil)
		if err == nil && ok {
			n++
		}
	}
	return n
}

// lineItems is `#line` summed over one entity's transport lines -- the number of
// items standing on it. `LuaTransportLine.Length` is the bound form of Lua's
// length operator, which is what the Lua used.
func lineItems(o fkapi.Object) int64 {
	e := fkapi.LuaEntity{Object: o}
	count, err := e.GetMaxTransportLineIndex()
	if err != nil {
		return 0
	}
	var n int64
	for i := uint32(1); i <= count; i++ {
		line, err := e.GetTransportLine(i)
		if err != nil {
			continue
		}
		if l, err := (fkapi.LuaTransportLine{Object: line}).Length(); err == nil {
			n += int64(l)
		}
	}
	return n
}

func seedAll() {
	var hidden, visible int64
	for _, entry := range surfaces() {
		s := fkapi.LuaSurface{Object: entry.Val}
		name, err := s.Name()
		if err != nil {
			continue
		}
		for _, o := range findOurs(s) {
			n := seedEntity(o)
			if name == hiddenSurface {
				hidden += n
			} else {
				visible += n
			}
		}
	}
	out.Open("seeded hidden=").I(hidden).S(" visible=").I(visible).
		S(" total=").I(hidden + visible).End()
}

// ---------------------------------------------------------------------------
// the samples
// ---------------------------------------------------------------------------

// delivered is WHAT THE RIGS HAVE DELIVERED, which only matters on one engine
// and is reported on both.
//
// On Factorio 2.1 every balancer in these fixtures is refused and its remnant
// torn down, so this number moves only because the rigs' own bare control belts
// and pass-through lines are still running -- it is reported and not asserted. On
// 2.0 nothing is torn down at all: the clusters are ADOPTED whole and the
// grandfather pass keeps them working, and "keeps working" has to mean items
// arriving somewhere rather than merely entities standing. Every sink in these
// worlds is an ordinary chest; the infinity chests are the SOURCES and are
// excluded, because their contents are held at a filter level and say nothing.
func delivered() int64 {
	var total int64
	container := fkapi.OfString("container")
	for _, entry := range surfaces() {
		s := fkapi.LuaSurface{Object: entry.Val}
		found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Type: &container})
		if err != nil {
			continue
		}
		for _, o := range found {
			if n := harness.InventoryTotal(o); n > 0 {
				total += n
			}
		}
	}
	return total
}

// groundItems totals every `item-on-ground` stack on one surface.
//
// NO AREA, which means the WHOLE surface. This observer builds no world and has
// no box to sweep: the fixtures' rigs are wherever a 2.0 binary put them, and a
// spill can land anywhere `spill_item_stack` reaches.
func groundItems(s fkapi.LuaSurface) int64 {
	name := fkapi.OfString("item-on-ground")
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{Name: &name})
	if err != nil {
		harness.Fatal("sweeping the ground", fk.LastError())
		return 0
	}
	var ground int64
	for _, o := range found {
		st, err := (fkapi.LuaEntity{Object: o}).Stack()
		if err != nil {
			continue
		}
		if n, err := (fkapi.LuaItemStack{Object: st}).Count(); err == nil {
			ground += int64(n)
		}
	}
	return ground
}

func sample(tag string) {
	var totParts, totIface, totStacked, totGround int64
	var totHidden, totHitems, totVitems int64

	for _, entry := range surfaces() {
		s := fkapi.LuaSurface{Object: entry.Val}
		name, err := s.Name()
		if err != nil {
			continue
		}
		partName := fkapi.OfString("bbb-balancer-part")
		parts, err := s.CountEntitiesFiltered(fkapi.EntitySearchFilters{Name: &partName})
		if err != nil {
			harness.Fatal("counting parts on "+name, fk.LastError())
		}
		found := findOurs(s)
		ground := groundItems(s)

		if name == hiddenSurface {
			// The hidden surface: the networks themselves, and what is standing in
			// their transport lines. That second number is where a teardown's
			// recovered items come FROM, and it is what has to reach zero for a
			// network this mod has condemned.
			var items int64
			for _, o := range found {
				items += lineItems(o)
			}
			totHidden += int64(len(found))
			totHitems += items
			out.Open("tag=").S(tag).S(" surface=").S(name).
				S(" hidden_entities=").I(int64(len(found))).
				S(" hidden_items=").I(items).
				S(" ground=").I(ground).End()
			continue
		}

		// A visible surface: the parts a player can see, the edge interfaces
		// standing on their tiles, and whether any TILE carries two
		// belt-connectables -- which is the thing 2.1 forbids and the engine has
		// already dealt with by the time anything here can look.
		var stacked, worst, items int64
		tiles := tileCounts{}
		for _, o := range found {
			pos, err := (fkapi.LuaEntity{Object: o}).Position()
			if err != nil {
				continue
			}
			tiles.add(floorInt(pos.X), floorInt(pos.Y))
			items += lineItems(o)
		}
		for _, n := range tiles.n {
			if n > 1 {
				stacked++
			}
			if int64(n) > worst {
				worst = int64(n)
			}
		}
		totParts += int64(parts)
		totIface += int64(len(found))
		totStacked += stacked
		totGround += ground
		totVitems += items
		out.Open("tag=").S(tag).S(" surface=").S(name).
			S(" parts=").I(int64(parts)).
			S(" ours=").I(int64(len(found))).
			S(" stacked_tiles=").I(stacked).
			S(" worst_per_tile=").I(worst).
			S(" iface_items=").I(items).
			S(" ground=").I(ground).End()
	}

	out.Open("total tag=").S(tag).
		S(" parts=").I(totParts).
		S(" ours=").I(totIface).
		S(" stacked_tiles=").I(totStacked).
		S(" ground=").I(totGround).
		S(" hidden_entities=").I(totHidden).
		S(" hidden_items=").I(totHitems).
		S(" iface_items=").I(totVitems).
		S(" delivered=").I(delivered()).End()
}

// tileCounts is the Lua's `by_tile` table: how many of this mod's entities stand
// on each tile of one surface.
//
// A SLICE AND A LINEAR SCAN, not a map, for the reason harness.Run is one: the
// biggest fixture here has 95 part tiles, the answer is a MULTISET the caller
// folds to two numbers, and a slice is the shape whose determinism needs no
// argument.
type tileCounts struct {
	xy []harness.XY
	n  []int
}

func (t *tileCounts) add(x, y int) {
	for i, p := range t.xy {
		if p.X == x && p.Y == y {
			t.n[i]++
			return
		}
	}
	t.xy = append(t.xy, harness.XY{X: x, Y: y})
	t.n = append(t.n, 1)
}

// floorInt is `math.floor` on a coordinate. A tile centre is x.5, so a plain
// truncation would be wrong for the negative half of every fixture.
func floorInt(v float64) int {
	i := int(v)
	if v < 0 && float64(i) != v {
		i--
	}
	return i
}

// ---------------------------------------------------------------------------
// the audit
// ---------------------------------------------------------------------------

// audit places the shipped marker, which is the only SYNCHRONOUS "re-classify the
// world and report" trigger there is: placing one drains the mod's deferred queue
// inside this dispatch, so the `[BBB] audit` line it writes describes the world
// at this tick rather than at some tick after it.
//
// SURFACE 1, TILE (0,0), and neither is this observer's choice: it builds no
// world, so the only surface it can be sure of is the fixture's first.
func audit(tag string) {
	all := surfaces()
	if len(all) == 0 {
		harness.Fatal("no surfaces to audit on", fk.LastError())
		return
	}
	harness.Audit(fkapi.LuaSurface{Object: all[0].Val}, 0, 0)
	out.Open("audited tag=").S(tag).End()
}

// ---------------------------------------------------------------------------
// the chart tripwire
// ---------------------------------------------------------------------------

// chartState is WHETHER THE FORCE CAN SEE THE GROUND ITS BALANCERS STAND ON, AND
// THE WALL THAT MAKES THAT UNANSWERABLE HERE.
//
// A `[gps=]` is a coordinate and nothing else: clicking one opens the map there
// whether or not the force has charted it, and an uncharted point is BLACK. So
// the mod charts what it pings, and the obvious check is `is_chunk_charted` after
// the message.
//
// IT ANSWERS FALSE FOR EVERYTHING ON A HEADLESS RUN. Measured and not assumed:
// with no players, `force.chart` charts nothing, `force.chart_all` over a fully
// generated surface charts nothing, a radar charts nothing, and NAUVIS'S OWN
// ORIGIN CHUNK -- which every real game charts at world creation -- reads
// uncharted too. A force with no players has no chart to write into. That puts
// the EFFECT behind the same player wall as the flying text and the hand-back;
// what is on this side of it is the mod's own `charted N from x,y to x,y`.
//
// SO THIS IS A TRIPWIRE, NOT A MEASUREMENT OF THE FIX: it reports zero before and
// zero after, and test/assert-mig21.py asserts exactly that -- so the day a
// Factorio charts headlessly the run fails and asks for the real assertion
// instead of this one.
//
// ONE CHUNK PER PART TILE, counted per surface. `is_chunk_charted` takes a CHUNK
// position, so a part at tile (x, y) is asked about at (floor(x/32),
// floor(y/32)); several parts share a chunk and the count is over DISTINCT
// chunks, which is what makes the samples comparable.
func chartState(tag string) {
	forces, err := fkapi.Game.Forces()
	if err != nil {
		harness.Fatal("reading game.forces", fk.LastError())
		return
	}
	players, err := fkapi.Game.Players()
	if err != nil {
		harness.Fatal("reading game.players", fk.LastError())
		return
	}
	nauvis, err := fkapi.Game.GetSurface(fkapi.OfString("nauvis"))
	if err != nil || nauvis == nil {
		harness.Fatal("no nauvis", fk.LastError())
		return
	}

	for _, fe := range forces {
		force := fkapi.LuaForce{Object: fe.Val}
		index, err := force.Index()
		if err != nil {
			continue
		}

		// Build the per-surface list first: the line is only written for a force
		// that owns a part somewhere, which is the Lua's `#per_surface > 0`.
		type perSurface struct {
			name              string
			charted, distinct int
		}
		var list []perSurface
		for _, entry := range surfaces() {
			s := fkapi.LuaSurface{Object: entry.Val}
			name, err := s.Name()
			if err != nil || name == hiddenSurface {
				continue
			}
			partName := fkapi.OfString("bbb-balancer-part")
			forceVal := fkapi.OfObject(fe.Val)
			found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{
				Name: &partName, Force: &forceVal,
			})
			if err != nil {
				continue
			}
			seen := tileCounts{}
			charted := 0
			for _, o := range found {
				pos, err := (fkapi.LuaEntity{Object: o}).Position()
				if err != nil {
					continue
				}
				cx, cy := floorInt(pos.X/32), floorInt(pos.Y/32)
				before := len(seen.xy)
				seen.add(cx, cy)
				if len(seen.xy) == before {
					continue // a chunk already asked about
				}
				ok, err := force.IsChunkCharted(entry.Val,
					fkapi.ChunkPosition{X: int32(cx), Y: int32(cy)})
				if err == nil && ok {
					charted++
				}
			}
			if n := len(seen.xy); n > 0 {
				list = append(list, perSurface{name: name, charted: charted, distinct: n})
			}
		}
		if len(list) == 0 {
			continue
		}

		// THE CONTROL, in the same line: nauvis's origin chunk, generated by world
		// creation and charted by nothing, for the same force.
		nauSurface := fkapi.LuaSurface{Object: *nauvis}
		origin := fkapi.ChunkPosition{X: 0, Y: 0}
		nauOK := false
		if gen, err := nauSurface.IsChunkGenerated(origin); err == nil && gen {
			if c, err := force.IsChunkCharted(*nauvis, origin); err == nil {
				nauOK = c
			}
		}

		out.Open("chart tag=").S(tag).S(" force=").U(uint64(index))
		for _, p := range list {
			out.S(" ").S(p.name).S(":").I(int64(p.charted)).S("/").I(int64(p.distinct))
		}
		out.S(" nauvis_origin=").B(nauOK).S(" players=").I(int64(len(players))).End()
	}
}

// hiddenState logs, per force, whether the mod's hidden surface is withheld
// from that force's surface lists -- what keeps it out of Space Age's remote
// view (issue #1). Visibility is a per-force flag defaulting to VISIBLE, and
// these fixtures are the one unfakeable specimen of the unfixed state: both
// were saved by a guest that predates the fix, so at `cfg` -- before the mod's
// migration has run -- every force must read hidden=false, which is what
// proves the probe can see the state the fix repairs. At `final` the fresh
// heap's rebuild has run and every force must read hidden=true. The judgement
// is assert-mig21.py's; the before/after pair is what makes it non-vacuous.
func hiddenState(tag string) {
	hs, err := fkapi.Game.GetSurface(fkapi.OfString(hiddenSurface))
	if err != nil || hs == nil {
		out.Open("surface-hidden tag=").S(tag).S(" surface=absent").End()
		return
	}
	forces, err := fkapi.Game.Forces()
	if err != nil {
		harness.Fatal("reading game.forces", fk.LastError())
		return
	}
	for _, fe := range forces {
		f := fkapi.LuaForce{Object: fe.Val}
		index, err := f.Index()
		if err != nil {
			continue
		}
		hid, err := f.GetSurfaceHidden(*hs)
		if err != nil {
			harness.Fatal("get_surface_hidden", fk.LastError())
			return
		}
		out.Open("surface-hidden tag=").S(tag).S(" force=").U(uint64(index)).
			S(" hidden=").B(hid).End()
	}
}

// ---------------------------------------------------------------------------

func init() { fkapi.Subscribe(fkapi.EventOnTick) }

// onConfigurationChanged is BEFORE, as near as any script can get to it -- and
// the one moment the networks can be given something to lose. See the header for
// both.
//
//go:wasmexport fk_on_configuration_changed
func onConfigurationChanged() {
	seedAll()
	sample("cfg")
	chartState("cfg")
	hiddenState("cfg")
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	switch fkapi.ReadOnTick(ptr).Tick {
	case 1:
		// The migration's own flush lands on the first deferred tick, so this is
		// the first moment everything it does has happened.
		sample("t1")
		chartState("t1")
		audit("t1")
	case 2:
		// One tick later, because the audit above forces a flush of its own and
		// what matters is that the state settled rather than that the audit ran.
		sample("post-audit")
	case 300:
		// And a long way after nothing has been touched: a refused cluster has to
		// be a STABLE state, not one that oscillates between teardown and rebuild.
		sample("final")
		chartState("final")
		hiddenState("final")
		audit("final")
	}
}

func main() {}
