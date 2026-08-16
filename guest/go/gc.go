package main

// THE COLLECTOR SEAM, AND WHERE THIS GUEST LETS ONE RUN.
//
// FkLua's `--gc=collected` (`../FkLua/agents/gc.md`) is a paced conservative
// mark-sweep over the guest's OWN heap: `-gc=custom` on the TinyGo side, one
// import here, and `fklua mod --gc=collected` on the package. `make GC=...` is
// the whole switch, and COLLECTED IS THE SHIPPED DEFAULT since 2026-08-02 --
// CLAUDE.md's "The third decision" is the measurement that flipped it and
// "The collected-mode postscript" is the whole history.
//
// UNDER `-gc=leaking` THIS FILE COSTS NOTHING. `guest/go/fkgc/off.go` is the
// whole package when the build tag `gc.custom` is absent: every function below
// is a no-op returning zero, with no state and no init. So the leaking build is
// byte-for-byte the build it was before this file existed, which is the property
// that lets both variants stay buildable from one source tree and one flag.
//
// WHERE A COLLECTION MAY START, and this is the only interesting decision in the
// file. `fkgc.CollectIfNeeded` is documented as "the call a guest puts in
// fk_on_tick" -- and this mod has no `fk_on_tick` and must never have one
// (CLAUDE.md: the zero-script steady state is the headline measurement). The
// nearest thing it has is `fk_on_deferred`, the one-shot flush `fk.Defer()`
// registers when an event moved something and tears down again from inside
// itself. That is exactly the right place and for exactly the right reason:
//
//   - it runs at an OUTERMOST DISPATCH BOUNDARY, which is the safe-point
//     precondition every argument in agents/gc.md rests on;
//   - it runs ONCE PER TICK IN WHICH THE WORLD CHANGED, which is precisely when
//     this guest has allocated anything at all;
//   - it does not run at idle, so an idle balancer starts no collection, steps
//     none, registers no `on_tick`, and pays what it paid before. The
//     zero-script property extends to zero-collector, and that is measured
//     rather than argued (CLAUDE.md, the collected-mode postscript).
//
// AND IT IS NOT ENOUGH ON ITS OWN, WHICH THE MARATHON SUITE MEASURED. A flush is
// deferred only when an event QUEUED A CLUSTER, and the guest's
// highest-multiplier path allocates without queuing one: a belt laid anywhere on
// the map with no cluster near it is entered, classified from guest memory and
// rejected, having bought a name. Over the `mar` suite's 680 operations the
// collected arm therefore reached `CollectIfNeeded` on a fraction of the ticks it
// had allocated on, the pacer fell behind, and `fkgc` degraded to leaking and
// said so: 2 forward-progress deadlines and one `fkgc: outrun` line, over 680
// operations. Those were OURS, not the collector's.
//
// `gcArmIfNeeded` below is the fix, and its shape is chosen so that the two
// properties above survive it. It does NOT collect, and it does not add a call
// site where a collection could start at a nested dispatch -- it ASKS FOR THE
// FLUSH THAT ALREADY EXISTS, from the end of `fk_on_event`, when and only when
// the collector says the pressure has reached its own threshold. The collection
// still begins in `fk_on_deferred`, at the same outermost boundary, one tick
// later. An idle guest raises no events, so it arms nothing, registers nothing
// and pays nothing: zero steady-state cost is a property of the trigger being an
// EVENT rather than a tick, and that is unchanged.
//
// WHAT THAT DELIBERATELY DOES NOT COVER is a `--create`. `--create` never
// reaches a tick, so the whole 200-balancer bench build happens inside ONE
// `bbb-audit` dispatch and no paced step can run inside it -- agents/gc.md
// stage A names this shape ("a mass-builder is not covered, at any budget worth
// having") and the honest answer is to let the heap grow and collect afterwards,
// not to start a collection that the save would then carry into the benchmark
// window. A collection left in flight across the save resumes on the first ticks
// after the load, which is the one thing that would put collector steps inside a
// measured idle window. So: not here.

import (
	"runtime"

	"github.com/Techrocket9/fklua/guest/go/fkgc"
)

