// Command bbb-fixplayer turns a save a graphical client made into the minimal
// committable fixture the `curs` suite loads: one player, one small nauvis, and
// nothing else.
//
// IT IS NOT AN OBSERVER AND IT IS NOT STAGED BY test/run.sh. `make
// player-fixture SAVE=<path>` runs it, twice, and the product is a file in
// test/fixtures-player/. Nothing in the estate imports it and no suite enables
// it; it is here because it is a Go guest and this repository holds no
// hand-written Lua.
//
// # Why a fixture is needed at all, and why it cannot be built by --create
//
// A `game.players` entry exists only where somebody once connected. A `--create`
// never has one, which is the wall every "needs a player" statement in CLAUDE.md
// names: `game.get_player(1)` is nil in fourteen of the fifteen suites because
// their worlds were made by a headless map generator. So the `curs` suite's
// world has to start from a save a CLIENT made, and the only thing it wants out
// of that save is the player object.
//
// # The save route, measured
//
// `--benchmark` NEVER WRITES A SAVE. `game.auto_save(name)` returns true there
// and produces no file and no log line, on 10 ticks and on 120 -- so the strip
// cannot be done in the mode the suites run in.
//
// `--start-server` DOES: `game.server_save(name)` writes
// <write-data>/saves/<name>.zip and logs its own progress. That is the route,
// and it costs one thing: a server load DISCONNECTS every player and destroys
// their character. What the fixture therefore holds is a player with no
// character, and `player.character = <a character this mod just created>` is
// refused on it by name -- "User isn't connected; can't write character".
//
// THE SUITE ANSWERS THAT WITH THE GOD CONTROLLER, and obs/curs's header carries
// the measurement: a disconnected player in god mode drives `build_from_cursor`
// and `mine_entity` and produces the same event trace, the same buffer and the
// same inventory arithmetic as a connected character does, to the item.
//
// # Two stages, and the second is the one that makes it committable
//
// A client save is recorded against a mod set this repository does not have, and
// its script.dat carries every one of those mods' state -- 5.8 MB of it on the
// save this fixture was cut from. So the strip runs TWICE: once with the source
// save's own mod set, where the world is emptied, and once with BASE ALONE,
// where Factorio drops every prototype and every `storage` the first pass left
// behind. Loading a save whose mods are missing is a warning in headless, which
// is what the `mig` suite's mod-set changes already rest on.
//
// What is left after both passes is the freeplay scenario's own baked initial
// level (level-init.dat), which nothing a script can reach removes.
package main

import (
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/fkapi"
	"github.com/Techrocket9/BetterBeltBalancer/guest/go/obs/harness"
	"github.com/Techrocket9/fklua/guest/go/fk"
)

// SaveName is what `game.server_save` is handed, and test/make-player-fixture.sh
// reads the file out of the server's own saves directory under it.
const SaveName = "bbb-player-fixture"

// KeepChunks is the Chebyshev radius in CHUNKS kept around the origin. One is
// nine chunks, which is more than the suite needs -- it builds its own surface --
// and enough that the player is standing on generated ground rather than in a
// hole.
const KeepChunks = 1

// The schedule. Every step is separated from the next because two of them are
// ASYNCHRONOUS in the engine: `game.delete_surface` takes several ticks to
// finish (measured: a surface deleted at tick 3 was still in `game.surfaces` at
// tick 9 and gone by tick 30), and `server_save` writes after the tick returns.
const (
	tickStrip  = 3
	tickEntity = 8
	tickChunks = 14
	tickVerify = 40
	tickSave   = 42
	tickDone   = 90
)

var out = harness.Line{Tag: "[FIXPLAYER] "}

// playerInventories is every inventory a player can be holding something in.
// Filled at init from the accessors rather than written down, because a define's
// number is Factorio's own and is not stable across versions (agents/verification/layout-check.md,
// "The layout check is gone").
var playerInventories []uint32

