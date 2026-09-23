# CLAUDE.md — working context for BetterBeltBalancer

A Factorio mod: automated belt balancers that click together into arbitrary shapes and balance items across the belts feeding and draining them (belt orientation decides input vs output). Two goals, in order:

1. **Demonstrate FkLua is fit for a serious mod** — the mod's brain is written in Go (TinyGo → wasm → Lua via [`../FkLua`](../FkLua)), not hand-written Lua.
2. **Significantly outperform belt-balancer-2**, the incumbent, which manipulates transport lines from Lua every tick.

Conventions are inherited from FkLua — read [`../FkLua/CLAUDE.md`](../FkLua/CLAUDE.md) before working here. The rules below repeat only what is load-bearing daily.

**This file is the index; the working notes live under [`agents/`](agents/).** Until 2026-09-22 everything below the rules was one 1.15 MB file. It was split by section, verbatim, into the tree the index at the end describes. Two consequences worth knowing:

- **A pointer of the form `CLAUDE.md, "<section>"`** (in a code comment, an `agents/` record, or a sibling repo such as `../FkLua/agents/gc.md`) names a heading that now lives under `agents/`. The index rows below say which file each former section went to; for a subsection, `grep -rn '^#.*<section>' agents/` finds it. A comment that says "CLAUDE.md records" without naming a section means these notes as a whole.
- **Moved text still says "this file", "above" and "below"** in places, meaning the old single file. The section names it quotes are the way through.

---

## Where things stand

- **Trunk targets Factorio 2.1 and its rule is one belt per balancer part** (since 2026-08-24): a part carries at most one edge, and a cluster that asks for more is refused. Design [`agents/single-edge.md`](agents/single-edge.md); rule `guest/go/sedge.go`; working note [`agents/features/single-edge-rule.md`](agents/features/single-edge-rule.md).
- **Two release arms out of one tree.** `master` ships 0.3.x for Factorio 2.1; `release/2.0` ships 0.2.x for Factorio 2.0 and is recut from trunk with a four-file stamp (`fklua.toml`, `fklua.lock`, `guest/go/fkapi/fkapi.go`, `mod-data/changelog.txt`). `test/check-release-arm.sh` holds the branch to lines trunk has. The newest tags are `0.3.6` on `master` and `0.2.6` on `release/2.0` (`git tag`, 2026-09-22).
- **Seventeen headless suites.** Fourteen answer the same on either engine; `mig21` and `mig` invert between 2.0 and 2.1, and `flip` exists on 2.0 only and prints a SKIP on 2.1.
- **No hand-written Lua anywhere in the repository**: the control guest, the data guest and every test observer are Go.
- **What is owed** is the outstanding list in [`agents/history/status.md`](agents/history/status.md) and the tail of [`agents/history/verification-log.md`](agents/history/verification-log.md), whose last entry is the newest record of what ran where.

## Critical rules

- **Rebase, never merge.** Trunk is `master`, history is linear, `git merge --ff-only` is the guard.
- **Determinism is a correctness property.** Factorio is lockstep multiplayer: no entropy, no wall clock, no iteration-order dependence in anything host-visible. Lua `pairs` order over tables keyed by anything but dense integers is a desync.
- **Documentation drift is a gate failure, not a follow-up.** Update the note that describes a change in the same commit as the change: this file for what it states itself, and otherwise the file under `agents/` that owns the topic (the index says which). A dated verification record is appended to [`agents/history/verification-log.md`](agents/history/verification-log.md). A new topic gets a file in the directory it belongs to and a row in the index here; this file stays an index, and a section that grows past a screen moves out. Create `agents/` files on demand only.
- **Human-facing documents follow the house style in [`agents/docs-style.md`](agents/docs-style.md).** `README.md`, `FKLUA-GAPS.md`, the bench and test READMEs are written for a public GitHub reader: no local paths, no change-history narrative, no milestone or round codes as if known, no CLAUDE.md section citations, no agent or process attribution, and no em-dashes or en-dashes anywhere. This file and `agents/` are working notes and are exempt. Read that file before editing any of them, and run its grep check before committing.
- **Measure before believing.** Every performance claim in this repo carries the command that produced it. A variant that computes a different answer is not a faster variant — benchmarks compare outcomes (item counts delivered per output) before timings are trusted.
- **A sentence that reaches a data-stage PROTOTYPE's `localised_name` or `localised_description` must fit 200 BYTES PER ELEMENT.** The engine refuses a longer element (bytes, not characters; per element, not per string) and stops the whole load. SETTING prototypes are exempt. FkRecipes chunks what it composes into a prototype since `137f4aa`, and `make datastage-check`'s `note-recipe` and `note-control` arms measure every element over what the engine put in the dump. What is not measured is a chunk boundary on screen. Long form, including the 0.3.3 release block it caused: [`agents/datastage/localised-string-ceiling.md`](agents/datastage/localised-string-ceiling.md).
- **Never test against `/opt/homebrew/bin/lua`** (it is 5.5). Factorio is doubles-only Lua 5.2.1; use `../FkLua/bin/lua52f` for host-side Lua testing.

