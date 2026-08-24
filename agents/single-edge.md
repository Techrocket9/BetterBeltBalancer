# single-edge.md — the 2.1 port: one belt per part

Status: **DESIGN, nothing implemented.** Drafted 2026-08-24 from the 2026-08-23
investigation and boskid's answer to the interface request
(forums t=135830). Read CLAUDE.md's migration and over-limit sections first;
this design reuses both wholesale.

## Why, in three sentences

Factorio 2.1 fixed the collision-mask equals-compare that let `bbb-linked-belt`
ship `not_colliding_with_itself` with default belt layers, and the validator now
demands every belt-connectable collide pairwise with every other one and with
itself — probed exhaustively on 2.1.14, no mask design passes and no runtime
bypass exists (`create_entity` nils, `teleport` returns false). boskid's answer
explains the invariant the check protects: **belt-to-belt connections are not
saved — they are re-derived at load, and one belt-connectable per tile is what
makes that re-derivation unambiguous**; two overlapping linked belts ever
rotated onto the same side "would explode", and the request is open but
explicitly no-priority. So the port is a RULE change, not an interface
redesign: at most **one belt per balancer part**, because one part tile may
carry at most one interface linked belt — and a single linked belt over the
part is verified legal on 2.1.14 (belt-over-non-belt placement is unchanged).

Two things worth internalizing from his answer before touching anything:

- **S1's "never two same-direction inputs on one tile" was the observed shadow
  of exactly this ambiguity.** The 2.0 mod's discipline dodged the dangerous
  configurations; the engine cannot rely on discipline, hence the validator.
- **Stacked linked belts standing in a 2.1 world are a latent engine risk on
  every load**, not merely unsupported. That decides the migration design
  below: they must be removed at first opportunity, never left standing under
  a drift flag the way an over-limit refusal is.

## The rule

A cluster tile may carry **at most one edge**. Everything else about edge
classification is unchanged: same `classifySide`, same six types, same
direction reading. The per-tile count falls out of the `classifyEdges` walk
that already visits every tile — no allocation, no extra host call, which is
the gate the `mar` suite enforces on anything touching that path.

Consequences, stated rather than discovered later:

- A 4-in/4-out balancer is **eight parts** (say a 2×4: four input parts, four
  output parts), not four. The smallest balancer is two parts. Footprint
  roughly doubles; every rig in every suite is built in the old idiom and gets
  rebuilt (see the test estate section).
