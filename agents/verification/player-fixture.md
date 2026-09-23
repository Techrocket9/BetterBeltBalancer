# A committed save with a player in it

**Every "needs a player" statement this file has ever made rests on one fact, and it is a fact about how a world is MADE rather than about what a script may do.** A `game.players` entry exists only where somebody once connected. Nothing creates one: not `create_entity`, not a force, not a controller. A headless `--create` therefore has none, `game.get_player(1)` is nil in sixteen of the seventeen suites, and everything downstream of a player index -- the miner's pocket's beneficiary, the over-limit hand-back, a cursor build -- returns before it does anything.

**So the sixteenth suite starts from a save a graphical CLIENT made**, reduced to the one thing it wants out of it. `test/fixtures-player/player-2.0.77.zip` is **382,986 B**: one player, one nauvis of nine chunks, no entities, no second surface, no inventory, recorded against BASE ALONE.

**IT IS REGENERABLE FROM ANY CLIENT SAVE AND THE COMMAND IS THE RECORD.**

```sh
make player-fixture SAVE="$HOME/Library/Application Support/factorio/saves/whatever.zip"
```

`test/make-player-fixture.sh` copies the source save before anything reads it -- the user's own saves and mods directories are read-only to it -- and runs `guest/go/obs/fixplayer` over the copy TWICE. The product's file name carries the engine it was recorded on, read off `--version` rather than written down. The save this one was cut from was made by **Factorio 2.0.77 (build 84539)** with base, the three DLC mods, `better-belt-balancer` and `wormhole-belts` loaded; the source is not committed and nothing here needs it again. End to end it is about eleven seconds, and it is DETERMINISTIC: two cuts from the same source save came out byte-identical, `26c75ecb8fd47549...`, so re-cutting one to check something moves nothing.

**THE SAVE ROUTE IS `--start-server` AND `game.server_save`, BECAUSE THE OTHER ONE DOES NOT EXIST.** `--benchmark` never writes a save: `game.auto_save(name)` returns true there and produces no file and no log line, measured over 10 ticks and over 120. A server saves on demand -- and `saves/` has to exist first, or `server_save` refuses with `filesystem error: in canonical: No such file or directory`, which is how a headless server that has never saved leaves it.

**THE SECOND PASS IS WHAT MAKES THE PRODUCT COMMITTABLE.** It loads the first pass's output with base alone, so Factorio drops the prototypes and the `storage` of every mod the source save names. Measured on this one: `script.dat` **5,868,208 B -> 1,281 B**, and the whole file 3,130,691 -> 382,986. What is left is the freeplay scenario's own baked initial level (`level-init.dat`, 735,458 B uncompressed), which nothing a script can reach removes.

**AND THE CUTTER IS THE ONE PACKAGE HERE BUILT `--persist=none`, WHICH IS A CORRECTNESS MATTER.** Its schedule is keyed off the first tick it sees and the two passes run over the same world; under `packed` the second pass ADOPTS the first's heap, `start` comes back holding the first pass's tick, every step is already behind it, and the run does nothing but log its last line -- measured, pass 2 wrote no save at all. A one-shot cutter has no state worth carrying across a save, so it carries none.

**WHAT THE FIXTURE CANNOT HOLD is a connected player**, and the `curs` suite's own section is where that is measured and answered: a server load disconnects everyone and destroys their character, a disconnected player cannot be given one back, and the GOD controller -- which such a player can hold -- drives `build_from_cursor` and `mine_entity` with the same event trace, the same mine buffer and the same inventory arithmetic a character produces.
