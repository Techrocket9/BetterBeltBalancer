// Command bbb-plat-test is the SPACE AGE suite. Two legs, and they are here
// together because Space Age is what they have in common and a second DLC-only
// suite would cost a second Factorio run for one rig.
//
// A COMPILED GO OBSERVER, not a Lua test mod: the same program
// `test/mods/bbb-plat-test/control.lua` was, band for band and log line for log
// line, with its data stage compiled too (obs/platdata). See
// agents/estate-port.md.
//
//  1. a balancer whose parts are on a SPACE PLATFORM surface while its network is
//     on the one global hidden surface, so its linked belts cross from a moving
//     platform to a surface that is not going anywhere;
//  2. BELT STACKING, which is a Space Age feature at the prototype level: a
//     loader's `max_belt_stack_size` is refused at load without the
//     `space_travel` feature flag, so no base-only suite can build a stacked belt
//     at all. This leg is what says a recompile hands a stacked network back
//     STACKED rather than merely conserved.
//
// The stacking leg builds on its own flat scratch surface, on its own FORCE
// (`bbb-stack`, `belt_stack_size_bonus = 3`), which is the other half of what
// stacking needs and is also the guest's own gate: the platform leg's rigs are
// on `player`, whose bonus stays 0, so one save exercises both arms of it.
//
// THE `smix` BAND IS STACKED SUSHI, AND IT IS THE ONLY RIG IN THIS REPO THAT
// REACHES `kindAt`'s MULTI-CANDIDATE BRANCH. `detailedTally` reads a stacked
// line position by position and `kindAt` decides which (name, quality) total a
// position belongs to; its `len(totals) == 1` branch is the only one any suite
// had ever run, because every stacked rig above is single-kind iron plate and
// every MULTI-kind rig lives in the base-only `mix` suite, where the stacking
// gate is shut and `detailedTally` is never called at all. Multi-kind AND
// stacked is Space Age, so it is here. See CLAUDE.md, "Stacked belts come back
// stacked".
//
// EVERY RIG HERE IS BUILT TO FACTORIO 2.1'S RULE: ONE BELT PER BALANCER PART.
// Every column of parts is TWO columns -- a west part carrying the row's input
// and an east part carrying its output -- because an edge is an interface linked
// belt standing on the cluster's own tile and 2.1's collision validator forbids
// two belt-connectables on one tile. What that costs a band is GEOMETRY AND
// NOTHING ELSE: N, M, the stack sizes and the item kinds in flight are
// properties of the BELTS, and the belts did not move.
//
// Each band also carries ONE EXTRA EDGELESS PART below its west column, and it
// is not decoration. Every recompile here is forced by laying a belt against the
// cluster, and under this rule a working balancer HAS NO FREE FACE -- so the
// belt each `recompile` used to put on the block's north face would now be
// REFUSED and the band would measure a refusal instead of a teardown. The
// edgeless part is the one tile a belt can still reach; `m2`'s conservation rig
// and the interactive checklist's band B reached the same conclusion
// independently. See agents/single-edge.md.
//
// It ASSERTS NOTHING. test/assert-plat.py decides.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/protos"
)

const (
	part   = "bbb-balancer-part"
	belt   = "express-transport-belt"
	hidden = "bbb-hidden"

	stkSurf  = "bbb-stk"   // the scratch surface
	stkForce = "bbb-stack" // the stacking force
	pitch    = 12

	platSurfaceTile = "space-platform-foundation"
)

var out = harness.Line{Tag: "[BBB-STK] "}
var plat = harness.Line{Tag: "[BBB-PLAT] "}

var east uint32

// The five bands. `full`, `plain` and `smix` are DEAD-ENDED on purpose: a
// network with nowhere to drain fills to capacity and then stops moving, which
// is what makes a before/after sample taken inside one tick a measurement of the
// teardown and of nothing else.
const (
	bandCtrl  = 0
	bandFull  = pitch
	bandFlow  = 2 * pitch
	bandPlain = 3 * pitch
	bandSmix  = 4 * pitch
)

// rowsOf is how many ROWS each band's part block is, which under the one-belt
// rule is half its part count -- and, with the edgeless part on the end, exactly
// where `recompile` has to put its belt: (-1, base + rows).
func bandBase(name string) int {
	switch name {
	case "full":
		return bandFull
	case "flow":
		return bandFlow
	case "plain":
		return bandPlain
	case "smix":
		return bandSmix
	}
	harness.Fatal("no band", name)
	return 0
}

func bandRows(name string) int {
	switch name {
	case "full", "flow":
		return 4
	case "plain", "smix":
		return 2
	}
	harness.Fatal("no band", name)
	return 0
}