- `P = next_pow2(max(N, M))` and `plan.MaxPorts = 64` are untouched. One nice
  simplification: under single-edge a cluster of C parts has at most C edges,
  so `spareOverLimitMerges`' conservative bound tightens from `4*csize <=
  MaxPorts` to `csize <= MaxPorts` — a 64-part cluster is provably safe, up
  from 16.
- The planner, the hidden network, carry, skin, the registry, the audit: **no
  change.** The hidden surface never stacked anything (entities sit adjacent),
  and one interface per part tile is the case the compiler already handles.

**Considered and rejected: making the part itself the belt-connectable** (the
part-as-linked-belt idea, which would delete the interface entity for the
single-edge case). It costs the M5 skin — `graphics_variation` is
`simple-entity-with-owner`'s, and a linked belt has direction-indexed structure
sprites, not a variation set — plus the whole fast-replace and mining surface,
for no capacity the current shape lacks. The current architecture needs no
entity redesign for single-edge; keep it.

## The mode: a runtime-global setting, and how the guest learns it

`bbb-multi-edge-parts`, bool, **runtime-global**, default **false** — and
runtime-global rather than startup is forced, not a taste: the grandfather
pass below has the MOD flip the setting, and **a script can write
`settings.global` but can never write a startup setting.** It also buys two
things startup could not: the player flips it mid-save with no restart, and
the flip arrives as an ordinary replicated event
(`on_runtime_mod_setting_changed`) instead of a whole
`on_configuration_changed` load cycle.

What used to force startup — the collision flag is a data-stage decision — is
dissolved by splitting the two questions the old design conflated:

| question | channel |
|---|---|
| **CAN the engine stack** (a fact about the Factorio version) | the data stage emits `not_colliding_with_itself` **unconditionally on 2.0.x, never on 2.1.x** (branch on the `mods` global; the version guard is what keeps a mispackaged zip from bricking a 2.1 load), and defines the marker prototype **`bbb-can-stack`** in the same branch — the `bbb-legacy-stub` idiom, a point query, no new binding, and the guest's belief cannot drift from the prototype's actual capability because they are one `if` |
| **MAY the compiler build stacks** (per-save policy) | the runtime-global setting, mirrored into one guest heap byte |

The effective rule is the AND of the two: on 2.1 the marker is absent and the
setting is not even defined (settings.lua branches on `mods["base"]`; fallback
`hidden = true` if the settings stage cannot see `mods` — S2 probe); on 2.0 a
false setting means the compiler refuses multi-edge exactly as 2.1 does, while
the prototype capability sits unused. A 2.0 save with the setting off is
therefore bit-compatible with a fresh single-edge world, which is the point:
it is the save that upgrades to 2.1 losing nothing.

**The heap byte is the reconciliation anchor and the re-entrancy guard in
one.** It records the mode the registry was last reconciled under; the
`on_runtime_mod_setting_changed` handler compares the setting against it and
acts only on a real change (flip OFF → the sweep below; flip ON → requeue
every refused cluster, whose fingerprints never matched, so the next flush
compiles them). The grandfather pass writes the byte before it writes the
setting, so its own write arrives at a handler that sees agreement and does
nothing — no self-write flag needed. Both sides are deterministic in
lockstep: the write happens in replicated script, the event fires on every
client identically.

Reading and writing `settings.global` from the guest, and subscribing to
`on_runtime_mod_setting_changed`, are binding-surface questions FkLua has not
been asked before — S2 probes, and candidate FKLUA-GAPS items if the
generated bindings do not cover LuaSettings' dictionary write.

Locale names the setting plainly: "Allow multiple belts per balancer part
(Factorio 2.0 only)", with a description saying why it is off by default
(forward compatibility with 2.1, where these balancers stop working).

## Refusal semantics for edits — the sixty-fifth belt, generalized

A second belt laid against a part that already has one is refused **exactly**
the way the sixty-fifth belt is (limit.go): the check runs in `compile()`
before any teardown, so the standing network keeps running; the player gets
flying text at the refused piece in cannot-build red, the sound, and the belt
back in their inventory via the post-`endCarry` revert; robots and scripts get
`force.print`; the feedback gate fires once per distinct edge state; the audit
reports `drift=1 unbuilt=0` while the refusal stands. New locale key
(`single-edge-refused`, worded as the rule: "each balancer part connects to
one belt"), same machinery, same wake-race discipline (a refusal issued from
inside `rebuildFromWorld` logs and requeues but never speaks).

The differences from the over-limit refusal, each small:

- **The predicate**: `overLimitShape(edges)` grows a sibling — any tile with
  ≥2 edges when the marker is absent. Both feed the same refusal path;
  `plan.Build` needs no backstop change (a multi-edge cluster is not a planner
  input problem, so the backstop for it lives beside `overLimitShape`'s).
- **A rotation can trigger it** (a belt turned to face a part that already has
  an edge). Same as over-limit: no revert exists for a rotation — the network
  stands, the player is told, drift=1 until they turn it back. Nothing new.
- **The merge pre-pass** (`spareOverLimitMerges`) must also spare a merge that
  would be refused for multi-edge, and the cheap classification-free bound
  does not exist for this rule. The saving theorem: predecessors standing
  compiled were valid single-edge, and adding a part changes no existing
  tile's adjacencies, so **the only tile that can newly violate is the
  bridging part's own** — classify just that tile (≤4 point queries, merges
  only, which are rare). Spare conservatively: also spare when either
  predecessor is already in the refused map. Being wrong costs the old
  behavior (predecessor teardowns for a refusal), never a demolished network.

Fingerprints do not encode the mode and do not need to: a refused cluster
stores no matching fingerprint, so flipping the setting ON simply lets the next
audit or event compile what was refused; flipping it OFF is the sweep below.

## Updating from the first GA release — the grandfather pass

**The first GA release has no setting and multi-edge always on; its players'
bases must survive the update that introduces the rule.** The strategy, as
specified: on the first load of the updated mod, a save with **no** invalid
(multi-edge) balancers ends with the setting **disabled**; a save that has
them keeps multi-edge **enabled** and gets a warning in chat explaining the
situation.

One translation from that spec to what the engine permits, stated so it can be
vetoed: a newly introduced setting arrives at its DEFAULT in an updating save
— there is no "leave it enabled", because the old version had no setting to be
enabled. So the implementable form inverts the flips and lands on the same end
states: the default is false, a clean save simply stays there, and a dirty
save is flipped **up** to true by the pass. (This is also why the setting is
runtime-global: the flip is a script write.)

The mechanics, and almost all of it is machinery that already exists:

- **When**: the update is a version bump, so the heap is declined and
  `fk_migrate` → `rebuildFromWorld` runs. The rebuild classifies every
  cluster's edges anyway; "does any tile carry ≥2 edges" is a fold over that
  pass, zero extra host calls.
- **The decision**: any multi-edge cluster found on 2.0 → write the heap byte,
  then `settings.global["bbb-multi-edge-parts"] = true`, then say so — once,
  from the informed flush, never from inside the rebuild (the wake-race
  principle). No multi-edge clusters → nothing happens; the false default
  stands, and the save is single-edge from then on.
- **The warning** (`force.print`, to each force owning an affected cluster):
  the mod kept multi-edge enabled for this save because N balancers use it;
  new saves default to one belt per part; **these balancers will stop working
  on Factorio 2.1** — with the settings-menu path for turning it off after
  rebuilding. Said once at grandfather time; while the setting stays true the
  mod stays quiet about it (the once-per-state gate idiom, not a nag on every
  load).
- **The latch is first-load-only.** A player who later rebuilds everything
  single-edge is not auto-flipped down — a silent downgrade under someone
  relying on multi-edge is a trap — they flip it off themselves, which is a
  Map setting and needs no restart; the flip-off runs the sweep and would
  find nothing.
- **The same update arriving on 2.1 directly** (the player skips the 2.0
  release and upgrades Factorio and the mod together): grandfathering is not
  on offer — the marker is absent — so the pass detects the same multi-edge
  clusters (edge classification reads the player's BELTS, which survive the
  engine's interface pruning) and the migration machinery below speaks
  instead: refusal, spill, GPS pings. One scan, two outcomes, chosen by the
  capability marker.

## Migration: a 2.0 multi-edge save arriving where multi-edge is not allowed

The case the user named, and the design's hardest part. Two triggers, one
mechanism:

- **2.0 → 2.1 upgrade** (or any mod upgrade landing where the marker is
  absent): fresh heap, `rebuildFromWorld` → the **adoption** path walks each
  cluster's standing interfaces already (`inspectNetwork`); it now refuses to
  adopt any network with two interfaces on one tile.
- **Setting flipped OFF on 2.0, same build**: heap survives, no rebuild;
  `on_runtime_mod_setting_changed` fires (a Map setting, no restart), the
  mode byte mismatches, and `sweepStackedInterfaces` walks the registry's
  networks looking for the same thing.

**What happens to an offending network: mandatory teardown, then refusal.**
This is deliberately NOT the over-limit standing-state idiom. The two triggers
arrive in different worlds, measured 2026-08-24 (see the S2 results below):

- **On the 2.1 load the engine has already half-done it, silently.** Loading a
  2.0 save under 2.1.14 does not crash: the engine **deletes all but one
  belt-connectable per tile, with no log line**, and leaves the hidden network
  fully intact — so the guest's rebuild wakes into clusters whose standing
  networks are missing most of their interfaces. Adoption compares the
  interface set against the re-derived edge list, mismatches, and falls back
  to a rebuild — which is exactly the path wanted: the teardown recovers
  everything still in the hidden network, and the compile then refuses the
  multi-edge cluster. What is unrecoverable is whatever the DELETED interfaces
  were holding when the engine removed them — at most 8 items per interface
  (two lanes of four positions), engine-caused, ours to document rather than
  to fix.
- **On a 2.0 setting flip the stacked interfaces genuinely still stand**, and
  the mandatory teardown is what takes them down before they can reach a 2.1
  load through a later Factorio upgrade.

The ordering carve-out serves both: when the STANDING network is itself
invalid (stacked or engine-pruned interfaces), tear down first, then refuse
the compile. The items follow existing carry semantics with no new code — the
teardown opens a pool, the refused compile claims nothing, and `closePool`
spills beside the cluster, which is the mod's removal behavior since
"A recompile is not a removal".

**What the player is told.** One `force.print` per affected force from the
informed flush (never from inside the rebuild — the wake-race principle),
summarizing: N balancers built under Factorio 2.0's multi-edge rule need
rebuilding, each part now serves one belt — followed by a clickable
`[gps=x,y,surface]` ping per affected cluster so a big base is a checklist,
not a scavenger hunt. Their drained items are on the ground beside each one.
Every affected cluster also logs, and the audit reports it as `drift=1
unbuilt=0` (edge list beyond what the mode can build) until the player
rearranges the belts.

**Considered and rejected: partial service** — compiling a degraded network
that keeps one deterministically-chosen edge per tile so items keep flowing.
Rejected because any pick is functionally arbitrary: a 4-part 4×4 keeps four
of its eight belts and which four decides whether it deadlocks, starves, or
silently delivers wrong ratios — and this mod's core promise is exact balance.
A stopped balancer that says why beats a running one that lies. Also rejected:
mining the player's excess belts to make the cluster valid (destroying player
property), and auto-placing chests for the drained items (inventing entities
the player did not build).

The gating unknown this section was drafted around — does 2.1 survive loading
a save with stacked interfaces at all — is ANSWERED and the answer is the
middle case nobody proposed: no crash, no intact standing, but silent per-tile
pruning. The measurements are in the S2 section below, and the fixture saves
they ran on are committed under `test/fixtures-2.0/`.

## The BB2/BB3 migration feature on 2.1

`legacy.go` is untouched: parts still convert, health and quality still carry,
the technology is still granted, item stacks still survive. What changes is the
outcome — **every adopted incumbent balancer is built two-edges-per-tile by
construction, so on 2.1 each converts and is then refused** with the same
single-edge messaging and GPS pings. That is the honest result (the incumbent's
geometry cannot function on 2.1 under any design), and it composes from the
two features with no new code — but the `mig` suite's expectations change
wholesale on 2.1 (adopted clusters: converted yes, networks no), and the
portal description must say it plainly: migrating from Belt Balancer on 2.1
converts your parts and hands you a rebuild checklist, not a working machine.

## The interactive and demo worlds — a stated requirement of this port

**`test/interactive/bbb-interactive-setup` is the test-world-generating mod
behind both the checklist and the portal GIF captures** (the GIF session's
`mod-list.json.bak` shows it installed; no separate demo mod ever existed),
and every world it stages is multi-edge. Reworking it is part of the port,
not a follow-up:

- **All five gesture bands need single-edge geometry.** The pocket band's
  4-part dead-ended 4×4 becomes eight parts; the 65th-belt column becomes 65
  parts (64 input parts, one output part) — same P = 64, same gesture; the
  bridge and fast-replace bands re-lay accordingly. The **shrink band** needs
  an actual redesign, not a re-lay: its gesture was "lay a belt on a free face
  of a 2-part balancer", which single-edge refuses. The gesture becomes "lay a
  belt against an attached EDGELESS part" — a 2→2 over four parts plus a fifth
  part carrying no belt; the belt on the fifth part takes P 2→4 and mining it
  takes P back to 2, which is the same boundary crossing `bmin` pins.
- **A sixth band stages the six GIF scenes**, so the portal captures become
  reproducible and versioned instead of living in saves that (measured) get
  dismantled before anyone saves them. Single-edge redesigns: the c-shape,
  compact-column and long-run 8→8s and the 8→9 grow to one part per belt;
  **`single-part-1-to-3-fanout` is unrepresentable** (one part cannot carry
  four belts) and retires in favor of the cross form — a plus of five parts,
  four arm parts each carrying one belt, which is the same 1→3 read.
- **The GIFs are re-captured after the port**, because the portal must show
  the default geometry players will actually build — a portal page
  demonstrating multi-edge builds that a fresh install refuses would be a
  standing support ticket.
- A **migration gesture** joins the checklist: open a real 2.0 multi-edge save
  under 2.1 and check the chat summary, the GPS pings, the spilled items and
  the refused clusters — the interactive half of the fixture suite.

## Packaging: one tree, two releases

`factorio_version` is one value per release; the portal serves the right
release per game version. Plan:

- **Trunk targets 2.1**: `fklua.toml` moves to `factorio_version = "2.1"`,
  `api` re-pins to the 2.1.x runtime JSON, bindings regenerate (`fklua api
  check --to` first, to see what moves; the 2026-08-23 smoke test showed the
  control stage and guest init already come up on 2.1.14). `base >= 2.1.0`.
- **A `release/2.0` branch** carries the pinned diffs: `factorio_version`,
  the 2.0.77 api pin and its bindings. Kept rebased on master per house rule.
  The mod-data tree is IDENTICAL on both branches — the engine-version guard
  in the data stage is what makes one tree correct on both engines — so the
  branch diff is the manifest and the generated bindings, nothing
  hand-written.
- **FkLua ask to record in FKLUA-GAPS.md**: `fklua mod` reads identity only
  from `fklua.toml` and takes no flags, so per-target packaging from one
  checkout needs either a manifest override flag or the branch dance above.
  The branch is fine at this scale; file the gap, don't block on it.

## The test estate

The bulk of the labor, and it splits by which binary can run it:

| work | binary | notes |
|---|---|---|
| every suite's rigs rebuilt single-edge | 2.1 | the geometry doubles; every calibrated number in CLAUDE.md's tables gets re-recorded. The suites' assertions themselves mostly survive — what changes is the worlds their `on_init` builds |
| new `sedge` legs: second belt refused (build, rotation, robot), handed back negative, merge-spare via the bridge-tile path, feedback-gate once | 2.1 | the `lim`/`brdg` idiom verbatim |
| migration suite: fixture 2.0 saves loaded under 2.1 | 2.1 + fixtures | **the fixtures exist and are committed**: `test/fixtures-2.0/` carries the m2, edge, m3 and qual saves from the last 2.0.77 suite run (2026-08-22), preserved 2026-08-24 minutes ahead of anything overwriting `test/tmp` — they cannot be regenerated without a 2.0 binary. Covers: the load survives with per-tile pruning (S2 probe 1's numbers graduate into assertions), the rebuild tears down the remnants, hidden items recovered and spilled conserved, GPS summary logged, audit stable after |
| setting-flip suite (ON→OFF sweep, OFF→ON recompile) | **2.0.77 only** | multi-edge cannot be enabled on 2.1 at all; this and the multi-edge regression run of the old rigs live on the release/2.0 branch and need the old headless binary pinned |
| bench re-baselines | 2.1 | the per-balancer marginal cost changes (2× parts per rig); RESULTS.md numbers are a new session against new controls anyway |

## Spike S2 — the two gating probes are DONE (2026-08-24), four remain

**1. Loading a 2.0 save with stacked interfaces under 2.1.14: survives, with
silent per-tile deletion.** Method: the last 2.0.77 suite runs' saves
(2026-08-22, now committed as `test/fixtures-2.0/`), loaded under 2.1.14 with
the version-bumped flag-dropped mod copy plus an observer mod counting
interfaces per tile. Measured:

| fixture | as built on 2.0.77 | after the 2.1.14 load |
|---|---|---|
| m2 (21 rigs, 77 parts, sat4 = 4 parts / 8 belts) | ~140 interfaces, most tiles carrying 2 | **77 interfaces, exactly 1 per tile, 0 stacked** — hidden surface intact (313 linked + 203 splitters) |
| edge (95 parts; `lim` = 64 in + 1 out over 32 parts) | 160+ interfaces, `lim` tiles carrying 2–3 | **95 interfaces, exactly 1 per tile** — hidden intact (779 + 531) |

No crash, no error, **no log line for any of it** — the deletion is completely
silent. The crippled networks then run: `ctrl` and `pass` (vanilla belts)
deliver full rate (1286), `sat4` delivers **14 items against ~1305**, most
rigs read zero, and `a3to5` shows the survivor lottery (two outputs at full
rate, three dead — which interface lived per tile decides everything). The
old guest with no refusal logic also mishandled a recompile in that world
(reinsertion returned 0, 48 items to ground) — not chased, since that guest
state is not the ported design; the ported guest refuses those clusters.

**2. A fresh single-edge network compiled on 2.1.14 works, at full rate.**
Scripted probe, flag-dropped mod: a 1→1 over two parts delivered **1310**
against a bare-belt control's 1294, and a 2→2 over four parts **1306 1306**,
0.00% spread, zero errors and zero alerts. The architecture needs no entity
redesign on 2.1 — only the rule.

Also found on the way: **2.1 removed `LuaGameScript::create_profiler`** — the
m2 test mod died on it (`doesn't contain key create_profiler`) until the call
was rewritten `helpers.create_profiler`, which works. Every test mod and the
bench harness carry that call; it goes on the port checklist. The guest calls
no profiler.

