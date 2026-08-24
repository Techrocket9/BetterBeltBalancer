# fixtures-2.0 — saves built by Factorio 2.0.77, for the 2.1 migration suite

Four suite saves copied verbatim from `test/tmp/` on 2026-08-24, from the last
full suite run on 2.0.77 (2026-08-22 21:36, before the installed Factorio moved
to 2.1.14). **They cannot be regenerated without a 2.0.x binary** — a 2.1
Factorio refuses to build multi-edge balancers at the prototype level — which
is why they are committed rather than staged. Each save's `create.log` is kept
beside it as the record of what the world contains.

| fixture | world |
|---|---|
| `m2-2.0.77.zip` | the M2 suite: 21 rigs, 77 parts, saturated, most parts carrying TWO interfaces (sat4 is 4 parts / 8 belts) |
| `edge-2.0.77.zip` | the edge suite: 15 clusters / 95 parts, including `lim` (64 inputs + 1 output over 32 parts — 2 to 3 interfaces per tile) and `brdg` |
| `m3-2.0.77.zip` | the M3 lifecycle suite's world |
| `qual-2.0.77.zip` | every part uncommon quality |

What they are for: the migration legs of the 2.1 port
([`agents/single-edge.md`](../../agents/single-edge.md)). Measured 2026-08-24 on
2.1.14: loading these saves does not crash — the engine **silently deletes all
but one belt-connectable per tile** (m2: 73 interfaces survive of ~140 built;
edge: 95 of 160+; no log line), leaves the hidden networks fully intact, and
the crippled networks then deliver a trickle (sat4: 14 items against a
control's 1294). The migration machinery's job is to turn that world into
refused-and-explained clusters with their hidden items recovered.

Loading one: the matching test mod is in `test/mods/`, the loading mod set must
bump `factorio_version` to 2.1 in the staged copies, and 2.1 removed
`LuaGameScript::create_profiler` — the test mods' calls become
`helpers.create_profiler` on the way past.