// ---------------------------------------------------------------------------
// the stacked-sushi band
//
// Two sushi sources over SIX distinct names, none of which is `iron-plate` --
// and that disjointness is load-bearing rather than tidy. Every other band in
// this suite runs iron plate, so a name is enough to say which network an item
// came out of, and the four bands above keep the numbers they were recorded with
// because every count and every profile in this file filters by it. One save,
// two independent measurements, no second Factorio run.
//
// SEVEN NAMES AND NOT FORTY-EIGHT. The carry pool's bound is 32 (name, quality,
// STACK SIZE) groups, and belt stacking multiplies a name by every stack size
// standing on the lines at once -- so seven names is up to 28 groups, inside the
// bound with room. Overflowing it is the `mix` suite's job ("More than
// thirty-two kinds"); overflowing it HERE would spill, and a spill is what this
// band asserts does not happen.
// ---------------------------------------------------------------------------

// smixItem is one entry of a rotation: a name, and optionally a quality.
type smixItem struct {
	Name    string
	Quality string
}

// ...AND ONE NAME AT THREE QUALITIES, which is the other branch. `kindAt`
// settles a position by `name_is` and reaches for `LuaItemStack.quality` only
// when the line's totals carry the SAME NAME twice -- the one thing a name
// cannot decide. `plastic-bar` is on source 1's list three times over, at
// normal, uncommon and rare, and the three are CONSECUTIVE bands so two of them
// land on one hidden line. `quality` is an optional key of
// `InfinityInventoryFilter` and the `quality` mod is already loaded for this
// suite, so no new dependency is taken.
var smixItems = [][]smixItem{
	{
		{Name: "copper-plate"}, {Name: "steel-plate"}, {Name: "iron-gear-wheel"},
		{Name: "plastic-bar"},
		{Name: "plastic-bar", Quality: "uncommon"},
		{Name: "plastic-bar", Quality: "rare"},
	},
	{
		{Name: "copper-cable"}, {Name: "electronic-circuit"}, {Name: "stone-brick"},
	},
}

// smixNames is every DISTINCT name on those lists, in first-seen order.
var smixNames []string

func initSmixNames() {
	smixNames = smixNames[:0]
	for _, band := range smixItems {
		for _, it := range band {
			known := false
			for _, n := range smixNames {
				if n == it.Name {
					known = true
					break
				}
			}
			if !known {
				smixNames = append(smixNames, it.Name)
			}
		}
	}
}

// mine splits every count in this file into the two independent measurements the
// one save carries: false is the four iron-plate bands, true is `smix`.
func mine(name string, smix bool) bool {
	for _, n := range smixNames {
		if n == name {
			return smix
		}
	}
	return !smix
}

// srotate is the rotation period, and it is a MEASURED constant rather than a
// preference. What `kindAt` needs is a hidden transport LINE carrying two names
// at once, and a hidden belt tile holds four item positions per lane: a band
// longer than that gives single-kind lines and the multi-candidate branch is
// never entered. A stacking loader at express rate emits ~3 items/tick over two
// lanes, so four ticks is ~1.5 stacked positions per lane -- comfortably shorter
// than a line. The `smixlines` sample is what proves it landed, and
// assert-plat.py fails the run if it did not.
const srotate = 4

// sushi is one rotating source, held as a TILE for the reason every registry in
// the estate holds one: `fk_on_init` runs during `--create` and the rotation
// runs during `--benchmark`, so anything held here crosses a save.
type sushi struct {
	X, Y   int
	Items  []smixItem
	Offset int
}

var sushis []sushi

// ---------------------------------------------------------------------------
// pieces
// ---------------------------------------------------------------------------

func stk() fkapi.LuaSurface { return harness.Surface(stkSurf) }

// sput is a placement on the STACKING FORCE. Everything on the scratch surface
// belongs to it; the platform leg's own pieces are on `player`, which is what
// makes one save exercise both arms of the stacking gate.
func sput(s fkapi.LuaSurface, name string, x, y int, dir *uint32, typ string) fkapi.Object {
	return harness.Place(s, harness.Piece{
		Name: name, X: x, Y: y, Dir: dir, Type: typ, Force: stkForce, Raise: true,
	})
}

func setFilter(c fkapi.Object, it smixItem) {
	q := it.Quality
	if q == "" {
		q = "normal"
	}
	harness.InfinityFilter(c, it.Name, q, 2000)
}

// source is a stacking source: an infinity chest behind a loader whose prototype
// allows a stack of 4. The force's bonus is the other half; without it this
// delivers singles, which is the `plain` band's whole point.
func source(s fkapi.LuaSurface, x, y int, stacking bool) {
	c := harness.Place(s, harness.Piece{
		Name: "infinity-chest", X: x, Y: y, Force: stkForce,
	})
	harness.InfinityFilter(c, "iron-plate", "", 2000)
	name := protos.PlatLoader
	if stacking {
		name = protos.PlatStackLoader
	}
	sput(s, name, x+1, y, &east, "output")
}

