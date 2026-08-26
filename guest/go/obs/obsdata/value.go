package obsdata

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// The prototype-table shorthands, which are the shipped data guest's
// (guest/go/data/value.go) under exported names.
//
// A data stage is one long table literal, and spelled out in full every field of
// it reads `fkdata.KVs("type", fkdata.Str("item"))` -- four tokens of ceremony
// around two of content. These are the same constructors under names short
// enough that a prototype in Go looks like the prototype it is:
//
//	Obj(
//	    F("type", Str("item")),
//	    F("stack_size", Num(50)),
//	    F("flags", Strs("placeable-neutral", "player-creation")),
//	)
//
// WHY THEY ARE COPIED RATHER THAN SHARED WITH THE SHIPPED GUEST. `guest/go/data`
// is `package main` -- it has to be, an `//go:wasmexport fk_data` lives in it --
// so nothing can import it. The alternative is a third package that both it and
// this one import, which would couple the mod's own data stage to the test
// estate's: a change made for an observer would relink the shipped guest, which
// is exactly what `GUEST_SRC` excluding `guest/go/obs` exists to prevent.
//
// Nothing here adds behaviour and nothing here validates. Factorio's own
// `data:extend` is the validator -- it refuses a prototype with no type or no
// name, by name, which is a better message than anything this layer could
// invent, and the real prototype check happens in C++ after every mod's stage
// has run.

// Obj is a prototype table.
//
//go:noinline
func Obj(kvs ...fkdata.KV) fkdata.V { return fkdata.Obj(kvs...) }

// F is one field of a prototype table.
func F(key string, v fkdata.V) fkdata.KV { return fkdata.KVs(key, v) }

// Str is a string value.
func Str(s string) fkdata.V { return fkdata.Str(s) }

// Num is a number value.
func Num(x float64) fkdata.V { return fkdata.Num(x) }

// List is a Lua array -- a table with 1..n and nothing else.
func List(vs ...fkdata.V) fkdata.V { return fkdata.Arr(vs...) }

// Strs is the array of strings that every `flags` is.
//
//go:noinline
func Strs(ss ...string) fkdata.V {
	vs := make([]fkdata.V, len(ss))
	for i, s := range ss {
		vs[i] = fkdata.Str(s)
	}
	return fkdata.Arr(vs...)
}

// Box is a BoundingBox: two corners, each a two-element array.
//
//go:noinline
func Box(x1, y1, x2, y2 float64) fkdata.V {
	return List(List(Num(x1), Num(y1)), List(Num(x2), Num(y2)))
}

// Layers is a CollisionMask's `layers`, whose entries are all `= true`. The
// names are the caller's; nothing here knows which ones exist.
//
//go:noinline
func Layers(names ...string) fkdata.V {
	kvs := make([]fkdata.KV, len(names))
	for i, n := range names {
		kvs[i] = F(n, fkdata.Bool(true))
	}
	return Obj(kvs...)
}

// EmptySprite is core's own `util.empty_sprite()`, written out.
//
// THE VALUE IS COPIED FROM core/lualib/util.lua RATHER THAN CALLED, and it has
// to be: `util` is a Lua library, a data guest has no Lua, and there is no
// fkdata primitive that could reach it. It is four fields and it has not changed
// since 1.0:
//
//	{ filename = "__core__/graphics/empty.png", priority = "extra-high",
//	  width = 1, height = 1 }
//
// The Lua stand-ins this replaces called `util.empty_sprite()` after their own
// `local util = require("util")`, so unlike the shipped guest's legacy stub
// there was no undeclared-global hazard here to kill -- what this removes is the
// require, not a load-order bet.
//
//go:noinline
func EmptySprite() fkdata.V {
	return Obj(
		F("filename", Str("__core__/graphics/empty.png")),
		F("priority", Str("extra-high")),
		F("width", Num(1)),
		F("height", Num(1)),
	)
}
