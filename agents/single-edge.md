# single-edge.md — the 2.1 port: one belt per part

Status: **PHASES 1 THROUGH 7 SHIPPED 2026-08-24** — the rule and its
refusal, then the setting, the grandfather pass and the migration, then the
interactive and demo worlds, then the rebuilt test estate in four tranches
(`m2`, `mar` and `upg`; `mix`, `plat` and `qual`; `m3` and `edge`; `mig`).
Drafted the same day from the 2026-08-23 investigation and boskid's answer to
the interface request (forums t=135830). Read CLAUDE.md's migration and
over-limit sections first; this design reuses both wholesale.


What is implemented and what is not is the seven "Implementation status"
sections at the end of this file. The short form: the rule enforces, the refusal
reaches the player through the sixty-fifth belt's own machinery, the merge is
spared, a 2.0 multi-edge save opened on 2.1 has its remnants torn down and its
owning forces told with a ping per balancer, every gesture rig and every demo
scene is single-edge and headlessly verified, and **all thirteen suites are
green on 2.1.14 in both `-gc` arms**. **The `mar` slopes are measured again**,
which is the gate the first three phases shipped without and which their
"nothing on any hot path moves" claims were waiting on. What is left is the GIF
re-capture, which needs a graphical client; the setting-flip legs and the
multi-edge regression run need a 2.0 binary and belong on the `release/2.0`
branch.

**One defect this estate work found and did not fix**, because it is a guest
decision rather than a test one: a BB2/BB3 conversion reaches the ORDINARY
per-piece refusal message rather than the migration summary, unless a
rebuild-from-world happens to follow it in the same session. Phase 7's "The
message a converted balancer gets" is the measurement and the proposed fix.


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
outcome — **every incumbent balancer built the incumbent's way is
two-edges-per-tile by construction, so on 2.1 each converts and is then
refused**. That is the honest result (the incumbent's geometry cannot function
on 2.1 under any design), and it composes from the two features with no new
code — but the `mig` suite's expectations change wholesale on 2.1 (converted
yes, networks no), and the portal description must say it plainly.

**MEASURED 2026-08-24, and the paragraph above needed two corrections.** Phase 7
built the suite and ran it:

- **"every adopted balancer" is too strong, and the exception is the good news.**
  A Belt Balancer user whose balancer happens to be one belt per part — a
  two-column block, inputs down one side and outputs down the other — has a
  shape 2.1 can build, and theirs converts into a **working network at exact
  rate**. So the portal sentence is *migrating from Belt Balancer on 2.1
  converts your parts and hands you a rebuild checklist, not a working machine —
  except balancers already built one belt per part, which keep working.*
- **"with the same single-edge messaging and GPS pings" is NOT what happens**,
  and that is the defect phase 7 found. See "The message a converted balancer
  gets" below.

## The interactive and demo worlds — a stated requirement of this port

**DONE 2026-08-24.** What was actually built, what deviates from the paragraphs
below and why, and the measurements: "Implementation status — phase 3" at the
end of this file. The GIF re-capture is the one part still outstanding, because
it needs a graphical client.

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
| ~~every suite's rigs rebuilt single-edge~~ **ALL THIRTEEN DONE 2026-08-24** | 2.1 | the geometry doubles; every calibrated number in CLAUDE.md's tables gets re-recorded. The suites' assertions themselves mostly survive — what changes is the worlds their `on_init` builds. Confirmed by all four tranches: **not one assertion in `m2`, `upg`, `plat`, `qual`, `m3` or `edge` had to be weakened**, every rate and every port count is the number it was, and `plat`'s whole stacking leg came back identical to the item. See "Implementation status — phase 4" for the two rigs that needed a redesign rather than a re-lay, "phase 5" for the one assertion that had to be retired rather than re-recorded, "phase 6" for the four `edge` rigs whose GESTURE the rule changed, and **"phase 7" for `mig`, the one suite whose rigs were deliberately NOT re-laid**: they are somebody else's world, so re-laying them would have been re-laying the thing under test |
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
network. **The `mar` suite could not say so when this was written**, because it
could not run: its rigs were multi-edge. **It says so now** — phase 4 re-laid them, and the three legs that measure the event path came back
byte-identical (32 B for a far belt built, 0 for one mined, 352 B for a belt laid
inside the neighbour gate and picked up again), with the 4×4 recompile term
unmoved at 3,736 B. See "Implementation status — phase 3" below.

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
| `dist/better-belt-balancer_0.2.0.zip` | 422,640 B | **439,832 B** | +4.07% |
| `fk_module.lua` | 2,837,744 B | **3,018,845 B** | +6.38% |
| `dist/bbb.wasm` | 1,189,328 B | 1,240,187 B | |
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
on 2.1. **The `mar` suite could not say so when this was written** -- its rigs
were multi-edge -- and it says so now, from phase 4: see "Implementation status
— phase 4" below for the slope table it produced.

### What is still not verified, and where it has to be

| path | state |
|---|---|
| the grandfather pass ACTUALLY FLIPPING the setting | **Implemented, and unreachable on 2.1 by construction**: the setting is not defined, so `GrandfatherNeeded`'s marker term is false. What IS pinned is the fold, exhaustively, by `go test ./edgemode/` -- including the state that matters most here, that the write is never attempted where the key does not exist -- and the NEGATIVE, by the `mig21` suite, which fails on any grandfather line, any failed-write alert and any setting-changed line. The positive needs a 2.0 binary and belongs on the `release/2.0` branch |
| the flip handler, both arms, and `sweepStackedInterfaces` | **Implemented, unreachable on 2.1**: nothing can change a setting that is not defined. Same split -- `edgemode.Reconcile` proves what each flip obliges, over all eighteen states; what a sweep DOES to a standing world is a 2.0 leg |
| the summary reaching a PLAYER rather than a force | `force.print` is what a headless run can see, and the suite asserts the LocalisedString crossed and the force resolved. Whether the `[gps=]` pings are clickable and land where they should is a graphical client's question and joins the interactive checklist |
| a fixture with SINGLE-EDGE clusters adopting beside the refused ones | **Not exercised, because neither fixture has one**: every m2 rig is multi-edge (a 1->1 over one part already carries two belts) and so is every edge cluster. The suite asserts `adopted + rebuilt == clusters` and reports the split, so a fixture that grew one would be visible rather than silently ignored |

## Implementation status — phase 3, 2026-08-24

**Shipped: every gesture rig and every demo scene rebuilt single-edge, a
headless gate over the staged world, and a migration gesture on the checklist.**
Everything below is measured on Factorio 2.1.14 against the shipped
configuration (`--persist=packed --gc=collected`).

| what | where |
|---|---|
| the five gesture bands and the five demo scenes, all single-edge | `test/interactive/bbb-interactive-setup/control.lua` |
| `factorio_version` 2.1, `base >= 2.1.0`, version 0.2.0 | `test/interactive/bbb-interactive-setup/info.json` |
| the checklist, rewritten for the new geometry and two new gestures | `test/interactive/README.md` |
| the headless staging gate | `test/assert-interactive.py`, `stage_interactive` and the `iact` leg in `test/run.sh` |