// sushiSource is the same stacking loader, but its chest holds ONE filter at a
// time and rewrites it every srotate ticks, with `remove_unfiltered_items` on so
// the previous kind is voided rather than left for the loader to prefer.
//
// One filter and not six. The `mix` suite measured the naive rig -- an infinity
// chest carrying every filter at once -- and it delivers ONE kind: a loader
// draws from the first stack it finds and the chest tops that same stack
// straight back up (2,292 items in 1 of 6 kinds). Rotating a single filter gives
// a BANDED belt, which is what a real sushi bus looks like anyway, and it is
// deterministic: the band boundaries are a function of `game.tick` and nothing
// else.
func sushiSource(s fkapi.LuaSurface, x, y int, items []smixItem, offset int) {
	c := harness.Place(s, harness.Piece{
		Name: "infinity-chest", X: x, Y: y, Force: stkForce,
	})
	harness.RemoveUnfiltered(c, true)
	setFilter(c, items[0])
	sput(s, protos.PlatStackLoader, x+1, y, &east, "output")
	sushis = append(sushis, sushi{X: x, Y: y, Items: items, Offset: offset})
}

func rotateSushi(tick uint64) {
	if tick%srotate != 0 {
		return
	}
	step := int(tick / srotate)
	s := stk()
	for _, sr := range sushis {
		c, ok := harness.FindAt(s, sr.X, sr.Y, "infinity-chest", "")
		if !ok {
			continue
		}
		setFilter(c, sr.Items[(step+sr.Offset)%len(sr.Items)])
	}
}

func sink(s fkapi.LuaSurface, x, y int) harness.XY {
	sput(s, protos.PlatLoader, x, y, &east, "input")
	harness.Place(s, harness.Piece{
		Name: "steel-chest", X: x + 1, Y: y, Force: stkForce,
	})
	return harness.XY{X: x + 1, Y: y}
}

func stkAudit() { harness.Audit(stk(), -30, 0) }

// ---------------------------------------------------------------------------
// counting
// ---------------------------------------------------------------------------

// conserved says whether an entity's contents are a conserved quantity. The one
// thing an infinity chest is not: it mints and voids items every tick by design,
// and `smix`'s sources void a whole band's worth on every rotation. Counting one
// would make every total a measurement of the source rather than of the
// balancer.
func conserved(o fkapi.Object) bool {
	return harness.EntityType(o) != "infinity-container"
}

// kindKey is a (NAME, QUALITY) pair, which is exactly the key `get_contents`
// returns and exactly what `kindAt` has to tell apart. Counting by name alone
// would let a teardown that handed back three normal plastic bars for three rare
// ones pass every check in this file.
func kindKey(name, quality string) string {
	if quality == "" {
		quality = "normal"
	}
	return name + "/" + quality
}

func visArea() fkapi.BoundingBox { return harness.Box(-12, -8, 12, 5*pitch+8) }

// hiddenArea is where the network lives, and it is the only place a teardown can
// take items from. Its slots are laid out 32x72 from the origin.
func hiddenArea() fkapi.BoundingBox { return harness.Box(-16, -16, 2200, 400) }

func hiddenSurface() (fkapi.LuaSurface, bool) {
	h, err := fkapi.Game.GetSurface(fkapi.OfString(hidden))
	if err != nil || h == nil {
		return fkapi.LuaSurface{}, false
	}
	return fkapi.LuaSurface{Object: *h}, true
}

// countAll totals everything countable: items on the ground, items on every
// transport line, and items in every chest. Conservation is only a claim if the
// sinks are counted too -- `flow` has delivered thousands by the time it is
// recompiled. `smix` selects which of the two independent measurements this save
// carries is being counted, and `kinds`, when given, collects the per-KIND
// breakdown -- which is what the stacked-sushi band asserts on.
func countAll(s fkapi.LuaSurface, area fkapi.BoundingBox, smix bool, kinds *harness.Tally) int64 {
	var n int64
	take := func(name, quality string, c int64) {
		if !mine(name, smix) {
			return
		}
		n += c
		if kinds != nil {
			kinds.Add(kindKey(name, quality), c)
		}
	}
	itemEntity := fkapi.OfString("item-entity")
	found, err := s.FindEntitiesFiltered(fkapi.EntitySearchFilters{
		Area: &area, Type: &itemEntity,
	})
	if err == nil {
		for _, e := range found {
			if name, c, ok := harness.GroundStack(e); ok {
				// A ground stack carries a quality too, and the Lua read it through
				// the same kindkey: `e.stack` is a LuaItemStack, whose quality is
				// userdata rather than the plain string a contents row carries.
				take(name, groundQuality(e), c)
			}
		}
	}
	for _, e := range harness.EntitiesIn(s, area, "") {
		if !conserved(e) {
			continue
		}
		harness.EachLine(e, func(l fkapi.LuaTransportLine) {
			c, err := l.GetContents()
			if err != nil {
				return
			}
			for _, item := range c {
				take(item.Name, item.Quality, int64(item.Count))
			}
		})
		for _, item := range harness.ChestContents(e) {
			take(item.Name, item.Quality, int64(item.Count))
		}
	}
	return n
}

