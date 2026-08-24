package edgemode

import "testing"

// allSettings and allModes are what the tests below sweep. Three by three by two
// is eighteen states, which is small enough to check exhaustively rather than at
// the interesting points -- and exhaustive is what this package is for: nothing
// in a headless 2.1 run can reach a Setting that is not Absent.
var (
	allSettings = []Setting{SettingAbsent, SettingOff, SettingOn}
	allModes    = []Mode{ModeUnknown, ModeSingle, ModeMulti}
	allMarkers  = []bool{false, true}
)

func (s Setting) String() string {
	switch s {
	case SettingAbsent:
		return "absent"
	case SettingOff:
		return "off"
	}
	return "on"
}

func (m Mode) String() string {
	switch m {
	case ModeUnknown:
		return "unknown"
	case ModeSingle:
		return "single"
	}
	return "multi"
}

func (a Action) String() string {
	switch a {
	case ActNone:
		return "none"
	case ActSweep:
		return "sweep"
	}
	return "requeue"
}

// THE ONE THAT MATTERS ON THE ENGINE THIS REPOSITORY CAN RUN. On 2.1 the marker
// prototype is not defined, and every path that could put two belts on one part
// has to be shut whatever the setting happens to hold -- including the case where
// a 2.0 save arrives carrying a `true` the grandfather pass wrote on the other
// engine, which is exactly the save the migration suite loads.
func TestFactorio21CannotBeTalkedIntoIt(t *testing.T) {
	for _, s := range allSettings {
		if Effective(false, s) {
			t.Errorf("Effective(marker=false, %v) is true: 2.1 cannot stack two "+
				"belt-connectables on one tile and no setting may say otherwise", s)
		}
		if ModeOf(Effective(false, s)) != ModeSingle {
			t.Errorf("the anchor for (marker=false, %v) is not single", s)
		}
	}
}

// The AND, stated from the other side: on 2.0 the setting is what decides, and
// the default (false) makes a 2.0 save bit-compatible with a fresh single-edge
// world -- which is the whole point of the mode existing.
func TestOn20TheSettingDecides(t *testing.T) {
	if Effective(true, SettingOn) != true {
		t.Fatal("a 2.0 engine with the setting on must allow multi-edge")
	}
	for _, s := range []Setting{SettingAbsent, SettingOff} {
		if Effective(true, s) {
			t.Errorf("Effective(marker=true, %v) is true; only an explicit on counts", s)
		}
	}
}

// THE WRITE IS GATED ON THE MARKER AS A CORRECTNESS MATTER, NOT AS POLICY.
// `settings.global[k] = v` for a k this engine does not define RAISES (measured
// on 2.1.14), and a 2.0 save opened on 2.1 is full of exactly the clusters this
// pass looks for -- so a fold that forgot the marker would raise inside the load
// of every save the migration feature exists for.
func TestGrandfatherNeverWritesWhereTheKeyDoesNotExist(t *testing.T) {
	for _, s := range allSettings {
		for _, n := range []uint32{0, 1, 21, 4096} {
			if GrandfatherNeeded(false, s, n) {
				t.Errorf("GrandfatherNeeded(marker=false, %v, %d) is true: that is "+
					"a write of an undefined settings key, which raises", s, n)
			}
		}
	}
}

// A clean save simply stays at the false default. There is nothing to keep
// working, so there is nothing to say and nothing to write.
func TestACleanSaveIsNotGrandfathered(t *testing.T) {
	for _, m := range allMarkers {
		for _, s := range allSettings {
			if GrandfatherNeeded(m, s, 0) {
				t.Errorf("GrandfatherNeeded(%v, %v, 0) is true with no multi-edge "+
					"balancer in the save", m, s)
			}
		}
	}
}

// ONCE PER SAVE, AND BY CONSTRUCTION RATHER THAN BY A LATCH: the pass writes the
// setting it tests, so the load after it reads SettingOn and this is false. A
// second latch would be a second thing to get wrong.
func TestGrandfatherIsOncePerSave(t *testing.T) {
	if !GrandfatherNeeded(true, SettingOff, 21) {
		t.Fatal("a 2.0 save with 21 multi-edge balancers and the setting off must " +
			"be grandfathered")
	}
	if !GrandfatherNeeded(true, SettingAbsent, 21) {
		t.Fatal("an unreadable setting on a 2.0 engine must still grandfather: the " +
			"balancers are standing either way")
	}
	if GrandfatherNeeded(true, SettingOn, 21) {
		t.Fatal("a save already grandfathered must not be grandfathered again, or " +
			"the warning is a nag on every load")
	}
}