### The bands, and the two that are redesigns rather than re-lays

Everything sits in one column at x = 20 as before, with the demo scenes in a
second column at x = 56. `COL` and `COL+1` are the west and east part columns;
a west part takes its input belt on its west face and an east part gives its
output on its east face, which is the `sedge` suite's idiom.

| band | was | is | y |
|---|---|---|---|
| A, the miner's pocket | 4 parts, 4 -> 4, dead-ended | **8 parts** in a 2x4 block, same 4 -> 4, still dead-ended | -24 |
| B, the belt at the edge | 2 parts with a free south face | **REDESIGN**: a 2 -> 2 over four parts plus a fifth EDGELESS part hanging below it | -12 |
| C, the sixty-fifth belt | 32 parts carrying 64 belts | **66 parts**: 64 input parts in a 2x32 block, one output part below, one edgeless part above | -1 to 33 |
| D, the bridge | two 16-part columns, gap flanked by two belts | **two 33-part balancers** (a 2x16 block plus an output part each), gap flanked by ONE belt | 43 to 80 |
| E, fast replace | 2 parts plus a line running past, and a 4-part column | **REDESIGN**: a 2 -> 2 over four parts with a line that ENDS on the target tile, and a FIVE-part column | 90 to 100 |

**Band B is a redesign because the old gesture no longer exists.** "Lay a belt
on a free face of a working balancer" is what the rule forbids: every part of a
working balancer already has its belt. So the rig carries an ATTACHED EDGELESS
PART, which is the only place a player's belt can change a balancer's port
count, and the belt goes there. Measured on the saturated rig: the belt takes
P from 2 to 4 with **72 items handed back and none spilled**, and mining it
again takes P back to 2 with **200 drained, 72 taken back and 128 that would
not fit** — the same 128 the `edge` suite's `bmin` leg records for the same
boundary crossing, which is what says the gesture still reaches the thing it
is about.

**Band B also carries the SINGLE-EDGE refusal**, and it is the only rig that
can: the new bound is reachable there without also crossing the port limit,
which a second belt anywhere on band C would do. A south-facing belt against
an occupied part's free north face is refused, the balancer keeps running, and
the audit reads `drift=1 unbuilt=0 refused=1`.

**Band C needed a sixty-sixth part, and that is forced rather than free.** Under
the rule every one of the 65 parts the design's paragraph names already carries
its belt, so a sixty-fifth belt has nowhere legal to land and would only ever
reach the single-edge bound. The spare edgeless part above the block is what
keeps this the PORT-limit gesture. Measured: `would need 128 ports for 65 inputs
and 1 outputs, over the limit of 64`, `drift=1 unbuilt=0 refused=1`, nothing
torn down.

**Band D's gap tile carries ONE flanking belt, not two**, for the same reason
in reverse: two belts on the bridging tile would make the merge a single-edge
refusal instead of a port-limit one. One belt gives the merged cluster 65
inputs over two outputs. Measured: `would merge into a cluster this mod cannot
build; left 2 standing network(s) alone`, then `128 ports for 65 inputs and 2
outputs`, audit `clusters=11 parts=229 nets=12 drift=1 unbuilt=0 refused=1`,
and mining the part back out rebuilds one half at `32->1 over 32 ports` with
396 items taken back and nothing spilled.

**Band E's forward half is a redesign because a part dropped MID-LINE is now
refused.** It would take the belt behind it as an input and the belt ahead as
an output, which is two belts on one part. So the line ENDS on the target tile
and the part that lands there gets one input. Measured: `can_fast_replace` true
over the line's last tile, and `compiled cluster 3->2 over 4 ports`.

**Band E's reverse half needs FIVE parts, not four.** The belt that splits the
column becomes an edge of the part above it and of the part below it, so both
of those must be otherwise edgeless — which puts the input part, an edgeless
part, the target, another edgeless part and the output part in a row. Measured:
`can_fast_replace` false over both END parts (each holds a `bbb-linked-belt`),
true over the middle, `a belt-connectable fast-replaced the part at 20,98`, and
two `1->1 over 1 ports` clusters where there was one.

**Band A's gesture got longer and did not change.** Eight parts means eight
mining steps, and every one of the shrinks overflows: mined one part per 30
ticks on the saturated rig, the three shrinks spilled **18 items each** and the
dissolve **178**, 232 in total. With a player all of it goes to the pocket, and
the checklist's "at every step, not only at the last part" is testable on this
rig rather than merely stated.

### The demo band, and the scene that retired

Five scenes in a column at x = 56, all saturated from world creation:

| scene | shape | parts | y |
|---|---|---|---|
| cross | 1 -> 3, P = 4 | 5: a plus whose four arms carry one belt each and whose centre carries none | -24 |
| compact column | 8 -> 8, P = 8 | 16: a 2x8 block, the smallest 8 -> 8 the rule allows | -10 |
| c-shape | 8 -> 8, P = 8 | 18: a ten-part spine with two four-part arms | 6 |
| c-shape express | 8 -> 9, P = 16 | 19: the same with a fifth part on the top arm | 30 |
| long run | 8 -> 8, P = 8 | 16: one row taking inputs from the north and giving outputs to the south, alternately | 60 |

**`single-part-1-to-3-fanout` is retired**, exactly as the design says: one part
cannot carry four belts. The cross is the same 1 -> 3 read and it is already a
scene, so the count goes from six scenes to five rather than needing a new one.

**The GIFs under `docs/media/` are the pre-port captures and are left in place**
until the re-capture lands, because deleting them would break the portal
description's links before there is anything to replace them with. Re-capturing
needs a graphical client and is the one part of this phase that is not done:
five scenes, plain and with alt-mode on for the two that have arrow variants,
and `single-part-1-to-3-fanout.gif` goes when its replacement is uploaded.

### The `iact` suite

One `--create` and no benchmark, because the whole question is answered by what
the guest logged at load. It fails on a placement the engine refused, on a shape
that is not the one the geometry intended, on any refusal at all, and on either
audit reading something other than what the geometry implies.

**Two audit markers, and they say different things.** An audit reports the
registry as its own dispatch finds it, and that dispatch is also what drains the
queue — so the first marker sees every cluster unbuilt and the second, placed
behind it, sees them built. A `--create` never reaches a tick, so there is no
third way to look. Measured: `(12, 228, 0, 0, 12, 0)` then
`(12, 228, 12, 0, 0, 0)`.

**The shapes are asserted as an exact multiset**, before anything else, because
a rig whose belts landed somewhere other than intended still compiles to
something plausible: `1->1/P1, 1->3/P4, 2->2/P2, 2->2/P2, 4->4/P4, 8->8/P8,
8->8/P8, 8->8/P8, 8->9/P16, 32->1/P32, 32->1/P32, 64->1/P64`.

