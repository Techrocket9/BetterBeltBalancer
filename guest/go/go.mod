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

require github.com/Techrocket9/fklua/guest/go v0.0.0

// FkLua is a sibling checkout, not a published module. The mod and the compiler
// are developed together and the guest substrate has to match the runtime the
// compiler ships, so a version would be a lie either way.
replace github.com/Techrocket9/fklua/guest/go => ../../../FkLua/guest/go
