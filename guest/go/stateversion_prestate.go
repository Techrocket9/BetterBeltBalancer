//go:build prestate

package main

// THE PRE-STATE BUILD IS A TEST FIXTURE AND NOTHING ELSE SHIPS IT.
//
// `curv`'s create phase and its benchmark phase run the same source -- test/run.sh
// makes an upgrade out of a build stamp, not out of two trees -- so with
// stateVersion 1 exported the created save would carry 1 and the load would not
// be undecided, and the suite would be measuring a save that never existed.
//
// -tags prestate is what makes the create phase's guest look like every build
// before 0.3.3: 0 is the number FkLua stores for a guest that does not export
// fk_state_version at all (runtime/lua/fk_mod.lua, `E.fk_state_version and
// E.fk_state_version() or 0`), so a save it writes is byte for byte a 0.3.2
// save as far as the watermark is concerned. NOTHING ELSE MOVES: the classifier,
// the compiler and the curve rule are the shipped ones, which is why the
// observer still has to forge the old-rule networks by laying its curve belts
// with no event.
//
// The control on that equivalence is the spike's genuine 0.3.2 save, which is
// run through the shipped guest by hand rather than by a suite -- see CLAUDE.md,
// "A save from before the curve rule keeps the reading it was built to".
const stateVersion = 0
