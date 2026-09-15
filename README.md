# Better Belt Balancer

A Factorio 2.1 mod. Balancer parts are 1x1 tiles: place several next to each other and they become one balancer, of whatever shape you built. The belts feeding it are its inputs and the belts it feeds are its outputs; orientation alone decides, so there is nothing to configure. Each part connects to one belt, so a four-in four-out balancer is eight parts: four carrying the inputs and four carrying the outputs. Items are balanced across every output exactly, per lane, under every load condition: saturated, starved, partially blocked, asymmetric.

The mod is written in Go, compiled to WebAssembly by TinyGo and then to Lua by [FkLua](https://github.com/Techrocket9/FkLua). That covers both halves of a Factorio mod: the control stage that runs while you play, and the settings and data stages that declare the prototypes. There is no hand-written Lua in the shipped mod.

## How it works

[Existing balancer mods](https://mods.factorio.com/mod/belt-balancer-3) move items from Lua on every tick: they hold the transport lines of every belt in every balancer and shuffle items between them, so their cost grows with how many balancers exist and how busy they are.

This mod compiles instead. When a cluster of parts changes, the guest (the Go program FkLua compiled to Lua) reads the belts around it, plans a network, and builds that network out of real splitters and belts on a hidden surface, stitched to the visible world with linked belts. Then it stops. A running balancer executes no script at all: every item that moves through it is moved by the engine, exactly as if you had built the splitter tree by hand.

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

The network is a butterfly over P = next_pow2(max(inputs, outputs)) lines, log2(P) stages of P/2 splitters, with a lane-splitter stage on entry that makes it lane-accurate rather than only belt-accurate. A 4x4 balancer is 32 hidden entities; an 8x8 is 84.

One balancer supports up to 64 belts per side (inputs and outputs counted separately); the limit is the size of the hidden-surface slot a network compiles into.

## One belt per part

Each part connects to one belt. A part already serving a belt refuses a second one: the balancer you have keeps running exactly as it was, the piece comes back to your inventory, and a message says why. Rotating a belt so that it points at a part that is already serving one is refused the same way, and so is placing a part that would join two balancers into one where the joining part would end up with a belt on each side.

The reason is the engine rather than a design choice. Each belt a balancer touches is served by a hidden interface standing on that part's tile, and Factorio 2.1 allows one of those per tile. Building a wider balancer is a matter of building it a part deeper: a column of parts on the input side and a column on the output side, with as many rows as you have belts.

## Belts that turn as they leave

A belt laid across a part's free face, with nothing behind it, is that part's output. The belt turns away from the balancer, so a corner does not need a tile of straight belt first, and the turn carries both lanes exactly as a straight output does.

What decides it is the two tiles the belt could otherwise draw from: the one behind it and the one on its far side. If either holds another balancer part, or anything that feeds the belt, the belt is left alone. That is why a belt line running past a balancer stays a line running past: every tile of it has the tile before it behind it. It is also why a belt that something else already feeds is never taken as an output, even when it looks like a corner: it would carry only half a lane into the balancer's output, and a half-full output is the one thing this mod is not allowed to produce.

A consequence worth knowing when you build: a belt line that *starts* beside a part which already serves a belt is a second belt on that part, and is refused and handed back. Start it one tile further out, or give it a belt behind it, and it stays unconnected as before.

That is a change to what a factory you already have means, so it is behind a map setting, "Belts may turn as they leave a balancer", on by default. It is under Settings > Mod settings > Map, it takes effect without a restart, and changing it re-checks every balancer in the save. Turn it off and a belt across a part's face is left unconnected again, exactly as it was before this version. A balancer whose only outputs were corners then has no outputs at all: it stops, and what it was holding is returned to the ground beside it, the same as when you take a machine apart. Turning the setting back on rebuilds it.

A save made before this version keeps its old behaviour without being asked. On its first load the mod finds every balancer whose belts were laid when a belt across a face meant nothing, leaves each of them exactly as it stood, turns the setting off for that save, and says so in chat with a map ping for each one. Nothing in the world moves and nothing is lost. Turn the setting on when you want the new rule, and those are the balancers that change.

## Fast replace

Balancer parts share base's `transport-belt` fast-replace group, so a part held over a belt, an underground belt end or a lane splitter replaces it the way a splitter does: the belt and whatever it was carrying go to your inventory and the part takes the tile. Dropping a balancer straight into a belt line you already have is one click per tile. Splitters and loaders are not replaced this way (a splitter is two tiles wide, and loaders are a different group).

The group works in both directions, so a belt held over a part replaces the part, and every part can be replaced that way. A part on the edge of a balancer also carries the hidden interface that connects it to your belt, and the two are replaced together. Dragging a belt line across a balancer therefore takes out every part it crosses, exactly as dragging a belt across a row of splitters does in the base game. The parts come back as items and the balancer recompiles around what is left.

On Factorio 2.1 there is one thing to know about replacing an edge part. The belt you lay sits on the tile the part occupied, which is next to the part beyond it, so it becomes that part's belt. If that part already has one, the balancer asks for two belts on one part, which 2.1 does not allow, and it is refused and stops until you take the belt away. What the machine was holding lands on the ground beside it, because the part is gone before the mod can find out that what is left cannot be built.

A part dropped into the middle of a belt line is a different matter, because the part is then left with the belt behind it as an input and the belt ahead of it as an output. On Factorio 2.1 that is two belts on one part, so the part is refused and handed back, and the belt it replaced is put back on its tile with the line running again. It comes back empty: whatever it was carrying stays in your inventory, where the engine put it when it mined the belt, and the returned belt is paid for with the belt item you were refunded, so the count in your inventory is the one you started with. If that item is not there any more, the gap stays and the log says which item was missing.

The collision mask is unchanged: a belt still cannot be laid *through* a balancer, only fast-replaced onto one part at a time.

## Cost and research

Two startup settings decide what a balancer part costs, and under each are fields where you write the value yourself: one ingredient list under the recipe setting, and three research fields under the cost setting. All of them are under Settings > Mod settings > Startup, and all default to what the mod has always shipped, so an existing save is unchanged by the update. Startup settings need a restart of Factorio to take effect.

They are declared through [FkRecipes](https://github.com/Techrocket9/FkRecipes), the shared data-stage library, under the same names, defaults and menu order earlier versions shipped, so a choice you have already made carries over. The balancer part item, its recipe and its technology are declared through the same library, under the names they have always had, so blueprints and saved research carry over too.

**Balancer part recipe** picks the ingredient list:

| Option | Ingredients |
|---|---|
| Default | 4 iron plates, 2 gears, 2 transport belts |
| Cheap | 2 iron plates, 1 transport belt |
| Fast belts | 4 iron plates, 2 gears, 2 fast transport belts |
| Express belts | 4 steel plates, 2 gears, 2 express transport belts |
| Splitter | 1 splitter, 2 iron plates |
| Express splitter | 1 express splitter, 2 steel plates |

**Balancer part ingredients** sits under that setting and applies whenever it does not say `default`. Write the amount, then the item's internal name, and separate ingredients with commas: `2 iron-plate, 1 splitter`. Internal names are the ones the game uses in its own data and in rich text (`iron-plate`, not "Iron plate"). The word `default` hands the list back to the option chosen above, whose tooltip lists every option written this way, so you can start from the one you were on. The word `none` empties the list, and a balancer part then costs nothing to craft. The recycling recipe the game derives from this one goes with it, so where you have recycling a balancer part can no longer be recycled. A name your game does not have, a display name, or a fluid is never guessed at: the whole list is set aside, the option chosen above applies instead, and the log carries one line naming the setting, the entry and the reason. The game loads, so you can correct the field and restart. Because the option takes over whenever the field cannot be used, a typo in it is a recipe change, with the consequence described at the end of this section. The full format, with every message and what causes it, is FkRecipes' [ingredient list reference](https://github.com/Techrocket9/FkRecipes/blob/master/docs/ingredient-list.md).

**Balancer research cost** picks which technology the balancer's own research is priced like: Logistics (the default), Logistics 2 or Logistics 3. The cost is read from that technology rather than written down, so it follows whatever your mods charge for that tier, and the balancer's prerequisite moves with it so that it sits beside its own tier in the technology tree.

Three settings under it reprice the research one field at a time: the science packs, written the same way as the recipe and taking science packs only (`1 automation-science-pack, 1 logistic-science-pack`); how many units the research costs; and how many seconds one unit takes. The pack field applies whenever it does not say `default`, and each number applies whenever it is not 0. A field you leave alone is decided by the tier above, which for Logistics in the base game is 20 automation science packs at 15 seconds each. Repricing never moves the research in the technology tree: the tier you picked is still its prerequisite, whatever the three fields say. A pack list this mod cannot use is set aside whole and the tier's own packs apply instead, exactly as an ingredient list is. Setting the unit count also drops a `count_formula` if the tier you picked has one: the engine prices a research carrying both by the formula, so the number you typed would be read by nobody, and the log says which setting took the formula. None of the three logistics tiers is priced that way in the base game.

The presets and the tiers are built to survive an overhaul pack, and the one place they can run out is the science packs. No ingredient name reaches the game unless that item is present: each one has a chain of substitutes ending at iron plate, and the first item your mod set actually has is the one used. Where a substitute is something the list already names, the two amounts are added and the recipe keeps one entry for it, so a pack with no transport belts makes a balancer part out of 6 iron plates and 2 gears instead of 4, 2 and 2, and what you craft can be a shorter list than the one the option shows. That last rung is the one thing the presets do assume: a mod set with no iron plate is outside what they cover. The same holds for the research: if the technology an option names is missing, or has been turned into a trigger technology with no research cost, the next tier down is used, and if none of them can be read the balancer costs what base charges for Logistics, sits at the root of the technology tree with no prerequisite, and says both in its own description. What the tier supplies is its whole research cost, including its level cap, so in a mod set that gave one of the three several levels the balancer research gains them too and only the first level unlocks anything. None of the three has a level cap in a stock game, with or without the expansions. A list you write yourself is the opposite on purpose: it is taken as written and nothing is substituted for a name you typed, so every name must exist in your game. A list with one that does not is set aside whole and the option chosen above applies instead, with the reason in the log, which is the answer you want for a list you typed. There are two ways into that. Where no tier can be read, the fallback cost applies and asks for an automation science pack; and where a tier is read but every pack it charges is missing from your game, or no longer a science pack in it, the fallback cost applies to that tier too, keeps its prerequisite, and asks for the same pack. If your game has no automation science pack either, or has that name as an ordinary item rather than as a science pack, the research is left with no science pack at all. The game loads, and the research is built with an empty cost, which means it completes for free; its own description in the technology screen says that this game has none of the science packs it names, and the log carries one line saying so. A research you were meant to pay for costing nothing is a balance change you did not choose, and the science packs field is what answers it: write the packs yourself, in packs your game does have, and that list is what the research costs.

Changing what a balancer part costs to craft, whether by picking another option, by editing the ingredients field, or by introducing a typo into a list that was working and so setting the whole list aside, rewrites the one recipe this mod has rather than adding a second one, so an assembling machine already set to make balancer parts loses whatever is in its input slots that the new list does not use: those items are destroyed rather than dropped on the ground, with nothing said in the log. What the machine had already finished stays in its output slot, and the chests and belts around it keep everything they were holding. Empty the machine before you change the option or edit the field if it is holding something you would miss.

## Belt speed

The belts inside the hidden network run at the speed of the fastest belt in your game, and never slower than 0.25 tiles per tick (120 items per second, about 2.7x express and 2x turbo). In a vanilla or Space Age game nothing is faster than that, so the network runs at 0.25 and a balancer never limits the belts feeding it.

If a mod adds a belt faster than 0.25, the network follows it. Every belt-connectable prototype in the game is read at the last data stage, the fastest speed among them wins, and the four hidden prototypes are given it. That covers transport belts, underground belts, splitters, lane splitters, loaders and linked belts, so a mod that raises only one of those families is still seen. A mod whose own final data stage runs after this one and raises a belt then is the one case that is missed, and the cost is the old behaviour: the network runs at the second-fastest belt's speed.

## Performance

Measured on Factorio 2.0.77 headless, base only, Apple M3 Pro, against belt-balancer-2 v2.0.9 and belt-balancer-3 v1.0.1 on identical rigs with a no-balancer control, every arm run back to back in one session. Method, caveats and raw rows: [`bench/baselines/RESULTS.md`](bench/baselines/RESULTS.md); harness: [`bench/README.md`](bench/README.md).

| per saturated 4x4 balancer, per tick | bb2 | bb3 | this mod | ratio |
| --- | --: | --: | --: | --- |
| express belts, whole tick | 21.9 µs | 23.1 µs | 0.49 µs | 45× / 47× |
| express belts, mod Lua only | 19.1 µs | 21.0 µs | 0 | equal to the control |
| normal belts, whole tick | 7.55 µs | 7.67 µs | 0.35 µs | 22× |

- 200 saturated express balancers cost 0.64 ms/tick against bb2's 4.92 ms: 4% of the 16.67 ms 60-UPS budget instead of 30%. 500 of them cost 2.05 ms/tick.
- `scriptUpdate` matches the no-mod control in every cell. There is no `on_tick` handler; all compiling happens when you build, and a benchmark of a finished save runs none of this mod's Lua.
- Balance is exact: 1,740,000 items over 200 rigs at a max/min of 1.001, with a per-output spread of 0.15% saturated. Starvation, blocked outputs, asymmetric port counts and recompiles under load all hold, with a headless test for each.
- A megabase mix (404 balancers of ten shapes plus a 16x16, a 32x32 and a 64x64; 4,376 hidden splitters) costs 0.33 µs per balancer per tick, and the 64x64 splits 64 ways at a max/min of 1.0028.

## Item conservation

- Editing a running balancer never deletes items or drops them on the floor: it drains the hidden network and puts the items straight back into the network it rebuilds (1,173 in, 1,173 out, checked inside one tick; zero items on the ground across a hundred add/remove cycles on a saturated rig). Only a real removal, the last part mined or the surface deleted, returns them to the world.
- Mining a balancer hands the drained items to the miner's inventory at every step, not only when the last part goes, and mining a belt off its edge counts too. Only what the inventory cannot take spills, as when mining a splitter that was holding items; a robot deconstruction still spills.
- Space Age stacked belts come back stacked.

## Migrating from Belt Balancer 2 or 3

If you are already using Belt Balancer 2 or Belt Balancer 3 this mod adopts what they built. Uninstall the old mod and load your save with BetterBeltBalancer installed: every balancer part left standing from the incumbent mod becomes one of this mod's parts, at the health and the quality it was standing at, and your stacks and blueprints of the old part keep working: each one places a compatibility part that becomes one of this mod's on the next tick. The conversion happens once, at load, before the first tick, and the log carries one line saying how many parts on how many surfaces became how many balancers.

**On Factorio 2.1 most of those balancers will not run again until you rebuild them, and that is worth knowing before you make the cutover save.** This mod allows one belt per balancer part, because 2.1 allows one belt-connectable per tile and every edge of a balancer is one. Belt Balancer's own layout is a single column of parts with a belt on both sides of each, which is two belts per part, so a balancer built that way is converted and then refused: you keep the parts, you keep the items on your belts, and each affected force gets one chat message naming how many balancers need rebuilding with a clickable map ping per balancer. **The exception is a balancer you already built one belt per part** -- a two-column block, inputs down one side and outputs down the other -- which converts into a working balancer and runs at full rate straight away.

So: migrating on 2.1 converts your parts and hands you a rebuild checklist rather than a working machine, except for the balancers already laid one belt per part.

Nothing happens while the old mod is still installed. Both can sit in a mod list together for as long as you like; this mod does not touch a `balancer-part` that belongs to a mod that is running, and it does not touch one that belongs to any other mod either.

**Make sure** you don't have anything of very high value inside your legacy balancer when you make your cutover save: the items the old mod was holding in its own buffer are **lost** in the migration. It keeps them in its own script state, which Factorio deletes along with the mod before anything of this one's runs. The items on the belts themselves are untouched.

## Building

Prerequisites: Go, TinyGo 0.41.1, binaryen (`wasm-opt`, which TinyGo's wasm build shells out to), Python 3 (the sprite check, the test assertion scripts and the art generator), a checkout of [FkLua](https://github.com/Techrocket9/FkLua) at `../FkLua` with `bin/fklua` built (`FKLUA=/path/to/fklua` overrides), and a checkout of [FkRecipes](https://github.com/Techrocket9/FkRecipes) at `../FkRecipes`, the shared data-stage library the mod's startup settings and its craftable prototypes are declared through. FkRecipes is not published yet, so the guest module consumes it through a `replace` pointing at that sibling checkout. The headless tests and the benchmarks also need a Factorio 2.1 install; set `FACTORIO_BIN` if it is not at the default Steam location on macOS.

```sh
make zip      # dist/better-belt-balancer_<version>.zip, a complete mod
make install  # unpacked, into your Factorio mods directory (MODS_DIR overrides)
make check    # pure-Go unit tests, bindings and lockfile current, gofmt
make test     # headless verification in a real Factorio
```

`make test` creates saves with the rigs already built, benchmarks them in a real Factorio, and asserts against the guest's own log lines. Two build switches:

- `QUIET=1` compiles out every `[BBB]` log line below the error level. The default build is verbose because the suites assert on those lines.
- `GC=leaking` builds the guest on FkLua's leaking arena instead of its paced collector. The shipped build is collected: over 3,400 teardown-and-rebuild cycles the leaking arm's heap doubled its way to 32 MiB with a 782 ms tick at the last doubling, where the collected arm ended at 0.5 MiB with a worst tick of 71 ms and no measurable steady-state difference. Both arms build, and the suites that run today are green in both.

- The icon, logo, and sprites were created by [Edjie Arts](https://edjie.carrd.co). A balancer reads as one continuous machine across any shape, with trim only along its real outline.

## Repository layout

| path | contents |
| --- | --- |
| `guest/go/` | the control guest; `data/` is the settings and data stages, `plan/` the network planner, `tune/` what the cost settings and the belt-speed derivation decide plus the FkRecipes plan, `fkapi/` the generated FkLua bindings |
| `mod-data/` | the assets the package carries verbatim: graphics, locale, changelog, thumbnail |
| [`bench/`](bench/README.md) | the head-to-head benchmark harness, its setup mod and the results |
| `test/` | the headless suites and their assertion scripts; `fixtures/` holds a small mod, also written in Go, that a data-stage check builds and stages, and `fixtures-player/` a small save with a player in it, so that the suites can drive a real cursor; [`test/interactive/`](test/interactive/README.md) is the checklist for the things a headless run cannot check, and the mod that stages both its rigs and the demo scenes |
| `fklua.toml` | mod identity, the API pin, guest language, GC mode and the data module |
| `CLAUDE.md`, `agents/` | maintainer design notes and the full measurement record |

## Licence

Released under the [MIT License](LICENSE), artwork included. FkLua, which compiles the guest, is MIT licensed as well.

The artwork under `mod-data/graphics/` and `mod-data/thumbnail.png` was created for this project by [Edjie Arts](https://edjie.carrd.co) and is distributed under that licence with attribution retained. If you redistribute or modify this mod, keep the credit.

## Credits

Artwork by [Edjie Arts](https://edjie.carrd.co).
