# Pure Go — one language, and what the Rust day measured

**Every guest in this repository is Go: the shipped control guest, the data guest, the six pure packages and all fourteen test observers.** `fklua.toml` reads `lang = ["go"]`, `make observers` from clean invokes no cargo, and there is no second toolchain in the build path. Decided by the user on **2026-08-26**: *BBB should be pure Go — let other mods handle Rust coverage for FkLua.*

**It was mixed for one day and the day is worth keeping.** Phase 8 of the estate port ([`agents/estate-port.md`](../estate-port.md), 2026-08-25) wrote the `mig21` observer a second time in RUST and staged it, to hold FkLua's Go/Rust member-id parity against a **real engine** rather than only against the mirror tests' host stub. **The measurement succeeded and it is banked**: 51 tagged log lines over two committed fixtures, byte-identical with **no mask at all**, and a packaged `fk_api_gen.lua` byte-identical across the two backends — two pruning passes over two entirely different wasm modules selecting the same twenty members and assigning them the same dense ids. Phase 8's own section records all of it and is unchanged by the reversion.

**What the reversion turns off is a CONTINUOUS exercise, not the claim.** A parity claim held on every `make test` forever is what phase 8 argued for and it is the half that moved: continuous Rust coverage for FkLua belongs to the **fklua-ports** repositories, which are the mods written for that purpose, and carrying a second toolchain here to say the same thing a second time is a cost this mod's build does not have to pay. What guards `mig21` from here is the suite: `test/assert-mig21.py` is unchanged and reads these lines.

**The reversion is transcript-exact and that is the gate it had to clear.** The Rust observer's transcript was captured as the golden BEFORE anything was touched, twice, self-diff identical; the Go observer was restored **verbatim** out of git history at `74eab29` and needed no edit at all to compile against today's bindings; and its transcript comes back **byte-identical to the Rust one on both fixture arms**, under phase 8's own normalisation and no mask. `fklua.lock`'s `bindings_sha256` came back to the exact pre-phase-8 value, `b3f41c12…`, which is the same statement from the packaging end. **The Rust implementation is recoverable in full from git history at `5f91937`.**

| | |
|---|---|
| golden diff, m2 fixture | **EMPTY — 26 lines byte-identical** |
| golden diff, edge fixture | **EMPTY — 25 lines byte-identical** |
| ...and the mod's own `[BBB]` lines beside them | identical, 237 and 243 |
| the packaged observer's `info.json` | **identical** to the Rust arm's, `dependencies` still `base` alone — the load-order property phase 2 red-proved |
| the packaged `fk_api_gen.lua` | **byte-identical** to the Rust arm's, `a51f0924…` |
| `fk_module.lua` | 577,418 B, phase 8's recorded Go figure to the byte (the Rust one was 493,791) |
| `fklua.lock` | back to `b3f41c12…`, the pre-phase-8 value |
| `make observers` from clean | **28.3 s, zero cargo/rustc/wasm32 hits** in the build log |

**What was deleted**: `guest/rust/` entire — the generated `fkapi` crate (130,775 lines), `obs/mig21`'s three source files, the workspace `Cargo.toml`/`Cargo.lock` and its `.gitignore`, ten tracked files. **What came back**: `guest/go/obs/mig21/main.go`, 541 lines, verbatim. The Makefile lost `RUST_DIR`/`RUST_SRC`/`RUST_TARGET`/`RUST_FLAGS`, the `obs-mig21-rs.wasm` rule with its RUSTFLAGS and `wasm-opt` lowering pass, and `clean`'s `guest/rust/target`; `$(OBS_MIG21_DIR)` is back on the standard Go pattern rule, **which carries `Makefile` as a prerequisite** — the 2026-08-25 build-graph hazard fix is preserved, not reverted with it.
