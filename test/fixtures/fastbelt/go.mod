// THE FAST-BELT FIXTURE: a whole Factorio mod, written in Go, whose only job is
// to define a belt faster than this mod's hidden network so that the speed
// derivation has something to derive.
//
// ITS OWN MODULE, separate from guest/go, because it is TEST MATERIAL and the
// guest module is what ships. Nothing here is ever packaged into
// better-belt-balancer; test/check-datastage.py builds it into a temporary
// directory, stages it beside the mod under test for one --dump-data run, and
// throws it away.
//
// It needs no `go.sum`: FkLua's guest module has no dependencies of its own, so
// the replace below is the whole graph.
module github.com/Techrocket9/BetterBeltBalancer/test/fixtures/fastbelt

go 1.24

require github.com/Techrocket9/fklua/guest/go v0.0.0

// The same sibling-checkout replace guest/go uses, one directory deeper. FkLua
// is not a published module and the mod and the compiler are developed
// together, so a version would be a lie either way.
replace github.com/Techrocket9/fklua/guest/go => ../../../../FkLua/guest/go
