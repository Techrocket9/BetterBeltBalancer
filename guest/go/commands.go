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
const callAudit = 1

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
		fkapi.InterfaceMethod{Name: "audit", ID: callAudit})
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
	_ = argp
	switch id {
	case callAudit:
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
