package main

// WHETHER A BELT MAY TURN AS IT LEAVES A BALANCER: the policy behind
// compile.go's `curvesFromCluster`, and the flip that re-classifies the save
// when a player moves it.
//
// ---------------------------------------------------------------------------
// WHY THERE IS A SETTING AT ALL
// ---------------------------------------------------------------------------
//
// The curve rule is a change to what an EXISTING factory means. A belt line that
// merely started beside a balancer part was nothing before 0.3.3 -- the
// incumbent's accepted limitation, inherited -- and is that part's output now.
// So a player who updates gets ports they did not lay, and on 2.1 a part that
// was already serving a belt is then asked for a second one and the whole
// balancer is refused until they take the line away. That is a real cost paid by
// somebody who was not asking for the feature, and this setting is what they
// turn off: with it off, such a belt is left unconnected exactly as it was.
//
// THE DEFAULT IS THE NEW BEHAVIOUR. The feature was asked for, the shape it
// declines (a belt whose rear is fed) is the one that would half-fill a port,
// and a default that shipped off would be a feature nobody found.
//
// ---------------------------------------------------------------------------
// RUNTIME-GLOBAL, AND HAND-ROLLED
// ---------------------------------------------------------------------------
//
// Runtime-global rather than startup for the reason `bbb-multi-edge-parts` is:
// what it controls is the geometry of machines standing in the save, so it has
// to be one answer for everybody in a multiplayer game and it has to travel with
// the save -- and a player gets to flip it from the Map tab with no restart,
// which is the difference between trying it and not.
//
// It is DEFINED ON BOTH ENGINES, which `bbb-multi-edge-parts` is not: what a
// belt at a balancer's face means is a decision this mod takes, not a fact about
// what the engine permits, so there is nothing for either version to say
// differently. The consequence for the write below is that it needs no
// capability gate.
//
// HAND-ROLLED RATHER THAN DECLARED THROUGH FkRecipes, and that is forced.
// FkRecipes has a `LegacyBoolSetting` that would carry the name across verbatim,
// and it emits `setting_type = "startup"` for every setting it declares
// (go/settings.go) with no way to ask for another kind. A startup setting cannot
// be flipped mid-save and cannot be written by a script, so it is not this
// setting. guest/go/data/settings.go declares it beside the other hand-rolled
// bool, and guest/go/tune's HandRolledSettings is what tells the library's
// locale checker that both exist.

// CurvedExitsSetting is the runtime-global bool, defaulting to TRUE.
//
// WRITTEN OUT RATHER THAN TAKEN FROM `tune`, which owns the same string as
// [tune.SettingCurvedExits] -- and that is forced rather than sloppy. `tune`
// imports FkRecipes, so a control guest that imported it would link the data
// stage's emit layer as well, and the two `main` packages' exports then collide:
// measured here, `tinygo build` of this package with the import in place dies on
// wasm-opt's `parse exception: duplicate export name`. `MultiEdgeSetting` above
// is the same literal for the same reason. What keeps the copies honest is that
// tune's is the one the prototype and the locale checker use, so a rename that
// missed this file would leave the guest reading a setting nobody defines --
// which reads Absent, which curvedExitsAllowed answers ON, which is the shipped
// default and therefore silent. That is the risk, stated; there is no import
// that can close it.
const CurvedExitsSetting = "bbb-curved-exits"

// The cached answer, resolved once per heap.
//
// TRI-STATE AND NOT A BOOL, for the reason edgeCap is: the zero value has to
// mean "not asked yet". A bool would make an unasked cache indistinguishable
// from a save whose player has turned the rule off, and the flip handler below
// reads the previous value to decide whether anything moved.
const (
	curveUnchecked = iota
	curveOff
	curveOn
)

var curveCache uint8

// curveRecheck throws the cached answer away.
//
// Called from the three load-time hooks and from the setting-changed handler,
// which are between them every moment the answer CAN move: a runtime-global
// setting changes through that event or through a load and by no third route.
// It is edgeModeRecheck's discipline exactly, and it sits beside it at all four
// call sites.
func curveRecheck() { curveCache = curveUnchecked }