// gcCollectIfNeeded starts a paced collection if the guest has taken more heap
// than the collector's threshold since the last one. It is a no-op, with no
// call at all after inlining, under `-gc=leaking`.
//
// IT USED TO BE FOLLOWED BY A HAND-ROLLED ROOT-SET CHECK, and that check is
// deleted rather than merely unused. It read `fkgc.RootWords()` on the first
// flush after a completed collection and wrote a `[BBB] error:` line when this
// guest's globals approached the step budget -- which was worth having when
// crossing that line meant a mark phase that could never terminate and nothing
// anywhere said so. Upstream closed that hole on 2026-08-03: `EffectiveBudget()`
// floors the budget at what the scan costs, so the condition is no longer a
// cliff, and the collector logs its own `fkgc:` line the first time the floor
// binds -- naming the cause and saying in terms that `SetBudget` is not the fix.
// Upstream states the principle this guest was violating: "a condition only one
// component can observe is that component's obligation to report", and the
// collector is the only thing holding both `rootWords` and `Budget()`. So the
// duplicate is gone, `test/run.sh` fails a run that contains the upstream line,
// and `logHeap` prints `budget=`/`eff=`/`roots=` so the numbers are still in
// every suite log.
func gcCollectIfNeeded() {
	fkgc.CollectIfNeeded()
}

// gcArmBytes is the heap pressure at which this guest asks for a flush so that
// `fk_on_deferred` can start a collection -- and it is fkgc's OWN default
// threshold rather than a number of our own. `init` installs it, so the two
// cannot drift apart on an upstream bump.
//
// The two must be the same number and not merely close. Arming BELOW the
// collector's threshold defers a flush that `CollectIfNeeded` then declines,
// and since nothing about the pressure has changed it would do that again on
// the next event, and the next: a per-event flush tick that collects nothing.
// Arming ABOVE it is the starvation this exists to fix, moved rather than
// removed.
//
// THAT "CANNOT DRIFT APART" WAS FALSE FOR THE WHOLE LIFE OF THE PACER FIX, and
// it is true now for a reason upstream had to build. `initialize()` assigned
// both knobs their defaults UNCONDITIONALLY, and on `-target=wasm-unknown` at
// TinyGo 0.41.1 it runs AFTER a guest's package initialisers whatever
// `runtime_wasmentry.go` reads like -- so `init`'s `SetThreshold` here was
// written and then overwritten, and this file installed nothing at all. Fixed
// upstream 2026-08-03 by LATCHING a non-zero value (`SetThreshold(0)` means
// "restore the default", so a non-zero field is always something a guest
// asked for), which is a rule about the VALUE rather than about who runs first.
//
// IT COST THIS MOD NOTHING, AND THAT IS AN ACCIDENT RATHER THAN A DESIGN. The
// value being installed was fkgc's `defaultThreshold` -- 256 KiB, the same
// number by construction -- so the collector was left holding exactly what the
// discarded call had asked for, and `gcArmIfNeeded` below (which compares
// against the constant, not against `Stats()`) agreed with it. Measured
// 2026-08-03 across the marathon and edge suites in both arms: not one number
// moved. The mod that found the defect asked for 128 KiB and got the divergence
// this comment describes.
//
// So the discipline the paragraph above states is real and was simply not
// enforced by anything. What enforces it now is the latch: an upstream bump of
// `defaultThreshold` moves the collector, `init` moves it back to `gcArmBytes`,
// and the two still agree. Before the latch that same bump would have split
// them silently -- BBB arming at 256 KiB against a collector that had been
// given something else -- which is the shape upstream calls "it fails in the
// direction that hides itself".
const gcArmBytes = 256 << 10