func init() {
	fkapi.Subscribe(fkapi.EventOnTick)
	playerInventories = []uint32{
		fkapi.DefinesInventoryCharacterMain(),
		fkapi.DefinesInventoryCharacterGuns(),
		fkapi.DefinesInventoryCharacterAmmo(),
		fkapi.DefinesInventoryCharacterArmor(),
		fkapi.DefinesInventoryCharacterTrash(),
		fkapi.DefinesInventoryGodMain(),
	}
}

func surfaces() []fkapi.EntryValueObject {
	all, err := fkapi.Game.Surfaces()
	if err != nil {
		harness.Fatal("reading game.surfaces", fk.LastError())
		return nil
	}
	return all
}

func nauvis() (fkapi.LuaSurface, bool) {
	o, err := fkapi.Game.GetSurface(fkapi.OfString("nauvis"))
	if err != nil || o == nil {
		return fkapi.LuaSurface{}, false
	}
	return fkapi.LuaSurface{Object: *o}, true
}

func players() []fkapi.EntryValueObject {
	all, err := fkapi.Game.Players()
	if err != nil {
		harness.Fatal("reading game.players", fk.LastError())
		return nil
	}
	return all
}

// dropSurfaces deletes every surface but nauvis. A hidden surface this mod under
// test created is one of them: it is script state, it survives the mod being
// removed, and a fixture that carried one would hand the suite a world its own
// mod had not made.
func dropSurfaces() {
	var dropped int64
	for _, e := range surfaces() {
		s := fkapi.LuaSurface{Object: e.Val}
		name, err := s.Name()
		if err != nil || name == "nauvis" {
			continue
		}
		if ok, err := fkapi.Game.DeleteSurface(e.Val); err == nil && ok {
			dropped++
			out.Open("dropped surface ").S(name).End()
		}
	}
	out.Open("surfaces dropped=").I(dropped).End()
}

// stripPlayers empties what a player is carrying and puts them at the origin.
//
// THE CURSOR FIRST, because clearing the inventory a cursor stack came out of
// does not clear the cursor, and a fixture whose player is holding something is
// a fixture whose first inventory count is not zero.
func stripPlayers() {
	nau, ok := nauvis()
	if !ok {
		harness.Fatal("no nauvis to put the player on", fk.LastError())
		return
	}
	for _, e := range players() {
		p := fkapi.LuaPlayer{Object: e.Val}
		if _, err := p.ClearCursor(); err != nil {
			out.Open("clear_cursor refused: ").S(fk.LastError()).End()
		}
		for _, id := range playerInventories {
			inv, err := p.GetInventory(id)
			if err != nil || inv == nil {
				continue
			}
			if err := (fkapi.LuaInventory{Object: *inv}).Clear(); err != nil {
				out.Open("inventory ").U(uint64(id)).S(" refused clear").End()
			}
		}
		if _, err := p.Teleport(fkapi.MapPosition{X: 0.5, Y: 0.5}, &nau.Object, nil, nil, nil); err != nil {
			out.Open("teleport refused: ").S(fk.LastError()).End()
		}
		reportPlayer("stripped", p)
	}
}

func reportPlayer(tag string, p fkapi.LuaPlayer) {
	name, _ := p.Name()
	index, _ := p.Index()
	connected, _ := p.Connected()
	ch, _ := p.Character()
	pos, _ := p.Position()
	surf, _ := p.Surface()
	sname, _ := (fkapi.LuaSurface{Object: surf}).Name()
	out.Open("player tag=").S(tag).
		S(" index=").U(uint64(index)).
		S(" name=").S(name).
		S(" connected=").B(connected).
		S(" character=").B(ch != nil).
		S(" surface=").S(sname).
		S(" x=").F1(pos.X).S(" y=").F1(pos.Y).End()
}

