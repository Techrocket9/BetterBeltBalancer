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
	// The shared DATA-STAGE library. This mod declares its two startup
	// dropdowns through it (guest/go/tune/plan.go) and its prototypes by hand;
	// tune/plan.go's header is why the split is where it is.
	github.com/Techrocket9/fkrecipes/go v0.1.0
)

// FkLua is a sibling checkout, not a published module. The mod and the compiler
// are developed together and the guest substrate has to match the runtime the
// compiler ships. The VERSION above is not a choice made here: FkRecipes
// requires guest/go v0.2.0 through the real channel, and minimal version
// selection takes the maximum of the two requirements -- so the number moved
// from v0.0.0 when this mod took that dependency. The replace still resolves it
// onto the sibling checkout, which is what is actually compiled.
replace github.com/Techrocket9/fklua/guest/go => ../../../FkLua/guest/go

// DEV-ONLY, and it is the only way to build this today: FkRecipes is a sibling
// checkout with no remote and no tag. v0.1.0 above is the tag its README names
// as its eventual Go one, written down so the require has something to say;
// nothing resolves it, because this line points the whole module path at the
// working tree next door. When FkRecipes is published this replace comes out
// and the version becomes real.
replace github.com/Techrocket9/fkrecipes/go => ../../../FkRecipes/go
