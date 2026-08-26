package harness

// THE OBSERVER LINE BUILDER, and it is the shipped guest's `logline.go` with
// one thing added and one thing taken away.
//
// TAKEN AWAY: the heap argument. That file exists because the shipped guest's
// log lines were, measured, its ENTIRE permanent heap -- 64 MiB of dead
// intermediate strings from `+` in a loop, called once per part placed, in
// every save and every multiplayer join. An observer logs a few dozen lines in
// a run that lasts seconds and is thrown away, so none of that applies here.
//
// ADDED: the reason it is a builder at all, which survives the heap argument
// intact. THE LINE TEXT IS THE ASSERTION SURFACE. `test/assert-*.py` keys every
// regex on what follows the `[BBB-...]` tag, so a space, a separator or a
// digit's format IS the contract with the assertion script -- and a builder
// with one `S`/`U`/`I` per field is the shape in which a transcription from Lua
// can be read against the Lua it came from, field for field. `fmt` would hide
// exactly that behind a format string, and would link TinyGo's reflection into
// a guest that has no other use for it.
//
// USAGE. One `Line` per observer, package level, with the tag baked in:
//
//	var out = harness.Line{Tag: "[BBB-SEDGE] "}
//	out.Open("sample tag=").S(tag).S(" tick=").U(tick).End()
//
// THE BUFFER IS SAFE TO BORROW for the same reason the shipped guest's is:
// `fk.Log` hands (pointer, length) to the host, which copies the bytes into a
// Lua string before it returns. Nothing between `Open` and `End` makes a host
// call, so nothing can interleave with a half-built line.

import (
	"unsafe"

	"github.com/Techrocket9/fklua/guest/go/fk"
)

// Line is one observer's log line under construction.
//
// A FIXED ARRAY, so an over-long line is TRUNCATED rather than grown: `copy` is
// a memcpy with no reallocation path behind it. 1 KiB rather than the shipped
// guest's 512 B because an observer's widest line is a per-rig chest sample --
// `sedge`'s is eight rigs of up to five chests each and grows with the estate --
// where the guest's widest is a bounded cluster list.
type Line struct {
	// Tag is written at the head of every line this builder opens, including
	// the trailing space: "[BBB-SEDGE] ". It is what the assertion scripts
	// select on and it is per observer, never per line.
	Tag string

	buf    [1024]byte
	n      int
	digits [20]byte
}

// Open starts a line: the tag, then s. Anything half-built is discarded.
func (l *Line) Open(s string) *Line {
	l.n = 0
	return l.S(l.Tag).S(s)
}

// S appends a string.
func (l *Line) S(s string) *Line {
	l.n += copy(l.buf[l.n:], s)
	return l
}

// U appends an unsigned decimal.
func (l *Line) U(v uint64) *Line {
	i := len(l.digits)
	for {
		i--
		l.digits[i] = byte('0' + v%10)
		v /= 10
		if v == 0 {
			break
		}
	}
	l.n += copy(l.buf[l.n:], l.digits[i:])
	return l
}

// B appends "true" or "false", which is what Lua's `tostring` on a boolean
// writes and therefore what the assertion scripts read.
func (l *Line) B(v bool) *Line {
	if v {
		return l.S("true")
	}
	return l.S("false")
}

// I appends a signed decimal. Tile coordinates go negative.
func (l *Line) I(v int64) *Line {
	if v < 0 {
		return l.S("-").U(uint64(-v))
	}
	return l.U(uint64(v))
}

// F1 appends a number to ONE decimal place, which is Lua's `%.1f` and therefore
// what the assertion scripts read.
//
// IT EXISTS FOR EXACTLY ONE THING: the `mig` suite's fidelity rig reports a
// part's health and max_health that way (`value=85.0 max=170.0`), and those two
// numbers are the whole of what says the conversion carried a wound across a mod
// swap. Nothing else in the estate logs a float.
//
// Rounding is half-away-from-zero where C's printf is half-to-even, and the
// difference is unreachable here: a health is set to an integer and a
// max_health comes off a prototype as one, so every value this ever sees is
// exact at one decimal. Anything that starts logging a real measurement should
// re-read this line first.
func (l *Line) F1(v float64) *Line {
	neg := v < 0
	if neg {
		v = -v
		l.S("-")
	}
	// Scale, round, and split rather than divide back: the integer part of a
	// large float is not recoverable through float arithmetic without drift.
	scaled := uint64(v*10 + 0.5)
	return l.U(scaled / 10).S(".").U(scaled % 10)
}

// F4 appends a number to FOUR decimal places, which is Lua's `%.4f`.
//
// IT ALSO EXISTS FOR EXACTLY ONE THING: the bench harness's per-shape line
// (`balance=%.4f`), which is how a megabase cell reports whether each class of
// balancer in a heterogeneous save is still splitting evenly.
//
// THE PADDING IS THE WHOLE DIFFERENCE FROM F1. One decimal place cannot need a
// leading zero and four can: `%.4f` of 1.0 is `1.0000`, so the fractional part
// is written with its zeros rather than as the number 0. F1's own rounding note
// applies here unchanged, and it is reachable here in a way it is not there --
// a balance is a real measurement -- so a value exactly halfway may round up
// where C's printf would round to even. A balance is a diagnostic read against
// bounds of 1.02 and 1.25; nothing anywhere compares its last digit.
func (l *Line) F4(v float64) *Line {
	if v < 0 {
		v = -v
		l.S("-")
	}
	scaled := uint64(v*10000 + 0.5)
	frac := scaled % 10000
	l.U(scaled / 10000).S(".")
	for div := uint64(1000); div > 1; div /= 10 {
		if frac < div {
			l.S("0")
		}
	}
	return l.U(frac)
}

// End hands the buffer to the host as a string that borrows it.
//
// `unsafe.String` rather than a conversion, which would copy. `buf` is inside a
// package-level struct in every real use, so its address is a static.
func (l *Line) End() {
	fk.Log(unsafe.String(&l.buf[0], l.n))
}