**One thing the placement guard had to learn.** `bbb-audit` destroys itself
inside the dispatch its own placement raises, so `create_entity` returns nil for
it every time — which the staging mod's own "did it land" wrapper reported as a
rig that failed. The marker is placed outside that wrapper now, with the reason
written beside it.

**Red-proven**: a second belt staged against an occupied part of band B, which
is the defect class this gate exists for. Three assertions fire — one refusal
over the one-belt-per-part bound, a compiled multiset missing that rig's
`2->2/P2`, and the second audit reading `nets=11 refused=1` — and the run that
follows the revert is green again.

### What is verified and what a human still has to do

Every band's GEOMETRY is now measured rather than asserted, which is the
fast-replace band's own precedent applied to all five: a throwaway probe mod
drove each gesture by script against the staged world and every outcome above
is one of its readings. What no script can reach is unchanged and is the reason
the checklist exists — the flying text, its colour and its position, the sound,
the piece arriving in an inventory, the cursor's fast-replace preview, and the
`[gps=]` pings being clickable and landing where they say.

**The migration gesture is new on the checklist** and it is the eyes-on half of
the `mig21` suite: open a real 2.0 multi-edge save on 2.1 and check the chat
summary, the pings, the spilled items beside each stopped balancer and the
audit's `refused=` count. The suite pins every number; what it cannot do is
tell anyone whether the message reads well or whether a ping goes where it
claims.

**The BB2/BB3 gesture gained a paragraph rather than changing.** An incumbent's
balancer is multi-edge by construction, so on 2.1 every adopted one is converted
and then refused. That is the composition of two shipped features and needs no
code, but a checklist that promised a working machine would be wrong, so it
promises a rebuild checklist.

## Implementation status — phase 4, the test estate's first tranche, 2026-08-24

**Shipped: `m2`, `mar` and `upg`, rebuilt single-edge, and the heap-slope gate
back.** No guest line changed and no package was rebuilt: the whole pass is three
test mods' `control.lua`, three `info.json` version tokens, three assertion
scripts and `test/run.sh`'s default suite list. Measured on Factorio 2.1.14,
shipped configuration, both `-gc` arms.

`test/run.sh` now defaults to **`m1 sedge mig21 m2 mar upg iact`** — seven of the
thirteen.

### The re-lay, in one rule

**Every column of parts became two: a west column carrying the row's inputs and
an east column carrying its outputs.** That is the whole transformation for
nineteen of the twenty-one `m2` rigs and for three of `mar`'s four permanent
ones, and it preserves the property each rig exists to prove because **N, M and
`P = next_pow2(max(N, M))` are properties of the BELTS**, and the belts did not
move. `m2` went from 77 parts to 156 over the same 21 clusters; not one
assertion had to be weakened and not one rate, spread or port count moved.

Three things needed more than a re-lay:

- **`m2`'s `fdbk`, the feedback loop.** Its return run used to come in through
  the cluster's NORTH face, which under the rule is a tile that already carries
  an input. The loop now runs UNDER the block and returns through the loop row's
  west part's SOUTH face — the only free face left, because every other tile of
  the cluster carries its one belt. Its westward leg passes directly beneath the
  east column, which is safe for the same reason `pass` is: a west-facing belt on
  a part's south face is neither `dir` nor `back` from that side.
- **`m2`'s item-conservation rig.** Its edit was "lay a belt on the cluster's one
  free face", which the rule now refuses — so the check would have measured a
  refusal instead of a recompile. The block is three rows tall now and the bottom
  row carries nothing: the belt goes against an EDGELESS part, which is a third
  input and takes P from 2 to 4. Same figures out the other side, to the item:
  2,680 before, 2,680 after, 72 drained and 72 put back inside.
- **`mar`'s leg E**, the six-entity paste. Two parts and four belts made a 2→2
  under the old idiom and make a 1→1 under the rule, so the leg measures a
  smaller network and its slope fell (736 → 560 B). The EVENT shape it exists to
  measure — six entities in one tick and six out in one tick — is unchanged.

**`mar`'s `BIG` rig did not move at all**, and that is the pass's own control: a
4×4 built the wide way (inputs on the west column, outputs on the east, two
interior columns carrying nothing) was ALREADY single-edge and never had a tile
with two belts on it. Its slope came back at **3,736 B, identical to the byte**.

### The slopes, which is the gate this tranche restores

`make GC=leaking test`, one invocation, 680 net-zero operations. Old column is
2.0.77 multi-edge, new is 2.1.14 single-edge:

| one operation | before | after |
|---|--:|--:|
| a belt-connectable mined or rotated far from anything | 0 B | **0 B** |
| a belt-connectable built far from anything | 32 B | **32 B** |
| a belt laid inside the two-tile gate and picked up again | 352 B | **352 B** |
| one teardown-and-rebuild of a 2→2 | 1,180 B | **1,209 B** |
| one teardown-and-rebuild of a 4×4 | 3,736 B | **3,736 B** |
| a whole 4-part balancer in and out under load | 1,216 B | **1,280 B** |
| a six-entity paste and its undo | 736 B | **560 B** |
| a balancer grown by a part, dissolved and rebuilt | 1,712 B | **2,080 B** |
| one `bbb-audit` | 1,136 B | **1,136 B** |
| linear memory over the run | 3.92 MiB | **3.92 MiB** |

Linearity ×1.00–×1.07 on every leg against a ×1.35 bound; the calibration is
1,136 B three times at **0.0% spread**; conservation over 100 add-part /
remove-everything cycles of a full network is **9,600 in, 9,600 out, 0 lost over
200 teardowns**; 681 audits at drift=0. The collected arm ends on **0.46 MiB with
a 10,192 B live set, 9 collections in 6 paced steps and 0 forward-progress
deadlines**.

**The three terms the 300-hour projection multiplies are all in the unmoved
set**, so that projection stands exactly as CLAUDE.md records it. Checked rather
than assumed.

### The red proofs, and two of them found something

| injected defect | what came out |
|---|---|
| **`m2`'s `pass` line turned SOUTH** — one token, and the passing belt becomes an edge on two parts that already carry one each | **six assertions**, where the pre-port geometry would have cost a rate and nothing else: `pass` delivering **0 0, 0.000x** and its own passing chest **0**, `nets != clusters`, the audit's new `refused=1`, and the named refusal `cluster 147 has 2 parts carrying more than one belt`. The whole balancer stops, which is what the rule does to a cluster it cannot build |
| **`mar`'s leg F rebuilding only half the churn rig** | **NOTHING, on the suite as it stood.** The item count never rose, nothing drifted, no cluster read `unbuilt` — a cluster with inputs and no outputs is a legitimate half-built state, not an unbuilt one — and the calibration spread stayed at 0.0%. The only trace was `calZ` re-classifying a world with two fewer parts and one fewer network than `cal` did, and nothing was looking. `assert-marathon.py` pins the `(clusters, parts, networks)` tuple of every leg's probe now, against constants written in the script; it fails by name: *"leg calZ audited a world of (3, 22, 2) and the rigs build (3, 24, 3)"* |
| **`bump_build` moving the mod VERSION and not the build stamp** | the saved guest heap is ADOPTED, `fk_migrate` never fires, `rebuildFromWorld` never runs, and `assert-upgrade.py` fails with *"the guest never rebuilt its registry from the world after the upgrade"*. That is **S2 result 6 red-proven in this repo** rather than quoted: a version bump alone does not decline the heap |

