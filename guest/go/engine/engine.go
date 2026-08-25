// Package engine is WHICH FACTORIO THIS IS, asked once and answered the same
// way everywhere.
//
// The whole 2.1 port turns on one fact -- can this Factorio put two
// belt-connectables on one tile -- and the data guest has to give the same
// answer in TWO of its hooks:
//
//	fk_data      emits `not_colliding_with_itself` on the linked belt and the
//	             `bbb-can-stack` marker prototype on 2.0.x, and never on 2.1.x
//	             (guest/go/data/hidden.go)
//	fk_settings  defines `bbb-multi-edge-parts` on 2.0.x and never on 2.1.x,
//	             because a setting that cannot do anything is a dead toggle in
//	             the player's menu (guest/go/data/settings.go)
//
// Factorio's settings stage is a SEPARATE LUA STATE from its data stages and
// nothing carries across, so when this was Lua the two could only agree by
// being one required file (mod-data/engine.lua, deleted). THAT PROBLEM IS GONE
// RATHER THAN SOLVED AGAIN: both hooks are exports of one compiled module now,
// so they call one Go function and there is no second copy for an edit to drift
// away from. Two copies of a version match would compile, would work, and would
// be one edit away from a game where the compiler believes it may stack and the
// prototype it would stack cannot -- a silent nil from `create_entity` on every
// second interface, forever.
//
// # Why this is a package and not a function in the data guest
//
// The same reason guest/go/edgemode is a package: THE ENGINE ITS INTERESTING
// STATE LIVES ON IS ONE THIS MACHINE CANNOT RUN. Trunk targets 2.1, so the
// `true` arm -- the collision flag, the marker and the setting -- is unreachable
// from any dump this repository can take, and the `--dump-data` golden for it
// is deferred to the `release/2.0` recut (test/check-datastage.py). Written
// inside the data guest this would be a branch nothing could execute and
// nothing could test, because a package that imports fkdata cannot be built by
// a host toolchain at all: //go:wasmimport is rejected outside GOARCH=wasm.
//
// Here it is ordinary Go over an ordinary string, and `make check` proves every
// arm of it.
//
// # It fails SAFE towards 2.1
//
// Anything unreadable is treated as 2.1, because emitting the flag on 2.1
// refuses the mod at load, while not emitting it on 2.0 merely costs the
// multi-edge geometry -- which the guest then refuses to build rather than
// building wrongly. See agents/single-edge.md and guest/go/sedge.go for the
// runtime half.
package engine

// Is2_0 reports whether a `mods["base"]` version string names Factorio 2.0.x.
//
// The two results of fkdata.ModVersion("base") go straight in, so the caller
// has no branch of its own to get wrong:
//
//	if engine.Is2_0(fkdata.ModVersion("base")) { ... }
//
// THE MATCH IS ON MAJOR.MINOR ALONE, so every 2.0.x point release is 2.0 and
// everything from 2.1.0 on is not. A patch number never moves this answer,
// which is what makes the whole port a property of the SERIES rather than of a
// build.
func Is2_0(version string, present bool) bool {
	if !present {
		return false
	}
	major, rest, ok := cut(version, '.')
	if !ok {
		return false
	}
	// The minor runs to the next '.' where there is one, and to the end of the
	// string where there is not -- so a bare "2.0" answers the same as
	// "2.0.77". Factorio always writes three components; accepting two costs
	// nothing and removes a way for this to be surprising.
	minor, _, _ := cut(rest, '.')
	return major == "2" && minor == "0"
}

// cut splits on the first occurrence of sep. strings.Cut with the standard
// library left out: this package is imported by a wasm guest whose whole job is
// to run once at load, and `strings` is not free there. It is also the entire
// reason this file has no imports at all, which is what lets a data module
// import it -- `fklua mod` refuses a data module that reaches any host module
// but fkdata and env.
func cut(s string, sep byte) (before, after string, found bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
