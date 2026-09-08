package edgemode

import "testing"

// The fold has six states and every one of them is reachable on either engine,
// which is the difference between it and the rest of this package.
func TestCurveKeepNeeded(t *testing.T) {
	cases := []struct {
		name string
		s    Setting
		n    uint32
		want bool
	}{
		{"on, and a save built to the old reading", SettingOn, 3, true},
		{"on, and nothing built to it", SettingOn, 0, false},
		// A game whose settings stage has not defined the key behaves as the
		// declared default does, which is on.
		{"absent, and a save built to the old reading", SettingAbsent, 1, true},
		{"absent, and nothing built to it", SettingAbsent, 0, false},
		// A SAVE DECIDED ON AN EARLIER LOAD reads Off, and asking again must
		// produce nothing whatever the count says. The count cannot really be
		// non-zero there -- the curve arm that produces one is behind the same
		// setting -- so what this pins is that the fold AGREES with that gate
		// rather than resting on it.
		{"already off, and a save built to the old reading", SettingOff, 3, false},
		{"already off, and nothing built to it", SettingOff, 0, false},
	}
	for _, c := range cases {
		if got := CurveKeepNeeded(c.s, c.n); got != c.want {
			t.Errorf("%s: CurveKeepNeeded(%v, %d) = %v, want %v", c.name, c.s, c.n, got, c.want)
		}
	}
}

// A SAVE AT THE CURRENT STATE RUNG IS COVERED BY THE COUNT AND NOT BY A SPECIAL
// CASE. Its load is never undecided, so no classification in it is ever read as
// evidence, so the count is zero -- which is the same state as a save that
// predates the rule and has no curve belt anywhere in it. A fourth Setting for
// "already decided" would be a second place to keep that in step with
// curveupg.go's watermark; there is none.
func TestAFreshSaveIsNeverFlipped(t *testing.T) {
	for _, s := range []Setting{SettingAbsent, SettingOn, SettingOff} {
		if CurveKeepNeeded(s, 0) {
			t.Errorf("a save with no old-rule clusters was flipped at setting %v", s)
		}
	}
}
