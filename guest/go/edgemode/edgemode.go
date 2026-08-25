// Package edgemode is the decision behind "may this compiler put two belts on
// one balancer part", and it is a separate package for one reason: THE ENGINE IT
// IS ABOUT CANNOT RUN IT.
//
// The rule Factorio 2.1 forces is one belt per part (guest/go/sedge.go,
// agents/single-edge.md). Multi-edge survives on 2.0 behind a runtime-global
// setting, and the setting can only be DEFINED on 2.0 -- so every interesting
// state of this fold (the setting reading true, a player flipping it, the
// grandfather pass writing it) is unreachable from a 2.1 headless run, which is
// the only Factorio this repository has. Written out inside `main` it would be
// four `if`s nothing could ever execute; written here, `make check` proves all
// eighteen states of it under an ordinary Go toolchain.
//
// It is the third package to earn that treatment after `plan`, `skin` and
// `carry`, and for the same reason each of those did: the question is pure, the
// trigger is not.
//
// ---------------------------------------------------------------------------
// THE TWO INPUTS, WHICH THE FIRST DESIGN CONFLATED
// ---------------------------------------------------------------------------
//
//	CAN the engine stack?    the `bbb-can-stack` marker prototype, emitted by the
//	                         data stage on 2.0.x and never on 2.1.x. A FACT about
//	                         the Factorio the save is open in.
//	MAY the compiler use it? `settings.global["bbb-multi-edge-parts"]`, a
//	                         runtime-global bool defaulting to false. A POLICY,
//	                         per save.
//
// The effective rule is the AND, and the marker is the OUTER term: on 2.1 the
// setting is not defined at all, so it reads Absent and the AND is false whatever
// anybody wanted. That asymmetry is load-bearing rather than tidy -- see
// GrandfatherNeeded, where writing a key this engine does not define RAISES.
package edgemode

// Setting is what `settings.global["bbb-multi-edge-parts"]` said.
//
// ABSENT IS NOT AN ERROR AND IS NOT OFF-BY-ANOTHER-NAME. Reading an undefined
// runtime setting returns nil with no raise (measured on 2.1.14), so Absent is
// the ordinary answer on every 2.1 game and on any 2.0 game whose settings stage
// declined to define it. It is kept distinct from Off because the two differ for
// exactly one caller: the grandfather pass may WRITE an Off and may never write
// an Absent.
type Setting uint8

const (
	SettingAbsent Setting = iota
	SettingOff
	SettingOn
)

// Mode is the reconciliation anchor: the mode the guest's REGISTRY was last
// reconciled under, which lives in the guest heap and therefore in the save.
//
// It is not a cache of the setting. It is the answer to "what did the networks
// standing in this world get built to", which is what a flip has to be compared
// against -- the setting can move between two loads of the same save, and the
// standing networks cannot.
type Mode uint8

const (
	// ModeUnknown is a fresh heap: nothing has been reconciled yet. It compares
	// equal to nothing, so a flip arriving on one always acts -- and acting on an
	// empty registry is a no-op, which is the safe direction.
	ModeUnknown Mode = iota
	ModeSingle
	ModeMulti
)

// Action is what a change of mode obliges the guest to do to the world.
type Action uint8

