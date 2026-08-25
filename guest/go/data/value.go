package main

import "github.com/Techrocket9/fklua/guest/go/fkdata"

// The prototype-table shorthands.
//
// A data stage is one long table literal, and spelled out in full every field
// of it reads `fkdata.KVs("type", fkdata.Str("item"))` -- four tokens of
// ceremony around two of content, repeated a few hundred times. These are the
// same constructors under names short enough that a prototype in Go looks like
// the prototype it is:
//
//	obj(
//	    f("type", str("item")),
//	    f("stack_size", num(50)),
//	    f("flags", strs("placeable-neutral", "player-creation")),
//	)
//
// Nothing here adds behaviour and nothing here validates. Factorio's own
// `data:extend` is the validator -- it refuses a prototype with no type or no
// name, by name, which is a better message than anything this layer could
// invent, and the real prototype check happens in C++ after every mod's stage
// has run.

//go:noinline
func obj(kvs ...fkdata.KV) fkdata.V { return fkdata.Obj(kvs...) }

// f is one field of a prototype table.
func f(key string, v fkdata.V) fkdata.KV { return fkdata.KVs(key, v) }

func str(s string) fkdata.V  { return fkdata.Str(s) }
func num(x float64) fkdata.V { return fkdata.Num(x) }

// list is a Lua array -- a table with 1..n and nothing else.
func list(vs ...fkdata.V) fkdata.V { return fkdata.Arr(vs...) }

// strs is the array of strings that every `flags` is.
//
//go:noinline
func strs(ss ...string) fkdata.V {
	vs := make([]fkdata.V, len(ss))
	for i, s := range ss {
		vs[i] = fkdata.Str(s)
	}
	return fkdata.Arr(vs...)
}

// yes and no are shared VALUES rather than constructors, which is safe because
// nothing mutates a V and the codec copies a map's pairs before sorting them.
var (
	yes = fkdata.Bool(true)
	no  = fkdata.Bool(false)
)

// box is a BoundingBox: two corners, each a two-element array.
//
//go:noinline
func box(x1, y1, x2, y2 float64) fkdata.V {
	return list(list(num(x1), num(y1)), list(num(x2), num(y2)))
}

// layers is a CollisionMask's `layers`, whose entries are all `= true`. The
// names are the caller's; nothing here knows which ones exist.
//
//go:noinline
func layers(names ...string) fkdata.V {
	kvs := make([]fkdata.KV, len(names))
	for i, n := range names {
		kvs[i] = f(n, yes)
	}
	return obj(kvs...)
}

// emptySprite is core's own `util.empty_sprite()`, written out.
//
// THE VALUE IS COPIED FROM core/lualib/util.lua RATHER THAN CALLED, and it has
// to be: `util` is a Lua library, a data guest has no Lua, and there is no
// fkdata primitive that could reach it. It is four fields and it has not
// changed since 1.0:
//
//	{ filename = "__core__/graphics/empty.png", priority = "extra-high",
//	  width = 1, height = 1 }
//
// A drawable-but-empty sprite rather than an ABSENT one, everywhere it is used
// here, and that is a decision the Lua took first: both are legal, but headless
// Factorio never opens a sprite file and test/check-sprites.py only checks that
// the paths we name exist, so the GRAPHICAL client is the first thing that
// would notice a shape the engine did not like. This is the shape this repo has
// already loaded in a real game.
//
// If core ever changes it, `test/check-datastage.py` is what says so: the value
// reaches five prototypes and the dump hashes all five.
//
//go:noinline
func emptySprite() fkdata.V {
	return obj(
		f("filename", str(emptyPNG)),
		f("priority", str("extra-high")),
		f("width", num(1)),
		f("height", num(1)),
	)
}

const emptyPNG = "__core__/graphics/empty.png"

// at is one prototype's place in data.raw, so that patching a clone reads as
// one line per field instead of one repeated two-element prefix per field.
//
//	linked := at{"linked-belt", "bbb-linked-belt"}
//	linked.set(num(speed), "speed")
//	linked.drop("minable")
type at struct{ typ, name string }

// set writes one field of this prototype, at any depth below it.
func (p at) set(v fkdata.V, field ...any) {
	path := make([]any, 0, len(field)+2)
	path = append(path, p.typ, p.name)
	path = append(path, field...)
	fkdata.Set(v, path...)
}

// drop DELETES a field rather than writing false, which is the whole reason
// fkdata.Set spells nil separately. Stripping a cloned prototype is eleven
// deletions, and a "write false" reading of them would leave eleven fields
// present-and-false in the prototype and in the dump.
//
// A field that is not there is not an error: only an INTERMEDIATE step of the
// path has to exist, and for every call here that is the prototype itself.
func (p at) drop(field ...any) { p.set(fkdata.Nil(), field...) }
