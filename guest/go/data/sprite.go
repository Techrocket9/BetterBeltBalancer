package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// The I/O arrows: which sides of a balancer take items in and which give them
// out, drawn on the edge tiles when the player holds Alt.
//
// Eight `sprite` prototypes over one 256x32 strip, named
// `bbb-arrow-<in|out>-<n|e|s|w>` where the letter is the SIDE THE BELT IS ON.
// The arrow itself points the way the items go -- inwards on an input edge,
// outwards on an output -- so `in-n` is a green chevron pointing SOUTH, sitting
// on the north edge of its tile.
//
// THE SHIFT AND THE ROTATION ARE BOTH BAKED IN, and that is the whole reason
// there are eight prototypes rather than one drawn with an `orientation` and an
// offset. The guest names one of these and passes a target and nothing else: no
// orientation to compute from a `defines.direction` value whose number this mod
// deliberately never writes down, and no offset table to marshal on every one of
// the eight-or-more draws a compile makes.
//
// Rendering objects with an ENTITY target are destroyed by the engine when that
// entity is (verified in the runtime doc, `ScriptRenderTarget`), and the entity
// these are drawn on is the visible linked belt the compiler places for the
// edge. So a teardown removes them for free and the guest stores no rendering
// ids at all. See guest/go/compile.go, drawArrow.
const arrowStrip = "__better-belt-balancer__/graphics/entity/io-arrows.png"

// side is a unit direction; the DISTANCE is per family, just below.
var sides = []struct {
	name string
	x, y float64
}{
	{"n", 0, -1},
	{"e", 1, 0},
	{"s", 0, 1},
	{"w", -1, 0},
}

// THE ART IS NOT CENTRED IN ITS CELL, so the shift cannot be one number.
//
// Each chevron is drawn flush against its TAIL edge, which puts the glyph's
// centroid 0.104 tiles BEHIND its tip. An input points inwards, so that bias
// pushes it further OUT and ADDS to the shift; an output points outwards, so the
// same bias pulls it IN and SUBTRACTS. Uncompensated the two families land 0.404
// and 0.196 tiles from the tile centre: an output stops reading as an edge
// marker and sits on the machine's own hub instead. Measured, both on the
// generated placeholder and on the 2026-08-19 artist delivery -- the centroid
// offset is +-6.6 px of a 32 px cell in BOTH, so this is a property of the
// convention rather than of one sheet.
//
// Compensating per family puts both at 0.3, which is what the shift always
// meant. If the art is ever redrawn centred, set this to 0 -- do not write the
// two distances out.
//
// ---------------------------------------------------------------------------
// IT IS A `var` AND NOT A `const`, AND THAT IS THE ONE PLACE THIS PORT COULD
// HAVE CHANGED A NUMBER WITHOUT CHANGING A LINE.
//
// Go's untyped constants are ARBITRARY PRECISION: `const bias = 0.104` makes
// `0.3 + bias` compute the exact decimal sum and round ONCE, giving the double
// nearest 0.404. Lua has no such thing -- `0.3 + bias` over a local is an IEEE
// f64 add of two already-rounded doubles, and it rounds TWICE, giving the double
// BELOW that one. Two different doubles, from two languages each doing the
// arithmetic correctly, and the difference is the last ulp.
//
// Upstream met this exact expression between Go and Rust and wrote it up
// (FkLua agents/datastage.md, D5/D6); a `var` forces f64 arithmetic in Go and
// makes this a transliteration of the Lua rather than a re-derivation of it.
//
// AND IT IS NOT SELF-EVIDENT THAT THE GATE WOULD HAVE CAUGHT THE OTHER CHOICE.
// Factorio's --dump-data renders a shift with about fifty significant digits and
// the value it prints does not round-trip to the double it came from, so the
// renderer is lossy somewhere and the two candidates might well print the same.
// A gate that cannot be relied on for one field is a reason to get that field
// right by construction, not a reason to shrug at it.
// ---------------------------------------------------------------------------
var artBias = 0.104

// dist is the per-family distance: index 0 is the `in` family and 1 is `out`,
// the same two-element table the Lua kept. Computed at package scope from the
// var above, so both entries are f64 arithmetic.
var dist = [2]float64{0.3 - artBias, 0.3 + artBias}

//go:noinline
func sprites() {
	// Eight prototypes in one Extend, which is one host call rather than eight.
	// The Lua accumulated them into an array and extended once for the same
	// reason, and `data:extend` takes an array by definition.
	arrows := make([]fkdata.V, 0, len(dist)*len(sides))
	for kind, family := range [...]string{"in", "out"} {
		d := dist[kind]
		for i, s := range sides {
			arrows = append(arrows, obj(
				f("type", str("sprite")),
				f("name", str("bbb-arrow-"+family+"-"+s.name)),
				f("filename", str(arrowStrip)),
				f("priority", str("extra-high-no-scale")),
				f("width", num(32)),
				f("height", num(32)),
				// The strip is one row: the four `in` cells, then the four
				// `out` cells, in the order `sides` lists them.
				f("x", num(float64((kind*len(sides)+i)*32))),
				// 32 px at scale 0.5 is half a tile: big enough to read at
				// default zoom, small enough that two of them on one tile do not
				// collide.
				f("scale", num(0.5)),
				f("shift", list(num(s.x*d), num(s.y*d))),
				f("flags", strs("no-crop")),
			))
		}
	}
	fkdata.Extend(arrows...)
}