### What this tranche did NOT do, and what the next one has to decide

Six suites are still multi-edge: `m3`, `plat`, `edge`, `mix`, `mig` and `qual`.
Four of them are a re-lay of the same shape as this one. Two are not, and the
reason is worth writing down before anyone starts. (**`mix`, `plat` and `qual`
landed the same day, as phase 5 below** — all three re-lays, and the paragraph on
`edge` and `mig` stands exactly as it was written:)

- **`edge`** has rigs whose GESTURE the rule changes rather than whose geometry
  it doubles. Its `lim` column is 64 belts over 32 parts and becomes 64 belts
  over 65 parts; several legs lay a belt on a working balancer's free face, which
  is now a refusal rather than a recompile, so each has to be re-aimed at an
  edgeless part exactly as `m2`'s conservation rig was. `bmin`'s port-boundary
  crossing and `brdg`'s over-limit merge both need re-deriving from scratch.
  **DONE the same day, in phase 5 below**, and every sentence of this paragraph
  turned out to be right except the arithmetic: `lim` is 64 belts over SIXTY-SIX
  parts, because the spare part the sixty-fifth belt lands on is a sixty-sixth.
- **`mig`** changes OUTCOME and not only geometry: every adopted incumbent
  balancer is multi-edge by construction, so on 2.1 each converts and is then
  refused. Its expectations move wholesale — adopted yes, networks no — which is
  the thing "The BB2/BB3 migration feature on 2.1" above already states and which
  no re-lay can paper over.

The interactive worlds are NOT on that list: phase 3 rebuilt them the same day,
and its band B reached the same conclusion `m2`'s conservation rig did from the
other side — under this rule a working balancer has no free face, so the only
place a player's belt can change a balancer's port count is an attached EDGELESS
part. Two independent passes needing the same trick is the strongest signal in
this file that it is the shape of the rule and not a workaround.

## Implementation status — phase 5, the test estate's second tranche, 2026-08-24

**Shipped: `mix`, `plat` and `qual`, rebuilt single-edge.** No guest line
changed and no package was rebuilt: the whole pass is three test mods'
`control.lua`, three `info.json` version tokens, three assertion scripts and
`test/run.sh`'s default suite list. Measured on Factorio 2.1.14, shipped
configuration, both `-gc` arms.

`test/run.sh` now defaults to **`m1 sedge mig21 m2 mar upg mix plat qual iact`**
— ten of the thirteen.

### The re-lay, and the one thing every conservation rig needed

Phase 4's rule again — every column of parts becomes two, a west column carrying
the row's inputs and an east column its outputs — plus the thing phase 4 met once
and this tranche met in every band of two suites:

**A WORKING BALANCER HAS NO FREE FACE.** Every conservation check in this
repository forces a recompile by laying a belt against the cluster, and under the
rule every part of a working balancer already carries its one belt — so that
belt is now REFUSED and the check measures a refusal instead of a teardown. Every
such rig carries one extra EDGELESS part below its west column and the belt goes
there. That is `m2`'s conservation rig, the interactive checklist's band B, and
now every band of `mix` and `plat`: **four independent passes needing the same
trick**, which is as strong a signal as this file has that it is the shape of the
rule rather than a workaround.

`qual` is the pleasant surprise: **three of its four rigs were already
single-edge**, and it is the only suite that could have been. `qblk` is a 2x2
whose west column carries the inputs and whose east column the outputs, `qcol`'s
two INTERIOR parts carry nothing (which is what made the fast replace legal in
the first place), and `qlone` has no belts at all.

### `mix`, and the assertion that had to be retired rather than re-recorded

Four clusters over twenty-eight parts, 3,220 ticks, base only. One saturated
express belt delivered **1,306 items** over t=1400..3140:

| rig | what it is | what came out |
|---|---|---|
| `duo` | 2 -> 2, two PURE belts (iron, copper), draining freely | **1306 1306**, 2.000x, spread 0.00%; conservation exact per name over 48 names and 20,076 items, **0 on the ground** |
| `quad` | 4 -> 4, two iron and two copper belts ALTERNATING | **1306 1304 1306 1304**, 3.997x, spread 0.15% |
| `mixfull` | 2 -> 2, two SUSHI belts, dead-ended | exact per name, 6,616 items, **0 on the ground** |
| `many` | 4x4, four sushi belts over 48 names, dead-ended | **7,936 in, 7,936 out, every name exact**, 18 on the ground, and the overflow alert: **64 items past the 32-group bound** |
| `probe` | one chest, six filters, no balancer | 2,292 items in **1 of 6 kinds**, electronic-circuit 100% — unchanged, and still the measurement that justifies the rotating source |

Final audit **`clusters=4 parts=28 nets=4 drift=0 unbuilt=0 refused=0`**, and
that tuple is new: this suite read no audit at all before. `unbuilt=0` alone
would not do it — a cluster with no inputs or no outputs is a legitimate
half-built state, so a rig rebuilt one column wide would be refused, deliver
nothing and still read `unbuilt=0`.

**THE ONE ASSERTION IN THIS WHOLE PASS THAT DID NOT SURVIVE RE-RECORDING**, and
it is worth the space because the finding is about the mod rather than about the
rigs. `duo` carried a per-output TYPE FLOOR at 15%: each output had to see at
least 15% of each kind, on the reasoning that "an output seeing only iron would
mean the two kinds took different paths through the network, which is a real
defect". Re-laid, `duo` delivers **100/0** — out1 all the copper, out2 all the
iron — exactly balanced by count at 0.00% spread.

That was worth chasing rather than re-tuning, and three measurements settled it:

- **the pristine 2 -> 2 separates too**, so it is not the conservation belt;
- **a 4 -> 4 separates as well**: `quad`, fed iron/copper/iron/copper, sends all
  the copper to outputs 1 and 2 and all the iron to 3 and 4, 1306 apiece. So it
  is not a two-line accident. **Under symmetric saturation this butterfly is a
  PERMUTATION**: every output takes exactly its share by count and exactly one
  kind;
- **the old 75/25 was a PORT ORDER.** The window used to open AFTER `duo`'s
  conservation belt had taken it from 2 -> 2 to 3 -> 2 over P=4 — an asymmetric
  network with a dead-ended spare port and a loopback, where the flows genuinely
  have to cross — and the multi-edge geometry put that belt FIRST in the edge
  list. Laid single-edge the same belt enters LAST and the same P=4 network
  delivers 100/0. Both are exactly balanced by count; nothing regressed.

