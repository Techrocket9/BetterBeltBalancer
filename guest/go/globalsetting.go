package main

// Reading and writing `settings.global`, which this mod does for two
// runtime-global bools and for nothing else.
//
// IT IS ONE PIECE OF MECHANICS AND NOT TWO. `bbb-multi-edge-parts` had these ten
// lines written out inside sedge.go until `bbb-curved-exits` needed the same
// two host calls, and this repository has a standing answer to that shape: a
// second question about one identity asks the same code or does not ask at all
// (guest/go/carry, "A claim is a Region", where the same predicate written twice
// disagreed with itself about a force). What stays in sedge.go is everything
// that is ABOUT multi-edge -- the capability gate on the write, the tri-state
// the fold wants -- and what is here is the custom-table access those two
// callers and curve.go share.
//
// A `settings.global` entry is a ModSetting: a table with one entry. So the read
// is `settings.global[name].value` and the write is `settings.global[name] =
// {value = v}` -- the whole table, never the field. There is no index-assign for
// a field of a value inside a custom table and there does not need to be.

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

const settingValueKey = "value"

// settingKV is the one-entry ModSetting table a write hands over, package level
// so that flipping a setting allocates nothing.
var settingKV [1]fkapi.KeyValue

// readGlobalBool is `settings.global[name].value`, and `present` is false for a
// setting this engine does not define.
//
// NIL IS AN ORDINARY ANSWER AND NOT AN ERROR. Reading an undefined runtime
// setting returns nil and raises nothing (measured on 2.1.14), which is what
// lets `bbb-multi-edge-parts` be read on 2.1 with no version gate in front of
// it. Only the WRITE has to be gated.
//
// `settings.global` is a LuaCustomTable, so this is the `prototypes.entity`
// idiom: the raw handle plus one index read, two host calls, against a
// whole-dictionary attribute that would materialise every runtime setting in the
// game. The handle is taken fresh every time -- no reference outlives its
// dispatch, which is this guest's oldest rule -- and both callers put a
// per-heap cache in front of it.
func readGlobalBool(name string) (on, present bool) {
	raw, err := fkapi.Settings.GlobalRaw()
	if err != nil {
		return false, false
	}
	v, err := fkapi.LuaCustomTable{Object: raw}.Get(fkapi.OfString(name))
	if err != nil || v.Tag != fkapi.TagMap {
		return false, false
	}
	for i := range v.Map {
		if v.Map[i].Key.Tag != fkapi.TagString || v.Map[i].Key.Str != settingValueKey {
			continue
		}
		return v.Map[i].Val.Tag == fkapi.TagBool && v.Map[i].Val.Bool, true
	}
	return false, false
}

// writeGlobalBool is `settings.global[name] = {value = on}`, and the two callers
// of it are the only writes this mod makes to anything outside its own entities.
//
// IT RAISES `on_runtime_mod_setting_changed` SYNCHRONOUSLY, inside the assigning
// statement, so this guest is re-entered before the call returns -- including
// for a write of the value already there (measured on 2.1.14). Every caller has
// to be somewhere a re-entrant handler is legal, and both of this mod's are: the
// grandfather pass runs from flush() after endCarry, and the remote methods are
// dispatched at the outermost level.
//
// WRITING A KEY THIS ENGINE DOES NOT DEFINE RAISES, so a setting that is not
// defined on every engine needs a gate in front of this. `writeMultiEdgeSetting`
// is that gate; `bbb-curved-exits` exists on both and needs none.
//
// It is expressible at all only since FkLua grew an index-assign member kind
// (FKLUA-GAPS.md item 23): the runtime API declares no write side on
// `LuaCustomTable`'s index operator, so the binding is emitted from an allowlist
// over what the description says in prose.
func writeGlobalBool(name string, on bool) bool {
	raw, err := fkapi.Settings.GlobalRaw()
	if err != nil {
		return false
	}
	settingKV[0] = fkapi.KeyValue{Key: fkapi.OfString(settingValueKey), Val: fkapi.OfBool(on)}
	err = fkapi.LuaCustomTable{Object: raw}.Set(fkapi.OfString(name),
		fkapi.Value{Tag: fkapi.TagMap, Map: settingKV[:]})
	return err == nil
}
