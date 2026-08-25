package engine

import "testing"

// THIS IS THE ONLY MACHINE-CHECKED PART OF THE 2.0 ARM OF THE DATA STAGE, and
// that is the whole reason the package exists. `test/check-datastage.py` proves
// what the data guest EMITS by hashing Factorio's own `--dump-data`, and it can
// only ever prove the flavour of the binary it runs on -- trunk's is 2.1, so the
// `true` arm's prototypes (the collision flag, the `bbb-can-stack` marker, the
// `bbb-multi-edge-parts` setting) have no dump golden until the release/2.0
// recut takes one.
//
// What is NOT deferred is the decision, and it is the half that could be wrong
// in a way nobody would notice: a version match that answered `true` on 2.1
// would emit `not_colliding_with_itself` and REFUSE THE MOD AT LOAD, and one
// that answered `false` on 2.0 would quietly cost every multi-edge save its
// geometry. Both arms and the fails-safe fallback are pinned here.

func TestEveryPointReleaseOf2_0IsTwoPointZero(t *testing.T) {
	for _, v := range []string{"2.0.0", "2.0.7", "2.0.77", "2.0.100", "2.0"} {
		if !Is2_0(v, true) {
			t.Errorf("base %q read as not 2.0; a patch number must never move this answer", v)
		}
	}
}

func TestEverythingFrom2_1OnIsNot(t *testing.T) {
	for _, v := range []string{"2.1.0", "2.1.14", "2.1.16", "2.2.0", "3.0.0", "2.10.0"} {
		if Is2_0(v, true) {
			t.Errorf("base %q read as 2.0; emitting the collision flag there refuses the mod at load", v)
		}
	}
}

// "2.10" IS NOT "2.1" AND IS NOT 2.0 EITHER, which is the one comparison a
// prefix test gets wrong. It is in the table above as well; it is called out
// here because `strings.HasPrefix(v, "2.0")` -- the obvious implementation --
// also accepts "2.0" of nothing, and a digit-wise minor is what makes it right.
func TestTheMinorIsAWholeComponentAndNotAPrefix(t *testing.T) {
	if Is2_0("2.01.0", true) {
		t.Error(`base "2.01.0" read as 2.0: the minor is being compared as a prefix`)
	}
	if !Is2_0("2.0.1", true) {
		t.Error(`base "2.0.1" read as not 2.0`)
	}
}

// IT FAILS SAFE TOWARDS 2.1, and this is the asymmetry that decides which way.
// Emitting the flag on 2.1 refuses the mod at load -- the player gets no mod at
// all. Not emitting it on 2.0 costs the multi-edge geometry, which the runtime
// then declines to build rather than building wrongly (guest/go/sedge.go). So
// anything unreadable is 2.1.
func TestAnythingUnreadableIs2_1(t *testing.T) {
	for _, v := range []string{"", "x", "2", "two.zero.zero", "1.0.0", ".0.0", "abc.def"} {
		if Is2_0(v, true) {
			t.Errorf("base %q read as 2.0; an unreadable version must fail safe towards 2.1", v)
		}
	}
}

// A base that is not installed at all cannot happen in a loaded mod -- every
// mod depends on base -- so this is the belt-and-braces arm. It matters because
// the caller passes fkdata.ModVersion's two results straight through, so a
// `false` present flag arrives here with a zero version string, and reading
// that as 2.0 would be the worst possible default.
func TestAnAbsentBaseIsNot2_0(t *testing.T) {
	if Is2_0("2.0.77", false) {
		t.Error("a base reported ABSENT was read as 2.0 on the strength of its version string")
	}
	if Is2_0("", false) {
		t.Error("an absent base was read as 2.0")
	}
}