## Repository layout

The annotated form, with the reasoning behind each file, is [`agents/architecture/repository-layout.md`](agents/architecture/repository-layout.md).

```
fklua.toml, fklua.lock   the mod project: identity, dependencies, data module, shipped GC arm, api pin
guest/go/                the CONTROL guest (its own Go module; TinyGo -> wasm -> Lua)
  main.go                events, subscriptions, load hooks, fk_state_version
  cluster.go             the part registry: orthogonally adjacent parts per surface and force
  compile.go             the network compiler and edge classification (straight + curved exits)
  lifecycle.go           clones, surface deletion, rebuild-from-world, the audit, force merges
  carry.go               where a teardown's items go (recompile reinserts, removal spills)
  limit.go               the 64-port refusal, merge pre-pass, hand-back, build notes
  sedge.go               Factorio 2.1's one-belt-per-part rule and the multi-edge setting
  curve.go, curveupg.go  the curved-exit setting, and pre-curve saves keeping their reading
  priority.go            priority ports: the three doors onto setPartPriority
  fastreplace.go         a belt replacing a part; the refused-replace belt restore
  legacy.go              adopting Belt Balancer 2/3 saves
  commands.go, probe.go  /bbb-audit, the remote interface, the insert probe
  gc.go, logline.go      the --gc=collected seam; the zero-allocation log line builder
  log_*.go               the [BBB] logging switch (`make QUIET=1`)
  globalsetting.go       settings.global read and written, for both runtime-global bools
  stateversion*.go       the save-format rung table, and the -tags prestate fixture
  findpart.go, skin.go   quality-blind tile lookup; sprite variant writes
  fkapi/                 GENERATED bindings, committed, hashed by fklua.lock; never hand-edited
  plan/ skin/ carry/ edgemode/ engine/ tune/   the six PURE packages `make check` tests on the host
  data/                  the DATA guest: settings, data and data-final-fixes (a second wasm)
  obs/                   the test observers and their shared harness, all compiled guests
mod-data/                assets only: graphics, locale, changelog.txt, thumbnail (no Lua)
tools/make-graphics.py   the sprite-sheet contract and placeholder art
test/                    run.sh, assert-*.py, check-datastage.py, check-release-arm.sh, fixtures
bench/                   the benchmark harness (separate concern, do not disturb)
dist/                    build output, gitignored
agents/                  the working notes (index below)
```

## Build and gates

The annotated command list, the TinyGo flags and the `--persist=packed` decision are [`agents/architecture/build.md`](agents/architecture/build.md); what each gate asks is [`agents/verification/gates.md`](agents/verification/gates.md).

| command | what it does |
|---|---|
| `make guest` | TinyGo -> `dist/bbb.wasm` and `dist/bbbdata.wasm`. `GC=leaking` builds the other `-gc` arm; `QUIET=1` drops `[BBB]` lines |
| `make mod` / `make zip` / `make install` | package from `fklua.toml`; fails on a lost event/member prune (`dist/.fklua-mod.log`) |
| `make check` | the six pure packages' tests, `go vet` of the data guest, bindings and lock current, gofmt, `test/check-release-arm.sh` |
| `make test` | all seventeen suites against the installed Factorio. `make GC=leaking test` is the other arm, run as ONE invocation |
| `make datastage-check` | Factorio `--dump-data` against `test/datastage-goldens.json`: 24 arms, not part of `make test` |
| `make prestate`, `make observers`, `make graphics` | the pre-state fixture for `curv`, the observer packages, the sprite sheet |
| `make player-fixture SAVE=...` | re-cut `test/fixtures-player/` from a client save ([`agents/verification/player-fixture.md`](agents/verification/player-fixture.md)) |