So the floor was never a statement about the balancer, it was a statement about
one asymmetric network's port assignment. It is **retired**, `duo`'s edit moved
to the END of the schedule so that the window measures the 2 -> 2 its own
description names, and what replaces it is the check the floor was groping at:
**every kind must come out at the rate it went in**, summed over the outputs —
`duo` 1.000 belt of each of two kinds, `quad` 1.998 of each, at 2%. True of a
permutation, false of a network that starves a kind, silent about a mix nothing
ever promised. `quad` is new and exists so that `duo` alone can never again be
mistaken for a property.

### `plat`: the geometry doubled and not one number moved

Five clusters over thirty-two parts across two surfaces. Every figure came back
**identical to the item** to its 2026-08-05 record:

| | measured |
|---|---|
| the platform 2 -> 2 (four parts now) | **676 676 against 676, 2.000x, 0.00% spread** |
| stacks formed before any recompile | 1,128 items over **336 positions — 72 of size 1, 264 of size 4** |
| `full` recompiled | 10,952 either side, **+0 single, +128 stacked** |
| `plain`, unstacked under an open gate | **+16 single, +0 stacked** |
| `flow`, recompiled under load | +0/+48, then **1496 1496 1492 1488 against 1504 — 3.971x, spread 0.54%** |
| `smix` | conservation **EXACT per (name, quality) over nine kinds**, 704 items; **+0 single, +64 stacked**, 56x4 -> 72x4; anti-vacuity **14 of 24 hidden lines carrying two names (all 14 stacked) and 6 carrying one name at two qualities (all 6 stacked)** |
| spills, overflows | **0** and **0** |
| final audit | `clusters=5 parts=32 nets=5 drift=0 unbuilt=0 refused=0` |

That the stacking numbers are unmoved is the expected result rather than a lucky
one, and it is the cleanest evidence in the whole port that the re-lay is a
geometry change and nothing else: **a hidden network is a function of the BELTS**,
and the belts did not move.

### `qual`: one rig moved, and it needed a sixty-sixth part

`qlim` was thirty-two parts with a belt on both sides of each — exactly what the
rule forbids — and is **sixty-six** now: one output part above, a 2x32 input
block (64 inputs, P = `plan.MaxPorts` exactly), and one EDGELESS part below for
the sixty-fifth belt to land on. That spare part is forced rather than free, and
for the reason phase 3's band C already records: under the rule all sixty-four
input parts already carry their belt, so a sixty-fifth belt against any of them
would ask the SINGLE-EDGE bound and this would stop being a test of
`forceOfCluster` at all.

Measured: **exactly one** refusal (`128 ports for 65 inputs and 1 outputs, over
the limit of 64`), **exactly one clean `told force 1`**, qlim delivering 900
items across the edit, `qblk` at **900 900 against the control's 900 — 2.000x,
0.00% spread**, zero hand-backs, and the audit walk at `(4, 75, 0, 0, 3, 0)`,
`(4, 75, 3, 0, 0, 0)`, `(5, 74, 2, 0, 0, 0)` and `(5, 74, 2, 1, 0, 1)` twice.

**Two things the skin assertion had to learn.** qlim's variation string is
derived from `guest/go/skin`'s own pure-Go `Variation` over the tile set the rig
builds — the same package `make check` proves — rather than read back off a run
of the thing under test, and the derivation was validated by reproducing
`qblk`'s, `qcol`'s and `qlone`'s recorded literals exactly before it was trusted
for the new shape. And **`logSkin` caps the list at 32 variations and writes a
literal `...`**, which no cluster in any other suite is big enough to reach: the
assertion matches the truncated line rather than parsing the marker away, so a
cap that moved would fail rather than pass silently.

### Red-proven three times, one per suite, each in the new geometry

| injected defect | what came out |
|---|---|
| **`mix`: the `tally` fix reverted** — `addOverflow` and the spill accounting replaced by the pre-2026-08-04 `return` | **16 item kinds and 64 items destroyed**, 7,936 in and 7,872 out, **nothing at all on the ground** and no alert anywhere in the run. **Three assertions fire** and they are three different statements: the kinds did not survive, nothing reached the ground on a rig that exists to overflow, and the guest never said so |
| **`plat`: the stacking gate forced shut** (`stacksPossible` returns false) | **nine assertions**, the same nine and the same **392 items on the ground** as the 2026-08-05 record: **+572 single positions on `full`** and **+304 on `flow`** where the fixed guest adds none, `smix`'s 56x4 histogram becoming **196x1**, and the stacked throughput falling to 3.914x at 4.62% spread. **Per-kind conservation stays EXACT** — the gate-off path is still conservation-correct, it just unstacks |
| **`qual`: `findOnTile` given `Quality = "normal"`** — the pre-fix `find_entity` semantics stated exactly | all three families at once: **24 skin lines instead of 6, every one `set=0`**, the lone part unregistered under a scripted COLLIDING belt with the audit walk failing at all four post-collide tags (`clusters=3 parts=74` where the world holds 4 and 75), and **the force told 0 times**. The true positives stay true in that arm, which is what makes them controls rather than assertions of the fix |

### The gates

`make check` green (bindings and lock unmoved — no guest line changed).
**All ten suites green in BOTH `-gc` arms**, each arm one invocation. The `mar`
slopes came back **identical to the byte** to phase 4's record — 0 B for a far
belt mined or rotated, 32 for one built, 352 inside the neighbour gate, 1,209
for a 2 -> 2 recompile, 3,736 for a 4x4, 1,280 for a whole balancer in and out,
560 for the six-entity paste, 2,080 for grown-and-dissolved, 1,136 for the
audit, and **3.92 MiB** of linear memory — with linearity x1.00–x1.07, 9,600
items in and 9,600 out over 200 teardowns, and 681 audits at drift=0. **No other
suite's numbers moved**, which for a test-only pass is the only result available.

## Implementation status — phase 6, the estate's third tranche, 2026-08-24

**Shipped: `m3` and `edge`, rebuilt single-edge, and one guest fix the rebuild
asked for.** `test/run.sh` now defaults to **`m1 sedge mig21 m2 m3 mar upg edge
mix plat qual iact`** — twelve of the thirteen, everything but `mig`. Measured
on Factorio 2.1.14, shipped configuration, both `-gc` arms.

### `m3`: a re-lay, plus two edits that had to be re-aimed

Every column of parts became two, exactly as phase 4's rule says, and **not one
rate, spread or "exactly zero" in the suite moved**: the control belt still
delivers 720 over t=540..1500, `live` is 4.000x, `swap` is 1.667x, `died`'s
orphaned row still receives exactly none, and the stress phase still recovers
15,856 of 16,000. What moved is part counts (a 2-in/2-out rig is four parts) and
the blueprint's 12 entities becoming 14.