const (
	// ActNone: the setting and the registry already agree. This is what a
	// SAME-VALUE WRITE produces, and that is not a defensive case -- Factorio
	// raises on_runtime_mod_setting_changed for a write of the value already
	// there (measured on 2.1.14), and the grandfather pass deliberately writes
	// the anchor before the setting so that its own re-entrant event lands here.
	ActNone Action = iota

	// ActSweep: multi-edge was allowed and is not any more, so the guest has to
	// go and LOOK at what is standing.
	//
	// IT DOES NOT MEAN "TEAR THEM DOWN", AND SAYING SO WAS A DEFECT. The design
	// called this a sweep -- condemn every multi-edge network, spill what it
	// held -- and the caller did exactly that until 2026-08-24, when a live 2.0
	// session turned the setting off and watched a full balancer's contents land
	// on the floor. Two things are wrong with a sweep and the first makes the
	// second unreachable by reasoning alone:
	//
	//	IT CANNOT STICK. Getting here means the marker is present (the anchor said
	//	Multi, which requires it), so the very next thing the caller asks is
	//	GrandfatherNeeded(marker, Off, n) -- and that is `n > 0`, which is exactly
	//	the condition under which a sweep finds anything. So a flip-off with
	//	multi-edge balancers standing is always VETOED: the setting is written
	//	back on and the player is told why.
	//	AND A VETOED FLIP MUST BE A NO-OP ON THE WORLD. Everything a sweep tore
	//	down was about to be told it may keep working.
	//
	// So the caller SCANS: it counts and announces and touches nothing (sedge.go,
	// scanStackedMultiEdge). The teardown-and-refuse path survives for the one
	// case that really needs it -- a 2.0 save opened on 2.1, where the engine has
	// already pruned the interfaces and a stacked linked belt standing in that
	// world is a latent engine risk on every load (boskid, forums t=135830:
	// belt-to-belt connections are re-derived at load and one belt-connectable
	// per tile is what makes that unambiguous). Its only producer is
	// `rebuildFromWorld`.
	ActSweep

	// ActRequeue: multi-edge was forbidden and is allowed now. Nothing has to be
	// torn down -- a refused cluster never got a network, and its stored
	// fingerprint never matched the world -- so re-queueing every cluster is
	// enough: the ones that were refused compile, and the rest skip.
	ActRequeue
)

// Effective reports whether the compiler may put two edges on one tile.
//
// The whole rule, and the marker is the outer term.
func Effective(marker bool, s Setting) bool {
	return marker && s == SettingOn
}

// ModeOf is the anchor an effective answer should be recorded as.
func ModeOf(multi bool) Mode {
	if multi {
		return ModeMulti
	}
	return ModeSingle
}

// Reconcile compares what the setting now says against what the registry was
// last reconciled under, and says what that obliges.
//
// The returned Mode is the new anchor and is never ModeUnknown: whatever else
// happens, the guest has just decided.
func Reconcile(marker bool, s Setting, anchor Mode) (Mode, Action) {
	want := ModeOf(Effective(marker, s))
	if want == anchor {
		return want, ActNone
	}
	if want == ModeSingle {
		return want, ActSweep
	}
	return want, ActRequeue
}

// GrandfatherNeeded reports whether this load should flip the setting UP and
// tell the player why.
//
// The spec it implements: a save updated from a release that had no setting and
// multi-edge always on must not have its balancers broken by the update. A save
// with none of them ends with the setting at its false default and is
// single-edge from then on; a save that HAS them keeps multi-edge and is told
// so, once.
//
// THE MARKER TERM IS A CORRECTNESS GATE AND NOT A POLICY ONE. Writing a
// `settings.global` key that this engine does not define RAISES -- measured on
// 2.1.14, `LuaCustomTable doesn't contain key` -- so a pass that asked only
// "are there multi-edge balancers" would take a 2.0 save opened on 2.1, find
// exactly the clusters it is looking for, and raise inside the load. On 2.1 the
// answer is always no, and what speaks instead is the migration summary.
//
// IT IS ONCE PER SAVE BY CONSTRUCTION rather than by a latch: the pass writes the
// setting it is testing, so the next load reads SettingOn and this is false. A
// player who later rebuilds everything single-edge is never flipped back DOWN --
// a silent downgrade under somebody relying on multi-edge is a trap, and turning
// it off is one click of a Map setting that needs no restart.
func GrandfatherNeeded(marker bool, s Setting, multiEdgeClusters uint32) bool {
	return marker && s != SettingOn && multiEdgeClusters > 0
}