Still open, in order:

3. **`settings.global` written from script**: the write itself, whether it
   raises `on_runtime_mod_setting_changed` for the writer, and whether
   FkLua's generated bindings cover LuaSettings' dictionary read AND write
   plus the event subscription — the grandfather pass stands on all three
   (candidate FKLUA-GAPS items if not).
4. `mods` availability in the settings stage (decides where the setting is
   defined vs hidden).
5. `fklua api check --to <2.1 pin>`: what this mod actually calls that moved
   (`create_profiler` says the answer is not "nothing").
6. Adoption of a VALID single-edge network across a 2.0→2.1 upgrade (the
   happy-path half of the fixture suite).

## Open decisions, with recommendations

| decision | recommendation | why |
|---|---|---|
| migrated invalid balancers | refuse + spill + GPS summary (not partial service, not standing) | honesty over arbitrary degradation — and "standing" is not on offer anyway: the engine prunes the interfaces at load, measured, so an unhandled multi-edge network arrives already crippled (sat4 at 14 items of 1305, survivor-lottery ports) |
| the setting's kind | runtime-global, default false, script-grandfathered | a script cannot write a startup setting, and the grandfather pass is the mod flipping it; also: no restart to flip, and the data-stage flag no longer depends on it |
| the grandfather latch | first load of the updating version only; never auto-flipped down afterwards | a silent downgrade under a player relying on multi-edge is a trap; flipping off is a one-click Map setting once they have rebuilt |
| where the setting lives on 2.1 | not defined at all | no dead toggles; falls back to `hidden` if the settings stage cannot see `mods` |
| refusal UX for the second belt | reuse limit.go verbatim, new locale key | it is the same gesture with a different bound |
| 2.0 branch vs FkLua packaging flag | branch now, file the gap | unblocks immediately; the gap is real but small |
| old multi-edge rigs | kept, 2.0-branch-only regression | multi-edge remains shipped (opt-in) on 2.0 and needs coverage while it exists |
