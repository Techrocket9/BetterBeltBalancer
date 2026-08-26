// Command fastbelt-inert is the fast-belt fixture's CONTROL guest, and it does
// nothing at all. It exists because `fklua mod` cannot package a mod that has
// only a data module.
//
// `fklua mod IN.wasm [--data-module DATA.wasm]` takes the control module as its
// one positional argument, and omitting it is `fklua mod: no input module`
// (measured against FkLua b185900, 2026-08-25). A data-stage-only mod is a real
// shape -- a mod that adds prototypes and no behaviour is most of the portal --
// so this is written down as FKLUA-GAPS.md item 25 and worked around here
// rather than forked around.
//
// The cost of the workaround is what the packager says about it: this compiles
// to about 113 KB of Lua that is `require`d and never called, plus the warning
// "This guest exports no event hook, so the mod will load and then never be
// called again", which is exactly true and exactly what is wanted. Nothing in
// this file reaches a running game, and nothing in this directory is ever
// packaged into better-belt-balancer.
//
// It exports no `fk_*` hook DELIBERATELY. Exporting one would give the fixture a
// control stage that runs during a --dump-data run's... nothing, because
// --dump-data stops before control.lua; but it would also mean a future test
// that loaded this fixture into a real save got behaviour from it. An inert mod
// should be inert on every path, not on the one path the gate uses.
package main

func main() {}
