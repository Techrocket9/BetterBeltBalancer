//go:build !bbbquiet

package main

// The logging switch. `make guest QUIET=1` builds with -tags bbbquiet and every
// [BBB] line below the error level disappears -- not "becomes cheap", but is
// eliminated by the compiler, because verboseLog is a constant and every call
// site is guarded by it.
//
// The default is ON, and `make test` depends on it: the guest's own log lines
// are the assertion surface for both the M1 cluster tests and the M2 network
// tests. A test that computed the expected answer in Lua would be a second
// implementation of the thing under test.
//
// Errors are NEVER switched off. A create_entity that came back nil is a
// player-visible bug and has to leave a trace in the log of the session it
// happened in.
// A line with variable parts is written through the builder in logline.go,
// inside an `if verboseLog` block so that a quiet build removes it outright;
// logInfo is the shorthand for the ones whose whole message is a constant.
const verboseLog = true

func logInfo(s string) { logStart(s); logEnd() }
