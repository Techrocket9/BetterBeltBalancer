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

// End hands the buffer to the host as a string that borrows it.
//
// `unsafe.String` rather than a conversion, which would copy. `buf` is inside a
// package-level struct in every real use, so its address is a static.
func (l *Line) End() {
	fk.Log(unsafe.String(&l.buf[0], l.n))
}
