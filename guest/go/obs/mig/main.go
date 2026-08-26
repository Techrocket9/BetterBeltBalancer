// Command bbb-mig-test builds a Belt Balancer 2 shaped world in phase one and
// reports, in phase two, what survived the swap.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-mig-test/control.lua` was, rig for rig and log line for log
// line, with its data stage compiled too (obs/migdata). See
// agents/estate-port.md.
//
// It ASSERTS NOTHING. test/assert-mig.py decides whether the numbers are right;
// an observer that computed the expected answer would be a second implementation
// of the thing under test.
//
// # It is present in BOTH phases and Better Belt Balancer is not
//
// That is the shape of the suite: `fk_on_init` runs once, in the phase where the
// incumbent owns `balancer-part`, and everything after it runs on the far side
// of a mod swap. So nothing in `fk_on_init` may name a Better Belt Balancer
// prototype, and everything that does is guarded on the prototype existing.
//
// THE GUARD IS NOT TIDINESS: `find_entities_filtered{name = ...}` RAISES for a
// prototype the game does not have, and `get_item_count` on such a name does
// too. In phase one of the added-as-removed leg `bbb-balancer-part` is not a
// prototype at all, and after the swap `balancer-part` is only a prototype
// because this mod's data stage kept it alive.
//
// AND THE DEPENDENCY LIST IS PART OF IT. Six optional dependencies and not one
// required one beyond base -- a hard dependency on the mod under test would
// refuse the load in exactly the phase `fk_on_init` runs in, and an optional one
// is not a no-op: it is what puts this observer AFTER whichever incumbent is
// installed in Factorio's load order, so a prototype lookup resolves to whoever
// really owns the name. See the Makefile recipe.
//
// # What the rigs are laid in is the incumbent's own idiom
//
// ...and on Factorio 2.1 that is the point. Belt Balancer 2 and 3 put a belt on
// every free face of a part, so their 4-in/4-out balancer is FOUR parts each
// carrying two belts -- and 2.1 allows one belt per balancer part
// (agents/single-edge.md). Every rig laid that way converts and is then REFUSED,
// which is not a defect in the migration: the incumbent's geometry cannot
// function on 2.1 under any design this mod could have, so what a player gets is
// their parts, their items and a rebuild checklist.
//
// WHICH IS WHY THE `sok` BAND EXISTS. A Belt Balancer user whose balancer
// happens to be one belt per part -- a two-column block, inputs down the west
// column and outputs down the east -- has a shape 2.1 can build, and theirs
// converts into a WORKING network. It is the only place this suite still proves
// that an adopted balancer balances at all, and it is the honest portal story in
// one world: some of your balancers keep working, the rest do not.
//
// # The rigs, one per band on a flat scratch surface
//
// Every part is placed as `balancer-part`, because in phase one that is the only
// balancer prototype in the game.
//
//	ctrl    a bare express belt, chest to chest. The yardstick: whatever this
//	        delivers in the sample window is what one saturated belt is worth.
//	m4x4    4 parts, 4 belts in, 4 belts out, saturated -- TWO BELTS ON EVERY
//	        PART. The shape a migrating player is most likely to have, and on 2.1
//	        the shape that stops.
//	m3to5   5 parts, 3 in and 5 out. Three of its five parts carry two belts and
//	        two carry one, which is why it is still here now that it can never
//	        deliver: the refusal names a COUNT of offending parts per cluster, and
//	        a rig whose count is neither zero nor the whole cluster is the only
//	        thing that separates a real classification from a constant.
//	sok2    THE SINGLE-EDGE BAND, first of two. 2 in and 2 out over FOUR parts in
//	        two columns, and no part carries two belts. A shape a Belt Balancer
//	        user could genuinely have, and the one that must come out of the
//	        conversion RUNNING.
//	sok4    the same idiom at 4 in and 4 out over eight parts, P=4. Two of them
//	        rather than one because one rig delivering its rate could be an
//	        accident of a network with no stages in it.
//	wit     THE CONSERVATION WITNESS. 2 parts, 2 in and 2 out, and NO source and
//	        NO sink at all -- its belts are hand-loaded with COPPER PLATE and
//	        nothing can add to them or take from them. Every other rig in this
//	        save runs iron plate, so a count of copper across every surface is
//	        exactly this rig's contents, before the swap and after it.
//	fid     THE FIDELITY RIG. 2 parts, belts either side, and nothing feeding it.
//	        One part is DAMAGED and the other is built at UNCOMMON quality, and
//	        those are the only two properties `legacyConvertOne` reads off the old
//	        entity and writes onto the new one. Until this rig existed every part
//	        in every leg was undamaged and normal, so both lines could have been
//	        absent and every assertion in this suite would still have passed.
//	frc     THE FORCE RIG: FOUR PARTS IN ONE COLUMN, the top two on the player
//	        force and the bottom two on a second force, TOUCHING. Clusters are per
//	        force, so that is two balancers; a flood fill that lost its force
//	        check fuses them into one, silently, with every item count in this
//	        suite unmoved.
//
// And a SECOND SURFACE, `bbb-mig-b`, carrying two more parts and their belts.
// `legacyScan` walks every surface in index order; until that rig existed every
// part in this suite stood on one surface, so a scan that stopped after the first
// surface it converted anything on would have passed every leg.
//
// Plus a steel chest holding a stack of the incumbent's ITEM, which is the other
// half of what a removed mod takes with it.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

