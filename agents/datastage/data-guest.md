# The shipped mod holds no Lua

**Every byte of Lua in the packaged mod is generated, and `mod-data/` is graphics, locale, a changelog and a thumbnail.** The settings and data stages were ten hand-written files there until 2026-08-25 and are `guest/go/data` now — a SECOND wasm module beside the control guest, from which `fklua mod` writes `settings.lua`, `data.lua` and `data-final-fixes.lua`, one per hook the module exports.

**The old reason for the exception was wrong, and it is worth saying which half.** It read: *Factorio's data stage is declarative with no runtime and no state, so there is nothing there for a Go guest to be the brain of.* The first clause is true of the STAGE and false of this mod's use of it. That "declarative" stage branched on the engine version to decide whether two belt-connectables may share a tile, defined a runtime setting only on the versions that have one, deep-copied four base belt prototypes and patched their speed and their sprites, computed eight arrow sprites from an offset constant, derived a technology's cost from whichever tier base says it hangs off, and defined a stub entity only when no incumbent had claimed the name. None of it could be reached by `go test`, and the version branch had to be a file two Lua states each `require`d because they share nothing else.

**Two modules and not more exports on one**, which is FkLua's own D1 and is measured rather than tidy: `require` re-executes at every stage, so a control guest hooked into the data family is parsed once per stage — **+150 ms per game load** for a 3.1 MB program the stage never calls — and its package initialisers would run against a runtime API that is not there. A data module is compiled `-gc=leaking` and packaged `--persist=none` whatever the control guest's arms are, because it runs once and dies with the Lua state that built it. `fklua mod` REFUSES a data module that imports `fkapi` at all.

**One Go module, though**, and that is what paid for the pass: `guest/go/data` and the control guest share `guest/go/engine` (the version branch, which both stage hooks now call) and `guest/go/skin` (the sprite sheet's cell count, which the prototype and the runtime had each written down separately). The shared-file problem is retired rather than re-solved.

**What must not be tidied:** the `//go:noinline` marks in `guest/go/data`. Lua 5.2 keeps a jump offset in an 18-bit field, this stage is exactly the straight-line-through-small-helpers shape LLVM inlines hardest, and at `-opt=2` all six sections folded into one wasm function that lowered to a 28,139-line Lua function the parser refused (`control structure too long near 'trap_unreachable'`, measured on the first packaged build). Marked, the largest is 3,414 lines and the emitted module is 27% smaller as well. `main.go`'s header is the long form.

**The gate is `test/check-datastage.py`**, and it is a different question from `make test` — see Verification.

**`fklua mod` still `rm -rf`s its output directory** before writing, so nothing hand-written can survive there; the tree named by `[mod] data` is merged in before the directory writer AND the zip writer, so both carry the same bytes.

## What it cost, measured either side of the port

Both packages built from clean on 2026-08-25, shipped config (`--persist=packed --gc=collected`), same FkLua, same pin, same machine:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.3.0.zip` | 453,975 B | **546,004 B** | +20.3% |
| `fk_module.lua` (the control guest) | 3,136,956 B | **3,136,956 B** | **byte-identical** |
| `fk_data_module.lua` (the data guest) | — | **1,498,245 B** | new |
| `fk_data.lua` (the shim, verbatim) | — | 24,106 B | new |
| `dist/bbbdata.wasm` | — | 141,799 B | new |
| hand-written Lua in the package | **45,037 B** | **0** | eleven files |
| members bound into the mod | 54 | **54** | of 4,859, none added |

**The control guest is byte-identical, and that is the shape of the whole change**: nothing in `guest/go` outside the new `data/` and `engine/` packages was touched, so `fk_module.lua` comes out the same file. What the zip pays is a second compiled guest for a stage that used to be 45 KB of source — 92 KB of download, once, against a data stage `go test` can reach and that cannot drift from the runtime consuming it.

**Nothing on any hot path moved, measured rather than argued.** A data module runs at load and is not in the game at all afterwards, so the standing gate is that the CONTROL guest's per-operation heap slopes do not move — and the `mar` suite's leaking arm came back **identical to the byte** against a build of the pre-port commit run in the same session: **1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B** per primitive over **3.92 MiB** of linear memory, 1,136 B of calibration at 0.0% spread, 0 items lost over 200 teardowns, 681 audits at drift=0. That is the set this file has recorded since the single-edge port, and it is what says a second guest costs the first one nothing.

**All fourteen suites are green in BOTH arms** over the ported package (1m49s collected, 1m59s leaking, one invocation each), and **no suite's numbers moved at all** — which, for a change that touches only the load-time half, is the only result available, and is the second validator behind the dump.
