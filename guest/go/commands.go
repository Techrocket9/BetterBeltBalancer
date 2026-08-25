package main

// THE OPERATOR SURFACE: one console command and one remote interface, through
// FkLua's callback seam.
//
// WHAT THIS CLOSES. `auditAll` has been the mod's diagnostic since M3 -- "does
// every cluster's stored fingerprint still equal the one a from-scratch
// classification of the world produces?" -- and until this file existed the only
// way to reach it was to place a `bbb-audit` entity, which is hidden and
// script-placeable only. That is exactly right for the test harness and useless
// to a player: somebody with a balancer misbehaving in their own save could not
// ask the question without writing a mod. `/bbb-audit` is the same call with a
// door on it.
//
// THE MARKER STAYS AND IS NOT REDUNDANT. It is the SYNCHRONOUS trigger every
// test mod's `on_init` uses, and `--create` never reaches a tick, so without it
// every network in every suite's save would compile on the first tick of the
// benchmark instead of into the save. A console command cannot be issued from
// `on_init` and cannot be issued by a headless run at all. Two doors onto one
// room, for two callers that cannot use each other's.
//
// HOW THE COMMAND IS VERIFIED, given that it cannot be triggered from script.
// 2.0.77 has no `commands.run_command`, so no suite can type `/bbb-audit`. The
// remote interface is what makes the seam testable anyway: ONE export serves
// both, id-dispatched, with no branch below that can tell which arrived -- so
// `remote.call('better-belt-balancer', 'audit')` exercising the handler is
// evidence about the command leg, and the command leg itself is asserted as far
// as Factorio's own registry (`commands.commands['bbb-audit'] ~= nil`), which is
// the engine's table rather than this mod's claim about itself. That is the
// qol-research port's pattern and the reasoning is its.

import "github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"

// The id space is this guest's own: the host stores nothing but the closure that
// captures one, so these are not registered anywhere and cannot collide with
// anything outside this file.
const (
	callAudit = 1

	// AND THE ONE DOOR ONTO `bbb-multi-edge-parts` THAT IS NOT THE SETTINGS GUI.
	//
	// WHY IT HAS TO EXIST, measured rather than assumed: Factorio refuses
	// `settings.global[k] = v` from anybody but the owning mod --
	// "Settings can only be changed by the owning player or the mod that made
	// the setting" -- and a runtime-global has no owning player. So the mod that
	// DEFINES the setting is the only script in the game that can write it, and
	// a human is the only other way it moves. That makes every transition of the
	// flip handler (guest/go/sedge.go: the requeue, the scan, the veto)
	// unreachable from a test mod, on the one engine where they exist at all.
	//
	// This is the `bbb-audit` argument exactly, and it is the third time this
	// repository has made it: a path only a human can reach is a path whose bugs
	// a player finds, and the alternative is a second implementation of the
	// thing under test. It ships for the same reason the audit and the insert
	// probe ship, and it answers a question a mod or scenario author has a
	// legitimate reason to ask -- there is no other programmatic way to set this
	// mod's compatibility mode.
	//
	// IT IS INERT ON 2.1 BY CONSTRUCTION and not by a branch here:
	// `writeMultiEdgeSetting` is gated on the capability marker, which is absent
	// there, so the method exists, returns false and writes nothing.
	callSetMultiEdge = 2
)

// The names. `CmdAudit` deliberately matches the marker prototype's name --
// they are the same operation and a player who has read the mod's log lines
// should not have to learn a second word for it. The two live in different
// namespaces (a command table and a prototype table) and cannot collide.
const (
	CmdAudit    = "bbb-audit"
	RemoteIface = "better-belt-balancer"
)