Two edits needed more than a re-lay, and both are the shape phase 4 named:

- **`phase_silent_notice`'s "unrelated placement"** was a south-facing belt on
  the top west part's north face, which under the rule is that part's second
  belt. It goes DIAGONALLY from the cluster now — inside the two-tile neighbour
  gate, so the cluster is re-classified, and orthogonally adjacent to nothing, so
  no tile gains an edge. The fingerprint still moves, because the belt phase 11
  destroyed silently is missing from the classification the placement provokes,
  which is the thing under test.
- **`died` kills the EAST part of the second row.** A row's output stands against
  its east part, so that is the kill that takes the row's OUTPUT off the machine
  and leaves its chest orphaned. Killing the west part takes an INPUT off instead
  and leaves both outputs live at half a belt each, which is a different
  measurement.

**And the stress churn AVOIDS the refusal rather than embracing it**, which is a
decision the brief asked to be taken deliberately. Its six randomised edits are
aimed at a row's own single input, its own single output and an EDGELESS part
below the west column, so no tile can ever be asked for a second belt — and
`assert-m3.py` asserts that negative over the whole run. The reason is that this
suite's subject is the twelve lifecycle paths and its sharpest assertion is
`drift=0 unbuilt=0` after 600 ticks of churn: a churn that generated refusals
would make the compile, build and teardown counters a function of the RULE rather
than of the path under test, and would leave clusters standing refused at the
final audit. The refusal has its own suite, which drives all three of its trigger
shapes.

**The final audit asserts a WORLD TUPLE now** — `(clusters, parts, networks)` =
`(14, 59, 14)` against constants in the script — because `unbuilt=0` is weak
evidence: a cluster with inputs and no outputs is a legitimate half-built state
and is never counted, so a rig that quietly lost its network satisfies it. That
is the `mar` suite's own idiom, applied here for the same reason it was added
there. `refused=0` is asserted beside it.

**Red-proven**: put the notice belt back where it used to be. Four assertions
fire — `noev` at **0 0, 0.000x**, one cluster refused for the one-belt rule
(twice, once per audit that reaches it), `refused=1` on the final audit, and the
world tuple at **(14, 59, 13)** — and **`unbuilt` stayed 0 through all of it**,
which is exactly why the `nets` column had to exist.

### `edge`: four rigs redesigned, not re-laid

Fifteen clusters over **one hundred and ninety-eight parts**. The re-lay is the
same rule, and what phase 4 chartered as needing real work needed exactly that:

| rig | what the rule did to it |
|---|---|
| `aout`, `ain` | each carries a **fifth ROW of parts holding nothing**. Both exist to take a belt on a working balancer, and a working balancer has no free face; the belt goes on the spare row. `aout`'s fifth output leaves south off it and `ain`'s fifth input arrives north into it |
| `bmin` | a 2->2 over four parts plus an **attached edgeless fifth**, which is where the third output belt lands. P still goes 2 -> 4 and back, which is the boundary crossing the rig exists for |
| `lim` | a **2x32 block** carrying one input belt per part (sixty-four inputs), an output part below it and a **spare part above it**: sixty-six parts. Every part that carries a belt has its one belt, so a sixty-fifth belt laid anywhere but the spare would reach the one-belt bound instead and the leg would measure the wrong refusal |
| `brdg` | two **thirty-three-part** halves, and the gap tile is flanked by **ONE** belt. Sixty-five inputs over two outputs |
| `ntch` | a **C of five parts** around a hole at (1, b+1), two in and two out. The old shape put an input and an output on one tile |
| `frepa` | the belt line **ENDS** on the tile the part is dropped onto, so the part takes one input and gives no output: 3 -> 2 over four ports. A part dropped mid-line takes the belt behind it as an input and the belt ahead as an output |
| `frepb` | a 1->1 row, a **three-part NECK carrying nothing**, and another 1->1 row: nine parts. The belt that replaces the middle of the neck is an edge of the part above it AND of the part below it, so both must be otherwise edgeless or one half would be refused; and a second column beside the target would keep the cluster connected around the belt, so there would be no split at all |
| the by-hand teardown | **ten parts and ten mines**, spare row first and then row by row, west part then east part. Every prefix leaves a CONNECTED cluster and eight of the nine shrinks leave a machine with at least one input and one output, so P really comes down 4 -> 2 -> 1 rather than the machine simply dying |

**The two over-limit refusals are told apart by their OUTPUT count now.** Both
are sixty-five inputs — `lim` gains its belt on a spare part and `brdg`'s gap
carries one flanking belt — so the discriminator moved from `inputs == 65 or 66`
to `outputs == 1 or 2`.

**And `compile()` asks the port bound FIRST, measured.** The obvious red proof
for `brdg` was "give the gap tile a second flanking belt, and the merge becomes a
single-edge refusal". It does not: a second belt has to stand on the bridging
part's own east face to be an edge of it at all, and the merged shape is then
illegal twice — sixty-six inputs AND two belts on one tile — and `compile()`
returns on the port check before it reaches the other one. So the refusal still
reads as a port refusal, and the only thing that can see the difference is the
INPUT COUNT in the assertion. Which is what fired: *"the refused merge names 128
ports for 66 inputs, expected 128 for 65"*. That is the red proof, and it is
worth more than the one that was expected because it is a fact about the guest
rather than about the rig.

**Everything else in the suite is the number it was**, which is the result this
tranche wanted: `lim` delivers 184 items over 246 ticks before its refusal and
185 after; `brdg`'s halves 186 and 185 before, 184 and 184 across the refusal and
after the un-merge; the dissolve spills 118; `bmin`'s port-boundary removal
spills 128; the aout recompile's 4->5 network delivers **300 300 300 300 300 at
0.00% spread**; `ntch` delivers **376 376** while the placement probe is sampled;
the probe reads **180-197 entities of ours, 0 off a part tile, every one of seven
samples**; the insert probe is 50/37/23/7 asked, taken and held; and `frepb`
goes **[262, 262] -> [132, 264]**, 76% of the column, which is the cascade the
2.0 rig measured to the item.

**`nets` is asserted at every tagged audit**, for `m3`'s reason. The one place it
is legitimately short of `clusters` is the ninth by-hand step: one part is left
standing, one part carries one belt, so the survivor has an input or an output
and never both — which `plan.Build` reads as a legitimate half-built cluster.
Under the old rule the last part carried both and the leg never met the case.

**And a NEGATIVE the suite did not have**: zero one-belt-per-part refusals over
the whole run. Every rig here is laid so that no tile ever carries two belts and
every edit is aimed at a spare part, at an edge the rig already has, or at a tile
with nothing on it; a refusal for the other bound would mean a rig or an edit had
quietly stopped being the thing it is named for, and every count above it would
go on passing.

### The rider: the hand-back lines name the bound that refused the piece