func groundQuality(o fkapi.Object) string {
	st, err := (fkapi.LuaEntity{Object: o}).Stack()
	if err != nil {
		return ""
	}
	return stackQuality(fkapi.LuaItemStack{Object: st})
}

// stackQuality is the `q.name` half of the Lua's kindkey: `quality` is a plain
// string on the (name, quality, count) rows `get_contents` returns and a
// LuaQualityPrototype -- userdata -- on a LuaItemStack. Both readings reach the
// counter, so both are named.
func stackQuality(s fkapi.LuaItemStack) string {
	q, err := s.Quality()
	if err != nil {
		return ""
	}
	n, err := (fkapi.LuaQualityPrototype{Object: q}).Name()
	if err != nil {
		return ""
	}
	return n
}

// hist is a stack-size histogram over belt POSITIONS, kept in ascending key
// order so the rendered string is a property of the program rather than of a
// hash.
type hist struct {
	sizes  []uint32
	counts []int64
}

func (h *hist) add(size uint32) {
	for i, s := range h.sizes {
		if s == size {
			h.counts[i]++
			return
		}
	}
	j := len(h.sizes)
	h.sizes = append(h.sizes, size)
	h.counts = append(h.counts, 1)
	for j > 0 && h.sizes[j-1] > h.sizes[j] {
		h.sizes[j-1], h.sizes[j] = h.sizes[j], h.sizes[j-1]
		h.counts[j-1], h.counts[j] = h.counts[j], h.counts[j-1]
		j--
	}
}

func (h *hist) write(l *harness.Line) {
	for i, s := range h.sizes {
		if i > 0 {
			l.S(",")
		}
		l.U(uint64(s)).S(":").I(h.counts[i])
	}
}

// profile is the stack PROFILE of a set of transport lines: how many items, how
// many belt POSITIONS they occupy, and the histogram of stack sizes over those
// positions.
//
// THIS IS THE MEASUREMENT THE LEG EXISTS FOR. Conservation compares item counts
// and was already exact before stacking was recovered; what was wrong is that 72
// items came back as 72 positions of 1 instead of 18 positions of 4.
func profile(s fkapi.LuaSurface, area fkapi.BoundingBox, smix bool) (items, positions int64, h hist) {
	for _, e := range harness.EntitiesIn(s, area, "") {
		if !conserved(e) {
			continue
		}
		harness.EachLine(e, func(l fkapi.LuaTransportLine) {
			d, err := l.GetDetailedContents()
			if err != nil {
				return
			}
			for _, item := range d {
				st := fkapi.LuaItemStack{Object: item.Stack}
				name, err := st.Name()
				if err != nil || !mine(name, smix) {
					continue
				}
				c, err := st.Count()
				if err != nil {
					continue
				}
				items += int64(c)
				positions++
				h.add(c)
			}
		})
	}
	return items, positions, h
}

// lineMix is THE ANTI-VACUITY SAMPLE, and without it the band proves nothing.
//
// `kindAt`'s cheap branch is `len(totals) == 1` -- one (name, quality) on the
// line, no host call at all -- and it is the branch every stacked rig this repo
// has ever built takes. What the multi-candidate branch needs is a single
// transport LINE carrying two names at once, and it needs at least one of those
// positions to be a STACK, because an unstacked line never reaches
// detailedTally in the first place. Neither is something a rotation period can
// be assumed into producing: if the sushi bands are longer than a line, every
// line is single-kind and the whole leg passes while exercising nothing.
//
// So this walks the hidden surface and reports, over the lines that carry
// anything of `smix`'s at all: how many carry two or more distinct NAMES, how
// many of those also carry a stacked position, and the richest line seen.
// assert-plat.py requires both counts to be non-zero.
type mixReport struct {
	Lines, Multi, MultiStacked, QTie, QTieStacked int64
	MaxNames, MaxKinds, MaxStack                  int64
}

func lineMix(s fkapi.LuaSurface, area fkapi.BoundingBox) mixReport {
	var r mixReport
	for _, e := range harness.EntitiesIn(s, area, "") {
		if !conserved(e) {
			continue
		}
		harness.EachLine(e, func(l fkapi.LuaTransportLine) {
			d, err := l.GetDetailedContents()
			if err != nil {
				return
			}
			// names counts DISTINCT KINDS PER NAME, which is the quality tiebreak's
			// own precondition: one name on this line carrying more than one
			// quality is the thing a `name_is` cannot settle.
			var names []string
			var perName []int64
			var kinds []string
			var stacked, big int64
			for _, item := range d {
				st := fkapi.LuaItemStack{Object: item.Stack}
				name, err := st.Name()
				if err != nil || !mine(name, true) {
					continue
				}
				ni := -1
				for i, n := range names {
					if n == name {
						ni = i
						break
					}
				}
				if ni < 0 {
					names = append(names, name)
					perName = append(perName, 0)
					ni = len(names) - 1
				}
				k := kindKey(name, stackQuality(st))
				known := false
				for _, kk := range kinds {
					if kk == k {
						known = true
						break
					}
				}
				if !known {
					kinds = append(kinds, k)
					perName[ni]++
				}
				c, err := st.Count()
				if err != nil {
					continue
				}
				if c > 1 {
					stacked++
				}
				if int64(c) > big {
					big = int64(c)
				}
			}
			if len(kinds) == 0 {
				return
			}
			r.Lines++
			if int64(len(names)) > r.MaxNames {
				r.MaxNames = int64(len(names))
			}
			if int64(len(kinds)) > r.MaxKinds {
				r.MaxKinds = int64(len(kinds))
			}
			if big > r.MaxStack {
				r.MaxStack = big
			}
			if len(names) >= 2 {
				r.Multi++
				if stacked >= 1 {
					r.MultiStacked++
				}
			}
			for _, q := range perName {
				if q >= 2 {
					r.QTie++
					if stacked >= 1 {
						r.QTieStacked++
					}
					break
				}
			}
		})
	}
	return r
}