## The suites

One note per suite under [`agents/verification/suites/`](agents/verification/suites/).

| suite | what it answers |
|---|---|
| [`m1`](agents/verification/suites/m1.md) | the cluster registry: counts and sizes across edits, and the sprite each part draws |
| [`m2`](agents/verification/suites/m2.md) | throughput and balance of every shape and edge type against a bare belt, conservation across a recompile, the curve band and its setting flip |
| [`m3`](agents/verification/suites/m3.md) | lifecycle kill-tests: clones, blueprints, forces, ghosts, robots, surface deletion, silent destroys, churn; and what M3 cannot verify |
| [`upg`](agents/verification/suites/upg.md) | a mod upgrade: rebuild-from-world adopts every network, then M2's assertions again |
| [`curv`](agents/verification/suites/curv.md) | a save from before the curved exit keeps its reading (three legs) |
| [`prio`](agents/verification/suites/prio.md) | priority ports: rates at every load, the toggle, the four refusals, the spill guard |
| [`plat`](agents/verification/suites/plat.md) | Space Age: a platform surface, belt stacking, stacked sushi |
| [`mar`](agents/verification/suites/mar.md) | the permanent-heap slope of every net-zero world operation |
| [`edge`](agents/verification/suites/edge.md) | edits that land on full, running networks: churn, merges, forces, the port limit, fast replace |
| [`mix`](agents/verification/suites/mix.md) | more than one item kind, and the carry pool's 32-group bound |
| [`mig`](agents/verification/suites/mig.md) | adopting a Belt Balancer 2/3 save; two phases under different mod sets |
| [`qual`](agents/verification/suites/qual.md) | every part at uncommon quality |
| [`sedge`](agents/verification/suites/sedge.md) | 2.1's one-belt-per-part rule and the four ways an edit can break it |
| [`mig21`](agents/verification/suites/mig21.md) | a committed 2.0 multi-edge save opened; no `--create` phase |
| [`flip`](agents/verification/suites/flip.md) | the 2.0-only multi-edge setting through all four transitions |
| [`iact`](agents/verification/suites/iact.md) | the interactive checklist's staged world; create only |
| [`curs`](agents/verification/suites/curs.md) | the gestures a player makes, from a committed client save; no `--create` phase |

## Environment

The record of which Factorio was installed when is [`agents/architecture/environment.md`](agents/architecture/environment.md).

- Factorio at `~/Library/Application Support/Steam/steamapps/common/Factorio/factorio.app/Contents/MacOS/factorio` (Steam, Space Age, mac-arm64); user dir `~/Library/Application Support/factorio/`. **Which version is installed is a Steam betas setting and it moves: ask `--version` rather than any sentence.** On 2026-09-22 it answered `2.0.77 (build 84539)`. `test/run.sh` stamps staged test mods for the running engine but GATES the packaged mod, so trunk's `make test` needs a 2.1 binary and the `release/2.0` arm a 2.0 one. `--benchmark` never saves.
- **FkLua is a sibling checkout** at `../FkLua` (a `replace` in `guest/go/go.mod`); the toolchain is `../FkLua/bin/fklua`. Its floor is `ddd9700` (the commit that added `fkdata.Raise`), which no manifest can enforce.
- **FkRecipes arrives through the real module channel**: `guest/go/go.mod` requires `github.com/Techrocket9/fkrecipes/go v0.1.1` with no `replace`.
- **TinyGo 0.41.1 + binaryen.** The host Go is 1.27.1 and TinyGo 0.41.1 refuses it, so every wasm build (including the speed arm inside `make datastage-check`) runs under `GOTOOLCHAIN=go1.26.6`.
- Where FkLua does not do what this mod needs, the gap is written down in [`FKLUA-GAPS.md`](FKLUA-GAPS.md) and worked around here. **Do not fork or patch FkLua for it** — that project wants the feedback, not a divergent copy.

