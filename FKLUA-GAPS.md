# What building this mod asked of FkLua

BetterBeltBalancer was the first mod built on [FkLua](https://github.com/Techrocket9/FkLua), partly to find out what a real mod runs into. This ledger records each thing the mod needed from FkLua and did not find: what was wanted, what happened instead, and how it was resolved. Nothing was forked or patched locally; every workaround lived in this repository until the fix landed upstream, and every one described here is since deleted.

Items are numbered in the order they were found; FkLua's own notes refer to these numbers, so they are kept even where the item is closed.

| status | items |
| --- | --- |
| Fixed upstream | 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 19, 20, 21, 22, 23, 24 |
| Closed, no change needed | 17 (a budget rather than a defect) |
| Open | 18 |

## 1. `fklua mod` could not carry the data stage

`fklua mod` produced exactly the five generated control-stage files and its directory writer removed its target first, so a mod's data stage (`data.lua`, `prototypes/`, `graphics/`, `locale/`) had nowhere to go and `--zip` could not produce a complete mod. The workaround was a `cp -R` of `mod-data/` over the output after packaging. Fixed upstream: `fklua mod --include DIR`, with `[mod] data` in `fklua.toml` as its default, merges a tree into the package before either writer runs, and a collision with a generated name is an error. `make zip` now produces a complete installable mod.

## 2. `fklua mod` did not read `fklua.toml`

`fklua init` wrote the mod's identity into `fklua.toml`, and `fklua mod` then took every field as a command-line flag and never read the file; there was also no way to set `info.json`'s `dependencies`. The Makefile `sed`-ed the values back out of the manifest into flags. Fixed upstream: identity comes from `[mod]`, `dependencies` reaches `info.json` verbatim, and this mod declares `base >= 2.0.0`.

## 3. `fk.subscribe` had no event filter

Factorio's `script.on_event` takes a filter list the engine applies in C++ before a handler runs. `fk.subscribe(id)` took only the id, so a guest caring about one prototype was entered for every build and mine event on the map and had to read `entity.name` (a host call plus a string crossing) to reject the rest. Fixed upstream: the filter list crosses as a tier-2 value decoded once at subscribe time. This mod subscribes its per-entity events with a three-term list, `{filter = "transport-belt-connectable"}` plus its part and its audit marker by name; enumerating names and types would have taken eleven terms.

## 4. Event payloads had no generated guest type

The bindings emitted an id constant per event and nothing describing its fields, so a guest read them by casting a pointer at a hand-derived byte offset; those offsets move whenever the API pin moves and being wrong is silent. The workaround was a script that re-derived every offset from the emitted `fk_api_gen.lua` on each build and failed the build on a mismatch. Fixed upstream: every event gets a Go struct and a `Read<Event>(ptr)` reader. The script is deleted; its last two offsets went with item 14.

## 5. Bindings did not flatten class inheritance

`LuaEntity` had no `Position()` or `SurfaceIndex()`; both belong to its parent `LuaControl`, and the generator emitted one type per class with no forwarding, so a guest had to cast a handle to `LuaControl{Object: h}`. Fixed upstream with one-line forwarders per inherited member (rather than embedding, which would have broken every `LuaEntity{Object: h}` composite literal here). Every such cast in the guest is gone.

## 6. There was no guest hook for the first tick after a load

Factorio's own `on_load` cannot read `game`, so a rebuild-from-world guest needs a one-shot on the first tick after a load; FkLua had none, which left a guest subscribing to `on_tick` forever to notice a load. Fixed upstream as `fk_after_load`. Not adopted here: a save whose build matches has its heap adopted, a rebuilt guest is told by `fk_migrate` (item 13), and a `registryReady` flag in the guest heap covers any other heap loss. FkLua's notes now also warn that this hook runs on every peer that loads the state, including a joining multiplayer client, so guest state must not be written from it.

## 7. Two commands wrote outside the project

Run inside a mod project, `fklua gen-bindings` wrote its census relative to the executable, rewriting a file in the FkLua checkout, and it ignored `fklua.toml`'s `lang`, generating Rust bindings into a Go-only project. Fixed upstream: the census is written only where the API description lives in the working directory, and both `gen-bindings` and `lock` read `lang` and `api` from the manifest.

## 8. Every host method call was malformed

`fk_abi.lua` called a method as `pcall(f, obj, ...)`, passing the object to a closure Factorio had already bound to it, and forwarded all four argument slots whatever the member declared. `game.get_surface("nauvis")` reached the engine as five arguments: `Arguments count error for '?': Expected 1 argument but 5 were given`. Attribute reads never touched the path, so a guest that only read attributes did not see it. The workaround was a script that patched the two lines in the copied `fk_abi.lua` at package time. Fixed upstream: `pcall(f, ...)` and an arity dispatch on the declared argument count. The script is deleted.

## 9. `LuaGameScript::create_surface` had no Go binding

The member was deferred because its optional `MapGenSettings` argument contains dictionaries, and a mod whose whole architecture is a hidden surface has no way around a missing `create_surface`. The workaround wrote the call out by hand (member id, argument block size, two field offsets), guarded by the same layout-check script as item 4. Fixed upstream: an optional argument the Go layer cannot express is omitted rather than deferring the member. The hand-written call outlived the fix briefly because of item 16 and is gone.

## 10. Marshalling allocated on every host call and never gave it back

Every tier-2 value crossing outwards and every array or string coming back was a fresh Go allocation, permanent under `-gc=leaking`: about 180 B of guest heap per host call, at roughly 350 calls per network compile. Under `--persist=packed` it was also a time cost, because the dirty watermark was a byte range that one call stretched across the heap: a 4x4 recompile was 41 ms packed against 4.2 ms in table mode. Fixed upstream in two steps. An arena reset at the bracket every binding opens took the per-call cost from 128 B to 0 (measured here: the table-mode save delta on a 300-compile map went from 1,373,514 B to 169,411 B and the packed 4x4 recompile from 41 ms to 7.4 ms), and a dirty page set replaced the byte range (a 200-compile create went from 447 s to 45.8 s packed, against 31.0 s table). This mod ships packed: at 200 balancers the save is 3.6 MB against 49.4 MB in table mode and the load 8.2 s against 21.6 s.

## 11. `defines` had no generated guest constants

The bindings emitted nothing for the 60 `defines` groups, so `defines.direction.east` was a `4` a guest author looked up and wrote down. This mod kept four cardinal directions as constants checked against the pinned description, which was itself wrong: a define's `order` field is a sort key, not its value, and the two merely coincide in 2.0.77. Fixed upstream: an accessor per define resolves by name at load through an `fk.define(id)` import and caches on first use, pruned by the same constant scan as members. This mod reads four of the 1,137.

## 12. There was no way to batch work across the events of one tick

A blueprint pasted as real entities is one build event per entity, and without a way to defer, each event was a separate opportunity to recompile a cluster: a 12-entity, 2-part balancer pasted in one tick rebuilt twice. The hook this entry asked for, at the end of the outermost dispatch, would have batched nothing, because a paste is that many separate dispatches. Fixed upstream as `fk.defer()`, a one-shot `on_tick` registered on demand and torn down from inside its own flush, so an idle guest still pays nothing per tick. Here the 12-entity paste is one build and the 200-balancer bench save compiles 200 networks instead of 800.

## 13. `fk_migrate` handed a rebuilt guest another build's linear memory

On a build-stamp mismatch the runtime adopted the saved heap wholesale before calling `fk_migrate`, and linear memory is not only the heap but `.data` and `.rodata` too, so a rebuilt guest's compiled-in addresses pointed at the previous build's bytes and the first line of the hook was already undefined. The workaround was to export no hook, take the clean reset, and rebuild the registry from the world behind a `registryReady` flag. Fixed upstream as proposed: `fk_migrate(old_version)` is a notification on a fresh heap and `fk_migrate_adopt` is a separate opt-in that really hands the bytes over. This mod exports `fk_migrate`; the flag stays as the fallback for other heap losses.

## 14. Event fields were marshalled eagerly and in full

Every field of an event was encoded before dispatch. `on_undo_applied` carries `actions`, an unbounded array of blueprint entities, and this mod reads one `uint32` from that event, so an undo of a thousand-entity blueprint deep-copied a thousand entities across the boundary for nothing. Fixed upstream: `fk.subscribe` takes a field mask over optional and container fields, with a generated `Skip<Event><Field>` constant per maskable field. Upstream measured this subscription at 20 actions: 725 µs per dispatch unmasked, 1.9 µs masked; at 200 actions, 7.49 ms against 2.7 µs. This mod masks `actions` on undo and redo.

## 15. Dictionary-returning members were deferred, so surfaces could not be enumerated

`game.surfaces` is a dictionary, and there is no other way to ask what surfaces exist. The workaround probed `game.get_surface(1)`, `(2)`, and so on until 64 consecutive misses: about 65 host calls once per session and a guess about how sparse indices can get. Fixed upstream: a dictionary return keyed by a tier-2 value binds as an ordered pair slice, and `LuaGameScript.Surfaces()` exists. The probe is deleted. Because the host's iteration order is unspecified and surface order here decides node ids and slot claims, this mod sorts the slice by index on arrival.

## 16. An absent trailing optional argument was forwarded as an explicit nil

The fix for item 8 forwarded the declared arity, and a separate fix made an absent optional decode to nil rather than its zero value. Together, a member whose trailing optional was absent reached the engine as an explicit `nil`, and Factorio counts what arrives: `game.create_surface(name)` failed with `bad argument #2 of 3 to '?' (table expected, got nil)` on the first compile of the first balancer. Fixed upstream: the forwarded arity is trimmed to the last argument actually present. The 88-line hand-written `create_surface` is deleted, and this mod hand-writes no host call at all.

## 17. A guest heap held as a Lua word table is a garbage-collection tail

The first head-to-head measured a 27.8 ms worst tick on an idle 200-balancer map with this mod's `scriptUpdate` at exactly zero and 16.75 ms of a 17 ms tick in `luaGarbageIncremental`: Lua's collector walking the guest's linear memory, which is a Lua table in every persistence mode. Upstream's answer was documentation and visibility: a "guest heap budget" of 0.2 ms of worst tick per MiB of linear memory (flat from 8 MiB to 128) and a log line reporting the memory once per doubling. That line showed a 64 MiB heap, which turned out to be this mod's own `[BBB]` log lines, built with `+` and `strconv` under a leaking allocator. Rebuilt on one reusable buffer with the same line formats: idle worst tick 18.05 ms to 2.26 ms against a 1.42 ms control at 200 balancers, and 49.7 ms to 5.08 ms (control 3.10) at 500. What remains is a budget: about 1.3 KB of permanent heap per compile in generated binding return values (a 4x4 teardown-and-rebuild 1,517 B, a 2-to-2 681 B, a belt built anywhere 32 B), which no downstream code can remove. That residual is why this mod ships `--gc=collected`: through 3,400 4x4 rebuilds the leaking arm's growth ticks were 48.7 ms at 2 to 4 MiB, 120.3 at 4 to 8, 226.1 at 8 to 16 and 782.4 ms at 16 to 32, where the collected arm ended on 0.52 MiB with a worst tick of 71 ms.

## 18. `fklua mod` packages a stale guest against fresh bindings without complaint

When the generator bound more members and renumbered the member-id space, `make mod` packaged a wasm built against the old ids with the new `fk_api_gen.lua`; every id resolved to a different member and the first symptom was `LuaGameScript doesn't contain key valid` at the first event of a real game. `fklua gen-bindings --check` and `fklua lock --check` catch the drift, but only when `make check` runs; `fklua mod` has the lock hash available and says nothing. Open. FkLua now stamps the API pin into the generated bindings and refuses to package a guest built at a different pin, which catches the cross-version form; a stale wasm at the same pin is still packaged silently.

## 19. A collector knob installed from `init()` was silently discarded

`fkgc.SetThreshold` called from a guest's package initialiser was overwritten by the collector's own initialisation, which assigned both knobs their defaults unconditionally, in both guest languages. Found by a Rust port in [fklua-ports-samples](https://github.com/Techrocket9/fklua-ports-samples) that asked for 128 KiB and ran at 256; invisible here because the value this mod asked for was the default already in place. Fixed upstream by latching a non-zero value in `initialize()`. No workaround was ever needed here.

## 20. An optional attribute generated non-optional

666 optional attributes generated a non-optional return, and an absent one arrived as `ERR_NO_MEMBER`, indistinguishable from a member the running Factorio does not have. Fixed upstream in both the generator and the host. This mod reads one optional attribute, `LuaEntity.linked_belt_neighbour`, when adopting a standing network; before the fix an unconnected linked belt arrived as an error and the adopt fell back to a rebuild, the right answer for the wrong reason. It arrives as nil now.

## 21. A guest whose globals exceeded one step's budget could never terminate a mark

The collector's termination attempt re-scanned the guest's globals wholesale and charged the scan against the same per-step budget as everything else, so once the globals cost more than one step the attempt could never complete: every step spent its allowance on the scan and deferred, with no error and correct results. Measured in a real Factorio on saves whose heaps were byte-identical: adding two package-level variables took the root scan from 4,070 to 4,278 words and the post-load collection from 71 paced steps and 67 ms of script to 982 steps and 1,041 ms. The workaround was a doubled budget plus a guest-side check on the root count. Fixed upstream: the effective budget is floored at the root-scan cost, the scan is reserved out of each step rather than charged on top, and the collector logs one line the first time the floor binds. Because a step now spends its whole budget in total, this mod's setting was re-derived to `4096 + reserve`; its own root check is deleted.

## 22. A guest could not observe the mod set changing

Factorio raises `script.on_configuration_changed` when a neighbouring mod is added, removed or moved to another version, and the glue registered exactly one thing on it: its own rebuild check, which returns immediately unless the guest's build stamp moved. A mod set changing does not move it, so the event that reports a neighbour being uninstalled arrived, was consumed, and told the guest nothing. This mod needed it for a once-per-save conversion of the balancers an uninstalled predecessor left behind; without it the best available trigger was the first event of the session, which converts late and on a tick nobody chose. Fixed upstream: `fk_on_configuration_changed()` is dispatched whenever Factorio raises the event, after the rebuild check, with no arguments. It is replicated, runs on the peer that loaded before the first tick, and may therefore write guest state, unlike the peer-local first-tick-after-a-load hook. This mod exports it and its migration converts at load.

## 23. A custom table's index-assign has no expression, so a mod cannot write its own runtime setting

`settings.global["name"] = {value = true}` is the only way a script changes a mod's own runtime-global setting, and it cannot be expressed through the bindings. Two layers: Factorio's `runtime-api.json` declares `LuaCustomTable`'s `index` operator with a `read_type` and no write side, and `LuaSettings::global` likewise, with the write capability appearing only in the attribute's prose description, so a generator that mirrors the description correctly emits no setter; and the ABI's member kinds have no shape for `obj[k] = v`, because the attribute-set kind takes its member name from the generation-time member table rather than as an argument. Measured on Factorio 2.1.14: the write works from any script context except an `on_init` dispatch, persists through save and load, stays per save rather than being written back to `mod-settings.dat`, and raises `on_runtime_mod_setting_changed` synchronously with no `player_index` in the payload; writing a key that is not defined raises an error. The ask is an index-assign member kind, key and value both arguments, mirroring the index read, plus a generator affordance to emit it for writable custom tables, which needs a short allowlist because the description declares no write side to key off. This mod needs it to flip its own multi-edge setting when adopting a save that predates the setting. Fixed upstream, in the shape asked for: the member kind takes key and value as arguments, and the generator emits it from an allowlist over the five custom tables the description says in prose are writable, so writing any other one comes back as a call failure carrying the engine's own "LuaCustomTable is read only" rather than as an unwind. Writing a key that is not a defined setting comes back the same way, which is why this mod gates the write on a marker prototype that only exists on the Factorio versions where the setting is defined. Its grandfather pass writes the setting through this member on the one load that adopts a save built before the setting existed.

## 24. A mod pinning a non-default API version is stranded when the census format moves

`fklua gen-bindings --check` compares the census it would take against the one committed beside the API description, and the census is written only from the checkout that owns the description, which is correct. But the only thing that regenerates a census is `gen-bindings` running at that checkout's default pin, so the censuses of every other committed description go stale the moment the generator gains a row, and a mod pinning one of those versions then fails `--check` with no command anywhere that can repair it: regenerating from the mod project is refused by design, and regenerating from the FkLua checkout would rewrite its own committed bindings to the wrong version. Hit here moving this mod's pin to a committed 2.1.14 description right after the index-assign feature added a census row; the pin move had to be reverted. Fixed upstream: `gen-bindings` in the FkLua checkout now refreshes the census of every description the checkout owns whatever pin was invoked, so staleness is structurally impossible, and a mod project's `--check` no longer fails on the compiler's own census at all, since nothing a mod builds reads it; a stale one prints a notice naming the checkout and the command. This mod's pin move to 2.1.14 went through on exactly that sequence with nothing hand-edited.

## Smaller notes

- Upstream now treats a runtime log line as API surface with a stable opening clause, because downstream tests match on them; this mod asserts only on its own `[BBB]` lines. Its guest notes also point at Factorio's built-in filter categories (see item 3).
- The generated define accessors are eliminated from the wasm's code section but not from its DWARF, so `dist/bbb.wasm` roughly doubled (363,051 to 713,899 bytes) while the code section grew 1.8 KB and the packaged zip 1.7%; the wasm's file size is not a measure of guest size. Calling an accessor from a guest's own `init()` is legal, since the define table is resolved before `_initialize` runs.
