# The layout check is gone

`make mod` used to run `test/check-layout.py`, which re-derived from the `fk_api_gen.lua` that `fklua mod` had just emitted **exactly the constants the guest derived by hand**, and failed the build when the two disagreed. The file is **deleted**, along with its Makefile hook, because that list is now empty. It went 25 event offsets plus two nested layouts → six offsets plus `host.go`'s four `create_surface` wire constants → two offsets plus four `defines.direction` values → **nothing**:

| what left | when |
|---|---|
| 19 event offsets and two nested layouts | `gen-bindings` began emitting a struct and a `Read<Event>(ptr)` per event |
| the four build events' offsets | a dictionary field inside a struct started generating, so all 218 events had readers |
| `host.go`'s four `create_surface` constants | `M.call` learned to trim an absent trailing optional; the file is deleted |
| `on_undo_applied` / `on_redo_applied` → `player_index` | `fk.subscribe` gained a **field mask**, so the expensive field is not marshalled and the generated reader is simply used ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 14) |
| the four `defines.direction` values | `gen-bindings` began emitting a `DefinesDirection*()` accessor per define path ([`FKLUA-GAPS.md`](../../FKLUA-GAPS.md) item 11) |

**Deleting it is the right call and not merely a tidy-up, because the last two rows were the ones it was worst at.** The offsets it could check honestly: it re-derived them from the generated table, which is the authority. The four directions it could not — it compared `plan.go`'s literals against the `order` field of the pinned `runtime-api.json`, and a define's `order` is *not* its value; it is a sort key that happens to coincide today. That check was the same guess, marked against itself, and it would have passed on a Factorio where the numbers had moved. The accessor asks the running game by name, so there is nothing left to check and nothing left that could be checked well.

Everything the guest reads — every event field, every nested `BoundingBox` and `TilePosition`, `on_brush_cloned`'s tile array and its stride, and now every define — comes from generated code and moves with the pin by construction.