// ---------------------------------------------------------------------------
// samples
// ---------------------------------------------------------------------------

func sample(tag string) {
	s := stk()
	vis := countAll(s, visArea(), false, nil)
	var hn, hi, hp int64
	var h hist
	if hs, ok := hiddenSurface(); ok {
		hn = countAll(hs, hiddenArea(), false, nil)
		hi, hp, h = profile(hs, hiddenArea(), false)
	}
	out.Open("").S(tag).S(" total=").I(vis + hn).S(" visible=").I(vis).
		S(" hidden=").I(hn).S(" hitems=").I(hi).S(" hpos=").I(hp).S(" hist=")
	h.write(&out)
	out.End()
}

// smixSample is the same sample for the stacked-sushi band, PER KIND, plus the
// profile and the mixed-line evidence. Sorted by kind, so two samples of the
// same world produce the same lines in the same order on every machine.
func smixSample(tag string) {
	s := stk()
	var kinds harness.Tally
	vis := countAll(s, visArea(), true, &kinds)
	var hn, hi, hp int64
	var h hist
	var r mixReport
	if hs, ok := hiddenSurface(); ok {
		hn = countAll(hs, hiddenArea(), true, &kinds)
		hi, hp, h = profile(hs, hiddenArea(), true)
		r = lineMix(hs, hiddenArea())
	}
	names := kinds.Names()
	out.Open("smix tag=").S(tag).S(" total=").I(vis + hn).S(" visible=").I(vis).
		S(" hidden=").I(hn).S(" kinds=").U(uint64(len(names))).
		S(" hitems=").I(hi).S(" hpos=").I(hp).S(" hist=")
	h.write(&out)
	out.End()
	out.Open("smixlines tag=").S(tag).S(" lines=").I(r.Lines).
		S(" multi=").I(r.Multi).S(" multistacked=").I(r.MultiStacked).
		S(" qtie=").I(r.QTie).S(" qtiestacked=").I(r.QTieStacked).
		S(" maxnames=").I(r.MaxNames).S(" maxkinds=").I(r.MaxKinds).
		S(" maxstack=").I(r.MaxStack).End()
	for _, name := range names {
		out.Open("smixkind tag=").S(tag).S(" name=").S(name).
			S(" count=").I(kinds.Get(name)).End()
	}
}

// ---------------------------------------------------------------------------
// the recompiles
// ---------------------------------------------------------------------------

// recompile lays a belt against the band's EDGELESS part, which is a genuine new
// edge -- a new INPUT, so the port count goes UP -- and the fingerprint moves,
// so the network comes down and goes back up. It goes there rather than on the
// block's north face because under the one-belt rule every part of a working
// balancer already carries its belt and a belt on any of them would be REFUSED.
// The audit marker in the same tick is what makes the rebuild happen inside this
// dispatch rather than on the next tick, so "before" and "after" are one atomic
// sample apart.
//
// THE PROFILER IS AROUND THE AUDIT, not around a tick pair, because the sample
// either side of it has to be atomic. That means it carries a whole-save
// re-classification as well as the recompile -- the same trade `assert-m2.py`
// documents -- so the number is only ever compared with the `audit only, nothing
// pending` line below it and with the same measurement from another build. It is
// NOT comparable with M2's tick-pair recompile timings.
func recompile(band string) {
	sample(band + " before")
	sput(stk(), belt, -1, bandBase(band)+bandRows(band), &east, "")
	p := harness.StartProfiler()
	stkAudit()
	p.Stop()
	p.Log("[BBB-STK] timing " + band + " recompile (audit-forced) ")
	sample(band + " after")
}

