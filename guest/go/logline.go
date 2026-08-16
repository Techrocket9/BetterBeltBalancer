package main

// THE LINE BUILDER, AND WHY IT IS THE WHOLE HEAP DIET.
//
// Every `[BBB] ` line this guest writes is assembled here, in one package-level
// byte buffer, and handed to fk.Log as a string that BORROWS that buffer. It
// allocates nothing after the first line of each length.
//
// What it replaces was ordinary Go: `"[BBB] state clusters=" + u32(n) + ...`,
// with `u32` being `strconv.FormatUint`. Under `-gc=leaking` -- which is
// mandatory (`../FkLua/agents/guests.md`) -- that is not a transient cost. Every
// intermediate string of every concatenation is permanent, in the guest's linear
// memory, in every save and every multiplayer join. And `+` in a loop is
// quadratic: `logState` appends up to 64 cluster sizes one at a time, so one
// call left ~9 KB of dead strings behind. It is called once per part placed.
//
// Measured, n=200 k=4 express, the bench save's `--create` (200 balancers,
// 3,200 parts), reading `fk_mod.lua`'s own doubling notice:
//
//	verbose, concatenated  16 -> 32 -> 64 MiB      idle worst tick 19.9 ms
//	QUIET=1 (no lines)     no doubling (<16 MiB)   idle worst tick  2.3 ms
//	verbose, THIS FILE     no doubling (<16 MiB)   idle worst tick  2.3 ms
//
// The entire guest heap was log lines. Everything else this guest does per
// compile -- the plan, the edge list, the create_entity tables -- was already
// on reusable buffers and together did not reach one doubling, which is why the
// verbose build and the quiet build now measure the same.
//
// THE VERBOSE BUILD IS THE SHIPPED BUILD and that is why this had to be fixed
// rather than switched off: the guest's own log lines are the assertion surface
// for all five headless suites (CLAUDE.md, "Verification"). `QUIET=1` still
// exists and still eliminates every line below the error level, but it is no
// longer the difference between a 16 MiB heap and a 64 MiB one.
//
// THE BUFFER IS SAFE TO BORROW. `fk.Log` passes (pointer, length) to the host,
// which copies the bytes into a Lua string before it returns; the Go string
// never outlives the call. Nothing between logStart and logEnd makes a host
// call, so a synchronously-raised nested event cannot interleave with a line
// that is half built -- that is an invariant of the call sites, and the reason
// every one of them builds its line in one uninterrupted run.
//
// USAGE. A verbose line is guarded by the `verboseLog` CONSTANT rather than by a
// function that tests it, so a quiet build removes the whole block at compile
// time:
//
//	if verboseLog {
//	    logStart("compiled cluster ")
//	    logU(root)
//	    logEnd()
//	}
//
// Errors and alerts are never switched off and take no guard.

import (
	"unsafe"

	"github.com/Techrocket9/fklua/guest/go/fk"
)

// lineBuf is the one buffer, and it is a FIXED ARRAY rather than a slice.
//
// A line that somehow exceeded it is TRUNCATED rather than grown, and that is
// the point: `copy` is one memcpy with no reallocation path behind it, where
// `append` carries a growth branch that LLVM inlines into every one of the ~200
// call sites. Measured, same source otherwise: wasm `code` 98,373 B with
// `append` against 81,457 with `copy`, which is 1,339,988 B of generated Lua
// against 1,035,941. Both unbounded lines cap themselves anyway (`state` at 64
// cluster sizes, `skin` at 32 variations), so the worst line this guest can
// write is 427 bytes and the truncation is a backstop, not a policy.
//
// WHAT WAS TRIED AND REJECTED: `//go:noinline` on the six writers below. It is
// worth 17 KB of wasm `code` and 160 KB of generated Lua -- the mod would ship
// SMALLER than before this file existed -- and it costs **2.1 ms on every 4x4
// recompile**, which is a per-edit cost a player feels. Measured interleaved,
// four reps each against the same base, `make test SUITES=m2`, medians minus
// each run's own `idle tick pair` control: 5.50 ms without, 7.59 ms with. The
// profiled window writes two log lines, so this is not the log path getting
// slower -- it is what not inlining these does to the code around the ~350 host
// calls a recompile makes. The 82 KB of Lua is the cheaper side of that trade
// and it is paid once per load, against 2.1 ms paid on every belt laid at a
// balancer's edge.
var lineBuf [512]byte
var lineLen int

// lineDigits is package level for the same reason: a local array whose address
// is taken is not reliably stack-promoted under TinyGo (`../FkLua/agents/
// abi.md`), and a byte that is not reliably on the stack is a byte in every
// save.
var lineDigits [10]byte

// logStart opens a line. The caller has already decided it wants one.
func logStart(s string) {
	lineLen = 0
	logS("[BBB] ")
	logS(s)
}

// logErrStart opens an ERROR line. Never switched off: a create_entity that came
// back nil is a player-visible bug and has to leave a trace in the log of the
// session it happened in. `test/run.sh` fails a run on any of these.
func logErrStart(s string) {
	lineLen = 0
	logS("[BBB] error: ")
	logS(s)
}

// logAlertStart opens an ALERT line: something outside this mod did something
// drastic and the mod coped. Deliberately not `error:`, which the test harness
// treats as a failed run and which means only one thing -- a compile did not
// produce a network.
func logAlertStart(s string) {
	lineLen = 0
	logS("[BBB] alert: ")
	logS(s)
}

func logS(s string) { lineLen += copy(lineBuf[lineLen:], s) }

func logU(v uint32) {
	i := len(lineDigits)
	for {
		i--
		lineDigits[i] = byte('0' + v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	lineLen += copy(lineBuf[lineLen:], lineDigits[i:])
}

// logI writes a signed tile coordinate.
func logI(v int32) {
	if v < 0 {
		logS("-")
		logU(uint32(-v))
		return
	}
	logU(uint32(v))
}

// logF1 writes a coordinate to one decimal place.
//
// Positions in this guest are always tile centres or tile boundaries, so one
// decimal is exact and `strconv.FormatFloat` -- which links a large chunk of
// formatting code into a guest that has no other use for it -- is not worth
// having. Same reasoning that kept the old `f2s` hand-written; this one writes
// into the line instead of building three strings to throw away.
func logF1(v float64) {
	if v < 0 {
		logS("-")
		v = -v
	}
	whole := uint32(v)
	logU(whole)
	logS(".")
	logU(uint32((v-float64(whole))*10 + 0.5))
}

// logEnd hands the buffer to the host as a string that borrows it.
//
// `unsafe.String` rather than `string(lineBuf[:lineLen])`, which would COPY --
// and a copy here is exactly the permanent allocation this file exists to
// remove. `lineBuf` is a package-level array, so its address is a static and
// never nil.
func logEnd() {
	fk.Log(unsafe.String(&lineBuf[0], lineLen))
}

// logError and logAlert are the constant-message forms of the two levels that
// are never switched off.
func logError(s string) {
	logErrStart(s)
	logEnd()
}

func logAlert(s string) {
	logAlertStart(s)
	logEnd()
}
