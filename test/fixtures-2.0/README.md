# fixtures-2.0: saves built by Factorio 2.0.77, for the 2.1 migration suite

Four suite saves copied verbatim from `test/tmp/` on 2026-08-24, from the last
full suite run on 2.0.77 (2026-08-22, before the installed Factorio moved to
2.1.14). **They cannot be regenerated without a 2.0.x binary**: a 2.1 Factorio
refuses to build multi-edge balancers at the prototype level, which is why these
are committed rather than staged. Each save's `create.log` is kept beside it as
the record of what the world contains.

| fixture | world |
|---|---|
| `m2-2.0.77.zip` | the M2 suite: 21 rigs, 77 parts, saturated, most parts carrying TWO interfaces (sat4 is 4 parts and 8 belts) |
| `edge-2.0.77.zip` | the edge suite: 15 clusters over 95 parts, including `lim` (64 inputs and 1 output over 32 parts, 2 to 3 interfaces per tile) and `brdg` |
| `m3-2.0.77.zip` | the M3 lifecycle suite's world |
| `qual-2.0.77.zip` | every part at uncommon quality |

## What they are for

The `mig21` suite, which is the migration half of the 2.1 port. It loads the m2
and edge fixtures under 2.1 with the current mod and asserts what the mod does to
a world built under a rule the engine no longer permits. It is the only suite
with no `--create` phase: the fixture is phase one.

`test/run.sh mig21` runs it, and it is in the default set.

## What the load does before any script runs

Measured 2026-08-24 on 2.1.14. Loading one of these does not crash. The engine
**silently deletes all but one belt-connectable per tile**, with no log line of
any kind, and leaves the hidden networks fully intact: the m2 save comes up with
77 interfaces over 77 part tiles where it was built with about 140, and the edge
save with 95 over 95. What the deleted interfaces were holding goes with them, at
most eight items each, and no mod can recover it.

Left alone, the crippled networks then deliver a trickle, and which port survives
per tile is a lottery: on the m2 save a saturated 4-in 4-out rig delivered 14
items against a bare belt's 1294, and an asymmetric rig ran two outputs at full
rate with three dead. The migration's job is to turn that into clusters that are
refused and explained, with everything the hidden half was still holding
recovered and put on the ground beside each one.

## Loading one by hand

`test/run.sh`'s `stage_fixture` does all of this; the list is here for anyone
reproducing it outside the suite.

- The matching test mod from `test/mods/` has to be staged too, for its DATA
  STAGE: these worlds are full of its loaders and lane splitters, and Factorio
  deletes every entity whose prototype went with a removed mod, at load, before
  any script runs.
- Its `factorio_version` has to be bumped to 2.1, because 2.1 refuses a mod whose
  `info.json` says 2.0 before it places an entity.
- Its control stage should be emptied. Its schedule would drive rig edits and
  measurements from tick 0 against a world its `on_init` never set up, and on the
  m2 save it raises outright.