// smixRecompile is the same gesture for `smix`, sampled per KIND. A belt
// arriving against the edgeless part is a new INPUT edge, so the port count goes
// UP -- P = next_pow2(max(N,M)) takes a 2->2 from 2 to 4 -- and the network the
// rebuild produces is strictly bigger than the one it replaced. That matters: a
// SHRINK would legitimately spill whatever no longer fits (carry.go, decision
// 4), and this band is about what the drain did with the kinds, not about
// capacity.
func smixRecompile() {
	smixSample("before")
	sput(stk(), belt, -1, bandSmix+bandRows("smix"), &east, "")
	p := harness.StartProfiler()
	stkAudit()
	p.Stop()
	p.Log("[BBB-STK] timing smix recompile (audit-forced) ")
	smixSample("after")
}

// ---------------------------------------------------------------------------
// anti-vacuity
// ---------------------------------------------------------------------------

// checkItems is FATAL on a missing prototype and names it, in the create log,
// rather than leaving a band that quietly carries fewer kinds than it claims --
// which here would be a multi-kind rig that is single-kind and passes every
// conservation check while proving nothing.
func checkItems() {
	var missing []string
	kinds := 0
	for _, name := range smixNames {
		if !harness.ItemProtoExists(name) {
			missing = append(missing, "item "+name)
		}
	}
	for _, band := range smixItems {
		for _, it := range band {
			kinds++
			if it.Quality != "" && !harness.QualityProtoExists(it.Quality) {
				missing = append(missing, "quality "+it.Quality)
			}
		}
	}
	harness.SortStrings(missing)
	if len(missing) > 0 {
		l := harness.Line{Tag: "[BBB-OBS] "}
		l.Open("error: no such prototype: ")
		for i, m := range missing {
			if i > 0 {
				l.S(", ")
			}
			l.S(m)
		}
		l.End()
		return
	}
	out.Open("smix item list ok: names=").U(uint64(len(smixNames))).
		S(" kinds=").U(uint64(kinds)).S(" rotate=").U(srotate).End()
}

// ---------------------------------------------------------------------------
// the stacking leg
// ---------------------------------------------------------------------------

// ctrlChest and flowChests are the stacking leg's sinks, held as tiles.
var ctrlChest harness.XY
var flowChests []harness.XY

func buildStacking() {
	rows := 5 * pitch
	s := harness.Flat{
		Name:        stkSurf,
		MapWidth:    512,
		MapHeight:   512,
		ChunkCenter: fkapi.MapPosition{X: 0, Y: float64(rows) / 2},
		// ceil(rows/32) + 3; rows is 60, so ceil is 2.
		ChunkRadius: 5,
		X0:          -12,
		Y0:          -8,
		X1:          12,
		Y1:          rows + 8,
		Tile:        "grass-1",
	}.Make()

	f := harness.CreateForce(stkForce)
	// The other half of belt stacking. It is a research result in a real game and
	// it only ever goes up, which is why the guest may cache it for a dispatch.
	if err := f.SetBeltStackSizeBonus(3); err != nil {
		harness.Fatal("setting belt_stack_size_bonus", stkForce)
	}
	bonus, _ := f.BeltStackSizeBonus()
	out.Open("force ").S(stkForce).S(" belt_stack_size_bonus=").U(uint64(bonus)).End()

	// ctrl: one uninterrupted stacked belt. The yardstick, and the thing that
	// fails loudly if belt stacking silently did not happen at all.
	source(s, -5, bandCtrl, true)
	for x := -3; x <= 3; x++ {
		sput(s, belt, x, bandCtrl, &east, "")
	}
	ctrlChest = sink(s, 4, bandCtrl)

	// Every band below is TWO COLUMNS OF PARTS plus one EDGELESS part on the end;
	// partsFor lays them, so no band can forget either half of the rule.
	//
	//	x=-5 chest  -4 loader  -3..-1 belts  0 WEST PART  1 EAST PART
	//	x=2..4 belts   5 sink loader   6 chest    (dead-ended bands stop at 4)
	partsFor := func(band string) {
		base, rows := bandBase(band), bandRows(band)
		for i := 0; i < rows; i++ {
			sput(s, part, 0, base+i, nil, "")
			sput(s, part, 1, base+i, nil, "")
		}
		sput(s, part, 0, base+rows, nil, "") // the edgeless one; see recompile
	}

	// full: 4 in, dead-ended out. Fills the hidden network with stacks and stops.
	partsFor("full")
	for i := 0; i <= 3; i++ {
		source(s, -5, bandFull+i, true)
		for x := -3; x <= -1; x++ {
			sput(s, belt, x, bandFull+i, &east, "")
		}
		for x := 2; x <= 4; x++ {
			sput(s, belt, x, bandFull+i, &east, "")
		}
	}

	// flow: 4 in, 4 out, running. Recompiled while stacked items are moving
	// through it, and then measured for another 700 ticks.
	flowChests = flowChests[:0]
	partsFor("flow")
	for i := 0; i <= 3; i++ {
		source(s, -5, bandFlow+i, true)
		for x := -3; x <= -1; x++ {
			sput(s, belt, x, bandFlow+i, &east, "")
		}
		for x := 2; x <= 4; x++ {
			sput(s, belt, x, bandFlow+i, &east, "")
		}
		flowChests = append(flowChests, sink(s, 5, bandFlow+i))
	}

	// plain: the same force, an ordinary loader, so the lines are UNSTACKED while
	// the gate is open. This is the branch that costs one host call per non-empty
	// line and then hands the flat totals back, and it has to conserve exactly.
	partsFor("plain")
	for i := 0; i <= 1; i++ {
		source(s, -5, bandPlain+i, false)
		for x := -3; x <= -1; x++ {
			sput(s, belt, x, bandPlain+i, &east, "")
		}
		for x := 2; x <= 4; x++ {
			sput(s, belt, x, bandPlain+i, &east, "")
		}
	}

	// smix: 2 in, 2 out, dead-ended, fed by STACKED SUSHI. The only rig in this
	// repo whose hidden transport lines carry more than one item name AND more
	// than one item per position at the same time, which is the pair of conditions
	// `kindAt`'s multi-candidate branch needs to be reached at all.
	partsFor("smix")
	for i := 0; i <= 1; i++ {
		// The offset staggers the two sources so they do not switch to their first
		// item on the same tick, which would band the whole rig in lockstep and
		// leave every line single-kind after all.
		sushiSource(s, -5, bandSmix+i, smixItems[i], i*2)
		for x := -3; x <= -1; x++ {
			sput(s, belt, x, bandSmix+i, &east, "")
		}
		for x := 2; x <= 4; x++ {
			sput(s, belt, x, bandSmix+i, &east, "")
		}
	}

	stkAudit()
	out.Open("init complete").End()
}