`revertOne` said "the over-limit piece" in both of its lines, and `tellRefusal`
and `revertOne` are shared by both bounds — so a belt handed back for the
one-belt-per-part rule was reported as an over-limit piece. That is the same
defect `spareMerge`'s line had one level up, and phase 1 fixed that one by making
it name NO bound, because a merge pre-pass genuinely cannot know which. Here the
guest does know: the refusal that queues the note is the one moment anything
does, and it knows for free, because `tellRefusal` already takes the clause for
its own force-wide line.

So `limPending` carries a `pendingPiece{note, why}` rather than a bare note, and
the two lines read

    handed the refused piece at 20,-2 (over the port limit) back to player 1
    player 1 could not be handed back the refused piece at 20,-2 (past the
      one-belt-per-part rule) -- no room in the inventory; it stays where it is,
      unconnected

The bound is CARRIED rather than looked up because by revert time there is
nothing left to look it up from: `revertOverLimit` runs after the drain, the
cluster may have been re-rooted or dissolved by then, and `overLimit` remembers a
fingerprint and not a reason.

**It costs nothing and the `mar` suite now says so rather than the comment
saying so.** `limPending` is empty on every tick nobody's build was refused, and
a headless run never appends to it at all; the seven slopes came back
**identical to the byte** — 1,280 / 352 / 1,209 / 32 / 560 / 3,736 / 2,080 B per
primitive, 1,136 B of calibration at 0.0% spread, **3.92 MiB** of linear memory
— against phase 4's record.

**And the package got SMALLER**, which was not the point but is worth recording:
`fk_module.lua` 3,043,634 → **3,036,333 B**, measured either side of the change
in one session with the same flags and the same pin. The struct field and the
extra `logS` are less generated code than the two hand-written sentences they
replaced parts of.

**Four assertion scripts and two documents moved with it**, and the four regexes
became LOOSER on purpose. All four are NEGATIVE assertions — a headless run has
no players, so `revertOne` returns before it mines anything — and an exact regex
over a negative is the one shape a rename in the guest can make VACUOUS: the line
stops matching, the assertion stops being able to fail, and nothing says so. They
match `piece at x,y` now, which is what both arms have in common and which
nothing else in this guest's vocabulary produces.

### Gates

`make check` green. `make test` and `make GC=leaking test` each ONE invocation,
both green over all nine suites. The `mar` slopes byte-identical to phase 4's
record in the leaking arm; the collected arm ends on 0.46 MiB. **No suite outside
`m3` and `edge` moved by a number**, which for a test-estate pass plus a log-line
change on a cold path is the only result available.

### What this leaves

**`mig` alone**, and it is the one the design has always said changes OUTCOME
rather than only geometry: every adopted incumbent balancer is multi-edge by
construction, so on 2.1 each converts and is then refused, and its expectations
move wholesale. **DONE the same day, as phase 7 below.**

## Implementation status — phase 7, `mig` and the end of the estate, 2026-08-24

**Shipped: the migration suite reworked for the convert-then-refuse outcome, and
`mig` back in the default.** `test/run.sh` now defaults to all **thirteen**. No
guest line changed and no package was rebuilt: the whole pass is one test mod's
`control.lua`, three `info.json` version tokens, one assertion script and
`test/run.sh`. Measured on Factorio 2.1.14, shipped configuration, both `-gc`
arms.

### The one suite whose rigs were deliberately NOT re-laid

Every other tranche re-laid its rigs, because they are OUR rigs and the rule is
ours to obey. **`mig`'s world is somebody else's.** Belt Balancer's own idiom is
a single column of parts with a belt on every free face, which is two belts per
part, and that is exactly what a migrating player's save contains — so re-laying
those rigs would have been re-laying the thing under test, and the suite would
have stopped measuring migration at all.

They stay as the incumbent builds them. What was ADDED is the **`sok` band**:
the same balancer laid two columns wide, which is a shape a Belt Balancer user
could genuinely have and which this engine can build. One world, both outcomes,
which is the honest portal story rather than a second leg staging a second
world.

**That also closes a gap phase 2 recorded and could not close.** `mig21`'s "what
is still not verified" says: *a fixture with SINGLE-EDGE clusters adopting
beside the refused ones — not exercised, because neither fixture has one.* This
world has two, and the `added` leg's rebuild-from-world **adopts them beside the
seven it refuses**, measured.

### The world, and what comes out of it

Nine clusters over thirty-one parts on three surfaces and two forces. The four
rigs that were there before are unchanged in every respect but their y offset;
`sok2` and `sok4` are new.

| rig | laid | what it is on 2.1 |
|---|---|---|
| `ctrl` | — | a bare express belt, the yardstick: **1306 items** over t=1800..3540 |
| `m4x4` | the incumbent's way, 4 parts | refused. **0 0 0 0** |
| `m3to5` | the incumbent's way, 5 parts | refused. **0 0 0 0 0** |
| **`sok2`** | **two columns, 4 parts** | **compiled. 1306 1306 — 2.000x one belt, spread 0.00%** |
| **`sok4`** | **two columns, 8 parts, P=4** | **compiled. 1304 1306 1306 1304 — 3.997x, spread 0.15%** |
| `wit` | the incumbent's way, 2 parts | refused; its 48 copper plates are where they always were |
| `fid` | the incumbent's way, 2 parts | refused; 85.0 of 170.0 health and `uncommon` both carried |
| `frc` | the incumbent's way, 2+2 on two forces | two clusters, both refused, both forces given the technology |
| surface B | the incumbent's way, 2 parts | refused |

**Every conversion number is what it was**, which is the result this pass wanted:
31 parts from 3 surfaces into 9 clusters, 2 forces researched, the item stack at
50 with its `place_result` flipped, `belt-balancer-1` gone, the per-surface
census at `bbb-mig-a:0/29 bbb-mig-b:0/2`, and the witness's copper at **48 at
every one of four samples**. `legacy.go` was not touched and did not need to be.

**And the audit is the whole outcome in one line**, identical in all five
conversion legs:

    clusters=9 parts=31 nets=2 drift=0 unbuilt=0 refused=7

`nets != clusters` is the port rather than a regression, and it is why the
suite's cluster check stopped being `nets == c`: seven clusters are refused and
a refused cluster never gets a network. `refused=` is what tells them apart from
a cluster the classifier never saw, and `unbuilt` stays 0 — that column is this
guest saying it should have built something and did not.

### The refusal, and the one thing this suite asserts that `mig21` asserts the opposite of

**Nothing is torn down and nothing is spilled.** `mig21`'s clusters were
STANDING when the save opened, so the remnant had to come down and everything it
held reached the ground; here the clusters are seconds old — `legacyScan`
creates the parts and the very next flush refuses them — so `hadNet` is false,
there is no teardown for the refusal to be in front of, no carry pool is opened
and nothing of the player's can reach the ground. The items are where they
always were, on their own belts, which is what the copper witness measures from
the other side. **0 teardowns and 0 spills over every leg**, asserted.