// A SAME-VALUE WRITE IS NOT A HYPOTHETICAL. Factorio raises
// on_runtime_mod_setting_changed for a write of the value already there
// (measured on 2.1.14), so the handler's comparison against the anchor is what
// stops the grandfather pass's own write from sweeping the balancers it just
// decided to keep.
func TestAgreementIsSilent(t *testing.T) {
	for _, m := range allMarkers {
		for _, s := range allSettings {
			anchor := ModeOf(Effective(m, s))
			want, act := Reconcile(m, s, anchor)
			if act != ActNone {
				t.Errorf("Reconcile(%v, %v, anchor=%v) wants %v: a write that "+
					"changed nothing must oblige nothing", m, s, anchor, act)
			}
			if want != anchor {
				t.Errorf("Reconcile(%v, %v, %v) moved the anchor to %v", m, s, anchor, want)
			}
		}
	}
}

// The two real flips, on the one engine that can make them.
func TestFlippingOffSweepsAndFlippingOnRequeues(t *testing.T) {
	want, act := Reconcile(true, SettingOff, ModeMulti)
	if want != ModeSingle || act != ActSweep {
		t.Fatalf("flipping multi-edge OFF on 2.0 gave (%v, %v), want (single, sweep): "+
			"the standing multi-edge networks have to come down", want, act)
	}
	want, act = Reconcile(true, SettingOn, ModeSingle)
	if want != ModeMulti || act != ActRequeue {
		t.Fatalf("flipping multi-edge ON gave (%v, %v), want (multi, requeue): the "+
			"refused clusters never got a network and never matched their "+
			"fingerprint, so re-queueing is the whole of it", want, act)
	}
}

// A FRESH HEAP MUST NEVER SILENTLY AGREE. ModeUnknown is what a rebuilt guest
// starts with, and comparing equal to either real mode would let a flip arrive
// on a registry that had never been reconciled and be dropped.
func TestAFreshAnchorAlwaysActs(t *testing.T) {
	for _, m := range allMarkers {
		for _, s := range allSettings {
			want, act := Reconcile(m, s, ModeUnknown)
			if act == ActNone {
				t.Errorf("Reconcile(%v, %v, unknown) is silent; a heap that has "+
					"reconciled nothing agrees with nothing", m, s)
			}
			if want == ModeUnknown {
				t.Errorf("Reconcile(%v, %v, unknown) left the anchor unknown", m, s)
			}
		}
	}
}

// Totality, which is the property a fold gets tested for when nothing else can
// execute it: every one of the eighteen states resolves to a real anchor and to
// an action that agrees with what Effective says.
func TestEveryStateResolvesAndAgreesWithEffective(t *testing.T) {
	for _, m := range allMarkers {
		for _, s := range allSettings {
			for _, anchor := range allModes {
				want, act := Reconcile(m, s, anchor)
				if want != ModeOf(Effective(m, s)) {
					t.Errorf("Reconcile(%v, %v, %v) wants %v but Effective says %v",
						m, s, anchor, want, Effective(m, s))
				}
				switch {
				case anchor == want && act != ActNone:
					t.Errorf("(%v, %v, %v): agreement obliged %v", m, s, anchor, act)
				case anchor != want && want == ModeSingle && act != ActSweep:
					t.Errorf("(%v, %v, %v): losing multi-edge obliged %v, not a sweep",
						m, s, anchor, act)
				case anchor != want && want == ModeMulti && act != ActRequeue:
					t.Errorf("(%v, %v, %v): gaining multi-edge obliged %v, not a requeue",
						m, s, anchor, act)
				}
			}
		}
	}
}

// The two features do not overlap, and the marker is what separates them: a load
// either offers to keep multi-edge (2.0) or refuses and explains (2.1), never
// both and never neither.
func TestOneScanTwoOutcomesChosenByTheMarker(t *testing.T) {
	const found = 21
	for _, m := range allMarkers {
		grand := GrandfatherNeeded(m, SettingOff, found)
		migrate := !Effective(m, SettingOff) && found > 0
		if !migrate {
			t.Fatalf("marker=%v: a save with %d multi-edge balancers and the "+
				"setting off must be one of the two cases", m, found)
		}
		if m && !grand {
			t.Fatal("2.0 with the setting off must grandfather rather than migrate")
		}
		if !m && grand {
			t.Fatal("2.1 must migrate rather than grandfather")
		}
	}
}