// ---------------------------------------------------------------------------
// the platform leg
// ---------------------------------------------------------------------------

// platSurface is the platform's own surface NAME, learned at init and held for
// the reporting ticks. Factorio names it (`platform-1`), so it cannot be
// written down.
var platSurface string

// platCtrl and platOut are the platform leg's sinks, as tiles on that surface.
var platCtrl harness.XY
var platOut []harness.XY

func buildPlatform() {
	force, ok := harness.ForceByName(harness.PlayerForce)
	if !ok {
		harness.Fatal("resolving the player force", harness.PlayerForce)
		return
	}
	f := fkapi.LuaForce{Object: force}
	if err := f.UnlockSpacePlatforms(); err != nil {
		plat.Open("could not unlock space platforms").End()
		return
	}
	planet, ok := harness.SpaceLocationProto("nauvis")
	if !ok {
		plat.Open("could not create a space platform: no such planet nauvis").End()
		return
	}
	name := "bbb-plat"
	p, err := f.CreateSpacePlatform(fkapi.LuaForceCreateSpacePlatformArgs{
		Name:        &name,
		Planet:      planet,
		StarterPack: fkapi.OfString("space-platform-starter-pack"),
	})
	if err != nil || p == nil {
		plat.Open("could not create a space platform: nil").End()
		return
	}
	sp := fkapi.LuaSpacePlatform{Object: *p}
	if err := applyStarterPack(sp); err != nil {
		plat.Open("could not apply the starter pack").End()
		return
	}
	so, err := sp.Surface()
	if err != nil {
		plat.Open("platform state=? surface=NIL").End()
		return
	}
	s := fkapi.LuaSurface{Object: so}
	state, _ := sp.State()
	sname, err := s.Name()
	if err != nil {
		sname = "NIL"
	}
	platSurface = sname
	plat.Open("platform state=").U(uint64(state)).S(" surface=").S(sname).End()

	harness.PaveBox(s, -14, -6, 14, 6, platSurfaceTile)
	for _, e := range harness.EntitiesIn(s, harness.Box(-14, -6, 15, 7), "") {
		if harness.EntityType(e) == "character" {
			continue
		}
		harness.Destroy(e, false)
	}

	pput := func(name string, x, y int, dir *uint32, typ string) bool {
		_, ok := harness.PlaceSoft(s, harness.Piece{
			Name: name, X: x, Y: y, Dir: dir, Type: typ, Raise: true,
		})
		return ok
	}

	// FOUR parts, not two: a west part carrying each row's input and an east part
	// carrying its output, because one tile may carry one belt. The machine is the
	// same 2->2 it always was -- N, M and P are properties of the belts.
	platOut = platOut[:0]
	for i := 0; i <= 1; i++ {
		for x := 0; x <= 1; x++ {
			if !pput(part, x, i, nil, "") {
				plat.Open("could not place a part").End()
				return
			}
		}
	}
	for i := 0; i <= 1; i++ {
		c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: -6, Y: i})
		harness.InfinityFilter(c, "iron-plate", "", 1000)
		pput(protos.PlatLoader, -5, i, &east, "output")
		for x := -4; x <= -1; x++ {
			pput(belt, x, i, &east, "")
		}
		for x := 2; x <= 3; x++ {
			pput(belt, x, i, &east, "")
		}
		pput(protos.PlatLoader, 4, i, &east, "input")
		harness.Place(s, harness.Piece{Name: "steel-chest", X: 5, Y: i})
		platOut = append(platOut, harness.XY{X: 5, Y: i})
	}
	// The yardstick: one uninterrupted belt on the same platform.
	c := harness.Place(s, harness.Piece{Name: "infinity-chest", X: -6, Y: 4})
	harness.InfinityFilter(c, "iron-plate", "", 1000)
	pput(protos.PlatLoader, -5, 4, &east, "output")
	for x := -4; x <= 3; x++ {
		pput(belt, x, 4, &east, "")
	}
	pput(protos.PlatLoader, 4, 4, &east, "input")
	harness.Place(s, harness.Piece{Name: "steel-chest", X: 5, Y: 4})
	platCtrl = harness.XY{X: 5, Y: 4}

	// The guest defers every recompile to the next tick (`fk.Defer`), and
	// `--create` never reaches one, so the network would otherwise be compiled on
	// the first tick of the benchmark instead of into the save. `bbb-audit` is the
	// marker that drains the queue synchronously.
	harness.Audit(s, 8, 0)
	plat.Open("init complete").End()
}

