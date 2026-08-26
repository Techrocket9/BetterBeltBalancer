package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/tune"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// THE SETTINGS STAGE. Three settings: two costs on every engine, and one
// engine-gated rule.
//
// ---------------------------------------------------------------------------
// THE TWO COST SETTINGS ARE STARTUP AND THAT IS FORCED, exactly the other way
// round from the rule setting below.
//
// A recipe and a technology are PROTOTYPES. They are built once, at the data
// stage, before a map exists -- so what they cost has to be readable there, and
// `fkdata.StartupSetting` is the only kind that is. A runtime setting would be
// a dropdown that changes nothing until the game is restarted, which is worse
// than a restart prompt.
//
// The consequence a player meets: changing either one restarts Factorio, and
// changing it on an EXISTING save re-costs the recipe under them. That is
// vanilla's own behaviour for every startup setting in the game and it is what
// the "(startup)" tab in the menu means.
//
// DEFINED ON BOTH ENGINES, unlike `bbb-multi-edge-parts`. What a balancer part
// costs means the same thing on 2.0 and on 2.1, so there is no version branch
// here and the `release/2.0` recut carries these two identically.
//
// THE ALLOWED VALUES AND THE DEFAULT COME OUT OF ONE PLACE. Factorio validates
// that `default_value` is a member of `allowed_values` and refuses the mod at
// load when it is not, so the two must not be two lists; guest/go/tune is the
// one list, its head is the default, and `go test ./tune/` also checks every
// value against the locale file -- which is the half no dump and no suite can
// see, because a value with no `[string-mod-setting]` entry renders as
// `Unknown key: ...` in the menu and loads perfectly.
// ---------------------------------------------------------------------------
//
// The engine-gated one:
//
// `bbb-multi-edge-parts` is the per-save policy half of the 2.1 port's rule:
// Factorio 2.1 allows one belt-connectable per tile, so a balancer part carries
// one interface and therefore serves ONE BELT (guest/go/sedge.go,
// agents/single-edge.md). On 2.0 the collision-mask loophole that permitted two
// is still open, and a save built before the rule existed must not be broken by
// an update -- so multi-edge survives there, opt-in, defaulting to off.
//
// ---------------------------------------------------------------------------
// RUNTIME-GLOBAL, AND THAT IS FORCED RATHER THAN PREFERRED
// ---------------------------------------------------------------------------
//
// The grandfather pass has THE MOD flip this setting -- a save updated from the
// release that had no setting keeps its multi-edge balancers working, and the
// only way to express that is for the control guest to write `true` on the first
// load (guest/go/sedge.go, `grandfatherMultiEdge`). A script can write
// `settings.global` and can NEVER write a startup setting: measured on 2.1.14,
// `settings.startup` answers `LuaCustomTable is read only`.
//
// What used to force startup was the collision flag being a data-stage decision,
// and that is dissolved by splitting the two questions the first design
// conflated. CAN the engine stack is a fact about the Factorio version and is
// answered by hidden.go's `bbb-can-stack` marker; MAY the compiler use it is
// this setting. The effective rule is the AND, and guest/go/edgemode is that
// fold with its eighteen states proved under `go test`.
//
// Runtime-global buys two more things startup could not: the player flips it
// mid-save with no restart, and the flip arrives as an ordinary replicated event
// (`on_runtime_mod_setting_changed`) instead of a whole load cycle.
//
// ---------------------------------------------------------------------------
// DEFINED ON 2.0.x AND NEVER ON 2.1.x
// ---------------------------------------------------------------------------
//
// No dead toggles: on 2.1 nothing this setting could say would change what the
// engine permits, so it is not in the menu at all. Two consequences the control
// guest depends on, both measured on 2.1.14:
//
//	READING an undefined runtime setting returns nil and raises nothing, so the
//	guest's policy read needs no version gate -- nil IS the "not defined on this
//	engine" answer (guest/go/sedge.go, `settingMultiEdge`).
//
//	WRITING one RAISES (`LuaCustomTable doesn't contain key ...`), so the
//	grandfather pass's write is gated on the `bbb-can-stack` marker as a
//	CORRECTNESS matter and not as policy. A 2.0 save opened on 2.1 is full of
//	exactly the clusters that pass looks for, so a fold that forgot the marker
//	would raise inside the load of every save the migration exists for. That
//	negative is the one half of this feature a 2.1-only test estate can pin, and
//	`TestGrandfatherNeverWritesWhereTheKeyDoesNotExist` is where it is pinned.
//
// THE VERSION BRANCH IS THE SAME FUNCTION THE DATA STAGE ASKS, and it is a Go
// call rather than a shared file now. Factorio's settings stage is a separate
// Lua state from its data stages with nothing carried across, so when this was
// Lua the two could only agree by requiring one file -- mod-data/engine.lua,
// which existed for that single reason and is deleted. Two exports of one
// compiled module have no second copy to drift. `mods` is visible in this stage
// (measured, along with `feature_flags`), which is what makes the question
// answerable here at all.
//
// ON A 2.1 ENGINE THE BOOL IS NOT EMITTED AT ALL, and `test/check-datastage.py`
// pins that from the other side: trunk's mod-settings dump carries the two
// startup dropdowns and no `runtime-global` entry. It used to be `{}` -- this
// function emitted nothing whatever on 2.1 -- and the two cost settings are why
// it is not any more. The early `return` is still the same early `return` the
// Lua had; what moved is that there are now two settings in front of it.
//
//go:noinline
func settings() {
	fkdata.Extend(
		stringSetting(tune.SettingRecipeCost, tune.RecipeOptions(), "a"),
		stringSetting(tune.SettingTechCost, tune.TechOptions(), "b"),
	)

	if !canStack() {
		return
	}

	fkdata.Extend(obj(
		f("type", str("bool-setting")),
		f("name", str("bbb-multi-edge-parts")),
		// Map, not global-per-user: what it controls is the geometry of machines
		// standing in the save, so it has to be one answer for everybody in a
		// multiplayer game and it has to travel with the save.
		f("setting_type", str("runtime-global")),
		// FALSE, so that a 2.0 save which never used multi-edge is bit-compatible
		// with a fresh single-edge world -- which is the save that upgrades to
		// 2.1 losing nothing. A save that DOES use it is flipped up by the
		// grandfather pass on its first load under this version, once, with a
		// warning.
		f("default_value", no),
		f("order", str("a")),
	))
}

// stringSetting is one startup dropdown: the allowed values, and the FIRST of
// them as the default.
//
// The default being the head of the list rather than a second argument is what
// makes "the default is a value the engine will accept" true by construction.
// Factorio refuses a mod whose `default_value` is not in `allowed_values`, by
// name, at load -- so the two coming from one slice removes the only way to get
// that wrong.
//
//go:noinline
func stringSetting(name string, values []string, order string) fkdata.V {
	return obj(
		f("type", str("string-setting")),
		// STARTUP, because what it decides is a prototype. See the header.
		f("setting_type", str("startup")),
		f("name", str(name)),
		f("default_value", str(values[0])),
		f("allowed_values", strs(values...)),
		f("order", str(order)),
	)
}