const (
	legacyPart  = "balancer-part"
	bbbPart     = "bbb-balancer-part"
	auditMarker = "bbb-audit"
	belt        = "express-transport-belt"

	witnessItem = "copper-plate"
	flowItem    = "iron-plate"

	surfA = "bbb-mig-a"
	surfB = "bbb-mig-b"
	pitch = 12
	halfX = 20

	// forceB is the second force, and it is created BEFORE any part of its own is
	// placed so that it is there for the whole life of the save and every
	// per-force report can name it without a guard.
	forceB = "bbb-mig-force-b"

	// What the fidelity rig is built with. The health is a real wound rather than
	// a round number for its own sake -- it has to be BELOW max_health, or an
	// equality across the swap is satisfied by a guest that copies nothing at all.
	fidHealth  = 85
	fidQuality = "uncommon"

	// THE LATE BUILD's tile: well clear of every rig, and placed after the final
	// audit so no count, no rate and no audit in this suite moves because of it.
	lateX, lateY = 12, 0
)

var out = harness.Line{Tag: "[BBB-MIG] "}

var east uint32

// forces is the list every per-force report walks: a written-down order rather
// than a walk over `game.forces`, which is a hash. Nothing here is host-visible,
// but a report line whose columns swapped between runs would be read by an
// assertion.
var forces = []string{harness.PlayerForce, forceB}

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func surface(name string) fkapi.LuaSurface { return harness.Surface(name) }

func put(s fkapi.LuaSurface, name string, x, y int, dir *uint32, force, quality string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Force: force, Quality: quality, Raise: true,
	})
}

func belts(s fkapi.LuaSurface, from, to, y int, force string) {
	for x := from; x <= to; x++ {
		put(s, belt, x, y, &east, force, "")
	}
}

func source(s fkapi.LuaSurface, x, y int) {
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: x, Y: y})
	harness.InfinityFilter(c, flowItem, "", 1000)
	harness.Place(s, harness.Piece{
		Name: protos.MigLoader, X: x + 1, Y: y, Dir: &east, Type: "output", Raise: true,
	})
}

func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	harness.Place(s, harness.Piece{
		Name: protos.MigLoader, X: x, Y: y, Dir: &east, Type: "input", Raise: true,
	})
	harness.Place(s, harness.Piece{Name: "steel-chest", X: x + 1, Y: y})
	return harness.XY{X: x + 1, Y: y}
}

// auditNow is the shipped mod's synchronous drain, and it is the one thing here
// that is only ever available in the phase where that mod IS installed -- which
// is the coexistence leg's phase one. It is what makes that leg's "nothing was
// converted" a statement about a guest that was asked to look.
func auditNow() {
	if !harness.EntityProtoExists(auditMarker) {
		out.Open("audit unavailable (this mod is not installed in this phase)").End()
		return
	}
	harness.Audit(surface(surfA), -16, 0)
}