## Index of `agents/`

Each row names the section of the old single file it was, where it was one. The long summaries the design notes and records carried in this index until 2026-09-22 are [`agents/history/record-summaries.md`](agents/history/record-summaries.md).

### Design notes and dated records (`agents/*.md`)

| file | covers |
|---|---|
| [`design.md`](agents/design.md) | **Read before implementing anything.** The compiled-network architecture, why the incumbent loses, the edge-interface problem, milestones, feature bar |
| [`spike-s1.md`](agents/spike-s1.md) | the empirical record behind the edge interfaces; every gotcha in it is load-bearing |
| [`docs-style.md`](agents/docs-style.md) | **Read before editing any human-facing document.** The public house style and its grep check |
| [`single-edge.md`](agents/single-edge.md) | the Factorio 2.1 port: one belt per part, the setting, the migration, and every phase's status |
| [`priority.md`](agents/priority.md) | priority ports: the construction, the flow model, the fit, the guest half, input priority |
| [`maxports.md`](agents/maxports.md) | the 64-port cap: where it comes from and what an uncap must clear |
| [`estate-port.md`](agents/estate-port.md) | the test estate's move from Lua to Go, nine phases, and the Rust arm and its reversion |
| [`fkrecipes-migration.md`](agents/fkrecipes-migration.md) | every FkRecipes round, long form, with the graded friction lists |
| [`migration-assessment.md`](agents/migration-assessment.md), [`-2`](agents/migration-assessment-2.md), [`-3`](agents/migration-assessment-3.md) | the three adversarial migration assessments (dated records; their corrections live in `fkrecipes-migration.md`) |

### Architecture (`agents/architecture/`)

| file | covers | was |
|---|---|---|
| [`repository-layout.md`](agents/architecture/repository-layout.md) | every directory and file, annotated | "Repository layout" |
| [`runtime-model.md`](agents/architecture/runtime-model.md) | M1 to M4 and their invariants: the failure envelope, rebuild on a fresh heap, batching, event filters and masks, `defines`, the fingerprint | the milestone paragraphs of "Status" |
| [`build.md`](agents/architecture/build.md) | the make targets, TinyGo flags, what ships, the `--persist=packed` decision | "Build" |
| [`environment.md`](agents/architecture/environment.md) | machine setup as recorded, with which Factorio was installed when | "Environment" |
| [`pure-go.md`](agents/architecture/pure-go.md) | one language, and what the one-day Rust arm measured | "Pure Go" |
| [`standardization-pass.md`](agents/architecture/standardization-pass.md) | every deviation from `fklua init`'s scaffold, kept or standardized, with why | "The standardization pass" |

### The data stage (`agents/datastage/`)

| file | covers | was |
|---|---|---|
| [`data-guest.md`](agents/datastage/data-guest.md) | the settings and data stages as a second compiled guest | "The shipped mod holds no Lua" |
| [`cost-research-speed.md`](agents/datastage/cost-research-speed.md) | the recipe and research cost settings and the derived belt speed | "Cost, research and belt speed" |
| [`fkrecipes-rounds.md`](agents/datastage/fkrecipes-rounds.md) | the settings and prototypes declared through FkRecipes, round by round to fix round 2b | the FkRecipes subsections of that section |
| [`localised-string-ceiling.md`](agents/datastage/localised-string-ceiling.md) | the 200-byte rule's long form | the critical rule |

### Features and fixes (`agents/features/`)

