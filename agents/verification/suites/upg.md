# `upg` -- `test/assert-upgrade.py`, then M2's own assertions again

A mod upgrade, done properly: the M2 map is created by one guest and loaded by another (version bumped, build stamp changed), so the saved heap is declined and the registry is empty while the world is full of running networks. Measured 2026-08-24 on Factorio 2.1.14, over the re-laid single-edge rigs:

    after the upgrade: 4 surfaces scanned, 156 parts, 21 clusters
    21 networks adopted as they stood, 0 rebuilt

and then **M2's entire assertion set is run over the result** -- 3.998x on `sat4`, 7.994x on `sat8`, exact item conservation across a forced recompile, all of it. A network adopted with the wrong slot, or adopted when it should have been rebuilt, does not balance. The whole rebuild-and-adopt cost ~19 ms on top of the recompile it happened to land inside.

**The `curv` suite is this one's shape used for a different question** -- one guest creates, another loads -- with the addition that its observer lays belts the mod is never told about, so that what the second guest inherits is a world built to a rule it no longer has. See its section below.

**It grew from 9 clusters to 21 when M2 gained the shape and edge-type bands**, and the twelve it gained are the interesting ones: adoption re-derives every edge list from the world through the same `classifyEdges`, so this is also where a P=16 butterfly, a feedback loop closed through the world, and edges made of undergrounds, splitters, loaders and lane splitters are adopted rather than merely compiled. The single-edge re-lay doubled the PARTS it walks -- 77 to 156 -- and moved nothing else: the twenty-one networks it re-derives are the same twenty-one networks, so all twenty-one still adopt.

**THE BUILD STAMP IS WHAT DECLINES THE HEAP, NOT THE VERSION, and `bump_build` moves both because only one of them does the work.** That is `agents/single-edge.md`'s S2 result 6 stated as a property of this suite, and it is red-proven here as of 2026-08-24: leave the `perl` that rewrites `build = "…"` out of `bump_build` so the mod version alone moves, and the saved heap is ADOPTED, `fk_migrate` never fires, `rebuildFromWorld` never runs, and `assert-upgrade.py` fails with *"the guest never rebuilt its registry from the world after the upgrade -- every later placement would have built a SECOND network"*. A leg that bumped only `info.json` would test nothing at all and would say so nowhere.