// gcBudget is one paced step's work allowance in granules of heap touched. It
// is a DELIBERATE DEVIATION from the default, it is chosen against a measured
// number, and the number it is chosen against changed upstream on 2026-08-03 --
// so read `fkgc.SetBudget`'s header and then this before touching it.
//
// WHY THIS GUEST SETS IT AT ALL, in one sentence: `deadlines=` is documented to
// be zero forever, the `mar` suite asserts exactly that, and at the default this
// guest cannot hold it. Upstream states the rule for reading the symptom, and
// this guest now logs both halves of it (see logHeap's `budget=`/`eff=`):
//
//	if Deadlines rises, compare EffectiveBudget() against Budget() FIRST.
//	Equal means [the dirty rate] and this is the knob. Larger means [the
//	root set], the collector has already applied the floor.
//
// MEASURED ON THE `mar` SUITE, 680 world operations, one arm per row, everything
// else held (2026-08-03, after the round that introduced the reserve below):
//
//	SetBudget       budget / eff    paced steps   deadlines
//	none (default)   1024 / 1180         66           7      floor BINDS
//	1024 + reserve   2048 / 2048         12           3
//	4096 + reserve   5120 / 5120          5           0      <- shipped
//	8192 + reserve   9216 / 9216          3           0
//
// `eff > budget` on the first row is the collector applying its own floor and
// saying so in an `fkgc:` line: the ROOT-SET cause, which SetBudget does not fix
// and no longer has to. Every other row has `eff == budget`, which is upstream's
// own test for the DIRTY-RATE cause -- the one this knob does fix, and the one
// this guest actually has. 4096 granules of real work is also the exact remedy
// upstream records for the other mod that met it (guests.md: nixie-tubes, "the
// default gave 15 outruns and 3 mark-termination deadlines, and
// fkgc.SetBudget(4096) gave a clean plateau with neither").
//
// WHY THE RESERVE IS STILL A TERM IN THE SUM, which is the part that is easy to
// get wrong now that upstream handles the root scan itself. A step spends at
// most `budget` IN TOTAL, and the scan is taken OUT of it rather than added on
// top -- `markStep` holds `rootScanCost()` back, gives the queues what is left,
// and adds it again at the attempt. So the real work a step does is
//
//	budget - rootScanCost()    ~= gcBudget - gcRootGranules
//
// and a budget written as a bare 4096 would buy this guest ~2980 granules of
// real work, not 4096. The sum keeps the two terms separate so that the one
// that is a DECISION (how much marking per step, hence the pause) is not
// silently eaten by the one that is a PROPERTY of the guest (how many
// package-level variables it declares).
//
// WHAT UPSTREAM'S 2026-08-03 ROUND DID CHANGE, because this constant used to be
// justified by a defect that no longer exists. It was 1024 + 1024, and its whole
// rationale was that the default budget could not collect this guest AT ALL: a
// termination attempt charged the root re-scan against the same allowance, so a
// guest whose globals crossed 16 KiB re-scanned, ran out, failed to terminate,
// and repeated until `markDeadline` forced an unbudgeted finish hundreds of
// ticks later. That was a silent cliff and this guest went over it (CLAUDE.md,
// "The root scan that could not fit in a step"). **The cliff is gone**:
// `EffectiveBudget()` floors the budget at `rootScanCost() + 64` and the
// collector logs the one `fkgc:` line that names the cause. So the constant is
// no longer holding a cliff open -- what it is doing now is buying enough real
// work per step to keep the DIRTY RATE from outrunning the mark, which is an
// ordinary tuning decision with an ordinary measurement behind it.
//
// AND THE RESERVE IS WHY THE NUMBER HAD TO GO UP. Before the round, the scan was
// charged after the queues had spent; now it is held back before they start. At
// the old 2048 that took this guest's real work per step from ~2048 to ~930 --
// which is the whole of the 12-steps-and-3-deadlines row above, and it is a
// change in what a budget MEANS rather than a regression in anything.
//
// WHAT IT COSTS is the pause, in a straight line: doubling the budget halves the
// number of steps and doubles each one, so the total work is unchanged by
// construction. The transient the mode decision is priced on is re-measured in
// CLAUDE.md rather than asserted here.
const (
	// gcRootGranules is what upstream RESERVES out of every step for this
	// guest's globals re-scan, so it is a property of the guest rather than a
	// choice. Today's `roots=` is ~4,470 words = ~1,118 granules; 1024 is the
	// round number just under it, which is why the shipped `eff` equals the
	// shipped `budget` with a little of the real-work half spent on the scan.
	// logHeap prints `roots=` so the two can be compared without a rebuild.
	gcRootGranules = 1024
	// gcRealGranules is the DECISION: how much marking one step does, hence the
	// pause. 4096 is the first value that holds `deadlines=0` on the `mar`
	// suite, and it is upstream's own documented remedy for this symptom.
	gcRealGranules = 4096
	gcBudget       = gcRealGranules + gcRootGranules
)

