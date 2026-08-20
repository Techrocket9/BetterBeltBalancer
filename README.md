# Better Belt Balancer

A Factorio 2.0 mod. Balancer parts are 1x1 tiles: place several next to each
other and they become one balancer, of whatever shape you built. The belts
feeding it are its inputs and the belts it feeds are its outputs; orientation
alone decides, so there is nothing to configure. Items are balanced across
every output exactly, per lane, under every load condition: saturated, starved,
partially blocked, asymmetric.

The mod's logic is written in Go, compiled to WebAssembly by TinyGo and then to
Lua by [FkLua](https://github.com/Techrocket9/FkLua). It was FkLua's first
downstream mod; [FKLUA-GAPS.md](FKLUA-GAPS.md) records what building it asked
of the compiler.

## How it works

Existing balancer mods move items from Lua on every tick: they hold the
transport lines of every belt in every balancer and shuffle items between them,
so their cost grows with how many balancers exist and how busy they are.

This mod compiles instead. When a cluster of parts changes, the guest (the Go
program FkLua compiled to Lua) reads the belts around it, plans a network, and
builds that network out of real splitters and belts on a hidden surface,
stitched to the visible world with linked belts. Then it stops. A running
balancer executes no script at all: every item that moves through it is moved
by the engine, exactly as if you had built the splitter tree by hand.

```
   what you build                        what the guest compiles it into
   --------------                        ------------------------------
                                         hidden surface, this cluster's slot
   ==>[#][#]==>                          [>] [L] --.   .-- [S] --.   .-- [>]
   ==>[#][#]==>       <== linked ==>     [>] [L] --'`-'`-- [S] --'`-'`-- [>]
   ==>[#][#]==>           belts          [>] [L] --.   .-- [S] --.   .-- [>]
   ==>[#][#]==>                          [>] [L] --'`-'`-- [S] --'`-'`-- [>]

   eight parts, one cluster.             [>] linked belt   [L] lane splitter
   Four belts point in on the west       [S] splitter.  Two butterfly stages
   face, four point away on the east:    for four lines; the lane splitters
   that is four inputs and four          are why the two lanes of one belt
   outputs, and nothing was configured.  balance and not just the belts.
```

The network is a butterfly over P = next_pow2(max(inputs, outputs)) lines,
log2(P) stages of P/2 splitters, with a lane-splitter stage on entry that makes
it lane-accurate rather than only belt-accurate. A 4x4 balancer is 32 hidden
entities; an 8x8 is 84.

One balancer supports up to 64 belts per side (inputs and outputs counted
separately); the limit is the size of the hidden-surface slot a network
compiles into. A cluster asking for more is refused before anything is touched:
the balancer you already had keeps running, you get the standard cannot-build
sound and a flying text naming the limit, and the belt or part comes back to
your inventory. A construction robot or another mod placing it gets a force-wide
message and the piece is left standing unconnected. Bridging two balancers into
one that would be over the limit is refused the same way, with both left
running. [`agents/maxports.md`](agents/maxports.md) records what raising the
limit would take.

## Fast replace

Balancer parts share base's `transport-belt` fast-replace group, so a part held
over a belt, an underground belt end or a lane splitter replaces it the way a
splitter does: the belt and whatever it was carrying go to your inventory and the
part takes the tile. Dropping a balancer straight into a belt line you already
have is one click per tile. Splitters and loaders are not replaced this way (a
splitter is two tiles wide, and loaders are a different group).

The group works in both directions, so a belt held over a part replaces the part.
Only parts with no belt against any of their free faces can be replaced that way:
a part on the edge of a balancer carries the hidden interface that connects it to
your belt, and that interface blocks the placement. Dragging a belt line across a
balancer therefore takes out the parts in the middle and is refused at the edges,
exactly as dragging a belt across a splitter does in the base game. The parts come
back as items and the balancer recompiles around what is left.

The collision mask is unchanged: a belt still cannot be laid *through* a
balancer, only fast-replaced onto one part at a time.

## Performance

Measured on Factorio 2.0.77 headless, base only, Apple M3 Pro, against
belt-balancer-2 v2.0.9 and belt-balancer-3 v1.0.1 on identical rigs with a
no-balancer control, every arm run back to back in one session. Method, caveats
and raw rows: [`bench/baselines/RESULTS.md`](bench/baselines/RESULTS.md);
harness: [`bench/README.md`](bench/README.md).

| per saturated 4x4 balancer, per tick | bb2 | bb3 | this mod | ratio |
| --- | --: | --: | --: | --- |
| express belts, whole tick | 21.9 µs | 23.1 µs | 0.49 µs | 45× / 47× |
| express belts, mod Lua only | 19.1 µs | 21.0 µs | 0 | equal to the control |
| normal belts, whole tick | 7.55 µs | 7.67 µs | 0.35 µs | 22× |

- 200 saturated express balancers cost 0.64 ms/tick against bb2's 4.92 ms:
  4% of the 16.67 ms 60-UPS budget instead of 30%. 500 of them cost 2.05 ms/tick.
- `scriptUpdate` matches the no-mod control in every cell. There is no
  `on_tick` handler; all compiling happens when you build, and a benchmark of a
  finished save runs none of this mod's Lua.
- Balance is exact: 1,740,000 items over 200 rigs at a max/min of 1.001, with a
  per-output spread of 0.15% saturated. Starvation, blocked outputs, asymmetric
  port counts and recompiles under load all hold, with a headless test for each.
- A megabase mix (404 balancers of ten shapes plus a 16x16, a 32x32 and a
  64x64; 4,376 hidden splitters) costs 0.33 µs per balancer per tick, and the
  64x64 splits 64 ways at a max/min of 1.0028.

## Item conservation

- Editing a running balancer never deletes items or drops them on the floor: it
  drains the hidden network and puts the items straight back into the network
  it rebuilds (1,173 in, 1,173 out, checked inside one tick; zero items on the
  ground across a hundred add/remove cycles on a saturated rig). Only a real
  removal, the last part mined or the surface deleted, returns them to the world.
- Mining a balancer hands the drained items to the miner's inventory at every
  step, not only when the last part goes, and mining a belt off its edge counts
  too. Only what the inventory cannot take spills, as when mining a splitter
  that was holding items; a robot deconstruction still spills.
- Space Age stacked belts come back stacked after a recompile: 928 items in 232
  four-stacks return as 232 four-stacks, exact per (name, quality), nothing
  spilled. Base Factorio pays nothing for the path, which is gated on the
  force's belt-stacking bonus.

## Migrating from Belt Balancer 2 or 3

If you are already using Belt Balancer, Belt Balancer 2, Belt Balancer 3 or Belt
Balancer Performance, this mod adopts what they built. Uninstall the old mod and
load your save: every balancer part it left standing becomes one of this mod's
parts, at the same tiles, on the same force, at the same quality and health, and
the belts around it become that balancer's inputs and outputs exactly as they
were. The conversion happens once, at load, before the first tick, and the log
carries one line saying how many parts on how many surfaces became how many
balancers.

Nothing happens while the old mod is still installed. Both can sit in a mod list
together for as long as you like; this mod does not touch a `balancer-part` that
belongs to a mod that is running, and it does not touch one that belongs to any
other mod either.

What comes across:

- **The balancers**, as balancers, working. They compile into hidden networks on
  the load that adopts them, so a save that had 50 of them has 50 of them.
- **Everything on the belts.** Those belts are vanilla and are not touched.
- **Your parts in chests and inventories.** A stack of the old mod's part item
  survives and places this mod's parts, so a chest full of them is still a chest
  full of balancer parts. The stacks keep the old name.
- **Your blueprints.** A blueprint or book taken with the old mod installed still
  places balancers; each part a robot builds from it becomes one of this mod's a
  tick later.
- **The ability to craft.** The old mod's technologies go with it, so any force
  that owned a balancer is given this mod's balancer technology.

What does not:

- **The items the old mod was holding.** Belt Balancer 2 and 3 take items off
  the belts and hold them in a Lua table of their own, up to two per output lane
  per balancer, and Factorio deletes a removed mod's saved state along with the
  mod before any script can read it. That is a handful of items per balancer and
  there is no mechanism that could recover them.
- **The old mod's technologies and recipes**, which the engine removes.

## Building

Prerequisites: Go, TinyGo 0.41.1, binaryen (`wasm-opt`, which TinyGo's wasm
build shells out to), Python 3 (the sprite check, the test assertion scripts and
the art generator), and a checkout of [FkLua](https://github.com/Techrocket9/FkLua)
at `../FkLua` with `bin/fklua` built (`FKLUA=/path/to/fklua` overrides). The
headless tests and the benchmarks also need a Factorio 2.0 install; set
`FACTORIO_BIN` if it is not at the default Steam location on macOS.

```sh
make zip      # dist/better-belt-balancer_<version>.zip, a complete mod
make install  # unpacked, into your Factorio mods directory (MODS_DIR overrides)
make check    # pure-Go unit tests, bindings and lockfile current, gofmt
make test     # headless verification: nine suites in a real Factorio
```

`make test` creates saves with the rigs already built, benchmarks them in a real
Factorio, and asserts against the guest's own log lines. Two build switches:

- `QUIET=1` compiles out every `[BBB]` log line below the error level. The
  default build is verbose because the suites assert on those lines.
- `GC=leaking` builds the guest on FkLua's leaking arena instead of its paced
  collector. The shipped build is collected: over 3,400 teardown-and-rebuild
  cycles the leaking arm's heap doubled its way to 32 MiB with a 782 ms tick at
  the last doubling, where the collected arm ended at 0.5 MiB with a worst tick
  of 71 ms and no measurable steady-state difference. Both arms pass all nine
  suites.

## Status

- Works and is benchmarked: cluster registry, network compiler, 2.0 lifecycle
  handling (clones, blueprints, ghosts, robots, undo, surface deletion, mod
  upgrade, space platforms), adaptive graphics. Not on the mod portal: no
  release, and no play-testing beyond the suites and the interactive checklist.
- The art is by [Edjie Arts](https://edjie.carrd.co): the 47-variant sprite
  sheet, the item icon, the input and output arrows and the mod logo. A
  balancer reads as one continuous machine across any shape, with trim only
  along its real outline. `tools/make-graphics.py` remains in the repository as
  the generator for the placeholder set it replaced, and as the definition of
  the 47-mask cell order the guest and the sheet must agree on.
- One visual residual. The edge interfaces are invisible, but an item reaching
  the far end of an interface belt overhangs the tile edge by about a sixth of a
  tile, so a balancer with a notch in it (a 2x2 missing a corner) shows a thin
  band of moving items in the empty corner. Factorio 2.0.77 has no prototype
  field that suppresses item drawing on a belt-connectable and no linked-belt
  equivalent of a loader's `belt_length`.

## Repository layout

| path | contents |
| --- | --- |
| `guest/go/` | the Go guest; `plan/` is the network planner, `fkapi/` the generated FkLua bindings |
| `mod-data/` | the hand-written data stage: prototypes, graphics, locale |
| [`bench/`](bench/README.md) | the head-to-head benchmark harness, its setup mod and the results |
| `test/` | the nine headless suites and their assertion scripts; [`test/interactive/`](test/interactive/README.md) is the checklist for the six player gestures a headless run cannot make |
| `fklua.toml` | mod identity, the API pin (2.0.77), guest language and GC mode |
| `CLAUDE.md`, `agents/` | maintainer design notes and the full measurement record |

## Licence

Released under the [MIT License](LICENSE). FkLua, which compiles the guest, is
MIT licensed as well.

The artwork under `mod-data/graphics/` and `mod-data/thumbnail.png` was created
for this project by [Edjie Arts](https://edjie.carrd.co) and is used with
credit. The placeholder set it replaced is still generated by
`tools/make-graphics.py`, and that generator and its output carry the project
licence.

## Credits

Artwork by [Edjie Arts](https://edjie.carrd.co).