// curvedExitsAllowed reports whether classifySide may read a belt across a face
// as an output.
//
// ONE INTEGER COMPARE AFTER THE FIRST CALL OF EACH HEAP, which is what it has to
// be: compile.go asks it for every face that has a perpendicular belt across it,
// and the answer gates two `find_entities_filtered` calls. With the rule off the
// probe is never reached, so a save that has turned it off pays what the guest
// paid before the rule existed -- which the `mar` suite's leg H is what holds it
// to.
//
// A SETTING THIS ENGINE CANNOT READ IS ON. It is defined on both engines, so
// `present` is false only for a read that failed, and the two directions are not
// symmetric: answering off would silently disable a rule the player's factory is
// built on, where answering on is the shipped default. It is also what a save
// written before this setting existed produces by a different route -- the
// engine supplies a prototype's default for a key `mod-settings.dat` has no
// entry for, so a 0.3.2 save reads true on its first load here and is
// classified under the new rule with nothing extra to do. Observed rather than
// inferred: only one leg of one suite writes a `mod-settings.dat` (`curv`'s
// second, which needs a save that was already decided), so every other run in
// the estate is on exactly that state -- `m2`'s curve rig compiles at load and
// `sedge`'s `scrv` refusal fires, both of which need this answer to be ON.
func curvedExitsAllowed() bool {
	// A LOAD THAT HAS DECIDED AND HAS NOT YET WRITTEN answers off whatever the
	// cache and the setting say, because the setting has not been asked yet and
	// the cache cannot survive the dispatch. See curveupg.go, curveForcedOff.
	//
	// AND NOT WHILE THAT LOAD IS STILL DECIDING, which is the second term and is
	// about the CHECKLIST rather than about the edges. The probe is what finds an
	// affected balancer, so switching it off at the first one would name one
	// machine to a player whose save has five. The edges it goes on producing are
	// dropped by classifyEdges, which is where the decision is applied.
	if curveForcedOff && !curveUndecided {
		return false
	}
	return curveSettingOn()
}

// curveSettingOn is the SETTING alone: what the save says, cached per heap and
// with none of this load's own state in it.
//
// Split out so that the cache is populated by every path that asks a question
// about the rule. It was not, and the flip handler is what noticed: it compares
// `curveCache` before and after, and a `curvedExitsAllowed` that returned early
// on `curveForcedOff` left the cache reading "not asked yet" -- which compares
// equal to neither answer, so a flip to false announced a change nobody made and
// a flip to TRUE was reported as "left unconnected" and then ignored.
func curveSettingOn() bool {
	if curveCache == curveUnchecked {
		curveCache = curveOn
		if on, present := readGlobalBool(CurvedExitsSetting); present && !on {
			curveCache = curveOff
		}
	}
	return curveCache == curveOn
}

// onCurvedExitsSettingChanged is the flip, and it is the whole reason the
// setting is runtime-global rather than startup: the save re-classifies itself
// under the new rule without a reload.
//
// EVERY CLUSTER IS RE-QUEUED AND ALMOST ALL OF THEM SKIP. The rule decides which
// belts are EDGES, so what a flip moves is the edge list of any cluster with a
// belt across one of its faces and nothing else -- and a cluster whose edge list
// did not move skips on the fingerprint it never lost. So the guest does not try
// to work out which clusters those are: it hands the whole save to the same
// `requeueEveryCluster` the grandfather pass uses, which is affordable for the
// same reason, this being a keypress rather than a tick.
//
// A CLUSTER MAY END UP WITH NO NETWORK, and that is the honest outcome rather
// than a hole. A balancer whose only outputs were curves has inputs and nothing
// else once the rule is off, which is a legitimate half-built state: plan.Build
// declines it, the audit does not count it `unbuilt`, and what its network was
// holding is spilled beside it by the ordinary teardown -- because a machine
// that no longer exists is a removal and this mod's rule for a removal's items
// is that they go back to the world. Turning the rule on again rebuilds it.
//
// NOTHING HERE FLUSHES, for onEdgeModeSettingChanged's reason: the remote
// method's write runs at the outermost level, but a player's keypress can arrive
// anywhere, so this queues and asks for the next tick's flush exactly as an
// ordinary event does.
func onCurvedExitsSettingChanged(name string) {
	if name != CurvedExitsSetting {
		return
	}
	// READ BEFORE INVALIDATING, so that a write of the value already there --
	// which Factorio raises this event for, measured on 2.1.14 -- costs a
	// re-read and nothing else. An unchecked cache compares equal to neither
	// answer and therefore acts, which is the safe direction: acting on a save
	// nothing has asked about yet is a save-wide fingerprint skip.
	before := curveCache
	curveRecheck()
	// AND THE LOAD-TIME DECISION IS SUPERSEDED, because somebody has now said
	// what they want. `curveForcedOff` is this load's answer held between the
	// scan that took it and the write that records it, and the only way it
	// survives a settle is a write that FAILED -- after which a player turning
	// the rule back on would otherwise be read as turning it off. Our own write
	// clears it one statement earlier and reaches here with the cache already
	// agreeing, so this changes nothing on that path.
	curveForcedOff = false
	on := curvedExitsAllowed()
	if curveCache == before {
		return
	}
	logStart("curved exits: a belt across a balancer's face is ")
	if on {
		logS("an output again")
	} else {
		logS("left unconnected")
	}
	logS("; ")
	logU(requeueEveryCluster())
	logS(" clusters re-queued, and every one whose edges did not move skips")
	logEnd()
	requestFlush()
}