// clearNauvis destroys everything standing on nauvis EXCEPT a character.
//
// A character is the one entity a strip may not take: where the input save has
// one it belongs to a player, and destroying it would leave that player pointing
// at nothing. (The server route has already destroyed it by the time this runs,
// so on the save this fixture was cut from the count is zero -- the rule is here
// because the input is "any client-made save" and not that one.)
func clearNauvis() {
	nau, ok := nauvis()
	if !ok {
		return
	}
	all, err := nau.FindEntitiesFiltered(fkapi.EntitySearchFilters{})
	if err != nil {
		harness.Fatal("sweeping nauvis", fk.LastError())
		return
	}
	no := false
	var killed, chars int64
	for _, o := range all {
		if harness.EntityTypeIs(o, "character") {
			chars++
			continue
		}
		if ok, err := (fkapi.LuaEntity{Object: o}).Destroy(
			fkapi.LuaEntityDestroyArgs{RaiseDestroy: &no}); err == nil && ok {
			killed++
		}
	}
	out.Open("nauvis swept: destroyed=").I(killed).S(" characters kept=").I(chars).End()
}

// dropChunks deletes every chunk outside KeepChunks of the origin.
//
// THE POSITIONS ARE COLLECTED BEFORE ANY OF THEM IS DELETED. `get_chunks`
// returns a live iterator over the surface's chunk map and `delete_chunk`
// mutates that map, so walking and deleting in one pass is iteration over a
// container being modified -- which is the defect class this repository's own
// flush loops are written around.
func dropChunks() {
	nau, ok := nauvis()
	if !ok {
		return
	}
	it, err := nau.GetChunks()
	if err != nil {
		harness.Fatal("get_chunks", fk.LastError())
		return
	}
	iter := fkapi.LuaChunkIterator{Object: it}
	var doomed []fkapi.ChunkPosition
	var kept int64
	for {
		c, err := iter.Call()
		if err != nil || c == nil {
			break
		}
		if abs32(c.X) <= KeepChunks && abs32(c.Y) <= KeepChunks {
			kept++
			continue
		}
		doomed = append(doomed, fkapi.ChunkPosition{X: c.X, Y: c.Y})
	}
	for _, c := range doomed {
		if err := nau.DeleteChunk(c); err != nil {
			out.Open("delete_chunk refused at ").I(int64(c.X)).S(",").I(int64(c.Y)).End()
		}
	}
	out.Open("chunks deleted=").I(int64(len(doomed))).S(" kept=").I(kept).End()
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// verify reports what the fixture is about to be. It asserts nothing -- that is
// test/make-player-fixture.sh's job -- but a run whose player count is zero has
// produced a fixture the suite cannot use, and the line is where that is seen.
func verify() {
	nau, ok := nauvis()
	if !ok {
		harness.Fatal("no nauvis at verify", fk.LastError())
		return
	}
	var chunks int64
	if it, err := nau.GetChunks(); err == nil {
		iter := fkapi.LuaChunkIterator{Object: it}
		for {
			c, err := iter.Call()
			if err != nil || c == nil {
				break
			}
			chunks++
		}
	}
	entities, _ := nau.CountEntitiesFiltered(fkapi.EntitySearchFilters{})
	all := players()
	for _, e := range all {
		reportPlayer("final", fkapi.LuaPlayer{Object: e.Val})
	}
	out.Open("fixture surfaces=").I(int64(len(surfaces()))).
		S(" chunks=").I(chunks).
		S(" entities=").I(int64(entities)).
		S(" players=").I(int64(len(all))).End()
}

//go:wasmexport fk_on_event
func onEvent(id, ptr uint32) {
	if id != fkapi.EventOnTick {
		return
	}
	// Relative to the first tick this guest sees, because the input save's tick
	// is the client's and is whatever it is.
	if start == 0 {
		start = fkapi.ReadOnTick(ptr).Tick
	}
	switch fkapi.ReadOnTick(ptr).Tick - start {
	case tickStrip:
		dropSurfaces()
		stripPlayers()
	case tickEntity:
		clearNauvis()
	case tickChunks:
		dropChunks()
	case tickVerify:
		verify()
	case tickSave:
		name := SaveName
		if err := fkapi.Game.ServerSave(&name); err != nil {
			harness.Fatal("server_save", fk.LastError())
			return
		}
		out.Open("server_save issued as ").S(SaveName).End()
	case tickDone:
		// The script polls for this: `server_save` returns before the writer has
		// finished, so the run has to outlive the call by some ticks.
		out.Open("fixture written").End()
	}
}

var start uint64

func main() {}