// REGISTRATION HAPPENS IN init AND THAT IS NOT A STYLE CHOICE. A command
// registration is not saved: Factorio re-executes control.lua on every load, so
// it has to be made on every load. `init` is what `_initialize` runs, so this
// gets it by construction -- no `storage` flag, no on_load re-arm, and no way
// for the two to disagree. A registration made from `fk_on_init` would exist in
// the session that created the map and in no other, and the failure would be
// invisible until somebody loaded a save.
//
// This is the same rule the event subscriptions in main.go's `init` follow, for
// the same reason.
func init() {
	// The help string is a LocalisedString, which is a tier-2 value; a plain
	// string is a legal one and is what this is. A localised key would be
	// `fkapi.OfArray(fkapi.OfString("better-belt-balancer.cmd-audit"))` and is
	// not used because this mod ships no locale entry for it, and a missing key
	// renders in the console as an error rather than as nothing.
	fkapi.AddCommand(callAudit, CmdAudit, fkapi.OfString(
		"Re-classify every balancer from the world and report drift, then repair it."))

	// The remote form. It is what makes the seam drivable from a script, which
	// is the only reason any of this is testable -- see the header.
	fkapi.AddInterface(RemoteIface,
		fkapi.InterfaceMethod{Name: "audit", ID: callAudit},
		fkapi.InterfaceMethod{Name: "set-multi-edge-parts", ID: callSetMultiEdge})
}

// fk_on_call is the whole inbound surface for both: one export, id-dispatched,
// exactly like `fk_on_event`.
//
// `argp` is a tier-2 ARRAY of the arguments as they arrived and `retp` is one
// tier-2 slot for the result. A command's trampoline ignores what is in `retp`;
// a remote method's caller does not, so the audit's cluster count is written
// there for the caller that has somewhere to put it. Neither leg reads `argp`:
// the audit takes no arguments, and a command's single `CustomCommandData`
// table has nothing in it this mod needs.
//
//go:wasmexport fk_on_call
func onCall(id, argp, retp uint32) uint32 {
	switch id {
	case callSetMultiEdge:
		// THE ONE LEG THAT READS `argp`. It is the tier-2 array of the arguments
		// as they arrived, so `remote.call(iface, 'set-multi-edge-parts', true)`
		// is a one-element array holding a bool. Anything else -- no argument, a
		// number, a string -- is read as FALSE rather than refused, which is the
		// safe direction: the default is off and the mode a caller cannot name is
		// the mode this engine's successor enforces.
		on := false
		if v := fkapi.ReadDyn(argp); v.Tag == fkapi.TagArray && len(v.Array) > 0 {
			on = v.Array[0].Tag == fkapi.TagBool && v.Array[0].Bool
		}
		// The SAME write the grandfather pass makes, and therefore the same
		// synchronous `on_runtime_mod_setting_changed` re-entry a player's flip
		// produces -- everything sedge.go does about a flip has happened by the
		// time this returns, except the deferred flush it asked for. A remote
		// dispatch is an outermost one, so there is no drain to re-enter and no
		// carry transaction to file against, which is what makes this legal here
		// and illegal from inside a flush.
		ok := writeMultiEdgeSetting(on)
		logStart("remote set-multi-edge-parts=")
		if on {
			logS("true")
		} else {
			logS("false")
		}
		if !ok {
			logS(" REFUSED: this Factorio does not define the setting")
		}
		logEnd()
		fkapi.WriteDyn(retp, fkapi.OfBool(ok))

	case callAudit:
		_ = argp
		logStart("command ")
		logS(CmdAudit)
		logS(": re-classifying every cluster from the world")
		logEnd()
		// `auditAll` ends in a `flush()`, so this dispatch does the compiling
		// too rather than deferring it -- which is what the marker does and is
		// what makes the answer describe the world the caller is looking at.
		fkapi.WriteDyn(retp, fkapi.OfNumber(float64(auditAll())))

	default:
		return uint32(fkapi.StatusNoMember)
	}

	// The same pacing rule as every other entry point: ask for the flush, never
	// collect here. A command dispatch IS an outermost one, but `auditAll`'s own
	// `flush()` recompiles, and a recompile raises build events that re-enter
	// this guest -- so by the time this line runs the guest has been re-entered
	// and returned. Asking is what gc.go's header requires; collecting would be
	// the safe-point violation it exists to avoid.
	gcArmIfNeeded()
	return 0
}
