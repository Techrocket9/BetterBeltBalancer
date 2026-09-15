// The guest is its own Go module, for the reason FkLua's own guest module is:
// //go:wasmimport is rejected outside GOARCH=wasm, so these files can never be
// built by a host toolchain and must not be inside a module that is.
//
// `fk` comes from FkLua itself (the host boundary: logging, build flags).
// `fkapi` is GENERATED INTO THIS REPO by `fklua gen-bindings` and committed --
// `fklua lock` hashes it at exactly this path, which is why the module lives at
// guest/go rather than guest/.
module github.com/Techrocket9/BetterBeltBalancer/guest/go

go 1.24

require (
	github.com/Techrocket9/fklua/guest/go v0.2.0
	// The shared DATA-STAGE library, through the REAL channel since 0.3.3:
	// its go/v0.1.0 tag on GitHub, no replace. This mod declares its settings,
	// its recipe and its technology through it (guest/go/tune/plan.go). The
	// dev-only replace onto the sibling checkout that stood here until the
	// library was published came out on 2026-09-14; a developer who wants the
	// sibling's working tree again adds it back locally and does not commit it.
	github.com/Techrocket9/fkrecipes/go v0.1.1
)

// FkLua is a sibling checkout, not a published module. The mod and the compiler
// are developed together and the guest substrate has to match the runtime the
// compiler ships. The VERSION above is not a choice made here: FkRecipes
// requires guest/go v0.2.0 through the real channel, and minimal version
// selection takes the maximum of the two requirements -- so the number moved
// from v0.0.0 when this mod took that dependency. The replace still resolves it
// onto the sibling checkout, which is what is actually compiled.
replace github.com/Techrocket9/fklua/guest/go => ../../../FkLua/guest/go