// ---------------------------------------------------------------------------
// THE CONVERSION ORIGIN, 2026-08-24
// ---------------------------------------------------------------------------
//
// Until the legacy conversion became a producer of the migration summary, every
// caller reached this fold with clusters the rebuild had ADOPTED: standing
// networks, built by an earlier release, running. A Belt Balancer 2 or 3
// conversion is the other shape entirely -- `legacyScan` creates those parts and
// the very next flush REFUSES them, so they have no network at all and never had
// one -- and it reaches the same fold on the same load.
//
// The two tests below are what that origin obliges, and neither is reachable
// from a headless 2.1 run: the marker is absent there, so `GrandfatherNeeded` is
// false whatever else is true, and this package is the only machine in the
// repository that can execute the 2.0 arm.

// PROVENANCE DOES NOT ENTER THE FOLD, and that is the decision rather than an
// accident of the signature. A player who uninstalls Belt Balancer on 2.0 is
// exactly the base-must-survive case grandfathering exists for: their balancers
// are the incumbent's idiom, which is two belts on every part, so a fold that
// treated a converted cluster as somehow less deserving would refuse every one
// of them on the load that adopted them -- the mod breaking a base at the moment
// it took responsibility for it.
func TestAConvertedSaveIsGrandfatheredLikeAnyOther(t *testing.T) {
	for _, tc := range []struct {
		name    string
		marker  bool
		setting Setting
		n       uint32
		want    bool
	}{
		{"a 2.0 save updated from the release with no setting", true, SettingOff, 21, true},
		{"a 2.0 Belt Balancer save just converted, same count", true, SettingOff, 21, true},
		{"...and one balancer is enough", true, SettingOff, 1, true},
		{"...with the setting unreadable, which is still a 2.0 world", true, SettingAbsent, 7, true},
		{"...already grandfathered on an earlier load", true, SettingOn, 7, false},
		{"a conversion that produced nothing multi-edge", true, SettingOff, 0, false},
		{"the same conversion on 2.1, where the key does not exist", false, SettingOff, 7, false},
	} {
		if got := GrandfatherNeeded(tc.marker, tc.setting, tc.n); got != tc.want {
			t.Errorf("%s: GrandfatherNeeded(%v, %v, %d) = %v, want %v",
				tc.name, tc.marker, tc.setting, tc.n, got, tc.want)
		}
	}
}

// AND WHAT THE WRITE OBLIGES, which the conversion origin is the first caller to
// need at all.
//
// Grandfathering moves the anchor to Multi from Single (a save reconciled under
// the rule) or from Unknown (a fresh heap, which is every load that declines a
// guest heap and therefore every load that can grandfather). Both are gaining
// multi-edge, and Reconcile says both oblige a REQUEUE -- so the guest asks it
// rather than assuming, and every cluster goes back on the build queue after the
// flip. For an adopted cluster that requeue is a fingerprint skip; for a
// converted one it is the ONLY thing that will ever give it a network, because
// the flush that refused it is the flush that just closed.
func TestGrandfatheringObligesARequeue(t *testing.T) {
	for _, anchor := range []Mode{ModeUnknown, ModeSingle} {
		want, act := Reconcile(true, SettingOn, anchor)
		if want != ModeMulti || act != ActRequeue {
			t.Errorf("a grandfather taken on anchor=%v gave (%v, %v), want "+
				"(multi, requeue): a converted balancer was refused seconds ago "+
				"and nothing but a requeue can compile it", anchor, want, act)
		}
	}
	// The one anchor a grandfather cannot be taken on, stated so that the guest's
	// `if act == ActRequeue` is known to be exhaustive rather than hopeful: an
	// anchor already at Multi means the setting was on at the last reconciliation,
	// GrandfatherNeeded is false, and there is nothing to re-queue.
	if _, act := Reconcile(true, SettingOn, ModeMulti); act != ActNone {
		t.Errorf("a save already reconciled under multi-edge obliged %v", act)
	}
	if GrandfatherNeeded(true, SettingOn, 21) {
		t.Fatal("...and it should not have been asked to grandfather at all")
	}
}