func makeSurface(name string, rows int, radius uint32) fkapi.LuaSurface {
	return harness.Flat{
		Name:        name,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		ChunkRadius: radius,
		X0:          -halfX,
		Y0:          -8,
		X1:          halfX,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()
}

// ---------------------------------------------------------------------------
// counting
//
// Over EVERY surface, so the hidden one this mod's compiler works on is included
// the moment it exists -- and it does not exist at all in the phase where the
// incumbent is installed, which is exactly why the count has to be written this
// way rather than against a named pair of surfaces.
// ---------------------------------------------------------------------------

func countItem(name string) int64 {
	var total int64
	for _, s := range harness.SurfacesByIndex() {
		for _, e := range harness.EntitiesOfForce(s, harness.PlayerForce) {
			if harness.EntityType(e) == "item-entity" {
				if n, c, ok := harness.GroundStack(e); ok && n == name {
					total += c
				}
				continue
			}
			total += harness.ItemCountIn(e, name)
		}
	}
	return total
}

// countNamed is how many of each balancer prototype are standing, anywhere. The
// migration's own headline in one line: `balancer-part` must reach zero and
// `bbb-balancer-part` must reach the number that were there.
func countNamed(name string) int {
	if !harness.EntityProtoExists(name) {
		return 0
	}
	n := 0
	for _, s := range harness.SurfacesByIndex() {
		n += harness.CountNamedOn(s, name, "")
	}
	return n
}

func countNamedOn(s fkapi.LuaSurface, name, force string) int {
	if !harness.EntityProtoExists(name) {
		return 0
	}
	return harness.CountNamedOn(s, name, force)
}

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

// reportSurfaces says WHERE the parts are, which the whole-world census cannot.
// The summary line the guest writes counts surfaces SCANNED rather than surfaces
// that had anything on them, so a scan that stopped after the first surface it
// converted something on would report the same number it reports now. This is
// the line that can see it.
func reportSurfaces(phase string) {
	out.Open("surfaces phase=").S(phase)
	for _, s := range harness.SurfacesByIndex() {
		name, err := s.Name()
		if err != nil {
			continue
		}
		out.S(" ").S(name).S(":").I(int64(countNamedOn(s, legacyPart, ""))).
			S("/").I(int64(countNamedOn(s, bbbPart, "")))
	}
	out.End()
}

// reportForceParts says WHOSE the parts are: the anti-vacuity half of the force
// rig, because a run where the second force's parts were never built would
// satisfy "two forces stayed two clusters" by having only one force in it.
func reportForceParts(phase string) {
	out.Open("force-parts phase=").S(phase)
	surfs := harness.SurfacesByIndex()
	for _, fname := range forces {
		n := 0
		for _, s := range surfs {
			n += countNamedOn(s, legacyPart, fname) + countNamedOn(s, bbbPart, fname)
		}
		out.S(" ").S(fname).S("=").I(int64(n))
	}
	out.End()
}

// partAt is the part standing on one tile, whichever prototype it is now. Both
// names are guarded, and the name is REPORTED rather than assumed: which of the
// two is there is exactly what the fidelity rig's tiles are being asked.
func partAt(x, y int) (fkapi.Object, string, bool) {
	s := surface(surfA)
	for _, name := range []string{legacyPart, bbbPart} {
		if !harness.EntityProtoExists(name) {
			continue
		}
		found := harness.EntitiesIn(s, harness.InnerBox(x, y), name)
		if len(found) > 0 {
			return found[0], name, true
		}
	}
	return fkapi.Object{}, "none", false
}

// fidTiles is where the fidelity rig's two special parts stand, written down at
// build time rather than into a constant -- the rig bases are computed.
var fidHealthTile, fidQualityTile harness.XY

// reportFidelity is THE TWO PROPERTIES THE CONVERSION CARRIES.
// `legacyConvertOne` reads the health and the quality off the entity it is about
// to destroy and writes them onto the one it creates; both are one call each and
// both are invisible on an undamaged normal-quality part, which is what every
// other part in every leg is.
func reportFidelity(phase string) {
	e, name, ok := partAt(fidHealthTile.X, fidHealthTile.Y)
	out.Open("health phase=").S(phase).S(" name=").S(name).S(" value=")
	if !ok {
		out.S("none").S(" max=none").End()
	} else {
		h, herr := (fkapi.LuaEntity{Object: e}).Health()
		if herr != nil || h == nil {
			out.S("none")
		} else {
			out.F1(float64(*h))
		}
		out.S(" max=")
		if m, err := (fkapi.LuaEntity{Object: e}).MaxHealth(); err == nil {
			out.F1(float64(m))
		} else {
			out.S("none")
		}
		out.End()
	}

	e, name, ok = partAt(fidQualityTile.X, fidQualityTile.Y)
	out.Open("quality phase=").S(phase).S(" name=").S(name).S(" value=")
	if !ok {
		out.S("none").End()
		return
	}
	q, err := (fkapi.LuaEntity{Object: e}).Quality()
	if err != nil {
		out.S("none").End()
		return
	}
	n, err := (fkapi.LuaQualityPrototype{Object: q}).Name()
	if err != nil {
		out.S("none").End()
		return
	}
	out.S(n).End()
}

func census(phase string) {
	out.Open("census phase=").S(phase).S(" ").S(legacyPart).S("=").
		I(int64(countNamed(legacyPart))).S(" ").S(bbbPart).S("=").
		I(int64(countNamed(bbbPart))).End()
}

// itemChest is where the stack of the incumbent's item lives.
var itemChest harness.XY

// reportItem is the item half. A stack of the incumbent's item in a chest is
// deleted with its prototype exactly as an entity is, and `place_result` is what
// makes a surviving stack useful rather than merely present.
func reportItem(phase string) {
	held := int64(-1)
	place := "nil"
	// GUARDED ON THE PROTOTYPE, and not for tidiness: without the data stage's
	// stub there is no `balancer-part` item at all after the swap, and
	// `get_item_count` on a name the game does not have RAISES. That is the
	// failure this suite is red-proved against, so it has to be reportable rather
	// than a crash.
	if harness.ItemProtoExists(legacyPart) {
		if c, ok := harness.FindOnTile(surface(surfA), "steel-chest", itemChest.X, itemChest.Y); ok {
			v := fkapi.OfString(legacyPart)
			if n, err := (fkapi.LuaControl{Object: c}).GetItemCount(&v); err == nil {
				held = int64(n)
			}
		}
		if r := harness.ItemPlaceResult(legacyPart); r != "" {
			place = r
		}
	} else {
		held = 0
	}
	out.Open("legacy-item phase=").S(phase).S(" held=").I(held).
		S(" place_result=").S(place).End()
}

func techWord(l *harness.Line, force fkapi.Object, ok bool, name string) {
	if !ok {
		l.S("absent")
		return
	}
	done, present := harness.Researched(force, name)
	if !present {
		l.S("absent")
		return
	}
	l.B(done)
}

func reportTech(phase string) {
	f, ok := harness.ForceByName(harness.PlayerForce)
	out.Open("tech phase=").S(phase).S(" bbb-balancer=")
	techWord(&out, f, ok, "bbb-balancer")
	out.S(" belt-balancer-1=")
	techWord(&out, f, ok, "belt-balancer-1")
	out.End()

	// THE SECOND FORCE, on a line of its own so the one above keeps the exact
	// shape every assertion in this suite already reads. `legacyScan` grants the
	// technology PER FORCE that owned a converted part, and a force left without
	// it is a player holding balancers they cannot craft a spare for -- which is
	// the thing the grant exists to prevent, one force further along than any leg
	// could see before the force rig existed.
	g, gok := harness.ForceByName(forceB)
	out.Open("tech-force phase=").S(phase).S(" force=").S(forceB).S(" bbb-balancer=")
	techWord(&out, g, gok, "bbb-balancer")
	out.End()
}

func reportCounts(phase string) {
	out.Open("count phase=").S(phase).S(" ").S(witnessItem).S("=").
		I(countItem(witnessItem)).End()
}

// lateBuild places one `balancer-part` in phase two, well clear of every rig,
// long after any scan has run.
//
// It is the BUILD path rather than the scan, and it is the one probe that can
// tell "this game's balancer-part is mine" apart from "somebody else's". A
// migrated save must swap it; a save where a stranger owns the prototype must
// leave it exactly where it is.
func lateBuild() {
	if !harness.EntityProtoExists(legacyPart) {
		out.Open("late-build unavailable").End()
		return
	}
	_, made := harness.PlaceSoft(surface(surfA), harness.Piece{
		Name: legacyPart, X: lateX, Y: lateY, Raise: true,
	})
	out.Open("late-build placed=").B(made).End()
}

func lateProbe() {
	s := surface(surfA)
	area := harness.Box(lateX-0.1, lateY-0.1, lateX+1.1, lateY+1.1)
	legacy, ours := 0, 0
	if harness.EntityProtoExists(legacyPart) {
		legacy = len(harness.EntitiesIn(s, area, legacyPart))
	}
	if harness.EntityProtoExists(bbbPart) {
		ours = len(harness.EntitiesIn(s, area, bbbPart))
	}
	out.Open("late-build legacy=").I(int64(legacy)).S(" ours=").I(int64(ours)).End()
}

// ---------------------------------------------------------------------------
// the rigs
// ---------------------------------------------------------------------------

type rigCfg struct {
	Name       string
	Parts      int
	Ins, Outs  int
	Rows       int
	SingleEdge bool
}

var rigCfgs = []rigCfg{
	{Name: "ctrl"},
	{Name: "m4x4", Parts: 4, Ins: 4, Outs: 4},
	{Name: "m3to5", Parts: 5, Ins: 3, Outs: 5},
	{Name: "sok2", Rows: 2, SingleEdge: true},
	{Name: "sok4", Rows: 4, SingleEdge: true},
}

// rig is one built band and the tiles of the chests its outputs drain into, in
// REGISTRATION ORDER -- which IS the order the sample line reports in, and the
// order test/assert-mig.py parses.
type rig struct {
	Name   string
	Chests []harness.XY
}

var rigs []rig

// buildRig lays THE INCUMBENT'S OWN IDIOM: one column of parts, belts on both
// free faces of every row. A part with an input on its west face and an output
// on its east face carries TWO belts, which is what Belt Balancer builds and
// what Factorio 2.1 forbids -- so on 2.1 everything built by this function
// converts and is then refused. That is deliberate and it is what a real
// incumbent save looks like; buildSingle below is the other half of the story.
func buildRig(cfg rigCfg, base int) rig {
	s := surface(surfA)
	r := rig{Name: cfg.Name}
	if cfg.Name == "ctrl" {
		source(s, -5, base)
		belts(s, -3, 3, base, "")
		r.Chests = append(r.Chests, sink(s, 4, base))
		return r
	}
	// Parts first, belts after: the belt-adjacency trigger is then on the critical
	// path of every rig rather than only of some.
	for i := 0; i < cfg.Parts; i++ {
		put(s, legacyPart, 0, base+i, nil, "", "")
	}
	for i := 0; i < cfg.Ins; i++ {
		source(s, -5, base+i)
		belts(s, -3, -1, base+i, "")
	}
	for i := 0; i < cfg.Outs; i++ {
		belts(s, 1, 3, base+i, "")
		r.Chests = append(r.Chests, sink(s, 4, base+i))
	}
	return r
}

// buildSingle is THE SINGLE-EDGE BAND: the same balancer laid in TWO COLUMNS,
// which is a shape a Belt Balancer user could genuinely have and which Factorio
// 2.1 can build.
//
// A west part's west face takes the row's input and its east face is INTERIOR
// (the east part is there); an east part's west face is interior and its east
// face gives the row's output. Every other face of every part is bare ground. So
// each part carries exactly one belt, N and M are the same N and M the
// one-column rig would have had, and `P = next_pow2(max(N, M))` is unmoved --
// the belts decide the network and the belts did not move.
//
// The gap either side is one tile narrower than the one-column rigs' because the
// block is one tile wider: sources and sinks stay where they are, so a count of
// items in a chest is comparable with `ctrl`'s to the item.
func buildSingle(cfg rigCfg, base int) rig {
	s := surface(surfA)
	r := rig{Name: cfg.Name}
	for i := 0; i < cfg.Rows; i++ {
		put(s, legacyPart, 0, base+i, nil, "", "")
		put(s, legacyPart, 1, base+i, nil, "", "")
	}
	for i := 0; i < cfg.Rows; i++ {
		source(s, -5, base+i)
		belts(s, -3, -1, base+i, "")
		belts(s, 2, 3, base+i, "")
		r.Chests = append(r.Chests, sink(s, 4, base+i))
	}
	return r
}

// buildWitness is THE WITNESS. Two parts, two belts in and two out, nothing
// feeding it and nothing draining it, hand-loaded with copper plate.
//
// `insert_at` is used rather than `insert_at_back` for the reason this mod's own
// carry path uses it: the back of a line is ONE position and accepts one item
// per tick, where a named position fills the line the way a compressed belt is
// filled.
func buildWitness(base int) {
	s := surface(surfA)
	for i := 0; i <= 1; i++ {
		put(s, legacyPart, 0, base+i, nil, "", "")
	}
	loaded := 0
	stack := fkapi.OfMap(
		fkapi.KeyValue{Key: fkapi.OfString("name"), Val: fkapi.OfString(witnessItem)},
		fkapi.KeyValue{Key: fkapi.OfString("count"), Val: fkapi.OfNumber(1)},
	)
	for i := 0; i <= 1; i++ {
		for x := -3; x <= -1; x++ {
			b := put(s, belt, x, base+i, &east, "", "")
			e := fkapi.LuaEntity{Object: b}
			for lane := uint32(1); lane <= 2; lane++ {
				line, err := e.GetTransportLine(lane)
				if err != nil {
					continue
				}
				for k := 0; k <= 3; k++ {
					ok, err := (fkapi.LuaTransportLine{Object: line}).
						InsertAt(float32(k)*0.25, stack, nil)
					if err == nil && ok {
						loaded++
					}
				}
			}
		}
		for x := 1; x <= 3; x++ {
			put(s, belt, x, base+i, &east, "", "")
		}
	}
	out.Open("witness loaded=").I(int64(loaded)).End()
}

// buildFidelity is THE FIDELITY RIG. Two parts and four belts, nothing feeding
// it: what is measured here is not a rate. One part is DAMAGED and the other is
// created at UNCOMMON quality, so the two properties `legacyConvertOne` carries
// across the swap are both non-default on exactly one tile each.
//
// IT HAS BELTS EITHER SIDE and it needs them: a cluster with no inputs or no
// outputs is a legitimate half-built state that compiles to nothing, and the
// audit's `nets == clusters` would then read as a cluster the classifier never
// saw. Every rig this suite adds carries one belt in and one belt out per part
// for that reason, fed or not.
func buildFidelity(base int) {
	s := surface(surfA)
	hurt := put(s, legacyPart, 0, base, nil, "", "")
	if err := (fkapi.LuaEntity{Object: hurt}).SetHealth(fidHealth); err != nil {
		harness.Fatal("damaging the fidelity part", legacyPart)
	}
	// Guarded on the quality existing rather than on the mod list: `mig_list`
	// enables the quality mod for both phases of every leg, and a guard is what
	// turns a mod list that stopped doing so into a failed assertion instead of a
	// failed script.
	quality := ""
	if harness.QualityProtoExists(fidQuality) {
		quality = fidQuality
	}
	put(s, legacyPart, 0, base+1, nil, "", quality)
	for i := 0; i <= 1; i++ {
		put(s, belt, -1, base+i, &east, "", "")
		put(s, belt, 1, base+i, &east, "", "")
	}
	fidHealthTile = harness.XY{X: 0, Y: base}
	fidQualityTile = harness.XY{X: 0, Y: base + 1}
}

// buildForces is THE FORCE RIG. Four parts in one column, the top two on the
// player force and the bottom two on a second force, so the pair in the middle
// TOUCHES across a force boundary. Two forces' parts touching are two balancers
// -- the flood fill, the compiler's own fill and the edge search all agree about
// it -- and a fusion is invisible to every count in this suite: the same parts,
// the same items, one fewer cluster.
func buildForces(base int) {
	s := surface(surfA)
	for i := 0; i <= 1; i++ {
		put(s, legacyPart, 0, base+i, nil, "", "")
		put(s, belt, -1, base+i, &east, "", "")
		put(s, belt, 1, base+i, &east, "", "")
	}
	for i := 2; i <= 3; i++ {
		put(s, legacyPart, 0, base+i, nil, forceB, "")
		put(s, belt, -1, base+i, &east, forceB, "")
		put(s, belt, 1, base+i, &east, forceB, "")
	}
}

// buildSurfaceB is THE SECOND SURFACE. `legacyScan` walks every surface in index
// order and the summary line it writes counts surfaces SCANNED, so a scan that
// stopped after the first surface it converted something on would report the
// number it reports now and leave these two parts standing as the incumbent's
// forever.
func buildSurfaceB() {
	// rows = 4, so ceil(4/32) + 3 = 4.
	s := makeSurface(surfB, 4, 4)
	for i := 0; i <= 1; i++ {
		put(s, legacyPart, 0, i, nil, "", "")
		put(s, belt, -1, i, &east, "", "")
		put(s, belt, 1, i, &east, "", "")
	}
}

// ---------------------------------------------------------------------------
// the schedule
// ---------------------------------------------------------------------------

func report(tick uint64) {
	s := surface(surfA)
	out.Open("sample tick=").U(tick)
	for _, r := range rigs {
		out.S(" ").S(r.Name).S("=")
		for i, c := range r.Chests {
			if i > 0 {
				out.S(",")
			}
			out.I(harness.ChestCount(s, "steel-chest", c.X, c.Y))
		}
	}
	out.End()
}

var schedule = []harness.Step{
	// The opening statement of phase two, taken as early as a script can: one tick
	// of belt movement after a load, and the conversion (if it happened) happened
	// before the first tick, at on_init or at on_configuration_changed.
	{Tick: 1, Do: func() {
		census("t1")
		reportCounts("t1")
		reportItem("t1")
		reportTech("t1")
		reportFidelity("t1")
		reportSurfaces("t1")
		reportForceParts("t1")
	}},
	{Tick: 60, Do: func() {
		auditNow()
		census("post-audit")
		reportCounts("post-audit")
	}},
	{Tick: 1800, Do: func() { report(1800) }},
	{Tick: 3540, Do: func() { report(3540) }},
	{Tick: 3560, Do: func() {
		auditNow()
		census("final")
		reportCounts("final")
		reportTech("final")
		reportFidelity("final")
		reportSurfaces("final")
		reportForceParts("final")
	}},
	// After everything else, so nothing above can see it. The swap is deferred by
	// one tick like every other build, hence the gap before the probe.
	{Tick: 3570, Do: lateBuild},
	{Tick: 3580, Do: lateProbe},
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	east = fkapi.DefinesDirectionEast()
}

//go:wasmexport fk_on_init
func onInit() {
	base := 0
	bases := make([]int, len(rigCfgs))
	for i := range rigCfgs {
		bases[i] = base
		base += pitch
	}
	witBase := base
	fidBase := witBase + pitch
	frcBase := fidBase + pitch
	rows := frcBase + pitch
	// ceil(rows/32) + 3; rows is 96, so ceil is 3.
	makeSurface(surfA, rows, 6)

	// BEFORE ANY PART OF ITS OWN IS PLACED, so `game.forces[forceB]` is there for
	// the whole life of the save and every per-force report can name it without a
	// guard.
	harness.CreateForce(forceB)

	rigs = rigs[:0]
	for i, cfg := range rigCfgs {
		if cfg.SingleEdge {
			rigs = append(rigs, buildSingle(cfg, bases[i]))
		} else {
			rigs = append(rigs, buildRig(cfg, bases[i]))
		}
	}
	buildWitness(witBase)
	buildFidelity(fidBase)
	buildForces(frcBase)
	// LAST, so the surface indices are fixed and deterministic: nauvis 1,
	// bbb-mig-a 2, bbb-mig-b 3, and the hidden surface -- when it exists at all --
	// after them.
	buildSurfaceB()

	// The other half of what a removed mod takes with it.
	c := harness.Place(surface(surfA), harness.Piece{
		Name: "steel-chest", X: 10, Y: witBase + pitch,
	})
	harness.InsertInto(c, legacyPart, 50)
	itemChest = harness.XY{X: 10, Y: witBase + pitch}

	census("create")
	reportCounts("create")
	reportItem("create")
	reportTech("create")
	reportFidelity("create")
	reportSurfaces("create")
	reportForceParts("create")
	auditNow()
	out.Open("init complete: ").U(uint64(len(rigCfgs))).
		S(" rigs plus the witness, the fidelity pair, the force column and a second surface").End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	harness.Run(schedule, fkapi.ReadOnTick(ptr).Tick)
}

func main() {}