func init() {
	// Constant-false under `-gc=leaking`, so this initialiser has no body in
	// the shipped-leaking build and no `init` is emitted for it.
	if !fkgc.Enabled() {
		return
	}
	fkgc.SetThreshold(gcArmBytes)
	fkgc.SetBudget(gcBudget)
}

// gcArmIfNeeded asks for the deferred flush when the collector is due one, and
// is the whole of the pacer fix. It is called at the end of `fk_on_event`, which
// is the guest's only always-run point that costs nothing when nothing happens.
//
// IT DOES NOT COLLECT AND MUST NOT. An event handler is not guaranteed to be an
// outermost dispatch -- a mod raising `script_raised_built` from inside its own
// handler reaches this one nested -- and a mark step at a nested dispatch is the
// safe-point violation agents/gc.md's whole marking argument rests on not
// happening. `fk.Defer()` registers a one-shot `on_tick`; `on_tick` is raised by
// the engine's own loop and never from inside another event, which is where the
// runtime asserts the depth is zero.
//
// The phase test is not an optimisation. While a collection is in flight the
// runtime is already driving `fk_gc_step` from its own one-shot registration,
// `CollectIfNeeded` would decline, and arming would buy a flush tick per event
// for the whole length of the collection.
func gcArmIfNeeded() {
	if !fkgc.Enabled() {
		return
	}
	s := fkgc.Stats()
	if s.Phase != 0 || s.SinceGC < gcArmBytes {
		return
	}
	requestFlush()
}

// logHeap writes the one line that lets the two build variants be compared at
// all, and it is the same line in both.
//
// `runtime.ReadMemStats` is the cross-arm part: under `-gc=leaking` it is
// TinyGo's own bump allocator reporting `Sys = heapEnd - heapStart` and
// `HeapAlloc` = every byte it ever handed out (which, leaking, is the same
// thing); under `-gc=custom` it is `fkgc`'s hook reporting the region the
// collector owns plus its static metadata. Either way `sys=` is the number that
// decides Factorio's worst tick -- 0.2 ms per MiB of linear memory, occupied or
// not (`../FkLua/agents/guests.md`, "the guest heap budget").
//
// Everything after `gc=1` exists only under the collector and is what stage C's
// gates are read from: `grows=` is the doubling counter agents/gc.md's
// acceptance test watches, and `deadlines=` is the livelock escape whose
// expected value is zero forever.
func logHeap(what string) {
	if !verboseLog {
		return
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	logStart("heap ")
	logS(what)
	logS(" sys=")
	logU(uint32(ms.Sys))
	logS(" alloc=")
	logU(uint32(ms.HeapAlloc))
	if !fkgc.Enabled() {
		logS(" gc=0")
		logEnd()
		return
	}
	s := fkgc.Stats()
	logS(" gc=1 heap=")
	logU(s.HeapBytes)
	logS(" live=")
	logU(s.LiveBytes)
	logS(" free=")
	logU(s.FreeBytes)
	logS(" since=")
	logU(s.SinceGC)
	logS(" cycles=")
	logU(s.Collections)
	logS(" grows=")
	logU(s.Grows)
	logS(" steps=")
	logU(s.Steps)
	logS(" deadlines=")
	logU(s.Deadlines)
	logS(" phase=")
	logU(s.Phase)
	logS(" meta=")
	logU(fkgc.MetaBytes())
	// THE FIRST THING UPSTREAM ASKS FOR WHEN `deadlines=` IS NOT ZERO, and this
	// guest used to make somebody derive it. `SetBudget`'s header: "if Deadlines
	// rises, compare EffectiveBudget() against Budget() FIRST. Equal means [the
	// dirty rate] and this is the knob. Larger means [the root set], the
	// collector has already applied the floor". Both numbers are one call each
	// and neither can be reconstructed from the rest of the line.
	logS(" budget=")
	logU(fkgc.Budget())
	logS(" eff=")
	logU(fkgc.EffectiveBudget())
	logS(" roots=")
	logU(fkgc.RootWords())
	logEnd()
}
