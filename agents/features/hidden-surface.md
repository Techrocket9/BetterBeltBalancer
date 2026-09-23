# The surface in the remote view — the first bug report from the portal

**With Space Age installed, `bbb-hidden` sat in every player's remote-view surface list.** Reported 2026-08-31 by CodeW0lf as the repository's first issue, with the mechanism already tested: surface visibility is a **PER-FORCE flag** whose default is VISIBLE, this mod never set it, and so the one surface no player must ever look at was offered to all of them in the surface switcher. Every suite was green while it happened and always would have been, for the same reason they were green over the tan streak and the flying text at the box centre: **no assertion anywhere reads a GUI**. Shipped as 0.3.2.

**The mechanism is the reporter's and it is the only one there is.** `LuaForce::set_surface_hidden(surface, hidden)` exists on both pinned engines (2.0.77 and 2.1.17, checked in both runtime descriptions before a line was written) and **neither has a surface-level flag** — there is nothing on `LuaSurface` or in `create_surface`'s settings that could say it once. So the fix is "hide for every force, at every moment a force and the surface can first meet", and the reporter's three trigger points translate into this mod's architecture as:

| the moment | where the hide runs |
|---|---|
| the surface is **CREATED**, or recovered by name after a clear | both discovery paths of `hiddenSurface()` call `hideFromAllForces` (compile.go) — which also covers the recreation after somebody deletes the hidden surface, since that path recreates lazily through the same function |
| an **EXISTING save** arrives on a fresh heap | `rebuildFromWorld`, immediately after `collectSurfaces` set `hiddenIdx` — which is the load that delivers this fix to every save built before it, because a new build declines the saved heap. The issue asked for `on_configuration_changed`; this is that moment in a guest whose registry rebuild already owns it. Zero extra host calls to find the surface: the object is in hand from the walk it was found by |
| a force is **CREATED** after the surface exists | `on_force_created`, the twenty-fourth event subscription. `ensureRegistry` precedes every dispatch, so `hiddenIdx` is authoritative by handler time: one integer compare in a save with no hidden surface, three host calls in one with. Deliberately NOT `hiddenSurface()`, which would CREATE the surface because somebody made a force |

**Two of the issue's details are deliberately not taken.** The `get_surface_hidden`-before-set pre-check is dropped — the set is idempotent, so the read would double the host calls to save the engine a no-op — and the `on_init` hook has nothing to do here, because on a fresh save the surface does not exist yet and the creation path is the one that will meet it. Nothing lands on a per-compile path: `hiddenSurface()`'s fast path returns before reaching any of it, so a session pays at discovery and never again.

**Verified on the genuinely unfixed save, which the `mig21` fixtures are by construction.** Both were written by a 2.0.77 guest that predates the fix and can never be regenerated, so they are the one unfakeable specimen of the broken state. The observer reads `get_surface_hidden` back for every force at `cfg` — before the mod's migration has run, which only this suite can sample at all — and at `final`: **every force reads `hidden=false` at `cfg` and `hidden=true` at `final`**, on both fixtures (three forces on m2, four on edge). The `cfg` half is the anti-vacuity control: a probe that cannot see the broken state proves nothing about the fixed one, and `assert-mig21.py` fails on a fixture that arrives pre-hidden. The assertion is engine-independent and runs on both arms — hiding is about who can SEE the surface, not what stands on it — and was measured on the 2.0 arm (the binary of the day); the 2.1 arm's expectations are identical.

**And on all three trigger paths, in `m3`.** Three per-force readbacks: `initial` (the surface created inside `on_init`'s audit flush, all four forces already standing), `recreated` (the surface built again after `phaseDeleteHidden` took it down at tick 450), and `late` — a fifth force, `bbb-late`, created at tick 500, AFTER the surface existed, so no hide-at-creation can have covered it and the `on_force_created` handler is the only thing that can. All thirteen lines read `hidden=true`, and not one pre-existing number in the suite moved: the twelve rig rates, the audits and the final world tuple are byte-for-byte the recorded ones.

**Red-proven twice, and the two proofs fire different assertion sets.** The whole fix stubbed out (`hideFromAllForces` and `onForceCreated` both returning immediately), rebuilt, and re-run:

| suite | what fired |
|---|---|
| `m3` | **exactly the three new assertions**, naming every exposed force at every tag — `the hidden surface is VISIBLE to force(s) bbb-late, bbb-other, enemy, neutral, player at tag=late` — with all twelve rig rates, both spreads and the final audit identical to the green run |
| `mig21` | **exactly one**: `VISIBLE to force(s) [1, 2, 3] after the rebuild`. The `cfg` pre-check still passes, as it must — the unfixed guest leaves the fixture's pre-state exactly as the fixed one finds it |

**What it costs.** Package built 2026-08-31, trunk pin (2.1.17), shipped config, measured either side of the change in one session:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_*.zip` | 579,419 B | **582,249 B** | +0.49% |
| `fk_module.lua` | 3,110,020 B | **3,146,816 B** | +1.18% |
| `dist/bbb.wasm` | 1,290,222 B | 1,297,571 B | |
| members bound into the mod | 54 | **56** | `LuaForce.set_surface_hidden` and `LuaGameScript.forces`; no re-pin, both were in the generated bindings as they stood |
| events subscribed | 23 | **24** | `on_force_created` |

(The before-zip is not the 560,991 B the 2.1.17 verification recorded, and the difference is not this change's: trunk's packaged mod has carried **all 225 event descriptors** since the last bindings regeneration — `fklua mod` prints *"an event id was not a compile-time constant"* on the unmodified tree — where the 2.0.77 pin still prunes to the events actually subscribed. That regression is filed separately; the 2.0 arm's package prints `24 events subscribed, of 219`, which is also what says the new subscription kept its literal-constant form.)

**Nothing on any hot path, and on this pass it is structural AND measured.** Every new call site is once-per-session (discovery, the rebuild) or per-force-creation (an event rarer than a force merge); the `mar` suite's leaking-arm slopes are the gate as always, and the suite never creates a force or rediscovers the surface inside its measured phase.

**What is behind the graphical wall is the pixel itself** — whether the flag removes the row from the remote-view GUI — and that half is the reporter's own tested claim, which is what the issue said in as many words. The headless half, the flag actually being set for every force at every moment it can matter, is the two suites above.

**THE VERIFICATION RAN ON THE 2.0 ARM AND THE TRUNK SUITES ARE OWED**, because the installed Factorio moved back to 2.0.77 between sessions (the Steam betas checkbox, exactly as the Environment section warns). The whole estate is green on 2.0.77 in BOTH `-gc` arms — the fix applied to the `release/2.0` recut in a sibling worktree, one invocation each, `flip` running and `mig21`/`mig` inverted — with the `mar` slopes **identical to the byte** (1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per primitive over 3.92 MiB, 1,136 B of calibration at 0.0% spread) and both red proofs taken there. The trunk (2.1.17) tree builds, `make check` is green and the package gates clean; its fourteen suites need the binary back on 2.1 and should be the first thing a 2.1 session runs. The code is engine-independent — `set_surface_hidden` is the same member at both pins and nothing here branches on the engine — which is what makes the 2.0 measurement evidence rather than hope. The `release/2.0` arm needs its own recut (0.2.2) to ship the fix there.
