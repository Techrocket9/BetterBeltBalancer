# single-edge.md — the 2.1 port: one belt per part

Status: **PHASES 1 AND 2 SHIPPED 2026-08-24** — the rule and its refusal, then
the setting, the grandfather pass and the migration. Drafted the same day from
the 2026-08-23 investigation and boskid's answer to the interface request
(forums t=135830). Read CLAUDE.md's migration and over-limit sections first;
this design reuses both wholesale.

What is implemented and what is not is the two "Implementation status" sections
at the end of this file. The short form: the rule enforces, the refusal reaches
the player through the sixty-fifth belt's own machinery, the merge is spared, a
2.0 multi-edge save opened on 2.1 has its remnants torn down and its owning
forces told with a ping per balancer, and the `sedge` and `mig21` suites are
green on 2.1.14. What is left is the rebuilt test estate and the interactive
worlds; the setting-flip legs and the multi-edge regression run need a 2.0
binary and belong on the `release/2.0` branch.

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

**Release identity, decided 2026-08-24**: the first GA release is the
multi-edge 2.0 mod as it stands, and the port ships as **0.2.0** on both
engine arms — the version bump is what makes `fk_migrate(old_version)` the
grandfather trigger's front door (with the build stamp doing the actual heap
decline; see S2 result 6). `fklua.toml` carries 0.2.0 on trunk already, along
with the portal description rewrite ("Belt balancers that click together into
arbitrary shapes. Highly performant; megabase-ready (UPS hit is ~equal to
vanilla hand-crafted belt balancers).").

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

**3. The settings surface (MEASURED 2026-08-24, all on 2.1.14).** The engine
side supports the grandfather pass; the binding side is one new member and one
hard gap:

- `mods` IS visible in the settings stage (and so is `feature_flags`), so
  define-on-2.0-only stands and the `hidden` fallback is not needed.
- Reading an undefined setting returns nil with no raise — so the guest's
  policy read needs no gate at all: nil IS the "not defined on this engine"
  answer. **Writing an undefined key RAISES** (`LuaCustomTable doesn't
  contain key …`), so the grandfather WRITE must be gated on the capability
  marker — the gate is load-bearing against a raise, not merely policy.
- The script write works, round-trips through save/load, and is **per-save**
  (never written back to `mod-settings.dat`) — the latch cannot escape the
  save it was taken on, which is exactly right. `settings.startup` is
  confirmed read-only (`LuaCustomTable is read only`).
- `on_runtime_mod_setting_changed` fires **synchronously, inside the
  assigning statement**, with NO `player_index` for a script write, does not
  fire in an `on_init` dispatch, does not fire on load — and **fires on a
  same-value write too**, so the handler's heap-byte comparison is
  load-bearing, not defensive. The synchronous dispatch is a re-entrancy
  hazard of the `mine_entity` class: a write issued mid-flush re-enters
  `fk_on_event` with the compile buffers live, so the write belongs after
  `endCarry()` exactly as `revertOverLimit` does, with byte-before-setting
  making the re-entrant handler a no-op.
- **Bindings**: the event is fully generated (id constant, reader, PlayerIndex
  mask); the READ costs one new bound member (`LuaSettings.global` as the
  handle-returning GETH form, then the already-bound `LuaCustomTable` index —
  the `prototypes.entity` idiom from legacy.go). **The WRITE is inexpressible
  — FKLUA-GAPS.md item 23**: the runtime API declares no write side on
  `LuaCustomTable`'s index operator (prose only) and the ABI has no
  index-assign member kind. Upstream ask filed; the shippable fallback if it
  does not land is the latch living in the heap byte with the setting as
  input only (effective = marker AND (setting OR grandfatheredByte); any
  setting-changed event clears the byte and hands control to the setting) —
  works, but the settings UI shows false on a grandfathered save, so the
  warning copy must explain it.

4. **Answered with 3**: settings-stage `mods` is available.

5. **DONE — the api-pin recon**: exactly 2 of 2.1's 202 breaking changes touch
   this guest and both are additive-at-the-end and unreadable by construction
   (the ABI marshals by field name Lua-side); zero removed fields across all
   22 subscribed events; the 2.0.77 pin is safe for the implementation work,
   and the trial re-pin branch (`api-pin-2.1`) is green and parked. FkLua
   already carries the 2.1.14 API descriptions committed. The only Lua-side
   breakage in the repo was `game.create_profiler` (12 sites), fixed on
   master with the version-agnostic `helpers.create_profiler`.

6. **DONE — adoption happy path**: a single-edge world built on 2.1.14 and
   loaded by a build-stamp-bumped mod reports `2 networks adopted, 0 rebuilt`
   with `compiles=0 builds=0 teardowns=0 creates=0` and delivery identical to
   the item across all arms (rig1 1310, rig2 1306/1306 against ctrl 1294),
   audit `drift=0 unbuilt=0`. **One correction to the design's premise: a
   VERSION bump alone does not decline the guest heap — the BUILD STAMP
   does.** A real release moves both by construction, but any fixture-suite
   leg that bumps only `info.json` silently tests nothing (the heap adopts
   and `rebuildFromWorld` never runs); `test/run.sh`'s `bump_build` moves
   both, and the migration suite must too. The grandfather trigger is
   therefore honestly "`fk_migrate`, which fires on a build-stamp change",
   not "a version bump".

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

## Implementation status — phase 1, 2026-08-24

**Shipped: the rule, its refusal, its merge arm and its suite.** Everything
below is measured on Factorio 2.1.14 against the shipped configuration
(`--persist=packed --gc=collected`).

| what | where |
|---|---|
| the data-stage version branch: `not_colliding_with_itself` on 2.0.x only, and the `bbb-can-stack` marker in the same `if` | `mod-data/prototypes/hidden.lua` |
| trunk targets 2.1 (`factorio_version`, `base >= 2.1.0`) | `fklua.toml` |
| the capability, the predicate, the refusal, the bridging-tile theorem and the backstop | `guest/go/sedge.go` |
| the per-tile count, taken out of the walk that was happening anyway | `classifyEdges`, `guest/go/compile.go` |
| the refusal asked in front of the teardown, beside the port limit's | `compile`, `guest/go/compile.go` |
| the three-way admission and the message, now shared by both bounds | `refuseAdmit` / `tellRefusal`, `guest/go/limit.go` |
| the merge pre-pass, asking both bounds | `overLimitMerge`, `guest/go/limit.go` |
| the tick's new part tiles | `noteAddedPart`, called from `AddPart` |
| the capability re-derived at the three load hooks | `edgeModeRecheck`, `guest/go/main.go` |
| the two locale keys | `mod-data/locale/en/better-belt-balancer.cfg` |
| the suite | `test/mods/bbb-sedge-test/`, `test/assert-sedge.py`, `test/run.sh` |

### The `sedge` suite, measured

Eight clusters over thirty-six parts on a flat scratch surface, 3,500 ticks,
base only. Five rigs measure the rule and three break it. One saturated express
belt delivered **974 items** over the settled window (t=2100 to t=3400):

| rig | what it is | per-output | total vs one belt | spread |
|---|---|---|---|---|
| `s11` | 1 -> 1 over **two** parts, P=1 | 976 | 1.002x | 0.00% |
| `s22` | 2 -> 2 over **four** parts (2x2), P=2 | 974 974 | 2.000x | 0.00% |
| `s44` | 4 -> 4 over **eight** parts (4x2), P=4 | 974 976 974 976 | 4.004x | 0.21% |
| `s35` | 3 -> 5 over **ten** parts (5x2), P=8 with loopbacks | 584 584 584 586 585 | 3.001x | 0.34% |
| `sbld` | a 2 -> 2 whose part was given a second belt at t=500 | 974 974 | 2.000x | 0.00% |
| `srot` | a 2 -> 2 whose neighbour belt was rotated onto it at t=600 and back at t=1200 | 974 974 | 2.000x | 0.00% |
| `smrg` | two 1 -> 1s bridged at t=1400 and un-bridged at t=1900 | 974 976 | 2.002x | 0.21% |

**The port counts are asserted before any rate is read**, as an exact multiset
over the create log: three 1->1 over 1, three 2->2 over 2, one 4->4 over 4 and
one 3->5 over 8. A rig whose belts did not land where the geometry intended
still delivers a plausible number; the port count is what says the machine is
the one that was asked for.

**The three refusals**, one per leg and exactly one per distinct edge state:

| leg | how the second belt arrives | what came out |
|---|---|---|
| `sbld` | script-built against an occupied part | **one** `alert:` naming 1 part at worst 2 belts, **one** clean `told force`, the standing network delivering 2.000x for the remaining 3,000 ticks |
| `srot` | a belt ROTATED onto an occupied part -- `entity.direction = ...` raises nothing at all, so the audit is what finds it | the audit reports `drift=1` before it repairs, the repair reaches the refusal, and rotating the belt back is a **SKIP**: `drift` returns to 0 with nothing torn down and nothing built |
| `smrg` | a part BRIDGING two working balancers, whose bridging tile carries a belt on each side | the merge pre-pass spares **2** standing networks, and across the whole dispatch **0 teardowns, 0 builds, 0 spills** -- both halves still delivering 2.018x one belt while the refusal stands, the same figure they delivered before it |

Audits, tag by tag, `(clusters, parts, nets, drift, unbuilt)`: `t0` (8, 36, 0, 0,
8) -- the audit inside `on_init`'s marker dispatch reports before the drain it
forces compiles anything -- then (8, 36, 8, 0, 0) built, (8, 36, 8, **1**, 0)
with `sbld` refused, (8, 36, 8, **2**, 0) with `srot` refused as well, back to
(8, 36, 8, 1, 0) when the rotation is undone, (**7**, **37**, 8, **2**, 0) while
the merge stands refused -- both spared networks counted under keys that are no
longer roots -- and (8, 36, 8, 1, 0) after it and at the end.

**`drift=1 unbuilt=0` and never the other way round.** A refused cluster still
HAS its network and knows its edge list has moved past what the mod can build;
`drift=0 unbuilt=1` is the signature of a refusal that demolished first and
asked afterwards, and it is what the red proof below produces.

**Zero hand-backs over the whole run**, which is the standing negative: a
headless `--create` has no players, so `game.get_player` resolves to nothing and
`revertOne` returns before it mines anything. A hand-back here would be a revert
firing for a script build. The flying text, the sound and the piece coming back
are behind the same player wall as the sixty-fifth belt's, and join it on the
interactive checklist.

**Zero spills over the whole run.** Every refusal happens in front of its
teardown, so no network ever comes down for one.

### The red proof

`multiEdgeAllowed()` forced to return true -- one line, and the semantic
equivalent of the pre-port guest -- rebuilt and re-run:

- **`test/run.sh` kills the run** before a single assertion is read, on **ten**
  `[BBB] error: create_entity returned nil for bbb-linked-belt` lines. That is
  the engine refusing the second interface on a tile, which is the whole reason
  the rule exists.
- Run against the logs by hand, **fifteen assertions fire**: zero refusals where
  three are expected, zero forces told, zero merges spared, the un-merge
  building **both** halves instead of one, **four spills totalling 64 items**,
  six audit lines reading `drift=0 unbuilt=1` or `unbuilt=2` -- the signature
  above -- and `sbld` and `smrg` delivering **0.000x one belt**: the balancers
  a player was using simply stop.

### Two deviations from the design, and why

**1. "Also spare a merge when either predecessor is already in the refused map"
is NOT implemented, because it is unsound.** The design offers it as free
conservatism; it is the one direction that is not free. `spareMerge` must be
exact in the YES direction -- sparing a merge that then compiles successfully
leaves both predecessors' networks standing beside the new one, three networks
over one cluster, two of them holding items nothing will ever come back for. And
a predecessor being refused does not imply the merged cluster is: a part is
fast-replaceable onto a belt (`fastreplace.go`), so the bridging part can be
placed ON one of the two belts that made a predecessor's tile multi-edge, taking
that tile to one edge and the merged cluster to perfectly buildable. The same
argument retires it for the port bound. What IS implemented is the sound half --
the bridging-tile theorem, which the design states correctly -- plus a fall-back
to the full classification for a candidate that has stranded networks under it,
where the theorem's premise ("the predecessors were valid") is the thing that
does not hold.

**2. The bridging-tile theorem needs the tick's new part tiles, and they are
written down rather than re-derived.** `AddPart` appends the tile to a
high-water slice truncated by every flush -- the `buildNotes` shape exactly.
Nothing at all is recorded on an engine that can stack or inside a
`rebuildFromWorld`, and the classification behind it is skipped outright unless
a merge would really strand a standing network, which is what keeps a blueprint
paste and a whole-world rebuild off it.

### What it costs

Package built 2026-08-24, shipped configuration, against the master build the
2026-08-23 probe packaged (same guest, hand-edited manifest):

| | before | after | |
|---|--:|--:|---|
| `fk_module.lua` | 2,747,461 B | **2,837,744 B** | +3.29% |
| `dist/bbb.wasm` | 1,163,064 B | 1,189,262 B | |
| `dist/better-belt-balancer_0.1.0.zip` | 414,060 B | **421,520 B** | +1.80% |
| members bound into the mod | 50 | **50** | of 4,257 -- and `fk_api_gen.lua` is **byte-identical** to master's |
| prototypes added | — | **1** | `bbb-can-stack`, CONDITIONAL: never defined on 2.1 |

The zip and wasm baselines are CLAUDE.md's recorded numbers rather than a
same-session rebuild; the `fk_module.lua` and API-table comparisons are against
the packaged artifact itself and are exact. `EntityRaw` and `LuaCustomTable.Get`
were already bound for the migration's marker probe, so the capability query
crosses nothing new -- which is what the byte-identical API table says.

**Nothing on any hot path moves, and it is structural rather than measured**:
the per-tile count is one integer per tile inside a walk that already visits
every side, the capability is one integer compare after the first call, and the
merge arm makes no host call at all unless a merge would strand a standing
network. **The `mar` suite cannot say so**, because it cannot run: its rigs are
multi-edge. That measurement is owed and is part of the test-estate phase.

### What runs on 2.1.14 and what does not

**Two of the eleven suites.** Measured rather than assumed, and in two layers:

- 2.1 refuses a mod whose `info.json` says `factorio_version: 2.0` outright --
  *"Incompatible Factorio version (current: 2.1, required: 2.0)"* -- so every
  suite fails at the loader before an entity is placed.
- **`m1` needed nothing but that one token and is GREEN**: 6 + 3 phases, cluster
  counts and sprite variations identical to their recorded values. It is
  belt-free, so the rule this port is about cannot touch it. Its manifest is
  bumped and it is back in the default.
- The other nine build multi-edge rigs, so bumping their manifests would only
  move the failure from the loader to the compiler. `test/run.sh`'s default is
  `m1 sedge`; the nine stay reachable by name.

### The two things this phase found that the design did not

- **`reclaimStranded`'s "simply released, with no teardown at all" is true of
  only one of two cases.** `removePart` marks the OLD ROOT dead
  unconditionally, and the old root of an un-merging cluster is whichever of the
  three nodes union-find kept. When it is the bridging part's own brand-new node
  the teardown is a no-op and both halves skip -- which is what the `edge`
  suite's `brdg` leg measures, by an accident of freed-id reuse. When it is a
  predecessor's root, which is what happens in a save that has freed no node ids
  (this suite's), that half is torn down and rebuilt. Conservation and placement
  are correct either way -- the `sedge` suite measures 10 items drained and 10
  put straight back, 0 spilled -- so it is a cost rather than a defect, and the
  suite asserts the pair rather than one of them. CLAUDE.md's "The merge that
  would be over the limit" already says the key must not be assumed; what it
  also says, that mining the bridge back out costs zero teardowns, is true only
  of the first case.
- **`spareMerge`'s log line named a bound it no longer owns.** It said "would
  merge past the port limit"; there are two bounds now, so it says "would merge
  into a cluster this mod cannot build" and the `alert:` a moment later says
  which. `test/assert-edge.py`'s `SPARED` regex has to move with it when that
  suite is rebuilt.

## Implementation status — phase 2, 2026-08-24

**Shipped: the setting, the grandfather pass, the migration and its fixture
suite.** Everything below is measured on Factorio 2.1.14 against the shipped
configuration (`--persist=packed --gc=collected`), except where it says
explicitly that it cannot be.

| what | where |
|---|---|
| the version branch, now shared by the DATA and SETTINGS stages instead of duplicated | `mod-data/engine.lua`, required by `prototypes/hidden.lua` and by `settings.lua` |
| `bbb-multi-edge-parts`, runtime-global, default false, defined on 2.0.x only | `mod-data/settings.lua` |
| the fold -- capability AND policy, the flip's obligation, and whether to grandfather -- as PURE GO with all eighteen of its states proved | `guest/go/edgemode/`, run by `make check` |
| the policy read, the write, the anchor, the flip handler, the sweep, the condemnation, and both summaries | `guest/go/sedge.go` |
| the ordering carve-out: a condemned network comes down BEFORE the refusal | `compile`, `guest/go/compile.go` |
| the summary spoken from the informed flush, after `endCarry` | `settleEdgeMode`, called from `flush`, `guest/go/compile.go` |
| the fold over the rebuild's own classification, and the condemnation | `rebuildFromWorld` / `inspectNetwork`, `guest/go/lifecycle.go` |
| `refused=` on the audit line | `auditAll`, `guest/go/lifecycle.go` |
| the subscription and its dispatch | `guest/go/main.go` |
| the two summary keys, the setting's name and its description | `mod-data/locale/en/better-belt-balancer.cfg` |
| the suite | `test/mods/bbb-mig21-observer/`, `test/assert-mig21.py`, `test/run.sh` |

### The shape of it, in one paragraph per decision

**The two questions stay split and the marker is the OUTER term.** `stackCapable`
is the `bbb-can-stack` point query phase 1 already had; `settingMultiEdge` is the
`settings.global` read beside it; `multiEdgeAllowed` is the AND, cached as one
integer compare because it is asked on `noteAddedPart`'s path. On 2.1 the setting
is never read at all -- the AND short-circuits -- which is exactly right, because
it is not defined there.

**The heap byte is the reconciliation anchor, not a cache of the setting.** It
records the mode the REGISTRY was last reconciled under, which is what a flip has
to be compared against: the setting can move between two loads of one save and
the standing networks cannot. `edgeAnchorSettle` writes it from
`rebuildFromWorld`, from `fk_on_init` and from both arms of the flip handler.

**The write raises `on_runtime_mod_setting_changed` synchronously**, inside the
assigning statement, and it fires for a same-value write too (both measured, S2
above). So the grandfather pass writes the ANCHOR FIRST and the setting second:
the re-entrant handler compares the setting against an anchor that already
agrees, and does nothing. No self-write flag exists and none is needed.

**The write is where `revertOverLimit`'s mine is**: `flush()`, after
`endCarry()`. Same three reasons -- a synchronous re-entry into the queues the
drain is iterating, the package-level compile buffers, and a carry transaction
that must be closed before anything can file against it.

**The ordering carve-out is the one place a refusal demolishes anything.** Every
other refusal leaves the standing network alone, because the machine is fine and
only the requested edit is not; a CONDEMNED cluster is the opposite case, and
only two producers can tell them apart -- `inspectNetwork`, which is the one
function that knows both what the world asks for and whether the standing
interfaces match it, and `sweepStackedInterfaces` for the 2.0 flip. Both invert
the stored fingerprint before condemning, which is what carries a condemned
cluster past the skip: flipping a setting moves nothing in the world.

**One scan, two outcomes, chosen by the capability marker.** The rebuild folds
"does any tile carry two belts" out of a classification it was making anyway --
zero extra host calls -- and notes the root. On 2.0 those clusters are ADOPTED,
because every interface is still standing and the adoption comparison matches
exactly, and the grandfather pass offers to keep them working. On 2.1 the engine
has already deleted all but one interface per tile, so the comparison cannot
match, the remnant is condemned, and the migration summary speaks instead.

**`multi` is RETURNED by `inspectNetwork` rather than read out of `sedgeWorst` by
its caller**, and that is not style. That function has three early returns that
never classify -- an empty cluster, a surface that has gone, and the ordinary
"nothing standing, this is a clean build" -- and on any of them the global still
holds the PREVIOUS cluster's answer. Reading it directly would attribute one
balancer's shape to another, which on this load means condemning a working
machine.

### The `mig21` suite, measured

Two committed fixtures, loaded under 2.1.14 with today's mod, 320 ticks each.
There is no `--create` phase and there cannot be: the worlds were built by a
2.0.77 binary that is gone. **The fixture is phase one.**

| | m2 | edge |
|---|---|---|
| what the save holds | 21 rigs, 77 parts, 4 surfaces | 15 clusters, 95 parts, 3 surfaces, including `lim` at 64 belts over 32 parts |
| the heap | **declined** -- `the mod was rebuilt` and a rebuild from world, both asserted rather than assumed | the same |
| the rebuild | 4 surfaces, 77 parts, 21 clusters, **0 adopted, 21 rebuilt** | 3 surfaces, 95 parts, 15 clusters, **0 adopted, 15 rebuilt** |
| what the ENGINE had already done, before any script | **77 interfaces over 77 part tiles, 0 stacked tiles**, hidden networks whole at 652 entities | **95 over 95, 0 stacked**, 1,898 hidden entities |
| seeded into those networks (see below) | 2,320 items | 6,540 |
| the teardowns | **21, recovering 2,320 -- every item, exactly** | **15, recovering 6,540** |
| the spills | **21, placing 2,320 -- all of it, since a refused compile claims nothing back** | **15, placing 6,540** |
| items put back INSIDE a network | **0** | **0** |
| on the ground afterwards | 1,006 of 2,320 (the rest landed on the player's own belts, which `spill_item_stack` allows) | 5,645 of 6,540 |
| the compiler's entities afterwards | **0 visible, 0 hidden**, at t1, post-audit and final | the same |
| the player's parts afterwards | **77**, untouched at every sample | **95** |
| refusals | 21 clusters over **42** lines | 15 over **30** |
| the summary | **one** `force.print`, to force 1, naming 21 balancers | **two**, force 1 about 14 and force 4 about 1 |
| the audit, twice and identical | `clusters=21 parts=77 nets=0 drift=0 unbuilt=0 refused=21` | `clusters=15 parts=95 nets=0 drift=0 unbuilt=0 refused=15` |

**Two refusal lines per cluster is the designed shape rather than a wart.** The
rebuild refuses with the worst information a refusal will ever have and is
forbidden to speak, so it logs and re-queues; the informed flush a tick later
refuses again and delivers the one message. That is `refuseAdmit`'s three-way
admission doing exactly what the wake race made it for.

**The edge fixture's second force is not decoration.** Two forces' parts touching
are two balancers, so the summary is spoken twice -- once to each -- and the
counts add up to the refused total. It is the only place anything checks that the
message is per FORCE rather than per balancer or per save.

### Two things the suite had to solve, and both are worth knowing

**THE FIXTURES ARE TICK-0 SAVES, SO THE NETWORKS ARE EMPTY.** A `--create` never
reaches a tick, so the rigs were built and the save written with every belt in
them empty -- and a migration that recovers nothing, spills nothing and conserves
nothing trivially would satisfy every count in this suite while proving none of
them. So the observer SEEDS: one item into every transport line of every entity
the compiler placed, on every surface, in the one moment before the migration
runs. That is better than a stand-in for a running balancer's contents, because
it is a KNOWN NUMBER -- "what the teardown recovered" is asserted as an equality
against it rather than as a floor.

**THE ONLY "BEFORE" ANY SCRIPT CAN REACH IS `on_configuration_changed`.** The
migration does not wait for a tick: the heap is declined, so `fk_migrate` fires
before tick 0 and by then the remnants are down. The observer therefore samples
and seeds from its own `on_configuration_changed`, which runs first because
`bbb-mig21-observer` sorts before `better-belt-balancer` and deliberately
declares no dependency on it -- a dependency would put it after. **It does not
have to be trusted**: if that order ever flipped there would be nothing left to
seed, the count would be zero, and the suite fails on a zero. It cannot pass
vacuously.

**And the `cfg` sample is already post-pruning whatever happens**, because the
engine deletes the second belt-connectable on every tile at LOAD, before any
script of any mod runs, with no log line at all. Nothing here can see the world
as the 2.0 binary left it, and every assertion is written knowing that. What the
engine deleted went with the interfaces it deleted -- at most eight items each --
and is not ours to recover.

### `refused=` on the audit line, and why it had to exist

The migration tears a condemned remnant down on purpose, so its clusters end with
no network -- and `drift=0 unbuilt=1` is, in this repository, **the signature of a
refusal that demolished first and asked afterwards**, which is the defect the
sixty-fifth belt pass fixed. Without a way to say "the mod DECLINED to build
this" the audit could not tell the two apart.

So `auditAll` compares each cluster's fresh fingerprint against the one the
feedback gate remembers refusing on, counts the matches, and reports them. A
refused cluster is never counted `unbuilt` -- `unbuilt` is this guest saying it
should have built something and did not. The column is **appended** rather than
inserted, because this line is the assertion surface for every suite in the
repository and several match it with an unanchored pattern over the five counters
that were there first; the `sedge` suite's recorded audit tuples are unchanged to
the digit. Measured there, where a refused cluster still HAS its network, the
column reads `refused=1` beside `drift=1 unbuilt=0` and `refused=2` while the
merge stands refused -- so the two shapes of refusal are visible and distinct.

### Red-proven three times, and each proof catches a different thing

Every one is an injected defect, built, run against the m2 fixture, and reverted.

| injected defect | what came out |
|---|---|
| **`condemnStanding` made a no-op** -- the adoption carve-out gone, so a refusal never demolishes | **eleven assertions**, and the state they describe is the phantom: **77 interfaces and 652 hidden entities still standing** at every sample, **0 teardowns and 0 spills**, **2,320 items stranded inside networks nothing will ever come back for**, nothing on the ground, and the audit reading **`nets=21 drift=21 unbuilt=0 refused=21`** -- twenty-one balancers that the mod knows are wrong and has left standing on an engine where a stacked linked belt is a latent risk on every load |
| **`settleEdgeMode` returning immediately** -- the summary and the write suppressed | **exactly two**, both about the message: no summary line and no force told. **Every other number is unmoved** -- the remnants still come down, 2,320 items are still recovered and spilled, and the audit still reads `refused=21`. The two proofs are not redundant with each other |
| **the announce check removed from `refuseSingleEdge`** -- so a migrated cluster falls through to the ORDINARY per-piece message | **exactly one**, and it is the one that says so: a migration announced with a sentence about an extra piece being left in place unconnected, when nobody placed anything. This is the check that would otherwise never have been able to fail |

### What it costs

All three of the suites that run on 2.1 are green in BOTH `-gc` arms, which
phase 1 could not manage: it built the leaking arm and did not run it.

Package built 2026-08-24, shipped configuration, against master at `b356515`
(the bindings regeneration that landed the index-assign) built the same way in
the same session:

| | before | after | |
|---|--:|--:|---|
| `dist/better-belt-balancer_0.2.0.zip` | 422,640 B | **439,888 B** | +4.08% |
| `fk_module.lua` | 2,837,744 B | **3,019,175 B** | +6.39% |
| `dist/bbb.wasm` | 1,189,328 B | 1,240,253 B | |
| members bound into the mod | 50 | **53** | of 4,259 |
| events subscribed | 22 | **23** | |
| prototypes added | — | **0** | |
| settings added | — | **1** | CONDITIONAL: never defined on 2.1 |

The three members are `LuaSettings.global` in its handle-returning form,
`LuaCustomTable`'s INDEX-ASSIGN operator -- which is FKLUA-GAPS.md item 23, asked
for by this feature and landed upstream before it was written -- and
`LuaSurface.name`, for the `[gps=x,y,surface]` pings.

**Nothing on any hot path moves, and it is structural rather than measured.**
`multiEdgeAllowed` is the same single integer compare it was in phase 1, with a
second cached lookup behind it that is resolved once per heap and short-circuits
on 2.1 before the setting is read at all. `settleEdgeMode` is one length test on
an empty slice per flush; `takeCondemned` is a scan of an empty slice per
compile; `sedgeAnnounce` and `sedgeCondemned` are nil in every save built under
the rule it is running under. The setting-changed subscription cannot fire at all
on 2.1. **The `mar` suite cannot say so**, because it cannot run -- its rigs are
multi-edge -- and that measurement is owed with the rest of the test estate.

### What is still not verified, and where it has to be

| path | state |
|---|---|
| the grandfather pass ACTUALLY FLIPPING the setting | **Implemented, and unreachable on 2.1 by construction**: the setting is not defined, so `GrandfatherNeeded`'s marker term is false. What IS pinned is the fold, exhaustively, by `go test ./edgemode/` -- including the state that matters most here, that the write is never attempted where the key does not exist -- and the NEGATIVE, by the `mig21` suite, which fails on any grandfather line, any failed-write alert and any setting-changed line. The positive needs a 2.0 binary and belongs on the `release/2.0` branch |
| the flip handler, both arms, and `sweepStackedInterfaces` | **Implemented, unreachable on 2.1**: nothing can change a setting that is not defined. Same split -- `edgemode.Reconcile` proves what each flip obliges, over all eighteen states; what a sweep DOES to a standing world is a 2.0 leg |
| the summary reaching a PLAYER rather than a force | `force.print` is what a headless run can see, and the suite asserts the LocalisedString crossed and the force resolved. Whether the `[gps=]` pings are clickable and land where they should is a graphical client's question and joins the interactive checklist |
| a fixture with SINGLE-EDGE clusters adopting beside the refused ones | **Not exercised, because neither fixture has one**: every m2 rig is multi-edge (a 1->1 over one part already carries two belts) and so is every edge cluster. The suite asserts `adopted + rebuilt == clusters` and reports the split, so a fixture that grew one would be visible rather than silently ignored |