| file | covers | was |
|---|---|---|
| [`carry.md`](agents/features/carry.md) | a recompile reinserts, a removal spills; the 32-group bound | "A recompile is not a removal" |
| [`miners-pocket.md`](agents/features/miners-pocket.md) | a player who mines a balancer keeps its contents; the claim identity | "The miner's pocket" |
| [`stacked-belts.md`](agents/features/stacked-belts.md) | stacked belts come back stacked; stacked sushi | "Stacked belts come back stacked" |
| [`sixty-fifth-belt.md`](agents/features/sixty-fifth-belt.md) | refusing past 64 ports without demolishing | "The sixty-fifth belt" |
| [`over-limit-merge.md`](agents/features/over-limit-merge.md) | the merge pre-pass that spares two working balancers | "The merge that would be over the limit" |
| [`wake-race.md`](agents/features/wake-race.md) | the rebuild never speaks; the playtest's findings | "The wake race" |
| [`curved-exits.md`](agents/features/curved-exits.md) | a belt that turns as it leaves is an output | "A belt that turns as it leaves" |
| [`curve-setting.md`](agents/features/curve-setting.md) | `bbb-curved-exits` and the flip | "The curve rule is a setting" |
| [`curve-upgrade.md`](agents/features/curve-upgrade.md) | pre-curve saves keep their reading; the state version; the ping list | "A save from before the curve rule keeps the reading it was built to" |
| [`priority-ports.md`](agents/features/priority-ports.md) | the priority flag, its survival, refusals, spill guard, player side | "Priorities: a port that is fed first" |
| [`fast-replace.md`](agents/features/fast-replace.md) | a part over a belt and a belt over a part; the refused-replace belt restore | "Fast replace" |
| [`single-edge-rule.md`](agents/features/single-edge-rule.md) | the working note for 2.1's rule: capability, setting, merge theorem, grandfather | "One belt per balancer part" |
| [`legacy-migration.md`](agents/features/legacy-migration.md) | adopting a Belt Balancer 2 or 3 save | "Adopting a Belt Balancer 2 or 3 save" |
| [`stub-item-cycle.md`](agents/features/stub-item-cycle.md) | the `place_result` cycle that froze a game | "The two-element cycle that froze a game" |
| [`quality-lookups.md`](agents/features/quality-lookups.md) | quality-blind tile lookups | "A part at uncommon quality is a part" |
| [`adaptive-graphics.md`](agents/features/adaptive-graphics.md) | the 47-cell sheet, the bitmask, the I/O arrows | "M5 is done" |
| [`tan-streak.md`](agents/features/tan-streak.md) | the hidden prototypes draw nothing | "The tan streak" |
| [`hidden-surface.md`](agents/features/hidden-surface.md) | `bbb-hidden` withheld from every force's surface list | "The surface in the remote view" |
| [`collision-defaults.md`](agents/features/collision-defaults.md) | following a pack that rewrites belt collision defaults (Cerys) | "A pack that rewrote the game's belt collision layers" |

### Verification (`agents/verification/`)

| file | covers | was |
|---|---|---|
| [`gates.md`](agents/verification/gates.md) | `make test`, `make datastage-check`, the release-arm gate, the dump goldens, the engine stamping | "Verification" |
| [`suites/*.md`](agents/verification/suites/) | one note per suite (table above) | the suite subsections of "Verification" |
| [`player-fixture.md`](agents/verification/player-fixture.md) | the committed client save with a player in it | "A committed save with a player in it" |
| [`layout-check.md`](agents/verification/layout-check.md) | why `test/check-layout.py` is deleted | "The layout check is gone" |

### Performance (`agents/performance/`)

| file | covers | was |
|---|---|---|
| [`heap-diet.md`](agents/performance/heap-diet.md) | the idle GC tail, and the three `-gc` decisions that ship `--gc=collected` | "The heap diet" |
| [`root-scan.md`](agents/performance/root-scan.md) | the collector budget and the post-load transient | "The root scan that could not fit in a step" |
| [`marathon-save.md`](agents/performance/marathon-save.md) | per-operation heap slopes and the 300-hour projection | "The marathon save" |
| [`benchmarks.md`](agents/performance/benchmarks.md) | the head-to-head, the re-checks, the megabase cell | "Benchmarks" |

### History (`agents/history/`)

| file | covers | was |
|---|---|---|
| [`status.md`](agents/history/status.md) | where the project stands, the old banner, the outstanding list | "Status" and the banner |
| [`verification-log.md`](agents/history/verification-log.md) | every dated verification entry, in order; append new ones here | the log paragraphs of "Status" |
| [`record-summaries.md`](agents/history/record-summaries.md) | the long-form index of the design notes and records | "Index of `agents/`" |