**Seven refusals, one per cluster, and the SHAPE is asserted as a multiset**:
`[2, 2, 2, 2, 2, 3, 4]` parts carrying more than one belt. `m3to5` is the row
that makes this a statement about a classification rather than about a constant
— three inputs and five outputs over five parts means three of them carry two
belts and two carry one, and nothing else in the world has a count that is
neither zero nor its whole size.

### The message a converted balancer gets, which is the defect this pass found

**A converted-and-refused balancer is announced with the ORDINARY per-piece
message, and in the commonest migration shape the player never sees the
migration summary at all.** Measured, not inferred:

| leg | how the player got here | ordinary per-cluster lines | migration summary |
|---|---|--:|---|
| `added` | this mod and the removal in ONE edit | **7** | **yes**, force 1 about 6 and force 4 about 1 |
| `later`, `bb3`, `fgone` | this mod already installed, incumbent removed later | **7** | **no** |
| `built`, `readd` | parts arriving through build events | **7** | **no** |

The mechanism is not subtle once the two producers of `sedgeAnnounce` are
written down: **both of them are `rebuildFromWorld`** (sedge.go — the rebuild's
own fold, and `refuseSingleEdge`'s rebuild arm). A LEGACY CONVERSION is neither.
So the summary arrives only when a rebuild-from-world happens to follow the
conversion in the same session, which is exactly the `added` leg: this mod is
new to the save, `fk_on_init` converts, and the `fk_on_configuration_changed`
that follows it — a newly added mod is itself a mod-set change — finds
`registryReady` false and rebuilds. In every other leg the rebuild already
happened in phase one over an empty registry.

**What the player gets instead is a sentence about an event that never
happened**: `single-edge-refused-unconnected` says the extra piece was left in
place, unconnected, and nobody placed anything. That is the same defect class
`mig21`'s third red proof exists for — *"a migration announced with a sentence
about an extra piece being left in place unconnected, when nobody placed
anything"* — reached through the other door. And in the `added` leg the player
gets **both**: seven per-piece lines and then the summary.

**Not fixed here, deliberately, and the reasoning is worth keeping.** It is a
guest-behaviour decision inside a test-estate pass, the fix has an untestable
arm, and the suite that would have to prove it now exists:

- the shape of the fix is `legacyScan`/`legacyRunBuilds` noting the converted
  roots the way the rebuild does — a flag around the conversion and its flush,
  read by `refuseSingleEdge` exactly as `rebuildingFromWorld` is;
- but `settleEdgeMode` then asks `edgemode.GrandfatherNeeded` of those roots,
  and **on 2.0 that would flip `bbb-multi-edge-parts` ON for a converted BB2
  save**. That may well be the right answer — a converted save whose balancers
  need multi-edge is precisely what grandfathering is for — but it is a
  behaviour change on an engine this machine cannot run, and taking it blind
  inside a test pass is how a `release/2.0` regression ships;
- so both arms are PINNED instead. `EXPECT_SUMMARY` in `test/assert-mig.py`
  asserts the summary's presence in `added` and its ABSENCE in the other five,
  with the reasoning beside it, so whatever the fix turns out to be has to move
  a number there and cannot land silently.

### Red proofs: five re-derived, and which of the seventeen retire

Two are about the new outcome and three are regressions across three different
families of the conversion. Every one is an injected defect, built, run and
reverted.

| injected defect | what came out |
|---|---|
| **`sok2` given a second belt** — one extra south-facing belt on its top west part's north face, so the rig stops being single-edge | **eight assertions across five families**: `sok2` at **0.000x** and 100% spread, the refused count 7 → **8**, the shape multiset gaining a **`1`** (`[1, 2, 2, 2, 2, 2, 3, 4]`), the summary naming 8 balancers, the per-force totals adding to 8 of 7, the rebuild adopting **1** where it should adopt 2, and the audit at `(9, 31, 1, 0, 0, 8)`. **`sok4` stays green**, which is what says the split discriminates per rig rather than per suite |
| **`m4x4` quietly re-laid TWO COLUMNS WIDE** — the anti-vacuity direction, and the one that would gut this suite silently | **eight assertions**: the world-size guard first (**35 parts where the rigs make 31**), then `m4x4` **delivering [1304, 1306, 1306, 1304] when it is supposed to be refused**, the refused count 7 → **6**, the shape multiset losing its `4`, the summary at 6, the rebuild adopting **3**, and the audit at `(9, 35, 3, 0, 0, 6)`. A suite whose incumbent rigs stopped being the incumbent's idiom would still pass every conversion check ever written; this is what stops it |
| **the marker guard removed** (`legacyStubPresent()` in `legacyCheck`) | the `foreign` leg fires **twenty-plus assertions**, and among them the NEW one: a stranger's converted balancers are then **refused** — `the audit reads (9, 31, 2, 0, 0, 7); this mod should own nothing at all in this save, which includes refusing nothing`. Converting somebody else's entities and then declining to build them is the loudest possible form of the thing this guard prevents |
| **the `SetHealth` copy skipped** in `legacyConvertOne` | **exactly two**, one per phase: *the damaged part is at 170.0 health at phase=t1 and was at 85.0 before the swap*. Nothing else in the leg moves — the conversion contracts have exactly the teeth they had before the port |
| **the force check removed** from `AddPart`'s adjacency loop | **seven**, including one the pre-port suite could not make: *the summary reached 1 force(s) and the force rig puts refused balancers on 2*. Plus the conversion's own cluster count 9 → **8**, the refused count 7 → 6, the shape multiset, and the audit tuple |

**None of the seventeen retire, and that is the finding rather than a
formality.** Every one of them is about the CONVERSION — the data-stage stub,
the marker guard, the stranger landing in Blocked, the four incumbent names, the
build path's phase gate, the quality-blind `find_entity`, the health, the
quality, the per-force grant, the two-forces-are-two-balancers fill, the two
surface statements, and the four harness `skip-is-a-pass` proofs — and the
single-edge port did not touch `legacy.go` at all. What retires is not a proof
but the ASSERTIONS the old suite made about delivery: `m4x4` at 3.997x and
`m3to5` at 2.995x are gone, because those balancers cannot run on this engine,
and they are replaced by two statements rather than one — the `sok` band at
rate, and the incumbent-idiom rigs at exactly zero.

Three of the five above were re-run against the reworked suite rather than
merely reasoned about; the other twelve were not re-run this pass, and the
reason is stated rather than assumed: their assertions are byte-identical to
what they were, and each one's evidence is recorded in CLAUDE.md's own
seventeen-row table against the same guest.

### Gates

`make check` green (bindings and lock unmoved — no guest line changed). `make
test` and `make GC=leaking test` each ONE invocation, both green over **all
thirteen suites**. The `mar` slopes came back **identical to the byte** to phase
4's record. **No suite outside `mig` moved by a number**, which for a
test-estate pass is the only result available. The suite costs **43 s** for
seven legs and two probes, against 30.6 s for the pre-port world of nineteen
parts.
