package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/tune"
	"github.com/Techrocket9/fklua/guest/go/fkdata"
)

// THE SETTINGS STAGE. Four settings: two costs and one classification rule on
// every engine, and one engine-gated rule.
//
// ---------------------------------------------------------------------------
// THE TWO COST SETTINGS ARE DECLARED THROUGH FkRecipes SINCE 2026-09-01, and
// this file's whole part in them is that main.go's `fk_settings` hook calls
// `tune.Plan().EmitSettings()` before it calls this function.
//
// FkRecipes is the shared data-stage library. The declarations are
// `guest/go/tune`'s [tune.Plan] -- two `LegacyDropdownSettingNeedingLocale`
// calls, which is the constructor that carries a shipped mod's own names and
// order strings across verbatim, because Factorio keys mod-settings.dat by NAME
// and has no rename mechanism. The acceptance criterion was
// `test/check-datastage.py`'s `mod_settings_sha256` not moving on either mod
// set, and it did not: what this stage emits is byte-identical to what it
// emitted when the two prototypes were built by hand here.
//
// EACH HOOK NAMES THE HALF IT RUNS. `EmitSettings` comes from `fk_settings` and
// `EmitData` from `fk_data`; nothing calls the dispatching `Emit`, because it
// reads the stage at run time and therefore links both planners. main.go's
// header is the long form. `fk_data_final_fixes` does not mention the library
// at all.
//
// STILL STARTUP, AND STILL FORCED. A recipe and a technology are PROTOTYPES,
// built once at the data stage before a map exists, so what they cost has to be
// readable there and `fkdata.StartupSetting` is the only kind that is. FkRecipes
// emits `setting_type = "startup"` for every setting it declares for exactly
// that reason, so the constraint is now the library's rather than this file's.
// The consequence a player meets is unchanged: changing either one restarts
// Factorio, and changing it on an EXISTING save re-costs the recipe under them,
// which is vanilla's behaviour for every startup setting in the game.
//
// DEFINED ON BOTH ENGINES, unlike `bbb-multi-edge-parts`. What a balancer part
// costs means the same thing on 2.0 and on 2.1, so there is no version branch
// over them: a `release/2.0` recut carries these two NAMES unchanged, because
// it carries this file unchanged. What it offers under them is whatever the
// trunk release it was cut from offered, which is not necessarily what head
// offers -- the branch carries the last 2.0 release rather than head, and
// `test/check-release-arm.sh` says which trunk source that is.
//
// THE ALLOWED VALUES AND THE DEFAULT STILL COME OUT OF ONE PLACE. Factorio
// refuses a mod whose `default_value` is not a member of `allowed_values`, by
// name, at load; guest/go/tune is the one list and its head is the default, and
// the library takes both from that one call. `go test ./tune/` checks the
// emitted plan field by field on the host, and checks every value against the
// LOCALE FILE through the library's own `CheckLocaleWith` -- which is the half
// no dump and no suite can see, because a value with no `[string-mod-setting]`
// entry renders as `Unknown key: ...` in the menu and loads perfectly.
//
// WHAT A PLANNING REFUSAL WOULD DO HERE, because `EmitSettings` routes every one
// of them through `fkdata.Raise` and a raise at the settings stage is a mod that
// does not load. Against this mod's constants exactly ONE is reachable, and it
// is not about this mod's settings at all: `fkrecipes: the mod name is empty, so
// nothing can be prefixed; package with an fklua that wires ModName`, which
// fires when `fkdata.ModName()` returns empty because the stage file was written
// by an fklua older than that argument. Both settings here are LEGACY and need
// no prefix, so this mod would fail to load over a name it does not use. It is a
// BUILD-TIME property rather than a runtime one -- the toolchain is pinned by
// `fklua.lock` and the dump gate would fail loudly on the next run -- so nothing
// guards it in code and this paragraph is the record. Every other refusal the
// library can produce is unreachable by construction: an empty setting name, an
// empty legacy order, a duplicate emitted name and a default outside its allowed
// values are all compile-time constants of guest/go/tune (the default IS the head
// of the option list), the numeric refusals need a numeric setting and this plan
// declares none, and a nil World or a zero `Lib` id cannot happen because the
// emit layer builds the first and `tune.Plan` builds the second with `New`.
//
// THE PROTOTYPE HALF FOLLOWED IN ROUND TWO. The item, the recipe and the
// technology are the same plan's, emitted from `fk_data`, and item.go, recipe.go
// and technology.go are gone. What round one measured out was the prefix -- every
// prototype name the library emitted carried the mod's own, so
// `bbb-balancer-part` became `better-belt-balancer-bbb-balancer-part` and
// entity.go's `minable.result` named an item that no longer existed -- plus two
// missing spec fields. The Legacy prototype constructors, `Order` and
// `PlaceResult` are the library's answers to all three. tune/plan.go's header is
// the long form and agents/fkrecipes-migration.md is the run.
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
// ON A 2.1 ENGINE THE MULTI-EDGE BOOL IS NOT EMITTED AT ALL, and
// `test/check-datastage.py` pins that from the other side: trunk's mod-settings
// dump carries the two startup dropdowns and ONE runtime-global -- the
// curved-exit rule -- where the 2.0 arm carries two. It used to be `{}`, this
// function emitting nothing whatever on 2.1, and the cost settings are why it is
// not any more. The early `return` is still the same early `return` the Lua had;
// what moved is what is in front of it.
//
//go:noinline
func settings() {
	// The two cost dropdowns are already out: main.go's fk_settings hook calls
	// `EmitSettings` before this function. What is left here are the two settings
	// FkRecipes has no verb for -- both runtime-global, which the library emits
	// nothing but startup.

	// THE CURVED-EXIT RULE, and it is emitted BEFORE the engine gate rather than
	// after it. What a belt across a balancer's face means is a decision this mod
	// takes and not a fact about what the engine permits, so unlike the bool
	// below it exists on 2.1 as well -- and on that engine this function returns
	// two lines down.
	//
	// DEFAULT TRUE, so that the rule is on for everybody who does not go looking.
	// A player whose existing factory it changes -- a belt line that merely
	// started beside a part is that part's output now -- is who turns it off, and
	// guest/go/curve.go's header is the whole of that argument.
	fkdata.Extend(obj(
		f("type", str("bool-setting")),
		f("name", str(tune.SettingCurvedExits)),
		// Map rather than global-per-user: what it controls is which belts are
		// PORTS of machines standing in the save, so it has to be one answer for
		// everybody in a multiplayer game.
		f("setting_type", str("runtime-global")),
		f("default_value", yes),
		// After the multi-edge bool on the engine that has one, and alone on the
		// engine that does not.
		f("order", str("b")),
	))

	if !canStack() {
		return
	}

	fkdata.Extend(obj(
		f("type", str("bool-setting")),
		f("name", str(tune.SettingMultiEdgeParts)),
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