// ---------------------------------------------------------------------------
// reporting
// ---------------------------------------------------------------------------

func stkReport(tick uint64) {
	s := stk()
	out.Open("t=").U(tick).S(" ctrl=").I(harness.ChestCount(s, "steel-chest", ctrlChest.X, ctrlChest.Y)).
		S(" flow=[")
	for i := 0; i < 4; i++ {
		if i > 0 {
			out.S(" ")
		}
		if i < len(flowChests) {
			out.I(harness.ChestCount(s, "steel-chest", flowChests[i].X, flowChests[i].Y))
		} else {
			out.I(-1)
		}
	}
	out.S("]").End()
}

func platReport(tick uint64) {
	get := func(c harness.XY, ok bool) int64 {
		if !ok || platSurface == "" {
			return -1
		}
		return harness.ChestCount(harness.Surface(platSurface), "steel-chest", c.X, c.Y)
	}
	plat.Open("t=").U(tick).S(" ctrl=").I(get(platCtrl, platSurface != "")).S(" out=[")
	for i := 0; i <= 1; i++ {
		if i > 0 {
			plat.S(" ")
		}
		if i < len(platOut) {
			plat.I(get(platOut[i], true))
		} else {
			plat.I(-1)
		}
	}
	plat.S("]").End()
}

// ---------------------------------------------------------------------------
// the schedule
//
// 400 ticks is long enough for a dead-ended 4x4 to fill and stop; the two
// recompiles are 100 ticks apart so that the guest's own log lines cannot be
// confused between them, and `flow`'s measurement window is the 700 ticks after
// its own recompile.
// ---------------------------------------------------------------------------

var schedule = []harness.Step{
	{Tick: 400, Do: func() { sample("formed") }},
	// The control every timing below is read against: the same whole-save
	// re-classification with nothing to rebuild.
	{Tick: 500, Do: func() {
		p := harness.StartProfiler()
		stkAudit()
		p.Stop()
		p.Log("[BBB-STK] timing audit only, nothing pending ")
	}},
	{Tick: 600, Do: func() { recompile("full") }},
	{Tick: 700, Do: func() { recompile("plain") }},
	{Tick: 800, Do: func() { recompile("flow") }},
	// `smix` last of the four, and 100 ticks after `flow` for the same reason the
	// others are spaced: the guest's own log lines cannot then be confused between
	// two recompiles. By t=900 each sushi source has run its three-item list
	// seventy-five times, so the dead-ended network is frozen holding a full
	// cross-section of bands.
	{Tick: 900, Do: smixRecompile},
	// 200 ticks after its own recompile, so the window measures a network that is
	// running rather than one still refilling: a rebuild puts every drained item
	// back at the HEAD of the butterfly, so the outputs are briefly starved by
	// construction. The `edge` suite measures the same shape the same way.
	{Tick: 1000, Do: func() { stkReport(1000) }},
	{Tick: 1500, Do: func() { stkReport(1500) }},
	// After the last sample, so it cannot disturb one, and after every recompile
	// this suite makes. Nothing has touched either surface since tick 900, so the
	// registry and the world must agree exactly -- and the cluster and part counts
	// are what say the bands are the shape this suite thinks they are, which no
	// stack profile and no rate can. assert-plat.py reads the LAST audit line in
	// the run, which is this one.
	{Tick: 1520, Do: stkAudit},
}

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	east = fkapi.DefinesDirectionEast()
}

//go:wasmexport fk_on_init
func onInit() {
	initSmixNames()
	checkItems()
	sushis = sushis[:0]
	buildStacking()
	buildPlatform()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	tick := fkapi.ReadOnTick(ptr).Tick
	rotateSushi(tick)
	if tick == 600 || tick == 1500 {
		platReport(tick)
	}
	harness.Run(schedule, tick)
}

func main() {}
